# V0.9c treatment isolation record

What differs between the baseline (llm-agent 3) and the preview-enabled
condition (llm-agent 4), and what does not. Written with the plan
([PLAN.md](PLAN.md)), before any live run.

## 1. Kept identical

| Component | How it is kept identical | Checked by |
|---|---|---|
| Model identifier and settings | the same V0.5c settings file for both conditions of a model | `eval.json` |
| System prompt | built in, llm-agent 2's; version 4 has none of its own | `test_llm_agent4.py` `Base` |
| Prompt `llm-agent-prompt/v2` | built by llm-agent 3's own code: version 4 loads `llm_agent3.py` unchanged (SHA-256 `ef2c17a0…`) and calls it | `Base.test_base_is_llm_agent3_unchanged` |
| Memory `llm-agent-memory/v1` | llm-agent 3's `Memory`, the same settings; it reads the same record fields | `Base.test_without_staffing_it_is_version_3` |
| First message at a decision point | the same function of the observation and the agent's earlier decisions | `Base.test_first_message_is_version_3s`; `TestLLMAgent4Preview` (every first request equal to version 3's on the same practice) |
| Retries, details turn, decision budget | llm-agent 3's rules and settings (one retry, 900 s) | `PreviewStep.test_refused_final_action` |
| Batching and validation | llm-agent 3's: operations first, each batch all-or-nothing, the engine's `validate` | `PreviewStep.test_revise` |
| Fallback | no batch is submitted that was not validated | `PreviewStep.test_abandon`, record checks |
| Scenario, seed, economics, horizon, cadence | one suite run; each pair shares entry and seed | `eval.json` |
| Action schema and catalog | the engine's, unchanged | engine 0.9.0-dev.3 |

**Without a staffing proposal, version 4 is version 3.** When no parsed
proposal at a decision point has a staffing action, version 4 sends the
same messages, makes the same protocol requests, submits the same
decisions and builds the same memory, with `preview: null` in each
attempt (`Base.test_without_staffing_it_is_version_3`).

**With a kept proposal, the practice is the same.** Against the engine, a
model that keeps every previewed proposal submits version 3's decisions,
and the sessions are byte-identical (`TestLLMAgent4Preview`). The preview
itself changes no simulator state: the digests after it are recorded at
every preview (`digestsUnchanged`) and checked again in replay.

## 2. The treatment difference

At a decision point, version 4 previews the first parsed proposal that has
a staffing action, before validating it, shows the model the simulator's
answer and takes its reply as the decision (PLAN §2). The only text the
treatment adds is the review message (`llm-agent-preview/v1`, PLAN §3,
SHA-256 `f8b7d787…5c54fa`): it describes the preview, says nothing has been
submitted, and asks for the final decision in the existing format. It
gives no goal, advice or heuristic.

**The preview is the protocol's existing `preview`**, the game's
staff-screen preview, unchanged:
* each action's result (accepted, or refused with the simulator's code and
  message);
* the hires, terminations, roster and allocation changes, teleradiology,
  slots opened and closed, and displaced bookings (each listed);
* the guaranteed and forecast money lines, each with its timing and basis;
* the funding test (commitment, headroom, funded), where the economics has
  one.

It is shown as compact JSON, exactly as returned. Nothing is summarized,
reworded or added, and no forecast value is invented.

## 3. Unavoidable differences

| Difference | Why | Reported as |
|---|---|---|
| One more model request at each decision point where a preview is shown | the model must answer the preview | review requests per run |
| A longer conversation at that point: the proposal, then the review message (up to 200,000 characters) | the preview is shown whole; a termination's preview lists every displaced booking | review message characters (each record's `preview.message.chars`) |
| Latency at that point | the extra request | live evidence only (`model.jsonl`, `summary.json`), never in the record |
| The details turn is not offered after a preview | at most one details turn per point, as in version 3; the reply to the preview must be a decision | — |
| A refused action can show in the preview before validate | the preview includes each action's result, as the staff screen does; version 3 learns of a refusal only from validate's feedback on a retry | previews in which the simulator refused an action |
| Preview requests in the protocol transcript | the request itself | the policy's request count (the harness's `policyRequests`) |

**What the treatment is not:** a broader prompt redesign, a different
model setting, more retries, a longer budget, a new action, a change to the
preview, or any coaching.

## 4. Versions and hashes

| | |
|---|---|
| Baseline agent | llm-agent 3, `agents/llm/llm_agent3.py`, `ef2c17a06749d7c287cda918f77ff0e518d79475048e87fd3404848c2fb0015c` |
| Preview-enabled agent | llm-agent 4, `agents/llm/llm_agent4.py`, `4cfc6a6906556b0cda50e95fa39263089d56e848adcef9bd04342e7f6aff998f` |
| Treatment | `llm-agent-preview/v1`, `f8b7d787b59c36373d4a18135c1503ccf64e3dd956c749be42fdfb0dbe5c54fa` |
| Records | `dreamteam.llm-decision-record/v3` (baseline), `/v4` (preview-enabled) |
| Prompt, memory | `llm-agent-prompt/v2`, `llm-agent-memory/v1`, both conditions |

Each v4 record carries `treatmentVersion` and `treatmentSha256`, and
`agent.json` the SHA-256 of both agent files. Replay refuses a record of
another treatment text.
