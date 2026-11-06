"""Tests of the V0.13d driver. The pure parts always run; the engine parts
run when DREAMTEAM_BINARY names a dreamteam binary (CI sets it). The full
scripted fidelity evidence is v13d_fidelity.py; these tests run its
machinery on a few cases."""

import json
import os
import sys
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import v13d as D  # noqa: E402
import v13d_fidelity as F  # noqa: E402

V = os.path.join(D.ROOT, "docs", "benchmark", "v13d-dev")
BINARY = os.environ.get("DREAMTEAM_BINARY")


class Pure(unittest.TestCase):
    def setUp(self):
        self.rows, self.csha = D.load_contexts(os.path.join(V, "contexts"))

    def test_protocol_is_made_from_the_committed_inputs(self):
        proto, sched = D.draft_protocol(self.rows, self.csha)
        with open(os.path.join(V, "protocol-draft.json"), encoding="utf-8") as f:
            self.assertEqual(json.load(f), json.loads(json.dumps(proto)))
        self.assertEqual(proto["status"], D.DRAFT)
        self.assertEqual(proto["draws"], len(sched))

    def test_schedule(self):
        proto, sched = D.draft_protocol(self.rows, self.csha)
        with open(os.path.join(V, "schedule-draft.json"), encoding="utf-8") as f:
            self.assertEqual(json.load(f)["schedule"], sched)
        # Every unit appears its declared number of times, both arms equally.
        for mod in proto["modelOrder"]:
            for cid, c in proto["contexts"].items():
                if c["kind"] == "historical" and c["model"] != mod:
                    self.assertFalse([s for s in sched if s["context"] == cid and s["model"] == mod])
                    continue
                n = proto["drawsPerUnit"][mod]["aa" if c["aa"] else "unit"]
                for arm in D.ARMS:
                    self.assertEqual(sum(1 for s in sched if (s["context"], s["model"], s["arm"]) == (cid, mod, arm)), n)
        # The arms alternate within rounds, not in blocks.
        arms = [s["arm"] for s in sched[:20]]
        self.assertNotEqual(arms, sorted(arms))

    def test_call_budget_is_the_sum_of_bounds(self):
        proto, sched = D.draft_protocol(self.rows, self.csha)
        self.assertEqual(proto["maxModelCalls"], sum(proto["contexts"][s["context"]]["callBound"] for s in sched))
        self.assertLessEqual(max(c["callBound"] for c in proto["contexts"].values()), proto["maxModelCallsPerDraw"])

    def test_contexts(self):
        hist = [c for c in self.rows if c["kind"] == "historical"]
        cons = [c for c in self.rows if c["kind"] == "constructed"]
        self.assertEqual((len(hist), len(cons)), (9, 12))
        for c in cons:
            self.assertEqual(D.sha256(c["prompt"]), c["promptSha256"])
            self.assertEqual(D.sha256(c["proposal"]["content"]), c["proposal"]["sha256"])

    def test_mover(self):
        a = {"difference": 0.5, "interval": [0.1, 0.8]}
        b = {"difference": 0.1, "interval": [-0.2, 0.4]}
        m = D.mover(a, b)
        self.assertAlmostEqual(m["difference"], 0.4)
        self.assertLess(m["interval"][0], 0.4)
        self.assertGreater(m["interval"][1], 0.4)
        self.assertIsNone(D.mover(None, b))

    def test_analyze_is_a_command(self):
        with self.assertRaises(SystemExit) as e:
            D.main(["analyze", "--help"])
        self.assertEqual(e.exception.code, 0)

    def test_run_doc_is_checked(self):
        sched = [{"seq": 1}, {"seq": 2}]
        rows = [{"seq": 1, "status": "completed"}]
        doc = {"status": "COMPLETE", "drawsCompleted": 1, "drawsScheduled": 2, "drawInterrupted": None, "stopped": None}
        with self.assertRaises(D.ExperimentError):
            D.check_run_doc(doc, rows, sched)
        D.check_run_doc(dict(doc, status="PARTIAL", stopped={"code": "x"}), rows, sched)

    def test_halt_must_follow_from_its_evidence(self):
        cfg = {"maxModelCallsPerDraw": 6, "maxModelCalls": 100, "request": {"timeoutSeconds": 300},
               "stop": {"wallClockSeconds": 14400, "consecutiveTransportFailures": 5}}
        D.check_halt({"code": "transport", "evidence": {"consecutiveTransportFailures": 5}}, cfg)
        for bad in ({"code": "wall_clock", "evidence": {"elapsedSeconds": 10}},
                    {"code": "draw_call_limit", "evidence": {"drawCalls": 2}}, {"code": "invented", "evidence": {}}):
            with self.assertRaises(D.ExperimentError):
                D.check_halt(bad, cfg)

    def test_transport_streak_stops_before_the_next_call(self):
        lim = D.Limits(6, 100, 10 ** 9, 300, max_fails=5, now=lambda: 0.0)
        for _ in range(5):
            lim.before_call(0)
            lim.after_call({"error": "timeout", "status": None})
        with self.assertRaises(D.Halt) as h:
            lim.before_call(0)
        self.assertEqual((h.exception.code, h.exception.evidence["consecutiveTransportFailures"]), ("transport", 5))

    def test_run_refuses_a_draft(self):
        with self.assertRaises(SystemExit):
            D.main(["run", "--protocol", os.path.join(V, "protocol-draft.json"), "--authorization", "x"])


@unittest.skipUnless(BINARY, "DREAMTEAM_BINARY is not set")
class Engine(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        rows, _ = D.load_contexts(os.path.join(V, "contexts"))
        env = {D.SB.SETTINGS_ENV: D.SB.REPLAY_URL}
        s = D.A3.load_settings(os.path.join(D.ROOT, D.settings_files()[D.QWEN]["file"]), env)
        extra = []
        for c in F.extra_contexts():
            prompt, dg = D.prompt_at(BINARY, c, s)
            c.update(prompt=prompt, promptSha256=D.sha256(prompt), digests=dg)
            extra.append(c)
        proto = {"modelOrder": [D.GEMMA, D.QWEN], "settings": {"files": D.settings_files(), "maxRetries": 1, "allowDetails": True},
                 "models": D.load(f"{D.V12B}/protocol.json")["models"],
                 "systemPromptSha256": D.load(f"{D.V12B}/protocol.json")["systemPromptSha256"],
                 "request": {"timeoutSeconds": 300}}
        cls.ctxs = D.Contexts(proto, [r for r in rows if r["kind"] == "constructed"] + extra, None, env)
        cls.pool = cls.ctxs.opener(BINARY)

    @classmethod
    def tearDownClass(cls):
        cls.pool.close()

    def case(self, name, ctx, arm):
        for c in F.CASES:
            if c[0] == name and c[1] == ctx and c[2] == arm:
                r, d = F.run_case(BINARY, {}, self.ctxs, self.pool, c)
                self.assertEqual(r["failed"], [], r)
                return r, d
        raise AssertionError(name)

    def test_selected_and_replayed(self):
        r, d = self.case("alternative selected", "S1-a", "treatment")
        self.assertEqual(d["record"]["inspected"], ["adjusted"])

    def test_operations_kept(self):
        r, d = self.case("mixed operations and staffing: select", "F-mixed", "treatment")
        self.assertEqual(d["batches"][0], [F.OPS])

    def test_hostile_request_id_never_sent(self):
        r, d = self.case("malicious free text", "F-hostile", "treatment")
        turn = d["record"]["turn"]
        self.assertNotIn(F.HOSTILE, turn["message"] + "".join(turn["inspect"].values()))

    def test_retry_exhaustion(self):
        r, d = self.case("retry exhaustion", "S1-a", "treatment")
        self.assertEqual(d["batches"], [])

    def test_tampered_draw_does_not_replay(self):
        r, d = self.case("alternative selected", "S1-a", "treatment")
        out = F.tamper_cases(BINARY, {}, self.ctxs, self.pool, {"alternative selected": d})
        self.assertTrue(all(x["result"] == "refused" for x in out), out)


if __name__ == "__main__":
    unittest.main()
