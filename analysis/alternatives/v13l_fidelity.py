#!/usr/bin/env python3
"""Successor recording and full replay across every scheduled context/model/arm.
Scripted record-only evidence, with historical replay inputs; no inference.
"""
import argparse
import json
from pathlib import Path
import sys
sys.path.insert(0,str(Path(__file__).resolve().parent))
import v13l as K
import v13l_verify as V
import v13l_scenarios as S


def run(binary, contexts, replays, scratch):
    rows, csha = K.H.load_contexts(contexts)
    proto, scheduled = K.draft_protocol(rows,csha)
    seen=set();schedule=[]
    for entry in scheduled:
        key=(entry['context'],entry['model'],entry['arm'])
        if key in seen:continue
        seen.add(key);schedule.append(dict(entry,seq=len(schedule)+1,draw=1))
    proto=dict(proto,id='record-only-v13l-fidelity-v1',notice='SCRIPTED RECORD ONLY: one draw per scheduled context/model/arm',draws=len(schedule),schedule=schedule)
    proto_sha=K.sha256(K.canon(proto))
    ctx=K.Contexts(proto,rows,replays,{K.SB.SETTINGS_ENV:K.SB.REPLAY_URL})
    clock=S.Clock();root=Path(scratch);root.mkdir(parents=True)
    # Empty-action proposals and replies exercise both arms without new seeds.
    # Comparison retries remain within the existing context bounds.
    reply=json.dumps({'reason':'scripted fidelity response','actions':[]})
    transports={m:K.H.scripted([reply]*802) for m in proto['modelOrder']}
    result=K.run_schedule(proto,proto_sha,schedule,ctx,ctx.opener(binary),transports,str(root),
                          'scripted fidelity',state_fn=S.Health(clock),now=lambda:0.0,utc=clock,
                          run_id=S.RUN_ID,provenance=S.SYNTHETIC)
    analysis=V.analyze(binary,ctx.run_dirs,ctx,proto,schedule,str(root),proto_sha,allow_synthetic=True)
    ledger=json.loads((root/'live/draws.json').read_text())
    if result['state']!='COMPLETE' or analysis['runStatus']!='COMPLETE' or len(ledger)!=len(schedule):
        raise K.ExperimentError('successor fidelity did not complete every scheduled unit')
    cases=[{'seq':r['seq'],'context':r['context'],'model':r['model'],'arm':r['arm'],
            'status':r['status'],'calls':r['modelCallsCompleted'],
            'engineEvents':len(r['engine']),'initialDigests':r['initialDigests'],
            'finalDigests':r['digestsAfter'],'submittedRepresentationSha256':r['submittedRepresentationSha256'],
            'agree':r['status']=='completed'} for r in ledger]
    K.write_json(str(root/'analysis.json'),analysis)
    K.write_json(str(root/'protocol.json'),proto)
    return {'schema':'dreamteam.v13l-fidelity/v1','driver':K.DRIVER,
            'lifecycle':{'id':K.LS.LIFECYCLE_ID,'sha256':K.LS.spec_sha256()},
            'workflow':{'version':K.H.ALT.WORKFLOW_VERSION,'sha256':K.H.ALT.WORKFLOW_SHA256},
            'historicalContexts':sum(c['kind']=='historical' for c in rows),'contexts':len(rows),
            'cases':cases,'summary':{'cases':len(cases),'agree':sum(c['agree'] for c in cases),
            'modelCalls':sum(c['calls'] for c in cases),'engineEvents':sum(c['engineEvents'] for c in cases),
            'fullAnalysis':analysis['runStatus'],'completedVerification':analysis['completedVerification']}}


def main():
    ap=argparse.ArgumentParser(description=__doc__)
    for name in ['binary','contexts','replays','scratch','out']:ap.add_argument('--'+name,required=True)
    a=ap.parse_args();doc=run(a.binary,a.contexts,a.replays,a.scratch);K.write_json(a.out,doc)
    print(json.dumps(doc['summary']))
    if doc['summary']['agree']!=doc['summary']['cases']:raise SystemExit(1)

if __name__=='__main__':main()
