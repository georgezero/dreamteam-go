# Run recording — dreamteam.alternatives-run/v5

DRAFT, NOT LOCKED. Independent technical re-review PENDING. Human
presentation review PENDING (and not concerned with recording). Engine
0.13.0-dev.2 unchanged. Draw records remain dreamteam.alternatives-draw/v4,
produced by the unchanged V0.13h `run_draw`.

The run is recorded by [v13i.py](../../analysis/alternatives/v13i.py)
through the lifecycle contract [LIFECYCLE.md](LIFECYCLE.md)
(`lifecycle_v1.json`). Every state change is a journal entry written by a
whole-manifest atomic replacement; the recorder cannot change state except
through the contract's transition table.

## Archive files (`<run>/live/`)

| File | Content |
|---|---|
| run.json | schema `dreamteam.alternatives-run/v5`, driver `dreamteam-v13i-driver/5`, lifecycle `{id, sha256}`, protocol identity and SHA-256, schedule SHA-256, UUID4 `runId`, authorization, `provenance {timestamps, runId}`, `serverObserved`, `started`/`ended`, claimed `state` and `status`, `journal`, `counters`, `healthChecks`, `stop`, `failureEvent`, `cleanup {status, failure, timestamp}`, `finalization {timestamp, artifacts}`, `verificationStatus` UNVERIFIED, `resume` never |
| draws.json | authoritative ledger: committed draw/v4 rows (completed, or the last interrupted by a guard halt), a schedule prefix |
| server-before.json | the full initial server state, when the initial read returned |
| aborted-draw.json | the next scheduled draw's aborted draw/v4 artifact, when the stop is from DRAW_ACTIVE |
| failure-event.json | `dreamteam.run-failure/v2`, identical to `run.failureEvent`, when a stop exists |
| FINALIZING | `dreamteam.finalization-marker/v1`, present only while finalization is incomplete |

No other file is admissible; V0.13h's draws.jsonl compatibility export and
legacy stop-event file are not written and make an archive INVALID.

## Health observations — dreamteam.health-observation/v1

`probe_health` issues one GET /health and records: request, outcome
(HEALTHY, UNHEALTHY, MALFORMED, READ_FAILED), category, HTTP status, exception
`{type, reasonType}`, response body `{bytes, sha256, base64}` (at most 64 KiB),
start and end time, verificationStatus UNVERIFIED. A response with a declared
length is read in full so that truncation raises IncompleteRead; other bodies
are read up to the limit. Categories come from the exception class or the body
bytes (contract `health`), never from message text; a URLError takes its
reason's class. An exception that is not a transport failure propagates and
stops the run as a driver failure; an interrupt propagates as an operator
stop. Nothing is substituted for a failed read.

The production `server_state` keeps the predecessor's server signature fields
(version, OpenAPI hash, experiment models) and replaces `health` with the
structured observation. The recorder refuses a state whose observation is
missing or inconsistent before it can justify continuing: it keeps the value
as `rejected` and stops the run (driver, "invalid health observation").

## Stops, failure event, cleanup

`stop` = `{code, state, message, failure, evidence, request, healthCheck,
beforeSignature}`. Codes and admissible states are in the contract. The
failure event binds run, protocol and schedule identities, the stop and its
state, the affected draw (aborted draw, interrupted row, or next scheduled
draw), completed draws, model calls and the stop transition's timestamp.
`cleanup` records the pooled session release separately; a release failure
never replaces an earlier stop; when there was none it creates a driver or
operator stop at CLEANUP equal to the release failure.

## Reconciliation and interrupts

SIGINT is deferred except inside health reads, draw execution and the session
release; even there, a signal handled in the recorder's own frame (before the
external call starts or after it has returned) is deferred, so it cannot
discard a returned draw or observation. A handled exception or interrupt first reconciles the committed
ledger with the recorder's memory: a draw that returned is committed, a row
already durable is recorded as `commit_draw` with `reconciled: true`, and
only an exception inside the draw produces an aborted artifact (with the
draw's own evidence, or a `draw_setup` / `draw_unrecorded` skeleton that
claims nothing). A disagreement between disk and memory fails closed.

## Persistence and finalization

Each JSON artifact is written to a same-directory temporary file, flushed,
fsynced and atomically replaced, then the directory is fsynced (EINVAL and
ENOTSUP directory-sync limitations tolerated; no directory sync on Windows).
Any failure inside that operation is a persistence failure and fails closed.
Finalization writes the FINALIZING marker, the failure event, the terminal
manifest with SHA-256 of every archive file, removes the marker and syncs the
directory; only then is COMPLETE or PARTIAL returned. If that last sync fails
the marker is restored. A remaining marker makes the archive INVALID.

Guarantees are limited to per-file atomic replacement and fsync where the
platform supports it. There is no multi-file transaction and no claim about
power loss or filesystem ordering. A hard kill (SIGKILL, SIGTERM) leaves an
unfinalized archive, which the verifier classifies INVALID while its
committed files remain on disk; no automatic recovery or resumption exists.

## Timestamps and identities

Ordinary runs record actual UTC timestamps (RFC 3339, microseconds) and a
fresh UUID4, `provenance` SYSTEM_UTC/UUID4. Reproducible record-only fixtures
declare SYNTHETIC_FIXTURE; the verifier refuses that provenance unless asked
to check a fixture. Timestamps and run identities are metadata, excluded from
deterministic replay identities.
