# V0.10a server configuration record

What is known about the inference setting, and how it is known. Taken
from the server's public endpoints only (`/health`, `/v1/models`,
`/openapi.json`), before any request of this study (`server-declared.json`)
and again before and after the run (`live/server-before.json`,
`live/server-after.json`). No admin endpoint was called.

| Property | Value | How known |
|---|---|---|
| Server | oMLX API, version 0.7.0 | `/openapi.json` `info` |
| API schema | SHA-256 of `/openapi.json` (sorted keys) `67525ae2d8656c68c6aeed8edba9e9d1ff1e8e47528c0eb2e4aba68bea1085ff` | `/openapi.json` |
| Models served | 17 in the pool, 2 loaded at the declaration; the default model is another one (`Qwen3.8-27B-oQ4e-mtp`) | `/health` |
| Qwen | `Qwen3.6-35B-A3B-oQ4e-mtp`, `max_model_len` 262,144 | `/v1/models` |
| Gemma | `gemma-4-26B-A4B-it-oQ4e-mtp`, `max_model_len` 262,144 | `/v1/models` |
| Quantization | `oQ4e`, with `mtp`: the model id's own label. Not verified | the model id |
| Weight files, hashes | unavailable: admin-only | — |
| Generation defaults (top_p, top_k, min_p, penalties, speculative decoding) | unavailable: admin-only (`generation_config`, model settings) | — |
| Request fields the server accepts | `temperature`, `top_p`, `top_k`, `min_p`, `seed`, `max_tokens`, `response_format`, `enable_thinking`, `specprefill` and others | `/openapi.json` (`ChatCompletionRequest`) |
| Seed | a public request field, `seed`; what it controls is not documented | `/openapi.json` |
| Prompt cache | the response reports `usage.prompt_tokens_details.cached_tokens`; the server has a hot cache and an SSD cache, whose clear controls are admin-only | responses; `/openapi.json` path names |
| Response | `id`, `object`, `created`, `model`, `choices` (`message.role`, `message.content`, `finish_reason`), `usage` with timings; no `system_fingerprint` and no reasoning field in the recorded V0.9c responses | a recorded V0.9c response |
| Other users | the server is shared; their load is not observable beyond `/health` | — |

**Model configurations** (the agents' settings, unchanged):
`docs/benchmark/v05c-dev/llm-qwen36-35b-a3b.json` and
`llm-gemma4-26b-a4b.json`: `temperature` 0, `maxOutputTokens` 4096,
`responseFormat` `json_object`, `requestTimeoutSeconds` 300.
