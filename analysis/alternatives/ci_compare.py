"""Fail closed on every historical analysis mismatch before artifact upload."""
import argparse
from pathlib import Path
import shutil


def compare(generated, reference, out, prefix, files):
    for name in files:
        if (generated / name).read_bytes() != (reference / name).read_bytes():
            raise ValueError('historical analysis mismatch: ' + name)
    out.mkdir(parents=True, exist_ok=True)
    for name in files:
        shutil.copyfile(generated / name, out / (prefix + name))


def main():
    p = argparse.ArgumentParser(__doc__)
    for name in ('generated', 'reference', 'out'):
        p.add_argument('--' + name, required=True, type=Path)
    p.add_argument('--prefix', default='')
    p.add_argument('files', nargs='+')
    a = p.parse_args()
    compare(a.generated, a.reference, a.out, a.prefix, a.files)


if __name__ == '__main__':
    main()
