# CANDIDATE - NOT LOCKED - awaiting human review

LOCK CANDIDATE - NOT LOCKED - awaiting human review

HUMAN REVIEW PENDING. Protocol-lock commit: NONE. Live inference NO GO.
V0.13.0 not tagged or released.

Engine provenance: fb8cf3ae6886926ba330770bf790ae8f0edf3646. The candidate
is the committed tree containing this document, resolved as HEAD by the
verifier. Engine and session trees are identical to that baseline; the
engine string remains 0.13.0-dev.2. A commit cannot contain its own hash:
FINAL_HASH_MANIFEST.txt pins file bytes, and the verifier binds them to the
actual committed tree. Any changed artifact requires a new manifest.
This is candidate tooling only and writes no protocol status.

Run python3 -I analysis/alternatives/lock_check.py from a committed checkout.
It checks every manifest file against both committed bytes and the working
file, plus preserved Git trees against the baseline. Mismatches fail closed.
The manifest excludes itself to avoid self-reference. The review package
checksum list covers its own contents, and is pinned by the final manifest.

| Pin | Artifact |
|---|---|
| Engine and branch builder | internal/sim, internal/session, internal/session/branch.go |
| Workflow source and specification | agents/llm/llm_agent_alternatives.py; PRESENTATION_EVIDENCE.md |
| Driver and analysis | analysis/alternatives/v13d.py; v13d_fidelity.py |
| Selection v2 | internal/alternatives/text/selection-v2.txt |
| Presentation v2 and view table | internal/alternatives/text/presentation-v2.tmpl; docs/benchmark/v13d-dev/presentation/view-table.txt |
| Control template | internal/alternatives/text/control-v1.tmpl |
| Comparison contract | internal/alternatives/comparison.go; docs/v13b/COMPARISON_CONTRACT.md |
| Branch-point manifest | docs/benchmark/v13c-dev/points/manifest.json |
| Sensitivity contexts | docs/benchmark/v13d-dev/contexts/contexts.jsonl; sensitivity-pairs.json |
| Protocol inputs and schedule | docs/benchmark/v13d-dev/protocol-draft.json; schedule-draft.json |
| Outcomes | OUTCOME_DEFINITIONS.md; internal/alternatives/turn.go |
| Review material | review-package/SHA256SUMS and every listed text |

All named files and every preserved evidence file have SHA-256 entries in
FINAL_HASH_MANIFEST.txt. Git tree IDs are also in BASELINE_HASHES.txt.
Frozen draft status is unchanged. Its stale descriptive estimand field is
recorded in REVIEW_RESPONSE.md; it must not silently override the approved
final-action scale. Owner disposition remains a prerequisite for a future
protocol. No new context or arm is introduced.

Models (exact frozen identifiers):
* gemma4-26b-a4b: `gemma-4-26B-A4B-it-oQ4e-mtp`
* qwen36-35b-a3b: `Qwen3.6-35B-A3B-oQ4e-mtp`

Settings files: docs/benchmark/v05c-dev/llm-gemma4-26b-a4b.json and
llm-qwen36-35b-a3b.json. Temperature 0, max_tokens 4096, json_object,
300-second timeout, maxRetries 1, details allowed, existing system prompt.
Their exact bytes, loader, workflow and draft protocol are pinned.

Nine historical contexts, six sensitivity pairs (twelve members), two arms,
262 scheduled draws, 802 maximum calls, four-hour (14,400 second) hard limit.
Schedule is re-derived by existing offline tests. The primary outcome is
actual final action; verified delivered-detail citations are secondary.
Sensitivity pairs are context comparisons, never isolated fact effects.

Failure rules: sequential calls; no silent retries; six-call per-draw guard;
stop before a sixth consecutive transport failure; no request unless its
full timeout fits; health/material-change stop; exception or interrupted draw
leaves PARTIAL. Record every call and engine exchange. No automatic resume,
replacement draws or complete-run comparisons from partial coverage.
Model-free replay reconstructs requests, answers, metadata and stop evidence.
Selection, validation and successful/partial execution remain separate.

RUN_PROCEDURE.md gives the currently accepted command syntax and explains
that the live implementation is absent. PARTIAL_RUN_PROCEDURE.md gives the
offline analysis and preservation rules. No live path was invoked.
