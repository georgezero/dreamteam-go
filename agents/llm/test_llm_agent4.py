"""Unit tests of llm-agent 4, the preview-enabled agent: it is version 3
until a proposal has a staffing action, then it shows the model the
simulator's preview of those actions, takes its reply as the decision, and
records all of it (dreamteam.llm-decision-record/v4), which its replay
checks.

    python3 -B -m unittest discover -s agents/llm

The exchange with the real engine and a mock model server is tested by
internal/bench/llm_agent4_test.go.
"""

import copy
import hashlib
import json
import os
import tempfile
import unittest

import llm_agent as A2
import llm_agent3 as A3
import llm_agent4 as A4

HERE = os.path.dirname(os.path.abspath(__file__))
OPS = {"type": "set_walkin_policy", "policy": "accept"}
STAFF = {"type": "set_reading_allocation", "op": "set", "employee": "E07", "effective": "2026-10-27",
         "siteAllocationBP": 10000}
STAFF_LESS = dict(STAFF, siteAllocationBP=6300)
BAD = {"type": "hire", "role": "x"}


def fixture():
    with open(os.path.join(HERE, "testdata", "v05a-point0.json"), encoding="utf-8") as f:
        return json.load(f)


def decision(reason, actions, **kw):
    return json.dumps({"reason": reason, "actions": actions, **kw})


def preview_answer(actions):
    """A preview answer of the shape the engine gives (abridged)."""
    return {"preview": {"asOf": "2026-10-13T07:00:00", "ok": True,
                        "results": [{"id": "", "batch": 0, "action": a, "ok": True, "code": "accepted",
                                     "message": "Allocation change: Dr. Avery (E07) from Tue Oct 27: site share 100%"}
                                    for a in actions],
                        "guaranteed": [{"label": "Salary allocation change · Dr. Avery (E07)", "cents": 1972602,
                                        "when": "per 30 days from 2026-10-27 until changed (14 days' notice)"}],
                        "funding": {"commitmentCents": 1972602, "headroomCents": 171539, "funded": False}}}


class Harness:
    """The protocol side, from the V0.5a point-0 fixture. validate answers
    are scripted ("ok" accepts a batch, any other string refuses its first
    action with that code); so are preview answers ("ok" answers, "fail"
    is a protocol error, "big" an answer too large to show, "mutate" an
    answer after which the digests differ)."""

    def __init__(self, fx, answers, previews=()):
        self.fx, self.answers, self.previews, self.log = fx, list(answers), list(previews), []
        self.mutated = False

    def request(self, op, args=None):
        self.log.append((op, copy.deepcopy((args or {}).get("actions"))))
        fx = self.fx
        if op == "digests":
            c = copy.deepcopy(fx["clock"])
            if self.mutated:
                c["digests"]["state"] = "0" * 64
            return c
        if op == "observe":
            out = {"observation": copy.deepcopy(fx["observation"])}
            if (args or {}).get("compact"):
                out["omitted"] = copy.deepcopy(fx["omitted"])
            return out
        if op == "actions":
            return copy.deepcopy(fx["catalog"])
        if op == "preview":
            p = self.previews.pop(0)
            if p == "fail":
                raise A3.ProtocolError("preview", {"code": "invalid_args", "message": "actions: not a batch"})
            ans = preview_answer(args["actions"])
            if p == "big":
                ans["preview"]["forecast"] = [{"label": "x" * 100, "cents": 0}] * 2000
            if p == "mutate":
                self.mutated = True
            return ans
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
    s = dict(A3.DEFAULTS)
    s.update({"model": "m", "endpoint": "http://localhost:9/v1", "systemPrompt": A3.SYSTEM_PROMPT, "maxRetries": 2})
    s.update(kw)
    return s


# (replies, validate answers, preview answers, maxRetries)
CASES = {
    "keep": ([decision("Raise Dr. Avery and accept walk-ins.", [OPS, STAFF]),
              decision("Keep it.", [STAFF, OPS])], ["ok", "ok"], ["ok"], 1),
    "revise": ([decision("Raise Dr. Avery to 100%.", [OPS, STAFF]),
                decision("Raise Dr. Avery to 63% instead.", [OPS, STAFF_LESS])], ["ok", "ok"], ["ok"], 1),
    "abandon": ([decision("Raise Dr. Avery to 100%.", [STAFF]),
                 decision("Not affordable; no change.", [])], [], ["ok"], 1),
    "drop_staffing": ([decision("Raise Dr. Avery and accept walk-ins.", [OPS, STAFF]),
                       decision("Only walk-ins.", [OPS])], ["ok"], ["ok"], 1),
    "refused_after_preview": ([decision("Raise Dr. Avery to 100%.", [STAFF]),
                               decision("Keep it.", [STAFF]),
                               decision("Raise Dr. Avery to 63% instead.", [STAFF_LESS])],
                              ["insufficient_funds", "ok"], ["ok"], 1),
    "multiple": (["I think we should raise the allocation.",
                  decision("Raise Dr. Avery to 100%.", [STAFF]),
                  "Let me reconsider.",
                  decision("Raise Dr. Avery to 63%.", [STAFF_LESS])], ["ok"], ["ok"], 2),
    "preview_fails": ([decision("Hire.", [BAD]), decision("No change.", [])], ["invalid_schema"], ["fail"], 1),
    "too_large": ([decision("Raise Dr. Avery to 100%.", [STAFF])], ["ok"], ["big"], 1),
    "operations_only": ([decision("Accept walk-ins.", [OPS])], ["ok"], [], 1),
    "details_then_preview": ([decision("I need the allocation entry.", [], details=["set_reading_allocation"]),
                              decision("Raise Dr. Avery to 100%.", [STAFF]),
                              decision("Keep it.", [STAFF])], ["ok"], ["ok"], 1),
    "timeout_after_preview": ([decision("Raise Dr. Avery to 100%.", [STAFF]), None, decision("Keep it.", [STAFF])],
                              ["ok"], ["ok"], 1),
    "mutate": ([decision("Raise Dr. Avery to 100%.", [STAFF]), decision("Keep it.", [STAFF])], ["ok"], ["mutate"], 1),
}


def run(mod, case, points=1, **kw):
    """Decision points of one agent version on the fixture: [(batches,
    record)], the payloads sent to the model, the memory log, the attempt
    log and the harness."""
    replies, answers, previews, retries = CASES[case]
    fx = fixture()
    s = settings(maxRetries=retries, **kw)
    client = Client(replies * points)
    h = Harness(fx, answers * points, previews * points)
    mem = A3.Memory(s["memoryDecisions"], s["memoryMetricPoints"], s["memoryMaxChars"])
    out = []
    with tempfile.TemporaryDirectory() as d:
        ev = A3.Evidence(d)
        for p in range(points):
            batches, rec = mod.ask(h, ev, client, s, mem, p)
            mem.record(rec)
            out.append((batches, json.loads(json.dumps(rec))))
        with open(os.path.join(d, "memory.jsonl"), encoding="utf-8") as f:
            memory = f.read()
        with open(os.path.join(d, "attempts.jsonl"), encoding="utf-8") as f:
            attempts = [json.loads(l) for l in f]
    return out, client.payloads, memory, attempts, h


def rec_of(case, **kw):
    out, payloads, _, _, h = run(A4, case, **kw)
    return out[0][1], payloads, h


class Base(unittest.TestCase):
    """14. Historical compatibility: version 3 is loaded unchanged, and
    the model's first message at a decision point is version 3's."""

    def test_base_is_llm_agent3_unchanged(self):
        self.assertIs(A4.A3, A3)
        self.assertEqual(A4.BASE_PATH, os.path.join(HERE, "llm_agent3.py"))
        with open(A4.BASE_PATH, "rb") as f:
            self.assertEqual(hashlib.sha256(f.read()).hexdigest(),
                             "ef2c17a06749d7c287cda918f77ff0e518d79475048e87fd3404848c2fb0015c")
        for name in ("SYSTEM_PROMPT", "PROMPT_VERSION", "MEMORY_FORMAT", "DEFAULTS", "PROMPT_SETTINGS"):
            self.assertEqual(getattr(A2, name), getattr(A3, name), name)
        self.assertEqual((A4.VERSION, A4.BASE_VERSION, A4.PROMPT_VERSION, A4.TREATMENT_VERSION),
                         ("4", "3", "llm-agent-prompt/v2", "llm-agent-preview/v1"))

    def test_treatment_hash(self):
        self.assertEqual(A4.TREATMENT_SHA256,
                         hashlib.sha256((A4.REVIEW_TEMPLATE + "\n" + A4.OPS_NOTE).encode()).hexdigest())

    def test_first_message_is_version_3s(self):
        for case in CASES:
            with self.subTest(case):
                _, p3, _, _, _ = run(A3, "keep")
                _, p4, _, _, _ = run(A4, case)
                self.assertEqual(p3[0], p4[0])

    def test_without_staffing_it_is_version_3(self):
        # No proposal with a staffing action: the same messages, decisions,
        # memory and attempts as version 3.
        for case4 in ("operations_only",):
            fx = fixture()
            outs = []
            for mod in (A3, A4):
                replies, answers, previews, retries = CASES[case4]
                client = Client(replies * 2)
                h = Harness(fx, answers * 2, previews * 2)
                s = settings(maxRetries=retries)
                mem = A3.Memory(s["memoryDecisions"], s["memoryMetricPoints"], s["memoryMaxChars"])
                recs = []
                with tempfile.TemporaryDirectory() as d:
                    ev = A3.Evidence(d)
                    for p in range(2):
                        b, r = mod.ask(h, ev, client, s, mem, p)
                        mem.record(r)
                        recs.append((b, json.loads(json.dumps(r))))
                    with open(os.path.join(d, "memory.jsonl"), encoding="utf-8") as f:
                        memory = f.read()
                outs.append((recs, client.payloads, memory, [op for op, _ in h.log]))
            (r3, p3, m3, ops3), (r4, p4, m4, ops4) = outs
            self.assertEqual((p3, m3, ops3), (p4, m4, ops4))
            for (b3, x), (b4, y) in zip(r3, r4):
                self.assertEqual(b3, b4)
                for k in ("point", "time", "digests", "promptSha256", "memorySha256", "outcome", "reason",
                          "refusals", "errors", "batches", "final"):
                    self.assertEqual(x[k], y[k], k)
                self.assertEqual([dict(a, preview=None) for a in x["attempts"]], y["attempts"])
                self.assertEqual(y["preview"]["status"], "not_applicable")


class PreviewStep(unittest.TestCase):
    def test_requested_correctly(self):
        # 1. The staffing actions only, before anything is validated; once.
        rec, _, h = rec_of("keep")
        ops = [op for op, _ in h.log]
        self.assertEqual(ops[:4], ["digests", "observe", "actions", "preview"])
        self.assertEqual(h.log[3][1], [STAFF])
        self.assertEqual(ops[4], "digests")
        self.assertEqual(ops.count("preview"), 1)
        self.assertEqual(ops[5:], ["validate", "validate"])
        pv = rec["attempts"][0]["preview"]
        self.assertEqual((pv["actions"], pv["indices"], pv["status"]), ([STAFF], [1], "ok"))
        self.assertIsNone(rec["attempts"][0]["validation"])
        # A proposal with no staffing action has no preview.
        rec, _, h = rec_of("operations_only")
        self.assertNotIn("preview", [op for op, _ in h.log])
        self.assertEqual(rec["preview"]["status"], "not_applicable")

    def test_response_preserved(self):
        # 2. The answer exactly as the harness gave it, its SHA-256, and the
        # message that showed it.
        rec, payloads, _ = rec_of("keep")
        pv = rec["attempts"][0]["preview"]
        want = preview_answer([STAFF])
        self.assertEqual(pv["answer"], want)
        self.assertEqual(pv["answerSha256"], A3.sha256(A3.compact(want)))
        shown = payloads[1]["messages"][-1]["content"]
        self.assertIn(A3.compact(want), shown)
        self.assertEqual(shown, A4.review_text(want, [1], True))
        self.assertTrue(shown.startswith("PREVIEW. Nothing has been submitted yet."))
        self.assertIn(A4.OPS_NOTE, shown)
        self.assertEqual(rec["attempts"][1]["input"], {"kind": "preview", "sha256": A3.sha256(shown),
                                                       "chars": len(shown)})
        self.assertEqual(pv["message"], {"sha256": A3.sha256(shown), "chars": len(shown)})
        self.assertEqual(payloads[1]["messages"][-2], {"role": "assistant",
                                                       "content": CASES["keep"][0][0]})
        self.assertNotIn(A4.OPS_NOTE, A4.review_text(want, [0], False))

    def test_keep(self):
        # 3. The same actions (in another order) are kept and submitted.
        rec, _, h = rec_of("keep")
        self.assertEqual(rec["preview"], {
            "version": "llm-agent-preview/v1", "status": "ok", "proposalAttempt": 1, "reviewAttempt": 2,
            "response": "kept", "responseAttempt": 2, "staffing": "kept", "operations": "kept", "final": "kept"})
        self.assertEqual([a["turn"] for a in rec["attempts"]], ["first", "review"])
        self.assertEqual([a["used"] for a in rec["attempts"]], [False, True])
        self.assertEqual(rec["batches"], [[OPS], [STAFF]])
        self.assertEqual(rec["final"], {"attempt": 2, "outcome": "actions"})

    def test_revise(self):
        # 4 and 6. The revised batch is the one validated and submitted.
        rec, _, h = rec_of("revise")
        p = rec["preview"]
        self.assertEqual((p["response"], p["staffing"], p["operations"], p["final"]),
                         ("revised", "revised", "kept", "revised"))
        self.assertEqual([a for op, a in h.log if op == "validate"], [[OPS], [STAFF_LESS]])
        self.assertEqual(rec["attempts"][1]["validation"][1]["actions"], [STAFF_LESS])
        self.assertEqual(rec["batches"], [[OPS], [STAFF_LESS]])
        rec, _, _ = rec_of("drop_staffing")
        p = rec["preview"]
        self.assertEqual((p["response"], p["staffing"], p["operations"]), ("revised", "dropped", "kept"))
        self.assertEqual(rec["batches"], [[OPS]])

    def test_abandon(self):
        # 5. "actions": [] submits nothing, and nothing is validated.
        rec, _, h = rec_of("abandon")
        self.assertEqual((rec["preview"]["response"], rec["preview"]["final"]), ("abandoned", "abandoned"))
        self.assertEqual((rec["outcome"], rec["batches"]), ("no_action", []))
        self.assertNotIn("validate", [op for op, _ in h.log])

    def test_refused_final_action(self):
        # 7. The reply after the preview is refused by validate: the retry
        # has version 3's feedback, and no second preview.
        rec, payloads, h = rec_of("refused_after_preview")
        a1, a2, a3 = rec["attempts"]
        self.assertEqual([a["turn"] for a in rec["attempts"]], ["first", "review", "retry"])
        self.assertEqual([r["code"] for r in a2["refusals"]], ["insufficient_funds"])
        self.assertEqual(a3["input"]["feedback"], a2["error"])
        self.assertEqual(payloads[2]["messages"][-1]["content"], A3.retry_text(a2["error"]))
        self.assertEqual([op for op, _ in h.log].count("preview"), 1)
        p = rec["preview"]
        self.assertEqual((p["response"], p["responseAttempt"], p["final"]), ("kept", 2, "revised"))
        self.assertEqual(rec["batches"], [[STAFF_LESS]])
        self.assertEqual(rec["refusals"], a2["refusals"])

    def test_multiple_attempts_and_malformed_output(self):
        # 8 and 9. Unparsed replies before and after the preview: one
        # preview, the response is the first parsed reply after it.
        rec, _, h = rec_of("multiple")
        self.assertEqual([a["id"] for a in rec["attempts"]], ["0.1", "0.2", "0.3", "0.4"])
        self.assertEqual([a["turn"] for a in rec["attempts"]], ["first", "retry", "review", "retry"])
        self.assertEqual(rec["attempts"][0]["parse"]["code"], "invalid_json")
        self.assertEqual(rec["attempts"][2]["parse"]["code"], "invalid_json")
        self.assertEqual(rec["attempts"][1]["preview"]["status"], "ok")
        self.assertEqual([op for op, _ in h.log].count("preview"), 1)
        p = rec["preview"]
        self.assertEqual((p["proposalAttempt"], p["reviewAttempt"], p["responseAttempt"], p["response"]),
                         (2, 3, 4, "revised"))
        self.assertEqual(rec["final"], {"attempt": 4, "outcome": "actions"})

    def test_preview_failure(self):
        # 10. A protocol error: recorded, not shown; the proposal goes to
        # validate as version 3's would.
        rec, payloads, h = rec_of("preview_fails")
        a1 = rec["attempts"][0]
        self.assertEqual(a1["preview"]["status"], "failed")
        self.assertEqual(a1["preview"]["error"]["code"], "invalid_args")
        self.assertIsNone(a1["preview"]["answer"])
        self.assertEqual([op for op, _ in h.log][3:6], ["preview", "digests", "validate"])
        self.assertEqual(rec["attempts"][1]["turn"], "retry")
        self.assertNotIn("PREVIEW", payloads[1]["messages"][-1]["content"])
        self.assertEqual(rec["preview"]["status"], "failed")
        self.assertIsNone(rec["preview"]["response"])
        # Too large to show: the same.
        rec, payloads, _ = rec_of("too_large")
        pv = rec["attempts"][0]["preview"]
        self.assertEqual((pv["status"], rec["preview"]["status"], len(payloads)), ("too_large", "too_large", 1))
        self.assertGreater(pv["message"]["chars"], A3.DEFAULTS["maxPromptChars"])
        self.assertIsNotNone(pv["answer"])
        self.assertEqual(rec["batches"], [[STAFF]])

    def test_details_and_timeout(self):
        rec, _, _ = rec_of("details_then_preview")
        self.assertEqual([a["turn"] for a in rec["attempts"]], ["first", "details", "review"])
        self.assertEqual(rec["preview"]["proposalAttempt"], 2)
        rec, _, _ = rec_of("timeout_after_preview")
        self.assertEqual([a["turn"] for a in rec["attempts"]], ["first", "review", "retry"])
        self.assertEqual(rec["attempts"][2]["input"]["kind"], "resend")
        self.assertEqual(rec["attempts"][2]["input"]["sha256"], rec["attempts"][1]["input"]["sha256"])
        self.assertEqual((rec["preview"]["response"], rec["preview"]["responseAttempt"]), ("kept", 3))

    def test_no_state_change_recorded(self):
        # 16. The digests after the preview are the point's (the engine's
        # own check is internal/bench/llm_agent4_test.go).
        self.assertTrue(rec_of("keep")[0]["attempts"][0]["preview"]["digestsUnchanged"])
        self.assertFalse(rec_of("mutate")[0]["attempts"][0]["preview"]["digestsUnchanged"])


class Recording(unittest.TestCase):
    def test_completeness(self):
        # 12. Every attempt and preview recorded.
        out, _, _, log, _ = run(A4, "refused_after_preview", 2)
        self.assertEqual(log, [a for _, rec in out for a in rec["attempts"]])
        c = out[0][1]["completeness"]
        self.assertEqual((c["attemptsRecorded"], c["attempts"], c["outputsRecorded"], c["previewsRequested"],
                          c["previewsRecorded"], c["inputsVerifiable"]), (True, 3, 3, 1, 1, True))
        rec = out[0][1]
        self.assertEqual((rec["schema"], rec["agentVersion"], rec["baseAgentVersion"], rec["treatmentVersion"],
                          rec["treatmentSha256"]),
                         ("dreamteam.llm-decision-record/v4", "4", "3", "llm-agent-preview/v1", A4.TREATMENT_SHA256))
        # Model output, parsed actions, preview, validation and the
        # decision submitted are distinct.
        a1, a2, _ = rec["attempts"]
        self.assertEqual((a1["output"]["text"], a1["actions"], a1["preview"]["answer"], a1["validation"]),
                         (CASES["refused_after_preview"][0][0], [STAFF], preview_answer([STAFF]), None))
        self.assertEqual(a2["validation"][0]["actions"], [STAFF])

    def test_deterministic(self):
        # 15. The same inputs give the same records and logs, byte for byte.
        for case in CASES:
            with self.subTest(case):
                x, y = run(A4, case, 2), run(A4, case, 2)
                self.assertEqual(json.dumps(x[0], sort_keys=True), json.dumps(y[0], sort_keys=True))
                self.assertEqual(x[3], y[3])
                ids = [a["id"] for _, r in x[0] for a in r["attempts"]]
                self.assertEqual(ids, sorted(ids, key=lambda i: tuple(int(n) for n in i.split("."))))


class Replay(unittest.TestCase):
    """13. Replay without a model, and 11. missing preview evidence."""

    def verify(self, case, rec=None, prompt=None):
        replies, answers, previews, _ = CASES[case]
        fx = fixture()
        if rec is None:
            rec, payloads, _ = rec_of(case)
            prompt = payloads[0]["messages"][1]["content"]
        h = Harness(fx, answers, previews)
        return A4.verify_attempts(h, rec, fx["observation"], fx["omitted"], fx["catalog"], prompt, fx["clock"]), h

    def test_untouched_passes(self):
        want = {"keep": (2, 2, 1), "revise": (2, 2, 1), "abandon": (2, 0, 1), "refused_after_preview": (3, 2, 1),
                "multiple": (4, 1, 1), "preview_fails": (2, 1, 1), "too_large": (1, 1, 1), "operations_only": (1, 1, 0),
                "details_then_preview": (3, 1, 1), "timeout_after_preview": (2, 1, 1), "mutate": (2, 1, 1)}
        for case, (i, v, p) in want.items():
            with self.subTest(case):
                got, h = self.verify(case)
                self.assertEqual(got, {"inputs": i, "validations": v, "previews": p})
                self.assertEqual(h.previews, [])  # every preview asked again

    def tampered(self, case, f, previews=None):
        rec, payloads, _ = rec_of(case)
        rec = json.loads(json.dumps(rec))
        f(rec)
        replies, answers, pv, _ = CASES[case]
        fx = fixture()
        h = Harness(fx, answers, pv if previews is None else previews)
        with self.assertRaises(A4.ReplayError):
            A4.verify_attempts(h, rec, fx["observation"], fx["omitted"], fx["catalog"],
                               payloads[0]["messages"][1]["content"], fx["clock"])

    def test_missing_or_changed_preview_evidence_is_refused(self):
        a = lambda r, k: r["attempts"][k]  # noqa: E731
        self.tampered("keep", lambda r: a(r, 0).update(preview=None))
        self.tampered("keep", lambda r: a(r, 0)["preview"].update(answer=None))
        self.tampered("keep", lambda r: a(r, 0)["preview"]["answer"]["preview"]["funding"].update(funded=True))
        self.tampered("keep", lambda r: a(r, 0)["preview"].update(answerSha256="0" * 64))
        self.tampered("keep", lambda r: a(r, 0)["preview"].update(indices=[0]))
        self.tampered("keep", lambda r: a(r, 1)["input"].update(sha256="0" * 64))
        self.tampered("keep", lambda r: a(r, 1)["input"].update(kind="retry"))
        self.tampered("keep", lambda r: r["preview"].update(response="revised"))
        self.tampered("keep", lambda r: r["attempts"].pop(1))
        self.tampered("operations_only", lambda r: a(r, 0).update(preview=a(r, 0)["preview"] or {"status": "ok"}))
        self.tampered("preview_fails", lambda r: a(r, 0)["preview"]["error"].update(code="not_previewable"))
        # The engine answers the preview differently than when recorded.
        self.tampered("keep", lambda r: None, previews=["fail"])
        self.tampered("preview_fails", lambda r: None, previews=["ok"])

    def test_budget_stop_after_preview(self):
        rec, payloads, _ = rec_of("keep")
        rec = json.loads(json.dumps(rec))
        rec["attempts"] = rec["attempts"][:1]
        rec["final"], rec["batches"], rec["outcome"] = {"attempt": None, "outcome": "fallback"}, [], "fallback"
        rec["preview"] = A4.review_summary(rec["attempts"])
        fx = fixture()
        with self.assertRaises(A4.ReplayError):
            A4.verify_attempts(Harness(fx, [], ["ok"]), rec, fx["observation"], fx["omitted"], fx["catalog"],
                               payloads[0]["messages"][1]["content"], fx["clock"])
        rec["completeness"]["stopped"] = "budget"
        got = A4.verify_attempts(Harness(fx, [], ["ok"]), rec, fx["observation"], fx["omitted"], fx["catalog"],
                                 payloads[0]["messages"][1]["content"], fx["clock"])
        self.assertEqual(got["previews"], 1)
        self.assertEqual((rec["preview"]["response"], rec["preview"]["final"]), ("no_reply", "fallback"))


class Recordings(unittest.TestCase):
    """14. Each version replays only its own recordings."""

    def write(self, recs):
        fd, p = tempfile.mkstemp(suffix=".jsonl")
        with os.fdopen(fd, "w") as f:
            for r in recs:
                f.write(json.dumps(r) + "\n")
        self.addCleanup(os.remove, p)
        return p

    def test_versions(self):
        v4 = rec_of("keep")[0]
        self.assertEqual(len(A4.load_recordings([self.write([v4])])[0][2]), 1)
        v3 = run(A3, "operations_only")[0][0][1]
        v08c = os.path.join(HERE, "..", "..", "docs", "benchmark", "v08c-dev", "eval",
                            "financial-pressure-credit-qwen36-35b-a3b-s41901-decisions.jsonl")
        for p in (self.write([v3]), v08c):
            with self.assertRaises(A4.SettingsError) as e:
                A4.load_recordings([p])
            self.assertIn("version 3 recordings replay with agents/llm/llm_agent3.py", str(e.exception))
        with self.assertRaises(A3.SettingsError):
            A3.load_recordings([self.write([v4])])
        # Another treatment text is another version.
        other = dict(v4, treatmentSha256="0" * 64)
        with self.assertRaises(A4.SettingsError):
            A4.load_recordings([self.write([other])])


if __name__ == "__main__":
    unittest.main()
