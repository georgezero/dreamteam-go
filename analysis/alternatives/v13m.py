#!/usr/bin/env python3
"""The V0.13m-lite experiment driver (dreamteam-v13m-driver/9): the V0.13h draw,
context, replay and protocol machinery unchanged, with the enclosing run
recorded through the safety contract dreamteam.run-lifecycle/v4
(analysis/alternatives/bindings_v2.json, docs/v13m/LIFECYCLE_CONTRACT.md):
safe-point interrupts, a durable call journal and a single completion seal.
V0.13l added durable evidence ownership: a returned model response is owned
by the recorder and recorded before the workflow consumes it
(dreamteam.response-handoff/v1), and a completed draw is owned from the
statement that receives it until the ledger commit is durable.

V0.13m-lite (dreamteam.acceptance-standard/v1, owner-approved fail-closed
persistence policy, docs/v13m/ACCEPTANCE_CONTRACT.md): when recovery of a
completed draw itself fails, or failure-evidence construction itself fails,
the run is stopped at once and never sealed (INVALID); no nested recovery,
retry, resumption or replacement draw is attempted, and possible evidence
loss is disclosed in the retained failure report
(dreamteam.failed-persistence-retention/v3, field evidenceLoss) and in the
structured recording error. Model behaviour and every model-visible byte are
unchanged.

    python3 -I -B analysis/alternatives/v13m.py protocol --contexts DIR --out FILE
    python3 -I -B analysis/alternatives/v13m.py schedule --contexts DIR --protocol FILE --out FILE
    python3 -I -B analysis/alternatives/v13m.py check --binary ./dt --contexts DIR --protocol FILE --replays DIR
    python3 -I -B analysis/alternatives/v13m.py run ... (refused: the protocol is not locked)

Offline verification and analysis are in v13m_verify.py, which does not
import this module. No command contacts a model.
"""
import argparse
import builtins
import http.client
import socket
import ssl
import base64
import copy
import errno
import hashlib
import json
import os
import signal
import subprocess
import sys
import tempfile
import threading
import urllib.error
import urllib.request
import uuid

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import lifecycle_v4_spec as LS  # noqa: E402
import v13h as H  # noqa: E402  (V0.13h contexts, draws, replay and protocol, unchanged)

SB, X, RR = H.SB, H.X, H.SB.RR
sha256, canon, write_json, Halt = H.sha256, H.canon, H.write_json, H.Halt
ExperimentError = H.ExperimentError
QWEN, GEMMA = H.QWEN, H.GEMMA

DRIVER = "dreamteam-v13m-driver/9"
PROTOCOL_SCHEMA = "dreamteam.alternatives-experiment/v9"
PROTOCOL_ID = "v13m-alt-1"
RUN_SCHEMA = "dreamteam.alternatives-run/v9"
FAILURE_SCHEMA = "dreamteam.run-failure/v3"
RETENTION_SCHEMA = "dreamteam.failed-persistence-retention/v3"
HEALTH_SCHEMA = "dreamteam.health-observation/v2"
CALLS_SCHEMA = "dreamteam.call-journal/v1"
SEAL_SCHEMA = "dreamteam.completion-seal/v1"
FINALIZATION = "dreamteam.run-finalization/v1"
RESULT_SCHEMA = "dreamteam.run-result/v1"
SEAL = "SEAL.json"
CALLS = "calls.jsonl"
LIFECYCLE_FILE = "analysis/alternatives/bindings_v2.json"
SIDE = ("server-before.json", "aborted-draw.json", "failure-event.json")


# --- protocol (V0.13h's, bound to this driver and the safety contract) ------------

def draft_protocol(rows, contexts_sha):
    proto, sched = H.draft_protocol(rows, contexts_sha)
    proto.update(schema=PROTOCOL_SCHEMA, id=PROTOCOL_ID, driver=DRIVER,
                 lifecycle={"id": LS.LIFECYCLE_ID, "file": LIFECYCLE_FILE, "sha256": LS.spec_sha256()})
    return proto, sched


# --- engine sessions isolated from terminal signals --------------------------------

def spawn(binary, args):
    """V0.13h's engine spawn with the same arguments, in its own session
    (process group) so that a terminal SIGINT reaches only the recorder."""
    scratch = tempfile.TemporaryDirectory(prefix="v13m-engine-")
    kw = {"creationflags": subprocess.CREATE_NEW_PROCESS_GROUP} if os.name == "nt" else {"start_new_session": True}
    proc = subprocess.Popen([os.path.abspath(binary), "agent", "--stdio", "-alternatives"] + args,
                            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True,
                            encoding="utf-8", cwd=scratch.name, **kw)

    def ask(r):
        proc.stdin.write(json.dumps(r) + "\n")
        proc.stdin.flush()
        line = proc.stdout.readline()
        if not line:
            raise ExperimentError("the engine closed the session")
        return json.loads(line)

    def close():
        try:
            proc.stdin.close()
        finally:
            proc.wait()
            proc.stdout.close()
            scratch.cleanup()
    return ask, close


class Fresh(H.Fresh):
    """H.Fresh with an isolated engine."""

    def __init__(self, binary, c):  # noqa: D401  (same steps as H.Fresh)
        self.ask, self._close = spawn(binary, ["-scenario", c["scenario"], "-seed", str(c["seed"])])
        try:
            a = self.request("adv", "advance", {"to": c["at"]})
            if not a.get("ok") or a["result"]["clock"]["time"] != c["at"]:
                raise ExperimentError(f"{c['id']}: the engine did not reach {c['at']}")
        except BaseException:
            self.close()
            raise


def open_context(binary, run_dirs, c):
    """H.open_context with an isolated engine."""
    if c["kind"] == "constructed":
        return Fresh(binary, c)
    run, pairs = X.transcript(run_dirs[c["id"]])
    ask, close = spawn(binary, ["-scenario", run["scenario"]["ref"], "-seed", str(run["seed"])])
    ses = X.Session(run, pairs, ask, close)
    try:
        ses.to_point(c["digests"])
    except BaseException:
        ses.close()
        raise
    return ses


class Contexts(H.Contexts):
    """H.Contexts whose engine sessions are isolated from terminal signals."""

    def opener(self, binary):
        return H.Pool(lambda c, arm: open_context(binary, self.run_dirs, c))


# --- the health probe (dreamteam.health-observation/v2) -----------------------------

def qualname(cls):
    return f"{cls.__module__}.{cls.__qualname__}"


_TRUSTED_MODULES = {"builtins": builtins, "http.client": http.client, "socket": socket, "ssl": ssl,
                    "urllib.error": urllib.error}
# Capture actual trusted objects once. Matching text or overloaded class equality grants no trust.
_TRUSTED_TYPES = {name: getattr(_TRUSTED_MODULES[name.rsplit(".", 1)[0]], name.rsplit(".", 1)[1])
                  for f in LS.load()["health"]["readFailureFamilies"].values() for name in f["types"]}


def normalize_exception(x, spec):
    """Classify actual trusted class objects by identity, then serialize the pinned name."""
    fam = LS.rule("B-HEALTH-004")["families"]
    by_type = [(_TRUSTED_TYPES[t], name, t) for name, f in fam.items()
               if name not in ("http_status", "url_error") for t in f["types"]]
    if type(x) is _TRUSTED_TYPES["urllib.error.HTTPError"]:
        return "http_status", {"type": "urllib.error.HTTPError", "reasonType": None, "reasonKind": None}
    if type(x) is _TRUSTED_TYPES["urllib.error.URLError"]:
        r = x.reason
        if isinstance(r, str):
            return "url_error", {"type": "urllib.error.URLError", "reasonType": None, "reasonKind": "text"}
        found = next(((name, t) for cls, name, t in by_type if type(r) is cls), None)
        if found is not None and fam[found[0]]["wrappable"]:
            return found[0], {"type": "urllib.error.URLError", "reasonType": found[1], "reasonKind": "exception"}
        return None
    found = next(((name, t) for cls, name, t in by_type if type(x) is cls), None)
    return (found[0], {"type": found[1], "reasonType": None, "reasonKind": None}) if found else None


def _body(raw):
    return None if raw is None else {"bytes": len(raw), "sha256": hashlib.sha256(raw).hexdigest(),
                                     "base64": base64.b64encode(raw).decode("ascii")}


def probe_health(base, timeout=20, utc=RR.utc, urlopen=urllib.request.urlopen, spec=None):
    """GET /health once. The observation records what was received or how
    the read failed; it never substitutes a response. An exception that is
    not a listed transport failure propagates."""
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
            raw = r.read() if isinstance(declared, int) and declared <= limit else r.read(limit + 1)
    except urllib.error.HTTPError as x:
        try:
            raw = x.read(limit + 1)
        except Exception:  # noqa: BLE001  (the error body is optional evidence)
            raw = None
        fam, ev = normalize_exception(x, spec) or (None, None)
        if fam != "http_status":
            raise
        obs.update(category=fam, httpStatus=x.code, exception=ev, body=_body(raw))
    except Exception as x:  # noqa: BLE001  (classified by exact class below, or propagated)
        failure = normalize_exception(x, spec)
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
    keys = {"schema", "request", "started", "ended", "outcome", "category", "httpStatus", "exception", "body",
            "verificationStatus"}
    if not isinstance(obs, dict) or set(obs) != keys or obs["schema"] != HEALTH_SCHEMA or obs["request"] != h["request"]:
        return "not a health observation"
    if obs["verificationStatus"] != "UNVERIFIED":
        return "the observation claims verification"
    t0, t1 = LS.parse_timestamp(obs["started"]), LS.parse_timestamp(obs["ended"])
    if t0 is None or t1 is None or t1 < t0:
        return "observation timestamps"
    cat, body, exc, status = obs["category"], obs["body"], obs["exception"], obs["httpStatus"]
    raw = None
    if body is not None:
        try:
            raw = base64.b64decode(body["base64"], validate=True)
            if set(body) != {"bytes", "sha256", "base64"} or len(raw) != body["bytes"] \
                    or hashlib.sha256(raw).hexdigest() != body["sha256"]:
                return "body evidence"
        except (TypeError, KeyError, ValueError):
            return "body evidence"
    ok_status = isinstance(status, int) and not isinstance(status, bool)
    if cat in h["bodyCategories"]:
        if exc is not None or raw is None or not (ok_status and 200 <= status < 300):
            return "a response category without a valid 2xx response"
        if LS.health_body_category(raw, h["maxBodyBytes"]) != cat or h["bodyCategories"][cat] != obs["outcome"]:
            return "body does not classify as recorded"
        return None
    fam = h["readFailureFamilies"]
    if cat not in fam:
        return "unknown health category"
    if obs["outcome"] != "READ_FAILED" or not isinstance(exc, dict) or set(exc) != {"type", "reasonType", "reasonKind"}:
        return "read failure without exception evidence"
    if cat == "http_status":
        if exc != {"type": "urllib.error.HTTPError", "reasonType": None, "reasonKind": None} or not ok_status \
                or 200 <= status < 300:
            return "http_status without a non-2xx HTTPError"
        return None
    if status is not None or body is not None:
        return "a read failure with response evidence"
    if cat == "url_error":
        return None if exc == {"type": "urllib.error.URLError", "reasonType": None, "reasonKind": "text"} else \
            "url_error evidence"
    if exc["type"] == "urllib.error.URLError":
        return None if exc["reasonKind"] == "exception" and fam[cat]["wrappable"] and \
            exc["reasonType"] in fam[cat]["types"] else "wrapped exception outside its family"
    return None if exc["reasonKind"] is None and exc["reasonType"] is None and exc["type"] in fam[cat]["types"] else \
        "exception outside its family"


# --- recording errors, interrupts and the filesystem --------------------------------

class RecordingError(Exception):
    """The archive was not sealed: it is INVALID evidence (its durable rows
    and call journal remain for audit)."""

    def __init__(self, message, cause=None, interrupts=None):
        super().__init__(f"{message}: {type(cause).__name__}: {cause}" if cause is not None else message)
        self.cause, self.interrupts = cause, interrupts


class OperatorStop(BaseException):
    """Raised only at the before_dispatch safe point: the draw is aborted
    before any part of the next request exists."""

    def __init__(self, signals):
        super().__init__("operator interrupt processed at safe point before_dispatch")
        self.signals = signals


def _describe(x):
    """{type, message} of an exception, never raising."""
    try:
        return {"type": type(x).__name__, "message": str(x)}
    except BaseException:  # noqa: BLE001
        return {"type": type(x).__name__, "message": None}


class _FailClosed(Exception):
    def __init__(self, what, cause):
        super().__init__(f"{what}: {type(cause).__name__}: {cause}")
        self.what, self.cause = what, cause


class ResponseNotRecorded(BaseException):
    """A model response returned but its outcome record could not be made
    durable. It is never consumed: the draw ends here (a BaseException, so no
    workflow handler converts it into a retry), no replacement call is made,
    and the received outcome is retained as failure evidence."""

    def __init__(self, k, cause):
        super().__init__(f"the response to model call {k} was received but could not be recorded durably; "
                         f"it is retained as failure evidence and not consumed")
        self.k, self.cause = k, cause


class _NoArchive(Exception):
    def __init__(self, cause):
        super().__init__(str(cause))
        self.cause = cause


class SignalLatch:
    """The minimal SIGINT handler: it only counts. Nothing is ever raised
    from it; safe points read the count."""

    def __init__(self):
        self.count, self.seen, self.prev, self.active = 0, 0, None, False

    def _handler(self, signum, frame):
        self.count += 1

    def install(self):
        if threading.current_thread() is not threading.main_thread():
            raise ExperimentError("the recorder must run in the main thread to own SIGINT")
        self.prev = signal.signal(signal.SIGINT, self._handler)
        self.active = True

    def restore(self):
        if self.active:
            signal.signal(signal.SIGINT, self.prev)
            self.active = False

    def take(self):
        c = self.count
        n, self.seen = c - self.seen, c
        return n


class OsFS:
    """The filesystem operations the recorder uses for its archive."""

    def mkstemp(self, directory, prefix):
        return tempfile.mkstemp(prefix=prefix, dir=directory)

    def write(self, fd, data):
        view = memoryview(data)
        while view:
            n = os.write(fd, view)
            view = view[n:]

    def fsync(self, fd):
        os.fsync(fd)

    def close(self, fd):
        os.close(fd)

    def replace(self, src, dst):
        os.replace(src, dst)

    def unlink(self, path):
        os.unlink(path)

    def open_append(self, path):
        return os.open(path, os.O_WRONLY | os.O_APPEND)

    def dir_fsync(self, directory):
        """'SYNCED', or 'UNSUPPORTED' where the platform has no directory sync."""
        if os.name == "nt":
            return "UNSUPPORTED"
        dfd = os.open(directory, os.O_RDONLY)
        try:
            try:
                os.fsync(dfd)
            except OSError as x:
                if x.errno in (errno.EINVAL, errno.ENOTSUP):
                    return "UNSUPPORTED"
                raise
        finally:
            os.close(dfd)
        return "SYNCED"


def encode_json(value):
    return (json.dumps(value, sort_keys=True, ensure_ascii=False, indent=1) + "\n").encode("utf-8")


def encode_line(value):
    return (json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True) + "\n").encode("ascii")


class SafeLimits(H.Limits):
    """V0.13h's limits with the before_dispatch safe point first and the
    durable dispatch intent last, all before PointModel records anything."""

    def __init__(self, recorder, *a, **k):
        super().__init__(*a, **k)
        self.recorder = recorder

    def before_call(self, draw_calls):
        self.recorder._safe_point("before_dispatch")
        super().before_call(draw_calls)
        self.recorder._dispatch_intent(draw_calls + 1)


# --- the recorder ----------------------------------------------------------------

class Recorder:
    """One run, recorded through the safety contract. hook(boundary, phase,
    context) is called at every declared fault-injection point; it is None in
    ordinary runs. fs is the archive filesystem (OsFS unless injected)."""

    def __init__(self, proto, proto_sha, sched, ctxs, opener, transports, out, authorization, state_fn=None,
                 now=SB.time.monotonic, utc=RR.utc, hook=None, run_id=None, provenance=None, draw_runner=None, fs=None):
        self.spec = LS.load()
        self.table = LS.transition_table(self.spec)
        self.points = LS.declared_points(self.spec)
        self.safe_kinds = {k: v["kind"] for k, v in self.spec["safePoints"]["points"].items()}
        self.proto, self.proto_sha, self.sched, self.ctxs = proto, proto_sha, sched, ctxs
        self.opener, self.transports, self.authorization = opener, transports, authorization
        self.state_fn, self.now, self.utc, self.hook = state_fn, now, utc, hook
        self.live = os.path.join(out, "live")
        self.run_id = run_id or str(uuid.uuid4())
        self.provenance = provenance or {"timestamps": "SYSTEM_UTC", "runId": "UUID4"}
        self.draw_runner = draw_runner or H.run_draw
        self.fs = fs or OsFS()
        self.latch = SignalLatch()
        self.state, self.doc, self.rows, self.pending, self.aborted = None, None, [], None, None
        self.stopping, self.closed, self.limits, self.signature = False, False, None, None
        self.reading, self.check_in_state, self.draw_entered, self.before_saved = None, None, False, False
        self.poisoned, self.deferred = None, None
        self.calls_k, self.draw_seq, self.draw_n = 0, None, 0
        self.calls_sha, self.calls_records = hashlib.sha256(), 0
        self.written = {}
        self.events = []
        self.seal = None
        self.handoff = None
        self.failed_candidates = []
        self.cleanup_visits = []
        self.received, self.unrecorded, self.partial = None, [], None
        self.evidence_loss = None

    # -- hooks ---------------------------------------------------------------------

    def _hook(self, boundary, phase, **ctx):
        if (boundary, phase) not in self.points:
            raise LS.SpecError(f"undeclared injection point {boundary}.{phase}")
        if self.hook is not None:
            durable = self.pending is not None and bool(self.rows) and self.rows[-1] is self.pending
            done = bool(self.doc) and any(e["event"] == "schedule_done" for e in self.doc["journal"])
            self.hook(boundary, phase, dict(ctx, state=self.state, stopping=self.stopping, committed=len(self.rows),
                                            pending=self.pending is not None, pendingDurable=durable,
                                            pendingRetained="pending-draw.json" in self.written,
                                            pendingStatus=self.pending["status"] if self.pending is not None else None,
                                            completedRows=sum(r["status"] == "completed" for r in self.rows),
                                            lastStatus=self.rows[-1]["status"] if self.rows else None,
                                            dispatched=self.calls_k, scheduleDone=done,
                                            receivedUnrecorded=len(self.unrecorded) + (self.received is not None),
                                            partialRetained=self.partial is not None,
                                            stopped=bool(self.doc) and self.doc.get("stop") is not None,
                                            deferred=self.deferred is not None))

    # -- persistence -----------------------------------------------------------------

    def _poison(self, what, x):
        if self.poisoned is None:
            self.poisoned = x
        return _FailClosed(what, x)

    def _check_poison(self):
        if self.poisoned is not None:
            raise _FailClosed("a persistence operation of this run failed; the archive is never sealed", self.poisoned)

    def _persist(self, boundary, name, data, on_durable=None, **ctx):
        """Same-directory atomic replacement with file and directory sync.
        Any failure inside the operation poisons the archive."""
        self._check_poison()
        path = os.path.join(self.live, name)
        self._hook(boundary, "before", **ctx)
        fd, tmp = None, None
        try:
            fd, tmp = self.fs.mkstemp(self.live, ".tmp-")
            self.fs.write(fd, data)
            self._hook(boundary, "write", **ctx)
            self._hook(boundary, "file_fsync", **ctx)
            self.fs.fsync(fd)
            self.fs.close(fd)
            fd = None
            self._hook(boundary, "replace", **ctx)
            self.fs.replace(tmp, path)
            tmp = None
            self._hook(boundary, "replaced", **ctx)
            self._hook(boundary, "dir_fsync", **ctx)
            self.fs.dir_fsync(self.live)
        except BaseException as x:
            raise self._poison("persisting " + name, x)
        finally:
            if fd is not None:
                try:
                    os.close(fd)
                except OSError:
                    pass
            if tmp is not None and name in ("draws.json", "pending-draw.json"):
                self.failed_candidates.append(os.path.basename(tmp))
            elif tmp is not None:
                try:
                    self.fs.unlink(tmp)
                except BaseException:  # noqa: BLE001  (the archive is poisoned; a stray file is itself INVALID)
                    pass
        self.written[name] = data
        if on_durable is not None:
            on_durable()
        self._hook(boundary, "after", **ctx)

    def _append(self, rec, before_poisons, **ctx):
        """Append one call-journal record and fsync it. With before_poisons,
        a fault before the write also poisons (an outcome must not go missing)."""
        self._check_poison()
        data = encode_line(rec)
        ctx = dict(ctx, record=rec["record"], k=rec["k"], draw=rec["draw"])
        if before_poisons:
            try:
                self._hook("append:calls", "before", **ctx)
            except BaseException as x:
                raise self._poison("before a call outcome record", x)
        else:
            self._hook("append:calls", "before", **ctx)
        fd = None
        try:
            fd = self.fs.open_append(os.path.join(self.live, CALLS))
            self.fs.write(fd, data)
            self._hook("append:calls", "write", **ctx)
            self._hook("append:calls", "file_fsync", **ctx)
            self.fs.fsync(fd)
            self.fs.close(fd)
            fd = None
        except BaseException as x:
            raise self._poison("appending to " + CALLS, x)
        finally:
            if fd is not None:
                try:
                    os.close(fd)
                except OSError:
                    pass
        self.calls_sha.update(data)
        self.calls_records += 1
        return ctx

    def _window(self, fn):
        """Inside a transport window no recorder exception propagates."""
        try:
            fn()
        except _FailClosed:
            pass  # already poisoned; the next STOP point or persistence fails closed
        except BaseException as x:  # noqa: BLE001  (deferred to the next STOP point)
            if self.deferred is None:
                self.deferred = x

    def _handoff(self, row):
        """B-HANDOFF-001: retain the completed result before any ledger mutation."""
        self.pending = row
        doc = {"schema": self.spec["completedHandoff"]["schema"], "draw": row,
               "rowSha256": sha256(canon(row))}
        self._persist("persist:handoff", "pending-draw.json", encode_json(doc), draw=row["seq"])
        self.handoff = doc

    def _clear_handoff(self):
        self._hook("handoff_clear", "before")
        try:
            self.fs.unlink(os.path.join(self.live, "pending-draw.json"))
            self.fs.dir_fsync(self.live)
        except BaseException as x:
            raise self._poison("clearing committed draw handoff", x)
        self.written.pop("pending-draw.json", None)
        self.handoff = None
        self._hook("handoff_clear", "after")

    def _received_evidence(self):
        """Every received outcome that is not known to be durable in the call journal."""
        out = list(self.unrecorded)
        r = self.received
        if r is not None and all(r is not u for u in out):
            if isinstance(r, tuple):  # owned at receipt, before the outcome record was built
                r = {"k": self.calls_k, "draw": self.draw_seq, "n": self.draw_n, "transportResult": list(r)}
            out.append(r)
        return copy.deepcopy(out)

    def _retain_failure(self, failure):
        """Best effort audit retention, without hooks, poisoning checks or candidate deletion.
        The primary error is never replaced by a secondary error during error handling.
        dreamteam.failed-persistence-retention/v3: the pending row, the partial draw,
        every received-but-unrecorded model outcome, the dispatch count and
        evidenceLoss (null, or the disclosure that evidence may have been lost).
        Fail-closed policy (V0.13m-lite): if building this document itself fails,
        the minimal fallback keeps the objects the recorder already owns (including
        the response currently held in received) by reference, without copying,
        and discloses the secondary failure; it never attempts further recovery."""
        try:
            row = self.pending or (self.handoff or {}).get("draw")
            doc = {"schema": RETENTION_SCHEMA, "stage": failure.what,
                   "error": {"type": type(failure.cause).__name__, "message": str(failure.cause)},
                   "pending": row, "partialDraw": self.partial, "receivedResponses": self._received_evidence(),
                   "dispatched": self.calls_k, "candidates": list(self.failed_candidates),
                   "evidenceLoss": self.evidence_loss}
        except BaseException as secondary:  # noqa: BLE001  (never replace the primary error)
            held = list(self.unrecorded)
            if self.received is not None and all(self.received is not u for u in held):
                held.append(self.received)
            doc = {"schema": RETENTION_SCHEMA, "stage": getattr(failure, "what", None), "error": None, "pending": self.pending,
                   "partialDraw": self.partial, "receivedResponses": held, "dispatched": self.calls_k,
                   "candidates": list(self.failed_candidates),
                   "evidenceLoss": {"possible": True, "kind": "failure-evidence construction failed",
                                    "detail": "the failure report was built from the recorder's owned objects without "
                                              "copying; some retained evidence may be incomplete",
                                    "secondaryError": _describe(secondary), "prior": self.evidence_loss}}
        fd = None
        try:
            try:
                data = encode_json(doc)
            except BaseException as secondary:  # noqa: BLE001  (an owned object that cannot be encoded)
                doc = dict(doc, receivedResponses=[repr(x) for x in doc.get("receivedResponses") or []],
                           evidenceLoss={"possible": True, "kind": "failure-evidence encoding failed",
                                         "detail": "received responses are retained as their text representation",
                                         "secondaryError": _describe(secondary), "prior": doc.get("evidenceLoss")})
                data = encode_json(doc)
            fd, candidate = self.fs.mkstemp(self.live, ".failure-")
            self.fs.write(fd, data)
            self.fs.fsync(fd)
            self.fs.close(fd)
            fd = None
            self.fs.replace(candidate, os.path.join(self.live, "persistence-failure.json"))
            self.fs.dir_fsync(self.live)
        except BaseException:
            pass
        finally:
            if fd is not None:
                try:
                    os.close(fd)
                except OSError:
                    pass
        return doc

    def _draw_result(self, *args, **kwargs):
        """B-HANDOFF-001 / R-HANDOFF-001: the completed result is owned by the
        recorder in the statement that receives it, and the transfer to the
        durable handoff is inside the recovery scope."""
        row = None
        try:
            self.pending = row = self.draw_runner(*args, **kwargs)
            self._handoff(row)
        except BaseException as x:
            try:
                self._own_result(x, row)
            except BaseException as secondary:  # noqa: BLE001
                # Fail-closed policy (V0.13m-lite, DT13L-03): recovery of the draw's
                # result itself failed. The run stops here and is never sealed; no
                # nested recovery, retry, resumption or replacement draw. The original
                # failure and the recovery failure are both disclosed. The archive is
                # poisoned first, so no later step can seal it even if disclosure fails.
                if self.poisoned is None:
                    self.poisoned = secondary
                try:
                    owned = self.pending is not None or row is not None
                    self.evidence_loss = {
                        "possible": not owned, "kind": "completed-draw recovery failed", "draw": self.draw_seq,
                        "detail": ("recovering the owned draw result failed; the owned result is retained" if owned
                                   else "the draw ended with an exception and recovering its result failed; a "
                                   "completed result may have been lost"),
                        "dispatched": self.calls_k, "originalError": _describe(x), "secondaryError": _describe(secondary)}
                finally:
                    raise _FailClosed("completed-draw recovery", secondary) from x
            raise
        return row

    def _own_result(self, x, row):
        """Keep the only copy of a draw that finished before x: the returned
        row, or the result inside run_draw's frame when x escaped run_draw
        after the workflow and final digests completed. The primary error is
        never replaced; a failed handoff here poisons the archive, which then
        retains the result as failure evidence."""
        candidate = row if row is not None else self.pending
        if candidate is None:
            candidate = self._recover_result(x)
        if candidate is None:
            partial = getattr(x, "partialDraw", None)
            if isinstance(partial, dict):
                self.partial = copy.deepcopy(partial)
            return
        self.pending = candidate
        if self.handoff is None or self.handoff.get("draw") is not candidate:
            try:
                self._handoff(candidate)
            except BaseException:  # noqa: BLE001  (poisoned or deferred; the result stays owned in self.pending)
                pass

    def _recover_result(self, x):
        tb = x.__traceback__
        while tb is not None:
            frame = tb.tb_frame
            if frame.f_code is H.run_draw.__code__:
                local = frame.f_locals
                out = local.get("out")
                if isinstance(out, dict) and out.get("status") in ("completed", "interrupted") \
                        and out.get("verificationStatus") == "RECORDED" and "failure" not in out \
                        and out.get("digestsAfter") is not None:
                    return self._finish_bookkeeping(copy.deepcopy(out), local.get("model"), local.get("h"))
            tb = tb.tb_next
        return None

    @staticmethod
    def _finish_bookkeeping(out, model, h):
        """Exactly the deterministic bookkeeping of run_draw's finally block
        (V0.13h, frozen): calls, modelCallsCompleted, process. Replay verifies
        every field of the result."""
        out["calls"] = copy.deepcopy(model.calls) if model else []
        out["modelCallsCompleted"] = len(out["calls"])
        if out["record"] is not None:
            out["process"] = {k: out["record"][k] for k in ("eligibility", "exposure", "validationFirstCorrection",
                                                             "correctionFailure", "deliberateNoAction", "processFailure",
                                                             "submittedAction", "executedAction")}
        else:
            turns = [e["answer"].get("result") for e in (h.log if h else []) if e["op"] == "alternatives.v3.turn"
                     and isinstance(e.get("answer"), dict) and e["answer"].get("ok")]
            turn = turns[-1] if turns else None
            pending = getattr(model, "lastRequest", {}).get("messages", [])
            correction = any("PHASE 2 validation correction" in (call.get("request", {}).get("messages", [{}])[-1].get("content") or "")
                             for call in out["calls"])
            out["process"] = {"eligibility": {"eligible": turn.get("due") if turn else None,
                                              "reason": turn.get("why") if turn else "UNVERIFIED before comparison"},
                              "exposure": bool(turn and turn.get("message") and any(
                                  turn["message"] == m.get("content") for call in out["calls"]
                                  for m in call.get("request", {}).get("messages", []))),
                              "validationFirstCorrection": correction, "correctionFailure": None if correction else False,
                              "deliberateNoAction": False, "processFailure": True, "intendedAction": None,
                              "validatedAction": None, "submittedAction": None,
                              "executedAction": {"status": "NOT_EXECUTED"},
                              "pendingMessage": pending[-1]["content"] if pending else None}
        return out

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
        return {"drawsScheduled": len(self.sched), "drawsCompleted": completed, "drawInterrupted": interrupted,
                "drawAborted": aborted, "drawsNotRun": len(self.sched) - completed - (interrupted is not None)
                - (aborted is not None), "modelCalls": self.calls_k,
                "healthChecks": len(self.doc["healthChecks"]) if self.doc else 0}

    def _transition(self, event, ts=None, extra=None, on_durable=None, **detail):
        key = (self.state, event)
        if key not in self.table or event in self.spec["sealOnlyEvents"]:
            raise _FailClosed("lifecycle", LS.SpecError(f"illegal transition {key}"))
        to = self.table[key]
        ts = ts or self.utc()
        doc = copy.deepcopy(self.doc)
        doc["journal"].append(dict({"n": len(doc["journal"]) + 1, "event": event, "from": self.state, "to": to,
                                    "timestamp": ts}, **detail))
        doc["state"] = to
        doc.update(extra or {})

        def durable():
            self.doc, self.state = doc, to
            self.check_in_state = None
            if on_durable:
                on_durable()
        doc["counters"] = self._counters()
        self._persist("persist:manifest", "run.json", encode_json(doc), on_durable=durable, event=event)

    # -- safe points ---------------------------------------------------------------

    def _safe_point(self, name, healthCheck=None):
        """Examine a pending interrupt at a declared safe point. Returns True
        when an operator stop was recorded here."""
        self._hook("safe_point", name, healthCheck=healthCheck)
        kind = self.safe_kinds[name]
        if kind == "NOTE":
            return False
        self._check_poison()
        if kind == "STOP":
            if self.deferred is not None:
                x, self.deferred = self.deferred, None
                raise x
            n = self.latch.take()
            if not n:
                return False
            if name == "before_dispatch":
                raise OperatorStop(n)
            self._stop("operator", safePoint=name, healthCheck=healthCheck, signals=n)
            return True
        visit_ts = self.utc()
        self.cleanup_visits.append({"n": len(self.cleanup_visits) + 1, "safePoint": name,
                                    "state": self.state, "timestamp": visit_ts})
        self.doc["cleanup"]["events"] = copy.deepcopy(self.cleanup_visits)
        n = self.latch.take()
        if n:
            disposition = "AFTER_STOP" if self.doc["stop"] is not None else "AFTER_SCHEDULE"
            self.events.append({"n": len(self.events) + 1, "safePoint": name, "state": self.state, "signals": n,
                                "disposition": disposition, "timestamp": visit_ts})
            self.doc["interrupts"] = {"signal": "SIGINT", "events": copy.deepcopy(self.events)}
        return False

    # -- the run -----------------------------------------------------------------

    def run(self):
        if os.path.exists(self.live):
            raise ExperimentError(f"{self.live} exists: a run is never resumed, extended or overwritten")
        if not self.authorization or not self.authorization.strip():
            raise ExperimentError("no authorization")
        self.latch.install()
        try:
            os.makedirs(self.live)
            result = self._run()
        except RecordingError as x:
            self.latch.restore()
            x.interrupts = self._interrupts(late=self.latch.take())
            raise
        finally:
            self.latch.restore()
        # The handler is restored: every interrupt counted after the final manifest is reported here.
        result["interrupts"] = self._interrupts(late=self.latch.take())
        return result

    def _interrupts(self, late):
        stop = (self.doc or {}).get("stop") or {}
        return {"signalsReceived": self.latch.count, "archived": copy.deepcopy(self.events), "unarchived": late,
                "stopSafePoint": stop.get("safePoint") if stop.get("code") == "operator" else None,
                "postCompletion": any(e["disposition"] == "AFTER_SCHEDULE" for e in self.events)
                or (late > 0 and self.doc is not None and self.doc.get("stop") is None)}

    def _create(self):
        self._persist("persist:ledger_init", "draws.json", encode_json([]))
        self._persist("persist:calls_init", CALLS, b"")
        ts = self.utc()
        entry = {"n": 1, "event": "create", "from": None, "to": "CREATED", "timestamp": ts}
        if self.table.get((None, "create")) != "CREATED":
            raise LS.SpecError("create")
        doc = {"schema": RUN_SCHEMA, "driver": DRIVER,
               "lifecycle": {"id": LS.LIFECYCLE_ID, "sha256": LS.spec_sha256()},
               "protocol": self.proto["id"], "protocolSha256": self.proto_sha,
               "scheduleSha256": sha256(canon(self.sched)), "runId": self.run_id,
               "authorization": self.authorization, "provenance": dict(self.provenance),
               "serverObserved": self.state_fn is not None, "started": ts,
               "state": "CREATED", "journal": [entry], "counters": None, "healthChecks": [],
               "stop": None, "failureEvent": None, "cleanup": {"status": "NOT_STARTED", "failure": None, "timestamp": None, "started": None, "events": []},
               "interrupts": {"signal": "SIGINT", "events": []}, "verificationStatus": "UNVERIFIED", "resume": "never"}
        self.doc = doc
        doc["counters"] = self._counters()

        def durable():
            self.state = "CREATED"
        self._persist("persist:manifest", "run.json", encode_json(doc), on_durable=durable, event="create")

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
        retained = self._retain_failure(failure)
        if not self.closed:
            self.closed = True
            try:
                self.opener.close()
            except BaseException:  # noqa: BLE001  (the recording failure stays primary)
                pass
        error = RecordingError("recording failed closed; the archive is not sealed", failure.cause)
        error.retainedEvidence = retained
        raise error from failure.cause

    def _main(self):
        self._transition("begin_preflight")
        stop = self.proto["stop"]
        self.limits = SafeLimits(self, self.proto["maxModelCallsPerDraw"], self.proto["maxModelCalls"],
                                 stop["wallClockSeconds"], self.proto["request"]["timeoutSeconds"],
                                 max_fails=stop["consecutiveTransportFailures"], now=self.now)
        request = dict(self.spec["health"]["request"])
        if self.state_fn is not None:
            if self._safe_point("before_health_check"):
                return
            st, obs, k = self._health_read(None)
            if st is not None:
                self._save_before(st)
            if obs is None:
                return self._stop("driver", failure={"type": "ExperimentError", "message": "invalid health observation"},
                                  healthCheck=k)
            if obs["outcome"] != "HEALTHY":
                return self._stop("health", healthCheck=k, request=request, message="the server did not answer healthy")
            if self._safe_point("after_health_check", healthCheck=k):
                return
        self._transition("preflight_passed")
        prev = None
        for i, entry in enumerate(self.sched):
            if self._safe_point("between_draws"):
                return
            if self.state_fn is not None and (entry["model"], entry["draw"]) != prev:
                prev = (entry["model"], entry["draw"])
                if self._safe_point("before_health_check"):
                    return
                k = len(self.doc["healthChecks"])
                self._transition("begin_health_check", healthCheck=k)
                st, obs, k = self._health_read(entry["seq"])
                if obs is None:
                    return self._stop("driver", failure={"type": "ExperimentError",
                                                         "message": "invalid health observation"}, healthCheck=k)
                if obs["outcome"] != "HEALTHY":
                    return self._stop("health", healthCheck=k, request=request,
                                      message="the server did not answer healthy")
                if X.server_signature(st) != self.signature:
                    return self._stop("server_change", healthCheck=k, beforeSignature=self.signature,
                                      message="server changed")
                if self._safe_point("after_health_check", healthCheck=k):
                    return
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
            if self._safe_point("after_draw_persisted"):
                return
            self._transition("schedule_done" if i == len(self.sched) - 1 else "continue")

    def _save_before(self, st):
        def durable():
            self.before_saved = True
        self.signature = X.server_signature(st)
        self._persist("persist:server_before", "server-before.json", encode_json(st), on_durable=durable)

    def _health_read(self, before_draw):
        """Append the health check, read once, record what was received.
        Returns (server state or None, valid observation or None, index)."""
        k = len(self.doc["healthChecks"])
        self._hook("health_read", "before", healthCheck=k)
        check = {"index": k, "state": self.state, "beforeDraw": before_draw, "serverState": None, "rejected": None}
        self.doc["healthChecks"].append(check)
        self.check_in_state, self.reading = k, k
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
        return st, obs, k

    def _new_draw(self):
        self.pending, self.draw_entered, self.draw_n = None, False, 0

    def _dispatch_intent(self, n):
        k = self.calls_k + 1
        rec = {"schema": CALLS_SCHEMA, "record": "dispatch", "k": k, "draw": self.draw_seq, "n": n,
               "timestamp": self.utc()}
        ctx = self._append(rec, before_poisons=False)
        self.calls_k, self.draw_n = k, n
        self._window(lambda: self._hook("append:calls", "after", **ctx))

    def _transport(self, entry, transport):
        """dreamteam.response-handoff/v1: dispatched (the intent is durable before
        this runs), received (owned by the recorder in the receiving statement),
        recorded (outcome appended and fsynced), consumed (only then returned)."""
        def wrapped(payload):
            k, n, seq = self.calls_k, self.draw_n, entry["seq"]
            req = hashlib.sha256(json.dumps(payload).encode("utf-8")).hexdigest()
            try:
                self._hook("draw_execute", "model_call", draw=seq, k=k)
                self.received = result = transport(payload)
            except BaseException as x:
                rec = {"schema": CALLS_SCHEMA, "record": "raised", "k": k, "draw": seq, "n": n, "requestSha256": req,
                       "exception": {"type": type(x).__name__, "message": str(x)}, "timestamp": self.utc()}
                self.received = rec
                if not self._outcome(rec):
                    self.unrecorded.append(rec)
                self.received = None
                raise
            status, body, err = result
            rec = {"schema": CALLS_SCHEMA, "record": "returned", "k": k, "draw": seq, "n": n, "requestSha256": req,
                   "status": status, "error": err, "responseSha256": sha256(body or ""), "body": body,
                   "timestamp": self.utc()}
            self.received = rec
            self._window(lambda: self._safe_point("after_transport_returns"))
            if not self._outcome(rec):
                self.unrecorded.append(rec)
                self.received = None
                raise ResponseNotRecorded(k, self.poisoned)
            self.received = None
            self._window(lambda: self._safe_point("after_call_persisted"))
            return status, body, err
        return wrapped

    def _outcome(self, rec):
        """Append the outcome; True once it is durable. A failure poisons the
        archive (the run is never sealed) and returns False."""
        try:
            ctx = self._append(rec, before_poisons=True)
        except _FailClosed:
            return False
        except BaseException as x:  # noqa: BLE001  (any other fault inside the window poisons too)
            self._poison("recording a call outcome", x)
            return False
        self._window(lambda: self._hook("append:calls", "after", **ctx))
        return True

    def _execute(self, entry):
        self._hook("draw_execute", "before", draw=entry["seq"])
        c = self.ctxs.by_id[entry["context"]]
        settings, mem = self.ctxs.settings[entry["model"]], self.ctxs.memory(c, entry["model"])
        wrapped = self._transport(entry, self.transports[entry["model"]])

        def opener(ctx, arm):
            self._hook("draw_execute", "engine_open", draw=entry["seq"])
            return self.opener(ctx, arm)
        self.draw_seq, self.draw_n = entry["seq"], 0
        self.draw_entered = True
        self.pending = self._draw_result(entry, c, settings, opener, mem, wrapped, self.limits, utc=self.utc, now=self.now)
        self._hook("draw_execute", "after", draw=entry["seq"])
        return self.pending

    def _commit(self, d, reconciled=False):
        rows = self.rows + [d]

        def durable():
            self.rows = rows
        self._persist("persist:ledger", "draws.json", encode_json(rows), on_durable=durable, draw=d["seq"])
        self._commit_transition(d, reconciled)

    def _commit_transition(self, d, reconciled):
        def durable():
            self.pending = None
        self._transition("commit_draw", on_durable=durable, draw=d["seq"], status=d["status"],
                         rowSha256=sha256(canon(d)), modelCalls=len(d["calls"]), reconciled=reconciled)
        self._clear_handoff()

    # -- stops ---------------------------------------------------------------------

    def _next_draw(self):
        if self.aborted is not None:
            return {k: self.aborted[k] for k in ("seq", "model", "context", "arm", "draw")}
        if self.rows and self.rows[-1]["status"] == "interrupted":
            return {k: self.rows[-1][k] for k in ("seq", "model", "context", "arm", "draw")}
        return self.sched[len(self.rows)] if len(self.rows) < len(self.sched) else None

    def _stop_object(self, code, state, message=None, failure=None, evidence=None, request=None, healthCheck=None,
                     beforeSignature=None, safePoint=None):
        if message is None:
            message = (f"{failure['type']}: {failure['message']}" if failure else
                       f"operator interrupt processed at safe point {safePoint}" if code == "operator" else code)
        return {"code": code, "state": state, "message": message, "failure": failure, "evidence": evidence,
                "request": request, "healthCheck": healthCheck, "beforeSignature": beforeSignature, "safePoint": safePoint}

    def _failure_event(self, stop, ts):
        counters = self._counters()
        return {"schema": FAILURE_SCHEMA, "runId": self.run_id, "protocolSha256": self.proto_sha,
                "scheduleSha256": self.doc["scheduleSha256"], "state": stop["state"], "stop": stop,
                "nextDraw": self._next_draw(), "drawsCompleted": counters["drawsCompleted"],
                "modelCalls": counters["modelCalls"], "timestamp": ts, "verificationStatus": "UNVERIFIED"}

    def _stop(self, code, signals=None, **fields):
        if self.deferred is not None:
            raise _FailClosed("a deferred recorder fault is unresolved when the run stops", self.deferred)
        self.stopping = True
        ts = self.utc()
        stop = self._stop_object(code, self.state, **fields)
        extra = {"stop": stop, "failureEvent": self._failure_event(stop, ts)}
        if code == "operator":
            self.events.append({"n": len(self.events) + 1, "safePoint": stop["safePoint"], "state": self.state,
                                "signals": signals, "disposition": "STOP", "timestamp": ts})
            extra["interrupts"] = {"signal": "SIGINT", "events": copy.deepcopy(self.events)}
        self._transition("stop", ts=ts, extra=extra, code=code)

    def _handle(self, x):
        """A driver exception, or an OperatorStop from before_dispatch, outside
        persistence: reconcile the committed ledger, keep a returned draw,
        record the stop."""
        self.stopping = True
        if self.received is not None or self.unrecorded:
            # R-RESPONSE-003: a received model response that did not pass through its
            # durable record is never silently dropped by ordinary stop handling.
            raise _FailClosed("response handoff", ExperimentError(
                "a received model response was not consumed through its durable record"))
        operator = isinstance(x, OperatorStop)
        code = "operator" if operator else "driver"
        failure = {"type": type(x).__name__, "message": str(x)}
        disk = self._read_ledger()
        if canon(disk) != canon(self.rows) and not (self.pending is not None and canon(disk) == canon(self.rows + [self.pending])):
            raise _FailClosed("reconciliation", ExperimentError("the committed ledger disagrees with the recorder"))
        if self.state == "DRAW_ACTIVE":
            if self.pending is not None:
                if self.handoff is None:
                    self._handoff(self.pending)
                if operator:
                    raise _FailClosed("reconciliation", ExperimentError("an operator stop after the draw returned"))
                if canon(disk) == canon(self.rows) and (not self.rows or self.rows[-1] is not self.pending):
                    self._commit(self.pending, reconciled=True)
                else:
                    self.rows = disk
                    self._commit_transition(self.pending, reconciled=True)
            else:
                entry = self.sched[len(self.rows)]
                aborted = getattr(x, "partialDraw", None)
                if aborted is None:
                    if operator:
                        raise _FailClosed("reconciliation", ExperimentError("an operator stop without draw evidence"))
                    unknown = self.draw_entered
                    aborted = {"schema": H.DRAW_SCHEMA, **entry, "status": "aborted", "halt": None,
                               "calls": None if unknown else [], "engine": None if unknown else [], "batches": None,
                               "record": None, "initialDigests": None, "digestsAfter": None, "process": None,
                               "submittedRepresentation": None, "submittedRepresentationSha256": None,
                               "engineInitialization": "UNKNOWN" if unknown else "NOT_STARTED",
                               "failureStage": "draw_unrecorded" if unknown else "draw_setup",
                               "modelCallsCompleted": None if unknown else 0, "verificationStatus": "UNVERIFIED",
                               "failure": failure}
                aborted = copy.deepcopy(aborted)
                aborted["stopReason"] = code

                def durable():
                    self.aborted = aborted
                self._persist("persist:aborted", "aborted-draw.json", encode_json(aborted), on_durable=durable,
                              draw=entry["seq"])
                if operator:
                    return self._stop("operator", safePoint="before_dispatch", signals=x.signals)
                return self._stop("driver", failure=aborted["failure"])
        if operator:
            raise _FailClosed("lifecycle", ExperimentError("an operator stop outside a draw"))
        if self.state == "DRAW_RECORDED" and self.handoff is not None:
            self._clear_handoff()
        checks = self.doc["healthChecks"]
        if checks and checks[0]["state"] == "PREFLIGHT" and checks[0]["serverState"] is not None and not self.before_saved:
            self._save_before(checks[0]["serverState"])  # the initial read returned; keep its evidence
        if self.state == "DRAW_RECORDED" and self.rows[-1]["status"] != "completed":
            halt = self.rows[-1]["halt"]  # the halt preceded this exception and stays the stop
            return self._stop(halt["code"], message=halt["message"], evidence=halt.get("evidence"))
        request = dict(self.spec["health"]["request"]) if self.reading is not None else None
        hc = self.check_in_state if self.state in ("PREFLIGHT", "HEALTH_CHECK") else None
        return self._stop("driver", failure=failure, request=request, healthCheck=hc)

    # -- cleanup and finalization ---------------------------------------------------

    def _cleanup_and_finalize(self):
        self.stopping = True
        self.doc["cleanup"]["started"] = self.utc()
        self._safe_point("before_cleanup")
        self._hook("session_cleanup", "before")
        failure = None
        try:
            self.closed = True
            self.opener.close()
            self._hook("session_cleanup", "during")
        except BaseException as x:  # noqa: BLE001  (recorded separately from any primary stop)
            failure = {"type": type(x).__name__, "message": str(x)}
        ts = self.utc()
        self.doc["cleanup"].update(status="FAILED" if failure else "SUCCEEDED", failure=failure, timestamp=ts)
        if failure is not None and self.doc["stop"] is None:
            stop = self._stop_object("driver", "CLEANUP", failure=failure)
            self.doc["stop"], self.doc["failureEvent"] = stop, self._failure_event(stop, ts)
        self._safe_point("after_cleanup")
        self._hook("session_cleanup", "after")
        self._safe_point("before_finalization")
        complete = (self.doc["stop"] is None and failure is None and len(self.rows) == len(self.sched)
                    and all(r["status"] == "completed" for r in self.rows))
        self._transition("cleanup_failed" if failure else "cleanup_succeeded", ts=ts)
        doc = self.doc
        if doc["failureEvent"] is not None:
            self._persist("persist:failure_event", "failure-event.json", encode_json(doc["failureEvent"]))
        terminal = "COMPLETE" if complete else "PARTIAL"
        event = "finalize_complete" if complete else "finalize_partial"
        if self.table.get(("FINALIZING", event)) != terminal or self.state != "FINALIZING":
            raise _FailClosed("lifecycle", LS.SpecError("illegal finalization"))
        commit = self._seal(doc, terminal, event)
        return {"schema": RESULT_SCHEMA, "state": terminal, "runId": self.run_id, "archive": self.live,
                "manifest": doc, "seal": self.seal, "interrupts": None, "commit": commit}

    def _seal_document(self, doc, terminal, event):
        names = sorted(os.listdir(self.live))
        allowed = set(self.spec["allowedArchiveFiles"]) - {SEAL}
        stray = [n for n in names if n not in allowed]
        if stray:
            raise _FailClosed("sealing", ExperimentError("unexpected archive files: " + ", ".join(stray)))
        raw = {}
        for n in names:
            with open(os.path.join(self.live, n), "rb") as f:
                raw[n] = f.read()
        for n, data in self.written.items():
            if n != CALLS and raw.get(n) != data:
                raise _FailClosed("sealing", ExperimentError(f"{n} on disk is not what was written"))
        if hashlib.sha256(raw.get(CALLS, b"")).hexdigest() != self.calls_sha.hexdigest() or \
                raw.get(CALLS, b"").count(b"\n") != self.calls_records:
            raise _FailClosed("sealing", ExperimentError("the call journal on disk is not what was appended"))
        h = lambda n: hashlib.sha256(raw[n]).hexdigest() if n in raw else None  # noqa: E731
        return {"schema": SEAL_SCHEMA, "finalization": FINALIZATION,
                "lifecycle": {"id": LS.LIFECYCLE_ID, "sha256": LS.spec_sha256()}, "runSchema": RUN_SCHEMA,
                "driver": DRIVER, "runId": self.run_id, "protocol": {"id": self.proto["id"], "sha256": self.proto_sha},
                "scheduleSha256": doc["scheduleSha256"], "terminalState": terminal,
                "transition": {"n": len(doc["journal"]) + 1, "event": event, "from": "FINALIZING", "to": terminal,
                               "timestamp": self.utc()},
                "manifest": {"file": "run.json", "sha256": h("run.json"), "journalEntries": len(doc["journal"])},
                "ledger": {"file": "draws.json", "sha256": h("draws.json"), "rows": len(self.rows),
                           "completed": sum(r["status"] == "completed" for r in self.rows),
                           "rowSha256": [sha256(canon(r)) for r in self.rows]},
                "calls": {"file": CALLS, "sha256": h(CALLS), "records": self.calls_records, "dispatched": self.calls_k},
                "health": {"checks": len(doc["healthChecks"]),
                           "checkSha256": [sha256(canon(c)) for c in doc["healthChecks"]],
                           "serverBefore": h("server-before.json")},
                "sideArtifacts": {n: h(n) for n in SIDE}, "counters": copy.deepcopy(doc["counters"])}

    def _seal(self, doc, terminal, event):
        """Write the completion seal; its rename is the commit point and the
        last required operation. Returns the post-commit report."""
        self._check_poison()
        self.seal = self._seal_document(doc, terminal, event)
        data = encode_json(self.seal)
        path = os.path.join(self.live, SEAL)
        self._hook("seal", "before")
        fd, tmp, committed = None, None, False
        try:
            fd, tmp = self.fs.mkstemp(self.live, ".seal-")
            self.fs.write(fd, data)
            self._hook("seal", "write")
            self._hook("seal", "file_fsync")
            self.fs.fsync(fd)
            self.fs.close(fd)
            fd = None
            self._hook("seal", "rename")
            try:
                self.fs.replace(tmp, path)
            except BaseException:
                # An error from the rename itself: committed only if the seal is in place, byte for byte.
                try:
                    with open(path, "rb") as f:
                        committed = f.read() == data
                except OSError:
                    committed = False
                if not committed:
                    raise
                tmp = None
                raise
            tmp, committed = None, True
        except BaseException as x:
            if not committed:
                raise self._poison("sealing", x)
            post_error = x
        else:
            post_error = None
        finally:
            if fd is not None:
                try:
                    os.close(fd)
                except OSError:
                    pass
            if tmp is not None:
                try:
                    self.fs.unlink(tmp)
                except BaseException:  # noqa: BLE001
                    pass
        # -- committed: nothing below is required; every outcome is reported, none changes the archive.
        report = {"sealRenamed": True, "directorySync": "NOT_ATTEMPTED", "postCommitError": None}
        if post_error is not None:
            report["postCommitError"] = {"type": type(post_error).__name__, "message": str(post_error)}
        try:
            self._hook("seal", "renamed")
        except BaseException as x:  # noqa: BLE001  (best effort after the commit point)
            report["postCommitError"] = report["postCommitError"] or {"type": type(x).__name__, "message": str(x)}
        try:
            self._hook("seal", "dir_fsync")
            report["directorySync"] = self.fs.dir_fsync(self.live)
        except BaseException as x:  # noqa: BLE001
            report["directorySync"] = "FAILED"
            report["postCommitError"] = report["postCommitError"] or {"type": type(x).__name__, "message": str(x)}
        try:
            self._hook("seal", "after")
        except BaseException as x:  # noqa: BLE001
            report["postCommitError"] = report["postCommitError"] or {"type": type(x).__name__, "message": str(x)}
        self.state = terminal
        return report


def run_schedule(proto, proto_sha, sched, ctxs, opener, transports, out, authorization, **kw):
    """Record one run. Returns the run result (dreamteam.run-result/v1:
    state COMPLETE or PARTIAL, manifest, seal, interrupts, commit report) or
    raises RecordingError (the archive is unsealed and INVALID; the error
    carries the interrupts summary)."""
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
