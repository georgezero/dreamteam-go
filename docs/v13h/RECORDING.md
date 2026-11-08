# Recording successor

DRAFT, NOT LOCKED. Independent re-review PENDING; human review PENDING.
The engine remains 0.13.0-dev.2. Decision records and all model-visible static
sources retain their prior identities. Draw/v4 changes only cleanup failure
preservation; run/v4 and analysis/v5 add the enclosing lifecycle.

run.json binds a UUID runId, protocol identity/hash and SHA-256 of the canonical
schedule JSON. currentStage is run_setup, initial_health, between_health,
draw_guard, draw, cleanup or finished. lifecycle is IN_PROGRESS or FINALIZED.
Counters distinguish completed draws, an interrupted row, an aborted artifact,
not-run draws and recorded model attempts, including failed attempts. Health
reads consume no model calls. Completed is never inferred from planned work.

failure-event.json mirrors run.failureEvent exactly. run-failure/v1 binds run,
protocol and schedule identities, stage, scheduled next/attempted draw, completed
prefix, call count, stop object, exception, attempted request and timestamp.
An interrupted row names its own attempted draw. Cleanup after the last draw
has nextDraw null. Cleanup has SUCCEEDED or FAILED plus a separate exception;
when primary failure already exists it is not replaced. Run cleanup describes the pooled opener.close() call only. Session cleanup errors
are separately retained in aborted-draw.cleanupFailure when a primary failure
already exists; a session-close-only failure is the aborted engine_close failure. No cleanup success is
inferred from an exception. Timestamps and runtime UUIDs are nondeterministic
metadata, excluded from reconstructed decision claims.

The atomic authoritative completed/interrupted ledger is draws.json. draws.jsonl
is a compatibility export, not the successor analyzer's source. An aborted
artifact remains separate. The analyzer requires the manifest, authoritative
ledger, required aborted artifact and matching failure event. Missing/truncated
JSON and unfinished checkpoints fail verification. No archive is auto-resumed.

Critical JSON uses a same-directory temporary file, flush, file fsync and
os.replace, then directory fsync on platforms supporting it. Only EINVAL and
ENOTSUP directory-sync limitations are tolerated; directory sync is omitted on
Windows. A pre-replacement interruption or failed replacement retains the old
complete JSON. A directory-sync error can occur after replacement. Filesystem
failure, hard kill or power loss can leave an unfinished checkpoint or missing
cross-file updates; these are rejected, never promoted to COMPLETE. Atomic
replacement of each file is not a multi-file transaction or a universal power
loss guarantee. Persistence failures propagate when evidence cannot be saved.
