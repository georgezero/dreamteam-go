# Slice 5C.4g final clinical policy audit

**Audited revision:** commit 85f70d6 (0.2.0-dev.20), the candidate v0.2-eval-5 clinical acceptance framework (not frozen, not in force), 2026-10-19. Line references are to 85f70d6.

The audit was independent, adversarial and read-only. The auditor read a `git archive` snapshot of 85f70d6 on a separate machine and ran only targeted existing tests and disposable synthetic witnesses kept outside the snapshot (see "Reproduction record"). It ran no strategy, reference seed, holdout seed, reserved stream PCG(20261201, ·) or unseal procedure.

* **Everything below the rule is the auditor's report as delivered, unchanged.**
* **No remediation is in this commit.** Maintainer checks, if any, will follow in a separate document.

---

**Independent audit — Slice 5C.4g, Final Clinical Policy Audit**

Audited snapshot: **85f70d6**, version **0.2.0-dev.20**.  
Policy: proposed **v0.2-eval-5**, not frozen.  
Audit date: **2026-10-19**.

**Final answer: NO GO for running the reserved confirmatory calibration as presently specified.**

The empirical Gate B calculation, deterioration safeguard and principal clinical gate mechanics implement their stated rules. The reserved protocol is not complete enough to spend the reserved stream: it leaves an acceptance tolerance unspecified, sets a concentrated-harm detection target inconsistent with the allowance, and includes a supposedly paired scenario whose demands do not match.

The severity rules also permit substantial boundary-window harm. These are material clinical-policy limitations, distinguished below from implementation defects. No threshold changes are proposed as automatic fixes.

| Required decision | Verdict | Scope |
|---|---|---|
| **A. Empirical Gate B** | **GO** | Correct observed, equally weighted seed-mean rule; no population-confidence certification. |
| **B. Per-seed safeguard** | **GO** | Correct thresholds and allowance integration; neither a veto on every harmed seed nor a calibrated simulator guarantee. |
| **C. Severity windows** | **GO, with material limitations** | Implements the specified acquisition-based windows. Partial windows and post-horizon acquisitions prevent a broader local-safety claim. |
| **D. Access Gate A** | **GO, with limitation** | Correct conditional timeliness rule; does not establish adequate absolute access. |
| **E. Full v5 evaluator** | **GO for the specified limited acceptance decision** | No blocking acceptance-calculation defect demonstrated. Reporting defects and clinical scope limitations remain. |
| **F. Reserved confirmatory protocol** | **NO GO** | Findings G1–G3 below. |
| **G. Running reserved calibration** | **NO GO** | Resolve and prospectively lock G1–G3 before any reserved draw. |
| **H. Freezing v0.2-eval-5** | **NO GO** | No completed, interpretable confirmation; remaining limitations must accompany any eventual freeze. |

Here, component GO means suitable to enter a properly specified calibration. It does not authorize execution or policy activation.

**Scope and evidence**

The repository remained unchanged. All disposable files, overlays, caches and logs were placed under `/tmp/dt-audit-5c4g`, outside the snapshot. No fixes were implemented.

The audit inspected the requested documents, the superseding 5C.4f protocol and its incorporated predecessors, evaluator arithmetic and aggregation, policy validation, record integrity, comparison and verdict code, simulator demand and lifecycle code, evaluation export, and compatibility paths.

Execution comprised:

- **37 targeted existing evaluator tests**, all passing.
- **Eight independently constructed synthetic witness functions**, all passing their assertions about observed behavior.
- Independent probability calculations.
- One small fidelity-model pairing witness using **PCG(1, ·)**.

Only one `go test` ran at a time, with `GOMAXPROCS=2`. No `DT_*` experiment, strategy evaluation, reference seed, holdout seed, reserved PCG(20261201, ·), unseal operation or freeze was run.

The supplied archive has no Git history. Commit identity is therefore the maintainer’s snapshot identification, rather than an independently verified Git object. Historical non-execution cannot be proved from this archive; the audit found no reserved-family use in executable Go or JSON artifacts, and introduced none.

The candidate SHA-256 matches the protocol:

```text
d901fe5877684567c6c2d19ad1c063e8cf36ce269ba6fd44ba54299f052e348b
```

**Disposition of previous findings**

| Finding | Classification | Evidence and remaining qualification |
|---|---|---|
| **F1 — Invalid population confidence bound** | **CLOSED** | The candidate uses exact empirical acceptance, not the former bound. The diagnostic does not decide acceptance. `internal/eval/lostdemand.go:271–274,312–350`. A separate statistical-exposition error remains in G7. |
| **F2 — Incorrect end capture** | **CLOSED** | Gate B selects the unique named `end` capture, verifies its timestamp and outcome-vector length. Capture permutation test passed. `lostdemand.go:158–173`. |
| **F3 — Invalid outcomes and unknown priorities** | **CLOSED** | v5 checks known priorities, known capture outcomes and terminal outcomes before clinical measurement. Invalid comparisons return INVALID before favorable classification. `clinical.go:603–650`; `compare.go:464–466`. |
| **F4 — Pairing and seed clustering** | **CLOSED WITH LIMITATION** | Immutable origins and request equality replace timestamp matching; each seed contributes one fraction. Missing or duplicated origins block acceptance. Exogeneity is supported for the inspected simulator model, not arbitrary changed demand models. The reserved seasonal scenario violates that premise: G3. |
| **F5 — Inadequate near-margin validation** | **STILL OPEN** | Near-margin scenarios were added, but T5’s false-acceptance tolerance is still unstated and the injection construction is ambiguous. No population-coverage target is required for this empirical rule. |
| **F6 — Misleading access adequacy** | **CLOSED WITH LIMITATION** | Every favorable comparison under the candidate carries the access limitation and empirical scope. Absolute adequacy remains unestablished. `compare.go:563–569`; `lostdemand.go:376–435`. |
| **F7 — Weekly severity interpretation** | **CLOSED WITH LIMITATION** | Fixed windows replace outcome-dependent pooling; urgent is explicitly disabled locally. Boundary exemptions and acquisition-cohort exclusions remain material: G4. |
| **F8 — Invalid allowance fractions** | **CLOSED** | Clinical fractions must be finite, within their specified range and whole basis points. `policy.go:568–578`. |
| **F9 — Misleading weekly reporting** | **REGRESSION — reporting only** | Calendar spans are now reported, but four-week windows report four occupied weeks even when acquisitions occupy only one. Missing-pair absolute fractions also remain zero-valued rather than undefined. G6. |

**G1 — The “complete” protocol still leaves consequential decision rules open**

**Severity:** High.  
**Category:** Confirmatory-protocol specification defect.  
**Blocks confirmation:** **Yes.**

**References:** `docs/calibration/clinical-5c4f/reserved-validation-protocol.md:6–9,70–100`; incorporated `docs/calibration/clinical-5c4c/reserved-validation-protocol.md:102–115,125–134`.

T5 says acceptance at 2.1 and 3 points will be compared with an owner tolerance stated before execution. That tolerance is absent. The protocol nevertheless declares that every needed owner decision has been made.

**Concrete witness:** Suppose an observed near-margin false-acceptance rate is 31%. It fails a 5% tolerance and passes a 35% tolerance. The present document cannot distinguish those decisions.

This is not a hypothetical numerical concern about the former confidence bound. Under the current empirical rule, consider 40 seeds of 150 eligible urgent orders, baseline imaging all orders, and independent candidate terminal loss probability 0.021. Gate B passes when the total losses are at most 120:

\[
P\{\operatorname{Binomial}(6000,0.021)\le120\}
=0.314087989.
\]

This is **Gate B acceptance**, not a claim that the full evaluator passes every such sample. It is permissible behavior for an empirical rule, but its acceptability must be decided prospectively.

Further unresolved choices affect confirmation:

- T2 uses “probability ≥0.90” without clearly specifying whether the point estimate or a confidence bound decides.
- T6 does not define “3 Monte Carlo errors,” including whether the standard error uses the null rate or observed rate.
- “Simultaneous intervals” lacks a specified method and family.
- 2,500 seeds yield 62 complete groups of 40 and 20 remaining seeds; their treatment is not fixed in the current protocol.
- T7 requires counting every seed with insufficient window evidence. A seed also failing another gate may be classified simply FAIL by the joint aggregation; T7 cannot be reconstructed solely from the joint insufficient-seed list.
- “A missed target is the owner’s decision” does not specify whether confirmation failed, was inconclusive, or may be declared successful despite the miss.

For 1,000 seeds, even 25 detections in 25 groups have a one-sided 95% lower confidence bound of only **88.707%**. A point-estimate criterion is possible, but it must be named.

**Expected versus observed:** A prospectively locked protocol should determine the interpretation of every result without choosing tolerances or inferential conventions afterward. This document does not.

**Consequence:** The same reserved output could support different confirmation decisions without changing the clinical policy JSON.

**Minimum remediation:** Before execution, record the numerical T5 tolerances, decision estimators, uncertainty methods, group/remainder handling, T6 formula, raw T7 counting rule, and mandatory consequences of failed or inconclusive criteria. Any post-result waiver must remain a documented deviation, not successful prespecified confirmation.

The text prohibits changing values after results without consuming the family. That prohibition is useful, but does not fill the missing values or close discretionary interpretation.

**G2 — T2 confuses detecting harmed seeds with rejecting their evaluation**

**Severity:** High.  
**Category:** Clinical-policy/protocol inconsistency.  
**Blocks confirmation:** **Yes.**

**References:** `docs/calibration/clinical-5c4f/reserved-validation-protocol.md:55–58,73–87`; `internal/eval/lostdemand.go:253–255,461–487`; `internal/eval/evaluate.go:665–692`.

T2 requires at least 90% rejection for concentrated urgent loss of 15 percentage points on 10% of seeds. The approved allowance permits four failing seeds in 40.

**Concrete witness:** Let otherwise passing, equal-volume seeds independently suffer that harm with probability 0.10. Every harmed seed is correctly flagged. If \(K\) seeds are harmed:

- Mean deterioration is \(0.15K/40\).
- Four harmed seeds give only **1.5 points** mean deterioration.
- Gate B passes.
- Four safeguard failures are within the allowance.
- Existing case-selection per-seed failures can also remain within their separate allowance.

With the other gates passing, safeguard rejection occurs at \(K\ge5\):

\[
P\{\operatorname{Binomial}(40,0.10)\ge5\}
=0.370982303.
\]

If “10%” means exactly four harmed seeds per group, the safeguard never rejects that group.

**Expected versus observed:** T6’s requirement to flag at least 95% of sufficiently harmed seeds is compatible with the implementation. T2’s 90% whole-evaluation rejection is not implied by it and conflicts with the isolated allowance witness.

**Consequence:** Incidental failures elsewhere could make the reserved experiment appear to meet T2 without demonstrating the claimed protection against concentrated access harm.

**Minimum remediation:** Fix the shock-generation mechanism and distinguish seed flagging from evaluation rejection. Prospectively reconcile the detection criterion with the approved allowance. Either retain the policy and adopt an explicitly justified compatible target, or reopen the clinical-policy decision before locking. Do not tune either after observing reserved results.

**G3 — Several reserved scenarios lack a valid, reproducible construction**

**Severity:** High.  
**Category:** Protocol/model integration defect.  
**Blocks confirmation:** **Yes.**

**References:** `docs/calibration/clinical-5c4f/reserved-validation-protocol.md:28–30,39–64`; `internal/eval/oc_fidelity_test.go:489–520,1009–1012`; `internal/eval/lostdemand.go:220–239`; `internal/eval/seed_safeguard_test.go:228–242`.

The heterogeneous-volume scenario specifies **A85 against seasonal B85, paired demand**. Those are different demand-generating models.

`recordPaired` multiplies demand intensity by the seasonal factor when `Seasonal` is enabled. Sharing the random source does not preserve realized arrivals or requests after changing that intensity.

**Executed witness:** Generate one A85 record and one B85 record with the same debug demand stream PCG(1,0), the same booking stream PCG(1,1), and the aligned takeover. Gate B returned:

| Priority | Matched seeds | Unmatched seeds | Status |
|---|---:|---|---|
| Urgent | 0 | [0] | INSUFFICIENT |
| Soon | 0 | [0] | INSUFFICIENT |
| Routine | 0 | [0] | INSUFFICIENT |

The failure is demand mismatch, independently of the minimum number of seeds.

Other construction gaps remain:

- “Exactly 2.001 points per seed, made by losing orders at random” does not distinguish a loss probability from an exact realized fraction. Exact 0.02001 requires compatible integer denominators; ordinary seed cohorts generally cannot realize it.
- The offset-harm scenario does not specify a baseline with enough unimaged demand to permit a five-point improvement. That improvement cannot simply be subtracted from a near-zero loss fraction.
- Scenario indices and complete baseline mappings are not enumerated in the current protocol.
- The proposed booking namespaces overlap across scenarios: baseline stream \(2{,}000{,}000+100{,}000k+i\) equals candidate stream \(1{,}000{,}000+100{,}000(k+10)+i\). This is not a collision within one pair, but cross-scenario reuse is implicit rather than specified.
- The existing loss-injection helper rounds counts, takes the first eligible orders and changes only the end outcomes and record objects it touches. It is not by itself the locked random-injection, fully reconciled-record construction required by the protocol.

**Expected versus observed:** Heterogeneous volumes should vary across seeds while candidate and baseline retain identical demand within each pair. The named seasonal comparison instead changes within-pair demand.

**Consequence:** Reserved results could measure pairing failure, rounding choices, infeasible improvements or inconsistent records instead of the intended acceptance behavior.

**Minimum remediation:** Commit a deterministic scenario manifest and witness construction before execution: exact scenario indices, demand families, baseline mappings, stream allocation, injection estimands, integer/randomization rules, and record reconciliation. Use a shared demand realization within each candidate/baseline pair. Validate these constructions on small nonreserved synthetic fixtures first.

**G4 — Partial-window exemptions and the horizon leave material clinical coverage gaps**

**Severity:** Medium to high clinical limitation.  
**Category:** Clinical-policy limitation; not a demonstrated departure from the written cohort rules.  
**Blocks confirmation:** **Not independently for a calibration explicitly limited to these rules.** It blocks any broader claim that every local period or every eligible patient’s reporting satisfies the stated ceilings.

**References:** `internal/eval/clinical.go:178–201,271–288,442–546,707–735`; `docs/calibration/clinical-5c4f/severity-windows.md:55–80,158–163`.

**Executed witness G4.1 — Substantial partial-window harm with a full GO**

Use the pinned candidate unchanged, 40 synthetic paired seeds, aligned Tuesday takeover, 150-day horizon, and sufficiently populated complete windows. Each candidate seed has:

- 24 soon acquisitions in the leading partial week.
- 16 of those reports **30 hours past due**.
- 360 soon acquisitions in T90, giving **16/360 = 4.444%** severe.
- 240 soon acquisitions in the tail, none severe.
- No absolute reporting violation.

Observed:

- Leading partial week: **16/24 = 66.667% severe**, **NOT_JUDGED**.
- Every judged soon block passes.
- Per-seed severe gates pass.
- Clinical safety passes.
- Full comparison: **GO**, with the required access caveats.

Across 40 seeds this is **640 severely delayed soon reports** concentrated in exempt leading windows.

The per-seed rate limits the total burden; it does not provide the missing local ceiling. The absolute maximum limits individual duration; it does not reject substantial concentration below 48 hours.

**Executed witness G4.2 — One second changes whether an absolute violation is covered**

On every candidate seed, select one soon order wanted before the horizon. Keep acquisition within its seven-day access limit and report it **60 hours past due**.

| Acquisition | Clinical safety | Full verdict |
|---|---|---|
| Exactly at horizon \(H\) | FAIL | NO GO |
| At \(H+1\) second | PASS | GO |

The latter records remained valid; Gate A, backlog and other gates passed. The reporting rule excludes studies acquired after \(H\), even when their orders were eligible for access assessment.

This is a cohort boundary, not a one-second error in the absolute-delay comparator.

**Additional boundary conclusions:**

- Complete zero-acquisition blocks are INSUFFICIENT.
- Partial blocks below 100 are NOT_JUDGED, even with high severe fractions.
- Partial blocks reaching 100 are judged normally.
- A short-window unit witness with no complete block produced zero judged blocks and no window insufficiency. The local summary can therefore pass vacuously. The pinned full policy separately requires at least a 150-day horizon, preventing that three-day record from becoming a valid full evaluation.
- Acquisitions before the horizon are classified using final reports after the horizon, through observation.
- Unfinished acquired studies remain in the cohort. Already severe unfinished studies count as severe; undecided severe-rate evidence remains insufficient.
- The pinned 31-day post-horizon observation requirement exceeds the applicable reporting and access closure durations.

**Expected versus observed:** The implementation follows the partial-window exemption and acquired-by-horizon scope. It does not ensure a local 5% ceiling everywhere, nor zero absolute violations among all orders wanted by the horizon.

**Consequence:** A plan could systematically concentrate moderate severe delays in exempt edge windows while satisfying the aggregate rate. Moving acquisition beyond the horizon can also remove reporting from the hard safety cohort while preserving timely access.

**Residual-risk classification:** **Material limitation.** The exemption is disclosed, but the magnitude and horizon interaction deserve explicit presentation. Merely saying the studies remain in a per-seed rate is not evidence that local coverage is adequate.

**Minimum remediation:** Carry these exact scope limitations and adversarial outcomes into the locked protocol and eventual result. State whether these outcomes are accepted consequences of the intended policy. If broader protection is intended, that requires a separate prospective policy decision; this audit does not silently change thresholds or cohorts.

**G5 — The shared clinical allowance is not a universal four-seed allowance**

**Severity:** Medium clinical-interpretation limitation.  
**Category:** Deliberate policy composition, with misleading explanatory wording.  
**Blocks confirmation:** No, if described accurately.

**References:** `internal/eval/clinical.go:765–788`; `internal/eval/lostdemand.go:461–487`; `internal/eval/compare.go:265–301,487–495`; `docs/calibration/clinical-5c4e/seed-safeguard.md:49–54`.

The shared clinical allowance combines service, backlog, clinical access, severity windows, severe rates and the safeguard. Existing v4 case-selection allowances remain separate.

**Executed witness:** On 40 otherwise passing seeds:

- Seeds 1–4 lose nine additional urgent orders out of 150: **6 points** overall.
- Their T90 case-selection served drop exceeds 5%; four failures are allowed.
- None exceeds the urgent safeguard’s 10-point threshold.
- Different seeds 5–8 fail a soon local severity block.
- Clinical failures are exactly `[5,6,7,8]`.
- Gate B passes; case selection passes; the full verdict is **GO**.

Thus eight distinct seeds can have relevant adverse per-seed outcomes across these two allowance systems.

The safeguard explanation also says the two rules overlap at urgent deterioration of 5–10 points. That is incorrect: the safeguard does not flag at or below 10 points. In otherwise comparable served/terminal-loss examples, overlap occurs **above 10 points**.

**Expected versus observed:** One union is correctly implemented for the designated clinical gates. It is not a union over every v4 case-selection failure.

**Consequence:** “At most four failing seeds” is misleading unless its scope is specified.

**Minimum remediation:** Describe the separate case-selection allowance explicitly, correct the overlap explanation, and retain the disjoint-failure witness. A universal allowance would be a policy change, not a documentation repair.

**G6 — Reporting defects remain despite correct gate decisions**

**Severity:** Low.  
**Category:** Confirmed reporting implementation defects.  
**Blocks confirmation:** No, provided these fields are not used as acceptance evidence.

**References:** `internal/eval/clinical.go:539–543`; `internal/eval/lostdemand.go:220–269,437–450`.

**Executed witness G6.1:** Put 100 acquisitions in the first 50 hours of a complete four-week soon block. Only one calendar week is occupied. The report gives:

```text
worstWeeks = 4
worstOccupiedWeeks = 4
```

The elapsed span is correct; the occupied-week count is not. The assignment sets both fields to the configured span.

**Executed witness G6.2:** Remove origin evidence so no seed matches. Gate B correctly returns INSUFFICIENT, but both seed-mean and pooled absolute fractions are reported as `[0,0]`.

**Expected versus observed:** Occupied weeks should count occupied weeks. Unavailable absolute fractions should be distinguishable from measured zero loss.

**Consequence:** A reader or downstream consumer can misunderstand temporal concentration or missing access evidence. Neither witness caused favorable acceptance.

**Minimum remediation:** Count occupied weeks independently; represent unavailable fractions explicitly and distinguish matched-subset summaries from all-demand summaries.

**G7 — The all-zero confidence limitation is overstated as an impossibility for every dataset**

**Severity:** Medium statistical-exposition error; no empirical acceptance impact.  
**Category:** Statistical documentation limitation.  
**Blocks confirmation:** No for empirical Gate B.

**References:** `docs/calibration/clinical-5c4d/gate-b-statistics.md:56–77`; `docs/calibration/clinical-5c4e/empirical-acceptance.md:53–60`; `internal/eval/bounds.go:117–121`.

The all-zero-sample argument is valid. Its extension to “whatever the data” is not. Strong observed improvements can support an upper bound below the two-point margin with 40 seeds.

**Executed witness:**

```text
40 seed differences, each −0.5:
betting upper bound = −0.292615588928

40 seed differences, each 0:
betting upper bound =  0.207384411072
```

**Expected versus observed:** The all-zero bound describes identical observed outcomes. It is not the smallest bound over all possible samples.

**Consequence:** The current explanation understates what the research diagnostic can establish under strong improvement. This does not restore a population claim to empirical acceptance, and does not require such a claim.

**Minimum remediation:** Restrict the impossibility statement to the all-zero configuration and correct the diagnostic explanation. Preserve historical policy behavior unless separately authorized.

**Empirical Gate B and safeguard verification**

The implementation computes, separately for each priority,

\[
D_s=\frac{C_s-B_s}{N_s},
\qquad
\frac1S\sum_s D_s\le0.02.
\]

`meanAtMost` uses `big.Rat`, so unequal denominators receive equal seed weighting without floating-point boundary decisions. Pooled fractions are reported separately.

Verified behavior:

- Origin identity is exported directly from the simulator’s immutable demand key without a random draw: `internal/sim/export_eval_test.go:111–115,402–417`.
- Referral generation uses origin-keyed draws; booking and rebooking retain the existing order and original wanted date: `internal/sim/demand.go:470–537,657–742`.
- Walk-in generation is keyed independently, within configured opening hours: `internal/sim/flow.go:303–350`.
- Pairing checks receipt time, wanted time, modality and walk-in status within each priority.
- Missing baseline records, end captures, identities, changed requests, duplicate origins, thin seeds or too few seeds cannot produce Gate B PASS.
- Lost, left and patient-ended outcomes all count in terminal-unimaged demand.
- Open overdue orders are handled by Gate A rather than being counted as terminal losses.
- Exact mean deterioration of two points passes; one extra loss above that rational boundary fails.
- Diagnostic confidence, bound size and `marginEstablishedAtConfidence` do not enter the empirical status switch.
- Favorable verdicts explicitly disclaim population statistical confidence.

**Independent safeguard boundary witnesses:**

| Priority | No flag at | Flag at |
|---|---:|---:|
| Urgent | 100/1,000 = 10% | 101/1,000 |
| Soon | 60/1,000 = 6% | 61/1,000 |
| Routine | 50/1,000 = 5% | 51/1,000 |

The threshold derivation reproduces the committed development values. Its independence approximation is disclosed. The synthetic practice’s much smaller no-effect loss variability cannot establish a simulator false-flag guarantee.

**Independent offset-harm and overlap witness:**

Baseline loses 20 urgent orders per seed out of 150. Candidate loses 65 on harmed seeds and none elsewhere. Harmed seeds also fail a soon local severity block.

| Harmed seeds | Mean deterioration | Gate B | Clinical union | Full verdict |
|---:|---:|---|---|---|
| 4 | −9.000 points | PASS | Four failures; PASS | GO |
| 5 | −7.9167 points | PASS | Five failures; FAIL | NO GO |

Each harmed seed is 30 points worse than baseline. Improvements elsewhere offset the mean, but do not erase the safeguard flag. Overlapping local-severity and safeguard failures count once. The five-seed case also fails existing case selection; its clinical-union failure is independently visible.

**Severity configuration and exact boundaries**

The candidate contains the requested configuration.

| Priority | Severe when past due exceeds | Local rule | Per-seed severe ceiling | Absolute maximum past due |
|---|---:|---|---:|---:|
| Urgent | 12 hours | None | 5% | 24 hours |
| Soon | 24 hours | Fixed four-week blocks; minimum 100 | 5% | 48 hours |
| Routine | 72 hours | Monday–Sunday weeks; minimum 100 | 5% | 14 days |

References: `clinical.go:159–161,178–201,271–288,442–546,664–744`; `exact.go:35–40`.

- Equality at a severe-duration threshold is not severe; one second beyond is severe.
- Equality at an absolute maximum is permitted; one second beyond violates it.
- Absolute violation comparisons use seconds before display truncation to minutes.
- Five severe reports among 100 pass; six fail.
- A complete cohort of 99 is insufficient; 100 is judged.
- Sunday’s last second and Monday’s first second occupy different windows.
- Soon blocks are anchored prospectively and are not pooled or slid.
- Soon and routine per-seed severe rates are gated in T90 and tail; urgent uses its configured whole gated period.
- Any covered absolute violation overrides the allowance.

The disabled urgent local gate does not disable urgent severe-rate or absolute checks.

**Access Gate A verification**

References: `clinical.go:204–217,308–317,559–593,691–702`; `measure.go:336–378,529–557`; simulator export at `export_eval_test.go:402–417`.

The rule correctly requires at least 95% timely access within:

- Urgent: 48 calendar hours.
- Soon: seven calendar days.
- Routine: 30 calendar days.

The clock is the original wanted time. Rebooking and displacement do not restart it.

A served order exactly at the deadline is timely. An open order beyond the deadline is untimely; an open order still within it is undecided. Patient cancellation before the deadline is withdrawn from Gate A and counted by Gate B; cancellation exactly at the deadline is untimely. Lost and left orders are counted separately under Gate B.

Wanted-at-horizon orders are included; wanted-after-horizon orders are excluded. Inherited demand is treated separately. Missing and contradictory capture evidence invalidates the comparison.

A practice losing 65% of urgent demand can still have Gate A timeliness of 100% among the remaining served orders and zero Gate B deterioration against an equally poor baseline. The implementation correctly reports that absolute adequacy is **not established**. Its favorable-verdict lead statements prevent this result from being presented as adequate absolute access.

Gate A’s 95% is an observed acceptance threshold, not a confidence level.

**Clinical reliability and insufficient evidence**

References: `clinical.go:765–856`; `evaluate.go:665–692,719–795`; `compare.go:464–507`.

The designated clinical union allows at most four adverse seeds out of 40. Service, backlog, access, severity and safeguard failures on the same seed consume one place. Distinct failures consume distinct places.

Gate B is outside that allowance: one missing paired seed makes its priority insufficient and blocks favorable comparison. Covered absolute violations also override the allowance. Invalid records cannot be rescued by the allowance or favorable finances.

“INSUFFICIENT never becomes PASS” needs a precise level:

- An insufficient individual gate remains insufficient.
- Up to four seeds lacking specified clinical evidence can be tolerated by the joint rule.
- A local summary can pass when its only windows are NOT_JUDGED.
- Neither behavior establishes that the missing evidence passed.

These are specified aggregation rules, not evidence-completion mechanisms.

**Reserved-protocol assessment**

The policy digest and principal clinical thresholds are pinned correctly. The protocol correctly separates empirical acceptance from the research bound and does not require population coverage from Gate B.

T1 and T7 explicitly include insufficiency and set 5% ceilings. T3’s numerical rare-event criterion is coherent: zero violations in 2,500 gives a one-sided 95% upper bound of approximately **0.11976%**, below 0.13%. That remains a model-specific, per-limit result, not a simultaneous clinical safety guarantee.

The remaining prerequisites are substantive:

- T5 needs numerical false-acceptance tolerances.
- T2 needs a target consistent with the allowance and an explicit estimator.
- T4 needs a current expected-outcome manifest, rather than an unqualified import of historical witnesses whose policies changed.
- T6 needs harm labels, denominators and its Monte Carlo formula fixed.
- T7 must count raw insufficient windows even on otherwise failing seeds.
- Scenario construction, pairing, stream indices and failure interpretation must be executable without post-result choices.

The reserved family remained untouched during this audit. The archive contains documentary references to it, not evidence of an executed reserved calibration.

**Historical neutrality and activation**

`TestV4Neutrality` passed against its fixed historical evaluation digest:

```text
eef98fab6cd6355eb854b86675f914ecc8be02ba6cfcd5bd83f607dd98f0b070
```

The v4 policy digest remains:

```text
4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3
```

Old 5C.4d and 5C.4e policy compatibility tests passed, including their pinned bytes and retained confidence/pooled-window or calendar-week behavior.

Historical replay registrations through dev.19 were inspected at `internal/session/record_v1.go:194–212`, together with the replay fixtures and tests. Replay execution was not part of this audit, so this report does not claim independently rerun historical replay results.

The inspected identity export adds serialization of an existing key without drawing randomness. No new simulator behavior or randomness change was identified. Without the parent tree, that is source-path evidence rather than a complete revision-diff proof.

The active policy path remains v4 at `internal/eval/holdout.go:158`. The candidate explicitly says proposal, not in force and not frozen. No activation occurred.

**Reproduction record**

The 37 existing tests covered capture selection, invalid evidence, identity and thin cohorts, empirical margin and claims, concentrated/offset harm, cancellation accounting, severity windows, policy validation, shared allowance, absolute overrides, access boundaries, old-policy compatibility and v4 byte neutrality.

Independent witnesses were supplied through a Go overlay mapping an existing test file to an external augmented copy. Production source and repository test files were not edited. The eight witness functions covered:

1. Partial-window severe concentration with full GO.
2. Offset harm and overlapping failures.
3. Exact safeguard boundaries.
4. Seasonal demand-pairing failure.
5. Reporting at and immediately beyond the horizon.
6. Occupied-week reporting and a short local horizon.
7. Research-bound wording and missing absolute fractions.
8. Separate clinical and case-selection allowances.

Commands used `/usr/local/go/bin/go test ./internal/eval -run ... -count=1`, with scratch `GOCACHE`, `GOTMPDIR`, and the external overlay for independent witnesses. No broad test suite or environment-gated experiment ran.

**Required disposition**

**NO GO** for the reserved run until:

1. **G1:** Every acceptance tolerance and result-interpretation rule is numerically and procedurally locked.
2. **G2:** The concentrated-harm detection requirement is reconciled with the approved four-of-40 allowance.
3. **G3:** Every scenario has a valid paired-demand construction, exact injection specification, baseline mapping and stream manifest.

The partial-window, horizon and separate-allowance witnesses must accompany that lock as explicit limits on interpretation. They must not be described as protections the evaluator supplies.

The empirical framework is substantially implemented, but the proposed confirmatory experiment is not yet sufficiently specified to justify consuming its reserved stream. **Do not run reserved calibration or freeze v0.2-eval-5 at this revision.**