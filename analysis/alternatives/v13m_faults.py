#!/usr/bin/env python3
"""The V0.13l fault-injection matrix, generated from the safety contract.

For each base scenario the recorder runs once with a recording hook; every
point it reaches is a declared (boundary, phase) of the contract (an
undeclared point is an error). Each reached point is combined with every
fault kind the contract declares for that operation and phase (a real SIGINT
delivered to the recording process, a recorder exception, a single
filesystem fault, a persistent filesystem failure that also defeats any later
write, a genuine transport failure, a guard halt), the run is repeated with
that single fault, and the observed outcome (recorder result, independent
classification, seal, stop, counters, interrupts, preserved evidence and
model-free replay analysis) is compared with the outcome the contract's
expected-outcome rule predicts from the base trace. Transition, stop-code and
safe-point coverage are measured from the archives, not counted from the
contract.

    python3 -I -B analysis/alternatives/v13m_faults.py run --binary ./dt --scratch DIR --out FILE [--jobs N]
    python3 -I -B analysis/alternatives/v13m_faults.py sample --binary ./dt --scratch DIR --matrix FILE --every K

No model is contacted; every reply is scripted.
"""
import argparse
import concurrent.futures
import errno
import json
import os
from pathlib import Path
import shutil
import signal
import sys

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import lifecycle_v4_spec as LS  # noqa: E402
import v13m_scenarios as S  # noqa: E402

SCHEMA = "dreamteam.v13m-fault-matrix/v1"
HALT_CODES = ("draw_call_limit", "call_budget", "transport", "wall_clock")


def P(code, state, completed, calls, **k):
    return dict({"classification": "PARTIAL", "stopCode": code, "stopState": state, "completed": completed,
                 "modelCalls": calls, "cleanup": "SUCCEEDED", "abortedStage": None}, **k)


# Declared base outcomes (from the contract's semantics, not observed).
BASES = {
    "A": ("two completed draws; server observed; every health read healthy",
          {"classification": "COMPLETE", "stopCode": None, "stopState": None, "completed": 2, "modelCalls": 2,
           "cleanup": "SUCCEEDED", "abortedStage": None}),
    "B": ("the round health check before draw 2 answers unhealthy", P("health", "HEALTH_CHECK", 1, 1)),
    "C": ("engine opening fails at draw 2", P("driver", "DRAW_ACTIVE", 1, 1, abortedStage="engine_open")),
    "D": ("per-draw call limit 0: draw 1 is interrupted at its first request", P("draw_call_limit", "DRAW_RECORDED", 0, 0)),
    "E": ("run call budget 1: draw 2 is interrupted at its first request", P("call_budget", "DRAW_RECORDED", 1, 1)),
    "F": ("one transport failure stops the run at the next request", P("transport", "DRAW_RECORDED", 0, 1)),
    "G": ("the wall clock passes during draw 1's first call", P("wall_clock", "DRAW_RECORDED", 0, 1)),
    "H": ("the server signature changes before draw 2", P("server_change", "HEALTH_CHECK", 1, 1)),
    "I": ("the pooled session release fails after both draws", P("driver", "CLEANUP", 2, 2, cleanup="FAILED")),
}
FULL_KINDS = {"A"}  # transport and guard-halt faults change a scenario's own mechanics; only base A takes them


class Injected(RuntimeError):
    pass


class Parts:
    def __init__(self, base, fixture):
        self.clock, self.now, self.fs = S.Clock(), S.Monotonic(), S.FailingFS()
        modes = {3: "unhealthy"} if base == "B" else {}
        self.health = S.Health(self.clock, modes=modes)
        if base == "H":
            inner, versions = self.health, iter(["one", "one", "two", "two"])
            self.health = lambda: dict(inner(), serverVersion=next(versions, "two"))
            self.health.inject = inner.inject
        pool = fixture.ctx.opener(fixture.binary)
        self.opener = pool
        if base == "C":
            self.opener = S.FailingOpener(pool, RuntimeError("scripted engine opening failure"), at=2)
        if base == "I":
            self.opener = S.FailingOpener(pool, close_error=RuntimeError("scripted pool release failure"))
        replies = [S.F.ABANDON] * 6
        if base in ("F", "G"):
            replies = [None] * 6
        self.model = S.Model(replies)
        if base == "G":
            inner, now = self.model, self.now

            def late(payload):
                now.jump()
                return inner(payload)
            late.inject = inner.inject
            self.model = late
        self.fixture = fixture
        if base == "D":
            self.fixture = fixture.with_limits(maxModelCallsPerDraw=0)
        if base == "E":
            self.fixture = fixture.with_limits(maxModelCalls=1)
        if base == "F":
            self.fixture = fixture.with_limits(consecutiveTransportFailures=1)


class Hook:
    def __init__(self, target=None, kind=None, parts=None):
        self.target, self.kind, self.parts = target, kind, parts
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
        if self.kind == "signal":
            os.kill(os.getpid(), signal.SIGINT)
        elif self.kind == "persistence":
            raise OSError(errno.EIO, "injected I/O error at " + where)
        elif self.kind == "persistent":
            self.parts.fs.activate()
        elif self.kind == "transport":
            (self.parts.health if boundary == "health_read" else self.parts.model).inject()
        elif self.kind == "guard_halt":
            self.parts.now.jump()


def run_one(fixture, base, out, target=None, kind=None):
    parts = Parts(base, fixture)
    hook = Hook(target, kind, parts)
    result = S.attempt(lambda: parts.fixture.record(out, health=parts.health, opener=parts.opener, model=parts.model,
                                                    hook=hook, now=parts.now, clock=parts.clock, fs=parts.fs))
    return parts, hook, result


def label(hits, i):
    b, p, ctx = hits[i]
    key = (b, p, ctx.get("event"), ctx.get("record"))
    occ = sum(1 for j in range(i + 1) if (hits[j][0], hits[j][1], hits[j][2].get("event"), hits[j][2].get("record")) == key)
    tag = f"[{ctx['event']}]" if ctx.get("event") else f"[{ctx['record']}]" if ctx.get("record") else ""
    return f"{b}{tag}#{occ}.{p}", occ


def _sealed_at(hits, i):
    """True when hit i is at or after the seal rename (the commit point)."""
    renamed = next((j for j, h in enumerate(hits) if h[0] == "seal" and h[1] == "renamed"), None)
    return renamed is not None and i >= renamed


def _next_stop_point(spec, hits, i, strict):
    """The first STOP safe point at or after (strictly after when strict) hit
    i that the base reaches while running, else None."""
    points = spec["safePoints"]["points"]
    for j in range(i + (1 if strict else 0), len(hits)):
        b, p, ctx = hits[j]
        if b == "safe_point" and points[p]["kind"] == "STOP" and not ctx["stopping"]:
            return j
        if ctx["stopping"]:
            return None
    return None


def _first_safe_point(spec, hits, i):
    points = spec["safePoints"]["points"]
    for j in range(i, len(hits)):
        b, p, ctx = hits[j]
        if b != "safe_point":
            continue
        kind = points[p]["kind"]
        if kind == "NOTE":
            continue
        if kind == "STOP" and ctx["stopping"]:
            continue
        return j
    return None


def _stop_at(hits, j, code, base_outcome, failure_stage=None):
    b, p, ctx = hits[j]
    if p == "before_dispatch":
        return P(code, "DRAW_ACTIVE", ctx["completedRows"], ctx["dispatched"], abortedStage="workflow")
    return P(code, ctx["state"], ctx["completedRows"], ctx["dispatched"])


def expect(spec, base, hits, i, kind):
    """The outcome the contract's expectedRule predicts for one fault. The
    pooled session release is a property of the base: when the base's release
    fails, every finalized outcome records cleanup FAILED beside its primary stop."""
    out = _expect(spec, base, hits, i, kind)
    if out.get("recorder") == "returned" and BASES[base][1]["cleanup"] == "FAILED":
        out["cleanup"] = "FAILED"
    return out


def _expect(spec, base, hits, i, kind):
    b, p, ctx = hits[i]
    op = spec["faultInjection"]["boundaries"][b]["operation"]
    base_out = dict(BASES[base][1])
    invalid = {"recorder": "RecordingError", "classification": "INVALID", "sealed": False}
    if kind == "kill":
        return dict(rule="KILLED", classification=base_out["classification"] if _sealed_at(hits, i) else "INVALID",
                    sealed=_sealed_at(hits, i))
    if _sealed_at(hits, i):
        out = dict(base_out, rule="COMMITTED", recorder="returned", sealed=True)
        if (kind in ("persistent", "persistence") and p in ("renamed", "dir_fsync")) or (kind == "exception" and p == "dir_fsync"):
            out["directorySync"] = "FAILED"
        elif kind == "exception":
            out["postCommitError"] = True
        if kind == "signal":
            out.update(unarchived=1, interruptDispositions=[])
        return out
    if kind == "signal":
        j = _first_safe_point(spec, hits, i)
        if j is None:
            return dict(base_out, rule="INTERRUPT", recorder="returned", sealed=True, unarchived=1, interruptDispositions=[])
        jb, jp, jctx = hits[j]
        kind_j = spec["safePoints"]["points"][jp]["kind"]
        if kind_j == "STOP":
            return dict(_stop_at(hits, j, "operator", base_out), rule="INTERRUPT", recorder="returned", sealed=True,
                        safePoint=jp, interruptDispositions=["STOP"], unarchived=0)
        disposition = "AFTER_STOP" if jctx["stopped"] else "AFTER_SCHEDULE"
        return dict(base_out, rule="INTERRUPT", recorder="returned", sealed=True, interruptDispositions=[disposition],
                    unarchived=0)
    if ctx["state"] is None:
        return dict(invalid, rule="NO_ARCHIVE")
    if kind in ("persistence", "persistent") or (op in ("persist", "append", "seal") and p not in ("before", "after")) \
            or (b == "seal" and p == "before") \
            or (b == "append:calls" and p == "before" and ctx.get("record") in ("returned", "raised")):
        return dict(invalid, rule="FAIL_CLOSED")
    deferred = (b == "append:calls" and p == "after") or \
        (b == "safe_point" and spec["safePoints"]["points"][p]["kind"] == "NOTE")
    if deferred:
        j = _next_stop_point(spec, hits, i, strict=True)
        if j is None:
            return dict(invalid, rule="FAIL_CLOSED")
        return dict(_stop_at(hits, j, "driver", base_out), rule="HANDLED", recorder="returned", sealed=True)
    if (ctx["stopping"] or ctx["state"] in ("CLEANUP", "FINALIZING")) and not (b == "session_cleanup" and p == "during"):
        return dict(invalid, rule="FAIL_CLOSED")
    if b == "session_cleanup" and p == "during":
        had = base_out["stopCode"]
        return dict(base_out, rule="CLEANUP_FAILED", recorder="returned", sealed=True, cleanup="FAILED",
                    classification="PARTIAL", stopCode=had or "driver", stopState=base_out["stopState"] if had else "CLEANUP")
    if b == "health_read" and p == "during" and kind == "transport":
        return dict(P("health", ctx["state"], ctx["completedRows"], ctx["dispatched"]), rule="HEALTH_STOP",
                    recorder="returned", sealed=True)
    if b == "draw_guard" and kind == "guard_halt":
        return dict(P("wall_clock", "READY", ctx["completedRows"], ctx["dispatched"]), rule="GUARD_STOP",
                    recorder="returned", sealed=True)
    if b == "draw_execute" and p == "model_call" and kind == "transport":
        return dict(base_out, rule="WORKFLOW_EVIDENCE", recorder="returned", sealed=True,
                    modelCalls=base_out["modelCalls"] + 1)
    out = {"rule": "HANDLED", "recorder": "returned", "sealed": True}
    if ctx["state"] == "DRAW_ACTIVE" and ctx["pending"]:
        if ctx["pendingStatus"] == "interrupted":
            out.update(P(base_out["stopCode"], "DRAW_RECORDED", ctx["completedRows"], ctx["dispatched"]))
        else:
            out.update(P("driver", "DRAW_RECORDED", ctx["completedRows"] + (0 if ctx["pendingDurable"] else 1),
                         ctx["dispatched"]))
    elif ctx["state"] == "DRAW_ACTIVE":
        stage = "draw_setup"
        if (b, p) == ("draw_execute", "engine_open"):
            stage = "engine_open"
        elif (b, p) in (("draw_execute", "model_call"), ("safe_point", "before_dispatch"), ("append:calls", "before")):
            stage = "workflow"
        out.update(P("driver", "DRAW_ACTIVE", ctx["completedRows"], ctx["dispatched"], abortedStage=stage))
    elif ctx["state"] == "DRAW_RECORDED" and ctx["lastStatus"] == "interrupted":
        out.update(P(base_out["stopCode"], "DRAW_RECORDED", ctx["completedRows"], ctx["dispatched"]))
    else:
        out.update(P("driver", ctx["state"], ctx["completedRows"], ctx["dispatched"]))
    return out


def observe(fixture, out, result, analyze):
    obs = S.summary(fixture, out, result)
    obs["recorder"] = S.recorder_label(result)
    if result.get("interrupts") is not None:
        obs["unarchived"] = result["interrupts"]["unarchived"]
    if result.get("commit") is not None:
        obs["directorySync"] = result["commit"]["directorySync"]
        obs["postCommitError"] = result["commit"]["postCommitError"] is not None
    live = Path(out) / "live"
    # Count distinct completed results across committed and pending copies.
    # A failed ledger candidate can also contain older committed rows.
    completed = set()
    for path in live.iterdir() if live.exists() else []:
        if not (path.name in ("draws.json", "pending-draw.json", "persistence-failure.json")
                or path.name.startswith((".tmp-", ".failure-"))):
            continue
        try:
            value = json.loads(path.read_bytes())
            rows = value if isinstance(value, list) else [value.get("draw"), value.get("pending")]
            completed.update(S.J.sha256(S.J.canon(row)) for row in rows
                             if isinstance(row, dict) and row.get("status") == "completed")
        except (ValueError, OSError, AttributeError, TypeError):
            pass
    obs["archiveCompletedRowSha256"] = sorted(completed)
    # Received responses retained in the failure report, or in the structured error, and
    # the calls whose outcome is durable in the journal.
    received = {}
    for source in ([live / "persistence-failure.json"] if (live / "persistence-failure.json").exists() else []):
        try:
            received.update((S.J.canon(r), r) for r in json.loads(source.read_bytes()).get("receivedResponses") or [])
        except (ValueError, OSError, AttributeError, TypeError):
            pass
    retained = result.get("retainedEvidence") if isinstance(result, dict) else None
    if isinstance(retained, dict):
        received.update((S.J.canon(r), r) for r in retained.get("receivedResponses") or [])
    obs["receivedResponses"] = len(received)
    obs["receivedKs"] = sorted({r.get("k") for r in received.values() if isinstance(r, dict) and isinstance(r.get("k"), int)})
    obs["journalOutcomeKs"] = sorted({r.get("k") for r in (S.journal(out) or [])
                                      if r.get("record") in ("returned", "raised") and isinstance(r.get("k"), int)})
    transitions = []
    if (live / "run.json").exists():
        try:
            doc = json.loads((live / "run.json").read_text())
            transitions = [[e["from"], e["event"]] for e in doc["journal"]]
        except (ValueError, KeyError, TypeError):
            pass
    if obs["sealed"]:
        seal = json.loads((live / S.J.SEAL).read_text())
        transitions.append([seal["transition"]["from"], seal["transition"]["event"]])
    obs["transitions"] = transitions
    if analyze:
        try:
            a = fixture.analyze(out)
            obs["analysis"] = {"accepted": True, "runStatus": a["runStatus"], "completedVerified": len(a["draws"]),
                               "comparisonsSuppressed": a["armDifferences"] == [] and a["pairs"] == [],
                               "aborted": (a["abortedVerification"] or {}).get("status")}
        except S.D.ExperimentError as x:
            obs["analysis"] = {"accepted": False, "refusal": str(x).split(":")[0]}
    return obs


def stored(obs):
    """The committed observation: the archive's transition list is used for
    the measured coverage and not stored per case."""
    return {k: v for k, v in obs.items() if k != "transitions"}


# DT13K-07: every generated case keeps every preservation expectation the
# oracle uses; a context that drops one fails instead of weakening the check.
CONTEXT_KEYS = ("state", "stopping", "committed", "completedRows", "lastStatus", "pending", "pendingDurable",
                "pendingStatus", "pendingRetained", "dispatched", "scheduleDone", "stopped", "receivedUnrecorded",
                "partialRetained")
PRESERVATION_KEYS = ("committed", "completedRows", "pendingStatus", "pendingDurable", "pendingRetained", "dispatched",
                     "receivedUnrecorded")


def agree(expected, observed, ctx):
    keys = [k for k in expected if k != "rule"]
    bad = [f"{k}: expected {expected[k]!r}, observed {observed.get(k)!r}" for k in keys if observed.get(k) != expected[k]]
    a = observed.get("analysis")
    cls = observed.get("classification")
    if a is not None:
        if cls == "INVALID" and a["accepted"]:
            bad.append("analysis accepted an INVALID archive")
        if cls in ("COMPLETE", "PARTIAL") and not a["accepted"]:
            bad.append("analysis refused a valid archive")
        if cls == "PARTIAL" and a.get("accepted") and not a["comparisonsSuppressed"]:
            bad.append("complete-run comparison for PARTIAL")
    if observed.get("recorder") == "returned" and observed.get("sealState") != cls:
        bad.append("the recorder's sealed state disagrees with the independent classification")
    if observed.get("recorder") == "RecordingError" and (observed.get("sealed") or cls != "INVALID"):
        bad.append("a recording failure left a sealed or accepted archive")
    if cls == "COMPLETE" and expected.get("classification") != "COMPLETE":
        bad.append("FALSE COMPLETE")
    if ctx is not None:
        missing = [k for k in PRESERVATION_KEYS if k not in ctx]
        if missing:
            bad.append("the case context dropped preservation expectations: " + ", ".join(missing))
            return bad
        if (observed.get("ledgerRows") or 0) < ctx["committed"]:
            bad.append("a durable completed row was lost")
        known = ctx["completedRows"] + int(observed.get("recorder") != "killed" and ctx.get("pendingStatus") == "completed" and not ctx.get("pendingDurable"))
        retained = observed.get("pendingCompleted", 0)
        if not ctx.get("pendingRetained"):
            retained = max(retained, observed.get("reportedCompleted", 0))
        count = (observed.get("ledgerCompleted") or 0) + retained
        if "archiveCompletedRowSha256" in observed:
            distinct = set(observed["archiveCompletedRowSha256"])
            if not ctx.get("pendingRetained") and observed.get("reportedRowSha256"):
                distinct.add(observed["reportedRowSha256"])
            count = len(distinct)
        if count < known:
            bad.append("a completed pending draw was lost")
        if (observed.get("journalDispatched") or 0) < ctx["dispatched"]:
            bad.append("a durable dispatched call was lost")
        if ctx["receivedUnrecorded"] and observed.get("recorder") != "killed":
            # The response received for call k survives as its durable journal outcome or as
            # retained failure evidence (failure report or structured error).
            k = ctx["dispatched"]
            if k not in (observed.get("journalOutcomeKs") or []) and k not in (observed.get("receivedKs") or []):
                bad.append("a received model response was lost")
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
    out = os.path.join(scratch, case["id"].replace(":", "_").replace("/", "_").replace("[", "_").replace("]", "_"))
    shutil.rmtree(out, ignore_errors=True)
    try:
        parts, hook, result = run_one(fixture, case["base"], out, case["index"], case["kind"])
        obs = observe(parts.fixture, out, result, analyze)
        obs["fired"] = hook.fired is not None and hook.fired == {"boundary": case["boundary"], "phase": case["phase"]}
    except BaseException as x:  # noqa: BLE001
        obs = {"harnessError": f"{type(x).__name__}: {x}"}
    finally:
        shutil.rmtree(out, ignore_errors=True)
    return case["id"], obs


def trace(binary, scratch, base):
    """The base run's reached points (with their contexts) and its observation."""
    fixture = _fixture(binary)
    out = os.path.join(scratch, "base-" + base)
    shutil.rmtree(out, ignore_errors=True)
    parts, hook, result = run_one(fixture, base, out)
    obs = observe(parts.fixture, out, result, True)
    shutil.rmtree(out, ignore_errors=True)
    return hook.hits, obs


def enumerate_cases(binary, scratch, bases=None):
    spec = LS.load()
    declared = LS.declared_points(spec)
    out_bases, cases, reached = {}, [], set()
    for base in bases or BASES:
        hits, obs = trace(binary, scratch, base)
        want = BASES[base][1]
        out_bases[base] = {"description": BASES[base][0], "declared": want, "observed": obs, "points": len(hits),
                           "agree": not [k for k in want if obs.get(k) != want[k]]}
        for i, (b, p, ctx) in enumerate(hits):
            if (b, p) not in declared:
                raise SystemExit(f"undeclared point reached: {b}.{p}")
            reached.add((b, p))
            name, occ = label(hits, i)
            op = spec["faultInjection"]["boundaries"][b]["operation"]
            for kind in spec["faultInjection"]["operations"][op]["phases"][p]:
                if kind in ("transport", "guard_halt") and base not in FULL_KINDS:
                    continue
                cases.append({"id": f"{base}:{i:03d}:{name}:{kind}", "base": base, "index": i, "boundary": b,
                              "phase": p, "event": ctx.get("event"), "record": ctx.get("record"), "occurrence": occ,
                              "kind": kind, "context": {k: ctx[k] for k in CONTEXT_KEYS},
                              "expected": expect(spec, base, hits, i, kind)})
    return spec, out_bases, cases, declared, reached


def coverage(spec, results):
    table = LS.transition_table(spec)
    seen_rules, seen_stops, seen_transitions, seen_points = {}, set(), set(), set()
    for c in results:
        seen_rules[c["expected"]["rule"]] = seen_rules.get(c["expected"]["rule"], 0) + 1
        o = c["observed"]
        if o.get("classification") == "PARTIAL" and o.get("stopCode"):
            seen_stops.add((o["stopCode"], o["stopState"], o.get("safePoint")))
        if o.get("classification") in ("COMPLETE", "PARTIAL"):
            seen_transitions.update((f, e) for f, e in o.get("transitions", []))
            for d in o.get("interruptDispositions") or []:
                seen_points.add(d)
    admissible = set()
    for code, r in spec["stopCodes"].items():
        for s in r["states"]:
            if code == "operator":
                for name, p in spec["safePoints"]["points"].items():
                    if p["kind"] == "STOP" and s in p["states"]:
                        admissible.add((code, s, name))
            else:
                admissible.add((code, s, None))
    transitions = set(table)
    return {"rules": dict(sorted(seen_rules.items())),
            "transitionsInContract": len(transitions),
            "transitionsObserved": sorted([str(f), e] for f, e in seen_transitions & transitions),
            "transitionsNotObserved": sorted([str(f), e] for f, e in transitions - seen_transitions),
            "stopStatesAdmissible": len(admissible),
            "stopStatesObserved": sorted([list(x) for x in seen_stops & admissible], key=str),
            "stopStatesNotObserved": sorted([list(x) for x in admissible - seen_stops], key=str),
            "stopStatesOutsideContract": sorted([list(x) for x in seen_stops - admissible], key=str),
            "interruptDispositionsObserved": sorted(seen_points)}


def cmd_run(a):
    os.makedirs(a.scratch, exist_ok=True)
    spec, bases, cases, declared, reached = enumerate_cases(a.binary, a.scratch, a.bases.split(",") if a.bases else None)
    work = [(a.binary, a.scratch, c, True) for c in cases]
    observed = {}
    with concurrent.futures.ProcessPoolExecutor(max_workers=a.jobs) as pool:
        for cid, obs in pool.map(_case, work, chunksize=4):
            observed[cid] = obs
    results = []
    for c in cases:
        o = observed[c["id"]]
        ctx = c["context"]
        mismatch = ["harness: " + o["harnessError"]] if "harnessError" in o else agree(c["expected"], o, ctx)
        if "harnessError" not in o and not o.get("fired"):
            mismatch.append("fault did not fire at its point")
        results.append(dict(c, observed=o, agree=not mismatch, mismatch=mismatch))
    cov = coverage(spec, results)
    for r in results:
        r["observed"] = stored(r["observed"])
    doc = {"schema": SCHEMA, "lifecycle": {"id": LS.LIFECYCLE_ID, "sha256": LS.spec_sha256()},
           "notice": "Generated from the safety contract; scripted model and scripted HTTP only; real SIGINT to the "
                     "recording process for kind signal; no inference.",
           "bases": bases, "declaredPoints": sorted(f"{b}.{p}" for b, p in declared),
           "unreachedDeclaredPoints": sorted(f"{b}.{p}" for b, p in declared - reached),
           "cases": results, "coverage": cov,
           "summary": {"cases": len(results), "agree": sum(r["agree"] for r in results),
                       "disagree": sum(not r["agree"] for r in results),
                       "points": len({(r["base"], r["index"]) for r in results}),
                       "falseComplete": sum("FALSE COMPLETE" in r["mismatch"] for r in results),
                       "classification": {k: sum(r["observed"].get("classification") == k for r in results)
                                          for k in ("COMPLETE", "PARTIAL", "INVALID")},
                       "kinds": {k: sum(r["kind"] == k for r in results) for k in sorted({r["kind"] for r in results})},
                       "basesAgree": all(b["agree"] for b in bases.values())}}
    S.D.write_json(a.out, doc)
    s = doc["summary"]
    print(f"fault matrix: {s['cases']} cases at {s['points']} points; {s['agree']} agree, {s['disagree']} disagree; "
          f"false COMPLETE {s['falseComplete']}; unreached declared points: {len(doc['unreachedDeclaredPoints'])}; "
          f"transitions not observed: {len(cov['transitionsNotObserved'])}; stop states not observed: "
          f"{len(cov['stopStatesNotObserved'])}")
    for r in results:
        if not r["agree"]:
            print("DISAGREE", r["id"], r["mismatch"])
    if s["disagree"] or doc["unreachedDeclaredPoints"] or not s["basesAgree"] or (not a.bases and (
            cov["transitionsNotObserved"] or cov["stopStatesNotObserved"] or cov["stopStatesOutsideContract"])):
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
        if stored(obs) != c["observed"] or not c["agree"]:
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
    p.add_argument("--bases")
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
