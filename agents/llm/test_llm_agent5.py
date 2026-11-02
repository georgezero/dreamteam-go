"""Unit tests of llm-agent 5, the validation-first agent
(llm-agent-workflow/v1): scripted model replies and scripted engine
answers only. No model and no engine.

    python3 -B -m unittest discover -s agents/llm

The exchange with the real engine and the mock model server is tested by
internal/bench/llm_agent5_test.go.
"""

import copy
import hashlib
import json
import os
import tempfile
import unittest

import llm_agent3 as A3
import llm_agent4 as A4
import llm_agent5 as A5
import test_llm_agent4 as T4

HERE = os.path.dirname(os.path.abspath(__file__))
OPS, STAFF, STAFF_LESS = T4.OPS, T4.STAFF, T4.STAFF_LESS
OPS_LATE = dict(OPS, policy="defer")
TELE = {"type": "contract_teleradiology", "op": "start", "plan": "standard-v1", "effective": "2026-11-03",
        "scope": "all", "windows": ["weeknights"], "requestId": "tr-1"}
FIRE = {"type": "terminate", "employee": "E03", "lastDay": "2026-11-02"}
decision = T4.decision


def results(actions, msg="accepted"):
    return [{"id": "", "batch": 0, "action": a, "ok": True, "code": "accepted", "message": msg} for a in actions]


def commitment_preview(actions):
    """A valid allocation's preview: a new salary obligation and the funding
    test's commitment (the engine's shape, abridged)."""
    return T4.preview_answer(actions) | {}


def no_effect_preview(actions):
    """A valid batch whose preview shows no direct effect: no commitment,
    a guaranteed line marked unchanged, nothing displaced or opened."""
    return {"preview": {"asOf": "2026-10-13T07:00:00", "ok": True, "results": results(actions, "unchanged"),
                        "guaranteed": [{"label": "Contract guarantee per full week · Riley (E03)", "cents": 180000,
                                        "when": "unchanged by the roster", "basis": "40 h × $36.00"}],
                        "funding": {"commitmentCents": 0, "headroomCents": 5000000, "funded": True, "basis": "b"}}}


def retainer_preview(actions):
    """A teleradiology start: a retainer to the earliest end (a retained
    obligation), with no commitment beyond it measured."""
    return {"preview": {"asOf": "2026-10-13T07:00:00", "ok": True, "results": results(actions),
                        "teleradiology": [{"op": "start", "plan": "standard-v1", "guaranteedCents": 309333}],
                        "guaranteed": [{"label": "Teleradiology retainer to the earliest end", "cents": 309333,
                                        "when": "accrued daily 2026-11-03 to 2026-12-02, paid on the 1st",
                                        "basis": "contract: minimum term and notice, even with no reads"}]}}


def termination_preview(actions):
    """A termination: severance, and booked appointments displaced."""
    return {"preview": {"asOf": "2026-10-13T07:00:00", "ok": True, "results": results(actions),
                        "terminations": [{"employee": "E03", "lastDay": "2026-11-02", "severanceCents": 144000}],
                        "displacements": [{"appointment": "A1", "reason": "unstaffed"},
                                          {"appointment": "A2", "reason": "unstaffed"}],
                        "guaranteed": [{"label": "Severance · Riley (E03)", "cents": 144000,
                                        "when": "recognized now", "basis": "two contracted weeks"}]}}


class Harness(T4.Harness):
    """llm-agent 4's scripted engine, with preview answers given per call:
    a function of the actions (an answer), or "fail", "big", "mutate" as
    there."""

    def request(self, op, args=None):
        if op == "preview" and self.previews and callable(self.previews[0]):
            self.log.append((op, copy.deepcopy((args or {}).get("actions"))))
            return self.previews.pop(0)(args["actions"])
        if op == "preview" and self.previews and self.previews[0] in ("big", "mutate"):
            kind = self.previews[0]
            self.previews[0] = "ok"
            ans = super().request(op, args)
            if kind == "big":
                ans["preview"]["forecast"] = [{"label": "x" * 100, "cents": 0}] * 2000
            else:
                self.mutated = True
            return ans
        return super().request(op, args)


def settings(**kw):
    return T4.settings(**kw)


C, N, R, T = commitment_preview, no_effect_preview, retainer_preview, termination_preview
# (replies, validate answers, preview answers, maxRetries)
CASES = {
    "operations_only": ([decision("Accept walk-ins.", [OPS])], ["ok"], [], 1),
    "not_eligible": ([decision("Move Riley's roster.", [STAFF])], ["ok"], [N], 1),
    "keep": ([decision("Raise Dr. Avery.", [STAFF]), decision("Keep it.", [STAFF])], ["ok", "ok"], [C], 1),
    "revise_after_preview": ([decision("Raise Dr. Avery to 100%.", [STAFF]),
                              decision("63% instead.", [STAFF_LESS])], ["ok", "ok"], [C], 1),
    "abandon_after_preview": ([decision("Raise Dr. Avery.", [STAFF]), decision("No change.", [])], ["ok"], [C], 1),
    "refused_corrected": ([decision("Raise Dr. Avery to 100%.", [STAFF]), decision("63%.", [STAFF_LESS])],
                          ["insufficient_funds", "ok"], [N], 1),
    "refused_corrected_then_previewed": ([decision("100%.", [STAFF]), decision("63%.", [STAFF_LESS]),
                                          decision("Keep 63%.", [STAFF_LESS])],
                                         ["insufficient_funds", "ok", "ok"], [C], 1),
    "refused_abandoned": ([decision("Raise Dr. Avery.", [STAFF]), decision("No change.", [])],
                          ["insufficient_funds"], [], 1),
    "repeated_invalid": ([decision("Raise Dr. Avery.", [STAFF]), decision("Raise Dr. Avery.", [STAFF])],
                         ["insufficient_funds", "insufficient_funds"], [], 1),
    "revised_fails": ([decision("Raise Dr. Avery.", [STAFF]), decision("100% after all.", [STAFF_LESS]),
                       decision("No change.", [])], ["ok", "insufficient_funds"], [C], 1),
    "retained_obligation": ([decision("Start teleradiology.", [TELE]), decision("Keep it.", [TELE])],
                            ["ok", "ok"], [R], 1),
    "displaced": ([decision("Terminate Riley.", [FIRE]), decision("Keep it.", [FIRE])], ["ok", "ok"], [T], 1),
    "mixed": ([decision("Walk-ins and Dr. Avery.", [OPS, STAFF]), decision("Keep both.", [STAFF, OPS])],
              ["ok", "ok", "ok", "ok"], [C], 1),
    "mixed_operations_refused": ([decision("Walk-ins and Dr. Avery.", [OPS_LATE, STAFF]),
                                  decision("Dr. Avery only.", [STAFF])], ["invalid_args", "ok", "ok"], [N], 1),
    "preview_error": ([decision("Raise Dr. Avery.", [STAFF])], ["ok"], ["fail"], 1),
    "too_large": ([decision("Raise Dr. Avery.", [STAFF])], ["ok"], ["big"], 1),
    "timeout": ([None, decision("Accept walk-ins.", [OPS])], ["ok"], [], 1),
    "invalid_output": (["I would raise Dr. Avery.", decision("Accept walk-ins.", [OPS])], ["ok"], [], 1),
    "budget": (["Let me think.", "Still thinking."], [], [], 1),
    "mutate": ([decision("Raise Dr. Avery.", [STAFF]), decision("Keep it.", [STAFF])], ["ok", "ok"], ["mutate"], 1),
}


def run(case, points=1, fx=None, mod=A5, **kw):
    replies, answers, previews, retries = CASES[case]
    fx = fx or T4.fixture()
    s = settings(maxRetries=retries, **kw)
    client = T4.Client(replies * points)
    h = Harness(fx, answers * points, previews * points)
    mem = A3.Memory(s["memoryDecisions"], s["memoryMetricPoints"], s["memoryMaxChars"])
    out, prev = [], None
    with tempfile.TemporaryDirectory() as d:
        ev = A3.Evidence(d)
        for p in range(points):
            batches, rec = mod.ask(h, ev, client, s, mem, p, prev) if mod is A5 else mod.ask(h, ev, client, s, mem, p)
            mem.record(rec)
            prev = rec
            out.append((batches, json.loads(json.dumps(rec))))
        with open(os.path.join(d, "attempts.jsonl"), encoding="utf-8") as f:
            attempts = [json.loads(l) for l in f]
        with open(os.path.join(d, "model.jsonl"), encoding="utf-8") as f:
            calls = [json.loads(l) for l in f]
    return out, client.payloads, attempts, calls, h


def rec_of(case, **kw):
    out, payloads, _, _, h = run(case, **kw)
    return out[0][0], out[0][1], payloads, h


def last_user(payloads, k):
    return payloads[k]["messages"][-1]["content"]


class Workflow(unittest.TestCase):
    def test_01_valid_without_preview(self):
        b, r, p, h = rec_of("operations_only")
        self.assertEqual((b, r["outcome"]), ([[OPS]], "actions"))
        self.assertNotIn("preview", [op for op, _ in h.log])
        self.assertEqual(len(p), 1)
        b, r, p, h = rec_of("not_eligible")
        self.assertEqual((b, r["outcome"], len(p)), ([[STAFF]], "actions", 1))
        pv = r["attempts"][0]["preview"]
        self.assertEqual((pv["status"], pv["eligibility"]["eligible"], pv["eligibility"]["reasons"], pv["message"]),
                         ("not_eligible", False, [], None))

    def test_02_07_valid_with_preview_kept(self):
        b, r, p, h = rec_of("keep")
        self.assertEqual(b, [[STAFF]])
        self.assertEqual([op for op, _ in h.log if op in ("validate", "preview")], ["validate", "preview", "validate"])
        pv = r["attempts"][0]["preview"]
        self.assertEqual((pv["status"], pv["eligibility"]["reasons"]), ("shown", ["commitment", "obligation"]))
        text = last_user(p, 1)
        self.assertEqual(text, A5.preview_text(pv["answer"], [0], False))
        self.assertIn(A3.compact(pv["answer"]), text)
        for s in ("passed validation", "Nothing has been submitted yet", "direct commitments and effects",
                  "do not establish that the decision is beneficial", "keep it", "revise it", "submit nothing"):
            self.assertIn(s, text)
        self.assertEqual((r["attempts"][1]["input"]["kind"], r["attempts"][1]["used"]), ("preview", True))
        self.assertEqual(r["stages"]["previewed"], {"attempt": "0.1", "status": "shown"})

    def test_03_refused_corrected(self):
        b, r, p, _ = rec_of("refused_corrected")
        self.assertEqual(b, [[STAFF_LESS]])
        at = r["attempts"]
        self.assertEqual((at[0]["used"], at[1]["input"]["kind"], at[1]["used"]), (False, "validation", True))
        text = last_user(p, 1)
        res = at[0]["validation"][0]["answer"]["results"][0]
        self.assertEqual(text, A5.feedback_text([{"action": 0, "result": res}]))
        for s in ("insufficient_funds", res["message"], A3.compact(STAFF), "Nothing has been submitted",
                  "\"actions\": [] to submit nothing"):
            self.assertIn(s, text)
        self.assertEqual(r["stages"]["refused"], ["0.1"])
        self.assertEqual(r["stages"]["validated"], ["0.2"])

    def test_03b_refused_corrected_then_previewed(self):
        b, r, p, _ = rec_of("refused_corrected_then_previewed")
        self.assertEqual(b, [[STAFF_LESS]])
        self.assertEqual([a["input"]["kind"] for a in r["attempts"]], ["prompt", "validation", "preview"])
        self.assertEqual(r["attempts"][1]["preview"]["status"], "shown")

    def test_04_refused_abandoned(self):
        b, r, _, _ = rec_of("refused_abandoned")
        self.assertEqual((b, r["outcome"]), ([], "no_action"))
        self.assertEqual(r["attempts"][1]["validation"], [])

    def test_05_19_repeated_invalid_exhausts_the_budget(self):
        b, r, p, _ = rec_of("repeated_invalid")
        self.assertEqual((b, r["outcome"], r["final"]["attempt"]), ([], "fallback", None))
        self.assertEqual(len(p), 2)  # 1 + maxRetries
        self.assertTrue(all(a["refusals"] for a in r["attempts"]))
        b, r, p, _ = rec_of("budget")
        self.assertEqual((b, r["outcome"], len(p)), ([], "fallback", 2))
        self.assertEqual([a["input"]["kind"] for a in r["attempts"]], ["prompt", "retry"])

    def test_06_revised_after_preview(self):
        b, r, _, h = rec_of("revise_after_preview")
        self.assertEqual(b, [[STAFF_LESS]])
        self.assertEqual([op for op, _ in h.log].count("preview"), 1)  # never previewed again
        self.assertIsNone(r["attempts"][1]["preview"])

    def test_08_previewed_then_abandoned(self):
        b, r, _, _ = rec_of("abandon_after_preview")
        self.assertEqual((b, r["outcome"]), ([], "no_action"))
        self.assertEqual(r["stages"]["submitted"], {"batches": 0, "actions": 0, "outcome": "no_action"})

    def test_09_revised_fails_validation(self):
        b, r, p, _ = rec_of("revised_fails")
        self.assertEqual((b, r["outcome"]), ([], "no_action"))
        self.assertEqual([a["input"]["kind"] for a in r["attempts"]], ["prompt", "preview", "validation"])
        self.assertTrue(r["attempts"][1]["refusals"])

    def test_10_insufficient_funds_message_is_the_engines(self):
        _, r, p, _ = rec_of("refused_abandoned")
        msg = r["attempts"][0]["validation"][0]["answer"]["results"][0]["message"]
        self.assertIn("$19,726.02", msg)
        self.assertIn(msg, last_user(p, 1))
        self.assertNotIn("PREVIEW", last_user(p, 1))

    def test_11_credit_supported_commitment(self):
        fx = T4.fixture()
        fin = fx["observation"].setdefault("finance", {})
        fin.setdefault("financing", {})["solvency"] = {"creditAvailableCents": 5000000}
        out, _, _, _, _ = run("keep", fx=fx)
        e = out[0][1]["attempts"][0]["preview"]["eligibility"]
        self.assertEqual(e["facts"]["commitmentCents"], 1972602)
        self.assertTrue(e["facts"]["creditSupported"])  # 1972602 > 171539 − 5000000
        self.assertIn("commitment", e["reasons"])
        _, r, _, _ = rec_of("keep")
        self.assertIsNone(r["attempts"][0]["preview"]["eligibility"]["facts"]["creditSupported"])

    def test_12_retained_obligation(self):
        b, r, _, _ = rec_of("retained_obligation")
        e = r["attempts"][0]["preview"]["eligibility"]
        self.assertEqual((e["eligible"], e["reasons"]), (True, ["obligation"]))
        self.assertEqual(b, [[TELE]])

    def test_13_displaced_bookings(self):
        _, r, _, _ = rec_of("displaced")
        e = r["attempts"][0]["preview"]["eligibility"]
        self.assertEqual(e["reasons"], ["obligation", "termination", "displacement"])
        self.assertEqual((e["facts"]["displacedBookings"], e["facts"]["terminations"]), (2, 1))

    def test_14_mixed_batches(self):
        b, r, p, h = rec_of("mixed")
        self.assertEqual(b, [[OPS], [STAFF]])  # operations first, then staffing: separate batches
        self.assertEqual([op for op, _ in h.log if op in ("validate", "preview")],
                         ["validate", "validate", "preview", "validate", "validate"])
        self.assertEqual(len(r["attempts"][0]["validation"]), 2)
        self.assertIn(A5.OPS_NOTE, last_user(p, 1))
        self.assertEqual(r["attempts"][0]["preview"]["actions"], [STAFF])
        b, r, p, _ = rec_of("mixed_operations_refused")
        self.assertEqual(b, [[STAFF]])
        items = A5.refused_items(r["attempts"][0]["validation"])
        self.assertEqual([x["action"] for x in items], [0])  # only the refused operations action
        self.assertEqual(r["attempts"][0]["validation"][1]["answer"]["valid"], True)

    def test_15_16_preview_unavailable_or_failing(self):
        for case, status in (("preview_error", "failed"), ("too_large", "too_large")):
            with self.subTest(case):
                b, r, p, _ = rec_of(case)
                pv = r["attempts"][0]["preview"]
                self.assertEqual((pv["status"], b, len(p)), (status, [[STAFF]], 1))
        _, r, _, _ = rec_of("preview_error")
        self.assertEqual(r["attempts"][0]["preview"]["error"]["code"], "invalid_args")

    def test_17_model_timeout(self):
        b, r, p, _ = rec_of("timeout")
        self.assertEqual(b, [[OPS]])
        self.assertEqual([a["input"]["kind"] for a in r["attempts"]], ["prompt", "resend"])
        self.assertEqual(p[0], p[1])

    def test_18_invalid_model_output(self):
        b, r, p, _ = rec_of("invalid_output")
        self.assertEqual(b, [[OPS]])
        self.assertEqual(r["attempts"][1]["input"]["kind"], "retry")
        self.assertEqual(last_user(p, 1), A3.retry_text(r["attempts"][0]["error"]))

    def test_24_no_state_mutation(self):
        for case in CASES:
            if case == "mutate":
                continue
            with self.subTest(case):
                _, r, _, _ = rec_of(case)
                self.assertTrue(r["digestsBeforeSubmit"]["unchanged"])
                for a in r["attempts"]:
                    if a.get("preview"):
                        self.assertTrue(a["preview"]["digestsUnchanged"])
        b, r, _, _ = rec_of("mutate")
        self.assertEqual((b, r["outcome"]), ([], "fallback"))
        self.assertFalse(r["digestsBeforeSubmit"]["unchanged"])
        self.assertTrue(any(e.startswith("state_changed") for e in r["errors"]))

    def test_refusal_never_becomes_accepted(self):
        for case in CASES:
            with self.subTest(case):
                b, r, _, _ = rec_of(case)
                used = [a for a in r["attempts"] if a["used"]]
                if b:
                    self.assertEqual(len(used), 1)
                    self.assertFalse(used[0]["refusals"])
                    self.assertTrue(all(v["answer"]["valid"] for v in used[0]["validation"]))
                    self.assertEqual(b, [x for x, _ in A3.split(used[0]["actions"])])


class Recording(unittest.TestCase):
    def test_20_completeness(self):
        for case in CASES:
            with self.subTest(case):
                out, payloads, attempts, calls, h = run(case)
                r = out[0][1]
                self.assertEqual(len(r["attempts"]), len(payloads))  # no discarded attempt
                self.assertEqual(attempts, r["attempts"])
                self.assertEqual(len(calls), len(payloads))
                self.assertEqual(r["completeness"]["validationsRecorded"],
                                 sum(1 for op, _ in h.log if op == "validate"))
                self.assertEqual(r["completeness"]["previewsRequested"], sum(1 for op, _ in h.log if op == "preview"))
                st = r["stages"]
                self.assertEqual(set(st), {"proposed", "validated", "refused", "previewed", "submitted"})
                for a in r["attempts"]:
                    if a.get("preview") and a["preview"]["answer"] is not None:
                        self.assertEqual(a["preview"]["answerSha256"], A3.sha256(A3.compact(a["preview"]["answer"])))
                self.assertEqual((r["schema"], r["workflowVersion"], r["workflowSha256"]),
                                 (A5.RECORD_SCHEMA, A5.WORKFLOW_VERSION, A5.WORKFLOW_SHA256))

    def test_execution_of_the_previous_decision(self):
        # Point 0 submits [[OPS], [STAFF]]; the harness applied the
        # operations batch and refused the staffing one (not logged).
        fx = T4.fixture()
        t = fx["clock"]["time"]
        fx["observation"]["actions"] = {"attempted": 1, "accepted": 1, "rejected": 0, "recent": [
            {"id": "X1", "batch": 7, "time": t, "action": OPS, "ok": True, "code": "accepted", "message": "ok"}]}
        out, _, _, _, _ = run("mixed", points=2, fx=fx)
        self.assertIsNone(out[0][1]["previousExecution"])
        ex = out[1][1]["previousExecution"]
        self.assertEqual(ex["point"], 0)
        self.assertEqual([(b["batch"], b["applied"], b["loggedBatch"]) for b in ex["batches"]],
                         [(0, True, 7), (1, False, None)])
        self.assertEqual(ex["unmatchedLogEntries"], 0)

    def test_23_deterministic(self):
        for case in CASES:
            with self.subTest(case):
                a, b = rec_of(case)[1], rec_of(case)[1]
                for x in (a, b):
                    for at in x["attempts"]:
                        at.pop("latencyMs", None)
                self.assertEqual(json.dumps(a, sort_keys=True), json.dumps(b, sort_keys=True))


class Replay(unittest.TestCase):
    """21. Model-free replay: every input rebuilt, every validate and
    preview asked again and compared."""

    def verify(self, case, rec=None, prompt=None, previews=None, answers=None):
        replies, ans, pv, _ = CASES[case]
        fx = T4.fixture()
        if rec is None:
            _, rec, payloads, _ = rec_of(case)
            prompt = payloads[0]["messages"][1]["content"]
        h = Harness(fx, ans if answers is None else answers, pv if previews is None else previews)
        return A5.verify_attempts(h, rec, fx["observation"], fx["omitted"], fx["catalog"], prompt, fx["clock"])

    def test_untouched_passes(self):
        for case in CASES:
            if case in ("mutate",):
                continue
            with self.subTest(case):
                got = self.verify(case)
                _, rec, _, h = rec_of(case)
                self.assertEqual(got["validations"], sum(1 for op, _ in h.log if op == "validate"))
                self.assertEqual(got["previews"], sum(1 for op, _ in h.log if op == "preview"))

    def tampered(self, case, f, **kw):
        _, rec, payloads, _ = rec_of(case)
        rec = json.loads(json.dumps(rec))
        f(rec)
        with self.assertRaises(A5.ReplayError):
            self.verify(case, rec, payloads[0]["messages"][1]["content"], **kw)

    def test_tampering_is_refused(self):
        a = lambda r, k: r["attempts"][k]  # noqa: E731
        self.tampered("keep", lambda r: a(r, 0).update(preview=None))
        self.tampered("keep", lambda r: a(r, 0)["preview"]["eligibility"].update(reasons=["capacity"]))
        self.tampered("keep", lambda r: a(r, 0)["preview"].update(status="not_eligible"))
        self.tampered("keep", lambda r: a(r, 1)["input"].update(sha256="0" * 64))
        self.tampered("refused_corrected", lambda r: a(r, 1)["input"].update(kind="retry"))
        self.tampered("refused_corrected", lambda r: a(r, 0).update(refusals=[]))
        self.tampered("refused_corrected", lambda r: a(r, 0)["validation"][0]["answer"].update(valid=True))
        self.tampered("refused_corrected", lambda r: r.update(batches=[[STAFF]]))
        self.tampered("repeated_invalid", lambda r: (a(r, 1).update(used=True), r["final"].update(attempt=2)))
        self.tampered("mixed", lambda r: a(r, 0)["validation"].pop(1))
        self.tampered("mixed", lambda r: r["stages"].update(refused=["0.1"]))
        self.tampered("keep", lambda r: r["attempts"].pop(1))
        # The engine answers differently than when recorded.
        self.tampered("keep", lambda r: None, previews=[N])
        self.tampered("refused_corrected", lambda r: None, answers=["ok", "ok"])

    def test_replay_point_checks_the_previous_execution(self):
        fx = T4.fixture()
        t = fx["clock"]["time"]
        fx["observation"]["actions"] = {"recent": [
            {"id": "X1", "batch": 7, "time": t, "action": OPS, "ok": True, "code": "accepted", "message": "ok"}]}
        out, _, _, _, _ = run("mixed", points=2, fx=copy.deepcopy(fx))
        recs = [r for _, r in out]
        with tempfile.TemporaryDirectory() as d:
            p = os.path.join(d, "decisions.jsonl")
            with open(p, "w", encoding="utf-8") as f:
                for r in recs:
                    f.write(json.dumps(r) + "\n")
            replayer = A3.Replayer(A5.load_recordings([p]))
            ev = A3.Evidence(os.path.join(d, "ev"))
            h = Harness(copy.deepcopy(fx), CASES["mixed"][1] * 2, CASES["mixed"][2] * 2)
            r0 = A5.replay_point(h, ev, replayer, 0)
            r1 = A5.replay_point(h, ev, replayer, 1, r0)
            self.assertEqual(r1["batches"], recs[1]["batches"])
            # A recorded execution that is not what the log shows is refused.
            recs[1]["previousExecution"]["batches"][1]["applied"] = True
            with open(p, "w", encoding="utf-8") as f:
                for r in recs:
                    f.write(json.dumps(r) + "\n")
            replayer = A3.Replayer(A5.load_recordings([p]))
            h = Harness(copy.deepcopy(fx), CASES["mixed"][1] * 2, CASES["mixed"][2] * 2)
            r0 = A5.replay_point(h, ev, replayer, 0)
            with self.assertRaisesRegex(A5.ReplayError, "previous point's execution"):
                A5.replay_point(h, ev, replayer, 1, r0)


class Compatibility(unittest.TestCase):
    """22. Historical compatibility: versions 3 and 4 loaded unchanged; each
    version replays only its own recordings."""

    def test_loads_versions_3_and_4_unchanged(self):
        self.assertIs(A5.A3, A3)
        self.assertIs(A5.A4, A4)
        for path, want in ((A5.BASE_PATH, "ef2c17a06749d7c287cda918f77ff0e518d79475048e87fd3404848c2fb0015c"),
                           (A5.PREVIEW_PATH, "4cfc6a6906556b0cda50e95fa39263089d56e848adcef9bd04342e7f6aff998f")):
            with open(path, "rb") as f:
                self.assertEqual(hashlib.sha256(f.read()).hexdigest(), want, path)
        self.assertEqual((A5.VERSION, A5.BASE_VERSION, A5.PROMPT_VERSION, A5.MEMORY_FORMAT),
                         ("5", "3", "llm-agent-prompt/v2", "llm-agent-memory/v1"))

    def test_first_message_is_version_3s(self):
        _, p3, _, _, _ = run("operations_only", mod=A3)
        for case in CASES:
            with self.subTest(case):
                _, p5, _, _, _ = run(case)
                self.assertEqual(p3[0], p5[0])

    def test_workflow_hash(self):
        self.assertEqual(A5.WORKFLOW_SHA256, A3.sha256("\n".join(
            [A5.WORKFLOW_VERSION, A5.FEEDBACK_TEMPLATE, A5.PREVIEW_TEMPLATE, A5.OPS_NOTE] +
            [f"{k}: {v}" for k, v in A5.ELIGIBILITY_RULE])))

    def write(self, d, recs):
        p = os.path.join(d, "decisions.jsonl")
        with open(p, "w", encoding="utf-8") as f:
            for r in recs:
                f.write(json.dumps(r) + "\n")
        return p

    def test_each_version_replays_its_own(self):
        _, r5, _, _ = rec_of("keep")
        out4, _, _, _, _ = T4.run(A4, "keep")
        r4 = out4[0][1]
        with tempfile.TemporaryDirectory() as d:
            p5 = self.write(d, [r5])
            self.assertEqual(len(A5.load_recordings([p5])), 1)
            with self.assertRaises(A4.SettingsError):
                A4.load_recordings([p5])
            d2 = os.path.join(d, "v4")
            os.makedirs(d2)
            p4 = self.write(d2, [r4])
            with self.assertRaisesRegex(A5.SettingsError, "version 4 recordings replay with"):
                A5.load_recordings([p4])


if __name__ == "__main__":
    unittest.main()
