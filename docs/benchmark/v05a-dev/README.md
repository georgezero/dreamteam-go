# V0.5a integration smoke test: v05a-dev

The V0.5a live run of the LLM agent ([LLM_AGENT.md](../../LLM_AGENT.md)).
It ran on the bundled sustainable scenario, seed 40601, for 7 days, with
decisions daily at 07:00 and no T90. Two runs: `noop`, the baseline, and
`llm`, the agent live with one local model.

**This is an integration smoke test, not a performance benchmark.** It
shows that a model can take part in a complete, replayable session. One
model, one seed and 7 days say nothing about decision quality, and the
report makes no claim.

## Files

* `config.json`, `llm-settings.json`: the configuration and the agent's
  settings. They were committed with the agent (b5ac92f) before any run on
  the seed.
  * The model, `Qwen3.6-35B-A3B-oQ4e-mtp`, was chosen for being available
    locally and quick, not for any result.
  * The server's address came from `DREAMTEAM_LLM_BASE_URL` and is not
    committed. No API key was needed.
* `manifest.json`, `report.json`, `report.txt`: the live run's manifest and
  report. The manifest records the agent's and the settings' SHA-256
  (`888d9be0…` and `67a2ca03…`, the committed files). It lists the
  agent's evidence files as nondeterministic.
* `llm-decisions.jsonl`: the agent's recorded decisions from the live run,
  its `policy-evidence/decisions.jsonl`, the replay input.
* `replay.json`, `replay-manifest.json`: the replay of those decisions
  without a model, and its manifest.

The full results, with the model exchanges (`policy-evidence/model.jsonl`,
760 KB), are kept outside the repository.

## Evidence (2026-10-27)

**Live run.** On a Raspberry Pi 4 (linux/arm64, Python 3.11), against a
local OpenAI-compatible server:
* both runs completed: `noop` in 12.8 s, `llm` in 62.9 s;
* every saved session replayed with every digest verified;
* `benchmark verify` passed 24 of 24 checks.

The agent used only `hello` (1), `digests` (7), `observe` (7), `actions`
(7) and `validate` (4). The gate denied nothing and no request erred. Its
standard error was empty.

| Point | Time | Outcome | Attempts | What happened |
|---|---|---|---|---|
| 0 | Tue 10-13 07:00 | no action | 1 | — |
| 1 | Wed 10-14 07:00 | no action | 1 | — |
| 2 | Thu 10-15 07:00 | no action | 2 | It proposed two reading-allocation changes effective today. `validate` refused them (`past_date`). On the retry the model chose no change. |
| 3 | Fri 10-16 07:00 | actions | 2 | It proposed extending two radiologists' shifts. `validate` refused one (`after_close`). On the retry it submitted one valid action: Dr. Avery (E07) +120 min. |
| 4 | Sat 10-17 07:00 | no action | 1 | — |
| 5 | Sun 10-18 07:00 | no action | 1 | — |
| 6 | Mon 10-19 07:00 | actions | 1 | Two shift extensions, E07 +120 min and E03 +60 min, valid at once. |

* **Applied:** 2 batches (3 actions); none refused by the harness.
* **Never submitted:** every refused proposal was caught by `validate`.
* **Fallbacks:** none.

**Tokens and latency** (the server's counts, wall-clock per request):

| Point | Attempt | Prompt tokens | Completion tokens | Latency |
|---|---|---|---|---|
| 0 | 1 | 22,046 | 66 | 5.3 s |
| 1 | 1 | 21,922 | 102 | 5.5 s |
| 2 | 1 | 22,545 | 290 | 7.0 s |
| 2 | 2 | 22,910 | 134 | 1.4 s |
| 3 | 1 | 23,016 | 174 | 6.7 s |
| 3 | 2 | 23,245 | 99 | 1.1 s |
| 4 | 1 | 22,159 | 110 | 6.1 s |
| 5 | 1 | 21,850 | 86 | 6.0 s |
| 6 | 1 | 22,391 | 164 | 6.7 s |
| **Total** | 9 calls | **202,084** | **1,225** | **45.7 s** (mean 5.1 s, max 7.0 s) |

Prompts were 67,000–70,000 characters, and no schema was left out. The
retries were quicker, probably because the server reused the shared prompt
prefix.

**Outcome against `noop`** (one seed, 7 days; not a claim):
* operating profit was $153 lower and net revenue $91 lower;
* there were 4 more exams, TAT p90 was 5 min lower and there were 2 fewer
  breaches;
* overtime was 307 min more, and one admitted patient was rescheduled
  because nobody could serve them safely that day (`noCoverage` 1, 0
  under `noop`).

**Replay.** `replay.json` ran the recorded decisions through the agent's
replay mode, with no model contacted.

| Check | Result |
|---|---|
| Recorded decisions and digests | matched at every decision point |
| `decisions.json`, `session.json`, `result.json`, final digests | byte-identical to the live run's |
| Two replays | byte-identical manifests (`replay-manifest.json`) |
| `benchmark verify` | 24 of 24 |
| `benchmark reproduce` | 19 of 19 |

The replay's transcript differs from the live run's: replay asks for
`digests` only, live also observes and validates. That is why the
manifests differ.
