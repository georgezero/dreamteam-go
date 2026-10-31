"""Tests of dreamteam.financial-evidence/v1: affordability, reversibility and
direct operational exposure on synthetic decision points, both recording
formats, stable identifiers and order, and the engine replay against a
stand-in engine (identical answers required, sources left unchanged). No
model is contacted and no simulator is run.

    python3 -I -B -m unittest discover -s analysis/evidence
"""

import hashlib
import json
import os
import random
import stat
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import financial_evidence as F  # noqa: E402

V3 = F.V3_SCHEMA
T = "2026-11-05T07:00:00"


def obs(cash=10_000_000, liabilities=6_000_000, reserved=0, limit=5_000_000, drawn=0, staff=None, policies=None,
        solvency=True):
    fin = {"creditLineModeled": solvency}
    if solvency:
        avail = limit - drawn
        fin["solvency"] = {"creditLimitCents": limit, "creditDrawnCents": drawn, "creditAvailableCents": avail,
                           "currentLiabilitiesCents": liabilities, "reservedCents": reserved,
                           "fundingHeadroomCents": cash + avail - liabilities - reserved}
    return {"time": T, "finance": {"cashCents": cash, "arCents": 30_000_000, "financing": fin},
            "staff": staff or [{"id": "E03", "role": "technologist", "employment": "active"},
                               {"id": "E07", "role": "radiologist", "employment": "active"}],
            "policies": policies or {"walkIn": "accept_if_gap", "worklist": "urgent_then_fifo"},
            "worklist": {"unread": 4}, "demand": {"modalities": [{"sinceStart": {"lost": 3}}]},
            "flow": {"sinceStart": {"examsDone": 100}}}


def headroom(o):
    return o["finance"]["financing"]["solvency"]["fundingHeadroomCents"]


def ok(a, code="accepted", msg="", eff=None):
    r = {"action": a, "ok": code in ("accepted", "unchanged"), "code": code, "message": msg}
    if eff:
        r["effectiveAt"] = eff
    return r


HIRE = {"type": "hire", "requestId": "h1", "role": "technologist", "skills": ["CT"], "rateCents": 4400,
        "contractedWeeklyMinutes": 2400, "startDate": "2026-11-28"}
ALLOC = {"type": "set_reading_allocation", "employee": "E07", "op": "set", "effective": "2026-11-19",
         "siteAllocationBP": 8000}
TERM = {"type": "terminate", "employee": "E03", "lastDay": "2026-11-19"}
ROSTER = {"type": "set_roster", "employee": "E03", "op": "set", "effective": "2026-11-23", "roster": {}}
TELE = {"type": "contract_teleradiology", "op": "start", "requestId": "t1", "plan": "standard-v1",
        "windows": ["weeknights"], "scope": "all", "effective": "2026-11-19"}
CANCEL = {"type": "contract_teleradiology", "op": "cancel", "contract": "T00001", "endDate": "2026-12-20"}
EXT = {"type": "extend_shift", "employee": "E03", "minutes": 60, "date": "2026-11-05"}
WORK = {"type": "set_worklist_policy", "policy": "oldest_first"}

LINES = {
    "fee": {"label": "Recruiting fee · CT technologist", "cents": 300_000, "when": "recognized now, paid 2026-12-01",
            "basis": "fixed offer table; not refundable"},
    "week": {"label": "Contract guarantee per full week · CT technologist", "cents": 220_000, "when": "from 2026-11-28",
             "basis": "40 h × $44.00 with benefits"},
    "exit": {"label": "Earliest lawful exit · CT technologist", "cents": 911_413, "when": "last day 2026-12-12",
             "basis": "contract guarantee from the start to the earliest last day, plus severance"},
    "alloc": {"label": "Salary allocation change · Dr. Avery (E07)", "cents": 986_302,
              "when": "per 30 days from 2026-11-19 until changed (14 days' notice)", "basis": "salary × 80% (was 60%)"},
    "sev": {"label": "Severance · Riley (E03)", "cents": 360_000, "when": "recognized now, paid 2026-11-27",
            "basis": "two contracted weeks"},
    "roster": {"label": "Contract guarantee per full week · Riley (E03)", "cents": 180_000,
               "when": "unchanged by the roster", "basis": "the contract"},
}


def preview(commitment, head, lines=(), **items):
    pv = {"ok": True, "guaranteed": [LINES[k] for k in lines],
          "funding": {"commitmentCents": commitment, "headroomCents": head, "funded": commitment <= head}}
    pv.update(items)
    return pv


def ctx(events, k, fmt="llm-agent 3", attempt="4.1", idx=None):
    return {"recording": {"path": "rec.jsonl", "group": "g", "sha256": "x", "format": fmt},
            "run": {"scenario": "s", "seed": 1}, "point": {"index": 4, "point": 4, "time": T}, "format": fmt,
            "events": events, "eventIndex": k, "catalog": {"types": []},
            "attempt": {"id": attempt, "recorded": True}, "attemptIdx": idx}


def afford(ev, o, events=None, **kw):
    an = F.Analysis()
    events = events or [ev]
    F.affordability(an, ctx(events, events.index(ev), **kw), 1, ev, o, "econ")
    return an.records[0]


def execute(apply_ev, val_ev, o, run_end="2026-12-13T07:00:00"):
    an = F.Analysis()
    events = [e for e in (val_ev, apply_ev) if e is not None]
    F.executed(an, ctx(events, events.index(val_ev or apply_ev)), 1, apply_ev, val_ev, o, run_end)
    return an.records


class Affordability(unittest.TestCase):
    def test_affordable_commitment(self):
        o = obs()
        ev = {"op": "validate", "actions": [HIRE], "answer": {"valid": True, "results": [ok(HIRE)]},
              "preview": preview(1_211_413, headroom(o), ("fee", "week", "exit"))}
        r = afford(ev, o)
        self.assertEqual(r["status"], "funded_without_credit")
        self.assertEqual(r["measurements"]["engineFunding"]["commitmentCents"], 1_211_413)
        self.assertEqual(r["execution"], "not_applied")

    def test_insufficient_funds(self):
        o = obs(cash=2_000_000)
        res = [ok(HIRE, "insufficient_funds", "Not funded: ...")]
        ev = {"op": "validate", "actions": [HIRE], "answer": {"valid": False, "results": res},
              "preview": preview(1_211_413, headroom(o), ("fee", "exit"))}
        r = afford(ev, o)
        self.assertEqual(r["status"], "refused_insufficient_funds")
        self.assertEqual(r["execution"], "refused")
        self.assertIn("Not funded", r["measurements"]["refusalMessage"])

    def test_credit_supported_commitment(self):
        o = obs(cash=6_500_000)  # headroom 5,500,000 of which 5,000,000 is undrawn credit
        ev = {"op": "validate", "actions": [HIRE], "answer": {"valid": True, "results": [ok(HIRE)]},
              "preview": preview(1_211_413, headroom(o), ("fee", "exit"))}
        r = afford(ev, o)
        self.assertEqual(r["status"], "funded_with_credit")
        self.assertEqual(r["measurements"]["fundingAtObservation"]["headroomWithoutCreditCents"], 500_000)

    def test_no_new_commitment(self):
        o = obs()
        ev = {"op": "validate", "actions": [TERM], "answer": {"valid": True, "results": [ok(TERM)]},
              "preview": preview(0, headroom(o), ("sev",))}
        self.assertEqual(afford(ev, o)["status"], "no_new_commitment")
        pol = {"op": "validate", "actions": [WORK], "answer": {"valid": True, "results": [ok(WORK)]}}
        self.assertEqual(afford(pol, o)["status"], "no_new_commitment")

    def test_refused_for_another_reason_is_not_a_funding_result(self):
        o = obs()
        ev = {"op": "validate", "actions": [HIRE], "answer": {"valid": False, "results": [ok(HIRE, "lead_too_short")]},
              "preview": {"ok": False, "guaranteed": []}}
        r = afford(ev, o)
        self.assertEqual(r["status"], "not_tested")
        self.assertIn("lead_too_short", r["basis"])

    def test_recurring_obligations_and_immediate_ledger(self):
        o = obs()
        tele = {"op": "start", "effective": "2026-11-19", "monthlyRetainerCents": 150_000}
        ev = {"op": "validate", "actions": [HIRE, ALLOC, ROSTER, TELE],
              "answer": {"valid": True, "results": [ok(HIRE), ok(ALLOC), ok(ROSTER), ok(TELE)]},
              "preview": preview(2_000_000, headroom(o), ("fee", "week", "exit", "alloc", "roster"),
                                 teleradiology=[tele])}
        m = afford(ev, o)["measurements"]
        rec = {x["label"]: x["cents"] for x in m["recurring"]}
        self.assertEqual(rec["Contract guarantee per full week · CT technologist"], 220_000)
        self.assertEqual(rec["Salary allocation change · Dr. Avery (E07)"], 986_302)
        self.assertEqual(rec["Teleradiology retainer, open-ended"], 150_000)
        self.assertNotIn("Contract guarantee per full week · Riley (E03)", rec)  # unchanged by the roster
        self.assertEqual([x["cents"] for x in m["immediateLedger"]], [300_000])

    def test_extension_commitment_is_the_engines_reservation(self):
        o = obs(cash=6_000_000)  # headroom without credit 0
        msg = "Shift extended: Riley today until 17:15 (+1h approved overtime · ≈$54) · staffs CT 16:15–17:15."
        val = {"op": "validate", "actions": [EXT], "answer": {"valid": True, "results": [ok(EXT, msg=msg)]}}
        app = {"op": "apply", "actions": [EXT], "answer": {"applied": True, "results": [ok(EXT, msg=msg)]},
               "fundingAfter": {**F.funding_obs(o), "reservedCents": 6_750}}
        r = afford(val, o, events=[val, app])
        self.assertEqual(r["measurements"]["extensionCommitmentCents"], 6_750)
        self.assertEqual(r["status"], "funded_with_credit")
        self.assertEqual(r["measurements"]["extensions"][0]["estimateDollars"], 54)
        # The same batch never applied: no reservation, so no exact commitment.
        self.assertEqual(afford(val, o, events=[val])["status"], "evidence_unavailable")

    def test_no_funding_rule_in_the_economics(self):
        o = obs(solvency=False)
        ev = {"op": "validate", "actions": [HIRE], "answer": {"valid": True, "results": [ok(HIRE)]},
              "preview": {"ok": True, "guaranteed": [LINES["fee"]]}}
        r = afford(ev, o)
        self.assertEqual(r["status"], "evidence_unavailable")
        self.assertEqual(r["basis"], "no funding rule in this economics")

    def test_missing_observation(self):
        ev = {"op": "validate", "actions": [HIRE], "answer": {"valid": True, "results": [ok(HIRE)]},
              "preview": preview(1_000, 2_000)}
        r = afford(ev, None)
        self.assertEqual(r["status"], "evidence_unavailable")
        self.assertFalse(r["completeness"]["observation"])


class Reversibility(unittest.TestCase):
    def test_reversible_action_withdrawable_until_its_notice(self):
        o = obs()
        item = {"employee": "E07", "op": "set", "effective": "2026-11-19", "withdrawBy": "2026-11-05",
                "beforeBP": 6000, "afterBP": 8000, "beforeSiteMinA": 1710, "beforeSiteMinB": 1550,
                "afterSiteMinA": 2280, "afterSiteMinB": 2070, "beforeSalary30Cents": 2_958_904,
                "afterSalary30Cents": 3_945_206, "backlogStudies": 5, "backlogUrgent": 0}
        val = {"op": "validate", "actions": [ALLOC], "answer": {"valid": True, "results": [ok(ALLOC)]},
               "preview": preview(986_302, headroom(o), ("alloc",), allocations=[item])}
        app = {"op": "apply", "actions": [ALLOC],
               "answer": {"applied": True, "results": [ok(ALLOC, eff="2026-11-19")]}}
        rev, exp = execute(app, val, o)
        self.assertEqual(rev["status"], "withdrawable")
        self.assertEqual(rev["facts"]["withdrawBy"], "2026-11-05")
        self.assertEqual(exp["directChange"]["afterSiteMinA"] - exp["directChange"]["beforeSiteMinA"], 570)

    def test_policy_change_is_reversible_at_no_recorded_cost(self):
        o = obs()
        app = {"op": "apply", "actions": [WORK], "answer": {"applied": True, "results": [ok(WORK)]}}
        rev, exp = execute(app, None, o)
        self.assertEqual(rev["status"], "reversible_no_recorded_cost")
        self.assertEqual(exp["directChange"], {"resource": "reporting order", "before": "urgent_then_fifo",
                                               "after": "oldest_first"})

    def test_irreversible_historical_cost_of_a_hire(self):
        o = obs()
        item = {"requestId": "h1", "offer": "CT technologist", "tier": "premium", "contractMin": 2400,
                "rosterMinA": 2250, "rosterMinB": 0, "startDate": "2026-11-28", "productiveDate": "2026-12-15",
                "trainingDays": 17, "opensSlots": [{"modality": "CT", "count": 45}]}
        val = {"op": "validate", "actions": [HIRE], "answer": {"valid": True, "results": [ok(HIRE)]},
               "preview": preview(1_211_413, headroom(o), ("fee", "week", "exit"), hires=[item])}
        app = {"op": "apply", "actions": [HIRE], "answer": {"applied": True, "results": [ok(HIRE, eff="2026-11-28")]}}
        rev, exp = execute(app, val, o)
        self.assertEqual(rev["status"], "reversible_with_retained_obligation")
        self.assertEqual(rev["facts"]["recruitingFeeCents"], 300_000)  # not refunded by any reversal
        self.assertEqual(rev["facts"]["earliestExitCents"], 911_413)
        self.assertIn("not refunded", F.MATRIX["hire"]["retained"])
        self.assertEqual(exp["directChange"]["opensSlots"], [{"modality": "CT", "count": 45}])
        self.assertEqual(exp["directChange"]["resource"], "technologist capacity")

    def test_termination_notice(self):
        o = obs()
        item = {"employee": "E03", "lastDay": "2026-11-19", "endDate": "2026-11-20", "noticeDays": 14,
                "severanceCents": 360_000, "severancePayday": "2026-11-27", "displaced": 7}
        val = {"op": "validate", "actions": [TERM], "answer": {"valid": True, "results": [ok(TERM)]},
               "preview": preview(0, headroom(o), ("sev",), terminations=[item])}
        app = {"op": "apply", "actions": [TERM], "answer": {"applied": True, "results": [ok(TERM, eff="2026-11-20")]}}
        rev, exp = execute(app, val, o)
        self.assertEqual(rev["status"], "not_reversible")
        self.assertEqual((rev["facts"]["noticeDays"], rev["facts"]["severanceCents"]), (14, 360_000))
        self.assertEqual(exp["directChange"]["displacedBookingsInBatch"], 7)
        self.assertEqual(exp["directChange"]["role"], "technologist")

    def test_contract_cancellation_and_start(self):
        o = obs()
        tele_c = {"op": "cancel", "contract": "T00001", "plan": "standard-v1", "windows": ["weeknights"],
                  "scope": "all", "effective": "2026-11-01", "endDate": "2026-12-20"}
        val = {"op": "validate", "actions": [CANCEL], "answer": {"valid": True, "results": [ok(CANCEL)]},
               "preview": preview(0, headroom(o), (), teleradiology=[tele_c])}
        app = {"op": "apply", "actions": [CANCEL], "answer": {"applied": True, "results": [ok(CANCEL)]}}
        rev, _ = execute(app, val, o)
        self.assertEqual((rev["matrix"], rev["status"]), ("contract_teleradiology:cancel", "not_reversible"))
        tele_s = {"op": "start", "plan": "standard-v1", "windows": ["weeknights"], "scope": "all",
                  "effective": "2026-11-19", "minimumTermEnd": "2026-12-19", "monthlyRetainerCents": 150_000,
                  "guaranteedCents": 150_000, "guaranteedThrough": "2026-12-19"}
        val = {"op": "validate", "actions": [TELE], "answer": {"valid": True, "results": [ok(TELE)]},
               "preview": preview(150_000, headroom(o), (), teleradiology=[tele_s])}
        app = {"op": "apply", "actions": [TELE], "answer": {"applied": True, "results": [ok(TELE, eff="2026-11-19")]}}
        rev, exp = execute(app, val, o)
        self.assertEqual(rev["status"], "reversible_with_retained_obligation")
        self.assertEqual(rev["facts"]["guaranteedCents"], 150_000)
        self.assertEqual(exp["directChange"]["resource"], "teleradiology coverage")

    def test_roster_change(self):
        o = obs()
        item = {"employee": "E03", "op": "set", "effective": "2026-11-23", "contractMin": 2400, "beforeMinA": 2400,
                "beforeMinB": 2400, "afterMinA": 1920, "afterMinB": 2400}
        val = {"op": "validate", "actions": [ROSTER], "answer": {"valid": True, "results": [ok(ROSTER)]},
               "preview": preview(0, headroom(o), ("roster",), rosters=[item])}
        app = {"op": "apply", "actions": [ROSTER], "answer": {"applied": True, "results": [ok(ROSTER, eff="2026-11-23")]}}
        rev, exp = execute(app, val, o)
        self.assertEqual(rev["status"], "withdrawable")
        self.assertEqual((exp["directChange"]["beforeMinA"], exp["directChange"]["afterMinA"]), (2400, 1920))
        self.assertIn("guarantee is owed whatever is rostered", F.MATRIX["set_roster"]["retained"])

    def test_missing_preview_leaves_the_rule_only(self):
        app = {"op": "apply", "actions": [HIRE], "answer": {"applied": True, "results": [ok(HIRE, eff="2026-11-28")]}}
        rev, exp = execute(app, None, obs())
        self.assertEqual(rev["evidence"], "rule only: no preview for the batch")
        self.assertEqual(exp["status"], "evidence_unavailable")

    def test_unchanged_results_are_not_executed_actions(self):
        app = {"op": "apply", "actions": [TERM], "answer": {"applied": True, "results": [ok(TERM, "unchanged")]}}
        self.assertEqual(execute(app, None, obs()), [])


class Exposure(unittest.TestCase):
    def test_direct_capacity_change_of_an_extension(self):
        msg = ("Shift extended: Riley today until 17:15 (+1h approved overtime · ≈$54) · staffs CT 16:15–17:15 · "
               "2 slots newly bookable.")
        app = {"op": "apply", "actions": [EXT], "answer": {"applied": True, "results": [ok(EXT, msg=msg)]}}
        _, exp = execute(app, None, obs())
        self.assertEqual(exp["status"], "direct_change")
        self.assertEqual(exp["directChange"]["covers"], "staffs CT")
        self.assertEqual((exp["directChange"]["minutes"], exp["directChange"]["role"]), (60, "technologist"))

    def test_subsequent_outcomes_are_separate_and_unattributed(self):
        app = {"op": "apply", "actions": [EXT], "answer": {"applied": True, "results": [ok(EXT)]}}
        _, exp = execute(app, None, obs())
        self.assertEqual(exp["subsequent"], {"observedIn": "runs[].timeline", "attribution": "none"})
        for k in ("cashCents", "ordersLostSinceStart", "operatingProfitCents"):
            self.assertNotIn(k, json.dumps(exp))
        st = F.observed_state(obs())
        self.assertEqual((st["ordersLostSinceStart"], st["unreadStudies"]), (3, 4))
        self.assertEqual(st["headcount"], {"radiologist:active": 1, "technologist:active": 1})


class Recordings(unittest.TestCase):
    def test_llm_agent_3_attempts_link_by_their_batches(self):
        rec = {"schema": V3, "attempts": [
            {"id": "4.1", "validation": [{"actions": [HIRE], "indices": [0]}]},
            {"id": "4.2", "validation": [{"actions": [EXT], "indices": [0]}, {"actions": [ALLOC], "indices": [1]}]}]}
        events = [{"op": "validate", "actions": [HIRE]}, {"op": "validate", "actions": [EXT]},
                  {"op": "validate", "actions": [ALLOC]}, {"op": "apply", "actions": [EXT]}]
        self.assertEqual(F.attempts_for(rec, events), {0: ("4.1", [0]), 1: ("4.2", [0]), 2: ("4.2", [1])})

    def test_llm_agent_2_links_the_final_reply_only(self):
        rec = {"agentVersion": "2", "attempts": 2, "proposed": [EXT, ALLOC],
               "refusals": [{"attempt": 1, "code": "lead_too_short"}]}
        events = [{"op": "apply", "actions": [EXT], "submission": True},
                  {"op": "apply", "actions": [ALLOC], "submission": True}]
        self.assertEqual(F.attempts_for(rec, events), {0: ("final", [0]), 1: ("final", [1])})
        self.assertEqual(F.unrecorded_attempts([rec]), (1, {"lead_too_short": 1}))
        self.assertEqual(F.unrecorded_attempts([{"schema": V3, "attempts": [{}, {}]}]), (0, {}))

    def test_llm_agent_2_refused_final_reply_is_validated_again(self):
        recs = [{"proposed": [EXT, HIRE], "batches": []}, {"proposed": [EXT], "batches": [[EXT]]},
                {"schema": V3, "proposed": [HIRE]}]
        self.assertEqual(F.v2_extras(recs), {0: [[EXT], [HIRE]]})
        rec = {"proposed": [EXT, HIRE], "validation": [{"valid": True}, {"valid": False}]}
        self.assertEqual(F.recorded_validation(rec)[F.canon([HIRE])], {"valid": False})


class Identity(unittest.TestCase):
    def records(self):
        o = obs()
        out = []
        events = [{"op": "validate", "actions": [a], "answer": {"valid": True, "results": [ok(a)]},
                   "preview": preview(0, headroom(o))} for a in (HIRE, TERM, EXT)]
        for k, ev in enumerate(events):
            an = F.Analysis()
            c = ctx(events, k)
            c["point"] = {"index": 3 - k, "point": 3 - k, "time": T}
            F.affordability(an, c, 1, ev, o, "econ")
            out += an.records
        return out

    def test_stable_evidence_ids(self):
        a, b = self.records(), self.records()
        self.assertEqual([r["id"] for r in a], [r["id"] for r in b])
        self.assertEqual(len({r["id"] for r in a}), 3)
        self.assertTrue(all(r["id"].startswith("fe1-") and len(r["id"]) == 20 for r in a))
        self.assertNotEqual(F.stable_id("p", 1, 0, "0", "AFFORDABILITY", 1),
                            F.stable_id("p", 1, 1, "0", "AFFORDABILITY", 1))

    def test_stable_output_order(self):
        recs = self.records()
        want = F.order(recs)
        for seed in range(5):
            shuffled = recs[:]
            random.Random(seed).shuffle(shuffled)
            self.assertEqual(F.order(shuffled), want)
        self.assertEqual([r["decision"]["index"] for r in want], [1, 2, 3])


# --- the engine replay, against a stand-in engine -------------------------------------------------

FAKE = r'''#!/usr/bin/env python3
import json, os, sys
pairs, req = [], None
for line in open(os.environ["FE_TRANSCRIPT"], encoding="utf-8"):
    d = json.loads(line)
    if "op" in d:
        req = d
    else:
        pairs.append(d)
canned = json.loads(os.environ["FE_CANNED"])
diverge = os.environ.get("FE_DIVERGE")
i = 0
for line in sys.stdin:
    r = json.loads(line)
    if r.get("op") == "save":
        open(r["args"]["path"], "w").write("{}")  # into the engine's working directory
    if str(r.get("id")).startswith("fe-"):
        out = canned[r["op"]]
    else:
        out = pairs[i]
        i += 1
        if diverge and r.get("op") == diverge:
            out = dict(out, result={"changed": True})
    print(json.dumps(out), flush=True)
'''


class EngineReplay(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        d = self.tmp.name
        self.binary = os.path.join(d, "engine")
        with open(self.binary, "w", encoding="utf-8") as f:
            f.write(FAKE)
        os.chmod(self.binary, os.stat(self.binary).st_mode | stat.S_IEXEC)
        self.rec = os.path.join(d, "rec", "e-p-s7-decisions.jsonl")
        os.makedirs(os.path.dirname(self.rec))
        dig = {"state": "a"}
        with open(self.rec, "w", encoding="utf-8") as f:
            f.write(json.dumps({"schema": V3, "point": 0, "time": T, "digests": dig, "attempts": [
                {"id": "0.1", "validation": [{"actions": [HIRE], "indices": [0]}]}]}) + "\n")
        self.cfg = os.path.join(d, "replay.json")
        with open(self.cfg, "w", encoding="utf-8") as f:
            json.dump({"external": [{"id": "p", "command": ["agent", "--replay", self.rec]}]}, f)
        run = os.path.join(d, "replays", "out", "runs", "run-1")
        os.makedirs(os.path.join(run, "policy-evidence"))
        with open(os.path.join(run, "policy-evidence", "decisions.jsonl"), "w", encoding="utf-8") as f:
            f.write(json.dumps({"replayed": os.path.basename(self.rec)}) + "\n")
        with open(os.path.join(run, "run.json"), "w", encoding="utf-8") as f:
            json.dump({"scenario": {"ref": "s", "id": "s"}, "seed": 7, "days": 1,
                       "final": {"time": "2026-11-06T07:00:00"}}, f)
        o = obs()
        hello = {"ok": True, "result": {"clock": {"t": 0}, "scenario": {"fingerprint": "f"}}}
        res = [ok(HIRE, eff="2026-11-28")]
        script = [({"id": 1, "op": "hello"}, hello),
                  ({"id": 2, "op": "digests"}, {"ok": True, "result": {"digests": dig}}),
                  ({"id": 3, "op": "observe", "args": {}}, {"ok": True, "result": {"observation": o}}),
                  ({"id": 4, "op": "actions"}, {"ok": True, "result": {"types": []}}),
                  ({"id": 5, "op": "validate", "args": {"actions": [HIRE]}},
                   {"ok": True, "result": {"valid": True, "results": res}}),
                  ({"id": 6, "op": "apply", "args": {"actions": [HIRE]}},
                   {"ok": True, "result": {"applied": True, "results": res}}),
                  ({"id": 7, "op": "advance", "args": {}}, {"ok": True, "result": {}}),
                  ({"id": 8, "op": "save", "args": {"path": "session.json"}},
                   {"ok": True, "result": {"path": "session.json"}})]
        self.transcript = os.path.join(run, "transcript.jsonl")
        with open(self.transcript, "w", encoding="utf-8") as f:
            for q, a in script:
                f.write(json.dumps(q) + "\n" + json.dumps(a) + "\n")
        hire_pv = {"requestId": "h1", "offer": "CT technologist", "startDate": "2026-11-28",
                   "productiveDate": "2026-12-15", "opensSlots": [{"modality": "CT", "count": 45}]}
        canned = {"preview": {"ok": True, "result": {"preview": preview(1_211_413, headroom(o), ("fee", "week", "exit"),
                                                                      hires=[hire_pv])}},
                  "observe": {"ok": True, "result": {"observation": obs(reserved=911_413)}},
                  "validate": {"ok": True, "result": {"valid": True, "results": []}}}
        os.environ["FE_TRANSCRIPT"] = self.transcript
        os.environ["FE_CANNED"] = json.dumps(canned)
        os.environ.pop("FE_DIVERGE", None)
        self.corpus = os.path.join(d, "corpus.json")
        with open(self.corpus, "w", encoding="utf-8") as f:
            json.dump({"schema": F.CORPUS_SCHEMA, "id": "t", "groups": [
                {"id": "g", "economics": "test economics", "suiteRuns": [],
                 "replays": [{"config": self.cfg, "out": "out"}]}]}, f)

    def tearDown(self):
        os.environ.pop("FE_DIVERGE", None)
        self.tmp.cleanup()

    def sources(self):
        out = {}
        for root, _, files in os.walk(self.tmp.name):
            for n in files:
                p = os.path.join(root, n)
                with open(p, "rb") as f:
                    out[p] = hashlib.sha256(f.read()).hexdigest()
        return out

    def test_source_immutability_and_determinism(self):
        before = self.sources()
        cwd = os.getcwd()
        os.chdir(self.tmp.name)
        try:
            a = F.analyze(self.binary, self.corpus, os.path.join(self.tmp.name, "replays"))
            b = F.analyze(self.binary, self.corpus, os.path.join(self.tmp.name, "replays"), jobs=4)
        finally:
            os.chdir(cwd)
        # Nothing written: no source changed, and the run's own save went to the engine's scratch directory.
        self.assertEqual(self.sources(), before)
        self.assertEqual(json.dumps(a, sort_keys=True), json.dumps(b, sort_keys=True))
        st = {(r["dimension"], r["status"]) for r in a["records"]}
        self.assertEqual(st, {("AFFORDABILITY", "funded_without_credit"),
                              ("REVERSIBILITY", "reversible_with_retained_obligation"),
                              ("OPERATIONAL_EXPOSURE", "direct_change")})
        self.assertEqual(a["runs"][0]["replay"], {"requests": 8, "identical": True, "pointsMatched": 1, "points": 1})
        aff = next(r for r in a["records"] if r["dimension"] == "AFFORDABILITY")
        self.assertEqual((aff["attempt"]["id"], aff["execution"]), ("0.1", "applied"))

    def test_a_different_answer_stops_the_analysis(self):
        os.environ["FE_DIVERGE"] = "apply"
        with self.assertRaises(F.ReplayMismatch):
            F.analyze(self.binary, self.corpus, os.path.join(self.tmp.name, "replays"))


class ParameterEvidenceFreeze(unittest.TestCase):
    """parameter-evidence/v1 is frozen at V0.9a: its committed outputs
    reproduce the freeze record, and any change to them is refused."""

    def test_committed_outputs_match_the_freeze(self):
        import check_parameter_frozen as C
        out = os.path.join(HERE, "..", "..", "docs", "benchmark", "v09a-dev")
        self.assertEqual(C.check(out), [])

    def test_a_changed_output_is_refused(self):
        import shutil
        import check_parameter_frozen as C
        src = os.path.join(HERE, "..", "..", "docs", "benchmark", "v09a-dev")
        with tempfile.TemporaryDirectory() as d:
            for n in ("parameter-evidence.json", "parameter-tables.md"):
                shutil.copy(os.path.join(src, n), d)
            with open(os.path.join(d, "parameter-tables.md"), "a", encoding="utf-8") as f:
                f.write("tuned\n")
            self.assertEqual(C.check(d), ["parameter-tables.md differs from the frozen output"])


if __name__ == "__main__":
    unittest.main()
