# Slice 5C.4o release-gate audit

**Audited revision:** commit ca9cbd1 (0.2.0-dev.24), the 5C.4n certification pipeline remediation of the 5C.4m blockers M1–M4 and M6 (not locked, not frozen, not in force), under the owner's scope decision of 2026-10-20 on the study swap (accepted only for trusted in-process evaluation). Audit date 2026-10-20. Line references are to ca9cbd1.

The audit was a bounded, independent, read-only release-gate review. The auditor read a `git archive` snapshot of ca9cbd1 on a separate machine, in a dedicated directory outside the repository, with the previous auditor's fabricated result file and fixtures and the maintainer's mutation script supplied beside it for reproduction. Disposable test fixtures, trial mutations and logs lived in a work copy and a scratch directory beside the snapshot; the snapshot (851 files) was verified byte-identical before and after. It ran no strategy, reference seed, holdout seed, reserved stream PCG(20261201, ·), unseal, activation or freeze, and did not execute the prospective reserved command. The paths below point into that directory, which was removed after the audit; the fixture source and logs are kept by the maintainer outside the repository for remediation.

* **Everything below the rule is the auditor's report as delivered, unchanged.**
* **No remediation is in this commit.** Maintainer checks, if any, will follow in a separate document.

---

**DreamTeam Slice 5C.4o — independent release-gate review**

Reviewed revision: **ca9cbd1**, **0.2.0-dev.24**. Review date: **2026-10-20**. All repository file:line references below refer to the immutable `snapshot/`, not trial changes in `work/`.

**NO GO for a prospective protocol-lock commit.** M1, M3 and M6 are closed within the approved scope. M2 and M4 remain open: incomplete or contradictory measurements still pass the mandatory criteria, and the offline reader silently accepts a malformed, truncated suffix. These are **Certification-critical** findings O1 and O2. Neither finding demonstrates that imported measurements can obtain certification through the repaired reader; that boundary now works.

| Release decision | Verdict |
|---|---|
| 1. M1 trusted certification | **GO**, for the actual trusted execution and non-certifying reader boundary. |
| 2. M2 measurement completeness | **NO GO** — O1. |
| 3. M3 protocol restriction | **GO** — the owner's restriction is prominent and matches the certification boundary. |
| 4. M4 error handling | **NO GO** — O2; the other tested error paths behave correctly. |
| 5. M6 validation evidence | **GO** — valid-record generation and byte-identical regeneration confirmed. |
| 6. Historical compatibility | **NO GO for the complete assurance claim** — checked policy bytes and semantics pass; exact historical replay and the parent-source comparison remain unverified, O4. This is an evidence limitation, not a demonstrated regression. |
| 7. Prospective protocol-lock readiness | **NO GO** — close O1 and O2 before locking. |
| 8. Reserved calibration readiness | **NO GO** — the blockers remain, the protocol is not locked, and this build refuses the reserved family. No reserved execution was performed or authorized. |

**Scope, preservation and reproduction environment**

The initial `cd snapshot && sha256sum -c --quiet ../snapshot.sha256` succeeded, exit 0, for all **851 files**. The final identical check also succeeded, exit 0. The final inventory matched both the initial inventory and the digest list: **no snapshot file changed, was added or was removed**. Evidence: `scratch/snapshot-start.log`, `scratch/snapshot-start-inventory.txt`, `scratch/snapshot-end.log` and `scratch/snapshot-end-inventory.log`.

All fixtures, scripts, regenerated evidence and logs are retained in `scratch/`. Added test sources were copied only into `work/`. The two trial mutations were restored byte-for-byte; no trial modification is represented as a remediation. No commit, tag, lock, freeze, activation or unseal was performed. No reference, holdout or reserved stream was executed. Simulator runs used no actions and seeds **901–910** only. Fidelity generation used **PCG(1, ·)** and **PCG(20261021, ·)** only. Synthetic measurement rows and evaluator records are not simulator executions. The adjudicator's fixed PCG(7, 7) predictor was exercised; it is not the reserved stream.

Commands below are relative to `work/`, unless explicitly stated otherwise. Source `../scratch/env.sh` first; it sets the prescribed temporary/cache locations under `scratch/`, `GOFLAGS=-mod=mod`, `GOMEMLIMIT=5GiB`, `GOMAXPROCS=4` and `GOPROXY=off`. The installed compiler was **Go 1.27.0, linux/arm64**. Heavy jobs ran one at a time, with `nice -n 10`, package parallelism `-p 1`, explicit timeouts and output streamed to logs. No temporary files were directed to `/tmp`.

**O1 — Measurement validation still accepts partial observations and contradictory measurements. Classification: Certification-critical. Blocks M2 and lock.**

The repair establishes many useful shape checks, but not the claimed completeness and coherence of every adjudicated measurement. `lineProblems` permits any nonempty urgent observation list whose length does not exceed `matchedSeeds` (`internal/eval/reserved_manifest_test.go:1342`, `:1350`, `:1356`). It does not require the observations belonging to all sufficiently populated matched pairs. It checks each `Difference` for finiteness but does not reconcile it with its counts (`:1345`), or reconcile the group increase and PASS status with the observations (`:1324`, `:1327`). Construction counts are checked separately, without bounding injected orders by served orders (`:1292`).

This matters to the actual adjudicator: `predictAcceptance` iterates only the supplied urgent observations (`:1960`), and `reservedCriteria` uses that prediction to judge T5a (`:1728`, `:1754`). T8's matched-pair equality (`:1779`) does not detect a partial observation list. `runCertified` uses these same criteria without a subsequent completeness check that closes these gaps (`:1072`).

An independently constructed, full-layout file contains **3,904 rows**, all groups, remainders, replicates and pairs, and a complete urgent observation set. All eleven mandatory criteria PASS; coverage has zero problems. It was generated by `scratch/make_5c4o.py`, without calling the maintainer's fabrication helpers, producing records or drawing a simulation stream. Its SHA-256 is:

`197b6e715f6056b5453804d00f857077b9386f69b07c5aabb3a9a4205976e6d1`

The following independent changes to that baseline each leave **every mandatory criterion PASS and coverageProblems empty**:

| Change | Why it is invalid |
|---|---|
| For every NM1 replicate, retain only the first of 40 urgent observations. | All 40 pairs still claim 100 eligible urgent orders, 25 served orders and matched identity. The other 39 observations are required, not thin-denominator exclusions. The prediction silently uses one seed. |
| Set NM1's first urgent `Difference` to −0.75. | Its counts remain candidate 75, baseline 25, orders 100: the difference is +0.50. |
| Set S-A85's urgent group increase to 0.90. | The supplied per-seed differences remain zero, Gate B remains PASS and the group remains unrejected. |
| Set NM1's first pair's injected count to 26. | That pair has only 25 served eligible orders before injection. The injection is impossible. |

The protocol requires prediction from the group’s own seeds (`docs/calibration/clinical-5c4n/reserved-validation-protocol.md:381`). The legitimate exclusion of thin seeds does not justify the first result: `internal/eval/lostdemand.go:249` omits observations below the minimum, whereas this witness explicitly supplies 100 eligible orders on every pair. The original gate computes a difference from its counts and aggregates those differences (`lostdemand.go:253`, `:272`); the summaries above cannot represent that computation consistently.

Minimal reproduction:

```sh
source ../scratch/env.sh
python3 ../scratch/make_5c4o.py
cp ../scratch/zz_audit_o_test.go internal/eval/zz_audit_o_test.go
nice -n 10 go test -p 1 ./internal/eval \
  -run '^TestAuditOMeasurements$/^(partial_T5_differences|difference_contradicts_counts|PASS_despite_90_point_increase|impossible_injected_count)$' \
  -count=1 -timeout=10m -v
```

The characterization tests report `allPASS=true coverageProblems=0 T5a=pass T8=pass`. Their passing test-process status records successful reproduction of the defect; it does not mean the measurements are acceptable. Full fixture source: `scratch/zz_audit_o_test.go`; final verification log: `scratch/verified-tests.log`.

The original zero-observation defects are closed: missing T2G severity and wholly absent T5 observations do not PASS. Independent missing-field, NaN/Inf, negative-count, impossible-matched-count, duplicate-pair, duplicate-observation, foreign-seed, invalid-T4-attribution and incomplete-T8 fixtures are also refused. A rejection/rule-presence mismatch is caught. These successful checks do not cover the partial and cross-field cases above.

Required closure is confined to M2: establish the required observation set and reconcile these already-present measurements before allowing favorable criteria. Do not change clinical cohorts, thresholds, attribution, tolerances or the manifest to accommodate the invalid summaries. This finding does not assert that current record generation emits these inconsistencies; it demonstrates that the release-gate validator still accepts them.

**O2 — A truncated second JSON value on a result line is ignored, with a successful reader exit. Classification: Certification-critical. Blocks M4 and lock.**

`analyzeFile` constructs a decoder for each line, decodes one object, then immediately appends it (`internal/eval/reserved_manifest_test.go:2033`). It never checks that the rest of the line is whitespace/end-of-input. `DisallowUnknownFields` checks the first object only. Consequently, a complete valid row followed on the same line by ` {"scenario":` is accepted even though the file contains a truncated JSON value.

The independent fixture removes the final newline from the all-PASS file and appends ` {"scenario":\n`. The actual `TestReservedCriteria` entry point reports zero coverage problems, emits all eleven PASS statuses and exits **0**. Its adjudication is byte-identical to the clean file’s adjudication. Its certification correctly remains **`not certified`**. This is an input/error-handling failure, not an M1 certification bypass. It contradicts the requirement that malformed or truncated input fails before output (`docs/calibration/clinical-5c4n/reserved-validation-protocol.md:672`).

Minimal reproduction, after generating the files above:

```sh
DT_MANIFEST_IN=../../../scratch/trailing-truncated.jsonl \
DT_CRITERIA_OUT=../../../scratch/trailing-repro.criteria.json \
nice -n 10 go test -p 1 ./internal/eval \
  -run '^TestReservedCriteria$' -count=1 -timeout=5m -v
```

The paths are interpreted in the test package directory, `work/internal/eval`. Fixture source: the final lines of `scratch/make_5c4o.py`; characterization: `TestAuditOTrailing`; entry-point evidence: `scratch/reader-trailing.log`, `scratch/reader-trailing.json` and `scratch/reader-matrix.log`.

Require consumption of the entire input line before accepting it. Retain the repaired reader's non-certifying status. No clinical change is needed.

**O3 — Documentation claims and the prospective lock recipe need correction. Classification: Documentation-only. No additional broad review required.**

First, a complete, all-PASS fabricated file does **not** necessarily cause a failing reader exit. The independent file exits 0 and remains `non-certifying analysis` / `not certified`. The old 5C.4m file exits nonzero because its measurements are incomplete. The maintainer response conflates these outcomes at `docs/audits/slice-5c4m-provenance-protocol-lock-audit-response.md:62`. The executable and protocol correctly make the certification field, not successful process termination, decisive. Reproduce with the O2 command using `own-all-pass.jsonl`; the matrix records both files.

Second, the surviving T5a mutation is equivalent only in a restricted sense. Source and mutation results are discussed below. The unqualified equivalence statement at the response's `:84` and `docs/MEASUREMENTS.md:682` should describe that restriction. Neither wording issue requires another broad review.

Third, the prospective lock recipe has inconsistent preconditions. It requires every committed test to pass at the lock (`reserved-validation-protocol.md:614`) while allowing only the family-admission change, LOCKED status and digest update (`:627`). But `TestReservedManifestRefusesReserved` explicitly fails when the reserved family is admitted (`internal/eval/reserved_manifest_test.go:2405`), as does `TestCertifiedExecutionSmall` (`internal/eval/remediation_5c4n_test.go:536`). Thus the stated changes alone cannot produce the required passing suite. The current refusal tests are correct for the current pre-lock build; the lock instructions must explicitly account for their corresponding admission/refusal expectations. This is a defect in the prospective recipe, not evidence of a reserved execution. Minimal static reproduction is to compare those three cited predicates; no enabling trial or reserved draw was performed. Correct this recipe as documentation/preflight bookkeeping alongside the focused blocker fixes.

**The surviving T5a mutation, examined independently**

`reservedCriteria` always starts with `manCoverage` (`internal/eval/reserved_manifest_test.go:1414`). Some malformed rows skip `lineProblems`, but the same coverage path records the malformed shape, duplicate/out-of-range group, construction error or invalid record (`:1227`). Those skips do not create complete coverage.

For the pinned manifest, all five T5 scenarios have replicates. On a validated injected T5 line, the urgent list is nonempty. With the hard-coded 20,000 draws, `predictAcceptance` returns an integer acceptance count divided by 20,000 (`:1981`); the explicit empty-list branch is its NaN return (`:1964`). A nil injection returns 0 or 1 directly. No validated line of the pinned manifest was found that returns NaN. A partially populated list can nevertheless return a finite, favorable prediction: that is O1, not protection supplied by `unpredicted`.

The survivor was tested alone and with the empty-urgent-list completeness check removed, only in `work/`, using `scratch/run_mutations.py`. The original source was restored after each trial.

| Input | Original | Guard removed | Guard and empty-list check removed |
|---|---|---|---|
| Pinned manifest; all injected NM1 observations absent | T5a INCONCLUSIVE; T8 INCONCLUSIVE | Same | T5a PASS; T8 PASS |
| Same absence, plus a separate NM2 prediction/observation mismatch | T5a INCONCLUSIVE | T5a FAIL | T5a FAIL |
| Synthetic alternate manifest: NM1 has zero replicates; one row per group; urgent observations absent | T5a INCONCLUSIVE; no coverage problems | T5a PASS; no coverage problems | T5a PASS; no coverage problems |

The second case disproves exact equivalence even on a pinned-manifest malformed input; both statuses still prevent confirmation. The third supplies a line that passes `lineProblems` yet predicts NaN: the no-observation check is conditional on `s.Replicates > 0`, whereas T5 classification follows the scenario's criteria. This alternate manifest was constructed only in memory. Its bytes would fail the trusted execution's manifest pin; it is not a certified-path bypass in this candidate.

The maintainer's completeness regression survives removal of the guard alone and fails when both checks are removed. The independent cases show why that is evidence of redundancy for complete pinned inputs, not universal semantic equivalence. The guard remains present in the reviewed snapshot.

Reproduce:

```sh
cp ../scratch/zz_audit_o_alt_test.go internal/eval/zz_audit_o_alt_test.go
nice -n 10 go test -p 1 ./internal/eval \
  -run '^TestAuditOSurvivor(AltManifest)?$' -count=1 -timeout=10m -v
python3 ../scratch/run_mutations.py
```

Logs: `scratch/verified-tests.log`, `scratch/mutation-survivor.log`, `scratch/mutation-combined.log`. The other 35 reported mutations were not independently rerun; no 36-of-37 assurance is inferred from these two trials.

**M1 — Trusted certification is closed within the approved boundary**

The actual entry point reads settings, refuses the reserved family, verifies inputs and records source identity before generation (`internal/eval/reserved_manifest_test.go:999`). `manRunScenario` constructs paired records and calls `protocolDecision` or `gateBDecision` in memory (`:921`). `runCertified` accumulates those measurements, calls `reservedCriteria` and then `certify` (`:1058`). Only after that return does the entry point serialize and write the lines and criteria (`:1025`, `:1036`, `:1039`). There is no result-file reopening or editable measurement intermediate in this path.

The five runtime inputs are compared with compiled digests; the manifest's candidate pin is separately checked (`:716`, `:728`). The engine version and package source digest are recorded. The protocol explicitly relies on a clean, tested lock checkout for the binary's identity; it does not claim that `sourceSHA256` is a binary digest. Family and full-scenario seed identity control the confirmatory designation (`:806`). Development or partial execution cannot obtain `certified GO`.

`analyzeFile` always calls `certify` with `Trusted:false` (`:2046`). Independently rereading the prior 3,904-line file, with verified SHA-256 `5c5748e60174f5db599308dc08b051b92a484eb7e3066d08b531a6ecb9bab408`, reproduced **38,102 problems**, a failing exit and no certification. Independently rereading the new complete all-PASS file produced zero problems and no certification. The new file's ordinary successful analysis exit is documented in O3.

The call-site, environment-variable, build-tag and helper search found no additional existing external-input certification entry point. There is an intentional synthetic decision-table call that constructs `certified GO` without generation (`internal/eval/remediation_5c4n_test.go:119`): it supplies a mock trusted context and asserts the result. It does not publish an execution certificate or import records into the trusted path. `certify` is a private function trusting its caller; the trust boundary is enforced by the actual entry points, not by an authenticated capability inside that helper.

The input-pin regression, development execution, fabricated-reader tests and coverage tests passed. This closes the substituted-file attack, independently of the remaining measurement-validity defect O1.

**M2 positive controls and T8 correction**

The generated two-seed identity witnesses on PCG(1, ·) produced:

| Witness | Urgent matched | Soon matched | Routine matched |
|---|---:|---:|---:|
| W-ID-REMOVED | 1 | 1 | 1 |
| W-ID-REPEATED | 1 | 1 | 1 |
| W-ID-CHANGED | 1 | 2 | 2 |

Every line passed `lineProblems`; provenance named precisely seed 0. This independently confirms the per-kind distinction in `idUnmatched` (`internal/eval/reserved_manifest_test.go:1823`). The equality check is appropriately strict; restoring the urgent match or treating the other two priorities as unmatched is incorrect.

`TestReservedManifestConstruction` also accepted genuine lines from every scenario on the debug family. The two committed construction-check passes have group summaries identical to 5C.4l's and are marked development/not certified. Their full 40- and 200-seed executions and raw-line byte-identity claims were not independently rerun; the archive contains their summaries, not the raw JSONL files. These limitations do not excuse the independently reproduced O1 cases.

**M3 — The approved protocol restriction is closed**

The protocol's opening block, `docs/calibration/clinical-5c4n/reserved-validation-protocol.md:14`, prominently restricts certification to trusted in-process generation and adjudication. It expressly excludes externally editable records and imported summaries, states that the outcome-only swap remains undetected, requires authenticated study-order provenance before future external-record certification, and forbids changing the clinical rules to address this limitation.

The same restriction appears in the limitations and execution sections (`:573`, `:608`). It matches the repaired reader/producer boundary. The study-swap regression still demonstrates GO for the swapped synthetic record and NO GO for its unswapped control. That is the accepted scoped limitation, not a newly discovered regression and not grounds for reopening clinical policy.

**M4 — Other error paths and the prospective command**

The entry-point matrix independently checked malformed/nonpositive seed counts, unknown/repeated/trailing-empty scenario IDs, malformed families, empty input, truncation within an object, truncation at a line boundary and output to a directory. Invalid settings and ordinary malformed input failed explicitly; incomplete input produced parseable non-certifying criteria and a failing exit; output-write failure failed. `TestReaderErrors` also verified that an unserializable adjudication creates no criteria file. O2 is the exception to the claimed malformed-input coverage.

The small actual producer execution used development family 20261021, two seeds each for W-ID-CHANGED and W-HORIZON. It wrote its outputs, recorded development/not certified and failed for 47 omitted-scenario coverage problems. Separate producer invocations with malformed settings and an unwritable output path also failed explicitly. No successful exit was treated as confirmation. Evidence: `scratch/development-entry.log` and its two output files.

Protocol §8 was inspected only. Its prospective command unsets `DT_MANIFEST_SEEDS` and `DT_MANIFEST_SCENARIOS`, sets the reserved family, invokes only `TestReservedManifestRun`, specifies both outputs and gives a 2,000-minute timeout (`reserved-validation-protocol.md:637`). This selects 49 scenarios at their manifest counts: 61,000 scenario assignments, 56,000 distinct base pairs and 3,904 output rows. The command does not use `DT_MANIFEST_IN`; a leftover value cannot substitute measurements in the producer. With no family setting, the code defaults to the development family; with no output path it skips instead of certifying. The documented command supplies those settings. `familyAllowed` refuses the reserved family before generation in this revision (`internal/eval/reserved_manifest_test.go:662`).

As written, the lock procedure would change only the family allowance, protocol status and protocol digest (`reserved-validation-protocol.md:627`); it requires clean-tree and test preconditions, absent output paths, a separately authorized draw and no retry after consumption. Its test-precondition conflict is recorded in O3. The `$BIN` location and `<lock directory>` remain prospective placeholders. No part of that reserved execution was run. The present NO GO does not authorize the proposed enabling change.

**M6 — Validation evidence independently regenerated**

`recordBaselineWith` now supplies the scenario fingerprint to `NewEvalRecorder` and measures integrity before returning the record (`internal/sim/clinical_profile_test.go:94`, `:126`). The integrity regression passed with a genuine seed-901 record; removing the fingerprint was rejected, including the helper's refusal path. Both evidence-producing tests use this helper. Their development seed set was verified as exactly 901–910.

The complete access and follow-up evidence was regenerated through the unchanged producer tests. Both regenerated files are byte-identical to the 5C.4n files and their 5C.4l counterparts:

| Evidence | SHA-256 |
|---|---|
| `sim-access-5c4n.json` | `41760c8589ae4fde1658343205ed082ba60f5e0c1ff310e856d3fa22e7c74001` |
| `followup-finance.json` | `2d24b9363b0b3fe29cd738a832bc8e2114e12a228962b3e9508fb2d111bd22db` |

Reproduction:

```sh
DT_SIM_ACCESS=../../../scratch/sim-access-regenerated.json \
DT_FIN_FOLLOWUP=../../../scratch/followup-regenerated.json \
nice -n 10 go test -p 1 ./internal/sim \
  -run '^(TestValidationHelperIntegrity|TestSimulatorAccessValidation|TestFollowUpFinancialObservation)$' \
  -count=1 -timeout=30m -v
python3 ../scratch/check_artifacts_5c4o.py
```

The simulator tests completed successfully in **188.515 seconds**. Evidence: `scratch/sim-validation.log`, both regenerated JSON files and `scratch/artifacts-check.json`. Numerical agreement is now supported by records that pass the actual integrity checks.

**O4 — Complete historical assurance cannot be independently established from this archive under the seed restriction. Classification: Out of scope for additional remediation; evidence limitation.**

The v4 policy digest equals the value recorded in the previous independent review. V4 neutrality and historical d/e/f candidate compatibility tests passed; the h-boundary and j-provenance controls retained their documented distinctions. The current candidate and manifest match their previously approved digests exactly. No new clinical-policy change or historical semantic regression was demonstrated.

Replay support through dev.23 is registered at `internal/session/record_v1.go:194`, including dev.23 at `:216`. Metadata decoding succeeded. Exact replay was not executed: the dev.22/dev.23 fixtures use seed 52 (`internal/session/dev23_test.go:21`), and the earlier fixtures likewise fall outside the permitted simulator seeds. Merely decoding them is not an exact-replay result.

The supplied archive and empty review-root `.git` contain no 2533a39 source tree. An operational `internal/sim` inventory is retained in `scratch/artifacts-check.json`, but cannot establish a parent diff without that parent. The maintainer's assertion that only test files changed is therefore not independently verified. Historical non-use of protected streams likewise cannot be proved from an archive; this review's own non-use is established by its commands and fixtures.

Minimal evidence reproduction: inspect the cited fixture seed and compatibility registration; run the metadata-only `TestAuditOReplayMetadata` from `scratch/zz_audit_o_session_test.go`. It never calls `Load` or advances a simulation. Log: `scratch/replay-metadata.log`. These assurance limits are not a third demonstrated implementation blocker and do not reopen previously approved policy decisions.

The deferred M5 capacity-grid defect remains outside the pinned manifest: the manifest's candidate imaging-capacity changes are reductions, not increases (`internal/eval/reserved_manifest_test.go:133`; `internal/eval/oc_fidelity_test.go:1136`). It was not expanded into this release gate.

**Verified artifact identities and execution evidence**

| Artifact, relative to the snapshot | SHA-256 |
|---|---|
| `docs/calibration/clinical-5c4b/candidate-policy-v5.json` | `83d481b7040b69d78a58de68afc938a588b5b37ba396438509f38c56ca163948` |
| `docs/calibration/clinical-5c4l/scenario-manifest.json` | `250bc7fc835d99140212505e0e79f74f2da1484dbe856fea6a5b93eec9153ef4` |
| `docs/calibration/clinical-5c4n/reserved-validation-protocol.md` | `7ab8988122688d563d0dbb5ba521256d7001c206b44d9be03c25fe213907af33` |
| `docs/evaluation/policy-v0.2-eval-4.json` | `4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3` |
| `internal/eval/testdata/sim-profile-5c4b.json` | `ac8621889813a2d582b4b19674f04871c5711d27d0ac1620839ba01df4596100` |

The exact synthetic sources are `scratch/make_5c4o.py`, `scratch/zz_audit_o_test.go` and `scratch/zz_audit_o_alt_test.go`. `scratch/reader_matrix.py` invokes the actual reader for all error cases; `scratch/producer_checks.py` exercises malformed producer settings and a small development execution; `scratch/run_mutations.py` performs and restores only the two requested mutations. `scratch/check_artifacts_5c4o.py` records digest and evidence comparisons. The focused evaluation tests completed successfully in 570.119 seconds; the metadata-only session test passed in 0.050 seconds. Original work-copy files were restored exactly (`scratch/work-diff.log`). The focused regression command and final outcomes are retained in `scratch/verification-commands.txt` and `scratch/verified-tests.log`; a full uncached repository suite was not run.

**Release disposition:** ca9cbd1 is **NO GO** for a prospective protocol-lock commit because **O1 (Certification-critical, M2)** permits incomplete/inconsistent measurements to pass and **O2 (Certification-critical, M4)** silently accepts malformed/truncated input with success. Close those precise gaps and rerun their focused reproductions. The documentation-only corrections in O3 do not justify another broad review. Preserve the accepted trust restriction, clinical policy and reserved-family prohibition; no reserved calibration is authorized by this report.
