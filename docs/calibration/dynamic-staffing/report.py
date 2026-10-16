#!/usr/bin/env python3
"""Tabulates the dynamic staffing benchmark: results/<set>/*.json ->
results/<set>/report.md and results/<set>/trajectory.csv.

    python3 docs/calibration/dynamic-staffing/report.py [dev|eval]

Every comparison is paired by seed against the set's no-action plan.
Spans: T30 = days 1-30; D31-90 and D91-180 are differences of cumulative
figures (finance, exams, reports, utilization counters) or, for orders and
studies, those received or acquired in the span.
"""
import csv
import glob
import json
import math
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
CONTROL = "A-no-action"
MODS = ["XR", "US", "CT", "MR"]
PRIS = ["urgent", "soon", "routine"]
# Two-sided 95% Student t quantiles by degrees of freedom.
T975 = {1: 12.706, 2: 4.303, 3: 3.182, 4: 2.776, 5: 2.571, 6: 2.447, 7: 2.365, 8: 2.306, 9: 2.262,
        10: 2.228, 11: 2.201, 12: 2.179, 13: 2.160, 14: 2.145, 15: 2.131, 16: 2.120, 17: 2.110,
        18: 2.101, 19: 2.093, 20: 2.086, 24: 2.064, 29: 2.045}


def load(set_):
    out = {}
    for p in sorted(glob.glob(os.path.join(HERE, "results", set_, "*.json"))):
        r = json.load(open(p))
        out[r["plan"]["name"]] = r
    return out


def mean(xs):
    return sum(xs) / len(xs) if xs else float("nan")


def ci(xs):
    """Mean and 95% t half-width."""
    n = len(xs)
    m = mean(xs)
    if n < 2:
        return m, float("nan")
    sd = math.sqrt(sum((x - m) ** 2 for x in xs) / (n - 1))
    return m, T975.get(n - 1, 1.96) * sd / math.sqrt(n)


def k(c):
    return c / 100_000  # cents -> $k


def fmt(x, d=1):
    return "—" if x is None or (isinstance(x, float) and math.isnan(x)) else f"{x:,.{d}f}"


def rng(xs, d=1):
    return f"{fmt(mean(xs), d)} ({fmt(min(xs), d)} to {fmt(max(xs), d)})"


def cis(xs, d=1):
    m, h = ci(xs)
    return f"{fmt(m, d)} ± {fmt(h, d)}"


# --- per-seed span figures -------------------------------------------------

SPANS = [("T30", 0, 30), ("D31-90", 30, 90), ("D91-180", 90, 180), ("T90", 0, 90), ("T180", 0, 180)]


def daily_at(s, field, d):
    return 0 if d == 0 else s["daily"][field][d - 1]


def fin(s, a, b):
    """Finance over days (a, b]: cumulative differences."""
    f = {}
    for key, field in [("profit", "profitCents"), ("netRevenue", "netRevenueCents"), ("opex", "opexCents")]:
        f[key] = k(daily_at(s, field, b) - daily_at(s, field, a))
    f["margin"] = 100 * f["profit"] / f["netRevenue"] if f["netRevenue"] > 0 else float("nan")
    return f


def window(s, d):
    return s["windows"][f"T{d}"]


def cum(s, d, f):
    return 0 if d == 0 else f(s, d)


def exams_mod(s, d):
    return dict(zip(window(s, d)["mods"], window(s, d)["exams"]))


def span_ops(s, a, b):
    """Operations over (a, b]."""
    o = {}
    ea = exams_mod(s, a) if a else {m: 0 for m in MODS}
    eb = exams_mod(s, b)
    o["exams"] = {m: eb[m] - ea[m] for m in MODS}
    ra = s["extra"][f"T{a}"]["reportsByModality"] if a else {}
    rb = s["extra"][f"T{b}"]["reportsByModality"]
    o["reports"] = {m: rb.get(m, 0) - ra.get(m, 0) for m in MODS}
    xa = s["extra"].get(f"T{a}") if a else None
    xb = s["extra"][f"T{b}"]

    def d(field, m=None):
        vb = xb[field][m] if m else xb[field]
        va = (xa[field][m] if m else xa[field]) if xa else 0
        return vb - va

    o["walkInsLeft"] = d("walkInsLeft")
    o["utilStaffed"] = {m: 100 * d("roomBusySec", m) / max(1, d("roomStaffedSec", m)) for m in MODS}
    o["utilOpen"] = {m: 100 * d("roomBusySec", m) / max(1, d("roomOpenSec", m)) for m in MODS}
    o["staffedH"] = {m: d("roomStaffedSec", m) / 3600 for m in MODS}
    o["techUtil"] = 100 * d("techBusySec") / max(1, d("techDutySec"))
    o["readUtil"] = 100 * d("readInBlockSec") / max(1, d("readBlockSec"))
    o["techDutyH"] = d("techDutySec") / 3600
    disp_b = xb["displacement"]
    disp_a = xa["displacement"] if xa else {}
    o["displaced"] = disp_b.get("events", 0) - disp_a.get("events", 0)
    o["rebooked"] = disp_b.get("rebooked", 0) - disp_a.get("rebooked", 0)
    o["dispLost"] = (disp_b.get("noFeasibleSlot", 0) + disp_b.get("declined", 0)
                     - disp_a.get("noFeasibleSlot", 0) - disp_a.get("declined", 0))
    # Orders and studies of the span.
    if a == 0:
        w = window(s, b)
        o["tat"] = w["tatByPriority"]
        o["orders"] = dict(zip(w["mods"], w["orders"]))
        o["lost"] = {m: w["lostDeclined"][i] + w["lostNoSlot"][i] for i, m in enumerate(w["mods"])}
        o["lossPrimary"] = w["lossPrimary"]
        o["leadP50"] = dict(zip(w["mods"], w["leadP50Days"]))
        o["leadP90"] = dict(zip(w["mods"], w["leadP90Days"]))
    else:
        sp = s["spans"][f"D{a + 1}-{b}"]
        o["tat"] = sp["tatByPriority"]
        o["orders"] = sp["orders"]
        o["lost"] = sp["lost"]
        o["lossPrimary"] = sp["lossPrimary"]
        o["leadP50"] = sp["leadP50Days"]
        o["leadP90"] = sp["leadP90Days"]
    return o


def breach_rate(o, p):
    t = o["tat"].get(p)
    return 100 * t["breaches"] / t["acquired"] if t and t["acquired"] else 0.0


# --- payback ---------------------------------------------------------------

def incremental(s, c, field="profitCents"):
    return [k(x - y) for x, y in zip(s["daily"][field], c["daily"][field])]


def sustained_breakeven(inc):
    """First day from which the cumulative increment stays >= 0 through
    the last day, or None."""
    if not inc or inc[-1] < 0:
        return None
    d = len(inc)
    while d > 0 and inc[d - 1] >= 0:
        d -= 1
    return d + 1


# --- report ------------------------------------------------------------------

def main():
    set_ = sys.argv[1] if len(sys.argv) > 1 else "eval"
    rs = load(set_)
    if CONTROL not in rs:
        sys.exit(f"no {CONTROL} results in {set_}")
    ctl = rs[CONTROL]
    seeds = ctl["seeds"]
    plans = [CONTROL] + sorted(p for p in rs if p != CONTROL)
    by = {p: {s["seed"]: s for s in rs[p]["perSeed"]} for p in plans}
    for p in plans:
        if rs[p]["seeds"] != seeds or rs[p]["scenarioFingerprint"] != ctl["scenarioFingerprint"]:
            sys.exit(f"{p}: seeds or scenario differ from the control")
    L = []
    w = L.append
    w(f"# Dynamic staffing benchmark — `{set_}` set\n")
    w("Generated by `report.py` from `results/%s/*.json`; do not edit by hand. Money in $k; "
      "± is a 95%% paired t half-width across seeds; ranges are min to max.\n" % set_)
    w("## Provenance\n")
    w(f"* Engine `{ctl['engine']}`; starting practice `{ctl['scenarioManifest']}`")
    w(f"* Scenario fingerprint `{ctl['scenarioFingerprint']}` (base `{ctl['baseScenarioFingerprint']}`, overlay SHA-256 `{ctl['overlaySHA256']}`)")
    w(f"* Seeds {seeds[0]}–{seeds[-1]} ({len(seeds)}); {ctl['days']} days from the takeover")
    for p in plans:
        w(f"* `{p}`: plan SHA-256 `{rs[p]['planSHA256'][:16]}…` — {rs[p]['plan']['description']}")
    w("")

    # Acceptance and lifecycle.
    w("## Action acceptance and lifecycle\n")
    w("| plan | actions | accepted on every seed | lifecycle as planned | max technologists | displaced (rebooked / lost) | inconsistent bookings | cash bridge reconciled | labor postings reconcile |")
    w("|---|---|---|---|---|---|---|---|---|")
    for p in plans:
        ss = list(by[p].values())
        n_act = len(ss[0]["actions"] or [])
        disp = [s["extra"]["T180"]["displacement"] for s in ss]
        w(f"| {p} | {n_act} | {all(s['checks']['allAccepted'] for s in ss)} | {all(s['checks']['lifecycleMatches'] for s in ss)} | "
          f"{max(s['checks']['maxTechnologists'] for s in ss)} | {sum(d.get('events', 0) for d in disp)} "
          f"({sum(d.get('rebooked', 0) for d in disp)} / {sum(d.get('noFeasibleSlot', 0) + d.get('declined', 0) for d in disp)}) | "
          f"{sum(s['checks']['inconsistentBookings'] for s in ss)} | {all(s['checks']['reconciled'] for s in ss)} | "
          f"{all(s['checks']['laborReconciles'] for s in ss)} |")
    w("")
    w("Lifecycle (identical on every seed; first seed shown):\n")
    w("| plan | employee | start | productive | last day | contract h |")
    w("|---|---|---|---|---|---|")
    for p in plans:
        s0 = by[p][seeds[0]]
        for e in s0["staff"]:
            if e.get("requestId") or e.get("lastDay"):
                who = e["id"] + (f" ({e['requestId']})" if e.get("requestId") else f" ({e['name']})")
                w(f"| {p} | {who} | {e.get('start', '')} | {e.get('productive', '')} | {e.get('lastDay', '')} | {e.get('contractMin', 0) / 60:g} |")
    w("")

    # Finance.
    w("## Finance by span\n")
    w("Operating profit is the evaluation window's baseline-adjusted accrual profit (net revenue less "
      "operating expense, recruiting and severance included).\n")
    w("| plan | span | net revenue | operating expense | operating profit | margin % | Δ profit vs no action | seeds with Δ > 0 |")
    w("|---|---|---|---|---|---|---|---|")
    for p in plans:
        for name, a, b in SPANS:
            f = [fin(by[p][sd], a, b) for sd in seeds]
            fc = [fin(by[CONTROL][sd], a, b) for sd in seeds]
            dl = [x["profit"] - y["profit"] for x, y in zip(f, fc)]
            pos = sum(1 for x in dl if x > 0)
            w(f"| {p} | {name} | {rng([x['netRevenue'] for x in f])} | {rng([x['opex'] for x in f])} | "
              f"{rng([x['profit'] for x in f])} | {rng([x['margin'] for x in f])} | "
              f"{'—' if p == CONTROL else cis(dl)} | {'—' if p == CONTROL else f'{pos}/{len(seeds)}'} |")
    w("")

    # Transition and labor costs.
    w("## Transition costs and contractual obligations (cumulative, $k)\n")
    w("Training: a hire's worked pay with benefits before its productive date. Overlap: a departing "
      "employee's worked pay on days its designated replacement is already productive. Guarantee: pay "
      "for contracted time not worked (all hourly staff). Severance includes benefits and is "
      "recognized when the termination is accepted.\n")
    w("| plan | endpoint | recruiting | severance | training | overlap | guarantee | transition total | labor (wages+benefits) Δ vs no action |")
    w("|---|---|---|---|---|---|---|---|---|")
    for p in plans:
        for d in [30, 90, 180]:
            lab = [by[p][sd]["labor"][f"T{d}"] for sd in seeds]
            labc = [by[CONTROL][sd]["labor"][f"T{d}"] for sd in seeds]
            tot = [k(x["recruitingCents"] + x["severanceCents"] + x["trainingCents"] + x["overlapCents"]) for x in lab]
            wages = [k(x["workedCents"] + x["guaranteeCents"] + x["salaryCents"]) - k(y["workedCents"] + y["guaranteeCents"] + y["salaryCents"])
                     for x, y in zip(lab, labc)]
            w(f"| {p} | T{d} | {fmt(mean([k(x['recruitingCents']) for x in lab]))} | {fmt(mean([k(x['severanceCents']) for x in lab]))} | "
              f"{fmt(mean([k(x['trainingCents']) for x in lab]))} | {fmt(mean([k(x['overlapCents']) for x in lab]))} | "
              f"{fmt(mean([k(x['guaranteeCents']) for x in lab]), 2)} | {fmt(mean(tot))} | {'—' if p == CONTROL else cis(wages)} |")
    w("")

    # Cash.
    w("## Cash and obligations\n")
    w("Cash settles later than accrual: receivables grow with volume, payroll and payables are paid in "
      "arrears, severance on the first payday after the last day. Liabilities: payables + accrued payroll "
      "+ severance payable.\n")
    w("| plan | lowest cash (day) | cash T30 / T90 / T180 | Δ cash vs no action T30 / T90 / T180 | receivables T180 | liabilities T30 / T90 / T180 | severance payable T30 / T90 / T180 |")
    w("|---|---|---|---|---|---|---|")
    for p in plans:
        ss = [by[p][sd] for sd in seeds]
        cs = [by[CONTROL][sd] for sd in seeds]
        mins = [min(s["daily"]["cashCents"]) for s in ss]
        mday = [s["daily"]["cashCents"].index(min(s["daily"]["cashCents"])) + 1 for s in ss]
        cash = lambda s, d: k(s["daily"]["cashCents"][d - 1])
        liab = lambda s, d: k(s["extra"][f"T{d}"]["liabilitiesCents"])
        sev = lambda s, d: k(s["extra"][f"T{d}"]["severancePayableCents"])
        dc = " / ".join(fmt(mean([cash(x, d) - cash(y, d) for x, y in zip(ss, cs)])) for d in (30, 90, 180))
        w(f"| {p} | {fmt(k(min(mins)))} min, {fmt(mean([k(x) for x in mins]))} mean (day {min(mday)}–{max(mday)}) | "
          f"{' / '.join(fmt(mean([cash(s, d) for s in ss])) for d in (30, 90, 180))} | {'—' if p == CONTROL else dc} | "
          f"{fmt(mean([k(s['daily']['arCents'][-1]) for s in ss]))} | "
          f"{' / '.join(fmt(mean([liab(s, d) for s in ss])) for d in (30, 90, 180))} | "
          f"{' / '.join(fmt(mean([sev(s, d) for s in ss])) for d in (30, 90, 180))} |")
    w("")

    # Payback.
    w("## Payback against the paired no-action run\n")
    w("Cumulative incremental operating profit (accrual) and cash, day by day. Break-even is the first "
      "day from which the cumulative increment stays at or above zero through day 180. Steady-state "
      "improvement: the increment's change over days 91–180, per 30 days. Extrapolated break-even "
      "(labelled) assumes that steady-state rate continues past day 180.\n")
    w("| plan | Δ T30 | Δ T90 | Δ T180 | steady state Δ / 30 d (D91–180) | break-even day (accrual): median, range | paid back by T90 / T180 | never by T180: extrapolated day | cash break-even: median | cash paid back by T180 |")
    w("|---|---|---|---|---|---|---|---|---|---|")
    traj = {}
    for p in plans:
        if p == CONTROL:
            continue
        incs = [incremental(by[p][sd], by[CONTROL][sd]) for sd in seeds]
        cinc = [incremental(by[p][sd], by[CONTROL][sd], "cashCents") for sd in seeds]
        traj[p] = (incs, cinc)
        be = [sustained_breakeven(x) for x in incs]
        cbe = [sustained_breakeven(x) for x in cinc]
        ss = [(x[179] - x[89]) / 90 * 30 for x in incs]
        never = []
        for x, b in zip(incs, be):
            if b is None:
                rate = (x[179] - x[89]) / 90
                never.append(f"{180 + math.ceil(-x[179] / rate)}" if rate > 0 else "no (declining)")
        okb = sorted(b for b in be if b is not None)
        okc = sorted(b for b in cbe if b is not None)
        med = lambda xs: xs[len(xs) // 2] if xs else None
        w(f"| {p} | {cis([x[29] for x in incs])} | {cis([x[89] for x in incs])} | {cis([x[179] for x in incs])} | {cis(ss)} | "
          f"{med(okb) if okb else 'none'}{f' ({okb[0]}–{okb[-1]})' if okb else ''} | "
          f"{sum(1 for b in be if b is not None and b <= 90)} / {sum(1 for b in be if b is not None)} of {len(seeds)} | "
          f"{', '.join(never) if never else '—'} | {med(okc) if okc else 'none'} | {len(okc)} of {len(seeds)} |")
    w("")
    w("Mean cumulative increment ($k) by day (accrual / cash):\n")
    days = [7, 14, 21, 28, 35, 42, 49, 56, 63, 70, 77, 84, 90, 105, 120, 135, 150, 165, 180]
    w("| day | " + " | ".join(p for p in traj) + " |")
    w("|---|" + "---|" * len(traj))
    for d in days:
        w(f"| {d} | " + " | ".join(f"{fmt(mean([x[d - 1] for x in traj[p][0]]))} / {fmt(mean([x[d - 1] for x in traj[p][1]]))}" for p in traj) + " |")
    w("")
    w("Per-seed cumulative increment at T30 / T90 / T180 and break-even day:\n")
    w("| seed | " + " | ".join(p for p in traj) + " |")
    w("|---|" + "---|" * len(traj))
    for i, sd in enumerate(seeds):
        cells = []
        for p in traj:
            x = traj[p][0][i]
            b = sustained_breakeven(x)
            cells.append(f"{fmt(x[29])} / {fmt(x[89])} / {fmt(x[179])} · {b if b else 'never'}")
        w(f"| {sd} | " + " | ".join(cells) + " |")
    w("")

    # Operations and service.
    w("## Imaging, reporting and access by span\n")
    w("Means across seeds. Exams and final reports are counts in the span; lost referrals are orders "
      "received in the span and lost; breach rate is breaches ÷ studies acquired in the span, as of its "
      "end; backlog is unread + reading at the endpoint.\n")
    w("| plan | span | exams XR / US / CT / MR | reports XR / US / CT / MR | lost referrals XR / US / CT / MR | walk-ins left | displaced (rebooked) |")
    w("|---|---|---|---|---|---|---|")
    ops = {p: {name: [span_ops(by[p][sd], a, b) for sd in seeds] for name, a, b in SPANS} for p in plans}
    for p in plans:
        for name, a, b in SPANS[:3]:
            o = ops[p][name]
            j = lambda key: " / ".join(fmt(mean([x[key].get(m, 0) for x in o]), 0) for m in MODS)
            w(f"| {p} | {name} | {j('exams')} | {j('reports')} | {j('lost')} | {fmt(mean([x['walkInsLeft'] for x in o]), 0)} | "
              f"{fmt(mean([x['displaced'] for x in o]), 1)} ({fmt(mean([x['rebooked'] for x in o]), 1)}) |")
    w("")
    w("| plan | span | breach rate urgent / soon / routine % | urgent p90 min | soon p90 min | routine p90 min | backlog at end (oldest min) | radiologist read/block % | technologist busy/duty % |")
    w("|---|---|---|---|---|---|---|---|---|")
    for p in plans:
        for name, a, b in SPANS[:3]:
            o = ops[p][name]
            br = " / ".join(fmt(mean([breach_rate(x, q) for x in o])) for q in PRIS)
            p90 = lambda q: fmt(mean([x["tat"].get(q, {}).get("p90Min", 0) for x in o]), 0)
            ex = [by[p][sd]["extra"][f"T{b}"] for sd in seeds]
            w(f"| {p} | {name} | {br} | {p90('urgent')} | {p90('soon')} | {p90('routine')} | "
              f"{fmt(mean([x['backlog'] for x in ex]), 0)} ({fmt(mean([x['backlogOldestMin'] for x in ex]), 0)}) | "
              f"{fmt(mean([x['readUtil'] for x in o]))} | {fmt(mean([x['techUtil'] for x in o]))} |")
    w("")
    w("Paired service differences against no action (D91–180, mean ± 95% CI):\n")
    w("| plan | Δ exams | Δ final reports | Δ lost referrals | Δ walk-ins left | Δ urgent breach rate pts | Δ soon breach rate pts | Δ routine breach rate pts | Δ backlog at T180 |")
    w("|---|---|---|---|---|---|---|---|---|")
    for p in plans:
        if p == CONTROL:
            continue
        o, c = ops[p]["D91-180"], ops[CONTROL]["D91-180"]
        dd = lambda f: cis([f(x) - f(y) for x, y in zip(o, c)], 1)
        bl = cis([by[p][sd]["extra"]["T180"]["backlog"] - by[CONTROL][sd]["extra"]["T180"]["backlog"] for sd in seeds])
        w(f"| {p} | {dd(lambda x: sum(x['exams'].values()))} | {dd(lambda x: sum(x['reports'].values()))} | "
          f"{dd(lambda x: sum(x['lost'].values()))} | {dd(lambda x: x['walkInsLeft'])} | "
          f"{dd(lambda x: breach_rate(x, 'urgent'))} | {dd(lambda x: breach_rate(x, 'soon'))} | {dd(lambda x: breach_rate(x, 'routine'))} | {bl} |")
    w("")
    w("Lost referrals by primary cause, D91–180 (mean per seed; booking-loss probe):\n")
    causes = ["shared_tech", "not_rostered", "category_full", "template_category", "no_template_slot", "notice", "no_supervision"]
    w("| plan | modality | " + " | ".join(causes) + " |")
    w("|---|---|" + "---|" * len(causes))
    for p in plans:
        o = ops[p]["D91-180"]
        for m in MODS:
            w(f"| {p} | {m} | " + " | ".join(fmt(mean([x["lossPrimary"].get(m, {}).get(cz, 0) for x in o]), 1) for cz in causes) + " |")
    w("")
    w("Booking lead (received → appointment day), D91–180, p50 / p90 days, and room busy ÷ staffed %:\n")
    w("| plan | " + " | ".join(f"{m} lead" for m in MODS) + " | " + " | ".join(f"{m} busy/staffed" for m in MODS) + " | staffed h XR / CT |")
    w("|---|" + "---|" * (2 * len(MODS) + 1))
    for p in plans:
        o = ops[p]["D91-180"]
        w(f"| {p} | " + " | ".join(f"{fmt(mean([x['leadP50'].get(m, 0) for x in o]), 0)} / {fmt(mean([x['leadP90'].get(m, 0) for x in o]), 0)}" for m in MODS)
          + " | " + " | ".join(fmt(mean([x["utilStaffed"][m] for x in o])) for m in MODS)
          + f" | {fmt(mean([x['staffedH']['XR'] for x in o]), 0)} / {fmt(mean([x['staffedH']['CT'] for x in o]), 0)} |")
    w("")

    if set_ == "dev":
        w("## Partial-plan selection (development seeds)\n")
        w("Rule (plans.py, fixed before any run): the B candidate with the highest mean T180 cumulative "
          "profit over no action, among those whose D91–180 urgent breach rate is at most 5 points above "
          "no action's; ties within $1k go to fewer actions.\n")
        w("| candidate | actions | mean Δ T180 $k | Δ urgent breach pts D91–180 | eligible |")
        w("|---|---|---|---|---|")
        best = None
        for p in plans:
            if not p.startswith("B"):
                continue
            inc = mean([x[179] for x in traj[p][0]])
            db = mean([breach_rate(x, "urgent") - breach_rate(y, "urgent") for x, y in zip(ops[p]["D91-180"], ops[CONTROL]["D91-180"])])
            n_act = len(by[p][seeds[0]]["actions"] or [])
            ok = db <= 5
            w(f"| {p} | {n_act} | {fmt(inc)} | {fmt(db)} | {ok} |")
            if ok and (best is None or inc > best[1] + 1 or (abs(inc - best[1]) <= 1 and n_act < best[2])):
                best = (p, inc, n_act)
        w(f"\n**Selected: `{best[0] if best else 'none'}`.**\n")

    out = os.path.join(HERE, "results", set_)
    with open(os.path.join(out, "report.md"), "w") as f:
        f.write("\n".join(L) + "\n")
    with open(os.path.join(out, "trajectory.csv"), "w", newline="") as f:
        cw = csv.writer(f)
        cw.writerow(["plan", "seed", "day", "incremental_profit_k", "incremental_cash_k", "profit_k", "cash_k"])
        for p in traj:
            for i, sd in enumerate(seeds):
                s = by[p][sd]
                for d in range(len(traj[p][0][i])):
                    cw.writerow([p, sd, d + 1, f"{traj[p][0][i][d]:.2f}", f"{traj[p][1][i][d]:.2f}",
                                 f"{k(s['daily']['profitCents'][d]):.2f}", f"{k(s['daily']['cashCents'][d]):.2f}"])


if __name__ == "__main__":
    main()
