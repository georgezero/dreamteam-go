# Slice 5C.4b closure: the simulator's order lifecycle

Read from the code (`internal/sim`, the evaluation recorder in
`internal/sim/export_eval_test.go`). Measured on the evaluation
scenario's practice with no action, development seeds 901–910
(`internal/sim/clinical_access_test.go`; output
`sim-access-5c4b.json`). No strategy ran.

## 1. States and transitions

**Order states:** booked, recall (wanted beyond the 120-day booking
horizon), lost, cancelled, visit, no-show, left (walk-in), imaged,
reported, displaced-lost.

| Event | Initiator | What happens |
|---|---|---|
| Receipt → offer | external (demand) | Booking is first fit: the earliest free slot of the exam's category on or after the wanted day, with 60 minutes' notice the same day. There is **no priority reserve**: booking ignores priority. |
| Offer finds no slot within the patient's tolerance | capacity against tolerance | **Lost immediately**: no waitlist. `lostWhy` is "declined, earliest slot …" or "no slot within 120 days". |
| Recall released into the horizon | practice booking | The same offer; it can be lost. |
| Cancellation (9%, flat, any priority; independent of the wait) | patient (random) | Half rebook from the next day (and can then be lost); the other half end cancelled. |
| No-show (4–8% by modality, plus 0.5% per 10 days of booking lead) | patient, rising with the practice's lead | 40% rebook the next day; the others end as no-shows. |
| Arrival after closing, or an exam that can't be served | practice | Rescheduled and re-offered; can be lost. |
| Walk-in patience runs out | patient, driven by the practice's queue | Left. |
| Staffing change displaces bookings (§7.2) | practice | Re-offered within the original tolerance, or displaced-lost. Each displacement is recorded. |
| Exam end / report final | practice | Imaged / reported. |

**Tolerance** by priority (days past the wanted day): urgent 0 / 0 / 2
(p0 / p50 / p100, so 75% of urgent orders must be imaged the same day),
soon 1 / 4 / 10, routine 5 / 21 / 60. **Priority never changes.**
**The wanted time never changes:** a rebooking re-offers from a later
day, but the record's wanted time is the original one.

## 2. What the evaluation record carries

| Internal end state | Record outcome | Initiator | Capacity-related |
|---|---|---|---|
| imaged, reported | served | practice | no |
| lost at the first offer | lost | capacity against tolerance | **yes** |
| lost after a cancellation, no-show, reschedule or recall | lost | patient or practice, then capacity | ambiguous (recorded as the practice's) |
| displaced-lost | lost | practice (staffing action) | yes |
| left (walk-in) | left | patient, driven by the queue | ambiguous |
| cancelled (no rebook) | patient | patient | no |
| no-show (no rebook yet) | patient | patient; its rate rises with lead | ambiguous |
| booked, recall, visit | open | none | none |

**Recorded:** priority, modality, walk-in, received, wanted, first
appointment, resolution and displacement times.

**Not recorded:**

* the lost reason;
* cancellation against no-show;
* rebooking counts;
* whether a loss followed a patient's cancellation;
* the recall flag (a recall order shows as open);
* any link between an order and its study.

The record's outcomes are not monotonic: a no-show may later be open
again, or lost.

## 3. Reconciliation (no action, ten development seeds)

| | Urgent | Soon | Routine |
|---|---|---|---|
| Orders received in the window | 4,591 (4.4%) | 15,694 (14.9%) | 84,958 (80.7%) |
| Served | 32.0% | 57.1% | 74.5% |
| **Lost** | **65.0%** | 31.5% | 6.5% |
| Left (walk-ins) | 0 | 6.8% | 6.8% |
| Patient (cancelled or no-show) | 2.9% | 4.6% | 6.9% |
| Open at the end | 0 | 0 | 5.3% |
| Lost at the first offer / later | 2,954 / 32 | 4,852 / 90 | 4,735 / 813 |
| Studies acquired | 1,471 (1.9%) | 8,972 (11.6%) | 67,170 (86.5%) |

* **The acquisition mix differs from the order mix through loss.**
  Urgent orders are 4.4% of orders but 1.9% of acquisitions: 65% are
  lost, almost all at the first offer.
* **Soon:** 31.5% lost and 6.8% left.
* **Routine:** acquisitions exceed served orders, because inherited
  orders are acquired after the takeover.
* **This is true patient loss, not a denominator artefact.** The orders
  are lost at receipt, by declined offers. They are not cancellations,
  reschedules, open orders or orders outside the window.

**The simulator's diagnosis of the urgent losses:**

| Cause | Count |
|---|---|
| Shared technologist conflict | 1,390 |
| Category full | 672 |
| No technologist rostered | 624 |
| Under 60 minutes' notice | 145 |
| Template category | 106 |
| No template slot | 49 |

* About two thirds of the losses (the technologist conflicts and the
  missing roster) are staffing, which a plan can change (hire, roster).
* The rest (full slots, notice, templates) are booking structure, which
  no plan action changes: there is no reserve or template action.

**Urgent orders wanted on Saturday from 12:00:** 104 of 108 were lost and
4 served. In the simulator the weekend gap shows mostly as **loss**, not
as delay.

**Lifecycle anomalies** (served before wanted, served without an
appointment, a patient outcome without a time, resolved but open, an
unknown priority): **none** in the ten records.
