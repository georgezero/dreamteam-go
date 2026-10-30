#!/usr/bin/env python3
"""Refusal-response evidence (dreamteam.refusal-response/v1): what an agent
proposed next after a proposal of its was refused.

    python3 -I -B analysis/evidence/refusal_response.py \
        --corpus docs/benchmark/v08a-dev/corpus.json \
        [--probes <funding probe output dir>] --out <dir>

An extension beside the frozen decision evidence (dreamteam.decision
-evidence/v1), which it does not change. It reads recordings only: the
llm-agent 2 recording format (point, outcome, proposed, validation,
refusals) and, where it exists, attempt-level evidence (each attempt's own
actions and validation, as the funding probes write it). No replay is
needed, and no model is contacted.

A refusal is an action refused by validate with any code but
batch_rejected (which only follows another action's refusal). Its
response is the next proposal the same agent made: the next attempt in
the same decision point if that attempt's actions are recorded, otherwise
the next decision point's proposal. Each refusal is classified by exact
action identity: the action's canonical JSON without its requestId (a
label the client chooses).

| Category | When |
|---|---|
| identical_repeated | the response contains the identical action, and it was refused again or not submitted |
| refusal_no_longer_applicable | the response contains the identical action, and validate accepted it |
| modified_proposal | the response contains an action of the same type for the same target, with other values |
| different_proposal | the response proposes actions, none of them the refused action's type and target |
| no_action | the response proposes no action |
| evidence_unavailable | the refused action's own payload is not recorded (an earlier attempt in the llm-agent 2 format), or no later proposal is recorded |

The categories describe what was proposed. None of them says whether the
agent understood the refusal, and a different proposal is not called
adaptive.
"""

import argparse
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import decision_evidence as D  # noqa: E402  (the frozen analyzer: corpus and model helpers only)

SCHEMA = "dreamteam.refusal-response/v1"
CATEGORIES = ("identical_repeated", "refusal_no_longer_applicable", "modified_proposal", "different_proposal",
              "no_action", "evidence_unavailable")


def identity(a):
    """An action's exact identity: its canonical JSON without requestId."""
    return json.dumps({k: v for k, v in a.items() if k != "requestId"}, sort_keys=True, separators=(",", ":"))


def target(a):
    """What an action acts on: its type and target key."""
    t = a.get("type")
    if t in ("extend_shift", "terminate", "set_roster", "set_reading_allocation"):
        return (t, a.get("employee"))
    if t == "hire":
        return (t, a.get("role"), tuple(sorted(a.get("skills") or [])))
    if t == "contract_teleradiology":
        return (t, a.get("op"), a.get("contract") if a.get("op") == "cancel" else a.get("plan"))
    return (t,)


def classify(refused, response):
    """The category of one refused action given the response: a list of
    (action, accepted-by-validate) pairs, [] for no action, or None if no
    response is recorded."""
    if refused is None or response is None:
        return "evidence_unavailable"
    if not response:
        return "no_action"
    rid, rt = identity(refused), target(refused)
    same = [ok for a, ok in response if identity(a) == rid]
    if same:
        return "refusal_no_longer_applicable" if any(same) else "identical_repeated"
    if any(target(a) == rt for a, _ in response):
        return "modified_proposal"
    return "different_proposal"


def attempts_of(rec, attempt_rec):
    """The decision point's attempts as [(attempt, actions or None,
    [(action index, accepted)])]: actions are known for every attempt
    in attempt-level evidence, and only for the final reply in the
    llm-agent 2 format."""
    if attempt_rec is not None:
        out = []
        for at in attempt_rec["attempts"]:
            ok = {}
            pos = 0
            batches = []
            for v in at.get("validation") or []:
                batches.append(v)
            # Validation results follow the agent's batches (operations, then staffing).
            parts = D.A.split(at["actions"]) if at["actions"] else []
            for (batch, idx), v in zip(parts, batches):
                for k, r in enumerate(v.get("results") or []):
                    ok[idx[k]] = bool(r.get("ok"))
            out.append((at["attempt"], at["actions"], ok))
            pos += 1
        return out
    proposed = rec.get("proposed")
    ok = {}
    if isinstance(proposed, list) and proposed and isinstance(rec.get("validation"), list):
        for (batch, idx), v in zip(D.A.split(proposed), rec["validation"]):
            for k, r in enumerate(v.get("results") or []):
                ok[idx[k]] = bool(r.get("ok"))
    final = max([r.get("attempt") or 0 for r in rec.get("refusals") or []] + [rec.get("attempts") or 1])
    return [(final, proposed if isinstance(proposed, list) else None, ok)]


def refused_actions(rec, attempt_rec):
    """Each refusal of a decision point: (attempt, action index, type, code,
    message, payload or None)."""
    out = []
    if attempt_rec is not None:
        for at in attempt_rec["attempts"]:
            parts = D.A.split(at["actions"]) if at["actions"] else []
            for (batch, idx), v in zip(parts, at.get("validation") or []):
                for k, r in enumerate(v.get("results") or []):
                    if not r.get("ok") and r.get("code") != "batch_rejected":
                        out.append((at["attempt"], idx[k], batch[k]["type"], r.get("code"), r.get("message"),
                                    batch[k]))
        return out
    proposed = rec.get("proposed") if isinstance(rec.get("proposed"), list) else []
    final_refused = set()
    if proposed and isinstance(rec.get("validation"), list):
        for (batch, idx), v in zip(D.A.split(proposed), rec["validation"]):
            for k, r in enumerate(v.get("results") or []):
                if not r.get("ok"):
                    final_refused.add((idx[k], r.get("code")))
    last = max([r.get("attempt") or 0 for r in rec.get("refusals") or []] or [0])
    for r in D.refusals_of(rec) or []:
        if r.get("code") == "batch_rejected":
            continue
        payload = None
        if r.get("attempt") == last and (r.get("action"), r.get("code")) in final_refused:
            payload = proposed[r["action"]]
        out.append((r.get("attempt"), r.get("action"), r.get("type"), r.get("code"), r.get("message"), payload))
    return out


def analyze_sequence(recs, attempt_recs, source):
    """Every refusal of one recording with its response."""
    events = []
    by_point = [attempts_of(r, attempt_recs[i] if attempt_recs else None) for i, r in enumerate(recs)]
    for i, rec in enumerate(recs):
        arec = attempt_recs[i] if attempt_recs else None
        for attempt, idx, typ, code, msg, payload in refused_actions(rec, arec):
            # The response: the next attempt of this point with its actions recorded, else the next point.
            resp, where = None, None
            later = [a for a in by_point[i] if a[0] > attempt]
            if later:
                at, actions, ok = later[0]
                if actions is not None:
                    resp, where = [(a, ok.get(k, False)) for k, a in enumerate(actions)], {"point": rec["point"],
                                                                                          "attempt": at}
            elif i + 1 < len(recs):
                nxt = by_point[i + 1]
                at, actions, ok = nxt[0] if arec is not None else nxt[-1]
                if actions is not None:
                    resp, where = [(a, ok.get(k, False)) for k, a in enumerate(actions)], {"point": recs[i + 1]["point"],
                                                                                          "attempt": at}
            cat = classify(payload, resp)
            why = None
            if payload is None:
                why = "the refused action's payload is not recorded (an earlier attempt)"
            elif resp is None:
                why = "no later proposal is recorded"
            key = json.dumps([source["sha256"], rec["point"], attempt, idx, code], sort_keys=True)
            events.append({
                "id": "rr1-" + hashlib.sha256(key.encode("utf-8")).hexdigest()[:16],
                "recording": source["path"], "group": source["group"], "model": source["model"],
                "point": rec["point"], "time": rec.get("time"), "attempt": attempt, "action": idx, "type": typ,
                "code": code, "message": msg, "refused": D.short_action(payload) if payload else None,
                "response": where,
                "responseActions": None if resp is None else [{"action": D.short_action(a), "acceptedByValidate": ok}
                                                             for a, ok in resp],
                "category": cat, "unavailable": why, "responseRecorded": resp is not None})
    return events


def jsonl(path):
    with open(path, "rb") as f:
        raw = f.read()
    return raw, [json.loads(l) for l in raw.decode("utf-8").splitlines() if l.strip()]


def run(corpus_path, probes_dir=None):
    sources = []
    with open(corpus_path, encoding="utf-8") as f:
        corpus = json.load(f)
    for g in corpus["groups"]:
        for rp in g["replays"]:
            with open(rp["config"], encoding="utf-8") as f:
                recmap = D.replay_recordings(json.load(f))
            for pid in sorted(recmap):
                for path in recmap[pid]:
                    sources.append((g["id"], path, None))
    if probes_dir:
        for name in sorted(os.listdir(probes_dir)):
            if name.endswith(".decisions.jsonl"):
                stem = name[:-len(".decisions.jsonl")]
                sources.append(("probes", os.path.join(probes_dir, name),
                                os.path.join(probes_dir, stem + ".attempts.jsonl")))
                # The same probes read in the llm-agent 2 format alone: what
                # the historical format lets this analysis see.
                sources.append(("probes-v2-format", os.path.join(probes_dir, name), None))
    events, inputs = [], []
    for group, path, apath in sources:
        raw, recs = jsonl(path)
        arecs = jsonl(apath)[1] if apath and os.path.exists(apath) else None
        probe = group.startswith("probes")
        name = path if not probe else "probes/" + os.path.basename(path)
        src = {"path": name, "sha256": hashlib.sha256(raw).hexdigest(), "group": group,
               "model": D.model_of(path, D.Inputs()) if not probe else "scripted probe"}
        if not any(i["path"] == name for i in inputs):
            inputs.append({"path": name, "sha256": src["sha256"]})
        events += analyze_sequence(recs, arecs, src)
    summary = {}
    for e in events:
        k = f"{e['group']}|{e['model']}"
        s = summary.setdefault(k, {"refusals": 0, "insufficientFunds": 0, "payloadUnrecordedResponseRecorded": 0,
                                   **{c: 0 for c in CATEGORIES}})
        s["refusals"] += 1
        s["payloadUnrecordedResponseRecorded"] += e["refused"] is None and e["responseRecorded"]
        s["insufficientFunds"] += e["code"] == "insufficient_funds"
        s[e["category"]] += 1
    return {"schema": SCHEMA, "categories": list(CATEGORIES),
            "identity": "canonical JSON of the action without requestId",
            "inputs": inputs, "summary": {k: summary[k] for k in sorted(summary)}, "events": events}


def tables(doc):
    L = ["# Refusal-response tables", "", f"`{doc['schema']}`. What was proposed next after a refusal; not why.", "",
         "| Group | Model | Refusals | insufficient_funds | " + " | ".join(doc["categories"])
         + " | of which unavailable only for the refused payload |",
         "|---|---|---|---|" + "---|" * len(doc["categories"]) + "---|"]
    for k, s in doc["summary"].items():
        g, m = k.split("|")
        L.append(f"| {g} | {m} | {s['refusals']} | {s['insufficientFunds']} | "
                 + " | ".join(str(s[c]) for c in doc["categories"]) + f" | {s['payloadUnrecordedResponseRecorded']} |")
    probe = [e for e in doc["events"] if e["group"].startswith("probes")]
    if probe:
        L += ["", "## Probe refusals", "", "| Read as | Recording | Point | Attempt | Refused | Code | Response | Category |",
              "|---|---|---|---|---|---|---|---|"]
        for e in probe:
            r = e["response"]
            L.append(f"| {'attempts' if e['group'] == 'probes' else 'llm-agent 2 format'} | "
                     f"`{os.path.basename(e['recording'])}` | {e['point']} | {e['attempt']} | {e['type']} | "
                     f"{e['code']} | {'point %s attempt %s' % (r['point'], r['attempt']) if r else '—'} | "
                     f"{e['category']} |")
    return "\n".join(L) + "\n"


def main(argv=None):
    ap = argparse.ArgumentParser(description="Refusal-response evidence (read-only; no model).")
    ap.add_argument("--corpus", required=True)
    ap.add_argument("--probes", help="a funding probe output directory")
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    doc = run(a.corpus, a.probes)
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, "refusal-response.json"), "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True, ensure_ascii=True)
        f.write("\n")
    with open(os.path.join(a.out, "refusal-response-tables.md"), "w", encoding="utf-8") as f:
        f.write(tables(doc))
    print(f"refusal responses: {len(doc['events'])} refusals classified")
    return 0


if __name__ == "__main__":
    sys.exit(main())
