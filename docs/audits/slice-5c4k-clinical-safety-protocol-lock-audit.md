# Slice 5C.4k clinical safety and protocol lock audit

**Audited revision:** commit 04e97e5 (0.2.0-dev.22), the 5C.4j remediation: the revised candidate v0.2-eval-5 clinical policy, the order–study contract, the 45-day follow-up, the simulator evaluation recorder and the proposed reserved protocol and adjudicator (not locked, not frozen, not in force), 2026-10-19 to 2026-10-20. Line references are to 04e97e5.

The audit was independent, adversarial and read-only. The auditor read a `git archive` snapshot of 04e97e5 in a dedicated directory outside the repository; disposable witnesses, overlays and logs lived in a work copy and a scratch directory beside the snapshot, and the snapshot was verified byte-identical before and after. It ran no strategy, reference seed, holdout seed, reserved stream PCG(20261201, ·), unseal, activation or freeze. The paths below point into that directory, which was removed after the audit; the witness source and logs are kept by the maintainer outside the repository for remediation.

* **Everything below the rule is the auditor's report as delivered, unchanged.**
* **No remediation is in this commit.** Maintainer checks, if any, will follow in a separate document.

---

# Slice 5C.4k: independent clinical safety and protocol lock audit

Audited snapshot: 04e97e5, version 0.2.0-dev.22

Audit dates: 2026-10-19 to 2026-10-20. The auditor read a read-only `git archive` snapshot of 04e97e5 (`/home/george/dt-audit-5c4k/snapshot/`; its 822-file digest list `snapshot.sha256` was verified before the audit and again after it: unchanged, no file added), ran targeted existing tests and independently constructed witnesses in a disposable copy (`/home/george/dt-audit-5c4k/work/`), and kept every log and witness source under `/home/george/dt-audit-5c4k/scratch/`. Line references are to the snapshot (commit 04e97e5). Reproduction commands are relative to the work copy. Every heavy run was memory-capped (`GOMEMLIMIT=5GiB`, `systemd-run --scope -p MemoryMax=6G`, `nice`) and run one at a time (`scratch/witnesses/run-go.sh`).

Nothing reserved was touched: no strategy optimization, no reference seed 1001–1020, no holdout seed, no stream of PCG(20261201, ·), no reserved calibration, no unseal, no activation, no freeze, no tag. Simulator runs used development seeds 901–910 only; fidelity-model runs used the debug family PCG(1, ·) only.

## Final decision

**NO GO for a prospective protocol-lock commit.**

The 5C.4j remediation is substantially correct: the observed 5% ceiling on partial windows, the 45-day follow-up and its boundary conventions, the simulator recorder repair, the prefix-stable fidelity draws, the remainder and coverage repairs of the adjudicator, the per-priority T2G materiality and the targeted T4 attribution all behave as written, and every historical compatibility check passes. Two confirmed evaluator defects remain in the one area the 5C.4i audit flagged as blocking (order–study integrity), and the adjudicator can report a criterion as PASS with nothing judged. These must be repaired before the evaluator and protocol are pinned.

| Verdict | Result |
|---|---|
| A. Partial-window safety | **GO** (conservative by design; one documentation defect, K5) |
| B. Order–study integrity | **NO GO** (two confirmed bypasses, K1 and K2) |
| C. Simulator recorder completeness | **GO** |
| D. Horizon and follow-up safety | **GO** (one documentation gap, K6) |
| E. RNG prefix stability | **GO** |
| F. Reserved adjudicator | **GO with one required repair** (K4) |
| G. Protocol consistency | **GO with documentation corrections** (K5, K6) |
| H. Historical compatibility | **GO** |
| I. Readiness for prospective lock | **NO GO** |
| J. Readiness for reserved calibration | **NO GO** (follows from I; the owner has in any case reserved this decision) |

Blockers, each with the kind of decision it needs:

1. **K1 (High; implementation repair).** A post-horizon absolute violation can be removed from the follow-up cohort by relinking its study to another order of the same request that is re-timed consistently, at the price of one lost order. The record stays VALID and the verdict is GO.
2. **K2 (High; implementation repair, plus an owner decision on pairing inherited work).** An inherited order imaged after the takeover can lose its study, be relabelled to a longer limit, or have its wanted time moved past the horizon; each is VALID and GO. The contract's second exemption is never needed by the simulator or the fidelity model.
3. **K4 (Medium; implementation repair and an execution restriction).** The adjudicator judges coverage against the seeds and scenarios its invocation declares, and a criterion with no judged group reports PASS. The maintainer's own `construction-checks.json` shows T4 and T2G "pass" with `groups: 0`.
4. **K5 and K6 (documentation corrections; required because the lock pins the documents).** `clinical-safety.md` states that no partial window fails on the development seeds, which its companion `sim-access-5c4j.json` contradicts; the protocol promises reports the adjudicator does not emit; the follow-up's effect on the financial observation is unstated.

What a lock commit must pin, once the blockers are closed, is in section 12.

## 1. What was read and run

Read: the 5C.4g and 5C.4i audits and the maintainer's responses; `docs/calibration/clinical-5c4j/{clinical-safety.md, reserved-validation-protocol.md, scenario-manifest.json, construction-checks.json, sim-access-5c4j.json, README.md}`; `docs/calibration/clinical-5c4b/candidate-policy-v5.json` and `order-lifecycle.md`; `internal/eval/{clinical.go, measure.go, policy.go, lostdemand.go, record.go, protocol.go, exact.go, compare.go and evaluate.go (relevant parts), reserved_manifest_test.go, oc_fidelity_test.go, remediation_5c4j_test.go, boundary_safety_test.go, fixture_test.go, clinical_test.go}`; `internal/sim/{export_eval_test.go, evaluation.go, flow.go, demand.go, ops.go, reading.go, world.go, engine.go, followup_prefix_test.go, clinical_profile_test.go, clinical_access_test.go}`; `internal/session` and `testdata/sessions`.

Run (logs in `scratch/logs/`): 24 independent witness tests (`scratch/witnesses/zz_audit_*_test.go`, 22 in package `eval`, 2 in package `sim`), 39 targeted existing tests of `internal/eval`, 9 of `internal/sim`, and the 14 replay tests dev.15–dev.21 of `internal/session`. Reproduction: copy `zz_audit_[abdefg]_test.go` into `internal/eval/` and `zz_audit_c_*_test.go` into `internal/sim/` of a checkout of 04e97e5, then

```
cd internal/eval && go test -c -o /path/eval.test . && /path/eval.test -test.v -test.run 'TestAudit'
cd internal/sim  && go test -c -o /path/sim.test  . && DT_AUDIT_SEEDS=901,902,903,904,905,906,907,908,909,910 /path/sim.test -test.v -test.run 'TestAuditC_Recorder'
```

`TestAuditE_*` and `TestAuditC_Recorder` skip under `-short`. The existing-test selections are in `scratch/witnesses/run-eval-h.sh` and `run-chain-2.sh`.

Digests of the artifacts as audited (`TestAuditG_Digests`, `scratch/logs/witness-bdfg-2.log`):

| File | SHA-256 |
|---|---|
| `docs/calibration/clinical-5c4b/candidate-policy-v5.json` | `9efa6e63f38de9cfbfdb829ac1a82eff114d6e5c523d32df5d46f53b8c429e7c` (as in protocol §1 and the response) |
| `docs/calibration/clinical-5c4j/scenario-manifest.json` (version 2) | `298b5f79ace3fa59cfe939cca61a10906d2a890e5571a4894be969157a4245a6` |
| `docs/calibration/clinical-5c4j/reserved-validation-protocol.md` | `7b6128dd90dedc9735145097cb5ce3604874cf15325750f8d66a62217ffee247` |
| `docs/calibration/clinical-5c4j/clinical-safety.md` | `25f1a709f9f849290eca4943ad283af79b035cf97d6dbc9ff140b9d4ef6a8321` |
| `internal/eval/testdata/candidate-policy-v5-5c4h.json` | `e41d53bac22c00920257c9437bba641b425c7fd2234ff4dddde45dd23ebdce0e` (the 5C.4i audit's candidate) |
| `internal/eval/testdata/candidate-policy-v5-5c4f.json` | `d901fe5877684567c6c2d19ad1c063e8cf36ce269ba6fd44ba54299f052e348b` |
| `docs/evaluation/policy-v0.2-eval-4.json` (in force) | `4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3` |
| `docs/evaluation/holdout-v0.2.json` | `c30478f1feeca3c925fdee868f87a59b1afbc56ab8e48a7215e36fcdcee5c699` |

No `docs/evaluation/holdout-unseal.json` exists (sealed). `familyAllowed` refuses 20261201 (`reserved_manifest_test.go:621`; `TestReservedManifestRefusesReserved` passed).

## 2. Findings

Each finding states its severity and kind. "Confirmed" means reproduced by a witness in this audit.

### K1 — Relink-and-retime bypass of the order–study contract. High. Confirmed implementation defect. Blocks lock.

The contract (`internal/eval/clinical.go:1095–1143`, `checkOrderStudies`) binds every post-takeover study to exactly one order of its priority and modality, served, resolved at the study's acquisition (`:1119–1129`), with no second study (`:1130`), and requires every served order received after the takeover to have exactly one study (`:1135–1141`). Every one of these comparisons is between fields of the same record. The only independent source the evaluator consults is the paired baseline's demand, and only for orders received after the takeover and wanted in the gated period (`lostdemand.go:182–201`, `eligibleDemand`, `:195`).

Witness: `zz_audit_b_test.go`, `TestAuditB_Contract`, case "relinked to a same-priority same-modality order wanted after H, re-timed consistently" (`scratch/logs/witness-bdfg-2.log` line 8). Starting point on every candidate seed: the 5C.4i I2 situation (the last soon order wanted by H, acquired H + 1 s, reported 60 h past due), confirmed NO GO with safety FAIL in the control case. The manipulation, within one record:

1. choose another soon CT order wanted after H (always present; outside Gate B's eligible set on both sides);
2. delete that order's own study; set its received, wanted and resolved times to the violating study's acquisition (H + 1 s) and its outcome to served;
3. relink the violating study to it;
4. mark the original order lost at receipt.

Expected: INVALID (the study no longer images the order it was acquired for). Observed: **VALID, GO** on all 40 seeds. The study is now "of an order wanted after H" and leaves the absolute cohort (`clinical.go:335–341`); the original order's loss costs 1 of about 900 soon orders in Gate A and Gate B; pairing is untouched because the re-timed order was never eligible on either side.

Clinical significance: a concealed absolute reporting violation obtains a favourable verdict, the same class as 5C.4i I2, whose original forms (duplicated row, naive relink, deletion) are now INVALID (confirmed) while this form is not. Eighteen other single-field manipulations were tried; every one was INVALID (section 4).

Required remediation (implementation): compare the candidate's whole post-takeover order set with the paired baseline's by origin, not only the eligible orders, and require identical requests (received, wanted, priority, modality, walk-in): under paired exogenous demand this identity holds for every order, so valid records lose nothing, and the re-timed order of step 2 is unmatched. Alternatively derive a post-horizon study's cohort membership from the baseline's wanted time for the same origin. Add the case to `TestOrderStudyContract`.

### K2 — Inherited orders are outside every check that would preserve their reporting obligations. High. Confirmed implementation defect; an owner decision is also needed. Blocks lock.

The contract exempts "an order received at or before the takeover and served after it without a post-takeover study" (`clinical.go:1137`, the `o.ReceivedSec > t0` guard; `clinical-safety.md` §2 calls it work "the records cannot reconcile"). Inherited orders are also outside Gate B pairing (`lostdemand.go:195`) and outside Gate A's decided cohorts (`clinical.go:263–280`: inherited access is reported; the maximum wait is report-only under the candidate, `maxWaitReportOnly: true`).

Witness: `TestAuditB_InheritedBypass` (records with 40 inherited orders so that case selection's inherited stratum has evidence; the baseline carries the same inherited order reported on time; `scratch/logs/witness-bdfg-2.log` lines 22–29):

| Manipulation of one inherited soon order (received t0 − 1 d) | Expected | Observed |
|---|---|---|
| control: imaged day 1, reported 60 h past due | NO GO (safety FAIL) | NO GO, safety fail |
| its post-takeover study deleted, the order still served | INVALID | **VALID, GO, safety pass** |
| order and study relabelled routine (14-day limit) | INVALID or NO GO | **VALID, GO** |
| control: imaged H + 1 s, reported 60 h past due | NO GO | NO GO |
| the same, study deleted | INVALID | **VALID, GO** |
| the same, wanted time moved to H + 1 s | INVALID | **VALID, GO** |
| control: imaged day 1, unfinished at E (overdue) | NO GO | NO GO |
| the same, study deleted | INVALID | **VALID, GO** |

The exemption is never exercised by either record producer. In the simulator a study is created in the same transaction that marks its order imaged (`internal/sim/ops.go:374–379` and `reading.go:76–82`: `acquiredAt` and the order's `endedAt` are both `e.Now()` at exam end), so an inherited order served after the takeover always has a post-takeover study. `TestAuditC_Recorder` on seeds 901–910 (`scratch/logs/sim-c-all.log`): 1,043 to 1,137 inherited orders served after the takeover per seed, **0** without a study on every seed. In the fidelity model a served order's resolution is its appointment and its study's acquisition (`oc_fidelity_test.go:592–598`); `TestAuditE_FidelityInherited` on A85: 1,066 inherited orders, 1,039 served after the takeover, 0 without a study.

Clinical significance: the inherited backlog is large (seed 901: 1,215 inherited orders in the record, 1,092 imaged after the takeover). Every one of those studies carries an absolute reporting obligation that can be dropped from a record without detection.

Required remediation: (implementation) remove the second exemption: every order served after the takeover, inherited or new, must have exactly one post-takeover study, and the first exemption must be limited to studies acquired at or before the takeover; (owner decision) whether inherited orders' requests are to be paired with the baseline's: they share origins and the pre-takeover history on a seed, so pairing is available and would close the relabel and wanted-time variants.

### K3 — Internally consistent re-timing is undetectable from one record. Limitation of the pure evaluator, to be stated. Not a blocker on its own.

`TestAuditB_Contract`, case "acquisition and resolution moved together 40 h later": both the violating study's acquisition and its order's resolution move 40 h later with the report unchanged, so the delay shrinks from 60 h to 20 h past due: VALID, GO. `TestAuditB_BothSidesEdited`: moving the wanted time past H on both records of a pair: VALID, GO. The evaluator is pure over records (`evaluate.go` package comment); samples are reconciled against the studies (`measure.go:1136–1160`) but are part of the record; `AppointmentsSHA256` is carried, not checked. The reserved run constructs its records in process and is not exposed; a file-based evaluation (the holdout runner) relies on the harness and the committed record digests (`protocol.go:298–326`). The protocol should state this reliance.

### K4 — A criterion with no judged group can be PASS; coverage is relative to the invocation. Medium. Confirmed adjudicator defect and execution-restriction gap. Required before lock.

`reservedCriteria` (`reserved_manifest_test.go:968`) takes a `seeds` function and judges coverage against it (`manCoverage`, `:901`). `TestReservedCriteria` (`:1465–1473`) derives it from `DT_MANIFEST_SEEDS` and `DT_MANIFEST_SCENARIOS`; a scenario outside the filter is declared to have zero seeds, so its absence is not incomplete coverage. A criterion whose scenarios are all absent has nothing to fail and reports PASS: T4 (`:1182`, `met != len(by[id])` with both zero), T2G (`:1164–1170`, no material scenario), T1 and T7 (`:1019–1049`, no group).

Evidence: the maintainer's `construction-checks.json`, second pass (the stable and near-margin scenarios at 200 seeds), reports `"T4": "pass"` with `"groups": 0, "met": 0` for all fourteen witnesses and `"T2G": "pass"` with `"groups": 0` for all eleven deterioration scenarios. Witness `TestAuditF_InvocationDependence` (`scratch/logs/witness-bdfg-2.log`): a run containing only S-A85's lines is T1 PASS when the other three stable scenarios are declared absent, INCONCLUSIVE under the full manifest; one group per scenario is T1 and T8 PASS under `DT_MANIFEST_SEEDS=40` semantics. The criteria output records `groups` per scenario but not the seeds-per-scenario function or the scenario filter in force.

Required remediation: (implementation) a criterion none of whose scenarios has a judged complete group must be INCONCLUSIVE, never PASS; the output must record the seeds per scenario and the filter it was judged under; (execution restriction) the lock must pin the exact invocation of the reserved run and of its adjudication: `DT_MANIFEST_SEEDS` and `DT_MANIFEST_SCENARIOS` unset, `DT_MANIFEST_FAMILY=20261201` accepted only by the authorized harness.

### K5 — `clinical-safety.md` misstates the partial-window result on the development seeds. Low–Medium. Documentation defect.

`clinical-safety.md` §1.4 ("The simulator's no-action practice (seeds 901–910, §3.1): no partial window fails") and §3.1 ("no partial window fails") are contradicted by `sim-access-5c4j.json` in the same directory: under the candidate (`partialRule: rateCeiling`, four-week soon windows) the leading partial soon window **fails** on seeds 901 (3 of 42), 902 (3 of 41) and 903 (4 of 41), each with `partialFailingWeeks: 1`. The seed-level statement ("every seed has the same clinical status and the same failing gates as in 5C.4h") holds only because the soon weekly gate already fails on complete blocks of those seeds. The statement must be corrected. It bears on the owner's false-rejection question: the evaluation scenario's own practice, with no action, has about 40 soon acquisitions in the five days before the anchor Monday, and the ceiling fires there on 3 of 10 seeds.

### K6 — Promised reports the adjudicator does not emit; an unstated financial effect of the follow-up. Low. Documentation defects.

* Protocol §4 T1 "Reported": the rejection rate on the stress practices (A92, B92, C92, Cs92) and partial-window ceiling failures by seed are not in the T1 detail (`:1043–1046`); the stress scenarios appear in no criterion's detail.
* T2 "Reported": per-scenario detection and "the run's exact conditional prediction from each group's harmed count and its other failing seeds" are not computed; only the constant 0.371 is echoed (`:1124`). The H10 and OH10 rates are not reported anywhere.
* T4 W-ACCESS65's expectation is tautological: `accessClaims` emits constants (`lostdemand.go:405–419`), so `witnessMet` (`:1323`) cannot fail on a valid group. It verifies that the constants are present, nothing about Gate B at 65% loss.
* The 45-day follow-up moves the `end` capture from H + 31 d to H + 45 d. Minimum cash is checked through the end of observation and the funding gap is taken at `end` (`measure.go:385–388, 406–409`); `clinical-safety.md` §3 lists the clinical recorders affected but not this change to the financial observation. v5 is not in force, so no historical evaluation changes, but the interaction the 5C.4i audit asked to be defined (I3, "measurements that already use end") is still undocumented.

### K7 — Statistical limitations of the development evidence. Not defects; to be carried into any result.

* 0 of 28 stable groups rejected (20 at 200 seeds, 8 at 40) bounds the per-group false-rejection probability only loosely (a Clopper–Pearson 95% upper bound of about 0.12 for 0 of 28 if the groups were independent, which they are not: the eight 40-seed groups reuse the 200-seed pass's seeds). It shows no regression on the development family, as the owner says; it does not establish the T1 target (at most 3 of 62).
* The ceiling's effect on the evaluation scenario's own practice is 3 of 10 seeds (K5), on seeds that fail other gates.
* T5a at 40 seeds did not conform on NM2 (observed 0.70 against a prediction near 0.5, one group, cluster SE 0) and was INCONCLUSIVE by the two-group rule; at 200 seeds it conformed (NM2 0.56 against 0.508, NM2.1 0.31 against 0.303). The 25-group reserved design will be the first test of T5a at its own size.

## 3. Part A — partial-window safety: GO

Code: `weeklySevereCalendar` (`clinical.go:520–649`). The rate-ceiling branch is `:613` (`rated && w.Acquired > 0 && rateAbove(w.Severe, w.Acquired, bpRate)`), judged before the below-minimum NOT_JUDGED branch (`:620`) and only for windows below the minimum; a partial window at or above 100 falls through to the ordinary rate (`:624`). `rateAbove` is an exact cross-product comparison (`exact.go:36`). Windows are anchored at the first Monday at or after the takeover (`:526–530`); severity is decided through E (`:575`, `delayAt(…, E)`).

Witnesses (`zz_audit_a_test.go`, all PASS, `scratch/logs/witness-abdfg.log`):

* `TestAuditA_WindowStatuses` (29 layouts): 0/6 NOT_JUDGED; 1/6, 5/6, 6/6, 1/1, 5/99, 1/19, 2/39, 4/79 FAIL; 4/99, 1/20, 2/40, 4/80 NOT_JUDGED; 5/100 PASS (judged); 6/100 FAIL; trailing 1/6 FAIL; two partial windows each judged alone (1/19 and 0/81: fail, not_judged, where pooled 1/100 would pass; 3/50 and 3/50: fail, fail; 2/50 and 3/50: not_judged, fail); empty partial windows NOT_JUDGED; empty or 99-study complete windows INSUFFICIENT even with 6 severe (never failed, never passed); complete 100 with 5 PASS and with 6 FAIL; a routine 1/3 leading week FAIL.
* `TestAuditA_ExactEquality`: for k = 1…100, k of 20k is never above 5% and k of 20k − 1 always is.
* `TestAuditA_AnchoringAndNoDilution`: a study one second before the anchor Monday is in the leading partial week (1/1 FAIL); at the Monday's first second it is in the first complete block (1/101 PASS); a study at H is in the trailing window, at H + 1 s outside the gated period; 500 later compliant acquisitions leave a 1/19 leading FAIL unchanged; a trailing-window study unfinished at E is severe (censored), one finished within the hour is not.
* `TestAuditA_NoJudgedWindow`: a short horizon whose only window is partial at 1/20 is INSUFFICIENT with the reason "no window judged … each at or below the rate (too few to establish it)"; at 2/20 it is FAIL.
* `TestAuditA_AllowanceInteraction` (full 40-seed evaluations): 1/6 on 4 seeds GO, on 5 NO GO; 3 partial-window seeds plus 2 seeds failing urgent timely access NO GO, plus 1 GO; a seed failing both gates counts once (4 distinct seeds GO, 5 NO GO); sparse front-loaded demand (1 severe of 3 in the leading week) on 5 seeds NO GO; the 640-report exploit (16 of 24 on every seed) NO GO with 40 failing seeds.
* `TestAuditA_Disclosure`: a passing seed's reason reads "2 partial windows below the minimum not judged, each at or below the rate (too few to establish it)".

Assessment: the rule is implemented exactly as written and is deliberately conservative. False rejection on stable practices: a practice whose leading or trailing partial window holds few soon acquisitions fails the seed on one severe report; up to four such seeds in 40 are absorbed by the shared allowance and a fifth rejects the evaluation. The maintainer measured 0 of 1,120 fidelity-model stable seeds firing; on the simulator's own practice the window fires on 3 of 10 development seeds (K5), each already failing elsewhere. That is the owner's deliberate choice, not a defect; the documentation must describe it accurately.

## 4. Part B — order–study integrity: NO GO

See K1, K2 and K3. The remaining attacks of the brief are each INVALID as required (`TestAuditB_Contract`, `scratch/logs/witness-bdfg-2.log` lines 2–19): duplicate order ID ("names 0 orders"); duplicate study ID ("its ID repeats"); study deleted with the order served ("served with no study"); two studies on one order ("more than one study"); relink to an order that has its own study (resolution mismatch); relink to a lost order ("ended lost, not served"); study acquisition moved alone; order resolution moved alone; modality mismatch; priority mismatch; order outcome open or lost with the study kept. Caught by the paired baseline rather than the contract (NO GO through Gate B INSUFFICIENT, 40 seeds unmatched): the order and study relabelled routine together; the order's wanted time moved to H + 1 s; the order received and wanted after H. Legitimate exemptions confirmed (`TestAuditB_InheritedLegitimate`): studies acquired before the takeover with no order in the record and with duplicate IDs are accepted and counted (against the candidate, never dropped).

Post-horizon cohort manipulation: `TestAuditD_Undecided` confirms that an order wanted after H and reported 60 h past due is outside the absolute cohort by design (protocol §7); K1 shows that a violation wanted by H can be moved into that position in one record.

Integrity checks comparing two fields of the same record, as the owner asked: the resolution/acquisition equality (`clinical.go:1126`), the served outcome (`:1123`), the request equality (`:1120`) and the imaged-before-wanted check (`:783`) are all intra-record. The paired baseline is the only external reference, and it reaches only orders received after the takeover and wanted in the gated period. K1 and K2 are the two consequences.

## 5. Part C — simulator recorder: GO

The repair (`internal/sim/export_eval_test.go:97–115`): the recorder reads the engine's pending events (`engine.go:243`, `PendingEvents`, a sorted clone of the heap; read only) and adds to the inherited orders every order in state `orderNoShow` with a `kindNoShowRebook` event queued, beside the booked, recall and on-site orders. The rebook event is pushed at accounting close (`flow.go:276–289`, 40% of no-shows, for 09:00 the next day) and re-offers from the current day (`flow.go:296–300`, `noShowRebook` → `offer`), which never writes `o.earliest` (no assignment to it exists outside order creation: `demand.go:535`, `flow.go:349`), so the recorded wanted time (`export_eval_test.go:417`, `StartOfDay(o.earliest)`) is the original one.

Witness `TestAuditC_Recorder` (`zz_audit_c_test.go` with read-only hooks `zz_audit_c_hooks_test.go`; `scratch/logs/sim-c-all.log`, seeds 901–910, 16 s a seed): the world is run to the takeover, every order's state and the queued rebookings are read, the run is recorded under the candidate to H + 45 d, and the record is compared with the takeover state.

| Seed | Orders at takeover | Inherited in record = expected | booked / recall / visit / no-show with rebooking | No-shows without rebooking | Post-takeover studies naming absent orders before the repair |
|---|---|---|---|---|---|
| 901 | 12,118 | 1,215 = 1,215 | 1,154 / 57 / 0 / 4 | 299 | 3 |
| 902 | 12,094 | 1,224 = 1,224 | 1,160 / 64 / 0 / 0 | 299 | 0 |
| 903 | 12,240 | 1,264 = 1,264 | 1,198 / 66 / 0 / 0 | 314 | 0 |
| 904 | 12,142 | 1,174 = 1,174 | 1,114 / 59 / 0 / 1 | 304 | 1 |
| 905 | 11,940 | 1,201 = 1,201 | 1,127 / 73 / 0 / 1 | 321 | 1 |
| 906 | 12,058 | 1,192 = 1,192 | 1,122 / 69 / 0 / 1 | 307 | 1 |
| 907 | 12,129 | 1,223 = 1,223 | 1,166 / 54 / 0 / 3 | 292 | 3 |
| 908 | 12,203 | 1,261 = 1,261 | 1,196 / 64 / 0 / 1 | 301 | 1 |
| 909 | 12,270 | 1,246 = 1,246 | 1,192 / 53 / 0 / 1 | 308 | 1 |
| 910 | 12,152 | 1,171 = 1,171 | 1,117 / 52 / 0 / 2 | 344 | 1 |

Verified on every seed: no inherited order appears twice; every expected inherited order is present and no other; each inherited order's wanted time, receipt, priority and walk-in flag equal the takeover state; every post-takeover study names an order of the record, with the world's own study → order map agreeing; no post-takeover study of the world is absent from the record; no no-show without a queued rebooking later produced a study; every served rebooked no-show has exactly one study acquired at its resolution; every inherited order served after the takeover has a study (0 without, all seeds); the 45-day record is VALID under the candidate with no integrity failure. The pre-repair record (the rebooked no-shows removed) is INVALID under the candidate on 8 of 10 seeds (901, 904–910; 1 to 3 studies each: "names 0 orders"), exactly as `clinical-safety.md` §3.1 reports; seeds 902 and 903 have no such order and are unchanged.

Lifecycle traced (seed 901, four routine orders wanted on day 179, no-show at the day-178 close six hours before the takeover, rebooking call queued): resolved +62.6 h, +182.0 h and +320.1 h after the takeover as served, each with one study acquired at the resolution and reported within 2 to 15 h; one resolved +43.8 h as a second no-show (`patient`); captures T30 … end carry the final outcome. Across the ten seeds the 14 rebooked no-shows end served (12), patient (1) and lost (1, the re-offer found no slot: an inherited loss, reported apart).

Gate A and Gate B denominators: inherited orders are never in the new-demand cohorts (`clinical.go:244`, `lostdemand.go:195`). `TestAuditC_Recorder` evaluated the repaired and the pre-repair record under the 5C.4f candidate (which has no link check, so both are valid): on seeds 901, 904 and 910 every clinical gate and every service gate is byte-identical and the clinical status is the same (`scratch/logs/sim-c-5c4f.log`); only the reported inherited-access cohort differs by the rebooked no-shows (seed 901: wanted 1,172 against 1,168, timely 1,050 against 1,047, withdrawn 95 against 94). So the repair is (1) a legitimate completeness change of the recorder, with (2) no simulator behaviour change: `TestFollowUpPrefixUnchanged` (31 against 45 days on seed 901: 13,875 orders, 9,322 studies identical through day 181), `TestDemandExogenousAcrossPlans`, the three `TestStateDigest*` tests, `TestRegress14DigestRightAfterNoise`, `TestEvalRecordOnRealRuns`, `TestHoldoutProtocolAuthorize` and `TestHoldoutRecordersGuarded` all pass (`scratch/logs/sim-existing.log`), and the recorder is test code (`export_eval_test.go`) outside the shipped binary; and (3) historical compatibility is kept: the dev.15–dev.21 replay fixtures replay and continue exactly (section 10), and no historical evaluation record was produced by this recorder (the pinned historical references are v0.1 reports, unchanged).

Residual observation, not a defect: a no-show without a queued rebooking is left out, as it should be (it can never be served again: `world.go:255` discards a stale rebook, and no other path leaves `orderNoShow`); the record's lifecycle document (`order-lifecycle.md` §2) still says "no link between an order and its study" is recorded, which has been untrue since 5C.4h.

## 6. Part D — horizon and follow-up: GO

Code: `observationNeeded` (`measure.go:337–351`) adds H + followUpDays; `Clinical.validate` (`policy.go:616–627`) refuses a follow-up without the cohort or shorter than access + target + absolute for any priority (routine 43,200 + 1,440 + 20,160 = 64,800 min = 45.00 d exactly, so 44 is refused and 45 is the least); the absolute loop (`clinical.go:325–367`) classifies at E through `delayAt` (`:190`): Observed when finished beyond the limit, Overdue when unfinished and E − due > limit, Pending when unfinished and E − due ≤ limit; `checkStudies` (`measure.go:513–527`) refuses any acquisition or final after E.

Witness `TestAuditD_Boundaries` (16 cases, all as expected, `scratch/logs/eval-h-and-fixes.log`): acquired H + 1 s and reported 60 h past due FAIL (observed, in the post-horizon cohort); acquired before H and reported H + 1 s FAIL; exactly 48 h past due PASS, one second more FAIL; unfinished at E − 52 h − 1 s FAIL (overdue), at E − 52 h INSUFFICIENT (pending: due + limit exactly at E), at E − 10 h and at E INSUFFICIENT; acquired E − 1 min and finished at E PASS; acquired E + 1 s INVALID; a report finished E + 1 s INVALID; routine wanted at H, imaged at its 30-day deadline and finished exactly at E PASS, unfinished INSUFFICIENT, imaged one second later and unfinished INSUFFICIENT, imaged at H + 29 d and unfinished FAIL (overdue), finished at E + 1 s INVALID. `TestAuditD_Observation`: a record observed to E − 1 s is INVALID.

`TestAuditD_Undecided`: an order wanted by H and still open at E is counted `unacquired` (1) and untimely by Gate A (undecided 0: after 45 days no open order is within any access limit); a pending seed's reason names the limit ("unfinished and not yet observable against the absolute pastDue limit"); pending on 40 seeds is NO GO.

No obligation disappears because its acquisition or report crosses the horizon, and nothing unresolved is PASS. The exact-deadline exception stands as documented (INSUFFICIENT). The residual limitations are those protocol §7 lists, plus the financial-observation gap of K6.

## 7. Part E — randomness stability: GO

Fidelity model: prefix-stable draws (`oc_fidelity_test.go:426–434, 640–646`) take each study's documentation and read uniforms from `splitmix64(bsalt + 2·idx)` and `(bsalt + 2·idx + 1)`, where `bsalt` is the booking stream's first draw and `idx` the order's creation index; the modality uniform is `splitmix64(salt + idx)` from the demand stream's first draw; booking and cancellation draws are made in time order inside the hour loop (`:535–600`), and the day factor is drawn from the demand stream at each midnight (`:544–547`). Nothing in the first 31 + 150 days depends on how many days follow.

Witness `TestAuditE_FidelityPrefix` (`scratch/logs/eval-e.log`): for every one of the 25 manifest practices (A85…WgU92, IL40-85, M), on the debug family PCG(1, 0) with booking stream PCG(1, 10,000,000), the 31-day and 45-day records agree up to the earlier end on every order's origin, priority, modality, receipt, wanted time, walk-in flag, first appointment and resolution, every capture outcome before the earlier end, every study's order, priority, modality, acquisition, documentation, claim, read start, remote flag and final time when by then, and every daily sample: 0 studies changed and 0 other differences in each (about 10,800 orders and 9,300 to 10,000 studies a record); every 45-day record has no integrity failure under the candidate. The maintainer's `TestFidelityFollowUpPrefixStable` (A85; the committed model shown changing) and the model's pinned digests (`TestOCFidelityModelUnchanged`), aligned weeks, paired demand and calibration tests all pass.

Stream collisions: role spans over the manifest's actual k and seeds are demand [0, 4998], then [10,000,000, 14,800,999], [20,000,000, 24,800,999], [30,000,000, 34,800,999], [40,000,000, 44,800,999]: disjoint (`TestAuditG_ManifestAgainstProtocol`). PCG(F, idx) streams are positions on one PCG-DXSM cycle selected by the second seed; an overlap within the draws used is improbable at the 2^128 period, not impossible, and should be stated as such. Order identity: IDs and origins are the creation index, assigned before the inherited filter (`:584, :611–618`), so they are stable across observation lengths (verified by the witness).

Simulator: the recorder draws nothing (`export_eval_test.go` reads state; `originID` and `PendingEvents` are read-only); `TestFollowUpPrefixUnchanged` passes (section 5). The operational simulator is unchanged by 5C.4j: the state digests, the demand-pairing contract and the dev.15–dev.21 replays pass.

## 8. Part F — reserved adjudicator: GO with one required repair (K4)

Shape (`TestAuditF_Shape`): a complete run holds, for each 2,500-seed scenario, 62 complete groups and one marked 20-seed remainder (seeds 2480–2499); 25 groups for a 1,000-seed scenario; 25 × 20 replicate lines for each near-margin scenario; T1 S-A85 `groups: 62, limit: 3`; T3 S-A85 `seeds: 2500`; T6 `seeds: 10000`; T5a NM2.1 `evaluations: 500`.

Malformed and incomplete result sets (`TestAuditF_Malformed`, 48 expectations, all met, `scratch/logs/eval-h-and-fixes.log`): the remainder marked as a complete group, a 40-pair remainder, a 63rd group beyond the manifest's seeds, a group missing one seed, a duplicated seed, a shifted seed range, a missing replicate, a duplicated replicate, an unknown scenario identity, an invalid record in a remainder, a construction error: each makes its criteria and T8 INCONCLUSIVE; 4 rejected groups beside an invalid record is FAIL (FAIL takes precedence over INCONCLUSIVE, as protocol §2 says).

Rules: T1 3 of 62 PASS, 4 FAIL, a rejected remainder not counted; T2 200 harmed and 190 flagged PASS, 189 FAIL, 199 harmed INCONCLUSIVE, a flagged harmed seed not failing in the allowance breaks T2-conformance; T2G soon 2 of 100 with routine 0 of 10,000 is material and FAILs when not rejected (the 5C.4i pooled witness), routine exactly 1% alone is material, both at 0.99% are not, 22 of 25 FAIL and 23 PASS; T3 one overdue violation in a remainder seed FAIL, a violation on a stress practice not judged; T4 W-HORIZON with 39 of 40 violating seeds FAIL, W-PENDING rejected by safety instead FAIL, W-MARGIN-AT rejected by Gate B FAIL, W-ID-REMOVED unmatched on the second seed FAIL, W-ID-REPEATED with two unmatched seeds FAIL; T5b 200 of 500 at 2.1 and 5 of 500 at 3 PASS, 201 or 6 FAIL (the comparison `rate > tolerance` is exact at 200/500 and 5/500), a missing or duplicated replicate of a passing run INCONCLUSIVE; T6 19 flags PASS, 20 with two in the remainder FAIL, counted per priority; T7 three raw-insufficient groups plus one with a pending seed FAIL (4 of 62), timely-access insufficiency is not a window gate, remainder insufficiency not counted; T8 a quota injection one short on either side INCONCLUSIVE, exact PASS. T5a cannot become INCONCLUSIVE once it FAILs (by design).

Raw T7 (`TestAuditF_RawInsufficiency`): a complete soon block of 99 beside a block failing its rate is gate FAIL and raw-insufficient (`clinicalInsufficientSeedsByGate` holds the seed, `clinical.go:995–1001`): the 5C.4i I6 masking is repaired.

File reading (`TestReservedCriteria` on a 3,904-line run written by `TestAuditF_WriteLines`): the intact file is judged (`scratch/logs/fake-lines.criteria.json`); the file truncated mid-line fails before judging ("unexpected end of JSON input", `reserved_manifest_test.go:1449`); the file cut at a line boundary (30 lines short) judges T4 and T8 INCONCLUSIVE and the others as before. A scanner error fails the test (`:1453`).

Not verifiable from result lines: the stream assignment and the witness mechanism behind a line (lines carry no stream). They are fixed by the harness code the lock pins (`manPair`, `:166–208`; `manApply`, `:245–402`; `streamIndex`, `:45`).

Every missing, invalid or inconclusive mandatory result prevents confirmation, with the one exception of K4 (an absent scenario declared absent by the invocation).

## 9. Part G — protocol consistency: GO with documentation corrections

* **Manifest and policy.** `manifestPolicy` is the fixture policy with the candidate file's clinical section (`:723`, `clinicalFrom`). `TestAuditG_HarnessPolicyEqualsCandidate`: every rule section of the harness policy (schema, horizons, service, caseSelection, backlog, finance, payback, seeds, clinical) is byte-identical to the candidate file's; only the scenario fingerprint differs ("fp" against the simulator scenario's), which the fidelity records carry. The lock should state this, or the harness should load the candidate file directly; the manifest's `policy.sha256` must be filled in.
* **Scenario construction.** `TestAuditG_ManifestAgainstProtocol`: the criterion sets of §4 are exactly the manifest's (T1/T3/T6/T7 the four stable practices at 2,500 seeds; T2 H15, H20, SHOCK, OH15, OH20; T2G the eleven deterioration scenarios; T4 the fourteen witnesses; T5 NM0–NM3 at 20 replicates on S-A85's pairs; T8 all 49; 10 scenarios report only); tolerances 0.40 and 0.01 pinned; group size 40; 62 × 40 = 2,480 and a remainder of 20. `TestReservedManifestWellFormed` passes.
* **Paired exogenous demand.** `manPair` draws both plans' demand from PCG(F, 2i) and refuses practices whose demand models differ (`:176`); `TestReservedManifestUnpaired` and `TestOCFidelityPairedDemand` pass.
* **Stream disjointness and group/remainder treatment.** Section 7 and section 8.
* **Numerical thresholds.** T1/T7 `len(gs)*5/100` = 3 at 62; T2 0.95 and 200; T2G 1%, 48 h, 3 points, 23 of 25; T3 zero; T5a 4 cluster SE + 0.02; T5b 0.40 and 0.01 read from the manifest (`manifestTolerances`, `:879`, requiring an "APPROVED" status); T6 19 of 10,000: all as written.
* **Intervals and multiplicity.** Clopper–Pearson two-sided 95% (`cpInterval`, checked against 0 of 2,500 → 0.1197%); Bonferroni at 0.05/4 for T1 and 0.10/12 for T3's one-sided bounds; the dependence qualification on T2 and T6: present. Reporting gaps: K6.
* **PASS / FAIL / INCONCLUSIVE semantics and mandatory consequences.** Computed by fixed code from the lines; §6's consequences are documented and the development runs state their non-confirmatory nature. The one way a result could be read differently after execution is K4.

## 10. Part H — historical compatibility: GO

All passed (`scratch/logs/eval-h-and-fixes.log`, `sim-existing.log`, `session-replay.log`, `eval-e.log`):

* **v4 byte-identical:** `TestV4Neutrality` (digest `eef98fab6cd6355eb854b86675f914ecc8be02ba6cfcd5bd83f607dd98f0b070`); `TestPolicyFrozen`; `TestCandidatePolicyFile`.
* **5C.4d/e/f/h candidates:** `TestOldV5PolicyCompatibility` (5C.4d), `TestOld5C4ECompatibility`, `TestOld5C4FCompatibility`, `TestPartialWindowCeiling` and `TestHorizonLinkIntegrity` (5C.4h), `TestRateCeilingFullVerdicts` and `TestOrderStudyContract` (the 5C.4h and 5C.4f verdicts reproduced beside the candidate's); the four testdata files at their digests (section 1).
* **Historical artifacts:** `TestHistoricalReferencesUnchanged`.
* **Replay through dev.21:** `TestDev15…TestDev21RecordReplays` and `…RecordContinues`, 14 tests, exact.
* **Simulator unchanged:** section 5 and section 7.
* **No reserved use, unseal, activation, freeze:** `TestHoldoutSealed`, `TestAuthorizeSealedFirst`, `TestGuardSeeds`, `TestGuardRepo`, `TestHoldoutRecordersGuarded`, `TestReservedManifestRefusesReserved`; no unseal record exists; `PolicyPath` is v0.2-eval-4; the candidate's status says "PROPOSAL, not in force and not frozen"; the manifest's status says "PROPOSED … not locked".

The archive has no Git history; commit identity rests on the supplied snapshot identification, and historical non-execution of reserved streams cannot be proved from an archive. None was introduced by this review.

## 11. The owner's two concerns

* **The recorder repair.** It is necessary and correct: the omitted orders were real inherited obligations (a no-show whose patient has already called for a new date) that are served after the takeover on 8 of 10 development seeds, and the repaired recorder exports exactly the booked, recall, on-site and rebooking no-show orders, each once, with the original wanted time (section 5). The change in the inherited-order population is 1 to 4 orders a seed, visible only in the reported inherited-access cohort; no new-demand gate, no Gate A or Gate B denominator and no simulator behaviour changes. Earlier simulator evidence documents (5C.4b–5C.4h `sim-access-*.json`) were produced by the previous recorder and remain as they were; they are not evaluated under the contract.
* **0 of 28 stable groups.** Agreed: evidence of no regression on the development family, not of the 4.8% false-rejection target (K7). The published statement that the ceiling never fires on the simulator's own practice is false (K5).

## 12. What the lock commit must pin, once the blockers are closed

1. The candidate policy file by SHA-256, and the statement that the harness evaluates under its rules with the fidelity fingerprint (section 9), or a harness that loads the file directly.
2. `scenario-manifest.json` by SHA-256, with its `policy.sha256` field filled in rather than "pinned at the lock commit".
3. `reserved-validation-protocol.md` and `clinical-safety.md` by SHA-256, corrected for K5 and K6, with K3 stated.
4. The evaluator and harness code (`internal/eval/*.go`, `reserved_manifest_test.go`, `oc_fidelity_test.go`) by the lock commit's tree, with the precondition that every committed witness test passes at that commit (protocol §1), the repaired K1, K2 and K4 cases among them.
5. The exact reserved invocation and adjudication commands: `DT_MANIFEST_FAMILY=20261201` accepted only by the authorized harness, `DT_MANIFEST_SEEDS` and `DT_MANIFEST_SCENARIOS` unset, the output files and their digests to be committed, and the adjudicator's record of the seeds per scenario it judged (K4).
6. The fidelity model's pinned digests (`TestOCFidelityModelUnchanged`) and the prefix-stability tests as preconditions.
7. The T5 tolerances (0.40, 0.01) and every numerical threshold of §4, as already written.

## 13. Witness and log inventory

Witness sources, kept in `scratch/witnesses/`: `zz_audit_a_test.go` (Part A), `zz_audit_b_test.go` (Part B), `zz_audit_d_test.go` (Part D), `zz_audit_e_test.go` (Part E), `zz_audit_f_test.go` (Part F), `zz_audit_g_test.go` (Part G), all package `eval`; `zz_audit_c_hooks_test.go` and `zz_audit_c_test.go` (Part C), package `sim`; the wrappers `run-go.sh`, `run-eval-h.sh`, `run-chain-2.sh`.

Logs in `scratch/logs/`: `witness-abdfg.log` (Parts A, B first run), `witness-bdfg-2.log` (Parts B, D, F, G), `eval-h-and-fixes.log` (Part D final, Part F malformed sets, the compatibility tests), `sim-c-901.log`, `sim-c-all.log`, `sim-c-5c4f.log` (Part C), `sim-existing.log`, `eval-e.log` (Part E and the fidelity tests), `fake-lines*.jsonl`, `*.judge.log`, `*.criteria.json` (file-based adjudication), `session-replay.log`, `chain-2.log`.

The snapshot was verified byte-identical after the audit (`sha256sum -c snapshot.sha256`: 822 files, all OK, none added); nothing was written inside it.
