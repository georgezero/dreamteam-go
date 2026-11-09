# V0.13l evidence binding inventory

Schema: `dreamteam.evidence-bindings/2` (supersedes `dreamteam.evidence-bindings/1`, which is unchanged). Lifecycle `dreamteam.run-lifecycle/v4`. DRAFT, NOT LOCKED. Independent review PENDING. Human presentation review PENDING. Protocol lock and live inference NO GO.

This inventory is committed before any V0.13l recorder or verifier change. The machine form is [bindings_v2.json](../../analysis/alternatives/bindings_v2.json); [lifecycle_v4_spec.py](../../analysis/alternatives/lifecycle_v4_spec.py) refuses an incomplete relationship inventory. V0.13l is contract-driven: instead of adding the five reviewed witnesses only, it states the cross-field relationships every archive must satisfy and tests each one under seven variations.

## Three guarantees

* **canonical** — Canonical consistency without coercion: raw bytes, parsed objects, journal records and reconstructed values agree under strictJson; Python equality is never the comparison for retained JSON.
* **causal** — Causal consistency: failures, stop messages and halts are derived from the recorded event sequence (calls, responses, engine log, workflow phase, limits) and compared with the archived claim; a well-formed label is never accepted on its own.
* **ownership** — Durable evidence ownership: once a model response or completed draw exists it is owned by the recorder (never only a local variable across an interruptible operation) and is retained even if later persistence fails.

## Strict JSON semantics (`dreamteam.strict-json/v1`)

* Parse, duplicateKeys: rejected at every depth (an ambiguous object is never resolved first-wins or last-wins).
* Parse, encoding: UTF-8 (strict); no byte order mark.
* Parse, floats: IEEE-754 binary64 from the decimal text (the protocol producer is Python json.dumps of the same value).
* Parse, integers: arbitrary precision, exact (no float conversion).
* Parse, nonFinite: NaN, Infinity and -Infinity literals rejected; a number whose float value overflows to infinity rejected.
* Types: `null`, `boolean`, `integer`, `float`, `string`, `array`, `object`.
* Equality, array: same length and strictly equal elements in order.
* Equality, boolean: true/false only; never equal to 1/0 or 1.0/0.0.
* Equality, float: bitwise binary64 equality (0.0 is not -0.0).
* Equality, integer: exact value; never equal to a boolean or a float (1 is not 1.0).
* Equality, null: null is a value; a key with null is not an absent key.
* Equality, object: identical key sets (order irrelevant) and strictly equal values.
* Equality, rule: two values are equal iff they have the same type tag and equal content, recursively.
* Equality, string: code-point equality; a numeric string is never a number.

A retained request is produced as requestBody = json.dumps(payload) and request = json.loads(requestBody) by the frozen transport wrapper. Its bytes are retained verbatim; strict reparse of requestBody must equal request; requestSha256 = SHA-256(requestBody bytes); the call journal binds the same hash; replay rebuilds the same payload bytes. Response bodies are retained verbatim; their derived metadata uses the workflow's own parser and is compared strictly.

Applies to: request bytes versus structured request; call journal versus draw calls (status, error, body, hashes, numbering); halt, failure and stop objects across artifacts; engine request arguments and answers in replay; digests, batches, submitted representations, process and record evidence; counters, seal bindings and failure mirrors; pending, candidate and failure-retention copies of a draw.

A seal that is internally consistent never overrides a strictJson mismatch in the evidence it covers.

## Recorder obligations (durable evidence ownership)

### Model response handoff (`dreamteam.response-handoff/v1`)

1. **dispatched** — the dispatch intent record is appended and fsynced before the transport is invoked.
1. **received** — the transport result is owned by the recorder in the statement that receives it (recorder.received).
1. **recorded** — the returned or raised outcome record is appended and fsynced to calls.jsonl.
1. **consumed** — only a recorded response is returned to the workflow.

On failure: an outcome that cannot be recorded is never consumed: the archive is poisoned, the draw ends at once with ResponseNotRecorded, no replacement model call is made and the call budget is not exceeded; the received outcome, its dispatch identity and the persistence error are retained in persistence-failure.json (receivedResponses) and in RecordingError.retainedEvidence; the archive is INVALID.

### Completed-draw handoff (`dreamteam.completed-draw-handoff/v1`)

ownership transfers in the same statement that receives run_draw's return value (recorder.pending = row = run_draw(...)); the return-to-handoff transfer is inside the recovery scope; a handled exception before return recovers the completed or interrupted result from run_draw's frame and completes only the deterministic bookkeeping run_draw's finally block states (calls, modelCallsCompleted, process); write and fsync the handoff before ledger mutation; retain failed candidates; clear only after ledger and commit_draw are durable; a failure at any boundary leaves the completed result in at least one of: ledger, pending-draw.json, persistence-failure.json, or the structured RecordingError; never a claimed commit that did not happen; no arbitrary power-loss or SIGKILL recovery requirement.

### Failure retention (`dreamteam.failed-persistence-retention/v2`)

File `persistence-failure.json`; fields `schema`, `stage`, `error`, `pending`, `partialDraw`, `receivedResponses`, `dispatched`, `candidates`. best-effort unsealed audit evidence, written without hooks, poisoning checks or candidate deletion; the same document is attached to RecordingError.retainedEvidence; never admissible in sealed evidence.

### Post-answer failure (DT13K-06)

a genuine handled failure after the final digest answer but inside run_draw is retained and classified INVALID (final_digests requires digestsAfter null); this is an availability limitation, not a correctness success; no false COMPLETE, no contradictory PARTIAL, no retry or resume, evidence preserved and reported.

## Relationship inventory

Every relationship is planned under the seven kinds `valid_agreement`, `missing_evidence`, `contradiction`, `duplicate_evidence`, `incorrect_identity`, `incorrect_type`, `persistence_interruption`. Archive variations recompute every dependent hash, row commitment, counter mirror and seal before FULL analysis; `persistence_interruption` is a real recorder run with a filesystem fault at the named boundary and a preservation expectation.

| Relationship | Guarantee | Relation |
|---|---|---|
| R-JSON-001 | canonical | raw request bytes versus structured request |
| R-JSON-002 | canonical | archive JSON documents are unambiguous |
| R-JSON-003 | canonical | response body versus derived response metadata |
| R-DISPATCH-001 | causal | dispatch intent versus call journal versus draw calls |
| R-RESPONSE-001 | canonical | retained response versus transport outcome record |
| R-RESPONSE-002 | causal | raised transport outcome versus call error versus primary failure |
| R-RESPONSE-003 | ownership | received model response versus durable record and consumption |
| R-HALT-001 | causal | halt versus reconstructed workflow |
| R-HALT-002 | causal | interrupted row and draw-guard halts versus run state |
| R-FAILURE-001 | causal | failure versus lifecycle stop and failure event |
| R-FAILURE-002 | causal | failed engine entry versus failure stage and primary failure |
| R-FAILURE-003 | causal | reconstructed workflow failure versus recorded failure |
| R-HANDOFF-001 | ownership | completed draw versus pending handoff record |
| R-HANDOFF-002 | ownership | pending handoff versus committed ledger and commit transition |
| R-SEAL-001 | canonical | ledger, journal and call journal versus completion seal |
| R-COUNT-001 | causal | call counts versus schedule limits |
| R-COUNT-002 | canonical | counters versus artifacts |

### R-JSON-001 — raw request bytes versus structured request

* Guarantee: canonical.
* Evidence: `calls.requestBody`, `calls.request`, `calls.requestSha256`, `callJournal.requestSha256`.
* Rule: strictParse(requestBody) is an object; strictEqual(it, request); requestSha256 = SHA-256(requestBody); the journal outcome binds the same hash; replay rebuilds the same bytes and object.
* Verifier: `Classifier._exchanges`, `analyze: replay request comparison`. Recorder: frozen transport wrapper retains json.dumps(payload) and its parse; no recorder copy.
* Missing evidence: INVALID: R-JSON-001 names the missing request field. Permitted UNVERIFIED: never.
* Planned variations:
  * `valid_agreement`: unmodified `complete` archive, accepted.
  * `missing_evidence`: `complete`: `delete` `rows.0.calls.0.requestBody`; rejected.
  * `contradiction`: `complete`: `custom` `request_temperature_changed`; rejected.
  * `duplicate_evidence`: `complete`: `custom` `request_body_duplicate_key`; rejected.
  * `incorrect_identity`: `complete`: `custom` `request_hash_of_other_body`; rejected.
  * `incorrect_type`: `complete`: `custom` `request_zero_versus_false`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `persist:ledger.file_fsync` (occurrence 1); expect INVALID; preserve dispatches, completedRows.

### R-JSON-002 — archive JSON documents are unambiguous

* Guarantee: canonical.
* Evidence: `run.json`, `draws.json`, `calls.jsonl`, `aborted-draw.json`, `failure-event.json`, `server-before.json`, `SEAL.json`.
* Rule: every archive document parses under strictJson; journal lines are canonical; no duplicate keys or non-finite numbers at any depth.
* Verifier: `Archive.json`, `Classifier._journal_records`. Recorder: canonical encoders only.
* Missing evidence: INVALID: L16/L04. Permitted UNVERIFIED: never.
* Planned variations:
  * `valid_agreement`: unmodified `complete` archive, accepted.
  * `missing_evidence`: `complete`: `custom` `truncate_ledger`; rejected.
  * `contradiction`: `complete`: `custom` `journal_noncanonical_line`; rejected.
  * `duplicate_evidence`: `complete`: `duplicate_key` `rows.0.calls.0.status`; rejected.
  * `incorrect_identity`: `complete`: `custom` `journal_line_wrong_schema`; rejected.
  * `incorrect_type`: `complete`: `custom` `nonfinite_latency`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `persist:manifest.write` (occurrence 3); expect INVALID; preserve dispatches.

### R-JSON-003 — response body versus derived response metadata

* Guarantee: canonical.
* Evidence: `calls.body`, `calls.responseSha256`, `calls.usage`, `calls.servedModel`, `calls.finishReason`.
* Rule: responseSha256 = SHA-256(body); usage, servedModel and finishReason equal the workflow parser's fields of body under strictJson.
* Verifier: `Classifier._exchanges`, `H.replay_draw metadata check`. Recorder: frozen PointModel.
* Missing evidence: INVALID: R-JSON-003. Permitted UNVERIFIED: never.
* Planned variations:
  * `valid_agreement`: unmodified `complete` archive, accepted.
  * `missing_evidence`: `complete`: `delete` `rows.0.calls.0.usage`; rejected.
  * `contradiction`: `complete`: `set` `rows.0.calls.0.servedModel` = `"another-model"`; rejected.
  * `duplicate_evidence`: `complete`: `duplicate_key` `rows.0.calls.0.finishReason`; rejected.
  * `incorrect_identity`: `complete`: `custom` `response_hash_of_other_body`; rejected.
  * `incorrect_type`: `complete`: `custom` `usage_number_as_string`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `persist:handoff.file_fsync` (occurrence 1); expect INVALID; preserve dispatches, completedRows.

### R-DISPATCH-001 — dispatch intent versus call journal versus draw calls

* Guarantee: causal.
* Evidence: `callJournal.dispatch`, `callJournal.returned`, `callJournal.raised`, `calls.n`, `run.counters.modelCalls`.
* Rule: k increases by one; each outcome binds its dispatch (k, draw, n); a draw's resolved outcomes equal its calls one to one; a sealed archive has no unresolved dispatch; counters equal dispatches.
* Verifier: `Classifier._calls`. Recorder: dispatch intent appended before transport invocation.
* Missing evidence: INVALID: L04. Permitted UNVERIFIED: never.
* Planned variations:
  * `valid_agreement`: unmodified `complete` archive, accepted.
  * `missing_evidence`: `complete`: `custom` `drop_outcome_record`; rejected.
  * `contradiction`: `complete`: `set` `calls.1.n` = `2`; rejected.
  * `duplicate_evidence`: `complete`: `custom` `duplicate_dispatch_pair`; rejected.
  * `incorrect_identity`: `complete`: `set` `calls.1.k` = `2`; rejected.
  * `incorrect_type`: `complete`: `set` `calls.0.n` = `true`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `append:calls.file_fsync` (occurrence 1); expect INVALID; preserve dispatches.

### R-RESPONSE-001 — retained response versus transport outcome record

* Guarantee: canonical.
* Evidence: `calls.status`, `calls.error`, `calls.body`, `calls.responseSha256`, `callJournal.returned.status`, `callJournal.returned.error`, `callJournal.returned.body`, `callJournal.returned.responseSha256`.
* Rule: every returned outcome field equals its draw call field under strictJson (status true is not 200; 200 is not 200.0).
* Verifier: `Classifier._calls`. Recorder: outcome record built from the owned transport result.
* Missing evidence: INVALID: L04. Permitted UNVERIFIED: never.
* Planned variations:
  * `valid_agreement`: unmodified `complete` archive, accepted.
  * `missing_evidence`: `complete`: `delete` `calls.1.body`; rejected.
  * `contradiction`: `complete`: `custom` `response_status_changed_both`; rejected.
  * `duplicate_evidence`: `complete`: `duplicate_key` `calls.1.status`; rejected.
  * `incorrect_identity`: `complete`: `set` `calls.1.draw` = `2`; rejected.
  * `incorrect_type`: `complete`: `custom` `status_float_in_journal`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `append:calls.write` (occurrence 2); expect INVALID; preserve dispatches, receivedResponses.

### R-RESPONSE-002 — raised transport outcome versus call error versus primary failure

* Guarantee: causal.
* Evidence: `callJournal.raised.exception`, `calls.error`, `aborted.failure`, `run.stop.failure`.
* Rule: a raised outcome is the last call of the aborted draw; call error = "client: T: m"; journal exception = {type T, message m}; the draw failure and the driver stop failure equal it.
* Verifier: `Classifier._aborted`, `verify_aborted`. Recorder: raised outcome recorded from the exception object.
* Missing evidence: INVALID: R-RESPONSE-002. Permitted UNVERIFIED: external cause of the exception only.
* Planned variations:
  * `valid_agreement`: unmodified `aborted_workflow` archive, accepted.
  * `missing_evidence`: `aborted_workflow`: `custom` `raised_exception_removed`; rejected.
  * `contradiction`: `aborted_workflow`: `custom` `raised_disagrees_with_primary`; rejected.
  * `duplicate_evidence`: `aborted_workflow`: `duplicate_key` `aborted.failure.message`; rejected.
  * `incorrect_identity`: `aborted_workflow`: `custom` `raised_type_renamed_everywhere_but_primary`; rejected.
  * `incorrect_type`: `aborted_workflow`: `custom` `raised_message_as_number`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `persist:aborted.file_fsync` (occurrence 1); expect INVALID; preserve dispatches.

### R-RESPONSE-003 — received model response versus durable record and consumption

* Guarantee: ownership.
* Evidence: `recorder.received`, `callJournal.returned`, `failed.receivedResponses`, `failed.partialDraw`, `RecordingError.retainedEvidence`.
* Rule: a returned response is consumed only after its outcome record is durable; if recording fails the response, its dispatch identity and the error are retained and the archive is INVALID; no replacement call.
* Verifier: `Classifier._preserved`, `v13l_preservation response cases`. Recorder: responseHandoff.
* Missing evidence: INVALID; preserved evidence reported. Permitted UNVERIFIED: never.
* Planned variations:
  * `valid_agreement`: unmodified `complete` archive, accepted.
  * `missing_evidence`: `complete`: `custom` `drop_outcome_record`; rejected.
  * `contradiction`: `complete`: `custom` `journal_body_differs_from_call`; rejected.
  * `duplicate_evidence`: `complete`: `custom` `duplicate_outcome_record`; rejected.
  * `incorrect_identity`: `complete`: `set` `calls.1.k` = `9`; rejected.
  * `incorrect_type`: `complete`: `custom` `journal_body_as_object`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `append:calls.file_fsync` (occurrence 2); expect INVALID; preserve dispatches, receivedResponses.

### R-HALT-001 — halt versus reconstructed workflow

* Guarantee: causal.
* Evidence: `halt.code`, `halt.message`, `halt.evidence`, `calls`, `callJournal`, `protocol.stop`, `protocol.maxModelCallsPerDraw`, `protocol.maxModelCalls`.
* Rule: inside a draw the halt stage is request_start; drawCalls = retained calls of the draw; runCalls = all retained calls of the run before the halt; consecutiveTransportFailures = streak over the run's retained calls; code = first guard that fires in limit order; message = the guard's template for that evidence; only elapsedSeconds is an external measurement.
* Verifier: `Classifier._halt`, `replay halt injection`. Recorder: frozen limits.
* Missing evidence: INVALID: R-HALT-001 names the missing or contradicted evidence. Permitted UNVERIFIED: elapsedSeconds (wall clock) only.
* Planned variations:
  * `valid_agreement`: unmodified `halt_close` archive, accepted.
  * `missing_evidence`: `halt_close`: `delete` `aborted.halt.evidence.consecutiveTransportFailures`; rejected.
  * `contradiction`: `halt_close`: `custom` `halt_five_failures_zero_calls`; rejected.
  * `duplicate_evidence`: `halt_close`: `duplicate_key` `aborted.halt.code`; rejected.
  * `incorrect_identity`: `halt_close`: `custom` `halt_stage_draw_start`; rejected.
  * `incorrect_type`: `halt_close`: `set` `aborted.halt.evidence.drawCalls` = `false`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `persist:aborted.write` (occurrence 1); expect INVALID; preserve dispatches.

### R-HALT-002 — interrupted row and draw-guard halts versus run state

* Guarantee: causal.
* Evidence: `rows.halt`, `run.stop.evidence`, `calls`, `callJournal`.
* Rule: an interrupted row's halt and a READY draw-guard halt satisfy the same derivation; the stop repeats the row halt exactly.
* Verifier: `Classifier._halt`, `Classifier._stop`. Recorder: frozen limits.
* Missing evidence: INVALID: R-HALT-001/B-STOP-002. Permitted UNVERIFIED: elapsedSeconds only.
* Planned variations:
  * `valid_agreement`: unmodified `halt_row` archive, accepted.
  * `missing_evidence`: `halt_row`: `delete` `rows.0.halt.evidence.runCalls`; rejected.
  * `contradiction`: `halt_row`: `custom` `row_halt_streak`; rejected.
  * `duplicate_evidence`: `halt_row`: `duplicate_key` `rows.0.halt.message`; rejected.
  * `incorrect_identity`: `halt_row`: `custom` `row_halt_message_other_limit`; rejected.
  * `incorrect_type`: `halt_row`: `custom` `row_halt_draw_calls_float`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `persist:ledger.replace` (occurrence 1); expect INVALID; preserve dispatches.

### R-FAILURE-001 — failure versus lifecycle stop and failure event

* Guarantee: causal.
* Evidence: `aborted.failure`, `aborted.stopReason`, `run.stop.failure`, `run.stop.message`, `run.failureEvent`, `failure-event.json`.
* Rule: stop.message is the derived message of its code and fields; a DRAW_ACTIVE driver stop failure equals the aborted failure; stopReason = stop code; the failure event mirrors the derived stop.
* Verifier: `Classifier._stop`. Recorder: single failure object propagated to every representation.
* Missing evidence: INVALID: R-FAILURE-001. Permitted UNVERIFIED: external cause only.
* Planned variations:
  * `valid_agreement`: unmodified `engine_close` archive, accepted.
  * `missing_evidence`: `engine_close`: `delete` `run.stop.failure`; rejected.
  * `contradiction`: `engine_close`: `custom` `stop_message_disagrees`; rejected.
  * `duplicate_evidence`: `engine_close`: `duplicate_key` `run.stop.message`; rejected.
  * `incorrect_identity`: `engine_close`: `custom` `stop_failure_type_changed`; rejected.
  * `incorrect_type`: `engine_close`: `custom` `stop_failure_message_bool`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `persist:failure_event.file_fsync` (occurrence 1); expect INVALID; preserve dispatches.

### R-FAILURE-002 — failed engine entry versus failure stage and primary failure

* Guarantee: causal.
* Evidence: `aborted.engine`, `aborted.failure`, `aborted.failureStage`, `run.stop.failure`.
* Rule: a failed engine entry is exactly {op, args, answer: null, error}; it is the last engine entry; its error equals "T: m" of the draw failure; its operation fits the failure stage (digests at initial/final digests).
* Verifier: `Classifier._aborted`, `verify_aborted`. Recorder: frozen harness logs the failing request before raising.
* Missing evidence: INVALID: R-FAILURE-002. Permitted UNVERIFIED: external cause only.
* Planned variations:
  * `valid_agreement`: unmodified `initial_digests` archive, accepted.
  * `missing_evidence`: `initial_digests`: `delete` `aborted.engine.0.error`; rejected.
  * `contradiction`: `initial_digests`: `set` `aborted.engine.0.error` = `"ValueError: a different recorded failure"`; rejected.
  * `duplicate_evidence`: `initial_digests`: `custom` `engine_failed_entry_twice`; rejected.
  * `incorrect_identity`: `initial_digests`: `set` `aborted.engine.0.op` = `"observe"`; rejected.
  * `incorrect_type`: `initial_digests`: `set` `aborted.engine.0.answer` = `false`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `persist:aborted.replace` (occurrence 1); expect INVALID; preserve dispatches.

### R-FAILURE-003 — reconstructed workflow failure versus recorded failure

* Guarantee: causal.
* Evidence: `aborted.failure`, `aborted.failureStage`, `aborted.engine`, `aborted.calls`.
* Rule: when replay reaches no external boundary, its reconstructed failure equals the recorded failure strictly; at an external boundary the recorded failure must match the boundary's retained evidence (engine error, raised call) or be the only representation of an external cause.
* Verifier: `verify_aborted`. Recorder: frozen draw machinery.
* Missing evidence: INVALID: R-FAILURE-003. Permitted UNVERIFIED: external cause only.
* Planned variations:
  * `valid_agreement`: unmodified `final_digests` archive, accepted.
  * `missing_evidence`: `final_digests`: `delete` `aborted.failure.message`; rejected.
  * `contradiction`: `final_digests`: `custom` `final_engine_error_disagrees`; rejected.
  * `duplicate_evidence`: `final_digests`: `duplicate_key` `aborted.failureStage`; rejected.
  * `incorrect_identity`: `final_digests`: `set` `aborted.failureStage` = `"workflow"`; rejected.
  * `incorrect_type`: `final_digests`: `set` `aborted.failure.type` = `0`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `persist:aborted.file_fsync` (occurrence 1); expect INVALID; preserve dispatches.

### R-HANDOFF-001 — completed draw versus pending handoff record

* Guarantee: ownership.
* Evidence: `recorder.pending`, `pending.draw`, `pending.rowSha256`, `failed.pending`, `RecordingError.retainedEvidence`.
* Rule: the completed result is owned by the recorder at return; pending.rowSha256 = SHA-256(canonical draw); the handoff is durable before ledger mutation; on any failure the completed result survives in ledger, handoff, failure report or structured error.
* Verifier: `Classifier._preserved`, `v13l_preservation boundary enumeration`. Recorder: completedHandoff.
* Missing evidence: INVALID; preserved evidence reported. Permitted UNVERIFIED: never.
* Planned variations:
  * `valid_agreement`: unmodified `complete` archive, accepted.
  * `missing_evidence`: `complete`: `custom` `pending_without_draw`; rejected.
  * `contradiction`: `complete`: `custom` `pending_draw_differs`; rejected.
  * `duplicate_evidence`: `complete`: `custom` `pending_duplicates_ledger`; rejected.
  * `incorrect_identity`: `complete`: `custom` `pending_wrong_row_hash`; rejected.
  * `incorrect_type`: `complete`: `custom` `pending_draw_as_list`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `persist:handoff.replace` (occurrence 1); expect INVALID; preserve dispatches, completedRows.

### R-HANDOFF-002 — pending handoff versus committed ledger and commit transition

* Guarantee: ownership.
* Evidence: `pending-draw.json`, `draws.json`, `run.journal.commit_draw`, `SEAL.ledger`.
* Rule: a sealed archive retains no handoff; commit_draw binds exactly the ledger row (rowSha256, status, modelCalls, draw); the handoff is cleared only after both are durable.
* Verifier: `Classifier._load`, `Classifier._ledger`. Recorder: completedHandoff.
* Missing evidence: INVALID: L16/L17. Permitted UNVERIFIED: never.
* Planned variations:
  * `valid_agreement`: unmodified `complete` archive, accepted.
  * `missing_evidence`: `complete`: `custom` `commit_without_row`; rejected.
  * `contradiction`: `complete`: `custom` `commit_row_hash_other`; rejected.
  * `duplicate_evidence`: `complete`: `custom` `ledger_row_twice`; rejected.
  * `incorrect_identity`: `complete`: `custom` `commit_names_other_draw`; rejected.
  * `incorrect_type`: `complete`: `custom` `commit_reconciled_as_int`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `handoff_clear.before` (occurrence 1); expect INVALID; preserve dispatches, completedRows.

### R-SEAL-001 — ledger, journal and call journal versus completion seal

* Guarantee: canonical.
* Evidence: `SEAL.ledger`, `SEAL.calls`, `SEAL.manifest`, `SEAL.counters`, `draws.json`, `calls.jsonl`, `run.json`.
* Rule: every seal binding equals the bytes and derived counts on disk; a self-consistent seal never overrides a content contradiction.
* Verifier: `Classifier._seal`. Recorder: unchanged seal format.
* Missing evidence: INVALID: S01-S04. Permitted UNVERIFIED: never.
* Planned variations:
  * `valid_agreement`: unmodified `complete` archive, accepted.
  * `missing_evidence`: `complete`: `custom` `seal_removed`; rejected.
  * `contradiction`: `complete`: `custom` `seal_ledger_rows_count`; rejected.
  * `duplicate_evidence`: `complete`: `custom` `seal_duplicate_key`; rejected.
  * `incorrect_identity`: `complete`: `custom` `seal_run_id_other`; rejected.
  * `incorrect_type`: `complete`: `custom` `seal_rows_as_bool`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `seal.file_fsync` (occurrence 1); expect INVALID; preserve dispatches, completedRows.

### R-COUNT-001 — call counts versus schedule limits

* Guarantee: causal.
* Evidence: `callJournal`, `calls`, `protocol.contexts.callBound`, `protocol.maxModelCallsPerDraw`, `protocol.maxModelCalls`.
* Rule: per draw: dispatches <= min(context bound, per-draw limit); run: dispatches <= min(maxModelCalls, 802); recorded calls never exceed retained dispatches.
* Verifier: `Classifier._counters`. Recorder: frozen limits before every dispatch.
* Missing evidence: INVALID: B-CALL-004. Permitted UNVERIFIED: never.
* Planned variations:
  * `valid_agreement`: unmodified `complete` archive, accepted.
  * `missing_evidence`: `complete`: `custom` `calls_beyond_journal`; rejected.
  * `contradiction`: `complete`: `custom` `run_budget_one`; rejected.
  * `duplicate_evidence`: `complete`: `custom` `calls_repeated_in_row`; rejected.
  * `incorrect_identity`: `complete`: `custom` `dispatch_for_unscheduled_draw`; rejected.
  * `incorrect_type`: `complete`: `set` `rows.0.modelCallsCompleted` = `1.0`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `append:calls.write` (occurrence 1); expect INVALID; preserve dispatches.

### R-COUNT-002 — counters versus artifacts

* Guarantee: canonical.
* Evidence: `run.counters`, `run.failureEvent.modelCalls`, `run.failureEvent.drawsCompleted`, `SEAL.counters`.
* Rule: every counter is recomputed from the ledger, journal, aborted artifact and health checks and compared strictly.
* Verifier: `Classifier._counters`, `Classifier._stop`. Recorder: counters derived from durable state.
* Missing evidence: INVALID: L03. Permitted UNVERIFIED: never.
* Planned variations:
  * `valid_agreement`: unmodified `health_stop` archive, accepted.
  * `missing_evidence`: `health_stop`: `delete` `run.counters.healthChecks`; rejected.
  * `contradiction`: `health_stop`: `set` `run.failureEvent.drawsCompleted` = `1`; rejected.
  * `duplicate_evidence`: `health_stop`: `duplicate_key` `run.counters.modelCalls`; rejected.
  * `incorrect_identity`: `health_stop`: `set` `run.failureEvent.runId` = `"00000000-0000-4000-8000-000000000000"`; rejected.
  * `incorrect_type`: `health_stop`: `set` `run.counters.drawsCompleted` = `false`; rejected.
  * `persistence_interruption`: recorder fault `persistence` at `persist:failure_event.write` (occurrence 1); expect INVALID; preserve dispatches.

## Inherited bindings

The 35 bindings `B-*` of `dreamteam.evidence-bindings/1` remain, with the field-level text documented in [the V0.13k inventory](../v13k/EVIDENCE_BINDINGS.md). V0.13l amends: `B-TRANSPORT-001` (strict JSON request equality), `B-STOP-002` (halt reconstruction, R-HALT-001), `B-HANDOFF-001` (return-to-handoff, run_draw bookkeeping and every line between completion and commit; failure report fields `failed.schema`, `failed.partialDraw`, `failed.receivedResponses`, `failed.dispatched`). Each binding keeps its ten V0.13k variations in the successor binding matrix.

| Binding | Group | Proves |
|---|---|---|
| B-MANIFEST-001 | run manifest | archive belongs to this contract, protocol and schedule |
| B-MANIFEST-002 | run manifest | authorized new run and timestamp provenance |
| B-MANIFEST-003 | run manifest | complete manifest has every field |
| B-LEDGER-001 | draw ledger | unique schedule prefix and valid completed/interrupted row |
| B-LEDGER-002 | draw ledger | completed row has complete recorded initialization |
| B-JOURNAL-001 | run manifest | legal ordered lifecycle chain |
| B-JOURNAL-002 | draw ledger | commit names exact durable row |
| B-CALL-001 | call journal | exactly one dispatch and outcome in run order |
| B-CALL-002 | call journal | outcome bytes and metadata bind dispatch |
| B-CALL-003 | call journal | dispatch precedes send and outcome within draw |
| B-CALL-004 | call journal | dispatch count respects scheduled context even with unknown draw calls |
| B-TRANSPORT-001 | transport exchanges | request bytes, response bytes, metadata and journal agree |
| B-DECISION-001 | decision records | final decision follows retained engine and transport events |
| B-DECISION-002 | decision records | summary reflects record or halted/aborted workflow |
| B-BATCH-001 | submitted batches | primary submitted action equals workflow final batches |
| B-ENGINE-001 | engine digests | all successful engine responses are reconstructable |
| B-ENGINE-002 | engine digests | initial and final state agree with recorded engine responses |
| B-HEALTH-001 | health visits | exact scheduled health visits |
| B-HEALTH-002 | health visits | observation internally classifies from evidence |
| B-HEALTH-003 | health visits | retained bytes match count and hash |
| B-HEALTH-004 | health visits | supported trusted class mapped to family |
| B-HEALTH-005 | health visits | stable observed server signature |
| B-STOP-001 | stop events | single admissible primary stop |
| B-STOP-002 | stop events | recorded halt follows limits and workflow boundary |
| B-INTERRUPT-001 | stop events | operator stop or later interrupt describes real safe-point visit |
| B-CLEANUP-001 | cleanup events | one cleanup visit and finalization in actual order |
| B-ABORT-001 | aborted-draw artifacts | next draw aborted at stated stage |
| B-ABORT-002 | aborted-draw artifacts | retained aborted workflow agrees across every store |
| B-FAILURE-001 | failure events | failure mirror binds exact stop and counts |
| B-HANDOFF-001 | draw ledger | completed pending draw retained across handled failures |
| B-SEAL-001 | completion seal | seal binds contract identities |
| B-SEAL-002 | completion seal | terminal state follows fully checked evidence |
| B-SEAL-003 | completion seal | manifest, ledger, journal bytes and counts committed |
| B-SEAL-004 | completion seal | all health/side evidence and counters committed |
| B-EXECUTION-001 | execution records | only verified submitted actions produce results |

## Review findings mapped

| Finding | Relationships | Obligation |
|---|---|---|
| DT13K-01 | R-JSON-001, R-JSON-002, R-JSON-003, R-RESPONSE-001 | strict typed comparison of every retained request/response representation |
| DT13K-02 | R-FAILURE-001, R-FAILURE-002, R-FAILURE-003, R-RESPONSE-002 | one failure identity across engine entry, raised call, aborted draw, stop, message and event |
| DT13K-03 | R-HALT-001, R-HALT-002 | halt derived from calls, limits and workflow phase; stored classification never trusted |
| DT13K-04 | R-HANDOFF-001, R-HANDOFF-002 | completed draw owned at return; every boundary to ledger commit retains it |
| DT13K-05 | R-RESPONSE-003, R-DISPATCH-001 | received response owned and recorded before consumption; retained on failure |
| DT13K-06 | (lifecycle.postAnswerFailure) | conservative INVALID, documented availability limitation |
| DT13K-07 | R-HANDOFF-001, R-RESPONSE-003 | generated fault contexts keep every preservation expectation |
