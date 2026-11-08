#!/usr/bin/env python3
"""The V0.13i experiment driver (dreamteam-v13i-driver/5): the V0.13h draw,
context, replay and protocol machinery unchanged, with the enclosing run
recorded through the explicit lifecycle contract dreamteam.run-lifecycle/v1
(analysis/alternatives/lifecycle_v1.json, docs/v13i/LIFECYCLE.md).

    python3 -I -B analysis/alternatives/v13i.py protocol --contexts DIR --out FILE
    python3 -I -B analysis/alternatives/v13i.py schedule --contexts DIR --protocol FILE --out FILE
    python3 -I -B analysis/alternatives/v13i.py check --binary ./dt --contexts DIR --protocol FILE --replays DIR
    python3 -I -B analysis/alternatives/v13i.py run ... (refused: the protocol is not locked)

Offline verification and analysis are in v13i_verify.py, which does not
import this module. No command contacts a model.
"""
import argparse
import base64
import copy
import errno
import hashlib
import http.client
import json
import os
import signal
import socket
import sys
import tempfile
import threading
import urllib.error
import urllib.request
import uuid

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import lifecycle_spec as LS  # noqa: E402
import v13h as H  # noqa: E402  (V0.13h contexts, draws, replay and protocol, unchanged)

SB, X, RR = H.SB, H.X, H.SB.RR
sha256, canon, write_json, Halt = H.sha256, H.canon, H.write_json, H.Halt
ExperimentError = H.ExperimentError
QWEN, GEMMA = H.QWEN, H.GEMMA

DRIVER = "dreamteam-v13i-driver/5"
PROTOCOL_SCHEMA = "dreamteam.alternatives-experiment/v5"
PROTOCOL_ID = "v13i-alt-1"
RUN_SCHEMA = "dreamteam.alternatives-run/v5"
FAILURE_SCHEMA = "dreamteam.run-failure/v2"
MARKER_SCHEMA = "dreamteam.finalization-marker/v1"
HEALTH_SCHEMA = "dreamteam.health-observation/v1"
MARKER = "FINALIZING"
LIFECYCLE_FILE = "analysis/alternatives/lifecycle_v1.json"


# --- protocol (V0.13h's, bound to this driver and the lifecycle contract) ---------

def draft_protocol(rows, contexts_sha):
    proto, sched = H.draft_protocol(rows, contexts_sha)
    proto.update(schema=PROTOCOL_SCHEMA, id=PROTOCOL_ID, driver=DRIVER,
                 lifecycle={"id": LS.LIFECYCLE_ID, "file": LIFECYCLE_FILE, "sha256": LS.spec_sha256()})
    return proto, sched


# --- the health probe (dreamteam.health-observation/v1) -----------------------------

def _class_category(x):
    if isinstance(x, http.client.RemoteDisconnected):
        return "remote_disconnected"
    if isinstance(x, ConnectionRefusedError):
        return "connection_refused"
    if isinstance(x, (ConnectionResetError, ConnectionAbortedError, BrokenPipeError)):
        return "connection_reset"
    if isinstance(x, http.client.IncompleteRead):
        return "incomplete_read"
    if isinstance(x, (TimeoutError, socket.timeout)):
        return "timeout"
    if isinstance(x, http.client.HTTPException):
        return "http_protocol"
    return None


def read_failure(x):
    """(category, exception record) for a transport failure, by class; None
    for an exception that is not a read failure (it propagates)."""
    if isinstance(x, urllib.error.URLError) and not isinstance(x, urllib.error.HTTPError):
        reason = x.reason if isinstance(x.reason, BaseException) else None
        return (_class_category(reason) if reason is not None else None) or "url_error", \
            {"type": type(x).__name__, "reasonType": type(reason).__name__ if reason is not None else None}
    c = _class_category(x)
    if c is None and isinstance(x, OSError):
        c = "os_error"
    return (c, {"type": type(x).__name__, "reasonType": None}) if c else None


def _body(raw):
    return None if raw is None else {"bytes": len(raw), "sha256": hashlib.sha256(raw).hexdigest(),
                                     "base64": base64.b64encode(raw).decode("ascii")}


def probe_health(base, timeout=20, utc=RR.utc, urlopen=urllib.request.urlopen, spec=None):
    """GET /health once. The observation records what was received or how
    the read failed; it never substitutes a response."""
    spec = spec or LS.load()
    limit = spec["health"]["maxBodyBytes"]
    root = base.rstrip("/")
    if root.endswith("/v1"):
        root = root[:-3]
    obs = {"schema": HEALTH_SCHEMA, "request": dict(spec["health"]["request"]), "started": utc(), "ended": None,
           "outcome": None, "category": None, "httpStatus": None, "exception": None, "body": None,
           "verificationStatus": "UNVERIFIED"}
    try:
        with urlopen(root + "/health", timeout=timeout) as r:
            status = r.status
            declared = getattr(r, "length", None)
            # A declared length is read in full so that truncation raises
            # IncompleteRead; otherwise the read is bounded.
            raw = r.read() if isinstance(declared, int) and declared <= limit else r.read(limit + 1)
    except urllib.error.HTTPError as x:
        try:
            raw = x.read(limit + 1)
        except Exception:  # noqa: BLE001  (the error body is optional evidence)
            raw = None
        obs.update(category="http_status", httpStatus=x.code, exception={"type": type(x).__name__, "reasonType": None},
                   body=_body(raw))
    except Exception as x:  # noqa: BLE001  (classified by class below, or propagated)
        failure = read_failure(x)
        if failure is None:
            raise
        obs.update(category=failure[0], exception=failure[1])
    else:
        obs.update(httpStatus=status, body=_body(raw), category=LS.health_body_category(raw, limit))
    obs["outcome"] = spec["health"]["bodyCategories"].get(obs["category"], "READ_FAILED")
    obs["ended"] = utc()
    return obs


def server_state(base, utc=RR.utc):
    """What the public endpoints say (no admin call); /health is a structured observation."""
    api = RR.get(base, "/openapi.json")
    models = RR.get(base, "/v1/models")
    return {"time": utc(), "health": probe_health(base, utc=utc),
            "serverVersion": (api.get("info") or {}).get("version"), "serverTitle": (api.get("info") or {}).get("title"),
            "openapiSha256": sha256(json.dumps(api, sort_keys=True)) if "error" not in api else None,
            "models": [{"id": m.get("id"), "ownedBy": m.get("owned_by"), "maxModelLen": m.get("max_model_len")}
                       for m in models.get("data") or []]}


def observation_problem(obs, spec):
    """The recorder's own refusal of an inconsistent observation before it
    can justify continuing (the verifier re-checks independently)."""
    h = spec["health"]
    if not isinstance(obs, dict) or obs.get("schema") != HEALTH_SCHEMA or obs.get("request") != h["request"]:
        return "not a health observation"
    cat, outcome, body, exc = obs.get("category"), obs.get("outcome"), obs.get("body"), obs.get("exception")
    if cat in h["bodyCategories"]:
        if exc is not None or not isinstance(body, dict):
            return "body category without a body, or with an exception"
        try:
            raw = base64.b64decode(body["base64"], validate=True)
        except Exception:  # noqa: BLE001
            return "undecodable body"
        if LS.health_body_category(raw, h["maxBodyBytes"]) != cat or h["bodyCategories"][cat] != outcome:
            return "body does not classify as recorded"
    elif cat in h["readFailureCategories"]:
        if outcome != "READ_FAILED" or not isinstance(exc, dict):
            return "read failure without an exception"
        if cat != "http_status" and body is not None:
            return "read failure with a body"
    else:
        return "unknown health category"
    return None


# --- recording errors ---------------------------------------------------------

class RecordingError(Exception):
    """The archive could not be finalized: it is unfinalized, keeps its
    FINALIZING marker, or was never created. It is INVALID evidence."""

    def __init__(self, message, cause=None):
        super().__init__(f"{message}: {type(cause).__name__}: {cause}" if cause is not None else message)
        self.cause = cause


class _FailClosed(Exception):
    def __init__(self, what, cause):
        super().__init__(f"{what}: {type(cause).__name__}: {cause}")
        self.what, self.cause = what, cause


class _NoArchive(Exception):
    def __init__(self, cause):
        super().__init__(str(cause))
        self.cause = cause


class _SignalGuard:
    """SIGINT deferred except inside interruptible regions (contract
    interrupts.operatorSignal). Only the main thread installs it."""

    def __init__(self):
        self.pending, self.depth, self.prev, self.active = False, 0, None, False
        # Frames of the recorder itself inside a region: a signal handled there
        # arrives before the external call starts or after it returned, so it
        # is deferred rather than allowed to discard the call's result.
        self.defer_codes = set()

    def install(self):
        if threading.current_thread() is threading.main_thread():
            self.prev = signal.signal(signal.SIGINT, self._handler)
            self.active = True

    def restore(self):
        if self.active:
            signal.signal(signal.SIGINT, self.prev)
            self.active = False

    def _handler(self, signum, frame):
        if self.depth > 0 and (frame is None or frame.f_code not in self.defer_codes):
            raise KeyboardInterrupt("operator interrupt")
        self.pending = True

    def poll(self):
        if self.pending:
            self.pending = False
            raise KeyboardInterrupt("deferred operator interrupt")

    def interruptible(self):
        return _Region(self)


class _Region:
    def __init__(self, guard):
        self.guard = guard

    def __enter__(self):
        self.guard.poll()
        self.guard.depth += 1

    def __exit__(self, *exc):
        self.guard.depth -= 1
        return False


# --- the recorder ----------------------------------------------------------------

class Recorder:
    """One run, recorded through the lifecycle contract. hook(boundary,
    phase, context) is called at every declared fault-injection point; it is
    None in ordinary runs."""

    def __init__(self, proto, proto_sha, sched, ctxs, opener, transports, out, authorization, state_fn=None,
                 now=SB.time.monotonic, utc=RR.utc, hook=None, run_id=None, provenance=None,
                 draw_runner=None, signal_guard=True):
        self.spec = LS.load()
        self.table = LS.transition_table(self.spec)
        self.points = {(b, p) for b, v in self.spec["faultInjection"]["boundaries"].items()
                       for p in self.spec["faultInjection"]["operations"][v["operation"]]["phases"]}
        self.proto, self.proto_sha, self.sched, self.ctxs = proto, proto_sha, sched, ctxs
        self.opener, self.transports, self.authorization = opener, transports, authorization
        self.state_fn, self.now, self.utc, self.hook = state_fn, now, utc, hook
        self.live = os.path.join(out, "live")
        self.run_id = run_id or str(uuid.uuid4())
        self.provenance = provenance or {"timestamps": "SYSTEM_UTC", "runId": "UUID4"}
        self.draw_runner = draw_runner or H.run_draw
        self.guard = _SignalGuard() if signal_guard else None
        if self.guard is not None:
            self.guard.defer_codes = {f.__code__ for f in (Recorder._execute, Recorder._health_read,
                                                          Recorder._cleanup_and_finalize, Recorder._interruptible,
                                                          _Region.__enter__, _Region.__exit__)}
        self.state, self.doc, self.rows, self.pending, self.aborted = None, None, [], None, None
        self.stopping, self.closed, self.limits, self.signature = False, False, None, None
        self.reading = None  # the health check index while a read is in progress
        self.check_in_state = None  # the health check made in the current PREFLIGHT/HEALTH_CHECK visit
        self.draw_entered = False
        self.before_saved = False
        self.deferred_after_stop = False

    # -- hooks and persistence ---------------------------------------------------

    def _hook(self, boundary, phase, **ctx):
        if (boundary, phase) not in self.points:
            raise LS.SpecError(f"undeclared injection point {boundary}.{phase}")
        if self.hook is not None:
            durable = self.pending is not None and bool(self.rows) and self.rows[-1] is self.pending
            self.hook(boundary, phase, dict(ctx, state=self.state, stopping=self.stopping, committed=len(self.rows),
                                            pending=self.pending is not None, pendingDurable=durable))

    def _dir_fsync(self, boundary, ctx):
        self._hook(boundary, "dir_fsync", **ctx)
        if os.name != "nt":
            dfd = os.open(self.live, os.O_RDONLY)
            try:
                try:
                    os.fsync(dfd)
                except OSError as x:
                    if x.errno not in (errno.EINVAL, errno.ENOTSUP):
                        raise
            finally:
                os.close(dfd)

    def _persist(self, boundary, name, value, on_durable=None, **ctx):
        """Same-directory atomic replacement with file and directory sync.
        Any failure inside the operation is a persistence failure."""
        data = (json.dumps(value, sort_keys=True, ensure_ascii=False, indent=1) + "\n").encode("utf-8")
        path = os.path.join(self.live, name)
        self._hook(boundary, "before", **ctx)
        tmp = None
        try:
            fd, tmp = tempfile.mkstemp(prefix=".metadata-", dir=self.live)
            with os.fdopen(fd, "wb") as f:
                f.write(data)
                f.flush()
                self._hook(boundary, "write", **ctx)
                self._hook(boundary, "file_fsync", **ctx)
                os.fsync(f.fileno())
            self._hook(boundary, "replace", **ctx)
            os.replace(tmp, path)
            tmp = None
            self._hook(boundary, "replaced", **ctx)
            self._dir_fsync(boundary, ctx)
        except BaseException as x:
            raise _FailClosed("persisting " + name, x)
        finally:
            if tmp is not None and os.path.exists(tmp):
                os.unlink(tmp)
        if on_durable is not None:
            on_durable()
        self._hook(boundary, "after", **ctx)

    def _read_ledger(self):
        try:
            with open(os.path.join(self.live, "draws.json"), encoding="utf-8") as f:
                return json.load(f)
        except BaseException as x:
            raise _FailClosed("reading the committed ledger", x)

    # -- transitions -------------------------------------------------------------

    def _counters(self):
        completed = sum(r["status"] == "completed" for r in self.rows)
        interrupted = next((r["seq"] for r in self.rows if r["status"] == "interrupted"), None)
        aborted = self.aborted["seq"] if self.aborted is not None else None
        calls = sum(len(r["calls"]) for r in self.rows) + (len(self.aborted["calls"]) if self.aborted else 0)
        return {"drawsScheduled": len(self.sched), "drawsCompleted": completed, "drawInterrupted": interrupted,
                "drawAborted": aborted, "drawsNotRun": len(self.sched) - completed - (interrupted is not None)
                - (aborted is not None), "modelCalls": calls, "healthChecks": len(self.doc["healthChecks"]) if self.doc else 0}

    def _entry(self, event, to, ts, detail):
        return dict({"n": len(self.doc["journal"]) + 1 if self.doc else 1, "event": event, "from": self.state,
                     "to": to, "timestamp": ts}, **detail)

    def _transition(self, event, ts=None, extra=None, on_durable=None, **detail):
        if not self.stopping and self.guard is not None:
            self.guard.poll()
        key = (self.state, event)
        if key not in self.table:
            raise _FailClosed("lifecycle", LS.SpecError(f"illegal transition {key}"))
        to = self.table[key]
        ts = ts or self.utc()
        doc = copy.deepcopy(self.doc)
        doc["journal"].append(self._entry(event, to, ts, detail))
        doc["state"] = to
        doc.update(extra or {})

        def durable():
            self.doc, self.state = doc, to
            self.check_in_state = None
            if on_durable:
                on_durable()
            self.doc["counters"] = self._counters()
        doc["counters"] = self._counters()
        self._persist("persist:manifest", "run.json", doc, on_durable=durable, event=event)

    # -- the run -----------------------------------------------------------------

    def run(self):
        if os.path.exists(self.live):
            raise ExperimentError(f"{self.live} exists: a run is never resumed, extended or overwritten")
        if not self.authorization or not self.authorization.strip():
            raise ExperimentError("no authorization")
        os.makedirs(self.live)
        if self.guard is not None:
            self.guard.install()
        try:
            return self._run()
        finally:
            if self.guard is not None:
                if self.guard.pending:
                    self.deferred_after_stop = True
                self.guard.restore()

    def _create(self):
        self._persist("persist:ledger_init", "draws.json", [])
        ts = self.utc()
        entry = {"n": 1, "event": "create", "from": None, "to": "CREATED", "timestamp": ts}
        if self.table.get((None, "create")) != "CREATED":
            raise LS.SpecError("create")
        doc = {"schema": RUN_SCHEMA, "driver": DRIVER,
               "lifecycle": {"id": LS.LIFECYCLE_ID, "sha256": LS.spec_sha256()},
               "protocol": self.proto["id"], "protocolSha256": self.proto_sha,
               "scheduleSha256": sha256(canon(self.sched)), "runId": self.run_id,
               "authorization": self.authorization, "provenance": dict(self.provenance),
               "serverObserved": self.state_fn is not None, "started": ts, "ended": None,
               "state": "CREATED", "status": None, "journal": [entry], "counters": None, "healthChecks": [],
               "stop": None, "failureEvent": None, "cleanup": {"status": "NOT_STARTED", "failure": None, "timestamp": None},
               "finalization": None, "verificationStatus": "UNVERIFIED", "resume": "never"}
        self.doc = doc
        doc["counters"] = self._counters()

        def durable():
            self.state = "CREATED"
        self._persist("persist:manifest", "run.json", doc, on_durable=durable, event="create")

    def _run(self):
        try:
            try:
                self._create()
            except BaseException as x:
                if self.state is None:
                    raise _NoArchive(x.cause if isinstance(x, _FailClosed) else x)
                raise
            self._main()
        except _NoArchive as x:
            raise RecordingError("the archive was not created", x.cause) from x.cause
        except _FailClosed as x:
            self._abandon(x)
        except BaseException as x:
            if self.stopping:
                self._abandon(_FailClosed("stop handling", x))
            try:
                self._handle(x)
            except _FailClosed as y:
                self._abandon(y)
            except BaseException as y:
                self._abandon(_FailClosed("stop handling", y))
        try:
            if self.state != "CLEANUP":
                raise _FailClosed("lifecycle", LS.SpecError("finalization outside CLEANUP"))
            return self._cleanup_and_finalize()
        except _FailClosed as x:
            self._abandon(x)
        except BaseException as x:
            self._abandon(_FailClosed("finalization", x))

    def _abandon(self, failure):
        if not self.closed:
            self.closed = True
            try:
                self.opener.close()
            except BaseException:  # noqa: BLE001  (the recording failure stays primary)
                pass
        raise RecordingError("recording failed closed; the archive is not finalized", failure.cause) from failure.cause

    def _main(self):
        self._transition("begin_preflight")
        stop = self.proto["stop"]
        self.limits = H.Limits(self.proto["maxModelCallsPerDraw"], self.proto["maxModelCalls"], stop["wallClockSeconds"],
                               self.proto["request"]["timeoutSeconds"], max_fails=stop["consecutiveTransportFailures"],
                               now=self.now)
        if self.state_fn is not None:
            st, obs = self._health_read(None)
            if st is not None:
                self._save_before(st)
            if obs is None:
                return self._stop("driver", failure={"type": "ExperimentError", "message": "invalid health observation"},
                                  healthCheck=len(self.doc["healthChecks"]) - 1)
            if obs["outcome"] != "HEALTHY":
                return self._stop("health", healthCheck=len(self.doc["healthChecks"]) - 1,
                                  request=dict(self.spec["health"]["request"]), message="the server did not answer healthy")
        self._transition("preflight_passed")
        prev = None
        for i, entry in enumerate(self.sched):
            if self.state_fn is not None and (entry["model"], entry["draw"]) != prev:
                prev = (entry["model"], entry["draw"])
                k = len(self.doc["healthChecks"])
                self._transition("begin_health_check", healthCheck=k)
                st, obs = self._health_read(entry["seq"])
                if obs is None:
                    return self._stop("driver", failure={"type": "ExperimentError", "message": "invalid health observation"},
                                      healthCheck=k)
                if obs["outcome"] != "HEALTHY":
                    return self._stop("health", healthCheck=k, request=dict(self.spec["health"]["request"]),
                                      message="the server did not answer healthy")
                if X.server_signature(st) != self.signature:
                    return self._stop("server_change", healthCheck=k, beforeSignature=self.signature,
                                      message="server changed")
                self._transition("health_passed", healthCheck=k)
            self._hook("draw_guard", "before", draw=entry["seq"])
            try:
                self._hook("draw_guard", "during", draw=entry["seq"])
                self.limits.before_draw()
            except Halt as x:
                return self._stop(x.code, message=x.message, evidence=getattr(x, "evidence", None))
            self._hook("draw_guard", "after", draw=entry["seq"])
            self._transition("begin_draw", on_durable=self._new_draw, draw=entry["seq"])
            d = self._execute(entry)
            self._commit(d)
            if d["status"] != "completed":
                halt = d["halt"]
                return self._stop(halt["code"], message=halt["message"], evidence=halt.get("evidence"))
            self._transition("schedule_done" if i == len(self.sched) - 1 else "continue")

    def _save_before(self, st):
        def durable():
            self.before_saved = True
        self.signature = X.server_signature(st)
        self._persist("persist:server_before", "server-before.json", st, on_durable=durable)

    def _health_read(self, before_draw):
        """Append the health check, read once, record what was received.
        Returns (server state or None, valid observation or None)."""
        if self.guard is not None:
            self.guard.poll()
        k = len(self.doc["healthChecks"])
        self._hook("health_read", "before", healthCheck=k)
        if self.guard is not None:
            self.guard.poll()  # an interrupt before the read is not an attempted read
        check = {"index": k, "state": self.state, "beforeDraw": before_draw, "serverState": None, "rejected": None}
        self.doc["healthChecks"].append(check)
        self.check_in_state, self.reading = k, k
        with self._interruptible():
            self._hook("health_read", "during", healthCheck=k)
            st = self.state_fn()
        self.reading = None
        obs = st.get("health") if isinstance(st, dict) else None
        if observation_problem(obs, self.spec) is not None:
            check["rejected"] = {"returned": st}
            st, obs = None, None
        else:
            check["serverState"] = st
        self._hook("health_read", "after", healthCheck=k)
        return st, obs

    def _interruptible(self):
        if self.guard is None:
            class Null:
                def __enter__(self):
                    pass

                def __exit__(self, *a):
                    return False
            return Null()
        return self.guard.interruptible()

    def _new_draw(self):
        self.pending, self.draw_entered = None, False

    def _execute(self, entry):
        self._hook("draw_execute", "before", draw=entry["seq"])
        c = self.ctxs.by_id[entry["context"]]
        settings, mem = self.ctxs.settings[entry["model"]], self.ctxs.memory(c, entry["model"])
        transport = self.transports[entry["model"]]

        def opener(ctx, arm):
            self._hook("draw_execute", "engine_open", draw=entry["seq"])
            return self.opener(ctx, arm)

        def hooked(payload):
            self._hook("draw_execute", "model_call", draw=entry["seq"])
            return transport(payload)
        with self._interruptible():
            self.draw_entered = True
            self.pending = self.draw_runner(entry, c, settings, opener, mem, hooked, self.limits, utc=self.utc, now=self.now)
        self._hook("draw_execute", "after", draw=entry["seq"])
        return self.pending

    def _commit(self, d, reconciled=False):
        rows = self.rows + [d]

        def durable():
            self.rows = rows
        self._persist("persist:ledger", "draws.json", rows, on_durable=durable, draw=d["seq"])
        self._commit_transition(d, reconciled)

    def _commit_transition(self, d, reconciled):
        def durable():
            self.pending = None
        self._transition("commit_draw", on_durable=durable, draw=d["seq"], status=d["status"],
                         rowSha256=sha256(canon(d)), modelCalls=len(d["calls"]), reconciled=reconciled)

    # -- stops ---------------------------------------------------------------------

    def _next_draw(self):
        if self.aborted is not None:
            return {k: self.aborted[k] for k in ("seq", "model", "context", "arm", "draw")}
        if self.rows and self.rows[-1]["status"] == "interrupted":
            return {k: self.rows[-1][k] for k in ("seq", "model", "context", "arm", "draw")}
        return self.sched[len(self.rows)] if len(self.rows) < len(self.sched) else None

    def _stop_object(self, code, state, message=None, failure=None, evidence=None, request=None, healthCheck=None,
                     beforeSignature=None):
        return {"code": code, "state": state, "message": message if message is not None else
                (f"{failure['type']}: {failure['message']}" if failure else code), "failure": failure,
                "evidence": evidence, "request": request, "healthCheck": healthCheck, "beforeSignature": beforeSignature}

    def _failure_event(self, stop, ts):
        counters = self._counters()
        return {"schema": FAILURE_SCHEMA, "runId": self.run_id, "protocolSha256": self.proto_sha,
                "scheduleSha256": self.doc["scheduleSha256"], "state": stop["state"], "stop": stop,
                "nextDraw": self._next_draw(), "drawsCompleted": counters["drawsCompleted"],
                "modelCalls": counters["modelCalls"], "timestamp": ts, "verificationStatus": "UNVERIFIED"}

    def _stop(self, code, **fields):
        self.stopping = True
        ts = self.utc()
        stop = self._stop_object(code, self.state, **fields)
        event = self._failure_event(stop, ts)
        self._transition("stop", ts=ts, extra={"stop": stop, "failureEvent": event}, code=code)

    def _handle(self, x):
        """A driver/operator exception outside persistence: reconcile the
        committed ledger, keep a returned draw, record the stop."""
        self.stopping = True
        code = "operator" if isinstance(x, KeyboardInterrupt) else "driver"
        failure = {"type": type(x).__name__, "message": str(x)}
        disk = self._read_ledger()
        if canon(disk) != canon(self.rows) and not (self.pending is not None and canon(disk) == canon(self.rows + [self.pending])):
            raise _FailClosed("reconciliation", ExperimentError("the committed ledger disagrees with the recorder"))
        if self.state == "DRAW_ACTIVE":
            if self.pending is not None:
                if canon(disk) == canon(self.rows) and (not self.rows or self.rows[-1] is not self.pending):
                    self._commit(self.pending, reconciled=True)
                else:
                    self.rows = disk
                    self._commit_transition(self.pending, reconciled=True)
            else:
                entry = self.sched[len(self.rows)]
                aborted = getattr(x, "partialDraw", None)
                if aborted is None:
                    stage = "draw_unrecorded" if self.draw_entered else "draw_setup"
                    aborted = {"schema": H.DRAW_SCHEMA, **entry, "status": "aborted", "halt": None, "calls": [],
                               "engine": [], "batches": None, "record": None, "initialDigests": None, "digestsAfter": None,
                               "submittedRepresentation": None, "submittedRepresentationSha256": None,
                               "engineInitialization": "UNKNOWN" if self.draw_entered else "NOT_STARTED",
                               "failureStage": stage, "modelCallsCompleted": 0, "verificationStatus": "UNVERIFIED",
                               "failure": failure}
                aborted = copy.deepcopy(aborted)
                aborted["stopReason"] = code

                def durable():
                    self.aborted = aborted
                self._persist("persist:aborted", "aborted-draw.json", aborted, on_durable=durable, draw=entry["seq"])
                return self._stop(code, failure=aborted["failure"])
        checks = self.doc["healthChecks"]
        if checks and checks[0]["state"] == "PREFLIGHT" and checks[0]["serverState"] is not None and not self.before_saved:
            self._save_before(checks[0]["serverState"])  # the initial read returned; keep its evidence
        if self.state == "DRAW_RECORDED" and self.rows[-1]["status"] != "completed":
            halt = self.rows[-1]["halt"]  # the halt preceded this exception and stays the stop
            return self._stop(halt["code"], message=halt["message"], evidence=halt.get("evidence"))
        request = dict(self.spec["health"]["request"]) if self.reading is not None else None
        hc = self.check_in_state if self.state in ("PREFLIGHT", "HEALTH_CHECK") else None
        return self._stop(code, failure=failure, request=request, healthCheck=hc)

    # -- cleanup and finalization ---------------------------------------------------

    def _cleanup_and_finalize(self):
        self.stopping = True
        self._hook("session_cleanup", "before")
        failure, cleanup_exc = None, None
        try:
            with self._interruptible():
                self.closed = True
                self.opener.close()
                self._hook("session_cleanup", "during")
        except BaseException as x:  # noqa: BLE001  (recorded separately from any primary stop)
            failure, cleanup_exc = {"type": type(x).__name__, "message": str(x)}, x
        ts = self.utc()
        self.doc["cleanup"] = {"status": "FAILED" if failure else "SUCCEEDED", "failure": failure, "timestamp": ts}
        if failure is not None and self.doc["stop"] is None:
            code = "operator" if isinstance(cleanup_exc, KeyboardInterrupt) else "driver"
            stop = self._stop_object(code, "CLEANUP", failure=failure)
            self.doc["stop"], self.doc["failureEvent"] = stop, self._failure_event(stop, ts)
        self._hook("session_cleanup", "after")
        doc = copy.deepcopy(self.doc)
        cleanup_event = "cleanup_failed" if failure else "cleanup_succeeded"
        complete = (doc["stop"] is None and failure is None and len(self.rows) == len(self.sched)
                    and all(r["status"] == "completed" for r in self.rows))
        final_event = "finalize_complete" if complete else "finalize_partial"
        mid = self.table.get((self.state, cleanup_event))
        terminal = self.table.get((mid, final_event))
        if mid is None or terminal is None:
            raise _FailClosed("lifecycle", LS.SpecError(f"illegal finalization from {self.state}"))
        end = self.utc()
        doc["journal"].append({"n": len(doc["journal"]) + 1, "event": cleanup_event, "from": self.state, "to": mid,
                               "timestamp": ts})
        doc["journal"].append({"n": len(doc["journal"]) + 1, "event": final_event, "from": mid, "to": terminal,
                               "timestamp": end})
        doc.update(state=terminal, status=terminal, ended=end)
        doc["counters"] = self._counters()
        marker = {"schema": MARKER_SCHEMA, "runId": self.run_id, "intendedState": terminal, "timestamp": end}
        self._persist("persist:marker", MARKER, marker)
        if doc["failureEvent"] is not None:
            self._persist("persist:failure_event", "failure-event.json", doc["failureEvent"])
        artifacts = {}
        for name in self.spec["allowedArchiveFiles"]:
            if name == "run.json":
                continue
            p = os.path.join(self.live, name)
            if os.path.exists(p):
                with open(p, "rb") as f:
                    artifacts[name] = hashlib.sha256(f.read()).hexdigest()
            else:
                artifacts[name] = None
        doc["finalization"] = {"timestamp": end, "artifacts": artifacts}
        self._persist("persist:final_manifest", "run.json", doc)
        self._remove_marker()
        self.doc, self.state = doc, terminal
        return doc

    def _remove_marker(self):
        path = os.path.join(self.live, MARKER)
        self._hook("unlink:marker", "before")
        try:
            self._hook("unlink:marker", "unlink")
            os.unlink(path)
            self._hook("unlink:marker", "unlinked")
            self._dir_fsync("unlink:marker", {})
        except BaseException as x:
            if not os.path.exists(path):
                try:  # fail closed: the archive must not look finalized
                    with open(path, "w", encoding="utf-8") as f:
                        json.dump({"schema": MARKER_SCHEMA, "runId": self.run_id, "intendedState": None,
                                   "status": "FINALIZATION_FAILED"}, f)
                        f.flush()
                        os.fsync(f.fileno())
                except BaseException:  # noqa: BLE001  (nothing further can be proven)
                    pass
            raise _FailClosed("removing the finalization marker", x)


def run_schedule(proto, proto_sha, sched, ctxs, opener, transports, out, authorization, **kw):
    """Record one run; returns the finalized manifest (COMPLETE or PARTIAL)
    or raises RecordingError (the archive is INVALID)."""
    return Recorder(proto, proto_sha, sched, ctxs, opener, transports, out, authorization, **kw).run()


# --- commands ---------------------------------------------------------------------

def cmd_protocol(a):
    rows, csha = H.load_contexts(a.contexts)
    proto, sched = draft_protocol(rows, csha)
    write_json(a.out, proto)
    print(f"protocol {proto['id']}: {proto['draws']} draws, at most {proto['maxModelCalls']} model calls")


def cmd_schedule(a):
    proto, raw = json.load(open(a.protocol, encoding="utf-8")), open(a.protocol, "rb").read()
    rows, csha = H.load_contexts(a.contexts)
    if csha != proto["contextsFileSha256"]:
        raise SystemExit("the contexts file is not the protocol's")
    again, sched = draft_protocol(rows, csha)
    if {k: v for k, v in again.items() if k != "status"} != {k: v for k, v in proto.items() if k not in ("status", "lock")}:
        raise SystemExit("the protocol is not the one the committed inputs make")
    doc = {"schema": H.SCHEDULE_SCHEMA, "protocol": proto["id"], "protocolSha256": sha256(raw),
           "orderRule": H.ORDER_RULE, "draws": len(sched), "schedule": sched}
    write_json(a.out, doc)
    print(f"schedule: {len(sched)} draws")


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("protocol")
    p.add_argument("--contexts", required=True)
    p.add_argument("--out", required=True)
    p = sub.add_parser("schedule")
    p.add_argument("--protocol", required=True)
    p.add_argument("--contexts", required=True)
    p.add_argument("--out", required=True)
    p = sub.add_parser("check")
    p.add_argument("--binary", required=True)
    p.add_argument("--protocol", required=True)
    p.add_argument("--contexts", required=True)
    p.add_argument("--replays")
    p = sub.add_parser("run")
    p.add_argument("--protocol", required=True)
    p.add_argument("--authorization", default="")
    a = ap.parse_args(argv)
    if a.cmd == "protocol":
        cmd_protocol(a)
    elif a.cmd == "schedule":
        cmd_schedule(a)
    elif a.cmd == "check":
        H.cmd_check(a)  # unchanged per-context engine check; no model is called
    elif a.cmd == "run":
        proto = json.load(open(a.protocol, encoding="utf-8"))
        if proto.get("status") != H.LOCKED:
            raise SystemExit(f"the protocol's status is {proto.get('status')!r}: only a locked protocol, with the "
                             "owner's authorization, can be run")
        if not a.authorization.strip():
            raise SystemExit("no --authorization: the lock does not authorize a run; the owner does")
        raise SystemExit("run: the live run is not implemented before the owner approves the protocol")


if __name__ == "__main__":
    main()
