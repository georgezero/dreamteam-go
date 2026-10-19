# Slice 5C.4c: reserved-stream confirmatory protocol

The one confirmatory clinical calibration before v0.2-eval-5 is frozen.
It supersedes the 5C.4b draft (`../clinical-5c4b/reserved-validation-protocol.md`).

**Status: complete, NOT TO BE RUN** until:

* the independent Astra High review (`astra-review-package.md`) is done
  and its findings are answered;
* the owner approves the values marked "owner approval" in
  `access-gates.md` §7;
* this protocol is re-committed with the candidate policy's digest
  pinned (§4) after those changes.

Nothing in it has run.

## 1. Streams

* **Reserved family:** PCG(**20261201**, ·). Never executed and never
  inspected (`git grep` / `git log -S` at 264fb83: protocol text only).
* **Seed i of scenario k:**
  * **demand** from PCG(20261201, 2·i), the same for every scenario, so
    that each scenario is paired with its baseline on identical
    eligible demand (`ocFidelity.recordPaired`, as the simulator's
    exogenous demand);
  * **everything else** from PCG(20261201, 10⁶ + 100000·k + i).
* **A baseline practice's own booking stream** for the false-rejection
  pairs (T5) is PCG(20261201, 2·10⁶ + 100000·k + i).
* **If anything here changes after the review,** the k-indexing is fixed
  again in the re-committed protocol before any run.

## 2. Model

The fidelity model (`fidelity-protocol.md`), with:

* **the aligned takeover** (`Takeover: fidAlignedTakeover`): evaluator
  weeks are calendar weeks;
* **separate demand and booking streams** (§1).

`TestOCFidelityModelUnchanged`, `TestOCFidelityAlignedWeeks` and
`TestOCFidelityPairedDemand` pin these.

* **No simulator-level validity is claimed.** At the simulator's own load
  the model's tails are heavier than the simulator's, and they are not
  validated at 0.85 or 0.92.
* **Its booking reserves capacity and loses little:** Gate B's behaviour
  on the simulator's loss-dominated access is not represented. That is
  why Gate B is judged on its own witnesses (§5).

## 3. Baselines

Every judged scenario is paired with its regime's stable practice of
the same coverage family:

* A85 for the 0.85 deterioration scenarios;
* A92 for the 0.92 ones;
* each stable practice with itself on an independent booking stream
  (T5).

Gate B compares the scenario's records with the baseline's, seed by
seed, on matched demand.

## 4. Policy judged, and the grid

**Judged:** `candidate-policy-v5.json`, its SHA-256 pinned here at the
re-commit, evaluated whole: Gates A and B, the severity rules, the
absolute maxima, the shared allowance, and v0.2-eval-4's service and
backlog gates.

**Reported alongside, never judged and never used to change a value:**

| Rule | Values reported |
|---|---|
| Urgent timeliness | 24, 48, 72 h at 90% and 95% |
| Soon timeliness | 3, 7, 10 days at 90%, 95% |
| Routine timeliness | 14, 21, 30 days at 90%, 95% |
| Lost-demand margin | 1, 2, 5 points |
| Weekly severe rate | 2, 5, 10% |
| Weekly minimum cohorts | urgent 10, 20, 40; soon 20, 40, 80; routine 50, 100, 200 |
| Severe definitions | urgent 6, 12 h; soon 12, 24 h; routine 48, 72 h |
| Absolute maxima | urgent 24, 36, 48 h; soon 48, 72 h; routine 7, 10, 14 days |

## 5. Scenarios

2,500 seeds per stable and stress scenario, 1,000 per other.

| Group | Scenarios |
|---|---|
| Stable (0.85 core load) | A85, B85, C85, Cs85 |
| Weekend gap | Wg85 (equal staffing), WgU85 (equal realized utilization), R085 |
| Stress (0.92), diagnostic only | A92, B92, C92, Cs92, Wg92, WgU92 |
| Reading deterioration | L10-85, L20-85, D-85, K-85, L10-92, K-92 |
| Imaging deterioration | IL10-85, IK-85, IL10-92, IK-92, and **IL40-85** (imaging 40% lower from day 30: material loss) |
| Reporting-delay witnesses | A85 with six urgent reports a week eight days late from day 60; A85 with one routine report 348 h past due |
| Access witnesses | A85 with 30 urgent orders a seed imaged 30 days late; A85 with 30 urgent orders a seed lost; A85 with 30 urgent orders a seed ended by the patient within an hour |
| Validation | the mirror M (reported only) |

## 6. Prospective pass and fail criteria

Judged once, on the reserved streams.

* **T1 (false rejection).** Every stable scenario's whole-policy
  40-seed rejection is at most 0.05 (point estimate; the simultaneous
  95% upper bound is reported).
* **T2 (detection).** Every deterioration whose clinical consequence is
  material is rejected with probability at least 0.90.
  * **Material** means, in the reserved run itself: a severe soon or
    routine share of at least 1%; or a worst soon delay of at least 48 h
    at p50; or an unimaged fraction at least 3 points above the
    baseline's.
  * A deterioration without material consequence is reported, not
    judged.
* **T3 (zero-tolerance admissibility).** On every stable scenario, each
  absolute maximum's one-sided 95% upper violation bound is at most
  0.13% (no violating seed in 2,500).
* **T4 (witnesses).** Every reporting and access witness is rejected in
  every 40-seed group, by the rule it targets:
  * the absolute or weekly rules for the reporting witnesses;
  * Gate A for the late witness;
  * Gate B for the lost and cancelled witnesses.
* **T5 (Gate B false rejection).** Each stable practice against itself on
  an independent booking stream: Gate B's 40-seed rejection is at most
  0.05 for every priority.

**A missed target is the owner's decision.** No value is changed after
any reserved result is seen; if one is, the reserved family counts as
used.

## 7. Monte Carlo uncertainty

* Clopper–Pearson 95% intervals for every fraction.
* Simultaneous intervals for the combined rejection.
* Disjoint groups of 40 judged whole.
* The same for Gate B's pairs.

## 8. Limitations carried into the result

* The model's heavier tails.
* The weekday priority mix, not reproduced.
* The model's booking, which reserves capacity, so access losses are
  small except in the witnesses and IL40.
* Rare-event bounds that are model quantities.
* Gate B certifies non-deterioration only, never adequate access.
