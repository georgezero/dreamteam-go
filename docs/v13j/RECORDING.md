# Run recording — dreamteam.alternatives-run/v6

DRAFT, NOT LOCKED. Independent technical re-review PENDING. Human
presentation review PENDING (and not concerned with recording). Engine
0.13.0-dev.2 unchanged. Draw records remain dreamteam.alternatives-draw/v4,
produced by the unchanged V0.13h `run_draw`.

The run is recorded by [v13j.py](../../analysis/alternatives/v13j.py)
through the safety contract [SAFETY_CONTRACT.md](SAFETY_CONTRACT.md)
(`lifecycle_v2.json`). Every state change before finalization is a journal
entry written by a whole-manifest atomic replacement; the terminal transition
exists only in the completion seal.

## Archive files (`<run>/live/`)

| File | Content |
|---|---|
| run.json | schema `dreamteam.alternatives-run/v6`, driver `dreamteam-v13j-driver/6`, lifecycle `{id, sha256}`, protocol identity and SHA-256, schedule SHA-256, UUID4 `runId`, authorization, `provenance {timestamps, runId}`, `serverObserved`, `started`, non-terminal `state` (FINALIZING when finished), `journal`, `counters`, `healthChecks`, `stop`, `failureEvent`, `cleanup {status, failure, timestamp}`, `interrupts {signal, events}`, `verificationStatus` UNVERIFIED, `resume` never |
| draws.json | authoritative ledger: committed draw/v4 rows (completed, or the last interrupted by a guard halt), a schedule prefix |
| calls.jsonl | `dreamteam.call-journal/v1`: `dispatch` intent, then `returned` or `raised` outcome, per model call, each appended and fsynced |
| server-before.json | the full initial server state, when the initial read kept an observation |
| aborted-draw.json | the next scheduled draw's aborted draw/v4 artifact, when the stop is from DRAW_ACTIVE |
| failure-event.json | `dreamteam.run-failure/v3`, identical to `run.failureEvent`, when a stop exists |
| SEAL.json | `dreamteam.completion-seal/v1`, written last; the commit point |

No other file or directory is admissible. The V0.13i FINALIZING marker is
not written; its presence makes an archive INVALID.

## The public result — dreamteam.run-result/v1

`run_schedule` returns `{state, runId, archive, manifest, seal, interrupts,
commit}`. `interrupts` = `{signalsReceived, archived, unarchived,
stopSafePoint, postCompletion}`: every SIGINT the latch counted, the events
written to the archive, the count received after the final manifest was
written (reported only here), the safe point of an operator stop, and whether
an operator event followed schedule completion. `commit` = `{sealRenamed,
directorySync (SYNCED, UNSUPPORTED or FAILED), postCommitError}`: the
best-effort steps after the commit point, which never change the archive. A
`RecordingError` (the archive is unsealed and INVALID) carries the same
`interrupts` summary.

## Health observations — dreamteam.health-observation/v2

As v1 (request, outcome, category, HTTP status, body `{bytes, sha256,
base64}`, start and end time, UNVERIFIED), with exception evidence `{type,
reasonType, reasonKind}` in exact `module.qualname` form. Families are
assigned by the exception's exact class from the contract's closed tables; a
URLError takes its reason's family when that class is listed and wrappable,
or `url_error` for a text reason. Any other exception (KeyboardInterrupt,
ValueError, unlisted OSError subclasses, a URLError with an unlisted reason)
propagates: the run stops as a driver failure with the attempted request and
no observation. The recorder refuses an observation that disagrees with
itself (category, outcome, status, exception, body, hash, timestamps) before
it can justify continuing; it keeps it as `rejected` and stops the run
(driver). Engine sessions are spawned in their own session, so a terminal
SIGINT reaches only the recorder.

## Stops, failure event, cleanup, interrupts

`stop` = `{code, state, message, failure, evidence, request, healthCheck,
beforeSignature, safePoint}`; every field is fixed by the code and state. An
operator stop comes only from a STOP safe point and names it; a driver stop
carries the exception `{type, message}` (a KeyboardInterrupt raised as an
exception is a driver failure). The failure event (`run-failure/v3`) binds
run, protocol and schedule identities, the stop, the affected draw, completed
draws, dispatched model calls and the stop's timestamp. `cleanup` records the
pooled session release, which always runs; a release failure never replaces
an earlier stop and otherwise creates a driver stop at CLEANUP.
`interrupts.events` = `[{n, safePoint, state, signals, disposition,
timestamp}]` with disposition STOP, AFTER_STOP or AFTER_SCHEDULE.

## Durable accounting and reconciliation

The draw guard's before-request check runs, in order: the `before_dispatch`
safe point (an operator stop aborts the draw at stage `workflow` before any
part of the request exists), the V0.13h limits (a halt), and the durable
`dispatch` intent. The transport wrapper appends the outcome before returning
the response to the unchanged V0.12b PointModel, which then records the call
in the draw. Inside those windows no recorder fault propagates (deferred to
the next STOP point; persistence faults poison the archive). A handled
exception reconciles the committed ledger: a returned draw is committed, a
durable row is recorded as `commit_draw` (reconciled); the draw's own aborted
evidence is kept; without it, `draw_setup` (not entered: no calls) or
`draw_unrecorded` (entered: `calls` and `engine` null, the call journal being
the only call evidence). Nothing is reported as a known zero.

## Persistence and the commit point

Each JSON artifact is written to a same-directory temporary file, flushed,
fsynced and atomically replaced, then the directory is fsynced (EINVAL and
ENOTSUP tolerated; no directory sync on Windows). A failure inside any
persistence operation poisons the run: the recorder fails closed and never
seals. Finalization writes the final manifest and failure event, re-reads
every archive file, checks it equals what was written, computes the seal,
writes it to a temporary file, fsyncs it and renames it to `SEAL.json`: the
commit point. If the rename itself reports an error, the recorder treats the
run as committed only when `SEAL.json` holds exactly the seal it wrote. The
directory fsync after the rename is best effort and reported. See the
durability boundary in [SAFETY_CONTRACT.md](SAFETY_CONTRACT.md#durability-boundary).

A hard kill leaves an unsealed archive (INVALID) whose ledger rows and
call-journal records remain for audit; no automatic recovery or resumption
exists.

## Timestamps and identities

Ordinary runs record actual UTC timestamps (`%Y-%m-%dT%H:%M:%S.%fZ`) and a
fresh UUID4, `provenance` SYSTEM_UTC/UUID4. Reproducible record-only fixtures
declare SYNTHETIC_FIXTURE; the verifier refuses it unless asked to check a
fixture. Timestamps and run identities are metadata, excluded from
deterministic replay identities.
