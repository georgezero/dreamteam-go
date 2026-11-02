#!/usr/bin/env python3
"""The DreamTeam LLM agent, validation-first (llm-agent, version 5).

An external benchmark policy (docs/BENCHMARK.md §4). Specification:
docs/LLM_AGENT.md §18 and docs/V0.11_AGENT_WORKFLOW.md.

Two workflows, chosen with --workflow (V0.11.0):
* validation-first (the default; llm-agent-validation-first-workflow/v1):
  steps 1-3 and 6 below, with no preview: a fully validated proposal is
  submitted;
* validation-first-preview (llm-agent-workflow/v1, unchanged since V0.11a):
  steps 1-6, with the selective preview.
They share the proposal, validation, refusal feedback, retries and
submission; the only difference is whether step 4 occurs. Every record
names its workflow; replay takes it from the recording. A record made
before the selector existed has llm-agent-workflow/v1 and no selection, and
replays with the preview, as it was made.

Version 5 is version 3 (agents/llm/llm_agent3.py) and version 4's preview
request (agents/llm/llm_agent4.py), both loaded unchanged from this
directory, in a different order:
1. the model proposes (version 3's prompt, memory and parser, unchanged);
2. every batch of the proposal is validated;
3. if any action is refused, the model is shown the simulator's results for
   the refused actions exactly as returned (llm-agent-validation-feedback/v1)
   and replies again: revise, or "actions": [] to submit nothing. A parse
   error is answered with version 3's retry message. Both count against
   maxRetries;
4. the first fully valid proposal with staffing actions has its staffing
   actions previewed (once per decision point); the preview is shown only
   when the eligibility rule (llm-agent-preview-eligibility/v1) finds a
   direct financial or capacity effect in the engine's own answer. The
   message (llm-agent-selective-preview/v1) says the proposal passed
   validation, that the preview gives direct commitments and effects, and
   that neither establishes the decision is beneficial;
5. the model keeps, revises or abandons; the reply is validated again, and a
   refusal goes back to step 3;
6. only a fully validated proposal is submitted, after a check that
   validation and preview changed no state. Otherwise nothing is.

The record is dreamteam.llm-decision-record/v5: every attempt as version 3
records it, each validate answer, the feedback message by SHA-256, the
preview request, answer, eligibility and message, the stages (proposed,
validated, previewed, submitted) and, read from the next decision point's
observation, how the previous point's batches were executed. Replay mode
(--replay FILE ...) checks all of it again without a model.
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


def _load(name, path):
    """A sibling agent module, unchanged: the copy already imported, or the
    file (python3 -I leaves this directory off sys.path)."""
    m = sys.modules.get(name)
    if m is None:
        spec = importlib.util.spec_from_file_location(name, path)
        m = importlib.util.module_from_spec(spec)
        sys.modules[name] = m
        spec.loader.exec_module(m)
    return m


A3 = _load("llm_agent3", BASE_PATH)
A4 = _load("llm_agent4", PREVIEW_PATH)

AGENT = A3.AGENT
VERSION = "5"
BASE_VERSION = A3.VERSION
PROMPT_VERSION = A3.PROMPT_VERSION
MEMORY_FORMAT = A3.MEMORY_FORMAT
EVIDENCE_SCHEMA = "dreamteam.llm-agent-evidence/v5"
RECORD_SCHEMA = "dreamteam.llm-decision-record/v5"
WORKFLOW_VERSION = "llm-agent-workflow/v1"
FEEDBACK_VERSION = "llm-agent-validation-feedback/v1"
PREVIEW_VERSION = "llm-agent-selective-preview/v1"
ELIGIBILITY_VERSION = "llm-agent-preview-eligibility/v1"
NEEDS = A4.NEEDS
STAFFING = A3.STAFFING
ProtocolError, ExchangeError, ReplayError, SettingsError = (A3.ProtocolError, A3.ExchangeError, A3.ReplayError,
                                                            A3.SettingsError)
sha256, compact = A3.sha256, A3.compact

# The workflow's only texts. FEEDBACK: {results} are the refused results,
# one compact JSON object per line, {"action": <number in the reply>,
# "result": <the simulator's result>}. PREVIEW: {indices} the staffing
# actions' numbers, {preview} the simulator's answer as compact JSON.
FEEDBACK_TEMPLATE = (
    "VALIDATION. Nothing has been submitted. The simulator refused actions in your reply. Its results for the "
    "refused actions, exactly as it returned them:\n{results}\n\n"
    "Reply with one JSON object {{\"reason\": ..., \"actions\": [...]}}: changed actions to revise your reply, or "
    "\"actions\": [] to submit nothing. Your reply is validated again before anything is submitted.")
PREVIEW_TEMPLATE = (
    "PREVIEW. The actions in your reply passed validation: the simulator would accept them now. Nothing has been "
    "submitted yet. This is the simulator's preview of the staffing actions in your reply (actions {indices}), "
    "exactly as the simulator returned it. It describes their direct commitments and effects:\n{preview}\n\n"
    "{ops_note}Passing validation and this preview do not establish that the decision is beneficial. Reply with your "
    "final decision, one JSON object {{\"reason\": ..., \"actions\": [...]}}: the same actions to keep it, changed "
    "actions to revise it, or \"actions\": [] to submit nothing. Your reply is validated again before anything is "
    "submitted.")
OPS_NOTE = "The operations actions in your reply also passed validation; the simulator has no preview for them.\n\n"
# The eligibility rule, in order: each reason is read from the engine's own
# preview answer for a valid batch; none uses an invented threshold.
ELIGIBILITY_RULE = (
    ("commitment", "funding.commitmentCents > 0: the engine's funding test measures a new commitment"),
    ("obligation", "a guaranteed money line whose 'when' does not say 'unchanged': a new or changed guaranteed "
                   "obligation (retainer, salary, severance, recruiting fee, earliest exit)"),
    ("termination", "terminations listed: severance, notice and lost capacity"),
    ("displacement", "displacements listed: booked appointments that lose coverage"),
    ("capacity", "slotsOpened, slotsClosed or hires listed: a change in bookable capacity or headcount"),
)
WORKFLOW_SHA256 = sha256("\n".join([WORKFLOW_VERSION, FEEDBACK_TEMPLATE, PREVIEW_TEMPLATE, OPS_NOTE] +
                                   [f"{k}: {v}" for k, v in ELIGIBILITY_RULE]))
# The validation-first workflow without the preview (V0.11.0): the same
# feedback text; the rule that replaces the preview step.
VALIDATION_FIRST_VERSION = "llm-agent-validation-first-workflow/v1"
VALIDATION_FIRST_RULE = "no preview: a fully validated proposal is submitted"
VALIDATION_FIRST_SHA256 = sha256("\n".join([VALIDATION_FIRST_VERSION, FEEDBACK_TEMPLATE, VALIDATION_FIRST_RULE]))
# The selectable workflows. llm-agent-workflow/v1 keeps its V0.11a identity.
WORKFLOWS = {
    "validation-first": {"version": VALIDATION_FIRST_VERSION, "sha256": VALIDATION_FIRST_SHA256, "preview": False},
    "validation-first-preview": {"version": WORKFLOW_VERSION, "sha256": WORKFLOW_SHA256, "preview": True},
}
DEFAULT_WORKFLOW = "validation-first"
LEGACY_WORKFLOW = "validation-first-preview"  # a record made before the selector names none


def workflow_of(r, error=None):
    """The workflow a record was made with, from its identity
    (workflowVersion, workflowSha256) and its selection ("workflow"). Only
    a record of llm-agent-workflow/v1 may lack the selection (made before
    the selector, with the preview). An unknown identity, or a selection
    that contradicts it, is refused: one workflow is never taken for
    another."""
    error = error or SettingsError
    ident = (r.get("workflowVersion"), r.get("workflowSha256"))
    named = [n for n, w in WORKFLOWS.items() if (w["version"], w["sha256"]) == ident]
    where = f"decision point {r.get('point')}"
    if not named:
        raise error(f"{where}: unknown workflow identity {ident[0]!r} ({str(ident[1])[:12]}…)")
    if "workflow" not in r:
        if named[0] != LEGACY_WORKFLOW:
            raise error(f"{where}: a {ident[0]} record without its workflow selection")
        return LEGACY_WORKFLOW
    if r["workflow"] != named[0]:
        raise error(f"{where}: the workflow selection {r['workflow']!r} contradicts the identity {ident[0]}")
    return named[0]


def file_sha256(path):
    with open(path, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


# --- validation feedback -------------------------------------------------

def refused_items(validation):
    """The refused results of an attempt's validate answers, in batch order:
    [{"action": number in the reply, "result": the simulator's result}]."""
    out = []
    for v in validation or []:
        idx = v["indices"]
        for k, r in enumerate((v.get("answer") or {}).get("results") or []):
            if not r.get("ok"):
                out.append({"action": idx[k] if k < len(idx) else k, "result": r})
    return out


def feedback_text(items):
    """The message that answers a refused proposal (validation feedback
    v1): the refused results exactly as the simulator returned them."""
    return FEEDBACK_TEMPLATE.format(results="\n".join(compact(x) for x in items))


# --- selective preview ---------------------------------------------------

def eligibility(answer, obs):
    """Whether a valid batch's preview is shown (llm-agent-preview-
    eligibility/v1): the reasons the rule finds in the engine's answer, and
    facts recorded with them (never used to decide)."""
    p = (answer or {}).get("preview") or {}
    fund = p.get("funding") or {}
    reasons = []
    if (fund.get("commitmentCents") or 0) > 0:
        reasons.append("commitment")
    if any("unchanged" not in (g.get("when") or "") for g in p.get("guaranteed") or []):
        reasons.append("obligation")
    if p.get("terminations"):
        reasons.append("termination")
    if p.get("displacements"):
        reasons.append("displacement")
    if p.get("slotsOpened") or p.get("slotsClosed") or p.get("hires"):
        reasons.append("capacity")
    sol = ((((obs or {}).get("finance") or {}).get("financing") or {}).get("solvency")) or {}
    facts = {"commitmentCents": fund.get("commitmentCents"), "headroomCents": fund.get("headroomCents"),
             "creditAvailableCents": sol.get("creditAvailableCents"),
             "creditSupported": None, "displacedBookings": len(p.get("displacements") or []),
             "terminations": len(p.get("terminations") or [])}
    if fund.get("commitmentCents") is not None and fund.get("headroomCents") is not None \
            and sol.get("creditAvailableCents") is not None:
        facts["creditSupported"] = fund["commitmentCents"] > fund["headroomCents"] - sol["creditAvailableCents"]
    return {"version": ELIGIBILITY_VERSION, "eligible": bool(reasons), "reasons": reasons, "facts": facts}


def preview_text(answer, indices, has_ops):
    return PREVIEW_TEMPLATE.format(indices=", ".join(str(i) for i in indices), preview=compact(answer),
                                   ops_note=OPS_NOTE if has_ops else "")


def request_preview(h, actions, clock, pset, obs):
    """The preview of a valid proposal's staffing actions and, when it is
    shown, its message. status: shown, not_eligible, too_large or failed."""
    staff, idx = A4.staffing_part(actions)
    pv = {"actions": staff, "indices": idx, "status": None, "answer": None, "answerSha256": None, "error": None,
          "eligibility": None, "digestsUnchanged": None, "message": None}
    text = None
    try:
        ans = h.request("preview", {"actions": staff})
    except ProtocolError as e:
        pv["status"] = "failed"
        pv["error"] = {"code": e.code, "message": A3.clip(e.message or "", A3.MAX_MESSAGE)}
    else:
        pv["answer"], pv["answerSha256"] = ans, sha256(compact(ans))
        pv["eligibility"] = eligibility(ans, obs)
        if not pv["eligibility"]["eligible"]:
            pv["status"] = "not_eligible"
        else:
            text = preview_text(ans, idx, len(staff) < len(actions))
            pv["message"] = {"sha256": sha256(text), "chars": len(text)}
            if len(text) > A4.preview_limit(pset):
                pv["status"], text = "too_large", None
            else:
                pv["status"] = "shown"
    pv["digestsUnchanged"] = h.request("digests").get("digests") == clock["digests"]
    return pv, text


def preview_due(at, done):
    """The first fully valid parsed proposal at the point with a staffing
    action is previewed."""
    return (not done and (at.get("parse") or {}).get("ok") and at.get("detailsRequested") is None
            and isinstance(at.get("actions"), list) and any(a.get("type") in STAFFING for a in at["actions"])
            and at.get("validation") is not None and not at.get("refusals") and at.get("protocolError") is None)


# --- execution of the previous decision ----------------------------------

def execution_of(obs, prev):
    """How the previous decision point's submitted batches were executed,
    read from this point's observation: the engine's action log entries at
    the previous decision's time. The harness applies the batches there, in
    order. An applied batch is logged with its results; a refused batch
    changes nothing, not even the log, so it shows here as not applied,
    and its refusal is in the harness's own record (decisions.json). Logged
    batches are matched to submitted ones in order, by their action types."""
    if prev is None:
        return None
    recent = ((obs or {}).get("actions") or {}).get("recent") or []
    groups = {}
    for r in reversed(recent):  # the log is newest first
        if r.get("time") == prev.get("time"):
            groups.setdefault(r.get("batch"), []).append(r)
    logged = [groups[k] for k in sorted(groups, key=lambda x: (x is None, x))]
    out, j = [], 0
    for i, b in enumerate(prev.get("batches") or []):
        types = [a.get("type") for a in b]
        g = logged[j] if j < len(logged) else None
        if g is not None and [(x.get("action") or {}).get("type") for x in g] == types:
            j += 1
            out.append({"batch": i, "actions": len(b), "applied": True, "loggedBatch": g[0].get("batch"),
                        "results": [{k: x.get(k) for k in ("ok", "code", "message", "effectiveAt", "contractId",
                                                           "employeeId")} for x in g]})
        else:
            out.append({"batch": i, "actions": len(b), "applied": False, "loggedBatch": None, "results": None})
    return {"point": prev["point"], "source": "the engine's action log in this decision point's observation",
            "refusals": "a refused batch leaves no log entry; its results are in the harness's decisions.json",
            "unmatchedLogEntries": sum(len(g) for g in logged[j:]), "batches": out}


# --- the record ----------------------------------------------------------

def stages(attempts, batches, outcome):
    """A: proposed; B: validated (every batch valid); C: previewed;
    D: submitted. E (executed) is read at the next decision point."""
    prop = [a["id"] for a in attempts if (a.get("parse") or {}).get("ok") and a.get("detailsRequested") is None]
    val = [a["id"] for a in attempts if a.get("validation") is not None and not a.get("refusals")
           and a.get("protocolError") is None]
    ref = [a["id"] for a in attempts if a.get("refusals")]
    pv = next((a for a in attempts if a.get("preview")), None)
    return {"proposed": prop, "validated": val, "refused": ref,
            "previewed": {"attempt": pv["id"], "status": pv["preview"]["status"]} if pv else None,
            "submitted": {"batches": len(batches), "actions": sum(len(b) for b in batches), "outcome": outcome}}


def completeness(attempts, stopped, cut, recorded=True):
    c = A3.completeness(attempts, stopped, cut, recorded)
    c["validationsRecorded"] = sum(len(a.get("validation") or []) for a in attempts)
    c["previewsRequested"] = sum(1 for a in attempts if a.get("preview"))
    c["previewsRecorded"] = sum(1 for a in attempts if a.get("preview") and
                                (a["preview"]["answer"] is not None or a["preview"]["error"] is not None))
    return c


def new_record(point, clock, mode, mem, prompt, left, pset, workflow):
    rec = A3.new_record(point, clock, mode, mem, prompt, left, pset)
    rec.update({"schema": RECORD_SCHEMA, "agentVersion": VERSION, "baseAgentVersion": BASE_VERSION,
                "workflowVersion": WORKFLOWS[workflow]["version"], "workflowSha256": WORKFLOWS[workflow]["sha256"]})
    return rec


# --- one decision point --------------------------------------------------

def validate_all(h, at, actions, k):
    """Every batch of the proposal validated (version 3's batches), each
    answer recorded. Returns the refused results (version 3's form)."""
    at["validation"], refused = [], []
    for batch_no, (batch, idx) in enumerate(A3.split(actions)):
        try:
            v = h.request("validate", {"actions": batch})
        except ProtocolError as e:
            at["protocolError"] = {"batch": batch_no, "code": e.code, "message": A3.clip(e.message or "", A3.MAX_MESSAGE)}
            raise
        at["validation"].append({"actions": batch, "indices": idx, "answer": v})
        if not v.get("valid"):
            refused += A3.refusals(v, idx, k)
    return refused


def ask(h, ev, client, settings, memory, point, prev=None, *, workflow=LEGACY_WORKFLOW):
    """One live decision point, with the named workflow (WORKFLOWS). Every
    attempt is recorded as it is made (attempts.jsonl) and in the record.
    A caller that names no workflow gets V0.11a's (with the preview), as
    before the selector; the agent's command line defaults to
    validation-first (DEFAULT_WORKFLOW) and always names it."""
    if workflow not in WORKFLOWS:
        raise SettingsError(f"unknown workflow {workflow!r}")
    preview_on = WORKFLOWS[workflow]["preview"]
    start = time.monotonic()
    clock = h.request("digests")
    pset = A3.prompt_settings(settings)
    obs, omitted, catalog, mem, prompt, left = A3.prepare(h, memory, point, pset)
    rec = new_record(point, clock, "live", mem, prompt, left, pset, workflow)
    attempts, cut, stopped, used = [], [], None, None
    rec.update({"attempts": attempts, "final": None, "batches": [], "outcome": "fallback", "reason": None,
                "refusals": [], "errors": [], "previousExecution": execution_of(obs, prev)})
    ev.append("memory.jsonl", {"point": point, "sha256": rec["memorySha256"], "chars": len(mem), "text": mem})
    if prompt is None:
        rec["errors"].append(f"prompt_too_large: over {settings['maxPromptChars']} characters")
        stopped = "prompt_too_large"
    else:
        types = {t["type"] for t in catalog.get("types", [])}
        messages = [{"role": "system", "content": settings["systemPrompt"]}, {"role": "user", "content": prompt}]
        failures, details_open, turn = 0, settings["allowDetails"], "first"
        inp = {"kind": "prompt", "sha256": sha256(prompt), "chars": len(prompt)}
        previewed = False
        while failures <= settings["maxRetries"]:
            if attempts and time.monotonic() - start >= settings["decisionBudgetSeconds"]:
                rec["errors"].append(f"budget: {settings['decisionBudgetSeconds']} s spent; no further attempt")
                stopped = "budget"
                break
            k = len(attempts) + 1
            at = A3.new_attempt(point, k, turn, inp)
            at["preview"] = None
            payload = client.payload(messages)
            t0 = time.monotonic()
            status, body, err = client.call(payload)
            call = {"point": point, "attempt": k, "turn": turn, "model": settings["model"],
                    "endpoint": settings["endpoint"], "request": payload, "status": status,
                    "latencyMs": round((time.monotonic() - t0) * 1000), "response": body[:A3.MAX_RESPONSE_KEPT],
                    "responseTruncated": len(body) > A3.MAX_RESPONSE_KEPT, "usage": None, "servedModel": None,
                    "content": None, "error": err}
            at["transport"] = "ok" if err is None else err.split(":", 1)[0]
            asked, review, refused = None, None, []
            if err is None:
                try:
                    content, call["usage"], call["servedModel"] = A3.completion(body)
                    call["content"] = content
                    at["output"] = {"sha256": sha256(content), "chars": len(content),
                                    "text": A3.bounded(content, A3.MAX_OUTPUT_TEXT, f"attempt {k} output", cut)}
                    try:
                        d = A3.parse_decision(content, types, details_open)
                    except ValueError as e:
                        at["parse"] = {"ok": False, "code": str(e).split(":", 1)[0],
                                       "message": A3.clip(str(e), A3.MAX_MESSAGE)}
                        raise
                    at["parse"] = {"ok": True}
                    at["reason"], at["actions"] = d["reason"], d["actions"]
                    rec["reason"] = d["reason"]
                    if "details" in d:
                        asked = at["detailsRequested"] = d["details"]
                    else:
                        if d["actions"]:
                            refused = validate_all(h, at, d["actions"], k)
                        else:
                            at["validation"] = []  # nothing to validate
                        if refused:
                            at["refusals"] = refused
                            rec["refusals"] += refused
                            raise ValueError("refused: " + "; ".join(
                                f"action {r['action']}: {r['code']}: {r['message']}" for r in refused))
                        if preview_on and preview_due(at, previewed):
                            previewed = True
                            at["preview"], review = request_preview(h, d["actions"], clock, pset, obs)
                except ProtocolError as e:
                    call["error"] = f"invalid_action: validate answered {e.code}: {e.message}"
                except ValueError as e:
                    call["error"] = str(e)
                    if at["output"] is None:
                        at["transport"] = str(e).split(":", 1)[0]
            ev.call(call)
            if call["error"] is not None:
                at["error"] = A3.bounded(call["error"], A3.MAX_FEEDBACK, f"attempt {k} error", cut)
            elif asked is None and review is None:
                at["used"] = True
            attempts.append(at)
            ev.append("attempts.jsonl", at)
            if asked is not None:
                details_open = False
                text = A3.details_text(h, asked, obs, omitted, catalog, settings["maxPromptChars"] // 4)
                messages = messages + [{"role": "assistant", "content": call["content"]},
                                       {"role": "user", "content": text}]
                turn, inp = "details", {"kind": "details", "sha256": sha256(text), "chars": len(text)}
                continue
            if review is not None:
                details_open = False
                messages = messages + [{"role": "assistant", "content": call["content"]},
                                       {"role": "user", "content": review}]
                turn, inp = "review", {"kind": "preview", "sha256": sha256(review), "chars": len(review)}
                continue
            if at["used"]:
                used = at
                break
            failures += 1
            turn = "retry"
            rec["errors"].append(f"attempt {k}: {call['error']}")
            if at["refusals"]:  # refused by validation: the simulator's results, exactly
                text = feedback_text(refused_items(at["validation"]))
                details_open = False
                messages = messages + [{"role": "assistant", "content": call["content"]},
                                       {"role": "user", "content": text}]
                inp = {"kind": "validation", "sha256": sha256(text), "chars": len(text)}
            elif call["content"] is not None:  # unreadable or a protocol error: version 3's retry message
                text = A3.retry_text(call["error"])
                messages = messages + [{"role": "assistant", "content": call["content"]},
                                       {"role": "user", "content": text}]
                inp = {"kind": "retry", "sha256": sha256(text), "chars": len(text), "feedback": at["error"],
                       "feedbackTruncated": len(call["error"]) > A3.MAX_FEEDBACK}
            else:  # no answer: the same messages again
                inp = {"kind": "resend", "sha256": inp["sha256"], "chars": inp["chars"]}
    if used is not None:
        rec["batches"] = [b for b, _ in A3.split(used["actions"])] if used["actions"] else []
        rec["outcome"] = "actions" if rec["batches"] else "no_action"
    else:
        rec["batches"], rec["outcome"] = [], "fallback"  # never a batch that was not validated
    # Validation and preview change nothing: the state must still be the decision point's.
    after = h.request("digests").get("digests")
    rec["digestsBeforeSubmit"] = {"unchanged": after == clock["digests"]}
    if after != clock["digests"] and rec["batches"]:
        rec["errors"].append("state_changed: the digests changed during the decision; nothing is submitted")
        rec["batches"], rec["outcome"] = [], "fallback"
    rec["final"] = {"attempt": used["attempt"] if used else None, "outcome": rec["outcome"]}
    rec["stages"] = stages(attempts, rec["batches"], rec["outcome"])
    rec["completeness"] = completeness(attempts, stopped, cut)
    return rec["batches"], rec


# --- replay --------------------------------------------------------------

def load_recordings(paths):
    """Each recorded decisions file: (path, SHA-256, records). Only
    dreamteam.llm-decision-record/v5 records are replayed here; version 4
    recordings replay with llm_agent4.py, version 3 with llm_agent3.py,
    versions 1 and 2 with llm_agent.py. Every record's workflow is checked
    (workflow_of), and all must have the same one."""
    out, workflows = [], set()
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
                    f"of {AGENT} {VERSION} ({MEMORY_FORMAT}, {PROMPT_VERSION}); version 4 "
                    "recordings replay with agents/llm/llm_agent4.py, version 3 with llm_agent3.py, versions 1 and "
                    "2 with llm_agent.py")
            try:
                workflows.add(workflow_of(r))
            except SettingsError as e:
                raise SettingsError(f"replay {p}: {e}")
        if len(workflows) > 1:
            raise SettingsError(f"replay {p}: the recordings mix the workflows {sorted(workflows)}")
        out.append((p, hashlib.sha256(raw).hexdigest(), recs))
    return out


def recorded_workflow(recordings):
    """The one workflow of loaded recordings (load_recordings checked it)."""
    return workflow_of(recordings[0][2][0])


def verify_attempts(h, r, obs, omitted, catalog, prompt, clock):
    """Every recorded attempt checked against this run: each input message
    rebuilt (prompt, details, validation feedback, preview, retry, resend),
    each validate and preview asked again and compared, the preview rule and
    its eligibility recomputed, the used attempt, batches and stages."""
    point, atts = r["point"], r.get("attempts")
    if not isinstance(atts, list) or not isinstance(r.get("final"), dict):
        raise ReplayError(f"decision point {point}: the record has no attempts or final decision")
    pset = r["promptSettings"]
    preview_on = WORKFLOWS[workflow_of(r, ReplayError)]["preview"]
    checked = {"inputs": 0, "validations": 0, "previews": 0}
    prev, previewed = None, False
    for k, at in enumerate(atts, 1):
        where = f"decision point {point}, attempt {k}"
        if not isinstance(at, dict) or at.get("attempt") != k or at.get("point") != point:
            raise ReplayError(f"{where}: missing or out of order")
        inp = at.get("input") or {}
        kind, text = inp.get("kind"), None
        if k == 1 and kind == "prompt":
            text = prompt
        elif k > 1 and kind == "details" and prev.get("detailsRequested"):
            text = A3.details_text(h, prev["detailsRequested"], obs, omitted, catalog, pset["maxPromptChars"] // 4)
        elif k > 1 and kind == "preview" and (prev.get("preview") or {}).get("status") == "shown" \
                and prev.get("error") is None:
            p = prev["preview"]
            text = preview_text(p["answer"], p["indices"], len(p["actions"]) < len(prev["actions"]))
        elif k > 1 and kind == "validation" and prev.get("refusals"):
            text = feedback_text(refused_items(prev.get("validation")))
        elif k > 1 and kind == "retry" and prev.get("error") is not None and not prev.get("refusals"):
            if inp.get("feedback") != prev["error"]:
                raise ReplayError(f"{where}: the feedback is not the previous attempt's error")
            text = None if inp.get("feedbackTruncated") else A3.retry_text(inp["feedback"])
        elif k > 1 and kind == "resend" and prev.get("output") is None:
            if inp.get("sha256") != (prev.get("input") or {}).get("sha256"):
                raise ReplayError(f"{where}: a resend that is not the previous message")
        else:
            raise ReplayError(f"{where}: an input of kind {kind!r} cannot follow the previous attempt")
        if text is not None:
            if sha256(text) != inp.get("sha256"):
                raise ReplayError(f"{where}: the {kind} message rebuilt from this run differs from the recording's")
            checked["inputs"] += 1
        acts = at.get("actions")
        parts = A3.split(acts) if isinstance(acts, list) and acts and at.get("detailsRequested") is None else []
        vals = at.get("validation") or []
        pe = at.get("protocolError")
        if len(vals) + (1 if pe else 0) > len(parts):
            raise ReplayError(f"{where}: more validated batches than the actions make")
        if (at.get("parse") or {}).get("ok") and at.get("detailsRequested") is None and acts and \
                len(vals) + (1 if pe else 0) < len(parts) and not pe and not at.get("refusals"):
            raise ReplayError(f"{where}: a proposal that was not validated in full")
        for i, (batch, idx) in enumerate(parts[:len(vals) + (1 if pe else 0)]):
            try:
                ans = h.request("validate", {"actions": batch})
            except ProtocolError as e:
                if not pe or pe.get("batch") != i or pe.get("code") != e.code:
                    raise ReplayError(f"{where}: validate answers {e.code}, unlike the recording")
                checked["validations"] += 1
                continue
            if i >= len(vals) or vals[i].get("actions") != batch or vals[i].get("indices") != idx \
                    or vals[i].get("answer") != ans:
                raise ReplayError(f"{where}: validate answers batch {i} differently from the recording")
            checked["validations"] += 1
        refused = []
        for v in vals:
            if not (v.get("answer") or {}).get("valid"):
                refused += A3.refusals(v["answer"], v["indices"], k)
        if refused != (at.get("refusals") or []):
            raise ReplayError(f"{where}: the refusals are not the validate answers'")
        pv = at.get("preview")
        due = preview_on and preview_due(at, previewed)
        if bool(pv) != bool(due):
            raise ReplayError(f"{where}: a preview {'missing' if due else 'not due'} at this attempt")
        if pv:
            previewed = True
            staff, idx = A4.staffing_part(at["actions"])
            if pv.get("actions") != staff or pv.get("indices") != idx:
                raise ReplayError(f"{where}: the previewed actions are not the proposal's staffing actions")
            again, again_text = request_preview(h, at["actions"], clock, pset, obs)
            for f in ("status", "answer", "answerSha256", "error", "eligibility", "digestsUnchanged", "message"):
                if again[f] != pv.get(f):
                    raise ReplayError(f"{where}: the preview's {f} differs from the recording's")
            checked["previews"] += 1
            if again_text is not None and (at.get("used") or at.get("error") is not None):
                raise ReplayError(f"{where}: a proposal whose preview was shown was used")
        prev = at
    if prev is not None and (prev.get("preview") or {}).get("status") == "shown" and prev.get("error") is None \
            and (r.get("completeness") or {}).get("stopped") != "budget":
        raise ReplayError(f"decision point {point}: the preview was never answered")
    used = [a for a in atts if a.get("used")]
    fin = r["final"]
    if len(used) > 1 or (used and used[-1] is not atts[-1]) or fin.get("attempt") != (used[0]["attempt"] if used else None):
        raise ReplayError(f"decision point {point}: the final decision is not the last attempt's")
    if used and (used[0].get("refusals") or used[0].get("validation") is None):
        raise ReplayError(f"decision point {point}: the used attempt was not validated in full")
    want = [b for b, _ in A3.split(used[0]["actions"])] if used and used[0]["actions"] else []
    after = h.request("digests").get("digests")
    if (r.get("digestsBeforeSubmit") or {}).get("unchanged") != (after == clock["digests"]):
        raise ReplayError(f"decision point {point}: the state check before submission differs")
    if after != clock["digests"]:
        want = []
    if r["batches"] != want or fin.get("outcome") != r.get("outcome"):
        raise ReplayError(f"decision point {point}: the submitted batches are not the used attempt's actions")
    if r.get("stages") != stages(atts, r["batches"], r["outcome"]):
        raise ReplayError(f"decision point {point}: the stages are not the attempts'")
    return checked


def replay_point(h, ev, replayer, point, prev=None):
    """One replayed decision point: the inputs, memory and prompt as version
    3 checks them, the attempts (verify_attempts) and the previous point's
    execution, read again."""
    clock = h.request("digests")
    r = replayer.next(point, clock)
    pset = r.get("promptSettings")
    if not isinstance(pset, dict) or set(pset) != set(A3.PROMPT_SETTINGS):
        raise ReplayError(f"decision point {point}: the recording has no usable promptSettings")
    if replayer.memory is None:
        replayer.memory = A3.Memory(pset["memoryDecisions"], pset["memoryMetricPoints"], pset["memoryMaxChars"])
    memory = replayer.memory
    obs, omitted, catalog, mem, prompt, left = A3.prepare(h, memory, point, pset)
    ev.append("memory.jsonl", {"point": point, "sha256": sha256(mem), "chars": len(mem), "text": mem})
    if sha256(mem) != r.get("memorySha256"):
        raise ReplayError(f"decision point {point}: the memory rebuilt from this run differs from the recording's "
                          f"({MEMORY_FORMAT})")
    if (sha256(prompt) if prompt else None) != r.get("promptSha256"):
        raise ReplayError(f"decision point {point}: the prompt rebuilt from this run differs from the recording's "
                          f"({PROMPT_VERSION})")
    if r.get("previousExecution") != execution_of(obs, prev):
        raise ReplayError(f"decision point {point}: the previous point's execution differs from the recording's")
    checked = verify_attempts(h, r, obs, omitted, catalog, prompt, clock)
    rec = dict(r)
    rec.update({"mode": "replay", "replayed": os.path.basename(replayer.chosen[0]), "replayChecked": checked})
    memory.record(r)
    return rec


def check_contract(hello):
    A4.check_contract(hello)


def summary(ev, records, settings):
    """Version 3's usage summary, with the workflow's counts."""
    s = A3.summary(ev, records, settings)
    s["retryRequests"] = sum(1 for c in ev.calls if c.get("turn") == "retry")
    s["reviewRequests"] = sum(1 for c in ev.calls if c.get("turn") == "review")
    st = {}
    for r in records:
        p = ((r.get("stages") or {}).get("previewed") or {}).get("status")
        if p is not None:
            st[p] = st.get(p, 0) + 1
    s["previewSteps"] = dict(sorted(st.items()))
    return s


def usage_text(s):
    t = A3.usage_text(s).replace(f"({AGENT} {BASE_VERSION})", f"({AGENT} {VERSION})")
    return t + (f"workflow              {s['reviewRequests']} preview reviews; previews by status "
                f"{', '.join(f'{k} {v}' for k, v in s['previewSteps'].items()) or 'none'}\n")


# --- main ----------------------------------------------------------------

def run(h, ev, client, settings, memory, replayer, records, workflow):
    """The policy loop: one decision per turn, every record kept. A live
    record names the workflow it was made with; a replayed one is the
    recording's."""
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
        else:
            try:
                batches, rec = ask(h, ev, client, settings, memory, point, prev, workflow=workflow)
            except ProtocolError as e:
                # Never expected (the agent uses permitted operations only);
                # recorded as such, and refused by replay.
                batches, rec = [], {"schema": RECORD_SCHEMA, "point": point, "time": None, "mode": "live",
                                    "agentVersion": VERSION, "baseAgentVersion": BASE_VERSION,
                                    "promptVersion": PROMPT_VERSION, "memoryFormat": MEMORY_FORMAT,
                                    "workflowVersion": WORKFLOWS[workflow]["version"],
                                    "workflowSha256": WORKFLOWS[workflow]["sha256"],
                                    "protocolError": str(e), "outcome": "fallback", "batches": [],
                                    "errors": [f"protocol: {e}"], "attempts": [],
                                    "final": {"attempt": None, "outcome": "fallback"},
                                    "stages": stages([], [], "fallback"), "previousExecution": None,
                                    "completeness": completeness([], "protocol_error", [], recorded=False),
                                    "digests": None}
            rec["workflow"] = workflow
        records.append(rec)
        if memory and rec.get("time") is not None:
            memory.record(rec)
        ev.append("decisions.jsonl", rec)
        prev = rec if rec.get("time") is not None else None
        yield
        h.decide(batches)


def main(argv=None):
    ap = argparse.ArgumentParser(description="The DreamTeam LLM agent, validation-first (a benchmark policy).")
    ap.add_argument("--settings", help="settings file (dreamteam.llm-agent-settings/v1); required live")
    ap.add_argument("--replay", action="append", default=[], metavar="FILE",
                    help="submit the decisions recorded in FILE (decisions.jsonl) instead of asking a model; repeatable")
    ap.add_argument("--evidence", default="evidence", help="evidence directory (default ./evidence)")
    ap.add_argument("--workflow", choices=sorted(WORKFLOWS),
                    help=f"live: the decision workflow (default {DEFAULT_WORKFLOW}; validation-first-preview adds the "
                         "selective preview). Replay takes it from the recording; if given, it must match")
    a = ap.parse_args(argv)
    try:
        if a.replay:
            settings, recordings = None, load_recordings(a.replay)
            workflow = recorded_workflow(recordings)
            if a.workflow is not None and a.workflow != workflow:
                raise SettingsError(f"--workflow {a.workflow}: the recordings were made with {workflow}")
        elif a.settings:
            settings, recordings = A3.load_settings(a.settings, os.environ), None
            workflow = a.workflow or DEFAULT_WORKFLOW
        else:
            raise SettingsError("give --settings (live) or --replay (replay)")
    except SettingsError as e:
        print(f"{AGENT}: {e}", file=sys.stderr)
        return 2
    ev = A3.Evidence(a.evidence)
    info = {"schema": EVIDENCE_SCHEMA, "agent": AGENT, "version": VERSION, "baseVersion": BASE_VERSION,
            "promptVersion": PROMPT_VERSION, "memoryFormat": MEMORY_FORMAT, "workflow": workflow,
            "workflowVersion": WORKFLOWS[workflow]["version"], "workflowSha256": WORKFLOWS[workflow]["sha256"],
            "preview": WORKFLOWS[workflow]["preview"], "feedbackVersion": FEEDBACK_VERSION, "previewVersion": PREVIEW_VERSION,
            "eligibilityVersion": ELIGIBILITY_VERSION, "agentSha256": file_sha256(os.path.abspath(__file__)),
            "baseAgentSha256": file_sha256(BASE_PATH), "previewAgentSha256": file_sha256(PREVIEW_PATH),
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
        for _ in run(h, ev, client, settings, memory, replayer, records, workflow):
            finish()
        return 0
    except (ExchangeError, ReplayError) as e:
        print(f"{AGENT}: {e}", file=sys.stderr)
        return 1
    finally:
        finish()


if __name__ == "__main__":
    sys.exit(main())
