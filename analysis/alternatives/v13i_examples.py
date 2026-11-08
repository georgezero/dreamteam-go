#!/usr/bin/env python3
"""Record-only V0.13i lifecycle example archives, byte-reproducible: the
two-draw fixture (its own protocol identity, not the 262-draw experiment),
scripted model replies, scripted HTTP, explicitly synthetic timestamps and
run identities (provenance SYNTHETIC_FIXTURE). Each example keeps its whole
archive, the independent classification and, when valid, the analysis.

    python3 -I -B analysis/alternatives/v13i_examples.py --binary ./dt --out DIR
    python3 -I -B analysis/alternatives/v13i_examples.py --binary ./dt --out DIR --check COMMITTED_DIR
"""
import argparse
import errno
import filecmp
import json
import os
from pathlib import Path
import shutil
import sys

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13i_scenarios as S  # noqa: E402
import v13i_witnesses as W  # noqa: E402

D = S.D


def examples():
    def health(**k):
        return lambda f, pool, clock: {"health": S.Health(clock, **k)}
    return [
        ("complete-both-arms", "both", lambda f, pool, clock: {"health": S.Health(clock)}),
        ("health-refused-before-any-draw", "one", health(modes={1: "refused"})),
        ("health-unhealthy-between-draws", "one", health(modes={3: "unhealthy"})),
        ("health-contradictory-body", "one", health(modes={1: "healthy_error"})),
        ("engine-open-failure", "one", lambda f, pool, clock: {"opener": S.FailingOpener(pool, RuntimeError("scripted engine opening failure"), at=2)}),
        ("interrupt-after-ledger-reconciled", "one", lambda f, pool, clock: {"health": S.Health(clock), "hook": W.At("persist:ledger", "after", 1)}),
        ("cleanup-failure-after-all-draws", "one", lambda f, pool, clock: {"opener": S.FailingOpener(pool, close_error=RuntimeError("scripted pool release failure"))}),
        ("draw-call-limit-halt", ("maxModelCallsPerDraw", 0), lambda f, pool, clock: {}),
        ("final-directory-sync-failure-invalid", "one", lambda f, pool, clock: {"hook": W.At("persist:final_manifest", "dir_fsync", error=OSError(errno.EIO, "scripted directory sync failure"))}),
    ]


def build(binary, out):
    base = {"one": S.Fixture(binary), "both": S.Fixture(binary, arms=("control", "treatment"))}
    index = []
    for name, fx, setup in examples():
        fixture = base[fx] if isinstance(fx, str) else base["one"].with_limits(**{fx[0]: fx[1]})
        dest = os.path.join(out, name)
        shutil.rmtree(dest, ignore_errors=True)
        os.makedirs(dest)
        D.write_json(os.path.join(dest, "protocol.json"), fixture.proto)
        clock = S.Clock()
        pool = fixture.ctx.opener(binary)
        kw = setup(fixture, pool, clock)
        kw.setdefault("opener", pool)
        result = S.attempt(lambda: fixture.record(dest, clock=clock, **kw))
        verdict = fixture.classify(dest)
        D.write_json(os.path.join(dest, "classification.json"), verdict)
        if verdict["state"] != "INVALID":
            D.write_json(os.path.join(dest, "analysis.json"), fixture.analyze(dest))
        index.append({"example": name, "recorder": "returned" if result["returned"] else result["error"],
                      "classification": verdict["state"], "violations": sorted({v["invariant"] for v in verdict["violations"]})})
    D.write_json(os.path.join(out, "index.json"), {"schema": "dreamteam.v13i-lifecycle-examples/v1",
                                                   "notice": "RECORD ONLY: synthetic timestamps and identities; scripted; no inference.",
                                                   "examples": index})
    return index


def same_tree(a, b):
    c = filecmp.dircmp(a, b)
    if c.left_only or c.right_only or c.diff_files or c.funny_files:
        return False
    _, mismatch, errors = filecmp.cmpfiles(a, b, c.common_files, shallow=False)
    return not mismatch and not errors and all(same_tree(os.path.join(a, d), os.path.join(b, d)) for d in c.common_dirs)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--check")
    a = ap.parse_args(argv)
    index = build(os.path.abspath(a.binary), a.out)
    print(f"lifecycle examples: {len(index)} built")
    if a.check and not same_tree(a.out, a.check):
        raise SystemExit("lifecycle examples differ from the committed examples")


if __name__ == "__main__":
    main()
