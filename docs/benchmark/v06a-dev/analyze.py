#!/usr/bin/env python3
"""V0.6a characterization tables (PROTOCOL.md §2) from a `dreamteam
benchmark run` result directory: the harness's report.json, and each run's
result.json and decisions.json. Existing metrics only, grouped by the
dimensions of the suite file; no composite score, no ranking, no test of
significance.

    python3 -I -B docs/benchmark/v06a-dev/analyze.py RESULTS > tables.md

Nothing is written elsewhere."""

import json
import os
import statistics
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
SUITE = os.path.join(HERE, "..", "suite", "dreamteam-dev-1.json")
STAFFING = ("hire", "terminate", "set_roster", "set_reading_allocation", "contract_teleradiology")
DIMS = [("operational", "Operational performance"), ("financial", "Financial performance"),
        ("staffingCapacity", "Staffing and capacity"), ("serviceReliability", "Service reliability"),
        ("sustainability", "Long-term sustainability"), ("actions", "Actions")]
PRIMARY = {
    "financial-pressure": ["operatingProfitCents", "operatingExpenseCents", "minCashCents", "daysBelowZero"],
    "reporting-backlog": ["tatP90Sec", "tatBreaches", "tatOverTarget", "backlogEnd"],
    "demand-growth": ["ordersLost", "lostRate", "examsCompleted", "waitToExamP90Sec"],
    "staffing-shortage": ["ordersLost", "noCoverage", "overtimeSec", "waitToExamP90Sec"],
    "stable": ["operatingProfitCents", "tatP90Sec", "ordersLost"],
}


def num(v):
    """A metric value as a number: ratios as num/den, absent as None."""
    if isinstance(v, dict):
        return v["num"] / v["den"] if v.get("den") else None
    return v


def fmt(name, v, signed=False):
    if v is None:
        return "–"
    s = "+" if signed and v > 0 else ""
    if name.endswith("Cents"):
        return f"{'-' if v < 0 else s}${abs(v) / 100:,.0f}"
    if name.endswith("Sec"):
        return f"{s}{v / 60:,.0f} min"
    if name in ("lostRate", "margin"):
        return f"{s}{100 * v:.1f}%"
    return f"{s}{v:,.0f}" if isinstance(v, (int, float)) else str(v)


def table(head, rows):
    out = ["| " + " | ".join(head) + " |", "|" + "---|" * len(head)]
    return "\n".join(out + ["| " + " | ".join(str(c) for c in r) + " |" for r in rows])


def load(d):
    with open(os.path.join(d, "report.json"), encoding="utf-8") as f:
        rep = json.load(f)
    with open(SUITE, encoding="utf-8") as f:
        suite = json.load(f)
    entry = {e["scenario"]: e for e in suite["scenarios"]}
    pairs = {(p["scenario"], p["seed"], p["policy"]): {x["name"]: num(x["value"]) for x in p["deltas"]}
             for p in rep.get("pairs") or []}
    runs = []
    for r in rep["runs"]:
        rd = os.path.join(d, "runs", r["id"])
        res = json.load(open(os.path.join(rd, "result.json"), encoding="utf-8"))
        dec = json.load(open(os.path.join(rd, "decisions.json"), encoding="utf-8"))
        accepted = {}
        for p in dec:
            for b in p.get("batches") or []:
                for x in b.get("results") or []:
                    if x.get("ok"):
                        t = x["action"]["type"]
                        accepted[t] = accepted.get(t, 0) + 1
        cal = res.get("calendar") or []
        runs.append({"id": r["id"], "scenario": r["scenario"], "suite": entry[r["scenario"]]["id"], "seed": r["seed"],
                     "policy": r["policy"]["id"], "status": r["status"], "replay": r.get("replayVerified"),
                     "match": r.get("actionsMatchSession"), "t90": r.get("t90") or {},
                     "fingerprint": res["scenario"]["fingerprint"],
                     "metrics": {m["name"]: num(m["value"]) for m in r.get("metrics") or []},
                     "delta": pairs.get((r["scenario"], r["seed"], r["policy"][ "id"])),
                     "accepted": accepted,
                     "orders": [sum(x["orders"] for x in cal[i:i + 30]) for i in (0, 30, 60)]})
    return rep, suite, runs


def spread(vals):
    vals = [v for v in vals if v is not None]
    if not vals:
        return None
    s = sorted(vals)
    return s[0], s[(len(s) - 1) // 2], s[-1]


def main(d):
    rep, suite, runs = load(d)
    p = print
    order = [e["id"] for e in suite["scenarios"]]
    seeds = sorted({r["seed"] for r in runs})
    by = {(r["suite"], r["seed"], r["policy"]): r for r in runs}

    p("## Runs\n")
    p(table(["scenario", "seed", "policy", "status", "replay verified", "actions match", "fingerprint", "T90"],
            [[r["suite"], r["seed"], r["policy"], r["status"], "yes" if r["replay"] else "NO",
              "yes" if r["match"] else "NO", r["fingerprint"][:12] + "…",
              r["t90"].get("verdict") or r["t90"].get("status", "–")] for r in runs]))
    p("\n**Starting states** (the harness's initial conditions: every policy's run of a scenario and seed "
      "starts from the same state):\n")
    ic = rep.get("initialConditions") or []
    p(table(["scenario", "seed", "runs", "identical", "state digest"],
            [[x["scenario"], x["seed"], x["runs"], "yes" if x["identical"] else "NO", x["stateDigest"][:16] + "…"] for x in ic]))
    fps = {r["suite"]: r["fingerprint"] for r in runs}
    p(f"\nScenario fingerprints all differ: {'yes' if len(set(fps.values())) == len(fps) else 'NO'}.")
    same = sorted({(x["seed"], tuple(sorted(e["id"] for e in suite["scenarios"] if e["scenario"] in (x["scenario"], y["scenario"]))))
                   for x in ic for y in ic if x["seed"] == y["seed"] and x["scenario"] < y["scenario"] and x["stateDigest"] == y["stateDigest"]})
    expected = all(pair == ("demand-growth", "reporting-backlog") for _, pair in same)
    p("Scenarios sharing a starting state on a seed: " + ("; ".join(f"{a} and {b} (seed {s})" for s, (a, b) in same) or "none") +
      (" (expected: demand growth differs from reporting backlog only from November)" if same and expected else
       "" if not same else " (NOT EXPECTED)") + ".")

    p("\n## No-action profile (minimum / median / maximum over the seeds)\n")
    for key, title in DIMS:
        names = suite["dimensions"][key]
        p(f"\n**{title}**\n")
        rows = []
        for sid in order:
            row = [sid]
            for n in names:
                sp = spread([by[(sid, s, "noop")]["metrics"].get(n) for s in seeds if (sid, s, "noop") in by])
                row.append("–" if sp is None else " / ".join(fmt(n, v) for v in sp))
            rows.append(row)
        p(table(["scenario"] + names, rows))

    p("\n## Intended conditions (no action, each seed; PROTOCOL.md §2.3)\n")
    rows = []
    for s in seeds:
        ref = by.get(("reporting-backlog", s, "noop"))
        for sid in order:
            r = by.get((sid, s, "noop"))
            if not r or not ref:
                continue
            m, rm = r["metrics"], ref["metrics"]
            if sid == "financial-pressure":
                ok, what = m["operatingProfitCents"] < 0, f"profit {fmt('operatingProfitCents', m['operatingProfitCents'])}"
            elif sid == "reporting-backlog":
                share = m["tatBreaches"] / m["reportsFinal"]
                ok = m["tatP90Sec"] > 600 * 60 and share > 0.05
                what = f"TAT p90 {fmt('tatP90Sec', m['tatP90Sec'])}, breaches {100 * share:.1f}% of final reports"
            elif sid == "demand-growth":
                ok = r["orders"][2] > r["orders"][0] and m["ordersReceived"] > rm["ordersReceived"]
                what = f"orders per 30 days {', '.join(f'{x:,}' for x in r['orders'])}; received {m['ordersReceived']:,} vs {rm['ordersReceived']:,}"
            elif sid == "staffing-shortage":
                ok, what = m["ordersLost"] > rm["ordersLost"], f"lost {m['ordersLost']:,} vs {rm['ordersLost']:,}"
            else:
                ok = m["operatingProfitCents"] > 0 and m["ordersLost"] < rm["ordersLost"] and m["tatP90Sec"] < rm["tatP90Sec"]
                what = (f"profit {fmt('operatingProfitCents', m['operatingProfitCents'])}; lost {m['ordersLost']:,} vs "
                        f"{rm['ordersLost']:,}; TAT p90 {fmt('tatP90Sec', m['tatP90Sec'])} vs {fmt('tatP90Sec', rm['tatP90Sec'])}")
            rows.append([s, sid, "holds" if ok else "DOES NOT HOLD", what])
    p(table(["seed", "scenario", "condition", "values"], rows))

    p("\n## Distinctness: primary metrics whose no-action ranges do not overlap\n")
    rows = []
    for i, a in enumerate(order):
        for b in order[i + 1:]:
            sep = []
            for n in dict.fromkeys(PRIMARY[a] + PRIMARY[b]):
                ra = spread([by[(a, s, "noop")]["metrics"].get(n) for s in seeds])
                rb = spread([by[(b, s, "noop")]["metrics"].get(n) for s in seeds])
                if ra and rb and (ra[2] < rb[0] or rb[2] < ra[0]):
                    sep.append(n)
            rows.append([a, b, len(sep), ", ".join(sep) or "none"])
    p(table(["scenario", "scenario", "separated", "metrics"], rows))

    p("\n## Baselines paired with no action (policy − noop, same scenario and seed)\n")
    for key, title in DIMS:
        names = suite["dimensions"][key]
        p(f"\n**{title}**\n")
        rows = [[r["suite"], r["seed"], r["policy"]] + [fmt(n, r["delta"].get(n), True) for n in names]
                for r in runs if r["delta"] is not None]
        p(table(["scenario", "seed", "policy"] + names, rows))
    p("\n**Accepted actions by type**\n")
    p(table(["scenario", "seed", "policy", "operations", "staffing", "by type"],
            [[r["suite"], r["seed"], r["policy"], sum(v for k, v in r["accepted"].items() if k not in STAFFING),
              sum(v for k, v in r["accepted"].items() if k in STAFFING),
              "; ".join(f"{k}: {v}" for k, v in sorted(r["accepted"].items())) or "none"]
             for r in runs if r["policy"] != "noop"]))

    p("\n## T90\n")
    p(table(["scenario", "seed", "policy", "status", "verdict", "classification"],
            [[r["suite"], r["seed"], r["policy"], r["t90"].get("status", "–"), r["t90"].get("verdict", "–"),
              r["t90"].get("classification", "–")] for r in runs]))
    p("\n## Missing metrics\n")
    for m in suite["missing"]:
        p(f"* {m}")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    main(sys.argv[1])
