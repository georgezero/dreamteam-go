#!/usr/bin/env python3
"""Verify every protected baseline file and the unchanged experiment design."""
import argparse
import hashlib
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0,str(ROOT/'analysis/alternatives'))
import v13k as K


def digest(data): return hashlib.sha256(data.encode() if isinstance(data,str) else data).hexdigest()
def decoded(path): return bytes.fromhex(path[4:]).decode() if path.startswith('hex:') else path


def verify():
    baseline=json.loads((ROOT/'docs/v13k/BASELINE.json').read_text())
    after={}
    for encoded,want in baseline['protectedFiles'].items():
        path=decoded(encoded)
        got=digest((ROOT/path).read_bytes())
        if got!=want: raise SystemExit('protected bytes changed: '+encoded)
        after[path]=got
    contexts,contexts_sha=K.H.load_contexts(str(ROOT/'docs/benchmark/v13g-dev/contexts'))
    proto,schedule=K.draft_protocol(contexts,contexts_sha)
    design={'nine historical contexts':[x for x in contexts if x['kind']=='historical'],
            'six sensitivity pairs':{k:[x for x in contexts if x.get('pair')==k] for k in sorted({x['pair'] for x in contexts if x.get('pair')})},
            '262 scheduled draws':schedule,'802-call ceiling':proto['maxModelCalls']}
    groups={}
    for name,g in baseline['groups'].items():
        if g['files']:
            text=''.join(after[decoded(p)]+'  '+decoded(p)+'\n' for p in sorted(g['files'],key=decoded))
            got=digest(text.encode())
        else: got=digest(json.dumps(design[name],sort_keys=True,separators=(",",":")))
        groups[name]={'count':g['count'],'before':g['sha256'],'after':got,'byteIdentical':got==g['sha256']}
        if got!=g['sha256']:raise SystemExit('preservation group changed: '+name)
    old=json.loads((ROOT/'analysis/alternatives/lifecycle_v2.json').read_text())
    if old['finalization']!=K.LS.load()['finalization']:raise SystemExit('seal finalization contract changed')
    return dict(schema='dreamteam.v13k-preservation/v1',startingCommit=baseline['commit'],
                protectedFiles=len(after),groups=groups,sealContractUnchanged=True,humanReview='PENDING')


def main():
    ap=argparse.ArgumentParser(description=__doc__);ap.add_argument('--out');a=ap.parse_args()
    doc=verify()
    if a.out:K.write_json(a.out,doc)
    print(str(doc['protectedFiles'])+' protected files and all experiment/presentation groups: PASS')

if __name__=='__main__':main()
