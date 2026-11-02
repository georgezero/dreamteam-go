"""Tests of the V0.11b workflow experiment (workflow_experiment.py,
shared-prefix/v4): each arm's driver against the real agents (llm-agent 3
for A, llm-agent 5 for B and C) on llm-agent 5's scripted engine and model;
the classification, the selection, the schedule and the replay checks.

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
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(ROOT, "agents", "llm"))
import workflow_experiment as W  # noqa: E402
import test_llm_agent5 as T5  # noqa: E402  (scripted engine, model and fixtures)

A3, A5 = W.A3, W.A5
T4 = T5.T4
STAFF, STAFF_LESS, OPS = T5.STAFF, T5.STAFF_LESS, T5.OPS
d = T5.decision
C, N = T5.C, T5.N


def real(mod, replies, answers, previews=(), retries=1):
    """The real agent at one decision point: the batches and the model
    requests' messages."""
    fx = T4.fixture()
    s = T4.settings(maxRetries=retries)
    client = T4.Client(list(replies))
    h = T5.Harness(fx, list(answers), list(previews))
    mem = A3.Memory(s["memoryDecisions"], s["memoryMetricPoints"], s["memoryMaxChars"])
    with tempfile.TemporaryDirectory() as dd:
        ev = A3.Evidence(dd)
        b, rec = mod.ask(h, ev, client, s, mem, 0)
    return b, [p["messages"] for p in client.payloads]


def driver(arm, first_messages, replies, answers, previews=(), retries=1):
    """The arm's driver from the proposal (replies[0]) on, with the rest of
    the replies: (trace, the messages it sent)."""
    fx = T4.fixture()
    h = T5.Harness(fx, list(answers), list(previews))
    env = W.Env(h, fx["observation"], fx["omitted"], fx["catalog"], fx["clock"], A3.DEFAULTS["maxPromptChars"])
    rest, sent = list(replies[1:]), []

    def model(messages):
        sent.append(json.loads(json.dumps(messages)))
        r = rest.pop(0)
        return ({"requestSha256": "x", "error": None}, r) if r is not None else \
            ({"requestSha256": "x", "error": "timeout: no answer"}, None)
    run = W.run_arm(arm, first_messages, replies[0], env, model, max_retries=retries)
    return run, sent


AGENT = {"A": A3, "B": A5, "C": A5}


class Fidelity(unittest.TestCase):
    """Given the same replies and engine answers, each arm sends the
    messages its agent sends and submits what it submits."""

    def check(self, arm, replies, answers, previews=(), real_previews=None, retries=1):
        b, msgs = real(AGENT[arm], replies, answers, previews if real_previews is None else real_previews, retries)
        t, sent = driver(arm, msgs[0], replies, answers, previews, retries)
        self.assertEqual(sent, msgs[1:])
        self.assertEqual(t["batches"], b)
        self.assertEqual(t["calls"], len(msgs) - 1)
        return t

    def test_a_is_llm_agent_3(self):
        cases = [([d("100%", [STAFF]), d("63%", [STAFF_LESS])], ["insufficient_funds", "ok"]),
                 ([d("100%", [STAFF]), d("none", [])], ["insufficient_funds"]),
                 ([d("100%", [STAFF]), d("100%", [STAFF])], ["insufficient_funds", "insufficient_funds"]),
                 ([d("100%", [STAFF]), "not json"], ["insufficient_funds"]),
                 ([d("100%", [STAFF])], ["ok"]),
                 ([d("both", [OPS, STAFF]), d("ops", [OPS])], ["ok", "insufficient_funds", "ok"])]
        for replies, answers in cases:
            with self.subTest(answers):
                self.check("A", replies, answers)

    def test_b_is_llm_agent_5_without_a_preview(self):
        # llm-agent 5 asks for a preview that is not eligible: the model sees
        # what B shows (B asks for none).
        cases = [([d("100%", [STAFF]), d("63%", [STAFF_LESS])], ["insufficient_funds", "ok"], [N]),
                 ([d("100%", [STAFF]), d("none", [])], ["insufficient_funds"], []),
                 ([d("100%", [STAFF]), d("100%", [STAFF])], ["insufficient_funds", "insufficient_funds"], []),
                 ([d("100%", [STAFF])], ["ok"], [N]),
                 ([d("ops", [OPS])], ["ok"], [])]
        for replies, answers, real_pv in cases:
            with self.subTest(answers):
                t = self.check("B", replies, answers, previews=(), real_previews=real_pv)
                self.assertFalse(any(a["preview"] for a in t["attempts"]))

    def test_c_is_llm_agent_5(self):
        cases = [([d("100%", [STAFF]), d("keep", [STAFF])], ["ok", "ok"], [C]),
                 ([d("100%", [STAFF]), d("63%", [STAFF_LESS])], ["ok", "ok"], [C]),
                 ([d("100%", [STAFF]), d("none", [])], ["ok"], [C]),
                 ([d("100%", [STAFF]), d("63%", [STAFF_LESS]), d("keep", [STAFF_LESS])],
                  ["insufficient_funds", "ok", "ok"], [C]),
                 ([d("100%", [STAFF]), d("63%", [STAFF_LESS]), d("none", [])], ["ok", "insufficient_funds"], [C]),
                 ([d("both", [OPS, STAFF]), d("both", [STAFF, OPS])], ["ok", "ok", "ok", "ok"], [C]),
                 ([d("100%", [STAFF])], ["ok"], [N]),
                 ([d("100%", [STAFF])], ["ok"], ["fail"])]
        for replies, answers, pv in cases:
            with self.subTest(answers):
                self.check("C", replies, answers, previews=pv)

    def test_timeouts_and_details(self):
        # A timeout resends the same messages, and counts against the budget.
        for arm in W.ARMS:
            with self.subTest(arm):
                self.check(arm, [d("100%", [STAFF]), None, d("63%", [STAFF_LESS])], ["insufficient_funds", "ok"],
                           previews=[N] if arm == "C" else (), real_previews=[N] if arm != "A" else (), retries=2)


class Workflows(unittest.TestCase):
    def test_valid_proposal(self):
        for arm, calls in (("A", 0), ("B", 0), ("C", 1)):
            with self.subTest(arm):
                t, sent = driver(arm, [{"role": "system", "content": "s"}, {"role": "user", "content": "p"}],
                                 [d("100%", [STAFF]), d("keep", [STAFF])], ["ok", "ok"], [C])
                self.assertEqual((t["calls"], t["batches"]), (calls, [[STAFF]]))
                self.assertEqual(W.classify(t, [STAFF]), ("kept", "accepted"))
                pv = W.preview_of(t)
                self.assertEqual(pv["response"] if pv else None, "kept" if arm == "C" else None)

    def test_classify(self):
        sys_ = [{"role": "system", "content": "s"}, {"role": "user", "content": "p"}]
        cases = [("A", [d("x", [STAFF]), d("y", [STAFF_LESS])], ["insufficient_funds", "ok"], ("revised", "accepted")),
                 ("B", [d("x", [STAFF]), d("y", [])], ["insufficient_funds"], ("abandoned", "no_action")),
                 ("B", [d("x", [STAFF]), d("x", [STAFF])], ["insufficient_funds", "insufficient_funds"], ("refused", "none")),
                 ("A", [d("x", [STAFF]), "garbage"], ["insufficient_funds"], ("invalid", "none"))]
        for arm, replies, answers, want in cases:
            with self.subTest(want):
                t, _ = driver(arm, sys_, replies, answers)
                self.assertEqual(W.classify(t, [STAFF]), want)
        t, _ = driver("A", sys_, [d("x", [STAFF]), "garbage"], ["insufficient_funds"])
        self.assertEqual(t["failures"], 2)


def row(model, validation, point, rec="r1", stake=0, credit=False, rev=None, types=("set_reading_allocation",),
        refusals=(), key=None, eligible=True):
    return {"model": model, "validation": validation, "point": point,
            "recording": f"x/staffing-shortage-credit-{rec}-s1-decisions.jsonl", "contextKey": key or f"{rec}{point}",
            "eligible": eligible, "shownPreview": False, "refusals": list(refusals), "displacedBookings": 0,
            "staffingTypes": list(types), "previewFunding": {"commitmentCents": stake},
            "proposal": {"commitmentCents": stake if validation == "accepted" else 0, "creditSupported": credit,
                         "reversibility": rev or {}, "directExposure": {}, "affordability": [], "outcome": "accepted"}}


class Selection(unittest.TestCase):
    def rows(self):
        G, Q = W.GEMMA, W.QWEN
        R = {"reversible_with_retained_obligation": 1}
        return [row(G, "accepted", 1, "g"), row(G, "refused", 2, "g", refusals=["hire:invalid_schema"]),
                row(Q, "accepted", 1, "a", stake=900, credit=True),
                row(Q, "accepted", 2, "b", stake=800, rev=R, types=("contract_teleradiology",)),
                row(Q, "accepted", 3, "c", stake=0, types=("terminate",)),
                row(Q, "accepted", 4, "d", stake=700),
                row(Q, "accepted", 5, "e", stake=600, types=("contract_teleradiology",)),
                row(Q, "accepted", 6, "e", stake=999, types=("contract_teleradiology",), eligible=False),
                row(Q, "refused", 1, "f", stake=50, refusals=["x:insufficient_funds"]),
                row(Q, "refused", 2, "g2", stake=40, refusals=["x:insufficient_funds"]),
                row(Q, "refused", 3, "h", refusals=["hire:lead_too_short"]),
                row(Q, "refused", 4, "i", refusals=["set_roster:not_monday"]),
                row(Q, "refused", 5, "j", refusals=["x:contract_overlap"])]

    def test_rule(self):
        got = W.select(self.rows())
        q = [(r["validation"], r["recording"].split("credit-")[1][0], r["category"]) for r in got if r["model"] == W.QWEN]
        self.assertEqual(q[:5], [("accepted", "a", "credit_supported"), ("accepted", "b", "retained_obligation"),
                                 ("accepted", "c", "termination"), ("accepted", "d", "allocation"),
                                 ("accepted", "e", "stake")])
        self.assertEqual([x[2] for x in q[5:]], ["insufficient_funds", "insufficient_funds", "timing", "timing", "other"])
        self.assertEqual(sum(1 for r in got if r["model"] == W.GEMMA), 2)
        self.assertNotIn(6, [r["point"] for r in got if r["model"] == W.QWEN and r["validation"] == "accepted"])

    def test_too_few(self):
        rows = [r for r in self.rows() if "timing" not in W.tags(r)]
        with self.assertRaisesRegex(W.ExperimentError, "timing"):
            W.select(rows)


class Schedule(unittest.TestCase):
    def test_reproducible_balanced(self):
        p = {"orderRule": W.ORDER_RULE, "modelOrder": [W.GEMMA, W.QWEN], "drawsPerArm": dict(W.DRAWS)}
        cs = [{"id": "g1", "model": W.GEMMA}, {"id": "q1", "model": W.QWEN}, {"id": "q2", "model": W.QWEN}]
        a, b = W.schedule(p, cs), W.schedule(p, copy.deepcopy(cs))
        self.assertEqual(a, b)
        self.assertEqual(len(a), 3 * 10 + 2 * 3 * 20)
        for dr in range(1, 21):
            self.assertEqual(sorted((r["point"], r["arm"]) for r in a if r["model"] == W.QWEN and r["draw"] == dr),
                             sorted((x, y) for x in ("q1", "q2") for y in W.ARMS))


class Replay(unittest.TestCase):
    def test_replay_model_checks_requests(self):
        e = {"id": "p", "request": {"model": "m", "messages": [], "temperature": 0, "max_tokens": 1,
                                    "response_format": {"type": "json_object"}}}
        msgs = [{"role": "user", "content": "a"}]
        sha = W.sha256(W.X.body_bytes(W.body_of(e, msgs)))
        ok = [{"requestSha256": sha, "body": "b", "responseSha256": W.sha256("b"), "content": "c", "error": None}]
        self.assertEqual(W.replay_model(e, ok)(msgs)[1], "c")
        with self.assertRaisesRegex(W.ExperimentError, "did not record"):
            W.replay_model(e, ok)([{"role": "user", "content": "other"}])
        bad = [dict(ok[0], body="changed")]
        with self.assertRaisesRegex(W.ExperimentError, "not the recorded one"):
            W.replay_model(e, bad)(msgs)
        with self.assertRaisesRegex(W.ExperimentError, "did not record"):
            W.replay_model(e, [])(msgs)

    def test_committed_points(self):
        dd = os.path.join(ROOT, "docs", "benchmark", "v11b-dev", "points")
        if not os.path.exists(os.path.join(dd, "manifest.json")):
            self.skipTest("no points committed yet")
        self.assertEqual(W.check_points(dd), len(W.load_points(dd)[1]))


if __name__ == "__main__":
    unittest.main()
