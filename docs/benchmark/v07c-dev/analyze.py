#!/usr/bin/env python3
"""V0.7c tables (PLAN.md §5) from one stage's suite-run results: the
contract report (contract.json), each run's result.json and decisions.json,
and the LLM agent's evidence (policy-evidence/decisions.jsonl, summary.json).
Existing metrics only: no composite score, no ranking, no test of
significance.

    python3 -I -B docs/benchmark/v07c-dev/analyze.py RESULTS > tables.md

Nothing is written elsewhere.

Action counts (§5.2):
* proposed: every action an agent put forward. For the LLM agents, each
  action of each validate attempt (an action validated, refused and
  proposed again counts twice; one validated and then submitted, once);
  for the built-in policies, which do not validate, the actions they
  submitted.
* refused by validate: a validate result that is not ok (the agent may
  then submit something else). batch_rejected is counted apart: it refuses
  an action only because another in its batch was refused.
* submitted, accepted, refused at apply: the harness's decisions.json.
* executed: accepted, and in effect by the end of the window (effectiveAt,
  where the result gives one, on or before the window's last day; an
  action without one takes effect when applied)."""

import json
import os
import sys

STAFFING = ("hire", "terminate", "set_roster", "set_reading_allocation", "contract_teleradiology")
INFRA = ("http_status", "timeout", "connection")
FUNDS = "insufficient_funds"
KEY = [("operatingProfitCents", "profit", "$"), ("netRevenueCents", "revenue", "$"),
       ("operatingExpenseCents", "expenses", "$"), ("tatP90Sec", "TAT p90", "min"), ("tatBreaches", "breaches", ""),
       ("examsCompleted", "exams", ""), ("ordersLost", "lost", ""), ("overtimeSec", "overtime", "min")]


def fmt(v, unit="$", signed=False):
    if v is None:
        return "–"
    s = "+" if signed and v > 0 else ""
    if unit == "$":
        return f"{'-' if v < 0 else s}${abs(v) / 100:,.0f}"
    if unit == "min":
        return f"{s}{v / 60:,.0f}"
    return f"{s}{v:,}"


def table(head, rows):
    out = ["| " + " | ".join(head) + " |", "|" + "---|" * len(head)]
    return "\n".join(out + ["| " + " | ".join(str(c) for c in r) + " |" for r in rows])


def clip(s, n):
    s = " ".join(str(s or "").split()).replace("|", "/")
    return s if len(s) <= n else s[:n - 1] + "…"


def load_json(p, default=None):
    if not os.path.exists(p):
        return default
    with open(p, encoding="utf-8") as f:
        return json.load(f)


def jsonl(p):
    if not os.path.exists(p):
        return None
    with open(p, encoding="utf-8") as f:
        return [json.loads(x) for x in f if x.strip()]


def metric(run, name):
    for d in run["dimensions"]:
        for m in d["metrics"]:
            if m["name"] == name:
                return m.get("value")
    return None


def code(err):
    rest = err.split(": ", 1)[1] if ": " in err else err
    return rest.split(":", 1)[0].strip()


def akey(a):
    """What identifies an action for repetition: its type and its target."""
    return (a.get("type"), a.get("employee") or a.get("contract") or a.get("modality") or a.get("policy") or "")


def adesc(a):
    keep = {k: v for k, v in a.items() if k != "type"}
    return f"{a.get('type')} " + ", ".join(f"{k}={v}" for k, v in sorted(keep.items()))


class Run:
    def __init__(self, d, c):
        self.c = c
        rd = os.path.join(d, "runs", c["id"])
        self.result = load_json(os.path.join(rd, "result.json"), {})
        self.decisions = load_json(os.path.join(rd, "decisions.json"), []) or []
        self.recs = jsonl(os.path.join(rd, "policy-evidence", "decisions.jsonl"))
        self.summary = load_json(os.path.join(rd, "policy-evidence", "summary.json"), {})
        self.cal = self.result.get("calendar") or []
        self.money = (self.result.get("window") or {}).get("money") or {}
        self.work = ((self.result.get("window") or {}).get("ops") or {}).get("workforce") or {}
        # The window ends at a decision time (07:00); an action effective on
        # that day took effect at 00:00, inside the window.
        self.end = (((self.result.get("window") or {}).get("to") or {}).get("time") or "")[:10]

    def applied(self):
        """(decision time, result) for every action submitted."""
        for p in self.decisions:
            for b in p.get("batches") or []:
                for x in b.get("results") or []:
                    yield p["time"], x

    def validated(self):
        """(point, time, attempt, result) for every action an LLM agent validated."""
        for r in self.recs or []:
            for i, v in enumerate(r.get("validation") or []):
                for x in v.get("results") or []:
                    yield r["point"], r["time"], i + 1, x

    def cash_before(self, t):
        """Cash and credit drawn at the end of the day before time t."""
        prev = [d for d in self.cal if d["date"] < t[:10]]
        if not prev:
            return None, None
        return prev[-1].get("cashEndCents"), prev[-1].get("creditDrawnEndCents", 0)

    def credit(self):
        drawn = max([d.get("creditDrawnEndCents", 0) for d in self.cal] or [0])
        return drawn, self.money.get("interestCents", 0), max(0, -(self.money.get("minCashCents") or 0))

    def counts(self):
        sub = [x for _, x in self.applied()]
        acc = [x for x in sub if x.get("ok")]
        exe = [x for x in acc if (x.get("effectiveAt") or "")[:10] <= self.end]
        if self.recs is None:
            prop, vref, vbr = len(sub), 0, 0
        else:
            val = list(self.validated())
            # Each validate attempt is a proposal; a batch submitted without
            # one (the agent validates first) is counted as proposed too.
            prop = len(val) + sum(len(b) for r in self.recs if not r.get("validation") for b in r.get("batches") or [])
            vref = sum(1 for *_, x in val if not x.get("ok") and x.get("code") != "batch_rejected")
            vbr = sum(1 for *_, x in val if x.get("code") == "batch_rejected")
        aref = sum(1 for x in sub if not x.get("ok") and x.get("code") != "batch_rejected")
        abr = sum(1 for x in sub if x.get("code") == "batch_rejected")
        funds_v = 0 if self.recs is None else sum(1 for *_, x in self.validated() if x.get("code") == FUNDS)
        funds_a = sum(1 for x in sub if x.get("code") == FUNDS)
        return {"proposed": prop, "vref": vref, "vbr": vbr, "submitted": len(sub), "accepted": len(acc),
                "aref": aref, "abr": abr, "executed": len(exe), "funds_v": funds_v, "funds_a": funds_a,
                "ops": sum(1 for x in exe if x["action"].get("type") not in STAFFING),
                "staff": sum(1 for x in exe if x["action"].get("type") in STAFFING)}


def behaviour(recs):
    seen, repeated, refusals, errors = set(), 0, {}, {}
    first_valid = details = 0
    out = {}
    for r in recs:
        here = set()
        for x in r.get("refusals") or []:
            k = (x.get("type"), x.get("code"))
            refusals[f"{k[0]} {k[1]}"] = refusals.get(f"{k[0]} {k[1]}", 0) + 1
            if k in seen:
                repeated += 1
            here.add(k)
        seen |= here
        errs = r.get("errors") or []
        for e in errs:
            errors[code(e)] = errors.get(code(e), 0) + 1
        if r.get("outcome") != "fallback" and not any(e.startswith("attempt 1:") for e in errs) and \
                not any(x.get("attempt") == 1 for x in r.get("refusals") or []):
            first_valid += 1
        details += 1 if r.get("detailsRequested") else 0
        out[r.get("outcome")] = out.get(r.get("outcome"), 0) + 1
    return {"points": len(recs), "actions": out.get("actions", 0), "none": out.get("no_action", 0),
            "fallback": out.get("fallback", 0), "first": first_valid, "refusals": refusals, "repeated": repeated,
            "errors": errors, "details": details}


def funds_events(run):
    """Every insufficient_funds refusal, at validate or at apply, in time
    order: (time, where, action)."""
    ev = []
    if run.recs is not None:
        ev += [(t, f"validate {i}", x["action"]) for _, t, i, x in run.validated() if x.get("code") == FUNDS]
    ev += [(t, "apply", x["action"]) for t, x in run.applied() if x.get("code") == FUNDS]
    return sorted(ev, key=lambda e: e[0])


def responses(run):
    """After each action refused insufficient_funds, what the policy put
    forward for the same type and target later in the run: how often, how
    many of those were refused again for funds, and how many were accepted."""
    ev = funds_events(run)
    if not ev:
        return []
    later = []  # (time, action, outcome code)
    if run.recs is not None:
        later += [(t, x["action"], x.get("code")) for _, t, _, x in run.validated()]
    later += [(t, x["action"], x.get("code")) for t, x in run.applied()]
    out, done = [], set()
    for t, where, a in ev:
        k = akey(a)
        if (k, t) in done:
            continue
        done.add((k, t))
        after = [(u, b, c) for u, b, c in later if u > t and akey(b) == k]
        out.append([t[:10], where, clip(adesc(a), 90), len({u for u, _, _ in after}),
                    sum(1 for *_, c in after if c == FUNDS), sum(1 for *_, c in after if c == "accepted")])
    return out


def main(d):
    k = load_json(os.path.join(d, "contract.json"))
    p = print
    entries = [e["id"] for e in k["entries"]]
    pols = [x["id"] for x in k["policies"]]
    runs = {(r["entry"], r["seed"], r["policy"]["id"]): Run(d, r) for r in k["runs"]}
    pairs = {(x["entry"], x["seed"], x["policy"]): {dl["name"]: dl["value"] for dim in x["dimensions"] for dl in dim["deltas"]}
             for x in k["pairs"]}
    p(f"# {k['suiteRun']}: suite {k['suite']['id']} version {k['suite']['version']}, seeds "
      f"{', '.join(map(str, k['seeds']))}, {k['days']} days, engine {k['engine']}, status {k['status']} "
      f"({k['completedRuns']} of {k['expectedRuns']} runs, {len(k.get('failedRuns') or [])} failed)\n")
    bad = [r.c["id"] for r in runs.values() if r.c["status"] != "completed" or not r.c["replayVerified"]]
    if bad:
        p(f"**Runs not completed or not replay-verified:** {', '.join(bad)}\n")

    p("## 1. Starting states (identical across the policies of each entry and seed)\n")
    rows = []
    for e in entries:
        for s in k["seeds"]:
            ds = {r.c["startingStateDigest"] for (e2, s2, _), r in runs.items() if (e2, s2) == (e, s)}
            rows.append([e, s, len(ds) == 1 and "yes" or f"**no** ({len(ds)} digests)"])
    p(table(["entry", "seed", "identical"], rows))

    p("\n## 2. Outcomes (absolute, each run)\n")
    p("Overtime cost is the harness's (all staff, before benefits); approved is the part from extend_shift. "
      "Most drawn is the credit line's largest end-of-day balance; shortfall is the largest unfunded amount "
      "(cash below zero with the line fully drawn).\n")
    rows = []
    for e in entries:
        for s in k["seeds"]:
            for pol in pols:
                r = runs.get((e, s, pol))
                if not r:
                    continue
                drawn, interest, short = r.credit()
                rows.append([e, s, pol, r.c["status"], "yes" if r.c["replayVerified"] else "NO"] +
                            [fmt(metric(r.c, n), u) for n, _, u in KEY] +
                            [fmt(r.work.get("overtimeCostCents")), fmt(r.work.get("approvedOvertimeCostCents")),
                             fmt(r.money.get("cashEndCents")), fmt(r.money.get("minCashCents")), fmt(drawn),
                             fmt(interest), fmt(short)])
    p(table(["entry", "seed", "policy", "status", "replay"] + [h for _, h, _ in KEY] +
            ["overtime cost", "approved overtime cost", "cash end", "min cash", "most drawn", "interest", "shortfall"], rows))

    p("\n## 3. Paired with no action (policy − noop, same entry and seed; minutes for times)\n")
    rows = []
    for e in entries:
        for s in k["seeds"]:
            base = runs.get((e, s, "noop"))
            for pol in pols:
                dl = pairs.get((e, s, pol))
                r = runs.get((e, s, pol))
                if dl is None or r is None or base is None:
                    continue
                better = (dl.get("tatP90Sec") or 0) < 0 or (dl.get("tatBreaches") or 0) < 0
                flag = "better service, lower profit" if better and (dl.get("operatingProfitCents") or 0) < 0 else ""
                dot = (r.work.get("overtimeCostCents") or 0) - (base.work.get("overtimeCostCents") or 0)
                dcash = (r.money.get("cashEndCents") or 0) - (base.money.get("cashEndCents") or 0)
                dmin = (r.money.get("minCashCents") or 0) - (base.money.get("minCashCents") or 0)
                rows.append([e, s, pol] + [fmt(dl.get(n), u, True) for n, _, u in KEY] +
                            [fmt(dot, "$", True), fmt(dcash, "$", True), fmt(dmin, "$", True), flag])
    p(table(["entry", "seed", "policy"] + ["Δ" + h for _, h, _ in KEY] +
            ["Δovertime cost", "Δcash end", "Δmin cash", "flag"], rows))

    p("\n## 4. Actions: proposed, refused, accepted, executed\n")
    p("Proposed counts each action each time it was put forward (each validate attempt; the built-in policies' submissions); "
      "`batch_rejected` is counted apart. Executed: accepted and in effect by the end of the window. "
      "Funds: refused insufficient_funds (at validate / at apply).\n")
    rows = []
    for e in entries:
        for s in k["seeds"]:
            for pol in pols:
                r = runs.get((e, s, pol))
                if not r:
                    continue
                c = r.counts()
                byt = {}
                for _, x in r.applied():
                    if x.get("ok"):
                        t = x["action"].get("type")
                        byt[t] = byt.get(t, 0) + 1
                rows.append([e, s, pol, c["proposed"], f"{c['vref']} (+{c['vbr']})", c["submitted"], c["accepted"],
                             f"{c['aref']} (+{c['abr']})", c["executed"], f"{c['ops']}/{c['staff']}",
                             f"{c['funds_v']}/{c['funds_a']}",
                             "; ".join(f"{t}: {n}" for t, n in sorted(byt.items())) or "none"])
    p(table(["entry", "seed", "policy", "proposed", "refused by validate (+batch)", "submitted", "accepted",
             "refused at apply (+batch)", "executed", "executed ops/staffing", "funds v/a", "accepted by type"], rows))

    p("\n## 5. Refused for funds, and what followed\n")
    p("Each action refused insufficient_funds; later in the run, the decision points at which the policy put "
      "forward the same type for the same target, how many of those were refused for funds again, and how many "
      "were accepted.\n")
    rows = []
    for (e, s, pol), r in sorted(runs.items()):
        for x in responses(r):
            rows.append([e, s, pol] + x)
    p(table(["entry", "seed", "policy", "date", "where", "action", "later points with the same", "refused for funds again",
             "accepted later"], rows) if rows else "None: no action was refused insufficient_funds.")

    llm = sorted((key, r) for key, r in runs.items() if r.recs is not None)
    if not llm:
        return
    p("\n## 6. LLM decisions\n")
    rows, refusal_rows = [], []
    for (e, s, pol), r in llm:
        b = behaviour(r.recs)
        rows.append([e, s, pol, b["points"], b["actions"], b["none"], b["fallback"], f"{b['first']}/{b['points']}",
                     b["repeated"], b["details"], sum(v for c, v in b["errors"].items() if c in INFRA)])
        refusal_rows.append([e, s, pol, "; ".join(f"{c}: {v}" for c, v in sorted(b["refusals"].items())) or "none",
                             "; ".join(f"{c}: {v}" for c, v in sorted(b["errors"].items())) or "none"])
    p(table(["entry", "seed", "policy", "points", "with actions", "no action", "fallback", "first-attempt valid",
             "repeated invalid", "details", "infrastructure errors"], rows))
    p("\n## 7. Refusals by validate, and agent errors\n")
    p(table(["entry", "seed", "policy", "refusals (type code: count)", "errors (code: count)"], refusal_rows))

    p("\n## 8. LLM actions in time order, with cash and credit\n")
    p("Every action an LLM agent submitted, with the outcome, and the cash and credit drawn at the end of the day "
      "before; and the agent's recorded reason at each point where an action was refused for funds.\n")
    for (e, s, pol), r in llm:
        rows = []
        for t, x in r.applied():
            cash, drawn = r.cash_before(t)
            eff = x.get("effectiveAt") or ""
            rows.append([t[:10], clip(adesc(x["action"]), 100), x.get("code"),
                         ("yes" if eff[:10] <= r.end else "no") if x.get("ok") else "", fmt(cash), fmt(drawn)])
        if not rows:
            continue
        p(f"\n### {e} {s} {pol}\n")
        p(table(["date", "action", "outcome", "executed", "cash day before", "drawn day before"], rows))
        why = [(x["time"][:10], x.get("outcome"), x.get("reason")) for x in r.recs
               if any(y.get("code") == FUNDS for y in x.get("refusals") or []) or
               any(y.get("code") == FUNDS for v in x.get("validation") or [] for y in v.get("results") or [])]
        for t, o, reason in why:
            p(f"* {t} ({o}), reason recorded: “{clip(reason, 400)}”")

    p("\n## 9. Tokens, latency and cost\n")
    rows = []
    for (e, s, pol), r in llm:
        sm = r.summary or {}
        t = sm.get("tokens") or {}
        rep, lat = t.get("reported") or {}, sm.get("latencyMs") or {}
        cost = sm.get("cost") or {}
        rows.append([e, s, pol, sm.get("requests"), sm.get("retries"), f"{rep.get('prompt', 0):,}", f"{rep.get('completion', 0):,}",
                     f"{(t.get('estimated') or {}).get('requests', 0)}/{(t.get('unknown') or {}).get('requests', 0)}",
                     f"{lat.get('mean', 0) / 1000:.1f}", f"{lat.get('max', 0) / 1000:.1f}", f"{lat.get('total', 0) / 1000:,.0f}",
                     cost.get("note") or cost.get("status") or ("none configured" if not cost else json.dumps(cost)[:40])])
    p(table(["entry", "seed", "policy", "requests", "retries", "prompt tokens", "completion tokens", "estimated/unknown",
             "latency mean s", "max s", "total s", "estimated cost"], rows))


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    main(sys.argv[1])
