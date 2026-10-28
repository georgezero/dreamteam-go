# DreamTeam LLM agent (V0.5a, V0.5b)

`agents/llm/llm_agent.py` is an agent that asks a language model how to
manage the practice. It is an external benchmark policy
([BENCHMARK.md](BENCHMARK.md) §4): `dreamteam benchmark run` starts it as a
separate process for each run, and it takes part through the policy
protocol (`dreamteam.benchmark-policy/v1`) like any other policy. It calls
the model through an OpenAI-compatible chat completions API, so local model
servers work without a provider-specific dependency. Standard library
Python only.

Status: V0.5b, development (0.5.0-dev.2). Not released. This is llm-agent
version 2: V0.5a's agent (version 1) with bounded memory, action guidance,
a targeted prompt, operations and staffing actions at one decision point,
and usage and cost reporting.

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
      "version": "2",
      "description": "the LLM agent, model …",
      "command": ["python3", "-I", "-B", "./agents/llm/llm_agent.py", "--settings", "./my-settings.json"],
      "timeoutSeconds": 1800
     }
    ],
   ```

   The harness records the SHA-256 of both files in the manifest. Set
   `timeoutSeconds` above `decisionBudgetSeconds` plus one request timeout
   and the time to observe (§3).

3. **Run it** from the repository root, with the environment set:

   ```
   export DREAMTEAM_LLM_BASE_URL=http://localhost:1234/v1   # your server
   export DREAMTEAM_LLM_MODEL=…                             # its model id
   export DREAMTEAM_LLM_API_KEY=…                           # only if it needs one
   dreamteam benchmark run my-config.json -out results
   dreamteam benchmark verify results
   cat results/runs/*-llm/policy-evidence/usage.txt
   ```

The smoke tests' files are complete examples:
[benchmark/v05b-dev/config.json](benchmark/v05b-dev/config.json) and
[benchmark/v05b-dev/llm-settings.json](benchmark/v05b-dev/llm-settings.json)
(V0.5b), and [benchmark/v05a-dev/](benchmark/v05a-dev/) (V0.5a, run with
version 1 of the agent, 3104e67).

**Credentials.** An API key is read from the environment variable that
`apiKeyEnv` names and sent only in the `Authorization` header. A settings
file that contains `apiKey` is refused, and so is a base URL with a user
name or password in it. No key appears in the agent's evidence, its
standard error or anything the harness writes (tested). The usage summary
names the model only, not the endpoint.

## 2. Settings (`dreamteam.llm-agent-settings/v1`)

A JSON object. Unknown fields are refused. The fields from `observation`
down were added in V0.5b; all are optional, so a V0.5a settings file
loads unchanged.

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
| `decisionBudgetSeconds` | 600 | 1–3600: no further request starts after this long at one decision point. |
| `responseFormat` | `json_object` | `json_object` asks for `response_format: {"type": "json_object"}`; `none` for servers that do not support it. |
| `maxPromptChars` | 200000 | 1000–2000000: the prompt's bound (§4). A details answer is cut to a quarter of it per item. |
| `systemPromptPath` | built in | A file replacing the built-in system prompt, relative to the settings file. Its SHA-256 is recorded. |
| `observation` | `summary` | `summary`: the observation summarized (§4.3); `compact`: the whole compact observation, as in V0.5a. |
| `catalog` | `guided` | `guided`: the action guidance only (§4.4); `full`: the guidance and the full catalog JSON. |
| `allowDetails` | true | Whether the model may ask once per decision point for full details (§5). |
| `memoryDecisions` | 6 | 0–50: recent decisions kept in full in the memory (§6). 0 turns the memory off (§6). |
| `memoryMetricPoints` | 6 | 0–30: metric rows kept. |
| `memoryMaxChars` | 6000 | 500–50000: the memory text's bound. |
| `pricing` | none | `{"currency": "USD", "inputPerMillion": n, "outputPerMillion": n}`: prices per million tokens, for the estimated cost (§9). |
| `description` | — | Free text. |

The command line: `--settings FILE` (live), or `--replay FILE` (repeatable;
replay, §8); `--evidence DIR` (default `evidence`, in the working
directory the harness gives the process).

## 3. A decision point (live)

At each decision turn the agent:

1. reads the clock and digests (`digests`);
2. reads the compact public observation (`observe` with `compact: true`,
   `recentLog: 0`) and the action catalog (`actions`);
3. updates its memory from the observation (§6);
4. builds the prompt (§4);
5. asks the model, and parses its reply strictly (§5); if the model asks
   for details, it answers once and asks again;
6. checks every action's type against the catalog, splits the actions
   into an operations batch and a staffing batch (§7) and validates each
   through the protocol (`validate`);
7. submits the batches if, and only if, every one is valid; otherwise it
   retries (§5) and in the end decides no change;
8. records the exchange, the decision and its memory (§10).

It uses `hello` (once), `digests`, `observe`, `actions` and `validate`:
read and check operations only, all through the harness's capability
gate. It never applies, advances, saves or asks for T90; the harness does.
It does not plan and has no tools. Its memory (§6) is its only state
between decision points.

## 4. The prompt (`llm-agent-prompt/v2`)

Two messages. The **system** message gives the role (operations, not
clinical decisions, in a synthetic practice), the objective (financial
sustainability with timely service: waits, lost demand, turnaround and
breaches, profit and cash; changes cost money and some have lead times;
doing nothing is valid and often right), the reply format with the
optional details request, and the rules: listed types, fields, targets and
values only; dates and times practice-local, within each action's earliest
and latest dates and the operating hours; operations and staffing actions
validated and submitted as two batches, each all-or-nothing; at most 64 of
each kind.

The **user** message, in this order:

1. **The time:** decision point, weekday, date, clock and day; the
   practice's phase, operating hours and today's scheduled close.
2. **Active problems:** public facts only, each shown when present: active
   alerts with their reasons; the reading backlog (unread, being read,
   urgent, overdue, oldest, breaches and TAT p90 today); lost demand;
   modalities whose first open slot is over 3 days ahead; today's
   no-coverage, missed, left and declined walk-ins; negative cash or days
   below zero; a negative operating profit month to date.
3. **Memory** (§6).
4. **State:** the observation summarized (§4.3), as compact JSON.
5. **Actions:** the action guidance (§4.4).
6. **Left out:** everything the prompt leaves out (§4.3), by path.

It holds no seed, no future event, no random state, no T90 result, no other
policy's results and nothing reserved or held out. The agent never asks
for T90. The prompt was written before any run on V0.5b's seed, from the
protocol's documented fields and V0.5a's validation messages, and has not
been tuned to outcomes.

### 4.1 Size

Measured on the same recorded observations: V0.5a's seven decision points
(sustainable, seed 40601), their observe and actions responses taken from
the V0.5a run's transcript. The V0.5a token counts are the server's from the
live run. The V0.5b counts are the same server's (the same model's
tokenizer), from a request with `max_tokens` 1 (system and user messages,
memory as at the start of a run).

| Decision point | V0.5a characters | V0.5a tokens | V0.5b characters | V0.5b tokens | Tokens saved |
|---|---|---|---|---|---|
| 0 | 69,043 | 22,046 | 28,652 | 9,735 | 55.8% |
| 1 | 68,608 | 21,922 | 28,708 | 9,790 | 55.3% |
| 2 | 70,061 | 22,545 | 28,810 | 9,845 | 56.3% |
| 3 | 71,128 | 23,016 | 28,839 | 9,875 | 57.1% |
| 4 | 69,454 | 22,159 | 28,606 | 9,641 | 56.5% |
| 5 | 68,801 | 21,850 | 28,331 | 9,465 | 56.7% |
| 6 | 69,760 | 22,391 | 29,185 | 9,936 | 55.6% |

About 59% fewer characters and 56% fewer tokens. A run's memory adds up to
`memoryMaxChars` (6,000 by default) as it fills. The unit tests check the
reduction on decision point 0 (`agents/llm/testdata/v05a-point0.json`).

### 4.2 What was cut, and why

In V0.5a, two thirds of the prompt was the catalog's JSON Schemas (the
`hire` and `set_roster` schemas repeat the roster's shape for 14 days,
about 28,000 characters), the roster targets (about 10,000) and the finance
section's histories and breakdowns (about 11,000). V0.5b states the same
content compactly and leaves out history and breakdowns, naming them.

### 4.3 The observation summary

The compact observation (the protocol's own projection) with these cut:

* **Left out entirely:** `finance.cashHistory`, `finance.accounts`,
  `finance.opening`, `finance.claims.aging`, each finance period's
  `byModality` and `fixedByItem`, `today.rooms`, `today.staff` and
  `today.demand.byModality`.
* **Fields left out:** each finance period's expense subcategories and
  gross figures (the totals stay: net revenue, labour, fixed, variable,
  operating expense and profit, margin, collections, disbursements, net
  cash flow, reports, denial rate); each worklist item's patient, exam
  name and acquisition time (study, modality, priority, state, age and
  overdue stay); each staff member's `startsAt` (in the shift),
  `severanceCents` (in the terminate targets) and `contractInferred`.
* **Lists cut:** the worklist's items to the first 5 (the counts, by
  priority and modality, stay), active alerts to 10, recent action results
  to 5, any other list to 20.

The prompt's LEFT OUT section names every path cut, and the protocol's own
compact omissions (open claims, journal, finalized reports, resolved
alerts) with their counts. The full compact observation is available with
`observation: compact`, and any path through details (§5). The complete
public observation stays available through the protocol (`observe`)
unchanged.

### 4.4 Action guidance

For the catalog as a whole: its batch rules, verbatim; how this agent
submits operations and staffing actions (§7); the operating hours,
today's scheduled close, the booking horizon, today's date and the time.

For **every** action type in the catalog, none hidden:

* its batch (operations or staffing) and summary;
* **fields:** its JSON Schema in one line: each field with its type,
  allowed values, range and description, optional fields marked `?`, and
  fields with the same schema listed together (`Mon|Tue|…`);
* **rules:** the catalog's constraints, verbatim;
* **timing:** every date and lead-time field of its targets (names with
  earliest or latest, or ending in Date, Day or Days), grouped by name, with
  dates shown as days from today. For example `earliestEffective
  2026-10-27 (in 14 days)` for reading allocations, `standardLeadDays
  14–28; premiumLeadDays 7–21` for hiring. Where the targets hold no such
  field, the prompt says that the catalog states no dates or lead times for
  the type and that `validate` decides, rather than guessing;
* **targets:** the current targets as compact JSON (staff with today's
  shift and cap, offers and headcount caps, editable rosters and
  allocations with earliest dates, plans). Rosters are shown as text, for
  example `A: Mon–Fri 07:45–16:15 @desk break 12:00–12:30; Sat,Sun off |
  B: same as A`, with the mapping to the JSON shape stated.

Everything shown is read from the catalog and the observation. No
simulator rule is added. The two V0.5a refusals show where this helps:

* `past_date`: the allocation's earliest effective date was in the
  catalog, but in a 23,000-token prompt. Now it is a short line with its
  distance from today.
* `after_close`: the operating hours and today's scheduled close are now
  stated, beside each employee's shift and cap. The catalog does not state
  the rule about extending past the close. When `validate` refuses it, the
  refusal stays in the memory (§6) for the rest of the run.

## 5. The decision

The model must reply with exactly one JSON object:

```json
{"reason": "one or two sentences", "actions": []}
```

* `reason`: a string, at most 2,000 characters.
* `actions`: a list of action objects in the protocol's format
  (`docs/AGENT_PROTOCOL.md` §4): each has a `type` from the catalog and
  follows that type's schema. Operations and staffing actions may be mixed
  (§7), at most 64 of each kind. An empty list is a valid decision: no
  change.
* Nothing else: another key, a non-object or trailing text is refused.

**Details.** Once per decision point (when `allowDetails` is true), the
model may instead reply `{"reason": …, "actions": [], "details": [names]}`
with 1–8 names. Each name is either an action type, which gets its full
catalog entry (JSON Schema, constraints and raw targets), or an
observation path. A path the summary cut comes from the compact
observation; a path the protocol's compact projection leaves out comes
from the full section (`observe` without `compact`). An unknown name is
answered as such. The agent then asks again. A details request is not a
failure and uses no retry; a second one at the same decision point is
refused as `invalid_decision`.

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
| `invalid_action: validate answered …` | The protocol refused a batch's arguments (for example an unknown field). |
| `refused: …` | `validate` refused an action (its number in the reply, code and message). |
| `prompt_too_large` | The prompt is over `maxPromptChars`; no request is made. |
| `budget` | `decisionBudgetSeconds` was spent; no further request. |

**Retries.** After a failure the agent asks again, up to `maxRetries`
times and within `decisionBudgetSeconds`. When the model did answer, the
retry adds its reply and one message saying what was wrong. A transport
failure resends the same request.

**Fallback.** When no attempt yields a valid decision, the agent decides
no change. It never submits a batch that `validate` has not accepted, and
never alters one. Nothing that fails changes the session: `validate`
changes nothing, and the harness applies only the submitted batches.

## 6. Memory (`llm-agent-memory/v1`)

The agent keeps a memory within a run. It lives in the process, and the
harness starts one process per run, so every run begins with an empty
memory (tested). It is derived only from:

* the public observation at each decision point;
* the action catalog at each decision point;
* the agent's own decision records: its reasons, submitted batches, and
  the `validate` refusals it was given.

It never holds hidden simulator state, future events, the seed or T90
results; the agent has none of them to give it (tested by handing it an
observation with extra fields, which never appear). Nothing is stored
outside the process: no files are read back, and there are no vector
stores, embeddings or external services.

**What it holds,** rendered as text in the prompt:

* **Recent decisions**, the last `memoryDecisions` in full. For each:
  - the decision point and time, the outcome and the reason (200
    characters at most);
  - proposals `validate` refused first, with code and message;
  - for a fallback, how many errors there were;
  - the submitted actions;
  - what became of them, as the next observation's action log shows it:
    accepted and refused counts and their messages.
* **Earlier decisions** as totals: decision points, outcomes, actions
  submitted, accepted and refused when applied, proposals refused by
  `validate`.
* **Validation refusals met in this run:** the most recent message for
  each action type and refusal code, at most 8. The model is not told the
  same rule twice, and does not learn from other runs.
* **Pending changes now:** recomputed at every decision point from the
  observation and catalog, at most 12:
  - roster and allocation versions marked pending, with their dates;
  - staff not yet active or leaving, with their dates;
  - contracts that can be cancelled;
  - approved extensions;
  - any teleradiology section.
* **Metrics** at the last `memoryMetricPoints` decision points:
  - cash;
  - operating profit month to date;
  - unread and overdue studies;
  - lost demand, exams done, overtime minutes and no-coverage events since
    the start.

  Successive rows show how the observed metrics changed.

**Bounds.** At most `memoryDecisions` entries, `memoryMetricPoints` rows, 8
refusals and 12 pending lines. The rendered text is at most
`memoryMaxChars`. Over the limit:
1. more recent decisions are folded into the totals;
2. then the oldest metric rows, refusals and pending lines are dropped;
3. as a last resort, the text is cut.

The size does not grow with the run's length. A 90-day mock run stays
within a 1,500-character limit, and a 120-point unit test within 6,000.

**Memory off.** With `memoryDecisions` 0 (since V0.5c, for its
memory-off control), the MEMORY section says the memory is off.
* **What it carries:** nothing from earlier decision points: no
  decisions, totals, refusals or metric rows, whatever
  `memoryMetricPoints` says.
* **What it keeps:** the pending changes, which come from the current
  observation and catalog.
* **What else changes:** nothing; the rest of the prompt is unchanged.
* **Before V0.5c:** 0 kept the earlier decisions as totals, with the
  refusals and metric rows.
* **Memory on:** unchanged, byte for byte; the V0.5b recording's replay
  checks it at every point.

## 7. Operations and staffing actions at one decision point

**The restriction.** A batch holding a V0.2 staffing action (`hire`,
`terminate`, `set_roster`, `set_reading_allocation`,
`contract_teleradiology`) may hold only staffing actions. Otherwise the
whole batch is refused, the first operations action as `mixed_batch`.

**What it is.**
* **A deliberate session invariant.** The simulator enforces it
  (`internal/sim/staffing.go`), as V0.2_DESIGN §5.7 "Atomic batches"
  specifies:
  - a staffing batch is atomic: every item is validated against the state
    plus the earlier items, the composed roster timelines are checked as a
    whole, and one refusal refuses all;
  - V0.1 batches keep their sequential, item-by-item semantics and exact
    messages.
  
  Saved sessions record batches under these rules and replay them under
  the same rules.
* **Not a transport limitation.** A benchmark decision is a list of up to
  64 batches, applied in order at the decision point
  ([BENCHMARK.md](BENCHMARK.md) §4).
* **Not a validation limitation.** `validate` checks any one batch with
  `apply`'s own rules.
* **Not an implementation gap.** One mixed atomic batch would need new
  combined semantics for two different validation models. That is a
  redesign of transaction atomicity, which V0.5b leaves alone.

**What V0.5b enables,** through the existing documented API with no change
of semantics:
* The model may put operations and staffing actions in one reply.
* The agent splits them into an operations batch and a staffing batch,
  each keeping the reply's order. It validates each one, and submits both
  at this decision point, operations first, only if both are valid.

**What remains.** The two batches are not one atomic unit. Each is
validated against the current state. The staffing batch is applied after
the operations batch and is not validated against its result. If the
operations batch changes something the staffing batch depends on (an
extension on a date a new roster also changes, say), the staffing batch
can still be refused when applied. The harness then records it as
refused, nothing of it is applied, and the next decision point's memory
shows it. The mock tests apply both batches at one decision point with
none refused. The restriction itself is unchanged and documented here and
in the protocol ([AGENT_PROTOCOL.md](AGENT_PROTOCOL.md) §3).

## 8. Replay

```
python3 llm_agent.py --replay decisions.jsonl [--replay other.jsonl …]
```

In replay mode the agent submits recorded decisions without contacting
any model (it needs no settings). At the first decision turn it picks the
recording whose first decision point and digests are this run's. Several
recordings can be given for a matrix, and two that begin alike but differ
are refused. At every turn the decision point and all five digests must
be the recorded ones. It then submits the recorded batches, byte for
byte.

**Memory and prompt.** For a version 2 recording, replay also rebuilds the
memory and the prompt at every decision point. It uses the same
observation and catalog (read again through the protocol, as live), the
recording's prompt settings and the recorded decisions. It then checks
their SHA-256 against the recording's `memorySha256` and `promptSha256`.
A change to the memory's or the prompt's formatting therefore cannot
silently change historical replay: a recording whose memory or prompt
differs is refused, naming the decision point and the format version.

**Versions.**

| Recording | Replay |
|---|---|
| llm-agent 1 (V0.5a, no memory fields) | Supported: batches submitted after the digest checks, as V0.5a did; nothing to rebuild. CI replays the V0.5a smoke test's recording. |
| llm-agent 2, `llm-agent-memory/v1` with `llm-agent-prompt/v2` | Supported, with the memory and prompt checks. |
| Any other memory format or prompt version | Refused before the run starts, with an explicit `version incompatible` error naming the recorded and supported versions. |
| A decision point recorded after a protocol error | Refused at that point; such errors are not expected. |

A mismatch, or recorded decision points left over at the end, stops the
agent with status 1. A recording refused before the run starts stops it
with status 2. Either way the harness classifies the run as
`policy_crash`.

**What replay shows.**
* Replayed under the same configuration, a recording gives the live run's
  outcome exactly. Its `decisions.json`, `session.json`, `result.json` and
  final digests are byte-identical to the live run's, and so is the
  rebuilt memory, point by point (tested, and shown by the smoke tests).
* A replay is deterministic: `benchmark reproduce` matches it, and CI
  compares its manifest across platforms.
* The live run is not deterministic: the same prompt can get a different
  reply, so a live rerun is not expected to reproduce. Replay separates
  model variability from simulator variability.

## 9. Usage and cost

`summary.json` (and the same as text in `usage.txt`) gives the run's model
usage:

* **The model:** its identifier as configured, and as the server named it
  if that differs.
* **Requests:** total, retries, details requests, and failed requests
  (`failedRequests`: those whose reply was not used, whether for a
  transport error, an invalid reply or a proposal `validate` refused).
* **Tokens, by how they are known:**

  | Kind | What it is |
  |---|---|
  | `reported` | The server's own counts. |
  | `estimated` | Requests the server answered without counts, estimated from characters (4 a token), labelled as estimates. |
  | `unknown` | Requests with no answer (timeout, connection, HTTP error): counted, no tokens claimed. |

  Prompt, completion and total tokens are given for each kind that has
  them.
* **Latency:** mean, median and maximum per request, and the total
  inference time (wall clock, including the server's queueing).
* **Decisions:** valid (with actions or no action), with a refused
  proposal, and fallbacks.
* **Estimated cost,** when `pricing` is set:
  - computed as reported and estimated tokens times the configured prices
    per million input and output tokens;
  - always labelled an estimate, not provider billing;
  - requests of unknown usage are counted beside it, not priced.

  Without prices there is no cost.

The summary names no endpoint and holds no credential. The raw exchanges
stay in `model.jsonl` (§10), outside the deterministic artifacts.

## 10. Evidence

The agent writes its evidence to `evidence/` in its working directory, and
the harness keeps it in the run's `policy-evidence/` directory
([BENCHMARK.md](BENCHMARK.md) §8). These files are listed in the manifest
as nondeterministic, not hashed. The harness's own files are separate and
deterministic given the decisions: transcript, decisions, session, result
and run record.

| File | Contents |
|---|---|
| `agent.json` | Schema `dreamteam.llm-agent-evidence/v2`; the agent, prompt and memory versions, the mode and the Python version. Live: the model, the endpoint (scheme, host, port and path only), temperature, limits, response format, prompt and memory settings, prices, the system prompt's source and SHA-256, and whether a key was set (never the key). Replay: each recording's name and SHA-256. |
| `model.jsonl` | One line per model request: decision point, attempt, turn (`first`, `details`, `retry`), model, endpoint, the request payload (all messages, model parameters), HTTP status, latency in ms, the raw response body (up to 1 MiB), the reply content, token usage as the server reported it, the served model, and the error. |
| `decisions.jsonl` | One line per decision point: point, time, day, digests, mode; agent, prompt and memory versions, the prompt settings; the memory's and the prompt's SHA-256 and length; what the prompt left out; attempts, retries, details requested, errors, refusals (attempt, action, type, code, message); outcome (`actions`, `no_action`, `fallback`), the model's reason, the proposed actions, the `validate` answers, and the submitted batches. **This file is the replay input.** |
| `memory.jsonl` | One line per decision point: the memory text as the prompt showed it, its SHA-256 and length (live and replay). |
| `summary.json`, `usage.txt` | Usage and estimated cost (§9). |

## 11. Testing

* `python3 -B -m unittest discover -s agents/llm` (26 tests):
  - settings: defaults, environment, V0.5a's file, refused fields,
    credentials and prices;
  - strict parsing, details requests, splitting into batches;
  - on the recorded V0.5a observation (`testdata/v05a-point0.json`): the
    prompt's size against V0.5a's; constraints, dates, lead times,
    operating hours and batch rules shown; every action type and every
    field present; details and the full catalog give the full entries;
    every cut path named;
  - memory updates, outcomes from the action log, refusals; bounds over
    120 points; reset; pending changes from public data only, with fields
    the observation does not have never appearing;
  - usage totals by kind, latency and decisions; cost estimates with and
    without prices;
  - replay matching and version refusals.
* `internal/bench/llm_agent_test.go` runs the agent as an external policy
  against a deterministic mock OpenAI-compatible server.
  - **Modes:** no action; a valid raise; a fenced reply; prose; an unknown
    action; a refused allocation; an extra field; failure then success; a
    slow server (request timeout); HTTP 500; operations and staffing
    actions together; a details request.
  - **Checks:**
    - every run completes, using only the permitted operations;
    - a failure gets exactly one retry, then no action, with nothing
      applied;
    - the memory is empty at the start of every run, carries each decision
      and its applied outcome to the next, shows pending allocation changes
      and refusals, and stays within its bound;
    - a details request returns the full catalog entry and a path from the
      full observation, without a retry;
    - mixed actions are applied as two batches at one decision point;
    - the usage totals match the mock's counts, and the cost matches the
      configured prices;
    - the evidence is kept and listed;
    - the stand-in API key reaches the server and appears in no result
      file.
  - **Replay:**
    - without the model, replay reproduces the live runs' decisions,
      sessions, results, digests and memory;
    - a recording of another run, one with tampered digests, one with a
      tampered memory hash, and one of another memory format are refused;
    - a 90-day run keeps the memory within a 1,500-character bound.
* CI runs the unit tests and replays both smoke tests' recorded decisions
  ([benchmark/v05a-dev/replay.json](benchmark/v05a-dev/replay.json), a
  version 1 recording, and
  [benchmark/v05b-dev/replay.json](benchmark/v05b-dev/replay.json), a
  version 2 one with memory checks) on linux/amd64, linux/arm64 and
  darwin/arm64, comparing the manifests.

## 12. Known limitations (V0.5b)

* **Bounded memory, no planning.** The memory is a short record of the
  agent's own decisions and what followed them. There is no plan, no goal
  tracking and no reasoning across runs, and older decisions survive only
  as totals.
* **Memory informs; it does not enforce.** A refusal in the memory does
  not stop the model proposing the same action again. In the V0.5b smoke
  test it did, and `validate` caught it each time.
* **Outcomes are read from the action log.** An applied decision's
  outcome is the change in the observation's accepted and refused counts
  and the results with that time. That is exact under the harness, where
  only the agent's batches are applied, but not where something else also
  applies actions.
* **Two batches, not one atomic unit** (§7). The staffing batch is
  validated without the operations batch applied.
* **Summaries can omit what matters.** The summary cuts breakdowns and
  history. The model must ask for details (once per decision point) or be
  run with `observation: compact`.
* **Live runs are not reproducible.** Even at temperature 0, servers may
  answer differently. Replay reproduces a recorded run; a rerun of the
  live configuration is a new sample.
* **Prompt size.** About 9,500–10,000 tokens per decision on the
  sustainable scenario, plus the memory. Token counts depend on the
  server's tokenizer.
* **Cost is an estimate.** Prices are configured by hand; nothing reads
  provider billing. Unknown-usage requests are not priced.
* **Trusted process.** Like any external policy, the agent is a trusted
  participant, not sandboxed ([BENCHMARK.md](BENCHMARK.md) §11). Its
  evidence is what it chose to write.
* **Development only.** One model, one seed, 14 days: an integration smoke
  test, not a measure of decision quality, and not a model comparison
  (V0.5c).
