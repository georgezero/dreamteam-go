#!/usr/bin/env python3
"""Pin successor identities and independently verify protected baseline bytes."""
import argparse
import hashlib
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0,str(ROOT/'analysis/alternatives'))
import v13l_preserved as P


def artifacts():
    rec='analysis/alternatives/v13l.py';ver='analysis/alternatives/v13l_verify.py'
    spec='analysis/alternatives/bindings_v2.json'
    doc='docs/v13l/';bench='docs/benchmark/v13l-dev/'
    paths={
      'dreamteam.evidence-bindings/2':[spec,doc+'EVIDENCE_BINDINGS.md'],
      'dreamteam.health-evidence-classification/v3':[spec,'analysis/alternatives/lifecycle_v4_spec.py',rec,ver,doc+'RECORDING.md'],
      'dreamteam.run-lifecycle/v4':[spec,'analysis/alternatives/lifecycle_v4_spec.py',doc+'LIFECYCLE_CONTRACT.md'],
      'dreamteam.strict-json/v1':[spec,ver,doc+'EVIDENCE_BINDINGS.md'],
      'dreamteam.response-handoff/v1':[rec,spec,doc+'RECORDING.md'],
      'dreamteam.completed-draw-handoff/v1':[rec,spec,doc+'RECORDING.md'],
      'dreamteam.failed-persistence-retention/v2':[rec,ver,spec,doc+'RECORDING.md'],
      'dreamteam-v13l-driver/8':[rec,doc+'DRIVER.md'],
      'dreamteam.alternatives-run/v8':[rec,doc+'RECORDING.md'],
      'dreamteam.alternatives-partial-run/v8':[ver,doc+'PARTIAL_RUN_PROCEDURE.md'],
      'dreamteam.alternatives-analysis/v9':[ver],
      'dreamteam.alternatives-experiment/v8':[bench+'protocol-draft.json'],
      'dreamteam.alternatives-schedule/v4#v13l-alt-1':[bench+'schedule-draft.json'],
      'dreamteam.binding-matrix/v2':['analysis/alternatives/v13l_binding_matrix.py',doc+'binding-matrix/results.json',doc+'BINDING_MATRIX_RESULTS.md'],
      'dreamteam.v13l-fault-matrix/v1':['analysis/alternatives/v13l_faults.py','analysis/alternatives/v13l_scenarios.py',doc+'fault-matrix/matrix.json'],
      'dreamteam.v13l-signal-tests/v1':['analysis/alternatives/v13l_signals.py',doc+'signals/results.json'],
      'dreamteam.v13l-loopback/v1':['analysis/alternatives/v13l_loopback.py',doc+'loopback/results.json'],
      'dreamteam.v13l-witnesses/v1':['analysis/alternatives/v13l_witnesses.py',doc+'witnesses/results.json'],
      'dreamteam.v13l-mutations/v1':['analysis/alternatives/v13l_mutations.py',doc+'mutations/results.json'],
      'dreamteam.v13l-regressions/v1':['analysis/alternatives/v13l_regressions.py',doc+'regressions/results.json'],
      'dreamteam.v13l-completed-preservation/v2':['analysis/alternatives/v13l_preservation.py',doc+'preservation/results.json'],
      'dreamteam.v13l-dt13k-witnesses/v1':['analysis/alternatives/v13l_dt13k.py',doc+'dt13k/before-v13k.json',doc+'dt13k/before-prefix-successor.json',doc+'dt13k/after-v13l.json'],
      'dreamteam.v13l-preservation/v1':['analysis/alternatives/v13l_preserved.py',doc+'BASELINE.json',doc+'PRESERVATION.json'],
      'dreamteam.v13l-fidelity/v1':['analysis/alternatives/v13l_fidelity.py',bench+'fidelity/results.json',doc+'FIDELITY.md'],
      'dreamteam.v13l-lifecycle-examples/v1':['analysis/alternatives/v13l_examples.py',bench+'lifecycle-examples/index.json'],
    }
    files={i:{p:hashlib.sha256((ROOT/p).read_bytes()).hexdigest() for p in ps} for i,ps in paths.items()}
    baseline=json.loads((ROOT/doc/'BASELINE.json').read_text())
    index={'schema':'dreamteam.contract-identities/v2','notice':'Independent review PENDING; human presentation review PENDING; protocol lock and live inference NO GO.',
           'identityHashEncoding':'SHA-256 of UTF-8 JSON {identity,files}, sorted keys, compact separators',
           'identities':files,'identitySha256':{i:hashlib.sha256(json.dumps({'identity':i,'files':ps},sort_keys=True,separators=(',',':')).encode()).hexdigest() for i,ps in files.items()},
           'unchanged':{'engine':'0.13.0-dev.2','dreamteam.alternatives-draw/v4':'Frozen V0.13h implementation',
                        'dreamteam.call-journal/v1':'Format preserved; strict typed binding to draw calls',
                        'dreamteam.health-observation/v2':'Format and supported families preserved; trusted class identity enforced',
                        'dreamteam.run-failure/v3':'Format preserved',
                        'dreamteam.run-result/v1':'Format preserved; RecordingError.retainedEvidence carries failed-persistence-retention/v2',
                        'dreamteam.human-review-freeze/v1':'ee987a1; 93 transcripts and seven static assets, unchanged',
                        'sealIdentities':baseline['sealIdentities']}}
    pins={'schema':'dreamteam.lifecycle-contract-pins/v1','files':{p:hashlib.sha256((ROOT/p).read_bytes()).hexdigest() for p in sorted({p for ps in paths.values() for p in ps})}}
    return {'CONTRACT-IDENTITIES.json':index,'CONTRACT-PINS.json':pins}


def main():
    ap=argparse.ArgumentParser(description=__doc__);g=ap.add_mutually_exclusive_group(required=True)
    g.add_argument('--write',action='store_true');g.add_argument('--check',action='store_true');a=ap.parse_args()
    P.verify()
    for name,obj in artifacts().items():
        path=ROOT/'docs/v13l'/name;data=(json.dumps(obj,indent=1,sort_keys=True)+'\n').encode()
        if a.write:path.write_bytes(data)
        elif path.read_bytes()!=data:raise SystemExit('contract mismatch: '+name)
    print('V0.13l successor identities, seal contract and protected evidence: PASS')

if __name__=='__main__':main()
