# Response to the Slice 5C.4c access-gates audit (maintainer)

This is the maintainer's account of what was done with each finding of
`slice-5c4c-access-gates-audit.md`. That file is kept verbatim and
unchanged. This document is separate, so that the auditor's claims stay
distinct from the remediation (Slice 5C.4d, 0.2.0-dev.18, under the
owner's decisions of 2026-10-18).

| Finding | Status | Where |
|---|---|---|
| F1 (Critical): Gate B's bound lacks its coverage | **Construction removed.** Gate B is now seed-level, with a finite-sample `betting` bound. At 40 seeds that bound cannot pass: its floor is 0.207, and no valid method can go below 0.072. **Blocked on an owner decision** (method, seeds or margin). | `docs/calibration/clinical-5c4d/gate-b-statistics.md`; `internal/eval/bounds.go`, `lostdemand.go`; `TestGateBCoverage`, `TestBettingBound` |
| F2 (High): array-position end capture | **Fixed.** | `endCapture`; `TestAuditF2CaptureOrder` |
| F3 (High): empty or unknown outcomes; unknown priority | **Fixed** (v5 integrity: INVALID, no panic). | `checkClinicalRecord`; `TestAuditF3UnknownEvidence` |
| F4 (High): duplicate keys; seed dependence | **Fixed** for identity (immutable `origin`; no pairing from timestamps) and for the sampling unit (the seed). Inference is blocked with F1. | `Order.Origin`; `TestDemandExogenousAcrossPlans`, `TestGateBMissingIdentity` |
| F5 (High): the protocol misses near-margin false acceptance | **Protocol updated** (near-margin pairs, coverage, clusters, identity, every witness). Not runnable until F1's decision. | `docs/calibration/clinical-5c4d/reserved-validation-protocol.md` |
| F6 (High clinical limitation) | **Fixed as the owner decided.** Every favourable verdict carries "Access non-deterioration only; absolute access adequacy not established.", and access statements are always shown. | `Comparison.AccessLimitation`, `Comparison.Access`; `TestAccessVerdictLimitation` |
| F7 (Medium): weekly severity's meaning and partition effects | **Owner decision recorded:** a strict local 5% ceiling, forward pooling kept provisionally. **Partition sensitivity measured** (synthetic witnesses; one of ten no-action simulator seeds for urgent) and recorded as a **freeze blocker**. | `remediation.md` §5; `TestWeeklyPartitionSensitivity`, `TestWeeklyBoundaries` |
| F8 (Medium): allowance above one | **Fixed** (every clinical fraction finite, in [0, 1], whole basis points). | `clinicalFraction`; `TestCandidateValidation` |
| F9 (Low): sparse-week spans; reporting | **Fixed** (calendar weeks, occupied weeks, dates). | `WeeklySevere`; `TestAuditF9WeeklySpans` |
