# DreamTeam — Release notes

Newest first. Step-by-step detail is in `docs/PROGRESS.md`; the acceptance
evidence in `docs/ACCEPTANCE.md`.

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
