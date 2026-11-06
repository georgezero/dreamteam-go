#!/usr/bin/env python3
"""The V0.13d experiment driver (dreamteam-v13d-driver/1): the
management-alternatives turn, control against treatment, from one decision
context and one initial proposal (docs/v13d/DRIVER.md).

Each draw starts from a context:
* historical: a V0.13c branch point (the V0.12b pool), opened on the
  recording's replay at the recorded digests, with the recorded request,
  memory and proposal (as V0.12b Stage B);
* constructed: a sensitivity-pair member (V0.13c), a fresh engine at the
  member's state (mini, seed 52, reached with no action), the agent's own
  prompt at that state with a fresh memory, and the scripted proposal.
Then it runs llm_agent_alternatives.ask under the draw's arm. The driver
reimplements no step: validation-first is llm_agent5's, the turn and the
outcome are the engine's (alternatives.turn, alternatives.decide).

    python3 -I -B analysis/alternatives/v13d.py contexts --binary ./dt --check DIR
    python3 -I -B analysis/alternatives/v13d.py protocol --out FILE
    python3 -I -B analysis/alternatives/v13d.py schedule --protocol FILE --out DIR
    python3 -I -B analysis/alternatives/v13d.py check --binary ./dt --replays DIR --protocol FILE
    python3 -I -B analysis/alternatives/v13d.py run ... (refused: the protocol is not locked)
    python3 -I -B analysis/alternatives/v13d.py analyze --binary ./dt --replays DIR --dir DIR --out DIR

No command contacts a model except run, which refuses a protocol that is
not locked or a run without the owner's authorization.
"""

import argparse
import collections
import json
import math
import os
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, os.path.join(ROOT, "analysis", "acknowledgment"))
sys.path.insert(0, os.path.join(ROOT, "agents", "llm"))
import stage_b as SB  # noqa: E402  (V0.12b's harness, limits and model wrapper, unchanged)
import llm_agent_alternatives as ALT  # noqa: E402

P, X, A3, A5 = SB.P, SB.X, SB.A3, SB.A5
if ALT.A5 is not A5 or ALT.A3 is not A3:
    raise ImportError("llm_agent_alternatives loaded another copy of llm_agent5 or llm_agent3")
sha256, canon, write_json, Halt = SB.sha256, SB.canon, SB.write_json, SB.Halt

DRIVER = "dreamteam-v13d-driver/1"
CONTEXTS_SCHEMA = "dreamteam.v13d-contexts/v1"
PROTOCOL_SCHEMA = "dreamteam.alternatives-experiment/v1"
SCHEDULE_SCHEMA = "dreamteam.alternatives-schedule/v1"
DRAW_SCHEMA = "dreamteam.alternatives-draw/v1"
ORDER_RULE = "dreamteam-v13d-order/1"
ARMS = ALT.ARMS
DRAFT = "DRAFT — NOT LOCKED — NOT AUTHORIZED"
LOCKED = "LOCKED — NOT RUN"
V13C = "docs/benchmark/v13c-dev"
V12B = "docs/benchmark/v12b-dev"
SCRIPTED_REASON = "Constructed context: this proposal was scripted for the experiment."
GEMMA, QWEN = P.GEMMA, P.QWEN


class ExperimentError(Exception):
    pass


def load(p):
    with open(os.path.join(ROOT, p), encoding="utf-8") as f:
        return json.load(f)


def file_sha256(p):
    with open(os.path.join(ROOT, p), "rb") as f:
        return sha256(f.read())


# --- the contexts ------------------------------------------------------------

def historical():
    """The V0.13c selection, each with its committed context (the V0.12b
    pool's committed source line) and V0.13c's turn expectation."""
    man, inv = load(f"{V13C}/points/manifest.json"), load(f"{V13C}/points/inventory.json")
    pool = {r["id"]: r for r in load(f"{V12B}/points/manifest.json")["pool"]}
    pts = {x["id"]: x for x in inv["points"]}
    src = P.committed_contexts()
    out = []
    for pid in man["selected"]:
        r = pool[pid]
        e, s, line_sha = src[r["contextKey"]]
        if line_sha != r["sourceLineSha256"]:
            raise ExperimentError(f"{pid}: the committed context is not the pool's")
        x = pts[pid]
        out.append({"id": pid, "kind": "historical", "model": r["model"], "category": x["category"],
                    "recording": e["recording"], "point": e["point"], "digests": e["digests"],
                    "contextKey": r["contextKey"], "source": s, "sourceLineSha256": line_sha,
                    "requestSha256": sha256(X.body_bytes(e["request"])),
                    "proposal": {"content": e["proposal"]["content"], "sha256": e["proposal"]["sha256"],
                                 "actions": e["proposal"]["actions"]},
                    "expected": {"stepDue": x["stepDue"], "stepWhy": x["stepWhy"]}})
    return out


def constructed():
    """The V0.13c sensitivity-pair members, each a context of its own."""
    out = []
    for p in load(f"{V13C}/sensitivity-pairs.json")["pairs"]:
        for side in ("a", "b"):
            m = p[side]
            content = json.dumps({"reason": SCRIPTED_REASON, "actions": m["proposal"]})
            out.append({"id": f"{p['id']}-{side}", "kind": "constructed", "pair": p["id"], "side": side,
                        "label": m["label"], "scenario": m["scenario"], "seed": m["seed"], "at": m["at"],
                        "proposal": {"content": content, "sha256": sha256(content), "actions": m["proposal"]},
                        "expected": {"stepDue": m["due"]}})
    return out


def spawn(binary, args):
    """A fresh engine with the alternatives ops: (ask, close)."""
    scratch = tempfile.TemporaryDirectory(prefix="v13d-engine-")
    proc = subprocess.Popen([os.path.abspath(binary), "agent", "--stdio", "-alternatives"] + args,
                            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True,
                            encoding="utf-8", cwd=scratch.name)

    def ask(r):
        proc.stdin.write(json.dumps(r) + "\n")
        proc.stdin.flush()
        line = proc.stdout.readline()
        if not line:
            raise ExperimentError("the engine closed the session")
        return json.loads(line)

    def close():
        try:
            proc.stdin.close()
        finally:
            proc.wait()
            scratch.cleanup()
    return ask, close


class Fresh:
    """A constructed context's engine: a fresh session advanced, with no
    action, to the member's time."""

    def __init__(self, binary, c):
        self.ask, self._close = spawn(binary, ["-scenario", c["scenario"], "-seed", str(c["seed"])])
        a = self.request("adv", "advance", {"to": c["at"]})
        if not a.get("ok") or a["result"]["clock"]["time"] != c["at"]:
            self.close()
            raise ExperimentError(f"{c['id']}: the engine did not reach {c['at']}")

    def request(self, rid, op, args=None):
        r = {"id": rid, "op": op}
        if args is not None:
            r["args"] = args
        return self.ask(r)

    def close(self):
        self._close()


def open_context(binary, run_dirs, c):
    if c["kind"] == "constructed":
        return Fresh(binary, c)
    run, pairs = X.transcript(run_dirs[c["id"]])
    ask, close = spawn(binary, ["-scenario", run["scenario"]["ref"], "-seed", str(run["seed"])])
    ses = X.Session(run, pairs, ask, close)
    try:
        ses.to_point(c["digests"])
    except BaseException:
        ses.close()
        raise
    return ses


def prompt_at(binary, c, settings):
    """A constructed context's prompt, made by the agent's own prepare() at
    the member's state with a fresh memory; and the state's digests."""
    ses = Fresh(binary, c)
    try:
        h = SB.EngineHarness(ses)
        d = h.request("digests")["digests"]
        pset = A3.prompt_settings(settings)
        mem = fresh_memory(settings)
        _, _, _, _, prompt, _ = A3.prepare(h, mem, 1, pset)
        return prompt, d
    finally:
        ses.close()


def fresh_memory(settings):
    return A3.Memory(settings["memoryDecisions"], settings["memoryMetricPoints"], settings["memoryMaxChars"])


def build_contexts(binary, settings):
    """The contexts file: historical, then constructed (each constructed
    member with its prompt's SHA-256 and its state's digests)."""
    rows = historical()
    for c in constructed():
        prompt, d = prompt_at(binary, c, settings)
        c["promptSha256"], c["prompt"], c["digests"] = sha256(prompt), prompt, d
        rows.append(c)
    body = "".join(json.dumps(r, sort_keys=True, ensure_ascii=False) + "\n" for r in rows).encode("utf-8")
    return rows, body


def load_contexts(d):
    with open(os.path.join(d, "contexts.jsonl"), "rb") as f:
        body = f.read()
    return [json.loads(l) for l in body.decode("utf-8").splitlines() if l.strip()], sha256(body)


def first_request(c, settings):
    """The request the initial proposal answers: the recorded one
    (historical), or the agent's own payload of its prompt (constructed)."""
    if c["kind"] == "historical":
        e = P.committed_contexts()[c["contextKey"]][0]
        return X.body_bytes(e["request"])
    msgs = [{"role": "system", "content": settings["systemPrompt"]}, {"role": "user", "content": c["prompt"]}]
    return X.body_bytes(A3.ModelClient(settings).payload(msgs))


# --- the model ----------------------------------------------------------------

class ContextModel(SB.PointModel):
    """V0.12b's PointModel at a context: the first request must be the
    context's, byte for byte; its reply is the initial proposal; every later
    request goes to the transport after the limits, and is recorded."""

    def __init__(self, c, want, client, transport, limits, utc=SB.RR.utc, now=SB.time.monotonic):
        super().__init__({"id": c["id"], "proposal": c["proposal"]}, client, transport, limits, utc=utc, now=now)
        self.want = want

    def call(self, payload):
        if self.first:
            self.first = False
            if json.dumps(payload).encode("utf-8") != self.want:
                raise ExperimentError(f"{self.e['id']}: the agent's first request is not the context's request")
            return 200, json.dumps({"choices": [{"message": {"content": self.e["proposal"]["content"]}}]}), None
        return SB.PointModel.call(self, payload)


# --- one draw -------------------------------------------------------------------

DETERMINISTIC = ("schema", "seq", "model", "context", "arm", "draw", "status", "halt", "initialDigests", "record",
                 "batches", "engine", "digestsAfter")


def run_draw(entry, c, settings, opener, mem, transport, limits, utc=SB.RR.utc, now=SB.time.monotonic):
    arm = entry["arm"]
    if arm not in ARMS:
        raise ExperimentError(f"draw {entry['seq']}: arm {arm!r}")
    out = {"schema": DRAW_SCHEMA, **{k: entry[k] for k in ("seq", "model", "context", "arm", "draw")},
           "status": None, "halt": None, "initialDigests": None, "record": None, "batches": None, "engine": [],
           "digestsAfter": None, "calls": []}
    ses = opener(c, arm)
    h = SB.EngineHarness(ses)
    model = ContextModel(c, first_request(c, settings), A3.ModelClient(settings), transport, limits, utc=utc, now=now)
    try:
        d = h.request("digests")["digests"]
        if d != c["digests"]:
            raise ExperimentError(f"draw {entry['seq']}: the engine is not at the context's state")
        out["initialDigests"] = d
        h.log = []
        try:
            batches, rec = ALT.ask(h, SB.Sink(), model, settings, mem, c.get("point", 1), None, arm=arm)
        except Halt as x:
            out.update(status="interrupted", halt={"code": x.code, "message": x.message})
        else:
            if model.first:
                raise ExperimentError(f"draw {entry['seq']}: the agent made no request")
            out.update(status="completed", record=rec, batches=batches)
        out["engine"] = h.log
        h.log = []
        out["digestsAfter"] = h.request("digests")["digests"]
    except BaseException as x:
        out["status"] = "aborted"
        out["engine"] = out["engine"] or h.log
        x.partialDraw = out
        raise
    finally:
        out["calls"] = model.calls
        ses.close()
    return out


def deterministic(draw):
    out = {k: draw[k] for k in DETERMINISTIC}
    out["calls"] = [{k: x[k] for k in ("n", "requestSha256", "status", "error", "responseSha256")} for x in draw["calls"]]
    return out


class Contexts:
    """Per context: its replay run (historical) and memory; per model:
    settings."""

    def __init__(self, proto, rows, replays, env):
        self.by_id = {c["id"]: c for c in rows}
        self.settings = {m: SB.settings_of(proto, m, env) for m in proto["modelOrder"]}
        self.run_dirs, self.mem = {}, {}
        runs = X.replay_runs(replays) if replays else {}
        for c in rows:
            if c["kind"] == "historical":
                name = os.path.basename(c["recording"])
                if name not in runs:
                    raise ExperimentError(f"{c['id']}: no replay of {name} in {replays}")
                self.run_dirs[c["id"]] = runs[name]
                self.mem[c["id"]] = SB.memory_text(runs[name], {"id": c["id"], "recording": c["recording"],
                                                                "point": c["point"]})

    def memory(self, c, model):
        if c["kind"] == "historical":
            return SB.FixedMemory(self.mem[c["id"]])
        return fresh_memory(self.settings[model])

    def opener(self, binary):
        return Pool(lambda c, arm: open_context(binary, self.run_dirs, c))


class Pool:
    """One engine session per context and arm, opened at its first draw and
    kept for that arm's later draws (V0.12b's Pool, for contexts that both
    models use): the arms never share an engine process, and every request
    a draw makes there is read-only (validate, preview, observe, actions,
    digests, alternatives.turn, alternatives.decide). Every draw checks the
    state at its start and records it at its end."""

    def __init__(self, open_fn):
        self.open_fn, self.sessions = open_fn, {}

    def __call__(self, c, arm):
        k = (c["id"], arm)
        if k not in self.sessions:
            self.sessions[k] = self.open_fn(c, arm)
        return SB.Shared(self.sessions[k])

    def close(self):
        for k in list(self.sessions):
            self.sessions.pop(k).close()


def scripted(replies):
    """A transport that answers each request with the next scripted
    content (a string), or a transport failure (None)."""
    it = iter(replies)

    def transport(payload):
        r = next(it, "<<end>>")
        if r == "<<end>>":
            raise ExperimentError("the scripted model has no more replies")
        if r is None:
            return None, "", "timeout: the scripted model did not answer"
        return 200, json.dumps({"choices": [{"message": {"content": r}}], "model": "scripted"}), None
    return transport


def replay_draw(r, c, settings, opener, mem):
    """A recorded draw again, without a model: everything but the live
    metadata must be the recorded draw's, and every response used."""
    left = list(r["calls"])

    def transport(payload):
        if not left:
            raise ExperimentError(f"draw {r['seq']}: a request the run did not record")
        x = left.pop(0)
        return SB.replay_transport({"id": c["id"]}, [x])(payload)
    limits = None
    if r["status"] == "interrupted":
        n, halt = len(r["calls"]), r["halt"]

        class At(SB.Limits):
            def before_call(self, draw_calls):
                if draw_calls >= n:
                    raise Halt(halt["code"], halt["message"])
        limits = At(0, 0, 0, 0)
    entry = {k: r[k] for k in ("seq", "model", "context", "arm", "draw")}
    again = run_draw(entry, c, settings, opener, mem, transport, limits, utc=lambda: None, now=lambda: 0.0)
    if left:
        raise ExperimentError(f"draw {r['seq']}: recorded responses the workflow did not use")
    if deterministic(again) != deterministic(r):
        raise ExperimentError(f"draw {r['seq']}: the replay differs from the record")
    return again


# --- execution ------------------------------------------------------------------

def execute(binary, run_dirs, c, batches):
    """The batches applied, in order, in a fresh engine at the context, with
    the engine's own apply; each batch's answer and the digests after, then
    the end of the day reached with no further action (the continuation).
    Partial application is recorded as it happens; nothing is undone."""
    ses = open_context(binary, run_dirs, c)
    try:
        h = SB.EngineHarness(ses)
        out = {"batches": [], "partial": False}
        for b in batches:
            try:
                a = h.request("apply", {"actions": b})
                out["batches"].append({"applied": bool(a.get("applied")), "results": a.get("results")})
            except A3.ProtocolError as e:
                out["batches"].append({"applied": False, "error": {"code": e.code, "message": e.message}})
        n = sum(1 for b in out["batches"] if b["applied"])
        out["executed"] = bool(batches) and n == len(batches)
        out["partial"] = 0 < n < len(batches)
        out["digestsAfterApply"] = h.request("digests")["digests"]
        h.request("advance", {"next": "end_of_day"})
        out["digestsAfterDay"] = h.request("digests")["digests"]
        return out
    finally:
        ses.close()


# --- classification ---------------------------------------------------------------

RESULTS = ("retained", "selected", "selected_no_action", "revised", "abandoned", "invalid_alternative", "no_decision")
NOT_DUE = "not_due"


def classify(d):
    """A completed draw's observable outcome (docs/v13d/OUTCOMES.md)."""
    rec = d["record"]
    turn = rec.get("turn") or {}
    exch = rec.get("exchanges") or []
    o = rec.get("outcome") or {}
    result = rec["result"] if rec["result"] in RESULTS else NOT_DUE
    flags = o.get("flags") or []
    return {"result": result, "why": rec["result"], "due": bool(turn.get("due")),
            "comparisonPresented": bool(turn.get("due")) and d["arm"] == "treatment",
            "inspected": "inspected" in flags, "compared": "compared" in flags,
            "noObservableComparison": not ({"inspected", "compared"} & set(flags)),
            "finalValid": bool(o.get("valid")) if result in RESULTS else None,
            "submitted": bool(d["batches"]), "modelCalls": len(d["calls"]),
            "retries": sum(1 for x in exch if x["kind"] in ("unreadable", "inspect_again", "transport")),
            "batchesSha256": sha256(canon(d["batches"]))}


# --- the protocol and the schedule -------------------------------------------------

def schedule(proto, rows):
    """Per model in modelOrder, its draws as rounds; each round has every
    unit (context, arm) of the model once, in the order of
    SHA-256('dreamteam-v13d-order/1|<model>|<round>|<context>|<arm>')."""
    out = []
    for mod in proto["modelOrder"]:
        units = [c["id"] for c in rows if c["kind"] == "constructed" or c["model"] == mod]
        n = proto["drawsPerUnit"][mod]
        for rnd in range(1, max(n.values()) + 1):
            these = [(cid, a) for cid in units for a in ARMS
                     if rnd <= n["aa" if proto["contexts"][cid]["aa"] else "unit"]]
            for cid, a in sorted(these, key=lambda x: sha256(f"{ORDER_RULE}|{mod}|{rnd}|{x[0]}|{x[1]}")):
                out.append({"seq": len(out) + 1, "model": mod, "context": cid, "arm": a, "draw": rnd})
    return out


def call_bound(c):
    """The most model calls a draw can make after its initial proposal: the
    validation-first correction and its retry when the proposal is refused
    (llm_agent5, maxRetries 1), then the turn's reply, one inspect answer and
    one retry."""
    refused = c.get("category", "").startswith("refused")
    return (2 if refused else 0) + 1 + ALT.INSPECT_ROUNDS + ALT.TURN_RETRIES


def draft_protocol(rows, contexts_sha):
    """The revised draft protocol (docs/v13d/PROTOCOL_DRAFT.md), made from
    the committed inputs only."""
    contexts = {}
    for c in rows:
        aa = c["kind"] == "historical" and not c["expected"]["stepDue"] and not c["category"].startswith("refused")
        contexts[c["id"]] = {"kind": c["kind"], "aa": aa, "model": c.get("model"), "callBound": call_bound(c)}
    proto = {"schema": PROTOCOL_SCHEMA, "id": "v13d-alt-1", "status": DRAFT, "driver": DRIVER,
             "workflow": {"version": ALT.WORKFLOW_VERSION, "sha256": ALT.WORKFLOW_SHA256,
                          "base": A5.VALIDATION_FIRST_VERSION, "baseSha256": A5.VALIDATION_FIRST_SHA256},
             "contextsFileSha256": contexts_sha, "contexts": contexts,
             "models": load(f"{V12B}/protocol.json")["models"], "modelOrder": [GEMMA, QWEN],
             "settings": {"files": settings_files(), "maxRetries": 1, "allowDetails": True},
             "systemPromptSha256": load(f"{V12B}/protocol.json")["systemPromptSha256"],
             "request": {"timeoutSeconds": 300}, "endpointEnv": "DREAMTEAM_LLM_BASE_URL",
             "drawsPerUnit": {GEMMA: {"unit": 3, "aa": 2}, QWEN: {"unit": 5, "aa": 2}},
             "orderRule": ORDER_RULE,
             "texts": {"presentation": {"id": "dreamteam.alternatives-presentation/v2",
                                        "file": "internal/alternatives/text/presentation-v2.tmpl",
                                        "sha256": file_sha256("internal/alternatives/text/presentation-v2.tmpl")},
                       "viewTable": {"id": "dreamteam.alternatives-view/v1",
                                     "file": "docs/benchmark/v13d-dev/presentation/view-table.txt",
                                     "sha256": file_sha256("docs/benchmark/v13d-dev/presentation/view-table.txt")},
                       "selectionRule": {"id": "dreamteam.alternative-selection/v2",
                                         "file": "internal/alternatives/text/selection-v2.txt",
                                         "sha256": file_sha256("internal/alternatives/text/selection-v2.txt")},
                       "arms": "dreamteam.alternatives-arms/v2"},
             "requests": requests_of(rows, settings_files()),
             "arms": {"control": "validation-first, then the shared head and foot only",
                      "treatment": "validation-first, then the shared head, the verified comparison (one inspect "
                                   "round) and the shared foot"},
             "analysis": {"estimand": "per context and model: the difference between the arms' distributions of the "
                                      "immediate final action (retained, selected, selected_no_action, revised, "
                                      "abandoned, invalid_alternative, no_decision, not_due), each category's "
                                      "difference with a Newcombe interval, and the total variation distance",
                          "pairs": "secondary, per sensitivity pair and model: departure (anything but retained) "
                                   "in member b minus member a, per arm (Newcombe), and treatment minus control "
                                   "(MOVER); a context pair, not one fact",
                          "observable": "comparison presented, inspected, compared (verified citations), no "
                                        "observable comparison, final valid, submitted, executed, partial, calls, "
                                        "retries",
                          "notClaimed": "a changed decision is not an improvement; the reason is never read"}}
    sched = schedule(proto, rows)
    proto["draws"] = len(sched)
    proto["maxModelCallsPerDraw"] = 6
    proto["maxModelCalls"] = sum(proto["contexts"][s["context"]]["callBound"] for s in sched)
    proto["stop"] = {"wallClockSeconds": 4 * 3600, "consecutiveTransportFailures": 5,
                     "health": "an unhealthy /health before a round stops the run",
                     "serverChange": "a material change before a round stops the run",
                     "partial": "a run that stops early is PARTIAL; the analysis reports only what was observed",
                     "resume": "never"}
    return proto, sched


# --- the run ------------------------------------------------------------------------

RUN_SCHEMA = "dreamteam.alternatives-run/v1"


def run_schedule(proto, proto_sha, sched, ctxs, opener, transports, out, authorization, state_fn=None,
                 now=SB.time.monotonic, utc=SB.RR.utc):
    """The schedule's draws in order, one at a time, each recorded as it
    ends; stopped by the first stop condition (V0.12b's, unchanged in
    meaning). Never resumed: a directory with a run is refused. The live
    command and the mock run both come here; only the transports differ."""
    live = os.path.join(out, "live")
    if os.path.exists(live):
        raise ExperimentError(f"{live} exists: a run is never resumed, extended or overwritten")
    if not authorization or not authorization.strip():
        raise ExperimentError("no authorization")
    os.makedirs(live)
    stop = proto["stop"]
    limits = SB.Limits(proto["maxModelCallsPerDraw"], proto["maxModelCalls"], stop["wallClockSeconds"],
                       proto["request"]["timeoutSeconds"], now=now)
    started, completed, stopped, interrupted, aborted, fails, prev = utc(), 0, None, None, None, 0, None
    sig = None
    if state_fn is not None:
        before = state_fn()
        write_json(os.path.join(live, "server-before.json"), before)
        sig = X.server_signature(before)
        if (before.get("health") or {}).get("status") != "healthy":
            stopped = {"code": "health", "message": "the server did not answer healthy before the run"}
    try:
        with open(os.path.join(live, "draws.jsonl"), "w", encoding="utf-8") as f:
            for r in sched if stopped is None else []:
                if state_fn is not None and (r["model"], r["draw"]) != prev:
                    prev = (r["model"], r["draw"])
                    st = state_fn()
                    if (st.get("health") or {}).get("status") != "healthy":
                        stopped = {"code": "health", "message": f"unhealthy before draw {r['seq']}"}
                        break
                    if X.server_signature(st) != sig:
                        stopped = {"code": "server_change", "message": f"server changed before draw {r['seq']}"}
                        break
                try:
                    limits.before_draw()
                except Halt as x:
                    stopped = {"code": x.code, "message": x.message}
                    break
                c = ctxs.by_id[r["context"]]
                try:
                    d = run_draw(r, c, ctxs.settings[r["model"]], opener, ctxs.memory(c, r["model"]),
                                 transports[r["model"]], limits, utc=utc, now=now)
                except BaseException as x:
                    aborted = r["seq"]
                    if getattr(x, "partialDraw", None) is not None:
                        write_json(os.path.join(live, "aborted-draw.json"), x.partialDraw)
                    stopped = {"code": "operator" if isinstance(x, KeyboardInterrupt) else "driver",
                               "message": f"draw {r['seq']}: {type(x).__name__}: {x}"}
                    if not isinstance(x, Exception) and not isinstance(x, KeyboardInterrupt):
                        raise
                    break
                f.write(json.dumps(d, sort_keys=True, ensure_ascii=False) + "\n")
                f.flush()
                if d["status"] != "completed":
                    interrupted, stopped = r["seq"], dict(d["halt"])
                    break
                completed += 1
                for x in d["calls"]:
                    fails = fails + 1 if x["error"] and x["status"] is None else 0
                if fails >= stop["consecutiveTransportFailures"]:
                    stopped = {"code": "transport", "message": f"{fails} consecutive transport failures"}
                    break
    finally:
        opener.close()
        status = "COMPLETE" if completed == len(sched) and stopped is None else "PARTIAL"
        doc = {"schema": RUN_SCHEMA, "driver": DRIVER, "protocol": proto["id"], "protocolSha256": proto_sha,
               "authorization": authorization, "started": started, "ended": utc(), "status": status,
               "drawsScheduled": len(sched), "drawsCompleted": completed, "drawInterrupted": interrupted,
               "drawAborted": aborted,
               "drawsNotRun": len(sched) - completed - (1 if interrupted else 0) - (1 if aborted else 0),
               "modelCalls": limits.calls, "stopped": stopped, "resume": "never"}
        write_json(os.path.join(live, "run.json"), doc)
    return doc


def check_rows(sched, rows):
    """The recorded draws are the schedule's first draws, in order, each
    once; only the last may be unfinished."""
    if len(rows) > len(sched):
        raise ExperimentError("more draws recorded than scheduled")
    for k, (r, x) in enumerate(zip(rows, sched)):
        if r.get("schema") != DRAW_SCHEMA or {y: r.get(y) for y in ("seq", "model", "context", "arm", "draw")} != x:
            raise ExperimentError(f"draw {k + 1}: not the schedule's draw {x['seq']}")
        if r["status"] != "completed" and k != len(rows) - 1:
            raise ExperimentError(f"draw {r['seq']}: an unfinished draw before the last")
        if r["status"] not in ("completed", "interrupted"):
            raise ExperimentError(f"draw {r['seq']}: status {r['status']!r}")


# --- the analysis -----------------------------------------------------------------

def mover(a, b):
    """MOVER interval for the difference of two differences, from their
    Newcombe intervals (Newcombe 1998; Zou and Donner 2008)."""
    if a is None or b is None:
        return None
    d = a["difference"] - b["difference"]
    (la, ua), (lb, ub) = a["interval"], b["interval"]
    return {"difference": round(d, 4), "interval": [
        round(d - math.sqrt((a["difference"] - la) ** 2 + (ub - b["difference"]) ** 2), 4),
        round(d + math.sqrt((ua - a["difference"]) ** 2 + (b["difference"] - lb) ** 2), 4)]}


FINAL = RESULTS + (NOT_DUE,)


def analyze(binary, run_dirs, ctxs, proto, sched, rows, run_doc):
    """Every recorded draw replayed without a model and classified; the
    batches executed (once per distinct context and batches); the
    distributions per context, model and arm; the arm differences (D9) and,
    for each sensitivity pair, the difference of differences in departure.
    Only what was observed is reported."""
    check_rows(sched, rows)
    opener = ctxs.opener(binary)
    out, cache = [], {}
    try:
        for r in rows:
            c = ctxs.by_id[r["context"]]
            replay_draw(r, c, ctxs.settings[r["model"]], opener, ctxs.memory(c, r["model"]))
            if r["status"] != "completed":
                continue
            x = classify(r)
            k = (c["id"], x["batchesSha256"])
            if k not in cache:
                cache[k] = execute(binary, run_dirs, c, r["batches"]) if r["batches"] else None
            e = cache[k]
            x["executed"] = None if e is None else e["executed"]
            x["partial"] = None if e is None else e["partial"]
            out.append(dict(x, seq=r["seq"], model=r["model"], context=r["context"], arm=r["arm"], draw=r["draw"]))
    finally:
        opener.close()
    cells = collections.defaultdict(list)
    for x in out:
        cells[(x["context"], x["model"], x["arm"])].append(x)
    table = []
    for (cid, mod, arm), xs in sorted(cells.items()):
        n = len(xs)
        cnt = collections.Counter(x["result"] for x in xs)
        table.append({"context": cid, "model": mod, "arm": arm, "observed": n,
                      "results": {k: cnt.get(k, 0) for k in FINAL},
                      "inspected": sum(x["inspected"] for x in xs), "compared": sum(x["compared"] for x in xs),
                      "presented": sum(x["comparisonPresented"] for x in xs),
                      "finalValid": sum(1 for x in xs if x["finalValid"]),
                      "submitted": sum(x["submitted"] for x in xs),
                      "executed": sum(1 for x in xs if x["executed"]), "partial": sum(1 for x in xs if x["partial"]),
                      "modelCalls": sum(x["modelCalls"] for x in xs), "retries": sum(x["retries"] for x in xs)})
    diffs = []
    complete = run_doc["status"] == "COMPLETE"
    by = {(t["context"], t["model"], t["arm"]): t for t in table}
    for (cid, mod, arm) in sorted(by):
        if arm != "treatment" or (cid, mod, "control") not in by:
            continue
        tt, tc = by[(cid, mod, "treatment")], by[(cid, mod, "control")]
        diffs.append({"context": cid, "model": mod,
                      "byResult": {k: X.newcombe(tt["results"][k], tt["observed"], tc["results"][k], tc["observed"])
                                   for k in FINAL},
                      "totalVariation": round(0.5 * sum(abs(tt["results"][k] / tt["observed"] - tc["results"][k] /
                                                            tc["observed"]) for k in FINAL), 4)})
    pairs = []
    for pid in sorted({c["pair"] for c in ctxs.by_id.values() if c["kind"] == "constructed" and c.get("pair")}):
        for mod in proto["modelOrder"]:
            d = {}
            for arm in ARMS:
                a, b = by.get((f"{pid}-a", mod, arm)), by.get((f"{pid}-b", mod, arm))
                if a and b:
                    dep = lambda t: t["observed"] - t["results"]["retained"]
                    d[arm] = X.newcombe(dep(b), b["observed"], dep(a), a["observed"])
            if len(d) == 2:
                pairs.append({"pair": pid, "model": mod, "departure": d,
                              "sensitivity": mover(d["treatment"], d["control"])})
    return {"schema": "dreamteam.alternatives-analysis/v1", "runStatus": run_doc["status"],
            "note": None if complete else "PARTIAL: only what was observed; no pooled summary",
            "draws": out, "table": table, "armDifferences": diffs, "pairs": pairs if complete else []}


def requests_of(rows, files):
    """Per context and model, the SHA-256 of the first request (the one the
    initial proposal answers)."""
    env = {SB.SETTINGS_ENV: SB.REPLAY_URL}
    st = {m: A3.load_settings(os.path.join(ROOT, f["file"]), env) for m, f in files.items()}
    out = {}
    for c in rows:
        mods = [c["model"]] if c["kind"] == "historical" else sorted(st)
        out[c["id"]] = {m: sha256(first_request(c, st[m])) for m in mods}
    return out


# --- commands ---------------------------------------------------------------------

def settings_files():
    f = load(f"{V12B}/protocol.json")["settings"]["files"]
    return {m: {"file": v["file"], "sha256": file_sha256(v["file"])} for m, v in f.items()}


def cmd_contexts(a):
    s = A3.load_settings(os.path.join(ROOT, settings_files()[QWEN]["file"]), {SB.SETTINGS_ENV: SB.REPLAY_URL})
    rows, body = build_contexts(a.binary, s)
    path = os.path.join(a.check or a.out, "contexts.jsonl")
    if a.check:
        with open(path, "rb") as f:
            if f.read() != body:
                raise SystemExit("contexts.jsonl differs from the contexts made again")
        print(f"contexts: {len(rows)} identical")
        return
    os.makedirs(a.out, exist_ok=True)
    with open(path, "wb") as f:
        f.write(body)
    print(f"contexts: {len(rows)} written")


def cmd_protocol(a):
    rows, csha = load_contexts(a.contexts)
    proto, sched = draft_protocol(rows, csha)
    write_json(a.out, proto)
    print(f"protocol {proto['id']}: {proto['draws']} draws, at most {proto['maxModelCalls']} model calls")


def cmd_schedule(a):
    proto, raw = json.load(open(a.protocol, encoding="utf-8")), open(a.protocol, "rb").read()
    rows, csha = load_contexts(a.contexts)
    if csha != proto["contextsFileSha256"]:
        raise SystemExit("the contexts file is not the protocol's")
    again, sched = draft_protocol(rows, csha)
    if {k: v for k, v in again.items() if k != "status"} != {k: v for k, v in proto.items() if k not in ("status", "lock")}:
        raise SystemExit("the protocol is not the one the committed inputs make")
    doc = {"schema": SCHEDULE_SCHEMA, "protocol": proto["id"], "protocolSha256": sha256(raw), "orderRule": ORDER_RULE,
           "draws": len(sched), "schedule": sched}
    write_json(a.out, doc)
    print(f"schedule: {len(sched)} draws")


def cmd_check(a):
    """Every protocol context and arm: a fresh engine at the context's
    state, the first request the context's, the turn due as declared. No
    model is called (a model request halts)."""
    proto = json.load(open(a.protocol, encoding="utf-8"))
    rows, csha = load_contexts(a.contexts)
    if csha != proto["contextsFileSha256"]:
        raise SystemExit("the contexts file is not the protocol's")
    env = {SB.SETTINGS_ENV: SB.REPLAY_URL}
    ctxs = Contexts(proto, rows, a.replays, env)
    pool = ctxs.opener(a.binary)
    bad = 0
    try:
        for c in rows:
            seen = []
            for arm in ARMS:
                mod = c.get("model") or proto["modelOrder"][0]
                entry = {"seq": 0, "model": mod, "context": c["id"], "arm": arm, "draw": 0}
                d = run_draw(entry, c, ctxs.settings[mod], pool, ctxs.memory(c, mod), lambda p: None,
                             SB.Limits(0, 0, 1, 0))
                t = next((e["answer"]["result"] for e in d["engine"] if e["op"] == "alternatives.turn"), None)
                seen.append((d["initialDigests"] == c["digests"], (t or {}).get("due"), (t or {}).get("why")))
            due = bool(seen[0][1])
            ok = seen[0] == seen[1] and seen[0][0] and (c["expected"].get("stepDue") is None or due == c["expected"]["stepDue"])
            bad += not ok
            print(f"{c['id']}: {'ok' if ok else 'FAILED'} (turn {'due' if due else seen[0][2] or 'not reached'})")
    finally:
        pool.close()
    print(f"check: {len(rows)} contexts, {bad} failed")
    if bad:
        raise SystemExit(1)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("schedule")
    p.add_argument("--protocol", required=True)
    p.add_argument("--contexts", required=True)
    p.add_argument("--out", required=True)
    p = sub.add_parser("check")
    p.add_argument("--binary", required=True)
    p.add_argument("--protocol", required=True)
    p.add_argument("--contexts", required=True)
    p.add_argument("--replays")
    p = sub.add_parser("contexts")
    p.add_argument("--binary", required=True)
    p.add_argument("--out")
    p.add_argument("--check")
    p = sub.add_parser("protocol")
    p.add_argument("--contexts", required=True)
    p.add_argument("--out", required=True)
    p = sub.add_parser("run")
    p.add_argument("--protocol", required=True)
    p.add_argument("--authorization", default="")
    a = ap.parse_args(argv)
    if a.cmd == "contexts":
        cmd_contexts(a)
    elif a.cmd == "schedule":
        cmd_schedule(a)
    elif a.cmd == "check":
        cmd_check(a)
    elif a.cmd == "protocol":
        cmd_protocol(a)
    elif a.cmd == "run":
        proto = json.load(open(a.protocol, encoding="utf-8"))
        if proto.get("status") != LOCKED:
            raise SystemExit(f"the protocol's status is {proto.get('status')!r}: only a locked protocol, with the "
                             "owner's authorization, can be run")
        if not a.authorization.strip():
            raise SystemExit("no --authorization: the lock does not authorize a run; the owner does")
        raise SystemExit("run: the live run is not implemented before the owner approves the protocol")


if __name__ == "__main__":
    main()
