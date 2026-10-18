#!/usr/bin/env python3
"""Slice 5C.2 exploratory diagnostic (development seeds only): how breaches
cluster in time. For each plan and priority, studies acquired in the gated
period (t0, H] are grouped by their acquisition day, and by their rota
week (Monday-based); the within-seed intra-cluster correlation of the
breach indicator (one-way ANOVA estimator) and the design effect
1 + (m - 1) rho are computed per seed and averaged. It reads records
written by DT_STRAT_RECORDS and decides nothing; the policy's thresholds
and allowance were not chosen from it. Its only use is to bound the
dependence range of the synthetic operating characteristics.

    python3 cluster_diagnostics.py RECORDS_DIR PLAN [PLAN...] > clustering.md
"""
import collections
import glob
import gzip
import json
import os
import sys

TAKEOVER_WD = 1  # Tuesday
PRI = ["urgent", "soon", "routine"]


def icc(groups):
    """One-way ANOVA ICC of 0/1 outcomes in groups (lists); mean group size."""
    groups = [g for g in groups if g]
    k = len(groups)
    n = sum(len(g) for g in groups)
    if k < 2 or n <= k:
        return None, 0
    mean = sum(sum(g) for g in groups) / n
    ssb = sum(len(g) * (sum(g) / len(g) - mean) ** 2 for g in groups)
    ssw = sum(sum((x - sum(g) / len(g)) ** 2 for x in g) for g in groups)
    msb, msw = ssb / (k - 1), ssw / (n - k)
    n0 = (n - sum(len(g) ** 2 for g in groups) / n) / (k - 1)
    if msb + (n0 - 1) * msw == 0:
        return 0.0, n / k
    return (msb - msw) / (msb + (n0 - 1) * msw), n / k


def main():
    d, plans = sys.argv[1], sys.argv[2:]
    print("| Plan | Priority | Breach % | Day ICC (raw) | Day ICC (weekday-adjusted) | Day size | Day design effect (adjusted) | Week ICC | Week size | Week design effect |")
    print("|---|---|---|---|---|---|---|---|---|---|")
    for plan in plans:
        files = sorted(glob.glob(os.path.join(d, plan + "-*.json.gz")))
        acc = {q: collections.defaultdict(list) for q in PRI}
        rate = {q: [0, 0] for q in PRI}
        for f in files:
            r = json.load(gzip.open(f))
            t0, h = r["takeoverSec"], r["horizonEndSec"]
            days = {q: collections.defaultdict(list) for q in PRI}
            weeks = {q: collections.defaultdict(list) for q in PRI}
            for s in r["studies"]:
                if not (t0 < s["acquiredSec"] <= h):
                    continue
                q = s["priority"]
                due = s["acquiredSec"] + r["tatTargetMin"][q] * 60
                b = 0 if (s["finalSec"] and s["finalSec"] <= due) else 1
                rate[q][0] += b
                rate[q][1] += 1
                day = (s["acquiredSec"] - t0 + 6 * 3600) // 86400  # local days from the takeover's midnight
                days[q][day].append(b)
                weeks[q][(day + TAKEOVER_WD) // 7].append(b)
            for q in PRI:
                # Days with the weekday's mean rate removed: the random
                # day-to-day component, without the fixed weekday pattern.
                wd = collections.defaultdict(list)
                for day, g in days[q].items():
                    wd[(day + TAKEOVER_WD) % 7] += g
                wmean = {k: sum(v) / len(v) for k, v in wd.items()}
                resid = {day: [x - wmean[(day + TAKEOVER_WD) % 7] for x in g] for day, g in days[q].items()}
                for lvl, g in (("day", days[q]), ("dayadj", resid), ("week", weeks[q])):
                    rho, m = icc(list(g.values()))
                    if rho is not None:
                        acc[q][lvl].append((rho, m))
        for q in PRI:
            def avg(lvl):
                xs = acc[q][lvl]
                if not xs:
                    return "—", "—", "—"
                rho = sum(x[0] for x in xs) / len(xs)
                m = sum(x[1] for x in xs) / len(xs)
                return "%.3f" % rho, "%.1f" % m, "%.2f" % (1 + (m - 1) * rho)
            dd, da, ww = avg("day"), avg("dayadj"), avg("week")
            print("| %s | %s | %.1f | %s | %s | %s | %s | %s | %s | %s |" % (plan, q, 100 * rate[q][0] / max(rate[q][1], 1), dd[0], da[0], dd[1], da[2], *ww))


if __name__ == "__main__":
    main()
