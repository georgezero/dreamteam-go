"""Tests of dreamteam.parameter-evidence/v1: each rule with its negative
controls, abstention, both recording formats, stable identifiers and
order. Synthetic replies only; no model is contacted.

    python3 -I -B -m unittest discover -s analysis/evidence
"""

import copy
import os
import sys
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import parameter_evidence as P  # noqa: E402

OBS = {"staff": [{"id": "E03", "name": "Riley"}, {"id": "E05", "name": "Casey"}, {"id": "E06", "name": "Morgan"},
                 {"id": "E07", "name": "Dr. Avery"}, {"id": "E08", "name": "Dr. Blake"}, {"id": "E09", "name": "Sam"},
                 {"id": "E10", "name": "Samantha"}]}
CAT = {"types": [{"type": "set_reading_allocation", "targets": [
    {"employee": "E07", "inEffectBP": 6000, "versions": [{"siteBP": 6000, "inEffect": True, "pending": False}]},
    {"employee": "E08", "inEffectBP": 5000, "versions": [{"siteBP": 5000, "inEffect": True, "pending": False}]}]}]}
DAY = "2026-11-05"  # a Thursday


def ext(emp, minutes=120, date=None):
    a = {"type": "extend_shift", "employee": emp, "minutes": minutes}
    if date:
        a["date"] = date
    return a


def alloc(emp, bp, eff="2026-11-19"):
    return {"type": "set_reading_allocation", "employee": emp, "op": "set", "effective": eff, "siteAllocationBP": bp}


def hire(cents, start="2026-12-03"):
    return {"type": "hire", "role": "technologist", "rateCents": cents, "contractedWeeklyMinutes": 2400,
            "startDate": start, "requestId": "h1"}


def ctx():
    return {"recording": {"path": "x.jsonl", "sha256": "0" * 64, "group": "g", "format": "llm-agent 2"},
            "run": {"model": "m"}, "lines": {3: 4}, "group": "g", "format": "llm-agent 2",
            "completeness": lambda rec, at: {"attemptLevel": at is not None}}


def run(reason, acts, obs=OBS, cat=CAT, at=None, vals=None, time=DAY + "T07:00:00", an=None):
    an = an or P.Analysis()
    rec = {"point": 3, "time": time}
    P.analyze_reply(an, ctx(), rec, at, reason, acts, vals or [None] * len(acts or []), obs, cat)
    return an


def got(an, rule=None):
    return [(c["rule"]["id"], c["status"]) for c in an.checks if rule is None or c["rule"]["id"] == rule]


class Amount(unittest.TestCase):
    def test_correct_and_incorrect_hourly_rate(self):
        self.assertEqual(got(run("I will hire a CT technologist at $44/hour.", [hire(4400)]), "AMOUNT_MISMATCH"),
                         [("AMOUNT_MISMATCH", "consistent")])
        an = run("I will hire a CT technologist at $45.50 per hour.", [hire(4400)])
        self.assertEqual(got(an, "AMOUNT_MISMATCH"), [("AMOUNT_MISMATCH", "supported")])
        c = an.checks[0]
        self.assertEqual((c["stated"]["value"], c["submitted"]["value"], c["normalization"]),
                         (4550, 4400, "N-MONEY-HOURLY"))

    def test_cents_versus_dollars(self):
        an = run("I will hire a technologist at $4,400 per hour.", [hire(4400)])
        self.assertEqual(got(an, "AMOUNT_MISMATCH"), [("AMOUNT_MISMATCH", "supported")])
        self.assertIn("cents read as dollars", an.checks[0]["basis"])

    def test_no_one_unit_tolerance(self):
        self.assertEqual(got(run("I will hire a technologist at $44.01/hr.", [hire(4400)]), "AMOUNT_MISMATCH"),
                         [("AMOUNT_MISMATCH", "supported")])

    def test_money_without_an_hourly_marker_is_not_bound(self):
        self.assertEqual(got(run("I will hire a technologist, about $91k a year.", [hire(4400)]), "AMOUNT_MISMATCH"),
                         [])

    def test_allocation_percent_and_basis_points(self):
        an = run("I will reduce Dr. Blake's allocation to 30% (3000 BP) from Nov 19.", [alloc("E08", 3000)])
        self.assertEqual(sorted(got(an)), [("AMOUNT_MISMATCH", "consistent"), ("DATE_MISMATCH", "consistent"),
                                           ("DIRECTION_MISMATCH", "consistent"), ("TARGET_MISMATCH", "consistent")])
        an = run("I will reduce Dr. Blake's allocation to 40%.", [alloc("E08", 3000)])
        self.assertIn(("AMOUNT_MISMATCH", "supported"), got(an))

    def test_change_by_an_amount_abstains(self):
        an = run("I will reduce Dr. Blake's allocation by 20%.", [alloc("E08", 3000)])
        self.assertEqual(got(an, "AMOUNT_MISMATCH"), [("AMOUNT_MISMATCH", "unresolved")])


class Target(unittest.TestCase):
    def test_correct_and_incorrect_target(self):
        self.assertEqual(got(run("I will extend Riley's shift by 2 hours.", [ext("E03")]), "TARGET_MISMATCH"),
                         [("TARGET_MISMATCH", "consistent")])
        an = run("I will extend Morgan's shift by 2 hours.", [ext("E05")])
        self.assertEqual(got(an, "TARGET_MISMATCH"), [("TARGET_MISMATCH", "supported")])
        c = an.checks[0]
        self.assertEqual((c["stated"]["value"], c["submitted"]["value"], c["action"]["ref"]), ("E06", "E05", "3.final#0"))
        # The duration of an action the statement does not name is not checked.
        self.assertEqual(got(an, "DURATION_MISMATCH"), [("DURATION_MISMATCH", "unresolved")])

    def test_similar_names(self):
        self.assertEqual(got(run("I will extend Samantha's shift by 2 hours.", [ext("E10")]), "TARGET_MISMATCH"),
                         [("TARGET_MISMATCH", "consistent")])
        self.assertEqual(got(run("I will extend Sam's shift by 2 hours.", [ext("E10")]), "TARGET_MISMATCH"),
                         [("TARGET_MISMATCH", "supported")])

    def test_a_name_that_fits_two_employees_abstains(self):
        obs = {"staff": OBS["staff"] + [{"id": "E11", "name": "Avery Jones"}]}
        an = run("I will extend Avery's shift by 2 hours.", [ext("E07")], obs=obs)
        self.assertEqual(got(an, "TARGET_MISMATCH"), [("TARGET_MISMATCH", "unresolved")])
        self.assertNotIn("supported", [s for _, s in got(an)])

    def test_several_named_staff(self):
        an = run("I will extend Riley and Casey by 2 hours.", [ext("E03"), ext("E05")])
        self.assertEqual(sorted(got(an, "TARGET_MISMATCH")), [("TARGET_MISMATCH", "consistent")] * 2)
        # A name after a subordinate break is not read as a target.
        an = run("I will extend Riley to cover for Casey.", [ext("E03")])
        self.assertEqual(got(an, "TARGET_MISMATCH"), [("TARGET_MISMATCH", "consistent")])
        # A second name in the phrase with no action of its own: unresolved, never supported.
        an = run("I will extend Riley with Casey by 2 hours.", [ext("E03")])
        self.assertEqual(sorted(got(an, "TARGET_MISMATCH")),
                         [("TARGET_MISMATCH", "consistent"), ("TARGET_MISMATCH", "unresolved")])

    def test_contract(self):
        c = {"type": "contract_teleradiology", "op": "cancel", "contract": "T00002", "endDate": "2026-11-30"}
        self.assertEqual(got(run("I will cancel teleradiology contract T00001.", [c]), "TARGET_MISMATCH"),
                         [("TARGET_MISMATCH", "supported")])
        self.assertEqual(got(run("I will cancel teleradiology contract T00002.", [c]), "TARGET_MISMATCH"),
                         [("TARGET_MISMATCH", "consistent")])


class Date(unittest.TestCase):
    def test_calendar_date(self):
        t = {"type": "terminate", "employee": "E03", "lastDay": "2026-11-19"}
        self.assertEqual(got(run("I will terminate Riley effective Nov 19.", [t]), "DATE_MISMATCH"),
                         [("DATE_MISMATCH", "consistent")])
        self.assertEqual(got(run("I will terminate Riley effective 2026-11-20.", [t]), "DATE_MISMATCH"),
                         [("DATE_MISMATCH", "supported")])

    def test_weekday(self):
        self.assertEqual(got(run("I will extend Riley on Monday by 2 hours.", [ext("E03", date="2026-11-09")]),
                             "DATE_MISMATCH"), [("DATE_MISMATCH", "consistent")])
        self.assertEqual(got(run("I will extend Riley on Tuesday by 2 hours.", [ext("E03", date="2026-11-09")]),
                             "DATE_MISMATCH"), [("DATE_MISMATCH", "supported")])

    def test_relative_dates(self):
        self.assertEqual(got(run("I will extend Riley today by 2 hours.", [ext("E03", date=DAY)]), "DATE_MISMATCH"),
                         [("DATE_MISMATCH", "consistent")])
        self.assertEqual(got(run("I will extend Riley tomorrow by 2 hours.", [ext("E03", date=DAY)]),
                             "DATE_MISMATCH"), [("DATE_MISMATCH", "supported")])
        an = run("I will reduce Dr. Blake's allocation to 30% starting next week.", [alloc("E08", 3000)])
        self.assertEqual(got(an, "DATE_MISMATCH"), [("DATE_MISMATCH", "unresolved")])

    def test_relative_date_without_the_decision_date(self):
        an = run("I will extend Riley today by 2 hours.", [ext("E03", date=DAY)], time=None)
        self.assertEqual(got(an, "DATE_MISMATCH"), [("DATE_MISMATCH", "insufficient_evidence")])

    def test_a_missing_date_parameter_is_not_inferred(self):
        self.assertEqual(got(run("I will extend Riley today by 2 hours.", [ext("E03")]), "DATE_MISMATCH"),
                         [("DATE_MISMATCH", "unresolved")])

    def test_qualified_date_abstains(self):
        an = run("I will hire a technologist starting no later than Dec 3.", [hire(4400, "2026-12-10")])
        self.assertEqual(got(an, "DATE_MISMATCH"), [("DATE_MISMATCH", "unresolved")])

    def test_a_date_defined_by_today(self):
        an = run("I will hire a technologist from today (2026-11-05).", [hire(4400, "2026-12-03")])
        self.assertEqual(got(an, "DATE_MISMATCH"), [("DATE_MISMATCH", "supported")])
        self.assertEqual(an.checks[0]["stated"]["value"], DAY)


class Duration(unittest.TestCase):
    def test_correct_and_incorrect(self):
        self.assertEqual(got(run("I will extend Riley's shift by 2 hours (120 minutes).", [ext("E03", 120)]),
                             "DURATION_MISMATCH"), [("DURATION_MISMATCH", "consistent")])
        an = run("I will extend Riley's shift by 60 minutes.", [ext("E03", 120)])
        self.assertEqual(got(an, "DURATION_MISMATCH"), [("DURATION_MISMATCH", "supported")])
        self.assertEqual((an.checks[-1]["stated"]["value"], an.checks[-1]["submitted"]["value"]), (60, 120))

    def test_clock_times_and_qualifiers(self):
        self.assertEqual(got(run("I will extend Riley's shift to 18:00.", [ext("E03", 120)]), "DURATION_MISMATCH"),
                         [])
        self.assertEqual(got(run("I will extend Riley's shift by at least 2 hours.", [ext("E03", 240)]),
                             "DURATION_MISMATCH"), [("DURATION_MISMATCH", "unresolved")])

    def test_distributive_statements(self):
        an = run("I will extend Riley and Casey by 2 hours each.", [ext("E03", 120), ext("E05", 240)])
        self.assertEqual(sorted((c["action"]["index"], c["status"]) for c in an.checks
                                if c["rule"]["id"] == "DURATION_MISMATCH"), [(0, "consistent"), (1, "supported")])
        an = run("I will extend both by 2 hours.", [ext("E03", 120), ext("E05", 120)])
        self.assertEqual(got(an, "DURATION_MISMATCH"), [("DURATION_MISMATCH", "consistent")] * 2)
        an = run("I will extend all three by 2 hours.", [ext("E03", 120), ext("E05", 120)])
        self.assertEqual(got(an, "DURATION_MISMATCH"), [("DURATION_MISMATCH", "unresolved")])
        # one named person without an action: nothing is bound
        an = run("I will extend Riley and Morgan by 2 hours each.", [ext("E03", 120), ext("E05", 120)])
        self.assertEqual(got(an, "DURATION_MISMATCH"), [("DURATION_MISMATCH", "unresolved")])

    def test_several_values_abstain(self):
        an = run("I will extend Riley by 2 hours and Casey by 4 hours.", [ext("E03", 120), ext("E05", 240)])
        self.assertEqual(got(an, "DURATION_MISMATCH"), [("DURATION_MISMATCH", "unresolved")])


class Direction(unittest.TestCase):
    def test_contradiction_and_agreement(self):
        an = run("I will increase Dr. Blake's allocation to 40%.", [alloc("E08", 4000)])
        self.assertEqual(got(an, "DIRECTION_MISMATCH"), [("DIRECTION_MISMATCH", "supported")])
        self.assertEqual(got(an, "AMOUNT_MISMATCH"), [("AMOUNT_MISMATCH", "consistent")])
        an = run("I will increase Dr. Blake's allocation to 60%.", [alloc("E08", 6000)])
        self.assertEqual(got(an, "DIRECTION_MISMATCH"), [("DIRECTION_MISMATCH", "consistent")])

    def test_pending_change_and_missing_catalog(self):
        cat = copy.deepcopy(CAT)
        cat["types"][0]["targets"][1]["versions"].append({"siteBP": 3000, "pending": True})
        an = run("I will increase Dr. Blake's allocation to 40%.", [alloc("E08", 4000)], cat=cat)
        self.assertEqual(got(an, "DIRECTION_MISMATCH"), [("DIRECTION_MISMATCH", "unresolved")])
        an = run("I will increase Dr. Blake's allocation to 40%.", [alloc("E08", 4000)], cat=None)
        self.assertEqual(got(an, "DIRECTION_MISMATCH"), [("DIRECTION_MISMATCH", "insufficient_evidence")])

    def test_a_verb_about_something_else(self):
        an = run("Raising Dr. Blake's allocation to 60% will reduce teleradiology spending.", [alloc("E08", 6000)])
        self.assertEqual(got(an, "DIRECTION_MISMATCH"), [("DIRECTION_MISMATCH", "consistent")])


class Abstention(unittest.TestCase):
    def test_broad_strategic_language(self):
        self.assertEqual(got(run("We need to cut costs and increase capacity over the coming weeks.",
                                 [ext("E03", 120)])), [])

    def test_descriptions_capabilities_and_other_roles(self):
        """Statements that describe, or name a value in another role, are
        not read: patterns found in the recordings during calibration."""
        cases = [
            ("Dr. Blake's site allocation is currently 6000 BP (60%), which is high.", [alloc("E08", 3000)]),
            ("Dr. Blake's site allocation is low (5000 BP).", [alloc("E08", 10000)]),
            ("The pending allocation change for Dr. Avery takes effect on Oct 31.", [alloc("E08", 3000)]),
            ("Dr. Blake's allocation increase is refused due to insufficient funds.", [alloc("E07", 6000)]),
            ("Dr. Avery can extend up to 19:30 cap, so +3.5h (210 min) is max.", [ext("E07", 120)]),
            ("Extending their shift allows them to read before Dr. Blake arrives.", [ext("E07", 120)]),
            ("Dr. Blake starts at 11:30, so extending them now is premature.", [ext("E07", 120)]),
            ("I will also adjust the teleradiology contract start date to 2026-11-19 as it has a 14-day lead time "
             "from today (Nov 5).", [{"type": "contract_teleradiology", "op": "start", "effective": "2026-11-19"}]),
        ]
        cases += [
            ("Extending Dr. Blake's shift is blocked by the 12-hour daily limit.", [ext("E07", 120)]),
            ("Extending Dr. Blake is not feasible today as they start at 11:30.", [ext("E07", 120)]),
            ("Extending Dr. Blake's shift is unnecessary.", [ext("E07", 120)]),
        ]
        for reason, acts in cases:
            with self.subTest(reason):
                self.assertNotIn("supported", [st for _, st in got(run(reason, acts))])

    def test_a_statement_about_an_action_not_submitted(self):
        """Another statement names the submitted action's employee: a
        statement naming someone else concerns an omitted action."""
        an = run("I will extend Dr. Avery's shift by 120 minutes. I will also extend Dr. Blake's shift.",
                 [ext("E07", 120)])
        self.assertEqual(sorted(got(an, "TARGET_MISMATCH")),
                         [("TARGET_MISMATCH", "consistent"), ("TARGET_MISMATCH", "unresolved")])
        an = run("I will also extend Dr. Blake's shift.", [ext("E07", 120)])
        self.assertEqual(got(an, "TARGET_MISMATCH"), [("TARGET_MISMATCH", "supported")])

    def test_hedged_negated_and_past(self):
        for r in ("I could extend Riley's shift by 60 minutes.", "I will not extend Riley's shift by 60 minutes.",
                  "Riley's shift was extended by 60 minutes yesterday.", "Extend Riley or Casey by 60 minutes."):
            with self.subTest(r):
                self.assertEqual(got(run(r, [ext("E03", 120)])), [])

    def test_revised_reason(self):
        an = run("I will extend Riley by 60 minutes. Wait, actually 2 hours is better.", [ext("E03", 120)])
        self.assertEqual(got(an), [])
        self.assertEqual(an.cov["g|llm-agent 2"]["revisedReasons"], 1)

    def test_multiple_possible_actions(self):
        an = run("I will extend the shifts by 2 hours today.", [ext("E03", 120, DAY), ext("E05", 120, DAY)])
        self.assertEqual(sorted(got(an)), [("DATE_MISMATCH", "unresolved"), ("DURATION_MISMATCH", "unresolved")])

    def test_missing_reason_and_missing_action(self):
        an = run(None, [ext("E03")])
        self.assertEqual((got(an), an.cov["g|llm-agent 2"]["reasonMissing"]), ([], 1))
        an = run("I will extend Riley's shift by 2 hours.", [])
        self.assertEqual((got(an), an.cov["g|llm-agent 2"]["noActionOfType"]), ([], 1))
        an = run("I will extend Riley's shift by 2 hours.", None)
        self.assertEqual((got(an), an.cov["g|llm-agent 2"]["actionsMissing"]), ([], 1))

    def test_missing_observation(self):
        an = run("I will extend E03 by 2 hours.", [ext("E03", 120)], obs=None)
        self.assertEqual(got(an, "TARGET_MISMATCH"), [("TARGET_MISMATCH", "insufficient_evidence")])
        self.assertEqual(got(an, "DURATION_MISMATCH"), [("DURATION_MISMATCH", "consistent")])

    def test_several_action_types_in_one_statement(self):
        an = run("I will extend Riley by 2 hours and hire a technologist at $44/hour.", [ext("E03"), hire(4400)])
        self.assertEqual(got(an), [])


class Formats(unittest.TestCase):
    def v3_record(self):
        def at(k, reason, acts, ok):
            res = [{"ok": ok, "code": "accepted" if ok else "after_close", "action": a} for a in acts]
            return {"id": f"3.{k}", "point": 3, "attempt": k, "parse": {"ok": True}, "reason": reason,
                    "actions": acts, "detailsRequested": None, "used": ok,
                    "validation": [{"actions": acts, "indices": list(range(len(acts))),
                                    "answer": {"valid": ok, "results": res}}]}
        details = {"id": "3.0", "point": 3, "attempt": 0, "parse": {"ok": True}, "reason": "I need details.",
                   "actions": [], "detailsRequested": ["staff"], "used": False, "validation": None}
        unparsed = {"id": "3.9", "point": 3, "attempt": 9, "parse": {"ok": False}, "reason": None, "actions": None,
                    "detailsRequested": None, "used": False, "validation": None}
        return {"schema": P.V3_SCHEMA, "point": 3, "time": DAY + "T07:00:00", "attempts": [
            details, at(1, "I will extend Riley's shift by 60 minutes.", [ext("E03", 120)], False), unparsed,
            at(2, "I will extend Riley's shift by 2 hours.", [ext("E03", 120)], True)]}

    def test_v3_rejected_attempt_is_checked_with_its_validation(self):
        rec = self.v3_record()
        rs = P.replies([rec])
        self.assertEqual([(at["id"] if at else None) for _, at, _, _, _ in rs], ["3.1", "3.2"])
        an = P.Analysis()
        for r, at, reason, acts, vals in rs:
            P.analyze_reply(an, ctx(), r, at, reason, acts, vals, OBS, CAT)
        d = [c for c in an.checks if c["rule"]["id"] == "DURATION_MISMATCH"]
        self.assertEqual([(c["attempt"]["id"], c["status"], c["action"]["validation"]["ok"]) for c in d],
                         [("3.1", "supported", False), ("3.2", "consistent", True)])
        self.assertEqual(d[0]["attempt"]["level"], "attempt")

    def test_llm_agent_2_final_reply_and_validation_mapping(self):
        acts = [ext("E03", 60), {"type": "hire", "role": "technologist", "rateCents": 4400, "startDate": "2026-12-03"},
                ext("E05", 120)]
        # operations first, then staffing: answers [extends], [hire]
        val = [{"valid": True, "results": [{"ok": True, "code": "accepted", "action": acts[0]},
                                           {"ok": False, "code": "after_close", "action": acts[2]}]},
               {"valid": True, "results": [{"ok": True, "code": "accepted", "action": acts[1]}]}]
        rec = {"point": 3, "time": DAY + "T07:00:00", "attempts": 2, "reason": "r", "proposed": acts, "validation": val}
        (_, at, _, _, vals), = P.replies([rec])
        self.assertIsNone(at)
        self.assertEqual([v["code"] for v in vals], ["accepted", "accepted", "after_close"])
        an = run("I will extend Casey's shift by 60 minutes.", acts, vals=vals)
        c = [x for x in an.checks if x["rule"]["id"] == "DURATION_MISMATCH"][0]
        self.assertEqual((c["status"], c["attempt"]["level"], c["action"]["validation"]["code"]),
                         ("supported", "final parsed reply", "after_close"))

    def test_older_format_without_actions(self):
        self.assertEqual(P.replies([{"point": 0, "reason": "r", "proposed": None}])[0][3], None)


class Stability(unittest.TestCase):
    def test_stable_identifiers(self):
        a = run("I will extend Morgan's shift by 60 minutes on Monday.", [ext("E05", 120, "2026-11-10")])
        b = run("I will extend Morgan's shift by 60 minutes on Monday.", [ext("E05", 120, "2026-11-10")])
        self.assertEqual([c["id"] for c in a.checks], [c["id"] for c in b.checks])
        self.assertTrue(all(c["id"].startswith("pe1-") for c in a.checks))
        self.assertEqual(len({c["id"] for c in a.checks}), len(a.checks))

    def test_stable_order(self):
        an = run("I will extend Morgan's shift by 60 minutes on Monday. I will extend Riley's shift by 2 hours.",
                 [ext("E05", 120, "2026-11-10"), ext("E03", 120)])
        for c in an.checks:
            c["recording"] = dict(c["recording"])
        once = [c["id"] for c in P.order(an.checks)]
        self.assertEqual(once, [c["id"] for c in P.order(list(reversed(an.checks)))])
        starts = [c["reasonSpan"]["start"] for c in P.order(an.checks)]
        self.assertEqual(starts, sorted(starts))


class WholeCorpus(unittest.TestCase):
    def test_a_corpus_without_replays_and_without_a_network(self):
        """The whole analysis on a synthetic corpus whose replay is
        missing: nothing is read from a network, and the observation's
        absence is an abstention, never a finding."""
        import json
        import socket
        import tempfile

        def refuse(*a, **k):
            raise AssertionError("a socket was opened")
        real = socket.socket
        socket.socket = refuse
        try:
            with tempfile.TemporaryDirectory() as d:
                rec = os.path.join(d, "r-decisions.jsonl")
                with open(rec, "w") as f:
                    f.write(json.dumps({"point": 0, "time": DAY + "T07:00:00", "agentVersion": "2", "reason":
                                        "I will extend E05 by 60 minutes today.", "proposed": [ext("E05", 120, DAY)],
                                        "validation": None}) + "\n")
                cfg = os.path.join(d, "replay.json")
                with open(cfg, "w") as f:
                    json.dump({"external": [{"id": "p", "command": ["python3", "a.py", "--replay", rec]}]}, f)
                corpus = os.path.join(d, "corpus.json")
                with open(corpus, "w") as f:
                    json.dump({"schema": "dreamteam.parameter-evidence-corpus/v1", "id": "t",
                               "groups": [{"id": "g", "replays": [{"config": cfg, "out": "none"}]}]}, f)
                doc = P.analyze(corpus, os.path.join(d, "replays"))
                again = P.analyze(corpus, os.path.join(d, "replays"))
        finally:
            socket.socket = real
        self.assertEqual(doc, again)
        self.assertEqual(sorted((c["rule"]["id"], c["status"]) for c in doc["checks"]),
                         [("DATE_MISMATCH", "consistent"), ("DURATION_MISMATCH", "supported"),
                          ("TARGET_MISMATCH", "insufficient_evidence")])
        self.assertEqual(doc["coverage"]["g|llm-agent 2"]["replayMissing"], 1)
        self.assertEqual(doc["byRule"]["DURATION_MISMATCH"]["eligible"], 1)
        self.assertTrue(P.tables(doc).startswith("# Parameter evidence"))


if __name__ == "__main__":
    unittest.main()
