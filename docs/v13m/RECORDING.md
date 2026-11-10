# V0.13m-lite recording specification

Driver `dreamteam-v13m-driver/9`; manifest `dreamteam.alternatives-run/v9`;
analysis `dreamteam.alternatives-analysis/v10`; partial-run procedure /9;
acceptance standard `dreamteam.acceptance-standard/v1`; failure retention
`dreamteam.failed-persistence-retention/v3`. Lifecycle /4, evidence bindings
/2, strict JSON /1, response handoff /1, completed-draw handoff /1, engine
`0.13.0-dev.2`, draw /4, call journal /1, health observation /2, failure
event /3, public result /1 and completion seal /1 retain their formats.

The recorder is the V0.13l recorder ([V0.13l recording](../v13l/RECORDING.md))
with two changes, both under the [fail-closed policy](ACCEPTANCE_CONTRACT.md):

1. **Completed-draw recovery (DT13L-03).** If recovering a draw's result
   after a first exception itself raises, the archive is poisoned first (so
   nothing later can seal it), the possible loss is recorded
   (`evidenceLoss`: kind `completed-draw recovery failed`, the draw, the
   dispatch count, the original and the secondary error), and the run fails
   closed at once (`RecordingError`, unsealed, INVALID). Previously the
   secondary exception replaced the original and the run could be sealed as
   an accepted `draw_unrecorded` PARTIAL.
2. **Failure-report construction (DT13L-04).** If building the retained
   failure document itself raises, the minimal fallback keeps the objects the
   recorder already owns by reference — including the response held in
   `received` at receipt — and records `evidenceLoss` (kind
   `failure-evidence construction failed`, the secondary error). If encoding
   fails, received responses are kept as text and that is disclosed.

`persistence-failure.json` (`failed-persistence-retention/v3`) =
`{schema, stage, error, pending, partialDraw, receivedResponses, dispatched,
candidates, evidenceLoss}`; the identical document is attached to
`RecordingError.retainedEvidence`. It is unsealed audit evidence only, never
part of a COMPLETE or PARTIAL archive.

No retry, resumption or replacement draw exists. Model-visible bytes,
request construction, the decision workflow and the response parser are
unchanged; retained response bytes are exact. The verifier
(`v13m_verify.py`) imports no recorder logic and shares only contract data
(`bindings_v2.json`, `acceptance_v1.json`).
