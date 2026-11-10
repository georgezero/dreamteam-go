#!/usr/bin/env python3
"""Contradictory archives (DT13I-01..06 and the per-stage, accounting, seal,
interrupt, health and timestamp rules): genuine V0.13l archives are recorded,
then each is altered the way a careless or deliberate forger could, and every
dependent value the forger could recompute is recomputed: commit bindings,
the failure-event mirror, and the completion seal's every hash, count and
binding. Verification therefore cannot rely on a checksum mismatch; only the
targeted semantic contradiction remains. Every mutant must classify INVALID
with a targeted invariant among its violations and analysis must refuse it.
Positive controls (changes the contract permits) must stay accepted.

    python3 -I -B analysis/alternatives/v13l_mutations.py --binary ./dt --scratch DIR --out FILE
"""
import argparse
import base64
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import sys

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13l_scenarios as S  # noqa: E402

SCHEMA = "dreamteam.v13l-mutations/v1"
D, J = S.D, S.J


class At:
    def __init__(self, boundary, phase, n=1, action="signal"):
        self.key, self.n, self.action, self.seen = (boundary, phase), n, action, 0

    def __call__(self, boundary, phase, ctx):
        if (boundary, phase) == self.key:
            self.seen += 1
            if self.seen == self.n and self.action == "signal":
                os.kill(os.getpid(), signal.SIGINT)


def unrecorded_runner(entry, c, settings, opener, mem, transport, limits, utc=None, now=None):
    """Lose the draw artifact after a genuine workflow dispatch and outcome."""
    def lose_result(payload):
        transport(payload)
        raise RuntimeError("the draw raised without evidence after one dispatched call")
    try:
        J.H.run_draw(entry, c, settings, opener, mem, lose_result, limits, utc=utc, now=now)
    except RuntimeError as error:
        if hasattr(error, "partialDraw"):
            del error.partialDraw
        raise



def record_bases(binary, scratch):
    bases = {}
    both = S.Fixture(binary, arms=("control", "treatment"))
    one = S.Fixture(binary)

    def rec(name, fixture, modes=None, **kw):
        out = os.path.join(scratch, "base-" + name)
        shutil.rmtree(out, ignore_errors=True)
        clock = S.Clock()
        fixture.record(out, clock=clock, health=S.Health(clock, modes=modes), **kw)
        v = fixture.classify(out)
        if v["state"] == "INVALID":
            raise SystemExit(f"base {name} is not valid: {v['violations']}")
        bases[name] = (fixture, out, v["state"])
    rec("complete", both)
    rec("health_stop", one, modes={1: "refused"})
    calls = [0]

    def second_call_fails(payload):
        calls[0] += 1
        if calls[0] == 2:
            return None, "", "timeout: scripted"
        if calls[0] == 3:
            raise RuntimeError("scripted transport failure after a failed exchange")
        return D.scripted([S.F.ABANDON] * 4)(payload)
    rec("aborted_workflow", one, model=second_call_fails)
    rec("halt_row", one.with_limits(maxModelCallsPerDraw=0))
    rec("cleanup_failed", one, opener=S.FailingOpener(one.ctx.opener(binary),
                                                      close_error=RuntimeError("scripted pool release failure")))
    rec("engine_open", both, opener=S.FailingOpener(both.ctx.opener(binary), RuntimeError("open failed"), at=1))
    rec("operator_dispatch", one, model=S.Model([None, S.F.ABANDON, S.F.ABANDON]),
        hook=At("safe_point", "after_call_persisted", 1))
    rec("operator_between", one, hook=At("safe_point", "between_draws", 2))
    rec("after_schedule", one, hook=At("safe_point", "before_cleanup", 1))
    rec("unrecorded", one, draw_runner=unrecorded_runner)
    return bases


def body(raw):
    return {"bytes": len(raw), "sha256": hashlib.sha256(raw).hexdigest(), "base64": base64.b64encode(raw).decode()}


class Arc:
    """An archive opened for forgery; write() recomputes every dependent value."""

    def __init__(self, out):
        self.live = Path(out) / "live"
        self.files = {p.name: json.loads(p.read_text()) for p in self.live.iterdir()
                      if p.suffix == ".json" and p.name != J.SEAL}
        self.calls = [json.loads(x) for x in (self.live / J.CALLS).read_text().splitlines() if x]
        self.seal = json.loads((self.live / J.SEAL).read_text()) if (self.live / J.SEAL).exists() else None

    @property
    def doc(self):
        return self.files["run.json"]

    @property
    def rows(self):
        return self.files["draws.json"]

    def write(self, commits=True, event=True, reseal=True, counters=False):
        doc, rows = self.doc, self.rows
        if commits:
            k = 0
            for e in doc["journal"]:
                if e.get("event") == "commit_draw" and k < len(rows):
                    e.update(rowSha256=D.sha256(D.canon(rows[k])), modelCalls=len(rows[k]["calls"]),
                             status=rows[k].get("status"), draw=rows[k].get("seq"))
                    k += 1
        if counters:
            doc["counters"]["modelCalls"] = sum(r["record"] == "dispatch" for r in self.calls)
        if event and doc.get("failureEvent") is not None:
            self.files["failure-event.json"] = copy.deepcopy(doc["failureEvent"])
        for name, value in self.files.items():
            if name != "run.json":
                (self.live / name).write_bytes(J.encode_json(value))
        (self.live / J.CALLS).write_bytes(b"".join(J.encode_line(r) for r in self.calls))
        (self.live / "run.json").write_bytes(J.encode_json(doc))
        if reseal and self.seal is not None:
            self.reseal()
        if self.seal is not None:
            (self.live / J.SEAL).write_bytes(J.encode_json(self.seal))

    def reseal(self):
        s, doc, rows = self.seal, self.doc, self.rows

        def h(n):
            p = self.live / n
            return hashlib.sha256(p.read_bytes()).hexdigest() if p.exists() else None
        s["manifest"] = {"file": "run.json", "sha256": h("run.json"), "journalEntries": len(doc["journal"])}
        s["ledger"] = {"file": "draws.json", "sha256": h("draws.json"), "rows": len(rows),
                       "completed": sum(r.get("status") == "completed" for r in rows),
                       "rowSha256": [D.sha256(D.canon(r)) for r in rows]}
        s["calls"] = {"file": J.CALLS, "sha256": h(J.CALLS), "records": len(self.calls),
                      "dispatched": sum(r.get("record") == "dispatch" for r in self.calls)}
        s["health"] = {"checks": len(doc["healthChecks"]), "checkSha256": [D.sha256(D.canon(c)) for c in doc["healthChecks"]],
                       "serverBefore": h("server-before.json")}
        s["sideArtifacts"] = {n: h(n) for n in J.SIDE}
        s["counters"] = copy.deepcopy(doc.get("counters"))
        s["transition"]["n"] = len(doc["journal"]) + 1


def health(a, k=0):
    return a.doc["healthChecks"][k]["serverState"]["health"]


def mirror_before(a):
    a.files["server-before.json"] = copy.deepcopy(a.doc["healthChecks"][0]["serverState"])


def set_initial_health(a, raw, category, outcome):
    health(a).update(body=body(raw), category=category, outcome=outcome)
    mirror_before(a)


def first(a, event):
    return next(e for e in a.doc["journal"] if e["event"] == event)


def walk(o, fn):
    if isinstance(o, dict):
        return {k: walk(v, fn) for k, v in o.items()}
    if isinstance(o, list):
        return [walk(v, fn) for v in o]
    return fn(o)


def all_times(a, fn):
    for name in list(a.files):
        a.files[name] = walk(a.files[name], lambda x: fn(x) if isinstance(x, str) and S.LS.parse_timestamp(x) else x)
    a.calls = [walk(r, lambda x: fn(x) if isinstance(x, str) and S.LS.parse_timestamp(x) else x) for r in a.calls]
    a.seal["transition"]["timestamp"] = fn(a.seal["transition"]["timestamp"])


def m(name, base, expect, fn, source="V0.13l", accept=False, replay=False, **write):
    """expect: invariants of which at least one must be violated. replay: a
    consistent rewrite that only model-free replay can expose (the draw's
    engine log and decision record still reflect the erased exchange); it
    must be refused by analysis."""
    return {"name": name, "base": base, "expect": set(expect), "fn": fn, "write": write, "source": source,
            "accept": accept, "replay": replay}


def complete_row(bases):
    return json.loads((Path(bases["complete"][1]) / "live/draws.json").read_text())[0]


def duplicate_health(a):
    """A second valid round-health visit before the same draw, every
    timestamp, index and counter consistent."""
    j = a.doc["journal"]
    ix = next(i for i, e in enumerate(j) if e["event"] == "begin_health_check")
    old = j[ix:ix + 2]
    new = copy.deepcopy(old)
    for e in j[ix + 2:]:
        if "healthCheck" in e:
            e["healthCheck"] += 1
    for e in new:
        e["healthCheck"] += 1
    j[ix + 2:ix + 2] = new
    for n, e in enumerate(j, 1):
        e["n"] = n
    ts = new[-1]["timestamp"]
    for e in old + new:
        e["timestamp"] = ts
    checks = a.doc["healthChecks"]
    checks.insert(2, copy.deepcopy(checks[1]))
    for k, c in enumerate(checks):
        c["index"] = k
    for c in checks[1:3]:
        h = c["serverState"]["health"]
        h["started"] = h["ended"] = ts
    a.doc["counters"]["healthChecks"] += 1


def premature_schedule_done(a):
    """Only the first of two draws, then schedule_done and the cleanup failure."""
    d = a.doc
    a.files["draws.json"] = a.rows[:1]
    j = d["journal"]
    ix = next(i for i, e in enumerate(j) if e["event"] == "continue")
    j[ix:] = [dict(j[ix], event="schedule_done", to="CLEANUP"), j[-1]]
    for n, e in enumerate(j, 1):
        e["n"] = n
    d["healthChecks"] = d["healthChecks"][:2]
    d["counters"].update(drawsCompleted=1, drawsNotRun=1, modelCalls=1, healthChecks=2)
    d["failureEvent"].update(drawsCompleted=1, modelCalls=1, nextDraw=a_sched(a)[1])
    a.calls = [r for r in a.calls if r["draw"] == 1]
    d["cleanup"]["timestamp"] = j[-1]["timestamp"]


def a_sched(a):
    return [{"seq": i, "model": D.QWEN, "context": "S1-a", "arm": "control", "draw": i} for i in (1, 2)]


def relabel_exception(a, typ, reason=None, kind=None, category=None):
    h = health(a)
    h["exception"] = {"type": typ, "reasonType": reason, "reasonKind": kind}
    if category:
        h["category"] = category
    mirror_before_if_any(a)


def mirror_before_if_any(a):
    if "server-before.json" in a.files and a.doc["healthChecks"][0]["serverState"] is not None:
        mirror_before(a)


def drop_call_everywhere(a):
    r = a.rows[0]
    r["calls"] = []
    r["modelCallsCompleted"] = 0
    a.calls = [x for x in a.calls if x["draw"] != r["seq"]]
    for n, x in enumerate(a.calls):
        pass
    k = 0
    for x in a.calls:
        if x["record"] == "dispatch":
            k += 1
        x["k"] = k
    a.doc["counters"]["modelCalls"] -= 1


def unknown_as_zero(a):
    ab = a.files["aborted-draw.json"]
    ab.update(calls=[], modelCallsCompleted=0, engine=[])


def erased_journal_count_kept(a):
    """Remove the unrecorded draw's journal entries; the counts still say one call."""
    a.calls = [x for x in a.calls if x["draw"] != a.files["aborted-draw.json"]["seq"]]


def stop_operator_relabel(a):
    a.doc["stop"].update(code="operator", failure=None, safePoint="between_draws")
    a.doc["failureEvent"]["stop"] = copy.deepcopy(a.doc["stop"])
    first(a, "stop").update(code="operator")


MUTATIONS = [
    # --- DT13I-04: the independent review's accepted contradictions -------------------------------
    m("engine_open_with_finished_record", "engine_open", {"L06"},
      lambda a, b: a.files["aborted-draw.json"].update({k: copy.deepcopy(complete_row(b)[k]) for k in (
          "record", "process", "digestsAfter", "submittedRepresentation", "submittedRepresentationSha256")}),
      "DT13I-04 forgeries engine_open_with_finished_record"),
    m("engine_open_with_transport_halt", "engine_open", {"L06"},
      lambda a, b: a.files["aborted-draw.json"].update(halt={"code": "transport", "message": "five consecutive failures",
                                                            "evidence": {"stage": "request_start", "drawCalls": 5,
                                                                         "runCalls": 5, "elapsedSeconds": 0,
                                                                         "consecutiveTransportFailures": 5}}),
      "DT13I-04 forgeries engine_open_with_transport_halt"),
    m("premature_schedule_done_cleanup_failure", "cleanup_failed", {"L14", "L03"}, lambda a, b: premature_schedule_done(a),
      "DT13I-04 forgeries premature_schedule_done_cleanup_failure"),
    m("duplicate_round_health", "complete", {"L14", "L07"}, lambda a, b: duplicate_health(a),
      "DT13I-04 extra-forgeries duplicate_round_health"),
    m("null_failure_event_file", "complete", {"L16"}, lambda a, b: a.files.__setitem__("failure-event.json", None),
      "DT13I-04 forgeries null_failure-event.json"),
    m("null_aborted_draw_file", "complete", {"L16"}, lambda a, b: a.files.__setitem__("aborted-draw.json", None),
      "DT13I-04 forgeries null_aborted-draw.json"),
    m("null_server_before_file", "health_stop", {"L16"}, lambda a, b: a.files.__setitem__("server-before.json", None)),
    # --- DT13I-03: exception names never trusted -----------------------------------------------------
    m("os_error_named_KeyboardInterrupt", "health_stop", {"L07"},
      lambda a, b: relabel_exception(a, "builtins.KeyboardInterrupt", category="network_os_error"),
      "DT13I-03 forgeries os_error_KeyboardInterrupt"),
    m("os_error_named_ValueError", "health_stop", {"L07"},
      lambda a, b: relabel_exception(a, "builtins.ValueError", category="network_os_error"),
      "DT13I-03 forgeries os_error_ValueError"),
    m("os_error_named_invented", "health_stop", {"L07"},
      lambda a, b: relabel_exception(a, "builtins.MadeUpNonOSError", category="network_os_error"),
      "DT13I-03 forgeries os_error_MadeUpNonOSError"),
    m("unqualified_type_name", "health_stop", {"L07"},
      lambda a, b: relabel_exception(a, "URLError", "ConnectionRefusedError", "exception")),
    m("wrapped_reason_in_wrong_family", "health_stop", {"L07"},
      lambda a, b: relabel_exception(a, "urllib.error.URLError", "http.client.IncompleteRead", "exception",
                                     category="timeout")),
    m("url_error_with_exception_reason", "health_stop", {"L07"},
      lambda a, b: relabel_exception(a, "urllib.error.URLError", "builtins.ConnectionRefusedError", "exception",
                                     category="url_error")),
    m("wrapped_keyboard_interrupt", "health_stop", {"L07"},
      lambda a, b: relabel_exception(a, "urllib.error.URLError", "builtins.KeyboardInterrupt", "exception",
                                     category="network_os_error")),
    m("http_status_without_status", "health_stop", {"L07"},
      lambda a, b: relabel_exception(a, "urllib.error.HTTPError", category="http_status")),
    # --- DT13I-06: timestamp grammar and order -------------------------------------------------------
    m("all_times_trailing_newline", "complete", {"L13", "L07", "L20"}, lambda a, b: all_times(a, lambda s: s + "\n"),
      "DT13I-06 extra-forgeries all_times_trailing_newline"),
    m("all_times_arabic_indic_digits", "complete", {"L13", "L07", "L20"},
      lambda a, b: all_times(a, lambda s: s.translate(str.maketrans("0123456789", "٠١٢٣٤٥٦٧٨٩"))),
      "DT13I-06 extra-forgeries all_times_unicode_digits"),
    m("all_times_fullwidth_digits", "complete", {"L13", "L07", "L20"},
      lambda a, b: all_times(a, lambda s: s.translate(str.maketrans("0123456789", "０１２３４５６７８９")))),
    m("timestamp_non_utc_offset", "complete", {"L13"},
      lambda a, b: (a.doc.update(started="2026-01-01T00:00:00-05:00"),
                    a.doc["journal"][0].update(timestamp="2026-01-01T00:00:00-05:00")), "DT13I-06 forgeries timestamp_non_utc"),
    m("timestamp_trailing_newline_single", "complete", {"L13"},
      lambda a, b: a.doc["journal"][3].update(timestamp=a.doc["journal"][3]["timestamp"] + "\n"),
      "DT13I-06 forgeries timestamp_trailing_newline"),
    m("timestamp_null", "complete", {"L13"}, lambda a, b: a.doc.update(started=None), "DT13I-06 forgeries timestamp_null"),
    m("impossible_calendar_timestamp", "complete", {"L13"}, lambda a, b: a.doc["journal"][3].update(timestamp="2026-02-30T00:00:00Z")),
    m("journal_time_reversed", "complete", {"L13"}, lambda a, b: a.doc["journal"][3].update(timestamp="2025-01-01T00:00:00Z")),
    m("seal_time_before_journal", "complete", {"S04"}, lambda a, b: a.seal["transition"].update(timestamp="2025-01-01T00:00:00Z")),
    m("observation_end_before_start", "complete", {"L07"},
      lambda a, b: (health(a).update(ended="2025-01-01T00:00:00Z"), mirror_before(a))),
    m("call_sent_before_dispatch", "complete", {"L13"},
      lambda a, b: a.rows[0]["calls"][0].update(sent=a.doc["journal"][0]["timestamp"])),
    m("call_sent_outside_draw", "complete", {"L13"}, lambda a, b: a.rows[0]["calls"][0].update(sent="2030-01-01T00:00:00Z")),
    m("call_sent_missing", "complete", {"L13"}, lambda a, b: a.rows[0]["calls"][0].update(sent=None)),
    m("dispatch_time_outside_draw", "complete", {"L13"},
      lambda a, b: a.calls[0].update(timestamp="2030-01-01T00:00:00Z")),
    m("interrupt_event_time_garbled", "after_schedule", {"L13", "L20"},
      lambda a, b: a.doc["interrupts"]["events"][0].update(timestamp="2026-01-01T00:00:00Z\n")),
    m("valid_call_metadata_shift", "complete", set(), lambda a, b: [
        c.update(sent=x["timestamp"]) for r in a.rows for c, x in zip(r["calls"], [y for y in a.calls
                                                                                if y["record"] == "dispatch" and y["draw"] == r["seq"]])],
      "DT13I-06 extra-forgeries valid_call_metadata_shift (positive control)", accept=True),
    # --- DT13I-01: no lost or fabricated calls ----------------------------------------------------------
    m("journal_dispatch_dropped", "complete", {"L04"},
      lambda a, b: setattr(a, "calls", [x for x in a.calls if x["k"] != 1]), "DT13I-01 known call missing from the journal"),
    m("journal_outcome_dropped", "complete", {"L04"},
      lambda a, b: setattr(a, "calls", [x for x in a.calls if not (x["k"] == 2 and x["record"] != "dispatch")])),
    m("journal_extra_dispatch", "complete", {"L04", "L03"},
      lambda a, b: a.calls.extend([dict(a.calls[-2], k=3, n=2), dict(a.calls[-1], k=3, n=2)])),
    m("journal_response_altered", "complete", {"R-RESPONSE-001"},
      lambda a, b: a.calls[1].update(body="{}", responseSha256=D.sha256("{}"))),
    m("journal_record_reordered", "complete", {"L04"}, lambda a, b: a.calls.__setitem__(slice(0, 2), a.calls[1::-1])),
    m("journal_raised_recorded_as_returned", "aborted_workflow", {"R-RESPONSE-001"},
      lambda a, b: a.calls.__setitem__(-1, {k: v for k, v in dict(a.calls[-1], record="returned", status=200, error=None,
                                                                    body="", responseSha256=D.sha256("")).items()
                                            if k != "exception"})),
    m("call_dropped_from_row_and_journal", "complete", {"L04", "L02", "L03"}, lambda a, b: drop_call_everywhere(a),
      "DT13I-01 a completed call erased everywhere, counts recomputed", replay=True),
    m("unrecorded_draw_reported_as_zero_calls", "unrecorded", {"L06", "L04"}, lambda a, b: unknown_as_zero(a),
      "DT13I-01 unknown calls reported as known zero"),
    m("unrecorded_draw_journal_erased", "unrecorded", {"L03", "L19"}, lambda a, b: erased_journal_count_kept(a),
      "DT13I-01 dispatched calls erased from the journal"),
    m("counter_modelcalls_from_rows_only", "aborted_workflow", {"L03"},
      lambda a, b: a.doc["counters"].update(modelCalls=a.doc["counters"]["modelCalls"] - 1)),
    m("torn_call_journal", "complete", {"L04", "L16"}, lambda a, b: None, tear=True),
    # --- seal ------------------------------------------------------------------------------------------
    m("seal_claims_complete_on_partial", "health_stop", {"S04"},
      lambda a, b: a.seal.update(terminalState="COMPLETE", transition=dict(a.seal["transition"], event="finalize_complete",
                                                                          to="COMPLETE"))),
    m("seal_claims_partial_on_complete", "complete", {"S04"},
      lambda a, b: a.seal.update(terminalState="PARTIAL", transition=dict(a.seal["transition"], event="finalize_partial",
                                                                         to="PARTIAL"))),
    m("seal_transition_n_wrong", "complete", {"S04"}, lambda a, b: None, seal_patch={"transition": {"n": 3}}),
    m("seal_run_identity_differs", "complete", {"S02"},
      lambda a, b: a.seal.update(runId="00000000-0000-4000-8000-00000000ffff")),
    m("seal_extra_field", "complete", {"S02"}, lambda a, b: a.seal.update(accepted=True)),
    m("seal_lifecycle_v1", "complete", {"S02"},
      lambda a, b: a.seal["lifecycle"].update(id="dreamteam.run-lifecycle/v1")),
    m("seal_finalization_contract_differs", "complete", {"S02"}, lambda a, b: a.seal.update(finalization="none")),
    m("seal_null_hash_for_present_file", "complete", {"S03"}, lambda a, b: None,
      seal_patch={"sideArtifacts": {"server-before.json": None}}),
    m("seal_manifest_hash_stale", "complete", {"S03"}, lambda a, b: a.doc.update(authorization="changed after sealing"),
      reseal=False),
    m("seal_row_hash_stale", "complete", {"S03"}, lambda a, b: None, seal_patch={"ledger": {"rowSha256": ["0" * 64, "0" * 64]}}),
    m("seal_counters_differ", "complete", {"S03"}, lambda a, b: None, seal_patch={"counters": {"modelCalls": 7}}),
    m("manifest_claims_terminal_state", "complete", {"L14"},
      lambda a, b: (a.doc["journal"].append({"n": len(a.doc["journal"]) + 1, "event": "finalize_complete",
                                             "from": "FINALIZING", "to": "COMPLETE",
                                             "timestamp": a.seal["transition"]["timestamp"]}),
                    a.doc.update(state="COMPLETE"))),
    m("manifest_journal_unfinished", "complete", {"L14", "L08"},
      lambda a, b: (a.doc["journal"].pop(), a.doc.update(state="CLEANUP"))),
    # --- interrupts --------------------------------------------------------------------------------------
    m("operator_stop_without_event", "operator_between", {"L20"},
      lambda a, b: a.doc["interrupts"].update(events=[])),
    m("operator_stop_event_other_point", "operator_between", {"L20"},
      lambda a, b: a.doc["interrupts"]["events"][0].update(safePoint="before_health_check")),
    m("operator_stop_at_note_point", "operator_between", {"L06", "L20"},
      lambda a, b: (a.doc["stop"].update(safePoint="after_call_persisted"),
                    a.doc["failureEvent"]["stop"].update(safePoint="after_call_persisted"),
                    a.doc["interrupts"]["events"][0].update(safePoint="after_call_persisted"))),
    m("after_stop_event_in_complete", "after_schedule", {"L20", "L09"},
      lambda a, b: a.doc["interrupts"]["events"][0].update(disposition="AFTER_STOP")),
    m("stop_event_in_complete", "after_schedule", {"L20", "L09"},
      lambda a, b: a.doc["interrupts"]["events"][0].update(disposition="STOP")),
    m("driver_relabelled_operator", "cleanup_failed", {"L06", "L08", "L20", "L19"},
      lambda a, b: (a.doc["stop"].update(code="operator", safePoint="before_cleanup", failure=None),
                    a.doc["failureEvent"]["stop"].update(code="operator", safePoint="before_cleanup", failure=None))),
    m("operator_relabelled_driver_keyboard", "operator_between", {"L06", "L20"},
      lambda a, b: (a.doc["stop"].update(code="driver", safePoint=None,
                                         failure={"type": "KeyboardInterrupt", "message": ""}),
                    a.doc["failureEvent"]["stop"].update(code="driver", safePoint=None,
                                                         failure={"type": "KeyboardInterrupt", "message": ""}),
                    first(a, "stop").update(code="driver"))),
    m("operator_before_dispatch_without_operatorstop", "operator_dispatch", {"L06"},
      lambda a, b: a.files["aborted-draw.json"]["failure"].update(type="RuntimeError")),
    m("health_stop_relabelled_operator", "health_stop", {"L06", "L20", "L07"}, lambda a, b: stop_operator_relabel(a)),
    # --- per-stage evidence ---------------------------------------------------------------------------------
    m("workflow_with_final_digests", "aborted_workflow", {"L06"},
      lambda a, b: a.files["aborted-draw.json"].update(digestsAfter=copy.deepcopy(a.files["aborted-draw.json"]["initialDigests"]))),
    m("workflow_with_record", "aborted_workflow", {"L06"},
      lambda a, b: a.files["aborted-draw.json"].update(record=copy.deepcopy(complete_row(b)["record"]))),
    m("workflow_process_exposure_flipped", "aborted_workflow", {"L06"},
      lambda a, b: a.files["aborted-draw.json"]["process"].update(
          exposure=not a.files["aborted-draw.json"]["process"]["exposure"])),
    m("engine_open_with_cleanup_failure", "engine_open", {"L06"},
      lambda a, b: a.files["aborted-draw.json"].update(cleanupFailure={"type": "RuntimeError", "message": "x"})),
    m("engine_open_marked_ready", "engine_open", {"L06"},
      lambda a, b: a.files["aborted-draw.json"].update(engineInitialization="READY")),
    m("engine_open_with_engine_log", "engine_open", {"L06"},
      lambda a, b: a.files["aborted-draw.json"].update(engine=[{"op": "digests", "args": None, "answer": None,
                                                                "error": "x"}])),
    m("draw_setup_with_calls", "aborted_workflow", {"L06", "L04"},
      lambda a, b: a.files["aborted-draw.json"].update(failureStage="draw_setup", engineInitialization="NOT_STARTED",
                                                       process=None)),
    m("aborted_stage_unknown", "aborted_workflow", {"L06"},
      lambda a, b: a.files["aborted-draw.json"].update(failureStage="somewhere")),
    m("completed_row_with_cleanup_failure", "complete", {"L02"},
      lambda a, b: a.rows[0].update(cleanupFailure={"type": "RuntimeError", "message": "session close failed"})),
    m("completed_row_process_differs", "complete", {"L02"},
      lambda a, b: a.rows[0]["process"].update(processFailure=True)),
    # --- transitions, ledger, stop, cleanup (L01-L19) --------------------------------------------------------
    m("schedule_done_after_first_draw", "complete", {"L14"},
      lambda a, b: first(a, "continue").update(event="schedule_done", to="CLEANUP")),
    m("continue_after_last_draw", "complete", {"L14"},
      lambda a, b: first(a, "schedule_done").update(event="continue", to="READY")),
    m("begin_draw_without_health_visit", "complete", {"L14", "L07"},
      lambda a, b: [a.doc["journal"].remove(e) for e in [x for x in a.doc["journal"]
                                                           if x["event"] in ("begin_health_check", "health_passed")][2:4]]),
    m("illegal_transition", "complete", {"L14"}, lambda a, b: a.doc["journal"][2].update(event="commit_draw")),
    m("unknown_event", "complete", {"L14"}, lambda a, b: a.doc["journal"][2].update(event="resume")),
    m("unknown_state", "complete", {"L14"}, lambda a, b: a.doc["journal"][2].update(to="WAITING")),
    m("transitions_reordered", "complete", {"L14"},
      lambda a, b: a.doc["journal"].__setitem__(slice(3, 5), list(reversed(a.doc["journal"][3:5])))),
    m("duplicate_completed_draws", "complete", {"L01"}, lambda a, b: a.rows.__setitem__(1, copy.deepcopy(a.rows[0]))),
    m("missing_schedule_entry", "complete", {"L01"}, lambda a, b: a.rows.pop(0)),
    m("truncated_ledger_claiming_complete", "complete", {"L17", "L10", "L04"}, lambda a, b: a.rows.pop()),
    m("aborted_status_row_in_ledger", "complete", {"L01", "L02"}, lambda a, b: a.rows[1].update(status="aborted")),
    m("aborted_artifact_beside_complete", "complete", {"L05", "L16"},
      lambda a, b: a.files.__setitem__("aborted-draw.json", dict(copy.deepcopy(a.rows[1]), status="aborted"))),
    m("stop_code_differs_in_failure_event_file", "health_stop", {"L16"},
      lambda a, b: a.files["failure-event.json"]["stop"].update(code="operator"), event=False),
    m("stop_code_differs_from_journal", "health_stop", {"L19"},
      lambda a, b: (a.doc["stop"].update(code="driver", failure={"type": "X", "message": ""}),
                    a.doc["failureEvent"]["stop"].update(code="driver", failure={"type": "X", "message": ""}))),
    m("second_stop_transition", "health_stop", {"L14", "L19"},
      lambda a, b: a.doc["journal"].insert(4, dict(first(a, "stop"), n=0))),
    m("halt_row_with_driver_stop", "halt_row", {"R-FAILURE-001"},
      lambda a, b: (a.doc["stop"].update(code="driver", evidence=None, failure={"type": "RuntimeError", "message": ""}),
                    a.doc["failureEvent"].update(stop=copy.deepcopy(a.doc["stop"])), first(a, "stop").update(code="driver"))),
    m("failure_event_call_count", "health_stop", {"L19"}, lambda a, b: a.doc["failureEvent"].update(modelCalls=3)),
    m("counters_not_conserved", "health_stop", {"L03"}, lambda a, b: a.doc["counters"].update(drawsNotRun=7)),
    m("complete_with_cleanup_failure", "complete", {"L08"},
      lambda a, b: a.doc["cleanup"].update(status="FAILED", failure={"type": "RuntimeError", "message": "x"})),
    m("cleanup_failure_replaces_primary", "health_stop", {"L19", "L08"},
      lambda a, b: (a.doc["cleanup"].update(status="FAILED", failure={"type": "RuntimeError", "message": "release"}),
                    a.doc["stop"].update(code="driver", state="CLEANUP", failure={"type": "RuntimeError", "message": "release"}))),
    m("cleanup_timestamp_differs", "complete", {"L08"}, lambda a, b: a.doc["cleanup"].update(timestamp=a.doc["started"])),
    m("reconciled_commit_without_stop", "complete", {"L17"}, lambda a, b: first(a, "commit_draw").update(reconciled=True)),
    m("commit_row_hash_stale", "complete", {"L17"},
      lambda a, b: a.rows[0]["record"].update(retryRequests=a.rows[0]["record"]["retryRequests"] + 1), commits=False),
    m("resume_permitted", "complete", {"L18"}, lambda a, b: a.doc.update(resume="allowed")),
    m("run_identity_not_uuid", "complete", {"L15"}, lambda a, b: a.doc.update(runId="run-1")),
    m("lifecycle_contract_differs", "complete", {"L15"}, lambda a, b: a.doc["lifecycle"].update(sha256="0" * 64)),
    m("recording_claims_verification", "complete", {"L15"}, lambda a, b: a.doc.update(verificationStatus="VERIFIED")),
    # --- health evidence ----------------------------------------------------------------------------------------
    m("complete_with_unhealthy_initial_observation", "complete", {"L07"},
      lambda a, b: set_initial_health(a, b'{"status":"unhealthy"}', "unhealthy_status", "UNHEALTHY")),
    m("healthy_label_on_contradictory_body", "complete", {"L07"},
      lambda a, b: set_initial_health(a, b'{"status":"healthy","error":"URLError"}', "healthy", "HEALTHY")),
    m("healthy_with_exception", "complete", {"L07"},
      lambda a, b: (health(a).update(exception={"type": "urllib.error.URLError",
                                                "reasonType": "builtins.ConnectionRefusedError", "reasonKind": "exception"}),
                    mirror_before(a))),
    m("healthy_with_http_503", "complete", {"L07"}, lambda a, b: (health(a).update(httpStatus=503), mirror_before(a))),
    m("healthy_body_hash_wrong", "complete", {"L07"},
      lambda a, b: (health(a)["body"].update(sha256="0" * 64), mirror_before(a))),
    m("health_field_missing", "health_stop", {"L07"},
      lambda a, b: (a.doc["healthChecks"][0]["serverState"].pop("health"), mirror_before(a))),
    m("unknown_read_failure_category", "health_stop", {"L07"},
      lambda a, b: (health(a).update(category="invented_failure"), mirror_before(a))),
    m("rejected_valid_observation", "complete", {"L07"},
      lambda a, b: a.doc["healthChecks"][1].update(rejected={"returned": a.doc["healthChecks"][1]["serverState"]},
                                                   serverState=None)),
    m("server_before_differs", "complete", {"L07"}, lambda a, b: a.files["server-before.json"].update(serverTitle="changed")),
]

FILE_MUTATIONS = [
    ("seal_missing", "complete", {"S01"}, lambda live: (live / J.SEAL).unlink(), "DT13I-02 completion without a seal"),
    ("seal_missing_partial", "health_stop", {"S01"}, lambda live: (live / J.SEAL).unlink(), "V0.13l"),
    ("seal_truncated", "complete", {"L16"}, lambda live: (live / J.SEAL).write_text("{"), "V0.13l"),
    ("seal_is_null", "complete", {"S02", "L16"}, lambda live: (live / J.SEAL).write_text("null\n"), "V0.13l"),
    ("stray_seal_temporary_file", "complete", {"L16"}, lambda live: (live / ".seal-x").write_text("{}"), "V0.13l"),
    ("finalizing_marker_beside_seal", "complete", {"L16"}, lambda live: (live / "FINALIZING").write_text("{}"),
     "V0.13i marker"),
    ("stray_temporary_file", "complete", {"L16"}, lambda live: (live / ".tmp-x").write_text("{}"), "V0.13l"),
    ("missing_manifest", "complete", {"L16"}, lambda live: (live / "run.json").unlink(), "V0.13l"),
    ("missing_ledger", "complete", {"L16"}, lambda live: (live / "draws.json").unlink(), "V0.13l"),
    ("missing_call_journal", "complete", {"L16"}, lambda live: (live / J.CALLS).unlink(), "V0.13l"),
    ("missing_failure_event", "health_stop", {"L16", "S03"}, lambda live: (live / "failure-event.json").unlink(), "V0.13l"),
    ("missing_aborted_artifact", "aborted_workflow", {"L05", "S03", "L04"}, lambda live: (live / "aborted-draw.json").unlink(),
     "V0.13l"),
    ("missing_server_before", "complete", {"L07", "S03"}, lambda live: (live / "server-before.json").unlink(), "V0.13l"),
    ("subdirectory_in_archive", "complete", {"L16"}, lambda live: (live / "extra").mkdir(), "V0.13l"),
    ("call_journal_noncanonical", "complete", {"L04"},
     lambda live: (live / J.CALLS).write_bytes((live / J.CALLS).read_bytes().replace(b'{"draw"', b'{ "draw"', 1)), "V0.13l"),
]


def run(binary, scratch, analyze=True, names=None):
    bases = record_bases(binary, scratch)
    results = {"bases": {k: v[2] for k, v in bases.items()}, "mutants": []}
    for mu in MUTATIONS + [{"name": n, "base": b, "expect": e, "file": f, "source": s, "accept": False, "replay": False}
                           for n, b, e, f, s in FILE_MUTATIONS]:
        if names is not None and mu["name"] not in names:
            continue
        fixture, src, _ = bases[mu["base"]]
        out = os.path.join(scratch, "mutant-" + mu["name"])
        shutil.rmtree(out, ignore_errors=True)
        shutil.copytree(src, out)
        live = Path(out) / "live"
        if "file" in mu:
            mu["file"](live)
        else:
            a = Arc(out)
            mu["fn"](a, bases)
            w = dict(mu["write"])
            tear, patch = w.pop("tear", False), w.pop("seal_patch", None)
            a.write(**w)
            if patch:
                for k, v in patch.items():
                    if isinstance(v, dict):
                        a.seal[k].update(v)
                    else:
                        a.seal[k] = v
                (live / J.SEAL).write_bytes(J.encode_json(a.seal))
            if tear:
                with open(live / J.CALLS, "ab") as f:
                    f.write(b'{"draw":2,"k":3')
                a.reseal()
                (live / J.SEAL).write_bytes(J.encode_json(a.seal))
        v = fixture.classify(out)
        found = sorted({x["invariant"] for x in v["violations"]})
        refused = None
        if analyze:
            try:
                fixture.analyze(out)
                refused = False
            except D.ExperimentError:
                refused = True
        if mu["accept"]:
            ok = v["state"] != "INVALID" and refused is not True
        elif mu["replay"]:
            ok = refused is True and (v["state"] == "INVALID" or analyze)
        else:
            ok = v["state"] == "INVALID" and bool(set(mu["expect"]) & set(found)) and refused is not False
        results["mutants"].append({"name": mu["name"], "base": mu["base"], "source": mu["source"],
                                   "expected": "accepted" if mu["accept"] else "refused by replay" if mu["replay"]
                                   else sorted(mu["expect"]),
                                   "classification": v["state"], "violations": found, "analysisRefused": refused,
                                   "agree": ok})
        shutil.rmtree(out, ignore_errors=True)
    results["summary"] = {"mutants": len(results["mutants"]), "agree": sum(x["agree"] for x in results["mutants"]),
                          "positiveControls": sum(1 for x in results["mutants"] if x["expected"] == "accepted")}
    return results


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    os.makedirs(a.scratch, exist_ok=True)
    r = run(a.binary, a.scratch)
    D.write_json(a.out, dict({"schema": SCHEMA, "notice": "Scripted archives only; every dependent hash, count and "
                                                          "seal binding recomputed; no inference."}, **r))
    print(f"mutations: {r['summary']['agree']} of {r['summary']['mutants']} as expected "
          f"({r['summary']['positiveControls']} positive controls accepted)")
    for x in r["mutants"]:
        if not x["agree"]:
            print("DISAGREE", x["name"], x["classification"], x["violations"], x["analysisRefused"])
    if r["summary"]["agree"] != r["summary"]["mutants"]:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
