"""V0.13l safety-contract tests. Scripted model replies, scripted HTTP byte
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
import lifecycle_v4_spec as LS  # noqa: E402
import v13l as J  # noqa: E402
import v13l_scenarios as S  # noqa: E402
import v13l_verify as V  # noqa: E402

BINARY = os.environ.get("DREAMTEAM_BINARY")
ROOT = HERE.parents[1]


class Contract(unittest.TestCase):
    def test_every_recorder_point_is_declared_and_every_declared_point_exists(self):
        spec = LS.load()
        declared = LS.declared_points(spec)
        src = (HERE / "v13l.py").read_text()
        literal = set(re.findall(r'_hook\("([^"]+)", "([^"]+)"', src))
        persist = set(re.findall(r'_persist\("([^"]+)"', src))
        phases = spec["faultInjection"]["operations"]
        used = literal | {(b, p) for b in persist for p in phases["persist"]["phases"]}
        used |= {("append:calls", p) for p in phases["append"]["phases"]}
        used |= {("safe_point", p) for p in re.findall(r'_safe_point\("([a-z_]+)"', src)}
        self.assertEqual(used - declared, set())
        self.assertEqual(declared - used, set())

    def test_inventory_covers_every_named_field_and_mutation_kind(self):
        inventory = json.loads(LS.spec_bytes())
        self.assertEqual(inventory["schema"], "dreamteam.evidence-bindings/2")
        self.assertEqual(len(inventory["bindings"]), 35)
        self.assertEqual(len(set(inventory["mutationKinds"])), 10)
        for binding in inventory["bindings"]:
            self.assertEqual(set(binding["fields"]), set(binding["fieldDetails"]), binding["id"])
            self.assertTrue(binding["relatedFields"])
            self.assertTrue(binding["replayVerification"])
            for desc in binding["fieldDetails"].values():
                self.assertEqual(desc["binding"], binding["id"])
                self.assertIsInstance(desc["nullable"], bool)

    def test_pending_preservation_does_not_double_count_committed_copies(self):
        import v13l_faults as M
        ctx = dict(committed=1, completedRows=1, pendingStatus="completed",
                   pendingDurable=False, pendingRetained=True, dispatched=0, receivedUnrecorded=0)
        obs = dict(recorder="RecordingError", classification="INVALID", sealed=False,
                   ledgerRows=1, ledgerCompleted=1, pendingCompleted=1,
                   archiveCompletedRowSha256=["0" * 64], journalDispatched=0,
                   reportedCompleted=1, reportedRowSha256="1" * 64)
        self.assertIn("a completed pending draw was lost", M.agree({}, obs, ctx))
        obs["archiveCompletedRowSha256"].append("1" * 64)
        self.assertEqual(M.agree({}, obs, ctx), [])
        obs["archiveCompletedRowSha256"].pop()
        ctx["pendingRetained"] = False
        self.assertEqual(M.agree({}, obs, ctx), [])

    def test_completion_seal_contract_is_preserved(self):
        old = json.loads((HERE / "lifecycle_v2.json").read_text())
        self.assertEqual(old["finalization"], LS.load()["finalization"])

    def test_transitions_come_only_from_the_contract(self):
        src = (HERE / "v13l.py").read_text()
        events = set(re.findall(r'_transition\("([a-z_]+)"', src))
        spec = LS.load()
        self.assertTrue(events <= set(spec["events"]) - set(spec["sealOnlyEvents"]), events)

    def test_verifier_does_not_import_the_recorder(self):
        tree = ast.parse((HERE / "v13l_verify.py").read_text())
        names = {a.name for n in ast.walk(tree) if isinstance(n, ast.Import) for a in n.names}
        names |= {n.module for n in ast.walk(tree) if isinstance(n, ast.ImportFrom)}
        for forbidden in ("v13l", "v13l_scenarios", "v13i", "v13i_verify", "lifecycle_spec", "lifecycle_v4_spec"):
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

    def test_class_objects_are_trusted_by_identity(self):
        for fam in LS.load()["health"]["readFailureFamilies"].values():
            for name in fam["types"]:
                module, _, short = name.rpartition(".")
                fake = type(short, (Exception,), {"__module__": module})("impostor")
                self.assertIsNone(J.normalize_exception(fake, LS.load()), name)
                self.assertIsNone(J.normalize_exception(urllib.error.URLError(fake), LS.load()), name)

    def test_health_json_has_no_ambiguous_or_nonfinite_fields(self):
        import io
        for body in (b'{"status":"unhealthy","status":"healthy"}', b'{"status":"healthy","load":NaN}'):
            class Response(io.BytesIO):
                status = 200
                length = len(body)
            obs = J.probe_health("http://localhost:9", urlopen=lambda *a, **kw: Response(body), utc=S.Clock())
            self.assertEqual((obs["outcome"], obs["category"]), ("MALFORMED", "invalid_json"))
            self.assertIsNone(J.observation_problem(obs, LS.load()))
            self.assertIsNone(V.Classifier(None, {}, None, []).observation_problem(obs))

    def test_overloaded_class_equality_cannot_impersonate_identity(self):
        class EqualType(type):
            def __hash__(cls):
                return hash(OSError)
            def __eq__(cls, other):
                return other is OSError or other is cls
        fake = EqualType("OSError", (ValueError,), {"__module__": "builtins"})("impostor")
        self.assertIsNone(J.normalize_exception(fake, LS.load()))
        self.assertIsNone(J.normalize_exception(urllib.error.URLError(fake), LS.load()))

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
        self.tmp = tempfile.TemporaryDirectory(prefix="v13l-")
        self.addCleanup(self.tmp.cleanup)

    def witnesses(self, prefix):
        import v13l_witnesses as W
        results = W.run(BINARY, self.tmp.name, prefix)
        self.assertTrue(results)
        for r in results:
            self.assertTrue(r["agree"], (r["name"], r["mismatch"], r["observed"]))
        return results

    def test_dt13j_original_regressions(self):
        import v13l_regressions as R
        r = R.run(BINARY, str(Path(self.tmp.name) / "originals"))
        self.assertEqual(r["summary"], {"cases": 16, "agree": 16})

    def test_nested_json_types_bind_without_numeric_boolean_coercion(self):
        import v13l_mutations as M
        f = S.Fixture(BINARY)
        for target in ("usage", "server_mirror"):
            out = str(Path(self.tmp.name) / target)
            c = S.Clock()
            inner = S.Health(c)
            def health():
                return dict(inner(), models=[{"id": "scripted", "slots": 1}])
            scripted = S.Model([S.F.ABANDON] * 4)
            def model(payload):
                status, body, error = scripted(payload)
                value = json.loads(body)
                value["usage"] = {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
                return status, json.dumps(value), error
            f.record(out, clock=c, health=health, model=model)
            f.analyze(out)
            arc = M.Arc(out)
            if target == "usage":
                arc.rows[0]["calls"][0]["usage"]["prompt_tokens"] = True
            else:
                arc.files["server-before.json"]["models"][0]["slots"] = True
            arc.write()
            with self.assertRaises(J.ExperimentError, msg=target):
                f.analyze(out)

    def test_completed_draw_preservation(self):
        import v13l_preservation as P
        scratch = str(Path(self.tmp.name) / "preservation")
        Path(scratch).mkdir()
        for kind in P.INHERITED:
            r = P.one(BINARY, scratch, kind)
            self.assertTrue(r["agree"], r)

    def test_response_handoff_preserves_a_received_response(self):
        import v13l_preservation as P
        scratch = str(Path(self.tmp.name) / "responses")
        Path(scratch).mkdir()
        for op, retention in (("write", "ok"), ("fsync", "rename"), ("close", "none")):
            r = P.response_case(BINARY, scratch, op, retention)
            self.assertTrue(r["agree"], r)
            self.assertEqual(r["transportInvocations"], r["journalDispatches"])
            self.assertEqual(r["classification"], "INVALID")

    def test_ownership_boundaries_sample(self):
        """A sample of the exhaustive boundary enumeration (the full set is
        generated into docs/v13l/preservation/results.json)."""
        import v13l_preservation as P
        scratch = Path(self.tmp.name) / "boundaries"
        scratch.mkdir()
        for kind in ("draw", "response"):
            window = P.Window(kind)
            f = S.Fixture(BINARY)
            P._boundary_run(BINARY, scratch / ("discover-" + kind), window,
                            [P.MARKER] + f.replies() if kind == "response" else None)
            self.assertTrue(window.closed and window.sites, kind)
            want = None
            if kind == "draw":
                want = J.canon(json.loads((scratch / "discover-draw/live/draws.json").read_text())[0])
            for site in window.sites[::max(1, len(window.sites) // 8)]:
                r = P._boundary_case((BINARY, str(scratch), kind, site, want))
                self.assertTrue(r["agree"], r)

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
        import v13l_mutations as M
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
        import v13l_signals as G
        cases = G.case_list(os.path.abspath(BINARY), self.tmp.name)
        picked = [c for c in cases if not c["name"].startswith("sigkill-self:")] + \
            [c for c in cases if c["name"].startswith("sigkill-self:")][::25]
        for case in picked:
            r = G.run_case(os.path.abspath(BINARY), self.tmp.name, case)
            self.assertTrue(r["agree"], (r["name"], r["mismatch"], r["observed"]))

    def test_loopback_health(self):
        import v13l_loopback as L
        for case in L.CASES:
            r = L.one(os.path.abspath(BINARY), self.tmp.name, case)
            self.assertTrue(r["agree"], (r["name"], r["mismatch"], r["observed"]))

    def test_fault_matrix_sample(self):
        import v13l_faults as M
        doc = json.loads((ROOT / "docs/v13l/fault-matrix/matrix.json").read_text())
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
        self.assertEqual({k: v["preserved"][k] for k in ("ledgerRows", "completedRows", "dispatchedCalls", "unresolvedCalls")}, {"ledgerRows": 2, "completedRows": 2, "dispatchedCalls": 2, "unresolvedCalls": 0})
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


class StrictJson(unittest.TestCase):
    """dreamteam.strict-json/v1 in the verifier (DT13K-01)."""

    def test_type_tags_are_never_coerced(self):
        for a, b in ((0, False), (1, True), (1, 1.0), (0.0, -0.0), ("0", 0), (None, False), ([], {}),
                     ({"a": None}, {}), ({"a": [1, {"b": 0}]}, {"a": [1, {"b": False}]}),
                     ({"a": 1}, {"a": 1, "b": None}), ([1, 2], [2, 1]), (2 ** 64 + 1, float(2 ** 64 + 1))):
            self.assertFalse(V._same(a, b), (a, b))
            self.assertFalse(V._same(b, a), (b, a))
        for a, b in ((0, 0), (False, False), (None, None), (1.5, 1.5), (2 ** 80, 2 ** 80), ("x", "x"),
                     ({"a": 1, "b": [None, True]}, {"b": [None, True], "a": 1}), ([], [])):
            self.assertTrue(V._same(a, b), (a, b))

    def test_parser_refuses_ambiguity(self):
        for bad in (b'{"a":1,"a":1}', b'{"a":{"b":1,"b":2}}', b'[NaN]', b'[Infinity]', b'[-Infinity]', b'[1e400]',
                    b'\xef\xbb\xbf{}', b'\xff'):
            with self.assertRaises(ValueError, msg=bad):
                V._strict_json(bad)
        self.assertEqual(V._strict_json(b'123456789012345678901234567890'), 123456789012345678901234567890)
        self.assertIs(type(V._strict_json(b'1.0')), float)
        self.assertIs(type(V._strict_json(b'1')), int)

    def test_produced_values_compare_as_archived(self):
        self.assertTrue(V._same_as_archived((1, 2), [1, 2]))
        self.assertFalse(V._same_as_archived({"t": 0}, {"t": False}))
        self.assertFalse(V._same_as_archived(float("nan"), None))

    def test_no_python_equality_on_retained_requests(self):
        src = (HERE / "v13l_verify.py").read_text()
        self.assertNotIn("json.loads(body) == req", src)
        self.assertIn("_strict_json(body)", src)


class Relationships(unittest.TestCase):
    def test_every_planned_operation_is_implemented(self):
        import v13l_binding_matrix as BM
        names = {p["name"] for r in LS.relationships() for p in r["matrix"].values() if p.get("op") == "custom"}
        self.assertEqual(names - set(BM.CUSTOM), set())
        bases = {p["base"] for r in LS.relationships() for p in r["matrix"].values() if "base" in p}
        self.assertEqual(bases - {"complete", "health_stop", "aborted_workflow", "halt_row", "engine_close",
                                  "halt_close", "initial_digests", "final_digests"}, set())

    def test_committed_relationship_matrix_has_no_survivor(self):
        doc = json.loads((ROOT / "docs/v13l/binding-matrix/results.json").read_text())
        rel = doc["crossFieldRelationships"]["summary"]
        self.assertEqual(rel["agree"], rel["cases"])
        self.assertEqual(rel["survivingContradictions"], [])
        self.assertEqual(rel["cases"], 7 * len(LS.relationships()))
        self.assertEqual(doc["summary"]["survivingContradictions"], [])


class GeneratedContexts(unittest.TestCase):
    """DT13K-07: generated fault cases keep every preservation expectation."""

    def test_committed_cases_keep_preservation_keys(self):
        import v13l_faults as M
        doc = json.loads((ROOT / "docs/v13l/fault-matrix/matrix.json").read_text())
        for c in doc["cases"]:
            self.assertEqual(set(M.PRESERVATION_KEYS) - set(c["context"]), set(), c["id"])

    def test_oracle_refuses_a_dropped_expectation_and_a_lost_handoff(self):
        import v13l_faults as M
        ctx = dict(state="DRAW_ACTIVE", stopping=False, committed=0, completedRows=0, lastStatus=None, pending=True,
                   pendingDurable=False, pendingStatus="completed", pendingRetained=True, dispatched=1,
                   scheduleDone=False, stopped=False, receivedUnrecorded=0, partialRetained=False)
        lost = {"recorder": "RecordingError", "classification": "INVALID", "sealed": False, "ledgerRows": 0,
                "ledgerCompleted": 0, "pendingCompleted": 0, "archiveCompletedRowSha256": [], "journalDispatched": 1,
                "reportedCompleted": 1, "reportedRowSha256": "1" * 64, "receivedResponses": 0}
        self.assertIn("a completed pending draw was lost", M.agree({}, lost, ctx))
        dropped = {k: v for k, v in ctx.items() if k != "pendingRetained"}
        self.assertTrue(any("dropped preservation expectations" in b for b in M.agree({}, lost, dropped)))
        unrecorded = dict(ctx, pendingStatus=None, pending=False, receivedUnrecorded=1)
        gone = dict(lost, reportedCompleted=0, journalOutcomeKs=[], receivedKs=[])
        self.assertIn("a received model response was lost", M.agree({}, gone, unrecorded))
        self.assertNotIn("a received model response was lost", M.agree({}, dict(gone, journalOutcomeKs=[1]), unrecorded))
        self.assertNotIn("a received model response was lost", M.agree({}, dict(gone, receivedKs=[1]), unrecorded))


@unittest.skipUnless(BINARY, "DREAMTEAM_BINARY is not set")
class Remediation(unittest.TestCase):
    """DT13K-01..05 witnesses on genuine successor archives (sample; the full
    before/after record is docs/v13l/dt13k/)."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="v13l-dt13k-")
        self.addCleanup(self.tmp.cleanup)

    def test_witnesses_meet_remediation_expectations(self):
        import v13l_dt13k as D
        impl, scratch, results = D.Impl("v13l"), Path(self.tmp.name), []
        D.strict_json_cases(impl, BINARY, scratch / "strict", results)
        D.causal_cases(impl, BINARY, scratch / "causal", results)
        for r in results:
            self.assertTrue(D.agree_after(r), (r["case"], r.get("analysis")))
        self.assertGreaterEqual(sum(r["finding"] == "DT13K-01" for r in results), 9)
        self.assertGreaterEqual(sum(r["finding"] in ("DT13K-02", "DT13K-03") for r in results), 9)


if __name__ == "__main__":
    unittest.main()
