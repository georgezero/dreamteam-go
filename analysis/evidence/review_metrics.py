#!/usr/bin/env python3
"""Review metrics of completed independent review forms
(dreamteam.evidence-review-metrics/v1).

    python3 -I -B analysis/evidence/review_metrics.py \
        --package docs/benchmark/v08b-dev/review \
        [--form REVIEWER-A/form.csv [--form REVIEWER-B/form.csv]] --out review-metrics.json

With no form it writes the review status, PENDING, and no figure. With
forms, every one must pass review_analysis.py's checks (every item
labelled with a protocol label, an independence declaration beside it)
and this script's (the frozen form's columns, rows and identities
unchanged; the declaration's own checks all made). Otherwise nothing is
reported.

What it reports, per reviewer:
* the number reviewed, and coverage by rule and rule output;
* agreement with the rule's output: an item labelled `supported` agrees
  (PROTOCOL.md: the evidence supports the output as stated);
* for supported findings: confirmed, unsupported, ambiguous and
  insufficient_evidence;
* precision per rule, only over that rule's reviewed supported findings,
  with its denominator, the number of v1 findings it was drawn from, and
  its qualification: a census when every finding was reviewed, otherwise
  the share among the reviewed findings of a deterministic stratified
  sample, not an estimate for the rest;
* with two or more reviewers, their agreement item by item.

What it never reports: recall. The sample is not a labelled population of
eligible statements, so it cannot show what the rules missed
(docs/benchmark/v09a-dev/review/RECALL_PROTOCOL.md).
"""

import argparse
import csv
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import review_analysis as RA  # noqa: E402

SCHEMA = "dreamteam.evidence-review-metrics/v1"
LABELS = RA.LABELS
RECALL = ("not reported: the 98 items are a stratified sample of the rules' outputs, not a labelled population of "
          "eligible statements, so they cannot show what the rules missed. See "
          "docs/benchmark/v09a-dev/review/RECALL_PROTOCOL.md.")


def load(package):
    with open(os.path.join(package, "manifest.json"), encoding="utf-8") as f:
        manifest = json.load(f)
    with open(os.path.join(package, "items.json"), "rb") as f:
        raw = f.read()
    if hashlib.sha256(raw).hexdigest() != manifest["itemsSha256"]:
        raise SystemExit("the review items differ from the frozen sample")
    with open(os.path.join(package, "form.csv"), encoding="utf-8", newline="") as f:
        rows = list(csv.reader(f))
    return manifest, rows[0], [dict(zip(rows[0], r)) for r in rows[1:]]


def check_form(path, manifest, header, frozen):
    """Problems and warnings beyond review_analysis.read_form's."""
    problems, warnings = [], []
    with open(path, encoding="utf-8", newline="") as f:
        rows = list(csv.reader(f))
    if not rows or rows[0] != header:
        return [f"{path}: the columns are not the frozen form's"], warnings
    want = {r["item"]: r for r in frozen}
    seen = []
    for r in rows[1:]:
        o = dict(zip(header, r + [""] * (len(header) - len(r))))
        seen.append(o["item"])
        f0 = want.get(o["item"])
        if f0 is None:
            continue  # read_form reports it
        if (o["id"], o["rule"], o["rule_output"]) != (f0["id"], f0["rule"], f0["rule_output"]):
            problems.append(f"{path}: item {o['item']} differs from the frozen form")
        if o["label"] and o["label"] != "supported" and not o["comment"].strip():
            warnings.append(f"{path}: item {o['item']} is labelled {o['label']} without an explanation")
    if len(seen) != len(set(seen)):
        problems.append(f"{path}: an item is listed twice")
    decl_path = os.path.join(os.path.dirname(os.path.abspath(path)), "reviewer.json")
    if os.path.exists(decl_path):
        with open(decl_path, encoding="utf-8") as f:
            decl = json.load(f)
        checks = decl.get("declarations")
        if checks is not None and not (isinstance(checks, dict) and checks and all(v is True for v in checks.values())):
            problems.append(f"{path}: the declaration's own checks are not all made")
        if decl.get("itemsSha256") not in (None, manifest["itemsSha256"]):
            problems.append(f"{path}: the declaration names another sample")
    return problems, warnings


def frozen_findings(evidence_path):
    """The number of v1 supported findings per rule: the population each
    reviewed cell of supported findings was drawn from."""
    with open(evidence_path, encoding="utf-8") as f:
        ev = json.load(f)
    return {r: c.get("supported", 0) for r, c in ev["byRule"].items()}


def counts(items, labels):
    c = {"reviewed": len(items), **{l: 0 for l in LABELS}}
    for it in items:
        c[labels[it["item"]]] += 1
    return c


def per_reviewer(manifest, labels, population):
    items = manifest["items"]
    by_rule, by_cell = {}, {}
    for it in items:
        rule, out = it["cell"].split("/", 1)
        by_rule.setdefault(rule, []).append(it)
        by_cell.setdefault(it["cell"], []).append(it)
    agree = sum(1 for it in items if labels[it["item"]] == "supported")
    sup = [it for it in items if it["cell"].endswith("/supported")]
    precision = {}
    for rule in sorted(by_rule):
        cell = by_cell.get(rule + "/supported") or []
        if not cell:
            continue
        ok = sum(1 for it in cell if labels[it["item"]] == "supported")
        n = population.get(rule)
        census = n is not None and len(cell) == n
        precision[rule] = {
            "confirmed": ok, "denominator": len(cell), "share": f"{ok}/{len(cell)}",
            "findingsOfRule": n,
            "qualification": ("census: every v1 supported finding of this rule was reviewed" if census else
                              f"a deterministic stratified sample of {len(cell)} of {n} v1 supported findings, not a "
                              "random sample: the share among the reviewed findings only, not an estimate for the "
                              "others"),
            "notCounted": {l: sum(1 for it in cell if labels[it["item"]] == l) for l in LABELS if l != "supported"},
        }
    return {
        "reviewed": len(items),
        "labels": counts(items, labels),
        "coverageByRule": {r: {"items": len(v), "byRuleOutput": {c.split("/", 1)[1]: len(by_cell[c])
                                                                for c in sorted(by_cell) if c.startswith(r + "/")}}
                           for r, v in sorted(by_rule.items())},
        "agreementWithRuleOutput": {"agree": agree, "of": len(items),
                                    "byCell": {c: {"agree": sum(1 for it in v if labels[it["item"]] == "supported"),
                                                   "of": len(v)} for c, v in sorted(by_cell.items())}},
        "supportedFindings": {"reviewed": len(sup), "confirmed": sum(1 for it in sup if labels[it["item"]] == "supported"),
                              "unsupported": sum(1 for it in sup if labels[it["item"]] == "unsupported"),
                              "ambiguous": sum(1 for it in sup if labels[it["item"]] == "ambiguous"),
                              "insufficient_evidence": sum(1 for it in sup
                                                           if labels[it["item"]] == "insufficient_evidence")},
        "precisionByRule": precision,
    }


def analyze(package, forms, evidence_path):
    manifest, header, frozen = load(package)
    base = {"schema": SCHEMA, "sample": manifest["seed"], "itemsSha256": manifest["itemsSha256"],
            "items": len(manifest["items"]), "recall": RECALL}
    if not forms:
        return dict(base, status="PENDING", reviewers=[],
                    note="INDEPENDENT REVIEW: PENDING. No completed independent review form exists, so no "
                         "agreement, confirmation or precision is reported."), [], []
    problems, warnings, done = [], [], []
    for path in forms:
        got, decl, p = RA.read_form(path, manifest)
        p2, w = check_form(path, manifest, header, frozen)
        problems += p + p2
        warnings += w
        done.append((path, got, decl))
    names = [d.get("reviewer") for _, _, d in done if d]
    if len(names) != len(set(names)):
        problems.append("two forms name the same reviewer")
    if problems:
        return None, problems, warnings
    population = frozen_findings(evidence_path)
    revs = []
    for path, got, decl in done:
        labels = {k: v["label"] for k, v in got.items()}
        revs.append(dict(per_reviewer(manifest, labels, population), reviewer=decl.get("reviewer")))
    doc = dict(base, status="COMPLETE", reviewers=revs)
    if len(done) > 1:
        dis = []
        for it in manifest["items"]:
            ls = [got[it["item"]]["label"] for _, got, _ in done]
            if len(set(ls)) > 1:
                dis.append({"item": it["item"], "id": it["id"], "cell": it["cell"], "labels": ls})
        doc["betweenReviewers"] = {"agreeOnAll": len(manifest["items"]) - len(dis), "of": len(manifest["items"]),
                                   "disagreements": dis}
    return doc, [], warnings


def main(argv=None):
    ap = argparse.ArgumentParser(description="Metrics of completed independent review forms (or PENDING).")
    ap.add_argument("--package", required=True)
    ap.add_argument("--form", action="append", default=[])
    ap.add_argument("--evidence", default=os.path.join(HERE, "..", "..", "docs", "benchmark", "v08a-dev",
                                                       "evidence.json"))
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    doc, problems, warnings = analyze(a.package, a.form, a.evidence)
    for w in warnings:
        print("WARNING:", w)
    if doc is None:
        for p in problems:
            print("NOT ANALYSED:", p)
        return 1
    with open(a.out, "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True)
        f.write("\n")
    print(f"review status: {doc['status']}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
