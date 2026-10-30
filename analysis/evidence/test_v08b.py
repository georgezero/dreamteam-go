"""Tests of the V0.8b additions beside the frozen decision evidence: the
freeze guard, the refusal-response extension, the review sample and the
review analysis.

    python3 -B -m unittest discover -s analysis/evidence
"""

import csv
import hashlib
import json
import os
import shutil
import tempfile
import unittest

import check_frozen as C
import refusal_response as R
import review_analysis as RA
import review_package as RP

HERE = os.path.dirname(os.path.abspath(__file__))

HIRE = {"type": "hire", "requestId": "a", "role": "technologist", "skills": ["XR"], "rateCents": 3600,
        "contractedWeeklyMinutes": 2400, "startDate": "2027-01-04"}
EXT = {"type": "extend_shift", "employee": "E07", "minutes": 120, "date": "2026-11-20"}


class Freeze(unittest.TestCase):
    def test_frozen_analyzer_unchanged(self):
        # Any change to the v1 rules needs a new rule version and the owner's approval.
        fr = C.load_freeze()
        self.assertEqual(C.sha256_file(os.path.join(HERE, "decision_evidence.py")), fr["analyzer"]["sha256"])
        self.assertEqual(fr["contract"], "dreamteam.decision-evidence/v1")
        self.assertTrue(all(r["version"] == 1 for r in fr["rules"]))

    def test_committed_evidence_is_the_frozen_one(self):
        fr = C.load_freeze()
        p = os.path.join(HERE, "..", "..", fr["outputs"]["evidencePath"])
        self.assertEqual(C.sha256_file(p), fr["outputs"]["evidenceSha256"])
        with open(p, encoding="utf-8") as f:
            self.assertEqual(C.normalized_sha256(json.load(f)), fr["outputs"]["evidenceNormalizedSha256"])

    def test_normalize_ignores_only_the_engine_build(self):
        doc = {"inputs": [{"path": "replays/x", "sha256": "1"}, {"path": "docs/r.jsonl", "sha256": "2"}],
               "findings": [{"run": {"engine": "0.8.0-dev.1", "seed": 1}}]}
        other = json.loads(json.dumps(doc))
        other["inputs"][0]["sha256"] = "9"
        other["findings"][0]["run"]["engine"] = "0.8.0-dev.2"
        self.assertEqual(C.normalized_sha256(doc), C.normalized_sha256(other))
        other["inputs"][1]["sha256"] = "9"
        self.assertNotEqual(C.normalized_sha256(doc), C.normalized_sha256(other))
        other = json.loads(json.dumps(doc))
        other["findings"][0]["run"]["seed"] = 2
        self.assertNotEqual(C.normalized_sha256(doc), C.normalized_sha256(other))


class Classify(unittest.TestCase):
    def test_categories(self):
        self.assertEqual(R.classify(HIRE, [(dict(HIRE, requestId="b"), False)]), "identical_repeated")
        self.assertEqual(R.classify(HIRE, [(HIRE, True)]), "refusal_no_longer_applicable")
        self.assertEqual(R.classify(HIRE, [(dict(HIRE, startDate="2027-01-11"), True)]), "modified_proposal")
        self.assertEqual(R.classify(HIRE, [(EXT, True)]), "different_proposal")
        self.assertEqual(R.classify(HIRE, []), "no_action")
        self.assertEqual(R.classify(HIRE, None), "evidence_unavailable")
        self.assertEqual(R.classify(None, [(HIRE, False)]), "evidence_unavailable")

    def test_similar_but_distinct(self):
        # Another employee, or another skill, is another target: a different proposal, not a modification.
        self.assertEqual(R.classify(EXT, [(dict(EXT, employee="E08"), False)]), "different_proposal")
        self.assertEqual(R.classify(HIRE, [(dict(HIRE, skills=["CT"]), False)]), "different_proposal")
        # The same extension on another day is the same target with another value.
        self.assertEqual(R.classify(EXT, [(dict(EXT, date="2026-11-23"), False)]), "modified_proposal")

    def rec(self, point, outcome, proposed, validation, refusals, attempts=1):
        return {"point": point, "time": f"t{point}", "outcome": outcome, "proposed": proposed,
                "validation": validation, "refusals": refusals, "attempts": attempts, "errors": []}

    def test_llm_agent_2_format(self):
        src = {"path": "r", "sha256": "0", "group": "g", "model": "m"}
        bad = {"valid": False, "results": [{"ok": False, "code": "insufficient_funds", "action": HIRE}]}
        recs = [
            # Attempt 1 refused (payload unrecorded), attempt 2 the final reply, also refused.
            self.rec(0, "fallback", [HIRE], [bad], [
                {"attempt": 1, "action": 0, "type": "hire", "code": "lead_too_short", "message": "m"},
                {"attempt": 2, "action": 0, "type": "hire", "code": "insufficient_funds", "message": "n"}], 2),
            self.rec(1, "no_action", [], None, []),
        ]
        ev = R.analyze_sequence(recs, None, src)
        self.assertEqual([e["category"] for e in ev], ["evidence_unavailable", "no_action"])
        self.assertIsNone(ev[0]["refused"])
        self.assertTrue(ev[0]["responseRecorded"])
        # The last decision point's refusal has no later proposal.
        ev = R.analyze_sequence(recs[:1], None, src)
        self.assertEqual(ev[1]["unavailable"], "no later proposal is recorded")

    def test_attempt_level(self):
        src = {"path": "r", "sha256": "0", "group": "probes", "model": "p"}
        recs = [self.rec(0, "actions", [EXT], None, [])]
        att = [{"point": 0, "attempts": [
            {"attempt": 1, "actions": [HIRE], "validation": [{"valid": False, "results": [
                {"ok": False, "code": "insufficient_funds"}]}]},
            {"attempt": 2, "actions": [EXT], "validation": [{"valid": True, "results": [{"ok": True,
                                                                                          "code": "accepted"}]}]}]}]
        ev = R.analyze_sequence(recs, att, src)
        self.assertEqual(len(ev), 1)
        self.assertEqual((ev[0]["category"], ev[0]["response"]), ("different_proposal", {"point": 0, "attempt": 2}))


class Sample(unittest.TestCase):
    def pool(self, n):
        return [{"id": f"x{i}", "recording": {"group": "g" + str(i % 3)}, "run": {"model": "m" + str(i % 2)},
                 "completeness": {"prompt": "verified"}} for i in range(n)]

    def test_order_key_is_stable(self):
        # A fixed hash, not a random number generator: the same on every Python and platform.
        self.assertEqual(RP.order_key("x"), hashlib.sha256(b"v08b-review-1|x").hexdigest())

    def test_draw_deterministic_and_stratified(self):
        a = [it["id"] for it in RP.draw(self.pool(30), 6)]
        b = [it["id"] for it in RP.draw(list(reversed(self.pool(30))), 6)]
        self.assertEqual(a, b)
        strata = {RP.stratum(it) for it in RP.draw(self.pool(30), 6)}
        self.assertEqual(len(strata), 6)
        self.assertEqual(len(RP.draw(self.pool(3), 10)), 3)


class ReviewAnalysis(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.mkdtemp()
        self.manifest = {"seed": "s", "itemsSha256": "h", "items": [
            {"item": "R001", "id": "a", "cell": "X/supported"}, {"item": "R002", "id": "b", "cell": "X/consistent"}]}

    def tearDown(self):
        shutil.rmtree(self.dir)

    def form(self, name, labels, declare=True):
        d = os.path.join(self.dir, name)
        os.makedirs(d)
        with open(os.path.join(d, "form.csv"), "w", newline="") as f:
            w = csv.writer(f)
            w.writerow(["item", "id", "rule", "rule_output", "label", "comment", "correction"])
            for (item, iid), lab in zip((("R001", "a"), ("R002", "b")), labels):
                w.writerow([item, iid, "X", "", lab, "", ""])
        if declare:
            with open(os.path.join(d, "reviewer.json"), "w") as f:
                json.dump({"reviewer": name, "independent": True, "statement": "did not write the rules"}, f)
        return os.path.join(d, "form.csv")

    def test_unreviewed_sample_gives_nothing(self):
        res, problems = RA.analyze(self.manifest, [self.form("a", ["", ""])])
        self.assertIsNone(res)
        self.assertTrue(problems)
        res, problems = RA.analyze(self.manifest, [self.form("b", ["supported", "supported"], declare=False)])
        self.assertIsNone(res)
        res, problems = RA.analyze(self.manifest, [])
        self.assertIsNone(res)

    def test_counts_and_disagreements(self):
        res, problems = RA.analyze(self.manifest, [self.form("a", ["supported", "ambiguous"]),
                                                   self.form("b", ["unsupported", "ambiguous"])])
        self.assertEqual(problems, [])
        self.assertEqual(res["byCell"]["X/supported"]["supported"], 1)
        self.assertEqual(res["byCell"]["X/supported"]["disagreements"], 1)
        self.assertEqual(res["byCell"]["X/consistent"]["disagreements"], 0)
        self.assertNotIn("precision", json.dumps(res).lower().replace("not a precision", ""))


if __name__ == "__main__":
    unittest.main()
