# V0.12b Stage B: controlled acknowledgment evaluation (proposed protocol)

**Status: proposed, for owner approval. Not frozen and not run.** No
request of this experiment has been sent, and no live model was contacted
to prepare it.

| File | SHA-256 |
|---|---|
| Protocol: `protocol.json` (`v12b-ack-1`) | the SHA-256 is listed in `README.md` |
| Points: `points/manifest.json`, `points/contexts.jsonl` | as listed in `README.md` |
| Schedule: `schedule.json` | as listed in `README.md` |

Freezing happens in two steps (§8). First the driver and its fidelity
tests are committed. Then the final `protocol.json`, with the driver's
hash, and its schedule are committed. Both come before any request.

## 1. The question

From the same recorded state, conversation and initial staffing
proposal, does requiring an exact acknowledgment of the engine-reported
consequences change what the agent decides? The comparison is llm-agent 6
(B) against llm-agent 5's validation-first preview workflow (A).

The central distinction is between two outcomes:
* **acknowledged and proceeded:** the model states the consequences
  exactly and submits the action unchanged;
* **acknowledged and reconsidered:** after the request, it revises or
  abandons the action.

* **Proceeding** shows that the safeguard works mechanically, not that
  judgment improved.
* **A revision** shows a behavioural effect. It is not necessarily a
  better outcome.

Neither is scored.

## 2. Branch points

**Selection:** `dreamteam-v12b-select/1`
(`analysis/acknowledgment/stage_b_points.py`). It reads no model
response and is reproducible: `points --check` makes it again, and CI
compares the result.

**The pool:** the V0.10c/V0.11b inventory of 56 staffing proposals
(`docs/benchmark/v10c-dev/points/inventory.json`, with eligibility from
the V0.11b manifest), restricted to proposals that meet two conditions:
* **Eligible under shared-prefix/v4:** no refusal before the proposal at
  its point.
* **A committed context:** the recorded request and proposal are already
  in `docs/benchmark/v10b-dev`, `v10c-dev` or `v11b-dev`
  (`points/contexts.jsonl`). Each is checked again against its recording.

That leaves 20 proposals. No trajectory is made or extended, and no
context is rebuilt from live material outside the repository.

**Rule:**
* **Gemma:** every pooled context, with at most 2 per recording.
* **Qwen:** the categories below, in order. Each takes the free candidate
  with the largest stake. Ties are broken by
  SHA-256(`dreamteam-v12b-select/1|<recording>|<point>`).
* **Limits:** a context is taken once, and at most 2 points come from one
  recording.
* **Empty category:** recorded as unavailable, never filled from another
  category.

**Selected:**

| Point | Model | Category | Proposal | Qualifying items (B) | Displaced | Stake | Refusals | Prior exposure |
|---|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | Gemma | gemma_valid | teleradiology start | 1 | 0 | $3,069.98 | — | V10B, V10C, V11B |
| fl-gemma-s41901-p28 | Gemma | gemma_refused | teleradiology | (1 if corrected) | 0 | $0.00 | invalid_schema | V10B, V10C, V11B |
| fl-gemma-s41901-p26 | Gemma | gemma_refused | teleradiology | (0) | 0 | $0.00 | invalid_schema | V10B, V10C, V11B |
| fl-qwen-s41902-p24 | Qwen | termination_displacement | two terminations | 3 (2 terminations, batch displacement) | 6 | $0.00 | — | V11B |
| ss-qwen-s42002-p7 | Qwen | teleradiology_guarantee | teleradiology start | 1 | 0 | $3,228.33 | — | V10C, V11B |
| ss-qwen-s42001-p20 | Qwen | multiple_qualifying | teleradiology start and roster | 2 (start, batch displacement) | 405 | $1,475.80 | — | V10C |
| ss-qwen-pv-s42002-p6 | Qwen | valid_expensive | allocation (credit-supported) | 0 | 0 | $44,687.30 | — | V10B, V10C, V11B |
| fl-qwen-s41902-p27 | Qwen | refused_termination | two terminations | (2 if corrected) | 0 | $0.00 | notice_too_short | V11B |
| fl-qwen-s41901-p9 | Qwen | refused_hire | hire and allocation | (1 if corrected) | 0 | $36,978.91 | insufficient_funds | V10C, V11B |
| fp-qwen-s42001-p7 | Qwen | refused_hire | hire | (1 if corrected) | 0 | $0.00 | lead_too_short | V11B |
| ss-qwen-s42001-p23 | Qwen | refused_teleradiology | teleradiology start | (1 if corrected) | 0 | $0.00 | past_date | V10C |
| fp-qwen-s41901-p25 | Qwen | nonqualifying_control | allocation | 0 | 0 | $9,863.02 | — | V11B |

* **Qualifying items:** what llm-agent 6 would ask to be acknowledged for
  the proposal as made, from its recorded actions and the inventory's
  preview fields. For a refused proposal they apply only if a correction
  of the same actions becomes valid. The engine decides at run time.
* **Displaced:** the inventory's count for the recorded proposal.

**Unavailable categories:**
* **termination_severance.** Every valid termination in the pool also
  displaces bookings. Severance is therefore covered only by
  fl-qwen-s41902-p24, which has both. There is no termination point
  with severance and no displacement.
* **hire_obligation.** No valid hire exists anywhere in the corpus;
  every recorded hire was refused. A hire's retained obligation can be
  reached only when a refused hire (fl-qwen-s41901-p9, fp-qwen-s42001-p7)
  is corrected into a valid one.

**Not filled from elsewhere:** these gaps are reported as limits. Under
the owner's rule, they are not filled with new trajectories.

**Prior exposure:** every pooled point was in an earlier experiment,
because only those have committed contexts. The rule reads no response of
those experiments and does not select by whether a difference appeared.
It was written knowing their results existed.

**Limits of the corpus:**
* Gemma has one valid point.
* The largest recorded displacement (100) came from a revision by a model
  in V0.11b, not from a recorded initial proposal. It is not a branch
  point here.
* fl-qwen-s41902-p24 displaces 6 bookings.
* ss-qwen-s42001-p20 displaces 405 bookings, through a roster change
  alongside a teleradiology start.

## 3. Arms

Every arm starts from the same four things:
* the recorded state, an engine session on the recording's replay at the
  recorded digests;
* the recorded conversation;
* the recorded initial proposal, taken as the model's first reply;
* the same model, frozen settings (`maxRetries` 1, `allowDetails`) and
  action catalog.

| | A: llm-agent 5, validation-first-preview | B: llm-agent 6, acknowledgment |
|---|---|---|
| Workflow | `llm-agent-workflow/v1` (`3ee7d66f…`), V0.11b's arm C, unchanged | `llm-agent-acknowledgment-workflow/v1` (`b9cb7982…`), contract `dreamteam.action-acknowledgment/v1` (`33c116da…`) |
| Refused proposal | `llm-agent-validation-feedback/v1` | the same |
| Valid proposal, qualifying | the selective preview, if `llm-agent-preview-eligibility/v1` finds a reason; keep, revise or abandon | the acknowledgment request; exact match; final re-validation and re-preview; submit or withhold |
| Valid proposal, not qualifying | the selective preview, if eligible | submitted, with no model call |

**Differences beyond the acknowledgment requirement (unavoidable).**
llm-agent 6 is built on llm-agent 5's validation-first workflow without
the selective preview. A as specified includes the preview. So:
* **At qualifying points:** A's preview message and B's acknowledgment
  request show the consequences in different forms. Only B requires the
  exact values to be restated, and only B withholds on a mismatch.
* **At non-qualifying points** (valid_expensive, nonqualifying_control,
  and corrections that do not qualify): A may show a preview, while B
  calls no model. B − A at those points measures the preview's absence,
  not the acknowledgment.
* **Model calls differ by arm** and are counted per draw, not equalized.
  The bound is 4 per draw for both arms: the proposal is recorded; then
  at most one review, preview or acknowledgment reply, one retry
  (`maxRetries` 1), and one details answer. The driver's tests confirm
  this bound (§8).

## 4. Draws, order and running

* **Draws per arm per point:** Gemma 10, Qwen 20, as in V0.10c and
  V0.11b. In V0.10b, V0.10c and V0.11b, Gemma gave one decision per arm
  at every point, and Qwen gave 1–6 distinct decisions per arm in 20
  draws.
* **Totals:** 3 × 2 × 10 + 9 × 2 × 20 = **420 draws**, at most **1,680
  model calls**. Most draws need fewer. V0.11b made 572 calls in 720
  draws, in 37 minutes.
* **Order** (`dreamteam-v12b-order/1`):
  * Gemma's draws, then Qwen's;
  * per model, each round has every point and arm once, in the order of
    SHA-256(`dreamteam-v12b-order/1|<model>|<round>|<point>|<arm>`);
  * the exact order is `schedule.json`.
* **One draw, and one model request, at a time.** No concurrency.
* **Models and server:**
  * the models are `gemma-4-26B-A4B-it-oQ4e-mtp` and
    `Qwen3.6-35B-A3B-oQ4e-mtp` on the same oMLX server as V0.11b,
    reached through `DREAMTEAM_LLM_BASE_URL`, which is never recorded;
  * no substitute model is used;
  * no admin call, load, unload, cache clear or restart.
* **Request:** the recorded proposal request, unchanged (temperature 0,
  `max_tokens` 4096, json_object, no seed), followed by the workflow's
  turns. Timeout 300 s, with no retries beyond the workflow's own.
* **Engine:** read-only requests only (validate, preview, observe,
  digests). Every request and answer is recorded.
* **Client:** the Pi, under `systemd-run --user --scope -p MemoryMax=2G
  nice -n 10`.
* **Server checks:** before the run, before each round and after.
* **Stop conditions:**
  * 5 consecutive transport failures;
  * 4 hours of wall clock;
  * an unhealthy `/health`;
  * a material server change.

  A stopped run is reported and analysed as far as it went. It is never
  resumed without the owner.
* **Failures:** every request is recorded. A failure is handled only by
  the arm's own workflow, never rerun outside it.
* **Inference budget:** at most 1,680 calls. Expect 1–2 hours, because
  acknowledgment replies restate the consequences and are longer than
  preview replies. The 4-hour stop is the time limit. There is no token
  cap beyond `max_tokens` per request. Token usage is recorded per call.

## 5. Outcomes

These are defined before any request. Each applies per model, point and
arm, with every draw counted.

**Per draw:**
* **The proposal:**
  * kept, revised or abandoned (llm-agent 4's rule: the final submitted
    actions against the initial proposal, by canonical content);
  * invalid (the workflow ended on an unreadable reply or a failed
    request).
* **The final proposal:**
  * whether it was valid, its last validation accepted;
  * whether it was submitted, handed to the harness;
  * whether it was executed: applied at the recorded state in the
    continuation (§7), and accepted by the engine.
* **The acknowledgment** (B):
  * whether it was requested, how many times, and for which proposals;
  * how many were **accurate** (accepted by the exact matcher);
  * how many failed, with each failure code;
  * the final check's agreement.
* **Retries, model calls, token usage** (prompt and completion, as the
  server reports) and **latency**. Latency and tokens are live metadata,
  reported apart in `inference.json`, outside the deterministic replay.

**The central classification** (B, at every draw):

| Class | Definition |
|---|---|
| **A. Acknowledged and proceeded** | An acknowledgment of the initial proposal was accepted, and the submitted actions equal the initial proposal. |
| **B. Acknowledged and revised** | The acknowledgment request for the initial proposal was shown, and the submitted actions are a different non-empty proposal. If that proposal qualifies, it has its own accepted acknowledgment. |
| **C. Acknowledged and abandoned** | The request was shown, and the model chose to submit nothing (`"actions": []`). |
| **D. Failed acknowledgment, withheld** | The request was shown, no accepted acknowledgment led to a submission, and the model did not abandon. The proposal was withheld (`acknowledgment_not_accepted`, `final_check_disagrees`, `consequences_unavailable`, or the budget). |
| **E. No acknowledgment required** | No request was shown: a non-qualifying proposal, a refused proposal not corrected into a qualifying one, or an ending before validation succeeded. |

* **Copying is not the same as reconsidering.** An accurate
  acknowledgment counts as A only when the action is unchanged; a changed
  action is B or C, whatever the acknowledgment said.
* **No reasoning is inferred** from the acknowledgment text or the
  reason. Classes B and C record that the decision changed, not why.
* **A's matching classes:** kept, revised or abandoned after a shown
  preview; no preview shown.

**Financial and operational exposure** (financial evidence v1,
unchanged): for each distinct final decision against the initial
proposal, report:
* guaranteed commitments;
* severance;
* retained obligations;
* credit-supported amounts;
* displaced bookings;
* direct capacity changes;
* reversibility.

A smaller commitment is not called better. An acknowledgment is not
evidence of a wise decision.

## 6. Statistics

* **Per branch point first:**
  * the counts of each outcome and class, with their denominators;
  * Wilson 95% intervals per arm;
  * Newcombe 95% intervals for B − A, per outcome;
  * whether the two arms' distributions of final action content are
    identical.
* **Then summarised per model and category:** per-point differences and
  their unweighted mean, with no interval across points (the points are
  clustered and few).
* **Kept separate:**
  * the workflow effect (between arms);
  * inference variability (within an arm);
  * branch-point heterogeneity;
  * Gemma against Qwen;
  * financial exposure.
* **Not done:**
  * no hypothesis test;
  * no composite score;
  * no model ranking;
  * no population-level claim.

  Draws are repetitions of one workflow at one state, not independent
  practice cases.

## 7. Replay and continuation

* **Recording:** each draw is recorded as it ends (`live/draws.jsonl`):
  * its schedule entry;
  * every model call: request SHA-256, the full response and its
    SHA-256, the content, finish reason, usage, latency and status;
  * the workflow trace: every attempt, its parse, validation answers,
    the consequence assessment (the engine's preview answer and its hash,
    the items, the payload and token), the acknowledgment (given, match,
    failures), the final check, the stages and the submitted batches;
  * every engine request and answer.
* **Replay, without a model:** every draw is run again on a fresh engine
  session at the point, with the recorded responses matched to the
  requests by SHA-256. The trace, every engine answer, the consequence
  payloads, the acknowledgment matches and the final actions must equal
  the recorded ones.
* **Continuation:** each distinct final decision is applied at the
  recorded state, if valid. The no-action suffix of shared-prefix/v2
  follows, with financial evidence v1, recording the apply outcome, the
  state digests and the financial ledger. It is a mechanics check with
  no long-run claim.
* **Provenance** (the owner's F2 decision):
  * replay establishes that the record is consistent with the recorded
    replies, not that a particular model produced them;
  * the run's request and response records are preserved with it;
  * no response signing.
* **Cross-platform:** CI regenerates the points check, the schedule and
  the analysis, and compares them on every platform.

## 8. Before the run (no live calls)

1. **The driver:** `analysis/acknowledgment/stage_b.py`.
   * Arm A is V0.11b's `workflow_experiment.run_arm("C")`, unchanged.
   * Arm B follows llm-agent 6's own loop from the recorded proposal on,
     with llm-agent 6's own functions: assessment, request and rejection
     texts, matching, final check and withholding.
2. **Fidelity tests:** on llm-agent 6's scripted engine and model, for
   the same replies, B must send exactly llm-agent 6's messages and
   submit or withhold exactly its batches. The tests cover every V0.12a
   scripted case and the F1 field cases. A must send llm-agent 5's, as
   in V0.11b. The tests also confirm the bound of 4 model calls per
   draw.
3. **Freeze:**
   * add the driver's SHA-256 to `protocol.json`;
   * regenerate `schedule.json`;
   * commit both, with CI green, before any request.

   Any change after that is an amendment, recorded with its reason.

## 9. What a comparison can say

* **Supported:**
  * at each point, for that state, conversation and proposal, whether B
    and A lead to different immediate decisions;
  * at B, how often accurate acknowledgment is followed by proceeding,
    revising or abandoning.
* **Assumptions:** as in shared-prefix/v4 (V0.11b SPEC §5):
  * the shared prefix is checked;
  * assignment is by the schedule;
  * inference noise is exchangeable between arms;
  * there is no interference through the shared cache.
* **Not supported:**
  * effects beyond these points, models and server;
  * separating the acknowledgment's content from its extra model call,
    or from the absence of the preview at non-qualifying points;
  * whether a revision is a better decision;
  * long-run outcomes;
  * the reasons for a decision.

## Amendments

None. This protocol is not yet frozen.
