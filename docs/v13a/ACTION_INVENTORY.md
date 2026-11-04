# V0.13a action inventory

Status: V0.13a (0.13.0-dev.1), design only. This is an inventory of
the engine at commit `8bbeca2` (V0.12.0). No action, rule, schema or
preview was added or changed to write it.

**Sources:**
* `internal/api/api.go`: the `Action` type and the action constants.
* `internal/sim/actions.go`: the operations actions, `Validate` and
  `ValidateBatch`.
* `internal/sim/staffing.go`: hire, terminate and atomic batches.
* `internal/sim/roster.go`, `allocation.go`, `telecontract.go`.
* `internal/sim/solvency.go`: the funding test.
* `internal/sim/preview.go`, `displace.go`.
* `internal/sim/offers.go`, `employment.go`.

The V0.12a consequence inventory (`docs/V0.12_CONSEQUENCE_INVENTORY.md`)
already lists which preview fields are exact. This document adds, for
every action type, what an alternative to it could be.

## 1. The eight action types

There are three operations actions and five staffing actions.

**Operations actions:**
* They apply one by one.
* They have no preview.

**Staffing actions:**
* They form one atomic batch, and an operations action may not share
  that batch (`mixed_batch`).
* They have a preview (`PreviewStaffing`).
* Under a credit line, they pass the funding test.

**Lead times:**
* `NoticeDays` is 14.
* The booking horizon is a scenario policy: 120 days in `established`.

## 2. Inventory

### 2.1 `extend_shift` (operations)

**Required inputs:**
* `employee`;
* `minutes`, from 1 to 240;
* optional `date`, today (the default) or a later day within the
  booking horizon.

**Validation:**
* `unknown_employee`, `invalid_minutes`, `invalid_date`;
* `past_date`, `beyond_horizon`;
* `not_on_shift`, `gone_home`, `shift_ended`;
* `exceeds_max_shift` (the shift cap);
* `after_close`: the shift already runs to the close;
* `no_coverage`: the extra time covers no room, desk or supervision;
* `insufficient_funds`: under a credit line, the paid time does not fit
  the headroom.

**Direct financial commitment:**
* The extra time is approved overtime, paid as worked.
* The engine computes the commitment only for the funding test. It does
  not report it to the caller.

**Direct operational change:**
* New bookable slots in the extra time, open at once.
* Today, the extra time also serves waiting patients.

**Reversibility:**
* No operation withdraws an approved extension.
* It is voided only when the employee leaves before that day.

**Lead time:** none. It applies at once.

**Preview fields:** none. The response says only whether the action is
accepted, with a message.

**No-action alternative:** meaningful. It means not extending.

**Existing actions that can serve as alternatives:**
* `extend_shift` with fewer minutes, or for another employee who is on
  shift;
* a `set_walkin_policy` change, which reduces demand instead of adding
  capacity.

None of these is previewable.

### 2.2 `set_walkin_policy` (operations)

**Required inputs:**
* `policy`: `accept_always`, `accept_if_gap`, `accept_until` or
  `schedule_later`;
* `cutoff` ("HH:MM") for `accept_until`, and only for it.

**Validation:**
* `unknown_policy`, `invalid_cutoff`.
* The current policy gives `unchanged`.

**Direct financial commitment:** none.

**Direct operational change:**
* How walk-ins are admitted, from now on.

**Reversibility:**
* Fully reversible: another `set_walkin_policy` restores the earlier
  policy.
* Patients admitted or turned away in the meantime are not restored.

**Lead time:** none.

**Preview fields:** none.

**No-action alternative:** meaningful. It means keeping the current
policy.

**Existing actions that can serve as alternatives:** the other three
policies, and `accept_until` with another cutoff.

### 2.3 `set_worklist_policy` (operations)

**Required inputs:**
* `policy`: `urgent_then_fifo`, `fifo`, `oldest_first` or
  `shortest_first`.

**Validation:**
* `unknown_policy`, `invalid_parameters`.
* The current policy gives `unchanged`.

**Direct financial commitment:** none.

**Direct operational change:**
* The order in which unread studies are offered.
* A read in progress is never interrupted.

**Reversibility:** fully reversible, from the next study offered.

**Lead time:** none.

**Preview fields:** none.

**No-action alternative:** meaningful.

**Existing actions that can serve as alternatives:** the other three
policies.

### 2.4 `hire` (staffing)

**Required inputs:**
* `requestId`, `role`, `skills`;
* `rateCents`: the standard or the premium rate of the offer;
* `contractedWeeklyMinutes`, from 1 to 2,400;
* `roster`: weeks A and B;
* `startDate`.

**Validation:**
* `invalid_schema`, `invalid_parameters`, `request_conflict`;
* `wrong_role`: only technologists and front desk;
* `invalid_skills`, `invalid_date`, `past_date`, `beyond_horizon`;
* `rate_out_of_band`, `invalid_contract_hours`;
* `lead_too_short`: the offer's standard or premium lead;
* `invalid_roster`, `headcount_limit`, `training_incomplete`;
* `insufficient_funds`.

**Direct financial commitments:**
* A recruiting fee, recognized at once and not refundable: $1,000 for
  front desk, $3,000 for a technologist.
* The contract guarantee per full week.
* The earliest lawful exit: the guarantee up to the earliest last day,
  plus two weeks' severance.

**Direct operational change:**
* A trainee from the start date.
* Productive after training: 40 hours for a technologist, 24 for front
  desk, within 60 days.
* Slots opened from the productive date (`opensSlots`).

**Reversibility:**
* Irreversible once committed. The `requestId` is idempotent, but there
  is no withdrawal.
* Undoing a hire takes a later `terminate`, with 14 days' notice after
  the start date and severance.

**Lead time:**
* 14 days (standard) or 7 (premium) for front desk.
* 28 or 21 for a technologist.

**Preview fields:**
* `hires[]`: offer, tier, rate, lead, contract, rostered minutes, start
  and productive dates, training and `opensSlots`;
* three `guaranteed[]` lines;
* `funding`.

**No-action alternative:** meaningful. It means not hiring.

**Existing actions that can serve as alternatives:**
* the other tier of the same offer;
* a later start date;
* fewer contracted minutes;
* `extend_shift` for current staff, which is temporary and not
  previewable;
* `contract_teleradiology` for reading capacity (another kind of
  capacity: not interchangeable).

### 2.5 `terminate` (staffing)

**Required inputs:**
* `employee`;
* `lastDay`, inclusive.

**Validation:**
* `invalid_schema`, `unknown_employee`;
* `invalid_parameters`: the same employee twice in one batch;
* `wrong_role`: only hourly technologists and front desk;
* `not_employed`;
* `invalid_date`;
* `already_terminating`: notice cannot be moved or withdrawn;
* `notice_too_short`: at least 14 days after both today and the start
  date;
* `beyond_horizon`.

There is no funding test: a termination commits no new spending that
the headroom must cover.

**Direct financial commitments:**
* Severance: two contracted weeks at the base rate with benefits. It is
  recognized at commit and paid on the first payday after the last day.
* Contract pay to the last day: the guarantee on the days still
  employed. It is a minimum, because worked pay replaces it.

**Direct operational changes:**
* The employee is employed to the last day.
* Rosters pending after the last day are voided.
* Approved extensions after the last day are voided.
* Every materialized day from the end date is re-planned.
* Booked appointments that lose coverage are displaced and resolved at
  commit, deterministically: rebooked or lost (`displace.go`).

**Reversibility:**
* Irreversible. A second `terminate` with another last day is refused
  (`already_terminating`).
* No operation rehires the same employee.
* A new `hire` is a different employee, with its own fee, lead and
  training.

**Lead time:** at least 14 days of notice.

**Preview fields:**
* `terminations[]`: employee, name, last day, end date, notice days,
  severance and its payday, `displaced`, voided rosters and extensions;
* `displacements[]`, each with a provisional replacement found by
  capacity alone;
* two `guaranteed[]` lines;
* one `forecast[]` line, "Contract cost no longer owed", labelled as an
  estimate.

`slotsClosed` is **not** reported for a batch that has only
terminations. It is computed only when the batch also changes a roster.
See `TERMINATION_VS_KEEP.md` §4.

**No-action alternative:** meaningful. It means keeping current
staffing.

**Existing actions that can serve as alternatives:**
* a later `lastDay`, up to the horizon;
* `set_roster` to reduce the employee's rostered hours (for hourly
  technologists and front desk). The contract guarantee is unchanged by
  a roster, so this changes capacity but not the guaranteed cost.

### 2.6 `set_roster` (staffing)

**Required inputs:**
* `op`: `set` or `remove`;
* `employee`;
* `effective`: a Monday at least 14 days ahead;
* `roster`, for op `set` only.

**Validation:**
* `invalid_schema`, `unknown_employee`, `wrong_role`, `not_employed`;
* `invalid_date`, `past_date`, `not_monday`, `notice_too_short`,
  `beyond_horizon`;
* `invalid_roster`, `invalid_parameters`;
* the composed-timeline checks.

**Direct financial commitment:**
* None new. The engine reports the contract guarantee as "unchanged by
  the roster".
* Idle contracted time and kept extensions are forecast estimates.

**Direct operational changes:**
* Coverage from the effective date (`slotsOpened`, `slotsClosed`).
* Possibly displaced bookings.
* A trainee's productive date may move.

**Reversibility:**
* Op `remove` withdraws a pending version, and its predecessor resumes.
* This is possible only while the version's effective date is still at
  least 14 days away.
* Bookings displaced at commit are not restored.

**Lead time:** 14 days, effective on a Monday.

**Preview fields:**
* `rosters[]`: before and after minutes, contract, productive dates and
  kept extensions;
* `slotsOpened`, `slotsClosed`, `displacements`;
* `guaranteed`, `forecast`.

**No-action alternative:** meaningful. It means the current roster.

**Existing actions that can serve as alternatives:**
* another roster;
* another effective Monday;
* `remove` of a pending version.

### 2.7 `set_reading_allocation` (staffing)

**Required inputs:**
* `op`: `set` or `remove`;
* `employee`: a salaried radiologist;
* `effective`: at least 14 days ahead;
* `siteAllocationBP`, from 3,000 to 10,000, for op `set` only.

**Validation:**
* `invalid_schema`, `unknown_employee`, `wrong_role`, `not_employed`;
* `invalid_date`, `past_date`, `notice_too_short`, `beyond_horizon`;
* `allocation_out_of_band`, `invalid_parameters`;
* `commitment_too_short`;
* `insufficient_funds`, for an increase.

**Direct financial commitment:**
* The change in the salary run rate, reported as guaranteed.
* It is "withdrawable at no cost until" the effective date minus 14
  days.

**Direct operational change:**
* Site reading blocks inside the same shifts.
* Supervision is unchanged, and no booking moves.

**Reversibility:**
* Op `remove` before the withdrawal date.
* Afterwards, a new version needs another 14 days.

**Lead time:** 14 days.

**Preview fields:**
* `allocations[]`: before and after allocation, site minutes, salary
  per 30 days, withdraw-by date and the backlog now;
* `guaranteed`, `forecast`, `funding`.

**No-action alternative:** meaningful.

**Existing actions that can serve as alternatives:**
* another allocation level;
* another effective date;
* `contract_teleradiology`, for reading capacity from outside.

### 2.8 `contract_teleradiology` (staffing)

**Required inputs:**
* op `start`: `requestId`, `plan`, `windows`, `scope`, and `effective`
  at least 14 days ahead;
* op `cancel`: `contract` and `endDate`.

**Validation:**
* `invalid_schema`, `invalid_parameters`, `request_conflict`;
* `unknown_plan`, `invalid_window`, `invalid_scope`;
* `invalid_date`, `past_date`, `notice_too_short`, `beyond_horizon`;
* `unknown_contract`, `already_cancelled`;
* `minimum_term`: at least 30 days after activation;
* `contract_overlap`;
* `insufficient_funds`.

**Direct financial commitments:**
* For op `start`: the monthly retainer, guaranteed through the earliest
  end, plus per-study fees as read.
* For op `cancel`: it fixes the end of an existing obligation and
  creates no new one.

**Direct operational change:**
* Remote reading in the contracted windows and scope, from activation.

**Reversibility:**
* A started contract can be cancelled only with at least 30 days'
  notice, and not before the minimum term.
* A cancellation fixes the end, and `already_cancelled` refuses a
  second one.

**Lead time:**
* 14 days to activate.
* 30 days to cancel.

**Preview fields:**
* `teleradiology[]`: retainer, guaranteed amount and through date,
  minimum term end;
* `guaranteed`, `funding`.

**No-action alternative:** meaningful.

**Existing actions that can serve as alternatives:**
* another plan, windows or scope;
* a later activation;
* `set_reading_allocation` for internal reading capacity.

## 3. Summary

| Action | Irreversible at commit | Preview | New guaranteed obligation | Funding test | No-action alternative |
|---|---|---|---|---|---|
| `extend_shift` | yes, no withdrawal | no | paid time, not reported | yes | yes |
| `set_walkin_policy` | no | no | none | no | yes |
| `set_worklist_policy` | no | no | none | no | yes |
| `hire` | yes | yes | fee, guarantee, earliest exit | yes | yes |
| `terminate` | **yes** | yes | severance and contract pay to the last day | no | yes |
| `set_roster` | no, until 14 days before its effective date | yes | none, guarantee unchanged | no | yes |
| `set_reading_allocation` | no, until its withdraw-by date | yes | salary change | for increases | yes |
| `contract_teleradiology` start | yes, cancellable after 30 days' notice | yes | retainer through the earliest end | yes | yes |

**Findings:**
1. Every action has a meaningful no-action alternative.
2. Every alternative listed above is an existing action with existing
   parameters. No new action is needed to express it.
3. Only staffing actions can be previewed. An alternative made of an
   operations action can be validated but not previewed. Its
   consequences are unavailable before it is applied.
4. Consequences are exact only for the batch as a whole. Displacement
   in particular is computed for the combined coverage of the batch. A
   comparison must therefore compare whole batches, not actions.
