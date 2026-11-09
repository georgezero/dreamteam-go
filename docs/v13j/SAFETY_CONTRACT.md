# Run safety contract — dreamteam.run-lifecycle/v2

DRAFT, NOT LOCKED. Independent technical re-review: PENDING. Human
presentation review: PENDING (presentation only, docs/v13i/human-review at
ee987a1). Protocol lock: NO GO. Live inference: NO GO. V0.13.0: NO GO.

The machine-readable contract is
[analysis/alternatives/lifecycle_v2.json](../../analysis/alternatives/lifecycle_v2.json),
loaded and self-checked by
[lifecycle_v2_spec.py](../../analysis/alternatives/lifecycle_v2_spec.py).
Where this page and the JSON differ, the JSON governs. It supersedes
dreamteam.run-lifecycle/v1 (V0.13i), which stays unchanged for its own
evidence. This contract was committed before any recorder change.

## 1. Principle and two separate properties

**Never falsely COMPLETE.** The completion seal (`SEAL.json`) is the only
evidence that a run reached a terminal state. No archive is classified
COMPLETE or PARTIAL because a file or marker is absent. A valid seal is
necessary and never sufficient: every semantic rule is verified
independently from the persisted files.

The contract separates **data integrity** (an accepted archive never states
anything false) from **run availability** (a run that failed is usable). An
archive safely classified INVALID may be unusable, but it has not produced a
false experimental conclusion.

| Rank | Defect | Disposition |
|---|---|---|
| 1 | False COMPLETE: an incomplete, contradictory or insufficiently durable archive classified COMPLETE | release blocker |
| 2 | False acceptance of PARTIAL: a contradictory archive accepted because it is PARTIAL or its cause UNVERIFIED | release blocker |
| 3 | A genuine handled failure classified INVALID (safely rejected, nothing misrepresented, nothing resumed) | reliability defect; documented availability limitation |
| 4 | A completed draw or a dispatched model call missing from the persisted evidence, even of a rejected run | major defect |

## 2. Acceptance rules

**COMPLETE** requires all of: a valid seal with terminal state COMPLETE;
every seal hash equal to the bytes on disk; every scheduled draw recorded as
a completed ledger row in schedule order; every dispatched model call
accounted (each dispatch intent has exactly one outcome, equal to the call in
its draw); every required health check exactly once, valid and HEALTHY with
an unchanged server signature; no stop, failure event, aborted draw,
interrupted row or unresolved dispatch; cleanup SUCCEEDED; every required
artifact present and every forbidden artifact absent; operator interrupts
only with disposition AFTER_SCHEDULE; independent model-free replay of every
completed row passing.

**PARTIAL** requires: a valid seal with terminal state PARTIAL; every seal
hash matching; a truthful stop record (one admissible stop with exactly the
fields its code and state require, or a cleanup-created stop); a valid
completed-draw prefix; no contradictory evidence under the same structural,
transition, accounting, health, cleanup, interrupt and timestamp rules as
COMPLETE; an explicit verification status (UNVERIFIED) for every event that
cannot be reconstructed; no complete-run comparison. UNVERIFIED never excuses
a structural contradiction.

**INVALID**: any archive that is unsealed, malformed, incomplete,
contradictory, unsupported, or unverifiable beyond the PARTIAL contract.

**Threat model.** The seal binds the archive's bytes so that partial
failure, accidental change and any contradiction, including one whose
dependent hashes were recomputed, are detected. It is a self-hash, not a
signature: a writer able to rewrite every file consistently can remove
evidence that leaves no other trace. Model-free replay exposes an erased
exchange that the draw's engine log or decision record still reflects.
Detecting a fully consistent rewrite would need an external anchor of the
seal (an owner decision; not provided).

## 3. Safe-point interrupts

The SIGINT handler only counts the signal; it never raises. A pending
interrupt is examined only at declared safe points:

| Safe point | Kind | State | Action when an interrupt is pending |
|---|---|---|---|
| before_dispatch | STOP | DRAW_ACTIVE | before the dispatch intent exists: the draw is aborted at stage `workflow` (failure type OperatorStop); calls already made are kept |
| after_transport_returns | NOTE | DRAW_ACTIVE | none: the response must first reach the call journal and the draw's evidence |
| after_call_persisted | NOTE | DRAW_ACTIVE | none: the draw must still record the call |
| after_draw_persisted | STOP | DRAW_RECORDED | after commit_draw is durable (completed row): operator stop |
| between_draws | STOP | READY | operator stop |
| before_health_check | STOP | PREFLIGHT, READY | operator stop; no health check appended |
| after_health_check | STOP | PREFLIGHT, HEALTH_CHECK | after a HEALTHY observation (health or server-change stops take precedence): operator stop with the check kept |
| before_cleanup | RECORD | CLEANUP | recorded; cleanup always runs |
| after_cleanup | RECORD | CLEANUP | recorded |
| before_finalization | RECORD | CLEANUP | recorded; the last point whose interrupts reach the archive |

Dispositions: STOP (the operator stop), AFTER_STOP (another stop was already
recorded; it stays primary), AFTER_SCHEDULE (after `schedule_done` with no
stop: a post-completion operator event, admissible in COMPLETE). An
interrupt arriving after `before_finalization` cannot alter an archive whose
content is fixed; it is reported in the public result as `unarchived`.

Inside the two transport windows (from the durable dispatch intent to the
transport call, and from the transport's return to the draw's own evidence)
no recorder exception propagates: a recorder fault there is deferred to the
next STOP point as a driver stop, and a persistence fault poisons the archive
so it is never sealed. Engine subprocesses run in their own session so that a
terminal SIGINT reaches only the recorder; the engine and its requests are
unchanged.

The public result of `run_schedule` carries `interrupts {signalsReceived,
archived, unarchived, stopSafePoint, postCompletion}`; a `RecordingError`
carries the same summary. No interrupt is silently discarded.

## 4. Durable call and draw accounting

`calls.jsonl` (dreamteam.call-journal/v1) receives, appended and fsynced
before the recorded operation continues: a `dispatch` record (run index `k`,
draw, per-draw index `n`, timestamp) immediately before the request is handed
to the transport, then exactly one outcome: `returned` (request and response
SHA-256, status, error, body) or `raised` (the transport raised; the
server-side outcome is unknown, UNRESOLVED). A dispatch without an outcome is
an unresolved call: kept as evidence in an unsealed archive, a contradiction
in a sealed one. `modelCalls` counts dispatches; a draw whose evidence was
lost (`draw_unrecorded`) records `calls: null`, never zero calls. Each draw's
row remains the unchanged draw/v4 record, committed by atomic replacement of
`draws.json` and bound by `commit_draw`.

## 5. Completion seal and finalization (dreamteam.run-finalization/v1)

1. Release pooled engine sessions (always attempted); record interrupts at
   before_cleanup, after_cleanup, before_finalization.
2. Write `run.json` with the cleanup transition (state FINALIZING), then
   `failure-event.json` if a stop exists: each a same-directory temporary
   file, flush, file fsync, atomic replace, directory fsync.
3. Refuse to seal if any persistence operation of the run failed.
4. Re-read every archive file and compute the seal from those bytes.
5. Write the seal to a temporary file in the archive directory; flush; fsync.
6. **Commit point:** `os.replace` of the temporary file to `SEAL.json`. This
   is the last required operation.
7. Best effort only: fsync the directory. Its result is reported
   (`commit.directorySync`) and never changes the archive or its classification.

The seal (dreamteam.completion-seal/v1) contains: finalization contract
version; lifecycle id and SHA-256; run schema and driver; run identity;
protocol identity and SHA-256; schedule SHA-256; terminal state and the
terminal transition (FINALIZING to COMPLETE or PARTIAL, `n`, timestamp);
manifest SHA-256 and journal length; ledger SHA-256, row count, completed
count and per-row canonical SHA-256; call-journal SHA-256, record and
dispatch counts; health-check count, per-check canonical SHA-256 and
`server-before.json` SHA-256; side-artifact hashes (null meaning the file
must be absent); counters. No seal: not terminal (INVALID). A malformed or
contradictory seal: INVALID. The terminal transition exists only in the seal;
`run.json` never claims a terminal state.

### Durability boundary

* **Process crash: claimed.** If the recording process is terminated at any
  instant, SIGKILL included, the archive is either unsealed (INVALID, with
  every durable row and call-journal record kept) or sealed with every byte
  the seal binds already written. A process killed after the rename leaves a
  sealed archive; that is correct because nothing required remains. Tested
  with real SIGKILL at every operation boundary the recorder reaches.
* **Power loss: not claimed.** Design intent only: every bound file and its
  directory entry is fsynced before the seal is written, and the seal is
  fsynced before its rename. On a filesystem that honours fsync, a seal that
  survives power loss binds durable content, and a lost rename reverts to an
  unsealed, INVALID archive. Hardware or filesystems that acknowledge fsync
  without persisting can defeat this. No test covers power loss. The
  conservative outcome of any uncertain case is INVALID, never COMPLETE.
* There is no multi-file transaction; consistency comes from the single
  commit point plus the verifier's hash and semantic checks.

## 6. Per-stage evidence (DT13I-04)

Stops: every field of `stop` (`failure`, `evidence`, `request`,
`healthCheck`, `beforeSignature`, `safePoint`) is fixed for each code and
state (`stopCodes`). Operator stops come only from a STOP safe point and
carry it; driver stops carry `{type, message}`; KeyboardInterrupt raised as
an exception is a driver failure, not an operator stop.

Aborted draws (`abortedDraw.stages`), every field fixed:

| Stage | Init | halt | initialDigests | record | batches | engine | digestsAfter | calls | submitted | process | cleanupFailure |
|---|---|---|---|---|---|---|---|---|---|---|---|
| draw_setup (recorder) | NOT_STARTED | null | null | null | null | [] | null | [] | null | null | forbidden |
| draw_unrecorded (recorder) | UNKNOWN | null | null | null | null | null | null | null (journal only) | null | null | forbidden |
| engine_open | FAILED | null | null | null | null | [] | null | [] | null | no-model | forbidden |
| model_setup | OPENED | null | null | null | null | [] | null | [] | null | no-model | optional |
| initial_digests | OPENED | null | null | null | null | one digests request | null | [] | null | no-model | optional |
| workflow | READY | null | context | null | null | non-empty | null | recorded = journal | null | derived | optional |
| final_digests | READY | null or halt | context | null or object (xor halt) | null | non-empty | null | recorded = journal | iff record | record's or derived | optional |
| engine_close | READY | null or halt | context | null or object (xor halt) | iff record | non-empty | object | recorded = journal | iff record | record's or derived | forbidden |

"Derived" process evidence is recomputed by the verifier from the artifact's
own engine log and calls. Transition predicates (`transitionPredicates`) are
enforced for every journal entry: `schedule_done` only after the last
scheduled draw completed; `continue` only with draws remaining; each required
health visit exactly once and no other; `begin_draw` only for the next
scheduled draw after its required check. A side artifact that must be absent
must not exist; a file containing JSON null is a contradiction, not absence.

## 7. Health exception families (DT13I-03)

Exception evidence is `{type, reasonType, reasonKind}` with exact
`module.qualname` class names. The recorder normalizes the exception object
by its exact class; the verifier checks the recorded strings against the
contract's closed tables and never trusts a name outside them.

| Family (category) | Exact types | Wrappable in URLError |
|---|---|---|
| http_status | urllib.error.HTTPError (non-2xx status) | no |
| connection_refused | builtins.ConnectionRefusedError | yes |
| connection_reset | builtins.ConnectionResetError, ConnectionAbortedError, BrokenPipeError | yes |
| remote_disconnected | http.client.RemoteDisconnected | yes |
| incomplete_read | http.client.IncompleteRead | yes |
| timeout | builtins.TimeoutError (socket.timeout is an alias) | yes |
| http_protocol | http.client.HTTPException and its listed subclasses (NotConnected, BadStatusLine, LineTooLong, …) | yes |
| network_os_error | builtins.OSError, ConnectionError, socket.gaierror/herror, ssl.SSLError and listed subclasses | yes |
| url_error | urllib.error.URLError with a text reason | — |

Any other exception, including KeyboardInterrupt, ValueError, an unlisted
OSError subclass or a URLError with an unlisted reason, is not a health read
failure: it propagates and stops the run as a driver failure with the
attempted request and no observation. An observation that disagrees with
itself is refused by the recorder (driver stop, value kept as rejected) and
rejected by the verifier.

## 8. Timestamp grammar (DT13I-06)

Whole-string ASCII match of
`[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z`
plus a valid calendar date and time. Trailing newlines, non-ASCII digits,
offsets other than Z, lower-case separators and impossible dates are
rejected. Required intervals must not be reversed (`timestamps.order`).
Timestamps stay outside deterministic replay identities.

## 9. Invariants

S01 seal present · S02 seal identity · S03 seal hashes · S04 seal terminal
state · L01 schedule prefix · L02 completed evidence · L03 draw conservation ·
L04 call accounting · L05 aborted is not completed · L06 per-stage evidence ·
L07 health evidence · L08 cleanup · L09 no unresolved failure in COMPLETE ·
L10 COMPLETE coverage · L11 PARTIAL suppression · L12 INVALID never passes ·
L13 timestamps · L14 transitions and predicates · L15 identity · L16 artifact
set · L17 journal–ledger binding · L18 no resume · L19 single stop · L20
interrupt evidence.

## 10. Fault injection

Operations, phases and fault kinds (exception, real signal, single
persistence fault, persistent filesystem failure, transport, guard halt,
SIGKILL) and the expected-outcome rule are declared in `faultInjection`. The
fault matrix is generated from these declarations and execution traces; the
signal harness delivers real SIGINT and SIGKILL to real child processes.
