# V0.5b integration smoke test: v05b-dev

The V0.5b live run of the LLM agent, version 2 ([LLM_AGENT.md](../../LLM_AGENT.md)):
bounded memory, action guidance, a targeted prompt, and usage reporting.
It ran on the bundled sustainable scenario, seed 40701, for 14 days, with
decisions daily at 07:00 and no T90. Two runs: `noop`, the baseline, and
`llm`, the agent live with one local model.

**This is an integration smoke test, not a performance benchmark and not a
model comparison.** It shows that the agent with memory takes part in a
complete, replayable session, carries its own history across decision
points, and stays within its bounds. One model, one seed and 14 days say
nothing about decision quality, and the report makes no claim.

## Files

* `config.json`, `llm-settings.json`: the configuration and the agent's
  settings, committed with the agent (e30aacf) before any run on the seed.
  * The model, `Qwen3.6-35B-A3B-oQ4e-mtp`, is V0.5a's, kept so that the
    slice changes the agent, not the model.
  * The server's address came from `DREAMTEAM_LLM_BASE_URL` and is not
    committed. No API key was needed, and no prices are configured (a local
    model has no billing).
  * 14 days, not 7, so that the memory runs past its 6-decision bound.
* `manifest.json`, `report.json`, `report.txt`: the live run's manifest and
  report. The manifest records the SHA-256 of the agent (`aebdc338…`) and
  of the settings (`47f1c5e0…`), the committed files. It lists the agent's
  evidence files as nondeterministic.
* `llm-decisions.jsonl`: the agent's recorded decisions, the replay input.
  For every decision point it holds the memory's and the prompt's SHA-256.
* `llm-usage.json`, `llm-usage.txt`: the agent's usage summary
  (`summary.json`, `usage.txt`), with no endpoint and no credential.
* `replay.json`, `replay-manifest.json`: the replay of those decisions
  without a model, and its manifest.

The model exchanges (`model.jsonl`, about 2 MB) and the memory text at each
point (`memory.jsonl`) are kept outside the repository.

## Evidence (2026-10-27)

**Live run.** On a Raspberry Pi 4 (linux/arm64, Python 3.11), engine
0.5.0-dev.2, against a local OpenAI-compatible server:
* both runs completed: `noop` in 17.2 s, `llm` in 114.8 s;
* every saved session replayed with every digest verified;
* `benchmark verify` passed 24 of 24 checks.

The agent used only `hello` (1), `digests` (14), `observe` (14), `actions`
(14) and `validate` (11). The gate denied nothing, no request erred, and
its standard error was empty.

| Point | Day | Outcome | Attempts | Refused by `validate` first | Submitted |
|---|---|---|---|---|---|
| 0 | Tue 10-13 | no action | 1 | — | — |
| 1 | Wed 10-14 | actions | 2 | 3 × `after_close` | worklist policy `oldest_first`; extensions E07 +210, E03 +180, E05 +150, E06 +150 min |
| 2 | Thu 10-15 | no action | 1 | — | — |
| 3 | Fri 10-16 | actions | 2 | `after_close` | E07 +180 min |
| 4 | Sat 10-17 | no action | 2 | `no_coverage`, `not_on_shift` | — (the retry chose no change) |
| 5 | Sun 10-18 | no action | 1 | — | — |
| 6 | Mon 10-19 | fallback | 2 | `after_close` (retry) | — (the first reply was cut off at 2,048 tokens, `invalid_json`) |
| 7 | Tue 10-20 | actions | 2 | `after_close` | E06 +120, E05 +120, E07 +120 min |
| 8 | Wed 10-21 | actions | 1 | — | E06 +120, E05 +120 min |
| 9 | Thu 10-22 | actions | 1 | — | E07 +180, E06 +120, E05 +120 min |
| 10 | Fri 10-23 | actions | 2 | — (the first reply was not an object, `invalid_decision`) | E07 +180, E05 +120, E06 +120 min |
| 11 | Sat 10-24 | no action | 1 | — | — |
| 12 | Sun 10-25 | no action | 1 | — | — |
| 13 | Mon 10-26 | no action | 1 | — | — |

* **Applied:** 6 batches, 17 actions, all accepted; none refused by the
  harness.
* **Never submitted:** every refused proposal was caught by `validate`.
* **Fallbacks:** 1 (point 6).
* **Action types used:** operations actions only (`extend_shift`, the
  worklist policy). The model proposed no staffing action, so mixed
  batches were not exercised live; the mock tests exercise them. It asked
  for no details.

**Memory.** The memory grew from 555 characters at point 0 to 5,976 at
point 11 and stayed under its 6,000-character bound. From point 7 the
earliest decisions were folded into totals. At every point it showed:
* the earlier decisions;
* the proposals `validate` refused, with the simulator's message;
* what the action log showed was applied (for example "5 accepted, 0
  refused" with each extension's effect);
* the metric rows.

**What the memory did and did not do.** The `after_close` refusal was in
the memory, with its message, from point 1 on. The model still proposed
Dr. Blake's extension past the close again on first attempts at points 3,
6 and 7; `validate` refused it each time, and the retry dropped it except
at point 6. In its reasons the model cited the refusal on retries
("Dr. Blake's shift already ends at the scheduled close (20:00), so
extending it is invalid"). No `past_date` refusal occurred, but the model
proposed no allocation change, so that is not evidence that the earliest
date guidance worked. At point 13 the model called a Monday "Sunday";
the prompt said Mon 2026-10-26.

**Tokens and latency** (the server's counts, wall-clock per request;
`llm-usage.txt`):

| | Value |
|---|---|
| Requests | 20 (14 first attempts, 6 retries, 0 details) |
| Prompt tokens | 236,061 reported (first attempts 9,613–12,506, mean 11,607, memory included) |
| Completion tokens | 8,797 reported |
| Unknown or estimated usage | none |
| Latency | mean 4.3 s, median 4.0 s, max 11.6 s; 86.1 s of inference in all |
| Decisions | 13 valid (6 with actions, 7 no action), 5 with a refused proposal, 1 fallback |
| Cost | none configured |

V0.5a's first-attempt prompts on seed 40601 were 21,850–23,016 tokens. On
the same recorded observations the V0.5b prompt is 9,465–9,936 tokens
before any memory ([LLM_AGENT.md](../../LLM_AGENT.md) §4.1).

**Outcome against `noop`** (one seed, 14 days; not a claim):
* operating profit was $30 higher and net revenue $1,842 higher;
* there were 4 fewer lost orders and 8 more exams, TAT p90 was 58 min
  lower and there were 6 fewer breaches;
* overtime was 2,081 min more.

**Replay.** `replay.json` ran the recorded decisions through the agent's
replay mode, with no model contacted. At every decision point the agent
rebuilt its memory and prompt from the run's own observations and the
recorded decisions, and checked their SHA-256 against the recording.

| Check | Result |
|---|---|
| Recorded decisions, digests, memory and prompt | matched at every decision point |
| `decisions.json`, `session.json`, `result.json`, final digests | byte-identical to the live run's |
| Memory text (`memory.jsonl`) | identical to the live run's at all 14 points |
| Two replays | byte-identical manifests (`replay-manifest.json`) |
| linux/amd64, Python 3.13 (horo) | byte-identical manifest |
| `benchmark verify` | 24 of 24 |
| `benchmark reproduce` | 19 of 19 |

The replay's transcript differs from the live run's: replay does not
validate. That is why the manifests differ.
