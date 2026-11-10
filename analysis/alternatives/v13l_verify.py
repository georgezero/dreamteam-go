#!/usr/bin/env python3
"""V0.13l independent archive verifier and analyzer
(dreamteam.alternatives-analysis/v9, partial-run procedure /8).

It loads the safety contract (bindings_v2.json) and reconstructs the
archive's state from the persisted files alone. The completion seal is the
only evidence that a run reached a terminal state, and it is never
sufficient: the verifier walks the journal through the contract's
transition table and predicates, binds the ledger, the call journal, the
aborted artifact, health evidence, stop, cleanup and interrupts to each other
field by field, recomputes every counter and hash, derives the terminal state
the evidence supports, and requires the seal to agree. It never takes a
claimed status as truth and does not import the recorder (v13l.py).
Completed and interrupted rows are then replayed without a model through the
unchanged V0.13h draw machinery.

V0.13l: every comparison of retained JSON evidence uses strict typed
equality (dreamteam.strict-json/v1: boolean, integer, float, null and
absence are distinct; duplicate keys and non-finite numbers are refused);
failures, stop messages and halts are derived from the recorded event
sequence and compared with the archived claims; only an external cause
may remain UNVERIFIED.

    python3 -I -B analysis/alternatives/v13l_verify.py classify --protocol FILE --contexts DIR --dir RUN
    python3 -I -B analysis/alternatives/v13l_verify.py analyze --binary ./dt --protocol FILE --contexts DIR \\
        --replays DIR --dir RUN --out DIR
"""
import argparse
import base64
import collections
import datetime
import hashlib
import json
import os
import re
import math
import copy
import struct
import sys
import uuid

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import v13h as H  # noqa: E402  (draw replay, classification and execution; not the run recorder)

SB, X = H.SB, H.X
sha256, canon = H.sha256, H.canon
ExperimentError = H.ExperimentError

RUN_SCHEMA = "dreamteam.alternatives-run/v8"
DRIVER = "dreamteam-v13l-driver/8"
FAILURE_SCHEMA = "dreamteam.run-failure/v3"
HEALTH_SCHEMA = "dreamteam.health-observation/v2"
CALLS_SCHEMA = "dreamteam.call-journal/v1"
SEAL_SCHEMA = "dreamteam.completion-seal/v1"
FINALIZATION = "dreamteam.run-finalization/v1"
ANALYSIS_SCHEMA = "dreamteam.alternatives-analysis/v9"
SEAL, CALLS = "SEAL.json", "calls.jsonl"
SIDE = ("server-before.json", "aborted-draw.json", "failure-event.json")
HALTS = {"draw_call_limit", "call_budget", "wall_clock", "transport"}
RUN_KEYS = {"schema", "driver", "lifecycle", "protocol", "protocolSha256", "scheduleSha256", "runId", "authorization",
            "provenance", "serverObserved", "started", "state", "journal", "counters", "healthChecks", "stop",
            "failureEvent", "cleanup", "interrupts", "verificationStatus", "resume"}
STOP_KEYS = {"code", "state", "message", "failure", "evidence", "request", "healthCheck", "beforeSignature", "safePoint"}
DETAIL = {"create": set(), "begin_preflight": set(), "preflight_passed": set(), "begin_health_check": {"healthCheck"},
          "health_passed": {"healthCheck"}, "begin_draw": {"draw"},
          "commit_draw": {"draw", "status", "rowSha256", "modelCalls", "reconciled"}, "continue": set(),
          "schedule_done": set(), "stop": {"code"}, "cleanup_succeeded": set(), "cleanup_failed": set()}
DRAW_KEYS = {"schema", "seq", "model", "context", "arm", "draw", "status", "halt", "initialDigests", "record", "batches",
             "engine", "digestsAfter", "calls", "submittedRepresentation", "submittedRepresentationSha256",
             "engineInitialization", "failureStage", "verificationStatus", "modelCallsCompleted", "process"}
PROCESS_KEYS = ("eligibility", "exposure", "validationFirstCorrection", "correctionFailure", "deliberateNoAction",
                "processFailure", "submittedAction", "executedAction")
IDENTITY = ("seq", "model", "context", "arm", "draw")
RECORD_FIELDS = {"dispatch": {"schema", "record", "k", "draw", "n", "timestamp"},
                 "returned": {"schema", "record", "k", "draw", "n", "requestSha256", "status", "error", "responseSha256",
                              "body", "timestamp"},
                 "raised": {"schema", "record", "k", "draw", "n", "requestSha256", "exception", "timestamp"}}


class ContractData:
    """Data access only. No recorder or shared validation implementation."""
    LIFECYCLE_ID = "dreamteam.run-lifecycle/v4"

    @staticmethod
    def raw():
        with open(os.path.join(HERE, "bindings_v2.json"), "rb") as source:
            return source.read()

    @classmethod
    def document(cls):
        doc = json.loads(cls.raw())
        if doc["schema"] != "dreamteam.evidence-bindings/2":
            raise ExperimentError("evidence binding identity")
        return doc

    @classmethod
    def load(cls):
        return cls.document()["lifecycle"]

    @classmethod
    def bindings(cls):
        return {b["id"]: b for b in cls.document()["bindings"]}

    @classmethod
    def rule(cls, identity):
        return cls.bindings()[identity]["rules"][0]

    @classmethod
    def spec_sha256(cls):
        return sha256(cls.raw())

    @staticmethod
    def transition_table(spec):
        return {(t["from"], t["event"]): t["to"] for t in spec["transitions"]}


DATA = ContractData
_TIMESTAMP = re.compile(r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]{1,6})?Z", re.ASCII)


def _timestamp(value):
    if not isinstance(value, str) or _TIMESTAMP.fullmatch(value) is None:
        return None
    try:
        return datetime.datetime.fromisoformat(value[:-1] + "+00:00")
    except ValueError:
        return None


def _health_category(raw, maximum):
    if len(raw) > maximum:
        return "oversized_body"
    try:
        raw.decode("utf-8", "strict")
    except UnicodeDecodeError:
        return "non_utf8_body"
    try:
        doc = _strict_json(raw)
    except ValueError:
        return "invalid_json"
    if not isinstance(doc, dict):
        return "not_an_object"
    status = doc.get("status")
    if "error" in doc and status == "healthy":
        return "contradictory_body"
    if not isinstance(status, str) or not status:
        return "missing_status"
    return "healthy" if status == "healthy" else "unhealthy_status"


class Invalid(Exception):
    def __init__(self, invariant, message):
        super().__init__(f"{invariant}: {message}")
        self.invariant, self.message = invariant, message


def _need(ok, invariant, message):
    if not ok:
        raise Invalid(invariant, message)


def _nat(x):
    return isinstance(x, int) and not isinstance(x, bool) and x >= 0


def _failure_shape(f):
    return isinstance(f, dict) and set(f) == {"type", "message"} and isinstance(f["type"], str) and bool(f["type"]) \
        and isinstance(f["message"], str)


def _ts(value, what):
    t = _timestamp(value)
    _need(t is not None, "L13", f"invalid or missing timestamp: {what}")
    return t


def _strict_json(data):
    """dreamteam.strict-json/v1 parsing: strict UTF-8, no duplicate keys at any
    depth, no NaN/Infinity literal and no number that overflows to infinity;
    integers stay exact."""
    def pairs(items):
        keys = [k for k, _ in items]
        if len(keys) != len(set(keys)):
            raise ValueError("duplicate key")
        return dict(items)

    def constant(name):
        raise ValueError("non-finite number " + name)

    def number(text):
        value = float(text)
        if not math.isfinite(value):
            raise ValueError("non-finite number " + text)
        return value
    if isinstance(data, str):
        data = data.encode("utf-8")
    return json.loads(data.decode("utf-8"), object_pairs_hook=pairs, parse_constant=constant, parse_float=number)


_TAGS = {type(None): "null", bool: "boolean", int: "integer", float: "float", str: "string", list: "array", dict: "object"}


def _tag(value):
    return _TAGS.get(type(value), "unsupported")


def _same(a, b):
    """dreamteam.strict-json/v1 equality: same type tag and equal content,
    recursively. bool is never int, int is never float, a numeric string is
    never a number, null is a value; floats compare bitwise."""
    stack = [(a, b)]
    while stack:
        x, y = stack.pop()
        t = _tag(x)
        if t != _tag(y) or t == "unsupported":
            return False
        if t == "object":
            if x.keys() != y.keys() or not all(type(k) is str for k in x):
                return False
            stack.extend((x[k], y[k]) for k in x)
        elif t == "array":
            if len(x) != len(y):
                return False
            stack.extend(zip(x, y))
        elif t == "float":
            if struct.pack(">d", x) != struct.pack(">d", y):
                return False
        elif x != y:
            return False
    return True


def _as_archived(value):
    """A value the frozen code produced, as the recorder archives it (json.dumps)
    and as this verifier reads it back (strict parse)."""
    return _strict_json(json.dumps(value, ensure_ascii=False, allow_nan=False))


def _same_as_archived(produced, archived):
    try:
        return _same(_as_archived(produced), archived)
    except (TypeError, ValueError):
        return False


def _member(value, options):
    return any(_same(value, o) for o in options)


class Archive:
    """The persisted files of one run directory, read once."""

    def __init__(self, directory):
        self.live = os.path.join(directory, "live")
        self.files = sorted(os.listdir(self.live)) if os.path.isdir(self.live) else None
        self.raw, self.dirs = {}, []
        for name in self.files or []:
            p = os.path.join(self.live, name)
            if os.path.isfile(p) and not os.path.islink(p):
                with open(p, "rb") as f:
                    self.raw[name] = f.read()
            else:
                self.dirs.append(name)

    def json(self, name):
        return _strict_json(self.raw[name])


class Classifier:
    """The authoritative classifier: S01-S04 and L01-L20 (L11 in analysis)."""

    def __init__(self, archive, proto, proto_sha, sched, contexts=None, allow_synthetic=False):
        self.a, self.proto, self.proto_sha, self.sched = archive, proto, proto_sha, sched
        self.contexts, self.allow_synthetic = contexts or {}, allow_synthetic
        self.bindings = DATA.bindings()
        self.spec = DATA.load()
        self.table = DATA.transition_table(self.spec)
        self.violations = []
        self.facts = {}
        self.derived = None
        self.doc = None
        self.by_draw, self.unresolved, self.dispatched = {}, [], None

    # -- entry --------------------------------------------------------------------

    def run(self):
        try:
            self._load()
        except Invalid as x:
            return self._result(x)
        for group in (self._field_types, self._identity, self._journal, self._ledger, self._calls, self._aborted, self._health,
                      self._stop, self._cleanup, self._interrupts, self._counters, self._terminal, self._seal,
                      self._timestamps):
            try:
                group()
            except Invalid as x:
                self.violations.append({"invariant": x.invariant, "message": x.message})
            except ExperimentError as x:
                self.violations.append({"invariant": "L06", "message": f"{group.__name__[1:]}: {x}"})
            except (KeyError, TypeError, ValueError, AttributeError, IndexError) as x:
                self.violations.append({"invariant": "L12", "message": f"malformed evidence in {group.__name__[1:]}: "
                                                                       f"{type(x).__name__}: {x}"})
        return self._result()

    def _result(self, fatal=None):
        if fatal is not None:
            self.violations.append({"invariant": fatal.invariant, "message": fatal.message})
        doc = self.doc
        seal = getattr(self, "seal", None)
        return {"state": "INVALID" if self.violations or self.derived is None else self.derived,
                "sealed": isinstance(seal, dict),
                "sealClaims": seal.get("terminalState") if isinstance(seal, dict) else None,
                "manifestState": doc.get("state") if isinstance(doc, dict) else None,
                "violations": self.violations, "derived": self.facts, "preserved": self._preserved()}

    def _preserved(self):
        """What durable evidence the archive keeps, whatever its classification
        (completed rows and dispatched calls are never silently lost)."""
        rows = getattr(self, "rows", None)
        recs = getattr(self, "records", None)
        out = {"ledgerRows": len(rows) if isinstance(rows, list) else None,
               "completedRows": sum(isinstance(r, dict) and r.get("status") == "completed" for r in rows)
               if isinstance(rows, list) else None}
        if isinstance(recs, list):
            d = sum(isinstance(r, dict) and r.get("record") == "dispatch" for r in recs)
            o = sum(isinstance(r, dict) and r.get("record") in ("returned", "raised") for r in recs)
            out.update(dispatchedCalls=d, unresolvedCalls=d - o)
        pending, received, partial = [], [], 0
        for name, raw in self.a.raw.items():
            if name == "persistence-failure.json":
                try:
                    value = _strict_json(raw)
                    received.extend(x for x in value.get("receivedResponses") or [] if isinstance(x, dict))
                    partial += isinstance(value.get("partialDraw"), dict)
                except (ValueError, TypeError, AttributeError):
                    pass
            if name == "pending-draw.json" or name.startswith((".tmp-", ".failure-")) or name == "persistence-failure.json":
                try:
                    value = _strict_json(raw)
                    candidates = value if isinstance(value, list) else [value.get("draw"), value.get("pending")]
                    pending.extend(x for x in candidates if isinstance(x, dict) and x.get("status") == "completed")
                except (ValueError, TypeError, AttributeError):
                    pass
        out["pendingCompleted"] = len({sha256(canon(r)) for r in pending})
        out["receivedResponses"] = len(received)
        out["partialDraws"] = partial
        return out

    def _field_types(self):
        roots = {"run": [self.doc], "draw": self.rows,
                 "aborted": [self.aborted] if self.aborted else [],
                 "journal": self.doc.get("journal", []), "callJournal": self.records,
                 "calls": [c for r in self.rows + ([self.aborted] if self.aborted else [])
                           for c in (r.get("calls") or [])],
                 "record": [r["record"] for r in self.rows + ([self.aborted] if self.aborted else []) if r.get("record")],
                 "process": [r["process"] for r in self.rows + ([self.aborted] if self.aborted else []) if r.get("process")],
                 "healthChecks": self.doc.get("healthChecks", []),
                 "health": [c["serverState"]["health"] for c in self.doc.get("healthChecks", []) if c.get("serverState")],
                 "serverState": [c["serverState"] for c in self.doc.get("healthChecks", []) if c.get("serverState")],
                 "stop": [self.doc["stop"]] if self.doc.get("stop") else [],
                 "halt": [r["halt"] for r in self.rows + ([self.aborted] if self.aborted else []) if r.get("halt")],
                 "interrupts": [self.doc.get("interrupts", {})], "cleanup": [self.doc.get("cleanup", {})],
                 "failureEvent": [self.doc["failureEvent"]] if self.doc.get("failureEvent") else [],
                 "SEAL": [self.seal] if self.seal else []}
        def descend(value, parts):
            if not parts:
                return [value]
            if isinstance(value, list):
                return [v for x in value for v in descend(x, parts)]
            if not isinstance(value, dict) or parts[0] not in value:
                return []  # required keys are checked by the closed stage/event shapes
            return descend(value[parts[0]], parts[1:])
        for binding in self.bindings.values():
            for field, desc in binding["fieldDetails"].items():
                prefix, _, rest = field.partition(".")
                for obj in roots.get(prefix, []):
                    for v in descend(obj, rest.split(".") if rest else []):
                        if v is None and desc["nullable"]:
                            continue
                        t = desc["type"]
                        ok = {"string": lambda: isinstance(v, str), "integer": lambda: _nat(v),
                              "boolean": lambda: isinstance(v, bool), "object": lambda: isinstance(v, dict),
                              "array": lambda: isinstance(v, list),
                              "sha256": lambda: isinstance(v, str) and re.fullmatch(DATA.rule("B-CALL-002")["pattern"], v) is not None,
                              "sha256 array": lambda: isinstance(v, list) and all(isinstance(x, str) and re.fullmatch(DATA.rule("B-CALL-002")["pattern"], x) is not None for x in v),
                              "UTC timestamp": lambda: _timestamp(v) is not None,
                              "finite nonnegative number": lambda: isinstance(v, (float, int)) and not isinstance(v, bool) and math.isfinite(v) and v >= 0,
                              "domain JSON": lambda: True}[t]()
                        _need(ok, binding["id"], field + " has invalid type or null")

    # -- files (L16, S01) -----------------------------------------------------------------

    def _load(self):
        a = self.a
        _need(a.files is not None, "L16", "no archive directory")
        _need(not a.dirs, "L16", "unexpected entries that are not regular files: " + ", ".join(a.dirs))
        stray = set(a.files) - set(self.spec["allowedArchiveFiles"])
        _need("run.json" in a.raw, "L16", "no run manifest")
        _need("draws.json" in a.raw, "L16", "no authoritative ledger")
        _need(CALLS in a.raw, "L16", "no call journal")
        try:
            self.doc = a.json("run.json")
            self.rows = a.json("draws.json")
            self.records = self._journal_records(a.raw[CALLS])
            side = {n: a.json(n) for n in SIDE if n in a.raw}
            self.seal = a.json(SEAL) if SEAL in a.raw else None
        except (ValueError, UnicodeDecodeError) as x:
            raise Invalid("L16", f"unreadable archive file: {x}")
        for n, v in side.items():
            _need(v is not None, "L16", f"{n} is present but JSON null; absence is required to be absence")
            _need(isinstance(v, dict), "L16", f"{n} is not a JSON object")
        self.aborted = side.get("aborted-draw.json")
        self.event_file = side.get("failure-event.json")
        self.before = side.get("server-before.json")
        _need(not stray, "L16", "unexpected archive files: " + ", ".join(sorted(stray)))
        _need(isinstance(self.doc, dict) and set(self.doc) == RUN_KEYS, "L15", "run manifest fields")
        _need(isinstance(self.rows, list) and all(isinstance(r, dict) for r in self.rows), "L01", "ledger is not a list of rows")
        self.journal = self.doc["journal"]
        _need(isinstance(self.journal, list) and self.journal and all(isinstance(e, dict) for e in self.journal),
              "L14", "journal missing")
        if SEAL not in a.raw:
            self.violations.append({"invariant": "S01", "message": "no completion seal: the run did not reach its "
                                                                   "commit point; it is not COMPLETE or PARTIAL"})
        else:
            _need(isinstance(self.seal, dict), "L16", "SEAL.json is present but not a JSON object (JSON null is not absence)")

    def _journal_records(self, raw):
        _need(raw == b"" or raw.endswith(b"\n"), "L04", "the call journal ends in a torn record")
        recs = []
        for line in raw.split(b"\n")[:-1] if raw else []:
            r = _strict_json(line)
            _need(isinstance(r, dict), "L04", "a call-journal record is not an object")
            again = (json.dumps(r, sort_keys=True, separators=(",", ":"), ensure_ascii=True)).encode("ascii")
            _need(again == line, "L04", "a call-journal record is not in canonical form")
            recs.append(r)
        return recs

    # -- identity (L15, L18) ------------------------------------------------------------

    def _identity(self):
        d = self.doc
        _need(d["schema"] == RUN_SCHEMA and d["driver"] == DRIVER, "L15", "run schema or driver")
        _need(d["lifecycle"] == {"id": DATA.LIFECYCLE_ID, "sha256": DATA.spec_sha256()}, "L15", "lifecycle identity")
        _need(d["protocol"] == self.proto["id"] and d["protocolSha256"] == self.proto_sha, "L15", "protocol identity")
        _need(d["scheduleSha256"] == sha256(canon(self.sched)), "L15", "schedule identity")
        prov = d["provenance"]
        _need(isinstance(prov, dict) and set(prov) == {"timestamps", "runId"}
              and prov["timestamps"] in ("SYSTEM_UTC", "SYNTHETIC_FIXTURE") and prov["runId"] in ("UUID4", "SYNTHETIC_FIXTURE"),
              "L15", "provenance")
        _need("SYNTHETIC_FIXTURE" not in prov.values() or self.allow_synthetic, "L13",
              "synthetic provenance outside an explicit fixture check")
        try:
            u = uuid.UUID(d["runId"])
        except (ValueError, TypeError, AttributeError):
            raise Invalid("L15", "run identity is not a UUID")
        _need(str(u) == d["runId"] and (prov["runId"] != "UUID4" or u.version == 4), "L15", "run identity form")
        _need(isinstance(d["authorization"], str) and d["authorization"].strip(), "L15", "authorization")
        _need(isinstance(d["serverObserved"], bool), "L15", "serverObserved")
        _need(d["resume"] == "never", "L18", "the manifest permits resumption")
        _need(d["verificationStatus"] == "UNVERIFIED", "L15", "the recording claims verification")

    # -- journal and transition predicates (L14, L17, L19) --------------------------------

    def _needs_check(self, i):
        if not self.doc["serverObserved"]:
            return False
        s = self.sched
        return i == 0 or (s[i]["model"], s[i]["draw"]) != (s[i - 1]["model"], s[i - 1]["draw"])

    def _journal(self):
        states, events = self.spec["states"], self.spec["events"]
        sealed_only = set(self.spec["sealOnlyEvents"])
        prev = None
        for k, e in enumerate(self.journal):
            ev = e.get("event")
            _need(ev in events and ev not in sealed_only, "L14", f"unknown or seal-only event {ev!r} in the journal")
            _need(set(e) == {"n", "event", "from", "to", "timestamp"} | DETAIL[ev], "L14", f"journal entry {k + 1} fields")
            _need(_same(e["n"], k + 1), "L14", "journal order")
            _need(e["from"] is None or e["from"] in states, "L14", f"unknown state {e['from']!r}")
            _need(e["from"] == prev, "L14", f"journal does not chain at entry {k + 1}")
            _need((e["from"], ev) in self.table and self.table[(e["from"], ev)] == e["to"], "L14",
                  f"illegal transition {e['from']} --{ev}--> {e['to']}")
            prev = e["to"]
        _need(self.journal[0]["event"] == "create", "L14", "the journal does not start with create")
        _need(prev == "FINALIZING", "L14", f"the manifest journal ends in {prev}, not FINALIZING (unfinalized run)")
        _need(self.doc["state"] == prev, "L14", "claimed state disagrees with the journal")
        self.events = [e["event"] for e in self.journal]
        stops = [k for k, ev in enumerate(self.events) if ev == "stop"]
        _need(len(stops) <= 1, "L19", "more than one stop")
        self.stop_at = stops[0] if stops else None
        # Transition predicates (contract transitionPredicates), recomputed from the journal and ledger.
        committed, served, rows = 0, set(), self.rows
        observed = self.doc["serverObserved"]
        for k, e in enumerate(self.journal):
            ev, before = e["event"], (self.journal[k - 1]["event"] if k else None)
            if ev == "begin_health_check":
                _need(observed and committed < len(self.sched) and self._needs_check(committed)
                      and committed not in served, "L14",
                      "a round health check where the schedule requires none, or a second visit for the same draw")
                served.add(committed)
            elif ev == "begin_draw":
                _need(committed < len(self.sched) and _same(e["draw"], self.sched[committed]["seq"]), "L17",
                      f"begin_draw {e['draw']!r} is not the next scheduled draw")
                want = "health_passed" if self._needs_check(committed) else ("preflight_passed" if committed == 0 else "continue")
                _need(before == want, "L14", f"draw {e['draw']} started after {before}, the schedule requires {want}")
                _need(not self._needs_check(committed) or committed in served, "L14", "draw started without its health visit")
            elif ev == "commit_draw":
                _need(before == "begin_draw" and _same(self.journal[k - 1]["draw"], e["draw"]), "L17",
                      "commit_draw does not follow its own begin_draw")
                committed += 1
                _need(committed <= len(rows), "L17", "journal commits beyond the ledger")
                nxt = self.journal[k + 1]["event"] if k + 1 < len(self.journal) else None
                _need(e["reconciled"] is False or nxt == "stop", "L17", "a reconciled commit not followed by its stop")
            elif ev in ("continue", "schedule_done"):
                _need(before == "commit_draw" and committed >= 1 and rows[committed - 1]["status"] == "completed", "L14",
                      f"{ev} after a row that did not complete")
                _need((committed == len(self.sched)) == (ev == "schedule_done"), "L14",
                      "schedule_done before the last scheduled draw" if ev == "schedule_done" else
                      "continue after the last scheduled draw")
            elif ev == "preflight_passed":
                _need(before == "begin_preflight", "L14", "preflight_passed without its preflight")
        self.committed_in_journal = committed
        _need(committed == len(rows), "L17", "journal commits disagree with the ledger")

    # -- ledger (L01, L02, L17) ----------------------------------------------------------

    def _ledger(self):
        rows, sched = self.rows, self.sched
        _need(len(rows) <= len(sched), "L01", "more rows than scheduled")
        seen = set()
        for k, r in enumerate(rows):
            _need(isinstance(r, dict) and set(r) == DRAW_KEYS, "L02", f"row {k + 1} fields")
            _need(r["schema"] == H.DRAW_SCHEMA and _same({f: r[f] for f in IDENTITY}, sched[k]), "L01",
                  f"row {k + 1} is not the schedule's draw {sched[k]['seq']}")
            _need(r["seq"] not in seen, "L01", "duplicate row")
            seen.add(r["seq"])
            _need(r["status"] in ("completed", "interrupted"), "L01", f"row status {r['status']!r}")
            _need(r["status"] == "completed" or k == len(rows) - 1, "L01", "an unfinished row before the last")
            _need(r["engineInitialization"] == "READY" and r["failureStage"] is None and r["verificationStatus"] == "RECORDED",
                  "L02", f"row {r['seq']} initialization or verification status")
            _need(isinstance(r["calls"], list) and _same(r["modelCallsCompleted"], len(r["calls"])), "L04", "row call count")
            self._engine_log(r["engine"], r, failed_allowed=False)
            _need(isinstance(r["initialDigests"], dict) and isinstance(r["digestsAfter"], dict)
                  and isinstance(r["engine"], list) and r["engine"], "L02", f"row {r['seq']} engine evidence")
            c = self.contexts.get(r["context"])
            if c is not None:
                _need(_same(r["initialDigests"], c["digests"]), "L02", f"row {r['seq']} initial digests")
            if r["status"] == "completed":
                rep = r["submittedRepresentation"]
                _need(r["halt"] is None and isinstance(r["record"], dict) and isinstance(r["batches"], list)
                      and isinstance(rep, str) and sha256(rep) == r["submittedRepresentationSha256"]
                      and self._parses_to(rep, r["batches"]), "L02", f"completed row {r['seq']} evidence")
                _need(_same(r["process"], {f: r["record"][f] for f in PROCESS_KEYS}), "L02",
                      f"completed row {r['seq']} process evidence disagrees with its record")
            else:
                h = r["halt"]
                _need(isinstance(h, dict) and set(h) == {"code", "message", "evidence"} and h["code"] in HALTS
                      and r["batches"] is None and r["record"] is None and r["submittedRepresentation"] is None
                      and r["submittedRepresentationSha256"] is None, "L02", f"interrupted row {r['seq']} evidence")
                _need(_same(r["process"], self._derived_process(r, r["process"])), "L02",
                      f"interrupted row {r['seq']} process evidence")
                prior = [c for x in rows[:k] for c in x["calls"]]
                self._halt(h, r["calls"], prior + r["calls"], "draw")
        commits = [e for e in self.journal if e["event"] == "commit_draw"]
        for k, e in enumerate(commits[:len(rows)]):
            r = rows[k]
            _need(_same(e["draw"], r["seq"]) and _same(e["status"], r["status"]) and e["rowSha256"] == sha256(canon(r))
                  and _same(e["modelCalls"], len(r["calls"])) and isinstance(e["reconciled"], bool), "L17",
                  f"commit {k + 1} does not bind its ledger row")
        begins = [e for e in self.journal if e["event"] == "begin_draw"]
        from_active = self.stop_at is not None and self.journal[self.stop_at]["from"] == "DRAW_ACTIVE"
        _need(len(begins) == len(rows) + int(from_active), "L17", "begun draws disagree with committed rows")
        self.completed = sum(r["status"] == "completed" for r in rows)
        self.interrupted = rows[-1]["seq"] if rows and rows[-1]["status"] == "interrupted" else None

    def _derived_process(self, r, recorded):
        """run_draw's process evidence for a draw without a record, recomputed
        from the artifact's own engine log and calls (pendingMessage, the last
        request the model made, is checked for type only)."""
        engine, calls = r["engine"] or [], r["calls"] or []
        turns = [e["answer"].get("result") for e in engine if isinstance(e, dict) and e.get("op") == "alternatives.v3.turn"
                 and isinstance(e.get("answer"), dict) and e["answer"].get("ok")]
        turn = turns[-1] if turns else None
        correction = any("PHASE 2 validation correction" in (call.get("request", {}).get("messages", [{}])[-1].get("content") or "")
                         for call in calls)
        want = {"eligibility": {"eligible": turn.get("due") if turn else None,
                                "reason": turn.get("why") if turn else "UNVERIFIED before comparison"},
                "exposure": bool(turn and turn.get("message") and any(turn["message"] == m.get("content") for call in calls
                                                                      for m in call.get("request", {}).get("messages", []))),
                "validationFirstCorrection": correction, "correctionFailure": None if correction else False,
                "deliberateNoAction": False, "processFailure": True, "intendedAction": None, "validatedAction": None,
                "submittedAction": None, "executedAction": {"status": "NOT_EXECUTED"}}
        pending = recorded.get("pendingMessage") if isinstance(recorded, dict) else None
        _need(pending is None or isinstance(pending, str), "L06", "pending message evidence")
        return dict(want, pendingMessage=pending)

    # -- call journal (L04) ------------------------------------------------------------

    def _calls(self):
        recs = self.records
        self.dispatches, self.by_draw, self.unresolved = [], collections.defaultdict(list), []
        begun = [e["draw"] for e in self.journal if e["event"] == "begin_draw"]
        k, i, last_draw_index = 0, 0, -1
        while i < len(recs):
            d = recs[i]
            _need(d.get("record") == "dispatch" and set(d) == RECORD_FIELDS["dispatch"] and d["schema"] == CALLS_SCHEMA,
                  "L04", f"call-journal record {i + 1} is not a dispatch intent")
            k += 1
            _need(_same(d["k"], k) and _nat(d["n"]) and d["n"] >= 1, "L04", f"dispatch {k} numbering")
            _need(_nat(d["draw"]) and _member(d["draw"], begun), "L04", f"dispatch {k} names draw {d['draw']!r}, which was never begun")
            di = next(i for i, b in enumerate(begun) if _same(b, d["draw"]))
            _need(di >= last_draw_index, "L04", f"dispatch {k} returns to an earlier draw")
            prior = self.by_draw[d["draw"]]
            _need(_same(d["n"], len(prior) + 1), "L04", f"dispatch {k} is not call {len(prior) + 1} of draw {d['draw']}")
            last_draw_index = di
            o = recs[i + 1] if i + 1 < len(recs) else None
            if o is None or o.get("record") == "dispatch":
                self.unresolved.append(d)
                _need(o is None, "L04", f"dispatch {k} has no outcome before the next dispatch")
                prior.append({"dispatch": d, "outcome": None})
                i += 1
                continue
            kind = o.get("record")
            _need(kind in ("returned", "raised") and set(o) == RECORD_FIELDS[kind] and o["schema"] == CALLS_SCHEMA
                  and _same(o["k"], k) and _same(o["draw"], d["draw"]) and _same(o["n"], d["n"]), "L04",
                  f"the outcome of dispatch {k} does not bind it")
            _need(isinstance(o["requestSha256"], str) and re.fullmatch(
                      DATA.rule("B-CALL-002")["pattern"], o["requestSha256"]) is not None, "L04", "request hash")
            if kind == "returned":
                _need(isinstance(o["responseSha256"], str) and re.fullmatch(
                    DATA.rule("B-CALL-002")["pattern"], o["responseSha256"]) is not None,
                    "B-CALL-002", "response hash grammar")
                _need((o["body"] is None or isinstance(o["body"], str)) and o["responseSha256"] == sha256(o["body"] or "")
                      and (o["status"] is None or _nat(o["status"])) and (o["error"] is None or isinstance(o["error"], str)),
                      "L04", f"returned outcome {k} evidence")
                if o["status"] is None:
                    rule = DATA.rule("B-CALL-002")["noStatus"]
                    _need(o["body"] in rule["body"] and isinstance(o["error"], str) and bool(o["error"]),
                          "B-CALL-002", "a response without HTTP status needs a transport error and no response body")
            else:
                _need(_failure_shape(o["exception"]), "L04", f"raised outcome {k} evidence")
            prior.append({"dispatch": d, "outcome": o})
            self.dispatches.append(d)
            i += 2
        self.dispatched = k
        if self.seal is not None:
            _need(not self.unresolved, "L04", f"{len(self.unresolved)} dispatched call(s) without an outcome in a sealed archive")
        # Each draw's resolved entries equal its recorded calls one to one.
        observed = {r["seq"]: r for r in self.rows}
        if self.aborted is not None:
            observed[self.aborted.get("seq")] = self.aborted
        for draw, entries in self.by_draw.items():
            r = observed.get(draw)
            _need(r is not None, "L04", f"draw {draw} dispatched {len(entries)} call(s) that no draw evidence keeps")
            if r is self.aborted and self.aborted.get("failureStage") == "draw_unrecorded":
                continue  # the journal is the only call evidence; checked in _aborted
            calls = r.get("calls")
            _need(isinstance(calls, list) and len(calls) == len(entries), "L04",
                  f"draw {draw}: {len(entries)} dispatched call(s) but {len(calls) if isinstance(calls, list) else calls!r} recorded")
            for e, call in zip(entries, calls):
                o = e["outcome"]
                _need(o is not None, "L04", f"draw {draw}: a recorded call whose dispatch is unresolved")
                _need(_same(call.get("n"), o["n"]) and _same(call.get("requestSha256"), o["requestSha256"]), "R-DISPATCH-001",
                      f"draw {draw} call {o['n']}: request disagrees with the call journal")
                if o["record"] == "returned":
                    _need(all(f in call and _same(call[f], o[f]) for f in ("status", "error", "body", "responseSha256")),
                          "R-RESPONSE-001", f"draw {draw} call {o['n']}: response disagrees with the call journal")
                else:
                    x = o["exception"]
                    _need(call.get("status") is None and call.get("body") is None
                          and _same(call.get("error"), f"client: {x['type']}: {x['message']}")
                          and call.get("responseSha256") == sha256(""), "R-RESPONSE-002",
                          f"draw {draw} call {o['n']}: a raised dispatch recorded differently")
        for seq, r in observed.items():
            calls = r.get("calls")
            if isinstance(calls, list) and calls:
                _need(seq in self.by_draw, "L04", f"draw {seq} records calls the call journal never dispatched")
        for draw, entries in self.by_draw.items():
            kinds = [e["outcome"]["record"] for e in entries if e["outcome"] is not None]
            if "raised" in kinds:
                _need(kinds.index("raised") == len(entries) - 1 and observed.get(draw) is self.aborted, "R-RESPONSE-002",
                      f"draw {draw}: a raised transport outcome that is not the aborted draw's last call")

    # -- aborted draw (L05, L06) -------------------------------------------------------------

    def _aborted(self):
        from_active = self.stop_at is not None and self.journal[self.stop_at]["from"] == "DRAW_ACTIVE"
        ab = self.aborted
        _need((ab is not None) == from_active, "L05", "aborted artifact present iff the stop is from DRAW_ACTIVE")
        if ab is None:
            return
        stages = self.spec["abortedDraw"]["stages"]
        stage = stages.get(ab.get("failureStage"))
        _need(stage is not None, "L06", f"unknown aborted failure stage {ab.get('failureStage')!r}")
        keys = DRAW_KEYS | {"failure", "stopReason"}
        _need(set(ab) - {"cleanupFailure"} == keys, "L05", "aborted artifact fields")
        _need(ab["schema"] == H.DRAW_SCHEMA and ab["status"] == "aborted" and ab["verificationStatus"] == "UNVERIFIED",
              "L05", "aborted artifact status")
        _need(len(self.rows) < len(self.sched) and _same({f: ab[f] for f in IDENTITY}, self.sched[len(self.rows)]), "L05",
              "aborted draw is not the next scheduled draw")
        _need(ab["engineInitialization"] in stage["engineInitialization"], "L06",
              f"failure stage {ab['failureStage']} contradicts initialization {ab['engineInitialization']}")
        _need(_failure_shape(ab["failure"]), "L06", "aborted failure evidence")
        stop = self.doc["stop"] if isinstance(self.doc["stop"], dict) else {}
        _need(_same(ab["stopReason"], stop.get("code")), "R-FAILURE-001", "aborted stopReason disagrees with the stop")
        operator = ab["failure"]["type"] == "OperatorStop"
        _need(operator == (stop.get("code") == "operator"), "L06", "OperatorStop failure iff an operator stop")
        _need(not operator or ab["failureStage"] == "workflow", "L06", "an operator stop outside the workflow stage")
        digests = (self.contexts.get(ab["context"]) or {}).get("digests")
        for field, rule in stage["fields"].items():
            self._field(ab, field, rule, digests)
        if stage["journalCalls"] == "none":
            _need(not self.by_draw.get(ab["seq"]), "L04", f"{ab['failureStage']} failure with dispatched calls")
        if isinstance(ab["engine"], list):
            self._engine_log(ab["engine"], ab, failed_allowed=True)
        self._raised_binding(ab)
        if ab["failureStage"] in ("workflow", "final_digests", "engine_close"):
            e0 = ab["engine"][0]
            _need(isinstance(e0, dict) and e0.get("op") == "digests" and isinstance(e0.get("answer"), dict)
                  and e0["answer"].get("ok") is True, "L06", "a READY draw without its initial digests request")
        if ab["failureStage"] in ("final_digests", "engine_close"):
            _need((ab["record"] is None) != (ab["halt"] is None), "B-ABORT-001",
                  "a finished workflow requires exactly one record or halt")
            if ab["record"] is not None:
                _need(self._parses_to(ab["submittedRepresentation"], ab["record"]["batches"]), "B-BATCH-001",
                      "aborted submission differs from completed workflow")
            else:
                prior = [c for x in self.rows for c in x["calls"]]
                self._halt(ab["halt"], ab["calls"], prior + ab["calls"], "draw")
        if ab["failureStage"] == "engine_close":
            _need(ab["engine"][-1].get("op") == "digests" and
                  _same(ab["engine"][-1].get("answer", {}).get("result", {}).get("digests"), ab["digestsAfter"]),
                  "B-ENGINE-002", "final digest differs from completed engine response")
        if ab["failureStage"] == "final_digests":
            _need(isinstance(ab["engine"][-1], dict) and ab["engine"][-1].get("op") == "digests", "L06",
                  "a final-digests failure whose last engine request is not the digests request")

    def _field(self, ab, field, rule, digests):
        present = field in ab
        v = ab.get(field)
        record = ab.get("record")
        msg = f"{ab['failureStage']}: {field} must be {rule}"
        if rule == "forbidden":
            _need(not present, "L06", msg)
            return
        if rule == "optionalFailure":
            _need(not present or _failure_shape(v), "L08", msg)
            return
        _need(present, "L06", msg)
        if rule == "null":
            ok = v is None
        elif rule == 0:
            ok = _same(v, 0)
        elif rule == "empty":
            ok = _same(v, [])
        elif rule == "nonEmpty":
            ok = isinstance(v, list) and len(v) > 0
        elif rule == "object":
            ok = isinstance(v, dict)
        elif rule == "context":
            ok = isinstance(v, dict) and (digests is None or _same(v, digests))
        elif rule == "recordedCalls":
            ok = isinstance(v, list)
        elif rule == "len(calls)":
            ok = isinstance(ab.get("calls"), list) and _same(v, len(ab["calls"]))
        elif rule == "one digests request":
            ok = isinstance(v, list) and len(v) == 1 and isinstance(v[0], dict) and v[0].get("op") == "digests"
        elif rule == "noModelProcess":
            ok = _same(v, self.spec["abortedDraw"]["noModelProcess"])
        elif rule == "derivedProcess":
            ok = _same(v, self._derived_process(ab, v))
        elif rule == "null, or a halt (then record null)":
            ok = v is None or (isinstance(v, dict) and set(v) == {"code", "message", "evidence"} and v["code"] in HALTS
                               and record is None)
        elif rule == "null, or an object (then halt null)":
            ok = v is None or (isinstance(v, dict) and ab.get("halt") is None)
        elif rule == "string iff record":
            ok = isinstance(v, str) if record is not None else v is None
        elif rule == "sha256 iff record":
            ok = (isinstance(ab.get("submittedRepresentation"), str) and v == sha256(ab["submittedRepresentation"])) \
                if record is not None else v is None
        elif rule == "list iff record":
            ok = (isinstance(v, list) and self._parses_to(ab["submittedRepresentation"], v)) if record is not None else v is None
        elif rule == "recordProcess iff record, else derivedProcess":
            ok = _same(v, {f: record[f] for f in PROCESS_KEYS}) if isinstance(record, dict) else _same(v, self._derived_process(ab, v))
        else:
            raise Invalid("L06", f"the verifier does not support the contract rule {rule!r} for {field}")
        _need(ok, "L06", msg)

    @staticmethod
    def _parses_to(text, value):
        try:
            return isinstance(text, str) and _same(_strict_json(text), value)
        except ValueError:
            return False

    def _engine_log(self, engine, r, failed_allowed):
        """R-FAILURE-002: every engine entry is a closed successful request, or
        (only in an aborted draw, only as the last entry) a closed failed
        request whose error is the draw's failure."""
        for i, e in enumerate(engine):
            _need(isinstance(e, dict) and isinstance(e.get("op"), str) and e["op"], "R-FAILURE-002",
                  f"draw {r['seq']} engine entry {i + 1} is not a request")
            if set(e) == {"op", "args", "answer"} and isinstance(e["answer"], dict):
                continue
            _need(failed_allowed and set(e) == {"op", "args", "answer", "error"} and e["answer"] is None
                  and isinstance(e["error"], str) and e["error"], "R-FAILURE-002",
                  f"draw {r['seq']} engine entry {i + 1} is neither a closed answered request nor a closed failed request")
            _need(i == len(engine) - 1, "R-FAILURE-002",
                  f"draw {r['seq']}: a failed engine request is not the last engine event (the failure ends the draw)")
            f = r.get("failure")
            _need(_failure_shape(f) and _same(e["error"], f"{f['type']}: {f['message']}"), "R-FAILURE-002",
                  f"draw {r['seq']}: the failed engine request's error disagrees with the draw failure")
            stage = r.get("failureStage")
            _need(stage in ("initial_digests", "workflow", "final_digests"), "R-FAILURE-002",
                  f"draw {r['seq']}: a failed engine request in failure stage {stage!r}")
            if stage in ("initial_digests", "final_digests"):
                _need(e["op"] == "digests", "R-FAILURE-002", f"draw {r['seq']}: {stage} failed on {e['op']!r}")

    def _raised_binding(self, ab):
        """R-RESPONSE-002: a raised transport outcome is the draw's failure."""
        entries = self.by_draw.get(ab["seq"], [])
        raised = [e["outcome"] for e in entries if e["outcome"] is not None and e["outcome"]["record"] == "raised"]
        if not raised or ab["failureStage"] == "draw_unrecorded":
            return
        x = raised[-1]["exception"]
        _need(_same(ab["failure"], {"type": x["type"], "message": x["message"]}), "R-RESPONSE-002",
              "the raised transport exception disagrees with the aborted draw failure")
        _need(ab["failureStage"] == "workflow", "R-RESPONSE-002", "a raised transport outcome outside the workflow stage")

    def _streak_of(self, calls):
        n = 0
        for c in calls:
            n = n + 1 if c.get("status") is None and c.get("error") else 0
        return n

    def _halt(self, halt, draw_calls, run_calls, where):
        """R-HALT-001: reconstruct a halt from the retained calls, the protocol's
        limits and the workflow phase, and compare it with the archived claim.
        Only elapsedSeconds (the wall clock) is an external measurement."""
        p, stop = self.proto, self.proto["stop"]
        _need(isinstance(halt, dict) and set(halt) == {"code", "message", "evidence"}, "R-HALT-001", "halt fields")
        ev = halt["evidence"]
        keys = {"drawCalls", "runCalls", "elapsedSeconds", "consecutiveTransportFailures", "stage"}
        _need(isinstance(ev, dict) and set(ev) == keys, "R-HALT-001",
              "halt evidence fields (missing: " + ", ".join(sorted(keys - set(ev if isinstance(ev, dict) else {}))) + ")")
        stage = "request_start" if where == "draw" else "draw_start"
        _need(_same(ev["stage"], stage), "R-HALT-001",
              f"a halt {'inside a draw' if where == 'draw' else 'at the draw guard'} must be at {stage}, not {ev['stage']!r}")
        streak = self._streak_of(run_calls)
        _need(_same(ev["drawCalls"], len(draw_calls)), "R-HALT-001",
              f"halt drawCalls {ev['drawCalls']!r} but the draw retains {len(draw_calls)} call(s)")
        _need(_same(ev["runCalls"], len(run_calls)), "R-HALT-001",
              f"halt runCalls {ev['runCalls']!r} but the run retains {len(run_calls)} call(s) before it")
        _need(_same(ev["consecutiveTransportFailures"], streak), "R-HALT-001",
              f"halt claims {ev['consecutiveTransportFailures']!r} consecutive transport failures; the retained calls give {streak}")
        e = ev["elapsedSeconds"]
        _need(type(e) in (int, float) and math.isfinite(e) and e >= 0, "R-HALT-001", "elapsed-time evidence")
        late = e + p["request"]["timeoutSeconds"] > stop["wallClockSeconds"]
        if where == "draw":
            want = ("transport" if streak >= stop["consecutiveTransportFailures"] else
                    "draw_call_limit" if len(draw_calls) >= p["maxModelCallsPerDraw"] else
                    "call_budget" if len(run_calls) >= p["maxModelCalls"] else "wall_clock" if late else None)
        else:
            want = "wall_clock" if late and not draw_calls else None
        _need(want is not None, "R-HALT-001", "no limit fires for the reconstructed evidence; the halt is impossible")
        _need(_same(halt["code"], want), "R-HALT-001", f"the halt claims {halt['code']!r}; the evidence gives {want}")
        msg = halt["message"]
        if want == "transport":
            ok = _same(msg, f"{streak} consecutive transport failures")
        elif want == "draw_call_limit":
            ok = _same(msg, f"a draw asked for model call {len(draw_calls) + 1}; the limit is {p['maxModelCallsPerDraw']}")
        elif want == "call_budget":
            ok = _same(msg, f"{len(run_calls)} model calls made; the limit is {p['maxModelCalls']}")
        else:
            tail = (f" s elapsed; a request of up to {p['request']['timeoutSeconds']} s would pass the "
                    f"{stop['wallClockSeconds']} s stop" if where == "draw" else
                    f" s elapsed; the next draw could pass the {stop['wallClockSeconds']} s stop")
            m = re.fullmatch(r"(0|[1-9][0-9]*)" + re.escape(tail), msg, re.ASCII) if isinstance(msg, str) else None
            ok = m is not None and int(m.group(1)) <= e + 0.5
        _need(ok, "R-HALT-001", "the halt message is not its guard's message for the reconstructed evidence")

    @staticmethod
    def _stop_message(stop):
        """The recorder's stop message, derived from the stop's own code and fields."""
        code, f = stop["code"], stop["failure"]
        if code in HALTS:
            return None  # the halt's message, bound by R-HALT-001
        if code == "operator":
            return f"operator interrupt processed at safe point {stop['safePoint']}"
        if code == "health":
            return "the server did not answer healthy"
        if code == "server_change":
            return "server changed"
        return f"{f['type']}: {f['message']}" if _failure_shape(f) else None

    # -- health (L07) --------------------------------------------------------------------

    def observation_problem(self, obs):
        """The verifier's independent check of a kept observation."""
        h = self.spec["health"]
        keys = {"schema", "request", "started", "ended", "outcome", "category", "httpStatus", "exception", "body",
                "verificationStatus"}
        if not isinstance(obs, dict) or set(obs) != keys or obs["schema"] != HEALTH_SCHEMA or obs["request"] != h["request"]:
            return "not a health observation"
        if obs["verificationStatus"] != "UNVERIFIED":
            return "health observation claims verification"
        t0, t1 = _timestamp(obs["started"]), _timestamp(obs["ended"])
        if t0 is None or t1 is None or t1 < t0:
            return "health observation timestamps"
        cat, body, exc, status = obs["category"], obs["body"], obs["exception"], obs["httpStatus"]
        raw = None
        if body is not None:
            if not (isinstance(body, dict) and set(body) == {"bytes", "sha256", "base64"} and isinstance(body["base64"], str)):
                return "body evidence"
            try:
                raw = base64.b64decode(body["base64"], validate=True)
            except (ValueError, TypeError):
                return "body evidence"
            if not _same(body["bytes"], len(raw)) or hashlib.sha256(raw).hexdigest() != body["sha256"]:
                return "body evidence"
        if cat in h["bodyCategories"]:
            if exc is not None or raw is None or not (_nat(status) and 200 <= status < 300):
                return "a response category without a valid 2xx response"
            if _health_category(raw, h["maxBodyBytes"]) != cat:
                return "the recorded body does not classify as recorded"
            if obs["outcome"] != h["bodyCategories"][cat]:
                return "outcome disagrees with category"
            return None
        families = h["readFailureFamilies"]
        if cat not in families:
            return "unknown health category"
        if obs["outcome"] != "READ_FAILED" or not (isinstance(exc, dict) and set(exc) == {"type", "reasonType", "reasonKind"}):
            return "read failure without exception evidence"
        t, rt, rk = exc["type"], exc["reasonType"], exc["reasonKind"]
        if cat == "http_status":
            if (t, rt, rk) != ("urllib.error.HTTPError", None, None) or not _nat(status) or 200 <= status < 300:
                return "http_status without a non-2xx HTTPError"
            return None
        if status is not None or body is not None:
            return "a read failure with response evidence"
        if cat == "url_error":
            return None if (t, rt, rk) == ("urllib.error.URLError", None, "text") else "url_error evidence"
        fam = families[cat]
        if t == "urllib.error.URLError":
            if rk != "exception" or not fam["wrappable"] or rt not in fam["types"]:
                return "a wrapped exception outside the family its category names"
            return None
        if rk is not None or rt is not None or t not in fam["types"]:
            return "an exception type outside the family its category names"
        return None

    def _health(self):
        doc, checks = self.doc, self.doc["healthChecks"]
        _need(isinstance(checks, list), "L07", "health checks")
        observed = doc["serverObserved"]
        self.signature, self.visits = None, []
        _need(observed or self.allow_synthetic, "L07", "an ordinary run must observe the server's health")
        if not observed:
            _need(not checks and "begin_health_check" not in self.events and self.before is None, "L07",
                  "health evidence in a run that does not observe the server")
            return
        for k, c in enumerate(checks):
            _need(isinstance(c, dict) and set(c) == {"index", "state", "beforeDraw", "serverState", "rejected"}
                  and _same(c["index"], k) and c["state"] in ("PREFLIGHT", "HEALTH_CHECK"), "L07", f"health check {k} fields")
            _need(c["serverState"] is None or c["rejected"] is None, "L07", "a health check both kept and rejected")
            if c["serverState"] is not None:
                _need(isinstance(c["serverState"], dict), "L07", "server state")
                problem = self.observation_problem(c["serverState"].get("health"))
                _need(problem is None, "L07", f"health check {k}: {problem}")
            if c["rejected"] is not None:
                _need(isinstance(c["rejected"], dict) and set(c["rejected"]) == {"returned"}, "L07", "rejected evidence")
                ret = c["rejected"]["returned"]
                obs = ret.get("health") if isinstance(ret, dict) else None
                _need(self.observation_problem(obs) is not None, "L07", "a valid observation was rejected")
        stop = doc["stop"] if isinstance(doc["stop"], dict) else None
        committed, used = 0, 0
        for k, e in enumerate(self.journal):
            ev = e["event"]
            if ev == "commit_draw":
                committed += 1
            if ev not in ("begin_preflight", "begin_health_check"):
                continue
            nxt = self.journal[k + 1] if k + 1 < len(self.journal) else None
            _need(nxt is not None and nxt["event"] in ("preflight_passed", "health_passed", "stop"), "L07",
                  "health visit without an admissible exit")
            state = "PREFLIGHT" if ev == "begin_preflight" else "HEALTH_CHECK"
            if state == "HEALTH_CHECK":
                _need(_same(e["healthCheck"], used), "L07", "begin_health_check names the wrong check")
            if nxt["event"] in ("preflight_passed", "health_passed"):
                consumes = True
            else:  # a stop from this visit consumes a check iff it names one
                consumes = stop is not None and stop.get("healthCheck") is not None
            check = None
            if consumes:
                _need(used < len(checks) and checks[used]["state"] == state, "L07",
                      "a health visit without the check it requires")
                check = checks[used]
                _need(_same(check["beforeDraw"], None if state == "PREFLIGHT" else self.sched[committed]["seq"]), "L07",
                      "health check names the wrong draw")
                used += 1
            self.visits.append({"state": state, "check": check, "enter": k, "exit": k + 1, "draw": committed})
            if nxt["event"] in ("preflight_passed", "health_passed"):
                _need(check["serverState"] is not None and check["serverState"]["health"]["outcome"] == "HEALTHY", "L07",
                      "a passing health transition without a HEALTHY observation")
                if nxt["event"] == "health_passed":
                    _need(_same(nxt["healthCheck"], check["index"]), "L07", "health_passed names another check")
                    _need(X.server_signature(check["serverState"]) == self.signature, "L07",
                          "health passed although the server signature changed")
            if state == "PREFLIGHT" and check is not None and check["serverState"] is not None:
                self.signature = X.server_signature(check["serverState"])
        _need(used == len(checks), "L07", "health checks outside any health visit")
        if observed and "preflight_passed" in self.events:
            _need(self.visits and self.visits[0]["check"] is not None, "L07", "preflight passed without its health check")
        first = checks[0] if checks and checks[0]["state"] == "PREFLIGHT" else None
        if first is not None and first["serverState"] is not None:
            _need(_same(self.before, first["serverState"]), "L07", "server-before.json disagrees with the initial health check")
        else:
            _need(self.before is None, "L07", "server-before.json without an initial kept observation")

    # -- stop and failure event (L06, L08, L19) -----------------------------------------------

    def _calls_list(self):
        out = []
        for r in self.rows + ([self.aborted] if self.aborted else []):
            if isinstance(r.get("calls"), list):
                out.extend(r["calls"])
        return out

    def _streak(self, calls):
        n = 0
        for c in calls:
            n = n + 1 if c.get("status") is None and c.get("error") else 0
        return n

    def _stop(self):
        doc, stop = self.doc, self.doc["stop"]
        cleanup_created = self.stop_at is None and "cleanup_failed" in self.events and stop is not None
        self.cleanup_created = cleanup_created
        if self.stop_at is None and not cleanup_created:
            _need(stop is None and doc["failureEvent"] is None and self.event_file is None, "L19",
                  "stop evidence without a stop transition")
            return
        _need(isinstance(stop, dict) and set(stop) == STOP_KEYS and isinstance(stop["message"], str), "L19", "stop fields")
        code = stop["code"]
        rule = self.spec["stopCodes"].get(code)
        _need(rule is not None, "L06", f"unknown stop code {code!r}")
        if cleanup_created:
            entry_ts = self.journal[self.events.index("cleanup_failed")]["timestamp"]
            _need(stop["state"] == "CLEANUP" and code == "driver", "L08", "cleanup-created stop outside CLEANUP or not driver")
        else:
            e = self.journal[self.stop_at]
            entry_ts = e["timestamp"]
            _need(stop["code"] == e["code"] and stop["state"] == e["from"], "L19", "stop disagrees with the journal")
            _need(stop["state"] != "CLEANUP", "L06", "a stop transition from CLEANUP")
        _need(stop["state"] in rule["states"], "L06", f"stop {code} cannot come from {stop['state']}")
        derived = self._stop_message(stop)
        if code not in HALTS:
            _need(derived is not None and _same(stop["message"], derived), "R-FAILURE-001",
                  "the stop message is not the message its code and failure determine")
        visit = next((v for v in reversed(getattr(self, "visits", [])) if v["exit"] == self.stop_at), None)
        check = visit["check"] if visit else None
        req = self.spec["health"]["request"]
        calls = self._calls_list()
        nulls = lambda *fields: all(stop[f] is None for f in fields)  # noqa: E731
        if code == "health":
            _need(check is not None and check["serverState"] is not None
                  and check["serverState"]["health"]["outcome"] != "HEALTHY", "L07", "health stop without a failed observation")
            _need(nulls("failure", "evidence", "beforeSignature", "safePoint") and _same(stop["request"], req)
                  and _same(stop["healthCheck"], check["index"]), "L06", "health stop evidence")
        elif code == "server_change":
            _need(check is not None and check["serverState"] is not None
                  and check["serverState"]["health"]["outcome"] == "HEALTHY"
                  and X.server_signature(check["serverState"]) != self.signature
                  and _same(stop["beforeSignature"], self.signature), "L07", "server-change stop evidence")
            _need(nulls("failure", "evidence", "request", "safePoint") and _same(stop["healthCheck"], check["index"]), "L06",
                  "server-change stop fields")
        elif code in HALTS:
            _need(nulls("failure", "request", "healthCheck", "beforeSignature", "safePoint"), "L06", "halt stop fields")
            halt = {k: stop[k] for k in ("code", "message", "evidence")}
            if stop["state"] == "DRAW_RECORDED":
                _need(self.interrupted is not None and _same(halt, self.rows[-1]["halt"]), "R-HALT-001",
                      "halt differs from the interrupted row")
                self._halt(halt, self.rows[-1]["calls"], calls, "draw")
            elif stop["state"] == "DRAW_ACTIVE":
                _need(self.aborted is not None and _same(halt, self.aborted.get("halt")), "R-HALT-001",
                      "halt stop differs from the aborted draw's halt")
                self._halt(halt, self.aborted["calls"], calls, "draw")
            else:
                self._halt(halt, [], calls, "guard")
            _need(_same(stop["evidence"]["runCalls"], self.dispatched), "R-COUNT-001",
                  "halt call evidence disagrees with the call journal")
        elif code == "operator":
            sp = stop["safePoint"]
            point = self.spec["safePoints"]["points"].get(sp)
            _need(point is not None and point["kind"] == "STOP" and stop["state"] in point["states"], "L06",
                  f"operator stop at {sp!r} is not a STOP safe point of {stop['state']}")
            _need(nulls("failure", "evidence", "request", "beforeSignature"), "L06", "operator stop fields")
            if sp == "after_health_check":
                _need(check is not None and check["serverState"] is not None
                      and check["serverState"]["health"]["outcome"] == "HEALTHY"
                      and _same(stop["healthCheck"], check["index"]), "L07",
                      "an operator stop after a health check needs that check, kept and HEALTHY")
                if stop["state"] == "HEALTH_CHECK":
                    _need(X.server_signature(check["serverState"]) == self.signature, "L07",
                          "an operator stop after a changed server signature (server_change is primary)")
            else:
                _need(stop["healthCheck"] is None and check is None, "L06",
                      f"an operator stop at {sp} with a health check of the visit")
            if sp == "before_health_check" and stop["state"] == "READY":
                _need(self._needs_check(len(self.rows)), "L06",
                      "an operator stop before a round health check the schedule does not require")
            if sp == "before_dispatch":
                _need(self.aborted is not None and self.aborted["failure"]["type"] == "OperatorStop"
                      and self.aborted["failureStage"] == "workflow", "L06",
                      "an operator stop before dispatch needs the aborted workflow artifact")
            if sp == "after_draw_persisted":
                _need(self.rows and self.rows[-1]["status"] == "completed", "L06",
                      "an operator stop after a draw that did not complete")
        else:
            f = stop["failure"]
            _need(_failure_shape(f) and f["type"] != "OperatorStop", "L06", "driver stop failure evidence")
            _need(nulls("evidence", "beforeSignature", "safePoint"), "L06", "driver stop fields")
            attempted = check is not None and check["serverState"] is None and check["rejected"] is None
            _need(_same(stop["request"], req if attempted else None), "L07", "attempted health request disagrees")
            _need(_same(stop["healthCheck"], check["index"] if check is not None else None), "L07",
                  "stop names the wrong health check")
            if stop["state"] == "DRAW_ACTIVE":
                _need(self.aborted is not None and _same(f, self.aborted["failure"]), "R-FAILURE-001",
                      "primary draw failure disagrees with the aborted artifact")
            if stop["state"] == "DRAW_RECORDED":
                _need(self.rows and self.rows[-1]["status"] == "completed", "L06",
                      "an exception stop after an interrupted row (the halt is the stop)")
            if cleanup_created:
                _need(_same(f, doc["cleanup"]["failure"]) and _same(stop["message"], f"{f['type']}: {f['message']}")
                      and stop["request"] is None and stop["healthCheck"] is None, "L08",
                      "cleanup-created stop disagrees with the cleanup failure")
        if check is not None and check["rejected"] is not None:
            _need(code == "driver", "L07", "a rejected observation must stop the run as a driver failure")
        if self.interrupted is not None:
            _need(code in HALTS and stop["state"] == "DRAW_RECORDED", "L06", "an interrupted row needs its halt as the stop")
        nxt = ({k: self.aborted[k] for k in IDENTITY} if self.aborted else
               {k: self.rows[-1][k] for k in IDENTITY} if self.interrupted is not None else
               self.sched[len(self.rows)] if len(self.rows) < len(self.sched) else None)
        want = {"schema": FAILURE_SCHEMA, "runId": doc["runId"], "protocolSha256": doc["protocolSha256"],
                "scheduleSha256": doc["scheduleSha256"], "state": stop["state"], "stop": stop, "nextDraw": nxt,
                "drawsCompleted": self.completed, "modelCalls": self.dispatched, "timestamp": entry_ts,
                "verificationStatus": "UNVERIFIED"}
        _need(_same(doc["failureEvent"], want), "L19", "failure event disagrees with the derived stop")
        _need(_same(self.event_file, doc["failureEvent"]), "L16", "failure-event.json missing or disagrees")

    # -- cleanup (L08) -----------------------------------------------------------------

    def _cleanup(self):
        c = self.doc["cleanup"]
        _need(isinstance(c, dict) and set(c) == set(DATA.rule("B-CLEANUP-001")["definition"]["fields"]) and c["status"] in ("SUCCEEDED", "FAILED"),
              "L08", "cleanup evidence missing")
        _need((c["status"] == "FAILED") == ("cleanup_failed" in self.events)
              and (c["status"] == "SUCCEEDED") == ("cleanup_succeeded" in self.events), "L08", "cleanup disagrees with the journal")
        _need((c["failure"] is None) == (c["status"] == "SUCCEEDED") and (c["failure"] is None or _failure_shape(c["failure"])),
              "L08", "contradictory cleanup failure")
        ev = "cleanup_failed" if c["status"] == "FAILED" else "cleanup_succeeded"
        _need(self.events[-1] == ev and c["timestamp"] == self.journal[-1]["timestamp"], "L08",
              "cleanup is not the journal's last transition with its timestamp")
        if c["status"] == "FAILED" and self.stop_at is None:
            _need(isinstance(self.doc["stop"], dict) and self.doc["stop"].get("state") == "CLEANUP", "L08",
                  "cleanup failure without a stop")

        definition = DATA.rule("B-CLEANUP-001")["definition"]
        visits = c["events"]
        _need(isinstance(visits, list) and [v.get("safePoint") for v in visits] == definition["order"],
              "B-CLEANUP-001", "cleanup visits missing, duplicated or out of order")
        for n, v in enumerate(visits, 1):
            _need(set(v) == set(definition["eventFields"]) and _same(v["n"], n) and v["state"] == "CLEANUP",
                  "B-CLEANUP-001", "cleanup visit fields or stage")
        begin = _ts(c["started"], "cleanup start")
        completed = _ts(c["timestamp"], "cleanup completion")
        times = [_ts(v["timestamp"], "cleanup visit") for v in visits]
        entry = _ts(self.journal[-2]["timestamp"], "cleanup entry")
        seal_time = _ts(self.seal["transition"]["timestamp"], "seal timestamp") if self.seal else times[-1]
        _need(entry <= begin <= times[0] <= completed <= times[1] <= times[2] <= seal_time,
              "B-CLEANUP-001", "cleanup/finalization time interval contradiction")
        self.cleanup_visits = {v["safePoint"]: v for v in visits}

    # -- interrupts (L20) ------------------------------------------------------------------

    def _interrupts(self):
        it = self.doc["interrupts"]
        _need(isinstance(it, dict) and set(it) == {"signal", "events"} and it["signal"] == "SIGINT"
              and isinstance(it["events"], list), "L20", "interrupt record")
        points = self.spec["safePoints"]["points"]
        stop = self.doc["stop"] if isinstance(self.doc["stop"], dict) else None
        operator = stop is not None and stop.get("code") == "operator"
        stop_ts = self.journal[self.stop_at]["timestamp"] if self.stop_at is not None else None
        done = "schedule_done" in self.events
        done_ts = self.journal[self.events.index("schedule_done")]["timestamp"] if done else None
        cleanup_created = getattr(self, "cleanup_created", False)
        stops, prev = 0, None
        for k, e in enumerate(it["events"]):
            _need(isinstance(e, dict) and set(e) == {"n", "safePoint", "state", "signals", "disposition", "timestamp"}
                  and _same(e["n"], k + 1) and _nat(e["signals"]) and e["signals"] >= 1, "L20", f"interrupt event {k + 1} fields")
            t = _ts(e["timestamp"], f"interrupt event {k + 1}")
            _need(prev is None or prev <= t, "L20", "interrupt events out of order")
            prev = t
            p = points.get(e["safePoint"])
            _need(p is not None and e["state"] in p["states"], "L20", f"interrupt event {k + 1} at an undeclared point")
            if e["disposition"] == "STOP":
                stops += 1
                _need(operator and p["kind"] == "STOP" and e["safePoint"] == stop["safePoint"]
                      and e["state"] == stop["state"] and e["timestamp"] == stop_ts and k == 0, "L20",
                      "a STOP interrupt event that is not the operator stop")
            elif e["disposition"] == "AFTER_STOP":
                _need(stop is not None and p["kind"] == "RECORD" and e["state"] == "CLEANUP", "L20",
                      "an AFTER_STOP event without a prior stop, or outside CLEANUP")
                if cleanup_created:
                    _need(e["safePoint"] in ("after_cleanup", "before_finalization"), "L20",
                          "an AFTER_STOP event before the cleanup failure that created the stop")
                else:
                    _need(t >= _timestamp(stop_ts), "L20", "an AFTER_STOP event before its stop")
            elif e["disposition"] == "AFTER_SCHEDULE":
                _need(done and self.stop_at is None and p["kind"] == "RECORD" and e["state"] == "CLEANUP"
                      and t >= _timestamp(done_ts), "L20", "an AFTER_SCHEDULE event before schedule_done or after a stop")
                if cleanup_created:
                    _need(e["safePoint"] == "before_cleanup", "L20", "an AFTER_SCHEDULE event after the cleanup failure")
            else:
                raise Invalid("L20", f"unknown interrupt disposition {e['disposition']!r}")
        _need(stops == (1 if operator else 0), "L20", "an operator stop needs exactly one STOP interrupt event")
        order = DATA.rule("B-INTERRUPT-001")["order"]
        seen = -1
        for e in it["events"]:
            if e["disposition"] == "STOP":
                continue
            i = order.index(e["safePoint"])
            _need(i > seen, "B-INTERRUPT-001", "duplicate or reversed cleanup interrupt safe point")
            seen = i
            visit = self.cleanup_visits[e["safePoint"]]
            _need(e["timestamp"] == visit["timestamp"] and e["state"] == visit["state"],
                  "B-INTERRUPT-001", "interrupt contradicts its cleanup safe-point visit")
        self.interrupt_events = it["events"]

    # -- counters (L03, L04) ---------------------------------------------------------------

    def _counters(self):
        observed = list(self.rows) + ([self.aborted] if self.aborted else [])
        derived = {"drawsScheduled": len(self.sched), "drawsCompleted": self.completed, "drawInterrupted": self.interrupted,
                   "drawAborted": self.aborted["seq"] if self.aborted else None,
                   "drawsNotRun": len(self.sched) - self.completed - int(self.interrupted is not None) - int(self.aborted is not None),
                   "modelCalls": self.dispatched, "healthChecks": len(self.doc["healthChecks"])}
        self.facts["counters"] = derived
        self.facts["unresolvedCalls"] = len(self.unresolved)
        _need(_same(self.doc["counters"], derived), "L03", "counters disagree with the artifacts")
        _need(derived["drawsNotRun"] >= 0, "L03", "draw counts are not conserved")
        for r in observed:
            calls = r.get("calls")
            if not isinstance(calls, list):
                continue
            for n, call in enumerate(calls, 1):
                _need(set(call) == {"n", "sent", "latencyMs", "request", "requestBody", "requestSha256",
                      "status", "error", "body", "responseSha256", "usage", "servedModel", "finishReason"},
                      "B-TRANSPORT-001", "call field set")
                latency = call["latencyMs"]
                _need(isinstance(latency, (int, float)) and not isinstance(latency, bool)
                      and math.isfinite(latency) and latency >= 0, "B-TRANSPORT-001", "call latency")
                _need(_same(call.get("n"), n), "L04", "call order")
                _need(call.get("body") is None or isinstance(call["body"], str), "R-JSON-003", "response body type")
                _need(call.get("responseSha256") == sha256(call.get("body") or ""), "R-JSON-003", "response hash")
                f = SB.body_fields(call.get("body"))
                for field in ("usage", "servedModel", "finishReason"):
                    _need(_same_as_archived(f.get(field) if f.get("parsedBody") else None, call.get(field)), "R-JSON-003",
                          f"response metadata {field} is not the workflow parser's field of the retained body")
                self._request_binding(call)
            bound = self.proto.get("contexts", {}).get(r["context"], {}).get("callBound", self.proto["maxModelCallsPerDraw"])
            _need(len(calls) <= min(bound, self.proto["maxModelCallsPerDraw"]), "L04", "impossible per-draw call total")
        scheduled = {e["seq"]: e for e in self.sched}
        limits = DATA.rule("B-CALL-004")
        for draw, entries in self.by_draw.items():
            _need(draw in scheduled, "B-CALL-004", "dispatch identity absent from schedule")
            cid = scheduled[draw]["context"]
            contexts = self.proto[limits["contextPath"]]
            _need(cid in contexts, "B-CALL-004", "scheduled context has no declared call bound")
            bound = contexts[cid][limits["boundField"]]
            _need(_nat(bound) and len(entries) <= min(bound, self.proto["maxModelCallsPerDraw"]),
                  "B-CALL-004", f"draw {draw}: impossible scheduled context dispatch total")
        _need(_nat(self.dispatched) and self.dispatched <= min(self.proto["maxModelCalls"], limits["globalCeiling"]),
              "B-CALL-004", "impossible run call total")

    def _request_binding(self, call):
        """R-JSON-001: the retained request bytes, independently reparsed under
        strict JSON, equal the structured request; the hash is of those bytes."""
        req, body = call.get("request"), call.get("requestBody")
        _need(isinstance(body, str), "R-JSON-001", "retained request bytes are missing or not text")
        _need(isinstance(req, dict), "R-JSON-001", "structured request is missing or not an object")
        try:
            parsed = _strict_json(body)
        except ValueError as x:
            raise Invalid("R-JSON-001", f"retained request bytes are not strict JSON ({x})")
        _need(_same(parsed, req), "R-JSON-001",
              "structured request disagrees with its retained bytes under strict JSON (type, value, null or key set)")
        _need(call.get("requestSha256") == sha256(body), "R-JSON-001", "recorded request hash")

    # -- terminal state (L09, L10) ------------------------------------------------------------

    def _terminal(self):
        checks = self.doc["healthChecks"]
        all_healthy = all(c["serverState"] is not None and c["rejected"] is None
                          and c["serverState"]["health"]["outcome"] == "HEALTHY" for c in checks)
        only_after_schedule = all(e["disposition"] == "AFTER_SCHEDULE" for e in getattr(self, "interrupt_events", []))
        complete = (self.doc["stop"] is None and self.stop_at is None and self.aborted is None and self.interrupted is None
                    and self.doc["failureEvent"] is None and self.event_file is None
                    and self.doc["cleanup"]["status"] == "SUCCEEDED" and len(self.rows) == len(self.sched)
                    and self.completed == len(self.sched) and all_healthy and not self.unresolved
                    and only_after_schedule and self.events[-2:] == ["schedule_done", "cleanup_succeeded"])
        partial = self.doc["stop"] is not None and self.events[-1] in ("cleanup_succeeded", "cleanup_failed")
        derived = "COMPLETE" if complete else "PARTIAL" if partial else None
        self.facts["state"] = derived
        _need(derived is not None, "L09", "the evidence supports neither COMPLETE nor PARTIAL")
        if self.seal is not None:
            self.derived = derived

    # -- completion seal (S02-S04) ------------------------------------------------------------

    def _seal(self):
        s = self.seal
        if s is None:
            return
        spec_fields = set(self.spec["finalization"]["seal"]["fields"])
        _need(set(s) == spec_fields, "S02", "seal fields")
        d = self.doc
        _need(s["schema"] == SEAL_SCHEMA and s["finalization"] == FINALIZATION
              and s["finalization"] == self.spec["finalization"]["contract"], "S02", "seal schema or finalization contract")
        _need(s["lifecycle"] == {"id": DATA.LIFECYCLE_ID, "sha256": DATA.spec_sha256()} and s["runSchema"] == RUN_SCHEMA
              and s["driver"] == DRIVER, "S02", "seal contract identities")
        _need(s["runId"] == d["runId"] and s["protocol"] == {"id": self.proto["id"], "sha256": self.proto_sha}
              and s["protocol"] == {"id": d["protocol"], "sha256": d["protocolSha256"]}
              and s["scheduleSha256"] == sha256(canon(self.sched)) == d["scheduleSha256"], "S02", "seal run identity")
        t = s["transition"]
        _need(isinstance(t, dict) and set(t) == {"n", "event", "from", "to", "timestamp"} and t["from"] == "FINALIZING"
              and self.table.get(("FINALIZING", t["event"])) == t["to"] and t["to"] == s["terminalState"]
              and t["event"] in self.spec["sealOnlyEvents"] and _same(t["n"], len(self.journal) + 1), "S04",
              "seal transition")
        end = _ts(t["timestamp"], "seal transition")
        _need(end >= _ts(self.journal[-1]["timestamp"], "last journal entry"), "S04", "the seal precedes the journal")
        raw = self.a.raw
        h = lambda n: hashlib.sha256(raw[n]).hexdigest() if n in raw else None  # noqa: E731
        _need(_same(s["manifest"], {"file": "run.json", "sha256": h("run.json"), "journalEntries": len(self.journal)}),
              "S03", "seal manifest hash")
        _need(_same(s["ledger"], {"file": "draws.json", "sha256": h("draws.json"), "rows": len(self.rows),
                                  "completed": self.completed, "rowSha256": [sha256(canon(r)) for r in self.rows]}),
              "S03", "seal ledger hashes")
        _need(_same(s["calls"], {"file": CALLS, "sha256": h(CALLS), "records": len(self.records), "dispatched": self.dispatched}),
              "S03", "seal call-journal hash")
        _need(_same(s["health"], {"checks": len(d["healthChecks"]), "checkSha256": [sha256(canon(c)) for c in d["healthChecks"]],
                                  "serverBefore": h("server-before.json")}), "S03", "seal health hashes")
        _need(isinstance(s["sideArtifacts"], dict) and set(s["sideArtifacts"]) == set(SIDE), "S03", "seal side artifacts")
        for n in SIDE:
            _need(s["sideArtifacts"][n] == h(n), "S03", f"seal hash for {n} (null requires absence)")
        _need(_same(s["counters"], d["counters"]), "S03", "seal counters")
        _need(self.derived is None or s["terminalState"] == self.derived, "S04",
              f"the seal claims {s['terminalState']} but the evidence supports {self.facts.get('state')}")
        if self.derived is not None and s["terminalState"] != self.derived:
            self.derived = None

    # -- timestamps (L13) ---------------------------------------------------------------------

    def _timestamps(self):
        d = self.doc
        start = _ts(d["started"], "run.started")
        _need(d["started"] == self.journal[0]["timestamp"], "L13", "run.started is not the create transition")
        times = [_ts(e["timestamp"], f"journal entry {e['n']}") for e in self.journal]
        _need(all(a <= b for a, b in zip(times, times[1:])), "L13", "journal timestamps out of order")
        end = times[-1]
        if isinstance(self.seal, dict) and isinstance(self.seal.get("transition"), dict):
            end = max(end, _ts(self.seal["transition"]["timestamp"], "seal transition"))
        _need(start <= end, "L13", "run interval reversed")
        for v in getattr(self, "visits", []):
            c = v["check"]
            if c is not None and c["serverState"] is not None:
                obs = c["serverState"]["health"]
                _need(times[v["enter"]] <= _ts(obs["started"], "observation start") <= _ts(obs["ended"], "observation end")
                      <= times[v["exit"]], "L13", f"health check {c['index']} outside its visit")
        if isinstance(d["failureEvent"], dict):
            _ts(d["failureEvent"].get("timestamp"), "failure event")
        if isinstance(d["cleanup"], dict) and d["cleanup"].get("timestamp") is not None:
            _ts(d["cleanup"]["timestamp"], "cleanup")
        for e in getattr(self, "interrupt_events", []):
            _need(start <= _ts(e["timestamp"], "interrupt event") <= end, "L13", "interrupt event outside the run")
        begins = [k for k, e in enumerate(self.journal) if e["event"] == "begin_draw"]
        observed = list(self.rows) + ([self.aborted] if self.aborted else [])
        for k, r in enumerate(observed):
            lo, hi = times[begins[k]], times[begins[k] + 1]
            entries = self.by_draw.get(r["seq"], [])
            calls = r.get("calls") if isinstance(r.get("calls"), list) else [None] * len(entries)
            prev = lo
            for e, call in zip(entries, calls):
                t0 = _ts(e["dispatch"]["timestamp"], f"draw {r['seq']} dispatch {e['dispatch']['n']}")
                seq = [t0]
                if call is not None:
                    seq.append(_ts(call.get("sent"), f"draw {r['seq']} call {call.get('n')} sent"))
                if e["outcome"] is not None:
                    seq.append(_ts(e["outcome"]["timestamp"], f"draw {r['seq']} call outcome {e['outcome']['n']}"))
                _need(all(a <= b for a, b in zip([prev] + seq, seq + [hi])), "L13",
                      f"draw {r['seq']} call times outside the draw or out of order (dispatch <= sent <= outcome)")
                prev = seq[-1]
            if isinstance(r.get("calls"), list) and len(r["calls"]) > len(entries):
                for call in r["calls"][len(entries):]:
                    _ts(call.get("sent"), f"draw {r['seq']} call {call.get('n')} sent")


def classify(directory, proto, proto_sha, sched, contexts=None, allow_synthetic=False):
    """The archive's state as the evidence supports it, with every violated invariant."""
    return Classifier(Archive(directory), proto, proto_sha, sched, contexts, allow_synthetic).run()


# --- analysis -------------------------------------------------------------------------

def verify_journal_prefix(r, records, ctxs, pool):
    """Reconstruct requests even when the failed draw lost its call array.
    Returned journal bodies determine the retained workflow prefix; the next
    dispatch boundary stops replay without inventing a missing response.
    """
    outcomes = [x for x in records if x["draw"] == r["seq"] and x["record"] != "dispatch"]
    if not outcomes:
        return 0
    consumed = [0]

    class Boundary(Exception):
        pass

    class Limits:
        calls = 0

        def before_call(self, n):
            if n >= len(outcomes):
                raise Boundary("end of retained journal prefix")

        def after_call(self, call):
            pass

    def transport(payload):
        outcome = outcomes[consumed[0]]
        if sha256(json.dumps(payload).encode()) != outcome["requestSha256"]:
            raise ExperimentError("unrecorded draw request hash contradicts journal-prefix reconstruction")
        consumed[0] += 1
        if outcome["record"] == "raised":
            raise Boundary("external transport failure")
        return outcome["status"], outcome["body"], outcome["error"]

    c = ctxs.by_id[r["context"]]
    try:
        H.run_draw({k: r[k] for k in IDENTITY}, c, ctxs.settings[r["model"]], pool,
                   ctxs.memory(c, r["model"]), transport, Limits(), utc=lambda: None, now=lambda: 0.0)
    except ExperimentError:
        raise
    except Exception as error:
        if not isinstance(error, Boundary):
            raise ExperimentError("unrecorded journal-prefix reconstruction failed") from error
    if consumed[0] != len(outcomes):
        raise ExperimentError("unrecorded draw journal has outcomes beyond the reconstructable workflow")
    return consumed[0]


def verify_aborted(r, ctxs, pool, spec, stop, proto, records):
    """B-ABORT-002: replay every reconstructable event. Only the external
    cause remains UNVERIFIED, including when a finished workflow failed to close."""
    base = {"seq": r["seq"], "failureStage": r["failureStage"], "causeStatus": "UNVERIFIED"}
    stage = r["failureStage"]
    if stage == "draw_unrecorded":
        count = verify_journal_prefix(r, records, ctxs, pool)
        return dict(base, status="VERIFIED", engineEvents=0, modelCalls=count,
                    workflow="JOURNAL_PREFIX_REPLAYED" if count else "NO_RETAINED_CALLS",
                    reason="retained journal requests reconstructed; missing draw result not invented")
    if stage in ("draw_setup", "engine_open", "model_setup"):
        # These exact shapes retain no engine or workflow events. Their journal,
        # identities, counts and bounds have already been checked independently.
        return dict(base, status="VERIFIED", engineEvents=0, workflow="NOT_STARTED", reason="all retained evidence checked; no response invented")
    expected = r["engine"]
    calls = r["calls"]
    pos, consumed = [0], [0]
    boundary = [None]
    c = ctxs.by_id[r["context"]]
    session = pool(c, r["arm"])

    class Boundary(Exception):
        pass

    class ReplaySession:
        def request(self, rid, op, args=None):
            if pos[0] >= len(expected):
                boundary[0] = "engine_prefix"
                raise Boundary("end of retained engine prefix")
            e = expected[pos[0]]
            if e.get("op") != op or not _same_as_archived(args, e.get("args")):
                raise ExperimentError("aborted engine request/arguments contradict workflow")
            pos[0] += 1
            if e.get("answer") is None and isinstance(e.get("error"), str):
                boundary[0] = "engine_failure"
                raise Boundary("external engine failure")
            if set(e) != {"op", "args", "answer"}:
                raise ExperimentError("aborted engine response fields")
            answer = session.request(rid, op, args)
            if not _same_as_archived(answer, e["answer"]):
                raise ExperimentError("aborted engine response contradicts reconstruction")
            return answer

        def close(self):
            if stage == "engine_close":
                boundary[0] = "engine_close"
                raise Boundary("external engine close failure")

    def transport(payload):
        if consumed[0] >= len(calls):
            boundary[0] = "dispatch_boundary"
            raise Boundary("end of retained call prefix")
        call = calls[consumed[0]]
        consumed[0] += 1
        if sha256(json.dumps(payload).encode()) != call["requestSha256"] or json.dumps(payload) != call["requestBody"] \
                or not _same_as_archived(payload, call["request"]):
            raise ExperimentError("aborted request contradicts reconstruction")
        if call.get("body") is None and call.get("error", "").startswith("client: "):
            boundary[0] = "transport_failure"
            raise Boundary("external transport failure")
        return call["status"], call["body"], call["error"]

    class Limits:
        calls = 0

        def before_call(self, n):
            if r["halt"] is not None and n >= len(calls):
                h = r["halt"]
                x = H.Halt(h["code"], h["message"])
                x.evidence = h["evidence"]
                raise x
            if stage == "workflow" and n >= len(calls):
                boundary[0] = "before_dispatch"
                raise Boundary("external failure before next dispatch")

        def after_call(self, call):
            pass

    entry = {k: r[k] for k in IDENTITY}
    try:
        again = H.run_draw(entry, c, ctxs.settings[r["model"]], lambda c, arm: ReplaySession(),
                           ctxs.memory(c, r["model"]), transport, Limits(), utc=lambda: None, now=lambda: 0.0)
    except BaseException as x:
        again = getattr(x, "partialDraw", None)
        if again is None:
            raise ExperimentError("aborted reconstruction failed without evidence") from x
        if isinstance(x, ExperimentError):
            raise
    if pos[0] != len(expected) or consumed[0] != len(calls):
        raise ExperimentError(f"aborted retained engine/call evidence not consumed: {pos[0]}/{len(expected)}, {consumed[0]}/{len(calls)}, {again.get('failure')}")
    if again["failureStage"] != stage:
        raise ExperimentError("aborted stage contradicts reconstructed workflow boundary")
    engine = again["engine"]
    if boundary[0] == "engine_prefix":
        engine = engine[:-1]
    if len(engine) != len(expected):
        raise ExperimentError("aborted engine prefix length differs")
    for e, want in zip(engine, expected):
        if e.get("answer") is None and want.get("answer") is None:
            if not _same_as_archived({k: e[k] for k in ("op", "args")}, {k: want[k] for k in ("op", "args")}):
                raise ExperimentError("aborted failed engine request differs")
        elif not _same_as_archived(e, want):
            raise ExperimentError("aborted successful engine response differs")
    for f in ("initialDigests", "record", "batches", "digestsAfter", "halt", "process", "submittedRepresentation",
              "submittedRepresentationSha256", "engineInitialization", "modelCallsCompleted"):
        if not _same_as_archived(again[f], r[f]):
            raise ExperimentError("aborted " + f + " contradicts workflow reconstruction")
    if len(again["calls"]) != len(calls):
        raise ExperimentError("aborted reconstructed call count differs")
    # R-FAILURE-003: the recorded failure against the reconstructed boundary.
    failure = r["failure"]
    if boundary[0] is None:
        if not _same_as_archived(again.get("failure"), failure):
            raise ExperimentError("the reconstructed workflow failure disagrees with the recorded failure")
        cause = "RECONSTRUCTED"
    elif boundary[0] == "engine_failure":
        if not _same(expected[-1].get("error"), f"{failure['type']}: {failure['message']}"):
            raise ExperimentError("the recorded failure disagrees with the failed engine request")
        cause = "UNVERIFIED"
    elif boundary[0] == "transport_failure":
        last = calls[-1]
        if not _same(last.get("error"), f"client: {failure['type']}: {failure['message']}"):
            raise ExperimentError("the recorded failure disagrees with the raised transport call")
        cause = "UNVERIFIED"
    else:
        cause = "UNVERIFIED"
    halt = None
    if r["halt"] is not None:
        halt = {"status": "RECONSTRUCTED", "code": r["halt"]["code"], "stage": "request_start",
                "derivedFrom": ["retained calls", "protocol limits", "workflow phase"],
                "external": ["elapsedSeconds (wall clock measurement)"]}
    return dict(base, causeStatus=cause, status="VERIFIED", engineEvents=len(expected), modelCalls=len(calls),
                workflow="REPLAYED", boundary=boundary[0], halt=halt,
                reason="all retained engine/workflow evidence reconstructed; external cause unverified")


def analyze(binary, run_dirs, ctxs, proto, sched, directory, proto_sha, allow_synthetic=False):
    verdict = classify(directory, proto, proto_sha, sched, ctxs.by_id, allow_synthetic)
    if verdict["state"] == "INVALID":
        raise ExperimentError("INVALID archive: " + "; ".join(f"{v['invariant']} {v['message']}" for v in verdict["violations"]))
    a = Archive(directory)
    run_doc, rows = a.json("run.json"), a.json("draws.json")
    aborted = a.json("aborted-draw.json") if "aborted-draw.json" in a.raw else None
    spec = DATA.load()
    pool = ctxs.opener(binary)
    out, ledger, aborted_verification = [], {}, None
    try:
        for r in rows:
            c = ctxs.by_id[r["context"]]
            again = H.replay_draw(r, c, ctxs.settings[r["model"]], pool, ctxs.memory(c, r["model"]), proto)
            if not _same_as_archived(H.deterministic(again), H.deterministic(r)):
                raise ExperimentError("completed/interrupted replay differs in JSON value or type")
            if len(again["calls"]) != len(r["calls"]):
                raise ExperimentError("completed/interrupted replay call count differs")
            for mine, kept in zip(again["calls"], r["calls"]):
                if mine["requestBody"] != kept["requestBody"] or not _same_as_archived(mine["request"], kept["request"]):
                    raise ExperimentError("R-JSON-001: the replayed request differs from the retained request bytes "
                                          "or structured request")
            if r["status"] != "completed":
                continue
            x = H.classify(r)
            ses = pool(c, r["arm"])
            actual = H.Harness(ses).request("alternatives.v3.canonical", {"batches": r["batches"]})
            if not _same_as_archived(actual, _as_archived(x["submittedAction"])):
                raise ExperimentError("submitted identity disagrees")
            rep = r["submittedRepresentation"]
            rep_sha = sha256(rep)
            if rep_sha != r["submittedRepresentationSha256"] or not _same(_strict_json(rep), r["batches"]):
                raise ExperimentError("exact submitted representation disagrees")
            key = sha256(c["id"] + "|" + rep_sha)
            x["submittedRepresentationSha256"], x["executionRecord"] = rep_sha, key
            if key not in ledger:
                e = H.execute(binary, run_dirs, c, json.loads(rep)) if r["batches"] else \
                    {"status": "EMPTY_SUBMISSION", "batches": [], "executed": False, "partial": False}
                ledger[key] = {"context": c["id"], "batches": r["batches"], "submittedRepresentation": rep,
                               "submittedRepresentationSha256": rep_sha, "draws": [], "execution": e}
            ledger[key]["draws"].append({k: r[k] for k in ("seq", "context", "model", "arm", "draw")})
            x["executedAction"] = ledger[key]["execution"]
            x.update(seq=r["seq"], context=r["context"], model=r["model"], arm=r["arm"], draw=r["draw"])
            out.append(x)
        if aborted is not None:
            aborted_verification = verify_aborted(aborted, ctxs, pool, spec, run_doc["stop"], proto, [_strict_json(line) for line in a.raw[CALLS].splitlines()])
    except ExperimentError:
        raise
    except Exception as x:  # noqa: BLE001  (any replay failure is a refusal, never an acceptance)
        raise ExperimentError(f"replay refused the archive: {type(x).__name__}: {x}") from x
    finally:
        pool.close()
    cells = collections.defaultdict(list)
    for x in out:
        cells[(x["context"], x["model"], x["arm"])].append(x)
    table = []
    for (cid, mod, arm), xs in sorted(cells.items()):
        exposed = [x for x in xs if x["exposure"]]
        table.append({"context": cid, "model": mod, "arm": arm, "observed": len(xs), "exposed": len(exposed),
                      "primarySubmitted": dict(sorted(collections.Counter(x["batchesSha256"] for x in exposed).items())),
                      "secondaryRoutes": dict(sorted(collections.Counter(x["secondaryRoute"] for x in exposed).items(),
                                                     key=lambda p: str(p[0]))),
                      "results": dict(sorted(collections.Counter(x["result"] for x in xs).items())),
                      **{k: sum(bool(x[k]) for x in xs) for k in ("validationFirstCorrection", "correctionFailure",
                                                                  "deliberateNoAction", "processFailure")},
                      "modelCalls": sum(x["modelCalls"] for x in xs), "retries": sum(x["retries"] for x in xs)})
    complete = verdict["state"] == "COMPLETE"
    differences, pairs = [], []
    if complete:
        for cid, mod, arm in sorted(cells):
            if arm != "treatment":
                continue
            comp = H.primary_comparison(cells.get((cid, mod, "control"), []), cells[(cid, mod, arm)])
            if comp is not None:
                differences.append(dict(comp, context=cid, model=mod))
        for pid in sorted({c["pair"] for c in ctxs.by_id.values() if c.get("pair")}):
            for mod in proto["modelOrder"]:
                dep = {}
                for arm in H.ARMS:
                    members = [[x for x in cells.get((pid + "-" + side, mod, arm), []) if x["exposure"]] for side in ("a", "b")]
                    if all(members):
                        ma, mb = members
                        count = lambda xs: sum(x["batchesSha256"] != x["proposalSubmittedSha256"] for x in xs)  # noqa: E731
                        dep[arm] = X.newcombe(count(mb), len(mb), count(ma), len(ma))
                if len(dep) == 2:
                    pairs.append({"pair": pid, "model": mod, "departure": dep, "sensitivity": H.mover(dep["treatment"], dep["control"])})
    stop = run_doc["stop"]
    return {"schema": ANALYSIS_SCHEMA, "runStatus": verdict["state"], "lifecycle": {
                "id": DATA.LIFECYCLE_ID, "sha256": DATA.spec_sha256(), "state": verdict["state"],
                "counters": verdict["derived"]["counters"], "violations": []},
            "draws": out, "executions": ledger, "table": table, "armDifferences": differences, "pairs": pairs,
            "abortedVerification": aborted_verification,
            "failureVerification": {"status": "UNVERIFIED", "scope": "external cause only",
                                    "retainedEvidence": "VERIFIED", "stop": stop} if stop is not None else None,
            "completedVerification": "VERIFIED",
            "note": "Exposed-only canonical submitted actions. Partial coverage yields no complete-run comparisons. "
                    "Pair estimates remain descriptive context comparisons; duplicate S1-a/S6-a retained."}


def _inputs(a):
    proto, raw = json.load(open(a.protocol, encoding="utf-8")), open(a.protocol, "rb").read()
    rows, csha = H.load_contexts(a.contexts)
    if csha != proto["contextsFileSha256"]:
        raise SystemExit("the contexts file is not the protocol's")
    return proto, sha256(raw), H.schedule(proto, rows), rows


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    for name in ("classify", "analyze"):
        p = sub.add_parser(name)
        p.add_argument("--protocol", required=True)
        p.add_argument("--contexts", required=True)
        p.add_argument("--dir", required=True)
        if name == "analyze":
            p.add_argument("--binary", required=True)
            p.add_argument("--replays")
            p.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    proto, proto_sha, sched, rows = _inputs(a)
    if a.cmd == "classify":
        verdict = classify(a.dir, proto, proto_sha, sched, {c["id"]: c for c in rows})
        print(json.dumps(verdict, indent=1, sort_keys=True))
        raise SystemExit(0 if verdict["state"] != "INVALID" else 1)
    ctxs = H.Contexts(proto, rows, a.replays, {SB.SETTINGS_ENV: SB.REPLAY_URL})
    doc = analyze(a.binary, ctxs.run_dirs, ctxs, proto, sched, a.dir, proto_sha)
    H.write_json(os.path.join(a.out, "analysis.json"), doc)
    print(f"analysis: {len(doc['draws'])} completed draws ({doc['runStatus']}), {len(doc['executions'])} executions")


if __name__ == "__main__":
    main()
