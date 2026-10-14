# DreamTeam — Progress

One section per roadmap step (ROADMAP §2), newest first. Each records what
was built, what the player can see, how it was tested, and known
limitations. The version shown in the TUI top bar (`DREAMTEAM v0.1.0-dev.N`)
and by `dreamteam version` names the latest step a build contains.

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
