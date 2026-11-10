# V0.13l partial-run procedure

Identity: `dreamteam.alternatives-partial-run/v8`. Independent review PENDING. Protocol lock and live inference NO GO.

1. Preserve the entire original archive, including call journal, failed candidates, handoff, failure report (with received responses and partial draw) and absent seal. Do not repair or remove inconsistent evidence to obtain acceptance.
2. Use `analysis/alternatives/v13l_verify.py` FULL analysis with the pinned protocol, exact schedule, original contexts and engine binary. Structural classification alone is insufficient.
3. Derive COMPLETE/PARTIAL/INVALID from the whole evidence set under strict JSON. A present seal does not override a semantic contradiction. An absent seal is always INVALID.
4. Replay completed/interrupted rows and every reconstructable aborted-stage prefix. Compare requests (bytes and structure), exchanges, decisions, submitted representation, batches, process and digests strictly. Bind every failure representation (engine entry, raised call, aborted draw, stop, message, event) and reconstruct every halt from calls, limits and phase. Only the external cause and the wall-clock measurement remain UNVERIFIED; the analysis reports the reconstruction boundary and the external evidence.
5. For PARTIAL, report verified completed draws and the explicit stopped/aborted draw, retain unknowns as unknown, and suppress complete-run arm differences and sensitivity comparisons.
6. For INVALID, retain audit counts, pending completed results and received responses; refuse analysis and comparisons. Retained evidence may be replayed independently, but does not make the archive a valid run. Recovery and resumption are unavailable.

Full-analysis entry point: `analyze(binary, run_dirs, contexts, protocol, schedule, archive, protocol_sha, allow_synthetic=False)`. Any replay failure is a refusal. Synthetic fixtures require explicit admission.
