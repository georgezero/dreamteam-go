# V0.8c plan: prospective funding-refusal evaluation

Plan version `v08c-plan/1`, written 2026-10-30. It is committed before any
live run, and it is not amended after any result is seen.

**Purpose.** V0.7c's agents never met a funding refusal. This evaluation
puts the same two frozen models, through a recording that keeps every
attempt (llm-agent 3), in a practice where funding binds more often. It
also keeps a control condition, so that behaviour after a refusal can be
told apart from ordinary management behaviour. It measures what the
agents do after a refusal. It does not judge whether they decide well.

**What it is not:**
* **Not a ranking:** no composite score, no leaderboard, no significance
  test, no claim beyond these seeds and models, and no clinical claim.
* **Not tuned:** no prompt, model setting, memory policy, decision
  cadence, action schema or simulator mechanic is changed. The
  low-liquidity condition is made of existing settings (§5). It was chosen
  by a deterministic rule before any live run, and it is never changed
  after one.
* **Not required to produce refusals:** if no model meets a funding
  refusal, that is the finding. Cash is not lowered further to make one
  appear.
* **Not a test of understanding:** a changed proposal after a refusal is
  described as a change, never as adaptation, understanding or success.

## 1. What is fixed

| | |
|---|---|
| Suite | `dreamteam-dev` version 4 ([suite file](../suite/dreamteam-dev-4.json)): version 3's ten entries unchanged, and `financial-pressure-lowliquidity` |
| Control condition | `financial-pressure-credit` (scenario `bench-financial-pressure-ot-credit`, fingerprint `586f3ef74742af8c537e7ff41c41d586fafbfa75c4426043122595ba7c8795b1`): V0.7c's financial-pressure entry, unchanged. Radiologist overtime paid; a $50,000 credit line at 10% a year; the funding rule; $150,000 opening cash |
| Low-liquidity condition | `financial-pressure-lowliquidity` (scenario `bench-financial-pressure-ot-lowliquidity`, fingerprint `3175656ee0248a369e3884d539086c2c1fd3b37b306150c2ae33a064d94dc781`): the control with $80,000 opening cash and nothing else changed (§5) |
| Fingerprints | checked by the runner before anything runs |
| T90 | not applicable |
| Contract | `dreamteam.benchmark-contract/v1`, unchanged |
| Engine | 0.8.0-dev.3 (this commit; no engine behaviour changed since 0.8.0-dev.2) |
| Cadence | daily decisions at 07:00 |
| Horizon | 30 days |
| Deterministic comparators | `noop` 1, `greedy` 1, `conservative-v2` 2, unchanged since V0.6c; no model |
| LLM agent | `agents/llm/llm_agent3.py`, llm-agent 3 (SHA-256 `ef2c17a06749d7c287cda918f77ff0e518d79475048e87fd3404848c2fb0015c`): llm-agent 2's system prompt, prompt `llm-agent-prompt/v2`, memory `llm-agent-memory/v1` (on), settings, retries, details turn, batching and fallback, unchanged; only the record differs (§3) |
| Models | `Qwen3.6-35B-A3B-oQ4e-mtp` (policy `qwen36-35b-a3b`) and `gemma-4-26B-A4B-it-oQ4e-mtp` (policy `gemma4-26b-a4b`), V0.6c's and V0.7c's two |
| Model settings | The V0.5c files, unchanged and used in V0.6c and V0.7c: [llm-qwen36-35b-a3b.json](../v05c-dev/llm-qwen36-35b-a3b.json) (`e19a47dc…`), [llm-gemma4-26b-a4b.json](../v05c-dev/llm-gemma4-26b-a4b.json) (`1a1edae5…`). Identical except for the model: temperature 0, `maxOutputTokens` 4,096, one retry, 300 s per request, 900 s per decision, `json_object`, summary observation, guided catalog, details allowed |
| Memory | on: 6 decisions, 6 metric rows, 6,000 characters |
| Server | the same local OpenAI-compatible server as V0.5c–V0.7c, reached through `DREAMTEAM_LLM_BASE_URL`, never committed; no key and no prices |
| Harness timeout | 1,800 s per decision point for the LLM agents |
| Configuration | [eval.json](eval.json) |

## 2. Conditions and runs

| Entry | Seeds | Days | Policies | Runs |
|---|---|---|---|---|
| financial-pressure-credit (control) | 41901, 41902 | 30 | noop, greedy, conservative-v2, qwen36-35b-a3b, gemma4-26b-a4b | 10 (4 LLM) |
| financial-pressure-lowliquidity | 41901, 41902 | 30 | the same | 10 (4 LLM) |

**In all:** 20 runs, of which 8 are live LLM runs, the most the owner
approved. There is no live pipeline check before the evaluation. The
agent's recording and replay were checked against a mock model server
(`internal/bench/llm_agent3_test.go`), and the configuration was expanded
with the three built-in policies for 2 days (not analysed, and not kept).

**Seeds.** Fresh: the next block after the V0.8b probes' 41801.
* 41900 is the scenario analysis seed (§5): no policy and no model ran on
  it.
* 41901 and 41902 are the evaluation seeds. No run of any kind used them
  before this plan, except the 2-day expansion check above.
* None appears elsewhere in the repository. They are not reference seeds
  (1001–1020), V0.2 development seeds (901–910), reserved-stream seeds or
  holdout seeds. No benchmark held-out seed is created and no held-out
  test is run.

## 3. The recording: `dreamteam.llm-decision-record/v3`

Specified in [LLM_AGENT.md §13](../../LLM_AGENT.md), as designed in V0.8b
([RECORDING_CONTRACT.md](../v08b-dev/RECORDING_CONTRACT.md)). At each
decision point it keeps:
* every attempt, rejected ones included, in order, each with a stable
  identifier (`point.attempt`);
* each attempt's input (the prompt, a details answer, a retry with its
  feedback, or a resend) by SHA-256;
* the model output, the parse result, the reason and actions, every
  validate answer, the refusals and the error;
* which attempt's decision was submitted (`final`), the batches, and how
  complete the record is (`completeness`);
* the prompt and memory SHA-256.

**What it keeps apart:**
* *Model output:* each attempt's text, bounded.
* *Parsed actions:* its `actions`.
* *Validated actions:* each `validation` entry, with the engine's answer.
* *Submitted actions:* `batches`.
* *Applied actions:* the harness's `decisions.json`. The agent does not
  see the apply answer.

Attempts are also appended to `attempts.jsonl` as they are made.

## 4. Replay requirements

* **Every completed LLM run is replayed** without any model, by llm-agent
  3's replay. At every decision point it checks:
  * the digests;
  * the memory and prompt, rebuilt;
  * every attempt's input, rebuilt from the observation and the record;
  * every validate answer, asked again of the engine and compared;
  * that the submitted batches are the used attempt's actions.
* **Matching:** the replay's decisions, sessions and results must equal the
  live runs'. That includes the financial ledger, credit draws, repayments
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
  * The model exchanges and the agent's settings record (with the
    server's address) stay outside the repository.

## 5. The low-liquidity condition

**The rule,** fixed before it was applied
([scenario_analysis.py](scenario_analysis.py)):

> The variant's opening cash is the control's opening cash, less the
> control's lowest no-action funding headroom at the 30 daily decision
> points (07:00) on the analysis seed 41900, rounded down to a whole
> $10,000.

**Why this rule.** Under no action the control's funding headroom never
falls below about $76,000 in 30 days. With the credit line, no discretionary
commitment the V0.8b probes measured on this practice comes near it. The
rule removes that cushion, rounded so that headroom stays above zero
under no action. Only opening cash changes:
* the credit line, its rate, the funding rule and the practice are the
  control's;
* the analysis uses no policy, no model and no evaluation seed.

**What it gives** ([scenario-analysis.json](scenario-analysis.json)):

| | Control | Low liquidity |
|---|---|---|
| Opening cash | $150,000 | $80,000 |
| No-action funding headroom, 30 points | $76,093 to $105,357 | $6,080 to $35,357 |
| Credit drawn under no action | $0 | up to $21,726 (at the month's disbursements, repaid within days) |
| Shortfall (distress) under no action | none | none |

**Points (of 30) where a commitment would not fit under no action:**

| Commitment | Control | Low liquidity |
|---|---|---|
| A technologist hire ($10,457) | 0 | 2 |
| A 40-point allocation raise ($19,726) | 0 | 11 |

**How to read those figures:**
* *Under no action only.* An agent's own commitments reserve headroom
  and lower it further.
* *Not a promise of refusals.* Whether any agent meets one is what the
  evaluation observes.

The condition is final. It is not changed after any live result.

## 6. Order, resources and stop conditions

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
  * The original failure's directory is kept unchanged under
    `failures/`, and the reason is documented.
  * A second failure is reported, and that run stays missing.
* **Nothing else is rerun,** and no completed run is replaced.

## 7. Primary outcomes

Each is reported per run, with its denominator, and summed by model and
condition. Nothing is combined into a score.

| # | Outcome | Definition | Denominator |
|---|---|---|---|
| 1 | Funding refusals encountered | actions refused `insufficient_funds` by validate, in any attempt; and actions refused at apply, by code | proposed actions; decision points |
| 2 | Attempts per decision | attempts recorded at each decision point (distribution, mean) | decision points |
| 3 | Identical refused proposals | refusal-response `identical_repeated` | refusals |
| 4 | Modified proposals | refusal-response `modified_proposal` | refusals |
| 5 | Different proposals | refusal-response `different_proposal` | refusals |
| 6 | No action after refusal | refusal-response `no_action` | refusals |
| 7 | Actions ultimately accepted | actions in submitted batches that the engine accepted at apply | actions submitted |
| 8 | Actions ultimately executed | accepted with a change (an action accepted with nothing to change is not counted) | actions accepted |
| 9 | Financial distress | shortfall after the credit line is exhausted, days and amount | runs |
| 10 | Credit utilization | the most drawn, days with a balance drawn, interest | runs |
| 11 | Profit | operating profit, and its difference from no action on the same entry and seed | runs |
| 12 | Service breaches | TAT breaches, all priorities | runs |
| 13 | Completed examinations | exams done | runs |
| 14 | Lost orders | orders lost | runs |
| 15 | Turnaround time | TAT p90 (minutes) | runs |

**Response categories.** Every refusal is put in exactly one category,
each counted separately:

| Category | Meaning |
|---|---|
| refused and repeated | `identical_repeated` |
| refused and revised | `modified_proposal` or `different_proposal`, each also reported on its own |
| refused, then no action | `no_action` |
| no longer applicable | `refusal_no_longer_applicable`, reported on its own |
| evidence unavailable | `evidence_unavailable` |

Every run that met no refusal is counted as "no refusal encountered". The
categories are those of `dreamteam.refusal-response/v1`
([DECISION_EVIDENCE.md §12](../../DECISION_EVIDENCE.md)). Funding refusals
are reported apart from refusals with other codes.

## 8. Analyses

All are versioned and kept apart:
* **Frozen `dreamteam.decision-evidence/v1`:** the frozen rules, run
  unchanged on the llm-agent 2 view of each v3 record (`final` and the
  decision's summary fields). They are not re-tuned and not replaced.
* **Repeat rule version 2:** a separately versioned rule that recognizes
  repeated refused proposals by action content, not by refusal wording.
  It abstains explicitly when an earlier attempt's actions are not
  recorded. Its differences from v1's `REPEATED_INVALID_PROPOSAL` are
  reported, and v1's findings are not replaced.
* **`dreamteam.refusal-response/v1`:** run on the v3 attempts.
* **Financial and operational outcomes:** paired with no action, the V0.8b
  analysis's separation kept. Stated reasons, submitted, accepted and
  executed actions are reported separately. Financial and operational
  outcomes are kept apart from the consistency findings.
* **Recording completeness:** the share of decision points with every
  attempt recorded, and with every input verifiable.

**Rules for reading them:**
* No model ranking or composite score.
* No action is labelled irrational from its outcome.
* No internal reasoning is inferred from behaviour. A reason is quoted as
  the agent recorded it.
* The independent review of the v1 findings stays PENDING unless an
  independent reviewer completes it. V0.8c makes no claim that the
  findings were independently validated.

The analysis code implements these definitions. It may be completed after
the runs, but no definition here changes.

## Amendments

None. This plan is not amended after any live result is seen.
