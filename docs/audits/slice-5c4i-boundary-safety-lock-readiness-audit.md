# Slice 5C.4i boundary safety and protocol lock readiness audit

**Audited revision:** commit 4908827 (0.2.0-dev.21), the revised candidate v0.2-eval-5 clinical policy and the proposed reserved protocol (not locked, not frozen, not in force), 2026-10-19. Line references are to 4908827.

The audit was independent, adversarial and read-only. The auditor read a `git archive` snapshot of 4908827 on a separate machine; disposable witnesses, overlays and logs lived in a scratch directory beside the snapshot, and the snapshot was verified byte-identical before and after. It ran no strategy, reference seed, holdout seed, reserved stream PCG(20261201, ·), unseal, activation or freeze. The links below point into that machine's snapshot and scratch directory, which were removed after the audit; the witness source and logs are kept by the maintainer outside the repository for remediation.

* **Everything below the rule is the auditor's report as delivered, unchanged.**
* **No remediation is in this commit.** Maintainer checks, if any, will follow in a separate document.

---

**Independent audit — Slice 5C.4i: Boundary Safety and Protocol Lock Readiness**

Audited snapshot: **4908827**, version **0.2.0-dev.21**.  
Policy: proposed v5 candidate.  
Audit date: **2026-10-19**.

**Final answer: NO GO for prospective lock or reserved calibration.**

The approved T5 tolerances are compatible with empirical Gate B and provide meaningful discrimination under the specified injection distributions. They do not resolve the clinical boundary problem or the implementation defects found here.

The revised partial-window rule correctly rejects the original 640-report exploit, but still permits **five severe reports among six acquisitions on every evaluated seed, with a full GO**. The horizon rule correctly distinguishes observed violations, overdue obligations and pending obligations when records are complete and correctly linked. Its record-integrity checks, however, allow reporting obligations to disappear from safety assessment.

The approved 45-day follow-up remains unimplemented. Simply extending the existing observation duration also changes earlier outcomes in the fidelity generator. The reserved result adjudicator has additional defects involving remainders, incomplete results, materiality, witness attribution and raw insufficiency.

| Decision | Verdict | Scope |
|---|---|---|
| **A. Partial-window safety** | **NO GO** | Correct count-ceiling implementation; inadequate evidence of small-cohort clinical safety. |
| **B. Horizon safety** | **NO GO** | Correct state comparisons, but incomplete linkage integrity and unimplemented follow-up. |
| **C. Clinical cohort consistency** | **NO GO** | Intended cohort distinctions are understandable; missing and mislinked studies can defeat them. |
| **D. T5 operating characteristics** | **GO for the approved tolerance design only** | Numerically discriminating under the specified distributions. Confirmation remains unestablished; the prediction implementation needs correction. |
| **E. Reserved protocol completeness** | **NO GO** | Written rules and executable adjudication disagree materially. |
| **F. Full revised v5 evaluator** | **NO GO** | Favorable verdicts remain possible with concealed reporting obligations and severe partial-window concentrations. |
| **G. Readiness for protocol lock** | **NO GO** | Resolve the clinical-policy decision and implementation blockers before pinning. |
| **H. Readiness for reserved calibration** | **NO GO** | Do not consume the reserved family. |

Component GO above does not authorize a lock, reserved execution, activation or freeze.

**Evidence and restrictions**

The review inspected the requested audits, response, policy, boundary documents, complete manifest, construction evidence, evaluator, simulator export and calibration harness.

Execution comprised **12 independently constructed witness tests**, **20 targeted existing evaluator tests**, and **six historical replay tests**. The final witness executions passed their assertions about the observed behaviors—including the defects below. The manifest construction test exercised all 49 scenarios on its permitted debug family.

All disposable files, overlays, caches and logs were placed under `scratch`. Tests ran one at a time with `GOMAXPROCS=2`. No strategy experiment, reference or holdout seed, environment-gated experiment, reserved-family draw, unseal, activation or freeze was performed.

A before/after content inventory found **808 files before and after, with no additions, removals or byte changes inside the snapshot**. See the [integrity result](/home/george/dt-audit-5c4i/scratch/logs/snapshot-integrity.json).

The candidate digest matches the protocol:

```text
e41d53bac22c00920257c9437bba641b425c7fd2234ff4dddde45dd23ebdce0e
```

The inspected manifest digest is:

```text
ee1cc84d5d05cdacf3e4b68669d40d6ee7ec20f73e865127437c410ebdd5b4e0
```

The archive has no Git history. Commit identity therefore rests on the supplied snapshot identification. Historical non-execution of reserved streams cannot be proved from an archive; no such execution was introduced by this review.

**I1 — The count ceiling closes the large exploit but does not establish partial-window safety. Severity: High. Clinical-policy limitation; blocks safety approval.**

The implemented comparison is exactly:

\[
\text{severe}\times10{,}000 >
\text{rate}_{bp}\times\text{minimum}.
\]

At 5% and minimum 100, a below-minimum partial window fails only with **more than five severe reports**. Otherwise it remains NOT_JUDGED. Complete below-minimum windows remain INSUFFICIENT.

References: [clinical.go:579](/home/george/dt-audit-5c4i/snapshot/internal/eval/clinical.go:579), [clinical.go:600](/home/george/dt-audit-5c4i/snapshot/internal/eval/clinical.go:600), [clinical.go:629](/home/george/dt-audit-5c4i/snapshot/internal/eval/clinical.go:629).

Independent full-evaluator witnesses used 40 matched synthetic seeds, sufficiently populated complete windows, and soon reports completed **30 hours past due**, below the 48-hour absolute maximum.

| Severe / partial-window acquisitions | Local severe fraction | Revised local status | Full verdict |
|---|---:|---|---|
| 5 / 6 | 83.33% | NOT_JUDGED | **GO** |
| 5 / 10 | 50.00% | NOT_JUDGED | **GO** |
| 5 / 50 | 10.00% | NOT_JUDGED | **GO** |
| 5 / 99 | 5.05% | NOT_JUDGED | **GO** |
| 6 / 99 | 6.06% | FAIL | **NO GO** |
| 5 / 100 | 5.00% | PASS | **GO** |
| 6 / 100 | 6.00% | FAIL | **NO GO** |
| 16 / 24 | 66.67% | FAIL | **NO GO** |

The original exploit was independently reproduced: **16 severe reports per seed × 40 seeds = 640 severe reports**. With the preserved 5C.4f policy it returned GO; with the revised candidate it returned NO GO. In this reconstruction, the T90 severe fraction was 16/558, so the per-seed rate did not reject it.

The smaller witness remains consequential: **200 severe reports across 40 leading windows**, each with five of six acquisitions severe, still produce GO. The T90 fraction is only 5/540 per seed. Dilution by later acquisitions explains why the aggregate rate does not repair the local safety gap.

Additional window witnesses established:

- Empty partial windows remain NOT_JUDGED.
- Two partial windows containing five severe reports each remain separately NOT_JUDGED; their counts are not combined.
- Two partial windows containing six severe reports each both fail.
- Empty complete windows are INSUFFICIENT.
- A short-horizon case with no judged window becomes INSUFFICIENT through `noJudgedWindow`. This is a window-level witness, not a valid full evaluation under the candidate’s minimum horizon.

There is **no adverse fail-to-pass cliff between 99 and 100 acquisitions** for the tested counts: six fails on both sides; five permits a favorable evaluation on both sides. The status changes from NOT_JUDGED to PASS at 100. The principal defect in the safety claim is the permissive treatment of small partial cohorts, not an arithmetic discontinuity at 100.

**Required remediation:** Resolve the small-cohort clinical-policy decision explicitly before lock. The present evidence does not support calling an 83.33%-severe boundary cohort safe. Any stronger rate rule, evidence requirement or other treatment must be specified and reviewed prospectively; none was substituted during this audit.

Witnesses: [TestAuditPartialFull](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go:22), [TestAuditPartialEmptyMultiple](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go:31). Results: [partial-window log](/home/george/dt-audit-5c4i/scratch/logs/witness.log).

**I2 — Missing and mislinked studies can conceal absolute reporting violations. Severity: High. Confirmed evaluator integrity defect; blocks lock.**

For post-horizon studies, validation checks that the referenced order ID occurs exactly once and has the same priority. It does not establish that the study actually corresponds to that order’s acquisition, or that every newly served order has its required study.

References: [clinical.go:700](/home/george/dt-audit-5c4i/snapshot/internal/eval/clinical.go:700), [clinical.go:330](/home/george/dt-audit-5c4i/snapshot/internal/eval/clinical.go:330), [measure.go:508](/home/george/dt-audit-5c4i/snapshot/internal/eval/measure.go:508), [measure.go:529](/home/george/dt-audit-5c4i/snapshot/internal/eval/measure.go:529).

The independent witness begins with an order wanted before H, acquired at H+1 second, and reported 60 hours past due.

| Record alteration | Record accepted as valid? | Safety/result |
|---|---|---|
| Missing study-order link | No | INVALID |
| Unknown order ID | No | INVALID |
| Ambiguous duplicate order ID | No | INVALID |
| Duplicate the violating study row | **Yes** | Counts **two** violations |
| Relink the violating study to an existing same-priority order wanted after H | **Yes** | **Safety PASS; full GO** |
| Delete the violating study, leaving its original order marked served | **Yes** | **Safety PASS; full GO** |

The relinking witness changes cohort membership without changing the actual acquisition or reporting timestamps. The deletion witness leaves an imaged order without its reporting obligation. `unacquired` does not catch it because that counter considers only open orders.

This is not unavoidable censoring. It is a correctable failure to reconcile orders and studies. Longer follow-up does not repair it.

**Required remediation:** Enforce the applicable order-study cardinality and consistency contract for newly acquired work: unique study identity, unambiguous links, acquisition/resolution consistency, appropriate request attributes, and reconciliation of served orders to studies. Preserve the explicitly supported historical and inherited-record exceptions rather than silently changing older candidate semantics.

Witness: [TestAuditLinks](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go:68). Results: [follow-up log](/home/george/dt-audit-5c4i/scratch/logs/witness-followup.log).

**I3 — The 45-day proposal is reasonable in scope, but remains unimplemented and overstates exact-boundary decidability. Severity: High for readiness; specification limitation at equality.**

The candidate still contains `settlementDays: 31`. `observationNeeded` takes the maximum of the existing closure, settlement and backlog requirements; it has no 45-day clinical requirement.

References: [candidate policy:131](/home/george/dt-audit-5c4i/snapshot/docs/calibration/clinical-5c4b/candidate-policy-v5.json:131), [measure.go:333](/home/george/dt-audit-5c4i/snapshot/internal/eval/measure.go:333), [boundary-safety.md:232](/home/george/dt-audit-5c4i/snapshot/docs/calibration/clinical-5c4h/boundary-safety.md:232).

The owner’s approval resolves the choice of follow-up length. It does not make the current implementation a 45-day implementation.

With synthetic records observed through **E = H + 45 days**, the independent witnesses produced:

| Case | Observed treatment |
|---|---|
| Acquisition at H+1 second; order wanted before H; report 60 hours past due | Observed violation; FAIL |
| Report at H+1 second, acquired earlier, 60 hours past due | Observed violation; FAIL |
| Post-H acquisition; report completed exactly 48 hours past due | PASS |
| Same completion one second later | FAIL |
| Unfinished soon study acquired E−53 hours | Overdue; FAIL |
| Unfinished soon study acquired E−52 hours | Exactly at absolute deadline; INSUFFICIENT |
| Unfinished soon study acquired E−10 hours | Pending; INSUFFICIENT |
| Routine order wanted at H, acquired at H+30 days, completed at E | Exactly at absolute limit; PASS |
| Same routine study unfinished at E | **INSUFFICIENT** |
| Routine acquisition at H+30 days+1 second, unfinished at E | INSUFFICIENT |
| Routine acquisition at H+29 days, unfinished at E | Overdue; FAIL |
| Record asserts completion at E+1 second | INVALID; completion is outside observation |

No tested overdue or pending reporting obligation silently passed.

The duration calculation is correct:

| Priority | Access limit + reporting target + absolute allowance |
|---|---:|
| Urgent | 3 days + 1 hour |
| Soon | 9 days + 4 hours |
| Routine | **45 days** |

However, “every timely acquisition would then be decidable” is not literally true under the retained strict comparator. For an order wanted exactly at H and acquired exactly at its routine access limit, the absolute deadline equals E. An unfinished report at that instant remains INSUFFICIENT.

This is safe treatment of the boundary, but the proposal’s claim needs qualification. Either retain that exact-deadline exception explicitly, or specify observation through a representable instant beyond the deadline. Do not silently turn equality into FAIL or PASS.

For acquisitions delayed beyond their access limit, no fixed follow-up guarantees observation of every eventual reporting deadline. An acquisition near E can remain pending; an order still unacquired has no reporting clock yet. Those are genuine finite-observation limitations. Gate A judges their access, but a small number of untimely orders can remain within its permitted rate.

**Required implementation changes:**

- Encode the approved clinical observation requirement and retain the existing T180, closure and backlog requirements.
- Validate the intended observation claim against **access + reporting target + absolute allowance**, including equality.
- Collect samples, captures and study outcomes through the new endpoint consistently.
- Update scenario construction, protocol wording, evidence requirements and pinned digests.
- Define the interaction with measurements that already use `end`, including access outcomes and financial observation; do not treat the extension as a metadata-only change.
- Preserve overdue FAIL and pending INSUFFICIENT, outside the shared allowance.

Witness: [TestAuditHorizon45](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go:44). Existing unacquired-order and wanted-after-H witnesses also passed.

**I4 — Extending observation changes earlier fidelity-model reporting outcomes. Severity: High. Confirmed construction defect relevant to the approved extension.**

The fidelity generator first consumes booking randomness for the entire observation period. It then uses the same booking stream for documentation and reading draws. Extending observation therefore shifts randomness used for earlier studies.

References: [oc_fidelity_test.go:400](/home/george/dt-audit-5c4i/snapshot/internal/eval/oc_fidelity_test.go:400), [oc_fidelity_test.go:516](/home/george/dt-audit-5c4i/snapshot/internal/eval/oc_fidelity_test.go:516), [oc_fidelity_test.go:618](/home/george/dt-audit-5c4i/snapshot/internal/eval/oc_fidelity_test.go:618).

An independent A85 construction used identical debug demand and booking seeds, changing only an in-memory policy copy’s observation requirement from 31 to 45 days:

- **7,810** studies acquired by H were present in both records.
- **Zero** acquisition timestamps changed.
- **7,781** documentation-entry timestamps changed.
- **7,794** final-report timestamps changed.
- **7,790** changed final-report timestamps were before H in both records.

This is not merely observing previously censored completions. Earlier realized reporting outcomes change.

The operational simulator’s inspected export reads the existing study-order index and draws no randomness: [export_eval_test.go:431](/home/george/dt-audit-5c4i/snapshot/internal/sim/export_eval_test.go:431). The demonstrated defect is in the fidelity generator, not evidence of an operational simulator regression.

**Required remediation:** Make the prospective generator preserve the common simulated prefix when observation is extended. Specify any revised random-stream allocation before lock, retain historical generator compatibility where required, and verify identical earlier decisions and random assignments. Recheck construction under the implemented 45-day policy before considering reserved execution.

Witness: [TestAuditFidelityObservationPrefix](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go:87). Results: [targeted log](/home/george/dt-audit-5c4i/scratch/logs/targeted.log).

**I5 — Reserved adjudication does not enforce the stated group and completeness rules. Severity: High. Confirmed protocol implementation defects; block lock.**

**Remainders enter group-level criteria.** The harness constructs and judges the last 20 seeds of a 2,500-seed scenario as another group. `reservedCriteria` then includes every emitted line in T1 and T7.

References: [reserved_manifest_test.go:630](/home/george/dt-audit-5c4i/snapshot/internal/eval/reserved_manifest_test.go:630), [reserved_manifest_test.go:896](/home/george/dt-audit-5c4i/snapshot/internal/eval/reserved_manifest_test.go:896).

The independent witness supplied 62 complete groups, exactly three rejected, followed by a rejected 20-seed remainder. The adjudicator reported **four rejected out of 63 and T1 FAIL**. The protocol requires **three out of 62 and T1 PASS**, with the remainder excluded from this criterion.

A 20-seed remainder naturally lacks Gate B’s 40-seed requirement. Treating that insufficiency as an additional rejected evaluation effectively tightens the intended T1 threshold.

**Incomplete and invalid evidence can satisfy construction checks.** The result reader does not enforce the exact scenario/group/replicate/seed matrix. T8 checks that a scenario has some output, not all required output, and skips matching checks when `Valid` is false.

References: [reserved_manifest_test.go:1023](/home/george/dt-audit-5c4i/snapshot/internal/eval/reserved_manifest_test.go:1023), [reserved_manifest_test.go:1089](/home/george/dt-audit-5c4i/snapshot/internal/eval/reserved_manifest_test.go:1089), [reserved_manifest_test.go:1227](/home/george/dt-audit-5c4i/snapshot/internal/eval/reserved_manifest_test.go:1227).

Executed witnesses:

- For the final witness scenario, **one passing group instead of the required 25** yielded T4 PASS and T8 PASS.
- Supplying one invalid, empty result line for each scenario yielded **T8 PASS**. Several other criteria also returned PASS. This particular witness did not establish overall confirmation because other criteria remained blocking.
- The reader also omits a scanner-error check after reading results, so input failure is not explicitly distinguished from normal completion.

The truncation defect is consequential: dropping the remaining groups of the last witness can leave its checks favorable without completing the protocol.

**Required remediation:** Separate per-seed remainder evidence from complete-group evidence. Validate exact expected scenario IDs, groups, seed coverage, pair identities, replicate counts, uniqueness, record validity and both sides’ injection counts before adjudication. Missing, duplicate, malformed, truncated or unreadable output must not confirm the policy.

Witnesses: [TestAuditCriteriaRemainder](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go:97), [TestAuditCriteriaMissingInvalid](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go:103), [TestAuditTruncatedWitness](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go:145).

**I6 — T7, T2G and targeted witness checks retain substantive gaps. Severity: Medium–High. Confirmed adjudication/reporting defects; require resolution before lock.**

**T7 loses insufficiency within a failing priority gate.** A severity-window gate becomes FAIL if any window fails, before considering another window’s insufficiency. The “raw” map then records only gates whose final status is INSUFFICIENT.

References: [clinical.go:838](/home/george/dt-audit-5c4i/snapshot/internal/eval/clinical.go:838), [clinical.go:957](/home/george/dt-audit-5c4i/snapshot/internal/eval/clinical.go:957).

The independent valid-record witness had one complete soon block below minimum and another failing its severe rate. Its measures showed:

```text
insufficientWindows = 1
failingWindows = 1
gate status = FAIL
raw insufficient seeds for the soon-window gate = []
```

The existing repair correctly preserves insufficiency when a *different gate* fails, but not when another window within the same gate fails.

**Required remediation:** Derive raw window evaluability from the underlying window measures, independently of failure precedence, and retain both facts.

**T2G pools soon and routine severity before deciding materiality.** The written criterion refers to a severe soon **or** routine share of at least 1%. The code instead combines their numerators and denominators.

Reference: [reserved_manifest_test.go:989](/home/george/dt-audit-5c4i/snapshot/internal/eval/reserved_manifest_test.go:989).

A fabricated 25-group witness had soon severity **2/100 = 2%**, routine severity **0/10,000**, no other materiality trigger and no rejected groups. The adjudicator diluted severity to **0.0198%**, classified the scenario non-material and returned T2G PASS.

**Required remediation:** Specify and implement priority-specific materiality, or explicitly reopen the criterion if combined-priority dilution is intended. The current text and implementation must not remain inconsistent.

**T4 does not always verify the targeted clinical mechanism.** For W-PARTIAL and W-ACC-LATE, `witnessMet` accepts an allowance rejection with every seed failing. It does not require the failures to be the partial-window ceiling or timely-access gate.

Reference: [reserved_manifest_test.go:1126](/home/george/dt-audit-5c4i/snapshot/internal/eval/reserved_manifest_test.go:1126).

A fabricated group with only an unrelated allowance failure satisfied both witness predicates.

The manifest also describes W-ID-CHANGED as changing wanted time, while the implementation changes modality: [reserved_manifest_test.go:344](/home/george/dt-audit-5c4i/snapshot/internal/eval/reserved_manifest_test.go:344).

**Required remediation:** Check the named gate and affected seeds directly, and make every manifest mechanism match its implementation.

Witnesses: [TestAuditRawWindowInsufficiency](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go:118), [TestAuditCriteriaMaterialityAndAttribution](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go:110).

**I7 — T5’s approved thresholds are coherent, but its prediction and reporting are not ready for lock. Severity: Medium. Statistical/conformance implementation limitation.**

For 40 equal-volume seeds, all initially served, independent urgent loss injection gives:

\[
P(\text{Gate B PASS})
=
P\{\operatorname{Binomial}(40N,\delta)\le 0.02(40N)\}.
\]

Independent calculations give:

| Urgent orders per seed | +2.0 points | +2.1 points | +3.0 points |
|---:|---:|---:|---:|
| 120 | 0.527110 | 0.337348 | 0.000010426 |
| 150 | 0.524254 | **0.314088** | 0.000000913 |
| 180 | 0.522145 | 0.294079 | 0.000000081 |

Thus **0.40 at +2.1 points** is compatible with the intended near-margin behavior, while **0.01 at +3.0 points** is substantially above the designed acceptance probability but still distinguishes appreciably looser behavior.

The approved exact sample boundaries were independently tested:

- **200/500** at +2.1 and **5/500** at +3.0: PASS.
- **201/500** at +2.1: FAIL.
- **6/500** at +3.0: FAIL.

These are empirical point-estimate criteria. They are not confidence guarantees that population acceptance probabilities lie below 0.40 or 0.01. The 500 evaluations use **25 underlying groups with 20 injection replicates each**; treating all 500 as independent would overstate evidence.

For illustration only, under 500 independent evaluations an acceptance probability of 0.50 has probability approximately **0.00000447** of meeting the 0.40 threshold. At the 0.01 threshold, a true acceptance probability of 0.02 still has approximately **6.52%** probability of producing at most five acceptances. Actual protocol uncertainty must respect its clustered design.

The prediction implementation is not the stated exact binomial construction for every permitted volume. `predictAcceptance` calls a helper that substitutes a rounded, clipped normal draw whenever served count exceeds 200.

References: [reserved_manifest_test.go:1151](/home/george/dt-audit-5c4i/snapshot/internal/eval/reserved_manifest_test.go:1151), [oc_test.go:236](/home/george/dt-audit-5c4i/snapshot/internal/eval/oc_test.go:236).

For \(n=201,p=0.021\), exact binomial probability of zero injections is approximately **0.01404**; the implemented approximation gives approximately **0.03359**. The manifest imposes no 200-order ceiling. The actual injection uses individual Bernoulli draws, so prediction and construction can use different distributions.

Additional reporting limitations:

- The protocol’s claim of less than \(10^{-6}\) acceptance at +3 points for \(N=120\) is incorrect; the value is approximately \(1.04\times10^{-5}\). This does not threaten the approved 0.01 tolerance.
- The criteria output does not produce all promised Clopper–Pearson, one-sided and Bonferroni summaries.
- Reusing demand seeds across scenarios means pooled T2/T6 observations are not automatically independent binomial trials. Their uncertainty reporting needs an explicit dependence qualification or appropriate treatment.
- Development results from five groups do not establish the reserved operating characteristics.
- The owner approvals are resolved externally, but the manifest still contains null tolerances and the current criteria entry point supplies an empty tolerance object.

**Required remediation:** Use the stated injection distribution in the prediction, complete and qualify uncertainty reporting, and pin the approved values consistently. No population-confidence requirement should be imposed on Gate B by reinterpretation.

Evidence: [approved-boundary witness](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go:153), [independent probability calculations](/home/george/dt-audit-5c4i/scratch/logs/probabilities.txt).

**Clinical denominator assessment**

| Component | Cohort and behavior | Assessment |
|---|---|---|
| Per-seed severe rates | Acquisitions within the configured evaluated windows | Correct specified acquisition-based arithmetic; permits dilution and excludes post-H acquisitions. |
| Local severity windows | Fixed acquisition windows; soon four weeks, routine one week | No outcome-dependent pooling found. Small partial cohorts remain materially underprotected. |
| Absolute reporting limits | Studies acquired by H, plus post-H studies linked to orders wanted by H | Appropriate union in principle; linkage defects undermine completeness. |
| Gate A | Eligible wanted-time cohort; served/open/late-patient-ended outcomes judged against access deadlines | Correct tested boundaries; conditional timeliness rather than absolute access adequacy. |
| Gate B | Matched new demand wanted in the gated period; terminal lost, left and patient-ended outcomes | Correct equal-seed empirical comparison, distinct from Gate A and reporting severity. |
| Shared allowance | Union of designated clinical failures/insufficiencies by seed | Overlapping failures count once. Absolute safety and Gate B mean remain outside the allowance. |
| Case selection | Separate baseline-relative safeguards and allowances | Eight distinct adverse seeds across separate allowance systems remain possible, as disclosed. |

A further independent horizon witness confirms the retained severity limitation. Moving one moderately severe soon acquisition from H to H+1 second changed the gated cohort from **1 severe / 900 acquisitions** to **0 / 899**. Absolute safety retained the study but passed it because its 30-hour past-due delay was below 48 hours. Both evaluations returned GO.

That behavior follows the written cohort definitions. The horizon revision extends the absolute maximum; it does **not** extend severe-rate protection to every order wanted by H.

Access boundary witnesses confirmed:

- Service exactly at the access deadline is timely; one second later is untimely.
- An open order at the deadline is undecided; one second later it is untimely.
- Patient withdrawal before the deadline leaves Gate A’s denominator; withdrawal exactly at or after it is untimely.
- Lost and left outcomes are counted separately for Gate B.
- Patient-ended outcomes enter Gate B even when withdrawn from Gate A.

An outcome affecting both gates is not automatically double counting: the gates answer different questions, and the shared allowance unions seed failures. The confirmed duplicate-study acceptance in I2 is actual duplicate clinical counting.

**Complete reserved-manifest assessment**

All **49 scenarios**, indices 0–48, were reviewed.

| Family | Scenarios |
|---|---|
| Stable, 4 | S-A85, S-B85, S-C85, S-Cs85 |
| Stress, 4 | S-A92, S-B92, S-C92, S-Cs92 |
| Structural, 3 | Wg85, WgU85, R085 |
| Deterioration, 11 | L10-85, L20-85, D-85, K-85, IL10-85, IK-85, IL40-85, L10-92, K-92, IL10-92, IK-92 |
| Concentrated/offset harm, 7 | H10, H15, H20, SHOCK, OH10, OH15, OH20 |
| Near margin, 5 | NM0, NM1, NM2, NM2.1, NM3 |
| Unequal volume, 1 | HV |
| Witnesses, 14 | W-URG-LATE, W-ROUT-348, W-ACC-LATE, W-LOST30, W-PAT30, W-ID-REMOVED, W-ID-REPEATED, W-ID-CHANGED, W-PARTIAL, W-HORIZON, W-PENDING, W-MARGIN-ABOVE, W-MARGIN-AT, W-ACCESS65 |

The repaired construction has substantive strengths:

- Candidate and baseline use matching exogenous demand models; seasonal practices pair with themselves.
- HV changes volume across seeds while sharing that multiplier within each pair.
- Booking and injection role namespaces are disjoint under the stated index bounds. Demand sharing, identical-booking witnesses and T5 base-pair reuse are explicit exceptions to independent construction.
- Quotas use ceiling counts; random selection is without replacement. Bernoulli injections use \(\delta N/\text{served}\), giving the stated expected injected increase when feasible.
- Infeasible injections are reported instead of silently reduced.
- Loss injections remove corresponding studies and rebuild samples/captures.
- Offset scenarios create baseline loss before testing improvement and concentrated harm.
- The two-seed debug construction check passed for every scenario. This establishes those constructions, not universal feasibility or operating characteristics.

References: [reserved_manifest_test.go:164](/home/george/dt-audit-5c4i/snapshot/internal/eval/reserved_manifest_test.go:164), [reserved_manifest_test.go:240](/home/george/dt-audit-5c4i/snapshot/internal/eval/reserved_manifest_test.go:240), [manifest stream map](/home/george/dt-audit-5c4i/snapshot/docs/calibration/clinical-5c4h/scenario-manifest.json:9).

The prospective decision rules have the following disposition:

| Criterion | Assessment |
|---|---|
| T1 | Written ≤3/62 rule is clear; remainder implementation is wrong. |
| T2 | Detection ≥0.95 with at least 200 harmed seeds is distinct from group rejection. Tested 190/200 PASS, 189/200 FAIL, fewer than 200 INCONCLUSIVE. |
| T2-conformance | Flagged harmed seeds must enter the allowance; more than four adverse seeds must reject. Targeted tests passed. |
| T2G | Written ≥23/25 material-scenario rejection rule is clear; severity materiality is incorrectly pooled across priorities. |
| T3 | Zero observed/overdue violations is clear and tested; complete valid seed coverage and promised bounds must be enforced. |
| T4 | Complete 25-group coverage and named-mechanism attribution are not fully enforced. |
| T5a | Cluster-based conformance rule is specified; prediction distribution needs correction. |
| T5b | Approved 0.40/0.01 comparisons work when supplied; current artifacts have not pinned them. |
| T6 | Fixed design-rate threshold of at most 19 flags in 10,000 is correctly encoded; coverage and dependence qualifications remain necessary. |
| T7 | Written ≤3/62 rule is clear; remainder inclusion and masked within-gate insufficiency are defects. |
| T8 | Does not adequately enforce completeness, validity, uniqueness or all injection evidence. |

T2 correctly no longer promises 90% whole-evaluation rejection for independent 10% concentrated shocks. With perfect flagging and no other failures, rejection through the allowance occurs with probability approximately **0.371**, because five or more harmed seeds are required. Detection and rejection must remain separate in reporting.

The written mandatory consequences are appropriate: every required criterion must PASS; FAIL or INCONCLUSIVE means NO GO; an aborted run consumes the family; post-result waivers are deviations; retuning requires a separately authorized family. The incomplete-result acceptance in I5 prevents those written consequences from being reliably enforced.

The 45-day requirement is **not yet applied consistently** across policy, generated observations, construction evidence and protocol. The current development artifacts describe the existing observation rule and cannot be relabeled as 45-day evidence.

**Historical compatibility**

The targeted checks established:

- **v4 evaluation neutrality:** the fixed evaluation still matches its recorded byte digest, `eef98fab6cd6355eb854b86675f914ecc8be02ba6cfcd5bd83f607dd98f0b070`.
- **5C.4d/e/f compatibility:** preserved candidate bytes load and retain their respective tested rules.
- **Historical artifacts:** the recorded reference-file digest checks pass.
- **Replay neutrality:** dev.15 through dev.20 fixtures replay exactly, including the tested chunked replays.
- **Operational export:** adding the study-order link reads existing state; no new random draw was found.
- **Reserved protection:** the present runner refuses the reserved family before scenario construction.
- **Policy status:** v4 remains in force; v5 and the reserved protocol remain proposals.

These checks support compatibility within their coverage. They do not prove every possible historical evaluation byte-identical, nor establish future 45-day prefix neutrality. The fidelity counterexample in I4 must remain separate from the successful historical checks.

Reproduction materials: [witness source](/home/george/dt-audit-5c4i/scratch/witness/audit_5c4i_test.go), [overlay](/home/george/dt-audit-5c4i/scratch/overlay.json), [test wrapper](/home/george/dt-audit-5c4i/scratch/run-go.sh), [targeted test selection](/home/george/dt-audit-5c4i/scratch/run-targeted.sh), [passed-test inventory](/home/george/dt-audit-5c4i/scratch/logs/passed-tests.json), [replay results](/home/george/dt-audit-5c4i/scratch/logs/replay.log).

**Exact blockers to prospective lock**

1. Resolve the clinical adequacy of allowing **5/6 severe acquisitions with full GO**; the >5 count ceiling does not establish small-cohort safety.
2. Close the demonstrated order-study integrity bypasses and duplicate counting.
3. Implement and consistently specify the approved 45-day follow-up, including exact-deadline and late-acquisition censoring.
4. Make observation extension preserve earlier fidelity-model decisions and random assignments.
5. Repair reserved adjudication: remainder exclusion, complete-result validation, raw T7 accounting, T2G materiality and targeted T4 attribution.
6. Correct T5’s prediction distribution, complete its reporting, and pin the already-approved tolerances and revised artifacts after review.

**NO GO. Passing implementation tests does not justify spending the reserved stream.**