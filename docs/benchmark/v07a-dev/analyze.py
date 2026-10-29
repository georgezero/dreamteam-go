#!/usr/bin/env python3
"""V0.7a tables (PLAN.md §4): the five V0.6 practices under the historical
economics (suite dreamteam-dev 1) and with radiologist overtime paid
(dreamteam-dev 2), paired by entry, seed and policy. Reads each results
directory's contract.json and every run's result.json and decisions.json.
Existing metrics only: no composite score, no ranking, no test of
significance.

    python3 -I -B docs/benchmark/v07a-dev/analyze.py V1_RESULTS V2_RESULTS > tables.md

Nothing is written elsewhere."""

import json
import os
import sys

POLICIES = ("noop", "greedy", "conservative-v2")


def load(d):
    with open(os.path.join(d, "contract.json"), encoding="utf-8") as f:
        k = json.load(f)
    runs = {}
    for r in k["runs"]:
        rd = os.path.join(d, "runs", r["id"])
        with open(os.path.join(rd, "result.json"), encoding="utf-8") as f:
            res = json.load(f)
        with open(os.path.join(rd, "decisions.json"), encoding="utf-8") as f:
            dec = json.load(f)
        runs[(r["entry"], r["seed"], r["policy"]["id"])] = {"contract": r, "result": res, "decisions": dec}
    return k, runs


def metric(run, name):
    for d in run["contract"]["dimensions"]:
        for m in d["metrics"]:
            if m["name"] == name:
                return m.get("value")
    return None


def rad_ot(run):
    """Radiologists' time on duty past the rostered end, seconds."""
    return sum(s["overtimeSec"] for s in run["result"]["window"]["ops"]["staff"] if s["role"] == "radiologist")


def actions(run):
    """The accepted actions in submission order, with when they were decided."""
    out = []
    for p in run["decisions"]:
        for b in p.get("batches") or []:
            for x in b.get("results") or []:
                if x.get("ok"):
                    out.append((p["time"], json.dumps(x["action"], sort_keys=True)))
    return out


def t90(run):
    t = run["contract"]["t90"]
    if t.get("applicability") == "not_applicable":
        return "not applicable"
    return t.get("verdict") or t.get("status", "–")


def money(v, signed=False):
    if v is None:
        return "–"
    s = "+" if signed and v > 0 else ""
    return f"{'-' if v < 0 else s}${abs(v) / 100:,.0f}"


def num(v, signed=False, div=1):
    if v is None:
        return "–"
    v = round(v / div) if div != 1 else v
    return f"{'+' if signed and v > 0 else ''}{v:,}"


def table(head, rows):
    out = ["| " + " | ".join(head) + " |", "|" + "---|" * len(head)]
    return "\n".join(out + ["| " + " | ".join(str(c) for c in r) + " |" for r in rows])


def main(d1, d2):
    k1, r1 = load(d1)
    k2, r2 = load(d2)
    entries = [e["id"] for e in k1["entries"]]
    seeds = k1["seeds"]
    print(f"# V0.7a characterization: {k1['suiteRun']} (suite {k1['suite']['id']} {k1['suite']['version']}) and "
          f"{k2['suiteRun']} (suite {k2['suite']['id']} {k2['suite']['version']})\n")
    print(f"Seeds {', '.join(map(str, seeds))}, {k1['days']} days. Version 1: {k1['status']} ({k1['completedRuns']} of "
          f"{k1['expectedRuns']}); version 2: {k2['status']} ({k2['completedRuns']} of {k2['expectedRuns']}). "
          f"Engines {k1['engine']}, {k2['engine']}.\n")

    print("## 1. Starting states\n")
    rows = []
    for e in entries:
        for s in seeds:
            a, b = r1[(e, s, "noop")]["contract"], r2[(e, s, "noop")]["contract"]
            rows.append([e, s, a["startingStateDigest"][:12] + "…", b["startingStateDigest"][:12] + "…",
                         "yes" if a["startingStateDigest"] == b["startingStateDigest"] else "no"])
    print(table(["entry", "seed", "v1 start", "v2 start", "identical"], rows) + "\n")

    print("## 2. Each run, version 1 and version 2\n")
    rows = []
    for e in entries:
        for s in seeds:
            for p in POLICIES:
                a, b = r1[(e, s, p)], r2[(e, s, p)]
                rows.append([e, s, p, a["contract"]["status"] + "/" + b["contract"]["status"],
                             "yes" if a["contract"]["replayVerified"] and b["contract"]["replayVerified"] else "NO",
                             money(metric(a, "operatingProfitCents")), money(metric(b, "operatingProfitCents")),
                             num(rad_ot(a), div=3600), num(rad_ot(b), div=3600),
                             len(actions(a)), len(actions(b)), t90(a), t90(b)])
    print(table(["entry", "seed", "policy", "status v1/v2", "replay", "profit v1", "profit v2", "radiologist OT h v1",
                 "radiologist OT h v2", "accepted v1", "accepted v2", "T90 v1", "T90 v2"], rows) + "\n")

    print("## 3. Version 2 minus version 1 (same entry, seed and policy)\n")
    print("Overtime cost is the harness's overtime cost (all staff, before benefits); its change is the radiologist "
          "overtime charge where operations are unchanged. Actions: whether the accepted actions (and when they were "
          "decided) are identical.\n")
    rows = []
    for e in entries:
        for s in seeds:
            for p in POLICIES:
                a, b = r1[(e, s, p)], r2[(e, s, p)]
                dm = lambda n: (None if metric(a, n) is None or metric(b, n) is None else metric(b, n) - metric(a, n))
                ota = a["result"]["window"]["ops"]["workforce"]["overtimeCostCents"]
                otb = b["result"]["window"]["ops"]["workforce"]["overtimeCostCents"]
                cash = b["result"]["window"]["money"]["cashEndCents"] - a["result"]["window"]["money"]["cashEndCents"]
                rows.append([e, s, p, "yes" if actions(a) == actions(b) else "**no**", money(otb - ota, True),
                             money(dm("operatingExpenseCents"), True), money(dm("operatingProfitCents"), True),
                             money(dm("netRevenueCents"), True), money(cash, True), money(dm("minCashCents"), True),
                             num(dm("tatP90Sec"), True, 60), num(dm("tatBreaches"), True), num(dm("examsCompleted"), True),
                             num(dm("ordersLost"), True), num(rad_ot(b) - rad_ot(a), True, 60)])
    print(table(["entry", "seed", "policy", "actions same", "Δ overtime cost", "Δ expense", "Δ profit", "Δ revenue",
                 "Δ cash end", "Δ min cash", "Δ TAT p90 min", "Δ breaches", "Δ exams", "Δ lost", "Δ radiologist OT min"],
                rows) + "\n")

    print("## 4. Paired with no action, each version (policy − noop, same entry and seed)\n")
    rows = []
    for e in entries:
        for s in seeds:
            for p in POLICIES[1:]:
                cells = [e, s, p]
                for rr in (r1, r2):
                    a, b = rr[(e, s, "noop")], rr[(e, s, p)]
                    cells += [money(metric(b, "operatingProfitCents") - metric(a, "operatingProfitCents"), True),
                              num(metric(b, "tatBreaches") - metric(a, "tatBreaches"), True)]
                rows.append(cells)
    print(table(["entry", "seed", "policy", "Δ profit v1", "Δ breaches v1", "Δ profit v2", "Δ breaches v2"], rows) + "\n")

    print("## 5. Where the accepted actions differ\n")
    any_diff = False
    for e in entries:
        for s in seeds:
            for p in POLICIES:
                a, b = actions(r1[(e, s, p)]), actions(r2[(e, s, p)])
                if a != b:
                    any_diff = True
                    print(f"* {e} {s} {p}:")
                    print(f"  * version 1: {[(t[:10], json.loads(x).get('type'), json.loads(x).get('siteAllocationBP')) for t, x in a]}")
                    print(f"  * version 2: {[(t[:10], json.loads(x).get('type'), json.loads(x).get('siteAllocationBP')) for t, x in b]}")
    if not any_diff:
        print("None: every policy took the same actions at the same times under both economics.")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit("usage: analyze.py V1_RESULTS V2_RESULTS")
    main(sys.argv[1], sys.argv[2])
