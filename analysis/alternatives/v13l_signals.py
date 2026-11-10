#!/usr/bin/env python3
"""Real-process signal and termination tests at lifecycle boundaries.

A child process records a base scenario. At the selected point (the i-th
point the base trace reaches) its hook either announces itself and blocks on
stdin while the parent delivers a real signal with os.kill (SIGINT to the
child, SIGINT to the child's whole process group, or SIGKILL), or the child
signals itself (SIGKILL, SIGTERM). Trace-line cases deliver a real SIGINT at a
named source line inside the unchanged V0.13h draw and V0.12b transport
bookkeeping (the DT13I-01 boundaries). The parent then classifies and
analyses the archive independently. Expected outcomes come from the same
contract rule as the fault matrix (kind signal or kill), computed from the
base trace, never from the faulted run.

    python3 -I -B analysis/alternatives/v13l_signals.py run --binary ./dt --scratch DIR --out FILE [--jobs N]
"""
import argparse
import concurrent.futures
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

SCHEMA = "dreamteam.v13l-signal-tests/v1"

# External SIGINT points (label selection in base A): every safe point, the transport windows,
# persistence phases and every finalization phase.
EXTERNAL = [
    ("safe_point", "before_health_check", 1), ("health_read", "during", 1), ("safe_point", "after_health_check", 1),
    ("safe_point", "between_draws", 1), ("safe_point", "before_health_check", 2), ("health_read", "during", 2),
    ("safe_point", "after_health_check", 2), ("draw_execute", "engine_open", 1), ("safe_point", "before_dispatch", 1),
    ("append:calls", "write", 1), ("draw_execute", "model_call", 1), ("safe_point", "after_transport_returns", 1),
    ("append:calls", "file_fsync", 2), ("safe_point", "after_call_persisted", 1), ("draw_execute", "after", 1),
    ("persist:ledger", "replace", 1), ("persist:manifest", "dir_fsync", 9), ("safe_point", "after_draw_persisted", 1),
    ("safe_point", "between_draws", 2), ("draw_execute", "model_call", 2), ("safe_point", "after_draw_persisted", 2),
    ("safe_point", "before_cleanup", 1), ("session_cleanup", "during", 1), ("safe_point", "after_cleanup", 1),
    ("safe_point", "before_finalization", 1), ("persist:manifest", "write", 14), ("seal", "before", 1),
    ("seal", "write", 1), ("seal", "file_fsync", 1), ("seal", "rename", 1), ("seal", "renamed", 1),
    ("seal", "dir_fsync", 1),
]
EXTERNAL += [("persist:handoff", phase, 1) for phase in ("before", "write", "file_fsync", "replace", "replaced", "dir_fsync", "after")]
EXTERNAL += [("handoff_clear", phase, 1) for phase in ("before", "after")]

GROUP = [("health_read", "during", 1), ("draw_execute", "model_call", 1), ("safe_point", "between_draws", 2),
         ("session_cleanup", "during", 1)]
EXTERNAL_KILL = [("append:calls", "write", 1), ("draw_execute", "model_call", 1), ("persist:ledger", "replace", 1),
                 ("persist:manifest", "write", 14), ("seal", "file_fsync", 1), ("seal", "rename", 1), ("seal", "renamed", 1)]
TERM = [("draw_execute", "model_call", 1), ("seal", "rename", 1), ("seal", "renamed", 1)]
# DT13I-01: real SIGINT on a named line inside the unchanged transport and draw bookkeeping.
TRACE_LINES = [
    ("after_transport_returns_in_point_model", "stage_b.py", "call", "lat = self.now() - t0"),
    ("before_calls_append_in_point_model", "stage_b.py", "call", "f = body_fields(body)"),
    ("draw_finally_bookkeeping", "v13h.py", "run_draw", 'out["calls"] = model.calls if model else []'),
    ("draw_return", "v13h.py", "run_draw", "return out"),
]


def child(a):
    import v13l_scenarios as S
    import v13l_faults as M
    fixture = S.Fixture(a.binary)
    parts = M.Parts(a.base, fixture)
    seen = [0]
    closes = []

    def hook(b, p, ctx):
        i = seen[0]
        seen[0] += 1
        if i != a.index:
            return
        if a.mode.startswith("self:"):
            os.kill(os.getpid(), getattr(signal, a.mode[5:]))
            return
        sys.stdout.write("AT\n")
        sys.stdout.flush()
        sys.stdin.readline()
    if a.trace:
        fname, func, text = a.trace.split("|", 2)
        fired = [False]

        def tracer(frame, event, arg):
            if fired[0]:
                return None
            if frame.f_code.co_name == func and os.path.basename(frame.f_code.co_filename) == fname:
                if event == "line":
                    import linecache
                    line = linecache.getline(frame.f_code.co_filename, frame.f_lineno).strip()
                    if line == text:
                        fired[0] = True
                        os.kill(os.getpid(), signal.SIGINT)
                        return None
                return tracer
            return tracer if event == "call" else None
        sys.settrace(tracer)
    opener = parts.opener

    class Counting:
        def __call__(self, c, arm):
            return opener(c, arm)

        def close(self):
            closes.append(1)
            return opener.close()
    try:
        r = parts.fixture.record(a.out, health=parts.health, opener=Counting(), model=parts.model, hook=hook,
                                 now=parts.now, clock=parts.clock, fs=parts.fs)
        result = {"returned": True, "state": r["state"], "interrupts": r["interrupts"], "commit": r["commit"]}
        code = 0
    except S.J.RecordingError as x:
        result, code = {"returned": False, "error": "RecordingError", "interrupts": x.interrupts}, 2
    except KeyboardInterrupt:
        result, code = {"returned": False, "error": "KeyboardInterrupt"}, 130
    finally:
        sys.settrace(None)
    result["poolReleases"] = len(closes)
    result["transportCalls"] = getattr(parts.model, "calls", None)
    sys.stdout.write("RESULT " + json.dumps(result) + "\n")
    sys.stdout.flush()
    raise SystemExit(code)


def launch(binary, out, base, index, mode, delivery, trace=None):
    """Run the child; deliver the signal; return (exit code, child result)."""
    cmd = [sys.executable, "-I", "-B", __file__, "child", "--binary", binary, "--out", out, "--base", base,
           "--index", str(index), "--mode", mode]
    if trace:
        cmd += ["--trace", trace]
    proc = subprocess.Popen(cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True,
                            start_new_session=True)
    result, delivered = None, delivery in ("self", "trace")
    try:
        for line in proc.stdout:
            if line.strip() == "AT":
                if delivery == "group":
                    os.killpg(proc.pid, signal.SIGINT)
                else:
                    os.kill(proc.pid, getattr(signal, mode))
                delivered = True
                time.sleep(0.3)
                try:
                    proc.stdin.write("go\n")
                    proc.stdin.flush()
                except (BrokenPipeError, ValueError, OSError):
                    pass
            elif line.startswith("RESULT "):
                result = json.loads(line[7:])
        try:
            proc.stdin.close()
        except (BrokenPipeError, OSError):
            pass
        code = proc.wait(timeout=300)
    finally:
        if proc.poll() is None:
            proc.kill()
            proc.wait()
    return code, result, delivered


def case_list(binary, scratch):
    import lifecycle_v4_spec as LS
    import v13l_faults as M
    spec = LS.load()
    traces = {base: M.trace(binary, scratch, base)[0] for base in ("A", "B", "C")}

    def find(hits, b, p, n):
        k = 0
        for i, (hb, hp, _) in enumerate(hits):
            if (hb, hp) == (b, p):
                k += 1
                if k == n:
                    return i
        raise SystemExit(f"point {b}.{p}#{n} not reached")
    cases = []
    hits = traces["A"]
    for b, p, n in EXTERNAL:
        i = find(hits, b, p, n)
        cases.append({"name": f"sigint:{b}.{p}#{n}", "base": "A", "index": i, "mode": "SIGINT", "delivery": "external",
                      "expected": M.expect(spec, "A", hits, i, "signal"), "context": hits[i][2]})
    for b, p, n in GROUP:
        i = find(hits, b, p, n)
        cases.append({"name": f"sigint-process-group:{b}.{p}#{n}", "base": "A", "index": i, "mode": "SIGINT",
                      "delivery": "group", "expected": M.expect(spec, "A", hits, i, "signal"), "context": hits[i][2]})
    for b, p, n in EXTERNAL_KILL:
        i = find(hits, b, p, n)
        cases.append({"name": f"sigkill-external:{b}.{p}#{n}", "base": "A", "index": i, "mode": "SIGKILL",
                      "delivery": "external", "expected": dict(M.expect(spec, "A", hits, i, "kill"), exit=-9),
                      "context": hits[i][2]})
    for b, p, n in TERM:
        i = find(hits, b, p, n)
        cases.append({"name": f"sigterm-self:{b}.{p}#{n}", "base": "A", "index": i, "mode": "self:SIGTERM",
                      "delivery": "self", "expected": dict(M.expect(spec, "A", hits, i, "kill"), exit=-15),
                      "context": hits[i][2]})
    for base in ("A", "B", "C"):
        bh = traces[base]
        for i, (b, p, ctx) in enumerate(bh):
            if base != "A" and not (ctx["stopping"] or ctx["state"] in ("CLEANUP", "FINALIZING")):
                continue
            name, _ = M.label(bh, i)
            cases.append({"name": f"sigkill-self:{base}:{i:03d}:{name}", "base": base, "index": i, "mode": "self:SIGKILL",
                          "delivery": "self", "expected": dict(M.expect(spec, base, bh, i, "kill"), exit=-9),
                          "context": ctx})
    for name, fname, func, text in TRACE_LINES:
        cases.append({"name": f"sigint-line:{name}", "base": "A", "index": -1, "mode": "SIGINT", "delivery": "trace",
                      "trace": f"{fname}|{func}|{text}",
                      "expected": dict(M.P("operator", "DRAW_RECORDED", 1, 1), recorder="returned", sealed=True,
                                       safePoint="after_draw_persisted", interruptDispositions=["STOP"], unarchived=0,
                                       transportCalls=1),
                      "context": None})
    return cases


def run_case(binary, scratch, case):
    import v13l_scenarios as S
    import v13l_faults as M
    out = os.path.join(scratch, case["name"].replace(":", "_").replace("/", "_").replace("#", "_").replace("[", "_")
                       .replace("]", "_"))
    shutil.rmtree(out, ignore_errors=True)
    code, result, delivered = launch(binary, out, case["base"], case["index"], case["mode"], case["delivery"],
                                     case.get("trace"))
    fixture = M.Parts(case["base"], S.Fixture(binary)).fixture
    obs = S.summary(fixture, out, {"returned": bool(result and result.get("returned"))})
    obs["recorder"] = ("returned" if result.get("returned") else result.get("error")) if result else "killed"
    obs["exit"] = code
    obs["signalDelivered"] = delivered
    # Durable journal outcomes and retained received responses (R-RESPONSE-003).
    obs["journalOutcomeKs"] = sorted({r.get("k") for r in (S.journal(out) or [])
                                      if r.get("record") in ("returned", "raised") and isinstance(r.get("k"), int)})
    report = Path(out) / "live" / "persistence-failure.json"
    kept = []
    if report.exists():
        try:
            kept = json.loads(report.read_bytes()).get("receivedResponses") or []
        except (ValueError, AttributeError):
            kept = []
    retained = (result or {}).get("retainedEvidence") if isinstance(result, dict) else None
    if isinstance(retained, dict):
        kept = kept + list(retained.get("receivedResponses") or [])
    obs["receivedKs"] = sorted({r.get("k") for r in kept if isinstance(r, dict) and isinstance(r.get("k"), int)})
    if result:
        if result.get("interrupts") is not None:
            obs["unarchived"] = result["interrupts"]["unarchived"]
            obs["signalsReceived"] = result["interrupts"]["signalsReceived"]
        if result.get("commit") is not None:
            obs["directorySync"] = result["commit"]["directorySync"]
        obs["poolReleases"] = result.get("poolReleases")
        obs["transportCalls"] = result.get("transportCalls")
    if obs["classification"] in ("COMPLETE", "PARTIAL"):
        try:
            a = fixture.analyze(out)
            obs["analysis"] = {"accepted": True, "runStatus": a["runStatus"], "completedVerified": len(a["draws"]),
                               "comparisonsSuppressed": a["armDifferences"] == [] and a["pairs"] == []}
        except S.D.ExperimentError as x:
            obs["analysis"] = {"accepted": False, "refusal": str(x)[:200]}
    expected = dict(case["expected"])
    if case["mode"] == "SIGINT" or case["delivery"] == "trace":
        expected.setdefault("exit", 0)
    if case["delivery"] in ("external", "self") and case["mode"] != "SIGINT":
        expected.pop("recorder", None)
    bad = M.agree(expected, obs, case["context"])
    if not delivered:
        bad.append("signal not delivered")
    if obs["recorder"] == "returned" and obs.get("poolReleases") != 1:
        bad.append("the pooled sessions were not released exactly once")
    if case["mode"] == "SIGINT" and obs.get("signalsReceived") not in (None, 1):
        bad.append("the interrupt is not reported once in the public result")
    shutil.rmtree(out, ignore_errors=True)
    obs.pop("transitions", None)
    return {"name": case["name"], "base": case["base"], "index": case["index"], "signal": case["mode"],
            "delivery": case["delivery"], "expected": expected, "observed": obs, "agree": not bad, "mismatch": bad}


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("run")
    p.add_argument("--binary", required=True)
    p.add_argument("--scratch", required=True)
    p.add_argument("--out", required=True)
    p.add_argument("--jobs", type=int, default=4)
    p.add_argument("--only")
    p = sub.add_parser("child")
    p.add_argument("--binary", required=True)
    p.add_argument("--out", required=True)
    p.add_argument("--base", required=True)
    p.add_argument("--index", type=int, required=True)
    p.add_argument("--mode", required=True)
    p.add_argument("--trace")
    a = ap.parse_args(argv)
    if a.cmd == "child":
        return child(a)
    os.makedirs(a.scratch, exist_ok=True)
    binary, scratch = os.path.abspath(a.binary), os.path.abspath(a.scratch)
    cases = case_list(binary, scratch)
    if a.only:
        cases = [c for c in cases if c["name"].startswith(a.only)]
    with concurrent.futures.ThreadPoolExecutor(max_workers=a.jobs) as pool:
        results = list(pool.map(lambda c: run_case(binary, scratch, c), cases))
    import v13l_scenarios as S
    kinds = {}
    for r in results:
        k = r["name"].split(":")[0]
        kinds.setdefault(k, [0, 0])
        kinds[k][0] += 1
        kinds[k][1] += r["agree"]
    S.D.write_json(a.out, {"schema": SCHEMA, "notice": "Real signals delivered to real child processes; scripted model "
                                                        "and scripted HTTP; no inference.", "cases": results,
                           "summary": {"cases": len(results), "agree": sum(r["agree"] for r in results),
                                       "byKind": {k: {"cases": v[0], "agree": v[1]} for k, v in sorted(kinds.items())},
                                       "falseComplete": sum("FALSE COMPLETE" in r["mismatch"] for r in results)}})
    print(f"signal tests: {sum(r['agree'] for r in results)} of {len(results)} agree")
    for r in results:
        if not r["agree"]:
            print("DISAGREE", r["name"], r["mismatch"])
    if not all(r["agree"] for r in results):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
