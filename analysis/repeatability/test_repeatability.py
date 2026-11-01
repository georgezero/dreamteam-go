"""Tests of the V0.10a repeatability tools: the prompt sample
(prompt_sample.py) and the live run (repeat_run.py), with a stub server.

    python3 -B -m unittest discover -s analysis/repeatability
"""

import json
import os
import tempfile
import unittest

import prompt_sample as S
import repeat_run as R

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
PROTOCOL = os.path.join(ROOT, "docs", "benchmark", "v10a-dev", "protocol.json")


class Sample(unittest.TestCase):
    def test_committed_sample_checks(self):
        self.assertEqual(S.check(os.path.join(ROOT, "docs", "benchmark", "v10a-dev", "sample")), 25)

    def test_category_order(self):
        out = {"sha256": "x"}
        ok = {"ok": True}
        self.assertIsNone(S.category({"output": None}, "r", {}))
        self.assertEqual(S.category({"output": out, "input": {"kind": "preview"}, "refusals": [1]}, "r", {}),
                         "preview_review")
        self.assertEqual(S.category({"output": out, "input": {"kind": "prompt"}, "refusals": [1]}, "r", {}),
                         "validation_refusal")
        st = {"output": out, "input": {"kind": "prompt"}, "id": "3.1", "parse": ok, "actions": [{"type": "hire"}]}
        self.assertEqual(S.category(st, "r", {("r", "3.1"): 1_000_000}), "financially_consequential")
        self.assertEqual(S.category(st, "r", {("r", "3.1"): 999_999}), "staffing_proposal")
        none = dict(st, actions=[])
        self.assertEqual(S.category(none, "r", {}), "no_action")
        self.assertIsNone(S.category(dict(none, detailsRequested=["hire"]), "r", {}))
        self.assertIsNone(S.category(dict(st, actions=[{"type": "extend_shift"}]), "r", {}))
        self.assertIsNone(S.category(dict(st, parse={"ok": False}), "r", {}))

    def test_selection_is_by_rank_only(self):
        c = [{"model": "qwen36-35b-a3b", "category": "no_action", "rank": r, "recording": "x", "attemptId": r}
             for r in ("d", "a", "c", "b")]
        got = S.select([dict(x) for x in c])
        self.assertEqual([x["rank"] for x in got], ["a", "b", "c"])
        self.assertEqual([x["id"] for x in got], ["p01", "p02", "p03"])

    def test_chain(self):
        rec = {"promptSha256": "P", "attempts": [
            {"id": "0.1", "input": {"kind": "prompt", "sha256": "P"}, "output": {"sha256": "O1"}},
            {"id": "0.2", "input": {"kind": "retry", "sha256": "T"}, "output": None},
            {"id": "0.3", "input": {"kind": "resend", "sha256": "T"}, "output": {"sha256": "O3"}}]}
        self.assertEqual(S.chain(rec, 1), [("system", S.SYSTEM_SHA256), ("user", "P")])
        self.assertEqual(S.chain(rec, 3), [("system", S.SYSTEM_SHA256), ("user", "P"), ("assistant", "O1"),
                                           ("user", "T")])

    def test_tampered_message_refused(self):
        d = os.path.join(ROOT, "docs", "benchmark", "v10a-dev", "sample")
        with open(os.path.join(d, "prompts.jsonl"), encoding="utf-8") as f:
            e = json.loads(f.readline())
        rec = S.recording_point(ROOT, e["recording"], e["point"])
        self.assertTrue(S.verify(e, rec))
        e["request"]["messages"][1]["content"] += " "
        with self.assertRaises(S.SampleError):
            S.verify(e, rec)


def stub_post(answers):
    calls = []

    def post(base, data, timeout):
        calls.append(json.loads(data))
        a = answers[(len(calls) - 1) % len(answers)]
        if a is None:
            return None, "", "connection: refused", 0.01
        body = {"id": "x", "model": "m", "choices": [{"message": {"content": a}, "finish_reason": "stop"}],
                "usage": {"prompt_tokens": 10, "completion_tokens": 2}}
        return 200, json.dumps(body), None, 0.01
    return post, calls


def stub_state(healthy=True):
    return lambda base: {"time": "t", "health": {"status": "healthy" if healthy else "busy"}, "models": []}


class Run(unittest.TestCase):
    def setUp(self):
        self.proto, self.sha, self.man, self.prompts = R.load(PROTOCOL)

    def test_schedule_deterministic_and_complete(self):
        a, b = R.schedule(self.proto, self.prompts), R.schedule(self.proto, self.prompts)
        self.assertEqual(a, b)
        n = {m: sum(1 for p in self.prompts if p["model"] == m) for m in self.proto["modelOrder"]}
        self.assertEqual(sum(len(s) for s in a), 15 * n["gemma4-26b-a4b"] + 15 * n["qwen36-35b-a3b"])
        self.assertEqual([r["seq"] for s in a for r in s], list(range(1, 376)))
        conc = [s for s in a if s[0]["condition"] == "concurrent"]
        self.assertTrue(all(len(s) == 2 for s in conc))
        with open(os.path.join(os.path.dirname(PROTOCOL), "schedule.json"), encoding="utf-8") as f:
            self.assertEqual(json.load(f), json.loads(json.dumps(R.schedule_doc(self.proto, self.sha, self.man, a))))

    def test_request_bytes_are_the_recorded_ones(self):
        p = self.prompts[0]
        self.assertEqual(R.sha256(R.body_of(p, {"seed": None})), p["requestSha256"])
        seeded = json.loads(R.body_of(p, {"seed": 0}))
        self.assertEqual(list(seeded)[-1], "seed")
        self.assertEqual({k: v for k, v in seeded.items() if k != "seed"}, p["request"])

    def test_run_records_every_request(self):
        post, calls = stub_post(['{"reason": "r", "actions": []}'])
        with tempfile.TemporaryDirectory() as d:
            doc = R.run(self.proto, self.sha, self.man, self.prompts, d, "http://stub/v1", post, stub_state())
            with open(os.path.join(d, "live", "requests.jsonl"), encoding="utf-8") as f:
                rows = [json.loads(l) for l in f]
        self.assertEqual((doc["requestsSent"], doc["stopped"]), (375, None))
        self.assertEqual(len(rows), 375)
        self.assertEqual(sorted(r["seq"] for r in rows), list(range(1, 376)))
        self.assertEqual(rows[0]["content"], '{"reason": "r", "actions": []}')
        self.assertTrue(all("stub" not in json.dumps(r) for r in rows))
        self.assertEqual(sum(1 for c in calls if "seed" in c), 3 * len(self.prompts))

    def test_stops_after_consecutive_failures(self):
        post, _ = stub_post([None])
        with tempfile.TemporaryDirectory() as d:
            doc = R.run(self.proto, self.sha, self.man, self.prompts, d, "http://stub/v1", post, stub_state())
        self.assertEqual((doc["requestsSent"], doc["stopped"]), (5, "5 consecutive transport failures"))

    def test_stops_when_unhealthy(self):
        post, calls = stub_post(["{}"])
        with tempfile.TemporaryDirectory() as d:
            doc = R.run(self.proto, self.sha, self.man, self.prompts, d, "http://stub/v1", post, stub_state(False))
        self.assertEqual((doc["requestsSent"], len(calls)), (0, 0))
        self.assertTrue(doc["stopped"].startswith("health"))


if __name__ == "__main__":
    unittest.main()
