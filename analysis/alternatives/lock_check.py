"""Verify candidate identities against committed bytes; never changes status or authorizes a run."""
import hashlib
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[2]
MANIFEST = 'docs/v13e/FINAL_HASH_MANIFEST.txt'
BASE = 'fb8cf3ae6886926ba330770bf790ae8f0edf3646'
PRESERVED = ('internal/sim', 'internal/session', 'internal/api', 'internal/alternatives/text',
             'agents/llm', 'docs/benchmark', 'docs/v13b', 'docs/v13c', 'docs/v13d', 'testdata/golden')


def git(*args):
    return subprocess.check_output(['git', '-C', str(ROOT), *args])


def main():
    failures = []
    for path in PRESERVED:
        if git('rev-parse', 'HEAD:' + path) != git('rev-parse', BASE + ':' + path):
            failures.append('preserved tree mismatch: ' + path)
        if git('diff', '--name-only', 'HEAD', '--', path).strip():
            failures.append('working tree mismatch: ' + path)
    committed = git('show', 'HEAD:' + MANIFEST)
    if (ROOT / MANIFEST).read_bytes() != committed:
        failures.append('manifest differs from committed bytes')
    entries = [line.split('  ', 1) for line in committed.decode().splitlines() if line and not line.startswith('#')]
    proc = subprocess.Popen(['git', '-C', str(ROOT), 'cat-file', '--batch'], stdin=subprocess.PIPE, stdout=subprocess.PIPE)
    try:
        for digest, path in entries:
            proc.stdin.write(('HEAD:' + path + '\n').encode())
            proc.stdin.flush()
            header = proc.stdout.readline().split()
            if len(header) != 3 or header[1] != b'blob':
                failures.append('missing committed file: ' + path)
                continue
            data = proc.stdout.read(int(header[2]))
            assert proc.stdout.read(1) == b'\n'
            if hashlib.sha256(data).hexdigest() != digest:
                failures.append('committed hash mismatch: ' + path)
            p = ROOT / path
            if not p.is_file() or hashlib.sha256(p.read_bytes()).hexdigest() != digest:
                failures.append('working file mismatch: ' + path)
    finally:
        proc.stdin.close()
        proc.wait()
    if failures:
        raise SystemExit('\n'.join(failures))
    print(f'Engineering candidate verification: {len(entries)} file hashes and {len(PRESERVED)} preserved trees match committed HEAD')
    print('LOCK CANDIDATE - NOT LOCKED - awaiting human review')
    print('HUMAN REVIEW PENDING')


if __name__ == '__main__':
    main()
