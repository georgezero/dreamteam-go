#!/usr/bin/env python3
"""V0.6c tables (PLAN.md §4) from a suite run's results: the contract
report (contract.json), each run's decisions.json, and the LLM agent's
evidence (policy-evidence/decisions.jsonl, summary.json). Existing metrics
only: no composite score, no ranking, no test of significance.

    python3 -I -B docs/benchmark/v06c-dev/analyze.py RESULTS > tables.md

Nothing is written elsewhere."""

import json
import os
import sys

STAFFING = ("hire", "terminate", "set_roster", "set_reading_allocation", "contract_teleradiology")
INFRA = ("http_status", "timeout", "connection")
KEY = [("operatingProfitCents", "profit", "$"), ("netRevenueCents", "revenue", "$"),
       ("operatingExpenseCents", "expenses", "$"), ("tatP90Sec", "TAT p90", "min"), ("tatBreaches", "breaches", ""),
       ("examsCompleted", "exams", ""), ("ordersLost", "lost", ""), ("overtimeSec", "overtime", "min")]


def fmt(v, unit, signed=False):
    if v is None:
        return "–"
    s = "+" if signed and v > 0 else ""
    if unit == "$":
        return f"{'-' if v < 0 else s}${abs(v) / 100:,.0f}"
    if unit == "min":
        return f"{s}{v / 60:,.0f}"
    return f"{s}{v:,}"


def table(head, rows):
    out = ["| " + " | ".join(head) + " |", "|" + "---|" * len(head)]
    return "\n".join(out + ["| " + " | ".join(str(c) for c in r) + " |" for r in rows])


def jsonl(p):
    if not os.path.exists(p):
        return None
    with open(p, encoding="utf-8") as f:
        return [json.loads(x) for x in f if x.strip()]


def metric(run, name):
    for d in run["dimensions"]:
        for m in d["metrics"]:
            if m["name"] == name:
                return m.get("value")
    return None


def code(err):
    rest = err.split(": ", 1)[1] if ": " in err else err
    return rest.split(":", 1)[0].strip()


def behaviour(recs):
    seen, repeated, refusals, errors = set(), 0, {}, {}
    first_valid = details = 0
    out = {}
    for r in recs:
        here = set()
        for x in r.get("refusals") or []:
            k = (x.get("type"), x.get("code"))
            refusals[f"{k[0]} {k[1]}"] = refusals.get(f"{k[0]} {k[1]}", 0) + 1
            if k in seen:
                repeated += 1
            here.add(k)
        seen |= here
        errs = r.get("errors") or []
        for e in errs:
            errors[code(e)] = errors.get(code(e), 0) + 1
        if r.get("outcome") != "fallback" and not any(e.startswith("attempt 1:") for e in errs) and \
                not any(x.get("attempt") == 1 for x in r.get("refusals") or []):
            first_valid += 1
        details += 1 if r.get("detailsRequested") else 0
        out[r.get("outcome")] = out.get(r.get("outcome"), 0) + 1
    return {"points": len(recs), "actions": out.get("actions", 0), "none": out.get("no_action", 0),
            "fallback": out.get("fallback", 0), "first": first_valid, "refusals": refusals, "repeated": repeated,
            "errors": errors, "details": details}


def main(d):
    with open(os.path.join(d, "contract.json"), encoding="utf-8") as f:
        k = json.load(f)
    p = print
    entries = [e["id"] for e in k["entries"]]
    pols = [x["id"] for x in k["policies"]]
    runs = {(r["entry"], r["seed"], r["policy"]["id"]): r for r in k["runs"]}
    pairs = {(x["entry"], x["seed"], x["policy"]): {dl["name"]: dl["value"] for dim in x["dimensions"] for dl in dim["deltas"]}
             for x in k["pairs"]}
    p(f"# {k['suiteRun']}: suite {k['suite']['id']} version {k['suite']['version']}, seeds "
      f"{', '.join(map(str, k['seeds']))}, {k['days']} days, status {k['status']} ({k['completedRuns']} of {k['expectedRuns']} runs)\n")

    p("## Outcomes (absolute, each run)\n")
    rows = []
    for e in entries:
        for s in k["seeds"]:
            for pol in pols:
                r = runs.get((e, s, pol))
                if not r:
                    continue
                a = r["actions"]
                rows.append([e, s, pol, r["status"], "yes" if r["replayVerified"] else "NO"] +
                            [fmt(metric(r, n), u) for n, _, u in KEY] +
                            [f"{a['operations']}/{a['staffing']}", "; ".join(f"{t}: {c}" for t, c in sorted(a["byType"].items())) or "none"])
    p(table(["entry", "seed", "policy", "status", "replay"] + [h for _, h, _ in KEY] + ["actions o/s", "accepted by type"], rows))

    p("\n## Paired with no action (policy − noop, same entry and seed; minutes for times)\n")
    rows = []
    for e in entries:
        for s in k["seeds"]:
            for pol in pols:
                dl = pairs.get((e, s, pol))
                if dl is None:
                    continue
                better = (dl.get("tatP90Sec") or 0) < 0 or (dl.get("tatBreaches") or 0) < 0
                flag = "better service, lower profit" if better and (dl.get("operatingProfitCents") or 0) < 0 else ""
                rows.append([e, s, pol] + [fmt(dl.get(n), u, True) for n, _, u in KEY] + [flag])
    p(table(["entry", "seed", "policy"] + ["Δ" + h for _, h, _ in KEY] + ["flag"], rows))

    llm = [(key, r) for key, r in runs.items() if os.path.exists(os.path.join(d, "runs", r["id"], "policy-evidence", "decisions.jsonl"))]
    if not llm:
        return
    p("\n## LLM decisions\n")
    rows, refusal_rows = [], []
    for (e, s, pol), r in sorted(llm):
        ev = os.path.join(d, "runs", r["id"], "policy-evidence")
        b = behaviour(jsonl(os.path.join(ev, "decisions.jsonl")) or [])
        rows.append([e, s, pol, b["points"], b["actions"], b["none"], b["fallback"], f"{b['first']}/{b['points']}",
                     b["repeated"], b["details"], sum(v for c, v in b["errors"].items() if c in INFRA)])
        refusal_rows.append([e, s, pol, "; ".join(f"{c}: {v}" for c, v in sorted(b["refusals"].items())) or "none",
                             "; ".join(f"{c}: {v}" for c, v in sorted(b["errors"].items())) or "none"])
    p(table(["entry", "seed", "policy", "points", "with actions", "no action", "fallback", "first-attempt valid",
             "repeated invalid", "details", "infrastructure errors"], rows))
    p("\n## Refusals by validate, and agent errors\n")
    p(table(["entry", "seed", "policy", "refusals (type code: count)", "errors (code: count)"], refusal_rows))

    p("\n## Tokens, latency and cost\n")
    rows = []
    for (e, s, pol), r in sorted(llm):
        sp = os.path.join(d, "runs", r["id"], "policy-evidence", "summary.json")
        sm = json.load(open(sp, encoding="utf-8")) if os.path.exists(sp) else {}
        t = sm.get("tokens") or {}
        rep, lat = t.get("reported") or {}, sm.get("latencyMs") or {}
        cost = sm.get("cost") or {}
        rows.append([e, s, pol, sm.get("requests"), sm.get("retries"), f"{rep.get('prompt', 0):,}", f"{rep.get('completion', 0):,}",
                     f"{(t.get('estimated') or {}).get('requests', 0)}/{(t.get('unknown') or {}).get('requests', 0)}",
                     f"{lat.get('mean', 0) / 1000:.1f}", f"{lat.get('max', 0) / 1000:.1f}", f"{lat.get('total', 0) / 1000:,.0f}",
                     cost.get("note") or cost.get("status") or ("none configured" if not cost else json.dumps(cost)[:40])])
    p(table(["entry", "seed", "policy", "requests", "retries", "prompt tokens", "completion tokens", "estimated/unknown",
             "latency mean s", "max s", "total s", "estimated cost"], rows))


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    main(sys.argv[1])
