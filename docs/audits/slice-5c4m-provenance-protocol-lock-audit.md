# Slice 5C.4m provenance and protocol lock readiness audit

**Audited revision:** commit 2533a39 (0.2.0-dev.23), the 5C.4l remediation: order provenance, inherited-order safety, the zero-evidence adjudicator, reporting corrections and the proposed reserved protocol (not locked, not frozen, not in force), under the owner's scope decision of 2026-10-20 on the study swap (accepted only for trusted in-process evaluation). Audit date 2026-10-20. Line references are to 2533a39.

The audit was independent and read-only. The auditor read a `git archive` snapshot of 2533a39 on a separate machine, in a dedicated directory outside the repository; disposable test fixtures, overlays and logs lived in a work copy and a scratch directory beside the snapshot, and the snapshot (838 files) was verified byte-identical before and after. A first attempt stopped early without a report and was discarded; the assignment was restated in neutral wording with the same scope and run again from a fresh work copy. It ran no strategy, reference seed, holdout seed, reserved stream PCG(20261201, ·), unseal, activation or freeze. The paths below point into that directory, which was removed after the audit; the fixture source and logs are kept by the maintainer outside the repository for remediation.

* **Everything below the rule is the auditor's report as delivered, unchanged.**
* **No remediation is in this commit.** Maintainer checks, if any, will follow in a separate document.

---

**DreamTeam Slice 5C.4m — independent audit**

Audited snapshot: **2533a39**, version **0.2.0-dev.23**. Audit date: **2026-10-20**. References below identify files and lines in that snapshot, not the disposable work copy.

**FINAL DECISION: NO GO for a prospective protocol-lock commit under the approved trusted-in-process restriction.**

The immutable-order reconciliation and removal of the inherited-order exemption work. The accepted study swap remains detectable neither by internal consistency nor by paired immutable requests. The ordinary record-generation path does not import a study-order mapping, but the final certification path reads unauthenticated, editable measurement summaries. An independently fabricated 3,904-line file, containing no clinical records and produced without a simulation, received **all eleven mandatory PASS statuses and `invocation: confirmatory` from the actual adjudication command**. This defeats the intended containment of the accepted limitation.

The principal blockers are M1–M4: result-origin enforcement, measurement completeness, the prominent certification restriction, and adjudication error handling. M6 also requires correction of the simulator-validation evidence and its documentation. M5 is a confirmed model defect outside the current manifest, not a reason to alter the clinical policy.

| Requested verdict | Decision and scope |
|---|---|
| A. Order provenance | **GO**, for reconciliation of immutable requests against a trusted paired baseline. This is not authentication of studies or of externally supplied records. |
| B. Inherited-order safety | **GO**, for the current contract, clinical attribution and correctly identified in-process records. The inherited-study deletion and request-mutation bypasses are rejected. |
| C. Record origin and data flow | **NO GO**. The final reader accepts substituted summaries as confirmatory evidence: M1. |
| D. Reserved adjudicator completeness | **NO GO**. Group coverage is repaired, but required measurement inputs can be absent, impossible evidence can pass, and incomplete inputs can produce an invalid output file: M2, M4. |
| E. Protocol and reporting consistency | **NO GO**. The binding certification restriction is missing; some validation claims exceed the evidence: M3, M6. Most K5/K6 reporting repairs are confirmed. |
| F. Historical compatibility | **NO GO for the complete assurance claim**. The checked v4 and historical-candidate regressions pass. Exact replay through dev.22 and unchanged operational source relative to the parent revision were not independently established; this is an evidence limitation, not a demonstrated regression. |
| G. Readiness for prospective protocol lock | **NO GO**. Repair and review the blockers before enabling the reserved family. |
| H. Readiness for reserved calibration | **NO GO**. No reserved execution was performed or authorized by this audit. |

**Scope and preservation**

The initial `sha256sum -c --quiet ../snapshot.sha256`, executed inside `snapshot/`, succeeded for all **838 files**. The initial inventory included hidden files. The final execution of the same checksum command also succeeded for all **838 files**. The final inventory exactly matches both the initial inventory and the digest list: **no file changed, no file was added, and no file was removed**. Logs: `scratch/snapshot-start.log`, `scratch/snapshot-end.log` and `scratch/snapshot-end-inventory.log`.

All fixture sources and logs are under `scratch/`; additional test files were copied only into `work/`. One work-copy manifest digest was deliberately corrupted for a reader test and restored byte for byte. No production repair was made. No commit, tag, protocol lock, freeze, activation or unseal was performed. No reference, holdout or reserved simulation was run. Simulator executions used **901–910 only**, with no plan actions. Fidelity record generation used the **debug family PCG(1, ·)**. Synthetic evaluator records and fabricated result rows are unit-test data, not simulation runs. The adjudicator's fixed internal Monte Carlo predictor was exercised; it does not draw the reserved family.

Historical replay fixtures use simulator seeds outside 901–910, including seed 52 for dev.22. Those tests were not executed under the assignment's seed restriction. The archive contains no parent source tree with which to establish a complete operational-code diff. Historical non-execution of protected streams also cannot be proved from an archive. These limitations are not treated as verified maintainer claims.

**M1 — Editable result summaries can obtain a favorable confirmatory verdict. High. Confirmed implementation defect; requires an implementation and execution-protocol repair. Blocks lock.**

The producer writes `groupResult`, not `Record`: `internal/eval/reserved_manifest_test.go:486` defines the serialized object; `:695` constructs groups, calls the clinical decision, writes JSONL and drops the records. The separate reader at `:1571` opens the arbitrary path in `DT_MANIFEST_IN`, unmarshals those objects and calls `reservedCriteria`.

There is no authenticated binding to the producing process, run, family, binary, candidate policy, manifest or model profile. The rows contain no family identifier or authenticated study-order history. Their `valid`, `rejected`, gate statuses, harm labels, violation lists, provenance failures and measurement counts are assertions supplied by the file. `invocation: confirmatory` is determined solely by equality of the reader's requested seed counts with the runtime manifest's seed counts (`:1385`). It does not establish that a reserved run occurred.

Independent reproduction:

* `scratch/witnesses/make_results.py` constructs all **3,904 rows** directly from the manifest's layout, without constructing a `Record`, drawing a simulation stream or calling the clinical evaluator. It supplies the flags expected by the witness scenarios and 250 claimed harmed-and-detected seeds.
* `TestAuditMResultOrigin` passes the rows to the adjudicator. Every mandatory criterion passes.
* `scratch/witnesses/run_reader.py` feeds the same file to the unmodified `TestReservedCriteria` entry point. `scratch/fabricated.criteria.json` contains PASS for T1, T2, T2-conformance, T2G, T3, T4, T5a, T5b, T6, T7 and T8, and `confirmatory` for the invocation. See `scratch/logs/fabricated.reader.log` and `reader-matrix.log`.
* Replacing the work manifest's `policy.sha256` with 64 zeroes still produces the same favorable statuses. The reader never validates that digest against the candidate. The manifest was restored afterwards. This succeeds in the reviewed build, whose generator cannot run the reserved family.

This is a substituted-measurement attack, not a claim that JSONL contains editable clinical records. Editing JSONL cannot retroactively exchange studies inside a discarded record; it can replace the decisions and observations that final confirmation relies on. A full development-family result would likewise have no family information by which this reader could distinguish it from a reserved result.

Protocol preflight checks and the instruction to commit output files unchanged are useful procedural controls. They do not authenticate the bytes between production and judgment. A digest computed after an edit authenticates neither the original producer nor the original study-order association. The current protocol's clean-tree check occurs before generation, while its output digests are committed afterwards (`docs/calibration/clinical-5c4l/reserved-validation-protocol.md:509`, `:561`).

Required repair: preserve the trusted boundary through final criteria evaluation. For example, calculate certification from the producer's in-memory measurements without reopening a mutable result file, and serialize the measurements and verdict afterwards. A separate reader may certify only with independently trusted production provenance binding the exact measurements and pinned inputs to the authorized execution. Otherwise its output must be explicitly non-certifying. General external-record evaluation still requires an authenticated order-to-study mapping or equivalent trusted event history; agreement among editable fields is insufficient.

**M2 — Complete row coverage does not establish complete or valid measurements. Medium. Confirmed implementation defect; requires validation of adjudicator inputs. Blocks the completeness claim and lock.**

The K4 repair checks the presence and identity of groups, replicates and pair rows (`reserved_manifest_test.go:941`). It does not validate all observations needed to interpret them.

The fabricated full-layout file demonstrates two zero-measurement PASS paths:

1. **T2G:** every deterioration group's `severeAcquired`, `worstSoonPastDueMin` and `increase` inputs are absent. The code leaves severe shares absent, initializes the median and increase to zero, calls the scenario non-material and reports PASS (`:1249`, `:1281`). There were no observations with which to establish non-materiality.
2. **T5a:** every near-margin group's `urgentSeeds` array is absent. For injected scenarios, `predictAcceptance` divides by the length of an empty seed list; the resulting NaN comparison never succeeds, giving a prediction of zero (`:1502`). Claimed zero acceptance then conforms and passes (`:1303`). The un-injected NM0 prediction follows its supplied gate flag. T5b also accepts the supplied zero rates.

Independent `TestAuditMSummaryShape` additionally changes every matched-seed count to **999,999**, and each W-HORIZON violation list to **40 copies of nonexistent seed −999**. **T4 and T8 still PASS.** T8 checks `matched < wanted`, not equality (`:1365`); several T4 witnesses check only list length, not that every actual pair seed appears exactly once (`:1436`). This contradicts the protocol's exact-match and targeted-seed requirements (`reserved-validation-protocol.md:348`, and its T4 table).

These defects remain distinct from M1: authenticating a result does not make missing or malformed measurements scientifically sufficient. Required inputs need explicit presence, cardinality, domain, uniqueness and cross-field checks. Missing materiality or prediction evidence must be INCONCLUSIVE or rejected as invalid input. Preserve the existing clinical cohort, severity thresholds and attribution rules.

**M3 — The protocol does not prominently impose the approved certification restriction. High documentation defect. Requires documentation correction. Blocks lock.**

The protocol's opening still asks whether the owner accepts the study-swap limitation (`reserved-validation-protocol.md:13`). Its detailed discussion appears near the end of §7 (`:476`, `:490`), says the issue does not bear on confirmation, and describes file-based trust through committed record digests. `clinical-provenance.md:135` makes the same categorical claim and retains clinical-policy alternatives that the binding scope decision has already excluded.

The required prominent rule is absent: **certified favorable v5 results are restricted to trusted in-process records; external, imported, edited or otherwise untrusted records cannot receive certification without independent study-order provenance.** The current wording neither states that prohibition clearly nor addresses the mutable summary-file boundary demonstrated in M1.

Update the opening, certification conditions and execution section to record the owner's actual scoped approval, the remaining inability to detect the swap, and the required provenance for any future external-record use. Retain the swap regression as a demonstration of the limitation. Remove the unresolved-choice framing. No new decision about clinical cohorts, thresholds or attribution is needed.

**M4 — The reader silently ignores invocation and serialization errors. Medium. Confirmed implementation defect; requires implementation and regression tests. Blocks reliable incomplete-run adjudication.**

An empty JSONL file reaches `reservedCriteria`, whose criterion statuses are INCONCLUSIVE. However, an empty T5 group list produces NaN through `meanSD(nil)` (`internal/eval/bounds.go:87`, `reserved_manifest_test.go:1333`). The reader discards the `json.MarshalIndent` error and writes **one newline**, then exits successfully (`:1616`). `scratch/zero.criteria.json` is not JSON; `scratch/logs/zero.reader.log` says the test passed. Removing an entire near-margin scenario exercises the same missing-group serialization problem.

This is **not** a claim that the internal criterion map returns PASS on a completely empty run. The repaired map does not. The defect is that the actual file-based execution fails to emit its required adjudication and does not report that failure.

Also, `DT_MANIFEST_SEEDS=nonsense` and `DT_MANIFEST_SEEDS=-1` silently select the manifest's full seed counts (`:1577`, `:1607`). Both produce all PASS and `confirmatory` on the fabricated file. The generator rejects invalid/nonpositive seed settings (`:783`); the reader does not.

Return an explicit error for malformed invocation settings; represent undefined statistics without NaN; check serialization errors. Add entry-point tests that parse the emitted criteria JSON, including empty and missing-scenario inputs. A successful test-process exit must never be mistaken for protocol confirmation.

**M5 — Shared history is not invariant to a candidate's larger imaging capacity. Medium. Confirmed fidelity-model defect outside the pinned manifest; requires implementation before broader model use. Not a blocker for the current 49 scenarios by itself.**

`recordTakeover` uses the baseline's booking stream and booking limits before takeover. However, it constructs slot timestamps using `max(candidate capacity, baseline capacity)` (`internal/eval/oc_fidelity_test.go:527`, especially `:532` and `:544`). Increasing candidate capacity changes the slot grid used for pre-takeover bookings, even though their stream and nominal limit remain the baseline's.

`TestAuditMFidelityOrigin` uses debug PCG(1, ·), A85 as baseline, and a candidate with imaging-capacity factor 2 from the warm-up's start. Despite requesting shared history, inherited orders **o9606 and o9626** are missing from the candidate; retained pre-takeover studies differ, **1 versus 3**. This is a history-consistency fixture, not a strategy-performance evaluation.

The sampled manifest pairs reconcile, and the maintainer's shared-history regression passes. Current manifest practices do not increase imaging capacity above their paired baseline (`oc_fidelity_test.go:1102`; `reserved_manifest_test.go:125`), so this witness does not demonstrate corruption of the current proposed run. The existing half-capacity regression cannot expose this direction of the error. Preserve the baseline's actual pre-takeover history when expanding the model's supported practices.

**M6 — The simulator validation helper produces records that fail the candidate's scenario-integrity check. Medium. Confirmed validation-harness and documentation defect; requires implementation and evidence correction.**

`recordBaselineWith` passes an empty scenario fingerprint to `NewEvalRecorder` (`internal/sim/clinical_profile_test.go:85`). The candidate requires fingerprint `e8718dbd1031feb326b4934d291f81eb14fd867d4a8fa6acc1673a75b55b76f8` (`docs/calibration/clinical-5c4b/candidate-policy-v5.json:15`). `Measure` correctly records an integrity failure for the mismatch (`internal/eval/measure.go:361`).

Independent `TestAuditMBaselineHarnessValidity` calls the **unmodified** helper on seed 901 and reproduces `scenario  is not the policy's`. This helper is used by the simulator-access and follow-up-finance validation paths. Those paths extract measurements without requiring empty integrity failures (`internal/sim/clinical_access_test.go:174`, `:226`; `followup_finance_test.go:51`). Thus the statement “every seed is valid” in `clinical-provenance.md:264` is not established by those validation records and is false for the reproduced helper output.

This is a metadata/integrity failure, not evidence that the simulator loses inherited obligations. The independent recorder fixture supplies the required scenario identity at recording time and checks its records against live simulator state. The financial numbers reproduce, but numerical agreement does not turn an invalid record into a valid one.

Correct the validation harness's scenario identity, require integrity success before publishing validity claims, and regenerate or qualify the affected evidence. The reserved fidelity harness uses its own matching fixture fingerprint, so this defect does not itself demonstrate a reserved-record mismatch. The archive does not establish when this helper defect was introduced.

**Complete execution and trust trace**

The reserved protocol uses the fidelity model, not saved simulator sessions.

| Stage | Source and observed behavior |
|---|---|
| Inputs | `loadManifest` reads a runtime JSON file (`reserved_manifest_test.go:106`). `manifestPolicy` obtains the fixture policy plus the candidate's clinical section (`:759`; `clinical_witness_test.go:52`). `simProfile` reads runtime `testdata/sim-profile-5c4b.json` (`oc_fidelity_test.go:54`). Separate tests pin some inputs; the execution entry points do not authenticate all these reads. |
| Demand and pairing | `manPair` checks equal demand models and supplies both plans with PCG(F, 2i), using separate booking-role streams and shared pre-takeover history (`reserved_manifest_test.go:166`). The family comes from an environment variable; this build rejects the reserved family in `familyAllowed` (`:654`). |
| Orders and studies | `recordTakeover` creates IDs/origins from creation indexes, books orders, records served resolutions and retains inherited open work (`oc_fidelity_test.go:624`, `:659`, `:675`). Its acquisition tuples carry the creating order ID into each study (`:687`). No serialized clinical record or externally supplied order-to-study map is read here. |
| Deliberate scenario operations | `manApply` implements the manifest's fixed injections and refreshes outcomes/samples (`reserved_manifest_test.go:251`). These intentionally construct harmful or invalid witness scenarios. There is no study-swap injection or arbitrary imported mapping operation. A mutable manifest can still change which constructions execute. |
| Clinical judgment | `protocolDecision` calls `planReport`, provenance reconciliation, Gate B, safety and the allowance in memory (`:537`). It excludes finance as specified. Near-margin replicates use `gateBDecision` (`:665`), with the base scenario testing the other rules. |
| Serialization | `manRunScenario` writes one summary per group/replicate or remainder (`:695`). Rows carry scenario/group/replicate identifiers, seed ranges, validity/rejection flags, gate summaries, seed lists, materiality inputs, access claims and `pairInfo` construction counts. They contain neither full orders/studies nor authenticated origin evidence. |
| Final judgment | `TestReservedCriteria` reads JSONL in a second invocation (`:1571`). It recomputes coverage, rates, counts, intervals, witness predicates and the T5 prediction **from supplied summaries**. It does not regenerate records, remeasure clinical safety, reconcile study links, or verify the producing family. It writes a separate criteria JSON file. |

Consequently, a direct outcome-only swap is not exposed by the fixed in-process record-generation interface, but end-to-end trusted certification is **not contained**: the editable measurement handoff can replace the clinical evidence altogether. Runtime manifests, profiles and policy bytes also need binding to the trusted execution; a same-binary assertion alone does not bind runtime files.

**Independent record-consistency and inherited-work results**

`zz_audit_m_records_test.go` defines its own 40 inherited soon-CT orders and two same-request orders on opposite sides of the horizon. Existing synthetic population/finance scaffolding supplies the surrounding complete records. The independently specified mutations are evaluated on complete 40-seed comparisons. All 36 subtests pass; the old-candidate relink control also passes its expected GO assertion.

| Case | Observed result |
|---|---|
| Clean records; legitimate loss of the later order | GO |
| By-horizon order imaged afterwards, reported 60 hours past due | NO GO |
| K1 relink-and-retime, original order lost | INVALID under current candidate; GO under 5C.4j |
| Outcome-only exchange of the two studies and resolutions | GO; the unswapped harmful control is NO GO |
| Same-request IDs exchanged together with study links | INVALID |
| Missing order; candidate-only order; repeated origin; repeated ID | INVALID |
| Post-horizon study reassigned without a consistent exchange | INVALID |
| Inherited order or its study deleted; duplicate inherited study | INVALID |
| Inherited reporting violation, before or after the horizon; still-pending inherited report | NO GO |
| Inherited order and study jointly relabelled; inherited wanted time moved | INVALID |
| ID, origin, receipt, wanted time, priority, modality or walk-in flag altered on inherited or later work | INVALID |

The implementation indexes **every order**, not merely Gate A/B-eligible demand, and checks immutable requests in both directions (`provenance.go:48`). `compare.go:38` applies this to comparisons. `checkOrderStudies` requires unique order IDs and one matching post-takeover study for every served inherited or new order (`clinical.go:1099`). Studies acquired at or before takeover retain their legitimate exemption. Neither that exemption nor the new pairing is an authenticated history of pre-takeover studies.

The accepted swap and consistent edits to outcomes or both records remain limitations of a pure record evaluator. Preserve the existing `TestStudySwapLimitation` regression (`remediation_5c4l_test.go:197`). Nothing in this audit recommends changing safety cohorts or attribution to hide that limitation.

The independent simulator recorder check compares its selection with live takeover state and queued rebookings, preserves original immutable requests, checks unique IDs/origins, checks each served order has exactly one study, and compares every post-takeover study's link and acquisition time against live state. All ten seeds pass with correctly supplied scenario identity: 1,171–1,264 inherited orders per seed, 14 queued no-show rebookings in total, no missing or duplicated expected order, and no discrepancy in the checked study associations. Every resulting record has an empty integrity-failure list.

The source explains why rebooking preserves wanted time: `noShowRebook` re-offers the existing order without overwriting its original earliest day (`internal/sim/flow.go:296`); the recorder derives wanted time from that original request (`export_eval_test.go:416`). Exam completion marks the order imaged and creates its associated study in the same transaction (`ops.go:370`, `reading.go:76`).

Gate A and Gate B continue to exclude inherited orders from new-demand denominators (`clinical.go:242`; `lostdemand.go:182`). The independent denominator fixture finds **1,201** eligible new soon orders in each gate and reports **40 inherited orders separately**; removing those inherited orders does not change the new-demand denominator. Inherited reporting obligations still enter safety. The post-horizon absolute loop uses the linked order's wanted time (`clinical.go:325`), which is precisely why an unauthenticated outcome swap remains significant.

**Adjudication, reporting and statistical scope**

The manifest has 49 scenarios. Complete-group rows are:

`8 × 62 + 36 × 25 + 5 × 25 × 20 = 3,896`.

Adding eight remainder rows gives **3,904**. There are **61,000 scenario seed assignments**, representing **56,000 distinct base pairs** because five near-margin scenarios reuse S-A85's first 1,000 pairs. The protocol's count is consistent with that reuse.

`TestAuditMCompleteness` exercises **258** scenario/group/duplicate/invalid/pair/replicate/remainder mutations across all applicable scenarios. Every one prevents confirmation. Empty input and an invocation declaring all scenarios absent produce no internal criterion PASS. Missing scenarios cannot be excused by filtering them out. Reduced counts and ordinary filters mark the invocation `development` and make coverage incomplete. The real reader rejects a truncated JSON line; truncation at a complete-line boundary makes affected criteria and T8 INCONCLUSIVE. Missing NM2.1 replicates, duplicate rows and invalid-record flags likewise prevent confirmation. These are substantive K4 repairs. M2 and M4 describe the remaining gaps.

The exact §8 command was inspected only. Its generator unsets the seed/scenario filters, sets the reserved family and output path, and runs only `TestReservedManifestRun`. Its reader unsets those filters, accepts the JSONL path and runs only `TestReservedCriteria`. The latter receives no authenticated family information. The source's default generator family is development; its default count covers each manifest scenario. The build command must be issued from the repository root before changing to `internal/eval`, despite the paragraph's ambiguous “from internal/eval” wording. This build still refuses the reserved family. No part of the pinned reserved invocation was executed.

Confirmed reporting repairs:

* T1 emits stable rejection counts, stress-practice reports and partial-ceiling seed lists. T2 emits per-scenario detection/lower bounds, the conditional allowance prediction, and H10/OH10 reports. T7 reads raw insufficiency rather than inferring it from an aggregate verdict (`reserved_manifest_test.go:1089`, `:1178`). Independent fixtures confirm remainder partial-hit disclosure and T7 failure at four raw-insufficient groups out of 62.
* W-ACCESS65 now requires favorable Gate B, both disclosed urgent loss fractions and the restricted claims (`:1458`). Removing those observations, lowering one fraction to 0.64, or making the gate fail causes the witness to fail. It is no longer merely a constants check.
* The corrected leading partial-window results are reproduced on simulator seeds **901: 3/42**, **902: 3/41**, **903: 4/41**. These are local FAILs. The committed 5C.4j and 5C.4l simulator-access JSON objects are identical; aggregate clinical failure already existed on those seeds. Local changes must remain distinct from whole-seed/evaluation decisions.
* The candidate specifies a **45-day follow-up** (`candidate-policy-v5.json:405`). The financial comparison reproduced `followup-finance.json` exactly: minimum cash, maximum captured past-due payable, funding gap and derived solvency/funding classifications are unchanged on 901–910; end cash increases. Observation extends the posting-level cash-minimum interval and moves the end funding/liquidity capture (`measure.go:385`, `:401`). M6 qualifies the original helper's record-validity claims. These no-action observations do not establish the effect on other practices or actions.

Shared history changed measured fidelity results. Published +2.1-point acceptance changes from **0.35 to 0.30** in construction and **0.31 to 0.33** in the 200-seed pass; +2-point acceptance in that latter pass changes from **0.56 to 0.65**. The old artifacts remain available. Clinical thresholds did not change, but the measurement inputs did.

The “0 of 28 stable groups” wording needs careful interpretation: the two artifacts contain **4 + 20 T1 group evaluations and 4 stress evaluations**. Each first-pass T1 group is repeated identically as group zero of the second pass. There are **20 distinct T1 groups**, not 28 independent T1 observations. `scratch/logs/development-counts.log` records the equality checks. Even those distinct groups share demand across scenarios. This evidence neither establishes the prospective false-rejection target nor substitutes for the specified confirmation. The small development runs, dependence, finite follow-up and empirical nature of Gate B remain statistical limitations, not grounds for tuning the approved policy.

**Policy bytes and historical compatibility**

Comparison with `internal/eval/testdata/candidate-policy-v5-5c4j.json` finds exactly three JSON differences: status text, `orderStudyIntegrity: contractInherited`, and the added `orderProvenance: pairedOrigin`. No severity threshold, cohort, window, allowance, follow-up or financial rule changed.

| Artifact | Audited SHA-256 |
|---|---|
| Candidate `clinical-5c4b/candidate-policy-v5.json` | `83d481b7040b69d78a58de68afc938a588b5b37ba396438509f38c56ca163948` |
| `clinical-5c4l/scenario-manifest.json` | `250bc7fc835d99140212505e0e79f74f2da1484dbe856fea6a5b93eec9153ef4` |
| `clinical-5c4l/reserved-validation-protocol.md` | `f14ff2725bd52effb49b3d11f41f9ffb085f6a6207b4c0f936f4a6036789a927` |
| `clinical-5c4l/clinical-provenance.md` | `817412551b1b8d0ff3fcc33c62d1f0e1dfca59d9b7e5c81338c8b45a3a071325` |
| In-force `docs/evaluation/policy-v0.2-eval-4.json` | `4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3` |
| Historical 5C.4j candidate | `9efa6e63f38de9cfbfdb829ac1a82eff114d6e5c523d32df5d46f53b8c429e7c` |

The candidate digest matches the assignment and manifest. Additional historical-candidate and evidence digests are in `scratch/logs/artifacts.log`.

`TestV4Neutrality` reproduces the pinned evaluation digest `eef98fab6cd6355eb854b86675f914ecc8be02ba6cfcd5bd83f607dd98f0b070`. Frozen-policy, historical-reference, old d/e/f-candidate, h-boundary and j-control checks pass. Dev.22 is registered in the compatibility map (`internal/session/record_v1.go:215`), and its replay tests exist (`dev22_test.go:23`); their exact-execution claim remains unverified here for the seed restriction stated above. No change to operational simulator behavior is demonstrated by this review, but a parent-tree comparison and exact replay cannot be replaced by that absence of evidence.

**Evidence and reproduction**

Run commands below from `work/`. `../scratch/run-go.sh` sets the specified Go path, scratch temporary/cache directories, `GOFLAGS=-mod=mod`, `GOMEMLIMIT=5GiB`, offline module access and `nice -n 10`. Heavy jobs were sequential, with `-p 1`, explicit timeouts and output redirected to scratch logs. The full eval/sim/session suites and full development manifests were not run.

Independent fixture sources:

* `scratch/witnesses/zz_audit_m_records_test.go`: complete comparison controls, K1, inherited work and request mutations.
* `zz_audit_m_results_test.go`: fabricated-result acceptance and exhaustive structural coverage mutations.
* `zz_audit_m_reporting_test.go`: denominators, reporting, W-ACCESS65 and malformed summary evidence.
* `zz_audit_m_fidelity_test.go`: debug-family pairing and the larger-capacity history defect.
* `zz_audit_m_hook_test.go`, `zz_audit_m_sim_test.go`: read-only state observations, recorder validation, partial windows and the unmodified baseline-helper integrity failure.
* `make_results.py`, `run_reader.py`, `check_artifacts.py`: reproducible fabricated inputs, real reader invocations and artifact comparisons.

```sh
cp ../scratch/witnesses/zz_audit_m_records_test.go internal/eval/
cp ../scratch/witnesses/zz_audit_m_results_test.go internal/eval/
cp ../scratch/witnesses/zz_audit_m_reporting_test.go internal/eval/
cp ../scratch/witnesses/zz_audit_m_fidelity_test.go internal/eval/
cp ../scratch/witnesses/zz_audit_m_hook_test.go internal/sim/
cp ../scratch/witnesses/zz_audit_m_sim_test.go internal/sim/
python3 ../scratch/witnesses/make_results.py
../scratch/run-go.sh test -p 1 ./internal/eval -count=1 -timeout=20m \
  -run '^TestAuditM' -v > ../scratch/logs/reproduce-eval.log 2>&1
../scratch/run-go.sh test -p 1 -c -o ../scratch/eval.test ./internal/eval
python3 ../scratch/witnesses/run_reader.py > ../scratch/logs/reproduce-reader.log 2>&1
../scratch/run-go.sh test -p 1 ./internal/sim -count=1 -timeout=15m \
  -run '^TestAuditM' -v > ../scratch/logs/reproduce-sim.log 2>&1
DT_FIN_FOLLOWUP=/home/george/dt-audit-5c4m/scratch/followup-finance-reproduced.json \
  ../scratch/run-go.sh test -p 1 ./internal/sim -count=1 -timeout=15m \
  -run '^TestFollowUpFinancialObservation$' -v \
  > ../scratch/logs/reproduce-finance.log 2>&1
python3 ../scratch/witnesses/check_artifacts.py > ../scratch/logs/reproduce-artifacts.log
```

These commands invoke no manifest generator and no reserved stream. The fabricated file is a test attack, not calibration evidence. Several tests deliberately assert that a defect is reproducible; their PASS status does not mean the product is correct.

Principal executed logs: `eval-targeted.log`, `independent-records.log`, `completeness-fidelity.log`, `inherited-followup.log`, `reader-matrix.log`, `simulator-finance.log`, `simulator-recorder.log`, `finance-comparison.log`, `artifacts.log`, and `development-counts.log`, all under `scratch/logs/`. The initial simulator fixture's missing fingerprint failure is retained; the corrected independent fixture supplies scenario metadata, while a separate test reproduces the defect in the unchanged repository helper. Fixture digests are in `scratch/fixtures.sha256`; `scratch/logs/work-diff.log` confirms that all original work-copy files were restored and only the six audit test files remain added. Snapshot preservation checks accompany the report.

**Required disposition**

Repair M1 and M2 before any certifying execution; repair M4 so incomplete and malformed executions produce an explicit, usable result or a reported error. Correct M3 to enact the existing owner decision prominently. Correct M6's harness and validity claims. Keep M5 scoped to the current manifest until its broader-history defect is repaired. Do not treat the accepted swap as universally fixed, and do not change clinical cohorts, thresholds or attribution to address it.

The current protocol permits only enabling the reserved family at lock (`reserved-validation-protocol.md:521`). That is insufficient: the necessary implementation and documentation changes must be reviewed first, then their resulting artifacts, runtime-input digests and trusted execution path pinned. The digests above identify what was audited; they are **not approval to lock or run these bytes**. No additional clinical-policy owner decision is required to close the confirmed blockers.
