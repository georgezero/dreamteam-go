"""Tests of the review workflow: the offline review page, the checks and
metrics of completed forms, and the PENDING status. Every label here is
synthetic, made by the test and discarded; none is a review.

    python3 -I -B -m unittest discover -s analysis/evidence
"""

import csv
import io
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import review_analysis as RA  # noqa: E402
import review_metrics as M  # noqa: E402
import review_tool as T  # noqa: E402

ROOT = os.path.normpath(os.path.join(HERE, "..", ".."))
PKG = os.path.join(ROOT, "docs", "benchmark", "v08b-dev", "review")
EVIDENCE = os.path.join(ROOT, "docs", "benchmark", "v08a-dev", "evidence.json")


def frozen():
    manifest, items, header, form = T.load_package(PKG)
    return manifest, items, header, form


def write_form(d, header, form, labels, comment="synthetic", decl=True, **decl_kw):
    os.makedirs(d, exist_ok=True)
    with open(os.path.join(d, "form.csv"), "w", encoding="utf-8", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(header)
        for r in form:
            lab = labels(r)
            w.writerow([r["item"], r["id"], r["rule"], r["rule_output"], lab,
                        "" if lab == "supported" else comment, ""])
    if decl:
        dd = {"reviewer": "synthetic reviewer " + os.path.basename(d), "independent": True, "statement": T.STATEMENT,
              "declarations": {"rules": True, "sample": True, "notes": True}}
        dd.update(decl_kw)
        with open(os.path.join(d, "reviewer.json"), "w", encoding="utf-8") as f:
            json.dump(dd, f)
    return os.path.join(d, "form.csv")


class Page(unittest.TestCase):
    def test_deterministic_complete_and_offline(self):
        manifest, items, header, form = frozen()
        a, b = T.page(manifest, items, header, form), T.page(manifest, items, header, form)
        self.assertEqual(a, b)
        for it in items:
            self.assertIn(it["id"], a)
        # nothing loaded from a network, and no label preset
        self.assertNotRegex(a, r"(?i)<(?:script|link|img)[^>]+(?:src|href)=")
        self.assertNotIn("fetch(", a)
        data = json.loads(a.split('<script type="application/json" id="data">')[1].split("</script>")[0])
        self.assertTrue(all(r["label"] == "" for r in data["form"]))
        self.assertEqual(data["itemsSha256"], manifest["itemsSha256"])

    def test_committed_page_is_current(self):
        manifest, items, header, form = frozen()
        with open(os.path.join(ROOT, "docs", "benchmark", "v09a-dev", "review", "review.html"), encoding="utf-8") as f:
            self.assertEqual(f.read(), T.page(manifest, items, header, form))

    def test_refuses_a_changed_package(self):
        with tempfile.TemporaryDirectory() as d:
            for f in ("manifest.json", "items.json", "form.csv"):
                shutil.copy(os.path.join(PKG, f), d)
            with open(os.path.join(d, "items.json"), "ab") as f:
                f.write(b" ")
            with self.assertRaises(SystemExit):
                T.load_package(d)

    @unittest.skipUnless(shutil.which("node"), "node is not installed")
    def test_page_logic_in_node(self):
        """The page's own functions: the form it saves, read back, and the
        declaration it saves, accepted by the checks and the metrics."""
        manifest, items, header, form = frozen()
        data = json.loads(T.page(manifest, items, header, form).split(
            '<script type="application/json" id="data">')[1].split("</script>")[0])
        js = T.LOGIC + r"""
const data = JSON.parse(require("fs").readFileSync(0, "utf8"));
const labels = data.labels, state = {};
data.form.forEach((f, k) => { const l = labels[k % 4]; state[f.item] = {label: l,
  comment: l === "supported" ? "" : "synthetic, with a comma, \"quotes\"\nand a new line", correction: l === "unsupported" ? "x" : "y"}; });
const decl = {reviewer: "synthetic node reviewer", checks: {rules: true, sample: true, notes: true}};
const csv = toCSV(data.header, formRows(data, state));
const back = readForm(data, csv);
const partial = missing(data, {}, {reviewer: "", checks: {rules: true}});
const tampered = readForm(data, csv.replace(data.form[0].id, "rv1-0000000000000000"));
process.stdout.write(JSON.stringify({csv, back, partial, tampered, complete: missing(data, state, decl),
  decl: declaration(data, decl), notAll: declaration(data, {reviewer: "x", checks: {rules: true}})}));
"""
        out = json.loads(subprocess.run(["node", "-e", js], input=json.dumps(data), capture_output=True, text=True,
                                        check=True).stdout)
        self.assertEqual(out["back"]["problems"], [])
        self.assertEqual(len(out["back"]["state"]), 98)
        self.assertEqual(out["complete"], [])
        self.assertTrue(any("98 of 98 items have no label" in p for p in out["partial"]))
        self.assertTrue(any("not declared" in p for p in out["partial"]))
        self.assertEqual(out["tampered"]["problems"], ["item R001 does not match the frozen form"])
        self.assertTrue(out["decl"]["independent"])
        self.assertFalse(out["notAll"]["independent"])
        rows = list(csv.reader(io.StringIO(out["csv"])))
        self.assertEqual(rows[0], header)
        # the correction is kept only for unsupported
        self.assertEqual({r[6] for r in rows[1:] if r[4] != "unsupported"}, {""})
        with tempfile.TemporaryDirectory() as d:
            with open(os.path.join(d, "form.csv"), "w", encoding="utf-8", newline="") as f:
                f.write(out["csv"])
            with open(os.path.join(d, "reviewer.json"), "w", encoding="utf-8") as f:
                json.dump(out["decl"], f)
            got, decl, problems = RA.read_form(os.path.join(d, "form.csv"), manifest)
            self.assertEqual(problems, [])
            self.assertIn("\n", got["R002"]["comment"])
            doc, problems, warnings = M.analyze(PKG, [os.path.join(d, "form.csv")], EVIDENCE)
            self.assertEqual((problems, warnings, doc["status"]), ([], [], "COMPLETE"))


class Metrics(unittest.TestCase):
    def test_pending_without_a_form(self):
        doc, problems, _ = M.analyze(PKG, [], EVIDENCE)
        self.assertEqual((doc["status"], doc["reviewers"], problems), ("PENDING", [], []))
        self.assertIn("INDEPENDENT REVIEW: PENDING", doc["note"])
        self.assertNotIn("precisionByRule", json.dumps(doc))

    def test_committed_status_is_pending(self):
        with open(os.path.join(ROOT, "docs", "benchmark", "v09a-dev", "review", "review-status.json")) as f:
            self.assertEqual(json.load(f), M.analyze(PKG, [], EVIDENCE)[0])

    def test_metrics_of_a_complete_form(self):
        _, _, header, form = frozen()
        with tempfile.TemporaryDirectory() as d:
            p = write_form(os.path.join(d, "a"), header, form,
                           lambda r: "unsupported" if r["item"] in ("R004",) or r["rule_output"] == "supported"
                           and r["rule"] == "FINANCIAL_AMOUNT_MISMATCH" else "supported")
            doc, problems, _ = M.analyze(PKG, [p], EVIDENCE)
            self.assertEqual(problems, [])
            r = doc["reviewers"][0]
            self.assertEqual(r["reviewed"], 98)
            self.assertEqual(r["agreementWithRuleOutput"], dict(r["agreementWithRuleOutput"], agree=94, of=98))
            pr = r["precisionByRule"]
            self.assertEqual((pr["FINANCIAL_AMOUNT_MISMATCH"]["share"], pr["EXPLICIT_ACTION_OMITTED"]["share"]),
                             ("0/3", "5/5"))
            self.assertTrue(pr["EXPLICIT_ACTION_OMITTED"]["qualification"].startswith("census"))
            self.assertIn("not a random sample", pr["REPEATED_INVALID_PROPOSAL"]["qualification"])
            self.assertEqual(pr["REPEATED_INVALID_PROPOSAL"]["findingsOfRule"], 107)
            self.assertNotIn("EXPLICIT_OPPOSITE_ACTION", pr)
            self.assertEqual(r["supportedFindings"]["reviewed"], 38)
            self.assertTrue(doc["recall"].startswith("not reported"))
            self.assertEqual(r["coverageByRule"]["NEGATIVE_CONTROL"], {"items": 8, "byRuleOutput": {"no_statement": 8}})

    def test_refusals(self):
        _, _, header, form = frozen()
        with tempfile.TemporaryDirectory() as d:
            cases = {
                "incomplete": write_form(os.path.join(d, "i"), header, form,
                                         lambda r: "" if r["item"] == "R098" else "supported"),
                "undeclared": write_form(os.path.join(d, "u"), header, form, lambda r: "supported", decl=False),
                "not independent": write_form(os.path.join(d, "n"), header, form, lambda r: "supported",
                                              independent=False),
                "a declaration not all made": write_form(os.path.join(d, "x"), header, form, lambda r: "supported",
                                                         declarations={"rules": True, "sample": False}),
                "another sample": write_form(os.path.join(d, "s"), header, form, lambda r: "supported",
                                             itemsSha256="0" * 64),
                "a label outside the protocol": write_form(os.path.join(d, "l"), header, form, lambda r: "correct"),
            }
            tampered = write_form(os.path.join(d, "t"), header, form, lambda r: "supported")
            with open(tampered, encoding="utf-8") as f:
                text = f.read()
            with open(tampered, "w", encoding="utf-8") as f:
                f.write(text.replace(",FACTUAL_OBSERVATION_MISMATCH,consistent,", ",FACTUAL_OBSERVATION_MISMATCH,"
                                     "supported,", 1))
            cases["a changed rule output"] = tampered
            for name, p in cases.items():
                with self.subTest(name):
                    doc, problems, _ = M.analyze(PKG, [p], EVIDENCE)
                    self.assertIsNone(doc)
                    self.assertTrue(problems)

    def test_warning_without_explanation_and_two_reviewers(self):
        _, _, header, form = frozen()
        with tempfile.TemporaryDirectory() as d:
            a = write_form(os.path.join(d, "a"), header, form, lambda r: "supported")
            b = write_form(os.path.join(d, "b"), header, form,
                           lambda r: "ambiguous" if r["item"] in ("R001", "R002") else "supported", comment="")
            doc, problems, warnings = M.analyze(PKG, [a, b], EVIDENCE)
            self.assertEqual(problems, [])
            self.assertEqual(len(warnings), 2)
            self.assertEqual((doc["betweenReviewers"]["agreeOnAll"], [x["item"] for x in
                                                                      doc["betweenReviewers"]["disagreements"]]),
                             (96, ["R001", "R002"]))
            c = write_form(os.path.join(d, "a2"), header, form, lambda r: "supported",
                           reviewer="synthetic reviewer a")
            doc, problems, _ = M.analyze(PKG, [a, c], EVIDENCE)
            self.assertIsNone(doc)


if __name__ == "__main__":
    unittest.main()
