# V0.13k engineering gates

Independent review PENDING. Human presentation review PENDING. Protocol lock, live inference and V0.13.0 release NO GO. These are engineering checks and review preparation, not independent clearance.

Starting commit: `cb5ac6832e7b43ef68212355ff395cf5958fff11`. Implementation: `87d00a6`; tests and evidence: `025ac5e`. Heavy checks run on horo in `/home/george/dt-v013k`, Linux amd64, Go 1.27.0 and Python 3.13. The Pi uses the requested 2 GiB memory scope and reduced priority for focused checks. Source transfers use the tracked/untracked nonignored file list, excluding the protected swap file and dt. Go inputs and all earlier source/evidence retain their hashes. The final source manifest and gate logs are retained externally.

| Location | Gate | Result |
| --- | --- | --- |
| Pi | Baseline identity and inventory | PASS; starting cb5ac6832e7b43ef68212355ff395cf5958fff11, 3,172 protected files |
| Pi | Focused successor and lifecycle-spec tests | PASS; 17 + 5; 17 binary tests deferred to horo, no local heavy duplication |
| horo | Full Go suite | PASS; 11 packages |
| horo | Race suite | PASS; 11 packages; longest package 5,348.344 s, 90-minute timeout |
| horo | Vet default, calibration and acceptance | PASS; three modes |
| horo | gofmt | PASS; 333 source files |
| horo | Cross-builds | PASS; linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64 |
| horo | Golden references | PASS; four references |
| horo | Benchmark verify/reproduce | PASS; 86 verification checks, 59 reproduced outcomes |
| horo | Recorded historical replay and deterministic Go evidence | PASS; recorded replay inputs only, four targeted historical Go tests |
| horo | V0.12b comparison | PASS; 420/420 draws, COMPLETE; three outputs byte-identical |
| horo | V0.13d/f fidelity with historical replays | PASS; 65 and 114 cases, zero failed checks |
| horo | V0.13g fidelity, protocol, schedule, checks and package | PASS; 125 cases, zero failed checks |
| horo | V0.13h fidelity, protocol, schedule, checks, examples and package | PASS; 126 cases, zero failed checks |
| horo | V0.13i witnesses, mutations, matrix sample and examples | PASS; 71, 69, 61 and nine; byte comparisons passed |
| horo | Python suites | PASS; 636 tests, no skips: 597 preserved suites, 34 successor and five lifecycle /3 |
| horo | Frozen evidence | PASS; decision and parameter evidence reproduced |
| horo | V0.12a and V0.11b comparisons | PASS; 27 scripted cases; 720 draws, 572 calls, 2,299 engine requests |
| horo | V0.13j fidelity and deterministic evidence | PASS; 21-context check, 4,916 fault cases at 1,511 points, 326 real signals, 12 loopback, 76 witnesses, 125 mutations, 16 examples; all byte-identical |
| horo | Successor context and schedule checks | PASS; 21 contexts, 262 scheduled draws and 802-call ceiling preserved |
| horo | Binding matrix | PASS; 35 relationships × ten kinds = 350 full analyses; 35 valid accepted, 315 contradictory rejected, zero survivors |
| horo | Successor lifecycle fault matrix | PASS; 5,288 cases at 1,619 points, all agree; zero false COMPLETE; full 19-transition and 22-stop-state coverage |
| horo | Successor real-process signals and termination | PASS; 353/353 |
| horo | Successor real localhost loopback sockets | PASS; 12/12 |
| horo | Focused original and supplementary regressions | PASS; 16/16, hashes and seals recomputed |
| horo | Completed-result preservation | PASS; six actual filesystem/return/error-handling cases; exact completed results retained and replayed |
| horo | Inherited witnesses and re-sealed mutations | PASS; 76/76 and 125/125, including one positive control |
| horo | Successor fidelity with historical replay inputs | PASS; all 66 scheduled context/model/arm units, 62 scripted calls, 834 engine events; full analysis VERIFIED |
| horo / CI | Deterministic examples, protocol, schedule and matrix | 16 successor examples generated and verified on horo; both workflows regenerate and compare these artifacts on three platforms |
| horo / Pi | Protected evidence, seal contract and human-review freeze | PASS; all 3,172 protected files, nine contexts, six pairs, 93 transcripts and seven assets unchanged; finalization subtree equal |
| CI | Final-push workflows | Run IDs and final results recorded in the owner handoff and the external CI_RESULTS.json; the triggering commit cannot contain its own results |

Python suite counts: agents-llm 168, analysis-acknowledgment 51, analysis-evidence 147, analysis-probes 3, analysis-repeatability 88, historical-alternatives 29, lifecycle-spec 4, lifecycle-v2-spec 5, v13g 21, v13h 35, v13i 21, v13j 25, v13k-except-matrix 33, v13k-matrix-sample 1, lifecycle-v3-spec 5.

The successor unit suite was run as 33 tests plus its one 36-case matrix sample after the final matrix was available. All 34 passed. The complete suite is run together by both workflows. Loopback evidence includes host/interpreter metadata and is validated by assertions rather than cross-platform byte comparison. Deterministic artifacts are compared byte-for-byte.

Development record: preliminary generations exposed fixture/mutation-generator problems, a missing handoff retry and persistence-error evidence path, canonical boolean/numeric coercion, response/status and health-JSON ambiguity, and a valid-hex false request accepted with a null draw call array. These were corrected before final generation. The preservation checker additionally deduplicates completed rows across committed and pending copies. Superseded runs and source revisions were stopped and retained as development evidence; their failures are excluded from final pass counts. The final matrix, fault, signal, loopback, witness, mutation, regression and preservation results all pass.

The existing completion seal and process-crash commit semantics are preserved. No arbitrary power-loss recovery is claimed. Persistent storage failures may leave INVALID archives with their retained candidates and returned error/result evidence. See [known limitations](../KNOWN_LIMITATIONS.md), [fault results](../FAULT_MATRIX.md), [preservation](../PRESERVATION.md) and [test map](TEST_MAP.md).

External retained evidence: `/home/george/dt-audit-kept/v013k/`. The final gate archive, source manifests, checksums, review brief, final-commit snapshot and final-push workflow results are recorded there. No model endpoint, clinical suite, historical live rerun, reserved/holdout seed, independent review, protocol lock, tag or release is part of these gates.
