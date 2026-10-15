#!/usr/bin/env python3
"""Generates the V0.2 Step 0B manifests: CT/XR booking-loss diagnosis and
CT/XR template variants, static overlays on the established practice.
Run from the repository root:

    python3 docs/calibration/feasibility/step0b/generate.py

then run them with the calibration harness:

    DT_EXP_DIR=../../docs/calibration/feasibility/step0b/experiments \\
    DT_RES_DIR=../../docs/calibration/feasibility/step0b/results \\
    go test -tags calibration ./internal/sim -run 'CalibrationExperiments$' -v -parallel 4 -timeout 600m

Step 0's levers (demand weighting, the evening MR technologist) come from
../generate.py; the CT/XR templates are defined here. Exam durations, rooms,
qualifications, pay and expense rates, reading, reimbursement, patient
behaviour, demand and opening hours stay those of the base scenario.
"""
import json, os, sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
import generate as s0  # noqa: E402

ROOT = os.path.dirname(os.path.abspath(__file__))
BASE = s0.BASE

# Phase 1: diagnosis of the base, demand x1.30 and the Step 0 best candidate.
DIAG = [("D100", ()), ("D130", ()), ("W130", ("MRfull",))]

# Phase 2: CT/XR template variants. Times are minutes from midnight.
def hm(m): return "%02d:%02d" % (m // 60, m % 60)
def mins(t): h, m = t.split(":"); return int(h) * 60 + int(m)
SAT_R1 = BASE["templates"]["R1"][1]  # Saturday XR, unchanged by every variant

def xr_slots(blocks):
    """XR_15-category slots from (from, to, step, length) blocks."""
    out = []
    for lo, hi, step, ln in blocks:
        t = mins(lo)
        while t + ln <= mins(hi):
            out.append({"start": hm(t), "lenMin": ln, "category": "XR_15"})
            t += step
    return out

WEEKDAYS = ["Mon", "Tue", "Wed", "Thu", "Fri"]
# The shared-technologist hours on weekdays (base roster): Riley staffs X-ray
# and CT alone all Monday and Tuesday-Friday until 11:30; Sam staffs both
# from 16:15 Tuesday-Friday.
TEMPLATES = {
    # 12-minute XR slots, 08:00-20:00 weekdays (60 instead of 48). XR exams
    # occupy the technologist p50 10-13 min (room 8-11 + 2-2.5 extra).
    "XR12": [{"days": WEEKDAYS, "slots": xr_slots([("08:00", "20:00", 12, 12)])}, SAT_R1],
    # XR slots every 30 minutes while one technologist staffs both rooms,
    # every 15 otherwise: the shared technologist's time is left to CT.
    "XRhalf": [
        {"days": ["Mon"], "slots": xr_slots([("08:00", "16:15", 30, 15)])},
        {"days": WEEKDAYS[1:], "slots": xr_slots([("08:00", "11:30", 30, 15), ("11:30", "16:15", 15, 15), ("16:30", "20:00", 30, 15)])},
        SAT_R1],
    # Control: the base XR grid offset by five minutes (staggered against CT).
    "STG": [{"days": WEEKDAYS, "slots": xr_slots([("08:05", "20:00", 15, 15)])}, SAT_R1],
}

def with_r1(overlay, variant):
    o = json.loads(json.dumps(overlay or {}))
    if variant:
        o.setdefault("templates", {})["R1"] = TEMPLATES[variant]
    return o

# An XR/CT technologist on X-ray 11:30-20:00 weekdays ($38/h, the Step 0
# XR/CT rate): unshares the evenings, when Sam staffs both rooms alone.
s0.STAFF["XRCTeve"] = s0.tech("E09", "Jamie", ["XR", "CT"], 3800, s0.shift("11:30", "20:00", "R1", ["15:30", "16:00"]))

VARIANTS = [  # (template, extra staff) on the Step 0 best candidate
    ("XR12", ()), ("XRhalf", ()), ("STG", ()),
    (None, ("XRCT",)), ("XR12", ("XRCT",)),
    (None, ("XRCTeve",)), ("XR12", ("XRCTeve",)),
]
DESC = {"XR12": "XR slots 12 min (60 a weekday)", "XRhalf": "XR slots halved in shared-technologist hours",
        "STG": "XR grid offset 5 min (control)"}

def manifest(name, desc, overlay, seeds):
    m = {"name": name, "description": "V0.2 Step 0B: " + desc, "scenario": "established"}
    if overlay:
        m["overlay"] = overlay
    m["seeds"], m["days"] = seeds, 90
    with open(os.path.join(ROOT, "experiments", name + ".json"), "w") as f:
        json.dump(m, f, indent=1)
        f.write("\n")

def main():
    n = 0
    for d, staff in DIAG:
        name = "diag-" + s0.name(d, staff, False, "A60")
        manifest(name, "booking-loss diagnosis; " + s0.describe(d, staff, False, "A60").split(": ", 1)[1],
                 s0.overlay(d, staff, False, "A60"), "1-9")
        n += 1
    for tmpl, extra in VARIANTS:
        staff = ("MRfull",) + extra
        parts = ["W130", "MRfull"] + list(extra) + ([tmpl] if tmpl else [])
        desc = s0.describe("W130", staff, False, "A60").split(": ", 1)[1].rstrip(".")
        if tmpl:
            desc += "; " + DESC[tmpl]
        manifest("-".join(parts), desc + ".", with_r1(s0.overlay("W130", staff, False, "A60"), tmpl), "1-9")
        n += 1
    # The evening XR/CT technologist moves the bottleneck to reading: the
    # same with the 65/55 allocation.
    staff = ("MRfull", "XRCTeve")
    manifest("W130-MRfull-XRCTeve-A65", s0.describe("W130", staff, False, "A65").split(": ", 1)[1],
             s0.overlay("W130", staff, False, "A65"), "1-9")
    n += 1
    # Phase 3: 20-seed confirmation of the best variant against the Step 0
    # best candidate, paired.
    for tmpl in (None, "XR12"):
        staff = ("MRfull",)
        desc = s0.describe("W130", staff, False, "A60").split(": ", 1)[1].rstrip(".")
        if tmpl:
            desc += "; " + DESC[tmpl]
        manifest("-".join(["W130", "MRfull"] + ([tmpl] if tmpl else [])) + "-s20", desc + ".",
                 with_r1(s0.overlay("W130", staff, False, "A60"), tmpl), "1-20")
        n += 1
    print(n, "manifests")

if __name__ == "__main__":
    main()
