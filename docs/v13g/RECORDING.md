# V0.13g recording contract

DRAFT, NOT LOCKED. Independent re-review PENDING. Human review PENDING.
The engine remains 0.13.0-dev.2; historical record schemas keep their original
implementations. The successors are draw/v3, run/v3 and analysis/v4.

## Canonical submission versus exact representation

`submittedAction` is computed by the engine's submitted-action/v2 canonicalizer.
Its SHA-256 determines the PRIMARY action category. It excludes response route,
reason, arm, menu choice and execution status. Empty submissions hash `[]`.

`submittedRepresentation` is a separate string: `json.dumps(batches)` using
Python's default JSON separators and ASCII escaping, preserving dictionary
insertion order and array order. `submittedRepresentationSha256` hashes that
string's UTF-8 bytes. It is captured at completion, before record persistence
can sort enclosing objects. Full original response text remains in deliveries
and calls; the submitted representation describes actual submitted batches,
not the original proposal's formatting.

Analysis checks its hash and decoded equality with `batches`, then independently
replays the workflow to check the exact string as a deterministic field. The
execution key is SHA-256 of `context + "|" + submittedRepresentationSha256`.
Analysis decodes this string for apply, retaining its key order. Each ledger
entry contains that representation, hash, exact batches and all scheduled draw
identities attributed to it. Canonical equivalence alone cannot share an entry.

Execution retains every actual apply request/answer, whether each batch applied,
partial application, digests after apply and day continuation, and the public
observation including action records and financial fields. These are record-only
and never appended to model-visible messages. Operations and staffing remain
separate batches with existing engine semantics and no cross-batch rollback.

## Lifecycle fields

Every caught draw failure binds seq/model/context/arm/draw to the next scheduled
entry. `engineInitialization` is NOT_STARTED, STARTED, OPENED, READY or FAILED.
`failureStage` names draw_setup, engine_open, model_setup, initial_digests,
workflow, final_digests or engine_close. The failure retains type/message,
stopReason, observed calls and engine exchanges, without invented responses.

`modelCallsCompleted` is the number of recorded non-replayed attempt records,
including failed attempts; it does not mean successful inference. The replayed
initial proposal never consumes a model call. `verificationStatus=RECORDED`
marks completed/interrupted recording, not independent approval. Aborted
skeletons use UNVERIFIED. Analysis separately reports reconstructed transport
failures or UNVERIFIED external failures, including initialization failures.
Aborted draws never contribute completed decisions or complete-run estimates.

Between-draw stops have their own stop-event.json and an identical run.stopEvent
object. The event binds the next draw, completed prefix, call count, reason and
supporting guard or external-state evidence. A missing or mismatched artifact
fails the command's verification. External state remains explicitly unverified.
The partial-run contract defines allowed relationships and accounting checks.
