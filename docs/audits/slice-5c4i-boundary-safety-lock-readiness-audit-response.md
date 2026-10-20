# Response to the Slice 5C.4i boundary safety and lock readiness audit (maintainer)

This is the maintainer's account of what was done with each finding of
`slice-5c4i-boundary-safety-lock-readiness-audit.md`.

* **The audit is unchanged.** It was committed alone and verbatim
  (4bead43) before any remediation, and this document is separate from
  it.
* **The remediation is Slice 5C.4j** (0.2.0-dev.22), under the owner's
  decisions of 2026-10-19. The owner chose:
  * the observed severe-rate ceiling for partial windows (I1);
  * the T5 tolerances of 0.40 and 0.01, which were already approved;
  * a 45-day follow-up (I3).
* **Reproduced as regressions.** The auditor's witness source was kept
  outside the repository, and its cases are reproduced as tests in
  `internal/eval/remediation_5c4j_test.go` and
  `reserved_manifest_test.go`.

**Candidate digest (0.2.0-dev.22):** `9efa6e63f38de9cfbfdb829ac1a82eff114d6e5c523d32df5d46f53b8c429e7c`.

**Changed policy semantics:**
* `partialWindows: rateCeiling` (was `countCeiling`);
* `followUpDays: 45` (new);
* the order–study contract under `safetyCohort: ordersByHorizon` (new
  integrity rule).

The severity thresholds, minimums, absolute limits, anchor, Gate A,
Gate B, its margin, the safeguard and the allowance are unchanged.

## Findings I1–I7

| Finding | Status | Where |
|---|---|---|
| **I1** (High): 5 severe of 6 got a full GO | **Closed by the owner's rule.** Every nonempty partial window fails when its observed severe fraction exceeds 5%, whatever its size. At or below 5% and under 100 acquisitions it is NOT_JUDGED (establishes nothing). Completeness rules unchanged. Its failures count in the shared allowance, so it is not an unconditional veto. On the full 40-seed evaluation: 5/6, 5/10, 5/50, 5/99, 1/6 and 6/99 are NO GO; 5/100 and 0/6 are GO; 6/100 is NO GO; the 640-report exploit is NO GO. Stable-practice false rejection: §1.4 of `clinical-safety.md` | `clinical.go` (`weeklySevereCalendar`), `policy.go` (`PartialRateCeiling`); `TestRateCeilingFullVerdicts`, `TestRateCeilingWindows` |
| **I2** (High): missing and mislinked studies | **Fixed.** The order–study contract: unique study IDs; exactly one order per study, of its priority and modality, served, resolved at the acquisition; no order with two studies; every served new order with exactly one study. Inherited work is explicitly exempt. All three of the audit's bypasses (duplicate row, relink after H, deleted study) and the further breaches are INVALID | `checkOrderStudies`; `TestOrderStudyContract`, `TestOrderStudyInheritedExempt` |
| **I3** (High): the 45-day follow-up unimplemented | **Implemented.** `followUpDays: 45` enters `observationNeeded`, so every recorder observes through H + 45 d: the simulator's evaluation recorder, the fidelity model and the fixture. Validation requires at least access + target + absolute limit for each priority (routine 45 d). The time conventions are exact: equality at the limit passes; a report due exactly at E and unfinished stays INSUFFICIENT; a report after E is INVALID; nothing is fabricated | `clinical-safety.md` §3; `TestFollowUp45` |
| **I4** (High): a longer observation changed earlier fidelity outcomes | **Fixed.** Prefix-stable draws: each study's documentation and read times come from its order's identity. Earlier demand, modalities, acquisitions, documentation, reading and final times are proven unchanged from 31 to 45 days; the committed model is kept and shown changing. The simulator's prefix is also proven unchanged on development seed 901 | `prefixStableDraws`; `TestFidelityFollowUpPrefixStable`, `TestFollowUpPrefixUnchanged` |
| **I5** (High): remainders, incomplete and invalid output | **Fixed.** The adjudicator checks coverage first (`manCoverage`). It needs every scenario's exact complete groups, its one marked remainder, every replicate, the right seeds and valid records. Any gap makes the affected criteria INCONCLUSIVE (unless FAIL), and T8 INCONCLUSIVE. The remainder enters only per-seed criteria. The reader fails on a scanner error | `reservedCriteria`, `manCoverage`; `TestReservedCriteriaRules` |
| **I6** (Medium–High): T7 masking, T2G pooling, T4 attribution, W-ID-CHANGED | **Fixed.** <ul><li>**T7:** raw window insufficiency is read from the window measures.</li><li>**T2G:** soon and routine materiality are judged separately.</li><li>**T4:** every witness must fail by its targeted gate or limit on the stated seeds.</li><li>**W-ID-CHANGED:** it now moves the wanted time, as specified.</li></ul> | `clinicalPlan`; `witnessMet`; `TestReservedCriteriaRules` |
| **I7** (Medium): T5 prediction and reporting | **Fixed.** <ul><li>**Prediction:** exact binomial throughout, by inverse cumulative probability (n = 201 at 0.021: P(0) = 0.01404).</li><li>**Correction:** the N = 120 figure at +3 points is now 1.04 × 10⁻⁵.</li><li>**Tolerances:** 0.40 and 0.01 are pinned in the manifest and read from it.</li><li>**Reporting:** intervals, one-sided bounds, Bonferroni bounds and the dependence qualifications are reported.</li></ul> | `predictAcceptance`, `binomialCDF`, `manifestTolerances`; `TestBinomialCDF` |

## Validation

**Tests.** Each of the audit's cases is a regression test:
* the I1 table, with full 40-seed verdicts (`TestRateCeilingFullVerdicts`),
  and the window cases (`TestRateCeilingWindows`);
* the I2 manipulations, before (the 5C.4h candidate) and after
  (`TestOrderStudyContract`), and the inherited exemption;
* the I3 boundary table (`TestFollowUp45`);
* I4 prefix stability for the fidelity model and the simulator;
* I5–I7 adjudication (`TestReservedCriteriaRules`, `TestBinomialCDF`).

**Mutation checks.** 24 mutations, applied one at a time; all 24 caught:
* rate ceiling (4): the ceiling off; judged against the minimum's count
  instead of the rate; failing at exactly 5%; a priority with nothing
  judged passing vacuously;
* contract (6): repeated study IDs, the request (priority and modality),
  the served outcome, the resolution time, two studies on one order and a
  served new order without a study, each left unchecked in turn;
* follow-up (2): dropped from `observationNeeded`, validation relaxed;
* prefix stability (1); raw T7 (1); remainder (1); missing groups (1);
  invalid records (1); T2G pooled (2); W-PARTIAL (1); W-ID-CHANGED (1);
  normal-approximation prediction (1); tolerances not read (1);
* the simulator recorder's inherited no-shows (1).

**Development runs** (the development family only):
* all 49 scenarios at 40 seeds, plus the stable and near-margin scenarios
  at 200;
* every judged criterion passes; T2 and T5a are inconclusive by size, as
  designed (`clinical-5c4j/README.md`);
* T5b at +2.1 points: 0.31 (tolerance 0.40); at +3: 0 (tolerance 0.01).

**Stable-practice false rejection** (the owner's question): 0 of 28
stable groups rejected; 0 of 1,120 stable seeds failing the
partial-window ceiling. No material increase, so there was no stop for
owner review, and no threshold changed (`clinical-safety.md` §1.4).

**Simulator** (seeds 901–910, 45 days): routine studies not yet
observable fall from 22 on 9 seeds to 3 on 3, and seed-level results are
unchanged. A recorder defect exposed by the contract (rebooked no-shows
at the takeover) was repaired (`clinical-safety.md` §3.1).

**Compatibility:**
* v0.2-eval-4 is byte-identical;
* the 5C.4d, 5C.4e, 5C.4f and 5C.4h candidates evaluate under their own
  rules, from their exact bytes;
* 0.2.0-dev.21 is registered for replay (`mini-seed51-dev21`), and replay
  through dev.21 is exact.

**Suite:** the full uncached suite passes in every package (one package at a time, memory-capped: eval 1,350 s, sim 548 s, session 175 s). `go vet` and `gofmt` are clean.

## Limitations and open owner decisions

* **The partial-window rule is conservative by design.** One severe
  report among six acquisitions fails the window. No stable development
  practice triggered it, but sparse, front-loaded soon demand could.
  This is the owner's rule, not tuned.
* **Censoring at E remains.** A report due exactly at E, late imaging in
  the follow-up, and orders unimaged at E stay INSUFFICIENT or go to
  Gate A. They are never PASS.
* **The development results are not the protocol's.** They use 1 or 5
  groups on the development family. The reserved run judges 62 groups,
  once.
* **The recorder repair** changes which inherited orders a
  simulator-derived record carries (the rebooked no-shows). Earlier
  slices' simulator evidence documents are kept as they were.
* **For the owner:**
  * whether the 5C.4k review should also examine the recorder repair;
  * the lock commit, only after a clean 5C.4k review.

## What this response does not claim

* **Re-review.** No earlier independent component GO carries over to
  the revised candidate. The 5C.4k review must judge it.
* **No lock, reserved draw, freeze or activation.** Nothing was run on
  PCG(20261201, ·), and no reference or holdout seed or unseal record
  was used.
