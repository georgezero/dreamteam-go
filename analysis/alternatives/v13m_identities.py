#!/usr/bin/env python3
"""Pin successor identities and independently verify protected baseline bytes."""
import argparse
import hashlib
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0,str(ROOT/'analysis/alternatives'))
import v13m_preserved as P


def artifacts():
    rec='analysis/alternatives/v13m.py';ver='analysis/alternatives/v13m_verify.py'
    spec='analysis/alternatives/bindings_v2.json'
    acc='analysis/alternatives/acceptance_v1.json'
    doc='docs/v13m/';bench='docs/benchmark/v13m-dev/';prev='docs/v13l/'
    paths={
      'dreamteam.acceptance-standard/v1':[acc,doc+'ACCEPTANCE_CONTRACT.md'],
      'dreamteam.evidence-bindings/2':[spec,prev+'EVIDENCE_BINDINGS.md'],
      'dreamteam.health-evidence-classification/v3':[spec,'analysis/alternatives/lifecycle_v4_spec.py',rec,ver,doc+'RECORDING.md'],
      'dreamteam.run-lifecycle/v4':[spec,'analysis/alternatives/lifecycle_v4_spec.py',prev+'LIFECYCLE_CONTRACT.md',doc+'ACCEPTANCE_CONTRACT.md'],
      'dreamteam.strict-json/v1':[spec,acc,ver,prev+'EVIDENCE_BINDINGS.md'],
      'dreamteam.response-handoff/v1':[rec,spec,doc+'RECORDING.md'],
      'dreamteam.completed-draw-handoff/v1':[rec,spec,acc,doc+'RECORDING.md'],
      'dreamteam.failed-persistence-retention/v3':[rec,ver,acc,doc+'RECORDING.md'],
      'dreamteam-v13m-driver/9':[rec,doc+'DRIVER.md'],
      'dreamteam.alternatives-run/v9':[rec,doc+'RECORDING.md'],
      'dreamteam.alternatives-partial-run/v9':[ver,doc+'PARTIAL_RUN_PROCEDURE.md'],
      'dreamteam.alternatives-analysis/v10':[ver],
      'dreamteam.alternatives-experiment/v9':[bench+'protocol-draft.json'],
      'dreamteam.alternatives-schedule/v4#v13m-alt-1':[bench+'schedule-draft.json'],
      'dreamteam.binding-matrix/v2':['analysis/alternatives/v13m_binding_matrix.py',doc+'binding-matrix/results.json'],
      'dreamteam.v13m-fault-matrix/v1':['analysis/alternatives/v13m_faults.py','analysis/alternatives/v13m_scenarios.py',doc+'fault-matrix/matrix.json'],
      'dreamteam.v13m-signal-tests/v1':['analysis/alternatives/v13m_signals.py',doc+'signals/results.json'],
      'dreamteam.v13m-loopback/v1':['analysis/alternatives/v13m_loopback.py',doc+'loopback/results.json'],
      'dreamteam.v13m-witnesses/v1':['analysis/alternatives/v13m_witnesses.py',doc+'witnesses/results.json'],
      'dreamteam.v13m-mutations/v1':['analysis/alternatives/v13m_mutations.py',doc+'mutations/results.json'],
      'dreamteam.v13m-regressions/v1':['analysis/alternatives/v13m_regressions.py',doc+'regressions/results.json'],
      'dreamteam.v13m-completed-preservation/v2':['analysis/alternatives/v13m_preservation.py',doc+'preservation/results.json'],
      'dreamteam.v13m-dt13l-witnesses/v1':['analysis/alternatives/v13m_dt13l.py',doc+'dt13l/before-v13l.json',doc+'dt13l/before-prefix-successor.json',doc+'dt13l/after-v13m.json'],
      'dreamteam.v13m-review-probes/v1':['analysis/alternatives/v13m_review_probes.py',doc+'review/probes-before-v13l.json',doc+'review/probes-after-v13m.json'],
      'dreamteam.v13l-dt13k-witnesses/v1#v13m':['analysis/alternatives/v13l_dt13k.py',doc+'dt13k/on-v13m.json'],
      'dreamteam.v13m-preservation/v1':['analysis/alternatives/v13m_preserved.py',doc+'BASELINE.json',doc+'PRESERVATION.json'],
      'dreamteam.v13m-fidelity/v1':['analysis/alternatives/v13m_fidelity.py',bench+'fidelity/results.json'],
      'dreamteam.v13m-lifecycle-examples/v1':['analysis/alternatives/v13m_examples.py',bench+'lifecycle-examples/index.json'],
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
                        'dreamteam.run-result/v1':'Format preserved; RecordingError.retainedEvidence carries failed-persistence-retention/v3 (adds evidenceLoss)',
                        'dreamteam.human-review-freeze/v1':'ee987a1; 93 transcripts and seven static assets, unchanged',
                        'sealIdentities':baseline['sealIdentities']}}
    pins={'schema':'dreamteam.lifecycle-contract-pins/v1','files':{p:hashlib.sha256((ROOT/p).read_bytes()).hexdigest() for p in sorted({p for ps in paths.values() for p in ps})}}
    return {'CONTRACT-IDENTITIES.json':index,'CONTRACT-PINS.json':pins}


def main():
    ap=argparse.ArgumentParser(description=__doc__);g=ap.add_mutually_exclusive_group(required=True)
    g.add_argument('--write',action='store_true');g.add_argument('--check',action='store_true');a=ap.parse_args()
    P.verify()
    for name,obj in artifacts().items():
        path=ROOT/'docs/v13m'/name;data=(json.dumps(obj,indent=1,sort_keys=True)+'\n').encode()
        if a.write:path.write_bytes(data)
        elif path.read_bytes()!=data:raise SystemExit('contract mismatch: '+name)
    print('V0.13m-lite successor identities, seal contract and protected evidence: PASS')

if __name__=='__main__':main()
