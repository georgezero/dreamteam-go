"""Reproduce the V0.13e review package from frozen rendered evidence; no network."""
import argparse
import hashlib
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[2]
DEST = ROOT / 'docs/v13e/review-package'


def artifacts():
    out = {}
    appendix = ['# Record-only appendix — not shown to the agent\n',
                'These source headers and section labels are metadata, never agent messages.\n']
    for src in sorted((ROOT / 'docs/benchmark/v13d-dev/turns').glob('S*-*.txt')):
        data = src.read_text(encoding='utf-8')
        parts = re.split(r'^=== (.+)\n', data, flags=re.M)
        assert len(parts) > 3 and 'record only, never sent' in parts[0]
        appendix += [f'## {src.name}\n', parts[0]]
        for label, body in zip(parts[1::2], parts[2::2]):
            appendix.append(label + '\n')
            # The evidence writer inserts a separator newline before each label.
            # Preserve every byte of the message itself, including its final newline.
            assert body.endswith('\n\n')
            body = body[:-1]
            suffix = label.replace(' (sent on request)', '').replace(' (sent)', '')
            suffix = suffix.lower().replace(' ', '-').replace(':', '-')
            out[f'messages/{src.stem}-{suffix}.txt'] = body.encode()
    for name in ('presentation-v2.tmpl', 'control-v1.tmpl', 'selection-v2.txt'):
        out[f'sources/{name}'] = (ROOT / 'internal/alternatives/text' / name).read_bytes()
    out['sources/view-table.txt'] = (ROOT / 'docs/benchmark/v13d-dev/presentation/view-table.txt').read_bytes()
    out['RECORD_ONLY_APPENDIX.md'] = '\n'.join(appendix).encode()
    pairs = json.loads((ROOT / 'docs/benchmark/v13d-dev/sensitivity-pairs.json').read_text())
    for pair in pairs['pairs']:
        for side in ('a', 'b'):
            for arm in ('control', 'treatment'):
                name = f"messages/{pair['id']}-{side}-{arm}.txt"
                assert hashlib.sha256(out[name]).hexdigest() == pair[side][arm + 'MessageSha256'], name
    items = ['neutrality' , 'readability', 'financial terminology', 'target identification',
             'no-action obligations', 'displaced bookings', 'slots closed', 'missing information',
             'reversibility', 'potentially misleading comparisons', 'information leakage',
             'whether the two arms differ only as declared']
    form = ['# Reviewer questionnaire\n', 'HUMAN REVIEW PENDING\n']
    for field in ('Reviewer name', 'Reviewer identity', 'Independence declaration', 'Date', 'Written findings'):
        form += [f'{field}: ____________________\n']
    for i, item in enumerate(items, 1):
        form += [f'{i}. {item}\n\nResponse: ____________________\n']
    out['QUESTIONNAIRE.md'] = '\n'.join(form).encode()
    out['README.md'] = b'''# Human presentation review package

HUMAN REVIEW PENDING

Read messages/ as the actual rendered agent-facing material. Each file is
one message, copied byte for byte from the frozen engine evidence with its
record headers and section separators removed. Inspect details are sent only
on request; reading them here does not imply an agent always sees them.
Sources are reference text, not additional agent messages. The labelled
RECORD_ONLY_APPENDIX.md is not shown to the agent. No future evaluation
outcomes are included in the messages.

Start with S1-a-control, S1-a-treatment, and S1-a-treatment-inspect-proposal
and inspect-no_action. Read both sides, including continuing salary and
severance obligations. Here 2,481 slots closed measures scheduled capacity
removed; zero displaced bookings measures appointments already booked.
They measure different things; demand for those closed slots is unavailable.
Then read S1-b and the remaining members for displaced bookings, severance,
retained commitments, unavailable information, and neutral target identifiers.
The six pairs compare contexts, with multiple visible differences.

Complete all twelve items and identity fields in QUESTIONNAIRE.md in a
separate review response. Judge understandability and balance, not agreement
with terminating an employee. Declare independence: no authorship or earlier
review of the templates or code, and no stake in the result. Do not consult
past model decisions. No engineering check substitutes for this review.

Reproduce with python3 -I analysis/alternatives/v13e_package.py --write;
check without writing with the same command and --check. SHA256SUMS covers
every included text. Its own SHA-256 identifies the package in the parent
presentation evidence document. No review has been performed.
'''
    out['SHA256SUMS'] = ''.join(f'{hashlib.sha256(v).hexdigest()}  {k}\n' for k, v in sorted(out.items())).encode()
    return out


def main():
    ap = argparse.ArgumentParser(__doc__)
    ap.add_argument('--write', action='store_true')
    ap.add_argument('--check', action='store_true')
    a = ap.parse_args()
    assert a.write != a.check, 'choose --write or --check'
    expected = artifacts()
    if a.write:
        for name, data in expected.items():
            p = DEST / name
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_bytes(data)
    else:
        actual = {str(p.relative_to(DEST)) for p in DEST.rglob('*') if p.is_file()}
        assert actual == set(expected), 'package file set differs'
        for name, data in expected.items():
            assert (DEST / name).read_bytes() == data, f'package mismatch: {name}'
    print(f'Engineering package check: {len(expected)} files; HUMAN REVIEW PENDING')


if __name__ == '__main__':
    main()
