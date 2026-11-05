# V0.13b slots closed by a termination (E4): `dreamteam.slots-closed/v1`

Status: implemented in 0.13.0-dev.2, `internal/sim/slotsclosed.go`,
narrowly approved by the owner. It is a separate, opt-in, read-only
request: `World.PreviewSlotsClosed(actions)` and
`Session.PreviewSlotsClosed(actions…)`. `PreviewStaffing` and its answer
are unchanged byte for byte; the agent protocol is unchanged.

## 1. What it counts

**The unit:** a template appointment slot of one room on one day: a slot
of the schedule the engine has already materialized for booking.

**A slot is counted when both hold:**
* staffing coverage makes it bookable now (`slotOK` under the current
  plan);
* it would not be bookable under the proposed registry: the batch's
  end dates set, nothing else changed.

**What a count is:**
* Each slot is counted once, booked or not. `bookedNow` is how many of
  the counted slots hold a booked appointment now.
* The counting rule is the roster preview's existing `slotsClosed` rule.
  A test checks that, for a roster batch, the same rule over every
  materialized day gives the roster preview's totals.

**The time window:**
* From the first day the batch changes anything: the earliest end date,
  the day after a last day, and never before tomorrow.
* To the last materialized day, today plus the booking horizon.
* `from`, `to` and `days` state it. If every end date is beyond the
  materialized days, the report is available with zero slots and no
  window.

**Breakdown:** by modality (`byModality`), and by day and modality
(`byDay`).

## 2. What it does not count or claim

* **Beyond the booking horizon:** no slot after the last materialized
  day is counted, because those slots do not exist yet. Capacity lost
  after the horizon is listed as unavailable (class C).
* **Demand:** a closed slot is capacity, not an order. The report does
  not say that a closed slot would have been booked, or that it is a
  lost patient, lost revenue or harm.
* **Displaced bookings:** these are a different count, the booked
  appointments that lose coverage. In every case tested, `bookedNow`
  equalled the preview's displaced count. This is an observation, not
  a rule of the report.
* **The front desk:** a front-desk termination closes no room slot.
  The report gives 0, because desk capacity is not a room slot.
* **Shared capacity:** a slot is counted by staffing coverage. Whether
  the shared-capacity rule would also let it be booked at a given
  moment is not part of the count.

## 3. When it is available

Only for a non-empty batch of `terminate` actions that the engine
accepts as a batch. Otherwise `available` is false, with:
* `not_termination_only`: empty, or any other action type in the batch;
* `invalid_batch`: the engine refuses the batch.

## 4. Exactness

**The test (`TestSlotsClosedExact`):**
1. Read the report.
2. Commit the same termination.
3. List every materialized slot from the end date that coverage makes
   bookable, before and after the commit.

**What must hold:**
* The slots that stop being bookable must be exactly the report's
  count, by modality.
* No slot may open.
* The window must start on the end date.

**Cases** (established scenario, takeover Tue 2026-10-13, slots counted
2026-10-28 to 2027-02-10, 106 days):

| Employee, last day, seed | Slots closed | Booked now | Preview displaced |
|---|---|---|---|
| E03, 2026-10-27, seed 1 | 3,540 | 175 | 175 |
| E04, 2026-10-27, seed 1 | 2,481 | 84 | 84 |
| E03, 2026-11-10, seed 2 (92 days) | 3,074 | 116 | 116 |

**Other tests:**
* Asking leaves the state, ledger and log digests unchanged.
* Every refusal leaves the run unchanged.
* The termination-only preview still has no `slotsClosed` field.
* The preview is byte-identical before and after a report.
