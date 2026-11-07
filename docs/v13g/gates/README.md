# V0.13g engineering gates

Independent technical re-review: PENDING. Human presentation review: PENDING.
This record reports engineering verification and supplies no human acceptance.
Protocol lock, live inference and V0.13.0 release remain NO GO.

Starting commit: cf9a12bf12fc94089e8a99f73dea12edd0cfd019. The two independent
audits remain verbatim. The v0.12.0 target remains
8bbeca29550da67c70d69efd313baadca6438244. The historical candidate checker
passes at e584107: 1,964 file hashes and ten preserved trees. The successor
preservation manifest checks 2,205 scoped predecessor files.

Heavy gates run on horo, Linux amd64, with Go 1.27.0 from /usr/local/go/bin.
The engine reports 0.13.0-dev.2. GO-SOURCE.sha256 binds all 335 Go/dependency
inputs to the full suite and race run; these inputs did not change after launch.
FINAL-RUNTIME-SOURCE.sha256 binds 445 source/workflow inputs to the final remote
checks. Local focused tests use systemd-run --user --scope -q -p MemoryMax=2G
nice -n 10. Scratch and retained evidence are under ~/dt-audit-kept/v013g/;
remote scratch is the dedicated ~/dt-v013g/.

| Location | Gate | Result / retained log |
|---|---|---|
| Pi | Focused batch, numeric and canonical regressions | PASS; go-focused.log |
| Pi | Original before/after witness assertions | PASS; ../WITNESS_RESULTS.json and retained before/after outputs |
| Pi and horo | Contract/source identities and 2,205 preserved predecessor hashes | PASS; final-identities.log; final-go-source.log and final-runtime-source-recheck.log bind final remote inputs |
| horo | Full Go suite | PASS; full-go.log |
| horo | Full race suite, timeout 90m | PASS; race.log, longest package 3,891.664 seconds; preliminary failure retained separately |
| horo | Vet default, calibration and acceptance | PASS; vet.log |
| horo | gofmt, 333 source files | PASS; gofmt.log is empty |
| horo | linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64 builds | PASS; cross-build.log |
| horo | Four golden references | PASS; golden.log |
| horo | Benchmark verification/reproduction | PASS; benchmark.log, 86 checks / 59 reproduced outcomes |
| horo | Four V0.8c/V0.9c historical replay prerequisites | PASS; historical-replays.log, 104 checks each |
| horo | Historical inventory and deterministic Go evidence | PASS; historical-go.log |
| horo | V0.12b analysis and all three report comparisons | PASS; v12b-check.log, v12b-analysis.log |
| horo | V0.13d contexts/protocol/schedule/fidelity byte reproduction | PASS; 65 cases, zero failed checks; v13d logs |
| horo | V0.13f contexts/protocol/schedule/fidelity/interactions/package byte reproduction | PASS; 114 cases, zero failed checks, 81 interactions; v13f logs |
| horo | V0.13g contexts/protocol/schedule/check/fidelity | PASS; 125 cases, zero failed checks, 92 complete interactions; final-fidelity.log |
| Pi and horo | Complete human-review package, byte/source checks | PASS; 115 files; final-package.log; human review PENDING |
| horo | All alternatives tests with engine, historical and successor | PASS; 29 + 21 tests, no skips; final-python historical/current alternatives logs |
| horo | Python agents, evidence, repeatability and acknowledgment | PASS; 168 / 147 / 88 / 51 tests; final-python logs |
| horo | Frozen decision and parameter evidence | PASS; frozen-launch.log, 90 recordings / 3,972 decision points; unchanged frozen outputs |
| horo | V0.12a acknowledgment | PASS; v12a.log, both scripted reports byte-identical |
| horo | V0.11b workflow evaluation and all three report comparisons | PASS; v11b.log, 720 historical draws, all three reports byte-identical |
| CI | Both workflows, all platform jobs and comprehensive comparisons | PENDING at this archival commit; final run/job results retained in ci-summary.json and owner handoff |

Package: docs/v13g/review-package/. SHA-256 of SHA256SUMS:
9e725617cf140384fb836d2d3449491cecdec517db800241a6c42b9172e229eb.
The package has 115 total files, including 92 complete interaction transcripts
and seven separate record-only partial JSON examples. No human finding,
reviewer assignment or acceptance is supplied by these engineering checks.

Before/after witnesses are summarized and hashed in ../WITNESS_RESULTS.json.
The original scripts and full adapted outputs remain under the retained root's
witnesses/, before/ and after/. The adapted witness setup now explicitly creates its named scratch directories
before configuring temporary files; the final numeric/partial witness reruns use
those directories. Before-fix passing assertions describe defects;
they do not approve those behaviors. The 65-action bypass, mixed partial
submission, execution misattribution, missing opening-failure artifact, numeric
schema mismatch, historical CI blocker, precision/collection defects and
unsupported stops were reproduced before the successor corrections.

The first preliminary fidelity invocation failed while writing the new partial
examples because its generator referenced an undefined local variable. That
complete failure is retained in preliminary-v13g-fidelity.log. A second loaded
instance was stopped by its exact process IDs, and its status/output are kept
as interrupted-final-evidence.status and interrupted-final-fidelity.log. The
corrected generator runs in a fresh output directory. No failed or interrupted
invocation is counted as a successful gate. After corrected fidelity passed
125 cases, a follow-on copy failed because its destination did not yet exist.
The failure is retained in fidelity-copy-setup.log/status. Creating that named
directory allowed package/tests to continue from the successful output without
substituting another fidelity result.

The first full race run failed only TestHoldoutContainment: while it was
running, a historical checkout had been extracted into the visible tree and
the source scanner correctly detected duplicate unreviewed paths. Other packages
passed and no data race was reported. The complete output remains in
preliminary-race.log and preliminary-go-gates.status. The checkout was moved
under the scanner-excluded .git directory, matching the corrected CI layout.
The containment regression and the full race suite both pass after correction; no check is
weakened and no source is changed to bypass the guard.

The original historical compound script invokes the preserved V0.13f manifest
against the changed CI workflow. Its failure remains in replay.status and
initial-v13f-preservation.log. The corrected invocation uses the complete
cf9a12b checkout. The complete original V0.13f suite also runs there because
its source hashes pin that workflow. Current-tree successor tests independently
check the new code and preservation. Later frozen-evidence, V0.12a and V0.11b
checks run in full after the corrected invocation; none is skipped or suppressed.

The first local archive extraction requested an argument unavailable in the
installed Python version and extracted nothing. The retry verified the archive
checksum and every member path/type, then used the supported extraction API.
The imported package and contract checks passed without source changes.

Both CI workflows use historical checkouts for historical candidate/source pins,
then execute current-tree tests and all later checks. The V0.11b comparison
command fails before copying outputs when any report differs; a deterministic
regression injects a mismatch in each report. Artifact upload and comprehensive
Linux amd64/Linux arm64/macOS arm64 comparisons remain required.

The archived preparation commit necessarily precedes its push-triggered CI runs.
Their run IDs, job results and any acquisition retry are retained separately in
~/dt-audit-kept/v013g/ci-summary.json and reported in the owner handoff. The
external re-review brief identifies the final commit and read-only archive hash.
Independent re-review is prepared only; a separate reviewer must conduct it.

No model endpoint, clinical suite, historical live rerun or reserved/holdout seed
is part of these gates. The protected untracked swap file and dt are excluded
from stages and source archives. All earlier evidence and agents retain their
bytes; new evidence and contracts use successor paths and identities.

All remote component gates pass. The full retained log/evidence archive is
final-horo-evidence.tar.gz under the retained root; its SHA-256 is
50b573a7e715bde39c3643dec357454917f97fe25d8822ecf09fee469fc4f966.
RETAINED-LOGS.sha256 pins the complete retained remote logs; selected complete
logs are included here. Remote cleanup follows durable archive retrieval and
CI completion; its result is retained in cleanup.log and the external brief.

The final staged whitespace check removed one trailing space from v13g.py.
Its parsed AST is byte-identical before/after; source-whitespace-adjustment.json
records both source hashes and the common AST hash. All 21 successor tests,
identity checks and package checks pass again. No Go input, delivered text,
fidelity output or package file changes. The final 445-input source manifest
is rechecked after correction. The supplemental final-whitespace-evidence.tar.gz
SHA-256 is ce4e79ea4edd8b63f8929f9201f857b31cd83a5ffdfd845afea60b452c983a66.
The original complete gate archive remains unchanged and retained separately.
