# DreamTeam — Progress

One section per roadmap step (ROADMAP §2), newest first. Each records what
was built, what the player can see, how it was tested, and known
limitations. The version shown in the TUI top bar (`DREAMTEAM v0.1.0-dev.N`)
and by `dreamteam version` names the latest step a build contains.

---

## V0.2 dev.6 — pre-Slice 5 remediation (0.2.0-dev.6)

An independent read-only audit of 13987e6 (0.2.0-dev.5) returned
CONDITIONAL GO, with two verified scheduler defects and an allocation
arbitrage that would distort Slice 5's comparison of internal reading
against outsourced reading. This step fixes those three and nothing else:
`contract_teleradiology` is not built, and demand, reimbursement, wages,
recruiting, read durations, service gates and the frozen plans are
unchanged. Design and as-built notes: `docs/V0.2_DESIGN.md` §5.11.

* **Fix 1: touching reading blocks.** At 100% the cycle's blocks touch.
  The next block's start (class 1) ran before the previous block's end
  (class 5) at the same instant, so the end closed reading and the
  radiologist claimed nothing for the rest of the shift.
  * `blockEnd` now continues reading when the day's next block starts
    exactly where this one ends.
  * The blocks, their sum and their boundaries are unchanged, so no
    capacity is added.
  * Reads stay nonpreemptive. Worklist order and the shift-end claim rule
    are unchanged.
* **Fix 2: overrun debt from time actually spent outside blocks.** dev.5
  charged the overrun only if a read ended outside a block, so a read
  running through a whole other-duty gap owed nothing. This affected
  allocations above 75%; the scenario and plan D allocations were not
  affected.
  * When a block opens while its radiologist is still reading, the gap
    since the last block ended is owed at once.
  * Repayment starts when the read ends, out of block time. What the
    block cannot cover carries forward as before, and debt lapses at the
    shift's end as since V0.1.
* **Fix 3: minimum allocation commitment (the Sunday arbitrage).** A
  version at 30% from a Sunday, restored on Monday, kept every shift's
  reading and cut the Sunday salary by $328.77 for Dr. Blake (about $904
  a Sunday for both radiologists at plan D's shares).
  * An allocation now stays in force at least 14 calendar days
    (`MinCommitDays`) before the next change for that radiologist.
  * The check runs on the timeline the whole batch leaves, after every
    item's own checks, so item order doesn't matter.
  * The refusal is `commitment_too_short` on the first item that makes
    the short pair; the rest of the batch is `batch_rejected` and nothing
    changes. Previews agree.
  * Replacing a pending version's value passes. Under this rule a
    withdrawal can only lengthen a pair.
  * Pairs already on an earlier build's timeline are grandfathered, but a
    withdrawal that would create a new short pair is refused.
  * Notice, effective dates, horizon, bounds, history, no-op rules and
    atomic mixed batches are unchanged. No new charge or account.
* **Tests (fail-before-fix).** All seven new tests fail with the fixes
  disabled and the tests unchanged.
  * `TestTouchingBlocksReadContinuously`: 4 and 5 claims instead of 18,
    for 30- and 20-minute reads.
  * `TestOverrunThroughOtherDutiesIsOwed` (95%, a read claimed at cycle
    minute 94 crossing 95–100, reads ending exactly on a block start or
    end): failed at the dev.5 claim minutes.
  * `TestOverrunThroughSeveralGapsIsRepaidLater` (30%, a 205-minute read
    crossing two gaps): next claim at 11:25 instead of 18:25.
  * `TestReadingTimeConservation`: 12, 12, 17 and 428 claims while
    owing.
  * The three tests in `allocation_commit_test.go`: the arbitrage, 13-
    against 14-day spacing and the earlier-timeline rules were all
    accepted.

  The micro fixture is `internal/sim/testdata/micro-blocks.json`: a
  saturated worklist across seven boundaries.
* **Conservation.** `TestReadingTimeConservation` keeps an independent
  account of every radiologist-day, built from the studies' claim and
  final times and the day's plan. It requires:
  * every minute outside a block borrowed once;
  * repayment, in order, only from later block time spent not reading;
  * no claim while time is owed or outside a block;
  * what is still owed at the shift's end lapses.

  Six weeks of the established practice at 100%/95%: 303,438 s borrowed
  = 297,557 repaid + 5,881 lapsed, with no claim while owing. Under
  dev.5's rules there were 428 claims while 2,790 minutes were owed.
* **Compatibility.** The fixes change outcomes, so this is not a digest
  migration.
  * Every record from an earlier build replays under that build's rules
    (`SetDev5Rules`, set by `session.Replay` until the record's end).
  * 0.2.0-dev.5 is registered as batch mode `v0.2_slice4`, with every
    digest verified.
  * There are two genuine dev.5 records from a clean 13987e6 build
    (`testdata/sessions/mini-seed24-dev5`, `mini-seed25-dev5-reading`).
    Unregistered, both were refused. Registered without dev.5's rules,
    the first fails at batch 2, and the second's log and run digests
    differ.
  * Both replay whole and chunked, and continue as a verbatim prefix
    under this build's rules.
  * A record claiming dev.5 but needing dev.6 rules is refused.
  * Unknown actions, v1 and v2 records, and dev.2–dev.4 records behave
    as before.
  * One existing session test moved a change from Nov 9 to Nov 10, from
    13 days apart to 14.
* **Neutrality.**
  * All seven golden references reproduce. MIGRATIONS.json lists the
    Slice 4 reference, now historical, with no changed field.
  * Plans A–D rerun under dev.6 reproduce `results/reading-dev` and
    `results/reading-eval` in every field but the engine name.
  * All nine session files (the benchmark's dev.4 and dev.5 files and
    the testdata records) verify as `earlier_build_exact`.
  * No plan was retuned and no seed was used to choose anything.
* **Verification.**
  * Full uncached suite on the Pi passes: sim 544 s, session 111 s, cmd
    87 s, tui 24 s.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * Plan session replay, A–D × seeds 901 and 1001 × 180 days, whole and
    in 13-hour chunks: 171 s on joofi.
  * Acceptance on joofi passes: staffing and roster invariants (50 seeds
    × 90 days each, after every transaction), turnover, churn, V0.1
    invariants and 1,000 chunkings. The first run hit a 120-minute
    timeout I had set (Slice 3 needed 128 min), during the V0.1 criteria;
    those were then rerun alone and passed in 42 min.
  * Performance (dev.5 against dev.6, interleaved, joofi): Mini180 917
    against 937 ms/op and EstablishedPlay90 376 against 386 ms/op
    (medians), inside a ≈15% spread. Allocations are identical. Details
    are in MEASUREMENTS.
* **Limitations.**
  * Debt still lapses at shift end, as in V0.1, now measured: 98 minutes
    in six weeks at 100%/95%.
  * The commitment is per radiologist only and does not price early
    exit.
  * The service-gate, backlog-rule and turnaround audit findings (Medium
    and below) are open, for Slice 5's pre-registration.
* **Next: Slice 5, `contract_teleradiology`.** This needs persistent
  remote reading ownership, service and solvency rules set before any
  strategy, liquidity as an independent constraint, and a fresh locked
  holdout. Seeds 1001–1020 stay as reference data.

---

## V0.2 Slice 4 — reading allocation (0.2.0-dev.5)

The owner approved `set_reading_allocation` after the dynamic staffing
benchmark, and nothing else (`contract_teleradiology` comes later).
Design and as-built notes: `docs/V0.2_DESIGN.md` §5.10. Benchmark:
`docs/calibration/dynamic-staffing/` README, "Plan D". No reimbursement,
demand, wage, reading-duration, equipment or opening-hour input changed.

* **Action.** `set_reading_allocation` changes an employed salaried
  radiologist's site share of paid time.
  * **Ops:** `op: set` / `remove`, `employee`, `effective`,
    `siteAllocationBP` 3,000–10,000.
  * **Dates:** any weekday at least 14 days ahead, within the booking
    horizon. Removal needs the same notice.
  * **Versions:** dated, like roster versions. A pending version can be
    replaced or withdrawn; versions in force or past never change; an
    append-only log keeps every operation. A set equal to the allocation
    in force, or the removal of an absent version, is `unchanged`; two
    writes to one radiologist and date in a batch are
    `invalid_parameters`.
  * **Distinct from contracts and rosters:** radiologists have no
    contract hours, and their shifts stay immutable.
* **Reading and supervision.**
  * The allocation in force each day sets that day's site reading blocks
    through the existing 100-minute cycle inside the same shifts.
    Worklist order, overrun debt, one read at a time, the p100-by-shift-
    end claim rule and origin-keyed read times are untouched.
  * Contrast supervision is presence on site and does not move (tested).
  * Booking coverage segments were also cut at reading-block boundaries,
    which allow shared-capacity handovers. They now keep the scenario
    allocation's boundaries (`DayPlan.cutReading`), so an allocation
    never changes technologist booking capacity and displaces nothing.
    The test that fails without this covers days built after the change
    and days a later hire rebuilds.
* **Money.** The existing daily salary line uses the allocation in force
  that day: daily rounding, benefits, and payday settlement through
  accrued payroll. One accrual per radiologist per day, by its key, so no
  time is charged twice. No new account or entry kind.
  * Tested to the cent across two changes, with the cash bridge
    reconciled.
  * Previews show the salary run-rate change per 30 days as a guaranteed
    obligation, with the date until which it can be withdrawn at no
    cost.
* **Batches and compatibility.**
  * The new type mixes atomically with `hire`, `terminate` and
    `set_roster`; any invalid item refuses the whole batch.
  * A `mixed_batch` message names `set_reading_allocation` only when the
    batch holds one, so every recorded message and golden is unchanged.
  * Allocation versions are encoded in the state digest only when
    present, so no-action state digests are unchanged too.
  * 0.2.0-dev.3 and dev.4 records replay with Slice 3's action set (batch
    mode `v0.2_slice3`). A genuine dev.4 record
    (`testdata/sessions/mini-seed23-dev4.*`, from a clean af475fd
    worktree) has the new type refused as `unknown_action` and
    `mixed_batch`. It failed before registration, and failed again under
    this build's action set; it now replays with every digest verified
    and continues as a prefix. The benchmark's dev.4 session files verify
    as `earlier_build_exact`.
* **Interface.**
  * **API:** `Employee.SiteBP` is the allocation in effect;
    `Employee.AllocationChanges` lists pending ones.
    `World/Session.ReadingAllocations()` gives each radiologist's
    timeline with site reading minutes per rota week and salary per 30
    days. `StaffingPreview.Allocations` gives before and after, the
    withdrawal date and the reporting backlog.
  * **CLI:** action files take the new type; golden
    `established-seed1-60d-allocation.json`.
  * **Session v2.**
  * **TUI:** staff screen rows per radiologist (←→ share in 2.5-point
    steps, `,` `.` date, `x` withdraw), the backlog line and a read-only
    preview.
* **Tests (fail-before-fix).** Nine `internal/sim/allocation_test.go`
  tests failed before the action existed. They cover:
  * date boundaries; bounds and shape; versions;
  * salary proration; supervision and booking invariance;
  * reading rules with urgent order; mixed batches; chunking; previews.

  Session tests cover save and replay across effective dates (whole,
  chunked, event-budgeted), a v1 record with the type name and the dev.4
  record. There are also a TUI test and the new golden.
* **Plan D benchmark (20 held-out seeds 1001–1020, 180 days).** D is plan
  C's staffing plus Dr. Avery 62.5% and Dr. Blake 52.5% from Tue Oct 27,
  the earliest legal date. D and its decision rule were fixed before any
  run.
  * **Reproduction:** A, B and C reproduce the original results exactly.
  * **D against C:** +$13.7k radiologist salary, +74 ± 20 reports, $12.5k
    of backlog revenue recovered. Profit −1.5 ± 3.2 $k at T180 and +0.6 ±
    1.2 $k per 30 days in steady state. Imaging and access are identical.
  * **Breach rates days 91–180 (urgent / soon / routine):** D 9.2 / 31.9
    / 4.4%, C 52.7 / 88.2 / 55.3%, no action 10.7 / 38.1 / 7.2%. The
    gates are met on the mean, but on all three gates only on 7/20 (T90)
    and 11/20 (days 91–180) seeds.
  * **Against no action:** +12.6 ± 1.4 $k at T180, break-even day 141
    (113–164) on 20/20 seeds, cash payback by T180 on 1/20.
  * **Decision: CONDITIONAL GO**, by the pre-registered rule. Its one
    failing test compares single backlog samples on day 90 and day 180;
    no action fails it too, and D's 30-day mean backlog is flat.
    Conditions:
    * the gates are met on average only;
    * the soon rate is at its gate;
    * the allocation buys service at about its own cost.

    No other allocation was tried on the held-out seeds.
* **Verification.**
  * The full uncached suite passed on the Pi in 9 min 1 s (sim 537 s,
    session 94 s, cmd 84 s, tui 27 s). An earlier run hit Go's default
    10-minute timeout in `sim` while the Pi was also compiling; rerun with
    `-timeout 60m` it passed.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * All seven golden references reproduce: the six historical ones under
    `MIGRATIONS.json` as before, the new one byte for byte.
  * `dreamteam verify` replays the three dev.4 benchmark sessions
    (`earlier_build_exact`) and D's.
  * Every plan replays whole and in 13-hour chunks (seeds 901 and 1001,
    180 days).
* **Limitations.**
  * One allocation was tested.
  * Gate pass rates per seed are modest.
  * The rule's backlog test uses point samples.
  * No recruiting uncertainty; ×1.30 demand only; 180 days; arm64 only.
  * Teleradiology, the alternative for marginal and evening reads, is not
    built yet.
* **Next: Slice 5, `contract_teleradiology`.** Then test substitution
  against allocation, on development seeds first.

---

## V0.2 dynamic staffing benchmark (0.2.0-dev.4)

The owner's milestone after Slice 3: can the sustainable scenario's shared
start be improved with the implemented actions alone, and at what
transition cost and payback? Benchmark: `docs/calibration/dynamic-staffing/`
(README, plans, per-seed results, reports, trajectories, sessions). Design:
`docs/V0.2_DESIGN.md` §5.9. **No engine behaviour, economics, demand,
wages, hours or patient behaviour changed, and no action was added.**

* **Plans.** `plans.py` writes them as fixed timed action batches, applied
  through `World.Apply` identically on every seed. Each submits one atomic
  batch at the takeover with standard offers; V0.1 controls stay at their
  defaults.
  * **A, no action.**
  * **B, partial:** convert the MR evening. `hire` a 20 h 16:00–20:00 MR
    technologist and `terminate` Alex (40 h) once it is productive.
  * **C, full:** the closest legal path to Step 0C's `DED-MRpt`. It hires a
    CT day core (40 h), an XR/CT evening (27.5 h) and an MR evening (20 h);
    terminates Sam and Alex; and moves Riley to X-ray only with `set_roster`
    from Monday Nov 23.
  * **Feasibility:** every step is legal (lead, training, notice,
    headcount 8 of 8, Monday effective date, no coverage gap). Only Step
    0C's reading allocation awaits Slice 4.
* **No overfitting.**
  * B was chosen among three candidates (MR conversion, Step 0C's XR/CT
    float, both) on development seeds 901–910. The rule was written into
    `plans.py` before any run: the best T180 increment with the urgent
    breach rate at most 5 points worse. The float candidates failed that
    rule.
  * The evaluation used 20 held-out seeds, 1001–1020, never used before
    (Steps 0–0C used 1–20), for 180 days.
* **Validation, every seed:** all actions were accepted; every lifecycle
  date matched the plan; at most 8 technologists; no inconsistent booking
  at any endpoint; the cash bridge reconciled; labor postings summed to
  the window's labor to the cent.
  * C displaced one CT contrast appointment on 2 of 20 seeds, rebooked the
    same day: a break-relief cover was needed in X-ray.
  * Each plan replays authoritatively from a saved session, whole and in
    chunks, to the same digests.
* **Results (20 held-out seeds, Δ against the paired no-action run).**
  * **B:** costs $9.8k (recruiting 3.0, severance 4.5, training 2.2). Δ
    profit at T30 / T90 / T180 is −8.0 / −2.0 / **+12.9 ± 0.1** $k, and
    +$5.0k a month in steady state. Break-even is **day 101** (100–106) on
    20/20 seeds; cash pays back on 20/20 (median day 114). Service is
    unchanged.
  * **C:** costs $23.7k (recruiting 9.0, severance 8.4, training 6.2). Δ
    profit at T30 / T90 / T180 is −19.0 / −10.4 / **+14.1 ± 2.7** $k, and
    +$8.2k a month. Break-even is **day 134** (113–175) on 20/20 seeds;
    cash pays back on 11/20 by T180. Lowest cash is $20.7k.
  * **C, days 91–180:** imaging +265 exams, lost referrals −179, walk-ins
    leaving −90. But reading cannot keep up:
    * urgent, soon and routine breach rates are 52.7%, 88.2% and 55.3%,
      against 10.7%, 38.1% and 7.2% with no action;
    * the backlog is 93 against 30 and still growing.
  * **Guaranteed pay** is negligible in both plans. **Unrecovered demand:**
    US "not rostered" losses (≈405 per 90 days) are untouched by every
    plan.
* **Decision: CONDITIONAL GO.**
  * Staffing actions give measurable agency with credible payback;
    transitions are complete by day 42 and do not call for a different
    starting state.
  * The full reorganization's service needs `set_reading_allocation`.
  * Next: Slice 4, then rerun the same frozen plans on the same held-out
    seeds with C plus an explicitly defined reading adjustment.
* **Code (test and session compatibility only).**
  * `internal/sim/calib_dynamic_test.go` is the harness (build tag
    `calibration`). It reads `benchmark.json` and `plans/`, samples daily,
    measures endpoints and spans, decomposes labor, and checks
    lifecycles. `calib_export_test.go` is a read-only ledger export for
    tests.
  * `internal/session/calib_dynamic_test.go` saves and replays every plan.
  * `report.py` builds the paired tables, payback and `trajectory.csv`.
* **Version and compatibility.** 0.2.0-dev.4 registers 0.2.0-dev.3 as an
  earlier V0.2 build: batch mode `v0.2_slice3`, the same action set.
  * A genuine dev.3 session file, `testdata/sessions/mini-seed22-slice3.*`,
    built from a clean 5fdfb95 worktree, replays with every digest and
    continues as a prefix (`TestSlice3RecordReplaysAndContinues`). It
    failed as an engine mismatch before the registration (fail-before-fix).
  * The Slice 3 golden is listed in `MIGRATIONS.json` with no changed
    field.
* **Verification.**
  * The full uncached suite passed on the Pi in 9 min 6 s; `go vet` is
    clean for the default, `acceptance` and `calibration` tags; gofmt is
    clean.
  * Benchmark results from the dev.3 and dev.4 binaries, on the Pi and on
    the Cortex-A76, are identical in every field but the engine name.
* **Limitations.**
  * One hand-authored plan per strategy, not optimized; premium offers and
    US coverage untested.
  * No recruiting uncertainty in the model.
  * Demand fixed at ×1.30 (×1.15 not rerun).
  * 180-day horizon.
  * Accrual expenses fees and severance at acceptance, and revenue waits
    for the report.
  * The provisional service gates are missed by every staffing-only plan,
    including no action on soon and routine.
  * amd64 not run.

---

## V0.2 Slice 3 — roster changes (0.2.0-dev.3)

The owner approved `set_roster` (2026-10-15). `set_reading_allocation` and
`contract_teleradiology` are not in this slice. Design: `docs/V0.2_DESIGN.md`
§5.8, with the deviations identified there. Without a `set_roster` action
every earlier outcome is unchanged.

* **Roster versions.**
  * An hourly technologist's or front desk employee's roster is now a
    timeline: the original (scenario or hire, immutable) and dated versions.
    Each version is a full A/B pattern from a Monday, in force until the
    next.
  * `op: set` inserts a version or replaces a pending one; `op: remove`
    withdraws a pending one, and its predecessor resumes. Several
    successive versions are allowed.
  * Versions in effect or past never change. Every operation, including
    replaced, withdrawn and voided versions, stays in the employee's roster
    log.
  * The version for a date is resolved behind the per-day registry view
    that the plan, the runtime and the booking predicates already use.
* **Validation** (in order): shape; employee (hourly technologist or front
  desk, still employed); effective date (`past_date`, `not_monday`, 14 days'
  notice, horizon, before the end of employment); each shift as for a hire
  (rooms of the employee's skills, desk for the front desk, breaks, cap);
  `unchanged` when equal to the roster already in force; then the
  whole-roster rules — at most six days a rota week, **net rostered time
  ≤ the contract** in each rota week (the employee's own original roster
  may always be restored), rest. Then the batch's composed timelines are
  checked as a whole:
  * no version after a termination (in either batch order);
  * rest across every version boundary, withdrawals included;
  * a trainee still completing training within 60 days
    (`training_incomplete`);
  * every approved extension still valid on its new shift
    (`extension_conflict`, naming the date: refused, never silently
    cancelled).
* **Rest rule, one for hiring, rosters and overtime:** 10 hours from a
  shift's cap — the latest permitted release, min(start + 12 h,
  23:59:59) — to the next day's start. Approved extensions and hold-over
  cannot pass the cap, so no approval can break the rule. Tested at the
  day boundary within a pattern (A→B and B→A), across versions, after a
  withdrawal, and at exactly 10 hours.
* **Contract and money.** Contracted hours, rate, accrual weights, notice
  and severance never change with the roster; nothing new is posted. A full
  week under a four-day or a zero-hour roster pays Riley's contract exactly
  ($1,440 + benefits); severance is identical before and after a cut.
* **Atomic batches and displacement.** A batch commits in input order,
  moves trainees' productive dates, then reconciles once, against the final
  timeline, from the earliest affected day: the same survivor and
  re-offer lifecycle as Slice 2. Causes: `employment_ended`,
  `roster_changed`, `training_delayed` (joined by `+`). A coordinated
  exchange (an XR+CT hire and Sam swapping Monday and Tuesday) displaces
  nobody in either order, while each half alone displaces 11 bookings.
  A termination voids pending versions on or after the end (logged), as
  it voids extensions (unchanged).
* **Training.** A roster change before the first working day or during
  training recomputes the productive date from the composed timeline (past
  days are the attended ones). A delay displaces the bookings that relied on
  the trainee (including XR bookings losing a shared technologist); an
  earlier completion opens capacity. Capacity is never granted before
  completion (invariant).
* **Preview.** Rostered hours before and after against the unchanged
  contract, the interval, a trainee's productive date before and after, the
  extensions kept, the batch's slots opened and closed by modality, exact
  displacements with provisional replacements, and the guaranteed weekly
  contract cost apart from forecasts (contracted time not rostered,
  extensions kept, displaced bookings). Pure and independent of the hidden
  future (tested).
* **Interfaces.** `api.Action` gains `op` and `effective`;
  `api.Employee.RosterChanges`; `World/Session.RosterPlan`; CLI actions
  files; session v2; the TUI roster editor (staff screen `m`, enter on a
  roster row): weekly or fortnightly grid, field editing, effective Monday,
  withdrawal, live preview.
* **Compatibility.**
  * State digest still `dreamteam.state/v3`: versions are encoded only when
    present.
  * A genuine 0.2.0-dev.2 save, built with that commit's binary and kept
    in `testdata/sessions`, replays under this build with every digest
    equal. Its batches apply with Slice 2's action set (`ApplySlice2`),
    where `set_roster` was an unknown action. Before the version bump the
    same file failed replay (fail-before-fix). Continued, it becomes the
    prefix of a new v2 record
    (`prefix:earlier_build_exact+native_exact`).
  * v1 → v2 continuation with roster changes replays.
  * The four V0.1 goldens are unchanged under the manifest. The Slice 2
    golden differs only in its action and run digests: its mixed batch's
    refusal message now names `set_roster`. This is recorded in
    `MIGRATIONS.json` and confirmed by diffing the dev.2 and dev.3 action
    records (log, ledger and state identical).
  * A sixth reference, `established-seed1-60d-roster.json`, pins the roster
    plan (36 displacements, all rebooked).
* Verification:
  * Full uncached suite (`go test -count=1 -timeout 60m ./...`) passes on
    the Pi in 8 min 19 s: cmd 86 s, session 88.5 s, sim 488 s, tui 24 s.
    `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * Acceptance (`-tags acceptance -run Acceptance`) passes on joofi in
    128 min: the V0.1 criteria, staffing turnover (155 hires over 720
    days) and roster churn (340 roster versions accepted over 720 days),
    with the invariants checked daily; 156 tests and subtests, no failure.
  * All six golden references and the Slice 2 session fixture give
    byte-identical outputs on the Pi and on joofi.
  * Fail-before-fix mutations: all 20 guards were caught (not_monday,
    notice, contract, pattern rest, boundary rest, extension conflict,
    extension on the composed view, training projection, trainee capacity
    early, void on termination, composed not_employed, duplicate date
    claim, unchanged, version resolution, reconcile after roster,
    per-item reconcile, guarantee after a roster cut, digest omit-empty,
    Slice 2 batch mode, original-roster restore only when exact).
* Performance (Pi, `docs/MEASUREMENTS.md`): the 90-day run is unchanged
  (best of 8 interleaved: 3.40 s dev.2, 3.41 s dev.3); EstablishedPlay90
  allocations unchanged (78.6 MB, 1.241M). A roster preview costs
  28.2–28.3 ms, a commit 25.3–25.7 ms; the editor renders in 48 ms.
* Limitations: the rest rule is conservative (measured from the cap, see
  deviation 1); break rules are not applied to approved extensions;
  radiologist and supervision displacement remain untestable until the
  reading actions; the result schema is still v1; amd64 and macOS were
  not run; coverage differences are reported per batch, not per change;
  the TUI edits only the first cover of a shift.

---

## V0.2 Slice 2 — hiring and termination (0.2.0-dev.2)

The owner approved two player actions, `hire` and `terminate` (2026-10-15).
`set_roster`, `set_reading_allocation` and `contract_teleradiology` are not
in this slice. Design: `docs/V0.2_DESIGN.md` §5.7, with the deviations
identified there. Without a V0.2 action every V0.1 outcome is unchanged.

* **Registry and identity.**
  * Hires get the first unused `H00001`, `H00002`, … (scenario IDs skipped),
    allocated at commit only.
  * A hire's terms are frozen in state; records are append-only and never
    reused.
  * Staffing sets are growable (no 64-employee mask); the scenario limit is
    a 1,000-employee sanity bound.
  * The state digest encoding is `dreamteam.state/v3`.
* **Hire.**
  * Validated in a fixed order: offer table (role, skills, exact standard or
    premium rate), contract 1–2,400 minutes, start ≥ today + the tier's lead,
    roster rules, 40 h (technologist) or 24 h (desk) of training within 60
    days, and the 8-technologist / 4-desk caps including notice.
  * The recruiting fee posts at acceptance (recruiting / A/P).
  * From the start date the employee is paid (contract guarantee in equal
    sevenths, training as worked time) but works under a training view with
    no capacity: no room, registration, supervision, hold-over or extension.
  * Independent from the midnight after the 40th (24th) attended hour.
    Future slots open from that date.
* **Terminate.**
  * Hourly technologists and front desk only.
  * Last day ≥ 14 days after today and the start; identical repeats are
    `unchanged`; a moved or withdrawn notice is refused.
  * Contract pay continues to the last day, whatever the roster; the last
    week closes on the last day.
  * Severance (two contracted weeks with benefits) is recognized at
    acceptance (severance / severance payable) and paid on the first payday
    after the last day.
  * Approved extensions after the end are voided with a log line.
  * Terminating a pending hire keeps the fee, runs the contract from the
    start and owes severance.
* **Displacement** (when the batch commits; design §5.7):
  * The days after an employee's end are re-planned from the final registry
    (a replacement hired in the same batch counts).
  * Bookings are re-inserted in survivor order with the booking rule's own
    predicates; those that cannot stay are displaced, all released first,
    then offered a same-category slot no earlier than their original time,
    accepted within the patient's tolerance of the original tolerance
    anchor, else lost to displacement.
  * The behavioral attempt, confirmation and no-show lead basis are kept, so
    no new random draw is made.
  * Each displacement has a record and its own counters, outside demand,
    loss, cancellation and no-show figures.
* **Atomic batches.** A batch with a V0.2 action holds only V0.2 actions
  (`mixed_batch`); one refusal refuses all (`batch_rejected`) and commits
  nothing; a valid batch commits in input order and reconciles once. V0.1
  batches are unchanged.
* **Finance.** New accounts `recruiting`, `severance`, `severance_payable`
  (listed only once they hold a balance, so the baseline digest is
  unchanged); statement lines; labor includes severance; the cash bridge
  and cash identity include severance; severance due appears in
  obligations.
* **Previews.** `PreviewStaffing`: validation, terms, timeline, slots a hire
  opens, exact potential displacements with capacity-only provisional
  replacements, guaranteed and forecast money in separate tables. Pure.
* **Session v2.** `dreamteam.session/v2`; v1 read by separate strict DTOs;
  an engine registry (`0.1.*`, `0.2.0-dev.0.*`, `0.2.0-dev.1`) replays v1
  records with outcomes verified; a continued v1 session saves as v2 with
  the v1 record embedded and this build's transition digests.
* **Interfaces.** API (`api.Action` fields, `ValidateBatch`,
  `PreviewStaffing`), CLI actions files and replay, TUI staff screen `m`.
* **Compatibility and neutrality.**
  * The four historical goldens are byte-for-byte unchanged. They still differ
    from this build in exactly `digests.state`, `digests.run` and
    `replay.runDigest`; every outcome field, the log, ledger and action
    digests and the baseline digest are identical (90 days and the V0.1
    action fixture included). `MIGRATIONS.json` now records this build's
    new values (`dreamteam.state/v3`) with the explanation.
  * A fifth reference, `established-seed1-60d-staffing.json` (V0.2 actions,
    written by this build), pins hire and terminate end to end.
  * The Step 0C sustainable start and lead candidate reproduce their
    committed per-seed and aggregate results exactly (9 seeds each).
  * All five references are byte-identical on the Pi 4 (Cortex-A72) and the
    Cortex-A76 machine. amd64 and macOS not run (no machine).
* **Regression tests.** New: `staffing_test.go` (lifecycle and training
  capacity, the §5.4 partial-week example to the cent, notice and severance
  with a zero roster, every rejection code, repeats and conflicts, atomic
  batches, headcount caps with notice, displacement records and preview
  agreement, a replacement in the same batch displacing nobody,
  shared-technologist displacement, tolerance anchor over 12
  terminations, read-only and hidden-future-independent previews, more than
  64 employees, the evaluation endpoint), `empset_test.go`,
  session `staffing_test.go` (save/replay at every lifecycle boundary,
  chunked and event-budgeted replay, v1 → v2 continuation and tampering),
  TUI `staff_test.go`, and two acceptance-tagged checks (invariants after
  every transaction for 50 seeds with the reference plan; 720 days of
  turnover past 64 lifetime employees, reproducible).
  * **Fail-before-fix evidence by mutation**: each guard was disabled in
    turn and its test run; all 17 fail as required — trainee capacity,
    guarantee from the start date, severance posting, severance payday,
    notice pay, reconciliation, attempt preserved, tolerance anchor (at the
    call site and in the rule), shared-technologist survivors, batch
    atomicity, notice in the cap, training required, v1 transition check,
    preview purity, sets past 64, v1 replay semantics for V0.2 type names. The first anchor test did not catch a
    more lenient rule; `TestDisplacementToleranceAnchor` and
    `TestAcceptReplacementAnchor` were added and do.
* **Remaining limitations.**
  * No `set_roster`: a hire's roster is fixed at hire; contracts change only
    by termination and hiring (owner decision).
  * Contrast-supervision displacement cannot arise yet (radiologists cannot
    be terminated); the path is shared with the tested staffing check.
  * The result schema stays `dreamteam.result/v1` with additive fields;
    the T90 vector and gates are not built.
  * Hire names are their IDs. The TUI offers fixed roster patterns, not a
    roster editor.
  * amd64/macOS determinism not yet run.
* **Performance** (MEASUREMENTS.md): about +5% CPU and +7–12% memory on
  a 90-day run (best of ten on a loaded, throttled Pi: 3.20 → 3.49 s); a
  termination preview ≈21–24 ms and its commit ≈30 ms; the staff screen
  renders in ≈25 ms with a live preview.
* **Verification:**
  * Full uncached `go test -count=1 -timeout 30m ./...` passed: 7 packages,
    8.5 min on the loaded Pi (sim 503 s, cmd 83 s, session 75 s). `go vet`
    clean for the default, calibration and acceptance tags; gofmt clean.
  * Goldens: the four historical references match under the migration
    manifest (90 days included); the V0.2 reference is byte-identical;
    `-verify-replay` matched on all five.
  * Acceptance on the Cortex-A76 (`-tags acceptance` test binary), all
    passed, ≈1 h 31 min with the suites sharing four cores: V0.1 criterion
    10 (50 seeds × 90 days, invariants after every transaction) and
    criterion 9 (1,000 chunkings, 20.7 s); new: invariants after every
    transaction for 50 seeds × 90 days with the V0.2 reference plan, and
    720 days of turnover — 155 hires, registry 163, slot staff sets three
    words wide, reproducible (44.5 s).
* **Slice 3 prerequisites (exact).**
  1. Date-aware roster versions: `planFor` composes registry terms by
     date (`dayTerms`); `set_roster` adds dated versions behind it, with the
     §6.4 composition (version keys, insert/replace/remove, tombstones).
  2. Reuse `reconcile` for roster reductions (cause `roster_reduction`)
     and `recoverDays` for additions; extensions that no longer fit a new
     version must reject (§6.4), not be voided.
  3. Training projections must be recomputed on a roster change before
     the productive date (`training_delayed` displacement).
  4. The regular-roster ≤ contract rule and §6.4's legacy-pattern rule for
     initial employees (frozen original week totals).
  5. Session v2 already carries arbitrary V0.2 actions; add the
     `set_roster` DTO fields and canonical comparison.
  6. Then the owner's benchmark: no action, partial and full
     reorganization over the same 20 seeds at T30, T90 and T180.

---

## V0.2 Slice 1 — employee contract and capacity foundation (0.2.0-dev.1)

The owner approved CONDITIONAL GO (2026-10-15). Decisions, design §18:
* The measured 1.5–5% margin band.
* ×1.30 demand.
* A shared-staffing start, with reorganization as the player's objective.
* Contracts kept at takeover; reductions only through termination and hiring.
* Provisional scenario-specific T90 gates.

Slice 1 builds the foundation and **adds no player action**. `established` and
`mini` outcomes are unchanged.

* **Contracts.** Hourly employees have contracted weekly hours. A scenario
  may state them (`contractedWeeklyMinutes`, 1–2,400); otherwise they are
  inferred from the roster as the smaller net rota week: E01/E03/E05 40 h,
  E04/E06 39.5 h. The accrual weights (the original roster by rota week and
  weekday) are frozen at load. Salaried staff have no contract hours; their
  salary is the guarantee.
* **Guaranteed pay.** Worked pay is posted exactly as before. At accounting
  close the week's guarantee earns the day's share, and a signed provision
  tops worked pay up to the contract (wages and benefits). Sunday's close
  finalizes the week. Guaranteed time is never worked time, so it creates no
  overtime and is never paid twice. A payday pays accrued payroll less the
  week's still-open guarantee. The provision stays inside accrued payroll
  rather than a new account, so the evaluation baseline's account list and
  digest are unchanged (design §5.5, revision of §9).
* **Employment registry** (state, digested): stable ID, lifecycle stage
  (only `active` until hiring exists), employed dates, contract and the
  week's pay counters. Checked every transaction. Observations show stage,
  contract minutes, the inferred flag, this week's open guarantee and the
  severance termination would owe (two contracted weeks with benefits).
* **Whole-schedule staffing check** (`World.CheckStaffing`, read-only). For
  a hypothetical roster change from a future day, it re-books every booked
  appointment in survivor order with the booking rule's own predicates. It
  reports each one that would be left `unstaffed`, without supervision
  (`no_supervision`), or competing for shared technologists
  (`shared_capacity`). It cancels, moves and displaces nothing, and makes no
  draw or log entry.
  * About 11 ms over the full horizon.
  * 150 unchanged-roster checks (established, the sustainable start and
    mini; 10 seeds; takeover to day 89) found no inconsistent existing
    booking.
* **Transition analysis** (design §5.6).
  * The shared start (`W130-SHR-MRft`) is built from existing scenario
    inputs. Its takeover contracts are Riley 40 h, Sam 39.5 h and Alex 40 h.
  * The lead staffing is reachable with the five approved actions:
    * `hire` Jamie, plus 27.5 h and 20 h replacements for Sam and Alex.
    * `terminate` Sam and Alex.
    * `set_roster` for Riley.
    * `set_reading_allocation` for the radiologists.
  * The plan reaches exactly the proposed 8-technologist cap during notice.
    At the earliest it is complete around day 41.
  * The saving comes from the part-time conversions: keeping every contract
    and only adding Jamie measured −$1.2k a month in Step 0C.
  * Estimated upfront cost is ≈$20–25k (recruiting, severance, training,
    overlap) against ≈$6k a month of headroom. **The full plan may not pay
    back within T90.** The dynamic benchmark must measure it; gameplay
    documentation does not promise +2.8%.
* **Compatibility.**
  * The state encoding is `dreamteam.state/v2`. Each historical golden
    reference differs in exactly `digests.state`, `digests.run` and
    `replay.runDigest`. Log, ledger, actions and baseline digests and every
    outcome value are identical.
  * The references stay byte-for-byte. `testdata/golden/MIGRATIONS.json`
    records old and new values with the explanation, and `-golden-migrations`
    requires both exactly.
  * Session records from v1-state builds (`0.1.*`, `0.2.0-dev.0.*`) replay
    with outcomes verified (action results, log, ledger, actions, summary)
    and the state change reported. Any other engine still needs every
    digest.
  * The Step 0C sustainable start and the lead candidate reproduce their
    committed per-seed results exactly (9 seeds each).
* **Regression tests.** Written before the implementation and shown failing
  against stubs:
  * The design's worked payroll examples to the cent (no work, 32 of 40,
    45 h with premium, 28 of 20, 40 of 40).
  * Provision reversal, partial week and rounding.
  * Contract inference; the registry in the state digest.
  * Guaranteed pay with a roster below the contract, and payday exclusion.
  * Neutrality of contracts at or below the roster: ledger and log hashes
    equal.
  * Extensions taking the week into overtime; `TestGuaranteeWithOvertime`
    passed against the stub too, so it is a guard.
  * Scenario validation and the start and lead contracts.
  * Unchanged-schedule, lost-coverage, shared-capacity, supervision,
    read-only and rejection staffing checks.
  * Golden migration strictness and v1-state replay. The replay test fails
    with the rule disabled.
* **Remaining for later slices:**
  * Growable staff sets (64-employee bit set).
  * Date-aware rosters and plan versions.
  * Recruiting and severance postings.
  * Displacement.
  * The five actions.
  * Session v2.
  * The named `sustainable` scenario.
  * Service-gate calibration from dynamic plans.
* **Verification:**
  * Full uncached `go test -count=1 ./...` passed: 7 packages, sim 349 s,
    cmd 65 s.
  * `go vet` clean (default, calibration and acceptance tags); gofmt clean.
  * The four goldens are unchanged and match under the migration manifest
    (90 days included).
  * `-verify-replay` matched on every reference.
  * The 90-day run time is unchanged within noise (MEASUREMENTS.md).

---

## V0.2 step 0C — late shifts and staffing architectures (0.2.0-dev.0.2)

Owner option G (2026-10-15): fix the noon shift-start limit, then test
explicitly different, fully costed technologist staffing architectures for a
larger practice. No action code; `established` and `mini` untouched.

* **Late shifts (engine).** A shift's cap, holdover included, is now the
  earlier of start + `maxShiftHours` and accounting close (23:59:59):
  `sim.Policies.ShiftCap`, used by runtime staffing, `extend_shift`, the
  roster view and alerts. Validation rejects shifts longer than
  `maxShiftHours` or ending after 23:59:59, not shifts starting after noon.
  Messages name accounting close when it is the binding limit.
  * Regression tests written failing first: `TestShortLateShifts`
    (16:00–20:00, 17:00–20:00, 11:30–20:00, 19:00–23:59 valid; over-cap and
    past-midnight rejected), `TestShiftCap`, `TestShortLateShiftsRun` (late
    shifts staff their rooms; invariants hold daily).
  * `TestRegress05MidnightBoundary` now runs its fixture: the late patient
    is rescheduled and nobody works past midnight.
  * Every existing roster starts before noon, so caps, goldens, replay and
    determinism are unchanged.
* **Harness** (calibration build tag, read-only): records conflict timing
  (weekday × hour), qualified staffed hours, technologist duty hours,
  reports by modality, estimated radiologist cost, payables, accrued
  payroll, receivables and the lowest daily cash. The test-only loss probe
  records each shared-conflict slot's start. The 20-seed Step 0B candidate
  reproduces exactly.
* **Sweep** (`docs/calibration/feasibility/step0c/`; 74 manifests; 9 paired
  seeds, 20 for the leaders):
  * 62% of CT shared-technologist losses were on Mondays.
  * A part-time MR evening is worth +$4.9k a month with no volume change.
  * Dedicated or float XR/CT coverage removes 80–95% of shared losses.
  * Part-time beats full-time for the same coverage by $2.6–7.8k a month.
  * The extra volume overwhelms reading: urgent p90 188 → 1,046 minutes at
    the base allocation.
* **Lead candidate** (×1.30 demand, 20 seeds): dedicated XR/CT day core,
  part-time evenings, reading at 62.5/52.5%.
  * Margin +2.8% (1.5–5.0%, median 2.6%); profitable on 20/20 seeds at T90
    and in days 31–90; lowest cash ≈$51k.
  * Breach rates better than the shared control: urgent 10.1% vs 10.4%,
    soon 27% vs 37%, routine 4.2% vs 7.0%.
  * 145 fewer lost referrals and 101 fewer walk-ins leaving.
  * At ×1.15 demand every architecture loses money (lead −4.0%); at ×1.50,
    out of range, +6.3%.
* **Verdict: CONDITIONAL GO.** Owner decisions (design §18): accept a
  ≈1.5–5% band; accept ×1.30 as the defining demand; choose the starting
  state (the candidate, or the shared practice with reorganization as the
  player's task); decide how part-time contracts are reached; freeze the
  service tolerances.
* **Housekeeping:** removed a committed `__pycache__` file; `.gitignore`
  ignores `__pycache__/`.
* **Verification:** new and updated regression tests; `go vet -tags
  calibration`; control reproduction; full uncached `go test -count=1 ./...`.

---

## V0.2 step 0B — CT/XR booking losses and templates (0.2.0-dev.0.1)

Owner option B (2026-10-15): find why CT/XR referrals are lost while their
rooms are ≈40% used, then test CT/XR templates as a permitted static lever.
No action, scenario or gameplay change; `established` and `mini` untouched.

* **Booking-loss probe** (diagnostics). Production change, explained: one
  function field `state.lossProbe` (excluded from the state digest, like the
  existing test hooks `dropOrder` and `forceNoShow`) and one nil-checked
  call where `offer` records a loss. The diagnosis itself
  (`internal/sim/lossprobe_export_test.go`, test builds only) classifies
  every compatible slot in the lost patient's acceptance window by the first
  FirstFit check it fails (booked, not rostered, no supervision, notice,
  shared technologist). It assigns one primary cause by documented
  precedence and also records every cause present. It reads state only.
* **Neutrality:** `TestLossProbeIsPassive` (log, ledger, state and action
  digests identical with and without the probe; every loss diagnosed once;
  no unexplained free slot). The 20-seed Step 0 candidate reproduces its
  per-seed results and fingerprint exactly with the probe on. Goldens
  unchanged.
* **Finding:** at the Step 0 best candidate, 72% of lost CT/XR referrals had
  a free, staffed slot in their window, blocked because one technologist
  covers both rooms (Riley mornings and Mondays, Sam evenings). 92% of them
  are urgent or soon patients. The template category mix explains ≈7% of CT
  losses.
* **Templates and coverage** (`docs/calibration/feasibility/step0b/`; 13
  manifests, 9 paired seeds, the best and its reference at 20):
  * 12-minute XR slots: +$1.16k per 90 days (CI +0.36 to +1.96), margin
    +0.4% → +0.5%. Recovered referrals displace walk-ins.
  * XR halved in shared hours: −$2.3k per 30 days.
  * XR grid staggered 5 minutes: −$5.2k (fragments shared time).
  * An XR/CT technologist (day or evening): recovers 145–170 losses and pays
    for itself, no more. Urgent breaches rise ×2–4: reading becomes the
    constraint. Restoring it with the 65/55 allocation: −$5.3k.
* **Verdict:** gate still not passed. The binding constraints are shared
  technologist staffing, low XR contribution, evening reading coverage, and
  fixed costs with salaried reading. Templates are exhausted. The best
  configuration is +0.5% mean T90 margin, 14 of 20 seeds profitable.
* **Tooling:** `report.py` takes a results directory and paired references
  and adds booking-loss cause tables. The calibration harness records loss
  causes per window.
* **Verification:** `TestLossProbeIsPassive`; `go vet -tags calibration`;
  calibration summary unchanged; full uncached `go test -count=1 ./...`.
* **Open:** owner decision on the premise (design §17.2: adopt the
  near-break-even candidate, build reading coverage and test teleradiology
  substitution dynamically, or revisit the staffing model). The noon
  shift-start fix stays a separate prerequisite. NO GO for action code.

---

## V0.2 step 0 — design draft 2 and the feasibility gate (0.2.0-dev.0)

Design and static experiments only: no engine, scenario, action or replay
change. No-action outcomes are those of v0.1.0.

* **V0.1.0 published.** GitHub Release "DreamTeam v0.1.0" from the existing
  tag (89b1779), after CI (run 38663474125) and a full uncached suite on the
  tagged tree passed.
* **Design draft 2** (`docs/V0.2_DESIGN.md`) applies the six owner decisions
  and resolves the 18 review findings (Appendix A): staff registry separate
  from the scenario; contracted, rostered and worked hours with guaranteed
  weekly pay and no double payment (legacy contracts inferred, including
  two 39.5-hour employees); the lifecycle from recruiting to severance; five
  actions with schemas, atomic batches, no-op rules and previews separating
  guaranteed from forecast costs; whole-schedule booking reconciliation and
  a practice-caused displacement lifecycle; employed vs outsourced reading
  with no price floor; finance postings; neutrality and golden migration;
  session v2 with a v1 reader; the T90 metric vector, baseline-relative
  service gates and an exploit table. `docs/ROADMAP.md` re-scoped V0.2–V0.6
  (V0.6 restored), V0.1 sections unchanged.
* **Step 0 feasibility sweep** (`docs/calibration/feasibility/`): 53 static
  overlays (demand ×1.00–1.30 uniform and imaging-weighted, XR/CT, US and MR
  technologists, 40-minute US slots, radiologist allocation 50/40–65/55),
  9 paired seeds at 30 and 90 days, the best three and the base at 20 seeds,
  plus a labelled out-of-range diagnostic (×1.50). Calibration harness:
  `DT_EXP_DIR`/`DT_RES_DIR`, parallel manifests, more aggregated figures;
  historical results untouched (summary regenerates byte-identically).
  `generate.py` writes the manifests, `report.py` the tables.
* **Gate result: not passed.** The best permitted configuration
  (imaging-weighted ×1.30 + MR evening technologist) breaks even: +0.4%
  mean T90 margin (−0.9 to +2.4), T90 profit +$2.8k mean (95% CI −$0.2k to
  +$5.7k), 12 of 20 seeds profitable. Demand ×1.50 gives only +1.6%: the
  practice cannot book it (CT and XR lose hundreds of referrals at ≈40% room
  use; US and MR are single rooms). Fixed costs and salaried reading are
  ≈58% of expenses; cutting the allocation collapses reporting at higher
  volume; US 40-minute slots change nothing; only the MR evening
  technologist pays.
* **Engine constraint found:** no shift may start after 12:00 (validation:
  start + 12 h `maxShiftHours` before midnight), so part-time evening shifts
  cannot be expressed; those cells are blocked (upper bound ≈+2.4–3.8%).
* **Verification:** `go vet -tags calibration ./internal/sim`;
  `TestCalibrationSummary` (historical summary unchanged); the sweep itself
  (four runs, all passing); `go test ./...`.
* **Open:** owner decision on the scenario premise (draft 2 §17.2: accept
  break-even, CT/XR template levers, cost base, reimbursement, teleradiology
  substitution, a second US room); then re-sweep. NO GO for action code.

---

## V0.1.0 — release preparation (v0.1.0)

Release preparation only: no change to mechanics, RNG, scenarios,
actions, accounting or replay.

* **Accepted deviations** (owner, 2026-10-14), recorded in
  `docs/ACCEPTANCE.md`, `docs/ROADMAP.md` and the README: the Pi 4's ≈3.3 s
  headless 90-day run (2 s target waived for V0.1; SHA-256 keyed RNG kept)
  and the naturally small or empty takeover worklist of the struggling
  default practice (no manufactured backlog; the 5–20 target moves to a
  future reporting-constrained scenario).
* **CI**: the Node 20 deprecation came from `actions/checkout@v4`,
  `actions/setup-go@v5`, `actions/upload-artifact@v4` and
  `actions/download-artifact@v4`; now `checkout@v7`, `setup-go@v7`,
  `upload-artifact@v7`, `download-artifact@v8` (all Node 24). Matrix and
  golden checks unchanged.
* **Release notes**: `docs/RELEASE_NOTES.md` (new; newest first).
* **Version** `0.1.0`. The golden references were not regenerated (no
  outcome changed): they still name the engine that wrote them, and the
  comparison reports the engine difference with outcomes identical.

---

## Steps 19–20 — Performance pass, acceptance results, release docs (v0.1.0-dev.20)

**Acceptance results.** Criterion 10: invariants after every transaction,
50 seeds × 90 days — passed (53.5 min on four Cortex-A76 cores; one seed
takes ≈7 min on the Pi 4). Criterion 9: 1,000 random chunkings of the real
simulation — passed. Criterion 8: the first CI run produced byte-identical
golden runs on linux/amd64, linux/arm64 and darwin/arm64. Every criterion
and its evidence: `docs/ACCEPTANCE.md`.

**Performance (criterion 16)**, details in `docs/MEASUREMENTS.md`:

| | Pi 4 (throttled, loaded) | Cortex-A76 (joofi) | Target |
|---|---|---|---|
| Warm-up | 1.44–1.49 s best | 0.53 s | < 1.5 s |
| `run -days 90` | ≈3.3 s | 1.1–1.2 s | < 2 s |
| TUI at 15 min/s | 7.4–8.2% of a core | — | < 15% |
| TUI paused | 0.4% (was 3.0%) | — | — |
| Max speed, worst frame | 52–94 ms | — | `space` within 100 ms |

The Pi 4 misses the 90-day target: its CPU has no SHA-256 instructions and
the keyed draws (the determinism contract) are SHA-256; the profile has no
other dominant hotspot. Changing the draw hash would change every outcome
and digest — an engine-version decision for the owner, not a tune-up. A
per-day free-slot cache for FirstFit was tried (outcomes identical) and
reverted: 3% at best.

**Changes.** The TUI's frame clock stops while paused and the renderer runs
at 20 fps (idle CPU 3.0% → 0.4%; outcomes unaffected). New benchmarks:
`BenchmarkEstablishedPlay90` (sim) and `BenchmarkMaxSpeedFrames` (TUI frame
times at Max). `dreamteam help` lists the `scenario` command.

**Release docs.** `README.md` (what it is, quick start, controls, actions,
scenarios, headless runs, sessions and replay, development, documents,
known limitations) and `docs/ACCEPTANCE.md`.

**Open for the owner.** (1) Accept criterion 16 on the Pi 4 as measured, or
decide on a faster keyed-draw hash for a later engine version. (2)
Criterion 1's "unread backlog at the takeover" does not hold for the
struggling practice (reading is over-provisioned at its volume). (3) Tag
V0.1 when satisfied.

---

## Steps 16–17 — Acceptance pass, part 1: TUI polish and end-to-end replay (v0.1.0-dev.18.3)

The V0.1 acceptance pass (ROADMAP §2), with the owner's list. The
checklist with every criterion's evidence is `docs/ACCEPTANCE.md`.

**Built.**
* **Help screen**: `?` from any screen shows the keys and the worklist
  rule; `?` or `esc` returns where it came from.
* **Event-log filters**: on the log screen `tab` cycles the category,
  `!` the severity (all · warnings & alerts · alerts only); the title shows
  the filter and how many entries match.
* **`run -log events.jsonl`**: the observable log from the takeover, one
  JSON object per line (seq, time, severity, category, text), through an
  output-only sink (never part of the state; the result is byte-identical
  with and without it). `-summary text|json` is an alias of `-format`.
* **NO_COLOR**: colors follow the profile Bubble Tea detects from the
  environment; with `NO_COLOR` no color sequence remains and severity stays
  marked (⚠ ‼). Documented in `dreamteam help`.
* **CI** (`.github/workflows/ci.yml`): vet (all tags), gofmt, the full
  suite and the golden references on linux/amd64, linux/arm64 and
  darwin/arm64, then a byte-for-byte comparison of the three platforms'
  results.
* **Acceptance suite** (`-tags acceptance`): criterion 9 on the real
  simulation (1,000 random chunkings of warm-up, takeover and a day of
  operations) and criterion 10 (invariants after every transaction, 50
  seeds × 90 days).

**Tests.** `TestTUIRunReplaysHeadless` (criterion 8: three randomized TUI
sessions — speeds, frame gaps 10–610 ms, pauses, steps, alert auto-pauses,
actions submitted from the actions screen — each reproduced by a headless
session from its action record and by replay; all five digests),
`TestHelpAndLogFilters`, `TestNoColor` (fails without NO_COLOR),
`TestRunLogJSONL`, `TestScreenSnapshots` (ten screens at 80×24, 120×36,
160×48; `-update-screens` to regenerate), and `TestRenderFitsTerminal` now
also covers 140×40, briefing, actions, help and the end-of-day summary.

**Deviation recorded.** Criterion 1 asks for an unread backlog at the
takeover; the struggling practice usually has none (reading is
over-provisioned for its volume, as characterized in step 18).

---

## Step 18.2 — Urgency always takes precedence on the worklist (v0.1.0-dev.18.2)

**A deliberate gameplay rule change (owner decision), not a calibration
correction.** Step 18.1 showed that shortest-first could postpone a long
urgent study indefinitely while short routine studies kept arriving. Now
urgent studies are offered before all others under every worklist policy,
and the policy orders within each class: default unchanged (urgent and
overdue first, then FIFO), FIFO → urgent first then FIFO, oldest-first →
urgent first then oldest, shortest-first → urgent first then shortest
published expected read. The existing `urgent` priority is the class (no
new severity); reads in progress are never interrupted; hidden read
durations are never used; ties still fall to the study ID. Reading blocks,
staffing, read durations and reimbursement are unchanged.

The TUI actions screen labels the choices "urgent first, then …" and says
"Urgent studies are always read first; the policy orders the rest"; the
worklist screen, API comments and SIMULATION_DESIGN §6.8 say the same.

**Tests.** `TestRegress18UrgentLongStudyVsShortRoutine` (every policy reads
the urgent MR next after the read in progress; before: FIFO and
oldest-first after 5 min, shortest-first never in 7 h) and
`TestRegress18WorklistUrgentFirst` (exact order per policy on a hand-built
worklist, determinism, shortest-first unchanged with no urgent study
waiting) fail on 049b304 and pass now. `TestWorklistPolicyOrdering` checks
shortest-first within each class. Full uncached suite green.

**Effect** (nine seeds, paired; details in the calibration report):
exams, lost bookings, walk-ins, revenue, opex and margin identical in every
run. 90 days: FIFO urgent breaches 32.9 → 15.1 and urgent p90 599 → 282
min (now equal to the default); shortest-first urgent breaches 18.2 → 16.3,
urgent MR median 99 → 52 min, routine p50 still 22 min. The default policy
is unchanged. The remaining urgent tail is the evening/weekend reading
coverage gap; with this rule the calibration phase is closed.

**Goldens.** Only `established-seed1-7d-actions.json` (it sets FIFO) was
regenerated; the three default-policy references are identical (compared
with the engine mismatch allowed). Its changes: reading order (urgent TAT
p50 1,407 → 803 s, max 3,628 → 2,256 s; breaches 13 → 12; one alert fewer;
radiologist busy time shifted between the two), the log digest
(6e34bd068f7f… → 625dfe83e5d8…), the ledger digest (6c18847e1bad… →
4482cec1c8d3…: revenue posts at report final, so entries move in time;
window money and balances are identical), state 72f117978ca0… →
5a626a6308b7…, run 9a3d8861926c… → fb0ccb14a79c…. Patient flow, exams and
money unchanged.

---

## Step 18.1 — Calibration review remediation (v0.1.0-dev.18.1)

The step 18 review (`docs/reviews/step-18-astra-review.md`, CONDITIONAL GO)
found five defects. Each got a regression test that fails on `b1a9ffb`,
then the smallest fix; every experiment was rerun on the fixed engine.

**Fixes.**
* `extend_shift` validates the raw minutes before converting them; huge
  values no longer wrap into a negative extension.
* Shared-staff booking is a real feasibility check: distinct qualified
  technologists for every overlapping appointment, handovers only at
  coverage boundaries, whatever the booking order. Default outcomes are
  identical; dispatch is unchanged; warm-up time is unchanged.
* Walk-ins: arrival, same-visit admission, the offer (declined by policy
  or unservable) and whether it booked, the same-visit exam and a later
  return exam are separate counters. Walk-in-origin booking events are
  out of referral demand (`demand.walkInOrigin`). Acceptance and
  abandonment are over the cohort of walk-ins who arrived in the period,
  each classified once (imaged same visit, converted, left, lost, on
  site). The TUI summary reads "walk-ins N: admitted, offered later
  (booked), left".
* The strategy helper no longer requests extensions past the scheduled
  close; the harness fails on any rejected action.

**Reproducible experiments.** `docs/calibration/experiments/*.json`
(scenario, overlay, strategy, seeds, horizon) run with
`go test -tags calibration ./internal/sim -run 'CalibrationExperiments$'`
and write `docs/calibration/results/<name>.json` (engine, fingerprints,
overlay hash, per-seed windows, aggregates) and `results/summary.md`. The
harness adds an order cohort, requested lead vs capacity delay, approved
overtime and turnaround by priority and modality.

**Results** (nine seeds; full tables in the report):

| | 30 days | 90 days |
|---|---|---|
| Baseline exams / margin | 1,363, −12.5% | 1,296 per 30 d, −14.0% |
| Three more technologists | 1,458, −16.7% | 1,390 per 30 d, −18.6% (step 18: −18.7%) |
| Faster XR (×0.75) | 1,390, −11.6% | 1,321 per 30 d, −13.1% |
| Reading allocation 45%/38% | TAT p50 20 h — unstable | TAT p50 22 h |

Strategies (30 days): B 1,431 exams, −9.4%; C 1,394, −13.4%; D 1,461,
−10.4%; E 1,394, −12.6%; F TAT p50 34 → 22 min (≈35%, not half). C and D
moved by at most one exam and 0.1 points with the booking fix; the
struggling-practice characterization stands.

**Urgent turnaround.** The default's urgent tail is a reading-coverage
gap: ≈¾ of urgent breaches are studies acquired after the last reading
block and read the next morning or Monday (max 31–41 h under every
policy). FIFO adds ≈15 in-hours urgent breaches per 90 days;
shortest-first ≈2, but it delays long urgent studies (urgent MR p50 99 vs
52 min) and, in a controlled micro case, starves them while short routine
work keeps arriving. No worklist semantics changed; the trade-off is
documented for an owner decision.

**Goldens.** All four regenerated: outcomes unchanged (log, ledger and
action digests identical), the state digest changed (new order and counter
fields in the state walk), and the result JSON changed in walk-in fields
and referral `lost`/`booked`.

| Golden | state digest | run digest |
|---|---|---|
| established-seed1-7d | 8a2a0fd92281… → c4d6b46c4d94… | af8fb442c246… → 7b99a6f7ffc1… |
| established-seed1-90d | 8f7de97d1fc0… → 8c6575d35565… | a25871c771cc… → 12157065771c… |
| mini-seed1-7d | 0db92e2b5226… → f2dff6e4d307… | e3b39b7707d0… → 26adad607323… |
| established-seed1-7d-actions | 08ab88ba9b21… → 72f117978ca0… | 9005416cd60a… → 9a3d8861926c… |

**Tests.** Regressions: `TestRegress18ExtensionMinutesRange`,
`TestRegress18SharedStaffFeasibility`, `TestRegress18WalkInCohorts`,
`TestRegress18FlowCountsFieldsFolded`, `TestRegress18OrderCohortReconciles`,
`TestRegress18UrgentLongStudyVsShortRoutine`. Full uncached suite green.

---

## Step 18 — Calibration: a struggling, capacity-constrained practice (v0.1.0-dev.18)

**Owner decisions (2026-10-14).** Calibrate in order — volume and flow,
reporting, economics, cross-seed — without forcing numbers. When the
measurements showed the margin target conflicts with the documented roster,
the owner chose: the default established practice is **struggling and
capacity-constrained**; documented reimbursement, costs, staffing and
operational assumptions stay; its margin is characterized, not forced; the
6–12% band belongs to a separate better-staffed preset, added only if it
can be configured cleanly (it cannot yet — next scenario milestone); the
mini scenario is unchanged; no hiring, roster editing, new actions or
artificial revenue.

**Method.** A calibration suite behind the `calibration` build tag
(`internal/sim/calib_test.go`, `calib_actions_test.go`): nine seeds, the
180-day warm-up, 30- and 90-day windows from the takeover, daily 07:30
worklist snapshots, per-person utilization, lost orders by reason, lead
times by modality; and a scenario overlay (`DT_OVERLAY`, the scenario
system's own merge) for trying candidate parameters without editing files.
The pre-calibration nine-seed record is
`docs/calibration/baseline-0.1.0-dev.15.json`; the full report with every
table is `docs/calibration/step-18-report.md`.

**Baseline diagnosis** (nine seeds, 90 days; 30-day in brackets):

| | Mean (range) |
|---|---|
| Demand: orders + walk-ins a month | 1,423 + 224 = 1,647 |
| Exams a month | 1,314 (1,293–1,335) [1,381] |
| Tue–Fri visits a day | 59.9: XR 27.0, US 10.0, CT 15.1, MR 7.7 |
| Lost bookings a month | 127: US 64, MR 29, CT 17, XR 17 (almost all "declined": earliest slot beyond tolerance) |
| Walk-ins leaving | ≈100 a month, 45% of walk-ins |
| No-shows / cancellations | 6.5% / 8.9% (as documented) |
| Technologists busy (of duty) | Sam 82%, Morgan 82%, Casey 81%, Riley 76% |
| Radiologists reading (of site blocks) | Avery 73%, Blake 63% |
| Worklist at takeover / 07:30 Tue–Fri | ≈0 / mean 3.3, max 9–18 |
| TAT p50 / p90 / over target | 32 min / 122 min / 5.2% |
| Net revenue, labor, fixed, variable a month | $200.4k, $106.8k, $87.8k, $32.7k |
| Operating margin | −14.0% (−15.6 to −13.0) [−12.5% (−16.9 to −11.1)] |

**Bottlenecks.** Demand matches the design; capacity does not. XR and CT
share technologist time most of the day (Sam and Riley saturated, rooms
idle 60%+): walk-ins register, wait 30–112 min and leave. US is one
technologist with 45-min slots for ~37-min exams (69% busy of staffed time,
the most lost demand, lead p50 19 days). MR is one technologist with nine
slots. Reading is over-provisioned for this volume (hence the empty
evening worklist and fast TAT).

**Experiments** (nine seeds; none applied): XR room times ×0.75 → +27
exams a month; walk-in policy `accept_always` → no change; radiologist
site allocation 45%/38% → unstable reading queue (TAT p50 20 h), 55%/45% →
stable, ≈$5k a month cheaper (not applied: documented staffing kept); three
more technologists → lost bookings −70% but exams +7% and margin −18.7%,
because demand, no-shows and cancellations bind. The design's reference
P&L counted ≈1,650 orders as ≈1,650 exams with technologists ~94% busy:
internally inconsistent with its own roster (now noted in §8.6).

**Parameters changed: none.** No change was defensible under the decision
(each candidate either altered documented staffing or only marginally
moved volume), so the established scenario's fingerprint is unchanged
(`b1cbc2a0…`), and the golden references are unchanged: their inputs and
outcomes did not change (rule: update only when they do). What changed is
the characterization: design §5.1, §6.2, §8.6, ROADMAP step 18 and the
V0.1 calibration criteria (now the better-staffed preset's), the briefing
note, and the tests. The four references still name engine 0.1.0-dev.15; under
0.1.0-dev.18 their outcomes reproduce exactly (`TestGoldenReferences`
compares with the engine difference allowed and reported). The CLI without
`-allow-engine-mismatch` reports them as not comparable (exit 5), as
designed; regenerating them would only restamp the engine.

**Actions** (30 days, nine seeds; demand identical across strategies,
asserted):

| Strategy | Exams | Lost | Margin |
|---|---|---|---|
| A no actions | 1,363 | 141 | −12.5% |
| B US + MR extended to close (approved ahead), walk-ins until 19:00 | 1,431 | 94 | −9.4% |
| C Riley extended (2nd XR/CT evenings), accept all walk-ins | 1,394 | 127 | −13.5% |
| D all three extensions, accept all | 1,460 | 79 | −10.5% |
| E decline walk-ins, FIFO | 1,394 | 178 | −12.6% |
| F shortest-first worklist | 1,363 (TAT p50 22, p90 157 min) | 141 | −12.5% |

Actions matter (B recovers a third of lost bookings; F halves TAT; E turns
walk-ins who would leave into appointments) and every added hour is paid
(117–194 h of approved overtime), but none reaches break-even: profitability
needs hiring and demand growth (V0.4). No loophole found: demand is
unchanged by any strategy, declined walk-ins stay counted, extensions are
costed, worklist ordering uses published read times only.

**Targets** (report table): demand, no-shows, MR utilization cap, US
longest lead and most losses, cash/profit divergence — met. Tue–Fri
visits (60 vs 65–85), XR and MR per day just below bands, US 80–90%
utilization (unreachable with 45-min slots, no overbooking), radiologist
capacity 105–115% (≈145%), 07:30 backlog 5–20 (mean 3.3), routine TAT 1–4 h
(32 min), T30 margin 6–12% — missed by the default and assigned to the
better-staffed preset.

**Tests.** `TestEstablishedCharacterization` (three seeds, 30 days, broad
bands: demand 1,450–1,850 a month, exams 1,250–1,500, margin −20% to −6%,
25–65% of walk-ins leave, US loses most, TAT p50 < 2 h and morning backlog
< 40, books reconcile). The calibration suite is tagged (≈30 s and ≈2 min).
Determinism, replay (step 14), warm-up boundary (step 11), closing rules
(step 8), demand independence and goldens are unchanged and pass:
`go test -count=1 ./...`.

**Limitations and next scenario milestone.**
* The better-staffed `moderate` established practice (6–12%) needs more
  staff and more demand; demand growth is V0.4 (referrals). Until then the
  default is the only established practice.
* The P&L workbook and `docs/ASSUMPTIONS.md` (ROADMAP step 18) move with it.
* Reading capacity exceeds this practice's demand; a struggling owner might
  reduce the site allocation — not a V0.1 action.
* Forty-five percent walk-in abandonment is a consequence of the roster;
  patience (30–120 min) is as documented.

---

## Step 15 — Canonical results and golden references (v0.1.0-dev.15)

**Owner decisions (2026-10-14).** Engine-version compatibility stays:
replay and golden comparison against another engine only with
`-allow-engine-mismatch`, and compatibility is always reported apart from
verification. Golden outputs are canonical JSON for machines plus a concise
text summary for people. Golden files hold observable and financial
outcomes and digests, never individual records or hidden state.

**Result schema** (`dreamteam.result/v1`, SIMULATION_DESIGN §4.5):

| Section | Contents | Kind |
|---|---|---|
| identity | schema, engine, scenario ID and fingerprint, seed, epoch date | — |
| `takeover`, `end`, `playDays` | simulation seconds and ISO time | times |
| `actions` | attempted / accepted / rejected, history digest, every result in order (rejections and no-ops included) | history |
| `digests` | log, ledger, actions, state, run | verification |
| `calendar` | each completed day, midnight to midnight (the takeover day flagged): orders, lost, walk-ins, no-shows, exams, reports, unread at close, wait p50, overtime, net revenue, expenses, profit, collections, disbursements, end-of-day cash, alerts | calendar flows (+ end-of-day cash) |
| `window` | from the exact takeover to the end: baseline digest, period metrics (demand, appointments, walk-ins, waits, rooms, staff, reporting, workforce, alert and action counts), window money (baseline-adjusted and as posted, cash bridge) | evaluation-window flows |
| `balances` | cash, A/R, A/P, payroll owed, equity, lowest cash, days below zero | point in time |
| `obligations` | next payroll and bills, open claims and aging, unread / reading, recall list, booked ahead, policies, approved extensions | point in time |
| `alerts` | raised in the window by severity; active at the end | counts |
| `replay` | verified, batches, replayed run digest (with `-verify-replay`) | verification |

Integers only; ratios as numerator and denominator; fixed field order;
arrays in defined order; no wall-clock time, paths or locale. Every field
is compared.

**CLI** (no new execution path: `run` and `replay` gained output flags):

    dreamteam run -scenario established -seed 1 -days 7 -format json            # canonical JSON on stdout
    dreamteam run -days 7 -actions FILE -verify-replay                           # text summary, replay checked
    dreamteam run -days 90 -verify-replay -format json -golden testdata/golden/established-seed1-90d.json
    dreamteam run -days 7 -format json -write-golden FILE [-overwrite]           # explicit, never silent
    dreamteam replay SESSION -format json [-golden FILE]                         # the replayed run's result
    dreamteam verify SESSION                                                     # "ok <run digest>"

Diagnostics go to stderr; with `-format json` stdout holds only the JSON.
Exit codes: 0 ok · 1 error (I/O, scenario) · 2 invalid input · 3 replay
failure · 4 golden mismatch · 5 incompatible or unsupported format (another
schema, engine, scenario, seed or run length; malformed golden; unknown
`-format`). A mismatch report gives compatibility, the number of
differences, the digests that differ and the first differing paths with
both values, e.g. `first: .window.ops.exams: golden 1, run 184`.

**Reference runs** (`testdata/golden/`, README there; each generated with
`-verify-replay`, aggregate outcomes plus digests, 16–52 KB):

| Reference | Exams | Orders / lost | Net revenue | Op. expense | Op. profit | Cash end | Run digest |
|---|---|---|---|---|---|---|---|
| established, seed 1, 7 days | 292 | 329 / 33 | $47,037 | $55,712 | −$8,675 | $144,817 | `af8fb442c246…` |
| established, seed 1, 90 days | 3,887 | 4,171 / 370 | $593,096 | $675,797 | −$82,702 | $16,985 | `a25871c771cc…` |
| mini, seed 1, 7 days | 184 | 360 / 4 | $27,893 | $50,784 | −$22,891 | $137,965 | `e3b39b7707d0…` |
| established, seed 1, 7 days, actions (8: 5 accepted incl. a no-op, 3 rejected) | 294 | 329 / 32 | $47,452 | $55,899 | −$8,447 | $144,707 | `9005416cd60a…` |

(Uncalibrated: the losses are the step 10–11 volume shortfall that step 18
calibration addresses. The 90-day run ends with cash near zero.)

**Tests** (`go test -count=1 ./...` passes):

| Requirement | Test |
|---|---|
| References reproduce (byte-identical when the engine matches) | `TestGoldenReferences` (all four, `-verify-replay`) |
| Repeated runs byte-identical; seeds and scenarios differ | `TestResultsRepeatableAndDistinct` |
| Meaningful changes, missing / unexpected fields, digests, calendar length, engine, fingerprint, seed, run length, schema, malformed / truncated / trailing files, CLI exit codes, no silent overwrite, unsupported format | `TestGoldenComparisonReports` |
| Action history changes detected; rejected actions represented | `TestActionHistoryInResult` |
| Chunk sizes and playback speeds (one jump, 37-min chunks, Max-speed event budgets, alert stops) | `TestResultIndependentOfChunkingAndSpeed` |
| No hidden data | `TestResultHasNoHiddenData` (no record IDs, no latent or queue fields) |
| Financial reconciliation, window boundaries | `TestResultReconciles` (balance sheet, daily cash roll-forward, cash bridge, window = takeover → end, calendar days = play days) |

**Cross-platform.** All four references reproduced byte-identically on a
second Linux arm64 machine (Cortex-A76, via the built binary) as well as
the Raspberry Pi 4 (Cortex-A72) that wrote them. Linux amd64 and macOS
arm64 have **not** been run (no such machine or emulator here); the
procedure is in `testdata/golden/README.md`.

**Performance** (MEASUREMENTS.md): `run -days 7` 1.9 s, `-days 90` 3.1–4.7
s (warm-up included); result build and encode after 90 days 0.54–0.62 s
(mostly the state digest); comparison 18–25 ms. Not in any output.

**Limitations.**
* An engine version bump requires `-allow-engine-mismatch` for old
  references (reported); regenerate to make them byte-identical again.
* Per-day calendar rows make the 90-day file 52 KB; per-patient detail is
  deliberately absent, so a digest-only change (e.g. the order of two
  same-day events) is reported as differing digests without a metric path.
* No amd64 or macOS verification yet.

---

## Step 14 remediation — replay review findings (v0.1.0-dev.14.1)

A focused review of replay correctness (`docs/reviews/step-14-astra-review.md`,
gpt-6-astra at high reasoning, read-only snapshot of `16c53a2`) returned
NO GO with eight findings; all were confirmed, each with a regression test
that fails on `16c53a2` (finding 7 by inspection: no injection point
existed) and passes after the smallest fix. No simulation outcome changed.
The suspected pause-then-act mismatch was ruled out.

| # | Finding | Fix |
|---|---|---|
| 1 (high) | Hour keys like `"+8"` were accepted beside `"08"`: compilation depended on map order | only canonical `"00"`–`"23"` |
| 2 (high) | The run digest did not bind the scenario: future rules could differ under the same digest | run digest includes the scenario fingerprint and seed |
| 3 (medium) | Sessions kept the caller's scenario object: later edits broke saves | private copy |
| 4 (medium) | The state digest included the last-transaction time cache | excluded |
| 5 (medium) | Trailing `]`/`}` after a record, data after a scenario, a second actions array accepted | EOF required |
| 6 (medium) | Re-signed edits to batch numbers, times, scenario ID/name, start/until and summary verified | all checked against what they describe |
| 7 (medium) | Directory-sync failure after rename reported as a durable save | reported; session stays unsaved |
| 8 (low) | Refusals printed "OK … digests match"; mismatch summaries were zeros | OK only when fully checked; real replay position reported |

Records saved by `16c53a2` no longer verify (the run digest changed); they
replay with `-allow-engine-mismatch`. Remaining test gaps are listed in the
review document.

---

## Step 14 — Sessions, replay and digests (v0.1.0-dev.14)

### Part A — playtest (before step 14)

Driven through the real TUI model (scripted keys, every screen read at
80×24) on the established practice, seed 1:

1. **Two days, no actions.** Alerts, queues, worklist and money agree:
   day 1 70 exams, 16 alerts (mostly XR waits behind the shared XR/CT
   technologist), 3h 01m overtime, profit $575; day 2 60 exams, an insight
   flags MR busy 86% of staffed time but staffed 70% of open hours. The
   17:00 dashboard shows self-registration after the desk staff leave;
   "MR room unstaffed — why: Casey on break until 13:00" matches the roster.
2. **US extension a week ahead** (Morgan, Wed Oct 21, +2h): the schedule
   for that day went from 2 to 5 open US slots (18:00, 18:45, 19:30), at
   once. Demand untouched (tests).
3. **Walk-ins: decline, then accept-if-gap.** The 3 walk-ins arriving
   while declining were offered appointments and counted as declined;
   nobody already waiting was removed; switching back took effect for the
   next arrival.
4. **Worklist: priority → FIFO** with an urgent and two routine studies
   unread: the urgent XR dropped from first to last at once; MR and US were
   read first, the urgent one at 38 min; the read in progress continued.

**Defects found and fixed (all in action feedback):**
* An extension's preview did not say what the extra time covers, and an
  extension that would cover nothing was accepted. Now each says
  "staffs US 18:00–20:00", "covers the desk …" or "on site for contrast
  supervision (reading blocks unchanged)", and one covering no room, desk
  or supervision is refused (`no_coverage`). Coverage is evaluated on the
  extended plan, the same plan dispatch and booking use, so the preview,
  the schedule and the effect agree (`TestExtensionWithoutCoverageRefused`).
* It did not say the benefit: it now names the slots it opens ("opens 3
  US slots for booking") or says none would be ("already staffed, or no
  template slots then" — e.g. Riley's XR evening, which Sam covers).
* At 80 columns the preview and result were cut off (the cost and effect
  lost) and dates were ISO. They now wrap over up to three lines ("Done ✓
  …", "If submitted: ✓/✗ …"), dates read "Wed Oct 21", and the next preview
  restarts at +60m instead of showing a confusing rejection.
* Declined walk-ins were cut off the 80-column dashboard: now "walk-ins 3
  (3 declined)".

Owner note: the five-day comparison's +$181 does not make the policy
optimal — 22 h of approved overtime for 8 exams may reverse once step 18
calibrates rates and staffing. That trade-off is for players and agents to
discover.

### Part B — sessions, replay, digests

**Principle.** A saved run is reproduced from its scenario, seed and
timestamped actions; no serialized world is trusted. The session file
(`dreamteam.session/v1`) is a run record: the fully resolved scenario with
its fingerprint, the seed, epoch and takeover, every action batch with its
results (rejections and no-ops too), the saved time, five digests, a
readable summary and a checksum. Format and rules: SIMULATION_DESIGN §4.5
(as built).

| Digest | Covers |
|---|---|
| log | every observable log entry, in order |
| ledger | every journal entry, in order |
| actions | the action record with results |
| state | the whole authoritative state, hidden fields included, and the live pending events; not caches, hooks, processing counters or event sequence numbers |
| run | all of the above and the clock |

**Load = replay.** Strict parse → checksum → schema → scenario fingerprint
→ batch structure → compile, warm up, apply every batch at its time
through `Apply`, compare each result → run to the saved time → compare the
digests. Mismatches name the first differing action (batch, time, action,
recorded vs replayed) or the differing digests, with both summaries.

**Compatibility.** Another schema: refused. Another engine version:
refused unless `-allow-engine-mismatch`, which replays and lets the digests
say whether outcomes agree. Changing the rules (any future gameplay step)
therefore changes which old saves verify: that is the point.

**Changes to existing behavior (no outcome changes).**
* Alerts keep the text they were raised (or escalated) with; before, the
  stored text was refreshed at each evaluation, so a *resolved* alert's
  displayed text depended on when unrelated events happened. Observations
  of active alerts still show the live text.
* A batch of only rejected or no-op actions no longer moves the metrics'
  accrual point or the clock cache; an empty batch is not a batch.
* `Session.Digest()` is now the run digest (log, ledger, actions, state,
  clock).
* Saved and loaded sessions without actions are identical to `08ccc5d`
  (ledger and every visit, study and appointment, established 21 days and
  mini 30 days, seeds 1–3).

**Controls.**
* TUI: `ctrl+s` save, `ctrl+o` reload (replay the saved file and resume),
  `ctrl+v` verify by replay; the top bar shows the seed and "● unsaved".
* CLI:
  * `dreamteam play [-load FILE] [-save FILE]`
  * `dreamteam run [-scenario ID] [-seed N] -days D [-actions FILE] [-save FILE]`
    — headless, prints each action result, the since-takeover figures and
    the digests. Actions file: `[{"time": "2026-10-13T07:00:00", "actions": [...]}]`.
  * `dreamteam replay FILE [-allow-engine-mismatch]` — full report.
  * `dreamteam verify FILE` — `ok <run digest>` or the mismatch, exit 1.

**Tests** (`go test -count=1 ./...` passes; `internal/sim` 158 s,
`internal/session` 21 s):

| Requirement | Test |
|---|---|
| Save/load round trip, resume | `TestSaveLoadRoundTrip` (accepted, rejected, no-op, two multi-action batches, a future-day extension; both sessions continue identically) |
| Replay after restart | `TestRunSaveReplayVerify` (CLI: separate run, file, replay, verify) |
| Midday takeover, cross-day and cross-week, payroll and claims across resume | `TestReplayMiddayTakeoverAcrossPayrollAndWeek` (13:00 takeover, saved Thursday 23:00, continued past Friday payroll and remittances to the next Tuesday) |
| Corrupt, truncated, edited, fingerprint, schema, engine, tampered results and digests, order | `TestCorruptAndIncompatibleFiles` (11 cases) |
| Atomic save failure | `TestAtomicSaveFailure` |
| Chunking | `TestReplayChunkingInvariance` (17 min, 3 h, single step) |
| Independence from unrelated events, reads, previews, rejections | `TestStateDigestIndependentOfNoiseAndReads` |
| Hidden state covered, nothing leaked | `TestStateDigestCoversHiddenState`, `TestSavingHasNoSideEffects` (no latent fields in the file; saving and verifying change nothing) |
| Determinism | `TestStateDigestDeterministic`, `TestCanonicalRoundTrip`, `TestParseISORoundTrip` |
| TUI | `TestSessionKeys` |

**Performance** (MEASUREMENTS.md): warm-up unchanged (1415 ms vs 1359 ms
best, within thermal noise; identical memory); save after 30 days
0.44–0.54 s; load = replay + verify of 30 days 2.1–2.3 s; `run -days 30`
plus save 4.0 s and 48 MB peak; file 35 KB.

**Limitations.**
* No snapshots: loading always replays from the epoch (warm-up included),
  about 1.4 s plus the played days. A snapshot cache verified against the
  digests can come with V0.2 save/load UI.
* The state digest is a reflection walk over the state's fields: a field
  added later is covered automatically, but a new cache must be added to
  its skip list (or the noise test fails, which is how this one was found).
* Records include the seed: fine for game saves; benchmark records will
  escrow it (V0.5).
* Golden 7- and 90-day digests (ROADMAP testing strategy) are not pinned
  yet; the start-state goldens are.

---

## Step 13 — Player actions (v0.1.0-dev.13)

**Owner decisions recorded (2026-10-14).** The four step 12 conventions are
approved. For #3, the window's operating result excluding inherited-claim
collection variance is a *baseline-adjusted* evaluation convention: the
since-takeover screen labels it "baseline-adjusted" and shows the ledger
basis ("as posted (the accrual P&L)") next to it; the calendar statements
and `WindowMoney.Ledger` keep the actual collection variance. A write-down
of an inherited receivable after takeover is visible there.

**What the player can do.** `a` opens the actions screen (auto-pause moved
to `A`): extend a shift today or on a day up to two weeks ahead (`[` `]`),
set the walk-in policy, set the worklist policy. Each row is previewed
live; `enter` submits; results and the latest actions are listed. The
dashboard shows the policies in force and approved extensions.

**One interface.** `World.Apply` / `Session.Apply(actions...)` is the only
way to change the simulation; `Validate` previews without effect. The TUI
calls the same functions an agent will. Batches apply at the current clock
between transactions, in order, each action against the state the previous
left; a rejected action changes nothing; one dispatch and the alert
observer run after a batch that changed something, none after rejections or
no-ops. Every attempt is recorded with a stable ID (`X00001`), batch
number, time, action, result code and message (`ActionLog`,
`Observation.Actions`, and per-period counts in the metrics and the
evaluation window). Schema, codes and rules: SIMULATION_DESIGN §9.5 (as
built).

| Action | Parameters | Effect |
|---|---|---|
| `extend_shift` | `employee`, `minutes` 1–240, `date` (optional; today to the booking horizon) | the scheduled end moves later that day; they stay on regular duty; the day's slots in the extra time become bookable at once; time past the rostered end is approved overtime |
| `set_walkin_policy` | `policy`: `accept_always`, `accept_if_gap`, `accept_until` (+ `cutoff` HH:MM), `schedule_later` | walk-ins arriving from now on; declined ones are offered appointments, counted apart |
| `set_worklist_policy` | `policy`: `urgent_then_fifo`, `fifo`, `oldest_first`, `shortest_first` | the order unread studies are offered; reads in progress continue |

**Extension vs hold-over** (the design issue you raised). Required hold-over
finishes patients already admitted; it starts past the scheduled end and
only while someone needs the person. An approved extension moves the
scheduled end, so the person keeps staffing their room on regular duty: new
capacity for waiting patients, walk-ins and bookings. Approved and required
overtime are recorded apart (`ApprovedOvertimeSec`/`…CostCents` in the flow
counters, staff metrics, workforce metrics and the window). Hard limits stay
hard: the 12-hour cap (rejection says how much is left), no retroactive
approval (`shift_ended`, `past_date`), no extension that would only add
time after scheduled close (`after_close`: hold-over covers that), breaks
and reading blocks unchanged, nobody else extended.

To give extensions something to fill, the established roster's US and MR
templates now continue to the evening (US 18:00–20:00, MR 16:30–19:30).
Those slots are unstaffed — not bookable — unless a shift is extended; runs
without actions are identical to step 12 (checked below).

**Did the decisions matter?** Same seed (1), established practice, 5 days
from takeover, no actions vs. a plan approved at takeover: US and MR
extended to the 20:00 close on every weekday, walk-ins accepted until
19:00 (`TestInterventionsChangeOutcomesNotDemand`):

| 5 days from takeover | no actions | interventions |
|---|---|---|
| orders received / walk-ins arrived | 254 / 39 | 254 / 39 (identical, same attributes) |
| exams | 259 | 267 |
| orders lost | 27 | 22 |
| US staffed / busy, of open hours | 57% / 34% | 71% / 40% |
| MR staffed / busy, of open hours | 58% / 48% | 82% / 52% |
| arrival→exam wait p90 | 52 min | 51 min |
| overtime (of it approved) | 6 h (0) | 28 h (22) |
| net revenue / expenses | $40,558 / $43,094 | $42,641 / $44,997 |
| operating profit (baseline-adjusted) | −$2,536 | −$2,355 |

The first version only extended shifts on the day itself: US and MR were
staffed, but almost nothing was booked into the new time, because bookings
are made days ahead (+4 exams for 22 h of overtime; profit fell to −$3,488).
Hence the `date` parameter: approved ahead, the slots open for booking now.
Uncalibrated figures; the point is that management decisions now change
operations and money while demand stays exogenous.

**Tests** (`go test ./...` passes; `internal/sim` 149 s):

| Requirement | Test |
|---|---|
| Valid and invalid extensions, cap | `TestExtendShiftValidation` (16 cases), `TestExtensionAheadOpensBookings` (dates, horizon, Sunday, cap) |
| Extension vs hold-over; payroll | `TestExtensionVersusHoldOver` (36 min required vs 60 approved; one wage entry = hourly pay of on-duty time; approved cost = overtime cost) |
| Capacity, not demand | `TestExtensionAddsCapacityNotDemand`, `TestExtensionAheadOpensBookings` (evening US slots booked ahead), `TestInterventionsChangeOutcomesNotDemand` |
| Walk-in policies, declines measurable, not retroactive | `TestWalkInPolicies`, `TestWalkInPolicyNotRetroactive` |
| Worklist ordering, read times, no interruption, no double read | `TestWorklistPolicyOrdering` (shortest-first follows the published p50 only) |
| Simultaneous actions | `TestSimultaneousActionsOrdered` (order-dependent results; intermediate policies inert) |
| Invalid-action atomicity | `TestInvalidActionAtomicity` |
| Replay | `TestActionReplayEquivalence` (re-applying the record reproduces log, books and results) |
| No retroactive effects | `TestActionsNotRetroactive` |
| Chunking, unrelated events | `TestActionsChunkingAndNoise` |
| Evaluation window | `TestEvaluationAttributesActions` |
| TUI | `TestActionsScreen` (submit, reject, policy, ahead; 80×24, 120×36) |

Without actions, the ledger and every visit, study and appointment are
identical to `4bbd03f` (established 21 days, mini 30 days, seeds 1–3), and
the pinned start-state digests are unchanged.

**Performance** (MEASUREMENTS.md): warm-up 1402 ms vs 1401–1434 ms (best
runs; the Pi alternates between ~1.4 s and ~2.5 s with throttling); one
preview 4 µs, one applied action with its dispatch 12–14 µs; 30 days after
takeover 405 ms without actions, 380 ms with daily interventions.

**Known limitations.**
* Extensions are whole-shift-end only: no early starts, no extra breaks
  for long shifts (the cap bounds them), cover windows keep their times,
  radiologists' reading blocks are not extended (their supervision is).
* A cost estimate ahead uses the base rate; the ×1.5 depends on the
  week's hours when worked.
* No undo: an approved extension stays (V0.2 `send_home`).
* The evening slots exist only for US and MR in the default roster.
* Rejected actions are in the action record, not the event log.
* Run records and replay files are step 14.

---

## Step 12 — Metrics, alerts, daily summaries (v0.1.0-dev.12)

**What the player sees.** The dashboard answers "what is going wrong, and
why": an **ALERTS** panel (active alerts, most severe first, time raised,
the screen that shows more, and a "why:" line built from observed facts —
"shared technologist Riley busy (documenting P00031)", "no XR technologist
rostered until 09:30") and a **Today** line (time-weighted utilization of
staffed hours by room, arrival→exam wait p50/p90, lost orders). `e` runs
today's operations to operational close and opens the **end-of-day
summary** (TUI_DESIGN §4.10): patients, demand, schedule, waits, room time
three ways (busy of open, busy of staffed, staffed of open), staff,
reporting, money, cash, alerts and rule-based insights; `←→` pages earlier
days, `tab` shows **since takeover**. It fits 80×24 untruncated (short
forms below 110 columns). The history and briefing screens now draw on the
same measurements (room busy time within open hours, so ≤ 100%).

**Reporting windows (owner decision, 2026-10-14).** Two windows:

| Window | Purpose | Definition |
|---|---|---|
| Calendar | TUI, daily history, statements | midnight to midnight, including the takeover day's activity before the takeover |
| Evaluation | future player/agent evaluation | from the exact takeover instant to `evaluationDays` (scenario; 0 = open-ended) |

The baseline is captured at the end of the takeover transaction, after the
opening adjustment, so everything at that instant belongs to the inherited
state and the window is (takeover, end]. It is immutable and digested:
balances and trial balance, open claims by state, unreported studies,
booked appointments, recall list, patients on site, staff on duty, staffed
rooms, ledger and log positions. Window money is the journal after the
baseline (a second day book), with operating performance separated from
cash flow and working capital and a cash bridge that reconciles to the cent.
No weighted score, weights or thresholds: measurements only.

**Metric definitions** (full table: SIMULATION_DESIGN §11.1). One
cumulative tally integrates the state between transactions; a period is the
difference of two snapshots (midnight, takeover, endpoint, or now).

* Rooms: open (scheduled hours); staffed (open with a qualified technologist
  on duty; a shared technologist counts fully for each room); busy (patient
  in the room, within open hours); after-hours busy apart. Utilization of
  open hours = busy/open, of staffed hours = busy/staffed, coverage =
  staffed/open: none can exceed 100%.
* Staff: on duty (paid time, breaks out, overtime in), busy (registration,
  exam, documentation, reading), break, overtime. Technologist utilization =
  busy/on duty per person (shared time counted once). Radiologist reading
  utilization = reads inside site-reading blocks / block time; overrun reads
  apart.
* Appointments (slots starting in the period): fill = booked/bookable slot
  time; no-show rate = no-shows (missed included until accounting close) /
  (scheduled − cancelled); cancellation rate = cancelled/scheduled.
* Demand: received, booked, recall, lost, cancelled; lost rate =
  lost/received. Walk-ins: acceptance = imaged/arrived, abandonment =
  left/arrived.
* Waits (exams started in the period): arrival→registration,
  arrival→exam, exam start past the appointment time.
* Reporting: TAT of studies imaged in the period, unfinished ones censored
  at the period end; TAT of reports finalized (inherited ones counted);
  breaches; backlog at both ends.
* Workforce: overtime and its cost, held-over, skipped breaks, coverage
  reschedules, patient-time waiting to register with no desk staff.
* Ratios keep numerator and denominator (an empty denominator is "—", never
  0%); percentiles are nearest-rank everywhere (the worklist's median and
  p90 now use the same definition).

**Alert behavior.** Ten rules over observable state, from the takeover on:
waits (30 min warn, 60 min alert; measured past the appointment time, or
since arrival for walk-ins and patients fitted in after grace), unstaffed
rooms with patients on site, report TAT overdue (urgent alert, routine
warn, soon tracked since each breach is already logged), held-over staff
(tracked) and the 12-hour cap approaching (alert), closing running late (30
min warn, 60 alert), reporting backlog (≥ 40, clears below 32), lost demand
(≥ 3 per modality per day, expires at midnight), cash low (< $25,000) and
negative (alert; replaces the step 11 shortfall entry, same text). One
raise per episode, at most one escalation, a logged resolution after a
10-minute clear for waits, rooms and backlog. Time-driven changes get one
exact passive wake (no polling); passive events never trigger dispatch, so
they cannot change outcomes. Seed 1 raised 16 alerts on day 1 and 11 on
day 2 (escalations included), mostly waits behind the shared XR/CT
technologist and short unstaffed spells during breaks, plus one late close. Thresholds are configurable (`alerts` in
the scenario) and uncalibrated.

**Conventions to confirm (owner).**
1. Events at the exact takeover instant (midnight payroll or bills, 10:00
   remittances, dispatch) belong to the inherited state, consistent with the
   step 11 opening rule; the window is (takeover, end].
2. The takeover day's midnight accruals (salaries, fixed costs) cover the
   whole day; the window bears the share after the takeover (18/24 at
   06:00), shown as "takeover-day accruals" and in the cash bridge.
3. Collection variance on inherited claims is excluded from window
   operating profit (reported apart); billing fees on collecting them stay
   window expenses (the fee is a cost incurred in the window).
4. Wait alerts measure from the appointment time (an early arrival waiting
   for a punctual exam is not a problem); the wait metrics report both.

**Outcome neutrality.** Metrics, alerts and summaries change nothing in
the simulation. Checked against `23b6ff8`: ledger digest and every visit,
study and appointment identical after 21 days (established) and 30 days
(mini), seeds 1–3; the pinned start-state digests (ledger and log) are
unchanged. The log gains the alert entries.

**Tests** (`go test ./...`: all packages pass; `internal/sim` 157 s on the
Pi). New, on hand-built micro practices with fixed durations where figures
are worked out by hand, and on the mini and established practices:

| Requirement | Test |
|---|---|
| Exact utilization denominators | `TestMetricsRoomAndStaffDenominators` (XR busy 46 min of 60 open, 34 min after hours; Tess 2h06m on duty, 36m overtime, 96m busy; an unused staffed room is 0%, not undefined) |
| Shared technologist | `TestMetricsSharedTechnologist` |
| Staffed vs open | `TestMetricsStaffedVersusOpen` (staffed = open − breaks) |
| Reading blocks | `TestMetricsReadingBlocks` (and TAT percentiles, censored) |
| Percentiles, empty denominators | `TestNearestRankPercentiles`, `TestMetricsEmptyDenominators`, wait percentiles in the denominators test |
| Partial day, midnight boundary | `TestMetricsPartialDay`, `TestMetricsMidnightBoundary` (only the censored TAT ages, by one second, from 23:59:59 to midnight) |
| Exact-timestamp boundary | `TestEvaluationExactTakeoverBoundary` (10:00 takeover with remittances at that instant; window books = journal after the baseline; orders before 10:00 in the calendar day only) |
| Inherited claims paid after takeover | `TestEvaluationInheritedClaims` |
| Imaged before, finalized after | `TestEvaluationPreTakeoverStudyFinalizedAfter` |
| Capital excluded | `TestEvaluationExcludesCapitalTransfers` (+$350,000 opening cash: identical operations and operating results) |
| Accrual allocation, whole-day window | `TestEvaluationAllocatesTakeoverDayAccruals`, `TestWindowEqualsDaysFromMidnightTakeover` |
| Alerts: trigger, escalate, resolve, dedupe, expire | `TestAlertsTriggerEscalateResolve` (exact times 08:20, 08:50, 09:20, 09:40, 09:43), `TestAlertsExpireAndDeduplicate` |
| Same-instant ordering | `TestAlertSameInstantOrdering` (an exam starting at the threshold instant raises nothing) |
| Daily summary reconciliation | `TestDailySummaryReconciliation` (days sum to since-day-1; cash rolls forward) |
| No future leakage | `TestMetricsAndAlertsRevealNoFuture` |
| Chunking, speed, unrelated events | `TestMetricsChunkingInvariance`; the TUI pacing test; `TestRegressOutcomeIndependentOfUnrelatedEvents` now includes both alert fixtures |
| Session, TUI | `TestEndOfDayStopsAtOperationalClose`, `TestEndOfDaySummary`, `TestDashboardAlerts`; render tests cover finance and history |
| Scenario fields | `TestValidationErrors`: 4 new cases (`evaluationDays`, `alerts` ranges and ordering) |

**Performance** (MEASUREMENTS.md): warm-up 1402 ms vs 1366 ms at `23b6ff8`
(best of four alternating runs, Pi at 81 °C, +2.6%); one observation 1.26
ms vs 1.20 ms; the evaluation window 1.3 ms, on request only. Completed
days' summaries are cached.

**Known limitations.**
* All overtime is required hold-over until actions arrive (step 13); the
  approved/required split is not yet reported.
* Alerts start at the takeover: history days have none.
* Thresholds, insight rules and the backlog limit are illustrative
  (calibration, step 18). Insights are two simple rules.
* A day appears in the history from midnight (before: from accounting
  close); after accounting close today's summary is still "so far".
* The baseline digest covers the baseline record, not the full state (the
  full-state digest is step 14). Scenarios leave `evaluationDays` at 0.
* No turnover state in V0.1, so busy = patient in the room.
* The history screen did not gain alert or wait-percentile columns (the
  summary screen pages through days instead).

---

## Step 11 remediation — review findings (v0.1.0-dev.11.1)

The step 11 review (`docs/reviews/step-11-astra-review.md`, gpt-6-astra at
high reasoning, read-only snapshot of `e5863b4`) returned a CONDITIONAL GO
with six confirmed defects and two test gaps. For each defect a regression
test was written first and run on the unfixed code (`e5863b4`); then the
smallest fix was made and the test rerun.

| # | Defect | Regression test | Before (on `e5863b4`) | After | Fix |
|---|---|---|---|---|---|
| 1 (high) | The takeover adjustment ran before same-instant events: a later-queued day start (payroll or bills) at a midnight takeover, 10:00 remittances, and dispatch. Opening cash came out wrong and the snapshot was taken too early. | `TestRegress11TakeoverCollisions` | FAIL: 5 operating entries after the opening at midnight on payday Fri Oct 16 and on Nov 1; 87 at a 10:00 takeover. Opening cash $100,801 (payroll paid after the injection), $27,142 (bills) and $156,239 (collections) instead of $150,000. | PASS: exactly $150,000; snapshot = books when play begins; the opening is the last entry of the instant | The takeover event only marks the adjustment due. It is posted at the end of that instant's dispatch, after every event and dispatch decision. |
| 2 (medium) | The cash chart and daily records showed pro forma (shifted) history as plain cash | `TestRegress11ActualHistoricalCash`, `TestRegress11CashHistoryLabelled` | FAIL: cash history Sep 14 $195,837, actual −$372,992; screens unlabelled | PASS | Actual cash everywhere by default. `DaySummary.ProFormaCashEndCents` is a separate, named field. The history screen says ACTUAL CASH and shows a takeover line ("owner capital +$568,829 … brought cash from −$418,829 to $150,000"). The chart says "Cash (actual)" and notes the capital at takeover. The ledger is untouched. |
| 3 (medium) | `submitTime`/`paymentTime` of 24:00 accepted, scheduling onto the next (weekend) day | `TestValidationErrors` (2 new cases) | FAIL: no validation error | PASS | Point-in-time financial clocks must be before 24:00. |
| 4 (low) | Ledger hash ignored the fixed-cost item | `TestRegress11LedgerHashCoversItem` | FAIL: same hash | PASS | The item is part of each hashed line. The golden ledger digest changed for this reason only; the log digest is unchanged. |
| 5 (low) | The balance line printed "=" between independently rounded dollars (the reported $1) | `TestRegress11BalanceEquationExact` | FAIL: whole dollars | PASS | The balance and equity lines show exact cents: $428,153.55 = $88,071.43 + $340,082.12. |
| 6 (low) | Dates outside years 0–9999 wrapped (10000 → 0000) | `TestRegress11DateFormattingAllYears` | FAIL: "0000-01-01" for 10000, "000/-12-31" for −1 | PASS | The fast path covers 0–9999; others use the previous `fmt` format. |

**The $1 discrepancy.** It was display rounding. The exact balances at
takeover (seed 1) were assets 42,815,355¢, liabilities 8,807,143¢ and
equity 34,008,212¢: an exact match in cents. The ledger also enforces
Σ = 0 on every posting and in the invariants.

**Test gaps closed**
* `TestRegress11SplitEqualsWholePendingState`: building then playing 2 days
  7 h equals one uninterrupted run. It compares the full pending event queue
  and every claim, appointment, study, visit and order, not only digests.
  This passed before as well.
* `TestRegress11ConvergenceRejectsNaN`: the convergence comparator now
  rejects non-finite ratios (0/0 used to pass). It failed before.
* `TestDrawMemoEqualsFreshHash`: memoized draws equal fresh hashes, with
  independent cursors.
* `TestLogDigestIndependentOfReadsAndEviction`: the hash covers every entry
  regardless of reads and ring eviction, and retained order is kept.
* The last two passed before; they confirm the review's finding that the
  optimizations are sound.

**Not changed (recorded):**
* The convergence test remains a nine-seed smoke test. It has no variance
  estimate, the seeds are unpaired across warm-up lengths, and liabilities
  and backlog are not compared. Step 18 measures variance.
* The review's other suggested fixtures remain future work: takeover during
  required overtime; a recall, no-show rebook and stale arrival at
  takeover; one claim in every state; an overnight TAT in history.

**Owner decisions**
1. **Capital stays explicit** (owner, 2026-10-14). The takeover injection
   is shown wherever opening cash is (briefing, log, finance screen,
   history line) and is never treated as a balancing mechanism. Calibration
   is not assumed to shrink it.
2. **Historical cash** (owner, 2026-10-14): actual by default with the
   injection visible. Pro forma only as an explicitly named value. Done.
3. **Scoring windows (open).** "Today" and "since day 1" are calendar-day
   windows from 00:00. A 06:00 takeover includes the 00:00 accruals; a
   13:00 takeover includes the morning. That is fine for display. For
   benchmark scoring (step 12 onward) the recommendation is to score
   post-takeover performance from the exact takeover instant against the
   opening balance sheet, and to keep capital transfers out of operating
   performance.

Verification: the full suite passes, and all step 8 and step 10
regressions pass.

---

## Step 11 — Warm-up, opening state and performance (v0.1.0-dev.11)

**What was built.**
* `dreamteam` now opens the **established practice** by default (bundled
  scenario `established`; `mini` remains).
* It runs the documented 180-day warm-up with the ordinary engine before
  the TUI starts ("Building 180 days of practice history…") and opens on a
  takeover briefing at Tue Oct 13 06:00, day 1.
* New code:
  * `internal/sim/opening.go`: the opening adjustment and snapshot, and
    financing-shortfall tracking;
  * `history.go`: daily records;
  * `draws.go`: the draw memo;
  * `internal/tui/briefing.go`: the briefing and history screens.
* Design: SIMULATION_DESIGN §8.8; TUI_DESIGN §4.7a.

**Warm-up semantics.** Warm-up and play are one continuous run of the same
engine. Nothing is approximated, skipped or reset. These carry over as the
history left them:
* appointments (868 booked from day 1 for seed 1) and the recall list
  (48);
* pending orders and patients on site;
* studies on the worklist and reads in progress;
* claims at every stage, receivables, payables and accrued payroll;
* staff weekly hours, for overtime;
* ID counters.

**Warm-up accounting conventions**
* Cash is 0 at the epoch, and the history's collections, payroll and bills
  accumulate.
* At the start time one journal entry, the **opening equity adjustment**
  (Dr/Cr Cash ↔ Opening equity), sets cash to the scenario's $150,000.
  This is the documented convention (§8.4): owner capital in, or a
  distribution out. It is not revenue or a cash flow.
* Nothing else changes. A/R, A/P, accrued payroll and every open claim
  continue, and future payment events stay queued. The history's operating
  result stays in equity, so losses are not erased.
* Seed 1 at takeover:

| | |
|---|---|
| Cash left by the history | −$418,829 |
| Owner capital added | $568,829 |
| Opening cash | $150,000 |
| Receivables (1,870 open claims) | $278,154 |
| Payables | $48,933 |
| Payroll owed | $39,139 |
| History result (180 days) | −$223,758 |
| Last 30 days: net revenue / expenses / profit (margin) | $192,344 / $222,591 / −$30,247 (−15.7%) |
| Balance sheet at takeover | assets $428,154 = liabilities $88,071 + equity $340,082 |

* The capital is large because the uncalibrated history loses money.
  Alternatives are listed under the decisions below.
* **Revenue recognized vs net revenue.** Revenue recognized is the expected
  net booked when each report is finalized. Net revenue adds the collection
  variance on claims that settled in the period. In step 10's 180-day run
  net revenue ($1,136,003) exceeded revenue recognized ($1,125,060) because
  the claims settled so far paid $10,943 more than expected: fast payments
  in full settle first, while denials and self-pay write-offs settle 30–120
  days later. The TUI rows are now "Revenue recognized", "+ collected vs
  expected" and "Net revenue", with a note.
* **Financing shortfall.** Negative cash stays allowed, as approved, but
  it is now tracked from the start:
  * the lowest balance and when it happened, and days ended below zero;
  * one alert per dip below zero, and a red **FINANCING SHORTFALL** on the
    finance screen and the dashboard.

  No credit line or bankruptcy is modelled.
* **Underlying categories.** Fixed-cost ledger lines carry their item, so
  every period reports fixed cost by item (rent, utilities, insurance, each
  lease and service contract, IT) whatever account it posts to.
* **History boundary.**
  * Days before the start date are history: daily records, ledger and log.
  * Today's counters, demand and statements since day 1, cash change today
    and the financing figures start at day 1. Day 1's 00:00 accruals are
    day 1's.
  * The opening adjustment never appears as today's cash change.
  * The cash history and day records restate the days before takeover by
    the adjustment, so the series is continuous. This is labelled.

**On screen**
* **Briefing** (`b`):
  * money: opening cash, owner capital, last-30-day P&L and cash flow,
    carried-over receivables, payables and payroll, next payroll and bills,
    and the financing note;
  * patients: last 7 days of demand and exams, waits, utilization of open
    and staffed hours, bookings ahead;
  * worklist and TAT;
  * today's roster.
* **Daily history** (`h`): 30 days, with history rows marked "·".
* **Finance:** a balance sheet line with the equity split, a financing
  line, and the clarified revenue rows.

**Determinism and correctness tests** (`internal/sim/warmup_test.go`):

| Requirement | Test |
|---|---|
| Opening balance sheet reconciles | `TestOpeningBalanceSheet`: one opening entry, at the start, after every warm-up entry; adjustment = target − cash before; cash = $150,000; assets = liabilities + equity; equity components match the journal; A/R, A/P and payroll carried over and non-zero; day-1 "today" figures empty; history ends the day before the start, restated to the opening cash |
| Claims, payments, reports, worklist, appointments, weekly overtime survive | `TestStateSurvivesTheBoundary` (13:00 start, so patients on site and the worklist cross the boundary). Every study pending at the start is finalized once. All of the 1,000+ claims open at the start settle within 145 days (the longest path). Warm-up bookings resolve. Riley's weekly paid time at the start equals Monday's pay exactly. |
| Warm-up vs uninterrupted run; chunking; unrelated events | `TestWarmupDeterminism`: a daily-chunked warm-up, a warm-up with live and stale noise events, and a rebuild give identical log and ledger hashes and identical observations; build then 3 days of play = one uninterrupted run; seed 2 differs |
| Golden start state | `TestStartStateGolden` (ledger and log digests, seed 1) |
| No hidden future in the opening state | `TestOpeningStateHidesTheFuture`: forcing every patient from day 2 to no-show and every claim adjudicated after the start to fail leaves the observation, the 180-day history and tomorrow's schedule unchanged; afterwards the books differ |
| Convergence (ROADMAP) | `TestWarmupConvergence`: nine-seed means, 180 vs 365 days. Forward fill to 90 days, A/R and A/R ≤ 60 days within ±5% (largest: 7–30-day fill −4.6%); tails within ±25%. A 90-day warm-up fails (−12.6% fill, no A/R over 90 days). |
| Financing shortfall | `TestFinancingShortfallTracked`: $10k opening cash; an alert per dip; lowest cash and days below zero agree with the daily history |
| Optimizations change nothing | Formatter equivalence tests against `fmt`. Before the step-11 behavior was added, a full warm-up plus 3 days produced byte-identical log text (61,226 entries) and ledger (`a37b36…`) to `32313ac`. |

All step 1–10 tests pass, including the step 8 remediation regressions and
the step 10 finance tests. The full suite takes about 2 minutes on the Pi;
the convergence test is 30 s of that and is skipped with `-short`.

**Performance (before / after).** Warm-up build, alternating old and new
with cool-downs:
* step 10: 1.80–2.38 s/op, 111.6 MB, 1.97M allocations;
* step 11: 1.35–1.97 s/op, 89.7 MB, 1.26M allocations.

That is about 22–25% faster at the median. The spread is thermal: the Pi
ran at 75–79 °C with throttling, and unthrottled builds take 1.34–1.35 s.
Details and the remaining profile are in MEASUREMENTS.md:
* What changed: lazy log time text, a streamed log hash, Sprintf-free
  ID and time formatting, a per-day memo of keyed draws (30% were
  repeats), the claim draw kept on the claim, and reused room staffing
  lists.
* What didn't: no randomness semantics, no event suppression and no
  concurrency.

**Remaining limitations**
* **Target ≤ 1.5 s only when unthrottled.** Throttled builds take
  1.7–1.9 s. The next steps are a lazy structured log, or the start-state
  cache once snapshots exist (step 14).
* **The worklist is empty at 06:00 on day 1.** Monday's studies are all
  read by Monday evening at the current (below-design) volume; the design's
  10–15 unread is a calibration matter (step 18).
* **Large owner capital.** The history loses money, so takeover needs
  $569k of owner capital. This is expected to shrink to a working-capital
  figure after calibration.
* **Utilization is minimal.** The daily records keep only exam time over
  open and staffed hours; the full metric set is step 12.
* **No financing model.** No credit line, interest or insolvency (V0.2).

**Decisions for owner approval**
1. **Opening capital convention** (implemented as documented): one
   adjustment at takeover, with the history's balances and losses kept.
   Alternatives:
   * (a) opening cash at the epoch and no adjustment, so the start cash is
     what the history leaves (−$268,829 for seed 1);
   * (b) no opening capital (−$418,829);
   * (c) keep the adjustment and calibrate so the history is about
     break-even (step 18).

   Recommendation: keep the adjustment, aim for (c).
2. **Restated cash history.** Before day 1 it is shown as if the opening
   capital had always been there (continuous series, labelled). The
   alternative is the raw series with a jump at takeover.
3. **The log digest format changed** (streamed SHA-256). Digests differ
   from step 10 by construction; outcomes do not.

---

## Step 10 — Ledger, claims, payroll and costs (v0.1.0-dev.10)

**What was built.**
* A double-entry ledger in integer cents (`internal/sim/ledger.go`).
* Revenue at report final, the claim lifecycle, payroll, and fixed and
  variable costs (`finance.go`).
* Statements, claims and the journal in the API (`finance_view.go`,
  `api.Finance`), and the TUI finance screen.
* All the assumptions are scenario data: the new `finance` section and
  catalog `baseRateCents` / `supplyCostCents`, taken from §8.2–8.3 of the
  design. They are illustrative and not tuned. Design detail is in
  SIMULATION_DESIGN §8.7.

**Accounting conventions**
* **Accrual for profit, cash for liquidity, one ledger.** Revenue is earned
  when the radiologist finalizes the report: Dr A/R / Cr revenue at expected
  net = base × payer multiplier × NCR. It is never earned at booking,
  arrival, registration, imaging, worklist entry or reading start. Cash
  moves only when a payment arrives or payroll or bills are paid. The
  practice can be profitable and short of cash at the same time.
* **Every entry** balances and is non-empty, and has a stable idempotency
  key (study, claim and payment number, employee and day, or day). It is
  never edited. Corrections are linked reversing entries, which can be
  posted once and never reversed themselves. The incremental check re-sums
  new entries independently, the day books must equal the balances, and
  the accounts sum to zero.
* **Claims.**
  * Pending, then submitted 1–3 business days after final, at 17:00 in one
    batch. Remittance comes after the payer's delay, on a business day at
    10:00.
  * Outcomes:
    * paid in full;
    * short paid at m ± 10 pp;
    * denied, then resubmitted the next business day, then recovered or
      written off 30–60 days after the denial;
    * self-pay unpaid, written off and not counted as a denial.
  * Settlement sends expected − paid to collection variance (contra
    revenue), so a closed claim has zero A/R.
  * Outcome, delay, short fraction, rework time and charge lag come from
    the order's origin key (`claim`, `claim.lag`). The payer class is drawn
    when the order is disclosed.
* **Payroll.**
  * Hourly staff accrue when they go home: worked time minus breaks taken,
    with skipped breaks and held-over time paid. Pay is at the regular rate
    until the week reaches 40 paid hours, then ×1.5 (the step 8 rule).
  * Salaried staff accrue annual ÷ 365 per calendar day. Radiologists
    accrue × their site allocation.
  * Benefits are 25% on everything.
  * Payroll is paid every other Friday from Oct 16 at 00:00 and covers
    everything accrued through Thursday.
* **Costs.**
  * Fixed monthly items accrue daily with exact monthly totals.
  * Supplies (contrast included) and $3 PACS post at acquisition.
  * Billing is 5% of each payment.
  * Accounts payable is paid on the 1st.
* **Statements.**
  * Today, month to date, trailing 30 days and since day 1.
  * Net revenue = revenue − collection variance.
  * Expenses are shown by account, and also as labor, fixed and variable.
  * Operating profit and margin; the margin is undefined at zero revenue.
  * Collections, disbursements and net cash flow.
  * Denial rate, days in A/R and A/R aging.
  * Revenue and direct cost by modality, plus a cash history.

**On screen**
* `f` finance has three views, switched with `tab`:
  * **overview**: periods side by side, cash and the next payroll and
    bills, A/R and denials, the modality table and a cash sparkline;
  * **claims**: open claims, oldest first;
  * **journal**: the latest entries as debit and credit lines.
* The dashboard's second line shows cash ▲/▼ today, revenue, expenses and
  operating profit today, and A/R.
* The report-final log line now ends "· revenue $395".
* The log adds lines for claim batches, payments, short payments, denials,
  write-offs, payroll and bills.
* Patient journeys continue through revenue, claim, denial and
  resubmission, and payment or write-off.

**Tests** (`internal/sim/finance_test.go`, `ledger_internal_test.go`, a TUI
test, and scenario validation cases):

| Requirement | Test |
|---|---|
| Revenue once, only at final; none for cancelled, no-show, left, lost, rescheduled-unperformed or unread exams | `TestRevenueOnlyAtReportFinal`: minute by minute on day 1, then 30 days; non-vacuity checks for each excluded state |
| Correct modality and payer reimbursement | `TestFinanceArithmetic` (NCR 0.950/0.980/0.934/0.400; CT w/ contrast, commercial = $416 allowed, $395.20 expected), `TestClaimLifecycleAndTiming` (every claim's allowed and expected amounts) |
| Claim lifecycle, payment timing, denials, receivables | `TestClaimLifecycleAndTiming`, 150 days. For every claim, the drawn charge lag, remittance date, rework date, state and paid amount match exactly. Non-vacuous for paid, short, denied → paid, denied → written off, self-pay unpaid, open and pending. A/R per claim and cash are rebuilt from the journal alone. |
| Collection variance mean ≈ 0 (ROADMAP) | `TestCollectionVarianceMeanZero`: 30,000 draws per payer; the collected fraction is within 0.6 pp of NCR |
| Balanced entries, idempotency, reversals | `TestLedgerPostingRules`: unbalanced, empty, duplicate, out-of-order and bad lines are rejected with no effect; reversal mirrors and links; no double reversal; tampering is detected. Every simulation test with invariants re-sums the journal. |
| Cash and receivable reconciliation | `checkFinance` invariants (cash = opening + receipts − disbursements from independent totals; A/R = open claims; revenue = Σ expected; claims = final reports; accrued payroll and A/P), checked after every transaction in all invariant-checked tests and at each day boundary in the finance tests |
| Payroll, regular and overtime; weekly threshold crossing | `TestPayrollWeeklyOvertime` (`micro-week.json`): Tue–Thu $630; Fri crosses 40 h after 8.5 h, $690; Sat all premium, $360; Monday resets, $630; unpaid lunch excluded; benefits 25% |
| Required overtime after scheduled close | `TestPayrollRequiredOvertime`: wages = scheduled part + exactly the step 8 overtime cost (36 min held over: $126 = $90 + $36; premium fixture $189 = $135 + $54); one entry per employee and day |
| Fixed and variable cost timing; day, week and month boundaries | `TestCostAndPaymentTiming`, Oct 13 to Dec 1: one fixed accrual per day at 00:00; November totals exactly the monthly sum; A/P paid Nov 1 and Dec 1 equals the balance before; supplies and PACS once per study at acquisition; billing fee = 5% of each payment; payroll only on the four anchored Fridays, equal to the accrual through Thursday; salaries once per employee and day; wages on the day worked |
| Same-instant ordering | The same test checks that at 00:00 bills and payroll are paid before that day's accruals. Claim batches and remittances are class 6, after operations at the same instant. |
| Sparse events, speed, replay | `TestFinanceDeterminism`: 75 days in one jump, daily, or in 13-minute chunks give identical ledger and log hashes. A replay is identical; a different seed differs. |
| Independence from unrelated events | `TestFinanceUnaffectedByUnrelatedEvents` (live and stale events at 10:00, 17:00 and around them for 50 days); the step 8 wake-completeness tests now also compare the ledger hash |
| Outcomes independent of operations | `TestClaimOutcomesIndependentOfOperations`: worklist FIFO vs shortest-first changes finalization times, but no payer, amount or outcome |
| Hidden outcomes not leaked | `TestFinanceOutcomesNotLeaked`: forcing every claim to fail leaves the full observation identical for 15 days (and all journeys on day 15), then changes cash. The latent-field deny-list now covers `api.Finance`. |

Mutation checks (each made the named test fail, then was reverted):
* revenue at exam end;
* claim draw keyed by processing order;
* weekly threshold ignoring earlier days;
* held-over time unpaid;
* payables paid on the 2nd;
* charge lag in calendar days (caught once the test checked the exact
  drawn lag);
* an observation revealing a forced outcome;
* a wrong short-paid amount;
* a cash identity off by a cent.

All step 1–9 tests pass, including the step 8 remediation regressions. The
fixtures gained rates and a `finance` section, but no expectation changed.

**Measured: 180 days of the mini practice, seed 1** (2.1–2.5 s/op on the
loaded Pi, MEASUREMENTS.md):

| | Since day 1 (181 days) | Trailing 30 days |
|---|---|---|
| Revenue earned (expected net) | $1,125,060 | $195,429 |
| Collection variance (favorable) | +$10,943 | +$2,429 |
| Net revenue | $1,136,003 | $197,858 |
| Operating expense | $1,336,861 | $222,177 |
| — labor (wages $508,015 + benefits $127,005) | $635,020 | $103,677 |
| — fixed | $529,192 | $86,692 |
| — variable (supplies $108,567, PACS $58,461, billing $41,498) | $172,649 | $31,809 |
| Operating profit | −$200,858 | −$24,319 |
| Operating margin | −17.7% | −12.3% |
| Cash collected | $829,949 | $179,528 |
| Disbursements | $1,260,067 | $223,235 |

* At the end: cash −$280,118 (opening $150,000); A/R $306,054 (46 days);
  A/P $43,247; accrued payroll $33,547.
* Denial rate: 5.9% (309 of 5,199 insurance adjudications).
* Claims: 7,514. Paid 4,935, short paid 249, written off 270, open 2,060.
* By modality (net revenue / direct cost):
  * XR: 3,538 reports, $182,866 / $34,074;
  * US: 1,216 reports, $213,322 / $30,321;
  * CT: 1,833 reports, $436,681 / $180,790;
  * MR: 927 reports, $303,134 / $223,453.
* Hot spots: SHA-256 22%, dispatch 23%, log formatting 14%, finance about
  10% (adjudication 4%, recognition 3%, posting 2%).

**Why the margin is negative.** Volume, not the financial assumptions.
Monthly-equivalent figures:
* Labor ($106.7k vs the reference $105.1k) and fixed costs ($88.9k vs
  $88.5k) match the §8.6 reference calculation.
* Net revenue is $190.8k vs $261.5k. That is about 1,260 reports a month
  instead of 1,650, the volume shortfall noted since step 7. Revenue per
  report is $151 vs $158.5, from the exam mix.

Nothing was tuned toward the 8.5% target, as instructed. That is step 18.
The cumulative collection variance is favorable (−1.0% of revenue) because
claims that settle early are mostly paid in full. Denials and self-pay
write-offs settle 30–120 days later. Over all outcomes the mean is zero,
as the draw test shows.

**Known limitations**
* The negative margin and cash are a volume artifact until calibration
  (step 18).
* Opening cash is posted at the epoch. The warm-up opening-equity
  adjustment at the start time is step 11.
* No credit line or interest: cash simply goes negative.
* No copays or deductibles (not specified). Bad debt and denials share
  collection variance (split in V0.3).
* No holidays; business days are Mon–Fri.
* No prepaid expenses (rent is paid in arrears on the 1st).
* No payroll tax remittance schedule separate from payroll.
* No reversal flow in V0.1 operations. The mechanism exists and is tested.
* The end-of-day summary's finance line comes with the summary screen
  (later step).
* Performance: finance adds about 0.3–0.4 s per 180 days. The step 11
  warm-up target now depends on the planned mitigations.

**Decisions for owner approval** (each is scenario data or a small,
documented rule; SIMULATION_DESIGN §8.7):
1. **Rework timing.** "Resubmitted after ReworkDelay" is read as denial →
   final outcome in 30–60 days, with the resubmission on the next business
   day.
2. **Remittance clock.** Business days only; submissions in a 17:00 batch,
   payments, denials and write-offs at 10:00.
3. **Self-pay.** A statement, not a claim. Non-payment is written off at
   the due date, never reworked, and excluded from the denial rate.
4. **Chart mapping.** Utilities and insurance go to *Rent & facility*; the
   monthly IT/RIS charge to *IT / PACS*; leases and service to *Equipment
   service & leases*, tagged by modality. No new accounts.
5. **Payables.** All of them (rent included) are paid in arrears on the
   1st, because the chart has no prepaid account. Payroll is paid at 00:00
   on the anchored Friday (Oct 16, then every 14 days) and covers what was
   accrued through Thursday, with benefits paid alongside.
6. **Salaried accrual.** Per calendar day (annual ÷ 365), weekends
   included. Radiologists × site allocation.
7. **US base rates** within the documented $140–170: abdomen $170, pelvis
   $155, venous $160, thyroid and renal $140.
8. **Negative cash** is allowed and shown in red (no financing in V0.1).

---

## Step 9 — Worklist, reading and turnaround (v0.1.0-dev.9)

**Behavior**

* **Studies**: acquired at exam end (imaging complete, TAT start), on the
  worklist once the technologist's documentation is done, then read, then
  final. There is one study per imaged visit, never a report before imaging,
  and never a second finalization.
* **Radiologists read** during their site reading blocks while on regular
  duty. Remote radiologists read too. Each claims the first study in worklist
  order whose longest read (p100) ends by their scheduled shift end, so
  reading never extends a shift or crosses midnight. One read at a time.
  Reads that run past a block's end finish, and the overrun delays the next
  block (§6.7, with a wake-up so the claim happens on time).
* **Supervision** stays presence-based and compatible with reading, so a
  radiologist can read while supervising. The supervisor of an exam in
  progress stays until it ends (step 8 remediation). A remote radiologist
  never supervises. A held-over radiologist claims no new reads.
* **Read times** are per case (catalog `readSec`, keyed by the order's
  origin), so worklist order and policy never change them.
* **Worklist order** (default urgent then FIFO): urgent studies and studies
  already past their TAT target first (no starvation), then time on the
  worklist, with the study ID as the final tie-break. FIFO, oldest-first and
  shortest-first (by the published expected time) are ready for the step-13
  action.
* **Turnaround**: TAT = imaging complete → report final. The figures shown
  are median and p90 over studies acquired today with unfinished ones at
  their age (censored), worklist wait, read time, backlog by modality and
  priority, the oldest unfinished study, and breaches. Targets: urgent 60
  min and routine 24 h (documented), soon 4 h (illustrative). Overdue
  urgent and soon reports are logged once.
* **After operational close** radiologists keep reading in their blocks.
  Operations close without waiting for reports, and the backlog carries into
  the next day (Saturday's afternoon studies are read on Monday).

**On screen**: `w` worklist (reading and unread studies, turnaround,
backlog, latest reports; `enter` opens the patient), a dashboard reporting
line, radiologists shown as reading / ready to read / other duties /
supervising, and journeys that continue to the final report and its TAT.

**Tests** (`internal/sim/reading_test.go`, new micro fixtures):

* A month of the mini practice:
  * exactly one study per imaged visit;
  * timestamps ordered, no duplicate finalization;
  * every claim inside the claimer's site block;
  * every read ends by shift end;
  * no radiologist reads two studies at once;
  * backlog crosses days.
* Priority and ties: the urgent study is read first, and two studies
  entering at the same instant go in ID order.
* TAT definition (acquired 08:23, final 08:29 → 6m).
* Overrun delays the next block to 09:51 (claim exactly then); a read ending
  exactly at block end owes nothing (claim at 09:40, same-instant ordering).
* Radiologist distinctions (ROADMAP): remote reads, never supervises; on-site
  in other duties supervises, claims no reads.
* Read time unchanged when the policy reorders the worklist. A deliberate
  dependency on claim time makes this test fail.
* Observation types carry no latent fields.
* Outcomes unchanged by injected live or stale events (now including the
  reading fixtures).
* All step 8 remediation tests, chunking and speed tests pass.

**Measured**: 180 days of the mini practice in 1.7–1.9 s on the loaded Pi:
7,528 studies, 7,514 final, 14 outstanding at the end. Reading is not a hot
path. See MEASUREMENTS.md.

**Limitations**

* Volumes are below calibration, so the reading backlog is small. Weekday
  TAT medians run 20–40 min, and Tuesday mornings start nearly clear (the
  design expects 10–15 unread from warm-up at full volume, steps 11 and 18).
* No report interruptions, addenda or protocoling (V0.2).
* Revenue at finalization is step 10.

---

## Step 8 remediation — review findings 1–10 (v0.1.0-dev.8.1)

The step 8 review (`docs/reviews/step-08-astra-review.md`, gpt-6-astra
at high reasoning) found 10 defects, all confirmed. For each, a regression
test reproducing the reviewer's scenario was written first and run against
the unfixed code (`ee26417`), then the fix was made and the same test
rerun. Tests are in `internal/sim/regress_test.go`, with fixtures in
`internal/sim/testdata/micro-*.json`.

| # | Defect | Regression test | Before (on `ee26417`) | After | Resolution |
|---|---|---|---|---|---|
| 1 | Held radiologist released while supervising; colleague kept past her cap | `TestRegress01RadiologistHandoverCap` | FAIL: Dr. Bo overtime 0s (want 33m), Dr. Rae 3m (want 0, cap 09:30) | PASS | The supervising radiologist is recorded at exam start and stays until the exam ends. A waiting contrast patient keeps a radiologist only if she could supervise in time and no on-site colleague on regular duty could. New invariant: every contrast exam has its supervisor on duty, on site, within cap. |
| 2 | Held employee kept past usefulness and cap (no wake) | `TestRegress02HeldPastUsefulnessAndCap` | FAIL: Tess overtime 45m, want 0 | PASS | "Needed" now uses the patient's earliest possible start and the full remaining task. A single deadline wake fires when a held employee stops being useful or a patient stops being servable. |
| 3 | Desk staff who can't fit a registration block self-registration; patient stranded | `TestRegress03DeskCannotBlockSelfRegistration` | FAIL: panic, 1 visit still on site at accounting close | PASS | Desk and ops staff block self-registration only while on regular duty and able to fit a registration. They never start registrations after shift end. The deadline wake covers the moment they stop fitting. |
| 4 | Last technologist released while their patient is registering | `TestRegress04HoldForRegisteringPatient` | FAIL: patient rescheduled "no XR technologist for the rest of today" | PASS | Registering patients count as outstanding for hold-over. |
| 5 | Cap at midnight: documentation ends at 00:00, after day start; overtime lost | `TestRegress05MidnightBoundary` | FAIL: overtime 0m recorded, want 15m | PASS (scenario now rejected: cap must end by accounting close) | Day boundary: shift start + max hours must be ≤ 23:59:59. Accounting close sends home anyone past shift end (overtime booked to that day) and fails loudly if anyone is busy. The tradeoff against carrying work across midnight is documented in SIMULATION_DESIGN §6.9. |
| 6 | FirstFit skipped an earlier preferred slot in a later room | `TestRegress06FirstFitPreferredAcrossRooms` | FAIL: booked room 0 at 15:00, want room 2 at 13:00 | PASS | The early exit compares any-window and preferred candidates separately. |
| 7 | Shared capacity checked only at slot start | `TestRegress07SharedCapacityAcrossSlot` | FAIL: XR booked 09:00 on day 7 although its technologist is booked in CT 09:30–10:00 | PASS (XR goes to the next free day) | Slot technologist sets cover the whole slot. `sharedFree` checks every constant-coverage stretch of the slot. |
| 8 | Opening schedules bypassed shared capacity | `TestRegress08OpeningRespectsSharedCapacity` | FAIL: no validation error | PASS | `sim.CheckOpening` books the opening schedule in order under the same rules, and scenario validation reports problems by JSON path. The mini opening Tuesday moved two CT appointments (08:00 → 09:15, 09:00 → 11:00), and `micro-shared.json` now alternates rooms. |
| 9 | A second break overwrote a deferred first break | `TestRegress09DeferredBreaksDoNotOverwrite` | FAIL: break taken 15m, want 30m | PASS | A break falling due while busy or on break waits its turn after the current one. An older waiting break is skipped and counted, never silently dropped. |
| 10 | Negative wait for patients not yet imaged | `TestRegress10WaitWhileRegistering` | FAIL: registering, waited -480 min | PASS | Waited runs to now while waiting, to exam start once imaged, and to departure otherwise. |

**Missing-wake detection.** `TestRegressOutcomeIndependentOfUnrelatedEvents`
reruns ten micro practices with no-op events (handled, so dispatch runs)
every 7 minutes, and with stale events. On `ee26417` four of them changed
outcome (`micro-cap`, `micro-contrast-cap`, `micro-cap-wake`,
`micro-desk-cap`). After the fixes none do. A four-day version on the full
mini practice with invariants on (`TestRegressMiniUnaffectedByUnrelatedEvents`)
passes. It also passed on `ee26417`: the mini practice rarely reaches these
edges, which is why the micro fixtures exist.

**Expected changes to existing tests.** In `TestCoverageExhaustedReschedules`
the patient is now rescheduled at 09:09:01, the second their last feasible
start passes (cap 09:30 minus 21 min). Before, it happened at the next
unrelated event (09:14). The contrast hold-over now logs "supervising a
contrast exam". `TestSharedTechnologistContention` samples at 08:20 because
its bookings now alternate rooms.

**Verification.** The full suite passes. That covers the step 8 closing
tests (acceptance criteria rechecked: patients served after scheduled
close, required overtime costed, explicit coverage reschedules, contrast
after hours, operational close after scheduled close), month-long
invariants, minute-by-minute chunking, and identical results across speeds
and event budgets. 180 days of the mini practice take 1.4–2.0 s on the
loaded Pi (MEASUREMENTS.md, step 11 risk).

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
