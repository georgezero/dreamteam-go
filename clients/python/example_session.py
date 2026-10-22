#!/usr/bin/env python3
"""An end-to-end DreamTeam session through the agent protocol, from outside.

It starts `dreamteam agent --stdio` and, using only the protocol:

1. reads hello, the observation sections and the action catalog;
2. observes compactly;
3. validates, previews and applies a reading-allocation change, built
   from the catalog's targets;
4. checks that an invalid batch is refused without changing any digest;
5. advances day by day;
6. saves the session;
7. asks for the T90 evaluation (--t90);
8. ends the session.

It then checks the save with the real headless commands: `dreamteam
verify` must report the run digest the protocol reported, and `dreamteam
t90 -format json` must give the evaluation the protocol gave. It exits
0 when every check passes. Standard library only.

    python3 clients/python/example_session.py --binary ./dreamteam --days 5 --t90
"""

import argparse
import datetime
import json
import os
import subprocess
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from dreamteam_agent import AgentClient, ProtocolError  # noqa: E402


def check(cond, what):
    if not cond:
        raise SystemExit(f"FAILED: {what}")


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--binary", default="./dreamteam")
    ap.add_argument("--scenario", default="sustainable")
    ap.add_argument("--seed", type=int, default=906)
    ap.add_argument("--days", type=int, default=5)
    ap.add_argument("--save", default="python-client.session.json")
    ap.add_argument("--transcript", default="")
    ap.add_argument("--t90", action="store_true", help="also evaluate (two ~200-day simulations)")
    a = ap.parse_args()
    binary = os.path.abspath(a.binary)
    transcript = open(a.transcript, "w", encoding="utf-8") if a.transcript else None
    summary = {}
    try:
        with AgentClient(binary, ["-scenario", a.scenario, "-seed", str(a.seed)], transcript) as c:
            hello = c.request("hello")
            check(hello["protocol"] == "dreamteam.agent/v1", "protocol version")
            check(hello["scenario"]["ref"] == a.scenario and hello["seed"] == a.seed, "the session is the one asked for")
            check({"observe", "sections", "actions", "validate", "preview", "apply", "advance", "save", "digests", "t90"} <= set(hello["ops"]), "ops")
            sections = c.request("sections")
            check(any(s["name"] == "worklist" for s in sections["sections"]), "sections listed")

            obs = c.request("observe", {"compact": True, "recentLog": 0})
            check(obs["compact"] and "worklist" in obs["observation"], "compact observation")
            small = c.request("observe", {"sections": ["worklist", "alerts"], "recentLog": 0})
            check(set(small["sections"]) >= {"worklist", "alerts", "time"} and "finance" not in small["observation"], "sectioned observation")
            try:
                c.request("observe", {"sections": ["no-such-section"]})
                check(False, "an unknown section is refused")
            except ProtocolError as e:
                check(e.code == "unknown_section", f"unknown_section, got {e.code}")

            catalog = c.request("actions")
            alloc = next(t for t in catalog["types"] if t["type"] == "set_reading_allocation")
            target = alloc["targets"][0]
            action = {"type": "set_reading_allocation", "op": "set", "employee": target["employee"],
                      "effective": target["earliestEffective"], "siteAllocationBP": min(target["maxBP"], target["inEffectBP"] + 250)}

            before = c.request("digests")
            invalid = dict(action, siteAllocationBP=100)
            v = c.request("validate", {"actions": [invalid]})
            check(not v["valid"] and v["results"][0]["code"] == "allocation_out_of_band", "the invalid batch is refused by validate")
            r = c.request("apply", {"actions": [invalid]})
            check(not r["applied"] and r["clock"]["digests"] == before["digests"], "a refused batch changes nothing")

            v = c.request("validate", {"actions": [action]})
            check(v["valid"], "the batch is valid")
            p = c.request("preview", {"actions": [action]})
            check(p["preview"]["ok"], "the preview accepts it")
            check(c.request("digests")["digests"] == before["digests"], "validate and preview change nothing")
            r = c.request("apply", {"actions": [action]})
            check(r["applied"] and r["results"][0]["code"] == "accepted", "applied")

            date = hello["clock"]["time"][:10]
            for _ in range(a.days):
                y, m, d = map(int, date.split("-"))
                date = (datetime.date(y, m, d) + datetime.timedelta(days=1)).isoformat()
                adv = c.request("advance", {"to": date + "T07:00:00"})
                check(adv["reason"] == "reached" and adv["clock"]["time"] == date + "T07:00:00", "advance")

            saved = c.request("save", {"path": a.save})
            check(saved["batches"] == 1, "one decision batch recorded")
            run_digest = saved["clock"]["digests"]["run"]
            evaluation = None
            if a.t90:
                t = c.request("t90")
                evaluation = t["evaluation"]
                check(evaluation["policyId"] == "v0.2-eval-4" and t["seed"] == a.seed, "the policy in force, on this seed")
                check(evaluation["projected"] == (evaluation["asOfDay"] < 90), "projection labelled")
                after = c.request("digests")
                check(after["digests"] == saved["clock"]["digests"], "t90 changes nothing")
                summary["verdict"] = evaluation["report"]["comparisons"][0]["verdict"]
        summary.update(scenario=a.scenario, seed=a.seed, days=a.days, run=run_digest)
    finally:
        if transcript:
            transcript.close()

    # The save, checked by the headless commands.
    out = subprocess.run([binary, "verify", a.save], capture_output=True, text=True, check=False)
    check(out.returncode == 0 and out.stdout.split() == ["ok", run_digest], f"dreamteam verify: {out.stdout} {out.stderr}")
    if evaluation is not None:
        out = subprocess.run([binary, "t90", a.save, "-format", "json"], capture_output=True, text=True, check=False)
        check(out.returncode == 0 and json.loads(out.stdout) == evaluation, "dreamteam t90 gives the protocol's evaluation")
    summary["checks"] = "passed"
    print(json.dumps(summary, sort_keys=True))


if __name__ == "__main__":
    main()
