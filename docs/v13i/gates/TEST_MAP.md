# V0.13i test map

Independent technical re-review PENDING. Human presentation review PENDING.
Every check uses scripted model replies, scripted HTTP byte streams, genuine
exception objects or localhost-only HTTP servers. No inference.

## Owner phases to checks

| Phase | Check | Location |
|---|---|---|
| 2 State machine | `lifecycle_v1.json` self-check; every recorder injection point and transition is declared, every declared point is reached | `test_lifecycle_spec.py`; `test_v13i.Contract`; fault matrix `unreachedDeclaredPoints` |
| 3 Shared invariants | verifier does not import the recorder; 19 invariants in one classifier | `test_v13i.Contract.test_verifier_does_not_import_the_recorder`; `v13i_verify.Classifier` |
| 4 DT13H-01 health | 13 genuine failure/response categories through the production probe; legacy and contradictory shapes refused | `test_v13i.HealthProbe`; witnesses `DT13H-01/*`; `test_v13i.Lifecycle.test_dt13h01_health_failures` |
| 5 DT13H-02 interruption | 20 V0.13h boundaries plus cleanup cases; 14 real SIGINT boundaries and 5 process-termination witnesses | witnesses `DT13H-02/*`; `v13i_sigint.py`; `test_dt13h02_interruption_boundaries`, `test_real_signals` |
| 6 DT13H-03 finalization | final rename before/during/after, file and directory sync, marker removal sync, failed-finalization comparison | witnesses `DT13H-03/*`; fault matrix `persist:marker`, `persist:failure_event`, `persist:final_manifest`, `unlink:marker`; `test_dt13h03_fail_closed_finalization`, `test_persistence_failure_never_completes` |
| 7 DT13H-04 contradictions | re-sealed forgeries: cleanup failure on completed row, unhealthy initial server under COMPLETE, impossible engine-open evidence, impossible call totals, duplicate/missing rows, contradictory stops, illegal transitions, inconsistent hashes, unsupported terminal states, file defects | `v13i_mutations.py`; `test_dt13h04_contradictory_evidence_and_dt13h05_timestamps` |
| 8 DT13H-05 timestamps | null, unparseable, impossible, reversed and out-of-order timestamps; synthetic provenance outside fixtures | mutations `*timestamp*`, `run_*`, `call_sent_*`; `test_no_resume_and_synthetic_provenance_refused_outside_fixtures` |
| 9 Fault matrix | generated from the contract over three base scenarios | `v13i_faults.py`; `docs/v13i/fault-matrix/matrix.json`; `test_fault_matrix_sample` |
| 10 Loopback | 11 server modes on localhost, recorder versus verifier versus analysis | `v13i_loopback.py`; `docs/v13i/loopback/results.json`; `test_loopback_health` |
| 11 Replay and analysis | completed rows replay under the unchanged draw contract; COMPLETE yields comparisons, PARTIAL none | `test_complete_run_has_comparisons_and_partial_suppresses_them`, `test_completed_rows_replay_under_the_predecessor_draw_contract`; lifecycle examples |
| 12 Human review freeze | 93 transcripts and 7 static assets byte-identical | `v13i_human_review.py --check` (also run by `v13i_identities.py --check`) |
| 15 Preservation | protocol and schedule equal to V0.13h except identity fields; 2,884 baseline files unchanged | `test_protocol_and_schedule_preserved`; `v13i_identities.py --check` |

## Earlier review witnesses (regression checks)

| Earlier witness | V0.13i regression |
|---|---|
| V0.13h `health_scripted.py` / `health_matrix.py` A–F, empty connection, truncated body, healthy+error | `DT13H-01/A_…` to `DT13H-01/connection_reset`, `DT13H-01/legacy_*`; loopback cases |
| V0.13h `lifecycle_boundaries.py` 20 line points, cleanup_only, primary_and_cleanup, 5 final persistence cases | `DT13H-02/*`, `DT13H-03/*` (line points mapped to declared injection points) |
| V0.13h `atomic_comparisons.py` | `DT13H-03/failed_finalization_comparison` |
| V0.13h `process_termination.py` | `v13i_sigint.py` kill/terminate/self-interrupt cases |
| V0.13h `verifier_mutations.py`, `retained_integrity_checks.py` | `v13i_mutations.py` |
| V0.13g `lifecycle_edges.py` | `DT13G/lifecycle-edges/*`, `DT13H-01/A_unavailable_before_any_draw` |
| V0.13g `partial_probes.py` (21 cases) and its mutations | `DT13G/partial-probes/*`; `v13i_mutations.py` |

The V0.13g `final_digests` probe fails the second digests request, which the
workflow itself issues; V0.13g recorded that as a workflow failure and so
does V0.13i. Before-fix evidence for all witnesses is the retained reviewer
output under /home/george/dt-audit-kept/v013i/witnesses/.
