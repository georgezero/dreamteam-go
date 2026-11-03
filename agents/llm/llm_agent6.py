#!/usr/bin/env python3
"""The DreamTeam LLM agent with high-impact action acknowledgment
(llm-agent, version 6).

An external benchmark policy (docs/BENCHMARK.md §4). Specification:
docs/V0.12_ACKNOWLEDGMENT.md (the workflow llm-agent-acknowledgment-
workflow/v1 and the contract dreamteam.action-acknowledgment/v1); the
engine's consequence inventory: docs/V0.12_CONSEQUENCE_INVENTORY.md.

Version 6 is version 5's validation-first workflow (agents/llm/
llm_agent5.py, loaded unchanged, with versions 3 and 4 beneath it) and one
added safeguard, at the agent workflow layer only:
1. the model proposes (version 3's prompt, memory and parser, unchanged;
   a reply may also carry "acknowledgment");
2. every batch is validated; a refusal is answered with the simulator's
   results (version 5's validation feedback, unchanged);
3. a fully valid proposal with staffing actions has its staffing batch
   previewed, and the consequence items are read from the engine's answer
   (consequences()): each termination, each new retained obligation (a
   hire; a teleradiology contract start) and the batch's displaced bookings,
   with their exact engine-reported values. None: the proposal is submitted
   as validated;
4. otherwise the model is shown the items (llm-agent-acknowledgment-
   request/v1) and must acknowledge each one: its action, type, target and
   every consequence value exactly, for this proposal (its token). A
   generic reply, another action, a wrong value or another proposal's
   acknowledgment is not accepted (match()); the failures are shown and the
   model may acknowledge again, revise (validated and assessed again, with
   a new request) or submit nothing, within maxRetries;
5. an accepted acknowledgment is followed by a final check: every batch
   validated again, the staffing batch previewed again and its items
   recomputed; both must equal what was acknowledged, with the state
   unchanged. Only then is the proposal submitted.
Fail-closed: a proposal that needs acknowledgment is never submitted
without an accepted one; if the consequences cannot be read, or the final
check disagrees, or the retries are spent, nothing is submitted and what
was withheld is recorded with the reason. The proposal is never rewritten:
its operations batch is not submitted on its own.

Acknowledgment is not approval. It records that the model stated the
engine's reported direct consequences of the exact proposal; it does not
establish that the action is clinically or financially wise, and the
engine's direct consequences are not all future outcomes. A valid action is
never refused because its consequences are large.

The record is dreamteam.llm-decision-record/v6. Replay (--replay FILE ...)
runs the same workflow again against the engine, with each recorded model
reply in place of the model, and requires the identical record: every
validation, preview, consequence item, payload hash, message, match and
final check is recomputed, and any difference is refused.
"""

import argparse
import hashlib
import importlib.util
import json
import os
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
BASE_PATH = os.path.join(HERE, "llm_agent3.py")
PREVIEW_PATH = os.path.join(HERE, "llm_agent4.py")
WORKFLOW5_PATH = os.path.join(HERE, "llm_agent5.py")


def _load(name, path):
    m = sys.modules.get(name)
    if m is None:
        spec = importlib.util.spec_from_file_location(name, path)
        m = importlib.util.module_from_spec(spec)
        sys.modules[name] = m
        spec.loader.exec_module(m)
    return m


A3 = _load("llm_agent3", BASE_PATH)
A4 = _load("llm_agent4", PREVIEW_PATH)
A5 = _load("llm_agent5", WORKFLOW5_PATH)

AGENT = A3.AGENT
VERSION = "6"
BASE_VERSION = A3.VERSION
PROMPT_VERSION = A3.PROMPT_VERSION
MEMORY_FORMAT = A3.MEMORY_FORMAT
EVIDENCE_SCHEMA = "dreamteam.llm-agent-evidence/v6"
RECORD_SCHEMA = "dreamteam.llm-decision-record/v6"
CONTRACT = "dreamteam.action-acknowledgment/v1"
WORKFLOW_VERSION = "llm-agent-acknowledgment-workflow/v1"
REQUEST_VERSION = "llm-agent-acknowledgment-request/v1"
STAFFING = A3.STAFFING
ProtocolError, ExchangeError, ReplayError, SettingsError = (A3.ProtocolError, A3.ExchangeError, A3.ReplayError,
                                                            A3.SettingsError)
sha256, compact = A3.sha256, A3.compact

# --- the contract (dreamteam.action-acknowledgment/v1) -------------------

# What qualifies, read from the engine's preview of a valid staffing batch.
# No dollar threshold; every other action is ordinary.
QUALIFYING = (
    ("termination", "a terminate action: its preview.terminations entry (employee, lastDay, noticeDays, "
                    "severanceCents, severancePayday)"),
    ("retained_obligation", "a hire (the guaranteed lines 'Recruiting fee' and 'Earliest lawful exit' of its offer, "
                            "matched by requestId and the engine's line order) or a contract_teleradiology start "
                            "(preview.teleradiology: guaranteedCents, guaranteedThrough, minimumTermEnd)"),
    ("displaced_bookings", "a staffing batch whose preview lists displacements: their number, for the whole batch, "
                           "never attributed to one action"),
)
# How an acknowledgment is matched to the proposal it answers.
MATCHING = (
    "only in a reply to the acknowledgment request for the same proposal: the reply's actions equal it exactly",
    "acknowledgment is an object with exactly 'proposal' (the request's token) and 'items' (a list)",
    "one entry per required item, no other: 'id', 'action' (an action item) or 'actions' (the batch item), "
    "'type', 'target', 'consequences', and no other key",
    "each value equals the engine-reported value exactly: the same JSON type and value",
    "a reply that changes the actions is a new proposal: validated, assessed and, if it qualifies, asked again",
)
CONTRACT_SHA256 = sha256("\n".join([CONTRACT] + [f"{k}: {v}" for k, v in QUALIFYING] + list(MATCHING)))

# The workflow's texts. REQUEST: {token} the proposal's token, {items} one
# compact JSON item per line. REJECTED: {failures} one compact JSON failure
# per line, then the request again.
REQUEST_TEMPLATE = (
    "ACKNOWLEDGMENT REQUIRED. Nothing has been submitted. The actions in your reply passed validation. They include "
    "high-impact actions, and the simulator reports these direct consequences of them, exactly as it computed them "
    "(proposal {token}):\n{items}\n\n"
    "These are direct consequences the simulator reports, not all future outcomes. Acknowledging them is not approval "
    "and does not establish that the actions are wise. To submit these actions, reply with one JSON object "
    "{{\"reason\": ..., \"actions\": [the same actions], \"acknowledgment\": {{\"proposal\": \"{token}\", \"items\": "
    "[...]}}}} with one entry for each item above, giving its \"id\", its \"action\" (or \"actions\"), \"type\", "
    "\"target\" and \"consequences\", with every value exactly as reported. Or reply with changed actions to revise "
    "your reply (validated and assessed again), or \"actions\": [] to submit nothing.")
REJECTED_TEMPLATE = (
    "ACKNOWLEDGMENT NOT ACCEPTED. Nothing has been submitted. What did not match:\n{failures}\n\n{request}")
WORKFLOW_SHA256 = sha256("\n".join([WORKFLOW_VERSION, CONTRACT, CONTRACT_SHA256, REQUEST_VERSION,
                                   A5.FEEDBACK_TEMPLATE, REQUEST_TEMPLATE, REJECTED_TEMPLATE]))


def same(a, b):
    """Exact JSON equality: the same types (an integer is not a float or a
    boolean) and values."""
    if type(a) is not type(b):
        return False
    if isinstance(a, dict):
        return a.keys() == b.keys() and all(same(a[k], b[k]) for k in a)
    if isinstance(a, list):
        return len(a) == len(b) and all(same(x, y) for x, y in zip(a, b))
    return a == b


# --- consequences --------------------------------------------------------

def consequences(staff, idx, answer):
    """The consequence items of a valid staffing batch, from the engine's
    preview answer: (items, problems). A problem means the consequences
    cannot be read reliably; the batch is then never submitted."""
    p = (answer or {}).get("preview") if isinstance(answer, dict) else None
    if not isinstance(p, dict):
        return [], ["the preview answer has no preview"]
    problems, items = [], []
    res = p.get("results") or []
    if p.get("ok") is not True or len(res) != len(staff) or not all(r.get("ok") for r in res):
        problems.append("the preview does not accept every action that validation accepted")
    terms = p.get("terminations") or []
    hires = p.get("hires") or []
    teles = p.get("teleradiology") or []
    guar = p.get("guaranteed") or []
    used_terms, tele_k = set(), 0
    for j, a in enumerate(staff):
        n, t = idx[j], a.get("type")
        if t == "terminate":
            m = [k for k, x in enumerate(terms) if x.get("employee") == a.get("employee")]
            if len(m) != 1 or m[0] in used_terms:
                problems.append(f"action {n}: no single termination entry for employee {a.get('employee')!r}")
                continue
            used_terms.add(m[0])
            x = terms[m[0]]
            items.append({"action": n, "type": t, "target": x["employee"],
                          "consequences": {"termination": {"lastDay": x.get("lastDay"),
                                                           "noticeDays": x.get("noticeDays")},
                                           "severance": {"severanceCents": x.get("severanceCents"),
                                                         "severancePayday": x.get("severancePayday")}},
                          "source": f"preview.terminations[{m[0]}]"})
        elif t == "hire":
            m = [k for k, x in enumerate(hires) if x.get("requestId") == a.get("requestId")]
            if len(m) != 1:
                problems.append(f"action {n}: no single hire entry for requestId {a.get('requestId')!r}")
                continue
            k, offer = m[0], hires[m[0]].get("offer")
            lines = guar[3 * k:3 * k + 3]
            want = [f"Recruiting fee · {offer}", f"Contract guarantee per full week · {offer}",
                    f"Earliest lawful exit · {offer}"]
            if len(hires) * 3 > len(guar) or [g.get("label") for g in lines] != want:
                problems.append(f"action {n}: the hire's guaranteed lines are not where the engine places them")
                continue
            items.append({"action": n, "type": t, "target": a.get("requestId"),
                          "consequences": {"retained_obligation": {"recruitingFeeCents": lines[0].get("cents"),
                                                                   "earliestExitCents": lines[2].get("cents")}},
                          "source": f"preview.hires[{k}]; preview.guaranteed[{3 * k}], [{3 * k + 2}]"})
        elif t == "contract_teleradiology":
            if tele_k >= len(teles):
                problems.append(f"action {n}: no teleradiology entry")
                continue
            x, k = teles[tele_k], tele_k
            tele_k += 1
            if x.get("op") != a.get("op") or (a.get("op") == "start" and (x.get("plan") != a.get("plan") or
                                                                          x.get("effective") != a.get("effective"))):
                problems.append(f"action {n}: the teleradiology entry is not this action's")
                continue
            if a.get("op") == "start":
                items.append({"action": n, "type": t, "target": x.get("plan"),
                              "consequences": {"retained_obligation": {
                                  "guaranteedCents": x.get("guaranteedCents"),
                                  "guaranteedThrough": x.get("guaranteedThrough"),
                                  "minimumTermEnd": x.get("minimumTermEnd")}},
                              "source": f"preview.teleradiology[{k}]"})
    if len(used_terms) != len(terms):
        problems.append("the preview lists terminations that are not the batch's")
    disp = p.get("displacements") or []
    if disp:
        items.append({"actions": list(idx), "type": "staffing_batch", "target": "staffing batch",
                      "consequences": {"displaced_bookings": {"count": len(disp)}},
                      "source": "preview.displacements (the whole batch)"})
    for k, it in enumerate(items, 1):
        it["id"] = f"C{k}"
    return [{"id": it["id"], **{f: v for f, v in it.items() if f != "id"}} for it in items], problems


def shown(item):
    """An item as the model sees it (without its source)."""
    return {k: v for k, v in item.items() if k != "source"}


def payload_of(actions, items):
    """The acknowledged payload: the exact proposal and its items. Its
    hash (of canonical JSON: sorted keys, so the same proposal written with
    its keys in another order is the same proposal) identifies the
    proposal; the token is its first 16 hex digits."""
    body = {"contract": CONTRACT, "proposal": actions, "items": [shown(i) for i in items]}
    s = sha256(json.dumps(body, sort_keys=True, separators=(",", ":"), ensure_ascii=False))
    return {"sha256": s, "token": s[:16]}


def assess(h, actions, clock):
    """A fully validated proposal's consequences: the staffing batch's
    preview and its items. status: ordinary (nothing to acknowledge),
    qualifying, or unavailable (the consequences cannot be read: the
    proposal is withheld)."""
    staff, idx = A4.staffing_part(actions)
    out = {"actions": staff, "indices": idx, "status": None, "answer": None, "answerSha256": None, "error": None,
           "items": [], "problems": [], "payload": None, "digestsUnchanged": None}
    if not staff:
        out["status"] = "ordinary"
        return out
    try:
        ans = h.request("preview", {"actions": staff})
    except ProtocolError as e:
        out["status"] = "unavailable"
        out["error"] = {"code": e.code, "message": A3.clip(e.message or "", A3.MAX_MESSAGE)}
    else:
        out["answer"], out["answerSha256"] = ans, sha256(compact(ans))
        out["items"], out["problems"] = consequences(staff, idx, ans)
        if out["problems"]:
            out["status"] = "unavailable"
        elif out["items"]:
            out["status"] = "qualifying"
            out["payload"] = payload_of(actions, out["items"])
        else:
            out["status"] = "ordinary"
    out["digestsUnchanged"] = h.request("digests").get("digests") == clock["digests"]
    if not out["digestsUnchanged"]:
        out["status"] = "unavailable"
        out["problems"].append("the state changed during the preview")
    return out


def request_text(assessment):
    return REQUEST_TEMPLATE.format(token=assessment["payload"]["token"],
                                   items="\n".join(compact(shown(i)) for i in assessment["items"]))


def rejected_text(failures, assessment):
    return REJECTED_TEMPLATE.format(failures="\n".join(compact(f) for f in failures),
                                    request=request_text(assessment))


# --- matching ------------------------------------------------------------

def match(ack, assessment, earlier_tokens=()):
    """An acknowledgment checked against the exact proposal it answers:
    {"accepted", "failures"}. Every failure is listed; nothing is
    accepted unless there is none."""
    f = []
    items = assessment["items"]
    token = assessment["payload"]["token"]
    if ack is None:
        return {"accepted": False, "failures": [{"code": "missing", "message": "no \"acknowledgment\" in the reply"}]}
    if not isinstance(ack, dict) or set(ack) != {"proposal", "items"} or not isinstance(ack.get("items"), list):
        return {"accepted": False, "failures": [{"code": "malformed", "message": "\"acknowledgment\" must be an "
                                                 "object with exactly \"proposal\" and \"items\" (a list)"}]}
    if ack["proposal"] != token:
        code = "stale_proposal" if ack["proposal"] in earlier_tokens else "wrong_proposal"
        f.append({"code": code, "expected": token, "got": ack["proposal"]})
    by_id, seen = {i["id"]: i for i in items}, set()
    for k, e in enumerate(ack["items"]):
        if not isinstance(e, dict) or not isinstance(e.get("id"), str):
            f.append({"code": "malformed_entry", "entry": k})
            continue
        if e["id"] not in by_id:
            f.append({"code": "unknown_item", "item": e["id"]})
            continue
        if e["id"] in seen:
            f.append({"code": "duplicate_item", "item": e["id"]})
            continue
        seen.add(e["id"])
        it = by_id[e["id"]]
        where = "action" if "action" in it else "actions"
        allowed = {"id", where, "type", "target", "consequences"}
        if set(e) != allowed:
            f.append({"code": "malformed_entry", "item": e["id"], "expected": sorted(allowed), "got": sorted(e)})
            continue
        if not same(e[where], it[where]):
            f.append({"code": "wrong_action", "item": e["id"], "expected": it[where], "got": e[where]})
        if not same(e["type"], it["type"]):
            f.append({"code": "wrong_type", "item": e["id"], "expected": it["type"], "got": e["type"]})
        if not same(e["target"], it["target"]):
            f.append({"code": "wrong_target", "item": e["id"], "expected": it["target"], "got": e["target"]})
        c, want = e["consequences"], it["consequences"]
        if not isinstance(c, dict) or set(c) != set(want):
            f.append({"code": "wrong_consequence_types", "item": e["id"], "expected": sorted(want),
                      "got": sorted(c) if isinstance(c, dict) else c})
            continue
        for kind, vals in want.items():
            got = c[kind]
            if not isinstance(got, dict) or set(got) != set(vals):
                f.append({"code": "wrong_consequence_fields", "item": e["id"], "consequence": kind,
                          "expected": sorted(vals), "got": sorted(got) if isinstance(got, dict) else got})
                continue
            for field, v in vals.items():
                if not same(got[field], v):
                    f.append({"code": "wrong_value", "item": e["id"], "field": f"{kind}.{field}", "expected": v,
                              "got": got[field]})
    for i in items:
        if i["id"] not in seen and not any(x.get("item") == i["id"] for x in f):
            f.append({"code": "missing_item", "item": i["id"]})
    return {"accepted": not f, "failures": f}


# --- parsing -------------------------------------------------------------

def parse_reply(content, types, details_allowed):
    """Version 3's decision parser, with an optional "acknowledgment" taken
    out first: (decision, acknowledgment present, acknowledgment)."""
    try:
        d = json.loads(A3.strip_wrapping(content))
    except json.JSONDecodeError:
        d = None
    if isinstance(d, dict) and "acknowledgment" in d:
        ack = d.pop("acknowledgment")
        return A3.parse_decision(json.dumps(d), types, details_allowed), True, ack
    return A3.parse_decision(content, types, details_allowed), False, None


# --- the record ----------------------------------------------------------

def stages(attempts, batches, outcome):
    """proposed, validated, refused, previewed (the consequences read),
    acknowledgment requested, acknowledged, submitted. Executed is read
    at the next decision point (previousExecution)."""
    prop = [a["id"] for a in attempts if (a.get("parse") or {}).get("ok") and a.get("detailsRequested") is None]
    val = [a["id"] for a in attempts if a.get("validation") is not None and not a.get("refusals")
           and a.get("protocolError") is None and all((v.get("answer") or {}).get("valid") for v in a["validation"])]
    ref = [a["id"] for a in attempts if a.get("refusals")]
    pv = [a["id"] for a in attempts if (a.get("assessment") or {}).get("answer") is not None
          or (a.get("assessment") or {}).get("error") is not None]
    req = [a["id"] for a in attempts if (a.get("assessment") or {}).get("status") == "qualifying"]
    ack = [a["id"] for a in attempts if ((a.get("acknowledgment") or {}).get("match") or {}).get("accepted")]
    return {"proposed": prop, "validated": val, "refused": ref, "previewed": pv, "acknowledgmentRequested": req,
            "acknowledged": ack,
            "submitted": {"batches": len(batches), "actions": sum(len(b) for b in batches), "outcome": outcome}}


def withheld_of(at, reason, detail):
    """What a proposal that was not submitted would have submitted, and
    why: each batch, and whether it needed acknowledgment."""
    a = at.get("assessment") or {}
    need = {n for i in a.get("items") or [] for n in ([i["action"]] if "action" in i else i["actions"])}
    out = []
    for b, idx in A3.split(at["actions"]):
        staffing = b[0]["type"] in STAFFING
        out.append({"actions": b, "indices": idx, "kind": "staffing" if staffing else "operations",
                    "qualifying": bool(staffing and (need or a.get("status") == "unavailable")),
                    "items": [i["id"] for i in a.get("items") or []] if staffing else []})
    return {"attempt": at["attempt"], "reason": reason, "detail": detail, "batches": out,
            "note": "the whole proposal is withheld: its other batch is not submitted on its own"}


def completeness(attempts, stopped, cut, recorded=True):
    c = A3.completeness(attempts, stopped, cut, recorded)
    c["validationsRecorded"] = sum(len(a.get("validation") or []) for a in attempts)
    c["assessmentsRecorded"] = sum(1 for a in attempts if a.get("assessment"))
    c["acknowledgmentsRecorded"] = sum(1 for a in attempts if a.get("acknowledgment"))
    return c


def new_record(point, clock, mode, mem, prompt, left, pset):
    rec = A3.new_record(point, clock, mode, mem, prompt, left, pset)
    rec.update({"schema": RECORD_SCHEMA, "agentVersion": VERSION, "baseAgentVersion": BASE_VERSION,
                "workflowVersion": WORKFLOW_VERSION, "workflowSha256": WORKFLOW_SHA256,
                "contract": CONTRACT, "contractSha256": CONTRACT_SHA256})
    return rec


# --- one decision point --------------------------------------------------

class LiveModel:
    """The model server, and the decision's time budget."""

    def __init__(self, client, settings):
        self.client, self.settings, self.start = client, settings, time.monotonic()

    def payload(self, messages):
        return self.client.payload(messages)

    def call(self, payload):
        return self.client.call(payload)

    def spent(self, attempts):
        return bool(attempts) and time.monotonic() - self.start >= self.settings["decisionBudgetSeconds"]


def final_check(h, actions, pending, clock):
    """Step 5: every batch validated again and the consequences read again;
    both must equal what was acknowledged, with the state unchanged."""
    out = {"validation": [], "assessmentSha256": None, "payloadSha256": None, "digestsUnchanged": None,
           "agrees": False, "differences": []}
    for batch_no, (batch, idx) in enumerate(A3.split(actions)):
        try:
            v = h.request("validate", {"actions": batch})
        except ProtocolError as e:
            out["differences"].append(f"batch {batch_no}: validate answered {e.code}")
            v = None
        out["validation"].append({"actions": batch, "indices": idx, "answer": v})
    before = [v["answer"] for v in pending["validation"]]
    if [v["answer"] for v in out["validation"]] != before:
        out["differences"].append("the validate answers differ from those the acknowledgment followed")
    again = assess(h, actions, clock)
    out["assessmentSha256"] = again["answerSha256"]
    out["payloadSha256"] = (again["payload"] or {}).get("sha256")
    if again["status"] != "qualifying" or again["payload"] != pending["assessment"]["payload"] \
            or again["answerSha256"] != pending["assessment"]["answerSha256"]:
        out["differences"].append("the consequences differ from those acknowledged")
    out["digestsUnchanged"] = h.request("digests").get("digests") == clock["digests"]
    if not out["digestsUnchanged"]:
        out["differences"].append("the state changed")
    out["agrees"] = not out["differences"]
    return out


def ask(h, ev, model, settings, memory, point, prev=None):
    """One decision point (live, or replayed with the recorded replies as
    the model): the batches to submit and the record."""
    clock = h.request("digests")
    pset = A3.prompt_settings(settings)
    obs, omitted, catalog, mem, prompt, left = A3.prepare(h, memory, point, pset)
    rec = new_record(point, clock, "live", mem, prompt, left, pset)
    rec["retryBudget"] = {"maxRetries": settings["maxRetries"],
                          "decisionBudgetSeconds": settings["decisionBudgetSeconds"]}
    attempts, cut, stopped, used = [], [], None, None
    rec.update({"attempts": attempts, "final": None, "batches": [], "outcome": "fallback", "reason": None,
                "refusals": [], "errors": [], "withheld": None, "previousExecution": A5.execution_of(obs, prev)})
    ev.append("memory.jsonl", {"point": point, "sha256": rec["memorySha256"], "chars": len(mem), "text": mem})
    pending, tokens, last_failure = None, [], None  # pending: the attempt awaiting acknowledgment
    if prompt is None:
        rec["errors"].append(f"prompt_too_large: over {settings['maxPromptChars']} characters")
        stopped = "prompt_too_large"
    else:
        types = {t["type"] for t in catalog.get("types", [])}
        messages = [{"role": "system", "content": settings["systemPrompt"]}, {"role": "user", "content": prompt}]
        failures, details_open, turn = 0, settings["allowDetails"], "first"
        inp = {"kind": "prompt", "sha256": sha256(prompt), "chars": len(prompt)}
        while failures <= settings["maxRetries"]:
            if model.spent(attempts):
                rec["errors"].append(f"budget: {settings['decisionBudgetSeconds']} s spent; no further attempt")
                stopped = "budget"
                break
            k = len(attempts) + 1
            at = A3.new_attempt(point, k, turn, inp)
            at.update({"assessment": None, "acknowledgment": None, "finalCheck": None})
            payload = model.payload(messages)
            t0 = time.monotonic()
            status, body, err = model.call(payload)
            # The model call's evidence (model.jsonl), not the record: its
            # latency is not replayed.
            call = {"point": point, "attempt": k, "turn": turn, "model": settings["model"],
                    "endpoint": settings["endpoint"], "request": payload, "status": status,
                    "latencyMs": round((time.monotonic() - t0) * 1000), "response": body[:A3.MAX_RESPONSE_KEPT],
                    "responseTruncated": len(body) > A3.MAX_RESPONSE_KEPT, "usage": None, "servedModel": None,
                    "content": None, "error": err}
            at["transport"] = "ok" if err is None else err.split(":", 1)[0]
            asked, request, refused, stop, detail = None, None, [], None, None
            if err is None:
                try:
                    content, call["usage"], call["servedModel"] = A3.completion(body)
                    call["content"] = content
                    at["output"] = {"sha256": sha256(content), "chars": len(content), "text": content}
                    try:
                        d, has_ack, ack = parse_reply(content, types, details_open)
                    except ValueError as e:
                        at["parse"] = {"ok": False, "code": str(e).split(":", 1)[0],
                                       "message": A3.clip(str(e), A3.MAX_MESSAGE)}
                        raise
                    at["parse"] = {"ok": True}
                    at["reason"], at["actions"] = d["reason"], d["actions"]
                    rec["reason"] = d["reason"]
                    if "details" in d:
                        asked = at["detailsRequested"] = d["details"]
                    elif pending is not None and d["actions"] and same(d["actions"], pending["actions"]):
                        # The answer to the acknowledgment request for this proposal.
                        m = match(ack if has_ack else None, pending["assessment"], tokens[:-1])
                        at["acknowledgment"] = {"forAttempt": pending["attempt"],
                                                "proposal": pending["assessment"]["payload"]["token"],
                                                "present": has_ack, "given": ack, "match": m}
                        if not m["accepted"]:
                            last_failure = "acknowledgment_not_accepted"
                            raise ValueError("acknowledgment_not_accepted: " + "; ".join(
                                x["code"] + (f" ({x['item']})" if "item" in x else "") for x in m["failures"]))
                        at["finalCheck"] = fc = final_check(h, pending["actions"], pending, clock)
                        at["validation"] = fc["validation"]
                        if not fc["agrees"]:
                            stop, detail = "final_check_disagrees", fc["differences"]
                    else:
                        if has_ack:  # never accepted: no request for this proposal was made
                            at["acknowledgment"] = {
                                "forAttempt": None, "proposal": None, "present": True, "given": ack,
                                "match": {"accepted": False, "failures": [{
                                    "code": "proposal_changed" if pending is not None else "not_requested",
                                    "message": "an acknowledgment counts only in a reply to the request for the "
                                               "same proposal"}]}}
                        revising, pending = pending is not None, None
                        if d["actions"]:
                            refused = A5.validate_all(h, at, d["actions"], k)
                        else:
                            at["validation"] = []
                        if refused:
                            at["refusals"] = refused
                            rec["refusals"] += refused
                            last_failure = "refused"
                            raise ValueError("refused: " + "; ".join(
                                f"action {r['action']}: {r['code']}: {r['message']}" for r in refused))
                        if d["actions"]:
                            at["assessment"] = a = assess(h, d["actions"], clock)
                            if a["status"] == "unavailable":
                                stop, detail = "consequences_unavailable", a["problems"] or [a["error"]]
                            elif a["status"] == "qualifying" and revising and failures + 1 > settings["maxRetries"]:
                                # A revision in answer to a request uses the retry budget; none is left.
                                failures += 1
                                stop, detail = "acknowledgment_not_accepted", [
                                    "the proposal was revised in answer to an acknowledgment request, and no retry "
                                    "is left to request acknowledgment of the revision"]
                            elif a["status"] == "qualifying":
                                failures += 1 if revising else 0
                                pending = at
                                tokens.append(a["payload"]["token"])
                                request = request_text(a)
                except ProtocolError as e:
                    call["error"] = f"invalid_action: validate answered {e.code}: {e.message}"
                except ValueError as e:
                    call["error"] = str(e)
                    if at["output"] is None:
                        at["transport"] = str(e).split(":", 1)[0]
            ev.call(call)
            if call["error"] is not None:
                at["error"] = call["error"]
            elif stop is not None:
                at["error"] = stop
            elif asked is None and request is None:
                at["used"] = True
            attempts.append(at)
            ev.append("attempts.jsonl", at)
            if stop is not None:
                stopped = stop
                rec["errors"].append(f"attempt {k}: {stop}")
                rec["withheld"] = withheld_of(at, stop, detail)
                break
            if asked is not None:
                details_open = False
                text = A3.details_text(h, asked, obs, omitted, catalog, settings["maxPromptChars"] // 4)
                messages = messages + [{"role": "assistant", "content": call["content"]},
                                       {"role": "user", "content": text}]
                turn, inp = "details", {"kind": "details", "sha256": sha256(text), "chars": len(text)}
                continue
            if request is not None:
                details_open = False
                messages = messages + [{"role": "assistant", "content": call["content"]},
                                       {"role": "user", "content": request}]
                turn, inp = "acknowledge", {"kind": "acknowledgment", "sha256": sha256(request), "chars": len(request),
                                            "proposal": pending["assessment"]["payload"]["token"]}
                continue
            if at["used"]:
                used = at
                break
            failures += 1
            turn = "retry"
            rec["errors"].append(f"attempt {k}: {call['error']}")
            details_open = False
            if at["refusals"]:
                text = A5.feedback_text(A5.refused_items(at["validation"]))
                kind = "validation"
            elif at["acknowledgment"] is not None and at["acknowledgment"]["forAttempt"] is not None:
                text = rejected_text(at["acknowledgment"]["match"]["failures"], pending["assessment"])
                kind = "acknowledgment_rejected"
            elif call["content"] is not None:
                text = A3.retry_text(call["error"])
                kind = "retry"
            else:
                inp = {"kind": "resend", "sha256": inp["sha256"], "chars": inp["chars"]}
                continue
            messages = messages + [{"role": "assistant", "content": call["content"]}, {"role": "user", "content": text}]
            inp = {"kind": kind, "sha256": sha256(text), "chars": len(text)}
        else:
            if pending is not None:
                stopped = "retries"
        if used is None and pending is not None and rec["withheld"] is None:
            rec["withheld"] = withheld_of(pending, "acknowledgment_not_accepted",
                                          "no accepted acknowledgment before the decision ended "
                                          f"({stopped or 'retries'}); "
                                          f"last failure: {last_failure or 'none'}")
    if used is not None:
        rec["batches"] = [b for b, _ in A3.split(used["actions"])] if used["actions"] else []
        rec["outcome"] = "actions" if rec["batches"] else "no_action"
    else:
        rec["batches"], rec["outcome"] = [], "fallback"
    after = h.request("digests").get("digests")
    rec["digestsBeforeSubmit"] = {"unchanged": after == clock["digests"]}
    if after != clock["digests"] and rec["batches"]:
        rec["errors"].append("state_changed: the digests changed during the decision; nothing is submitted")
        rec["withheld"] = withheld_of(used, "state_changed", "the digests changed during the decision")
        rec["batches"], rec["outcome"] = [], "fallback"
    rec["final"] = {"attempt": used["attempt"] if used else None, "outcome": rec["outcome"]}
    rec["stages"] = stages(attempts, rec["batches"], rec["outcome"])
    rec["completeness"] = completeness(attempts, stopped, cut)
    return rec["batches"], rec


# --- replay --------------------------------------------------------------

class RecordedModel:
    """A recorded decision's replies, in place of the model: each attempt's
    recorded output, or its recorded transport failure; the budget is spent
    where the recording says it was."""

    def __init__(self, r):
        self.r, self.k = r, 0
        self.atts = r.get("attempts") or []

    def payload(self, messages):
        return {"messages": messages}

    def call(self, payload):
        if self.k >= len(self.atts):
            raise ReplayError(f"decision point {self.r.get('point')}: the workflow asks for attempt {self.k + 1}, "
                              f"which the recording does not have")
        at = self.atts[self.k]
        self.k += 1
        out = at.get("output") if isinstance(at, dict) else None
        if isinstance(out, dict) and isinstance(out.get("text"), str):
            if sha256(out["text"]) != out.get("sha256"):
                raise ReplayError(f"decision point {self.r.get('point')}, attempt {self.k}: the reply's text is not "
                                  f"the one its hash records")
            body = json.dumps({"choices": [{"message": {"content": out["text"]}}]})
            return 200, body, None
        err = at.get("error") if isinstance(at, dict) else None
        if not isinstance(err, str):
            raise ReplayError(f"decision point {self.r.get('point')}, attempt {self.k}: no reply and no failure "
                              "recorded")
        return None, "", err

    def spent(self, attempts):
        return (bool(attempts) and len(attempts) == len(self.atts)
                and (self.r.get("completeness") or {}).get("stopped") == "budget")


class NullEvidence:
    """Replay keeps its own evidence; the re-run's calls are not model calls."""

    def __init__(self, ev):
        self.ev = ev

    def append(self, name, obj):
        if name != "attempts.jsonl":
            self.ev.append(name, obj)

    def call(self, call):
        pass


def load_recordings(paths):
    """Each recorded decisions file: (path, SHA-256, records). Only
    dreamteam.llm-decision-record/v6 records of this workflow and contract
    replay here; version 5 recordings replay with llm_agent5.py."""
    out = []
    for p in paths:
        try:
            with open(p, "rb") as f:
                raw = f.read()
            recs = [json.loads(l) for l in raw.decode("utf-8").splitlines() if l.strip()]
        except (OSError, UnicodeDecodeError, json.JSONDecodeError) as e:
            raise SettingsError(f"replay {p}: {e}")
        if not recs or any(not isinstance(r, dict) or "point" not in r or "digests" not in r
                           or not isinstance(r.get("batches"), list) for r in recs):
            raise SettingsError(f"replay {p}: not a recorded decisions file")
        for r in recs:
            got = (r.get("schema"), r.get("agentVersion"), r.get("memoryFormat"), r.get("promptVersion"))
            if got != (RECORD_SCHEMA, VERSION, MEMORY_FORMAT, PROMPT_VERSION):
                raise SettingsError(
                    f"replay {p}: version incompatible: decision point {r['point']} is not a {RECORD_SCHEMA} record "
                    f"of {AGENT} {VERSION}; version 5 recordings replay with agents/llm/llm_agent5.py, version 4 "
                    "with llm_agent4.py, version 3 with llm_agent3.py, versions 1 and 2 with llm_agent.py")
            ident = (r.get("workflowVersion"), r.get("workflowSha256"), r.get("contract"), r.get("contractSha256"))
            if ident != (WORKFLOW_VERSION, WORKFLOW_SHA256, CONTRACT, CONTRACT_SHA256) or \
                    r.get("workflow") != "acknowledgment":
                raise SettingsError(f"replay {p}: decision point {r['point']}: unknown workflow or contract identity "
                                    f"{ident[0]!r} {str(ident[1])[:12]}… {ident[2]!r} {str(ident[3])[:12]}…")
        out.append((p, hashlib.sha256(raw).hexdigest(), recs))
    return out


def first_difference(a, b, path="record"):
    """Where two JSON values first differ (for the replay's message)."""
    if type(a) is not type(b):
        return path
    if isinstance(a, dict):
        for k in sorted(set(a) | set(b)):
            if k not in a or k not in b:
                return f"{path}.{k}"
            d = first_difference(a[k], b[k], f"{path}.{k}")
            if d:
                return d
        return None
    if isinstance(a, list):
        if len(a) != len(b):
            return f"{path} (length)"
        for i, (x, y) in enumerate(zip(a, b)):
            d = first_difference(x, y, f"{path}[{i}]")
            if d:
                return d
        return None
    return None if a == b else path


def replay_point(h, ev, replayer, point, prev):
    """One replayed decision point: the workflow run again with the
    recorded replies; the record it makes must be the recording's."""
    clock = h.request("digests")
    r = replayer.next(point, clock)
    if r.get("protocolError") is not None:
        raise ReplayError(f"decision point {point}: a protocol error was recorded; it cannot be replayed")
    pset = r.get("promptSettings")
    if not isinstance(pset, dict) or set(pset) != set(A3.PROMPT_SETTINGS):
        raise ReplayError(f"decision point {point}: the recording has no usable promptSettings")
    if replayer.memory is None:
        replayer.memory = A3.Memory(pset["memoryDecisions"], pset["memoryMetricPoints"], pset["memoryMaxChars"])
    settings = replay_settings(r)
    _, again = ask(h, NullEvidence(ev), RecordedModel(r), settings, replayer.memory, point, prev)
    again["workflow"] = r.get("workflow")
    want = {k: v for k, v in r.items() if k != "mode"}
    got = {k: v for k, v in again.items() if k != "mode"}
    if not same(got, want):
        raise ReplayError(f"decision point {point}: the workflow re-run differs from the recording at "
                          f"{first_difference(got, want)}")
    rec = dict(r)
    rec.update({"mode": "replay", "replayed": os.path.basename(replayer.chosen[0]),
                "replayChecked": {"attempts": len(r["attempts"]),
                                  "assessments": r["completeness"]["assessmentsRecorded"],
                                  "acknowledgments": r["completeness"]["acknowledgmentsRecorded"]}})
    return rec


def replay_settings(r):
    """The settings a replayed decision runs with: the recorded prompt
    settings and retry budget. No model, endpoint or system prompt is used
    (the record holds no part of them)."""
    b = r.get("retryBudget")
    if not isinstance(b, dict) or set(b) != {"maxRetries", "decisionBudgetSeconds"}:
        raise ReplayError(f"decision point {r.get('point')}: the recording has no usable retryBudget")
    s = {"model": None, "endpoint": None, "systemPrompt": "", **b}
    s.update(r["promptSettings"])
    return s


def check_contract(hello):
    A4.check_contract(hello)


def summary(ev, records, settings):
    s = A3.summary(ev, records, settings)
    s["retryRequests"] = sum(1 for c in ev.calls if c.get("turn") == "retry")
    s["acknowledgmentRequests"] = sum(1 for c in ev.calls if c.get("turn") == "acknowledge")
    s["acknowledged"] = sum(len((r.get("stages") or {}).get("acknowledged") or []) for r in records)
    s["withheld"] = {}
    for r in records:
        w = r.get("withheld")
        if w:
            s["withheld"][w["reason"]] = s["withheld"].get(w["reason"], 0) + 1
    s["withheld"] = dict(sorted(s["withheld"].items()))
    return s


def usage_text(s):
    t = A3.usage_text(s).replace(f"({AGENT} {BASE_VERSION})", f"({AGENT} {VERSION})")
    return t + (f"acknowledgment        {s['acknowledgmentRequests']} requests; {s['acknowledged']} accepted; withheld "
                f"{', '.join(f'{k} {v}' for k, v in s['withheld'].items()) or 'none'}\n")


# --- main ----------------------------------------------------------------

def run(h, ev, client, settings, memory, replayer, records):
    checked, prev = False, None
    while True:
        msg = h.turn()
        if msg["turn"] == "end":
            if replayer:
                replayer.finish()
            return 0
        point = msg["point"]
        if not checked:
            check_contract(h.request("hello"))
            checked = True
        if replayer:
            rec = replay_point(h, ev, replayer, point, prev)
            batches = rec["batches"]
            memory = replayer.memory
        else:
            try:
                batches, rec = ask(h, ev, LiveModel(client, settings), settings, memory, point, prev)
            except ProtocolError as e:
                batches, rec = [], {"schema": RECORD_SCHEMA, "point": point, "time": None, "mode": "live",
                                    "agentVersion": VERSION, "baseAgentVersion": BASE_VERSION,
                                    "promptVersion": PROMPT_VERSION, "memoryFormat": MEMORY_FORMAT,
                                    "workflowVersion": WORKFLOW_VERSION, "workflowSha256": WORKFLOW_SHA256,
                                    "contract": CONTRACT, "contractSha256": CONTRACT_SHA256,
                                    "protocolError": str(e), "outcome": "fallback", "batches": [],
                                    "errors": [f"protocol: {e}"], "attempts": [], "withheld": None,
                                    "final": {"attempt": None, "outcome": "fallback"},
                                    "stages": stages([], [], "fallback"), "previousExecution": None,
                                    "completeness": completeness([], "protocol_error", [], recorded=False),
                                    "digests": None}
            rec["workflow"] = "acknowledgment"
        records.append(rec)
        if memory and rec.get("time") is not None:
            memory.record(rec)
        ev.append("decisions.jsonl", rec)
        prev = rec if rec.get("time") is not None else None
        yield
        h.decide(batches)


def main(argv=None):
    ap = argparse.ArgumentParser(
        description="The DreamTeam LLM agent with high-impact action acknowledgment (a benchmark policy).")
    ap.add_argument("--settings", help="settings file (dreamteam.llm-agent-settings/v1); required live")
    ap.add_argument("--replay", action="append", default=[], metavar="FILE",
                    help="re-run the decisions recorded in FILE (decisions.jsonl) with the recorded replies instead "
                         "of a model; repeatable")
    ap.add_argument("--evidence", default="evidence", help="evidence directory (default ./evidence)")
    a = ap.parse_args(argv)
    try:
        if a.replay:
            settings, recordings = None, load_recordings(a.replay)
        elif a.settings:
            settings, recordings = A3.load_settings(a.settings, os.environ), None
        else:
            raise SettingsError("give --settings (live) or --replay (replay)")
    except SettingsError as e:
        print(f"{AGENT}: {e}", file=sys.stderr)
        return 2
    ev = A3.Evidence(a.evidence)
    info = {"schema": EVIDENCE_SCHEMA, "agent": AGENT, "version": VERSION, "baseVersion": BASE_VERSION,
            "promptVersion": PROMPT_VERSION, "memoryFormat": MEMORY_FORMAT, "workflow": "acknowledgment",
            "workflowVersion": WORKFLOW_VERSION, "workflowSha256": WORKFLOW_SHA256, "contract": CONTRACT,
            "contractSha256": CONTRACT_SHA256, "requestVersion": REQUEST_VERSION,
            "feedbackVersion": A5.FEEDBACK_VERSION, "agentSha256": A5.file_sha256(os.path.abspath(__file__)),
            "baseAgentSha256": A5.file_sha256(BASE_PATH), "previewAgentSha256": A5.file_sha256(PREVIEW_PATH),
            "workflow5AgentSha256": A5.file_sha256(WORKFLOW5_PATH),
            "mode": "replay" if recordings else "live", "python": sys.version.split()[0]}
    if settings:
        info.update({k: settings[k] for k in ("model", "endpoint", "temperature", "maxOutputTokens",
                                               "requestTimeoutSeconds", "maxRetries", "decisionBudgetSeconds",
                                               "responseFormat", "systemPromptSource", "pricing", *A3.PROMPT_SETTINGS)})
        info["systemPromptSha256"] = sha256(settings["systemPrompt"])
        info["apiKey"] = "set" if settings["apiKey"] else "none"
    else:
        info["replay"] = [{"file": os.path.basename(p), "sha256": s} for p, s, _ in recordings]
    ev.write("agent.json", info)
    client = A3.ModelClient(settings) if settings else None
    replayer = A3.Replayer(recordings) if recordings else None
    memory = A3.Memory(settings["memoryDecisions"], settings["memoryMetricPoints"],
                       settings["memoryMaxChars"]) if settings else None
    h = A3.Harness(sys.stdin, sys.stdout)
    records = []

    def finish():
        s = summary(ev, records, settings)
        ev.write("summary.json", s)
        ev.text("usage.txt", usage_text(s))
    try:
        for _ in run(h, ev, client, settings, memory, replayer, records):
            finish()
        return 0
    except (ExchangeError, ReplayError) as e:
        print(f"{AGENT}: {e}", file=sys.stderr)
        return 1
    finally:
        finish()


if __name__ == "__main__":
    sys.exit(main())
