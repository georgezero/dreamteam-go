#!/usr/bin/env python3
"""The V0.11b experiment: three decision workflows from one recorded
proposal (dreamteam.shared-prefix/v4; docs/benchmark/v11b-dev/SPEC.md and
PLAN.md).

At each selected branch point every arm starts from the same recorded state,
the same recorded conversation and the same recorded initial staffing
proposal, and then runs a whole decision workflow with live model calls:
* A, llm-agent 3: the proposal validated; a refusal answered with version
  3's retry message; retries within maxRetries; no preview;
* B, llm-agent 5 with the preview disabled (experiment configuration
  v11b-B-validation-only): validation first, the exact refusal feedback,
  revision or abandonment; no preview;
* C, llm-agent 5 (llm-agent-workflow/v1): B's steps, plus the selective
  preview of the first fully valid staffing proposal and its review.
Only a fully validated proposal is submitted; otherwise nothing.

The driver (run_arm) follows each agent's own loop from the proposal on,
with the agents' own functions (parser, batching, retry message, feedback
message, preview request, eligibility and preview message); its messages
and decisions are tested against the agents' ask() (test_workflow_experiment.py).

    python3 -I -B analysis/repeatability/workflow_experiment.py points --binary ./dt --replays DIR \\
        --live DIR [--live DIR] --out DIR
    python3 -I -B analysis/repeatability/workflow_experiment.py points --binary ./dt --replays DIR --check DIR
    python3 -I -B analysis/repeatability/workflow_experiment.py schedule --protocol PROTOCOL --out DIR
    DREAMTEAM_LLM_BASE_URL=... python3 -I -B analysis/repeatability/workflow_experiment.py run \\
        --binary ./dt --replays DIR --protocol PROTOCOL --out DIR
    python3 -I -B analysis/repeatability/workflow_experiment.py analyze --binary ./dt --replays DIR --dir DIR --out DIR
"""

import argparse
import collections
import json
import math
import os
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import feedback_experiment as F  # noqa: E402  (shared-prefix/v3, unchanged: its inventory)
X, A3, A4, S, FE, RR = F.X, F.A3, F.A4, F.S, F.FE, F.RR
sys.path.insert(0, os.path.join(X.ROOT, "agents", "llm"))
import llm_agent5 as A5  # noqa: E402  (unchanged: the workflow's functions)

sha256, canon, write_json, load_jsonl = X.sha256, X.canon, X.write_json, X.load_jsonl
VERSION = "shared-prefix/v4"
PROTOCOL_SCHEMA = "dreamteam.shared-prefix/v4"
POINTS_SCHEMA = "dreamteam.shared-prefix-points/v4"
SCHEDULE_SCHEMA = "dreamteam.shared-prefix-schedule/v4"
RUN_SCHEMA = "dreamteam.shared-prefix-run/v4"
ANALYSIS_SCHEMA = "dreamteam.shared-prefix-analysis/v4"
ANALYZER = "shared-prefix-analysis 4"
ARMS = ("A", "B", "C")
ARM_NAMES = {"A": "llm-agent 3 (validation and retry)",
             "B": "llm-agent 5, preview disabled (v11b-B-validation-only)",
             "C": "llm-agent 5 (llm-agent-workflow/v1)"}
B_CONFIG = "v11b-B-validation-only"
SELECT_RULE = "dreamteam-v11b-select/1"
ORDER_RULE = "dreamteam-v11b-order/1"
GEMMA, QWEN = F.GEMMA, F.QWEN
DRAWS = {GEMMA: 10, QWEN: 20}
MAX_RETRIES = 1        # the frozen settings' (docs/benchmark/v05c-dev/llm-*.json)
ALLOW_DETAILS = True   # likewise
MAX_PER_RECORDING = 2
TIMING = {"lead_too_short", "notice_too_short", "past_date", "not_monday", "commitment_too_short"}
ACCEPTED_COVER = ("credit_supported", "retained_obligation", "termination", "allocation")
OUTCOMES = ("kept", "revised", "abandoned", "refused", "invalid")


class ExperimentError(Exception):
    pass


# --- the workflows ---------------------------------------------------------

class Env:
    """A decision point's engine side and inputs: the harness (validate,
    preview, digests, observe), the observation, what compact left out, the
    catalog and the clock, and the prompt settings' bound."""

    def __init__(self, h, obs, omitted, catalog, clock, max_prompt_chars):
        self.h, self.obs, self.omitted, self.catalog, self.clock = h, obs, omitted, catalog, clock
        self.types = {t["type"] for t in catalog.get("types", [])}
        self.pset = {"maxPromptChars": max_prompt_chars}


def run_arm(arm, messages, proposal, env, model, failures=0, details_open=ALLOW_DETAILS, max_retries=MAX_RETRIES):
    """One arm's workflow from the recorded proposal: the shared first
    reply, the model's answer to messages. model(messages) ->
    (call, content or None). Returns the trace: every attempt, the batches
    submitted and the outcome."""
    msgs = list(messages)
    attempts, used = [], None
    previewed = arm != "C"
    content, call = proposal, None
    while True:
        at = {"source": "shared" if call is None else "model", "call": call, "parse": None, "reason": None,
              "actions": None, "details": None, "validation": None, "refusals": [], "protocolError": None,
              "preview": None, "error": None, "used": False}
        asked, review, batch_no = None, None, None
        if content is None:
            at["error"] = (call or {}).get("error") or "no_content"
        else:
            try:
                try:
                    d = A3.parse_decision(content, env.types, details_open)
                except ValueError as e:
                    at["parse"] = str(e).split(":", 1)[0]
                    raise
                at["parse"], at["reason"], at["actions"] = "ok", d["reason"], d["actions"]
                if "details" in d:
                    asked = at["details"] = d["details"]
                elif d["actions"]:
                    at["validation"], refused = [], []
                    for batch_no, (batch, idx) in enumerate(A3.split(d["actions"])):
                        v = env.h.request("validate", {"actions": batch})
                        at["validation"].append({"actions": batch, "indices": idx, "answer": v})
                        if not v.get("valid"):
                            refused += A3.refusals(v, idx, len(attempts) + 1)
                    if refused:
                        at["refusals"] = refused
                        raise ValueError("refused: " + "; ".join(
                            f"action {r['action']}: {r['code']}: {r['message']}" for r in refused))
                    if not previewed and any(a.get("type") in A3.STAFFING for a in d["actions"]):
                        previewed = True
                        at["preview"], review = A5.request_preview(env.h, d["actions"], env.clock, env.pset, env.obs)
                else:
                    at["validation"] = []
            except A3.ProtocolError as e:
                at["error"] = f"invalid_action: validate answered {e.code}: {e.message}"
                at["protocolError"] = {"batch": batch_no, "code": e.code}
            except ValueError as e:
                at["error"] = str(e)
        attempts.append(at)
        if asked is not None:
            details_open = False
            text = A3.details_text(env.h, asked, env.obs, env.omitted, env.catalog, env.pset["maxPromptChars"] // 4)
            msgs = msgs + [{"role": "assistant", "content": content}, {"role": "user", "content": text}]
            kind = "details"
        elif review is not None:
            details_open = False
            msgs = msgs + [{"role": "assistant", "content": content}, {"role": "user", "content": review}]
            kind = "preview"
        elif at["error"] is None:
            at["used"], used = True, at
            break
        else:
            failures += 1
            if failures > max_retries:
                break
            if at["refusals"] and arm in ("B", "C"):
                text = A5.feedback_text(A5.refused_items(at["validation"]))
                details_open = False
                kind = "validation"
            elif content is not None:
                text = A3.retry_text(at["error"])
                kind = "retry"
            else:
                text, kind = None, "resend"
            if text is not None:
                msgs = msgs + [{"role": "assistant", "content": content}, {"role": "user", "content": text}]
        at["next"] = kind
        call, content = model(msgs)
    batches = [b for b, _ in A3.split(used["actions"])] if used and used["actions"] else []
    out = {"arm": arm, "attempts": attempts, "batches": batches,
           "calls": sum(1 for a in attempts if a["source"] == "model"), "failures": failures}
    if arm in ("B", "C"):  # llm-agent 5's check: validation and preview changed no state
        unchanged = env.h.request("digests").get("digests") == env.clock["digests"]
        out["digestsUnchanged"] = unchanged
        if not unchanged:
            out["batches"], used = [], None
    out["used"] = attempts.index(used) if used else None
    return out


def classify(trace, proposal_actions):
    """(outcome, final): kept, revised or abandoned against the proposal
    (llm-agent 4's rule) when a decision was used; otherwise refused (the
    last attempt was refused) or invalid (unreadable, failed or no reply)."""
    if trace["used"] is not None:
        acts = trace["attempts"][trace["used"]]["actions"]
        return A4.response_of(proposal_actions, acts), ("accepted" if acts else "no_action")
    last = trace["attempts"][-1]
    return ("refused" if last["refusals"] else "invalid"), "none"


def preview_of(trace):
    a = next((x for x in trace["attempts"] if x.get("preview")), None)
    if a is None:
        return None
    pv = a["preview"]
    after = trace["attempts"][trace["attempts"].index(a) + 1:] if pv["status"] == "shown" else []
    first = next((x for x in after if x["parse"] == "ok" and x["details"] is None), None)
    return {"status": pv["status"], "reasons": (pv.get("eligibility") or {}).get("reasons"),
            "facts": (pv.get("eligibility") or {}).get("facts"),
            "response": A4.response_of(a["actions"], first["actions"]) if first else
            ("no_reply" if pv["status"] == "shown" else None)}


# --- the branch points -----------------------------------------------------------

def inventory_rows(root=X.ROOT):
    """The V0.10c inventory (committed, checked again by its own CI step),
    with each candidate's prefix: attempts before the proposal, failures
    among them, whether one was refused, and prior exposure in V0.10b or
    V0.10c."""
    with open(os.path.join(root, "docs/benchmark/v10c-dev/points/inventory.json"), encoding="utf-8") as f:
        inv = json.load(f)
    seen = {}
    for name in ("v10b-dev", "v10c-dev"):
        with open(os.path.join(root, "docs/benchmark", name, "points", "contexts.jsonl"), encoding="utf-8") as f:
            for l in f:
                seen.setdefault(json.loads(l)["contextKey"], []).append(name[:5].replace("-", "").upper())
    rows = []
    for r in inv["contexts"]:
        rec = S.recording_point(root, r["recording"], r["point"])
        prior = rec["attempts"][:r["attempt"] - 1]
        x = dict(r)
        x.update({"failuresBefore": sum(1 for a in prior if a.get("error") is not None),
                  "refusedBefore": any(a.get("refusals") for a in prior),
                  "detailsBefore": any(a.get("detailsRequested") for a in prior),
                  "priorExposure": sorted(seen.get(r["contextKey"], []))})
        if x["refusedBefore"]:
            x.update(eligible=False, exclusion="a refusal before the proposal: the arms' conversations would differ")
        rows.append(x)
    return inv, rows


def tags(r):
    t = F.tags(r)
    if "terminate" in r["staffingTypes"]:
        t.append("termination")
    if "set_reading_allocation" in r["staffingTypes"]:
        t.append("allocation")
    codes = {x.split(":", 1)[1] for x in r["refusals"]}
    if codes & TIMING:
        t.append("timing")
    return t


def select(rows):
    """Gemma: every distinct eligible context. Qwen accepted (5): for each
    category of ACCEPTED_COVER the context with it and the largest stake not
    yet taken, then the largest stakes. Qwen refused (5): insufficient funds
    (2), timing (2), any other refusal (1), each by stake. Ties by SHA-256
    ('dreamteam-v11b-select/1|<recording>|<point>'); a context once; at most
    2 points from one recording."""
    chosen, seen, per = [], set(), collections.Counter()

    def tie(r):
        return sha256(f"{SELECT_RULE}|{r['recording']}|{r['point']}")

    def free(r):
        return r["eligible"] and r["contextKey"] not in seen and per[r["recording"]] < MAX_PER_RECORDING

    def take(r):
        seen.add(r["contextKey"])
        per[r["recording"]] += 1
        chosen.append(r)

    def best(cands):
        c = sorted((r for r in cands if free(r)), key=lambda r: (-F.stake(r), tie(r)))
        return c[0] if c else None
    for st in ("accepted", "refused"):
        for r in sorted((r for r in rows if r["model"] == GEMMA and r["validation"] == st),
                        key=lambda r: (not r["shownPreview"], tie(r))):
            if r["eligible"] and r["contextKey"] not in seen:
                take(r)
    acc = [r for r in rows if r["model"] == QWEN and r["validation"] == "accepted"]
    for cat in ACCEPTED_COVER:
        r = best([x for x in acc if cat in tags(x)])
        if r:
            take(r)
    while sum(1 for r in chosen if r["model"] == QWEN and r["validation"] == "accepted") < 5:
        r = best(acc)
        if r is None:
            raise ExperimentError("too few accepted Qwen contexts")
        take(r)
    ref = [r for r in rows if r["model"] == QWEN and r["validation"] == "refused"]
    for name, test, n in (("insufficient_funds", lambda r: "insufficient_funds" in tags(r), 2),
                          ("timing", lambda r: "timing" in tags(r) and "insufficient_funds" not in tags(r), 2),
                          ("other", lambda r: not ({"timing", "insufficient_funds"} & set(tags(r))), 1)):
        for _ in range(n):
            r = best([x for x in ref if test(x)])
            if r is None:
                raise ExperimentError(f"too few refused Qwen contexts: {name}")
            r = dict(r, category=name)
            take(r)
    for r in chosen:
        r["id"] = F.point_id(r)
        r.setdefault("category", next((c for c in ACCEPTED_COVER if c in tags(r)), "stake")
                     if r["validation"] == "accepted" else
                     ("insufficient_funds" if "insufficient_funds" in tags(r) else
                      "timing" if "timing" in tags(r) else "other"))
    return chosen


def context(r, rec, call):
    """A selected point's context: the recorded request for the proposal and
    the proposal, every message checked against the recording."""
    at = rec["attempts"][r["attempt"] - 1]
    req, proposal = call["request"], call["content"]
    if [(m["role"], sha256(m["content"])) for m in req["messages"]] != S.chain(rec, r["attempt"]):
        raise ExperimentError(f"{r['id']}: the recorded request's messages differ from the recording's hashes")
    if sha256(proposal or "") != at["output"]["sha256"]:
        raise ExperimentError(f"{r['id']}: the recorded proposal is not the recording's output")
    if req.get("temperature") != 0 or set(req) != {"model", "messages", "temperature", "max_tokens", "response_format"} \
            or req.get("model") != S.MODELS[r["model"]]:
        raise ExperimentError(f"{r['id']}: the request's settings are not the recorded agent's")
    e = {k: r[k] for k in ("id", "model", "validation", "refusals", "category", "recording", "recordingSha256", "point",
                           "attempt", "attemptId", "contextKey", "staffingTypes", "priorExposure", "failuresBefore")}
    e.update({"tags": tags(r), "stakeCents": F.stake(r), "digests": rec["digests"],
              "maxPromptChars": rec["promptSettings"]["maxPromptChars"],
              "detailsOpen": ALLOW_DETAILS and not any(a.get("detailsRequested") for a in rec["attempts"][:r["attempt"] - 1]),
              "request": req, "proposal": {"content": proposal, "sha256": sha256(proposal), "actions": at["actions"]}})
    return e


def check_context(e, rec):
    at = rec["attempts"][e["attempt"] - 1]
    if [(m["role"], sha256(m["content"])) for m in e["request"]["messages"]] != S.chain(rec, e["attempt"]):
        raise ExperimentError(f"{e['id']}: the request's messages differ from the recording's hashes")
    if sha256(e["proposal"]["content"]) != at["output"]["sha256"] or e["proposal"]["actions"] != at["actions"]:
        raise ExperimentError(f"{e['id']}: the proposal is not the recording's")
    if e["digests"] != rec["digests"]:
        raise ExperimentError(f"{e['id']}: the digests are not the recording's")


def points_manifest(rows, chosen, contexts_sha):
    return {"schema": POINTS_SCHEMA, "version": VERSION, "rule": SELECT_RULE, "acceptedCover": list(ACCEPTED_COVER),
            "timing": sorted(TIMING), "maxPerRecording": MAX_PER_RECORDING,
            "inventory": "docs/benchmark/v10c-dev/points/inventory.json",
            "candidates": [{k: r.get(k) for k in ("recording", "point", "model", "validation", "refusals", "eligible",
                                                  "exclusion", "failuresBefore", "refusedBefore", "priorExposure",
                                                  "contextKey")} | {"tags": tags(r), "stakeCents": F.stake(r)}
                           for r in rows],
            "selected": [{k: r[k] for k in ("id", "model", "validation", "category", "refusals", "recording", "point",
                                            "contextKey", "priorExposure")} | {"tags": tags(r), "stakeCents": F.stake(r)}
                         for r in chosen],
            "contextsFileSha256": contexts_sha}


def make_points(live, out, root=X.ROOT):
    inv, rows = inventory_rows(root)
    chosen = select(rows)
    lr = S.live_runs(live)
    contexts = []
    for r in chosen:
        rec = S.recording_point(root, r["recording"], r["point"])
        calls = [x for x in load_jsonl(os.path.join(lr[r["recordingSha256"]], "model.jsonl"))
                 if x.get("point") == r["point"] and x.get("attempt") == r["attempt"]]
        if len(calls) != 1:
            raise ExperimentError(f"{r['id']}: {len(calls)} recorded calls")
        e = context(r, rec, calls[0])
        check_context(e, rec)
        contexts.append(e)
    os.makedirs(out, exist_ok=True)
    body = "".join(json.dumps(e, ensure_ascii=False) + "\n" for e in contexts).encode("utf-8")
    with open(os.path.join(out, "contexts.jsonl"), "wb") as f:
        f.write(body)
    m = points_manifest(rows, chosen, sha256(body))
    write_json(os.path.join(out, "manifest.json"), m)
    return m


def load_points(d):
    with open(os.path.join(d, "manifest.json"), encoding="utf-8") as f:
        m = json.load(f)
    with open(os.path.join(d, "contexts.jsonl"), "rb") as f:
        body = f.read()
    if sha256(body) != m["contextsFileSha256"]:
        raise ExperimentError("contexts.jsonl is not the manifest's")
    return m, [json.loads(l) for l in body.decode("utf-8").splitlines() if l.strip()]


def check_points(d, root=X.ROOT):
    m, contexts = load_points(d)
    _, rows = inventory_rows(root)
    chosen = select(rows)
    if [r["id"] for r in chosen] != [e["id"] for e in contexts]:
        raise ExperimentError("the selection made again differs from the committed points")
    for e in contexts:
        check_context(e, S.recording_point(root, e["recording"], e["point"]))
    if json.loads(json.dumps(points_manifest(rows, chosen, m["contextsFileSha256"]))) != m:
        raise ExperimentError("the manifest is not the one the points make")
    return len(contexts)


# --- the engine at a point, and the model -------------------------------------------

class EngineHarness:
    """The agent's harness interface (request(op, args) -> result, raising
    the agents' ProtocolError) over an engine session at the recorded point;
    every request and answer logged."""

    def __init__(self, ses):
        self.ses, self.log = ses, []

    def request(self, op, args=None):
        a = self.ses.request("v11b", op, args)
        self.log.append({"op": op, "args": args, "answer": a})
        if not a.get("ok"):
            raise A3.ProtocolError(op, a.get("error") or {})
        return a["result"]


def open_point(binary, run_dir, e):
    """An engine session at the point, and its Env (the compact observation,
    what it leaves out, the catalog, the clock: llm-agent 3's look())."""
    ses = X.open_session(binary, run_dir)
    ses.to_point(e["digests"])
    h = EngineHarness(ses)
    clock = h.request("digests")
    if clock["digests"] != e["digests"]:
        raise ExperimentError(f"{e['id']}: not the recorded state")
    o = h.request("observe", {"compact": True, "recentLog": 0})
    env = Env(h, o["observation"], o.get("omitted") or {}, h.request("actions"), clock, e["maxPromptChars"])
    h.log = []
    return ses, env


def body_of(e, messages):
    return {k: (messages if k == "messages" else v) for k, v in e["request"].items()}


def start_messages(e):
    """The recorded request's messages: the proposal answers the last."""
    return list(e["request"]["messages"])


def live_model(base, e, timeout, sink):
    """The model, live: one request, its response recorded in sink."""
    def model(messages):
        data = X.body_bytes(body_of(e, messages))
        t = RR.utc()
        status, body, err, lat = RR.post(base, data, timeout)
        f = RR.response_fields(body)
        content = f.get("content") if (err is None and status == 200 and f.get("parsedBody")) else None
        if err is None and content is None:
            err = f"http_status: HTTP {status}" if status != 200 else "no_content: the response has no content"
        rec = {"requestSha256": sha256(data), "sent": t, "status": status, "error": err,
               "latencyMs": round(lat * 1000), "body": body, "responseSha256": sha256(body or ""),
               **{k: f.get(k) for k in ("content", "finishReason", "usage", "servedModel")}}
        sink.append(rec)
        return {"requestSha256": rec["requestSha256"], "error": err}, content
    return model


def replay_model(e, responses):
    """The model, replayed: the recorded responses in order, each checked to
    answer the same request."""
    it = iter(responses)

    def model(messages):
        r = next(it, None)
        data = X.body_bytes(body_of(e, messages))
        if r is None or r["requestSha256"] != sha256(data):
            raise ExperimentError(f"{e['id']}: the workflow sent a request the run did not record")
        if r.get("responseSha256") != sha256(r.get("body") or ""):
            raise ExperimentError(f"{e['id']}: a response is not the recorded one")
        content = r.get("content") if (r.get("error") is None) else None
        return {"requestSha256": r["requestSha256"], "error": r.get("error")}, content
    return model


def strip_trace(t):
    """The deterministic trace: no live metadata."""
    return json.loads(json.dumps(t))


# --- the schedule and the live run -----------------------------------------------------

def load_protocol(path):
    with open(path, "rb") as f:
        raw = f.read()
    proto = json.loads(raw)
    if proto.get("schema") != PROTOCOL_SCHEMA:
        raise ExperimentError(f"{path} is not a {PROTOCOL_SCHEMA} protocol")
    man, contexts = load_points(os.path.join(os.path.dirname(os.path.abspath(path)), proto["points"]))
    return proto, sha256(raw), man, contexts


def schedule(proto, contexts):
    """Draws, one workflow at a time. Per model, its draws as rounds; each
    round has every point and arm of the model once, in the order of
    SHA-256('<orderRule>|<model>|<round>|<point>|<arm>')."""
    rule, out = proto["orderRule"], []
    for m in proto["modelOrder"]:
        ids = [e["id"] for e in contexts if e["model"] == m]
        for r in range(1, proto["drawsPerArm"][m] + 1):
            for p, a in sorted(((p, a) for p in ids for a in ARMS),
                               key=lambda x: sha256(f"{rule}|{m}|{r}|{x[0]}|{x[1]}")):
                out.append({"seq": len(out) + 1, "model": m, "point": p, "arm": a, "draw": r})
    return out


def schedule_doc(proto, proto_sha, man, sched):
    return {"schema": SCHEDULE_SCHEMA, "protocol": proto["id"], "protocolSha256": proto_sha,
            "contextsFileSha256": man["contextsFileSha256"], "draws": len(sched), "concurrency": 1,
            "maxModelCallsPerDraw": proto["maxModelCallsPerDraw"], "schedule": sched}


def run(proto, proto_sha, man, contexts, out, base, binary, replays, state_fn=RR.server_state, model_fn=live_model,
        now=time.monotonic):
    """The live run: every draw's workflow, one model request at a time,
    each draw recorded as it ends; no retry, nothing dropped."""
    sched = schedule(proto, contexts)
    by_id = {e["id"]: e for e in contexts}
    runs = X.replay_runs(replays)
    live = os.path.join(out, "live")
    os.makedirs(live, exist_ok=True)
    stop = proto["stop"]
    before = state_fn(base)
    write_json(os.path.join(live, "server-before.json"), before)
    sig = X.server_signature(before)
    t0, started, fails, done, stopped, checks, prev, calls = now(), RR.utc(), 0, 0, None, [], None, 0
    sessions = {}
    if (before.get("health") or {}).get("status") != "healthy":
        stopped = "health: the server did not answer healthy before the run"
    try:
        with open(os.path.join(live, "draws.jsonl"), "w", encoding="utf-8") as f:
            for r in sched if stopped is None else []:
                if (r["model"], r["draw"]) != prev:
                    prev = (r["model"], r["draw"])
                    st = state_fn(base)
                    checks.append({"seq": r["seq"], "model": r["model"], "draw": r["draw"], "time": st.get("time"),
                                   "health": st.get("health"), "sameConfiguration": X.server_signature(st) == sig})
                    if (st.get("health") or {}).get("status") != "healthy":
                        stopped = f"health: the server did not answer healthy before draw {r['seq']}"
                        break
                    if X.server_signature(st) != sig:
                        stopped = f"server configuration changed before draw {r['seq']}"
                        break
                if now() - t0 > stop["wallClockSeconds"]:
                    stopped = "wall clock"
                    break
                e = by_id[r["point"]]
                if e["id"] not in sessions:
                    sessions[e["id"]] = open_point(binary, runs[os.path.basename(e["recording"])], e)
                ses, env = sessions[e["id"]]
                env.h.log = []
                sink = []
                trace = run_arm(r["arm"], start_messages(e), e["proposal"]["content"], env,
                                model_fn(base, e, proto["request"]["timeoutSeconds"], sink),
                                failures=e["failuresBefore"], details_open=e["detailsOpen"])
                rec = dict(r, calls=sink, trace=strip_trace(trace), engine=env.h.log)
                f.write(json.dumps(rec, sort_keys=True, ensure_ascii=False) + "\n")
                f.flush()
                done += 1
                calls += len(sink)
                for c in sink:
                    fails = fails + 1 if c["error"] and c["status"] is None else 0
                if fails >= stop["consecutiveTransportFailures"]:
                    stopped = f"{fails} consecutive transport failures"
                    break
    finally:
        for ses, _ in sessions.values():
            ses.close()
    after = state_fn(base)
    write_json(os.path.join(live, "server-after.json"), after)
    doc = {"schema": RUN_SCHEMA, "protocol": proto["id"], "protocolSha256": proto_sha,
           "contextsFileSha256": man["contextsFileSha256"], "started": started, "ended": RR.utc(),
           "drawsScheduled": len(sched), "drawsRun": done, "modelCalls": calls, "stopped": stopped,
           "serverChecks": checks, "sameConfigurationAfter": X.server_signature(after) == sig,
           "endpoint": "from " + proto["endpointEnv"] + " (not recorded)", "client": {"python": sys.version.split()[0]}}
    write_json(os.path.join(live, "run.json"), doc)
    return doc


# --- the analysis ----------------------------------------------------------------------

def replay_draws(binary, replays, proto, contexts, rows):
    """Every recorded draw run again without the model: the same workflow,
    the recorded responses, a fresh engine session at the point. The trace
    and the engine answers must be the recorded ones."""
    sched = schedule(proto, contexts)
    if [(r["seq"], r["point"], r["arm"], r["draw"]) for r in rows] != \
            [(s["seq"], s["point"], s["arm"], s["draw"]) for s in sched[:len(rows)]]:
        raise ExperimentError("the recorded draws are not the schedule's, in its order")
    runs, by_id = X.replay_runs(replays), {e["id"]: e for e in contexts}
    out, checked = [], {"draws": 0, "modelCalls": 0, "engineRequests": 0}
    sessions = {}
    try:
        for r in rows:
            e = by_id[r["point"]]
            if e["id"] not in sessions:
                sessions[e["id"]] = open_point(binary, runs[os.path.basename(e["recording"])], e)
            ses, env = sessions[e["id"]]
            env.h.log = []
            trace = run_arm(r["arm"], start_messages(e), e["proposal"]["content"], env, replay_model(e, r["calls"]),
                            failures=e["failuresBefore"], details_open=e["detailsOpen"])
            if strip_trace(trace) != r["trace"] or env.h.log != r["engine"]:
                raise ExperimentError(f"draw {r['seq']}: the replayed workflow differs from the recorded one")
            if trace["calls"] != len(r["calls"]):
                raise ExperimentError(f"draw {r['seq']}: recorded calls the workflow did not make")
            checked["draws"] += 1
            checked["modelCalls"] += len(r["calls"])
            checked["engineRequests"] += len(env.h.log)
            out.append((r, trace))
    finally:
        for ses, _ in sessions.values():
            ses.close()
    return out, checked


def analyze(binary, d, replays):
    proto, proto_sha, man, contexts = load_protocol(os.path.join(d, "protocol.json"))
    with open(os.path.join(d, "live", "run.json"), encoding="utf-8") as f:
        run_doc = json.load(f)
    if run_doc["protocolSha256"] != proto_sha or run_doc["contextsFileSha256"] != man["contextsFileSha256"]:
        raise ExperimentError("the live run is not of this protocol and these points")
    rows = load_jsonl(os.path.join(d, "live", "draws.jsonl"))
    replayed, checked = replay_draws(binary, replays, proto, contexts, rows)
    runs = X.replay_runs(replays)
    by_point = collections.defaultdict(list)
    for r, t in replayed:
        by_point[r["point"]].append((r, t))
    points, draws = [], []
    for e in contexts:
        prop = e["proposal"]["actions"]
        lists = {canon(prop): prop, canon([]): []}
        for r, t in by_point[e["id"]]:
            if t["used"] is not None:
                acts = t["attempts"][t["used"]]["actions"]
                lists.setdefault(canon(acts), acts)
        ev = {k: X.decision_evidence(binary, runs[os.path.basename(e["recording"])], e, v)
              for k, v in sorted(lists.items())}
        pfe = ev[canon(prop)]["financialEvidence"]
        decisions = []
        for k in sorted(lists):
            x = ev[k]
            decisions.append({"key": sha256(k), "actions": lists[k], "isProposal": k == canon(prop),
                              "displacedBookings": F.displaced(x["records"]),
                              **{f: x[f] for f in ("validation", "outcome", "applied", "validateApplyAgree",
                                                   "digestsAfterDecision", "suffix", "observedEnd",
                                                   "financialEvidence", "records")}})
        for r, t in by_point[e["id"]]:
            o, fin = classify(t, prop)
            acts = t["attempts"][t["used"]]["actions"] if t["used"] is not None else None
            k = canon(acts) if acts is not None else canon([])
            fe = ev[k]["financialEvidence"]
            pv = preview_of(t)
            draws.append({"seq": r["seq"], "point": e["id"], "model": e["model"], "arm": r["arm"], "draw": r["draw"],
                          "outcome": o, "final": fin, "calls": t["calls"],
                          "retries": sum(1 for a in t["attempts"] if a.get("next") in ("retry", "validation", "resend")),
                          "inputs": [a.get("next") for a in t["attempts"] if a.get("next")],
                          "preview": pv, "actionsKey": sha256(k) if acts is not None else None,
                          "commitmentCents": fe["commitmentCents"] if acts is not None else 0,
                          "creditSupported": bool(fe["creditSupported"]) if acts is not None else False,
                          "commitmentChangeCents": (fe["commitmentCents"] if acts is not None else 0) - pfe["commitmentCents"],
                          "changes": X.changes(prop, acts) if o == "revised" else None,
                          "rawSha256": [sha256(c["content"]) if c.get("content") is not None else None
                                        for c in r["calls"]]})
        fo = FE.funding_obs(_obs_at(binary, runs, e)) or {}
        points.append({"id": e["id"], "model": e["model"], "validation": e["validation"], "category": e["category"],
                       "refusals": e["refusals"], "tags": e["tags"], "stakeCents": e["stakeCents"],
                       "priorExposure": e["priorExposure"], "recording": e["recording"], "point": e["point"],
                       "proposalSha256": e["proposal"]["sha256"], "proposal": prop,
                       "headroomWithoutCreditCents": fo.get("headroomWithoutCreditCents"),
                       "fundingHeadroomCents": fo.get("fundingHeadroomCents"), "decisions": decisions})
    return {"schema": ANALYSIS_SCHEMA, "analyzer": ANALYZER, "protocol": proto["id"], "protocolSha256": proto_sha,
            "contextsFileSha256": man["contextsFileSha256"], "drawsScheduled": len(schedule(proto, contexts)),
            "drawsAnalyzed": len(rows), "stopped": run_doc["stopped"], "replayChecked": checked,
            "points": points, "draws": draws, "summary": summarize(contexts, draws, proto), "notes": NOTES}


def _obs_at(binary, runs, e):
    ses = X.open_session(binary, runs[os.path.basename(e["recording"])])
    try:
        obs, _ = ses.to_point(e["digests"])
    finally:
        ses.close()
    return obs


NOTES = [
    "Per point and arm, every scheduled draw is in the denominator. A draw is one whole workflow: the shared "
    "proposal, then whatever model calls its arm makes (none, for A and B at a valid proposal).",
    "Intervals are Wilson 95% score intervals for a proportion and Newcombe hybrid score intervals for a difference. "
    "They are descriptive: each treats a point's draws as independent repetitions of one workflow, which V0.10a does "
    "not establish. At a valid proposal A and B make no model call, so their draws repeat one deterministic result.",
    "The primary comparison is C − B; the secondary B − A.",
    "Financial evidence v1's definitions, unchanged, at the recorded state; a refused or failed workflow submits "
    "nothing. Long-run outcomes under the fixed suffix are a mechanics check and are not attributed to any arm.",
]
COMPARISONS = (("C", "B"), ("B", "A"), ("C", "A"))


def arm_summary(ds, n):
    cnt = collections.Counter(x["outcome"] for x in ds)
    fin = collections.Counter(x["final"] for x in ds)
    com = sorted(x["commitmentCents"] for x in ds)
    pv = collections.Counter((x["preview"] or {}).get("status") or "none" for x in ds)
    resp = collections.Counter((x["preview"] or {}).get("response") or "none" for x in ds
                               if (x["preview"] or {}).get("status") == "shown")
    return {"draws": n, "observed": len(ds), "notRun": n - len(ds),
            "outcomes": {o: {"count": cnt.get(o, 0), "wilson95": X.wilson(cnt.get(o, 0), n)} for o in OUTCOMES},
            "final": {k: {"count": fin.get(k, 0), "wilson95": X.wilson(fin.get(k, 0), n)}
                      for k in ("accepted", "no_action", "none")},
            "modelCalls": sum(x["calls"] for x in ds), "retries": sum(x["retries"] for x in ds),
            "preview": dict(sorted(pv.items())), "previewResponse": dict(sorted(resp.items())),
            "creditSupported": sum(1 for x in ds if x["creditSupported"]),
            "commitmentCents": {"min": com[0] if com else None, "median": com[len(com) // 2] if com else None,
                                "max": com[-1] if com else None},
            "distinctDecisions": len({x["actionsKey"] or "none:" + x["outcome"] for x in ds}),
            "actionContent": dict(sorted(collections.Counter(x["actionsKey"] or "none" for x in ds).items()))}


def summarize(contexts, draws, proto):
    out = {"points": [], "pooled": {}}
    for e in contexts:
        n = proto["drawsPerArm"][e["model"]]
        row = {"id": e["id"], "model": e["model"], "validation": e["validation"], "category": e["category"],
               "draws": n, "arms": {a: arm_summary([x for x in draws if x["point"] == e["id"] and x["arm"] == a], n)
                                    for a in ARMS}, "comparisons": {}}
        for t, c in COMPARISONS:
            T, C = row["arms"][t], row["arms"][c]
            row["comparisons"][f"{t}-{c}"] = {
                "outcomes": {o: X.newcombe(T["outcomes"][o]["count"], n, C["outcomes"][o]["count"], n) for o in OUTCOMES},
                "final": {k: X.newcombe(T["final"][k]["count"], n, C["final"][k]["count"], n)
                          for k in ("accepted", "no_action", "none")},
                "creditSupported": X.newcombe(T["creditSupported"], n, C["creditSupported"], n),
                "sameActionContent": T["actionContent"] == C["actionContent"]}
        out["points"].append(row)
    for m in (GEMMA, QWEN, "all"):
        for st in ("accepted", "refused", "all"):
            rows = [r for r in out["points"] if m in ("all", r["model"]) and st in ("all", r["validation"])]
            if not rows:
                continue
            k = f"{m}/{st}"
            out["pooled"][k] = {"points": len(rows), "interval": "none: the points are clusters"}
            for t, c in COMPARISONS:
                key = f"{t}-{c}"
                diffs = {o: [r["comparisons"][key]["outcomes"][o]["difference"] for r in rows] for o in OUTCOMES}
                out["pooled"][k][key] = {
                    "perPointDifferences": diffs,
                    "unweightedMean": {o: round(math.fsum(v) / len(v), 4) for o, v in diffs.items()},  # fsum: identical on every Python version
                    "pointsSameActionContent": sum(1 for r in rows if r["comparisons"][key]["sameActionContent"])}
    return out


def inference(d):
    rows = load_jsonl(os.path.join(d, "live", "draws.jsonl"))
    g = collections.defaultdict(list)
    for r in rows:
        g[(r["point"], r["arm"])].extend(r["calls"])
    out = {}
    for (p, a), cs in sorted(g.items()):
        lat = sorted(c["latencyMs"] for c in cs)
        u = [c.get("usage") or {} for c in cs]
        out[f"{p}/{a}"] = {"calls": len(cs), "status200": sum(1 for c in cs if c.get("status") == 200),
                           "finishReasons": dict(sorted(collections.Counter(str(c.get("finishReason")) for c in cs).items())),
                           "cachedTokensNonzero": sum(1 for x in u if (x.get("prompt_tokens_details") or {}).get("cached_tokens")),
                           "latencyMs": {"min": lat[0], "median": lat[len(lat) // 2], "max": lat[-1]} if lat else None}
    return out


def tables(doc):
    s = doc["summary"]
    L = ["# V0.11b decision distributions: A llm-agent 3, B validation only, C validation + preview", "",
         "Each point and arm: every scheduled draw (Gemma 10, Qwen 20), each one whole workflow. Wilson 95% intervals; "
         "differences with Newcombe 95% intervals. Descriptive only. The primary comparison is C − B; the secondary B − A.",
         "", "## Per point and arm", "",
         "| Point | Proposal | Category | Arm | Kept | Revised | Abandoned | Refused | Invalid | Final accepted / no action / "
         "none | Model calls | Retries | Preview shown (kept / revised / abandoned after) | Credit-supported |",
         "|---|---|---|---|---|---|---|---|---|---|---|---|---|---|"]
    for r in s["points"]:
        for a in ARMS:
            x = r["arms"][a]
            o, f = x["outcomes"], x["final"]
            pr = x["previewResponse"]
            L.append(f"| {r['id']} | {r['validation']} | {r['category']} | {a} | " +
                     " | ".join(f"{o[k]['count']}/{x['draws']}" for k in OUTCOMES) +
                     f" | {f['accepted']['count']} / {f['no_action']['count']} / {f['none']['count']} | {x['modelCalls']} | "
                     f"{x['retries']} | {x['preview'].get('shown', 0)} ({pr.get('kept', 0)} / {pr.get('revised', 0)} / "
                     f"{pr.get('abandoned', 0)}) | {x['creditSupported']} |")
    for t, c in COMPARISONS:
        key = f"{t}-{c}"
        L += ["", f"## {t} − {c}" + (" (primary)" if key == "C-B" else " (secondary)" if key == "B-A" else ""), "",
              "| Point | Kept | Revised | Abandoned | Refused | Invalid | Final accepted | Credit-supported | Same action content |",
              "|---|---|---|---|---|---|---|---|---|"]
        for r in s["points"]:
            cm = r["comparisons"][key]
            L.append(f"| {r['id']} | " + " | ".join(F.pp(cm["outcomes"][o]) for o in OUTCOMES) +
                     f" | {F.pp(cm['final']['accepted'])} | {F.pp(cm['creditSupported'])} | "
                     f"{'yes' if cm['sameActionContent'] else 'no'} |")
    L += ["", "## Pooled (after the per-point results)", "", "Unweighted means of the per-point differences; no interval.", "",
          "| Scope | Points | Comparison | Kept | Revised | Abandoned | Refused | Invalid | Points with the same action content |",
          "|---|---|---|---|---|---|---|---|---|"]
    for k, p in s["pooled"].items():
        for t, c in COMPARISONS:
            q = p[f"{t}-{c}"]
            m = q["unweightedMean"]
            L.append(f"| {k.replace(GEMMA, 'gemma').replace(QWEN, 'qwen')} | {p['points']} | {t}-{c} | " +
                     " | ".join(f"{100 * m[o]:+.1f} pp" for o in OUTCOMES) + f" | {q['pointsSameActionContent']} |")
    L += ["", "## Financial evidence v1 of the distinct final decisions", "",
          "| Point | Proposal | Draws A / B / C | Outcome | Commitment | Credit-supported | Retained obligations | "
          "Reversibility | Direct exposure | Displaced bookings |", "|---|---|---|---|---|---|---|---|---|---|"]
    for p in doc["points"]:
        for dd in p["decisions"]:
            n = [sum(1 for x in doc["draws"] if x["point"] == p["id"] and x["actionsKey"] == dd["key"] and x["arm"] == a)
                 for a in ARMS]
            fe = dd["financialEvidence"]
            L.append(f"| {p['id']} | {'yes' if dd['isProposal'] else ''} | {' / '.join(map(str, n))} | {dd['outcome']} | "
                     f"{X.money(fe['commitmentCents'])} | {'yes' if fe['creditSupported'] else 'no'} | {F.retained(fe)} | "
                     f"{', '.join(f'{k} {v}' for k, v in fe['reversibility'].items()) or '—'} | "
                     f"{', '.join(f'{k} {v}' for k, v in fe['directExposure'].items()) or '—'} | {dd['displacedBookings']} |")
    L += ["", "## Funding at the points", "", "| Point | Funding headroom | Of it without credit | Proposal's stake |",
          "|---|---|---|---|"]
    for p in doc["points"]:
        L.append(f"| {p['id']} | {X.money(p['fundingHeadroomCents'])} | {X.money(p['headroomWithoutCreditCents'])} | "
                 f"{X.money(p['stakeCents'])} |")
    return "\n".join(L) + "\n"


# --- main ------------------------------------------------------------------------------

def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("points")
    p.add_argument("--live", action="append", default=[])
    p.add_argument("--out")
    p.add_argument("--check")
    s = sub.add_parser("schedule")
    s.add_argument("--protocol", required=True)
    s.add_argument("--out", required=True)
    r = sub.add_parser("run")
    for q in (r,):
        q.add_argument("--binary", required=True)
        q.add_argument("--replays", required=True)
        q.add_argument("--protocol", required=True)
        q.add_argument("--out", required=True)
    a = sub.add_parser("analyze")
    a.add_argument("--binary", required=True)
    a.add_argument("--replays", required=True)
    a.add_argument("--dir", required=True)
    a.add_argument("--out", required=True)
    x = ap.parse_args(argv)
    try:
        if x.cmd == "points":
            if x.check:
                print(f"shared-prefix v4 points: {check_points(x.check)} contexts, selection and requests verified")
            else:
                m = make_points(x.live, x.out)
                print(f"shared-prefix v4 points: {len(m['selected'])} selected of {len(m['candidates'])} candidates")
            return 0
        if x.cmd == "analyze":
            doc = analyze(x.binary, x.dir, x.replays)
            write_json(os.path.join(x.out, "decisions.json"), doc)
            write_json(os.path.join(x.out, "inference.json"), {"schema": ANALYSIS_SCHEMA + "#inference",
                                                               "groups": inference(x.dir)})
            with open(os.path.join(x.out, "decision-tables.md"), "w", encoding="utf-8") as f:
                f.write(tables(doc))
            print(f"workflow analysis: {doc['drawsAnalyzed']} draws at {len(doc['points'])} points; replay checked "
                  f"{doc['replayChecked']}")
            return 0
        proto, proto_sha, man, contexts = load_protocol(x.protocol)
        sched = schedule(proto, contexts)
        if x.cmd == "schedule":
            write_json(os.path.join(x.out, "schedule.json"), schedule_doc(proto, proto_sha, man, sched))
            print(f"schedule: {len(sched)} draws, one at a time")
            return 0
        base = os.environ.get(proto["endpointEnv"])
        if not base:
            raise ExperimentError(f"set {proto['endpointEnv']}")
        doc = run(proto, proto_sha, man, contexts, x.out, base, x.binary, x.replays)
        print(f"workflow experiment: {doc['drawsRun']} of {doc['drawsScheduled']} draws, {doc['modelCalls']} model calls"
              + (f"; stopped: {doc['stopped']}" if doc["stopped"] else ""))
        return 0 if not doc["stopped"] else 1
    except (ExperimentError, F.ExperimentError, X.ExperimentError, S.SampleError) as e:
        print(f"workflow experiment: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
