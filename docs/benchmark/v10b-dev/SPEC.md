# Shared-prefix comparison, version 2 (`shared-prefix/v2`)

The live version of the V0.10a design ([../v10a-dev/DESIGN.md](../v10a-dev/DESIGN.md),
`shared-prefix/v1`). It keeps v1's shared prefix and shared initial
proposal. It changes three things:
* what follows the proposal in the control arm;
* where the replies come from;
* how many replies each arm has.

Code: `analysis/repeatability/decision_experiment.py`. Protocol schema:
`dreamteam.shared-prefix/v2`.

## 1. A branch point

A branch point is a decision point of a committed recording at which the
recorded agent made a staffing proposal. That is the first parsed reply
at the point with a staffing action, which is llm-agent 4's preview rule.
The point carries:

| Item | What it is | How it is checked |
|---|---|---|
| State | the recorded digests at the point | the recording's replay transcript is fed back to the engine (`dt agent --stdio`), and every recorded answer must come back identical up to these digests |
| Conversation | the exact request the agent sent for the proposal (system prompt, prompt, any earlier exchange at the point), from the live run's `model.jsonl` | each message against the recording's SHA-256 chain: system prompt, `promptSha256`, earlier outputs and inputs |
| Initial proposal | the model's recorded reply | its SHA-256 equals the recorded output's |
| Preview | the engine's answer to `preview` of the proposal's staffing actions, asked at the recorded state | deterministic. At a V0.9c point where llm-agent 4 showed it, the message it makes is the recorded review input, byte for byte |
| Context key | SHA-256 of the model, the digests, the message chain and the proposal's hash | two candidates with one key are one context |

## 2. The arms

Both arms send the same request up to and including the proposal:
* the recorded request's settings (model, temperature 0, `max_tokens`
  4096, `response_format` json_object), with nothing added;
* its messages;
* the proposal as the assistant's reply.

Then one user message, which is the only designed difference:

* **Control:**

  > CONTINUE. Nothing has been submitted yet.
  >
  > Now reply with your final decision, one JSON object {"reason": ..., "actions": [...]}. It replaces your reply above: give the same actions to keep it, changed actions to revise it, or "actions": [] to submit nothing. It is then validated and submitted as before.

  305 characters, SHA-256 `bab212b0…`. Its second paragraph is the last
  paragraph of the treatment message, byte for byte.

* **Treatment:** llm-agent 4's review message, unchanged
  (`llm-agent-preview/v1`, `f8b7d787…`). It opens "PREVIEW. Nothing has
  been submitted yet. This is the simulator's preview of the staffing
  actions in your reply (actions …), exactly as the simulator returned
  it:", then:
  * the preview as compact JSON;
  * llm-agent 4's operations note, when the proposal also has operations
    actions;
  * the same final paragraph as the control.

What differs between the arms, besides the preview's content:
* the first word (CONTINUE or PREVIEW) and the sentence that introduces
  the preview;
* the operations note;
* the length: the treatment message is 809 to 5,370 characters at the
  selected points, the control 305.

The control is not a placebo for the preview's content. It controls for a
second turn, the same final request and the same response schema.

## 3. The reply

Each arm gets one reply per draw. The reply is the arm's final decision:
* it is read with llm-agent 3's strict parser, with no details request
  allowed;
* it is classified against the proposal with llm-agent 4's rule
  (`response_of`): **kept** (the same actions, in any order), **revised**
  (other actions) or **abandoned** (no action);
* a failed request or a reply the parser rejects is **invalid**.

There is no retry: a reply whose batches the engine refuses is recorded as
refused and is not submitted. That is a departure from the agents, which
would retry. It keeps each arm to one designed difference and one model
request.

## 4. Draws and order

* 20 draws per arm per point, one request at a time.
* Per model (Gemma, then Qwen), the draws are rounds. Each round has
  every point and arm of the model once, in the order of
  SHA-256(`dreamteam-v10b-order/1|<round>|<point>|<arm>`).
* Both arms of a point are therefore in every round, in an order that
  depends on neither the state nor the model.

## 5. After the reply: the fixed suffix

Each distinct final decision at a point is evaluated once on the engine,
at the recorded state:
* its batches (llm-agent 3's split) are previewed (staffing) and
  validated;
* if every batch is valid, each is applied in order, with the engine's
  observation after each;
* then every later day of the run is advanced with no action.

The record holds:
* the apply results and whether they agree with the validation;
* the digests after the decision and on each later day;
* the observed state at the end.

Financial evidence v1 (`analysis/evidence/financial_evidence.py`,
unchanged) is applied to these events:
* AFFORDABILITY per batch;
* REVERSIBILITY and OPERATIONAL_EXPOSURE per executed action.

The suffix is a mechanics and reproducibility check. Its long-run
outcomes are not attributed to the preview.

## 6. Records

* `points/`: every candidate with its stratum, eligibility, rank and
  context key; the selected points with their full contexts.
* `schedule.json`: every request in order.
* `live/requests.jsonl`, one line per request, written as it ends, failed
  or not. It holds:
  * the sequence number, point, arm and draw;
  * the request's SHA-256, status, error and latency;
  * the full response body and its content;
  * the finish reason, usage (cached tokens included) and served model.
* `live/server-*.json` and `live/run.json`: the server's public state
  before and after, a check before each round, and the run's counts and
  stop reason. The endpoint's address is never recorded.
* The analysis (`decisions.json`, `decision-tables.md`) is a function of
  the recorded responses, the points and the engine.
* Live metadata (`inference.json`: tokens, latency, cache) is kept
  apart.

## 7. Replay

The analysis contacts no model. It rebuilds every request from the points
and checks its SHA-256 against the recorded request. It also checks the
order against the schedule, and replays every engine answer from the
recordings' transcripts. CI regenerates the points' check, the schedule
and the analysis, and compares them byte for byte across platforms.

## 8. What a comparison can and cannot say

**Supported,** under the assumptions below: at each point, the
distribution of the final decision under each arm, given that state, that
conversation and that proposal, and the difference between the two. Both
are estimated from 20 draws per arm.

**Assumptions:**
1. The arms differ only by the user message after the proposal. This is
   checked byte for byte.
2. Arm assignment does not depend on the state or the model. This holds
   by the schedule's hash order, fixed before any request.
3. Inference noise is exchangeable between arms, apart from the message
   itself: the server's variation (V0.10a) does not depend on which arm
   sent a request. This is not checked. Interleaving within rounds
   spreads drift over both arms, but does not remove it.
4. There is no interference between requests: one request's answer does
   not depend on earlier ones, for example through the prompt cache. The
   cache is shared, and the arms share a prefix of tens of thousands of
   characters. This is observed (cached tokens), not controlled.
5. The draws of an arm are repetitions of one request. They are not
   independent samples of a population, and V0.10a showed that their
   variation depends on server conditions.

**Not supported:**
* an effect averaged over states, prompts, models or practices. The 8
  points are a declared selection, not a sample of a population;
* an effect of the preview's content apart from the message's length and
  wording;
* any long-run financial effect.
