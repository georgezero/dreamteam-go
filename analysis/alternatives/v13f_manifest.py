"""Verify preserved historical bytes and successor contract identities offline."""
import hashlib
import json
from pathlib import Path

ROOT=Path(__file__).resolve().parents[2]


def main():
    count=0
    for line in (ROOT/'docs/v13f/BASELINE-HASHES.sha256').read_text().splitlines():
        want,path=line.split('  ',1)
        if hashlib.sha256((ROOT/path).read_bytes()).hexdigest()!=want:
            raise SystemExit('historical byte mismatch: '+path)
        count+=1
    pins=json.loads((ROOT/'docs/v13f/CONTRACT-PINS.json').read_text())
    for path,want in pins['files'].items():
        if hashlib.sha256((ROOT/path).read_bytes()).hexdigest()!=want:
            raise SystemExit('successor contract mismatch: '+path)
    if hashlib.sha256(pins['parserVersion'].encode()).hexdigest()!=pins['parserVersionSha256']:
        raise SystemExit('parser identity mismatch')
    print(f'Historical bytes: {count} files; successor contracts: {len(pins["files"])} files; PASS')
    print('Independent re-review: PENDING; protocol lock: NO GO; live inference: NO GO')

if __name__=='__main__':main()
