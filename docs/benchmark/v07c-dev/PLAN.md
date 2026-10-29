# V0.7c plan: frozen agents under overtime and solvency economics

Plan version `v07c-plan/1`, written 2026-10-29. It is committed before any
run of either stage, and it is not amended after any result is seen.

**Purpose.** V0.7a pays radiologists for overtime, and V0.7b adds a credit
line and a funding rule. This evaluation measures how the two frozen LLM
agents of V0.6c and the three built-in policies behave in that
environment, against no action on the same practice and seed. It is the
last development evaluation before V0.7.0.

**What it is not:**
* **Not a ranking:** no composite score, no leaderboard, no significance
  test, no claim beyond these seeds and models, and no clinical claim.
* **Not a comparison with V0.6:** the new economics change both what the
  agents observe and what their actions cost, and the funding rule can
  refuse actions. Every comparison here is within V0.7, from identical
  starting states. V0.6c's results are context only.
* **Not a new feature:** no simulator mechanic, action, scenario, policy,
  prompt or setting is changed.

## 1. What is fixed

| | |
|---|---|
| Suite | `dreamteam-dev` version 3 ([suite file](../suite/dreamteam-dev-3.json)), the five `-credit` entries: financial pressure, reporting backlog, demand growth, staffing shortage, stable |
| Economics | Every entry is the V0.7a `-ot` scenario (radiologist overtime paid at salary rate) with V0.7b's `$50,000` credit line at 10% a year and the funding rule ([V0.7_SOLVENCY_DESIGN.md](../../V0.7_SOLVENCY_DESIGN.md)) |
| Why `-credit` | The owner approved five scenarios with overtime costing and solvency enabled. The `-credit` entries carry the credit facility V0.7b designed. The `-nocredit` entries are the zero-credit edge; running them too would double the runs approved (75). Chosen before any result. |
| Scenario fingerprints | financial pressure `586f3ef74742af8c…`, reporting backlog `1fc90da25e025f8c…`, demand growth `28c16bf37eb6b67e…`, staffing shortage `5b49b4ca7b7304b3…`, stable `cfae9ca57ac46589…`; checked by the runner before anything runs |
| T90 | not applicable (version 1's sustainable scenario only) |
| Contract | `dreamteam.benchmark-contract/v1`, unchanged |
| Engine | 0.7.0-dev.3 (this commit; no engine behaviour changed since 0.7.0-dev.2) |
| Cadence | daily decisions at 07:00 |
| Built-in policies | `noop` 1, `greedy` 1, `conservative-v2` 2, unchanged since V0.6c |
| LLM agent | `agents/llm/llm_agent.py`, llm-agent 2, prompt `llm-agent-prompt/v2`, memory `llm-agent-memory/v1` (on); unchanged since V0.5c (8c62526; SHA-256 `5214e7bf…`) and the agent of V0.6c |
| Models | `Qwen3.6-35B-A3B-oQ4e-mtp` (policy `qwen36-35b-a3b`) and `gemma-4-26B-A4B-it-oQ4e-mtp` (policy `gemma4-26b-a4b`), V0.6c's two |
| Model settings | The V0.5c files, unchanged and used in V0.6c: [llm-qwen36-35b-a3b.json](../v05c-dev/llm-qwen36-35b-a3b.json), [llm-gemma4-26b-a4b.json](../v05c-dev/llm-gemma4-26b-a4b.json). Identical except for the model: temperature 0, `maxOutputTokens` 4,096, one retry, 300 s per request, 900 s per decision, `json_object`, summary observation, guided catalog, details allowed |
| Memory | on: 6 decisions, 6 metric rows, 6,000 characters |
| Server | the same local OpenAI-compatible server as V0.5c and V0.6c, reached through `DREAMTEAM_LLM_BASE_URL`, never committed; no key and no prices, so no inference cost is estimated (reported as "none configured") |
| Harness timeout | 1,800 s per decision point for the LLM agents |

**What the frozen agents see of the new economics.** Nothing in the agent
was changed for V0.7. Its summary observation keeps, from what the engine
already reports:
* the practice's name, which states the economics (for example
  "radiologist overtime paid, $50,000 credit line at 10%");
* the overtime minutes and overtime cost of the day;
* the financing block: credit limit, rate, amount drawn, amount available,
  current liabilities, reserved commitments, the funding headroom and the
  interest charged;
* each period's labour, expenses and profit, which include overtime and
  interest;
* the engine's refusal messages, with their codes. An `insufficient_funds`
  refusal states the commitment and the headroom. The agent's memory keeps
  the refusals it met in the run.

This is reported so that the results can be read. It is not a claim that
an agent used any of it.

## 2. Pipeline check (before this plan was committed)

[pipeline-check.json](pipeline-check.json): the five policies on the
financial-pressure entry, seed 41600, 3 days. It checks that both models
are served and answer under 0.7.0-dev.3, and that the runner, the
recorded-decision replay and [analyze.py](analyze.py) work. **It is not a
result:** it is not analysed or reported beyond whether it worked, and
nothing in this plan was chosen from its outcomes.

## 3. The evaluation

| Stage | Configuration | Seeds | Days | Policies | Runs | Limit |
|---|---|---|---|---|---|---|
| A | [stage-a.json](stage-a.json) | 41601, 41602 | 30 | noop, greedy, conservative-v2, qwen36-35b-a3b, gemma4-26b-a4b | 50 (20 LLM) | 6 h |
| B | [stage-b.json](stage-b.json) | 41701 | 90 | the same | 25 (10 LLM) | 10 h |

**Seeds.** Fresh: the next block after V0.7b's 41501–41503. None appears
elsewhere in the repository. They are not reference seeds (1001–1020),
V0.2 development seeds (901–910), reserved-stream seeds or holdout seeds.
No benchmark held-out seed is created and no held-out test is run.

**Order.** One stage at a time, on a Raspberry Pi 4 (linux/arm64): stage A,
then stage B.
* **Stage B does not depend on stage A.** Its seed, horizon and policies
  are fixed here. It runs after stage A whatever stage A shows.
* **Only an outage postpones it:** if the model server is unavailable,
  stage B waits until it is back. It is never cancelled, shortened or
  changed.

**Failures.** A model that cannot be loaded or cannot complete the
protocol is recorded and classified. It is never replaced by another
model.
* **Inside a run:** a request that fails is retried once by the agent (the
  frozen setting). A decision that still fails is a fallback (no action),
  counted and reported. The run continues.
* **A run that fails** (status not `completed`) is reported with its
  status and evidence.
  * If the cause is infrastructure (the server, the connection or the
    machine, not the agent or the simulator), it is rerun once, in full,
    with its own configuration (the same entry, seed and policy) into a new
    directory. Both are kept, and the reason is documented.
  * A second failure is reported, and that run stays missing.
* **At a limit,** the command is stopped. The completed runs are
  reported, and the stage is reported as partial. No horizon is
  shortened.

## 4. Replay and integrity

* **Suite checks.** For every stage, `benchmark verify`:
  * the scenario fingerprints;
  * the expansion;
  * the contract report;
  * every run's session, result and digests.
* **Starting states.** Equal across the five policies of each entry and
  seed, from the contract report.
* **LLM replay.** Every completed LLM run is replayed from its recorded
  decisions, without any model. The memory and prompt are rebuilt and
  checked against their recorded SHA-256 at every decision point.
  * **One configuration per stage and entry:** demand growth and reporting
    backlog share their starting state, and the agent's replay refuses
    recordings it cannot tell apart.
  * **Matching:** the replay's decisions, sessions, results (the financial
    ledger, credit draws, repayments and interest included), memory and
    final digests must equal the live runs'.
  * **Repeatability:** each replay is reproduced (`benchmark reproduce`) on
    linux/arm64 and on linux/amd64, with byte-identical manifests.
* **CI** replays the recordings on three platforms (linux/amd64,
  linux/arm64, darwin/arm64) and compares the manifests.
* **Live and replay are kept apart.** Live inference (latency, tokens,
  retries) is kept in each run's policy evidence and reported separately.
  The model exchanges stay outside the repository. The replay is
  deterministic; live inference is not.

## 5. What is reported

[analyze.py](analyze.py) (committed with this plan) writes each stage's
tables. Every comparison is paired: policy minus `noop` on the same entry
and seed, from the contract report.

**1. Outcomes, absolute and paired with no action:**
* operating profit, net revenue, expenses;
* cash at the end, the lowest cash, the most drawn on the credit line,
  interest, and any unfunded shortfall (financial distress);
* overtime cost, and the part approved in advance (`extend_shift`);
* TAT p90 and breaches;
* exams completed and lost orders.

**2. Actions, each counted at every stage:**
* **proposed:** every action put forward, including those refused by
  validate before submission;
* **refused by validate**, then **submitted**, **accepted**, and **refused
  at apply**;
* **executed:** accepted and in effect by the end of the window. A
  proposal is never counted as a change made.
* Executed actions by category (operations or staffing) and type.
* Refusals by code, `insufficient_funds` separately.

**3. Refusal responses:**
* every action refused `insufficient_funds`;
* how often the same type and target was put forward again later, refused
  again, or accepted;
* the agent's recorded reason at that decision point.

**4. LLM decisions:**
* points with and without actions, fallbacks;
* first-attempt validity and repeated invalid proposals;
* details requests;
* agent and infrastructure errors.

**5. Inference:** requests, retries, prompt and completion tokens, latency
(mean, maximum, total); cost is "none configured".

**6. The financial-reasoning questions,** answered from the figures and the
agents' recorded actions and reasons:
1. Do agents keep extending shifts after the overtime cost is visible?
2. Do they try to spend beyond the funding headroom?
3. Do they repeat actions refused for insufficient funds?
4. Do they reduce spending when cash or headroom falls?
5. Are service improvements accompanied by their financial cost?
6. Do they avoid unnecessary intervention in the stable practice?

**Rules for answering them:**
* **No inferred reasoning.** Intent is described only from the reasons an
  agent recorded, never from outcomes alone.
* **No verdict.** No action is not assumed to be correct, and service and
  financial outcomes are reported together. "Appropriate" is left to the
  owner.

## Amendments

None. This plan is not amended after any result of stage A or stage B is
seen.
