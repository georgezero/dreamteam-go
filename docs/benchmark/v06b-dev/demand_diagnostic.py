#!/usr/bin/env python3
"""Demand diagnostic dreamteam.demand-diagnostic/v1 (V0.6b; specification:
docs/BENCHMARK_SUITE.md §8). From a suite run's results, for each entry,
seed and 30-day window of the run, per operating day:

* generated demand: referral orders received (the result calendar's
  `orders`, the simulator's count of orders as they arrive, before any
  booking) and walk-ins arrived (`walkIns`);
* completed imaging: exams completed (`exams`);
* lost orders (`lost`).

An operating day is a calendar day on which the scenario's opening hours
(from the saved session's embedded scenario) are not closed. These are
different quantities: completed imaging and lost orders are outcomes of
capacity and decisions, and generated demand is never inferred from them.

Declared conditions (prospective, for the demand-growth entry; evaluated
with no action):
  G1  referral orders per operating day are higher in days 61-90 than in
      days 1-30, on every seed;
  G2  the ratio of demand growth's referral orders per operating day to
      reporting backlog's, on the same seed and window, rises from days
      1-30 to days 61-90, and exceeds 1 in days 31-60 and 61-90, on every
      seed.
Reliability check:
  R1  generated demand (orders and walk-ins, day by day) is identical
      across the policies of an entry and seed; if not, the diagnostic
      says that generated demand depends on decisions and is not a clean
      measure.

    python3 -I -B docs/benchmark/v06b-dev/demand_diagnostic.py RESULTS > demand.md

Nothing is written elsewhere."""

import json
import os
import sys

VERSION = "dreamteam.demand-diagnostic/v1"
WINDOWS = ((0, 30), (30, 60), (60, 90))
GROWTH, BACKLOG = "demand-growth", "reporting-backlog"


def load(d):
    with open(os.path.join(d, "contract.json"), encoding="utf-8") as f:
        k = json.load(f)
    runs = {}
    for r in k["runs"]:
        if r["status"] != "completed":
            continue
        rd = os.path.join(d, "runs", r["id"])
        with open(os.path.join(rd, "result.json"), encoding="utf-8") as f:
            cal = json.load(f)["calendar"]
        with open(os.path.join(rd, "session.json"), encoding="utf-8") as f:
            hours = json.load(f)["scenario"]["resolved"]["hours"]
        runs[(r["entry"], r["seed"], r["policy"]["id"])] = (cal, hours)
    return k, runs


def windows(cal, hours):
    out = []
    for a, b in WINDOWS:
        days = cal[a:b]
        if len(days) < b - a:
            out.append(None)
            continue
        op = [x for x in days if hours.get(x["weekday"]) is not None]
        wk = [x for x in days if x["weekday"] not in ("Sat", "Sun")]
        n = len(op)
        out.append({
            "days": len(days), "operatingDays": n, "weekdays": len(wk),
            "orders": sum(x["orders"] for x in days), "walkIns": sum(x["walkIns"] for x in days),
            "exams": sum(x["exams"] for x in days), "lost": sum(x["lost"] for x in days),
            "ordersPerOp": sum(x["orders"] for x in op) / n, "walkInsPerOp": sum(x["walkIns"] for x in op) / n,
            "examsPerOp": sum(x["exams"] for x in op) / n, "lostPerOp": sum(x["lost"] for x in op) / n,
            "ordersPerWeekday": sum(x["orders"] for x in wk) / len(wk),
            "offDayOrders": sum(x["orders"] for x in days if hours.get(x["weekday"]) is None),
        })
    return out


def table(head, rows):
    out = ["| " + " | ".join(head) + " |", "|" + "---|" * len(head)]
    return "\n".join(out + ["| " + " | ".join(str(c) for c in r) + " |" for r in rows])


def main(d):
    k, runs = load(d)
    p = print
    seeds = k["seeds"]
    p(f"# Demand diagnostic ({VERSION})\n")
    p(f"Suite {k['suite']['id']} version {k['suite']['version']}, suite run {k['suiteRun']}, seeds "
      f"{', '.join(map(str, seeds))}, {k['days']} days. Per operating day (opening hours not closed); "
      "30-day windows from the takeover.\n")

    p("## R1: generated demand across policies\n")
    rows = []
    for e in [x["id"] for x in k["entries"]]:
        for s in seeds:
            cals = [(pol, runs[(e, s, pol)][0]) for pol in [x["id"] for x in k["policies"]] if (e, s, pol) in runs]
            ref = [(x["orders"], x["walkIns"]) for x in cals[0][1]] if cals else None
            same = all([(x["orders"], x["walkIns"]) for x in c] == ref for _, c in cals)
            rows.append([e, s, len(cals), "identical" if same else "DIFFERS"])
    p(table(["entry", "seed", "policies", "orders and walk-ins, day by day"], rows))

    p("\n## Per operating day, no action\n")
    rows = []
    for e in [x["id"] for x in k["entries"]]:
        for s in seeds:
            if (e, s, "noop") not in runs:
                continue
            for i, w in enumerate(windows(*runs[(e, s, "noop")])):
                if w is None:
                    continue
                rows.append([e, s, f"{WINDOWS[i][0] + 1}-{WINDOWS[i][1]}", f"{w['operatingDays']} ({w['weekdays']} weekdays)",
                             f"{w['ordersPerOp']:.1f}", f"{w['walkInsPerOp']:.1f}", f"{w['examsPerOp']:.1f}",
                             f"{w['lostPerOp']:.1f}", f"{w['ordersPerWeekday']:.1f}", w["offDayOrders"]])
    p(table(["entry", "seed", "days", "operating days", "referral orders", "walk-ins", "exams completed", "lost orders",
             "orders per weekday (secondary)", "orders on closed days"], rows))

    p("\n## Declared conditions (demand growth, no action)\n")
    rows = []
    for s in seeds:
        g, b = runs.get((GROWTH, s, "noop")), runs.get((BACKLOG, s, "noop"))
        if not g or not b:
            rows.append([s, "–", "not evaluated (a run is missing)", "–", "not evaluated"])
            continue
        gw, bw = windows(*g), windows(*b)
        if None in gw or None in bw:
            rows.append([s, "–", "not evaluated (the run is shorter than 90 days)", "–", "not evaluated"])
            continue
        g1 = gw[2]["ordersPerOp"] > gw[0]["ordersPerOp"]
        ratio = [gw[i]["ordersPerOp"] / bw[i]["ordersPerOp"] for i in range(3)]
        g2 = ratio[2] > ratio[0] and ratio[1] > 1 and ratio[2] > 1
        rows.append([s, f"{gw[0]['ordersPerOp']:.1f} → {gw[2]['ordersPerOp']:.1f}", "holds" if g1 else "DOES NOT HOLD",
                     ", ".join(f"{r:.3f}" for r in ratio), "holds" if g2 else "DOES NOT HOLD"])
    p(table(["seed", "G1: referral orders per operating day, days 1-30 → 61-90", "G1",
             "G2: ratio to reporting backlog, by window", "G2"], rows))


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    main(sys.argv[1])
