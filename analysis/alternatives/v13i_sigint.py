#!/usr/bin/env python3
"""Real-process signal tests at lifecycle boundaries.

A child process records the two-draw fixture. At the named declared point
its hook announces itself and blocks reading its stdin; the parent then
delivers a real SIGINT to the child with os.kill and, after a pause, lets it
continue. Inside an interruptible region (health read, draw execution,
session release) the interrupt lands immediately; elsewhere the recorder
defers it to the next transition boundary, or past finalization once the run
has stopped. Process-termination witnesses have the child signal itself
(SIGKILL, SIGTERM, SIGINT) at the point. The parent then classifies and
analyses the archive independently.

    python3 -I -B analysis/alternatives/v13i_sigint.py run --binary ./dt --scratch DIR --out FILE
"""
import argparse
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import time

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

SCHEMA = "dreamteam.v13i-signal-tests/v1"


def P(**k):
    return dict({"classification": "PARTIAL", "cleanup": "SUCCEEDED", "abortedStage": None, "exit": 0}, **k)


INVALID = {"classification": "INVALID"}
# (name, point boundary, phase, occurrence, event, mode, expected)
CASES = [
    ("before_first_health_check", "health_read", "before", 1, None, "sigint",
     P(stopCode="operator", stopState="PREFLIGHT", completed=0, attemptedRead=False)),
    ("during_first_health_check", "health_read", "during", 1, None, "sigint",
     P(stopCode="operator", stopState="PREFLIGHT", completed=0, attemptedRead=True)),
    ("after_health_success", "health_read", "after", 1, None, "sigint",
     P(stopCode="operator", stopState="PREFLIGHT", completed=0, attemptedRead=False)),
    ("before_draw_execution", "draw_execute", "before", 1, None, "sigint",
     P(stopCode="operator", stopState="DRAW_ACTIVE", completed=0, abortedStage="draw_setup")),
    ("during_engine_open", "draw_execute", "engine_open", 1, None, "sigint",
     P(stopCode="operator", stopState="DRAW_ACTIVE", completed=0, abortedStage="engine_open")),
    ("during_draw_model_call", "draw_execute", "model_call", 1, None, "sigint",
     P(stopCode="operator", stopState="DRAW_ACTIVE", completed=0, abortedStage="workflow")),
    ("after_draw_evidence_written", "persist:ledger", "after", 1, None, "sigint",
     P(stopCode="operator", stopState="DRAW_RECORDED", completed=1)),
    ("before_manifest_update", "persist:manifest", "before", 1, "commit_draw", "sigint",
     P(stopCode="operator", stopState="DRAW_RECORDED", completed=1)),
    ("between_draws", "draw_guard", "before", 2, None, "sigint",
     P(stopCode="operator", stopState="READY", completed=1)),
    ("during_later_health_check", "health_read", "during", 3, None, "sigint",
     P(stopCode="operator", stopState="HEALTH_CHECK", completed=1, attemptedRead=True)),
    ("during_cleanup", "session_cleanup", "during", 1, None, "sigint",
     P(stopCode="operator", stopState="CLEANUP", completed=2, cleanup="FAILED")),
    ("during_finalization_marker", "persist:marker", "file_fsync", 1, None, "sigint",
     {"classification": "COMPLETE", "stopCode": None, "completed": 2, "exit": 0, "deferredAfterStop": True}),
    ("during_final_manifest_replace", "persist:final_manifest", "replace", 1, None, "sigint",
     {"classification": "COMPLETE", "stopCode": None, "completed": 2, "exit": 0, "deferredAfterStop": True}),
    ("during_ledger_write", "persist:ledger", "file_fsync", 1, None, "sigint",
     P(stopCode="operator", stopState="DRAW_RECORDED", completed=1)),
    # Process termination witnesses (the child signals itself at the point).
    ("kill_during_initial_health", "health_read", "during", 1, None, "self:SIGKILL", dict(INVALID, exit=-9)),
    ("terminate_at_draw_guard", "draw_guard", "before", 1, None, "self:SIGTERM", dict(INVALID, exit=-15)),
    ("kill_during_cleanup", "session_cleanup", "during", 1, None, "self:SIGKILL", dict(INVALID, exit=-9)),
    ("kill_during_final_write", "persist:final_manifest", "replace", 1, None, "self:SIGKILL",
     dict(INVALID, exit=-9, marker=True)),
    ("self_interrupt_after_ledger", "persist:ledger", "after", 1, None, "self:SIGINT",
     P(stopCode="operator", stopState="DRAW_RECORDED", completed=1)),
]


def child(a):
    import v13i_scenarios as S
    fixture = S.Fixture(a.binary)
    boundary, phase, n, event, mode = a.point.split("|")
    n = int(n)
    seen = [0]

    def hook(b, p, ctx):
        if (b, p) != (boundary, phase) or (event and ctx.get("event") != event):
            return
        seen[0] += 1
        if seen[0] != n:
            return
        if mode.startswith("self:"):
            os.kill(os.getpid(), getattr(signal, mode[5:]))
            return
        sys.stdout.write("AT\n")
        sys.stdout.flush()
        sys.stdin.readline()
    clock = S.Clock()
    recorder = {}
    try:
        import v13i as I
        rec = I.Recorder(fixture.proto, fixture.proto_sha, fixture.sched, fixture.ctx, fixture.ctx.opener(a.binary),
                         {S.D.QWEN: S.Model([S.F.ABANDON] * 6)}, a.out, "scripted signal test",
                         state_fn=S.Health(clock), now=S.Monotonic(), utc=clock, hook=hook,
                         run_id="00000000-0000-4000-8000-0000000013a2", provenance=S.SYNTHETIC)
        doc = rec.run()
        recorder = {"returned": True, "state": doc["state"], "deferredAfterStop": rec.deferred_after_stop}
        code = 0
    except I.RecordingError as x:
        recorder, code = {"returned": False, "error": "RecordingError", "cause": type(x.cause).__name__}, 2
    except KeyboardInterrupt:
        recorder, code = {"returned": False, "error": "KeyboardInterrupt"}, 130
    sys.stdout.write("RESULT " + json.dumps(recorder) + "\n")
    sys.stdout.flush()
    raise SystemExit(code)


def parent_case(binary, scratch, case):
    import v13i_scenarios as S
    name, boundary, phase, n, event, mode, expected = case
    out = os.path.join(scratch, name)
    shutil.rmtree(out, ignore_errors=True)
    point = f"{boundary}|{phase}|{n}|{event or ''}|{mode}"
    proc = subprocess.Popen([sys.executable, "-I", "-B", __file__, "child", "--binary", binary, "--out", out,
                             "--point", point], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                            text=True)
    delivered, recorder = False, None
    try:
        for line in proc.stdout:
            if line.strip() == "AT":
                proc.send_signal(signal.SIGINT)
                delivered = True
                time.sleep(0.5)
                try:
                    proc.stdin.write("go\n")
                    proc.stdin.flush()
                except (BrokenPipeError, ValueError):
                    pass
            elif line.startswith("RESULT "):
                recorder = json.loads(line[7:])
        try:
            proc.stdin.close()
        except (BrokenPipeError, OSError):
            pass
        code = proc.wait(timeout=300)
    finally:
        if proc.poll() is None:
            proc.kill()
            proc.wait()
    fixture = S.Fixture(binary)
    obs = S.summary(fixture, out, {"returned": bool(recorder and recorder.get("returned"))})
    obs.pop("recorder", None)
    obs["exit"] = code
    obs["signalDelivered"] = delivered or mode.startswith("self:")
    if recorder:
        obs["deferredAfterStop"] = recorder.get("deferredAfterStop", False)
    live = Path(out) / "live"
    if (live / "run.json").exists():
        doc = json.loads((live / "run.json").read_text())
        stop = doc.get("stop") or {}
        obs["attemptedRead"] = stop.get("request") is not None
    if obs["classification"] in ("COMPLETE", "PARTIAL"):
        try:
            a = fixture.analyze(out)
            obs["analysis"] = {"runStatus": a["runStatus"], "completedVerified": len(a["draws"]),
                               "comparisonsSuppressed": a["armDifferences"] == [] and a["pairs"] == []}
        except S.D.ExperimentError as x:
            obs["analysis"] = {"refused": str(x)[:200]}
    bad = [k for k, v in expected.items() if obs.get(k) != v]
    if not obs["signalDelivered"]:
        bad.append("signal not delivered")
    if obs["classification"] in ("COMPLETE", "PARTIAL") and "refused" in obs.get("analysis", {}):
        bad.append("analysis refused a valid archive")
    shutil.rmtree(out, ignore_errors=True)
    return {"name": name, "point": f"{boundary}.{phase}#{n}" + (f"[{event}]" if event else ""), "signal": mode,
            "expected": expected, "observed": obs, "agree": not bad, "mismatch": bad}


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("run")
    p.add_argument("--binary", required=True)
    p.add_argument("--scratch", required=True)
    p.add_argument("--out", required=True)
    p = sub.add_parser("child")
    p.add_argument("--binary", required=True)
    p.add_argument("--out", required=True)
    p.add_argument("--point", required=True)
    a = ap.parse_args(argv)
    if a.cmd == "child":
        return child(a)
    os.makedirs(a.scratch, exist_ok=True)
    binary = os.path.abspath(a.binary)
    results = [parent_case(binary, os.path.abspath(a.scratch), c) for c in CASES]
    import v13i_scenarios as S
    S.D.write_json(a.out, {"schema": SCHEMA, "notice": "Real signals delivered to a real child process; scripted model and "
                                                       "scripted HTTP; no inference.", "cases": results,
                           "summary": {"cases": len(results), "agree": sum(r["agree"] for r in results)}})
    print(f"signal tests: {sum(r['agree'] for r in results)} of {len(results)} agree")
    for r in results:
        if not r["agree"]:
            print("DISAGREE", r["name"], r["mismatch"], json.dumps(r["observed"], sort_keys=True))
    if not all(r["agree"] for r in results):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
