#!/usr/bin/env python3
"""The V0.12b Stage B driver: two whole decision workflows from one
recorded proposal (docs/benchmark/v12b-dev/PROTOCOL.md; the driver's
specification: docs/benchmark/v12b-dev/DRIVER.md).

At each selected branch point, every draw starts from the same three
things: the recorded state (a fresh engine session on the recording's replay
at the recorded digests), the recorded conversation, and the recorded
initial staffing proposal. Then it runs one arm's decision workflow:
* A: llm-agent 5, validation-first-preview (llm-agent-workflow/v1);
* B: llm-agent 6, consequence acknowledgment
  (llm-agent-acknowledgment-workflow/v1).

The driver does not reimplement either workflow. Each draw calls the
agent's own ask() (llm_agent5.ask, llm_agent6.ask), with its frozen
settings file, its own prompt and its own model client. Four things are
supplied from outside:
* the engine, through a harness at the recorded point that reads the
  engine's lines as the agent's own harness does;
* the memory text, the recording's own, as its replay rebuilt it;
* the first model reply, the recorded proposal, given only after the
  agent's first request equals the recorded request byte for byte;
* the transport of every later request: the agent's ModelClient (live),
  the recorded responses (replay), or scripted replies (tests).
So every message, validation, preview, acknowledgment, retry and final
check is the agent's own.

    python3 -I -B analysis/acknowledgment/stage_b.py schedule --protocol FILE --out DIR
    python3 -I -B analysis/acknowledgment/stage_b.py check --binary ./dt --replays DIR --protocol FILE
    DREAMTEAM_LLM_BASE_URL=... python3 -I -B analysis/acknowledgment/stage_b.py run --binary ./dt \\
        --replays DIR --protocol FILE --out DIR --authorization TEXT
    python3 -I -B analysis/acknowledgment/stage_b.py analyze --binary ./dt --replays DIR --dir DIR --out DIR

A run is never resumed, extended or overwritten: `run` refuses a
directory that already holds one. A run that stops before its last draw is
PARTIAL, and its analysis reports what was observed only.
"""

import argparse
import collections
import json
import math
import os
import subprocess
import sys
import tempfile
import time

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(ROOT, "analysis", "repeatability"))
sys.path.insert(0, os.path.join(ROOT, "agents", "llm"))
import stage_b_points as P  # noqa: E402  (dreamteam-v12b-select/1 and dreamteam-v12b-order/1, unchanged)
W = P.W
X, S, RR, FE, F, A3, A4, A5 = W.X, W.S, W.RR, W.FE, W.F, W.A3, W.A4, W.A5
import llm_agent6 as A6  # noqa: E402  (unchanged: arm B's workflow)

if A6.A3 is not A3 or A6.A5 is not A5:  # one copy of each agent module, so one ProtocolError
    raise ImportError("llm_agent6 loaded another copy of llm_agent3 or llm_agent5")

sha256, canon, write_json, load_jsonl = X.sha256, X.canon, X.write_json, X.load_jsonl
DRIVER = "dreamteam-v12b-driver/1"
PROTOCOL_SCHEMA = "dreamteam.acknowledgment-experiment/v1"
SCHEDULE_SCHEMA = "dreamteam.acknowledgment-schedule/v2"
RUN_SCHEMA = "dreamteam.acknowledgment-run/v1"
DRAW_SCHEMA = "dreamteam.acknowledgment-draw/v1"
ANALYSIS_SCHEMA = "dreamteam.acknowledgment-analysis/v1"
ANALYZER = "acknowledgment-analysis 1"
ARMS = ("A", "B")
ARM_WORKFLOW = {"A": "validation-first-preview", "B": "acknowledgment"}
GEMMA, QWEN = P.GEMMA, P.QWEN
LOCKED = "LOCKED — NOT RUN"
# Every outcome of a draw's proposal (the final decision against the initial
# proposal; the last three: nothing submitted, and why).
OUTCOMES = ("kept", "revised", "abandoned", "refused", "withheld", "invalid")
# Arm B's classes (PROTOCOL.md §5).
CLASSES = ("A_proceeded", "B_revised", "C_abandoned", "D_withheld", "E_not_required")
# Arm A's matching classes: after a shown preview, or none shown.
PREVIEW_CLASSES = ("proceeded", "revised", "abandoned", "failed", "not_shown")


class ExperimentError(Exception):
    pass


class Halt(Exception):
    """A limit reached before a model request: the draw ends unfinished and
    the run stops. Not a ValueError or ProtocolError, so no agent catches it."""

    def __init__(self, code, message):
        super().__init__(f"{code}: {message}")
        self.code, self.message = code, message


ERRORS = (ExperimentError, P.W.ExperimentError, X.ExperimentError, S.SampleError, A3.SettingsError)


# --- the protocol, the points and the schedule -------------------------------

def file_sha256(path):
    with open(path, "rb") as f:
        return sha256(f.read())


def rel(path):
    return os.path.relpath(os.path.abspath(path), ROOT)


def load_protocol(path):
    """The protocol, its SHA-256, the points manifest and contexts (both
    checked against the committed selection)."""
    with open(path, "rb") as f:
        raw = f.read()
    proto = json.loads(raw)
    if proto.get("schema") != PROTOCOL_SCHEMA:
        raise ExperimentError(f"{path} is not a {PROTOCOL_SCHEMA} protocol")
    pdir = os.path.join(os.path.dirname(os.path.abspath(path)), proto["points"])
    man = P.check(pdir)
    with open(os.path.join(pdir, "contexts.jsonl"), "rb") as f:
        contexts = [json.loads(l) for l in f.read().decode("utf-8").splitlines() if l.strip()]
    if [e["id"] for e in contexts] != [s["id"] for s in man["selected"]]:
        raise ExperimentError("contexts.jsonl is not the manifest's selection")
    src = P.committed_contexts()
    for e, s in zip(contexts, man["selected"]):
        o, _, line_sha = src[e["contextKey"]]
        if line_sha != s["sourceLineSha256"] or canon(o["request"]) != canon(e["request"]):
            raise ExperimentError(f"{e['id']}: the context is not its committed source's")
        e["sourceRequest"] = o["request"]  # the recorded request, its keys in the client's order
    return proto, sha256(raw), man, contexts


def initial_request(e):
    """The recorded request the initial proposal answered, as the agent's
    client sent it: from the committed source context, whose keys keep the
    client's order (the Stage B contexts file sorts them)."""
    return e["sourceRequest"]


def schedule(proto, man, contexts):
    """dreamteam-v12b-order/1 (stage_b_points.schedule), each draw with what
    it must start from: the point's digests, the initial request and the
    initial proposal, by SHA-256."""
    by_id = {e["id"]: e for e in contexts}
    out = []
    for s in P.schedule(proto, man):
        e = by_id[s["point"]]
        out.append(dict(s, expected={"digestsSha256": sha256(canon(e["digests"])),
                                     "requestSha256": sha256(X.body_bytes(initial_request(e))),
                                     "proposalSha256": e["proposal"]["sha256"]}))
    return out


def schedule_doc(proto, proto_sha, man, sched):
    return {"schema": SCHEDULE_SCHEMA, "protocol": proto["id"], "protocolSha256": proto_sha,
            "orderRule": proto["orderRule"], "contextsFileSha256": man["contextsFileSha256"], "draws": len(sched),
            "concurrency": 1, "maxModelCallsPerDraw": proto["maxModelCallsPerDraw"],
            "maxModelCalls": proto["maxModelCalls"], "schedule": sched}


def check_schedule(proto, sched, path, proto_sha=None):
    """The committed schedule file: the one the protocol and points make,
    byte for byte, and the one the protocol pins."""
    with open(path, "rb") as f:
        raw = f.read()
    doc = json.loads(raw)
    if doc.get("schedule") != sched:
        raise ExperimentError("schedule.json is not the schedule the protocol and points make")
    if proto_sha is not None and (doc.get("schema") != SCHEDULE_SCHEMA or doc.get("protocolSha256") != proto_sha):
        raise ExperimentError("schedule.json is not made from this protocol (schema or protocolSha256)")
    pin = (proto.get("lock") or {}).get("scheduleEntriesSha256")  # the entries: the file names the protocol's hash
    if pin is not None and pin != sha256(canon(doc["schedule"])):
        raise ExperimentError("schedule.json is not the schedule the protocol pins")
    if len(sched) != proto["draws"] or len({s["seq"] for s in sched}) != len(sched):
        raise ExperimentError("the schedule does not have the protocol's draws, each once")
    return doc


def check_lock(proto, proto_path):
    """A locked protocol's pins, against the files as they are now: every
    pinned file's SHA-256, and the status. Returns the problems."""
    lock = proto.get("lock")
    if not lock:
        return ["the protocol is not locked"]
    problems = []
    if proto.get("status") != LOCKED:
        problems.append(f"status is not {LOCKED!r}")
    for p, want in sorted((lock.get("files") or {}).items()):
        full = os.path.join(ROOT, p)
        if not os.path.exists(full):
            problems.append(f"{p}: missing")
        elif file_sha256(full) != want:
            problems.append(f"{p}: SHA-256 differs from the lock")
    ident = {"A": (A5.WORKFLOW_VERSION, A5.WORKFLOW_SHA256, None, None),
             "B": (A6.WORKFLOW_VERSION, A6.WORKFLOW_SHA256, A6.CONTRACT, A6.CONTRACT_SHA256)}
    for arm, (wv, ws, c, cs) in ident.items():
        a = proto["arms"][arm]
        if (a["workflowVersion"], a["workflowSha256"], a.get("contract"), a.get("contractSha256")) != (wv, ws, c, cs):
            problems.append(f"arm {arm}: the workflow or contract identity is not the agent's")
    return problems


# --- the engine at a point ------------------------------------------------------

def spawn(binary, run, loads):
    """A fresh engine session on the run's scenario and seed, its lines
    read with `loads` (the arm's agent's own reader): (ask, close)."""
    scratch = tempfile.TemporaryDirectory(prefix="v12b-engine-")
    proc = subprocess.Popen([os.path.abspath(binary), "agent", "--stdio", "-scenario", run["scenario"]["ref"],
                             "-seed", str(run["seed"])], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL, text=True, encoding="utf-8", cwd=scratch.name)

    def ask(r):
        proc.stdin.write(json.dumps(r) + "\n")
        proc.stdin.flush()
        line = proc.stdout.readline()
        if not line:
            raise ExperimentError("the engine closed the session")
        try:
            return loads(line)
        except ValueError as e:
            raise ExperimentError(f"the engine sent a line that is not JSON: {e}")

    def close():
        try:
            proc.stdin.close()
        finally:
            proc.wait()
            scratch.cleanup()
    return ask, close


READERS = {"A": json.loads, "B": A6.json_loads}  # llm_agent3.Harness._read; llm_agent6.Harness._read


class EngineHarness:
    """The agent's harness interface over an engine session at the point:
    request(op, args) -> result, raising the agent's ProtocolError on a
    refusal (llm_agent3.Harness.request), with the agent's request ids;
    every request and answer logged."""

    def __init__(self, ses):
        self.ses, self.log, self.n = ses, [], 0

    def request(self, op, args=None):
        self.n += 1
        rid = f"r{self.n}"
        a = self.ses.request(rid, op, args)
        if a.get("id") != rid or a.get("protocol") != A3.AGENT_PROTOCOL:  # as llm_agent3.Harness.request
            raise ExperimentError(f"the engine answered {a.get('id')!r} ({a.get('protocol')!r}) to {rid}")
        self.log.append({"op": op, "args": args, "answer": a})
        if not a.get("ok"):
            raise A3.ProtocolError(op, a.get("error") or {})
        return a["result"]


def open_point(binary, run_dir, e, arm):
    """A fresh engine session at the point (its recorded requests fed and
    every answer checked), and its harness; the state checked."""
    run, pairs = X.transcript(run_dir)
    ask, close = spawn(binary, run, READERS[arm])
    ses = X.Session(run, pairs, ask, close)
    try:
        ses.to_point(e["digests"])
    except BaseException:
        ses.close()
        raise
    return ses


class FixedMemory:
    """The recording's memory at the point: its text as the recording's
    replay rebuilt it (llm_agent3.Memory.render), given to the agent's
    prepare(); nothing is recorded into it."""

    def __init__(self, text):
        self.text = text

    def observe(self, point, obs):
        pass

    def render(self, pending):
        return self.text

    def record(self, rec):
        pass


def memory_text(run_dir, e):
    """The memory text at the point, from the replay's evidence, checked
    against the recording's memorySha256."""
    rec = S.recording_point(ROOT, e["recording"], e["point"])
    for m in load_jsonl(os.path.join(run_dir, "policy-evidence", "memory.jsonl")):
        if m.get("point") == e["point"]:
            if sha256(m["text"]) != rec["memorySha256"]:
                raise ExperimentError(f"{e['id']}: the replay's memory is not the recording's")
            return m["text"]
    raise ExperimentError(f"{e['id']}: the replay has no memory at the point")


SETTINGS_ENV = "DREAMTEAM_LLM_BASE_URL"
REPLAY_URL = "http://replay.invalid/v1"  # no request is ever sent to it


def settings_of(proto, model, env):
    """The model's frozen settings file (docs/benchmark/v05c-dev), loaded by
    the agent's own loader; checked against the protocol."""
    f = proto["settings"]["files"][model]
    path = os.path.join(ROOT, f["file"])
    if file_sha256(path) != f["sha256"]:
        raise ExperimentError(f"{f['file']}: not the protocol's settings file")
    s = A3.load_settings(path, env)
    if s["model"] != proto["models"][model]:
        raise ExperimentError(f"{model}: the settings name {s['model']!r}, not the protocol's model")
    if sha256(s["systemPrompt"]) != proto["systemPromptSha256"]:
        raise ExperimentError("the system prompt is not the protocol's")
    if s["requestTimeoutSeconds"] != proto["request"]["timeoutSeconds"] or \
            (s["maxRetries"], s["allowDetails"]) != (proto["settings"]["maxRetries"], proto["settings"]["allowDetails"]):
        raise ExperimentError(f"{model}: the settings' timeout, maxRetries or allowDetails are not the protocol's")
    return s


# --- the model ------------------------------------------------------------------

class Limits:
    """The run's limits, checked before every model request: the calls per
    draw and in all, and the wall clock (a request starts only if its whole
    timeout fits before the stop)."""

    def __init__(self, per_draw, total, wall_seconds, timeout, now=time.monotonic, t0=None):
        self.per_draw, self.total, self.wall, self.timeout, self.now = per_draw, total, wall_seconds, timeout, now
        self.t0 = now() if t0 is None else t0
        self.calls = 0

    def elapsed(self):
        return self.now() - self.t0

    def before_call(self, draw_calls):
        if draw_calls >= self.per_draw:
            raise Halt("draw_call_limit", f"a draw asked for model call {draw_calls + 1}; the limit is {self.per_draw}")
        if self.calls >= self.total:
            raise Halt("call_budget", f"{self.calls} model calls made; the limit is {self.total}")
        if self.elapsed() + self.timeout > self.wall:
            raise Halt("wall_clock", f"{self.elapsed():.0f} s elapsed; a request of up to {self.timeout} s would "
                                     f"pass the {self.wall} s stop")

    def before_draw(self):
        if self.elapsed() + self.timeout > self.wall:
            raise Halt("wall_clock", f"{self.elapsed():.0f} s elapsed; the next draw could pass the {self.wall} s stop")


class PointModel:
    """The agent's model interface at a branch point (payload, call, spent:
    llm_agent6.LiveModel's; llm_agent5.ask uses payload and call).

    payload: the agent's own client's body. The first request must be the
    recorded request, byte for byte; it is not sent, and the recorded
    proposal is its reply. Every later request goes to the transport, after
    the limits; each is recorded in `calls`."""

    def __init__(self, e, client, transport, limits, utc=RR.utc, now=time.monotonic):
        self.e, self.client, self.transport, self.limits, self.utc, self.now = e, client, transport, limits, utc, now
        self.first, self.calls, self.start = True, [], now()
        self.settings = client.s

    def payload(self, messages):
        return self.client.payload(messages)

    def call(self, payload):
        data = json.dumps(payload).encode("utf-8")  # what ModelClient.call sends
        if self.first:
            self.first = False
            want = X.body_bytes(initial_request(self.e))
            if data != want:
                raise ExperimentError(f"{self.e['id']}: the agent's first request is not the recorded request")
            return 200, json.dumps({"choices": [{"message": {"content": self.e["proposal"]["content"]}}]}), None
        if self.limits is not None:
            self.limits.before_call(len(self.calls))
        sent, t0 = self.utc(), self.now()
        try:
            status, body, err = self.transport(payload)
        except BaseException as x:  # recorded and counted, then raised on (review D6, D7)
            if self.limits is not None:
                self.limits.calls += 1
            self.calls.append({"n": len(self.calls) + 1, "requestSha256": sha256(data), "status": None,
                               "error": f"client: {type(x).__name__}: {x}", "body": None,
                               "responseSha256": sha256(""), "usage": None, "finishReason": None,
                               "servedModel": None, "sent": sent, "latencyMs": round((self.now() - t0) * 1000)})
            raise
        lat = self.now() - t0
        if self.limits is not None:
            self.limits.calls += 1
        f = body_fields(body)
        self.calls.append({"n": len(self.calls) + 1, "requestSha256": sha256(data), "status": status, "error": err,
                           "body": body, "responseSha256": sha256(body or ""),
                           "usage": f.get("usage") if f.get("parsedBody") else None,
                           "finishReason": f.get("finishReason") if f.get("parsedBody") else None,
                           "servedModel": f.get("servedModel") if f.get("parsedBody") else None,
                           "sent": sent, "latencyMs": round(lat * 1000)})
        return status, body, err

    def spent(self, attempts):  # llm_agent6.LiveModel.spent, unchanged
        return bool(attempts) and self.now() - self.start >= self.settings["decisionBudgetSeconds"]


def body_fields(body):
    """repeat_run.response_fields, for any body: one that is not a JSON
    object of the usual shape has no fields (re-review R3)."""
    if not body:
        return {"parsedBody": False}
    try:
        return RR.response_fields(body)
    except (ValueError, AttributeError, TypeError, IndexError, KeyError):
        return {"parsedBody": False}


def replay_transport(e, recorded):
    """The recorded responses in order, each only for the request it
    answered (by SHA-256), each body checked against its hash."""
    it = iter(recorded)

    def transport(payload):
        r = next(it, None)
        if r is None or r["requestSha256"] != sha256(json.dumps(payload).encode("utf-8")):
            raise ExperimentError(f"{e['id']}: the workflow sent a request the run did not record")
        if r["responseSha256"] != sha256(r["body"] or ""):
            raise ExperimentError(f"{e['id']}: a recorded response is not the one its hash records")
        return r["status"], r["body"], r["error"]
    return transport


# --- one draw -------------------------------------------------------------------

DETERMINISTIC = ("schema", "seq", "model", "point", "arm", "draw", "expected", "status", "halt", "initialDigests",
                 "record", "batches", "engine", "digestsAfter")


def run_draw(entry, e, arm_settings, opener, mem, transport, limits, utc=RR.utc, now=time.monotonic):
    """One scheduled draw: a fresh engine at the point, the arm's agent's
    ask(), the recorded proposal as its first reply. Returns the draw's
    record (dreamteam.acknowledgment-draw/v1)."""
    arm = entry["arm"]
    if arm not in ARMS:
        raise ExperimentError(f"draw {entry['seq']}: arm {arm!r} is not an arm of the protocol")
    exp = entry["expected"]
    if exp["proposalSha256"] != sha256(e["proposal"]["content"]) or \
            exp["requestSha256"] != sha256(X.body_bytes(initial_request(e))):
        raise ExperimentError(f"draw {entry['seq']}: the point's proposal or request is not the schedule's")
    out = {"schema": DRAW_SCHEMA, **{k: entry[k] for k in ("seq", "model", "point", "arm", "draw", "expected")},
           "status": None, "halt": None, "initialDigests": None, "record": None, "batches": None, "engine": [],
           "digestsAfter": None, "calls": []}
    ses = opener(e, arm)
    h = EngineHarness(ses)
    model = PointModel(e, A3.ModelClient(arm_settings), transport, limits, utc=utc, now=now)
    try:
        d = h.request("digests")["digests"]
        if sha256(canon(d)) != exp["digestsSha256"]:
            raise ExperimentError(f"draw {entry['seq']}: the engine is not at the recorded state")
        out["initialDigests"] = d
        h.log = []
        sink = Sink()
        try:
            if arm == "A":
                batches, rec = A5.ask(h, sink, model, arm_settings, mem, e["point"], None,
                                      workflow=ARM_WORKFLOW["A"])
            else:
                batches, rec = A6.ask(h, sink, model, arm_settings, mem, e["point"], None)
                rec["workflow"] = ARM_WORKFLOW["B"]  # as llm_agent6.run records it
        except Halt as x:
            out.update(status="interrupted", halt={"code": x.code, "message": x.message})
        else:
            if model.first:
                raise ExperimentError(f"draw {entry['seq']}: the agent made no request")
            out.update(status="completed", record=rec, batches=batches)
        out["engine"] = h.log
        h.log = []
        out["digestsAfter"] = h.request("digests")["digests"]
    except BaseException as x:  # the unfinished draw goes with the exception (review D6, D7)
        out["status"] = "aborted"
        out["engine"] = out["engine"] or h.log
        x.partialDraw = out
        raise
    finally:
        out["calls"] = model.calls
        ses.close()
    return out


class Sink:
    """The agent's evidence interface: the driver keeps its own records
    (the model calls in PointModel, the decision record ask returns)."""

    def append(self, name, obj):
        pass

    def call(self, rec):
        pass


def deterministic(draw):
    """A draw without its live metadata (the calls' send times and
    latencies): what replay must reproduce."""
    out = {k: draw[k] for k in DETERMINISTIC}
    out["calls"] = [{k: c[k] for k in ("n", "requestSha256", "status", "error", "responseSha256")}
                    for c in draw["calls"]]
    return out


# --- the run ------------------------------------------------------------------------

class Points:
    """Per point: the replay run, its memory text; per model: settings."""

    def __init__(self, proto, contexts, replays, env):
        runs = X.replay_runs(replays)
        self.by_id = {e["id"]: e for e in contexts}
        self.run_dir, self.mem = {}, {}
        for e in contexts:
            name = os.path.basename(e["recording"])
            if name not in runs:
                raise ExperimentError(f"{e['id']}: no replay of {name} in {replays}")
            self.run_dir[e["id"]] = runs[name]
            self.mem[e["id"]] = memory_text(runs[name], e)
        self.settings = {m: settings_of(proto, m, env) for m in proto["modelOrder"]}

    def opener(self, binary):
        """A pool of engine sessions, one per point and arm (Pool)."""
        return Pool(lambda e, arm: open_point(binary, self.run_dir[e["id"]], e, arm))


class Pool:
    """One engine session per point and arm, opened at its first draw and
    kept for that arm's later draws at the point: the arms never share an
    engine process. Every draw checks the state at its start (the
    schedule's digests) and records it at its end; the agents' requests at
    a point (validate, preview, observe, actions, digests) are read-only.
    Calling the pool gives the draw its session; the draw's close leaves it
    open for the next."""

    def __init__(self, open_fn):
        self.open_fn, self.sessions = open_fn, {}

    def __call__(self, e, arm):
        k = (e["id"], arm)
        if k not in self.sessions:
            ses = self.open_fn(e, arm)
            ses.model = e["model"]
            self.sessions[k] = ses
        return Shared(self.sessions[k])

    def close(self, model=None):
        """Every session (or every session of one model's points)."""
        for k in [k for k, v in self.sessions.items() if model is None or v.model == model]:
            self.sessions.pop(k).close()


class Shared:
    def __init__(self, ses):
        self.ses = ses

    def request(self, rid, op, args=None):
        return self.ses.request(rid, op, args)

    def close(self):
        pass


def check_points_run(binary, proto, pts):
    """Before any request: at every point and arm, a fresh engine at the
    recorded state, and the agent's first request equal to the recorded
    request (its prompt rebuilt byte for byte). No model is called."""
    out, pool = [], pts.opener(binary)
    for pid, e in pts.by_id.items():
        pool.close()
        for arm in ARMS:
            def refuse(payload):
                raise ExperimentError("the check sends no model request")

            entry = {"seq": 0, "model": e["model"], "point": pid, "arm": arm, "draw": 0,
                     "expected": {"digestsSha256": sha256(canon(e["digests"])),
                                  "requestSha256": sha256(X.body_bytes(initial_request(e))),
                                  "proposalSha256": e["proposal"]["sha256"]}}
            limits = Limits(0, 0, 1, 0)  # any model request halts
            d = run_draw(entry, e, pts.settings[e["model"]], pool, FixedMemory(pts.mem[pid]), refuse, limits)
            out.append({"point": pid, "arm": arm, "status": d["status"],
                        "halt": (d["halt"] or {}).get("code"), "promptSha256": sha256(
                            initial_request(e)["messages"][1]["content"]),
                        "stateUnchanged": d["digestsAfter"] == e["digests"]})
    pool.close()
    return out


def run(proto, proto_sha, man, contexts, out, binary, replays, authorization, transport_factory, env,
        state_fn=RR.server_state, now=time.monotonic, utc=RR.utc, base=None, lock_check=True, proto_path=None,
        pts=None, client_info=None):
    """The live run (PROTOCOL.md §4): the schedule's draws in order, one at
    a time, each recorded as it ends; stopped by the first stop condition.
    Never resumed: a directory with a run in it is refused."""
    live = os.path.join(out, "live")
    if os.path.exists(live):
        raise ExperimentError(f"{live} exists: a run is never resumed, extended or overwritten")
    if not authorization or not authorization.strip():
        raise ExperimentError("no --authorization: the lock does not authorize a run; the owner does")
    if lock_check:
        problems = check_lock(proto, proto_path)
        if problems:
            raise ExperimentError("the protocol lock does not hold: " + "; ".join(problems))
    sched = schedule(proto, man, contexts)
    check_schedule(proto, sched, os.path.join(os.path.dirname(os.path.abspath(proto_path)), "schedule.json"),
                   proto_sha if lock_check else None)
    pts = pts or Points(proto, contexts, replays, env)
    opener = pts.opener(binary)
    os.makedirs(live)
    stop = proto["stop"]
    limits = Limits(proto["maxModelCallsPerDraw"], proto["maxModelCalls"], stop["wallClockSeconds"],
                    proto["request"]["timeoutSeconds"], now=now)
    started = utc()
    before = state_fn(base)
    write_json(os.path.join(live, "server-before.json"), before)
    sig = X.server_signature(before)
    fails, completed, stopped, checks, prev, interrupted, aborted = 0, 0, None, [], None, None, None
    if (before.get("health") or {}).get("status") != "healthy":
        stopped = {"code": "health", "message": "the server did not answer healthy before the run"}
    transports = {m: transport_factory(pts.settings[m]) for m in proto["modelOrder"]}
    try:
        with open(os.path.join(live, "draws.jsonl"), "w", encoding="utf-8") as f:
            for r in sched if stopped is None else []:
                if (r["model"], r["draw"]) != prev:
                    if prev is not None and prev[0] != r["model"]:
                        opener.close(prev[0])  # one model's sessions open at a time
                    prev = (r["model"], r["draw"])
                    st = state_fn(base)
                    checks.append({"seq": r["seq"], "model": r["model"], "draw": r["draw"], "time": st.get("time"),
                                   "health": st.get("health"), "sameConfiguration": X.server_signature(st) == sig})
                    if (st.get("health") or {}).get("status") != "healthy":
                        stopped = {"code": "health", "message": f"the server did not answer healthy before draw {r['seq']}"}
                        break
                    if X.server_signature(st) != sig:
                        stopped = {"code": "server_change", "message": f"server configuration changed before draw {r['seq']}"}
                        break
                try:
                    limits.before_draw()
                except Halt as x:
                    stopped = {"code": x.code, "message": x.message}
                    break
                e = pts.by_id[r["point"]]
                try:
                    d = run_draw(r, e, pts.settings[r["model"]], opener, FixedMemory(pts.mem[e["id"]]),
                                 transports[r["model"]], limits, utc=utc, now=now)
                except BaseException as x:  # the run ends cleanly, and says why
                    aborted = r["seq"]
                    if getattr(x, "partialDraw", None) is not None:  # its calls and engine requests so far
                        write_json(os.path.join(live, "aborted-draw.json"), x.partialDraw)
                    if isinstance(x, KeyboardInterrupt):
                        stopped = {"code": "operator", "message": f"interrupted by the operator during draw {r['seq']}"}
                        break
                    if not isinstance(x, Exception):
                        stopped = {"code": "driver", "message": f"draw {r['seq']}: {type(x).__name__}: {x}"}
                        raise
                    stopped = {"code": "driver", "message": f"draw {r['seq']}: {type(x).__name__}: {x}"}
                    break
                f.write(json.dumps(d, sort_keys=True, ensure_ascii=False) + "\n")
                f.flush()
                if d["status"] != "completed":
                    interrupted, stopped = r["seq"], dict(d["halt"])
                    break
                completed += 1
                for c in d["calls"]:
                    fails = fails + 1 if c["error"] and c["status"] is None else 0
                if fails >= stop["consecutiveTransportFailures"]:
                    stopped = {"code": "transport", "message": f"{fails} consecutive transport failures"}
                    break
    except KeyboardInterrupt:
        stopped = {"code": "operator", "message": "interrupted by the operator"}
    finally:
        opener.close()
        after = state_fn(base)
        write_json(os.path.join(live, "server-after.json"), after)
        status = "COMPLETE" if completed == len(sched) and stopped is None else "PARTIAL"
        doc = {"schema": RUN_SCHEMA, "driver": DRIVER, "protocol": proto["id"], "protocolSha256": proto_sha,
               "contextsFileSha256": man["contextsFileSha256"], "authorization": authorization,
               "started": started, "ended": utc(), "status": status,
               "drawsScheduled": len(sched), "drawsCompleted": completed, "drawInterrupted": interrupted,
               "drawAborted": aborted, "drawsNotRun": len(sched) - completed - (1 if interrupted else 0) - (1 if aborted else 0),
               "modelCalls": limits.calls, "elapsedSeconds": round(limits.elapsed(), 1), "stopped": stopped,
               "serverChecks": checks, "sameConfigurationAfter": X.server_signature(after) == sig,
               "resume": "never: a continuation needs the owner's separate authorization and a documented "
                         "protocol decision",
               "endpoint": "from " + proto["endpointEnv"] + " (not recorded)",
               "client": client_info or {"python": sys.version.split()[0]}}
        write_json(os.path.join(live, "run.json"), doc)
    return doc


# --- replay ---------------------------------------------------------------------------

def check_rows(sched, rows):
    """The recorded draws: the schedule's first draws, in its order, each
    once, with its point, model, arm, draw and expected start; only the last
    may be unfinished."""
    if len(rows) > len(sched):
        raise ExperimentError("more draws recorded than scheduled")
    for k, (r, s) in enumerate(zip(rows, sched)):
        if r.get("schema") != DRAW_SCHEMA:
            raise ExperimentError(f"draw {k + 1}: not a {DRAW_SCHEMA} record")
        got = {x: r.get(x) for x in ("seq", "model", "point", "arm", "draw", "expected")}
        if got != s:
            raise ExperimentError(f"draw {k + 1}: not the schedule's draw {s['seq']} (seq, point, model, arm, draw or "
                                  f"expected start differ)")
        if r["status"] != "completed" and k != len(rows) - 1:
            raise ExperimentError(f"draw {r['seq']}: an unfinished draw before the last")
        if r["status"] not in ("completed", "interrupted"):
            raise ExperimentError(f"draw {r['seq']}: unknown status {r['status']!r}")


def check_run_doc(run_doc, rows, sched):
    """run.json against the recorded draws (review D11): its counts, its
    interrupted draw and its status."""
    completed = sum(1 for r in rows if r["status"] == "completed")
    last = rows[-1] if rows else None
    if run_doc["status"] not in ("COMPLETE", "PARTIAL") or run_doc["drawsCompleted"] != completed:
        raise ExperimentError("run.json does not describe the recorded draws")
    if run_doc.get("drawInterrupted") != (last["seq"] if last and last["status"] == "interrupted" else None):
        raise ExperimentError("run.json's drawInterrupted is not the recorded interrupted draw")
    if run_doc["status"] == "COMPLETE" and (run_doc.get("stopped") is not None or completed != len(sched)
                                            or run_doc.get("drawAborted") is not None):
        raise ExperimentError("run.json says COMPLETE, but not every scheduled draw was completed without a stop")
    if run_doc["status"] == "PARTIAL" and run_doc.get("stopped") is None and completed == len(sched):
        raise ExperimentError("run.json says PARTIAL with every draw completed and no stop")


def replay_draw(r, e, arm_settings, opener, mem):
    """One recorded draw run again without a model: a fresh engine, the
    agent's ask(), the recorded responses. Everything but the live metadata
    must be the recorded draw's, and every recorded response used."""
    left = list(r["calls"])
    used = []

    def transport(payload):
        if not left:
            raise ExperimentError(f"draw {r['seq']}: the workflow sent a request the run did not record")
        c = left.pop(0)
        used.append(c)
        return replay_transport(e, [c])(payload)
    halt = None
    if r["status"] == "interrupted":
        n = len(r["calls"])
        halt = r["halt"]["code"]

        class At(Limits):
            def before_call(self, draw_calls):
                if draw_calls >= n:
                    raise Halt(halt, r["halt"]["message"])
        limits = At(0, 0, 0, 0)
    else:
        limits = None
    entry = {k: r[k] for k in ("seq", "model", "point", "arm", "draw", "expected")}
    again = run_draw(entry, e, arm_settings, opener, mem, transport, limits, utc=lambda: None, now=lambda: 0.0)
    if left:
        raise ExperimentError(f"draw {r['seq']}: recorded responses the workflow did not use")
    for c in r["calls"]:  # the body-derived metadata, from the hash-checked body (review D5)
        f = body_fields(c["body"])
        derived = {k: f.get(k) if f.get("parsedBody") else None for k in ("usage", "finishReason", "servedModel")}
        if derived != {k: c.get(k) for k in derived}:
            raise ExperimentError(f"draw {r['seq']}, call {c['n']}: usage, finish reason or served model is not "
                                  "the recorded body's")
    got, want = deterministic(again), deterministic(r)
    if got != want:
        raise ExperimentError(f"draw {r['seq']}: the replayed draw differs from the recorded one at "
                              f"{A6.first_difference(got, want, 'draw')}")
    return again


# --- classification -------------------------------------------------------------------

def last_proposal(atts):
    """The last attempt with a parsed proposal (not a details request)."""
    return next((a for a in reversed(atts) if (a.get("parse") or {}).get("ok") and a.get("detailsRequested") is None
                 and isinstance(a.get("actions"), list)), None)


def requests_of(rec):
    """Arm B: every acknowledgment request shown, in order (the attempts
    whose proposal qualified)."""
    return [a for a in rec["attempts"] if (a.get("assessment") or {}).get("status") == "qualifying"]


def ack_entries(rec):
    """Arm B: every reply that carried an acknowledgment, with whether it
    was accepted, its failure codes, and whether it states the consequences
    of the request it answered exactly (accurate), even when the reply
    changed the actions (llm_agent6.match against that request)."""
    out, pending = [], None
    for a in rec["attempts"]:
        ack = a.get("acknowledgment")
        if ack is not None:
            m = ack["match"]
            accurate = None
            if pending is not None and ack.get("given") is not None:
                accurate = A6.match(ack["given"], pending["assessment"])["accepted"]
            out.append({"attempt": a["attempt"], "forAttempt": ack["forAttempt"], "present": ack["present"],
                        "accepted": m["accepted"], "failures": sorted({x["code"] for x in m["failures"]}),
                        "accurate": accurate, "sameActions": ack["forAttempt"] is not None})
        if (a.get("assessment") or {}).get("status") == "qualifying":
            pending = a
    return out


def classify(r, e):
    """A completed draw's outcomes (PROTOCOL.md §5)."""
    rec = r["record"]
    atts = rec["attempts"]
    if not atts or atts[0].get("actions") != e["proposal"]["actions"]:
        raise ExperimentError(f"draw {r['seq']}: the first attempt is not the recorded proposal")
    # The agent's own parse of the recorded proposal, keys in the reply's
    # order: llm-agent 4's rule compares actions by their compact JSON,
    # which the Stage B contexts file (keys sorted) would not match.
    initial = atts[0]["actions"]
    # Nothing submitted when the agent's outcome is fallback, even with a
    # used attempt: a state_changed withholding (re-review R2).
    used = rec["final"]["attempt"] if rec["outcome"] != "fallback" else None
    final = atts[used - 1]["actions"] if used else None
    if final is not None:
        outcome = response_of(initial, final)
    elif atts[-1].get("refusals"):
        outcome = "refused"
    elif rec.get("withheld") or rec["final"]["attempt"] is not None:
        outcome = "withheld"
    else:
        outcome = "invalid"
    # The final proposal's validity: the last validation of the last
    # proposal's actions (an acknowledgment reply repeating them is
    # validated only by the final check, when accepted; review D2).
    lp = last_proposal(atts)
    lv = next((a for a in reversed(atts) if lp is not None and a.get("validation") is not None
               and isinstance(a.get("actions"), list) and A6.same(a["actions"], lp["actions"])), None)
    lp_valid = bool(lv and not lv.get("refusals") and lv.get("protocolError") is None
                    and all((v.get("answer") or {}).get("valid") for v in lv["validation"]))
    out = {"seq": r["seq"], "point": r["point"], "model": r["model"], "arm": r["arm"], "draw": r["draw"],
           "outcome": outcome, "finalActions": final, "finalActionsKey": sha256(canon(final)) if final is not None else None,
           "finalProposalValid": lp_valid, "submitted": bool(r["batches"]), "noAction": rec["outcome"] == "no_action",
           "modelCalls": len(r["calls"]), "attempts": len(atts),
           "retries": sum(1 for a in atts if a.get("turn") == "retry"),
           "inputs": [(a.get("input") or {}).get("kind") for a in atts[1:]],
           "tokens": tokens(r["calls"]), "stopped": rec["completeness"].get("stopped")}
    if r["arm"] == "B":
        reqs = requests_of(rec)
        acks = ack_entries(rec)
        out["acknowledgment"] = {
            "requests": [{"attempt": a["attempt"], "token": a["assessment"]["payload"]["token"],
                          "items": [i["type"] + ":" + next(iter(i["consequences"])) for i in a["assessment"]["items"]]}
                         for a in reqs],
            "replies": acks, "accepted": sum(1 for x in acks if x["accepted"]),
            "accurate": sum(1 for x in acks if x["accurate"]),
            "failureCodes": sorted({c for x in acks for c in x["failures"]}),
            "finalCheck": next(((a["finalCheck"] or {}).get("agrees") for a in reversed(atts) if a.get("finalCheck")),
                               None),
            "withheld": (rec.get("withheld") or {}).get("reason"),
            "assessments": [a["assessment"]["status"] for a in atts if a.get("assessment")]}
        out["class"], out["reorderedOnly"] = ack_class(reqs, acks, used, final, atts)
        out["accurateBeforeChange"] = any(x["accurate"] and not x["sameActions"] for x in acks) if \
            out["class"] in ("B_revised", "C_abandoned") else None
        out["dReason"] = d_reason(rec, atts) if out["class"] == "D_withheld" else None
    else:
        out["preview"] = preview_class(atts, used, final)
        out["reorderedOnly"] = out["preview"].pop("reorderedOnly")
    return out


def d_reason(rec, atts):
    """Why a class D draw submitted nothing: the agent's withheld reason, or
    how the workflow ended when it records none (review D4)."""
    w = (rec.get("withheld") or {}).get("reason")
    if w:
        return w  # the agent's: acknowledgment_not_accepted, final_check_disagrees, consequences_unavailable, state_changed
    last = atts[-1]
    if last.get("refusals"):
        return "revision_refused"
    if last.get("output") is None:
        return "failed_request"
    return "invalid_reply"


def same_content(a, b):
    """The same actions by canonical content: each action's keys sorted,
    the actions in any order (llm-agent 4's same_actions compares compact
    JSON, which is sensitive to key order; re-review R1)."""
    return sorted(canon(x) for x in a) == sorted(canon(x) for x in b)


def response_of(proposal, actions):
    """llm-agent 4's kept, revised or abandoned, by canonical content."""
    if same_content(proposal, actions):
        return "kept"
    return "abandoned" if not actions else "revised"


def unchanged(final, actions):
    """(the same actions as `actions` by canonical content: the outcome's
    rule; and only reordered). One rule for "unchanged" in every
    classification (review D1, re-review R1); a resubmission that reorders
    the list, which llm-agent 6 treats as a new proposal, is flagged. Key
    order inside an action is no change at all."""
    same = same_content(actions, final)
    return same, same and not A6.same(final, actions)


def ack_class(reqs, acks, used, final, atts):
    """Arm B's class and whether the submission only reordered R.
    Acknowledgment is required at the first proposal (R) whose assessment
    qualified (a request was shown) or could not be read (unavailable:
    withheld, fail-closed). Then: A proceeded (R submitted unchanged, in
    canonical content), B revised (other non-empty actions submitted), C
    abandoned ("actions": [] submitted), D withheld (nothing submitted).
    E: acknowledgment never required."""
    first = next((a for a in atts if (a.get("assessment") or {}).get("status") in ("qualifying", "unavailable")), None)
    if first is None:
        return "E_not_required", False
    if used is None:
        return "D_withheld", False
    if first["assessment"]["status"] == "unavailable":
        raise ExperimentError("a proposal whose consequences could not be read was followed by a submission")
    if final == []:
        return "C_abandoned", False
    same, reordered = unchanged(final, first["actions"])
    if same:
        if not any(x["accepted"] for x in acks):
            raise ExperimentError("a qualifying proposal was submitted without an accepted acknowledgment")
        return "A_proceeded", reordered
    return "B_revised", False


def preview_class(atts, used, final):
    """Arm A: the selective preview and what followed it."""
    pv = next((a for a in atts if a.get("preview")), None)
    if pv is None:
        return {"status": "none", "class": "not_shown", "reasons": None, "reorderedOnly": False}
    p = pv["preview"]
    info = {"status": p["status"], "reasons": (p.get("eligibility") or {}).get("reasons"), "reorderedOnly": False}
    if p["status"] != "shown":
        return dict(info, **{"class": "not_shown"})
    if used is None:
        return dict(info, **{"class": "failed"})
    if final == []:
        return dict(info, **{"class": "abandoned"})
    same, reordered = unchanged(final, pv["actions"])
    return dict(info, **{"class": "proceeded" if same else "revised", "reorderedOnly": reordered})


def tokens(calls):
    u = [c.get("usage") or {} for c in calls]
    return {"prompt": sum(x.get("prompt_tokens") or 0 for x in u),
            "completion": sum(x.get("completion_tokens") or 0 for x in u),
            "reported": sum(1 for x in u if x)}


# --- the continuation and financial evidence ---------------------------------------

def exposure(ev):
    """Engine-reported direct consequences of a decision at the recorded
    state (financial evidence v1 and the engine's previews of its staffing
    batches), as reported: nothing inferred."""
    pvs = [x.get("preview") for x in ev["records_events"] if x.get("preview")]
    terms = [t for p in pvs for t in p.get("terminations") or []]
    guar = [g for p in pvs for g in p.get("guaranteed") or []]
    fe = ev["financialEvidence"]
    return {"commitmentCents": fe["commitmentCents"], "creditSupported": fe["creditSupported"],
            "severanceCents": [t.get("severanceCents") for t in terms],
            "retainedObligations": [{"label": g.get("label"), "cents": g.get("cents"), "when": g.get("when")}
                                    for g in guar],
            "displacedBookings": sum(len(p.get("displacements") or []) for p in pvs),
            "capacity": fe["directExposure"], "reversibility": fe["reversibility"],
            "affordability": fe["affordability"]}


def continuation(binary, run_dir, e, actions):
    """The decision applied at the recorded state, then the no-action
    suffix (shared-prefix/v2's decision_evidence, unchanged), and the
    engine's previews of its staffing batches."""
    ev = X.decision_evidence(binary, run_dir, e, actions)
    ses = X.open_session(binary, run_dir)
    events = []
    try:
        ses.to_point(e["digests"])
        for b, _ in A3.split(actions):
            if FE.staffing_batch(b):
                pv = ses.request("v12b-preview", "preview", {"actions": b})
                events.append({"preview": (pv.get("result") or {}).get("preview") if pv.get("ok") else None})
    finally:
        ses.close()
    ev["records_events"] = events
    return {"actionsKey": sha256(canon(actions)), "actions": actions,
            **{k: ev[k] for k in ("validation", "outcome", "applied", "validateApplyAgree", "digestsAfterDecision",
                                  "suffix")},
            "executed": bool(ev["applied"]) and all(a["applied"] for a in ev["applied"]),
            "ledgerSha256": sha256(canon(ev["records"])), "exposure": exposure(ev)}


# --- the analysis -----------------------------------------------------------------

def wilson(k, n):
    return X.wilson(k, n)


def arm_summary(ds, scheduled, arm):
    n = len(ds)
    cnt = collections.Counter(x["outcome"] for x in ds)
    s = {"scheduled": scheduled, "observed": n,
         "outcomes": {o: {"count": cnt.get(o, 0), "wilson95": wilson(cnt.get(o, 0), n)} for o in OUTCOMES},
         "submitted": sum(1 for x in ds if x["submitted"]), "noAction": sum(1 for x in ds if x["noAction"]),
         "executed": sum(1 for x in ds if x.get("executed")),
         "finalProposalValid": sum(1 for x in ds if x["finalProposalValid"]),
         "reorderedOnly": sum(1 for x in ds if x["reorderedOnly"]),
         "modelCalls": sum(x["modelCalls"] for x in ds), "retries": sum(x["retries"] for x in ds),
         "callsPerDraw": dict(sorted(collections.Counter(str(x["modelCalls"]) for x in ds).items())),
         "distinctDecisions": len({x["finalActionsKey"] or "none:" + x["outcome"] for x in ds}),
         "actionContent": dict(sorted(collections.Counter(x["finalActionsKey"] or "none" for x in ds).items()))}
    if arm == "B":
        c = collections.Counter(x["class"] for x in ds)
        s["classes"] = {k: {"count": c.get(k, 0), "wilson95": wilson(c.get(k, 0), n)} for k in CLASSES}
        s["accurateBeforeChange"] = sum(1 for x in ds if x.get("accurateBeforeChange"))
        s["acknowledgmentFailures"] = dict(sorted(collections.Counter(
            f for x in ds for f in x["acknowledgment"]["failureCodes"]).items()))
        s["withheld"] = dict(sorted(collections.Counter(x["acknowledgment"]["withheld"] for x in ds
                                                        if x["acknowledgment"]["withheld"]).items()))
        s["dReasons"] = dict(sorted(collections.Counter(x["dReason"] for x in ds if x["dReason"]).items()))
    else:
        c = collections.Counter(x["preview"]["class"] for x in ds)
        s["previewClasses"] = {k: c.get(k, 0) for k in PREVIEW_CLASSES}
    return s


def summarize(proto, contexts, draws, complete):
    """Per point and arm; B − A per point only for a complete run; then the
    per-point differences per model and category (no interval)."""
    out = {"points": [], "pooled": {} if complete else None}
    for e in contexts:
        n = proto["drawsPerArm"][e["model"]]
        row = {"id": e["id"], "model": e["model"], "category": e["stageB"]["category"],
               "arms": {a: arm_summary([x for x in draws if x["point"] == e["id"] and x["arm"] == a], n, a)
                        for a in ARMS}}
        if complete:
            A, B = row["arms"]["A"], row["arms"]["B"]
            row["BminusA"] = {
                "outcomes": {o: X.newcombe(B["outcomes"][o]["count"], n, A["outcomes"][o]["count"], n) for o in OUTCOMES},
                "submitted": X.newcombe(B["submitted"], n, A["submitted"], n),
                "executed": X.newcombe(B["executed"], n, A["executed"], n),
                "sameActionContent": A["actionContent"] == B["actionContent"]}
        else:
            row["BminusA"] = None
        out["points"].append(row)
    if complete:
        groups = collections.defaultdict(list)
        for r in out["points"]:
            groups[r["model"] + "/all"].append(r)
            groups[r["model"] + "/" + r["category"]].append(r)
        for k, rows in sorted(groups.items()):
            diffs = {o: [r["BminusA"]["outcomes"][o]["difference"] for r in rows] for o in OUTCOMES}
            out["pooled"][k] = {"points": len(rows), "interval": "none: the points are clusters",
                                "perPointDifferences": diffs,
                                "unweightedMean": {o: round(math.fsum(v) / len(v), 4) for o, v in diffs.items()},
                                "pointsSameActionContent": sum(1 for r in rows if r["BminusA"]["sameActionContent"])}
    return out


NOTES = [
    "A draw is one whole workflow from the recorded proposal: the agent's own ask(), the recorded proposal as its "
    "first reply, then whatever model calls its arm makes. Model calls are counted per draw and differ by arm.",
    "Per point and arm the denominator is every scheduled draw for a COMPLETE run. A PARTIAL run reports only what "
    "was observed, with the observed and scheduled counts; it has no B − A comparison and no pooled summary.",
    "Wilson 95% and Newcombe 95% intervals are descriptive: draws repeat one workflow at one state.",
    "Arm B's classes: A proceeded, B revised, C abandoned, D withheld (after a shown acknowledgment request), E no "
    "request. A revision is a change of decision, not a better one; no reasoning is inferred.",
    "Financial evidence v1, unchanged, and the engine's previews at the recorded state: reported as the engine "
    "reports them, never attributed to later outcomes.",
]


def analyze(binary, d, replays, proto_path):
    proto, proto_sha, man, contexts = load_protocol(proto_path)
    with open(os.path.join(d, "live", "run.json"), encoding="utf-8") as f:
        run_doc = json.load(f)
    if run_doc["protocolSha256"] != proto_sha or run_doc["contextsFileSha256"] != man["contextsFileSha256"]:
        raise ExperimentError("the run is not of this protocol and these points")
    rows = load_jsonl(os.path.join(d, "live", "draws.jsonl"))
    sched = schedule(proto, man, contexts)
    check_rows(sched, rows)
    completed = [r for r in rows if r["status"] == "completed"]
    if run_doc["drawsCompleted"] != len(completed) or run_doc["status"] not in ("COMPLETE", "PARTIAL"):
        raise ExperimentError("run.json does not describe the recorded draws")
    check_run_doc(run_doc, rows, sched)
    complete = run_doc["status"] == "COMPLETE" and len(completed) == len(sched)
    if run_doc["status"] == "COMPLETE" and not complete:
        raise ExperimentError("run.json says COMPLETE, but not every scheduled draw was completed")
    pts = Points(proto, contexts, replays, {SETTINGS_ENV: REPLAY_URL})
    opener = pts.opener(binary)
    checked = {"draws": 0, "modelCalls": 0, "engineRequests": 0}
    draws = []
    try:
        for r in rows:
            e = pts.by_id[r["point"]]
            replay_draw(r, e, pts.settings[r["model"]], opener, FixedMemory(pts.mem[e["id"]]))
            checked["draws"] += 1
            checked["modelCalls"] += len(r["calls"])
            checked["engineRequests"] += len(r["engine"])
            if r["status"] == "completed":
                draws.append(classify(r, e))
    finally:
        opener.close()
    points = []
    for e in contexts:
        lists = {canon(e["proposal"]["actions"]): e["proposal"]["actions"]}
        for x in draws:
            if x["point"] == e["id"] and x["finalActions"]:
                lists.setdefault(canon(x["finalActions"]), x["finalActions"])
        decs = [dict(continuation(binary, pts.run_dir[e["id"]], e, v), isProposal=k == canon(e["proposal"]["actions"]))
                for k, v in sorted(lists.items())]
        by_key = {x["actionsKey"]: x for x in decs}
        for x in draws:
            if x["point"] == e["id"]:
                c = by_key.get(x["finalActionsKey"])
                x["executed"] = c["executed"] if (c and x["submitted"]) else False
        points.append({"id": e["id"], "model": e["model"], "category": e["stageB"]["category"],
                       "validation": e["validation"], "recording": e["recording"], "point": e["point"],
                       "proposalSha256": e["proposal"]["sha256"], "proposal": e["proposal"]["actions"],
                       "decisions": decs})
    return {"schema": ANALYSIS_SCHEMA, "analyzer": ANALYZER, "driver": DRIVER, "protocol": proto["id"],
            "protocolSha256": proto_sha, "contextsFileSha256": man["contextsFileSha256"], "status": run_doc["status"],
            "drawsScheduled": len(sched), "drawsCompleted": len(completed),
            "drawInterrupted": run_doc["drawInterrupted"], "stopped": run_doc["stopped"],
            "coverage": f"{len(completed)} of {len(sched)}", "replayChecked": checked, "points": points,
            "draws": draws, "summary": summarize(proto, contexts, draws, complete), "notes": NOTES}


def inference(d):
    """The live metadata, apart from the deterministic analysis: per point
    and arm, calls, statuses, finish reasons, tokens and latency."""
    rows = load_jsonl(os.path.join(d, "live", "draws.jsonl"))
    g = collections.defaultdict(list)
    for r in rows:
        g[(r["point"], r["arm"])].extend(r["calls"])
    out = {}
    for (p, a), cs in sorted(g.items()):
        lat = sorted(c["latencyMs"] for c in cs)
        u = [c.get("usage") or {} for c in cs]
        out[f"{p}/{a}"] = {"calls": len(cs), "status200": sum(1 for c in cs if c.get("status") == 200),
                           "errors": dict(sorted(collections.Counter((c["error"] or "").split(":", 1)[0]
                                                                     for c in cs if c["error"]).items())),
                           "finishReasons": dict(sorted(collections.Counter(str(c.get("finishReason")) for c in cs).items())),
                           "promptTokens": sum(x.get("prompt_tokens") or 0 for x in u),
                           "completionTokens": sum(x.get("completion_tokens") or 0 for x in u),
                           "cachedTokensNonzero": sum(1 for x in u if (x.get("prompt_tokens_details") or {}).get("cached_tokens")),
                           "latencyMs": {"min": lat[0], "median": lat[len(lat) // 2], "max": lat[-1]} if lat else None}
    return out


def tables(doc):
    s = doc["summary"]
    L = [f"# V0.12b Stage B: A llm-agent 5 (validation-first-preview), B llm-agent 6 (acknowledgment)", "",
         f"**Run status: {doc['status']}** — {doc['coverage']} scheduled draws completed"
         + (f"; stopped: {doc['stopped']['code']}: {doc['stopped']['message']}" if doc["stopped"] else "") + ".", ""]
    if doc["status"] != "COMPLETE":
        L += ["A PARTIAL run: the counts below are what was observed, out of the observed draws. There is no B − A "
              "comparison and no pooled summary, and no conclusion about the complete experiment.", ""]
    L += ["## Per point and arm", "",
          "| Point | Category | Arm | Observed / scheduled | Kept | Revised | Abandoned | Refused | Withheld | Invalid | "
          "Submitted | Executed | Model calls | Retries |", "|---|---|---|---|---|---|---|---|---|---|---|---|---|---|"]
    for r in s["points"]:
        for a in ARMS:
            x = r["arms"][a]
            o = x["outcomes"]
            L.append(f"| {r['id']} | {r['category']} | {a} | {x['observed']} / {x['scheduled']} | " +
                     " | ".join(str(o[k]["count"]) for k in OUTCOMES) +
                     f" | {x['submitted']} | {x['executed']} | {x['modelCalls']} | {x['retries']} |")
    L += ["", "## Arm B: acknowledgment classes", "",
          "| Point | A proceeded | B revised | C abandoned | D withheld | E not required | Accurate before a change | "
          "Reordered only | Failure codes | Why D |", "|---|---|---|---|---|---|---|---|---|---|"]
    for r in s["points"]:
        x = r["arms"]["B"]
        L.append(f"| {r['id']} | " + " | ".join(str(x["classes"][k]["count"]) for k in CLASSES) +
                 f" | {x['accurateBeforeChange']} | {x['reorderedOnly']} | "
                 f"{', '.join(f'{k} {v}' for k, v in x['acknowledgmentFailures'].items()) or '—'}"
                 f" | {', '.join(f'{k} {v}' for k, v in x['dReasons'].items()) or '—'} |")
    L += ["", "## Arm A: the selective preview", "",
          "| Point | Proceeded | Revised | Abandoned | Failed | Not shown | Reordered only |",
          "|---|---|---|---|---|---|---|"]
    for r in s["points"]:
        x = r["arms"]["A"]
        L.append(f"| {r['id']} | " + " | ".join(str(x["previewClasses"][k]) for k in PREVIEW_CLASSES) +
                 f" | {x['reorderedOnly']} |")
    if doc["status"] == "COMPLETE":
        L += ["", "## B − A per point (Newcombe 95%)", "",
              "| Point | Kept | Revised | Abandoned | Refused | Withheld | Invalid | Submitted | Executed | Same action content |",
              "|---|---|---|---|---|---|---|---|---|---|"]
        for r in s["points"]:
            c = r["BminusA"]
            L.append(f"| {r['id']} | " + " | ".join(F.pp(c["outcomes"][o]) for o in OUTCOMES) +
                     f" | {F.pp(c['submitted'])} | {F.pp(c['executed'])} | {'yes' if c['sameActionContent'] else 'no'} |")
    L += ["", "## Engine-reported direct consequences of the distinct final decisions", "",
          "| Point | Proposal | Draws A / B | Valid | Executed | Commitment | Credit-supported | Severance | "
          "Retained obligations | Displaced | Capacity | Reversibility |", "|---|---|---|---|---|---|---|---|---|---|---|---|"]
    for p in doc["points"]:
        for dd in p["decisions"]:
            n = [sum(1 for x in doc["draws"] if x["point"] == p["id"] and x["finalActionsKey"] == dd["actionsKey"]
                     and x["arm"] == a) for a in ARMS]
            x = dd["exposure"]
            L.append(f"| {p['id']} | {'yes' if dd['isProposal'] else ''} | {' / '.join(map(str, n))} | {dd['outcome']} | "
                     f"{'yes' if dd['executed'] else 'no'} | {X.money(x['commitmentCents'])} | "
                     f"{'yes' if x['creditSupported'] else 'no'} | "
                     f"{', '.join(X.money(v) for v in x['severanceCents']) or '—'} | "
                     f"{'; '.join(g['label'] + ' ' + X.money(g['cents']) for g in x['retainedObligations']) or '—'} | "
                     f"{x['displacedBookings']} | {', '.join(f'{k} {v}' for k, v in x['capacity'].items()) or '—'} | "
                     f"{', '.join(f'{k} {v}' for k, v in x['reversibility'].items()) or '—'} |")
    return "\n".join(L) + "\n"


# --- main ------------------------------------------------------------------------

def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    s = sub.add_parser("schedule")
    s.add_argument("--protocol", required=True)
    s.add_argument("--out", required=True)
    c = sub.add_parser("check")
    r = sub.add_parser("run")
    c.add_argument("--unlocked-ok", action="store_true",
                   help="exit 0 although the protocol is not locked (only before the lock commit)")
    for q in (c, r):
        q.add_argument("--binary", required=True)
        q.add_argument("--replays", required=True)
        q.add_argument("--protocol", required=True)
    r.add_argument("--out", required=True)
    r.add_argument("--authorization", required=True,
                   help="the owner's authorization of this run, recorded verbatim in run.json")
    a = sub.add_parser("analyze")
    a.add_argument("--binary", required=True)
    a.add_argument("--replays", required=True)
    a.add_argument("--dir", required=True)
    a.add_argument("--protocol")
    a.add_argument("--out", required=True)
    x = ap.parse_args(argv)
    try:
        if x.cmd == "analyze":
            doc = analyze(x.binary, x.dir, x.replays, x.protocol or os.path.join(x.dir, "protocol.json"))
            write_json(os.path.join(x.out, "decisions.json"), doc)
            write_json(os.path.join(x.out, "inference.json"), {"schema": ANALYSIS_SCHEMA + "#inference",
                                                               "groups": inference(x.dir)})
            with open(os.path.join(x.out, "decision-tables.md"), "w", encoding="utf-8") as f:
                f.write(tables(doc))
            print(f"stage B analysis: {doc['status']}, {doc['coverage']} draws; replay checked {doc['replayChecked']}")
            return 0
        proto, proto_sha, man, contexts = load_protocol(x.protocol)
        sched = schedule(proto, man, contexts)
        if x.cmd == "schedule":
            os.makedirs(x.out, exist_ok=True)
            write_json(os.path.join(x.out, "schedule.json"), schedule_doc(proto, proto_sha, man, sched))
            print(f"stage B schedule: {len(sched)} draws, one at a time")
            return 0
        if x.cmd == "check":
            problems = check_lock(proto, x.protocol)
            check_schedule(proto, sched, os.path.join(os.path.dirname(os.path.abspath(x.protocol)), "schedule.json"),
                           proto_sha)
            pts = Points(proto, contexts, x.replays, {SETTINGS_ENV: REPLAY_URL})
            res = check_points_run(x.binary, proto, pts)
            bad = [y for y in res if not y["stateUnchanged"]]
            print(f"stage B check: {len(res)} point-arm starts, each at the recorded state with the agent's first "
                  f"request equal to the recorded request; lock: {'; '.join(problems) or 'holds'}")
            return 1 if bad or (problems and not x.unlocked_ok) else 0
        base = os.environ.get(proto["endpointEnv"])
        if not base:
            raise ExperimentError(f"set {proto['endpointEnv']}")
        doc = run(proto, proto_sha, man, contexts, x.out, x.binary, x.replays, x.authorization,
                  lambda st: A3.ModelClient(st).call, dict(os.environ), base=base, proto_path=x.protocol)
        print(f"stage B run: {doc['status']}; {doc['drawsCompleted']} of {doc['drawsScheduled']} draws completed, "
              f"{doc['modelCalls']} model calls" + (f"; stopped: {doc['stopped']['code']}" if doc["stopped"] else ""))
        return 0 if doc["status"] == "COMPLETE" else 1
    except ERRORS as e:
        print(f"stage B: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
