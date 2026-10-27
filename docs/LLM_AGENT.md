# DreamTeam LLM agent (V0.5a)

`agents/llm/llm_agent.py` is an agent that asks a language model how to
manage the practice. It is an external benchmark policy
([BENCHMARK.md](BENCHMARK.md) §4): `dreamteam benchmark run` starts it as a
separate process for each run, and it takes part through the policy
protocol (`dreamteam.benchmark-policy/v1`) like any other policy. It calls
the model through an OpenAI-compatible chat completions API, so local model
servers work without a provider-specific dependency. Standard library
Python only.

Status: V0.5a, development (0.5.0-dev.1). Not released.

```
dreamteam benchmark run CONFIG
        │  benchmark-policy/v1 over stdio (one process per run)
        ▼
agents/llm/llm_agent.py
        │  OpenAI-compatible HTTP API
        ▼
a model server (local or remote)
```

The simulator stays deterministic and knows nothing of the model. The
agent may be nondeterministic. What it submits, and what came of it, is
recorded by the harness as for any policy, and the model exchanges by the
agent itself.

## 1. Setup

Requirements: Python 3.9 or later (standard library only), a DreamTeam
build, and an OpenAI-compatible server offering `POST
{baseUrl}/chat/completions`, such as LM Studio, llama.cpp's server, vLLM,
Ollama's OpenAI endpoint or a hosted API.

1. **Write a settings file** (§2). [agents/llm/settings.example.json](../agents/llm/settings.example.json)
   takes the model and the address from the environment:

   ```json
   {
    "schema": "dreamteam.llm-agent-settings/v1",
    "modelEnv": "DREAMTEAM_LLM_MODEL",
    "baseUrlEnv": "DREAMTEAM_LLM_BASE_URL",
    "apiKeyEnv": "DREAMTEAM_LLM_API_KEY",
    "requestTimeoutSeconds": 120,
    "temperature": 0,
    "maxOutputTokens": 2048,
    "maxRetries": 1,
    "decisionBudgetSeconds": 600
   }
   ```

2. **Declare the agent in a benchmark configuration** as an external
   policy, with the settings file in its command:

   ```json
    "policies": ["noop", "llm"],
    "external": [
     {
      "id": "llm",
      "version": "1",
      "description": "the LLM agent, model …",
      "command": ["python3", "-I", "-B", "./agents/llm/llm_agent.py", "--settings", "./my-settings.json"],
      "timeoutSeconds": 1800
     }
    ],
   ```

   The harness records the SHA-256 of both files in the manifest. Set
   `timeoutSeconds` above `decisionBudgetSeconds` plus the time to observe
   (§3).

3. **Run it** from the repository root, with the environment set:

   ```
   export DREAMTEAM_LLM_BASE_URL=http://localhost:1234/v1   # your server
   export DREAMTEAM_LLM_MODEL=…                             # its model id
   export DREAMTEAM_LLM_API_KEY=…                           # only if it needs one
   dreamteam benchmark run my-config.json -out results
   dreamteam benchmark verify results
   ```

The V0.5a smoke test's files are a complete example:
[benchmark/v05a-dev/config.json](benchmark/v05a-dev/config.json) and
[benchmark/v05a-dev/llm-settings.json](benchmark/v05a-dev/llm-settings.json).

**Credentials.** An API key is read from the environment variable that
`apiKeyEnv` names and sent only in the `Authorization` header. A settings
file that contains `apiKey` is refused, and so is a base URL with a user
name or password in it. No key appears in the agent's evidence, its
standard error or anything the harness writes (tested).

## 2. Settings (`dreamteam.llm-agent-settings/v1`)

A JSON object. Unknown fields are refused.

| Field | Default | Rule |
|---|---|---|
| `schema` | — | `"dreamteam.llm-agent-settings/v1"` |
| `model` or `modelEnv` | — | Exactly one: the model identifier, or the name of an environment variable that holds it. |
| `baseUrl` or `baseUrlEnv` | — | Exactly one: the API base (up to `/v1`), or the name of an environment variable that holds it. http or https; no credentials. |
| `apiKeyEnv` | none | The environment variable holding the API key, if the server needs one. |
| `requestTimeoutSeconds` | 120 | 1–3600: one model request. |
| `temperature` | 0 | 0–2. |
| `maxOutputTokens` | 1024 | 1–65536 (`max_tokens`). |
| `maxRetries` | 1 | 0–3: further model requests at one decision point after a failed one. |
| `decisionBudgetSeconds` | 600 | 1–3600: no retry starts after this long at one decision point. |
| `responseFormat` | `json_object` | `json_object` asks for `response_format: {"type": "json_object"}`; `none` for servers that do not support it. |
| `maxPromptChars` | 200000 | 1000–2000000: the prompt's bound (§4). |
| `systemPromptPath` | built in | A file replacing the built-in system prompt, relative to the settings file. Its SHA-256 is recorded. |
| `description` | — | Free text. |

The command line: `--settings FILE` (live), or `--replay FILE` (repeatable;
replay, §6); `--evidence DIR` (default `evidence`, in the working
directory the harness gives the process).

## 3. A decision point (live)

At each decision turn the agent:

1. reads the clock and digests (`digests`);
2. reads the compact public observation (`observe` with `compact: true`,
   `recentLog: 0`);
3. reads the action catalog (`actions`);
4. builds the prompt (§4);
5. asks the model, and parses its reply strictly (§5);
6. checks every action's type against the catalog, then validates the
   batch through the protocol (`validate`);
7. submits it as one batch if, and only if, it is valid; otherwise it
   retries (§5) and in the end decides no change;
8. records the exchange and the decision (§7).

It uses `hello` (once), `digests`, `observe`, `actions` and `validate`:
read and check operations only, all through the harness's capability
gate. It never applies, advances, saves or asks for T90; the harness does.
It is stateless: no memory between decision points, no plan, no tool
calls, no further exchange with the model once a batch is valid.

## 4. The prompt (`llm-agent-prompt/v1`)

Two messages:

* **System:** the role (operations, not clinical decisions, in a
  synthetic practice), the objective (financial sustainability with timely
  service: waits, lost demand, turnaround and breaches, profit and cash;
  changes cost money and some have lead times; doing nothing is valid and
  often right), the reply format, and the batch rules (catalog types and
  schemas only, one atomic batch, V0.1 and staffing actions not mixed, at
  most 64 actions, `[]` for no change).
* **User:** the decision point and its time, then the compact observation
  and the action catalog (batch rules, and for each type its JSON Schema,
  constraints and current targets), each as compact JSON.

The observation is the protocol's own: the time, practice, rooms, staff,
desk, demand, flow, worklist, finance, teleradiology, today's metrics,
alerts and policies, with history listings left out and counted. It holds
no seed, no future event, no random state, no T90 result and no other
policy's results; the agent never asks for T90.

**Bound.** If the prompt is longer than `maxPromptChars`, the largest
action schemas are left out first (their constraints and targets stay,
and the record names them); if it is still too long, the agent decides
no change and records `prompt_too_large`. On the sustainable scenario the
prompt is 67,000–70,000 characters, about 22,000 tokens.

The prompt was written before any run and has not been tuned to
outcomes.

## 5. The decision

The model must reply with exactly one JSON object:

```json
{"reason": "one or two sentences", "actions": []}
```

* `reason`: a string, at most 2,000 characters.
* `actions`: a list of at most 64 action objects in the protocol's
  format (`docs/AGENT_PROTOCOL.md` §4): each has a `type` from the catalog
  and follows that type's schema. An empty list is a valid decision: no
  change.
* Nothing else: another key, a non-object or trailing text is refused.

Two wrappings are tolerated, and only these: one leading
`<think>…</think>` block, and one Markdown code fence around the whole
reply.

**What can go wrong**, each recorded with a stable code:

| Code | When |
|---|---|
| `timeout`, `connection`, `http_status` | The request failed. |
| `response_not_json`, `response_shape`, `empty_content` | The answer is not a chat completion with content. |
| `invalid_json`, `invalid_decision`, `invalid_action` | The reply is not a decision as specified. |
| `unknown_action` | An action type that is not in the catalog. |
| `invalid_action: validate answered …` | The protocol refused the batch's arguments (for example an unknown field). |
| `refused: …` | `validate` refused an action (its code and message). |

**Retries.** After a failure the agent asks again, up to `maxRetries`
times and within `decisionBudgetSeconds`. When the model did answer, the
retry adds its reply and one message saying what was wrong. A transport
failure resends the same request.

**Fallback.** When no attempt yields a valid decision, the agent decides
no change. It never submits a batch that `validate` has not accepted, and
never alters one. Nothing that fails changes the session: `validate`
changes nothing, and the harness applies only the submitted batch.

## 6. Replay

```
python3 llm_agent.py --replay decisions.jsonl [--replay other.jsonl …]
```

In replay mode the agent submits recorded decisions without contacting
any model (it needs no settings). At the first decision turn it picks the
recording whose first decision point and digests are this run's. Several
recordings can be given for a matrix, and two that begin alike but differ
are refused. At every turn the decision point and all five digests must
be the recorded ones. It then submits the recorded batches, byte for
byte. A mismatch, or recorded decision points left over at the end, stops
the agent with status 1, and the harness classifies the run as
`policy_crash`.

Replayed under the same configuration, a recording gives the live run's
outcome exactly: its `decisions.json`, `session.json`, `result.json` and
final digests are byte-identical to the live run's (tested, and shown by
the smoke test). A replay is deterministic: `benchmark reproduce` matches
it, and CI compares its manifest across platforms. The live run is not
deterministic: the same prompt can get a different reply, so a live
rerun is not expected to reproduce. Replay separates model variability
from simulator variability.

## 7. Evidence

The agent writes its evidence to `evidence/` in its working directory, and
the harness keeps it in the run's `policy-evidence/` directory (V0.5a,
[BENCHMARK.md](BENCHMARK.md) §8). These files are listed in the manifest
as nondeterministic, not hashed. The harness's own files are separate and
deterministic given the decisions: transcript, decisions, session, result
and run record.

| File | Contents |
|---|---|
| `agent.json` | Schema `dreamteam.llm-agent-evidence/v1`; the agent and prompt versions, the mode and the Python version. Live: the model, the endpoint (scheme, host, port and path only), temperature, limits, response format, the system prompt's source and SHA-256, and whether a key was set (never the key). Replay: each recording's name and SHA-256. |
| `model.jsonl` | One line per model request: decision point, attempt, model, endpoint, the request payload (system and user messages, model parameters), HTTP status, latency in ms, the raw response body (up to 1 MiB), the reply content, token usage as the server reported it, and the error. |
| `decisions.jsonl` | One line per decision point: point, time, day, digests, mode, attempts, errors, outcome (`actions`, `no_action`, `fallback`; or the recorded one in replay), the model's reason, the proposed actions, the `validate` answer, and the submitted batches. **This file is the replay input.** |
| `summary.json` | Decision points, outcomes, model calls, failed calls, retries, prompt, completion and total tokens (when reported), and latency (total, mean, max). |

Token counts are the server's; a server that reports none leaves them
null. Latency is wall-clock time per request.

## 8. Testing

* `python3 -B -m unittest discover -s agents/llm`: settings (defaults,
  environment, refused fields and credentials), endpoint sanitizing,
  strict decision parsing, the prompt bound and replay matching.
* `internal/bench/llm_agent_test.go` runs the agent as an external policy
  against a deterministic mock OpenAI-compatible server. Its modes are:
  no action; a valid raise; a fenced reply; prose; an unknown action; a
  refused allocation; an extra field; failure then success; a slow server
  (request timeout); and HTTP 500. It checks:
  * every run completes, using only the permitted operations;
  * a failure gets exactly one retry, then no action, with nothing
    applied;
  * the evidence is kept and listed;
  * the stand-in API key reaches the server and appears in no result
    file;
  * a replay without the model reproduces the live runs' decisions,
    sessions, results and digests;
  * a recording of another run, or a tampered one, is refused.
* CI runs the unit tests and replays the smoke test's recorded decisions
  ([benchmark/v05a-dev/replay.json](benchmark/v05a-dev/replay.json)) on
  linux/amd64, linux/arm64 and darwin/arm64, comparing the manifests.

## 9. Known limitations (V0.5a)

* **Stateless.** No memory across decision points and no planning: each
  decision sees only the current observation and catalog, so the agent
  cannot follow up on its own earlier changes except through what the
  observation shows.
* **One batch per decision point.** V0.1 actions (`extend_shift`, the
  policies) and staffing actions cannot share a batch, so one decision
  cannot do both.
* **Live runs are not reproducible.** Even at temperature 0, servers may
  answer differently. Replay reproduces a recorded run; a rerun of the
  live configuration is a new sample.
* **Prompt size.** About 22,000 tokens per decision on the sustainable
  scenario. A model with a smaller context needs a lower `maxPromptChars`,
  which leaves schemas out.
* **Trusted process.** Like any external policy, the agent is a trusted
  participant, not sandboxed ([BENCHMARK.md](BENCHMARK.md) §11). Its
  evidence is what it chose to write.
* **No cost accounting** beyond the server's token counts and latency.
* **Development only.** One model, one seed, 7 days: an integration smoke
  test, not a measure of decision quality.
