# Slice 5C.4b closure: reserved-stream validation protocol (draft)

The confirmatory clinical calibration that runs once, before
v0.2-eval-5 is frozen.

**Status: DRAFT, not runnable.** It may be run only after:

* **the owner's two open decisions** (`acceptance-design.md` §2 and §3):
  the urgent access design (option A, B or C) and the weekly count;
* **this protocol completed with them and committed**, with the access
  grid of §4 replacing its placeholder;
* **the model changes in §3** implemented and committed, with their
  tests.

Nothing in it has run. **No reserved result exists.**

## 1. Streams

* **Reserved:** PCG(**20261201**, 100000·k + i), with k the scenario's
  index in §5 of the final protocol and i the seed.
* **Never used:** the value appears only in protocol text (`git grep`
  and `git log -S` at 478fcbb: `protocol.md`, `correction-protocol.md`
  and `fidelity-protocol.md`). No code has executed it, and no result
  on it has been inspected.
* **Fresh streams are not needed.** If the final protocol changes the
  scenario list after this draft, k is re-indexed in the committed
  final version before any run.

## 2. Model

The fidelity model (`fidelity-protocol.md`, `ocFidelity`), unchanged
in its parameters, plus the changes of §3.

* **No simulator-level validity is claimed.** At the simulator's load
  its tails are heavier than the simulator's (findings-fidelity §8).
* The results are the operating characteristics of this synthetic
  practice.

## 3. Model changes required before the run (no tuning)

* **Evaluator weeks aligned with the synthetic calendar.**
  * The synthetic takeover instant (1e9 s) falls in an evaluator week
    that starts on Wednesday at about 04:00. That shifts the weekly
    concentration windows.
  * The takeover instant becomes the evaluator's own Tuesday 06:00, so
    Monday weeks are calendar weeks.
  * A test pins the weekday.
* **If the owner chooses access option B:** the evaluator's two access
  gates, and the synthetic booking's loss.
  * The synthetic booking loses almost no urgent orders, so the loss
    gate would be exercised only by the access-deterioration witnesses
    (§5). That is stated as a limitation: the model is not the
    simulator's booking.

## 4. Candidate grid

**Fixed (the owner's provisional values):**

| Rule | Values judged |
|---|---|
| Urgent absolute maximum | 24 h past due (also reported: 36, 48 h) |
| Soon absolute maximum | 48 h past due (also reported: 72 h) |
| Routine absolute maximum | 14 days past due (also reported: 7, 10 days) |
| Severe delay | more than 24 h past due |
| Severe rate | 5% per priority (also reported: 2, 10%) |
| Severe a week | the owner's decision (§3 of acceptance-design); also reported: 5, 10, 20, 40, and proportional to weekly volume at 2, 5, 10% |
| Shared allowance | 4 of 40 |
| Zero tolerance | the absolute maxima, every seed |

**Access** (calendar clock, every eligible order). **PLACEHOLDER**,
replaced by the owner's design:

* urgent 48 h at 90% and at 95%; 24 h reported;
* soon 7 days at 90%;
* routine 30 days at 90%;
* the maximum waits.

**The bundle judged** is the candidate policy (`candidate-policy-v5.json`
as finally amended) as a whole, with v0.2-eval-4's service and backlog
gates.

## 5. Scenarios

All with the §3 changes, 2,500 seeds per stable and stress scenario
and 1,000 per other.

| Group | Scenarios |
|---|---|
| Stable (0.85 core load) | A85, B85, C85, Cs85 |
| Weekend gap | Wg85 (equal staffing), WgU85 (equal realized utilization), R085 |
| Stress (0.92) | A92, B92, C92, Cs92, Wg92, WgU92 |
| Reading deterioration | L10-85, L20-85, D-85, K-85, L10-92, K-92 |
| Imaging deterioration | IL10-85, IK-85, IL10-92, IK-92 |
| Severe reporting-delay witnesses | A85 with, from day 60, six urgent reports a week eight days late; one routine report 348 h past due |
| Access deterioration witnesses | A85 with thirty urgent orders a seed lost or delayed thirty days |
| Validation | the mirror M (reported, not judged) |

## 6. Pass and fail criteria (judged once, on the reserved streams)

* **T1 (false rejection).** For every stable scenario, the whole
  bundle's 40-seed rejection must be at most 0.05. The simultaneous
  95% upper bound is reported; the point estimate is judged.
* **T2 (detection).** For every reading deterioration whose clinical
  consequence is material (the reserved run's own consequences: severe
  soon or routine share at least 1%, or worst soon delay at least 48 h
  at p50), the 40-seed rejection must be at least 0.90. A deterioration
  without material consequence is reported, not judged.
* **T3 (zero-tolerance admissibility).** On every stable scenario, each
  absolute maximum's one-sided 95% upper bound of per-seed violation
  must be at most 0.13% (no violating seed in 2,500).
* **T4 (witnesses).** Every severe-delay and access witness is rejected
  in every 40-seed group.
* **Stress, weekend-gap and the mirror:** reported, never judged.

**A target missed is the owner's decision.** No value is adjusted
after the reserved results are seen. If one is, the reserved streams
count as used, and a new protocol and new streams are needed.

## 7. Monte Carlo uncertainty

* Clopper–Pearson 95% intervals for every fraction.
* Simultaneous intervals for the combined 40-seed rejection.
* Disjoint groups of 40, judged whole.

As in `correction-protocol.md` §6.

## 8. Model-validity limitations (carried into the result)

* **Heavier tails** than the simulator at its own load. The model is
  not validated at 0.85 or 0.92.
* **The weekday priority mix** is not reproduced.
* **Simplified booking with reserves.** Access results are the model's,
  not the simulator's, whose urgent access is loss-dominated.
* **Rare-event probabilities** (zero-tolerance bounds) are model
  quantities.
