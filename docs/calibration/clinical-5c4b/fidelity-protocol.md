# Slice 5C.4b: simulator-fidelity calibration (protocol)

The final calibration correction before the owner chooses the
v0.2-eval-5 values. Written and committed **before any result on its
streams was computed**.

* **The earlier runs stay as generated:** `protocol.md` with
  `findings.md`, and `correction-protocol.md` with
  `findings-corrected.md`.
* **Owner instructions** (2026-10-18):
  * correct the synthetic Saturday share from a measured simulator
    distribution;
  * add individual readers with stochastic read times;
  * match the priority mix;
  * revalidate both regimes and the warm start;
  * compare the E3-like practice at equal realized utilization and at
    equal staffing;
  * rerun the exploratory characteristics on new streams;
  * compare the model with the simulator;
  * keep the v4 gates unchanged; do not freeze v5 and do not choose its
    values.
* **What is run.**
  * The simulator once: the evaluation scenario's own practice with
    **no action**, on the development seeds, to measure it (§1).
  * Synthetic practices otherwise.
  * No strategy, no reference or holdout seed, no unseal record. No
    simulator or evaluator code changes.

## 1. The simulator profile

`internal/sim/clinical_profile_test.go` (`DT_SIM_PROFILE=<file>`) writes
`internal/eval/testdata/sim-profile-5c4b.json`:

* **Scenario:** the development sets' manifest
  `docs/calibration/feasibility/step0c/experiments/W130-SHR-MRft-s20.json`,
  that is, `established` with its overlay.
* **Seeds:** the development seeds 901–910 (`eval.DevelopmentSeeds`),
  each built through the holdout guard (`NewEvalWorld`).
* **The run:** no action, recorded by the evaluation recorder exactly as
  a plan is, to the frozen policy's observation end.
* **Window:** acquisitions (orders) after the takeover and by the
  horizon: 150 days, the calendar of every synthetic record.

Measured, means over the ten seeds:

| | Simulator |
|---|---|
| Acquisitions | 51.7 a day (7,761 in the window) |
| Acquisitions by weekday, Mon–Sun | 11.9, 22.1, 20.9, 20.1, 19.3, **5.9**, 0% |
| Acquisition mix: urgent, soon, routine | **1.9, 11.6, 86.5%** |
| Order mix (receipt) | 4.4, 14.9, 80.7%: 65% of urgent and 32% of soon orders are lost before imaging |
| Acquisition mix by weekday | urgent 0.6–2.6%, soon 3.2% (Monday) to 15.6% (Friday) |
| Read time, p50 (p0–p100) | XR 170 s (90–299), US 351 (240–539), CT 702 (360–1,199), MR 976 (600–1,499); 477 s mean per acquisition |
| Readers | Avery Mon–Fri 07:30–16:00 at 60%, Blake 11:30–20:00 at 50%, one of them Saturday 10:00–14:00 (remote, alternate weeks): about 3,000 block minutes a week |
| Reading utilization (read time over block time) | 0.952 (0.947–0.961) |
| Breach share: urgent, soon, routine | 0.10, 0.36, 0.07 |
| Severe share (more than 24 h past due) | 1.4, 4.5, 1.9% |
| Worst past due: urgent, soon, routine | 36 h (13–45), 44 h (42–45), 41 h (40–43) |
| Urgent acquired Saturday after 14:00, worst past due | 33 h (0–45) |
| Oldest unread age at the daily samples, p90 | 42–57 h |
| At the takeover | 5 unread (1–21), 0.7 overdue, 1,216 open orders |

## 2. The fidelity model

`internal/eval/oc_fidelity_test.go`, the type `ocFidelity`. The earlier
model (`ocPractice`) is unchanged, so both earlier runs still reproduce.

* **Demand.**
  * Orders arrive at the simulator's daily rate for each weekday, by the
    scenario's receipt hours (07:00–19:00).
  * The rates are scaled so that the window's acquisitions are the
    simulator's 7,761, after cancellations.
  * The priority mix is the simulator's **acquisition** mix, the same
    every weekday.
  * Routine leads are the scenario's, soon wants 0–2 days ahead,
    urgent the same day; seasonal and bursty variants as before.
  * The simulator's priority mix by weekday is not reproduced, and the
    run reports the difference.
* **Imaging.**
  * Each weekday's slot capacity is its share of the simulator's
    acquisitions, times a fitted factor, at imaging utilization 0.80.
  * The factors were fitted on the debug stream, so that acquisitions by
    weekday match the simulator's: Mon–Sat 0.862, 0.857, 0.867, 0.898,
    1.528, 0.843 (`fidImgAdjust`).
  * Each day's slots sit at the quantiles of the simulator's acquisition
    hours (weekday or Saturday). A booking takes a free slot at random
    at or after its wanted time.
  * The routine cap (85%), tolerances and 5% routine cancellations are
    as before. The booking model stays simplified, so its access results
    are **provisional**.
* **Studies.**
  * Modality by priority, the simulator's mix.
  * Documentation time drawn from the simulator's per-modality
    distribution.
  * One read time per case, drawn from the simulator's per-modality
    distribution and scaled by the regime's factor (below).
* **Readers.**
  * **The simulator's two radiologists,** with its shifts by rota week.
    Each reads in 100-minute cycles at its site allocation and claims
    only inside a block.
  * **Shift-end rule:** a study is passed over if its modality's longest
    read would end after the shift.
  * **Overrun debt:** reading outside a block is owed out of the next
    block time, as in `internal/sim/reading.go`.
  * **The worklist** is urgent_then_fifo: urgent and past-target studies
    first, then the rest, each in entry order. Ties go to employee
    order, then the lanes.
* **Coverages.**
  * **`rota`:** the radiologists alone (R0 and the mirror).
  * **`satgap`, E3-like:** the rota, plus two urgent-only lanes Mon–Fri
    18:00–08:00 with a 10-minute handoff (the simulator's teleradiology
    weeknights window).
  * **`""`, stable:** satgap, plus a weekend reader Saturday 14:00–18:00
    and Sunday 08:00–12:00.
* **Regimes.**
  * The read-time scale is set so that the window's expected workload is
    the target share of the **core** capacity, the rota radiologists'
    block time.
  * Weekend readers and lanes add coverage on top of it. A85 therefore
    has the same staffing and read times as Wg85, plus its weekend
    reader.
  * **Measured and reported:** core load (all reads over core block
    time), the rota radiologists' own utilization, and the general
    utilization over every general reader.
* **Warm start.** 181 days before the takeover, inheriting the orders
  and studies open then.

**Model development, on the debug stream PCG(1, ·) only, before this
protocol was committed:**

* Uniform acquisition times and the unfitted weekday capacity gave
  Monday 13.8% and Saturday 7.0% of acquisitions. That led to the
  fitted factors and the slot times.
* The first profile counted 151 days in a 150-day window. Demand is now
  calibrated on the window itself.
* **The regime was first set over all general block time.** A85 then
  failed v4's soon gate on every seed (breach 0.45). Its weekend reader's
  hours are mostly idle, Sunday having no acquisitions, so its weekday
  radiologists ran at about 0.93. Setting the regime on core capacity
  makes A85 the stable practice it is meant to be (soon breach about
  0.14). This was decided from v4's own service gate and the meaning of
  the regime, not from any v5 value.
* **The mirror** (the rota alone at the simulator's own read times)
  measured utilization 0.957–0.968, against the simulator's 0.952. Its
  breach shares were still higher: soon about 0.43–0.55 against 0.36.
  * Overrun debt is the dominant term (without it, breaches fall below
    the simulator's).
  * The debt rule follows the simulator's code, so the residual is
    reported as a discrepancy (§6). The model is not tuned to remove it.
* **Calibration check** (debug stream, four seeds):
  * core load A85 0.86, B85 0.84, C85 0.85, Cs85 0.85; A92 0.94;
  * acquisitions by weekday within 0.3 points of the simulator's;
  * Saturday 5.9–6.1%;
  * every record valid, with inherited work.

## 3. Equivalence checks fixed before the run

The mirror **M** (1,000 seeds) is compared with the profile on:

* weekday acquisitions, the priority mix and read times;
* utilization;
* breach and severe shares, and the worst past due by priority;
* urgent studies acquired on Saturday afternoon;
* queue age;
* the takeover state.

**Simulator-level validity is claimed only if** the mirror's mean breach
and severe shares and worst past-due delays all lie within the
simulator's seed range (§1). From the debug-stream checks this is not
expected to hold for breach shares.

## 4. What changed from the corrected run

* Demand, mix, hours and modalities now come from the simulator.
* Individual readers in blocks replace a smooth rate.
* The weekend reader adds to, rather than shares, the core capacity.
* The weekend comparison is made at equal staffing (Wg) and at equal
  realized general utilization (WgU).

## 5. Scenarios

Streams: seed i of scenario k (its index in `ocFidelityScenarios`)
draws from **PCG(20261103, 100000·k + i)**, never used before. The
confirmatory streams PCG(20261201, ·) stay reserved.

| Regime 0.85 | Regime 0.92 | Departure | Role | Seeds |
|---|---|---|---|---|
| A85 | A92 | stable coverage, stationary | stable / stress | 2,500 |
| B85 | B92 | seasonal | stable / stress | 2,500 |
| C85 | C92 | bursty (day CV 0.15) | stable / stress | 2,500 |
| Cs85 | Cs92 | seasonal and bursty | stable / stress | 2,500 |
| Wg85 | Wg92 | E3-like (no reading Saturday after 14:00 or Sunday) at A's staffing and read times | structural (equal staffing) | 1,000 |
| WgU85 | WgU92 | E3-like at A's realized general utilization (faster reading) | structural (equal realized utilization) | 1,000 |
| R085 | | the rota alone at the same core load | structural | 1,000 |
| L10-85 | L10-92 | reading 10% lower from day 60 | deterioration | 1,000 |
| L20-85 | | reading 20% lower from day 60 | deterioration | 1,000 |
| D-85 | | reading 25% lower from day 30 | deterioration | 1,000 |
| K-85 | K-92 | reading eroding 0.2% a day from day 30 | deterioration (gradual) | 1,000 |
| IL10-85 | IL10-92 | imaging 10% lower from day 60 | deterioration (access) | 1,000 |
| IK-85 | IK-92 | imaging eroding 0.2% a day from day 30 | deterioration (access) | 1,000 |
| M | | the mirror: the rota alone at the simulator's read times | validation | 1,000 |

* **Reading capacity losses** lengthen every read by 1 / factor.
* **Deterioration is not presumed harmful.** Its clinical consequence
  is reported beside its detection.
* **0.92 is a stress regime** (owner: out of calibration). It is never a
  false-rejection target.

## 6. Reported

As in `correction-protocol.md` §6:

* per-gate failing fractions with Clopper–Pearson intervals;
* the three bundles, their components and sole causes;
* the 40-seed rejections with simultaneous intervals, and the disjoint
  groups;
* consequences: severe shares, worst past due, access, losses and the
  backlog.

Added:

* read-time scale, core load and core-reader utilization;
* Saturday share, weekday shares and mix;
* the takeover backlog, overdue studies and open orders;
* mean unread in days 1–30 against 121–150 (queue stability);
* the daily samples' oldest-unread age;
* urgent studies acquired on Saturday afternoon;
* breach shares.

The v4 service and backlog gates are unchanged. No value is selected.

## 7. Reproduction

    DT_SIM_PROFILE=<file> go test ./internal/sim -run 'TestSimulatorClinicalProfile$' -timeout 60m
    DT_OC_FIDELITY=1 go test ./internal/eval -run 'TestClinicalOCFidelity$' -v -timeout 600m

The output is committed as
`docs/evaluation/oc-v0.2-eval-5-exploratory-fidelity.json`.

* **A subset:** `DT_OC_SCENARIOS=… DT_OC_OUT=<file>` reproduces only
  those scenarios.
* **Tests.**
  * `TestSimProfileDerived` checks the committed profile.
  * `TestOCReaders` and `TestOCReaderRules` check blocks, debt, the
    shift-end rule, promotion and the lanes.
  * `TestOCFidelityCalibration` checks load, weekday shares and the mix
    on the debug stream.
  * `TestClinicalOCFidelityAgreesWithEvaluate` checks exact agreement
    with the production evaluator.
