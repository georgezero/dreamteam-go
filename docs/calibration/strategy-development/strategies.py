#!/usr/bin/env python3
"""Writes Slice 5C.1's development strategy files (strategies/*.json,
dreamteam.strategy/v1) and strategy sets (sets/*.json,
dreamteam.strategyset/v1, each strategy pinned by SHA-256). Nothing here
depends on a seed or on a simulation output: every strategy is fixed
dated batches of supported player actions. Run from anywhere:

    python3 docs/calibration/strategy-development/strategies.py

Plans A-D are the frozen dynamic-staffing plans
(../dynamic-staffing/plans/*.json), their actions copied unchanged.
"""
import hashlib
import json
import os

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.normpath(os.path.join(HERE, "..", "..", ".."))
REL = os.path.relpath(HERE, ROOT)
HIST = os.path.join(HERE, "..", "dynamic-staffing", "plans")
SCENARIO = "docs/calibration/feasibility/step0c/experiments/W130-SHR-MRft-s20.json"
TAKEOVER = "2026-10-13T06:00:00"  # Tuesday
EARLIEST = "2026-10-27"           # 14 days' notice from the takeover


def historical(name, new_name=None):
    p = json.load(open(os.path.join(HIST, name + ".json")))
    return {"name": new_name or p["name"], "description": p["description"], "batches": p["batches"]}


def tele(request_id, windows, scope, effective=EARLIEST):
    """A standard-v1 contract start; open-ended (no cancellation)."""
    return {"type": "contract_teleradiology", "op": "start", "requestId": request_id, "plan": "standard-v1",
            "windows": windows, "scope": scope, "effective": effective}


def with_tele(base, name, description, contract):
    """base plus a contract in its takeover batch (a separate batch: a
    contract cannot share a batch with staffing actions)."""
    s = {"name": name, "description": description, "teleOnlyOf": base["name"],
         "batches": [b for b in base["batches"]] + [{"time": TAKEOVER, "actions": [contract]}]}
    return s


A = historical("A-no-action")
B = historical("B-partial")
C = historical("C-full-reorganization")
D = historical("D-full-reading")

ALL_WINDOWS = ["weeknights", "saturday", "sunday"]

# Phase 2 (fixed before any teleradiology run on the development seeds,
# from D's breach timing, results/p1-historical/breach-timing.md): every
# contract is standard-v1, accepted at the takeover, effective Tue Oct 27
# (the earliest, 14 days' notice), open-ended and not cancelled during the
# evaluation (its earliest end is 30 days after the activation; the
# retainer runs until a cancellation, which needs 30 days' notice).
E = with_tele(D, "E-urgent-tele",
              "Plan D plus an urgent-only standard-v1 contract on the weeknight and saturday windows (where D's reachable urgent "
              "breaches are), effective Tue 2026-10-27, open-ended. Retainer $2,300 a month plus urgent read fees.",
              tele("urgent-tele", ["weeknights", "saturday"], "urgent"))
F = with_tele(D, "F-broad-tele",
              "Plan D plus an all-priority standard-v1 contract on every window (weeknights, saturday, sunday), effective "
              "Tue 2026-10-27, open-ended. Retainer $3,100 a month plus read fees.",
              tele("broad-tele", ALL_WINDOWS, "all"))
G = with_tele(B, "G-partial-broad-tele",
              "Plan B (the lowest-cost staffing change) plus F's contract: all priorities on every window, effective "
              "Tue 2026-10-27, open-ended. Tests whether remote reading can carry service without the full reorganization and "
              "the reading allocation.",
              tele("broad-tele", ALL_WINDOWS, "all"))



def with_allocation(base, name, description, avery_bp, blake_bp, tele_only_of=None):
    """base with its two allocation changes set to other levels (same
    employees, same effective date); everything else unchanged."""
    import copy
    s = copy.deepcopy(base)
    s["name"], s["description"] = name, description
    if tele_only_of is None:
        s.pop("teleOnlyOf", None)
    for b in s["batches"]:
        for a in b["actions"]:
            if a["type"] == "set_reading_allocation":
                a["siteAllocationBP"] = {"E07": avery_bp, "E08": blake_bp}[a["employee"]]
    return s


# Phase 8 refinements (hypotheses in README, written before each run).
# R1: E with the allocation one step higher (65.0 / 55.0%).
E2 = with_allocation(E, "E2-urgent-tele-alloc65",
                     "Refinement R1 of E-urgent-tele: the same staffing and urgent-only weeknight and saturday contract, with Dr. "
                     "Avery at 65.0% and Dr. Blake at 55.0% (one 2.5-point step above D's 62.5/52.5%) from Tue 2026-10-27.",
                     6500, 5500)
# R2: D plus F's all-priority contract on the weeknight window only.
H = with_tele(D, "H-weeknight-tele",
              "Refinement R2 of F-broad-tele: Plan D plus an all-priority standard-v1 contract on the weeknight window only "
              "(Mon-Fri 18:00-08:00), effective Tue 2026-10-27, open-ended. Retainer $1,500 a month plus read fees.",
              tele("weeknight-tele", ["weeknights"], "all"))

# R3: E2 with its urgent contract on the weeknight window only.
E3 = with_tele(with_allocation(D, "x", "x", 6500, 5500), "E3-urgent-weeknights-alloc65",
               "Refinement R3 of E2-urgent-tele-alloc65: the same staffing and 65.0/55.0% allocation, with the urgent-only "
               "standard-v1 contract on the weeknight window only (Mon-Fri 18:00-08:00), effective Tue 2026-10-27, open-ended. "
               "Retainer $1,500 a month plus urgent read fees.",
               tele("urgent-tele", ["weeknights"], "urgent"))
# R4: E2 without its contract: D's staffing at 65.0/55.0%.
D2 = with_allocation(D, "D2-full-reading-alloc65",
                     "Refinement R4 of E2-urgent-tele-alloc65: the same staffing and 65.0/55.0% allocation from Tue 2026-10-27, "
                     "without a teleradiology contract.", 6500, 5500)
E3["teleOnlyOf"] = D2["name"]

STRATEGIES = {s["name"]: s for s in [A, B, C, D, E, F, G, E2, H, E3, D2]}

SETS = {
    # Phase 1: the historical plans, unchanged, under v0.2-eval-2.
    "p1-historical": ["A-no-action", "B-partial", "C-full-reorganization", "D-full-reading"],
    # Phase 2: the prespecified family.
    "p2-family": ["A-no-action", "B-partial", "C-full-reorganization", "D-full-reading", "E-urgent-tele", "F-broad-tele",
                  "G-partial-broad-tele"],
    # Phase 8, refinements R1 and R2, with their parents.
    "p3-refine": ["A-no-action", "D-full-reading", "E-urgent-tele", "F-broad-tele", "E2-urgent-tele-alloc65", "H-weeknight-tele"],
    # Phase 8, refinements R3 and R4, with their parent E2.
    "p4-refine": ["A-no-action", "E2-urgent-tele-alloc65", "E3-urgent-weeknights-alloc65", "D2-full-reading-alloc65"],
}


def write(path, obj):
    raw = (json.dumps(obj, indent=1, ensure_ascii=False) + "\n").encode()
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "wb") as f:
        f.write(raw)
    return hashlib.sha256(raw).hexdigest()


def main():
    pins = {}
    for name, s in STRATEGIES.items():
        obj = {"schema": "dreamteam.strategy/v1", "name": name, "description": s["description"]}
        if s.get("teleOnlyOf"):
            obj["teleOnlyOf"] = s["teleOnlyOf"]
        obj["batches"] = s["batches"]
        pins[name] = write(os.path.join(HERE, "strategies", name + ".json"), obj)
    for sid, names in SETS.items():
        write(os.path.join(HERE, "sets", sid + ".json"), {
            "schema": "dreamteam.strategyset/v1", "id": sid,
            "description": "Slice 5C.1 development set %s (development seeds only; not a frozen holdout set)" % sid,
            "scenarioManifest": SCENARIO, "baseline": "A-no-action",
            "plans": [{"name": n, "path": REL + "/strategies/" + n + ".json", "sha256": pins[n]} for n in names]})


if __name__ == "__main__":
    main()
