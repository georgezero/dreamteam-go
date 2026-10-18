# Slice 5C.4b: corrected exploratory findings

From `docs/evaluation/oc-v0.2-eval-5-exploratory-corrected.json`, the run
fixed by `correction-protocol.md` (committed before it, d11964f).

* **How it ran.** On joofi from a compiled binary of d11964f, streams
  PCG(20261102, ·). The Pi reproduced A85, Wg85 and L10-92 identically.
  The original run still reproduces byte for byte from the new code.
* **Nothing is selected.** The values are the owner's to choose. The
  confirmatory streams are untouched.
* **No strategy, development, reference or holdout run was used.**
* **Synthetic model only.** No simulator-level validity is claimed
  (§7).

Notation: "fails" is the share of seeds failing; R40 is the 40-seed
rejection. Delays are past the turnaround target, in hours.

## 1. The utilization is now what was intended

Measured per seed, as acquisitions over the general reading capacity
offered, from the takeover to the horizon. Mean, with the p05–p95 range
across seeds:

| Practice | Target | Measured | Last 30 days |
|---|---|---|---|
| A85, B85, C85, Cs85 | 0.85 | 0.845–0.849 (0.82–0.87) | 0.84–0.85 |
| Wg85, R085 (same capacity, other hours) | 0.85 | 0.847–0.848 | 0.84 |
| WgS85 (A85's rate, weekend hours removed) | (0.90) | 0.899 | 0.89 |
| A92, B92, C92, Cs92 | 0.92 | 0.914–0.919 (0.89–0.94) | 0.91–0.92 |
| WgS92 | (0.98) | 0.973 | 0.96 |

* **Compared with the first run.** Its stable practice measured 0.67
  (0.74 over the last 30 days), and its "0.92" practice 0.73.
* **Records.** Every one of the 37,000 is valid, with work inherited at
  the takeover. Imaging slot utilization is 0.80, and no order is lost.

## 2. Stable practice at 0.85: false rejection

**Per gate.** No severe-rate, weekly, access or maximum-wait value of
the grid fails a single seed of A85, B85, C85 or Cs85 (2,500 each).

**Absolute limits (T3: the one-sided 95% bound must be at most 0.13%).**

| Limit | A85–Cs85 violating | T3 |
|---|---|---|
| urgent 12 h | 72–76% | inadmissible |
| urgent 24 h to 120 h | 0 of 2,500 each | admissible (bound 0.12%) |
| soon 24 h | 1 of 2,500 (Cs85; 34.9 h) | **inadmissible** (bound 0.19%) |
| soon 48 h to 168 h | 0 | admissible |
| routine 7 to 14 days | 0 | admissible |

**Whole bundles.** Every failing stable seed fails v4's own service or
backlog gate; no v5 clinical component fails any.

| | Seeds failing | R40, point [simultaneous 95%] | Groups of 40 rejected |
|---|---|---|---|
| A85 | 1.0% | 0.000 [0.000, 0.068] | 0 / 62 |
| B85 | 0.8% | 0.000 [0.000, 0.068] | 0 / 62 |
| C85 | 1.2% | 0.000 [0.000, 0.068] | 0 / 62 |
| Cs85 | 2.4% | 0.002 [0.001, 0.075] | 1 / 62 |

* The three bundles give the same figures here.
* **T1c (at most 0.05) is met at the point.** The upper bounds exceed
  0.05 only because a zero violation count in 2,500 seeds cannot bound
  the zero-tolerance term more tightly (1 − (1 − 0.0015)^40 ≈ 0.057).

## 3. The 0.92 stress regime

**v4 already rejects it.**

* 85–89% of A92, B92, C92 and Cs92 seeds fail v4 service, so R40 is
  1.00 under every bundle.
* The failing gate is **v4's soon breach rate.** That is the share of
  soon studies past their 4-hour target, limit 0.32.
* It runs at 0.335 (40-seed diagnostic of A92), against 0.246 at 0.85.

**Clinical harm is small.**

* Severe-delay shares are 0.00–0.08% by priority.
* The worst past-due p50 is 13.7 h (urgent) and 14.2 h (soon).

**v5 components at 0.92** (2,500 seeds each):

* urgent 24 h or 36 h absolute: 0.1–1.4% violating, inadmissible
  by T3;
* urgent 48 h: 0–0.3%;
* urgent 72 h: none;
* soon 24 h: 1.5–9.6%;
* soon 48 h: 0.1–1.7%;
* soon 72 h: none;
* soon weekly count at 3: up to 4.8%;
* routine severe rate at 1%: up to 1.2%.

**So the regimes split.** At 0.85, v4 and v5 accept. At 0.92, v4's
soon turnaround gate rejects, although severe delays stay rare.

## 4. Weekend coverage, separated from tolerance

All three compared practices are at the **same effective utilization**,
0.85. They differ only in when capacity is offered.

| Practice | Urgent worst, p50 / max | Soon worst, p50 / max | Severe share, urgent / soon | v4 service fails |
|---|---|---|---|---|
| A85, weekend reading | 13.7 / 15.9 h | 13.7 / 14.9 h | 0 / 0 | 0.5% |
| Wg85, E3-like: urgent weeknight reader, no reading Sat after 14:00 or Sun | 41.7 / 44.0 h | 39.6 / 41.5 h | 3.0% / 7.4% | 100% |
| R085, the simulator's default rota | 41.8 / 43.6 h | 39.6 / 41.6 h | 3.1% / 7.4% | 100% |
| WgS85, Wg at A85's rate (utilization 0.90) | 42.1 / 47.0 h | 40.1 / 44.6 h | 3.0% / 7.5% | 100% |
| Wg92 | 42.3 / 49.7 h | 40.4 / 62.2 h | 3.1% / 7.7% | 100% |
| WgS92 (0.97) | 44.1 / 53.5 h | 44.3 / 83.9 h | 3.0% / 9.1% | 100% |

Share of seeds violating each urgent absolute limit:

| Urgent limit | A85 | Wg85 | WgS85 | Wg92 | WgS92 |
|---|---|---|---|---|---|
| 24 h or 36 h | 0 | 99.2% | 99.5% | 99.2% | 99.5% |
| 48 h | 0 | 0 | 0 | 0.2% | 12.3% |
| 72 h | 0 | 0 | 0 | 0 | 0 |

* **The weekend delay is structural.** Removing weekend reading costs
  about 28 h of worst urgent delay at equal capacity. Utilization adds
  only a few hours on top.
* **The cut is between 44 and 48 h at 0.85.** It moves above 48 h once
  the practice is loaded (Wg92 0.2%, WgS92 12%).
  * An urgent limit of 36 h or less makes the absence of weekend
    afternoon reading itself a safety violation.
  * 48 h is borderline under load.
  * 72 h never binds.
* **Soon:** at 24 h every weekend-gap practice violates; at 48 h, Wg85
  0, Wg92 0.5%, WgS92 20%.
* **Without weekend reading the practice already fails v4.** Wg, R0 and
  WgS fail v4 service on every seed, mostly routine (8% breach against
  the 0.06 limit) and soon (0.33 against 0.32).
  * That is independent of any v5 value, but it contradicts the
    simulator, where E3 passes v4.
  * The likely cause is the synthetic practice's Saturday: 12.8% of its
    acquisitions fall on Saturday, against about 5% of the simulator's
    XR slots and demand (§7). The weekend's service effects are
    overstated here; the delay structure (Saturday afternoon to Monday
    morning) is not.
* **E3.** These results do not show what E3 itself would do. They show
  the following:
  * if the owner sets an urgent limit of 36 h or less, a practice that
    does not read on Saturday afternoon or Sunday is ineligible
    whatever its utilization;
  * at 48 h or more it is not, at 0.85.
  * The choice should follow from the clinical tolerance for an
    urgent study acquired on Saturday afternoon, not from E3.

## 5. Deterioration, its consequence, and the full bundle

Each scenario's clinical consequence, beside its detection:

| Scenario | Utilization, last 30 d | Severe share, urgent / soon / routine | Soon worst, p50 | Unread at horizon, p50 | Seeds failing, lenient / moderate / strict | R40, every bundle | What fails |
|---|---|---|---|---|---|---|---|
| A85 (baseline) | 0.84 | 0 / 0 / 0 | 13.7 h | 6 | 1.0% | 0.00 | v4 service, backlog |
| L10-85: reading 10% lower | 0.94 | 0.01 / 0.03 / 0.01% | 14.1 h | 8 | 94% | 1.00 | v4 soon breach rate (88% sole); severe week ≤ 1.5% |
| L20-85: 20% lower | 1.05 | 2.0 / 39 / 26% | 106 h | 155 | 100% | 1.00 | everything |
| D-85: 25% lower from day 30 | 1.12 | 3.4 / 72 / 68% | 305 h | 546 | 100% | 1.00 | everything |
| K-85: eroding 0.2%/day | 1.07 | 0.6 / 10 / 4.8% | 83 h | 109 | 100% | 1.00 | v4 backlog and service, severe rate and week |
| IL10-85: imaging 10% lower | 0.85 | 0 / 0 / 0 | 13.5 h | 6 | 0.2 / 0.2 / 1.2% | 0.00 | strict access, rarely |
| IK-85: imaging eroding | 0.82 | 0 / 0 / 0 | 13.5 h | 4 | 0 / 0.4 / 39% | 0 / 0 / 1.00 | strict urgent timely access only |
| L10-92: reading 10% lower at 0.92 | 1.01 | 1.0 / 14 / 5.3% | 62 h | 58 | 100% | 1.00 | backlog (89%), severe week and rate, v4 service |
| K-92 | 1.15 | 1.5 / 33 / 26% | 176 h | 292 | 100% | 1.00 | everything |
| IL10-92 | 0.92 | 0 / 0 / 0 | 14.1 h | 8 | 46% | 1.00 | v4 soon breach rate (less than A92's 85%) |
| IK-92 | 0.89 | 0 / 0 / 0 | 14.0 h | 5 | 42 / 42 / 66% | 1.00 | v4 service; strict access |

The 40-seed rejection intervals are degenerate at 0 or 1, except
I92-85 under strict: 0.67 [0.50, 0.85].

* **Capacity deterioration is not clinical harm.**
  * L10 at 0.85 takes the practice to 0.94 in its last month. Severe
    delays stay below 0.05% and the backlog is 8 studies. Clinically
    it is barely distinguishable from A85.
  * It is rejected, all the same, because v4's 4-hour soon turnaround
    gate fails: 0.34 against 0.32.
  * That is an operational detection, not a clinical one. Of the v5
    clinical components, only the weekly count fails seeds under
    lenient and moderate (0.3% and 0.5%). Under strict, the absolute
    limits fail 1.8% and the weekly count 1.5%.
* **Clinically harmful losses are detected by the clinical components
  themselves.**
  * L20, D, K and L10-92 are all caught: severe soon or routine delays
    of 5–72%, worst soon delays of 60–300 h, backlogs of 58–546.
  * Severe rate, severe week, backlog and the absolute limits all fire,
    under every bundle.
* **Imaging losses are absorbed.**
  * IL10 and IK leave delays and access essentially unchanged: urgent
    timely access 0.99, nothing lost.
  * They are detected only by strict urgent access (IK) or by v4's soon
    gate at 0.92.
  * On this evidence neither is harmful in this model. Its booking is
    much simpler than the simulator's (§7).

## 6. For the owner's choice

| Decision | What the corrected run shows |
|---|---|
| Urgent absolute limit | 24 h or more is admissible for the 0.85 stable practices; 12 h is not. ≤ 36 h makes no weekend afternoon reading a violation (99%). 48 h is clear at 0.85 and borderline under load. 72 h never binds. At 0.92, 24–36 h violate on up to 1.4% of seeds. |
| Soon absolute limit | 24 h is **not** admissible at 0.85 (one Cs85 seed at 34.9 h); 48 h or more is. 48 h violates on up to 1.7% at 0.92. |
| Severe rate, weekly count | No grid value costs a 0.85 stable seed anything. They separate the harmful reading losses at every bundle. Soon at 5% or 5 a week also separates the weekend-gap practices. |
| Access | Costs nothing at 0.85 and detects only imaging erosion under strict values (urgent within 1 day at 0.95: IK 39%, I92 14%). |
| Combined bundle | At 0.85: R40 ≤ 0.002 for every bundle; every rejection there is v4's own. Harmful reading losses: R40 1.00 under every bundle, from the clinical components. |
| 0.92 regime | Rejected by v4's soon turnaround gate, not by severe delays. Whether that is wanted is a separate question from the v5 values. |

## 7. Remaining model limitations

Fixed in `correction-protocol.md` §3 before the run. Confirmed or
quantified by it:

* **Readers.** The synthetic practice reads at one fluid rate. The
  simulator has two radiologists in reading blocks, with random,
  exam-dependent read times, overruns and skipping at the end of a
  shift. Its tails are probably heavier. In particular, the 0.85
  stable practice's very thin tails (urgent worst 13.7–16 h) are
  likely optimistic, and so is the admissibility of zero tolerance.
* **Saturday load.**
  * 12.8% of the synthetic practice's acquisitions fall on Saturday:
    its routine bookings fill Saturday's slots, two thirds of a
    weekday's.
  * The simulator's Saturday demand factor is about 5% of the week, and
    its XR room has 12 Saturday slots against 48 on weekdays.
  * The weekend's service and severity effects are therefore overstated
    here. That is probably why the weekend-gap practices fail v4, which
    E3 does not in the simulator.
* **Worklist.** Now the simulator's urgent_then_fifo. The strict order
  (P85) gives soon breach 0.09 against 0.25. That is why v4's soon gate
  binds here: the result depends on the worklist rule a strategy uses.
* **Booking and access.** Day slots, a routine cap, the longest patient
  tolerance, and 5% routine cancellations only. The simulator has room
  slots, time windows, drawn tolerances, 9% cancellations of every
  priority, no-shows and walk-ins. The access gates' insensitivity
  here may not hold in the simulator.
* **Mix and modalities.** 2% urgent and 12% soon in one modality,
  against roughly 5% and 15% across four in the simulator.
* **No simulator-level validity is claimed.** Before the confirmatory
  run, the owner may want the Saturday load brought to the simulator's,
  or the readers made discrete. Either is a model change, and needs a
  new protocol and new streams.
