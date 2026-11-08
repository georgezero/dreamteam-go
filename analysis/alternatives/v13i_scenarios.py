"""Deterministic, model-free scenarios for the V0.13i lifecycle checks (fault
matrix, real-signal tests, loopback tests, regression witnesses, record-only
examples). Every model reply is scripted; every health response is either a
scripted HTTP byte stream parsed by the standard library or a genuine
exception object raised into the production probe. No model endpoint.
"""
import copy
import datetime
import errno
import http.client
import io
import json
import os
from pathlib import Path
import sys
import urllib.error

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13i as I  # noqa: E402
import v13h as D  # noqa: E402
import v13g_fidelity as F  # noqa: E402
import lifecycle_spec as LS  # noqa: E402

ROOT = HERE.parents[1]
CONTEXTS = str(ROOT / "docs/benchmark/v13g-dev/contexts")
SYNTHETIC = {"timestamps": "SYNTHETIC_FIXTURE", "runId": "SYNTHETIC_FIXTURE"}


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
    "http_503": b'HTTP/1.1 503 Service Unavailable\r\nContent-Length: 22\r\n\r\n{"status":"unhealthy"}',
}


def scripted_urlopen(mode):
    """A urlopen whose response or failure is genuine for its mode."""
    def urlopen(url, timeout=20):
        if mode == "refused":
            raise urllib.error.URLError(ConnectionRefusedError(errno.ECONNREFUSED, "Connection refused"))
        if mode == "timeout":
            raise TimeoutError("timed out")
        if mode == "connect_timeout":
            raise urllib.error.URLError(TimeoutError("timed out"))
        if mode == "reset":
            raise ConnectionResetError(errno.ECONNRESET, "Connection reset by peer")
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
        return {"health": I.probe_health("http://localhost:9", urlopen=scripted_urlopen(mode), utc=self.utc)}


class Model:
    """A scripted model transport; inject() makes the next call a recorded
    transport failure (no response)."""

    def __init__(self, replies):
        self.t, self.fail_next = D.scripted(replies), False

    def inject(self):
        self.fail_next = True

    def __call__(self, payload):
        if self.fail_next:
            self.fail_next = False
            return None, "", "timeout: injected transport failure"
        return self.t(payload)


class Fixture:
    """The record-only two-draw lifecycle fixture: its own protocol identity,
    not the 262-draw experiment."""

    def __init__(self, binary, arms=("control", "control")):
        rows, h = D.load_contexts(CONTEXTS)
        proto, _ = I.draft_protocol(rows, h)
        self.binary = binary
        self.sched = [{"seq": i, "model": D.QWEN, "context": "S1-a", "arm": arm, "draw": i} for i, arm in enumerate(arms, 1)]
        self.proto = {"id": "record-only-lifecycle-fixture", "notice": "SCRIPTED RECORD ONLY: not the experimental schedule",
                      "modelOrder": [D.QWEN], "contexts": {"S1-a": proto["contexts"]["S1-a"]},
                      "maxModelCallsPerDraw": proto["maxModelCallsPerDraw"], "maxModelCalls": proto["maxModelCalls"],
                      "request": proto["request"], "stop": proto["stop"], "schedule": self.sched,
                      "lifecycle": proto["lifecycle"], "driver": I.DRIVER}
        self.proto_sha = D.sha256(D.canon(self.proto))
        self.ctx = D.Contexts(proto, [r for r in rows if r["kind"] == "constructed"], None,
                              {D.SB.SETTINGS_ENV: D.SB.REPLAY_URL})

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

    def record(self, out, health=None, opener=None, replies=None, model=None, hook=None, now=None, clock=None,
               run_id="00000000-0000-4000-8000-0000000013a1", signal_guard=True):
        clock = clock or Clock()
        if replies is None:
            replies = [F.ABANDON if e["arm"] == "control" else F.SEL("no_action") for e in self.sched] + [F.ABANDON] * 4
        model = model or Model(replies)
        return I.run_schedule(self.proto, self.proto_sha, self.sched, self.ctx, opener or self.ctx.opener(self.binary),
                              {D.QWEN: model}, out, "scripted lifecycle fixture", state_fn=health, now=now or Monotonic(),
                              utc=clock, hook=hook, run_id=run_id, provenance=SYNTHETIC, signal_guard=signal_guard)

    def classify(self, out):
        import v13i_verify as V
        return V.classify(out, self.proto, self.proto_sha, self.sched, self.ctx.by_id, allow_synthetic=True)

    def analyze(self, out):
        import v13i_verify as V
        return V.analyze(self.binary, {}, self.ctx, self.proto, self.sched, out, self.proto_sha, allow_synthetic=True)


class FailingOpener:
    """Pool wrapper: fail engine opening at the nth draw, or fail release."""

    def __init__(self, pool, open_error=None, at=None, close_error=None):
        self.pool, self.open_error, self.at, self.close_error, self.n = pool, open_error, at, close_error, 0

    def __call__(self, c, arm):
        self.n += 1
        if self.open_error is not None and (self.at is None or self.n == self.at):
            raise self.open_error
        return self.pool(c, arm)

    def close(self):
        self.pool.close()
        if self.close_error is not None:
            raise self.close_error


def summary(fixture, out, result):
    """The observed lifecycle outcome of one recorded run."""
    live = Path(out) / "live"
    doc = json.loads((live / "run.json").read_text()) if (live / "run.json").exists() else None
    verdict = fixture.classify(out)
    ab = json.loads((live / "aborted-draw.json").read_text()) if (live / "aborted-draw.json").exists() else None
    obs = {"recorder": result, "classification": verdict["state"],
           "violations": sorted({v["invariant"] for v in verdict["violations"]}),
           "manifest": doc is not None, "marker": (live / I.MARKER).exists()}
    if doc is not None:
        stop = doc.get("stop") or {}
        obs.update(claimed=doc.get("state"), stopCode=stop.get("code"), stopState=stop.get("state"),
                   completed=(doc.get("counters") or {}).get("drawsCompleted"),
                   modelCalls=(doc.get("counters") or {}).get("modelCalls"),
                   cleanup=(doc.get("cleanup") or {}).get("status"), abortedStage=ab["failureStage"] if ab else None)
    return obs


def attempt(fn):
    try:
        doc = fn()
        return {"returned": True, "state": doc["state"] if isinstance(doc, dict) else None}
    except I.RecordingError as x:
        return {"returned": False, "error": "RecordingError", "cause": type(x.cause).__name__ if x.cause else None}
    except BaseException as x:  # noqa: BLE001  (an escaped exception is itself an observation)
        return {"returned": False, "error": type(x).__name__, "message": str(x)}
