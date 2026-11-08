# Run lifecycle contract — dreamteam.run-lifecycle/v1

DRAFT, NOT LOCKED. Independent technical re-review: PENDING. Human review:
PENDING (presentation only). Protocol lock and live inference: NO GO.

The machine-readable contract is
[analysis/alternatives/lifecycle_v1.json](../../analysis/alternatives/lifecycle_v1.json)
(loaded and self-checked by [lifecycle_spec.py](../../analysis/alternatives/lifecycle_spec.py)).
This page explains it; where the two differ, the JSON governs.

The contract is written before the recorder and verifier that implement it.
Both load the same file. The recorder may change state only through the
transition table. The verifier never takes the recorder's claimed status as
truth: it rebuilds the journal through the same table, recomputes every
counter and stop from the persisted artifacts, derives the terminal state
those artifacts support, and classifies the archive INVALID on any
disagreement. The two share the contract and its pure definitions (health
body categories, timestamp syntax), not transition or classification code.

## States

| State | Meaning |
|---|---|
| CREATED | archive directory, empty ledger (`draws.json`) and initial manifest (`run.json`) persisted; nothing fallible has run |
| PREFLIGHT | limits configured; the initial health read (when the server is observed) in progress |
| READY | between draws; the next scheduled draw starts once its round health check and the draw guard pass |
| HEALTH_CHECK | the round-boundary health read before the next draw in progress |
| DRAW_ACTIVE | the draw guard passed for `schedule[committed]`; the draw is executing, or returned and not yet committed |
| DRAW_RECORDED | the draw's row is durable in the ledger |
| CLEANUP | no further draw starts; pooled engine sessions are being released |
| FINALIZING | the FINALIZING marker is durable; final artifacts and the final manifest are being written |
| COMPLETE | terminal: every scheduled draw completed, no stop, cleanup succeeded, finalization durable |
| PARTIAL | terminal: finalized after a recorded stop or a cleanup failure |
| INVALID | verifier-only classification of any archive violating an invariant; never written by the recorder |

## Transitions

Every legal transition; anything else is illegal (the recorder refuses it and
the verifier classifies the archive INVALID, invariant L14).

| From | Event | To |
|---|---|---|
| — | create | CREATED |
| CREATED | begin_preflight | PREFLIGHT |
| CREATED | stop | CLEANUP |
| PREFLIGHT | preflight_passed | READY |
| PREFLIGHT | stop | CLEANUP |
| READY | begin_health_check | HEALTH_CHECK |
| READY | begin_draw | DRAW_ACTIVE |
| READY | stop | CLEANUP |
| HEALTH_CHECK | health_passed | READY |
| HEALTH_CHECK | stop | CLEANUP |
| DRAW_ACTIVE | commit_draw | DRAW_RECORDED |
| DRAW_ACTIVE | stop | CLEANUP |
| DRAW_RECORDED | continue | READY |
| DRAW_RECORDED | schedule_done | CLEANUP |
| DRAW_RECORDED | stop | CLEANUP |
| CLEANUP | cleanup_succeeded | FINALIZING |
| CLEANUP | cleanup_failed | FINALIZING |
| FINALIZING | finalize_complete | COMPLETE |
| FINALIZING | finalize_partial | PARTIAL |

The journal (`run.json` `journal`) records every transition in order with its
event, source and target state, timestamp and event detail (draw sequence,
row SHA-256, status and call count for `commit_draw`; the health-check index
for health events). Every checkpoint is a whole-manifest atomic replacement.

## Stop codes and the states they may come from

| Code | States | Evidence |
|---|---|---|
| health | PREFLIGHT, HEALTH_CHECK | that state's health check has a non-HEALTHY observation; request GET /health; no exception |
| server_change | HEALTH_CHECK | HEALTHY observation whose server signature differs from the initial one |
| wall_clock | READY, DRAW_RECORDED | READY: draw-guard halt (stage draw_start); DRAW_RECORDED: the committed row is interrupted with that halt |
| draw_call_limit, call_budget, transport | DRAW_RECORDED | the committed row is interrupted with that halt (stage request_start) |
| driver / operator | CREATED, PREFLIGHT, READY, HEALTH_CHECK, DRAW_ACTIVE, DRAW_RECORDED, CLEANUP | exception `{type, message}`; operator iff KeyboardInterrupt. DRAW_ACTIVE requires the aborted artifact; DRAW_RECORDED requires a completed row; CLEANUP only through `cleanup_failed` with no earlier stop |

There is at most one stop (L19). `run.stop`, `run.failureEvent.stop`,
`failure-event.json` and the journal's stop transition agree on code and state.
A stop interrupted during a health read carries the attempted request and no
observation; nothing is invented.

## Aborted draws

An aborted draw is the next scheduled draw, never a ledger row (L05). Its
`failureStage` fixes what the artifact may contain (L06):

| failureStage | engineInitialization | Evidence | Verification |
|---|---|---|---|
| draw_setup | NOT_STARTED | no calls, engine log or digests | UNVERIFIED |
| draw_unrecorded | UNKNOWN | none (the draw was entered but raised without its own evidence) | UNVERIFIED |
| engine_open | FAILED | none | UNVERIFIED |
| model_setup | OPENED | none | UNVERIFIED |
| initial_digests | OPENED or READY | at most the digests request; no calls | UNVERIFIED |
| workflow | READY | recorded calls and engine log | transport replay when the last call failed without a body, else UNVERIFIED |
| final_digests | READY | recorded | UNVERIFIED |
| engine_close | READY | recorded; no cleanupFailure (the close failure is primary) | UNVERIFIED |

`cleanupFailure` is admissible only on an aborted artifact beside a different
primary failure; on a completed or interrupted row it is a contradiction.

## Health evidence

The successor probe records a `dreamteam.health-observation/v1` for each read:
request, outcome, category, HTTP status, exception `{type, reasonType}`, the
response body (bytes, SHA-256, base64) and start/end time. Categories come from
the exception's class or from the body bytes, never from message text.

* Body categories: healthy (HEALTHY); unhealthy_status (UNHEALTHY);
  oversized_body, non_utf8_body, invalid_json, not_an_object, missing_status,
  contradictory_body (MALFORMED). `{"status":"healthy","error":…}` is
  contradictory_body, never HEALTHY.
* Read failures (READ_FAILED): http_status, connection_refused,
  connection_reset, remote_disconnected, incomplete_read, timeout,
  http_protocol, url_error, os_error. A URLError takes its reason's category
  when listed. Every read failure stops the run (`health`); cause UNVERIFIED.
* The verifier re-classifies every observation from its stored body or
  exception type; any disagreement is INVALID evidence (L07).

Ordinary runs observe the server; only an explicitly checked record-only
fixture may run unobserved. Health checks follow `healthSchedule`: one in PREFLIGHT, then one in
HEALTH_CHECK before `schedule[i]` whenever i = 0 or the (model, round)
changes. `server-before.json` equals the initial state.

## Counters (L03, L04)

`drawsScheduled = drawsCompleted + [drawInterrupted] + [drawAborted] + drawsNotRun`.
`modelCalls` equals the recorded exchanges in ledger rows plus the aborted
artifact, failed attempts included. `healthChecks` counts recorded checks.
The verifier recomputes each counter from the artifacts; the manifest's values
are compared, never trusted.

## Interrupts, reconciliation and fail-closed behaviour

* SIGINT is deferred everywhere except inside the interruptible regions: the
  health read, draw execution and session release (and even there when it is
  handled in the recorder's own frame, before the external call starts or
  after it returned). A deferred interrupt is
  raised at the next transition boundary while the run is still running; once
  the run has stopped it cannot interrupt cleanup bookkeeping or finalization.
* An exception or interrupt outside persistence and outside the
  stop/cleanup/finalization sequence is handled: the recorder reconciles the
  committed ledger and records stop driver/operator at the current state.
* Reconciliation: a draw that already returned is committed before the stop;
  a row already durable in `draws.json` is recorded as `commit_draw`
  (reconciled). No aborted artifact is invented for a completed draw and no
  row is invented for a draw that did not return.
* Fail closed: a failed persistence operation, or any exception inside the
  stop/cleanup/finalization sequence other than the session release itself,
  propagates as a recording failure. The archive stays unfinalized or keeps
  its FINALIZING marker and is INVALID. It is never reported COMPLETE.

## Finalization (L10, L16)

1. Write the `FINALIZING` marker (file and directory sync).
2. Write `failure-event.json` when a stop exists.
3. Write `run.json` in its terminal state, with SHA-256 of every archive file.
4. Remove the marker.
5. Sync the directory.

COMPLETE or PARTIAL is reported only after all five succeed. If the final
directory sync fails after the marker was removed, the recorder restores the
marker before failing. A remaining marker makes the archive INVALID. The
contract claims atomic same-directory replacement per file and fsync where
supported; it claims no multi-file transaction and no power-loss or
filesystem-ordering guarantee.

## Timestamps (L13)

Required: `run.started`, `run.ended`, every journal timestamp, every health
observation's start and end, the cleanup and finalization timestamps, the
failure event timestamp, and every recorded model call's `sent`. Each must be
a valid UTC RFC 3339 time. Order: started ≤ journal (non-decreasing) ≤ ended;
ended equals the finalize transition; observations lie inside their health
check; the failure event time equals the stop transition. Timestamps are
metadata, excluded from deterministic replay identities; a missing timestamp
is never inferred. `run.provenance.timestamps` is SYSTEM_UTC for ordinary runs
and SYNTHETIC_FIXTURE only for reproducible record-only fixtures, which the
verifier accepts only when explicitly asked to check a fixture.

## Verification statuses

* VERIFIED — a completed or interrupted row reconstructed exactly by
  model-free replay.
* UNVERIFIED — truthfully recorded but not independently reconstructable
  (external health causes, operator interrupts, most aborted draws).
* INVALID — the archive violates an invariant; analysis refuses it.

## Terminal conditions

* COMPLETE: every scheduled draw completed, journal tail `schedule_done`,
  `cleanup_succeeded`, `finalize_complete`; no stop, failure event, aborted
  artifact, interrupted row, failed cleanup or non-HEALTHY observation; marker
  absent; all artifact hashes match (L09, L10).
* PARTIAL: finalized with a valid stop (or a cleanup-created stop); every
  completed row is verified by replay; no complete-run arm comparison or pair
  summary is produced (L11).
* INVALID: anything else, including an unreadable or unfinalized archive.

## Invariants

L01 schedule prefix · L02 completed evidence · L03 draw conservation ·
L04 model-call reconciliation · L05 aborted is not completed · L06 failure
stage · L07 health evidence · L08 cleanup distinguishable · L09 no unresolved
failure in COMPLETE · L10 COMPLETE coverage and finalization · L11 PARTIAL
suppression · L12 INVALID never passes · L13 timestamps · L14 unknown
transitions fail closed · L15 identity binding · L16 artifact set and hashes ·
L17 journal–ledger binding · L18 no resume · L19 single consistent stop.
The exact rules are in the JSON.

## Fault injection

The contract declares every operation's injection phases and fault kinds
(`faultInjection.operations`), the boundaries where those operations occur,
and the expected outcome rule (`expectedRule`). The fault matrix is generated
from these declarations and an execution trace of the base scenarios; it is
not a list of earlier audit witnesses.
