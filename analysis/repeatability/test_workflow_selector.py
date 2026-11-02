"""llm-agent 5's workflows (V0.11.0) against the V0.11b experiment's
drivers (workflow_experiment.py, unchanged): --workflow validation-first
sends exactly the messages of arm B (v11b-B-validation-only) and submits
what it submits; validation-first-preview those of arm C. Scripted engine
and model only.

    python3 -B -m unittest discover -s analysis/repeatability
"""

import os
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(ROOT, "agents", "llm"))
import test_workflow_experiment as TW  # noqa: E402  (its scripted driver)

W, A3, A5, T4, T5 = TW.W, TW.A3, TW.A5, TW.T4, TW.T5
STAFF, STAFF_LESS, OPS = TW.STAFF, TW.STAFF_LESS, TW.OPS
d, C, N, R, T = TW.d, T5.C, T5.N, T5.R, T5.T
TELE, FIRE = T5.TELE, T5.FIRE


def agent(workflow, replies, answers, previews=(), retries=1):
    """llm-agent 5 at one decision point with the named workflow: the
    batches and the model requests' messages."""
    fx = T4.fixture()
    s = T4.settings(maxRetries=retries)
    client = T4.Client(list(replies))
    h = T5.Harness(fx, list(answers), list(previews))
    mem = A3.Memory(s["memoryDecisions"], s["memoryMetricPoints"], s["memoryMaxChars"])
    with tempfile.TemporaryDirectory() as dd:
        b, rec = A5.ask(h, A3.Evidence(dd), client, s, mem, 0, workflow=workflow)
    return b, [p["messages"] for p in client.payloads], rec


# (replies, validate answers, preview answers for the preview workflow)
CASES = [
    ([d("100%", [STAFF])], ["ok"], [C]),                                              # valid, eligible
    ([d("100%", [STAFF])], ["ok"], [N]),                                              # valid, not eligible
    ([d("ops", [OPS])], ["ok"], []),                                                  # operations only
    ([d("tele", [TELE])], ["ok"], [R]),                                               # retained obligation
    ([d("fire", [FIRE])], ["ok"], [T]),                                               # termination
    ([d("100%", [STAFF]), d("63%", [STAFF_LESS])], ["insufficient_funds", "ok"], [C]),  # refused, corrected
    ([d("100%", [STAFF]), d("none", [])], ["insufficient_funds"], []),                 # refused, abandoned
    ([d("100%", [STAFF]), d("100%", [STAFF])], ["insufficient_funds", "insufficient_funds"], []),
    ([d("100%", [STAFF]), "not json"], ["insufficient_funds"], []),                   # unreadable reply
    ([d("both", [OPS, STAFF]), d("ops", [OPS])], ["ok", "insufficient_funds", "ok"], []),  # mixed, refused
    ([d("both", [OPS, STAFF])], ["ok", "ok"], [C]),                                   # mixed, valid
]


class Equivalence(unittest.TestCase):
    def check(self, arm, workflow, replies, answers, previews, retries=1):
        b, msgs, rec = agent(workflow, replies, answers, previews, retries)
        t, sent = TW.driver(arm, msgs[0], replies, answers, previews if arm == "C" else (), retries)
        self.assertEqual(sent, msgs[1:])
        self.assertEqual(t["batches"], b)
        self.assertEqual(t["calls"], len(msgs) - 1)
        return t, rec

    def test_validation_first_is_arm_b(self):
        for replies, answers, previews in CASES:
            with self.subTest(answers=answers, first=replies[0]):
                t, rec = self.check("B", "validation-first", replies, answers, previews)
                self.assertFalse(any(a["preview"] for a in t["attempts"]))
                self.assertEqual(rec["workflowVersion"], A5.VALIDATION_FIRST_VERSION)

    def test_validation_first_preview_is_arm_c(self):
        for replies, answers, previews in CASES:
            replies = list(replies) + [replies[-1]]  # the answer to a shown preview: keep
            with self.subTest(answers=answers, first=replies[0]):
                t, rec = self.check("C", "validation-first-preview", replies, answers + ["ok"] * 2, previews)
                self.assertEqual(rec["workflowVersion"], A5.WORKFLOW_VERSION)

    def test_timeouts(self):
        for arm, wf in (("B", "validation-first"), ("C", "validation-first-preview")):
            with self.subTest(arm):
                self.check(arm, wf, [d("100%", [STAFF]), None, d("63%", [STAFF_LESS])], ["insufficient_funds", "ok"],
                           [N], retries=2)


if __name__ == "__main__":
    unittest.main()
