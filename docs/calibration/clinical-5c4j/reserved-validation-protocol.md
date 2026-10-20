# Slice 5C.4j: reserved-stream confirmatory protocol (proposed)

> **Superseded** by `../clinical-5c4l/reserved-validation-protocol.md`
> (Slice 5C.4l, after the independent 5C.4k audit). It is kept as it was
> proposed and was never locked or run.

This is the one confirmatory clinical calibration before v0.2-eval-5 is
frozen. It supersedes `../clinical-5c4h/reserved-validation-protocol.md`
and answers the independent 5C.4i audit
(`docs/audits/slice-5c4i-boundary-safety-lock-readiness-audit.md`,
findings I1–I7) under the owner's decisions of 2026-10-19.

**Status: PROPOSED, NOT LOCKED, NOT RUN.** The owner has approved the T5
tolerances (§5) and the 45-day follow-up. Locking still needs two
things:

1. **An independent review of 5C.4j passes** (5C.4k). The revised
   clinical policy (`clinical-safety.md`) and this protocol's
   adjudicator are new, so no earlier component GO carries over to them.
2. **A lock commit** re-pins the candidate's digest, this document and
   the manifest's digest, and changes nothing else. Its harness alone
   may draw the reserved family.

The reserved run is then executed once. Nothing in this protocol has
run on the reserved family, and this build's harness refuses to run it
(`reservedFamily`).

**Owner direction (2026-10-19).** A failed confirmatory run is a
documented **NO GO**. It is never followed by threshold tuning on the
same streams.

## 1. Pinned

* **Candidate policy:** `docs/calibration/clinical-5c4b/candidate-policy-v5.json`,
  SHA-256 at 0.2.0-dev.22: `9efa6e63f38de9cfbfdb829ac1a82eff114d6e5c523d32df5d46f53b8c429e7c`. It is re-pinned at the
  lock commit. Its rules:
  * Gate B: rule `empirical`, margin 2 points, safeguard 10 / 6 / 5
    points, `minSeeds` 40;
  * severity windows: partition `calendar`; windows urgent none / soon
    four weeks / routine one week; minimums 100 / 100;
  * boundary rules: `partialWindows` `rateCeiling` (every nonempty
    partial window fails above 5% observed severe), `safetyCohort`
    `ordersByHorizon` with the order–study contract, `followUpDays` 45;
  * shared allowance: 4 of 40.
* **Scenario manifest:** `scenario-manifest.json` (schema
  `dreamteam.reserved-scenario-manifest/v1`, version 2, the T5
  tolerances pinned in it), SHA-256 pinned at the lock. It is executed by
  `internal/eval/reserved_manifest_test.go` (`manRunScenario`,
  `protocolDecision`) and judged by `reservedCriteria` (§4).
* **Model:** the fidelity model at the aligned takeover, observed 45
  days past the horizon. It uses:
  * **exogenous modalities:** each order's modality is drawn with its
    demand, as in the simulator;
  * **prefix-stable draws:** each study's documentation and read times
    are drawn from its own order's identity, so a longer observation
    changes no earlier outcome (`TestFidelityFollowUpPrefixStable`;
    audit I4).

  Its tests pin it: `TestOCFidelityModelUnchanged`, `TestOCFidelityAlignedWeeks`,
  `TestOCFidelityPairedDemand`, `TestReservedManifestConstruction`.
* **Evaluator:** the lock commit's. Every committed witness test passes
  there. That is a precondition, checked before any reserved draw.

## 2. Streams and seeds

The full stream map is in the manifest. The reserved family is
PCG(F = **20261201**, ·).

* **Demand:** seed i draws its demand from PCG(F, 2i). It is the same
  for both plans of every pair, and the same seed i in every scenario.
  A pair is constructed only between practices with the same demand
  model: seasonality, day CV, volume and takeover. The harness refuses
  any other pair, and `TestReservedManifestWellFormed` checks the
  manifest for it. **No A85-against-B85 comparison exists** (audit G3):
  the seasonal practices are compared only with themselves.
* **Roles:** each role draws from PCG(F, role + 100 000·k + i):

  | Role | Offset |
  |---|---|
  | baseline booking | 10 000 000 |
  | candidate booking | 20 000 000 |
  | candidate injection | 30 000 000 |
  | baseline injection | 40 000 000 |

  With k < 100 and i < 100 000, no two roles, scenarios or seeds share a
  stream. This repairs the 5C.4f layout, where offsets of 10⁶ and 2·10⁶
  overlapped across scenarios.
* **Groups:** disjoint consecutive groups of 40 seeds. Seeds 40g to
  40g + 39 form group g.
* **Remainders:**
  * a 2,500-seed scenario has **62 groups and 20 remainder seeds**
    (2480–2499);
  * a 1,000-seed scenario has 25 groups and no remainder.

  Remainder seeds enter only **per-seed** criteria (T3, T6 and T2's
  detection). They are judged together as one marked remainder line
  (`remainder`) for those per-seed facts. They never enter a group-level
  criterion (T1, T2-conformance, T2G, T4, T5, T7), and are never
  discarded from per-seed counts (audit I5).
* **Coverage before judgment** (`manCoverage`). The adjudicator first
  checks the run against the manifest:
  * every scenario's complete groups, its remainder and every replicate
    are present exactly once;
  * every group has the right seeds in the right order;
  * every record is valid, with no construction error.

  A criterion any of whose scenarios fails these checks is INCONCLUSIVE
  unless it already FAILs. Missing, duplicated, malformed, truncated,
  invalid or unreadable output never confirms.

## 3. The protocol decision

The protocol decision is computed for each group by `protocolDecision`
on the group's 40 pairs: the candidate's records and the baseline's. A
group is **rejected** when any of the following holds, each attributed
to a rule:

| Rule | Condition |
|---|---|
| `integrity` | a record is INVALID, including any breach of the order–study contract |
| `gateB:<priority>` | Gate B is not PASS (FAIL or INSUFFICIENT) |
| `allowance` | the shared clinical allowance with the safeguard (`clinicalJointWithSafeguard`) is not PASS |
| `pooled` | a gated pooled clinical rate is FAIL or INSUFFICIENT |
| `safety` | clinical safety FAIL: an absolute violation, observed or overdue |
| `pending` | clinical safety INSUFFICIENT: a study not yet observable |
| `caseSelection` | v4 case selection is not PASS |

This is the policy's verdict without its financial rules
(`compare.go` `verdict`). The fidelity records carry no finances.

A construction error makes the group INCONCLUSIVE rather than judged:
* an infeasible injection;
* practices that do not pair;
* fewer than six soon studies for the partial-window witness.

## 4. Prospective criteria

Each criterion gives **PASS**, **FAIL** or **INCONCLUSIVE**, by the rule
stated for it, and has a mandatory consequence (§6). "Point estimate"
means the decision compares the observed proportion with the threshold.
Intervals are reported, never used for the decision, unless a criterion
says otherwise.

**Interval conventions:**
* **Each proportion:** a two-sided 95% Clopper–Pearson interval.
* **Simultaneous intervals:** within one criterion's family, Bonferroni
  Clopper–Pearson at 1 − 0.05/m for m proportions. They are reported
  only.
* **Multiplicity:** a criterion over several scenarios or priorities
  PASSes only if every member passes (an intersection–union rule). That
  needs no multiplicity adjustment of the decision.

### T1: false rejection of stable practices

| Item | Specification |
|---|---|
| Scenarios | S-A85, S-B85, S-C85, S-Cs85: each stable practice against itself on an independent booking stream, paired demand |
| Estimand | the probability that a 40-seed group is rejected (§3) |
| Estimator | rejected groups ÷ 62, per scenario (remainder seeds excluded) |
| PASS | at most **3 of 62** (0.048) rejected in every scenario |
| FAIL | 4 or more of 62 in any scenario |
| INCONCLUSIVE | incomplete coverage or a construction error in these scenarios |
| Reported | each scenario's rate with its interval and Bonferroni interval (m = 4); rejections by rule, with `pending` and the severity windows apart; partial-window ceiling failures by seed; the rate on the stress practices (A92, B92, C92, Cs92, never judged) |

### T2: concentrated harm, seed-level detection

| Item | Specification |
|---|---|
| Scenarios | H15, H20, SHOCK, OH15, OH20 |
| Estimand | the probability that the safeguard flags urgent on a seed harmed by an injected urgent loss of at least 15 points, where the realized injection is ⌈0.15 N⌉ orders or more |
| Estimator | flagged harmed seeds ÷ harmed seeds, pooled over the five scenarios and all their seeds (remainders included; harm drawn per seed with the stated probability) |
| PASS | point estimate at least **0.95**, with at least 200 harmed seeds |
| FAIL | point estimate below 0.95 |
| INCONCLUSIVE | fewer than 200 harmed seeds (expected about 430), or a construction error |
| Reported | per scenario; the Clopper–Pearson lower bound |

**Whole-evaluation rejection is reported separately and is never a
target.** The allowance tolerates four failing seeds in 40, so a
40-seed evaluation in which each seed is harmed independently with
probability 0.10 is rejected through the safeguard only when at least 5
seeds are harmed:

$$
P\{\mathrm{Bin}(40, 0.10) \ge 5\} = 0.3710,
$$

assuming every harmed seed is flagged and no other seed fails.
* **H15, H20, OH15, OH20:** the observed group rejection rate is
  reported against 0.371 and against the run's exact conditional
  prediction from each group's harmed count and its other failing
  seeds. The H10 and OH10 rates are reported too.
* **SHOCK (3%):** P{Bin(40, 0.03) ≥ 5} = 0.0233.

The 37% rejection is not 90% protection, and is never described as
such.

**T2-conformance (mandatory, deterministic).** In every group of every
T2 scenario:
* each harmed seed that is flagged is among the failing seeds of the
  allowance;
* a group is rejected whenever the allowance's failing and insufficient
  seeds exceed 4.

**PASS** if this holds in every group; otherwise **FAIL**, which is an
implementation defect.

### T2G: material global deterioration

| Item | Specification |
|---|---|
| Scenarios | L10-85, L20-85, D-85, K-85, IL10-85, IK-85, IL40-85 against A85; L10-92, K-92, IL10-92, IK-92 against A92 |
| Materiality | decided by the 5C.4c rule on the run itself, before any rejection is examined. Each trigger is judged on its own, never pooled across priorities (audit I6): a severe **soon** share of at least 1% across the complete groups; or a severe **routine** share of at least 1%; or a median worst soon delay of at least 48 h; or a Gate B unimaged seed-mean increase of at least 3 points in any priority |
| Estimand | the probability that a 40-seed group of a material scenario is rejected |
| Estimator | rejected groups ÷ 25 |
| PASS | at least **23 of 25** (0.92 ≥ 0.90) rejected in every material scenario |
| FAIL | 22 or fewer in any material scenario |
| INCONCLUSIVE | a construction error |
| Reported | non-material scenarios, never judged |

### T3: zero-tolerance admissibility

| Item | Specification |
|---|---|
| Scenarios | the four T1 scenarios, all 2,500 seeds, remainders included |
| Estimand | per scenario and per absolute limit (urgent, soon and routine past due), the probability that a seed violates it (observed or overdue) |
| PASS | **no violating seed** for any limit in any scenario. 0 of 2,500 gives a one-sided 95% upper bound of 0.1197%, at most 0.13% |
| FAIL | one or more violating seeds anywhere |
| Reported | per-limit bounds; Bonferroni bounds over the 12 limit-scenario cells; `pending` seeds, counted in T1 and T7 |

### T4: witnesses

| Item | Specification |
|---|---|
| Scenarios | every manifest scenario of group `witness` (14), 25 groups each |
| Expected | each scenario's `acceptanceRule` in the manifest. In every complete group the targeted gate or limit itself must be the cause on the stated seeds; an aggregate rejection by another rule does not count (audit I6) |
| PASS | every group of every witness meets its expectation |
| FAIL | any group does not |
| INCONCLUSIVE | incomplete coverage (fewer than 25 groups, say) or a construction error |

The targeted rule for each witness:

| Witness | Targeted rule | Expectation |
|---|---|---|
| W-URG-LATE | `safety`, the urgent past-due limit on every seed | rejected |
| W-ROUT-348 | `safety`, the routine past-due limit on every seed | rejected |
| W-ACC-LATE | `allowance`, every seed failing `timely_access gated urgent` | rejected |
| W-LOST30, W-PAT30 | `gateB:urgent` (FAIL) | rejected |
| W-ID-REMOVED, W-ID-REPEATED, W-ID-CHANGED (the wanted time moved one minute) | `gateB:urgent` (INSUFFICIENT) | rejected, the group's first seed the one unmatched |
| W-PARTIAL | `allowance` | rejected, every seed failing the soon partial-window ceiling |
| W-HORIZON | `safety`, the soon past-due limit on every seed | rejected |
| W-PENDING | `pending` on every seed | rejected |
| W-MARGIN-ABOVE | `gateB:urgent` (FAIL) | rejected |
| W-MARGIN-AT | — | Gate B urgent PASS |
| W-ACCESS65 | — | wherever Gate B urgent passes, `absoluteClinicalAdequacy` "not established" and `populationStatisticalConfidence` "not claimed" |

The audit and slice witness tests, which use synthetic fixtures rather
than reserved draws, are a precondition at the lock commit (§1). They
are not part of T4. The `T4` list here is the current expected-outcome
manifest: it is not an import of historical witnesses whose policies
have since changed.

### T5: Gate B near the margin (tolerances approved)

| Item | Specification |
|---|---|
| Scenarios | NM0, NM1, NM2, NM2.1 and NM3 on S-A85's pairs, seeds 0–999: 25 groups × 20 injection replicates = 500 evaluations each |
| Construction | each served eligible urgent order of the candidate is lost independently with probability δ·N ÷ served (N: the seed's eligible urgent orders). That gives an expected injected increase of exactly δ on every seed, added to the pair's natural difference |
| Estimand | the probability that Gate B urgent PASSes at each δ |
| Decision per replicate | Gate B alone (`gateBDecision`). The rest of the policy is judged on the same pairs without injection: S-A85 in T1 |
| Estimator | passing evaluations ÷ 500 |
| Uncertainty | between-group (cluster) standard error: the standard deviation of the 25 group means ÷ √25. The 500 evaluations are not independent |
| Prediction | for each group, the exact conditional acceptance under the injection's binomial distribution, by Monte Carlo (20 000 draws, each seed's count drawn exactly from its binomial distribution by inverse cumulative probability, for any number of served orders: audit I7) from the group's own seeds (N, served, and natural candidate and baseline unimaged counts); its mean over the 25 groups |
| **T5a conformance** | PASS when every δ's observed rate is within 4 cluster standard errors, plus 0.02, of its prediction. FAIL otherwise: the construction or the evaluator does not do what the rule says |
| **T5b tolerance** (approved) | at δ = 2.1 the observed rate is at most **0.40**, and at δ = 3 at most **0.01**, as point estimates over the 500 evaluations: at most 200 and 5 accepted. FAIL if either is exceeded |
| INCONCLUSIVE | incomplete coverage, a construction error, or fewer than two groups (no cluster standard error) |

The tolerances are pinned in the manifest (`t5Tolerances`) and read
from it by the adjudicator. They are empirical operating-characteristic
tolerances under these exact injection distributions, not population
confidence guarantees. The 500 evaluations are 25 groups × 20
replicates, so their uncertainty is the cluster standard error, not a
binomial interval. **No 5% false-acceptance tolerance is imposed**: it
would contradict the approved rule, which accepts a true 2.1-point
increase about a third of the time at urgent volumes.

### T6: safeguard false flags on no-effect pairs

| Item | Specification |
|---|---|
| Scenarios | the four T1 scenarios: unharmed by construction (no injection); 10,000 seeds |
| Estimand | per priority, the probability θ that a no-effect seed is flagged by the safeguard |
| Estimator | θ̂ = flagged seeds ÷ 10,000, per priority |
| Monte Carlo error | the standard error under the design rate θ₀ = 0.001: √(θ₀(1 − θ₀)/n) = 0.000316. It is fixed before the run and is not the observed rate's |
| Threshold | θ₀ + 3 × 0.000316 = 0.00195: **at most 19 flagged seeds in 10,000** |
| PASS | at most 19 flags in every priority |
| FAIL | 20 or more in any priority |
| Kept distinct | T6 contains no harmed seed. Failure to detect injected harm is T2 only |
| Dependence | the seeds share their demand across scenarios (paired demand), so the 10,000 seeds are not independent binomial trials. The decision is a prespecified count, and its intervals are reported for orientation only |

### T7: severity-window and safety evaluability

| Item | Specification |
|---|---|
| Scenarios | the four T1 scenarios, 62 groups each |
| Estimand | the probability that a 40-seed group has any seed raw-insufficient on any severity-window gate, or not yet observable for safety |
| Counting | **raw, per seed and per gate**, from `clinicalInsufficientSeedsByGate`. Every seed is counted that lacks window evidence on the soon or routine gate: a complete window below the minimum, or no window judged. That includes a seed failing another gate (FAIL in the joint rule), and a seed whose same gate also fails on another window; it is read from the window measures, not the gate's status (audit I6). Safety-pending seeds come from `clinicalSafety.pending`. Insufficiency is never inferred from the aggregate verdict |
| PASS | at most **3 of 62** groups in every scenario |
| FAIL | 4 or more in any scenario. The minimums are not adjusted afterwards: the confirmation fails |
| Reported | by gate; complete-window cohorts; partial-window ceiling hits |

### T8: construction integrity

| Item | Specification |
|---|---|
| All scenarios | complete coverage (§2); every record valid; every pair matched on every priority (Gate B's `matchedSeeds` = the group's pairs), except the identity witnesses' first seeds; no construction error; every quota injection exactly ⌈effect × N⌉ and every count injection exactly its count, on the candidate's side and on the baseline's |
| PASS | all of these hold |
| INCONCLUSIVE | any does not. The run cannot confirm the policy, and the family is spent |

## 5. T5 tolerances (approved)

**Status: APPROVED by the owner (2026-10-19):** τ₂.₁ = 0.40 and
τ₃ = 0.01. They are pinned in the manifest and read from it. The rest of
this section is the 5C.4h proposal's evidence, kept with the one
correction the audit made (I7, at N = 120).

### 5.1 What the approved rule does near its margin

Under the T5 construction, each seed's injected increase is
Binomial(served, δN ÷ served) ÷ N. Gate B accepts when the mean over 40
seeds is at most 2 points. With N urgent orders on each of 40 seeds, and
almost no natural difference (the fidelity model's paired practices
lose almost nothing):

| Urgent orders a seed | δ = 2.0 | δ = 2.1 | δ = 2.5 | δ = 3.0 |
|---|---|---|---|---|
| 120 | 0.527 | 0.337 | 0.013 | 1.04 × 10⁻⁵ (corrected: 5C.4h said < 10⁻⁶) |
| 150 | 0.524 | **0.314** (the audit's figure) | 0.006 | 9.1 × 10⁻⁷ |
| 180 | 0.522 | 0.294 | 0.003 | 8.1 × 10⁻⁸ |

**Development measurement, 5C.4h** (`../clinical-5c4h/construction-checks.json`; the 5C.4j re-measurement is in `construction-checks.json`): the fidelity
pairs on PCG(20261021, ·), 5 groups × 20 replicates, 120–188 urgent
orders a seed.

| δ | Accepted | Prediction from the same pairs | Cluster SE |
|---|---|---|---|
| 0 | 1.00 | 1.00 | — |
| 1 | 1.00 | 1.00 | — |
| 2.0 | 0.58 | 0.517 | 0.044 |
| **2.1** | **0.32** | 0.308 | 0.030 |
| **3.0** | **0** | 0 | — |

The standard deviation of the group means at 2.1 is 0.067. On 25 groups
the cluster standard error is about 0.013.

### 5.2 The approved tolerances and their operating characteristics

| | τ₂.₁ | τ₃ | Pass probability if the rule behaves as designed (0.31, SE 0.013) | Fail probability if acceptance at 2.1 were 0.50 |
|---|---|---|---|---|
| **Approved** | **0.40** | **0.01** | > 0.9999 | > 0.9999 |

At +3 points the designed acceptance is far below 0.01. A looser
evaluator accepting 2% of evaluations there would still pass the 0.01
tolerance about 6.5% of the time on 500 independent evaluations (audit
I7). The tolerance distinguishes appreciably looser behaviour, not
every departure. T5a's conformance check covers the finer one.

**What the approved values mean clinically:**
* a plan whose true urgent loss is 0.1 points above the 2-point margin
  may be accepted up to 40% of the time;
* one that is 1 point above it is accepted at most 1% of the time (by
  design, essentially never).

T5b would fail only if the evaluator or the construction made
acceptance near the margin materially looser than the rule's designed
behaviour. T5a checks the behaviour against the run's own prediction.

After any reserved result they cannot be changed (§6).

## 6. Consequences

* **Confirmation succeeds** only if T1, T2, T2-conformance, T2G, T3,
  T4, T5a, T5b, T6, T7 and T8 are all PASS.
* **Any FAIL** means confirmation has failed: the documented outcome is
  **NO GO** for freezing v0.2-eval-5 on this evidence. No threshold,
  minimum, tolerance or construction is then changed and re-judged on
  the reserved family. Any revision needs a new, separately authorized
  family and protocol.
* **Any INCONCLUSIVE, with no FAIL,** means confirmation was not
  achieved: also **NO GO** for freezing. The reserved family counts as
  used.
* **A waiver after the result** (accepting a missed criterion,
  re-reading a threshold, excluding a group) is a **protocol
  deviation**. It is recorded as such and is never reported as
  prespecified success.
* **The family is consumed by the run** whatever its outcome. A
  partial, aborted or repeated execution also consumes it.

## 7. Limitations carried into any result

* **Gate B is an empirical acceptance on the evaluated paired seeds.**
  It makes no population coverage claim and never establishes adequate
  access. Every favourable verdict states the access limitation and the
  empirical scope.
* **The 4-of-40 allowance is not universal.**
  * The v4 case-selection per-seed allowance is separate, so up to eight
    distinct seeds can have adverse outcomes across the two
    (`../clinical-5c4h/allowance-interaction.md`).
  * Gate B's mean and the absolute limits are outside every allowance.
* **Whole-evaluation rejection of concentrated harm is limited by the
  allowance.** It is 0.371 for independent 10% shocks. Seed-level
  detection is a different quantity.
* **The safeguard's thresholds come from the development seeds** under
  an independence approximation. T6 measures the model's false flags,
  not the simulator's.
* **Partial windows below the minimum** fail when their observed severe
  fraction exceeds 5%, whatever their size. This is a conservative safety
  veto, not an estimate: one severe report among 6 fails. At or below
  5% they are NOT_JUDGED, which establishes nothing: disclosed, never
  passed.
* **The follow-up is finite.** At 45 days past the horizon, every order
  wanted by it and imaged within its access limit has its absolute
  reporting deadline within the observation. A report due exactly at
  the end of observation and unfinished stays not yet observable
  (INSUFFICIENT). Orders imaged after their access limit can remain not
  yet observable, and an order still unimaged has no reporting clock;
  Gate A judges their access, within its 5% allowance.
* **The per-seed severe rates and the local windows remain acquisition
  cohorts of the gated period.** A severe report that does not breach
  the absolute limit, acquired after the horizon, is outside them. The
  absolute limits cover it.
* **The soon block verdict near 5% can depend on the fixed anchor.** It
  changed on 1–2 of 10 development seeds; the anchor is fixed in
  advance.
* **The fidelity model.** Its tails are heavier than the simulator's.
  The weekday priority mix is not reproduced. Its booking reserves
  capacity. Rare-event bounds are model quantities.
