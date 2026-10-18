# Slice 5C.4b: combined clinical operating characteristics (protocol)

Written and committed **before any combined result was computed**. It
fixes the synthetic model, scenarios, streams, value grids, bundles and
targets of the exploratory run that informs the owner's choice of the
v0.2-eval-5 limits (`docs/evaluation/proposals/v5-clinical-acceptance.md`).

**Owner decisions behind it** (2026-10-18):

* Build the v5 measurements first and choose the values afterwards, from
  synthetic operating characteristics only, never from a strategy
  result.
* Leave the weekend question open: the urgent absolute-limit grid
  covers both sides of the structural Saturday gap (about 41 h).

**What it uses.** No strategy, development, reference or holdout run.
Synthetic queues only.

## 1. Two stages

* **Exploratory (this slice).**
  * The run reports, for every gate and value in §4, how often each
    synthetic scenario's seeds fail.
  * It also reports the shared-allowance 40-seed rejection for the
    fixed bundles in §5.
  * **Nothing is selected.** The owner chooses the values.
* **Confirmatory (the freeze slice).**
  * The owner's chosen values are evaluated once on **fresh streams**
    (§3), against the targets in §6, before v0.2-eval-5 is frozen.
  * If they miss a target, the owner decides. A limit is never adjusted
    to make a scenario pass after its confirmatory result is seen.
* **Why two stages.** It keeps the exploration that informs the
  choice apart from the evidence that validates it. Audit M2 noted that
  in 5C.3 the same counts both selected and estimated.

## 2. The synthetic practice

An hour-by-hour queue of 181 days from the takeover (Tue 2026-10-13
06:00), extending the Slice 5C.3 reading queue (`oc_queue_test.go`)
with an imaging stage. It produces complete evaluation records (orders,
studies, daily samples, captures), which the production `Evaluate`
judges.

* **Orders.** 52 a day, by the scenario's weekday and hour profiles,
  and by the season factor when seasonal. They are Poisson, with a
  gamma day factor when bursty. Priority mix: urgent 2%, soon 12%,
  routine 86%.
* **Wanted time.**
  * urgent: on receipt;
  * soon: receipt plus 0–2 days, uniform;
  * routine: receipt plus the scenario's lead distribution
    (`leadTimeDays`: p0 0, p8 1, p30 4, p70 15, p92 61, p100 181 days,
    linear between).
* **Imaging.**
  * Hours: Mon–Fri 08:00–20:00, Sat 10:00–18:00, no Sunday.
  * Daily slot capacity: 52 × 7 orders a week over 68 imaging hours, at
    imaging utilization 0.80, times the scenario's capacity factor.
  * Booking: at receipt, into the first day at or after the wanted day
    with a free slot. Routine may use only 85% of a day's slots, so the
    remainder is kept for urgent and soon work.
  * **Lost:** an order whose first free day is beyond its priority's
    longest patient tolerance (urgent 2, soon 10, routine 60 days;
    `waitToleranceDays` p100).
  * **Cancelled:** 5% of booked routine orders, at a uniform instant
    between receipt and the appointment.
  * **Acquisition:** uniform within the booked day's imaging hours,
    after the wanted time on the wanted day.
* **Reading.** The 5C.3 reading model: the team's rate at reading
  utilization 0.85 of season-1 demand; urgent first, then soon, then
  routine, first in first out.
  * **Arrivals are sorted** by acquisition time, which fixes the audit's
    M1 ordering defect. The 5C.3 queue (`ocQueue.record`) is kept
    unchanged, so `oc-v0.2-eval-4.json` still reproduces byte for byte.
  * **Reading hours of the stable practice** (`readingHours`). Every
    study is read within its priority's target or soon after:
    * Mon–Fri 07:30–20:00 at the team's rate, then one reader (half the
      rate) for urgent studies 20:00–22:00;
    * Sat 10:00–14:00 at the team's rate, then one reader to 18:00;
    * Sun one reader 08:00–12:00.
  * **Structural variants** (reported, never targets):
    * *Wg*: the stable hours without Saturday after 14:00 or Sunday. It
      keeps only the urgent weeknight reader, as the E-family
      strategies do.
    * *R0*: the scenario's own rota, Mon–Fri 07:30–20:00 and Sat
      10:00–14:00 (the 5C.3 queue's).
  * **Structural delays** follow from those hours. Without weekend
    reading (Wg, R0), an urgent study acquired on Saturday afternoon
    waits until Monday, up to about 40.5 h past due. Without the urgent
    weeknight reader (R0), one acquired late on a weekday waits about
    10.5 h.
* **Model development, before this protocol was committed.**
  * The first version gave the stable practice the scenario's own rota.
  * On a debug stream, PCG(1, ·), never used for results, its urgent
    and routine breach rates failed v0.2-eval-4's own service gates
    (about 21% and 8%). It could not represent an acceptable practice.
  * The stable hours above were set from that check: the coverage of the
    imaging hours, not any threshold. The old rota is kept as R0.
  * No result on the exploratory streams had been seen.
* **Money.** Constant, consistent synthetic finance (it plays no part
  here). Records are observed to the policy's observation end, with a
  common horizon of day 150.

## 3. Streams and seeds

* **Exploratory streams.**
  * Seed i of scenario k draws from its own stream, PCG(20261101,
    100000·k + i), so seeds can run in parallel and every result is
    reproducible.
  * The stable scenarios (A, B, C, Cs) use 2,500 seeds each, enough to
    bound a zero-tolerance limit at 0.13% (§6). Every other scenario
    uses 1,000.
* **Confirmatory streams (reserved):** PCG(20261201, 100000·k + i),
  never used before the confirmatory run.
* **Never used:** neither stream family has been used before. No
  holdout or reference seed is involved.

## 4. Scenarios

| | Scenario | Departure from the stable practice |
|---|---|---|
| A | Stable, stationary | none |
| B | Stable, seasonal | season factor |
| C | Stable, bursty | day CV 0.15 |
| Cs | Stable, seasonal and bursty | both |
| Wg | Structural: no weekend reading after Saturday 14:00 | reported only (the weekend decision) |
| R0 | Structural: the scenario's own reading rota | reported only (it fails v0.2-eval-4's service gates) |
| B92 | Stress: reading utilization 0.92, seasonal | reported only (owner: out of calibration) |
| I92 | Stress: imaging utilization 0.92, seasonal | reported only |
| L10 | Reading capacity 10% lower from day 60 | harmful |
| K | Reading capacity eroding 0.2% a day from day 30 | harmful |
| IL10 | Imaging capacity 10% lower from day 60 | harmful |
| IK | Imaging capacity eroding 0.2% a day from day 30 | harmful |
| D | Reading capacity 25% lower from day 30 | harmful (sustained deterioration) |

## 5. Values reported (per gate, per scenario)

Each is reported as the share of seeds failing it, with a Clopper–Pearson
95% interval. The severe threshold is fixed at 1,440 minutes (v4's
`staleAfterDueMin`, still **[proposed]**).

| Rule | Values |
|---|---|
| A.1/A.2 absolute past due: urgent | 12, 24, 36, 48, 72, 120 h |
| soon | 24, 48, 72, 120, 168 h |
| routine | 7, 10, 14 days |
| A.3 severe rate r_p: urgent, soon | 0.05, 0.10, 0.15, 0.20 |
| routine | 0.01, 0.02, 0.05, 0.10 |
| A.4 severe per week k_p: urgent | 1, 2, 3, 5 |
| soon | 3, 5, 8, 12 |
| routine | 5, 10, 20, 40 |
| B access limit A_p and timely fraction g_p | urgent 1, 2, 3 days; soon 3, 7, 10; routine 14, 21, 30; g 0.80, 0.85, 0.90, 0.95 |
| B.2 maximum wait | urgent 3, 7, 14 days; soon 14, 21, 31; routine 31 |

**Bundles**, fixed here, for the shared-allowance 40-seed rejection.
Each also applies v4's breach gates and v4's backlog rule (without the
sampled age limit):

| Bundle | Urgent / soon absolute | r urgent, soon, routine | k urgent, soon, routine | A urgent, soon, routine; g | Wmax urgent, soon, routine |
|---|---|---|---|---|---|
| lenient | 120 h / 168 h | 0.20, 0.20, 0.10 | 5, 12, 40 | 3, 10, 30 d; 0.80 | 14, 31, 31 d |
| moderate | 72 h / 120 h | 0.10, 0.10, 0.05 | 3, 8, 20 | 2, 7, 21 d; 0.90 | 7, 21, 31 d |
| strict | 24 h / 48 h | 0.05, 0.05, 0.02 | 1, 5, 10 | 1, 3, 14 d; 0.95 | 3, 14, 31 d |

Each bundle is also judged whole on disjoint groups of 40 seeds of each
scenario. This applies the seed rules and every gated pooled rate
(breach and timely access) together, as the production evaluator does.

The routine absolute limit is 14 days in every bundle. A bundle is
reported with each absolute limit as absolute, and with the maximum
wait seed-level.

## 6. Targets

They are fixed here and judged in the confirmatory run. In the
exploratory run they are only reported.

* **T1c (false rejection).** For every stable scenario (A, B, C, Cs),
  the 40-seed rejection by the combined clinical rule is at most
  0.05:
  * the shared allowance, P(Binomial(40, q) > 4), where q is the share
    of seeds failing any seed-level clinical gate;
  * together with the absolute limits, 1 − (1 − v)^40, where v is the
    share of seeds with any violation.
* **T2c (detection).** For every harmful scenario (L10, K, IL10, IK,
  D), the 40-seed rejection is at least 0.90.
* **T3 (admissibility of zero tolerance).** A limit may be absolute
  only if, on every stable scenario, the exact one-sided 95% upper bound
  of its per-seed violation probability is at most **0.13%**. With 2,500
  seeds this needs no violating seed.
* **Stress and structural (B92, I92, Wg, R0).** Reported, never a
  target. Wg and R0 show what the urgent absolute limit means for a
  practice without weekend reading (the owner's weekend decision).

## 7. Reproduction

    DT_OC_CLINICAL=1 go test ./internal/eval -run 'TestClinicalOperatingCharacteristics$' -v -timeout 600m

The output is committed as `docs/evaluation/oc-v0.2-eval-5-exploratory.json`.
A test checks that the production `Evaluate` and the fast per-gate
tabulation agree exactly, on sampled seeds and bundles.
