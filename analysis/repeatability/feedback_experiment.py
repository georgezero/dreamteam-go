#!/usr/bin/env python3
"""The V0.10c experiment: validation feedback against preview information
(dreamteam.shared-prefix/v3; docs/benchmark/v10c-dev/SPEC.md and PLAN.md).

At each selected branch point every arm sends the model the recorded
conversation up to the recorded initial proposal, the proposal itself as
the model's reply, and one more user message:
* A, neutral: the V0.10b control message, unchanged;
* B, validation: the engine's exact validate answer for the proposal's
  staffing actions, then the same final request;
* C, preview: the engine's exact preview of the same actions (llm-agent
  4's message, llm-agent-preview/v1), then the same final request.
The model's one reply is the arm's final decision.

    # the API information inventory and the candidate inventory (no model)
    python3 -I -B analysis/repeatability/feedback_experiment.py inventory --binary ./dt --replays DIR --out DIR
    # the branch points (needs the live runs' model.jsonl, kept locally)
    python3 -I -B analysis/repeatability/feedback_experiment.py points --binary ./dt --replays DIR \
        --live DIR [--live DIR] --out DIR
    # the committed points checked again (no live files)
    python3 -I -B analysis/repeatability/feedback_experiment.py points --binary ./dt --replays DIR --check DIR
    # the schedule; the live run; the analysis (no model)
    python3 -I -B analysis/repeatability/feedback_experiment.py schedule --protocol PROTOCOL --out DIR
    DREAMTEAM_LLM_BASE_URL=... python3 -I -B analysis/repeatability/feedback_experiment.py run --protocol PROTOCOL --out DIR
    python3 -I -B analysis/repeatability/feedback_experiment.py analyze --binary ./dt --replays DIR --dir DIR --out DIR

It reuses shared-prefix/v2's code (decision_experiment.py) unchanged: the
engine sessions on the recordings' replays, the candidates, the requests,
the classification, financial evidence v1 and the statistics.
"""

import argparse
import collections
import json
import os
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import decision_experiment as X  # noqa: E402  (shared-prefix/v2, unchanged)

A3, A4, S, FE, RR = X.A3, X.A4, X.S, X.FE, X.RR
sha256, canon, write_json, load_jsonl = X.sha256, X.canon, X.write_json, X.load_jsonl

VERSION = "shared-prefix/v3"
PROTOCOL_SCHEMA = "dreamteam.shared-prefix/v3"
INVENTORY_SCHEMA = "dreamteam.feedback-inventory/v1"
POINTS_SCHEMA = "dreamteam.shared-prefix-points/v3"
SCHEDULE_SCHEMA = "dreamteam.shared-prefix-schedule/v3"
RUN_SCHEMA = "dreamteam.shared-prefix-run/v3"
ANALYSIS_SCHEMA = "dreamteam.shared-prefix-analysis/v3"
ANALYZER = "shared-prefix-analysis 3"
ARMS = ("neutral", "validation", "preview")
OUTCOMES = X.OUTCOMES

# Arm B's only text, built like llm-agent 4's review message: the engine's
# validate answer instead of its preview, the same final paragraph.
VALIDATION_TEMPLATE = (
    "VALIDATION. Nothing has been submitted yet. This is the simulator's validation of the staffing actions in "
    "your reply (actions {indices}), exactly as the simulator returned it:\n{validation}\n\n{ops_note}")
VALIDATION_OPS_NOTE = "The operations actions in your reply are not shown here (this covers the staffing actions only).\n\n"
NEUTRAL_TEXT = X.CONTROL_TEXT


class ExperimentError(Exception):
    pass


def validation_text(answer, actions):
    staff, idx = A4.staffing_part(actions)
    return VALIDATION_TEMPLATE.format(indices=", ".join(str(i) for i in idx), validation=A3.compact(answer),
                                      ops_note=VALIDATION_OPS_NOTE if len(staff) < len(actions) else "") \
        + X.FINAL_REQUEST


def arm_texts(proposal_actions, validate_answer, preview_answer):
    return {"neutral": NEUTRAL_TEXT, "validation": validation_text(validate_answer, proposal_actions),
            "preview": X.treatment_text(preview_answer, proposal_actions)}


# --- the information inventory -----------------------------------------------------

def keys_of(x, prefix=""):
    """Every field path in a JSON value (lists merged): a sorted list."""
    out = set()
    if isinstance(x, dict):
        for k, v in x.items():
            p = f"{prefix}.{k}" if prefix else k
            out.add(p)
            out |= set(keys_of(v, p))
    elif isinstance(x, list):
        for v in x:
            out |= set(keys_of(v, prefix + "[]"))
    return sorted(out)


def answers_at(binary, replays, cands, root=X.ROOT):
    """{(recording, point): (validate answer, preview answer, observation)}
    for the staffing actions of each candidate's proposal, asked at the
    recorded state (read-only, one session per recording)."""
    runs, out, by_rec = X.replay_runs(replays), {}, collections.defaultdict(list)
    for c in cands:
        by_rec[c["recording"]].append(c)
    for path, cs in sorted(by_rec.items()):
        ses = X.open_session(binary, runs[os.path.basename(path)])
        try:
            for c in sorted(cs, key=lambda c: c["point"]):
                rec = S.recording_point(root, path, c["point"])
                staff, _ = A4.staffing_part(rec["attempts"][c["attempt"] - 1]["actions"])
                obs, _ = ses.to_point(rec["digests"])
                v = ses.request("v10c-validate", "validate", {"actions": staff})
                p = ses.request("v10c-preview", "preview", {"actions": staff})
                if not v.get("ok") or not p.get("ok"):
                    raise ExperimentError(f"{path} point {c['point']}: validate or preview failed")
                out[(path, c["point"])] = (v["result"], p["result"], obs)
        finally:
            ses.close()
    return out


def money_class(c, fe):
    return {"commitmentCents": fe["commitmentCents"], "creditSupported": fe["creditSupported"],
            "affordability": fe["affordability"], "reversibility": fe["reversibility"], "directExposure": fe["directExposure"]}


def inventory(binary, replays, root=X.ROOT):
    """The API information inventory (which fields each answer carries,
    per staffing action type and result) and the candidate inventory (each
    V0.8c and V0.9c staffing proposal's financial evidence v1)."""
    cands = X.candidates(binary, replays, root)
    ans = answers_at(binary, replays, cands, root)
    runs = X.replay_runs(replays)
    fields = collections.defaultdict(lambda: {"n": 0, "validate": set(), "preview": set()})
    rows, same_results = [], 0
    for c in cands:
        rec = S.recording_point(root, c["recording"], c["point"])
        acts = rec["attempts"][c["attempt"] - 1]["actions"]
        v, p, obs = ans[(c["recording"], c["point"])]
        pv = p["preview"]
        same = v["results"] == pv["results"]
        same_results += same
        types = sorted({a["type"] for a in A4.staffing_part(acts)[0]})
        status = "accepted" if v["valid"] else "refused"
        for t in types:
            f = fields[(t, status)]
            f["n"] += 1
            f["validate"] |= set(keys_of(v))
            f["preview"] |= set(keys_of(p))
        e = {"id": f"{c['recording']}#{c['point']}", "recording": c["recording"], "point": c["point"],
             "digests": rec["digests"]}
        fe = X.decision_evidence(binary, runs[os.path.basename(c["recording"])], e, acts)
        fund = pv.get("funding") or {}
        fo = FE.funding_obs(obs) or {}
        rows.append({
            "recording": c["recording"], "recordingSha256": c["recordingSha256"], "point": c["point"],
            "attempt": c["attempt"], "attemptId": c["attemptId"], "model": c["model"],
            "contextKey": c["contextKey"], "shownPreview": c["shownPreview"], "rank": c["rank"],
            "eligible": c["eligible"], "exclusion": c["exclusion"],
            "staffingTypes": types, "validation": status,
            "refusals": sorted({f"{(r.get('action') or {}).get('type')}:{r.get('code')}" for r in v["results"]
                                if not r.get("ok")}),
            "previewResultsEqualValidate": same,
            "previewSections": sorted(k for k in pv if k not in ("asOf", "ok", "results")),
            "previewFunding": fund or None,
            "headroomWithoutCreditCents": fo.get("headroomWithoutCreditCents"),
            "displacedBookings": len(pv.get("displacements") or []),
            "validateChars": len(A3.compact(v)), "previewChars": len(A3.compact(p)),
            "proposal": {"outcome": fe["outcome"], **money_class(c, fe["financialEvidence"])}})
    api = []
    for (t, status), f in sorted(fields.items()):
        api.append({"type": t, "validation": status, "proposals": f["n"],
                    "validateFields": sorted(f["validate"]), "previewFields": sorted(f["preview"]),
                    "both": sorted(set(f["validate"]) & {k.split("preview.", 1)[-1] for k in f["preview"]}),
                    "previewOnly": sorted(k for k in f["preview"]
                                          if k.split("preview.", 1)[-1] not in f["validate"] and k != "preview")})
    return {"schema": INVENTORY_SCHEMA, "version": VERSION, "candidates": len(cands),
            "previewResultsEqualValidate": {"equal": same_results, "of": len(cands)},
            "api": api, "contexts": rows}


# --- the branch points ----------------------------------------------------------------

SELECT_RULE = "dreamteam-v10c-select/1"
ORDER_RULE = "dreamteam-v10c-order/1"
# Declared in PLAN.md §2 after the inventory and before any request.
GEMMA, QWEN = "gemma4-26b-a4b", "qwen36-35b-a3b"
QUOTA = {GEMMA: {"accepted": None, "refused": None}, QWEN: {"accepted": 4, "refused": 4}}  # None: every context
MAX_PER_RECORDING = 2
DRAWS = {GEMMA: 10, QWEN: 20}
ACCEPTED_COVER = ("credit_supported", "retained_obligation", "withdrawable", "displaces_bookings", "not_reversible")


def tags(row):
    """The financial and operational categories of a candidate's proposal
    (financial evidence v1 at the recorded state, and the preview)."""
    pr = row["proposal"]
    rev = pr["reversibility"]
    t = []
    if pr["creditSupported"]:
        t.append("credit_supported")
    if rev.get("reversible_with_retained_obligation"):
        t.append("retained_obligation")
    if rev.get("withdrawable"):
        t.append("withdrawable")
    if rev.get("not_reversible"):
        t.append("not_reversible")
    if row["displacedBookings"]:
        t.append("displaces_bookings")
    if any(r.endswith(":insufficient_funds") for r in row["refusals"]):
        t.append("insufficient_funds")
    return t


def stake(row):
    """A proposal's money at stake: the larger of the commitment the
    engine's funding test measures for its staffing actions (the preview)
    and the commitment financial evidence v1 measures for the whole
    proposal when it is applied; 0 if neither."""
    return max(((row.get("previewFunding") or {}).get("commitmentCents")) or 0,
               (row["proposal"].get("commitmentCents") or 0) if row["proposal"].get("outcome", "accepted") == "accepted"
               else 0)


def point_id(r):
    pv = "-pv" if "-preview-" in os.path.basename(r["recording"]) else ""
    return f"{X.SHORT[X.entry_of(r['recording'])]}-{X.MODEL_SHORT[r['model']]}{pv}-s{X.seed_of(r['recording'])}-p{r['point']}"


def select(rows):
    """Gemma: every distinct eligible context. Qwen, per stratum (the
    proposal's staffing actions valid or refused at the recorded state):
    * accepted: for each category of ACCEPTED_COVER in turn, the context
      with that category and the largest stake not yet taken; then the
      largest stakes;
    * refused: refusals for insufficient funds first, then by stake;
    ties by SHA-256('dreamteam-v10c-select/1|<recording>|<point>'); a
    context already taken is skipped, and at most MAX_PER_RECORDING points
    come from one recording."""
    chosen, seen, per_rec = [], set(), collections.Counter()

    def take(r):
        seen.add(r["contextKey"])
        per_rec[r["recording"]] += 1
        chosen.append(r)

    def free(r):
        return r["eligible"] and r["contextKey"] not in seen and per_rec[r["recording"]] < MAX_PER_RECORDING

    def tie(r):
        return sha256(f"{SELECT_RULE}|{r['recording']}|{r['point']}")
    for m in sorted(QUOTA):
        for st in ("accepted", "refused"):
            cell = [r for r in rows if r["model"] == m and r["validation"] == st]
            q = QUOTA[m][st]
            if q is None:
                for r in sorted(cell, key=lambda r: (not r["shownPreview"], tie(r))):
                    if r["eligible"] and r["contextKey"] not in seen:
                        take(r)
                continue
            got = 0
            if st == "accepted":
                for cat in ACCEPTED_COVER:
                    if got == q:
                        break
                    c = sorted((r for r in cell if free(r) and cat in tags(r)), key=lambda r: (-stake(r), tie(r)))
                    if c:
                        take(c[0])
                        got += 1
                order = sorted(cell, key=lambda r: (-stake(r), tie(r)))
            else:
                order = sorted(cell, key=lambda r: ("insufficient_funds" not in tags(r), -stake(r), tie(r)))
            for r in order:
                if got == q:
                    break
                if free(r):
                    take(r)
                    got += 1
            if got < q:
                raise ExperimentError(f"{m} {st}: {got} contexts can be taken, {q} declared")
    for r in chosen:
        r["id"] = point_id(r)
    return chosen


def context(r, rec, call, v, p):
    """A selected point's full context, every message checked against the
    recording, the three arms' messages built from the engine's answers."""
    at = rec["attempts"][r["attempt"] - 1]
    req, proposal = call["request"], call["content"]
    if [(m["role"], sha256(m["content"])) for m in req["messages"]] != S.chain(rec, r["attempt"]):
        raise ExperimentError(f"{r['id']}: the recorded request's messages differ from the recording's hashes")
    if sha256(proposal or "") != at["output"]["sha256"]:
        raise ExperimentError(f"{r['id']}: the recorded proposal is not the recording's output")
    if req.get("temperature") != 0 or set(req) != {"model", "messages", "temperature", "max_tokens", "response_format"} \
            or req.get("model") != S.MODELS[r["model"]]:
        raise ExperimentError(f"{r['id']}: the request's settings are not the recorded agent's")
    texts = arm_texts(at["actions"], v, p)
    arms = {a: {"text": t, "textSha256": sha256(t), "textChars": len(t),
                "requestSha256": sha256(X.body_bytes(X.arm_request(req, proposal, t)))} for a, t in texts.items()}
    e = {k: r[k] for k in ("id", "model", "validation", "refusals", "shownPreview", "recording", "point",
                           "attemptId", "contextKey", "staffingTypes")}
    e.update({"attempt": r["attempt"], "recordingSha256": r["recordingSha256"], "tags": tags(r), "stakeCents": stake(r),
              "digests": rec["digests"], "maxPromptChars": rec["promptSettings"]["maxPromptChars"],
              "request": req, "proposal": {"content": proposal, "sha256": sha256(proposal), "actions": at["actions"]},
              "validate": {"answer": v, "sha256": sha256(A3.compact(v))},
              "preview": {"answer": p, "sha256": sha256(A3.compact(p))}, "arms": arms})
    return e


def check_context(e, rec, v, p):
    at = rec["attempts"][e["attempt"] - 1]
    if [(m["role"], sha256(m["content"])) for m in e["request"]["messages"]] != S.chain(rec, e["attempt"]):
        raise ExperimentError(f"{e['id']}: the request's messages differ from the recording's hashes")
    if sha256(e["proposal"]["content"]) != at["output"]["sha256"] or e["proposal"]["actions"] != at["actions"]:
        raise ExperimentError(f"{e['id']}: the proposal is not the recording's")
    if e["validate"]["answer"] != v or e["preview"]["answer"] != p:
        raise ExperimentError(f"{e['id']}: the validate or preview answer is not the engine's")
    texts = arm_texts(at["actions"], v, p)
    for a in ARMS:
        x = e["arms"][a]
        b = X.arm_request(e["request"], e["proposal"]["content"], x["text"])
        if x["text"] != texts[a] or sha256(x["text"]) != x["textSha256"] or \
                sha256(X.body_bytes(b)) != x["requestSha256"]:
            raise ExperimentError(f"{e['id']}: the {a} arm is not built by the declared rule")
    rv = X.shown_review(rec, at)
    if rv is not None:
        b = X.arm_request(e["request"], e["proposal"]["content"], e["arms"]["preview"]["text"])
        if [(m["role"], sha256(m["content"])) for m in b["messages"]] != S.chain(rec, rv["attempt"]):
            raise ExperimentError(f"{e['id']}: the preview request is not the recorded review request")


def inventory_rows(binary, replays, root=X.ROOT):
    """The inventory and its candidate rows (stable order)."""
    doc = inventory(binary, replays, root)
    return doc, [dict(r) for r in doc["contexts"]]


def points_manifest(doc, chosen, contexts_sha):
    return {"schema": POINTS_SCHEMA, "version": VERSION, "rule": SELECT_RULE, "quota": QUOTA,
            "maxPerRecording": MAX_PER_RECORDING, "acceptedCover": list(ACCEPTED_COVER),
            "inventorySha256": sha256(canon(doc)),
            "selected": [{k: r[k] for k in ("id", "model", "validation", "refusals", "recording", "point", "attemptId",
                                            "contextKey")} | {"tags": tags(r), "stakeCents": stake(r)}
                         for r in chosen],
            "contextsFileSha256": contexts_sha}


def make_points(binary, replays, live, out, root=X.ROOT):
    doc, rows = inventory_rows(binary, replays, root)
    chosen = select(rows)
    ans = answers_at(binary, replays, chosen, root)
    lr = S.live_runs(live)
    contexts = []
    for r in chosen:
        rec = S.recording_point(root, r["recording"], r["point"])
        pe = lr.get(r["recordingSha256"])
        if pe is None:
            raise ExperimentError(f"{r['recording']}: no live run with this recording")
        calls = [x for x in load_jsonl(os.path.join(pe, "model.jsonl"))
                 if x.get("point") == r["point"] and x.get("attempt") == r["attempt"]]
        if len(calls) != 1:
            raise ExperimentError(f"{r['id']}: {len(calls)} recorded calls")
        v, p, _ = ans[(r["recording"], r["point"])]
        e = context(r, rec, calls[0], v, p)
        check_context(e, rec, v, p)
        contexts.append(e)
    os.makedirs(out, exist_ok=True)
    write_json(os.path.join(out, "inventory.json"), doc)
    body = "".join(json.dumps(e, ensure_ascii=False) + "\n" for e in contexts).encode("utf-8")
    with open(os.path.join(out, "contexts.jsonl"), "wb") as f:
        f.write(body)
    m = points_manifest(doc, chosen, sha256(body))
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


def check_points(binary, replays, d, root=X.ROOT):
    m, contexts = load_points(d)
    doc, rows = inventory_rows(binary, replays, root)
    with open(os.path.join(d, "inventory.json"), encoding="utf-8") as f:
        if json.load(f) != json.loads(json.dumps(doc)):
            raise ExperimentError("the inventory made again differs from the committed one")
    chosen = select(rows)
    if [r["id"] for r in chosen] != [e["id"] for e in contexts]:
        raise ExperimentError("the selection made again differs from the committed points")
    ans = answers_at(binary, replays, chosen, root)
    for r, e in zip(chosen, contexts):
        v, p, _ = ans[(r["recording"], r["point"])]
        check_context(e, S.recording_point(root, e["recording"], e["point"]), v, p)
    if json.loads(json.dumps(points_manifest(doc, chosen, m["contextsFileSha256"]))) != m:
        raise ExperimentError("the manifest is not the one the points make")
    return len(contexts)


# --- the schedule and the live run ------------------------------------------------------

def load_protocol(path):
    with open(path, "rb") as f:
        raw = f.read()
    proto = json.loads(raw)
    if proto.get("schema") != PROTOCOL_SCHEMA:
        raise ExperimentError(f"{path} is not a {PROTOCOL_SCHEMA} protocol")
    man, contexts = load_points(os.path.join(os.path.dirname(os.path.abspath(path)), proto["points"]))
    return proto, sha256(raw), man, contexts


def schedule(proto, contexts):
    """One request at a time. Per model (in the declared order), its
    draws as rounds; each round has every point and arm of the model once,
    in the order of SHA-256('<orderRule>|<model>|<round>|<point>|<arm>')."""
    rule, out = proto["orderRule"], []
    for m in proto["modelOrder"]:
        ids = [e["id"] for e in contexts if e["model"] == m]
        for r in range(1, proto["drawsPerArm"][m] + 1):
            cells = sorted(((p, a) for p in ids for a in ARMS),
                           key=lambda x: sha256(f"{rule}|{m}|{r}|{x[0]}|{x[1]}"))
            for p, a in cells:
                out.append({"seq": len(out) + 1, "model": m, "point": p, "arm": a, "draw": r})
    return out


def schedule_doc(proto, proto_sha, man, sched):
    return {"schema": SCHEDULE_SCHEMA, "protocol": proto["id"], "protocolSha256": proto_sha,
            "contextsFileSha256": man["contextsFileSha256"], "requests": len(sched), "concurrency": 1,
            "schedule": sched}


def run(proto, proto_sha, man, contexts, out, base, post_fn=RR.post, state_fn=RR.server_state, now=time.monotonic):
    """The live run: one request at a time, each response recorded as it
    ends; no retry, nothing dropped; server checks before every round."""
    sched = schedule(proto, contexts)
    by_id = {e["id"]: e for e in contexts}
    live = os.path.join(out, "live")
    os.makedirs(live, exist_ok=True)
    stop = proto["stop"]
    before = state_fn(base)
    write_json(os.path.join(live, "server-before.json"), before)
    sig = X.server_signature(before)
    t0, started, fails, done, stopped, checks, prev = now(), RR.utc(), 0, 0, None, [], None
    if (before.get("health") or {}).get("status") != "healthy":
        stopped = "health: the server did not answer healthy before the run"
    with open(os.path.join(live, "requests.jsonl"), "w", encoding="utf-8") as f:
        for r in sched if stopped is None else []:
            if (r["model"], r["draw"]) != prev:
                prev = (r["model"], r["draw"])
                st = state_fn(base)
                checks.append({"seq": r["seq"], "model": r["model"], "draw": r["draw"], "time": st.get("time"),
                               "health": st.get("health"), "sameConfiguration": X.server_signature(st) == sig})
                if (st.get("health") or {}).get("status") != "healthy":
                    stopped = f"health: the server did not answer healthy before request {r['seq']}"
                    break
                if X.server_signature(st) != sig:
                    stopped = f"server configuration changed before request {r['seq']}"
                    break
            if now() - t0 > stop["wallClockSeconds"]:
                stopped = "wall clock"
                break
            e = by_id[r["point"]]
            data = X.body_bytes(X.arm_request(e["request"], e["proposal"]["content"], e["arms"][r["arm"]]["text"]))
            if sha256(data) != e["arms"][r["arm"]]["requestSha256"]:
                raise ExperimentError(f"{r['point']} {r['arm']}: the request is not the declared one")
            t = RR.utc()
            status, body, err, lat = post_fn(base, data, proto["request"]["timeoutSeconds"])
            res = dict(r, sent=t, requestSha256=sha256(data), status=status, error=err, latencyMs=round(lat * 1000),
                       body=body, **RR.response_fields(body))
            res["responseSha256"] = sha256(body or "")
            f.write(json.dumps(res, sort_keys=True, ensure_ascii=False) + "\n")
            f.flush()
            done += 1
            fails = fails + 1 if err and status is None else 0
            if fails >= stop["consecutiveTransportFailures"]:
                stopped = f"{fails} consecutive transport failures"
                break
    after = state_fn(base)
    write_json(os.path.join(live, "server-after.json"), after)
    doc = {"schema": RUN_SCHEMA, "protocol": proto["id"], "protocolSha256": proto_sha,
           "contextsFileSha256": man["contextsFileSha256"], "started": started, "ended": RR.utc(),
           "requestsScheduled": len(sched), "requestsSent": done, "stopped": stopped, "serverChecks": checks,
           "sameConfigurationAfter": X.server_signature(after) == sig,
           "endpoint": "from " + proto["endpointEnv"] + " (not recorded)", "client": {"python": sys.version.split()[0]}}
    write_json(os.path.join(live, "run.json"), doc)
    return doc


# --- the analysis ---------------------------------------------------------------------

def check_rows(proto, contexts, rows):
    """The recorded requests are the schedule's, in order, each the
    declared request byte for byte, each response its recorded hash."""
    sched = schedule(proto, contexts)
    if [(r["seq"], r["point"], r["arm"], r["draw"]) for r in rows] != \
            [(s["seq"], s["point"], s["arm"], s["draw"]) for s in sched[:len(rows)]]:
        raise ExperimentError("the live requests are not the schedule's, in its order")
    by_id, by_point = {e["id"]: e for e in contexts}, collections.defaultdict(list)
    for r in rows:
        e = by_id[r["point"]]
        b = X.arm_request(e["request"], e["proposal"]["content"], e["arms"][r["arm"]]["text"])
        if r["requestSha256"] != sha256(X.body_bytes(b)) or r["requestSha256"] != e["arms"][r["arm"]]["requestSha256"]:
            raise ExperimentError(f"request {r['seq']}: not the declared request")
        if r.get("responseSha256") != sha256(r.get("body") or ""):
            raise ExperimentError(f"request {r['seq']}: the response is not the recorded one")
        if r.get("parsedBody") and RR.response_fields(r["body"]).get("content") != r.get("content"):
            raise ExperimentError(f"request {r['seq']}: the content is not the response's")
        by_point[r["point"]].append(r)
    return by_point


def displaced(records):
    return max([((r.get("directChange") or {}).get("batchSlots") or {}).get("displacedBookings") or 0
                for r in records if r["dimension"] == "OPERATIONAL_EXPOSURE"] or [0])


def retained(fe):
    return fe["reversibility"].get("reversible_with_retained_obligation", 0)


def analyze(binary, d, replays):
    proto, proto_sha, man, contexts = load_protocol(os.path.join(d, "protocol.json"))
    with open(os.path.join(d, "live", "run.json"), encoding="utf-8") as f:
        run_doc = json.load(f)
    rows = load_jsonl(os.path.join(d, "live", "requests.jsonl"))
    if run_doc["protocolSha256"] != proto_sha or run_doc["contextsFileSha256"] != man["contextsFileSha256"]:
        raise ExperimentError("the live run is not of this protocol and these points")
    sched = schedule(proto, contexts)
    by_point = check_rows(proto, contexts, rows)
    runs = X.replay_runs(replays)
    points, draws = [], []
    for e in contexts:
        run_dir = runs[os.path.basename(e["recording"])]
        ses = X.open_session(binary, run_dir)
        try:
            obs, catalog = ses.to_point(e["digests"])
            # The arms' engine answers, asked again: B's and C's messages are the engine's.
            for op in ("validate", "preview"):
                staff = A4.staffing_part(e["proposal"]["actions"])[0]
                a = ses.request(f"v10c-{op}", op, {"actions": staff})
                if a.get("result") != e[op]["answer"]:
                    raise ExperimentError(f"{e['id']}: the {op} answer is not the committed one")
        finally:
            ses.close()
        types = {t["type"] for t in catalog.get("types", [])}
        prop = e["proposal"]["actions"]
        cls = [(r, *X.classify(r, prop, types)) for r in by_point[e["id"]]]
        lists = {canon(prop): prop, canon([]): []}
        for _, _, _, acts in cls:
            if acts is not None:
                lists.setdefault(canon(acts), acts)
        ev = {k: X.decision_evidence(binary, run_dir, e, v) for k, v in sorted(lists.items())}
        none_key, pk = canon([]), canon(prop)
        pfe = ev[pk]["financialEvidence"]
        decisions = []
        for k in sorted(lists):
            x = ev[k]
            decisions.append({"key": sha256(k), "actions": lists[k], "isProposal": k == pk,
                              "displacedBookings": displaced(x["records"]),
                              **{f: x[f] for f in ("validation", "outcome", "applied", "validateApplyAgree",
                                                   "digestsAfterDecision", "suffix", "observedEnd",
                                                   "financialEvidence", "records")}})
        for r, o, why, acts in cls:
            k = canon(acts) if acts is not None else none_key
            x = ev[k]
            fe = x["financialEvidence"]
            end = x if x["outcome"] != "refused" else ev[none_key]
            draws.append({
                "seq": r["seq"], "point": e["id"], "model": e["model"], "arm": r["arm"], "draw": r["draw"],
                "outcome": o, "invalidReason": why, "finishReason": r.get("finishReason"),
                "rawSha256": sha256(r["content"]) if r.get("content") is not None else None,
                "responseSha256": r["responseSha256"], "parsedSha256": sha256(X.parsed_canon(r.get("content"))),
                "actionsKey": sha256(k) if acts is not None else None,
                "final": x["outcome"] if acts is not None else "invalid",
                "changes": X.changes(prop, acts) if o == "revised" else None,
                "distinctions": {
                    "informationChars": e["arms"][r["arm"]]["textChars"],
                    "proposalChanged": o in ("revised", "abandoned"),
                    "becameValid": acts is not None and ev[pk]["outcome"] == "refused" and x["outcome"] == "accepted",
                    "commitmentChanged": acts is not None and fe["commitmentCents"] != pfe["commitmentCents"],
                    "exposureChanged": acts is not None and fe["directExposure"] != pfe["directExposure"]},
                "commitmentCents": fe["commitmentCents"] if acts is not None else None,
                "commitmentChangeCents": fe["commitmentCents"] - pfe["commitmentCents"] if acts is not None else None,
                "endDigestsSha256": sha256(canon(end["suffix"]["endDigests"]))})
        fo = FE.funding_obs(obs) or {}
        points.append({"id": e["id"], "model": e["model"], "validation": e["validation"], "refusals": e["refusals"],
                       "tags": e["tags"], "stakeCents": e["stakeCents"], "recording": e["recording"],
                       "point": e["point"], "proposalSha256": e["proposal"]["sha256"], "proposal": prop,
                       "validateSha256": e["validate"]["sha256"], "previewSha256": e["preview"]["sha256"],
                       "headroomWithoutCreditCents": fo.get("headroomWithoutCreditCents"),
                       "fundingHeadroomCents": fo.get("fundingHeadroomCents"),
                       "arms": {a: {k: e["arms"][a][k] for k in ("textSha256", "textChars", "requestSha256")}
                                for a in ARMS},
                       "decisions": decisions})
    return {"schema": ANALYSIS_SCHEMA, "analyzer": ANALYZER, "protocol": proto["id"], "protocolSha256": proto_sha,
            "contextsFileSha256": man["contextsFileSha256"], "requestsScheduled": len(sched),
            "requestsAnalyzed": len(rows), "stopped": run_doc["stopped"], "points": points, "draws": draws,
            "summary": summarize(contexts, draws, proto), "notes": NOTES}


NOTES = X.NOTES[:1] + [
    "Intervals are Wilson 95% score intervals for a proportion and Newcombe hybrid score intervals for a difference. "
    "They are descriptive: each treats a point's draws as independent repetitions of one request, which V0.10a "
    "does not establish.",
    "The primary comparison is preview minus validation (C − B); validation minus neutral (B − A) and preview minus "
    "neutral (C − A) are references.",
] + X.NOTES[2:]

COMPARISONS = (("preview", "validation"), ("validation", "neutral"), ("preview", "neutral"))


def arm_summary(ds, n):
    cnt = collections.Counter(x["outcome"] for x in ds)
    fin = collections.Counter(x["final"] for x in ds)
    com = sorted(x["commitmentCents"] for x in ds if x["commitmentCents"] is not None)
    return {"draws": n, "observed": len(ds), "notRun": n - len(ds),
            "outcomes": {o: {"count": cnt.get(o, 0), "proportion": round(cnt.get(o, 0) / n, 4),
                             "wilson95": X.wilson(cnt.get(o, 0), n)} for o in OUTCOMES},
            "final": {k: {"count": fin.get(k, 0), "wilson95": X.wilson(fin.get(k, 0), n)}
                      for k in ("accepted", "refused", "no_action", "invalid")},
            "invalidReasons": dict(sorted(collections.Counter(x["invalidReason"] for x in ds
                                                              if x["outcome"] == "invalid").items())),
            "distinctDecisions": len({x["actionsKey"] or ("invalid:" + str(x["invalidReason"])) for x in ds}),
            "distinctActionProposals": len({x["actionsKey"] for x in ds if x["actionsKey"]}),
            "actionContent": dict(sorted(collections.Counter(x["actionsKey"] or "invalid" for x in ds).items())),
            "agreement": {"raw": X.modal([x["rawSha256"] for x in ds]),
                          "canonicalJson": X.modal([x["parsedSha256"] for x in ds]),
                          "actions": X.modal([x["actionsKey"] or "invalid" for x in ds])},
            "distinctions": {k: sum(1 for x in ds if x["distinctions"][k])
                             for k in ("proposalChanged", "becameValid", "commitmentChanged", "exposureChanged")},
            "commitmentCents": {"draws": len(com), "min": com[0] if com else None,
                                "median": com[len(com) // 2] if com else None, "max": com[-1] if com else None}}


def summarize(contexts, draws, proto):
    out = {"points": [], "pooled": {}}
    for e in contexts:
        n = proto["drawsPerArm"][e["model"]]
        row = {"id": e["id"], "model": e["model"], "validation": e["validation"], "draws": n,
               "arms": {a: arm_summary([x for x in draws if x["point"] == e["id"] and x["arm"] == a], n)
                        for a in ARMS},
               "comparisons": {}, "blocked": {}}
        for t, c in COMPARISONS:
            key = f"{t}-{c}"
            T, C = row["arms"][t], row["arms"][c]
            row["comparisons"][key] = {
                "outcomes": {o: X.newcombe(T["outcomes"][o]["count"], n, C["outcomes"][o]["count"], n)
                             for o in OUTCOMES},
                "final": {k: X.newcombe(T["final"][k]["count"], n, C["final"][k]["count"], n)
                          for k in ("accepted", "refused", "no_action", "invalid")},
                "sameActionContent": T["actionContent"] == C["actionContent"]}
            pairs = collections.Counter()
            for r in range(1, n + 1):
                td = next((x["outcome"] for x in draws if x["point"] == e["id"] and x["arm"] == t and x["draw"] == r),
                          "not_run")
                cd = next((x["outcome"] for x in draws if x["point"] == e["id"] and x["arm"] == c and x["draw"] == r),
                          "not_run")
                pairs[f"{cd}|{td}"] += 1
            row["blocked"][key] = {"rounds": n, "pairs": dict(sorted(pairs.items())),
                                   "discordant": sum(v for k, v in pairs.items() if k.split("|")[0] != k.split("|")[1])}
        out["points"].append(row)
    for scope in (GEMMA, QWEN, "all"):
        for st in ("accepted", "refused", "all"):
            rows = [r for r in out["points"] if scope in ("all", r["model"]) and st in ("all", r["validation"])]
            if not rows:
                continue
            k = f"{scope}/{st}"
            out["pooled"][k] = {"points": len(rows), "comparisons": {}}
            for t, c in COMPARISONS:
                key = f"{t}-{c}"
                diffs = {o: [r["comparisons"][key]["outcomes"][o]["difference"] for r in rows] for o in OUTCOMES}
                out["pooled"][k]["comparisons"][key] = {
                    "perPointDifferences": diffs,
                    "unweightedMean": {o: round(sum(v) / len(v), 4) for o, v in diffs.items()},
                    "pointsPositive": {o: sum(1 for d in v if d > 0) for o, v in diffs.items()},
                    "pointsNegative": {o: sum(1 for d in v if d < 0) for o, v in diffs.items()},
                    "pointsSameActionContent": sum(1 for r in rows if r["comparisons"][key]["sameActionContent"])}
            out["pooled"][k]["interval"] = "none: the points are clusters, too few for an interval over them"
    return out


def inference(d):
    """Live metadata, apart: tokens, latency, cache and finish reasons per
    point and arm."""
    rows = load_jsonl(os.path.join(d, "live", "requests.jsonl"))
    groups = collections.defaultdict(list)
    for r in rows:
        groups[(r["point"], r["arm"])].append(r)
    out = {}
    for (p, a), rs in sorted(groups.items()):
        u = [r.get("usage") or {} for r in rs]
        lat = sorted(r["latencyMs"] for r in rs)
        cached = [((x.get("prompt_tokens_details") or {}).get("cached_tokens")) for x in u]
        out[f"{p}/{a}"] = {"requests": len(rs), "status200": sum(1 for r in rs if r.get("status") == 200),
                           "finishReasons": dict(sorted(collections.Counter(str(r.get("finishReason"))
                                                                            for r in rs).items())),
                           "promptTokens": sorted({x.get("prompt_tokens") for x in u if x.get("prompt_tokens")}),
                           "completionTokens": {"min": min((x.get("completion_tokens") or 0) for x in u),
                                                "max": max((x.get("completion_tokens") or 0) for x in u)},
                           "cachedTokens": {"reported": sum(1 for c in cached if c is not None),
                                            "nonzero": sum(1 for c in cached if c)},
                           "latencyMs": {"min": lat[0], "median": lat[len(lat) // 2], "max": lat[-1]},
                           "servedModels": sorted({str(r.get("servedModel")) for r in rs})}
    return out


ARM_LABEL = {"neutral": "A neutral", "validation": "B validation", "preview": "C preview"}


def pp(c):
    return f"{100 * c['difference']:+.0f} pp {X.iv(c['interval'])}" if c else "n/a"


def tables(doc):
    s = doc["summary"]
    L = ["# V0.10c decision distributions: neutral, validation, preview", "",
         "Each point and arm: every scheduled draw in the denominator (Gemma 10, Qwen 20 per arm). Wilson 95% "
         "intervals; differences with Newcombe 95% intervals. Descriptive only. The primary comparison is C − B.", "",
         "## Per point and arm", "",
         "| Point | Proposal | Arm | Message chars | Kept | Revised | Abandoned | Invalid | Final accepted / refused / "
         "no action | Distinct decisions |", "|---|---|---|---|---|---|---|---|---|---|"]
    by = {p["id"]: p for p in doc["points"]}
    for r in s["points"]:
        for a in ARMS:
            x = r["arms"][a]
            o, f = x["outcomes"], x["final"]
            L.append(f"| {r['id']} | {r['validation']} | {ARM_LABEL[a]} | {by[r['id']]['arms'][a]['textChars']:,} | "
                     + " | ".join(f"{o[k]['count']}/{x['draws']} {X.iv(o[k]['wilson95'])}" for k in OUTCOMES)
                     + f" | {f['accepted']['count']} / {f['refused']['count']} / {f['no_action']['count']} | "
                     f"{x['distinctDecisions']} |")
    for t, c in COMPARISONS:
        key = f"{t}-{c}"
        L += ["", f"## {ARM_LABEL[t]} − {ARM_LABEL[c]}" + (" (primary)" if key == "preview-validation" else ""), "",
              "| Point | Kept | Revised | Abandoned | Invalid | Final accepted | Same action content | "
              "Discordant rounds |", "|---|---|---|---|---|---|---|---|"]
        for r in s["points"]:
            cm = r["comparisons"][key]
            L.append(f"| {r['id']} | " + " | ".join(pp(cm["outcomes"][o]) for o in OUTCOMES) +
                     f" | {pp(cm['final']['accepted'])} | {'yes' if cm['sameActionContent'] else 'no'} | "
                     f"{r['blocked'][key]['discordant']}/{r['blocked'][key]['rounds']} |")
    L += ["", "## Pooled (after the per-point results)", "",
          "Unweighted means of the per-point differences; the points are clusters, too few for an interval.", "",
          "| Scope | Points | Comparison | Kept | Revised | Abandoned | Invalid | Points with the same action content |",
          "|---|---|---|---|---|---|---|---|"]
    for k, p in s["pooled"].items():
        for key, c in p["comparisons"].items():
            m = c["unweightedMean"]
            L.append(f"| {k.replace(GEMMA, 'gemma').replace(QWEN, 'qwen')} | {p['points']} | {key} | " +
                     " | ".join(f"{100 * m[o]:+.1f} pp" for o in OUTCOMES) + f" | {c['pointsSameActionContent']} |")
    L += ["", "## What changed, per arm (draws)", "",
          "| Point | Arm | Proposal changed | Became valid | Commitment changed | Exposure changed |",
          "|---|---|---|---|---|---|"]
    for r in s["points"]:
        for a in ARMS:
            dd = r["arms"][a]["distinctions"]
            L.append(f"| {r['id']} | {ARM_LABEL[a]} | {dd['proposalChanged']} | {dd['becameValid']} | "
                     f"{dd['commitmentChanged']} | {dd['exposureChanged']} |")
    L += ["", "## Financial evidence v1 of the distinct final decisions", "",
          "| Point | Proposal | Draws A / B / C | Outcome | Commitment | Credit-supported | Retained obligations | "
          "Reversibility | Direct exposure | Displaced bookings |", "|---|---|---|---|---|---|---|---|---|---|"]
    for p in doc["points"]:
        for dd in p["decisions"]:
            n = [sum(1 for x in doc["draws"] if x["point"] == p["id"] and x["actionsKey"] == dd["key"] and
                     x["arm"] == a) for a in ARMS]
            fe = dd["financialEvidence"]
            L.append(f"| {p['id']} | {'yes' if dd['isProposal'] else ''} | {' / '.join(map(str, n))} | {dd['outcome']} | "
                     f"{X.money(fe['commitmentCents'])} | {'yes' if fe['creditSupported'] else 'no'} | {retained(fe)} | "
                     f"{', '.join(f'{k} {v}' for k, v in fe['reversibility'].items()) or '—'} | "
                     f"{', '.join(f'{k} {v}' for k, v in fe['directExposure'].items()) or '—'} | "
                     f"{dd['displacedBookings']} |")
    L += ["", "## Funding at the points", "", "| Point | Funding headroom | Of it without credit | Proposal's stake |",
          "|---|---|---|---|"]
    for p in doc["points"]:
        L.append(f"| {p['id']} | {X.money(p['fundingHeadroomCents'])} | {X.money(p['headroomWithoutCreditCents'])} | "
                 f"{X.money(p['stakeCents'])} |")
    return "\n".join(L) + "\n"


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    i = sub.add_parser("inventory")
    i.add_argument("--binary", required=True)
    i.add_argument("--replays", required=True)
    i.add_argument("--out", required=True)
    p = sub.add_parser("points")
    p.add_argument("--binary", required=True)
    p.add_argument("--replays", required=True)
    p.add_argument("--live", action="append", default=[])
    p.add_argument("--out")
    p.add_argument("--check")
    s = sub.add_parser("schedule")
    s.add_argument("--protocol", required=True)
    s.add_argument("--out", required=True)
    r = sub.add_parser("run")
    r.add_argument("--protocol", required=True)
    r.add_argument("--out", required=True)
    a = sub.add_parser("analyze")
    a.add_argument("--binary", required=True)
    a.add_argument("--replays", required=True)
    a.add_argument("--dir", required=True)
    a.add_argument("--out", required=True)
    x = ap.parse_args(argv)
    try:
        if x.cmd == "inventory":
            doc = inventory(x.binary, x.replays)
            write_json(os.path.join(x.out, "inventory.json"), doc)
            print(f"inventory: {doc['candidates']} staffing proposals; preview results equal validate's in "
                  f"{doc['previewResultsEqualValidate']['equal']}")
            return 0
        if x.cmd == "points":
            if x.check:
                print(f"shared-prefix v3 points: {check_points(x.binary, x.replays, x.check)} contexts, inventory, "
                      "selection, engine answers and requests verified")
            else:
                m = make_points(x.binary, x.replays, x.live, x.out)
                print(f"shared-prefix v3 points: {len(m['selected'])} selected")
            return 0
        if x.cmd == "analyze":
            doc = analyze(x.binary, x.dir, x.replays)
            write_json(os.path.join(x.out, "decisions.json"), doc)
            write_json(os.path.join(x.out, "inference.json"), {"schema": ANALYSIS_SCHEMA + "#inference",
                                                               "groups": inference(x.dir)})
            with open(os.path.join(x.out, "decision-tables.md"), "w", encoding="utf-8") as f:
                f.write(tables(doc))
            print(f"decision analysis: {doc['requestsAnalyzed']} draws at {len(doc['points'])} points")
            return 0
        proto, proto_sha, man, contexts = load_protocol(x.protocol)
        sched = schedule(proto, contexts)
        if x.cmd == "schedule":
            write_json(os.path.join(x.out, "schedule.json"), schedule_doc(proto, proto_sha, man, sched))
            print(f"schedule: {len(sched)} requests, one at a time")
            return 0
        base = os.environ.get(proto["endpointEnv"])
        if not base:
            raise ExperimentError(f"set {proto['endpointEnv']}")
        doc = run(proto, proto_sha, man, contexts, x.out, base)
        print(f"feedback experiment: {doc['requestsSent']} of {doc['requestsScheduled']} requests"
              + (f"; stopped: {doc['stopped']}" if doc["stopped"] else ""))
        return 0 if not doc["stopped"] else 1
    except (ExperimentError, X.ExperimentError, S.SampleError) as e:
        print(f"feedback experiment: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
