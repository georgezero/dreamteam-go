# V0.5c development evaluation: v05c-dev

Three local LLMs managed the same practice through the frozen V0.5b agent
([LLM_AGENT.md](../../LLM_AGENT.md)), under identical settings, on fresh
seeds declared before any run ([PLAN.md](PLAN.md), `v05c-plan/1`, committed
in 8c62526). One model also ran with its memory off.

**This is a development evaluation, not a benchmark result.** It shows that
several models complete sessions through the external agent, that their
decisions are recorded and replay exactly, and that outcomes, refusals,
usage and latency can be compared. Five seeds, one scenario and at most 90
days say nothing general about any model. There is no composite score, no
ranking and no significance test; the differences below are descriptions
of these runs, not claims.

## Files

| File | What it is |
|---|---|
| `PLAN.md` | The plan, committed before any run. No amendment was needed. |
| `llm-*.json` | The agent's settings, one per model, identical except for the model (and `memoryDecisions` 0 in `-nomem`). |
| `stage-{a,b,c,d}.json` | The stage configurations, as run. |
| `stage-*/report.txt`, `report.json`, `manifest.json` | Each live stage's benchmark report and manifest. |
| `stage-*/<policy>-s<seed>-decisions.jsonl` | Each live LLM run's recorded decisions: the replay input, with the memory's and prompt's SHA-256 at every point. |
| `stage-*/<policy>-s<seed>-usage.json` | Each LLM run's usage summary (no endpoint, no credential). |
| `stage-*-replay.json`, `stage-*/replay-manifest.json` | The replay of every recorded decision without a model, and its manifest. CI runs these on three platforms. |
| `analyze.py`, `tables.md` | The planned tables (§5 of the plan) and the script that builds them from the live result directories. |

Kept outside the repository, because of their size: the model requests and
responses (`model.jsonl`), the memory text at each point (`memory.jsonl`)
and the transcripts.

## 1. Model inventory

All three ran on one server: omlx 0.7.0 (MLX), an OpenAI-compatible
`chat/completions` server on an Apple silicon Mac (128 GB, macOS 26.4),
reached over an SSH tunnel whose address is not committed. No API key.

| Policy | Model | Architecture | Weights |
|---|---|---|---|
| `qwen36-35b-a3b` | `Qwen3.6-35B-A3B-oQ4e-mtp` | mixture of experts, 35B, about 3B active | `Jundot/Qwen3.6-35B-A3B-oQ4e-mtp`, 21.6 GB |
| `gemma4-26b-a4b` | `gemma-4-26B-A4B-it-oQ4e-mtp` | mixture of experts, 26B, about 4B active | `Jundot/gemma-4-26B-A4B-it-oQ4e-mtp`, 16.7 GB |
| `gemma4-31b` | `gemma-4-31B-it-oQ4e-mtp` | dense, 31B | `Jundot/gemma-4-31B-it-oQ4e-mtp`, 20.0 GB |

* **Shared:** 4-bit affine quantization (oQ4e), multi-token prediction on,
  thinking off. The weights' index and config SHA-256 are in PLAN.md §2.
* **Context:** 262,144 tokens supported; the server caps each request at
  32,768. First-attempt prompts were 10,000–12,600 tokens.
* **Sampling:** `temperature` 0 from the agent; server defaults top_p
  0.95, top_k 0, repetition penalty 1.0. Live inference is not
  deterministic even so.
* **Output limit:** 4,096 tokens for every model, chosen before any run.
* **Agent:** llm-agent 2, prompt `llm-agent-prompt/v2`, memory
  `llm-agent-memory/v1` (6 decisions, 6 metric rows, 6,000 characters);
  daily decisions at 07:00; harness timeout 1,800 s per point.
* **Runtime:** benchmarks on a Raspberry Pi 4 (linux/arm64, Python 3.11),
  engine 0.5.0-dev.3, one stage at a time.
* **Not used:** `Qwen3.6-27B-oQ4e-mtp`, whose weights fail to load on the
  server (HTTP 409; an incomplete safetensors file). It was excluded before
  any run, not replaced after one. Other exclusions are in PLAN.md §2.

## 2. Plan and how it ran

| Stage | Seeds | Days | Policies | Wall clock | Limit | Result |
|---|---|---|---|---|---|---|
| A, integration | 40801 | 7 | noop + 3 models | 4.9 min | 2 h | all completed; all 3 models passed the gate |
| B, comparison | 40802–40804 | 30 | noop + 3 models | 77 min | 8 h | 12 of 12 completed |
| C, memory off | 40802–40804 | 30 | noop + Qwen, memory off | 15 min | 4 h | 6 of 6 completed |
| D, extended | 40805 | 90 | noop + 3 models | 2 h 33 min | 8 h | 4 of 4 completed |

* **No infrastructure failure** in any stage: no HTTP, timeout or
  connection error, so no rerun. No run was incomplete and no horizon was
  shortened.
* **Nothing changed between stages:** prompt, settings, models and seeds
  as committed. No amendment.
* **Noop control:** Stage C's noop runs have the same final digests as
  Stage B's, as the plan requires.

## 3. Stage A: integration (seed 40801, 7 days)

| Model | Completed | With actions / no action / fallback | Refusals by `validate` | Accepted | Outcome vs noop |
|---|---|---|---|---|---|
| Qwen 35B-A3B | yes | 1 / 5 / 1 | `extend_shift`: `after_close` 3, `exceeds_max_shift` 2 | 3 `extend_shift` | profit +$623, 8 more exams, overtime +634 min |
| Gemma 26B-A4B | yes | 0 / 7 / 0 | none | none | identical to noop |
| Gemma 31B | yes | 0 / 7 / 0 | none | none | identical to noop |

No protocol, request or denied errors; usage recorded for every request;
every saved session verified; `verify` 42 of 42. The Gemma models' reasons
were that a radiologist's reading block was about to start.

## 4. Stage B: comparison (seeds 40802–40804, 30 days)

**Decisions** (valid = no fallback; first-attempt valid = the first reply
parsed and was not refused by `validate`):

| Seed | Model | Actions / none / fallback | First-attempt valid | Repeated invalid | Accepted (staffing) |
|---|---|---|---|---|---|
| 40802 | Qwen | 5 / 24 / 1 | 25/30 | 2 | 5 (2) |
| 40802 | Gemma 26B | 3 / 21 / 6 | 19/30 | 13 | 4 (3) |
| 40802 | Gemma 31B | 0 / 30 / 0 | 30/30 | 0 | 0 |
| 40803 | Qwen | 5 / 25 / 0 | 25/30 | 1 | 7 (3) |
| 40803 | Gemma 26B | 0 / 30 / 0 | 30/30 | 0 | 0 |
| 40803 | Gemma 31B | 0 / 30 / 0 | 30/30 | 0 | 0 |
| 40804 | Qwen | 6 / 23 / 1 | 24/30 | 5 | 7 (0) |
| 40804 | Gemma 26B | 0 / 30 / 0 | 30/30 | 0 | 0 |
| 40804 | Gemma 31B | 2 / 28 / 0 | 29/30 | 0 | 2 (1) |

**Paired with noop** (policy − noop; times in minutes):

| Seed | Model | Δprofit | Δrevenue | Δexpenses | Δreports | ΔTAT p50 | ΔTAT p90 | Δbreaches | Δlost | Δovertime |
|---|---|---|---|---|---|---|---|---|---|---|
| 40802 | Qwen | −$5,500 | +$2,523 | +$8,022 | +13 | −31 | −25 | −28 | 0 | +360 |
| 40802 | Gemma 26B | −$2,160 | $0 | +$2,160 | 0 | −3 | −53 | −36 | 0 | +120 |
| 40802 | Gemma 31B | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| 40803 | Qwen | −$8,585 | +$3,022 | +$11,607 | +14 | −30 | −29 | −15 | −1 | +232 |
| 40803 | Gemma 26B, 31B | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| 40804 | Qwen | −$161 | −$16 | +$145 | −1 | −1 | 0 | −2 | 0 | +1,363 |
| 40804 | Gemma 26B | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| 40804 | Gemma 31B | −$1,296 | +$562 | +$1,858 | +5 | −19 | −100 | −28 | 0 | +210 |

Exams completed did not change by more than 1 in any Stage B run. Absolute
values and every metric are in `tables.md` and `stage-b/report.txt`.

## 5. Stage C: memory off (Qwen, seeds 40802–40804, 30 days)

With `memoryDecisions` 0 the MEMORY section says the memory is off and
lists only the pending changes from the current observation (189–1,946
characters, against up to 5,585 with memory on). The memory-on runs are
Qwen's Stage B runs: same seeds, settings, agent and engine.

| Seed | Memory | Valid | First-attempt valid | Points with a refusal | Refusals | Repeated invalid | Fallbacks | Accepted (staffing) | Prompt / completion tokens | Requests |
|---|---|---|---|---|---|---|---|---|---|---|
| 40802 | on | 29 | 25 | 3 | 6 | 2 | 1 | 5 (2) | 429,262 / 6,091 | 35 |
| 40802 | off | 29 | 20 | 10 | 14 | 11 | 1 | 24 (0) | 412,370 / 9,166 | 40 |
| 40803 | on | 30 | 25 | 4 | 5 | 1 | 0 | 7 (3) | 430,898 / 8,995 | 35 |
| 40803 | off | 28 | 20 | 7 | 16 | 5 | 2 | 23 (10) | 467,877 / 17,500 | 40 |
| 40804 | on | 29 | 24 | 6 | 9 | 5 | 1 | 7 (0) | 408,494 / 5,749 | 36 |
| 40804 | off | 21 | 14 | 15 | 55 | 39 | 9 | 11 (10) | 523,158 / 17,183 | 46 |

| Seed | Memory | Δprofit vs noop | ΔTAT p90 | Δbreaches | Δexams | Δlost | Δovertime |
|---|---|---|---|---|---|---|---|
| 40802 | on | −$5,500 | −25 | −28 | 0 | 0 | +360 |
| 40802 | off | −$340 | −13 | −14 | +2 | −1 | +2,158 |
| 40803 | on | −$8,585 | −29 | −15 | +1 | −1 | +232 |
| 40803 | off | −$17,747 | −71 | −41 | +18 | +39 | +1,119 |
| 40804 | on | −$161 | 0 | −2 | 0 | 0 | +1,363 |
| 40804 | off | −$17,639 | −159 | −57 | −20 | +22 | −1 |

**What these runs show** (three seeds; no causal claim):
* On every seed, with memory off Qwen had fewer first-attempt valid
  replies, more refusals and more repeated invalid proposals, used more
  requests, and accepted more actions.
* On seed 40804 with memory off, 9 of 30 points fell back, and
  `validate` refused 13 shift extensions past the maximum and 12 past the
  close.
* With memory off it ran a mixed operations-and-staffing decision (an
  operations batch and a staffing batch at one point) four times; with
  memory on, once (Stage D). All were accepted.
* Prompt tokens were not lower with memory off on two of three seeds: the
  memory saved up to about 1,400 tokens a request, and the extra retries
  used more.

## 6. Stage D: extended (seed 40805, 90 days)

| Model | Actions / none / fallback | First-attempt valid | Repeated invalid | Accepted | Δprofit | Δexpenses | ΔTAT p50 | ΔTAT p90 | Δbreaches | Δlost | Δovertime |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Qwen | 7 / 83 / 0 | 85/90 | 2 | 9: 5 `extend_shift`, worklist `oldest_first`, 2 teleradiology (start, then cancel), 1 `hire` | −$34,565 | +$34,693 | −78 | −859 | −295 | −3 | +639 |
| Gemma 26B | 2 / 86 / 2 | 87/90 | 1 | 3 teleradiology (start Sundays; cancel and start a wider contract) | −$13,517 | +$14,122 | −24 | −107 | −173 | 0 | 0 |
| Gemma 31B | 0 / 90 / 0 | 90/90 | 0 | none | 0 | 0 | 0 | 0 | 0 | 0 | 0 |

* Noop over 90 days: profit $2,940, TAT p90 980 min, 400 breaches.
* **Qwen** started weeknight and weekend teleradiology at point 5 (in the
  same decision as the worklist policy), then at point 49 cancelled it
  from 2026-12-31 to stop the retainer, and at point 60 hired a CT
  technologist, resubmitting with the start date its earlier `hire` had
  been refused for (`lead_too_short`). TAT p90 fell from 980 to 121 min.
* **Gemma 26B** started Sunday teleradiology at point 5 and at point 47
  replaced it with weeknight and weekend cover, after refusals for notice,
  overlap and schema.
* **Gemma 31B** decided no action at all 90 points; its run's final digest
  equals noop's.

## 7. Action categories

Accepted actions, all runs of a policy:

| Policy | Operations accepted | Staffing accepted | Staffing proposed or refused | Types used |
|---|---|---|---|---|
| Qwen (A, B, D) | 23 | 8 | 11 | `extend_shift`, `set_worklist_policy`, `set_reading_allocation`, `contract_teleradiology`, `hire` |
| Qwen, memory off (C) | 38 | 20 | 54 | all but `set_walkin_policy` |
| Gemma 26B (A, B, D) | 1 | 6 | 36 | `extend_shift`, `contract_teleradiology` |
| Gemma 31B (A, B, D) | 1 | 1 | 1 | `extend_shift`, `contract_teleradiology` |

No batch was refused when applied in any run: every refused proposal was
caught by `validate` before submission.

## 8. Invalid proposals

Repeated invalid proposal: a `validate` refusal whose action type and code
were already refused at an earlier point of the same run.

* **Qwen, memory on:** 2, 1, 5 (Stage B), 2 (Stage A), 2 (Stage D). Mostly
  `extend_shift` past the close (`after_close`) or past the maximum
  shift, the pattern seen in V0.5b; the refusals stay in memory, and the
  retry usually drops them.
* **Qwen, memory off:** 11, 5, 39. On 40804, refusals for `after_close`
  (12), `exceeds_max_shift` (13), `not_on_shift` (10) and seven rejected
  reading-allocation batches.
* **Gemma 26B, seed 40802:** 13. It proposed teleradiology at points 6–10
  and 20, and was refused in turn for `notice_too_short`,
  `contract_overlap` and `invalid_schema`, with the earlier refusals in its
  memory; 6 points fell back. On Stage D, 1.
* **Gemma 31B:** none (one `after_close` refusal, not repeated).

## 9. Tokens and latency

As the server reported them; no estimated or unknown usage in any run.
Latency is wall-clock per request on a shared server, measured, not
controlled.

| Policy | Runs | Requests per point | Mean first prompt (tokens) | Completion tokens per run | Latency mean |
|---|---|---|---|---|---|
| Qwen | A, B, D | 1.06–1.29 | 10,200–12,600 | 2,100–12,500 | 3.0–5.1 s |
| Qwen, memory off | C | 1.33–1.53 | 10,200–11,500 | 9,200–17,500 | 4.1–6.1 s |
| Gemma 26B | A, B, D | 1.00–1.37 | 10,200–12,400 | 550–6,800 | 3.5–7.1 s |
| Gemma 31B | A, B, D | 1.00–1.03 | 10,100–11,500 | 440–5,700 | 21.6–84.2 s |

* Gemma 31B is a dense model and the slowest: 33–37 s a request in Stage
  B, 84 s in Stage D (7,582 s of inference in its 90-day run, against 480
  s for Qwen). Why Stage D was slower was not established; the server is
  shared.
* No model asked for details.
* One Qwen reply (Stage B, 40803) was cut off at 4,096 tokens.
* No cost: local models have no billing, and no prices are configured.

Per-run figures are in `tables.md`.

## 10. Replay and integrity

Every live run's recorded decisions were replayed without a model
(`stage-*-replay.json`), with the memory and prompt rebuilt at every
decision point and checked against their recorded SHA-256.

| Check | A | B | C | D |
|---|---|---|---|---|
| Live `benchmark verify` | 42/42 | 114/114 | 60/60 | 42/42 |
| Replay `verify` (linux/amd64, Python 3.13) | 42/42 | 114/114 | 60/60 | 42/42 |
| Replay `reproduce` | 31/31 | 79/79 | 43/43 | 31/31 |
| Two replays: manifests | identical | identical | identical | identical |
| Replay time (both on linux/amd64) | 10 s | 75 s | 36 s | 70 s |

For all 18 LLM runs, the replay's `decisions.json`, `session.json`,
`result.json`, `memory.jsonl` and final digests are byte-identical to the
live run's. CI replays all four stages on linux/amd64, linux/arm64 and
darwin/arm64 and compares the manifests byte for byte.

**Deterministic and not.** The replay is deterministic. Live inference is
not: a rerun of a live run may decide differently, even at temperature 0,
and its latency will differ. The manifests list the agent's live evidence
as nondeterministic.

**Unchanged:** the simulator, the protocol, scoring and the golden
references. The engine moved to 0.5.0-dev.3 only to record V0.5b's
sessions as a release fixture; earlier sessions still replay.

## 11. Limitations

* **Small:** one scenario, five seeds, three models, one server, one
  quantization each. Stage D is one seed. Nothing here is a ranking.
* **Not repeated:** each live run ran once; how much a model's decisions
  vary between reruns is unmeasured.
* **Memory comparison:** three seeds, one model. The memory-on and -off
  runs differ in many decisions, so the outcome differences cannot be
  attributed to the memory.
* **Inactive models look like noop:** Gemma 31B acted on 2 of its 187 points;
  Gemma 26B on 5 of 187. Their outcomes mostly equal noop's, which says
  little about what they would do when they act.
* **Outcomes are the simulator's:** costs and revenues follow its model of
  a practice; "better service at a financial cost" is reported, and
  whether the cost is excessive is left to the reader.
* **Prompt fixed:** with a different prompt the models might act very
  differently; prompt tuning was out of scope.
* **Server:** shared and not controlled; latency and even availability can
  change.

## Flags

| Flag | Runs |
|---|---|
| Never acts | Gemma 31B on 40801, 40802, 40803, 40805; Gemma 26B on 40801, 40803, 40804 |
| Repeats invalid proposals | Qwen on every seed (1–5 with memory, 5–39 without); Gemma 26B on 40802 (13) and 40805 (1) |
| Uses staffing actions | Qwen (accepted on 40802, 40803, 40805; with memory off on all three); Gemma 26B (40802, 40805); Gemma 31B (40804) |
| Asks for details | none |
| Better service, lower profit | Qwen on 40802 (−$5,500), 40803 (−$8,585), 40804 (−$161), 40805 (−$34,565); Qwen memory off on 40802 (−$340), 40803 (−$17,747), 40804 (−$17,639); Gemma 26B on 40802 (−$2,160), 40805 (−$13,517); Gemma 31B on 40804 (−$1,296) |
