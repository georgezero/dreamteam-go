#!/usr/bin/env python3
"""Durable evidence ownership (B-HANDOFF-001, R-HANDOFF-001, R-RESPONSE-003).

1. The six inherited completed-draw filesystem/return/error-handling cases.
2. Model-response handoff faults: a returned response whose outcome record
   fails at write, fsync or close, with retention succeeding, failing at its
   rename, or failing entirely (every later write refused).
3. Exhaustive boundary enumeration: every line event of the recorder (and of
   run_draw's bookkeeping) from the moment a draw completes until its ledger
   commit is durable, and from the moment a model response is received until
   its outcome record is durable, each made the site of a handled exception
   in its own run. The completed draw or received response must survive
   (ledger, journal, handoff, failure report or structured error) and an
   accepted archive must contain it; no replacement model call is made.
Scripted replies and a real local engine; no inference.
"""
import argparse
import concurrent.futures
import copy
import errno
import json
import linecache
import os
from pathlib import Path
import shutil
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import v13l as K
import v13l_scenarios as S
import lifecycle_v4_spec as LS

INHERITED = ["close_failure_after_fsync", "error_handling_failure", "fsync_failure", "interruption_before_return",
             "ledger_commit_failure", "rename_failure"]
assert set(INHERITED) <= set(LS.rule("B-HANDOFF-001")["cases"])
MARKER = "UNIQUE_RECEIVED_RESPONSE_MARKER"
RECORDER = os.path.abspath(K.__file__)
DRAWFILE = os.path.abspath(K.H.__file__)


def one(binary, scratch, kind):
    fixture = S.Fixture(binary)
    out = Path(scratch) / kind
    clock = S.Clock()
    armed, fired, after_sync = [False], [False], [False]

    class FS(K.OsFS):
        def fail(self, operation):
            fired[0] = True
            raise OSError(errno.EIO, "completed draw " + operation)

        def fsync(self, fd):
            if armed[0] and kind == "fsync_failure" and not fired[0]:
                self.fail("fsync failure")
            super().fsync(fd)
            if armed[0]:
                after_sync[0] = True

        def close(self, fd):
            super().close(fd)
            if armed[0] and kind == "close_failure_after_fsync" and not fired[0]:
                assert after_sync[0]
                self.fail("close failure after successful fsync")

        def replace(self, src, dst):
            if armed[0] and kind == "rename_failure" and not fired[0]:
                self.fail("rename failure")
            super().replace(src, dst)

    recorder = K.Recorder(fixture.proto, fixture.proto_sha, fixture.sched, fixture.ctx,
                          fixture.ctx.opener(binary), {S.D.QWEN: S.Model(fixture.replies())}, str(out),
                          "scripted completed draw preservation", state_fn=S.Health(clock), utc=clock,
                          now=S.Monotonic(), run_id=S.RUN_ID, provenance=S.SYNTHETIC, fs=FS())

    def hook(b, p, ctx):
        if b == "persist:ledger" and p == "before":
            armed[0] = True
        if b == "persist:ledger" and p == "after":
            armed[0] = False
        if kind == "ledger_commit_failure" and b == "persist:manifest" and p == "file_fsync" \
                and ctx.get("event") == "commit_draw" and not fired[0]:
            fired[0] = True
            raise OSError(errno.EIO, "completed draw ledger commit failure")
        if kind == "error_handling_failure":
            if b == "draw_execute" and p == "after" and not fired[0]:
                fired[0] = True
                raise RuntimeError("completed draw before ledger handoff")
            if recorder.stopping and b == "persist:ledger" and p == "file_fsync":
                raise OSError(errno.EIO, "completed draw failure during error handling")
    recorder.hook = hook
    captured = []

    def tracer(frame, event, arg):
        if event == "line" and frame.f_code is K.H.run_draw.__code__ \
                and linecache.getline(frame.f_code.co_filename, frame.f_lineno).strip() == "return out" \
                and not fired[0]:
            fired[0] = True
            captured.append(copy.deepcopy(frame.f_locals["out"]))
            raise RuntimeError("completed draw interrupted immediately before return")
        return tracer
    previous = sys.gettrace()
    if kind == "interruption_before_return":
        sys.settrace(tracer)
    try:
        result = S.attempt(recorder.run)
    finally:
        sys.settrace(previous)
    verdict = fixture.classify(str(out))
    live = out / "live"
    rows = json.loads((live / "draws.json").read_text())
    retained = list(rows)
    pending = []
    failures = []
    for p in live.iterdir():
        if p.name not in ("pending-draw.json", "persistence-failure.json") and not p.name.startswith(".tmp-"):
            continue
        try:
            doc = json.loads(p.read_bytes())
        except (ValueError, UnicodeDecodeError):
            continue
        candidates = doc if isinstance(doc, list) else [doc.get("draw"), doc.get("pending")]
        pending.extend(d for d in candidates if isinstance(d, dict) and d.get("status") == "completed")
        if p.name == "persistence-failure.json":
            failures.append(doc)
    retained.extend(pending)
    complete = {d["seq"]: d for d in retained if d.get("status") == "completed"}
    agree = fired[0] and 1 in complete and bool(complete[1]["record"]) and bool(complete[1]["calls"])
    # Independently replay the retained completed result, including a pending row of an INVALID archive.
    replay = False
    if 1 in complete:
        row = complete[1]
        pool = fixture.ctx.opener(binary)
        try:
            K.H.replay_draw(row, fixture.ctx.by_id[row["context"]], fixture.ctx.settings[row["model"]],
                            pool, fixture.ctx.memory(fixture.ctx.by_id[row["context"]], row["model"]))
            replay = True
        finally:
            pool.close()
    agree = agree and replay and (verdict["state"] == "PARTIAL" if kind == "interruption_before_return"
                                 else verdict["state"] == "INVALID" and not (live / K.SEAL).exists())
    if captured:
        agree = agree and K.canon(complete[1]) == K.canon(captured[0])
    return {"case": kind, "binding": "B-HANDOFF-001", "fired": fired[0], "classification": verdict["state"],
            "recorder": S.recorder_label(result), "completedRetained": len(complete), "pendingCopies": len(pending),
            "journalDispatches": sum(r["record"] == "dispatch" for r in S.journal(str(out))),
            "failedCandidateRetained": any(p.name.startswith(".tmp-") for p in live.iterdir()),
            "completedReplay": replay, "failureEvidence": failures, "agree": bool(agree)}


# --- model-response handoff faults -------------------------------------------------------

def response_case(binary, scratch, op, retention):
    """op: the failing operation of the first returned-outcome append (write,
    fsync, close); retention: ok, rename (the failure report's rename fails) or
    none (every write after the fault fails)."""
    fixture = S.Fixture(binary)
    out = Path(scratch) / f"response-{op}-retention-{retention}"
    state = {"fired": False, "active": False, "invocations": 0}

    class FS(K.OsFS):
        def _after(self):
            if state["fired"] and retention == "none":
                raise OSError(errno.EIO, "every write after the fault fails")

        def mkstemp(self, directory, prefix):
            self._after()
            return super().mkstemp(directory, prefix)

        def write(self, fd, data):
            self._after()
            state["active"] = b'"record":"returned"' in data and not state["fired"]
            if op == "write" and state["active"]:
                state["fired"] = True
                raise OSError(errno.EIO, "outcome write fault")
            return super().write(fd, data)

        def fsync(self, fd):
            self._after()
            if op == "fsync" and state["active"] and not state["fired"]:
                state["fired"] = True
                raise OSError(errno.EIO, "outcome fsync fault")
            return super().fsync(fd)

        def close(self, fd):
            super().close(fd)
            if op == "close" and state["active"] and not state["fired"]:
                state["fired"] = True
                raise OSError(errno.EIO, "outcome close fault")

        def replace(self, src, dst):
            self._after()
            if retention == "rename" and state["fired"] and os.path.basename(dst) == "persistence-failure.json":
                raise OSError(errno.EIO, "failure report rename fault")
            return super().replace(src, dst)

    real = S.Model([MARKER] + fixture.replies())

    def model(payload):
        state["invocations"] += 1
        return real(payload)
    clock = S.Clock()
    rec = K.Recorder(fixture.proto, fixture.proto_sha, fixture.sched, fixture.ctx, fixture.ctx.opener(binary),
                     {S.D.QWEN: model}, str(out), "scripted response handoff", state_fn=S.Health(clock), utc=clock,
                     now=S.Monotonic(), run_id=S.RUN_ID, provenance=S.SYNTHETIC, fs=FS())
    result = S.attempt(rec.run)
    return response_observation(fixture, out, result, state, "response-" + op + "-retention-" + retention)


def response_observation(fixture, out, result, state, name):
    live = Path(out) / "live"
    journal = S.journal(str(out)) or []
    in_journal = any(r.get("record") == "returned" and MARKER in (r.get("body") or "") for r in journal)
    in_report = False
    if (live / "persistence-failure.json").exists():
        report = json.loads((live / "persistence-failure.json").read_bytes())
        in_report = any(MARKER in json.dumps(x) for x in report.get("receivedResponses") or [])
    in_error = MARKER in json.dumps((result.get("retainedEvidence") or {}).get("receivedResponses") or [])
    verdict = fixture.classify(str(out))
    try:
        a = fixture.analyze(str(out))
        accepted = True
    except Exception:  # noqa: BLE001
        a, accepted = None, False
    dispatched = sum(r.get("record") == "dispatch" for r in journal)
    agree = bool(state["fired"] and (in_journal or in_report or in_error) and state["invocations"] == dispatched
                 and (not accepted or in_journal))
    return {"case": name, "fired": state["fired"], "recorder": S.recorder_label(result),
            "classification": verdict["state"], "fullAnalysisAccepted": accepted,
            "responseIn": [k for k, v in (("calls.jsonl", in_journal), ("persistence-failure.json", in_report),
                                          ("RecordingError.retainedEvidence", in_error)) if v],
            "transportInvocations": state["invocations"], "journalDispatches": dispatched, "agree": agree}


# --- exhaustive boundary enumeration ------------------------------------------------------

def _site(frame):
    code = frame.f_code
    return f"{os.path.basename(code.co_filename)}:{code.co_qualname}:{frame.f_lineno}"


class Window:
    """Line events inside an ownership window. kind draw: from run_draw's first
    line with a completed result for draw 1 until draw 1's ledger commit is
    durable and its handoff cleared; kind response: from the first model
    response's receipt until its outcome record is durable."""

    def __init__(self, kind, inject=None):
        self.kind, self.inject = kind, inject
        self.recorder, self.open, self.closed, self.sites, self.counts = None, False, False, [], {}
        self.fired, self.captured = False, None

    def _opens(self, frame):
        r = self.recorder
        if self.kind == "draw":
            if frame.f_code is K.H.run_draw.__code__:
                out = frame.f_locals.get("out") or {}
                return out.get("seq") == 1 and out.get("status") == "completed" \
                    and out.get("verificationStatus") == "RECORDED" and out.get("digestsAfter") is not None
            return False
        return r is not None and r.received is not None and r.calls_k == 1 \
            and os.path.abspath(frame.f_code.co_filename) == RECORDER

    def _closes(self):
        r = self.recorder
        if self.kind == "draw":
            return bool(r.rows) and r.rows[0].get("status") == "completed" and r.pending is None and r.handoff is None
        return r.received is None and r.calls_records >= 2

    def __call__(self, frame, event, arg):
        if self.closed or self.fired:
            return None
        if event != "line":
            return self
        f = os.path.abspath(frame.f_code.co_filename)
        if f not in (RECORDER, DRAWFILE):
            return self
        if not self.open:
            if not self._opens(frame):
                return self
            self.open = True
        elif self._closes():
            self.closed = True
            return None
        if f == DRAWFILE and frame.f_code is not K.H.run_draw.__code__:
            return self
        site = _site(frame)
        n = self.counts[site] = self.counts.get(site, 0) + 1
        self.sites.append([site, n])
        if self.inject is not None and self.inject == [site, n]:
            self.fired = True
            if self.kind == "draw" and self.recorder is not None:
                self.captured = True
            raise RuntimeError(f"boundary injection at {site}#{n}")
        return self


def _boundary_run(binary, out, window, replies=None):
    fixture = S.Fixture(binary)
    state = {"invocations": 0}
    real = S.Model(replies if replies is not None else fixture.replies())

    def model(payload):
        state["invocations"] += 1
        return real(payload)
    clock = S.Clock()
    rec = K.Recorder(fixture.proto, fixture.proto_sha, fixture.sched, fixture.ctx, fixture.ctx.opener(binary),
                     {S.D.QWEN: model}, str(out), "scripted ownership boundary", state_fn=S.Health(clock), utc=clock,
                     now=S.Monotonic(), run_id=S.RUN_ID, provenance=S.SYNTHETIC)
    window.recorder = rec
    previous = sys.gettrace()
    sys.settrace(window)
    try:
        result = S.attempt(rec.run)
    finally:
        sys.settrace(previous)
    return fixture, result, state


def _completed_anywhere(out, result):
    live = Path(out) / "live"
    found = {}
    for path in sorted(live.iterdir()) if live.exists() else []:
        try:
            value = json.loads(path.read_bytes())
        except (ValueError, UnicodeDecodeError, IsADirectoryError):
            continue
        rows = value if isinstance(value, list) else [value.get("draw"), value.get("pending")] if isinstance(value, dict) else []
        for row in rows:
            if isinstance(row, dict) and row.get("status") == "completed" and row.get("seq") == 1:
                name = ("ledger" if path.name == "draws.json" else "failed candidate" if path.name.startswith(".tmp-")
                        else "failure report candidate" if path.name.startswith(".failure-") else path.name)
                found.setdefault(K.canon(row), set()).add(name)
    pending = (result.get("retainedEvidence") or {}).get("pending")
    if isinstance(pending, dict) and pending.get("status") == "completed" and pending.get("seq") == 1:
        found.setdefault(K.canon(pending), set()).add("RecordingError.retainedEvidence")
    return found


def _boundary_case(args):
    binary, scratch, kind, site, want = args
    out = Path(scratch) / f"{kind}-{site[0].replace(':', '_').replace('.', '_').replace('<', '').replace('>', '')}-{site[1]}"
    shutil.rmtree(out, ignore_errors=True)
    window = Window(kind, inject=site)
    try:
        fixture, result, state = _boundary_run(binary, out, window, [MARKER] + S.Fixture(binary).replies()
                                               if kind == "response" else None)
        verdict = fixture.classify(str(out))
        try:
            a = fixture.analyze(str(out))
            accepted, completed = True, len(a["draws"])
        except Exception:  # noqa: BLE001
            accepted, completed = False, 0
        row = {"kind": kind, "site": site[0], "occurrence": site[1], "fired": window.fired,
               "recorder": S.recorder_label(result), "classification": verdict["state"], "fullAnalysisAccepted": accepted}
        if kind == "draw":
            found = _completed_anywhere(out, result)
            exact = sorted(found.get(want, ()))
            replayed = False
            if exact:
                r = json.loads(want)
                pool = fixture.ctx.opener(binary)
                try:
                    c = fixture.ctx.by_id[r["context"]]
                    K.H.replay_draw(r, c, fixture.ctx.settings[r["model"]], pool, fixture.ctx.memory(c, r["model"]))
                    replayed = True
                except Exception:  # noqa: BLE001
                    replayed = False
                finally:
                    pool.close()
            # An injection on run_draw's own session-close line is a genuine close failure: the frozen
            # draw machinery records the finished workflow as an aborted engine_close draw that keeps
            # every result field; full analysis replays it.
            closed = False
            ab_path = out / "live" / "aborted-draw.json"
            if not exact and ab_path.exists():
                ab, r = json.loads(ab_path.read_text()), json.loads(want)
                closed = ab.get("seq") == 1 and ab.get("failureStage") == "engine_close" and all(
                    K.canon(ab.get(k)) == K.canon(r.get(k)) for k in (
                        "record", "batches", "engine", "calls", "initialDigests", "digestsAfter",
                        "submittedRepresentation", "submittedRepresentationSha256", "process"))
            row.update(completedRetainedIn=exact + (["aborted-draw.json (engine_close)"] if closed else []),
                       completedReplayed=replayed or (closed and accepted),
                       agree=bool(window.fired and ((exact and replayed and (not accepted or completed >= 1))
                                                    or (closed and accepted))))
        else:
            obs = response_observation(fixture, out, result, {"fired": window.fired, "invocations": state["invocations"]}, "")
            row.update(responseRetainedIn=obs["responseIn"], transportInvocations=obs["transportInvocations"],
                       journalDispatches=obs["journalDispatches"], agree=obs["agree"])
    except BaseException as x:  # noqa: BLE001  (a harness failure is a disagreement)
        row = {"kind": kind, "site": site[0], "occurrence": site[1], "harnessError": f"{type(x).__name__}: {x}",
               "agree": False}
    finally:
        shutil.rmtree(out, ignore_errors=True)
    return row


def boundaries(binary, scratch, jobs):
    rows = []
    for kind in ("draw", "response"):
        base = Path(scratch) / f"discover-{kind}"
        shutil.rmtree(base, ignore_errors=True)
        window = Window(kind)
        fixture = S.Fixture(binary)
        _, result, _ = _boundary_run(binary, base, window, [MARKER] + fixture.replies() if kind == "response" else None)
        want = None
        if kind == "draw":
            want = K.canon(json.loads((base / "live" / "draws.json").read_text())[0])
        if not window.sites or not window.closed:
            raise SystemExit(f"the {kind} ownership window was not observed")
        args = [(binary, scratch, kind, site, want) for site in window.sites]
        with concurrent.futures.ProcessPoolExecutor(max_workers=jobs) as pool:
            rows.extend(pool.map(_boundary_case, args))
        shutil.rmtree(base, ignore_errors=True)
    return rows


def run(binary, scratch, jobs=1):
    Path(scratch).mkdir(parents=True, exist_ok=True)
    cases = [one(binary, scratch, k) for k in INHERITED]
    responses = [response_case(binary, scratch, op, retention) for op in ("write", "fsync", "close")
                 for retention in ("ok", "rename", "none")]
    sites = boundaries(binary, scratch, jobs)
    by = {k: [r for r in sites if r["kind"] == k] for k in ("draw", "response")}
    return {"schema": "dreamteam.v13l-completed-preservation/v2", "cases": cases, "responseHandoff": responses,
            "boundaries": sites,
            "summary": {"cases": len(cases), "agree": sum(c["agree"] for c in cases),
                        "responseCases": len(responses), "responseAgree": sum(c["agree"] for c in responses),
                        "drawBoundaries": len(by["draw"]), "drawBoundariesAgree": sum(r["agree"] for r in by["draw"]),
                        "responseBoundaries": len(by["response"]),
                        "responseBoundariesAgree": sum(r["agree"] for r in by["response"])}}


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--binary", required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--jobs", type=int, default=1)
    a = ap.parse_args()
    doc = run(os.path.abspath(a.binary), os.path.abspath(a.scratch), a.jobs)
    # Full pending row data are retained in scratch; the committed summary keeps the exact errors and counts.
    for case in doc["cases"]:
        for report in case["failureEvidence"]:
            row = report.pop("pending", None)
            report["pendingRowSha256"] = K.sha256(K.canon(row)) if row else None
            report["candidates"] = ["failed ledger candidate" for _ in report["candidates"]]
    K.write_json(a.out, doc)
    print(json.dumps(doc["summary"]))
    if not all(c["agree"] for c in doc["cases"] + doc["responseHandoff"] + doc["boundaries"]):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
