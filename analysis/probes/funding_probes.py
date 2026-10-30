#!/usr/bin/env python3
"""Deterministic funding-refusal probes (dreamteam.funding-probes/v1).

    python3 -I -B analysis/probes/funding_probes.py --binary ./dt \
        --probes docs/benchmark/v08b-dev/probes.json --out <dir>

Each probe starts `dreamteam agent --stdio` on a V0.7b solvency scenario
and seed, and plays scripted decision points through the existing agent
protocol: no new action, no model, no network. A scripted decision is
validated and submitted the way the LLM agent submits its own
(agents/llm/llm_agent.py): its actions split into an operations batch and
a staffing batch, both validated, and both applied only if both are valid;
a refused attempt is followed by the next scripted attempt, and a decision
whose last attempt is refused falls back to no action.

For every point it records the public observation of funding (cash,
current liabilities, reserved commitments, headroom, credit), every
validate, preview and apply answer, and whether each action's code is the
one the probe expects. It writes:
* probe-log.json: everything, and the expectations met or not;
* <probe>.decisions.jsonl: the decisions in the llm-agent 2 recording
  format, for the frozen decision evidence rules;
* <probe>.attempts.jsonl: every attempt with its own reason and actions,
  the attempt-level evidence the historical format lacks.
The exit status is 1 if any expectation is not met.
"""

import argparse
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, "..", "..", "clients", "python"))
sys.path.insert(0, os.path.join(HERE, "..", "..", "agents", "llm"))
from dreamteam_agent import AgentClient, ProtocolError  # noqa: E402
import llm_agent as A  # noqa: E402  (split: the agent's batching, used read-only)

SCHEMA = "dreamteam.funding-probes/v1"
LOG_SCHEMA = "dreamteam.funding-probe-log/v1"


def funding(obs):
    """The public observation of funding at a decision point."""
    f = obs.get("finance") or {}
    s = (f.get("financing") or {}).get("solvency") or {}
    return {"time": obs.get("time"), "weekday": obs.get("weekday"), "cashCents": f.get("cashCents"),
            "currentLiabilitiesCents": s.get("currentLiabilitiesCents"), "reservedCents": s.get("reservedCents"),
            "fundingHeadroomCents": s.get("fundingHeadroomCents"), "creditLimitCents": s.get("creditLimitCents"),
            "creditDrawnCents": s.get("creditDrawnCents"), "creditAvailableCents": s.get("creditAvailableCents")}


def obs_sha(obs):
    return hashlib.sha256(json.dumps(obs, sort_keys=True, separators=(",", ":")).encode("utf-8")).hexdigest()


def results_of(answer):
    return [{"ok": bool(r.get("ok")), "code": r.get("code"), "message": r.get("message"),
             **({"id": r["id"]} if r.get("id") else {})} for r in (answer or {}).get("results") or []]


class Probe:
    def __init__(self, c):
        self.c = c
        self.unmet = []

    def observe(self):
        o = self.c.request("observe", {"compact": True, "recentLog": 0})["observation"]
        return o

    def validate(self, actions):
        """validate (and, for staffing actions, preview's funding) of one batch."""
        v = self.c.request("validate", {"actions": actions})
        out = {"actions": actions, "valid": bool(v.get("valid")), "results": results_of(v)}
        if all(a["type"] in A.STAFFING for a in actions):
            try:
                p = self.c.request("preview", {"actions": actions})["preview"]
                if p.get("funding") is not None:
                    out["funding"] = p["funding"]
            except ProtocolError as e:
                out["previewError"] = e.code
        return out

    def expect(self, where, got, want):
        ok = got == want
        if not ok:
            self.unmet.append(f"{where}: expected {want}, got {got}")
        return {"expected": want, "got": got, "met": ok}

    def check(self, where, chk):
        vs = [self.validate(b) for b, _ in A.split(chk["actions"])]
        codes = [r["code"] for v in vs for r in v["results"]]
        return {"probe": chk["probe"], "label": chk["label"], "validate": vs,
                "expectation": self.expect(where, codes, chk["expect"])}


def play_sequence(p, probe):
    """The scripted decision points: (log points, recording, attempts)."""
    points, recording, attempt_log = [], [], []
    for n, pt in enumerate(probe["points"]):
        p.c.request("advance", {"to": pt["time"]})
        clock = p.c.request("digests")
        obs = p.observe()
        entry = {"point": n, "time": pt["time"], "probes": pt["probes"], "funding": funding(obs),
                 "observationSha256": obs_sha(obs), "checksBefore": [], "attempts": [], "checksAfter": []}
        for chk in pt.get("checksBefore") or []:
            entry["checksBefore"].append(p.check(f"{probe['id']} point {n} check before", chk))
        refusals, errors, outcome, final = [], [], "fallback", None
        for k, at in enumerate(pt["attempts"], 1):
            rec = {"attempt": k, "reason": at["reason"], "actions": at["actions"], "validation": [], "applied": []}
            if not at["actions"]:
                outcome, final = "no_action", rec
                rec["expectation"] = p.expect(f"{probe['id']} point {n} attempt {k}", [], at["expect"])
                entry["attempts"].append(rec)
                break
            parts = A.split(at["actions"])
            for batch, idx in parts:
                v = p.validate(batch)
                rec["validation"].append(v)
                for j, r in enumerate(v["results"]):
                    if not r["ok"]:
                        refusals.append({"attempt": k, "action": idx[j], "type": batch[j]["type"], "code": r["code"],
                                         "message": r["message"]})
            if all(v["valid"] for v in rec["validation"]):
                for batch, _ in parts:
                    ap = p.c.request("apply", {"actions": batch})
                    rec["applied"].append({"actions": batch, "results": results_of(ap)})
                codes = [r["code"] for b in rec["applied"] for r in b["results"]]
                outcome, final = "actions", rec
            else:
                codes = [r["code"] for v in rec["validation"] for r in v["results"]]
                bad = [r for r in refusals if r["attempt"] == k]
                errors.append(f"attempt {k}: refused: " + "; ".join(
                    f"action {r['action']}: {r['code']}: {r['message']}" for r in bad))
                final = rec
            rec["expectation"] = p.expect(f"{probe['id']} point {n} attempt {k}", codes, at["expect"])
            entry["attempts"].append(rec)
            if outcome == "actions":
                break
        entry["outcome"] = outcome
        for chk in pt.get("checksAfter") or []:
            entry["checksAfter"].append(p.check(f"{probe['id']} point {n} check after", chk))
        points.append(entry)
        recording.append({
            "point": n, "time": pt["time"], "day": obs.get("day"), "digests": clock.get("digests"), "mode": "probe",
            "attempts": len(entry["attempts"]), "errors": errors, "refusals": refusals, "outcome": outcome,
            "reason": final["reason"] if final else None, "proposed": final["actions"] if final else None,
            "validation": [{"valid": v["valid"], "results": [dict(r, action=a) for r, a in zip(v["results"], v["actions"])]}
                           for v in final["validation"]] if final and final["validation"] else None,
            "batches": [b for b, _ in A.split(final["actions"])] if outcome == "actions" else []})
        attempt_log.append({"point": n, "time": pt["time"], "outcome": outcome,
                            "attempts": [{"attempt": a["attempt"], "reason": a["reason"], "actions": a["actions"],
                                          "validation": [{"valid": v["valid"], "results": v["results"]}
                                                         for v in a["validation"]],
                                          "applied": a["applied"]} for a in entry["attempts"]]})
    return points, recording, attempt_log


def boundary(p, probe):
    """Probe J: the largest value of the field that the headroom funds, by
    bisection with validate, and the next value. The varied action follows
    any fixed prefix actions in one batch."""
    b = probe["boundary"]
    p.c.request("advance", {"to": b["time"]})
    obs = p.observe()

    def ok(v):
        a = dict(b["action"], **{b["field"]: v})
        return p.validate(list(b.get("prefix") or []) + [a])

    lo, hi = b["from"], b["to"]
    if not ok(lo)["valid"]:
        p.unmet.append(f"{probe['id']}: the smallest value {lo} is not funded")
        return {"funding": funding(obs)}
    if ok(hi)["valid"]:
        return {"funding": funding(obs), "funded": hi, "note": "every value up to the maximum is funded"}
    while hi - lo > 1:
        mid = (lo + hi) // 2
        if ok(mid)["valid"]:
            lo = mid
        else:
            hi = mid
    below, above = ok(lo), ok(hi)
    head = (below.get("funding") or {}).get("headroomCents")
    cb = (below.get("funding") or {}).get("commitmentCents")
    ca = (above.get("funding") or {}).get("commitmentCents")
    holds = isinstance(head, int) and isinstance(cb, int) and isinstance(ca, int) and cb <= head < ca
    # The varied action is the batch's last: its own code decides.
    exp = p.expect(f"{probe['id']} boundary", [below["results"][-1]["code"], above["results"][-1]["code"], holds],
                   ["accepted", "insufficient_funds", True])
    return {"probe": b["probe"], "funding": funding(obs), "field": b["field"], "lastFunded": lo, "firstRefused": hi,
            "below": below, "above": above, "commitmentBelowCents": cb, "commitmentAboveCents": ca,
            "headroomCents": head, "expectation": exp,
            "checks": [p.check(f"{probe['id']} check", chk) for chk in probe.get("checks") or []]}


def run(binary, probes_path, out):
    with open(probes_path, encoding="utf-8") as f:
        spec = json.load(f)
    if spec.get("schema") != SCHEMA:
        raise SystemExit(f"{probes_path}: not a {SCHEMA} file")
    os.makedirs(out, exist_ok=True)
    log = {"schema": LOG_SCHEMA, "probes": spec["id"],
           "probesSha256": hashlib.sha256(open(probes_path, "rb").read()).hexdigest(), "runs": []}
    unmet = []
    for probe in spec["probes"]:
        with AgentClient(binary, ["-scenario", probe["scenario"], "-seed", str(probe["seed"])]) as c:
            hello = c.request("hello")
            p = Probe(c)
            entry = {"id": probe["id"], "scenario": probe["scenario"], "seed": probe["seed"],
                     "engine": hello.get("engine"), "fingerprint": (hello.get("scenario") or {}).get("fingerprint")}
            if "points" in probe:
                points, recording, attempts = play_sequence(p, probe)
                entry["points"] = points
                for name, rows in (("decisions", recording), ("attempts", attempts)):
                    with open(os.path.join(out, f"{probe['id']}.{name}.jsonl"), "w", encoding="utf-8") as f:
                        for r in rows:
                            f.write(json.dumps(r, separators=(",", ":"), sort_keys=True) + "\n")
            if "boundary" in probe:
                entry["boundary"] = boundary(p, probe)
            entry["unmet"] = p.unmet
            unmet += p.unmet
        log["runs"].append(entry)
    log["expectationsUnmet"] = unmet
    with open(os.path.join(out, "probe-log.json"), "w", encoding="utf-8") as f:
        json.dump(log, f, indent=1, sort_keys=True, ensure_ascii=True)
        f.write("\n")
    return log


def main(argv=None):
    ap = argparse.ArgumentParser(description="Run the deterministic funding-refusal probes (no model).")
    ap.add_argument("--binary", required=True, help="the dreamteam executable")
    ap.add_argument("--probes", required=True, help="a dreamteam.funding-probes/v1 file")
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    log = run(a.binary, a.probes, a.out)
    for u in log["expectationsUnmet"]:
        print("UNMET:", u)
    print(f"funding probes: {len(log['runs'])} probes, {len(log['expectationsUnmet'])} expectations unmet")
    return 1 if log["expectationsUnmet"] else 0


if __name__ == "__main__":
    sys.exit(main())
