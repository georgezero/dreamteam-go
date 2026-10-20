# Response to the Slice 5C.4m provenance and protocol lock audit (maintainer)

This is the maintainer's account of what was done with each finding of
`slice-5c4m-provenance-protocol-lock-audit.md`.

* **The audit is unchanged.** It was committed alone and verbatim
  (c456c94) before any remediation, and this document is separate from
  it.
* **The remediation is Slice 5C.4n** (0.2.0-dev.24), the final planned
  remediation before a release-gate review, under the owner's decisions
  of 2026-10-20:
  * the study swap is accepted only for trusted in-process evaluation;
    externally editable, imported or untrusted records are not certified
    without authenticated study-order provenance;
  * close M1, M2, M3, M4 and M6 without changing the clinical policy,
    the simulator's behaviour, the severity thresholds or the reserved
    seed protocol; defer M5.
* **The clinical policy is unchanged** (candidate SHA-256
  `83d481b7040b69d78a58de68afc938a588b5b37ba396438509f38c56ca163948`),
  and so is the manifest
  (`250bc7fc835d99140212505e0e79f74f2da1484dbe856fea6a5b93eec9153ef4`).
* **The auditor's witnesses** (kept outside the repository) are
  reproduced as regression tests in
  `internal/eval/remediation_5c4n_test.go` and
  `internal/sim/validation_integrity_test.go`, and the auditor's own
  files were re-read by this build's reader (below).

## Release blockers

| Finding | Status | Where |
|---|---|---|
| **M1** (High): editable summaries certify | **Closed.** The execution constructs, judges, adjudicates and certifies in one process, and writes only afterwards; the certification is decided from how the lines were computed (trusted, family, seeds), never from the lines. A file read back is a non-certifying analysis, whatever it contains. Runtime inputs (candidate, manifest, protocol, policy in force, simulator profile) are verified against digests compiled into the build, and the manifest's candidate pin is checked. A fabricated, complete 3,904-line file on which every mandatory criterion PASSes is `not certified` | `runCertified`, `certify`, `verifyInputs`, `analyzeFile`; `TestFabricatedResultsCannotCertify`, `TestCertifiedInputsPinned`, `TestCertifiedExecutionSmall` |
| **M2** (Medium): complete rows, incomplete measurements | **Closed.** Every judged line's measurements are checked for presence, cardinality, range, finiteness, seed identity and consistency before any criterion reads them; T2G's undetermined materiality, T5a's missing prediction, T8's matched counts (equality) and T4's seed sets (exact) are repaired. The auditor's T2G, T5a, 999,999-matched and −999-seed witnesses no longer pass | `lineProblems`, `reservedCriteria`, `witnessMet`, `predictAcceptance`; `TestMeasurementCompleteness` |
| **M3** (High, documentation): the restriction absent | **Closed.** The protocol opens with the CERTIFICATION SCOPE (trusted in-process only; editable records and imported summaries cannot certify; the accepted swap, undetected; authenticated provenance for future external records; no clinical rule changed for it). §2, §4, §6 and §8 describe the executable behaviour. The 5C.4l documents are marked superseded or corrected | `../calibration/clinical-5c4n/reserved-validation-protocol.md` |
| **M4** (Medium): errors ignored | **Closed.** Malformed settings fail in the execution and the reader; malformed or truncated input fails before writing; an unserializable adjudication is an error and writes nothing; T5 no longer computes statistics from no groups; incomplete coverage fails the exit after writing | `manInvocation`, `writeCriteria`, `analyzeFile`; `TestReaderErrors` |
| **M6** (Medium): the validation helper's records invalid | **Closed.** The helper records the evaluation scenario's fingerprint and refuses any record failing integrity. The access and follow-up evidence, regenerated, is byte-identical and now rests on valid records | `recordBaselineWith`; `TestValidationHelperIntegrity` |
| **M5** (Medium, model): shared history with a larger candidate capacity | **Deferred**, as directed. No manifest scenario has such a candidate; documented in protocol §7 | — |

**Verdict F (historical compatibility, an evidence limitation).** Replay
through dev.22 and dev.23 runs exactly in the full suite at this commit
(`TestDev22RecordReplays`, `TestDev23RecordReplays` and every earlier
fixture); the operational simulator's code is unchanged since 2533a39
(`git diff 2533a39 -- internal/sim` touches only test files).

## Corrections

* **20 distinct stable T1 groups**, not "0 of 28".
* **Near-margin results changed with the shared history** (0.35 → 0.30
  and 0.31 → 0.33 at +2.1 points; 0.56 → 0.65 at +2 points).
* **"Every seed is valid"** and **"no bearing on the confirmatory run"**
  corrected.

Each is marked in place (`../calibration/clinical-5c4l/README.md`,
`clinical-provenance.md`, `slice-5c4k-…-audit-response.md`) and listed in
`../calibration/clinical-5c4n/certification.md` §6.

## Validation

All on this machine, one heavy job at a time, memory-capped
(`docs/MEASUREMENTS.md`, 2026-10-20, Slice 5C.4n).

* **A fabricated file cannot certify.** The auditor's fabricated
  3,904-line file, and a fabricated complete file on which every
  mandatory criterion PASSes, read back by the reader: `non-certifying
  analysis`, `not certified`, a failing exit
  (`TestFabricatedResultsCannotCertify`, under three settings). The
  fabricated file also fails 38,102 measurement checks.
* **The auditor's eight files** (fabricated, duplicated, invalid,
  missing replicate, missing scenario, truncated at a line boundary,
  truncated mid-line, empty), each under three settings: every one a
  failing exit, none certified; `DT_MANIFEST_SEEDS=nonsense` and `-1`
  refused before anything is written.
* **Development executions by the certified path** on PCG(20261021, ·):
  49 scenarios × 40 seeds (exit 0; `development`, `not certified`) and
  the stable and near-margin scenarios × 200 seeds (exit 1 on the 40
  scenarios it does not run, as specified). Every line is byte-identical
  to 5C.4l's; no genuine line has a measurement problem.
* **One defect of this remediation, found and fixed in it.** The first
  40-seed run read T8 INCONCLUSIVE: the new equality expected
  W-ID-CHANGED's first seed unmatched in every priority, where only
  urgent is. The expectation is now exact per witness kind, the test
  fixture states it independently of the adjudicator, a mutation guards
  it, and both runs were repeated.
* **Mutations:** 37, one at a time, re-run in full on the final
  code: 36 caught; the survivor is equivalent (T5a's no-prediction
  guard sits behind the completeness check, which already refuses a
  near-margin line without seed differences; removing both is caught).
* **Full uncached suite**, one package at a time: every package passes
  (2,870 s); vet and gofmt clean; replay through dev.23 exact.

## What this response does not claim

* **No independent review yet.** The focused 5C.4o release-gate audit
  must answer: can fabricated measurements certify; can missing
  measurements pass; are errors fatal; does the validation helper
  produce valid records; does the protocol match the executable
  behaviour.
* **No lock, reserved draw, freeze or activation.** Nothing was run on
  PCG(20261201, ·); no reference or holdout seed or unseal record was
  used.
* **The study swap is not closed**: accepted only in the trusted scope.
