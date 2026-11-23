"""V0.13r: lock-manifest contract, locked analysis and runner preflight (LC-02).

    python3 -B -m unittest discover -s analysis/alternatives -p test_v13r.py

Fast checks without Git. The Git-identity, archive and command-line cases
are the witnesses (v13r_witness.py, docs/v13r/witness/results.json).
"""
import contextlib
import io
from pathlib import Path
import re
import sys
import unittest

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13m_verify as V  # noqa: E402
import v13p  # noqa: E402
import v13p_verify as PV  # noqa: E402
import v13r_analyze as RA  # noqa: E402
import v13r_lock as L  # noqa: E402
import v13r_preflight as RP  # noqa: E402
import test_v13p  # noqa: E402

H64, C40 = "a" * 64, "b" * 40


def template():
    """A structurally complete TEST ONLY manifest (placeholder hashes; never a lock)."""
    definition = {"estimand": "x"}
    return {
        "schema": L.SCHEMA, "status": L.STATUS, "purpose": L.TEST, "lockId": "TEST-ONLY-unit", "notice": L.NOTICE[L.TEST],
        "protocol": {"file": "p.json", "sha256": H64, "id": "v13o-alt-1"},
        "acceptance": {"file": "analysis/alternatives/acceptance_v1.json", "sha256": H64},
        "schedule": {"file": "s.json", "sha256": H64, "entries": 2},
        "recorder": {"entryPoint": L.ENTRY["recorder"], "modules": {L.ENTRY["recorder"]: H64}},
        "verifier": {"entryPoint": L.ENTRY["verifier"], "modules": {L.ENTRY["verifier"]: H64},
                     "data": {"analysis/alternatives/bindings_v2.json": H64}},
        "analysis": {"entryPoint": L.ENTRY["analysis"], "modules": {L.ENTRY["analysis"]: H64}},
        "engine": {"version": "0.13.0-dev.2", "sourceTreeSha256": H64, "binarySha256": H64, "platform": "test"},
        "models": {"models": {}, "modelOrder": [], "settings": {}, "request": {}, "maxModelCallsPerDraw": 6},
        "presentation": {"texts": {}, "systemPromptSha256": H64, "workflow": {},
                         "humanReviewFreeze": {"commit": C40, "manifest": "m.json", "manifestSha256": H64,
                                               "transcripts": 93, "transcriptsConcatSha256": H64, "staticAssets": 7,
                                               "staticAssetsConcatSha256": H64},
                         "humanReview": {"verdict": L.REVIEW[L.TEST], "record": {"file": "r.md", "sha256": H64}}},
        "outcome": {"analysisSchema": "dreamteam.alternatives-analysis/v10", "definition": definition,
                    "sha256": L.sha(L.canon(definition).encode())},
        "design": {"draws": 2, "maxModelCalls": 12, "wallClockSeconds": 14400, "timeoutSeconds": 300,
                   "toleranceSeconds": 60},
        "approval": {"owner": {"name": "TEST ONLY", "email": "t@fixture.invalid"}, "decision": L.DECISION[L.TEST],
                     "date": "2000-01-01", "record": {"file": "a.md", "sha256": H64}},
        "creation": {"tool": "analysis/alternatives/v13r_lock.py", "candidateSha256": H64, "baseCommit": C40,
                     "lockFile": "lock/LOCK.json", "created": "2000-01-01"},
    }


class Schema(unittest.TestCase):
    def refused(self, doc, code):
        with self.assertRaises(L.LockError) as cm:
            L.validate(doc)
        self.assertEqual(cm.exception.code, code, cm.exception.message)

    def test_complete_test_only_manifest_is_structurally_valid(self):
        L.validate(template())

    def test_every_field_is_required(self):
        for path, keys in L.KEYS.items():
            for k in keys:
                with self.subTest(path=path, key=k):
                    d = template()
                    L._obj(d, path).pop(k)
                    with self.assertRaises(L.LockError):
                        L.validate(d)

    def test_unexpected_fields_are_refused(self):
        for path in L.KEYS:
            with self.subTest(path=path):
                d = template()
                L._obj(d, path)["extra"] = 1
                self.refused(d, "LK-SCHEMA")

    def test_refusals(self):
        cases = {
            "candidate": (lambda d: d.update(schema=L.CANDIDATE_SCHEMA), "LK-SCHEMA"),
            "other-schema": (lambda d: d.update(schema="dreamteam.protocol-lock/v0"), "LK-SCHEMA"),
            "status-candidate": (lambda d: d.update(status="CANDIDATE"), "LK-STATUS"),
            "status-lowercase": (lambda d: d.update(status="locked"), "LK-STATUS"),
            "purpose-other": (lambda d: d.update(purpose="TEST"), "LK-PURPOSE"),
            "notice-other": (lambda d: d.update(notice="x"), "LK-SCHEMA"),
            "sha-uppercase": (lambda d: d["protocol"].update(sha256=H64.upper()), "LK-SCHEMA"),
            "sha-short": (lambda d: d["schedule"].update(sha256="a" * 63), "LK-SCHEMA"),
            "base-abbreviated": (lambda d: d["creation"].update(baseCommit="b" * 12), "LK-SCHEMA"),
            "entry-other": (lambda d: d["analysis"].update(entryPoint="analysis/alternatives/v13p_verify.py"),
                            "LK-SCHEMA"),
            "module-absolute": (lambda d: d["verifier"]["modules"].update({"/etc/x.py": H64}), "LK-SCHEMA"),
            "module-dotdot": (lambda d: d["verifier"]["modules"].update({"../x.py": H64}), "LK-SCHEMA"),
            "entries-bool": (lambda d: d["schedule"].update(entries=True), "LK-SCHEMA"),
            "design-float": (lambda d: d["design"].update(toleranceSeconds=60.0), "LK-SCHEMA"),
            "test-claims-owner-approval": (lambda d: d["approval"].update(decision=L.DECISION[L.RUN]), "LK-APPROVAL"),
            "approval-date": (lambda d: d["approval"].update(date="1 Jan"), "LK-APPROVAL"),
            "approval-owner-empty": (lambda d: d["approval"]["owner"].update(name=""), "LK-APPROVAL"),
            "test-claims-review": (lambda d: d["presentation"]["humanReview"].update(verdict="ACCEPT"), "LK-REVIEW"),
            "outcome-sha": (lambda d: d["outcome"].update(sha256=H64), "LK-OUTCOME"),
            "run-without-accepted-review": (lambda d: d.update(purpose=L.RUN, notice=L.NOTICE[L.RUN]), "LK-REVIEW"),
            "run-other-design": (lambda d: (d.update(purpose=L.RUN, notice=L.NOTICE[L.RUN]),
                                            d["presentation"]["humanReview"].update(verdict=L.REVIEW[L.RUN])),
                                 "LK-DESIGN"),
        }
        for name, (fn, code) in cases.items():
            with self.subTest(name):
                d = template()
                fn(d)
                self.refused(d, code)

    def test_manifest_parsing_is_strict(self):
        raw = L.encode(template())
        for bad in (raw.replace(b'\n "lockId":', b'\n "lockId": "x",\n "lockId":', 1),
                    raw.replace(b'"draws": 2', b'"draws": 2, "draws": 262', 1),
                    raw.replace(b'"draws": 2', b'"draws": NaN', 1),
                    raw.replace(b'"draws": 2', b'"draws": 1e999', 1),
                    raw + b"{}", b"\xff" + raw):
            with self.subTest(bad[:40]):
                with self.assertRaises(L.LockError) as cm:
                    L.load(bad)
                self.assertEqual(cm.exception.code, "LK-PARSE")
        L.load(raw)

    def test_committed_candidate_is_not_a_lock(self):
        raw = (HERE.parents[1] / "docs/v13q/LOCK-CANDIDATE.json").read_bytes()
        with self.assertRaises(L.LockError) as cm:
            L.load(raw)
        self.assertEqual(cm.exception.code, "LK-SCHEMA")


class Parsers(unittest.TestCase):
    def test_lock_parser_accepts_identical_syntax(self):
        for i, data in enumerate(test_v13p.Parser.CORPUS):
            with self.subTest(i=i):
                out = []
                for fn in (L.strict_json, v13p.strict_json, V._strict_json):
                    try:
                        out.append(("ok", fn(data)))
                    except ValueError:
                        out.append(("refused", None))
                self.assertEqual({o[0] for o in out}, {out[0][0]})
                if out[0][0] == "ok":
                    self.assertTrue(V._same(out[0][1], out[2][1]) and V._same(out[1][1], out[2][1]))


class FailsBeforeVerifier(unittest.TestCase):
    def setUp(self):
        self.calls = []
        self.saved = PV.classify, PV.analyze
        PV.classify = lambda *a, **k: self.calls.append("classify")
        PV.analyze = lambda *a, **k: self.calls.append("analyze")

    def tearDown(self):
        PV.classify, PV.analyze = self.saved

    def run_main(self, argv):
        out = io.StringIO()
        with self.assertRaises(SystemExit) as cm, contextlib.redirect_stdout(out), \
                contextlib.redirect_stderr(io.StringIO()):
            RA.main(argv)
        return cm.exception.code, out.getvalue()

    def test_refusals_never_call_the_verifier(self):
        cand = str(HERE.parents[1] / "docs/v13q/LOCK-CANDIDATE.json")
        common = ["--protocol", "p", "--schedule", "s", "--contexts", "c", "--binary", "b", "--dir", "d"]
        for argv in (["classify", "--lock", cand, "--lock-commit", C40] + common,
                     ["analyze", "--lock", cand, "--lock-commit", C40, "--out", "o"] + common,
                     ["classify", "--lock", "/nonexistent/LOCK.json", "--lock-commit", C40] + common,
                     ["classify", "--lock-commit", C40] + common,
                     ["analyze", "--lock", cand] + common + ["--out", "o"]):
            with self.subTest(argv[:3]):
                code, _ = self.run_main(argv)
                self.assertNotEqual(code, 0)
        self.assertEqual(self.calls, [])

    def test_no_fallback_option_exists(self):
        src = (HERE / "v13r_analyze.py").read_text(encoding="utf-8")
        self.assertEqual(re.findall(r'add_argument\("--lock"[^)]*\)', src), ['add_argument("--lock", required=True)'])
        self.assertIn('add_argument("--lock-commit", required=True)', src)


class Runner(unittest.TestCase):
    def test_record_after_preflight_requires_a_passed_test_only_preflight(self):
        with self.assertRaises(L.LockError) as cm:
            RP.record_after_preflight(object(), None, None, {}, "o", "x")
        self.assertEqual(cm.exception.code, "LK-RUNNER")
        ident = {"lockId": "x", "purpose": L.RUN, "lockSha256": H64, "lockCommit": C40}
        pf = RP.Preflight(b"{}", {}, [], [], ident, L.binding(ident, H64), [])
        saved = v13p.run_schedule
        v13p.run_schedule = lambda *a, **k: self.fail("recorded under a RUN lock")
        try:
            with self.assertRaises(L.LockError) as cm:
                RP.record_after_preflight(pf, None, None, {}, "o", "x")
            self.assertEqual(cm.exception.code, "LK-RUNNER")
            pf.identity = dict(ident, purpose=L.TEST)
            with self.assertRaises(L.LockError):
                RP.record_after_preflight(pf, None, None, {}, "o", "  ")
        finally:
            v13p.run_schedule = saved

    def test_binding_names_lock_commit_and_protocol(self):
        ident = {"lockId": "x", "purpose": L.TEST, "lockSha256": H64, "lockCommit": C40}
        self.assertEqual(L.binding(ident, "c" * 64), f"{L.BINDING} lock={H64} commit={C40} protocol={'c' * 64} "
                                                     f"purpose=TEST ONLY;")


class Independence(unittest.TestCase):
    def imports(self, name):
        src = (HERE / name).read_text(encoding="utf-8")
        return set(re.findall(r"^\s*(?:import|from)\s+([A-Za-z0-9_]+)", src, re.M))

    def test_lock_contract_imports_only_the_standard_library(self):
        self.assertFalse({i for i in self.imports("v13r_lock.py") if i.startswith(("v1", "stage", "llm"))})

    def test_analysis_imports_no_recorder_and_preflight_no_verifier(self):
        self.assertEqual({i for i in self.imports("v13r_analyze.py") if i.startswith("v1")}, {"v13r_lock", "v13p_verify"})
        self.assertEqual({i for i in self.imports("v13r_preflight.py") if i.startswith("v1")}, {"v13r_lock", "v13p"})


if __name__ == "__main__":
    unittest.main()
