# V0.13k lifecycle fault verification

Independent review PENDING. Owner protocol lock and live inference decisions remain NO GO.

The shared contract generated 5,288 fault cases at 1,619 points over nine bases. All agree, with zero false COMPLETE, no unreached declared points, all 19 transitions and all 22 admissible stop states observed. The checker includes pending completed draws and returned retained-error evidence, not only committed ledger rows. Durable handoffs must survive archive checks; the public error fallback is allowed only before a durable handoff when persistent storage rejects further writes.

Real-process signal/termination checks: 353/353; real localhost socket checks: 12/12. All ran on horo. Existing seal format and process-crash commit semantics remain unchanged. Arbitrary power-loss recovery is outside this milestone.

| Completed-draw fault | Classification | Completed retained | Replay |
| --- | --- | ---: | --- |
| fsync_failure | INVALID | 1 | PASS |
| close_failure_after_fsync | INVALID | 1 | PASS |
| rename_failure | INVALID | 1 | PASS |
| interruption_before_return | PARTIAL | 1 | PASS |
| ledger_commit_failure | INVALID | 1 | PASS |
| error_handling_failure | INVALID | 1 | PASS |

Machine results: [fault matrix](fault-matrix/matrix.json), [signals](signals/results.json), [loopback](loopback/results.json), [preservation](preservation/results.json). The exact error and failed candidate evidence are included in the preservation results; full scratch archives are retained outside the repository.
