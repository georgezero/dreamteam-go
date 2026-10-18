# Slice 5C.4b closure: access denominators and clocks

How v5 measures patient access, rule by rule. Each rule is as
implemented by `classifyAccess` and `measureClinical`
(`internal/eval/clinical.go`, built in cb3bb31), and is exercised by the
witnesses in `internal/eval/clinical_witness_test.go`. The no-action
simulator results come from `sim-access-5c4b.json`.

## 1. The rules

| | Rule | Implementation |
|---|---|---|
| **A. Eligible demand** | Every order received after the takeover whose **original wanted time** falls in the window, whatever became of it: served, open, lost, left, or cancelled after its deadline. Walk-ins are included, wanted on arrival. | `Access[window][priority].Wanted` |
| **B. Timely numerator** | Served with acquisition − wanted time ≤ the limit, in **calendar time**. | `Timely` |
| **C. Unresolved** | Open at the end of observation: untimely if the end is more than the limit after the wanted time, otherwise **undecided**. An undecided order makes the gate *insufficient*, which is never a pass. An order open past its deadline never leaves the denominator. | W3 |
| **D. Cancellations** (record outcome `patient`: a cancellation or a no-show, cause not recorded) | Resolved **before** the deadline: withdrawn (out of the denominator). Resolved **at or after** it: untimely, because the practice had already failed the patient. A rebooked no-show left without a time counts to the end of observation. | W2, W6 |
| **E. Lost demand** (`lost`: declined, no slot, displaced and lost, or lost after a rebooking; `left`: walk-in abandonment) | **Always untimely**, whenever it happened. | W2 |
| **F. Rescheduling** | The wanted time never moves; a displacement or rebooking never resets the clock. A rescheduled order imaged within the limit is timely; past it, untimely. | W4 |
| **G. Censoring** | Everything is judged at the policy's end of observation, at least 30 days after the horizon, the same instant for every plan. | W7 |
| **H. Received before the takeover** | Inherited cohort: open at the takeover and wanted by the horizon, timed from max(wanted, takeover). Subject to the maximum wait and reported; it has no rate gate (it is not new demand). | W7 |
| **I. Near the horizon** | Wanted by the horizon: in the cohort, followed to the end of observation. Wanted after it: outside (v4's case selection covers it). | W7 |
| **J. Priority changes** | Not supported: the simulator fixes priority at receipt, and the record has one priority per order. | none |

## 2. Gaming

| Attempt | What stops it |
|---|---|
| Deferring appointments | The wanted time is fixed: deferred orders are served late or open, and so untimely (W4, W12). |
| Cancelling difficult orders | A practice cancellation is not a `patient` outcome. The simulator records it as lost, which is untimely. |
| Moving acquisitions across windows | Urgent is judged over the whole gated period, soon and routine in both T90 and the tail; an order is placed by its wanted time, not its acquisition. |
| Suppressing unresolved demand | An open order stays in the denominator (C). |
| Reclassifying orders | The clinical gates cannot see a relabelled priority. v4's case selection against the paired baseline catches the fall in urgent acquisitions (W12: NO GO). |

**Residual ambiguity** (the record cannot tell): a patient's
cancellation caused by a long offered wait, and a no-show whose
likelihood rose with the booking lead. Both are excluded before the
deadline.

## 3. The two clocks (urgent)

* **A. Calendar:** wanted time to acquisition, as experienced by the
  patient.
* **B. Service opportunity:** the elapsed imaging-open hours (Mon–Fri
  08:00–20:00, Sat 10:00–18:00).

| | Calendar | Service opportunity |
|---|---|---|
| Interpretability | Direct: "imaged within 48 hours" | Operational: "within 24 open hours" |
| Patient experience | Is what the patient waits | Hides the weekend: Saturday 17:00 to Monday 08:30 is 39.5 calendar hours but 1.5 open hours (W5) |
| Feasibility | Weekend and Sunday closure bind | Always feasible for served orders |
| Risk of hiding failure | None | Hides weekend access delay |
| Verification | Two timestamps | Needs the practice's opening hours |

**Simulator, no action:**

* served urgent orders are imaged within 24 calendar hours in 82–89% of
  cases per seed, within 48 in 98–100%;
* within 12 open hours, 83–89%; within 24 open hours, 98.6–100%.

The clocks barely differ for served orders, because the simulator's
weekend urgent orders are **lost** rather than delayed. Lost orders are
untimely under either clock.

**Recommendation.** Calendar time is the primary, gated clock.
Service-opportunity time is at most a reported diagnostic. A 24-hour
calendar limit cannot be met through Sunday closure. At 48 hours the
closure binds only through the loss.

## 4. No-action simulator access (all eligible orders)

The pooled timely fraction over ten seeds, with the per-seed range:

| Priority | Limit | Timely | Untimely: lost / left / served late / open / patient late |
|---|---|---|---|
| urgent | 24 h | 0.284 (0.25–0.31) | 2,986 / 0 / 205 / 0 / 8 |
| urgent | **48 h** | **0.328 (0.30–0.35)** | 2,986 / 0 / 10 / 0 / 0 |
| urgent | 72 h | 0.330 | 2,986 / 0 / 1 / 0 / 0 |
| soon | 3 days | 0.409 | 4,855 / 1,068 / 2,751 / 0 / 154 |
| soon | 7 days | 0.578 (0.55–0.62) | 4,855 / 1,068 / 298 / 0 / 14 |
| soon | 10 days | 0.595 | 4,855 / 1,068 / 63 / 0 / 4 |
| routine | 14 days | 0.777 | 4,766 / 5,797 / 4,399 / 89 / 418 |
| routine | 21 days | 0.801 | 4,766 / 5,797 / 2,846 / 89 / 282 |
| routine | 30 days | 0.829 (0.82–0.85) | 4,766 / 5,797 / 1,042 / 89 / 101 |

* **Delay is not what limits access in the simulator; loss is.** Of
  the urgent orders served, 99% are imaged within 48 hours. Two in three
  urgent orders are never imaged.
* **A 90% or 95% urgent threshold over all eligible orders is
  unreachable** for this practice. It is likely unreachable for any plan
  too:
  * plans can recover at most the staffing-caused losses, about two
    thirds of them (`order-lifecycle.md` §3);
  * the rest are booking structure, which no plan action changes.
  * Even all the staffing losses recovered would give about 0.76.
* **The synthetic models** reserve capacity for urgent and soon work and
  lose almost nothing. Their access results (findings-fidelity §6) do
  not represent the simulator's access.
