#!/usr/bin/env python3
"""DT13K-01..07 witnesses, before and after remediation.

Each witness is the independent V0.13k reviewer's construction, ported to an
implementation chosen by --implementation: v13k (the unchanged V0.13k
recorder and verifier, which the witnesses defeat) or v13l (the successor).
Archives are genuine recordings of the chosen implementation; every semantic
mutation recomputes row commitments, the failure-event mirror, counters and
every manifest/ledger/journal/health/side-artifact hash and seal binding
before classification and FULL analysis. Scripted model replies and scripted
health only; no model endpoint.

    python3 -I -B analysis/alternatives/v13l_dt13k.py --binary ./dt --implementation v13l \\
        --scratch DIR --out FILE
"""
import argparse
import copy
import errno
import importlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

EXPECT_AFTER = {
    "DT13K-01": "rejected", "DT13K-02": "rejected", "DT13K-03": "rejected",
    "DT13K-04": "completed result retained", "DT13K-05": "received response retained, not consumed",
    "DT13K-06": "conservative INVALID with evidence retained", "DT13K-07": "generated contexts keep preservation expectations",
}


class Impl:
    def __init__(self, name):
        self.name = name
        self.K = importlib.import_module(name)
        self.V = importlib.import_module(name + "_verify")
        self.S = importlib.import_module(name + "_scenarios")
        self.W = importlib.import_module(name + "_witnesses")
        self.M = importlib.import_module(name + "_mutations")
        self.F = importlib.import_module(name + "_faults")


def write(path, value):
    Path(path).parent.mkdir(parents=True, exist_ok=True)
    Path(path).write_text(json.dumps(value, indent=1, sort_keys=True, default=str) + "\n")


def assess(impl, fixture, out):
    v = fixture.classify(str(out))
    r = {"classification": v["state"], "violations": sorted({x["invariant"] for x in v["violations"]}),
         "preserved": v["preserved"]}
    try:
        a = fixture.analyze(str(out))
        r["analysis"] = {"accepted": True, "runStatus": a["runStatus"], "completedVerified": len(a["draws"]),
                         "abortedVerification": a["abortedVerification"], "armDifferences": len(a["armDifferences"])}
    except Exception as x:  # noqa: BLE001  (any refusal; its type is reported)
        r["analysis"] = {"accepted": False, "refusal": f"{type(x).__module__}.{type(x).__name__}: {x}"[:400]}
    return r


def record(impl, fixture, out, **kw):
    clock = impl.S.Clock()
    return impl.S.attempt(lambda: fixture.record(str(out), clock=clock, health=impl.S.Health(clock), **kw))


def mutate(impl, base, out, fn):
    shutil.copytree(base, out)
    a = impl.M.Arc(out)
    fn(a)
    a.write(commits=True, event=True, reseal=True)
    return a


class FinalSession:
    """Fail the draw's actual final-digests request (run_draw's stage final_digests)."""

    def __init__(self, impl, s):
        self.impl, self.s = impl, s

    def request(self, rid, op, args=None):
        frame = sys._getframe(1)
        while frame:
            if frame.f_code is self.impl.K.H.run_draw.__code__ and frame.f_locals.get("stage") == "final_digests":
                raise BrokenPipeError("actual final-digests failure")
            frame = frame.f_back
        return self.s.request(rid, op, args)

    def close(self):
        self.s.close()


class FinalPool:
    def __init__(self, impl, p):
        self.impl, self.p = impl, p

    def __call__(self, *a):
        return FinalSession(self.impl, self.p(*a))

    def close(self):
        self.p.close()


def rejected(r):
    return not r["analysis"]["accepted"]


# --- DT13K-01: canonical consistency ---------------------------------------------------

def strict_json_cases(impl, binary, scratch, results):
    S = impl.S
    f = S.Fixture(binary, arms=("control", "treatment"))
    base = scratch / "base-complete"
    record(impl, f, base)
    results.append(dict(finding="control", case="complete-positive-control", expected="accepted", **assess(impl, f, base)))

    def req(a):
        return a.rows[0]["calls"][0]["request"]

    def body_edit(a, fn):
        call = a.rows[0]["calls"][0]
        parsed = json.loads(call["requestBody"])
        text = fn(parsed, call["requestBody"])
        call["requestBody"] = text
        call["requestSha256"] = S.D.sha256(text.encode("utf-8"))
        call["request"] = json.loads(text)
        for rec in a.calls:
            if rec.get("draw") == a.rows[0]["seq"] and rec.get("n") == 1 and rec.get("record") != "dispatch":
                rec["requestSha256"] = call["requestSha256"]

    variants = {
        "complete-request-zero-versus-false": lambda a: req(a).update(temperature=False),
        "request-zero-versus-float": lambda a: req(a).update(temperature=0.0),
        "request-numeric-string-versus-number": lambda a: req(a).update(temperature="0"),
        "request-null-versus-missing": lambda a: req(a).update(seed=None),
        "request-missing-key": lambda a: req(a).pop("temperature"),
        "request-nested-message-role-changed": lambda a: req(a)["messages"][0].update(role=["system"]),
        "request-large-integer-structured-only": lambda a: req(a).update(max_tokens=2 ** 64 + 1),
        "request-body-duplicate-key-rehashed": lambda a: body_edit(a, lambda p, t: t[:-1] + ', "temperature": 0}'),
        "request-body-true-versus-one-rehashed": lambda a: body_edit(a, lambda p, t: t.replace('"temperature": 0', '"temperature": false')),
    }
    for name, fn in variants.items():
        out = scratch / name
        mutate(impl, base, out, fn)
        results.append(dict(finding="DT13K-01", case=name, expected="rejected", **assess(impl, f, out)))
    # Reordered keys in the structured request are the same JSON value: accepted.
    out = scratch / "request-reordered-keys-control"
    mutate(impl, base, out, lambda a: a.rows[0]["calls"][0].update(request=dict(reversed(list(req(a).items())))))
    results.append(dict(finding="control", case="request-reordered-keys-control", expected="accepted", **assess(impl, f, out)))
    return f, base


def ordinary_cli(impl, binary, scratch, results):
    """The same contradiction through ordinary CLI analysis of a non-fixture run."""
    K, S, V = impl.K, impl.S, impl.V
    rows, _ = K.H.load_contexts(S.CONTEXTS)
    rows = [r for r in rows if r["id"] == "S1-a"]
    cd = scratch / "cli-contexts"
    cd.mkdir(parents=True, exist_ok=True)
    raw = "".join(json.dumps(r, sort_keys=True) + "\n" for r in rows)
    (cd / "contexts.jsonl").write_text(raw)
    p, _ = K.draft_protocol(rows, S.D.sha256(raw))
    p["modelOrder"] = [K.QWEN]
    p["drawsPerUnit"] = {K.QWEN: {"aa": 1, "unit": 1}}
    sched = K.H.schedule(p, rows)
    write(scratch / "cli-protocol.json", p)
    ph = S.D.sha256((scratch / "cli-protocol.json").read_bytes())
    ctx = K.Contexts(p, rows, None, {K.H.SB.SETTINGS_ENV: K.H.SB.REPLAY_URL})
    replies = [S.F.ABANDON if e["arm"] == "control" else S.F.SEL("no_action") for e in sched]
    source = scratch / "cli-ordinary"
    K.run_schedule(p, ph, sched, ctx, ctx.opener(binary), {K.QWEN: S.Model(replies)}, str(source),
                   "model-free CLI witness", state_fn=S.Health(K.RR.utc), utc=K.RR.utc)
    out = scratch / "cli-ordinary-request-type-contradiction"
    a = mutate(impl, source, out, lambda a: a.rows[0]["calls"][0]["request"].update(temperature=False))
    cmd = [sys.executable, "-I", "-B", str(HERE / (impl.name + "_verify.py")), "analyze", "--binary", binary,
           "--contexts", str(cd), "--protocol", str(scratch / "cli-protocol.json"), "--dir", str(out),
           "--out", str(out / "cli-analysis")]
    run = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
    v = V.classify(str(out), p, ph, sched, ctx.by_id)
    results.append(dict(finding="DT13K-01", case="ordinary-cli-request-zero-versus-false", expected="rejected",
                        classification=v["state"], violations=sorted({x["invariant"] for x in v["violations"]}),
                        analysis={"accepted": run.returncode == 0, "exit": run.returncode,
                                  "refusal": (run.stderr.strip().splitlines() or [""])[-1][:400]}))


# --- DT13K-02 and DT13K-03: causal consistency ---------------------------------------------

def causal_cases(impl, binary, scratch, results):
    S, W = impl.S, impl.W
    f = S.Fixture(binary, arms=("control", "treatment"))
    bases = {}
    for stage in ("initial_digests", "workflow", "engine_close"):
        bases[stage] = scratch / ("base-" + stage)
        record(impl, f, bases[stage], opener=W.FailPool(f.ctx.opener(binary), stage))
    bases["final_digests"] = scratch / "base-actual-final-digests"
    record(impl, f, bases["final_digests"], opener=FinalPool(impl, f.ctx.opener(binary)))

    def raise_model(payload):
        raise RuntimeError("original transport failure")
    bases["raised"] = scratch / "base-raised"
    record(impl, f, bases["raised"], model=raise_model)
    fh = f.with_limits(maxModelCallsPerDraw=0)
    bases["halt_close"] = scratch / "base-halt-close"
    record(impl, fh, bases["halt_close"], opener=W.FailPool(fh.ctx.opener(binary), "engine_close"))
    for name, path in bases.items():
        fx = fh if name == "halt_close" else f
        results.append(dict(finding="control", case="base-" + name, expected="accepted", **assess(impl, fx, path)))

    def engine_error(a):
        a.files["aborted-draw.json"]["engine"][-1]["error"] = "ValueError: a different recorded failure"

    def extra(a):
        a.files["aborted-draw.json"]["engine"][-1]["unexpectedSuccessfulResult"] = {"ok": True}

    def stop_message(a):
        d = a.doc
        d["stop"]["message"] = "ValueError: a different failure than stop.failure"
        d["failureEvent"]["stop"] = copy.deepcopy(d["stop"])

    def raised_failure(a):
        a.calls[1]["exception"] = {"type": "ValueError", "message": "different transport failure"}
        a.files["aborted-draw.json"]["calls"][0]["error"] = "client: ValueError: different transport failure"

    def halt_streak(a):
        h = a.files["aborted-draw.json"]["halt"]
        h["code"], h["message"] = "transport", "5 consecutive transport failures"
        h["evidence"]["consecutiveTransportFailures"] = 5

    def halt_stage(a):
        h = a.files["aborted-draw.json"]["halt"]
        h["code"], h["message"] = "wall_clock", "wall clock"
        h["evidence"].update(stage="draw_start", elapsedSeconds=20000)

    def halt_stage_consistent(a):
        h = a.files["aborted-draw.json"]["halt"]
        h["code"] = "wall_clock"
        h["message"] = "20000 s elapsed; the next draw could pass the 14400 s stop"
        h["evidence"].update(stage="draw_start", elapsedSeconds=20000)

    cases = [
        ("DT13K-02", "initial_digests-engine-error-disagrees-with-primary", "initial_digests", engine_error, f),
        ("DT13K-02", "workflow-engine-error-disagrees-with-primary", "workflow", engine_error, f),
        ("DT13K-02", "final-digests-engine-error-disagrees-with-primary", "final_digests", engine_error, f),
        ("DT13K-02", "initial-digests-failed-request-extra-fields", "initial_digests", extra, f),
        ("DT13K-02", "engine-close-stop-message-disagrees-with-failure", "engine_close", stop_message, f),
        ("DT13K-02", "workflow-raised-call-disagrees-with-primary", "raised", raised_failure, f),
        ("DT13K-03", "halt-close-five-failures-with-zero-calls", "halt_close", halt_streak, fh),
        ("DT13K-03", "halt-close-draw-start-inside-workflow", "halt_close", halt_stage, fh),
        ("DT13K-03", "halt-close-draw-start-with-guard-message", "halt_close", halt_stage_consistent, fh),
    ]
    for finding, name, base, fn, fx in cases:
        out = scratch / name
        mutate(impl, bases[base], out, fn)
        results.append(dict(finding=finding, case=name, expected="rejected", **assess(impl, fx, out)))


# --- DT13K-04: completed result between return and ledger commitment ---------------------------

def completed_boundaries(impl, binary, scratch, results):
    """An exception at the first recorder line after run_draw returns a
    completed result, and at two bookkeeping lines inside run_draw's finally."""
    import linecache
    K, S = impl.K, impl.S
    recorder_file = os.path.abspath(K.__file__)
    for kind in ("first_line_after_return", "draw_finally_before_calls", "draw_finally_before_process"):
        f = S.Fixture(binary)
        out = scratch / ("boundary-" + kind)
        captured, state = [], {"armed": False, "fired": False}

        def trace(frame, event, arg):
            if state["fired"]:
                return None
            code = frame.f_code
            if kind == "first_line_after_return":
                if event == "return" and code is K.H.run_draw.__code__ and isinstance(arg, dict) \
                        and arg.get("status") == "completed":
                    captured.append(copy.deepcopy(arg))
                    state["armed"] = True
                elif event == "line" and state["armed"] and os.path.abspath(code.co_filename) == recorder_file:
                    state["fired"] = True
                    raise RuntimeError("injected at the first recorder line after a completed draw returned")
            elif event == "line" and code is K.H.run_draw.__code__:
                line = linecache.getline(code.co_filename, frame.f_lineno).strip()
                out_ = frame.f_locals.get("out") or {}
                hit = (kind == "draw_finally_before_calls" and line == 'out["calls"] = model.calls if model else []'
                       or kind == "draw_finally_before_process" and line == 'if out["record"] is not None:')
                if hit and out_.get("status") == "completed":
                    value = copy.deepcopy(out_)
                    model = frame.f_locals.get("model")
                    value["calls"] = copy.deepcopy(model.calls) if model else []
                    value["modelCallsCompleted"] = len(value["calls"])
                    value["process"] = {k: value["record"][k] for k in (
                        "eligibility", "exposure", "validationFirstCorrection", "correctionFailure",
                        "deliberateNoAction", "processFailure", "submittedAction", "executedAction")}
                    captured.append(value)
                    state["fired"] = True
                    raise RuntimeError("injected inside run_draw bookkeeping " + kind)
            return trace
        clock = S.Clock()
        sys.settrace(trace)
        try:
            result = S.attempt(lambda: f.record(str(out), clock=clock, health=S.Health(clock)))
        finally:
            sys.settrace(None)
        results.append(dict(finding="DT13K-04", case=kind, expected="completed result retained", fired=state["fired"],
                            **preservation(impl, f, out, result, captured[:1])))


def preservation(impl, f, out, result, captured):
    """Where the exact completed result survives, and whether it replays."""
    S = impl.S
    live = Path(out) / "live"
    found = []
    for path in sorted(live.iterdir()) if live.exists() else []:
        try:
            value = json.loads(path.read_bytes())
        except (ValueError, UnicodeDecodeError, IsADirectoryError):
            continue
        rows = value if isinstance(value, list) else [value.get("draw"), value.get("pending")] if isinstance(value, dict) else []
        for row in rows:
            if isinstance(row, dict) and row.get("status") == "completed":
                found.append((path.name, row))
    reported = (result.get("retainedEvidence") or {}).get("pending") if isinstance(result, dict) else None
    if isinstance(reported, dict) and reported.get("status") == "completed":
        found.append(("RecordingError.retainedEvidence", reported))
    want = S.D.canon(captured[0]) if captured else None
    where = sorted({name for name, row in found if want is not None and S.D.canon(row) == want})
    replay = None
    if where:
        row = next(row for name, row in found if S.D.canon(row) == want)
        pool = f.ctx.opener(f.binary)
        try:
            c = f.ctx.by_id[row["context"]]
            S.D.replay_draw(row, c, f.ctx.settings[row["model"]], pool, f.ctx.memory(c, row["model"]), f.proto)
            replay = "REPLAYED"
        except BaseException as x:  # noqa: BLE001
            replay = f"{type(x).__name__}: {x}"[:300]
        finally:
            pool.close()
    r = assess(impl, f, out)
    accepted_lost = r["analysis"]["accepted"] and r["analysis"].get("completedVerified", 0) < 1
    return dict(recorder=S.recorder_label(result), completedCaptured=bool(captured), exactRetainedIn=where,
                retainedReplay=replay, acceptedWithoutTheCompletedDraw=accepted_lost, **r)


# --- DT13K-05: a returned model response and a transient outcome-write fault -----------------------

def known_response(impl, binary, scratch, results):
    K, S = impl.K, impl.S
    f = S.Fixture(binary)
    marker = "MALFORMED_UNIQUE_KNOWN_RESPONSE"
    for op in ("write", "fsync", "close"):
        out = scratch / ("known-response-" + op)
        state = {"fired": False, "invocations": 0, "active": False}

        class FS(K.OsFS):
            def write(self, fd, data):
                state["active"] = b'"record":"returned"' in data and not state["fired"]
                if op == "write" and state["active"]:
                    state["fired"] = True
                    raise OSError(errno.EIO, "one transient outcome write fault")
                return super().write(fd, data)

            def fsync(self, fd):
                if op == "fsync" and state["active"] and not state["fired"]:
                    state["fired"] = True
                    raise OSError(errno.EIO, "one transient outcome fsync fault")
                return super().fsync(fd)

            def close(self, fd):
                super().close(fd)
                if op == "close" and state["active"] and not state["fired"]:
                    state["fired"] = True
                    raise OSError(errno.EIO, "one transient outcome close fault")

        real = S.Model([marker] + f.replies())

        def model(payload):
            state["invocations"] += 1
            return real(payload)
        clock = S.Clock()
        rec = K.Recorder(f.proto, f.proto_sha, f.sched, f.ctx, f.ctx.opener(binary), {K.QWEN: model}, str(out),
                         "known response preservation", state_fn=S.Health(clock), utc=clock, now=S.Monotonic(),
                         run_id=S.RUN_ID, provenance=S.SYNTHETIC, fs=FS())
        result = S.attempt(rec.run)
        raw = {p.name: p.read_text(errors="replace") for p in (out / "live").iterdir() if p.is_file()}
        journal = S.journal(str(out)) or []
        r = assess(impl, f, out)
        results.append(dict(finding="DT13K-05", case="transient-outcome-" + op + "-fault", fired=state["fired"],
                            expected="received response retained, not consumed",
                            recorder=S.recorder_label(result),
                            knownBodyArchived=any(marker in x for x in raw.values()),
                            knownBodyInStructuredError=marker in json.dumps(result.get("retainedEvidence"), default=str),
                            transportInvocations=state["invocations"],
                            journalDispatched=sum(x.get("record") == "dispatch" for x in journal),
                            files=sorted("failed candidate" if n.startswith((".tmp-", ".failure-")) else n for n in raw), **r))


# --- DT13K-06 and DT13K-07 ---------------------------------------------------------------------

def post_answer_failure(impl, binary, scratch, results):
    K, S = impl.K, impl.S
    f = S.Fixture(binary, arms=("control", "treatment"))
    out = scratch / "successful-final-answer-then-failure"
    fired = [False]

    def trace(frame, event, arg):
        if event == "line" and not fired[0] and frame.f_code is K.H.run_draw.__code__ \
                and frame.f_locals.get("stage") == "final_digests" \
                and (frame.f_locals.get("out") or {}).get("digestsAfter") is not None:
            fired[0] = True
            raise RuntimeError("after final answer before engine log assignment")
        return trace
    clock = S.Clock()
    sys.settrace(trace)
    try:
        result = S.attempt(lambda: f.record(str(out), clock=clock, health=S.Health(clock)))
    finally:
        sys.settrace(None)
    live = out / "live"
    ab = json.loads((live / "aborted-draw.json").read_text()) if (live / "aborted-draw.json").exists() else None
    r = assess(impl, f, out)
    results.append(dict(finding="DT13K-06", case="successful-final-answer-then-failure", fired=fired[0],
                        expected="conservative INVALID with evidence retained", recorder=S.recorder_label(result),
                        retainedDigestsAfter=bool(ab and ab.get("digestsAfter")), **r))


def generated_contexts(impl, binary, scratch, results):
    """DT13K-07: the generated fault-case contexts and the preservation oracle."""
    F = impl.F
    spec, bases, cases, declared, reached = F.enumerate_cases(binary, str(scratch / "matrix"), bases=["A"])
    missing = sum("pendingRetained" not in c["context"] for c in cases)
    case = next(c for c in cases if c["boundary"] == "persist:ledger" and c["phase"] == "file_fsync"
                and c["kind"] == "persistence")
    ctx = dict(case["context"])
    obs = {"recorder": "RecordingError", "classification": "INVALID", "sealed": False, "ledgerRows": 0,
           "ledgerCompleted": 0, "pendingCompleted": 0, "archiveCompletedRowSha256": [],
           "journalDispatched": ctx["dispatched"], "reportedCompleted": 1, "reportedRowSha256": "1" * 64,
           "receivedResponses": 0}
    generated = F.agree({}, obs, ctx)
    dropped = F.agree({}, obs, {k: v for k, v in ctx.items() if k != "pendingRetained"})
    results.append(dict(finding="DT13K-07", case="generated-context-preservation-oracle",
                        expected="generated contexts keep preservation expectations",
                        cases=len(cases), contextsMissingPendingRetained=missing,
                        lostDurableHandoffDetectedWithGeneratedContext=bool(generated),
                        droppedExpectationDetected=bool(dropped), generatedContext=ctx))


def agree_after(r):
    finding = r["finding"]
    a = r.get("analysis") or {}
    if finding == "control":
        return a.get("accepted") is True
    if finding in ("DT13K-01", "DT13K-02", "DT13K-03"):
        return a.get("accepted") is False
    if finding == "DT13K-04":
        return bool(r["fired"] and r["exactRetainedIn"] and r["retainedReplay"] == "REPLAYED"
                    and not r["acceptedWithoutTheCompletedDraw"])
    if finding == "DT13K-05":
        return bool(r["fired"] and (r["knownBodyArchived"] or r["knownBodyInStructuredError"])
                    and r["classification"] == "INVALID" and not a.get("accepted")
                    and r["transportInvocations"] == r["journalDispatched"])
    if finding == "DT13K-06":
        return bool(r["fired"] and r["classification"] == "INVALID" and not a.get("accepted") and r["retainedDigestsAfter"])
    if finding == "DT13K-07":
        return bool(r["contextsMissingPendingRetained"] == 0 and r["lostDurableHandoffDetectedWithGeneratedContext"]
                    and r["droppedExpectationDetected"])
    return False


def run(binary, scratch, implementation):
    impl = Impl(implementation)
    scratch = Path(scratch)
    scratch.mkdir(parents=True, exist_ok=True)
    os.environ["TMPDIR"] = str(scratch)
    results = []
    strict_json_cases(impl, binary, scratch, results)
    ordinary_cli(impl, binary, scratch, results)
    causal_cases(impl, binary, scratch, results)
    completed_boundaries(impl, binary, scratch, results)
    known_response(impl, binary, scratch, results)
    post_answer_failure(impl, binary, scratch, results)
    generated_contexts(impl, binary, scratch, results)
    for r in results:
        r["meetsRemediationExpectation"] = agree_after(r)
    by = {}
    for r in results:
        by.setdefault(r["finding"], []).append(r["meetsRemediationExpectation"])
    return {"schema": "dreamteam.v13l-dt13k-witnesses/v1", "implementation": implementation,
            "expectationAfterRemediation": EXPECT_AFTER, "cases": results,
            "summary": {k: {"cases": len(v), "meetExpectation": sum(v)} for k, v in sorted(by.items())},
            "allMeetExpectation": all(r["meetsRemediationExpectation"] for r in results)}


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--implementation", choices=("v13k", "v13l"), required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    doc = run(str(Path(a.binary).resolve()), a.scratch, a.implementation)
    write(a.out, doc)
    print(json.dumps(doc["summary"]), "all:", doc["allMeetExpectation"])
    if a.implementation == "v13l" and not doc["allMeetExpectation"]:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
