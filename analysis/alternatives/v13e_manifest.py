"""Write deterministic SHA-256 identities for the candidate's repository files."""
import argparse
import hashlib
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[2]
TARGET = 'docs/v13e/FINAL_HASH_MANIFEST.txt'
EXCLUDE = {TARGET, 'docs/audits/.slice-5c4m-provenance-protocol-lock-audit.md.swp', 'dt'}


def main():
    ap = argparse.ArgumentParser(__doc__)
    ap.add_argument('--write', action='store_true', required=True)
    ap.parse_args()
    names = subprocess.check_output(['git', '-C', str(ROOT), 'ls-files', '-co', '--exclude-standard', '-z'])
    paths = sorted(set(names.decode().split('\0')) - EXCLUDE - {''})
    lines = ['# V0.13e candidate file identities; SHA-256; self excluded.',
             '# LOCK CANDIDATE - NOT LOCKED - awaiting human review',
             '# HUMAN REVIEW PENDING']
    for name in paths:
        p = ROOT / name
        assert p.is_file(), name
        lines.append(f'{hashlib.sha256(p.read_bytes()).hexdigest()}  {name}')
    (ROOT / TARGET).write_text('\n'.join(lines) + '\n', encoding='utf-8')
    print(f'Candidate manifest: {len(paths)} files')


if __name__ == '__main__':
    main()
