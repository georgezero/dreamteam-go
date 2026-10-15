# DreamTeam

A deterministic simulator and management game of an outpatient radiology
practice: patients, referrals and walk-ins, technologists and rooms (XR,
US, CT, MR), radiologists reading a worklist, claims, payroll and cash. You
take over a struggling imaging center on a Tuesday morning and run it from a
terminal UI; the same engine runs headless for research and, later, as an
AI-agent benchmark.

Status: **V0.1** (`0.1.0`). Release notes: `docs/RELEASE_NOTES.md`. Everything is synthetic:
patients, staff, referrers, rates and costs are illustrative assumptions,
not operational data. DreamTeam simulates operations; it makes no clinical
decisions and produces no clinical content.

## Quick start

Requires Go 1.26 or later. Runs on Linux and macOS (developed on a
Raspberry Pi 4; CI covers linux/amd64, linux/arm64 and darwin/arm64).

    go run ./cmd/dreamteam                 # play: the established practice, seed 1
    go run ./cmd/dreamteam -seed 7         # another seed: same practice, different patients
    go build -o dreamteam ./cmd/dreamteam  # or build once

The game opens paused on the takeover briefing (Tuesday 06:00). Press
`enter` for the dashboard and `space` to start the day. Press `?` at any
time for the keys.

## Controls

| Key | Action |
|---|---|
| `space` | run / pause |
| `1`–`5` | speed: 1 min/s · 5 min/s · 15 min/s · 1 h/s · Max |
| `n` | step to the next log entry |
| `e` | finish today's operations; end-of-day summary |
| `A` | auto-pause on alerts on / off |
| `d` `b` `h` | dashboard · takeover briefing · daily history |
| `s` `p` `w` | schedule · patients · radiology worklist |
| `l` `f` `a` | event log · finance · actions |
| `↑↓` `j` `k` `←→` `t` | move, scroll, page days (`t`: today) |
| `tab` / `!` | next filter or view / log severity filter |
| `enter` / `esc` | open or submit / back |
| `ctrl+s` `ctrl+o` `ctrl+v` | save · reload the saved session · verify it by replay |
| `?` | help |
| `q` | quit |

Colors follow your terminal; `NO_COLOR=1` turns them off (severity stays
marked with ⚠ and ‼). The terminal must be at least 60×16; 80×24 and larger
layouts are supported.

### What you can do (V0.1)

Three actions, from the actions screen (`a`):

* **Extend a shift**: approve overtime for a technologist today or on a
  future day (up to 240 minutes, to their cap). Extensions ahead of time
  open slots that new orders can book.
* **Walk-in policy**: accept always, accept if there is a gap, accept until
  a cutoff time, or decline (offer an appointment instead).
* **Worklist policy**: the order radiologists read in — urgent & overdue
  first then FIFO (default), FIFO, oldest first, or shortest expected read.
  Urgent studies are always read first; the policy orders the rest.

Patients already admitted are always served, after closing if necessary
(with the overtime it costs); nobody is sent home because of the clock.

## Scenarios

| ID | Practice |
|---|---|
| `established` (**default**: the struggling practice) | DreamTeam Imaging Center: four rooms (XR, US, CT, MR), four technologists (two sharing XR and CT, one US, one MR), a front-desk clerk and an operations manager, two radiologists; open weekdays 08:00–20:00 and Saturdays 10:00–18:00 (XR and CT). 180 days of history before the takeover. A **struggling, capacity-constrained practice**: demand ≈1,650 a month, ≈1,300–1,380 exams, operating margin around −12% to −14%. See `docs/calibration/step-18-report.md`. |
| `mini` | The same practice with no history: a short opening schedule and an empty past, for tests and quick looks. |

    dreamteam scenario list
    dreamteam scenario validate established
    dreamteam scenario coverage established   # weekly staffing and bookable slots

A scenario is a JSON file (`scenarios/`); `-scenario FILE` runs your own.

## Headless runs, sessions and replay

Every run is reproducible from its scenario, seed and actions.

    dreamteam run -days 90 -seed 7                     # text summary
    dreamteam run -days 90 -seed 7 -format json        # canonical result JSON
    dreamteam run -days 30 -actions plan.json -save run.session.json
    dreamteam run -days 7 -log events.jsonl            # the observable event log, one JSON object per line
    dreamteam replay run.session.json                  # rebuild and check every digest
    dreamteam verify run.session.json                  # the same, quietly

An actions file is a list of timestamped batches:

```json
[
 {"time": "2026-10-13T07:00:00", "actions": [
   {"type": "extend_shift", "employee": "E06", "minutes": 120, "date": "2026-10-15"},
   {"type": "set_walkin_policy", "policy": "accept_until", "cutoff": "19:00"}]},
 {"time": "2026-10-14T12:00:00", "actions": [{"type": "set_worklist_policy", "policy": "fifo"}]}
]
```

A saved session (`ctrl+s` in the game, or `-save`) records the scenario,
seed and action batches with five digests (event log, ledger, actions,
state, run). Loading it replays from scratch; a mismatch is reported, never
silently accepted. Exit codes: 0 ok · 1 error · 2 invalid input · 3 replay
failure · 4 golden mismatch · 5 incompatible.

Golden references in `testdata/golden/` pin four runs; see its README for
checking and (deliberately) regenerating them.

## Development

    go test ./...                                    # the suite (a few minutes on a Pi 4)
    go test -tags acceptance ./internal/sim -run Acceptance -v -timeout 600m
                                                     # V0.1 criteria 9 and 10 at full scale (slow)
    go test -tags calibration ./internal/sim -run 'CalibrationExperiments$' -v -timeout 120m
                                                     # nine-seed calibration experiments
    go test ./internal/tui -run ScreenSnapshots -update-screens
                                                     # regenerate screen snapshots after a layout change

Layout: `internal/sim` (engine and domain), `internal/session` (sessions,
records, replay, results), `internal/tui` (Bubble Tea terminal UI, a client
of `session` and `api` only), `internal/api` (observations and actions),
`internal/scenario`, `internal/rng` (keyed, platform-independent draws),
`cmd/dreamteam`.

## Documents

| Document | Contents |
|---|---|
| `docs/PLAN.md` | vision, decisions, scenarios |
| `docs/SIMULATION_DESIGN.md` | domain model, engine, randomness, finance, invariants, metrics |
| `docs/TUI_DESIGN.md` | screens and controls |
| `docs/ROADMAP.md` | milestones V0.1–V0.6, acceptance criteria |
| `docs/ACCEPTANCE.md` | V0.1 acceptance checklist with evidence |
| `docs/RELEASE_NOTES.md` | release notes |
| `docs/PROGRESS.md` | what each step built and how it was tested |
| `docs/MEASUREMENTS.md` | performance measurements |
| `docs/calibration/` | calibration report, experiment manifests and results |
| `docs/reviews/` | external design reviews and their resolutions |

## Known limitations (V0.1)

* One practice, three actions. No hiring, roster or template editing,
  equipment failures, staff absences, rescheduling or overbooking actions
  (V0.2 and later).
* The default practice loses money by design; a better-staffed preset that
  can reach a 6–12% margin is the next scenario milestone.
* Urgent studies acquired after the last reading block wait for the next
  morning (or Monday): V0.1 has no action to add evening or weekend reading
  coverage.
* Save and load are keyboard-only (`ctrl+s`, `ctrl+o`); there is no file
  picker. No mouse support.
* On a Raspberry Pi 4, `run -days 90` takes about 3.3 s (about 1.1 s on a
  Cortex-A76): the Pi 4's CPU has no SHA-256 instructions, and every keyed
  draw is hashed. The 2 s target is waived for the Pi 4 in V0.1 (an
  accepted deviation; the RNG is kept). Interactive play meets its targets.
  See `docs/MEASUREMENTS.md`.
* The default practice usually starts with an empty reading worklist: it is
  short of imaging capacity, not reporting capacity (an accepted
  deviation from the original V0.1 criterion; a reporting-constrained
  scenario is future work).
