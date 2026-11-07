# V0.13f engineering gates

Independent re-review: PENDING. Human review: PENDING.
This record reports engineering checks only. It supplies no approval.

Baseline e584107c1eacbcca10163aafbf8372f22745e0d8 and tag target
8bbeca29550da67c70d69efd313baadca6438244 were verified before changes.
The original candidate checker passed at baseline: 1,964 file hashes and
10 preserved trees. The successor baseline-file test checks 1,173 hashes.

The remote gate host reports Linux x86_64, Go 1.27.0 and dreamteam
0.13.0-dev.2. The module minimum and CI toolchain selection remain unchanged.

Local focused commands use systemd-run --user --scope -q -p MemoryMax=2G
nice -n 10. Go and temporary caches are under .git/dt-v013f/. No fallback
was needed. Retained scripts and logs: ~/dt-audit-kept/v013f/.

| Location | Gate | Result / retained log |
|---|---|---|
| Pi | Canonical action, system precedence, commitments, capacity, notice and numeric-unit regressions | PASS; go-focused.log, typed-mapping.log, final-typed-regressions.log |
| Pi | V0.13d evidence byte reproduction | PASS; go-focused.log |
| Pi | Successor driver/workflow, partial verification, aborted reconstruction and explicit UNVERIFIED cases | PASS; python-v13f.log, correction-status.log |
| Pi | All scoped historical file bytes | PASS; Pure.test_historical_bytes |
| horo | Preliminary full Go suite | PASS; preliminary full-go.log; source refresh follows final typed-reference regression |
| horo | Final full Go suite | PASS; full-go-final.log; all 330 Go/dependency inputs checked |
| horo | Full race suite (-timeout 90m) | PASS; race.log; longest package 4,112.687 seconds |
| horo | Vet default, calibration and acceptance | PASS; vet.log |
| horo | Source gofmt | PASS; gofmt.log is empty; 328 committed Go files |
| horo | Cross-build linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64 | PASS; cross-build.log |
| horo | Four golden references, benchmark verify (86) / reproduce (59), four historical replays (104 checks each), V0.12b 24 starts and three reproduced reports | PASS; replay-gates.log |
| horo | V0.13d contexts/protocol/schedule/turns and historical-inclusive fidelity | PASS; 65 cases; both fidelity reports byte-identical |
| horo | V0.13f contexts/protocol/schedule/check and historical-inclusive fidelity | PASS; 114 cases, zero failed checks, 81 complete interactions |
| Pi and horo | Complete review package and exact-byte/source checks | PASS; 97 files; package-final.log, preservation-final.log |
| horo | Frozen decision and parameter evidence from reconstructed recorded replays | PASS; frozen-launcher.log |
| horo | Python agent (168), evidence (147), repeatability (88) and acknowledgment (51) tests | PASS; python logs |
| horo | All alternatives workflow/driver tests with an engine supplied | PASS; 29 tests, no skips; python-alternatives-engine-final.log |
| CI | Cross-platform successor workflow and legacy evidence regeneration | PENDING at this archival commit; checked after push, with run IDs/results retained in ci-summary.json and the handoff |
| CI | Historical candidate verification in current HEAD | FAIL (mechanical current-commit witness); CI execution follows push. BLOCKED: immutable exact-tree verifier conflicts with required successor additions; invocation exception awaits owner decision |

An initial remote launch preceded source extraction and failed with
"directory prefix . does not contain main module or its selected dependencies".
It is retained as setup-full-go.log and was not counted as a code gate.
The subsequent preliminary full Go suite passed. Its first race launch was
stopped by exact process IDs for a source refresh; it is not a race pass.
The final-source race suite subsequently passed. full-go-final.log is the
authoritative normal full-suite result; earlier full-go/source-refresh/preliminary
logs are retained as earlier snapshots, not substituted for the final result.

No model endpoint, clinical suite, historical live rerun or reserved/holdout
seed is part of these gates. Independent re-review is preparation only.

The first adapted replay script stopped at a frozen check because the committed
reference directory omits traces.jsonl. That failed invocation is retained in
frozen-evidence.log and replay-gates.status. The committed corpus replays were
then reconstructed, and both analyzers and frozen checks passed against the
new outputs (frozen-gates.status). Subsequent successor steps passed separately
(successor-gates.status). The original compound-script failure is not erased.

The current-commit historical candidate checker exits 1: three exact directory
trees differ through successor additions, and five allowed existing files
differ. The original baseline passes all 1,964 file hashes and ten trees in an
isolated checkout. The current check's failure does not indicate alteration of
the 1,173 preserved historical files. Its exact output is retained in
current-candidate-check.log; the proposed invocation-only patch is retained as
historical-ci-invocation.patch and has not been applied.

The before-fix-witnesses.log excerpt is copied verbatim from the initial
go-focused.log. Its passing assertions confirm the preserved implementation's
defects; they do not approve the historical behavior. Source line numbers
therefore identify that earlier test-file snapshot.

The first broad format walk traversed .git scratch module caches and compiler
output. No repository Go file appeared in gofmt-initial.log. The compound
go-gates.status therefore records FAIL after all race and vet checks passed.
The corrected source-only format check uses the committed Go-file inventory
on the transfer host, and git ls-files in successor CI. Its output is empty.
format-cross.status records PASS for that check and all five cross-builds.
No dependency or generated cache file was reformatted.

Complete retained logs live under ~/dt-audit-kept/v013f/horo/;
RETAINED-LOGS.sha256 pins every retained remote log and selected command report.
Selected complete logs and the verbatim before-fix excerpt are included here.
The archived commit necessarily precedes its push-triggered CI runs. Those
run IDs, conclusions and any acquisition retry are retained separately in
~/dt-audit-kept/v013f/ci-summary.json and reported in the final handoff.
The old workflow and its current-tree candidate check remain unchanged.

Remote cleanup: PASS. After the complete log archive was copied back, the
first exact-path removal encountered read-only module-cache directories.
The error was "Permission denied" beneath
~/dt-v013f/.git/dt-v013f/go-mod/; complete captured stderr is retained in
cleanup-attempt1.log. Owner write permission was restored on 121 owned cache
directories inside that already authorized deletion target. The exact
~/dt-v013f removal then succeeded (cleanup.log). No other path was removed.
This was an ordinary cache-mode repair, without privilege escalation, sandbox
configuration changes or a scratch-space fallback.
