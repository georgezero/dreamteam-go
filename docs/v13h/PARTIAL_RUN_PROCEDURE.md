# Partial-run successor — dreamteam.alternatives-partial-run/v4

Independent re-review PENDING. Human review PENDING. Preserve the entire run
without resume, overwrite, replacement draws or complete-run comparisons.

The predecessor row, request/body/hash, response, model-call, guard and exact
execution checks remain. The successor additionally checks run UUID, protocol
and schedule binding, finalized lifecycle, event identity/stage, scheduled draw,
nonnegative counts, stop relationships, cleanup and required aborted evidence.
An unknown stop code or unsupported health-read error is INVALID and the
analyzer fails; INVALID archives never produce a passing analysis.

Supported failed-read objects have exactly one key, error, whose value is
URLError, TimeoutError, ConnectionRefusedError, ConnectionResetError, HTTPError,
JSONDecodeError or UnicodeDecodeError. These are explicit production-getter
failure shapes, not server replies. Objects combining error and status, unknown
error names or additional error-object fields are refused. Actual unhealthy
status responses are accepted separately. JSON responses lacking a valid status
are retained as malformed responses; failed JSON decoding retains JSONDecodeError.
A healthy response cannot justify a health stop. The attempted request is
GET /health and the stage is initial_health or between_health. No response is
invented when a read is interrupted; its exception is recorded instead.

Driver/operator boundary exceptions require a valid run-failure event. At the
draw stage they also require the identified next scheduled aborted artifact;
boundary interruptions require no invented aborted draw. Cleanup-only failure
may follow all completed draws and still yields PARTIAL. A primary failure plus
cleanup failure keeps the primary event and separately records cleanup failure.

VERIFIED means deterministically reconstructed and matched completed-draw
behavior. Reconstructible aborted transport exchanges retain their reconstruction
result, with external cause unverified. UNVERIFIED means truthfully recorded but
not independently reconstructable; supported health and lifecycle causes remain
UNVERIFIED. Structural validation cannot establish that an external outage
actually occurred. A supported PARTIAL archive may contain UNVERIFIED evidence.
Its armDifferences and pairs are always empty, even if cleanup failed after the
last draw. IN_PROGRESS manifests and inconsistent cross-file checkpoints are
not valid finalized archives and cannot be represented as COMPLETE.

Offline analysis uses analysis/alternatives/v13h.py analyze with the successor
protocol and the unchanged docs/benchmark/v13g-dev/contexts input. No command
performs inference. Hard crashes need not leave a finalized manifest.
