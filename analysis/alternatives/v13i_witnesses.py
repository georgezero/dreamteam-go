#!/usr/bin/env python3
"""Regression witnesses: the earlier independent reviews' counterexamples
(V0.13g lifecycle edges and 21 partial probes; V0.13h DT13H-01 health
matrix, DT13H-02 boundary interruptions, DT13H-03 final persistence and the
failed-finalization comparison), re-expressed against the V0.13i recorder
and verifier. They check the design; they are not the design. Each expected
outcome follows from the lifecycle contract.

    python3 -I -B analysis/alternatives/v13i_witnesses.py --binary ./dt --scratch DIR --out FILE
"""
import argparse
import errno
import os
from pathlib import Path
import shutil
import subprocess
import sys

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13i_scenarios as S  # noqa: E402

D, F = S.D, S.F
SCHEMA = "dreamteam.v13i-witnesses/v1"


class At:
    """Raise once at the nth occurrence of a declared point."""

    def __init__(self, boundary, phase, n=1, event=None, error=None):
        self.key, self.n, self.event = (boundary, phase), n, event
        self.error = error if error is not None else KeyboardInterrupt(f"witness interrupt at {boundary}.{phase}#{n}")
        self.seen = 0

    def __call__(self, boundary, phase, ctx):
        if (boundary, phase) == self.key and (self.event is None or ctx.get("event") == self.event):
            self.seen += 1
            if self.seen == self.n:
                raise self.error


class Jump:
    def __init__(self, now, boundary="draw_guard", phase="during", n=1):
        self.now, self.key, self.n, self.seen = now, (boundary, phase), n, 0

    def __call__(self, boundary, phase, ctx):
        if (boundary, phase) == self.key:
            self.seen += 1
            if self.seen == self.n:
                self.now.jump()


class FailingSession:
    def __init__(self, s, stage):
        self.s, self.stage, self.n = s, stage, 0

    def request(self, rid, op, args=None):
        if op == "digests":
            self.n += 1
        if (self.stage == "initial_digests" and op == "digests" and self.n == 1) or \
                (self.stage == "workflow" and op == "actions") or \
                (self.stage == "final_digests" and op == "digests" and self.n == 2):
            raise BrokenPipeError("scripted engine pipe failure")
        return self.s.request(rid, op, args)

    def close(self):
        self.s.close()
        if self.stage == "engine_close":
            raise RuntimeError("scripted session close failure")


class FailPool:
    def __init__(self, pool, stage):
        self.pool, self.stage = pool, stage

    def __call__(self, c, arm):
        return FailingSession(self.pool(c, arm), self.stage)

    def close(self):
        self.pool.close()


def raising(error):
    def transport(payload):
        raise error
    return transport


def P(**k):
    return dict({"classification": "PARTIAL", "recorder": "returned", "cleanup": "SUCCEEDED", "abortedStage": None}, **k)


INVALID = {"classification": "INVALID", "recorder": "RecordingError"}


def witnesses():
    """(name, source, setup, expected). setup(fixture, pool, clock, now) -> record kwargs."""
    W = []

    def w(name, source, setup, expected, fixture="one"):
        W.append({"name": name, "source": source, "setup": setup, "expected": expected, "fixture": fixture})
    # DT13H-01: health matrix A-F and the additional attacks (production probe, genuine failures).
    for name, mode, at, cat in [("A_unavailable_before_any_draw", "refused", 1, "connection_refused"),
                                ("B_unavailable_between_draws", "refused", 3, "connection_refused"),
                                ("C_connect_timeout", "connect_timeout", 1, "timeout"),
                                ("D_read_timeout", "timeout", 1, "timeout"),
                                ("E_malformed_json", "malformed", 1, "invalid_json"),
                                ("E_json_list", "list", 1, "not_an_object"),
                                ("E_json_null", "null", 1, "not_an_object"),
                                ("F_explicit_unhealthy", "unhealthy", 1, "unhealthy_status"),
                                ("empty_connection_remote_disconnected", "disconnect", 1, "remote_disconnected"),
                                ("truncated_body_incomplete_read", "incomplete", 1, "incomplete_read"),
                                ("healthy_with_error_contradictory", "healthy_error", 1, "contradictory_body"),
                                ("http_503", "http_503", 1, "http_status"),
                                ("connection_reset", "reset", 1, "connection_reset")]:
        w("DT13H-01/" + name, "V0.13h re-review section 4",
          lambda f, pool, clock, now, mode=mode, at=at: {"health": S.Health(clock, modes={at: mode})},
          P(stopCode="health", stopState="HEALTH_CHECK" if at > 1 else "PREFLIGHT", completed=1 if at > 1 else 0,
            modelCalls=1 if at > 1 else 0, category=cat))
    for name, health in [("legacy_error_object", {"error": "URLError"}),
                         ("legacy_healthy_error_object", {"status": "healthy", "error": "URLError"}),
                         ("health_field_missing", None)]:
        w("DT13H-01/" + name, "V0.13h re-review section 4 (legacy shapes)",
          lambda f, pool, clock, now, health=health: {"health": lambda: {"health": health} if health is not None else {}},
          P(stopCode="driver", stopState="PREFLIGHT", completed=0, modelCalls=0))
    # DT13H-02: interruptions at the 20 enclosing-run boundaries, mapped to declared points.
    healthy = lambda f, pool, clock, now: {"health": S.Health(clock)}  # noqa: E731
    for name, point, expected in [
            ("start_uuid", ("persist:ledger_init", "before", 1, None), INVALID),
            ("start_timestamp", ("persist:ledger_init", "after", 1, None), INVALID),
            ("after_initial_save", ("persist:manifest", "after", 1, "create"), P(stopCode="operator", stopState="CREATED", completed=0)),
            ("setup", ("persist:manifest", "after", 1, "begin_preflight"), P(stopCode="operator", stopState="PREFLIGHT", completed=0)),
            ("first_health", ("health_read", "during", 1, None), P(stopCode="operator", stopState="PREFLIGHT", completed=0)),
            ("before_draw_guard", ("draw_guard", "before", 1, None), P(stopCode="operator", stopState="READY", completed=0)),
            ("draw_setup", ("draw_execute", "before", 1, None), P(stopCode="operator", stopState="DRAW_ACTIVE", completed=0, abortedStage="draw_setup")),
            ("after_draw_returns", ("draw_execute", "after", 1, None), P(stopCode="operator", stopState="DRAW_RECORDED", completed=1)),
            ("after_ledger", ("persist:ledger", "after", 1, None), P(stopCode="operator", stopState="DRAW_RECORDED", completed=1)),
            ("before_completed_count", ("persist:manifest", "before", 1, "commit_draw"), P(stopCode="operator", stopState="DRAW_RECORDED", completed=1)),
            ("after_completed_count", ("persist:manifest", "after", 1, "commit_draw"), P(stopCode="operator", stopState="DRAW_RECORDED", completed=1)),
            ("next_iteration", ("persist:manifest", "after", 1, "continue"), P(stopCode="operator", stopState="READY", completed=1)),
            ("second_health", ("health_read", "during", 3, None), P(stopCode="operator", stopState="HEALTH_CHECK", completed=1)),
            ("second_guard", ("draw_guard", "before", 2, None), P(stopCode="operator", stopState="READY", completed=1)),
            ("second_draw_returns", ("draw_execute", "after", 2, None), P(stopCode="operator", stopState="DRAW_RECORDED", completed=2)),
            ("before_cleanup", ("persist:manifest", "before", 1, "schedule_done"), P(stopCode="operator", stopState="DRAW_RECORDED", completed=2)),
            ("cleanup_call", ("session_cleanup", "during", 1, None), P(stopCode="operator", stopState="CLEANUP", completed=2, cleanup="FAILED")),
            ("after_cleanup", ("session_cleanup", "after", 1, None), INVALID),
            ("final_timestamp", ("persist:marker", "before", 1, None), INVALID),
            ("final_write_call", ("persist:final_manifest", "before", 1, None), INVALID)]:
        w("DT13H-02/" + name, "V0.13h re-review section 5",
          lambda f, pool, clock, now, point=point: dict(healthy(f, pool, clock, now), hook=At(point[0], point[1], point[2], point[3])),
          expected)
    w("DT13H-02/cleanup_only_after_all_draws", "V0.13h re-review section 5",
      lambda f, pool, clock, now: {"opener": S.FailingOpener(pool, close_error=KeyboardInterrupt("cleanup failure"))},
      P(stopCode="operator", stopState="CLEANUP", completed=2, cleanup="FAILED"))
    w("DT13H-02/primary_and_cleanup", "V0.13h re-review section 5",
      lambda f, pool, clock, now: {"opener": S.FailingOpener(pool, close_error=KeyboardInterrupt("cleanup failure")),
                                   "model": raising(ValueError("primary failure"))},
      P(stopCode="driver", stopState="DRAW_ACTIVE", completed=0, cleanup="FAILED", abortedStage="workflow"))
    # DT13H-03: final persistence and the failed-finalization comparison.
    for name, phase, error in [("replace_before", "replace", OSError(errno.EIO, "final replacement failed")),
                               ("replace_after", "replaced", OSError(errno.EIO, "failure after final replacement")),
                               ("fsync_file", "file_fsync", KeyboardInterrupt("final file sync interrupted")),
                               ("fsync_directory", "dir_fsync", OSError(errno.EIO, "final directory sync failed")),
                               ("replace_after_interrupt", "replaced", KeyboardInterrupt("interrupt after final replacement"))]:
        w("DT13H-03/" + name, "V0.13h re-review section 5",
          lambda f, pool, clock, now, phase=phase, error=error: {"hook": At("persist:final_manifest", phase, error=error)},
          dict(INVALID, marker=True))
    w("DT13H-03/marker_removal_directory_sync", "V0.13i contract finalization step 5",
      lambda f, pool, clock, now: {"hook": At("unlink:marker", "dir_fsync", error=OSError(errno.EIO, "directory sync failed"))},
      dict(INVALID, marker=True))
    w("DT13H-03/failed_finalization_comparison", "V0.13h re-review atomic-comparisons",
      lambda f, pool, clock, now: {"hook": At("persist:final_manifest", "dir_fsync", error=OSError(errno.EIO, "final directory sync failed"))},
      dict(INVALID, marker=True, analysisRefused=True), fixture="both")
    # V0.13g lifecycle edges.
    w("DT13G/lifecycle-edges/aborted-then-close-interrupt", "V0.13g re-review DT13G-02",
      lambda f, pool, clock, now: {"opener": S.FailingOpener(pool, close_error=KeyboardInterrupt("pool shutdown")),
                                   "model": raising(RuntimeError("scripted draw failure"))},
      P(stopCode="driver", stopState="DRAW_ACTIVE", completed=0, cleanup="FAILED", abortedStage="workflow"))
    w("DT13G/lifecycle-edges/completed-then-close-interrupt", "V0.13g re-review DT13G-02",
      lambda f, pool, clock, now: {"opener": S.FailingOpener(pool, close_error=KeyboardInterrupt("pool shutdown")),
                                   "replies": [F.ABANDON]},
      P(stopCode="driver", stopState="DRAW_ACTIVE", completed=1, cleanup="FAILED", abortedStage="workflow"))
    w("DT13G/lifecycle-edges/initial-state-interrupt", "V0.13g re-review DT13G-02",
      lambda f, pool, clock, now: {"health": S.Health(clock), "hook": At("health_read", "during", 1)},
      P(stopCode="operator", stopState="PREFLIGHT", completed=0))
    w("DT13G/lifecycle-edges/next-state-interrupt", "V0.13g re-review DT13G-02",
      lambda f, pool, clock, now: {"health": S.Health(clock), "hook": At("health_read", "during", 3)},
      P(stopCode="operator", stopState="HEALTH_CHECK", completed=1))
    # V0.13g 21 partial probes.
    for name, error, code in [("runtime", RuntimeError("opening failure"), "driver"),
                              ("timeout", TimeoutError("opening timeout"), "driver"),
                              ("interrupt", KeyboardInterrupt(), "operator"),
                              ("process", subprocess.CalledProcessError(3, ["engine"]), "driver"),
                              ("missing", FileNotFoundError("engine absent"), "driver")]:
        w("DT13G/partial-probes/open-" + name, "V0.13g re-review DT13F-03",
          lambda f, pool, clock, now, error=error: {"opener": S.FailingOpener(pool, error)},
          P(stopCode=code, stopState="DRAW_ACTIVE", completed=0, abortedStage="engine_open"))
    # The probe named final_digests fails the second digests request, which the
    # workflow itself issues; V0.13g recorded it as a workflow failure too.
    for stage, recorded in (("initial_digests", "initial_digests"), ("workflow", "workflow"),
                            ("final_digests", "workflow"), ("engine_close", "engine_close")):
        w("DT13G/partial-probes/" + stage, "V0.13g re-review DT13F-03",
          lambda f, pool, clock, now, stage=stage: {"opener": FailPool(pool, stage)},
          P(stopCode="driver", stopState="DRAW_ACTIVE", completed=0, abortedStage=recorded))
    w("DT13G/partial-probes/pool-close", "V0.13g re-review DT13F-03",
      lambda f, pool, clock, now: {"opener": S.FailingOpener(pool, close_error=BrokenPipeError("pooled engine shutdown"))},
      P(stopCode="driver", stopState="CLEANUP", completed=2, cleanup="FAILED"))
    for name, mode, cat in [("healthy-stop", "unhealthy", "unhealthy_status"), ("health-timeout", "timeout", "timeout"),
                            ("health-connection", "refused", "connection_refused")]:
        w("DT13G/partial-probes/" + name, "V0.13g re-review DT13F-07",
          lambda f, pool, clock, now, mode=mode: {"health": S.Health(clock, default=mode)},
          P(stopCode="health", stopState="PREFLIGHT", completed=0, category=cat))
    w("DT13G/partial-probes/health-missing", "V0.13g re-review DT13F-07",
      lambda f, pool, clock, now: {"health": lambda: {"health": {}}}, P(stopCode="driver", stopState="PREFLIGHT", completed=0))
    w("DT13G/partial-probes/wall-clock", "V0.13g re-review DT13F-07",
      lambda f, pool, clock, now: {"hook": Jump(now)}, P(stopCode="wall_clock", stopState="READY", completed=0))
    versions = iter(["one", "two", "two", "two"])
    w("DT13G/partial-probes/server-change", "V0.13g re-review DT13F-07",
      lambda f, pool, clock, now: {"health": lambda h=S.Health(clock): dict(h(), serverVersion=next(versions))},
      P(stopCode="server_change", stopState="HEALTH_CHECK", completed=0))
    w("DT13G/partial-probes/transport-interrupt", "V0.13g re-review DT13F-03",
      lambda f, pool, clock, now: {"model": raising(KeyboardInterrupt())},
      P(stopCode="operator", stopState="DRAW_ACTIVE", completed=0, abortedStage="workflow"))
    w("DT13G/partial-probes/transport-exception", "V0.13g re-review DT13F-03",
      lambda f, pool, clock, now: {"model": raising(RuntimeError("transport failure"))},
      P(stopCode="driver", stopState="DRAW_ACTIVE", completed=0, abortedStage="workflow"))
    w("DT13G/partial-probes/call-budget", "V0.13g re-review DT13F-07", lambda f, pool, clock, now: {},
      P(stopCode="call_budget", stopState="DRAW_RECORDED", completed=0), fixture=("maxModelCalls", 0))
    w("DT13G/partial-probes/draw-limit", "V0.13g re-review DT13F-07", lambda f, pool, clock, now: {},
      P(stopCode="draw_call_limit", stopState="DRAW_RECORDED", completed=0), fixture=("maxModelCallsPerDraw", 0))
    w("DT13G/partial-probes/transport-limit", "V0.13g re-review DT13F-07",
      lambda f, pool, clock, now: {"replies": [None, None, None]},
      P(stopCode="transport", stopState="DRAW_RECORDED", completed=0), fixture=("consecutiveTransportFailures", 1))

    def late_wall(f, pool, clock, now):
        state = {"n": 0}

        def model(payload):
            state["n"] += 1
            now.jump()
            return None, "", "timeout: scripted"
        return {"model": model}
    w("V0.13i/wall-clock-at-request-start", "V0.13i contract stop codes",
      late_wall, P(stopCode="wall_clock", stopState="DRAW_RECORDED", completed=0))
    return W


def run(binary, scratch, prefix=""):
    base = {"one": S.Fixture(binary), "both": S.Fixture(binary, arms=("control", "treatment"))}
    results = []
    for wit in [x for x in witnesses() if x["name"].startswith(prefix)]:
        fx = wit["fixture"]
        fixture = base[fx] if isinstance(fx, str) else base["one"].with_limits(**{fx[0]: fx[1]})
        out = os.path.join(scratch, wit["name"].replace("/", "_"))
        shutil.rmtree(out, ignore_errors=True)
        clock, now = S.Clock(), S.Monotonic()
        pool = fixture.ctx.opener(binary)
        kw = wit["setup"](fixture, pool, clock, now)
        kw.setdefault("opener", pool)
        if "health" in kw and isinstance(kw["health"], S.Health):
            kw["health"].utc = clock
        result = S.attempt(lambda: fixture.record(out, clock=clock, now=now, **kw))
        obs = S.summary(fixture, out, result)
        obs["recorder"] = "returned" if result["returned"] else result["error"]
        live = Path(out) / "live"
        if obs.get("manifest"):
            import json
            doc = json.loads((live / "run.json").read_text())
            stop = doc.get("stop") or {}
            if stop.get("healthCheck") is not None and doc["healthChecks"][stop["healthCheck"]]["serverState"]:
                obs["category"] = doc["healthChecks"][stop["healthCheck"]]["serverState"]["health"]["category"]
        try:
            a = fixture.analyze(out)
            obs["analysisRefused"] = False
            obs["analysis"] = {"runStatus": a["runStatus"], "completedVerified": len(a["draws"]),
                               "comparisonsSuppressed": a["armDifferences"] == [] and a["pairs"] == [],
                               "aborted": (a["abortedVerification"] or {}).get("status")}
        except D.ExperimentError:
            obs["analysisRefused"] = True
        bad = [k for k, v in wit["expected"].items() if obs.get(k) != v]
        if obs["classification"] in ("PARTIAL", "COMPLETE") and obs["analysisRefused"]:
            bad.append("analysis refused a valid archive")
        if obs["classification"] == "PARTIAL" and not obs.get("analysis", {}).get("comparisonsSuppressed", True):
            bad.append("complete-run comparison for PARTIAL")
        results.append({"name": wit["name"], "source": wit["source"], "expected": wit["expected"], "observed": obs,
                        "agree": not bad, "mismatch": bad})
        shutil.rmtree(out, ignore_errors=True)
    return results


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    os.makedirs(a.scratch, exist_ok=True)
    results = run(a.binary, a.scratch)
    D.write_json(a.out, {"schema": SCHEMA, "notice": "Scripted model and scripted HTTP only; no inference.",
                         "witnesses": results, "summary": {"witnesses": len(results), "agree": sum(r["agree"] for r in results)}})
    print(f"witnesses: {sum(r['agree'] for r in results)} of {len(results)} agree")
    for r in results:
        if not r["agree"]:
            print("DISAGREE", r["name"], r["mismatch"], r["observed"])
    if not all(r["agree"] for r in results):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
