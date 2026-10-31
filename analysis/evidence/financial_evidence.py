#!/usr/bin/env python3
"""Financial decision evidence (dreamteam.financial-evidence/v1): what a
recorded management decision committed the practice to, how far it can be
undone under the simulator's rules, and which simulator-controlled
resources it changed directly.

    python3 -I -B analysis/evidence/financial_evidence.py --binary ./dt \
        --corpus docs/benchmark/v09b-dev/corpus.json --replays <dir> --out <dir>

Three dimensions, kept apart and never combined into a score:
* AFFORDABILITY, one record per submitted batch: the engine's own funding
  test (V0.7b: funding headroom = cash + undrawn credit - current
  liabilities - reserved commitments; receivables and profit are never
  counted) and the commitment it measures, with cash, credit and the
  recurring and immediate money lines the engine states;
* REVERSIBILITY, one record per executed action: the simulator's own
  rules for undoing it (effective date, notice, the inverse action if one
  exists, what stays owed after it), with the engine's exact amounts;
* OPERATIONAL_EXPOSURE, one record per executed action: the direct state
  change (A) the engine states for it, kept apart from what was observed
  later in the run (B) and from the run-level comparison with no action
  (C). Nothing observed later is attributed to one action.

Where the numbers come from. Every value is the engine's. The replay's own
protocol transcript is fed back to the same engine (`dreamteam agent
--stdio` on the run's scenario and seed) with one read-only `preview`
inserted before each staffing batch's `validate`; every original answer
must come back byte-identical, or the analysis stops. The preview carries
what the game's staff screen shows: the funding test, guaranteed and
forecast money lines, hires, terminations, roster and allocation changes,
teleradiology and slots. An agent could have asked for the same preview at
the same moment (it is a permitted operation), so it is evidence of what
was knowable at the decision, not of what the agent read.

Where the engine gives no exact value, the record says so: a shift
extension's commitment is not reported exactly (its message gives an
estimate without benefits), so its credit dependence is settled only when
the observed headroom decides it. A rule the simulator does not have is
never assumed: reversibility the engine cannot establish is `unknown`.

Separate from the frozen evidence rules (decision evidence v1, repeat v2,
refusal-response v1 and v2, parameter evidence v1), which are unchanged.
No model is contacted.
"""

import argparse
import concurrent.futures
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(HERE, "..", "..", "agents", "llm"))
import decision_evidence as D  # noqa: E402  (corpus helpers, read-only)
import financial_outcomes as FO  # noqa: E402  (paired-run helpers, read-only)
import llm_agent as A  # noqa: E402  (the agent's batching, read-only)

SCHEMA = "dreamteam.financial-evidence/v1"
CORPUS_SCHEMA = "dreamteam.financial-evidence-corpus/v1"
ANALYZER = "financial-evidence 1"
RULES = {"AFFORDABILITY": 1, "REVERSIBILITY": 1, "OPERATIONAL_EXPOSURE": 1}
V3_SCHEMA = "dreamteam.llm-decision-record/v3"
STAFFING = tuple(A.STAFFING)

AFFORDABILITY = ("funded_without_credit", "funded_with_credit", "refused_insufficient_funds", "no_new_commitment",
                 "not_tested", "evidence_unavailable")
REVERSIBILITY = ("withdrawable", "reversible_with_retained_obligation", "not_reversible",
                 "reversible_no_recorded_cost", "unknown")
EXPOSURE = ("direct_change", "no_direct_change", "evidence_unavailable")

# The simulator's rules for undoing each action (internal/sim; the codes
# and limits are the engine's own). Per-record amounts come from the
# engine's preview; this table only names the rule.
MATRIX = {
    "hire": {
        "class": "reversible_with_retained_obligation",
        "effective": "the start date (at least the offer's lead time ahead)",
        "notice": "a termination's last day is at least 14 days after both today and the start date",
        "inverse": "terminate (also before the start date)",
        "retained": "the recruiting fee (recognized at acceptance, not refunded), the contract guarantee to the last "
                    "day, and two contracted weeks' severance with benefits",
        "reversalCost": "the preview's \"Earliest lawful exit\" line: the guarantee from the start to the earliest "
                        "last day plus severance; the recruiting fee in addition",
        "source": "internal/sim/staffing.go, offers.go (NoticeDays = 14)"},
    "terminate": {
        "class": "not_reversible",
        "effective": "employment ends at the midnight after the last day",
        "notice": "at least 14 days; contract pay continues to the last day",
        "inverse": "none: a termination cannot be moved or withdrawn (already_terminating); capacity returns only "
                   "through a new hire, with its own fee and lead time",
        "retained": "severance (two contracted weeks, recognized at acceptance) and contract pay to the last day",
        "reversalCost": "not applicable",
        "source": "internal/sim/staffing.go"},
    "set_reading_allocation": {
        "class": "withdrawable",
        "effective": "a dated version, at least 14 days ahead",
        "notice": "14 days; once in force an allocation stays in force at least 14 days",
        "inverse": "op remove of the pending version, until the preview's withdrawBy date (the same notice); "
                   "after that, a new set with 14 days' notice",
        "retained": "nothing while withdrawable; once in force, the salary accrued under it",
        "reversalCost": "none recorded for a withdrawal",
        "source": "internal/sim/allocation.go (NoticeDays = 14, MinCommitDays = 14)"},
    "set_roster": {
        "class": "withdrawable",
        "effective": "a Monday at least 14 days ahead",
        "notice": "14 days, also to remove a pending version",
        "inverse": "op remove of the pending version up to 14 days before it takes effect; a version in effect "
                   "or past never changes",
        "retained": "the contract guarantee is owed whatever is rostered: no roster change saves or costs pay",
        "reversalCost": "none recorded",
        "source": "internal/sim/roster.go"},
    "contract_teleradiology:start": {
        "class": "reversible_with_retained_obligation",
        "effective": "the activation, at least 14 days ahead",
        "notice": "a cancellation ends at least 30 days ahead and at least 30 days after activation",
        "inverse": "contract_teleradiology op cancel",
        "retained": "the retainer to the minimum-term end, payable even if cancelled before activation",
        "reversalCost": "the preview's guaranteed retainer (GuaranteedCents through GuaranteedThrough)",
        "source": "internal/sim/telecontract.go"},
    "contract_teleradiology:cancel": {
        "class": "not_reversible",
        "effective": "the exclusive end date",
        "notice": "at least 30 days",
        "inverse": "none: a fixed end cannot be moved or withdrawn (already_cancelled); coverage returns only "
                   "through a new contract, with 14 days' notice",
        "retained": "the retainer to the end; claims already made complete at their fees",
        "reversalCost": "not applicable",
        "source": "internal/sim/telecontract.go"},
    "extend_shift": {
        "class": "not_reversible",
        "effective": "the extension's date",
        "notice": "none",
        "inverse": "none: no action withdraws an approved extension; a later roster change or termination can "
                   "void it",
        "retained": "the extra time is paid as worked",
        "reversalCost": "not applicable",
        "source": "internal/sim/actions.go"},
    "set_walkin_policy": {
        "class": "reversible_no_recorded_cost",
        "effective": "at once", "notice": "none", "inverse": "set_walkin_policy again, at once",
        "retained": "no money line", "reversalCost": "none recorded", "source": "internal/sim/actions.go"},
    "set_worklist_policy": {
        "class": "reversible_no_recorded_cost",
        "effective": "at once", "notice": "none", "inverse": "set_worklist_policy again, at once",
        "retained": "no money line", "reversalCost": "none recorded", "source": "internal/sim/actions.go"},
}
REMOVE_OPS = {"set_reading_allocation", "set_roster"}


# --- identities -------------------------------------------------------------------------

def sha256_bytes(b):
    return hashlib.sha256(b).hexdigest()


def stable_id(*parts):
    return "fe1-" + sha256_bytes("|".join(str(p) for p in parts).encode("utf-8"))[:16]


def canon(x):
    return json.dumps(x, sort_keys=True, separators=(",", ":"))


def matrix_key(a):
    t = a.get("type")
    if t == "contract_teleradiology":
        return t + ":" + str(a.get("op"))
    return t


# --- the engine replay ------------------------------------------------------------------

class ReplayMismatch(Exception):
    pass


def staffing_batch(acts):
    return bool(acts) and all(isinstance(a, dict) and a.get("type") in STAFFING for a in acts)


def engine_replay(binary, run_dir, run, digests, extras=None):
    """Feed the run's transcript back to the engine and return ({point
    index: segment}, requests). Read-only requests are inserted, and every
    original answer must come back identical:
    * a preview before each staffing batch's validate, and before each
      staffing apply that no validate of the same batch preceded (the
      llm-agent 2 replays apply the recorded batches without validating);
    * an observe after each applied batch (the reservations it added);
    * extras {point index: [batch]}: recorded proposals the transcript
      does not submit (an llm-agent 2 final reply that was refused), each
      validated, and previewed if staffing, at its decision point."""
    extras = extras or {}
    pairs, req = [], None
    with open(os.path.join(run_dir, "transcript.jsonl"), encoding="utf-8") as f:
        for line in f:
            d = json.loads(line)
            if "op" in d:
                req = d
            elif req is not None:
                pairs.append((req, d))
                req = None
    # The transcript ends with the run's own save: the engine writes it into
    # a scratch directory of its own, never into the caller's.
    scratch = tempfile.TemporaryDirectory(prefix="fe-replay-")
    proc = subprocess.Popen([os.path.abspath(binary), "agent", "--stdio", "-scenario", run["scenario"]["ref"],
                             "-seed", str(run["seed"])], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL, text=True, encoding="utf-8", cwd=scratch.name)

    def ask(r):
        proc.stdin.write(json.dumps(r) + "\n")
        proc.stdin.flush()
        line = proc.stdout.readline()
        if not line:
            raise ReplayMismatch("the engine closed the session")
        return json.loads(line)

    def preview(acts):
        pv = ask({"id": "fe-preview", "op": "preview", "args": {"actions": acts}})
        return ((pv.get("result") or {}).get("preview") if pv.get("ok") else None,
                None if pv.get("ok") else (pv.get("error") or {}).get("code"))

    segs, cur, ci, i, n = {}, None, None, 0, 0
    try:
        for r, want in pairs:
            op = r.get("op")
            if cur is not None and op in ("advance", "save") and not cur.get("extrasDone"):
                cur["extrasDone"] = True
                for acts in extras.get(ci) or []:
                    ev = {"op": "validate", "actions": acts, "inserted": True}
                    if staffing_batch(acts):
                        ev["preview"], ev["previewError"] = preview(acts)
                    v = ask({"id": "fe-validate", "op": "validate", "args": {"actions": acts}})
                    ev["answer"] = v.get("result") if v.get("ok") else None
                    ev["protocolError"] = None if v.get("ok") else (v.get("error") or {}).get("code")
                    cur["events"].append(ev)
            acts = (r.get("args") or {}).get("actions") or []
            pv = None
            if want.get("ok") and staffing_batch(acts) and (
                    op == "validate" or (op == "apply" and cur is not None and canon(acts) not in cur["validated"])):
                pv = preview(acts)
            got = ask(r)
            n += 1
            after = None
            if op == "apply" and want.get("ok") and (want.get("result") or {}).get("applied"):
                after = ask({"id": "fe-observe", "op": "observe", "args": {"compact": True, "recentLog": 0}})
            if op == "hello":
                same = ((got.get("result") or {}).get("clock") == (want.get("result") or {}).get("clock") and
                        ((got.get("result") or {}).get("scenario") or {}).get("fingerprint") ==
                        ((want.get("result") or {}).get("scenario") or {}).get("fingerprint"))
            else:
                same = got == want
            if not same:
                raise ReplayMismatch(f"request {n} ({op}) answered differently")
            if not want.get("ok"):
                continue
            res = want.get("result") or {}
            if op == "digests":
                cur, ci = None, None
                if i < len(digests) and res.get("digests") == digests[i]:
                    cur = segs.setdefault(i, {"events": [], "validated": set()})
                    ci = i
                    i += 1
            elif cur is None:
                continue
            elif op == "observe" and "obs" not in cur:
                cur["obs"] = res.get("observation")
            elif op == "actions" and "catalog" not in cur:
                cur["catalog"] = res
            elif op in ("validate", "apply"):
                ev = {"op": op, "actions": acts, "answer": res}
                if op == "validate":
                    cur["validated"].add(canon(acts))
                if after is not None:
                    ev["fundingAfter"] = funding_obs((after.get("result") or {}).get("observation")) \
                        if after.get("ok") else None
                if pv is not None:
                    ev["preview"], ev["previewError"] = pv
                cur["events"].append(ev)
    finally:
        proc.stdin.close()
        proc.wait()
        scratch.cleanup()
    for seg in segs.values():
        seg.pop("validated", None)
        seg.pop("extrasDone", None)
    return segs, n


def v2_extras(recs, transcript_applies=None):
    """{point index: [batch]}: an llm-agent 2 final reply's batches, to be
    validated again at their point; the batches the transcript applies are
    left to the transcript."""
    out = {}
    for i, rec in enumerate(recs):
        if rec.get("schema") == V3_SCHEMA:
            continue
        acts = rec.get("proposed")
        if not isinstance(acts, list) or not acts or not all(isinstance(a, dict) and a.get("type") for a in acts):
            continue
        applied = {canon(b) for b in rec.get("batches") or []}
        todo = [b for b, _ in A.split(acts) if canon(b) not in applied]
        if todo:
            out[i] = todo
    return out


# --- the observation ----------------------------------------------------------------------

def funding_obs(obs):
    """The public observation of funding at a decision point."""
    if not obs:
        return None
    f = obs.get("finance") or {}
    fin = f.get("financing") or {}
    s = fin.get("solvency")
    out = {"cashCents": f.get("cashCents"), "receivablesCents": f.get("arCents"),
           "creditLineModeled": fin.get("creditLineModeled")}
    if s:
        for k in ("currentLiabilitiesCents", "reservedCents", "fundingHeadroomCents", "creditLimitCents",
                  "creditDrawnCents", "creditAvailableCents"):
            out[k] = s.get(k)
        out["headroomWithoutCreditCents"] = s["fundingHeadroomCents"] - s["creditAvailableCents"]
    return out


def staff_of(obs):
    return {s.get("id"): s for s in (obs or {}).get("staff") or []}


def observed_state(obs):
    """The run's observed state at a decision point (B: observed, never
    attributed)."""
    f = funding_obs(obs) or {}
    heads = {}
    for s in (obs or {}).get("staff") or []:
        if s.get("employment") in (None, "terminated"):
            continue
        k = f"{s.get('role')}:{s.get('employment')}"
        heads[k] = heads.get(k, 0) + 1
    lost = sum(((m.get("sinceStart") or {}).get("lost") or 0) for m in ((obs or {}).get("demand") or {})
               .get("modalities") or [])
    flow = (((obs or {}).get("flow") or {}).get("sinceStart") or {})
    return {"time": (obs or {}).get("time"), "cashCents": f.get("cashCents"),
            "fundingHeadroomCents": f.get("fundingHeadroomCents"), "creditDrawnCents": f.get("creditDrawnCents"),
            "reservedCents": f.get("reservedCents"), "headcount": dict(sorted(heads.items())),
            "unreadStudies": ((obs or {}).get("worklist") or {}).get("unread"),
            "ordersLostSinceStart": lost, "examsDoneSinceStart": flow.get("examsDone")}


# --- money lines ----------------------------------------------------------------------------

def lines_of(pv):
    return [{"label": m.get("label"), "cents": m.get("cents"), "when": m.get("when"), "basis": m.get("basis")}
            for m in (pv or {}).get("guaranteed") or []]


def recurring(lines):
    """Lines the engine states per period. A line it marks unchanged (a
    roster batch restates the existing contract guarantee) is not an
    obligation the batch creates."""
    return [m for m in lines if "unchanged" not in (m.get("when") or "") + (m.get("basis") or "") and
            (re.search(r"\bper (?:full week|week|30 days|month)\b", m.get("when") or "") or
             re.search(r"\bper (?:full week|week)\b", m.get("label") or ""))]


def immediate(lines):
    return [m for m in lines if "recognized now" in (m.get("when") or "")]


EXT_PAID = re.compile(r"approved overtime · ≈\$(?P<d>[\d,]+)")


# --- records ----------------------------------------------------------------------------------

class Analysis:
    def __init__(self):
        self.records = []

    def add(self, ctx, dim, status, seq, actions, fields):
        point = ctx["point"]
        rid = stable_id(ctx["recording"]["path"], point["index"], ctx["eventIndex"],
                        ",".join(str(a["index"]) for a in actions), dim, RULES[dim])
        rc = ctx["recording"]
        rec = {"id": rid, "dimension": dim, "rule": {"id": dim, "version": RULES[dim]}, "status": status,
               "recording": {"path": rc["path"], "group": rc["group"]},
               "run": {"scenario": ctx["run"]["scenario"], "seed": ctx["run"]["seed"]}, "decision": point,
               "attempt": ctx["attempt"], "submission": seq, "actions": actions}
        rec.update(fields)
        self.records.append(rec)
        return rec


def action_refs(acts, res, attempt_idx=None):
    out = []
    for k, a in enumerate(acts):
        r = res[k] if k < len(res) else {}
        ref = {"index": k, "attemptIndex": attempt_idx[k] if attempt_idx and k < len(attempt_idx) else None,
               "type": a.get("type"), "op": a.get("op"),
               "employee": a.get("employee"), "requestId": a.get("requestId"), "contract": a.get("contract"),
               "date": a.get("date") or a.get("effective") or a.get("startDate") or a.get("lastDay") or
               a.get("endDate"),
               "validation": {"ok": bool(r.get("ok")), "code": r.get("code")}}
        out.append({k2: v for k2, v in ref.items() if v is not None or k2 == "attemptIndex"})
    return out


def attempts_for(rec, events):
    """Link each validate event at a point to the recorded attempt that
    sent it: {event index: (attempt id, indices in the reply)}."""
    vals = [k for k, e in enumerate(events) if e["op"] == "validate"]
    out = {}
    if rec.get("schema") == V3_SCHEMA:
        sent = []
        for at in rec.get("attempts") or []:
            for b in at.get("validation") or []:
                sent.append((at.get("id"), b.get("actions"), b.get("indices")))
        j = 0
        for k in vals:
            while j < len(sent) and canon(sent[j][1]) != canon(events[k]["actions"]):
                j += 1
            if j < len(sent):
                out[k] = (sent[j][0], sent[j][2])
                j += 1
        return out
    acts = rec.get("proposed") if isinstance(rec.get("proposed"), list) else []
    try:
        batches = A.split(acts) if acts and all(isinstance(a, dict) and a.get("type") for a in acts) else []
    except Exception:  # noqa: BLE001 - a malformed reply: no attempt link
        batches = []
    subs = [k for k, e in enumerate(events) if e.get("submission")]
    for b, idx in batches:
        hit = [k for k in subs if canon(events[k]["actions"]) == canon(b)]
        if hit:
            out[hit[-1]] = ("final", idx)
    return out


def unrecorded_attempts(recs):
    """llm-agent 2 records the final reply only: the number of earlier
    attempts, whose actions and reasons are not recorded, and the refusal
    codes it did record for them."""
    n, codes = 0, {}
    for r in recs:
        if r.get("schema") == V3_SCHEMA or not isinstance(r.get("attempts"), int):
            continue
        n += max(0, r["attempts"] - 1)
        for x in r.get("refusals") or []:
            if x.get("attempt", 0) < r["attempts"]:
                codes[x.get("code")] = codes.get(x.get("code"), 0) + 1
    return n, dict(sorted(codes.items()))


def recorded_validation(rec):
    """An llm-agent 2 record's own validate answers, by batch."""
    if rec.get("schema") == V3_SCHEMA:
        return {}
    acts = rec.get("proposed") if isinstance(rec.get("proposed"), list) else []
    answers = rec.get("validation") if isinstance(rec.get("validation"), list) else []
    if not acts or not all(isinstance(a, dict) and a.get("type") for a in acts):
        return {}
    batches = A.split(acts)
    if len(batches) != len(answers):
        return {}
    return {canon(b): ans for (b, _), ans in zip(batches, answers)}


def state_before(events, k, fo):
    """The funding state just before event k: the engine's observation after
    the last batch applied before it at this point, else the decision
    point's observation."""
    for e in reversed(events[:k]):
        if e["op"] == "apply" and e["answer"].get("applied") and e.get("fundingAfter"):
            return e["fundingAfter"]
    return fo


def applied_index(events, k):
    """The index of the apply event of submission k's batch at the same
    point (k itself for an apply), if the batch was applied."""
    if events[k]["op"] == "apply":
        return k if events[k]["answer"].get("applied") else None
    for j in range(k + 1, len(events)):
        e = events[j]
        if e["op"] == "apply" and canon(e["actions"]) == canon(events[k]["actions"]):
            return j
    return None


def applied_after(events, k):
    """The apply answer of submission k's batch at the same point, if the
    batch was applied."""
    if events[k]["op"] == "apply":
        return events[k]
    for e in events[k + 1:]:
        if e["op"] == "apply" and canon(e["actions"]) == canon(events[k]["actions"]):
            return e
    return None


def affordability(an, ctx, seq, ev, obs, economics):
    acts, ans = ev["actions"], ev["answer"]
    if ev["op"] == "apply":
        # Applied without a validate: the apply answers the same test, at the same state.
        ans = {"valid": bool(ans.get("applied")), "results": ans.get("results") or []}
    res = ans.get("results") or []
    codes = [r.get("code") for r in res]
    refs = action_refs(acts, res, ctx.get("attemptIdx"))
    staffing = all(a.get("type") in STAFFING for a in acts)
    fo = funding_obs(obs)
    rule = bool(fo and "fundingHeadroomCents" in fo)
    pv = ev.get("preview")
    fund = (pv or {}).get("funding")
    before = state_before(ctx["events"], ctx["eventIndex"], fo)
    meas = {"fundingAtObservation": fo, "fundingRuleInEconomics": rule}
    if before is not fo:
        meas["fundingBefore"] = before
        meas["fundingBeforeSource"] = "the engine's observation after the batch applied before it at this point"
    lines = lines_of(pv)
    if staffing:
        meas["engineFunding"] = fund
        meas["guaranteed"] = lines
        meas["recurring"] = recurring(lines) + [
            {"label": "Teleradiology retainer, open-ended", "cents": t.get("monthlyRetainerCents"),
             "when": "per month from " + str(t.get("effective")) + " until a cancellation fixes the end",
             "basis": "the preview's monthly retainer (contract op start)"}
            for t in (pv or {}).get("teleradiology") or [] if t.get("op") == "start" and not t.get("endDate")]
        meas["immediateLedger"] = immediate(lines)
    paid_ext = []
    for a, r in zip(acts, res):
        if a.get("type") == "extend_shift" and r.get("ok"):
            m = EXT_PAID.search(r.get("message") or "")
            paid_ext.append({"employee": a.get("employee"), "minutes": a.get("minutes"), "date": a.get("date"),
                             "paid": bool(m), "estimateDollars": int(m["d"].replace(",", "")) if m else None,
                             "estimateBasis": "the engine's message: an estimate, without benefits; not the "
                                              "commitment"})
    if paid_ext:
        meas["extensions"] = paid_ext
    if "insufficient_funds" in codes:
        status, basis = "refused_insufficient_funds", "the engine refused the batch with insufficient_funds"
        meas["refusalMessage"] = next(r.get("message") for r in res if r.get("code") == "insufficient_funds")
    elif not ans.get("valid"):
        status = "not_tested"
        basis = "refused for another reason before or instead of the funding test: " + ", ".join(
            sorted({c for c in codes if c not in ("accepted", "unchanged", "batch_rejected")}) or ["batch_rejected"])
    elif staffing:
        if not rule or fund is None:
            status, basis = "evidence_unavailable", ("no funding rule in this economics" if not rule else
                                                     "the preview carries no funding test")
        elif fund.get("commitmentCents", 0) == 0:
            status, basis = "no_new_commitment", "the engine's funding test measures no commitment"
        elif before["fundingHeadroomCents"] != fund.get("headroomCents"):
            status, basis = "evidence_unavailable", "the observed funding is not the state the preview tested"
        elif fund["commitmentCents"] <= before["headroomWithoutCreditCents"]:
            status, basis = "funded_without_credit", "commitment within the headroom without undrawn credit"
        else:
            status, basis = "funded_with_credit", "commitment exceeds the headroom without undrawn credit"
    else:
        paid = [x for x in paid_ext if x["paid"]]
        k_ap = applied_index(ctx["events"], ctx["eventIndex"])
        ap = ctx["events"][k_ap] if k_ap is not None else None
        before = state_before(ctx["events"], k_ap, fo) if k_ap is not None else fo
        if before is not fo:
            meas["fundingBefore"] = before
            meas["fundingBeforeSource"] = "the engine's observation after the batch applied before it at this point"
        if paid and ap and ap.get("fundingAfter") and before and "reservedCents" in before:
            # The engine reserves an approved extension while it is still to
            # be worked (V0.7b): the reservation it adds is its commitment.
            meas["extensionCommitmentCents"] = ap["fundingAfter"]["reservedCents"] - before["reservedCents"]
            meas["fundingAfterApply"] = ap["fundingAfter"]
        if not paid:
            status, basis = "no_new_commitment", "no paid extension and no other action that adds a cost"
        elif not rule:
            status, basis = "evidence_unavailable", "no funding rule in this economics"
        elif "extensionCommitmentCents" not in meas:
            status, basis = "evidence_unavailable", ("funded, but never applied, so the engine reserved nothing "
                                                     "that measures its commitment")
        elif meas["extensionCommitmentCents"] <= before["headroomWithoutCreditCents"]:
            status, basis = "funded_without_credit", ("the reservation the engine added is within the headroom "
                                                      "without undrawn credit")
        else:
            status, basis = "funded_with_credit", ("the reservation the engine added exceeds the headroom without "
                                                   "undrawn credit")
    ap = applied_after(ctx["events"], ctx["eventIndex"])
    execution = ("refused" if not ans.get("valid") else
                 "applied" if ap and ap["answer"].get("applied") else "not_applied")
    an.add(ctx, "AFFORDABILITY", status, seq, refs,
           {"basis": basis, "validation": {"valid": bool(ans.get("valid")), "codes": codes},
            "execution": execution, "measurements": meas, "source": source_of(ev),
            "completeness": completeness(ctx, obs, staffing, pv, ev)})


def source_of(ev):
    """Where the submission's answer comes from."""
    if ev.get("inserted"):
        return {"kind": "validated again at its point (a recorded final reply the replay does not submit)",
                "recordedValidation": ev.get("recordedValidation")}
    if ev["op"] == "apply":
        return {"kind": "the replay's apply (the recorded batch, submitted without a validate)"}
    return {"kind": "the replay's validate"}


def completeness(ctx, obs, staffing, pv, ev):
    out = {"attemptLevel": ctx["format"] == "llm-agent 3", "attemptLinked": ctx["attempt"]["id"] is not None,
           "observation": obs is not None, "catalog": ctx.get("catalog") is not None}
    if staffing:
        out["preview"] = pv is not None
        if ev.get("previewError"):
            out["previewError"] = ev["previewError"]
    return out


def find_preview_item(pv, key, a):
    for x in (pv or {}).get(key) or []:
        if key == "hires" and x.get("requestId") == a.get("requestId"):
            return x
        if key in ("terminations", "rosters", "allocations") and x.get("employee") == a.get("employee"):
            return x
        if key == "teleradiology" and x.get("op") == a.get("op"):
            return x
    return None


def executed(an, ctx, seq, ev, val_ev, obs, run_end):
    """Reversibility and direct-change records for each action the engine
    applied with a change."""
    acts, res = ev["actions"], ev["answer"].get("results") or []
    pv = (val_ev or {}).get("preview")
    lines = lines_of(pv)
    staff = staff_of(obs)
    pol = (obs or {}).get("policies") or {}
    for k, (a, r) in enumerate(zip(acts, res)):
        if not r.get("ok") or r.get("code") != "accepted":
            continue
        ref = action_refs([a], [r], [ctx["attemptIdx"][k]] if ctx.get("attemptIdx") and
                          k < len(ctx["attemptIdx"]) else None)
        ref[0]["index"] = k
        t = a.get("type")
        eff = r.get("effectiveAt") or a.get("date")
        in_run = bool(eff and run_end and eff[:10] < run_end[:10])
        staffing = t in STAFFING
        # --- reversibility
        mk = matrix_key(a)
        rule = MATRIX.get(mk)
        facts = {"effective": eff, "effectiveWithinRun": in_run if eff else None}
        if rule is None:
            rstatus = "unknown"
        else:
            rstatus = rule["class"]
            if t in REMOVE_OPS and a.get("op") == "remove":
                rstatus = "withdrawable"  # a removal is itself undone by setting the version again
        item = None
        if t == "hire":
            item = find_preview_item(pv, "hires", a)
            exit_line = next((m for m in lines if (m["label"] or "").startswith("Earliest lawful exit")), None)
            fee = next((m for m in lines if (m["label"] or "").startswith("Recruiting fee")), None)
            facts.update({"startDate": (item or {}).get("startDate"),
                          "recruitingFeeCents": (fee or {}).get("cents"),
                          "earliestExitCents": (exit_line or {}).get("cents"),
                          "earliestExitWhen": (exit_line or {}).get("when"),
                          "weeklyGuarantee": next(({"cents": m["cents"], "when": m["when"]} for m in lines
                                                   if (m["label"] or "").startswith("Contract guarantee per full week")),
                                                  None)})
            if len([x for x in acts if x.get("type") == "hire"]) > 1:
                facts["amountsAre"] = "batch lines; the batch has several hires"
        elif t == "terminate":
            item = find_preview_item(pv, "terminations", a)
            facts.update({k2: (item or {}).get(k2) for k2 in ("lastDay", "endDate", "noticeDays", "severanceCents",
                                                               "severancePayday")})
        elif t == "set_reading_allocation":
            item = find_preview_item(pv, "allocations", a)
            facts.update({"withdrawBy": (item or {}).get("withdrawBy"), "until": (item or {}).get("until")})
        elif t == "set_roster":
            item = find_preview_item(pv, "rosters", a)
            facts.update({"until": (item or {}).get("until")})
        elif t == "contract_teleradiology":
            item = find_preview_item(pv, "teleradiology", a)
            facts.update({k2: (item or {}).get(k2) for k2 in ("minimumTermEnd", "monthlyRetainerCents",
                                                               "guaranteedCents", "guaranteedThrough", "endDate")})
        if staffing and pv is None:
            rcomp = "rule only: no preview for the batch"
        elif staffing and item is None:
            rcomp = "rule only: the preview names no matching item"
        else:
            rcomp = "complete"
        an.add(ctx, "REVERSIBILITY", rstatus, seq, ref,
               {"matrix": mk, "facts": facts, "evidence": rcomp,
                "completeness": completeness(ctx, obs, staffing, pv, val_ev or {})})
        # --- operational exposure (A: direct change only)
        ch = {}
        if t == "extend_shift":
            s = staff.get(a.get("employee")) or {}
            m = re.search(r"approved overtime[^)]*\) · (?P<c>.+?) \d{1,2}:\d\d–\d{1,2}:\d\d", r.get("message") or "")
            ch = {"resource": "scheduled duty", "employee": a.get("employee"), "role": s.get("role"),
                  "date": a.get("date"), "minutes": a.get("minutes"), "covers": m["c"] if m else None}
        elif t in ("set_walkin_policy", "set_worklist_policy"):
            key = "walkIn" if t == "set_walkin_policy" else "worklist"
            ch = {"resource": "walk-in policy" if key == "walkIn" else "reporting order",
                  "before": pol.get(key), "after": a.get("policy")}
        elif t == "hire" and item:
            ch = {"resource": "technologist capacity" if a.get("role") == "technologist" else "front-desk capacity",
                  **{k2: item.get(k2) for k2 in ("offer", "tier", "contractMin", "rosterMinA", "rosterMinB",
                                                  "startDate", "productiveDate", "trainingDays", "opensSlots")}}
        elif t == "terminate" and item:
            s = staff.get(a.get("employee")) or {}
            ch = {"resource": {"technologist": "technologist capacity", "front_desk": "front-desk capacity"}
                  .get(s.get("role"), f"{s.get('role')} capacity"), "employee": a.get("employee"),
                  "role": s.get("role"), "lastDay": item.get("lastDay"), "endDate": item.get("endDate"),
                  "displacedBookingsInBatch": item.get("displaced"),
                  "voidedExtensions": len(item.get("voidedExtensions") or []),
                  "voidedRosters": item.get("voidedRosters") or []}
        elif t == "set_roster" and item:
            ch = {"resource": "rostered time", "employee": a.get("employee"), "op": item.get("op"),
                  **{k2: item.get(k2) for k2 in ("effective", "contractMin", "beforeMinA", "beforeMinB", "afterMinA",
                                                  "afterMinB", "productiveBefore", "productiveAfter")}}
        elif t == "set_reading_allocation" and item:
            ch = {"resource": "reporting allocation", "employee": a.get("employee"), "op": item.get("op"),
                  **{k2: item.get(k2) for k2 in ("effective", "beforeBP", "afterBP", "beforeSiteMinA",
                                                  "beforeSiteMinB", "afterSiteMinA", "afterSiteMinB",
                                                  "beforeSalary30Cents", "afterSalary30Cents")},
                  "backlogAtDecision": {"studies": item.get("backlogStudies"), "urgent": item.get("backlogUrgent")}}
        elif t == "contract_teleradiology" and item:
            ch = {"resource": "teleradiology coverage",
                  **{k2: item.get(k2) for k2 in ("op", "plan", "windows", "scope", "effective", "endDate")}}
        if staffing and pv:
            ch["batchSlots"] = {"opened": pv.get("slotsOpened") or [], "closed": pv.get("slotsClosed") or [],
                                "displacedBookings": len(pv.get("displacements") or [])}
        if not ch:
            estatus = "evidence_unavailable"
        elif t in ("set_walkin_policy", "set_worklist_policy") and ch.get("before") == ch.get("after"):
            estatus = "no_direct_change"
        else:
            estatus = "direct_change"
        an.add(ctx, "OPERATIONAL_EXPOSURE", estatus, seq, ref,
               {"directChange": ch, "effective": eff, "effectiveWithinRun": in_run if eff else None,
                "subsequent": {"observedIn": "runs[].timeline", "attribution": "none"},
                "completeness": completeness(ctx, obs, staffing, pv, val_ev or {})})


# --- the corpus ---------------------------------------------------------------------------------

def order(records):
    return sorted(records, key=lambda r: (r["recording"]["group"], r["recording"]["path"], r["decision"]["index"],
                                          r["submission"], list(RULES).index(r["dimension"]),
                                          r["actions"][0]["index"] if r["actions"] else -1, r["id"]))


def run_dirs(replays, out_name, recmap):
    """recording path -> replay run directory, from the run's own record of
    the recording it replayed."""
    by_base = {os.path.basename(p): p for ps in recmap.values() for p in ps}
    runs = {}
    d = os.path.join(replays, out_name, "runs")
    if os.path.isdir(d):
        for rid in sorted(os.listdir(d)):
            ev = os.path.join(d, rid, "policy-evidence", "decisions.jsonl")
            if not os.path.exists(ev):
                continue
            with open(ev, encoding="utf-8") as f:
                first = f.readline()
            rb = json.loads(first).get("replayed") if first.strip() else None
            if rb in by_base:
                runs[by_base[rb]] = os.path.join(d, rid)
    return runs


def paired(inputs, suite_runs, replays, corpus, group):
    """C: each LLM run with its no-action pair, from the committed contract
    reports (financial_outcomes' helpers)."""
    out = {}
    for sr in suite_runs:
        c = D.load_json(inputs, sr["contract"])
        rmap = FO.replay_runs(corpus, replays, group)
        base = {(r["entry"], r["seed"]): r for r in c["runs"] if r["policy"]["id"] == c["baseline"]}
        for pair in c["pairs"]:
            run = next(r for r in c["runs"] if r["entry"] == pair["entry"] and r["seed"] == pair["seed"]
                       and r["policy"]["id"] == pair["policy"])
            b = base[(pair["entry"], pair["seed"])]
            row = {"suiteRun": sr["id"], "contract": sr["contract"], "economics": sr["economics"],
                   "entry": pair["entry"], "seed": pair["seed"], "days": run["days"], "noActionRun": b["id"]}
            for d_, n in FO.FIN + FO.OPS:
                row[n] = {"policy": FO.metric(run["dimensions"], d_, n), "noAction": FO.metric(b["dimensions"], d_, n),
                          "difference": FO.metric(pair["dimensions"], d_, n, "deltas")}
            ex_p, ex_b = FO.replay_extras(rmap.get(run["id"])), FO.replay_extras(rmap.get(b["id"]))
            if ex_p and ex_b:
                for k in ("overtimeCostCents", "maxCreditDrawnCents", "interestCents"):
                    row[k] = {"policy": ex_p[k], "noAction": ex_b[k], "difference": (ex_p[k] or 0) - (ex_b[k] or 0)}
            out[run["id"]] = row
    return out


def replay_all(binary, corpus, replays, jobs):
    """Every run's engine replay, run concurrently (each is its own
    process; the results do not depend on the order they finish in)."""
    tasks = []
    for g in corpus["groups"]:
        for rp in g["replays"]:
            with open(rp["config"], encoding="utf-8") as f:
                recmap = D.replay_recordings(json.load(f))
            for path, run_dir in sorted(run_dirs(replays, rp["out"], recmap).items()):
                with open(path, encoding="utf-8") as f:
                    recs = [json.loads(l) for l in f if l.strip()]
                with open(os.path.join(run_dir, "run.json"), encoding="utf-8") as f:
                    run = json.load(f)
                tasks.append((run_dir, run, [r.get("digests") for r in recs], v2_extras(recs)))
    with concurrent.futures.ThreadPoolExecutor(max_workers=max(1, jobs)) as ex:
        futs = {t[0]: ex.submit(engine_replay, binary, *t) for t in tasks}
        return {k: f.result() for k, f in futs.items()}


def analyze(binary, corpus_path, replays, inputs=None, jobs=1):
    inputs = inputs or D.Inputs(replays)
    corpus = D.load_json(inputs, corpus_path)
    if corpus.get("schema") != CORPUS_SCHEMA:
        raise SystemExit(f"{corpus_path}: not a {CORPUS_SCHEMA} file")
    done = replay_all(binary, corpus, replays, jobs)
    an = Analysis()
    runs_out = []
    for g in corpus["groups"]:
        pairs = paired(inputs, g.get("suiteRuns") or [], replays, corpus, g["id"])
        for rp in g["replays"]:
            cfg = D.load_json(inputs, rp["config"])
            recmap = D.replay_recordings(cfg)
            rdirs = run_dirs(replays, rp["out"], recmap)
            for pid in sorted(recmap):
                for path in recmap[pid]:
                    raw = inputs.read(path)
                    recs = [json.loads(l) for l in raw.decode("utf-8").splitlines() if l.strip()]
                    fmt = ("llm-agent 3" if recs and recs[0].get("schema") == V3_SCHEMA else
                           f"llm-agent {recs[0].get('agentVersion') or 1}" if recs else "empty")
                    run_dir = rdirs.get(path)
                    rec_id = {"path": path, "sha256": sha256_bytes(raw), "group": g["id"], "format": fmt}
                    entry = {"recording": rec_id, "policy": pid, "model": D.model_of(path, inputs),
                             "economics": g["economics"], "attemptLevel": fmt == "llm-agent 3"}
                    if fmt != "llm-agent 3":
                        entry["unrecordedAttempts"], entry["unrecordedAttemptRefusals"] = unrecorded_attempts(recs)
                    if not run_dir:
                        entry["replay"] = None
                        runs_out.append(entry)
                        continue
                    with open(os.path.join(run_dir, "run.json"), encoding="utf-8") as f:
                        run = json.load(f)
                    segs, nreq = done[run_dir]
                    run_end = (run.get("final") or {}).get("time")
                    run_id = {"policy": pid, "model": entry["model"], "scenario": (run.get("scenario") or {}).get("id"),
                              "seed": run.get("seed"), "days": run.get("days"),
                              "replayRun": os.path.join(rp["out"], "runs", os.path.basename(run_dir)),
                              "economics": g["economics"]}
                    entry.update({"run": run_id, "replay": {"requests": nreq, "identical": True,
                                                            "pointsMatched": len(segs), "points": len(recs)},
                                  "paired": pairs.get(os.path.basename(run_dir)),
                                  "timeline": []})
                    for i, rec in enumerate(recs):
                        seg = segs.get(i) or {"events": []}
                        obs = seg.get("obs")
                        if obs is not None:
                            entry["timeline"].append({"point": i, **observed_state(obs)})
                        events = seg["events"]
                        seen = set()
                        for ev in events:
                            # A submission: a validated batch, or a batch applied without a validate
                            # (the llm-agent 2 replays apply the recorded batches directly).
                            if ev["op"] == "validate":
                                ev["submission"] = ev.get("answer") is not None
                                seen.add(canon(ev["actions"]))
                            elif ev["op"] == "apply":
                                ev["submission"] = canon(ev["actions"]) not in seen
                        recorded = recorded_validation(rec)
                        for ev in events:
                            if ev.get("inserted"):
                                want = recorded.get(canon(ev["actions"]))
                                ev["recordedValidation"] = ("not recorded" if want is None else
                                                            "reproduced" if want == ev.get("answer") else "differs")
                        links = attempts_for(rec, events)
                        point = {"index": i, "point": rec.get("point"), "time": rec.get("time")}
                        seq = 0
                        for k, ev in enumerate(events):
                            link = links.get(k)
                            ctx = {"recording": rec_id, "run": run_id, "point": point, "format": fmt,
                                   "events": events, "eventIndex": k, "catalog": seg.get("catalog"),
                                   "attempt": {"id": link[0] if link else None,
                                               "recorded": fmt == "llm-agent 3" or bool(link)},
                                   "attemptIdx": link[1] if link else None}
                            if ev.get("submission"):
                                seq += 1
                                ev["seq"] = seq
                                ev["ctx"] = ctx
                                affordability(an, ctx, seq, ev, obs, g["economics"])
                            if ev["op"] == "apply" and ev["answer"].get("applied"):
                                val = ev if ev.get("submission") else next(
                                    (e for e in reversed(events[:k]) if e["op"] == "validate" and
                                     canon(e["actions"]) == canon(ev["actions"])), None)
                                vctx = (val or {}).get("ctx") or ctx
                                executed(an, vctx, (val or {}).get("seq", 0), ev, val, obs, run_end)
                    runs_out.append(entry)
    an.records = order(an.records)
    ids = [r["id"] for r in an.records]
    if len(ids) != len(set(ids)):
        seen, dup = set(), []
        for r in an.records:
            if r["id"] in seen:
                dup.append(r)
            seen.add(r["id"])
        for r in dup[:5]:
            print("duplicate", r["id"], r["dimension"], r["recording"]["path"], r["decision"], r["submission"],
                  [(a["index"], a["type"]) for a in r["actions"]], file=sys.stderr)
        raise SystemExit("record identifiers are not unique")
    doc = {"schema": SCHEMA, "analyzer": ANALYZER,
           "rules": [{"id": r, "version": v} for r, v in RULES.items()],
           "statuses": {"AFFORDABILITY": list(AFFORDABILITY), "REVERSIBILITY": list(REVERSIBILITY),
                        "OPERATIONAL_EXPOSURE": list(EXPOSURE)},
           "reversibilityMatrix": MATRIX,
           "corpus": {"path": corpus_path, "id": corpus.get("id")},
           "inputs": [{"path": p, "sha256": h} for p, h in sorted(inputs.files.items())],
           "runs": sorted(runs_out, key=lambda e: (e["recording"]["group"], e["recording"]["path"])),
           "summary": summary(an.records, runs_out),
           "records": an.records}
    return doc


# --- summaries -----------------------------------------------------------------------------------

def summary(records, runs):
    out = {}
    for r in records:
        g = r["recording"]["group"]
        s = out.setdefault(g, {"AFFORDABILITY": {}, "REVERSIBILITY": {}, "OPERATIONAL_EXPOSURE": {},
                               "byType": {}, "gaps": {}})
        s[r["dimension"]][r["status"]] = s[r["dimension"]].get(r["status"], 0) + 1
        if r["dimension"] == "AFFORDABILITY":
            for a in r["actions"]:
                bt = s["byType"].setdefault(a["type"], {"submitted": 0, "accepted": 0, "refused": 0, "executed": 0})
                bt["submitted"] += 1
                if a["validation"]["ok"]:
                    bt["accepted"] += 1
                else:
                    bt["refused"] += 1
            c = r["completeness"]
            for k, bad in (("attemptNotLinked", not c["attemptLinked"]), ("noObservation", not c["observation"]),
                           ("noPreview", c.get("preview") is False)):
                if bad:
                    s["gaps"][k] = s["gaps"].get(k, 0) + 1
        elif r["dimension"] == "REVERSIBILITY":
            for a in r["actions"]:
                s["byType"].setdefault(a["type"], {"submitted": 0, "accepted": 0, "refused": 0, "executed": 0})
                s["byType"][a["type"]]["executed"] += 1
            if r["evidence"] != "complete":
                s["gaps"]["reversibilityRuleOnly"] = s["gaps"].get("reversibilityRuleOnly", 0) + 1
    for g, s in out.items():
        for k in ("AFFORDABILITY", "REVERSIBILITY", "OPERATIONAL_EXPOSURE", "byType", "gaps"):
            s[k] = dict(sorted(s[k].items()))
        s["recordings"] = sum(1 for e in runs if e["recording"]["group"] == g)
        s["unrecordedAttempts"] = sum(e.get("unrecordedAttempts") or 0 for e in runs if e["recording"]["group"] == g)
        ur = {}
        for e in runs:
            if e["recording"]["group"] == g:
                for c, k in (e.get("unrecordedAttemptRefusals") or {}).items():
                    ur[c] = ur.get(c, 0) + k
        s["unrecordedAttemptRefusals"] = dict(sorted(ur.items()))
        s["replayed"] = sum(1 for e in runs if e["recording"]["group"] == g and e.get("replay"))
    return dict(sorted(out.items()))


# --- tables ---------------------------------------------------------------------------------------

def cents(c):
    """Exact money: dollars and cents, no rounding."""
    if not isinstance(c, int):
        return "—"
    sign = "−" if c < 0 else ""
    c = abs(c)
    return f"{sign}${c // 100:,}.{c % 100:02d}"


def scents(c):
    return "—" if not isinstance(c, int) else ("+" if c > 0 else "") + cents(c)


def num(v):
    return "—" if not isinstance(v, int) else f"{v:,}".replace("-", "−")


def snum(v):
    return "—" if not isinstance(v, int) else ("+" if v > 0 else "") + num(v)


def commitment_of(r):
    m = r.get("measurements") or {}
    if (m.get("engineFunding") or {}).get("commitmentCents") is not None:
        return m["engineFunding"]["commitmentCents"], "staffing"
    if m.get("extensionCommitmentCents") is not None:
        return m["extensionCommitmentCents"], "extension"
    return None, None


def act_text(a):
    t = a["type"]
    bits = [x for x in (a.get("op"), a.get("employee"), a.get("contract"), a.get("requestId"), a.get("date")) if x]
    return t + (" (" + ", ".join(bits) + ")" if bits else "")


def tables(doc):
    recs = doc["records"]
    groups = sorted(doc["summary"])
    L = ["# Financial decision evidence: tables", "",
         f"`{doc['schema']}`, analyzer `{doc['analyzer']}`. Three dimensions, each counted on its own; nothing here "
         "is a score, a ranking or a cause. Money is exact, in dollars and cents, as the engine states it. Each "
         "group has its own economics and is never combined with another.", ""]
    L += ["## 1. Coverage", "", "| Group | Economics | Recordings | Replayed through the engine | Submissions | "
          "Executed actions | Earlier attempts not recorded |", "|---|---|---|---|---|---|---|"]
    for g in groups:
        s = doc["summary"][g]
        econ = next(e["economics"] for e in doc["runs"] if e["recording"]["group"] == g)
        L.append(f"| {g} | {econ} | {s['recordings']} | {s['replayed']} | {sum(s['AFFORDABILITY'].values())} | "
                 f"{sum(s['REVERSIBILITY'].values())} | {s['unrecordedAttempts']} |")
    L += ["", "Earlier attempts not recorded: llm-agent 2 (V0.6c, V0.7c) records only its final reply; the earlier "
          "attempts' actions are unknown, their refusal codes are listed in §8.", ""]
    L += ["## 2. Affordability (one record per submitted batch)", "",
          "| Group | " + " | ".join(AFFORDABILITY) + " | Total |", "|---|" + "---|" * (len(AFFORDABILITY) + 1)]
    for g in groups:
        a = doc["summary"][g]["AFFORDABILITY"]
        L.append(f"| {g} | " + " | ".join(num(a.get(k, 0)) for k in AFFORDABILITY) + f" | {num(sum(a.values()))} |")
    L += ["", "Actions by type: submitted, accepted by validation, refused, and executed with a change.", "",
          "| Group | Type | Submitted | Accepted | Refused | Executed |", "|---|---|---|---|---|---|"]
    for g in groups:
        for t, v in doc["summary"][g]["byType"].items():
            L.append(f"| {g} | {t} | {num(v['submitted'])} | {num(v['accepted'])} | {num(v['refused'])} | "
                     f"{num(v['executed'])} |")
    L += ["", "Commitments the engine measured for executed batches (the funding test's commitment for a staffing "
          "batch; the reservation an applied extension batch added), by funding status:", "",
          "| Group | Kind | Status | Batches | Total commitment | Largest |", "|---|---|---|---|---|---|"]
    for g in groups:
        agg = {}
        for r in recs:
            if r["dimension"] != "AFFORDABILITY" or r["recording"]["group"] != g or r["execution"] != "applied":
                continue
            c, kind = commitment_of(r)
            if c is None or r["status"] not in ("funded_without_credit", "funded_with_credit"):
                continue
            x = agg.setdefault((kind, r["status"]), [0, 0, 0])
            x[0] += 1
            x[1] += c
            x[2] = max(x[2], c)
        for (kind, st), (n, tot, mx) in sorted(agg.items()):
            L.append(f"| {g} | {kind} | {st} | {num(n)} | {cents(tot)} | {cents(mx)} |")
    L += ["", "Refused for insufficient funds (the engine's commitment and headroom):", "",
          "| Recording | Point | Actions | Commitment | Headroom | Headroom without credit |", "|---|---|---|---|---|---|"]
    for r in recs:
        if r["dimension"] == "AFFORDABILITY" and r["status"] == "refused_insufficient_funds":
            m = r["measurements"]
            f = m.get("engineFunding") or {}
            b = m.get("fundingBefore") or m.get("fundingAtObservation") or {}
            L.append(f"| {os.path.basename(r['recording']['path'])} | {r['decision']['time']} | "
                     f"{', '.join(act_text(a) for a in r['actions'])} | {cents(f.get('commitmentCents'))} | "
                     f"{cents(f.get('headroomCents', b.get('fundingHeadroomCents')))} | "
                     f"{cents(b.get('headroomWithoutCreditCents'))} |")
    L += ["", "Recurring obligations and immediate ledger effects the engine states for executed staffing batches:", "",
          "| Group | Line | Lines | Total |", "|---|---|---|---|"]
    for g in groups:
        agg = {}
        for r in recs:
            if r["dimension"] != "AFFORDABILITY" or r["recording"]["group"] != g or r["execution"] != "applied":
                continue
            m = r["measurements"]
            for kind, lines in (("recurring", m.get("recurring") or []), ("immediate", m.get("immediateLedger") or [])):
                for ln in lines:
                    label = re.sub(r"\s*·.*", "", ln["label"] or "")
                    when = re.sub(r"\s+from .*|\d{4}-\d\d-\d\d", "", ln["when"] or "").strip()
                    x = agg.setdefault((kind, label, when), [0, 0])
                    x[0] += 1
                    x[1] += ln["cents"] or 0
        for (kind, label, when), (n, tot) in sorted(agg.items()):
            L.append(f"| {g} | {kind}: {label} ({when}) | {num(n)} | {cents(tot)} |")
    L += ["", "## 3. Credit usage", "",
          "Observed at the decision points (B), and the most drawn at any day's end in the run with interest charged "
          "(from the replay's calendar, C). V0.6c has no credit line.", "",
          "| Recording | Most drawn at a decision point | Most drawn at a day's end | Interest |", "|---|---|---|---|"]
    for e in doc["runs"]:
        if e["recording"]["group"] == "v06c" or not e.get("timeline"):
            continue
        mx = max((t.get("creditDrawnCents") or 0) for t in e["timeline"])
        p = e.get("paired") or {}
        if mx or ((p.get("maxCreditDrawnCents") or {}).get("policy") or 0):
            L.append(f"| {os.path.basename(e['recording']['path'])} | {cents(mx)} | "
                     f"{cents((p.get('maxCreditDrawnCents') or {}).get('policy'))} | "
                     f"{cents((p.get('interestCents') or {}).get('policy'))} |")
    L += ["", "Recordings not listed drew no credit.", "",
          "## 4. Reversibility (one record per executed action)", "",
          "| Group | Action | " + " | ".join(REVERSIBILITY) + " | Effective within the run | After the run ends |",
          "|---|---|" + "---|" * (len(REVERSIBILITY) + 2)]
    for g in groups:
        agg = {}
        for r in recs:
            if r["dimension"] == "REVERSIBILITY" and r["recording"]["group"] == g:
                x = agg.setdefault(r["matrix"], {k: 0 for k in REVERSIBILITY + ("in", "after")})
                x[r["status"]] += 1
                x["in" if r["facts"].get("effectiveWithinRun") else "after"] += 1
        for mk, x in sorted(agg.items()):
            L.append(f"| {g} | {mk} | " + " | ".join(num(x[k]) for k in REVERSIBILITY) +
                     f" | {num(x['in'])} | {num(x['after'])} |")
    L += ["", "The rule behind each class is in the reversibility matrix (docs/benchmark/v09b-dev/REVERSIBILITY.md); "
          "the amounts are per record in financial-evidence.json.", "",
          "## 5. Operational exposure: direct state changes (A)", "",
          "| Group | Resource | Actions (batches, for booked appointments) | Detail |", "|---|---|---|---|"]
    for g in groups:
        agg, batches = {}, set()
        for r in recs:
            if r["dimension"] != "OPERATIONAL_EXPOSURE" or r["recording"]["group"] != g:
                continue
            ch = r["directChange"]
            x = agg.setdefault(ch.get("resource") or "unknown", {"n": 0, "min": 0, "slots": 0, "displaced": 0,
                                                                 "siteMin": 0})
            x["n"] += 1
            x["min"] += ch.get("minutes") or 0
            x["slots"] += sum(s.get("count") or 0 for s in ch.get("opensSlots") or [])
            key = (r["recording"]["path"], r["decision"]["index"], r["submission"])
            if ch.get("displacedBookingsInBatch") is not None and key not in batches:
                batches.add(key)  # a batch's displaced bookings: once per batch, not per resource
                dis = agg.setdefault("booked appointments (termination batches)",
                                     {"n": 0, "min": 0, "slots": 0, "displaced": 0, "siteMin": 0})
                dis["n"] += 1
                dis["displaced"] += ch["displacedBookingsInBatch"]
            if ch.get("afterSiteMinA") is not None:
                x["siteMin"] += (ch["afterSiteMinA"] + ch["afterSiteMinB"]) - (ch["beforeSiteMinA"] + ch["beforeSiteMinB"])
        for res, x in sorted(agg.items()):
            det = []
            if x["min"]:
                det.append(f"{num(x['min'])} extension minutes")
            if x["slots"]:
                det.append(f"{num(x['slots'])} template slots opened from the productive dates (each hire's own "
                           "count, summed: hires in one batch can count the same slots)")
            if x["displaced"]:
                det.append(f"{num(x['displaced'])} booked appointments displaced (counted once per batch)")
            if x["siteMin"]:
                det.append(f"{snum(x['siteMin'])} site reading minutes per two rota weeks")
            L.append(f"| {g} | {res} | {num(x['n'])} | {'; '.join(det) or '—'} |")
    L += ["", "## 6. Subsequent observed state (B): first and last decision point of each run", "",
          "Observed, never attributed to an action.", "",
          "| Recording | Cash | Funding headroom | Credit drawn | Orders lost since start | Unread studies | Headcount "
          "(employed: active, notice, onboarding) |", "|---|---|---|---|---|---|---|"]
    for e in doc["runs"]:
        tl = e.get("timeline") or []
        if not tl:
            continue
        a, b = tl[0], tl[-1]
        act = employed
        L.append(f"| {os.path.basename(e['recording']['path'])} | {cents(a['cashCents'])} → {cents(b['cashCents'])} | "
                 f"{cents(a['fundingHeadroomCents'])} → {cents(b['fundingHeadroomCents'])} | "
                 f"{cents(a['creditDrawnCents'])} → {cents(b['creditDrawnCents'])} | {num(b['ordersLostSinceStart'])} | "
                 f"{num(a['unreadStudies'])} → {num(b['unreadStudies'])} | {act(a)} → {act(b)} |")
    L += ["", "## 7. Paired with no action (C): the run's own contract report", "",
          "Policy minus no action on the same entry, seed, horizon and economics. A difference describes two runs; "
          "it is not allocated to any action.", "",
          "| Recording | Days | Profit | Lowest cash | Orders lost | Exams completed | Overtime cost |",
          "|---|---|---|---|---|---|---|"]
    for e in doc["runs"]:
        p = e.get("paired")
        if not p:
            continue
        d = lambda k: (p.get(k) or {}).get("difference")
        L.append(f"| {os.path.basename(e['recording']['path'])} | {p['days']} | {scents(d('operatingProfitCents'))} | "
                 f"{scents(d('minCashCents'))} | {snum(d('ordersLost'))} | {snum(d('examsCompleted'))} | "
                 f"{scents(d('overtimeCostCents'))} |")
    L += ["", "## 8. Evidence gaps", "", "| Group | Gap | Count |", "|---|---|---|"]
    for g in groups:
        s = doc["summary"][g]
        for k, v in s["gaps"].items():
            L.append(f"| {g} | {k} | {num(v)} |")
        L.append(f"| {g} | earlier attempts not recorded (llm-agent 2) | {num(s['unrecordedAttempts'])} |")
        for c, v in s["unrecordedAttemptRefusals"].items():
            L.append(f"| {g} | … refused with {c} (actions not recorded) | {num(v)} |")
        ab = sum(v for k, v in s["AFFORDABILITY"].items() if k == "evidence_unavailable")
        L.append(f"| {g} | affordability not established (evidence_unavailable) | {num(ab)} |")
    L.append("")
    return "\n".join(L)


# --- case studies ---------------------------------------------------------------------------------

def staff_text(t):
    """Employed staff by role, with those on notice or onboarding named."""
    by = {}
    for k, v in t["headcount"].items():
        role, stage = k.split(":", 1)
        by.setdefault(role, {})[stage] = v
    out = []
    for role, st in sorted(by.items()):
        extra = [f"{st[x]} {x}" for x in ("notice", "onboarding", "recruiting") if st.get(x)]
        out.append(f"{role} {st.get('active', 0)}" + (f" (+{', '.join(extra)})" if extra else ""))
    return ", ".join(out)


def employed(t):
    return sum(t["headcount"].values())


def known_line(t):
    if not t:
        return "no observation"
    act = staff_text(t)
    return (f"cash {cents(t['cashCents'])}; funding headroom {cents(t['fundingHeadroomCents'])}; credit drawn "
            f"{cents(t['creditDrawnCents'])}; reserved {cents(t['reservedCents'])}; staff: {act}; "
            f"unread studies {num(t['unreadStudies'])}; orders lost so far {num(t['ordersLostSinceStart'])}")


def case_study(doc, spec):
    e = next(x for x in doc["runs"] if x["recording"]["path"] == spec["recording"])
    recs = [r for r in doc["records"] if r["recording"]["path"] == spec["recording"]]
    tl = {t["point"]: t for t in e.get("timeline") or []}
    L = [f"## {spec['title']}", "", spec["why"], "",
         f"* Recording: `{spec['recording']}` ({e['recording']['format']}), policy {e['policy']}, model {e['model']}.",
         f"* Run: {e['run']['scenario']}, seed {e['run']['seed']}, {e['run']['days']} days; {e['economics']}.",
         f"* Engine replay: {e['replay']['requests']} requests, every original answer identical; "
         f"{e['replay']['pointsMatched']} of {e['replay']['points']} decision points matched.",
         ""]
    if not e["attemptLevel"]:
        L.append(f"* Earlier attempts not recorded: {e.get('unrecordedAttempts', 0)}; their refused actions' codes: "
                 f"{', '.join(f'{c} {n}' for c, n in (e.get('unrecordedAttemptRefusals') or {}).items()) or 'none'} "
                 "(an attempt can be lost to a parse or protocol error, which has no refusal code).")
        L.append("")
    L += ["### Decision points with a submission", "",
          "Each point shows what was observable when the decision was made, then the engine's answer. Nothing later "
          "is shown here.", ""]
    by_point = {}
    for r in recs:
        by_point.setdefault(r["decision"]["index"], []).append(r)
    for i in sorted(by_point):
        rs = by_point[i]
        L.append(f"**Point {i} ({rs[0]['decision']['time']}).** Known: {known_line(tl.get(i))}.")
        L.append("")
        for r in rs:
            if r["dimension"] == "AFFORDABILITY":
                c, kind = commitment_of(r)
                m = r["measurements"]
                extra = []
                if c is not None:
                    extra.append(f"commitment {cents(c)}")
                for ln in (m.get("immediateLedger") or []):
                    extra.append(f"{ln['label']}: {cents(ln['cents'])} ({ln['when']})")
                for ln in (m.get("recurring") or []):
                    extra.append(f"{ln['label']}: {cents(ln['cents'])} {ln['when']}")
                L.append(f"* Submitted {', '.join(act_text(a) for a in r['actions'])} → "
                         f"{', '.join(sorted(set(r['validation']['codes'])))}; affordability **{r['status']}**"
                         f" ({r['basis']}); {r['execution']}" + ("; " + "; ".join(extra) if extra else "") + ".")
            elif r["dimension"] == "REVERSIBILITY":
                f = r["facts"]
                det = ", ".join(f"{k} {cents(v) if k.endswith('Cents') else v}" for k, v in f.items()
                                if v is not None and k not in ("effectiveWithinRun",) and not isinstance(v, dict))
                L.append(f"  * Executed {act_text(r['actions'][0])}: reversibility **{r['status']}**; "
                         f"{det or 'no dated terms (in effect at once)'}.")
            else:
                ch = {k: v for k, v in r["directChange"].items() if k not in ("batchSlots",) and v not in (None, [], {})}
                L.append(f"  * Direct change: {json.dumps(ch, ensure_ascii=False, sort_keys=True)}.")
        L.append("")
    L += ["### Observed later (B): not attributed to any decision", "",
          "| Point | Time | Cash | Headroom | Credit drawn | Reserved | Staff (active, + notice/onboarding) | Unread | "
          "Orders lost so far |",
          "|---|---|---|---|---|---|---|---|---|"]
    pts = sorted(tl)
    step = max(1, len(pts) // 12)
    for i in pts[::step] + ([pts[-1]] if pts and pts[-1] not in pts[::step] else []):
        t = tl[i]
        L.append(f"| {i} | {t['time']} | {cents(t['cashCents'])} | {cents(t['fundingHeadroomCents'])} | "
                 f"{cents(t['creditDrawnCents'])} | {cents(t['reservedCents'])} | {staff_text(t)} | "
                 f"{num(t['unreadStudies'])} | "
                 f"{num(t['ordersLostSinceStart'])} |")
    p = e.get("paired")
    if p:
        d = lambda k: (p.get(k) or {}).get("difference")
        L += ["", "### Paired with no action (C)", "",
              f"Same entry, seed ({p['seed']}), {p['days']} days and economics, from `{p['contract']}`: profit "
              f"{scents(d('operatingProfitCents'))}, lowest cash {scents(d('minCashCents'))}, orders lost "
              f"{snum(d('ordersLost'))}, exams completed {snum(d('examsCompleted'))}, overtime cost "
              f"{scents(d('overtimeCostCents'))}, most credit drawn {cents((p.get('maxCreditDrawnCents') or {}).get('policy'))}"
              f" (no action: {cents((p.get('maxCreditDrawnCents') or {}).get('noAction'))}). These are differences "
              "between two whole runs; no part of them is allocated to a single decision."]
    L += ["", spec.get("limits", ""), ""]
    return "\n".join(L)


def case_studies(doc, specs):
    L = ["# Financial decision evidence: case studies", "",
         "Generated from financial-evidence.json; every number is the engine's. For each decision point the record "
         "shows only what was observable then (the decision point's observation and the engine's answers to the "
         "submitted batch); what happened later is in a separate table and is not attributed to any decision. No "
         "agent reasoning is inferred, and no decision is labelled good or bad from its outcome.", ""]
    for spec in specs:
        L.append(case_study(doc, spec))
    return "\n".join(L)


def matrix_page(doc):
    """The reversibility matrix as a page, with how often each rule applied."""
    cnt = {}
    for r in doc["records"]:
        if r["dimension"] == "REVERSIBILITY":
            k = (r["matrix"], r["recording"]["group"])
            cnt[k] = cnt.get(k, 0) + 1
    groups = sorted(doc["summary"])
    L = ["# Reversibility matrix", "",
         "The simulator's own rules for undoing each management action, as `dreamteam.financial-evidence/v1` "
         "applies them (`MATRIX` in analysis/evidence/financial_evidence.py; this page is generated from it). Each "
         "rule names the engine source it comes from. Nothing here is a judgement of the action: a reversible "
         "action is not harmless, and an irreversible one is not a mistake.", "",
         "The per-action amounts (fees, severance, earliest exit, guaranteed retainer, withdraw-by dates) are the "
         "engine's own preview values, per record in financial-evidence.json.", "",
         "| Action | Class | Effective | Notice | Inverse action | What remains owed | Cost of reversing | Source | "
         "Executed (" + " / ".join(groups) + ") |", "|---|---|---|---|---|---|---|---|---|"]
    for k, m in MATRIX.items():
        n = " / ".join(str(cnt.get((k, g), 0)) for g in groups)
        L.append(f"| `{k}` | {m['class']} | {m['effective']} | {m['notice']} | {m['inverse']} | {m['retained']} | "
                 f"{m['reversalCost']} | {m['source']} | {n} |")
    L += ["", "Notes:", "",
          "* **Withdrawal versus reversal.**",
          "  * An allocation or roster version is withdrawable only while it is pending, and only with the same "
          "notice.",
          "  * Once in effect, it changes only by a new version with notice. An allocation then stays in force "
          "at least 14 days (`MinCommitDays`).",
          "  * An executed op `remove` is itself classed `withdrawable`: setting the version again restores it, "
          "under the same notice.",
          "* **A hire.**",
          "  * It can be ended before its start date by `terminate`, but the recruiting fee is not refunded and "
          "the contract runs from the start (staffing.go).",
          "  * The cheapest exit is the preview's \"Earliest lawful exit\" amount.",
          "* **A teleradiology start** carries the retainer to the minimum-term end. It is payable even if the "
          "contract is cancelled before activation (telecontract.go).",
          "* **A termination and a cancellation** cannot be moved or withdrawn (`already_terminating`, "
          "`already_cancelled`). What they removed returns only through a new hire or contract, with its own fee, "
          "lead time and notice. Those costs are not computed here, because no such hire or contract was decided.",
          "* **An approved extension** has no withdrawal action. A later roster change or termination can void "
          "it, and the engine then releases its reservation.",
          "* **`unknown`** is the class for any action type the matrix does not cover. None occurred in the "
          "corpus.", ""]
    return "\n".join(L)


def write_json(path, obj):
    with open(path, "w", encoding="utf-8") as f:
        json.dump(obj, f, indent=1, sort_keys=True, ensure_ascii=True)
        f.write("\n")


def main(argv=None):
    ap = argparse.ArgumentParser(description="Financial decision evidence (read-only; no model).")
    ap.add_argument("--binary", required=True)
    ap.add_argument("--corpus", required=True)
    ap.add_argument("--replays", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--jobs", type=int, default=1, help="engine replays run at once (the output does not depend on it)")
    a = ap.parse_args(argv)
    try:
        doc = analyze(a.binary, a.corpus, a.replays.rstrip("/"), jobs=a.jobs)
    except ReplayMismatch as e:
        print(f"financial evidence: the engine replay diverged: {e}", file=sys.stderr)
        return 1
    os.makedirs(a.out, exist_ok=True)
    write_json(os.path.join(a.out, "financial-evidence.json"), doc)
    with open(os.path.join(a.out, "financial-tables.md"), "w", encoding="utf-8") as f:
        f.write(tables(doc))
    with open(a.corpus, encoding="utf-8") as f:
        specs = json.load(f).get("caseStudies") or []
    with open(os.path.join(a.out, "case-studies.md"), "w", encoding="utf-8") as f:
        f.write(case_studies(doc, specs))
    with open(os.path.join(a.out, "REVERSIBILITY.md"), "w", encoding="utf-8") as f:
        f.write(matrix_page(doc))
    print(f"financial evidence: {len(doc['records'])} records from {len(doc['runs'])} recordings")
    return 0


if __name__ == "__main__":
    sys.exit(main())
