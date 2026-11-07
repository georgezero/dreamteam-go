# Partial-run contract — dreamteam.alternatives-partial-run/v2

Independent re-review: PENDING. No run has been authorized.
A halted, aborted or interrupted run remains PARTIAL. Preserve the complete
run directory and aborted-draw.json. Never resume, overwrite, replace missing
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
