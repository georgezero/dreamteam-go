# DT13L remediation and policy dispositions

Baseline: [the immutable V0.13l independent review](../audits/v0.13l-independent-review.md),
starting commit `133e21d2181c10f910da00c0339fce77257d51f9` (V0.13l implementation
`7b21d93` plus the verbatim review). Owner decision (V0.13m-lite): fix DT13L-01
and DT13L-02; for DT13L-03 and DT13L-04 adopt the conservative
[fail-closed persistence policy](ACCEPTANCE_CONTRACT.md) instead of further
double-fault recovery machinery — an explicit, owner-approved change to the
acceptance standard. Protocol lock and live inference: NO GO.

| Finding | Severity | Disposition | Implementation |
| --- | --- | --- | --- |
| DT13L-01 ambiguous response JSON accepted as COMPLETE | BLOCKER | **FIXED** | R-JSON-004 in `v13m_verify.py`: every retained response body (draw calls and call-journal outcomes) reparsed under strict JSON; a body the workflow parser reads must be read identically by the strict parser; metadata derived from the strict reading. Exact bytes and the frozen workflow/parser unchanged. |
| DT13L-02 impossible wall-clock message accepted as PARTIAL | BLOCKER | **FIXED** | R-HALT-003: displayed seconds bounded below by the guard threshold (`threshold − 0.5 < m ≤ elapsed + 0.5`), elapsed must trigger the guard, and with SYSTEM_UTC timestamps every halt's elapsed must agree with the structured journal/call-journal timestamps (60 s tolerance). The four-hour request-start guard is unchanged; no hard deadline claimed. |
| DT13L-03 nested recovery failure loses a completed draw | MAJOR | **POLICY DISPOSITION (fail closed) — not resolved** | Recorder: a failure while recovering a draw's result poisons the archive and fails closed at once (unsealed, INVALID), disclosing `evidenceLoss` (possible: true) with both errors; no retry/resume/replacement. Verifier: R-HANDOFF-003 refuses a `draw_unrecorded` PARTIAL whose retained calls complete the workflow. The completed result held only in the failed frame can still be lost; the run is rejected and the loss reported. |
| DT13L-04 retention fallback omits the received response | MAJOR | **POLICY DISPOSITION (fail closed)** | The run was already INVALID. The minimal fallback now keeps the owned objects by reference, including the response held at receipt, and discloses `evidenceLoss`; no additional recovery machinery. In the reviewer's witness the response is now retained in `persistence-failure.json` and in `RecordingError.retainedEvidence`. |

## Witnesses before and after

`analysis/alternatives/v13m_dt13l.py` ports the reviewer's constructions
(response-envelope edits with every dependent hash, journal copy, delivery
copy, row commitment, mirror, counter and seal recomputed; genuine recordings
of altered wire bytes; the reviewer's ordinary-CLI recordings; nested
recovery faults) and runs them on genuine archives of each implementation:
[before: V0.13l](dt13l/before-v13l.json), [before: pre-fix successor](dt13l/before-prefix-successor.json)
(the unchanged V0.13l recorder and verifier mechanically renamed to the
successor identities), [after: V0.13m-lite](dt13l/after-v13m.json).

| Group | V0.13l | Pre-fix successor | V0.13m-lite |
| --- | --- | --- | --- |
| DT13L-01 (33: 20 ambiguous rejections, 13 controls incl. 2 unparsed) | 12/33 | 12/33 | 33/33 |
| DT13L-02 (14: 10 impossible, 4 valid) | 6/14 | 6/14 | 14/14 |
| DT13L-03 (fail closed; plus one verifier-defense case after) | 0/2 | 0/2 | 3/3 |
| DT13L-04 | 0/1 | 0/1 | 1/1 |
| Positive controls, primary counterexample, persistent-storage policy | 6/6 | 6/6 | 6/6 |

Key cases (classification, full analysis):

| Case | V0.13l | V0.13m-lite |
| --- | --- | --- |
| Duplicate `choices` (resealed, wire, ordinary CLI) | COMPLETE, accepted | INVALID, rejected (R-JSON-004) |
| Escaped-equivalent duplicate key, nested and deep-nested duplicates | COMPLETE, accepted | INVALID, rejected |
| NaN / Infinity / -Infinity / 1e999 / -1e400 | COMPLETE, accepted | INVALID, rejected |
| Unicode literal/escape, 2^80+1, `-0.0 1E+2 1e308 5e-324 -0`, valid usage | COMPLETE, accepted | COMPLETE, accepted |
| Trailing content / truncated body (neither parser reads it) | COMPLETE, accepted (unparsed reply) | COMPLETE, accepted (unparsed reply, no metadata) |
| "0 s elapsed" at the draw guard and at request start | PARTIAL, accepted | INVALID, rejected (R-HALT-003) |
| Message one second below the guard threshold | PARTIAL, accepted | INVALID, rejected |
| Ordinary CLI: "0 s elapsed"; elapsed 10^9 with a seconds-long timestamp span | PARTIAL, accepted | INVALID, rejected |
| Ordinary CLI, linked clocks: valid halt just past / well past the threshold | PARTIAL, accepted | PARTIAL, accepted |
| Ordinary CLI, linked clocks: elapsed beyond the timestamps; message below threshold | PARTIAL, accepted | INVALID, rejected |
| Nested recovery failure (bookkeeping; entry) | accepted `draw_unrecorded` PARTIAL, result lost silently | `RecordingError`, unsealed, INVALID, rejected; `evidenceLoss.possible = true`; transport calls = dispatches; no later draw |
| Recorder without the guard (verifier defense) | — | PARTIAL classification, full analysis refused (R-HANDOFF-003) |
| Retention construction failure | INVALID, response lost | INVALID, response retained (archive and structured error), loss disclosed |

The reviewer's original V0.13l ordinary-CLI archives (`duplicate-response`,
`wall-message`) are accepted (exit 0) by the V0.13l verifier and rejected
(exit 1) by the V0.13m-lite verifier, which reports the semantic violation
(R-JSON-004, R-HALT-003) as well as the identity mismatch.

DT13L-03/04 are recorded as a policy disposition under the owner's explicit
acceptance-standard change, not as resolved defects: evidence can still be
lost in an unrecoverable double failure; the run is then rejected and the
possible loss is reported.
