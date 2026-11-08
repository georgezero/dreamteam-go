#!/usr/bin/env python3
"""V0.13i independent archive verifier and analyzer
(dreamteam.alternatives-analysis/v6).

It loads the lifecycle contract (lifecycle_v1.json) and reconstructs the
archive's state from the persisted files alone: it replays the journal
through the contract's transition table, recomputes every counter, stop,
health classification and artifact hash, derives the terminal state the
evidence supports, and classifies the archive COMPLETE, PARTIAL or INVALID.
It never takes the recorder's claimed status as truth and does not import
the recorder (v13i.py). Completed and interrupted rows are then replayed
without a model through the unchanged V0.13h draw machinery.

    python3 -I -B analysis/alternatives/v13i_verify.py classify --protocol FILE --contexts DIR --dir RUN
    python3 -I -B analysis/alternatives/v13i_verify.py analyze --binary ./dt --protocol FILE --contexts DIR \\
        --replays DIR --dir RUN --out DIR
"""
import argparse
import base64
import collections
import hashlib
import json
import os
import sys
import uuid

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import lifecycle_spec as LS  # noqa: E402
import v13h as H  # noqa: E402  (draw replay, classification and execution; not the run recorder)

SB, X = H.SB, H.X
sha256, canon = H.sha256, H.canon
ExperimentError = H.ExperimentError

RUN_SCHEMA = "dreamteam.alternatives-run/v5"
DRIVER = "dreamteam-v13i-driver/5"
FAILURE_SCHEMA = "dreamteam.run-failure/v2"
HEALTH_SCHEMA = "dreamteam.health-observation/v1"
ANALYSIS_SCHEMA = "dreamteam.alternatives-analysis/v6"
MARKER = "FINALIZING"
HALTS = {"draw_call_limit", "call_budget", "wall_clock", "transport"}
RUN_KEYS = {"schema", "driver", "lifecycle", "protocol", "protocolSha256", "scheduleSha256", "runId", "authorization",
            "provenance", "serverObserved", "started", "ended", "state", "status", "journal", "counters",
            "healthChecks", "stop", "failureEvent", "cleanup", "finalization", "verificationStatus", "resume"}
STOP_KEYS = {"code", "state", "message", "failure", "evidence", "request", "healthCheck", "beforeSignature"}
DETAIL = {"create": set(), "begin_preflight": set(), "preflight_passed": set(), "begin_health_check": {"healthCheck"},
          "health_passed": {"healthCheck"}, "begin_draw": {"draw"},
          "commit_draw": {"draw", "status", "rowSha256", "modelCalls", "reconciled"}, "continue": set(),
          "schedule_done": set(), "stop": {"code"}, "cleanup_succeeded": set(), "cleanup_failed": set(),
          "finalize_complete": set(), "finalize_partial": set()}
DRAW_KEYS = {"schema", "seq", "model", "context", "arm", "draw", "status", "halt", "initialDigests", "record", "batches",
             "engine", "digestsAfter", "calls", "submittedRepresentation", "submittedRepresentationSha256",
             "engineInitialization", "failureStage", "verificationStatus", "modelCallsCompleted"}
IDENTITY = ("seq", "model", "context", "arm", "draw")


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
    t = LS.parse_timestamp(value)
    _need(t is not None, "L13", f"invalid or missing timestamp: {what}")
    return t


class Archive:
    """The persisted files of one run directory, read once."""

    def __init__(self, directory):
        self.live = os.path.join(directory, "live")
        self.files = sorted(os.listdir(self.live)) if os.path.isdir(self.live) else None
        self.raw = {}
        for name in self.files or []:
            p = os.path.join(self.live, name)
            if os.path.isfile(p):
                with open(p, "rb") as f:
                    self.raw[name] = f.read()

    def json(self, name):
        return json.loads(self.raw[name].decode("utf-8"))


class Classifier:
    """The authoritative invariant classifier (L01-L19, L11 in analysis)."""

    def __init__(self, archive, proto, proto_sha, sched, contexts=None, allow_synthetic=False):
        self.a, self.proto, self.proto_sha, self.sched = archive, proto, proto_sha, sched
        self.contexts, self.allow_synthetic = contexts or {}, allow_synthetic
        self.spec = LS.load()
        self.table = LS.transition_table(self.spec)
        self.violations = []

    # -- entry --------------------------------------------------------------------

    def run(self):
        try:
            self._load()
        except Invalid as x:
            return self._result(x)
        for group in (self._identity, self._journal, self._ledger, self._aborted, self._health, self._stop,
                      self._cleanup, self._counters, self._terminal, self._finalization, self._timestamps):
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
        doc = getattr(self, "doc", None)
        return {"state": "INVALID" if self.violations else self.derived,
                "claimed": doc.get("status") if isinstance(doc, dict) else None,
                "violations": self.violations, "derived": getattr(self, "facts", None)}

    # -- files (L16) -------------------------------------------------------------------

    def _load(self):
        a = self.a
        self.doc = None
        _need(a.files is not None, "L16", "no archive directory")
        _need(MARKER not in a.files, "L16", "FINALIZING marker present: finalization did not complete")
        _need("run.json" in a.raw, "L16", "no run manifest")
        _need("draws.json" in a.raw, "L16", "no authoritative ledger")
        stray = set(a.files) - set(self.spec["allowedArchiveFiles"])
        _need(not stray, "L16", "unexpected archive files: " + ", ".join(sorted(stray)))
        try:
            self.doc = a.json("run.json")
            self.rows = a.json("draws.json")
            self.aborted = a.json("aborted-draw.json") if "aborted-draw.json" in a.raw else None
            self.event_file = a.json("failure-event.json") if "failure-event.json" in a.raw else None
            self.before = a.json("server-before.json") if "server-before.json" in a.raw else None
        except (ValueError, UnicodeDecodeError) as x:
            raise Invalid("L16", f"unreadable archive file: {x}")
        _need(isinstance(self.doc, dict) and set(self.doc) == RUN_KEYS, "L15", "run manifest fields")
        _need(isinstance(self.rows, list), "L01", "ledger is not a list")
        self.journal = self.doc["journal"]
        _need(isinstance(self.journal, list) and self.journal and all(isinstance(e, dict) for e in self.journal),
              "L14", "journal missing")
        self.derived = None
        self.facts = {}

    # -- identity (L15, L18) ------------------------------------------------------------

    def _identity(self):
        d = self.doc
        _need(d["schema"] == RUN_SCHEMA and d["driver"] == DRIVER, "L15", "run schema or driver")
        _need(d["lifecycle"] == {"id": LS.LIFECYCLE_ID, "sha256": LS.spec_sha256()}, "L15", "lifecycle identity")
        _need(d["protocol"] == self.proto["id"] and d["protocolSha256"] == self.proto_sha, "L15", "protocol identity")
        _need(d["scheduleSha256"] == sha256(canon(self.sched)), "L15", "schedule identity")
        prov = d["provenance"]
        _need(isinstance(prov, dict) and set(prov) == {"timestamps", "runId"}
              and prov["timestamps"] in ("SYSTEM_UTC", "SYNTHETIC_FIXTURE") and prov["runId"] in ("UUID4", "SYNTHETIC_FIXTURE"),
              "L15", "provenance")
        synthetic = "SYNTHETIC_FIXTURE" in prov.values()
        _need(not synthetic or self.allow_synthetic, "L13", "synthetic provenance outside an explicit fixture check")
        try:
            u = uuid.UUID(d["runId"])
        except (ValueError, TypeError, AttributeError):
            raise Invalid("L15", "run identity is not a UUID")
        _need(str(u) == d["runId"] and (prov["runId"] != "UUID4" or u.version == 4), "L15", "run identity form")
        _need(isinstance(d["authorization"], str) and d["authorization"].strip(), "L15", "authorization")
        _need(isinstance(d["serverObserved"], bool), "L15", "serverObserved")
        _need(d["resume"] == "never", "L18", "the manifest permits resumption")
        _need(d["verificationStatus"] == "UNVERIFIED", "L15", "the recording claims verification")

    # -- journal (L14, L19) ------------------------------------------------------------

    def _journal(self):
        states, events = self.spec["states"], self.spec["events"]
        prev = None
        for k, e in enumerate(self.journal):
            ev = e.get("event")
            _need(ev in events, "L14", f"unknown event {ev!r}")
            _need(set(e) == {"n", "event", "from", "to", "timestamp"} | DETAIL[ev], "L14", f"journal entry {k + 1} fields")
            _need(e["n"] == k + 1, "L14", "journal order")
            _need(e["from"] is None or e["from"] in states, "L14", f"unknown state {e['from']!r}")
            _need(e["from"] == prev, "L14", f"journal does not chain at entry {k + 1}")
            _need((e["from"], ev) in self.table and self.table[(e["from"], ev)] == e["to"], "L14",
                  f"illegal transition {e['from']} --{ev}--> {e['to']}")
            prev = e["to"]
        _need(self.journal[0]["event"] == "create", "L14", "journal does not start with create")
        _need(states[prev]["terminal"], "L14", f"run was not finalized (journal ends in {prev})")
        _need(self.doc["state"] == prev and self.doc["status"] == prev, "L14", "claimed state disagrees with the journal")
        stops = [k for k, e in enumerate(self.journal) if e["event"] == "stop"]
        _need(len(stops) <= 1, "L19", "more than one stop")
        self.stop_at = stops[0] if stops else None
        self.events = [e["event"] for e in self.journal]

    # -- ledger (L01, L02, L17) ----------------------------------------------------------

    def _ledger(self):
        rows, sched = self.rows, self.sched
        _need(len(rows) <= len(sched), "L01", "more rows than scheduled")
        seen = set()
        for k, r in enumerate(rows):
            _need(isinstance(r, dict) and DRAW_KEYS <= set(r), "L02", f"row {k + 1} lacks required evidence")
            _need(r["schema"] == H.DRAW_SCHEMA and {f: r[f] for f in IDENTITY} == sched[k], "L01",
                  f"row {k + 1} is not the schedule's draw {sched[k]['seq']}")
            _need(r["seq"] not in seen, "L01", "duplicate row")
            seen.add(r["seq"])
            _need(r["status"] in ("completed", "interrupted"), "L01", f"row status {r['status']!r}")
            _need(r["status"] == "completed" or k == len(rows) - 1, "L01", "an unfinished row before the last")
            _need(not ({"failure", "cleanupFailure", "stopReason"} & set(r)), "L02",
                  f"row {r['seq']} carries failure evidence contradicting its status")
            _need(r["engineInitialization"] == "READY" and r["failureStage"] is None and r["verificationStatus"] == "RECORDED",
                  "L02", f"row {r['seq']} initialization or verification status")
            _need(isinstance(r["calls"], list) and r["modelCallsCompleted"] == len(r["calls"]), "L04", "row call count")
            _need(isinstance(r["initialDigests"], dict) and isinstance(r["digestsAfter"], dict)
                  and isinstance(r["engine"], list) and r["engine"], "L02", f"row {r['seq']} engine evidence")
            c = self.contexts.get(r["context"])
            if c is not None:
                _need(r["initialDigests"] == c["digests"], "L02", f"row {r['seq']} initial digests")
            if r["status"] == "completed":
                rep = r["submittedRepresentation"]
                _need(r["halt"] is None and isinstance(r["record"], dict) and isinstance(r["batches"], list)
                      and isinstance(rep, str) and sha256(rep) == r["submittedRepresentationSha256"]
                      and json.loads(rep) == r["batches"] and isinstance(r.get("process"), dict), "L02",
                      f"completed row {r['seq']} evidence")
            else:
                h = r["halt"]
                _need(isinstance(h, dict) and set(h) == {"code", "message", "evidence"} and h["code"] in HALTS
                      and r["batches"] is None and r["record"] is None and r["submittedRepresentation"] is None,
                      "L02", f"interrupted row {r['seq']} evidence")
        begins = [e for e in self.journal if e["event"] == "begin_draw"]
        commits = [e for e in self.journal if e["event"] == "commit_draw"]
        _need(len(commits) == len(rows), "L17", "journal commits disagree with the ledger")
        for e in commits:
            if e["reconciled"] is True:
                nxt = self.journal[e["n"]] if e["n"] < len(self.journal) else None
                _need(nxt is not None and nxt["event"] == "stop", "L17", "a reconciled commit not followed by its stop")
        for k, e in enumerate(commits):
            r = rows[k]
            _need(e["draw"] == r["seq"] and e["status"] == r["status"] and e["rowSha256"] == sha256(canon(r))
                  and e["modelCalls"] == len(r["calls"]) and isinstance(e["reconciled"], bool), "L17",
                  f"commit {k + 1} does not bind its ledger row")
        from_active = self.stop_at is not None and self.journal[self.stop_at]["from"] == "DRAW_ACTIVE"
        _need(len(begins) == len(rows) + int(from_active), "L17", "begun draws disagree with committed rows")
        for k, e in enumerate(begins):
            _need(k < len(sched) and e["draw"] == sched[k]["seq"], "L17", f"begin_draw {k + 1} is not the next scheduled draw")
        self.completed = sum(r["status"] == "completed" for r in rows)
        self.interrupted = rows[-1]["seq"] if rows and rows[-1]["status"] == "interrupted" else None

    # -- aborted draw (L05, L06) -------------------------------------------------------------

    def _aborted(self):
        from_active = self.stop_at is not None and self.journal[self.stop_at]["from"] == "DRAW_ACTIVE"
        ab = self.aborted
        _need((ab is not None) == from_active, "L05", "aborted artifact present iff the stop is from DRAW_ACTIVE")
        if ab is None:
            return
        _need(isinstance(ab, dict) and (DRAW_KEYS | {"failure", "stopReason"}) <= set(ab), "L05", "aborted artifact fields")
        _need(ab["schema"] == H.DRAW_SCHEMA and ab["status"] == "aborted", "L05", "aborted artifact status")
        _need(len(self.rows) < len(self.sched) and {f: ab[f] for f in IDENTITY} == self.sched[len(self.rows)], "L05",
              "aborted draw is not the next scheduled draw")
        _need(ab["seq"] not in {r["seq"] for r in self.rows}, "L05", "an aborted draw is also a ledger row")
        stage = self.spec["abortedDraw"]["stages"].get(ab["failureStage"])
        _need(stage is not None, "L06", f"unknown aborted failure stage {ab['failureStage']!r}")
        _need(ab["engineInitialization"] in stage["engineInitialization"], "L06",
              f"failure stage {ab['failureStage']} contradicts initialization {ab['engineInitialization']}")
        _need(_failure_shape(ab["failure"]) and ab["verificationStatus"] == "UNVERIFIED", "L06", "aborted failure evidence")
        _need(isinstance(ab["calls"], list) and ab["modelCallsCompleted"] == len(ab["calls"]), "L04", "aborted call count")
        _need(isinstance(ab["engine"], list), "L06", "aborted engine log")
        ctx_digests = (self.contexts.get(ab["context"]) or {}).get("digests")
        if stage["calls"] == "none":
            _need(not ab["calls"], "L06", f"{ab['failureStage']} failure with recorded model calls")
        if stage["engine"] == "none":
            _need(not ab["engine"], "L06", f"{ab['failureStage']} failure with engine exchanges")
        elif stage["engine"] == "at most one digests request":
            _need(len(ab["engine"]) <= 1 and all(e.get("op") == "digests" for e in ab["engine"]), "L06",
                  "initial-digests failure with other engine exchanges")
        if stage["initialDigests"] == "null":
            _need(ab["initialDigests"] is None, "L06", f"{ab['failureStage']} failure with initial digests")
        elif stage["initialDigests"] == "the context digests":
            _need(isinstance(ab["initialDigests"], dict) and (ctx_digests is None or ab["initialDigests"] == ctx_digests),
                  "L06", f"{ab['failureStage']} failure without the context's initial digests")
        else:
            _need(ab["initialDigests"] is None or ctx_digests is None or ab["initialDigests"] == ctx_digests, "L06",
                  "initial digests disagree")
        _need(ab["batches"] is None or ab["failureStage"] == "engine_close", "L05", "aborted draw submitted actions")
        if "cleanupFailure" in ab:
            _need(ab["failureStage"] != "engine_close" and _failure_shape(ab["cleanupFailure"]), "L08",
                  "session cleanup failure without a different primary failure")

    # -- health (L07) --------------------------------------------------------------------

    def observation_problem(self, obs):
        h = self.spec["health"]
        keys = {"schema", "request", "started", "ended", "outcome", "category", "httpStatus", "exception", "body",
                "verificationStatus"}
        if not isinstance(obs, dict) or set(obs) != keys or obs["schema"] != HEALTH_SCHEMA or obs["request"] != h["request"]:
            return "not a health observation"
        if obs["verificationStatus"] != "UNVERIFIED":
            return "health observation claims verification"
        t0, t1 = LS.parse_timestamp(obs["started"]), LS.parse_timestamp(obs["ended"])
        if t0 is None or t1 is None or t1 < t0:
            return "health observation timestamps"
        cat, body, exc, status = obs["category"], obs["body"], obs["exception"], obs["httpStatus"]

        def body_bytes():
            if not (isinstance(body, dict) and set(body) == {"bytes", "sha256", "base64"}):
                return None
            try:
                raw = base64.b64decode(body["base64"], validate=True)
            except (ValueError, TypeError):
                return None
            if len(raw) != body["bytes"] or hashlib.sha256(raw).hexdigest() != body["sha256"]:
                return None
            return raw
        if cat in h["bodyCategories"]:
            raw = body_bytes()
            if exc is not None or raw is None or not (_nat(status) and 200 <= status < 300):
                return "a response category without a valid response"
            if LS.health_body_category(raw, h["maxBodyBytes"]) != cat:
                return "the recorded body does not classify as recorded"
            if obs["outcome"] != h["bodyCategories"][cat]:
                return "outcome disagrees with category"
            return None
        rules = h["readFailureCategories"]
        if cat not in rules:
            return "unknown health category"
        if obs["outcome"] != "READ_FAILED" or not (isinstance(exc, dict) and set(exc) == {"type", "reasonType"}
                                                   and isinstance(exc["type"], str) and exc["type"]
                                                   and (exc["reasonType"] is None or isinstance(exc["reasonType"], str))):
            return "read failure without exception evidence"
        if status is not None and not _nat(status):
            return "invalid HTTP status"
        specific = {c: r for c, r in rules.items() if c not in ("url_error", "os_error", "http_status")}
        if cat == "http_status":
            if exc["type"] != "HTTPError" or exc["reasonType"] is not None or not _nat(status) or 200 <= status < 300:
                return "http_status without a non-2xx HTTPError"
            if body is not None and body_bytes() is None:
                return "invalid error body"
            return None
        if body is not None:
            return "read failure with a response body"
        if exc["type"] == "URLError":
            listed = {t for r in specific.values() for t in r["urlErrorReasonTypes"]}
            want = next((c for c, r in specific.items() if exc["reasonType"] in r["urlErrorReasonTypes"]), "url_error")
            if cat != want or (cat == "url_error" and exc["reasonType"] in listed):
                return "URLError categorised against its reason"
            return None
        if exc["reasonType"] is not None:
            return "reason type outside a URLError"
        want = next((c for c, r in specific.items() if exc["type"] in r["exceptionTypes"]), None)
        if want is None:
            want = "os_error" if exc["type"] not in ("HTTPError",) else None
        if cat != want:
            return "exception categorised against its class"
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
                  and c["index"] == k and c["state"] in ("PREFLIGHT", "HEALTH_CHECK"), "L07", f"health check {k} fields")
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
        # Walk the journal: every visit to PREFLIGHT / HEALTH_CHECK and every draw start.
        committed, used, self.visits = 0, 0, []
        for k, e in enumerate(self.journal):
            ev = e["event"]
            if ev == "commit_draw":
                committed += 1
            if ev in ("begin_preflight", "begin_health_check"):
                nxt = self.journal[k + 1] if k + 1 < len(self.journal) else None
                _need(nxt is not None, "L07", "health visit without an exit")
                state = "PREFLIGHT" if ev == "begin_preflight" else "HEALTH_CHECK"
                if state == "HEALTH_CHECK":
                    _need(e["healthCheck"] == used and committed < len(self.sched) and self._needs_check(committed), "L07",
                          "round health check where the schedule requires none, or out of order")
                check = checks[used] if used < len(checks) and checks[used]["state"] == state else None
                if check is not None:
                    want = None if state == "PREFLIGHT" else self.sched[committed]["seq"]
                    _need(check["beforeDraw"] == want, "L07", "health check names the wrong draw")
                    used += 1
                self.visits.append({"state": state, "check": check, "enter": k, "exit": k + 1, "draw": committed})
                if nxt["event"] in ("preflight_passed", "health_passed"):
                    _need(check is not None and check["serverState"] is not None
                          and check["serverState"]["health"]["outcome"] == "HEALTHY", "L07",
                          "a passing health transition without a HEALTHY observation")
                    if nxt["event"] == "health_passed":
                        _need(nxt["healthCheck"] == check["index"], "L07", "health_passed names another check")
                        _need(X.server_signature(check["serverState"]) == self.signature, "L07",
                              "health passed although the server signature changed")
                if state == "PREFLIGHT" and check is not None and check["serverState"] is not None:
                    self.signature = X.server_signature(check["serverState"])
            if ev == "begin_draw":
                i = committed
                prior = self.journal[k - 1]["event"]
                _need(prior == ("health_passed" if self._needs_check(i) else "continue"), "L07",
                      f"draw {self.sched[i]['seq']} started without the round health check the schedule requires"
                      if self._needs_check(i) else f"draw {self.sched[i]['seq']} started after an unrequired event")
        _need(used == len(checks), "L07", "health checks outside any health visit")
        first = checks[0] if checks and checks[0]["state"] == "PREFLIGHT" else None
        if first is not None and first["serverState"] is not None:
            _need(self.before == first["serverState"], "L07", "server-before.json disagrees with the initial health check")
        else:
            _need(self.before is None, "L07", "server-before.json without an initial health observation")

    def _needs_check(self, i):
        if not self.doc["serverObserved"]:
            return False
        s = self.sched
        return i == 0 or (s[i]["model"], s[i]["draw"]) != (s[i - 1]["model"], s[i - 1]["draw"])

    # -- stop and failure event (L06, L08, L19) -----------------------------------------------

    def _calls_before(self):
        return [c for r in self.rows for c in r["calls"]] + (list(self.aborted["calls"]) if self.aborted else [])

    def _streak(self, calls):
        n = 0
        for c in calls:
            n = n + 1 if c.get("status") is None and c.get("error") else 0
        return n

    def _stop(self):
        doc, stop = self.doc, self.doc["stop"]
        cleanup_created = self.stop_at is None and "cleanup_failed" in self.events and stop is not None
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
            _need(stop["state"] == "CLEANUP", "L08", "cleanup-created stop outside CLEANUP")
        else:
            e = self.journal[self.stop_at]
            entry_ts = e["timestamp"]
            _need(stop["code"] == e["code"] and stop["state"] == e["from"], "L19", "stop disagrees with the journal")
            _need(stop["state"] != "CLEANUP", "L06", "a stop transition from CLEANUP")
        _need(stop["state"] in rule["states"], "L06", f"stop {code} cannot come from {stop['state']}")
        visit = next((v for v in reversed(getattr(self, "visits", [])) if v["exit"] == self.stop_at), None)
        check = visit["check"] if visit else None
        req = self.spec["health"]["request"]
        calls = self._calls_before()
        if code == "health":
            _need(check is not None and check["serverState"] is not None
                  and check["serverState"]["health"]["outcome"] != "HEALTHY", "L07", "health stop without a failed observation")
            _need(stop["failure"] is None and stop["evidence"] is None and stop["request"] == req
                  and stop["healthCheck"] == check["index"] and stop["beforeSignature"] is None, "L06", "health stop evidence")
        elif code == "server_change":
            _need(check is not None and check["serverState"] is not None
                  and check["serverState"]["health"]["outcome"] == "HEALTHY"
                  and X.server_signature(check["serverState"]) != self.signature
                  and stop["beforeSignature"] == self.signature, "L07", "server-change stop evidence")
            _need(stop["failure"] is None and stop["evidence"] is None and stop["request"] is None
                  and stop["healthCheck"] == check["index"], "L06", "server-change stop fields")
        elif code in HALTS:
            _need(stop["failure"] is None and stop["request"] is None and stop["healthCheck"] is None
                  and stop["beforeSignature"] is None, "L06", "halt stop fields")
            H.check_halt({"code": code, "evidence": stop["evidence"]}, self.proto)
            ev = stop["evidence"]
            if stop["state"] == "DRAW_RECORDED":
                _need(self.interrupted is not None and {k: stop[k] for k in ("code", "message", "evidence")}
                      == self.rows[-1]["halt"], "L06", "halt differs from the interrupted row")
                _need(ev["drawCalls"] == len(self.rows[-1]["calls"]), "L04", "halt draw call evidence")
            else:
                _need(ev["stage"] == "draw_start" and ev["drawCalls"] == 0, "L06", "draw-guard halt evidence")
            _need(ev["runCalls"] == len(calls) and ev["consecutiveTransportFailures"] == self._streak(calls), "L04",
                  "halt call evidence disagrees with recorded calls")
        else:
            f = stop["failure"]
            _need(_failure_shape(f) and (code == "operator") == (f["type"] == "KeyboardInterrupt"), "L06",
                  "exception stop evidence")
            _need(stop["evidence"] is None and stop["beforeSignature"] is None, "L06", "exception stop fields")
            interrupted_read = check is not None and check["serverState"] is None and check["rejected"] is None
            _need(stop["request"] == (req if interrupted_read else None), "L07", "attempted health request disagrees")
            _need(stop["healthCheck"] == (check["index"] if check is not None else None), "L07",
                  "stop names the wrong health check")
            if check is not None and check["rejected"] is not None:
                _need(code == "driver", "L07", "a rejected observation must stop the run as a driver failure")
            if stop["state"] == "DRAW_ACTIVE":
                _need(self.aborted is not None and f == self.aborted["failure"] and self.aborted["stopReason"] == code,
                      "L06", "primary draw failure disagrees with the aborted artifact")
            if stop["state"] == "DRAW_RECORDED":
                _need(self.rows and self.rows[-1]["status"] == "completed", "L06",
                      "an exception stop after an interrupted row (the halt is the stop)")
            if cleanup_created:
                _need(f == doc["cleanup"]["failure"] and stop["message"] == f"{f['type']}: {f['message']}"
                      and stop["request"] is None, "L08", "cleanup-created stop disagrees with the cleanup failure")
        if self.interrupted is not None:
            _need(code in HALTS and stop["state"] == "DRAW_RECORDED", "L06", "an interrupted row needs its halt as the stop")
        nxt = ({k: self.aborted[k] for k in IDENTITY} if self.aborted else
               {k: self.rows[-1][k] for k in IDENTITY} if self.interrupted is not None else
               self.sched[len(self.rows)] if len(self.rows) < len(self.sched) else None)
        want = {"schema": FAILURE_SCHEMA, "runId": doc["runId"], "protocolSha256": doc["protocolSha256"],
                "scheduleSha256": doc["scheduleSha256"], "state": stop["state"], "stop": stop, "nextDraw": nxt,
                "drawsCompleted": self.completed, "modelCalls": len(calls), "timestamp": entry_ts,
                "verificationStatus": "UNVERIFIED"}
        _need(doc["failureEvent"] == want, "L19", "failure event disagrees with the derived stop")
        _need(self.event_file == doc["failureEvent"], "L16", "failure-event.json missing or disagrees")

    # -- cleanup (L08) -----------------------------------------------------------------

    def _cleanup(self):
        c = self.doc["cleanup"]
        _need(isinstance(c, dict) and set(c) == {"status", "failure", "timestamp"} and c["status"] in ("SUCCEEDED", "FAILED"),
              "L08", "cleanup evidence missing")
        _need((c["status"] == "FAILED") == ("cleanup_failed" in self.events)
              and (c["status"] == "SUCCEEDED") == ("cleanup_succeeded" in self.events), "L08", "cleanup disagrees with the journal")
        _need((c["failure"] is None) == (c["status"] == "SUCCEEDED") and (c["failure"] is None or _failure_shape(c["failure"])),
              "L08", "contradictory cleanup failure")
        ev = "cleanup_failed" if c["status"] == "FAILED" else "cleanup_succeeded"
        _need(c["timestamp"] == self.journal[self.events.index(ev)]["timestamp"], "L13", "cleanup timestamp")
        if c["status"] == "FAILED" and self.stop_at is None:
            _need(self.doc["stop"] is not None and self.doc["stop"]["state"] == "CLEANUP", "L08",
                  "cleanup failure without a stop")

    # -- counters (L03, L04) ---------------------------------------------------------------

    def _counters(self):
        observed = list(self.rows) + ([self.aborted] if self.aborted else [])
        calls = sum(len(r["calls"]) for r in observed)
        derived = {"drawsScheduled": len(self.sched), "drawsCompleted": self.completed, "drawInterrupted": self.interrupted,
                   "drawAborted": self.aborted["seq"] if self.aborted else None,
                   "drawsNotRun": len(self.sched) - self.completed - int(self.interrupted is not None) - int(self.aborted is not None),
                   "modelCalls": calls, "healthChecks": len(self.doc["healthChecks"])}
        self.facts["counters"] = derived
        _need(self.doc["counters"] == derived, "L03", "counters disagree with the artifacts")
        _need(derived["drawsNotRun"] >= 0, "L03", "draw counts are not conserved")
        for r in observed:
            for n, call in enumerate(r["calls"], 1):
                _need(call.get("n") == n, "L04", "call order")
                _need(call.get("responseSha256") == sha256(call.get("body") or ""), "L04", "response hash")
                f = SB.body_fields(call.get("body"))
                for field in ("usage", "servedModel", "finishReason"):
                    _need(call.get(field) == (f.get(field) if f.get("parsedBody") else None), "L04", "response metadata")
                req, body = call.get("request"), call.get("requestBody")
                _need(isinstance(req, dict) and isinstance(body, str) and json.loads(body) == req
                      and call.get("requestSha256") == sha256(body), "L04", "recorded request hash")
            bound = self.proto.get("contexts", {}).get(r["context"], {}).get("callBound", self.proto["maxModelCallsPerDraw"])
            _need(len(r["calls"]) <= min(bound, self.proto["maxModelCallsPerDraw"]), "L04", "impossible per-draw call total")
        _need(calls <= self.proto["maxModelCalls"], "L04", "impossible run call total")

    # -- terminal state (L09, L10) ------------------------------------------------------------

    def _terminal(self):
        checks = self.doc["healthChecks"]
        all_healthy = all(c["serverState"] is not None and c["serverState"]["health"]["outcome"] == "HEALTHY" for c in checks)
        complete = (self.doc["stop"] is None and self.stop_at is None and self.aborted is None and self.interrupted is None
                    and self.doc["cleanup"]["status"] == "SUCCEEDED" and len(self.rows) == len(self.sched)
                    and self.completed == len(self.sched) and all_healthy
                    and self.events[-3:] == ["schedule_done", "cleanup_succeeded", "finalize_complete"])
        partial = self.doc["stop"] is not None and self.events[-1] == "finalize_partial"
        self.derived = "COMPLETE" if complete else "PARTIAL" if partial else None
        _need(self.derived is not None, "L09" if self.events[-1] == "finalize_complete" else "L10",
              "the evidence supports neither COMPLETE nor PARTIAL")
        _need(self.doc["status"] == self.derived, "L09" if self.doc["status"] == "COMPLETE" else "L10",
              f"claimed {self.doc['status']} but the evidence supports {self.derived}")
        self.facts["state"] = self.derived

    # -- finalization (L16) ------------------------------------------------------------------

    def _finalization(self):
        fin = self.doc["finalization"]
        names = [n for n in self.spec["allowedArchiveFiles"] if n != "run.json"]
        _need(isinstance(fin, dict) and set(fin) == {"timestamp", "artifacts"} and isinstance(fin["artifacts"], dict)
              and set(fin["artifacts"]) == set(names), "L16", "finalization record")
        for n in names:
            want = hashlib.sha256(self.a.raw[n]).hexdigest() if n in self.a.raw else None
            _need(fin["artifacts"][n] == want, "L16", f"artifact hash disagrees: {n}")
        _need(fin["timestamp"] == self.doc["ended"] == self.journal[-1]["timestamp"], "L13", "finalization timestamp")

    # -- timestamps (L13) ---------------------------------------------------------------------

    def _timestamps(self):
        d = self.doc
        start, end = _ts(d["started"], "run.started"), _ts(d["ended"], "run.ended")
        _need(d["started"] == self.journal[0]["timestamp"], "L13", "run.started is not the create transition")
        times = [_ts(e["timestamp"], f"journal entry {e['n']}") for e in self.journal]
        _need(all(a <= b for a, b in zip(times, times[1:])), "L13", "journal timestamps out of order")
        _need(start <= end and times[-1] == end, "L13", "run.ended is not the finalize transition")
        for v in getattr(self, "visits", []):
            c = v["check"]
            if c is not None and c["serverState"] is not None:
                obs = c["serverState"]["health"]
                _need(times[v["enter"]] <= _ts(obs["started"], "observation start") <= _ts(obs["ended"], "observation end")
                      <= times[v["exit"]], "L13", f"health check {c['index']} outside its visit")
        begins = [k for k, e in enumerate(self.journal) if e["event"] == "begin_draw"]
        observed = list(self.rows) + ([self.aborted] if self.aborted else [])
        for k, r in enumerate(observed):
            lo = times[begins[k]]
            hi = times[begins[k] + 1]
            prev = lo
            for call in r["calls"]:
                t = _ts(call.get("sent"), f"draw {r['seq']} call {call.get('n')} sent")
                _need(prev <= t <= hi, "L13", f"draw {r['seq']} call times outside the draw or out of order")
                prev = t


def classify(directory, proto, proto_sha, sched, contexts=None, allow_synthetic=False):
    """The archive's state as the evidence supports it, with every violated invariant."""
    return Classifier(Archive(directory), proto, proto_sha, sched, contexts, allow_synthetic).run()


# --- analysis -------------------------------------------------------------------------

def verify_aborted(r, ctxs, pool, spec):
    stage = spec["abortedDraw"]["stages"][r["failureStage"]]
    base = {"seq": r["seq"], "failureStage": r["failureStage"], "causeStatus": "UNVERIFIED"}
    if r["failureStage"] != "workflow" or not r["calls"] or r["calls"][-1].get("body") is not None:
        return dict(base, status="UNVERIFIED", reason=f"{r['failureStage']}: {stage['reconstruction']}; no response invented")

    class RecordedFailure(Exception):
        pass
    calls = list(r["calls"])

    def transport(payload):
        if not calls:
            raise ExperimentError("unrecorded aborted request")
        call = calls.pop(0)
        if sha256(json.dumps(payload).encode()) != call["requestSha256"]:
            raise ExperimentError("aborted request disagrees")
        if call.get("body") is None:
            raise RecordedFailure(call.get("error"))
        return call["status"], call["body"], call["error"]
    c = ctxs.by_id[r["context"]]
    try:
        H.run_draw(r, c, ctxs.settings[r["model"]], pool, ctxs.memory(c, r["model"]), transport, None,
                   utc=lambda: None, now=lambda: 0.0)
    except RecordedFailure as x:
        if calls or x.partialDraw["engine"] != r["engine"]:
            raise ExperimentError("aborted engine exchanges disagree")
        return dict(base, status="VERIFIED", reason="recorded transport failure reproduced; external cause unverified")
    raise ExperimentError("aborted draw completed during reconstruction")


def analyze(binary, run_dirs, ctxs, proto, sched, directory, proto_sha, allow_synthetic=False):
    verdict = classify(directory, proto, proto_sha, sched, ctxs.by_id, allow_synthetic)
    if verdict["state"] == "INVALID":
        raise ExperimentError("INVALID archive: " + "; ".join(f"{v['invariant']} {v['message']}" for v in verdict["violations"]))
    a = Archive(directory)
    run_doc, rows = a.json("run.json"), a.json("draws.json")
    aborted = a.json("aborted-draw.json") if "aborted-draw.json" in a.raw else None
    spec = LS.load()
    pool = ctxs.opener(binary)
    out, ledger, aborted_verification = [], {}, None
    try:
        for r in rows:
            c = ctxs.by_id[r["context"]]
            H.replay_draw(r, c, ctxs.settings[r["model"]], pool, ctxs.memory(c, r["model"]), proto)
            if r["status"] != "completed":
                continue
            x = H.classify(r)
            ses = pool(c, r["arm"])
            actual = H.Harness(ses).request("alternatives.v3.canonical", {"batches": r["batches"]})
            if actual != x["submittedAction"]:
                raise ExperimentError("submitted identity disagrees")
            rep = r["submittedRepresentation"]
            rep_sha = sha256(rep)
            if rep_sha != r["submittedRepresentationSha256"] or json.loads(rep) != r["batches"]:
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
            aborted_verification = verify_aborted(aborted, ctxs, pool, spec)
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
                "id": LS.LIFECYCLE_ID, "sha256": LS.spec_sha256(), "state": verdict["state"],
                "counters": verdict["derived"]["counters"], "violations": []},
            "draws": out, "executions": ledger, "table": table, "armDifferences": differences, "pairs": pairs,
            "abortedVerification": aborted_verification,
            "failureVerification": {"status": "UNVERIFIED", "stop": stop} if stop is not None else None,
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
