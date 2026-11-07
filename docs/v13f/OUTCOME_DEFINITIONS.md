# V0.13f outcome contract — dreamteam.alternatives-outcomes/v2

Status: DRAFT, NOT LOCKED. Independent re-review: PENDING.
This successor supersedes the V0.13d/V0.13e action estimand for future work.
Historical contracts and evidence retain their original bytes.

The estimand is the effect of assignment to the complete alternatives
interface on the distribution of immediate canonical submitted action batches,
conditional on reaching the comparison turn in these fixed contexts. The
interface comprises information, the candidate menu, selection shortcuts,
optional inspection, additional decision opportunities and model calls, and
presentation order. This design does not isolate an information-only effect.
Changed actions do not establish better decisions or improved outcomes.

## Four distinct action fields

| Field | Meaning |
|---|---|
| intendedAction | The parsed requested batches, resolving a listed selection; null when no usable intended batch exists |
| validatedAction | The final batches that passed validation, including an explicitly valid empty batch; null when final validation did not succeed |
| submittedAction | The canonical batch the harness returns as the turn's decision of record; its identity is PRIMARY |
| executedAction | NOT_EXECUTED in a read-only workflow; analysis separately applies submitted batches and records every result, partial application and subsequent digests |

The full original replies and all attempts remain in the record. Validation
alone is not submission or execution. Operations apply before staffing; these
are separate batches, without cross-batch rollback. Rejected final batches
submit nothing; execution records never silently substitute another action.

## Primary identity and denominator

`dreamteam.submitted-action/v1` decodes operations and staffing through the
engine action type, omits empty batches, and hashes deterministic JSON.
Whitespace and object-key order do not matter. An omitted extend_shift date
resolves to the decision-point date, exactly as the engine does. Numeric spellings accepted by
the engine, such as integer -0, normalize through the typed action fields.
Contract windows normalize in catalog order because parseTeleWindows and
teleCanon treat them as a set. Hire skills follow canonicalSkills table order;
hire and roster covers sort by room and start, following hireCanon and
sameCovers. Breaks remain in chronological order as the engine requires. Action order and batch order are preserved:
operations apply sequentially, staffing validation reads earlier items, and
acceptance/action records retain that order. Other nested arrays are retained.

The single empty identity is the SHA-256 of `[]`. Abandonment, selecting
no_action without original operations, and an empty whole-reply revision all
share it. Selecting no_action while preserving operations is not an empty
submission. Invalid and failed final turns also submit nothing; their
secondary statuses disclose why. Canonical identities do not depend on any
response-route label, menu ID, reason text, arm, or success status.

Per context, model and arm, the exposed denominator contains completed draws
whose comparison/confirmation turn was actually reached (`exposure=true`).
Both arms use the same eligibility test. Non-exposed no-action, correction
failure and nonqualifying contexts never enter that denominator. Zero exposure
produces no arm estimate. Exposure can depend on pre-turn correction; report
its frequency and the conditional interpretation explicitly. Partial coverage
produces descriptive observed records only, without complete-run comparisons.

Report per-identity treatment-minus-control proportions, Newcombe intervals
and empirical total variation. The audit witness uses five control
abandonments and five treatment no_action selections: primary TV is exactly
0, while secondary route TV is 1. Small empirical TV does not establish
stability, and positive empirical TV is possible without a population effect.

## Secondary process fields

| Field | Meaning |
|---|---|
| eligibility | true/false with engine reason; null when eligibility cannot be established because the pre-turn proposal failed |
| exposure | The comparison/confirmation turn was actually reached, in either arm |
| validationFirstCorrection | The first proposal failed validation and a correction request was made |
| correctionFailure | The correction did not yield a validated proposal |
| deliberateNoAction | A valid, deliberately empty decision; never an invalid correction |
| processFailure | A transport, response-shape, parse or limit failure occurred, including recovered failures |
| responseRoute | Original final reply route, preserved independently of the primary identity |
| retries / modelCalls | Actual retry requests and actual non-replayed model calls; inspection opportunities are not retries |

The seven F7 labels remain secondary: original_retained, listed_alternative,
no_staffing_action, other_batch, abandoned, invalid, no_decision. The
arm-specific result labels are secondary too. Neither is an action outcome
space. Pre-turn not_due is eligibility/process information, never an eighth
primary category. Retry exhaustion has secondary no_decision and its actual
last response route; it is not deliberate inaction.

Citation credit is secondary. Only actually delivered detail counts. Up to
three cited items must show a verified difference for at least two
alternatives including one not chosen; copying all common rows does not
count. Integer counts 2481 and 2,481 are equivalent. Typed citations retain
units; cents, dollars, minutes, slots and bookings are distinct. Non-numeric
fields match exactly. Correct reproduction does not establish internal
reasoning, comparison quality, learning or decision improvement.

S1-a and S6-a have identical public prompts and proposals. Both remain in
the six labelled sensitivity pairs. They add repetition, not independent
scenario diversity. Pair departures use canonical submitted identity versus
the validated proposal identity, within exposed members, and remain
secondary context comparisons. They do not identify single-fact effects.
