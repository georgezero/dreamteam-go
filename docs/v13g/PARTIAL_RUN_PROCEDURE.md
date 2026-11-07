# Partial-run contract — dreamteam.alternatives-partial-run/v3

Independent re-review: PENDING. No run has been authorized.
A halted, aborted or interrupted run remains PARTIAL. Preserve the complete
run directory and, when a draw was aborted, aborted-draw.json. Never resume, overwrite, replace missing
draws or report partial coverage as a complete experiment.

The executable verifier checks the exact schedule prefix, status, completed,
interrupted, aborted and remaining counts, nonnegative integer accounting,
protocol binding, model-call totals, per-draw and run call bounds, response
hashes and metadata, exact recorded request bodies, defined stop codes and
limit evidence. An aborted draw must be the next scheduled draw and its
artifact must match the context, arm, model and repetition. Missing or extra
aborted artifacts fail verification. The audit's aborted draw 999, remaining
count -77, 999 calls and invented stop code are all rejected.

Defined stops are draw_call_limit, call_budget, wall_clock, transport, health,
server_change, driver and operator. Limit stops must follow from their
recorded thresholds and call accounting. Health/material-change stops retain
their state evidence. A failed driver or operator interruption is recorded
as an external cause rather than treated as a completed draw.

Completed and interrupted exchanges replay through the actual workflow.
Recorded transport aborts are reconstructed where the request/response record
supports it; their external cause remains unverified. When an engine failure
or external interruption cannot be reproduced from recorded responses, the
aborted exchange is explicitly UNVERIFIED. It never contributes a fabricated
complete decision or execution. Partial analysis contains only observed
completed records and an aborted verification status; complete-run arm and
pair estimates remain absent.

The four-hour mechanism is a request-start guard, reserving the request's
full timeout. It does not interrupt blocked engine reads, arbitrary transports
or shutdown. No hard end-to-end wall-clock guarantee is claimed.

Driver/operator stops require the next scheduled aborted artifact in both
directions. Aborted skeletons bind initialization status, failure stage,
stopReason, modelCallsCompleted and UNVERIFIED status. Calls and responses
remain independently checked. Initialization failures with zero calls are
valid PARTIAL evidence, never reconstructed successes.

Between-draw health, material-server-change and draw-start wall-clock stops
require a dedicated stop-event.json, exactly mirrored by run.stopEvent. It
binds the next scheduled draw, completed prefix, total calls, supporting
state/guard evidence and verification status. External state is explicitly
UNVERIFIED_EXTERNAL_STATE; a guard can be EVIDENCE_CHECKED. Driver/operator
between-draw stops are unsupported. Events attached to completed, interrupted
or aborted runs are refused. The command verifies the separate event file.
Missing artifacts, invented event relationships, negative remaining counts,
impossible calls, unknown identities and contradictory statuses are refused.
