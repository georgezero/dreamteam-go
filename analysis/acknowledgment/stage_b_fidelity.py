#!/usr/bin/env python3
"""The Stage B driver's fidelity evidence (dreamteam.acknowledgment-
fidelity/v1), on the real engine at the twelve branch points, with scripted
model replies. No model is contacted.

    python3 -I -B analysis/acknowledgment/stage_b_fidelity.py fidelity --binary ./dt --replays DIR \\
        --protocol FILE --out DIR
    python3 -I -B analysis/acknowledgment/stage_b_fidelity.py mockrun --binary ./dt --replays DIR \\
        --protocol FILE --out DIR

fidelity: at every point and arm, each scripted case runs twice:
* through the driver (stage_b.run_draw: the pooled engine session, the
  driver's harness, PointModel, the fixed memory);
* as the agent itself would run it: the agent's own Harness class
  (llm_agent3.Harness for A, llm_agent6.Harness for B) on the pipe of a
  separate engine session (one per point), its own ModelClient's payloads, its own Evidence writer,
  and for B its own LiveModel.
Both must make the same model requests (byte for byte), the same engine
requests with the same answers, the same decision record and the same
batches. The driver's draw is then replayed without the model. At the end of
each point, the reference applies the last submitted decision on its own
session, and its state digests must be the analysis continuation's.

mockrun: the protocol's own schedule run by stage_b.run with scripted
replies and a scripted clock that reaches the four-hour stop part-way
through, so the run is PARTIAL by construction (one draw interrupted); then
stage_b.analyze on it. Its outputs are the evidence that a partial run is
recorded, replayed and reported as partial.

Each output is a function of the engine, the agents, the driver and the
scripts; CI makes it again and compares it on every platform.
"""

import argparse
import copy
import json
import os
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import stage_b as B  # noqa: E402

A3, A5, A6, X, P = B.A3, B.A5, B.A6, B.X, B.P
sha256, canon, write_json = B.sha256, B.canon, B.write_json
SCHEMA = "dreamteam.acknowledgment-fidelity/v1"
MOCK_AUTHORIZATION = "none: a scripted mock run (stage_b_fidelity.py mockrun); no model is contacted"


# --- the scripted model ---------------------------------------------------------

def last(messages, role):
    return next((m["content"] for m in reversed(messages) if m["role"] == role), None)


def actions_of(content):
    try:
        d = json.loads(A3.strip_wrapping(content))
    except (TypeError, ValueError):
        return None
    return d.get("actions") if isinstance(d, dict) else None


def request_items(messages):
    """The latest acknowledgment request in the conversation: (token, its
    items as shown, every earlier token)."""
    tokens, found = [], None
    for m in messages:
        if m["role"] != "user" or "(proposal " not in m["content"]:
            continue
        t = m["content"]
        k = t.rindex("(proposal ")
        tok = t[k + 10:k + 26]
        body = t[t.index("):\n", k) + 3:].split("\n\n", 1)[0]
        tokens.append(tok)
        found = (tok, [json.loads(l) for l in body.splitlines() if l.strip()])
    if found is None:
        return None, None, tokens
    return found[0], found[1], tokens[:-1]


def wrong(items):
    """The items with one consequence value changed (the first one)."""
    items = copy.deepcopy(items)
    vals = items[0]["consequences"][next(iter(items[0]["consequences"]))]
    f = next(iter(vals))
    vals[f] = vals[f] + 1 if isinstance(vals[f], int) else "2099-01-01"
    return items


def content_for(kind, messages):
    """A scripted reply of one kind to the conversation so far."""
    acts = actions_of(last(messages, "assistant"))
    tok, items, earlier = request_items(messages)
    if kind == "garbage":
        return "I need to think about this further before deciding."
    if kind == "details":
        return json.dumps({"reason": "More detail first.", "details": ["finance.cashHistory"]})
    if kind == "abandon":
        return json.dumps({"reason": "On reflection, no change.", "actions": []})
    if kind == "keep":
        return json.dumps({"reason": "Proceed as proposed.", "actions": acts})
    if kind == "revise_drop":
        return json.dumps({"reason": "Only the first part.", "actions": acts[:-1]})
    if kind.startswith("ack"):
        if tok is None:
            raise B.ExperimentError(f"scripted {kind}: no acknowledgment request to answer")
        ack = {"proposal": tok, "items": items}
        if kind == "ack_wrong":
            ack["items"] = wrong(items)
        elif kind == "ack_stale":
            ack["proposal"] = earlier[0] if earlier else "0" * 16
        elif kind == "ack_field":
            ack["items"] = [{k: v for k, v in items[0].items() if k != "consequences"}] + items[1:]
        elif kind == "ack_missing":
            return json.dumps({"reason": "Proceed.", "actions": acts})
        elif kind != "ack":
            raise B.ExperimentError(f"unknown scripted reply {kind!r}")
        return json.dumps({"reason": "Acknowledged.", "actions": acts, "acknowledgment": ack})
    raise B.ExperimentError(f"unknown scripted reply {kind!r}")


def body_of(payload, content):
    return json.dumps({"id": "scripted", "model": payload["model"], "choices": [
        {"index": 0, "message": {"role": "assistant", "content": content}, "finish_reason": "stop"}],
        "usage": {"prompt_tokens": len(json.dumps(payload)) // 4, "completion_tokens": len(content) // 4,
                  "total_tokens": len(json.dumps(payload)) // 4 + len(content) // 4}})


class Script:
    """A transport answering each request with the next scripted kind:
    a reply kind (content_for), "timeout" (no answer) or "http500". The
    payloads it was sent are kept."""

    def __init__(self, kinds):
        self.kinds, self.payloads = list(kinds), []

    def __call__(self, payload):
        self.payloads.append(json.loads(json.dumps(payload)))
        if not self.kinds:
            raise B.ExperimentError("the workflow asked for more replies than were scripted")
        k = self.kinds.pop(0)
        if k == "timeout":
            return None, "", "timeout: no answer within 300 s"
        if k == "http500":
            return 500, '{"error":"scripted"}', "http_status: HTTP 500"
        return 200, body_of(payload, content_for(k, payload["messages"])), None


def mock_transport(seed):
    """A deterministic scripted model for a mock run: per request, a reply
    kind from the conversation and a hash of it and of the request's
    number (so that repeated draws differ, as sampled replies may)."""
    n = [0]

    def transport(payload):
        msgs = payload["messages"]
        tok, _, _ = request_items(msgs)
        n[0] += 1
        h = int(sha256(f"{seed}|{n[0]}|" + canon(msgs))[:8], 16)
        if tok is not None and "ACKNOWLEDGMENT" in last(msgs, "user")[:40]:
            kind = ("ack", "ack", "ack_wrong", "abandon", "ack_missing")[h % 5]
        elif last(msgs, "user").startswith("VALIDATION."):
            kind = ("keep", "abandon", "garbage")[h % 3]
        else:
            kind = ("keep", "keep", "abandon", "garbage")[h % 4]
        if h % 23 == 0:
            return None, "", "timeout: no answer within 300 s"
        return 200, body_of(payload, content_for(kind, msgs)), None
    return transport


# --- the reference: the agent as it runs itself ------------------------------------

class Pipe:
    """The engine's pipe as the agent's Harness uses it (readline, write,
    flush), every line kept."""

    def __init__(self, proc):
        self.proc, self.lines = proc, []

    def readline(self):
        line = self.proc.stdout.readline()
        self.lines.append(("in", line))
        return line

    def write(self, s):
        self.lines.append(("out", s))
        self.proc.stdin.write(s)

    def flush(self):
        self.proc.stdin.flush()


class Reference:
    """An engine session at the point, for the agent's own Harness."""

    def __init__(self, binary, run_dir, e):
        run, pairs = X.transcript(run_dir)
        self.scratch = tempfile.TemporaryDirectory(prefix="v12b-ref-")
        self.proc = subprocess.Popen([os.path.abspath(binary), "agent", "--stdio", "-scenario", run["scenario"]["ref"],
                                      "-seed", str(run["seed"])], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                     stderr=subprocess.DEVNULL, text=True, encoding="utf-8", cwd=self.scratch.name)

        def ask(r):
            self.proc.stdin.write(json.dumps(r) + "\n")
            self.proc.stdin.flush()
            return json.loads(self.proc.stdout.readline())
        X.Session(run, pairs, ask).to_point(e["digests"])

    def close(self):
        self.proc.stdin.close()
        self.proc.wait()
        self.scratch.cleanup()


class RefClient:
    """The agent's ModelClient payloads; the recorded proposal as the first
    reply (as the recording had it), then the script."""

    def __init__(self, settings, e, transport):
        self.mc, self.e, self.transport, self.first, self.payloads = A3.ModelClient(settings), e, transport, True, []

    def payload(self, messages):
        return self.mc.payload(messages)

    def call(self, payload):
        self.payloads.append(json.loads(json.dumps(payload)))
        if self.first:
            self.first = False
            return 200, json.dumps({"choices": [{"message": {"content": self.e["proposal"]["content"]}}]}), None
        return self.transport(payload)


def reference(ref, e, arm, settings, mem, kinds):
    """The agent's own run of the case: (batches, record, model payloads
    after the proposal, engine exchanges [(request, answer)])."""
    pipe = Pipe(ref.proc)
    h = (A6.Harness if arm == "B" else A3.Harness)(pipe, pipe)
    client = RefClient(settings, e, Script(kinds))
    with tempfile.TemporaryDirectory(prefix="v12b-ev-") as d:
        ev = A3.Evidence(d)
        if arm == "B":
            batches, rec = A6.ask(h, ev, A6.LiveModel(client, settings), settings, mem, e["point"], None)
            rec["workflow"] = B.ARM_WORKFLOW["B"]
        else:
            batches, rec = A5.ask(h, ev, client, settings, mem, e["point"], None, workflow=B.ARM_WORKFLOW["A"])
    reader = A6.json_loads if arm == "B" else json.loads
    ex, req = [], None
    for way, line in pipe.lines:
        if way == "out":
            req = json.loads(line)
        else:
            ex.append((req, reader(line)))
    return batches, rec, client.payloads[1:], ex


# --- the cases -----------------------------------------------------------------------

def cases_for(e, arm):
    """The scripted cases at a point and arm (each a list of reply kinds)."""
    multi = len(e["proposal"]["actions"]) > 1
    qualifying = e["stageB"]["category"] in ("gemma_valid", "termination_displacement", "teleradiology_guarantee",
                                             "multiple_qualifying")
    if e["validation"] == "refused":
        return [["keep"], ["abandon"], ["garbage"], ["http500", "abandon"]]
    if arm == "A":
        return [["keep"], ["abandon"], ["garbage", "keep"], ["timeout", "timeout"]] + ([["revise_drop"]] if multi else [])
    if not qualifying:
        return [[]]
    out = [["ack"], ["ack_wrong", "ack"], ["ack_wrong", "ack_wrong"], ["abandon"], ["ack_stale", "ack"],
           ["timeout", "ack"], ["ack_missing", "garbage"], ["ack_field", "ack"]]
    if multi:
        out.append(["revise_drop", "ack"])
    return out


def entry_for(e, arm, n):
    return {"seq": n, "model": e["model"], "point": e["id"], "arm": arm, "draw": n,
            "expected": {"digestsSha256": sha256(canon(e["digests"])),
                         "requestSha256": sha256(X.body_bytes(B.initial_request(e))),
                         "proposalSha256": e["proposal"]["sha256"]}}


def fidelity(binary, replays, proto_path):
    proto, _, man, contexts = B.load_protocol(proto_path)
    pts = B.Points(proto, contexts, replays, {B.SETTINGS_ENV: B.REPLAY_URL})
    pool = pts.opener(binary)
    rows, applied = [], []
    try:
        for e in contexts:
            st, mem = pts.settings[e["model"]], B.FixedMemory(pts.mem[e["id"]])
            ref = Reference(binary, pts.run_dir[e["id"]], e)  # the agents' own runs, both arms, one session
            try:
                last_batches = None
                for arm in B.ARMS:
                    for n, kinds in enumerate(cases_for(e, arm), 1):
                        script = Script(kinds)
                        d = B.run_draw(entry_for(e, arm, n), e, st, pool, mem, script, None)
                        rb, rrec, rpay, rex = reference(ref, e, arm, st, mem, kinds)
                        dex = [({"id": f"r{k + 2}", "op": x["op"], **({"args": x["args"]} if x["args"] is not None
                                                                     else {})}, x["answer"])
                               for k, x in enumerate(d["engine"])]
                        eq = {"modelRequests": [X.body_bytes(p) for p in script.payloads] ==
                                               [X.body_bytes(p) for p in rpay],
                              "record": A6.same(json.loads(json.dumps(d["record"])), json.loads(json.dumps(rrec))),
                              "batches": d["batches"] == rb,
                              "engine": canon([(q["op"], q.get("args"), a.get("ok"), a.get("result"), a.get("error"))
                                               for q, a in dex]) ==
                                        canon([(q["op"], q.get("args"), a.get("ok"), a.get("result"), a.get("error"))
                                               for q, a in rex]),
                              "stateUnchanged": d["digestsAfter"] == e["digests"]}
                        again = B.replay_draw(d, e, st, pool, mem)
                        eq["replay"] = B.deterministic(again) == B.deterministic(d)
                        x = B.classify(d, e)
                        rows.append({"point": e["id"], "arm": arm, "script": kinds, "modelCalls": len(d["calls"]),
                                     "unusedReplies": len(script.kinds), "outcome": x["outcome"],
                                     "class": x.get("class"), "preview": (x.get("preview") or {}).get("class"),
                                     "inputs": x["inputs"], "submitted": x["submitted"],
                                     "acknowledgment": {k: x["acknowledgment"][k] for k in ("accepted", "accurate",
                                                                                          "failureCodes", "withheld")}
                                     if arm == "B" else None,
                                     "modelRequestsSha256": sha256(canon(script.payloads)),
                                     "recordSha256": sha256(canon(d["record"])), "engineRequests": len(d["engine"]),
                                     "equal": eq, "faithful": all(eq.values())})
                        if d["batches"]:
                            last_batches = (arm, d["batches"])
                if last_batches:  # the last submitted decision, applied through the agent's harness
                    arm, bs = last_batches
                    acts = [a for b in bs for a in b]
                    p = Pipe(ref.proc)
                    h = A3.Harness(p, p)
                    res = [h.request("apply", {"actions": b}) for b in bs]
                    dig = h.request("digests")["digests"]
                    cont = B.continuation(binary, pts.run_dir[e["id"]], e, acts)
                    applied.append({"point": e["id"], "arm": arm, "actionsKey": sha256(canon(acts)),
                                    "referenceApplied": [r.get("applied") for r in res],
                                    "continuationApplied": [a["applied"] for a in cont["applied"]],
                                    "digestsEqual": dig == cont["digestsAfterDecision"]})
            finally:
                ref.close()
            pool.close()
    finally:
        pool.close()
    return {"schema": SCHEMA, "driver": B.DRIVER, "cases": len(rows),
            "faithful": sum(1 for r in rows if r["faithful"]), "rows": rows, "applied": applied,
            "allFaithful": all(r["faithful"] for r in rows) and all(a["digestsEqual"] and a["referenceApplied"] ==
                                                                   a["continuationApplied"] for a in applied)}


def fidelity_md(doc):
    L = ["# V0.12b Stage B driver: fidelity on the real engine (scripted replies)", "",
         f"{doc['faithful']} of {doc['cases']} cases faithful; every apply check equal: "
         f"{'yes' if doc['allFaithful'] else 'NO'}.", "",
         "Each case: the driver's draw against the agent's own run (its Harness on its own engine session, its "
         "ModelClient payloads, its Evidence), then the draw replayed without the model.", "",
         "| Point | Arm | Script | Calls | Outcome | Class / preview | Inputs | Model requests | Record | Engine | "
         "Batches | Replay |", "|---|---|---|---|---|---|---|---|---|---|---|---|"]
    for r in doc["rows"]:
        q = r["equal"]
        L.append(f"| {r['point']} | {r['arm']} | {' → '.join(r['script']) or '—'} | {r['modelCalls']} | {r['outcome']} | "
                 f"{r['class'] or r['preview']} | {' → '.join(r['inputs']) or '—'} | " +
                 " | ".join("same" if q[k] else "DIFFERS" for k in ("modelRequests", "record", "engine", "batches",
                                                                     "replay")) + " |")
    L += ["", "## Apply at the recorded state: the agent's harness against the analysis continuation", "",
          "| Point | Arm | Decision | Applied (reference / continuation) | Digests |", "|---|---|---|---|---|"]
    for a in doc["applied"]:
        L.append(f"| {a['point']} | {a['arm']} | {a['actionsKey'][:12]} | {a['referenceApplied']} / "
                 f"{a['continuationApplied']} | {'equal' if a['digestsEqual'] else 'DIFFER'} |")
    return "\n".join(L) + "\n"


# --- the mock run ------------------------------------------------------------------------

class Clock:
    """A scripted clock: each model request advances it by `step` seconds
    (less than the 300 s timeout, as a real request), and each server check
    by 10; send times are counted, not read."""

    def __init__(self, step):
        self.t, self.step, self.n = 0.0, step, 0

    def now(self):
        return self.t

    def utc(self):
        self.n += 1
        return f"mock-{self.n:06d}"


def mockrun(binary, replays, proto_path, out, step=290.0):
    proto, proto_sha, man, contexts = B.load_protocol(proto_path)
    clock = Clock(step)
    base_tr = mock_transport("dreamteam-v12b-mock/1")

    def factory(settings):
        def transport(payload):
            clock.t += clock.step
            return base_tr(payload)
        return transport

    def state(base):
        clock.t += 10
        return {"time": clock.utc(), "health": {"status": "healthy"}, "serverVersion": "mock",
                "openapiSha256": "mock", "models": []}
    doc = B.run(proto, proto_sha, man, contexts, out, binary, replays, MOCK_AUTHORIZATION, factory,
                {B.SETTINGS_ENV: B.REPLAY_URL}, state_fn=state, now=clock.now, utc=clock.utc, lock_check=False,
                proto_path=proto_path, client_info={"python": "mock"})
    an = B.analyze(binary, out, replays, proto_path)
    write_json(os.path.join(out, "decisions.json"), an)
    write_json(os.path.join(out, "inference.json"), {"schema": B.ANALYSIS_SCHEMA + "#inference",
                                                     "groups": B.inference(out)})
    with open(os.path.join(out, "decision-tables.md"), "w", encoding="utf-8") as f:
        f.write(B.tables(an))
    return doc, an


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    for name in ("fidelity", "mockrun"):
        q = sub.add_parser(name)
        q.add_argument("--binary", required=True)
        q.add_argument("--replays", required=True)
        q.add_argument("--protocol", required=True)
        q.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    try:
        if a.cmd == "fidelity":
            doc = fidelity(a.binary, a.replays, a.protocol)
            os.makedirs(a.out, exist_ok=True)
            write_json(os.path.join(a.out, "fidelity.json"), doc)
            with open(os.path.join(a.out, "fidelity.md"), "w", encoding="utf-8") as f:
                f.write(fidelity_md(doc))
            print(f"stage B fidelity: {doc['faithful']} of {doc['cases']} cases faithful; all checks "
                  f"{'pass' if doc['allFaithful'] else 'DO NOT pass'}")
            return 0 if doc["allFaithful"] else 1
        doc, an = mockrun(a.binary, a.replays, a.protocol, a.out)
        print(f"stage B mock run: {doc['status']}; {doc['drawsCompleted']} of {doc['drawsScheduled']} draws, stopped: "
              f"{(doc['stopped'] or {}).get('code')}; analysis {an['status']}, replay checked {an['replayChecked']}")
        return 0
    except B.ERRORS as e:
        print(f"stage B fidelity: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
