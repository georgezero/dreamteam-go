# Three decision workflows from one proposal (`shared-prefix/v4`)

The shared-prefix method with whole decision workflows as arms. Earlier
versions compared single messages ([../v10c-dev/SPEC.md](../v10c-dev/SPEC.md)).
Code: `analysis/repeatability/workflow_experiment.py`. Protocol schema:
`dreamteam.shared-prefix/v4`.

## 1. A branch point

As in v3 (state, recorded request, recorded initial staffing proposal,
context key), with two more conditions:
* **No refusal before the proposal.** If the recorded agent had a
  proposal refused earlier at the point, its next message was llm-agent
  3's retry message, which llm-agent 5 would not have sent. The arms'
  conversations would then differ before the branch, so such a point is
  excluded.
* **Budgets carried over.** Failures and a details request already used
  before the proposal count, as they would in the agent.

## 2. The arms

Each arm starts from the same recorded state, the same messages and the
same recorded proposal, taken as the model's first reply. It then runs its
whole workflow, with live model calls when the workflow needs them.

| Arm | Workflow | After a refused proposal | After a valid proposal |
|---|---|---|---|
| **A** llm-agent 3 | its validation and retry loop | version 3's retry message: `Your reply could not be used: refused: action i: code: message; … Reply again …` | submitted (no model call) |
| **B** llm-agent 5, preview disabled (`v11b-B-validation-only`) | validation first | `llm-agent-validation-feedback/v1`: the engine's results for the refused actions, verbatim | submitted (no model call) |
| **C** llm-agent 5 (`llm-agent-workflow/v1`) | validation first, selective preview | as B | the preview of its staffing actions, shown if eligible (`llm-agent-preview-eligibility/v1`, unchanged); the model keeps, revises or abandons; the reply validated again |

**Common to all three:**
* `maxRetries` 1 and `allowDetails`, from the frozen settings;
* an unreadable reply or a protocol error is answered with version 3's
  retry message, and a failed request is resent; both count against the
  retries;
* a details request is answered while still open;
* only a fully validated proposal is submitted. Otherwise nothing is.

**B is an experiment configuration.** The driver runs llm-agent 5's
functions with the preview step turned off. llm-agent 5 itself is
unchanged, and no production agent has this mode.

**Fidelity.** `run_arm` follows each agent's own loop and uses the agents'
functions: the parser, batching, retry message, feedback message, preview
request, eligibility rule and preview message.
`test_workflow_experiment.py` checks, on llm-agent 5's scripted engine
and model, that for the same replies and answers:
* A sends exactly llm-agent 3's messages and submits its batches;
* B sends llm-agent 5's when no preview is shown;
* C sends llm-agent 5's.

The cases cover refusals, corrections, abandonment, repeated refusals,
unreadable replies, mixed batches, previews kept, revised or abandoned,
failed previews and timeouts.

**Unavoidable differences.** The workflows make different numbers of
model calls:
* at a valid proposal, A and B make none and C makes one when the
  preview is shown;
* at a refused proposal, each makes one per retry, and C makes one more
  when the corrected proposal's preview is shown.

So the arms do not have equal inference exposure. The calls are counted
for every draw. At a valid proposal, A and B are deterministic: the same
submission in every draw.

## 3. Draw, outcomes and continuation

A draw is one arm's whole workflow at one point. Its outcome against the
proposal is one of:
* **kept, revised or abandoned:** llm-agent 4's rule;
* **refused:** the workflow ended on a refusal;
* **invalid:** it ended on an unreadable reply or a failed request.

The final decision is accepted, no action, or none submitted. Also
recorded:
* the model calls and the retries;
* the preview's status and eligibility;
* the response after a shown preview;
* the final action content.

Each distinct final decision is evaluated at the recorded state with
financial evidence v1 and the fixed no-action suffix (shared-prefix/v2).

## 4. Records and replay

Each draw is recorded as it ends (`live/draws.jsonl`):
* its schedule entry;
* every model call: request SHA-256, the full response, its SHA-256, the
  content, finish reason, usage, latency and status;
* the workflow's trace: every attempt, its parse, actions, validation
  answers, refusals, preview record, the next message's kind, and the
  submitted batches;
* every engine request and answer.

The analysis runs every draw's workflow again:
* with the recorded responses, each matched to the request the workflow
  now sends, by SHA-256;
* on a fresh engine session at the point.

The trace and every engine answer must equal the recorded ones. It
contacts no model. Live metadata (latency, tokens, cache) is reported
apart in `inference.json`.

## 5. What a comparison can say

**Supported** (at each point, for that state, conversation and proposal):
* C − B: the difference the selective preview makes within the
  validation-first workflow;
* B − A: the difference between llm-agent 5's validation-first feedback
  and llm-agent 3's retry loop.

**Assumptions:**
* the workflows share everything before the proposal (checked);
* assignment by the schedule (holds);
* inference noise exchangeable between arms;
* no interference through the shared cache;
* draws are repetitions of one workflow, not independent cases.

**Not supported:**
* effects beyond these points, models and server;
* separating a workflow's message content from its extra model call;
* long-run outcomes;
* the reasons for a decision.
