#!/usr/bin/env python3
"""The management-alternatives workflow (V0.13d): llm-agent 5's
validation-first workflow, unchanged, then one final-decision turn built by
the engine.

Specification: docs/v13d/WORKFLOW.md. Identity: llm-agent-alternatives-
workflow/v1 (WORKFLOW_VERSION, WORKFLOW_SHA256). It is an experimental
workflow for the V0.13d experiment driver; no earlier agent changes.

The steps, for one decision point and one arm (control or treatment):
1. The model proposes; every batch is validated; a refusal is shown and the
   model may correct it: llm_agent5.ask(..., workflow="validation-first"),
   called unchanged. Its record is kept whole.
2. If it ends with a fully validated proposal that has actions, the engine
   is asked for the arm's turn (alternatives.turn, served by
   `dreamteam agent --stdio -alternatives`). The engine decides whether the
   turn is due, by one test for both arms, and makes its text from the
   comparison it has bound to the decision point (DecodeAgentAt) and the
   typed view (dreamteam.alternatives-view/v1). If it is not due, the
   validated proposal is submitted, as validation-first does.
3. If it is due, the turn's message is sent, verbatim, as the next user
   message of the same conversation. Each model reply goes to the engine
   (alternatives.decide), which reads it and answers either with a fixed
   text to send next (the inspect answer, the once-only notice, the retry
   text) or with the classified outcome. The workflow sends those texts
   verbatim and nothing else.
4. Limits: one inspect round; one retry for an unreadable reply or a
   transport failure (TURN_RETRIES); a second inspect request counts as a
   failure. When the retries are spent, the result is no_decision and
   nothing is submitted.
5. The outcome's batches (the reply's operations batch kept with select, a
   revision replacing both, abandon submitting none; each validated by the
   engine at the decision point) are submitted, after a check that nothing
   changed the state. Otherwise nothing is.

Nothing here reads or interprets the model's reason. No text is made here.
"""

import hashlib
import importlib.util
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))


def _load(name, path):
    m = sys.modules.get(name)
    if m is None:
        spec = importlib.util.spec_from_file_location(name, path)
        m = importlib.util.module_from_spec(spec)
        sys.modules[name] = m
        spec.loader.exec_module(m)
    return m


A5 = _load("llm_agent5", os.path.join(HERE, "llm_agent5.py"))
A3 = A5.A3
sha256 = A3.sha256

WORKFLOW_VERSION = "llm-agent-alternatives-workflow/v1"
RECORD_SCHEMA = "dreamteam.alternatives-decision-record/v1"
BASE_WORKFLOW = "validation-first"
ARMS = ("control", "treatment")
TURN_RETRIES = 1
INSPECT_ROUNDS = 1
# The engine's pinned texts this workflow sends (internal/alternatives).
PRESENTATION = "dreamteam.alternatives-presentation/v2"
ARMS_ID = "dreamteam.alternatives-arms/v2"
RULES = (
    "1 validation-first: llm_agent5.ask(workflow=validation-first), unchanged",
    "2 turn: alternatives.turn {arm, actions: the used attempt's actions}, only when validation-first ends with actions",
    "3 not due: submit validation-first's batches",
    "4 due: send turn.message as the next user message, after the used attempt's reply",
    "5 replies: alternatives.decide {arm, actions, inspected, round, content}; send answer verbatim unless the kind is decision",
    "6 limits: inspect rounds 1; retries 1 (unreadable, transport failure, a second inspect); then no_decision",
    "7 submit: the outcome's batches, if valid and the digests are unchanged; otherwise nothing",
)
WORKFLOW_SHA256 = sha256("\n".join((WORKFLOW_VERSION, A5.VALIDATION_FIRST_SHA256, PRESENTATION, ARMS_ID) + RULES))


class Capture:
    """The agent's model interface, passed through, keeping the last
    request's messages and the last reply's content (the conversation the
    turn continues)."""

    def __init__(self, model):
        self.model, self.messages, self.content = model, None, None

    def payload(self, messages):
        self.messages = messages
        return self.model.payload(messages)

    def call(self, payload):
        status, body, err = self.model.call(payload)
        self.content = None
        if err is None:
            try:
                self.content = A3.completion(body)[0]
            except ValueError:
                pass
        return status, body, err

    def __getattr__(self, name):  # spent() and anything else the agents use
        return getattr(self.model, name)


def used_attempt(rec):
    """The attempt validation-first used (its actions are the reply the
    turn is about)."""
    k = (rec.get("final") or {}).get("attempt")
    for at in rec.get("attempts", []):
        if at["attempt"] == k:
            return at
    return None


def ask(h, ev, model, settings, memory, point, prev=None, *, arm):
    """One decision point under the named arm. Returns (batches, record)."""
    if arm not in ARMS:
        raise A3.SettingsError(f"unknown arm {arm!r}")
    cap = Capture(model)
    batches, base = A5.ask(h, ev, cap, settings, memory, point, prev, workflow=BASE_WORKFLOW)
    rec = {"schema": RECORD_SCHEMA, "workflowVersion": WORKFLOW_VERSION, "workflowSha256": WORKFLOW_SHA256, "arm": arm,
           "validationFirst": base, "turn": None, "exchanges": [], "outcome": None, "result": None,
           "batches": batches, "digestsBeforeSubmit": None, "errors": []}
    used = used_attempt(base)
    if base["outcome"] != "actions" or used is None or cap.messages is None or cap.content is None:
        rec["result"] = "not_due:validation_first_" + base["outcome"]
        return batches, rec
    actions = used["actions"]
    turn = h.request("alternatives.turn", {"arm": arm, "actions": actions})
    rec["turn"] = turn
    if not turn["due"]:
        rec["result"] = turn["why"]
        return batches, rec
    clock = h.request("digests")
    messages = list(cap.messages) + [{"role": "assistant", "content": cap.content},
                                     {"role": "user", "content": turn["message"]}]
    inspected, rounds, failures, outcome = [], 0, 0, None
    while outcome is None and failures <= TURN_RETRIES:
        payload = model.payload(messages)
        status, body, err = model.call(payload)
        x = {"n": len(rec["exchanges"]) + 1, "inputSha256": sha256(messages[-1]["content"]), "status": status,
             "error": err, "contentSha256": None, "kind": None, "answerSha256": None}
        content = None
        if err is None:
            try:
                content = A3.completion(body)[0]
            except ValueError as e:
                x["error"] = str(e)
        if content is None:
            failures += 1
            x["kind"] = "transport"
            rec["exchanges"].append(x)
            continue  # the same messages again
        x["contentSha256"] = sha256(content)
        a = h.request("alternatives.decide", {"arm": arm, "actions": actions, "inspected": inspected,
                                              "round": rounds, "content": content})
        x["kind"], x["answerSha256"], x["parse"] = a["kind"], a["answerSha256"], a["parse"]
        rec["exchanges"].append(x)
        if a["kind"] == "decision":
            outcome = a["outcome"]
            break
        if a["kind"] == "inspect":
            rounds += 1
            inspected += [i for i in a["inspected"] if i not in inspected]
        else:  # unreadable, or a second inspect
            failures += 1
        messages = messages + [{"role": "assistant", "content": content}, {"role": "user", "content": a["answer"]}]
    rec["inspected"] = inspected
    if outcome is None:
        rec["result"] = "no_decision"
        rec["outcome"] = {"result": "no_decision", "why": f"no decision after {TURN_RETRIES} retry", "batches": []}
        rec["batches"] = []
        return [], rec
    rec["outcome"], rec["result"] = outcome, outcome["result"]
    out = outcome["batches"] if outcome["valid"] else []
    after = h.request("digests").get("digests")
    rec["digestsBeforeSubmit"] = {"unchanged": after == clock["digests"]}
    if after != clock["digests"]:
        rec["errors"].append("state_changed: the digests changed during the turn; nothing is submitted")
        out = []
    rec["batches"] = out
    return out, rec
