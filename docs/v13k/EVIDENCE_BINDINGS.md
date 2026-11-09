# V0.13k evidence binding inventory

Schema: `dreamteam.evidence-bindings/1`. DRAFT, NOT LOCKED. Independent review PENDING. Protocol lock and live inference NO GO.

This inventory precedes recorder and verifier implementation. The machine form is [bindings_v1.json](../../analysis/alternatives/bindings_v1.json). Its executable rule data includes lifecycle /3, stage shapes, hash grammar, scheduled context limits, cleanup ordering and preservation cases. The completion seal and its commit semantics remain unchanged.

UNVERIFIED applies only to an external cause. It never excuses retained contradictory evidence. FULL analysis replays every reconstructable event. Structured fields below include every descendant: whole recursive equality binds variable domain/action/request/engine payloads, including optional workflow branches. No leaf is exempt.

## Field types

* `run`: closed object.
* `journal`: list of event-specific closed objects.
* `draw`: closed draw/v4 object.
* `callJournal`: list of closed dispatch/returned/raised variants.
* `calls`: closed per-call object; request and usage recursive domain JSON.
* `record`: workflow branch-specific object; every descendant recursively compared, including validationFirst attempts/inputs/parses/validations/stages/completeness, turn proposal/items/views/pins, deliveries messages/content/status/error/body, exchanges parse/answer hashes and outcome actions/citations.
* `engine`: list of closed {op:string,args:domain JSON,answer:protocol JSON}; error:string only for failed request; all descendants replay compared.
* `batches`: ordered arrays of domain action JSON; recursive exact comparison.
* `process`: closed record summary or derived halted/aborted object.
* `healthChecks`: list of closed visit objects.
* `health`: closed observation object.
* `serverState`: object of external server metadata with recursively bound health; initial mirror and stable signature checked.
* `stop`: closed code-specific object.
* `halt`: closed guard evidence.
* `interrupts`: closed object with ordered event list.
* `cleanup`: closed result with start and exact event list.
* `aborted`: stage-specific closed draw object plus failure fields.
* `failureEvent`: closed failure mirror.
* `pending`: closed handoff object.
* `failed`: unsealed failure report.
* `SEAL`: closed completion-seal/v1 object.
* `analysis`: analysis/v8; recursive reconstructed output.

## B-MANIFEST-001 — run manifest

archive belongs to this contract, protocol and schedule.

* Fields (including all descendants): `run.schema`, `run.driver`, `run.lifecycle.id`, `run.lifecycle.sha256`, `run.protocol`, `run.protocolSha256`, `run.scheduleSha256`.
* Source: recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `SEAL.lifecycle`, `SEAL.driver`, `SEAL.protocol`, `SEAL.scheduleSha256`.
* Consistency / invalidity: exact successor identities and input hashes.
* Replay: recompute all identities.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-MANIFEST-002 — run manifest

authorized new run and timestamp provenance.

* Fields (including all descendants): `run.runId`, `run.authorization`, `run.provenance.timestamps`, `run.provenance.runId`, `run.serverObserved`, `run.resume`, `run.verificationStatus`.
* Source: recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `SEAL.runId`, `failureEvent.runId`, `healthChecks`.
* Consistency / invalidity: canonical UUID; nonempty authorization; resume never; ordinary runs observe server; fixtures explicitly admitted.
* Replay: validate provenance; never infer validity from claimed status.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: external cause only; retained data always checked.
* Executable rule: `semantic`.

## B-MANIFEST-003 — run manifest

complete manifest has every field.

* Fields (including all descendants): `run.state`, `run.started`, `run.journal`, `run.counters`, `run.healthChecks`, `run.stop`, `run.failureEvent`, `run.cleanup`, `run.interrupts`.
* Source: recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `ledger`, `callJournal`, `SEAL`.
* Consistency / invalidity: closed field set and field types; derive terminal state independently.
* Replay: reconstruct all constituent bindings.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-LEDGER-001 — draw ledger

unique schedule prefix and valid completed/interrupted row.

* Fields (including all descendants): `draw.schema`, `draw.seq`, `draw.model`, `draw.context`, `draw.arm`, `draw.draw`, `draw.status`.
* Source: draw runner; recorder commits.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `schedule`, `journal.begin_draw`, `journal.commit_draw`, `aborted.seq`.
* Consistency / invalidity: exact scheduled identities; at most last row interrupted; no aborted ledger row.
* Replay: replay scheduled context; compare deterministic draw.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-LEDGER-002 — draw ledger

completed row has complete recorded initialization.

* Fields (including all descendants): `draw.engineInitialization`, `draw.failureStage`, `draw.verificationStatus`, `draw.modelCallsCompleted`.
* Source: draw runner.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `draw.calls`, `draw.status`, `draw.failure`.
* Consistency / invalidity: READY, null failure stage, RECORDED, exact call count; no failure or cleanupFailure.
* Replay: compare draw replay.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-JOURNAL-001 — run manifest

legal ordered lifecycle chain.

* Fields (including all descendants): `journal.n`, `journal.event`, `journal.from`, `journal.to`, `journal.timestamp`.
* Source: recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `run.state`, `cleanup`, `SEAL.transition`.
* Consistency / invalidity: closed event-specific field sets; legal table and predicates; FINALIZING tail; terminal transition only in seal.
* Replay: walk shared transition data.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-JOURNAL-002 — draw ledger

commit names exact durable row.

* Fields (including all descendants): `journal.draw`, `journal.status`, `journal.rowSha256`, `journal.modelCalls`, `journal.reconciled`.
* Source: recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `draw`, `schedule`, `callJournal`, `stop`.
* Consistency / invalidity: begin names next scheduled draw; commit row hash/status/calls; reconciled commit followed by stop.
* Replay: hash canonical row and compare.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-CALL-001 — call journal

exactly one dispatch and outcome in run order.

* Fields (including all descendants): `callJournal.schema`, `callJournal.record`, `callJournal.k`, `callJournal.draw`, `callJournal.n`.
* Source: recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `schedule`, `journal.begin_draw`, `draw.calls`.
* Consistency / invalidity: closed record variants; contiguous positive integers; alternating dispatch/outcome; no unresolved call sealed; no return to prior draw.
* Replay: reconcile every recorded call.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-CALL-002 — call journal

outcome bytes and metadata bind dispatch.

* Fields (including all descendants): `callJournal.requestSha256`, `callJournal.responseSha256`, `callJournal.body`, `callJournal.status`, `callJournal.error`, `callJournal.exception.type`, `callJournal.exception.message`.
* Source: transport wrapper.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `draw.calls`, `record.deliveries`, `record.exchanges`.
* Consistency / invalidity: lowercase hexadecimal SHA-256; body and response hash match; raised exception has no returned fields; retained request hashes recomputed.
* Replay: rebuild request through workflow; compare status/error/body.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `hashGrammar`.

## B-CALL-003 — call journal

dispatch precedes send and outcome within draw.

* Fields (including all descendants): `callJournal.timestamp`, `calls.sent`.
* Source: recorder; transport.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `journal.begin_draw`, `journal.commit_draw`, `journal.stop`.
* Consistency / invalidity: strict timestamps; monotonic per draw and globally; dispatch <= sent <= outcome.
* Replay: validate intervals; timestamp metadata excluded from replay.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `timestamp`.

## B-CALL-004 — call journal

dispatch count respects scheduled context even with unknown draw calls.

* Fields (including all descendants): `callJournal.draw`, `callJournal.n`, `run.counters.modelCalls`, `failureEvent.modelCalls`, `halt.evidence.runCalls`.
* Source: recorder; scheduled context.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `schedule.context`, `protocol.contexts.callBound`, `protocol.maxModelCallsPerDraw`, `protocol.maxModelCalls`.
* Consistency / invalidity: derive bound from schedule context identity; no fallback for missing context; limit independently of nullable draw.calls; global ceiling 802.
* Replay: count dispatches directly and reconcile totals.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `callLimits`.

## B-TRANSPORT-001 — transport exchanges

request bytes, response bytes, metadata and journal agree.

* Fields (including all descendants): `calls.n`, `calls.sent`, `calls.latencyMs`, `calls.request`, `calls.requestBody`, `calls.requestSha256`, `calls.status`, `calls.error`, `calls.body`, `calls.responseSha256`, `calls.usage`, `calls.servedModel`, `calls.finishReason`.
* Source: point transport.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `callJournal`, `record.deliveries`, `record.exchanges`.
* Consistency / invalidity: exact call fields; n order; nonnegative finite latency; JSON request equals bytes; hashes and body-derived metadata exact.
* Replay: workflow reconstructs exact request; response metadata re-parsed.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-DECISION-001 — decision records

final decision follows retained engine and transport events.

* Fields (including all descendants): `record.schema`, `record.workflowVersion`, `record.workflowSha256`, `record.contractPins`, `record.arm`, `record.validationFirst`, `record.turn`, `record.exchanges`, `record.outcome`, `record.result`, `record.batches`, `record.errors`, `record.inspected`, `record.eligibility`, `record.exposure`, `record.validationFirstCorrection`, `record.correctionFailure`, `record.deliberateNoAction`, `record.processFailure`, `record.retryRequests`, `record.deliveries`, `record.intendedAction`, `record.validatedAction`, `record.executedAction`, `record.proposalSubmittedAction`, `record.responseRoute`, `record.digestsBeforeSubmit`, `record.submittedAction`.
* Source: unchanged workflow.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: completed, final_digests, engine_close.
* Related: `engine`, `calls`, `batches`, `process`, `initialDigests`, `digestsAfter`.
* Consistency / invalidity: all fields recursively equal workflow replay, including proposal, final response, halt exclusivity, submissions and pins; conditional fields absent only on workflow branches that omit them.
* Replay: run unchanged workflow using recorded calls; compare complete record recursively.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-DECISION-002 — decision records

summary reflects record or halted/aborted workflow.

* Fields (including all descendants): `process.eligibility`, `process.exposure`, `process.validationFirstCorrection`, `process.correctionFailure`, `process.deliberateNoAction`, `process.processFailure`, `process.intendedAction`, `process.validatedAction`, `process.submittedAction`, `process.executedAction`, `process.pendingMessage`.
* Source: draw runner.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `record`, `engine`, `calls`.
* Consistency / invalidity: record-derived fields exact; otherwise derive from engine/calls; pending message must equal reconstructed workflow request.
* Replay: compare reconstructed process recursively.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-BATCH-001 — submitted batches

primary submitted action equals workflow final batches.

* Fields (including all descendants): `draw.batches`, `draw.submittedRepresentation`, `draw.submittedRepresentationSha256`.
* Source: draw runner.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `record.batches`, `record.submittedAction`, `record.proposalSubmittedAction`, `engine.canonical`.
* Consistency / invalidity: exact representation parses to batches; hash exact; no abandoned proposal substituted; null iff no returned record at aborted stage.
* Replay: compare workflow batches and canonical engine answer.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-ENGINE-001 — engine digests

all successful engine responses are reconstructable.

* Fields (including all descendants): `draw.engine.op`, `draw.engine.args`, `draw.engine.answer`, `draw.engine.error`.
* Source: engine harness.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `record`, `calls`, `initialDigests`, `digestsAfter`.
* Consistency / invalidity: ordered exact request/argument/answer sequence; only failed external request may have null answer and an error; no invented entries.
* Replay: fresh context session; replay each successful request and compare entire response.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-ENGINE-002 — engine digests

initial and final state agree with recorded engine responses.

* Fields (including all descendants): `draw.initialDigests`, `draw.digestsAfter`.
* Source: engine harness.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `context.digests`, `engine.digests`, `record.digestsBeforeSubmit`.
* Consistency / invalidity: initial equals context and initial response; final equals final response; stage fixes presence.
* Replay: re-run digests against context.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-HEALTH-001 — health visits

exact scheduled health visits.

* Fields (including all descendants): `healthChecks.index`, `healthChecks.state`, `healthChecks.beforeDraw`, `healthChecks.serverState`, `healthChecks.rejected`.
* Source: recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `journal.healthCheck`, `schedule`, `server-before.json`.
* Consistency / invalidity: one initial visit then one per required round; visit identity and order exact; no extra, duplicate or unowned check; kept/rejected exclusive.
* Replay: derive visits from schedule and transition chain.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-HEALTH-002 — health visits

observation internally classifies from evidence.

* Fields (including all descendants): `health.schema`, `health.request.method`, `health.request.path`, `health.started`, `health.ended`, `health.outcome`, `health.category`, `health.httpStatus`, `health.exception`, `health.body`, `health.verificationStatus`.
* Source: health probe.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `healthChecks`, `stop`, `journal.passed`.
* Consistency / invalidity: closed fields; body category requires 2xx and no exception; read family requires exact allowed serialized identity; rejection must be invalid; only external read cause unverified.
* Replay: decode and reclassify bytes; compare category/outcome/status.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: external cause only; retained data always checked.
* Executable rule: `semantic`.

## B-HEALTH-003 — health visits

retained bytes match count and hash.

* Fields (including all descendants): `health.body.bytes`, `health.body.sha256`, `health.body.base64`.
* Source: health probe.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `health.category`, `health.outcome`.
* Consistency / invalidity: strict base64; exact byte count; lowercase SHA-256; UTF-8 and strict JSON classification; no duplicate keys or nonfinite values.
* Replay: recompute hashes and categories.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-HEALTH-004 — health visits

supported trusted class mapped to family.

* Fields (including all descendants): `health.exception.type`, `health.exception.reasonType`, `health.exception.reasonKind`.
* Source: health probe.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `health.category`, `health.outcome`.
* Consistency / invalidity: recorder uses actual imported class identity; wrapped exceptions only wrappable; impostor same text refused.
* Replay: verifier checks closed names; cannot independently reconstruct external exception object.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: external cause only; retained data always checked.
* Executable rule: `exceptionIdentity`.

## B-HEALTH-005 — health visits

stable observed server signature.

* Fields (including all descendants): `server-before.json`, `serverState.time`, `serverState.serverVersion`, `serverState.serverTitle`, `serverState.openapiSha256`, `serverState.models`.
* Source: health probe.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `healthChecks`, `journal.passed`, `stop.beforeSignature`.
* Consistency / invalidity: initial file equals initial kept state; passing rounds have unchanged signature; health times within visit; metadata externally observed.
* Replay: recompute signature from retained server fields.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: external cause only; retained data always checked.
* Executable rule: `semantic`.

## B-STOP-001 — stop events

single admissible primary stop.

* Fields (including all descendants): `stop.code`, `stop.state`, `stop.message`, `stop.failure`, `stop.evidence`, `stop.request`, `stop.healthCheck`, `stop.beforeSignature`, `stop.safePoint`.
* Source: recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `journal.stop`, `aborted.failure`, `draw.halt`, `cleanup.failure`, `failureEvent.stop`.
* Consistency / invalidity: field presence/null per stop code and state; stop matches journal; earlier stop stays primary.
* Replay: rederive stop evidence and call guard priority.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: external cause only; retained data always checked.
* Executable rule: `semantic`.

## B-STOP-002 — stop events

recorded halt follows limits and workflow boundary.

* Fields (including all descendants): `halt.code`, `halt.message`, `halt.evidence.stage`, `halt.evidence.drawCalls`, `halt.evidence.runCalls`, `halt.evidence.elapsedSeconds`, `halt.evidence.consecutiveTransportFailures`.
* Source: guard.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `calls`, `callJournal`, `protocol.stop`, `stop`.
* Consistency / invalidity: exact guard priority; counts and streak derived from calls; stage and elapsed bound consistent; aborted record xor halt.
* Replay: inject halt at replay dispatch count and compare workflow prefix.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-INTERRUPT-001 — stop events

operator stop or later interrupt describes real safe-point visit.

* Fields (including all descendants): `interrupts.signal`, `interrupts.events.n`, `interrupts.events.safePoint`, `interrupts.events.state`, `interrupts.events.signals`, `interrupts.events.disposition`, `interrupts.events.timestamp`.
* Source: signal latch and recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `stop`, `cleanup.events`, `journal.schedule_done`, `SEAL.transition`.
* Consistency / invalidity: SIGINT; positive counts; one STOP iff operator; RECORD events ordered nonduplicated subsequence; exact visit timestamps; state and disposition agree.
* Replay: reconcile events with stops and mandatory cleanup visits.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `orderedSubsequence`.

## B-CLEANUP-001 — cleanup events

one cleanup visit and finalization in actual order.

* Fields (including all descendants): `cleanup.status`, `cleanup.failure`, `cleanup.timestamp`, `cleanup.started`, `cleanup.events.n`, `cleanup.events.safePoint`, `cleanup.events.state`, `cleanup.events.timestamp`.
* Source: recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `journal.cleanup`, `interrupt.events`, `SEAL.transition`.
* Consistency / invalidity: required exact three visits; order/multiplicity/stage; start <= before <= completion <= after <= finalization <= seal; result and cleanup transition agree.
* Replay: validate sequence, times, disposition and seal.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `cleanupSequence`.

## B-ABORT-001 — aborted-draw artifacts

next draw aborted at stated stage.

* Fields (including all descendants): `aborted.schema`, `aborted.seq`, `aborted.model`, `aborted.context`, `aborted.arm`, `aborted.draw`, `aborted.status`, `aborted.failureStage`, `aborted.engineInitialization`, `aborted.verificationStatus`, `aborted.stopReason`, `aborted.failure`, `aborted.cleanupFailure`.
* Source: draw runner or recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `schedule`, `journal.begin_draw`, `stop`.
* Consistency / invalidity: closed fields; exact per-stage shape; record xor halt after workflow; no completed pending draw reclassified unrecorded.
* Replay: replay all reconstructable evidence before reporting external cause unverified.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: external cause only; retained data always checked.
* Executable rule: `stageShape`.

## B-ABORT-002 — aborted-draw artifacts

retained aborted workflow agrees across every store.

* Fields (including all descendants): `aborted.record`, `aborted.batches`, `aborted.engine`, `aborted.initialDigests`, `aborted.digestsAfter`, `aborted.calls`, `aborted.modelCallsCompleted`, `aborted.process`, `aborted.halt`, `aborted.submittedRepresentation`, `aborted.submittedRepresentationSha256`.
* Source: draw runner.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `record`, `engine`, `calls`, `callJournal`, `batches`, `context`.
* Consistency / invalidity: every completed engine/workflow event and final proposal replayed; no missing call with retained exchange; no wrong digest or stage; UNVERIFIED only external cause.
* Replay: stage-aware full workflow/engine prefix replay.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: external cause only; retained data always checked.
* Executable rule: `replayAllRetained`.

## B-FAILURE-001 — failure events

failure mirror binds exact stop and counts.

* Fields (including all descendants): `failureEvent.schema`, `failureEvent.runId`, `failureEvent.protocolSha256`, `failureEvent.scheduleSha256`, `failureEvent.state`, `failureEvent.stop`, `failureEvent.nextDraw`, `failureEvent.drawsCompleted`, `failureEvent.modelCalls`, `failureEvent.timestamp`, `failureEvent.verificationStatus`.
* Source: recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `run.failureEvent`, `failure-event.json`, `stop`, `counters`, `schedule`.
* Consistency / invalidity: exact field set and equality; external cause may stay unverified; all reconstructed fields verified.
* Replay: derive event from stop and artifacts.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: external cause only; retained data always checked.
* Executable rule: `semantic`.

## B-HANDOFF-001 — draw ledger

completed pending draw retained across handled failures.

* Fields (including all descendants): `pending.schema`, `pending.draw`, `pending.rowSha256`, `failed.stage`, `failed.error`, `failed.pending`, `failed.candidates`.
* Source: recorder.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: completed, handoff, ledger_commit, error_handling.
* Related: `draws.json`, `callJournal`, `journal.commit_draw`, `failed`, `candidate`, `bytes`.
* Consistency / invalidity: durable pending handoff before ledger mutation; never delete failed candidate containing completed draw; pending cleared only after durable commit; retained failure evidence unsealed INVALID.
* Replay: inspect candidate/pending alongside committed ledger; preservation checks include both.
* Permitted absence: pending absent only before result completion or after durable ledger and commit transition; failure report optional if storage refuses writes.
* Permitted UNVERIFIED: external cause only; retained data always checked.
* Executable rule: `preserveCompleted`.

## B-SEAL-001 — completion seal

seal binds contract identities.

* Fields (including all descendants): `SEAL.schema`, `SEAL.finalization`, `SEAL.lifecycle`, `SEAL.runSchema`, `SEAL.driver`, `SEAL.runId`, `SEAL.protocol`, `SEAL.scheduleSha256`.
* Source: recorder finalization.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `run`, `protocol`, `schedule`.
* Consistency / invalidity: exact existing completion-seal/v1 schema and finalization/v1 semantics; successor lifecycle/driver identities.
* Replay: recompute all identity bindings.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-SEAL-002 — completion seal

terminal state follows fully checked evidence.

* Fields (including all descendants): `SEAL.terminalState`, `SEAL.transition.n`, `SEAL.transition.event`, `SEAL.transition.from`, `SEAL.transition.to`, `SEAL.transition.timestamp`.
* Source: recorder finalization.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `journal`, `cleanup`, `ledger`, `stop`.
* Consistency / invalidity: terminal transition only here; n=len(journal)+1; FINALIZING to derived state; timestamp after all safe points.
* Replay: derive state and compare seal; seal necessary never sufficient.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-SEAL-003 — completion seal

manifest, ledger, journal bytes and counts committed.

* Fields (including all descendants): `SEAL.manifest.file`, `SEAL.manifest.sha256`, `SEAL.manifest.journalEntries`, `SEAL.ledger.file`, `SEAL.ledger.sha256`, `SEAL.ledger.rows`, `SEAL.ledger.completed`, `SEAL.ledger.rowSha256`, `SEAL.calls.file`, `SEAL.calls.sha256`, `SEAL.calls.records`, `SEAL.calls.dispatched`.
* Source: recorder finalization.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `run`, `draws.json`, `calls.jsonl`.
* Consistency / invalidity: exact file identities; all raw and canonical hashes and counts recomputed.
* Replay: hash bytes read once; compare canonical rows.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-SEAL-004 — completion seal

all health/side evidence and counters committed.

* Fields (including all descendants): `SEAL.health.checks`, `SEAL.health.checkSha256`, `SEAL.health.serverBefore`, `SEAL.sideArtifacts`, `SEAL.counters`.
* Source: recorder finalization.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `healthChecks`, `server-before.json`, `aborted-draw.json`, `failure-event.json`, `run.counters`.
* Consistency / invalidity: exact required side artifact set; null requires absence; hashes counts exact; no stray accepted archive files.
* Replay: recompute all hashes/counters.
* Permitted absence: only when the containing optional artifact or stage-specific parent is absent.
* Permitted UNVERIFIED: never.
* Executable rule: `semantic`.

## B-EXECUTION-001 — execution records

only verified submitted actions produce results.

* Fields (including all descendants): `analysis.executions`, `analysis.draws`, `analysis.table`, `analysis.armDifferences`, `analysis.pairs`, `analysis.abortedVerification`, `analysis.failureVerification`, `analysis.completedVerification`, `analysis.lifecycle`.
* Source: independent analyzer.
* Type: explicit per-field types, nullability and required-parent rules in the corresponding machine `fieldDetails`; structured descendants are compared recursively with JSON type identity.
* Required: yes within applicable parent; stages: all.
* Related: `draw.record`, `batches`, `engine`, `stop`, `SEAL`.
* Consistency / invalidity: execute canonical submitted batches; deduplicate by context and exact representation hash; PARTIAL suppresses comparisons; reject contradictory archive; failure status only external cause.
* Replay: recompute execution record and all analyses; never trust recorder status.
* Permitted absence: analysis absent for INVALID archives; no analysis file is part of sealed live archive.
* Permitted UNVERIFIED: external cause only; retained data always checked.
* Executable rule: `semantic`.

## Review findings

| Finding | Bindings |
|---|---|
| DT13J-01 | B-ABORT-001/002, B-DECISION-001/002, B-BATCH-001, B-ENGINE-001/002, B-CALL-001/002 |
| DT13J-02 | B-CLEANUP-001, B-INTERRUPT-001, B-SEAL-002 |
| DT13J-03 | B-CALL-002/003/004, B-LEDGER-001, B-TRANSPORT-001 |
| DT13J-04 | B-HANDOFF-001, B-JOURNAL-002, B-LEDGER-002 |
| DT13J-05 | B-HEALTH-004 |

The binding matrix enumerates every binding against the ten mutation kinds in the machine inventory. Applicability and expected results must be explicit per case; all contradictory archives must be refused by FULL analysis after dependent hashes and seals are recomputed. Positive controls must be accepted.

## Explicit field types and conditional presence

The machine inventory enumerates fixed descendant fields in `fieldDetails` for each binding. Each entry states its type, nullability, required-parent condition and owning binding. Stage tables determine when parents may be absent; null never grants an exemption from retained-evidence checks. Canonical JSON equality preserves boolean versus numeric identity. Variable domain subtrees are defined by typed reconstruction from the frozen workflow and engine, covering every key and array position. The initial inventory spelling `calls.latencySeconds` was corrected to the persisted `calls.latencyMs` before implementation.

### B-MANIFEST-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `run.driver` | string | no |
| `run.lifecycle.id` | string | no |
| `run.lifecycle.sha256` | sha256 | no |
| `run.protocol` | string | no |
| `run.protocolSha256` | sha256 | no |
| `run.scheduleSha256` | sha256 | no |
| `run.schema` | string | no |

### B-MANIFEST-002 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `run.authorization` | string | no |
| `run.provenance.runId` | string | no |
| `run.provenance.timestamps` | string | no |
| `run.resume` | string | no |
| `run.runId` | string | no |
| `run.serverObserved` | boolean | no |
| `run.verificationStatus` | string | no |

### B-MANIFEST-003 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `run.cleanup` | object | no |
| `run.counters` | object | no |
| `run.counters.drawAborted` | integer | stage-dependent |
| `run.counters.drawInterrupted` | integer | stage-dependent |
| `run.counters.drawsCompleted` | integer | no |
| `run.counters.drawsNotRun` | integer | no |
| `run.counters.drawsScheduled` | integer | no |
| `run.counters.healthChecks` | integer | no |
| `run.counters.modelCalls` | integer | no |
| `run.failureEvent` | object | stage-dependent |
| `run.healthChecks` | array | no |
| `run.interrupts` | object | no |
| `run.journal` | array | no |
| `run.started` | UTC timestamp | no |
| `run.state` | string | no |
| `run.stop` | object | stage-dependent |

### B-LEDGER-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `draw.arm` | string | no |
| `draw.context` | string | no |
| `draw.draw` | integer | no |
| `draw.model` | string | no |
| `draw.schema` | string | no |
| `draw.seq` | integer | no |
| `draw.status` | string | stage-dependent |

### B-LEDGER-002 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `draw.engineInitialization` | string | no |
| `draw.failureStage` | string | stage-dependent |
| `draw.modelCallsCompleted` | integer | stage-dependent |
| `draw.verificationStatus` | string | no |

### B-JOURNAL-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `journal.event` | string | no |
| `journal.from` | string | stage-dependent |
| `journal.n` | integer | no |
| `journal.timestamp` | UTC timestamp | no |
| `journal.to` | string | no |

### B-JOURNAL-002 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `journal.draw` | integer | no |
| `journal.modelCalls` | integer | no |
| `journal.reconciled` | boolean | no |
| `journal.rowSha256` | sha256 | no |
| `journal.status` | string | stage-dependent |

### B-CALL-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `callJournal.draw` | integer | no |
| `callJournal.k` | integer | no |
| `callJournal.n` | integer | no |
| `callJournal.record` | string | stage-dependent |
| `callJournal.schema` | string | no |

### B-CALL-002 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `callJournal.body` | string | stage-dependent |
| `callJournal.error` | string | stage-dependent |
| `callJournal.exception.message` | string | no |
| `callJournal.exception.type` | string | no |
| `callJournal.requestSha256` | sha256 | no |
| `callJournal.responseSha256` | sha256 | no |
| `callJournal.status` | integer | stage-dependent |

### B-CALL-003 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `callJournal.timestamp` | UTC timestamp | no |
| `calls.sent` | UTC timestamp | no |

### B-CALL-004 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `callJournal.draw` | integer | no |
| `callJournal.n` | integer | no |
| `failureEvent.modelCalls` | integer | no |
| `halt.evidence.runCalls` | integer | no |
| `run.counters.modelCalls` | integer | no |

### B-TRANSPORT-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `calls.body` | string | stage-dependent |
| `calls.error` | string | stage-dependent |
| `calls.finishReason` | string | stage-dependent |
| `calls.latencyMs` | finite nonnegative number | no |
| `calls.n` | integer | no |
| `calls.request` | object | stage-dependent |
| `calls.requestBody` | string | stage-dependent |
| `calls.requestSha256` | sha256 | no |
| `calls.responseSha256` | sha256 | no |
| `calls.sent` | UTC timestamp | no |
| `calls.servedModel` | string | stage-dependent |
| `calls.status` | integer | stage-dependent |
| `calls.usage` | object | stage-dependent |

### B-DECISION-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `record.arm` | string | no |
| `record.batches` | array | stage-dependent |
| `record.contractPins` | object | no |
| `record.correctionFailure` | boolean | no |
| `record.deliberateNoAction` | boolean | no |
| `record.deliveries` | array | no |
| `record.digestsBeforeSubmit` | object | stage-dependent |
| `record.eligibility` | object | no |
| `record.errors` | array | no |
| `record.exchanges` | array | no |
| `record.executedAction` | object | stage-dependent |
| `record.exposure` | boolean | no |
| `record.inspected` | array | no |
| `record.intendedAction` | array | stage-dependent |
| `record.outcome` | object | stage-dependent |
| `record.processFailure` | boolean | no |
| `record.proposalSubmittedAction` | object | stage-dependent |
| `record.responseRoute` | string | stage-dependent |
| `record.result` | string | no |
| `record.retryRequests` | integer | no |
| `record.schema` | string | no |
| `record.submittedAction` | object | stage-dependent |
| `record.turn` | object | stage-dependent |
| `record.validatedAction` | array | stage-dependent |
| `record.validationFirst` | object | no |
| `record.validationFirstCorrection` | boolean | no |
| `record.workflowSha256` | sha256 | no |
| `record.workflowVersion` | string | no |

### B-DECISION-002 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `process.correctionFailure` | boolean | no |
| `process.deliberateNoAction` | boolean | no |
| `process.eligibility` | object | no |
| `process.executedAction` | object | stage-dependent |
| `process.exposure` | boolean | no |
| `process.intendedAction` | array | stage-dependent |
| `process.pendingMessage` | domain JSON | stage-dependent |
| `process.processFailure` | boolean | no |
| `process.submittedAction` | object | stage-dependent |
| `process.validatedAction` | array | stage-dependent |
| `process.validationFirstCorrection` | boolean | no |

### B-BATCH-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `draw.batches` | array | stage-dependent |
| `draw.submittedRepresentation` | string | stage-dependent |
| `draw.submittedRepresentationSha256` | sha256 | stage-dependent |

### B-ENGINE-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `draw.engine.answer` | object | stage-dependent |
| `draw.engine.args` | domain JSON | stage-dependent |
| `draw.engine.error` | object | stage-dependent |
| `draw.engine.op` | string | no |

### B-ENGINE-002 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `draw.digestsAfter` | object | stage-dependent |
| `draw.initialDigests` | object | stage-dependent |

### B-HEALTH-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `healthChecks.beforeDraw` | integer | stage-dependent |
| `healthChecks.index` | integer | no |
| `healthChecks.rejected` | domain JSON | stage-dependent |
| `healthChecks.serverState` | object | stage-dependent |
| `healthChecks.state` | string | no |

### B-HEALTH-002 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `health.body` | object | stage-dependent |
| `health.category` | string | no |
| `health.ended` | UTC timestamp | no |
| `health.exception` | object | stage-dependent |
| `health.httpStatus` | integer | stage-dependent |
| `health.outcome` | string | no |
| `health.request.method` | string | no |
| `health.request.path` | string | no |
| `health.schema` | string | no |
| `health.started` | UTC timestamp | no |
| `health.verificationStatus` | string | no |

### B-HEALTH-003 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `health.body.base64` | domain JSON | no |
| `health.body.bytes` | integer | no |
| `health.body.sha256` | sha256 | no |

### B-HEALTH-004 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `health.exception.reasonKind` | string | stage-dependent |
| `health.exception.reasonType` | string | stage-dependent |
| `health.exception.type` | string | no |

### B-HEALTH-005 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `server-before.json` | object | no |
| `serverState.models` | domain JSON | stage-dependent |
| `serverState.openapiSha256` | sha256 | stage-dependent |
| `serverState.serverTitle` | string | stage-dependent |
| `serverState.serverVersion` | string | stage-dependent |
| `serverState.time` | string | stage-dependent |

### B-STOP-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `stop.beforeSignature` | object | stage-dependent |
| `stop.code` | string | no |
| `stop.evidence` | object | stage-dependent |
| `stop.failure` | object | stage-dependent |
| `stop.healthCheck` | integer | stage-dependent |
| `stop.message` | string | no |
| `stop.request` | object | stage-dependent |
| `stop.safePoint` | string | stage-dependent |
| `stop.state` | string | no |

### B-STOP-002 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `halt.code` | string | no |
| `halt.evidence.consecutiveTransportFailures` | integer | no |
| `halt.evidence.drawCalls` | integer | no |
| `halt.evidence.elapsedSeconds` | finite nonnegative number | no |
| `halt.evidence.runCalls` | integer | no |
| `halt.evidence.stage` | string | no |
| `halt.message` | string | no |

### B-INTERRUPT-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `interrupts.events.disposition` | string | no |
| `interrupts.events.n` | integer | no |
| `interrupts.events.safePoint` | string | no |
| `interrupts.events.signals` | integer | no |
| `interrupts.events.state` | string | no |
| `interrupts.events.timestamp` | UTC timestamp | no |
| `interrupts.signal` | string | no |

### B-CLEANUP-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `cleanup.events.n` | integer | no |
| `cleanup.events.safePoint` | string | no |
| `cleanup.events.state` | string | no |
| `cleanup.events.timestamp` | UTC timestamp | no |
| `cleanup.failure` | object | stage-dependent |
| `cleanup.started` | UTC timestamp | no |
| `cleanup.status` | string | stage-dependent |
| `cleanup.timestamp` | UTC timestamp | no |

### B-ABORT-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `aborted.arm` | string | no |
| `aborted.cleanupFailure` | object | stage-dependent |
| `aborted.context` | string | no |
| `aborted.draw` | integer | no |
| `aborted.engineInitialization` | string | no |
| `aborted.failure` | object | stage-dependent |
| `aborted.failureStage` | string | stage-dependent |
| `aborted.model` | string | no |
| `aborted.schema` | string | no |
| `aborted.seq` | integer | no |
| `aborted.status` | string | stage-dependent |
| `aborted.stopReason` | string | no |
| `aborted.verificationStatus` | string | no |

### B-ABORT-002 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `aborted.batches` | array | stage-dependent |
| `aborted.calls` | array | stage-dependent |
| `aborted.digestsAfter` | object | stage-dependent |
| `aborted.engine` | array | stage-dependent |
| `aborted.halt` | object | stage-dependent |
| `aborted.initialDigests` | object | stage-dependent |
| `aborted.modelCallsCompleted` | integer | stage-dependent |
| `aborted.process` | object | stage-dependent |
| `aborted.record` | object | stage-dependent |
| `aborted.submittedRepresentation` | string | stage-dependent |
| `aborted.submittedRepresentationSha256` | sha256 | stage-dependent |

### B-FAILURE-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `failureEvent.drawsCompleted` | integer | no |
| `failureEvent.modelCalls` | integer | no |
| `failureEvent.nextDraw` | object | stage-dependent |
| `failureEvent.nextDraw.arm` | string | no |
| `failureEvent.nextDraw.context` | string | no |
| `failureEvent.nextDraw.draw` | integer | no |
| `failureEvent.nextDraw.model` | string | no |
| `failureEvent.nextDraw.seq` | integer | no |
| `failureEvent.protocolSha256` | sha256 | no |
| `failureEvent.runId` | string | no |
| `failureEvent.scheduleSha256` | sha256 | no |
| `failureEvent.schema` | string | no |
| `failureEvent.state` | string | no |
| `failureEvent.stop` | object | stage-dependent |
| `failureEvent.timestamp` | UTC timestamp | no |
| `failureEvent.verificationStatus` | string | no |

### B-HANDOFF-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `failed.candidates` | array | no |
| `failed.error` | object | stage-dependent |
| `failed.error.message` | string | no |
| `failed.error.type` | string | no |
| `failed.pending` | object | no |
| `failed.stage` | string | no |
| `pending.draw` | object | no |
| `pending.rowSha256` | sha256 | no |
| `pending.schema` | string | no |

### B-SEAL-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `SEAL.driver` | string | no |
| `SEAL.finalization` | string | no |
| `SEAL.lifecycle` | object | no |
| `SEAL.lifecycle.id` | string | no |
| `SEAL.lifecycle.sha256` | sha256 | no |
| `SEAL.protocol` | object | no |
| `SEAL.protocol.id` | string | no |
| `SEAL.protocol.sha256` | sha256 | no |
| `SEAL.runId` | string | no |
| `SEAL.runSchema` | string | no |
| `SEAL.scheduleSha256` | sha256 | no |
| `SEAL.schema` | string | no |

### B-SEAL-002 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `SEAL.terminalState` | string | no |
| `SEAL.transition.event` | string | no |
| `SEAL.transition.from` | string | no |
| `SEAL.transition.n` | integer | no |
| `SEAL.transition.timestamp` | UTC timestamp | no |
| `SEAL.transition.to` | string | no |

### B-SEAL-003 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `SEAL.calls.dispatched` | integer | no |
| `SEAL.calls.file` | string | no |
| `SEAL.calls.records` | integer | no |
| `SEAL.calls.sha256` | sha256 | no |
| `SEAL.ledger.completed` | integer | no |
| `SEAL.ledger.file` | string | no |
| `SEAL.ledger.rowSha256` | sha256 array | no |
| `SEAL.ledger.rows` | integer | no |
| `SEAL.ledger.sha256` | sha256 | no |
| `SEAL.manifest.file` | string | no |
| `SEAL.manifest.journalEntries` | integer | no |
| `SEAL.manifest.sha256` | sha256 | no |

### B-SEAL-004 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `SEAL.counters` | object | no |
| `SEAL.counters.drawAborted` | integer | stage-dependent |
| `SEAL.counters.drawInterrupted` | integer | stage-dependent |
| `SEAL.counters.drawsCompleted` | integer | no |
| `SEAL.counters.drawsNotRun` | integer | no |
| `SEAL.counters.drawsScheduled` | integer | no |
| `SEAL.counters.healthChecks` | integer | no |
| `SEAL.counters.modelCalls` | integer | no |
| `SEAL.health.checkSha256` | sha256 array | no |
| `SEAL.health.checks` | integer | no |
| `SEAL.health.serverBefore` | sha256 | stage-dependent |
| `SEAL.sideArtifacts` | object | no |
| `SEAL.sideArtifacts.aborted-draw.json` | sha256 | stage-dependent |
| `SEAL.sideArtifacts.failure-event.json` | sha256 | stage-dependent |
| `SEAL.sideArtifacts.server-before.json` | sha256 | stage-dependent |

### B-EXECUTION-001 field types

| Field | Type | Nullable within applicable stage |
| --- | --- | --- |
| `analysis.abortedVerification` | object | no |
| `analysis.armDifferences` | array | no |
| `analysis.completedVerification` | string | no |
| `analysis.draws` | array | no |
| `analysis.executions` | object | no |
| `analysis.failureVerification` | object | no |
| `analysis.lifecycle` | object | no |
| `analysis.pairs` | array | no |
| `analysis.table` | array | no |

## Retained response consistency

B-CALL-002 requires a nonempty transport error and an absent/empty body when HTTP status is absent, including unknown-draw journal paths. B-HEALTH-003 binds valid JSON bytes to their category: duplicate keys and nonfinite values use the existing `invalid_json` category. The trusted health exception families and observation format are unchanged. These supplementary archived witnesses are in the regression suite.

For B-ABORT-002, a null draw call array still requires full-analysis journal-prefix replay: reconstruct the frozen workflow requests from retained journal outcomes, compare every request hash, and reject extra outcomes beyond that workflow. Missing draw results are never invented.
