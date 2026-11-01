# V0.10a inference repeatability: prospective protocol

`v10a-repeatability-1` (`protocol.json`, SHA-256
`077d3d56328c07921d869ba6b12b92c99f09432611650d766f0a5bc63df9f24b`).
Committed before any request of this study was sent. A change after that
commit is an amendment, recorded at the end with its reason; none is
planned.

**The question.** When the same model on the same server is sent the same
request again, how often is the answer the same? At which level does a
difference appear: the text, the parsed reply, the proposed actions, the
engine's validation of them, or the decision (act or not)?

This is a measurement of the inference setting V0.8c and V0.9c used. It
is not a benchmark, compares no agents, ranks no models and changes no
agent, prompt, setting or simulator behaviour.

## 1. What is held fixed

| Item | Value | Source |
|---|---|---|
| Models | `Qwen3.6-35B-A3B-oQ4e-mtp`, `gemma-4-26B-A4B-it-oQ4e-mtp` | the V0.5c settings files (`docs/benchmark/v05c-dev/llm-*.json`), unchanged |
| Weights and quantization | identified by the server's model id only: `oQ4e` quantization with `mtp` (multi-token prediction). Weight file hashes are not exposed by the public API: **unavailable** | `server-declared.json` |
| Endpoint implementation | oMLX ("LLM inference, optimized for your Mac"), OpenAI-compatible chat completions, server version 0.7.0, reached through the same local tunnel as V0.8c and V0.9c. The host and server version of those runs were not recorded, so they are not known to be the same | `server-declared.json` (`serverVersion`, `openapiSha256`) |
| Sampling | `temperature` 0; nothing else sent. Every other sampling setting (top_p, top_k, min_p, penalties, speculative decoding) is the server's model default, which only the admin API shows: **not observed** | the recorded request |
| Context | the server reports `max_model_len` 262,144 for both models; the largest sampled request is 50,913 characters | `server-declared.json`, `sample/manifest.json` |
| Output-token limit | `max_tokens` 4096 | the recorded request |
| Response format | `response_format` `{"type": "json_object"}` | the recorded request |
| Request bytes | the recorded request, serialized exactly as the agent serialized it (`requestSha256` of each prompt); the seeded condition alone appends `"seed": 0` | `sample/prompts.jsonl` |
| Timeout, retries | 300 s per request (the agent's), no retry | `protocol.json` |

The endpoint's address is read from `DREAMTEAM_LLM_BASE_URL` and is never
recorded.

## 2. The prompt sample

`sample/` (`dreamteam.repeatability-sample/v1`, made by
`analysis/repeatability/prompt_sample.py`):
* **Source:** the committed V0.8c and V0.9c recordings, the only ones whose
  live requests survive in full (the live runs' `model.jsonl`, kept
  locally, never committed).
* **Categories,** each recorded attempt the model answered, in this order:
  `preview_review`, `validation_refusal`, `financially_consequential` (a
  commitment of at least $10,000 in financial evidence v1),
  `staffing_proposal`, `no_action`. Anything else is not sampled.
* **Selection:** per model and category, the candidates ranked by
  SHA-256 of `dreamteam-v10a-sample/1|<recording>|<attempt id>`, the first
  three taken. It uses no response of this study and was not chosen by
  hand.
* **Result:** 25 requests from 724 candidates:

  | Model | No action | Staffing | Financially consequential | Validation refusal | Preview review |
  |---|---|---|---|---|---|
  | Gemma | 3 of 350 | 3 of 3 | 0 of 0 | 3 of 4 | 1 of 1 |
  | Qwen | 3 of 233 | 3 of 25 | 3 of 3 | 3 of 97 | 3 of 8 |

  p05 and p06 are the same request bytes: Gemma's identical prompt at the
  same point of the baseline and preview runs of staffing shortage, seed
  42001. Both are kept, as the rule selected them, so 24 distinct
  requests.
* **Verification:** every message of every request is checked against the
  committed recording: the system prompt against the built-in one
  (`0efc36e2…`), the first user message against `promptSha256`, each
  assistant message against the earlier attempt's output SHA-256, each
  later user message (retry, details, preview review) against that
  attempt's input SHA-256. The original response is checked against the
  recorded output's SHA-256. Nothing is rebuilt by guessing. CI repeats the
  check (`prompt_sample.py --check`).

## 3. Conditions

Per model, in this order (Gemma, then Qwen), each condition a block:

| Condition | Requests per prompt | In flight | Order |
|---|---|---|---|
| `interleaved` | 5 (one per round, 5 rounds) | 1 | each round in the order of SHA-256(`dreamteam-v10a-order/1\|interleaved\|<round>\|<prompt>`) |
| `back_to_back` | 3, one right after another | 1 | prompts by SHA-256(`…\|back_to_back\|<prompt>`) |
| `concurrent` | 4: 2 rounds of 2 identical requests sent at once | 2 | prompts by SHA-256(`…\|concurrent\|<prompt>`) |
| `seeded` | 3, one after another, with `"seed": 0` | 1 | prompts by SHA-256(`…\|seeded\|<prompt>`) |

375 requests in all: Gemma 150, Qwen 225. The exact order is
`schedule.json` (SHA-256
`b3099c50b1159b75f2e660a243f165431d547536f88ff834b1c970d001792cb6`),
committed with this plan.

**Unavailable, and not invented:**
* **cache cleared:** the server's cache controls (hot cache, SSD cache) are
  admin-only, need a key this study does not have, and would affect the
  server's other users. Not authorized.
* **cache disabled:** there is no per-request cache switch in the public
  request schema, and server settings are admin-only.
* **cold start:** it would need a model unload or a server restart, which
  affects other users. Not authorized.

**Observed, not controlled:**
* the cached prompt tokens the server reports for each request
  (`usage.prompt_tokens_details.cached_tokens`);
* the server's health and loaded model count before each block (`/health`);
* other users' requests, which may share the server at any time.

The first request of each prompt in this study is reported as such, with
its cached tokens. It is not called cold.

## 4. Comparison levels

Each response is compared, at each level, with the prompt's first
response in the same condition and with the original recorded response.

| Level | What is compared |
|---|---|
| A. Raw | the message content's bytes, exactly |
| B. Parsed | the content parsed as llm-agent 3 parses it (wrapping stripped, one JSON object), serialized canonically; an unparsable reply is its own value |
| C. Actions | the parsed actions (or the details request), canonical JSON |
| D. Validation | each distinct action list split into llm-agent 3's batches and validated by the engine at the recorded decision point, on a replay of the recording; the outcome is each batch's valid flag and refusal codes |
| E. Decision | `no_action`, `actions` (every batch valid), `refused`, `details` or `unparsable` |

**Measures**, each with its denominator, per model, category and
condition:
* prompts whose responses are all identical, at each level;
* the number of distinct values per prompt, at each level;
* requests equal to the condition's first response, and to the original
  recorded response;
* prompt, completion and cached tokens; latency; finish reasons; parse
  failures; transport failures.

Whether a difference stays textual or reaches the actions, the validation
or the decision is reported per prompt. No model is ranked, no rate is
generalized beyond these 25 requests, and nothing is called deterministic
or nondeterministic beyond what was observed.

## 5. Running

* **Client:** `analysis/repeatability/repeat_run.py` on the Pi, under
  `systemd-run --user --scope -p MemoryMax=1G nice -n 10`, at most 2
  requests in flight, one run only.
* **Stop:** 5 consecutive transport failures, 4 hours of wall clock, or a
  `/health` answer other than healthy before a block. A stopped run is
  reported as stopped, with every request it sent. No rerun, no
  replacement request, no dropped response.
* **Recorded** (`live/`): every request's response body, content,
  finish reason, usage, served model, latency and send time; the server
  state before and after; the run's start, end and counts. The recorded
  live metadata (times, latency) is kept apart from the deterministic
  analysis, which reads the responses only.
* **Shared service:** no admin call, no model load or unload, no cache
  clear, no restart.

## 6. Analysis

`analysis/repeatability/repeat_analysis.py`, written after this plan,
reads `live/requests.jsonl`, the sample and the replays CI already makes.
Its output (`repeatability.json`, `repeatability-tables.md`) is a function
of those files, and CI regenerates it and compares.

## Amendments

None.
