# DT13J remediation map

Remediation baseline: [the immutable independent review](../audits/v0.13j-independent-review.md), starting commit `cb5ac6832e7b43ef68212355ff395cf5958fff11`. All five owner dispositions are FIX, with exception identity fixed narrowly. Independent review of this successor remains PENDING.

| Finding | Bindings | Implementation and tests |
| --- | --- | --- |
| DT13J-01 | B-ABORT-001/002, B-DECISION-001/002, B-BATCH-001, B-ENGINE-001/002, B-TRANSPORT-001, B-CALL-001/002/003/004 | Full retained aborted-stage reconstruction; exact stage shapes; original final digest, abandoned proposal, erased call, missing record/halt and engine response counterexamples in `v13k_regressions.py`; all-stage inherited witnesses; binding matrix |
| DT13J-02 | B-CLEANUP-001, B-INTERRUPT-001, B-JOURNAL-001, B-SEAL-002 | Mandatory ordered cleanup visits and exact timestamp intervals; duplicate/reversed/early original counterexamples; ten-kind matrix; real signal cases |
| DT13J-03 | B-CALL-001/002/003/004, B-MANIFEST-001, B-LEDGER-001 | Journal bounds from scheduled context independently of null call arrays; lowercase hexadecimal request/response hashes; original null-calls over-context and nonhex and body-without-status/error counterexamples; lifecycle guard bases and matrix |
| DT13J-04 | B-HANDOFF-001, B-LEDGER-002, B-JOURNAL-002, B-CALL-001, B-FAILURE-001 | Durable pending result, retained failed candidates, error report and traceback transfer at the return boundary; six real filesystem/return/error-handling cases in `v13k_preservation.py`; pending-aware lifecycle fault checks |
| DT13J-05 | B-HEALTH-004, B-HEALTH-002 | Trusted imported class objects only; all supported genuine families and wrapped reasons retained; every listed class impersonated in `test_v13k.py`; wrapped/unwrapped original impersonation regressions |

Original counterexamples are re-expressed as engineering regressions, with dependent hashes and seals recomputed. The original witness source archive is retained outside the repository under `/home/george/dt-audit-kept/v013k/witnesses/`; its historical absolute paths refer to a removed directory and must be adapted to that retained location. No historical source or recording is modified.

The initial inventory commit precedes implementation. Implementation amendments enumerate fixed descendant types and conditional nullability explicitly, correct the persisted latency field spelling, and preserve canonical JSON type identity. The original V0.13j finalization subtree, seal format and commit semantics remain unchanged. A completed result whose first handoff hook fails is handed off again during handled reconciliation before ledger commit; persistent failures remain INVALID.

Results and exact counts: [binding matrix](BINDING_MATRIX_RESULTS.md), [engineering gates](gates/README.md), [preservation](PRESERVATION.md). No independent clearance is implied.

Supplementary original witnesses cover duplicate health-status keys and nonfinite health JSON. They use the existing invalid_json category; this closes B-HEALTH-003 byte/category consistency and does not broaden exception families. Trusted exception identity remains the narrow DT13J-05 change.

A supplementary valid-hex request-hash contradiction with a null draw call array is rejected by journal-prefix replay (B-ABORT-002, B-CALL-002/004). This is covered by the retained-evidence requirement even though the array itself is absent.
