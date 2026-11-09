#!/usr/bin/env python3
"""Regression witnesses: every counterexample of the V0.13i independent review
(DT13I-01..06, from its adversarial, forgeries, extra-forgeries and
health-interrupt-edges scripts) and the earlier reviews' witnesses (V0.13h
DT13H-01..03, V0.13g lifecycle edges and partial probes), re-expressed
against the V0.13j recorder and verifier. Contradictory-archive witnesses
are in v13j_mutations.py and real-process termination in v13j_signals.py;
the ones here record genuine runs in-process (real SIGINT via os.kill,
genuine exception objects, persistent filesystem failures). Expected
outcomes follow from the safety contract; where a point of the base trace is
named, the expectation is computed by the contract rule the fault matrix uses.

    python3 -I -B analysis/alternatives/v13j_witnesses.py --binary ./dt --scratch DIR --out FILE
"""
import argparse
import errno
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
from unittest import mock

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13j_scenarios as S  # noqa: E402
import v13j_faults as M  # noqa: E402

D, F, J = S.D, S.F, S.J
SCHEMA = "dreamteam.v13j-witnesses/v1"


class At:
    """At the nth occurrence of a declared point: deliver a real SIGINT, raise
    an exception, or activate a persistent filesystem failure."""

    def __init__(self, boundary, phase, n=1, event=None, action="signal", error=None, fs=None, record=None):
        self.key, self.n, self.event, self.action, self.error, self.fs = (boundary, phase), n, event, action, error, fs
        self.record, self.seen = record, 0

    def __call__(self, boundary, phase, ctx):
        if (boundary, phase) != self.key or (self.event and ctx.get("event") != self.event) \
                or (self.record and ctx.get("record") != self.record):
            return
        self.seen += 1
        if self.seen != self.n:
            return
        if self.action == "signal":
            os.kill(os.getpid(), signal.SIGINT)
        elif self.action == "persistent":
            self.fs.activate()
        else:
            raise self.error


class Jump:
    def __init__(self, now):
        self.now, self.seen = now, 0

    def __call__(self, boundary, phase, ctx):
        if (boundary, phase) == ("draw_guard", "during"):
            self.seen += 1
            if self.seen == 1:
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


class Tracer:
    """A real SIGINT on a named source line of the unchanged draw/transport code."""

    def __init__(self, fname, func, text):
        self.fname, self.func, self.text, self.fired = fname, func, text, False

    def __call__(self, frame, event, arg):
        if self.fired:
            return None
        if frame.f_code.co_name == self.func and os.path.basename(frame.f_code.co_filename) == self.fname:
            if event == "line":
                import linecache
                if linecache.getline(frame.f_code.co_filename, frame.f_lineno).strip() == self.text:
                    self.fired = True
                    os.kill(os.getpid(), signal.SIGINT)
                    return None
            return self
        return self if event == "call" else None


def unrecorded_runner(entry, c, settings, opener, mem, transport, limits, utc=None, now=None):
    limits.before_call(0)
    transport({"model": "scripted", "messages": [{"role": "user", "content": "unrecorded draw"}]})
    raise RuntimeError("the draw raised without evidence after one dispatched call")


def P(**k):
    return dict({"classification": "PARTIAL", "recorder": "returned", "sealed": True, "cleanup": "SUCCEEDED",
                 "abortedStage": None}, **k)


INVALID = {"classification": "INVALID", "recorder": "RecordingError", "sealed": False}


def witnesses():
    """(name, source, setup(fixture, pool, clock, now, fs) -> record kwargs, expected, fixture)."""
    W = []

    def w(name, source, setup, expected, fixture="one", trace=None, patch=None):
        W.append({"name": name, "source": source, "setup": setup, "expected": expected, "fixture": fixture,
                  "trace": trace, "patch": patch})
    healthy = lambda f, pool, clock, now, fs: {"health": S.Health(clock)}  # noqa: E731

    # --- DT13I-01: handled SIGINT inside the unchanged transport and draw bookkeeping -----------------
    for name, fname, func, text in [("sigint_after_transport", "stage_b.py", "call", "lat = self.now() - t0"),
                                    ("sigint_before_calls_append", "stage_b.py", "call", "f = body_fields(body)"),
                                    ("sigint_draw_finally", "v13h.py", "run_draw", 'out["calls"] = model.calls if model else []'),
                                    ("sigint_draw_return", "v13h.py", "run_draw", "return out")]:
        w("DT13I-01/" + name, "V0.13i review adversarial.py " + name, healthy,
          P(stopCode="operator", stopState="DRAW_RECORDED", safePoint="after_draw_persisted", completed=1, modelCalls=1,
            journalDispatched=1, journalUnresolved=0, interruptDispositions=["STOP"]),
          trace=(fname, func, text))
    w("DT13I-01/transport_raised_call_kept", "V0.13j contract callJournal raised",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock), "model": raising(RuntimeError("transport broke"))},
      P(stopCode="driver", stopState="DRAW_ACTIVE", completed=0, modelCalls=1, abortedStage="workflow",
        journalDispatched=1, journalUnresolved=0))
    w("DT13I-01/unrecorded_draw_calls_not_zero", "V0.13j contract abortedDraw.draw_unrecorded",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock), "draw_runner": unrecorded_runner},
      P(stopCode="driver", stopState="DRAW_ACTIVE", completed=0, modelCalls=1, abortedStage="draw_unrecorded",
        journalDispatched=1, journalUnresolved=0))
    w("DT13I-01/operator_before_second_dispatch", "V0.13j safe point before_dispatch",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock), "model": S.Model([None, F.ABANDON, F.ABANDON]),
                                       "hook": At("safe_point", "after_call_persisted", 1)},
      P(stopCode="operator", stopState="DRAW_ACTIVE", safePoint="before_dispatch", completed=0, modelCalls=1,
        abortedStage="workflow", journalDispatched=1, interruptDispositions=["STOP"]))

    # --- DT13I-02: no completion without the seal; persistent failures defeat recovery -------------
    for name, point, exp in [
            ("persistent_from_final_manifest", ("persist:manifest", "write", 1, "cleanup_succeeded"), INVALID),
            ("persistent_before_seal", ("seal", "before", 1, None), INVALID),
            ("persistent_during_seal_write", ("seal", "write", 1, None), INVALID),
            ("persistent_during_seal_fsync", ("seal", "file_fsync", 1, None), INVALID),
            ("persistent_before_seal_rename", ("seal", "rename", 1, None), INVALID),
            ("persistent_after_seal_rename", ("seal", "renamed", 1, None),
             {"classification": "COMPLETE", "recorder": "returned", "sealed": True, "directorySync": "FAILED"})]:
        w("DT13I-02/" + name, "V0.13i review adversarial.py marker_restore_io_failure",
          lambda f, pool, clock, now, fs, point=point: {"health": S.Health(clock), "fs": fs,
                                                        "hook": At(point[0], point[1], point[2], point[3],
                                                                   action="persistent", fs=fs)}, exp)
    w("DT13I-02/os_patched_persistent_failure_after_final_manifest", "V0.13i review adversarial.py (os functions patched)",
      healthy, INVALID, patch="after_final_manifest")
    for name, phase, error in [("seal_write_error", "write", OSError(errno.EIO, "seal write")),
                               ("seal_fsync_error", "file_fsync", OSError(errno.EIO, "seal fsync")),
                               ("seal_rename_error", "rename", OSError(errno.EIO, "seal rename")),
                               ("seal_exception_before", "before", RuntimeError("before the seal"))]:
        w("DT13I-02/" + name, "V0.13j finalization steps 5-6",
          lambda f, pool, clock, now, fs, phase=phase, error=error: {"health": S.Health(clock),
                                                                     "hook": At("seal", phase, action="raise", error=error)},
          INVALID)
    w("DT13I-02/post_commit_directory_sync_failure", "V0.13j finalization step 7",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock),
                                       "hook": At("seal", "dir_fsync", action="raise", error=OSError(errno.EIO, "dir sync"))},
      {"classification": "COMPLETE", "recorder": "returned", "sealed": True, "directorySync": "FAILED"})

    # --- DT13I-03: genuine exceptions through the production probe ------------------------------------
    for mode, cat in [("url_incomplete", "incomplete_read"), ("url_bad_status", "http_protocol"),
                      ("not_connected", "http_protocol"), ("remote_disconnected", "remote_disconnected"),
                      ("url_text", "url_error"), ("unreachable", "network_os_error"), ("dns", "network_os_error"),
                      ("refused", "connection_refused"), ("timeout", "timeout"), ("connect_timeout", "timeout"),
                      ("reset", "connection_reset"), ("disconnect", "remote_disconnected"),
                      ("incomplete", "incomplete_read"), ("bad_status_line", "http_protocol"),
                      ("http_503", "http_status")]:
        w("DT13I-03/genuine_" + mode, "V0.13i review adversarial.py health families",
          lambda f, pool, clock, now, fs, mode=mode: {"health": S.Health(clock, modes={1: mode})},
          P(stopCode="health", stopState="PREFLIGHT", completed=0, modelCalls=0, category=cat))
    for mode in ("keyboard", "value_error", "permission", "url_keyboard"):
        w("DT13I-03/not_a_read_failure_" + mode, "V0.13j health readFailureRule",
          lambda f, pool, clock, now, fs, mode=mode: {"health": S.Health(clock, modes={1: mode})},
          P(stopCode="driver", stopState="PREFLIGHT", completed=0, modelCalls=0, attemptedRead=True))
    for name, typ in [("os_error_KeyboardInterrupt", "builtins.KeyboardInterrupt"), ("os_error_ValueError", "builtins.ValueError"),
                      ("os_error_MadeUpNonOSError", "builtins.MadeUpNonOSError")]:
        def masquerade(f, pool, clock, now, fs, typ=typ):
            def state():
                obs = J.probe_health("http://localhost:9", urlopen=S.scripted_urlopen("unreachable"), utc=clock)
                obs["exception"] = {"type": typ, "reasonType": None, "reasonKind": None}
                return {"health": obs}
            return {"health": state}
        w("DT13I-03/" + name, "V0.13i review forgeries.py " + name, masquerade,
          P(stopCode="driver", stopState="PREFLIGHT", completed=0, rejected=True))
    for name, change in [("healthy_http_503", {"httpStatus": 503}), ("healthy_hash_wrong", "hash"),
                         ("healthy_null_time", {"started": None})]:
        def edge(f, pool, clock, now, fs, change=change):
            inner = S.Health(clock)
            n = [0]

            def state():
                s = inner()
                n[0] += 1
                if n[0] == 1:
                    if change == "hash":
                        s["health"]["body"]["sha256"] = "0" * 64
                    else:
                        s["health"].update(change)
                return s
            return {"health": state}
        w("DT13I-03/" + name, "V0.13i review health_and_interrupt_edges.py " + name, edge,
          P(stopCode="driver", stopState="PREFLIGHT", completed=0, rejected=True))

    # --- DT13I-05: deferred interrupts visible; cleanup never skipped -----------------------------------
    w("DT13I-05/sigint_during_final_manifest_public_api", "V0.13i review adversarial.py deferred_public_api",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock),
                                       "hook": At("persist:manifest", "file_fsync", 1, "cleanup_succeeded")},
      {"classification": "COMPLETE", "recorder": "returned", "sealed": True, "unarchived": 1, "signalsReceived": 1,
       "interruptDispositions": []})
    w("DT13I-05/sigint_during_seal_write_public_api", "V0.13i review adversarial.py deferred_public_api",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock), "hook": At("seal", "write")},
      {"classification": "COMPLETE", "recorder": "returned", "sealed": True, "unarchived": 1, "signalsReceived": 1})
    w("DT13I-05/sigint_before_cleanup_releases_pool", "V0.13i review health_and_interrupt_edges.py sigint_before_cleanup",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock), "hook": At("session_cleanup", "before")},
      {"classification": "COMPLETE", "recorder": "returned", "sealed": True, "poolReleases": 1, "unarchived": 0,
       "interruptDispositions": ["AFTER_SCHEDULE"], "postCompletion": True})
    w("DT13I-05/sigint_at_before_cleanup_after_stop", "V0.13j safe point before_cleanup",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock, modes={3: "unhealthy"}),
                                       "hook": At("safe_point", "before_cleanup")},
      P(stopCode="health", stopState="HEALTH_CHECK", completed=1, poolReleases=1, interruptDispositions=["AFTER_STOP"]))
    w("DT13I-05/two_signals_one_stop", "V0.13j SignalLatch",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock), "hook": lambda b, p, c: (
          os.kill(os.getpid(), signal.SIGINT), os.kill(os.getpid(), signal.SIGINT))
          if (b, p) == ("draw_execute", "model_call") and c["dispatched"] == 1 else None},
      P(stopCode="operator", stopState="DRAW_RECORDED", safePoint="after_draw_persisted", completed=1, modelCalls=1,
        signalsReceived=2, interruptDispositions=["STOP"]))

    # --- predecessor witnesses: DT13H-01 legacy health shapes -------------------------------------------
    for name, health in [("legacy_error_object", {"error": "URLError"}),
                         ("legacy_healthy_error_object", {"status": "healthy", "error": "URLError"}),
                         ("health_field_missing", None)]:
        w("DT13H-01/" + name, "V0.13h re-review section 4 (legacy shapes)",
          lambda f, pool, clock, now, fs, health=health: {"health": lambda: {"health": health} if health is not None else {}},
          P(stopCode="driver", stopState="PREFLIGHT", completed=0, modelCalls=0, rejected=True))
    w("DT13H-01/unavailable_between_draws", "V0.13h re-review section 4 B",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock, modes={3: "refused"})},
      P(stopCode="health", stopState="HEALTH_CHECK", completed=1, modelCalls=1, category="connection_refused"))

    # --- DT13H-03: final persistence faults never complete ---------------------------------------------
    for name, b, p, error in [("final_manifest_replace", "persist:manifest", "replace", OSError(errno.EIO, "replace")),
                              ("final_manifest_replaced", "persist:manifest", "replaced", OSError(errno.EIO, "after")),
                              ("final_manifest_dir_fsync", "persist:manifest", "dir_fsync", OSError(errno.EIO, "dir")),
                              ("failure_event_write", "persist:failure_event", "write", OSError(errno.EIO, "fe"))]:
        health = (lambda clock: S.Health(clock, modes={3: "unhealthy"})) if "failure_event" in b else S.Health
        event = None if "failure_event" in b else "cleanup_succeeded"
        w("DT13H-03/" + name, "V0.13h re-review section 5",
          lambda f, pool, clock, now, fs, b=b, p=p, error=error, health=health, event=event:
          {"health": health(clock), "hook": At(b, p, 1, event, action="raise", error=error)}, INVALID)
    w("DT13H-03/failed_finalization_comparison", "V0.13h re-review atomic-comparisons",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock),
                                       "hook": At("persist:manifest", "dir_fsync", 1, "cleanup_succeeded", action="raise",
                                                  error=OSError(errno.EIO, "final directory sync failed"))},
      dict(INVALID, analysisRefused=True), fixture="both")

    # --- V0.13g lifecycle edges and partial probes -------------------------------------------------------
    w("DT13G/aborted-then-close-interrupt", "V0.13g re-review DT13G-02",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock),
                                       "opener": S.FailingOpener(pool, close_error=KeyboardInterrupt("pool shutdown")),
                                       "model": raising(RuntimeError("scripted draw failure"))},
      P(stopCode="driver", stopState="DRAW_ACTIVE", completed=0, cleanup="FAILED", abortedStage="workflow"))
    w("DT13G/completed-then-close-interrupt", "V0.13g re-review DT13G-02",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock),
                                       "opener": S.FailingOpener(pool, close_error=KeyboardInterrupt("pool shutdown"))},
      P(stopCode="driver", stopState="CLEANUP", completed=2, cleanup="FAILED"))
    for name, error in [("runtime", RuntimeError("opening failure")), ("timeout", TimeoutError("opening timeout")),
                        ("interrupt", KeyboardInterrupt()), ("process", subprocess.CalledProcessError(3, ["engine"])),
                        ("missing", FileNotFoundError("engine absent"))]:
        w("DT13G/open-" + name, "V0.13g re-review DT13F-03 (an exception, KeyboardInterrupt included, is a driver failure)",
          lambda f, pool, clock, now, fs, error=error: {"health": S.Health(clock), "opener": S.FailingOpener(pool, error)},
          P(stopCode="driver", stopState="DRAW_ACTIVE", completed=0, abortedStage="engine_open"))
    for stage, recorded in (("initial_digests", "initial_digests"), ("workflow", "workflow"),
                            ("final_digests", "workflow"), ("engine_close", "engine_close")):
        w("DT13G/stage-" + stage, "V0.13g re-review DT13F-03",
          lambda f, pool, clock, now, fs, stage=stage: {"health": S.Health(clock), "opener": FailPool(pool, stage)},
          P(stopCode="driver", stopState="DRAW_ACTIVE", completed=0, abortedStage=recorded))
    w("DT13G/pool-close", "V0.13g re-review DT13F-03",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock),
                                       "opener": S.FailingOpener(pool, close_error=BrokenPipeError("pooled engine shutdown"))},
      P(stopCode="driver", stopState="CLEANUP", completed=2, cleanup="FAILED"))
    w("DT13G/wall-clock", "V0.13g re-review DT13F-07",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock), "hook": Jump(now)},
      P(stopCode="wall_clock", stopState="READY", completed=0))
    versions = iter(["one", "one", "two", "two"])
    w("DT13G/server-change", "V0.13g re-review DT13F-07",
      lambda f, pool, clock, now, fs: {"health": lambda h=S.Health(clock): dict(h(), serverVersion=next(versions))},
      P(stopCode="server_change", stopState="HEALTH_CHECK", completed=1))
    w("DT13G/transport-interrupt", "V0.13g re-review DT13F-03 (KeyboardInterrupt raised by the transport)",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock), "model": raising(KeyboardInterrupt())},
      P(stopCode="driver", stopState="DRAW_ACTIVE", completed=0, abortedStage="workflow", modelCalls=1))
    w("DT13G/call-budget", "V0.13g re-review DT13F-07", healthy,
      P(stopCode="call_budget", stopState="DRAW_RECORDED", completed=0), fixture=("maxModelCalls", 0))
    w("DT13G/draw-limit", "V0.13g re-review DT13F-07", healthy,
      P(stopCode="draw_call_limit", stopState="DRAW_RECORDED", completed=0), fixture=("maxModelCallsPerDraw", 0))
    w("DT13G/transport-limit", "V0.13g re-review DT13F-07",
      lambda f, pool, clock, now, fs: {"health": S.Health(clock), "replies": [None, None, None]},
      P(stopCode="transport", stopState="DRAW_RECORDED", completed=0), fixture=("consecutiveTransportFailures", 1))
    return W


def patched_run(fn, live):
    """The independent review's persistent failure, with the os functions
    patched: from the moment the final manifest is in place, every
    directory fsync, file fsync, replace and open-for-write in the archive
    directory fails."""
    real_fsync, real_replace, real_os_open = os.fsync, os.replace, os.open
    on = [False]

    def fsync(fd):
        if on[0]:
            raise OSError(errno.EIO, "persistent filesystem failure")
        return real_fsync(fd)

    def replace(src, dst):
        if on[0]:
            raise OSError(errno.EIO, "persistent filesystem failure")
        r = real_replace(src, dst)
        if str(dst).endswith("run.json"):
            try:
                import json
                if json.loads(Path(dst).read_text()).get("state") == "FINALIZING":
                    on[0] = True
            except ValueError:
                pass
        return r

    def os_open(path, flags, *a, **k):
        if on[0] and str(path).startswith(str(live)) and flags & (os.O_WRONLY | os.O_RDWR | os.O_CREAT):
            raise OSError(errno.EIO, "persistent filesystem failure")
        return real_os_open(path, flags, *a, **k)
    with mock.patch.object(os, "fsync", fsync), mock.patch.object(os, "replace", replace), \
            mock.patch.object(os, "open", os_open):
        return fn()


def run(binary, scratch, prefix=""):
    base = {"one": S.Fixture(binary), "both": S.Fixture(binary, arms=("control", "treatment"))}
    results = []
    for wit in [x for x in witnesses() if x["name"].startswith(prefix)]:
        fx = wit["fixture"]
        fixture = base[fx] if isinstance(fx, str) else base["one"].with_limits(**{fx[0]: fx[1]})
        out = os.path.join(scratch, wit["name"].replace("/", "_"))
        shutil.rmtree(out, ignore_errors=True)
        clock, now, fs = S.Clock(), S.Monotonic(), S.FailingFS()
        pool = fixture.ctx.opener(binary)
        kw = wit["setup"](fixture, pool, clock, now, fs)
        opener = kw.pop("opener", pool)
        releases = []

        class Counting:
            def __call__(self, c, arm):
                return opener(c, arm)

            def close(self):
                releases.append(1)
                return opener.close()
        model = kw.pop("model", None)
        if model is None:
            model = S.Model(kw.pop("replies", None) or fixture.replies())
        counted = [0]

        def transport(payload, model=model):
            counted[0] += 1
            return model(payload)

        def go():
            return fixture.record(out, clock=clock, now=now, opener=Counting(), model=transport, **kw)
        tracer = Tracer(*wit["trace"]) if wit["trace"] else None
        if tracer:
            sys.settrace(tracer)
        try:
            if wit["patch"]:
                result = S.attempt(lambda: patched_run(go, Path(out) / "live"))
            else:
                result = S.attempt(go)
        finally:
            sys.settrace(None)
        obs = M.observe(fixture, out, result, False)
        obs.pop("transitions", None)
        obs["poolReleases"] = len(releases)
        obs["transportCalls"] = counted[0]
        if result.get("interrupts") is not None:
            obs["signalsReceived"] = result["interrupts"]["signalsReceived"]
            obs["postCompletion"] = result["interrupts"]["postCompletion"]
        live = Path(out) / "live"
        if obs.get("manifest"):
            import json
            doc = json.loads((live / "run.json").read_text())
            stop = doc.get("stop") or {}
            hc = stop.get("healthCheck")
            if hc is not None and hc < len(doc["healthChecks"]):
                check = doc["healthChecks"][hc]
                if check["serverState"]:
                    obs["category"] = check["serverState"]["health"]["category"]
                obs["rejected"] = check["rejected"] is not None
            obs["attemptedRead"] = stop.get("request") is not None
        try:
            a = fixture.analyze(out)
            obs["analysisRefused"] = False
            obs["analysis"] = {"accepted": True, "runStatus": a["runStatus"], "completedVerified": len(a["draws"]),
                               "comparisonsSuppressed": a["armDifferences"] == [] and a["pairs"] == [],
                               "aborted": (a["abortedVerification"] or {}).get("status")}
        except D.ExperimentError:
            obs["analysisRefused"] = True
            obs["analysis"] = {"accepted": False}
        bad = M.agree(wit["expected"], obs, None)
        if wit["expected"].get("modelCalls") is not None and obs.get("modelCalls") != obs["transportCalls"] \
                and wit["name"] != "DT13I-01/unrecorded_draw_calls_not_zero":
            bad.append("the archive's call count differs from the transport's actual invocations")
        if obs.get("recorder") == "returned" and obs["poolReleases"] != 1:
            bad.append("pooled sessions not released exactly once")
        if tracer and not tracer.fired:
            bad.append("the traced line was never reached")
        results.append({"name": wit["name"], "source": wit["source"], "expected": wit["expected"], "observed": obs,
                        "agree": not bad, "mismatch": bad})
        shutil.rmtree(out, ignore_errors=True)
    return results


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--prefix", default="")
    a = ap.parse_args(argv)
    os.makedirs(a.scratch, exist_ok=True)
    results = run(a.binary, a.scratch, a.prefix)
    D.write_json(a.out, {"schema": SCHEMA, "notice": "Scripted model, scripted HTTP, genuine exception objects and real "
                                                     "in-process SIGINT only; no inference.",
                         "witnesses": results, "summary": {"witnesses": len(results), "agree": sum(r["agree"] for r in results)}})
    print(f"witnesses: {sum(r['agree'] for r in results)} of {len(results)} agree")
    for r in results:
        if not r["agree"]:
            print("DISAGREE", r["name"], r["mismatch"])
    if not all(r["agree"] for r in results):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
