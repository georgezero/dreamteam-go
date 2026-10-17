# Slice 5C.0 evaluation-policy audit (Astra High)

Independent read-only audit of the frozen V0.2 evaluation policy at commit e2ce1fd (0.2.0-dev.9), 2026-10-17. Snapshot of the audited files only; no strategy was run, no file changed, no holdout seed generated. Line references are to e2ce1fd.

**A. Slice 5C.1 strategy development: CONDITIONAL GO**

Development on **901–910 only** can proceed as exploratory work. The framework is not yet suitable for definitive strategy ranking or freezing a winner. Resolve the cohort, tail-comparison, backlog and reliability decisions below before selection; otherwise development will optimize against known gaps.

**B. Holdout unsealing: NO GO**

Keep the holdout sealed until the selection rules are clarified, the identified implementation defects are corrected, and the evaluation harness binds the complete seed set, strategies, policy and engine.

This audit used the supplied snapshot only. No strategy was run, no files were changed, and no holdout seeds were generated. Go tests and the actual holdout digest were not executed or independently verified. Numerical calculations below use synthetic binomial assumptions.

**C. Ranked findings**

No **Critical** defect was verified. The following **High** findings can change eligibility or invalidate the intended interpretation.

“Blocks development” below means blocks definitive strategy comparison/selection; implementation and exploratory development can continue.

**H1. The urgent evidence minimum does not establish compatibility with the seed-reliability requirement.**

Evidence: [policy.go:175](../../internal/eval/policy.go), [evaluate.go:337](../../internal/eval/evaluate.go), and [README.md:77](../../docs/evaluation/README.md).

The formula is correctly described as a **standard-error heuristic at the assumed gate probability**:

\[
n_{\min}=\left\lceil g(1-g)/0.05^2\right\rceil.
\]

It gives 43/88/23, but establishes neither a narrow 95% interval nor a high probability of passing the per-seed gate. For example, **5/43 urgent breaches passes**, although its Wilson 95% interval is approximately **5.1%–24.5%**. Approximate sample sizes for a 95% half-width of five points *at the gate* would instead be 163/335/87; these are illustrative precision calculations, not recommended replacements.

For fixed cohort size \(n\), independent breaches with probability \(p\), and independent seeds:

\[
q(n,p)=P\{\mathrm{Binomial}(n,p)>\lfloor0.12n\rfloor\},
\quad
R(n,p)=P\{\mathrm{Binomial}(40,q)>4\}.
\]

Each table cell is **per-seed failure probability / probability more than four of forty fail**:

| Urgent count | True breach 6% | True breach 8% | True breach 10% |
|---:|---:|---:|---:|
| 47 | 6.09% / 9.40% | 17.08% / 83.65% | 32.86% / 99.92% |
| 49 | 7.18% / 15.67% | 19.54% / 91.37% | 36.54% / 99.98% |
| 50 | 2.89% / 0.57% | 10.19% / 38.64% | 22.98% / 96.88% |
| 58 | 5.79% / 7.94% | 17.95% / 86.84% | 35.97% / 99.98% |
| 59 | 2.41% / 0.27% | 9.69% / 34.59% | 23.42% / 97.30% |
| 66 | 4.34% / 2.87% | 15.57% / 76.77% | 33.86% / 99.95% |
| 67 | 1.84% / 0.08% | 8.54% / 25.34% | 22.39% / 96.26% |
| 71 | 2.59% / 0.36% | 11.26% / 47.40% | 27.75% / 99.40% |

Across **every integer \(n=47\ldots71\)**:

| True breach | Per-seed failure range | Forty-seed rejection range |
|---:|---:|---:|
| 6% | 1.84%–7.18% | 0.08%–15.67% |
| 8% | 8.54%–19.54% | 25.34%–91.37% |
| 10% | 22.39%–36.54% | 96.26%–99.98% |

The discontinuities are real: the permitted breach count increases at 50, 59 and 67 studies.

These fixed-size cohorts are all sufficient, so the table has **no insufficiency failures**. With random volume, the relevant probability becomes:

\[
q=P(N<43)+\sum_{n\ge43}P(N=n)\,q(n,p).
\]

The snapshot does not establish that volume distribution. As a strictly hypothetical illustration, Poisson mean 47 gives a **26.0%** probability of fewer than 43 acquisitions. It must not be interpreted as a measured simulation result.

Consequences:

- A true 10% urgent rate is practically incompatible with reliable passage at these volumes, despite being below 12%.
- A true 8% rate can also fail frequently.
- Passage is not mathematically unattainable: sufficiently better service can pass.
- Queue-related dependence and seed heterogeneity mean the binomial calculation is a benchmark, not a validated simulation probability model.

**Minimal correction:** retain the owner’s strict 10% allowance, but prospectively specify the acceptable underlying service margin and rejection probability, then choose a fixed observation/cohort design using its operating characteristics. Simply raising the minimum count without increasing observation would increase insufficiency.

**Blocks development:** Yes, before selection unless the owner explicitly accepts these operating characteristics. **Blocks unsealing:** Yes.

**H2. Paired tail comparisons can compare different durations and different demand cohorts.**

Evidence: [measure.go:163](../../internal/eval/measure.go), [compare.go:49](../../internal/eval/compare.go), and [compare.go:223](../../internal/eval/compare.go). The existing extended-tail test explicitly permits a candidate ending at day 170: [eval_test.go:511](../../internal/eval/eval_test.go).

`tailEnd` is calculated separately for each record. Comparisons match the name `"tail"` without checking equal bounds. A no-action baseline normally has a 60-day tail; a candidate can have an 80-day tail.

**Consequence:** raw acquisitions and financial totals are not comparable. Illustratively, 80 days at 75% of the baseline acquisition rate produces the same count as 60 baseline days. Later successful demand can dilute earlier deterioration. This is a mathematical consequence of the comparison, not an executed strategy.

**Minimal correction:** use common paired tail bounds, extending the baseline and necessary captures to match; freeze the comparison-horizon rule prospectively.

**Blocks development:** Yes, for comparisons with unequal tails. **Blocks unsealing:** Yes.

**H3. Access gates lose outstanding orders across receipt-window boundaries.**

Evidence: [measure.go:379](../../internal/eval/measure.go), especially lines 385 and 408–414; [measure.go:449](../../internal/eval/measure.go); [compare.go:197](../../internal/eval/compare.go).

Access includes only orders **received inside the current window**, then asks whether they were due by its end.

A referral received on day 89 and originally wanted on day 100 is:

- not due in the T90 access denominator;
- excluded from tail access because it was received before day 90;
- included in T180 access reporting, but T180 case selection is not gated.

Inherited orders are also excluded from new-demand access gates. Orders already due at T90 are counted there, but their later resolution or worsening is not followed by tail access gates.

**Consequence:** difficult future-due or carried-over patients can remain unresolved without an order-level hard gate following them through service. Aggregate acquisition counts can be maintained by serving other patients.

**Minimal correction:** carry unresolved and not-yet-due receipt cohorts forward, preserve original order identity and wanted time, and gate their eventual due outcomes. Include an explicit inherited-order requirement.

**Blocks development:** Yes, before selection involving access changes. **Blocks unsealing:** Yes.

**H4. Backlog stability does not prevent clinically important persistent old work.**

Evidence: [measure.go:516](../../internal/eval/measure.go), [measure.go:533](../../internal/eval/measure.go), and [README.md:168](../../docs/evaluation/README.md).

Only the difference between two block means determines stability. Slope, rolling means, ages, overdue fractions and flow do not affect it.

For the stated takeover, rota anchor and a day-150 tail:

- reference block: **days 55–82**;
- final block: **days 111–138**;
- days **139–150** are outside the final comparison block.

Verified consequences of the rule:

- A persistent group of old studies can remain in both blocks with zero growth.
- Fresh backlog can be replaced by equally numerous much older studies.
- Intermediate deterioration that recovers before the final block can escape.
- Deterioration after day 138 can escape this backlog gate.
- A deliberately high reference backlog makes the permitted later level higher.

The breach gates do not repair this: each study contributes at most one breach, regardless of whether it is one minute or months late. Inherited studies are not breach-gated.

**Minimal correction:** keep rota alignment, but add a prospectively approved overdue-age/completion safeguard covering inherited and earlier cohorts; assess intermediate rolling blocks and the residual period through tailEnd.

**Blocks development:** Yes, before selection. **Blocks unsealing:** Yes.

**H5. The implementation allows 10% failure separately per gate, not 10% of seeds failing any service gate.**

Evidence: [evaluate.go:253](../../internal/eval/evaluate.go), [evaluate.go:288](../../internal/eval/evaluate.go), and [evaluate.go:381](../../internal/eval/evaluate.go). Compare [README.md:74](../../docs/evaluation/README.md) with its explicitly per-gate wording at lines 440–441.

The code computes each seed’s combined service status, but does not use the fraction of combined failures for plan eligibility.

**Consequence:** four different failing seeds in each of six priority/window gates can allow **24/40 seeds to fail at least one service gate**, while every individual gate and pooled rate passes. On development seeds, six distinct failures could similarly leave only 4/10 seeds passing all six gates.

Within an individual gate, pooling cannot hide more than the allowed failures. It can, however, hide arbitrarily severe outcomes on the permitted failing seeds if other seeds supply enough passing volume.

`failLimit` correctly produces **1/10, 2/20 and 4/40**. Insufficient service cohorts count toward these limits; an unresolved cohort also makes the pool undefined.

**Minimal correction:** explicitly choose per-gate or joint reliability. Given the owner’s stated strict reliability objective, recommend gating the union of service failures as well as the pooled rates.

**Blocks development:** Yes, until the intended interpretation is fixed. **Blocks unsealing:** Yes.

**H6. Holdout protection is a cooperative harness convention, not an enforced one-time evaluation protocol.**

Evidence: [holdout.go:145](../../internal/eval/holdout.go), [holdout.go:175](../../internal/eval/holdout.go), and [eval_test.go:620](../../internal/eval/eval_test.go).

Verified gaps:

- A strategy digest need only be nonempty; the test accepts `"x"`.
- `approvedBy` need only be nonempty; `date` is not checked.
- File existence is checked, not whether the unseal record or strategies are committed.
- No engine digest, strategy-content comparison, run-consumption record or reproduction-byte check is enforced.
- `Evaluate` checks matching candidate/baseline seed sets, not membership in the required complete 40-seed set.
- `PlanSHA256` is stored but not validated or required consistently across a plan’s seeds.

The supplied calibration entry points are guarded: [calib_test.go:182](../../internal/sim/calib_test.go), [calib_dynamic_test.go:252](../../internal/sim/calib_dynamic_test.go), and [session/calib_dynamic_test.go:75](../../internal/session/calib_dynamic_test.go). But direct seed-taking helpers do not guard themselves: [calib_dynamic_test.go:326](../../internal/sim/calib_dynamic_test.go) and [evalrecord_test.go:46](../../internal/sim/evalrecord_test.go). This proves bypass paths exist; it does not establish that any were used.

The generator and digest procedure are internally consistent **by inspection**: big-endian first-eight-byte reduction, duplicate/exclusion filtering, then SHA-256 of comma-separated decimal seeds in generation order. The manifest’s range is disjoint from all explicitly identified exposed sets in the supplied documents. Actual digest equality remains unverified in this audit.

**Minimal correction:** create a canonical guarded runner that checks committed artifact hashes, explicit approval provenance, engine identity, the complete seed/strategy matrix, and immutable first-run outputs. Treat alternate run paths as outside the approved protocol.

**Blocks development:** No, with development-seed restrictions. **Blocks unsealing:** Yes.

**H7. GO establishes incremental benefit, not absolute profitability.**

Evidence: [compare.go:385](../../internal/eval/compare.go), versus [V0.2_DESIGN.md:3112](../../docs/V0.2_DESIGN.md).

GO requires a positive lower confidence bound for **incremental** T90 profit and a positive mean T180 increment. There is no absolute operating-profit requirement.

**Consequence:** a loss-making candidate can receive GO if the baseline loses more and cash remains adequate. This matches the frozen JSON/README, but conflicts with §13.3’s “positive operating profit” requirement if GO means financial success.

The negative-increment restoration branch is correctly CONDITIONAL GO, but tests `!base.Eligible`, which includes baseline insolvency and backlog failure—not specifically service restoration.

**Minimal correction:** distinguish financial superiority, absolute profitability and **service purchase** explicitly. Either require absolute profitability for financial GO or label GO as relative improvement. Label a negative-increment service-restoring result “CONDITIONAL GO — service purchase,” with its cost.

**Blocks development:** Yes, before selecting by verdict. **Blocks unsealing:** Yes.

**Medium findings**

**M1. Exact permitted case-selection boundaries can fail through floating-point subtraction.**

Evidence: [compare.go:241](../../internal/eval/compare.go), [compare.go:296](../../internal/eval/compare.go), [compare.go:327](../../internal/eval/compare.go), and [compare.go:340](../../internal/eval/compare.go).

Synthetic binary64 arithmetic gives:

- `1 − 98/100 = 0.020000000000000018`: exactly 2% acquisition loss fails.
- `80/100 − 75/100 = 0.050000000000000044`: exactly five points fails.
- `90/100 − 88/100 > 0.02`: exactly two served points fails.
- `7/100 − 5/100 > 0.02`: exactly two loss points fails.

**Minimal correction:** compare rational counts using appropriately safe integer arithmetic, or a documented numerical tolerance. Add exact-boundary tests.

**Blocks development:** Yes, for reliable ranking near boundaries. **Blocks unsealing:** Yes.

**M2. Case-selection tolerances constrain volume loss; they do not guarantee an unchanged difficulty mix.**

Evidence: [compare.go:217](../../internal/eval/compare.go), [compare.go:237](../../internal/eval/compare.go), [compare.go:292](../../internal/eval/compare.go), and [README.md:307](../../docs/evaluation/README.md).

Verified denominator definitions are:

- served fraction: served **due orders / due orders**, including patient-caused failures in the denominator;
- acquisition drop: candidate acquired / baseline acquired, a **relative** loss;
- referral loss: lost referrals / all referrals received;
- abandonment: walk-ins left / all walk-ins received.

Rebooking does not move `WantedSec` inside the evaluator. Cancellation/no-show classification does not remove an order from the due denominator.

However:

- Difficulty, exam subtype and priority-by-modality intersections are not gated.
- Small selective losses within the tolerances remain permissible.
- Per-seed checks cover priority, not modality.
- Baseline strata below 30 due are skipped per seed.
- The code also exempts **pooled** strata below 30 due, although the README describes that minimum only for per-seed checks.
- A rare stratum can retain due orders but serve none and still pass its pooled served check.

**Minimal correction:** document the exemptions accurately, remove the unintended pooled exception or approve it explicitly, and narrow the claim from “cannot select easier cases” to the protection actually provided. Add targeted mix checks where the simulation exposes suitable attributes.

**Blocks development:** Yes, until accepted selection scope is clear. **Blocks unsealing:** Yes.

**M3. Financial eligibility has a narrower obligation scope and observation contract than the design suggests.**

Evidence: [export_eval_test.go:149](../../internal/sim/export_eval_test.go), [record.go:114](../../internal/eval/record.go), [measure.go:372](../../internal/eval/measure.go), and [evaluate.go:271](../../internal/eval/evaluate.go).

Verified behavior:

- Solvency uses the **end capture’s** transaction-minimum field, through the entire observation period—normally day 181, not merely tailEnd.
- Funding gaps block GO **only at tailEnd**.
- Future minimum obligations computed by the recorder cover remote retainers, not future unavoidable employee wages/notice commitments described in §13.3.
- `MinCashSec` is populated only when `st.fin.minCash == ev.Money.MinCashCents`; otherwise it silently remains zero.
- Past-due amounts are asserted to be zero by construction.

The visible bridge keeps receivables and unbilled revenue out of cash. It adds in-flight fees separately from ledger payables and starts future retainers tomorrow, consistent with current-day midnight accrual. **No definite remote-fee/retainer double counting was found.**

The production implementations of `Evaluation`, `noteCash`, `inFlight`, `retainerBetween` and payroll provisioning are absent. Consequently, transaction-minimum provenance, current-day retainer accounting and interactions between guarantee provisions and unposted wages cannot be fully certified from this snapshot. The synthetic cash test supplies the minimum itself; it does not test the production minimum tracker.

**Minimal correction:** specify the exact obligation scope and solvency horizon; include promised employee commitments or explicitly narrow the claim; provide an evaluation-window minimum timestamp; add production recorder checks across posting, payroll and contract boundaries.

**Blocks development:** No for exploration; financial selection needs verification. **Blocks unsealing:** Yes.

**M4. Record integrity checks do not enforce a complete, correctly timed observation record.**

Evidence: [measure.go:201](../../internal/eval/measure.go), [measure.go:239](../../internal/eval/measure.go), [measure.go:621](../../internal/eval/measure.go), and [compare.go:124](../../internal/eval/compare.go).

Checks reconcile supplied samples, but do not require a complete ordered daily grid, validate sample dates against timestamps, reject duplicate capture names, or comprehensively validate study timestamp ordering and observation bounds.

Closure is explicitly required only for hard-gated windows, not T180. Payback silently excludes seeds missing daily samples. Thus incomplete records can change reported payback denominators without invalidating the evaluation.

**Minimal correction:** validate required captures, timestamps, unique identities and complete samples; enforce validation observation requirements; make missing payback data explicit rather than silently excluding seeds.

**Blocks development:** No for controlled exploration. **Blocks unsealing:** Yes.

**M5. Some supposedly informational §13.3 measures are absent or measured differently.**

Evidence: [record.go:52](../../internal/eval/record.go), [measure.go:35](../../internal/eval/measure.go), [measure.go:401](../../internal/eval/measure.go), and [export_eval_test.go:214](../../internal/sim/export_eval_test.go).

The record lacks the coverage/workforce vector and actual/current wanted-to-service access distributions described in §13.2–13.3. Booking lead is receipt-to-first-appointment, not the specified access measures.

Moreover, displacement counts are collected at `Finish`, then reused for earlier windows. A displacement after T90 can therefore appear in T90 access reporting.

**Minimal correction:** separate “reported,” “not implemented,” and “not gated”; capture displacement history with event times; supply the promised informational vector or explicitly defer it.

**Blocks development:** No. **Blocks unsealing:** Yes for truthful report scope; adding every deferred metric is an owner scope decision.

**Low findings: statistical reporting labels overstate some outputs.**

Evidence: [stats.go:24](../../internal/eval/stats.go), [stats.go:60](../../internal/eval/stats.go), [compare.go:75](../../internal/eval/compare.go), [evaluate.go:414](../../internal/eval/evaluate.go), and [README.md:203](../../docs/evaluation/README.md).

- Paired-t arithmetic, Wilson arithmetic and nearest-rank definitions are correct by inspection. The t-quantile implementation uses the appropriate incomplete-beta CDF; tests include df 1/9/19/39/1000.
- Wilson intervals describe **failing-seed proportions**, not pooled study breach rates.
- Four failures in forty gives approximately **3.96%–23.05%**, not evidence with 95% confidence that population failure is at most 10%.
- Even when the underlying seed-failure probability is exactly 10%, the probability of observing more than four failures is **37.1%**.
- `worstSeedBreachRate` contains separate candidate and baseline maxima, potentially on different seeds; it is not a paired worst-seed difference.
- No worst incremental-profit seed is stored despite the README promise.
- Undefined cohort values can enter reported means/worst calculations as zero.
- The censored payback median is correctly the `ceil(n/2)`-th repayment when all included censoring is at the common horizon. Its limitation is the missing-seed exclusion above.

**Minimal correction:** correct labels, include defined counts and worst-seed identities, and add the missing worst-profit result.

**Blocks development:** No. **Blocks unsealing:** No independently, provided interpretation is corrected.

**D. Recommendations on the nine policy decisions**

| Decision | Recommendation |
|---|---|
| **1. Due-time breach classification** | **Approve.** For acquisition \(s\in(a,b]\), due time is at most \(b+1440\) minutes. A routine acquisition exactly at \(b\) is due exactly at closure. Finalization at due passes; unfinished at due breaches. Settle all endpoint events before capture. Add the exact routine-boundary test. |
| **2. Minimum counts 43/88/23** | **Retain only as a provisional evidence floor.** Do not treat it as a reliability or 95%-precision guarantee. Resolve H1 prospectively. |
| **3. Maximum 10% failing seeds** | **Approve the strict risk choice**, but resolve per-gate versus joint failure and distinguish a sample acceptance rule from a confidence guarantee. Recommend joint service reliability. |
| **4. Selection tolerances** | **Provisional approval after correction.** Denominators mostly match the intended definitions, but carry-forward cohorts, unequal tails, rare-stratum exemptions and exact-boundary arithmetic need fixing. |
| **5. Backlog stability** | **Do not approve as the sole backlog safeguard.** Retain rota-aligned means; add old-work and intermediate/residual-period protection. |
| **6. Zero negative cash; funding gap blocks GO** | **Approve.** Verify recorder provenance and define complete obligations and the exact gate instants. Receivables must remain outside available cash. |
| **7. GO conditions** | **Approve precedence, revise classification.** INVALID precedes hard-gate NO GO; profitability cannot offset failed hard gates. Distinguish absolute profitability, incremental superiority and service purchase. |
| **8. Forty-seed locked holdout and one-time protocol** | **Approve the design**, conditional on enforceable artifact binding, complete evaluation and reproduction controls. The present guard is insufficient to substantiate the stronger protocol claims. |
| **9. Other §13.3 baseline-relative tolerances remain informational** | **Approve.** Promote none after examining candidate outcomes. Any promotion must be prospectively frozen before selection. Clearly identify measures that are absent, rather than merely informational. |

The due-time implementation itself is sound: [measure.go:273](../../internal/eval/measure.go) and [measure.go:329](../../internal/eval/measure.go). Policy validation requires closure at least as long as every target: [policy.go:146](../../internal/eval/policy.go).

The 24-hour extension is sufficient to **decide breach status**, not to establish eventual completion or clinical acceptability of prolonged delay.

**E. Additional unresolved decisions**

The README has **ten numbered open-decision entries**, not nine or eleven: [README.md:427](../../docs/evaluation/README.md). Entries 1–9 are the nine listed in the request. Entry 10 combines two scope assumptions:

1. remote diagnostic quality is not modelled;
2. payer strata are omitted.

Counting those separately produces eleven subjects, but there are not eleven numbered policy decisions. The earlier enumeration should be corrected explicitly.

Further choices requiring resolution are:

- Acceptable urgent-gate rejection probability and the fixed cohort/tail design.
- Per-gate versus joint seed reliability.
- Common paired tail bounds.
- Required resolution/age limits for inherited and carried-over work.
- Whether GO requires absolute profitability.
- Whether baseline financial failure alone can justify a “service restoration” conditional verdict.
- Employee commitment scope and whether funding gaps after tailEnd also block GO.
- Rare pooled strata and permitted within-stratum case-mix changes.
- One primary holdout strategy versus multiple confirmatory candidates. Testing several and selecting the apparent winner requires a prospectively declared multiplicity/selection rule.
- Which §13.2–13.3 informational measures must actually be present.

Development-seed confidence intervals should be treated as exploratory after repeated tuning. A locked independent evaluation is what provides the separation from selection.

**F. Can the evaluator be gamed?**

| Mechanism | Answer |
|---|---|
| **Patient selection** | **Yes.** Carry-forward orders can escape gated access follow-up; marginal strata do not preserve difficulty mix; tolerances permit bounded selective losses; unequal tails can mask acquisition reductions. Cancellation/rebooking does not directly remove a due denominator inside the evaluator. |
| **Unfinished reports** | **Yes, through continued deferral after a breach.** Keeping a report unfinished cannot erase its due-time breach, but delaying an already-breached report indefinitely adds no further hard-gate penalty if backlog counts remain acceptable. |
| **Backlog aging** | **Yes.** Persistent old studies, replacement of fresh backlog by old backlog, intermediate spikes and the residual period after the last complete block can escape the hard backlog gate. |
| **Financial timing** | **Yes at the evaluation-rule level.** GO checks funding gaps only at tailEnd and financial benefit remains posting-based; unposted costs and later funding gaps do not independently veto GO. Future employee minimum commitments are absent from the bridge. An executable strategy exploiting these paths was not demonstrated, and no payment-deferral action or definite ledger double count was verified. |

These are vulnerabilities of the evaluated rules. They do not establish that historical plans A–D exploited them. Historical D’s mean-versus-individual-seed distinction is already documented at [PROGRESS.md:607](../../docs/PROGRESS.md), and its old backlog result should remain historical evidence under its original rule.

**G. Minimal remediation plan**

1. **Correct exact threshold comparisons and misleading labels.** Add equality tests for 5-point, 2-point and 2% boundaries. **Before unsealing.**
2. **Resolve the policy ambiguities prospectively:** urgent operating characteristics, joint reliability, absolute versus incremental profit, service-purchase classification, and common tails. Publish a new policy version before definitive strategy selection. **Before unsealing.**
3. **Compare identical paired tail windows and carry outstanding order cohorts forward.** Preserve original wanted times and follow inherited obligations separately. **Before unsealing.**
4. **Add the smallest effective old-work safeguard.** Cover inherited/earlier studies, intermediate backlog deterioration and the interval after the final complete block. **Before unsealing.**
5. **Strengthen record validation and financial integration evidence.** Require complete timed captures/samples; verify transaction minima, obligation components, wage posting and retainer boundaries; make missing data explicit. **Before unsealing.**
6. **Build the canonical holdout runner and artifact contract.** Bind actual strategy files, policy, engine, approval and all forty seeds; retain immutable first-run records and verify reproduction-only reruns. **Before unsealing.**
7. **Add adversarial regression fixtures** for staggered per-gate failures, receipt-before/due-after boundaries, unequal tails, persistent old studies, temporary spikes, rare strata and malformed records. Existing tests cover basic failures but do not establish resistance to these mechanisms.
8. **Freeze the strategy set and selection rule only after those changes.** Then obtain owner approval for the concrete unseal record and conduct the one-time evaluation.