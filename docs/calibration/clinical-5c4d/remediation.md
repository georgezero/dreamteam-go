# Slice 5C.4d: audit remediation and seed-level lost-demand inference

This slice responds to the independent audit
(`docs/audits/slice-5c4c-access-gates-audit.md`, kept verbatim) under the
owner's decisions of 2026-10-18.

* **Version:** 0.2.0-dev.18. It registers 0.2.0-dev.17 for replay.
* **Not frozen,** and no v5 policy is active.
* **Reserved streams untouched.** No strategy, reference or holdout run,
  and no unseal record.

## 1. Implementation findings fixed

| Finding | Fix | Test |
|---|---|---|
| **F2**: Gate B read the last capture in the array | `endCapture`: the unique capture named `end`, at the end of observation, with an outcome per order; none means the seed is unmatched | `TestAuditF2CaptureOrder`: captures permuted so T30 is last give the same gates |
| **F3**: empty or unknown outcomes passed; an unknown priority could panic | v5 integrity (`checkClinicalRecord`): every order's and study's priority known; every capture outcome known; a resolved order's final outcome terminal; otherwise INVALID and no clinical measure | `TestAuditF3UnknownEvidence` (empty, unknown, unknown order and study priorities; no panic) |
| **F8**: the allowance validated above 1; NaN and infinity passed | `clinicalFraction`: finite, in [0, 1], whole basis points, for every clinical fraction | `TestCandidateValidation` (above one, NaN, infinite) |
| **F9**: a sparse block's span was its occupied weeks | `WeeklySevere.WorstWeeks` is now calendar weeks, `WorstOccupiedWeeks` the weeks with acquisitions, `WorstFrom` / `WorstTo` the Monday and Sunday dates | `TestAuditF9WeeklySpans` |

The v5 integrity checks run only under a v5 policy: v0.2-eval-4 is
byte-identical (`TestV4Neutrality`).

## 2. Stable order identity

* **The field.** Evaluation records carry `origin` on every order: the
  simulator's existing origin key (day, source, hour block, index; walk-ins
  source −1; the opening schedule −1 / −1 / −1 / index).
* **Its properties.**
  * It is set when the order is created from the exogenous, keyed demand
    draws.
  * It is unchanged by booking, rebooking, cancellation, no-show, loss
    or acquisition.
  * No plan action can change it.
  * The recorder reads it; it draws nothing.
  * It is test-only code (the evaluation recorder), so the binary is
    unchanged.
* **Older records** have none, and Gate B then lacks evidence.
* **Tests.**
  * `TestDemandExogenousAcrossPlans`: under three plans with different
    bookings, losses and readings, every order has a unique origin, and
    each origin carries the same request under every plan.
  * `TestEvalRecordOnRealRuns`: recording changes no state digest.
  * The session replays of dev.15, dev.16 and dev.17 are exact.
* **Gate B pairs by origin only.** A missing or repeated origin, or one
  whose request differs, makes the seed unmatched and Gate B
  insufficient (`TestGateBMissingIdentity`, `TestGateNoncomparableDemand`).

## 3. Gate B: seed-level inference

The design, the impossibility result and the coverage evidence are in
`gate-b-statistics.md`.

* **The conditional discordant-pair bound is removed** from the active
  path.
* **Gate B now uses one difference per matched seed:** the candidate's
  unimaged fraction less the baseline's. Its estimand is their
  population mean, the seed-averaged increase.
* **Two procedures are implemented:**
  * `betting`, with a finite-sample guarantee for any distribution on
    [−1, 1];
  * `t`, approximate.
* **The candidate uses `betting`.** At 40 seeds its bound can never
  reach the 2-point margin: its floor is 0.207, and no valid method can
  go below 0.072. **Gate B is therefore INSUFFICIENT for every plan.**
* **This is the owner's stopping condition** (gate-b-statistics.md §7).
  A correct bound makes the gate unable to pass. It is reported, not
  weakened, and the alternatives are listed for decision.

## 4. Verdict limitation

* **The limitation.** Every favourable v5 verdict with lost-demand gates
  states, first among its reasons and in `Comparison.AccessLimitation`:
  "Access non-deterioration only; absolute access adequacy not
  established."
* **The access statement.** Every such comparison shows
  `Comparison.Access` per priority:
  * timeliness and its status;
  * the absolute unimaged fractions, candidate and baseline,
    seed-averaged and pooled;
  * Gate B's increase, bound, method and status;
  * `absoluteAdequacyEstablished`, always false.
* **The witness** (`TestAccessVerdictLimitation`): a candidate and a
  baseline both leaving 65% of urgent demand unimaged.
  * With the `t` bound, the verdict is favourable only with the
    limitation, and both 65% fractions are shown.
  * With `betting`, it is NO GO, with the access statement still shown.

Clinical, access and financial verdicts stay distinct: the limitation
does not change the financial classification, and no access gate claims
adequacy.

## 5. Weekly severity: partition sensitivity

The strict 5% local ceiling and forward pooling are unchanged. A
**backward** pooling of the same weeks is now also reported
(`backwardFailingBlocks`), and `partitionSensitive` is set when the two
disagree.

* **Boundaries** (`TestWeeklyBoundaries`):
  * exactly 5% (2 of 40) passes, 3 of 40 fails;
  * Sunday 23:00 and Monday 01:00 fall in different weeks;
  * acquisitions after the horizon are not counted;
  * an unfinished study is severe once observation shows it past the
    threshold.
* **A partition alone changes the verdict** (`TestWeeklyPartitionSensitivity`).
  * Five weeks of 20 soon acquisitions, 3 severe in the second week,
    fail forward (weeks 1–2: 3 of 40) and pass backward (3 of 60).
  * Adding one non-severe acquisition flips the forward verdict (the
    audit's F7.2).
* **In the no-action simulator records** (`sim-access-5c4d.json`, ten
  development seeds):
  * soon fails on every seed under both partitions;
  * urgent is partition-sensitive on one seed of ten (902: passes
    forward, fails backward);
  * routine never fails.
* **Freeze blocker.** The same clinical sequence can change verdict
  through a different valid partition. That happens in synthetic
  witnesses and on one simulator seed. The owner must decide before the
  freeze:
  * a partition-free rule (for example, a rolling window of the
    minimum cohort);
  * a fixed calendar partition with an explicit low-volume rule;
  * or accepting the instability with it reported.

## 6. Mutation checks

The remediation's rules were mutated one at a time, and the
remediation, gate, witness, validation and bound tests rerun.

* **All are caught:**
  * the last capture again (F2);
  * unknown priorities accepted (F3);
  * fractions unbounded (F8);
  * occupied weeks as the span (F9);
  * backward pooling not computed;
  * Gate B's design floor not checked;
  * request differences under one origin accepted;
  * a missing origin accepted;
  * thin seeds accepted;
  * the limitation not stated;
  * the betting λ cap made invalid;
  * the t quantile two-sided.
* **Four first survived** and are now caught, after
  `TestGateBOldRecordsAndThinSeeds` and `TestTBoundValue` were added:
  * a missing origin accepted;
  * thin seeds accepted;
  * the t quantile two-sided;
  * the two F3 outcome checks, each off alone. They are mutually
    redundant: each alone rejects every witness. Turned off together,
    they are caught.

## 7. Remaining blockers

1. **Gate B's statistical method** (owner decision, gate-b-statistics.md
   §7). With the guaranteed method, Gate B cannot pass at 40 seeds.
2. **The weekly partition instability** (§5).
3. **The owner approvals** still open from 5C.4c:
   * the severity definitions;
   * the weekly minimum cohorts;
   * the soon and routine timeliness thresholds;
   * patient outcomes counted as unimaged in Gate B.
4. **The reserved protocol** (`reserved-validation-protocol.md`). It is
   complete except for the Gate B method and the weekly partition rule,
   and is not runnable until they are decided.
