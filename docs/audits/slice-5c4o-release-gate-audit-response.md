# Response to the Slice 5C.4o release-gate audit (maintainer)

This is the maintainer's account of what was done with each finding of
`slice-5c4o-release-gate-audit.md`.

* **The audit is unchanged.** It was committed alone and verbatim
  (36d34f2) before any change, and this document is separate from it.
* **The audit's verdict was NO GO**, on two certification-critical
  findings: O1 (M2) and O2 (M4). M1, M3 and M6 were GO.
* **Under the owner's stopping rule** only those blockers were fixed and
  only their checks rerun; O3 (documentation-only) was corrected
  alongside; O4 needs no remediation. Nothing was added beyond them. The
  version is 0.2.0-dev.25.
* **Unchanged:** the clinical policy (candidate SHA-256
  `83d481b7040b69d78a58de68afc938a588b5b37ba396438509f38c56ca163948`),
  the manifest
  (`250bc7fc835d99140212505e0e79f74f2da1484dbe856fea6a5b93eec9153ef4`),
  the simulator, the thresholds, the trust boundary and the reserved
  seed protocol.

## Findings

| Finding | Status | Where |
|---|---|---|
| **O1** (Certification-critical, M2): partial or contradictory measurements pass | **Fixed.** Each line's Gate B is reconciled with its own measurements by the candidate's rule: PASS or FAIL only with every pair matched, urgent's only with a difference on every pair and at least the rule's 40 seeds; each difference exactly its counts'; each safeguard flag and the deteriorated seeds exactly those above the per-seed limit; the increase their mean; the status the mean's against the margin, exactly. A near-margin replicate needs a difference on every pair, and no pair may inject more than it served or left unimaged. A pair construction error the line does not carry is refused. All five of the auditor's witnesses, applied to an otherwise consistent line, are refused | `gateBReconciles`, `lineProblems`; `TestReleaseGateMeasurements` |
| **O2** (Certification-critical, M4): a truncated second value on a line is ignored | **Fixed.** The reader refuses anything but whitespace after a line's value, before anything is judged | `analyzeFile`; `TestReleaseGateReader` |
| **O3** (Documentation-only) | **Corrected in place.** The "failing exit" claim is limited to the 5C.4m file, and the certification, not the exit, decides; the T5a survivor's equivalence is limited to the pinned manifest's validated inputs; the lock recipe names the two test assertions of the reserved family's refusal that the lock inverts, so the committed tests can all pass at the lock | `slice-5c4m-…-audit-response.md`, `docs/MEASUREMENTS.md`, protocol §8 |
| **O4** (Out of scope; evidence limitation) | **No remediation.** Exact replay runs in the full suite on the committed seed-52 replay fixtures, which are recorded test fixtures, not simulator calibration seeds; the audit's seed restriction (901–910) kept the auditor from running them. The operational simulator's code is unchanged since 2533a39 (`git diff 2533a39 -- internal/sim` touches only test files) | the suite at this commit |

**Why the fixtures changed too.** The criteria tests ran on synthetic
lines (`fakeRun`, `fabricatedRun`) that carried one urgent seed
difference a line for speed and set Gate B statuses directly. Under O1
those lines are inconsistent and were refused, as they should be. They
now carry the rule's own computation (`gateBUrgent`, `failUrgent`,
`flagUrgent`), and identity witnesses read Insufficient in every
unmatched priority, as genuine lines do. No expected verdict changed.

**The auditor's fixtures, rerun against this build** (their files,
unchanged): their four O1 cases and the hidden construction error are
no longer all-PASS; their trailing-truncated file fails the reader at
line 3904 ("data after the line's value"). Their own baseline file is
now refused as well: it was not consistent with Gate B's computation
(for example a near-margin PASS with an increase of 0.5 over seed
differences averaging 0), so the five cases were also rebuilt on a
consistent line in `TestReleaseGateMeasurements`, where each one alone
is refused.

## Validation

All on this machine, one heavy job at a time, memory-capped
(`docs/MEASUREMENTS.md`, 2026-10-20, release-gate fixes).

* **The auditor's fixtures** (their exact files, SHA-256 `197b6e71…6d1`
  for the all-PASS baseline): every O1 case and the hidden construction
  error no longer all-PASS; the trailing-truncated file fails the reader.
* **Each witness alone** on an otherwise consistent line, and each of
  the reconciliation's relations isolated so that no other check
  catches it (`TestReleaseGateMeasurements`); four trailing-data forms
  refused and trailing whitespace accepted (`TestReleaseGateReader`).
* **Genuine lines are not refused.** The 664 lines of the two 5C.4n
  development executions, re-read by this build: no measurement
  problem; the 40-seed pass exits 0 with every criterion as written by
  the execution; the 200-seed pass reports only its 40 scenarios not
  run, every verdict unchanged. `TestReservedManifestConstruction`
  (every scenario on the debug family) and `TestCertifiedExecutionSmall`
  pass.
* **Mutations:** 49, one at a time (the 5C.4n set and 13 for the fixes).
  The full run caught 44; the five touched by the final merge of the
  empty and partial difference checks were rerun after it, and four are
  caught. The survivor is the T5a guard alone, equivalent for the pinned
  manifest's validated inputs (O3); removing it together with the
  difference checks is caught.
* **Full uncached suite**, one package at a time: every package passes
  (3,431 s); vet and gofmt clean; replay through dev.24 exact.

## Observed, not changed

* **T5a's predicted mean is summed in map order.** Re-reading the
  200-seed lines gave NM2.1's prediction as 0.34404 where the execution
  wrote 0.34403999999999996: the group predictions are averaged in Go's
  map iteration order. No verdict can change (T5a's tolerance is four
  cluster standard errors plus 0.02), but the adjudication file's bytes
  are not reproducible to the last digit. Left as found under the
  stopping rule; the lock or the run report should record it.
  * **Fixed before the lock (0.2.0-dev.26), at the owner's direction:**
    the predictions are summed in group order. The value is now 0.34404
    on every adjudication; no formula, threshold or verdict changed
    (`TestAdjudicationDeterministic`).

## What this response does not claim

* **No further broad review.** The stopping rule applies: the fixed
  blockers' focused reproductions are rerun here; the owner decides
  whether they close the gate.
* **No lock, reserved draw, freeze or activation.** Nothing was run on
  PCG(20261201, ·); no reference or holdout seed or unseal record was
  used.
* **The study swap is not closed**: accepted only in the trusted scope.
