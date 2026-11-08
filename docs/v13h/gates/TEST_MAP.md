# V0.13h adversarial test map

Independent re-review PENDING. Human review PENDING.

The numbered cases correspond to the owner's requested cases. Each uses scripted failures or loopback-only HTTP, with no inference.

| Case | Test |
|---|---|
| 01 | `Lifecycle.test_01_server_unavailable_before_run` |
| 02 | `Lifecycle.test_02_server_unavailable_mid_run` |
| 03 | `Lifecycle.test_03_health_timeout` |
| 04 | `Lifecycle.test_04_malformed_health_response` |
| 05 | `Lifecycle.test_05_unhealthy_response` |
| 06 | `Lifecycle.test_06_first_health_interruption` |
| 07 | `Lifecycle.test_07_later_health_interruption` |
| 08 | `Lifecycle.test_08_engine_open_failure` |
| 09 | `Lifecycle.test_09_draw_level_failure` |
| 10 | `Lifecycle.test_10_cleanup_failure` |
| 11 | `Lifecycle.test_11_primary_plus_cleanup_failure` |
| 12 | `Lifecycle.test_12_missing_manifest` |
| 13 | `Lifecycle.test_13_truncated_manifest` |
| 14 | `Lifecycle.test_14_failed_atomic_replacement` |
| 15 | `Lifecycle.test_15_unknown_error_type` |
| 16 | `Lifecycle.test_16_invented_stop_code` |
| 17 | `Lifecycle.test_17_wrong_event_stage` |
| 18 | `Lifecycle.test_18_wrong_scheduled_draw` |
| 19 | `Lifecycle.test_19_negative_draw_count` |
| 20 | `Lifecycle.test_20_impossible_call_count` |
| 21 | `Lifecycle.test_21_missing_aborted_artifact` |
| 22 | `Lifecycle.test_22_contradictory_cleanup_event` |
| 23 | `Lifecycle.test_23_genuine_partial_analysis` |
| 24 | `Lifecycle.test_24_partial_comparison_suppression` |
| 25 | `Lifecycle.test_25_exact_model_free_completed_replay` |

Additional tests: `test_error_shapes_are_closed`, `test_connection_refused_production_getter`, `test_no_resume_and_unfinalized_rejected`, `test_event_artifact_required`, `test_preserved_protocol_and_schedule`, `test_primary_survives_session_cleanup`, `test_failed_checkpoint_stays_partial_and_propagates`, `test_setup_exception_is_analyzable`, `test_contradictory_side_artifacts_rejected`, `test_no_boundary_stop_after_completed_schedule`.

Source: analysis/alternatives/test_v13h.py. Results and complete original witness outputs are retained under /home/george/dt-audit-kept/v013h/.
Before-fix original health verification fails; cleanup and initial-health interruption lose manifests; later-health interruption lacks a stop code. After-fix witness acceptance and regression results are indexed in WITNESS_RESULTS.json. New adversarial cases do not imply that every case failed before remediation.
