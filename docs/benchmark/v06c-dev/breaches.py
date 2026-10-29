#!/usr/bin/env python3
"""conservative-v2's daily breach records and money conditions (PLAN.md §2),
read back from its own observe requests in each run's transcript, by the
rule of internal/bench/conservative2.go: an entry of category "reading"
whose text contains " report overdue: ", with the priority as its second
word, counted once from the decision point after it was logged.

    python3 -I -B docs/benchmark/v06c-dev/breaches.py RESULTS

Nothing is written elsewhere."""

import json
import os
import sys


def main(res):
    print("| run | decision points | urgent days | soon days | sustained (of points with 7 days' history) "
          "| trailing-30 profit < 0 | cash < $50,000 | raise conditions all met |")
    print("|---|---|---|---|---|---|---|---|")
    for run in sorted(os.listdir(os.path.join(res, "runs"))):
        if not run.endswith("conservative-v2"):
            continue
        req, days, last = {}, [], None
        neg = poor = met = sustained = history = 0
        with open(os.path.join(res, "runs", run, "transcript.jsonl"), encoding="utf-8") as f:
            for line in f:
                m = json.loads(line)
                if "op" in m:
                    req[m["id"]] = m
                    continue
                q = req.get(m.get("id"))
                if not q or q["op"] != "observe" or "recentLog" not in (q.get("args") or {}).get("sections", []):
                    continue
                o = m["result"]["observation"]
                log = o.get("recentLog") or []
                u = s = 0
                for e in log:
                    if last is not None and e["seq"] <= last:
                        continue
                    if e.get("category") != "reading" or " report overdue: " not in e.get("text", ""):
                        continue
                    w = e["text"].split()
                    u += w[1] == "urgent"
                    s += w[1] == "soon"
                if log:
                    last = max(last or 0, max(e["seq"] for e in log))
                days.append((u, s))
                profit, cash = o["finance"]["trailing30"]["operatingProfitCents"], o["finance"]["cashCents"]
                neg += profit < 0
                poor += cash < 5_000_000
                if len(days) >= 7:
                    history += 1
                    week = days[-7:]
                    sus = sum(1 for a, _ in week if a) >= 3 or sum(1 for _, b in week if b) >= 5
                    sustained += sus
                    met += sus and profit >= 0 and cash >= 5_000_000
        print(f"| {run} | {len(days)} | {sum(1 for a, _ in days if a)} | {sum(1 for _, b in days if b)} | "
              f"{sustained} of {history} | {neg} | {poor} | {met} |")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    main(sys.argv[1])
