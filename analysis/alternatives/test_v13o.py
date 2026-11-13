"""V0.13o: the acceptance standard pinned in the protocol (DT13M-01).

    python3 -B -m unittest discover -s analysis/alternatives -p test_v13o.py
"""
import copy
import hashlib
import json
import os
from pathlib import Path
import re
import sys
import tempfile
import unittest

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[1]
sys.path.insert(0, str(HERE))
import v13m  # noqa: E402
import v13m_verify as V  # noqa: E402
import v13o  # noqa: E402
import v13o_verify as OV  # noqa: E402

BENCH = ROOT / "docs/benchmark/v13o-dev"
ACCEPTANCE = HERE / "acceptance_v1.json"


def protocol():
    return json.loads((BENCH / "protocol-draft.json").read_text(encoding="utf-8"))


class Binding(unittest.TestCase):
    def test_committed_protocol_binds_the_acceptance_file(self):
        p = protocol()
        b = p["acceptance"]
        self.assertEqual(b["sha256"], hashlib.sha256(ACCEPTANCE.read_bytes()).hexdigest())
        self.assertEqual(b["id"], "dreamteam.acceptance-standard/v1")
        self.assertEqual(b["file"], "analysis/alternatives/acceptance_v1.json")
        c = b["clockReconciliation"]
        self.assertIs(type(c["toleranceSeconds"]), int)
        self.assertEqual(c["toleranceSeconds"], 60)
        self.assertEqual(c["toleranceSeconds"], V.ACCEPTANCE["R-HALT-003"]["toleranceSeconds"])
        self.assertIn("operational assumption", c["basis"])
        self.assertIn("not a measured clock-accuracy guarantee", c["basis"])
        self.assertIn("does not change the four-hour request-start guard", c["scope"])
        self.assertIn("no retry, resumption or replacement draw", c["onExcess"])
        v13o.check_binding(p)
        self.assertEqual(OV.pinned_acceptance(p)["toleranceSeconds"], 60)

    def test_only_the_binding_and_identity_differ_from_v13m(self):
        m = json.loads((ROOT / "docs/benchmark/v13m-dev/protocol-draft.json").read_text(encoding="utf-8"))
        o = protocol()
        self.assertEqual(set(o) - set(m), {"acceptance"})
        self.assertEqual(set(m) - set(o), set())
        self.assertEqual({k for k in m if m[k] != o[k]}, {"id", "schema"})
        self.assertEqual((o["draws"], o["maxModelCalls"], o["stop"]["wallClockSeconds"]), (262, 802, 14400))
        sm = json.loads((ROOT / "docs/benchmark/v13m-dev/schedule-draft.json").read_text(encoding="utf-8"))
        so = json.loads((BENCH / "schedule-draft.json").read_text(encoding="utf-8"))
        self.assertEqual(sm["schedule"], so["schedule"])
        self.assertEqual(so["protocolSha256"], hashlib.sha256((BENCH / "protocol-draft.json").read_bytes()).hexdigest())

    def test_verifier_and_recorder_policy_statements_agree(self):
        self.assertEqual((OV.SCOPE, OV.ON_EXCESS, OV.BASIS),
                         (v13o.CLOCK_POLICY["scope"], v13o.CLOCK_POLICY["onExcess"], v13o.CLOCK_POLICY["basis"]))

    def test_verifier_imports_no_recorder(self):
        src = (HERE / "v13o_verify.py").read_text(encoding="utf-8")
        imports = re.findall(r"^\s*(?:import|from)\s+([A-Za-z0-9_]+)", src, re.M)
        self.assertNotIn("v13o", imports)
        self.assertNotIn("v13m", imports)
        self.assertNotIn("v13o", sys.modules["v13m_verify"].__dict__)


class Refusals(unittest.TestCase):
    CASES = {
        "absent": lambda p: p.pop("acceptance"),
        "null": lambda p: p.update(acceptance=None),
        "extra-key": lambda p: p["acceptance"].update(note="x"),
        "other-id": lambda p: p["acceptance"].update(id="dreamteam.acceptance-standard/v2"),
        "other-file": lambda p: p["acceptance"].update(file="acceptance_v1.json"),
        "sha-altered": lambda p: p["acceptance"].update(sha256="0" * 64),
        "sha-uppercase": lambda p: p["acceptance"].update(sha256=p["acceptance"]["sha256"].upper()),
        "tolerance-120": lambda p: p["acceptance"]["clockReconciliation"].update(toleranceSeconds=120),
        "tolerance-float": lambda p: p["acceptance"]["clockReconciliation"].update(toleranceSeconds=60.0),
        "tolerance-bool": lambda p: p["acceptance"]["clockReconciliation"].update(toleranceSeconds=True),
        "tolerance-string": lambda p: p["acceptance"]["clockReconciliation"].update(toleranceSeconds="60"),
        "rule-other": lambda p: p["acceptance"]["clockReconciliation"].update(rule="R-HALT-002"),
        "basis-weakened": lambda p: p["acceptance"]["clockReconciliation"].update(basis="clock-accuracy guarantee"),
        "scope-widened": lambda p: p["acceptance"]["clockReconciliation"].update(scope="also relaxes the guard"),
        "excess-retry": lambda p: p["acceptance"]["clockReconciliation"].update(onExcess="retry the draw"),
        "policy-missing-key": lambda p: p["acceptance"]["clockReconciliation"].pop("basis"),
    }

    def test_each_alteration_is_refused_by_recorder_and_verifier(self):
        for name, fn in self.CASES.items():
            with self.subTest(name):
                p = protocol()
                fn(p)
                with self.assertRaises(v13m.ExperimentError):
                    v13o.check_binding(p)
                with self.assertRaises(V.Invalid) as cm:
                    OV.pinned_acceptance(p)
                self.assertEqual(cm.exception.invariant, "R-PIN-001")

    def test_altered_acceptance_file_is_refused(self):
        p = protocol()
        with tempfile.TemporaryDirectory() as d:
            raw = ACCEPTANCE.read_bytes()
            for name, data in (("tolerance-120", raw.replace(b'"toleranceSeconds": 60', b'"toleranceSeconds": 120')),
                               ("whitespace", raw + b"\n"),
                               ("duplicate-key", raw.replace(b'{\n "schema"', b'{\n "schema": "x",\n "schema"', 1))):
                with self.subTest(name):
                    self.assertNotEqual(data, raw)
                    f = os.path.join(d, name + ".json")
                    open(f, "wb").write(data)
                    with self.assertRaises(V.Invalid):
                        OV.pinned_acceptance(p, f)
                    with self.assertRaises(v13m.ExperimentError):
                        if p["acceptance"] != v13o.acceptance_binding(f):
                            raise v13m.ExperimentError("differs")

    def test_applied_standard_must_equal_pinned_bytes(self):
        p = protocol()
        saved = copy.deepcopy(V.ACCEPTANCE)
        try:
            V.ACCEPTANCE["R-HALT-003"]["toleranceSeconds"] = 120
            with self.assertRaises(V.Invalid):
                OV.pinned_acceptance(p)
        finally:
            V.ACCEPTANCE.clear()
            V.ACCEPTANCE.update(saved)
        OV.pinned_acceptance(p)

    def test_classify_marks_unpinned_protocol_invalid(self):
        p = protocol()
        p.pop("acceptance")
        with tempfile.TemporaryDirectory() as d:
            v = OV.classify(d, p, "0" * 64, [], {})
        self.assertEqual(v["state"], "INVALID")
        self.assertEqual(v["violations"][0]["invariant"], "R-PIN-001")


class Commands(unittest.TestCase):
    def test_protocol_and_schedule_regenerate_byte_identically(self):
        with tempfile.TemporaryDirectory() as d:
            ctx = str(ROOT / "docs/benchmark/v13g-dev/contexts")
            v13o.main(["protocol", "--contexts", ctx, "--out", os.path.join(d, "p.json")])
            self.assertEqual(Path(d, "p.json").read_bytes(), (BENCH / "protocol-draft.json").read_bytes())
            v13o.main(["schedule", "--contexts", ctx, "--protocol", str(BENCH / "protocol-draft.json"),
                       "--out", os.path.join(d, "s.json")])
            self.assertEqual(Path(d, "s.json").read_bytes(), (BENCH / "schedule-draft.json").read_bytes())

    def test_run_refuses_unpinned_and_unlocked(self):
        with tempfile.TemporaryDirectory() as d:
            p = protocol()
            f = os.path.join(d, "p.json")
            Path(f).write_text(json.dumps(p))
            with self.assertRaises(SystemExit) as cm:
                v13o.main(["run", "--protocol", f, "--authorization", "x"])
            self.assertIn("locked protocol", str(cm.exception))
            p.pop("acceptance")
            Path(f).write_text(json.dumps(p))
            with self.assertRaises(v13m.ExperimentError):
                v13o.main(["run", "--protocol", f, "--authorization", "x"])


if __name__ == "__main__":
    unittest.main()
