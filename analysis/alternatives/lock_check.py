"""Verify candidate identities against committed bytes; never changes status or authorizes a run."""
import hashlib
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[2]
MANIFEST = 'docs/v13e/FINAL_HASH_MANIFEST.txt'
BASE = 'fb8cf3ae6886926ba330770bf790ae8f0edf3646'
PRESERVED = {
    'internal/sim': 'c95fb739fcd940ba961a28eeaf5bad35b45c4eaa',
    'internal/session': 'bcc4e005925190179eabfa4de7b977eec0271999',
    'internal/api': '0318986604edc8ae5d05e80059ba149763cf2f22',
    'internal/alternatives/text': 'b51b2f6e766534986e4d2792451ef0dbd501878c',
    'agents/llm': '7f97277f2795d060a624123d8c06ee840b7a28e0',
    'docs/benchmark': '8be4c172acc62f561779a99c6c1c26aa2edc696d',
    'docs/v13b': '3155c0908add0a03dca29058cead2cc7f8bdabe9',
    'docs/v13c': '3d8058c74d6f9e6238f0504d808dea5cb2c4c24f',
    'docs/v13d': '25a4492f62c45003e048e4d0507d2879c3664114',
    'testdata/golden': '2adbfd24f5689b1cc9edd1238fa5eb1f796267c3',
}



def git(*args):
    return subprocess.check_output(['git', '-C', str(ROOT), *args])


def main():
    failures = []
    for path, expected_tree in PRESERVED.items():
        if git('rev-parse', 'HEAD:' + path).decode().strip() != expected_tree:
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
