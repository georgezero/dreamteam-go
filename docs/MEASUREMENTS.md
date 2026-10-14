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

## 2026-10-14 — V0.1 step 10 (same machine, load average ≈ 4.8)

| Benchmark | Command | Result |
|---|---|---|
| Mini scenario, 180 days, now with the ledger, claims, payroll and costs | `go test ./internal/sim -run X -bench Mini180 -benchtime 5x -count 3 -benchmem` | 2.11–2.48 s/op, 114 MB/op, 2.0M allocs/op |

One 180-day run (seed 1) posts 27,070 journal entries, handles 7,514 claims
and logs about one finance line per claim, plus one line for each day's
submission batch.

Profile:
* SHA-256: ≈ 22% (keyed draws and the running log hash).
* Log text formatting: ≈ 14%.
* Dispatch: ≈ 23%.
* Finance in total: about 10%.
  * Adjudication: 4%, including the claim draw, which is hashed at
    submission and again at remittance.
  * Revenue recognition: 3%.
  * Ledger posting: 2%.

The step 9 figure (1.67–1.88 s) was taken at a lower load (≈ 3.3). On top
of that, finance adds about 0.3–0.4 s. The step-11 mitigations are now
required, since the warm-up target is < 1.5 s:
* lazy or structured warm-up log text;
* per-day log hashing;
* a cached built start state.

A smaller one is available too: cache the claim draw from submission until
remittance.

## 2026-10-14 — V0.1 step 11: the 180-day warm-up (same machine)

`BenchmarkEstablishedWarmup` builds the established practice: `sim.New`,
then run to the start time, the same work as `session.Open`. Previous
(`32313ac`) and new binaries were run alternately three times each. There
was a 45 s cool-down before each run, and `-benchtime 5x -count 2`:

| Build | ns/op (6 samples) | Memory | Allocations |
|---|---|---|---|
| step 10 (`32313ac`) | 1.80, 2.19, 2.38, 1.80, 2.32, 2.35 s | 111.6 MB/op | 1.97M/op |
| step 11 | 1.54, 1.71, 1.73, 1.88, 1.97, 1.35 s | 89.7 MB/op | 1.26M/op |

**Conditions.** The SoC was at 75–79 °C, at the soft throttling limit.
`vcgencmd get_throttled` reported throttling earlier in the session, and
load averages were 2–4 from other work. The spread is mostly throttling:
unthrottled runs give 1.34–1.35 s, throttled runs 1.7–1.9 s.

**Result.** About 22–25% faster at the median, with 20% less memory and 36%
fewer allocations. The ≤ 1.5 s target is met unthrottled and missed by up
to 0.45 s when throttled. A cooler or idle Pi, or the step 14 start-state
cache, closes the rest.

Profile after the changes:
* Dispatch: 28%.
* Keyed SHA-256 draws: 19%. A key encodes to 86–102 bytes, so every draw
  is at least two compressions, and the encoding is randomness semantics,
  so it was not changed. 125,067 draws per warm-up, of which 87,297 are
  distinct; same-day repeats are now memoized.
* FirstFit booking: 9%.
* fmt at call sites: 8%.
* Log: 7% (was 21%: ISO time text 8% and per-entry hash-chain finalization
  are gone).
* GC: about 4%.

Not done:
* Lazy, structured log text. It would remove most of the remaining fmt and
  log cost, but it changes every emit call site.
* A cached start state. Snapshots are step 14; the roadmap fallback stays
  available.

## 2026-10-14 — V0.1 step 12 (Pi at 81 °C, throttling flagged)

Old (`23b6ff8`) and new test binaries run alternately, four rounds, to cancel
thermal drift; best of four:

| Benchmark | Command | `23b6ff8` | step 12 |
|---|---|---|---|
| Established warm-up (180 days) | `go test ./internal/sim -run X -bench EstablishedWarmup` | 1366 ms (1366–2248) | 1402 ms (1402–1420) |
| One observation at 11:00 (a TUI frame) | `-bench ObserveMidday` | 1.20 ms | 1.26 ms |
| Evaluation window after 30 days (on request) | `-bench Evaluation30Days` | — | 1.27–2.01 ms, 108 allocs |

Warm-up memory 90.6 MB/op, 1.26M allocs/op (step 11: 89.7 MB, 1.26M). The
interval accrual runs once per transaction; alerts run only from the
takeover, so the warm-up pays only the accrual and the daily snapshots
(+2.6%). Completed days' summaries are cached; the evaluation window is
computed only when a screen asks for it. At 10 frames a second the
observation costs about 1.3% of one core.
