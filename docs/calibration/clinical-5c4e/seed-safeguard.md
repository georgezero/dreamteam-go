# Slice 5C.4e: the per-seed deterioration safeguard

A proposal for the owner's approval (`lostDemand.maxSeedIncrease` in the
candidate policy). The thresholds were fixed by a rule written before
any plan was compared, from development-only evidence; E3 and every
other strategy were not run.

## 1. The rule

For each priority, the threshold τ is

> τ = max(5 points, ⌈100 · z₀.₉₉₉ · s_max⌉ / 100)

where s_max is the largest standard deviation, among the development
seeds, of a seed's no-effect difference under independent outcomes,
s = √(2u(1 − u)/n), with u the no-action plan's Gate B unimaged
fraction and n its eligible orders on that seed; z₀.₉₉₉ = 3.09.

* **Why a no-effect scale.** The safeguard must not flag a seed whose
  difference is booking noise around zero (the same practice, another
  booking path). The independence approximation gives the noise's
  scale from the development records alone; at z₀.₉₉₉ a no-effect seed
  is flagged about once in a thousand, so forty no-effect seeds are all
  clear with probability about 0.96, and the safeguard does not by
  itself consume the shared allowance.
* **Why a floor of 5 points.** The per-seed rule must not repeat the
  mean rule: a threshold near 2 points would fail a plan on the seeds
  the mean rule already weighs. Five points is two and a half margins:
  a seed losing that much more of its demand is materially worse even
  when the mean is within the margin. Policy validation refuses a
  threshold at or below the margin.
* **Why the largest seed's sd.** The scale differs by seed only through
  n; the largest keeps the rule conservative on every seed.
* **Why development evidence.** The development seeds 901–910 are the
  only simulator seeds permitted; the reference and holdout seeds, and
  the reserved streams, were not touched.

## 2. The derivation (`sim-access-5c4e.json`, no action, seeds 901–910)

| Priority | Eligible orders a seed | Unimaged fraction | s_max | z · s_max | **τ** |
|---|---|---|---|---|---|
| urgent | 434–484 | 0.66–0.71 | 0.0320 (seed 904) | 0.099 | **0.10** |
| soon | 1505–1641 | 0.39–0.44 | 0.0180 (seed 902) | 0.056 | **0.06** |
| routine | 7268–7543 | 0.19–0.21 | 0.0068 | 0.021 | **0.05** (the floor) |

`TestSafeguardThresholdsFromDevelopment` recomputes the rule from the
committed file and requires the candidate's values.

Against the mean rule: urgent 10 points is five margins, soon three,
routine two and a half.

**Against the allowance (corrected in Slice 5C.4h, audit G5).** The v4
case-selection per-seed served-drop rule (5 points, with its own
allowance of four seeds in 40) stays in force beside the safeguard.
* **Above 10 points, both can flag a seed.** A seed losing more than 5
  points of urgent service can fail case selection; the safeguard flags
  it only above 10 points. The text previously said "5–10 points",
  which was wrong: the safeguard never flags at or below 10.
* **The two counts are separate.** A seed losing 5–10 points can fail
  case selection alone. Case selection counts it in its own allowance;
  the safeguard does not flag it. The safeguard's seeds count in the
  clinical allowance.
* **So the four-seed allowance is not universal.** The two allowances
  may cover disjoint seeds. Four seeds can fail case selection and four
  different seeds a clinical gate, and the verdict stays favourable
  (`TestAllowanceInteraction`; `../clinical-5c4h/allowance-interaction.md`).

## 3. Synthetic evidence (`seed-safeguard-evidence.json`)

The fidelity practice with paired demand, 200 seeds on the development
stream PCG(20261020, ·), the candidate's booking drawn from another
stream than the baseline's in every scenario (so the no-effect path
divergence is always present). Seed differences in points:

| Scenario (urgent) | mean | sd | p90 | p99 | max | flagged at τ = 10 | at 5 | at 2 |
|---|---|---|---|---|---|---|---|---|
| no effect | 0.00 | 0.23 | 0.0 | 0.7 | 1.2 | 0 | 0 | 0 |
| imaging −10% from day 30 | 0.55 | 0.64 | 1.4 | 2.1 | 2.7 | 0 | 0 | 0.02 |
| imaging −20% from day 30 | 5.56 | 2.88 | 9.3 | 13.7 | 14.1 | 0.08 | 0.49 | 0.92 |
| no effect + 5% injected urgent loss | 5.01 | 0.31 | 5.3 | 5.9 | 6.4 | 0 | 0.50 | 1 |
| + 10% | 10.03 | 0.29 | 10.3 | 10.8 | 11.1 | 0.50 | 1 | 1 |
| + 15% | 15.03 | 0.31 | 15.3 | 15.8 | 16.4 | **1** | 1 | 1 |
| + 20% | 19.99 | 0.27 | 20.3 | 20.4 | 21.1 | **1** | 1 | 1 |

Soon and routine: no scenario moves them (soon's booking tolerance
absorbs the cuts; routine's no-effect sd 0.38 points, max 1.1 points);
nothing flagged at any threshold.

* **Detection.** An urgent loss of 15 points or more on a seed is
  flagged on every seed; 10 points, at the threshold, on half (by
  construction); 5 points never at τ = 10 (that is the mean rule's
  region: a plan losing 5 points on every seed fails Gate B's mean).
* **No false flags.** No no-effect or mildly deteriorated seed is
  flagged at any threshold of 2 points or more. The model's no-effect
  scale (0.23 points) is a tenth of the simulator-derived s_max
  (3.2 points), because the fidelity practice loses almost no urgent
  demand (0.1% unimaged against the simulator's 66%): the evidence
  checks the detection side and the absence of false flags on the
  model, not the simulator's no-effect scale, which the rule takes
  from the development records.
* **Offset harm.** The 20% imaging cut (mean 5.6 points) fails the
  mean rule on its own; the safeguard additionally flags its worst 8%
  of seeds, which is the intended overlap: the safeguard is for harm
  the mean rule would not see.
* **One reading regime.** The fidelity practice's booking, and with it
  every unimaged outcome, does not depend on reading, so the 0.92
  regime gives the same differences (checked on 8 seeds: identical).

## 4. What the safeguard does and does not do

* It flags a seed on the difference of its own unimaged fraction, by
  priority, above τ; the comparison counts the seed as failing in the
  shared clinical allowance, once.
* It does not alter the mean rule, and a seed it flags is still in the
  mean.
* It does not claim a false-flag rate on the simulator: that is the
  reserved protocol's T6, on the model.
* It is a proposal: the owner may set other values, but any value at
  or below the 2-point margin is refused as a repeat of the mean rule.
