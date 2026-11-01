# V0.9c plan: prospective preview evaluation

Plan version `v09c-plan/1`, written 2026-10-31. It is committed before any
live run, and it is not amended after any result is seen.

**Question.** V0.9b showed that the simulator states, at the moment a
management decision is made, exactly what it commits the practice to: the
funding test, the guaranteed and forecast money lines, and its direct
effects, such as the bookings a termination displaces. No agent ever asked
for that preview. This evaluation asks one thing: **does showing the agent
the simulator's existing preview before it submits a staffing proposal
change the management decisions it proposes and executes?**

**What it is not:**
* **Not a ranking:** no composite score, no leaderboard, no significance
  test, no claim beyond these seeds, scenarios and models, and no clinical
  claim.
* **Not a forecast:** the preview reports immediate commitments and direct
  effects under the simulator's rules. It does not predict profit or
  turnaround, and it is never described as predictive.
* **Not tuned:** no prompt, model setting, memory policy, cadence, action
  schema, scenario or simulator mechanic is changed, before or after a
  result. The treatment's text is fixed here (§3).
* **Not a judgement:** a changed proposal is reported as a change. It is
  never called better, adaptive or understood because it changed, and the
  preview is not called beneficial because an agent changed its proposal.
* **Not causal at the level of one decision:** no single-decision
  intervention is run. Paired differences describe whole runs.

## 1. What is fixed

| | |
|---|---|
| Suite | `dreamteam-dev` version 4 ([suite file](../suite/dreamteam-dev-4.json), SHA-256 `e95b9158c8f4f84699ee0ab8b5f78ecbcb7683c59db86f8d1ac1e5771934d082`), unchanged since V0.8c |
| Financial pressure | `financial-pressure-credit` (scenario `bench-financial-pressure-ot-credit`, fingerprint `586f3ef74742af8c537e7ff41c41d586fafbfa75c4426043122595ba7c8795b1`): V0.7c's and V0.8c's control practice. Radiologist overtime paid; a $50,000 credit line at 10% a year; the funding rule; $150,000 opening cash |
| Staffing shortage | `staffing-shortage-credit` (scenario `bench-staffing-shortage-ot-credit`, fingerprint `5b49b4ca7b7304b39c814481b5de23fb631c620607d4f1ccbafb5cbf44c383fa`): V0.7c's staffing-shortage practice, with the same credit line and funding rule |
| Fingerprints | checked by the runner before anything runs |
| T90 | not applicable |
| Contract | `dreamteam.benchmark-contract/v1`, unchanged |
| Engine | 0.9.0-dev.3 (this commit; no engine behaviour changed since 0.8.0) |
| Cadence | daily decisions at 07:00 |
| Horizon | 30 days |
| Deterministic comparator | `noop` 1 (no action), unchanged |
| Models | `Qwen3.6-35B-A3B-oQ4e-mtp` and `gemma-4-26B-A4B-it-oQ4e-mtp`, the models of V0.6c, V0.7c and V0.8c |
| Model settings | The V0.5c files, unchanged since V0.5c, in both conditions: [llm-qwen36-35b-a3b.json](../v05c-dev/llm-qwen36-35b-a3b.json) (`e19a47dce948a8acbd3262ef5053b89f1e23c32603860e03628d38b184acbf5d`), [llm-gemma4-26b-a4b.json](../v05c-dev/llm-gemma4-26b-a4b.json) (`1a1edae5a492b67a72086bec51faecc8ed0930800bdb6e548412b65fa444aa6a`). Temperature 0, `maxOutputTokens` 4,096, one retry, 300 s per request, 900 s per decision, `json_object`, summary observation, guided catalog, details allowed |
| Memory | on: 6 decisions, 6 metric rows, 6,000 characters (`llm-agent-memory/v1`) |
| System prompt and prompt | llm-agent 2's, built in (SHA-256 of the system prompt `0efc36e21b592abc4acb4c8b78ee7634b39755580a0f5ac348a5be1f8a80c4e1`), `llm-agent-prompt/v2`, in both conditions |
| Server | the same local OpenAI-compatible server as V0.5c–V0.8c, reached through `DREAMTEAM_LLM_BASE_URL`, never committed; no key and no prices |
| Harness timeout | 1,800 s per decision point for the LLM agents |
| Configuration | [eval.json](eval.json) (SHA-256 `a15184ac9831ce84566bb42702cf8790ef2b5a53a031ebf0664645ebfea4ba41`) |

## 2. The two conditions

| Condition | Policy ids | Agent | SHA-256 |
|---|---|---|---|
| Baseline | `qwen36-35b-a3b`, `gemma4-26b-a4b` | llm-agent 3, [llm_agent3.py](../../../agents/llm/llm_agent3.py), unchanged since V0.8c | `ef2c17a06749d7c287cda918f77ff0e518d79475048e87fd3404848c2fb0015c` |
| Preview-enabled | `qwen36-35b-a3b-preview`, `gemma4-26b-a4b-preview` | llm-agent 4, [llm_agent4.py](../../../agents/llm/llm_agent4.py): llm-agent 3 loaded unchanged from the same file, with the preview step `llm-agent-preview/v1` | `4cfc6a6906556b0cda50e95fa39263089d56e848adcef9bd04342e7f6aff998f` |

**Identical across conditions:** the model identifier and settings, the
initial scenario state, the seed, the cadence, the action schema and
catalog, the memory configuration, validation, the economics and the
horizon. The system prompt, the prompt and the memory are built by the same
code, so **the model's first message at a decision point is the same
function, in both conditions, of the observation and the agent's own
earlier decisions** (checked by
`agents/llm/test_llm_agent4.py` and `internal/bench/llm_agent4_test.go`).

**The only treatment difference** is the preview step
([TREATMENT.md](TREATMENT.md)). At a decision point, the first parsed
proposal that has a staffing action is not validated at once. The agent:
1. asks the simulator for the protocol's existing `preview` of that
   proposal's staffing actions;
2. shows the model the answer, exactly as the simulator returned it, in the
   fixed message of §3;
3. takes the model's reply as the decision: the same actions (kept),
   changed actions (revised) or `"actions": []` (abandoned);
4. validates and submits that decision exactly as llm-agent 3 does (an
   operations batch first, both all-or-nothing, a refusal retried with
   version 3's feedback).

**Safe behaviour when the preview cannot be shown** (declared here):
* a protocol error answering `preview` (`failed`);
* a review message longer than `maxPromptChars`, 200,000 characters
  (`too_large`; a termination's preview lists every booking it displaces).

In both cases the preview is recorded but not shown, no review turn
happens, and the proposal goes to validation unchanged, as llm-agent 3's
would. The point is counted as a preview failure and is never reported as
previewed.

**At most once per decision point.** A revised proposal, or one made on a
retry after a refusal, is not previewed again. The details turn is not
offered after a preview. Operations actions (shift extensions, walk-in and
worklist policies) have no preview in the simulator: a proposal with none
of the staffing actions goes straight to validation, as in version 3.

**What the extra step adds** (unavoidable, and reported per run): one model
request per preview shown, and a longer conversation at that point (the
proposal and the review message, whose length is recorded).

## 3. The treatment's text

`llm-agent-preview/v1`, SHA-256
`f8b7d787b59c36373d4a18135c1503ccf64e3dd956c749be42fdfb0dbe5c54fa` (of the
template, a newline and the operations note; `TREATMENT_SHA256` in
`llm_agent4.py`):

> PREVIEW. Nothing has been submitted yet. This is the simulator's preview
> of the staffing actions in your reply (actions {indices}), exactly as the
> simulator returned it:
> {preview}
>
> {ops_note}Now reply with your final decision, one JSON object
> {"reason": ..., "actions": [...]}. It replaces your reply above: give the
> same actions to keep it, changed actions to revise it, or "actions": []
> to submit nothing. It is then validated and submitted as before.

`{indices}` are the staffing actions' positions in the reply, numbered as
in version 3's refusal feedback; `{preview}` is the simulator's answer as
compact JSON; `{ops_note}`, only when the reply also had operations
actions, is "The operations actions in your reply have no preview (the
simulator previews staffing actions only)." followed by a blank line.

It names no goal, gives no financial advice and adds no heuristic. The
system prompt is not changed.

## 4. Runs

| Entry | Seeds | Days | Policies | Runs |
|---|---|---|---|---|
| financial-pressure-credit | 42001, 42002 | 30 | noop, qwen36-35b-a3b, gemma4-26b-a4b, qwen36-35b-a3b-preview, gemma4-26b-a4b-preview | 10 (8 LLM) |
| staffing-shortage-credit | 42001, 42002 | 30 | the same | 10 (8 LLM) |

**In all:** 20 runs, of which 16 are live LLM runs, as the owner approved,
and 4 deterministic no-action runs. Each baseline run has a
preview-enabled run on the same model, entry and seed: 8 pairs.

**Seeds.** Fresh: the next unused block after V0.8c's 41900–41902.
* 42001 and 42002 are the evaluation seeds. No run of any kind used them
  before this plan.
* Neither appears as a seed elsewhere in the repository. They are not
  reference seeds (1001–1020), V0.2 development seeds (901–910), clinical
  calibration seeds, reserved-stream seeds or holdout seeds. No benchmark
  held-out seed is created and no held-out test is run.

**Checks before this plan, none on the evaluation seeds:** the agent's
recording and replay against the engine and a mock model server
(`internal/bench/llm_agent4_test.go`), and the configuration run for one
day on seed 777 against a stub server that answers no action (not
analysed, not kept). No live model was asked anything by llm-agent 4
before the evaluation.

## 5. The recording: `dreamteam.llm-decision-record/v4`

Specified in [LLM_AGENT.md §14](../../LLM_AGENT.md). It is llm-agent 3's
record (`dreamteam.llm-decision-record/v3`, §13), with:
* in each attempt, `preview`: the actions previewed and their indices, the
  status, the simulator's answer and its SHA-256 (or the protocol error),
  whether the digests were unchanged after it, and the review message's
  SHA-256 and length;
* the review turn, with input kind `preview`;
* at each decision point, `preview`: the step's status, the proposal,
  review and response attempts, the response (kept, revised, abandoned or
  no reply) for the decision as a whole and for its staffing and
  operations parts, and the decision finally submitted against the
  proposal;
* `treatmentVersion` and `treatmentSha256`.

The model's output, the parsed actions, the preview, the validation and
the submitted batches stay distinct; every attempt is kept, and attempts
go to `attempts.jsonl` as they are made.

## 6. Replay requirements

* **Every completed LLM run is replayed** without any model: baseline runs
  by llm-agent 3's replay, preview-enabled runs by llm-agent 4's. At every
  decision point the replay checks:
  * the digests;
  * the memory and prompt, rebuilt;
  * every attempt's input, rebuilt (the review message from the recorded
    preview);
  * every validate answer, asked again of the engine and compared;
  * every preview, asked again of the engine and compared in full, at
    exactly the attempt the rule puts it, with the digests unchanged after
    it;
  * that the submitted batches are the used attempt's actions, and the
    preview summary the attempts'.
* **Matching:** the replay's decisions, sessions and results must equal
  the live runs', including the financial ledger, credit draws, repayments
  and interest.
* **Configurations:** one replay configuration per entry.
* **Repeatability:**
  * each replay is reproduced (`benchmark reproduce`) on linux/arm64 and
    linux/amd64, with byte-identical manifests;
  * CI replays the recordings on three platforms and compares the
    manifests.
* **Live and replay are kept apart:**
  * Live inference (latency, tokens, served model) stays in the run's
    policy evidence and is reported separately.
  * The model exchanges and the agent's settings record (with the server's
    address) stay outside the repository.

## 7. Order, resources and stop conditions

**Order.** One suite run ([eval.json](eval.json)), on a Raspberry Pi 4
(linux/arm64):
* the runs go one at a time, in the runner's order;
* it is the only heavy job on the machine, run with a memory cap
  (`GOMEMLIMIT=5GiB`, `MemoryMax=6G`, `nice 10`).

**Limit:** 6 hours. At the limit the command is stopped. The completed
runs are reported, and the evaluation is reported as partial. No horizon
is shortened.

**Failures:**
* **A model that cannot be loaded or served** is recorded as unavailable.
  It is never replaced by another model.
* **Inside a run:** a request that fails is retried once by the agent (the
  frozen setting). A decision that still fails is a fallback (no action):
  counted, reported, and the run continues. Points where every attempt
  failed at the transport (timeout, connection, HTTP) are reported
  separately as infrastructure-affected.
* **A run that fails** (status not `completed`) is reported with its status
  and evidence.
  * If the cause is infrastructure (the server, the connection or the
    machine, not the agent or the simulator), it is rerun once, in full,
    with its own configuration (the same entry, seed and policy), into a
    new directory.
  * The original failure's directory is kept unchanged under `failures/`,
    and the reason is documented.
  * A second failure is reported, and that run stays missing. Its pair is
    reported without a partner.
* **Nothing else is rerun,** and no completed run is replaced.

## 8. Outcomes

Each is reported per run, with its denominator, and by model, scenario and
condition. Nothing is combined into a score.

| # | Outcome | Definition | Denominator |
|---|---|---|---|
| 1 | Initial proposals | decision points whose first parsed proposal has an action; of them, with a staffing action (eligible for the preview) | decision points |
| 2 | Successful preview requests | points where the preview was shown (`ok`); also, of them, previews in which the simulator refused an action | eligible points |
| 3 | Preview failures | `failed` and `too_large`, separately | eligible points |
| 4 | Proposals kept | response `kept` | previews shown |
| 5 | Proposals revised | response `revised`; also by part (staffing kept, revised or dropped; operations) | previews shown |
| 6 | Proposals abandoned | response `abandoned`; `no_reply` reported on its own | previews shown |
| 7 | Final submitted actions | actions in the submitted batches, staffing and operations | decision points |
| 8 | Accepted actions | actions in submitted batches that the engine accepted at apply | actions submitted |
| 9 | Refused actions | actions refused by validate in any attempt, by code (funding refusals apart); and actions refused at apply | actions proposed |
| 10 | Funding commitments | `dreamteam.financial-evidence/v1` AFFORDABILITY: statuses, and the engine's commitments of accepted batches | submitted batches |
| 11 | Credit-supported commitments | `funded_with_credit` | submitted batches |
| 12 | Reversibility categories | REVERSIBILITY statuses | executed actions |
| 13 | Direct operational exposure | OPERATIONAL_EXPOSURE direct changes, by action type | executed actions |
| 14 | Displaced appointments | bookings the engine stated were displaced by executed batches; and the run's `displacedAppointments` | runs |
| 15 | Financial distress | shortfall after the credit line, days below zero; credit drawn (most, days, interest) | runs |
| 16 | Profit difference | operating profit: preview-enabled minus baseline on the same model, entry and seed; and each against no action | pairs; runs |
| 17 | Completed examinations | exams done, the same differences | pairs; runs |
| 18 | Lost orders | orders lost, the same differences | pairs; runs |
| 19 | Service breaches | TAT breaches, all priorities, the same differences | pairs; runs |
| 20 | Turnaround time | TAT p90 (minutes), the same differences | pairs; runs |

**Also reported:** the model requests and review messages added by the
preview step (count and characters), the attempts per decision point, and
recording completeness.

## 9. Analyses

All are versioned and kept apart. The paired comparison is within the
same model, scenario, seed, horizon and economics. No action is an extra
reference, never a substitute for the paired comparison.

* **A. Changes in proposed decisions:** the first parsed proposals at
  matched decision points (same model, entry, seed and point index) in the
  two conditions. Until the conditions' decisions differ, the first
  message is the same function of the same observation; after that, the
  practices differ and later points are not matched as identical
  situations.
* **B. Changes after the preview:** outcomes 2–6, from the v4 records.
* **C. Financial commitments:** `dreamteam.financial-evidence/v1`, run
  unchanged on both conditions' recordings (v4 records are read through
  their v3 view: the same attempt fields), outcomes 10–11.
* **D. Reversibility:** outcome 12.
* **E. Direct operational exposure:** outcomes 13–14.
* **F. Subsequent operational outcomes:** outcomes 17–20, by run and pair;
  the V0.9b timeline of observed state.
* **G. Financial outcomes:** outcomes 15–16.

**Rules for reading them:**
* Direct effects (A–E) are kept apart from subsequent run outcomes (F–G).
* No later outcome is attributed to one decision.
* No change is labelled an improvement because it is a change.
* Eight pairs on two seeds are a development set: no generalization
  beyond them is claimed.
* A reason is quoted as the agent recorded it; no internal reasoning is
  inferred.
* The independent review stays PENDING unless an independent reviewer
  completes it.

The analysis code implements these definitions. It may be completed after
the runs, but no definition here changes.

## Amendments

None. This plan is not amended after any live result is seen.
