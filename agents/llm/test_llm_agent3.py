"""Unit tests of llm-agent 3: it sends the model exactly what llm-agent 2
sends, submits the same decisions, and records every attempt
(dreamteam.llm-decision-record/v3), which its replay checks.

    python3 -B -m unittest discover -s agents/llm

The exchange with the harness and a mock model server is tested by
internal/bench/llm_agent3_test.go.
"""

import copy
import json
import os
import tempfile
import unittest

import llm_agent as A2
import llm_agent3 as A3

HERE = os.path.dirname(os.path.abspath(__file__))
OPS = {"type": "set_walkin_policy", "policy": "accept"}
STAFF = {"type": "set_reading_allocation", "op": "set", "employee": "E07", "effective": "2026-10-27",
         "siteAllocationBP": 10000}
STAFF_LESS = dict(STAFF, siteAllocationBP=6300)


def fixture():
    with open(os.path.join(HERE, "testdata", "v05a-point0.json"), encoding="utf-8") as f:
        return json.load(f)


def decision(reason, actions, **kw):
    return json.dumps({"reason": reason, "actions": actions, **kw})


class Harness:
    """The protocol side, from the V0.5a point-0 fixture; validate answers
    are scripted: "ok" accepts a batch, any other string refuses its first
    action with that code."""

    def __init__(self, fx, answers):
        self.fx, self.answers, self.log = fx, list(answers), []

    def request(self, op, args=None):
        self.log.append(op)
        fx = self.fx
        if op == "digests":
            return copy.deepcopy(fx["clock"])
        if op == "observe":
            out = {"observation": copy.deepcopy(fx["observation"])}
            if (args or {}).get("compact"):
                out["omitted"] = copy.deepcopy(fx["omitted"])
            return out
        if op == "actions":
            return copy.deepcopy(fx["catalog"])
        if op == "validate":
            code = self.answers.pop(0)
            acts = args["actions"]
            if code == "ok":
                return {"valid": True, "results": [{"ok": True, "action": a} for a in acts]}
            return {"valid": False, "results": [
                {"ok": False, "code": code, "message": f"{code}: commitment $19,726.02 over headroom $1,715.39",
                 "action": acts[0]}] + [{"ok": False, "code": "batch_rejected", "message": "another action was refused",
                                         "action": a} for a in acts[1:]]}
        raise AssertionError(op)


class Client:
    """The model: scripted replies (None: no answer, a timeout)."""

    def __init__(self, replies):
        self.replies, self.payloads = list(replies), []

    def payload(self, messages):
        return {"model": "m", "messages": messages}

    def call(self, payload):
        self.payloads.append(json.loads(json.dumps(payload)))
        r = self.replies.pop(0)
        if r is None:
            return None, "", "timeout: no answer within 1 s"
        return 200, json.dumps({"choices": [{"message": {"content": r}}], "model": "m"}), None


def settings(**kw):
    s = dict(A2.DEFAULTS)
    s.update({"model": "m", "endpoint": "http://localhost:9/v1", "systemPrompt": A2.SYSTEM_PROMPT, "maxRetries": 2})
    s.update(kw)
    return s


# (replies, validate answers, maxRetries)
CASES = {
    "refused_then_modified": ([
        "I think we should raise the allocation.",
        decision("Raise Dr. Avery to 100% and accept walk-ins.", [OPS, STAFF]),
        decision("Raise Dr. Avery to 63% instead.", [OPS, STAFF_LESS]),
    ], ["ok", "insufficient_funds", "ok", "ok"], 2),
    "details_then_none": ([
        decision("I need the allocation entry.", [], details=["set_reading_allocation", "worklist.recent"]),
        decision("Nothing needs to change.", []),
    ], [], 1),
    "timeout_then_accepted": ([None, decision("Accept walk-ins.", [OPS])], ["ok"], 1),
    "refused_twice": ([
        decision("Raise Dr. Avery to 100%.", [STAFF]),
        decision("Raise Dr. Avery to 100% again.", [STAFF]),
    ], ["insufficient_funds", "insufficient_funds"], 1),
}


def run(mod, case, points=2):
    """Decision points of one agent version on the fixture: [(batches,
    record)], the payloads sent to the model, and the harness."""
    replies, answers, retries = CASES[case]
    fx = fixture()
    s = settings(maxRetries=retries)
    client = Client(replies * points)
    h = Harness(fx, answers * points)
    mem = mod.Memory(s["memoryDecisions"], s["memoryMetricPoints"], s["memoryMaxChars"])
    out = []
    with tempfile.TemporaryDirectory() as d:
        ev = mod.Evidence(d)
        for p in range(points):
            batches, rec = mod.ask(h, ev, client, s, mem, p)
            mem.record(rec)
            out.append((batches, rec))
        with open(os.path.join(d, "memory.jsonl"), encoding="utf-8") as f:
            memory = f.read()
        attempts = None
        if os.path.exists(os.path.join(d, "attempts.jsonl")):
            with open(os.path.join(d, "attempts.jsonl"), encoding="utf-8") as f:
                attempts = [json.loads(l) for l in f]
    return out, client.payloads, memory, attempts


class SameAsVersion2(unittest.TestCase):
    """Version 3 changes the record only: the model is sent the same
    messages, the same decisions are submitted, and the memory is the same."""

    def test_shared_code_is_identical(self):
        for name in ("SYSTEM_PROMPT", "PROMPT_VERSION", "MEMORY_FORMAT", "DEFAULTS", "PROMPT_SETTINGS",
                     "SUMMARY_CUTS", "STAFFING"):
            self.assertEqual(getattr(A2, name), getattr(A3, name), name)
        self.assertEqual((A2.VERSION, A3.VERSION), ("2", "3"))

    def test_retry_text_is_version_2s(self):
        # llm-agent 2 builds it inline; the payloads of a retried point
        # carry it (refused_then_modified, below).
        _, p2, _, _ = run(A2, "refused_then_modified", 1)
        self.assertEqual(p2[2]["messages"][-1]["content"], A3.retry_text(
            "refused: action 1: insufficient_funds: insufficient_funds: commitment $19,726.02 over headroom $1,715.39"))

    def test_same_messages_decisions_and_memory(self):
        for case in CASES:
            with self.subTest(case):
                r2, p2, m2, _ = run(A2, case)
                r3, p3, m3, _ = run(A3, case)
                self.assertEqual(p2, p3)
                self.assertEqual(m2, m3)
                for (b2, x), (b3, y) in zip(r2, r3):
                    self.assertEqual(b2, b3)
                    for k in ("point", "time", "digests", "promptSha256", "memorySha256", "outcome", "reason",
                              "refusals", "errors", "batches", "promptSettings", "leftOut"):
                        self.assertEqual(x[k], y[k], k)
                    self.assertEqual(x["attempts"], len(y["attempts"]))


class Record(unittest.TestCase):
    def test_every_attempt(self):
        rec = run(A3, "refused_then_modified", 1)[0][0][1]
        self.assertEqual(rec["schema"], "dreamteam.llm-decision-record/v3")
        a1, a2, a3 = rec["attempts"]
        self.assertEqual([a["id"] for a in rec["attempts"]], ["0.1", "0.2", "0.3"])
        self.assertEqual([a["turn"] for a in rec["attempts"]], ["first", "retry", "retry"])
        # Attempt 1: model output kept, not parsed.
        self.assertEqual(a1["parse"]["code"], "invalid_json")
        self.assertEqual(a1["output"]["text"], "I think we should raise the allocation.")
        self.assertIsNone(a1["actions"])
        # Attempt 2: the refused proposal, with its own reason and actions.
        self.assertEqual(a2["actions"], [OPS, STAFF])
        self.assertEqual(a2["reason"], "Raise Dr. Avery to 100% and accept walk-ins.")
        self.assertEqual([v["answer"]["valid"] for v in a2["validation"]], [True, False])
        self.assertEqual([(r["action"], r["code"]) for r in a2["refusals"]], [(1, "insufficient_funds")])
        self.assertEqual(a2["input"]["kind"], "retry")
        self.assertEqual(a2["input"]["feedback"], a1["error"])
        self.assertEqual(a3["input"]["feedback"], a2["error"])
        self.assertTrue(a2["error"].startswith("refused: action 1: insufficient_funds"))
        # Attempt 3: used.
        self.assertEqual([a["used"] for a in rec["attempts"]], [False, False, True])
        self.assertEqual(rec["final"], {"attempt": 3, "outcome": "actions"})
        self.assertEqual(rec["batches"], [[OPS], [STAFF_LESS]])
        c = rec["completeness"]
        self.assertEqual((c["attemptsRecorded"], c["attempts"], c["outputsRecorded"], c["reasonsRecorded"],
                          c["actionsRecorded"], c["inputsVerifiable"], c["truncated"], c["stopped"]),
                         (True, 3, 3, 2, 2, True, [], None))

    def test_attempts_log_matches_record(self):
        out, _, _, log = run(A3, "refused_then_modified", 2)
        self.assertEqual(log, [a for _, rec in out for a in rec["attempts"]])

    def test_details_timeout_and_fallback(self):
        rec = run(A3, "details_then_none", 1)[0][0][1]
        self.assertEqual([a["turn"] for a in rec["attempts"]], ["first", "details"])
        self.assertEqual(rec["attempts"][0]["detailsRequested"], ["set_reading_allocation", "worklist.recent"])
        self.assertEqual(rec["attempts"][1]["input"]["kind"], "details")
        self.assertEqual(rec["final"], {"attempt": 2, "outcome": "no_action"})
        rec = run(A3, "timeout_then_accepted", 1)[0][0][1]
        a1, a2 = rec["attempts"]
        self.assertEqual((a1["transport"], a1["output"], a2["input"]["kind"]), ("timeout", None, "resend"))
        self.assertEqual(a2["input"]["sha256"], a1["input"]["sha256"])
        rec = run(A3, "refused_twice", 1)[0][0][1]
        self.assertEqual(rec["final"], {"attempt": None, "outcome": "fallback"})
        self.assertEqual(rec["batches"], [])
        self.assertEqual([a["actions"] for a in rec["attempts"]], [[STAFF], [STAFF]])

    def test_bounded(self):
        cut = []
        self.assertEqual(A3.bounded("x" * 10, 4, "w", cut), "xxxx")
        self.assertEqual(cut, ["w"])
        self.assertEqual(A3.bounded("xy", 4, "w", cut), "xy")


class ReplayChecks(unittest.TestCase):
    def setUp(self):
        out, payloads, _, _ = run(A3, "refused_then_modified", 1)
        self.rec = out[0][1]
        self.prompt = payloads[0]["messages"][1]["content"]
        self.fx = fixture()

    def verify(self, rec, answers=("ok", "insufficient_funds", "ok", "ok")):
        h = Harness(self.fx, answers)
        return A3.verify_attempts(h, rec, self.fx["observation"], self.fx["omitted"], self.fx["catalog"], self.prompt)

    def test_untouched_passes(self):
        self.assertEqual(self.verify(self.rec), {"inputs": 3, "validations": 4})

    def test_details_rebuilt(self):
        out, payloads, _, _ = run(A3, "details_then_none", 1)
        h = Harness(self.fx, [])
        got = A3.verify_attempts(h, out[0][1], self.fx["observation"], self.fx["omitted"], self.fx["catalog"],
                                 payloads[0]["messages"][1]["content"])
        self.assertEqual(got, {"inputs": 2, "validations": 0})

    def test_tampering_is_refused(self):
        def t(f, answers=("ok", "insufficient_funds", "ok", "ok")):
            rec = json.loads(json.dumps(self.rec))  # as read from the file
            f(rec)
            with self.assertRaises(A3.ReplayError):
                self.verify(rec, answers)
        t(lambda r: r["attempts"].pop(0))
        t(lambda r: r["attempts"][1]["input"].update(feedback="something else"))
        t(lambda r: r["attempts"][1]["validation"][1]["answer"]["results"][0].update(code="not_employed"))
        t(lambda r: r["attempts"][1]["validation"][1]["actions"][0].update(siteAllocationBP=9000))
        t(lambda r: r["final"].update(attempt=2))
        t(lambda r: r["batches"].pop())
        t(lambda r: r["attempts"][0]["input"].update(sha256="0" * 64))
        # The engine answers differently than when recorded.
        t(lambda r: None, answers=("ok", "ok", "ok", "ok"))

    def test_truncated_feedback_is_not_verifiable(self):
        rec = copy.deepcopy(self.rec)
        rec["attempts"][1]["input"]["feedbackTruncated"] = True
        self.assertEqual(self.verify(rec), {"inputs": 2, "validations": 4})


class Recordings(unittest.TestCase):
    def write(self, recs):
        fd, p = tempfile.mkstemp(suffix=".jsonl")
        with os.fdopen(fd, "w") as f:
            for r in recs:
                f.write(json.dumps(r) + "\n")
        self.addCleanup(os.remove, p)
        return p

    def test_only_version_3(self):
        rec = run(A3, "timeout_then_accepted", 1)[0][0][1]
        self.assertEqual(len(A3.load_recordings([self.write([rec])])[0][2]), 1)
        v2 = run(A2, "timeout_then_accepted", 1)[0][0][1]
        v05c = os.path.join(HERE, "..", "..", "docs", "benchmark", "v07c-dev", "stage-a",
                            "stable-credit-qwen36-35b-a3b-s41601-decisions.jsonl")
        for p in (self.write([v2]), v05c):
            with self.assertRaises(A3.SettingsError) as e:
                A3.load_recordings([p])
            self.assertIn("replay with agents/llm/llm_agent.py", str(e.exception))


if __name__ == "__main__":
    unittest.main()
