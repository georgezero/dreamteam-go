#!/usr/bin/env python3
"""The V0.13i fault-injection matrix, generated from the lifecycle contract.

For each base scenario the recorder runs once with a recording hook; every
injection point it reaches is a declared (boundary, phase) of the contract
(an undeclared point is an error). Each reached point is then combined with
every fault kind the contract declares for that operation and phase, the run
is repeated with that single fault, and the observed outcome (recorder
result, independent classification, stop, counters, cleanup, aborted stage,
and model-free replay analysis of every valid archive) is compared with the
outcome the contract's expectedRule predicts.

    python3 -I -B analysis/alternatives/v13i_faults.py run --binary ./dt --scratch DIR --out FILE [--jobs N]
    python3 -I -B analysis/alternatives/v13i_faults.py sample --binary ./dt --scratch DIR --matrix FILE --every K

No model is contacted; every reply is scripted.
"""
import argparse
import concurrent.futures
import errno
import json
import os
from pathlib import Path
import shutil
import sys

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import lifecycle_spec as LS  # noqa: E402
import v13i_scenarios as S  # noqa: E402

SCHEMA = "dreamteam.v13i-fault-matrix/v1"
BASES = {
    "A": "two completed draws; server observed; every health read healthy",
    "B": "the round health check before draw 2 answers unhealthy (a natural health stop)",
    "C": "engine opening fails at draw 2 (a natural aborted draw)",
}
STOP_SEQUENCE_ONLY = {"B", "C"}  # their other points repeat base A's


class Injected(RuntimeError):
    pass


def base_parts(base, fixture, scratch):
    """(health, opener, model, now) for a base scenario."""
    clock = S.Clock()
    health = S.Health(clock, modes={3: "unhealthy"} if base == "B" else {})
    pool = fixture.ctx.opener(fixture.binary)
    opener = S.FailingOpener(pool, RuntimeError("scripted engine opening failure"), at=2) if base == "C" else pool
    return clock, health, opener, S.Model(None or [S.F.ABANDON] * 6), S.Monotonic()


class Hook:
    def __init__(self, spec, target=None, kind=None, parts=None):
        self.spec, self.target, self.kind, self.parts = spec, target, kind, parts
        self.hits, self.fired = [], None

    def __call__(self, boundary, phase, ctx):
        i = len(self.hits)
        self.hits.append((boundary, phase, ctx))
        if i != self.target:
            return
        self.fired = {"boundary": boundary, "phase": phase}
        where = f"{boundary}.{phase}#{i}"
        if self.kind == "exception":
            raise Injected("injected fault at " + where)
        if self.kind == "keyboard":
            raise KeyboardInterrupt("injected interrupt at " + where)
        if self.kind == "persistence":
            raise OSError(errno.EIO, "injected I/O error at " + where)
        clock, health, opener, model, now = self.parts
        if self.kind == "transport":
            (health if boundary == "health_read" else model).inject()
        elif self.kind == "guard_halt":
            now.jump()


def run_one(fixture, base, out, target=None, kind=None):
    spec = LS.load()
    parts = base_parts(base, fixture, out)
    clock, health, opener, model, now = parts
    hook = Hook(spec, target, kind, parts)
    result = S.attempt(lambda: fixture.record(out, health=health, opener=opener, model=model, hook=hook, now=now,
                                              clock=clock))
    return hook, result


def label(hits, i):
    b, p, ctx = hits[i]
    key = (b, p, ctx.get("event"))
    occ = sum(1 for j in range(i + 1) if (hits[j][0], hits[j][1], hits[j][2].get("event")) == key)
    ev = f"[{ctx['event']}]" if ctx.get("event") else ""
    return f"{b}{ev}#{occ}.{p}", occ


def expect(spec, base_obs, hit, kind):
    """The outcome the contract's expectedRule predicts for one fault."""
    b, p, ctx = hit
    op = spec["faultInjection"]["boundaries"][b]["operation"]
    invalid = {"recorder": "RecordingError", "classification": "INVALID"}
    if ctx["state"] is None:
        # Before the initial manifest is durable; run.json can exist (not synced) only after its replacement.
        late = b == "persist:manifest" and p in ("replaced", "dir_fsync")
        return dict(invalid, rule="NO_ARCHIVE", manifest=late)
    if kind == "persistence" or (op in ("persist", "unlink") and p not in ("before", "after")):
        return dict(invalid, rule="FAIL_CLOSED")
    if (ctx["stopping"] or ctx["state"] in ("CLEANUP", "FINALIZING")) and not (b == "session_cleanup" and p == "during"):
        return dict(invalid, rule="FAIL_CLOSED")
    code = "operator" if kind == "keyboard" else "driver"
    if b == "session_cleanup" and p == "during":
        had = base_obs.get("stopCode")
        return {"rule": "CLEANUP_FAILED", "recorder": "returned", "classification": "PARTIAL", "cleanup": "FAILED",
                "stopCode": had or code, "stopState": base_obs.get("stopState") if had else "CLEANUP",
                "completed": base_obs["completed"]}
    if b == "health_read" and p == "during" and kind == "transport":
        return {"rule": "HEALTH_STOP", "recorder": "returned", "classification": "PARTIAL", "stopCode": "health",
                "stopState": ctx["state"], "completed": ctx["committed"], "cleanup": "SUCCEEDED"}
    if b == "draw_guard" and kind == "guard_halt":
        return {"rule": "GUARD_STOP", "recorder": "returned", "classification": "PARTIAL", "stopCode": "wall_clock",
                "stopState": "READY", "completed": ctx["committed"], "cleanup": "SUCCEEDED"}
    if b == "draw_execute" and p == "model_call" and kind == "transport":
        return {"rule": "WORKFLOW_EVIDENCE", "recorder": "returned", "classification": base_obs["classification"],
                "stopCode": base_obs.get("stopCode"), "completed": base_obs["completed"],
                "modelCalls": base_obs["modelCalls"] + 1}
    out = {"rule": "HANDLED", "recorder": "returned", "classification": "PARTIAL", "stopCode": code, "cleanup": "SUCCEEDED"}
    if ctx["state"] == "DRAW_ACTIVE" and ctx["pending"]:
        out.update(stopState="DRAW_RECORDED", completed=ctx["committed"] + (0 if ctx["pendingDurable"] else 1),
                   abortedStage=None)
    elif ctx["state"] == "DRAW_ACTIVE":
        out.update(stopState="DRAW_ACTIVE", completed=ctx["committed"],
                   abortedStage={"engine_open": "engine_open", "model_call": "workflow"}.get(p, "draw_setup")
                   if b == "draw_execute" else "draw_setup")
    else:
        out.update(stopState=ctx["state"], completed=ctx["committed"], abortedStage=None)
    return out


def observe(fixture, out, result, analyze):
    obs = S.summary(fixture, out, result)
    obs["recorder"] = "returned" if result["returned"] else result["error"]
    if analyze:
        try:
            a = fixture.analyze(out)
            obs["analysis"] = {"accepted": True, "runStatus": a["runStatus"], "completedVerified": len(a["draws"]),
                               "comparisonsSuppressed": a["armDifferences"] == [] and a["pairs"] == []}
        except S.D.ExperimentError as x:
            obs["analysis"] = {"accepted": False, "refusal": str(x).split(":")[0]}
    return obs


def agree(expected, observed):
    keys = [k for k in expected if k != "rule"]
    bad = [k for k in keys if observed.get(k) != expected[k]]
    a = observed.get("analysis")
    if a is not None:
        if observed["classification"] == "INVALID" and a["accepted"]:
            bad.append("analysis accepted an INVALID archive")
        if observed["classification"] in ("COMPLETE", "PARTIAL") and not a["accepted"]:
            bad.append("analysis refused a valid archive")
        if observed["classification"] == "PARTIAL" and a.get("accepted") and not a["comparisonsSuppressed"]:
            bad.append("complete-run comparison for PARTIAL")
    if observed["classification"] == "COMPLETE" and expected.get("rule") not in ("WORKFLOW_EVIDENCE",):
        bad.append("COMPLETE after a lifecycle fault")
    return bad


_FIXTURE = None


def _fixture(binary):
    global _FIXTURE
    if _FIXTURE is None:
        _FIXTURE = S.Fixture(binary)
    return _FIXTURE


def _case(args):
    binary, scratch, case, analyze = args
    fixture = _fixture(binary)
    out = os.path.join(scratch, case["id"].replace(":", "_").replace("/", "_"))
    shutil.rmtree(out, ignore_errors=True)
    try:
        hook, result = run_one(fixture, case["base"], out, case["index"], case["kind"])
        obs = observe(fixture, out, result, analyze)
        obs["fired"] = hook.fired is not None and hook.fired == {"boundary": case["boundary"], "phase": case["phase"]}
    except BaseException as x:  # noqa: BLE001
        obs = {"harnessError": f"{type(x).__name__}: {x}"}
    finally:
        shutil.rmtree(out, ignore_errors=True)
    return case["id"], obs


def enumerate_cases(binary, scratch):
    spec = LS.load()
    fixture = _fixture(binary)
    bases, cases, reached = {}, [], set()
    for base in BASES:
        out = os.path.join(scratch, "base-" + base)
        shutil.rmtree(out, ignore_errors=True)
        hook, result = run_one(fixture, base, out)
        bases[base] = {"description": BASES[base], "observed": observe(fixture, out, result, True), "points": len(hook.hits)}
        shutil.rmtree(out, ignore_errors=True)
        for i, hit in enumerate(hook.hits):
            b, p, ctx = hit
            reached.add((b, p))
            if base in STOP_SEQUENCE_ONLY and not (ctx["stopping"] or ctx["state"] in ("CLEANUP", "FINALIZING")):
                continue
            name, occ = label(hook.hits, i)
            op = spec["faultInjection"]["boundaries"][b]["operation"]
            for kind in spec["faultInjection"]["operations"][op]["phases"][p]:
                cases.append({"id": f"{base}:{i:03d}:{name}:{kind}", "base": base, "index": i, "boundary": b, "phase": p,
                              "event": ctx.get("event"), "occurrence": occ, "kind": kind,
                              "context": {k: ctx[k] for k in ("state", "stopping", "committed", "pending", "pendingDurable")},
                              "expected": expect(spec, bases[base]["observed"], hit, kind)})
    declared = {(b, p) for b, v in spec["faultInjection"]["boundaries"].items()
                for p in spec["faultInjection"]["operations"][v["operation"]]["phases"]}
    return spec, bases, cases, declared, reached


def coverage(spec, results):
    table = LS.transition_table(spec)
    seen_rules, seen_stops = {}, set()
    for c in results:
        seen_rules[c["expected"]["rule"]] = seen_rules.get(c["expected"]["rule"], 0) + 1
        o = c["observed"]
        if o.get("classification") in ("PARTIAL",) and o.get("stopCode"):
            seen_stops.add((o["stopCode"], o["stopState"]))
    admissible = sorted((code, s) for code, r in spec["stopCodes"].items() for s in r["states"])
    return {"rules": dict(sorted(seen_rules.items())), "transitions": len(table),
            "stopCodeStatesObserved": sorted(list(x) for x in seen_stops),
            "stopCodeStatesAdmissible": [list(x) for x in admissible]}


def cmd_run(a):
    os.makedirs(a.scratch, exist_ok=True)
    spec, bases, cases, declared, reached = enumerate_cases(a.binary, a.scratch)
    work = [(a.binary, a.scratch, c, True) for c in cases]
    observed = {}
    with concurrent.futures.ProcessPoolExecutor(max_workers=a.jobs) as pool:
        for cid, obs in pool.map(_case, work, chunksize=4):
            observed[cid] = obs
    results = []
    for c in cases:
        o = observed[c["id"]]
        mismatch = ["harness: " + o["harnessError"]] if "harnessError" in o else agree(c["expected"], o)
        if "harnessError" not in o and not o.get("fired"):
            mismatch.append("fault did not fire at its point")
        results.append(dict(c, observed=o, agree=not mismatch, mismatch=mismatch))
    doc = {"schema": SCHEMA, "lifecycle": {"id": LS.LIFECYCLE_ID, "sha256": LS.spec_sha256()},
           "notice": "Generated from the lifecycle contract; scripted model and scripted HTTP only; no inference.",
           "bases": bases, "declaredPoints": sorted(f"{b}.{p}" for b, p in declared),
           "unreachedDeclaredPoints": sorted(f"{b}.{p}" for b, p in declared - reached),
           "cases": results, "coverage": coverage(spec, results),
           "summary": {"cases": len(results), "agree": sum(r["agree"] for r in results),
                       "disagree": sum(not r["agree"] for r in results),
                       "points": len({(r["base"], r["index"]) for r in results}),
                       "classification": {k: sum(r["observed"].get("classification") == k for r in results)
                                          for k in ("COMPLETE", "PARTIAL", "INVALID")}}}
    S.D.write_json(a.out, doc)
    s = doc["summary"]
    print(f"fault matrix: {s['cases']} cases at {s['points']} points; {s['agree']} agree, {s['disagree']} disagree; "
          f"unreached declared points: {len(doc['unreachedDeclaredPoints'])}")
    if s["disagree"] or doc["unreachedDeclaredPoints"]:
        raise SystemExit(1)


def cmd_sample(a):
    """Rerun every K-th committed case and require the same observation."""
    doc = json.load(open(a.matrix, encoding="utf-8"))
    if doc["lifecycle"]["sha256"] != LS.spec_sha256():
        raise SystemExit("the committed matrix is for another lifecycle contract")
    os.makedirs(a.scratch, exist_ok=True)
    picked = doc["cases"][::a.every]
    bad = 0
    for c in picked:
        cid, obs = _case((a.binary, a.scratch, c, True))
        if obs != c["observed"] or not c["agree"]:
            bad += 1
            print("MISMATCH", cid, json.dumps(obs, sort_keys=True))
    print(f"fault matrix sample: {len(picked)} of {len(doc['cases'])} cases re-run; {bad} mismatches")
    if bad:
        raise SystemExit(1)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("run")
    p.add_argument("--binary", required=True)
    p.add_argument("--scratch", required=True)
    p.add_argument("--out", required=True)
    p.add_argument("--jobs", type=int, default=max(1, (os.cpu_count() or 2) // 2))
    p = sub.add_parser("sample")
    p.add_argument("--binary", required=True)
    p.add_argument("--scratch", required=True)
    p.add_argument("--matrix", required=True)
    p.add_argument("--every", type=int, default=25)
    a = ap.parse_args(argv)
    (cmd_run if a.cmd == "run" else cmd_sample)(a)


if __name__ == "__main__":
    main()
