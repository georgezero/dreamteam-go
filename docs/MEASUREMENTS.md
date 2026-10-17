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

## 2026-10-14 — V0.1 step 13 (Pi at 79–81 °C; timings bimodal with throttling)

Old (`4bbd03f`) and new binaries alternated; each lands in either a ~1.4 s
or a ~2.4–2.9 s mode depending on throttling, so the best runs compare:

| Benchmark | Command | `4bbd03f` | step 13 |
|---|---|---|---|
| Established warm-up (180 days) | `-bench EstablishedWarmup -benchmem` | 1401–1434 ms, 90.75 MB, 1.266M allocs | 1402–1411 ms, 90.78 MB, 1.266M allocs |
| One observation at 11:00 | `-bench ObserveMidday` | 1.20–1.30 ms | 1.32–1.35 ms |
| Validate one action (preview) | `-bench ValidateOnly` | — | 3.8–4.2 µs |
| Apply one action (policy switch + dispatch) | `-bench ApplyValidate` | — | 12–14 µs |
| 30 days after takeover, no actions / daily interventions | `-bench ThirtyDaysWithActions` | — | 405 ms / 380 ms (best of 3; same within noise) |

Action processing has no cost when no action is taken; the evening
template slots (unstaffed unless extended) add nothing measurable.

## 2026-10-14 — V0.1 step 14 (Pi at 80–82 °C)

| Measurement | Result |
|---|---|
| Established warm-up, `08ccc5d` vs step 14 alternated (best of 4) | 1359 ms vs 1415 ms; 90.78 MB, 1.266M allocs both (noise) |
| `dreamteam run -days 30` (established, 4 actions) + save | 3.98 s wall, 48 MB peak RSS; session file 35 KB |
| `dreamteam replay` of that file | 3.95 s, 53 MB peak RSS |
| `dreamteam verify` | 2.96 s, 49 MB peak RSS |
| Save 30 days after takeover (`-bench Save30Days`) | 0.44–0.54 s (mostly the state digest), 1.3 MB |
| Load = replay + verify, 30 days (`-bench Replay30Days`) | 2.1–2.3 s, 117 MB allocated |
| State digest after 10 days of play | ≈0.3 s |

## 2026-10-14 — V0.1 step 15 (Pi at ~80 °C)

| Measurement | Result |
|---|---|
| `dreamteam run -days 7 -format json` (established, warm-up included) | 1.91–1.98 s, 40–42 MB peak RSS |
| `dreamteam run -days 90 -format json` | 3.06–4.70 s, 60–61 MB peak RSS |
| Golden generation with `-verify-replay` (run + replay) | 7 d 4.2 s · 90 d 8.0 s · mini 7 d 0.7 s · 7 d with actions 5.6 s |
| Result build + encode after 90 days (`-bench Result90Days`) | 0.54–0.62 s (mostly the state digest), 5.4 MB |
| Golden comparison, 90-day result (`-bench Compare90Days`) | 18–25 ms |
| Reference sizes | 16–18 KB (7 days), 52 KB (90 days) |

## 2026-10-14 — V0.1 step 18 (calibration suite)

| Run | Time |
|---|---|
| `-tags calibration -run CalibrationReport`, 9 seeds × 90 days | ≈29 s |
| `-run CalibrationActions`, 9 seeds × 6 strategies × 30 days | ≈115 s |
| `TestEstablishedCharacterization` (3 seeds × 30 days, untagged) | ≈5.4 s |

No engine change; warm-up and run times as in step 15.

## 2026-10-14 — V0.1 step 18.1 (Pi at ~80 °C, timings bimodal)

| Measurement | Result |
|---|---|
| Established warm-up, old `sharedFree` (`b1a9ffb`) vs the feasibility check, interleaved best-of | 1.43–1.60 s vs 1.47–1.54 s: no measurable change |
| First feasibility version (cut every booked slot on each call) | 2.4–3.0 s; profiling showed the cutting dominated, so booked slots are now screened by shared staff and time window before cutting |
| `-tags calibration -run 'CalibrationExperiments$'`: 10 manifests, 9 seeds each (6 × 90 days, 4 × 30 days) | ≈270 s |
| `TestRegress18WalkInCohorts` (established, 21 days, per-day cohort checks) | ≈3 s |

## 2026-10-14 — V0.1 step 19: performance pass (criterion 16)

Raspberry Pi 4 (Cortex-A72, 4 cores), at 80–83 °C (throttling) with other
users' processes keeping the load average at 4–11; best of interleaved runs.
The A72 has no SHA-256 instructions (`/proc/cpuinfo` features: fp asimd
evtstrm crc32 cpuid), so Go uses its generic SHA-256.

| Measurement | Command | Result | Target |
|---|---|---|---|
| Warm-up (180 days, in process) | `-bench EstablishedWarmup$` | 1.44–1.49 s best of six (1.6–2.4 s in loaded runs) | < 1.5 s: met at best, not under load |
| 90 days of play after the takeover (in process) | `-bench EstablishedPlay90` (new) | 1.07–1.10 s best of six | — |
| `dreamteam run -days 90 -seed 7 -format json` (warm-up, play, result with digests) | `meas.py` wall time | 3.3 s best of three (4.2–5.3 s loaded), 57–62 MB peak RSS | < 2 s: **not met** |
| `dreamteam run -days 0` (warm-up and result) | same | 2.05 s best of three | — |
| TUI paused (idle), 140×40 | `/proc/PID/stat` over 60 s in tmux | 0.4% of one core (was 3.0%) | — |
| TUI running at 15 min/s, 10 fps, 140×40 | same | 7.4–8.2% of one core | < 15%: met |
| Max speed: frame time (simulation step + render), 20 days | `-bench MaxSpeedFrames` (new) | p50 23–27 ms, worst 52–94 ms (one frame = one simulated day) | `space` within 100 ms: met |

The same commit on joofi (Cortex-A76, 4 cores, with SHA-256 and AES
instructions; load average 5–10): warm-up 0.53–0.61 s (`-bench`),
90 days of play 0.41–0.47 s, `dreamteam run -days 90 -seed 7 -format json`
1.09–1.22 s wall, `-days 0` 0.60–0.74 s — **every criterion 16 target is
met on that class of machine**.

Acceptance suite (`-tags acceptance`, test binary of 75d217c on joofi):
criterion 10, invariants after every transaction for 50 seeds × 90 days of
the established practice, passed in 53.5 min on four cores; criterion 9,
1,000 random chunkings of the real simulation, passed in 22.6 s. On the Pi
4, one seed of criterion 10 takes ≈7 min (the full invariant check walks
every record after every transaction).

**Profile of a 90-day run** (`-cpuprofile`, warm-up plus play): SHA-256
16–17% (keyed draws `rng.Hash` and the log hash), order disclosure and
booking 24% (of which FirstFit 13% and the shared-staff feasibility check
6%), `fmt` for log text ≈6%, allocation and GC ≈15%, alerts 5.6%, the
ledger 2.8%. No single hotspot: reaching 2 s on a Pi 4 needs either a
different keyed-draw hash (it would change every outcome and digest, so
it is an engine-version decision, not a tune-up) or broad allocation work.

**Changes kept.** The frame clock stops while the game is paused (no
renders between keys), and the renderer runs at 20 fps instead of Bubble
Tea's default 60: idle CPU 3.0% → 0.4%; outcomes are unaffected (the TUI
never changes them; `TestTUIRunReplaysHeadless`).

**Tried and reverted.** A per-day free-slot count to skip full days in
FirstFit: outcomes identical, but best-of-six play time moved only 1.10 →
1.07 s (days are rarely full for the category searched), so not worth the
extra state.

## 2026-10-15 — V0.2 Slice 1: employee contracts (Pi 4, ≈79 °C, load average ≈3)

Interleaved wall times of `dreamteam run -scenario established -seed 7 -days
90 -format json`, the 0.2.0-dev.0.2 build (44c2df4) against this one, four
runs each: 3.36–4.41 s before, 3.30–4.84 s after (best 3.36 → 3.30 s): the
daily guarantee close is not measurable.

| Measurement | Command | Result |
|---|---|---|
| Whole-schedule staffing check: Sam removed from tomorrow, the established practice's full booked horizon at the takeover | `go test ./internal/sim -run X -bench StaffingCheck -benchtime 10x -count 3 -benchmem` | 11.3–16.1 ms/op, 1.7 MB/op, 11.8k allocs/op |

## 2026-10-15 — V0.2 Slice 2: hire and terminate (Pi 4, 82–84 °C and throttled, load average ≈5.5 from unrelated processes)

Interleaved wall times of `dreamteam run -scenario established -seed 7 -days
90 -format json`, Slice 1 (19dcd10) against this build, ten runs each:
best 3.20 s before, 3.49 s after; medians overlap (≈4 s, load-dominated).
The load-independent figures are the benchmarks' allocation counts
(`go test ./internal/sim -run X -bench 'EstablishedWarmup$|EstablishedPlay90$' -benchtime 3x -count 2 -benchmem`):

| Benchmark | Slice 1 | Slice 2 |
|---|---|---|
| Established warm-up (180 days) | 1.49–1.68 s/op, 93.4 MB, 1.397M allocs | 1.61–2.46 s/op, 104.3 MB, 1.415M allocs |
| Established, 90 days of play | 1.09–1.45 s/op, 73.4 MB, 1.229M allocs | 1.14–1.66 s/op, 78.6 MB, 1.241M allocs |

So about +1% allocations and +7–12% bytes (wider slots for the growable
staff set, appointment and order fields, the per-day registry view) and,
on best times, about +5% CPU. No new hot spot: `sharedFree` is ≈10% of the
profile, as before. The Pi 4's 90-day waiver (V0.1) still applies; not
re-measured on the Cortex-A76 (busy with the acceptance runs).

| Measurement | Command | Result |
|---|---|---|
| Whole-schedule staffing check (Slice 1 benchmark) | `-bench StaffingCheck$ -benchtime 10x -count 2 -benchmem` | 12.2–23.0 ms/op, 2.1 MB/op, 12.1k allocs/op |
| Termination preview at the takeover (validation, the booked horizon re-planned, survivors, provisional replacements) | `-bench StaffingPreviewTermination` | 20.8–24.0 ms/op, 3.5 MB/op, 21.6k allocs/op |
| Committing that termination (severance, reconciliation, 84 displacements re-offered) | `-bench StaffingCommitTermination` | 29.9–30.6 ms/op, 2.9 MB/op, 25.9k allocs/op |
| Staff screen render with a termination preview, 160×48 | `go test ./internal/tui -run StaffScreenRenderTime -v` | 25 ms per frame |
| 60 days with the V0.2 reference plan vs no actions | `dreamteam run -days 60 [-actions testdata/golden/established-staffing.actions.json]` | 3.29–4.98 s vs 3.05–4.27 s: no difference beyond noise |

## 2026-10-15 — V0.2 Slice 3: set_roster (Pi 4, ≈80 °C, load average ≈4–6 from unrelated processes)

Interleaved wall times of `dreamteam run -scenario established -seed 7 -days
90 -format json`, Slice 2 (2da900d) against this build, eight runs each:
best 3.40 s before, 3.41 s after; medians ≈3.9 s both (load-dominated). The
90-day play benchmark allocates exactly as in Slice 2 (78.6 MB, 1.241M
allocations): without a version, resolving the roster costs one empty-slice
check per employee and day.

| Measurement | Command | Result |
|---|---|---|
| Roster preview at the takeover: Sam's Tuesdays off (validation, composed timeline, the booked horizon re-planned, survivors, coverage differences, provisional replacements) | `go test ./internal/sim -run X -bench RosterPreview -benchtime 10x -count 2 -benchmem` | 28.2–28.3 ms/op, 4.1 MB/op, 32.5k allocs/op |
| Committing it (version, one reconciliation, displacements re-offered) | `-bench RosterCommit` | 25.3–25.7 ms/op, 2.6 MB/op, 22.9k allocs/op |
| Slice 2 termination preview / commit, same run (unchanged allocations) | `-bench 'StaffingPreviewTermination|StaffingCommitTermination'` | 30.8–31.7 / 29.2–33.6 ms/op; 21.6k / 26.0k allocs |
| Roster editor render with a displacing preview, 160×48 | `go test ./internal/tui -run RosterEditorRenderTime -v` | 48 ms per frame (under load) |
| The six golden references and the Slice 2 session on the Cortex-A76 | `run_refs` (each `run … -verify-replay -golden`) | 9.2 s in all; outputs byte-identical to the Pi's |

## 2026-10-16 — V0.2 dynamic staffing benchmark (Pi 4 at 80–83 °C; Cortex-A76 "joofi", shared with other work, load average up to ≈13)

No engine code changed. The benchmark harness (`-tags calibration`,
`internal/sim/calib_dynamic_test.go`) plays 180 days per seed and plan.
Each day it samples the evaluation window and balances; at T30, T90 and
T180 it measures and runs a whole-schedule staffing check.

| Measurement | Command | Result |
|---|---|---|
| One plan, one seed, 180 days (full plan C), Pi, alone | `DT_DYN_SET=dev DT_DYN_SEEDS=901 DT_DYN_PLANS=C-full-reorganization go test -tags calibration ./internal/sim -run 'DynamicStaffing$'` | 9.9 s |
| Development set, 5 plans × 10 seeds × 180 days, Pi | `DT_DYN_SET=dev … -parallel 3` | 6 min 57 s |
| Evaluation set, 3 plans × 20 seeds × 180 days, Cortex-A76 (compiled test binary) | `DT_DYN_SET=eval ./sim-cal.test -test.parallel 3` | 202 s idle; slower while shared |
| Session replay check, 6 plans × 2 seeds × 180 days, saved and replayed whole and in 13-hour chunks, Cortex-A76 | `./session-cal.test -test.run DynamicPlansReplay` (`DT_DYN_DAYS=180 DT_DYN_SEEDS=901,1001`) | 226 s (loaded) |
| Full uncached suite after the version bump, Pi | `go test -count=1 -timeout 60m ./...` | 9 min 6 s (sim 530 s, session 104 s, cmd 100 s, tui 31 s) |

The result files from the 0.2.0-dev.3 binary (development set on the Pi,
evaluation set on the Cortex-A76) and from the 0.2.0-dev.4 binary (both
sets on the Cortex-A76) are identical in every field but the engine name.

## 2026-10-16 — V0.2 Slice 4: set_reading_allocation and plan D (Pi 4 at ≈82 °C, load average ≈6–8; Cortex-A76 "joofi", load average ≈2–6)

| Measurement | Command | Result |
|---|---|---|
| Plan D validation, 3 plans × 10 development seeds × 180 days, Cortex-A76 (compiled test binary) | `DT_DYN_SET=reading-dev ./sim-cal.test -test.run 'DynamicStaffing$' -test.parallel 3` | 70 s |
| Evaluation, 4 plans × 20 held-out seeds × 180 days, Cortex-A76 | `DT_DYN_SET=reading-eval ./sim-cal.test … -test.parallel 4` | 176 s |
| Session replay check, 7 plans × 2 seeds × 180 days, whole and in 13-hour chunks, Cortex-A76 | `./session-cal.test -test.run DynamicPlansReplay` (`DT_DYN_DAYS=180 DT_DYN_SEEDS=901,1001`) | 169 s |
| `sim` package alone, uncached, Pi | `go test ./internal/sim -count=1 -timeout 60m` | 565 s |
| Full uncached suite, Pi | `go test ./... -count=1 -timeout 60m` | 9 min 1 s (sim 537 s, session 94 s, cmd 84 s, tui 27 s) |

The default `go test ./...` timeout (10 minutes) was exceeded once by `sim`
while the Pi was compiling in parallel; use `-timeout 60m` on the Pi. A, B
and C under 0.2.0-dev.5 reproduce the 0.2.0-dev.4 evaluation results in
every field but the engine name. Their seed-1001 sessions also reach the
same run digests.


## 2026-10-16 — V0.2 dev.6 pre-Slice 5 remediation (Pi 4 at ≈82 °C; Cortex-A76 "joofi" at ≈84 °C, shared, load average ≈1.5–18)

| Measurement | Command | Result |
|---|---|---|
| Engine benchmarks, 0.2.0-dev.5 against 0.2.0-dev.6, interleaved in both orders (7 samples each), Cortex-A76 | `./sim-devN.test -test.bench 'EstablishedPlay90\|Mini180' -test.run '^$' -test.benchmem` | Mini180: median 917 ms/op (dev.5) vs 937 ms/op (dev.6). EstablishedPlay90: 376 vs 386 ms/op. Run-to-run spread ≈15%, so no measurable difference. Bytes and allocations per op identical (183.7 MB, 3.73M allocs; 78.6 MB, 1.24M allocs) |
| Plan D validation, 4 plans × 10 development seeds × 180 days, Cortex-A76 | `DT_DYN_SET=reading-dev ./calib.test -test.run 'DynamicStaffing$' -test.parallel 3` | 54 s (dev.5: 70 s, shared) |
| Evaluation, 4 plans × 20 held-out seeds × 180 days, Cortex-A76 | `DT_DYN_SET=reading-eval ./calib.test … -test.parallel 4` | 194 s (dev.5: 176 s; load ≈18 from other work) |
| Session replay check, 7 plans × 2 seeds × 180 days, whole and in 13-hour chunks, Cortex-A76 | `./sesscalib.test -test.run DynamicPlansReplay` (`DT_DYN_DAYS=180 DT_DYN_SEEDS=901,1001`) | 171 s |
| Acceptance (`-tags acceptance`), Cortex-A76 | `./acc.test -test.v` | Staffing and roster invariants, turnover and churn: 2 h 0 min while shared. The run hit its 120-minute limit during the V0.1 criteria, which then passed alone in 42 min (`-test.run 'TestAcceptance(Invariants|Chunking)$'`; chunking 20.2 s). Set `-test.timeout 240m` |
| Full uncached suite, Pi | `go test ./... -count=1 -timeout 60m` | sim 544 s (9 min 5 s wall alone), session 111 s, cmd 87 s, tui 24 s |

The benchmark results under 0.2.0-dev.6 match the 0.2.0-dev.5 archive
(`results/reading-dev`, `results/reading-eval`) in every field but the
engine name.


## 2026-10-16 — V0.2 Slice 5A: remote reading engine (Pi 4 at ≈81 °C, load average ≈4–5; Cortex-A76 "joofi", shared, load average ≈2–5)

| Measurement | Command | Result |
|---|---|---|
| Engine benchmarks, 0.2.0-dev.6 against 0.2.0-dev.7, interleaved in both orders (6 samples each), Cortex-A76 | `./sim-devN.test -test.bench 'EstablishedPlay90\|Mini180' -test.run '^$' -test.benchmem` | Mini180: median 947 ms/op (dev.6) vs 959 ms/op (dev.7). EstablishedPlay90: 392 vs 388 ms/op. Run-to-run spread ≈7%, so no measurable difference. Bytes and allocations per op identical (183.5 MB, 3.73M allocs; 78.6 MB, 1.24M allocs) |
| Engine benchmarks, same, Pi (3 samples each) | same | Mini180: 2.66–3.69 s/op (dev.6) vs 2.61–3.34 s/op (dev.7). EstablishedPlay90: 1.15–1.69 vs 1.35–1.63 s/op. Spread ≈±25% under load; no measurable difference; allocations identical |
| Plan D validation, 4 plans × 10 development seeds × 180 days, Cortex-A76 | `DT_DYN_SET=reading-dev ./calib.test -test.run 'DynamicStaffing$' -test.parallel 3` | 55 s |
| Evaluation, 4 plans × 20 held-out seeds × 180 days, Cortex-A76 | `DT_DYN_SET=reading-eval ./calib.test … -test.parallel 4` | 171 s |
| Session replay check, 7 plans × 2 seeds × 180 days, whole and in 13-hour chunks, Cortex-A76 | `./sesscalib.test -test.run DynamicPlansReplay` (`DT_DYN_DAYS=180 DT_DYN_SEEDS=901,1001`) | 165 s |
| Acceptance (`-tags acceptance`), Cortex-A76 | `./acc.test -test.v -test.timeout 240m` | 2 h 11 min while shared, 369 subtests passed |
| Full uncached suite, Pi | `go test ./... -count=1 -timeout 60m` | 10 min 11 s (sim 589 s, session 138 s, cmd 113 s, tui 36 s); sim alone 544 s |

The benchmark results under 0.2.0-dev.7 match the committed archive
(`results/reading-dev`, `results/reading-eval`) in every field but the
engine name.


## 2026-10-17 — V0.2 Slice 5B: teleradiology contracts (Pi 4 at ≈82 °C, load average ≈3.5; Cortex-A76 "joofi", shared, load average ≈1–3)

| Measurement | Command | Result |
|---|---|---|
| Engine benchmarks, 0.2.0-dev.7 against 0.2.0-dev.8, interleaved in both orders (6 samples each), Cortex-A76 | `./sim-devN.test -test.bench 'EstablishedPlay90\|Mini180' -test.run '^$' -test.benchmem -test.benchtime 4x` | Mini180: median 946 ms/op (dev.7) vs 970 ms/op (dev.8), ranges 935–959 vs 937–990. EstablishedPlay90: 387 vs 390 ms/op. No measurable difference. Allocations identical (3.73M; 1.24M); EstablishedPlay90 bytes 78.6 vs 78.2 MB/op |
| Engine benchmarks, same, Pi (6 samples each, `-test.benchtime 2x`) | same | Mini180: median 2.89 vs 2.94 s/op. EstablishedPlay90: 1.34 vs 1.34 s/op. Spread ≈±25% under heat and load; allocations identical |
| Plan D validation, 4 plans × 10 development seeds × 180 days, Cortex-A76 | `DT_DYN_SET=reading-dev ./calib.test -test.run 'DynamicStaffing$' -test.parallel 3` | 66 s |
| Evaluation, 4 plans × 20 held-out seeds × 180 days, Cortex-A76 | `DT_DYN_SET=reading-eval ./calib.test … -test.parallel 3` | 220 s |
| Session replay check, 7 plans × 2 seeds × 180 days, whole and in 13-hour chunks, Cortex-A76 | `./sesscalib.test -test.run DynamicPlansReplay` (`DT_DYN_DAYS=180 DT_DYN_SEEDS=901,1001`) | 179 s |
| Acceptance (`-tags acceptance`), Cortex-A76 | `./acc.test -test.v -test.timeout 240m` | 2 h 35 min while shared, 396 subtests passed, none failed |
| Full uncached suite, Pi | `go test ./... -count=1 -timeout 60m` | 9 min 58 s (sim 560 s, session 133 s, cmd 98 s, tui 28 s) |

The benchmark results under 0.2.0-dev.8 match the committed archive
(`results/reading-dev`, `results/reading-eval`) in every field but the
engine name.


## 2026-10-17 — V0.2 Slice 5C.0: frozen evaluation framework (Pi 4 at ≈77 °C, load average ≈3.4; Cortex-A76 "joofi", load average ≈0.4–1.6)

| Measurement | Command | Result |
|---|---|---|
| Full uncached suite, Pi | `go test ./... -count=1 -timeout 60m` | 11 min 31 s (sim 668 s, session 169 s, cmd 95 s, tui 29 s, eval 13 s) |
| Evaluator tests, Pi | `go test ./internal/eval -count=1` | 3–13 s (19 tests on synthetic 10-seed × 181-day fixtures) |
| Recorder integration, Pi | `go test ./internal/sim -run EvalRecordOnRealRuns` | 4.9 s (five mini runs of 41 days) |
| Plan D validation, 4 plans × 10 development seeds × 180 days, Cortex-A76 | `DT_DYN_SET=reading-dev ./calib.test -test.run 'DynamicStaffing$' -test.parallel 3` | 55 s |
| Evaluation, 4 plans × 20 reference seeds × 180 days, Cortex-A76 | `DT_DYN_SET=reading-eval ./calib.test … -test.parallel 3` | 197 s |
| Session replay check, 7 plans × 2 seeds × 180 days, Cortex-A76 | `./sesscalib.test -test.run DynamicPlansReplay` | 163 s |

The simulation's production code is unchanged, so the engine benchmarks
were not rerun. The benchmark results under 0.2.0-dev.9 match the
committed archive in every field but the engine name.

