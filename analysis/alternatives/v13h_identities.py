"""Pin successor lifecycle contracts and verify scoped predecessor bytes."""
import argparse
import hashlib
import json
from pathlib import Path

ROOT=Path(__file__).resolve().parents[2]
DEST=ROOT/'docs/v13h'


def digest(path):return hashlib.sha256((ROOT/path).read_bytes()).hexdigest()


def artifacts():
    driver='analysis/alternatives/v13h.py'
    paths={
        'dreamteam-v13h-driver/4':[driver,'docs/v13h/DRIVER.md'],
        'dreamteam.alternatives-run/v4':[driver,'docs/v13h/RECORDING.md'],
        'dreamteam.alternatives-draw/v4':[driver],
        'dreamteam.alternatives-analysis/v5':[driver],
        'dreamteam.alternatives-partial-run/v4':[driver,'docs/v13h/PARTIAL_RUN_PROCEDURE.md'],
        'dreamteam.run-failure/v1':[driver,'docs/v13h/RECORDING.md'],
        'dreamteam.atomic-json/v1':[driver,'docs/v13h/RECORDING.md'],
        'dreamteam.alternatives-experiment/v4':['docs/benchmark/v13h-dev/protocol-draft.json'],
        'dreamteam.alternatives-schedule/v4':['docs/benchmark/v13h-dev/schedule-draft.json'],
        'dreamteam.v13h-fidelity/v1':['analysis/alternatives/v13h_fidelity.py','analysis/alternatives/v13h_examples.py','docs/v13h/FIDELITY.md'],
        'dreamteam.review-package/v3':['analysis/alternatives/v13h_package.py'],
    }
    index={'schema':'dreamteam.contract-identities/v2','notice':'Independent re-review PENDING; human review PENDING.',
           'identityHashEncoding':'SHA-256 of UTF-8 JSON {identity,files}, sorted keys, compact separators',
           'identities':{i:{p:digest(p) for p in ps} for i,ps in paths.items()}}
    index['identitySha256']={i:hashlib.sha256(json.dumps({'identity':i,'files':files},sort_keys=True,separators=(',',':')).encode()).hexdigest() for i,files in index['identities'].items()}
    pins={'schema':'dreamteam.lifecycle-contract-pins/v1','files':{p:digest(p) for p in sorted({p for ps in paths.values() for p in ps})}}
    return {'CONTRACT-IDENTITIES.json':index,'CONTRACT-PINS.json':pins}


def main():
    ap=argparse.ArgumentParser(__doc__);g=ap.add_mutually_exclusive_group(required=True)
    g.add_argument('--write',action='store_true');g.add_argument('--check',action='store_true');a=ap.parse_args()
    for name,obj in artifacts().items():
        data=(json.dumps(obj,indent=1,sort_keys=True)+'\n').encode()
        if a.write:(DEST/name).write_bytes(data)
        elif (DEST/name).read_bytes()!=data:raise SystemExit('contract mismatch: '+name)
    for line in (DEST/'BASELINE-HASHES.sha256').read_text().splitlines():
        want,path=line.split('  ',1)
        if digest(path)!=want:raise SystemExit('preserved bytes changed: '+path)
    print('Successor lifecycle identities and preserved bytes: PASS')


if __name__=='__main__':main()
