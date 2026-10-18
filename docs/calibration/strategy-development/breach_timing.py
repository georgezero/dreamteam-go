#!/usr/bin/env python3
"""Where a plan's TAT breaches happen in the week, from development
records (DT_STRAT_RECORDS, gzip JSON): for every study acquired in
(t0, H] that breached (not final by its due time), the local weekday and
hour it was acquired, and which teleradiology windows (standard-v1:
weeknights Mon-Fri 18:00-08:00, saturday 14:00-24:00, sunday all day)
overlap the interval from its entry to the reading worklist to its due
time, i.e. the windows in which a remote reader could have claimed it in
time. An upper bound on what a contract can recover: two lanes, and the
employed readers claim first. Reads records only.

    python3 breach_timing.py RECORDS_DIR/SET PLAN [PLAN...]
"""
import collections
import glob
import gzip
import json
import os
import sys

DAYS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
TAKEOVER_WD, TAKEOVER_MIN = 1, 6 * 60  # Tuesday 06:00


def local(r, sec):
    """(weekday index, minute of day) of a simulation second."""
    m = (sec - r["takeoverSec"]) // 60 + TAKEOVER_WD * 1440 + TAKEOVER_MIN
    return (m // 1440) % 7, m % 1440


def week_min(r, sec):
    wd, mm = local(r, sec)
    return wd * 1440 + mm


# windows as [start, end) minute-of-week intervals
WIN = {
    "weeknights": [(d * 1440 + 18 * 60, d * 1440 + 32 * 60) for d in range(5)],
    "saturday": [(5 * 1440 + 14 * 60, 6 * 1440)],
    "sunday": [(6 * 1440, 7 * 1440)],
}


def overlaps(r, a, b, w):
    """Does window w cover any instant of [a, b) (simulation seconds)?"""
    if b <= a:
        return False
    if b - a >= 7 * 86400:
        return True
    s = week_min(r, a)
    e = s + (b - a) // 60
    for lo, hi in WIN[w]:
        for k in (0, 10080):
            if lo + k < e and s < hi + k:
                return True
    return False


def main():
    d, plans = sys.argv[1], sys.argv[2:]
    for plan in plans:
        tot = collections.Counter()
        br = collections.Counter()
        where = collections.Counter()
        cover = collections.Counter()
        for f in sorted(glob.glob(os.path.join(d, plan + "-*.json.gz"))):
            r = json.load(gzip.open(f))
            t0, h = r["takeoverSec"], r["horizonEndSec"]
            for s in r["studies"]:
                if not (t0 < s["acquiredSec"] <= h):
                    continue
                q = s["priority"]
                tot[q] += 1
                due = s["acquiredSec"] + r["tatTargetMin"][q] * 60
                if s["finalSec"] and s["finalSec"] <= due:
                    continue
                br[q] += 1
                wd, mm = local(r, s["acquiredSec"])
                where[(q, DAYS[wd], mm // 60)] += 1
                entered = s["enteredSec"] or s["acquiredSec"]
                ws = tuple(w for w in WIN if overlaps(r, entered, due, w))
                cover[(q, ws)] += 1
        print("##", plan)
        for q in ["urgent", "soon", "routine"]:
            print("%s: %d of %d breached (%.1f%%)" % (q, br[q], tot[q], 100 * br[q] / max(tot[q], 1)))
            for (qq, ws), n in sorted(cover.items(), key=lambda x: -x[1]):
                if qq == q:
                    print("   windows reachable before due %-34s %5d" % (",".join(ws) or "none", n))
            byday = collections.Counter()
            for (qq, day, hr), n in where.items():
                if qq == q:
                    byday[(day, hr)] += n
            top = sorted(byday.items(), key=lambda x: -x[1])[:12]
            print("   acquired (top): " + ", ".join("%s %02d:00 %d" % (dd, hh, n) for (dd, hh), n in top))


if __name__ == "__main__":
    main()
