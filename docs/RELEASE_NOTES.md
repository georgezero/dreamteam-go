# DreamTeam — Release notes

Newest first. Step-by-step detail is in `docs/PROGRESS.md`; the acceptance
evidence in `docs/ACCEPTANCE.md`.

---

## V0.2.0 — practice design

You can now change how the practice is staffed and read, and see whether
your plan is better than doing nothing.

**Five staffing and reading actions** (staff screen `m`, or batches in an
actions file). Each one is validated and previewed, and it replays exactly.
* **Hire** from a fixed offer table, at a standard or premium rate. The
  premium buys a shorter recruiting lead. The recruiting fee, paid
  onboarding and 40 hours of training (24 for the desk) come before any
  patient capacity.
* **Terminate**, with at least 14 days' notice at contract pay and two
  weeks' severance.
* **Set rosters**: dated weekly or fortnightly patterns, from a Monday at
  least 14 days ahead.
* **Set a radiologist's site reading allocation** (30–100%), from at least
  14 days ahead.
* **Contract teleradiology** (`standard-v1`): weeknight, Saturday and
  Sunday windows, urgent or all studies, monthly retainers and per-read
  fees, and a 30-day minimum term on cancellation.

Batches are atomic. A refused action carries a stable code and changes
nothing. Displaced bookings are rebooked or lost, and the preview names
them first.

**Employment obligations.**
* Contracts are inferred for the existing staff at the takeover.
* Contracted hours are guaranteed, whatever is rostered.
* Weekly overtime premium applies above 40 actual hours.
* Severance and vendor costs are posted and settled in the double-entry
  ledger.

**The `sustainable` scenario.**
* The established practice at ×1.30 referral demand, with 12-minute X-ray
  slots and a full-time evening MR technologist.
* Shared XR/CT staffing, near break-even (≈+0.5% mean T90 margin).
* Reorganizing it is the game. The measured lead staffing earned ≈+2.8% at
  this demand, but getting there costs recruiting, notice, severance and
  training, and is not promised.

**The T90 evaluation** (summary screen `e`, `tab` twice, or
`dreamteam t90 FILE`).
* Your decisions before day 90 are judged against no action, on the same
  scenario and seed, by the frozen evaluation policy in force,
  v0.2-eval-4.
* **The metric vector** shows profit, margin, cash, the funding gap, served
  and lost demand, and report breaches and turnaround by priority. Your
  plan, the baseline and the difference are side by side, with no weighted
  score.
* **The service gates** show pass, fail or insufficient against the
  policy's limits, with their counts.
* **The plan gates and the policy's verdict** follow, with its reasons.
* Days not yet played are marked as projected, the post-day-90 tail as
  fixed-policy, and unfinished work as incomplete.
* The report is the canonical evaluation runner's, byte for byte.

**Sessions.**
* Session v2 records action batches with their results.
* Saves from V0.1 and from every V0.2 development build replay with
  their digests verified.
* The V0.1 golden references are unchanged.

**Known limitations** (`README.md`).
* TUI actions are committed one at a time, without showing the stable
  code; rostered and worked hours are not in the staff table.
* The T90 evaluation covers the sustainable scenario only.
* The Pi 4's 90-day headless run (≈3.3–3.9 s) keeps V0.1's waiver.
* No operations-control actions, disruptions or agent interface yet: V0.3
  is next.
* The clinical candidate policy v0.2-eval-5 completed its confirmatory
  evaluation with a NO GO and is not part of this release
  (`docs/calibration/clinical-v5-closure/`).

**Validation** (`docs/v02-acceptance/`).
* All five actions are verified through the simulator, API, sessions,
  headless runs and the TUI.
* Save, replay and determinism hold across amd64 and arm64.
* All 7 golden references and all 25 earlier session fixtures pass.
* The race detector finds no race.
* 150 × 90-day runs check every invariant after every transaction; 720-day
  turnover and roster churn pass.
* The full uncached suite passes.

---

## V0.1.0 — first playable release

A deterministic simulation and terminal game of an outpatient radiology
practice. You take over a struggling imaging center on a Tuesday morning
and run it.

**Simulation.** A discrete-event engine with keyed, platform-independent
randomness (SHA-256): the same scenario, seed and actions give the same
outcome on every machine, at any speed and any frame rate. Referrals and
walk-ins, booking against room templates and real technologist coverage,
arrivals, no-shows and cancellations, registration, imaging, a radiology
worklist with turnaround targets, contrast supervision, closing rules
(admitted patients are always served, with costed overtime). A 180-day
warm-up gives the practice a history before the takeover.

**Terminal UI** (Bubble Tea). Dashboard, schedule, patients and journeys,
worklist, finance (overview, claims, journal), event log with filters,
takeover briefing, daily history, end-of-day summary, actions and help.
Five speeds from 1 min/s to Max, step to the next event, auto-pause on
alerts. Layouts from 80×24 to 160×48; honors `NO_COLOR`.

**Three management actions.** Extend a shift (approved overtime, today or
ahead, opening bookable slots); walk-in policy (accept always, if a gap,
until a cutoff, or decline with an appointment offer); worklist policy
(urgent & overdue then FIFO, FIFO, oldest first, shortest expected read —
urgent studies always first).

**Finance and metrics.** Double-entry ledger with accrual P&L, cash and a
balance sheet; claims with payer lag, denials and rework; payroll and
overtime; fixed and variable costs. Operational metrics (utilization,
waits, TAT by priority, lost demand, walk-in cohorts), alerts that say
why, and an evaluation window from the exact takeover.

**Sessions and replay.** Save and load (`ctrl+s`, `ctrl+o`); a session is
its scenario, seed and actions, and loading replays it from scratch,
checking five digests (log, ledger, actions, state, run).

**Headless CLI.** `dreamteam run` (text or canonical JSON results,
`-actions`, `-save`, `-log events.jsonl`, `-verify-replay`), `replay`,
`verify`, `scenario list|validate|coverage`; golden references with
explicit, reviewed regeneration; documented exit codes.

**Scenarios.** `established` (default): the struggling, capacity-constrained
practice — demand ≈1,650 a month, ≈1,300–1,380 exams, operating margin
around −12% to −14% (nine-seed characterization in
`docs/calibration/`). `mini`: the same practice without history.

**Platforms.** Linux and macOS with Go 1.26 or later. CI: linux/amd64,
linux/arm64, darwin/arm64, with the golden runs byte-identical across all
three. Developed on a Raspberry Pi 4.

**Validation.** Invariants after every transaction for 50 seeds × 90 days;
1,000 random chunkings of the real simulation; TUI sessions reproduced
headless; leakage tests (no hidden future in observations or results).

**Known limitations.** One practice and three actions: no hiring, roster
or template editing, equipment failures, absences, rescheduling or
overbooking. Urgent studies acquired after the last reading block wait
for the next morning or Monday, with no action yet to add evening or
weekend reading. No mouse, no file picker.

**Accepted deviations from the V0.1 criteria** (owner decision):
* On a Raspberry Pi 4, `dreamteam run -days 90` takes ≈3.3 s against a
  2 s target (its CPU lacks SHA-256 instructions); interactive targets are
  met (warm-up 1.44–1.49 s, TUI 7.4–8.2% of a core at 15 min/s, Max-speed
  frames under 100 ms). A Cortex-A76 meets every target. The RNG is kept.
* The default practice's takeover worklist is small or empty (it is
  imaging-constrained, not reporting-constrained); the 5–20-study target
  is deferred to a future reporting-constrained scenario.

**Next (V0.2).** A better-staffed, profitable scenario and more
management actions — hiring, roster adjustments, and evening and weekend
radiologist coverage.
