#!/usr/bin/env python3
"""Record-only V0.13l safety-contract example archives, byte-reproducible: the
two-draw fixture (its own protocol identity, not the 262-draw experiment),
scripted model replies, scripted HTTP, real in-process SIGINT where named,
explicitly synthetic timestamps and run identities (provenance
SYNTHETIC_FIXTURE). Each example keeps its whole archive (sealed or not), the
independent classification, the public result's interrupt summary and, when
valid, the analysis.

    python3 -I -B analysis/alternatives/v13m_examples.py --binary ./dt --out DIR
    python3 -I -B analysis/alternatives/v13m_examples.py --binary ./dt --out DIR --check COMMITTED_DIR
"""
import argparse
import errno
import filecmp
import os
from pathlib import Path
import shutil
import sys

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13m_scenarios as S  # noqa: E402
import v13m_witnesses as W  # noqa: E402

D = S.D


def examples():
    def health(**k):
        return lambda f, pool, clock, fs: {"health": S.Health(clock, **k)}
    sig = lambda b, p, n=1: lambda f, pool, clock, fs: {"health": S.Health(clock), "hook": W.At(b, p, n)}  # noqa: E731
    return [
        ("complete-both-arms", "both", health()),
        ("complete-with-post-schedule-interrupt", "one", sig("safe_point", "before_cleanup")),
        ("health-refused-before-any-draw", "one", health(modes={1: "refused"})),
        ("health-wrapped-incomplete-read", "one", health(modes={1: "url_incomplete"})),
        ("health-unhealthy-between-draws", "one", health(modes={3: "unhealthy"})),
        ("health-contradictory-body", "one", health(modes={1: "healthy_error"})),
        ("health-keyboard-interrupt-is-not-a-read-failure", "one", health(modes={1: "keyboard"})),
        ("engine-open-failure", "one", lambda f, pool, clock, fs: {"health": S.Health(clock), "opener": S.FailingOpener(
            pool, RuntimeError("scripted engine opening failure"), at=2)}),
        ("operator-stop-between-draws", "one", sig("safe_point", "between_draws", 2)),
        ("operator-stop-after-draw-persisted", "one", sig("draw_execute", "model_call", 1)),
        ("operator-stop-before-dispatch", "one", lambda f, pool, clock, fs: {
            "health": S.Health(clock), "replies": [None, S.F.ABANDON, S.F.ABANDON],
            "hook": W.At("safe_point", "after_call_persisted", 1)}),
        ("unrecorded-draw-calls-unknown", "one", lambda f, pool, clock, fs: {
            "health": S.Health(clock), "draw_runner": W.unrecorded_runner,
            "replies": [None, S.F.ABANDON, S.F.ABANDON]}),
        ("unrecorded-draw-retained-calls-complete-refused", "one", lambda f, pool, clock, fs: {
            "health": S.Health(clock), "draw_runner": W.unrecorded_runner}),
        ("cleanup-failure-after-all-draws", "one", lambda f, pool, clock, fs: {"health": S.Health(clock), "opener": S.FailingOpener(
            pool, close_error=RuntimeError("scripted pool release failure"))}),
        ("draw-call-limit-halt", ("maxModelCallsPerDraw", 0), health()),
        ("final-manifest-sync-failure-unsealed", "one", lambda f, pool, clock, fs: {"health": S.Health(clock), "hook": W.At(
            "persist:manifest", "dir_fsync", 1, "cleanup_succeeded", action="raise",
            error=OSError(errno.EIO, "scripted directory sync failure"))}),
        ("persistent-failure-before-seal-unsealed", "one", lambda f, pool, clock, fs: {
            "health": S.Health(clock), "fs": fs, "hook": W.At("seal", "before", action="persistent", fs=fs)}),
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
        clock, fs = S.Clock(), S.FailingFS()
        pool = fixture.ctx.opener(binary)
        kw = setup(fixture, pool, clock, fs)
        kw.setdefault("opener", pool)
        result = S.attempt(lambda: fixture.record(dest, clock=clock, **kw))
        verdict = fixture.classify(dest)
        D.write_json(os.path.join(dest, "classification.json"), verdict)
        D.write_json(os.path.join(dest, "result.json"), {"recorder": S.recorder_label(result), "state": result.get("state"),
                                                         "interrupts": result.get("interrupts")})
        analysis = "NOT_ATTEMPTED (INVALID)"
        if verdict["state"] != "INVALID":
            try:
                D.write_json(os.path.join(dest, "analysis.json"), fixture.analyze(dest))
                analysis = "ACCEPTED"
            except D.ExperimentError as x:  # V0.13m-lite: full analysis may refuse what classification admits
                D.write_json(os.path.join(dest, "analysis-refusal.json"), {"refused": str(x)})
                analysis = "REFUSED"
        index.append({"example": name, "recorder": S.recorder_label(result), "sealed": verdict["sealed"],
                      "classification": verdict["state"], "violations": sorted({v["invariant"] for v in verdict["violations"]}),
                      "analysis": analysis, "preserved": verdict["preserved"]})
    D.write_json(os.path.join(out, "index.json"), {"schema": "dreamteam.v13m-lifecycle-examples/v1",
                                                   "notice": "RECORD ONLY: synthetic timestamps and identities; scripted; "
                                                             "no inference.", "examples": index})
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
