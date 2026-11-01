"""Tests of the shared-prefix prototype (shared_prefix.py): the two arms at
a branch point, with llm-agent 4's test harness (agents/llm/test_llm_agent4.py)
for the engine and scripted replies for the model.

    python3 -B -m unittest discover -s analysis/repeatability
"""

import copy
import json
import os
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, os.path.join(ROOT, "agents", "llm"))
import shared_prefix as SP  # noqa: E402
import test_llm_agent4 as T4  # noqa: E402  (its harness, fixture and scripted client)

A3, A4 = SP.A3, SP.A4
STAFF, STAFF_LESS, OPS = T4.STAFF, T4.STAFF_LESS, T4.OPS


def settings():
    return T4.settings(maxRetries=1)


def live_record(agent, replies, answers, previews=()):
    """A decision point recorded by the real agent (llm-agent 3 or 4) with
    scripted replies: (record, the model requests' messages, harness)."""
    fx = T4.fixture()
    h = T4.Harness(fx, answers, previews)
    c = T4.Client(replies)
    s = settings()
    mem = A3.Memory(s["memoryDecisions"], s["memoryMetricPoints"], s["memoryMaxChars"])
    with tempfile.TemporaryDirectory() as d:
        batches, rec = agent.ask(h, A3.Evidence(d), c, s, mem, 0)
    return json.loads(json.dumps(rec)), [p["messages"] for p in c.payloads], batches


def arm(name, rec, replies, answers, previews=(), outputs=None):
    """One arm of the prototype at the recorded point: (batches, trace,
    harness)."""
    fx = T4.fixture()
    h = T4.Harness(fx, answers, previews)
    pset = rec["promptSettings"]
    mem = A3.Memory(pset["memoryDecisions"], pset["memoryMetricPoints"], pset["memoryMaxChars"])
    clock = h.request("digests")
    obs, omitted, catalog, m, prompt, _ = A3.prepare(h, mem, 0, pset)
    b, t = SP.branch(h, name, rec, obs, omitted, catalog, prompt, clock, SP.Replies(replies), 1, outputs)
    return b, t, h


def scripted(content, answers=None):
    return {"source": "scripted", "from": None, "answers": answers, "content": content}


PROPOSAL = T4.decision("raise Avery", [OPS, STAFF])


class Arms(unittest.TestCase):
    def setUp(self):
        # The source: llm-agent 4 at the point, proposal kept after the preview.
        self.rec, self.sent, _ = live_record(A4, [PROPOSAL, PROPOSAL], ["ok", "ok"], ["ok"])

    def test_identical_initial_state_and_proposal(self):
        _, c, _ = arm("control", self.rec, [], ["ok", "ok"])
        _, t, _ = arm("treatment", self.rec, [scripted(PROPOSAL)], ["ok", "ok"], ["ok"])
        self.assertEqual(c["proposalSha256"], t["proposalSha256"])
        self.assertEqual(c["proposalSha256"], self.rec["attempts"][0]["output"]["sha256"])
        self.assertEqual(c["messages"][:2], t["messages"][:2])
        self.assertEqual(c["messages"][1]["sha256"], self.rec["promptSha256"])

    def test_control_validates_the_proposal(self):
        b, t, h = arm("control", self.rec, [], ["ok", "ok"])
        self.assertEqual(b, [[OPS], [STAFF]])
        self.assertIsNone(t["preview"])
        self.assertEqual([op for op, _ in h.log if op == "preview"], [])
        self.assertEqual((t["outcome"], t["final"], len(t["steps"])), ("actions", "kept", 1))

    def test_treatment_kept(self):
        b, t, h = arm("treatment", self.rec, [scripted(PROPOSAL)], ["ok", "ok"], ["ok"])
        self.assertEqual(b, [[OPS], [STAFF]])
        self.assertEqual((t["preview"]["status"], t["response"], t["final"]), ("ok", "kept", "kept"))
        self.assertTrue(t["preview"]["digestsUnchanged"])
        self.assertEqual([x[1] for x in h.log if x[0] == "preview"], [[STAFF]])

    def test_treatment_revised(self):
        b, t, _ = arm("treatment", self.rec, [scripted(T4.decision("less", [OPS, STAFF_LESS]))], ["ok", "ok"], ["ok"])
        self.assertEqual(b, [[OPS], [STAFF_LESS]])
        self.assertEqual((t["response"], t["final"]), ("revised", "revised"))

    def test_treatment_abandoned(self):
        b, t, _ = arm("treatment", self.rec, [scripted(T4.decision("none", []))], [], ["ok"])
        self.assertEqual(b, [])
        self.assertEqual((t["outcome"], t["response"], t["final"]), ("no_action", "abandoned", "abandoned"))

    def test_invalid_preview_validates_the_proposal_unchanged(self):
        b, t, _ = arm("treatment", self.rec, [], ["ok", "ok"], ["fail"])
        self.assertEqual(b, [[OPS], [STAFF]])
        self.assertEqual((t["preview"]["status"], t["preview"]["error"]["code"]), ("failed", "invalid_args"))
        self.assertEqual(len(t["steps"]), 1)

    def test_control_retry_after_refusal(self):
        b, t, _ = arm("control", self.rec, [scripted(T4.decision("ops only", [OPS]))],
                      ["ok", "insufficient_funds", "ok"])
        self.assertEqual(b, [[OPS]])
        self.assertTrue(t["steps"][0]["error"].startswith("refused: action 1: insufficient_funds"))
        self.assertEqual(t["steps"][1]["answers"], t["messages"][-1]["sha256"])
        self.assertEqual(t["final"], "revised")

    def test_fallback_after_the_retries(self):
        b, t, _ = arm("control", self.rec, [scripted("not json")], ["ok", "insufficient_funds"])
        self.assertEqual((b, t["outcome"], t["final"]), ([], "fallback", "fallback"))


class MatchesTheAgents(unittest.TestCase):
    """Given the same replies, each arm sends the messages the real agent
    sends and submits what it submits."""

    def check(self, agent, name, replies, answers, previews=()):
        rec, sent, batches = live_record(agent, replies, answers, previews)
        b, t, _ = arm(name, rec, [scripted(r) for r in replies[1:]], list(answers), list(previews))
        self.assertEqual(b, batches)
        self.assertEqual([m["sha256"] for m in t["messages"]], [A3.sha256(m["content"]) for m in sent[-1]])
        self.assertEqual(len(t["steps"]), len(replies))

    def test_control_is_llm_agent_3(self):
        self.check(A3, "control", [PROPOSAL, T4.decision("ops", [OPS])], ["ok", "insufficient_funds", "ok"])

    def test_treatment_is_llm_agent_4(self):
        self.check(A4, "treatment", [PROPOSAL, T4.decision("less", [OPS, STAFF_LESS])], ["ok", "ok"], ["ok"])

    def test_treatment_retry_is_llm_agent_4(self):
        self.check(A4, "treatment", [PROPOSAL, PROPOSAL, T4.decision("ops", [OPS])],
                   ["ok", "insufficient_funds", "ok"], ["ok"])


class MissingEvidence(unittest.TestCase):
    def setUp(self):
        self.rec, _, _ = live_record(A4, [PROPOSAL, PROPOSAL], ["ok", "ok"], ["ok"])

    def code(self, f):
        with self.assertRaises(SP.MissingEvidence) as cm:
            f()
        return cm.exception.code

    def test_truncated_proposal(self):
        rec = copy.deepcopy(self.rec)
        rec["attempts"][0]["output"]["text"] = rec["attempts"][0]["output"]["text"][:10]
        self.assertEqual(self.code(lambda: arm("control", rec, [], ["ok", "ok"])), "output_truncated")
        b, _, _ = arm("control", rec, [], ["ok", "ok"], outputs={"0.1": PROPOSAL})
        self.assertEqual(b, [[OPS], [STAFF]])
        self.assertEqual(self.code(lambda: arm("control", rec, [], ["ok", "ok"], outputs={"0.1": PROPOSAL + " "})),
                         "output_differs")

    def test_no_staffing_proposal(self):
        rec, _, _ = live_record(A4, [T4.decision("ops", [OPS])], ["ok"])
        self.assertEqual(self.code(lambda: arm("treatment", rec, [], ["ok"])), "no_staffing_proposal")

    def test_reply_for_another_message(self):
        self.assertEqual(self.code(lambda: arm("treatment", self.rec, [scripted(PROPOSAL, answers="0" * 64)],
                                               ["ok", "ok"], ["ok"])), "reply_not_for_this_message")

    def test_no_reply_left(self):
        self.assertEqual(self.code(lambda: arm("treatment", self.rec, [], ["ok", "ok"], ["ok"])), "no_reply")

    def test_changed_prompt(self):
        rec = copy.deepcopy(self.rec)
        rec["promptSha256"] = "0" * 64
        self.assertEqual(self.code(lambda: arm("control", rec, [], ["ok", "ok"])), "prompt_differs")


class Determinism(unittest.TestCase):
    def test_same_trace_twice(self):
        rec, _, _ = live_record(A4, [PROPOSAL, PROPOSAL], ["ok", "ok"], ["ok"])
        a = arm("treatment", rec, [scripted(PROPOSAL)], ["ok", "ok"], ["ok"])[1]
        b = arm("treatment", rec, [scripted(PROPOSAL)], ["ok", "ok"], ["ok"])[1]
        self.assertEqual(json.dumps(a, sort_keys=True), json.dumps(b, sort_keys=True))

    def test_committed_specs_are_rebuilt(self):
        d = os.path.join(ROOT, "docs", "benchmark", "v10a-dev", "prototype")
        with tempfile.TemporaryDirectory() as out:
            n, names = SP.specs(out, prior=d)
            got = sorted(names) + sorted("specs/" + f for f in os.listdir(os.path.join(out, "specs")))
            for name in got:
                with open(os.path.join(out, name), "rb") as f, open(os.path.join(d, name), "rb") as g:
                    self.assertEqual(f.read(), g.read(), name)
        self.assertEqual(n, 9)
        self.assertEqual(len(got), 12)

if __name__ == "__main__":
    unittest.main()
