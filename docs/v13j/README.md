# V0.13j — safe-point interrupts and sealed run completion

Independent technical re-review (Extra High): PENDING. Human presentation
review: PENDING (separate; the frozen material in docs/v13i/human-review at
ee987a1 is unchanged). Protocol lock: NO GO. Live inference: NO GO. V0.13.0
release: NO GO. Engine 0.13.0-dev.2 unchanged.

| Document | Content |
|---|---|
| [SAFETY_CONTRACT.md](SAFETY_CONTRACT.md) | the safety contract dreamteam.run-lifecycle/v2 (machine-readable: `analysis/alternatives/lifecycle_v2.json`): acceptance rules, severity hierarchy, safe points, call journal, completion seal, durability boundary, per-stage evidence, health exception families, timestamp grammar, invariants |
| [RECORDING.md](RECORDING.md) | run/v6 archive, public run result, call journal, health-observation/v2, run-failure/v3, the commit point |
| [PARTIAL_RUN_PROCEDURE.md](PARTIAL_RUN_PROCEDURE.md) | the independent verifier and analysis/v7 |
| [DRIVER.md](DRIVER.md) | driver/6 and the successor protocol v13j-alt-1 |
| [FAULT_MATRIX.md](FAULT_MATRIX.md) | fault matrix, real-process signal and termination tests, loopback, regression witnesses, contradictory archives |
| [gates/README.md](gates/README.md), [gates/TEST_MAP.md](gates/TEST_MAP.md) | engineering gates and the owner-phase test map |
| [CONTRACT-IDENTITIES.json](CONTRACT-IDENTITIES.json), [CONTRACT-PINS.json](CONTRACT-PINS.json) | successor identities |
| [BASELINE.json](BASELINE.json), [BASELINE-HASHES.sha256](BASELINE-HASHES.sha256) | starting-commit hashes by category and per file |
| [../audits/v0.13j-audit-response.md](../audits/v0.13j-audit-response.md) | DT13I-01..06 disposition |

Evidence: `docs/v13j/fault-matrix/matrix.json`, `docs/v13j/signals/results.json`,
`docs/v13j/loopback/results.json`, `docs/v13j/witnesses/results.json`,
`docs/v13j/mutations/results.json`, `docs/benchmark/v13j-dev/` (protocol,
schedule, record-only lifecycle examples).
