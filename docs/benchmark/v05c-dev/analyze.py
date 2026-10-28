#!/usr/bin/env python3
"""V0.5c development evaluation tables (PLAN.md §5) from benchmark result
directories: the harness's report.json and decisions.json, and the LLM
agent's evidence (decisions.jsonl, summary.json, and model.jsonl when it is
there). Existing metrics only: no composite score, no ranking, no test of
significance.

    python3 -I -B docs/benchmark/v05c-dev/analyze.py RESULTS [RESULTS ...] > tables.md

Each RESULTS is a `dreamteam benchmark run` output directory. The output
is Markdown; nothing is written elsewhere."""

import json
import os
import statistics
import sys

STAFFING = ("hire", "terminate", "set_roster", "set_reading_allocation", "contract_teleradiology")
INFRA = ("http_status", "timeout", "connection")
OUTCOME = [("operatingProfitCents", "Δprofit", "$"), ("netRevenueCents", "Δrevenue", "$"),
           ("operatingExpenseCents", "Δexpenses", "$"), ("reportsFinal", "Δreports", ""),
           ("tatP50Sec", "ΔTAT p50", "min"), ("tatP90Sec", "ΔTAT p90", "min"), ("tatBreaches", "Δbreaches", ""),
           ("examsCompleted", "Δexams", ""), ("ordersLost", "Δlost", ""), ("overtimeSec", "Δovertime", "min")]
ABS = [("operatingProfitCents", "profit", "$"), ("netRevenueCents", "revenue", "$"),
       ("operatingExpenseCents", "expenses", "$"), ("reportsFinal", "reports", ""), ("tatP50Sec", "TAT p50", "min"),
       ("tatP90Sec", "TAT p90", "min"), ("tatBreaches", "breaches", ""), ("examsCompleted", "exams", ""),
       ("ordersLost", "lost", ""), ("overtimeSec", "overtime", "min")]


def fmt(v, unit, signed=False):
    if v is None:
        return "–"
    s = "+" if signed and v > 0 else ""
    if unit == "$":
        return f"{'-' if v < 0 else s}${abs(v) / 100:,.0f}"
    if unit == "min":
        return f"{s}{v / 60:,.0f} min" if not signed else f"{s}{v / 60:,.0f}"
    return f"{s}{v:,}"


def jsonl(p):
    if not os.path.exists(p):
        return None
    with open(p, encoding="utf-8") as f:
        return [json.loads(x) for x in f if x.strip()]


def load(dirs):
    runs = []
    for d in dirs:
        with open(os.path.join(d, "report.json"), encoding="utf-8") as f:
            rep = json.load(f)
        pairs = {(p["seed"], p["policy"]): {x["name"]: x["value"] for x in p["deltas"]} for p in rep.get("pairs") or []}
        for r in rep["runs"]:
            rd = os.path.join(d, "runs", r["id"])
            ev = os.path.join(rd, "policy-evidence")
            run = {"stage": rep["benchmark"], "id": r["id"], "seed": r["seed"], "policy": r["policy"]["id"],
                   "status": r["status"], "counts": r["counts"], "replay": r.get("replayVerified"),
                   "match": r.get("actionsMatchSession"), "digest": r.get("finalRunDigest"),
                   "metrics": {m["name"]: m["value"] for m in r.get("metrics") or []},
                   "delta": pairs.get((r["seed"], r["policy"]["id"])), "days": rep["days"]}
            hd = os.path.join(rd, "decisions.json")
            run["harness"] = json.load(open(hd, encoding="utf-8")) if os.path.exists(hd) else []
            run["agent"] = jsonl(os.path.join(ev, "decisions.jsonl"))
            sp = os.path.join(ev, "summary.json")
            run["summary"] = json.load(open(sp, encoding="utf-8")) if os.path.exists(sp) else None
            run["model"] = jsonl(os.path.join(ev, "model.jsonl"))
            runs.append(run)
    return runs


def code(err):
    """The error code of an agent error line ("attempt N: code: …")."""
    rest = err.split(": ", 1)[1] if ": " in err else err
    return rest.split(":", 1)[0].strip()


def behaviour(run):
    """Decision-level counts from the agent's records and the harness's
    action results."""
    recs = run["agent"] or []
    seen, repeated, refused_codes, errors = set(), 0, {}, {}
    first_valid = details = 0
    for r in recs:
        here = set()
        for x in r.get("refusals") or []:
            k = (x.get("type"), x.get("code"))
            refused_codes[f"{k[0]} {k[1]}"] = refused_codes.get(f"{k[0]} {k[1]}", 0) + 1
            if k in seen:
                repeated += 1
            here.add(k)
        seen |= here
        errs = r.get("errors") or []
        for e in errs:
            c = code(e)
            errors[c] = errors.get(c, 0) + 1
        if r.get("outcome") != "fallback" and not any(e.startswith("attempt 1:") for e in errs) and \
                not any(x.get("attempt") == 1 for x in r.get("refusals") or []):
            first_valid += 1
        if r.get("detailsRequested"):
            details += 1
    proposed, accepted, refused_apply = {}, {}, 0
    for r in recs:
        for b in r.get("proposed") or []:
            for a in (b if isinstance(b, list) else [b]):
                if isinstance(a, dict):
                    proposed[a.get("type")] = proposed.get(a.get("type"), 0) + 1
    staffing_proposed = sum(v for k, v in proposed.items() if k in STAFFING) + \
        sum(1 for r in recs for x in r.get("refusals") or [] if x.get("type") in STAFFING)
    for p in run["harness"]:
        for b in p.get("batches") or []:
            for x in b.get("results") or []:
                t = (x.get("action") or {}).get("type")
                if x.get("ok"):
                    accepted[t] = accepted.get(t, 0) + 1
                else:
                    refused_apply += 1
    truncated = sum(1 for m in run["model"] or [] if '"finish_reason":"length"' in (m.get("response") or "").replace(" ", ""))
    out = {r.get("outcome"): 0 for r in recs}
    for r in recs:
        out[r.get("outcome")] += 1
    return {"points": len(recs), "actions": out.get("actions", 0), "no_action": out.get("no_action", 0),
            "fallback": out.get("fallback", 0), "first_valid": first_valid,
            "refused_points": sum(1 for r in recs if r.get("refusals")), "refusals": refused_codes,
            "repeated": repeated, "errors": errors, "details": details, "proposed": proposed,
            "accepted": accepted, "staffing_proposed": staffing_proposed, "refused_apply": refused_apply, "truncated": truncated if run["model"] else None}


def cat(counts, staffing):
    return sum(v for k, v in counts.items() if (k in STAFFING) == staffing)


def table(head, rows):
    out = ["| " + " | ".join(head) + " |", "|" + "---|" * len(head)]
    out += ["| " + " | ".join(str(c) for c in r) + " |" for r in rows]
    return "\n".join(out)


def main(dirs):
    runs = load(dirs)
    llm = [r for r in runs if r["agent"] is not None]
    for r in llm:
        r["b"] = behaviour(r)
    p = print
    p("## Runs\n")
    p(table(["stage", "run", "status", "points", "batches applied/refused", "protocol errors", "denied", "request errors",
             "replay verified", "final run digest"],
            [[r["stage"], r["id"], r["status"], r["counts"]["decisionPoints"],
              f"{r['counts']['applied']}/{r['counts']['refused']}", r["counts"]["protocolErrors"], r["counts"]["denied"],
              r["counts"]["requestErrors"], "yes" if r["replay"] else "NO", (r["digest"] or "")[:12] + "…"] for r in runs]))
    p("\n## Decisions\n")
    p(table(["stage", "run", "points", "with actions", "no action", "fallback", "valid", "first-attempt valid",
             "points with a refused proposal", "repeated invalid proposals", "details requests", "replies cut off"],
            [[r["stage"], r["id"], b["points"], b["actions"], b["no_action"], b["fallback"],
              f"{b['points'] - b['fallback']}/{b['points']}", f"{b['first_valid']}/{b['points']}", b["refused_points"],
              b["repeated"], b["details"], "–" if b["truncated"] is None else b["truncated"]]
             for r in llm for b in [r["b"]]]))
    p("\n## Refusals by validate, and agent errors\n")
    p(table(["stage", "run", "refusals (type code: count)", "errors (code: count)", "infrastructure errors"],
            [[r["stage"], r["id"], "; ".join(f"{k}: {v}" for k, v in sorted(b["refusals"].items())) or "none",
              "; ".join(f"{k}: {v}" for k, v in sorted(b["errors"].items())) or "none",
              sum(v for k, v in b["errors"].items() if k in INFRA)] for r in llm for b in [r["b"]]]))
    p("\n## Actions\n")
    p(table(["stage", "run", "last proposals (type: count)", "accepted (type: count)", "operations accepted",
             "staffing accepted", "staffing proposed or refused", "refused when applied"],
            [[r["stage"], r["id"], "; ".join(f"{k}: {v}" for k, v in sorted(b["proposed"].items())) or "none",
              "; ".join(f"{k}: {v}" for k, v in sorted(b["accepted"].items())) or "none",
              cat(b["accepted"], False), cat(b["accepted"], True), b["staffing_proposed"], b["refused_apply"]]
             for r in llm for b in [r["b"]]]))
    p("\n## Outcomes (absolute)\n")
    p(table(["stage", "seed", "policy"] + [h for _, h, _ in ABS],
            [[r["stage"], r["seed"], r["policy"]] + [fmt(r["metrics"].get(k), u) for k, _, u in ABS] for r in runs]))
    p("\n## Paired with noop (policy − noop, same stage and seed; minutes for times)\n")
    p(table(["stage", "seed", "policy"] + [h for _, h, _ in OUTCOME] + ["flag"],
            [[r["stage"], r["seed"], r["policy"]] + [fmt(r["delta"].get(k), u, True) for k, _, u in OUTCOME] +
             [flag(r)] for r in runs if r["delta"]]))
    p("\n## Tokens and latency\n")
    rows = []
    for r in llm:
        s = r["summary"] or {}
        t = s.get("tokens") or {}
        rep, est = t.get("reported") or {}, t.get("estimated") or {}
        lat = s.get("latencyMs") or {}
        firsts = [m["usage"]["prompt_tokens"] for m in r["model"] or []
                  if m.get("turn") == "first" and isinstance(m.get("usage"), dict) and m["usage"].get("prompt_tokens")]
        rows.append([r["stage"], r["id"], s.get("requests"), s.get("retries"), s.get("detailRequests"),
                     f"{rep.get('prompt', 0):,}", f"{rep.get('completion', 0):,}",
                     f"{est.get('requests', 0)}/{(t.get('unknown') or {}).get('requests', 0)}",
                     f"{statistics.mean(firsts):,.0f}" if firsts else "–",
                     f"{lat.get('mean', 0) / 1000:.1f}", f"{lat.get('median', 0) / 1000:.1f}",
                     f"{lat.get('max', 0) / 1000:.1f}", f"{lat.get('total', 0) / 1000:,.0f}"])
    p(table(["stage", "run", "requests", "retries", "details", "prompt tokens", "completion tokens",
             "estimated/unknown requests", "mean first prompt", "latency mean s", "median s", "max s", "total s"], rows))


def flag(r):
    d = r["delta"]
    better = (d.get("tatP90Sec") or 0) < 0 or (d.get("tatBreaches") or 0) < 0
    return "better service, lower profit" if better and (d.get("operatingProfitCents") or 0) < 0 else ""


if __name__ == "__main__":
    if len(sys.argv) < 2:
        sys.exit(__doc__)
    main(sys.argv[1:])
