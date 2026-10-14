# DreamTeam — Progress

One section per roadmap step (ROADMAP §2), newest first. Each records what
was built, what the player can see, how it was tested, and known
limitations. The version shown in the TUI top bar (`DREAMTEAM v0.1.0-dev.N`)
and by `dreamteam version` names the latest step a build contains.

---

## Step 8 — Registration, imaging, overtime and closing rules (v0.1.0-dev.8)

**Planning changes first** (your decision): scheduled closing time no
longer cuts off admitted patients. PLAN D21, SIMULATION_DESIGN §3.4.2 (three
closes) and §6.9 (closing time and required overtime), the §6.4–6.5 rule
changes, invariants 7–8a, closing metrics, ROADMAP step 8, acceptance
criterion 5a, an exploit risk row, and the TUI closing status.

**Behavior**

* **Registration**: front desk, then the operations manager (×1.25), then,
  only with nobody desk-capable on duty, the patient's technologist (×1.5).
  The queue is urgent first, then scheduled patients by appointment time,
  then walk-ins.
* **Imaging**: patients across all free rooms are ranked by the queue policy
  (urgent, due by appointment, late fitted in, walk-ins under the walk-in
  policy, early arrivals) and started when a qualified technologist on the
  room is free and can finish by their 12-hour cap. Contrast also needs an
  on-site radiologist whose cap covers the exam. Room time and documentation
  are drawn per exam. The technologist documents for 1–12 min after the
  room is released.
* **Shared technologists**: one person, one exam. Booking now respects it
  too: a slot whose technologist also staffs another room isn't bookable
  while that person is already booked there (schedule shows `¦`).
* **Scheduled close** stops admissions only. Admitted patients are still
  registered and imaged. Staff past their shift end are **held over** while
  admitted patients need them (a technologist for the rooms they had, an
  on-site radiologist for contrast). They are released at the first instant
  they're not needed. Overtime minutes and cost (×1.5 past 40 paid hours a
  week, hourly staff) are recorded per person and day.
* **Coverage runs out** → the patient is rescheduled explicitly with the
  reason (for example "no XR technologist can finish within the 12-hour shift
  limit", "no radiologist on site to supervise contrast"), on arrival or
  while waiting. Nobody is sent home for the clock.
* **Operational close**: the first moment after scheduled close with nobody
  on site and nobody mid-task. It is logged with after-hours exams and
  overtime. Accounting close at 23:59:59 checks nobody is still on site.
* Breaks due while busy are deferred up to 60 min, or skipped (counted).
* **Template change [illustrative]**: Saturday XR/CT reduced to 12 + 7
  staggered slots. Sam works both rooms alone and registers everyone.

**On screen**

* Practice line: `● OPEN 08:00–20:00`, then `◐ CLOSING — 3 patients
  remaining · OT +42m ($38) after 20:00 · held over: Sam`, then
  `○ CLOSED · operations closed 20:47 (scheduled 20:00) · OT +1h 12m ($63)`.
* Rooms show scans in progress (`● Scanning P00614 Head w/o 14/12m Sam`,
  amber when past the expected time) and how many are waiting. Staff show
  their task (registering, exam, documenting) and overtime.
* Patient flow: on site by stage (registering, waiting by modality, in
  exam), longest wait, exams done, after-hours exams, coverage reschedules.
* Journey: registered (by whom, when), exam (technologist, room, wait, time
  taken against the expected time, after scheduled close), departed or
  rescheduled with the reason.

**Tests**

* Micro practices (`internal/sim/testdata/micro-*.json`, fixed durations)
  check exact timelines:
  * waiting patients stay after scheduled close, and an exam starts after it;
  * operations close at 09:36 against a scheduled 09:00;
  * 36 min of required overtime costs $36, or $54 past 40 weekly hours;
  * the shift cap forces an explicit reschedule with its reason;
  * a shared XR/CT technologist never runs two exams at once;
  * an on-site radiologist is held over for contrast after hours, and runs
    out of cap with an explicit reason;
  * patience expiry beats an exam start at the same instant, and a patient
    being registered can't leave.
* Across a month of the mini practice: every visit ends imaged, left, or
  rescheduled with a reason, overtime is recorded, and some days close
  late. Invariants are checked after every transaction: shift cap, one task
  per person, room occupancy, timestamp order, nobody on site at accounting
  close.
* Determinism: minute-by-minute chunks and various speeds and budgets give
  identical results.

**Measured on the mini practice** (28 days, seed 1): wait from arrival to
exam has median 14 min, p90 44 min, max ≈ 3 h (busy Saturdays and
evenings). Overtime averages ≈ 30 min a day (≈ $15). There are a few
after-hours exams a week.

**Limitations**

* No study or report yet. The worklist and radiologist reading are step 9.
  Radiologists only supervise contrast here.
* Held-over staff take no breaks. There is no minimum rest before the next
  shift yet (V0.2), so a late night doesn't thin the next morning.
* `extend_shift` (approved overtime) comes with the actions in step 13.
  Until then all overtime is *required*.
* Performance: 180 days of the mini practice take ≈ 1.3–1.7 s on the loaded
  Pi. Step 11 must apply the warm-up mitigations (MEASUREMENTS.md).

---

## Step 7 — Arrivals, missed appointments, walk-ins, patience (v0.1.0-dev.7)

**Behavior**

* **Scheduled patients arrive** early (≈ 60%, 5–25 min), on time (≈ 28%),
  late (≈ 9%, 5–20 min) or very late (≈ 3%, 20–45 min), or never. The no-show
  rate is a base per modality (XR 4%, US 7%, CT 6%, MR 8%) plus 0.5 points
  per 10 days of booking lead, capped at +6. Whether and when each patient
  comes is drawn by origin key when the appointment's day starts, and stays
  hidden until it happens.
* **Missed → no-show:** at start + 15 min an absent patient's appointment is
  marked *missed* (provisional). If they arrive later the same day they are
  fitted in (late policy `fit_in`) or rescheduled (`reschedule`). At
  accounting close (23:59:59) appointments still missed become *no-shows*.
  40% of no-shows ask for a new date the next morning. A patient who
  arrives after closing is rescheduled.
* **Walk-ins** (≈ 8 per weekday, about 85% X-ray, some venous ultrasound and
  head CT, never MR) arrive only in open hours, peaking late morning and
  late afternoon. Each becomes a patient and an order like any referral. If
  their modality has no technologist for the rest of the day (or the
  walk-in policy is `schedule_later`) they are booked an appointment and
  leave. Otherwise they wait.
* **Patience:** a waiting walk-in leaves after 30–120 min (hidden until it
  happens) and is counted as having left without an exam.
* **Closing time:** no exam starts after closing, so anyone still waiting is
  sent home unseen. **Until imaging exists (step 8) that is everyone who
  arrives**, so the waiting room fills through the day and empties at close.
* **IDs:** visits get `V…` IDs when the patient arrives. Each appointment and
  each order has at most one visit.

**On screen**

* The top bar shows the version: `DREAMTEAM v0.1.0-dev.7`.
* Dashboard **PATIENT FLOW**: waiting now by modality and the longest wait,
  no-shows at the last close, today's arrivals (late, after grace),
  walk-ins, walk-ins who left, provisional misses and patients sent home
  unseen. Each room line shows how many are waiting for it. On 80-column
  terminals the staff summary moves to the practice line, and the log keeps
  5 entries (8 on larger terminals).
* **Patients (`p`)**: a day's list with appointment time, actual arrival
  (with the offset when more than 5 min off), time waited and status.
  Filters (`tab`): all, waiting, missed / no-show, left / unseen, walk-ins.
  `←→` pages days. `enter` opens the **journey**: ordered or walked in,
  booked, missed (provisional), arrived (how early or late), waiting, left,
  sent home, no-show, rescheduled, lost. Only observed facts appear.

**Tests**

* Arrival offsets match the punctuality classes over 8 weeks. No-shows run
  4–10%, are finalized only at accounting close, and never have a visit.
  Patients who were marked missed and then arrived appear in the journey as
  "Missed (provisional) > Arrived".
* Conservation: every appointment of a finished day ended arrived, no-show,
  cancelled or rescheduled, with at most one visit per appointment and per
  order. Invariants (visit ↔ appointment ↔ order states, waiting list) are
  checked after every transaction in the simulation tests.
* Same instant: an arrival beats a grace check and a cancellation whatever
  the push order; grace then cancellation leaves the appointment missed; a
  patience expiry at closing time is handled before closing.
* Walk-ins: 6–12 per weekday, ≥ 75% X-ray, no MR, only eligible exams, only
  in open hours. Patience: departures between 30 and 120 min, counters match.
* Hidden-state noninterference: forcing every later patient to be a no-show
  and removing later orders and walk-ins changes nothing observable before
  it could be seen. A deliberate leak makes this test fail.
* Speed: steps of 1 min, 7 min and 2 h, and an event budget, give identical
  observations and patient lists. The existing booking, coverage and
  determinism tests pass.

**Limitations**

* No registration or imaging yet (step 8), so the waiting room only empties
  at closing time and rooms stay idle with patients waiting.
* `accept_if_gap` and `accept_always` both let walk-ins wait. The gap rule is
  applied when exams are dispatched (step 8).
* A no-show's new-date request comes the next calendar day at 09:00, even on
  Sunday (booking is centralized).
* Performance: 180 days of the mini scenario now take ≈ 0.8 s on the Pi
  (MEASUREMENTS.md). The step-11 warm-up target (< 1.5 s) will need the
  noted mitigations.

---

## Step 6 — Demand and booking (`ff54f40`)

Referral orders from six fictional sources arrive through the day. Counts,
times and attributes are keyed by origin, and IDs are given only when an
order arrives. FirstFit books slots that the roster staffs for the whole
slot (contrast slots also need an on-site radiologist). It tries the
patient's preferred time of day, then any time, within their wait
tolerance. Otherwise the order is lost (declined, or no slot within 120
days). Orders wanted beyond the horizon go to a recall list. Hidden
cancellations free the slot at once, and half the patients ask for a new
date from the next day. The mini scenario has a hand-built Tuesday of 47
appointments. TUI: dashboard DEMAND panel, compact dashboard log, full log
(`l`), schedule (`s`) with 15-minute timelines, slot list, day paging and
demand against capacity. Tests: determinism, demand unchanged by booking
capacity, feasible and non-overlapping bookings, demand accounting, recall
and cancellation paths, ID noninterference.

## Step 5 — Scenarios, rooms, staff, roster (`6ed41f3`)

Scenario JSON with `extends` and preset merging, strict decoding and
validation errors that name the JSON path. The bundled practice and the mini
scenario. Fortnight rota (weeks A/B), breaks, cover windows, radiologist
site reading blocks. Room states closed / unstaffed / idle, with one
staffing rule shared by the runtime and the coverage report
(`dreamteam scenario coverage mini`). TUI rooms and staff panels. Tests:
coverage timelines for Mon/Tue/Sat in both rota weeks, and the runtime
matches the plan minute by minute over a fortnight.

## Step 4 — API, session, TUI shell (`7ef6abf`)

Observation contract, session pacing (advance, event budget, step to the
next visible entry), Bubble Tea TUI with five speeds, pause, step and
auto-pause. The TUI cannot import the simulation (tested). Frame rate and
speed never change outcomes (tested).

## Step 3 — Event engine (`76476b2`)

Timestamp transactions with event classes, stale-event skipping, invariant
hooks. Chunking invariance tested, plus a synthetic 180-day load benchmark.

## Step 2 — Randomness (`0cbcf95`, `214d2d9`)

SHA-256-keyed draws with four fixed lanes per key, ChaCha8 streams,
integer-only uniform, Bernoulli, weighted, quantile and Poisson samplers,
golden values cross-checked against Python.

## Step 1 — Simulated time (`a3b4015`)

Integer seconds, practice calendar, clock and date formatting.
