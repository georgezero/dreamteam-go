"""Tests of the funding-refusal probe runner.

    DREAMTEAM_BINARY=./dt python3 -B -m unittest discover -s analysis/probes

Without DREAMTEAM_BINARY only the runner's pure parts are tested; with it,
the committed probes run twice through the agent protocol, and every
expectation must be met with identical outputs.
"""

import filecmp
import os
import shutil
import tempfile
import unittest

import funding_probes as P

HERE = os.path.dirname(os.path.abspath(__file__))
PROBES = os.path.join(HERE, "..", "..", "docs", "benchmark", "v08b-dev", "probes.json")


class Pure(unittest.TestCase):
    def test_funding_fields(self):
        obs = {"time": "t", "weekday": "Tue", "finance": {"cashCents": 5, "financing": {"solvency": {
            "currentLiabilitiesCents": 3, "reservedCents": 1, "fundingHeadroomCents": 1, "creditLimitCents": 0,
            "creditDrawnCents": 0, "creditAvailableCents": 0}}}}
        f = P.funding(obs)
        self.assertEqual((f["cashCents"], f["fundingHeadroomCents"], f["weekday"]), (5, 1, "Tue"))

    def test_results(self):
        r = P.results_of({"results": [{"ok": False, "code": "insufficient_funds", "message": "m", "id": ""}]})
        self.assertEqual(r, [{"ok": False, "code": "insufficient_funds", "message": "m"}])


@unittest.skipUnless(os.environ.get("DREAMTEAM_BINARY"), "set DREAMTEAM_BINARY to run the probes")
class Run(unittest.TestCase):
    def test_probes_met_and_repeatable(self):
        d = tempfile.mkdtemp()
        try:
            a, b = os.path.join(d, "a"), os.path.join(d, "b")
            la = P.run(os.environ["DREAMTEAM_BINARY"], PROBES, a)
            P.run(os.environ["DREAMTEAM_BINARY"], PROBES, b)
            self.assertEqual(la["expectationsUnmet"], [])
            for name in sorted(os.listdir(a)):
                self.assertTrue(filecmp.cmp(os.path.join(a, name), os.path.join(b, name), shallow=False), name)
        finally:
            shutil.rmtree(d)


if __name__ == "__main__":
    unittest.main()
