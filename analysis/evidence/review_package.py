#!/usr/bin/env python3
"""The independent review package of the frozen decision evidence
(dreamteam.evidence-review-sample/v1).

    python3 -I -B analysis/evidence/review_package.py \
        --corpus docs/benchmark/v08a-dev/corpus.json --replays <dir> --out <dir>

It runs the frozen analyzer (decision_evidence.py, unchanged) with an
instrumented copy of its Analysis class that keeps, beside each finding,
each check the rules judged consistent, together with the evidence a
reviewer needs: the reason span, the proposed actions, the validation and
execution, the relevant part of the prompt the model saw, the rule's output
and the evidence completeness. The instrumentation only reads the rule
functions' arguments; the rules and their outputs are those of the freeze.

It then draws a deterministic, stratified sample (seed SEED): every cell of
(rule, status) gets a fixed number of items, spread over the strata
(group, model, prompt completeness) in turn, ordered within each stratum by
SHA-256 of the seed and the item's identifier, so the draw depends on no
random number generator. Negative controls are reasons that mention an
action but in which the rules found no explicit statement.

Writes manifest.json, items.json, items.md and form.csv. Never contacts a
model; reads its inputs only.
"""

import argparse
import csv
import hashlib
import io
import json
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import decision_evidence as D  # noqa: E402  (the frozen analyzer, used as is)

SCHEMA = "dreamteam.evidence-review-sample/v1"
SEED = "v08b-review-1"
# Items per (rule, status): every supported A and C finding (8 in all),
# and fixed numbers elsewhere. Cells with fewer items take them all.
ALLOCATION = {
    ("EXPLICIT_ACTION_OMITTED", "supported"): 5, ("EXPLICIT_ACTION_OMITTED", "unresolved"): 6,
    ("EXPLICIT_ACTION_OMITTED", "consistent"): 6,
    ("EXPLICIT_OPPOSITE_ACTION", "supported"): 5, ("EXPLICIT_OPPOSITE_ACTION", "unresolved"): 3,
    ("EXPLICIT_OPPOSITE_ACTION", "consistent"): 4,
    ("FINANCIAL_AMOUNT_MISMATCH", "supported"): 3, ("FINANCIAL_AMOUNT_MISMATCH", "unresolved"): 6,
    ("FINANCIAL_AMOUNT_MISMATCH", "consistent"): 6,
    ("REPEATED_INVALID_PROPOSAL", "supported"): 10, ("REPEATED_INVALID_PROPOSAL", "consistent"): 4,
    ("STATED_ACTION_NOT_EXECUTED", "supported"): 10, ("STATED_ACTION_NOT_EXECUTED", "consistent"): 6,
    ("FACTUAL_OBSERVATION_MISMATCH", "supported"): 10, ("FACTUAL_OBSERVATION_MISMATCH", "unresolved"): 2,
    ("FACTUAL_OBSERVATION_MISMATCH", "insufficient_evidence"): 3,
    ("FACTUAL_OBSERVATION_MISMATCH", "consistent"): 6,
    ("NEGATIVE_CONTROL", "no_statement"): 8,
}
LABELS = ("supported", "unsupported", "ambiguous", "insufficient_evidence")
ACTION_WORDS = re.compile(r"\b(?:extend\w*|hir(?:e|ing)|terminat\w+|cancel\w*|teleradiology|allocation|roster|"
                          r"worklist policy|walk-in policy)\b", re.I)


def order_key(item_id):
    return hashlib.sha256(f"{SEED}|{item_id}".encode("utf-8")).hexdigest()


def excerpt(prompt, needles, width=180):
    """The prompt text around the first needle found: what the model read."""
    if not prompt:
        return None
    for n in needles:
        if not n:
            continue
        i = prompt.find(n)
        if i >= 0:
            s, e = max(0, i - width), min(len(prompt), i + len(n) + width)
            return ("…" if s else "") + prompt[s:e].replace("\n", " ⏎ ") + ("…" if e < len(prompt) else "")
    return None


def amount_needles(vals):
    out = []
    for p, v in vals or []:
        key = re.sub(r"\[.*?\]", "", p.rsplit(".", 1)[-1])
        out += [f'"{key}":{v}', D.A.money(v), str(v)]
    return out


def fact_needles(f, v):
    k = f["kind"]
    if k == "unread":
        return [f"Reading backlog: {v} unread", '"worklist":{"policy":']
    if k == "overdue":
        return [f" {v} overdue,", "Reading backlog:", '"worklist":{"policy":']
    if k == "overdue_priority":
        return [f"{v} {f.get('priority')} reports overdue", f"{f.get('priority')} reports overdue",
                '"worklist":{"policy":']
    if k == "lost_total":
        return [f"{v} since the start.", "Lost demand:"]
    if k in ("weekday", "closed"):
        return ["Decision point "]
    return []


class Instrumented(D.Analysis):
    """The frozen Analysis, keeping each consistent check and each finding
    with the evidence around it. It reads the calling rule function's
    arguments; it changes nothing."""

    def __init__(self):
        super().__init__()
        self.consistent_items = []
        self.finding_context = {}

    def consistent(self, rule):
        super().consistent(rule)
        loc = sys._getframe(1).f_locals
        ctx = loc.get("ctx")
        rec = loc.get("rec")
        if rule == "REPEATED_INVALID_PROPOSAL":
            occ = loc.get("occ") or []
            rec = loc["recs"][occ[0][0]] if occ else None
        if ctx is None:
            return
        item = {"rule": rule, "status": "consistent", "recording": ctx["recording"], "run": ctx["run"],
                "point": (rec or {}).get("point"), "time": (rec or {}).get("time"),
                "completeness": ctx["completeness"](rec) if rec else None}
        reason = loc.get("reason")
        item.update(self._context(rule, loc, reason))
        key = json.dumps([ctx["recording"]["sha256"], item["point"], rule, item.get("check")], sort_keys=True)
        item["id"] = "rv1-" + hashlib.sha256(key.encode("utf-8")).hexdigest()[:16]
        self.consistent_items.append(item)

    def add(self, ctx, rule, status, basis, rec=None, span=None, reason=None, actions=None, exec_=None,
            observation=None, extra=None, sub="", key=None):
        super().add(ctx, rule, status, basis, rec, span, reason, actions, exec_, observation, extra, sub, key)
        loc = dict(sys._getframe(1).f_locals)
        if "ctx" not in loc:  # called through check_statement's add lambda
            loc = dict(sys._getframe(2).f_locals)
        self.finding_context[self.findings[-1]["id"]] = self._context(rule, loc, reason)

    def _context(self, rule, loc, reason):
        out = {"reason": reason}
        prompt = loc.get("prompt")
        if rule in ("EXPLICIT_ACTION_OMITTED", "EXPLICIT_OPPOSITE_ACTION", "STATED_ACTION_NOT_EXECUTED"):
            st = loc.get("st")
            proposed = loc.get("proposed") or []
            ex = loc.get("ex") or []
            if st:
                out["span"] = st["span"]
                out["check"] = {"kind": st["kind"], "type": st["type"], "targets": st["targets"],
                                "params": st["params"], "flags": st["flags"]}
            out["proposed"] = [D.short_action(a) for a in proposed]
            # The public observation these rules use: who each employee ID is.
            ids = set((st or {}).get("targets") or []) | {a.get("employee") for a in proposed if a.get("employee")}
            staff = {x.get("id"): x.get("name") for x in ((loc.get("obs") or {}).get("staff") or [])
                     if isinstance(x, dict)}
            out["observedStaff"] = {i: staff.get(i) for i in sorted(ids) if i}
            out["execution"] = [{k: x[k] for k in ("index", "validate", "submitted", "apply", "state")} for x in ex]
        elif rule == "FINANCIAL_AMOUNT_MISMATCH":
            a = loc.get("a") or {}
            out["span"] = a.get("span")
            vals = loc.get("vals") or []
            out["check"] = {"amount": a.get("text"), "quantity": loc.get("q"),
                            "observed": [{"path": p, "cents": v} for p, v in vals]}
            out["promptExcerpt"] = excerpt(prompt, amount_needles(vals))
        elif rule == "FACTUAL_OBSERVATION_MISMATCH":
            f = loc.get("f") or {}
            out["span"] = f.get("span")
            out["check"] = {"kind": f.get("kind"), "stated": f.get("value"), "observedPath": loc.get("path"),
                            "observed": loc.get("v")}
            out["promptExcerpt"] = excerpt(prompt, fact_needles(f, loc.get("v")))
        elif rule == "REPEATED_INVALID_PROPOSAL":
            key, occ = loc.get("key"), loc.get("occ") or []
            out["check"] = {"type": key[0] if key else None, "code": key[1] if key else None,
                            "message": key[2] if key else None,
                            "occurrences": [{"point": p, "attempt": at} for _, p, at, _ in occ]}
        return out


def negative_controls(corpus_path):
    """Reasons that mention an action and in which the rules found no
    explicit statement: (recording, point, reason)."""
    with open(corpus_path, encoding="utf-8") as f:
        corpus = json.load(f)
    out = []
    for g in corpus["groups"]:
        for rp in g["replays"]:
            with open(rp["config"], encoding="utf-8") as f:
                recmap = D.replay_recordings(json.load(f))
            for pid in sorted(recmap):
                for path in recmap[pid]:
                    with open(path, encoding="utf-8") as f:
                        for line in f:
                            r = json.loads(line)
                            reason = r.get("reason")
                            if not reason or not ACTION_WORDS.search(reason):
                                continue
                            st, _ = D.extract(reason, {})
                            if not st:
                                out.append((g["id"], path, r))
    return out


def stratum(item):
    c = item.get("completeness") or {}
    return f"{item['recording']['group']}|{item['run'].get('model') or '?'}|{c.get('prompt')}"


def draw(pool, n):
    """n items spread over the pool's strata in turn, each stratum in its
    seeded order."""
    by = {}
    for it in pool:
        by.setdefault(stratum(it), []).append(it)
    for k in by:
        by[k].sort(key=lambda it: order_key(it["id"]))
    keys = sorted(by, key=order_key)
    out, i = [], 0
    while len(out) < n and any(by.values()):
        k = keys[i % len(keys)]
        if by[k]:
            out.append(by[k].pop(0))
        i += 1
    return out


def run(corpus, replays):
    """The frozen analysis with instrumentation: (doc, pool of items)."""
    holder = {}

    class Keep(Instrumented):
        def __init__(self):
            super().__init__()
            holder["an"] = self

    orig = D.Analysis
    D.Analysis = Keep
    try:
        doc, _, _ = D.analyze(corpus, replays)
    finally:
        D.Analysis = orig
    an = holder["an"]
    pool = []
    for f in doc["findings"]:
        ctxt = an.finding_context.get(f["id"], {})
        pool.append({"id": f["id"], "rule": f["category"], "status": f["status"], "recording": f["recording"],
                     "run": f["run"], "point": (f["decision"] or {}).get("point"),
                     "time": (f["decision"] or {}).get("time"), "completeness": f["completeness"],
                     "ruleOutput": {"status": f["status"], "subrule": f["rule"]["subrule"], "basis": f["basis"],
                                    "observation": f.get("observation"), "repeat": f.get("repeat")},
                     "sources": f["sources"], "reasonSpan": f["reasonSpan"], "actions": f["actions"],
                     "execution": f["execution"], **{k: v for k, v in ctxt.items() if k not in ("check",)},
                     "check": ctxt.get("check")})
    for c in an.consistent_items:
        c["ruleOutput"] = {"status": "consistent", "basis": "The rule checked this and found the statement and the "
                                                            "evidence in agreement."}
        sp = c.get("span")
        c["reasonSpan"] = None if not sp or not c.get("reason") else {
            "start": sp[0], "end": sp[1], "text": c["reason"][sp[0]:sp[1]]}
        pool.append(c)
    return doc, pool


def sample(doc, pool, controls):
    items = []
    for (rule, status), n in ALLOCATION.items():
        if rule == "NEGATIVE_CONTROL":
            continue
        cell = [it for it in pool if it["rule"] == rule and it["status"] == status]
        for it in draw(cell, n):
            items.append(dict(it, cell=f"{rule}/{status}"))
    cand = []
    for group, path, r in controls:
        iid = "nc1-" + hashlib.sha256(f"{path}|{r['point']}".encode("utf-8")).hexdigest()[:16]
        cand.append({"id": iid, "rule": "NEGATIVE_CONTROL", "status": "no_statement",
                     "recording": {"path": path, "group": group}, "run": {"model": D.model_of(path, D.Inputs())},
                     "point": r["point"], "time": r.get("time"), "completeness": {"prompt": None},
                     "reason": r.get("reason"), "proposed": [D.short_action(a) for a in r.get("proposed") or []],
                     "ruleOutput": {"status": "no_statement", "basis": "The reason mentions an action, but the rules "
                                                                       "found no explicit statement of intention in it."}})
    for it in draw(cand, ALLOCATION[("NEGATIVE_CONTROL", "no_statement")]):
        items.append(dict(it, cell="NEGATIVE_CONTROL/no_statement"))
    items.sort(key=lambda it: order_key(it["id"]))
    for k, it in enumerate(items):
        it["item"] = f"R{k + 1:03d}"
    return items


ITEM_FIELDS = ("item", "id", "cell", "rule", "status", "recording", "run", "point", "time", "reason", "reasonSpan",
               "proposed", "actions", "execution", "check", "observedStaff", "promptExcerpt", "ruleOutput", "sources",
               "completeness")


def item_md(it):
    L = [f"### {it['item']} — {it['rule']} · rule output: {it['status']}", ""]
    L.append(f"* Identifier: `{it['id']}`")
    rec = it["recording"]
    L.append(f"* Recording: `{rec['path']}`, decision point {it.get('point')} ({it.get('time') or ''}); "
             f"model {it['run'].get('model') or 'see recording'}")
    if it.get("reason"):
        L.append(f"* Reason (whole): {it['reason']}")
    rs = it.get("reasonSpan")
    if rs:
        L.append(f"* Relevant span: «{rs['text']}»")
    if it.get("proposed") is not None:
        L.append(f"* Proposed actions: `{json.dumps(it.get('proposed'), sort_keys=True)}`")
    if it.get("execution"):
        L.append(f"* Validation and execution: `{json.dumps(it['execution'], sort_keys=True)}`")
    if it.get("check"):
        L.append(f"* What the rule checked: `{json.dumps(it['check'], sort_keys=True)}`")
    if it.get("observedStaff"):
        L.append(f"* Staff in the observation: `{json.dumps(it['observedStaff'], sort_keys=True)}`")
    if it.get("promptExcerpt"):
        L.append(f"* Prompt excerpt (what the model was shown): `{it['promptExcerpt']}`")
    ro = it["ruleOutput"]
    L.append(f"* Rule output: **{ro['status']}**" + (f" ({ro.get('subrule')})" if ro.get("subrule") else "")
             + f". {ro['basis']}")
    if it.get("completeness"):
        L.append(f"* Evidence completeness: `{json.dumps(it['completeness'], sort_keys=True)}`")
    L.append("")
    return "\n".join(L)


def write(out, doc, items):
    os.makedirs(out, exist_ok=True)
    clean = [{k: it.get(k) for k in ITEM_FIELDS} for it in items]
    items_text = json.dumps(clean, indent=1, sort_keys=True, ensure_ascii=True) + "\n"
    with open(os.path.join(out, "items.json"), "w", encoding="utf-8") as f:
        f.write(items_text)
    cells = {}
    for it in items:
        cells[it["cell"]] = cells.get(it["cell"], 0) + 1
    strata = {}
    for it in items:
        k = f"{it['recording']['group']}|{it['run'].get('model') or '-'}"
        strata[k] = strata.get(k, 0) + 1
    manifest = {
        "schema": SCHEMA, "seed": SEED, "status": "PENDING",
        "statusNote": "No independent reviewer has completed this review. The package is frozen; no label exists.",
        "frozenEvidence": {"contract": doc["schema"], "analyzer": doc["analyzer"],
                           "rules": doc["rules"]},
        "allocation": [{"rule": r, "status": s, "items": n} for (r, s), n in ALLOCATION.items()],
        "cells": {k: cells[k] for k in sorted(cells)},
        "strata": {k: strata[k] for k in sorted(strata)},
        "items": [{"item": it["item"], "id": it["id"], "cell": it["cell"]} for it in items],
        "itemsSha256": hashlib.sha256(items_text.encode("utf-8")).hexdigest(),
        "labels": list(LABELS),
    }
    with open(os.path.join(out, "manifest.json"), "w", encoding="utf-8") as f:
        json.dump(manifest, f, indent=1, sort_keys=True, ensure_ascii=True)
        f.write("\n")
    md = ["# Decision evidence review items", "",
          f"Sample `{SEED}` of the frozen `dreamteam.decision-evidence/v1` outputs: {len(items)} items, in a "
          "seeded order that mixes rules and outputs. Review each item on its own evidence; label it in "
          "form.csv (see PROTOCOL.md). The label is about this rule output, not about the model.", ""]
    md += [item_md(it) for it in items]
    with open(os.path.join(out, "items.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(md))
    buf = io.StringIO()
    w = csv.writer(buf, lineterminator="\n")
    w.writerow(["item", "id", "rule", "rule_output", "label", "comment", "correction"])
    for it in items:
        w.writerow([it["item"], it["id"], it["rule"], it["status"], "", "", ""])
    with open(os.path.join(out, "form.csv"), "w", encoding="utf-8", newline="") as f:
        f.write(buf.getvalue())
    return manifest


def main(argv=None):
    ap = argparse.ArgumentParser(description="Build the seeded review package of the frozen decision evidence.")
    ap.add_argument("--corpus", required=True)
    ap.add_argument("--replays", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    doc, pool = run(a.corpus, a.replays.rstrip("/"))
    items = sample(doc, pool, negative_controls(a.corpus))
    m = write(a.out, doc, items)
    print(f"review sample {SEED}: {len(m['items'])} items, status {m['status']}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
