"""Tests of the management-alternatives workflow (llm_agent_alternatives),
without an engine or a model: validation-first is stubbed, the engine's two
operations are scripted, and every message the workflow sends is checked.
The engine side is tested in internal/alternatives (Go) and, end to end, by
analysis/alternatives/test_v13d.py."""

import json
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import llm_agent_alternatives as ALT  # noqa: E402

A5 = ALT.A5


class Model:
    def __init__(self, replies):
        self.replies, self.payloads = list(replies), []

    def payload(self, messages):
        return {"messages": messages}

    def call(self, payload):
        self.payloads.append(payload)
        r = self.replies.pop(0)
        if r is None:
            return None, "", "timeout: no answer"
        return 200, json.dumps({"choices": [{"message": {"content": r}}]}), None


class Harness:
    def __init__(self, turn, answers, digests=("d0", "d0")):
        self.turn, self.answers, self.digests, self.log = turn, list(answers), list(digests), []

    def request(self, op, args=None):
        self.log.append((op, args))
        if op == "alternatives.turn":
            return self.turn
        if op == "alternatives.decide":
            return self.answers.pop(0)
        if op == "digests":
            return {"digests": self.digests.pop(0) if self.digests else "d0"}
        raise AssertionError(op)


ACTIONS = [{"type": "terminate", "employee": "E04", "lastDay": "2026-10-27"}]


def stub_vf(outcome="actions"):
    def ask(h, ev, client, settings, memory, point, prev=None, workflow=None):
        assert workflow == "validation-first"
        msgs = [{"role": "system", "content": "s"}, {"role": "user", "content": "p"}]
        client.payload(msgs)
        client.call({"messages": msgs})
        rec = {"outcome": outcome, "final": {"attempt": 1}, "attempts": [{"attempt": 1, "actions": ACTIONS}]}
        return ([ACTIONS] if outcome == "actions" else []), rec
    return ask


def answer(kind, text="", outcome=None, inspected=()):
    return {"kind": kind, "answer": text, "answerSha256": "x", "parse": {}, "inspected": list(inspected),
            "outcome": outcome}


TURN = {"due": True, "why": "due", "message": "TURN TEXT", "inspect": {}, "record": {}}


class Workflow(unittest.TestCase):
    def setUp(self):
        self.real = A5.ask

    def tearDown(self):
        A5.ask = self.real

    def run_ask(self, harness, replies, outcome="actions", arm="treatment"):
        A5.ask = stub_vf(outcome)
        first = json.dumps({"reason": "r", "actions": ACTIONS})
        m = Model([first] + list(replies))
        batches, rec = ALT.ask(harness, None, m, {}, None, 1, arm=arm)
        return batches, rec, m

    def test_identity(self):
        self.assertEqual(ALT.WORKFLOW_SHA256, ALT.sha256("\n".join(
            (ALT.WORKFLOW_VERSION, A5.VALIDATION_FIRST_SHA256, ALT.PRESENTATION, ALT.ARMS_ID) + ALT.RULES)))
        self.assertEqual(ALT.BASE_WORKFLOW, "validation-first")

    def test_not_due_keeps_validation_first(self):
        h = Harness(dict(TURN, due=False, why="not_due:rule"), [])
        batches, rec, m = self.run_ask(h, [])
        self.assertEqual((batches, rec["result"], len(m.payloads)), ([ACTIONS], "not_due:rule", 1))

    def test_no_validated_proposal_has_no_turn(self):
        h = Harness(TURN, [])
        batches, rec, m = self.run_ask(h, [], outcome="fallback")
        self.assertEqual((batches, rec["turn"], rec["result"]), ([], None, "not_due:validation_first_fallback"))
        self.assertFalse([op for op, _ in h.log if op.startswith("alternatives")])

    def test_messages_are_the_engines_verbatim(self):
        out = {"result": "selected", "valid": True, "batches": [[{"type": "terminate"}]]}
        h = Harness(TURN, [answer("inspect", "DETAIL TEXT", inspected=["adjusted"]), answer("decision", outcome=out)])
        batches, rec, m = self.run_ask(h, ['{"inspect":["adjusted"]}', '{"decision":"select","id":"adjusted"}'])
        sent = [p["messages"][-1]["content"] for p in m.payloads[1:]]
        self.assertEqual(sent, ["TURN TEXT", "DETAIL TEXT"])
        # The turn follows the conversation and the reply it is about.
        self.assertEqual([x["role"] for x in m.payloads[1]["messages"]], ["system", "user", "assistant", "user"])
        self.assertEqual((batches, rec["result"], rec["inspected"]), (out["batches"], "selected", ["adjusted"]))
        decide = [a for op, a in h.log if op == "alternatives.decide"]
        self.assertEqual([d["round"] for d in decide], [0, 1])
        self.assertEqual(decide[1]["inspected"], ["adjusted"])
        self.assertTrue(all(d["actions"] == ACTIONS and d["arm"] == "treatment" for d in decide))

    def test_retries_then_no_decision(self):
        h = Harness(TURN, [answer("unreadable", "RETRY"), answer("unreadable", "RETRY")])
        batches, rec, m = self.run_ask(h, ["x", "y"])
        self.assertEqual((batches, rec["result"], len(m.payloads)), ([], "no_decision", 3))

    def test_timeouts_count_as_failures(self):
        out = {"result": "retained", "valid": True, "batches": [ACTIONS]}
        h = Harness(TURN, [answer("decision", outcome=out)])
        batches, rec, m = self.run_ask(h, [None, '{"decision":"select","id":"proposal"}'])
        self.assertEqual((rec["result"], len(m.payloads), rec["exchanges"][0]["kind"]), ("retained", 3, "transport"))
        # The same messages again after a timeout.
        self.assertEqual(m.payloads[1]["messages"], m.payloads[2]["messages"])
        h = Harness(TURN, [])
        batches, rec, m = self.run_ask(h, [None, None])
        self.assertEqual((batches, rec["result"]), ([], "no_decision"))

    def test_second_inspect_is_a_failure(self):
        out = {"result": "retained", "valid": True, "batches": [ACTIONS]}
        h = Harness(TURN, [answer("inspect", "D", inspected=["no_action"]), answer("inspect_again", "ONCE"),
                           answer("decision", outcome=out)])
        batches, rec, m = self.run_ask(h, ["i", "i", "s"])
        self.assertEqual((rec["result"], len(m.payloads)), ("retained", 4))
        h = Harness(TURN, [answer("inspect", "D", inspected=["no_action"]), answer("inspect_again", "ONCE"),
                           answer("unreadable", "RETRY")])
        batches, rec, m = self.run_ask(h, ["i", "i", "x"])
        self.assertEqual((batches, rec["result"]), ([], "no_decision"))

    def test_invalid_outcome_submits_nothing(self):
        out = {"result": "invalid_alternative", "valid": False, "batches": []}
        h = Harness(TURN, [answer("decision", outcome=out)])
        batches, rec, m = self.run_ask(h, ['{"decision":"select","id":"x"}'])
        self.assertEqual(batches, [])

    def test_changed_state_submits_nothing(self):
        out = {"result": "retained", "valid": True, "batches": [ACTIONS]}
        h = Harness(TURN, [answer("decision", outcome=out)], digests=("d0", "d1"))
        batches, rec, m = self.run_ask(h, ['{"decision":"select","id":"proposal"}'])
        self.assertEqual((batches, rec["digestsBeforeSubmit"]["unchanged"]), ([], False))

    def test_unknown_arm(self):
        with self.assertRaises(ALT.A3.SettingsError):
            ALT.ask(None, None, None, {}, None, 1, arm="both")


if __name__ == "__main__":
    unittest.main()
