#!/usr/bin/env python3
"""Checks a run of the frozen parameter evidence analysis
(dreamteam.parameter-evidence/v1, analyzer parameter-evidence 1) against its
freeze record, analysis/evidence/frozen-parameter-v1.json.

    python3 -I -B analysis/evidence/check_parameter_frozen.py --out <analysis output dir>

The analyzer and its tests must be byte-identical to the ones frozen, and
so must the corpus and every recording it reads. parameter-evidence.json
and parameter-tables.md must be byte-identical to the frozen outputs: they
name no engine build, so nothing is set aside. The committed V0.9a
outputs are never rewritten, and the rules are not tuned against these
recordings again.
"""

import argparse
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.normpath(os.path.join(HERE, "..", ".."))
FREEZE = os.path.join(HERE, "frozen-parameter-v1.json")
FREEZE_SCHEMA = "dreamteam.parameter-evidence-freeze/v1"


def sha256_file(path):
    with open(path, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def load_freeze(path=FREEZE):
    with open(path, encoding="utf-8") as f:
        fr = json.load(f)
    if fr.get("schema") != FREEZE_SCHEMA:
        raise SystemExit(f"{path}: not a {FREEZE_SCHEMA} record")
    return fr


def recordings_fingerprint(doc):
    paths = sorted(i["path"] for i in doc.get("inputs") or [] if i["path"].endswith("-decisions.jsonl"))
    lines = "".join(f"{sha256_file(os.path.join(ROOT, p))}  {p}\n" for p in paths)
    return len(paths), hashlib.sha256(lines.encode("utf-8")).hexdigest()


def check(out, freeze=None):
    """The differences from the freeze record: [] when the run reproduces it."""
    fr = freeze or load_freeze()
    bad = []
    for key, path in (("sha256", fr["analyzer"]["path"]), ("testsSha256", fr["analyzer"]["tests"])):
        if sha256_file(os.path.join(ROOT, path)) != fr["analyzer"][key]:
            bad.append(f"{path} differs from the frozen one")
    if sha256_file(os.path.join(ROOT, fr["corpus"]["path"])) != fr["corpus"]["sha256"]:
        bad.append(f"{fr['corpus']['path']} differs from the frozen corpus")
    with open(os.path.join(out, "parameter-evidence.json"), encoding="utf-8") as f:
        doc = json.load(f)
    if recordings_fingerprint(doc) != (fr["corpus"]["recordings"], fr["corpus"]["recordingsFingerprint"]):
        bad.append("the recordings differ from the frozen corpus")
    for name, key in (("parameter-evidence.json", "parameterEvidenceSha256"),
                      ("parameter-tables.md", "parameterTablesSha256")):
        if sha256_file(os.path.join(out, name)) != fr["outputs"][key]:
            bad.append(f"{name} differs from the frozen output")
    return bad


def main(argv=None):
    ap = argparse.ArgumentParser(description="Check a parameter evidence run against the v1 freeze record.")
    ap.add_argument("--out", required=True, help="the analysis output directory")
    a = ap.parse_args(argv)
    bad = check(a.out)
    for b in bad:
        print("FROZEN PARAMETER V1 MISMATCH:", b)
    if not bad:
        print("parameter evidence v1: frozen analyzer, corpus, recordings and outputs reproduced")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
