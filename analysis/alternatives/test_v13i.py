"""V0.13i lifecycle tests. Scripted model replies, scripted HTTP byte streams,
genuine exception objects and localhost-only servers; no inference."""
import ast
import errno
import json
import os
from pathlib import Path
import re
import signal
import sys
import tempfile
import unittest
import urllib.error

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import lifecycle_spec as LS  # noqa: E402
import v13i as I  # noqa: E402
import v13i_scenarios as S  # noqa: E402
import v13i_verify as V  # noqa: E402

BINARY = os.environ.get("DREAMTEAM_BINARY")
ROOT = HERE.parents[1]


class Contract(unittest.TestCase):
    def test_every_recorder_point_is_declared_and_every_declared_point_exists(self):
        spec = LS.load()
        declared = {(b, p) for b, v in spec["faultInjection"]["boundaries"].items()
                    for p in spec["faultInjection"]["operations"][v["operation"]]["phases"]}
        src = (HERE / "v13i.py").read_text()
        literal = set(re.findall(r'_hook\("([^"]+)", "([^"]+)"', src))
        persist = set(re.findall(r'_persist\("([^"]+)"', src))
        phases = spec["faultInjection"]["operations"]["persist"]["phases"]
        used = literal | {(b, p) for b in persist for p in phases}
        used |= {("unlink:marker", "dir_fsync")} | {(b, "dir_fsync") for b in persist}
        self.assertEqual(used - declared, set())
        self.assertEqual(declared - used, set())

    def test_transitions_come_only_from_the_contract(self):
        src = (HERE / "v13i.py").read_text()
        events = set(re.findall(r'_transition\("([a-z_]+)"', src)) | {"create", "stop", "commit_draw",
                                                                      "cleanup_succeeded", "cleanup_failed",
                                                                      "finalize_complete", "finalize_partial"}
        self.assertTrue(events <= set(LS.load()["events"]), events - set(LS.load()["events"]))

    def test_verifier_does_not_import_the_recorder(self):
        tree = ast.parse((HERE / "v13i_verify.py").read_text())
        names = {a.name for n in ast.walk(tree) if isinstance(n, ast.Import) for a in n.names}
        names |= {n.module for n in ast.walk(tree) if isinstance(n, ast.ImportFrom)}
        self.assertNotIn("v13i", names)
        self.assertNotIn("v13i_scenarios", names)

    def test_protocol_and_schedule_preserved(self):
        import v13h as H
        rows, h = H.load_contexts(S.CONTEXTS)
        old, sched = H.draft_protocol(rows, h)
        new, current = I.draft_protocol(rows, h)
        self.assertEqual(sched, current)
        self.assertEqual((len(rows), len(current), new["maxModelCalls"]), (21, 262, 802))
        ignore = {"id", "schema", "driver", "lifecycle"}
        self.assertEqual({k: v for k, v in old.items() if k not in ignore}, {k: v for k, v in new.items() if k not in ignore})
        self.assertEqual(new["lifecycle"], {"id": LS.LIFECYCLE_ID, "file": I.LIFECYCLE_FILE, "sha256": LS.spec_sha256()})


class HealthProbe(unittest.TestCase):
    """DT13H-01: structured categories from genuine exceptions and real HTTP parsing."""

    def probe(self, mode):
        return I.probe_health("http://localhost:9", urlopen=S.scripted_urlopen(mode), utc=S.Clock())

    def test_categories(self):
        want = {"healthy": ("HEALTHY", "healthy"), "unhealthy": ("UNHEALTHY", "unhealthy_status"),
                "malformed": ("MALFORMED", "invalid_json"), "list": ("MALFORMED", "not_an_object"),
                "null": ("MALFORMED", "not_an_object"), "healthy_error": ("MALFORMED", "contradictory_body"),
                "disconnect": ("READ_FAILED", "remote_disconnected"), "incomplete": ("READ_FAILED", "incomplete_read"),
                "refused": ("READ_FAILED", "connection_refused"), "timeout": ("READ_FAILED", "timeout"),
                "connect_timeout": ("READ_FAILED", "timeout"), "reset": ("READ_FAILED", "connection_reset"),
                "http_503": ("READ_FAILED", "http_status")}
        spec = LS.load()
        checker = V.Classifier(None, {}, None, [])
        for mode, (outcome, category) in want.items():
            obs = self.probe(mode)
            self.assertEqual((obs["outcome"], obs["category"]), (outcome, category), mode)
            self.assertIsNone(I.observation_problem(obs, spec), mode)
            self.assertIsNone(checker.observation_problem(obs), mode)

    def test_message_text_is_never_trusted(self):
        def urlopen(url, timeout=20):
            raise urllib.error.URLError("timed out: connection refused")  # a string reason
        obs = I.probe_health("http://localhost:9", urlopen=urlopen, utc=S.Clock())
        self.assertEqual(obs["category"], "url_error")

    def test_non_transport_exceptions_propagate(self):
        def urlopen(url, timeout=20):
            raise ValueError("unknown url type")
        with self.assertRaises(ValueError):
            I.probe_health("http://localhost:9", urlopen=urlopen, utc=S.Clock())

        def interrupted(url, timeout=20):
            raise KeyboardInterrupt()
        with self.assertRaises(KeyboardInterrupt):
            I.probe_health("http://localhost:9", urlopen=interrupted, utc=S.Clock())

    def test_contradictory_observations_are_refused_by_both_sides(self):
        spec = LS.load()
        checker = V.Classifier(None, {}, None, [])
        good = self.probe("healthy")
        for change in (dict(exception={"type": "URLError", "reasonType": "ConnectionRefusedError"}),
                       dict(category="unhealthy_status"), dict(outcome="UNHEALTHY"), dict(body=None),
                       dict(httpStatus=503), dict(ended="2020-01-01T00:00:00Z"), dict(started=None)):
            bad = dict(good, **change)
            self.assertIsNotNone(checker.observation_problem(bad), change)
        refused = self.probe("refused")
        for change in (dict(category="timeout"), dict(exception={"type": "TimeoutError", "reasonType": None}),
                       dict(outcome="HEALTHY"), dict(body=good["body"]), dict(exception=None)):
            self.assertIsNotNone(checker.observation_problem(dict(refused, **change)), change)
        self.assertIsNotNone(I.observation_problem({"error": "URLError"}, spec))
        self.assertIsNotNone(I.observation_problem({"status": "healthy", "error": "URLError"}, spec))


class SignalGuard(unittest.TestCase):
    """Deferred versus interruptible regions under a real SIGINT."""

    def test_deferred_and_interruptible(self):
        g = I._SignalGuard()
        g.install()
        try:
            os.kill(os.getpid(), signal.SIGINT)
            self.assertTrue(g.pending)
            with self.assertRaises(KeyboardInterrupt):
                g.poll()
            with self.assertRaises(KeyboardInterrupt):
                with g.interruptible():
                    os.kill(os.getpid(), signal.SIGINT)
                    for _ in range(1000):
                        pass
            self.assertFalse(g.pending)
        finally:
            g.restore()


@unittest.skipUnless(BINARY, "DREAMTEAM_BINARY is not set")
class Lifecycle(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="v13i-")
        self.addCleanup(self.tmp.cleanup)

    def witnesses(self, prefix):
        import v13i_witnesses as W
        results = W.run(BINARY, self.tmp.name, prefix)
        self.assertTrue(results)
        for r in results:
            self.assertTrue(r["agree"], (r["name"], r["mismatch"], r["observed"]))
        return results

    def test_dt13h01_health_failures(self):
        self.witnesses("DT13H-01/")

    def test_dt13h02_interruption_boundaries(self):
        self.witnesses("DT13H-02/")

    def test_dt13h03_fail_closed_finalization(self):
        self.witnesses("DT13H-03/")

    def test_v13g_and_contract_regressions(self):
        self.witnesses("DT13G/")
        self.witnesses("V0.13i/")

    def mutations(self, names=None):
        import v13i_mutations as M
        r = M.run(BINARY, self.tmp.name, names=names)
        for x in r["mutants"]:
            self.assertTrue(x["agree"], x)
        return r

    def test_dt13h04_contradictory_evidence_and_dt13h05_timestamps(self):
        r = self.mutations()
        names = {x["name"] for x in r["mutants"]}
        for required in ("completed_row_with_cleanup_failure", "complete_with_unhealthy_initial_observation",
                         "engine_open_failure_with_workflow_evidence", "impossible_run_model_calls",
                         "duplicate_completed_draws", "missing_schedule_entry", "stop_code_differs_from_journal",
                         "illegal_transition", "finalization_hash_forged", "unsupported_terminal_status",
                         "null_journal_timestamp", "unparseable_journal_timestamp", "run_started_after_ended",
                         "journal_time_reversed", "health_field_missing"):
            self.assertIn(required, names)

    def test_real_signals(self):
        import v13i_sigint as G
        for case in G.CASES:
            r = G.parent_case(os.path.abspath(BINARY), self.tmp.name, case)
            self.assertTrue(r["agree"], (r["name"], r["mismatch"], r["observed"]))

    def test_loopback_health(self):
        import v13i_loopback as L
        for case in L.CASES:
            r = L.one(os.path.abspath(BINARY), self.tmp.name, case)
            self.assertTrue(r["agree"], (r["name"], r["mismatch"], r["observed"]))

    def test_fault_matrix_sample(self):
        import v13i_faults as M
        doc = json.loads((ROOT / "docs/v13i/fault-matrix/matrix.json").read_text())
        self.assertEqual(doc["lifecycle"]["sha256"], LS.spec_sha256())
        self.assertEqual(doc["summary"]["disagree"], 0)
        self.assertEqual(doc["unreachedDeclaredPoints"], [])
        for c in doc["cases"][::40]:
            cid, obs = M._case((BINARY, self.tmp.name, c, True))
            self.assertEqual(obs, c["observed"], cid)

    def test_complete_run_has_comparisons_and_partial_suppresses_them(self):
        f = S.Fixture(BINARY, arms=("control", "treatment"))
        out = os.path.join(self.tmp.name, "complete")
        clock = S.Clock()
        doc = f.record(out, health=S.Health(clock), clock=clock)
        self.assertEqual(doc["state"], "COMPLETE")
        a = f.analyze(out)
        self.assertEqual((a["runStatus"], len(a["draws"]), a["completedVerification"]), ("COMPLETE", 2, "VERIFIED"))
        self.assertEqual(len(a["armDifferences"]), 1)
        out = os.path.join(self.tmp.name, "partial")
        clock = S.Clock()
        f.record(out, health=S.Health(clock), clock=clock, opener=S.FailingOpener(
            f.ctx.opener(BINARY), close_error=RuntimeError("release")))
        a = f.analyze(out)
        self.assertEqual((a["runStatus"], len(a["draws"]), a["armDifferences"], a["pairs"]), ("PARTIAL", 2, [], []))

    def test_completed_rows_replay_under_the_predecessor_draw_contract(self):
        import v13h as H
        f = S.Fixture(BINARY)
        out = os.path.join(self.tmp.name, "replay")
        f.record(out)
        rows = json.loads(Path(out, "live/draws.json").read_text())
        c, pool = f.ctx.by_id["S1-a"], f.ctx.opener(BINARY)
        try:
            for row in rows:
                H.replay_draw(row, c, f.ctx.settings[H.QWEN], pool, f.ctx.memory(c, H.QWEN))
        finally:
            pool.close()

    def test_no_resume_and_synthetic_provenance_refused_outside_fixtures(self):
        f = S.Fixture(BINARY)
        out = os.path.join(self.tmp.name, "once")
        f.record(out)
        with self.assertRaises(I.ExperimentError):
            f.record(out)
        v = V.classify(out, f.proto, f.proto_sha, f.sched, f.ctx.by_id)
        self.assertEqual(v["state"], "INVALID")
        self.assertIn("L13", {x["invariant"] for x in v["violations"]})

    def test_persistence_failure_never_completes(self):
        f = S.Fixture(BINARY)
        for phase in ("write", "file_fsync", "replace", "replaced", "dir_fsync"):
            out = os.path.join(self.tmp.name, "persist-" + phase)

            def hook(b, p, ctx, phase=phase):
                if b == "persist:ledger" and p == phase:
                    raise OSError(errno.EIO, "injected")
            with self.assertRaises(I.RecordingError):
                f.record(out, hook=hook)
            self.assertEqual(f.classify(out)["state"], "INVALID", phase)


if __name__ == "__main__":
    unittest.main()
