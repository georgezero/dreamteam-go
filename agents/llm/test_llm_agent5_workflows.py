"""Unit tests of llm-agent 5's workflow selector (V0.11.0): --workflow
validation-first (the default, llm-agent-validation-first-workflow/v1) and
validation-first-preview (llm-agent-workflow/v1, unchanged since V0.11a).
Scripted model replies and scripted engine answers only. No model and no
engine.

    python3 -B -m unittest discover -s agents/llm

The cases of test_llm_agent5.py are reused unchanged; the V0.11a scripted
evidence they make (docs/benchmark/v11a-dev) is not changed by this file.
The exchange with the real engine is tested by
internal/bench/llm_agent5_test.go, for both workflows.
"""

import contextlib
import copy
import io
import json
import os
import tempfile
import unittest

import llm_agent3 as A3
import llm_agent5 as A5
import test_llm_agent4 as T4
import test_llm_agent5 as T5

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
CASES, STAFF, STAFF_LESS, OPS = T5.CASES, T5.STAFF, T5.STAFF_LESS, T5.OPS
VF, PV = "validation-first", "validation-first-preview"
# The pinned identities.
VF_ID = ("llm-agent-validation-first-workflow/v1", "a0204f628c7631d43ebfbb48e3b6e31071b67252daa76f141e3ad3a43ff5f184")
PV_ID = ("llm-agent-workflow/v1", "3ee7d66f4f10fd93e05a908725c792b16b76d01dc29095360dae9d67c0c09d0a")
# SHA-256 of every case's validation-first batches, record and model
# requests (canonical JSON): the same on every platform.
VF_CASES_SHA256 = "fc54dfe438e15e6c0ab46b9ca83aaa97cee1237a88871aaaad6098b80c0b16cc"


def run(case, workflow, points=1, fx=None):
    """test_llm_agent5.run, with the workflow named: (outputs, model
    requests, attempts, the engine)."""
    replies, answers, previews, retries = CASES[case]
    fx = fx or T4.fixture()
    s = T5.settings(maxRetries=retries)
    client = T4.Client(replies * points)
    h = T5.Harness(fx, answers * points, previews * points)
    mem = A3.Memory(s["memoryDecisions"], s["memoryMetricPoints"], s["memoryMaxChars"])
    out, prev = [], None
    with tempfile.TemporaryDirectory() as d:
        ev = A3.Evidence(d)
        for p in range(points):
            batches, rec = A5.ask(h, ev, client, s, mem, p, prev, workflow=workflow)
            mem.record(rec)
            prev = rec
            out.append((batches, json.loads(json.dumps(rec))))
    return out, client.payloads, None, h


def rec_of(case, workflow):
    out, payloads, _, h = run(case, workflow)
    return out[0][0], out[0][1], payloads, h


def kinds(rec):
    return [a["input"]["kind"] for a in rec["attempts"]]


def stamped(rec, workflow):
    """A record as the agent's run loop writes it: with its selection."""
    return dict(rec, workflow=workflow)


class TurnHarness(T5.Harness):
    """The scripted engine with the benchmark's turns: decide at each point,
    then end."""

    def __init__(self, fx, answers, previews=(), points=1):
        super().__init__(fx, answers, previews)
        self.turns = [{"turn": "decide", "point": p} for p in range(points)] + [{"turn": "end"}]
        self.decided = []

    def turn(self):
        return self.turns.pop(0)

    def decide(self, batches):
        self.decided.append(batches)

    def request(self, op, args=None):
        if op == "hello":
            return {"protocol": A3.AGENT_PROTOCOL, "benchmark": {"decision": A3.POLICY_PROTOCOL,
                                                                 "permittedOps": list(A5.NEEDS)}}
        return super().request(op, args)


def loop(case, workflow, points=2, replayer=None, records=None):
    """The agent's policy loop (llm_agent5.run), live or replayed, at
    `points` decision points of one case: (records, decided, evidence dir
    contents)."""
    replies, answers, previews, retries = CASES[case]
    fx = T4.fixture()
    s = T5.settings(maxRetries=retries)
    client = None if replayer else T4.Client(replies * points)
    h = TurnHarness(copy.deepcopy(fx), answers * points, previews * points, points)
    mem = None if replayer else A3.Memory(s["memoryDecisions"], s["memoryMetricPoints"], s["memoryMaxChars"])
    records = [] if records is None else records
    with tempfile.TemporaryDirectory() as d:
        ev = A3.Evidence(d)
        for _ in A5.run(h, ev, client, None if replayer else s, mem, replayer, records, workflow):
            pass
        with open(os.path.join(d, "decisions.jsonl"), encoding="utf-8") as f:
            written = [json.loads(l) for l in f]
    return written, h.decided, h


def write(d, recs, name="decisions.jsonl"):
    p = os.path.join(d, name)
    with open(p, "w", encoding="utf-8") as f:
        for r in recs:
            f.write(json.dumps(r) + "\n")
    return p


def main_with(args, stdin='{"protocol": "dreamteam.benchmark-policy/v1", "turn": "end"}\n'):
    """llm_agent5.main with the harness's input given: (exit code,
    agent.json, stderr)."""
    with tempfile.TemporaryDirectory() as d:
        sp = os.path.join(d, "settings.json")
        with open(sp, "w", encoding="utf-8") as f:
            json.dump({"schema": A3.SETTINGS_SCHEMA, "model": "m", "baseUrl": "http://localhost:9/v1"}, f)
        args = [a.replace("SETTINGS", sp).replace("DIR", d) for a in args]
        err, out = io.StringIO(), io.StringIO()
        old = A5.sys.stdin, A5.sys.stdout
        A5.sys.stdin, A5.sys.stdout = io.StringIO(stdin), out
        try:
            with contextlib.redirect_stderr(err):
                try:
                    code = A5.main(args + ["--evidence", os.path.join(d, "ev")])
                except SystemExit as e:
                    code = e.code
        finally:
            A5.sys.stdin, A5.sys.stdout = old
        info = None
        p = os.path.join(d, "ev", "agent.json")
        if os.path.exists(p):
            with open(p, encoding="utf-8") as f:
                info = json.load(f)
    return code, info, err.getvalue()


class Selection(unittest.TestCase):
    """The command line: the default, explicit choices, an invalid one."""

    def test_default_is_validation_first(self):
        code, info, err = main_with(["--settings", "SETTINGS"])
        self.assertEqual(code, 0, err)
        self.assertEqual((info["workflow"], info["workflowVersion"], info["workflowSha256"], info["preview"]),
                         (VF, *VF_ID, False))
        self.assertEqual(A5.DEFAULT_WORKFLOW, VF)

    def test_explicit_validation_first(self):
        code, info, err = main_with(["--settings", "SETTINGS", "--workflow", VF])
        self.assertEqual((code, info["workflow"], info["workflowVersion"], info["preview"]), (0, VF, VF_ID[0], False))

    def test_explicit_preview(self):
        code, info, err = main_with(["--settings", "SETTINGS", "--workflow", PV])
        self.assertEqual((code, info["workflow"], info["workflowVersion"], info["workflowSha256"], info["preview"]),
                         (0, PV, *PV_ID, True))

    def test_invalid_workflow_argument(self):
        for bad in ("preview", "validation_first", "", "llm-agent-workflow/v1"):
            with self.subTest(bad):
                code, info, err = main_with(["--settings", "SETTINGS", "--workflow", bad])
                self.assertEqual(code, 2)
                self.assertIsNone(info)
                self.assertIn("invalid choice", err)
        with self.assertRaisesRegex(A5.SettingsError, "unknown workflow"):
            rec_of("keep", "bogus")

    def test_library_default_is_unchanged(self):
        # A caller of ask() that names no workflow gets V0.11a's, as before.
        _, rec, _, _ = T5.rec_of("keep")
        self.assertEqual((rec["workflowVersion"], rec["workflowSha256"]), PV_ID)
        self.assertEqual(rec["stages"]["previewed"]["status"], "shown")


class Workflows(unittest.TestCase):
    """The two workflows share validation, refusal feedback, retries and
    submission; only the preview step differs."""

    def test_valid_proposal_without_preview(self):
        b, rec, p, h = rec_of("keep", VF)
        self.assertEqual((b, rec["outcome"], len(p), kinds(rec)), ([[STAFF]], "actions", 1, ["prompt"]))
        self.assertNotIn("preview", [op for op, _ in h.log])
        self.assertIsNone(rec["stages"]["previewed"])
        self.assertIsNone(rec["attempts"][0]["preview"])
        self.assertEqual(rec["completeness"]["previewsRequested"], 0)

    def test_refused_proposal_and_correction(self):
        for case in ("refused_corrected", "refused_abandoned", "repeated_invalid", "mixed_operations_refused"):
            with self.subTest(case):
                b, rec, p, _ = rec_of(case, VF)
                b2, rec2, p2, _ = rec_of(case, PV)
                self.assertEqual(kinds(rec)[:2], ["prompt", "validation"])
                self.assertEqual(p[1]["messages"][-1]["content"],
                                 A5.feedback_text(A5.refused_items(rec["attempts"][0]["validation"])))
                self.assertEqual(p[:2], p2[:2])  # the same feedback, byte for byte
                self.assertEqual(rec["refusals"], rec2["refusals"])
        b, rec, _, _ = rec_of("refused_corrected", VF)
        self.assertEqual((b, rec["outcome"]), ([[STAFF_LESS]], "actions"))

    def test_preview_eligibility_when_enabled(self):
        _, rec, p, _ = rec_of("not_eligible", PV)
        pv = rec["attempts"][0]["preview"]
        self.assertEqual((pv["status"], pv["eligibility"]["eligible"], len(p)), ("not_eligible", False, 1))
        for case, reason in (("keep", "commitment"), ("retained_obligation", "obligation"),
                             ("displaced", "termination")):
            with self.subTest(case):
                _, rec, _, _ = rec_of(case, PV)
                self.assertIn(reason, rec["attempts"][0]["preview"]["eligibility"]["reasons"])

    def test_preview_shown_when_enabled(self):
        for case in ("keep", "revise_after_preview", "abandon_after_preview", "retained_obligation", "displaced"):
            with self.subTest(case):
                _, rec, p, _ = rec_of(case, PV)
                self.assertEqual(rec["stages"]["previewed"]["status"], "shown")
                self.assertEqual(kinds(rec)[1], "preview")

    def test_preview_never_shown_when_disabled(self):
        for case in CASES:
            with self.subTest(case):
                _, rec, p, h = rec_of(case, VF)
                self.assertNotIn("preview", [op for op, _ in h.log])
                self.assertNotIn("preview", kinds(rec))
                self.assertFalse(any(a.get("preview") for a in rec["attempts"]))
                self.assertFalse(any(m["content"].startswith("PREVIEW.") for x in p for m in x["messages"]))
                self.assertIsNone(rec["stages"]["previewed"])

    def test_same_semantics_until_the_preview(self):
        # Every case: until the preview workflow shows a preview, the two
        # send the same messages; where none is shown, they are the same
        # workflow in full (messages, batches, outcome).
        for case in CASES:
            with self.subTest(case):
                b, rec, p, _ = rec_of(case, VF)
                b2, rec2, p2, _ = rec_of(case, PV)
                shown = "preview" in kinds(rec2)
                if not shown:
                    self.assertEqual(p, p2)
                    self.assertEqual((b, rec["outcome"], rec["refusals"]), (b2, rec2["outcome"], rec2["refusals"]))
                    continue
                k = kinds(rec2).index("preview")
                self.assertEqual(p[:k], p2[:k])
                # Without the preview, the validated proposal is submitted.
                self.assertTrue(rec["attempts"][k - 1]["used"])
                acts = rec["attempts"][k - 1]["actions"]
                self.assertEqual(b, [x for x, _ in A3.split(acts)] if acts else [])

    def test_workflow_identity(self):
        self.assertEqual((A5.VALIDATION_FIRST_VERSION, A5.VALIDATION_FIRST_SHA256), VF_ID)
        self.assertEqual((A5.WORKFLOW_VERSION, A5.WORKFLOW_SHA256), PV_ID)
        self.assertEqual(A5.VALIDATION_FIRST_SHA256, A3.sha256("\n".join(
            [A5.VALIDATION_FIRST_VERSION, A5.FEEDBACK_TEMPLATE, A5.VALIDATION_FIRST_RULE])))
        for wf, ident in ((VF, VF_ID), (PV, PV_ID)):
            with self.subTest(wf):
                _, rec, _, _ = rec_of("keep", wf)
                self.assertEqual((rec["workflowVersion"], rec["workflowSha256"]), ident)
                written, _, _ = loop("keep", wf, points=1)
                self.assertEqual((written[0]["workflow"], written[0]["workflowVersion"]), (wf, ident[0]))


class Recordings(unittest.TestCase):
    """Replay takes the workflow from the recording; a legacy recording
    keeps its preview; an unknown or contradictory identity is refused."""

    def test_historical_v011a_recordings(self):
        # The committed V0.11a scripted evidence: every case's record, made
        # before the selector, is the record llm-agent 5 makes today when
        # no workflow is named, and replays with the preview. (The evidence
        # file sorts keys; the replay uses the record in its written order,
        # as decisions.jsonl keeps it.)
        with open(os.path.join(ROOT, "docs", "benchmark", "v11a-dev", "scripted-cases.json"), encoding="utf-8") as f:
            doc = json.load(f)
        self.assertEqual((doc["workflowVersion"], doc["workflowSha256"]), PV_ID)
        for x in doc["cases"]:
            with self.subTest(x["case"]):
                _, rec, payloads, _ = T5.rec_of(x["case"])
                self.assertEqual(rec, x["record"])
                self.assertNotIn("workflow", rec)
                self.assertEqual(A5.workflow_of(rec), PV)
                if x["case"] == "mutate":
                    continue
                with tempfile.TemporaryDirectory() as d:
                    loaded = A5.load_recordings([write(d, [rec])])
                self.assertEqual(A5.recorded_workflow(loaded), PV)
                fx = T4.fixture()
                _, ans, pv, _ = CASES[x["case"]]
                got = A5.verify_attempts(T5.Harness(fx, ans, pv), loaded[0][2][0], fx["observation"], fx["omitted"],
                                         fx["catalog"], payloads[0]["messages"][1]["content"], fx["clock"])
                self.assertEqual(got["previews"], 1 if rec["stages"]["previewed"] else 0)

    def test_missing_legacy_workflow_field(self):
        _, rec, _, _ = rec_of("keep", PV)
        self.assertEqual(A5.workflow_of(rec), PV)  # legacy: preview, as made
        _, rec, _, _ = rec_of("keep", VF)
        with self.assertRaisesRegex(A5.SettingsError, "without its workflow selection"):
            A5.workflow_of(rec)
        with tempfile.TemporaryDirectory() as d:
            with self.assertRaisesRegex(A5.SettingsError, "without its workflow selection"):
                A5.load_recordings([write(d, [rec])])

    def test_unknown_workflow_identity(self):
        _, rec, payloads, _ = rec_of("keep", PV)
        for change in ({"workflowVersion": "llm-agent-workflow/v2"}, {"workflowSha256": "0" * 64},
                       {"workflowVersion": None, "workflowSha256": None}):
            with self.subTest(change):
                bad = dict(rec, **change)
                with tempfile.TemporaryDirectory() as d:
                    with self.assertRaisesRegex(A5.SettingsError, "unknown workflow identity"):
                        A5.load_recordings([write(d, [bad])])
                fx = T4.fixture()
                with self.assertRaisesRegex(A5.ReplayError, "unknown workflow identity"):
                    A5.verify_attempts(T5.Harness(fx, ["ok", "ok"], [T5.C]), bad, fx["observation"], fx["omitted"],
                                       fx["catalog"], payloads[0]["messages"][1]["content"], fx["clock"])

    def test_tampered_workflow_identity(self):
        _, pv, _, _ = rec_of("keep", PV)
        _, vf, _, _ = rec_of("keep", VF)
        cases = [
            (stamped(pv, VF), "contradicts the identity"),           # a preview record called validation-first
            (stamped(vf, PV), "contradicts the identity"),           # and the reverse
            (stamped(dict(pv, workflowVersion=VF_ID[0], workflowSha256=VF_ID[1]), VF), "a preview not due"),
            (stamped(dict(vf, workflowVersion=PV_ID[0], workflowSha256=PV_ID[1]), PV), "a preview missing"),
            (dict(vf, workflowSha256=PV_ID[1]), "unknown workflow identity"),
            (stamped(pv, "bogus"), "contradicts the identity"),
        ]
        fx = T4.fixture()
        prompt = run("keep", PV)[1][0]["messages"][1]["content"]
        for bad, want in cases:
            with self.subTest(want):
                with self.assertRaisesRegex(A5.ReplayError, want):
                    A5.verify_attempts(T5.Harness(fx, ["ok", "ok"], [T5.C]), bad, fx["observation"], fx["omitted"],
                                       fx["catalog"], prompt, fx["clock"])
        # One recording, two workflows.
        with tempfile.TemporaryDirectory() as d:
            with self.assertRaisesRegex(A5.SettingsError, "mix the workflows"):
                A5.load_recordings([write(d, [stamped(vf, VF), dict(pv, point=1)])])
            p1 = write(d, [stamped(vf, VF)], "a.jsonl")
            p2 = write(d, [stamped(pv, PV)], "b.jsonl")
            with self.assertRaisesRegex(A5.SettingsError, "mix the workflows"):
                A5.load_recordings([p1, p2])
            # The command line never overrides the recording's workflow.
            code, info, err = main_with(["--replay", p1, "--workflow", PV])
            self.assertEqual(code, 2)
            self.assertIn("the recordings were made with validation-first", err)

    def test_model_free_replay(self):
        for wf in (VF, PV):
            for case in ("keep", "refused_corrected", "mixed", "displaced", "timeout"):
                with self.subTest((wf, case)):
                    written, decided, _ = loop(case, wf, points=2)
                    self.assertTrue(all(r["workflow"] == wf for r in written))
                    with tempfile.TemporaryDirectory() as d:
                        loaded = A5.load_recordings([write(d, written)])
                        self.assertEqual(A5.recorded_workflow(loaded), wf)
                        replayer = A3.Replayer(loaded)
                        # The replay is driven by the recording; the
                        # workflow argument of the loop is not used.
                        again, decided2, h = loop(case, "validation-first" if wf == PV else PV, points=2,
                                                  replayer=replayer)
                    self.assertEqual(decided2, decided)
                    self.assertEqual([r["batches"] for r in again], [r["batches"] for r in written])
                    self.assertTrue(all(r["workflow"] == wf and r["mode"] == "replay" for r in again))
                    ops = [op for op, _ in h.log]
                    self.assertEqual("preview" in ops, wf == PV and case != "timeout")

    def test_cross_platform_determinism(self):
        # The validation-first outputs of every case, as canonical JSON:
        # pinned here and checked on every CI platform.
        doc = []
        for case in sorted(CASES):
            b, rec, p, h = rec_of(case, VF)
            doc.append({"case": case, "batches": b, "record": rec, "engine": [op for op, _ in h.log],
                        "requests": [A3.sha256(json.dumps(x["messages"], sort_keys=True)) for x in p]})
        got = A3.sha256(json.dumps(doc, sort_keys=True, ensure_ascii=False, separators=(",", ":")))
        self.assertEqual(got, VF_CASES_SHA256)


if __name__ == "__main__":
    unittest.main()
