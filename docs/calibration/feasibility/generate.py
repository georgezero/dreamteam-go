#!/usr/bin/env python3
"""Generates the V0.2 Step 0 feasibility-sweep manifests (static overlays on
the established practice). Run from the repository root:

    python3 docs/calibration/feasibility/generate.py

then run them with the calibration harness:

    DT_EXP_DIR=../../docs/calibration/feasibility/experiments \
    DT_RES_DIR=../../docs/calibration/feasibility/results \
    go test -tags calibration ./internal/sim -run 'CalibrationExperiments$' -v -timeout 600m

Only the documented levers move: referral volume and source weighting,
technologist coverage at documented pay rates, US slot packing and
radiologist site allocation. Reimbursement, payers, patient behaviour,
exam and read durations and every expense rate are the base scenario's.
"""
import json, os, sys

ROOT = os.path.dirname(os.path.abspath(__file__))
BASE = json.load(open(os.path.join(ROOT, "../../../scenarios/base/established.json")))
WEEKDAYS = ["Mon", "Tue", "Wed", "Thu", "Fri"]

# --- demand ----------------------------------------------------------------
SRC = {s["id"]: s["baseDailyMilliOrders"] for s in BASE["referralSources"]}
BROAD = {"SRC1", "SRC5"}  # primary care, urgent care: XR-heavy
def demand(total):
    """Uniform scaling: every source x total."""
    return [{"id": i, "baseDailyMilliOrders": round(v * total)} for i, v in SRC.items()]
def weighted(total, broad=1.0):
    """Imaging-weighted: primary/urgent care x broad, the US/CT/MR-heavy
    sources (orthopedics, OB/GYN, oncology, other) scaled so the total
    matches uniform x total."""
    b = sum(v for i, v in SRC.items() if i in BROAD)
    o = sum(v for i, v in SRC.items() if i not in BROAD)
    k = (sum(SRC.values()) * total - b * broad) / o
    return [{"id": i, "baseDailyMilliOrders": round(v * (broad if i in BROAD else k))} for i, v in SRC.items()]

DEMAND = {
    "D100": None,
    "D115": demand(1.15),
    "D130": demand(1.30),
    "W115": weighted(1.15, 1.0),
    "W130": weighted(1.30, 1.10),
    # Diagnostic only, beyond the documented x1.15-1.30 range: how much
    # demand a margin needs. Not a candidate.
    "W150": weighted(1.50, 1.20),
}

# --- technologists (documented pay rates; IDs after the base E01-E08) -------
# Part-time evening shifts (16:00-20:00) are not expressible: scenario
# validation requires shift start + maxShiftHours (12 h) to end by midnight,
# so no shift may start after 12:00 (internal/scenario/validate.go). Evening
# coverage is therefore a full 11:30-20:00 shift.
def shift(start, end, room, brk=None, days=WEEKDAYS):
    s = {"start": start, "end": end, "assignment": room}
    if brk:
        s["breaks"] = [brk]
    return {d: dict(s) for d in days}
def tech(eid, name, skills, rate, roster):
    return {"id": eid, "name": name, "role": "technologist", "skills": skills,
            "pay": {"type": "hourly", "rateCents": rate}, "roster": roster}
STAFF = {
    # XR/CT day technologist on R3 (CT mornings are covered today only by
    # Riley from X-ray); the step 18 three-technologist experiment's Jamie.
    "XRCT": tech("E09", "Jamie", ["CT", "XR"], 3800, shift("08:00", "16:30", "R3", ["12:30", "13:00"])),
    # US evening: the template runs to 20:00, Morgan works to 18:00.
    "USfull": tech("E10", "Drew", ["US"], 4400, shift("11:30", "20:00", "R2", ["15:30", "16:00"])),
    # MR evening: the template's 16:30-19:30 slots are unstaffed today.
    "MRfull": tech("E11", "Alex", ["MR"], 4500, shift("11:30", "20:00", "R4", ["15:30", "16:00"])),
}

# --- US slot packing ----------------------------------------------------------
def hm(m): return "%02d:%02d" % (m // 60, m % 60)
def mins(s): h, m = s.split(":"); return int(h) * 60 + int(m)
def us40():
    """US_45 slots at 40 minutes, weekday template reflowed in its blocks
    09:30-13:30 and 14:00-20:00 (the 17:45-18:00 gap closed), in the base
    order; the freed minutes of the afternoon block hold one more US_45.
    Saturday is unchanged."""
    wk, sat = BASE["templates"]["R2"]
    seq = []
    for s in wk["slots"]:
        for _ in range(s.get("count", 1)):
            seq.append((mins(s["start"]), s["category"]))
    out = []
    for lo, hi in ((mins("09:30"), mins("13:30")), (mins("14:00"), mins("20:00"))):
        t = lo
        for start, cat in seq:
            if lo <= start < hi:
                ln = 40 if cat == "US_45" else 30
                out.append({"start": hm(t), "lenMin": ln, "category": cat})
                t += ln
        while t + 40 <= hi:
            out.append({"start": hm(t), "lenMin": 40, "category": "US_45"})
            t += 40
    return [{"days": wk["days"], "slots": out}, sat]

ALLOC = {"A60": None, "A55": (5500, 4500), "A50": (5000, 4000), "A65": (6500, 5500)}

def overlay(d="D100", staff=(), pack=False, alloc="A60"):
    o = {}
    if DEMAND[d]:
        o["referralSources"] = DEMAND[d]
    emps = [STAFF[s] for s in staff]
    if ALLOC[alloc]:
        emps += [{"id": "E07", "siteAllocationBP": ALLOC[alloc][0]}, {"id": "E08", "siteAllocationBP": ALLOC[alloc][1]}]
    if emps:
        o["employees"] = emps
    if pack:
        o["templates"] = {"R2": us40()}
    return o

def name(d, staff, pack, alloc):
    parts = [d] + list(staff) + (["P40"] if pack else []) + ([alloc] if alloc != "A60" else [])
    return "-".join(parts)

def describe(d, staff, pack, alloc):
    bits = {"D100": "documented referral demand", "D115": "referral demand x1.15 (every source)",
            "D130": "referral demand x1.30 (every source)",
            "W115": "referral demand x1.15, imaging-weighted (primary/urgent care x1.00, the US/CT/MR-heavy sources x%.3f)" % ((sum(SRC.values()) * 1.15 - 31000) / 32000),
            "W130": "referral demand x1.30, imaging-weighted (primary/urgent care x1.10, the US/CT/MR-heavy sources x%.3f)" % ((sum(SRC.values()) * 1.30 - 31000 * 1.1) / 32000),
            "W150": "DIAGNOSTIC, outside the documented demand range: referral demand x1.50, imaging-weighted (primary/urgent care x1.20, the US/CT/MR-heavy sources x%.3f)" % ((sum(SRC.values()) * 1.50 - 31000 * 1.2) / 32000)}
    s = [bits[d]]
    names = {"XRCT": "XR/CT day technologist (08:00-16:30, $38/h)", "USfull": "US technologist 11:30-20:00 ($44/h)",
             "MRfull": "MR technologist 11:30-20:00 ($45/h)"}
    s += ["+ " + names[x] for x in staff]
    if pack:
        s.append("US_45 slots at 40 min (one more US_45 per weekday)")
    if alloc != "A60":
        s.append("radiologist site allocation %d%%/%d%%" % (ALLOC[alloc][0] // 100, ALLOC[alloc][1] // 100))
    return "V0.2 Step 0 feasibility sweep (static overlay): " + "; ".join(s) + "."

RUNS = []
# Stage 1: one lever at a time from the base.
RUNS += [("D100", (), False, "A60")]
RUNS += [(d, (), False, "A60") for d in ("D115", "D130", "W115", "W130")]
RUNS += [("D100", (s,), False, "A60") for s in ("XRCT", "USfull", "MRfull")]
RUNS += [("D100", (), True, "A60"), ("D100", (), False, "A55"), ("D100", (), False, "A50")]
# Stage 2: staffing packages x demand, with US packing and the 55/45 allocation.
PACKAGES = [("MRfull",), ("MRfull", "XRCT"), ("MRfull", "USfull"), ("MRfull", "USfull", "XRCT")]
for d in ("D100", "D115", "D130", "W115", "W130"):
    for k in PACKAGES:
        RUNS.append((d, k, True, "A55"))

# Stage 3 (after stage 2 showed the 55/45 allocation starving reading at
# +15-30% demand, and 40-minute US packing changing nothing): the staffing
# packages at the base 60/50 allocation without packing, and the two richest
# at 65/55.
for d in ("D115", "D130", "W130"):
    for k in PACKAGES:
        RUNS.append((d, k, False, "A60"))
for d in ("D130", "W130"):
    for k in (("MRfull",), ("MRfull", "USfull")):
        RUNS.append((d, k, False, "A65"))

# Stage 4: the base and the three best stage-3 candidates at 20 seeds
# (paired), and the
# diagnostic beyond the documented demand range.
CONFIRM = [("D100", ()), ("W130", ("MRfull",)), ("D130", ("MRfull",)), ("W130", ("MRfull", "USfull"))]
DIAGNOSTIC = [("W150", ("MRfull",)), ("W150", ("MRfull", "USfull"))]

def main():
    outdir = os.path.join(ROOT, "experiments")
    seeds = sys.argv[1] if len(sys.argv) > 1 else "1-9"
    for d, staff, pack, alloc in RUNS:
        write(outdir, d, staff, pack, alloc, seeds, "")
    for d, staff in CONFIRM:
        write(outdir, d, staff, False, "A60", "1-20", "-s20")
    for d, staff in DIAGNOSTIC:
        write(outdir, d, staff, False, "A60", seeds, "")
    print(len(RUNS) + len(CONFIRM) + len(DIAGNOSTIC), "manifests")

def write(outdir, d, staff, pack, alloc, seeds, suffix):
    n = name(d, staff, pack, alloc) + suffix
    m = {"name": n, "description": describe(d, staff, pack, alloc), "scenario": "established"}
    o = overlay(d, staff, pack, alloc)
    if o:
        m["overlay"] = o
    m["seeds"], m["days"] = seeds, 90
    with open(os.path.join(outdir, n + ".json"), "w") as f:
        json.dump(m, f, indent=1)
        f.write("\n")

if __name__ == "__main__":
    main()
