# V0.13j engineering gates

Independent technical re-review: PENDING. Human presentation review: PENDING.
Protocol lock, live inference and V0.13.0 release: NO GO. These are
engineering checks, not independent review or human acceptance.

Starting commit: 72dc35aa7e73bccf6ca64a3857bc3d58dfabfd43. All remote gates
ran on the exact tree of implementation commit 07106db ("V0.13j: safe-point
interrupts, durable call journal and sealed run completion"); this record is a
docs-only follow-up commit. Remote host: horo, Linux amd64, Go 1.27.0
(/usr/local/go/bin), Python 3.13, directory /home/george/dt-v013j (a fresh
transfer of the committed tree; removed after the archive was copied and
verified). The tree was transferred as a tar of tracked and untracked
non-ignored files, excluding the protected swap file and dt;
FINAL-RUNTIME-SOURCE.sha256 binds all 3,176 transferred files and
GO-SOURCE.sha256 the 375 Go/dependency inputs, both re-verified at the end.
The Pi ran only memory-limited focused checks. No model endpoint, clinical
suite, reserved/holdout seed, historical live rerun, protocol lock or release
is part of this work.

| Location | Gate | Result |
|---|---|---|
| Pi | Baseline identity: HEAD 72dc35a, 2,987 tracked files hashed | PASS; docs/v13j/BASELINE.json, BASELINE-HASHES.sha256 |
| Pi | Safety contract self-check; non-binary V0.13j tests | PASS (5 + 11 tests) |
| Pi | Focused binary checks on linux/arm64, Python 3.11 (DT13I witnesses 49/49, lifecycle examples byte-identical, fault-matrix sample 20/20) | PASS |
| horo | Full Go suite | PASS; 11 packages |
| horo | Race suite, timeout 90m | PASS; 11 packages (internal/eval 4,292 s) |
| horo | Vet default/calibration/acceptance | PASS |
| horo | gofmt (333 Go source files) | PASS |
| horo | Cross-builds linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64 | PASS |
| horo | Four golden references | PASS; outcomes identical |
| horo | Benchmark verify/reproduce | PASS; 86 verification checks, 59 reproduced outcomes |
| horo | V0.8c/V0.9c recorded historical replays | PASS |
| horo | Historical deterministic Go evidence | PASS |
| horo | V0.12b analysis and comparisons | PASS; 420 of 420 draws, COMPLETE |
| horo | V0.13d/f fidelity with historical replays | PASS; 65 and 114 cases, zero failed checks |
| horo | V0.13g contexts/protocol/schedule/check/fidelity/package | PASS; 125 cases, zero failed checks |
| horo | V0.13h protocol/schedule/check/fidelity/lifecycle examples/package/identities | PASS; 126 cases, zero failed checks |
| horo | V0.13i protocol/schedule/check, witnesses and mutations (cmp with committed), examples, fault-matrix sample, identities | PASS; 71/71 witnesses, 69/69 mutations, 61 matrix cases, 9 examples |
| horo | Python suites | PASS; 597 tests: agents 168, acknowledgment 51, evidence 147, probes 3, repeatability 88, historical alternatives (cf9a12b) 29, lifecycle spec v1 4, lifecycle spec v2 5, V0.13g 21, V0.13h 35, V0.13i 21, V0.13j 25 |
| horo | Frozen evidence | PASS; decision and parameter evidence reproduced |
| horo | V0.12a and V0.11b comparisons | PASS; 27 scripted cases; 720 V0.11b draws, 572 calls, 2,299 engine requests |
| horo | V0.13j protocol/schedule regeneration (cmp), 21-context engine check | PASS |
| horo | V0.13j fault matrix regenerated, cmp with committed | PASS; 4,916 cases at 1,511 points, 4,916 agree, 0 false COMPLETE, full transition and stop coverage |
| horo | V0.13j real-process signal and termination tests, cmp | PASS; 326 of 326 |
| horo | V0.13j real loopback tests (sockets), cmp | PASS; 12 of 12 |
| horo | V0.13j regression witnesses, cmp | PASS; 76 of 76 |
| horo | V0.13j contradictory archives, cmp | PASS; 125 of 125 |
| horo | V0.13j lifecycle examples rebuilt, byte-identical | PASS; 16 examples |
| horo | V0.13j identities, 2,984 preserved files, human-review freeze (93 transcripts, 7 static assets) | PASS |
| CI | Both workflows on 07106db | ci 42621374913 SUCCESS (test on ubuntu-latest, ubuntu-24.04-arm, macos-latest; cross-platform); v13f-offline 42621374869 SUCCESS (remediation on the three platforms; compare) |
| CI | Both workflows on this gate-record commit | recorded in the owner handoff (cannot be in the commit that triggers them) |

Full remote evidence archive: /home/george/dt-audit-kept/v013j/final-horo-evidence.tar.gz,
SHA-256 0b7bfb9d89683c286bb6913d4a08debd2f9f097bf7fc7b002b2aadf3287b0e35.
Gate scripts: /home/george/dt-audit-kept/v013j/gates/.

Development notes retained in /home/george/dt-audit-kept/v013j/dev-horo/:
the first full fault-matrix run had 254 disagreeing cases, all in the
harness's expectations (base I's failing release recorded cleanup FAILED
beside earlier stops; a persistent failure armed after the last
filesystem operation cannot fail the directory sync); the first mutation
runs found a fixture clock error in one base, two expectations to correct (a
consistent erasure detected only by replay; an erasure with recomputed counts
that is not a contradiction, replaced by one that is) and one verifier
refinement (a JSON-null seal is now reported as a present null, L16, rather
than as unsealed, S01; INVALID either way); one witness harness had a removed
import. The verifier was also tightened before the final generation (an
operator stop labelled before_health_check in READY requires the schedule to
need that check). No recorder or verifier safety failure (false COMPLETE or
accepted contradiction) was found and no failure is counted as a pass.

Test map: [TEST_MAP.md](TEST_MAP.md).
