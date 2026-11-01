# Shared-prefix comparison, version 3 (`shared-prefix/v3`)

Three arms at a fixed decision point. The method is v2's
([../v10b-dev/SPEC.md](../v10b-dev/SPEC.md)) with one more arm: the
engine's validation answer, beside v2's neutral continuation and v2's
preview. Code: `analysis/repeatability/feedback_experiment.py`. It
imports v2's module and changes none of it.

## 1. A branch point

As in v2 §1:
* the recorded state (digests, checked by replaying the recording's
  transcript);
* the exact recorded request for the initial proposal (every message
  checked against the recording's SHA-256 chain);
* the recorded proposal (its output SHA-256);
* the context key.

v3 adds the engine's two answers for the proposal's staffing actions,
asked at the recorded state, both read-only:
* `validate`: `{"valid", "results"}`;
* `preview`: `{"preview": {...}}`.

Both are committed with the point, and the analysis asks for them again
and compares.

## 2. The arms

Every arm sends the recorded request unchanged:
* model, temperature 0, `max_tokens` 4096, json_object;
* the same messages, then the proposal as the model's reply;
* then one user message.

All three messages end with the same final paragraph, byte for byte:

> Now reply with your final decision, one JSON object {"reason": ..., "actions": [...]}. It replaces your reply above: give the same actions to keep it, changed actions to revise it, or "actions": [] to submit nothing. It is then validated and submitted as before.

| Arm | Message before the final paragraph | Source |
|---|---|---|
| **A, neutral** | `CONTINUE. Nothing has been submitted yet.` | v2's control, unchanged (`bab212b0…`) |
| **B, validation** | `VALIDATION. Nothing has been submitted yet. This is the simulator's validation of the staffing actions in your reply (actions …), exactly as the simulator returned it:` then the validate answer as compact JSON; when the proposal also has operations actions: `The operations actions in your reply are not shown here (this covers the staffing actions only).` | new in v3, built like llm-agent 4's message |
| **C, preview** | llm-agent 4's review message: `PREVIEW. … the simulator's preview of the staffing actions in your reply (actions …), exactly as the simulator returned it:` then the preview answer as compact JSON, and llm-agent 4's operations note | v2's treatment, unchanged (`llm-agent-preview/v1`, `f8b7d787…`) |

* B and C cover the same actions (the proposal's staffing actions) and
  show the engine's answer verbatim. Nothing is summarized, added or
  invented.
* B differs from llm-agent 3's own feedback. llm-agent 3 shows nothing
  for a valid proposal. After a refusal it sends `Your reply could not be
  used: refused: action i: code: message; … Reply again …`, which holds
  the refused results' codes and messages and nothing else.
* B shows the whole validate answer, accepted results included, framed
  like C so that B and C differ by the engine answer they carry. B is
  therefore "the validation answer the agent could ask for", not
  llm-agent 3's retry message.

What each answer contains is in [API_INVENTORY.md](API_INVENTORY.md).

## 3. The reply, outcomes and continuation

As in v2 §§3 and 5:
* one reply per draw, read with llm-agent 3's strict parser;
* classified kept, revised, abandoned or invalid;
* no retry, and a refused final decision is not submitted;
* each distinct final decision validated, applied if valid, and followed
  by the fixed no-action suffix;
* financial evidence v1 applied to it, unchanged.

v3 adds, per draw, five separate facts (PLAN §5):
* the information the arm received (its message length; for C, the
  preview's sections);
* whether the proposal changed;
* whether it became valid;
* whether the commitment changed;
* whether the direct exposure changed.

## 4. Draws and order

* The draws per arm are fixed per model.
* Per model (Gemma, then Qwen), the draws are rounds. Each round has
  every point and arm of the model once, in the order of
  SHA-256(`dreamteam-v10c-order/1|<model>|<round>|<point>|<arm>`).
* One request at a time.

## 5. Records and replay

As in v2 §§6–7. In addition, every response's SHA-256 is recorded, and
the analysis checks:
* each response against its hash;
* each content against its response;
* each request against the points;
* the order against the schedule;
* the validate and preview answers against a fresh engine session.

Live timing, tokens and cache are in `inference.json`, apart from the
deterministic analysis.

## 6. What the comparisons can say

* **C − B (primary).** The difference in the final-decision
  distribution between seeing the preview and seeing the validation
  answer for the same actions, at that state, conversation and proposal.
* **B − A, C − A (references).** Each kind of feedback against none.

**Assumptions:** v2 §8's 1–5:
1. the arms differ only by the last message (checked);
2. assignment by the schedule (holds);
3. inference noise exchangeable between arms (unchecked);
4. no interference through the cache (unchecked);
5. draws are repetitions of one request.

**In addition:** C's message differs from B's in content and in length.
* At a refused proposal the preview carries the validation's results plus
  at most the funding test (API_INVENTORY §2). C − B there is mostly a
  difference of wrapper, length and the funding object.
* At a valid proposal the preview carries much more (terms, money lines,
  slots, displacements, the funding test). C − B there is a difference of
  information and length together, and the two cannot be separated.
* No length-matched control is used. Padding would be a new stimulus or
  invented content (PLAN §3).

**Not supported:**
* effects on other states, models, prompts or practices;
* any long-run effect;
* that a changed, smaller or valid decision is a better one.
