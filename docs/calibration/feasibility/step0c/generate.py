#!/usr/bin/env python3
"""Generates the V0.2 Step 0C manifests: alternative technologist staffing
architectures for the sustainable scenario, static overlays on the
established practice. Run from the repository root:

    python3 docs/calibration/feasibility/step0c/generate.py

then run them with the calibration harness:

    DT_EXP_DIR=../../docs/calibration/feasibility/step0c/experiments \\
    DT_RES_DIR=../../docs/calibration/feasibility/step0c/results \\
    go test -tags calibration ./internal/sim -run 'CalibrationExperiments$' -v -parallel 4 -timeout 600m

Every candidate starts from the best Step 0B configuration (imaging-weighted
referral demand, 12-minute XR slots, evening MR coverage) and changes only
who works when: technologist rosters, new technologists at the documented
pay rates, and (Phase 4) the radiologists' salaried site allocation. Rooms,
equipment, qualifications, opening hours, templates other than Step 0B's XR
grid, exam and read durations, reimbursement, payers, patient behaviour,
wage and benefit rates and fixed expenses stay those of the base scenario.

Employment cost: hourly staff are paid for every rostered minute except
their unpaid break, plus any run-over, at their rate (x1.5 past 40 h a
week), plus benefits and payroll tax at the scenario's 25% of wages: the
same rules as every existing employee. A technologist's contracted weekly
hours are the paid hours of the roster written here (the static payroll
obligation; the contract lifecycle is V0.2 action work). Salaried
radiologists cost annual salary x site allocation x 1.25.
"""
import json, os, sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, ".."))
import generate as s0  # noqa: E402  (Step 0 levers)
import importlib.util  # noqa: E402
_spec = importlib.util.spec_from_file_location("s0b", os.path.join(HERE, "..", "step0b", "generate.py"))
s0b = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(s0b)

WEEKDAYS = s0.WEEKDAYS
BASE_EMP = {e["id"]: e for e in s0.BASE["employees"]}


def sh(start, end, room, brk=None, covers=()):
    """A full shift object: every key written, so the merge replaces the
    base shift instead of mixing with it (null deletes a base key)."""
    return {"start": start, "end": end, "assignment": room, "breaks": [brk] if brk else None,
            "covers": [dict(c) for c in covers] or None}


def cover(room, frm=None, until=None):
    c = {"room": room}
    if frm:
        c["from"] = frm
    if until:
        c["until"] = until
    return c


def tech(eid, name, skills, rate, roster):
    return {"id": eid, "name": name, "role": "technologist", "skills": skills,
            "pay": {"type": "hourly", "rateCents": rate}, "roster": roster}


def days(shift, ds=WEEKDAYS):
    return {d: dict(shift) for d in ds}


# --- MR evening coverage ------------------------------------------------------
# Casey (E05) works 08:00-16:30; the MR template runs to 20:00.
MR = {
    # Step 0's evening MR technologist, a full 11:30-20:00 shift (the
    # 11:30-16:30 overlap with Casey is a second technologist in one room).
    "MRft": [tech("E11", "Alex", ["MR"], 4500, days(sh("11:30", "20:00", "R4", ["15:30", "16:00"])))],
    # Part-time: 16:00-20:00, the uncovered evening plus a 30-minute handover.
    "MRpt": [tech("E11", "Alex", ["MR"], 4500, days(sh("16:00", "20:00", "R4")))],
}

# --- XR/CT technologist architectures ------------------------------------------
# Base (struggling practice): Riley (E03, XR/CT, $36) 07:45-16:15 Mon-Fri on
# X-ray, covering CT all Monday and Tue-Fri until 11:30, break 11:30-12:00;
# Sam (E04, CT/XR, $40) 11:30-20:00 Tue-Fri on CT, covering X-ray from 16:15,
# break 15:30-16:00, and Saturday 10:00-18:00 both rooms. Nobody staffs X-ray
# or CT on Monday after 16:15.
RILEY_XR = sh("07:45", "16:15", "R1", ["11:30", "12:00"], [cover("R3", "12:30", "13:00")])
JAMIE_CT = sh("07:45", "16:15", "R3", ["12:30", "13:00"], [cover("R1", "11:30", "12:00")])
SAM_EVE = sh("16:00", "20:00", "R3", None, [cover("R1", "16:15")])
SAM_SAT = BASE_EMP["E04"]["roster"]["Sat"]


def riley(roster):
    return {"id": "E03", "roster": roster}


def sam(roster):
    return {"id": "E04", "roster": roster}


ARCH = {
    # A. Control: the base shared coverage.
    "SHR": [],
    # B. Dedicated core: Riley on X-ray only and a new CT technologist on CT
    # only, 07:45-16:15 Mon-Fri, each relieving the other's break; Sam
    # becomes the shared evening technologist, 16:00-20:00 Mon-Fri (Monday
    # evening newly staffed), Saturday unchanged.
    "DED": [riley(days(RILEY_XR)), tech("E09", "Jamie", ["CT", "XR"], 3800, days(JAMIE_CT)),
            sam({**days(SAM_EVE), "Sat": SAM_SAT})],
    # B, full-time evening: the same core with Sam's base full shift
    # (11:30-20:00 Tue-Fri on CT, sharing X-ray from 16:15).
    "DEDft": [riley(days(RILEY_XR)), tech("E09", "Jamie", ["CT", "XR"], 3800, days(JAMIE_CT))],
    # C. Dedicated core plus a part-time evening X-ray technologist,
    # 16:00-20:00 Mon-Fri: evenings dedicated too (Sam on CT only).
    "DPT": [riley(days(RILEY_XR)), tech("E09", "Jamie", ["CT", "XR"], 3800, days(JAMIE_CT)),
            sam({**days(sh("16:00", "20:00", "R3")), "Sat": SAM_SAT}),
            tech("E10", "Robin", ["XR", "CT"], 3800, days(sh("16:00", "20:00", "R1")))],
    # C, full-time pair: Sam's full shift and a full 11:30-20:00 evening
    # X-ray technologist (Step 0B's XRCTeve) instead of the two 4-hour shifts.
    "DPTft": [riley(days(RILEY_XR)), tech("E09", "Jamie", ["CT", "XR"], 3800, days(JAMIE_CT)),
              sam({d: {**BASE_EMP["E04"]["roster"][d], "covers": None} for d in WEEKDAYS[1:]}),
              tech("E10", "Robin", ["XR", "CT"], 3800, days(sh("11:30", "20:00", "R1", ["15:30", "16:00"])))],
}
# D. Float: the base shared structure plus a cross-trained technologist in
# the measured shared-conflict periods. At the control (W130-SHR-MRft, 9
# seeds, 90 days) 62% of CT shared-technologist losses fall on Monday, when
# Riley staffs both rooms all day (Monday 12:00-16:00 alone: the contrast
# block), and the Tue-Fri conflicts fall in the evening (CT 16:00-18:00, XR
# 17:00-20:00), when Sam staffs both; Tue-Fri mornings are minor. The float
# works Monday 07:45-16:15 on CT and Tue-Fri 16:00-20:00 on X-ray: 24 paid
# hours a week, part-time.
FLOAT_MON = sh("07:45", "16:15", "R3", ["12:30", "13:00"])
ARCH["FLT"] = [tech("E09", "Jamie", ["CT", "XR"], 3800,
                    {"Mon": FLOAT_MON, **days(sh("16:00", "20:00", "R1"), WEEKDAYS[1:])})]
# E. Its full-time pair: the same conflict periods inside a 40-hour week
# (Monday as above, Tue-Fri 11:30-20:00 on X-ray).
ARCH["FLTft"] = [tech("E09", "Jamie", ["CT", "XR"], 3800,
                      {"Mon": FLOAT_MON, **days(sh("11:30", "20:00", "R1", ["15:30", "16:00"]), WEEKDAYS[1:])})]


ALLOC = {"A60": None, "A62": (6250, 5250), "A65": (6500, 5500), "A70": (7000, 6000)}
DEMAND = {"W115": "W115", "W130": "W130", "W150": "W150"}


def overlay(dem, arch, mr, alloc="A60"):
    o = {"referralSources": s0.DEMAND[dem], "templates": {"R1": s0b.TEMPLATES["XR12"]}}
    emps = list(ARCH[arch]) + list(MR[mr])
    if ALLOC[alloc]:
        emps += [{"id": "E07", "siteAllocationBP": ALLOC[alloc][0]}, {"id": "E08", "siteAllocationBP": ALLOC[alloc][1]}]
    o["employees"] = json.loads(json.dumps(emps))
    return o


def name(dem, arch, mr, alloc="A60"):
    return "-".join([dem, arch, mr] + ([alloc] if alloc != "A60" else []))


DESC = {"SHR": "shared XR/CT coverage (base)", "DED": "dedicated XR and CT day core, shared 16:00-20:00 evening",
        "DEDft": "dedicated XR and CT day core, shared evening on Sam's full shift",
        "DPT": "dedicated day core plus part-time 16:00-20:00 evening X-ray and CT",
        "DPTft": "dedicated day core plus full-time evening X-ray (11:30-20:00)",
        "FLT": "shared coverage plus part-time CT float in the measured conflict hours",
        "FLTft": "shared coverage plus full-time CT float 07:45-16:15",
        "MRft": "MR evening 11:30-20:00", "MRpt": "MR evening part-time 16:00-20:00",
        "W115": "referral demand x1.15 imaging-weighted", "W130": "referral demand x1.30 imaging-weighted",
        "W150": "DIAGNOSTIC, outside the documented range: referral demand x1.50 imaging-weighted"}


def manifest(dem, arch, mr, alloc, seeds, suffix=""):
    n = name(dem, arch, mr, alloc) + suffix
    d = "; ".join([DESC[dem], DESC[arch], DESC[mr], "12-minute XR slots"])
    if alloc != "A60":
        d += "; radiologist site allocation %g%%/%g%%" % (ALLOC[alloc][0] / 100, ALLOC[alloc][1] / 100)
    m = {"name": n, "description": "V0.2 Step 0C: " + d + ".", "scenario": "established",
         "overlay": overlay(dem, arch, mr, alloc), "seeds": seeds, "days": 90}
    os.makedirs(os.path.join(HERE, "experiments"), exist_ok=True)
    with open(os.path.join(HERE, "experiments", n + ".json"), "w") as f:
        json.dump(m, f, indent=1)
        f.write("\n")
    return n


def merge(b, o):
    """The scenario system's merge (internal/scenario Merge), for rosters."""
    if isinstance(o, dict) and isinstance(b, dict):
        out = dict(b)
        for k, v in o.items():
            if v is None:
                out.pop(k, None)
            else:
                out[k] = merge(out[k], v) if k in out else v
        return out
    return o


def mins_of(t):
    h, m = t.split(":")
    return int(h) * 60 + int(m)


def contracts(arch, mr):
    """Technologists' weekly paid hours (A/B-week days count half) and
    monthly employment cost, wages x 1.25 x 52/12, for an architecture."""
    emps = {e["id"]: e for e in json.loads(json.dumps(s0.BASE["employees"]))}
    for e in ARCH[arch] + MR[mr]:
        emps[e["id"]] = merge(emps[e["id"]], e) if e["id"] in emps else e
    rows = []
    for e in emps.values():
        if e["role"] != "technologist":
            continue
        h = 0.0
        for d, x in e["roster"].items():
            if not x:
                continue
            m = mins_of(x["end"]) - mins_of(x["start"]) - sum(mins_of(b) - mins_of(a) for a, b in (x.get("breaks") or []))
            h += m / 60 * (0.5 if x.get("weeks") else 1)
        rate = e["pay"]["rateCents"] / 100
        reg, ot = min(h, 40), max(h - 40, 0)
        cost = (reg + 1.5 * ot) * rate * 1.25 * 52 / 12
        rows.append((e["id"], e["name"], h, rate, cost))
    return rows


def contract_table():
    print("| Architecture | Technologists (paid h/week) | XR/CT tech h/week | Technologist cost $/month |")
    print("|---|---|---|---|")
    for mr in MR:
        for arch in ARCH:
            rows = contracts(arch, mr)
            xrct = sum(h for i, n, h, r, c in rows if i in ("E03", "E04", "E09", "E10"))
            print("| %s-%s | %s | %.1f | %.0f |" % (arch, mr, ", ".join("%s %.1f" % (n, h) for i, n, h, r, c in rows),
                                                   xrct, sum(c for *_, c in rows)))


def main():
    stage = sys.argv[1] if len(sys.argv) > 1 else "all"
    if stage == "contracts":
        contract_table()
        return
    out = []
    if stage in ("conflict", "all"):
        # Phase 2a: where the shared-technologist conflicts fall, at the
        # Step 0B best configuration (the control).
        out.append(manifest("W130", "SHR", "MRft", "A60", "1-9"))
    if stage in ("grid", "all"):
        # Phase 3: every architecture x MR evening coverage, at the primary
        # demand (x1.30), the lower sensitivity (x1.15) and the out-of-range
        # diagnostic (x1.50); nine paired seeds.
        for dem in ("W130", "W115", "W150"):
            for arch in ARCH:
                for mr in MR:
                    out.append(manifest(dem, arch, mr, "A60", "1-9"))
    if stage in ("reading", "all"):
        # Phase 4: the salaried allocation needed to restore reading at the
        # leading candidates (x1.30 demand).
        for arch in ("SHR", "DED", "DPT", "FLT"):
            for alloc in ("A62", "A65", "A70"):
                out.append(manifest("W130", arch, "MRpt", alloc, "1-9"))
        # The reading-restored candidates at the demand sensitivities.
        for dem in ("W115", "W150"):
            for arch in ("SHR", "DED", "DPT", "FLT"):
                out.append(manifest(dem, arch, "MRpt", "A65", "1-9"))
            for arch in ("DED", "FLT"):
                out.append(manifest(dem, arch, "MRpt", "A62", "1-9"))
    if stage in ("confirm", "all"):
        # Phase 3/5: 20 paired seeds for the control, the part-time-MR
        # control, and the leading candidates with and without restored
        # reading.
        for arch, mr, alloc in (("SHR", "MRft", "A60"), ("SHR", "MRpt", "A60"), ("DED", "MRpt", "A60"),
                                ("FLT", "MRpt", "A60"), ("DED", "MRpt", "A62"), ("FLT", "MRpt", "A62"),
                                ("DED", "MRpt", "A65"), ("FLT", "MRpt", "A65")):
            out.append(manifest("W130", arch, mr, alloc, "1-20", "-s20"))
    print(len(out), "manifests")


if __name__ == "__main__":
    main()
