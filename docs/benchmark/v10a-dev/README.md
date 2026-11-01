# V0.10a: inference repeatability and controlled comparison design

Status: **done** (0.10.0-dev.1), awaiting owner review. Declared at
41b16c9 before any request ([PLAN.md](PLAN.md), `protocol.json`,
`sample/`, `schedule.json`, [SERVER.md](SERVER.md)); run, analysed and
prototyped after. V0.10 is unreleased. The independent evidence review is
still PENDING.

V0.10a is a methodology and measurement milestone. It changes no
simulator mechanics, agent, prompt, setting or scoring, and runs no
multi-day live benchmark. It ranks no model.

**The answer, for these 25 requests on this server:**
* **The same request did not reliably get the same text.** Across all 15
  requests of each prompt, the raw text was identical for 2 of 10 Gemma
  prompts and 0 of 15 Qwen prompts.
* **Most text differences did not reach the decision.** Requests sent one
  at a time (11 per prompt) changed the decision (act, no action, refused,
  unparsable) for 3 of 25 prompts. Sent two at a time (4 per prompt), 8 of
  25 changed.
* **Sending requests at once made them less alike** at every level, and
  `seed` 0 changed nothing observable.
* **The prompt cache does not explain the variation.** Almost every
  request was served with its prompt nearly all cached, including those
  whose answers differed.
* **A shared prefix can hold the state and the proposal fixed.** The
  offline prototype branched both arms from 9 recorded V0.9c preview
  points: identical prefixes and proposals, the treatment arm equal to the
  recording, deterministic. It identifies nothing: its control replies are
  scripted.

## 1. Baseline

At clean `main` 7fa9557 (the V0.9.0 release, tag `v0.9.0`, object
a5d7e32), before any V0.10a change. Each aggregate is the SHA-256 of the
lines `<path> <sha256>\n`, in path order; a single file's own SHA-256 is
given instead.

| Set | Files | SHA-256 |
|---|---|---|
| Simulator: Go sources without tests and the version package; scenarios; go.mod | 129 | `cbbad4231e26de3a25e74e4e08ceb2476795af1d6e3b7ba3ec0e417ac76e8e68` |
| llm-agent 1 and 2: agents/llm/llm_agent.py | 1 | `5214e7bfb456075f38aca92a5c45a6db3141d4042fb1cd5822c8c1c65c286d87 (the file)` |
| llm-agent 3: agents/llm/llm_agent3.py | 1 | `ef2c17a06749d7c287cda918f77ff0e518d79475048e87fd3404848c2fb0015c (the file)` |
| llm-agent 4: agents/llm/llm_agent4.py | 1 | `4cfc6a6906556b0cda50e95fa39263089d56e848adcef9bd04342e7f6aff998f (the file)` |
| Model configurations: docs/benchmark/v05c-dev/llm-*.json | 4 | `b0a1ff334b58ba0c8769fecc23dcacaa0a1472e16020c0b3e89e3b4a3bc9ecd2` |
| Historical recordings, V0.5a-V0.9c (*-decisions.jsonl) | 114 | `336024d8ba6dd490c99b72c7e626b7f0792ad10d312767dba2deacf45b8f2107` |
| Evidence contracts: frozen-v1.json, frozen-parameter-v1.json, the analyzers and their checks | 25 | `15c04973c677828a9fc23dede93f90ffc59864ae6bdecb1edd489194e6774c7b` |
| Review package and page (docs/benchmark/v08b-dev/review/, v09a-dev/review/) | 10 | `b3826aac8cc147af67894a08cb173b0f231afdbaa84c683041164de7b157ddd6` |
| Benchmark suites (docs/benchmark/suite/) | 4 | `63c007ccedc023a501690aa14ca76d400a18be68024cfa658ec5804bd7265c65` |
| Golden references, without MIGRATIONS.json | 12 | `d84f80718f3ab0463cc3768cf89b5ba09885b654e674a92a87c82c6998a2b25f` |
| All development evidence (docs/benchmark/v0*) | 479 | `e109a4de7e4de385c412695b6c9d7827fd32a9bc82400976d5adbe188b4b732f` |
| Session fixtures (testdata/sessions/*.json) | 108 | `7f50a0ac535eb9b5f91667b2876bbae698265bde1a25bef10c04e976e780ded6` |
| Prompt version | | `llm-agent-prompt/v2`, memory `llm-agent-memory/v1` |
| Built-in system prompt | | `0efc36e21b592abc4acb4c8b78ee7634b39755580a0f5ac348a5be1f8a80c4e1` |
| Preview treatment | | `llm-agent-preview/v1`, `f8b7d787b59c36373d4a18135c1503ccf64e3dd956c749be42fdfb0dbe5c54fa` |

**After the V0.10a changes,** every set is unchanged except:
* the simulator aggregate, through the replay registration of 0.9.0 only
  (`internal/session/record.go`, `record_v1.go`; batch mode `v0.9.0`);
* the session fixtures: `mini-seed52-v090.actions.json` and
  `mini-seed52-v090.session.json` added, made from a clean 7fa9557
  worktree.

The agents, prompts, model configurations, recordings, evidence contracts,
suites and golden references are unchanged.

## 2. Prospective protocol

`v10a-repeatability-1` ([PLAN.md](PLAN.md), `protocol.json`
`077d3d56…f9f24b`), committed at 41b16c9 before any request. The 25
sampled requests were sent again, unchanged, under four conditions, per
model:

| Condition | Requests per prompt | In flight | Order |
|---|---|---|---|
| `interleaved` | 5, one per round | 1 | each round in a hash order of the prompts |
| `back_to_back` | 3 in a row | 1 | hash order |
| `concurrent` | 2 rounds of 2 identical requests at once | 2 | hash order |
| `seeded` | 3 in a row, with `"seed": 0` | 1 | hash order |

375 requests in the committed order (`schedule.json` `b3099c50…`).
Unavailable, and not invented: **cache cleared**, **cache disabled** and
**cold start** (admin-only, or disruptive to the shared server's other
users; not authorized). No amendment.

## 3. Prompt sample and hashes

`sample/manifest.json` (`fac1902a…`), `sample/prompts.jsonl`
(`d1815fd7…`), `dreamteam.repeatability-sample/v1`:
* hash-ranked from the 724 candidate attempts of the V0.8c and V0.9c
  recordings, 3 per model and category;
* every message of every request verified against the recorded SHA-256
  (system prompt, prompt, earlier outputs, retry, details and preview
  inputs), and the original response against the recorded output;
* checked again in CI (`prompt_sample.py --check`).

| Model | No action | Staffing | Financially consequential (≥ $10,000) | Validation refusal | Preview review | All |
|---|---|---|---|---|---|---|
| Gemma | 3 of 350 | 3 of 3 | 0 of 0 | 3 of 4 | 1 of 1 | 10 |
| Qwen | 3 of 233 | 3 of 25 | 3 of 3 | 3 of 97 | 3 of 8 | 15 |

p05 and p06 are byte-identical requests: 24 distinct requests. Requests
are 29,317 to 50,913 characters, 2 to 6 messages.

## 4. Model and server configuration

[SERVER.md](SERVER.md), `server-declared.json`, `live/server-before.json`,
`live/server-after.json`:
* **Server:** oMLX 0.7.0, OpenAI-compatible, shared, reached through the
  local tunnel used in V0.8c and V0.9c.
* **Models:** `Qwen3.6-35B-A3B-oQ4e-mtp`, `gemma-4-26B-A4B-it-oQ4e-mtp`;
  `max_model_len` 262,144.
* **Request:** `temperature` 0, `max_tokens` 4096, `response_format`
  `json_object`; nothing else (the V0.5c settings, unchanged).
* **Not observable without admin access:** weight hashes, the server's
  generation defaults (top_p, top_k, speculative decoding). The `mtp` in
  the model id is the id's own label.
* **Run:** 2026-11-01 14:23:05 to 14:47:58 UTC. 375 of 375 requests were
  answered with HTTP 200 and `finish_reason` `stop`, with no transport
  failure and no stop condition met.

## 5. Live repeatability results

[repeatability.json](repeatability.json),
[repeatability-tables.md](repeatability-tables.md). Prompts whose
responses were **all identical** within a condition:

| Model | Condition | Prompts | Requests | Raw | Parsed | Actions | Validation | Decision |
|---|---|---|---|---|---|---|---|---|
| Gemma | interleaved | 10 | 50 | 8 | 8 | 9 | 9 | 9 |
| Gemma | back_to_back | 10 | 30 | 9 | 9 | 9 | 9 | 9 |
| Gemma | concurrent | 10 | 40 | 3 | 3 | 8 | 8 | 8 |
| Gemma | seeded | 10 | 30 | 9 | 9 | 9 | 9 | 9 |
| Qwen | interleaved | 15 | 75 | 6 | 6 | 12 | 12 | 13 |
| Qwen | back_to_back | 15 | 45 | 6 | 6 | 12 | 12 | 13 |
| Qwen | concurrent | 15 | 60 | 0 | 0 | 6 | 7 | 9 |
| Qwen | seeded | 15 | 45 | 6 | 6 | 12 | 12 | 13 |

**Across all 15 requests of a prompt,** prompts with one value:

| Model | Prompts | Raw | Parsed | Actions | Validation | Decision |
|---|---|---|---|---|---|---|
| Gemma | 10 | 2 | 2 | 7 | 8 | 8 |
| Qwen | 15 | 0 | 0 | 6 | 7 | 9 |

**Against the original recorded response** (V0.8c or V0.9c), requests
equal to it:

| Model | Requests | Raw | Actions | Decision |
|---|---|---|---|---|
| Gemma | 150 | 110 | 135 | 143 |
| Qwen | 225 | 54 | 176 | 205 |

* **Parse failures:** 2 of 375 replies were refused by llm-agent 3's strict
  reading (`invalid_decision`: valid JSON, not a valid decision), both
  Qwen, both concurrent (p22, p25). Every reply parsed as one JSON
  object.
* **Parsed equals raw:** at every prompt the parsed reply had as many
  distinct values as the raw text. No difference was only formatting
  (whitespace, a fence, key order).
* **The first request of each prompt** in this study matched the original
  recorded response at the decision level for 25 of 25 prompts, and at the
  raw level for 11.

## 6. Action-level variation

The deepest level at which a prompt's 15 responses differed:

| Deepest difference | Gemma | Qwen |
|---|---|---|
| none: identical text | 2 | 0 |
| text only (reason wording; actions identical) | 5 | 6 |
| actions differ, validation identical | 1 | 1 |
| validation differs, decision identical | 0 | 2 |
| decision differs | 2 | 6 |
| all | 10 | 15 |

**Decision differences, by how the requests were sent:**

| Prompt | Category | Sequential (11 requests) | Concurrent (4) | Original |
|---|---|---|---|---|
| p06 Gemma | staffing | actions 11 | actions 3, no action 1 | actions |
| p07 Gemma | validation refusal | actions 4, refused 7 | actions 2, refused 2 | refused |
| p12 Qwen | no action | no action 11 | no action 3, refused 1 | no action |
| p15 Qwen | staffing | actions 8, refused 3 | actions 1, no action 3 | actions |
| p16 Qwen | staffing | actions 3, no action 2, refused 6 | actions 2, no action 1, refused 1 | actions |
| p17 Qwen | financially consequential | actions 11 | actions 3, refused 1 | actions |
| p22 Qwen | validation refusal | refused 11 | refused 3, unparsable 1 | refused |
| p25 Qwen | preview review | actions 11 | actions 3, unparsable 1 | actions |

* Requests sent one at a time changed the decision at 3 of 25 prompts
  (p07, p15, p16); sent two at a time, at 8 of 25.
* **Staffing proposals varied most:** p15 and p16 alternated between
  submitting, being refused and doing nothing. These are the decisions the
  V0.9c preview was about.
* **p05 and p06,** the same request bytes, gave 3 and 5 distinct texts and,
  at p06, one no-action decision among 15.
* No-action prompts mostly stayed no action (5 of 6 at every request).

## 7. Cache and concurrency findings

**Declared conditions:**
* **Concurrent** requests were the least repeatable at every level, for
  both models (§5).
* **Interleaved and back-to-back** (sequential, with other prompts between
  repeats or not) gave the same counts for Qwen, and nearly the same for
  Gemma. The order of sequential requests made no visible difference.
* **Seeded** (`"seed": 0`) gave the same counts as back-to-back, its
  unseeded counterpart. A fixed seed did not make the replies repeatable.

**Exploratory** (not declared in the plan; [repeatability-tables.md](repeatability-tables.md)):

| Model | Concurrent pairs with identical raw text | Sequential requests with cached tokens, equal to the prompt's most common text | Without cached tokens | Prompts whose seeded and back-to-back sequences are identical |
|---|---|---|---|---|
| Gemma | 7 of 20 | 97 of 104 | 5 of 6 | 9 of 10 |
| Qwen | 4 of 30 | 108 of 161 | 3 of 4 | 5 of 15 |

* Two identical requests sent at once usually got different text.
* **The cache did not decide it.** Almost every request was served with
  nearly its whole prompt cached (10 of 375 had none). Sequential requests
  served from the cache still differed from the prompt's most common text
  (Qwen 53 of 161).
* **No cold condition was achievable.** Even the first request of most
  prompts in this study found its prompt cached, from earlier use of the
  server. "Cold" is therefore not claimed for any request.
* **Latency** (live metadata): median 1.3–2.4 s for Gemma and 2.2–6.2 s
  for Qwen, longest when concurrent.

**Not established:** why the replies vary. Batched computation, the
speculative decoding the `mtp` label suggests, and server-side state are
all possible. None was controlled or observed.

## 8. Shared-prefix design

[DESIGN.md](DESIGN.md), `shared-prefix/v1`. Both arms replay the
recording's decisions before the branch point with the recording agent's
own checks, rebuild the point's prompt and conversation, and take the
recorded initial proposal as their shared first reply. Then:
* **control** continues as llm-agent 3 (validate);
* **treatment** continues as llm-agent 4 (preview, shown exactly, then
  validate).

Replies after the branch come from a declared source, each bound to the
message it answers. After the branch point a declared suffix follows:
here no action, so the simulator continues deterministically.

| | |
|---|---|
| **Held fixed (checked)** | the state at the branch point (digests); memory and prompt (SHA-256); the initial proposal (SHA-256); model, settings, agent rules; the suffix |
| **Different by design** | treatment only: one preview request (state unchanged, checked), one more message and one more model request |
| **Uncontrolled** | inference variation after the branch (§5–§7); the server's cache, load and other users; later decisions with a live suffix |
| **Identifiable, with repeated live draws** | the distribution of the decision at the branch point under each arm, for that state and proposal; under the fixed suffix, the run-level consequence of each decision |
| **Not identifiable** | an effect averaged over states or scenarios without a declared sample of branch points; effects on later decisions; anything from the offline prototype |

A causal reading would also need exchangeable inference noise between
arms, no interference through the server's cache, enough draws and a
declared selection of branch points (DESIGN.md §4). Sharing a starting
state is not enough.

## 9. Offline prototype

`analysis/repeatability/shared_prefix.py`, [prototype/](prototype/):
* 9 specs, one per V0.9c point at which a preview was shown;
* the treatment arm uses the recorded replies, each bound to its recorded
  input and output SHA-256;
* the control arm has one scripted no-action reply, used only when the
  proposal is refused;
* each arm runs as an external benchmark policy, 30 days, then no action.

[prototype-tables.md](prototype/prototype-tables.md):

| Check | Result |
|---|---|
| Prefix digests equal in both arms and equal to the recording | 9 of 9 |
| Initial proposal identical in both arms | 9 of 9 |
| Preview shown in the treatment arm | 9 of 9 (ok) |
| Treatment arm submits what the V0.9c recording submitted | 9 of 9 |
| Arms submit the same batches | 5 of 9 (the 4 kept proposals, and one point where both ended with nothing) |
| Arms' digests never diverge when the submissions are the same | 5 of 5 |
| Benchmark verify of the prototype runs | 41, 113 and 77 checks, 0 failed |

Where the proposal was refused, the control arm's scripted reply was no
action, so its decision is not model behaviour. **The run-level
differences in prototype-tables.md are not effects of the preview.** They
show only that the comparison is computed.

Tests (`test_shared_prefix.py`, 18) cover:
* identical state and proposal;
* control; treatment kept, revised or abandoned; an invalid preview;
* retries and fallback;
* missing evidence (truncated output, no staffing proposal, a reply for
  another message, no reply left, a changed prompt);
* agreement with llm-agent 3 and 4 given the same replies;
* determinism and the committed specs rebuilt.

## 10. Reproducibility

* **Historical recordings, frozen evidence, golden references, suites,
  agents, prompts and model configurations:** unchanged (§1).
* **Historical session replay:** every fixture, including
  `mini-seed52-v090` (0.9.0, batch mode `v0.9.0`).
* **The V0.8c and V0.9c replays,** on which the validation level is
  computed: 104 checks, 0 failed, for each of the 4 entries.
* **The analysis and the prototype,** each run twice: byte-identical.
  Linux/arm64 (the Pi) and linux/amd64 (horo) give the same files. CI
  regenerates them, compares each with the committed file and compares
  the platforms.
* **Live metadata** (`live/`: response bodies, latency, send times, server
  state) is kept apart. The analysis reads the responses and reports
  timings separately; nothing deterministic depends on them.
* No historical live evaluation was rerun.

## 11. Quality gates

| Gate | Where | Result |
|---|---|---|
| Focused tests: `analysis/repeatability` | Pi and CI | 32 tests, OK |
| Python tests: `agents/llm`, `analysis/evidence` | Pi and CI | 60 and 147 tests, OK |
| Go suite (`go test ./...`), golden files | CI, three platforms | run by CI on this commit |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64, Go 1.27.0 | exit 0; 10 packages ok, no failure |
| `go vet ./...`, `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64) | horo | all ok |
| Historical replays (V0.8c, V0.9c recordings) | Pi, horo and CI | 104 checks, 0 failed, each of 4 replays |
| Benchmark verify and reproduce; prototype runs verified | CI | run by CI on this commit; prototype runs 41, 113 and 77 checks, 0 failed |
| Analysis, specs, prototype and comparison regenerated and compared byte for byte | Pi, horo and CI | identical |
| Cross-platform CI (linux, macOS, Windows) | CI | run by CI on this commit |

Not run: no clinical suite, no historical live evaluation rerun, and no
local repeat of the gates CI runs (Go suite, golden files, verify and
reproduce), per the project's practice of running only the race and the
cross-builds locally.

## 12. Known limitations

* **25 requests, two models, one server, one afternoon.** The counts
  describe these requests, not the models in general or any other server.
* **Cache state was observed, not controlled.** Cache clearing, disabling
  and a cold start were unavailable. No request is called cold.
* **Server internals are unknown:** generation defaults, speculative
  decoding and weight files are visible only to the admin API.
* **Shared server:** other users' requests may have run at the same time;
  their effect cannot be separated.
* **The server version of V0.8c and V0.9c was not recorded,** so whether
  their setting was the same is not known.
* **Concurrency of 2 only,** and only identical requests in flight
  together.
* **The validation level** is the engine's answer at the recorded state.
  It is not what a live agent would have done next (a retry, for
  example).
* **The prototype's control replies are scripted** and its suffix is
  fixed. It demonstrates the method; it estimates nothing.
* **Exploratory findings** (§7) were not declared before the run.

## 13. Recommended V0.10b scope

**Shared-prefix decision interventions, live, small and declared.**
* **Branch points:** the 9 V0.9c preview points, or a declared hash-ranked
  sample of staffing proposals. Fixed before any draw.
* **Draws:** both arms at each point with live replies, at least 10 per
  arm, sent sequentially (§7: concurrency adds variation), with the arms
  interleaved in a declared random order.
* **Measured:** the decision distribution per arm and point, and the
  within-arm variation from Part A, so that a between-arm difference is
  judged against it.
* **Suffix:** no action at first, so that run-level consequences are the
  simulator's, deterministic, and stated as conditional on that suffix. A
  live suffix belongs to V0.10c.
* **Also:** a repeat of Part A's sequential condition at the start, to
  check the setting has not changed; the server version recorded with
  every run.
* **Not yet:** a multi-day live benchmark.

