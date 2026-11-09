# V0.13j test map

Independent technical re-review PENDING. Human presentation review PENDING.
Every check uses scripted model replies, scripted HTTP byte streams, genuine
exception objects, real signals to real processes, or localhost-only HTTP
servers. No inference.

## Owner phases to checks

| Phase | Check | Location |
|---|---|---|
| 1 Baseline | starting commit 72dc35a, per-file and per-category hashes; preserved bytes checked | `docs/v13j/BASELINE.json`, `BASELINE-HASHES.sha256`; `v13j_identities.py --check`; `test_v13j.Contract.test_historical_code_is_untouched` |
| 2 Safety contract | committed first (8b37738); self-check; refused inconsistent variants | `lifecycle_v2.json`, `lifecycle_v2_spec.py`, `test_lifecycle_v2_spec.py`, `docs/v13j/SAFETY_CONTRACT.md` |
| 3 Safe-point SIGINT | handler only counts; ten declared safe points; real SIGINT in-process at every reached point (fault matrix kind `signal`), from a parent process at every safe point, window and finalization phase, to the process group (engines isolated), and on the four DT13I-01 source lines | `v13j_faults.py`; `v13j_signals.py` (`sigint:*`, `sigint-process-group:*`, `sigint-line:*`); witnesses DT13I-01/*; `test_v13j.SafePoints` |
| 3 Known calls | durable dispatch intent and outcome; unresolved dispatches kept; `draw_unrecorded` calls null; every killed archive keeps its durable rows and journal | `calls.jsonl`; witnesses DT13I-01/*; matrix and signal agreement checks "durable row/call lost" |
| 4 Sealed completion | seal schema and commit point; verifier requires a valid seal; never sufficient | `v13j.Recorder._seal`, `v13j_verify.Classifier._seal`; mutations `seal_*`; `test_unsealed_archive_is_never_terminal_whatever_the_manifest_says` |
| 5 Fault injection | exception, real signal, single I/O fault, persistent failure (also defeating recovery), transport, guard halt at every reached point of nine bases; real SIGKILL at every point of base A and the stop sequences of B and C; SIGTERM; external SIGKILL | `v13j_faults.py`, `docs/v13j/fault-matrix/matrix.json`; `v13j_signals.py`, `docs/v13j/signals/results.json` |
| 6 Health evidence (DT13I-03) | closed exception families; genuine wrapped/unwrapped failures; masquerades refused by recorder and verifier; healthy/unhealthy/malformed/disconnected/incomplete/timeout/wrapped cases | `test_v13j.HealthFamilies`; witnesses DT13I-03/*; `v13j_loopback.py`; mutations `os_error_named_*`, `wrapped_*` |
| 7 Semantic verification (DT13I-04) | per-stage fields, transition predicates, health visits once, JSON-null side files, cleanup, stop fields | `v13j_mutations.py` (re-sealed); `v13j_verify.Classifier` |
| 8 Interrupt reporting (DT13I-05) | public result `interrupts`; cleanup always runs; AFTER_STOP / AFTER_SCHEDULE | witnesses DT13I-05/*; signal tests at RECORD points and finalization phases |
| 9 Timestamps (DT13I-06) | strict grammar; order incl. dispatch <= sent <= outcome; replay identity excludes timestamps | `test_lifecycle_v2_spec.test_timestamp_grammar_dt13i06`; mutations `all_times_*`, `timestamp_*`, `valid_call_metadata_shift` |
| 10 Systematic testing | matrix generated from the contract and execution traces; measured transition, stop-state and disposition coverage; recorder and analyzer both checked; loopback on horo | `docs/v13j/fault-matrix/matrix.json` `coverage`; `docs/v13j/loopback/results.json` |
| 12 Quality gates | see gates/README.md | horo logs |
| 13 Preservation | model-visible material, schedule, protocol fields, engine | `v13j_identities.py --check`; `test_v13j.Contract.test_protocol_and_schedule_preserved` |

## The independent review's witnesses (DT13I-01..06)

| Original witness (V0.13i review scratch) | V0.13j regression |
|---|---|
| adversarial.py `sigint_after_transport`, `sigint_before_calls_append`, `sigint_draw_finally`, `sigint_draw_return` | witnesses `DT13I-01/sigint_*` (in-process) and signal tests `sigint-line:*` (real child processes) |
| adversarial.py `kill_after_unlink` | no marker exists; signal tests `sigkill-self:A:*` at every reached point incl. every seal phase, `sigkill-external:seal.*` |
| adversarial.py `marker_restore_io_failure` | witnesses `DT13I-02/persistent_*` and `os_patched_persistent_failure_after_final_manifest`; matrix kind `persistent` |
| adversarial.py `deferred_public_api` | witnesses `DT13I-05/sigint_during_*_public_api`; signal tests `sigint:seal.*` |
| adversarial.py `url_incomplete`, `url_bad_status`, `http_not_connected`, `http_remote` | witnesses `DT13I-03/genuine_*` |
| forgeries.py `os_error_*` | witnesses `DT13I-03/os_error_*` (recorder refuses); mutations `os_error_named_*` (verifier refuses) |
| forgeries.py `engine_open_with_finished_record`, `engine_open_with_transport_halt`, `premature_schedule_done_cleanup_failure`, `null_*`, `timestamp_*` | mutations of the same names |
| extra_forgeries.py `duplicate_round_health`, `all_times_trailing_newline`, `all_times_unicode_digits`, `valid_call_metadata_shift` | mutations `duplicate_round_health`, `all_times_trailing_newline`, `all_times_arabic_indic_digits`, `valid_call_metadata_shift` (positive control) |
| health_and_interrupt_edges.py `healthy_http_503`, `healthy_hash_wrong`, `healthy_null_time`, `sigint_before_cleanup` | witnesses `DT13I-03/healthy_*`, `DT13I-05/sigint_before_cleanup_releases_pool` |

The reviewer's retained scratch is extracted (not into the repository) at
/home/george/dt-audit-kept/v013j/witnesses/scratch/; its paths refer to the
review host.
