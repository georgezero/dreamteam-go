#!/usr/bin/env python3
"""Additional adversarial probes for the V0.13m-lite bounded technical
acceptance review (implementer-performed; not an independent review and not
a human review). They target the five review items with cases not in
v13m_dt13l.py: ambiguous responses in every retained location (workflow
delivery copy only, call journal only, aborted and interrupted draws,
draw_unrecorded journal), deep nesting, halt elapsed-time inflation for a
non-wall-clock halt with genuine provenance, and fail-closed behaviour
under a third fault, a BaseException secondary fault and a nested failure
after a committed draw. Every mutated archive has its dependent hashes,
mirrors, counters and seal recomputed and is assessed by FULL analysis
(ordinary CLI for genuine provenance). Scripted replies only; no model
endpoint.

    python3 -I -B analysis/alternatives/v13m_review_probes.py --binary ./dt --implementation v13m \\
        --scratch DIR --out FILE
"""
import argparse
import copy
import json
import os
from pathlib import Path
import sys
from unittest import mock

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13m_dt13l as D  # noqa: E402

DUP = lambda b: '{"choices":[], ' + b[1:]  # noqa: E731
UNUSABLE = "this reply is not a decision"  # a 200 response the workflow must correct: the draw needs another call


def body_everywhere_but(arc, where, fn):
    """Change the first retained response body in one location only (hashes recomputed there)."""
    r = arc.rows[0]
    call = next(c for c in r["calls"] if c.get("status") == 200 and c.get("body"))
    old, new = call["body"], fn(call["body"])
    if where == "delivery":
        for d in r["record"]["deliveries"]:
            if d.get("body") == old:
                d["body"] = new
    elif where == "journal":
        for rec in arc.calls:
            if rec.get("record") == "returned" and rec.get("draw") == r["seq"] and rec.get("n") == call["n"]:
                rec.update(body=new, responseSha256=D.impl_sha(new))


def aborted_body(arc, fn):
    ab = arc.files["aborted-draw.json"]
    call = next(c for c in ab["calls"] if c.get("status") == 200 and c.get("body"))
    new = fn(call["body"])
    call["body"], call["responseSha256"] = new, D.impl_sha(new)
    for rec in arc.calls:
        if rec.get("record") == "returned" and rec.get("draw") == ab["seq"] and rec.get("n") == call["n"]:
            rec.update(body=new, responseSha256=call["responseSha256"])


def unrecorded_journal_body(arc, fn):
    for rec in arc.calls:
        if rec.get("record") == "returned" and rec.get("status") == 200 and rec.get("body"):
            rec.update(body=fn(rec["body"]), responseSha256=D.impl_sha(fn(rec["body"])))
            return
    raise AssertionError("no retained response body to change")


def run(binary, scratch, implementation):
    scratch = Path(scratch)
    scratch.mkdir(parents=True, exist_ok=True)
    os.environ["TMPDIR"] = str(scratch)
    impl = D.Impl(implementation, scratch)
    K, S, W, M = impl.K, impl.S, impl.W, impl.M
    out = []

    def add(item, case, expected, **kw):
        out.append(dict(item=item, case=case, expected=expected, **kw))

    f = S.Fixture(binary, arms=("control", "treatment"))
    base = scratch / "base-complete"
    D.record(impl, f, base)
    for where in ("delivery", "journal"):
        p = scratch / ("ambiguous-only-in-" + where)
        D.mutate(impl, base, p, lambda a, w=where: body_everywhere_but(a, w, DUP))
        add("strict response JSON", "ambiguous body only in the " + where + " copy", "rejected", **D.assess(impl, f, p))

    one = S.Fixture(binary)
    # Aborted workflow draw: call 1 needs a correction, call 2 raises.
    n = [0]
    real = S.Model([UNUSABLE, S.F.ABANDON, S.F.ABANDON])

    def second_raises(payload):
        n[0] += 1
        if n[0] == 2:
            raise RuntimeError("scripted transport failure on call 2")
        return real(payload)
    ab = scratch / "base-aborted-workflow"
    D.record(impl, one, ab, model=second_raises)
    add("strict response JSON", "aborted-workflow positive control", "accepted", **D.assess(impl, one, ab))
    p = scratch / "ambiguous-in-aborted-draw"
    D.mutate(impl, ab, p, lambda a: aborted_body(a, DUP))
    add("strict response JSON", "ambiguous body in an aborted draw", "rejected", **D.assess(impl, one, p))

    # Interrupted row: call 1 needs a correction; the per-draw limit of 1 halts call 2.
    lim = one.with_limits(maxModelCallsPerDraw=1)
    hr = scratch / "base-interrupted-row"
    D.record(impl, lim, hr, replies=[UNUSABLE, S.F.ABANDON, S.F.ABANDON])
    add("strict response JSON", "interrupted-row positive control", "accepted", **D.assess(impl, lim, hr))
    p = scratch / "ambiguous-in-interrupted-row"
    D.mutate(impl, hr, p, lambda a: D.rebind_body(a, 1, DUP, K.SB.body_fields))
    add("strict response JSON", "ambiguous body in an interrupted (halted) row", "rejected", **D.assess(impl, lim, p))

    # draw_unrecorded whose retained journal response is ambiguous.
    un = scratch / "base-unrecorded"
    D.record(impl, one, un, draw_runner=W.unrecorded_runner, replies=[UNUSABLE, S.F.ABANDON, S.F.ABANDON])
    add("fail-closed persistence", "draw_unrecorded positive control (retained call cannot complete the workflow)",
        "accepted", **D.assess(impl, one, un))
    p = scratch / "ambiguous-in-unrecorded-journal"
    D.mutate(impl, un, p, lambda a: unrecorded_journal_body(a, DUP))
    add("strict response JSON", "ambiguous body in a draw_unrecorded journal outcome", "rejected", **D.assess(impl, one, p))

    # Deep nesting beyond the parsers' recursion limit, on the wire.
    seen = [0]
    real2 = S.Model(f.replies())

    def deep(payload):
        status, body, error = real2(payload)
        seen[0] += 1
        return (status, body[:-1] + ',"u":' + "[" * 100000 + "]" * 100000 + "}", error) if seen[0] == 1 else (status, body, error)
    p = scratch / "wire-deep-nesting"
    rec = D.record(impl, f, p, model=deep)
    rows = json.loads((p / "live" / "draws.json").read_text()) if (p / "live" / "draws.json").exists() else []
    first = rows[0]["calls"][0] if rows and rows[0].get("calls") else None
    add("strict response JSON", "response nested beyond the parser limit", "not COMPLETE with that response read",
        recorder=rec.get("state") or rec.get("error"),
        firstCallMetadata={k: first.get(k) for k in ("usage", "servedModel", "finishReason")} if first else None,
        **D.assess(impl, f, p))

    # Genuine provenance: a draw-call-limit halt whose elapsed evidence is inflated.
    cli = D.cli_setup(impl, scratch)
    cli["binary"] = binary
    cli["p"]["maxModelCallsPerDraw"] = 0
    D.write(scratch / "cli-protocol.json", cli["p"])
    cli["ph"] = S.D.sha256((scratch / "cli-protocol.json").read_bytes())
    hb = scratch / "cli-call-limit"
    K.run_schedule(cli["p"], cli["ph"], cli["sched"], cli["ctx"], cli["ctx"].opener(binary),
                   {K.QWEN: S.Model(cli["replies"])}, str(hb), "model-free review probe",
                   state_fn=S.Health(K.RR.utc), utc=K.RR.utc)
    add("wall-clock halt causality", "ordinary CLI draw-call-limit halt, genuine control", "accepted",
        **D.cli_analyze(impl, scratch, cli, hb))
    p = scratch / "cli-call-limit-elapsed-inflated"
    D.mutate(impl, hb, p, lambda a: D.shift_elapsed(a, 3 * 3600))
    add("wall-clock halt causality", "ordinary CLI non-wall halt with elapsed inflated by 3 h", "rejected",
        **D.cli_analyze(impl, scratch, cli, p))

    # Fail-closed persistence under harder nested faults.
    def nested(name, fixture, third=None, secondary=MemoryError("second fault"), at_draw=1):
        import linecache
        p = scratch / ("nested-" + name)
        clock, fired, known = S.Clock(), [], []
        real3 = S.Model(fixture.replies())

        def model(payload):
            known.append(1)
            return real3(payload)
        rec = K.Recorder(fixture.proto, fixture.proto_sha, fixture.sched, fixture.ctx, fixture.ctx.opener(binary),
                         {K.QWEN: model}, str(p), "review probe", state_fn=S.Health(clock), utc=clock,
                         now=S.Monotonic(), run_id=S.RUN_ID, provenance=S.SYNTHETIC)

        def trace(frame, event, arg):
            if event != "line" or fired:
                return trace
            line = linecache.getline(frame.f_code.co_filename, frame.f_lineno).strip()
            if frame.f_code is K.H.run_draw.__code__ and line == 'out["calls"] = model.calls if model else []' \
                    and frame.f_locals["out"]["status"] == "completed" and frame.f_locals["out"]["seq"] == at_draw:
                fired.append("first")
                raise RuntimeError("first fault")
            return trace
        original = rec._finish_bookkeeping

        def second(*a, **k):
            if fired == ["first"]:
                fired.append("second")
                raise secondary
            return original(*a, **k)
        patches = [mock.patch.object(rec, "_finish_bookkeeping", side_effect=second)]
        if third:
            def describe(x):
                fired.append("third")
                raise MemoryError("third fault")
            patches.append(mock.patch.object(K, "_describe", side_effect=describe) if hasattr(K, "_describe")
                           else mock.patch.object(rec, "_retain_failure", wraps=rec._retain_failure))
        for pt in patches:
            pt.start()
        sys.settrace(trace)
        try:
            result = S.attempt(rec.run)
        finally:
            sys.settrace(None)
            for pt in reversed(patches):
                pt.stop()
        journal = S.journal(str(p))
        retained = result.get("retainedEvidence") or {}
        a = D.assess(impl, fixture, p)
        return dict(faults=fired, recorder=result.get("state") or result.get("error"),
                    sealed=(p / "live" / "SEAL.json").exists(), transportInvocations=len(known),
                    journalDispatched=sum(x["record"] == "dispatch" for x in journal),
                    drawsBegun=sorted({x["draw"] for x in journal}),
                    ledgerCompleted=a["preserved"]["completedRows"],
                    evidenceLoss=retained.get("evidenceLoss") if isinstance(retained, dict) else None, **a)
    add("fail-closed persistence", "third fault while disclosing the loss", "fail closed",
        **nested("third-fault", S.Fixture(binary), third=True))
    add("fail-closed persistence", "BaseException (KeyboardInterrupt) as the secondary fault", "fail closed",
        **nested("base-exception", S.Fixture(binary), secondary=KeyboardInterrupt()))
    add("fail-closed persistence", "nested failure in draw 2 after draw 1 committed", "fail closed, draw 1 retained",
        **nested("second-draw", S.Fixture(binary), at_draw=2))
    for r in out:
        r["meetsExpectation"] = meets(r)
    return {"schema": "dreamteam.v13m-review-probes/v1", "implementation": implementation,
            "notice": "Implementer-performed bounded technical acceptance review probes; not an independent or human review.",
            "probes": out, "summary": {"probes": len(out), "meetExpectation": sum(r["meetsExpectation"] for r in out)}}


def meets(r):
    a = r.get("analysis") or {}
    exp = r["expected"]
    if exp == "accepted":
        return a.get("accepted") is True
    if exp == "rejected":
        return a.get("accepted") is False
    if exp.startswith("not COMPLETE"):
        # Either refused, or accepted only with the response as an unparsed reply (no metadata).
        return a.get("accepted") is not True or r.get("firstCallMetadata") in (
            None, {"usage": None, "servedModel": None, "finishReason": None})
    if exp.startswith("fail closed"):
        ok = (r["recorder"] == "RecordingError" and not r["sealed"] and r["classification"] == "INVALID"
              and not a.get("accepted") and r["transportInvocations"] == r["journalDispatched"])
        if "draw 1 retained" in exp:
            ok = ok and r["ledgerCompleted"] == 1 and r["drawsBegun"] == [1, 2]
        return ok
    return False


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--implementation", choices=("v13l", "v13m"), required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    doc = run(str(Path(a.binary).resolve()), a.scratch, a.implementation)
    D.write(a.out, doc)
    print(json.dumps(doc["summary"]))


if __name__ == "__main__":
    main()
