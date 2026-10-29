#!/usr/bin/env python3
"""V0.7b tables (PLAN.md §4): the five practices without solvency (suite
dreamteam-dev 2) and with it (dreamteam-dev 3: a $50,000 credit line,
-credit, and none, -nocredit), paired by practice, seed and policy. Reads
each results directory's contract.json and every run's result.json and
decisions.json. Existing metrics only: no composite score, no ranking, no
test of significance.

    python3 -I -B docs/benchmark/v07b-dev/analyze.py V2_RESULTS V3_RESULTS > tables.md

Nothing is written elsewhere."""

import json
import os
import sys

POLICIES = ("noop", "greedy", "conservative-v2")
VARIANTS = ("credit", "nocredit")


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


def accepted(run):
    """The accepted actions in submission order, with when they were decided."""
    out = []
    for p in run["decisions"]:
        for b in p.get("batches") or []:
            for x in b.get("results") or []:
                if x.get("ok"):
                    out.append((p["time"], json.dumps(x["action"], sort_keys=True)))
    return out


def refused(run):
    """The actions refused insufficient_funds: (decision time, action type, share)."""
    out = []
    for p in run["decisions"]:
        for b in p.get("batches") or []:
            for x in b.get("results") or []:
                if x.get("code") == "insufficient_funds":
                    a = x["action"]
                    out.append((p["time"][:10], a.get("type"), a.get("employee"), a.get("siteAllocationBP")))
    return out


def credit(run):
    """Most drawn at a day's end, interest in the window, and the distress
    shortfall: the lowest cash in the window below zero (under solvency,
    cash is below zero only when the line is fully drawn; amendment 1)."""
    res = run["result"]
    drawn = max([d.get("creditDrawnEndCents", 0) for d in res["calendar"]] or [0])
    interest = res["window"]["money"].get("interestCents", 0)
    return drawn, interest, max(0, -res["window"]["money"]["minCashCents"])


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


def main(d2, d3):
    k2, r2 = load(d2)
    k3, r3 = load(d3)
    for k, rr in ((k2, r2), (k3, r3)):
        bad = [i for i, r in rr.items() if r["contract"]["status"] != "completed" or not r["contract"]["replayVerified"]]
        if bad:
            print(f"**Runs not completed or not replay-verified in {k['suiteRun']}: {bad}**\n")
    entries = [e["id"] for e in k2["entries"]]
    seeds = k2["seeds"]
    v3 = lambda e, s, p, v: r3[(f"{e}-{v}", s, p)]
    print(f"# V0.7b characterization: {k2['suiteRun']} (suite {k2['suite']['id']} {k2['suite']['version']}) and "
          f"{k3['suiteRun']} (suite {k3['suite']['id']} {k3['suite']['version']})\n")
    print(f"Seeds {', '.join(map(str, seeds))}, {k2['days']} days. Version 2: {k2['status']} ({k2['completedRuns']} of "
          f"{k2['expectedRuns']}); version 3: {k3['status']} ({k3['completedRuns']} of {k3['expectedRuns']}). "
          f"Engines {k2['engine']}, {k3['engine']}.\n")

    print("## 1. Starting states\n")
    rows = []
    for e in entries:
        for s in seeds:
            a = r2[(e, s, "noop")]["contract"]["startingStateDigest"]
            bs = [v3(e, s, "noop", v)["contract"]["startingStateDigest"] for v in VARIANTS]
            rows.append([e, s, a[:12] + "…", "yes" if all(b == a for b in bs) else "**no**"])
    print(table(["entry", "seed", "v2 start", "v3 (both variants) identical"], rows) + "\n")

    print("## 2. Binding: any refusal, draw, interest or distress\n")
    rows = []
    for e in entries:
        for p in POLICIES:
            for v in VARIANTS:
                runs = [v3(e, s, p, v) for s in seeds]
                n_ref = sum(len(refused(r)) for r in runs)
                cr = [credit(r) for r in runs]
                same = sum(accepted(r) == accepted(r2[(e, s, p)]) for r, s in zip(runs, seeds))
                bound = n_ref > 0 or any(c[0] or c[1] or c[2] for c in cr)
                rows.append([e, p, v, "**yes**" if bound else "no", n_ref, money(max(c[0] for c in cr)),
                             money(sum(c[1] for c in cr)), sum(1 for c in cr if c[2]), money(max(c[2] for c in cr)),
                             f"{same} of {len(seeds)}"])
    print(table(["entry", "policy", "variant", "binds", "refusals (3 seeds)", "most drawn", "interest (3 seeds)",
                 "runs in distress", "largest shortfall", "actions as v2"], rows) + "\n")

    print("## 3. Each run\n")
    rows = []
    for e in entries:
        for s in seeds:
            for p in POLICIES:
                a = r2[(e, s, p)]
                cells = [e, s, p, money(metric(a, "operatingProfitCents")), money(metric(a, "minCashCents")),
                         len(accepted(a))]
                for v in VARIANTS:
                    b = v3(e, s, p, v)
                    drawn, interest, short = credit(b)
                    ok = b["contract"]["status"] == "completed" and b["contract"]["replayVerified"]
                    cells += [money(metric(b, "operatingProfitCents")), money(metric(b, "minCashCents")),
                              len(accepted(b)), len(refused(b)), money(drawn), money(interest), money(short),
                              "ok" if ok else "**check**"]
                rows.append(cells)
    head = ["entry", "seed", "policy", "profit v2", "min cash v2", "accepted v2"]
    for v in VARIANTS:
        head += [f"profit {v}", f"min cash {v}", f"accepted {v}", f"refused {v}", f"most drawn {v}", f"interest {v}",
                 f"shortfall {v}", f"status {v}"]
    print(table(head, rows) + "\n")

    print("## 4. Version 3 minus version 2 (same practice, seed and policy)\n")
    rows = []
    for e in entries:
        for s in seeds:
            for p in POLICIES:
                a = r2[(e, s, p)]
                for v in VARIANTS:
                    b = v3(e, s, p, v)
                    dm = lambda n: (None if metric(a, n) is None or metric(b, n) is None else metric(b, n) - metric(a, n))
                    cash = b["result"]["window"]["money"]["cashEndCents"] - a["result"]["window"]["money"]["cashEndCents"]
                    rows.append([e, s, p, v, "yes" if accepted(a) == accepted(b) else "**no**",
                                 money(dm("operatingExpenseCents"), True), money(dm("operatingProfitCents"), True),
                                 money(dm("netRevenueCents"), True), money(cash, True), money(dm("minCashCents"), True),
                                 num(dm("tatP90Sec"), True, 60), num(dm("tatBreaches"), True), num(dm("examsCompleted"), True),
                                 num(dm("ordersLost"), True)])
    print(table(["entry", "seed", "policy", "variant", "actions same", "Δ expense", "Δ profit", "Δ revenue", "Δ cash end",
                 "Δ min cash", "Δ TAT p90 min", "Δ breaches", "Δ exams", "Δ lost"], rows) + "\n")

    print("## 5. Paired with no action (policy − noop, same entry and seed)\n")
    rows = []
    for e in entries:
        for s in seeds:
            for p in POLICIES[1:]:
                cells = [e, s, p]
                for get in (lambda q: r2[(e, s, q)], lambda q: v3(e, s, q, "credit"), lambda q: v3(e, s, q, "nocredit")):
                    a, b = get("noop"), get(p)
                    cells += [money(metric(b, "operatingProfitCents") - metric(a, "operatingProfitCents"), True),
                              num(metric(b, "tatBreaches") - metric(a, "tatBreaches"), True)]
                rows.append(cells)
    print(table(["entry", "seed", "policy", "Δ profit v2", "Δ breaches v2", "Δ profit credit", "Δ breaches credit",
                 "Δ profit nocredit", "Δ breaches nocredit"], rows) + "\n")

    print("## 6. Refusals and differing actions\n")
    any_diff = False
    for e in entries:
        for s in seeds:
            for p in POLICIES:
                for v in VARIANTS:
                    a, b = r2[(e, s, p)], v3(e, s, p, v)
                    if accepted(a) != accepted(b) or refused(b):
                        any_diff = True
                        fmt = lambda xs: [(t[:10], json.loads(x).get("type"), json.loads(x).get("employee"),
                                           json.loads(x).get("siteAllocationBP")) for t, x in xs]
                        print(f"* {e} {s} {p} {v}:")
                        print(f"  * version 2 accepted: {fmt(accepted(a))}")
                        print(f"  * version 3 accepted: {fmt(accepted(b))}")
                        print(f"  * refused insufficient_funds: {refused(b)}")
    if not any_diff:
        print("None: every policy took the same actions at the same times, and nothing was refused for funding.")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit("usage: analyze.py V2_RESULTS V3_RESULTS")
    main(sys.argv[1], sys.argv[2])
