# Slice 5C.4b: exploratory findings

From `docs/evaluation/oc-v0.2-eval-5-exploratory.json`, the exploratory
stage of `protocol.md`. It ran on joofi from a compiled binary of
cb3bb31, with the protocol committed before it.

* **Nothing is selected here.** The values are the owner's to choose.
* **The confirmatory streams are untouched.**
* **No strategy, development, reference or holdout run was used.**

> **Erratum (recorded after the run; §7).** The stable practice was
> meant to read at utilization 0.85. It read at about 0.67 over the
> gated period and 0.74 over its last 30 days. These results are
> therefore **not calibrated at 0.85**; they are kept as generated. The
> corrected calibration is `correction-protocol.md`.

## 1. Stable practices: no false rejection at any bundle

On A, B, C and Cs (2,500 seeds each):

* **Bundles.** No seed of any of them fails the lenient, moderate or
  strict bundle, except one C seed under strict (urgent timely access
  within a day at 0.95).
* **Every gate value** of the grid fails no seed, except an urgent
  absolute limit of 12 hours (below).
* **Admissibility (T3).** With 0 of 2,500, every absolute limit of 24
  hours or more (urgent) and 24 hours or more (soon) meets the 0.13%
  admissibility bound.
* **Forty-seed rejection.** It is 0 for every bundle, on the formula
  and on all 62 disjoint groups of 40, pooled rates included.

## 2. The weekend question: a clean cut

| Practice | Urgent worst past due per seed, p50 / max | Soon worst, p50 / max |
|---|---|---|
| Stable (reading covers imaging, Sunday morning reader) | 13.0 h / 14.0 h | 10.8 h / 11.9 h |
| Wg: no reading Saturday after 14:00 or Sunday | 39.0 h / 40.8 h | 37.5 h / 38.5 h |
| R0: the scenario's own rota | 39.1 h / 40.8 h | 37.5 h / 39.0 h |

Share of seeds violating each urgent absolute limit:

| Urgent limit | Stable (A) | Wg | R0 |
|---|---|---|---|
| 12 h | 69% | 98.5% | 100% |
| 24 h | 0 | 98.5% | 99.3% |
| 36 h | 0 | 97.3% | 98.2% |
| 48 h and above | 0 | 0 | 0 |

* **Below about 41 hours, the limit decides the weekend.** An urgent
  absolute limit there rejects, at zero tolerance, essentially every
  practice that does not read on weekend afternoons. An E-family
  strategy with an urgent weeknight reader only is such a practice. At
  48 hours or more, the limit does not bear on the weekend at all.
* **Soon behaves the same way.** A 24-hour soon limit rejects every
  practice without weekend reading; 48 hours rejects none.
* **Wg and R0 already fail elsewhere.** Both fail v0.2-eval-4's service
  gates on every seed, mainly routine: Saturday imaging is read on
  Monday. That is independent of this choice. It may overstate what a
  real practice without weekend reading does (limitation §5).

## 3. The severity gates see the weekend; the access gates see little

* **Severe-delay rate and weekly concentration.** Soon is the clearest
  case:
  * a 5% severe-rate limit fails 99% of Wg and R0 seeds;
  * at most 5 severe soon delays a week fails 82–84%;
  * none of them fails a stable seed.
* **Routine severity detects sustained reading loss** (D, 25% from day
  30). A 1% routine severe-rate limit fails 25% of D's seeds, and 5%
  fails 8%.
* **Access gates.**
  * Timely access detects only imaging erosion (IK), and only with
    strict values: urgent within a day at 0.95 fails 20% of seeds; at
    0.90, 3%.
  * Imaging 10% lower (IL10) and imaging at utilization 0.92 (I92) are
    essentially invisible to every access value in the grid.
  * In this model, routine booking leads are long and the urgent and
    soon reserve holds, so access absorbs these losses.

## 4. Detection of the harmful scenarios (T2c, reported)

| Scenario | Lenient / moderate / strict, 40-seed rejection | What fails |
|---|---|---|
| D, reading 25% lower | 1.00 / 1.00 / 1.00 | v4 service (56%), backlog (31%), routine severity |
| K, reading erosion | 1.00 / 1.00 / 1.00 | v4 backlog (70%) |
| IK, imaging erosion | 0 / 0 / 0.93 | strict urgent timely access only |
| L10, reading 10% lower | 0 / 0 / 0 | nothing |
| IL10, imaging 10% lower | 0 / 0 / 0 | nothing (3 seeds under strict) |

* **L10 and IL10 cause no clinical harm in this model.** No breach
  rate, severe delay, access fraction or backlog check moves
  materially.
* **Why.** With the stable practice's weekend reading hours, its
  effective reading utilization is about 0.80, not the 5C.3 queue's
  0.85. A 10% loss leaves it near 0.89, still a stable queue.
* **What this does not mean.** It does not show that the service and
  access gates compensate for the backlog gate's blind spot. It shows
  that, in this model, the 10% step-downs are not clinically harmful,
  so nothing can be detected. Whether they would be in the real
  simulator is not answered here.

## 5. Limitations

* **The stable practice is benign.**
  * Its worst urgent delay varies by under an hour across 2,500 seeds,
    so the synthetic queue has thin tails.
  * Zero tolerance passes T3 easily here. The real simulator, with
    individual readers, breaks and stochastic read times, is likely to
    have fatter tails.
  * T3's bound holds for this model only.
* **The harmful scenarios are mild at this utilization.** For the
  confirmatory stage the owner may want harmful scenarios defined by
  clinical consequence, for example a reading or imaging loss
  calibrated to push a stable practice past its gates, rather than by a
  fixed percentage.
* **Saturday routine imaging.** It follows the wanted day uniformly,
  which may overstate the routine breach rate of practices without
  weekend reading (Wg, R0).
* **Model scope.** One modality pool, day-level booking and constant
  finance.

## 6. For the owner's choice

| Decision | What the exploratory run shows |
|---|---|
| Urgent absolute limit | ≤ 36 h: a closed weekend afternoon is a safety violation (Wg and R0 fail at 97–99%). ≥ 48 h: it is not. Stable practices pass anything from 24 h. 12 h fails stable practices. |
| Soon absolute limit | 24 h: the same weekend cut. 48 h or more: none. |
| Severe rate and weekly count | Soon at 5% and 5 a week separate weekend-gap practices; routine at 1–2% sees sustained reading loss. None costs stable practices anything here. |
| Access limits | No value tested separates harm from stability except strict urgent (1 day, 0.95), and only for imaging erosion. Values may be set on clinical meaning without a false-rejection cost in this model. |
| Shared allowance | Its cost (5C.4a §C.3) does not appear here: stable seeds fail no component. |
| Confirmatory stage | Consider re-specifying the harmful scenarios by clinical consequence before it runs. |

## 7. Erratum: the utilization was not 0.85

Recorded after the run, when the owner asked for utilization to be a
measured property of the synthetic practice. The results above are
unchanged; only their interpretation is corrected.

* **What was intended.** Reading utilization 0.85 for the stable
  practice (0.92 for B92), as in the 5C.3 queue.
* **Three causes.**
  * **The rate was set over the wrong hours.** It was sized for 66.5
    reading hours a week, the 5C.3 rota. The stable practice reads
    70.5 general hours: Saturday afternoon and Sunday morning were
    added. So 0.85 × 66.5 / 70.5 = 0.80 of its capacity. This is the
    "about 0.80" of §4.
  * **The cancellations were not counted.** 5% of routine orders
    (4.3% of all) are cancelled before imaging, so the workload is
    lower still: 0.80 × 0.957 ≈ 0.77 in the steady state.
  * **The practice started empty.** It had no booked appointments at
    the takeover. Routine orders are booked up to 181 days ahead, so
    acquisitions build up over the first months. The 5C.3 queue's
    arrivals were acquisitions, so it had no such ramp.
* **Measured** (acquisitions over the general reading capacity offered,
  first 200 seeds of each scenario's own exploratory stream; the
  corrected output's `originalPracticesUtilization`):

  | Scenario | Intended | Takeover to horizon (mean) | Last 30 days (mean) |
  |---|---|---|---|
  | A, B, C, Cs | 0.85 | 0.67 | 0.74 |
  | Wg, R0 | 0.85 | 0.71 | 0.78 |
  | B92 | 0.92 | 0.73 | 0.81 |
  | L10 (after the loss) | | 0.71 | 0.82 |
  | K | | 0.74 | 0.94 |
  | D | | 0.84 | 0.99 |

* **What it changes.**
  * The false-rejection results of §1 are for a practice with
    considerable spare capacity, so they are optimistic.
  * The non-detection of L10 and IL10 (§4) is for that practice, and
    so is the absence of harm.
  * B92 is not a 0.92 stress case.
  * The weekend cut of §2 is structural. It depends on the reading
    hours, not on utilization, and is re-measured in the correction.
