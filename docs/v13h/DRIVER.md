# Run recorder successor — dreamteam-v13h-driver/4

Independent technical re-review PENDING. Human review PENDING.
Protocol lock, live inference and V0.13.0 release: NO GO.

The successor in analysis/alternatives/v13h.py changes only failure recording,
run lifecycle, persistence and verification. The decision workflow, parser,
selection, canonical identity, presentation, outcomes and engine operations
retain their V0.13g implementations. The protocol's schema, id and driver
binding are successors; all other protocol fields and all schedule entries
are identical to V0.13g: 21 contexts, nine historical contexts, six pairs,
262 draws and at most 802 model calls. No automatic resume is implemented.
The run command continues to refuse inference even with nominal authorization.

A run directory is exclusive. The initial PARTIAL / IN_PROGRESS manifest is
written before limits, health reads or engine operations. Checkpoints record
current stage, identity, protocol and schedule hashes, completed prefix,
interrupted or aborted draw, observed attempt count and cleanup status.
COMPLETE is written only after all scheduled draws and successful cleanup.
An unfinished checkpoint is never accepted as a finalized run.

Health checks retain the scheduled boundary and attempted GET /health, actual
returned state or raised exception, failure timestamp and original stop code.
Failed reads never become successful requests or model calls. Boundary failure
and cleanup records are record-only. Causes that cannot be reconstructed are
UNVERIFIED. A primary failure survives both session and pooled cleanup errors.

The four-hour mechanism remains a request-start guard, not a deadline for
blocked engine operations or cleanup. No clinical or live evaluation is added.
