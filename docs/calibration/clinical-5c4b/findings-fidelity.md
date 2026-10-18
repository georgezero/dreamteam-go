# Slice 5C.4b: simulator-fidelity findings

From `docs/evaluation/oc-v0.2-eval-5-exploratory-fidelity.json`, the run
fixed by `fidelity-protocol.md` (committed before it, 18b21d6).

* **How it ran.** On joofi from a compiled binary of 18b21d6, streams
  PCG(20261103, ·): 24 scenarios, 36,000 seeds.
* **Reproduction.** The Pi reproduced A85, Wg85 and the mirror M
  identically. The simulator profile reproduced byte for byte on the
  Pi.
* **Nothing is selected.** The confirmatory streams are untouched.
* **Simulator use:** only the no-action profile on the development
  seeds. No strategy, reference or holdout run.
* **No simulator-level validity is claimed** (§8).

Notation: "fails" is the share of seeds failing; R40 is the 40-seed
rejection. Delays are past the turnaround target, in hours.

## 1. Weekday and priority distribution

| | Simulator (profile) | Fidelity model (A85, 2,500 seeds) | First two models |
|---|---|---|---|
| Acquisitions, Mon–Sat | 11.9, 22.1, 20.9, 20.1, 19.3, **5.9%** | 11.9, 22.0, 20.8, 20.1, 19.3, **5.8%** | Saturday 12.8% |
| Acquisitions a day | 51.7 | 52.0 (debug-stream check, 16 seeds); set to the simulator's window total | about 50 |
| Mix: urgent, soon, routine | 1.9, 11.6, 86.5% | 2.0, 12.1, 85.9% | 2, 12, 86% |

* **The mix by weekday is not reproduced.** In the simulator soon work
  is 3.2% of Monday's acquisitions and 15.6% of Friday's. The model's
  mix is the same every day.
* **The order mix is not the acquisition mix.** The simulator receives
  4.4% urgent and 14.9% soon orders but loses 65% and 32% of them before
  imaging. The model reproduces the acquisitions, which are what the
  delay rules judge.
* **Access is provisional:** the model's access is not the simulator's
  (§9).

## 2. The individual-reader model

* **Readers.** The simulator's two radiologists, its shifts by rota
  week and 100-minute reading blocks at their site allocation (60% and
  50%).
* **Read times.** One draw per case from the simulator's per-modality
  distribution, scaled per regime.
* **Claim rules:** the shift-end rule, overrun debt and the
  urgent_then_fifo worklist.
* **Urgent-only weeknight lanes** (E3-like and stable practices), with a
  10-minute handoff.
* **A weekend reader** (stable practices only).
* **Tests.** `TestOCReaders` and `TestOCReaderRules` pin each rule. 13
  targeted mutations were all caught.

## 3. Achieved utilization

The regime is the core load: the window's read time over the
radiologists' block time. Means, with the p05–p95 range:

| | Core load | Radiologists' own utilization | General utilization (with the weekend reader) | Read-time scale |
|---|---|---|---|---|
| A85, B85, C85, Cs85 | 0.847–0.852 (0.82–0.88) | 0.81 | 0.73 | 0.892 |
| A92, B92, C92, Cs92 | 0.917–0.921 (0.89–0.95) | 0.86 | 0.79 | 0.966 |
| Wg85 / WgU85 | 0.851 / 0.736 | 0.85 / 0.74 | 0.85 / 0.74 | 0.892 / 0.772 |
| Mirror M | 0.954 | 0.954 | 0.954 | 1 (simulator) |

## 4. Warm start and queue stability

| | Unread at takeover, p50 (max) | Overdue at takeover | Open orders | Mean unread, days 1–30 / 121–150 | Drift, p05 / p50 / p95 |
|---|---|---|---|---|---|
| A85–Cs85 | 1 (5–7) | 0 (2–3) | 1,036–1,055 | 5.7–6.0 / 5.6–5.8 | −3 / 0 / +3 |
| A92–Cs92 | 1 (7) | 0 (3) | 1,037–1,053 | 8.8–9.3 / 8.9–9.2 | −5 / 0 / +5 |
| Wg85 | 1 (34) | 0 (3) | 1,052 | 9.6 / 9.6 | −4 / 0 / +4 |
| L10-85 | 1 | 0 | 1,056 | 5.7 / 10.9 | +1 / +5 / +10 |
| K-85, L20-85 | 1 | 0 | 1,055 | 5.6–5.7 / 25 | +12 / +19 / +32 |
| D-85 | 1 | 0 | 1,053 | 5.6 / 55 | +22 / +40 / +100 |
| Mirror M | 2 (112) | 0 (75) | 1,052 | 23.5 / 26.1 | −23 / +1 / +34 |
| Simulator | 5 (1–21) | 0.7 | 1,216 | | |

* **Stable and stress practices:** their queues do not drift, and their
  oldest unread at the daily samples is about 13 h (0.85) or 14 h
  (0.92) at p90.
* **The deterioration scenarios drift**, in proportion to the loss.
* **The model's open orders** at the takeover are fewer than the
  simulator's, which keeps recall lists and lost orders.

## 5. Weekend coverage

E3-like practices without weekend reading, at **equal staffing** (Wg)
and at **equal realized utilization** (WgU), against the stable
practice. The Saturday acquisition share is 5.84% in all of them.

| 0.85 | A85 (weekend reading) | Wg85 (equal staffing) | WgU85 (equal utilization) |
|---|---|---|---|
| Urgent worst, p50 / max | 0.1 / 18.6 h | 39.6 / 47.9 h | 39.0 / 44.3 h |
| Soon worst, p50 / max | 14.2 / 37.3 h | 43.0 / 48.0 h | 39.6 / 45.4 h |
| Severe share: urgent / soon / routine | 0 / 0 / 0 | 0.82 / 5.06 / 0.38% | 0.77 / 4.53 / 0.04% |
| Urgent acquired Saturday after 14:00, worst p50 | 0 | 39.6 h | 39.0 h |
| Seeds violating urgent 24 h or 36 h | 0 | 71.0% | 68.5% |
| … urgent 48 h | 0 | 0 | 0 |
| Seeds over 5 severe soon a week / 5 severe routine | 0 / 0 | 0 / 77% | 0 / 15% |
| Soon severe rate over 5% | 0 | 77% | 45% |
| v4 service fails | 0 | 0.7% | 0 |
| Bundle fails: lenient / moderate / strict | 0.1 / 0.1 / 99.9% | 3.5 / 29.3 / 100% | 0 / 0.6 / 100% |
| R40: lenient / moderate | 0.000 / 0.000 | 0.012 [0.002, 0.20] / 0.997 | 0.000 / 0.000 |

At 0.92:

* Wg92 violates urgent 24 h or 36 h on 68% of seeds, 48 h on 1.4%,
  and 72 h on none.
* WgU92 violates 24 h or 36 h on 67% of seeds and 48 h on none.

* **Without weekend reading, the worst urgent delay is about 40 h**
  whatever the utilization. It comes from an urgent study acquired on
  Saturday afternoon and waiting until Monday.
* **Only about 70% of seeds acquire such a study** in 150 days: the
  simulator's Saturday mix is 0.6% urgent. So a 24-hour or 36-hour
  urgent limit violates about 70% of E3-like seeds, and at zero
  tolerance rejects an E3-like plan over 40 seeds with certainty.
* **A 48-hour limit** does not bind at 0.85, and only barely under
  load.
* **With the urgent limit at 24 h, E3 must change, not the limit.**
  This follows the owner's provisional choice. Weekend afternoon
  urgent coverage would make an E3-like practice A-like here. The
  results do not depend on E3 itself.
* **The weekly concentration separates the weekend gap more than the
  rate does.** At equal staffing, more than 5 severe routine a week fails
  77% of Wg85 seeds, and a soon severe rate over 5% fails 77%.
* **The full bundles separate it only at moderate or strict.** Lenient
  passes Wg85 (R40 0.012).

## 6. Severe-delay tails

**Stable practices at 0.85** (10,000 seeds):

* no severe delay in any priority;
* worst urgent at most 14–20 h;
* worst soon p50 14.2–14.4 h, maximum 20–41 h;
* worst routine at most 37 h.

**Stress practices at 0.92:**

* severe soon 0.01–0.02%;
* worst soon maximum 56–59 h;
* worst urgent maximum 45–47 h.

The owner's provisional values, as the share of seeds failing each
gate:

| Value | A85–Cs85 | A92–Cs92 | Zero-tolerance T3 (bound 0.13%) |
|---|---|---|---|
| Urgent absolute 24 h | 0 / 10,000 | 4–11 / 2,500 (0.2–0.4%) | admissible at 0.85 (bound 0.12% per scenario); not at 0.92 |
| Soon absolute 48 h | 0 | 1–7 / 2,500 (0.04–0.3%) | admissible at 0.85; not at 0.92 |
| Routine absolute 7, 10, 14 days | 0 | 0 | admissible in both |
| Severe rate 5%: urgent, soon, routine | 0 | 0 | |
| Severe a week at most 5: urgent, soon, routine | 0 | soon 2–13, routine 8–19 / 2,500 (to 0.8%) | |
| **Urgent acquired within 1 day, 95%** | **99.9–100%** | **99.9–100%** | |
| Urgent within 2 days, 95% | 2.3–3.2% | 1.6–3.2% | |
| Urgent within 2 days, 90% | 0 | 0 | |

* **At zero tolerance, 40 seeds at 0.92 with the 24-hour urgent
  limit** reject with probability 1 − (1 − 0.002…0.0044)^40, that is
  8–16%.
* **The 1-day, 95% urgent access value is infeasible here.**
  * Imaging is closed on Sunday and shuts at 18:00 on Saturday, so an
    urgent order received late on Saturday cannot be acquired within a
    day.
  * About 5% of urgent orders arrive on Saturday, which puts the
    practice right at the 95% limit.
  * The rule at 0.90 still fails 93% of stable seeds.
  * The simulator has the same imaging hours. Its urgent orders are
    mostly lost instead, which the model does not reproduce, so the
    model's access figures are provisional.

## 7. Full-bundle operating characteristics

The protocol's bundles, unchanged (v4's service and backlog gates
included).

| Scenario | Seeds failing: lenient / moderate / strict | R40 lenient [simultaneous 95%] | R40 moderate | Groups of 40 rejected (moderate) |
|---|---|---|---|---|
| A85, B85, C85 | 0.1–0.3% / same / 99.9–100% | 0.000 [0, 0.068] | 0.000 | 0 / 62 |
| Cs85 | 1.2% / 1.2% / 100% | 0.000 [0, 0.068] | 0.000 | 0 / 62 |
| A92 | 1.7 / 1.8 / 99.9% | 0.001 [0, 0.070] | 0.001 | 1 / 62 |
| B92, C92 | 3.6–5.0 / 3.7–5.1 / 100% | 0.013–0.049 | 0.015–0.052 | 1–7 / 62 |
| Cs92 | 7.6 / 7.8 / 100% | 0.182 [0.111, 0.326] | 0.196 | 13 / 62 |
| L10-85 | 14.5 / 14.5 / 99.9% | 0.707 [0.54, 0.87] | 0.707 | 19 / 25 |
| L20-85, D-85, K-85 | 100% | 1.00 | 1.00 | 25 / 25 |
| IL10-85 | 0.1 / 0.2 / 100% | 0.000 | 0.000 | 0 / 25 |
| IK-85 | 0 / 4.7 / 100% | 0.000 | 0.038 | 3 / 25 |
| L10-92, K-92 | 99.6–100% | 1.00 | 1.00 | 25 / 25 |
| IL10-92 | 0.3 / 0.3 / 100% | 0.000 | 0.000 | 0 / 25 |
| IK-92 | 0.1 / 5.7 / 100% | 0.000 | 0.076 | 9 / 25 |

* **False rejection at 0.85 is essentially nil** under lenient and
  moderate.
  * Every failing stable seed fails v4's backlog rule, nothing else
    (2–30 of 2,500).
  * The simultaneous upper bounds (0.068) reflect only the Monte Carlo
    limit of a zero count.
  * Strict fails every stable seed through the 1-day urgent access
    value (§6).
* **At 0.92 the seasonal-and-bursty practice is rejected 18–20% of the
  time** under lenient and moderate, by v4's service and backlog gates.
  The 0.92 regime is a stress test: the owner keeps v4's 32% soon
  breach limit, which these practices meet on average (soon breach 0.22).
* **Clinically harmful losses are detected by the clinical gates as
  well as v4's.**
  * L20, D and K at 0.85, and L10 and K at 0.92, have severe soon
    delays of 0.6–18%, worst soon delays of 41–66 h at p50, and
    drifting queues.
  * Severe weekly counts and rates fire on 14–99% of their seeds.
* **L10 at 0.85 is caught 71% of the time, by v4's service gate**
  (127 of 145 failing seeds sole). Its clinical consequence is small:
  severe soon delays 0.02%, worst urgent p50 0.2 h. This is the
  capacity-against-harm distinction the owner asked for.
* **Imaging losses (IL10, IK) leave delays unchanged.** Moderate
  access detects imaging erosion on 4.7–5.7% of seeds (R40 0.04–0.08),
  and strict access fails everything.

## 8. Synthetic against simulator

The mirror M (the simulator's rota at its own read times, 1,000 seeds)
against the profile (10 development seeds):

| | Simulator | Mirror M | |
|---|---|---|---|
| Weekday acquisitions, mix | §1 | §1 | matched |
| Read times by modality | profile | profile | by construction |
| Reading utilization | 0.952 (0.947–0.961) | 0.954 | matched |
| Breach share: urgent / soon / routine | 0.10 / 0.36 / 0.07 | 0.16 / 0.49 / 0.12 | heavier |
| Severe share: urgent / soon / routine | 1.4 / 4.5 / 1.9% | 2.9 / 8.3 / 5.1% | heavier, 2–3× |
| Worst past due, p50: urgent / soon / routine | 36 (13–45) / 44 (42–45) / 41 (40–43) h | 44 / 63 / 45 h | soon and routine heavier |
| Urgent acquired Saturday after 14:00, worst | 33 h (0–45) | 41.5 h p50 | similar |
| Oldest unread age, p90 | 42–57 h | 42 h (p50), 88 h (max) | similar median, heavier tail |
| Unread at takeover | 5 (1–21) | 2 (to 112) | heavier tail |

* **The validity criterion fixed in the protocol (§3) fails.** The
  mirror's breach and severe shares and its worst soon and routine
  delays lie outside the simulator's seed range. No simulator-level
  validity is claimed.
* **The model is pessimistic** at the simulator's own load. Its
  reading tails are heavier than the simulator's, mainly through the
  overrun-debt term (`fidelity-protocol.md` §2), and possibly through
  the arrival timing by priority (soon and urgent acquisitions
  concentrate in the simulator's afternoon).
* **The calibration regimes are far from that load** (0.85 and 0.92
  core load against 0.95). There the model's stable practices show no
  severe delay at all.

## 9. Remaining limitations

* **Reading tails.**
  * Pessimistic against the simulator at its own load (§8).
  * Not validated at 0.85 or 0.92, because the simulator's own practice
    runs only at 0.95. Validating there would need simulator runs of
    other staffing, that is, strategies, which this slice does not
    run.
* **Access and booking.** Day slots, a routine cap and the longest
  tolerance. The model does not reproduce the simulator's urgent and
  soon losses, cancellations of every priority, no-shows or walk-ins.
  Every access result (the 1-day urgent infeasibility included) is
  provisional.
* **Mix by weekday:** not reproduced (§1).
* **Weekend reader and lanes:** synthetic coverage of the protocol's
  choosing (Saturday 14:00–18:00, Sunday 08:00–12:00, weeknight lanes
  18:00–08:00). A strategy's own coverage may differ.
* **Severity definition:** severe is fixed at 1,440 minutes past due,
  the provisional value. Other definitions are not explored here.

## 10. For the owner

| Provisional value | What this run shows |
|---|---|
| Urgent absolute 24 h | Admissible at zero tolerance for every 0.85 stable practice (0 of 10,000 seeds). It violates on 0.2–0.4% of 0.92 seeds (40-seed rejection 8–16%). It makes a practice without weekend afternoon reading ineligible: 70% of seeds violate. |
| Soon absolute 48 h | Admissible at 0.85 (0 of 10,000). It violates on 0.04–0.3% of 0.92 seeds. |
| Routine absolute | No routine delay passes 7 days in any stable or stress practice; the value can be set on clinical meaning. |
| Severe rate 5%, at most 5 a week | They cost no stable seed at 0.85, and at most 0.8% (the weekly count) at 0.92. They detect the harmful reading losses and the weekend gap. |
| **Urgent access 95% within 1 day** | **Infeasible in this model:** it fails 99.9–100% of stable seeds through Sunday closure. Within 2 days, 95% fails 2–3%; within 2 days, 90% fails none. The access model is provisional. |
| Shared allowance at 4 of 40 | Its rejection at 0.85 is nil, with every failure v4's own. |
