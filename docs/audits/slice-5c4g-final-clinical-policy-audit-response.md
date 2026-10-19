# Response to the Slice 5C.4g final clinical policy audit (maintainer)

This is the maintainer's account of what was done with each finding of
`slice-5c4g-final-clinical-policy-audit.md`.

* **The audit is unchanged.** It was committed alone and verbatim
  (c55f3cb) before any remediation, and this document is separate from
  it.
* **The remediation is Slice 5C.4h** (0.2.0-dev.21), under the owner's
  decisions of 2026-10-19.

The owner's decisions:
* **Protocol:** close every protocol rule prospectively.
* **Allowances:** keep the 4-of-40 allowance, the 2-point Gate B margin
  and the 10 / 6 / 5-point safeguard.
* **T2:** separate seed-level detection from whole-evaluation rejection.
* **Boundaries:** close the partial-window and horizon loopholes rather
  than only disclosing them.
* **v4 case selection:** keep its allowance and disclose the cumulative
  consequences.
* **No reserved run and no freeze.**

## Findings G1–G7

| Finding | Status | Where |
|---|---|---|
| **G1** (High): open protocol decisions | **Closed in the proposed protocol, except T5's tolerances,** which await the owner's approval (proposed in §5). Each criterion has its estimand, estimator, interval, family rule, numeric threshold, group and remainder handling, and PASS / FAIL / INCONCLUSIVE. Any FAIL or INCONCLUSIVE is a documented NO GO, and a post-result waiver is a protocol deviation. T6's Monte Carlo error is fixed under the design rate (at most 19 flags in 10,000). T7 counts raw insufficiency per gate (`clinicalInsufficientSeedsByGate`). The protocol is **not locked** | `docs/calibration/clinical-5c4h/reserved-validation-protocol.md`; `reservedCriteria`; `TestReservedCriteriaRules`; `PlanReport.ClinicalInsufficient`, `TestRawInsufficiencyByGate` |
| **G2** (High): detection versus the allowance | **Closed by owner decision.** The allowance is kept. T2 is now harmed-seed detection of at least 0.95 at 15 points or more. Whole-evaluation rejection is reported against P{Bin(40, 0.1) ≥ 5} = 0.371 and never presented as protection. A deterministic T2-conformance check verifies that every group whose allowance is exceeded is rejected | protocol §4 T2 |
| **G3** (High): invalid or underspecified scenarios | **Closed.** A versioned, executable manifest gives 49 scenarios, each with its mechanism, constructions, streams, seeds, injection, effect, metric, acceptance rule and failure consequence. The changes: <ul><li>**Seasonal pairing:** no A85-against-B85 pair exists, and the seasonal practices pair only with themselves. Unequal volumes come from a per-seed demand multiplier shared by both plans (HV).</li><li>**Streams:** the role ranges are disjoint.</li><li>**Injections:** they are random under a stated estimand, reconciled through the study-to-order link, with samples and captures rebuilt. Near-margin injections are Bernoulli with an expected δ; the 2.001-point case is replaced by exact ⌊0.02N⌋ and ⌊0.02N⌋ + 1 witnesses. Offset harm gets a baseline with 5 points of injected loss.</li><li>**Seed shocks:** drawn per seed from the injection stream.</li></ul>Construction is checked on every scenario: `TestReservedManifestConstruction` on PCG(1, ·), and the development run on PCG(20261021, ·) (`construction-checks.json`) | `scenario-manifest.json`; `internal/eval/reserved_manifest_test.go` |
| **G4.1** (Medium–High): partial-window concentration | **Closed by a policy revision, pending re-review.** A partial window below the minimum fails when it holds more severe reports than a compliant minimum window may (5). A priority with nothing judged is INSUFFICIENT. The audit's 640-report witness, GO under 5C.4f, is NO GO | `boundary-safety.md` §1; `TestPartialWindowConcentration`, `TestPartialWindowCeiling`, `TestPartialWindowNoJudgedWindow` |
| **G4.2** (Medium–High): the horizon | **Closed by a policy revision, pending re-review.** The absolute limits now cover every study of an order wanted by the horizon, observed in the follow-up. Each study is in one of four states: observed violation, unresolved overdue, not yet observable (INSUFFICIENT, never PASS), or completed in the follow-up. Unacquired orders are counted, and Gate A judges them. The one-second witness, GO under 5C.4f, is NO GO | `boundary-safety.md` §2; `TestHorizonCrossingAcquisition`, `TestHorizonFollowUpStates`, `TestHorizonUnacquiredAndLaterOrders`, `TestHorizonLinkIntegrity` |
| **G5** (Medium): separate allowances | **Disclosed; semantics unchanged.** The disjoint-seed witness gives GO with eight adverse seeds. The safeguard document's wrong "5–10 points" overlap is corrected to above 10 | `allowance-interaction.md`; `TestAllowanceInteraction`; `clinical-5c4e/seed-safeguard.md` |
| **G6.1** (Low): occupied weeks | **Fixed.** Weeks with acquisitions are counted | `WeekSevere.Occupied`; `TestOccupiedWeeks` |
| **G6.2** (Low): zero fractions when nothing is paired | **Fixed.** The fractions are null (unavailable), and they describe matched demand only | `LostDemandGate.SeedFraction`; `TestUnavailableFractions` |
| **G7** (Medium exposition): the impossibility "whatever the data" | **Corrected.** The impossibility now applies only to the all-zero sample. A sample of −0.5 differences bounds at −0.293. The 5C.4d rule's design check is kept as that rule's behaviour | `clinical-5c4d/gate-b-statistics.md` §4; `clinical-5c4e/empirical-acceptance.md`; `bounds.go`; `TestGateBStrongImprovementBound` |

## Found by the development checks

* **Construction.** Running the manifest on the development family
  found one construction defect. The fidelity model drew study
  modality from the booking stream, so v4 case selection rejected 4 of
  20 stable groups on a spurious 2–3% MR drop. It is repaired by an
  exogenous modality, used in the manifest's practices only; the
  committed model is unchanged. Afterwards 0 of 20 stable groups are
  rejected.
* **Follow-up length.** On the no-action simulator seeds, the follow-up
  leaves 1–6 routine studies a seed not yet observable (9 of 10 seeds).
  Safety is INSUFFICIENT, as specified. A 45-day follow-up is offered
  for the owner's decision (`boundary-safety.md` §2.5).

## Previous findings

| Finding | 5C.4g classification | Now |
|---|---|---|
| F5: near-margin validation | STILL OPEN | **Construction and decision rule closed.** The tolerance numbers await the owner (T5b) |
| F9: weekly reporting | REGRESSION (reporting only) | **Fixed** (G6.1) |
| F1–F4, F6–F8 | closed or closed with limitation | unchanged; F7's boundary limitation is narrowed by G4.1 |

## What 5C.4h does not claim

* **Re-review.** The 5C.4g component GO verdicts (A–E) do not carry over
  to the revised clinical policy. The partial-window ceiling and the
  follow-up cohort need a targeted independent re-review.
* **No reserved draw.** Nothing was run on PCG(20261201, ·), and the
  harness refuses it. No reference or holdout seed and no unseal
  record were used.
* **No lock and no freeze.** The protocol is not locked, and v5 is not
  frozen.
