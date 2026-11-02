#!/usr/bin/env python3
"""The V0.11a deterministic test evidence (dreamteam.workflow-scripted-cases/v1):
llm-agent 5 (llm-agent-workflow/v1) at one decision point for every scripted
case of agents/llm/test_llm_agent5.py, with scripted model replies and
scripted engine answers. No model and no engine.

    python3 -I -B analysis/workflow/scripted_cases.py --out DIR

Writes scripted-cases.json (every case's decision record and the requests
it made, in a stable order) and scripted-cases.md (a summary table). The
output is a function of the agents and the test fixtures only; CI makes it
again and compares it on every platform.
"""

import argparse
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, os.path.join(ROOT, "agents", "llm"))
import test_llm_agent5 as T  # noqa: E402  (its cases, fixtures and scripted harness)

A5 = T.A5
SCHEMA = "dreamteam.workflow-scripted-cases/v1"


def case_record(case):
    out, payloads, attempts, calls, h = T.run(case)
    batches, rec = out[0]
    return {"case": case, "batches": batches, "record": rec,
            "engineRequests": [op for op, _ in h.log],
            "modelRequests": [{"turn": c["turn"], "lastMessageSha256": A5.sha256(c["request"]["messages"][-1]["content"]),
                               "messages": len(c["request"]["messages"])} for c in calls]}


def summary_row(x):
    r = x["record"]
    pv = (r.get("stages") or {}).get("previewed")
    el = next((a["preview"]["eligibility"] for a in r["attempts"] if a.get("preview") and
               a["preview"].get("eligibility")), None)
    return ("| {case} | {kinds} | {pv} | {reasons} | {refused} | {outcome} | {sub} | {state} |".format(
        case=x["case"], kinds=" → ".join(a["input"]["kind"] for a in r["attempts"]),
        pv=pv["status"] if pv else "—", reasons=", ".join(el["reasons"]) if el and el["reasons"] else "—",
        refused=", ".join(sorted({f"{f['type']}:{f['code']}" for f in r["refusals"]})) or "—",
        outcome=r["outcome"], sub=sum(len(b) for b in r["batches"]),
        state="unchanged" if r["digestsBeforeSubmit"]["unchanged"] else "changed (nothing submitted)"))


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    cases = [case_record(c) for c in sorted(T.CASES)]
    doc = {"schema": SCHEMA, "agent": f"{A5.AGENT} {A5.VERSION}", "workflowVersion": A5.WORKFLOW_VERSION,
           "workflowSha256": A5.WORKFLOW_SHA256, "recordSchema": A5.RECORD_SCHEMA, "cases": cases}
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, "scripted-cases.json"), "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True, ensure_ascii=False)
        f.write("\n")
    lines = ["# llm-agent 5: scripted cases", "",
             f"`{A5.WORKFLOW_VERSION}` (`{A5.WORKFLOW_SHA256[:12]}…`), one decision point per case, scripted model "
             "replies and scripted engine answers (`agents/llm/test_llm_agent5.py`). Made by "
             "`analysis/workflow/scripted_cases.py`.", "",
             "| Case | Inputs, in order | Preview | Eligibility reasons | Refusals | Outcome | Actions submitted | "
             "State before submission |", "|---|---|---|---|---|---|---|---|"]
    lines += [summary_row(x) for x in cases]
    with open(os.path.join(a.out, "scripted-cases.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")
    print(f"scripted cases: {len(cases)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
