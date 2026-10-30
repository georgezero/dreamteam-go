#!/usr/bin/env python3
"""Descriptive financial and operational outcomes of the V0.6c and V0.7c
evaluations, each policy paired with no action
(dreamteam.financial-outcomes/v1).

    python3 -I -B analysis/evidence/financial_outcomes.py \
        --corpus docs/benchmark/v08a-dev/corpus.json --replays <dir> --out <dir>

For every suite run (V0.6c 30 and 90 days; V0.7c stages A and B), every
policy is compared with no action on the same entry, seed, horizon and
economics, from the committed contract report (contract.json). For the LLM
runs and their no-action pairs, which the corpus replays, it adds what
the contract report does not carry: overtime cost, the most credit drawn,
interest, and the actions executed by type. It keeps five things apart:
1. financial outcomes; 2. operational outcomes; 3. stated reasons (the
explicit commitments the frozen rules read, by action type); 4. submitted
decisions; 5. executed decisions.

Nothing is combined across suite runs: V0.6c (dreamteam-dev 1, no overtime
cost, no credit) and V0.7c (dreamteam-dev 3, overtime paid, $50,000 credit
line) are different economics. No score, ranking, cause or counterfactual
is computed: a difference from no action is a description of two runs.
"""

import argparse
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import decision_evidence as D  # noqa: E402  (the frozen analyzer: extract and corpus helpers only)

SCHEMA = "dreamteam.financial-outcomes/v1"
SUITE_RUNS = [
    {"id": "v06c-compare-30", "group": "v06c", "credit": False, "contract": "docs/benchmark/v06c-dev/compare-30/contract.json",
     "economics": "dreamteam-dev 1 (V0.6): overtime not costed, no credit line"},
    {"id": "v06c-compare-90", "group": "v06c", "credit": False, "contract": "docs/benchmark/v06c-dev/compare-90/contract.json",
     "economics": "dreamteam-dev 1 (V0.6): overtime not costed, no credit line"},
    {"id": "v07c-stage-a", "group": "v07c", "credit": True, "contract": "docs/benchmark/v07c-dev/stage-a/contract.json",
     "economics": "dreamteam-dev 3 (V0.7): radiologist overtime paid, $50,000 credit line at 10%"},
    {"id": "v07c-stage-b", "group": "v07c", "credit": True, "contract": "docs/benchmark/v07c-dev/stage-b/contract.json",
     "economics": "dreamteam-dev 3 (V0.7): radiologist overtime paid, $50,000 credit line at 10%"},
]
FIN = [("financial", "operatingProfitCents"), ("financial", "netRevenueCents"),
       ("financial", "operatingExpenseCents"), ("sustainability", "minCashCents")]
OPS = [("operational", "examsCompleted"), ("staffingCapacity", "ordersLost"), ("operational", "tatP90Sec"),
       ("serviceReliability", "tatBreaches"), ("staffingCapacity", "overtimeSec")]
STAFFING_TYPES = ("hire", "terminate", "set_roster", "set_reading_allocation", "contract_teleradiology")


def metric(dims, dim, name, key="metrics"):
    for d in dims or []:
        if d.get("dimension") == dim:
            for m in d.get(key) or []:
                if m.get("name") == name:
                    return m.get("value")
    return None


def load(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def replay_runs(corpus, replays, group):
    """run id -> replay run directory, for the group's replays."""
    out = {}
    for g in corpus["groups"]:
        if g["id"] != group:
            continue
        for rp in g["replays"]:
            d = os.path.join(replays, rp["out"], "runs")
            if os.path.isdir(d):
                for rid in sorted(os.listdir(d)):
                    out.setdefault(rid, os.path.join(d, rid))
    return out


def replay_extras(run_dir):
    """What the replay adds: overtime cost, credit, executed actions by type."""
    if not run_dir:
        return None
    res = load(os.path.join(run_dir, "result.json"))
    w = res.get("window") or {}
    work = (w.get("ops") or {}).get("workforce") or {}
    cal = res.get("calendar") or []
    executed = {}
    for p in load(os.path.join(run_dir, "decisions.json")) or []:
        for b in p.get("batches") or []:
            for r in b.get("results") or []:
                if r.get("ok") and r.get("code") != "unchanged":
                    t = (r.get("action") or {}).get("type")
                    executed[t] = executed.get(t, 0) + 1
    return {"overtimeCostCents": work.get("overtimeCostCents"),
            "approvedOvertimeCostCents": work.get("approvedOvertimeCostCents"),
            "maxCreditDrawnCents": max([d.get("creditDrawnEndCents", 0) or 0 for d in cal] or [0]),
            "interestCents": (w.get("money") or {}).get("interestCents"),
            "executedByType": {k: executed[k] for k in sorted(executed)}}


def recording_counts(path):
    """Stated commitments, proposals and submissions by action type, from a
    recording (the frozen statement reader, without staff names)."""
    stated, proposed, submitted = {}, {}, {}
    with open(path, encoding="utf-8") as f:
        for line in f:
            r = json.loads(line)
            st, _ = D.extract(r.get("reason") or "", {})
            for s in st:
                if s["kind"] == "commit" and s["type"]:
                    stated[s["type"]] = stated.get(s["type"], 0) + 1
            for a in r.get("proposed") or []:
                proposed[a.get("type")] = proposed.get(a.get("type"), 0) + 1
            for b in r.get("batches") or []:
                for a in b:
                    submitted[a.get("type")] = submitted.get(a.get("type"), 0) + 1
    srt = lambda d: {k: d[k] for k in sorted(d)}
    return {"statedCommitments": srt(stated), "proposed": srt(proposed), "submitted": srt(submitted)}


def recordings_by_run(corpus, group):
    """(entry scenario, seed, policy) -> recording path, from the replay
    configurations' recording names."""
    out = {}
    for g in corpus["groups"]:
        if g["id"] != group:
            continue
        for rp in g["replays"]:
            cfg = load(rp["config"])
            for pid, paths in D.replay_recordings(cfg).items():
                for p in paths:
                    out[(pid, os.path.basename(p))] = p
    return out


def analyze(corpus_path, replays):
    corpus = load(corpus_path)
    doc = {"schema": SCHEMA, "suiteRuns": []}
    for sr in SUITE_RUNS:
        c = load(sr["contract"])
        runs = {r["id"]: r for r in c["runs"]}
        rmap = replay_runs(corpus, replays, sr["group"])
        recs = recordings_by_run(corpus, sr["group"])
        base = {(r["entry"], r["seed"]): r for r in c["runs"] if r["policy"]["id"] == c["baseline"]}
        rows = []
        for pair in c["pairs"]:
            b = base[(pair["entry"], pair["seed"])]
            run = next(r for r in c["runs"] if r["entry"] == pair["entry"] and r["seed"] == pair["seed"]
                       and r["policy"]["id"] == pair["policy"])
            row = {"entry": pair["entry"], "seed": pair["seed"], "policy": pair["policy"], "days": run["days"],
                   "runId": run["id"], "baselineRunId": b["id"]}
            row["financial"] = {n: {"policy": metric(run["dimensions"], d, n), "noAction": metric(b["dimensions"], d, n),
                                    "difference": metric(pair["dimensions"], d, n, "deltas")} for d, n in FIN}
            row["operational"] = {n: {"policy": metric(run["dimensions"], d, n),
                                      "noAction": metric(b["dimensions"], d, n),
                                      "difference": metric(pair["dimensions"], d, n, "deltas")} for d, n in OPS}
            ex_p, ex_b = replay_extras(rmap.get(run["id"])), replay_extras(rmap.get(b["id"]))
            if ex_p and ex_b:
                for k in ("overtimeCostCents", "maxCreditDrawnCents", "interestCents"):
                    row["financial"][k] = {"policy": ex_p[k], "noAction": ex_b[k],
                                           "difference": (ex_p[k] or 0) - (ex_b[k] or 0)}
            row["contractActions"] = run.get("actions")
            rec = recs.get((pair["policy"], f"{pair['entry']}-{pair['policy']}-s{pair['seed']}-decisions.jsonl"))
            if rec:
                row["recording"] = rec
                row.update(recording_counts(rec))
            if ex_p:
                row["executedByType"] = ex_p["executedByType"]
            rows.append(row)
        doc["suiteRuns"].append({"id": sr["id"], "contract": sr["contract"], "economics": sr["economics"],
                                 "creditLine": sr["credit"],
                                 "engine": c.get("engine"), "suite": c.get("suite"), "days": c.get("days"),
                                 "baseline": c["baseline"], "pairs": rows})
    return doc


def money(c):
    if not isinstance(c, int):
        return "—"
    return ("−" if c < 0 else "") + f"${(abs(c) + 50) // 100:,}"  # to the nearest dollar


def signed(c, f=money):
    if not isinstance(c, int):
        return "—"
    return ("+" if c > 0 else "") + f(c)


def num(v):
    return "—" if not isinstance(v, int) else f"{v:,}".replace("-", "−")


def snum(v):
    return "—" if not isinstance(v, int) else ("+" if v > 0 else "") + num(v)


def credit(sr, f, key):
    """Credit drawn or interest: n/a without a credit line, — without a replay."""
    if not sr["creditLine"]:
        return "n/a"
    if key not in f:
        return "—"
    return money((f[key] or {}).get("policy") or 0)


def counts(d):
    return ", ".join(f"{k} {v}" for k, v in sorted((d or {}).items())) or "none"


def tables(doc):
    L = ["# Financial and operational outcomes, paired with no action", "",
         f"`{doc['schema']}`. Each row compares one policy's run with the no-action run on the same entry, seed, "
         "horizon and economics. A difference describes two runs; it is not attributed to any one decision, and "
         "no row is a score. The suite runs are never combined. Money is rounded to the nearest dollar. "
         "Overtime cost, credit and executed actions come from the replays, which exist for the LLM runs and their "
         "no-action pairs only (— for the other policies); n/a: no credit line in that economics.", ""]
    for sr in doc["suiteRuns"]:
        L += [f"## {sr['id']}: {sr['economics']}", "", f"Contract `{sr['contract']}`, engine {sr['engine']}, "
              f"{sr['days']} days.", "", "### 1. Financial outcomes (policy minus no action)", "",
              "| Entry | Seed | Policy | Profit | Revenue | Expense | Lowest cash (policy) | Lowest cash difference | "
              "Overtime cost | Most credit drawn | Interest |", "|---|---|---|---|---|---|---|---|---|---|---|"]
        for r in sr["pairs"]:
            f = r["financial"]
            g = lambda k, part="difference": (f.get(k) or {}).get(part)
            L.append(f"| {r['entry']} | {r['seed']} | {r['policy']} | {signed(g('operatingProfitCents'))} | "
                     f"{signed(g('netRevenueCents'))} | {signed(g('operatingExpenseCents'))} | "
                     f"{money(g('minCashCents', 'policy'))} | {signed(g('minCashCents'))} | "
                     f"{signed(g('overtimeCostCents'))} | {credit(sr, f, 'maxCreditDrawnCents')} | "
                     f"{credit(sr, f, 'interestCents')} |")
        L += ["", "### 2. Operational outcomes (policy minus no action)", "",
              "| Entry | Seed | Policy | Exams completed | Orders lost | Report TAT p90 (min) | TAT breaches | "
              "Overtime (h) |", "|---|---|---|---|---|---|---|---|"]
        for r in sr["pairs"]:
            o = r["operational"]
            d = lambda k: (o.get(k) or {}).get("difference")
            tat = d("tatP90Sec")
            ot = d("overtimeSec")
            L.append(f"| {r['entry']} | {r['seed']} | {r['policy']} | {snum(d('examsCompleted'))} | "
                     f"{snum(d('ordersLost'))} | {snum(tat // 60 if isinstance(tat, int) and tat >= 0 else -((-tat) // 60) if isinstance(tat, int) else None)} | "
                     f"{snum(d('tatBreaches'))} | {snum(ot // 3600 if isinstance(ot, int) and ot >= 0 else -((-ot) // 3600) if isinstance(ot, int) else None)} |")
        L += ["", "### 3–5. Stated reasons, submitted and executed decisions (LLM runs), and the contract's action "
              "counts (every policy)", "",
              "Stated: explicit commitments by action type, as the frozen rules read them (a commitment can be "
              "hedged). Submitted: actions in submitted batches. Executed: actions the engine accepted with a "
              "change (replay apply results).", "",
              "| Entry | Seed | Policy | Stated commitments | Submitted | Executed | Contract report: operations, "
              "staffing, refused |", "|---|---|---|---|---|---|---|"]
        for r in sr["pairs"]:
            ca = r.get("contractActions") or {}
            L.append(f"| {r['entry']} | {r['seed']} | {r['policy']} | "
                     f"{counts(r.get('statedCommitments')) if 'statedCommitments' in r else 'not an LLM'} | "
                     f"{counts(r.get('submitted')) if 'submitted' in r else '—'} | "
                     f"{counts(r.get('executedByType')) if 'executedByType' in r else '—'} | "
                     f"{ca.get('operations', '—')}, {ca.get('staffing', '—')}, {ca.get('refused', '—')} |")
        L.append("")
    return "\n".join(L)


def main(argv=None):
    ap = argparse.ArgumentParser(description="Descriptive outcomes paired with no action (read-only; no model).")
    ap.add_argument("--corpus", required=True)
    ap.add_argument("--replays", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    doc = analyze(a.corpus, a.replays.rstrip("/"))
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, "financial-outcomes.json"), "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True, ensure_ascii=True)
        f.write("\n")
    with open(os.path.join(a.out, "financial-outcomes.md"), "w", encoding="utf-8") as f:
        f.write(tables(doc))
    print(f"financial outcomes: {sum(len(s['pairs']) for s in doc['suiteRuns'])} pairs in {len(doc['suiteRuns'])} "
          "suite runs")
    return 0


if __name__ == "__main__":
    sys.exit(main())
