#!/usr/bin/env python3
"""The V0.13g experiment driver (dreamteam-v13g-driver/3): the
management-alternatives turn, control against treatment, from one decision
context and one initial proposal (docs/v13g/DRIVER.md).

Each draw starts from a context:
* historical: a V0.13c branch point (the V0.12b pool), opened on the
  recording's replay at the recorded digests, with the recorded request,
  memory and proposal (as V0.12b Stage B);
* constructed: a sensitivity-pair member (V0.13c), a fresh engine at the
  member's state (mini, seed 52, reached with no action), the agent's own
  prompt at that state with a fresh memory, and the scripted proposal.
Then it runs llm_agent_alternatives_v3.ask under the draw's arm, using the
successor validation-first workflow and read-only alternatives.v3.turn,
alternatives.v3.decide and alternatives.v3.canonical engine extensions.

    python3 -I -B analysis/alternatives/v13g.py contexts --binary ./dt --check DIR
    python3 -I -B analysis/alternatives/v13g.py protocol --out FILE
    python3 -I -B analysis/alternatives/v13g.py schedule --protocol FILE --out DIR
    python3 -I -B analysis/alternatives/v13g.py check --binary ./dt --replays DIR --protocol FILE
    python3 -I -B analysis/alternatives/v13g.py run ... (refused: the protocol is not locked)
    python3 -I -B analysis/alternatives/v13g.py analyze --binary ./dt --replays DIR --dir DIR --out DIR

No command contacts a model. The live run implementation is absent; run
refuses execution even if given a nominal lock and authorization.
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
import llm_agent_alternatives_v3 as ALT  # noqa: E402

P, X, A3, A5 = SB.P, SB.X, SB.A3, SB.A5
if ALT.A5 is not A5 or ALT.A3 is not A3:
    raise ImportError("llm_agent_alternatives loaded another copy of llm_agent5 or llm_agent3")
sha256, canon, write_json, Halt = SB.sha256, SB.canon, SB.write_json, SB.Halt

DRIVER = "dreamteam-v13g-driver/3"
CONTEXTS_SCHEMA = "dreamteam.v13g-contexts/v1"
PROTOCOL_SCHEMA = "dreamteam.alternatives-experiment/v3"
SCHEDULE_SCHEMA = "dreamteam.alternatives-schedule/v3"
DRAW_SCHEMA = "dreamteam.alternatives-draw/v3"
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
    scratch = tempfile.TemporaryDirectory(prefix="v13g-engine-")
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
            proc.stdout.close()
            scratch.cleanup()
    return ask, close


class Fresh:
    """A constructed context's engine: a fresh session advanced, with no
    action, to the member's time."""

    def __init__(self, binary, c):
        self.ask, self._close = spawn(binary, ["-scenario", c["scenario"], "-seed", str(c["seed"])])
        try:
            a = self.request("adv", "advance", {"to": c["at"]})
            if not a.get("ok") or a["result"]["clock"]["time"] != c["at"]:
                raise ExperimentError(f"{c['id']}: the engine did not reach {c['at']}")
        except BaseException:
            self.close()
            raise

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
        _, _, _, _, prompt, _ = ALT.prepare(h, mem, 1, pset)
        return prompt, d
    finally:
        ses.close()


def fresh_memory(settings):
    return A3.Memory(settings["memoryDecisions"], settings["memoryMetricPoints"], settings["memoryMaxChars"])


def build_contexts(binary, settings):
    """The contexts file: historical, then constructed (each constructed
    member with its prompt's SHA-256 and its state's digests)."""
    rows = historical()
    for c in rows:
        c["sourceRequestSha256"]=c["requestSha256"]
        c["requestSha256"]=sha256(first_request(c,settings))
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
        payload=json.loads(X.body_bytes(e["request"]))
        payload["messages"][0]["content"]=ALT.SYSTEM_PROMPT
        payload["messages"][-1]["content"]=ALT.INITIAL_TEMPLATE.format(context=payload["messages"][-1]["content"])
        return json.dumps(payload).encode("utf-8")
    msgs = [{"role": "system", "content": ALT.SYSTEM_PROMPT}, {"role": "user", "content": c["prompt"]}]
    return X.body_bytes(A3.ModelClient(settings).payload(msgs))


def settings_of(proto, model, env):
    historical=dict(proto,systemPromptSha256=load(f"{V12B}/protocol.json")["systemPromptSha256"])
    s=SB.settings_of(historical,model,env)
    s["systemPrompt"]=ALT.SYSTEM_PROMPT
    if proto["systemPromptSha256"]!=sha256(ALT.SYSTEM_PROMPT):
        raise ExperimentError("successor system instruction differs")
    return s


# --- limits and the engine harness ---------------------------------------------

class Limits(SB.Limits):
    """V0.12b's limits, checked before every model request, with the
    consecutive transport-failure stop checked there too, call by call
    (review F9), and the evidence of a halt recorded with it (review F6)."""

    def __init__(self, per_draw, total, wall_seconds, timeout, max_fails=None, now=SB.time.monotonic, t0=None):
        super().__init__(per_draw, total, wall_seconds, timeout, now=now, t0=t0)
        self.max_fails, self.fails = max_fails, 0

    def state(self, draw_calls):
        return {"drawCalls": draw_calls, "runCalls": self.calls, "elapsedSeconds": self.elapsed(),
                "consecutiveTransportFailures": self.fails}

    def before_call(self, draw_calls):
        try:
            if self.max_fails is not None and self.fails >= self.max_fails:
                raise Halt("transport", f"{self.fails} consecutive transport failures")
            super().before_call(draw_calls)
        except Halt as x:
            x.evidence = dict(self.state(draw_calls), stage="request_start")
            raise

    def before_draw(self):
        try:
            super().before_draw()
        except Halt as x:
            x.evidence=dict(self.state(0), stage="draw_start")
            raise

    def after_call(self, call):
        self.fails = self.fails + 1 if call["error"] and call["status"] is None else 0


def check_halt(halt,cfg):
    e=halt.get("evidence") or {}
    for key in ("drawCalls","runCalls","consecutiveTransportFailures"):
        if not isinstance(e.get(key),int) or isinstance(e[key],bool) or e[key]<0:
            raise ExperimentError("invalid halt count: "+key)
    elapsed=e.get("elapsedSeconds")
    if isinstance(elapsed,bool) or not isinstance(elapsed,(int,float)) or not math.isfinite(elapsed) or elapsed<0:
        raise ExperimentError("invalid elapsed-time evidence")
    late=elapsed+cfg["request"]["timeoutSeconds"]>cfg["stop"]["wallClockSeconds"]
    if e.get("stage")=="draw_start":
        expected="wall_clock" if e["drawCalls"]==0 and late else None
    elif e.get("stage")=="request_start":
        expected=("transport" if e["consecutiveTransportFailures"]>=cfg["stop"]["consecutiveTransportFailures"] else
                  "draw_call_limit" if e["drawCalls"]>=cfg["maxModelCallsPerDraw"] else
                  "call_budget" if e["runCalls"]>=cfg["maxModelCalls"] else "wall_clock" if late else None)
    else:expected=None
    if expected is None or halt.get("code")!=expected:
        raise ExperimentError("halt does not follow from the recorded guard stage and evidence")


class Harness(SB.EngineHarness):
    """V0.12b's harness, with every request logged before its answer is
    checked, and a failed request logged with its error (review F5)."""

    def request(self, op, args=None):
        self.n += 1
        rid = f"r{self.n}"
        try:
            a = self.ses.request(rid, op, args)
        except BaseException as x:
            self.log.append({"op": op, "args": args, "answer": None, "error": f"{type(x).__name__}: {x}"})
            raise
        self.log.append({"op": op, "args": args, "answer": a})
        if a.get("id") != rid or a.get("protocol") != A3.AGENT_PROTOCOL:
            raise ExperimentError(f"the engine answered {a.get('id')!r} ({a.get('protocol')!r}) to {rid}")
        if not a.get("ok"):
            raise A3.ProtocolError(op, a.get("error") or {})
        return a["result"]


# --- the model ----------------------------------------------------------------

class ContextModel(SB.PointModel):
    """V0.12b's PointModel at a context: the first request must be the
    context's, byte for byte; its reply is the initial proposal; every later
    request goes to the transport after the limits, and is recorded."""

    def __init__(self, c, want, client, transport, limits, utc=SB.RR.utc, now=SB.time.monotonic):
        super().__init__({"id": c["id"], "proposal": c["proposal"]}, client, transport, limits, utc=utc, now=now)
        self.want = want

    def call(self, payload):
        self.lastRequest=payload
        if self.first:
            self.first = False
            if json.dumps(payload).encode("utf-8") != self.want:
                raise ExperimentError(f"{self.e['id']}: the agent's first request is not the context's request")
            return 200, json.dumps({"choices": [{"message": {"content": self.e["proposal"]["content"]}}]}), None
        n = len(self.calls)
        try:
            return SB.PointModel.call(self, payload)
        finally:
            if self.limits is not None and hasattr(self.limits, "after_call") and len(self.calls) > n:
                self.limits.after_call(self.calls[-1])
            if len(self.calls) > n:
                self.calls[-1]["request"] = json.loads(json.dumps(payload))
                self.calls[-1]["requestBody"] = json.dumps(payload)


# --- one draw -------------------------------------------------------------------

DETERMINISTIC = ("schema", "seq", "model", "context", "arm", "draw", "status", "halt", "initialDigests", "record",
                 "batches", "engine", "digestsAfter", "process", "engineInitialization", "failureStage", "modelCallsCompleted", "verificationStatus", "submittedRepresentation", "submittedRepresentationSha256")


def run_draw(entry, c, settings, opener, mem, transport, limits, utc=SB.RR.utc, now=SB.time.monotonic):
    arm = entry["arm"]
    if arm not in ARMS:
        raise ExperimentError(f"draw {entry['seq']}: arm {arm!r}")
    out = {"schema": DRAW_SCHEMA, **{k: entry[k] for k in ("seq", "model", "context", "arm", "draw")},
           "status": None, "halt": None, "initialDigests": None, "record": None, "batches": None, "engine": [],
           "digestsAfter": None, "calls": []}
    out.update(submittedRepresentation=None, submittedRepresentationSha256=None)
    ses, h, model = None, None, None
    out.update(engineInitialization="NOT_STARTED", failureStage=None, verificationStatus="UNVERIFIED")
    stage = "engine_open"
    try:
        out["engineInitialization"] = "STARTED"
        ses = opener(c, arm)
        out["engineInitialization"] = "OPENED"
        stage = "model_setup"
        h = Harness(ses)
        model = ContextModel(c, first_request(c, settings), A3.ModelClient(settings), transport, limits, utc=utc, now=now)
        stage = "initial_digests"
        d = h.request("digests")["digests"]
        if d != c["digests"]:
            raise ExperimentError(f"draw {entry['seq']}: the engine is not at the context's state")
        out["initialDigests"] = d
        out["engineInitialization"] = "READY"
        stage = "workflow"
        try:
            batches, rec = ALT.ask(h, SB.Sink(), model, settings, mem, c.get("point", 1), None, arm=arm)
        except Halt as x:
            out.update(status="interrupted", halt={"code": x.code, "message": x.message,
                                                   "evidence": getattr(x, "evidence", None)})
        else:
            if model.first:
                raise ExperimentError(f"draw {entry['seq']}: the agent made no request")
            turn=rec.get("turn")
            if turn:
                pins=turn["record"]
                if pins["presentationSha256"]!=file_sha256("internal/alternatives/text/presentation-v3.tmpl") or pins["viewTableSha256"]!=file_sha256("docs/benchmark/v13g-dev/presentation/view-table.txt"):
                    raise ExperimentError("engine presentation identities differ from the successor contract")
            representation=json.dumps(batches)
            out.update(status="completed", record=rec, batches=batches,
                       submittedRepresentation=representation, submittedRepresentationSha256=sha256(representation))
        stage = "final_digests"
        out["digestsAfter"] = h.request("digests")["digests"]
        out["engine"] = h.log
        out["verificationStatus"] = "RECORDED"
    except BaseException as x:
        out["status"] = "aborted"
        out["verificationStatus"] = "UNVERIFIED"
        out["batches"] = None
        out["engine"] = h.log if h else []
        out["failureStage"] = stage
        out["failure"] = {"type":type(x).__name__, "message":str(x)}
        if stage == "engine_open": out["engineInitialization"] = "FAILED"
        x.partialDraw = out
        raise
    finally:
        out["calls"] = model.calls if model else []
        out["modelCallsCompleted"] = len(out["calls"])
        if out["record"] is not None:
            out["process"]={k:out["record"][k] for k in ("eligibility","exposure","validationFirstCorrection","correctionFailure","deliberateNoAction","processFailure","submittedAction","executedAction")}
        else:
            turns=[e["answer"].get("result") for e in (h.log if h else []) if e["op"]=="alternatives.v3.turn" and isinstance(e.get("answer"),dict) and e["answer"].get("ok")]
            turn=turns[-1] if turns else None
            pending=getattr(model,"lastRequest",{}).get("messages",[])
            correction=any("PHASE 2 validation correction" in (call.get("request",{}).get("messages",[{}])[-1].get("content") or "") for call in out["calls"])
            out["process"]={"eligibility":{"eligible":turn.get("due") if turn else None,"reason":turn.get("why") if turn else "UNVERIFIED before comparison"},
                            "exposure":bool(turn and turn.get("message") and any(turn["message"]==m.get("content") for call in out["calls"] for m in call.get("request",{}).get("messages",[]))),"validationFirstCorrection":correction,
                            "correctionFailure":None if correction else False,"deliberateNoAction":False,"processFailure":True,
                            "intendedAction":None,"validatedAction":None,"submittedAction":None,"executedAction":{"status":"NOT_EXECUTED"},
                            "pendingMessage":pending[-1]["content"] if pending else None}
        if ses is not None:
            try: ses.close()
            except BaseException as x:
                out.update(status="aborted", failureStage="engine_close", verificationStatus="UNVERIFIED",
                           failure={"type":type(x).__name__, "message":str(x)})
                x.partialDraw = out
                raise
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
        self.settings = {m: settings_of(proto, m, env) for m in proto["modelOrder"]}
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


def replay_draw(r, c, settings, opener, mem, cfg=None):
    """A recorded draw again, without a model: everything but the live
    metadata must be the recorded draw's, and every response used. The
    response metadata (usage, finish reason, served model) is made again
    from each body; an interruption is raised again at the same call, and
    its cause must be what its recorded evidence gives under cfg's limits
    (review F6)."""
    for x in r["calls"]:
        f = SB.body_fields(x["body"]) if x.get("body") is not None else {"parsedBody": False}
        want = {k: (f.get(k) if f.get("parsedBody") else None) for k in ("usage", "finishReason", "servedModel")}
        if {k: x.get(k) for k in want} != want:
            raise ExperimentError(f"draw {r['seq']}: call {x['n']}'s metadata is not its body's")
    if r["status"] == "interrupted":
        if cfg is None:
            raise ExperimentError(f"draw {r['seq']}: an interrupted draw is replayed only with the protocol's limits")
        check_halt(r["halt"], cfg)
        if (r["halt"].get("evidence") or {}).get("drawCalls") != len(r["calls"]):
            raise ExperimentError(f"draw {r['seq']}: the halt's evidence is not the draw's calls")
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
                    x = Halt(halt["code"], halt["message"])
                    x.evidence = halt.get("evidence")
                    raise x
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
        out["observationAfterDay"] = h.request("observe", {"recentLog":2000})
        out["engine"] = h.log
        return out
    finally:
        ses.close()


# --- classification ---------------------------------------------------------------

RESULTS = ("retained", "selected", "selected_no_action", "revised", "abandoned", "invalid_alternative", "no_decision")
NOT_DUE = "not_due"


def classify(d):
    rec=d["record"]
    turn=rec.get("turn") or {}
    o=rec.get("outcome") or {}
    result=rec["result"] if rec["result"] in RESULTS else NOT_DUE
    sub=rec["submittedAction"]
    flags=o.get("flags") or []
    return {"result":result,"responseRoute":rec.get("responseRoute"),"secondaryRoute":o.get("final") or ("no_decision" if result=="no_decision" else None),
            "eligibility":rec["eligibility"],"exposure":rec["exposure"],
            "validationFirstCorrection":rec["validationFirstCorrection"],"correctionFailure":rec["correctionFailure"],
            "deliberateNoAction":rec["deliberateNoAction"],"processFailure":rec["processFailure"],
            "intendedAction":rec["intendedAction"],"validatedAction":rec["validatedAction"],
            "submittedAction":sub,"executedAction":rec["executedAction"],
            "comparisonPresented":bool(turn.get("due")) and d["arm"]=="treatment",
            "inspected":"inspected" in flags,"compared":"compared" in flags,
            "finalValid":bool(o.get("valid")) if rec["exposure"] else bool(rec["validatedAction"] is not None),
            "submitted":bool(d["batches"]),"modelCalls":len(d["calls"]),"retries":rec["retryRequests"],
            "batchesSha256":sub["sha256"],"proposalSubmittedSha256":(rec.get("proposalSubmittedAction") or {}).get("sha256")}


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
    """The most model calls a draw can make after its initial proposal: one
    correction when the proposal is refused (llm_agent5, maxRetries 1: the
    refusal itself uses the one failure allowed, so a failed correction ends
    the point), then the turn's reply, one inspect answer and one retry."""
    refused = c.get("category", "").startswith("refused")
    return (1 if refused else 0) + 1 + ALT.INSPECT_ROUNDS + ALT.TURN_RETRIES


def draft_protocol(rows, contexts_sha):
    """The revised draft protocol (docs/v13g/PROTOCOL_DRAFT.md), made from
    the committed inputs only."""
    contexts = {}
    for c in rows:
        aa = c["kind"] == "historical" and not c["expected"]["stepDue"] and not c["category"].startswith("refused")
        contexts[c["id"]] = {"kind": c["kind"], "aa": aa, "model": c.get("model"), "callBound": call_bound(c)}
    proto = {"schema": PROTOCOL_SCHEMA, "id": "v13g-alt-1", "status": DRAFT, "driver": DRIVER,
             "workflow": {"version": ALT.WORKFLOW_VERSION, "sha256": ALT.WORKFLOW_SHA256,
                          "contracts": ALT.CONTRACT_PINS},
             "contextsFileSha256": contexts_sha, "contexts": contexts,
             "models": load(f"{V12B}/protocol.json")["models"], "modelOrder": [GEMMA, QWEN],
             "settings": {"files": settings_files(), "maxRetries": 1, "allowDetails": True},
             "systemPromptSha256": sha256(ALT.SYSTEM_PROMPT),
             "request": {"timeoutSeconds": 300},
             "drawsPerUnit": {GEMMA: {"unit": 3, "aa": 2}, QWEN: {"unit": 5, "aa": 2}},
             "orderRule": ORDER_RULE,
             "texts": {"presentation": {"id": "dreamteam.alternatives-presentation/v3",
                                        "file": "internal/alternatives/text/presentation-v3.tmpl",
                                        "sha256": file_sha256("internal/alternatives/text/presentation-v3.tmpl")},
                       "viewTable": {"id": "dreamteam.alternatives-view/v2",
                                     "file": "docs/benchmark/v13g-dev/presentation/view-table.txt",
                                     "sha256": file_sha256("docs/benchmark/v13g-dev/presentation/view-table.txt")},
                       "selectionRule": {"id": "dreamteam.alternative-selection/v2",
                                         "file": "internal/alternatives/text/selection-v2.txt",
                                         "sha256": file_sha256("internal/alternatives/text/selection-v2.txt")},
                       "arms": "dreamteam.alternatives-arms/v3"},
             "requests": requests_of(rows, settings_files()),
             "arms": {"control": "validation-first, then the shared head and foot only",
                      "treatment": "validation-first, then the shared head, the verified comparison (one inspect "
                                   "round) and the shared foot"},
             "analysis": {"estimand": "per context and model: the difference between the arms' distributions of the "
                                      "canonical submitted action batches within comparison-exposed draws; per-identity "
                                      "treatment minus control differences and total variation; route labels secondary",
                          "pairs": "secondary, per sensitivity pair and model: departure of canonical submitted identity "
                                   "from the validated proposal identity, in exposed member b minus exposed member a, "
                                   "per arm (Newcombe), and treatment minus control (MOVER); a context pair, not one fact",
                          "observable": "eligibility, exposure, validation-first correction, correction failure, deliberate "
                                        "no action, process failure, original response route, seven secondary process labels, "
                                        "intended/validated/submitted/executed action, inspection, citations, calls and retries",
                          "notClaimed": "a changed decision is not an improvement; the reason is never read"}}
    sched = schedule(proto, rows)
    proto["duplicateInputs"] = [["S1-a", "S6-a"]]
    proto["intervention"] = "complete alternatives interface: information, menu, inspection, selection shortcuts, additional calls, presentation order"
    proto["draws"] = len(sched)
    proto["maxModelCallsPerDraw"] = 6
    proto["maxModelCalls"] = sum(proto["contexts"][s["context"]]["callBound"] for s in sched)
    proto["stop"] = {"wallClockSeconds": 4 * 3600, "timeGuarantee": "request-start guard; blocked operations are not bounded", "consecutiveTransportFailures": 5,
                     "health": "an unhealthy /health before a round stops the run",
                     "serverChange": "a material change before a round stops the run",
                     "partial": "a run that stops early is PARTIAL; the analysis reports only what was observed",
                     "resume": "never"}
    return proto, sched


# --- the run ------------------------------------------------------------------------

RUN_SCHEMA = "dreamteam.alternatives-run/v3"


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
    limits = Limits(proto["maxModelCallsPerDraw"], proto["maxModelCalls"], stop["wallClockSeconds"],
                    proto["request"]["timeoutSeconds"], max_fails=stop["consecutiveTransportFailures"], now=now)
    started, completed, stopped, interrupted, aborted, fails, prev = utc(), 0, None, None, None, 0, None
    sig = None
    if state_fn is not None:
        before = state_fn()
        write_json(os.path.join(live, "server-before.json"), before)
        sig = X.server_signature(before)
        if (before.get("health") or {}).get("status") != "healthy":
            stopped = {"code": "health", "message": "the server did not answer healthy before the run", "state": before}
    try:
        with open(os.path.join(live, "draws.jsonl"), "w", encoding="utf-8") as f:
            for r in sched if stopped is None else []:
                if state_fn is not None and (r["model"], r["draw"]) != prev:
                    prev = (r["model"], r["draw"])
                    st = state_fn()
                    if (st.get("health") or {}).get("status") != "healthy":
                        stopped = {"code": "health", "message": f"unhealthy before draw {r['seq']}", "state": st}
                        break
                    if X.server_signature(st) != sig:
                        stopped = {"code": "server_change", "message": f"server changed before draw {r['seq']}", "state": st, "beforeSignature": sig}
                        break
                try:
                    limits.before_draw()
                except Halt as x:
                    stopped = {"code": x.code, "message": x.message, "evidence": getattr(x,"evidence",None)}
                    break
                c = ctxs.by_id[r["context"]]
                try:
                    d = run_draw(r, c, ctxs.settings[r["model"]], opener, ctxs.memory(c, r["model"]),
                                 transports[r["model"]], limits, utc=utc, now=now)
                except BaseException as x:
                    aborted = r["seq"]
                    partial = getattr(x, "partialDraw", None)
                    if partial is None:
                        partial = {"schema":DRAW_SCHEMA, **r, "status":"aborted", "calls":[], "engine":[],
                                   "batches":None, "record":None, "initialDigests":None, "digestsAfter":None,
                                   "engineInitialization":"NOT_STARTED", "failureStage":"draw_setup",
                                   "modelCallsCompleted":0, "verificationStatus":"UNVERIFIED",
                                   "failure":{"type":type(x).__name__, "message":str(x)}}
                    partial["stopReason"] = "operator" if isinstance(x, KeyboardInterrupt) else "driver"
                    write_json(os.path.join(live, "aborted-draw.json"), partial)
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
    finally:
        opener.close()
        status = "COMPLETE" if completed == len(sched) and stopped is None else "PARTIAL"
        doc = {"schema": RUN_SCHEMA, "driver": DRIVER, "protocol": proto["id"], "protocolSha256": proto_sha,
               "authorization": authorization, "started": started, "ended": utc(), "status": status,
               "drawsScheduled": len(sched), "drawsCompleted": completed, "drawInterrupted": interrupted,
               "drawAborted": aborted,
               "drawsNotRun": len(sched) - completed - (1 if interrupted else 0) - (1 if aborted else 0),
               "modelCalls": limits.calls, "stopped": stopped, "resume": "never"}
        doc["stopEvent"] = None
        if stopped is not None and interrupted is None and aborted is None:
            event = {"schema":"dreamteam.between-draw-stop/v1", "completed":completed,
                     "nextDraw":sched[completed] if completed < len(sched) else None,
                     "modelCalls":limits.calls, "stop":stopped,
                     "verificationStatus":"EVIDENCE_CHECKED" if stopped["code"]=="wall_clock" else "UNVERIFIED_EXTERNAL_STATE"}
            doc["stopEvent"] = event
            write_json(os.path.join(live, "stop-event.json"), event)
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


FINALS = ("original_retained", "listed_alternative", "no_staffing_action", "other_batch", "abandoned", "invalid",
          "no_decision", NOT_DUE)


STOP_CODES={"draw_call_limit","call_budget","wall_clock","transport","health","server_change","driver","operator"}

def check_run_doc(doc,rows,sched,proto_sha=None,proto=None,aborted=None):
    check_rows(sched,rows)
    def require(ok,message):
        if not ok: raise ExperimentError(message)
    integer=lambda x:isinstance(x,int) and not isinstance(x,bool) and x>=0
    for key in ("drawsScheduled","drawsCompleted","drawsNotRun","modelCalls"):
        require(integer(doc.get(key)),"invalid count: "+key)
    if proto:
        require(doc.get("schema")==RUN_SCHEMA and doc.get("driver")==DRIVER and doc.get("protocol")==proto["id"],"run schema, driver or protocol identity disagrees")
    require(doc.get("drawInterrupted") is None or integer(doc["drawInterrupted"]),"invalid interrupted draw identity")
    for row in rows:
        require(integer(row.get("seq")) and integer(row.get("draw")),"invalid recorded draw identity")
    completed=sum(r["status"]=="completed" for r in rows)
    interrupted=next((r["seq"] for r in rows if r["status"]=="interrupted"),None)
    require(doc["drawsScheduled"]==len(sched) and doc["drawsCompleted"]==completed,"draw counts disagree")
    require(doc.get("drawInterrupted")==interrupted,"interrupted draw disagrees")
    if proto_sha is not None:require(doc.get("protocolSha256")==proto_sha,"protocol identity disagrees")
    aid=doc.get("drawAborted")
    if aid is not None:
        require(integer(aid) and len(rows)<len(sched) and aid==sched[len(rows)]["seq"] and interrupted is None,"aborted draw is not the next scheduled draw")
        require(aborted is not None,"aborted draw artifact missing")
        require(aborted.get("schema")==DRAW_SCHEMA and aborted.get("status")=="aborted","aborted artifact status disagrees")
        require(aborted.get("engineInitialization") in {"NOT_STARTED","STARTED","OPENED","READY","FAILED"},"aborted initialization status missing")
        require(aborted.get("failureStage") in {"draw_setup","engine_open","model_setup","initial_digests","workflow","final_digests","engine_close"},"aborted failure stage missing")
        require(aborted.get("verificationStatus")=="UNVERIFIED" and isinstance(aborted.get("failure"),dict),"aborted failure evidence missing")
        require(integer(aborted.get("modelCallsCompleted")) and aborted["modelCallsCompleted"]==len(aborted.get("calls",[])),"aborted model calls disagree")
        require(aborted.get("batches") is None or aborted.get("failureStage")=="engine_close","aborted draw submitted actions")
        require(integer(aborted.get("seq")) and integer(aborted.get("draw")),"invalid aborted draw identity")
        require({k:aborted.get(k) for k in ("seq","model","context","arm","draw")}==sched[len(rows)],"aborted artifact is not the scheduled draw")
    else:require(aborted is None,"unexpected aborted artifact")
    require(doc["drawsNotRun"]==len(sched)-completed-int(interrupted is not None)-int(aid is not None),"remaining draw count disagrees")
    observed=list(rows)+([aborted] if aborted is not None else [])
    calls=sum(len(r["calls"]) for r in observed)
    streak=0
    for draw in observed:
        for call in draw["calls"]:
            streak=streak+1 if call.get("status") is None and call.get("error") else 0
    require(doc["modelCalls"]==calls,"model calls disagree with recorded exchanges")
    for r in observed:
        for n,call in enumerate(r["calls"],1):
            require(integer(call.get("n")) and call.get("n")==n,"call order disagrees")
            require(call.get("responseSha256")==sha256(call.get("body") or ""),"response hash disagrees")
            f=SB.body_fields(call.get("body"))
            for field in ("usage","servedModel","finishReason"):
                require(call.get(field)==(f.get(field) if f.get("parsedBody") else None),"response metadata disagrees")
            request=call.get("request")
            request_body=call.get("requestBody")
            require(isinstance(request,dict) and isinstance(request_body,str) and json.loads(request_body)==request and call.get("requestSha256")==sha256(request_body),"recorded request hash disagrees")
        if proto:
            bound=proto.get("contexts",{}).get(r["context"],{}).get("callBound",proto["maxModelCallsPerDraw"])
            require(len(r["calls"])<=min(bound,proto["maxModelCallsPerDraw"]),"impossible per-draw call total")
    if proto:require(calls<=proto["maxModelCalls"],"impossible run call total")
    stop=doc.get("stopped")
    require(doc.get("status") in ("COMPLETE","PARTIAL"),"unknown run status")
    if doc["status"]=="COMPLETE":
        require(completed==len(sched) and len(rows)==len(sched) and stop is None and aid is None and doc.get("stopEvent") is None,"partial coverage cannot be COMPLETE")
    else:
        require(isinstance(stop,dict) and stop.get("code") in STOP_CODES,"unknown or missing stop code")
        if stop["code"]=="health":
            health=stop.get("state",{}).get("health") or {}
            require(isinstance(health.get("status"),str) and bool(health["status"]) and health["status"]!="healthy","health stop lacks evidence")
        if stop["code"]=="server_change":require(stop.get("beforeSignature") is not None and bool(stop.get("state")) and X.server_signature(stop["state"])!=stop["beforeSignature"],"server-change stop lacks evidence")
        if interrupted:
            require(stop==rows[-1]["halt"],"halt differs from interrupted draw")
        if aid is not None:
            require(stop["code"] in ("driver","operator") and aborted.get("stopReason")==stop["code"],"aborted draw stop code disagrees")
        if stop["code"] in ("driver","operator"):
            require(aid is not None,"driver/operator stop requires aborted draw")
        event=doc.get("stopEvent")
        if interrupted is None and aid is None:
            require(completed < len(sched),"between-draw stop after completed schedule")
            require(stop["code"] in ("health","server_change","wall_clock"),"unsupported between-draw stop")
            require(isinstance(event,dict) and event.get("schema")=="dreamteam.between-draw-stop/v1", "between-draw stop event missing")
            require(integer(event.get("completed")) and integer(event.get("modelCalls")),"invalid stop event count")
            next_draw=event.get("nextDraw")
            require(isinstance(next_draw,dict) and integer(next_draw.get("seq")) and integer(next_draw.get("draw")),"invalid stop event draw identity")
            require(event["completed"]==completed and next_draw==sched[len(rows)] and event["modelCalls"]==calls and event.get("stop")==stop,"between-draw stop event contradicts run")
            expected="EVIDENCE_CHECKED" if stop["code"]=="wall_clock" else "UNVERIFIED_EXTERNAL_STATE"
            require(event.get("verificationStatus")==expected,"stop event verification status disagrees")
        else:
            require(event is None,"invented between-draw stop event")
            if interrupted:require(stop["code"] in {"draw_call_limit","call_budget","wall_clock","transport"},"unsupported interrupted stop")
        if proto and stop["code"] in {"draw_call_limit","call_budget","wall_clock","transport"}:
            check_halt(stop,proto)
            e=stop.get("evidence") or {}
            require(e.get("runCalls")==calls,"halt run call evidence disagrees")
            require(e.get("consecutiveTransportFailures")==streak,"halt failure streak disagrees with recorded calls")
            require(e.get("drawCalls")== (len(rows[-1]["calls"]) if interrupted else 0),"halt draw call evidence disagrees")
    require(doc.get("resume")=="never","partial-run metadata permits resumption")


def tv(a, b):
    """Total variation distance between two count tables."""
    na, nb = sum(a.values()), sum(b.values())
    keys = set(a) | set(b)
    return round(0.5 * sum(abs(a.get(k, 0) / na - b.get(k, 0) / nb) for k in keys), 4)


def primary_comparison(control,treatment):
    a=collections.Counter(x["batchesSha256"] for x in control if x["exposure"])
    b=collections.Counter(x["batchesSha256"] for x in treatment if x["exposure"])
    na,nb=sum(a.values()),sum(b.values())
    if not na or not nb:return None
    return {"controlExposed":na,"treatmentExposed":nb,"totalVariation":tv(a,b),
            "bySubmittedIdentity":{k:X.newcombe(b[k],nb,a[k],na) for k in sorted(set(a)|set(b))}}


def analyze(binary,run_dirs,ctxs,proto,sched,rows,run_doc,proto_sha=None,aborted=None):
    check_run_doc(run_doc,rows,sched,proto_sha,proto,aborted)
    pool=ctxs.opener(binary)
    out,ledger=[],{}
    aborted_verification=None
    try:
        for r in rows:
            c=ctxs.by_id[r["context"]]
            replay_draw(r,c,ctxs.settings[r["model"]],pool,ctxs.memory(c,r["model"]),proto)
            if r["status"]!="completed":continue
            x=classify(r)
            # The canonical identity is recomputed by the engine, independently
            # of every stored route/outcome field.
            ses=pool(c,r["arm"])
            actual=Harness(ses).request("alternatives.v3.canonical",{"batches":r["batches"]})
            if actual!=x["submittedAction"]:raise ExperimentError("submitted identity disagrees")
            representation = r["submittedRepresentation"]
            representation_sha = sha256(representation)
            if representation_sha != r["submittedRepresentationSha256"] or json.loads(representation) != r["batches"]:
                raise ExperimentError("exact submitted representation disagrees")
            key=sha256(c["id"]+"|"+representation_sha)
            x["submittedRepresentationSha256"] = representation_sha
            x["executionRecord"] = key
            if key not in ledger:
                e=execute(binary,run_dirs,c,json.loads(representation)) if r["batches"] else {"status":"EMPTY_SUBMISSION","batches":[],"executed":False,"partial":False}
                ledger[key]={"context":c["id"],"batches":r["batches"],"submittedRepresentation":representation,"submittedRepresentationSha256":representation_sha,"draws":[],"execution":e}
            ledger[key]["draws"].append({k:r[k] for k in ("seq","context","model","arm","draw")})
            x["executedAction"]=ledger[key]["execution"]
            x.update(seq=r["seq"],context=r["context"],model=r["model"],arm=r["arm"],draw=r["draw"])
            out.append(x)
        if aborted is not None:
            aborted_verification=verify_aborted(aborted,ctxs,pool,proto)
    finally:pool.close()
    cells=collections.defaultdict(list)
    for x in out:cells[(x["context"],x["model"],x["arm"])].append(x)
    table=[]
    for (cid,mod,arm),xs in sorted(cells.items()):
        exposed=[x for x in xs if x["exposure"]]
        table.append({"context":cid,"model":mod,"arm":arm,"observed":len(xs),"exposed":len(exposed),
                      "primarySubmitted":dict(sorted(collections.Counter(x["batchesSha256"] for x in exposed).items())),
                      "secondaryRoutes":dict(sorted(collections.Counter(x["secondaryRoute"] for x in exposed).items(),key=lambda p:str(p[0]))),
                      "results":dict(sorted(collections.Counter(x["result"] for x in xs).items())),
                      **{k:sum(bool(x[k]) for x in xs) for k in ("validationFirstCorrection","correctionFailure","deliberateNoAction","processFailure")},
                      "modelCalls":sum(x["modelCalls"] for x in xs),"retries":sum(x["retries"] for x in xs)})
    differences=[]
    if run_doc["status"]=="COMPLETE":
        for cid,mod,arm in sorted(cells):
            if arm!="treatment":continue
            comp=primary_comparison(cells.get((cid,mod,"control"),[]),cells[(cid,mod,arm)])
            if comp is not None:differences.append(dict(comp,context=cid,model=mod))
    pairs=[]
    if run_doc["status"]=="COMPLETE":
        for pid in sorted({c["pair"] for c in ctxs.by_id.values() if c.get("pair")}):
            for mod in proto["modelOrder"]:
                dep={}
                for arm in ARMS:
                    members=[[x for x in cells.get((pid+"-"+side,mod,arm),[]) if x["exposure"]] for side in ("a","b")]
                    if all(members):
                        a,b=members
                        count_dep=lambda xs:sum(x["batchesSha256"]!=x["proposalSubmittedSha256"] for x in xs)
                        dep[arm]=X.newcombe(count_dep(b),len(b),count_dep(a),len(a))
                if len(dep)==2:pairs.append({"pair":pid,"model":mod,"departure":dep,"sensitivity":mover(dep["treatment"],dep["control"])})
    return {"schema":"dreamteam.alternatives-analysis/v4","runStatus":run_doc["status"],"draws":out,"executions":ledger,
            "table":table,"armDifferences":differences,"pairs":pairs,"abortedVerification":aborted_verification,
            "note":"Exposed-only canonical submitted actions. Partial coverage yields no complete-run comparisons. Pair estimates remain descriptive context comparisons; duplicate S1-a/S6-a retained."}


def verify_aborted(r,ctxs,pool,proto):
    if r.get("engineInitialization") in ("FAILED", "NOT_STARTED", "STARTED", "OPENED"):
        if r["calls"] or r.get("initialDigests") is not None or r.get("batches") is not None:
            raise ExperimentError("initialization failure invents completed workflow evidence")
        return {"status":"UNVERIFIED","seq":r["seq"],"failureStage":r.get("failureStage"),
                "reason":"engine initialization failure cannot be reconstructed; no responses invented"}
    # External driver/operator failures are not invented during replay. If the
    # recorded failing transport supports reconstruction, replay it; otherwise
    # mark its exchange evidence UNVERIFIED and prohibit complete-run claims.
    if not r["calls"] or r["calls"][-1].get("body") is not None:
        return {"status":"UNVERIFIED","seq":r["seq"],"reason":"external failure cannot be reconstructed from recorded responses"}
    class RecordedFailure(Exception):pass
    calls=list(r["calls"])
    def transport(payload):
        if not calls:raise ExperimentError("unrecorded aborted request")
        call=calls.pop(0)
        if sha256(json.dumps(payload).encode())!=call["requestSha256"]:raise ExperimentError("aborted request disagrees")
        if call.get("body") is None:raise RecordedFailure(call.get("error"))
        return call["status"],call["body"],call["error"]
    c=ctxs.by_id[r["context"]]
    try:run_draw(r,c,ctxs.settings[r["model"]],pool,ctxs.memory(c,r["model"]),transport,None)
    except RecordedFailure as x:
        p=x.partialDraw
        if calls or p["engine"]!=r["engine"]:raise ExperimentError("aborted engine exchanges disagree")
        return {"status":"RECONSTRUCTED","seq":r["seq"],"reason":"recorded transport failure reproduced; external cause unverified"}
    raise ExperimentError("aborted draw completed during reconstruction")



def check_stop_artifact(live, doc):
    path=os.path.join(live,"stop-event.json")
    event=json.load(open(path,encoding="utf-8")) if os.path.exists(path) else None
    if event != doc.get("stopEvent"):raise ExperimentError("stop event artifact missing or disagrees")


def cmd_analyze(a):
    proto, raw = json.load(open(a.protocol, encoding="utf-8")), open(a.protocol, "rb").read()
    rows_c, csha = load_contexts(a.contexts)
    if csha != proto["contextsFileSha256"]:
        raise SystemExit("the contexts file is not the protocol's")
    sched = schedule(proto, rows_c)
    env = {SB.SETTINGS_ENV: SB.REPLAY_URL}
    ctxs = Contexts(proto, rows_c, a.replays, env)
    rows = SB.load_jsonl(os.path.join(a.dir, "live", "draws.jsonl"))
    run_doc = json.load(open(os.path.join(a.dir, "live", "run.json"), encoding="utf-8"))
    apath=os.path.join(a.dir,"live","aborted-draw.json")
    aborted=json.load(open(apath,encoding="utf-8")) if os.path.exists(apath) else None
    check_stop_artifact(os.path.join(a.dir,"live"), run_doc)
    doc = analyze(a.binary, ctxs.run_dirs, ctxs, proto, sched, rows, run_doc, sha256(raw),aborted)
    write_json(os.path.join(a.out, "analysis.json"), doc)
    print(f"analysis: {len(doc['draws'])} completed draws ({doc['runStatus']}), {len(doc['executions'])} executions")


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
                t = next((e["answer"]["result"] for e in d["engine"] if e["op"] == "alternatives.v3.turn"), None)
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
    p = sub.add_parser("analyze")
    p.add_argument("--binary", required=True)
    p.add_argument("--protocol", required=True)
    p.add_argument("--contexts", required=True)
    p.add_argument("--replays")
    p.add_argument("--dir", required=True)
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
    elif a.cmd == "analyze":
        cmd_analyze(a)
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
