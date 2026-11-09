# V0.13k binding matrix results

Engineering verification on horo; independent review PENDING. Protocol lock and live inference NO GO.

35 relationships × ten mutation kinds = 350 full-analysis cases. All 35 matching cases were accepted; all 315 contradictory cases were rejected. No false COMPLETE or contradictory PARTIAL survived. Dependent row commitments, mirrors and seals were recomputed before full analysis, so semantic cases do not rely on stale checksums. Duplicate-key variants deliberately exercise the strict retained-evidence grammar.

Bindings SHA-256: `fa3cb0a2ddab09f0f1ccfe7f13d80c2d73262b960a3d4c84d03587f54ed44b33`. Machine results: [binding-matrix/results.json](binding-matrix/results.json).

The ten kinds are `valid_matching_evidence`, `missing_required_evidence`, `contradictory_evidence`, `duplicated_evidence`, `wrong_stage`, `wrong_identity`, `wrong_count`, `wrong_hash`, `wrong_ordering`, `invalid_null`.

| Binding | Matching accepted | Contradictions rejected |
| --- | ---: | ---: |
| B-ABORT-001 | 1 | 9 |
| B-ABORT-002 | 1 | 9 |
| B-BATCH-001 | 1 | 9 |
| B-CALL-001 | 1 | 9 |
| B-CALL-002 | 1 | 9 |
| B-CALL-003 | 1 | 9 |
| B-CALL-004 | 1 | 9 |
| B-CLEANUP-001 | 1 | 9 |
| B-DECISION-001 | 1 | 9 |
| B-DECISION-002 | 1 | 9 |
| B-ENGINE-001 | 1 | 9 |
| B-ENGINE-002 | 1 | 9 |
| B-EXECUTION-001 | 1 | 9 |
| B-FAILURE-001 | 1 | 9 |
| B-HANDOFF-001 | 1 | 9 |
| B-HEALTH-001 | 1 | 9 |
| B-HEALTH-002 | 1 | 9 |
| B-HEALTH-003 | 1 | 9 |
| B-HEALTH-004 | 1 | 9 |
| B-HEALTH-005 | 1 | 9 |
| B-INTERRUPT-001 | 1 | 9 |
| B-JOURNAL-001 | 1 | 9 |
| B-JOURNAL-002 | 1 | 9 |
| B-LEDGER-001 | 1 | 9 |
| B-LEDGER-002 | 1 | 9 |
| B-MANIFEST-001 | 1 | 9 |
| B-MANIFEST-002 | 1 | 9 |
| B-MANIFEST-003 | 1 | 9 |
| B-SEAL-001 | 1 | 9 |
| B-SEAL-002 | 1 | 9 |
| B-SEAL-003 | 1 | 9 |
| B-SEAL-004 | 1 | 9 |
| B-STOP-001 | 1 | 9 |
| B-STOP-002 | 1 | 9 |
| B-TRANSPORT-001 | 1 | 9 |

Each case records its exact target and base. The bindings cover 293 named fixed fields plus recursive domain descendants. Invalid-only handoff and execution artifact bindings use permitted absence as their matching case and forbidden sealed artifacts as their negative cases; the separate six-case preservation suite exercises genuine pending-result writes and failures.

Supplementary closure: 16 focused regressions, 76 inherited witnesses, 125 re-sealed mutations including one positive control, and six completed-draw preservation cases. All run full analysis. These are engineering checks and do not replace the required separate independent review.
