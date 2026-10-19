# Slice 5C.4d: reserved-stream confirmatory protocol

The one confirmatory clinical calibration before v0.2-eval-5 is frozen.
It supersedes `../clinical-5c4c/reserved-validation-protocol.md`.

**Status: NOT RUNNABLE.** Two owner decisions are open:

* **Gate B's statistical method** (`gate-b-statistics.md` §7). With
  `betting`, Gate B is insufficient for every plan at 40 seeds.
* **The weekly partition rule** (`remediation.md` §5).

It is re-committed with both settled before any reserved run. Nothing in
it has run.

## 1. Pinned

* **The candidate policy:** `docs/calibration/clinical-5c4b/candidate-policy-v5.json`,
  SHA-256 `2bd8f152b75c91e97cfa8bb3437da8fcafb25bc6d19f4256d804e8652f48e2c6`
  at this commit (Gate B method `betting`). It is re-pinned at the
  re-commit.
* **The evaluator and model:** the commit that re-commits this protocol.
  The fidelity model runs with `Takeover: fidAlignedTakeover` and
  separate demand streams.
* **The statistical procedure:** `upperBound(method, seed differences,
  0.95, −1, 1)`, with the method named at the re-commit.

## 2. Streams

* **Reserved family:** PCG(**20261201**, ·). Never executed and never
  inspected (`git grep` / `git log -S`: protocol text only).
* **Demand of seed i:** PCG(20261201, 2·i), the same for every scenario
  (paired demand).
* **Everything else of scenario k, seed i:**
  PCG(20261201, 10⁶ + 100000·k + i).
* **A baseline's independent booking stream:** for Gate B's
  false-rejection pairs, PCG(20261201, 2·10⁶ + 100000·k + i).
* **Seeds per evaluation:**
  * 40, as the evaluation's;
  * if the owner chooses option A (more seeds for Gate B), the
    lost-demand comparison uses seeds 0…N−1 of the same family, N fixed
    at the re-commit.

## 3. Scenarios

2,500 seeds per stable and stress scenario and 1,000 per other, as in
the 5C.4c protocol, plus the following.

| Added | What it checks |
|---|---|
| **Gate B near-margin pairs:** A85 against A85 with paired demand and an urgent unimaged increase of exactly 0, 1, 2, 2.001, 2.1 and 3 points per seed, made by losing orders at random | false acceptance at and above the margin, and the bound's coverage, on the model's own records |
| **Seed-cluster dependence:** A85 against A85 with an all-or-nothing seed shock (3% of seeds lose every urgent order) | the audit's F4.2 |
| **Heterogeneous volumes:** A85 against B85 (seasonal), paired demand | the seed-averaged estimand under unequal seed sizes |
| **Identity checks:** the candidate's records with an origin removed, an origin repeated, a request changed under one origin | INSUFFICIENT on that seed, never a match |
| **Absolute-access limitation:** both plans losing 65% of urgent demand | the favourable verdict carries the limitation; NO GO under `betting` |
| **Weekly partition sensitivity:** every stable and stress practice | the forward and backward partitions' disagreement rate, reported |
| **The audit's witnesses** | F1.1 and F1.2 (one-directional and zero discordance), F2 (capture permutation), F3 (empty and unknown outcomes, unknown priority), F4.1 (duplicate-key opposite outcomes, now made impossible by identity), F6.2 and F6.3 (closing overdue orders as lost; reclassification), F7.1 and F7.2 (the weekly ceiling at a 5% rate; pooling boundaries) |

## 4. Prospective criteria

Judged once.

* **T1 (false rejection).** Every stable scenario's whole-policy
  40-seed rejection is at most 0.05. Gate B's insufficiency is counted
  as a rejection, and its share is reported apart.
* **T2 (detection).** As the 5C.4c protocol: a material deterioration is
  rejected with probability ≥ 0.90.
* **T3 (zero-tolerance admissibility).** As 5C.4c.
* **T4 (witnesses).** Every audit and protocol witness gives its
  expected outcome in every 40-seed group.
* **T5 (Gate B validity).** On the near-margin pairs:
  * coverage of the true seed-averaged increase ≥ 0.95 − 3 Monte Carlo
    errors;
  * acceptance at 2.1 and 3 points ≤ 0.05 + 3 Monte Carlo errors;
  * on the stable-against-itself pairs, Gate B's own false rejection is
    reported (and is 1 under `betting` at 40 seeds).
* **T6 (partition).** The weekly rule's forward/backward disagreement
  rate on stable practices is reported, against the owner's decision on
  partition sensitivity.

A missed target is the owner's decision. No value is changed after any
reserved result is seen; if one is, the reserved family counts as used.

## 5. Uncertainty

* Clopper–Pearson 95% intervals for every fraction.
* Simultaneous intervals for combined rejections.
* Disjoint groups of 40 judged whole.

## 6. Limitations carried into the result

* The fidelity model's heavier tails.
* The weekday priority mix, not reproduced.
* The model's reserving booking.
* Rare-event bounds that are model quantities.
* Gate B certifies non-deterioration only, never adequate access.
* The weekly rule's partition dependence.
