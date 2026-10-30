#!/usr/bin/env python3
"""Checks a run of the frozen decision evidence analysis
(dreamteam.decision-evidence/v1, analyzer decision-evidence 1) against its
freeze record, analysis/evidence/frozen-v1.json.

    python3 -I -B analysis/evidence/check_frozen.py --out <analysis output dir>

The frozen analyzer's source must be byte-identical to the one frozen.
tables.md and traces.jsonl must be byte-identical to the frozen outputs.
evidence.json must equal the frozen one in every field except two that
name the engine build the replays were made with: each finding's
run.engine, and the SHA-256 of the replay files (they record the engine
version in their hello and run records). Those are removed from both
before the comparison (normalize). The committed v1 artifacts are never
rewritten.
"""

import argparse
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
FREEZE = os.path.join(HERE, "frozen-v1.json")
FREEZE_SCHEMA = "dreamteam.decision-evidence-freeze/v1"


def sha256_file(path):
    with open(path, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def normalize(doc):
    """The evidence without the fields that name the engine build."""
    doc = json.loads(json.dumps(doc))
    doc["inputs"] = [i for i in doc.get("inputs") or [] if not i["path"].startswith("replays/")]
    for f in doc.get("findings") or []:
        f["run"]["engine"] = None
    return doc


def normalized_sha256(doc):
    text = json.dumps(normalize(doc), sort_keys=True, separators=(",", ":"), ensure_ascii=True)
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def load_freeze(path=FREEZE):
    with open(path, encoding="utf-8") as f:
        fr = json.load(f)
    if fr.get("schema") != FREEZE_SCHEMA:
        raise SystemExit(f"{path}: not a {FREEZE_SCHEMA} record")
    return fr


def check(out, freeze=None):
    """The differences from the freeze record: [] when the run reproduces it."""
    fr = freeze or load_freeze()
    bad = []
    if sha256_file(os.path.join(HERE, "decision_evidence.py")) != fr["analyzer"]["sha256"]:
        bad.append("analysis/evidence/decision_evidence.py differs from the frozen analyzer")
    with open(os.path.join(out, "evidence.json"), encoding="utf-8") as f:
        doc = json.load(f)
    if normalized_sha256(doc) != fr["outputs"]["evidenceNormalizedSha256"]:
        bad.append("evidence.json differs from the frozen evidence (beyond the engine build)")
    for name, key in (("tables.md", "tablesSha256"), ("traces.jsonl", "tracesSha256")):
        if sha256_file(os.path.join(out, name)) != fr["outputs"][key]:
            bad.append(f"{name} differs from the frozen output")
    return bad


def main(argv=None):
    ap = argparse.ArgumentParser(description="Check a decision evidence run against the v1 freeze record.")
    ap.add_argument("--out", required=True, help="the analysis output directory")
    a = ap.parse_args(argv)
    bad = check(a.out)
    for b in bad:
        print("FROZEN V1 MISMATCH:", b)
    if not bad:
        print("decision evidence v1: frozen analyzer, evidence, tables and traces reproduced")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
