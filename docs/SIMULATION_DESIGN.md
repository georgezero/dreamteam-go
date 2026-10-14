# DreamTeam — Simulation Design

Status: **draft for review**. No code has been written yet.
Companion documents: [PLAN.md](PLAN.md) (vision, decisions, critique),
[ROADMAP.md](ROADMAP.md) (milestones, acceptance, testing),
[TUI_DESIGN.md](TUI_DESIGN.md) (screens and controls).

This document specifies the simulation engine: the domain model, the event
engine, the randomness model, the patient, workforce and financial processes,
the invariants, and the boundary between true state and what a player or agent
may see. A coding agent should be able to implement V0.1 from it without
redesigning anything. Items marked **[V0.x]** are later; the design leaves room
for them but V0.1 does not implement them.

> **Assumption labels.** Every number in this document is one of:
> **[illustrative]** — plausible and chosen for gameplay, not validated against
> practice data; **[calibration target]** — a property the default scenario must
> exhibit, and its parameters are tuned until it does; **[structural]** — a rule
> of the model, not a tunable value. No number here is validated operational
> data. All values live in scenario files, never in Go constants.

---

## 1. Modelling principles

1. **Discrete-event, not time-stepped.** State changes only when an event fires.
   Time between events has no effect on state. Nothing is computed "per frame"
   or "per minute".
2. **Individuals, not aggregates.** Each patient, appointment, study, employee,
   room and claim is a distinct entity with an ID and a state machine.
3. **Exogenous demand.** Referral demand is generated independently of the
   practice's capacity. The practice converts demand into bookings, or loses it.
   Lost demand is measured.
4. **Latent truth vs observed facts.** Each entity may carry *latent* attributes
   (a patient will no-show; an exam will take 41 minutes; a claim will be denied).
   The engine knows them. Players and agents learn them only when the event that
   reveals them happens.
5. **Integers everywhere in the engine.** Time is integer seconds; money is
   integer cents; probabilities are integer parts-per-million; rates are basis
   points. Floating point is used only for display and offline statistics.
6. **Policy-driven automation, explicit overrides.** The engine runs by itself
   under a set of policies (booking, walk-ins, dispatch, worklist, overtime).
   Player/agent actions change policies or make one-off interventions.
7. **Events are data.** No closures in the queue. Every pending event is a plain
   struct that can be serialized, diffed and replayed.

---

## 2. Domain model

### 2.1 Entity overview

```
ReferralSource ──generates──▶ Order ──booked as──▶ Appointment ──arrives as──▶ Visit
                                 │                                            │
                                 │ (one Order = one Exam in V0.1)             ▼
                                 └──────────────────────────────────────▶ Study ──▶ Report
                                                                            │
                                                                            ▼
                                                                          Claim ──▶ Payment(s)
Practice ─┬─ Room/Equipment (modality)
          ├─ Employee ── Shift(s) ── Assignment(s)
          ├─ Worklist (derived view over Studies)
          ├─ Ledger (journal entries)
          └─ Policies
Scenario ──builds──▶ initial State (incl. warm-up)
Event (queue item)  ·  LogEntry (observable record)
```

Patient-level identity (`Patient`) owns demographics-free synthetic attributes
and links its orders. The design deliberately separates **Order** (what was
requested), **Appointment** (a reserved slot), **Visit** (a physical arrival),
**Study** (the images acquired) and **Claim** (the billing artifact). Collapsing
these into a single "patient" record is the most common way such simulators
become impossible to extend (rescheduling, multi-exam visits, repeat imaging,
partial denials all need the separation).

### 2.2 IDs

There are two kinds of identifier:

* **Origin keys** (internal, never exposed) identify exogenous things before
  anyone at the practice knows about them. Examples: an order is
  `(day, sourceIndex, hourBlock, k)` and a walk-in is `(day, hourBlock, k)`.
  Randomness is keyed by origin keys (§4.3).
* **Public IDs** (`PatientID`, `OrderID`, `ApptID`, `VisitID`, `StudyID`,
  `ClaimID`; plus the static `EmployeeID`, `RoomID`, `SourceID`) are typed
  integers from per-type monotonic counters. Display form: `P01042`, `O…`, `A…`,
  `V…`, `S…`, `C…`, `E…`, `R…`.

**Public IDs are allocated only at disclosure**, i.e. in the handler of the
event that makes the entity observable (`OrderReceived`, `WalkInArrived`,
booking, arrival, acquisition, claim creation). Demand scheduled at the day
boundary exists only as origin-keyed pending events, with no public ID. So ID
gaps and ordering reveal nothing about unseen demand. A noninterference test
changes only future hidden demand and asserts identical IDs, observations and
log up to the divergence point.

IDs are **not** used to key randomness, because actions can change
allocation order.

### 2.3 Practice

| Field | Type | Notes |
|---|---|---|
| `Name` | string | "DreamTeam Imaging Center" |
| `Hours` | `[7]DayHours` | open/close per weekday; Sunday closed |
| `Rooms` | `[]Room` | one per scanner in V0.1 |
| `Employees` | `[]Employee` | |
| `Sources` | `[]ReferralSource` | |
| `Policies` | `Policies` | see §5.5, §5.9, §6.4, §6.8 |
| `Ledger` | `Ledger` | §8 |
| `Catalog` | `ExamCatalog` | exam types, from scenario |

### 2.4 Exam type (catalog entry)

Internal codes, not CPT codes (CPT descriptors are licensed; a mapping table can
be added later if ever needed).

| Field | Notes |
|---|---|
| `Code` | e.g. `CT_ABD_PEL_WC` |
| `Modality` | XR, US, CT, MR |
| `SlotMinutes` | booked slot length |
| `RoomTime` | quantile table (seconds) — patient-in-room through room ready |
| `TechExtra` | quantile table — tech documentation after room is released |
| `ReadTime` | quantile table — radiologist interpretation |
| `Contrast` | bool — requires a supervising radiologist on duty (§6.5) |
| `Prep` | enum: none, fasting, screening, labs — V0.1 records only; **[V0.2]** prep failures |
| `WalkInEligible` | bool — XR yes; most CT/MR no |
| `BaseRateCents` | reference reimbursement, before payer multiplier (§8.2) |
| `SupplyCostCents` | variable supplies incl. contrast agent |

### 2.5 Room / Equipment

| Field | Notes |
|---|---|
| `ID`, `Modality`, `Name` | |
| `State` | `Closed`, `Unstaffed`, `Idle`, `Occupied`, `Turnover`, `Down` |
| `Current` | `StudyID` when occupied |
| `DownUntil` | latent end of a failure (hidden) — the player sees "down, ETA unknown" or a vendor estimate |
| `BusyAccum` | integrated busy seconds (for utilization) |

State transitions:

```
Closed ──open──▶ Unstaffed ⇄ Idle          (⇄ = a qualified tech covering this room
                                             comes on/off duty or break)
Idle ──start exam──▶ Occupied ──exam end──▶ Turnover ──turnover end──▶ Idle | Unstaffed
Idle | Unstaffed ──failure / planned maintenance──▶ Down ──repair──▶ Unstaffed | Idle
any (not Occupied) ──practice close──▶ Closed       (Occupied finishes first)
```

`Unstaffed`/`Idle` is recomputed at the end of every transaction from shifts,
breaks and cover windows. A tech covering two rooms makes both `Idle`. The
state intervals drive utilization denominators (§11).

V0.1 folds turnover into `RoomTime` (one `Occupied` interval) and implements
`Down` only for scenario-scripted outages. Stochastic failures are **[V0.2]**.

### 2.6 Employee and shift

| Field | Notes |
|---|---|
| `ID`, `Name` (synthetic), `Role` | FrontDesk, Technologist, Radiologist, OpsManager |
| `Skills` | set of modality credentials (XR, CT, MR, US) for technologists; `Registration` for desk-capable staff |
| `PayType` | Hourly (non-exempt) or Salaried (exempt) |
| `RateCents` | hourly rate or annual salary |
| `SiteAllocationBP` | radiologists: share of paid time committed to this site (basis points) |
| `Roster` | weekly shift template per weekday |
| `State` | `OffDuty`, `Available`, `Busy`, `OnBreak`, `Overtime` (flag), `Absent` |
| `Task` | current task reference (registration, exam, read, documentation) |
| `WeekPaidSeconds` | for overtime computation |

A **Shift** is a concrete instance for one day: `{Employee, Day, Start, End,
Breaks[], Assignment}`. `Assignment` is the primary station (a modality room,
front desk, reading). Shifts are materialized from the roster at day start
(00:00 event) so that actions can modify a single day without changing the
template.

Employee state transitions:

```
OffDuty ──shift start──▶ Available ⇄ Busy
Available ──break start──▶ OnBreak ──break end──▶ Available
Busy ──break due──▶ (break deferred until task ends)
Available ──shift end──▶ OffDuty
Busy ──shift end──▶ Busy+Overtime ──task end──▶ OffDuty      (finish-in-progress rule)
any ──absence event──▶ Absent  [V0.2]
```

### 2.7 Referral source

| Field | Notes |
|---|---|
| `ID`, `Name` (synthetic), `Kind` | primary care group, orthopedics, OB/GYN, oncology, urgent care, self/other |
| `ModalityMix` | weights per exam code |
| `BaseDailyRate` | expected orders per weekday (fixed-point milli-orders) |
| `Relationship` | 0–1000 strength — fixed in V0.1, **[V0.4]** dynamic (TAT, access, outreach) |
| `UrgentShareBP` | |

### 2.8 Patient and order

`Patient`: `ID`, synthetic display name (generated from a fixed fictional
name list), age band, `PayerClass`, mobility flag **[V0.2]**, and latent
traits (§5.2). No real person data, ever.

`Order`: `ID`, `Patient`, `Source`, `ExamCode`, `Priority` (Routine, Soon,
Urgent), `CreatedAt`, `EarliestDate`, `PreferredWindow` (morning, afternoon,
evening, any), `OriginKey` (§4.3), `State`.

Order states:

```
Received ──booked──▶ Booked ──visit completes imaging──▶ Imaged
   │                   │  ├──cancel──▶ Cancelled
   │                   │  ├──no-show──▶ NoShow ──(rebook by policy)──▶ Booked
   │                   │  └──reschedule──▶ Booked (new Appointment)
   └──declined offer / no capacity──▶ Lost
```

### 2.9 Appointment

`ID`, `Order`, `Room` (modality), `Start`, `SlotLen`, `Kind` (scheduled,
same-day add-on, walk-in placeholder), `BookedAt`, `State`, latent
`ArrivalOffset`, `WillNoShow`, `CancelAt`.

```
Booked ──cancel event──▶ Cancelled
Booked ──arrival (≤ start + grace)──▶ Arrived
Booked ──(start + grace, not arrived)──▶ Missed          (provisional; slot released)
Missed ──late arrival, same day──▶ ArrivedLate ──LatePolicy──▶ Arrived | Rescheduled
Missed ──accounting close, never arrived──▶ NoShow       (terminal; rebook draw)
```

`Missed` is provisional. The patient may still turn up (latent arrival offsets
reach +45 min). `NoShow` is terminal and only assigned at accounting close
(§3.4.2), so an appointment can never be both rebooked and arriving. Patients
with `WillNoShow` never arrive.

### 2.10 Visit (the patient on site)

```
Arrived ──registration start──▶ Registering ──done──▶ Waiting
Waiting ──exam start──▶ InExam ──exam end──▶ Departed
Waiting ──patient leaves (walk-in patience exhausted)──▶ LeftWithoutExam
Waiting ──room down / staff gap──▶ (Waiting, or rescheduled by policy) 
```

Timestamps captured at every transition (arrival, registration start/end,
exam start/end, departure) — they feed wait-time metrics directly.

### 2.11 Study and report

```
Acquired ──enters worklist──▶ Unread ──claimed by radiologist──▶ Reading ──▶ Final
                                         ▲                            │
                                         └──── interrupted [V0.2] ────┘
```

`Study`: `ID`, `Order`, `Visit`, `ExamCode`, `Priority`, `AcquiredAt`,
`ClaimedAt`, `ClaimedBy`, `FinalAt`, latent `ReadSeconds`.
`Report`: V0.1 does not store report text — only status, author, timestamps,
and a synthetic one-line "impression placeholder" (never generated clinical
content). Turnaround time (TAT) = `FinalAt − AcquiredAt`.

### 2.12 Claim and payments

```
Pending(charge lag) ──submit──▶ Submitted ──adjudicate──▶ Paid | ShortPaid | Denied
Denied ──rework──▶ Resubmitted ──adjudicate──▶ Paid | WrittenOff
Paid | ShortPaid | WrittenOff ──settle──▶ Closed        (A/R balance → 0)
```

Latent: `Outcome`, `AdjudicationDelay`, `PartialFractionBP`, `Recovered`,
`ReworkDelay`. Outcome model and postings: §8.1–8.2.

### 2.13 Financial transaction

Double-entry journal entry: `{At, Memo, Lines[{Account, DebitCents, CreditCents}],
Ref}`. Accounts in V0.1:

| Account | Type |
|---|---|
| Cash | asset |
| Accounts receivable | asset |
| Patient service revenue (net expected) | revenue |
| Payroll expense | expense |
| Benefits & payroll tax expense | expense |
| Rent & facility | expense |
| Equipment service & leases | expense |
| Medical supplies & contrast | expense |
| IT / PACS | expense |
| Billing service fees | expense |
| Collection variance (denials, write-offs, short pays, over/under expected) | contra-revenue |
| Accrued payroll | liability |
| Accounts payable | liability |
| Opening equity | equity |

Double entry is barely more work than a single cash number and gives a free,
strong invariant (§10). It also makes the V0.3 accounting expansion additive.

### 2.14 Scenario

See PLAN.md §8 for the file format. In the engine a scenario is an immutable
`Scenario` value: catalog, practice, roster, sources, demand parameters,
behavioral parameters, financial parameters, policies, warm-up length, start
time, scripted events, preset overlays. `scenario.Build(sc, seed)` produces an
initial `State`.

### 2.15 Simulation event and log entry

`Event` — internal, may be hidden:

```go
type Event struct {
    At    simtime.Time // int64 seconds since scenario epoch
    Class uint8        // ordering class, §3.3
    Seq   uint64       // global monotonic insertion counter
    Kind  EventKind    // enum
    Gen   uint32       // generation of the target entity at push time (§3.2)
    A, B  int64        // entity refs / small args, meaning per Kind
}
```

`LogEntry` — observable, emitted by handlers when something a manager would
know happens: `{At, Severity, Category, Text, Refs}`. The event queue itself is
never exposed outside the engine.

---

## 3. Engine architecture

### 3.1 Packages

| Package | Responsibility | May import |
|---|---|---|
| `internal/simtime` | `Time` (int64 s), `Duration`, calendar (day index → date, weekday), formatting. No `time.Now`. | std |
| `internal/rng` | keyed deterministic streams, integer samplers (Bernoulli ppm, quantile tables, Poisson tables, weighted choice) | std |
| `internal/api` | **public contract**: `Observation`, `Action`, `ActionResult`, `LogEntry`, metric DTOs, JSON tags, schema version | simtime |
| `internal/sim` | authoritative `State`, entities, event queue, handlers, dispatch, ledger, policies, action validation/application, invariants, metric accumulators, projection to `api.Observation` | simtime, rng, api |
| `internal/scenario` | scenario JSON types, defaults, presets, validation, `Build` (incl. warm-up) | sim, simtime, rng |
| `internal/session` | owns one engine: `Advance`, `StepVisible`, `RunToEndOfDay`, `Apply`, `Observe`; action log; record/replay; digest; stop conditions | sim, scenario, api |
| `internal/tui` | Bubble Tea v2 UI | session, api, simtime |
| `cmd/dreamteam` | CLI entry | session, scenario, tui |

Rules:

* `sim` has **no** dependency on terminal, wall clock, filesystem, database,
  network or goroutines.
* `tui` (and later `web`, `agent`) must not import `sim`. They see the world
  only through `api.Observation`. An import-boundary test enforces this.
* Latent attributes are **unexported** fields inside `sim`. Go's visibility
  rules then make leakage a compile error for every other package.
* No interfaces for "pluggable" subsystems in V0.1. Policies are enums plus
  parameters with a `switch`. Introduce interfaces when the second real
  implementation appears (e.g., a scripted agent policy in V0.5).

### 3.2 Clock and queue

* `simtime.Time` is seconds since the scenario epoch, which is 00:00 practice
  local time on the scenario's epoch date. The epoch is derived:
  `epoch = startDate − warmupDays` (§8.4). Practice local time has **no DST and
  no time zones** **[structural]**. This removes a class of calendar bugs, and
  DST effects on demand don't matter at this fidelity.
* The queue is a binary min-heap ordered by `(At, Class, Seq)`. `Seq` is
  assigned at insertion from a global counter, which gives a total,
  deterministic order. No two events compare equal.
* Scheduling an event with `At < Now` is a programming error (panic in debug,
  invariant failure in tests).
* Cancellation is lazy. Each cancellable entity carries a `uint32` generation,
  and each event stores the generation (`Gen`) it was created against. An event
  whose `Gen` no longer matches is **stale** and is discarded on pop. Stale
  events never trigger dispatch, never advance anything except the clock, and
  are serialized like any other event (with `Gen`) so snapshots stay exact. A
  test checks that injecting arbitrary stale events never changes the digest.

### 3.3 Event ordering classes at equal timestamps **[structural]**

| Class | Kinds | Reason |
|---|---|---|
| 0 `DayBoundary` | day start (00:00): materialize shifts, schedule the day's order arrivals, daily fixed-cost accrual | everything else that day depends on it |
| 1 `CapacityUp` | practice open, shift start, break end, reading-block start, room repair | capacity appears before work that could use it |
| 2 `Completion` | registration end, exam end, read end, documentation end | resources are released before new claims |
| 3 `Arrival` | patient arrival, walk-in arrival, order received | |
| 4 `Deadline` | missed-slot detection (start + grace), walk-in patience expiry, cancellation, eligibility wake-ups (§3.4) | |
| 5 `CapacityDown` | break start, shift end, reading-block end, practice close, room failure | completions at the same instant finish first |
| 6 `Financial` | claim submit, adjudication, payment | |
| 7 `DayClose` | 23:59:59 accounting close: payroll accrual, missed → no-show finalization, KPI snapshot | |

### 3.4 Timestamp transactions **[structural]**

The unit of simulation is a **timestamp transaction**: all work at one instant
*t*, processed to completion. Nothing outside the engine (TUI frames, Max-speed
budgets, auto-pause, snapshots, observations) can observe or interrupt the
state in the middle of a transaction.

```
RunUntil(target, stop):
    while queue.peek().At <= target:
        t := queue.peek().At ; clock := t
        for class c := 0 .. 7:                         // microsteps
            while queue.peek() is (t, c, ·):
                e := pop()
                if stale(e): continue
                handle(e)          // may push events at t only with class ≥ c
        if any non-stale event was handled at t:
            dispatch()             // may push events only at > t
        assertInvariants()         // tests / --check-invariants
        if stop(visible entries emitted at t): clock stays t; return StopReason
    clock := target
```

Rules:

1. **No backward-class insertion.** While processing class *c* at time *t*, a
   handler may push events at time *t* only with class ≥ *c*. Anything else is a
   programming error (panic). Same-class pushes at *t* run within the same
   microstep, after the current event (by `Seq`).
2. **Termination.** At most `MaxEventsPerTimestamp` (default 100,000) events
   per transaction, or the run fails with an invariant error. Administrative
   cycles are bugs, not behavior.
3. **Dispatch once per transaction**, after all classes. A patient arriving at
   10:30 and a room freeing at 10:30 then compete fairly, regardless of
   insertion order. Dispatch only creates work with duration ≥ 1 s, so it never
   pushes events at *t*.
4. **Explicit wake-ups.** Dispatch never relies on unrelated events to wake it.
   After each dispatch, for every room left idle while visits wait for it
   (because none is eligible yet, e.g. early-start allowance, contrast
   supervision starting at a later shift, a break ending), the engine schedules
   one `EligibilityWake` (class 4) at the earliest time any waiting visit could
   become startable. It uses generation-based cancellation so only the latest
   wake per room is live. This makes outcomes independent of whether other
   events happen to occur in between.
5. **Chunking invariance** **[tested]**: `RunUntil(a); RunUntil(b)` ≡
   `RunUntil(b)` for any `a ≤ b`. Setting the clock to `target` with no events
   changes nothing. That is why TUI frame rate and speed can't affect outcomes.
   The TUI's Max-speed budget is checked only **between** transactions.
6. **Stop conditions never change state.** Auto-pause only truncates how far
   `RunUntil` goes, and only at transaction boundaries.

### 3.4.1 Actions and decision batches

* Actions are applied in **batches**: `ApplyBatch(t, actions)`. A batch is one
  decision. That's one TUI confirmation, or the action list an agent returns at
  a decision epoch. A batch applies at the current clock, after the transaction
  at `clock` (if any) completed.
* Within a batch, actions are validated and applied **in order**, each against
  the state left by the previous ones. A rejected action changes nothing and
  doesn't stop later actions. No dispatch runs between actions in a batch, so
  intermediate policy states have no effect.
* After the batch, events it pushed at the current time run as a transaction
  (classes ≥ 1, same microstep rules). Then **one** dispatch runs, but only if
  at least one action changed state. A batch where every action was rejected
  or was a no-op (e.g., setting the current policy again) triggers no dispatch.
* A policy change *can* legitimately cause work at a time with no other events
  (e.g., switching walk-in policy lets a waiting walk-in start now). That's
  intended, and it's deterministic because the batch boundary is recorded.
* The run record stores `{at, batchSeq, actions[], results[]}` per batch.
  Replay calls `RunUntil(at)` then `ApplyBatch` with the same actions, and
  asserts identical results.

### 3.4.2 Day boundaries: operational vs accounting

* **Operational day end**: the first transaction after practice close at which
  no visit is on site and no exam is in progress. It emits an observable
  `OperationalDayEnd` entry. The TUI's `e` key runs to it, and the end-of-day
  summary (operations metrics) is produced there.
* **Accounting close** (`DayClose`, 23:59:59): payroll accrual for the day,
  finalization of missed appointments as no-shows (§5.8), the daily KPI
  snapshot, and the daily P&L. Financial figures in the end-of-day summary are
  labelled "as of operational close". The daily P&L is final after accounting
  close.

### 3.5 Dispatch (resource contention)

`dispatch()` runs a fixed sequence of greedy matchers, each iterating entities in
**ID order** (never map order):

1. **Registration**: for each idle desk-capable on-duty employee (front desk
   first, then ops manager if `OpsMgrCoversDesk`, then — if no desk staff on
   duty — the visit's modality technologist when `TechSelfRegistration`),
   take the next `Arrived` visit by queue order (scheduled visits by
   appointment time, then walk-ins by arrival time).
2. **Imaging**: for each room in fixed order (XR, US, CT, MR): if room `Idle`,
   find an on-duty technologist with the skill whose assignment covers this
   room and who is `Available`; choose the next `Waiting` visit by the
   **imaging queue policy** (§5.9). Check constraints (§6.5). Start the exam:
   room → Occupied, tech → Busy, visit → InExam, push `ExamEnd`.
3. **Reading**: for each `Available` radiologist with site-reading time on the
   current block, claim the next `Unread` study by the **worklist policy**
   (§6.8). Push `ReadEnd`.

Greedy dispatch is intentional for V0.1. It is how real departments operate
(people pick the next patient), it is cheap, and it is a sensible baseline that
smarter policies (and agents) can beat. A shared technologist covering two
rooms (XR + CT) is modelled naturally: the tech is a single resource; whichever
room's matcher reaches them first claims them — so room order matters, and
**[V0.2]** a "tech prefers room with the most overdue patient" policy is an
obvious improvement.

### 3.6 Snapshots

`State` serializes to versioned, **canonical** JSON via explicit `MarshalJSON`
in `sim`. That includes every pending event (sorted by `(At, Class, Seq)`, with
`Gen`), all ID and `Seq` counters, entity generations, latent fields, policies,
the ledger and the running log hash (§4.5). Keyed RNG has no hidden stream
position (§4.3), so there is nothing else to save.

Snapshots are **not** observations. They contain hidden data, are only for save
games and debugging, and are *game* artifacts that benchmark agents never
receive. A snapshot is only taken at a transaction boundary. Round-trip test:
run → snapshot → load → continue gives the same digest as an uninterrupted
run. V0.1 implements snapshot round-trip in tests only; save/load UI is V0.2.

### 3.7 Performance

Rough volume: ~70–100 visits/day × ~12 events per visit + staffing/finance ≈
1,500 events per operating day. 90 days ≈ 150k events; warm-up of 180 days ≈
270k events. With a heap and no allocation-heavy logging this is well under a
second on a Raspberry Pi 5 class machine. Targets:

* `run --days 90` on the default scenario: **< 2 s** on the dev Raspberry Pi,
  < 0.5 s on a laptop **[target]**.
* Scenario build including warm-up: **< 1 s** on the dev Raspberry Pi.
* Observation projection: < 2 ms (it runs every TUI frame while playing).
* Memory: completed visits/studies older than a retention window are compacted
  into daily aggregates; the full event log is streamed to a writer (JSONL) when
  requested, otherwise kept as a bounded ring buffer of `LogEntry`s.

Benchmark runs parallelize across runs (one engine per goroutine), never inside
an engine.

---

## 4. Randomness and reproducibility

### 4.1 Requirements

* Same scenario + seed + action log ⇒ byte-identical event log and state digest.
* Identical across Go versions in the module's supported range, and across
  `amd64` and `arm64` (development happens on arm64; CI may be amd64).
* Changes in one entity's handling should not perturb random outcomes of
  unrelated entities (**common random numbers**) — essential for low-variance
  policy comparisons.

### 4.2 Hazards and mitigations

| Hazard | Mitigation |
|---|---|
| Go map iteration order is randomized | Never range over maps in `sim` for anything that affects state or output; entities live in slices indexed by ID; lint rule + review checklist |
| `time.Now`, global `math/rand`, goroutines | Forbidden in `sim`, `scenario`, `rng` (grep-based test) |
| **Floating-point divergence across architectures**: Go may fuse `x*y+z` into FMA on arm64 but not on amd64; `math.Exp/Log` have arch-specific assembly | **No floats in state-affecting code.** All sampling uses integer methods (§4.4). Cross-arch golden digest in CI |
| A single shared RNG stream: one extra draw shifts every later draw | Keyed streams (§4.3) |
| Unstable sorting | `slices.SortStableFunc` with total-order keys (ending in ID) |
| JSON float round-trip in scenarios | Scenario numbers are integers (cents, seconds, ppm, bp) |
| Library RNG algorithm change | `math/rand/v2` **ChaCha8**, whose output is specified and stable, plus golden tests |
| Seed recovery by a benchmark agent (matching observed history against candidate seeds) | Test-pack seeds are 256-bit secrets. Draws are SHA-256 outputs over keys containing the secret (and ChaCha8 streams seeded from them), so observing outputs doesn't reveal the seed or other draws (§4.3) |

### 4.3 Keyed random streams

**Derivation (byte-exact, so independent implementations agree):**

```
streamSeed = SHA-256( "dreamteam/v1" ‖ seed[32] ‖ enc(domain) ‖ enc(k1) ‖ … ‖ enc(kn) )
enc(string) = uvarint(len) ‖ utf8 bytes ;  enc(int64) = 0x01 ‖ 8-byte big-endian
stream      = rand.NewChaCha8(streamSeed)          // math/rand/v2
```

The game and dev seeds are small integers expanded to 32 bytes as
`SHA-256("dreamteam-seed" ‖ 8-byte big-endian n)`. Benchmark test seeds are
random 32-byte secrets.

**One fixed lane per attribute.** A SHA-256 digest yields four 64-bit
*lanes*. `rng.Hash(seed, domain, keys…)` returns them, and each lane is
permanently assigned to one attribute. Adding or changing an attribute never
shifts another. A fifth attribute in a group gets a new domain. Attributes
needing an unbounded number of draws (Poisson splits beyond four parts, lists)
use `rng.New`, a ChaCha8 stream seeded with the same digest. Origin keys (§2.2)
don't depend on public IDs or player actions:

| Domain | Keys | Lane 0 | Lane 1 | Lane 2 | Lane 3 |
|---|---|---|---|---|---|
| `demand.count` | day, sourceIndex, hourBlock | Poisson count (one lane per part; stream if > 4 parts) | | | |
| `order.core` | day, sourceIndex, hourBlock, k | second within block | exam | priority | lead time |
| `order.ext` | day, sourceIndex, hourBlock, k | preferred window | payer | — | — |
| `patient` | originKey | punctuality class | wait tolerance | walk-in patience | — |
| `attempt` | originKey, attempt | no-show | arrival offset | cancels? | cancellation lead |
| `attempt.exam` | originKey, attempt | room time | tech extra time | rebooks after no-show? | rebooks after cancellation? |
| `read` | originKey | read time | — | — | — |
| `claim` | originKey | outcome (paid / short / denied-recovered / denied-written-off) | adjudication delay | short-paid fraction | rework delay |
| `walkin.count` | day, hourBlock | Poisson count | | | |
| `walkin` | day, hourBlock, k | second within block | exam | payer | — |
| `failure` **[V0.2]** | roomIndex, day | | | | |
| `absence` **[V0.2]** | employeeIndex, day | | | | |

*attempt* increments per rebooking. `originKey` is the order's key tuple
`(day, sourceIndex, hourBlock, k)`, or the walk-in's `(day, hourBlock, k)`
with sourceIndex = −1.

Consequence: if an agent reschedules patient X, patient Y's exam still takes
exactly as long as it did under the baseline policy. Comparisons between
policies measure the policy, not the dice.

Each draw is a pure function of its key, so it doesn't matter *when* it's
computed, and latents can be computed on demand rather than stored.

Cost, measured on the dev Raspberry Pi 4: one `Hash` ≈ 2.7 µs (SHA-256 without
hardware acceleration), and a `New` stream ≈ 3.5 µs. About 7 hashes per order
plus demand counts is ≈ 600 hashes per simulated day, or ≈ 0.3 s for a 180-day
warm-up. One hash per attribute (~20 per order) would have cost ≈ 0.8 s; that
is why attributes are grouped into lanes.

### 4.4 Integer samplers

All arithmetic is on `uint64`/`int64`, with `math/bits.Mul64` for 128-bit
products. No floating point.

* **Uniform**: `u = stream.Uint64()`. Use `bits.Mul64(u, n)` high word for
  `[0, n)` (Lemire's method without rejection; bias < 2⁻⁴⁰ for our n, accepted
  and documented).
* **Bernoulli**: `uniform(1_000_000) < ppm`.
* **Weighted choice**: cumulative integer weights, binary search on
  `uniform(total)`.
* **Durations**: piecewise-linear inverse CDF from a quantile table in the
  scenario, e.g. `{"p0":360,"p10":480,"p50":600,"p90":900,"p100":1320}`
  (percentile keys strictly increasing, values non-decreasing, all ≥ 1 s). Draw
  `u = uniform(1_000_000)`. Find segment `[pa, pb]` with values `[va, vb]`
  (percentiles in ppm). Result = `va + floor((vb − va) × (u − pa) / (pb − pa))`.
  Exact and portable. It's also the format a practice would derive from its own
  RIS timestamps when calibrating against real data.
* **Counts (Poisson)**: built at scenario load for each distinct rate λ
  (milli-units). P(0) = e^(−λ) is computed in Q2.62 fixed point by an
  integer-only routine: range reduction plus a Taylor series with a fixed
  number of terms, specified in `rng` and golden-tested. The CDF follows by the
  recurrence P(k) = P(k−1)·λ/k in fixed point. The table is truncated at the
  first k where 1 − CDF < 2⁻⁴⁰, and the remaining mass goes to that last k.
  Sampled by inverse transform with `uniform(2⁶²)`. Rates above 16 are split
  into n = ⌈λ/16⌉ equal parts and the sample is the sum of n part samples
  (exactly Poisson; keeps e^(−λ) well inside fixed-point precision). Rates are
  capped at λ ≤ 200 per block (validation error above that).
* **Arrival times** within an hour block: `uniform(3600)` seconds per arrival,
  then sorted by `(second, k)`.

### 4.5 Digests and replay

* **Log hash**: a running hash `h ← SHA-256(h ‖ canonical(entry))` updated each
  time an observable `LogEntry` is emitted. It's stored in state and snapshots,
  so it doesn't depend on log retention or streaming.
* **State digest**: SHA-256 of the canonical snapshot serialization (§3.6).
* `Digest = SHA-256(logHash ‖ stateDigest)`.
* A **run record** is

  ```
  {schemaVersion, engineVersion, scenario: <fully resolved scenario JSON>,
   scenarioSHA256, seed (omitted/escrowed for benchmark test runs),
   until: <final sim time>, batches: [{at, batchSeq, actions[], results[]}],
   finalDigest}
  ```

  It embeds the resolved scenario (after `extends` and preset merge) and the
  explicit stop time `until`. A record therefore fully determines the run.
  `dreamteam replay record.json` re-runs to `until` and compares digests.
* Golden digests live under `testdata/golden/` for (scenario, seed, until,
  batches) tuples. Any intentional gameplay change updates them in the same
  commit, with the reason stated in the commit message.

---

## 5. Patients, demand and appointments

### 5.1 Demand generation

At each day boundary, for each referral source and each operating-relevant hour
block (orders arrive 07:00–19:00 on weekdays, reduced Saturday, none Sunday),
draw an order count from Poisson(rate) and schedule one origin-keyed
`OrderReceived` event per order at its arrival second. The order gets its public
ID, attributes and booking only when that event fires (§2.2). The rate is

```
rate = BaseDailyRate × HourProfile[h] × WeekdayFactor[d] × SeasonFactor[month]
       × RelationshipFactor (=1.0 in V0.1) × PresetDemandMultiplier
```

Each order draws exam code (source mix), priority, payer class, preferred
window, and **desired lead time**:

| Desired lead time | Share **[illustrative]** | Typical source |
|---|---|---|
| Same day (add-on/walk-in) | 8% | urgent care, primary care |
| 1–3 days | 22% | |
| 4–14 days | 40% | |
| 15–60 days | 22% | follow-ups |
| 61–180 days | 8% | surveillance imaging |

**[calibration targets]** are per weekday type, because the Saturday rotation
thins Mondays (§6.3):

| Day type | XR (incl. walk-ins) | US | CT | MR | Total visits |
|---|---|---|---|---|---|
| Tue–Fri | 28–40 | 10–13 | 14–20 | 8–11 | 65–85 |
| Monday, US week | 22–32 | 10–13 | 10–15 | 8–11 | 52–70 |
| Monday, no-US week | 22–32 | 0 | 10–15 | 8–11 | 42–58 |
| Saturday | 12–18 | 0 or 6–8 | 5–8 | 0 | 18–34 |

Weekday calibration criteria in ROADMAP apply to **Tue–Fri** only. Monthly
figures aggregate all day types.

### 5.2 Patient latent behavior

| Trait | Distribution **[illustrative]** | Revealed when |
|---|---|---|
| Arrival offset vs appointment | 60% early (5–25 min), 28% on time ±5, 9% late 5–20, 3% late 20–45 | patient arrives |
| No-show | base 6% (XR 4%, US 7%, CT 6%, MR 8%) + 0.5 pp per 10 days of lead time, capped +6 pp | start + grace passes |
| Cancellation | 9% of bookings; 55% within 72 h of appt, rest earlier; 50% of cancellers request a new date | cancel event |
| Wait tolerance (booking) | max acceptable days until appt: depends on priority and lead time; exceeded ⇒ declines offer ⇒ order **Lost** | at booking |
| Walk-in patience | 30–120 min before leaving | patient leaves |

Lead-time effect on no-shows is well documented qualitatively; the magnitude
here is illustrative.

### 5.3 Booking

V0.1 assumes **centralized scheduling**: booking happens at the moment the order
is received, by the automated booking policy, without front-desk labor.
(**[V0.2]** phone/scheduling workload consumes front desk time; unanswered
requests queue and some abandon — a real and important constraint, deferred for
scope.)

Booking policy `FirstFit` (default):

1. Candidate slots: template slots of the exam's modality whose category admits
   the exam (§5.4), on or after `EarliestDate`, matching the preferred window if
   possible, within the booking horizon (default 120 days), not reserved (§5.5)
   unless priority allows. A slot is **bookable only if the roster, as known at
   booking time, staffs that room with a qualified tech at that time** (and, for
   contrast categories, an on-site radiologist). Later roster changes that
   strand booked slots raise a conflict alert (V0.2 resolution actions).
2. Orders whose `EarliestDate` is beyond the horizon go to a **recall list**
   (surveillance follow-ups) and are booked when the date enters the horizon.
   They aren't lost.
3. Offer the earliest candidate. Patient accepts if `days ≤ WaitTolerance`;
   otherwise tries the earliest slot in *any* window; otherwise the order is
   **Lost** (logged as "declined — earliest slot in 23 days").
4. Urgent orders may use reserved add-on slots and, under policy
   `OverbookUrgent`, overbook today.
5. Slots freed by cancellations are immediately bookable again by later orders
   (FirstFit naturally backfills them). Freed slots less than 24 h out rarely
   refill, which is realistic.

Implementation details (V0.1 step 6):

* Lanes are read by index (`Draws.Lane`), so an attribute keeps its lane
  whether or not earlier lanes are used. `order.core` lane 0 (arrival second)
  is read when the day's demand is scheduled; the other lanes are read when
  the order is disclosed. `order.ext` lane 1 (payer) and `patient` lanes 0
  and 2 are reserved for later steps.
* Priority shapes the wanted date: urgent orders want today (lead 0), soon
  orders at most 3 days ahead, routine orders draw the lead-time table.
* Preferred windows: morning before 12:00, afternoon 12:00–16:00, evening
  from 16:00, or any.
* Same-day slots need `bookingNoticeMin` (default 60 min) of notice.
* Slots become bookable or not when a day is laid out (it enters the booking
  horizon), from the roster as known then.
* A patient who cancels and asks for a new date (rebook draw) is offered
  slots from the day after the cancelled appointment, never the freed slot.
  If no acceptable slot exists, the order is lost.
* Opening-schedule appointments (scenarios without warm-up) are confirmed:
  they carry no cancellation draw.
* Each order logs one line: booked, recall list, or lost (with the reason:
  declined because the earliest slot was too far, or no slot within the
  horizon).

Lost demand is a first-class metric: it is the cost of insufficient capacity
and the main signal that justifies hiring or extending hours.

### 5.4 Appointment templates

Per room, per weekday: a list of slots `{Start, Length, Category}` where
category is e.g. `MR_45` (brain/spine/joint w/o), `MR_60` (with contrast), `CT_15`,
`CT_30C` (contrast), `US_30`, `US_45`, `XR_15`. Templates are scenario data.
The established-practice template **[calibration target]** deliberately has
flaws for the player to fix:

* MR template ends at 16:30 because the MR tech leaves (evening capacity wasted).
* CT contrast slots placed after 18:00 when only one radiologist is on duty
  and that radiologist leaves at 20:00 — contrast exams cluster late.
* US runs 09:30–18:00 only (one sonographer), so it falls short of demand. US has
  the longest lead times and the most lost demand, while the room stays dark
  08:00–09:30 and 18:00–20:00. Template: 12 slots in the 8 staffed hours
  (7 × `US_45` and 5 × `US_30`, around the 13:30 break). Room time
  **[illustrative]**: `US_45` exams p0 25 / p50 38 / p100 52 min; `US_30` exams
  p0 18 / p50 26 / p100 36 min. Expected busy time ≈ 11 completed exams × ~33 min
  + ~1 add-on ≈ 390–410 min of 480 staffed (**80–90% staffed utilization**,
  ≈ 55% of open hours), after ~16% cancellations/no-shows with partial backfill.
* No reserved same-day CT add-on capacity.

Slot length vs reality: room time is sampled per exam; slots are only a plan.
Running late propagates: if room time exceeds slot length the next patient waits.

### 5.5 Reserved capacity and overbooking

Policies (V0.1 exposes only `WalkInPolicy`; the rest are **[V0.2]**):

* `ReserveAddOn{Room, Slots, ReleaseHoursBefore}` — hold slots for urgent
  add-ons; release to general booking N hours before.
* `OverbookPPM{Room}` — allow up to x% overbooking, accepting risk of delays.
* `WalkInPolicy` — `AcceptAlways`, `AcceptIfGap` (default: only if the room has
  an expected gap ≥ expected exam time before the next arrived scheduled
  patient), `ScheduleLater` (book a later slot instead).

### 5.6 Walk-ins

Walk-ins are patients holding an order but no appointment. Mostly XR. Per day
**[illustrative]**: XR 6–10, US 0–2, CT 0–1, MR ~0 (MR almost never walk-in:
safety screening, protocoling, authorization). Arrival profile peaks
10:00–12:00 and 16:00–18:00. Each walk-in has a patience; if not imaged in
time, they leave (`LeftWithoutExam`, demand lost).

### 5.7 Arrival, registration and waiting

* Scheduled patients arrive at `Start + ArrivalOffset`.
* Registration takes 3–7 min scheduled, 6–12 min walk-in **[illustrative]**,
  ×1.5 when performed by a technologist (self-registration), ×1.25 by the ops
  manager (interrupted by other work).
* After registration the visit is `Waiting` for its modality.

### 5.8 Late arrivals and no-shows

* Grace period 15 min (policy). At `Start + grace`, if the patient hasn't
  arrived, the appointment becomes **Missed**. The slot's remaining time
  becomes idle capacity that walk-ins or early arrivals can use.
* A patient arriving after grace (appointment `Missed`) is a **late arrival**.
  Under `LatePolicy=FitIn` (default) they are registered and wait for a gap
  (lowest scheduled priority). Under `Reschedule` they're offered a new slot
  (attempt+1) and leave. Either way they keep the same order and get one visit,
  so there are never duplicate visits.
* At accounting close, appointments still `Missed` become **NoShow**. The
  patient rebooks with probability 40% **[illustrative]** (new attempt, new
  latents; the rebooking is disclosed as a new order event the next business
  day).
* Absence is only revealed at `Start + grace`, and no-show only at close, so an
  agent can't "know" a slot will be free. It can only estimate from published
  base rates.

### 5.9 Imaging queue policy

Default `AppointmentOrder`: among `Waiting` visits for the room, pick the one
with the earliest appointment start that is ≤ now + early-start allowance
(10 min); else an early-arrived scheduled patient if the room would otherwise
idle (configurable `StartEarly`); else a walk-in per `WalkInPolicy`. Urgent
priority preempts the queue order (never an exam in progress).

### 5.10 Exam execution

```
ExamStart: room Occupied, tech Busy, visit InExam
ExamEnd (after RoomTime): visit Departed, room Idle, Study created → Unread,
         tech Busy for TechExtra (documentation), then Available
```

Optional **[V0.2]**: exam incomplete (MR claustrophobia ~2–4%, motion repeats),
contrast reaction delays, prep failures (patient ate before US abdomen →
reschedule).

### 5.11 Patient lifecycle (V0.1, end to end)

```
Order Received → Booked → (Cancelled | NoShow | Arrived)
Arrived → Registering → Waiting → InExam → Departed
Study Unread → Reading → Final → Claim Pending → Submitted → Paid/Denied → Closed
```

---

## 6. Workforce

### 6.1 Roles

| Role | Does | Pay **[illustrative]** |
|---|---|---|
| Front desk | registration, check-in; **[V0.2]** phones/scheduling | $21/h hourly |
| Technologist | exams in credentialed modalities; self-registration when desk unstaffed | XR $36, CT $40, US $44, MR $45 /h hourly |
| Radiologist | reading, contrast supervision; **[V0.2]** protocoling | $480k/yr salaried, site allocation 50–60% |
| Operations manager | management, covers registration (slower) | $95k/yr salaried |

Benefits and payroll taxes: +25% of wages **[illustrative]**.

### 6.2 The coverage arithmetic (why the starting roster is short)

Open hours: 5 × 12 h + 8 h = **68 h/week**. Covering all four rooms for every
open hour needs **272 room-hours/week** of technologist presence. Each default
tech shift is 8.5 h on site with a 30-min **unpaid** lunch, so it gives 8 paid,
available hours. Four techs × 5 shifts give **160 available tech-hours/week**,
a **41% shortfall** before accounting for sharing. Front desk needs 68 h; one
person provides 40. A tech who covers two rooms (XR + CT) is counted as staffing
both rooms (each room is "staffed") but can run only one exam at a time, so
shared coverage raises *staffed* room-hours without raising throughput.
Therefore **[structural]**:

* Roughly 40% of room-hours have no dedicated tech under the default roster
  (fewer are fully dark, because of sharing).
* The default scenario chooses *which* hours are unstaffed (see §6.3), making
  some capacity invisible-but-real: the room exists, the template may even
  contain slots, but nobody can run them.
* Saturday coverage means some employees work Tue–Sat, leaving Monday thinner
  (per-weekday targets in §5.1).

This is the intended first management problem.

### 6.3 Default roster — weekday **[illustrative]**

| Employee | Skills | Weekday shift | Breaks | Primary assignment |
|---|---|---|---|---|
| Front desk — Jordan | Registration | 07:45–16:15 | 12:00–12:30 | Desk |
| Ops manager — Taylor | Registration (slow) | 08:00–17:00 | flexible | Admin; covers desk |
| Tech — Riley | XR, CT | 07:45–16:15 | 11:30–12:00 | XR; covers CT until 11:30 |
| Tech — Sam | CT, XR | 11:30–20:00 | 15:30–16:00 | CT; covers XR after 16:15 |
| Tech — Casey | MR | 08:00–16:30 | 12:30–13:00 | MR |
| Tech — Morgan | US | 09:30–18:00 | 13:30–14:00 | US |
| Dr. Avery | reading, supervision | 07:30–16:00, 60% site | — | Reading |
| Dr. Blake | reading, supervision | 11:30–20:00, 50% site | — | Reading |

Consequences the player will see on day 1: CT runs on a shared tech in the
morning; XR and CT share a tech in the evening; MR is dark after 16:30; US is dark
before 09:30 and after 18:00; no front desk after 16:15 (ops manager until
17:00, then techs self-register).

Saturday and the Monday consequence (fortnight rota):

| | Mon | Tue–Fri | Sat |
|---|---|---|---|
| Riley (XR/CT) | 07:45–16:15 | 07:45–16:15 | off |
| Sam (CT/XR) | **off** | 11:30–20:00 | 10:00–18:00 |
| Morgan (US), week A | 09:30–18:00 | 09:30–18:00 | off |
| Morgan (US), week B | **off** | 09:30–18:00 | 10:00–18:00 |
| Casey (MR) | 08:00–16:30 | 08:00–16:30 | off |
| Jordan (desk) | 07:45–16:15 | 07:45–16:15 | off |
| Radiologist | both, as weekday | both | one, **remote** 10:00–14:00 (rotating) |

So on Mondays XR/CT is dark after 16:15, and US is dark on alternate Mondays.
Saturdays have no desk (techs self-register), no MR, and no contrast exams,
because the Saturday radiologist is remote (§6.5). Templates are defined per
weekday, and booking only offers slots the roster staffs (§5.3), so the
alternating weeks are handled without biweekly templates.

Names are fictional placeholders.

The bundled roster (`scenarios/base/established.json`) fills in what the
tables leave open. On Mondays Riley covers CT for the whole shift, because Sam
is off. On Saturdays Sam staffs XR and CT alone, with a 13:30–14:00 break.
Breaks are not covered: a single-tech room is unstaffed during its tech's
break. `dreamteam scenario coverage mini` prints staffed against open hours
for every day of the fortnight; about 74% of open room-hours are staffed.

### 6.4 Shifts, breaks and overtime

* Shifts and breaks are materialized each day from the roster (§2.6).
* A break is deferred if the employee is busy, up to 60 min; skipped breaks are
  counted (a workload signal).
* **Finish-in-progress rule**: an employee busy at shift end finishes the task.
  The extra minutes are **run-over time**: paid, counted as overtime pay if
  weekly hours exceed the threshold, and reported as *unapproved overtime* (a
  soft metric). Run-over is not a violation.
* **Start rule (soft, shift end)**: a technologist doesn't start an exam whose
  *expected* duration (catalog p50, not the latent sample) would end more than
  `OvertimeAllowance` (default 15 min) past shift end, unless overtime is
  approved for that shift (action `extend_shift`).
* **Admission rule (hard, shift cap)**: a task may start only if
  `now + maxDuration ≤ shiftStart + MaxShiftHours`, where `maxDuration` is the
  quantile table's p100 (room time + tech extra). Sampled durations never
  exceed p100, so the cap holds even in the worst case. `extend_shift` is
  rejected if it would push the shift past the cap.
* Overtime pay: ×1.5 for hourly staff beyond 40 paid hours in the week
  **[structural for US FLSA; configurable]**; daily-overtime rules (some states)
  configurable. Salaried staff: no overtime pay; extra hours count toward
  workload only.

### 6.5 Coverage constraints (hard)

Enforced by dispatch and by action validation:

1. An exam **starts** only with a technologist credentialed in the room's
   modality who is on duty (or on approved overtime) and not on break. Finishing
   past shift end is allowed (finish-in-progress).
2. A **contrast** exam starts only if a radiologist is **on duty and on site**
   for its whole expected duration (`ContrastSupervision=OnSite`). On-site status
   is part of each shift (`location: onsite|remote`), and it is separate from
   reading availability: a remote radiologist can read but not supervise.
   **[V0.2]** `Virtual` mode lets remote radiologists supervise where scenario
   rules allow. Without supervision, contrast patients wait (an eligibility wake
   is scheduled for when supervision starts) or are rescheduled.
3. One person, one task at a time; one patient per room.
4. No employee's shift exceeds `MaxShiftHours` (default 12) from shift start,
   run-over included (guaranteed by the admission rule, §6.4).
5. The practice does not start exams outside open hours (patients already in
   the building may finish).

Soft constraints, measured not enforced: skipped breaks, weekly hours above
target, consecutive long days, desk-unstaffed minutes with patients present.

### 6.6 Workload and fatigue

V0.1 tracks workload metrics only (busy %, overtime minutes, skipped breaks,
weekly hours). **[V0.4]** sustained overload raises a latent turnover risk
(resignation events) and modest slowdown — implemented only once hiring exists,
so the player has a remedy.

### 6.7 Radiologist interpretation capacity

* Radiologists read remotely or on site; reading is not bound to room hours.
* Site allocation (e.g., 60%) is modelled as alternating time blocks generated
  deterministically from the allocation: for 60%, 60 min reading this site then
  40 min other duties, repeating. That's simpler and more realistic than
  slowing every read. Block start and end are class-1 and class-5 events.
  **Overrun rule**: a read in progress at block end finishes, and the overrun
  counts as site reading time and is subtracted from the next site block. No
  new read is claimed during an other-duty block. Contrast supervision
  doesn't depend on blocks; it only needs the radiologist on duty and on site.
* Read time samples from per-exam quantile tables **[illustrative]**: XR 1.5–4
  min, US 4–9, CT 6–15 (A/P 10–20), MR 10–25.
* Default calibration **[calibration target]** (Tue–Fri): capacity
  8.5 h × (0.60 + 0.50) = **9.35 h** of site reading. Demand derived from the
  modality-weighted mix at p50 read times: XR 32 × 2.5 min + US 11 × 6 +
  CT 17 × 11 + MR 10 × 16 ≈ 80 + 66 + 187 + 160 = **493 min ≈ 8.2 h**, plus the
  Saturday and Monday-evening overflow arriving Tuesday. Capacity is therefore
  ≈ 105–115% of same-day demand, so the backlog drifts down through the week
  and spikes after weekends or when either radiologist is off. Tuesday starts with ~10–15 unread studies left from
  Monday evening (emerges from warm-up, not hand-placed).
* TAT targets (policy, used for metrics/alerts): urgent ≤ 60 min, routine ≤ 24 h
  **[illustrative]**.

### 6.8 Worklist policies

`UrgentThenFIFO` (default), `FIFO`, `OldestFirst`, `ShortestFirst`
(maximizes count, starves MR — a known exploit; benchmark metrics catch it via
TAT p90 per modality). Exposed as action `set_worklist_policy` in V0.1.

---

## 7. Equipment

* Rooms open at practice open if any qualified tech is on duty; daily QA
  (CT warm-up, MR QA) **[V0.2]** consumes 15–30 min of tech time before the first
  patient.
* **[V0.2]** Stochastic unplanned downtime: per-room daily failure hazard
  (e.g., MR ~1 failure / 60 operating days), repair duration distribution,
  vendor ETA visible with noise; planned maintenance windows the player schedules.
* V0.1: downtime only via scenario `scriptedEvents` (e.g., "CT down 13:10–15:40")
  to exercise the code path and the alerting UI.

---

## 8. Finance

### 8.1 Model

* **Global billing**: the practice employs its radiologists and bills one global
  charge (technical + professional) per study **[structural assumption,
  configurable later]**.
* **Revenue recognition**: when the report is finalized. The service is
  complete and billable then; a claim can't be submitted before the report is
  signed. Recognized at **expected net** = allowed amount × NCR, where allowed
  = base rate × payer multiplier, and **NCR** (net collection rate) is the
  payer class's *expected lifetime* collected fraction. NCR already includes
  denials, recoveries, short payments and self-pay non-payment (§8.2), so
  nothing is counted twice. Revenue is never recognized at exam completion or
  because time passes.
* **Charge lag**: claim submitted 1–3 business days after finalization.
* **Adjudication**: payer-class delay distribution and a latent outcome (§8.2).
  V0.1 denial policy is automatic rework: a denied claim is resubmitted after
  `ReworkDelay` and either recovered (paid in full) or written off.
* **Cash** moves only on payments received and on paid expenses.

**Journal postings for claims** (every claim ends with zero A/R balance):

| Event | Entry |
|---|---|
| Report final | Dr A/R / Cr Patient service revenue — expected net (E) |
| Payment received (any outcome, any number of payments) | Dr Cash / Cr A/R — amount paid |
| Claim settles (Paid in full, ShortPaid final payment, or WrittenOff) | residual R = E − Σpaid. If R > 0: Dr Collection variance (contra-revenue) / Cr A/R — R. If R < 0: Dr A/R / Cr Collection variance — −R |

Net revenue = Patient service revenue − Collection variance. Because E is the
expected value of collections, collection variance averages to ~0 over many
claims. A persistent bias is a calibration bug, and a test checks it. Bad debt
and denial write-offs are both collection variance in V0.1 (split by reason in
V0.3).

### 8.2 Reimbursement **[illustrative]**

Base rates (≈ a public-payer-like reference, round numbers, not a fee schedule):

| Exam group | Base |
|---|---|
| XR (1–3 views) | $45 |
| US | $140–170 |
| CT without contrast | $160 |
| CT with contrast | $260 |
| MR without contrast | $260 |
| MR with & without contrast | $380 |

Outcome model per payer class. Probabilities are mutually exclusive; *d* =
denied, *r* = recovered on rework, *s* = short-paid, *m* = mean paid fraction
when short-paid (uniform ±10 pp around *m*):

| Payer class | Mix | Multiplier | Delay | d | r | s | m | **NCR** = (1−d−s) + s·m + d·r |
|---|---|---|---|---|---|---|---|---|
| Commercial | 45% | 1.60 | 25–60 d | 7% | 50% | 5% | 0.70 | **0.950** |
| Medicare | 35% | 1.00 | 14–30 d | 4% | 60% | 2% | 0.80 | **0.980** |
| Medicaid | 10% | 0.75 | 30–75 d | 9% | 40% | 4% | 0.70 | **0.934** |
| Self-pay | 10% | 0.90 | 30–120 d | — | — | — | — | **0.400** (30% pay in full, 20% pay half on average, 50% pay nothing) |

Rework delay 30–60 days. Mix-weighted revenue factor =
0.45·1.60·0.950 + 0.35·1.00·0.980 + 0.10·0.75·0.934 + 0.10·0.90·0.400
= 0.6840 + 0.3430 + 0.0701 + 0.0360 = **1.133** (expected net per $1 of base).

### 8.3 Expenses **[illustrative]**

| Item | Amount | Timing |
|---|---|---|
| Rent & facility | $22,000/mo | accrued daily, paid 1st of month |
| Equipment leases (CT, MR) | $9,000 + $18,000/mo | same |
| Service contracts (MR, CT, US, XR) | $12,000 + $8,000 + $1,500 + $1,000/mo | same |
| Utilities | $7,000/mo | same |
| IT, RIS/PACS | $6,000/mo + $3/study | same; per-study at acquisition |
| Insurance (liability, property) | $4,000/mo | same |
| Supplies | XR $3, US $8, CT $15, MR $20 per exam; contrast +$35 CT, +$60 MR | at exam end |
| Billing service | 5% of cash collected | at payment |
| Payroll | §6.1 | accrued at shift/task end, paid biweekly |

Accrual vs payment: daily accrual entries make daily P&L meaningful; cash
payments happen on schedule (payroll every other Friday, rent on the 1st). This is
why the Tuesday screen can show positive operating profit and falling cash at the
same time — a real phenomenon the player must learn.

### 8.4 Opening balances (established practice)

* Warm-up runs the full engine (operations and finance) from
  `epoch = startDate − warmupDays` to `startDate startTime` under default
  policies. With the default 180 days, every lead-time bucket (max 180 days) has
  had time to fill the forward schedule. A **convergence test** checks that
  forward-booking fill by lead bucket and A/R aging at start are within ±5% of a
  365-day warm-up.
* At the start time, after warm-up, one journal entry `Opening equity
  adjustment` (Dr/Cr Cash vs Opening equity) sets cash to the preset's figure
  (default $150,000).
* **Boundaries:** everything before the start time is *history* (shown in
  trends, never scored). Score windows, cash-change metrics and the cash
  invariant baseline all start at the start time, post-adjustment:
  `Cash(t) = Cash(start) + receipts(start..t] − disbursements(start..t]`.
  During warm-up the same invariant is checked with baseline 0 at the epoch.

### 8.5 KPIs

Daily / month-to-date / trailing 30 days:

* Net revenue (recognized), operating expenses, **operating profit**, operating
  margin, EBITDA (= operating profit in V0.1 since leases are opex).
* Cash, cash change, days cash on hand, A/R, **days in A/R**, denial rate,
  collection rate.
* Revenue per exam, cost per exam, revenue per scanner-hour, labor cost %.
* **[V0.4]** practice valuation proxy: trailing-12-month EBITDA × multiple,
  gated by quality constraints (§11 of PLAN.md).

### 8.6 Profitability presets **[calibration targets]**

| Preset | T30 operating margin | Levers |
|---|---|---|
| `struggling` | −8% to −2% | demand ×0.85, commercial share 35%, denial 10%, rent +20% |
| `break-even` | −2% to +3% | demand ×0.92, denial 8% |
| `moderate` (default) | +6% to +12% | as §8.2–8.3 |
| `high` | +18% to +25% | demand ×1.10, commercial 55%, rent −10% |

A calibration test (§ ROADMAP testing) runs 20 seeds × 60 days per preset and
asserts the band. Values in §8.2–8.3 are starting points to be tuned to hit
these bands, not the reverse.

**Single reference calculation for `moderate`** (monthly, steady state). The
P&L calibration workbook (ROADMAP §10) reproduces it. Change inputs there and
here together.

| Line | Derivation | Amount |
|---|---|---|
| Exams | ≈ 17.3 Tue–Fri × 75 + 4.3 Mon × 58 + 4.3 Sat × 26 | ≈ 1,650 |
| Base per exam | weekday mix XR 32 × $45, US 11 × $155, CT 17 × $210 (half contrast), MR 10 × $308 (40% contrast) = $9,795 / 70 | $139.9 |
| Expected net per exam | $139.9 × 1.133 | $158.5 |
| **Net revenue** | 1,650 × $158.5 | **$261.5k** |
| Payroll + 25% benefits | techs (36+40+45+44) × 173.3 h = $28.6k; desk $3.6k; ops $7.9k; radiologists $480k × (0.60+0.50) / 12 = $44.0k; × 1.25 | $105.1k |
| Fixed non-payroll | rent 22 + leases 27 + service 22.5 + utilities 7 + IT 6 + insurance 4 | $88.5k |
| Supplies, contrast, PACS | ≈ $16.8 + $3 per exam × 1,650 | $32.7k |
| Billing fee | 5% of collections (= net revenue in steady state) | $13.1k |
| **Operating profit** | | **≈ $22k (margin ≈ 8.5%)** |

Two full-time
radiologists charged 100% to this site would push the practice into losses —
which is one reason radiologist site allocation is explicit (see PLAN.md §2).

---

## 9. Observations and actions

### 9.1 Three views of the world

| View | Contains | Who sees it |
|---|---|---|
| **Truth** (`sim.State`) | everything incl. latents and the event queue | engine; debug tools behind `--debug-omniscient` |
| **Observation** (`api.Observation`) | what a competent practice manager could know right now | TUI, WebUI, CLI summaries, agents |
| **Metrics** (`api.Metrics`) | aggregate outcomes, computed from observed facts | everyone; benchmark scoring |

### 9.2 Observable vs hidden **[structural]**

| Observable | Hidden |
|---|---|
| Schedule (all booked appointments, their status) | whether a booked patient will show, arrive late, or cancel |
| Expected durations (catalog means / quantiles) | the sampled duration of a specific exam until it ends |
| Elapsed time of an exam in progress | its remaining time |
| Arrived/waiting patients and how long they've waited | a walk-in's patience |
| Unread worklist with priorities and ages | read times |
| Staff rosters, who is on duty/break/busy | future absences |
| Rooms up/down; vendor ETA (noisy) when down | true repair time; future failures |
| Claims submitted and their age; payments received | adjudication outcome and date |
| Published base rates (no-show %, payer delays) | per-entity draws |
| Demand history, lost-demand counts | future orders |
| The event *log* | the event *queue* |

### 9.3 Leakage channels and their closures

1. **Event-queue timing**: "advance to next event" must step to the next
   *observable log entry*, not the next internal event (otherwise stepping
   reveals e.g. a hidden cancellation's timing). `session.StepVisible()`.
2. **Auto-pause** triggers only on observable entries.
3. **Snapshot files** contain latents; never given to agents; benchmark runner
   rejects agents that request them.
4. **Action validation errors** must not reveal hidden facts (e.g., "cannot
   offer slot: patient will not show" is forbidden; "slot unavailable" is fine).
5. **Seeds**: benchmark test seeds are 256-bit secrets, and streams are
   SHA-256-keyed ChaCha8 (§4.3), so matching observed history against candidate
   seeds is infeasible. Agents never receive a seed. (Small integer dev seeds
   are public by design and only used for development packs.)
6. **Benchmark packs** have a public part (scenario family, horizon, action
   set, metric spec, *categories* of possible perturbations) and a private part
   (seeds, perturbation times and targets). Pack IDs and descriptions are
   neutral ("disruption-pack-03", not "ct-tech-sick-0730").
7. **Process isolation**: agents run as separate processes that talk to the
   runner only through the observation/action protocol. They have no access
   to the runner's filesystem, run records, snapshots or truth view.
8. **ID side channel**: public IDs are allocated at disclosure (§2.2), so ID
   gaps reveal nothing about undisclosed demand.
9. **Timing side channels**: observations are built at decision epochs only.
   Nothing reveals how many internal events were processed.
10. **Structural tests**: a reflection test walks `api.Observation` and fails if
   any field name matches a deny-list (`latent`, `willNoShow`, `actual`,
   `remaining`, `outcome`…); a property test checks that observations at time t
   are identical across two runs whose futures differ (same past, different seed
   for future-only streams).

### 9.4 Observation schema (excerpt)

```json
{
  "schema": "dreamteam.observation/v1",
  "time": "2026-10-13T10:42:00",
  "day": 1, "weekday": "Tue",
  "practice": {"open": true, "closesAt": "20:00"},
  "rooms": [
    {"id": "R1", "modality": "XR", "state": "occupied", "patient": "P01142",
     "exam": "XR_CHEST_2V", "startedAt": "10:38", "expectedMinutes": 10,
     "staffedBy": ["E03"], "utilizationToday": 0.78,
     "nextAppointment": "10:45", "waiting": 2}
  ],
  "staff": [
    {"id": "E03", "role": "technologist", "name": "Riley", "skills": ["XR","CT"],
     "status": "busy", "task": "exam", "shiftEnds": "16:15",
     "overtimeApprovedMinutes": 0, "weekPaidHours": 15.5}
  ],
  "patients": {
    "waiting": [{"visit":"V3310","patient":"P01151","modality":"US",
                 "appointment":"10:15","arrivedAt":"10:02","waitedMinutes":40,
                 "priority":"routine"}],
    "counts": {"bookedToday":71,"arrived":38,"walkIns":4,"completed":34,
               "noShows":2,"cancelledToday":3}
  },
  "worklist": {"unread": 12, "urgent": 2, "oldestMinutes": 192,
               "items": [{"study":"S2201","modality":"CT","priority":"urgent",
                          "ageMinutes":21}]},
  "finance": {"cashCents": 14285000, "revenueTodayCents": 1194000,
              "expenseTodayCents": 861500, "arCents": 41230000},
  "policies": {"walkIn":"accept_if_gap","worklist":"urgent_then_fifo"},
  "alerts": [{"code":"room_behind","room":"R2","minutes":15}],
  "recentLog": [{"at":"10:42","text":"Walk-in P01163 requests XR extremity"}]
}
```

Time in observations is formatted practice-local ISO-8601 without zone; money
is integer cents; fractions are floats (display only — never fed back into the
engine).

### 9.5 Action schema

```json
{"type": "extend_shift", "employee": "E05", "minutes": 60}
{"type": "set_walkin_policy", "policy": "accept_if_gap"}
{"type": "set_worklist_policy", "policy": "urgent_then_fifo"}
```

Response:

```json
{"ok": false, "code": "exceeds_max_shift", "message": "Shift would exceed 12 h."}
```

Action catalog by version:

| Version | Actions |
|---|---|
| V0.1 | `extend_shift`, `set_walkin_policy`, `set_worklist_policy` |
| V0.2 | `reassign_technologist`, `call_in_staff`, `send_home`, `approve_overtime_week`, `edit_template`, `reserve_addon_slots`, `set_overbooking`, `set_late_policy`, `reschedule_appointment`, `offer_earlier_slot`, `schedule_maintenance` |
| V0.3 | `set_denial_rework`, `set_self_pay_policy`, `set_billing_priority` |
| V0.2 (roster) | `change_roster` (weekly roster editor, ROADMAP V0.2) |
| V0.4 | `hire`, `terminate`, `lease_equipment`, `extend_hours`, `referral_outreach` |

Validation is in `sim` and is the only gatekeeper — TUI, CLI, Web and agents
call the same `Apply`. Each action has: schema validation, permission
(is it allowed in this scenario/version), feasibility (hard constraints §6.5),
and an effect. Rejected actions are logged (they matter for agent evaluation)
but do not change state.

Time controls (pause, speed, step) are **not** actions: they never change the
simulated world.

---

## 10. Invariants

Checked after every timestamp transaction (§3.4) in tests and with `--check-invariants`
(off by default in play for speed; on in CI and benchmarks).

**Time and queue**
1. Clock is non-decreasing; every queued event has `At ≥ Now`.
2. Every non-stale event references an existing entity in a compatible state.

**Resources**
3. Each room has ≤ 1 occupying study; `Occupied ⇔ Current ≠ nil`.
4. Each employee has ≤ 1 task; `Busy ⇔ Task ≠ nil`.
5. Every exam **started** with a technologist holding the room's modality skill
   who was on duty or on approved overtime and not on break. Run-over past
   shift end only happens to tasks already in progress (§6.4).
6. Every contrast exam started with a radiologist on duty and on site (or
   virtual supervision enabled).
7. No exam started outside open hours.
8. No shift exceeds `MaxShiftHours` from shift start, run-over included.

**Patients**
9. Each visit is in exactly one state; transitions follow the allowed graph.
10. Conservation per day: `booked = arrived + missed + noShow + cancelled +
    pending`, where `missed` is non-zero only between start + grace and
    accounting close; each appointment has at most one visit.
11. Timestamps are ordered: arrival ≤ reg start ≤ reg end ≤ exam start ≤ exam end.
12. A study exists iff its visit reached `Departed` after an exam; a report is
    final only after acquisition; a claim exists only for a final report.

**Money**
13. Ledger balances: Σ debits = Σ credits, per entry and overall.
14. `Cash(t) = Cash(start) + Σ receipts − Σ disbursements` since the start
    time (baseline per §8.4; warm-up uses its own baseline).
15. A/R = Σ over open claims of (expected net − paid). Closed claims have zero
    balance. A/R by claim may go negative only between an overpayment and its
    settlement.
16. Revenue is recognized exactly once per finalized study, and each claim
    settles exactly once.

**Metrics**
17. For every room/day: busy-in-staffed ≤ staffed time ≤ open time, and
    busy-in-open ≤ open time (§11).

---

## 11. Metrics (computed from observed facts)

| Dimension | Metrics |
|---|---|
| Financial | net revenue, operating expense, operating profit & margin, cash, days in A/R, cost per exam |
| Productivity | exams by modality, room utilization (vs open hours **and** vs staffed hours), revenue per scanner-hour, tech busy % |
| Access | lead time from **original order** to exam (across reschedules) by modality, **lost demand**, walk-in left-without-exam rate, same-day add-on acceptance, orders still unserved at window end with their age |
| Patient experience | wait from **arrival** to exam start (median, p90), late starts, waits > 30 min |
| Clinical service | report TAT median/p90 by priority and modality **over all studies acquired in the window, with unfinished ones counted at their age at window end (censored)**; urgent TAT breaches including still-unread overdue studies; unread backlog at close; overdue-outstanding count |
| Workforce | overtime hours & cost, skipped breaks, desk-unstaffed patient-minutes, max weekly hours |
| Resilience **[V0.2]** | time to recover schedule after disruption, patients rescheduled due to disruption |

Utilization is reported two ways because they tell different stories.
**Open-hours utilization** exposes the staffing gap. **Staffed-hours
utilization** exposes scheduling inefficiency. Definitions are interval-based,
per room and day:

* `open` = seconds the practice is open.
* `staffed` = seconds within `open` the room is in `Idle`, `Occupied` or
  `Turnover` (a qualified covering tech is on duty; shared coverage counts
  fully for each covered room).
* `busy` = seconds the room is `Occupied` or `Turnover`.
* Open-hours utilization = busy∩open / open. Staffed utilization =
  busy∩staffed / staffed. Busy time outside open hours (run-over after close)
  is reported separately as after-hours busy.

Upper bounds follow from the roster. For example, MR staffed 8 of 12 open
hours caps MR open-hours utilization at 66.7%. Targets below that cap are
calibration results, not guarantees.

**Anti-gaming in benchmarks.** Packs add an **evaluation tail**: after the
agent's horizon, the default policy runs N more days (default 7) and the
service metrics include the tail. Orders still unserved, studies unread and
patients pushed past the horizon count against the agent at their age. Cohort
metrics (share of studies or orders breaching target, unfinished included)
complement percentiles, so sacrificing a 10% tail can't hide inside a p90.
