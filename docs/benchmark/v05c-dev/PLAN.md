# V0.5c development evaluation plan

Plan version `v05c-plan/1`, written 2026-10-27 and committed before any
run on the seeds below. V0.5c compares local LLMs as managers of the same
practice under identical conditions, and tests whether the agent's
bounded memory changes their behaviour. **It is a development evaluation,
not a benchmark result:**
* no composite score, ranking or significance test;
* no claim that generalizes beyond these models, seeds and horizons.

Anything this plan does not settle is decided before the stage it affects
runs, and committed as a dated amendment at the end of this file. An
amendment never changes a seed, a model, a setting or the agent.

## 1. The agent, frozen

The V0.5b agent, `agents/llm/llm_agent.py`:
* llm-agent 2, prompt `llm-agent-prompt/v2`, memory `llm-agent-memory/v1`;
* the same:
  * action guidance;
  * observation summary (`observation: summary`);
  * catalog (`catalog: guided`);
  * details request (`allowDetails: true`);
  * validation and retry (`maxRetries: 1`);
  * decision budget (900 s);
  * response format (`json_object`).

**The one change.** V0.5c changes the agent in one place only: with
`memoryDecisions: 0` the memory is now **off**.
* **Off:** the MEMORY section says the memory is off and carries nothing
  from earlier decision points: no decisions, refusals, totals or metric
  rows. It still lists the pending changes, which come from the current
  observation and catalog (§6 of [LLM_AGENT.md](../../LLM_AGENT.md)).
* **Before:** 0 kept the earlier decisions as totals, with the refusals
  and metric rows.
* **On:** with memory on, every memory and prompt is byte-for-byte as in
  V0.5b. The V0.5b recording's replay rebuilds them at all 14 decision
  points and matches their SHA-256.

**The settings,** one file per model in this directory, are identical
except for the model. A unit test checks this
(`test_v05c_settings_identical`):

| Setting | Value |
|---|---|
| `temperature` | 0 |
| `maxOutputTokens` | **4096** for every model, chosen before any run. In V0.5b one reply was cut off at 2,048 tokens. |
| `maxRetries` | 1 |
| `requestTimeoutSeconds`, `decisionBudgetSeconds` | 300, 900 |
| `responseFormat` | `json_object` |
| `maxPromptChars` | 200,000 |
| `observation`, `catalog`, `allowDetails` | `summary`, `guided`, true |
| `memoryDecisions`, `memoryMetricPoints`, `memoryMaxChars` | 6, 6, 6,000 (the memory-off control: `memoryDecisions` 0) |
| Prices | none: local models have no billing, so no cost is estimated |

The server's address comes from `DREAMTEAM_LLM_BASE_URL` and is never
committed. No API key is used.

**Decision cadence:** daily at 07:00 practice time. The harness's timeout
is 1,800 s per decision point, above the agent's own 900 s budget.

## 2. Models

Three models were chosen from the 17 the server lists, before any run on
these seeds. The aim was two model families, and both mixture-of-experts
and dense models:

| Policy ID | Model | Architecture | Weights (on the server) |
|---|---|---|---|
| `qwen36-35b-a3b` | `Qwen3.6-35B-A3B-oQ4e-mtp` | Qwen3.6, mixture of experts: 35B in all, 256 experts, 8 active (about 3B active), 40 layers | `Jundot/Qwen3.6-35B-A3B-oQ4e-mtp`, 21.6 GB; index SHA-256 `a0fa68e6…87ebb9`, config `8ed2dcf3…81c8ff` |
| `gemma4-26b-a4b` | `gemma-4-26B-A4B-it-oQ4e-mtp` | Gemma 4 instruct, mixture of experts: 26B in all, 128 experts, 8 active (about 4B active), 30 layers | `Jundot/gemma-4-26B-A4B-it-oQ4e-mtp`, 16.7 GB; index `fa8d28e0…7c2844`, config `97704c41…d5757c` |
| `gemma4-31b` | `gemma-4-31B-it-oQ4e-mtp` | Gemma 4 instruct, dense, 31B, 60 layers | `Jundot/gemma-4-31B-it-oQ4e-mtp`, 20.0 GB; index `4019c66e…bec087`, config `a3d961d0…ccfabc` |

**What the three models share:**
* **Quantization:** 4-bit affine (oQ4e), group size 64, with some tensors
  at 5 or 8 bits.
* **Speculative decoding:** multi-token prediction (MTP) is on.
* **Thinking:** off for Qwen (the server's per-model setting); Gemma has
  none set.

**The server:**
* **Software and hardware:** omlx 0.7.0 (MLX), an OpenAI-compatible
  `chat/completions` server, on an Apple silicon Mac with 128 GB of memory
  and macOS 26.4. It is reached over an SSH tunnel, and its address is
  not committed.
* **Context:** each model supports a context of 262,144 tokens, but the
  server caps every request at 32,768 tokens. Its default output limit is
  32,768 tokens; the agent asks for at most 4,096.
* **Sampling:** the agent sends `temperature` 0. The server's defaults for
  the rest are top_p 0.95, top_k 0 and repetition penalty 1.0.
* **Sharing:** the server may serve other clients and load or unload
  models, so latency is measured, not controlled.

**Probe** (2026-10-27): one request per model on the V0.5b prompt for V0.5a's
recorded point 0, from seed 40601 and already used. All three answered with
valid JSON and stopped normally (`stop`):

| Model | Prompt tokens | Latency | Note |
|---|---|---|---|
| `Qwen3.6-35B-A3B-oQ4e-mtp` | 9,756 | 4.0 s | |
| `gemma-4-26B-A4B-it-oQ4e-mtp` | 9,828 | 3.3 s | |
| `gemma-4-31B-it-oQ4e-mtp` | 9,828 | 23.2 s | 5.6 s of it loading the model |

**Not chosen:**

| Model | Why |
|---|---|
| `Qwen3.6-27B-oQ4e-mtp` | Unavailable: its weights fail to load on the server (a safetensors file is incomplete). |
| `Qwen3.8-27B-oQ4e-mtp` | Thinking is on in the server's settings for it, so with the frozen agent it would spend the output budget on reasoning. Changing server settings per model is out of scope. |
| `Qwen3.8-Flash-Next-oQ4e-mtp` | Three models at most; the chosen three already cover two families and both architectures. |
| `-heretic`, `-unsloth`, `-optiq`, `-4bit` variants | Other quantizations of the same base models. |
| `gemma-4-e4b`, `gemma-e4b`, `qwen3.5-9b` | Three models at most. |

**Failure policy.** A model that fails is classified and reported. It is
never replaced by another model.

## 3. Scenario, seeds and stages

* **Scenario:** the bundled `sustainable` scenario throughout.
* **Seeds:** 40801–40805, the next unused block, fixed now:
  * not used anywhere else in the repository;
  * none is a reference seed (1001–1020), a V0.2 development seed
    (901–910), a seed from a reserved stream, or one of the earlier
    development seeds (40401–40405, 40501–40505, 40601, 40701);
  * `TestCommittedConfigs` checks every configuration against the sealed
    holdout guard without running anything.
* **T90:** none.
* **Baseline:** every stage runs `noop` as its own baseline, on every seed.

| Stage | Configuration | Seeds | Days | Policies | Runs |
|---|---|---|---|---|---|
| A, integration | `stage-a.json` | 40801 | 7 | noop, the three models | 4 |
| B, comparison | `stage-b.json` | 40802, 40803, 40804 | 30 | noop, the three models, memory on | 12 |
| C, memory | `stage-c.json` | 40802, 40803, 40804 | 30 | noop, `qwen36-35b-a3b-nomem` (memory off) | 6 |
| D, extended | `stage-d.json` | 40805 | 90 | noop, the models that passed Stage A | 4 |

**The memory model** is `Qwen3.6-35B-A3B-oQ4e-mtp`, chosen now, before any
Stage B outcome is seen. In V0.5b it repeated a refused proposal although
the refusal was in its memory, and it is the fastest of the three.
* **Memory on:** its Stage B runs, with the same seeds, settings, agent
  and engine.
* **Memory off:** Stage C, which runs noop again so that each stage is
  self-contained. Noop is deterministic, so its runs must equal Stage B's,
  and the report checks that.

The stages run in order: A, B, C, D, one benchmark at a time on the
Raspberry Pi 4 (engine 0.5.0-dev.3), against the local server.

## 4. Stop conditions and resource limits

1. **Stage A gate.** A model passes Stage A if its 7-day run completes,
   with a saved session that verifies, and at least one decision point
   ends without a fallback.
   * **Failure:** the run is classified as protocol, crash, timeout,
     invalid output (every decision a fallback) or infrastructure, and the
     model drops out of Stages B to D.
   * **Removal:** an amendment removes it from those configurations before
     they run, and adds nothing.
   * **Fewer than two passing models:** V0.5c stops after Stage A and
     reports.
   * **Memory model fails:** Stage C does not run.
2. **Infrastructure failure.** An HTTP or connection error, a request
   timeout, or the server unavailable is an infrastructure failure. A
   model's own errors are not: invalid JSON, an invalid decision, a reply
   cut off by the token limit, or a refusal.
   * **First failure:** a run hit by one is reported and rerun once in
     full, into a new directory. Both runs are kept and reported.
   * **Second failure:** if the rerun fails as well, the stage stops,
     with the remaining runs reported as incomplete.
3. **Resource limits** (wall clock for the whole benchmark command):

   | Stage | Limit |
   |---|---|
   | A | 2 h |
   | B | 8 h |
   | C | 4 h |
   | D | 8 h |

   * **Estimate:** from the probe and V0.5b's 1.4 requests per decision
     point, B is about 2 h and D about 2–3 h.
   * **At the limit:** the command is stopped. The completed runs are
     reported, and the rest are reported as incomplete.
   * **No silent shortening:** a horizon is never shortened.
4. **Stage D** runs only if A and B finish without a major infrastructure
   failure: a run that failed twice, or a server unavailable for more than
   an hour.
5. **Nothing tuned.** Between stages nothing is changed: not the prompt,
   settings, retries, models or seeds. If a defect in the harness or the
   agent blocks a run, V0.5c stops and reports. Any fix is an amendment,
   committed before a rerun, and the earlier results are kept.

## 5. What is reported

**Sources:** existing metrics only, from the benchmark report
(`dreamteam.result/v1`) and the agent's evidence (`decisions.jsonl`,
`summary.json`).

**Stage A, for each model:**
* whether the run completed;
* decision outcomes: with actions, no action, fallback;
* validation refusals by code;
* protocol, request and denied errors;
* whether usage was recorded;
* verification and replay of the saved session;
* replay of the recorded decisions and `reproduce`.

**Stage B,** paired by seed, each model minus noop, and the three models
side by side:

| Area | Metrics |
|---|---|
| Finances | Operating profit, net revenue, operating expenses |
| Reports | Reports finalized; turnaround p50 and p90; breaches |
| Imaging | Exams completed and orders lost. There is no capacity metric; these are what the result records. |
| Labour | Overtime |
| Actions | Staffing changes (accepted staffing actions); action counts by type and category |
| Decisions | Refusals and fallbacks |

**Action categories**, as the agent defines them:
* **Staffing:** `hire`, `terminate`, `set_roster`,
  `set_reading_allocation`, `contract_teleradiology`.
* **Operations:** `extend_shift`, `set_walkin_policy`,
  `set_worklist_policy`.

**Stage C,** paired by seed, memory on minus memory off:
* repeated invalid proposals;
* the valid-decision rate;
* refusals and fallbacks;
* prompt and completion tokens;
* the Stage B outcome metrics.

No causal claim.

**Stage D:** each model minus noop on seed 40805 over 90 days, with the
Stage B metrics.

**Definitions:**
* **Valid decision:** a decision point that does not end in a fallback.
* **First-attempt valid:** the model's first reply parsed and was not
  refused by `validate`.
* **Repeated invalid proposal:** a `validate` refusal whose action type
  and code were already refused at an earlier decision point of the same
  run.

**Flagged explicitly:**
* **Never acts:** no accepted action in a run.
* **Repeats invalid proposals:** at least one in a run; the count is given.
* **Uses staffing actions:** any proposed, and any accepted.
* **Asks for details:** the number of requests.
* **Better operations at a financial cost:** turnaround p90 or breaches
  better than noop's while operating profit is lower. Both differences are
  given; whether the cost is excessive is left to the owner.

**Tokens and latency, for each model:**
* requests, retries and details requests;
* prompt and completion tokens, as reported, estimated or unknown;
* latency mean, median and maximum;
* total inference time.

**Replay:**
* every live run's recorded decisions are replayed without a model, by
  stage;
* the decisions, session, result and final digests must be byte-identical
  to the live run's, and the memory identical at every point;
* `verify` and `reproduce` must pass;
* the replay is checked again on a second machine (linux/amd64).

## 6. Evidence

**Committed, for each stage:**
* the benchmark report and manifest;
* each LLM run's recorded decisions and usage summary;
* the replay configuration and the replay's manifest.

**Kept outside the repository** (`~/dt-audit-kept/v05c`), because of
their size:
* the model requests and responses;
* the memory at each point;
* the transcripts of public observations.

The manifests record the SHA-256 of the agent and of every settings file.

**Deterministic and not:**
* **Deterministic,** so a rerun gives the same files: the replay of the
  recorded decisions.
* **Not deterministic:** live inference. A rerun of a live run may decide
  differently.

## Amendments

None.
