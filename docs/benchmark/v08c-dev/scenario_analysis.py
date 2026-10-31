#!/usr/bin/env python3
"""The V0.8c low-liquidity scenario's deterministic analysis
(dreamteam.low-liquidity-analysis/v1; PLAN.md §5).

    python3 -I -B docs/benchmark/v08c-dev/scenario_analysis.py --binary ./dt \
        --out docs/benchmark/v08c-dev/scenario-analysis.json

No model and no policy: the control practice (suite dreamteam-dev version
3's financial-pressure-credit) and the variant are each played with no
action through `dreamteam agent --stdio` on the analysis seed, 41900, and
their public funding observation is read at the 30 daily decision points
(07:00). The rule fixed in PLAN.md §5 gives the variant's opening cash:

    the control's opening cash, less its lowest no-action funding headroom
    at the 30 decision points, rounded down to a whole $10,000.

The script applies the rule to the control's figures, checks that the
bundled variant has that opening cash and nothing else changed (the suite
test checks the rest), and records both trajectories. It exits 1 if the
variant does not follow the rule.
"""

import argparse
import datetime
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.join(HERE, "..", "..", "..")
sys.path.insert(0, os.path.join(ROOT, "clients", "python"))
from dreamteam_agent import AgentClient  # noqa: E402

SCHEMA = "dreamteam.low-liquidity-analysis/v1"
CONTROL, VARIANT = "bench-financial-pressure-ot-credit", "bench-financial-pressure-ot-lowliquidity"
SEED, POINTS, AT = 41900, 30, "07:00:00"
STEP = 1_000_000  # $10,000, in cents
# Commitments the V0.8b probes measured on this practice (PROBES.md), for
# description only: how often each would not fit under no action.
REFERENCE = [("paid radiologist extension, 2 h", 57_700), ("allocation raise, 3 points", 147_945),
             ("teleradiology contract retainer", 300_000), ("technologist hire", 1_045_706),
             ("allocation raise, 40 points", 1_972_602)]


def trajectory(binary, scenario):
    with AgentClient(binary, ["-scenario", scenario, "-seed", str(SEED)]) as c:
        hello = c.request("hello")
        start = c.request("digests")["time"]
        rows = []
        for d in range(POINTS):
            day = datetime.date.fromisoformat(start[:10]) + datetime.timedelta(days=d)
            c.request("advance", {"to": f"{day.isoformat()}T{AT}"})
            o = c.request("observe", {"compact": True, "recentLog": 0})["observation"]
            f = o["finance"]
            fin = f.get("financing") or {}
            s = fin.get("solvency") or {}
            rows.append({"date": o["date"], "cashCents": f["cashCents"],
                         "currentLiabilitiesCents": s.get("currentLiabilitiesCents"),
                         "reservedCents": s.get("reservedCents"), "fundingHeadroomCents": s.get("fundingHeadroomCents"),
                         "creditDrawnCents": s.get("creditDrawnCents"), "interestChargedCents": s.get("interestChargedCents"),
                         "shortfallCents": fin.get("shortfallCents"), "daysBelowZero": fin.get("daysBelowZero")})
    sc = hello.get("scenario") or {}
    return {"scenario": scenario, "fingerprint": sc.get("fingerprint"), "engine": hello.get("engine"),
            "start": start, "openingCashCents": rows[0]["cashCents"], "points": rows}


def summary(t):
    h = [r["fundingHeadroomCents"] for r in t["points"]]
    return {"headroomMinCents": min(h), "headroomMaxCents": max(h),
            "creditDrawnMaxCents": max(r["creditDrawnCents"] or 0 for r in t["points"]),
            "shortfallMaxCents": max(r["shortfallCents"] or 0 for r in t["points"]),
            "pointsBelow": [{"commitment": n, "cents": c, "points": sum(1 for x in h if x < c)} for n, c in REFERENCE]}


def main(argv=None):
    ap = argparse.ArgumentParser()
    ap.add_argument("--binary", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    control, variant = trajectory(a.binary, CONTROL), trajectory(a.binary, VARIANT)
    low = min(r["fundingHeadroomCents"] for r in control["points"])
    reduction = (low // STEP) * STEP
    want = control["openingCashCents"] - reduction
    with open(os.path.join(ROOT, "scenarios", VARIANT + ".json"), encoding="utf-8") as f:
        file_cash = json.load(f).get("openingCashCents")
    ok = file_cash == want == variant["openingCashCents"]
    doc = {"schema": SCHEMA, "seed": SEED, "points": POINTS, "at": AT[:5],
           "rule": "variant opening cash = control opening cash - (lowest no-action funding headroom at the 30 "
                   "decision points, rounded down to a whole $10,000)",
           "control": dict(control, summary=summary(control)), "variant": dict(variant, summary=summary(variant)),
           "result": {"controlLowestHeadroomCents": low, "reductionCents": reduction,
                      "variantOpeningCashCents": want, "variantFileOpeningCashCents": file_cash, "followsRule": ok}}
    with open(a.out, "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True)
        f.write("\n")
    s = doc["variant"]["summary"]
    print(f"control lowest headroom ${low / 100:,.2f}; reduction ${reduction / 100:,.0f}; variant opening cash "
          f"${want / 100:,.0f} ({'follows' if ok else 'DOES NOT follow'} the rule); variant headroom "
          f"${s['headroomMinCents'] / 100:,.2f} to ${s['headroomMaxCents'] / 100:,.2f}, credit drawn up to "
          f"${s['creditDrawnMaxCents'] / 100:,.2f}, shortfall ${s['shortfallMaxCents'] / 100:,.2f}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
