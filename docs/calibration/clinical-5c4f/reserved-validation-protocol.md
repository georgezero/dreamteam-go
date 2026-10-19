# Slice 5C.4f: reserved-stream confirmatory protocol

The one confirmatory clinical calibration before v0.2-eval-5 is frozen.
It supersedes `../clinical-5c4e/reserved-validation-protocol.md`.

**Superseded** by `../clinical-5c4h/reserved-validation-protocol.md`
(2026-10-19). The independent 5C.4g audit found it NO GO (G1–G3). Kept
as written.

**Status: COMPLETE, NOT RUN.** Every owner decision it needs is made
(2026-10-19). The windows are evaluable on every development seed
(`severity-windows.md` §4). It is to be locked by the owner after the
independent review, re-pinned at that commit and run once. Nothing in
it has run.

**Gate B's claim.** Gate B is an empirical benchmark acceptance rule on
the evaluated paired seeds. It has **no 95% population coverage**: the
protocol measures its empirical operating characteristics (how often it
accepts or rejects under each scenario) and reports them as such. The
finite-sample `betting` bound is computed beside it as a research
diagnostic only, never as a criterion.

## 1. Pinned

* **The candidate policy:** `docs/calibration/clinical-5c4b/candidate-policy-v5.json`,
  SHA-256 `d901fe5877684567c6c2d19ad1c063e8cf36ce269ba6fd44ba54299f052e348b`
  at this commit: rule `empirical`, margin 2
  points, safeguard 10 / 6 / 5 points (approved; `../clinical-5c4e/seed-safeguard.md`),
  `minSeeds` 40, partition `calendar`, windows urgent none / soon
  four weeks / routine one week, minimums 100 / 100. It is
  re-pinned at the re-commit.
* **The evaluator and model:** the commit that re-commits this protocol.
  The fidelity model runs with `Takeover: fidAlignedTakeover` and
  separate demand streams.
* **The decision:** `lostDemandGates` under the empirical rule and
  `seedSafeguardJoint`; `weeklySevereCalendar` with each priority's
  window.

## 2. Streams

* **Reserved family:** PCG(**20261201**, ·). Never executed and never
  inspected (`git grep` / `git log -S`: protocol text only).
* **Demand of seed i:** PCG(20261201, 2·i), the same for every scenario
  (paired demand).
* **Everything else of scenario k, seed i:**
  PCG(20261201, 10⁶ + 100000·k + i).
* **A baseline's independent booking stream:** PCG(20261201,
  2·10⁶ + 100000·k + i).
* **Seeds per evaluation:** 40, as the evaluation's; disjoint groups of
  40 judged whole.

## 3. Scenarios

2,500 seeds per stable and stress scenario and 1,000 per other, as in
the 5C.4c protocol, plus the following.

| Added | What it checks |
|---|---|
| **Gate B near-margin pairs:** A85 against A85 with paired demand and an urgent unimaged increase of exactly 0, 1, 2, 2.001, 2.1 and 3 points per seed, made by losing orders at random | the empirical rule's acceptance rate at and about the margin (reported, no coverage target) |
| **Concentrated harm:** A85 against A85 with 3% of seeds losing 10, 15 and 20 points of urgent demand, the rest unchanged | the safeguard's flag rate on the harmed and the unharmed seeds; the mean rule alone would pass |
| **Offset harm:** the candidate better by 5 points on 90% of seeds and worse by 10, 15 and 20 on 10% | the safeguard's rejection through the allowance |
| **Seed-cluster dependence:** an all-or-nothing seed shock (3% of seeds lose every urgent order) | the safeguard flags every shocked seed |
| **Heterogeneous volumes:** A85 against B85 (seasonal), paired demand | the seed-averaged estimand under unequal seed sizes |
| **Identity checks:** an origin removed, repeated, a request changed under one origin | INSUFFICIENT on that seed, never a match |
| **Absolute-access limitation:** both plans losing 65% of urgent demand | the favourable verdict carries the limitation and the empirical scope |
| **Severity windows:** every stable and stress practice | the share of complete soon blocks and routine weeks insufficient; the share of 40-seed evaluations with any seed insufficient on a window; the soon block cohort distribution |
| **Anchoring:** every stable practice, soon blocks re-summed at anchor offsets 1–3 weeks (reported only) | the share of seeds whose soon status changes with the anchor |
| **The audit's witnesses** | F1–F7 as in the 5C.4d protocol, plus the 5C.4e witnesses 1–20 and the 5C.4f window witnesses |

## 4. Prospective criteria

Judged once.

* **T1 (false rejection).** Every stable scenario's whole-policy
  40-seed rejection is at most 0.05. Insufficiency is counted as a
  rejection, and the weekly rule's share of it is reported apart.
* **T2 (detection).** A material deterioration (the 5C.4c protocol's
  list, and concentrated urgent loss of 15 points on 10% of seeds) is
  rejected with probability ≥ 0.90.
* **T3 (zero-tolerance admissibility).** As 5C.4c.
* **T4 (witnesses).** Every audit and protocol witness gives its
  expected outcome in every 40-seed group.
* **T5 (Gate B, empirical).** On the near-margin pairs the acceptance
  rate at 0, 1, 2, 2.001, 2.1 and 3 points is reported with
  Clopper–Pearson intervals. No coverage target is set, because the
  rule claims none; the rates at 2.1 and 3 points are reported against
  the owner's tolerance for false acceptance, which the owner states
  before the run.
* **T6 (safeguard).** On the unharmed seeds of every scenario the
  safeguard's flag rate is at most 0.001 + 3 Monte Carlo errors; on
  seeds harmed by 15 points or more it is at least 0.95.
* **T7 (window evaluability).** On every stable practice the share of
  40-seed evaluations in which a severity window leaves any seed
  insufficient is at most 0.05. Above it, the minimums are the owner's
  decision; they are not adjusted after the result.

A missed target is the owner's decision. No value is changed after any
reserved result is seen; if one is, the reserved family counts as used.

## 5. Uncertainty

* Clopper–Pearson 95% intervals for every fraction.
* Simultaneous intervals for combined rejections.
* Disjoint groups of 40 judged whole.

## 6. Limitations carried into the result

* Gate B is an empirical acceptance on the evaluated seeds: no
  population coverage, and never adequate access.
* The safeguard's thresholds are development-derived under an
  independence approximation (`seed-safeguard.md` §1).
* The fidelity model's heavier tails, the weekday priority mix not
  reproduced, the model's reserving booking, rare-event bounds that are
  model quantities.
* The soon block verdict near 5% can depend on the fixed anchor (1–2
  of 10 development seeds); the anchor is fixed in advance.
* The safeguard is an empirical benchmark safeguard: no calibrated
  false-flag guarantee.
