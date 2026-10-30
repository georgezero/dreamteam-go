# DreamTeam — Release notes

Newest first. Step-by-step detail is in `docs/PROGRESS.md`; the acceptance
evidence in `docs/ACCEPTANCE.md`.

---

## V0.7.0 — financial realism for practice-management simulation

V0.7 makes the financial consequences of management decisions more
faithful. Radiologist overtime now costs money, and a practice can run
short of the means to finance new commitments. Both are **opt-in
scenario settings**: every earlier scenario, save, golden reference,
fingerprint and V0.6 benchmark result is unchanged, and suite
`dreamteam-dev` version 1 (the V0.6 benchmark) is still available as it
was. The new economics are deliberately simple. They are **not a
validated model of radiology practice finance** (design notes
`docs/V0.7_OVERTIME_DESIGN.md`, `docs/V0.7_SOLVENCY_DESIGN.md`).

**Overtime (V0.7a):**
1. **Radiologist overtime accounting.** Under the scenario policy
   `radiologistOvertime: "salary_rate"`, an on-site salaried
   radiologist's time on duty past the rostered end is paid. Omitted or
   `"unpaid"` means the earlier behaviour.
2. **Actual worked overtime.** The charge is for time actually on duty
   past the rostered end, approved extensions and hold-overs alike. An
   extension that is never worked (for example after a roster change)
   costs nothing; idle presence inside one is paid.
3. **Salary-derived cost.** The annual salary over 52 weeks of the
   scenario's `weeklyOvertimeHours` (2,080 hours for the bundled
   radiologists: $230.77 an hour), straight time, integer cents, floored
   once per radiologist-day.
4. **Benefits and payroll.** It accrues through the existing payroll with
   the existing 25% on-cost and is paid at the next payroll. It appears in
   `overtimeCostCents`, `approvedOvertimeCostCents`, wages and operating
   profit. An `extend_shift` acceptance message carries the estimate.

**Solvency (V0.7b):**
5. **Opt-in financial solvency.** The scenario setting
   `finance.solvency` (`creditLimitCents` 0 to $1,000,000,
   `creditRateBP` 0 to 5,000) turns it on. Without it nothing below
   exists or runs.
6. **Cash and credit constraints.** Funding headroom is cash plus undrawn
   credit, less current liabilities (accrued payroll, payables, severance
   payable) and reserved commitments. Receivables and profit do not
   count.
7. **Credit interest and repayment.** A disbursement that would take cash
   below zero draws on the line, up to its limit. Simple interest accrues
   daily on the drawn balance (rounded half up) to payables, as its own
   expense line. Cash above zero repays the line at midnight.
8. **Financial distress accounting.** Cash below zero with the line fully
   drawn is distress. The obligation is still paid (there is no missed
   payroll and no shutdown); the shortfall is logged and reported.
9. **Management-action affordability.** A hire, salary-raising allocation
   change, teleradiology contract or paid extension must fit within the
   funding headroom, or it is refused `insufficient_funds` (an ordinary
   action result; the rest of its batch is `batch_rejected`). Cuts,
   terminations and changes that commit nothing are never refused.
   Preview, validation and application share one check, and accepted
   commitments are reserved and released over their period.

**Benchmark and compatibility:**
10. **Historical compatibility.** Both settings are omitted from canonical
    JSON when unset, compiled into configuration outside the state digest,
    and encoded only when used. All golden references are identical, and
    saves from V0.1 to V0.6 and from every V0.7 development build replay
    with every digest verified.
11. **Versioned benchmark suites.** `dreamteam-dev` version 1 is the V0.6
    benchmark, unchanged. Version 2 is its five practices with paid
    radiologist overtime (`-ot`). Version 3 is those practices with a
    $50,000 credit line at 10% (`-ot-credit`) and with none
    (`-ot-nocredit`). Each entry is checked by fingerprint.
12. **Frozen-agent evaluation.** V0.7c ran V0.6c's two LLM configurations,
    unchanged, with the deterministic baselines, on version 3's credit
    practices (`docs/benchmark/v07c-dev/`), declared before it ran.
13. **Recorded-decision replay.** All 30 V0.7c LLM runs replay from their
    recordings without a model, byte for byte, including the ledger,
    credit draws, repayments and interest.
14. **Cross-platform reproducibility.** CI replays every committed
    recording (V0.5, V0.6 and V0.7) on linux/amd64, linux/arm64 and
    darwin/arm64 and compares the manifests byte for byte.

**Characterization of the new economics** (deterministic baselines;
`docs/benchmark/v07a-dev/`, `v07b-dev/`):
* With no action, radiologists are held over 39–69 hours in 90 days, and
  paid overtime lowers profit by $11,294–$19,788.
* Under solvency, `noop` and `conservative-v2` never meet the funding
  rule. `greedy` meets it in financial pressure with a $50,000 line (3
  refusals, one draw of $12,584) and in every practice with no credit (65
  refusals in 14 of 15 runs). One distress was recorded: $1,981, recovered
  the same day.
* The five practices generally have enough liquidity that solvency is not
  their dominant management problem. No scenario was made to force
  insolvency.

**Development findings** (V0.7c, `docs/benchmark/v07c-dev/README.md`).
These are observations from development seeds, not a ranking and not a
claim about either model in general.
* **Scope:** 75 runs, all completed with no failure or rerun:
  * five scenarios (financial pressure, reporting backlog, demand growth,
    staffing shortage, stable), version 3 with credit;
  * `noop`, `greedy`, `conservative-v2` and two frozen LLM configurations
    (Qwen3.6-35B-A3B and Gemma 4 26B-A4B, llm-agent 2, prompt
    `llm-agent-prompt/v2`, the V0.6c settings);
  * 30 days on seeds 41601–41602 (50 runs) and 90 days on seed 41701 (25
    runs);
  * every comparison paired with no action from the same starting state;
    financial and operational outcomes reported together.
* **Qwen intervened frequently,** in all 15 of its runs, mostly by
  extending shifts. Over 30 days it reduced profit by $3,197–$20,304
  against no action, and lowered the lowest cash in every run, while
  generally improving TAT and breaches.
* **Gemma intervened comparatively little.** Over 30 days it submitted
  actions in 4 of 10 runs, all teleradiology contracts. Over 90 days it
  made no change in financial pressure or stable, and extended shifts in
  demand growth (27) and staffing shortage (53).
* **`conservative-v2`'s financial guardrails** kept it from acting except
  in the stable practice: one raise not yet in effect at 30 days, and two
  raises over 90 days (profit −$2,712, breaches −12).
* **Every pairing that improved service lowered profit.** No pairing
  improved service and raised profit.
* **One development observation, staffing shortage, 90 days, Qwen.** It
  hired two technologists, then from 2026-11-27 gave notice to them and
  to every other technologist and the front desk. Its recorded reasons
  were "Cash is $0 and critically low; immediate cost reduction is
  required to avoid insolvency" and "This is a survival move". It drew up
  to $17,200 on the credit line ($14 of interest). Against no action:
  operating profit −$181,589, 867 fewer exams completed, 1,450 more
  orders lost. No recorded reason in that run mentions the credit line,
  although the agent's observation carries it. This is one run on one
  seed.
* **Repeated invalid proposals.** Every refusal came at the agents' own
  validation step, for timing or form. The same invalid action was
  proposed again up to 15 times in a run (Qwen) and 27 times (Gemma).
* **Replay.** Every LLM run replays without the model, byte-identical to
  the live run, and the replay manifests are identical on linux/arm64 and
  linux/amd64.

**Solvency: what V0.7 does not show.** No evaluated V0.7c policy action
was refused for insufficient funds, in any of the 75 runs. The agents'
responses to funding refusals therefore remain unobserved. Solvency
accounting and enforcement were tested separately, through deterministic
tests and the V0.7b baseline characterization. Nothing in this release
validates LLM behaviour under funding refusals.

**Known limitations:**
* **Development seeds only;** no held-out benchmark evaluation; no
  clinical validation. Everything is synthetic.
* **Two frozen LLM configurations,** with one prompt and one set of
  settings.
* **Live inference is nondeterministic.** Exact replay requires the
  recorded decisions.
* **External agent processes are trusted,** not sandboxed.
* **Simplified overtime economics:** straight time at the salary's hourly
  rate, the same on-cost as all pay, no premiums or call pay.
* **No manager overtime accounting.** The operations manager's overtime
  remains unpaid.
* **A simplified credit facility:** a fixed limit and simple daily
  interest; no covenants, fees, scoring or rate changes. The funding rule
  reserves only near-term committed costs.
* **No automatic practice shutdown,** and limited insolvency
  consequences: distress refuses new commitments and nothing else.
  Payroll is never missed.
* **Limited demand trend modelling:** monthly factors, no dated shocks.
* **No staff absence or turnover model.**
* **Repeated invalid LLM proposals,** refused by validation but recurring.
* **Stated reasons and submitted actions may not match.** A recorded
  reason is evidence of what an agent said, not of why it acted.
* **No inference pricing was configured.** Tokens and latency are
  reported instead, and latency depends on server load.

**Compatibility.**
* No change to mechanics, randomness, actions, accounting, evaluation or
  scoring for any existing scenario. The V0.7 behaviour exists only in
  the new scenario variants.
* All golden references give identical outcomes, and suite version 1's
  fingerprints are unchanged.
* Saves from V0.1 to V0.6 and from 0.7.0-dev.1 to 0.7.0-dev.3 replay with
  every digest verified. 0.7.0-dev.3 is registered as batch mode
  `v0.7_dev3`.
* The LLM agent, its prompt and settings, the policy and agent protocols,
  and the contract report are unchanged. Every committed recording
  replays.

**Validation.**
* The full Go test suite passes, and the race detector finds no race (with
  `-timeout 90m` for the slow packages).
* vet and gofmt are clean, and the Python agent tests pass.
* Every session fixture replays, and every golden reference is identical.
* Suite verify and reproduce pass for the V0.7 suites.
* Every recorded LLM decision, V0.5 to V0.7, replays byte for byte.
* Cross-builds succeed for five targets, and the committed seed guards
  pass.
* CI is green on all three platforms and in the cross-platform
  comparison.

---

## V0.6.0 — a reproducible benchmark for practice-management agents

DreamTeam is now a reproducible benchmark for radiology practice-management
agents. A suite of five practices is run in one command. Every policy is
reported on six dimensions against no action, and every decision is
recorded and can be replayed and verified on any platform
(`docs/BENCHMARK_SUITE.md`, `docs/BENCHMARK.md`). It is a development
benchmark on synthetic practices, **not a clinically validated one**. The
simulator, the agent and policy protocols, the LLM agent and its prompt,
and scoring are unchanged.

**What the benchmark provides:**
1. **Five benchmark scenarios** (suite `dreamteam-dev`, version 1), each a
   different management problem. They are financial pressure, reporting
   backlog (the `sustainable` scenario), demand growth, staffing shortage
   and a stable practice where intervention is costly.
2. **Versioned scenario fingerprints.** Each suite entry names its
   scenario and its fingerprint. The runner checks the fingerprint before
   anything runs, and confirms that every policy starts from the same
   state.
3. **Multi-scenario suite execution.** `dreamteam benchmark suite run`
   expands a suite-run file (`dreamteam.benchmark-suite-run/v1`) into
   ordinary harness runs: every entry × seed × policy.
4. **A six-dimensional reporting contract**
   (`dreamteam.benchmark-contract/v1`). The dimensions are operations,
   finance, staffing and capacity, service reliability, sustainability, and
   actions. Each run is reported on all six, using metrics the simulator
   already had. There is no composite score, weighting, ranking or
   leaderboard. Absent metrics are named, not estimated. T90 is reported
   only where its evaluation policy applies.
5. **Paired comparisons against no action.** Each policy minus `noop` on
   the same entry and seed.
6. **Deterministic policy baselines.** There are five:
   * `noop`, which never acts;
   * `greedy` and `rules`, which raise the reading allocation;
   * `conservative` v1 (V0.6b), the historical baseline;
   * `conservative-v2` (V0.6c). It acts only on a week of urgent or soon
     report breaches it can afford, reviews each raise, and steps back.
     It is a fixed, declared rule, not an optimized policy.
7. **External LLM agent evaluation.** The V0.5 LLM agent, unchanged
   (`llm-agent-prompt/v2`), takes part like any other policy.
8. **Recorded-decision replay.** An LLM run's recorded decisions are
   resubmitted without any model. The memory and prompt are rebuilt and
   checked at every decision point, and the result is the live run's,
   byte for byte.
9. **Cross-platform reproducibility.** CI runs the suite and replays every
   committed recording on linux/amd64, linux/arm64 and darwin/arm64. It
   then compares the manifests byte for byte.
10. **Development evaluation results** (`docs/benchmark/v06a-dev/`,
    `v06b-dev/`, `v06c-dev/`), each declared before it ran, and kept as
    run.

**Development findings** (V0.6c, `docs/benchmark/v06c-dev/`). These come
from a small number of seeds, scenarios and frozen configurations: five
scenarios, three seeds at 30 days and one at 90, and two local models
with one prompt and one set of settings. They are observations, not a
ranking. They make no claim of generality or statistical significance.
* **Qwen3.6-35B-A3B** frequently improved reporting metrics at a
  substantial financial cost. Over 90 days in staffing shortage, TAT p90
  fell from 761 to 131 minutes and breaches by 67, while operating profit
  fell by $137,507 against no action.
* **Gemma 4 26B-A4B** intervened less often. It made one accepted change
  in 450 decision points over 90 days.
* **`conservative-v2`** used its financial guardrails and often chose no
  action. Where it acted, it raised the reading allocation and, in the
  stable practice, stepped back at its review.
* **`greedy`** repeatedly raised the reading allocation, at $27,000–
  $34,000 of profit over 90 days in every scenario.
* **Replay.** Every evaluated LLM recording (40 runs) reproduced without
  model access.
* **T90.** Every applicable T90 evaluation (reporting backlog) returned
  NO GO.
* **Explanations.** The models' stated reasons sometimes contradicted
  their submitted actions, or misread the observation (cents as
  dollars).

**V0.6a disclosure, unchanged.** The demand-growth scenario's declared
within-run check (raw 30-day demand totals rising) failed on two of
three characterization seeds: weekday counts confound the totals. The
result stands as recorded. V0.6b then declared a demand diagnostic
(`dreamteam.demand-diagnostic/v1`) prospectively, and on fresh seeds its
checks held.

**Known limitations.** These matter for any use of the results:
* **Limited development seeds.** Three seeds at 30 days and one at 90;
  no statistical claim.
* **Two frozen LLM configurations,** with **one agent prompt version**
  (`llm-agent-prompt/v2`).
* **No clinical validation.** Everything is synthetic. The scenarios do
  not reproduce all the economics of real radiology practices.
* **No held-out benchmark execution.** The held-out governance is
  defined, but no held-out set exists and none was run.
* **Live inference is nondeterministic,** even at temperature 0.
  **Exact replay requires the recordings.**
* **External agents are trusted, not OS-sandboxed.**
* **Radiologist shift extensions have no incremental salary cost.**
  Radiologists are salaried in the simulator, so extending their shifts
  costs nothing. This can materially distort the apparent financial
  consequences of management decisions. It is a limitation of the
  simulator's model, not fixed in this release.
* **Financial insolvency has no simulator consequence.** Negative cash
  stops nothing.
* **Demand growth uses monthly factors,** not an explicit dated shock.
* **Some important metrics are absent:** imaging capacity utilization and
  trends within a run.
* **Model explanations may contradict submitted actions.** A stated
  reason is not evidence of what an agent did.
* **LLMs may repeatedly propose invalid actions.** `validate` refuses
  them, but they recur, up to 18 times in one run.
* **Inference costs were not priced.** Tokens and latency are reported
  instead.
* **Cross-stage latency comparisons are affected by server load.**
* **`conservative-v2` has been neither optimized nor validated.**
* **Intervention rate is not a verdict.** A model's low intervention rate
  is neither inherently good nor bad.

**Compatibility.**
* No change to mechanics, randomness, scenarios, actions, accounting,
  evaluation, scoring, the protocols, the LLM agent or its prompt.
* All golden references give identical outcomes.
* Saves from V0.1 to V0.5 and from every V0.6 development build replay
  with every digest verified. 0.6.0-dev.3 is registered as batch mode
  `v0.6_dev3`.
* Every committed recording, V0.5 and V0.6, still replays.

**Validation.**
* The full Go test suite passes, and the race detector finds no race (with
  `-timeout 90m` for the slow packages).
* vet and gofmt are clean, and the Python agent tests pass.
* Every session fixture replays, and every golden reference is identical.
* Suite verify and reproduce pass.
* Every recorded LLM decision replays byte for byte.
* Cross-builds succeed for five targets, and the committed seed guards
  pass.
* CI is green on all three platforms and in the cross-platform
  comparison.

---

## V0.5.0 — LLM agents

A language model can now manage the practice. An external agent asks any
OpenAI-compatible model server what to do at each decision point. It takes
part through the V0.4 benchmark harness like any other policy, and its
decisions are recorded and replay exactly without the model
(`docs/LLM_AGENT.md`). The simulator, the protocol and scoring are
unchanged.

**The agent** (`agents/llm/llm_agent.py`, llm-agent 2; standard-library
Python, one process per run):
1. **External, OpenAI-compatible.** It calls `chat/completions` on any
   compatible server, local or remote, with no provider-specific
   dependency. The address comes from the environment, and an API key only
   from a named environment variable. A key is never written to the
   settings or the evidence.
2. **Structured decisions, validated.** Each reply is a strict
   `{"reason", "actions"}` object. Every action is checked with the
   protocol's `validate` before anything is submitted.
3. **Safe retries and a no-action fallback.** A reply that does not parse,
   or that `validate` refuses, is retried a bounded number of times with
   the refusal shown. If no valid decision results, the agent submits
   nothing; a failure never changes the practice.
4. **Bounded memory** (`llm-agent-memory/v1`). Within a run the agent
   remembers its recent decisions, the refusals it met, what was applied
   and recent metrics, within a fixed size. It is built from public
   observations and its own records only, and empty at the start of every
   run. `memoryDecisions` 0 turns it off.
5. **Compact observations and prompts** (`llm-agent-prompt/v2`). A
   summarized observation that names what it leaves out, with a request
   for details once per decision point. 56% fewer tokens than V0.5a's
   prompt on the same observations.
6. **Action constraints and guidance.** For every action type: its fields,
   rules, earliest dates, lead times and targets, beside the operating
   hours and scheduled close.
7. **Several batches per decision.** Operations and staffing actions in
   one reply are submitted as two validated batches at one decision point.
8. **Usage and cost accounting.** Per run: requests, retries, tokens
   (reported, estimated or unknown), latency and inference time, and an
   estimated cost from configured prices, labelled as an estimate.
9. **Recorded-decision replay.** `--replay` resubmits a recording through
   the protocol without any model. It rebuilds the memory and prompt at
   every decision point and checks their SHA-256, giving the live run's
   decisions, session, result and digests byte for byte. Recordings of
   other agent versions are refused explicitly.

**10. Multi-model development evaluation** (`docs/benchmark/v05c-dev/`).
* **Setup:** three local models (Qwen3.6-35B-A3B, Gemma 4 26B-A4B,
  Gemma 4 31B) through the same frozen agent and settings, on seeds
  declared before any run.
* **Stages:** 7 days on one seed; 30 days on three; a memory-off control
  on those three; 90 days on one.
* **Completion:** all 26 runs completed with no infrastructure failure.
  All 18 model recordings replay byte for byte, and CI replays them on
  three platforms.
* **Findings** (development observations, not claims):
  * the model that acted most improved turnaround and breaches against no
    action, at a lower operating profit: over 90 days TAT p90 fell from
    980 to 121 minutes and breaches by 295, and profit was $34,565 lower;
  * the other two models mostly chose no action;
  * with the memory off, the same model had 85 validation refusals and 55
    repeated invalid proposals over three seeds, against 20 and 8 with it
    on.

**Evidence.** The harness keeps an external policy's own evidence in each
run's `policy-evidence/`, hashed in the manifest as nondeterministic: the
agent's decisions, the memory at each point, the model exchanges and its
usage summary. No endpoint and no credential appear.

**Compatibility.**
* No change to mechanics, randomness, scenarios, actions, accounting,
  evaluation, scoring or the agent and policy protocols.
* All golden references give identical outcomes.
* Saves from V0.1 to V0.4 and from the V0.5 development builds replay with
  every digest verified.
* The V0.5a recording (llm-agent 1) still replays.

**Known limitations** (`docs/LLM_AGENT.md` §12):
* **Live inference is not deterministic.** Even at temperature 0, a rerun
  may decide differently. Exact replay requires the recording.
* **Trusted processes.** External agents are trusted participants, not
  OS-sandboxed.
* **Not jointly atomic.** An operations batch and a staffing batch at one
  decision point are two batches, each validated and applied on its own.
* **Memory informs; it does not enforce.** Models may repeatedly propose
  invalid actions despite the refusals in their memory; `validate` catches
  them.
* **Configuration-dependent.** Model behaviour depends heavily on the
  prompt, settings and server.
* **No action.** The tested models frequently chose no action.
* **Cost of service.** Service improvements sometimes substantially
  reduced operating profit.
* **Development results only.** They establish neither model superiority
  nor generality: one scenario, five seeds, three models, each live run
  once.
* **Not validated.** No model or agent is clinically validated or
  benchmark-certified.

**Validation.**
* The Go tests of the affected packages pass on linux/arm64.
* The race detector finds no race (linux/amd64).
* vet and gofmt are clean.
* The Python agent tests pass.
* Every session fixture replays, and every golden reference is identical.
* Benchmark verify and reproduce pass.
* Every recorded LLM decision replays byte for byte.
* Cross-builds succeed for five targets.
* CI is green on all three platforms and in the cross-platform
  comparison.

## V0.4.0 — the agent evaluation harness

Agent policies can now be compared reproducibly: built-in policies or
independently running programs play the same scenarios and seeds through
the agent protocol, and every decision and outcome is recorded, replayed
and verified (`docs/BENCHMARK.md`).

**`dreamteam benchmark`**
* `run CONFIG` plays every scenario × seed × policy of a configuration,
  with the same initial state, horizon and decision cadence for each.
  The harness alone applies batches, advances time, saves and evaluates
  T90; each run has its own session (and process, for an external
  policy).
* `report` compares the runs: per run, per policy, and paired with a
  baseline. The metrics are the simulator's own operational and
  financial measurements. There is no composite score and no ranking.
* `verify` checks the manifest's hashes, replays every saved session,
  rebuilds the results and the report, and with `-t90` recomputes T90
  (the existing v4 evaluation, v0.2-eval-4).
* `reproduce` reruns the recorded configuration and compares every
  deterministic file and the manifest byte for byte.

**Policies.**
* Built in: `noop` (the baseline), `greedy` and `rules`.
* **External policies** run as a fresh process per run, in an empty
  working directory, speaking the versioned
  `dreamteam.benchmark-policy/v1` protocol (JSON lines over stdio). Each
  request goes through the public-agent capability gate, which permits
  only reads and checks. The per-decision timeout is configurable, and
  standard error is captured separately.
* **Failures** are classified (`policy_crash`, `policy_timeout`,
  `policy_error`). The process is killed and reaped, and the other runs
  go on. Refused, denied and malformed requests are counted.
* **Seed withholding.** Under the harness, `hello` omits the seed and
  says `seedWithheld`; no permitted response contains it. Outside the
  harness, `hello` is unchanged.
* **Reference Python policy** `python-ref` v1
  (`clients/python/reference_policy.py`, standard library only).

**Recorded evidence.** Each run keeps its protocol transcript, decisions,
saved session, canonical result and T90. The manifest records the
engine, protocol, policies (with an external program's SHA-256), metric
definitions and the hash of every deterministic file.

**Development results** (`docs/benchmark/v04b-dev/`: sustainable
scenario, five seeds, 90 days, four policies, 20 runs). All runs
completed and replayed. The results are byte-identical on linux/arm64
and linux/amd64 and across Python versions. They are development
observations, not confirmatory performance claims:
* `python-ref` v1 never acted. Its weekly review missed the backlog
  threshold, so its runs equal `noop`'s. It is kept unchanged.
* `greedy` and `rules` improved reporting against `noop` (TAT p90 204–472
  minutes lower). They also cost about $33,000 more, and operating
  profit was $27,357–$32,719 lower. 90 days does not separate them.
* All 20 runs received T90 NO GO.

**Compatibility.**
* No change to mechanics, randomness, scenarios, actions, accounting or
  the evaluation.
* All golden references give identical outcomes.
* Saves from V0.1, V0.2, V0.3 and the V0.4 development builds replay
  with their digests verified.
* The agent protocol `dreamteam.agent/v1` is unchanged outside the
  harness. The V0.3 Python client's transcript is identical apart from
  the engine version.

**Known limitations** (`docs/BENCHMARK.md` §12).
* External policies are trusted development participants, not
  OS-sandboxed processes.
* Seed withholding prevents disclosure through the benchmark protocol,
  not arbitrary local inspection.
* Wall-clock timeouts are not deterministic; a run that times out is not
  reproducible.
* A failed external process may leave child processes running.
* A policy gets no feedback between its batches at one decision point.
* The policies are demonstrations, and all three acting ones use one
  lever (the reading allocation). No LLM agent is provided.
* T90 takes about 40 s per run on a Raspberry Pi 4.

**Validation.**
* The full uncached suite passes on linux/arm64 and linux/amd64.
* The race detector finds no race.
* vet and gofmt are clean.
* Every session fixture replays, and every golden reference is
  identical.
* The Python client and the Python reference policy pass.
* Benchmark verify and reproduce pass.
* Cross-builds succeed for five targets.
* CI is green on all three platforms and in the cross-platform
  comparison.

---

## V0.3.0 — the agent interface

An external program can now run a DreamTeam practice: discover the
actions, observe, act, advance time, save, replay and ask for the T90
evaluation, through a deterministic JSON protocol over the same session
API the game uses.

**`dreamteam agent --stdio`** (`dreamteam.agent/v1`,
`docs/AGENT_PROTOCOL.md`).
* **Format.** One JSON request per line in, one JSON response per line
  out. Ids are echoed, error codes are stable, serialization is fixed,
  and only responses go to stdout.
* **Operations:**
  * `hello`;
  * `observe`: in full, or as named `sections`, or `compact`, listed by
    `sections`;
  * `actions`: every action type with its JSON Schema, constraints and
    current targets;
  * `validate` and `preview` (for the staffing actions);
  * `apply`: atomic; a refused batch changes nothing, not even a digest;
  * `advance`: by minutes, to a time, to the next visible event, or to
    the end of the day;
  * `save` and `digests`;
  * `t90`.
* **Observation projections** are cut from the full observation's own
  JSON. `compact` leaves out four declared record listings and counts
  them, ≈75% smaller.
* **`t90`** is the same evaluation as `dreamteam t90`. It is explicit
  only (never part of `observe`), read-only and draws no randomness.
  Sessions saved by V0.2.0 are evaluated with their earlier decisions.
* **Determinism.** Identical requests give byte-identical responses, and
  a saved agent session replays headlessly (`dreamteam verify`).
* **Scripted runner.** `dreamteam agent -policy noop|greedy` exercises
  the protocol in process.

**Reference Python client** (`clients/python/`, standard library only).
It runs a complete session through `dreamteam agent --stdio` as a
subprocess, then checks the save with `dreamteam verify` and `dreamteam
t90`. CI runs it on linux/amd64, linux/arm64 and darwin/arm64 and
compares its transcript byte for byte.

**Fixes.**
* **The T90 view in a continued session.** A session saved by an
  earlier build is now evaluated with its earlier decisions, when they
  replay under the current action rules. This preserves the T90
  evaluation of sessions saved by V0.2.0.
* **CI.** It had been failing since the V0.2 Step 0C commit, on two
  defects of its own:
  * the golden step lacked the migration manifest;
  * a neutrality test held only on arm64.

  Both are fixed, and CI is green on every platform.

**Numerical change on arm64.** The evaluation now rounds each product
before a sum, because arm64 fuses multiply-add and amd64 does not.
* T90 output is byte-identical across platforms.
* arm64 T90 statistics (design effects, Wilson and t interval bounds,
  dispersion) can differ in their last digit from V0.2.0's. amd64's are
  unchanged.
* No status, verdict, rule, policy or simulation outcome changed.
* In developer-only multi-seed evaluations, the joint service gate's
  exact Clopper–Pearson bound still goes through the Go math library and
  can differ in its last digit between processor families.

**Compatibility.**
* No change to mechanics, randomness, scenarios, actions or accounting.
* All V0.1 and V0.2 golden references give identical outcomes.
* Saves from V0.1, every V0.2 build, V0.2.0 and the V0.3 development
  builds replay with their digests verified.
* The evaluation policy in force is still v0.2-eval-4. The clinical
  candidate v0.2-eval-5 is closed, not approved, and plays no part.

**Known limitations** (`README.md`).
* `t90` takes about 40 s on a Raspberry Pi.
* Previews cover the staffing actions only.
* The transport is stdio only: no HTTP, no model integration, no agent
  scoring or benchmark yet.
* The roadmap's V0.3 operations-control actions (reassign, call-in,
  send-home, templates, rescheduling, maintenance, disruptions) remain
  planned.

**Validation.**
* The full uncached suite passes on linux/arm64 and linux/amd64.
* The race detector finds no race on any shipped package.
* vet and gofmt are clean.
* Every session fixture replays, and every golden reference is
  identical.
* The external Python client passes.
* CI is green across all three platforms and the cross-platform
  comparison.

---

## V0.2.0 — practice design

You can now change how the practice is staffed and read, and see whether
your plan is better than doing nothing.

**Five staffing and reading actions** (staff screen `m`, or batches in an
actions file). Each one is validated and previewed, and it replays exactly.
* **Hire** from a fixed offer table, at a standard or premium rate. The
  premium buys a shorter recruiting lead. The recruiting fee, paid
  onboarding and 40 hours of training (24 for the desk) come before any
  patient capacity.
* **Terminate**, with at least 14 days' notice at contract pay and two
  weeks' severance.
* **Set rosters**: dated weekly or fortnightly patterns, from a Monday at
  least 14 days ahead.
* **Set a radiologist's site reading allocation** (30–100%), from at least
  14 days ahead.
* **Contract teleradiology** (`standard-v1`): weeknight, Saturday and
  Sunday windows, urgent or all studies, monthly retainers and per-read
  fees, and a 30-day minimum term on cancellation.

Batches are atomic. A refused action carries a stable code and changes
nothing. Displaced bookings are rebooked or lost, and the preview names
them first.

**Employment obligations.**
* Contracts are inferred for the existing staff at the takeover.
* Contracted hours are guaranteed, whatever is rostered.
* Weekly overtime premium applies above 40 actual hours.
* Severance and vendor costs are posted and settled in the double-entry
  ledger.

**The `sustainable` scenario.**
* The established practice at ×1.30 referral demand, with 12-minute X-ray
  slots and a full-time evening MR technologist.
* Shared XR/CT staffing, near break-even (≈+0.5% mean T90 margin).
* Reorganizing it is the game. The measured lead staffing earned ≈+2.8% at
  this demand, but getting there costs recruiting, notice, severance and
  training, and is not promised.

**The T90 evaluation** (summary screen `e`, `tab` twice, or
`dreamteam t90 FILE`).
* Your decisions before day 90 are judged against no action, on the same
  scenario and seed, by the frozen evaluation policy in force,
  v0.2-eval-4.
* **The metric vector** shows profit, margin, cash, the funding gap, served
  and lost demand, and report breaches and turnaround by priority. Your
  plan, the baseline and the difference are side by side, with no weighted
  score.
* **The service gates** show pass, fail or insufficient against the
  policy's limits, with their counts.
* **The plan gates and the policy's verdict** follow, with its reasons.
* Days not yet played are marked as projected, the post-day-90 tail as
  fixed-policy, and unfinished work as incomplete.
* The report is the canonical evaluation runner's, byte for byte.

**Sessions.**
* Session v2 records action batches with their results.
* Saves from V0.1 and from every V0.2 development build replay with
  their digests verified.
* The V0.1 golden references are unchanged.

**Known limitations** (`README.md`).
* TUI actions are committed one at a time, without showing the stable
  code; rostered and worked hours are not in the staff table.
* The T90 evaluation covers the sustainable scenario only.
* The Pi 4's 90-day headless run (≈3.3–3.9 s) keeps V0.1's waiver.
* No operations-control actions, disruptions or agent interface yet: V0.3
  is next.
* The clinical candidate policy v0.2-eval-5 completed its confirmatory
  evaluation with a NO GO and is not part of this release
  (`docs/calibration/clinical-v5-closure/`).

**Validation** (`docs/v02-acceptance/`).
* All five actions are verified through the simulator, API, sessions,
  headless runs and the TUI.
* Save, replay and determinism hold across amd64 and arm64.
* All 7 golden references and all 25 earlier session fixtures pass.
* The race detector finds no race.
* 150 × 90-day runs check every invariant after every transaction; 720-day
  turnover and roster churn pass.
* The full uncached suite passes.

---

## V0.1.0 — first playable release

A deterministic simulation and terminal game of an outpatient radiology
practice. You take over a struggling imaging center on a Tuesday morning
and run it.

**Simulation.** A discrete-event engine with keyed, platform-independent
randomness (SHA-256): the same scenario, seed and actions give the same
outcome on every machine, at any speed and any frame rate. Referrals and
walk-ins, booking against room templates and real technologist coverage,
arrivals, no-shows and cancellations, registration, imaging, a radiology
worklist with turnaround targets, contrast supervision, closing rules
(admitted patients are always served, with costed overtime). A 180-day
warm-up gives the practice a history before the takeover.

**Terminal UI** (Bubble Tea). Dashboard, schedule, patients and journeys,
worklist, finance (overview, claims, journal), event log with filters,
takeover briefing, daily history, end-of-day summary, actions and help.
Five speeds from 1 min/s to Max, step to the next event, auto-pause on
alerts. Layouts from 80×24 to 160×48; honors `NO_COLOR`.

**Three management actions.** Extend a shift (approved overtime, today or
ahead, opening bookable slots); walk-in policy (accept always, if a gap,
until a cutoff, or decline with an appointment offer); worklist policy
(urgent & overdue then FIFO, FIFO, oldest first, shortest expected read —
urgent studies always first).

**Finance and metrics.** Double-entry ledger with accrual P&L, cash and a
balance sheet; claims with payer lag, denials and rework; payroll and
overtime; fixed and variable costs. Operational metrics (utilization,
waits, TAT by priority, lost demand, walk-in cohorts), alerts that say
why, and an evaluation window from the exact takeover.

**Sessions and replay.** Save and load (`ctrl+s`, `ctrl+o`); a session is
its scenario, seed and actions, and loading replays it from scratch,
checking five digests (log, ledger, actions, state, run).

**Headless CLI.** `dreamteam run` (text or canonical JSON results,
`-actions`, `-save`, `-log events.jsonl`, `-verify-replay`), `replay`,
`verify`, `scenario list|validate|coverage`; golden references with
explicit, reviewed regeneration; documented exit codes.

**Scenarios.** `established` (default): the struggling, capacity-constrained
practice — demand ≈1,650 a month, ≈1,300–1,380 exams, operating margin
around −12% to −14% (nine-seed characterization in
`docs/calibration/`). `mini`: the same practice without history.

**Platforms.** Linux and macOS with Go 1.26 or later. CI: linux/amd64,
linux/arm64, darwin/arm64, with the golden runs byte-identical across all
three. Developed on a Raspberry Pi 4.

**Validation.** Invariants after every transaction for 50 seeds × 90 days;
1,000 random chunkings of the real simulation; TUI sessions reproduced
headless; leakage tests (no hidden future in observations or results).

**Known limitations.** One practice and three actions: no hiring, roster
or template editing, equipment failures, absences, rescheduling or
overbooking. Urgent studies acquired after the last reading block wait
for the next morning or Monday, with no action yet to add evening or
weekend reading. No mouse, no file picker.

**Accepted deviations from the V0.1 criteria** (owner decision):
* On a Raspberry Pi 4, `dreamteam run -days 90` takes ≈3.3 s against a
  2 s target (its CPU lacks SHA-256 instructions); interactive targets are
  met (warm-up 1.44–1.49 s, TUI 7.4–8.2% of a core at 15 min/s, Max-speed
  frames under 100 ms). A Cortex-A76 meets every target. The RNG is kept.
* The default practice's takeover worklist is small or empty (it is
  imaging-constrained, not reporting-constrained); the 5–20-study target
  is deferred to a future reporting-constrained scenario.

**Next (V0.2).** A better-staffed, profitable scenario and more
management actions — hiring, roster adjustments, and evening and weekend
radiologist coverage.
