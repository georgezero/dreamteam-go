#!/usr/bin/env python3
"""Writes the dynamic staffing benchmark's fixed action plans (plans/*.json)
and benchmark.json. Every plan is a list of timed action batches, applied
identically on every seed; nothing here depends on a seed or on any
simulation output. Run from anywhere:

    python3 docs/calibration/dynamic-staffing/plans.py
"""
import json
import os

HERE = os.path.dirname(os.path.abspath(__file__))
TAKEOVER = "2026-10-13T06:00:00"  # Tuesday; rota week A starts Monday 2026-10-12
WEEKDAYS = ["Mon", "Tue", "Wed", "Thu", "Fri"]


def shift(start, end, room, breaks=(), covers=()):
    return {"start": start, "end": end, "breaks": [list(b) for b in breaks], "assignment": room,
            "covers": [{"room": r, "from": f, "until": u} for r, f, u in covers]}


def week(days):
    w = {d: None for d in ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]}
    w.update(days)
    return w


def roster(days):
    """The same pattern in rota weeks A and B."""
    return {"A": week(days), "B": week(days)}


def every(days, s):
    return {d: s for d in days}


# --- the roles of the Step 0C lead staffing (W130-DED-MRpt), as hires -----

# Dedicated CT day technologist ("Jamie"): CT 07:45-16:15, relieving the
# X-ray technologist's 11:30 break; 40 h.
CORE_CT = roster(every(WEEKDAYS, shift("07:45", "16:15", "R3", [("12:30", "13:00")], [("R1", "11:30", "12:00")])))

# Shared XR/CT evening (Sam's lead roster): CT 16:00-20:00, X-ray from
# 16:15, Monday to Friday, plus Saturday 10:00-18:00 on CT covering X-ray;
# 27.5 h.
EVENING_XRCT = roster({**every(WEEKDAYS, shift("16:00", "20:00", "R3", [], [("R1", "16:15", "20:00")])),
                       "Sat": shift("10:00", "18:00", "R3", [("13:30", "14:00")], [("R1", "10:00", "18:00")])})

# Part-time MR evening (Alex's lead roster): 16:00-20:00 Monday to Friday; 20 h.
EVENING_MR = roster(every(WEEKDAYS, shift("16:00", "20:00", "R4")))

# Riley (E03) on X-ray only, relieving the CT technologist's 12:30 break;
# 40 h of a 40 h contract.
RILEY_XR = roster(every(WEEKDAYS, shift("07:45", "16:15", "R1", [("11:30", "12:00")], [("R3", "12:30", "13:00")])))

# Step 0C's cross-trained float (W130-FLT): Monday CT 07:45-16:15, Tuesday
# to Friday X-ray 16:00-20:00; 24 h.
FLOAT = roster({"Mon": shift("07:45", "16:15", "R3", [("12:30", "13:00")]),
                **every(WEEKDAYS[1:], shift("16:00", "20:00", "R1"))})


def hire(req, skills, rate, minutes, start, r):
    return {"type": "hire", "requestId": req, "role": "technologist", "skills": skills, "rateCents": rate,
            "contractedWeeklyMinutes": minutes, "startDate": start, "roster": r}


def terminate(emp, last):
    return {"type": "terminate", "employee": emp, "lastDay": last}


def set_roster(emp, effective, r):
    return {"type": "set_roster", "op": "set", "employee": emp, "effective": effective, "roster": r}


def set_allocation(emp, effective, bp):
    return {"type": "set_reading_allocation", "op": "set", "employee": emp, "effective": effective, "siteAllocationBP": bp}


# Standard offers (no premium): XR+CT $38, MR $45, 28-day recruiting lead,
# so the earliest start from the takeover is Tuesday 2026-11-10.
START = "2026-11-10"
H_CORE = hire("core-ct", ["XR", "CT"], 3800, 2400, START, CORE_CT)
H_EVE = hire("evening-xrct", ["XR", "CT"], 3800, 1650, START, EVENING_XRCT)
H_MR = hire("evening-mr", ["MR"], 4500, 1200, START, EVENING_MR)
H_FLOAT = hire("float-xrct", ["XR", "CT"], 3800, 1440, START, FLOAT)

# Training is 40 attended hours on the hire's own roster. From Tuesday
# 2026-11-10: the CT core works 8 h a day (done Monday 11-16, productive
# Tuesday 11-17); the XR/CT evening 4 h weekdays and 7.5 h Saturday (done
# Friday 11-20, productive Saturday 11-21); the MR evening 4 h weekdays
# (done Monday 11-23, productive Tuesday 11-24); the float 8 h Monday and
# 4 h Tuesday to Friday (done Friday 11-20, productive Saturday 11-21).
#
# Departures: each leaver works until the replacement is productive (last
# day the day before), so coverage never lapses and nothing is displaced.
# Sam (E04, Tue-Sat) last works Friday 11-20; the evening hire is
# productive Saturday 11-21, and the CT core covers Sam's midday CT from
# 11-17. Alex (E11) last works Monday 11-23. Riley's X-ray-only roster
# takes effect on the first Monday after the CT core is productive
# (11-23; a Monday at least 14 days ahead).

PLANS = [
    {"name": "A-no-action",
     "description": "The unchanged sustainable starting practice (shared XR/CT staffing, full-time MR evening); no action.",
     "batches": [], "expect": []},
    {"name": "B1-mr-evening",
     "description": "Partial candidate: convert the full-time MR evening (Alex, 40 h) to a part-time 16:00-20:00 MR hire (20 h); one termination, one hire.",
     "batches": [{"time": TAKEOVER, "actions": [H_MR, terminate("E11", "2026-11-23")]}],
     "expect": [{"employee": "evening-mr", "start": START, "productive": "2026-11-24", "replaces": "E11"},
                {"employee": "E11", "lastDay": "2026-11-23"}]},
    {"name": "B2-float",
     "description": "Partial candidate: hire Step 0C's cross-trained XR/CT float (Monday CT day, Tuesday-Friday X-ray evening; 24 h); no termination, no other change.",
     "batches": [{"time": TAKEOVER, "actions": [H_FLOAT]}],
     "expect": [{"employee": "float-xrct", "start": START, "productive": "2026-11-21"}]},
    {"name": "B3-float-mr-evening",
     "description": "Partial candidate: B1 and B2 together (two hires, one termination).",
     "batches": [{"time": TAKEOVER, "actions": [H_FLOAT, H_MR, terminate("E11", "2026-11-23")]}],
     "expect": [{"employee": "float-xrct", "start": START, "productive": "2026-11-21"},
                {"employee": "evening-mr", "start": START, "productive": "2026-11-24", "replaces": "E11"},
                {"employee": "E11", "lastDay": "2026-11-23"}]},
    {"name": "C-full-reorganization",
     "description": "The closest feasible path to Step 0C's lead staffing (W130-DED-MRpt) without reading actions: hire a dedicated CT day technologist, a 27.5 h XR/CT evening technologist and a 20 h MR evening technologist; terminate Sam and Alex once their replacements are productive; move Riley to X-ray only. One atomic batch at the takeover, standard offers.",
     "batches": [{"time": TAKEOVER, "actions": [H_CORE, H_EVE, H_MR,
                                                terminate("E04", "2026-11-20"), terminate("E11", "2026-11-23"),
                                                set_roster("E03", "2026-11-23", RILEY_XR)]}],
     "expect": [{"employee": "core-ct", "start": START, "productive": "2026-11-17"},
                {"employee": "evening-xrct", "start": START, "productive": "2026-11-21", "replaces": "E04"},
                {"employee": "evening-mr", "start": START, "productive": "2026-11-24", "replaces": "E11"},
                {"employee": "E04", "lastDay": "2026-11-20"},
                {"employee": "E11", "lastDay": "2026-11-23"},
                {"employee": "E03", "rosterEffective": ["2026-11-23"]}]},
]

# Plan D (Slice 4, fixed 2026-10-16 before any run of it; V0.2_DESIGN
# §5.10): plan C's six staffing actions, unchanged and in the same takeover
# batch, followed by the Step 0C lead's reading allocation (A62): Dr. Avery
# (E07) 60% -> 62.5%, Dr. Blake (E08) 50% -> 52.5%, at the earliest legal
# effective date. An allocation change needs 14 days' notice on any
# weekday: from the takeover (Tuesday 2026-10-13) that is Tuesday
# 2026-10-27. D is never adjusted on evaluation results; if it does not
# restore acceptable reporting, other allocations belong to a separate
# development-seed experiment, not to these held-out seeds.
ALLOC_FROM = "2026-10-27"
_c = next(p for p in PLANS if p["name"] == "C-full-reorganization")
PLANS.append({"name": "D-full-reading",
              "description": "Plan C's staffing actions and timing, unchanged, plus the Step 0C lead's reading allocation: Dr. Avery (E07) 60% -> 62.5% and Dr. Blake (E08) 50% -> 52.5% from the earliest legal date, Tuesday 2026-10-27. One atomic batch at the takeover.",
              "batches": [{"time": TAKEOVER, "actions": _c["batches"][0]["actions"] + [set_allocation("E07", ALLOC_FROM, 6250),
                                                                                      set_allocation("E08", ALLOC_FROM, 5250)]}],
              "expect": _c["expect"] + [{"employee": "E07", "allocation": [{"effective": ALLOC_FROM, "siteBP": 6250}]},
                                        {"employee": "E08", "allocation": [{"effective": ALLOC_FROM, "siteBP": 5250}]}]})

# Choice of the partial plan, fixed before any run: B-partial is the B
# candidate with the highest mean T180 cumulative operating profit over the
# paired no-action run on the development seeds, among candidates whose
# urgent-report breach rate over days 91-180 is at most 5 points above the
# no-action run's (mean over the same seeds). A tie (within $1k) goes to
# the plan with fewer actions. The evaluation seeds are never used to
# choose or adjust any plan.

# Applied on the development seeds (results/dev/report.md): B1 (+$13.0k at
# T180, urgent breach rate +0.6 points) is selected; B2 and B3 earn less or
# more but raise the urgent breach rate by 12 points, beyond the limit.
SELECTED = "B1-mr-evening"
_sel = next(p for p in PLANS if p["name"] == SELECTED)
PLANS.append({**_sel, "name": "B-partial",
              "description": "Partial reorganization (selected on the development seeds from B1-B3): " + _sel["description"].split(": ", 1)[1]})

BENCHMARK = {
    "scenario": "../feasibility/step0c/experiments/W130-SHR-MRft-s20.json",
    "days": 180,
    "endpoints": [30, 90, 180],
    "sets": {
        # Development: plan validation and the choice of the partial plan.
        "dev": {"seeds": "901-910",
                "plans": ["A-no-action", "B1-mr-evening", "B2-float", "B3-float-mr-evening", "C-full-reorganization"]},
        # Held out: never run before the plans were frozen.
        "eval": {"seeds": "1001-1020", "plans": ["A-no-action", "B-partial", "C-full-reorganization"]},
        # Slice 4 (0.2.0-dev.5): plan D validated on the development seeds
        # (acceptance and lifecycle only; nothing is chosen), then A-D on the
        # same held-out seeds. Results go to their own directories; the
        # original dev and eval results above are kept as they were.
        "reading-dev": {"seeds": "901-910", "plans": ["A-no-action", "C-full-reorganization", "D-full-reading"]},
        "reading-eval": {"seeds": "1001-1020", "plans": ["A-no-action", "B-partial", "C-full-reorganization", "D-full-reading"]},
    },
}


def main():
    for p in PLANS:
        with open(os.path.join(HERE, "plans", p["name"] + ".json"), "w") as f:
            json.dump(p, f, indent=1)
            f.write("\n")
    with open(os.path.join(HERE, "benchmark.json"), "w") as f:
        json.dump(BENCHMARK, f, indent=1)
        f.write("\n")


if __name__ == "__main__":
    main()
