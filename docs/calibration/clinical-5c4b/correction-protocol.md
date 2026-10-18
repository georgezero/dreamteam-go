# Slice 5C.4b: calibration correction (protocol)

Written and committed **before any corrected result was computed**. It
corrects the synthetic practice of `protocol.md` and fixes the
scenarios, streams and reporting of a second exploratory run. The
first run's results stay as generated (`findings.md`, with its erratum
in §7).

**Owner instructions behind it** (2026-10-18):

* Do not select clinical thresholds, and do not create the v0.2-eval-5
  policy yet.
* Correct the calibration scenario, not the evaluator. Make utilization
  a measured property of the practice.
* Keep two regimes: a stable practice at about 0.85 and a stress case
  at about 0.92.
* Separate weekend coverage from clinical tolerance. Compare the
  E3-like practice with and without weekend reading.
* Do not require a capacity loss to be rejected merely because
  capacity fell. Report its clinical consequence, and keep capacity
  deterioration apart from clinical harm.
* Validate the full bundle, not only single gates.
* Verify the model against the simulator, and document what still
  differs.

**What it uses.** No strategy, development, reference or holdout run.
Synthetic practices only. The evaluator, the v1–v4 policies, the
simulator's economics and demand are not changed.

## 1. What was wrong

From the erratum (`findings.md` §7), measured on the first run's own
streams:

* The stable practice read at about **0.67** of its general reading
  capacity over the gated period and **0.74** over its last 30 days,
  not 0.85.
* Its rate was sized for 66.5 reading hours a week, but it reads 70.5.
* The routine cancellations (4.3% of orders) were not counted.
* It started empty, so its acquisitions built up over months.

## 2. The corrected practice

The first run's model (`protocol.md` §2) with three changes. Each is a
field of `ocPractice` whose zero value is the original model; a test
run of the first protocol still reproduces its output byte for byte.

* **Utilization is defined and measured.**
  * **Effective reading utilization:** studies acquired over the
    general reading capacity offered, in studies, over the same
    period. General capacity is the team's rate in every hour open to
    all priorities (urgent-only hours excluded), times the scenario's
    capacity factor.
  * It is reported for each seed over the gated period (takeover to
    horizon) and its last 30 days. Imaging slot utilization (slots
    booked, cancelled bookings included, over slots offered) is
    reported too.
* **The rate is set from the target, over the practice's own hours.**
  * Rate = expected weekly acquisitions / (general hours a week ×
    target).
  * Expected weekly acquisitions are 52 × 7 orders less the routine
    cancellations (86% × 5%): 348.3 at season-1 demand. Orders lost at
    imaging are negligible at imaging utilization 0.80 (measured: 0).
  * General hours a week are 70.5 for the stable hours and 66.5 for
    Wg and R0.
  * This sets the rate from the target utilization only. It is not
    adjusted to any clinical result.
* **Steady state at the takeover.**
  * The practice runs for 181 days (the longest routine lead) before
    the takeover, as the simulator's world has its warm-up.
  * The record inherits the orders still open at the takeover (booked
    appointments) and the studies unread then.
  * The evaluator times the inherited work from the takeover
    (`inheritedFrom: takeover`).
* **The simulator's worklist** (`internal/sim/reading.go`, its default
  `urgent_then_fifo`).
  * The first group is urgent studies plus any study past its target.
    The second group is the rest. Each group is read in entry order.
  * Soon is therefore not read ahead of routine before its 4-hour
    target.
  * An urgent-only reader reads studies of urgent priority only, never
    promoted ones, as the simulator's urgent-only teleradiology does.
  * The first model's strict urgent > soon > routine order is kept as
    the variant P85.

**Calibration check** on the debug stream PCG(1, ·), never used for
results, two seeds each. The scenarios share the stream, so their
seed noise is common.

* Measured utilization: A85 0.835, B85 0.849, C85 0.846, Cs85 0.861,
  Wg85 0.833; A92 0.903, B92 0.919, C92 0.915, Cs92 0.932.
* Acquisitions were 49.1–50.7 a day against the 49.8 expected, with no
  order lost. Every record was valid.

**Model development, before this protocol was committed.**

* The first implementation of the warm-up booked nothing before the
  takeover. The original code's "no slot" sentinel was day −1, which a
  warm-up day can be. It was found on the debug stream, by a test that
  requires an inherited backlog, and fixed with an explicit flag. The
  first model is unchanged by the fix: its days are never negative.
* No result on the correction stream had been seen.

## 3. Operational equivalence with the simulator

The comparison is by reading the code, `internal/sim` and
`scenarios/base/established.json`; no simulation was run.

**Matched:**

* The worklist order (§2).
* The imaging hours: Mon–Fri 08:00–20:00, Sat 10:00–18:00, closed on
  Sunday.
* The weekday demand profile: Monday 1.1, Saturday 0.25, Sunday 0.
* The turnaround targets: urgent 60, soon 240, routine 1,440 minutes,
  from acquisition.
* A warm-up before the takeover, with inherited work.
* Turnaround starts at acquisition.

**Not matched** (ranked by likely effect on the delay tails):

1. **Readers.** The simulator has two individual radiologists, each
   reading one study at a time. Reads are random and depend on the
   exam: XR 1.5–5 minutes, CT and MR up to 15–25.
   * They read in 100-minute cycles, 60% and 50% of their time.
   * A read can overrun its block, and the overrun delays the next
     one.
   * Near the end of a shift, long reads are skipped.
   * The synthetic practice has one fluid rate, with no read time.
   * So the simulator's capacity is burstier, and its tails are
     probably heavier.
2. **Reading hours.** By default the simulator reads Mon–Fri
   07:30–20:00 (two staggered shifts) and Sat 10:00–14:00. It has no
   evening reader and no Sunday reading. That is the R0 coverage.
   Strategies change it: E3's urgent weeknight reader is Wg's.
3. **Modalities.** The simulator has four, with very different read
   times. The synthetic practice has one pool.
4. **Worklist entry.** In the simulator a study enters 1–12 minutes
   after acquisition, once documented. In the synthetic practice it
   enters on acquisition.
5. **Booking.**
   * The simulator has slots per room and category, with first fit,
     time windows and 60 minutes' notice for same-day slots. It has no
     routine cap and no urgent or soon reserve.
   * Patient tolerances are drawn per patient.
   * Cancellations are 9% of every priority, and half rebook.
   * No-shows are 4–14%, with partial rebooking, and walk-ins may
     leave.
   * The synthetic practice books day slots with a routine cap of 85%,
     its patients take the longest tolerance, and 5% of routine orders
     cancel.
   * So access, the access gates in particular, is not modelled at the
     simulator's level.
6. **Mix and volume.** The simulator has about 63 referrals and 9.5
   walk-ins a day, with roughly 5% urgent and 15% soon (an estimate
   from its referral sources). The synthetic practice has 52 orders a
   day, 2% urgent and 12% soon.

**Conclusion, fixed before the results.** The corrected practice
matches the simulator's worklist rule, imaging hours and steady state.
It does not match the simulator's readers, read times, modalities or
booking. **No simulator-level validity is claimed.** The results are
operating characteristics of this synthetic practice.

## 4. Scenarios

Two regimes. Each has its stable practices and departures, all with
the corrected model (target utilization u, warm-up, simulator
worklist).

| Regime 0.85 | Regime 0.92 | Departure | Role |
|---|---|---|---|
| A85 | A92 | none, stationary | stable (0.85) / stress (0.92) |
| B85 | B92 | seasonal | stable / stress |
| C85 | C92 | bursty, day CV 0.15 | stable / stress |
| Cs85 | Cs92 | seasonal and bursty | stable / stress |
| Wg85 | Wg92 | no reading Saturday after 14:00 or Sunday, at the same utilization (the weekend's capacity on weekdays) | structural: E3-like without weekend reading |
| WgS85 | WgS92 | the same hours at A's team rate (the weekend's capacity removed: utilization 0.90 / 0.98) | structural |
| R085 | | the scenario's own rota (no evening reader), at the same utilization | structural: the simulator's default coverage |
| P85 | | A85 read urgent, then soon, then routine (the first model's worklist) | model variant |
| L10-85 | L10-92 | reading 10% lower from day 60 | deterioration |
| L20-85 | | reading 20% lower from day 60 | deterioration |
| D-85 | | reading 25% lower from day 30 | deterioration |
| K-85 | K-92 | reading eroding 0.2% a day from day 30 | deterioration (gradual) |
| IL10-85 | IL10-92 | imaging 10% lower from day 60 | deterioration |
| IK-85 | IK-92 | imaging eroding 0.2% a day from day 30 | deterioration (gradual) |
| I92-85 | | imaging utilization 0.92 | stress |

* **Seeds.** The stable and stress practices A, B, C and Cs use 2,500
  seeds in both regimes, enough for the 0.13% zero-tolerance bound.
  Every other scenario uses 1,000.
* **The E3-like comparison** is A against Wg and WgS in each regime:
  A85 is the stable practice with weekend reading, Wg85 the same
  capacity without it.
* **Deterioration is not presumed harmful.**
  * For each deterioration scenario, the run reports its clinical
    consequence beside its detection: measured utilization, severe
    delays by priority, worst past due, timely access, lost orders and
    the unread backlog.
  * A scenario whose consequence is small may pass and still be
    acceptable. The owner judges which consequences are clinically
    harmful.
* **Stress and 0.92.** The owner placed 0.92 out of calibration
  (2026-10-18). Its practices are reported in full, so that limits are
  not calibrated only to the easier case. They are never a
  false-rejection target.

## 5. Streams

* **The corrected run:** seed i of scenario k (its row in §4, in the
  order of `ocCorrectedScenarios`) draws from PCG(20261102,
  100000·k + i).
* **Never used before:** no file or commit contains it before this
  protocol.
* **The confirmatory streams PCG(20261201, ·) stay reserved.**
* **The original practices' utilization** (the erratum): the first 200
  seeds of each one's own exploratory stream, PCG(20261101, ·).

## 6. What is reported

For every scenario:

* **Per gate:** the grids and values of `protocol.md` §5, as the
  share of seeds failing, with a Clopper–Pearson 95% interval and the
  one-sided 95% upper bound.
* **Combined:** the bundles of `protocol.md` §5 (lenient, moderate,
  strict), unchanged. Each also applies v4's breach gates and backlog
  rule.
  * Per-seed failing and violating counts.
  * The seeds failing each component: v4 service, v4 backlog, severe
    rate, severe week, timely access, maximum wait, absolute. Also the
    seeds for which that component is the only failure.
* **40-seed rejection:**
  * shared allowance only, absolute limits only, and both together;
  * disjoint groups of 40 judged whole, pooled rates included.
* **Monte Carlo uncertainty:**
  * Clopper–Pearson 95% intervals for every fraction, mapped through
    the rejection formulas;
  * a simultaneous 95% interval for the combined rejection, from the
    97.5% bounds of the failing and violating fractions (the rejection
    increases in both);
  * a Clopper–Pearson interval for the disjoint groups.
* **Consequences:** as in §4.
* **Targets.** The targets of `protocol.md` §6 are reported against
  the 0.85 stable practices. Nothing is selected, and no target is
  judged here: this is still the exploratory stage.

## 7. Reproduction

    DT_OC_CORRECTED=1 go test ./internal/eval -run 'TestClinicalOCCorrected$' -v -timeout 600m

The output is committed as
`docs/evaluation/oc-v0.2-eval-5-exploratory-corrected.json`.

* **A subset:** `DT_OC_SCENARIOS=A85,Wg85 DT_OC_OUT=<file>` reproduces
  only those scenarios, on the same streams, into a scratch file.
* **Tests.**
  * `TestClinicalOCCorrectedAgreesWithEvaluate` checks that the
    tabulation agrees exactly with the production `Evaluate`.
  * `TestOCCorrectedCalibration` checks each corrected practice's
    utilization on the debug stream.
  * `TestSimPick` checks the worklist rule against the simulator's.
