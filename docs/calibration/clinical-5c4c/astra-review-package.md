# Slice 5C.4c: independent review package (Astra High)

This package prepares an independent, read-only review of the v0.2-eval-5
clinical acceptance design before the reserved-stream confirmatory run.

**Status: prepared, not yet run.** The review runs only on the owner's
instruction.

## 1. Scope

**Under review, at the commit that adds this file (0.2.0-dev.17):**

* the candidate policy `docs/calibration/clinical-5c4b/candidate-policy-v5.json`;
* the evaluator's v5 clinical rules:
  * `internal/eval/clinical.go`;
  * `internal/eval/lostdemand.go`;
  * the v5 paths of `internal/eval/compare.go` and `evaluate.go`;
  * the policy validation in `policy.go`;
* the specification:
  * `docs/calibration/clinical-5c4c/access-gates.md`;
  * `docs/calibration/clinical-5c4c/reserved-validation-protocol.md`;
  * `docs/calibration/clinical-5c4b/acceptance-design.md`,
    `order-lifecycle.md` and `access-denominators.md`;
* the evidence:
  * `findings-fidelity.md` and the fidelity protocol;
  * the no-action access files `clinical-5c4b/sim-access-5c4b.json` and
    `clinical-5c4c/sim-access-5c4c.json`;
* the tests:
  * `internal/eval/clinical_gates_test.go`;
  * `clinical_witness_test.go`;
  * `clinical_test.go`;
  * `internal/sim/demand_pairing_test.go`;
  * `internal/sim/clinical_access_test.go`.

**Out of scope:**

* any strategy;
* reference or holdout seeds;
* the reserved streams PCG(20261201, ·);
* the holdout unseal.

## 2. Questions the owner wants challenged

1. **Can the two-gate design certify a strategy that preserves
   catastrophically poor baseline access?** The simulator's no-action
   practice loses 65% of urgent orders. Gate B is non-deterioration only.
   * Is the reporting of absolute fractions enough?
   * Should an absolute floor, or a mandatory disclosure in the
     verdict, exist?
2. **Is the lost-demand gate manipulable through cancellations,
   relabelling or the observation window?**
   * Patient outcomes are counted as unimaged.
   * Relabelling makes demand unmatched (insufficient).
   * Orders wanted after the horizon are outside both gates.
3. **Are the paired comparisons valid?**
   * Is the demand key (receipt and wanted times, priority, modality,
     walk-in) a real patient identity in the simulator?
   * Is the exogeneity test (`TestDemandExogenousAcrossPlans`)
     sufficient? Could a plan change demand through any action (walk-in
     policy, hours, recall)?
   * Is the tie-group rule sound?
   * Is the conditional exact bound appropriate: one-directional
     discordance gives a bound equal to the observed increase, and the
     variation of the discordant count itself is ignored?
4. **Statistical operating characteristics.**
   * Gate A: an exact per-seed rate test, 95% at the cohort minimums.
   * Gate B: the 2-point margin, 95% one-sided bound, pooled matched
     orders.
   * Weekly blocks of a minimum cohort, at 5%.
   * The shared allowance of 4 of 40.
   * Absolute maxima at zero tolerance.
   * Is the reserved protocol's T1–T5 design adequate to judge them,
     and are its "material consequence" definitions prospective enough?
5. **Clinical validity.**
   * The severity definitions (12 / 24 / 72 h past due) against the
     absolute maxima (24 h / 48 h / 14 days).
   * The 95% timeliness among orders not lost.
   * The exclusion of patient outcomes before the deadline from Gate A
     (but not from Gate B).
   * Calendar time as the clock.
   * The weekly rule's behaviour on the simulator's soon work, which
     fails every no-action seed.
6. **Implementation.**
   * Can a v5 policy without the new fields reach the new code?
   * Is v4 byte-identical?
   * Is a missing or contradictory record ever a pass?

## 3. Constraints for the reviewer

* **Read-only.** No change to the repository. Any witness files are
  disposable, outside the repository.
* **On joofi:** a temporary directory holding only compiled test binaries
  and the documents they need, removed afterwards.
* **No strategy runs; no reference, holdout or reserved-stream access.**
* **Findings verbatim.** They are committed alone to `docs/audits/`
  before any remediation. Maintainer checks go in a separate appendix.

## 4. Deliverable

A report with:

* findings ranked by severity, each with a concrete witness or failure
  scenario;
* an answer to each question of §2;
* the reviewer's GO / NO GO for running the reserved protocol.
