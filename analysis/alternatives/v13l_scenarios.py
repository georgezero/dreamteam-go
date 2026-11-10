"""Deterministic, model-free scenarios for the V0.13l safety checks (fault
matrix, real-process signal tests, loopback tests, regression witnesses,
contradictory archives, record-only examples). Every model reply is scripted;
every health response is either a scripted HTTP byte stream parsed by the
standard library or a genuine exception object raised into the production
probe. No model endpoint.
"""
import copy
import datetime
import errno
import http.client
import io
import json
from pathlib import Path
import sys
import urllib.error

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13l as J  # noqa: E402
import v13h as D  # noqa: E402
import v13g_fidelity as F  # noqa: E402
import lifecycle_v4_spec as LS  # noqa: E402

ROOT = HERE.parents[1]
CONTEXTS = str(ROOT / "docs/benchmark/v13g-dev/contexts")
SYNTHETIC = {"timestamps": "SYNTHETIC_FIXTURE", "runId": "SYNTHETIC_FIXTURE"}
RUN_ID = "00000000-0000-4000-8000-0000000013b1"


class Clock:
    """Strictly increasing synthetic UTC timestamps (one microsecond apart)."""

    def __init__(self):
        self.t = datetime.datetime(2026, 1, 1, tzinfo=datetime.timezone.utc)

    def __call__(self):
        self.t += datetime.timedelta(microseconds=1)
        return self.t.strftime("%Y-%m-%dT%H:%M:%S.%fZ")


class Monotonic:
    """The run clock; jump() makes the next reading pass the wall-clock stop."""

    def __init__(self):
        self.value = 0.0

    def jump(self):
        self.value = 10.0 ** 9

    def __call__(self):
        return self.value


class _Wire:
    def __init__(self, data):
        self.data = data

    def makefile(self, *a, **k):
        return io.BytesIO(self.data)


def wire_response(data):
    """A real http.client response parsed from scripted bytes."""
    r = http.client.HTTPResponse(_Wire(data))
    r.begin()
    return r


WIRE = {
    "healthy": b'HTTP/1.1 200 OK\r\nContent-Length: 20\r\n\r\n{"status":"healthy"}',
    "unhealthy": b'HTTP/1.1 200 OK\r\nContent-Length: 22\r\n\r\n{"status":"unhealthy"}',
    "malformed": b'HTTP/1.1 200 OK\r\nContent-Length: 1\r\n\r\n{',
    "list": b'HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\n[]',
    "null": b'HTTP/1.1 200 OK\r\nContent-Length: 4\r\n\r\nnull',
    "healthy_error": b'HTTP/1.1 200 OK\r\nContent-Length: 39\r\n\r\n{"status":"healthy","error":"URLError"}',
    "disconnect": b'',
    "incomplete": b'HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n{"status":',
    "bad_status_line": b'NOT-HTTP garbage\r\n\r\n',
    "http_503": b'HTTP/1.1 503 Service Unavailable\r\nContent-Length: 22\r\n\r\n{"status":"unhealthy"}',
}

# Genuine exception objects raised by urlopen for each mode (DT13I-03 families).
RAISED = {
    "refused": lambda: urllib.error.URLError(ConnectionRefusedError(errno.ECONNREFUSED, "Connection refused")),
    "timeout": lambda: TimeoutError("timed out"),
    "connect_timeout": lambda: urllib.error.URLError(TimeoutError("timed out")),
    "reset": lambda: ConnectionResetError(errno.ECONNRESET, "Connection reset by peer"),
    "url_incomplete": lambda: urllib.error.URLError(http.client.IncompleteRead(b"x", 3)),
    "url_bad_status": lambda: urllib.error.URLError(http.client.BadStatusLine("bad")),
    "not_connected": lambda: http.client.NotConnected(),
    "remote_disconnected": lambda: http.client.RemoteDisconnected("closed"),
    "url_text": lambda: urllib.error.URLError("no host given"),
    "unreachable": lambda: urllib.error.URLError(OSError(errno.ENETUNREACH, "Network is unreachable")),
    "dns": lambda: urllib.error.URLError(__import__("socket").gaierror(-2, "Name or service not known")),
    # Not transport failures: they propagate and stop the run as driver failures.
    "keyboard": lambda: KeyboardInterrupt(),
    "value_error": lambda: ValueError("unknown url type"),
    "permission": lambda: PermissionError(errno.EACCES, "not a socket error"),
    "url_keyboard": lambda: urllib.error.URLError(KeyboardInterrupt()),
}


def scripted_urlopen(mode):
    """A urlopen whose response or failure is genuine for its mode."""
    def urlopen(url, timeout=20):
        if mode in RAISED:
            raise RAISED[mode]()
        r = wire_response(WIRE[mode])
        if r.status >= 400:
            raise urllib.error.HTTPError(url, r.status, r.reason, r.headers, r)
        return r
    return urlopen


class Health:
    """state_fn: the production probe over scripted HTTP; modes[n] (1-based
    read number) overrides the default healthy response; inject() makes the
    next read a genuine connection refusal."""

    def __init__(self, utc, modes=None, default="healthy"):
        self.utc, self.modes, self.default, self.n, self.fail_next = utc, dict(modes or {}), default, 0, False

    def inject(self):
        self.fail_next = True

    def __call__(self):
        self.n += 1
        mode = self.modes.get(self.n, self.default)
        if self.fail_next:
            self.fail_next, mode = False, "refused"
        return {"health": J.probe_health("http://localhost:9", urlopen=scripted_urlopen(mode), utc=self.utc)}


class Model:
    """A scripted model transport; inject() makes the next call a recorded
    transport failure (no response). calls counts transport invocations."""

    def __init__(self, replies):
        self.t, self.fail_next, self.calls = D.scripted(replies), False, 0

    def inject(self):
        self.fail_next = True

    def __call__(self, payload):
        self.calls += 1
        if self.fail_next:
            self.fail_next = False
            return None, "", "timeout: injected transport failure"
        return self.t(payload)


def contexts(proto, rows):
    return J.Contexts(proto, rows, None, {D.SB.SETTINGS_ENV: D.SB.REPLAY_URL})


class Fixture:
    """The record-only two-draw lifecycle fixture: its own protocol identity,
    not the 262-draw experiment."""

    def __init__(self, binary, arms=("control", "control")):
        rows, h = D.load_contexts(CONTEXTS)
        proto, _ = J.draft_protocol(rows, h)
        self.binary = binary
        self.sched = [{"seq": i, "model": D.QWEN, "context": "S1-a", "arm": arm, "draw": i} for i, arm in enumerate(arms, 1)]
        self.proto = {"id": "record-only-lifecycle-fixture-v2", "notice": "SCRIPTED RECORD ONLY: not the experimental schedule",
                      "modelOrder": [D.QWEN], "contexts": {"S1-a": proto["contexts"]["S1-a"]},
                      "maxModelCallsPerDraw": proto["maxModelCallsPerDraw"], "maxModelCalls": proto["maxModelCalls"],
                      "request": proto["request"], "stop": proto["stop"], "schedule": self.sched,
                      "lifecycle": proto["lifecycle"], "driver": J.DRIVER}
        self.proto_sha = D.sha256(D.canon(self.proto))
        self.ctx = contexts(proto, [r for r in rows if r["kind"] == "constructed"])

    def with_limits(self, **patch):
        f = copy.copy(self)
        f.proto = copy.deepcopy(self.proto)
        for k, v in patch.items():
            if k == "consecutiveTransportFailures":
                f.proto["stop"][k] = v
            else:
                f.proto[k] = v
        f.proto_sha = D.sha256(D.canon(f.proto))
        return f

    def replies(self):
        return [F.ABANDON if e["arm"] == "control" else F.SEL("no_action") for e in self.sched] + [F.ABANDON] * 4

    def record(self, out, health=None, opener=None, replies=None, model=None, hook=None, now=None, clock=None,
               run_id=RUN_ID, fs=None, draw_runner=None, provenance=SYNTHETIC):
        clock = clock or Clock()
        model = model or Model(replies if replies is not None else self.replies())
        return J.run_schedule(self.proto, self.proto_sha, self.sched, self.ctx, opener or self.ctx.opener(self.binary),
                              {D.QWEN: model}, out, "scripted lifecycle fixture", state_fn=health, now=now or Monotonic(),
                              utc=clock, hook=hook, run_id=run_id, provenance=provenance, fs=fs, draw_runner=draw_runner)

    def classify(self, out, synthetic=True):
        import v13l_verify as V
        return V.classify(out, self.proto, self.proto_sha, self.sched, self.ctx.by_id, allow_synthetic=synthetic)

    def analyze(self, out, synthetic=True):
        import v13l_verify as V
        return V.analyze(self.binary, {}, self.ctx, self.proto, self.sched, out, self.proto_sha,
                         allow_synthetic=synthetic)


class FailingOpener:
    """Pool wrapper: fail engine opening at the nth draw, or fail release.
    closes counts release calls."""

    def __init__(self, pool, open_error=None, at=None, close_error=None):
        self.pool, self.open_error, self.at, self.close_error, self.n, self.closes = pool, open_error, at, close_error, 0, 0

    def __call__(self, c, arm):
        self.n += 1
        if self.open_error is not None and (self.at is None or self.n == self.at):
            raise self.open_error
        return self.pool(c, arm)

    def close(self):
        self.closes += 1
        self.pool.close()
        if self.close_error is not None:
            raise self.close_error


class FailingFS(J.OsFS):
    """The archive filesystem with a persistent failure: once activated,
    every write, sync, replace, rename, unlink and append open fails with
    EIO, including any recovery operation attempted afterwards."""

    def __init__(self):
        self.active, self.failures = False, 0

    def activate(self):
        self.active = True

    def _check(self, what):
        if self.active:
            self.failures += 1
            raise OSError(errno.EIO, f"persistent filesystem failure ({what})")

    def mkstemp(self, directory, prefix):
        self._check("mkstemp")
        return super().mkstemp(directory, prefix)

    def write(self, fd, data):
        self._check("write")
        return super().write(fd, data)

    def fsync(self, fd):
        self._check("fsync")
        return super().fsync(fd)

    def replace(self, src, dst):
        self._check("replace")
        return super().replace(src, dst)

    def unlink(self, path):
        self._check("unlink")
        return super().unlink(path)

    def open_append(self, path):
        self._check("open_append")
        return super().open_append(path)

    def dir_fsync(self, directory):
        self._check("dir_fsync")
        return super().dir_fsync(directory)


def journal(out):
    p = Path(out) / "live" / J.CALLS
    if not p.exists():
        return None
    recs = []
    for line in p.read_bytes().split(b"\n"):
        if line:
            try:
                recs.append(json.loads(line))
            except ValueError:
                recs.append({"record": "torn"})
    return recs


def summary(fixture, out, result, synthetic=True):
    """The observed outcome of one recorded run, from the recorder's result,
    the independent classification and the persisted files."""
    live = Path(out) / "live"
    doc = json.loads((live / "run.json").read_text()) if (live / "run.json").exists() else None
    verdict = fixture.classify(out, synthetic)
    ab = json.loads((live / "aborted-draw.json").read_text()) if (live / "aborted-draw.json").exists() else None
    seal = json.loads((live / J.SEAL).read_text()) if (live / J.SEAL).exists() else None
    rows = json.loads((live / "draws.json").read_text()) if (live / "draws.json").exists() else None
    recs = journal(out) or []
    dispatched = sum(r.get("record") == "dispatch" for r in recs)
    resolved = sum(r.get("record") in ("returned", "raised") for r in recs)
    reported = (result.get("retainedEvidence") or {}).get("pending") if isinstance(result, dict) else None
    obs = {"recorder": result, "classification": verdict["state"],
           "violations": sorted({v["invariant"] for v in verdict["violations"]}),
           "manifest": doc is not None, "sealed": seal is not None,
           "sealState": seal.get("terminalState") if isinstance(seal, dict) else None,
           "ledgerRows": len(rows) if isinstance(rows, list) else None,
           "ledgerCompleted": sum(r.get("status") == "completed" for r in rows) if isinstance(rows, list) else None,
           "journalDispatched": dispatched, "journalUnresolved": dispatched - resolved,
           "pendingCompleted": verdict["preserved"].get("pendingCompleted", 0),
           "reportedCompleted": int(isinstance(reported, dict) and reported.get("status") == "completed"),
           "reportedRowSha256": D.sha256(D.canon(reported)) if isinstance(reported, dict) else None}
    if doc is not None:
        stop = doc.get("stop") or {}
        obs.update(stopCode=stop.get("code"), stopState=stop.get("state"), safePoint=stop.get("safePoint"),
                   completed=(doc.get("counters") or {}).get("drawsCompleted"),
                   modelCalls=(doc.get("counters") or {}).get("modelCalls"),
                   cleanup=(doc.get("cleanup") or {}).get("status"), abortedStage=ab["failureStage"] if ab else None,
                   interruptDispositions=[e["disposition"] for e in (doc.get("interrupts") or {}).get("events", [])])
    return obs


def attempt(fn):
    try:
        r = fn()
        return {"returned": True, "state": r["state"] if isinstance(r, dict) else None,
                "interrupts": r.get("interrupts") if isinstance(r, dict) else None,
                "commit": r.get("commit") if isinstance(r, dict) else None}
    except J.RecordingError as x:
        return {"returned": False, "error": "RecordingError", "cause": type(x.cause).__name__ if x.cause else None,
                "interrupts": x.interrupts, "retainedEvidence": getattr(x, "retainedEvidence", None)}
    except BaseException as x:  # noqa: BLE001  (an escaped exception is itself an observation)
        return {"returned": False, "error": type(x).__name__, "message": str(x)}


def recorder_label(result):
    return "returned" if result["returned"] else result["error"]
