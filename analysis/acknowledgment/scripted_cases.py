#!/usr/bin/env python3
"""The V0.12a deterministic test evidence
(dreamteam.acknowledgment-scripted-cases/v1): llm-agent 6
(llm-agent-acknowledgment-workflow/v1) at one decision point for each
scripted case below. The model replies are scripted, and so is the engine,
using agents/llm/test_llm_agent6.py's engine, whose preview answers have
the real engine's shape. No model and no engine.

    python3 -I -B analysis/acknowledgment/scripted_cases.py --out DIR

Writes scripted-cases.json (every case's decision record, the requests it
made and its model-free replay check, in a stable order) and
scripted-cases.md (a summary table). The output is a function of the agents
and the test fixtures only; CI makes it again and compares it on every
platform.
"""

import argparse
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, os.path.join(ROOT, "agents", "llm"))
import test_llm_agent6 as T  # noqa: E402  (its scripted engine and replies)

A6, A3 = T.A6, T.A3
SCHEMA = "dreamteam.acknowledgment-scripted-cases/v1"
FIRE, FIRE2, HIRE, TELE, ROSTER, OPS, STAFF = T.FIRE, T.FIRE2, T.HIRE, T.TELE, T.ROSTER, T.OPS, T.STAFF
r, ack = T.reply, T.ack_of


def changed(acts, displaced=0, **edit):
    """ack_of with one item's field replaced: edit={"item": k, "path": (...), "value": v}."""
    a = ack(acts, displaced)
    obj = a["items"][edit["item"]]
    for k in edit["path"][:-1]:
        obj = obj[k]
    obj[edit["path"][-1]] = edit["value"]
    return a


def dropped(acts, k):
    a = ack(acts)
    del a["items"][k]
    return a


# name: (replies, engine arguments, maxRetries)
CASES = {
    "ordinary_allocation": ([r("Raise Dr. Avery.", [STAFF])], {}, 2),
    "operations_only": ([r("Walk-ins.", [OPS])], {}, 2),
    "termination_acknowledged": ([r("Terminate Riley.", [FIRE]), r("Ack.", [FIRE], ack([FIRE]))], {}, 2),
    "termination_not_acknowledged": ([r("Terminate Riley.", [FIRE]), r("Do it.", [FIRE]), r("Do it.", [FIRE])], {}, 1),
    "displaced_acknowledged": ([r("Terminate Riley.", [FIRE]), r("Ack.", [FIRE], ack([FIRE], 3))],
                               {"displaced": 3}, 2),
    "roster_displacement": ([r("E05's roster.", [ROSTER]), r("Ack.", [ROSTER], ack([ROSTER], 2))],
                            {"displaced": 2}, 2),
    "hire_acknowledged": ([r("Hire.", [HIRE]), r("Ack.", [HIRE], ack([HIRE]))], {}, 2),
    "teleradiology_acknowledged": ([r("Teleradiology.", [TELE]), r("Ack.", [TELE], ack([TELE]))], {}, 2),
    "multiple_qualifying": ([r("Restructure.", [HIRE, FIRE, FIRE2]),
                             r("Ack.", [HIRE, FIRE, FIRE2], ack([HIRE, FIRE, FIRE2], 4))], {"displaced": 4}, 2),
    "one_missing": ([r("Two.", [FIRE, FIRE2]), r("Ack.", [FIRE, FIRE2], dropped([FIRE, FIRE2], 1))], {}, 0),
    "wrong_employee": ([r("Riley.", [FIRE]), r("Ack.", [FIRE], changed([FIRE], item=0, path=("target",),
                                                                        value="E04"))], {}, 0),
    "wrong_type": ([r("Riley.", [FIRE]), r("Ack.", [FIRE], changed([FIRE], item=0, path=("type",), value="hire"))],
                   {}, 0),
    "wrong_severance": ([r("Riley.", [FIRE]), r("Ack.", [FIRE], changed(
        [FIRE], item=0, path=("consequences", "severance", "severanceCents"), value=350000))], {}, 0),
    "wrong_displaced_count": ([r("Riley.", [FIRE]), r("Ack.", [FIRE], changed(
        [FIRE], 3, item=1, path=("consequences", "displaced_bookings", "count"), value=2))], {"displaced": 3}, 0),
    "stale": ([r("Riley.", [FIRE]), r("Sam instead.", [FIRE2]), r("Ack.", [FIRE2], ack([FIRE])),
               r("Ack.", [FIRE2], ack([FIRE]))], {}, 2),
    "revised_after_acknowledgment": ([r("Riley.", [FIRE]), r("Sam instead.", [FIRE2], ack([FIRE])),
                                      r("Ack.", [FIRE2], ack([FIRE2]))], {}, 2),
    "generic": ([r("Riley.", [FIRE]), r("Ack.", [FIRE], "I understand.")], {}, 0),
    "agreement_in_prose": ([r("Riley.", [FIRE]), r("I understand the severance and accept it.", [FIRE])], {}, 0),
    "unparseable_then_acknowledged": ([r("Riley.", [FIRE]), "I acknowledge.", r("Ack.", [FIRE], ack([FIRE]))],
                                      {}, 2),
    "timeout_then_acknowledged": ([r("Riley.", [FIRE]), None, r("Ack.", [FIRE], ack([FIRE]))], {}, 2),
    "consequences_unreadable": ([r("Riley.", [FIRE])], {"preview": "orphan"}, 2),
    "preview_unavailable": ([r("Riley.", [FIRE])], {"preview": "fail"}, 2),
    "mixed_acknowledged": ([r("Walk-ins and Riley.", [OPS, FIRE]), r("Ack.", [OPS, FIRE], ack([OPS, FIRE]))], {}, 2),
    "mixed_withheld": ([r("Walk-ins and Riley.", [OPS, FIRE]), r("Ack.", [OPS, FIRE])], {}, 0),
    "refused": ([r("Riley.", [FIRE]), r("Nothing.", [])], {"refuse": {"terminate": "invalid_parameters"}}, 2),
    "retries_exhausted": ([r("Riley.", [FIRE])] + [r("Ack.", [FIRE], changed([FIRE], item=0, path=("target",),
                                                                               value="E99"))] * 3, {}, 2),
    "final_check_disagrees": ([r("Riley.", [FIRE]), r("Ack.", [FIRE], ack([FIRE]))],
                              {"final": "insufficient_funds", "final_after": 1}, 2),
}


def case_record(name):
    replies, kw, retries = CASES[name]
    written, h, client = T.run(replies, T.Engine(**kw), retries)
    rec = written[0]
    again, _ = T.replay(written, T.Engine(**kw))
    same = {k: v for k, v in again[0].items() if k not in ("mode", "replayed", "replayChecked")} == \
        {k: v for k, v in rec.items() if k != "mode"}
    return {"case": name, "maxRetries": retries, "engine": kw, "submitted": h.decided[0], "record": rec,
            "engineRequests": [op for op, _ in h.log],
            "modelRequests": [{"lastMessageSha256": A3.sha256(p["messages"][-1]["content"]),
                               "lastMessageStart": p["messages"][-1]["content"][:40],
                               "messages": len(p["messages"])} for p in client.payloads],
            "replay": {"identical": same, "checked": again[0]["replayChecked"]}}


def summary_row(x):
    rec = x["record"]
    items = next((a["assessment"]["items"] for a in rec["attempts"] if (a.get("assessment") or {}).get("items")), [])
    matches = [("accepted" if a["acknowledgment"]["match"]["accepted"] else
                ",".join(f["code"] for f in a["acknowledgment"]["match"]["failures"]))
               for a in rec["attempts"] if a.get("acknowledgment")]
    w = rec.get("withheld")
    return ("| {case} | {kinds} | {items} | {matches} | {outcome} | {withheld} | {sub} | {replay} |".format(
        case=x["case"], kinds=" → ".join(a["input"]["kind"] for a in rec["attempts"]),
        items=", ".join(i["type"] for i in items) or "—", matches="; ".join(matches) or "—",
        outcome=rec["outcome"], withheld=w["reason"] if w else "—",
        sub=sum(len(b) for b in x["submitted"]), replay="identical" if x["replay"]["identical"] else "DIFFERS"))


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    cases = [case_record(c) for c in CASES]
    doc = {"schema": SCHEMA, "agent": f"{A6.AGENT} {A6.VERSION}", "workflowVersion": A6.WORKFLOW_VERSION,
           "workflowSha256": A6.WORKFLOW_SHA256, "contract": A6.CONTRACT, "contractSha256": A6.CONTRACT_SHA256,
           "recordSchema": A6.RECORD_SCHEMA, "cases": cases}
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, "scripted-cases.json"), "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True, ensure_ascii=False)
        f.write("\n")
    lines = [
        "# V0.12a scripted cases: high-impact action acknowledgment", "",
        f"llm-agent {A6.VERSION}, `{A6.WORKFLOW_VERSION}` (`{A6.WORKFLOW_SHA256[:12]}…`), contract `{A6.CONTRACT}` "
        f"(`{A6.CONTRACT_SHA256[:12]}…`). Scripted model replies and a scripted engine whose preview answers have the "
        "real engine's shape; no model and no engine. Made by `analysis/acknowledgment/scripted_cases.py`; the full "
        "records are in `scripted-cases.json`.", "",
        "Inputs: the message each attempt answered. Items: the consequence items the engine's answer required "
        "acknowledging. Matches: each acknowledgment's result. Withheld: why a proposal was not submitted. Submitted: "
        "actions handed to the harness. Replay: the workflow re-run with the recorded replies, without a model.", "",
        "| Case | Inputs | Items | Matches | Outcome | Withheld | Submitted | Replay |",
        "|---|---|---|---|---|---|---|---|"]
    lines += [summary_row(x) for x in cases]
    lines += ["", "Acknowledgment is not approval. A matching acknowledgment shows that the reply stated the engine's "
              "reported direct consequences of the exact proposal; it does not show that the action is wise, and the "
              "engine's direct consequences are not all future outcomes.", ""]
    with open(os.path.join(a.out, "scripted-cases.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(lines))
    print(f"scripted cases: {len(cases)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
