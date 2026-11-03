#!/usr/bin/env python3
"""The V0.12b Stage B branch points (dreamteam-v12b-select/1) and draw
schedule (dreamteam-v12b-order/1): a deterministic selection from existing
evidence only, declared before any request of the experiment. No model and
no engine; no model response is read.

    python3 -I -B analysis/acknowledgment/stage_b_points.py points --out DIR
    python3 -I -B analysis/acknowledgment/stage_b_points.py points --check DIR
    python3 -I -B analysis/acknowledgment/stage_b_points.py schedule --protocol FILE --out DIR

The pool is every V0.10c/V0.11b inventory proposal that is:
* eligible under shared-prefix/v4 (no refusal before it at its point; the
  V0.11b manifest's `eligible`); and
* already has a committed context, the recorded request and proposal, in
  docs/benchmark/v10b-dev, v10c-dev or v11b-dev/points/contexts.jsonl.
  Every context is checked against its recording again.

No new trajectory is made, and no context is rebuilt from live material
outside the repository. The categories (docs/benchmark/v12b-dev/PROTOCOL.md
§2) are filled in order. Each takes the free candidate with the largest
stake, ties broken by SHA-256('dreamteam-v12b-select/1|<recording>|<point>').
A context is taken once, and at most 2 points come from one recording. A
category that has no candidate is recorded as unavailable, never filled
from elsewhere.
"""

import argparse
import collections
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, os.path.join(ROOT, "analysis", "repeatability"))
import workflow_experiment as W  # noqa: E402  (shared-prefix/v4, unchanged: its context checks)

SELECT_RULE = "dreamteam-v12b-select/1"
ORDER_RULE = "dreamteam-v12b-order/1"
POINTS_SCHEMA = "dreamteam.acknowledgment-points/v1"
SCHEDULE_SCHEMA = "dreamteam.acknowledgment-schedule/v1"
GEMMA, QWEN = W.GEMMA, W.QWEN
ARMS = ("A", "B")
MAX_PER_RECORDING = 2
SOURCES = ("docs/benchmark/v11b-dev/points/contexts.jsonl", "docs/benchmark/v10c-dev/points/contexts.jsonl",
           "docs/benchmark/v10b-dev/points/contexts.jsonl")  # the latest experiment's entry is used
INVENTORY = "docs/benchmark/v10c-dev/points/inventory.json"
MANIFEST = "docs/benchmark/v11b-dev/points/manifest.json"
# (category, validation, test, how many) for Qwen, in order.
CATEGORIES = (
    ("termination_displacement", "accepted", lambda c: "terminate" in c["types"] and c["displaced"] > 0, 1),
    ("termination_severance", "accepted", lambda c: "terminate" in c["types"], 1),
    ("hire_obligation", "accepted", lambda c: "hire" in c["types"], 1),
    ("teleradiology_guarantee", "accepted", lambda c: c["teleStarts"] > 0, 1),
    ("multiple_qualifying", "accepted", lambda c: c["items"] >= 2, 1),
    ("valid_expensive", "accepted", lambda c: True, 1),
    ("refused_termination", "refused", lambda c: "terminate" in c["types"], 1),
    ("refused_hire", "refused", lambda c: "hire" in c["types"], 2),
    ("refused_teleradiology", "refused", lambda c: c["teleStarts"] > 0, 1),
    ("nonqualifying_control", "accepted", lambda c: c["items"] == 0, 1),
)


def sha256(s):
    return hashlib.sha256(s.encode("utf-8") if isinstance(s, str) else s).hexdigest()


def load(p):
    with open(os.path.join(ROOT, p), encoding="utf-8") as f:
        return json.load(f)


def committed_contexts():
    """contextKey -> (the latest committed context entry, its source file, the line's SHA-256)."""
    out = {}
    for src in SOURCES:
        with open(os.path.join(ROOT, src), "rb") as f:
            for line in f.read().splitlines():
                if line.strip():
                    e = json.loads(line)
                    out.setdefault(e["contextKey"], (e, src, sha256(line)))
    return out


def classify(proposal_actions, inv):
    """What llm-agent 6 would ask to be acknowledged for a valid proposal,
    from the recorded actions and the inventory's preview fields: the
    qualifying types, teleradiology starts, displaced bookings, and the
    number of consequence items (one per termination, hire and
    teleradiology start, one for the batch's displacements)."""
    staff = [a for a in proposal_actions if a.get("type") in W.A3.STAFFING]
    terms = sum(1 for a in staff if a["type"] == "terminate")
    hires = sum(1 for a in staff if a["type"] == "hire")
    starts = sum(1 for a in staff if a["type"] == "contract_teleradiology" and a.get("op") == "start")
    disp = inv.get("displacedBookings") or 0
    return {"types": sorted({a["type"] for a in staff}), "terminations": terms, "hires": hires, "teleStarts": starts,
            "displaced": disp, "items": terms + hires + starts + (1 if disp else 0)}


def pool():
    inv = {x["contextKey"]: x for x in load(INVENTORY)["contexts"]}
    cands = load(MANIFEST)["candidates"]
    ctx = committed_contexts()
    rows, seen = [], set()
    for c in cands:
        k = c["contextKey"]
        if k in seen or not c["eligible"] or k not in ctx:
            continue
        seen.add(k)
        e, src, line_sha = ctx[k]
        W.check_context(e, W.S.recording_point(W.X.ROOT, e["recording"], e["point"]))
        r = {"id": e["id"], "model": c["model"], "validation": c["validation"], "recording": e["recording"],
             "point": e["point"], "attempt": e["attempt"], "contextKey": k, "stakeCents": c["stakeCents"],
             "refusals": c["refusals"], "tags": c["tags"], "source": src, "sourceLineSha256": line_sha,
             "priorExposure": sorted({s.split("/")[2].replace("-dev", "").upper() for s in SOURCES
                                      if k in {json.loads(l)["contextKey"] for l in
                                               open(os.path.join(ROOT, s), encoding="utf-8") if l.strip()}})}
        r.update(classify(e["proposal"]["actions"], inv[k]))
        rows.append(r)
    return rows


def tie(r):
    return sha256(f"{SELECT_RULE}|{r['recording']}|{r['point']}")


def select(rows):
    """Gemma: every pooled context. Qwen: the categories in order."""
    chosen, seen, per, unavailable = [], set(), collections.Counter(), []

    def free(r):
        return r["contextKey"] not in seen and per[r["recording"]] < MAX_PER_RECORDING

    def take(r, cat):
        seen.add(r["contextKey"])
        per[r["recording"]] += 1
        chosen.append(dict(r, category=cat))
    for r in sorted((r for r in rows if r["model"] == GEMMA), key=lambda r: (r["validation"] != "accepted", tie(r))):
        if free(r):
            take(r, "gemma_" + ("valid" if r["validation"] == "accepted" else "refused"))
    for cat, st, test, n in CATEGORIES:
        for _ in range(n):
            c = sorted((r for r in rows if r["model"] == QWEN and r["validation"] == st and test(r) and free(r)),
                       key=lambda r: (-r["stakeCents"], tie(r)))
            if not c:
                unavailable.append(cat)
                break
            take(c[0], cat)
    return chosen, unavailable


def manifest(rows, chosen, unavailable, contexts_sha):
    keep = ("id", "model", "validation", "category", "recording", "point", "attempt", "contextKey", "stakeCents",
            "refusals", "tags", "types", "terminations", "hires", "teleStarts", "displaced", "items", "priorExposure",
            "source", "sourceLineSha256")
    return {"schema": POINTS_SCHEMA, "rule": SELECT_RULE, "maxPerRecording": MAX_PER_RECORDING,
            "categories": [{"category": c, "validation": st, "count": n} for c, st, _, n in CATEGORIES],
            "inventory": INVENTORY, "eligibility": MANIFEST, "sources": list(SOURCES),
            "pool": [{k: r[k] for k in keep if k in r} for r in rows],
            "selected": [{k: r[k] for k in keep} for r in chosen],
            "unavailable": unavailable, "contextsFileSha256": contexts_sha}


def build():
    rows = pool()
    chosen, unavailable = select(rows)
    ctx = committed_contexts()
    body = "".join(json.dumps(dict(ctx[r["contextKey"]][0], stageB={"category": r["category"]}),
                              ensure_ascii=False, sort_keys=True) + "\n" for r in chosen).encode("utf-8")
    return manifest(rows, chosen, unavailable, sha256(body)), body


def write(d, m, body):
    os.makedirs(d, exist_ok=True)
    with open(os.path.join(d, "contexts.jsonl"), "wb") as f:
        f.write(body)
    with open(os.path.join(d, "manifest.json"), "w", encoding="utf-8") as f:
        json.dump(m, f, indent=1, sort_keys=True, ensure_ascii=False)
        f.write("\n")


def check(d):
    m, body = build()
    with open(os.path.join(d, "contexts.jsonl"), "rb") as f:
        if f.read() != body:
            raise SystemExit("contexts.jsonl differs from the selection made again")
    with open(os.path.join(d, "manifest.json"), encoding="utf-8") as f:
        if json.load(f) != json.loads(json.dumps(m)):
            raise SystemExit("manifest.json differs from the selection made again")
    return m


def schedule(proto, m):
    """Per model in modelOrder, its draws as rounds; each round has every
    point and arm of the model once, in the order of
    SHA-256('dreamteam-v12b-order/1|<model>|<round>|<point>|<arm>')."""
    out = []
    for mod in proto["modelOrder"]:
        ids = [s["id"] for s in m["selected"] if s["model"] == mod]
        for rnd in range(1, proto["drawsPerArm"][mod] + 1):
            for p, a in sorted(((p, a) for p in ids for a in ARMS),
                               key=lambda x: sha256(f"{ORDER_RULE}|{mod}|{rnd}|{x[0]}|{x[1]}")):
                out.append({"seq": len(out) + 1, "model": mod, "point": p, "arm": a, "draw": rnd})
    return out


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("points")
    g = p.add_mutually_exclusive_group(required=True)
    g.add_argument("--out")
    g.add_argument("--check")
    s = sub.add_parser("schedule")
    s.add_argument("--protocol", required=True)
    s.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    if a.cmd == "points":
        if a.check:
            m = check(a.check)
            print(f"stage B points: {len(m['selected'])} selected from {len(m['pool'])}, checked "
                  f"(unavailable: {', '.join(m['unavailable']) or 'none'})")
        else:
            m, body = build()
            write(a.out, m, body)
            print(f"stage B points: {len(m['selected'])} selected from {len(m['pool'])} "
                  f"(unavailable: {', '.join(m['unavailable']) or 'none'})")
        return 0
    with open(a.protocol, "rb") as f:
        raw = f.read()
    proto = json.loads(raw)
    pdir = os.path.join(os.path.dirname(os.path.abspath(a.protocol)), proto["points"])
    m = check(pdir)
    sched = schedule(proto, m)
    doc = {"schema": SCHEDULE_SCHEMA, "protocol": proto["id"], "protocolSha256": sha256(raw),
           "contextsFileSha256": m["contextsFileSha256"], "draws": len(sched), "concurrency": 1,
           "maxModelCallsPerDraw": proto["maxModelCallsPerDraw"], "schedule": sched}
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, "schedule.json"), "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True)
        f.write("\n")
    print(f"stage B schedule: {len(sched)} draws")
    return 0


if __name__ == "__main__":
    sys.exit(main())
