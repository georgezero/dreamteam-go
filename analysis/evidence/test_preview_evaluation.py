"""Unit tests of the V0.9c preview evaluation analysis
(analysis/evidence/preview_evaluation.py). The whole analysis runs in CI on
the committed recordings and is compared with the committed outputs.

    python3 -B -m unittest discover -s analysis/evidence
"""

import json
import os
import tempfile
import unittest

import preview_evaluation as P

STAFF = {"type": "set_reading_allocation", "op": "set", "employee": "E07", "effective": "2026-10-27",
         "siteAllocationBP": 7000}
OPS = {"type": "extend_shift", "employee": "E03", "minutes": 30}
ANSWER = {"preview": {"ok": False, "results": [{"ok": False, "code": "insufficient_funds"}],
                      "terminations": [{"employee": "E04", "displaced": 37}, {"employee": "E05", "displaced": 5}],
                      "funding": {"commitmentCents": 493150, "headroomCents": 1000, "funded": False}}}


def attempt(k, turn, actions=None, parsed=True, details=None, preview=None, used=False, refusals=()):
    return {"id": f"0.{k}", "attempt": k, "turn": turn, "parse": {"ok": parsed} if parsed is not None else None,
            "actions": actions, "detailsRequested": details, "preview": preview, "used": used,
            "refusals": list(refusals), "output": {"sha256": f"o{k}"}, "reason": f"r{k}"}


def v4(point, attempts, summary, batches=(), outcome="actions", prompt="p"):
    return {"schema": P.V4, "point": point, "time": f"t{point}", "attempts": attempts, "preview": summary,
            "batches": list(batches), "outcome": outcome, "promptSha256": prompt,
            "completeness": {"attemptsRecorded": True, "inputsVerifiable": True}}


def summary(status="ok", response="kept", final="kept", staffing="kept", operations="none proposed", resp_at=2):
    return {"version": "llm-agent-preview/v1", "status": status, "proposalAttempt": 1, "reviewAttempt": 2,
            "response": response, "responseAttempt": resp_at, "staffing": staffing, "operations": operations,
            "final": final}


PV_OK = {"actions": [STAFF], "indices": [0], "status": "ok", "answer": ANSWER, "message": {"chars": 900},
         "digestsUnchanged": True, "error": None}


class Records(unittest.TestCase):
    def test_preview_counts_and_denominators(self):
        recs = [
            v4(0, [attempt(1, "first", [STAFF], preview=PV_OK), attempt(2, "review", [STAFF], used=True)],
               summary(), batches=[[STAFF]]),
            v4(1, [attempt(1, "first", [STAFF], preview=PV_OK), attempt(2, "review", [], used=True)],
               summary(response="abandoned", final="abandoned", staffing="dropped"), outcome="no_action"),
            v4(2, [attempt(1, "first", [OPS], used=True)], summary(status="not_applicable", response=None, final=None),
               batches=[[OPS]]),
            v4(3, [attempt(1, "first", [STAFF], preview=dict(PV_OK, status="failed", answer=None, message=None,
                                                             error={"code": "invalid_args"}),
                           refusals=[{"code": "invalid_roster"}])],
               summary(status="failed", response=None, final=None), outcome="fallback"),
        ]
        m = P.record_metrics(recs)
        self.assertEqual(m["initialProposals"], {"withAction": 4, "withStaffingAction": 3,
                                                 "denominator": "decision points"})
        p = m["preview"]
        self.assertEqual(p["eligiblePoints"], 3)
        self.assertEqual(p["status"], {"not_applicable": 1, "ok": 2, "failed": 1, "too_large": 0})
        self.assertEqual(p["response"], {"kept": 1, "revised": 0, "abandoned": 1, "no_reply": 0})
        self.assertEqual(p["final"], {"kept": 1, "revised": 0, "abandoned": 1, "fallback": 0})
        self.assertEqual(p["staffing"]["dropped"], 1)
        self.assertEqual((p["reviewRequests"], p["reviewMessageChars"]), (2, {"total": 1800, "max": 900}))
        self.assertEqual(p["shownWithRefusedAction"], 2)
        self.assertTrue(p["digestsUnchangedAll"])
        self.assertEqual([x["point"] for x in p["previews"]], [0, 1, 3])
        self.assertEqual(p["previews"][1]["reasonAfter"], "r2")
        self.assertEqual(m["submittedActions"], {"total": 2, "staffing": 1, "operations": 1,
                                                 "denominator": "decision points"})
        self.assertEqual(m["validateRefusals"]["actions"], 1)
        self.assertEqual(m["outcomes"], {"actions": 2, "fallback": 1, "no_action": 1})

    def test_baseline_records_have_no_preview_section(self):
        rec = dict(v4(0, [attempt(1, "first", [STAFF], used=True)], None, batches=[[STAFF]]), schema=P.V3)
        m = P.record_metrics([rec])
        self.assertNotIn("preview", m)
        self.assertEqual(m["initialProposals"]["withStaffingAction"], 1)

    def test_first_proposal_skips_details_and_unparsed(self):
        rec = v4(0, [attempt(1, "first", None, parsed=False), attempt(2, "retry", [], details=["hire"]),
                     attempt(3, "details", [OPS], used=True)], summary(status="not_applicable"))
        self.assertEqual(P.first_proposal(rec)["attempt"], 3)

    def test_preview_facts(self):
        f = P.preview_facts(ANSWER)
        self.assertEqual((f["commitmentCents"], f["headroomCents"], f["funded"], f["displacedBookings"],
                          f["terminations"], f["refusedCodes"]), (493150, 1000, False, 42, 2, ["insufficient_funds"]))
        self.assertIsNone(P.preview_facts({"preview": {"ok": True}})["commitmentCents"])


class Matched(unittest.TestCase):
    def test_divergence(self):
        a = [v4(i, [attempt(1, "first", [STAFF] if i == 1 else [], used=True)], None,
                batches=[[STAFF]] if i == 1 else [], prompt=f"p{i}") for i in range(4)]
        b = json.loads(json.dumps(a))
        b[1]["batches"] = []
        b[2]["promptSha256"] = "other"
        b[3]["promptSha256"] = "other"
        m = P.matched(a, b)
        self.assertEqual(m, {"points": 4, "samePrompt": 2, "sameFirstOutput": 2, "sameFirstProposal": 2,
                             "firstPromptDifference": 2, "firstSubmittedDifference": 1})


class V3ViewInputs(unittest.TestCase):
    def test_v4_read_as_v3_with_its_own_hash(self):
        with tempfile.TemporaryDirectory() as d:
            p = os.path.join(d, "x-decisions.jsonl")
            raw = (json.dumps({"schema": P.V4, "point": 0}) + "\n").encode()
            with open(p, "wb") as f:
                f.write(raw)
            inp = P.V3View(None)
            got = [json.loads(l) for l in inp.read(p).decode().splitlines()]
            self.assertEqual(got[0]["schema"], P.V3)
            self.assertEqual(inp.files[p], P.FE.sha256_bytes(raw))
            self.assertEqual(inp.formats[os.path.normpath(p)], "llm-agent 4")
            q = os.path.join(d, "y-decisions.jsonl")
            with open(q, "wb") as f:
                f.write((json.dumps({"schema": P.V3, "point": 0}) + "\n").encode())
            self.assertNotIn(os.path.normpath(q), (inp.read(q), inp.formats)[1])


class PreviewChecks(unittest.TestCase):
    def test_only_the_digests_check_after_a_preview_is_removed(self):
        lines = [{"id": "r1", "op": "digests"}, {"id": "r1", "ok": True, "result": {"digests": "d0"}},
                 {"id": "r2", "op": "preview", "args": {}}, {"id": "r2", "ok": True, "result": {}},
                 {"id": "r3", "op": "digests"}, {"id": "r3", "ok": True, "result": {"digests": "d0"}},
                 {"id": "r4", "op": "validate", "args": {}}, {"id": "r4", "ok": True, "result": {}},
                 {"id": "r5", "op": "digests"}, {"id": "r5", "ok": True, "result": {"digests": "d0"}}]
        with tempfile.TemporaryDirectory() as run, tempfile.TemporaryDirectory() as scratch:
            with open(os.path.join(run, "transcript.jsonl"), "w") as f:
                f.writelines(json.dumps(l) + "\n" for l in lines)
            d, n = P.preview_checks_removed(run, scratch)
            self.assertEqual(n, 1)
            with open(os.path.join(d, "transcript.jsonl")) as f:
                got = [json.loads(l) for l in f]
            self.assertEqual(got, lines[:4] + lines[6:])
            with open(os.path.join(run, "transcript.jsonl"), "w") as f:
                f.writelines(json.dumps(l) + "\n" for l in lines[6:])
            self.assertEqual(P.preview_checks_removed(run, scratch), (run, 0))


class Arithmetic(unittest.TestCase):
    def test_diff(self):
        self.assertEqual(P.diff(5, 3), 2)
        self.assertIsNone(P.diff(None, 3))
        self.assertIsNone(P.diff(True, 1))


if __name__ == "__main__":
    unittest.main()
