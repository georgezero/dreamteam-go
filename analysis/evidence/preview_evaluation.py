#!/usr/bin/env python3
"""The V0.9c prospective preview evaluation (dreamteam.preview-evaluation/v1):
the outcomes and analyses declared in docs/benchmark/v09c-dev/PLAN.md §8-§9.

    python3 -I -B analysis/evidence/preview_evaluation.py --binary ./dt \
        --dir docs/benchmark/v09c-dev --replays <dir> --out <dir> [--jobs N]

Two conditions on the same models, entries, seeds and horizon: llm-agent 3
(the baseline, dreamteam.llm-decision-record/v3) and llm-agent 4 (the
preview step, /v4). Everything is read from committed files (the plan's
corpus, the evaluation's recordings and contract report) and from the
replays CI makes of them; no model is contacted.

* Preview behaviour (A, B) comes from the records themselves.
* Financial commitments, reversibility and direct exposure (C-E) come from
  dreamteam.financial-evidence/v1, run unchanged on both conditions'
  recordings. A v4 record is given to it as its v3 view (its attempts have
  the v3 fields the analyzer links); the run list keeps each recording's
  own SHA-256 and format.
* Run outcomes (F, G) come from the contract report and the replays.

Direct effects are kept apart from run outcomes, the paired comparison
(preview-enabled minus baseline on the same model, entry and seed) apart
from the comparison with no action, and nothing is combined into a score.
"""

import argparse
import json
import os
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import decision_evidence as D  # noqa: E402  (corpus helpers, read-only)
import financial_evidence as FE  # noqa: E402  (dreamteam.financial-evidence/v1, unchanged)
import financial_outcomes as FO  # noqa: E402  (metric helpers, read-only)

SCHEMA = "dreamteam.preview-evaluation/v1"
ANALYZER = "preview-evaluation 1"
V3 = "dreamteam.llm-decision-record/v3"
V4 = "dreamteam.llm-decision-record/v4"
STAFFING = FE.STAFFING
PREVIEW_STATUSES = ("not_applicable", "ok", "failed", "too_large")
RESPONSES = ("kept", "revised", "abandoned", "no_reply")
FINALS = ("kept", "revised", "abandoned", "fallback")
PARTS = ("kept", "revised", "dropped", "added", "none proposed")
OUTCOMES = [("financial", "operatingProfitCents"), ("operational", "examsCompleted"),
            ("staffingCapacity", "ordersLost"), ("serviceReliability", "tatBreaches"),
            ("operational", "tatP90Sec"), ("sustainability", "minCashCents"), ("sustainability", "shortfallCents"),
            ("sustainability", "daysBelowZero"), ("serviceReliability", "displacedAppointments"),
            ("financial", "netRevenueCents"), ("financial", "operatingExpenseCents")]


def count(xs, keys=None):
    out = {k: 0 for k in keys or ()}
    for x in xs:
        out[x] = out.get(x, 0) + 1
    return out if keys else {k: out[k] for k in sorted(out, key=str)}


class V3View(D.Inputs):
    """Inputs that give financial evidence v1 a v4 record as its v3 view,
    and record each file's own SHA-256."""

    def __init__(self, replays):
        super().__init__(replays)
        self.formats = {}

    def read(self, path):
        b = super().read(path)
        if path.endswith("-decisions.jsonl"):
            lines = b.decode("utf-8").splitlines()
            recs = [json.loads(l) for l in lines if l.strip()]
            if recs and recs[0].get("schema") == V4:
                self.formats[os.path.normpath(path)] = "llm-agent 4"
                return "".join(json.dumps(dict(r, schema=V3), separators=(",", ":")) + "\n" for r in recs).encode()
        return b


def preview_checks_removed(run_dir, scratch):
    """The run's transcript without llm-agent 4's own digests request after
    each preview (its check that the preview changed nothing). Financial
    evidence v1 starts a decision point at each digests request that
    carries the point's digests, so that check would end the point early.
    The request is read-only: removing it changes no other answer, and the
    engine replay still requires every remaining answer identical. Returns
    the directory to replay and how many requests were removed."""
    with open(os.path.join(run_dir, "transcript.jsonl"), encoding="utf-8") as f:
        lines = f.readlines()
    out, removed, last_op, skip_answer = [], 0, None, False
    for line in lines:
        d = json.loads(line)
        if "op" in d:
            if d["op"] == "digests" and last_op == "preview":
                removed += 1
                skip_answer = True
                last_op = d["op"]
                continue
            last_op = d["op"]
        elif skip_answer:
            skip_answer = False
            continue
        out.append(line)
    if not removed:
        return run_dir, 0
    d = os.path.join(scratch, os.path.basename(run_dir))
    os.makedirs(d, exist_ok=True)
    with open(os.path.join(d, "transcript.jsonl"), "w", encoding="utf-8") as f:
        f.writelines(out)
    return d, removed


def financial(binary, corpus_path, replays, jobs):
    """dreamteam.financial-evidence/v1, unchanged, on the corpus: v4 records
    as their v3 view, and transcripts without the preview checks."""
    inputs = V3View(replays)
    original, removed = FE.engine_replay, {}
    scratch = tempfile.TemporaryDirectory(prefix="pe-transcripts-")

    def engine_replay(binary_, run_dir, run, digests, extras=None):
        d, n_ = preview_checks_removed(run_dir, scratch.name)
        removed[run_dir] = n_
        return original(binary_, d, run, digests, extras)
    FE.engine_replay = engine_replay
    try:
        doc = FE.analyze(binary, corpus_path, replays, inputs, jobs)
    finally:
        FE.engine_replay = original
        scratch.cleanup()
    for e in doc["runs"]:
        rd = (e.get("run") or {}).get("replayRun")
        if rd is not None and e.get("replay"):
            e["replay"]["previewChecksRemoved"] = removed.get(os.path.join(replays, rd), 0)
    for e in doc["runs"]:
        p = os.path.normpath(e["recording"]["path"])
        if p in inputs.formats:
            e["recording"]["format"] = inputs.formats[p] + " (read as its v3 view)"
        e["recording"]["sha256"] = inputs.files[e["recording"]["path"]]
    doc["inputs"] = [{"path": p, "sha256": h} for p, h in sorted(inputs.files.items())]
    return doc


# --- the records ---------------------------------------------------------------------------------

def first_proposal(rec):
    """The first parsed proposal at a decision point (not a details
    request): its attempt and actions, or None."""
    for a in rec.get("attempts") or []:
        if (a.get("parse") or {}).get("ok") and a.get("detailsRequested") is None:
            return a
    return None


def kinds(actions):
    return {"staffing": sum(1 for a in actions if a.get("type") in STAFFING),
            "operations": sum(1 for a in actions if a.get("type") not in STAFFING)}


def preview_facts(answer):
    """What a shown preview said, in the engine's own fields: the funding
    test, whether it refused an action, and the bookings it displaces."""
    pv = (answer or {}).get("preview") or {}
    f = pv.get("funding") or {}
    res = pv.get("results") or []
    return {"ok": pv.get("ok"), "refusedCodes": sorted({r.get("code") for r in res if not r.get("ok")}),
            "commitmentCents": f.get("commitmentCents"), "headroomCents": f.get("headroomCents"),
            "funded": f.get("funded"),
            "displacedBookings": sum(t.get("displaced") or 0 for t in pv.get("terminations") or []),
            "terminations": len(pv.get("terminations") or []), "hires": len(pv.get("hires") or []),
            "allocations": len(pv.get("allocations") or []), "rosters": len(pv.get("rosters") or []),
            "teleradiology": 1 if pv.get("teleradiology") else 0}


def record_metrics(recs):
    """Outcomes 1-9 that the records hold, and the preview step's details."""
    points = len(recs)
    first = [first_proposal(r) for r in recs]
    proposed_any = sum(1 for a in first if a and a["actions"])
    proposed_staff = sum(1 for a in first if a and kinds(a["actions"])["staffing"])
    attempts = [len(r.get("attempts") or []) for r in recs]
    proposed_actions = sum(len(a["actions"]) for r in recs for a in r.get("attempts") or []
                           if (a.get("parse") or {}).get("ok") and a.get("detailsRequested") is None)
    refusals = [x for r in recs for a in r.get("attempts") or [] for x in a.get("refusals") or []]
    real = [x for x in refusals if x.get("code") != "batch_rejected"]
    submitted = [a for r in recs for b in r.get("batches") or [] for a in b]
    out = {"decisionPoints": points,
           "outcomes": count(r.get("outcome") for r in recs),
           "initialProposals": {"withAction": proposed_any, "withStaffingAction": proposed_staff,
                                "denominator": "decision points"},
           "attemptsPerPoint": {"total": sum(attempts), "max": max(attempts or [0]),
                                "distribution": count(attempts)},
           "proposedActions": proposed_actions,
           "validateRefusals": {"actions": len(refusals), "byCode": count(x.get("code") for x in refusals),
                                "fundingRefusals": sum(1 for x in real if x.get("code") == "insufficient_funds"),
                                "denominator": "actions in parsed proposals (every attempt)"},
           "submittedActions": {"total": len(submitted), **kinds(submitted), "denominator": "decision points"},
           "completeness": {"pointsAttemptsRecorded": sum(1 for r in recs if (r.get("completeness") or {}).get(
               "attemptsRecorded")), "pointsInputsVerifiable": sum(1 for r in recs if (r.get("completeness") or {}).get(
                   "inputsVerifiable"))}}
    if recs and recs[0].get("schema") == V4:
        steps = [r.get("preview") or {} for r in recs]
        shown = [s for s in steps if s.get("status") == "ok"]
        previews = []
        for r in recs:
            s = r.get("preview") or {}
            if s.get("status") in (None, "not_applicable"):
                continue
            at = next(a for a in r["attempts"] if a.get("preview"))
            pv = at["preview"]
            previews.append({"point": r["point"], "time": r["time"], "status": s["status"],
                             "proposal": [a.get("type") for a in at["actions"]],
                             "previewed": [a.get("type") for a in pv["actions"]],
                             "messageChars": (pv.get("message") or {}).get("chars"),
                             "digestsUnchanged": pv.get("digestsUnchanged"),
                             "error": (pv.get("error") or {}).get("code"),
                             "facts": preview_facts(pv.get("answer")) if pv.get("answer") is not None else None,
                             "response": s.get("response"), "staffing": s.get("staffing"),
                             "operations": s.get("operations"), "final": s.get("final"),
                             "reasonBefore": at.get("reason"),
                             "reasonAfter": next((a.get("reason") for a in r["attempts"]
                                                  if a["attempt"] == s.get("responseAttempt")), None)})
        chars = [p["messageChars"] for p in previews if p["status"] == "ok"]
        out["preview"] = {
            "eligiblePoints": sum(1 for s in steps if s.get("status") in ("ok", "failed", "too_large")),
            "status": count((s.get("status") for s in steps), PREVIEW_STATUSES),
            "shownWithRefusedAction": sum(1 for p in previews if p["status"] == "ok" and p["facts"]["refusedCodes"]),
            "response": count((s.get("response") for s in shown), RESPONSES),
            "staffing": count((s.get("staffing") for s in shown), PARTS),
            "operations": count((s.get("operations") for s in shown), PARTS),
            "final": count((s.get("final") for s in shown), FINALS),
            "reviewRequests": sum(1 for r in recs for a in r.get("attempts") or [] if a.get("turn") == "review"),
            "reviewMessageChars": {"total": sum(chars), "max": max(chars or [0])},
            "digestsUnchangedAll": all(p["digestsUnchanged"] for p in previews),
            "previews": previews}
    return out


# --- the financial evidence, by run -------------------------------------------------------------

def validated_batches(recs):
    """Batches the records say were validated (with an answer): each is a
    submission financial evidence v1 must have found."""
    return sum(len(a.get("validation") or []) for r in recs for a in r.get("attempts") or [])


def fin_metrics(records):
    aff = [r for r in records if r["dimension"] == "AFFORDABILITY"]
    rev = [r for r in records if r["dimension"] == "REVERSIBILITY"]
    exp = [r for r in records if r["dimension"] == "OPERATIONAL_EXPOSURE"]
    funded = [r for r in aff if r["status"] in ("funded_without_credit", "funded_with_credit")]
    commit = [FE.commitment_of(r) for r in funded]  # (cents, "staffing" or "extension")
    displaced = {}
    for r in exp:
        d = r.get("directChange") or {}
        if "displacedBookingsInBatch" in d:
            displaced[(r["decision"]["index"], r["submission"])] = d["displacedBookingsInBatch"]
    return {"affordability": {"status": count((r["status"] for r in aff), FE.AFFORDABILITY),
                              "denominator": "submitted batches", "batches": len(aff),
                              "fundedCommitmentCents": sum(c for c, _ in commit if isinstance(c, int)),
                              "fundedCommitmentByKind": {k: sum(c for c, kk in commit if kk == k and isinstance(c, int))
                                                         for k in ("staffing", "extension")},
                              "fundedWithoutExactCommitment": sum(1 for c, _ in commit if not isinstance(c, int)),
                              "creditSupported": sum(1 for r in aff if r["status"] == "funded_with_credit")},
            "reversibility": {"status": count((r["status"] for r in rev), FE.REVERSIBILITY),
                              "denominator": "executed actions", "actions": len(rev)},
            "exposure": {"status": count((r["status"] for r in exp), FE.EXPOSURE),
                         "byType": count(r["actions"][0]["type"] for r in exp if r["status"] == "direct_change"),
                         "denominator": "executed actions", "actions": len(exp),
                         "displacedBookings": sum(displaced.values()), "terminationBatches": len(displaced)}}


# --- run outcomes --------------------------------------------------------------------------------

def credit_days(run_dir):
    if not run_dir:
        return None
    res = FO.load(os.path.join(run_dir, "result.json"))
    return sum(1 for d in res.get("calendar") or [] if (d.get("creditDrawnEndCents") or 0) > 0)


def run_outcomes(run, extras, cdays):
    out = {n: FO.metric(run["dimensions"], d, n) for d, n in OUTCOMES}
    out.update({k: (extras or {}).get(k) for k in ("maxCreditDrawnCents", "interestCents", "overtimeCostCents")})
    out["creditDays"] = cdays
    out["actions"] = {n: FO.metric(run["dimensions"], "actions", n)
                      for n in ("actionsAttempted", "actionsAccepted", "actionsRejected")}
    return out


def diff(a, b):
    if isinstance(a, int) and isinstance(b, int) and not isinstance(a, bool) and not isinstance(b, bool):
        return a - b
    return None


# --- matched decision points (A) -----------------------------------------------------------------

def matched(base, prev):
    """Decision points where the two conditions' first message is the same
    (the same prompt SHA-256: the same observation and memory), and, of
    them, where the first output and the first proposal are the same."""
    n = min(len(base), len(prev))
    same_prompt = same_output = same_proposal = 0
    diverge = None
    for i in range(n):
        b, p = base[i], prev[i]
        if b.get("promptSha256") != p.get("promptSha256"):
            diverge = i
            break
        same_prompt += 1
        ob = ((b.get("attempts") or [{}])[0].get("output") or {}).get("sha256")
        op = ((p.get("attempts") or [{}])[0].get("output") or {}).get("sha256")
        same_output += ob is not None and ob == op
        fb, fp = first_proposal(b), first_proposal(p)
        same_proposal += (fb is None and fp is None) or (fb is not None and fp is not None and
                                                         FE.canon(fb["actions"]) == FE.canon(fp["actions"]))
    sub = next((i for i in range(n) if base[i].get("batches") != prev[i].get("batches")), None)
    return {"points": n, "samePrompt": same_prompt, "sameFirstOutput": same_output,
            "sameFirstProposal": same_proposal, "firstPromptDifference": diverge,
            "firstSubmittedDifference": sub}


# --- the evaluation ------------------------------------------------------------------------------

def analyze(binary, vdir, replays, jobs=1):
    corpus_path = os.path.join(vdir, "corpus.json")
    fin = financial(binary, corpus_path, replays, jobs)
    inputs = D.Inputs(replays)
    corpus = D.load_json(inputs, corpus_path)
    group = corpus["groups"][0]
    contract = D.load_json(inputs, group["suiteRuns"][0]["contract"])
    rmap = FO.replay_runs(corpus, replays, group["id"])
    runs_by = {(r["entry"], r["seed"], r["policy"]["id"]): r for r in contract["runs"]}
    fe_by = {e["recording"]["path"]: e for e in fin["runs"]}
    recs_by = {}
    for r in fin["records"]:
        recs_by.setdefault(r["recording"]["path"], []).append(r)
    entries = []
    for rp in group["replays"]:
        cfg = D.load_json(inputs, rp["config"])
        for pid, paths in sorted(D.replay_recordings(cfg).items()):
            for path in paths:
                recs = [json.loads(l) for l in inputs.read(path).decode("utf-8").splitlines() if l.strip()]
                fe = fe_by[path]
                run_id = fe.get("run") or {}
                key = (fe["paired"]["entry"], fe["paired"]["seed"], pid)
                run = runs_by[key]
                entries.append({
                    "recording": {"path": path, "sha256": inputs.files[path], "format": fe["recording"]["format"]},
                    "policy": pid, "condition": "preview" if pid.endswith("-preview") else "baseline",
                    "model": fe["model"], "entry": key[0], "seed": key[1], "days": run["days"],
                    "run": run["id"], "status": run["status"], "replayVerified": run.get("replayVerified"),
                    "engineReplay": fe.get("replay"),
                    "records": record_metrics(recs), "financialEvidence": fin_metrics(recs_by.get(path, [])),
                    "linkedBatches": {"validatedInRecords": validated_batches(recs)},
                    "outcomes": run_outcomes(run, FO.replay_extras(rmap.get(run["id"])),
                                             credit_days(rmap.get(run["id"]))),
                    "_recs": recs})
    noop = {}
    for (entry, seed, pid), run in runs_by.items():
        if pid == contract["baseline"]:
            noop[(entry, seed)] = {"run": run["id"], "outcomes": run_outcomes(
                run, FO.replay_extras(rmap.get(run["id"])), credit_days(rmap.get(run["id"])))}
    pairs = []
    by = {(e["model"], e["entry"], e["seed"], e["condition"]): e for e in entries}
    for (model, entry, seed, cond), b in sorted(by.items()):
        if cond != "baseline":
            continue
        p = by.get((model, entry, seed, "preview"))
        na = noop.get((entry, seed))
        row = {"model": model, "entry": entry, "seed": seed, "baseline": b["run"],
               "preview": p["run"] if p else None, "noAction": na["run"] if na else None}
        if p:
            row["matched"] = matched(b["_recs"], p["_recs"])
            row["previewMinusBaseline"] = {k: diff(p["outcomes"][k], b["outcomes"][k]) for k in b["outcomes"]
                                           if k != "actions"}
            row["direct"] = {
                "submittedActions": diff(p["records"]["submittedActions"]["total"],
                                         b["records"]["submittedActions"]["total"]),
                "fundedCommitmentCents": diff(p["financialEvidence"]["affordability"]["fundedCommitmentCents"],
                                              b["financialEvidence"]["affordability"]["fundedCommitmentCents"]),
                "creditSupported": diff(p["financialEvidence"]["affordability"]["creditSupported"],
                                        b["financialEvidence"]["affordability"]["creditSupported"]),
                "displacedBookings": diff(p["financialEvidence"]["exposure"]["displacedBookings"],
                                          b["financialEvidence"]["exposure"]["displacedBookings"])}
        if na:
            row["againstNoAction"] = {c: {k: diff(x["outcomes"][k], na["outcomes"][k]) for k in na["outcomes"]
                                          if k != "actions"} for c, x in (("baseline", b), ("preview", p)) if x}
        pairs.append(row)
    for e in entries:
        del e["_recs"]
        lb = e["linkedBatches"]
        lb["affordabilityRecords"] = e["financialEvidence"]["affordability"]["batches"]
        if lb["validatedInRecords"] != lb["affordabilityRecords"]:
            raise SystemExit(f"{e['recording']['path']}: {lb['validatedInRecords']} validated batches recorded, "
                             f"{lb['affordabilityRecords']} found by financial evidence v1")
    entries.sort(key=lambda e: (e["entry"], e["seed"], e["model"], e["condition"]))
    return {"schema": SCHEMA, "analyzer": ANALYZER, "plan": os.path.join(vdir, "PLAN.md"),
            "suiteRun": {"id": contract["suiteRun"], "status": contract["status"],
                         "expectedRuns": contract["expectedRuns"], "completedRuns": contract["completedRuns"],
                         "failedRuns": contract["failedRuns"], "seeds": contract["seeds"], "days": contract["days"]},
            "inputs": [{"path": p, "sha256": h} for p, h in sorted({**dict((i["path"], i["sha256"])
                                                                       for i in fin["inputs"]),
                                                                    **inputs.files}.items())],
            "runs": entries, "noAction": [{"entry": k[0], "seed": k[1], **v} for k, v in sorted(noop.items())],
            "pairs": pairs, "summary": summarize(entries)}, fin


def summarize(entries):
    """By model, entry and condition: sums of the counted outcomes, each with
    its denominator."""
    out = {}
    for e in entries:
        k = f"{e['model']} | {e['entry']} | {e['condition']}"
        s = out.setdefault(k, {"runs": 0, "decisionPoints": 0, "withStaffingProposal": 0, "submittedActions": 0,
                               "batches": 0, "creditSupported": 0, "executedActions": 0, "displacedBookings": 0,
                               "fundedCommitmentCents": 0})
        r, f = e["records"], e["financialEvidence"]
        s["runs"] += 1
        s["decisionPoints"] += r["decisionPoints"]
        s["withStaffingProposal"] += r["initialProposals"]["withStaffingAction"]
        s["submittedActions"] += r["submittedActions"]["total"]
        s["batches"] += f["affordability"]["batches"]
        s["creditSupported"] += f["affordability"]["creditSupported"]
        s["executedActions"] += f["reversibility"]["actions"]
        s["displacedBookings"] += f["exposure"]["displacedBookings"]
        s["fundedCommitmentCents"] += f["affordability"]["fundedCommitmentCents"]
        if "preview" in r:
            pv = s.setdefault("preview", {"eligible": 0, "status": {k: 0 for k in PREVIEW_STATUSES},
                                          "response": {k: 0 for k in RESPONSES}, "final": {k: 0 for k in FINALS}})
            pv["eligible"] += r["preview"]["eligiblePoints"]
            for g in ("status", "response", "final"):
                for k, v in r["preview"][g].items():
                    pv[g][k] = pv[g].get(k, 0) + v
    return {k: out[k] for k in sorted(out)}


# --- tables --------------------------------------------------------------------------------------

def money(c):
    return FE.cents(c) if isinstance(c, int) else "—"


def smoney(c):
    return FE.scents(c) if isinstance(c, int) else "—"


def n(v):
    return "—" if v is None else str(v)


def sn(v):
    return "—" if v is None else (f"+{v}" if v > 0 else str(v))


def short(m):
    return "Qwen" if m.startswith("Qwen") else "Gemma" if m.startswith("gemma") else m


def tables(doc):
    L = ["# V0.9c tables: prospective preview evaluation", "",
         f"Generated by `{ANALYZER}` from the committed recordings and contract report "
         "(docs/benchmark/v09c-dev/PLAN.md §8). Each count has its denominator. Direct effects (§1-§5) are kept "
         "apart from run outcomes (§6-§7); no score is formed.", ""]
    runs = doc["runs"]
    lab = lambda e: f"{short(e['model'])} {e['entry'].split('-')[0]} {e['seed']} {e['condition']}"  # noqa: E731
    L += ["## 1. Proposals (outcomes 1, 7, 9)", "",
          "| Run | Points | First proposal with an action | … with a staffing action | Attempts | "
          "Validate refusals (funding) / actions proposed | Submitted actions (staffing, operations) | Fallbacks |",
          "|---|---|---|---|---|---|---|---|"]
    for e in runs:
        r = e["records"]
        L.append(f"| {lab(e)} | {r['decisionPoints']} | {r['initialProposals']['withAction']} | "
                 f"{r['initialProposals']['withStaffingAction']} | {r['attemptsPerPoint']['total']} | "
                 f"{r['validateRefusals']['actions']} ({r['validateRefusals']['fundingRefusals']}) / "
                 f"{r['proposedActions']} | {r['submittedActions']['total']} ({r['submittedActions']['staffing']}, "
                 f"{r['submittedActions']['operations']}) | {r['outcomes'].get('fallback', 0)} |")
    L += ["", "## 2. The preview step (outcomes 2-6; preview-enabled runs)", "",
          "| Run | Eligible points | Shown | Failed | Too large | Shown with a refused action | Kept | Revised | "
          "Abandoned | No reply | Final: kept, revised, abandoned, fallback | Review requests | Review characters (max) |",
          "|---|---|---|---|---|---|---|---|---|---|---|---|---|"]
    for e in runs:
        p = e["records"].get("preview")
        if not p:
            continue
        st, rs, fi = p["status"], p["response"], p["final"]
        L.append(f"| {lab(e)} | {p['eligiblePoints']} | {st['ok']} | {st['failed']} | {st['too_large']} | "
                 f"{p['shownWithRefusedAction']} | {rs['kept']} | {rs['revised']} | {rs['abandoned']} | "
                 f"{rs['no_reply']} | {fi['kept']}, {fi['revised']}, {fi['abandoned']}, {fi['fallback']} | "
                 f"{p['reviewRequests']} | {p['reviewMessageChars']['total']:,} ({p['reviewMessageChars']['max']:,}) |")
    L += ["", "Denominators: eligible points are decision points whose first parsed proposal had a staffing action; "
          "responses and finals are of previews shown.", "",
          "### Each preview shown", "",
          "| Run | Point | Proposal | Funding test (commitment / headroom) | Refused in preview | Displaced bookings | "
          "Response (staffing, operations) | Final |", "|---|---|---|---|---|---|---|---|"]
    for e in runs:
        p = e["records"].get("preview")
        for x in (p or {}).get("previews") or []:
            f = x["facts"] or {}
            ft = (f"{money(f.get('commitmentCents'))} / {money(f.get('headroomCents'))} "
                  f"({'funded' if f.get('funded') else 'not funded' if f.get('funded') is False else '—'})"
                  if f.get("commitmentCents") is not None else "—")
            L.append(f"| {lab(e)} | {x['point']} | {', '.join(x['proposal'])} | {ft if x['status'] == 'ok' else x['status']} | "
                     f"{', '.join(f.get('refusedCodes') or []) or '—'} | {f.get('displacedBookings', '—')} | "
                     f"{x['response'] or '—'} ({x['staffing'] or '—'}, {x['operations'] or '—'}) | {x['final'] or '—'} |")
    L += ["", "## 3. Financial commitments (outcomes 10-11; financial evidence v1)", "",
          "| Run | Batches | Funded without credit | Funded with credit | Refused, insufficient funds | "
          "No new commitment | Not tested | Unavailable | Funded commitments |", "|---|---|---|---|---|---|---|---|---|"]
    for e in runs:
        a = e["financialEvidence"]["affordability"]
        s = a["status"]
        L.append(f"| {lab(e)} | {a['batches']} | {s['funded_without_credit']} | {s['funded_with_credit']} | "
                 f"{s['refused_insufficient_funds']} | {s['no_new_commitment']} | {s['not_tested']} | "
                 f"{s['evidence_unavailable']} | {money(a['fundedCommitmentCents'])} (staffing "
                 f"{money(a['fundedCommitmentByKind']['staffing'])}, extensions "
                 f"{money(a['fundedCommitmentByKind']['extension'])})"
                 + (f" ({a['fundedWithoutExactCommitment']} without an exact figure)"
                    if a["fundedWithoutExactCommitment"] else "") + " |")
    L += ["", "## 4. Reversibility (outcome 12)", "",
          "| Run | Executed actions | Withdrawable | Reversible, obligation stays | Not reversible | "
          "Reversible, no recorded cost | Unknown |", "|---|---|---|---|---|---|---|"]
    for e in runs:
        v = e["financialEvidence"]["reversibility"]
        s = v["status"]
        L.append(f"| {lab(e)} | {v['actions']} | {s['withdrawable']} | {s['reversible_with_retained_obligation']} | "
                 f"{s['not_reversible']} | {s['reversible_no_recorded_cost']} | {s['unknown']} |")
    L += ["", "## 5. Direct operational exposure (outcomes 13-14)", "",
          "| Run | Executed actions | Direct changes by type | Termination batches | Displaced bookings (engine) |",
          "|---|---|---|---|---|"]
    for e in runs:
        x = e["financialEvidence"]["exposure"]
        L.append(f"| {lab(e)} | {x['actions']} | {', '.join(f'{k} {v}' for k, v in x['byType'].items()) or '—'} | "
                 f"{x['terminationBatches']} | {x['displacedBookings']} |")
    L += ["", "## 6. Run outcomes (outcomes 15-20)", "",
          "| Run | Operating profit | Exams | Orders lost | TAT breaches | TAT p90 (min) | Most credit drawn | "
          "Days with credit drawn | Interest | Shortfall | Days below zero | Actions accepted / attempted |",
          "|---|---|---|---|---|---|---|---|---|---|---|---|"]
    rows = runs + [dict(x, model="", condition="no action") for x in doc["noAction"]]
    for e in rows:
        o = e["outcomes"]
        name = lab(e) if e["model"] else f"no action {e['entry'].split('-')[0]} {e['seed']}"
        L.append(f"| {name} | {money(o['operatingProfitCents'])} | {n(o['examsCompleted'])} | {n(o['ordersLost'])} | "
                 f"{n(o['tatBreaches'])} | {n(o['tatP90Sec'] // 60 if isinstance(o['tatP90Sec'], int) else None)} | "
                 f"{money(o['maxCreditDrawnCents'])} | {n(o['creditDays'])} | {money(o['interestCents'])} | "
                 f"{money(o['shortfallCents'])} | {n(o['daysBelowZero'])} | "
                 f"{n(o['actions']['actionsAccepted'])} / {n(o['actions']['actionsAttempted'])} |")
    L += ["", "## 7. Pairs: preview-enabled minus baseline (same model, entry, seed, horizon, economics)", "",
          "Whole-run differences. They describe two runs; no difference is attributed to one decision or to the "
          "preview alone, and eight pairs on two seeds support no generalization.", "",
          "| Pair | Same first message (points) | First submitted difference (point) | Submitted actions | "
          "Funded commitments | Credit-supported batches | Displaced bookings | Operating profit | Exams | "
          "Orders lost | TAT breaches | TAT p90 (min) | Most credit drawn |",
          "|---|---|---|---|---|---|---|---|---|---|---|---|---|"]
    for p in doc["pairs"]:
        if not p.get("preview"):
            L.append(f"| {short(p['model'])} {p['entry'].split('-')[0]} {p['seed']} | no preview-enabled run |")
            continue
        d, m, x = p["previewMinusBaseline"], p["matched"], p["direct"]
        L.append(f"| {short(p['model'])} {p['entry'].split('-')[0]} {p['seed']} | {m['samePrompt']} of {m['points']} | "
                 f"{n(m['firstSubmittedDifference'])} | {sn(x['submittedActions'])} | {smoney(x['fundedCommitmentCents'])} | "
                 f"{sn(x['creditSupported'])} | {sn(x['displacedBookings'])} | {smoney(d['operatingProfitCents'])} | "
                 f"{sn(d['examsCompleted'])} | {sn(d['ordersLost'])} | {sn(d['tatBreaches'])} | "
                 f"{sn(d['tatP90Sec'] // 60 if isinstance(d['tatP90Sec'], int) else None)} | "
                 f"{smoney(d['maxCreditDrawnCents'])} |")
    L += ["", "## 8. Against no action (each condition)", "",
          "| Pair | Condition | Operating profit | Exams | Orders lost | TAT breaches | Most credit drawn |",
          "|---|---|---|---|---|---|---|"]
    for p in doc["pairs"]:
        for c, d in (p.get("againstNoAction") or {}).items():
            L.append(f"| {short(p['model'])} {p['entry'].split('-')[0]} {p['seed']} | {c} | "
                     f"{smoney(d['operatingProfitCents'])} | {sn(d['examsCompleted'])} | {sn(d['ordersLost'])} | "
                     f"{sn(d['tatBreaches'])} | {smoney(d['maxCreditDrawnCents'])} |")
    L += ["", "## 9. Matched decision points (A)", "",
          "Points where both conditions' first message was the same (same prompt SHA-256: same observation and "
          "memory), and of them where the model's first output and first proposal were the same.", "",
          "| Pair | Points | Same first message | Same first output | Same first proposal | First message difference |",
          "|---|---|---|---|---|---|"]
    for p in doc["pairs"]:
        m = p.get("matched")
        if m:
            L.append(f"| {short(p['model'])} {p['entry'].split('-')[0]} {p['seed']} | {m['points']} | {m['samePrompt']} | "
                     f"{m['sameFirstOutput']} | {m['sameFirstProposal']} | {n(m['firstPromptDifference'])} |")
    return "\n".join(L) + "\n"


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--dir", required=True, help="the evaluation's directory (docs/benchmark/v09c-dev)")
    ap.add_argument("--replays", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--jobs", type=int, default=1)
    a = ap.parse_args(argv)
    doc, fin = analyze(a.binary, a.dir, a.replays, a.jobs)
    os.makedirs(a.out, exist_ok=True)
    FE.write_json(os.path.join(a.out, "preview-evaluation.json"), doc)
    with open(os.path.join(a.out, "preview-tables.md"), "w", encoding="utf-8") as f:
        f.write(tables(doc))
    FE.write_json(os.path.join(a.out, "financial-evidence.json"), fin)
    with open(os.path.join(a.out, "financial-tables.md"), "w", encoding="utf-8") as f:
        f.write(FE.tables(fin))
    print(f"preview evaluation: {len(doc['runs'])} LLM runs, {len(doc['pairs'])} pairs; "
          f"financial evidence: {len(fin['records'])} records")
    return 0


if __name__ == "__main__":
    sys.exit(main())
