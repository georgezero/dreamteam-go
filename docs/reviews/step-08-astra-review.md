# Step 8 review — gpt-6-astra, high reasoning

* Reviewed: commit `ee26417` (V0.1 step 8), read-only snapshot.
* Command: `codex exec -m gpt-6-astra -c model_reasoning_effort="high" -s read-only`
* Prompt: the eight focus areas listed in the assessment below (determinism and
  missing wakes, §6.9 safety limits, hold-over, overtime accounting, shared
  booking, leakage, same-instant ordering, test gaps).
* The findings below are the model's output, unedited. File links point at the
  temporary snapshot path used for the review; the line numbers match
  `ee26417`.

## Findings (verbatim)

I found **10 concrete defects**, ranked below. These are verified by tracing code paths, not executed reproductions. The offline test attempt failed because Go could not create its build directory on the read-only filesystem. No files were modified or modules downloaded.

1. **High — Radiologist handover can force another radiologist past their cap.**  
   [ops.go:402](/tmp/dt-review.sJC9a2/internal/sim/ops.go:402), [staff.go:185](/tmp/dt-review.sJC9a2/internal/sim/staff.go:185)

   With `MaxShift=2h`, radiologist A works 07:30–09:30, capped at 09:30; B works 08:00–09:00, capped at 10:00. A contrast exam starts at 08:55 and lasts 40 minutes. `supervisor` correctly accepts B because A cannot cover its duration. At 09:00, however, `neededFor(B)` sees A on regular duty and releases B without checking A’s cap. At 09:30, A is held until the exam finishes at 09:35, violating the hard cap.

   An intervening transaction can trigger an invariant panic; otherwise A can go off at exam completion before the invariant notices.

   **Fix:** retain a supervisor whose cap covers the remaining examination. Replacement checks must consider that commitment, and hold-over must have an explicit cap deadline.

2. **High — Missing cap and eligibility-expiry wakes let idle employees remain past their caps.**  
   [ops.go:587](/tmp/dt-review.sJC9a2/internal/sim/ops.go:587), [staff.go:96](/tmp/dt-review.sJC9a2/internal/sim/staff.go:96)

   Set `StartEarly=false`, `EarlyStart=0`. T1 works 07:30–09:00 with a 09:30 cap; T2 starts at 09:45. A patient booked for 10:00 arrives early and finishes registration before 09:00. At shift end, T1 is held because `neededFor` sees a waiting patient and an exam that currently fits within the cap.

   The appointment wake is at 10:00; there is no cap or last-feasible-start wake. With no intervening live event, T1 remains on duty until T2 starts at 09:45, then receives 45 minutes of overtime despite the 09:30 cap. Stale events and `RunUntil` frame boundaries cannot release T1.

   **Fix:** schedule explicit deadlines for held employees’ caps and relevant feasibility changes. Include appointment eligibility when deciding whether holding someone can actually help.

3. **High — Desk release happens too late for self-registration, potentially stranding a patient until accounting-close panic.**  
   [ops.go:82](/tmp/dt-review.sJC9a2/internal/sim/ops.go:82), [ops.go:103](/tmp/dt-review.sJC9a2/internal/sim/ops.go:103), [ops.go:21](/tmp/dt-review.sJC9a2/internal/sim/ops.go:21)

   Use a 09:00 close and `MaxShift=1h`. The desk employee works 08:00–09:00; the technologist works 08:30–09:00, capped at 09:30. A scheduled patient arrives at 08:59, registration takes two minutes, and exam plus documentation takes 21 minutes.

   At 08:59, desk registration cannot fit before the desk employee’s cap, but their presence blocks self-registration. At 09:00, matching again runs before `releaseHeld`: the desk still blocks self-registration, then goes home. The technologist stays, but registration is never retried and no wake is scheduled for this unregistered patient. With no further live events, `accountingClose` panics.

   The same ordering also allows desk/ops employees with remaining cap to start fresh registrations after their scheduled shift ends, contrary to §6.9.

   **Fix:** settle idle desk/ops departures before registration matching, and exclude expired desk staff from both registrar selection and the self-registration gate.

4. **High — The last technologist is released while their patient is registering.**  
   [ops.go:372](/tmp/dt-review.sJC9a2/internal/sim/ops.go:372)

   A patient arrives at 08:59 and registers with front desk until 09:01. Their only technologist’s scheduled shift ends at 09:00, with ample cap remaining.

   `neededFor` counts only `visitArrived` and `visitWaiting`, excluding `visitRegistering`. The technologist goes home at 09:00. When registration completes, coverage checking reschedules a patient who could have been served through required overtime.

   **Fix:** include registering patients in outstanding imaging demand when evaluating technologist hold-over.

5. **High — Midnight completions are invalidated before overtime and paid time are recorded.**  
   [staff.go:80](/tmp/dt-review.sJC9a2/internal/sim/staff.go:80), [world.go:190](/tmp/dt-review.sJC9a2/internal/sim/world.go:190), [validate.go:273](/tmp/dt-review.sJC9a2/internal/scenario/validate.go:273)

   Validation permits `shiftStart + MaxShift == 24:00`. Consider a technologist working 12:00–23:45 with a midnight cap. An exam runs 23:39–23:59, followed by one minute of documentation.

   The patient has departed at accounting close, so that check passes. At midnight, class-0 `dayStart` resets the employee and increments their generation before the class-2 documentation completion. That completion becomes stale. `goOff` never runs: paid time and overtime are lost, and the previous closing phase is cleared without its operational-close event.

   **Fix:** enforce a finish boundary compatible with accounting close, or explicitly finalize previous-day tasks and payroll before materializing the next day. Do not invalidate unfinished tasks during rollover.

6. **Medium — FirstFit skips an earlier preferred slot in a later room.**  
   [demand.go:403](/tmp/dt-review.sJC9a2/internal/sim/demand.go:403)

   An afternoon-preferring order has these free, compatible slots: room 0 at 08:00 and 15:00; room 1 at 13:00. After scanning room 0, `anyAt=08:00` and `dayPref=15:00`. The optimization skips room 1’s 13:00 slot because it is after `anyAt` and a preferred candidate already exists. The patient gets 15:00 instead of 13:00.

   **Fix:** skip only when a slot cannot improve either candidate: compare preferred candidates against `prefAt`, independently of `anyAt`.

7. **Medium — Slot-start staffing snapshots admit conflicting shared-technologist bookings.**  
   [demand.go:211](/tmp/dt-review.sJC9a2/internal/sim/demand.go:211), [demand.go:237](/tmp/dt-review.sJC9a2/internal/sim/demand.go:237)

   A technologist staffs XR throughout the morning and begins covering CT at 09:30. There is an XR slot 09:00–10:00 and a CT slot 09:30–10:00. Book CT first, then XR.

   XR’s `shared` flag is false because it was calculated at 09:00. Consequently, `sharedFree` immediately accepts XR despite the existing CT appointment requiring the same person during the second half-hour. Both slots independently pass `SlotBookable`.

   Runtime dispatch still serializes examinations, but booking promises overlapping capacity the roster cannot supply.

   **Fix:** evaluate shared capacity across every staffing and appointment boundary within the slot, rather than using only its starting segment.

8. **Medium — Opening appointments bypass shared-capacity validation entirely.**  
   [demand.go:444](/tmp/dt-review.sJC9a2/internal/sim/demand.go:444), [demand.go:548](/tmp/dt-review.sJC9a2/internal/sim/demand.go:548)

   `loadOpening` calls `book` directly; `book` checks occupancy and `why`, but never `sharedFree`. Scenario validation likewise checks individual room coverage, not shared capacity.

   The existing `micro-shared.json` demonstrates the accepted configuration: one technologist has simultaneous XR and CT opening appointments at 08:00 and again at 08:15. These bookings violate the rule applied to incoming orders.

   **Fix:** enforce shared feasibility for opening schedules and at the common booking boundary. If deliberate overbooking is needed for contention tests, inject it through an explicit test-only mechanism.

9. **Medium — Deferred breaks can overwrite another break and corrupt paid-time accounting.**  
   [staff.go:128](/tmp/dt-review.sJC9a2/internal/sim/staff.go:128), [staff.go:148](/tmp/dt-review.sJC9a2/internal/sim/staff.go:148)

   Schedule breaks at 13:00–13:15 and 13:20–13:35. Work delays the first until 13:18, so it should end at 13:33. At 13:20, the second break handler sees `onBreak`, falls through to `startBreak`, and replaces `breakFrom` and `breakUntil`.

   The 13:33 completion becomes stale. At 13:35, only 15 minutes are recorded although the employee was continuously unavailable for 17 minutes; part of the intended break allowance also disappears without a skipped-break count. Inflated paid time can change the weekly overtime premium.

   **Fix:** handle an already-active break explicitly and preserve accrued break time. Also avoid overwriting an existing deferred break without recording its disposition.

10. **Low — Patients reports negative waiting times before registration completes.**  
    [patients.go:151](/tmp/dt-review.sJC9a2/internal/sim/patients.go:151)

    For `visitArrived` and `visitRegistering`, `endedAt` is still zero, but `fillVisit` uses it instead of `now`. A patient arriving at 08:00 on simulation day zero therefore reports `WaitedMin=-480` while registering; later days produce larger negative values.

    **Fix:** use `now` for arrived/registering/waiting visits, `examStart` once imaging begins, and `endedAt` for departures without imaging.

I did not find a direct `RunUntil` chunking divergence, wall-clock/map-order dependency, latent-value leak through the reviewed projections, or backward-class same-time push in the validated paths. The missing-wake defects instead make outcomes depend on unrelated **live events**, which chunking tests alone will not detect.

The most valuable additional tests are:

- Sparse-event closing scenarios, especially desk departure followed by self-registration; repeat with stale events inserted.
- Unequal radiologist caps during an ongoing contrast exam, checking continuous supervision and cap compliance.
- Technologist shift end while the last patient is registering, plus early-arrival waits extending beyond a held employee’s cap.
- Midnight documentation completion and payroll rollover, including Sunday-to-Monday.
- Multi-room FirstFit preferences; shared capacity across coverage changes, opening bookings, and cancellations.
- Overlapping deferred breaks and patient wait projections in every visit state.
## Assessment

Each finding was checked against the code at `ee26417`.

| # | Severity | Verdict | Notes |
|---|---|---|---|
| 1 | High | **Confirmed** | `neededFor` (radiologist) releases a held radiologist when any other on-site radiologist is on regular duty, without checking that colleague's cap against the exam in progress. |
| 2 | High | **Confirmed** | Held staff are released only in dispatch, and nothing wakes dispatch at a held employee's cap. With `startEarly: true` (default) it needs a waiting patient who can't start, such as contrast with no supervisor, but it is reachable. |
| 3 | High | **Confirmed** | `deskOnDuty` counts desk staff past their shift end, so they block self-registration even when they can't fit a registration. Desk staff past shift end can also start new registrations, contrary to §6.9 (desk and ops are never held over). |
| 4 | High | **Confirmed** | `neededFor` (technologist) ignores patients in `visitRegistering`. |
| 5 | High | **Confirmed (edge)** | Validation allows shift start + `maxShiftHours` = 24:00 exactly. A cap must end before accounting close. |
| 6 | Medium | **Confirmed** | The FirstFit early-exit compares a preferred-window slot against the earliest *any* slot instead of the current preferred pick. |
| 7 | Medium | **Confirmed** | The slot staff mask is taken at slot start only. A slot that crosses a coverage boundary can share a technologist without being flagged. |
| 8 | Medium | **Confirmed** | `loadOpening` books without the shared-capacity rule. The mini opening Tuesday and `micro-shared.json` both double-book Riley/Tess at the same times. |
| 9 | Medium | **Confirmed (edge)** | A second scheduled break can start while a deferred first break is still running and overwrite it. |
| 10 | Low | **Confirmed** | `fillVisit` uses the zero `endedAt` for arrived and registering visits, giving negative waits in the patient list. |

No finding was rejected. Astra found no chunking divergence, map-order or
wall-clock dependency, observation leak, or backward-class same-time push.
Its point that missing wakes make outcomes depend on unrelated live events is
the key one: chunking tests can't catch that. Each fix gets a sparse-event
test, ideally with stale events inserted.

## Resolution

All ten findings were fixed in the step 8 remediation pass (version
`0.1.0-dev.8.1`). Each fix has a regression test in
`internal/sim/regress_test.go` that fails on `ee26417` and passes after the
fix. The test evidence is in `docs/PROGRESS.md` ("Step 8 remediation").
