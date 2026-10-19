# Slice 5C.4d: Gate B statistical design (seed-level lost-demand inference)

Written before Gate B's new implementation. It replaces the conditional
discordant-pair construction that the audit showed invalid
(`docs/audits/slice-5c4c-access-gates-audit.md` F1, F4).

## 1. Estimand

* **Seed difference.** For priority q and seed s:
  D_s = c_s − b_s, where c_s and b_s are the candidate's and the paired
  baseline's **unimaged fractions** of that seed's matched eligible
  orders.
  * **Eligible:** received after the takeover and wanted in the gated
    period.
  * **Unimaged:** ending lost, left or by the patient.
  * **Matched:** the same orders under both plans, identified by the
    immutable origin (§3).
* **Estimand:** μ_q = E[D_s], the **population mean of seed
  differences** (owner decision, 2026-10-18). Every seed weighs the
  same, whatever its volume.
* **Hypothesis of the gate:** non-deterioration, μ_q ≤ 0.02 (the
  prospective margin, inclusive). Gate B passes only when a one-sided 95%
  upper confidence bound U_q of μ_q is at most 0.02.

## 2. Sampling assumptions

* **The independent unit is the seed.** Seeds are independent and
  identically distributed draws of the scenario's randomness.
* **Orders within a seed are dependent:** they share capacity, staffing,
  queues and calendar. The procedure never treats them as independent
  observations. A seed contributes one number, D_s, however many orders
  it has.
* **Range.** D_s ∈ [−1, 1], and nothing smaller is assumed: a seed can
  lose all its orders.
* The evaluation's seeds are fixed in advance. Every gate uses all of
  them; there is no sequential stopping.

## 3. Matching

* **Identity.** The simulator's evaluation records carry each order's
  `origin`, the order's exogenous demand key (day, source, hour block,
  index). It is fixed at creation, unchanged by booking, rebooking,
  cancellation, loss or acquisition, and recorded without any draw
  (`TestDemandExogenousAcrossPlans`).
* **A seed is matched** when the candidate's and baseline's eligible
  orders have the same set of origins. Each origin must also carry the
  same request: receipt and wanted times, priority, modality, walk-in.
* **Insufficient, never a favourable match:**
  * an order with no origin;
  * an origin twice in one record;
  * a set or request that differs.
* No pairing is ever inferred from timestamps (audit F4).

## 4. The impossibility at 40 seeds

**Any procedure that guarantees 95% coverage of μ for every
distribution of D on [−1, 1] gives, on 40 seeds that all show no
difference, an upper bound of at least 7.2 points.**

Proof:

* Let D = 1 with probability p and 0 otherwise, so μ = p.
* All n seeds show D = 0 with probability (1 − p)ⁿ.
* For coverage at that p, the bound on the all-zero sample, u₀, must be
  at least p whenever (1 − p)ⁿ > 0.05.
* So u₀ ≥ 1 − 0.05^(1/n): 0.072 at n = 40, 0.020 at n = 149.

This is the audit's correlated-shock witness F4.2 in general form. A
practice in which an occasional seed loses everything is not
distinguishable, in 40 seeds, from one that never does.

**Consequence.** With 40 seeds, no distribution-free valid bound can
establish non-deterioration within 2 points **from the all-zero
sample**: identical outcomes on every seed. Establishing the margin
from that configuration needs at least 149 seeds, and more for
realistic procedures (§5).

> **Correction (Slice 5C.4h, audit G7, 2026-10-19).** This section
> previously said the margin could never be established at 40 seeds
> "whatever the data". That overstated the result. The argument above
> bounds only the all-zero sample. It is not the smallest bound over all
> possible samples. Strong observed improvements can give a valid bound
> below the margin with 40 seeds. The `betting` bound of 40 seed
> differences of −0.5 each is −0.2926, against 0.2074 for 40 zeros
> (`TestGateBStrongImprovementBound`).
>
> * **The 5C.4d confidence rule.** Its INSUFFICIENT for every plan at 40
>   seeds came from its own design check, not from this impossibility:
>   Gate B is insufficient when the all-zero bound exceeds the margin
>   (§5). That check is kept unchanged, as that historical rule's
>   behaviour.
> * **The empirical rule (5C.4e onward).** It uses no bound for
>   acceptance, and this correction gives it none.

## 5. Procedures implemented (`internal/eval/bounds.go`)

| | `betting` (finite-sample) | `t` (approximate) |
|---|---|---|
| Bound | U = inf{m : Kₙ(m) ≥ 1/α}, the capital Kₙ(m) = Π(1 + λᵢ(m − Dᵢ)) with λᵢ predictable (from D₁…Dᵢ₋₁ only), 0 ≤ λᵢ ≤ c/(b − a), c = 3/4, seeds in their evaluation order | U = D̄ + t₀.₉₅,ₙ₋₁ · s / √n |
| Guarantee | **finite-sample**: P(U < μ) ≤ α for every distribution on [−1, 1], any n (Ville's inequality on the nonnegative martingale Kₙ(μ)) | **none in finite samples**; coverage approaches 95% for large n when D has finite variance (central limit theorem); fails for rare large shocks (§6) |
| Correlated orders | irrelevant (one value per seed) | irrelevant (one value per seed) |
| Heterogeneous seed volumes | the estimand weighs seeds equally; volume changes only D's spread | the same |
| Zero variance | the bound is the method's floor at that value (all-zero data: **0.207** at 40 seeds) | U = D̄ (zero width): exact when the differences are truly constant; anti-conservative when a sample merely shows no variance (all-or-nothing shocks) |
| Near the margin | conservative | close to nominal when D is near normal |
| Minimum seeds | the floor (all-zero data) is 0.207 at 40 seeds, 0.081 at 100, 0.054 at 149, 0.027 at 300 and **0.020 at 400**; the best any valid method can do is 1 − 0.05^(1/n): 0.072 at 40 | 2 |

**λ rule (power only, not validity):** the predictable plug-in
λᵢ = min(√(2 ln(1/α) / (n σ̂²ᵢ₋₁)), c/(b − a)), with σ̂² a running
variance from a prior of 0.25.

**Gate rule** (both procedures):

* **insufficient** when:
  * a seed is unmatched or its identity evidence is missing;
  * a seed has fewer matched orders than the policy's per-seed minimum;
  * there are fewer seeds than the policy's minimum;
  * or the procedure's bound for an all-zero sample of the same size
    exceeds the margin: the rule's design check. It is not an
    impossibility for every sample, since strong observed improvements
    can bound below the margin (§4, corrected);
* **fail** when U > margin;
* **pass** otherwise.

Every gate reports:

* the seed differences;
* their mean, standard deviation and extremes;
* U and the zero-difference floor;
* the absolute unimaged fractions, seed-averaged and pooled.

## 6. Coverage evidence

`internal/eval/bounds_test.go`, `TestGateBCoverage`. Its table is
committed as `gate-b-coverage.json`, 2,000 replicates per scenario,
Monte Carlo standard error 0.0049 at 95%.

* **Scenarios** (simulated seed differences, built from simulated paired
  orders, not reserved streams):
  * true increase 0, 1, 2, 2.001, 2.1 and 3 points;
  * zero, one-directional and bidirectional discordance;
  * perfect (all-or-nothing) and strong within-seed correlation;
  * heterogeneous volumes;
  * heavy-tailed differences;
  * 10, 20 and 40 seeds;
  * constant differences.
* **Measured:** coverage P(U ≥ μ) and acceptance P(U ≤ 0.02), with
  Monte Carlo standard errors.

| Scenario (40 seeds of 200 orders unless stated) | True μ | Coverage, betting | Coverage, t | Acceptance, betting | Acceptance, t |
|---|---|---|---|---|---|
| increase 0 | 0 | 1.000 | 0.952 | 0 | 1.000 |
| increase 1 point | 0.010 | 1.000 | 0.946 | 0 | 0.988 |
| increase 2 points (at the margin) | 0.020 | 1.000 | 0.948 | 0 | 0.052 |
| increase 2.001 points | 0.020 | 1.000 | 0.951 | 0 | 0.048 |
| increase 2.1 points | 0.021 | 1.000 | 0.957 | 0 | 0.021 |
| increase 3 points | 0.030 | 1.000 | 0.945 | 0 | 0 |
| zero discordance | 0 | 1.000 | 1.000 | 0 | 1.000 |
| one-directional, 2.1 points | 0.021 | 1.000 | 0.940 | 0 | 0.016 |
| bidirectional, 2.1 points | 0.021 | 1.000 | 0.945 | 0 | 0.036 |
| **all-or-nothing seed shocks, 3 points** | 0.030 | 1.000 | **0.699** | 0 | **0.301** |
| strong within-seed correlation, 2.1 points | 0.021 | 1.000 | 0.952 | 0 | 0.039 |
| heterogeneous volumes (20–600), 2.1 points | 0.021 | 1.000 | 0.946 | 0 | 0.029 |
| heavy-tailed differences (t₂), 2.1 points | 0.021 | 1.000 | 0.953 | 0 | 0.027 |
| 10 seeds, 2.1 points | 0.021 | 1.000 | 0.939 | 0 | 0.047 |
| 20 seeds, 2.1 points | 0.021 | 1.000 | 0.948 | 0 | 0.029 |
| constant differences, 2.1 points | 0.021 | 1.000 | 1.000 | 0 | 0 |
| constant differences, 0 | 0 | 1.000 | 1.000 | 0 | 1.000 |

* **`betting`** keeps its guarantee everywhere (coverage 1.000) and,
  at 40 seeds, never accepts: Gate B is insufficient by design (§4).
* **`t`** is close to nominal in every smooth scenario (0.939–0.957,
  within about two Monte Carlo errors of 0.95). Its false acceptance
  just above the margin is about the nominal 5% (2.1–4.8%).
* **But `t` fails where the audit said it would.** Under all-or-nothing
  seed shocks it accepts a 3-point increase 30% of the time. The
  samples there show no variance although the distribution has it, the
  case in §4. Its guarantee is asymptotic, not finite-sample.

## 7. Recommendation and owner decision

* **Active candidate method:** `betting`, the only one with the
  required guarantee. Gate B is then **INSUFFICIENT for every plan at 40
  seeds**, through the rule's design check on the all-zero sample (§4,
  corrected), so every plan is NO GO on access.
* **This is the owner's stopping condition:** a correct bound makes the
  gate unable to pass, and is reported rather than weakened.
* **Alternatives**, each needing explicit owner approval:

| Option | What changes | What is given up |
|---|---|---|
| A. Keep `betting` and run Gate B on more seeds | about 400 seeds per plan for the lost-demand comparison only (this procedure's floor reaches 2 points at 400; no valid procedure can below 149) | runtime: 10× the seeds for the baseline and every candidate; a new reserved-protocol design |
| B. `t` (approximate) | usable power at 40 seeds | finite-sample coverage: its undercoverage under shocks and zero variance is measured (§6) and must be accepted explicitly |
| C. A prespecified range assumption \|D_s\| ≤ R, with a finite-sample procedure on [−R, R] | power at 40 seeds if R is small (e.g. 0.10) | validity rests on an unverifiable assumption; a seed observed beyond R would be reported as a breach |
| D. A wider margin | a finite-sample bound can pass at 40 seeds only if the margin exceeds the floor (≥ 20.7 points for `betting`; ≥ 7.2 for any valid method) | clinical meaning |
| E. An empirical rule without a confidence claim | e.g. mean seed difference ≤ 2 points and no seed above k points | the population claim (the audit's remediation option 6) |
