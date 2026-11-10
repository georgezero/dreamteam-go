#!/usr/bin/env python3
"""DT13L-01..04 witnesses and the fail-closed persistence policy, before and
after V0.13m-lite.

Each witness ports the independent V0.13l reviewer's construction
(response-envelope JSON ambiguity, impossible wall-clock halt messages,
nested recovery failures) to an implementation chosen by --implementation:

* v13l    the unchanged V0.13l recorder and verifier;
* prefix  the pre-fix successor: the unchanged V0.13l recorder and verifier
          mechanically renamed to the V0.13m identities (materialized in the
          scratch directory; reproducible from the unchanged V0.13l files);
* v13m    the successor.

Archives are genuine recordings of the chosen implementation. Every semantic
mutation recomputes the response hash, the call-journal copy, the workflow's
delivery copy, row commitments, the failure-event mirror, counters and every
manifest/ledger/journal/health/side-artifact hash and seal binding before
classification and FULL analysis. Genuine-provenance witnesses are analysed
through the ordinary command line. Scripted model replies and scripted health
only; no model endpoint.

    python3 -I -B analysis/alternatives/v13m_dt13l.py --binary ./dt --implementation v13m \\
        --scratch DIR --out FILE
"""
import argparse
import copy
import datetime
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
from unittest import mock

HERE = Path(__file__).resolve().parent
SCHEMA = "dreamteam.v13m-dt13l-witnesses/v1"
RENAMES = [("v13l", "v13m"), ("dreamteam-v13m-driver/8", "dreamteam-v13m-driver/9"),
           ("alternatives-experiment/v8", "alternatives-experiment/v9"), ("alternatives-run/v8", "alternatives-run/v9"),
           ("alternatives-analysis/v9", "alternatives-analysis/v10"),
           ("alternatives-partial-run/v8", "alternatives-partial-run/v9"),
           ("partial-run procedure /8", "partial-run procedure /9"),
           ("failed-persistence-retention/v2", "failed-persistence-retention/v3")]


def prefix_successor(scratch):
    """The unchanged V0.13l recorder and verifier, renamed to the successor identities."""
    d = Path(scratch) / "prefix-modules"
    d.mkdir(parents=True, exist_ok=True)
    for src, dst in (("v13l.py", "v13m.py"), ("v13l_verify.py", "v13m_verify.py")):
        text = (HERE / src).read_text(encoding="utf-8")
        for a, b in RENAMES:
            text = text.replace(a, b)
        (d / dst).write_text(text, encoding="utf-8")
    shutil.copyfile(HERE / "bindings_v2.json", d / "bindings_v2.json")
    return d


class Impl:
    def __init__(self, name, scratch):
        import importlib
        self.name, self.prefix_dir = name, None
        if name == "prefix":
            self.prefix_dir = prefix_successor(scratch)
            sys.path.insert(0, str(self.prefix_dir))
            base = "v13m"
        else:
            base = name
        sys.path.insert(len(sys.path) if name == "prefix" else 0, str(HERE))
        self.K = importlib.import_module(base)
        self.V = importlib.import_module(base + "_verify")
        if name == "prefix":
            assert Path(self.K.__file__).parent == self.prefix_dir and Path(self.V.__file__).parent == self.prefix_dir
        self.S = importlib.import_module(base + "_scenarios")
        self.M = importlib.import_module(base + "_mutations")
        self.W = importlib.import_module(base + "_witnesses")

    def verify_command(self):
        if self.prefix_dir is None:
            return [sys.executable, "-I", "-B", str(HERE / (self.name + "_verify.py"))]
        boot = ("import runpy,sys;sys.path[:0]=[%r,%r];sys.argv=[%r]+sys.argv[1:];runpy.run_path(%r,run_name='__main__')"
                % (str(self.prefix_dir), str(HERE), str(self.prefix_dir / "v13m_verify.py"),
                   str(self.prefix_dir / "v13m_verify.py")))
        return [sys.executable, "-I", "-B", "-c", boot]


def write(path, value):
    Path(path).parent.mkdir(parents=True, exist_ok=True)
    Path(path).write_text(json.dumps(value, indent=1, sort_keys=True, default=str) + "\n")


def scrub(text):
    return str(text).replace(str(HERE.parents[1]), "<repo>")


def assess(impl, fixture, out):
    v = fixture.classify(str(out))
    r = {"classification": v["state"], "violations": sorted({x["invariant"] for x in v["violations"]}),
         "preserved": v["preserved"]}
    try:
        a = fixture.analyze(str(out))
        r["analysis"] = {"accepted": True, "runStatus": a["runStatus"], "completedVerified": len(a["draws"]),
                         "abortedVerification": a["abortedVerification"], "armDifferences": a["armDifferences"]}
    except Exception as x:  # noqa: BLE001  (any refusal; its type is reported)
        r["analysis"] = {"accepted": False, "refusal": scrub(f"{type(x).__name__}: {x}")[:400]}
    return r


def record(impl, fixture, out, **kw):
    clock = kw.pop("clock", None) or impl.S.Clock()
    return impl.S.attempt(lambda: fixture.record(str(out), clock=clock, health=impl.S.Health(clock), **kw))


def rebind_body(arc, k, fn, fields):
    """Change the k-th returned response body everywhere it is retained, with the
    structured metadata the recorder's (permissive) workflow parser derives from it."""
    seen = 0
    for r in arc.rows:
        for call in r["calls"]:
            if call.get("body") is None:
                continue
            seen += 1
            if seen != k:
                continue
            old, new = call["body"], fn(call["body"])
            call["body"], call["responseSha256"] = new, impl_sha(new)
            f = fields(new)
            for field in ("usage", "servedModel", "finishReason"):
                call[field] = f.get(field) if f.get("parsedBody") else None
            for d in (r.get("record") or {}).get("deliveries") or []:
                if d.get("body") == old:
                    d["body"] = new
            for rec in arc.calls:
                if rec.get("record") == "returned" and rec.get("draw") == r["seq"] and rec.get("n") == call["n"]:
                    rec.update(body=new, responseSha256=call["responseSha256"])
            return
    raise AssertionError("no such response")


def impl_sha(text):
    import hashlib
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def mutate(impl, base, out, fn):
    shutil.copytree(base, out)
    a = impl.M.Arc(out)
    fn(a)
    a.write(commits=True, event=True, reseal=True)
    return a


# --- DT13L-01: model-response JSON ----------------------------------------------------------

BODY_CASES = {
    # name: (change of the response body text, expectation)
    "duplicate-choices": (lambda b: '{"choices":[], ' + b[1:], "rejected"),
    "escaped-duplicate-choices": (lambda b: '{"choic\\u0065s":[], ' + b[1:], "rejected"),
    "duplicate-nested": (lambda b: b[:-1] + ',"unconsumed":{"a":0,"a":false}}', "rejected"),
    "duplicate-deep-nested": (lambda b: b[:-1] + ',"unconsumed":[{"x":[{"k":1,"\\u006b":1}]}]}', "rejected"),
    "duplicate-usage-key": (lambda b: b[:-1] + ',"usage":{"total_tokens":1},"usage":{"total_tokens":2}}', "rejected"),
    "nan-unused": (lambda b: b[:-1] + ',"unconsumed":NaN}', "rejected"),
    "infinity-unused": (lambda b: b[:-1] + ',"unconsumed":Infinity}', "rejected"),
    "negative-infinity-unused": (lambda b: b[:-1] + ',"unconsumed":-Infinity}', "rejected"),
    "overflow-unused": (lambda b: b[:-1] + ',"unconsumed":1e999}', "rejected"),
    "negative-overflow-unused": (lambda b: b[:-1] + ',"unconsumed":-1e400}', "rejected"),
    "unicode-literal-control": (lambda b: b[:-1] + ',"unconsumed":"é中\U0001f600"}', "accepted"),
    "unicode-escape-control": (lambda b: b[:-1] + ',"unconsumed":"\\u00e9\\ud83d\\ude00"}', "accepted"),
    "large-integer-control": (lambda b: b[:-1] + ',"unconsumed":%d}' % (2 ** 80 + 1), "accepted"),
    "numeric-forms-control": (lambda b: b[:-1] + ',"unconsumed":[-0.0,1E+2,1e308,5e-324,-0]}', "accepted"),
    "valid-usage-control": (lambda b: b[:-1] + ',"usage":{"prompt_tokens":10,"completion_tokens":3}}', "accepted"),
}


def strict_response_cases(impl, binary, scratch, results):
    S = impl.S
    f = S.Fixture(binary, arms=("control", "treatment"))
    base = scratch / "base-complete"
    record(impl, f, base)
    r = assess(impl, f, base)
    a = r["analysis"]
    results.append(dict(finding="control", case="complete-positive-control", expected="accepted", **r))
    # The primary-outcome counterexample: control abandonment and treatment select:no_action both submit [].
    rows = json.loads((base / "live" / "draws.json").read_text())
    results.append(dict(finding="primary", case="primary-outcome-counterexample", expected="zero primary difference",
                        submitted=[x["batches"] for x in rows], routes=[x["record"]["outcome"]["final"] for x in rows],
                        armDifferences=a.get("armDifferences"), analysisAccepted=a.get("accepted")))
    for name, (fn, expect) in BODY_CASES.items():
        out = scratch / ("resealed-" + name)
        mutate(impl, base, out, lambda arc, fn=fn: rebind_body(arc, 1, fn, impl.K.SB.body_fields))
        results.append(dict(finding="DT13L-01", case="resealed-" + name, expected=expect, **assess(impl, f, out)))
    # Genuine recordings of altered wire bytes (the first response only).
    for name, (fn, expect) in list(BODY_CASES.items()) + [
            ("trailing-content", (lambda b: b + ' {"choices":[]}', "accepted-unparsed")),
            ("truncated", (lambda b: b[:-1], "accepted-unparsed"))]:
        real, seen = S.Model(f.replies()), [0]

        def model(payload, fn=fn, real=real, seen=seen):
            status, body, error = real(payload)
            seen[0] += 1
            return (status, fn(body), error) if seen[0] == 1 else (status, body, error)
        out = scratch / ("wire-" + name)
        rec = record(impl, f, out, model=model)
        row = dict(finding="DT13L-01", case="wire-" + name, expected=expect, recorder=rec.get("state") or rec.get("error"),
                   **assess(impl, f, out))
        if expect == "accepted-unparsed":
            calls = json.loads((out / "live" / "draws.json").read_text())[0]["calls"] if (out / "live" / "draws.json").exists() else []
            row["firstCallMetadata"] = {k: calls[0].get(k) for k in ("usage", "servedModel", "finishReason")} if calls else None
        results.append(row)


def cli_setup(impl, scratch):
    K, S = impl.K, impl.S
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
    return dict(cd=cd, p=p, ph=ph, sched=sched, ctx=ctx, replies=replies)


def cli_analyze(impl, scratch, cli, out):
    cmd = impl.verify_command() + ["analyze", "--binary", cli["binary"], "--contexts", str(cli["cd"]),
                                   "--protocol", str(scratch / "cli-protocol.json"), "--dir", str(out),
                                   "--out", str(out / "cli-analysis")]
    run = subprocess.run(cmd, capture_output=True, text=True, timeout=900)
    v = impl.V.classify(str(out), cli["p"], cli["ph"], cli["sched"], cli["ctx"].by_id)
    return dict(classification=v["state"], violations=sorted({x["invariant"] for x in v["violations"]}),
                analysis={"accepted": run.returncode == 0, "exit": run.returncode,
                          "stdout": scrub(run.stdout.strip())[:300],
                          "refusal": scrub((run.stderr.strip().splitlines() or [""])[-1])[:400]})


class LinkedClock:
    """A UTC clock that advances with the run's monotonic clock (as real clocks
    do), plus one microsecond per reading; recorded as SYSTEM_UTC."""

    def __init__(self, mono):
        self.mono, self.n = mono, 0
        self.base = datetime.datetime(2026, 11, 1, tzinfo=datetime.timezone.utc)

    def __call__(self):
        self.n += 1
        t = self.base + datetime.timedelta(seconds=self.mono(), microseconds=self.n)
        return t.strftime("%Y-%m-%dT%H:%M:%S.%fZ")


class Mono:
    def __init__(self):
        self.value = 0.0

    def __call__(self):
        return self.value


def ordinary_cli_cases(impl, binary, scratch, results):
    """Genuine-provenance recordings analysed through the ordinary command line."""
    K, S = impl.K, impl.S
    cli = cli_setup(impl, scratch)
    cli["binary"] = binary

    def run(name, model=None, hook=None, now=None, utc=None):
        out = scratch / name
        K.run_schedule(cli["p"], cli["ph"], cli["sched"], cli["ctx"], cli["ctx"].opener(binary),
                       {K.QWEN: model or S.Model(cli["replies"])}, str(out), "model-free CLI witness",
                       state_fn=S.Health(utc or K.RR.utc), utc=utc or K.RR.utc, now=now or K.SB.time.monotonic, hook=hook)
        return out

    out = run("cli-control")
    results.append(dict(finding="control", case="ordinary-cli-genuine-control", expected="accepted",
                        **cli_analyze(impl, scratch, cli, out)))
    real = S.Model(cli["replies"])

    def duplicate(payload):
        status, body, error = real(payload)
        return status, '{"choices":[], ' + body[1:], error
    out = run("cli-duplicate-response", model=duplicate)
    results.append(dict(finding="DT13L-01", case="ordinary-cli-duplicate-response", expected="rejected",
                        **cli_analyze(impl, scratch, cli, out)))

    # DT13L-02 with genuine provenance. (a) The reviewer's construction: real UTC,
    # a monotonic jump, then the message rewritten to "0 s elapsed".
    mono = S.Monotonic()

    def jump(b, phase, c, mono=mono):
        if (b, phase) == ("draw_guard", "during"):
            mono.jump()
    base = run("cli-wall-base-unlinked", hook=jump, now=mono)
    out = scratch / "cli-wall-zero-message"
    mutate(impl, base, out, lambda a: zero_message(a))
    results.append(dict(finding="DT13L-02", case="ordinary-cli-wall-zero-message", expected="rejected",
                        **cli_analyze(impl, scratch, cli, out)))
    results.append(dict(finding="DT13L-02", case="ordinary-cli-wall-elapsed-unlinked-to-timestamps", expected="rejected",
                        **cli_analyze(impl, scratch, cli, base)))
    # (b) A consistent genuine wall-clock halt: the UTC clock advances with the monotonic one.
    stop = cli["p"]["stop"]["wallClockSeconds"]
    threshold = stop - cli["p"]["request"]["timeoutSeconds"]
    for name, value in (("just-past-threshold", threshold + 0.2), ("well-past-threshold", threshold + 3600.7)):
        m = Mono()

        def hop(b, phase, c, m=m, value=value):
            if (b, phase) == ("draw_guard", "during") and m.value == 0.0:
                m.value = value
        out = run("cli-wall-linked-" + name, hook=hop, now=m, utc=LinkedClock(m))
        results.append(dict(finding="DT13L-02", case="ordinary-cli-wall-linked-" + name, expected="accepted",
                            message=json.loads((out / "live" / "run.json").read_text())["stop"]["message"],
                            **cli_analyze(impl, scratch, cli, out)))
        if name == "well-past-threshold":
            for mname, fn in (("elapsed-beyond-timestamps", lambda a: shift_elapsed(a, 7200)),
                              ("message-below-threshold", lambda a: set_message_seconds(a, threshold - 1))):
                mout = scratch / ("cli-wall-linked-" + mname)
                mutate(impl, out, mout, fn)
                results.append(dict(finding="DT13L-02", case="ordinary-cli-wall-linked-" + mname, expected="rejected",
                                    **cli_analyze(impl, scratch, cli, mout)))


def _stops(a):
    yield a.doc["stop"]
    if a.doc.get("failureEvent"):
        yield a.doc["failureEvent"]["stop"]


def set_message_seconds(a, seconds):
    for s in _stops(a):
        s["message"] = str(seconds) + s["message"][s["message"].index(" s elapsed"):]
    for r in a.rows:
        if r.get("halt"):
            r["halt"]["message"] = a.doc["stop"]["message"]
    if "aborted-draw.json" in a.files and a.files["aborted-draw.json"].get("halt"):
        a.files["aborted-draw.json"]["halt"]["message"] = a.doc["stop"]["message"]


def zero_message(a):
    set_message_seconds(a, 0)


def shift_elapsed(a, delta):
    for s in _stops(a):
        s["evidence"]["elapsedSeconds"] += delta
    for r in a.rows:
        if r.get("halt"):
            r["halt"]["evidence"]["elapsedSeconds"] = a.doc["stop"]["evidence"]["elapsedSeconds"]


def wall_clock_cases(impl, binary, scratch, results):
    """The reviewer's synthetic-fixture wall-clock witnesses (draw-start and request-start)."""
    S = impl.S
    f = S.Fixture(binary, arms=("control", "treatment"))
    threshold = f.proto["stop"]["wallClockSeconds"] - f.proto["request"]["timeoutSeconds"]
    for where, point in (("guard", ("draw_guard", "during")), ("request", ("draw_execute", "before"))):
        mono = S.Monotonic()

        def jump(b, phase, c, mono=mono, point=point):
            if (b, phase) == point:
                mono.jump()
        base = scratch / ("base-wall-" + where)
        record(impl, f, base, now=mono, hook=jump)
        results.append(dict(finding="control", case="wall-" + where + "-positive-control", expected="accepted",
                            **assess(impl, f, base)))
        for name, fn, expect in (("impossible-zero-message", zero_message, "rejected"),
                                 ("message-one-below-threshold", lambda a: set_message_seconds(a, threshold - 1), "rejected"),
                                 ("message-at-threshold", lambda a: set_message_seconds(a, threshold), "accepted"),
                                 ("message-above-elapsed", lambda a: set_message_seconds(a, 10 ** 9 + 1), "rejected")):
            out = scratch / ("wall-" + where + "-" + name)
            mutate(impl, base, out, fn)
            results.append(dict(finding="DT13L-02", case="wall-" + where + "-" + name, expected=expect,
                                **assess(impl, f, out)))


# --- DT13L-03/04: nested failures under the fail-closed policy ----------------------------------

def nested_failure_cases(impl, binary, scratch, results):
    import linecache
    K, S = impl.K, impl.S
    kinds = ["draw-recovery-bookkeeping", "draw-recovery-entry", "response-retention-construction"]
    if impl.name == "v13m":
        kinds.append("verifier-defense-unguarded-recovery")
    for kind in kinds:
        f = S.Fixture(binary)
        p = scratch / ("nested-" + kind)
        clock, fired, captured, known = S.Clock(), [], [], []
        real = S.Model(["REVIEW_UNIQUE_RESPONSE"] + f.replies() if kind.startswith("response") else f.replies())

        def model(payload, real=real, known=known):
            ret = real(payload)
            known.append(ret)
            return ret
        rec = K.Recorder(f.proto, f.proto_sha, f.sched, f.ctx, f.ctx.opener(binary), {K.QWEN: model}, str(p),
                         "nested recovery faults", state_fn=S.Health(clock), utc=clock, now=S.Monotonic(),
                         run_id=S.RUN_ID, provenance=S.SYNTHETIC)

        def trace(frame, event, arg, kind=kind):
            if event != "line" or fired:
                return trace
            line = linecache.getline(frame.f_code.co_filename, frame.f_lineno).strip()
            hit = (not kind.startswith("response") and frame.f_code is K.H.run_draw.__code__
                   and line == 'out["calls"] = model.calls if model else []' and frame.f_locals["out"]["status"] == "completed")
            hit |= (kind.startswith("response") and frame.f_code.co_name == "wrapped"
                    and frame.f_code.co_filename == K.__file__ and line == "status, body, err = result")
            if hit:
                if not kind.startswith("response"):
                    loc = frame.f_locals
                    captured.append(K.Recorder._finish_bookkeeping(copy.deepcopy(loc["out"]), loc.get("model"), loc.get("h")))
                fired.append({"fault": "first", "source": line})
                raise RuntimeError("independent first boundary fault")
            return trace
        target = {"draw-recovery-bookkeeping": "_finish_bookkeeping", "draw-recovery-entry": "_own_result",
                  "response-retention-construction": "_received_evidence",
                  "verifier-defense-unguarded-recovery": "_finish_bookkeeping"}[kind]
        original = getattr(rec, target)

        def fail_once(*args, original=original, target=target, **kwargs):
            if len(fired) == 1:
                fired.append({"fault": "second", "method": target})
                raise MemoryError("independent transient recovery allocation fault")
            return original(*args, **kwargs)
        patches = [mock.patch.object(rec, target, side_effect=fail_once)]
        if kind == "verifier-defense-unguarded-recovery":
            # A recorder without the fail-closed guard (the V0.13l handoff body), to show that
            # the verifier itself refuses the resulting archive (R-HANDOFF-003).
            def unguarded(*args, **kwargs):
                row = None
                try:
                    rec.pending = row = rec.draw_runner(*args, **kwargs)
                    rec._handoff(row)
                except BaseException as x:
                    rec._own_result(x, row)
                    raise
                return row
            patches.append(mock.patch.object(rec, "_draw_result", side_effect=unguarded))
        for pt in patches:
            pt.start()
        sys.settrace(trace)
        try:
            result = S.attempt(rec.run)
        finally:
            sys.settrace(None)
            for pt in reversed(patches):
                pt.stop()
        live = p / "live"
        raw = {q.name: q.read_text(errors="replace") for q in live.iterdir() if q.is_file()} if live.exists() else {}
        journal = S.journal(str(p))
        failure = json.loads(raw["persistence-failure.json"]) if "persistence-failure.json" in raw else None
        retained = result.get("retainedEvidence") or {}
        row = dict(finding="DT13L-04" if kind.startswith("response") else "DT13L-03", case=kind,
                   expected=("rejected, response retained, loss disclosed" if kind.startswith("response") else
                             "rejected by the verifier" if kind.startswith("verifier") else
                             "fail closed: rejected, possible loss disclosed"),
                   faults=fired, recorder=result.get("state") or result.get("error"), sealed="SEAL.json" in raw,
                   transportInvocations=len(known), journalDispatched=sum(x["record"] == "dispatch" for x in journal),
                   drawsBegun=sorted({x["draw"] for x in journal}),
                   responseArchived=any("REVIEW_UNIQUE_RESPONSE" in v for v in raw.values()),
                   responseInStructuredError="REVIEW_UNIQUE_RESPONSE" in json.dumps(retained, default=str),
                   evidenceLossInReport=(failure or {}).get("evidenceLoss"),
                   evidenceLossInStructuredError=retained.get("evidenceLoss") if isinstance(retained, dict) else None,
                   completedCaptured=bool(captured), **assess(impl, f, p))
        results.append(row)


def persistent_storage_case(impl, binary, scratch, results):
    """Fail-closed policy: storage refuses every write from the first ledger commit on."""
    S = impl.S
    f = S.Fixture(binary)
    fs = S.FailingFS()
    known = []
    real = S.Model(f.replies())

    def model(payload):
        known.append(1)
        return real(payload)

    def hook(b, phase, c):
        if (b, phase) == ("persist:ledger", "before"):
            fs.activate()
    out = scratch / "persistent-storage-failure"
    rec = record(impl, f, out, model=model, hook=hook, fs=fs)
    journal = S.journal(str(out))
    results.append(dict(finding="policy", case="persistent-storage-failure",
                        expected="fail closed: rejected, no retry, no replacement draw",
                        recorder=rec.get("state") or rec.get("error"), transportInvocations=len(known),
                        journalDispatched=sum(x["record"] == "dispatch" for x in journal),
                        drawsBegun=sorted({x["draw"] for x in journal}), **assess(impl, f, out)))


def meets(r):
    finding, a, exp = r["finding"], r.get("analysis") or {}, r["expected"]
    if finding == "primary":
        zero = all(d["totalVariation"] == 0 and all(x["difference"] == 0 for x in d["bySubmittedIdentity"].values())
                   for d in r["armDifferences"] or [])
        return r["analysisAccepted"] is True and zero and all(b == [] for b in r["submitted"])
    if exp == "accepted":
        return a.get("accepted") is True
    if exp == "accepted-unparsed":
        return a.get("accepted") is True and r.get("firstCallMetadata") == {"usage": None, "servedModel": None,
                                                                            "finishReason": None}
    if exp == "rejected":
        return a.get("accepted") is False
    if r["case"].startswith("verifier-defense"):
        return a.get("accepted") is False and r["classification"] in ("PARTIAL", "INVALID")
    no_retry = r["transportInvocations"] == r["journalDispatched"]
    if finding == "policy":
        return (r["recorder"] == "RecordingError" and r["classification"] == "INVALID" and not a.get("accepted")
                and no_retry and r["drawsBegun"] in ([], [1]))
    if finding == "DT13L-03":
        loss = r["evidenceLossInStructuredError"] or {}
        return (r["recorder"] == "RecordingError" and not r["sealed"] and r["classification"] == "INVALID"
                and not a.get("accepted") and no_retry and r["drawsBegun"] == [1] and loss.get("possible") is True
                and r["evidenceLossInReport"] == loss)
    if finding == "DT13L-04":
        return (r["classification"] == "INVALID" and not a.get("accepted") and no_retry and r["responseArchived"]
                and r["responseInStructuredError"] and (r["evidenceLossInStructuredError"] or {}).get("possible") is True)
    return False


def run(binary, scratch, implementation):
    scratch = Path(scratch)
    scratch.mkdir(parents=True, exist_ok=True)
    os.environ["TMPDIR"] = str(scratch)
    impl = Impl(implementation, scratch)
    results = []
    strict_response_cases(impl, binary, scratch, results)
    wall_clock_cases(impl, binary, scratch, results)
    ordinary_cli_cases(impl, binary, scratch, results)
    nested_failure_cases(impl, binary, scratch, results)
    persistent_storage_case(impl, binary, scratch, results)
    for r in results:
        r["meetsRemediationExpectation"] = meets(r)
    by = {}
    for r in results:
        by.setdefault(r["finding"], []).append(r["meetsRemediationExpectation"])
    return {"schema": SCHEMA, "implementation": implementation, "cases": results,
            "summary": {k: {"cases": len(v), "meetExpectation": sum(v)} for k, v in sorted(by.items())},
            "allMeetExpectation": all(r["meetsRemediationExpectation"] for r in results)}


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--implementation", choices=("v13l", "prefix", "v13m"), required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    doc = run(str(Path(a.binary).resolve()), a.scratch, a.implementation)
    write(a.out, doc)
    print(json.dumps(doc["summary"]), "all:", doc["allMeetExpectation"])
    if a.implementation == "v13m" and not doc["allMeetExpectation"]:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
