#!/usr/bin/env python3
"""Descriptive agreement between completed independent review forms and
the frozen decision evidence (dreamteam.evidence-review-result/v1).

    python3 -I -B analysis/evidence/review_analysis.py \
        --manifest docs/benchmark/v08b-dev/review/manifest.json \
        --form REVIEWER-A.csv [--form REVIEWER-B.csv] --out result.json

A form counts only if it is complete: every item of the frozen sample
labelled with one of supported, unsupported, ambiguous or
insufficient_evidence, and a reviewer declaration (reviewer.json beside
it: {"reviewer": ..., "independent": true, "statement": ...}) saying the
reviewer did not write or tune the rules. Without one the script reports
nothing: no agreement is computed from an unreviewed sample, and no
precision or recall is computed at all. The results are kept apart from
the frozen rule outputs.
"""

import argparse
import csv
import hashlib
import json
import os
import sys

SCHEMA = "dreamteam.evidence-review-result/v1"
LABELS = ("supported", "unsupported", "ambiguous", "insufficient_evidence")


def read_form(path, manifest):
    with open(path, encoding="utf-8", newline="") as f:
        rows = list(csv.DictReader(f))
    want = {i["item"]: i for i in manifest["items"]}
    got = {}
    problems = []
    for r in rows:
        item = r.get("item")
        if item not in want:
            problems.append(f"{path}: unknown item {item}")
            continue
        if r.get("id") != want[item]["id"]:
            problems.append(f"{path}: item {item} has another identifier")
        label = (r.get("label") or "").strip()
        if label not in LABELS:
            problems.append(f"{path}: item {item} has no valid label")
        got[item] = {"label": label, "comment": r.get("comment") or "", "correction": r.get("correction") or ""}
    missing = sorted(set(want) - set(got))
    if missing:
        problems.append(f"{path}: {len(missing)} items not labelled")
    decl_path = os.path.join(os.path.dirname(os.path.abspath(path)), "reviewer.json")
    decl = None
    if os.path.exists(decl_path):
        with open(decl_path, encoding="utf-8") as f:
            decl = json.load(f)
    if not decl or decl.get("independent") is not True or not decl.get("reviewer"):
        problems.append(f"{path}: no declaration of an independent reviewer (reviewer.json)")
    return got, decl, problems


def analyze(manifest, forms):
    """Agreement by cell (rule/output) from complete forms; None with the
    problems if any form is incomplete or undeclared."""
    problems, done = [], []
    for path in forms:
        got, decl, p = read_form(path, manifest)
        problems += p
        done.append((path, got, decl))
    if problems or not done:
        return None, problems or ["no completed form"]
    cells = {}
    for it in manifest["items"]:
        c = cells.setdefault(it["cell"], {"reviewed": 0, **{l: 0 for l in LABELS}, "disagreements": 0})
        labels = [got[it["item"]]["label"] for _, got, _ in done]
        c["reviewed"] += 1
        c[labels[0]] += 1
        c["disagreements"] += len(set(labels)) > 1
    return {"schema": SCHEMA, "sample": manifest["seed"], "itemsSha256": manifest["itemsSha256"],
            "reviewers": [d.get("reviewer") for _, _, d in done],
            "byCell": {k: cells[k] for k in sorted(cells)},
            "note": "Counts of labels on a seeded sample of the frozen outputs, by the first reviewer, and the items "
                    "on which reviewers disagreed. Not a precision or a recall."}, []


def main(argv=None):
    ap = argparse.ArgumentParser(description="Agreement of completed independent review forms (descriptive).")
    ap.add_argument("--manifest", required=True)
    ap.add_argument("--form", action="append", default=[])
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    with open(a.manifest, encoding="utf-8") as f:
        manifest = json.load(f)
    with open(os.path.join(os.path.dirname(a.manifest), "items.json"), "rb") as f:
        if hashlib.sha256(f.read()).hexdigest() != manifest["itemsSha256"]:
            print("the review items differ from the frozen sample")
            return 1
    res, problems = analyze(manifest, a.form)
    if res is None:
        for p in problems:
            print("NOT ANALYSED:", p)
        return 1
    with open(a.out, "w", encoding="utf-8") as f:
        json.dump(res, f, indent=1, sort_keys=True)
        f.write("\n")
    print(f"review result: {len(manifest['items'])} items, {len(res['reviewers'])} reviewer(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
