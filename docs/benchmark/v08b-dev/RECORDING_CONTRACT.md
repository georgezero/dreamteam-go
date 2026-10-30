# Proposed recording contract for V0.8c (not deployed)

Status: a design, proposed in V0.8b for owner review. Nothing is deployed
or changed:
* not the agent (llm-agent 2, `llm-agent-prompt/v2`, `llm-agent-memory/v1`);
* not its recording format;
* not the agent protocol;
* not the replay.

If approved, it would ship with a new agent version, so that every
existing recording keeps replaying as it does now.

## 1. Why

V0.8a and V0.8b measured what the llm-agent 2 recording lets an analysis
see.

**What it records.** At each decision point, only the final parsed reply's
reason and actions. Its `refusals` keep each earlier attempt's refused
actions by type, code and message, but not their payload. Its `errors`
keep one line per failed attempt.

**What that costs:**
* **Refusal responses go unclassified.** Of the 559 historical refusals,
  452 cannot be classified by exact action identity only because the
  refused attempt's actions were not recorded, although the response was
  (`refusal-response-tables.md`). Read in this format, the funding probes
  lose 3 of their 7 classifications. Read with the attempt-level evidence
  the probes write, they lose none.
* **Earlier attempts' reasons are not recorded at all,** so a statement in
  attempt 1 cannot be compared with attempt 1's actions.
* **The model exchanges** (`model.jsonl`) hold everything, but they stay
  outside the repository with the server's details. They are live
  inference records, not replay inputs.

## 2. Principles

1. **Replay inputs and live metadata stay apart:**
   * *Replay inputs:* what the replay needs and checks (decisions,
     digests, prompt and memory fingerprints). They are deterministic and
     committed.
   * *Live metadata:* latency, tokens, retries' timing and the served
     model name. It is kept beside the replay inputs, never used by
     replay, and never compared across platforms.
2. **Bounded:** every text is cut to a stated limit and every list to a
   stated length, and the record says when anything was cut.
3. **No secrets or infrastructure:** no endpoint, host name, port, API
   key, credential, server configuration, environment variable value or
   file system path outside the run directory.
4. **Structured, not transcribed:** parsed fields, not raw model text.
   The raw response stays in the live, uncommitted `model.jsonl` as now.
5. **Stable order:** attempts numbered 1, 2, … in the order they were
   made, and actions numbered as in the reply.
6. **A recording says how complete it is.**

## 3. The record: `dreamteam.llm-decision-record/v3`

One JSON object per decision point, one per line. Replay inputs:

| Field | Contents |
|---|---|
| `schema` | `dreamteam.llm-decision-record/v3` |
| `point`, `time`, `day`, `digests` | as now: the replay's identity checks |
| `agentVersion`, `promptVersion`, `memoryFormat`, `promptSettings` | as now |
| `promptSha256`, `promptChars`, `memorySha256`, `memoryChars` | as now: fingerprints of the first turn's prompt and memory. The replay rebuilds both and checks them. |
| `attempts` | the list below, in order |
| `final` | `{attempt, outcome}`: the attempt whose decision was used (or null for a fallback), and `actions`, `no_action` or `fallback` |
| `batches` | as now: what was submitted, the replayed decision |
| `completeness` | §5 |

**Each attempt** (`attempts[i]`):

| Field | Contents |
|---|---|
| `attempt` | 1, 2, … |
| `turn` | `first`, `details` or `retry` |
| `inputSha256` | SHA-256 of the message this attempt answered: the prompt, a details answer or a retry message (§4) |
| `feedback` | for a retry: the error the agent sent back, as now in `errors` (bounded, 500 characters) |
| `parse` | `ok`, or the error code (`invalid_json`, `invalid_decision`, …) with its bounded message |
| `reason` | the attempt's own reason (bounded, 2,000 characters, as the agent already limits it), or null |
| `actions` | the attempt's own actions, as parsed (each bounded by the action schema), or null |
| `detailsRequested` | the names asked for, if this turn asked for details |
| `validation` | one entry per batch, in order, each with every action's result: `ok`, `code`, bounded `message`, and the action as validated |
| `refusals` | the attempt's refused actions: `{action, type, code, message}` |
| `protocolError` | if validate answered with a protocol error: its code and bounded message |

**Live metadata,** in the run's policy evidence and unhashed, as now,
keyed by `point` and `attempt`: latency, token usage, served model, HTTP
status, and whether a response was cut.

## 4. Inputs of later attempts

A retry or details turn answers a message the agent built:
* the retry instruction with the previous error;
* or the details text.

To keep replay deterministic, the record stores that message's SHA-256
(`inputSha256`), and the agent rebuilds the message from the record and
the observation in replay:
* the retry text from `feedback`;
* the details text from the observation.

Replay checks the hash, as it checks the first prompt today. The full
conversation is the first prompt plus each attempt's reply and input, so
it can be rebuilt without storing any of it twice.

## 5. Completeness

| Field | Meaning |
|---|---|
| `attemptsRecorded` | every attempt made is in `attempts` (true unless a budget or crash cut the point short; then `false` with the reason) |
| `reasonsRecorded` | the number of attempts with a reason |
| `actionsRecorded` | the number of attempts with parsed actions |
| `inputsVerifiable` | every `inputSha256` can be rebuilt in replay |
| `truncated` | the fields cut to their bound, if any |

An analysis reads `completeness` instead of inferring gaps, as V0.8a had
to.

## 6. Compatibility

* **A new agent version would write v3:** llm-agent 3, with the same
  prompt and memory formats if unchanged. llm-agent 2 recordings keep
  replaying through llm-agent 2's replay, unchanged.
* **The replay of v3:** it uses `final` and `batches` exactly as now, and
  checks the extra fingerprints. Recorded decisions stay what is replayed.
  Nothing in the record is needed to decide anything new.
* **The analyses:**
  * `dreamteam.decision-evidence/v1` reads the final reply (`final` →
    `attempts[final.attempt - 1]`).
  * `dreamteam.refusal-response/v1` reads `attempts` as it reads the
    probes' attempt-level evidence today.
  * A rule version 2 could check each attempt's reason against that
    attempt's actions. That needs owner approval.
* **The agent protocol:** unchanged. The record is the agent's own
  evidence.

## 7. What V0.8c would still not know

* Why a model chose anything: the record holds what it wrote and what it
  did.
* Anything about attempts whose reply could not be parsed beyond the
  parse error. The raw text stays in live, uncommitted evidence.
