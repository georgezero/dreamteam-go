# DreamTeam — Measurements

Performance evidence for the targets in ROADMAP §2 (criterion 16). Each entry
records the machine, the command and the result. Go 1.27.1, linux/arm64.

## 2026-10-14 — V0.1 steps 2–3 (dev machine: Raspberry Pi 4 Model B, 4 cores)

| Benchmark | Command | Result |
|---|---|---|
| Engine, synthetic 180-day warm-up load (270k events, toy handlers) | `go test ./internal/sim -run X -bench SyntheticWarmupLoad -count 3` | 0.18–0.25 s/op, 33 MB/op, 113k allocs/op |
| One keyed hash draw (`rng.Hash`, SHA-256) | `go test ./internal/rng -run X -bench HashDraw -count 2` | 2.7 µs/op |
| One keyed stream draw (`rng.New`, SHA-256 + ChaCha8 setup) | `go test ./internal/rng -run X -bench KeyedDraw -count 2` | 3.5 µs/op |

Projection for the real warm-up: about 0.2 s of engine work, plus about
0.3 s of keyed hashing (≈ 600 hashes per simulated day, using the lane
grouping in SIMULATION_DESIGN §4.3), plus domain handler cost, which is not
yet measured. The < 1.5 s target looks reachable. It gets confirmed with the
real build at step 11.

## 2026-10-14 — V0.1 step 6 (same machine)

| Benchmark | Command | Result |
|---|---|---|
| Mini scenario, 180 days: roster, staffing, demand and booking (≈ 8.9k orders, 17k events; no arrivals or imaging yet) | `go test ./internal/sim -run X -bench Mini180 -benchtime 5x -count 3` | 0.47–0.50 s/op, 33 MB/op, 552k allocs/op (load average ≈ 5 from other work; single runs ranged up to 0.95 s) |

A CPU profile shows SHA-256 at ≈ 45% of the time: the keyed draws (three
hashes per order at disclosure plus one when it is scheduled) and the
running log hash (one per log entry; every order logs one line). Formatting
log text is ≈ 12%. Arrivals, imaging, reading and finance still have to fit
in the remaining warm-up budget. Options if step 11 misses < 1.5 s: hash
the warm-up log once per day instead of per entry, build warm-up log text
lazily, or cache the built start state (ROADMAP §2, step 11).

## 2026-10-14 — V0.1 step 7 (same machine, loaded: a separate Java process at ~170% CPU)

| Benchmark | Command | Result |
|---|---|---|
| Mini scenario, 180 days, now with arrivals, missed and no-show lifecycle, walk-ins and patience | `go test ./internal/sim -run X -bench Mini180 -benchtime 5x` | 0.78–0.83 s/op (step 6 measured alternately in the same conditions: 0.45–0.75 s/op), 66 MB/op, 1.1M allocs/op |

Each appointment now costs two more keyed hashes and about three more log
entries, and each log entry is hashed into the running log hash. Keyed
hashing and the log together are about half the time. **Risk for step 11:**
with imaging, reading and finance still to add, the < 1.5 s warm-up target
needs the mitigations noted under step 6 (hash the warm-up log per day,
build warm-up log text lazily, or cache the built start state).

## 2026-10-14 — V0.1 step 8 (same machine, load average ≈ 5.5 from other work)

| Benchmark | Command | Result |
|---|---|---|
| Mini scenario, 180 days, now with registration, imaging dispatch, exam timing, hold-over and closing rules | `go test ./internal/sim -run X -bench Mini180 -benchtime 5x -count 2` | 1.26–1.69 s/op, 78 MB/op, 1.5M allocs/op |

Profile: SHA-256 ≈ 27% (keyed draws plus the running log hash), formatting
log text ≈ 19%, dispatch ≈ 26% (imaging match 11%, room states 7%),
FirstFit ≈ 8%. **The step-11 warm-up (< 1.5 s, with reading and finance
still to come) now needs the mitigations**: build warm-up log text lazily
or not at all (hash structured fields), hash the warm-up log per day, and
cache the built start state per scenario and seed if it is still over.
The load from other processes during these runs inflates the figures; a
quiet-machine measurement goes with step 11.

## 2026-10-14 — V0.1 step 8 remediation (same machine, load average ≈ 3.7)

`go test ./internal/sim -run X -bench Mini180 -benchtime 3x -count 2`: 1.4–2.0 s/op. Roughly
unchanged; the deadline wake adds a scan of on-site patients and staff per dispatch.

## 2026-10-14 — V0.1 step 9 (same machine, load average ≈ 3.3)

| Benchmark | Command | Result |
|---|---|---|
| Mini scenario, 180 days, now with studies, worklist, reading and TAT | `go test ./internal/sim -run X -bench Mini180 -benchtime 5x -count 2` | 1.67–1.88 s/op, 90 MB/op, 1.8M allocs/op |

One 180-day run (seed 1): 58,830 events, 10,220 orders, 7,528 studies,
7,514 reports finalized, 14 still on the worklist at the end.
Profile: dispatch ≈ 27%, SHA-256 ≈ 22%, log text formatting ≈ 18%, FirstFit ≈ 7%.
Worklist matching isn't a hot path. Three log lines per study (enters the
worklist, final, and reading starts for urgent studies only) add to the
logging share. There's no simple safe win here. The step-11 mitigations
(lazy or structured warm-up log, per-day log hashing, cached start state)
remain the plan.
