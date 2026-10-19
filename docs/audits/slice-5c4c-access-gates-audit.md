# Slice 5C.4c clinical access-gates audit (Astra High)

**Audited revision:** commit a52fc20 (0.2.0-dev.17), the candidate v0.2-eval-5 clinical acceptance framework (not frozen, not in force), 2026-10-18. Line references are to a52fc20.

The audit was independent, adversarial and read-only. The auditor read a `git archive` snapshot of a52fc20 on a separate machine with no Go toolchain. It ran only a filtered selection of existing tests from a test binary compiled from that commit (section 16). It ran no strategy, reference seed, holdout seed, reserved stream PCG(20261201, ·) or unseal procedure.

* **Everything below the rule is the auditor's report as delivered, unchanged.** The disposable witness files it lists in section 16 were kept outside the repository; their inputs and results are reproduced in the report itself.
* **No remediation is in this commit.** Maintainer checks, if any, will follow in a separate appendix or document.

---

# Independent audit of Slice 5C.4c

**Commit:** a52fc20  
**Version:** 0.2.0-dev.17  
**Date:** 2026-10-18  
**Policy under review:** proposed v0.2-eval-5 clinical acceptance framework

## 1. Executive judgment

**NO GO for reserved-stream confirmation as currently specified. NO GO for policy freeze.**

Gate B’s claimed 95% upper confidence bound is mathematically invalid for the population increase in unimaged demand. It can certify deterioration beyond the two-percentage-point margin with probabilities exceeding 50%, even under independent observations. Duplicate demand keys and within-seed dependence create additional inferential problems.

Two implementation defects independently block confirmation:

1. Gate B reads the last capture in the record array instead of the validated, named end capture.
2. Missing or unknown terminal outcomes can escape integrity rejection and permit favorable evaluation results.

The access design can support a deliberately limited question about intervention non-deterioration. It cannot establish adequate absolute patient access. That limitation must appear explicitly in favorable verdicts. Moreover, the current statistical defect means that even the intended population non-deterioration claim is not yet established.

The soon weekly failures contain genuine reporting delays. They are not sufficient evidence that the denominator is wrong or that the rule should be relaxed. The unresolved question is whether the strict maximum over pooled weekly blocks expresses the intended clinical standard and has acceptable operating characteristics.

| Required judgment | Verdict | Reason |
|---|---|---|
| **A. Statistical validity of Gate B** | **NO GO** | Invalid population confidence bound; unresolved pairing and clustering assumptions. |
| **B. Clinical validity of access gates** | **NO GO as an adequate-access certification** | High absolute loss can coexist with passing gates. Qualified non-deterioration acceptance remains possible after remediation. |
| **C. Weekly severity rule** | **NO GO for freezing the present interpretation** | Meaningful delays are detected, but the ceiling, pooling effects and rejection behavior need explicit approval and calibration. |
| **D. Full v5 evaluator implementation** | **NO GO** | Capture-selection and outcome-validation defects. |
| **E. Running reserved confirmatory streams** | **NO GO** | Repair the statistical and implementation defects, then recommit a complete prospective protocol. |
| **F. Freezing v0.2-eval-5** | **NO GO** | Statistical, implementation and clinical decisions remain unresolved. |

## 2. Scope, methods and execution limits

The review covered the package and every directly referenced file:

- `docs/calibration/clinical-5c4c/astra-review-package.md`
- `docs/calibration/clinical-5c4b/candidate-policy-v5.json`
- `internal/eval/clinical.go`, `lostdemand.go`, `compare.go`, `evaluate.go`, and `policy.go`
- `docs/calibration/clinical-5c4c/access-gates.md` and `reserved-validation-protocol.md`
- The 5C.4b acceptance design, order lifecycle, access denominators, fidelity findings and fidelity protocol
- Both committed no-action access JSON files
- The referenced evaluator clinical, witness and gate tests
- `internal/sim/demand_pairing_test.go` and `clinical_access_test.go`

Supporting source inspection covered evaluator integrity, records, aggregation, arithmetic and statistical functions, and simulator demand generation, booking, recall, cancellation, no-show, walk-in, acquisition, reporting and record export.

Evidence is distinguished throughout:

- **Executed test:** an existing, permitted test in the supplied binary.
- **Arithmetic:** calculation from explicitly stated inputs.
- **Code reasoning:** a deterministic consequence of inspected code; not a newly executed Go test or strategy.

**Execution result:** 44 selected evaluator tests passed. These covered clinical gates and witnesses, candidate validation, v4 neutrality, and fidelity unit checks. The simulator pairing and access tests were inspected as source only.

The snapshot was not edited. No fixes were implemented. No strategy, reference seed, holdout seed, reserved stream, unseal procedure or network access was used. No `DT_*` environment variable was set.

Passing tests are treated as evidence of tested behavior, not proof that the statistical or clinical specification is correct.

## 3. Findings ranked by severity

| ID | Severity | Finding | Blocks confirmation? |
|---|---|---|---|
| F1 | **Critical** | Gate B’s upper bound lacks its claimed 95% population coverage. | **Yes** |
| F2 | **High** | Valid capture-array permutations change Gate B’s observation endpoint. | **Yes** |
| F3 | **High** | Empty or unknown terminal outcomes can remain valid and pass acceptance. | **Yes** |
| F4 | **High** | Duplicate keys suppress discordance; pooled inference ignores seed dependence. | **Yes** |
| F5 | **High** | T1–T5 omit adequate near-margin false-acceptance and coverage assessment. | **Yes** |
| F6 | **High clinical limitation** | Favorable verdicts do not enforce the distinction between non-deterioration and adequate access. | **Yes, until the claim is explicitly limited or an absolute requirement is approved** |
| F7 | **Medium; owner decision required** | Weekly severity has strong sampling and partition effects without a settled clinical interpretation. | **Yes for freezing the current rule; resolve its meaning before confirmation** |
| F8 | **Medium** | Clinical allowance validation accepts fractions above one. | **Not independently for the pinned candidate’s 0.1 value** |
| F9 | **Low** | Sparse-week duration and unmatched-demand reporting can mislead readers. | **Not independently; correct or qualify the reporting** |

## 4. Gate B: independent statistical derivation

### F1 — Conditional directional inference is not a population risk-difference bound

**References:** `internal/eval/lostdemand.go:175–178`; `internal/eval/stats.go:304–321`; `docs/calibration/clinical-5c4c/access-gates.md:59–75`; `internal/eval/clinical_gates_test.go:81–99`.

**Requirement:** the reported one-sided upper bound must support the stated 95% confidence claim for increased unimaged demand.

For an individually identified paired order, define:

\[
p_{10}=P(\text{candidate unimaged, baseline imaged}),
\]

\[
p_{01}=P(\text{candidate imaged, baseline unimaged}).
\]

Write:

\[
q=p_{10}+p_{01},\qquad
\pi=\frac{p_{10}}{p_{10}+p_{01}}.
\]

The population risk difference is:

\[
\Delta=p_{10}-p_{01}=q(2\pi-1).
\]

Under independent, identically distributed paired observations, conditional on observing \(D=d\) discordant pairs, the candidate-only count satisfies:

\[
B\mid D=d\sim\operatorname{Binomial}(d,\pi).
\]

The Clopper–Pearson construction in `binomUpper` supplies an upper bound \(\pi^+\) for this conditional directional probability. For \(b<d\), it solves:

\[
P_{\pi^+}\{X\le b\}=0.05.
\]

For \(b=d\), it returns one.

Gate B computes:

\[
U=\frac{D}{N}(2\pi^+-1).
\]

The defect is the substitution of the observed, random \(D/N\) for the unknown discordance probability \(q\), without accounting for its uncertainty. Conditional coverage for \(\pi\) does not establish unconditional coverage for \(\Delta\).

A conditional statement about a target involving the observed \(D/N\) would be a different, data-dependent claim. It would not establish the population deterioration claimed here.

### Witness F1.1 — One-directional discordance

**Type:** exact probability arithmetic.

**Inputs:**

- Unique demand keys.
- Baseline images every order.
- Candidate independently loses each order with probability \(p\).
- \(N\) matched eligible orders.
- Confidence .95; margin .02.

Then \(C=0\) and \(B=D=X\), where:

\[
X\sim\operatorname{Binomial}(N,p).
\]

For every positive \(X\), `binomUpper(X,X,.95)` is one. For \(X=0\), the code leaves `IncreaseUpper` at zero. Therefore, for every observation:

\[
U=X/N.
\]

The purported upper confidence bound is simply the observed loss fraction.

**Reproduction:** calculate

\[
P(\text{PASS})=
\sum_{x=0}^{\lfloor .02N\rfloor}
{N\choose x}p^x(1-p)^{N-x}.
\]

| Matched orders | True increase | Probability Gate B passes | Actual bound coverage |
|---:|---:|---:|---:|
| 200 | 2.1 percentage points | **58.9419%** | **41.0581%** |
| 400 | 2.1 percentage points | **53.6309%** | **46.3691%** |
| 200 | 3 percentage points | **28.0980%** | — |
| 400 | 3 percentage points | **15.1053%** | — |
| 6,000 | 2.001 percentage points | **52.2049%** | — |
| 6,000 | 2.1 percentage points | **31.4088%** | — |

The first two sample sizes meet the proposed Gate B minimums. These examples isolate Gate B; other gates may be insufficient at low per-seed volumes. The 6,000-order example also matches the existing fixture’s scale of 40 seeds with 150 urgent orders each.

**Expected:** a valid one-sided 95% population bound should have at least its claimed coverage under its assumptions. Certifying an above-margin population increase should not occur at these probabilities.

**Actual:** substantial undercoverage occurs even before introducing clustering, duplicate keys or simulator complexity.

**Consequence:** the central statistical acceptance claim is false. This blocks confirmation.

### Witness F1.2 — Zero discordance does not establish zero population increase

**Type:** exact arithmetic.

Set \(N=200\) and \(p=.005\) in the same model.

\[
P(D=0)=.995^{200}=0.3669578.
\]

On that event the reported upper bound is zero, although the population increase is positive.

Zero observed discordance is not evidence sufficient to set the population upper bound to zero. The minimum cohort requirement does not fix this.

### Exact margin and numerical comparison

The candidate sets an inclusive two-percentage-point margin. The existing test requires three additional losses among 150 urgent orders per seed to pass with upper bound exactly .02.

That test confirms the implementation’s one-directional behavior; it also locks in the defect.

The comparator in `lostdemand.go:199–201` allows a numerical tolerance of \(10^{-12}\). This is negligible compared with the inferential error and is not the reason for rejection of the method.

The documentation’s statement that all one-directional discordance yields the observed increase is also too broad. It holds for candidate-only losses. For baseline-only discordance, the directional upper bound generally differs from the observed negative increase.

### Required statistical remediation

Before any reserved run:

1. Specify the population estimand and independent sampling unit.
2. Replace this construction with a justified paired risk-difference procedure.
3. Address seed dependence separately from patient pairing.
4. Establish coverage and false acceptance at, below and immediately above the margin.
5. Include zero, one-directional and bidirectional discordance.
6. If the intended rule is only an empirical observed-difference rule, remove the population-confidence claim and approve that weaker interpretation explicitly.

An unconditional paired-multinomial procedure could address independent paired observations. It would not, by itself, solve seed clustering.

## 5. Pairing validity

### Demand is substantially exogenous under the current supported action model

**References:** `internal/sim/demand.go:22–28`, `:471`, `:512`; `internal/sim/flow.go:296–350`; `internal/sim/config.go:466`; `internal/sim/export_eval_test.go:402–411`.

Direct source inspection supports these conclusions:

- Referral counts, receipt times and demand attributes use seed- and origin-keyed draws.
- Booking changes do not consume a shared demand draw sequence.
- Rebooking and recall operate on an existing order.
- Original wanted times and priority are not reset by those operations.
- Walk-in policy acts after disclosure and changes disposition rather than the existence of that request.
- Walk-in generation depends on opening hours. In the inspected action model, those hours come from configuration; staffing and operational changes do not provide an opening-hours action.

Exogeneity is therefore credible for the current fixed scenario and supported actions. It is not a universal property of any future action or demand model.

### The existing exogeneity test is useful but insufficient

**Reference:** `internal/sim/demand_pairing_test.go:20–64`.

The test compares two seeds and selected action combinations: teleradiology, walk-in scheduling policy and worklist policy. It demonstrates that those examples preserve the demand multiset while changing outcomes.

It does not establish:

- invariance under every action family;
- invariance under a future hours-changing action;
- individual identity inside duplicate-key groups;
- correct statistical treatment of paired outcomes.

The documentation’s description of “one seed” also understates the actual two-seed test.

### F4 — Matching a demand multiset is not matching individual patients

**References:** `internal/eval/lostdemand.go:72–78`, `:134–154`; `internal/sim/demand.go:22–28`; `internal/sim/export_eval_test.go:409–411`.

Priority is accounted for by the surrounding priority loop. The remaining key consists of receipt time, wanted time, modality and walk-in status.

Those attributes can coincide for different patients. They omit the simulator’s actual `originKey`: day, source, hour block and index. Thus equality establishes identical recorded demand populations, not necessarily individual identity.

Within each duplicate-key group, the implementation retains only the net excess of losses. That selects the minimum possible discordance, equivalent to assuming maximal concordance.

#### Witness F4.1 — Opposing outcomes disappear inside ties

**Type:** code reasoning and arithmetic.

**Inputs:**

- 200 urgent orders.
- Twenty duplicate-key groups containing two patients each.
- In each group, the candidate loses patient 1 and images patient 2.
- The baseline images patient 1 and loses patient 2.
- The other 160 orders are imaged under both.

**Reproduction:** apply the group-count subtraction at `lostdemand.go:145–154`.

**Actual:** every group has one loss under each plan. The implementation records \(B=C=0\), reports \(U=0\), and passes.

**Expected:** individual discordance is \(B=C=20\). If identity is unavailable, uncertainty about pairing must be preserved rather than silently resolved in the most favorable direction.

For comparison, applying the current directional formula to the known identities gives:

\[
\pi^+=0.63890834,\qquad
U=(2\pi^+-1)\frac{40}{200}=0.05556334,
\]

which fails. This comparison demonstrates suppression of discordance; it does not endorse the current formula.

**Consequence:** aggregate fractions remain correct, but inferential uncertainty does not. This blocks the claimed paired inference.

**Remediation:** provide stable exogenous identity, or establish a conservative method for unidentified groups. If neither is available, mark the comparison insufficient. Report collision counts.

### F4 — Pooling orders does not remove seed dependence

**References:** `internal/eval/lostdemand.go:128–177`.

Orders within a seed share capacity, staffing, calendar conditions and queues. Demand matching does not make their outcomes independent.

#### Witness F4.2 — Correlated capacity shocks

**Type:** exact arithmetic.

**Inputs:**

- 40 independent seeds.
- 200 matched urgent orders per seed.
- Baseline images every order.
- Independently per seed, probability .03 of a candidate shock that loses all 200 orders; otherwise none are lost.

The population order-weighted increase is three percentage points.

One shocked seed gives an observed increase of \(200/8000=.025\), so the current rule fails. With no shocks it passes. Therefore:

\[
P(\text{PASS})=.97^{40}=0.2957123.
\]

**Expected:** inference must recognize 40 independent shock opportunities, rather than treating 8,000 orders as independent replicates.

**Actual:** the pooled calculation provides no seed-cluster adjustment.

**Consequence:** an independent-order statistical repair would still be insufficient.

The owner must specify whether the target is order-weighted deterioration or average seed-level deterioration. These differ when demand volumes vary. Worst-seed increases are useful diagnostics, but reporting them does not constrain them.

### Missing and changed demand

The current unmatched-demand rule is appropriately conservative:

- Extra or missing eligible requests cause insufficient evidence.
- Changed priority generally changes both affected priority cohorts.
- Changed receipt or wanted times generally destroy matching.
- Wanted-after-horizon orders are deliberately outside the eligible population.

The existing tests cover several such mismatches. Balanced exchanges between indistinguishable requests remain undetectable as individual changes.

Requiring all priorities to pass valid noninferiority tests is an intersection-union decision. It does not automatically require Bonferroni correction for false overall certification. Simultaneous confidence statements and Monte Carlo interval families are separate matters.

## 6. Implementation and record integrity

### F2 — Gate B uses array position instead of the validated endpoint

**References:** `internal/eval/lostdemand.go:91`; `internal/eval/measure.go:451–482`, `:530–559`; `internal/eval/clinical.go:165`.

**Requirement:** measurements must use the prescribed observation endpoint, independent of irrelevant serialization order.

#### Witness F2.1 — Capture permutation

**Type:** code reasoning; not a newly executed Go test.

**Inputs and reproduction:**

1. Start with valid paired records containing 150 eligible urgent orders per seed.
2. Make 30 orders received after T30 terminally lost only under the candidate.
3. Preserve every named capture, timestamp, money value and outcome vector.
4. Permute both capture arrays so T30 is last.

**Expected:** the evaluation is unchanged, or the ordering is explicitly rejected.

**Actual code behavior:**

- `checkCaptures` validates captures by name and timestamp without imposing array order.
- `checkOrders` traverses named captures.
- Gate A uses the named end capture.
- Gate B uses `r.Captures[len(r.Captures)-1]`.
- The selected orders are `not_disclosed` at T30 and therefore do not count as unimaged.

Gate B changes from a .20 increase and failure to zero increase and pass.

This is a Gate B bypass witness. Independent case-selection gates can still reject the large-loss example; no complete-GO claim is made for that particular construction.

**Consequence:** valid representation changes corrupt a clinical measurement. This blocks confirmation.

**Remediation:** obtain the unique validated `end` capture at `ObservedUntilSec`; add permutation-invariance and wrong-last-capture tests.

### F3 — Missing terminal outcomes can escape integrity rejection

**References:** `internal/eval/measure.go:530–559`, especially `:552`; `internal/eval/clinical.go:390–404`; `internal/eval/lostdemand.go:57–69`; `internal/eval/measure.go:1132–1162`.

**Requirement:** missing or invalid evidence must not support PASS or a favorable verdict.

For resolved orders, `checkOrders` accepts any outcome other than `open` or `not_disclosed`, provided it agrees with the end outcome. It does not validate membership in the terminal-outcome enumeration.

#### Witness F3.1 — Empty outcome

**Type:** code reasoning.

**Inputs and reproduction:**

1. Start with otherwise passing paired records.
2. Select one candidate urgent order wanted on day 60.
3. Preserve its valid receipt, wanted and resolution times and its study.
4. Replace `served` with `""` in every capture at or after resolution.
5. Preserve correct earlier `not_disclosed` or `open` outcomes.

**Expected:** INVALID because terminal evidence is missing.

**Actual code behavior:**

- Lifecycle consistency accepts the empty string.
- The additional clinical timestamp check applies only to `served`.
- Gate A treats the empty outcome as untimely through its default branch.
- One such order among 150 leaves timeliness at \(149/150=99.33\%\), passing.
- Gate B ignores the outcome through `Unimaged.add`’s default branch.
- The small served-count change can remain inside case-selection tolerances.
- Study samples and finances remain unchanged. Reconciliation does not link orders to studies.

There is no integrity rejection. With otherwise passing clinical and financial evidence, a favorable full verdict remains possible.

This is a malformed-record witness, not an observed simulator-generated record.

**Consequence:** the “missing or contradictory evidence is never accepted” claim is false. This blocks confirmation.

**Remediation:** validate outcome enums before measurement and distinguish unknown evidence from a known adverse outcome. Add missing and unknown enum witnesses.

An unknown eligible order priority has a related failure mode: order validation does not reject it before clinical measurement dereferences the priority-indexed cohort. That can panic instead of returning INVALID.

### F8 — Shared-allowance fractions above one validate

**References:** `internal/eval/policy.go:483`; `internal/eval/exact.go:18–23`; `internal/eval/evaluate.go:651–676`.

**Type:** code reasoning.

**Input:** change only `clinical.maxFailingSeedFraction` to `1.01`.

**Expected:** policy validation rejects a fraction greater than one.

**Actual:** `wholeBP` checks nonnegativity and whole basis points, but not an upper bound. For 40 seeds, the resulting allowance is 40 failures.

**Consequence:** the schema validator does not enforce the intended fraction domain. Absolute limits and Gate B remain separate.

This does not change the current candidate’s .1 allowance and is not an independent blocker for that pinned value. It should be corrected before claiming general policy-validation completeness.

## 7. Access certification and cancellation/loss gaming

### F6 — The gates cannot establish adequate absolute access

**References:** `internal/eval/clinical.go:264–270`, `:448–458`; `internal/eval/compare.go:462–519`; `docs/calibration/clinical-5c4c/access-gates.md:87–91`.

Gate A excludes lost and left demand and patient endings before the deadline. Gate B constrains deterioration relative to baseline, allowing a proposed two-point margin. Neither imposes an absolute access floor.

#### Witness F6.1 — Catastrophic access preserved

**Type:** code reasoning.

**Inputs:**

- Candidate and baseline lose the same 65% of urgent demand.
- Both image the remaining 35% within 48 hours.
- Cohorts are sufficiently large.
- Reporting, backlog and other gates pass.

**Actual:**

- Gate A: 100%.
- Gate B: no discordance, current bound zero, PASS.
- Absolute urgent loss: 65%.
- Qualifying financial evidence can produce a favorable overall verdict.

**Expected:** this may support an explicitly limited intervention-acceptance decision, but must not be described as adequate patient access.

The actual no-action practice fails other clinical gates. This witness does **not** claim that the committed no-action records receive a full GO. It demonstrates that poor absolute access is not itself disqualifying.

Repairing Gate B’s interval does not resolve this distinction: with sufficient evidence, unchanged high loss can satisfy a valid non-deterioration test.

### Committed no-action evidence

**Sources:** `docs/calibration/clinical-5c4c/sim-access-5c4c.json`, priority access entries across seeds 901–910; `access-gates.md:149–173`.

| Priority | Gate A timely fraction | Timely / all eligible demand |
|---|---:|---:|
| Urgent | 1461 / 1471 = **99.32%** | 1461 / 4591 = **31.82%** |
| Soon | 8553 / 8865 = **96.48%** | 8553 / 15491 = **55.21%** |
| Routine | 57342 / 58574 = **97.90%** | 57342 / 74357 = **77.12%** |

These are different questions, not interchangeable measures.

The quoted 65%, 38% and 14% losses count lost plus left orders. They are not the complete Gate B terminal-unimaged fractions, which also include patient-ended outcomes.

**Required owner decision:**

- Approve a clinically justified absolute-access requirement; or
- Explicitly authorize acceptance limited to baseline-relative intervention effects.

If choosing the latter, every favorable verdict should state:

> **Access non-deterioration accepted under the specified margin; adequate absolute patient access has not been established.**

That statement becomes appropriate only after the statistical defects are repaired. At present, the population non-deterioration claim is also unsupported.

Disclosure elsewhere in the report is insufficient. The current verdict code adds Gate B reasons on failure but does not require the limitation on success.

### Classification changes can improve the gates without additional imaging

**References:** `internal/eval/clinical.go:264–270`, `:359–391`; `internal/eval/lostdemand.go:57–69`.

Counting patient outcomes in Gate B correctly prevents a simple large-loss escape by renaming losses as cancellations. Nevertheless, the partition between the gates remains category-sensitive.

#### Witness F6.2 — Closing overdue open orders as lost

**Type:** arithmetic and code reasoning.

**Inputs:** 100 urgent orders per seed: 94 timely served and six open past 48 hours.

Initially, Gate A is 94%, failing. End two open orders as lost:

- Imaging remains 94.
- Gate A becomes \(94/98=95.92\%\), passing.
- Against the original baseline, terminal unimaged demand increases by exactly two points.
- The current one-directional Gate B passes at that margin.

A corrected inferential procedure may reject that finite sample at the observed boundary. The clinical partition still permits conditional timeliness to improve when additional demand becomes terminally lost.

#### Witness F6.3 — Both reported gate measures improve

**Type:** arithmetic and record-level code reasoning; not an executable strategy.

**Inputs per seed:**

- 100 timely served;
- ten patient endings at or after the access deadline;
- 90 lost.

Initial Gate A is \(100/110=90.91\%\). Terminal unimaged demand is 50%.

Reclassify the ten late patient endings as lost, and one existing lost order as open past the deadline, with consistent lifecycle timestamps and captures:

- Imaging remains 100.
- Gate A becomes \(100/101=99.01\%\).
- Terminal unimaged demand falls to 49.5%.
- Against the initial records over 40 seeds, current Gate B’s upper bound changes from zero to approximately \(-0.0042784\).

Both access measures improve without another patient being imaged.

The simulator does not expose arbitrary outcome relabelling as a plan action. This is a classification-sensitivity witness, not proof of an executable strategy exploit. Other gates may also constrain particular transformations.

The term “unimaged” must remain precise: Gate B counts selected terminal outcomes, not every patient still unimaged at observation end. Open orders are handled by Gate A.

### Lifecycle and observation-boundary assessment

| Scenario | Assessed behavior |
|---|---|
| Patient cancellation before deadline | Withdrawn from Gate A; counted in Gate B. |
| Patient cancellation exactly at deadline | Untimely in Gate A because withdrawal uses strict `<`; counted in Gate B. |
| No-show | Patient outcome; later rebooking can change its eventual outcome. |
| Lost order or walk-in who leaves | Excluded from Gate A under the new policy; counted in Gate B. |
| Rebooking or displacement | Original wanted time remains the access clock; administrative movement does not restart it. |
| Recall | Existing order is reoffered; original demand identity and wanted time remain relevant. |
| Open beyond deadline | Untimely in Gate A; not terminal-unimaged in Gate B. |
| Open within deadline at observation end | Gate A insufficient, not an individual gate pass. |
| Wanted at the horizon | Included. |
| Wanted after the horizon | Excluded from both access gates by design. |
| Received at or before takeover | Inherited cohort, rather than new-demand access rate. |
| Priority or demand-key change | Generally unmatched and insufficient; coarse-key collisions remain a limitation. |

Inherited access is reported separately. With maximum wait report-only, it has no new-demand clinical access-rate gate. Existing inherited case-selection comparisons do not establish absolute inherited access adequacy.

### Gate A’s 95% is a target, not a confidence level

**References:** `internal/eval/clinical.go:449–458`; `internal/eval/exact.go:39–40`.

Gate A compares the observed fraction with .95 using exact integer arithmetic. It does not calculate a 95% binomial confidence bound.

At the routine minimum of 23, 22 timely orders pass. If the true timely probability is .94:

\[
P(X\ge22),\quad X\sim\operatorname{Binomial}(23,.94)
=0.5947039.
\]

That is not necessarily wrong for an explicitly empirical acceptance rule. It must not be described as a 95% population-confidence test.

## 8. Weekly severity

### F7 — Real delayed work is detected; clinical calibration remains unresolved

**References:** `internal/eval/clinical.go:276–354`; `docs/calibration/clinical-5c4c/sim-access-5c4c.json:302–323`; `access-gates.md:104–122`, `:164–170`.

The denominator is acquisitions of the same priority in a block. It is appropriate for reporting timeliness among acquired studies. Replacing it with all demand would dilute reporting failures with patients never imaged.

The committed soon failures include:

| Seed | Severe / acquired over period | Worst block |
|---|---:|---:|
| 901 | 44 / 912 | 10 / 82 = **12.20%** |
| 906 | 44 / 876 | 7 / 43 = **16.28%** |
| 909 | 36 / 877 | 4 / 50 = **8.00%** |

Seed 901 fails six of 17 blocks. These are actual delays beyond the proposed severity threshold, not invented events caused by a denominator error.

Soon severe means more than 24 hours past its four-hour target: more than 28 hours from acquisition to report. That can be clinically consequential while remaining below the absolute maximum of 52 hours from acquisition.

**Assessment:** there is evidence of genuine delayed reporting. There is insufficient evidence to conclude that every weekly rejection represents an unacceptable practice under an approved clinical standard, or that the threshold’s operating characteristics are appropriate. The baseline’s failure alone cannot decide between those interpretations.

### Witness F7.1 — A population rate of 5% almost always fails the weekly maximum

**Type:** exact probability arithmetic.

**Inputs:**

- Soon blocks containing exactly 40 studies.
- Independent severe probability .05.
- Fifteen independent blocks.

A block fails at three or more severe cases:

\[
P(X\ge3),\quad X\sim\operatorname{Binomial}(40,.05)
=0.3232642.
\]

Therefore:

\[
P(\text{any of 15 blocks fails})
=1-(1-0.3232642)^{15}
=0.9971405.
\]

**Expected:** if 5% is intended as an underlying rate that an acceptable practice may attain, rejection should be calibrated to that interpretation.

**Actual:** the rule is a strict ceiling on every observed block and almost always rejects this model.

This behavior is coherent if the owner intends a strict local empirical ceiling. It is not a confidence test of an underlying 5% rate. The interpretation must be fixed before confirmation.

### Witness F7.2 — Pooling boundaries are non-monotone in nonsevere volume

**Type:** deterministic arithmetic.

**Inputs:** soon minimum cohort 40.

- Week 1: 39 acquisitions, three severe.
- Week 2: 100 acquisitions, none severe.

**Actual:** forward pooling gives 3/139 = 2.16%, passing.

Add one nonsevere acquisition in week 1:

- Week 1 now closes at 40.
- Its rate is 3/40 = 7.5%, failing.

More nonsevere service changes PASS to FAIL without increasing severe cases.

A final-remainder example behaves in the opposite direction:

- Last full block: 3/40, failing.
- Append a final remainder of 20 nonsevere acquisitions.
- The remainder merges backward, giving 3/60 = 5%, passing.

These are consequences of the specified algorithm, not mistakes in its implementation. They require explicit acceptance and prospective boundary witnesses.

### Counting, alignment and insufficiency

The implementation substantially follows the stated rule:

- Minimum acquisitions: urgent 20, soon 40, routine 100.
- Weeks accumulate forward until the minimum is reached.
- A short final remainder joins the last completed block.
- If the whole period is below the minimum, the weekly gate is insufficient.
- Equality at 5% passes; comparison uses integer cross-products.
- Each study contributes once to its acquisition-week block. A long-running severe study is not counted again every week.
- Monday alignment uses local calendar seconds. The aligned fidelity fixture tests passed.
- Acquisition exactly at takeover is excluded from new weekly cohorts; acquisition at the horizon is included.
- Delays are evaluated using eventual completion within observation, or observation end if unfinished.

The weekly statistic alone does not represent unresolved severity uncertainty: unfinished cases not yet severe are counted as nonsevere there. The companion severe-rate gate marks those cases undecided. The combined result must therefore retain that distinction.

**F9:** zero-acquisition weeks do not enter the weekly map. They do not alter numerator or denominator, but `WorstWeeks` counts occupied week entries rather than elapsed calendar duration. A sparse block can be described as fewer weeks than it actually spans.

**Required decision:** approve the local ceiling, minimum volume, pooling behavior and acceptable combined rejection rate. Do not relax the soon rule simply to make the baseline pass.

## 9. Absolute reporting limits

**References:** `internal/eval/clinical.go:108–117`, `:224–243`, `:579–600`; `internal/eval/measure.go:507–523`; `internal/eval/clinical_witness_test.go:337–349`.

The implemented limits are:

| Priority | Severe threshold past due | Absolute maximum past due | Absolute maximum from acquisition |
|---|---:|---:|---:|
| Urgent | More than 12 hours | 24 hours | 25 hours |
| Soon | More than 24 hours | 48 hours | 52 hours |
| Routine | More than 72 hours | 14 days | 15 days |

These translations follow the existing one-hour, four-hour and one-day turnaround targets.

### Exact timestamp witnesses

**Type:** code reasoning, supported by executed boundary and clinical witness tests.

For a study with due timestamp \(t_d\), let \(L\) be its absolute limit in seconds.

| Input | Expected and implemented result |
|---|---|
| `FinalSec = t_d + L` | No absolute violation. |
| `FinalSec = t_d + L + 1` | Absolute violation. |
| `FinalSec = 0`, observation ends at `t_d + L` | No observed absolute violation yet. |
| `FinalSec = 0`, observation ends at `t_d + L + 1` | Absolute violation. |
| Acquired by horizon, finalized after horizon but within observation | Final timestamp still used. |
| Final timestamp after observation | Invalid timestamp evidence. |
| Missing final timestamp | Treated as unfinished, not silently dropped. |

The violation comparison uses seconds before display values are truncated to minutes. A report can therefore violate at 24 hours plus one second while its displayed worst whole-minute value still equals 1,440 minutes.

**Daily sampling cannot hide an absolute violation.** These checks use study timestamps directly. Daily samples support reconciliation but do not determine the absolute maximum.

The executed 348-hour routine witness correctly fails against the 336-hour limit. Absolute violations bypass the shared allowance.

Important scope limits remain:

- Studies acquired after the horizon are outside this absolute reporting cohort, even if their originating access demand was eligible.
- Inherited unfinished studies are included, with the candidate’s takeover-based timing rule.
- Completed pre-takeover work is not the inherited unfinished cohort.
- An observed absence of violations does not guarantee future safety outside the observation.

### Rare stochastic violations at 0.92 utilization

The committed fidelity findings report urgent absolute violations on 4–11 of 2,500 stress seeds and soon violations on 1–7 of 2,500. These are model results, not newly executed results.

For independent seeds, per-seed urgent violation probabilities .002 and .0044 imply 40-seed rejection probabilities:

\[
1-.998^{40}=7.70\%,
\]

\[
1-.9956^{40}=16.17\%.
\]

Thus a queue can be stable and still fail frequently under a zero-tolerance maximum. This is not an implementation defect or justification for relaxing the maximum. It is a clinical and operating-domain decision.

The protocol designates 0.92 as stress rather than the stable false-rejection reference. That limitation must remain explicit in any eventual freeze.

## 10. Shared allowance

**References:** `internal/eval/clinical.go:509–532`, `:553–600`; `internal/eval/evaluate.go:648–677`; `docs/calibration/clinical-5c4b/acceptance-design.md:44–52`.

For the current .1 setting and 40 valid seeds, the shared allowance is four seeds.

Executed witnesses confirm:

- Four seeds each failing several gates count as four.
- Adding a fifth failing seed causes failure.
- Four service-failing seeds plus four different backlog-failing seeds count as eight.
- Any absolute reporting violation fails independently of the allowance.
- Gate B failure or insufficient pairing blocks the comparison outside the allowance.

There is an important distinction concerning insufficient evidence:

- Missing or malformed records should invalidate evaluation.
- Valid records with undersized clinical cohorts can consume the shared seed allowance.
- Unmatched Gate B demand blocks the comparison.

The implementation can therefore return a passing joint rule with up to four insufficient seed-level statuses. This is consistent with the older acceptance design, which explicitly counts insufficient gates as failing seeds. It is not consistent with an unqualified statement that every insufficient gate automatically makes the whole plan NO GO.

A seed with both failure and insufficiency is classified as failing. It still counts once, but the aggregate insufficient-seed list does not enumerate every evidence gap.

The four-of-40 rule is an empirical acceptance rule, not a 95% population reliability guarantee. Its reported confidence bound is not itself the acceptance criterion.

## 11. Reserved protocol assessment

### F5 — T1–T5 can miss the demonstrated statistical failure

**References:** `docs/calibration/clinical-5c4c/reserved-validation-protocol.md:43–48`, `:85–123`, `:129–143`.

**Requirement:** confirmation must assess the claims actually made by the acceptance framework, including clinically harmful false acceptance.

T5 measures rejection when stable practices are compared against themselves. T4 includes large injected loss and cancellation effects. A bare observed-difference rule can perform well on both while falsely certifying deterioration immediately beyond two points.

**Reproduction:** apply the existing Gate B rule to witness F1.1. Stable zero-deterioration cases can pass and large-loss cases can fail, while true 2.1-point deterioration passes at the probabilities already shown.

**Expected:** prospective assessment of coverage and false acceptance around the margin.

**Actual:** those targets are absent.

**Consequence:** successful T1–T5 results would not validate Gate B’s central confidence claim. This blocks the reserved run as written.

### T1 — Prospective empirical criterion, limited inferential strength

T1 explicitly uses a point estimate of whole-policy rejection. That is a prospective criterion, but weaker than demonstrating that the true rejection probability is at most .05.

With 2,500 seeds, disjoint groups of 40 produce 62 complete group decisions and 20 unused seeds.

For zero rejections among 62 groups, the one-sided 95% upper bound is:

\[
1-.05^{1/62}=4.717\%.
\]

A simultaneous bound across four scenarios is wider; a Bonferroni construction gives approximately 6.824%.

Specify the estimator, interval family, simultaneous method and treatment of leftover seeds. Do not describe passing a point-estimate target as proving the population target.

### T2 — Prospective materiality wording, incomplete decision specification

The criteria are written before the reserved run, which is a strength. However, materiality is determined from the same run’s noisy observed consequences.

Before execution, specify:

- Whether severe shares and loss increases are pooled or averaged by seed.
- The priority and window for each criterion.
- Treatment of uncertainty near materiality thresholds.
- Which stress scenarios are judged versus diagnostic.
- Treatment of urgent harm, which is not directly represented by the stated soon/routine materiality predicates.
- How every prespecified deterioration is reported even if classified nonmaterial.

For 1,000 seeds there are 25 independent groups of 40. Even 25/25 detections has a one-sided 95% lower bound:

\[
.05^{1/25}=88.707\%,
\]

below .90. T2 must either explicitly judge a point estimate or use a design capable of supporting the stronger confidence statement.

### T3 — Useful per-limit rare-event assessment

Zero violations in 2,500 gives:

\[
1-.05^{1/2500}=0.119758\%.
\]

That satisfies the stated .13% per-limit criterion. The bound corresponds to approximately 4.680% rejection over 40 independent seeds for one limit.

Separate bounds for three limits and multiple scenarios are not a simultaneous bound on whole-policy rejection. Retain a direct whole-policy assessment and state the uncertainty family.

The fidelity model failed its simulator-equivalence criterion at the simulator’s load and is unvalidated at the calibration loads. Rare-event bounds remain model quantities.

### T4 — Add witnesses that isolate each mechanism

Eight-day urgent reporting delays trigger absolute and weekly rules together. Add a weekly-concentration witness whose delays exceed the severe threshold but remain below the absolute maximum.

Also include:

- Exact seconds at and beyond each absolute boundary.
- Final-remainder and minimum-cohort transitions.
- Sparse and zero-volume weeks.
- Open, cancellation and no-show transitions.
- Capture permutations and missing terminal outcomes.

The existing unit suite already provides examples of isolating weekly severity below the absolute maximum.

### T5 — Stable comparisons are necessary but insufficient

Independent booking streams with shared demand are useful. They do not establish:

- population risk-difference coverage;
- near-margin false acceptance;
- behavior under baseline loss near 65%;
- robustness to within-seed shocks;
- validity with duplicate keys;
- behavior at small discordance counts.

Add analytical models with known truth and a prospective parameter grid spanning those conditions before using reserved streams.

### Prospective control and reproducibility

The protocol appropriately prohibits execution before review, owner decisions and digest pinning. It does not yet provide a fully frozen execution contract.

Before any reserved draw, pin:

- Candidate policy digest.
- Evaluator and model revision.
- Scenario indices and baseline mapping.
- Judged versus diagnostic scenarios.
- Estimands, primary targets and interval methods.
- Exact witness construction.
- Group partitioning and remainder handling.
- Treatment of ambiguous or missed targets.

A missed target must remain recorded as a missed target. An owner can explicitly accept a deviation, but it must not be described as successful confirmation. Changing a judged design after seeing results consumes the reserved family.

## 12. Historical compatibility and activation

**References:** `internal/eval/policy.go:308–309`, `:410–483`; `internal/eval/compare.go:142–144`; `internal/eval/neutrality_test.go:10–55`; `internal/eval/oc_fidelity_test.go:1377`.

The inspected activation paths support the intended separation:

- The default policy schema remains v4.
- A v5 policy must have a clinical section.
- Gate B activates only when `LostDemand` is present.
- The weekly-rate path activates only when its new field is present.
- Earlier v5 policies without these fields retain their earlier clinical behavior.

A v5 schema label alone therefore does not identify the candidate acceptance design. Confirmation must pin the complete policy.

Executed compatibility evidence:

- `TestV4Neutrality` passed against its historical fixed-fixture digest.
- `TestCandidatePolicyFile` passed.
- `TestOCFidelityModelUnchanged` passed its sampled record digests.

These support targeted byte compatibility, not a proof over every possible input.

The v4 policy SHA-256 matches the candidate’s declared superseded policy:

```text
4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3
```

The reviewed candidate SHA-256 is:

```text
ee3a2e53a9d62aced8f7337843c7e8a91db416666eb3586e0feb7845d68ebe28
```

Recording that digest here does not freeze or activate the policy.

Historical policies were not modified. The older acceptance and denominator documents describe superseded rules and should remain historical. The current normative specification and verdict prose need a clear precedence relationship.

### F9 — Reporting qualifications

Two additional reporting limitations deserve correction:

1. Gate B’s absolute fractions cover matched seeds only. Unmatched seeds are excluded; if none match, zero-valued fractions can appear despite insufficient evidence. Report all-demand absolute fractions separately from matched-subset fractions.
2. The no-action JSON’s whole-period `candidateTimelyFraction` does not alone establish that every configured T90/tail gate passes.

Neither issue independently permits acceptance because of the corresponding surrounding checks, but both can overstate what was measured.

## 13. Clinical interpretation

The framework must keep five conclusions separate:

| Measure or verdict | What it can establish |
|---|---|
| Reporting timeliness | Delay after acquisition relative to the reporting target, within the specified cohort and observation. |
| Access timeliness | Timely imaging among the defined Gate A denominator. |
| Lost-demand non-deterioration | A baseline-relative change in specified terminal outcomes, subject to a valid inferential method. |
| Absolute access adequacy | Whether patients receive clinically acceptable access across all relevant demand; currently not established. |
| Economic performance | Profit, cash, funding and financial superiority after required clinical gates; not clinical adequacy. |

The documentation acknowledges several of these limitations. The verdict machinery does not consistently carry them into favorable classifications.

Neither financial superiority nor reporting timeliness compensates for inadequate absolute access. Equally, a baseline-relative intervention can be worth accepting without asserting that the practice has become clinically adequate. That is an owner decision requiring explicit wording.

## 14. Required remediation before confirmation

1. **Define the acceptance claim.** Separate empirical compliance, population non-deterioration and adequate absolute access.
2. **Replace Gate B’s invalid bound.** Specify the estimand, sampling unit, clustering and collision treatment.
3. **Validate statistical operating characteristics before reserved use.** Include exact analytical counterexamples, zero discordance and near-margin false acceptance.
4. **Repair endpoint selection.** Use the validated named end capture.
5. **Reject missing and unknown outcomes and priorities before measurement.**
6. **Resolve pairing identity.** Export stable identity or adopt a justified conservative alternative.
7. **Approve clinical interpretation.** Decide the absolute-access limitation, weekly local ceiling, severity definitions, allowance for insufficient cohorts, inherited exclusions and intended utilization domain.
8. **Make verdict limitations mandatory.** Display absolute access measures alongside any non-deterioration verdict.
9. **Reconcile current specification and inherited verdict prose.**
10. **Recommit a complete reserved protocol with pinned policy and implementation.**
11. **Freeze only after prespecified results are reviewed and deviations recorded explicitly.**

These are remediation requirements, not implemented changes.

## 15. Optional improvements after freeze

Provided the blocking issues have already been resolved:

- Richer cancellation and loss-cause recording.
- Versioned order-to-study links.
- Calendar-week and pooled-block displays with actual date spans.
- Separate terminal-unimaged and still-open fractions.
- Broader simulator-fidelity studies at the intended operating loads.
- Expanded regression coverage across action families.
- Routine validation of every policy fraction’s domain.

Stable identity or an adequate alternative is a prerequisite for the chosen inference, not optional if patient-level pairing remains its foundation.

## 16. Reproduction record

The existing binary was run from `snapshot/internal/eval` with this exact selection:

```sh
./eval.test \
  -test.run '^(TestClinical(SevereReportDelay|AgeLimitExact|DelayedAccess|UnresolvedAndLostOrders|PatientCensoring|ConcentratedDelays|AccessDeterioration|ContradictoryTimestamps|SharedAllowance|AbsoluteApartFromAllowance|InheritedTiming|InsufficientBaselineNotRestored|PolicyValidation)|TestWitness.*|TestGate(LostUrgentOrders|DelayedUrgentAcquisition|LossNotOffsetByTimeliness|CancellationManipulation|Rebooking|HorizonCensoring|LowVolumeWeeks|OverlappingFailures|MissingBaseline|NoncomparableDemand|WeeklyRemainder)|TestCandidate(PolicyFile|Validation)|TestV4Neutrality|TestSimProfileDerived|TestOCReaders|TestOCReaderRules|TestOCFidelity(Calibration|ModelUnchanged|AlignedWeeks|PairedDemand))$' \
  -test.v \
  -test.timeout=10m
```

**Observed:** 44 selected tests passed.

The new statistical and implementation counterexamples above were not added as Go tests. Their exact inputs, calculations and expected-versus-implemented outcomes are preserved in this report. Disposable supporting files are:

- `witness/targeted-tests-command.txt`
- `witness/targeted-tests.log`
- `witness/gate_b_arithmetic.py`
- `witness/gate_b_arithmetic.txt`
- `witness/implementation-witnesses.txt`

The reserved streams remain untouched.