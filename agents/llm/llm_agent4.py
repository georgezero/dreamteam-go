#!/usr/bin/env python3
"""The DreamTeam LLM agent, preview-enabled (llm-agent, version 4).

An external benchmark policy (docs/BENCHMARK.md §4). Specification:
docs/LLM_AGENT.md §14.

Version 4 is version 3 (agents/llm/llm_agent3.py, loaded unchanged from
this directory) with one addition, the treatment `llm-agent-preview/v1`:
before a proposal with staffing actions is validated, the agent asks the
simulator for its existing preview of those actions (the protocol's
`preview`, the game's staff-screen preview), shows the model the answer
exactly as the simulator returned it, and lets the model keep, revise or
abandon the proposal. Its reply replaces the proposal and is then
validated and submitted exactly as version 3 does.

Everything else is version 3's: the system prompt, the prompt
(llm-agent-prompt/v2), the memory (llm-agent-memory/v1), the settings,
retries, details turn, batching, validation and fallback. The model sees
the same first message at every decision point. The preview step:
* happens at most once per decision point, for the first parsed proposal
  that has a staffing action (operations actions have no preview);
* is skipped, with the reason recorded, when the preview fails (a protocol
  error) or its message would be longer than maxPromptChars (the prompt's
  own bound; a termination's preview lists every booking it displaces):
  the proposal then goes to validation unchanged, as version 3's would, and
  the failure is counted, never shown as a preview;
* adds one model request when it happens; the details turn is not offered
  after it.

The record is dreamteam.llm-decision-record/v4: version 3's, with each
attempt's preview request, answer (and its SHA-256) or failure, the review
message by SHA-256 (input kind "preview"), and at each decision point the
preview step's status and what the model did after it. Replay mode
(--replay FILE ...) checks all of it again without a model: the preview is
requested again and compared, and the review message rebuilt.
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


def _base():
    """llm-agent 3, the unchanged base: the copy already imported, or the
    file next to this one (python3 -I leaves this directory off sys.path)."""
    m = sys.modules.get("llm_agent3")
    if m is None:
        spec = importlib.util.spec_from_file_location("llm_agent3", BASE_PATH)
        m = importlib.util.module_from_spec(spec)
        sys.modules["llm_agent3"] = m
        spec.loader.exec_module(m)
    return m


A3 = _base()

AGENT = A3.AGENT
VERSION = "4"
BASE_VERSION = A3.VERSION
PROMPT_VERSION = A3.PROMPT_VERSION
MEMORY_FORMAT = A3.MEMORY_FORMAT
EVIDENCE_SCHEMA = "dreamteam.llm-agent-evidence/v4"
RECORD_SCHEMA = "dreamteam.llm-decision-record/v4"
TREATMENT_VERSION = "llm-agent-preview/v1"
NEEDS = A3.NEEDS + ("preview",)
STAFFING = A3.STAFFING
ProtocolError, ExchangeError, ReplayError, SettingsError = (A3.ProtocolError, A3.ExchangeError, A3.ReplayError,
                                                            A3.SettingsError)
sha256, compact = A3.sha256, A3.compact

# The treatment's only text: the message that shows the preview. {indices}
# are the staffing actions' numbers in the model's reply, {preview} the
# simulator's answer as compact JSON; OPS_NOTE is added when the reply also
# had operations actions.
REVIEW_TEMPLATE = (
    "PREVIEW. Nothing has been submitted yet. This is the simulator's preview of the staffing actions in your "
    "reply (actions {indices}), exactly as the simulator returned it:\n{preview}\n\n{ops_note}"
    "Now reply with your final decision, one JSON object {{\"reason\": ..., \"actions\": [...]}}. It replaces your "
    "reply above: give the same actions to keep it, changed actions to revise it, or \"actions\": [] to submit "
    "nothing. It is then validated and submitted as before.")
OPS_NOTE = "The operations actions in your reply have no preview (the simulator previews staffing actions only).\n\n"
TREATMENT_SHA256 = sha256(REVIEW_TEMPLATE + "\n" + OPS_NOTE)


def file_sha256(path):
    with open(path, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def review_text(answer, indices, has_ops):
    """The message that shows the preview (llm-agent-preview/v1)."""
    return REVIEW_TEMPLATE.format(indices=", ".join(str(i) for i in indices), preview=compact(answer),
                                  ops_note=OPS_NOTE if has_ops else "")


def preview_limit(pset):
    """The longest review message shown: the prompt's own bound."""
    return pset["maxPromptChars"]


def staffing_part(actions):
    """The staffing actions of a reply and their indices in it."""
    idx = [i for i, a in enumerate(actions) if a["type"] in STAFFING]
    return [actions[i] for i in idx], idx


def preview_due(at, done):
    """Whether this attempt's proposal is previewed: the first parsed
    proposal at the decision point with a staffing action, not a details
    request."""
    return (not done and (at.get("parse") or {}).get("ok") and at.get("detailsRequested") is None
            and isinstance(at.get("actions"), list) and any(a.get("type") in STAFFING for a in at["actions"]))


def request_preview(h, actions, clock, pset):
    """Ask for the preview of the staffing actions: the attempt's preview
    record and, when it can be shown, the review message."""
    staff, idx = staffing_part(actions)
    pv = {"actions": staff, "indices": idx, "status": None, "answer": None, "answerSha256": None,
          "error": None, "digestsUnchanged": None, "message": None}
    text = None
    try:
        ans = h.request("preview", {"actions": staff})
    except ProtocolError as e:
        pv["status"] = "failed"
        pv["error"] = {"code": e.code, "message": A3.clip(e.message or "", A3.MAX_MESSAGE)}
    else:
        pv["answer"], pv["answerSha256"] = ans, sha256(compact(ans))
        text = review_text(ans, idx, len(staff) < len(actions))
        pv["message"] = {"sha256": sha256(text), "chars": len(text)}
        if len(text) > preview_limit(pset):
            pv["status"], text = "too_large", None
        else:
            pv["status"] = "ok"
    # A preview changes nothing: the session's digests are the decision point's.
    pv["digestsUnchanged"] = h.request("digests").get("digests") == clock["digests"]
    return pv, text


def same_actions(a, b):
    """The same actions, in any order."""
    return sorted(compact(x) for x in a) == sorted(compact(x) for x in b)


def response_of(proposal, actions):
    if actions is None:
        return None
    if same_actions(proposal, actions):
        return "kept"
    return "abandoned" if not actions else "revised"


def part_of(proposal, actions, staffing):
    """What happened to one kind of action (staffing or operations)."""
    p = [a for a in proposal if (a["type"] in STAFFING) == staffing]
    q = [a for a in actions if (a["type"] in STAFFING) == staffing]
    if not p:
        return "none proposed" if not q else "added"
    if same_actions(p, q):
        return "kept"
    return "dropped" if not q else "revised"


def review_summary(attempts):
    """The decision point's preview step: its status (not_applicable when
    no parsed proposal had a staffing action), the attempts involved, and
    what the model did after the preview: its first parsed reply after it
    (response) and the decision finally submitted (final), each kept,
    revised or abandoned against the previewed proposal."""
    out = {"version": TREATMENT_VERSION, "status": "not_applicable", "proposalAttempt": None,
           "reviewAttempt": None, "response": None, "responseAttempt": None, "staffing": None, "operations": None,
           "final": None}
    prop = next((a for a in attempts if a.get("preview")), None)
    if prop is None:
        return out
    out["status"], out["proposalAttempt"] = prop["preview"]["status"], prop["attempt"]
    if out["status"] != "ok":
        return out
    after = attempts[prop["attempt"]:]
    out["reviewAttempt"] = after[0]["attempt"] if after else None
    first = next((a for a in after if (a.get("parse") or {}).get("ok")), None)
    if first is None:
        out["response"] = "no_reply"
    else:
        out["response"], out["responseAttempt"] = response_of(prop["actions"], first["actions"]), first["attempt"]
        out["staffing"] = part_of(prop["actions"], first["actions"], True)
        out["operations"] = part_of(prop["actions"], first["actions"], False)
    used = next((a for a in attempts if a.get("used")), None)
    out["final"] = "fallback" if used is None else response_of(prop["actions"], used["actions"] or [])
    return out


def completeness(attempts, stopped, cut, recorded=True):
    c = A3.completeness(attempts, stopped, cut, recorded)
    pv = [a["preview"] for a in attempts if a.get("preview")]
    c["previewsRequested"] = len(pv)
    c["previewsRecorded"] = sum(1 for p in pv if p["answer"] is not None or p["error"] is not None)
    return c


def new_record(point, clock, mode, mem, prompt, left, pset):
    rec = A3.new_record(point, clock, mode, mem, prompt, left, pset)
    rec.update({"schema": RECORD_SCHEMA, "agentVersion": VERSION, "baseAgentVersion": BASE_VERSION,
                "treatmentVersion": TREATMENT_VERSION, "treatmentSha256": TREATMENT_SHA256})
    return rec


def ask(h, ev, client, settings, memory, point):
    """One live decision point: version 3's, with the preview step. Every
    attempt is recorded as it is made (attempts.jsonl) and in the record."""
    start = time.monotonic()
    clock = h.request("digests")
    pset = A3.prompt_settings(settings)
    obs, omitted, catalog, mem, prompt, left = A3.prepare(h, memory, point, pset)
    rec = new_record(point, clock, "live", mem, prompt, left, pset)
    attempts, cut, stopped, used = [], [], None, None
    rec.update({"attempts": attempts, "final": None, "batches": [], "outcome": "fallback", "reason": None,
                "refusals": [], "errors": []})
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
            asked, batch_no, review = None, None, None
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
                        if preview_due(at, previewed):
                            previewed = True
                            at["preview"], review = request_preview(h, d["actions"], clock, pset)
                        if review is not None:
                            pass  # the model answers the preview before anything is validated
                        elif d["actions"]:
                            at["validation"], refused = [], []
                            for batch_no, (batch, idx) in enumerate(A3.split(d["actions"])):
                                v = h.request("validate", {"actions": batch})
                                at["validation"].append({"actions": batch, "indices": idx, "answer": v})
                                if not v.get("valid"):
                                    refused += A3.refusals(v, idx, k)
                            if refused:
                                at["refusals"] = refused
                                rec["refusals"] += refused
                                raise ValueError("refused: " + "; ".join(
                                    f"action {r['action']}: {r['code']}: {r['message']}" for r in refused))
                            rec["batches"], rec["outcome"] = [b for b, _ in A3.split(d["actions"])], "actions"
                        else:
                            rec["outcome"] = "no_action"
                except ProtocolError as e:
                    call["error"] = f"invalid_action: validate answered {e.code}: {e.message}"
                    at["protocolError"] = {"batch": batch_no, "code": e.code,
                                           "message": A3.clip(e.message or "", A3.MAX_MESSAGE)}
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
            if call["content"] is not None:  # the model answered: say what was wrong
                text = A3.retry_text(call["error"])
                messages = messages + [{"role": "assistant", "content": call["content"]},
                                       {"role": "user", "content": text}]
                inp = {"kind": "retry", "sha256": sha256(text), "chars": len(text), "feedback": at["error"],
                       "feedbackTruncated": len(call["error"]) > A3.MAX_FEEDBACK}
            else:  # no answer: the same messages again
                inp = {"kind": "resend", "sha256": inp["sha256"], "chars": inp["chars"]}
    if used is None:
        rec["batches"], rec["outcome"] = [], "fallback"  # never a batch that was not validated
    rec["final"] = {"attempt": used["attempt"] if used else None, "outcome": rec["outcome"]}
    rec["preview"] = review_summary(attempts)
    rec["completeness"] = completeness(attempts, stopped, cut)
    return rec["batches"], rec


# --- replay --------------------------------------------------------------

def load_recordings(paths):
    """Each recorded decisions file: (path, SHA-256, records). Only
    dreamteam.llm-decision-record/v4 records are replayed here; version 3
    recordings replay with agents/llm/llm_agent3.py, versions 1 and 2 with
    agents/llm/llm_agent.py."""
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
            got = (r.get("schema"), r.get("agentVersion"), r.get("memoryFormat"), r.get("promptVersion"),
                   r.get("treatmentVersion"), r.get("treatmentSha256"))
            if got != (RECORD_SCHEMA, VERSION, MEMORY_FORMAT, PROMPT_VERSION, TREATMENT_VERSION, TREATMENT_SHA256):
                raise SettingsError(
                    f"replay {p}: version incompatible: decision point {r['point']} is not a {RECORD_SCHEMA} record "
                    f"of {AGENT} {VERSION} ({MEMORY_FORMAT}, {PROMPT_VERSION}, {TREATMENT_VERSION}); version 3 "
                    "recordings replay with agents/llm/llm_agent3.py, versions 1 and 2 with agents/llm/llm_agent.py")
        out.append((p, hashlib.sha256(raw).hexdigest(), recs))
    return out


def verify_attempts(h, r, obs, omitted, catalog, prompt, clock):
    """Every recorded attempt of a decision point, checked against this run
    as version 3 checks them, and the preview step: it was requested
    exactly where the rule puts it, the simulator answers it again the same,
    the review message is rebuilt, and the point's preview summary is the
    attempts'. Returns how many inputs, validate and preview answers were
    checked."""
    point, atts = r["point"], r.get("attempts")
    if not isinstance(atts, list) or not isinstance(r.get("final"), dict):
        raise ReplayError(f"decision point {point}: the record has no attempts or final decision")
    pset = r["promptSettings"]
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
        elif k > 1 and kind == "preview" and (prev.get("preview") or {}).get("status") == "ok" \
                and prev.get("error") is None:
            p = prev["preview"]
            text = review_text(p["answer"], p["indices"], len(p["actions"]) < len(prev["actions"]))
        elif k > 1 and kind == "retry" and prev.get("error") is not None:
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
        pv = at.get("preview")
        due = preview_due(at, previewed)
        if bool(pv) != bool(due):
            raise ReplayError(f"{where}: a preview {'missing' if due else 'not due'} at this attempt")
        shown = False
        if pv:
            previewed = True
            staff, idx = staffing_part(at["actions"])
            if pv.get("actions") != staff or pv.get("indices") != idx:
                raise ReplayError(f"{where}: the previewed actions are not the proposal's staffing actions")
            again, again_text = request_preview(h, at["actions"], clock, pset)
            for f in ("status", "answer", "answerSha256", "error", "digestsUnchanged", "message"):
                if again[f] != pv.get(f):
                    raise ReplayError(f"{where}: the preview's {f} differs from the recording's")
            checked["previews"] += 1
            shown = again_text is not None
            if shown and (at.get("validation") or at.get("error") is not None or at.get("used")):
                raise ReplayError(f"{where}: a previewed proposal was validated or used")
        acts = at.get("actions")
        parts = A3.split(acts) if isinstance(acts, list) and acts and at.get("detailsRequested") is None \
            and not shown else []
        vals = at.get("validation") or []
        pe = at.get("protocolError")
        if len(vals) + (1 if pe else 0) > len(parts):
            raise ReplayError(f"{where}: more validated batches than the actions make")
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
        prev = at
    if prev is not None and (prev.get("preview") or {}).get("status") == "ok" and prev.get("error") is None \
            and (r.get("completeness") or {}).get("stopped") != "budget":
        raise ReplayError(f"decision point {point}: the preview was never answered")
    used = [a for a in atts if a.get("used")]
    fin = r["final"]
    if len(used) > 1 or (used and used[-1] is not atts[-1]) or fin.get("attempt") != (used[0]["attempt"] if used else None):
        raise ReplayError(f"decision point {point}: the final decision is not the last attempt's")
    want = [b for b, _ in A3.split(used[0]["actions"])] if used and used[0]["actions"] else []
    if r["batches"] != want or fin.get("outcome") != r.get("outcome"):
        raise ReplayError(f"decision point {point}: the submitted batches are not the used attempt's actions")
    if r.get("preview") != review_summary(atts):
        raise ReplayError(f"decision point {point}: the preview summary is not the attempts'")
    return checked


def replay_point(h, ev, replayer, point):
    """One replayed decision point: version 3's checks, with the preview
    step's (verify_attempts)."""
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
                          f"({MEMORY_FORMAT}): the memory or its formatting changed without a new format version")
    if (sha256(prompt) if prompt else None) != r.get("promptSha256"):
        raise ReplayError(f"decision point {point}: the prompt rebuilt from this run differs from the recording's "
                          f"({PROMPT_VERSION}): the prompt changed without a new prompt version")
    checked = verify_attempts(h, r, obs, omitted, catalog, prompt, clock)
    rec = dict(r)
    rec.update({"mode": "replay", "replayed": os.path.basename(replayer.chosen[0]), "replayChecked": checked})
    memory.record(r)
    return rec


def check_contract(hello):
    A3.check_contract(hello)
    if "preview" not in ((hello.get("benchmark") or {}).get("permittedOps") or []):
        raise ExchangeError("not the expected benchmark contract (missing ['preview'])")


def summary(ev, records, settings):
    """Version 3's usage summary, with the preview step's counts."""
    s = A3.summary(ev, records, settings)
    s["reviewRequests"] = sum(1 for c in ev.calls if c.get("turn") == "review")
    st = {}
    for r in records:
        p = (r.get("preview") or {}).get("status")
        if p is not None:
            st[p] = st.get(p, 0) + 1
    s["previewSteps"] = dict(sorted(st.items()))
    return s


def usage_text(s):
    t = A3.usage_text(s).replace(f"({AGENT} {BASE_VERSION})", f"({AGENT} {VERSION})")
    return t + (f"preview step          {s['reviewRequests']} review requests; decision points by status "
                f"{', '.join(f'{k} {v}' for k, v in s['previewSteps'].items()) or 'none'}\n")


# --- main ----------------------------------------------------------------

def main(argv=None):
    ap = argparse.ArgumentParser(description="The DreamTeam LLM agent, preview-enabled (a benchmark policy).")
    ap.add_argument("--settings", help="settings file (dreamteam.llm-agent-settings/v1); required live")
    ap.add_argument("--replay", action="append", default=[], metavar="FILE",
                    help="submit the decisions recorded in FILE (decisions.jsonl) instead of asking a model; repeatable")
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
            "promptVersion": PROMPT_VERSION, "memoryFormat": MEMORY_FORMAT, "treatmentVersion": TREATMENT_VERSION,
            "treatmentSha256": TREATMENT_SHA256, "agentSha256": file_sha256(os.path.abspath(__file__)),
            "baseAgentSha256": file_sha256(BASE_PATH), "mode": "replay" if recordings else "live",
            "python": sys.version.split()[0]}
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
    records, checked = [], False

    def finish():
        s = summary(ev, records, settings)
        ev.write("summary.json", s)
        ev.text("usage.txt", usage_text(s))
    try:
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
                rec = replay_point(h, ev, replayer, point)
                batches = rec["batches"]
            else:
                try:
                    batches, rec = ask(h, ev, client, settings, memory, point)
                except ProtocolError as e:
                    # Never expected (the agent uses permitted operations
                    # only); recorded as such, and refused by replay.
                    batches, rec = [], {"schema": RECORD_SCHEMA, "point": point, "time": None, "mode": "live",
                                        "agentVersion": VERSION, "baseAgentVersion": BASE_VERSION,
                                        "promptVersion": PROMPT_VERSION, "memoryFormat": MEMORY_FORMAT,
                                        "treatmentVersion": TREATMENT_VERSION, "treatmentSha256": TREATMENT_SHA256,
                                        "protocolError": str(e), "outcome": "fallback", "batches": [],
                                        "errors": [f"protocol: {e}"], "attempts": [],
                                        "final": {"attempt": None, "outcome": "fallback"},
                                        "preview": review_summary([]),
                                        "completeness": completeness([], "protocol_error", [], recorded=False),
                                        "digests": None}
            records.append(rec)
            if memory and rec.get("time") is not None:
                memory.record(rec)
            ev.append("decisions.jsonl", rec)
            finish()
            h.decide(batches)
    except (ExchangeError, ReplayError) as e:
        print(f"{AGENT}: {e}", file=sys.stderr)
        return 1
    finally:
        finish()


if __name__ == "__main__":
    sys.exit(main())
