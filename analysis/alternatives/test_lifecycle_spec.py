"""The lifecycle contract's own consistency and its pure definitions."""
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))
import lifecycle_spec as L


class Contract(unittest.TestCase):
    def test_contract_is_consistent(self):
        spec = L.load()
        table = L.transition_table(spec)
        self.assertEqual(table[(None, "create")], "CREATED")
        terminal = {s for s, v in spec["states"].items() if v["terminal"]}
        self.assertEqual(terminal, {"COMPLETE", "PARTIAL", "INVALID"})
        self.assertFalse(any(f in terminal for f, _ in table if f))
        self.assertNotIn("INVALID", table.values())

    def test_inconsistent_contracts_are_refused(self):
        spec = L.load()
        for mutate in (lambda s: s["transitions"].append({"from": "COMPLETE", "event": "stop", "to": "CLEANUP"}),
                       lambda s: s["transitions"].append({"from": "READY", "event": "invented", "to": "READY"}),
                       lambda s: s["transitions"].append({"from": "READY", "event": "stop", "to": "INVALID"}),
                       lambda s: s["invariants"].pop("L13")):
            bad = L.load()
            mutate(bad)
            with self.assertRaises(L.SpecError):
                L.validate(bad)

    def test_health_body_categories(self):
        cases = {b'{"status":"healthy"}': "healthy", b'{"status":"healthy","error":"URLError"}': "contradictory_body",
                 b'{"status":"unhealthy"}': "unhealthy_status", b'[]': "not_an_object", b'null': "not_an_object",
                 b'{': "invalid_json", b'\xff': "non_utf8_body", b'{}': "missing_status", b'{"status":""}': "missing_status",
                 b'{"status":"healthy","version":"1"}': "healthy", b'x' * 70000: "oversized_body"}
        for body, want in cases.items():
            self.assertEqual(L.health_body_category(body, 65536), want, body[:40])

    def test_timestamps(self):
        for good in ("2026-01-01T00:00:00Z", "2026-11-08T12:34:56.123456Z", "2024-02-29T23:59:59Z"):
            self.assertIsNotNone(L.parse_timestamp(good))
        for bad in (None, "", "not-a-time", "2026-02-30T00:00:00Z", "2026-01-01 00:00:00Z", "2026-01-01T00:00:00+00:00",
                    "2026-01-01T00:00:00.1234567Z", 0, "2026-1-01T00:00:00Z"):
            self.assertIsNone(L.parse_timestamp(bad), bad)


if __name__ == "__main__":
    unittest.main()
