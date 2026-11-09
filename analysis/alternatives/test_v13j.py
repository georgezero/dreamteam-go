"""V0.13j safety-contract tests. Scripted model replies, scripted HTTP byte
streams, genuine exception objects, real SIGINT/SIGKILL and localhost-only
servers; no inference."""
import ast
import errno
import http.client
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
import lifecycle_v2_spec as LS  # noqa: E402
import v13j as J  # noqa: E402
import v13j_scenarios as S  # noqa: E402
import v13j_verify as V  # noqa: E402

BINARY = os.environ.get("DREAMTEAM_BINARY")
ROOT = HERE.parents[1]


class Contract(unittest.TestCase):
    def test_every_recorder_point_is_declared_and_every_declared_point_exists(self):
        spec = LS.load()
        declared = LS.declared_points(spec)
        src = (HERE / "v13j.py").read_text()
        literal = set(re.findall(r'_hook\("([^"]+)", "([^"]+)"', src))
        persist = set(re.findall(r'_persist\("([^"]+)"', src))
        phases = spec["faultInjection"]["operations"]
        used = literal | {(b, p) for b in persist for p in phases["persist"]["phases"]}
        used |= {("append:calls", p) for p in phases["append"]["phases"]}
        used |= {("safe_point", p) for p in re.findall(r'_safe_point\("([a-z_]+)"', src)}
        self.assertEqual(used - declared, set())
        self.assertEqual(declared - used, set())

    def test_transitions_come_only_from_the_contract(self):
        src = (HERE / "v13j.py").read_text()
        events = set(re.findall(r'_transition\("([a-z_]+)"', src))
        spec = LS.load()
        self.assertTrue(events <= set(spec["events"]) - set(spec["sealOnlyEvents"]), events)

    def test_verifier_does_not_import_the_recorder(self):
        tree = ast.parse((HERE / "v13j_verify.py").read_text())
        names = {a.name for n in ast.walk(tree) if isinstance(n, ast.Import) for a in n.names}
        names |= {n.module for n in ast.walk(tree) if isinstance(n, ast.ImportFrom)}
        for forbidden in ("v13j", "v13j_scenarios", "v13i", "v13i_verify", "lifecycle_spec"):
            self.assertNotIn(forbidden, names)

    def test_historical_code_is_untouched(self):
        for line in (ROOT / "docs/v13j/BASELINE-HASHES.sha256").read_text().splitlines():
            want, path = line.split("  ", 1)
            if path in ("analysis/alternatives/v13h.py", "analysis/acknowledgment/stage_b.py",
                        "analysis/alternatives/lifecycle_v1.json", "analysis/alternatives/v13i.py",
                        "analysis/alternatives/v13i_verify.py"):
                import hashlib
                self.assertEqual(hashlib.sha256((ROOT / path).read_bytes()).hexdigest(), want, path)

    def test_protocol_and_schedule_preserved(self):
        import v13h as H
        rows, h = H.load_contexts(S.CONTEXTS)
        old, sched = H.draft_protocol(rows, h)
        new, current = J.draft_protocol(rows, h)
        self.assertEqual(sched, current)
        self.assertEqual((len(rows), len(current), new["maxModelCalls"]), (21, 262, 802))
        hist = [r for r in rows if r["kind"] == "historical"]
        self.assertEqual(len(hist), 9)
        self.assertEqual(len({r["pair"] for r in rows if r.get("pair")}), 6)
        ignore = {"id", "schema", "driver", "lifecycle"}
        self.assertEqual({k: v for k, v in old.items() if k not in ignore}, {k: v for k, v in new.items() if k not in ignore})
        self.assertEqual(new["lifecycle"], {"id": LS.LIFECYCLE_ID, "file": J.LIFECYCLE_FILE, "sha256": LS.spec_sha256()})


class HealthFamilies(unittest.TestCase):
    """DT13I-03: exception families by exact class; recorder normalization and
    verifier checks agree on genuine exceptions and both refuse masquerades."""

    def probe(self, mode):
        return J.probe_health("http://localhost:9", urlopen=S.scripted_urlopen(mode), utc=S.Clock())

    def test_genuine_wrapped_and_unwrapped_failures(self):
        want = {"healthy": ("HEALTHY", "healthy"), "unhealthy": ("UNHEALTHY", "unhealthy_status"),
                "malformed": ("MALFORMED", "invalid_json"), "list": ("MALFORMED", "not_an_object"),
                "healthy_error": ("MALFORMED", "contradictory_body"),
                "disconnect": ("READ_FAILED", "remote_disconnected"), "incomplete": ("READ_FAILED", "incomplete_read"),
                "bad_status_line": ("READ_FAILED", "http_protocol"),
                "refused": ("READ_FAILED", "connection_refused"), "timeout": ("READ_FAILED", "timeout"),
                "connect_timeout": ("READ_FAILED", "timeout"), "reset": ("READ_FAILED", "connection_reset"),
                "http_503": ("READ_FAILED", "http_status"), "url_incomplete": ("READ_FAILED", "incomplete_read"),
                "url_bad_status": ("READ_FAILED", "http_protocol"), "not_connected": ("READ_FAILED", "http_protocol"),
                "remote_disconnected": ("READ_FAILED", "remote_disconnected"), "url_text": ("READ_FAILED", "url_error"),
                "unreachable": ("READ_FAILED", "network_os_error"), "dns": ("READ_FAILED", "network_os_error")}
        spec = LS.load()
        checker = V.Classifier(None, {}, None, [])
        for mode, (outcome, category) in want.items():
            obs = self.probe(mode)
            self.assertEqual((obs["outcome"], obs["category"]), (outcome, category), mode)
            self.assertIsNone(J.observation_problem(obs, spec), mode)
            self.assertIsNone(checker.observation_problem(obs), mode)

    def test_every_listed_class_normalizes_to_its_family(self):
        spec = LS.load()
        import importlib
        for fam, f in spec["health"]["readFailureFamilies"].items():
            for t in f["types"]:
                mod, _, name = t.rpartition(".")
                cls = getattr(importlib.import_module(mod), name)
                self.assertEqual(J.qualname(cls), t)
                if fam in ("http_status", "url_error"):
                    continue
                try:
                    x = cls("x") if cls is not http.client.IncompleteRead else cls(b"x")
                except TypeError:
                    x = cls.__new__(cls)
                self.assertEqual(J.normalize_exception(x, spec)[0], fam, t)
                self.assertEqual(J.normalize_exception(urllib.error.URLError(x), spec)[0], fam, t)

    def test_incompatible_exceptions_propagate(self):
        spec = LS.load()
        for x in (KeyboardInterrupt(), ValueError("x"), PermissionError(errno.EACCES, "x"), FileNotFoundError("x"),
                  urllib.error.URLError(KeyboardInterrupt()), urllib.error.URLError(ValueError("x")),
                  type("ConnectionRefusedError", (Exception,), {})("an impostor named like a listed class")):
            self.assertIsNone(J.normalize_exception(x, spec), repr(x))
        for mode in ("keyboard", "value_error", "permission", "url_keyboard"):
            with self.assertRaises(BaseException):
                self.probe(mode)

    def test_masquerading_evidence_is_refused_by_both_sides(self):
        spec = LS.load()
        checker = V.Classifier(None, {}, None, [])
        good = self.probe("unreachable")
        for exc in ({"type": "builtins.KeyboardInterrupt", "reasonType": None, "reasonKind": None},
                    {"type": "builtins.ValueError", "reasonType": None, "reasonKind": None},
                    {"type": "OSError", "reasonType": None, "reasonKind": None},
                    {"type": "urllib.error.URLError", "reasonType": "builtins.KeyboardInterrupt", "reasonKind": "exception"},
                    {"type": "urllib.error.URLError", "reasonType": "http.client.IncompleteRead", "reasonKind": "text"}):
            bad = dict(good, exception=exc)
            self.assertIsNotNone(checker.observation_problem(bad), exc)
            self.assertIsNotNone(J.observation_problem(bad, spec), exc)
        ok = self.probe("healthy")
        for change in (dict(httpStatus=503), dict(started=None), dict(ended="2020-01-01T00:00:00Z"),
                       dict(started="2026-01-01T00:00:00.000001Z\n"), dict(outcome="UNHEALTHY"),
                       dict(exception={"type": "urllib.error.URLError", "reasonType": None, "reasonKind": "text"})):
            self.assertIsNotNone(checker.observation_problem(dict(ok, **change)), change)
            self.assertIsNotNone(J.observation_problem(dict(ok, **change), spec), change)


class SafePoints(unittest.TestCase):
    def test_latch_counts_and_never_raises(self):
        latch = J.SignalLatch()
        latch.install()
        try:
            for _ in range(3):
                os.kill(os.getpid(), signal.SIGINT)
            self.assertEqual(latch.take(), 3)
            self.assertEqual(latch.take(), 0)
        finally:
            latch.restore()

    def test_recorder_refuses_a_non_main_thread(self):
        import threading
        errors = []

        def go():
            try:
                J.SignalLatch().install()
            except J.ExperimentError as x:
                errors.append(x)
        t = threading.Thread(target=go)
        t.start()
        t.join()
        self.assertEqual(len(errors), 1)


@unittest.skipUnless(BINARY, "DREAMTEAM_BINARY is not set")
class Lifecycle(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="v13j-")
        self.addCleanup(self.tmp.cleanup)

    def witnesses(self, prefix):
        import v13j_witnesses as W
        results = W.run(BINARY, self.tmp.name, prefix)
        self.assertTrue(results)
        for r in results:
            self.assertTrue(r["agree"], (r["name"], r["mismatch"], r["observed"]))
        return results

    def test_dt13i01_known_calls_and_draws(self):
        self.witnesses("DT13I-01/")

    def test_dt13i02_sealed_completion(self):
        self.witnesses("DT13I-02/")

    def test_dt13i03_health_families(self):
        self.witnesses("DT13I-03/")

    def test_dt13i05_interrupt_reporting(self):
        self.witnesses("DT13I-05/")

    def test_predecessor_witnesses(self):
        for prefix in ("DT13H-", "DT13G/"):
            self.witnesses(prefix)

    def test_contradictory_archives_dt13i04_dt13i06(self):
        import v13j_mutations as M
        r = M.run(BINARY, self.tmp.name)
        for x in r["mutants"]:
            self.assertTrue(x["agree"], x)
        names = {x["name"] for x in r["mutants"]}
        for required in ("engine_open_with_finished_record", "engine_open_with_transport_halt",
                         "premature_schedule_done_cleanup_failure", "duplicate_round_health", "null_failure_event_file",
                         "null_aborted_draw_file", "os_error_named_KeyboardInterrupt", "all_times_trailing_newline",
                         "all_times_arabic_indic_digits", "valid_call_metadata_shift", "seal_missing",
                         "seal_claims_complete_on_partial", "unrecorded_draw_reported_as_zero_calls"):
            self.assertIn(required, names)

    def test_real_signals_sample(self):
        import v13j_signals as G
        cases = G.case_list(os.path.abspath(BINARY), self.tmp.name)
        picked = [c for c in cases if not c["name"].startswith("sigkill-self:")] + \
            [c for c in cases if c["name"].startswith("sigkill-self:")][::25]
        for case in picked:
            r = G.run_case(os.path.abspath(BINARY), self.tmp.name, case)
            self.assertTrue(r["agree"], (r["name"], r["mismatch"], r["observed"]))

    def test_loopback_health(self):
        import v13j_loopback as L
        for case in L.CASES:
            r = L.one(os.path.abspath(BINARY), self.tmp.name, case)
            self.assertTrue(r["agree"], (r["name"], r["mismatch"], r["observed"]))

    def test_fault_matrix_sample(self):
        import v13j_faults as M
        doc = json.loads((ROOT / "docs/v13j/fault-matrix/matrix.json").read_text())
        self.assertEqual(doc["lifecycle"]["sha256"], LS.spec_sha256())
        self.assertEqual(doc["summary"]["disagree"], 0)
        self.assertEqual(doc["summary"]["falseComplete"], 0)
        self.assertEqual(doc["unreachedDeclaredPoints"], [])
        self.assertEqual(doc["coverage"]["transitionsNotObserved"], [])
        self.assertEqual(doc["coverage"]["stopStatesNotObserved"], [])
        for c in doc["cases"][::150]:
            cid, obs = M._case((BINARY, self.tmp.name, c, True))
            self.assertEqual(M.stored(obs), c["observed"], cid)

    def test_complete_run_has_comparisons_and_partial_suppresses_them(self):
        f = S.Fixture(BINARY, arms=("control", "treatment"))
        out = os.path.join(self.tmp.name, "complete")
        clock = S.Clock()
        r = f.record(out, health=S.Health(clock), clock=clock)
        self.assertEqual((r["state"], r["seal"]["terminalState"]), ("COMPLETE", "COMPLETE"))
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
        clock = S.Clock()
        f.record(out, health=S.Health(clock), clock=clock)
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
        clock = S.Clock()
        f.record(out, health=S.Health(clock), clock=clock)
        with self.assertRaises(J.ExperimentError):
            f.record(out)
        v = f.classify(out, synthetic=False)
        self.assertEqual(v["state"], "INVALID")
        self.assertIn("L13", {x["invariant"] for x in v["violations"]})

    def test_unsealed_archive_is_never_terminal_whatever_the_manifest_says(self):
        f = S.Fixture(BINARY)
        out = os.path.join(self.tmp.name, "unsealed")
        clock = S.Clock()
        f.record(out, health=S.Health(clock), clock=clock)
        os.unlink(os.path.join(out, "live", J.SEAL))
        v = f.classify(out)
        self.assertEqual((v["state"], v["sealed"]), ("INVALID", False))
        self.assertIn("S01", {x["invariant"] for x in v["violations"]})
        self.assertEqual(v["preserved"], {"ledgerRows": 2, "completedRows": 2, "dispatchedCalls": 2, "unresolvedCalls": 0})
        with self.assertRaises(J.ExperimentError):
            f.analyze(out)

    def test_persistence_failure_never_seals(self):
        f = S.Fixture(BINARY)
        for phase in ("write", "file_fsync", "replace", "replaced", "dir_fsync"):
            out = os.path.join(self.tmp.name, "persist-" + phase)

            def hook(b, p, ctx, phase=phase):
                if b == "persist:ledger" and p == phase:
                    raise OSError(errno.EIO, "injected")
            clock = S.Clock()
            with self.assertRaises(J.RecordingError):
                f.record(out, health=S.Health(clock), clock=clock, hook=hook)
            v = f.classify(out)
            self.assertEqual((v["state"], v["sealed"]), ("INVALID", False), phase)


if __name__ == "__main__":
    unittest.main()
