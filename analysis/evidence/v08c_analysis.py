#!/usr/bin/env python3
"""The V0.8c evaluation's analysis (dreamteam.v08c-analysis/v1;
docs/benchmark/v08c-dev/PLAN.md §7 and §8).

    python3 -I -B analysis/evidence/v08c_analysis.py \
        --corpus docs/benchmark/v08c-dev/corpus.json --contract docs/benchmark/v08c-dev/eval/contract.json \
        --replays <dir> --out <dir>

It reads the committed llm-agent 3 recordings, the committed contract
report and the corpus's replays (every run of the evaluation, the
deterministic comparators included, replayed without any model). It
writes, each kept apart and versioned:
* **Recording completeness** of every LLM run (dreamteam.llm-decision
  -record/v3's own completeness, and transport failures).
* **The primary outcomes** of the plan (§7), per run, with their
  denominators, and each policy paired with no action on the same entry
  and seed.
* **Refusal responses**, by dreamteam.refusal-response/v2.
* **Repeated refused proposals**, by repeat rule version 2, with the frozen
  rules' REPEATED_INVALID_PROPOSAL (dreamteam.decision-evidence/v1) beside
  it.
* **The frozen rules (v1)** on the llm-agent 2 view of each record
  (record_v3.v2_view): the analyzer runs unchanged on view files it writes
  under the replays directory.
No score, ranking, cause or counterfactual is computed. No model is
contacted.
"""

import argparse
import json
import os
import shutil
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import decision_evidence as D  # noqa: E402  (the frozen analyzer, unchanged)
import record_v3 as V3  # noqa: E402
import refusal_response as R  # noqa: E402
import repeat_v2 as P  # noqa: E402

SCHEMA = "dreamteam.v08c-analysis/v1"
VIEW = "v08c-v1-view"
METRICS = [("financial", "operatingProfitCents"), ("financial", "netRevenueCents"),
           ("financial", "operatingExpenseCents"), ("sustainability", "minCashCents"),
           ("sustainability", "shortfallCents"), ("sustainability", "daysBelowZero"),
           ("serviceReliability", "tatBreaches"), ("operational", "examsCompleted"),
           ("staffingCapacity", "ordersLost"), ("operational", "tatP90Sec"), ("staffingCapacity", "overtimeSec")]


def load(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def jsonl(path):
    with open(path, encoding="utf-8") as f:
        return [json.loads(l) for l in f if l.strip()]


def metric(dims, dim, name, key="metrics"):
    for d in dims or []:
        if d.get("dimension") == dim:
            for m in d.get(key) or []:
                if m.get("name") == name:
                    return m.get("value")
    return None


# --- inputs --------------------------------------------------------------

def replay_dirs(corpus, replays):
    """run id -> replay run directory."""
    out = {}
    for g in corpus["groups"]:
        for rp in g["replays"]:
            d = os.path.join(replays, rp["out"], "runs")
            if os.path.isdir(d):
                for rid in sorted(os.listdir(d)):
                    out.setdefault(rid, os.path.join(d, rid))
    return out


def recordings(corpus):
    """(policy, recording basename) -> path."""
    out = {}
    for g in corpus["groups"]:
        for rp in g["replays"]:
            for pid, paths in D.replay_recordings(load(rp["config"])).items():
                for p in paths:
                    out[(pid, os.path.basename(p))] = p
    return out


# --- completeness and the primary outcomes -------------------------------

def completeness(recs):
    c = {"points": len(recs), "attemptsRecorded": 0, "inputsVerifiable": 0, "attempts": 0, "outputsRecorded": 0,
         "reasonsRecorded": 0, "actionsRecorded": 0, "truncatedFields": 0, "stopped": {}, "transportFailures": 0,
         "infrastructureAffectedPoints": 0, "attemptsPerPoint": {}}
    for r in recs:
        k = r.get("completeness") or {}
        c["attemptsRecorded"] += bool(k.get("attemptsRecorded"))
        c["inputsVerifiable"] += bool(k.get("inputsVerifiable"))
        for f in ("attempts", "outputsRecorded", "reasonsRecorded", "actionsRecorded"):
            c[f] += k.get(f) or 0
        c["truncatedFields"] += len(k.get("truncated") or [])
        if k.get("stopped"):
            c["stopped"][k["stopped"]] = c["stopped"].get(k["stopped"], 0) + 1
        atts = r.get("attempts") or []
        bad = [a for a in atts if a.get("transport") not in ("ok", None)]
        c["transportFailures"] += len(bad)
        c["infrastructureAffectedPoints"] += bool(atts) and len(bad) == len(atts)
        n = str(len(atts))
        c["attemptsPerPoint"][n] = c["attemptsPerPoint"].get(n, 0) + 1
    c["attemptsPerPoint"] = {k: c["attemptsPerPoint"][k] for k in sorted(c["attemptsPerPoint"], key=int)}
    return c


def proposals(recs):
    """Actions proposed in every parsed attempt, validate refusals by code,
    and the stated commitments (frozen statement reader) of the final
    reason."""
    proposed, refusals, stated, submitted = 0, {}, {}, 0
    for r in recs:
        for a in r.get("attempts") or []:
            acts = V3.proposal(a)
            proposed += len(acts or [])
            for x in a.get("refusals") or []:
                refusals[x["code"]] = refusals.get(x["code"], 0) + 1
        st, _ = D.extract(r.get("reason") or "", {})
        for s in st:
            if s["kind"] == "commit" and s["type"]:
                stated[s["type"]] = stated.get(s["type"], 0) + 1
        submitted += sum(len(b) for b in r.get("batches") or [])
    srt = lambda d: {k: d[k] for k in sorted(d)}
    return {"proposedActions": proposed, "validateRefusals": srt(refusals),
            "fundingRefusalsAtValidate": refusals.get("insufficient_funds", 0),
            "statedCommitments": srt(stated), "submittedActions": submitted}


def applied(run_dir):
    """From the replay: actions submitted, accepted at apply, executed (with
    a change), refused at apply by code; credit and interest."""
    out = {"submitted": 0, "accepted": 0, "executed": 0, "refusedAtApply": {}, "executedByType": {}}
    for p in load(os.path.join(run_dir, "decisions.json")) or []:
        for b in p.get("batches") or []:
            for r in b.get("results") or []:
                out["submitted"] += 1
                if r.get("ok"):
                    out["accepted"] += 1
                    if r.get("code") != "unchanged":
                        out["executed"] += 1
                        t = (r.get("action") or {}).get("type")
                        out["executedByType"][t] = out["executedByType"].get(t, 0) + 1
                else:
                    out["refusedAtApply"][r.get("code")] = out["refusedAtApply"].get(r.get("code"), 0) + 1
    res = load(os.path.join(run_dir, "result.json"))
    cal = res.get("calendar") or []
    w = res.get("window") or {}
    out["executedByType"] = {k: out["executedByType"][k] for k in sorted(out["executedByType"])}
    out["credit"] = {"maxDrawnCents": max([d.get("creditDrawnEndCents", 0) or 0 for d in cal] or [0]),
                     "daysDrawn": sum(1 for d in cal if (d.get("creditDrawnEndCents") or 0) > 0),
                     "interestCents": (w.get("money") or {}).get("interestCents") or 0}
    out["overtimeCostCents"] = ((w.get("ops") or {}).get("workforce") or {}).get("overtimeCostCents")
    return out


def run_rows(contract, rdirs, recmap, rr_events, rp_events):
    base = {(r["entry"], r["seed"]): r for r in contract["runs"] if r["policy"]["id"] == contract["baseline"]}
    rows = []
    for run in contract["runs"]:
        pid = run["policy"]["id"]
        b = base[(run["entry"], run["seed"])]
        row = {"runId": run["id"], "entry": run["entry"], "scenario": run["scenario"], "seed": run["seed"],
               "policy": pid, "status": run["status"], "days": run["days"]}
        m = {n: metric(run["dimensions"], d, n) for d, n in METRICS}
        mb = {n: metric(b["dimensions"], d, n) for d, n in METRICS}
        row["outcomes"] = m
        row["noAction"] = None if pid == contract["baseline"] else {
            n: (m[n] - mb[n]) if isinstance(m[n], int) and isinstance(mb[n], int) else None for n in m}
        rd = rdirs.get(run["id"])
        row["replayed"] = rd is not None
        if rd:
            row.update(applied(rd))
            rb = applied(rdirs[b["id"]]) if b["id"] in rdirs else None
            if rb and pid != contract["baseline"]:
                row["noAction"]["maxCreditDrawnCents"] = row["credit"]["maxDrawnCents"] - rb["credit"]["maxDrawnCents"]
                row["noAction"]["interestCents"] = row["credit"]["interestCents"] - rb["credit"]["interestCents"]
        rec = recmap.get((pid, f"{run['entry']}-{pid}-s{run['seed']}-decisions.jsonl"))
        if rec:
            recs = jsonl(rec)
            row["recording"] = rec
            row["completeness"] = completeness(recs)
            row.update(proposals(recs))
            evs = [e for e in rr_events if e["recording"] == rec]
            row["refusalResponses"] = {"all": cats(evs), "fundingRefusals": cats(
                [e for e in evs if e["code"] == "insufficient_funds"])}
            row["fundingResponse"] = ("no funding refusal encountered" if not row["refusalResponses"]["fundingRefusals"]["refusals"]
                                      else "funding refusal encountered")
            pe = [e for e in rp_events if e["recording"] == rec]
            row["repeatV2"] = {s: sum(1 for e in pe if e["status"] == s) for s in P.STATUSES}
        rows.append(row)
    return rows


def cats(evs):
    out = {"refusals": len(evs), **{c: 0 for c in R.CATEGORIES}}
    for e in evs:
        out[e["category"]] += 1
    out["refusedAndRevised"] = out["modified_proposal"] + out["different_proposal"]
    return out


# --- the frozen rules on the llm-agent 2 view ----------------------------

def v1_on_view(corpus_path, replays):
    """The frozen analyzer on view files: each v3 recording rewritten by
    record_v3.v2_view, with replay configurations naming them, under
    <replays>/v08c-v1-view/. Returns the analyzer's output."""
    corpus = load(corpus_path)
    vd = os.path.join(replays, VIEW)
    shutil.rmtree(vd, ignore_errors=True)
    os.makedirs(vd)
    vc = {"schema": corpus["schema"], "id": corpus["id"] + "-v1-view", "description": "llm-agent 2 view",
          "groups": []}
    for g in corpus["groups"]:
        vg = dict(g, replays=[])
        for rp in g["replays"]:
            cfg = load(rp["config"])
            for e in cfg.get("external") or []:
                cmd = list(e["command"])
                for i, x in enumerate(cmd):
                    if i and cmd[i - 1] == "--replay":
                        base = os.path.basename(x)
                        with open(os.path.join(vd, base), "w", encoding="utf-8") as f:
                            for r in jsonl(x):
                                f.write(json.dumps(V3.v2_view(r), separators=(",", ":")) + "\n")
                        usage = x[:-len("decisions.jsonl")] + "usage.json"
                        if os.path.exists(usage):
                            shutil.copyfile(usage, os.path.join(vd, base[:-len("decisions.jsonl")] + "usage.json"))
                        cmd[i] = os.path.join(vd, base)
                e["command"] = cmd
            p = os.path.join(vd, os.path.basename(rp["config"]))
            with open(p, "w", encoding="utf-8") as f:
                json.dump(cfg, f, indent=1)
            vg["replays"].append({"config": p, "out": rp["out"]})
        vc["groups"].append(vg)
    cp = os.path.join(vd, "corpus.json")
    with open(cp, "w", encoding="utf-8") as f:
        json.dump(vc, f, indent=1)
    return D.analyze(cp, replays)[0]


def v1_summary(ev, recmap):
    """What the frozen rules found, with each recording named by its
    committed path."""
    path = {os.path.basename(p): p for p in recmap.values()}
    out = {"byRule": ev.get("byRule"), "byStatus": ev.get("byStatus"),
           "coverage": [{k: c[k] for k in ("recordings", "points", "promptVerified", "promptMismatch", "reasonMissing",
                                           "observation", "replayMissing") if k in c} for c in ev.get("coverage") or []],
           "findings": []}
    for f in ev["findings"]:
        out["findings"].append({"id": f["id"], "rule": f["rule"]["id"], "subrule": f["rule"].get("subrule"),
                                "status": f["status"], "recording": path.get(os.path.basename(f["recording"]["path"])),
                                "point": (f.get("decision") or {}).get("point"), "basis": f.get("basis"),
                                "repeat": f.get("repeat")})
    return out


# --- the document --------------------------------------------------------

def analyze(corpus_path, contract_path, replays):
    replays = os.path.normpath(replays)
    corpus, contract = load(corpus_path), load(contract_path)
    rdirs, recmap = replay_dirs(corpus, replays), recordings(corpus)
    rr = R.run(corpus_path, None, version=2)
    ev1 = v1_on_view(corpus_path, replays)
    v1 = v1_summary(ev1, recmap)
    findings_for_compare = [dict(f, recording=dict(f["recording"], path=recmap_path(f, recmap)))
                            for f in ev1["findings"]]
    rp = P.run(corpus_path, None, None, v1_findings=findings_for_compare)
    rows = run_rows(contract, rdirs, recmap, rr["events"], rp["events"])
    live_match = []
    for g in corpus["groups"]:
        for r in g["replays"]:
            c = os.path.join(replays, r["out"], "contract.json")
            if os.path.exists(c):
                rc = {x["id"]: x["dimensions"] for x in load(c)["runs"]}
                lc = {x["id"]: x["dimensions"] for x in contract["runs"] if x["id"] in rc}
                live_match.append({"replay": r["out"], "runs": len(rc), "metricsEqualLive": rc == lc})
    return {"schema": SCHEMA, "plan": "docs/benchmark/v08c-dev/PLAN.md", "contract": contract_path,
            "suite": contract.get("suite"), "engine": contract.get("engine"), "days": contract.get("days"),
            "seeds": contract.get("seeds"), "baseline": contract["baseline"], "replayMatchesLive": live_match,
            "runs": rows, "refusalResponseV2": {"summary": rr["summary"], "events": rr["events"]},
            "repeatV2": {"summary": rp["summary"], "versusV1": rp["versusV1"], "findings": rp["findings"],
                         "events": rp["events"]},
            "decisionEvidenceV1": v1}


def recmap_path(f, recmap):
    base = os.path.basename(f["recording"]["path"])
    return next((p for p in recmap.values() if os.path.basename(p) == base), f["recording"]["path"])


def money(c):
    if not isinstance(c, int):
        return "—"
    return ("−" if c < 0 else "") + f"${(abs(c) + 50) // 100:,}"


def signed(c, f=money):
    if not isinstance(c, int):
        return "—"
    return ("+" if c > 0 else "") + f(c)


def num(v):
    return "—" if not isinstance(v, int) else f"{v:,}".replace("-", "−")


def snum(v):
    return "—" if not isinstance(v, int) else ("+" if v > 0 else "") + num(v)


def tables(doc):
    L = ["# V0.8c evaluation tables", "",
         f"`{doc['schema']}`; suite {doc['suite']}, engine {doc['engine']}, {doc['days']} days, seeds "
         f"{', '.join(str(s) for s in doc['seeds'])}. Every figure describes runs; none is a score. Money to the "
         "nearest dollar. Differences are policy minus no action on the same entry and seed.", ""]
    L += ["## Recording completeness (LLM runs)", "",
          "| Entry | Seed | Policy | Points | Every attempt recorded | Inputs verifiable | Attempts | Attempts per point "
          "| Outputs, reasons, actions recorded | Transport failures | Infrastructure-affected points | Truncated fields |",
          "|---|---|---|---|---|---|---|---|---|---|---|---|"]
    for r in doc["runs"]:
        c = r.get("completeness")
        if c:
            L.append(f"| {r['entry']} | {r['seed']} | {r['policy']} | {c['points']} | {c['attemptsRecorded']} | "
                     f"{c['inputsVerifiable']} | {c['attempts']} | "
                     + ", ".join(f"{k}: {v}" for k, v in c["attemptsPerPoint"].items())
                     + f" | {c['outputsRecorded']}, {c['reasonsRecorded']}, {c['actionsRecorded']} | "
                       f"{c['transportFailures']} | {c['infrastructureAffectedPoints']} | {c['truncatedFields']} |")
    L += ["", "## Funding refusals and what followed (LLM runs)", "",
          "Refusals by validate with code `insufficient_funds`, in any attempt, and their responses "
          "(dreamteam.refusal-response/v2). Proposed: actions in every parsed attempt.", "",
          "| Entry | Seed | Policy | Proposed actions | Funding refusals | Repeated | Revised: modified | Revised: different "
          "| No action | No longer applicable | Evidence unavailable | All validate refusals |",
          "|---|---|---|---|---|---|---|---|---|---|---|---|"]
    for r in doc["runs"]:
        if "refusalResponses" in r:
            f = r["refusalResponses"]["fundingRefusals"]
            L.append(f"| {r['entry']} | {r['seed']} | {r['policy']} | {r['proposedActions']} | {f['refusals']} | "
                     f"{f['identical_repeated']} | {f['modified_proposal']} | {f['different_proposal']} | "
                     f"{f['no_action']} | {f['refusal_no_longer_applicable']} | {f['evidence_unavailable']} | "
                     + (", ".join(f"{k} {v}" for k, v in r["validateRefusals"].items()) or "none") + " |")
    L += ["", "## Refusals of every code and what followed (LLM runs; dreamteam.refusal-response/v2)", "",
          "| Entry | Seed | Policy | Refusals | Repeated | Modified | Different | No action | No longer applicable | "
          "Evidence unavailable |", "|---|---|---|---|---|---|---|---|---|---|"]
    for r in doc["runs"]:
        if "refusalResponses" in r:
            f = r["refusalResponses"]["all"]
            L.append(f"| {r['entry']} | {r['seed']} | {r['policy']} | {f['refusals']} | {f['identical_repeated']} | "
                     f"{f['modified_proposal']} | {f['different_proposal']} | {f['no_action']} | "
                     f"{f['refusal_no_longer_applicable']} | {f['evidence_unavailable']} |")
    L += ["", "## Actions submitted, accepted and executed (every run; replay apply results)", "",
          "| Entry | Seed | Policy | Submitted | Accepted at apply | Executed (with a change) | Refused at apply | "
          "Executed by type |", "|---|---|---|---|---|---|---|---|"]
    for r in doc["runs"]:
        if r.get("replayed"):
            L.append(f"| {r['entry']} | {r['seed']} | {r['policy']} | {r['submitted']} | {r['accepted']} | "
                     f"{r['executed']} | " + (", ".join(f"{k} {v}" for k, v in r["refusedAtApply"].items()) or "0")
                     + " | " + (", ".join(f"{k} {v}" for k, v in r["executedByType"].items()) or "none") + " |")
    L += ["", "## Financial outcomes (every run)", "",
          "| Entry | Seed | Policy | Profit | Profit vs no action | Lowest cash | Most credit drawn | Days drawn | "
          "Interest | Shortfall (distress) | Days below zero |", "|---|---|---|---|---|---|---|---|---|---|---|"]
    for r in doc["runs"]:
        o, d, c = r["outcomes"], r["noAction"] or {}, r.get("credit") or {}
        L.append(f"| {r['entry']} | {r['seed']} | {r['policy']} | {money(o['operatingProfitCents'])} | "
                 f"{signed(d.get('operatingProfitCents')) if r['noAction'] else 'baseline'} | "
                 f"{money(o['minCashCents'])} | {money(c.get('maxDrawnCents'))} | {num(c.get('daysDrawn'))} | "
                 f"{money(c.get('interestCents'))} | {money(o['shortfallCents'])} | {num(o['daysBelowZero'])} |")
    L += ["", "## Operational outcomes (every run)", "",
          "| Entry | Seed | Policy | Exams | vs no action | Lost orders | vs no action | TAT breaches | vs no action | "
          "TAT p90 (min) | vs no action |", "|---|---|---|---|---|---|---|---|---|---|---|"]
    for r in doc["runs"]:
        o, d = r["outcomes"], r["noAction"] or {}
        mins = lambda v: None if not isinstance(v, int) else (v // 60 if v >= 0 else -((-v) // 60))
        L.append(f"| {r['entry']} | {r['seed']} | {r['policy']} | {num(o['examsCompleted'])} | "
                 f"{snum(d.get('examsCompleted'))} | {num(o['ordersLost'])} | {snum(d.get('ordersLost'))} | "
                 f"{num(o['tatBreaches'])} | {snum(d.get('tatBreaches'))} | {num(mins(o['tatP90Sec']))} | "
                 f"{snum(mins(d.get('tatP90Sec')))} |")
    L += ["", "## Repeated refused proposals: version 2 beside the frozen v1", "",
          "| Group | Model | Refusals | " + " | ".join(P.STATUSES) + " | v2 findings |",
          "|---|---|---|" + "---|" * len(P.STATUSES) + "---|"]
    for k, s in doc["repeatV2"]["summary"].items():
        g, m = k.split("|")
        L.append(f"| {g} | {m} | {s['refusals']} | " + " | ".join(str(s[x]) for x in P.STATUSES) + f" | {s['findings']} |")
    L += ["", "| Group | Model | v1 links | v2 links | both | v1 only: v2 abstains | v1 only: different actions | "
          "v2 only: message differs | v2 only: other |", "|---|---|---|---|---|---|---|---|---|"]
    for k, s in doc["repeatV2"]["versusV1"].items():
        g, m = k.split("|")
        L.append(f"| {g} | {m} | {s['v1Links']} | {s['v2Links']} | {s['both']} | {s['v1OnlyV2Abstains']} | "
                 f"{s['v1OnlyDifferentActions']} | {s['v2OnlyMessageDiffers']} | {s['v2OnlyOther']} |")
    v1 = doc["decisionEvidenceV1"]
    L += ["", "## The frozen rules (dreamteam.decision-evidence/v1) on the llm-agent 2 view", "",
          "Findings by status: " + ", ".join(f"{k} {v}" for k, v in sorted((v1["byStatus"] or {}).items())) + ".", "",
          "| Rule | Supported | Unresolved | Insufficient evidence |", "|---|---|---|---|"]
    for rule, s in sorted((v1["byRule"] or {}).items()):
        if isinstance(s, dict):
            L.append(f"| {rule} | {s.get('supported', 0)} | {s.get('unresolved', 0)} | {s.get('insufficient_evidence', 0)} |")
    L += ["", "The independent review of the v1 findings is PENDING: none of these findings has been independently "
          "validated."]
    return "\n".join(L) + "\n"


def main(argv=None):
    ap = argparse.ArgumentParser(description="The V0.8c evaluation's analysis (read-only; no model).")
    ap.add_argument("--corpus", required=True)
    ap.add_argument("--contract", required=True)
    ap.add_argument("--replays", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    doc = analyze(a.corpus, a.contract, a.replays)
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, "v08c-analysis.json"), "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True, ensure_ascii=True)
        f.write("\n")
    with open(os.path.join(a.out, "v08c-tables.md"), "w", encoding="utf-8") as f:
        f.write(tables(doc))
    print(f"V0.8c analysis: {len(doc['runs'])} runs, {len(doc['refusalResponseV2']['events'])} refusals, "
          f"{len(doc['repeatV2']['findings'])} repeat v2 findings")
    return 0


if __name__ == "__main__":
    sys.exit(main())
