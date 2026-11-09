# V0.13k partial-run procedure

Identity: `dreamteam.alternatives-partial-run/v7`. Independent review PENDING. Protocol lock and live inference NO GO.

1. Preserve the entire original archive, including call journal, failed candidates, handoff, failure report and absent seal. Do not repair or remove inconsistent evidence to obtain acceptance.
2. Use `analysis/alternatives/v13k_verify.py` FULL analysis with the pinned protocol, exact schedule, original contexts and engine binary. Structural classification alone is insufficient.
3. Derive COMPLETE/PARTIAL/INVALID from the whole evidence set. A present seal does not override semantic contradictions. An absent seal is always INVALID. Unknown call arrays never disable journal limits or reconciliation.
4. Replay completed/interrupted ledger rows and every reconstructable aborted-stage engine/workflow prefix. Compare requests, exchanges, decisions, submitted representation, batches, process and digests with typed exactness. Only the external cause remains UNVERIFIED.
5. For PARTIAL, report verified completed draws and the explicit stopped/aborted draw, retain unknowns as unknown, and suppress complete-run arm differences and sensitivity comparisons. No completed evidence may disappear because a later transfer failed.
6. For INVALID, retain audit counts and pending completed results; refuse analysis and comparisons. A pending row in a failed candidate may be replayable independently, but does not authorize treating the entire archive as a valid run. Recovery/resumption is unavailable.

Full-analysis entry point is `analyze(binary, run_dirs, contexts, protocol, schedule, archive, protocol_sha, allow_synthetic=False)`. Synthetic fixtures require explicit admission. The ordinary CLI cannot authorize inference; see driver help for read-only commands.
