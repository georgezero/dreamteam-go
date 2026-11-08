"""Rebuild record-only lifecycle examples with deterministic synthetic metadata."""
import json
from pathlib import Path
import uuid
from unittest import mock

import v13h as D
import v13g_fidelity as F


def build(binary, out):
    rows,h=D.load_contexts(str(Path(D.ROOT)/'docs/benchmark/v13g-dev/contexts'))
    proto,_=D.draft_protocol(rows,h)
    ctx=D.Contexts(proto,[c for c in rows if c['kind']=='constructed'],None,
                   {D.SB.SETTINGS_ENV:D.SB.REPLAY_URL})
    sched=[{'seq':i,'model':D.QWEN,'context':'S1-a','arm':'control','draw':i} for i in (1,2)]
    cases=[('connection-before',1,{'error':'URLError'},None,None,None),
           ('connection-between',3,{'error':'URLError'},None,None,None),
           ('timeout',1,{'error':'TimeoutError'},None,None,None),
           ('malformed',1,{'error':'JSONDecodeError'},None,None,None),
           ('unhealthy',1,{'status':'unhealthy'},None,None,None),
           ('initial-interrupt',1,None,KeyboardInterrupt('scripted health interruption'),None,None),
           ('later-interrupt',3,None,KeyboardInterrupt('scripted health interruption'),None,None),
           ('engine-open',None,None,None,RuntimeError('scripted engine opening failure'),None),
           ('cleanup',None,None,None,None,RuntimeError('scripted pool cleanup failure')),
           ('primary-and-cleanup',None,None,None,RuntimeError('scripted engine opening failure'),KeyboardInterrupt('scripted pool cleanup interruption'))]
    # A two-draw fixture is not the official 262-draw experiment. Give it
    # its own explicit identity and actual protocol-file hash.
    fixture={"id":"record-only-lifecycle-fixture", "notice":"SCRIPTED RECORD ONLY: not the experimental schedule",
             "modelOrder":[D.QWEN], "contexts":{"S1-a":proto["contexts"]["S1-a"]},
             "maxModelCallsPerDraw":proto["maxModelCallsPerDraw"], "maxModelCalls":proto["maxModelCalls"],
             "request":proto["request"], "stop":proto["stop"], "schedule":sched}
    for name,at,health,interruption,opening,closing in cases:
        dest=Path(out)/name
        dest.mkdir(parents=True)
        D.atomic_json(str(dest/'protocol.json'),fixture)
        protocol_sha=D.file_sha256(str(dest/'protocol.json'))
        pool=ctx.opener(binary)
        class Pool:
            def __call__(self,c,arm):
                if opening:raise opening
                return pool(c,arm)
            def close(self):
                pool.close()
                if closing:raise closing
        count=[0]
        def state():
            count[0]+=1
            if count[0]==at:
                if interruption:raise interruption
                return {'health':health}
            return {'health':{'status':'healthy'}}
        with mock.patch.object(D.uuid,'uuid4',return_value=uuid.uuid5(uuid.NAMESPACE_URL,'record-only/'+name)):
            doc=D.run_schedule(fixture,protocol_sha,sched,ctx,Pool(),{D.QWEN:D.scripted([F.ABANDON]*2)},
                               str(dest),'scripted test',state_fn=state if at else None,now=lambda:0.,
                               utc=lambda:'2026-01-01T00:00:00Z')
        recorded=D.load_run(str(dest))
        analysis=D.analyze(binary,{},ctx,fixture,sched,recorded[1],doc,protocol_sha,recorded[2])
        D.atomic_json(str(dest/'analysis.json'),analysis)
    return len(cases)
