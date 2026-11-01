#!/usr/bin/env python3
"""The V0.10b live shared-prefix decision experiment (dreamteam.shared-prefix/v2;
docs/benchmark/v10b-dev/SPEC.md and PLAN.md).

At each selected branch point both arms send the model the recorded
conversation up to the recorded initial proposal, the proposal itself as
the model's reply, and then one more user message:
* control: a neutral continuation that asks for the final decision;
* treatment: the engine's exact preview of the proposal's staffing actions
  (llm-agent 4's message, llm-agent-preview/v1), then the same request.
The model's one reply is the arm's final decision. Nothing else is asked.

    # the branch points: candidates, selection, contexts (needs the live
    # runs' model.jsonl, kept locally, and the replays of the recordings)
    python3 -I -B analysis/repeatability/decision_experiment.py points --binary ./dt --replays DIR \
        --live DIR [--live DIR] --out DIR
    # the committed points checked again (no live files)
    python3 -I -B analysis/repeatability/decision_experiment.py points --binary ./dt --replays DIR --check DIR
    # the schedule (a function of the protocol and the points)
    python3 -I -B analysis/repeatability/decision_experiment.py schedule --protocol PROTOCOL --out DIR
    # the live run, sequential, every response recorded
    DREAMTEAM_LLM_BASE_URL=... python3 -I -B analysis/repeatability/decision_experiment.py run \
        --protocol PROTOCOL --out DIR
    # the analysis of the recorded responses: no model is contacted
    python3 -I -B analysis/repeatability/decision_experiment.py analyze --binary ./dt --replays DIR \
        --dir DIR --out DIR

The engine's answers come from the recordings' replays: each replay's
protocol transcript is fed back to the same engine (`dt agent --stdio` on
the run's scenario and seed), every recorded answer must come back
identical, and the experiment's requests are asked at the recorded
decision point. No recording, agent, prompt or evidence rule is changed.
"""

import argparse
import collections
import glob
import hashlib
import json
import math
import os
import subprocess
import sys
import tempfile
import time

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, os.path.join(ROOT, "agents", "llm"))
sys.path.insert(0, os.path.join(ROOT, "analysis", "evidence"))
sys.path.insert(0, HERE)
import llm_agent3 as A3  # noqa: E402  (unchanged: the parser, batching and constants)
import llm_agent4 as A4  # noqa: E402  (unchanged: the preview rule and its message)
import financial_evidence as FE  # noqa: E402  (unchanged: financial evidence v1's definitions)
import prompt_sample as S  # noqa: E402  (the recorded-request checks of V0.10a)
import repeat_run as RR  # noqa: E402  (the V0.10a client: post, server state, response fields)

VERSION = "shared-prefix/v2"
PROTOCOL_SCHEMA = "dreamteam.shared-prefix/v2"
POINTS_SCHEMA = "dreamteam.shared-prefix-points/v2"
SCHEDULE_SCHEMA = "dreamteam.shared-prefix-schedule/v2"
RUN_SCHEMA = "dreamteam.shared-prefix-run/v2"
ANALYSIS_SCHEMA = "dreamteam.shared-prefix-analysis/v2"
ANALYZER = "shared-prefix-analysis 2"
SELECT_RULE = "dreamteam-v10b-select/1"
ARMS = ("control", "treatment")
STRATA = ("accepted", "refused")
# The owner-approved design (option A of docs/audits/v0.10b-branch-point-inventory.md).
QUOTA = {"gemma4-26b-a4b": {"accepted": 1, "refused": 3}, "qwen36-35b-a3b": {"accepted": 2, "refused": 2}}
SHORT = {"financial-pressure-credit": "fp", "financial-pressure-lowliquidity": "fl", "staffing-shortage-credit": "ss"}
MODEL_SHORT = {"gemma4-26b-a4b": "gemma", "qwen36-35b-a3b": "qwen"}

# The control arm's only text. Its second paragraph is the treatment
# message's last paragraph (llm-agent-preview/v1), byte for byte: the arms
# differ by the preview paragraph and its first words alone.
FINAL_REQUEST = A4.REVIEW_TEMPLATE[A4.REVIEW_TEMPLATE.index("{ops_note}") + len("{ops_note}"):].replace(
    "{{", "{").replace("}}", "}")
CONTROL_TEXT = "CONTINUE. Nothing has been submitted yet.\n\n" + FINAL_REQUEST
OUTCOMES = ("kept", "revised", "abandoned", "invalid")
Z95 = 1.959963984540054


class ExperimentError(Exception):
    pass


def sha256(b):
    return hashlib.sha256(b if isinstance(b, bytes) else b.encode("utf-8")).hexdigest()


def canon(x):
    return json.dumps(x, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def write_json(path, obj):
    os.makedirs(os.path.dirname(os.path.abspath(path)), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(obj, f, indent=1, sort_keys=True, ensure_ascii=False)
        f.write("\n")


def load_jsonl(path):
    with open(path, encoding="utf-8") as f:
        return [json.loads(l) for l in f if l.strip()]


# --- the engine at a recorded decision point ----------------------------

def replay_runs(replays):
    """{recording basename: replay run directory}, from each replay run's
    own record of the recording it replayed."""
    out = {}
    for ev in sorted(glob.glob(os.path.join(replays, "*", "runs", "*", "policy-evidence", "decisions.jsonl"))):
        with open(ev, encoding="utf-8") as f:
            first = f.readline()
        rb = json.loads(first).get("replayed") if first.strip() else None
        if rb:
            out[rb] = os.path.dirname(os.path.dirname(ev))
    return out


def transcript(run_dir):
    with open(os.path.join(run_dir, "run.json"), encoding="utf-8") as f:
        run = json.load(f)
    pairs, req = [], None
    with open(os.path.join(run_dir, "transcript.jsonl"), encoding="utf-8") as f:
        for line in f:
            d = json.loads(line)
            if "op" in d:
                req = d
            elif req is not None:
                pairs.append((req, d))
                req = None
    return run, pairs


def spawn(binary, run):
    """A fresh engine session on the run's scenario and seed: (ask, close).
    The engine writes nothing into the caller's directories."""
    scratch = tempfile.TemporaryDirectory(prefix="v10b-engine-")
    proc = subprocess.Popen([os.path.abspath(binary), "agent", "--stdio", "-scenario", run["scenario"]["ref"],
                             "-seed", str(run["seed"])], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL, text=True, encoding="utf-8", cwd=scratch.name)

    def ask(r):
        proc.stdin.write(json.dumps(r) + "\n")
        proc.stdin.flush()
        line = proc.stdout.readline()
        if not line:
            raise ExperimentError("the engine closed the session")
        return json.loads(line)

    def close():
        proc.stdin.close()
        proc.wait()
        scratch.cleanup()
    return ask, close


class Session:
    """One engine session on a replay's transcript. to_point feeds the
    recorded requests up to a decision point's digests and that point's
    read-only requests (observe, actions); every answer must be the
    recorded one. After that, only the caller's requests are sent."""

    def __init__(self, run, pairs, ask, close=lambda: None):
        self.run, self.pairs, self.ask, self.close, self.i = run, pairs, ask, close, 0

    def feed(self, r, want):
        got = self.ask(r)
        if r.get("op") == "hello":
            g, w = got.get("result") or {}, want.get("result") or {}
            same = g.get("clock") == w.get("clock") and (g.get("scenario") or {}).get("fingerprint") == \
                (w.get("scenario") or {}).get("fingerprint")
        else:
            same = got == want
        if not same:
            raise ExperimentError(f"{self.run.get('id')}: recorded request {self.i + 1} ({r.get('op')}) answered "
                                  "differently")
        return got

    def to_point(self, digests):
        """(observation, catalog) at the decision point whose digests are
        these."""
        reached, obs, catalog = False, None, None
        while self.i < len(self.pairs):
            r, want = self.pairs[self.i]
            op = r.get("op")
            if reached and op not in ("observe", "actions"):
                break
            got = self.feed(r, want)
            self.i += 1
            res = got.get("result") or {}
            if not reached and op == "digests" and got.get("ok") and res.get("digests") == digests:
                reached = True
            elif reached and op == "observe" and obs is None:
                obs = res.get("observation")
            elif reached and op == "actions" and catalog is None:
                catalog = res
        if not reached:
            raise ExperimentError(f"{self.run.get('id')}: the decision point was not reached")
        return obs, catalog

    def later_advances(self):
        return [r for r, _ in self.pairs[self.i:] if r.get("op") == "advance"]

    def request(self, rid, op, args=None):
        r = {"id": rid, "op": op}
        if args is not None:
            r["args"] = args
        return self.ask(r)


def open_session(binary, run_dir):
    run, pairs = transcript(run_dir)
    ask, close = spawn(binary, run)
    return Session(run, pairs, ask, close)


# --- the branch points ------------------------------------------------------

def proposal_attempt(rec):
    """The recorded initial proposal: the first parsed reply at the point
    with a staffing action (llm-agent 4's preview rule), or None."""
    for at in rec.get("attempts") or []:
        if A4.preview_due(at, False):
            return at
    return None


def entry_of(path):
    name = os.path.basename(path)
    return next(e for e in SHORT if name.startswith(e + "-"))


def seed_of(path):
    return int(os.path.basename(path).split("-s")[-1].split("-")[0])


def context_key(model, rec, at):
    """Two candidates are one context when the model, the state, every
    message up to the proposal and the proposal are the same."""
    return sha256(canon([model, rec["digests"], S.chain(rec, at["attempt"]), at["output"]["sha256"]]))


def shown_review(rec, at):
    """The recorded llm-agent 4 attempt that answered the shown preview of
    this proposal, or None."""
    if (at.get("preview") or {}).get("status") != "ok":
        return None
    return next((x for x in rec["attempts"] if (x.get("input") or {}).get("kind") == "preview"), None)


def preview_status(answer):
    p = (answer or {}).get("preview") or {}
    refused = sorted({f"{(x.get('action') or {}).get('type')}:{x.get('code')}" for x in p.get("results") or []
                      if not x.get("ok")})
    return ("accepted" if p.get("ok") else "refused"), refused


def treatment_text(answer, actions):
    staff, idx = A4.staffing_part(actions)
    return A4.review_text(answer, idx, len(staff) < len(actions))


def candidates(binary, replays, root=ROOT):
    """Every V0.8c and V0.9c decision point with a staffing proposal, its
    preview asked of the engine at the recorded state, in a stable order."""
    runs = replay_runs(replays)
    out = []
    for pat in S.RECORDINGS:
        for p in sorted(glob.glob(os.path.join(root, pat))):
            path = os.path.relpath(p, root).replace(os.sep, "/")
            with open(p, "rb") as f:
                raw = f.read()
            recs = [json.loads(l) for l in raw.decode("utf-8").splitlines() if l.strip()]
            found = [(rec, proposal_attempt(rec)) for rec in recs]
            found = [(rec, at) for rec, at in found if at is not None]
            if not found:
                continue
            base = os.path.basename(path)
            if base not in runs:
                raise ExperimentError(f"{base}: no replay in {replays}")
            ses = open_session(binary, runs[base])
            try:
                for rec, at in found:
                    staff, _ = A4.staffing_part(at["actions"])
                    ses.to_point(rec["digests"])
                    ans = ses.request("v10b-preview", "preview", {"actions": staff})
                    model = S.model_of(path)
                    c = {"recording": path, "recordingSha256": sha256(raw), "point": rec["point"],
                         "attempt": at["attempt"], "attemptId": at["id"], "model": model,
                         "agentRecord": rec.get("schema"), "entry": entry_of(path), "seed": seed_of(path),
                         "rank": sha256(f"{SELECT_RULE}|{path}|{rec['point']}"),
                         "shownPreview": shown_review(rec, at) is not None, "eligible": True, "exclusion": None}
                    try:
                        c["contextKey"] = context_key(model, rec, at)
                    except S.SampleError as e:
                        c.update(eligible=False, exclusion=f"prefix: {e}", contextKey=None)
                    if not ans.get("ok"):
                        c.update(eligible=False, exclusion="preview: " + str((ans.get("error") or {}).get("code")),
                                 stratum=None, refusals=None, previewSha256=None)
                    else:
                        res = ans["result"]
                        c["stratum"], c["refusals"] = preview_status(res)
                        c["previewSha256"] = sha256(A3.compact(res))
                        text = treatment_text(res, at["actions"])
                        c["treatmentChars"] = len(text)
                        if len(text) > A4.preview_limit(rec["promptSettings"]):
                            c.update(eligible=False, exclusion="preview: too_large (llm-agent 4 would not show it)")
                        rv = shown_review(rec, at)
                        if rv is not None and sha256(text) != rv["input"]["sha256"]:
                            raise ExperimentError(f"{path} point {rec['point']}: the engine's preview is not the "
                                                  "one shown")
                    out.append(c)
            finally:
                ses.close()
    return out


def select(cands):
    """Per model and stratum: the V0.9c points whose preview was shown
    first, then the rest, each by rank; a context already taken is
    skipped; the quota taken."""
    chosen, seen = [], set()
    for m in sorted(QUOTA):
        for s in STRATA:
            cell = sorted((c for c in cands if c["model"] == m and c["eligible"] and c["stratum"] == s),
                          key=lambda c: (not c["shownPreview"], c["rank"]))
            took = 0
            for c in cell:
                if took == QUOTA[m][s]:
                    break
                if c["contextKey"] in seen:
                    continue
                seen.add(c["contextKey"])
                chosen.append(c)
                took += 1
            if took < QUOTA[m][s]:
                raise ExperimentError(f"{m} {s}: {took} distinct eligible contexts, {QUOTA[m][s]} declared")
    for c in chosen:
        c["id"] = f"{SHORT[entry_of(c['recording'])]}-{MODEL_SHORT[c['model']]}-s{seed_of(c['recording'])}-p{c['point']}"
    return chosen


def distribution(cands):
    out = {}
    for m in sorted(QUOTA):
        for s in STRATA:
            cell = [c for c in cands if c["model"] == m and c["stratum"] == s]
            el = [c for c in cell if c["eligible"]]
            out[f"{m}/{s}"] = {"candidates": len(cell), "eligible": len(el),
                               "distinctContexts": len({c["contextKey"] for c in el}),
                               "shownPreview": len({c["contextKey"] for c in el if c["shownPreview"]})}
    return out


def arm_request(request, proposal, text):
    """The arm's request: the recorded request, its messages followed by
    the proposal (the model's reply) and the arm's message; nothing else
    changed, keys in the recorded order."""
    body = {}
    for k, v in request.items():
        body[k] = list(v) + [{"role": "assistant", "content": proposal}, {"role": "user", "content": text}] \
            if k == "messages" else v
    return body


def body_bytes(body):
    return json.dumps(body).encode("utf-8")


def context(c, rec, call, preview_answer):
    """A selected point's full context, from the live run's recorded
    request and reply, every message checked against the recording."""
    at = rec["attempts"][c["attempt"] - 1]
    req, proposal = call["request"], call["content"]
    if [(m["role"], sha256(m["content"])) for m in req["messages"]] != S.chain(rec, c["attempt"]):
        raise ExperimentError(f"{c['id']}: the recorded request's messages differ from the recording's hashes")
    if sha256(proposal or "") != at["output"]["sha256"]:
        raise ExperimentError(f"{c['id']}: the recorded proposal is not the recording's output")
    if req.get("temperature") != 0 or set(req) != {"model", "messages", "temperature", "max_tokens", "response_format"} \
            or req.get("model") != S.MODELS[c["model"]]:
        raise ExperimentError(f"{c['id']}: the request's settings are not the recorded agent's")
    texts = {"control": CONTROL_TEXT, "treatment": treatment_text(preview_answer, at["actions"])}
    arms = {}
    for arm in ARMS:
        b = arm_request(req, proposal, texts[arm])
        arms[arm] = {"text": texts[arm], "textSha256": sha256(texts[arm]), "textChars": len(texts[arm]),
                     "requestSha256": sha256(body_bytes(b)), "requestChars": sum(len(m["content"]) for m in
                                                                               b["messages"])}
    rv = shown_review(rec, at)
    e = {k: c[k] for k in ("id", "model", "stratum", "refusals", "shownPreview", "recording", "recordingSha256",
                           "point", "attempt", "attemptId", "entry", "seed", "contextKey")}
    e.update({"digests": rec["digests"], "maxPromptChars": rec["promptSettings"]["maxPromptChars"],
              "request": req, "proposal": {"content": proposal, "sha256": sha256(proposal),
                                           "actions": at["actions"]},
              "preview": {"answer": preview_answer, "sha256": sha256(A3.compact(preview_answer))},
              "arms": arms, "recordedReview": None})
    if rv is not None:
        e["recordedReview"] = {"attemptId": rv["id"], "inputSha256": rv["input"]["sha256"],
                               "outputSha256": (rv.get("output") or {}).get("sha256")}
    return e


def check_context(e, rec, preview_answer):
    """A committed context checked again without the live files."""
    at = rec["attempts"][e["attempt"] - 1]
    if [(m["role"], sha256(m["content"])) for m in e["request"]["messages"]] != S.chain(rec, e["attempt"]):
        raise ExperimentError(f"{e['id']}: the request's messages differ from the recording's hashes")
    if sha256(e["proposal"]["content"]) != at["output"]["sha256"] or e["proposal"]["actions"] != at["actions"]:
        raise ExperimentError(f"{e['id']}: the proposal is not the recording's")
    if e["preview"]["answer"] != preview_answer:
        raise ExperimentError(f"{e['id']}: the preview is not the engine's")
    for arm in ARMS:
        a = e["arms"][arm]
        b = arm_request(e["request"], e["proposal"]["content"], a["text"])
        if sha256(body_bytes(b)) != a["requestSha256"] or sha256(a["text"]) != a["textSha256"]:
            raise ExperimentError(f"{e['id']}: the {arm} request is not the declared one")
    if e["arms"]["control"]["text"] != CONTROL_TEXT or \
            e["arms"]["treatment"]["text"] != treatment_text(preview_answer, at["actions"]):
        raise ExperimentError(f"{e['id']}: an arm's message is not built by the declared rule")
    rv = shown_review(rec, at)
    if rv is not None:
        # The treatment request is the one llm-agent 4 sent at the point.
        want = S.chain(rec, rv["attempt"])
        b = arm_request(e["request"], e["proposal"]["content"], e["arms"]["treatment"]["text"])
        if [(m["role"], sha256(m["content"])) for m in b["messages"]] != want:
            raise ExperimentError(f"{e['id']}: the treatment request is not the recorded review request")


def previews_for(binary, replays, chosen, root=ROOT):
    """{id: the engine's preview answer} at each chosen point."""
    runs, out = replay_runs(replays), {}
    for c in chosen:
        rec = S.recording_point(root, c["recording"], c["point"])
        at = rec["attempts"][c["attempt"] - 1]
        ses = open_session(binary, runs[os.path.basename(c["recording"])])
        try:
            ses.to_point(rec["digests"])
            ans = ses.request("v10b-preview", "preview", {"actions": A4.staffing_part(at["actions"])[0]})
        finally:
            ses.close()
        if not ans.get("ok"):
            raise ExperimentError(f"{c['id']}: the preview failed")
        out[c["id"]] = ans["result"]
    return out


def points_manifest(cands, chosen, contexts_sha):
    return {"schema": POINTS_SCHEMA, "version": VERSION, "rule": SELECT_RULE, "quota": QUOTA,
            "strata": list(STRATA), "recordings": list(S.RECORDINGS),
            "selection": "per model and stratum (the preview at the recorded state accepted or refused): the "
                         "V0.9c points whose preview was shown first, then the rest, each by SHA-256 of "
                         f"'{SELECT_RULE}|<recording>|<point>'; a context already taken is skipped",
            "distribution": distribution(cands),
            "candidates": [{k: c.get(k) for k in ("recording", "point", "attemptId", "model", "agentRecord",
                                                  "stratum", "refusals", "shownPreview", "eligible", "exclusion",
                                                  "rank", "contextKey", "previewSha256", "treatmentChars")}
                           for c in cands],
            "selected": [{k: c[k] for k in ("id", "model", "stratum", "refusals", "shownPreview", "recording",
                                            "point", "attemptId", "contextKey", "previewSha256")} for c in chosen],
            "contextsFileSha256": contexts_sha}


def write_points(out, cands, chosen, contexts):
    os.makedirs(out, exist_ok=True)
    body = "".join(json.dumps(e, ensure_ascii=False, sort_keys=False) + "\n" for e in contexts).encode("utf-8")
    with open(os.path.join(out, "contexts.jsonl"), "wb") as f:
        f.write(body)
    m = points_manifest(cands, chosen, sha256(body))
    write_json(os.path.join(out, "manifest.json"), m)
    return m


def make_points(binary, replays, live, out, root=ROOT):
    cands = candidates(binary, replays, root)
    chosen = select(cands)
    runs = S.live_runs(live)
    pvs = previews_for(binary, replays, chosen, root)
    contexts = []
    for c in chosen:
        rec = S.recording_point(root, c["recording"], c["point"])
        pe = runs.get(c["recordingSha256"])
        if pe is None:
            raise ExperimentError(f"{c['recording']}: no live run with this recording")
        calls = [x for x in load_jsonl(os.path.join(pe, "model.jsonl"))
                 if x.get("point") == c["point"] and x.get("attempt") == c["attempt"]]
        if len(calls) != 1:
            raise ExperimentError(f"{c['id']}: {len(calls)} recorded calls")
        e = context(c, rec, calls[0], pvs[c["id"]])
        check_context(e, rec, pvs[c["id"]])
        contexts.append(e)
    return write_points(out, cands, chosen, contexts)


def load_points(d):
    with open(os.path.join(d, "manifest.json"), encoding="utf-8") as f:
        m = json.load(f)
    with open(os.path.join(d, "contexts.jsonl"), "rb") as f:
        body = f.read()
    if sha256(body) != m["contextsFileSha256"]:
        raise ExperimentError("contexts.jsonl is not the manifest's")
    return m, [json.loads(l) for l in body.decode("utf-8").splitlines() if l.strip()]


def check_points(binary, replays, d, root=ROOT):
    m, contexts = load_points(d)
    cands = candidates(binary, replays, root)
    chosen = select(cands)
    if [c["id"] for c in chosen] != [e["id"] for e in contexts]:
        raise ExperimentError("the selection made again differs from the committed points")
    pvs = previews_for(binary, replays, chosen, root)
    for c, e in zip(chosen, contexts):
        if c["recordingSha256"] != e["recordingSha256"] or c["contextKey"] != e["contextKey"]:
            raise ExperimentError(f"{e['id']}: the recording or context differs")
        check_context(e, S.recording_point(root, e["recording"], e["point"]), pvs[c["id"]])
    if points_manifest(cands, chosen, m["contextsFileSha256"]) != m:
        raise ExperimentError("the manifest is not the one the points make")
    return len(contexts)


# --- the schedule and the live run ------------------------------------------------

def load_protocol(path):
    with open(path, "rb") as f:
        raw = f.read()
    proto = json.loads(raw)
    if proto.get("schema") != PROTOCOL_SCHEMA:
        raise ExperimentError(f"{path} is not a {PROTOCOL_SCHEMA} protocol")
    d = os.path.join(os.path.dirname(os.path.abspath(path)), proto["points"])
    man, contexts = load_points(d)
    return proto, sha256(raw), man, contexts


def schedule(proto, contexts):
    """One request at a time. Per model (in the declared order), draws
    1 … N as rounds; each round has every point and arm of the model
    once, in the order of SHA-256('<orderRule>|<round>|<point>|<arm>')."""
    rule, out = proto["orderRule"], []
    for m in proto["modelOrder"]:
        ids = [e["id"] for e in contexts if e["model"] == m]
        for r in range(1, proto["drawsPerArm"] + 1):
            cells = sorted(((p, a) for p in ids for a in ARMS), key=lambda x: sha256(f"{rule}|{r}|{x[0]}|{x[1]}"))
            for p, a in cells:
                out.append({"seq": len(out) + 1, "model": m, "point": p, "arm": a, "draw": r})
    return out


def schedule_doc(proto, proto_sha, man, sched):
    return {"schema": SCHEDULE_SCHEMA, "protocol": proto["id"], "protocolSha256": proto_sha,
            "contextsFileSha256": man["contextsFileSha256"], "requests": len(sched),
            "concurrency": 1, "schedule": sched}


def server_signature(state, models=tuple(S.MODELS.values())):
    """What would be a material change: the server's version, its API, or
    either experiment model's entry (another model added or removed on the
    shared server is not)."""
    return {"serverVersion": state.get("serverVersion"), "openapiSha256": state.get("openapiSha256"),
            "models": sorted(canon(m) for m in state.get("models") or [] if m.get("id") in models)}


def run(proto, proto_sha, man, contexts, out, base, post_fn=RR.post, state_fn=RR.server_state,
        now=time.monotonic):
    """The live run: the schedule's requests one at a time, each response
    recorded as it ends; no retry, nothing dropped."""
    sched = schedule(proto, contexts)
    by_id = {e["id"]: e for e in contexts}
    live = os.path.join(out, "live")
    os.makedirs(live, exist_ok=True)
    stop = proto["stop"]
    before = state_fn(base)
    write_json(os.path.join(live, "server-before.json"), before)
    sig = server_signature(before)
    t0, started, fails, done, stopped, checks = now(), RR.utc(), 0, 0, None, []
    if (before.get("health") or {}).get("status") != "healthy":
        stopped = "health: the server did not answer healthy before the run"
    with open(os.path.join(live, "requests.jsonl"), "w", encoding="utf-8") as f:
        for r in sched if stopped is None else []:
            if r["draw"] != (sched[r["seq"] - 2]["draw"] if r["seq"] > 1 else None) or r["seq"] == 1:
                st = state_fn(base)
                checks.append({"seq": r["seq"], "model": r["model"], "draw": r["draw"], "time": st.get("time"),
                               "health": st.get("health"), "sameConfiguration": server_signature(st) == sig})
                if (st.get("health") or {}).get("status") != "healthy":
                    stopped = f"health: the server did not answer healthy before request {r['seq']}"
                    break
                if server_signature(st) != sig:
                    stopped = f"server configuration changed before request {r['seq']}"
                    break
            if now() - t0 > stop["wallClockSeconds"]:
                stopped = "wall clock"
                break
            e = by_id[r["point"]]
            body = arm_request(e["request"], e["proposal"]["content"], e["arms"][r["arm"]]["text"])
            data = body_bytes(body)
            if sha256(data) != e["arms"][r["arm"]]["requestSha256"]:
                raise ExperimentError(f"{r['point']} {r['arm']}: the request is not the declared one")
            t = RR.utc()
            status, rbody, err, lat = post_fn(base, data, proto["request"]["timeoutSeconds"])
            res = dict(r, sent=t, requestSha256=sha256(data), status=status, error=err, latencyMs=round(lat * 1000),
                       body=rbody, **RR.response_fields(rbody))
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
           "sameConfigurationAfter": server_signature(after) == sig,
           "endpoint": "from " + proto["endpointEnv"] + " (not recorded)", "client": {"python": sys.version.split()[0]}}
    write_json(os.path.join(live, "run.json"), doc)
    return doc


# --- the analysis -------------------------------------------------------------------

def classify(row, proposal_actions, types):
    """A draw's final decision: (outcome, reason, actions or None)."""
    if row.get("error") or row.get("status") != 200 or not row.get("parsedBody"):
        return "invalid", "request_failed", None
    if row.get("content") is None:
        return "invalid", "no_content", None
    try:
        d = A3.parse_decision(row["content"], types, False)
    except ValueError as e:
        return "invalid", str(e).split(":", 1)[0], None
    return A4.response_of(proposal_actions, d["actions"]), None, d["actions"]


def changes(proposal, final):
    """What a revision changed, action by action: an action matched by
    type and its employee, request or role; the fields that differ."""
    def ident(a):
        return (a.get("type"), a.get("employee") or a.get("role") or a.get("op"))
    left = list(proposal)
    out = {"removed": [], "added": [], "changed": [], "unchanged": 0}
    for a in final:
        m = next((p for p in left if canon(p) == canon(a)), None) or next((p for p in left if ident(p) == ident(a)),
                                                                          None)
        if m is None:
            out["added"].append(a.get("type"))
            continue
        left.remove(m)
        diff = sorted(k for k in set(m) | set(a) if k != "requestId" and m.get(k) != a.get(k))
        if diff:
            out["changed"].append({"type": a.get("type"), "fields": diff})
        else:
            out["unchanged"] += 1
    out["removed"] = [p.get("type") for p in left]
    return out


def outcome_of(acts, validation):
    if not acts:
        return "no_action"
    return "accepted" if all(v.get("valid") for v in validation) else "refused"


def decision_evidence(binary, run_dir, e, acts, catalog_check=None):
    """The decision at the point, then the declared suffix (no action):
    each batch previewed (staffing) and validated; if every batch is valid,
    each applied in order with the engine's observation after it; then
    every later day advanced with no action. Financial evidence v1's
    AFFORDABILITY, REVERSIBILITY and OPERATIONAL_EXPOSURE records for it."""
    ses = open_session(binary, run_dir)
    try:
        obs, catalog = ses.to_point(e["digests"])
        if catalog_check is not None:
            catalog_check(catalog)
        bs = [b for b, _ in A3.split(acts)]
        events, validation = [], []
        for b in bs:
            ev = {"op": "validate", "actions": b}
            if FE.staffing_batch(b):
                pv = ses.request("v10b-preview", "preview", {"actions": b})
                ev["preview"] = (pv.get("result") or {}).get("preview") if pv.get("ok") else None
                ev["previewError"] = None if pv.get("ok") else (pv.get("error") or {}).get("code")
            v = ses.request("v10b-validate", "validate", {"actions": b})
            if not v.get("ok"):
                raise ExperimentError(f"{e['id']}: validate answered {(v.get('error') or {}).get('code')}")
            ev["answer"] = v["result"]
            events.append(ev)
            validation.append({"valid": bool(v["result"].get("valid")), "refused": sorted(
                r.get("code") for r in v["result"].get("results") or [] if not r.get("ok"))})
        applied, agree = [], True
        if bs and all(x["valid"] for x in validation):
            for b, vev in zip(bs, list(events)):
                a = ses.request("v10b-apply", "apply", {"actions": b})
                if not a.get("ok"):
                    raise ExperimentError(f"{e['id']}: apply answered {(a.get('error') or {}).get('code')}")
                res = a["result"]
                o = ses.request("v10b-observe", "observe", {"compact": True, "recentLog": 0})
                ev = {"op": "apply", "actions": b, "answer": res,
                      "fundingAfter": FE.funding_obs((o.get("result") or {}).get("observation"))}
                events.append(ev)
                codes = [r.get("code") for r in res.get("results") or []]
                agree = agree and bool(res.get("applied")) and codes == [r.get("code") for r in
                                                                          vev["answer"].get("results") or []]
                applied.append({"applied": bool(res.get("applied")), "codes": codes})
        after = ses.request("v10b-digests", "digests")["result"]["digests"]
        advances = ses.later_advances()
        run_end = advances[-1]["args"]["to"] if advances else None
        suffix = []
        for r in advances:
            a = ses.ask(r)
            if not a.get("ok"):
                raise ExperimentError(f"{e['id']}: advance answered {(a.get('error') or {}).get('code')}")
            suffix.append({"to": r["args"]["to"],
                           "digests": ses.request("v10b-digests", "digests")["result"]["digests"]})
        end = ses.request("v10b-observe", "observe", {"compact": True, "recentLog": 0})
        end_obs = (end.get("result") or {}).get("observation")
    finally:
        ses.close()
    an = FE.Analysis()
    ctx = {"recording": {"path": e["recording"], "group": e["id"]}, "point": {"index": e["point"]},
           "run": {"scenario": ses.run["scenario"]["ref"], "seed": ses.run["seed"]}, "attempt": {"id": None},
           "attemptIdx": None, "format": VERSION, "catalog": catalog, "events": events}
    # Submission j (1-based) is batch j: its validate, then its apply.
    for k, ev in enumerate(events):
        ctx["eventIndex"] = k
        if ev["op"] == "validate":
            FE.affordability(an, ctx, k + 1, ev, obs, None)
        elif ev["answer"].get("applied"):
            j = k - len(bs)
            FE.executed(an, ctx, j + 1, ev, events[j], obs, run_end)
    return {"validation": validation, "outcome": outcome_of(acts, validation), "applied": applied,
            "validateApplyAgree": agree if applied else None, "digestsAfterDecision": after,
            "suffix": {"kind": "no_action", "days": len(suffix), "endDigests": suffix[-1]["digests"] if suffix else
                       after, "digestsSha256": sha256(canon([s["digests"] for s in suffix]))},
            "observedEnd": FE.observed_state(end_obs), "financialEvidence": fe_summary(an.records),
            "records": an.records}


def commitment(r):
    m = r.get("measurements") or {}
    c = (m.get("engineFunding") or {}).get("commitmentCents")
    return m.get("extensionCommitmentCents") if c is None else c


def fe_summary(records):
    aff = [r for r in records if r["dimension"] == "AFFORDABILITY"]
    rev = [r for r in records if r["dimension"] == "REVERSIBILITY"]
    exp = [r for r in records if r["dimension"] == "OPERATIONAL_EXPOSURE"]
    cs = [commitment(r) for r in aff if r["execution"] == "applied" and commitment(r) is not None]
    return {"affordability": [r["status"] for r in aff],
            "commitmentCents": sum(cs) if cs else 0,
            "creditSupported": any(r["status"] == "funded_with_credit" and r["execution"] == "applied" for r in aff),
            "reversibility": dict(sorted(collections.Counter(r["status"] for r in rev).items())),
            "directExposure": dict(sorted(collections.Counter(
                (r.get("directChange") or {}).get("resource") or r["status"] for r in exp
                if r["status"] == "direct_change").items()))}


def wilson(k, n):
    if n == 0:
        return None
    p, z2 = k / n, Z95 * Z95
    c = (p + z2 / (2 * n)) / (1 + z2 / n)
    h = Z95 * math.sqrt(p * (1 - p) / n + z2 / (4 * n * n)) / (1 + z2 / n)
    return [round(max(0.0, c - h), 4), round(min(1.0, c + h), 4)]


def newcombe(k1, n1, k0, n0):
    """The difference p1 − p0 and its hybrid score interval (Newcombe 1998,
    method 10), from the arms' Wilson intervals."""
    if n1 == 0 or n0 == 0:
        return None
    p1, p0 = k1 / n1, k0 / n0
    (l1, u1), (l0, u0) = wilson(k1, n1), wilson(k0, n0)
    d = p1 - p0
    return {"difference": round(d, 4), "interval": [round(d - math.sqrt((p1 - l1) ** 2 + (u0 - p0) ** 2), 4),
                                                    round(d + math.sqrt((u1 - p1) ** 2 + (p0 - l0) ** 2), 4)]}


def parsed_canon(content):
    if content is None:
        return "no_content"
    try:
        d = json.loads(A3.strip_wrapping(content))
    except ValueError:
        return "unparsable"
    return canon(d) if isinstance(d, dict) else "not_an_object"


def modal(values):
    c = collections.Counter(values)
    return {"distinct": len(c), "modalCount": max(c.values()) if c else 0}


def check_rows(proto, contexts, rows):
    """The recorded requests are the schedule's, in its order, each the
    declared request byte for byte (rebuilt from the points): {point:
    rows}."""
    sched = schedule(proto, contexts)
    if [(r["seq"], r["point"], r["arm"], r["draw"]) for r in rows] != \
            [(s["seq"], s["point"], s["arm"], s["draw"]) for s in sched[:len(rows)]]:
        raise ExperimentError("the live requests are not the schedule's, in its order")
    by_id, by_point = {e["id"]: e for e in contexts}, collections.defaultdict(list)
    for r in rows:
        e = by_id[r["point"]]
        b = arm_request(e["request"], e["proposal"]["content"], e["arms"][r["arm"]]["text"])
        if r["requestSha256"] != sha256(body_bytes(b)) or r["requestSha256"] != e["arms"][r["arm"]]["requestSha256"]:
            raise ExperimentError(f"request {r['seq']}: not the declared request")
        by_point[r["point"]].append(r)
    return by_point


def analyze(binary, d, replays):
    proto_path = os.path.join(d, "protocol.json")
    proto, proto_sha, man, contexts = load_protocol(proto_path)
    with open(os.path.join(d, "live", "run.json"), encoding="utf-8") as f:
        run_doc = json.load(f)
    rows = load_jsonl(os.path.join(d, "live", "requests.jsonl"))
    if run_doc["protocolSha256"] != proto_sha or run_doc["contextsFileSha256"] != man["contextsFileSha256"]:
        raise ExperimentError("the live run is not of this protocol and these points")
    sched = schedule(proto, contexts)
    by_point = check_rows(proto, contexts, rows)
    runs = replay_runs(replays)
    points, draws = [], []
    for e in contexts:
        run_dir = runs[os.path.basename(e["recording"])]
        ses = open_session(binary, run_dir)
        try:
            _, catalog = ses.to_point(e["digests"])
        finally:
            ses.close()
        types = {t["type"] for t in catalog.get("types", [])}
        prop = e["proposal"]["actions"]
        rs = by_point[e["id"]]
        cls = [(r, *classify(r, prop, types)) for r in rs]
        lists = {canon(prop): prop, canon([]): []}
        for _, o, _, acts in cls:
            if acts is not None:
                lists.setdefault(canon(acts), acts)
        ev = {k: decision_evidence(binary, run_dir, e, v) for k, v in sorted(lists.items())}
        none_key = canon([])
        decisions = []
        for k in sorted(lists):
            x = ev[k]
            decisions.append({"key": sha256(k), "actions": lists[k], "isProposal": k == canon(prop),
                              **{f: x[f] for f in ("validation", "outcome", "applied", "validateApplyAgree",
                                                   "digestsAfterDecision", "suffix", "observedEnd",
                                                   "financialEvidence", "records")}})
        prop_fe = ev[canon(prop)]["financialEvidence"]
        for r, o, why, acts in cls:
            k = canon(acts) if acts is not None else none_key
            x = ev[k]
            # A refused decision is not submitted: the agent would have
            # retried, and this experiment allows no retry. Nothing is
            # applied, as at the end of the agent's retries.
            fe = x["financialEvidence"]
            draws.append({"seq": r["seq"], "point": e["id"], "model": e["model"], "arm": r["arm"], "draw": r["draw"],
                          "outcome": o, "invalidReason": why, "finishReason": r.get("finishReason"),
                          "rawSha256": sha256(r["content"]) if r.get("content") is not None else None,
                          "parsedSha256": sha256(parsed_canon(r.get("content"))),
                          "actionsKey": sha256(k) if acts is not None else None,
                          "final": x["outcome"] if acts is not None else "invalid",
                          "changes": changes(prop, acts) if o == "revised" else None,
                          "commitmentChangeCents": fe["commitmentCents"] - prop_fe["commitmentCents"]
                          if acts is not None else None,
                          "endDigestsSha256": sha256(canon(x["suffix"]["endDigests"] if x["outcome"] != "refused"
                                                           else ev[none_key]["suffix"]["endDigests"]))})
        points.append({"id": e["id"], "model": e["model"], "stratum": e["stratum"], "refusals": e["refusals"],
                       "shownPreview": e["shownPreview"], "recording": e["recording"], "point": e["point"],
                       "proposalSha256": e["proposal"]["sha256"], "proposal": prop,
                       "previewSha256": e["preview"]["sha256"],
                       "arms": {a: {k: e["arms"][a][k] for k in ("textSha256", "textChars", "requestSha256",
                                                                  "requestChars")} for a in ARMS},
                       "decisions": decisions})
    return {"schema": ANALYSIS_SCHEMA, "analyzer": ANALYZER, "protocol": proto["id"], "protocolSha256": proto_sha,
            "contextsFileSha256": man["contextsFileSha256"], "requestsScheduled": len(sched),
            "requestsAnalyzed": len(rows), "stopped": run_doc["stopped"], "points": points, "draws": draws,
            "summary": summarize(contexts, draws, proto), "notes": NOTES}


NOTES = [
    "Per point and arm, every scheduled draw is in the denominator; an invalid reply or a failed request is an "
    "outcome, not a dropped draw.",
    "Intervals are Wilson 95% score intervals for a proportion and Newcombe hybrid score intervals for a "
    "difference (treatment minus control). They are descriptive: each treats the point's draws as independent "
    "repetitions of one request, which Part A of V0.10a does not establish.",
    "A refused final decision is not submitted (no retry in this experiment); its continuation is the no-action one.",
    "Financial evidence v1's definitions, unchanged, at the recorded state; long-run outcomes under the fixed "
    "suffix are a mechanics check and are not attributed to the preview.",
]


def summarize(contexts, draws, proto):
    n_per = proto["drawsPerArm"]
    out = {"points": [], "pooled": {}}
    for e in contexts:
        row = {"id": e["id"], "model": e["model"], "stratum": e["stratum"], "arms": {}, "differences": {},
               "blocked": None}
        for a in ARMS:
            ds = [x for x in draws if x["point"] == e["id"] and x["arm"] == a]
            n = n_per
            cnt = collections.Counter(x["outcome"] for x in ds)
            fin = collections.Counter(x["final"] for x in ds)
            com = sorted(x["commitmentChangeCents"] for x in ds if x["commitmentChangeCents"] is not None)
            row["arms"][a] = {
                "draws": n, "observed": len(ds), "notRun": n - len(ds),
                "outcomes": {o: {"count": cnt.get(o, 0), "proportion": round(cnt.get(o, 0) / n, 4),
                                 "wilson95": wilson(cnt.get(o, 0), n)} for o in OUTCOMES},
                "invalidReasons": dict(sorted(collections.Counter(x["invalidReason"] for x in ds
                                                                  if x["outcome"] == "invalid").items())),
                "final": dict(sorted(fin.items())),
                "distinctDecisions": len({x["actionsKey"] or ("invalid:" + str(x["invalidReason"])) for x in ds}),
                "distinctActionProposals": len({x["actionsKey"] for x in ds if x["actionsKey"]}),
                "agreement": {"raw": modal([x["rawSha256"] for x in ds]),
                              "canonicalJson": modal([x["parsedSha256"] for x in ds]),
                              "actions": modal([x["actionsKey"] or "invalid" for x in ds])},
                "commitmentChangeCents": {"draws": len(com), "min": com[0] if com else None,
                                          "median": com[len(com) // 2] if com else None,
                                          "max": com[-1] if com else None},
            }
        c, t = row["arms"]["control"]["outcomes"], row["arms"]["treatment"]["outcomes"]
        row["differences"] = {o: newcombe(t[o]["count"], n_per, c[o]["count"], n_per) for o in OUTCOMES}
        # Blocked: control and treatment draws of the same round (the
        # schedule places both in one round, in random order).
        pairs = collections.Counter()
        for r in range(1, n_per + 1):
            cd = next((x["outcome"] for x in draws if x["point"] == e["id"] and x["arm"] == "control" and
                       x["draw"] == r), "not_run")
            td = next((x["outcome"] for x in draws if x["point"] == e["id"] and x["arm"] == "treatment" and
                       x["draw"] == r), "not_run")
            pairs[f"{cd}|{td}"] += 1
        row["blocked"] = {"rounds": n_per, "pairs": dict(sorted(pairs.items())),
                          "discordant": sum(v for k, v in pairs.items() if k.split("|")[0] != k.split("|")[1])}
        out["points"].append(row)
    for scope in sorted(QUOTA) + ["all"]:
        rows = [r for r in out["points"] if scope in ("all", r["model"])]
        out["pooled"][scope] = {
            "points": len(rows),
            "perPointDifferences": {o: [r["differences"][o]["difference"] for r in rows] for o in OUTCOMES},
            "unweightedMean": {o: round(sum(r["differences"][o]["difference"] for r in rows) / len(rows), 4)
                               for o in OUTCOMES} if rows else None,
            "pointsPositive": {o: sum(1 for r in rows if r["differences"][o]["difference"] > 0) for o in OUTCOMES},
            "pointsNegative": {o: sum(1 for r in rows if r["differences"][o]["difference"] < 0) for o in OUTCOMES},
            "interval": "none: the points are clusters, and too few for an interval over them"}
    return out


def inference(d):
    """Live metadata, kept apart from the deterministic analysis: tokens,
    latency, cache and finish reasons per point and arm."""
    rows = load_jsonl(os.path.join(d, "live", "requests.jsonl"))
    out = {}
    groups = collections.defaultdict(list)
    for r in rows:
        groups[(r["point"], r["arm"])].append(r)
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


def pct(x):
    return f"{100 * x:.0f}%"


def iv(w):
    return f"[{pct(w[0])}, {pct(w[1])}]" if w else "n/a"


def money(c):
    if c is None:
        return "n/a"
    s = "−" if c < 0 else ""
    return f"{s}${abs(c) / 100:,.2f}"


def tables(doc):
    s = doc["summary"]
    L = ["# V0.10b decision distributions", "",
         "Each point and arm: 20 scheduled draws, all in the denominator. Wilson 95% intervals; differences are "
         "treatment − control with Newcombe 95% intervals. Descriptive only.", ""]
    L += ["## Per point", "",
          "| Point | Model | Preview | Arm | Kept | Revised | Abandoned | Invalid | Final accepted / refused / "
          "no action | Distinct decisions |", "|---|---|---|---|---|---|---|---|---|---|"]
    for r in s["points"]:
        for a in ARMS:
            x = r["arms"][a]
            o = x["outcomes"]
            f = x["final"]
            L.append(f"| {r['id']} | {MODEL_SHORT[r['model']]} | {r['stratum']} | {a} | " + " | ".join(
                f"{o[k]['count']}/{x['draws']} {iv(o[k]['wilson95'])}" for k in OUTCOMES) +
                f" | {f.get('accepted', 0)} / {f.get('refused', 0)} / {f.get('no_action', 0)} | "
                f"{x['distinctDecisions']} |")
    L += ["", "## Differences (treatment − control)", "",
          "| Point | Kept | Revised | Abandoned | Invalid | Discordant rounds |", "|---|---|---|---|---|---|"]
    for r in s["points"]:
        L.append(f"| {r['id']} | " + " | ".join(
            f"{100 * r['differences'][k]['difference']:+.0f} pp {iv(r['differences'][k]['interval'])}"
            for k in OUTCOMES) + f" | {r['blocked']['discordant']}/{r['blocked']['rounds']} |")
    L += ["", "## Pooled (after the per-point results)", "",
          "Unweighted means of the per-point differences; the points are clusters, and there are too few for an "
          "interval over them.", "",
          "| Scope | Points | Kept | Revised | Abandoned | Invalid | Points with kept higher / lower under treatment |",
          "|---|---|---|---|---|---|---|"]
    for k, p in s["pooled"].items():
        m = p["unweightedMean"]
        L.append(f"| {MODEL_SHORT.get(k, k)} | {p['points']} | " + " | ".join(f"{100 * m[o]:+.1f} pp" for o in OUTCOMES)
                 + f" | {p['pointsPositive']['kept']} / {p['pointsNegative']['kept']} |")
    L += ["", "## Agreement within an arm (distinct values / modal count)", "",
          "| Point | Arm | Raw | Canonical JSON | Actions |", "|---|---|---|---|---|"]
    for r in s["points"]:
        for a in ARMS:
            g = r["arms"][a]["agreement"]
            L.append(f"| {r['id']} | {a} | " + " | ".join(f"{g[k]['distinct']} / {g[k]['modalCount']}"
                                                          for k in ("raw", "canonicalJson", "actions")) + " |")
    L += ["", "## Financial evidence v1 of the distinct final decisions", "",
          "| Point | Proposal | Draws (control / treatment) | Outcome | Commitment | Credit-supported | "
          "Reversibility | Direct exposure |", "|---|---|---|---|---|---|---|---|"]
    for p in doc["points"]:
        for dd in p["decisions"]:
            nc = sum(1 for x in doc["draws"] if x["point"] == p["id"] and x["actionsKey"] == dd["key"] and
                     x["arm"] == "control")
            nt = sum(1 for x in doc["draws"] if x["point"] == p["id"] and x["actionsKey"] == dd["key"] and
                     x["arm"] == "treatment")
            fe = dd["financialEvidence"]
            L.append(f"| {p['id']} | {'yes' if dd['isProposal'] else ''} | {nc} / {nt} | {dd['outcome']} | "
                     f"{money(fe['commitmentCents'])} | {'yes' if fe['creditSupported'] else 'no'} | "
                     f"{', '.join(f'{k} {v}' for k, v in fe['reversibility'].items()) or '—'} | "
                     f"{', '.join(f'{k} {v}' for k, v in fe['directExposure'].items()) or '—'} |")
    return "\n".join(L) + "\n"


# --- main -----------------------------------------------------------------------------

def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
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
        if x.cmd == "points":
            if x.check:
                print(f"shared-prefix points: {check_points(x.binary, x.replays, x.check)} contexts, selection, "
                      "previews and requests verified")
            else:
                m = make_points(x.binary, x.replays, x.live, x.out)
                print(f"shared-prefix points: {len(m['selected'])} of {len(m['candidates'])} candidates")
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
        if x.cmd == "run":
            base = os.environ.get(proto["endpointEnv"])
            if not base:
                raise ExperimentError(f"set {proto['endpointEnv']}")
            doc = run(proto, proto_sha, man, contexts, x.out, base)
            print(f"decision experiment: {doc['requestsSent']} of {doc['requestsScheduled']} requests"
                  + (f"; stopped: {doc['stopped']}" if doc["stopped"] else ""))
            return 0 if not doc["stopped"] else 1
    except (ExperimentError, S.SampleError) as e:
        print(f"decision experiment: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
