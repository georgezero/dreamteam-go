# DreamTeam

A deterministic simulator and management game of an outpatient radiology
practice: patients, referrals and walk-ins, technologists and rooms (XR,
US, CT, MR), radiologists reading a worklist, claims, payroll and cash. You
take over a struggling imaging center on a Tuesday morning and run it from a
terminal UI; the same engine runs headless for research and, later, as an
AI-agent benchmark.

Status: **V0.5** (`0.5.0`, LLM agents) is the current release: an external
agent lets a language model, on any OpenAI-compatible server, manage the
practice through the benchmark harness, with its decisions recorded and
replayed exactly (`agents/llm/`, `docs/LLM_AGENT.md`). It builds on V0.4
(`0.4.0`, the agent evaluation harness: policies, built in or separate
processes, compared reproducibly on identical scenarios and seeds,
`dreamteam benchmark`, `docs/BENCHMARK.md`), V0.3 (`0.3.0`, the agent interface: an
external program runs the practice through a deterministic JSON protocol,
`docs/AGENT_PROTOCOL.md`, with a reference Python client) and V0.2 (`0.2.0`, practice design: staffing, rosters, reading
allocation and teleradiology actions, a sustainable scenario and the T90
evaluation of your plan). Release notes: `docs/RELEASE_NOTES.md`. Everything is synthetic:
patients, staff, referrers, rates and costs are illustrative assumptions,
not operational data. DreamTeam simulates operations; it makes no clinical
decisions and produces no clinical content.

## Quick start

Requires Go 1.26 or later. Runs on Linux and macOS (developed on a
Raspberry Pi 4; CI covers linux/amd64, linux/arm64 and darwin/arm64).

    go run ./cmd/dreamteam                 # play: the established practice, seed 1
    go run ./cmd/dreamteam -seed 7         # another seed: same practice, different patients
    go run ./cmd/dreamteam -scenario sustainable   # the practice you can make sustainable
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
| `m` | staff: hire, terminate, rosters, reading allocation and teleradiology, with a preview |
| `↑↓` `j` `k` `←→` `t` | move, scroll, page days (`t`: today) |
| `tab` / `!` | next filter or view (on the summary: the day, since takeover, the T90 evaluation) / log severity filter |
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

### Staffing and reading coverage (V0.2)

Five actions, from the staff screen (`m`) one at a time with a preview,
or as batches in an actions file (below):

* **Hire** from a fixed offer table (XR, CT, XR+CT, US or MR technologist,
  front desk) at a standard or premium rate. The premium rate buys a
  shorter recruiting lead (28 or 21 days for technologists, 14 or 7 for the
  desk), not productivity.
  * The recruiting fee ($3,000 or $1,000) is charged at once.
  * The new employee is paid from the start date. They work no patients
    until they have attended 40 hours of rostered training (24 for the
    desk), and work independently from the next midnight.
  * The contract's weekly hours are guaranteed, and the regular roster may
    not exceed them.
  * At most 8 technologists and 4 desk staff at once; people on notice
    still count.
* **Terminate** an hourly technologist or front desk employee.
  * The last day is at least 14 days ahead. Contract pay continues to the
    last day, whatever the roster.
  * Two contracted weeks' severance is recognized at once and paid on the
    first payday after the last day.
  * Booked patients who lose their technologist are rebooked (no earlier
    than their original time, within their tolerance) or lost to
    displacement. The preview names each one before you commit.

* **Change a roster** (`set_roster`) of an hourly technologist or front
  desk employee: a new weekly or fortnightly pattern from a Monday at least
  14 days ahead, replacing the previous one until the next change. Pending
  changes can be replaced or withdrawn; past ones never change.
  * The contract does not change: contracted hours are paid whatever is
    rostered, so a shorter (even empty) roster saves nothing, and notice
    and severance stay as they were. The roster may not exceed the
    contract.
  * The same rules as a hire's roster: qualified rooms only, breaks, at
    most six days a week, and 10 hours' rest from the latest time a shift
    may run to (its cap) to the next start, across the change too.
  * An approved overtime extension the new roster cannot carry refuses
    the change; a trainee's roster change moves the date they become
    independent.
  * In the staff screen, enter on a roster row opens the roster editor.

* **Change a radiologist's reading allocation** (`set_reading_allocation`):
  the share of an employed salaried radiologist's paid time spent reading
  for the practice, from 30% to 100%, from a date at least 14 days ahead.
  It changes their reading blocks and the salary charged to the practice.
* **Contract teleradiology** (`contract_teleradiology`): a remote reading
  contract on the `standard-v1` plan for weeknights, Saturday and/or
  Sunday, for urgent studies or all, from at least 14 days ahead.
  * Monthly retainers: weeknights $1,500, Saturday $800, Sunday $800.
  * Per finalized read: XR $12, US $30, CT $45, MR $60, +20% for a study
    claimed from 22:00 to 08:00.
  * Open-ended until cancelled; a cancellation ends it at least 30 days
    ahead and at least 30 days after it starts. Contracts never overlap
    and share the practice's two remote lanes.

A batch containing any of these actions is atomic: one invalid item refuses
the whole batch, and the bookings affected are reconciled once, against the
batch's final staffing (so two employees can swap shifts without
displacing anyone). A refused action has a stable code (for example
`notice_too_short`, `allocation_out_of_band`, `request_conflict`) and a
message, and changes nothing.

### The T90 evaluation: did your decisions help? (V0.2)

On the summary screen (`e`, then `tab` twice) or with
`dreamteam t90 FILE`, your plan (every decision before day 90) is
evaluated against doing nothing, on the same scenario and seed, by the
frozen evaluation policy in force (`v0.2-eval-4`, `docs/evaluation/`):

* **Metric vector**, for your plan, no action and the difference:
  operating profit and margin, net revenue, profit on days 31–90, cash at
  day 90 and the lowest cash, the funding gap, the share of due demand
  served, lost and still-open orders, and report breaches, turnaround and
  unfinished reports for urgent, soon and routine studies, remote reading.
  There is no weighted score.
* **Service gates**: each priority's report-breach rate against the
  policy's limit, by window, as pass, fail or insufficient (too few
  studies to decide), with its counts. The deciding gates set your
  plan's service; the others are reported.
* **Plan gates and verdict**: service, backlog, solvency, funding and case
  selection, and the policy's verdict (GO, CONDITIONAL GO or NO GO) with
  its reasons, on your seed.

Days you have not played yet are **projected**: your plan as it stands,
with no further decision. After day 90 both plans run with no decision
(the evaluation tail); tail and day-180 values describe that fixed
policy, not your later play. The evaluation runs two ~200-day
simulations (about 40 s on a Raspberry Pi; the game goes on meanwhile).
It is defined for the `sustainable` scenario. The clinical candidate
policy v0.2-eval-5 was not approved and plays no part in it.

## Scenarios

| ID | Practice |
|---|---|
| `established` (**default**: the struggling practice) | DreamTeam Imaging Center: four rooms (XR, US, CT, MR), four technologists (two sharing XR and CT, one US, one MR), a front-desk clerk and an operations manager, two radiologists; open weekdays 08:00–20:00 and Saturdays 10:00–18:00 (XR and CT). 180 days of history before the takeover. A **struggling, capacity-constrained practice**: demand ≈1,650 a month, ≈1,300–1,380 exams, operating margin around −12% to −14%. See `docs/calibration/step-18-report.md`. |
| `sustainable` | The same practice at ×1.30 referral demand with 12-minute X-ray slots and a full-time evening MR technologist (Alex): shared XR/CT staffing near break-even (≈+0.5% mean T90 margin). Reorganizing it is your task; the evaluated Step 0C lead staffing earned ≈+2.8% at this demand, but reaching it costs recruiting, notice, severance and training (`docs/V0.2_DESIGN.md` §5.6, §18). The T90 evaluation is defined for this scenario. Its internal id is `established` (the evaluation policy pins its exact bytes); results show its distinct fingerprint `e8718dbd…`. |
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
    dreamteam t90 run.session.json                     # the T90 evaluation (sustainable scenario)
    dreamteam t90 run.session.json -format json        # with the policy's full report

### Agent interface (V0.3)

`dreamteam agent --stdio` runs a session for an external program: one
JSON request per line on stdin, one JSON response per line on stdout.
The operations:
* observe, in full or as sections or a compact projection;
* the action catalog, with schemas and targets;
* validate, preview and atomic apply;
* advance, save and digests;
* the T90 evaluation, on request.

It is a thin client of the same session API, deterministic, and shows
only what a player sees. `docs/AGENT_PROTOCOL.md` is the specification;
`clients/python/` is a reference client (standard library only).

    echo '{"id":1,"op":"hello"}' | dreamteam agent --stdio -scenario sustainable -seed 7
    printf '%s\n' '{"id":1,"op":"observe","args":{"sections":["worklist","alerts"],"compact":true}}' \
      | dreamteam agent --stdio -scenario sustainable -seed 7
    python3 clients/python/example_session.py --binary ./dreamteam --days 5 --t90
    dreamteam agent -policy greedy -days 30 -scenario sustainable -transcript t.jsonl   # a scripted check

An actions file is a list of timestamped batches:

```json
[
 {"time": "2026-10-13T07:00:00", "actions": [
   {"type": "extend_shift", "employee": "E06", "minutes": 120, "date": "2026-10-15"},
   {"type": "set_walkin_policy", "policy": "accept_until", "cutoff": "19:00"}]},
 {"time": "2026-10-14T12:00:00", "actions": [{"type": "set_worklist_policy", "policy": "fifo"}]}
]
```

Hiring and termination use the same file:

```json
[
 {"time": "2026-10-13T06:00:00", "actions": [
   {"type": "terminate", "employee": "E04", "lastDay": "2026-11-10"},
   {"type": "hire", "requestId": "ct-1", "role": "technologist", "skills": ["XR", "CT"],
    "rateCents": 4200, "contractedWeeklyMinutes": 960, "startDate": "2026-11-03",
    "roster": {
      "A": {"Tue": {"start": "11:30", "end": "20:00", "breaks": [["15:30", "16:00"]], "assignment": "R3",
                    "covers": [{"room": "R1", "from": "16:15", "until": "20:00"}]},
            "Wed": {"start": "11:30", "end": "20:00", "breaks": [["15:30", "16:00"]], "assignment": "R3", "covers": []}},
      "B": {"Tue": {"start": "11:30", "end": "20:00", "breaks": [["15:30", "16:00"]], "assignment": "R3", "covers": []}}}}]}
]
```

Weekdays left out of a rota week are days off; the two weeks alternate
from the scenario's week A. A roster change has the same shape:

```json
[
 {"time": "2026-10-13T06:00:00", "actions": [
   {"type": "set_roster", "op": "set", "employee": "E03", "effective": "2026-11-02",
    "roster": {"A": {"Mon": {"start": "07:45", "end": "16:15", "breaks": [["11:30", "12:00"]], "assignment": "R1", "covers": []},
                     "Tue": {"start": "07:45", "end": "16:15", "breaks": [["11:30", "12:00"]], "assignment": "R1", "covers": []}},
               "B": {"Mon": {"start": "07:45", "end": "16:15", "breaks": [["11:30", "12:00"]], "assignment": "R1", "covers": []}}}}]},
 {"time": "2026-10-20T09:00:00", "actions": [
   {"type": "set_roster", "op": "remove", "employee": "E03", "effective": "2026-11-02"}]}
]
```

(`testdata/golden/established-staffing.actions.json` and
`established-roster.actions.json` are complete ones.)

A saved session (`ctrl+s` in the game, or `-save`) records the scenario,
seed and action batches with five digests (event log, ledger, actions,
state, run). Loading it replays from scratch; a mismatch is reported, never
silently accepted. This build writes `dreamteam.session/v2`. It still loads
v1 files: those from earlier builds replay with their outcomes verified,
and saving such a session writes a v2 file that keeps the v1 record as its
verified prefix. Saves from every V0.2 development build (0.2.0-dev.2 to
0.2.0-dev.27) replay with every digest verified, with that build's
actions, and are kept the same way when continued. Exit codes: 0 ok · 1 error · 2 invalid input
· 3 replay failure · 4 golden mismatch · 5 incompatible.

### Agent evaluation harness (V0.4)

`dreamteam benchmark run CONFIG` plays every policy of a configuration on
every scenario and seed through the agent protocol: the same initial state,
horizon and decision cadence for each. A policy sees the session only
through the protocol's read and check operations, and under the harness the
seed is withheld; the harness applies its batches, advances time, saves and
asks for T90. A policy is built in (`noop`, `greedy`, the V0.3
demonstration, and `rules`, a modest rule-based baseline) or an external
program run as a separate process per run, speaking JSON lines over stdio
(V0.4b; the reference is `clients/python/reference_policy.py`). Every saved
session is replayed and measured with the simulator's own canonical result,
and the results directory is hashed in a manifest. `dreamteam benchmark
report` compares the runs (per run, per policy, paired with a baseline; no
composite score), `dreamteam benchmark verify` checks the hashes, replays
the sessions and rebuilds the results, and `dreamteam benchmark reproduce`
reruns the configuration and compares every deterministic file. External
policies are trusted development participants: the harness withholds the
seed from the protocol but does not sandbox them. Specification:
`docs/BENCHMARK.md`; the development runs: `docs/benchmark/v04a-smoke/`
(30 days) and `docs/benchmark/v04b-dev/` (90 days, with the Python policy).

### LLM agent (V0.5)

`agents/llm/llm_agent.py` asks a language model how to manage the
practice. It runs as an external benchmark policy, a separate process per
run, and calls any OpenAI-compatible chat completions API: a local server
or a hosted one. At each decision point it reads the compact public
observation and the action catalog, asks the model for a strict JSON
decision (`{"reason": …, "actions": […]}`), validates it through the
protocol and submits it only if it is valid. It retries a bounded number
of times, then decides no change. Every model exchange (prompt, raw
reply, token usage, latency, errors) is kept in the run's
`policy-evidence/`, with no secrets. A recorded decision sequence replays
through the agent without the model and reproduces the live run's
outcome exactly.

Since V0.5b the agent:
* keeps a bounded memory of its own decisions, the refusals it met and
  what followed them, reset for every run;
* sees each action's fields, rules, earliest dates and lead times beside
  the operating hours, in a prompt about 56% smaller (it can ask for
  details it was not shown);
* may combine operations and staffing actions, submitted as two validated
  batches;
* reports its model usage and an estimated cost (`usage.txt`).

Settings, prompt, memory, decision format, evidence, usage and replay:
`docs/LLM_AGENT.md`; the live smoke tests: `docs/benchmark/v05a-dev/` and
`docs/benchmark/v05b-dev/`; the multi-model development evaluation, three
local models with a memory-off control: `docs/benchmark/v05c-dev/`.

Golden references in `testdata/golden/` pin four V0.1 runs and three V0.2
runs (hire and terminate; roster changes; reading allocation); see its
README for checking and (deliberately) regenerating them.

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
`internal/agent` (the agent protocol), `internal/bench` (the evaluation
harness), `cmd/dreamteam`; `clients/python` (the protocol client and the
reference policy) and `agents/llm` (the LLM agent).

## Documents

| Document | Contents |
|---|---|
| `docs/PLAN.md` | vision, decisions, scenarios |
| `docs/SIMULATION_DESIGN.md` | domain model, engine, randomness, finance, invariants, metrics |
| `docs/TUI_DESIGN.md` | screens and controls |
| `docs/ROADMAP.md` | milestones V0.1–V0.6, acceptance criteria |
| `docs/ACCEPTANCE.md` | V0.1 acceptance checklist with evidence |
| `docs/v02-acceptance/` | V0.2 acceptance pass, matrix and findings |
| `docs/V0.2_DESIGN.md` | V0.2 design: staffing, reading coverage, the T90 evaluation |
| `docs/AGENT_PROTOCOL.md` | the agent protocol (V0.3) |
| `docs/BENCHMARK.md` | the agent evaluation harness and policy protocol (V0.4) |
| `docs/LLM_AGENT.md` | the LLM agent: setup, settings, prompt, memory, decisions, evidence, usage, replay (V0.5) |
| `docs/RELEASE_NOTES.md` | release notes |
| `docs/PROGRESS.md` | what each step built and how it was tested |
| `docs/MEASUREMENTS.md` | performance measurements |
| `docs/calibration/` | calibration report, experiment manifests and results |
| `docs/reviews/` | external design reviews and their resolutions |

## Known limitations (V0.5)

* **LLM agent** (`docs/LLM_AGENT.md` §12):
  * Live model inference is not deterministic, even at temperature 0;
    exact replay requires the recording.
  * External agents are trusted processes, not OS-sandboxed.
  * Operations and staffing actions at one decision point go in two
    batches, which are not jointly atomic.
  * Its memory is bounded, and it does no planning. Models may repeatedly
    propose invalid actions despite the refusals in their memory.
  * Model behaviour depends heavily on the prompt and configuration.
  * In the development evaluation the models frequently chose no action,
    and service improvements sometimes substantially reduced operating
    profit (by $34,565 over 90 days in one run).
  * Cost is an estimate from configured prices.
  * The development results establish neither model superiority nor
    generality, and no model or agent is clinically validated or
    benchmark-certified.

### From V0.4

* **Agent evaluation harness** (`docs/BENCHMARK.md` §12):
  * External policies are trusted development participants, not
    OS-sandboxed processes.
  * Seed withholding prevents disclosure through the benchmark protocol,
    not arbitrary local inspection (files, the binary, a brute-force
    search from the digests).
  * Per-decision timeouts are wall-clock and not deterministic; a run
    that times out is not reproducible.
  * A failed external process is killed and reaped, but children it
    started may be left running.
  * A policy gets no feedback between its batches at one decision point.
  * The policies are demonstrations: all three acting ones use only the
    reading allocation. In the 90-day development matrix the reference
    Python policy never acted (its weekly review missed the backlog
    threshold); `greedy` and `rules` improved reporting but added about
    $33,000 of cost; all 20 runs received T90 NO GO. These are
    development observations, not confirmatory performance claims.

### From V0.3

* **Agent interface:**
  * `t90` runs two ~200-day simulations, about 40 s on a Raspberry Pi.
  * Previews cover the five staffing actions; `validate` checks every
    action.
  * The protocol is stdio only: no HTTP or WebSocket transport, no model
    integration, no benchmark scoring.
* **Cross-platform numbers:**
  * T90 output, the agent protocol and the golden runs are
    byte-identical on linux/amd64, linux/arm64 and darwin/arm64.
  * Since V0.3 the evaluation rounds each product before a sum, so arm64
    T90 statistics (design effects, interval bounds) can differ in their
    last digit from V0.2.0's. Statuses, verdicts and simulation outcomes
    are unchanged.
  * In the developer-only multi-seed evaluations, the joint service
    gate's exact Clopper–Pearson bound can still differ in its last digit
    between processor families.

The V0.2 limitations below still apply.

### From V0.2

* **Terminal UI:** actions are previewed and committed one at a time
  (batches of several actions are headless only); a refused action shows
  its message but not its stable code; the staff table shows contracted
  hours, not rostered and worked hours; displacements are shown in the
  preview, not afterwards; teleradiology always uses the one catalog plan
  (`standard-v1`).
* The T90 evaluation is defined for the `sustainable` scenario only, and
  judges one seed: the policy's statistical classifications are designed
  for many seeds.
* The long-run invariant tests cover the V0.1, staffing and roster plans;
  reading allocation and teleradiology are covered by the suite and the
  golden references.
* No operations-control actions yet (reassign, call in, send home,
  templates, rescheduling, maintenance) or disruptions: planned
  (`docs/ROADMAP.md`).
* On a Raspberry Pi 4, `run -days 90` takes about 3.3–3.9 s (about 1.1 s
  on a Cortex-A76): the Pi 4's CPU has no SHA-256 instructions and every
  keyed draw is hashed. The 2 s target stays waived for the Pi 4 (V0.1's
  accepted deviation). See `docs/MEASUREMENTS.md`.
* Save and load are keyboard-only (`ctrl+s`, `ctrl+o`); no file picker,
  no mouse.
* The `established` practice loses money by design and usually starts
  with an empty reading worklist (V0.1's accepted deviations); the
  `sustainable` scenario is the one to manage toward a profit.
