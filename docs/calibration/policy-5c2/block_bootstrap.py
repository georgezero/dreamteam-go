#!/usr/bin/env python3
"""Slice 5C.2 exploratory diagnostic (development seeds only): the chance
that a plan passes the frozen service rule on a 40-seed evaluation,
estimated without assuming independent patients. Each replicate draws 40
seeds with replacement from the plan's development seeds, and within
each drawn seed resamples whole Monday-based weeks (with replacement)
inside T90 (days 1-90) and inside the tail (days 91-H), so day and week
dependence is kept. It applies the per-seed gates (urgent over the gated
period; soon and routine in T90 and the tail; evidence minimums), the
joint rule (at most floor(10% x 40) failing seeds) and the pooled rates.
Ten development seeds bound what this can show: between-seed variation
is only what those ten seeds contain. It informs; it selects nothing.

    python3 block_bootstrap.py RECORDS_DIR REPLICATES PLAN [PLAN...]
"""
import collections
import glob
import gzip
import json
import os
import random
import sys

GATE = {"urgent": 0.12, "soon": 0.32, "routine": 0.06}
MINC = {"urgent": 43, "soon": 88, "routine": 23}
PER_SEED = {"urgent": ["gated"], "soon": ["T90", "tail"], "routine": ["T90", "tail"]}


def weeks_of(path):
    """Per priority and window, the list of (n, breaches) per week."""
    r = json.load(gzip.open(path))
    t0, h = r["takeoverSec"], r["horizonEndSec"]
    t90 = t0 + 90 * 86400
    out = {q: {"T90": collections.defaultdict(lambda: [0, 0]), "tail": collections.defaultdict(lambda: [0, 0])} for q in GATE}
    for s in r["studies"]:
        a = s["acquiredSec"]
        if not (t0 < a <= h):
            continue
        q = s["priority"]
        due = a + r["tatTargetMin"][q] * 60
        b = 0 if (s["finalSec"] and s["finalSec"] <= due) else 1
        w = "T90" if a <= t90 else "tail"
        wk = (a // 86400 + 3) // 7
        out[q][w][wk][0] += 1
        out[q][w][wk][1] += b
    return {q: {w: list(v.values()) for w, v in out[q].items()} for q in out}


def main():
    d, reps, plans = sys.argv[1], int(sys.argv[2]), sys.argv[3:]
    print("| Plan | Replicates | P(pass joint rule, 40 seeds) | P(pass joint rule and pooled rates) | Mean failing seeds | P(seed fails) |")
    print("|---|---|---|---|---|---|")
    for plan in plans:
        seeds = [weeks_of(f) for f in sorted(glob.glob(os.path.join(d, plan + "-*.json.gz")))]
        rng = random.Random(20261018)
        ok_joint = ok_all = 0
        fails_tot = 0
        seed_fail = 0
        for _ in range(reps):
            fails = 0
            pooled = {(q, w): [0, 0] for q in GATE for w in ("T90", "tail")}
            for _ in range(40):
                s = rng.choice(seeds)
                samp = {}
                for q in GATE:
                    for w in ("T90", "tail"):
                        ws = s[q][w]
                        draw = [rng.choice(ws) for _ in ws] if ws else []
                        samp[(q, w)] = [sum(x[0] for x in draw), sum(x[1] for x in draw)]
                        pooled[(q, w)][0] += samp[(q, w)][0]
                        pooled[(q, w)][1] += samp[(q, w)][1]
                failed = False
                for q in GATE:
                    for w in PER_SEED[q]:
                        if w == "gated":
                            n = samp[(q, "T90")][0] + samp[(q, "tail")][0]
                            b = samp[(q, "T90")][1] + samp[(q, "tail")][1]
                        else:
                            n, b = samp[(q, w)]
                        if n < MINC[q] or b * 10000 > GATE[q] * 10000 * n:
                            failed = True
                fails += failed
            seed_fail += fails
            fails_tot += fails
            joint = fails <= 4
            pooled_ok = all(v[0] > 0 and v[1] * 10000 <= GATE[q] * 10000 * v[0] for (q, w), v in pooled.items())
            ok_joint += joint
            ok_all += joint and pooled_ok
        print("| %s | %d | %.3f | %.3f | %.2f | %.3f |" % (plan, reps, ok_joint / reps, ok_all / reps, fails_tot / reps, seed_fail / (40 * reps)))


if __name__ == "__main__":
    main()
