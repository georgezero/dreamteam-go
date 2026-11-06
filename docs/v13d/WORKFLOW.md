# V0.13d workflow: `llm-agent-alternatives-workflow/v1`

**Code:** `agents/llm/llm_agent_alternatives.py` (the workflow);
`internal/alternatives/turn.go` and `ops.go` (the engine side);
`cmd/dreamteam/agent.go` (`-alternatives`). Identity: `WORKFLOW_VERSION`
and `WORKFLOW_SHA256`, the SHA-256 of the version, llm-agent 5's
validation-first identity, the presentation and arms identities, and the
seven rules below.

No earlier agent changes. llm-agent 5 is called, unchanged, as a module.

## 1. Steps, for one decision point and one declared arm

| # | Step | Whose code |
|---|---|---|
| 1 | Receive the initial proposal: the model's first reply | llm-agent 5 (`llm_agent5.ask`, workflow `validation-first`), unchanged |
| 2 | Validate it: every batch of the reply, as the agents batch it (operations, then staffing) | llm-agent 5, through the engine's `validate` |
| 2a | A refusal: the engine's results shown verbatim (`llm-agent-validation-feedback/v1`), one correction | llm-agent 5 |
| 3 | Eligibility: is the turn due? | the engine, `alternatives.turn`: rule v2 applies to the staffing batch, the engine accepts both batches, and the comparison has a typed view; one test for both arms |
| 4 | Construct the alternatives | the engine: `SelectV2` |
| 5 | Verify the comparison | the engine: `BuildAgent`, `Encode`, `DecodeAgentAt` (rebuilt from the decision-point session, byte for byte), then `BuildView` |
| 6 | Present the declared arm | the engine renders the arm's text (presentation v2); the workflow sends it verbatim as the next user message, after the reply it is about |
| 7 | Accept select, revise or abandon | the engine, `alternatives.decide`: reads the reply (`ParseTurnReply`); answers an inspect request with the details asked for, once; answers an unreadable reply with the retry text |
| 8 | Validate the final proposal | the engine: each batch to be submitted is validated at the decision point (`Decide`) |
| 9 | Submit only valid actions | the workflow returns the outcome's batches if all are valid and the digests are unchanged; otherwise nothing |
| 10 | Record the complete process | the workflow's record (§3) |

**When the turn is not due** (the rule does not apply, the proposal or the
operations batch is refused, or no view can be made), the workflow
returns validation-first's batches, unchanged, and records why. Both arms
behave identically there.

**When validation-first ends without a validated proposal** (`no_action`
or `fallback`), there is no turn, and validation-first's result stands.

## 2. The turn

* **Messages:** the turn's message follows the conversation as it stood
  after validation-first's last call: its messages, then that call's reply
  as the assistant message, then the turn as the user message.
* **Model replies:** each goes to `alternatives.decide` with the arm, the
  reply's actions (the same ones the turn was built from), the IDs already
  inspected and the number of inspect rounds so far. The engine rebuilds
  the turn from the session and those actions, so the decision is always
  classified against the comparison that was shown.
* **Inspect:** one round, of one to four IDs. The answer has the details of
  the listed IDs asked for, in the order asked, each once. In control the
  answer is the fixed "There are no details to inspect".
* **Limits:** `TURN_RETRIES` is 1. An unreadable reply, a transport
  failure, or a second inspect request uses it. When it is spent, the
  result is `no_decision` and nothing is submitted. The most calls the
  turn can make is 3 (the reply, the reply after inspect, one retry).
* **Strict reading:** the reply must be exactly one JSON object with only
  the foot's fields (`decision`, `id`, `actions`, `reason`, `cite`) or an
  inspect request (`inspect`, optionally `reason`). Anything else is
  unreadable. Nothing is guessed, and nothing is defaulted.

## 3. The record (`dreamteam.alternatives-decision-record/v1`)

* `validationFirst`: llm-agent 5's own record, whole
  (`dreamteam.llm-decision-record/v5`);
* `turn`: the engine's answer to `alternatives.turn`, whole, including the
  message, the inspect texts and `record` (the batches, the rule, the
  presentation and view identities, the comparison and view hashes, the
  message hash, the inspect hashes and the neutral mapping);
* `exchanges`: per model call after the turn, the hash of the message it
  answered, its status and error, the hash of its content, the engine's
  reading of it and the hash of the answer sent next;
* `inspected`, `outcome` (the engine's `Outcome`), `result`, `batches` (what
  was submitted), `digestsBeforeSubmit`, `errors`.

## 4. Operations batches (owner decision D10)

| Decision | Operations batch | Staffing batch |
|---|---|---|
| select (any listed ID, `no_action` included) | the reply's, unchanged | the selected alternative's (none for `no_action`) |
| revise | replaced: the revision's own | replaced: the revision's own |
| abandon | none | none |

* A revision whose operations batch equals the reply's is classified by its
  staffing batch: a listed batch is that alternative ("a listed batch is
  that alternative", V0.13b). Otherwise it is a revision of the whole reply.
* Each batch is validated alone, at the decision point, as the agents'
  batches always are. They are applied one after the other with the
  engine's `apply`. No cross-batch atomicity is assumed. If a batch is
  refused at final validation, nothing is submitted
  (`invalid_alternative`). If one is refused when applied, after the one
  before it was applied, that is a partial application, and the driver's
  execution check records it.

## 5. Outcome categories

| Result | Meaning |
|---|---|
| `retained` | original retained: the proposal submitted unchanged (selected, or revised to the same batch) |
| `selected` | alternative selected: another listed alternative (treatment) |
| `selected_no_action` | no action selected (treatment): no staffing action; the operations batch kept |
| `revised` | proposal revised: a batch different from every listed alternative |
| `abandoned` | proposal abandoned: nothing submitted |
| `invalid_alternative` | an unlisted or refused alternative, or a batch the engine refuses at final validation: nothing submitted |
| `no_decision` | no usable reply within the retries, or a contradictory one: nothing submitted |

Control can produce every result except `selected` and
`selected_no_action`, because it lists only the proposal. That difference
is part of the information the treatment adds.

**Flags:** `inspected` (an inspect request for a listed alternative other
than the proposal); `compared` (the reply's `cite` names items and values
exactly as shown, for two or more alternatives, at least one not chosen,
with a difference, at most three entries, fewer than the items shown, and
not every row shown for every alternative: copying the table is not a
comparison); `no_observable_comparison` (neither). The reason text is
recorded and never read.
