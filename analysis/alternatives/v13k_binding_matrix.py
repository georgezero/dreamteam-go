#!/usr/bin/env python3
"""Binding-driven full-analysis matrix. Each relationship has ten named variations.
Paths name the exact field or associated ordering/count/hash constraint under test.
The archive writer recomputes all dependent commitments before full analysis.
"""
import argparse
import copy
import json
from pathlib import Path
import shutil
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import lifecycle_v3_spec as LS
import v13k as K
import v13k_mutations as M
import v13k_regressions as R
import v13k_scenarios as S

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
    return dict(schema='dreamteam.binding-matrix/v1', bindingsSha256=LS.spec_sha256(),
                relationships=len(ids), mutationKinds=spec['mutationKinds'], cases=results,
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
    if doc['summary']['agree'] != doc['summary']['cases']: raise SystemExit(1)

if __name__ == '__main__': main()
