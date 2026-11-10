#!/usr/bin/env python3
"""Binding-driven full-analysis matrix.

Part 1 (inherited): each of the 35 field bindings has ten named variations.
Part 2 (V0.13l): each cross-field relationship of bindings_v2.json is run under
its seven planned kinds: valid agreement, missing evidence, contradiction,
duplicate evidence, incorrect identity, incorrect type and a real recorder
persistence interruption with preservation expectations. The plans are data
in bindings_v2.json; this module implements their named operations.
Paths name the exact field or associated ordering/count/hash constraint under test.
The archive writer recomputes all dependent commitments before full analysis.
"""
import argparse
import copy
import errno
import json
from pathlib import Path
import shutil
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import lifecycle_v4_spec as LS
import v13m as K
import v13m_mutations as M
import v13m_regressions as R
import v13m_scenarios as S
import v13m_witnesses as W

# A path starts in a persisted artifact: run, rows, calls, aborted, seal or before.
# The seven facets are required, contradiction, stage, identity, count, hash, order.
# Duplicate and null variations target the required facet. Order facets are lists.
# A semantic mutation of a seal is applied AFTER recomputing its dependencies.
PLANS = {
 'B-MANIFEST-001': ('complete', 'run.schema', 'run.driver', 'run.state', 'run.protocol', 'run.counters.modelCalls', 'run.protocolSha256', 'run.journal'),
 'B-MANIFEST-002': ('complete', 'run.runId', 'run.resume', 'run.state', 'run.runId', 'run.counters.modelCalls', 'run.scheduleSha256', 'run.journal'),
 'B-MANIFEST-003': ('complete', 'run.counters', 'run.counters.drawsCompleted', 'run.state', 'run.lifecycle.id', 'run.counters.drawsNotRun', 'run.lifecycle.sha256', 'run.journal'),
 'B-LEDGER-001': ('complete', 'rows.0.seq', 'rows.0.context', 'rows.0.status', 'rows.0.model', 'rows.0.draw', 'rows.0.submittedRepresentationSha256', 'rows'),
 'B-LEDGER-002': ('complete', 'rows.0.engineInitialization', 'rows.0.verificationStatus', 'rows.0.failureStage', 'rows.0.schema', 'rows.0.modelCallsCompleted', 'rows.0.submittedRepresentationSha256', 'rows.0.engine'),
 'B-JOURNAL-001': ('complete', 'run.journal.0.event', 'run.journal.2.to', 'run.journal.0.from', 'run.journal.0.event', 'run.journal.1.n', 'run.lifecycle.sha256', 'run.journal'),
 'B-JOURNAL-002': ('complete', 'commit.draw', 'commit.status', 'commit.reconciled', 'commit.draw', 'commit.modelCalls', 'commit.rowSha256', 'run.journal'),
 'B-CALL-001': ('complete', 'calls.0.record', 'calls.0.draw', 'calls.0.record', 'calls.0.schema', 'calls.0.k', 'calls.1.requestSha256', 'calls'),
 'B-CALL-002': ('complete', 'calls.1.requestSha256', 'calls.1.body', 'calls.1.record', 'calls.1.draw', 'calls.1.n', 'calls.1.responseSha256', 'calls'),
 'B-CALL-003': ('complete', 'calls.0.timestamp', 'rows.0.calls.0.sent', 'calls.0.timestamp', 'calls.0.draw', 'calls.0.n', 'calls.1.requestSha256', 'calls'),
 'B-CALL-004': ('unrecorded', 'calls.0.draw', 'run.counters.modelCalls', 'calls.0.record', 'calls.0.draw', 'calls.0.n', 'calls.1.requestSha256', 'calls'),
 'B-TRANSPORT-001': ('complete', 'rows.0.calls.0.request', 'rows.0.calls.0.request', 'rows.0.calls.0.sent', 'rows.0.calls.0.servedModel', 'rows.0.calls.0.n', 'rows.0.calls.0.requestSha256', 'rows.0.engine'),
 'B-DECISION-001': ('complete', 'rows.0.record.outcome', 'rows.0.record.result', 'rows.0.status', 'rows.0.record.workflowVersion', 'rows.0.record.retryRequests', 'rows.0.record.workflowSha256', 'rows.0.engine'),
 'B-DECISION-002': ('complete', 'rows.0.process.exposure', 'rows.0.process.processFailure', 'rows.0.status', 'rows.0.arm', 'rows.0.modelCallsCompleted', 'rows.0.submittedRepresentationSha256', 'rows.0.engine'),
 'B-BATCH-001': ('complete', 'rows.0.batches', 'rows.0.batches', 'rows.0.status', 'rows.0.arm', 'rows.0.modelCallsCompleted', 'rows.0.submittedRepresentationSha256', 'rows.0.engine'),
 'B-ENGINE-001': ('engine_close', 'aborted.engine.0.answer', 'aborted.engine.0.answer', 'aborted.failureStage', 'aborted.engine.0.op', 'aborted.modelCallsCompleted', 'aborted.initialDigests.state', 'aborted.engine'),
 'B-ENGINE-002': ('engine_close', 'aborted.digestsAfter', 'aborted.digestsAfter.state', 'aborted.failureStage', 'aborted.context', 'aborted.modelCallsCompleted', 'aborted.initialDigests.state', 'aborted.engine'),
 'B-HEALTH-001': ('complete', 'run.healthChecks.0.index', 'run.healthChecks.0.state', 'run.healthChecks.0.state', 'run.healthChecks.1.beforeDraw', 'run.healthChecks.1.index', 'run.healthChecks.0.serverState.health.body.sha256', 'run.healthChecks'),
 'B-HEALTH-002': ('complete', 'run.healthChecks.0.serverState.health.outcome', 'run.healthChecks.0.serverState.health.category', 'run.healthChecks.0.state', 'run.healthChecks.0.serverState.health.request.path', 'run.healthChecks.0.serverState.health.httpStatus', 'run.healthChecks.0.serverState.health.body.sha256', 'run.healthChecks'),
 'B-HEALTH-003': ('complete', 'run.healthChecks.0.serverState.health.body.base64', 'run.healthChecks.0.serverState.health.body.base64', 'run.healthChecks.0.serverState.health.outcome', 'run.healthChecks.0.serverState.health.schema', 'run.healthChecks.0.serverState.health.body.bytes', 'run.healthChecks.0.serverState.health.body.sha256', 'run.healthChecks'),
 'B-HEALTH-004': ('health_stop', 'run.healthChecks.0.serverState.health.exception.type', 'run.healthChecks.0.serverState.health.exception.reasonKind', 'run.healthChecks.0.serverState.health.outcome', 'run.healthChecks.0.serverState.health.exception.type', 'run.healthChecks.0.serverState.health.httpStatus', 'run.protocolSha256', 'run.journal'),
 'B-HEALTH-005': ('complete', 'before.health', 'before.health.category', 'run.healthChecks.0.state', 'before.health.schema', 'run.counters.healthChecks', 'before.health.body.sha256', 'run.healthChecks'),
 'B-STOP-001': ('health_stop', 'run.stop.code', 'run.stop.failure', 'run.stop.state', 'run.stop.code', 'run.stop.healthCheck', 'run.failureEvent.protocolSha256', 'run.journal'),
 'B-STOP-002': ('halt_row', 'rows.0.halt.code', 'rows.0.halt.evidence.stage', 'rows.0.halt.evidence.stage', 'rows.0.halt.code', 'rows.0.halt.evidence.runCalls', 'run.protocolSha256', 'run.journal'),
 'B-INTERRUPT-001': ('after_schedule', 'run.interrupts.events.0.safePoint', 'run.interrupts.events.0.disposition', 'run.interrupts.events.0.state', 'run.interrupts.signal', 'run.interrupts.events.0.n', 'run.scheduleSha256', 'run.cleanup.events'),
 'B-CLEANUP-001': ('complete', 'run.cleanup.started', 'run.cleanup.status', 'run.cleanup.events.0.state', 'run.cleanup.events.0.safePoint', 'run.cleanup.events.1.n', 'run.lifecycle.sha256', 'run.cleanup.events'),
 'B-ABORT-001': ('engine_close', 'aborted.failureStage', 'aborted.engineInitialization', 'aborted.failureStage', 'aborted.context', 'aborted.seq', 'aborted.submittedRepresentationSha256', 'aborted.engine'),
 'B-ABORT-002': ('engine_close', 'aborted.record', 'aborted.process.exposure', 'aborted.failureStage', 'aborted.arm', 'aborted.modelCallsCompleted', 'aborted.digestsAfter.state', 'aborted.engine'),
 'B-FAILURE-001': ('health_stop', 'run.failureEvent.nextDraw', 'run.failureEvent.state', 'run.failureEvent.state', 'run.failureEvent.runId', 'run.failureEvent.modelCalls', 'run.failureEvent.scheduleSha256', 'run.journal'),
 'B-SEAL-001': ('complete', 'seal.schema', 'seal.driver', 'seal.runSchema', 'seal.runId', 'seal.transition.n', 'seal.lifecycle.sha256', 'seal.ledger.rowSha256'),
 'B-SEAL-002': ('complete', 'seal.transition.event', 'seal.terminalState', 'seal.transition.from', 'seal.transition.event', 'seal.transition.n', 'seal.lifecycle.sha256', 'run.journal'),
 'B-SEAL-003': ('complete', 'seal.ledger.rows', 'seal.ledger.completed', 'run.state', 'seal.ledger.file', 'seal.calls.dispatched', 'seal.calls.sha256', 'seal.ledger.rowSha256'),
 'B-SEAL-004': ('complete', 'seal.health.checks', 'seal.counters.drawsCompleted', 'run.state', 'seal.lifecycle.id', 'seal.health.checks', 'seal.health.serverBefore', 'seal.health.checkSha256'),
}


def roots(a):
    return dict(run=a.doc, rows=a.rows, calls=a.calls, seal=a.seal,
                aborted=a.files.get('aborted-draw.json'), before=a.files.get('server-before.json'),
                commit=M.first(a, 'commit_draw') if any(e['event']=='commit_draw' for e in a.doc['journal']) else None)


def locate(a, path):
    parts = path.split('.')
    p = roots(a)
    for k in parts[:-1]:
        p = p[int(k)] if isinstance(p, list) else p[k]
    k = int(parts[-1]) if isinstance(p, list) else parts[-1]
    return p, k


def changed(value):
    if isinstance(value, bool): return not value
    if isinstance(value, (int, float)): return value + 17
    if isinstance(value, str): return value + '-contradiction'
    if isinstance(value, list): return value + [{'contradiction': True}]
    if isinstance(value, dict): return dict(value, contradiction=True)
    return 'contradiction'


def mutate(a, path, kind):
    p, k = locate(a, path)
    old = p[k]
    if kind == 'missing_required_evidence': del p[k]
    elif kind == 'invalid_null': p[k] = None
    elif kind == 'wrong_hash': p[k] = 'z' * 64
    elif kind == 'wrong_ordering':
        # Reverse the bound sequence; retain original ordinals and content.
        value = list(reversed(old))
        if path == 'rows': a.files['draws.json'] = value
        elif path == 'calls': a.calls = value
        else: p[k] = value
    elif kind == 'wrong_count': p[k] = 999
    elif kind == 'wrong_stage':
        leaf = path.rsplit('.', 1)[-1]
        if isinstance(old, bool): p[k] = not old
        elif leaf in ('timestamp', 'sent'): p[k] = '2030-01-01T00:00:00Z'
        elif leaf == 'status': p[k] = 'interrupted' if old == 'completed' else 'completed'
        elif leaf == 'record': p[k] = 'returned' if old == 'dispatch' else 'dispatch'
        elif leaf == 'stage': p[k] = 'draw_start' if old == 'request_start' else 'request_start'
        elif leaf == 'failureStage': p[k] = 'engine_open' if old != 'engine_open' else 'engine_close'
        elif leaf == 'outcome': p[k] = 'READ_FAILED' if old == 'HEALTHY' else 'HEALTHY'
        elif leaf == 'runSchema': p[k] = 'dreamteam.alternatives-run/v6'
        else: p[k] = 'READY' if old != 'READY' else 'CLEANUP' 
    else: p[k] = changed(old)


def duplicate_key(a, path):
    parent, key = locate(a, path)
    prefix = path.split('.')[0]
    artifact = {'run':'run.json','rows':'draws.json','calls':K.CALLS,'aborted':'aborted-draw.json',
                'before':'server-before.json','seal':K.SEAL,'commit':'run.json'}[prefix]
    def encode(value):
        if isinstance(value, dict):
            entries = [json.dumps(k)+':'+encode(v) for k,v in sorted(value.items())]
            if value is parent:
                entries.append(json.dumps(key)+':'+encode(parent[key]))
            return '{'+','.join(entries)+'}'
        if isinstance(value, list): return '['+','.join(encode(v) for v in value)+']'
        return json.dumps(value, ensure_ascii=True, allow_nan=False)
    if prefix == 'calls':
        data = ''.join(encode(v)+'\n' for v in a.calls).encode()
    else:
        value = a.seal if prefix == 'seal' else a.files[artifact]
        data = (encode(value)+'\n').encode()
    (a.live / artifact).write_bytes(data)


# --- Part 2: the relationship matrix ------------------------------------------------------

class _FinalSession:
    """Fail the draw's actual final-digests request."""

    def __init__(self, s):
        self.s = s

    def request(self, rid, op, args=None):
        frame = sys._getframe(1)
        while frame:
            if frame.f_code is K.H.run_draw.__code__ and frame.f_locals.get("stage") == "final_digests":
                raise BrokenPipeError("actual final-digests failure")
            frame = frame.f_back
        return self.s.request(rid, op, args)

    def close(self):
        self.s.close()


class _FinalPool:
    def __init__(self, p):
        self.p = p

    def __call__(self, *a):
        return _FinalSession(self.p(*a))

    def close(self):
        self.p.close()


def relationship_bases(binary):
    """name -> (fixture, health modes, recorder keyword factory)."""
    both = S.Fixture(binary, arms=("control", "treatment"))
    one = S.Fixture(binary)
    halt = both.with_limits(maxModelCallsPerDraw=0)

    def second_call_fails():
        calls = [0]

        def model(payload):
            calls[0] += 1
            if calls[0] == 2:
                return None, "", "timeout: scripted"
            if calls[0] == 3:
                raise RuntimeError("scripted transport failure after a failed exchange")
            return S.D.scripted([S.F.ABANDON] * 4)(payload)
        return {"model": model}
    return {
        "complete": (both, None, lambda f: {}),
        "health_stop": (one, {1: "refused"}, lambda f: {}),
        "aborted_workflow": (one, None, lambda f: second_call_fails()),
        "halt_row": (one.with_limits(maxModelCallsPerDraw=0), None, lambda f: {}),
        "engine_close": (both, None, lambda f: {"opener": W.FailPool(f.ctx.opener(binary), "engine_close")}),
        "halt_close": (halt, None, lambda f: {"opener": W.FailPool(f.ctx.opener(binary), "engine_close")}),
        "initial_digests": (both, None, lambda f: {"opener": W.FailPool(f.ctx.opener(binary), "initial_digests")}),
        "final_digests": (both, None, lambda f: {"opener": _FinalPool(f.ctx.opener(binary))}),
    }


def _record_base(registry, name, out, hook=None):
    fixture, modes, kw = registry[name]
    clock = S.Clock()
    extra = kw(fixture)
    if hook is not None:
        extra["hook"] = hook
    result = S.attempt(lambda: fixture.record(out, clock=clock, health=S.Health(clock, modes=modes), **extra))
    return fixture, result


def _rehash_request(a, row, n, text):
    call = row["calls"][n - 1]
    call["requestBody"] = text
    call["requestSha256"] = K.sha256(text.encode("utf-8"))
    for rec in a.calls:
        if rec.get("draw") == row["seq"] and rec.get("n") == n and rec.get("record") != "dispatch":
            rec["requestSha256"] = call["requestSha256"]


def _raw_file(a, name, data, out):
    """Replace one artifact's bytes and recompute the seal over the bytes on disk."""
    a.write()
    (Path(out) / "live" / name).write_bytes(data)
    if a.seal is not None:
        a.reseal()
        (Path(out) / "live" / K.SEAL).write_bytes(K.encode_json(a.seal))


def _set_stop_failure(a, failure):
    a.doc["stop"]["failure"] = failure
    a.doc["stop"]["message"] = f"{failure['type']}: {failure['message']}"
    a.doc["failureEvent"]["stop"] = copy.deepcopy(a.doc["stop"])


def _pending(a, value):
    a.files["pending-draw.json"] = value


def _commit(a, k=0):
    return [e for e in a.doc["journal"] if e["event"] == "commit_draw"][k]


CUSTOM = {}


def custom(fn):
    CUSTOM[fn.__name__] = fn
    return fn


@custom
def request_temperature_changed(a, out):
    a.rows[0]["calls"][0]["request"]["temperature"] = 1
    a.write()


@custom
def request_zero_versus_false(a, out):
    a.rows[0]["calls"][0]["request"]["temperature"] = False
    a.write()


@custom
def request_body_duplicate_key(a, out):
    row = a.rows[0]
    _rehash_request(a, row, 1, row["calls"][0]["requestBody"][:-1] + ', "temperature": 0}')
    a.write()


@custom
def request_hash_of_other_body(a, out):
    row, other = a.rows[0], a.rows[1]
    want = other["calls"][0]["requestSha256"]
    row["calls"][0]["requestSha256"] = want
    for rec in a.calls:
        if rec.get("draw") == row["seq"] and rec.get("n") == 1 and rec.get("record") != "dispatch":
            rec["requestSha256"] = want
    a.write()


@custom
def truncate_ledger(a, out):
    _raw_file(a, "draws.json", K.encode_json(a.rows)[:-40], out)


@custom
def journal_noncanonical_line(a, out):
    lines = [K.encode_line(r) for r in a.calls]
    lines[0] = (json.dumps(a.calls[0], sort_keys=True) + "\n").encode("ascii")
    _raw_file(a, K.CALLS, b"".join(lines), out)


@custom
def journal_line_wrong_schema(a, out):
    a.calls[0]["schema"] = "dreamteam.call-journal/v0"
    a.write()


@custom
def nonfinite_latency(a, out):
    a.rows[0]["calls"][0]["latencyMs"] = float("nan")
    data = (json.dumps(a.rows, sort_keys=True, ensure_ascii=False, indent=1) + "\n").encode()
    a.write()
    (Path(out) / "live" / "draws.json").write_bytes(data)
    _commit(a)["rowSha256"] = K.sha256(K.canon(a.rows[0]))
    (Path(out) / "live" / "run.json").write_bytes(K.encode_json(a.doc))
    a.reseal()
    (Path(out) / "live" / K.SEAL).write_bytes(K.encode_json(a.seal))


@custom
def response_hash_of_other_body(a, out):
    want = K.sha256("another response body")
    a.rows[0]["calls"][0]["responseSha256"] = want
    a.calls[1]["responseSha256"] = want
    a.write()


@custom
def served_model_as_list(a, out):
    call = a.rows[0]["calls"][0]
    call["servedModel"] = [call["servedModel"]]
    a.write()


@custom
def drop_outcome_record(a, out):
    del a.calls[1]
    a.write()


@custom
def duplicate_dispatch_pair(a, out):
    a.calls = a.calls[:2] + copy.deepcopy(a.calls[:2]) + a.calls[2:]
    a.write()


@custom
def duplicate_outcome_record(a, out):
    a.calls.insert(2, copy.deepcopy(a.calls[1]))
    a.write()


@custom
def journal_status_differs(a, out):
    a.calls[1]["status"] = 201
    a.write()


@custom
def status_float_in_journal(a, out):
    a.calls[1]["status"] = float(a.calls[1]["status"])
    a.write()


@custom
def journal_body_differs_from_call(a, out):
    a.calls[1]["body"] = a.calls[1]["body"] + " "
    a.calls[1]["responseSha256"] = K.sha256(a.calls[1]["body"])
    a.write()


@custom
def journal_body_as_object(a, out):
    a.calls[1]["body"] = json.loads(a.calls[1]["body"])
    a.write()


def _raised(a):
    return next(r for r in a.calls if r.get("record") == "raised")


@custom
def raised_exception_removed(a, out):
    del _raised(a)["exception"]
    a.write()


@custom
def raised_disagrees_with_primary(a, out):
    _raised(a)["exception"] = {"type": "ValueError", "message": "different transport failure"}
    a.files["aborted-draw.json"]["calls"][-1]["error"] = "client: ValueError: different transport failure"
    a.write()


@custom
def raised_type_renamed_everywhere_but_primary(a, out):
    x = _raised(a)["exception"]
    x["type"] = "ConnectionError"
    a.files["aborted-draw.json"]["calls"][-1]["error"] = f"client: ConnectionError: {x['message']}"
    a.write()


@custom
def raised_message_as_number(a, out):
    _raised(a)["exception"]["message"] = 7
    a.write()


def _halt(a):
    return a.files["aborted-draw.json"]["halt"]


@custom
def halt_five_failures_zero_calls(a, out):
    h = _halt(a)
    h["code"], h["message"] = "transport", "5 consecutive transport failures"
    h["evidence"]["consecutiveTransportFailures"] = 5
    a.write()


@custom
def halt_stage_draw_start(a, out):
    h = _halt(a)
    h["code"], h["message"] = "wall_clock", "20000 s elapsed; the next draw could pass the 14400 s stop"
    h["evidence"].update(stage="draw_start", elapsedSeconds=20000)
    a.write()


def _row_halt(a, fn):
    fn(a.rows[0]["halt"])
    stop = a.doc["stop"]
    stop.update(code=a.rows[0]["halt"]["code"], message=a.rows[0]["halt"]["message"],
                evidence=copy.deepcopy(a.rows[0]["halt"]["evidence"]))
    a.doc["failureEvent"]["stop"] = copy.deepcopy(stop)
    for e in a.doc["journal"]:
        if e["event"] == "stop":
            e["code"] = stop["code"]
    a.write()


@custom
def row_halt_streak(a, out):
    def fn(h):
        h["code"], h["message"] = "transport", "3 consecutive transport failures"
        h["evidence"]["consecutiveTransportFailures"] = 3
    _row_halt(a, fn)


@custom
def row_halt_message_other_limit(a, out):
    def fn(h):
        h["code"], h["message"] = "call_budget", f"{h['evidence']['runCalls']} model calls made; the limit is 802"
    _row_halt(a, fn)


@custom
def row_halt_draw_calls_float(a, out):
    _row_halt(a, lambda h: h["evidence"].update(drawCalls=float(h["evidence"]["drawCalls"])))


@custom
def stop_message_disagrees(a, out):
    a.doc["stop"]["message"] = "ValueError: a different failure than stop.failure"
    a.doc["failureEvent"]["stop"] = copy.deepcopy(a.doc["stop"])
    a.write()


@custom
def stop_failure_type_changed(a, out):
    f = dict(a.doc["stop"]["failure"], type="OSError")
    _set_stop_failure(a, f)
    a.write()


@custom
def stop_failure_message_bool(a, out):
    a.doc["stop"]["failure"]["message"] = False
    a.doc["failureEvent"]["stop"] = copy.deepcopy(a.doc["stop"])
    a.write()


@custom
def engine_failed_entry_twice(a, out):
    ab = a.files["aborted-draw.json"]
    ab["engine"].append(copy.deepcopy(ab["engine"][-1]))
    a.write()


@custom
def final_engine_error_disagrees(a, out):
    f = {"type": "ValueError", "message": "a different final failure"}
    a.files["aborted-draw.json"]["failure"] = f
    _set_stop_failure(a, f)
    a.write()


@custom
def pending_without_draw(a, out):
    _pending(a, {"schema": "dreamteam.completed-draw-handoff/v1", "rowSha256": K.sha256(K.canon(a.rows[-1]))})
    a.write()


@custom
def pending_draw_differs(a, out):
    row = copy.deepcopy(a.rows[-1])
    row["draw"] = 99
    _pending(a, {"schema": "dreamteam.completed-draw-handoff/v1", "draw": row, "rowSha256": K.sha256(K.canon(row))})
    a.write()


@custom
def pending_duplicates_ledger(a, out):
    row = copy.deepcopy(a.rows[-1])
    _pending(a, {"schema": "dreamteam.completed-draw-handoff/v1", "draw": row, "rowSha256": K.sha256(K.canon(row))})
    a.write()


@custom
def pending_wrong_row_hash(a, out):
    row = copy.deepcopy(a.rows[-1])
    _pending(a, {"schema": "dreamteam.completed-draw-handoff/v1", "draw": row, "rowSha256": K.sha256(K.canon(a.rows[0]))})
    a.write()


@custom
def pending_draw_as_list(a, out):
    _pending(a, {"schema": "dreamteam.completed-draw-handoff/v1", "draw": [copy.deepcopy(a.rows[-1])],
                 "rowSha256": K.sha256(K.canon(a.rows[-1]))})
    a.write()


@custom
def commit_without_row(a, out):
    a.files["draws.json"] = a.rows[:-1]
    a.write(commits=False)


@custom
def commit_row_hash_other(a, out):
    _commit(a, 0)["rowSha256"] = K.sha256(K.canon(a.rows[1]))
    a.write(commits=False)


@custom
def ledger_row_twice(a, out):
    a.files["draws.json"] = a.rows + [copy.deepcopy(a.rows[-1])]
    a.write()


@custom
def commit_names_other_draw(a, out):
    _commit(a, 0)["draw"] = a.rows[1]["seq"]
    a.write(commits=False)


@custom
def commit_reconciled_as_int(a, out):
    _commit(a, 0)["reconciled"] = 0
    a.write(commits=False)


@custom
def seal_removed(a, out):
    a.write()
    (Path(out) / "live" / K.SEAL).unlink()


@custom
def seal_ledger_rows_count(a, out):
    a.write()
    a.seal["ledger"]["rows"] += 1
    (Path(out) / "live" / K.SEAL).write_bytes(K.encode_json(a.seal))


@custom
def seal_duplicate_key(a, out):
    a.write()
    text = K.encode_json(a.seal).decode()
    text = text.replace('"terminalState": "COMPLETE"', '"terminalState": "COMPLETE",\n "terminalState": "PARTIAL"', 1)
    (Path(out) / "live" / K.SEAL).write_bytes(text.encode())


@custom
def seal_run_id_other(a, out):
    a.write()
    a.seal["runId"] = "00000000-0000-4000-8000-000000000001"
    (Path(out) / "live" / K.SEAL).write_bytes(K.encode_json(a.seal))


@custom
def seal_rows_as_bool(a, out):
    a.write()
    a.seal["ledger"]["rows"] = float(a.seal["ledger"]["rows"])
    (Path(out) / "live" / K.SEAL).write_bytes(K.encode_json(a.seal))


@custom
def calls_beyond_journal(a, out):
    row = a.rows[0]
    extra = copy.deepcopy(row["calls"][0])
    extra["n"] = len(row["calls"]) + 1
    row["calls"].append(extra)
    row["modelCallsCompleted"] = len(row["calls"])
    a.write()


@custom
def run_budget_one(a, out, fixture=None):
    a.write()


@custom
def calls_repeated_in_row(a, out):
    row = a.rows[0]
    dup = copy.deepcopy(row["calls"][0])
    dup["n"] = 2
    row["calls"].append(dup)
    row["modelCallsCompleted"] = 2
    d, o = copy.deepcopy(a.calls[0]), copy.deepcopy(a.calls[1])
    for rec in (d, o):
        rec.update(n=2)
    a.calls = a.calls[:2] + [d, o] + a.calls[2:]
    for k, rec in enumerate([r for r in a.calls if r["record"] == "dispatch"], 1):
        rec["k"] = k
    k = 0
    for rec in a.calls:
        if rec["record"] == "dispatch":
            k = rec["k"]
        else:
            rec["k"] = k
    a.doc["counters"]["modelCalls"] = k
    a.write()


@custom
def dispatch_for_unscheduled_draw(a, out):
    for rec in a.calls[:2]:
        rec["draw"] = 99
    a.write()


def _fault_hook(plan, seen):
    target = (plan["boundary"], plan["phase"])

    def hook(boundary, phase, ctx):
        if (boundary, phase) != target or seen.get("fired"):
            return
        seen["count"] = seen.get("count", 0) + 1
        if seen["count"] == plan["occurrence"]:
            seen["fired"], seen["context"] = True, dict(ctx)
            raise OSError(errno.EIO, f"relationship matrix persistence fault at {boundary}.{phase}")
    return hook


def _preserved(out, result, ctx, keys):
    live = Path(out) / "live"
    completed, received = set(), set()
    for path in live.iterdir() if live.exists() else []:
        try:
            value = json.loads(path.read_bytes())
        except (ValueError, OSError, UnicodeDecodeError):
            continue
        rows = value if isinstance(value, list) else [value.get("draw"), value.get("pending")] if isinstance(value, dict) else []
        completed.update(K.sha256(K.canon(r)) for r in rows if isinstance(r, dict) and r.get("status") == "completed")
        if path.name == "persistence-failure.json" and isinstance(value, dict):
            received.update(K.canon(r) for r in value.get("receivedResponses") or [])
    retained = result.get("retainedEvidence") if isinstance(result, dict) else None
    if isinstance(retained, dict):
        if isinstance(retained.get("pending"), dict) and retained["pending"].get("status") == "completed":
            completed.add(K.sha256(K.canon(retained["pending"])))
        received.update(K.canon(r) for r in retained.get("receivedResponses") or [])
    journal = S.journal(out) or []
    have = {"dispatches": sum(r.get("record") == "dispatch" for r in journal), "completedRows": len(completed),
            "receivedResponses": len(received)}
    known = {"dispatches": ctx.get("dispatched", 0),
             "completedRows": ctx.get("completedRows", 0) + int(ctx.get("pendingStatus") == "completed"
                                                               and not ctx.get("pendingDurable")),
             "receivedResponses": ctx.get("receivedUnrecorded", 0)}
    lost = [k for k in keys if have[k] < known[k]]
    if ctx.get("receivedUnrecorded") and have["receivedResponses"] < known["receivedResponses"]:
        lost.append("receivedResponses")
    return have, known, sorted(set(lost))


def run_relationships(binary, scratch):
    rels = LS.relationships()
    registry = relationship_bases(binary)
    root = Path(scratch) / "relationships"
    root.mkdir(parents=True, exist_ok=True)
    bases = {}
    for name in sorted({p["base"] for r in rels for p in r["matrix"].values() if "base" in p}):
        out = str(root / ("base-" + name))
        fixture, result = _record_base(registry, name, out)
        bases[name] = (fixture, out)
    results = []
    for r in rels:
        valid_base = r["matrix"]["valid_agreement"]["base"]
        for kind in LS.RELATIONSHIP_KINDS:
            plan = r["matrix"][kind]
            name = (r["id"] + "-" + kind)
            out = str(root / name)
            row = dict(relationship=r["id"], guarantee=r["guarantee"], kind=kind, expectedAccepted=kind == "valid_agreement")
            if plan.get("op") == "fault":
                seen = {}
                fixture, result = _record_base(registry, valid_base, out, hook=_fault_hook(plan, seen))
                cls = fixture.classify(out)
                try:
                    fixture.analyze(out)
                    accepted = True
                except Exception:  # noqa: BLE001
                    accepted = False
                have, known, lost = _preserved(out, result, seen.get("context") or {}, plan["preserve"])
                ok = bool(seen.get("fired")) and cls["state"] == plan["expect"] and not accepted and not lost \
                    and S.recorder_label(result) == "RecordingError"
                row.update(target=f"{plan['boundary']}.{plan['phase']}#{plan['occurrence']}", base=valid_base,
                           fired=bool(seen.get("fired")), classification=cls["state"], fullAnalysisAccepted=accepted,
                           preserved=have, knownAtFault=known, lost=lost, agree=ok)
                results.append(row)
                shutil.rmtree(out)
                continue
            fixture, source = bases[plan["base"]]
            shutil.copytree(source, out)
            a = M.Arc(out)
            if kind == "valid_agreement":
                target = "unmodified"
            elif plan["op"] == "custom":
                target = plan["name"]
                if plan["name"] == "run_budget_one":
                    fixture = fixture.with_limits(maxModelCalls=1)
                    a.doc["protocolSha256"] = fixture.proto_sha
                    a.seal["protocol"]["sha256"] = fixture.proto_sha
                    if a.doc.get("failureEvent"):
                        a.doc["failureEvent"]["protocolSha256"] = fixture.proto_sha
                CUSTOM[plan["name"]](a, out)
            elif plan["op"] == "duplicate_key":
                target = plan["path"]
                a.write()
                duplicate_key(a, target)
                if not target.startswith("seal."):
                    a.reseal()
                    (Path(out) / "live" / K.SEAL).write_bytes(K.encode_json(a.seal))
            else:
                target = plan["path"]
                parent, key = locate(a, target)
                if plan["op"] == "delete":
                    del parent[key]
                else:
                    parent[key] = copy.deepcopy(plan["value"])
                a.write(commits=not target.startswith("commit."))
            try:
                fixture.analyze(out)
                accepted, error = True, None
            except K.ExperimentError as x:
                accepted, error = False, str(x)[:300]
            except (KeyError, TypeError, ValueError, AttributeError, IndexError) as x:
                accepted, error = True, "uncaught " + repr(x)[:300]
            row.update(target=target, base=plan["base"], fullAnalysisAccepted=accepted, refusal=error,
                       agree=accepted == row["expectedAccepted"])
            results.append(row)
            shutil.rmtree(out)
    return results


def run(binary, scratch):
    Path(scratch).mkdir(parents=True, exist_ok=True)
    base = M.record_bases(binary, scratch)
    (Path(scratch) / 'originals').mkdir()
    f, extra = R.bases(binary, str(Path(scratch) / 'originals'))
    base['engine_close'] = (f, extra['engine_close'], 'PARTIAL')
    spec = json.loads(LS.spec_bytes())
    ids = [b['id'] for b in spec['bindings']]
    assert set(ids) == set(PLANS) | {'B-HANDOFF-001','B-EXECUTION-001'}
    results = []
    facets = {'missing_required_evidence':1, 'duplicated_evidence':1, 'invalid_null':1,
              'contradictory_evidence':2, 'wrong_stage':3, 'wrong_identity':4, 'wrong_count':5,
              'wrong_hash':6, 'wrong_ordering':7}
    for binding in ids:
        plan = PLANS.get(binding, PLANS['B-LEDGER-001'])
        fixture, source, state = base[plan[0]]
        for kind in spec['mutationKinds']:
            name = binding + '/' + kind
            out = str(Path(scratch) / name.replace('/', '-'))
            shutil.copytree(source, out)
            a = M.Arc(out)
            target = plan[facets[kind]] if kind != 'valid_matching_evidence' else 'all bound evidence'
            if kind != 'valid_matching_evidence':
                if binding in ('B-HANDOFF-001', 'B-EXECUTION-001'):
                    # Neither uncommitted handoffs nor claimed analysis may be
                    # imported as authoritative evidence into a sealed archive.
                    namefile = 'pending-draw.json' if binding == 'B-HANDOFF-001' else 'analysis.json'
                    value = {'schema': 'dreamteam.completed-draw-handoff/v1', 'draw': copy.deepcopy(a.rows[0]),
                             'rowSha256': K.sha256(K.canon(a.rows[0]))} if binding == 'B-HANDOFF-001' else fixture.analyze(out)
                    # The retained artifact itself is the stage contradiction.
                    # Each variation also changes the specified facet of it.
                    if kind == 'missing_required_evidence': value.pop('schema', None)
                    elif kind == 'invalid_null': value = None
                    elif kind == 'duplicated_evidence': value = [value, copy.deepcopy(value)]
                    else: value[kind] = 'contradiction'
                    a.files[namefile] = value
                    target = namefile + ' (forbidden in sealed archive)'
                    a.write()
                elif kind == 'duplicated_evidence':
                    p, k = locate(a, target)
                    value = copy.deepcopy(p[k])
                    a.write()
                    duplicate_key(a, target)
                    # Reseal duplicate bytes, without reparsing ambiguous JSON.
                    if not target.startswith('seal.'):
                        a.reseal()
                        (Path(out)/'live'/K.SEAL).write_bytes(K.encode_json(a.seal))
                elif target.startswith('seal.'):
                    a.write()
                    mutate(a, target, kind)
                    (Path(out)/'live'/K.SEAL).write_bytes(K.encode_json(a.seal))
                else:
                    mutate(a, target, kind)
                    # Targeted commit mutations must not be silently repaired.
                    a.write(commits=not target.startswith('commit.'))
            try:
                fixture.analyze(out)
                accepted, error = True, None
            except K.ExperimentError as x:
                accepted, error = False, str(x)
            except (KeyError, TypeError, ValueError) as x:
                # Parser exceptions are not an acceptable verifier disposition.
                accepted, error = True, 'uncaught ' + repr(x)
            expected = kind == 'valid_matching_evidence'
            results.append(dict(binding=binding, mutation=kind, target=target, base=plan[0],
                                expectedAccepted=expected, fullAnalysisAccepted=accepted,
                                refusal=error, agree=accepted == expected))
            shutil.rmtree(out)
    rel = run_relationships(binary, scratch)
    return dict(schema='dreamteam.binding-matrix/v2', bindingsSha256=LS.spec_sha256(),
                relationships=len(ids), mutationKinds=spec['mutationKinds'], cases=results,
                crossFieldRelationships=dict(
                    kinds=LS.RELATIONSHIP_KINDS, cases=rel,
                    summary=dict(relationships=len({r['relationship'] for r in rel}), cases=len(rel),
                                 agree=sum(r['agree'] for r in rel),
                                 validAccepted=sum(r['expectedAccepted'] and r['fullAnalysisAccepted'] for r in rel),
                                 contradictionsRejected=sum(not r['expectedAccepted'] and not r['fullAnalysisAccepted'] for r in rel),
                                 persistenceInterruptionsPreserved=sum(r['kind'] == 'persistence_interruption' and r['agree'] for r in rel),
                                 survivingContradictions=[r['relationship'] + '/' + r['kind'] for r in rel
                                                          if not r['expectedAccepted'] and r['fullAnalysisAccepted']],
                                 disagreements=[r['relationship'] + '/' + r['kind'] for r in rel if not r['agree']])),
                summary=dict(cases=len(results), agree=sum(r['agree'] for r in results),
                             validAccepted=sum(r['expectedAccepted'] and r['fullAnalysisAccepted'] for r in results),
                             contradictionsRejected=sum(not r['expectedAccepted'] and not r['fullAnalysisAccepted'] for r in results),
                             survivingContradictions=[r['binding']+'/'+r['mutation'] for r in results if not r['expectedAccepted'] and r['fullAnalysisAccepted']]))


def main():
    ap=argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--binary', required=True); ap.add_argument('--scratch', required=True); ap.add_argument('--out',required=True)
    a=ap.parse_args()
    doc=run(str(Path(a.binary).resolve()),str(Path(a.scratch).resolve()))
    K.write_json(a.out,doc)
    print(json.dumps(doc['summary']))
    print(json.dumps(doc['crossFieldRelationships']['summary']))
    rs = doc['crossFieldRelationships']['summary']
    if doc['summary']['agree'] != doc['summary']['cases'] or rs['agree'] != rs['cases']: raise SystemExit(1)

if __name__ == '__main__': main()
