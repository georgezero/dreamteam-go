"""V0.13p: strict policy and protocol parsing (DT13O-01, DT13O-04).

    python3 -B -m unittest discover -s analysis/alternatives -p test_v13p.py
"""
import contextlib
import copy
import hashlib
import io
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
import v13p  # noqa: E402
import v13p_verify as PV  # noqa: E402
import test_v13o  # noqa: E402

BENCH = ROOT / "docs/benchmark/v13o-dev"
ACCEPTANCE = HERE / "acceptance_v1.json"
RAW = ACCEPTANCE.read_bytes()
PROTO_RAW = (BENCH / "protocol-draft.json").read_bytes()
CLOCK = b'"toleranceSeconds": 60'


def _once(raw, old, new):
    assert raw.count(old) >= 1, old
    return raw.replace(old, new, 1)


def _edit(fn):
    doc = json.loads(RAW)
    fn(doc)
    return (json.dumps(doc, indent=1, ensure_ascii=False) + "\n").encode("utf-8")


def _clock(doc):
    return next(r for r in doc["relationships"] if r["id"] == "R-HALT-003")


def _dup_rule(doc):
    r = copy.deepcopy(_clock(doc))
    r["toleranceSeconds"] = 120
    doc["relationships"].append(r)


# Acceptance files that both components must refuse (R-POLICY-001).
BAD_POLICY = {
    # DT13O-01, first counterexample: a second R-HALT-003 (60 then 120).
    "duplicate-rule-60-then-120": _edit(_dup_rule),
    # DT13O-01, second counterexample: a duplicate top-level key.
    "duplicate-top-level-key": _once(RAW, b'{\n "schema"', b'{\n "schema": "x",\n "schema"'),
    "duplicate-top-level-relationships": _once(RAW, b'\n "relationships": [', b'\n "relationships": [],\n "relationships": ['),
    "nested-duplicate-in-rule": _once(RAW, CLOCK, CLOCK + b',\n   "toleranceSeconds": 120'),
    "nested-duplicate-tolerance-120-then-60": _once(RAW, CLOCK, b'"toleranceSeconds": 120,\n   ' + CLOCK),
    "deep-duplicate-in-failClosedPersistence": _once(RAW, b'"notBuilt":', b'"notBuilt": "x",\n  "notBuilt":'),
    "duplicate-rule-id-other": _edit(lambda d: d["relationships"].append(copy.deepcopy(d["relationships"][0]))),
    "nan": _once(RAW, CLOCK, b'"toleranceSeconds": NaN'),
    "infinity": _once(RAW, CLOCK, b'"toleranceSeconds": Infinity'),
    "negative-infinity": _once(RAW, CLOCK, b'"toleranceSeconds": -Infinity'),
    "overflow-1e999": _once(RAW, CLOCK, b'"toleranceSeconds": 1e999'),
    "non-finite-elsewhere": _once(RAW, b'{\n "schema"', b'{\n "x": NaN,\n "schema"'),
    "not-utf8": _once(RAW, b'"blocking"', b'"blocking\xff"'),
    "trailing-content": RAW + b"{}",
    "top-level-array": b"[" + RAW + b"]",
    "schema-other": _edit(lambda d: d.update(schema="dreamteam.acceptance-standard/v2")),
    "relationships-absent": _edit(lambda d: d.pop("relationships")),
    "relationships-object": _edit(lambda d: d.update(relationships={"R-HALT-003": {}})),
    "relationships-empty": _edit(lambda d: d.update(relationships=[])),
    "relationship-not-object": _edit(lambda d: d["relationships"].append("R-HALT-003")),
    "relationship-id-missing": _edit(lambda d: d["relationships"][0].pop("id")),
    "relationship-id-integer": _edit(lambda d: d["relationships"][0].update(id=4)),
    "relationship-id-empty": _edit(lambda d: d["relationships"][0].update(id="")),
    "clock-rule-absent": _edit(lambda d: d.update(relationships=[r for r in d["relationships"] if r["id"] != "R-HALT-003"])),
    "tolerance-absent": _edit(lambda d: _clock(d).pop("toleranceSeconds")),
    "tolerance-float": _edit(lambda d: _clock(d).update(toleranceSeconds=60.0)),
    "tolerance-bool": _edit(lambda d: _clock(d).update(toleranceSeconds=True)),
    "tolerance-string": _edit(lambda d: _clock(d).update(toleranceSeconds="60")),
    "tolerance-zero": _edit(lambda d: _clock(d).update(toleranceSeconds=0)),
    "tolerance-negative": _edit(lambda d: _clock(d).update(toleranceSeconds=-60)),
}

# Well-formed acceptance files (other bytes): both components accept the policy.
GOOD_POLICY = {
    "committed": RAW,
    "reserialized": _edit(lambda d: None),
    "trailing-newline": RAW + b"\n",
    "tolerance-120": _edit(lambda d: _clock(d).update(toleranceSeconds=120)),
}

# Protocol texts that both components must refuse (R-PROTO-001).
TOL = b'"toleranceSeconds": 60'
BAD_PROTOCOL = {
    # DT13O-04: the reviewer's text, 120 then 60 (last-key-wins applied 60).
    "nested-duplicate-tolerance-120-then-60": _once(PROTO_RAW, TOL, b'"toleranceSeconds": 120,\n  ' + TOL),
    "duplicate-top-level-id": _once(PROTO_RAW, b'\n "id": "v13o-alt-1"', b'\n "id": "v13o-alt-2",\n "id": "v13o-alt-1"'),
    "duplicate-acceptance-sha": _once(PROTO_RAW, b'"sha256": "c6af', b'"sha256": "' + b"0" * 64 + b'",\n  "sha256": "c6af'),
    "deep-duplicate-in-stop": _once(PROTO_RAW, b'"wallClockSeconds": 14400', b'"wallClockSeconds": 28800, "wallClockSeconds": 14400'),
    "nan": _once(PROTO_RAW, TOL, b'"toleranceSeconds": NaN'),
    "infinity": _once(PROTO_RAW, b'"wallClockSeconds": 14400', b'"wallClockSeconds": Infinity'),
    "overflow-1e999": _once(PROTO_RAW, b'"wallClockSeconds": 14400', b'"wallClockSeconds": 1e999'),
    "trailing-content": PROTO_RAW + b"{}",
    "top-level-array": b"[" + PROTO_RAW + b"]",
    "not-utf8": _once(PROTO_RAW, b'"v13o-alt-1"', b'"v13o-alt-1\xff"'),
}


def _accepts(fn, *errors):
    try:
        fn()
        return True
    except errors:
        return False


class CommittedArtifacts(unittest.TestCase):
    def test_committed_acceptance_and_protocol_keep_meaning_and_identity(self):
        self.assertEqual(hashlib.sha256(RAW).hexdigest(),
                         "c6af89ede52284a023f1413ea7b3d46b38a7da516ccd87da92841efe1195a721")
        self.assertEqual(hashlib.sha256(PROTO_RAW).hexdigest(),
                         "202548ad613d2c95b639b345e4fdff262386879d0a17a723fabf50d11e1b7b34")
        p = v13p.load_protocol(PROTO_RAW)
        self.assertTrue(V._same(p, PV.load_protocol(PROTO_RAW)))
        self.assertTrue(V._same(p, json.loads(PROTO_RAW)))  # strict reading == V0.13o's reading
        self.assertEqual(p["id"], "v13o-alt-1")
        self.assertEqual((p["draws"], p["maxModelCalls"], p["stop"]["wallClockSeconds"]), (262, 802, 14400))
        self.assertEqual(v13p.acceptance_binding(), v13o.acceptance_binding())
        v13p.check_binding(p)
        self.assertEqual(PV.pinned_acceptance(p), OV.pinned_acceptance(p))
        self.assertEqual(PV.pinned_acceptance(p)["toleranceSeconds"], 60)
        self.assertEqual(len(PV.policy()), len(V.ACCEPTANCE))

    def test_protocol_and_schedule_regenerate_byte_identically(self):
        with tempfile.TemporaryDirectory() as d:
            ctx = str(ROOT / "docs/benchmark/v13g-dev/contexts")
            with contextlib.redirect_stdout(io.StringIO()):
                v13p.main(["protocol", "--contexts", ctx, "--out", os.path.join(d, "p.json")])
                v13p.main(["schedule", "--contexts", ctx, "--protocol", str(BENCH / "protocol-draft.json"),
                           "--out", os.path.join(d, "s.json")])
            self.assertEqual(Path(d, "p.json").read_bytes(), PROTO_RAW)
            self.assertEqual(Path(d, "s.json").read_bytes(), (BENCH / "schedule-draft.json").read_bytes())

    def test_verifier_imports_no_recorder(self):
        src = (HERE / "v13p_verify.py").read_text(encoding="utf-8")
        imports = re.findall(r"^\s*(?:import|from)\s+([A-Za-z0-9_]+)", src, re.M)
        self.assertEqual(sorted(set(imports) - {"argparse", "json", "os", "sys"}), ["v13o_verify"])
        for mod in ("v13o_verify", "v13m_verify"):
            self.assertFalse({"v13m", "v13o", "v13p"} & set(sys.modules[mod].__dict__))


class Parser(unittest.TestCase):
    CORPUS = [b'{}', b'[]', b'{"a":1,"a":1}', b'{"a":{"b":1,"b":2}}', b'[{"x":0},{"x":0,"x":0}]',
              b'{"a":"\\u0061","a":1}', b'{"a":1,"\\u0061":2}', b'1e999', b'-1e999', b'1e308', b'NaN', b'[Infinity]',
              b'{"n":-Infinity}', b'12345678901234567890123', b'0.1', b'-0.0', b'"\xff"', b'"\xc3\xa9"', b'{} {}',
              b' {"a":[1,2,{"b":null}]} ', b'{"a":true,"b":false}', b'{"":1,"":2}', b'"\\ud800"', b'',
              RAW, PROTO_RAW] + list(BAD_POLICY.values()) + list(BAD_PROTOCOL.values())

    def test_recorder_and_verifier_parsers_accept_identical_syntax(self):
        for i, data in enumerate(self.CORPUS):
            with self.subTest(i=i, data=data[:60]):
                try:
                    r = ("ok", v13p.strict_json(data))
                except ValueError:
                    r = ("refused", None)
                try:
                    v = ("ok", V._strict_json(data))
                except ValueError:
                    v = ("refused", None)
                self.assertEqual(r[0], v[0])
                if r[0] == "ok":
                    self.assertTrue(V._same(r[1], v[1]))


class Policy(unittest.TestCase):
    def _write(self, d, name, data):
        f = os.path.join(d, name + ".json")
        Path(f).write_bytes(data)
        return f

    def test_bad_policies_are_refused_by_both(self):
        with tempfile.TemporaryDirectory() as d:
            for name, data in BAD_POLICY.items():
                with self.subTest(name):
                    f = self._write(d, name, data)
                    with self.assertRaises(v13p.ParseError):
                        v13p.acceptance_binding(f)
                    with self.assertRaises(V.Invalid) as cm:
                        PV.policy(f)
                    self.assertEqual(cm.exception.invariant, "R-POLICY-001")
                    # A protocol re-pinned to these exact bytes is refused by both too.
                    p = v13p.load_protocol(PROTO_RAW)
                    p["acceptance"]["sha256"] = hashlib.sha256(data).hexdigest()
                    with self.assertRaises(v13m.ExperimentError):
                        v13p.check_binding(p, f)
                    with self.assertRaises(V.Invalid):
                        PV.pinned_acceptance(p, f)

    def test_good_policies_are_accepted_by_both(self):
        with tempfile.TemporaryDirectory() as d:
            for name, data in GOOD_POLICY.items():
                with self.subTest(name):
                    f = self._write(d, name, data)
                    b = v13p.acceptance_binding(f)
                    self.assertEqual(b["sha256"], hashlib.sha256(data).hexdigest())
                    self.assertEqual(PV.policy(f), [r["id"] for r in json.loads(data)["relationships"]])
                    v13p.check_binding(dict(acceptance=b), f)

    def test_dt13o01_reproduced_on_v13o_and_closed_on_v13p(self):
        """The recheck's two counterexamples: V0.13o's components disagree; V0.13p's agree (refuse)."""
        with tempfile.TemporaryDirectory() as d:
            # 1. Duplicate R-HALT-003 (60 then 120), protocol re-pinned with 120.
            data = BAD_POLICY["duplicate-rule-60-then-120"]
            f = self._write(d, "dup-rule", data)
            p = v13p.load_protocol(PROTO_RAW)
            p["acceptance"]["sha256"] = hashlib.sha256(data).hexdigest()
            p["acceptance"]["clockReconciliation"]["toleranceSeconds"] = 120
            with self.assertRaises(v13m.ExperimentError):
                v13o.acceptance_binding(f)                 # V0.13o recorder refuses
            saved = copy.deepcopy(V.ACCEPTANCE)
            try:  # the verifier as loaded next to that file (last-wins collapse)
                V.ACCEPTANCE.clear()
                V.ACCEPTANCE.update({r["id"]: r for r in json.loads(data)["relationships"]})
                self.assertEqual(OV.pinned_acceptance(p, f)["toleranceSeconds"], 120)  # V0.13o verifier: 120 applied
                with self.assertRaises(V.Invalid) as cm:
                    PV.pinned_acceptance(p, f)                                       # V0.13p verifier refuses
                self.assertEqual(cm.exception.invariant, "R-POLICY-001")
            finally:
                V.ACCEPTANCE.clear()
                V.ACCEPTANCE.update(saved)
            with self.assertRaises(v13p.ParseError):
                v13p.acceptance_binding(f)                 # V0.13p recorder refuses
            # 2. Duplicate top-level key, protocol re-pinned (60).
            data = BAD_POLICY["duplicate-top-level-key"]
            f = self._write(d, "dup-top", data)
            p = v13p.load_protocol(PROTO_RAW)
            p["acceptance"]["sha256"] = hashlib.sha256(data).hexdigest()
            self.assertEqual(v13o.acceptance_binding(f), p["acceptance"])  # V0.13o recorder would record
            with self.assertRaises(V.Invalid):
                OV.pinned_acceptance(p, f)                 # V0.13o verifier refuses
            with self.assertRaises(v13p.ParseError):
                v13p.acceptance_binding(f)                 # V0.13p recorder refuses
            with self.assertRaises(V.Invalid):
                PV.pinned_acceptance(p, f)                 # V0.13p verifier refuses

    def test_recorder_and_verifier_agree_on_every_policy(self):
        with tempfile.TemporaryDirectory() as d:
            for name, data in list(BAD_POLICY.items()) + list(GOOD_POLICY.items()):
                with self.subTest(name):
                    f = self._write(d, name, data)
                    self.assertEqual(_accepts(lambda: v13p.acceptance_binding(f), v13m.ExperimentError),
                                     _accepts(lambda: PV.policy(f), V.Invalid))


class Protocol(unittest.TestCase):
    def test_bad_protocols_are_refused_by_both(self):
        for name, data in BAD_PROTOCOL.items():
            with self.subTest(name):
                with self.assertRaises(v13p.ParseError):
                    v13p.load_protocol(data)
                with self.assertRaises(V.Invalid) as cm:
                    PV.load_protocol(data)
                self.assertEqual(cm.exception.invariant, "R-PROTO-001")

    def test_dt13o04_reproduced_on_v13o(self):
        """V0.13o read the duplicated text last-key-wins and accepted it."""
        p = json.loads(BAD_PROTOCOL["nested-duplicate-tolerance-120-then-60"])
        v13o.check_binding(p)
        OV.pinned_acceptance(p)
        self.assertEqual(p["acceptance"]["clockReconciliation"]["toleranceSeconds"], 60)

    def test_commands_refuse_duplicated_protocol_text(self):
        data = BAD_PROTOCOL["nested-duplicate-tolerance-120-then-60"]
        ctx = str(ROOT / "docs/benchmark/v13g-dev/contexts")
        with tempfile.TemporaryDirectory() as d:
            f = os.path.join(d, "p.json")
            Path(f).write_bytes(data)
            for argv in (["schedule", "--protocol", f, "--contexts", ctx, "--out", os.path.join(d, "s.json")],
                         ["check", "--binary", "./dt", "--protocol", f, "--contexts", ctx],
                         ["run", "--protocol", f, "--authorization", "x"]):
                with self.subTest(argv[0]):
                    with self.assertRaises(v13p.ParseError):
                        v13p.main(argv)
            for cmd in ("classify", "analyze"):
                with self.subTest(cmd):
                    argv = [cmd, "--protocol", f, "--contexts", ctx, "--dir", d]
                    if cmd == "analyze":
                        argv += ["--binary", "./dt", "--out", d]
                    out = io.StringIO()
                    with self.assertRaises(SystemExit) as cm, contextlib.redirect_stdout(out):
                        PV.main(argv)
                    if cmd == "classify":
                        self.assertEqual(cm.exception.code, 1)
                        v = json.loads(out.getvalue())
                        self.assertEqual((v["state"], v["violations"][0]["invariant"]), ("INVALID", "R-PROTO-001"))
                    else:
                        self.assertIn("R-PROTO-001", str(cm.exception.code))


class Pin(unittest.TestCase):
    def test_r_pin_001_alterations_refused_by_both(self):
        for name, fn in test_v13o.Refusals.CASES.items():
            with self.subTest(name):
                p = v13p.load_protocol(PROTO_RAW)
                fn(p)
                with self.assertRaises(v13m.ExperimentError):
                    v13p.check_binding(p)
                with self.assertRaises(V.Invalid) as cm:
                    PV.pinned_acceptance(p)
                self.assertEqual(cm.exception.invariant, "R-PIN-001")

    def test_classify_marks_unpinned_protocol_invalid(self):
        p = v13p.load_protocol(PROTO_RAW)
        p.pop("acceptance")
        with tempfile.TemporaryDirectory() as d:
            v = PV.classify(d, p, "0" * 64, [], {})
        self.assertEqual((v["state"], v["violations"][0]["invariant"]), ("INVALID", "R-PIN-001"))


class RunnerEntryPoint(unittest.TestCase):
    def test_run_schedule_takes_exact_bytes_and_refuses_before_recording(self):
        calls = []
        saved = v13m.run_schedule
        v13m.run_schedule = lambda proto, sha, *a, **k: calls.append((proto, sha)) or "recorded"
        try:
            with self.assertRaises(v13m.ExperimentError):
                v13p.run_schedule(v13p.load_protocol(PROTO_RAW), [], None, None, {}, "out", "auth")
            with self.assertRaises(v13p.ParseError):
                v13p.run_schedule(BAD_PROTOCOL["nested-duplicate-tolerance-120-then-60"], [], None, None, {}, "o", "a")
            p = json.loads(PROTO_RAW)
            p.pop("acceptance")
            with self.assertRaises(v13m.ExperimentError):
                v13p.run_schedule(json.dumps(p).encode(), [], None, None, {}, "o", "a")
            self.assertEqual(calls, [])
            self.assertEqual(v13p.run_schedule(PROTO_RAW, [], None, None, {}, "o", "a"), "recorded")
            self.assertEqual(calls[0][1], hashlib.sha256(PROTO_RAW).hexdigest())
        finally:
            v13m.run_schedule = saved


if __name__ == "__main__":
    unittest.main()
