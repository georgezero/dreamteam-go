"""The V0.13k safety contract's own consistency and its pure definitions."""
import copy
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))
import lifecycle_v3_spec as L  # noqa: E402


class Contract(unittest.TestCase):
    def test_contract_is_consistent(self):
        spec = L.load()
        table = L.transition_table(spec)
        self.assertEqual(table[(None, "create")], "CREATED")
        terminal = {s for s, v in spec["states"].items() if v["terminal"]}
        self.assertEqual(terminal, {"COMPLETE", "PARTIAL", "INVALID"})
        self.assertFalse(any(f in terminal for f, _ in table if f))
        self.assertNotIn("INVALID", table.values())
        self.assertEqual({t["event"] for t in spec["transitions"] if t["to"] in ("COMPLETE", "PARTIAL")},
                         set(spec["sealOnlyEvents"]))

    def test_owner_safe_points_and_seal_fields_are_declared(self):
        spec = L.load()
        self.assertEqual(set(spec["safePoints"]["points"]), {
            "before_dispatch", "after_transport_returns", "after_call_persisted", "after_draw_persisted",
            "between_draws", "before_health_check", "after_health_check", "before_cleanup", "after_cleanup",
            "before_finalization"})
        seal = set(spec["finalization"]["seal"]["fields"])
        for field in ("runId", "protocol", "scheduleSha256", "manifest", "ledger", "calls", "health", "sideArtifacts",
                      "finalization"):
            self.assertIn(field, seal)
        self.assertEqual(spec["finalization"]["contract"], "dreamteam.run-finalization/v1")
        self.assertIn("NOT CLAIMED", spec["finalization"]["durability"]["powerLoss"])
        self.assertIn("CLAIMED", spec["finalization"]["durability"]["processCrash"])
        self.assertEqual([r["rank"] for r in spec["acceptance"]["severity"]], [1, 2, 3, 4])

    def test_inconsistent_contracts_are_refused(self):
        for mutate in (lambda s: s["transitions"].append({"from": "COMPLETE", "event": "stop", "to": "CLEANUP"}),
                       lambda s: s["transitions"].append({"from": "READY", "event": "invented", "to": "READY"}),
                       lambda s: s["transitions"].append({"from": "READY", "event": "stop", "to": "INVALID"}),
                       lambda s: s["invariants"].pop("L13"),
                       lambda s: s["invariants"].pop("S03"),
                       lambda s: s["transitionPredicates"].pop("schedule_done"),
                       lambda s: s["safePoints"]["points"]["between_draws"].update(kind="MAYBE"),
                       lambda s: s["stopCodes"]["operator"]["states"].append("CLEANUP"),
                       lambda s: s["health"]["readFailureFamilies"]["network_os_error"]["types"].append(
                           "builtins.KeyboardInterrupt"),
                       lambda s: s["health"]["readFailureFamilies"]["timeout"]["types"].append(
                           "http.client.IncompleteRead"),
                       lambda s: s["abortedDraw"]["stages"]["workflow"]["fields"].pop("process"),
                       lambda s: s["faultInjection"]["operations"]["safe_point"]["phases"].pop("between_draws")):
            bad = copy.deepcopy(L.load())
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

    def test_timestamp_grammar_dt13i06(self):
        for good in ("2026-01-01T00:00:00Z", "2026-11-08T12:34:56.123456Z", "2024-02-29T23:59:59Z",
                     "2026-01-01T00:00:00.1Z"):
            self.assertIsNotNone(L.parse_timestamp(good), good)
        arabic = "2026-01-01T00:00:00Z".translate(str.maketrans("0123456789", "٠١٢٣٤٥٦٧٨٩"))
        fullwidth = "2026-01-01T00:00:00Z".translate(str.maketrans("0123456789", "０１２３４５６７８９"))
        for bad in (None, "", "not-a-time", "2026-02-30T00:00:00Z", "2025-02-29T00:00:00Z", "2026-13-01T00:00:00Z",
                    "2026-01-01T24:00:00Z", "2026-01-01T23:60:00Z", "2026-01-01T23:59:60Z",
                    "2026-01-01 00:00:00Z", "2026-01-01T00:00:00+00:00", "2026-01-01T00:00:00-05:00",
                    "2026-01-01T00:00:00.1234567Z", "2026-01-01T00:00:00.Z", 0, 1.5, "2026-1-01T00:00:00Z",
                    "2026-01-01T00:00:00Z\n", "2026-01-01T00:00:00Z\r\n", " 2026-01-01T00:00:00Z",
                    "2026-01-01T00:00:00z", "2026-01-01t00:00:00Z", arabic, fullwidth,
                    "2026-01-01T00:00:00Z\x00"):
            self.assertIsNone(L.parse_timestamp(bad), repr(bad))


if __name__ == "__main__":
    unittest.main()
