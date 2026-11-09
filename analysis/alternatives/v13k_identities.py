#!/usr/bin/env python3
"""Pin successor identities and independently verify protected baseline bytes."""
import argparse
import hashlib
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0,str(ROOT/'analysis/alternatives'))
import v13k_preserved as P


def artifacts():
    rec='analysis/alternatives/v13k.py';ver='analysis/alternatives/v13k_verify.py'
    spec='analysis/alternatives/bindings_v1.json'
    doc='docs/v13k/';bench='docs/benchmark/v13k-dev/'
    paths={
      'dreamteam.evidence-bindings/1':[spec,doc+'EVIDENCE_BINDINGS.md'],
      'dreamteam.health-evidence-classification/v3':[spec,'analysis/alternatives/lifecycle_v3_spec.py',rec,ver,doc+'RECORDING.md'],
      'dreamteam.run-lifecycle/v3':[spec,'analysis/alternatives/lifecycle_v3_spec.py',doc+'LIFECYCLE_CONTRACT.md'],
      'dreamteam.completed-draw-handoff/v1':[rec,spec,doc+'RECORDING.md'],
      'dreamteam.failed-persistence-retention/v1':[rec,ver,spec,doc+'RECORDING.md'],
      'dreamteam-v13k-driver/7':[rec,doc+'DRIVER.md'],
      'dreamteam.alternatives-run/v7':[rec,doc+'RECORDING.md'],
      'dreamteam.alternatives-partial-run/v7':[ver,doc+'PARTIAL_RUN_PROCEDURE.md'],
      'dreamteam.alternatives-analysis/v8':[ver],
      'dreamteam.alternatives-experiment/v7':[bench+'protocol-draft.json'],
      'dreamteam.alternatives-schedule/v4#v13k-alt-1':[bench+'schedule-draft.json'],
      'dreamteam.binding-matrix/v1':['analysis/alternatives/v13k_binding_matrix.py',doc+'binding-matrix/results.json',doc+'BINDING_MATRIX_RESULTS.md'],
      'dreamteam.v13k-fault-matrix/v1':['analysis/alternatives/v13k_faults.py','analysis/alternatives/v13k_scenarios.py',doc+'fault-matrix/matrix.json'],
      'dreamteam.v13k-signal-tests/v1':['analysis/alternatives/v13k_signals.py',doc+'signals/results.json'],
      'dreamteam.v13k-loopback/v1':['analysis/alternatives/v13k_loopback.py',doc+'loopback/results.json'],
      'dreamteam.v13k-witnesses/v1':['analysis/alternatives/v13k_witnesses.py',doc+'witnesses/results.json'],
      'dreamteam.v13k-mutations/v1':['analysis/alternatives/v13k_mutations.py',doc+'mutations/results.json'],
      'dreamteam.v13k-regressions/v1':['analysis/alternatives/v13k_regressions.py',doc+'regressions/results.json'],
      'dreamteam.v13k-completed-preservation/v1':['analysis/alternatives/v13k_preservation.py',doc+'preservation/results.json'],
      'dreamteam.v13k-preservation/v1':['analysis/alternatives/v13k_preserved.py',doc+'BASELINE.json',doc+'PRESERVATION.json'],
      'dreamteam.v13k-fidelity/v1':['analysis/alternatives/v13k_fidelity.py',bench+'fidelity/results.json',doc+'FIDELITY.md'],
      'dreamteam.v13k-lifecycle-examples/v1':['analysis/alternatives/v13k_examples.py',bench+'lifecycle-examples/index.json'],
    }
    files={i:{p:hashlib.sha256((ROOT/p).read_bytes()).hexdigest() for p in ps} for i,ps in paths.items()}
    baseline=json.loads((ROOT/doc/'BASELINE.json').read_text())
    index={'schema':'dreamteam.contract-identities/v2','notice':'Independent review PENDING; human presentation review PENDING; protocol lock and live inference NO GO.',
           'identityHashEncoding':'SHA-256 of UTF-8 JSON {identity,files}, sorted keys, compact separators',
           'identities':files,'identitySha256':{i:hashlib.sha256(json.dumps({'identity':i,'files':ps},sort_keys=True,separators=(',',':')).encode()).hexdigest() for i,ps in files.items()},
           'unchanged':{'engine':'0.13.0-dev.2','dreamteam.alternatives-draw/v4':'Frozen V0.13h implementation',
                        'dreamteam.call-journal/v1':'Format preserved; binding enforcement corrected',
                        'dreamteam.health-observation/v2':'Format and supported families preserved; trusted class identity enforced',
                        'dreamteam.run-failure/v3':'Format preserved','dreamteam.run-result/v1':'Format preserved',
                        'dreamteam.human-review-freeze/v1':'ee987a1; 93 transcripts and seven static assets, unchanged',
                        'sealIdentities':baseline['sealIdentities']}}
    pins={'schema':'dreamteam.lifecycle-contract-pins/v1','files':{p:hashlib.sha256((ROOT/p).read_bytes()).hexdigest() for p in sorted({p for ps in paths.values() for p in ps})}}
    return {'CONTRACT-IDENTITIES.json':index,'CONTRACT-PINS.json':pins}


def main():
    ap=argparse.ArgumentParser(description=__doc__);g=ap.add_mutually_exclusive_group(required=True)
    g.add_argument('--write',action='store_true');g.add_argument('--check',action='store_true');a=ap.parse_args()
    P.verify()
    for name,obj in artifacts().items():
        path=ROOT/'docs/v13k'/name;data=(json.dumps(obj,indent=1,sort_keys=True)+'\n').encode()
        if a.write:path.write_bytes(data)
        elif path.read_bytes()!=data:raise SystemExit('contract mismatch: '+name)
    print('V0.13k successor identities, seal contract and protected evidence: PASS')

if __name__=='__main__':main()
