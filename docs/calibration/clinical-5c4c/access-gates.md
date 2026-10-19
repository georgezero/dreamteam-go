# Slice 5C.4c: clinical access and weekly severity gates

This slice implements the owner's modified option B (2026-10-18) and
the volume-normalized weekly severity.

* **Version:** 0.2.0-dev.17. It registers 0.2.0-dev.16 for replay.
* **Where:** the evaluator (`internal/eval/clinical.go`,
  `internal/eval/lostdemand.go`, `internal/eval/compare.go`).
* **Reached only by a v5 policy that sets the new fields.**
  * v0.2-eval-4 is byte-identical (`TestV4Neutrality`).
  * The earlier v5 code paths are unchanged for policies without the
    fields, so the 5C.4b calibration runs still reproduce.
* **Candidate policy:** `docs/calibration/clinical-5c4b/candidate-policy-v5.json`
  (a proposal, not in force).
* **Not frozen.** Nothing else was run: no strategy, reference or
  holdout seed, unseal record, or reserved stream.

## 1. Implemented rules

Each rule below is cited to its implementation and its tests.

| Rule | Implementation | Tests |
|---|---|---|
| **Gate A, timeliness** (per seed, in the shared allowance) | Of the eligible orders (received after the takeover, original wanted time in the window), served within the limit ÷ (served, open past the limit, or ended by the patient at or after the limit). **Lost and left orders are counted apart** (`AccessCohort.Lost`). A patient outcome before the limit is withdrawn. An order open within the limit at the end of observation is undecided, so the gate is insufficient (never a pass). | C2, C5, C6, W1–W7 |
| Values | urgent 48 calendar hours, soon 7 days, routine 30 days, each **≥ 95%** (soon and routine provisional) | `candidate-policy-v5.json` |
| **Gate B, lost demand** (per plan, against the no-action baseline) | §2 | C1, C3, C4, C9, C10, W2 |
| Gates A and B | Both are required, and neither offsets the other: a gain in timeliness never excuses an increase in loss (C3). Absolute unimaged fractions are always reported (`Comparison.LostDemand[].unimagedFraction`). | C3 |
| **Weekly severity** (per seed, in the shared allowance) | §3 | C7, C8, W9 |
| Severe delay | urgent more than **12 h** past due, soon **24 h**, routine **72 h**. Each is defined apart from, and below, its absolute limit (validation refuses otherwise), so the rate rules are not redundant with the absolute ones. | `TestCandidateValidation` |
| Severe-delay rate | 5% per priority, per seed, in the per-seed windows | as built (5C.4b) |
| Absolute maxima | urgent 24 h, soon 48 h, routine 14 days past due; zero tolerance, outside the allowance | W8, W9 |
| Maximum wait | **reported only** (`maxWaitReportOnly`; no gate) | W1, W3 |
| Shared allowance | 4 of 40 seeds over service, timeliness, severity and backlog, each seed counted once | C8, W10 |

## 2. Lost-demand comparison methodology

**Unimaged demand:** the eligible orders that end lost, left or by the
patient (cancellation or no-show). Patient outcomes are included, so
that a plan cannot hide loss as cancellation (C4). The components are
reported apart.

**Pairing.**

* **Within a seed, only where the candidate's eligible demand equals the
  baseline's, order for order.** An order is identified by its receipt
  and wanted times, priority, modality and walk-in flag: what the
  patient asked for, not what became of it.
* **Groups of identical requests** (indistinguishable patients) are
  compared by counts only, and so never paired arbitrarily.
* **Premise: the simulator's demand is exogenous.**
  `TestDemandExogenousAcrossPlans` checks it. On one seed, plans with
  different bookings, losses and readings receive identical orders: no
  action; a teleradiology contract; walk-ins scheduled later together
  with another worklist policy, which changed every outcome category.
  Order identity is therefore observed, not fabricated.
* **No pairing is assumed from the seed alone.** A seed whose demand
  differs (C10) is unmatched.

**The test.**

* Over the matched orders, pooled over the seeds: b orders unimaged
  under the candidate only, c under the baseline only.
* Conditional on the discordant orders d = b + c, the exact one-sided
  upper bound π⁺ on b's share (Clopper–Pearson, at the policy's
  confidence, 0.95) gives the upper bound of the pooled increase:
  (2π⁺ − 1) · d / n.
* The gate **passes** when that bound is at most the margin, **2
  percentage points** for every priority (inclusive). It **fails** when
  the bound is above it. It is **insufficient** (NO GO) if any seed's
  demand is unmatched or the matched orders are too few (urgent 200,
  soon 400, routine 400).
* With all discordant orders one way, the bound equals the observed
  increase (C1: three urgent orders lost a seed, exactly 2 points:
  pass; four: fail). With discordance both ways it is wider (C1: four
  lost against one: point 2 points, bound above: fail).

**Prospective margin.**

* The 2-point margin is the owner-approval value proposed here, fixed
  before any strategy result.
* It is v0.2-eval-4's existing loss margin (lost referrals, pooled
  overall, +2 points), now per priority and paired.
* It is not derived from any E3 or D2 result: none was run or read.
* The confirmatory run reports the gate's false rejection on stable
  paired practices (`reserved-validation-protocol.md` T5).

**What the gate is not.** It is a **non-deterioration requirement, not a
certification of adequate access.** A practice losing 65% of its urgent
orders passes it if the plan loses no more. The absolute fractions are
reported beside every verdict, and the owner's decision stands that
baseline-relative acceptance is not proof of acceptable access.

**Conservative alternative where pairing fails** (marked for owner
approval):

* unmatched demand on any seed makes the gate insufficient, so the plan
  is NO GO;
* no unpaired comparison is substituted.

In the simulator's records pairing holds by construction (exogenous
demand). Under a future demand model where it does not, the owner
decides a replacement.

## 3. Weekly severity: denominator and low-volume rule

* **Per priority:** the studies acquired in the gated period by
  evaluator Monday week. A study is severe by the definition of §1.
* **Blocks:** weeks are taken in order and joined into blocks of at
  least `minWeeklyCohort` acquisitions (urgent 20, soon 40, routine 100;
  provisional). A short remainder joins the last block.
* **Gate:** no block may have a severe rate above **5%**
  (`maxWeeklySevereRate`; exact rational comparison).
* **Too few acquisitions:** a period with fewer acquisitions in total
  than the minimum is insufficient (C7).
* **Low-volume priorities are never escaped:** urgent work, about 7–14 a
  week, is judged in blocks of two or three weeks, and every severe study
  is in some block.
* **Calendar weeks:** the simulator's local seconds make evaluator
  weeks calendar weeks. The synthetic model's takeover instant is now a
  field: `fidAlignedTakeover` gives calendar weeks
  (`TestOCFidelityAlignedWeeks`), and the default stays t0 so the
  committed runs reproduce (`TestOCFidelityModelUnchanged`).

## 4. Adversarial tests (`internal/eval/clinical_gates_test.go`)

| | Case | Expected | Result |
|---|---|---|---|
| C1 | 30 urgent orders a seed lost (baseline images them) | Gate A passes (lost apart); Gate B fails (increase 0.20); **NO GO** | pass |
| C1 | 3 / 4 lost a seed; 4 against 1 | at the margin: pass; above: fail; mixed discordance, bound above: fail | pass |
| C2 | 30 urgent orders imaged 10 days late | Gate A fails every seed (0.80); Gate B passes; **NO GO** | pass |
| C3 | baseline 15 urgent orders late; candidate on time but 15 lost | Gate A 1.0 passes; Gate B fails; **NO GO** | pass |
| C4 | 30 urgent orders ended by the patient within an hour | withdrawn from A (passes); unimaged in B: fails | pass |
| C5 | 30 displaced twice, imaged within 40 h / at 5 days | timely / untimely (the clock never resets) | pass |
| C6 | open at the end, wanted before the horizon; wanted after the horizon in the candidate only; 10 deferred past the horizon | untimely; outside both gates (demand still paired); untimely | pass |
| C7 | urgent at 7 a week: 2 severe in a week / 1; minimum above the period's volume | block over 5%: fail; within: pass; insufficient: **NO GO** | pass |
| C8 | 4 seeds failing timeliness, weekly severity and backlog; then 5 | counted once: 4 ≤ 4, joint pass; 5: **NO GO** | pass |
| C9 | baseline lacking a seed; baseline without captures | **INVALID**; Gate B insufficient | pass |
| C10 | receipt times moved on one seed; urgent relabelled routine | unmatched: insufficient, **NO GO** (both priorities when relabelled) | pass |
| — | validation: both weekly rules; severe at the limit; report-only with a wait; no weekly minimum; margin or confidence out of range | refused | pass |

**The 5C.4b witnesses W1–W12** are updated to the new rules:

* W1 and W3: the maximum wait is reported, not gated.
* W2: lost orders are apart from Gate A and fail Gate B; patient
  outcomes before the limit are withdrawn from A but fail B.

All pass. The mutation results are in §6.

## 5. No-action simulator check (`sim-access-5c4c.json`)

Development seeds 901–910, the evaluation scenario's practice with no
action.

| Priority | Limit | Gate A pooled (per seed) | Lost or left apart | Withdrawn |
|---|---|---|---|---|
| urgent | 48 h | **0.993** (0.979–1.000) | 2,986 of 4,591 (65%) | 134 |
| soon | 7 days | **0.965** (0.951–0.981) | 5,923 of 15,491 (38%) | 703 |
| routine | 30 days | **0.979** (0.972–0.987) | 10,563 of 74,357 (14%) | 5,220 |

* **Gate A passes the baseline's served and open orders.** Soon passes
  narrowly, with a margin of 0.1 points on its worst seed.
* **Gate B cannot judge the baseline against itself.** Its absolute
  fractions (65% / 38% / 14% lost or left) are reported, not certified.
* **Weekly severity at 5%:**
  * soon fails every seed: 3 to 10 of about 15 blocks, worst blocks
    8–16%, concentrated on weekends;
  * urgent fails 6 of 10 seeds;
  * routine never fails: no routine study is more than 72 h past due.
  * This baseline reads at 0.95 without weekend reading and already
    fails v4. These are properties of the baseline, not calibration.
* **Urgent severe rate (12 h) at 5%:** exceeded on 2 of 10 seeds (6.2%,
  5.6%).
* **Absolute urgent 24 h:** violated on 8 of 10 seeds (the weekend gap).

## 6. Mutation checks

The evaluator's new rules were mutated one at a time, and the gate,
witness and paired-demand tests rerun. **10 of 10 were caught:**

* lost orders counted in timeliness again;
* patient outcomes not unimaged;
* unmatched demand paired anyway;
* the point estimate in place of the bound;
* unmatched seeds not insufficient;
* the verdict ignoring lost demand;
* no low-volume pooling;
* the short remainder dropped (first survived; `TestGateWeeklyRemainder`
  added, then caught);
* too few acquisitions passing;
* the report-only maximum wait still gated.

## 7. Remaining owner decisions

| Decision | Proposed here | Status |
|---|---|---|
| Lost-demand margin | 2 points per priority, at 95% one-sided, pooled matched orders | **owner approval** |
| Patient outcomes in Gate B | counted as unimaged (anti-gaming) | **owner approval** |
| Unpaired demand | insufficient (NO GO), no substitute | **owner approval** |
| Severity definitions | urgent 12 h, soon 24 h, routine 72 h past due | **owner decision** |
| Weekly minimum cohorts | urgent 20, soon 40, routine 100 | provisional |
| Weekly rate | 5% | provisional |
| Soon and routine timeliness | 7 and 30 days at 95% | provisional |
| Lost-demand minimum cohort | urgent 200, soon 400, routine 400 matched orders | provisional |
| Gate B per seed | not in the shared allowance (a plan-level gate) | **owner approval** |
