# V0.13i engineering gates

Independent technical re-review: PENDING. Human presentation review: PENDING.
Protocol lock, live inference and V0.13.0 release: NO GO. These are
engineering checks, not independent review or human acceptance.

Starting commit: a4d7204c14f36977fee044b3d1a17518d58cc02c. All remote gates
ran on the exact tree of implementation commit
c5b83790 ("V0.13i: run lifecycle state machine and systematic fault
verification"); this record is a docs-only follow-up commit. Remote host:
horo, Linux amd64, Go 1.27.0 (/usr/local/go/bin), Python 3.13, directory
/home/george/dt-v013i (removed after the archive was copied and verified).
The tree was transferred as a tar of tracked and untracked non-ignored files,
excluding the protected swap file and dt; FINAL-RUNTIME-SOURCE.sha256 binds
all 2,986 transferred files and GO-SOURCE.sha256 the 335 Go/dependency inputs,
both re-verified at the end. The Pi ran only memory-limited focused checks.
No model endpoint, clinical suite, reserved/holdout seed, historical live
rerun, protocol lock or release is part of this work.

| Location | Gate | Result |
|---|---|---|
| Pi | Baseline identity: HEAD a4d7204, 2,887 tracked-file hashes | PASS; baseline-hashes.txt |
| Pi | Human-review freeze (93 transcripts, 7 static assets) | PASS; committed first (ee987a1) |
| Pi | Lifecycle contract self-check; non-binary V0.13i tests | PASS |
| Pi | Focused binary tests on linux/arm64, Python 3.11 (fault-matrix sample, DT13H-03 witnesses, persistence) | PASS |
| horo | Full Go suite | PASS; 11 packages |
| horo | Race suite, timeout 90m | PASS; 11 packages (internal/eval 4,098 s) |
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
| horo | V0.13h protocol/schedule/check/fidelity/lifecycle examples/package/identities (checked, not regenerated) | PASS; 126 cases, zero failed checks |
| horo | Python suites | PASS; 567 tests: agents 168, acknowledgment 51, evidence 147, probes 3, repeatability 88, historical alternatives (cf9a12b) 29, lifecycle spec 4, V0.13g 21, V0.13h 35, V0.13i 21 |
| horo | Frozen evidence | PASS; decision and parameter evidence reproduced |
| horo | V0.12a and V0.11b comparisons | PASS; 720 V0.11b draws, 572 calls, 2,299 engine requests |
| horo | V0.13i protocol/schedule regeneration (cmp), 21-context engine check | PASS |
| horo | V0.13i fault matrix regenerated, cmp with committed | PASS; 601 cases at 240 points, 601 agree |
| horo | V0.13i real-process signal tests, cmp | PASS; 19 of 19 |
| horo | V0.13i real loopback tests (sockets), cmp | PASS; 11 of 11 |
| horo | V0.13i regression witnesses, cmp | PASS; 71 of 71 |
| horo | V0.13i contradictory-evidence mutants, cmp | PASS; 69 of 69 rejected |
| horo | V0.13i lifecycle examples rebuilt, byte-identical | PASS; 9 examples |
| horo | V0.13i identities, 2,884 preserved files, human-review freeze | PASS |
| CI | Both workflows on the pushed commit | recorded in the owner handoff (cannot be in the commit that triggers them) |

Full remote evidence archive: /home/george/dt-audit-kept/v013i/final-horo-evidence.tar.gz,
SHA-256 eeda5912d7180260eeb027bc123be123ec6b48e8cc6807fff2cf6dc3d59d48cd.

Development notes retained in /home/george/dt-audit-kept/v013i/dev-horo/:
the first full fault-matrix run found 2 disagreeing cases (a stale "draw
entered" flag between draws), fixed before the committed run; early witness
runs found a truncated-body classification gap (a bounded read hid
IncompleteRead) and three mis-specified mutants, all corrected. No failure is
counted as a pass.

Test map: [TEST_MAP.md](TEST_MAP.md).
