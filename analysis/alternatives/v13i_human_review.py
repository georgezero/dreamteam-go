"""Freeze, or check, the exact model-visible material offered for the
limited human presentation review (docs/v13i/human-review/).

    python3 -I -B analysis/alternatives/v13i_human_review.py --check
    python3 -I -B analysis/alternatives/v13i_human_review.py --write

The review covers presentation neutrality and clarity of the 93 unchanged
model-visible transcripts and the seven active static assets only. It does
not cover the run-recording system or the experimental protocol.
"""
import argparse
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
DEST = ROOT / "docs/v13i/human-review"
PACKAGE = "docs/v13h/review-package"
FROZEN_AT = "a4d7204c14f36977fee044b3d1a17518d58cc02c"
ACTIVE = {
    "alternatives-system-v2.txt": "agents/llm/text/alternatives-system-v2.txt",
    "initial-v2.txt": "agents/llm/text/initial-v2.txt",
    "correction-v2.txt": "agents/llm/text/correction-v2.txt",
    "response-schemas-v2.json": "agents/llm/text/response-schemas-v2.json",
    "presentation-v3.tmpl": "internal/alternatives/text/presentation-v3.tmpl",
    "view-table.txt": "docs/benchmark/v13g-dev/presentation/view-table.txt",
    "selection-v2.txt": "internal/alternatives/text/selection-v2.txt",
}


def digest(path):
    return hashlib.sha256((ROOT / path).read_bytes()).hexdigest()


def manifest():
    transcripts = sorted((ROOT / PACKAGE / "transcripts").glob("*.json"))
    if len(transcripts) != 93:
        raise SystemExit(f"expected 93 transcripts, found {len(transcripts)}")
    assets = []
    for name, source in sorted(ACTIVE.items()):
        path = f"{PACKAGE}/sources/ACTIVE/{name}"
        if digest(path) != digest(source):
            raise SystemExit("package static asset differs from its active source: " + name)
        assets.append({"path": path, "activeSource": source, "sha256": digest(path)})
    sums = (ROOT / PACKAGE / "SHA256SUMS").read_bytes()
    doc = {
        "schema": "dreamteam.human-review-freeze/v1",
        "status": "HUMAN REVIEW PENDING",
        "frozenAtCommit": FROZEN_AT,
        "scope": "presentation neutrality and clarity of the complete model-visible interactions only; "
                 "not the run-recording system, not the lifecycle verifier, not the experimental protocol",
        "package": {"path": PACKAGE, "SHA256SUMS": hashlib.sha256(sums).hexdigest()},
        "transcripts": [{"path": str(p.relative_to(ROOT)), "sha256": digest(p.relative_to(ROOT))} for p in transcripts],
        "staticAssets": assets,
        "supporting": [{"path": f"{PACKAGE}/{n}", "sha256": digest(f"{PACKAGE}/{n}")}
                       for n in ("FAILURE_FEEDBACK_DISCLOSURE.md", "RECORD_ONLY_INDEX.json")],
        "rereviewRule": "if any listed byte changes, every affected transcript or asset requires human re-review",
    }
    lines = "".join(f"{x['sha256']}  {x['path']}\n" for x in doc["transcripts"] + doc["staticAssets"])
    return {"MANIFEST.json": (json.dumps(doc, indent=1, sort_keys=True) + "\n").encode(),
            "MANIFEST.sha256": lines.encode()}


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    g = ap.add_mutually_exclusive_group(required=True)
    g.add_argument("--write", action="store_true")
    g.add_argument("--check", action="store_true")
    a = ap.parse_args()
    for name, data in manifest().items():
        if a.write:
            (DEST / name).write_bytes(data)
        elif (DEST / name).read_bytes() != data:
            raise SystemExit("frozen human-review material changed: " + name)
    print("Human-review freeze: 93 transcripts, 7 static assets; HUMAN REVIEW PENDING")


if __name__ == "__main__":
    main()
