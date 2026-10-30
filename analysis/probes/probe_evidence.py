#!/usr/bin/env python3
"""The frozen decision evidence rules (dreamteam.decision-evidence/v1)
applied to the funding probes' recording, as they would be to an agent's:
REPEATED_INVALID_PROPOSAL, and each proposed action's execution state.

    python3 -I -B analysis/probes/probe_evidence.py --probes-out <dir> --out <file>

It shows what the frozen rules see of a scripted refusal sequence. It uses
the frozen functions as they are; it changes no rule.
"""

import argparse
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, "..", "evidence"))
import decision_evidence as D  # noqa: E402  (frozen, used as is)


def run(probes_out):
    out = {"schema": "dreamteam.funding-probe-evidence/v1", "rules": "dreamteam.decision-evidence/v1",
           "recordings": []}
    for name in sorted(os.listdir(probes_out)):
        if not name.endswith(".decisions.jsonl"):
            continue
        with open(os.path.join(probes_out, name), "rb") as f:
            raw = f.read()
        recs = [json.loads(l) for l in raw.decode("utf-8").splitlines() if l.strip()]
        ctx = {"recording": {"path": "probes/" + name, "sha256": hashlib.sha256(raw).hexdigest(), "group": "probes",
                             "replayConfig": None},
               "run": {"policy": "scripted probe", "model": None, "scenario": None, "seed": None, "days": None,
                       "engine": None, "replayRun": None},
               "lines": {r["point"]: i + 1 for i, r in enumerate(recs)}, "sources": lambda rec: [],
               "completeness": lambda rec: {}}
        an = D.Analysis()
        D.check_repeats(an, ctx, recs, {})
        # The apply results: the attempt-level record beside the recording.
        applied = {}
        with open(os.path.join(probes_out, name[:-len(".decisions.jsonl")] + ".attempts.jsonl"),
                  encoding="utf-8") as f:
            for line in f:
                a = json.loads(line)
                applied[a["point"]] = a["attempts"][-1]["applied"] if a["outcome"] == "actions" else []
        execution = []
        for r in recs:
            ex = D.execution(r, applied.get(r["point"]))
            execution.append({"point": r["point"], "outcome": r["outcome"],
                              "states": [x["state"] for x in ex]})
        out["recordings"].append({
            "path": "probes/" + name, "checks": an.checks["REPEATED_INVALID_PROPOSAL"],
            "repeated": [{"id": f["id"], "type": f["repeat"]["type"], "code": f["repeat"]["code"],
                          "count": f["repeat"]["count"], "occurrences": f["repeat"]["occurrences"],
                          "withinPoint": f["repeat"]["withinPoint"]} for f in an.findings
                         if f["category"] == "REPEATED_INVALID_PROPOSAL"],
            "distinctRefusedProposals": an.checks["REPEATED_INVALID_PROPOSAL"]["statements"],
            "execution": execution})
    return out


def main(argv=None):
    ap = argparse.ArgumentParser(description="The frozen v1 rules on the probe recording.")
    ap.add_argument("--probes-out", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    doc = run(a.probes_out)
    with open(a.out, "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True, ensure_ascii=True)
        f.write("\n")
    for r in doc["recordings"]:
        print(f"{r['path']}: {r['distinctRefusedProposals']} distinct refused proposals, "
              f"{len(r['repeated'])} repeated under v1")
    return 0


if __name__ == "__main__":
    sys.exit(main())
