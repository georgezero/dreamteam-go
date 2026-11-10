# V0.13m-lite acceptance contract

Identity: `dreamteam.acceptance-standard/v1`. Machine contract:
[`analysis/alternatives/acceptance_v1.json`](../../analysis/alternatives/acceptance_v1.json).
DRAFT, NOT LOCKED. Protocol lock: NO GO. Live inference: NO GO. V0.13.0
release: NO GO. Human presentation review: PENDING.

## Owner-approved change to the acceptance standard

The owner ended the open-ended recorder-hardening cycle (V0.13m-lite,
"Experimental Readiness Closure"). Two scientific-integrity blockers were
fixed (DT13L-01 ambiguous model-response JSON; DT13L-02 impossible
wall-clock halt). For DT13L-03 and DT13L-04 the owner adopted a conservative
**fail-closed persistence policy instead of further double-fault recovery
machinery**. This is an explicit change to the acceptance standard used for
V0.13j–V0.13l, which required that a completed draw or received response is
never erased, under any failure. DT13L-03/04 are **not relabelled as
resolved**: the underlying possibility of evidence loss in an unrecoverable
double failure remains; what changed is that such a run must be rejected and
the possible loss reported, and the owner accepts that as non-blocking.

## Revised standard

**BLOCKING** (any one is a NO GO):

1. False COMPLETE.
2. Contradictory PARTIAL accepted.
3. Invalid evidence included in scientific analysis.
4. Automatic retries or replacement draws that alter the protocol.

**NON-BLOCKING IF DISCLOSED:**

1. A genuine failed run classified INVALID.
2. Loss of evidence during an unrecoverable double failure, provided the run
   is rejected.
3. Lack of automatic crash recovery.
4. Lack of power-loss guarantees.

## Fail-closed persistence policy

If persistence or recovery fails and evidence cannot be reliably retained —
including a failure while recovering a draw's result after a first failure,
or a failure while building the failure report — the recorder:

* stops the run at once and never seals it (the archive is INVALID and
  unusable);
* never accepts it as COMPLETE, and the verifier never accepts contradictory
  or insufficient evidence as a valid PARTIAL;
* excludes it from primary analysis (analysis refuses every INVALID archive);
* makes no automatic retry, no automatic resumption and no replacement of
  scheduled draws;
* reports possible evidence loss: `persistence-failure.json`
  (`dreamteam.failed-persistence-retention/v3`, field `evidenceLoss`) and the
  identical `RecordingError.retainedEvidence`; the verifier lists it under
  `preserved.possibleEvidenceLoss`;
* preserves evidence where possible (the pending row, the partial draw,
  received responses — including the response held at receipt — the dispatch
  count and failed candidates), and never hides an inability to preserve it.

No additional nested recovery system is built to survive multiple
simultaneous persistence failures. The completion seal format and its
process-crash commit point (`dreamteam.run-finalization/v1`, rename of
`SEAL.json`) are unchanged.

## New relationships (verifier)

| Id | Finding | Rule (summary; the machine contract is authoritative) |
|---|---|---|
| R-JSON-004 | DT13L-01 | Every retained model-response body (draw calls and call-journal outcomes) is reparsed under `dreamteam.strict-json/v1`. A body the permissive workflow parser reads must be read identically by the strict parser — no duplicate keys at any depth (including escaped-equivalent keys), no NaN/Infinity, no number overflowing to infinity, no trailing content — and structured metadata must be the strict reading's fields. A body neither parser reads is an unparsed reply to the frozen workflow and carries no metadata. Otherwise the archive is INVALID: no ambiguous response contributes to COMPLETE or PARTIAL analysis. Exact response bytes are preserved; nothing model-visible changes. |
| R-HALT-003 | DT13L-02 | A wall-clock halt is valid only if the guard could have fired: `elapsedSeconds + timeoutSeconds > wallClockSeconds`, and the displayed seconds `m` satisfy `threshold − 0.5 < m ≤ elapsedSeconds + 0.5` (threshold = wallClockSeconds − timeoutSeconds; the message is rounded from a reading between the guard's and the evidence's). With SYSTEM_UTC timestamps, every halt's `elapsedSeconds` must agree with the structured journal and call-journal timestamps within 60 s. The four-hour request-start guard is preserved; no hard deadline is claimed. |
| R-HANDOFF-003 | DT13L-03 | A `draw_unrecorded` draw is acceptable in a PARTIAL only if replaying the frozen workflow over its retained journal outcomes stops at a dispatch or external boundary. If the retained outcomes complete the workflow, a completed result may have been lost, and the archive is rejected with possible loss reported. |

## Unchanged

Lifecycle `dreamteam.run-lifecycle/v4` and evidence bindings
`dreamteam.evidence-bindings/2` (`bindings_v2.json`), completion seal /1,
finalization /1, call journal /1, health observation /2, failure event /3,
draw /4, response handoff /1, completed-draw handoff /1, the engine
(0.13.0-dev.2), simulator and accounting, session/API/scoring, decision
workflow, parser, presentation, alternative selection, primary estimand,
nine historical contexts, six sensitivity pairs, 262 scheduled draws,
802-call ceiling, 93 transcripts and seven static assets.
