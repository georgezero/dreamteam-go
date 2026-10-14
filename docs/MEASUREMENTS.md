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
