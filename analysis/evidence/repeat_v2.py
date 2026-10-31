#!/usr/bin/env python3
"""Repeated refused proposals, version 2 (dreamteam.repeat-evidence/v2).

    python3 -I -B analysis/evidence/repeat_v2.py \
        --corpus docs/benchmark/v08a-dev/corpus.json \
        --v1 docs/benchmark/v08a-dev/evidence.json \
        [--probes <funding probe output dir>] --out <dir>

A separately versioned rule beside the frozen decision evidence
(dreamteam.decision-evidence/v1), which it neither changes nor replaces.
Version 1's REPEATED_INVALID_PROPOSAL recognizes a repeat by the refusal:
its type, code and message with dates normalized. A message that states an
amount differs from day to day, so v1 misses those repeats (defect V1-D1),
and two different actions refused with the same message are merged.
Version 2 recognizes a repeat by the action itself:

* **Identity.** Exact action identity: the action's canonical JSON without
  its requestId, the same identity as dreamteam.refusal-response/v1. The
  refusal's wording plays no part.
* **Occurrence.** Every action refused by validate (any code but
  batch_rejected), in any attempt, at any decision point.
* **Repeat.** The identical action proposed again at any later attempt of
  the recording:
  * refused again: `repeated_refused`;
  * accepted by validate: `repeated_accepted` (the refusal no longer
    applied);
  * neither (proposed but not validated): `repeated_unvalidated`.
* **No repeat.** No later attempt proposes it: `not_repeated`, only when
  every later attempt's actions are recorded. When some are not (the
  llm-agent 2 format keeps only the final reply's actions), it abstains:
  `unresolved`.
* **Missing evidence.** When the refused attempt's own actions are not
  recorded, it abstains: `insufficient_evidence`.
* **Similarity is kept apart.** A later action of the same type and target
  with other values (a modified proposal) is counted as `related`. It is
  never a repeat.

**Findings.** A finding (`supported`) is a group of two or more refused
occurrences of one identical action in a recording.

**Versus v1.** The output compares v2's repeat links (one occurrence to
the next refused occurrence of the same identity) with v1's (one
occurrence to the next refusal with the same key), and names why they
differ. The v1 findings are read from v1's own output and left as they
are.

It reads the llm-agent 1 and 2 formats, llm-agent 3's records
(dreamteam.llm-decision-record/v3) and the funding probes' attempt-level
evidence. No replay is needed, and no model is contacted.
"""

import argparse
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import decision_evidence as D  # noqa: E402  (the frozen analyzer: helpers only)
import record_v3 as V3  # noqa: E402
from refusal_response import identity, target  # noqa: E402

SCHEMA = "dreamteam.repeat-evidence/v2"
RULE = {"id": "REPEATED_REFUSED_PROPOSAL", "version": 2,
        "replaces": None, "beside": "REPEATED_INVALID_PROPOSAL version 1 (frozen)"}
STATUSES = ("repeated_refused", "repeated_accepted", "repeated_unvalidated", "not_repeated", "unresolved",
            "insufficient_evidence")


# --- attempts, in one form for every recording format ---------------------

def _results(actions, validation, results_of):
    """{action index: (accepted, code, message)} of a validated proposal."""
    out = {}
    if not actions or not isinstance(validation, list):
        return out
    for (batch, idx), v in zip(D.A.split(actions), validation):
        for k, r in enumerate(results_of(v)):
            if k < len(idx):
                out[idx[k]] = (bool(r.get("ok")), r.get("code"), r.get("message"))
    return out


def attempts_v3(rec, pi):
    out = []
    for at in rec.get("attempts") or []:
        acts = V3.proposal(at)
        if acts is None:
            continue
        out.append({"pi": pi, "point": rec["point"], "attempt": at["attempt"], "actions": acts,
                    "results": _results(acts, at.get("validation"), lambda v: (v.get("answer") or {}).get("results") or []),
                    "unknown": []})
    return out


def attempts_probe(arec, pi, point):
    out = []
    for at in arec["attempts"]:
        out.append({"pi": pi, "point": point, "attempt": at["attempt"], "actions": at["actions"],
                    "results": _results(at["actions"], at.get("validation"), lambda v: v.get("results") or []),
                    "unknown": []})
    return out


def parsed_attempt(rec, n):
    """The attempt whose reply is the llm-agent 2 record's `proposed`: the
    last attempt, unless the point fell back, then the last attempt whose
    reply was parsed and validated (its error is a refusal or a validate
    protocol error)."""
    if rec.get("outcome") != "fallback":
        return n
    k = None
    for e in rec.get("errors") or []:
        s = str(e)
        if s.startswith("attempt ") and (": refused: " in s or ": invalid_action: validate answered" in s):
            k = int(s.split(":", 1)[0].split()[1])
    return k


def attempts_v2(rec, pi):
    """The llm-agent 1 and 2 formats: only the parsed reply's actions are
    recorded; every other attempt is there with its refusals (type, code,
    message), and its actions unknown."""
    rs = D.refusals_of(rec)
    n = max([rec.get("attempts") or 1] + [r.get("attempt") or 0 for r in rs or []])
    kp = parsed_attempt(rec, n)
    proposed = rec.get("proposed") if isinstance(rec.get("proposed"), list) else None
    out = []
    for k in range(1, n + 1):
        known = k == kp and proposed is not None
        res = _results(proposed, rec.get("validation"), lambda v: v.get("results") or []) if known else {}
        unknown = [] if known else [(r.get("action"), r.get("type"), r.get("code"), r.get("message"))
                                    for r in rs or [] if r.get("attempt") == k and r.get("code") != "batch_rejected"]
        if known:
            # Refusals of the parsed reply that its validation does not show
            # (the validation of an earlier attempt): payload unknown.
            for r in rs or []:
                if r.get("attempt") == k and r.get("code") != "batch_rejected" and \
                        res.get(r.get("action"), (True,))[0] is not False:
                    unknown.append((r.get("action"), r.get("type"), r.get("code"), r.get("message")))
        out.append({"pi": pi, "point": rec["point"], "attempt": k, "actions": proposed if known else None,
                    "results": res, "unknown": unknown})
    return out


def sequence(recs, arecs=None):
    """Every attempt of a recording, in order."""
    out = []
    for pi, rec in enumerate(recs):
        if V3.is_v3(rec):
            out += attempts_v3(rec, pi)
        elif arecs is not None:
            out += attempts_probe(arecs[pi], pi, rec["point"])
        else:
            out += attempts_v2(rec, pi)
    return out


# --- the rule ------------------------------------------------------------

def norm(msg):
    return D.DATE.sub("<date>", str(msg or ""))


def occurrences(seq):
    """Every refused action (not batch_rejected): (attempt index, action
    index, type, code, message, payload or None)."""
    out = []
    for i, e in enumerate(seq):
        for k in sorted(e["results"]):
            ok, code, msg = e["results"][k]
            if not ok and code != "batch_rejected":
                a = e["actions"][k]
                out.append((i, k, a.get("type"), code, msg, a))
        for k, typ, code, msg in e["unknown"]:
            out.append((i, k, typ, code, msg, None))
    return out


def analyze_recording(seq, source):
    occ = occurrences(seq)
    events, groups = [], {}
    for i, k, typ, code, msg, payload in occ:
        e = seq[i]
        ev = {"point": e["point"], "attempt": e["attempt"], "action": k, "type": typ, "code": code,
              "message": msg, "refused": D.short_action(payload) if payload else None,
              "identity": hashlib.sha256(identity(payload).encode("utf-8")).hexdigest()[:16] if payload else None,
              "status": None,
              "repeat": None, "related": 0, "laterUnrecorded": 0}
        later = seq[i + 1:]
        ev["laterUnrecorded"] = sum(1 for x in later if x["actions"] is None)
        if payload is None:
            ev["status"] = "insufficient_evidence"
        else:
            rid, rt = identity(payload), target(payload)
            hit = None
            for x in later:
                if x["actions"] is None:
                    continue
                for j, a in enumerate(x["actions"]):
                    if identity(a) == rid and hit is None:
                        hit = (x, j)
                    elif identity(a) != rid and target(a) == rt:
                        ev["related"] += 1
            if hit:
                x, j = hit
                r = x["results"].get(j)
                ev["status"] = ("repeated_unvalidated" if r is None else
                                "repeated_accepted" if r[0] else "repeated_refused")
                ev["repeat"] = {"point": x["point"], "attempt": x["attempt"], "action": j,
                                "samePoint": x["pi"] == e["pi"], "code": None if r is None else r[1],
                                "sameMessage": None if r is None or r[0] else norm(r[2]) == norm(msg)}
            else:
                ev["status"] = "unresolved" if ev["laterUnrecorded"] else "not_repeated"
            groups.setdefault(rid, []).append(len(events))
        key = json.dumps([source["sha256"], e["point"], e["attempt"], k, code], sort_keys=True)
        ev["id"] = "rp2-" + hashlib.sha256(key.encode("utf-8")).hexdigest()[:16]
        events.append(ev)
    findings = []
    for rid, idxs in groups.items():
        if len(idxs) < 2:
            continue
        evs = [events[j] for j in idxs]
        within = sum(1 for a, b in zip(evs, evs[1:]) if a["point"] == b["point"])
        messages = sorted({norm(x["message"]) for x in evs})
        key = json.dumps([source["sha256"], rid], sort_keys=True)
        findings.append({"id": "rf2-" + hashlib.sha256(key.encode("utf-8")).hexdigest()[:16],
                         "rule": RULE["id"], "version": RULE["version"], "status": "supported",
                         "recording": source, "action": evs[0]["refused"], "type": evs[0]["type"],
                         "count": len(evs), "withinPoint": within, "acrossPoints": len(evs) - 1 - within,
                         "codes": sorted({x["code"] for x in evs}), "distinctMessages": len(messages),
                         "occurrences": [{"point": x["point"], "attempt": x["attempt"], "action": x["action"],
                                          "code": x["code"]} for x in evs],
                         "basis": f"The identical action ({evs[0]['type']}) was refused {len(evs)} times: {within} "
                                  f"time(s) again within a decision point's retries, {len(evs) - 1 - within} time(s) "
                                  f"at a later decision point, with {len(messages)} distinct refusal message(s)."})
    findings.sort(key=lambda f: (f["occurrences"][0]["point"], f["occurrences"][0]["attempt"], f["id"]))
    for ev in events:
        ev.update({"recording": source["path"], "group": source["group"], "model": source["model"]})
    return events, findings


# --- versus v1 -----------------------------------------------------------

def links_v1(findings, path):
    """v1's repeat links in one recording: consecutive occurrences of each
    supported REPEATED_INVALID_PROPOSAL finding, as (point, attempt, type,
    code) pairs."""
    out = []
    for f in findings:
        if f["rule"]["id"] != "REPEATED_INVALID_PROPOSAL" or f["status"] != "supported" or f["recording"]["path"] != path:
            continue
        r = f["repeat"]
        occ = [(o["point"], o["attempt"], r["type"], r["code"]) for o in r["occurrences"]]
        out += list(zip(occ, occ[1:]))
    return out


def links_v2(events):
    """v2's repeat links: each refused occurrence to its next refused
    occurrence of the identical action."""
    by = {}
    for e in events:
        if e["identity"] is not None:
            by.setdefault(e["identity"], []).append(e)
    out = []
    for evs in by.values():
        occ = [(e["point"], e["attempt"], e["type"], e["code"]) for e in evs]
        out += list(zip(occ, occ[1:]))
    return out


def compare(v1_links, events):
    """The links of both versions, and why they differ: a v1 link v2 does
    not make because it abstains (an end's actions are not recorded) or
    because the two refused actions differ; a v2 link v1 does not make
    because the refusal messages differ, or because the codes do."""
    v2 = links_v2(events)
    known = {(e["point"], e["attempt"], e["type"], e["code"]): e["identity"] is not None for e in events}
    msg = {}
    for e in events:
        msg.setdefault((e["point"], e["attempt"], e["type"], e["code"]), set()).add(norm(e["message"]))
    s1, s2 = set(v1_links), set(v2)
    out = {"v1Links": len(s1), "v2Links": len(s2), "both": len(s1 & s2), "v1OnlyV2Abstains": 0,
           "v1OnlyDifferentActions": 0, "v2OnlyMessageDiffers": 0, "v2OnlyOther": 0, "examples": []}
    for a, b in sorted(s1 - s2):
        why = "v1OnlyV2Abstains" if not (known.get(a) and known.get(b)) else "v1OnlyDifferentActions"
        out[why] += 1
        if len(out["examples"]) < 6:
            out["examples"].append({"kind": why, "from": list(a), "to": list(b)})
    for a, b in sorted(s2 - s1):
        why = "v2OnlyMessageDiffers" if msg.get(a) != msg.get(b) else "v2OnlyOther"
        out[why] += 1
        if len([x for x in out["examples"] if x["kind"].startswith("v2")]) < 6:
            out["examples"].append({"kind": why, "from": list(a), "to": list(b)})
    return out


# --- inputs --------------------------------------------------------------

def jsonl(path):
    with open(path, "rb") as f:
        raw = f.read()
    return raw, [json.loads(l) for l in raw.decode("utf-8").splitlines() if l.strip()]


def sources_of(corpus_path, probes_dir=None):
    out = []
    with open(corpus_path, encoding="utf-8") as f:
        corpus = json.load(f)
    for g in corpus["groups"]:
        for rp in g["replays"]:
            with open(rp["config"], encoding="utf-8") as f:
                recmap = D.replay_recordings(json.load(f))
            for pid in sorted(recmap):
                for path in recmap[pid]:
                    out.append((g["id"], path, None))
    if probes_dir:
        for name in sorted(os.listdir(probes_dir)):
            if name.endswith(".decisions.jsonl"):
                stem = name[:-len(".decisions.jsonl")]
                out.append(("probes", os.path.join(probes_dir, name), os.path.join(probes_dir, stem + ".attempts.jsonl")))
                out.append(("probes-v2-format", os.path.join(probes_dir, name), None))
    return out


def run(corpus_path, v1_path=None, probes_dir=None, v1_findings=None):
    """v1_findings: the frozen rules' findings given directly instead of
    read from v1_path (V0.8c reads them from its own run of the rules)."""
    v1 = v1_findings or []
    if v1_path:
        with open(v1_path, encoding="utf-8") as f:
            v1 = json.load(f)["findings"]
    events, findings, inputs, comparison = [], [], [], {}
    for group, path, apath in sources_of(corpus_path, probes_dir):
        raw, recs = jsonl(path)
        arecs = jsonl(apath)[1] if apath and os.path.exists(apath) else None
        probe = group.startswith("probes")
        name = path if not probe else "probes/" + os.path.basename(path)
        src = {"path": name, "sha256": hashlib.sha256(raw).hexdigest(), "group": group,
               "model": D.model_of(path, D.Inputs()) if not probe else "scripted probe"}
        if not any(i["path"] == name for i in inputs):
            inputs.append({"path": name, "sha256": src["sha256"]})
        evs, fs = analyze_recording(sequence(recs, arecs), src)
        events += evs
        findings += fs
        if (v1_path or v1_findings is not None) and not probe:
            c = compare(links_v1(v1, path), evs)
            k = f"{group}|{src['model']}"
            s = comparison.setdefault(k, {"v1Links": 0, "v2Links": 0, "both": 0, "v1OnlyV2Abstains": 0,
                                          "v1OnlyDifferentActions": 0, "v2OnlyMessageDiffers": 0, "v2OnlyOther": 0,
                                          "examples": []})
            for f_ in s:
                if f_ == "examples":
                    s[f_] += [dict(x, recording=path) for x in c[f_]][:max(0, 6 - len(s[f_]))]
                else:
                    s[f_] += c[f_]
    summary = {}
    for e in events:
        k = f"{e['group']}|{e['model']}"
        s = summary.setdefault(k, {"refusals": 0, "insufficientFunds": 0, **{x: 0 for x in STATUSES},
                                   "related": 0, "findings": 0})
        s["refusals"] += 1
        s["insufficientFunds"] += e["code"] == "insufficient_funds"
        s[e["status"]] += 1
        s["related"] += e["related"] > 0
    for f in findings:
        summary[f"{f['recording']['group']}|{f['recording']['model']}"]["findings"] += 1
    return {"schema": SCHEMA, "rule": RULE, "statuses": list(STATUSES),
            "identity": "canonical JSON of the action without requestId",
            "inputs": inputs, "v1": v1_path, "summary": {k: summary[k] for k in sorted(summary)},
            "versusV1": {k: comparison[k] for k in sorted(comparison)}, "findings": findings, "events": events}


def tables(doc):
    L = ["# Repeated refused proposals, version 2", "",
         f"`{doc['schema']}`, rule `{doc['rule']['id']}` version 2, beside the frozen v1 rule. A repeat is the "
         "identical action (canonical JSON without requestId) proposed again after a refusal; the refusal's wording "
         "plays no part.", "",
         "| Group | Model | Refusals | insufficient_funds | " + " | ".join(doc["statuses"])
         + " | with a related (modified) proposal | findings |",
         "|---|---|---|---|" + "---|" * len(doc["statuses"]) + "---|---|"]
    for k, s in doc["summary"].items():
        g, m = k.split("|")
        L.append(f"| {g} | {m} | {s['refusals']} | {s['insufficientFunds']} | "
                 + " | ".join(str(s[c]) for c in doc["statuses"]) + f" | {s['related']} | {s['findings']} |")
    if doc["versusV1"]:
        L += ["", "## Versus v1 (repeat links: one refused occurrence to the next of the same proposal)", "",
              "| Group | Model | v1 links | v2 links | both | v1 only: v2 abstains | v1 only: different actions | "
              "v2 only: message differs | v2 only: other |", "|---|---|---|---|---|---|---|---|---|"]
        for k, s in doc["versusV1"].items():
            g, m = k.split("|")
            L.append(f"| {g} | {m} | {s['v1Links']} | {s['v2Links']} | {s['both']} | {s['v1OnlyV2Abstains']} | "
                     f"{s['v1OnlyDifferentActions']} | {s['v2OnlyMessageDiffers']} | {s['v2OnlyOther']} |")
    return "\n".join(L) + "\n"


def main(argv=None):
    ap = argparse.ArgumentParser(description="Repeated refused proposals, version 2 (read-only; no model).")
    ap.add_argument("--corpus", required=True)
    ap.add_argument("--v1", help="the frozen v1 evidence of the same corpus, to compare with")
    ap.add_argument("--probes", help="a funding probe output directory")
    ap.add_argument("--out", required=True)
    ap.add_argument("--name", default="repeat-v2")
    a = ap.parse_args(argv)
    doc = run(a.corpus, a.v1, a.probes)
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, a.name + ".json"), "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True, ensure_ascii=True)
        f.write("\n")
    with open(os.path.join(a.out, a.name + "-tables.md"), "w", encoding="utf-8") as f:
        f.write(tables(doc))
    print(f"repeat v2: {len(doc['events'])} refusals, {len(doc['findings'])} findings")
    return 0


if __name__ == "__main__":
    sys.exit(main())
