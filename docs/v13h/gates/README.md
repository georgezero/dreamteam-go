# V0.13h engineering gates

Independent focused technical re-review: PENDING. Human review: PENDING.
Protocol lock, live inference and V0.13.0 release: NO GO. These are engineering
checks, not independent review or human acceptance.

Starting commit: 24df940db349657bcf088ae56f480526d015a945. All three committed
independent audits remain verbatim. Full before/after hashes, scripts, original
witness outputs and complete logs are retained under
/home/george/dt-audit-kept/v013h/. The remote directory is /home/george/dt-v013h
on horo, Linux amd64, using /usr/local/go/bin (Go 1.27.0).

The Pi runs only focused checks under systemd-run --user --scope -q
-p MemoryMax=2G nice -n 10. Heavy checks run remotely. Source transfers use
tracked/untracked nonignored-file tar archives excluding the protected swap
file and dt. No model endpoint, clinical suite, reserved/holdout seed,
historical live rerun, protocol lock or release is part of this work.

| Location | Gate | Result / retained evidence |
|---|---|---|
| Pi | Baseline identity and verbatim committed independent re-review | PASS; baseline-hashes.txt |
| Pi | Original lifecycle before/after | PASS; before/, after-final/, ../WITNESS_RESULTS.json |
| Pi | Original 21 partial probes and feedback disclosure | PASS; after-final/partial-probes/ and feedback-text-results.json |
| Pi | Focused lifecycle regressions | PASS; focused-final.log, 32 tests; later strict health and side-artifact cases also run remotely |
| Pi | Predecessor byte preservation | PASS; preservation.json; only two workflows and PROGRESS intentionally change |
| horo | Full Go suite | PASS; full-go.log |
| horo | Full race suite, timeout 90m | PASS; race.log; evaluation package 4,256 seconds |
| horo | Vet default/calibration/acceptance | PASS; vet.log |
| horo | gofmt | PASS; all 333 Go source files; gofmt.log |
| horo | Five cross-build targets | PASS; linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64; cross-build.log |
| horo | Four golden references | PASS; golden.log |
| horo | Benchmark verify/reproduce | PASS; 86 verification checks and 59 reproduced outcomes; benchmark.log |
| horo | V0.8c/V0.9c recorded historical replays | PASS; historical-replays.log |
| horo | Historical deterministic Go evidence | PASS; historical-go.log |
| horo | V0.12b analysis and three comparisons | PASS; v12b logs, all 420 recorded draws |
| horo | V0.13d/f fidelity with historical replays | PASS; 65 and 114 cases, zero failed checks; versioned fidelity logs |
| horo | V0.13g fidelity with historical replays and package | PASS; 125 cases, zero failed checks; v13g logs |
| horo | V0.13h protocol/schedule/check/fidelity and package | PASS; 126 cases, zero failed checks; 93 interactions; 178 package files |
| horo | Historical and successor Python suites | PASS; 542 tests including 35 successor tests and three probe tests; python logs |
| horo | Frozen evidence | PASS; 90 recordings, 3,972 decision points, frozen decision/parameter outputs reproduced; frozen logs |
| horo | V0.12a and V0.11b comparisons | PASS; two V0.12a reports; 720 V0.11b draws, 572 recorded calls, 2,299 engine requests, three comparisons; v12a/v11b logs |
| CI | Both workflows, all platform jobs and comparisons | PENDING at archival commit; final run IDs and conclusions retained in ci-summary.json and owner handoff |

The exact 25-case adversarial map is [TEST_MAP.md](TEST_MAP.md). Additional
cases check session cleanup, setup failure, production connection refusal,
unsupported health objects, side artifacts, interrupted writes, failed
checkpoint replacement and refusal to resume. A failed evidence write propagates
without promoting the prior PARTIAL checkpoint to COMPLETE.

Preliminary setup/test failures are retained in preliminary-dispositions.txt:
unsupported local extraction API, an extraction attempted before archive transfer
completed, a tuple-mutation test-helper error, and the original witness's
expectation of an initialized empty compatibility export. Corrected invocations
are distinguished from failures. None is substituted for a successful gate.

The original cleanup witness has one scripted reply for two draws; its nominal
completed-then-close branch also exhausts its script on draw two. The dedicated
cleanup-after-completion regression supplies enough replies and proves PARTIAL
with all draws complete, no aborted draw, and no complete-run comparison.

The scoped committed preservation manifest omits only the deliberately changed
existing files and a few historical pathnames. The full external baseline and
preservation report cover every one of the 2,513 starting files. All simulator,
session, API, accounting/scoring, decision workflow, templates, outcomes,
selection, historical evidence, schedules and frozen references retain bytes.
The inherited decision functions are also compared by parsed syntax tree;
only lifecycle/verification functions and successor protocol binding differ.

The successor package adds one complete invalid-number feedback interaction;
all 92 predecessor delivered transcripts and static sources remain byte-identical
package members. Human review of the exact successor package is required.
Record-only lifecycle examples use synthetic UUIDs/timestamps solely to support
byte reproduction; ordinary run recording retains actual metadata.

No focused independent re-review is performed here. The external review brief
and read-only final-commit snapshot are prepared under the retained root. A
separate independent reviewer must examine DT13G-01/02, DT13G-03 disclosure,
partial-run integrity and preservation of the prior fixes. Human identity,
independence, findings and acceptance remain PENDING.

Successor package SHA-256 of SHA256SUMS:
a4cfe19a613693f8f0669e8342c38f113cf3a5e5a9f8abf89f3246311014ab60.
MODEL_VISIBLE_PRESERVATION.json records all 92 predecessor transcript hashes
with multiplicities, all seven active static hashes, the added transcript and
its exact delivered failure message/hash. No predecessor message changes.
The additional complete interaction changes the exact review package. Human
review of this exact package remains PENDING.

GO-SOURCE.sha256 binds 335 Go/dependency files. FINAL-RUNTIME-SOURCE.sha256
binds 451 runtime/workflow files. The 35-test final successor run and source
checks pass remotely. Source timing evidence records installation before the
fidelity process and example/package generators used those files.

Complete remote gate archive: final-horo-evidence.tar.gz; SHA-256
4029d9cb845c4575581a91230c91beb2d0928edee94e619f6c51530b1abab807.
Final source checks cover every bound Go/dependency and runtime/workflow input.
All remote engineering gates passed before the final commit. CI conclusions
for that commit are recorded externally after the push; they cannot be included
in the same commit that triggers them.
