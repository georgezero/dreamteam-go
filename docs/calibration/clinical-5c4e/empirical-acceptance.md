# Slice 5C.4e: empirical clinical acceptance and weekly severity closure

The owner's decisions of 2026-10-19, implemented in the v5 evaluator
(0.2.0-dev.19). Gate B is now an **empirical benchmark acceptance rule**,
a deliberate change of the claim being made, not another confidence
interval; the weekly severity is judged over **fixed Monday–Sunday
calendar weeks**, never pooled.

* **Not frozen.** The candidate policy remains a proposal
  (`../clinical-5c4b/candidate-policy-v5.json`).
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream. The simulator ran with no action on
  the development seeds 901–910; the synthetic fidelity practice ran on
  the development stream PCG(20261020, ·).
* **v0.2-eval-4 is byte-identical** (`TestV4Neutrality`); the 5C.4d
  candidate's exact bytes still load and keep their rules
  (`TestOldV5PolicyCompatibility`).

## 1. Gate B: the empirical rule (`lostDemand.rule = "empirical"`)

### 1.1 What is computed

Unchanged from 5C.4d: the candidate and the paired baseline on the same
seeds; the eligible orders (received after the takeover, wanted in the
gated period) matched by immutable origin, each with the same request;
unimaged = lost + left + patient-ended; per seed and priority, the
candidate's unimaged fraction less the baseline's.

### 1.2 What decides

Per priority, over the matched seeds:

| Condition | Status |
|---|---|
| any seed unmatched (missing, duplicated or changed identity; no baseline record or end capture) | **INSUFFICIENT**, never PASS |
| any matched seed with fewer than `minSeedOrders` eligible orders | INSUFFICIENT |
| fewer than `minSeeds` (40) paired seeds | INSUFFICIENT |
| the mean of the seed differences above `maxIncrease` (2 points), compared exactly as a rational: Σ (cᵢ − bᵢ)/nᵢ ≤ 0.02 × seeds | **FAIL** |
| otherwise | **PASS** |

The decision uses the observed values only. No bound is computed for
it, no confidence enters it, and its reasons name no bound, confidence
or population.

### 1.3 What is reported with it

* the absolute unimaged fractions of both plans, seed-averaged and
  pooled;
* the seed differences and their distribution (`seedDifferenceDistribution`:
  min, p10, median, p90, max; seeds worse, equal and better);
* the per-seed safeguard's threshold and the seeds it flags (§2);
* the matched seeds too thin to judge;
* the **research diagnostic** (`researchDiagnostic`, labelled "research
  diagnostic only: not used for acceptance"): the finite-sample
  `betting` bound on the same differences at the policy's confidence,
  the bound the design gives with no difference at all, and whether the
  bound reaches the margin (at 40 seeds it never does: 0.207 with no
  difference). It is what a population claim could and could not
  establish from these seeds. The approximate `t` method is refused by
  policy validation, even as the diagnostic.

### 1.4 The claims, kept apart

Every comparison under the empirical rule carries `accessClaims`:

| Claim | Value |
|---|---|
| `empiricalBenchmarkAcceptance` | Gate B's status over every priority, on the `pairedSeeds` |
| `populationStatisticalConfidence` | "not claimed: the empirical rule has no population coverage; the finite-sample bound is a research diagnostic" |
| `absoluteClinicalAdequacy` | "not established" |

Every favourable v5 verdict leads with, in order:

1. "Access non-deterioration only; absolute access adequacy not
   established." (unchanged, mandatory);
2. "Empirical benchmark acceptance on the evaluated paired seeds only;
   no population statistical confidence is claimed."

A 40-seed empirical PASS is a benchmark acceptance of the candidate
against its baseline on those seeds. It is not proof, nor a 95%
statement, of population non-deterioration; `gate-b-statistics.md` (5C.4d)
shows no valid method can make that statement at 40 seeds and a 2-point
margin. `TestEmpiricalNoPopulationClaim` checks the reasons of a passing
and a failing comparison for "upper bound", "95%", "confidence", "proof"
and "population non-deterioration".

### 1.5 The 5C.4d confidence rule

Still implemented (`rule` empty, method `betting` or `t`), for the
research comparison and old v5 policies. The candidate no longer uses
it.

## 2. The per-seed safeguard

`lostDemand.maxSeedIncrease`, per priority: a matched seed whose own
increase exceeds it (exactly, (cᵢ − bᵢ)/nᵢ > τ) is **deteriorated**. The
comparison's `clinicalJointWithSafeguard` is the candidate's shared
clinical allowance (`maxFailingSeedFraction` 0.1: 4 of 40) with each
such seed counted as failing, each unmatched or thin seed as lacking
evidence, every seed once with its own plan status; the verdict judges
clinical seed reliability by it in place of the plan's own joint gate.
It does not change Gate B's mean rule, and a seed flagged by it still
counts once whatever else fails on it.

The thresholds' derivation and evidence are in `seed-safeguard.md`.

## 3. Weekly severity: fixed calendar weeks (`weeklyPartition = "calendar"`)

Per priority and seed, every Monday–Sunday week that meets the gated
period (`weeklySevere.weeks`: dates, cohort, severe count, rate,
status, complete or partial):

| Week | Below `minWeeklyCohort` | At or above it |
|---|---|---|
| **complete** (Monday 00:00 to Sunday 24:00 within the period) | **INSUFFICIENT**: the seed lacks evidence | PASS, or FAIL when severe ÷ acquired > 5% (exact) |
| **partial** (the takeover's or the horizon's week) | **NOT_JUDGED**: reported, never a pass, and not held against the seed (its shortfall is the window's edge, the same for every plan) | judged like a complete week |

The seed's gate is FAIL when any judged week fails, else INSUFFICIENT
when any complete week is below the minimum, else PASS. Nothing is
pooled forward or backward; no remainder joins a neighbour. A partial
week's severe delays stay in the per-seed severe rate and the absolute
limits. Severity: urgent > 12 h, soon > 24 h, routine > 72 h past due;
the minimums 20 / 40 / 100 (provisional).

The 5C.4d partition witness (five weeks of 20 soon, 3 severe in week 2)
is now five insufficient weeks: no partition can change it.

### 3.1 Insufficient-evidence rates on development data (stop condition)

The no-action simulator records on the ten development seeds, 22
calendar weeks each (20 complete, 2 partial; `sim-access-5c4e.json`):

| Priority | Complete weeks | Acquired a week (mean, min–max) | PASS | FAIL | INSUFFICIENT | Seeds with an insufficient week | Seeds with a failing week |
|---|---|---|---|---|---|---|---|
| urgent | 200 | 6.7, 1–14 | 0 | 0 | **200** | **10 of 10** | 0 |
| soon | 200 | 41.5, 24–66 | 86 | 50 | **71** | **10 of 10** | 10 of 10 |
| routine | 200 | 305, 224–339 | 200 | 0 | 0 | 0 | 0 |

**The urgent rule is systematically unevaluable.** The practice acquires
about 6.7 urgent studies a week (at most 14 on any seed); a cohort of
20 needs three weeks. Under fixed calendar weeks every complete urgent
week on every seed is insufficient, so every seed lacks evidence and
every plan fails the shared allowance: no 40-seed evaluation can pass.
The soon rule is insufficient on 36% of complete weeks and on every
seed (each has at least one week of 24–39). Routine is evaluable.

This is the prompt's stop condition. The minimums and the partition are
unchanged; the owner's options are:

* **a lower urgent minimum** (the weekly acquisitions make 5 the
  largest minimum met by most weeks, which is too few for a 5% rate: one
  severe study is 20%);
* **a fixed multi-week calendar block for urgent** (for example four
  fixed calendar weeks, Monday-aligned from the first complete week:
  partition-free, cohort about 27);
* **no weekly rule for urgent** (the per-seed severe rate over the
  gated period, 5% of about 160, and the absolute 24 h limit remain);
* **a lower soon minimum** (30 would leave about 5% of weeks
  insufficient; 25 about 1%).

The synthetic fixture (7 urgent a week) shows the same
(`TestCalendarUrgentUnevaluable`).

## 4. Gate A (unchanged, approved values)

Urgent ≥ 95% within 48 calendar hours, soon within 7 days, routine
within 30 days, from the original wanted time; overdue open orders are
untimely; rebooking does not reset the clock; patient cancellations
before the deadline are withdrawn from Gate A and counted unimaged under
Gate B; lost and left are counted apart. Absolute adequacy is not
certified.

## 5. Witnesses (`internal/eval/empirical_acceptance_test.go`)

Synthetic fixtures only (40 seeds, one urgent order a day). The Gate B
witnesses use the candidate with the pooled weekly blocks, because the
fixture's 7 urgent a week cannot meet the calendar minimum (§3.1).

| # | Case | Result |
|---|---|---|
| 1 | mean increase exactly +2 points (3 of 150 lost on every seed) | PASS (the exact rational decides) |
| 2 | just above (one seed loses 4: 121/6000) | FAIL, NO GO |
| 3 | below (2 or 3) | PASS |
| 4 | concentrated harm: seed 7 loses 30 points, mean 0.75 points | Gate B PASS; seed 7 flagged, failing in the comparison's allowance (1 of 4); with four other seeds already failing, NO GO |
| 5 | harm offset: baseline loses 20 a seed, candidate 65 on five seeds and none elsewhere (mean −7.9 points) | Gate B PASS; five seeds flagged, allowance exceeded, NO GO; without the safeguard the clinical rule passes it |
| 6 | missing baseline (no record; no captures) | INSUFFICIENT |
| 7 | mismatched origin on one seed | INSUFFICIENT, that seed without evidence in the allowance |
| 8 | duplicate origin | INSUFFICIENT |
| 9 | the same 30 orders lost or relabelled patient-cancelled | the same Gate B FAIL; Gate A withdraws the cancellations and counts the lost apart |
| 10 | all outcomes unimaged, both plans | Gate B PASS at 1.0 against 1.0, Gate A insufficient, NO GO; the candidate alone: +100 points, every seed flagged |
| 11 | zero observed deterioration | PASS; the claims apart; the betting diagnostic 0.207, labelled, not decisive; no bound in the decision |
| 12 | an empty complete week | INSUFFICIENT |
| 13 | a week exactly at the minimum (40) | judged |
| 14 | just below (39), and 39 then 100 | INSUFFICIENT, not pooled |
| 15 | exactly 5% (2 of 40) / 3 of 40 | PASS / FAIL |
| 16 | Sunday 23:59:59 and Monday 00:00:00 | different weeks |
| 17 | partial first and last weeks | judged when at the minimum (and failing above the rate); NOT_JUDGED below it, never a pass, not insufficient |
| 18 | the 5C.4d candidate's exact bytes | loads; confidence rule and pooled blocks kept; no empirical field |
| 19 | v4 | byte-identical (`TestV4Neutrality`) |
| 20 | historical replay | dev.15–dev.18 sessions exact; golden references exact |

Policy validation refuses: `t` under the empirical rule; a safeguard at
or below the mean margin; a safeguard under the confidence rule; an
unknown rule or partition; a partition without the weekly rate.

## 6. Mutation checks

Each rule turned off alone, the 5C.4e witnesses and the Gate B and
limitation tests rerun. **All 14 caught:**

* the mean rule never failing; the mean compared as a float;
* flagged seeds not failing in the allowance; unmatched seeds not
  lacking evidence in it; the safeguard compared with the mean margin;
* the verdict using the plan's allowance instead of the safeguard's;
* the empirical scope not stated; the diagnostic claiming the margin
  established;
* partial weeks below the minimum made insufficient; complete weeks
  below it not judged; the pooled blocks used instead; a week above the
  rate not failing;
* `t` accepted under the empirical rule; a safeguard at the margin
  accepted.
