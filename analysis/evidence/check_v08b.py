#!/usr/bin/env python3
"""Check regenerated V0.8b outputs against the committed, frozen ones.

    python3 -I -B analysis/evidence/check_v08b.py --review <dir> --probes <dir>

The review package (docs/benchmark/v08b-dev/review/) and the probe log
(docs/benchmark/v08b-dev/probes-out/) record the engine build that made
them (0.8.0-dev.2). A later build that changes no behaviour regenerates
them with its own build number and nothing else different. They are
frozen, so they are not regenerated; this check requires every byte to
match except those build numbers:
* items.json: equal with each item's run.engine set aside;
* manifest.json: equal except itemsSha256, which must be the committed
  items.json's own SHA-256;
* items.md, form.csv, the probes' decisions and attempts: byte for byte;
* probe-log.json: equal with each probe's engine set aside.
Exits 1 on any other difference.
"""

import argparse
import hashlib
import json
import os
import sys

V = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "docs", "benchmark", "v08b-dev")


def raw(p):
    with open(p, "rb") as f:
        return f.read()


def main(argv=None):
    ap = argparse.ArgumentParser()
    ap.add_argument("--review", required=True)
    ap.add_argument("--probes", required=True)
    a = ap.parse_args(argv)
    bad = []
    rv, cv = os.path.join(V, "review"), a.review
    items = [json.loads(raw(os.path.join(d, "items.json"))) for d in (rv, cv)]
    for its in items:
        for it in its:
            it["run"].pop("engine", None)
    if items[0] != items[1]:
        bad.append("review/items.json differs beyond run.engine")
    man = [json.loads(raw(os.path.join(d, "manifest.json"))) for d in (rv, cv)]
    if man[0]["itemsSha256"] != hashlib.sha256(raw(os.path.join(rv, "items.json"))).hexdigest():
        bad.append("the committed manifest's itemsSha256 is not the committed items.json's")
    for m in man:
        m.pop("itemsSha256")
    if man[0] != man[1]:
        bad.append("review/manifest.json differs beyond itemsSha256")
    for f in ("items.md", "form.csv"):
        if raw(os.path.join(rv, f)) != raw(os.path.join(cv, f)):
            bad.append(f"review/{f} differs")
    po = os.path.join(V, "probes-out")
    for f in ("sequence-nocredit.decisions.jsonl", "sequence-nocredit.attempts.jsonl"):
        if raw(os.path.join(po, f)) != raw(os.path.join(a.probes, f)):
            bad.append(f"probes/{f} differs")
    logs = [json.loads(raw(os.path.join(d, "probe-log.json"))) for d in (po, a.probes)]
    for lg in logs:
        for p in lg["runs"]:
            p.pop("engine", None)
    if logs[0] != logs[1]:
        bad.append("probes/probe-log.json differs beyond each probe's engine")
    for b in bad:
        print(b)
    if bad:
        return 1
    print("V0.8b review package and probes: reproduced (build numbers aside)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
