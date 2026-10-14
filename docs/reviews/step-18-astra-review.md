# Step 18 review — gpt-6-astra, high reasoning

* Reviewed: commit `b1a9ffb` (V0.1 step 18), read-only snapshot copied to a
  temporary directory (deleted afterwards).
* Command: `codex exec -m gpt-6-astra -c model_reasoning_effort="high" -s read-only`
* Prompt: the owner's calibration and gameplay-balance review brief
  (parameter realism, bottlenecks, walk-in policy exploits, extension
  economics, worklist policies, cross-seed robustness, financial realism,
  gameplay quality, golden and replay integrity), with a note that the
  review host has no Go toolchain.
* The findings below are the model's output, unedited. File links point at
  the temporary snapshot; line numbers match `b1a9ffb`.

## Findings (verbatim)

**CONDITIONAL GO for V0.1 completion.** The evidence credibly describes a struggling, capacity-constrained practice. Failure to reach the former positive-margin target is not a defect.

Completion should wait for fixes to extension validation, shared-staff booking feasibility, and walk-in accounting; correction of the calibration report’s metric interpretations; and targeted reproduction of the affected strategies. No critical defect was identified, but the current implementation is not exploit-free.

This review used the supplied source, documents, calibration data, and golden references. No files changed. Tests, simulator runs, and current-version replay were not executed because no Go toolchain is available. The snapshot lacks Git metadata, so historical claims about unchanged parameters could not be independently verified.

**A. Confirmed defects, ranked**

**1. High — A positive extension value can overflow into a negative duration.**

Reference: [actions.go:141](/tmp/dt-review.ZydY0S/internal/sim/actions.go:141), [actions.go:185](/tmp/dt-review.ZydY0S/internal/sim/actions.go:185), [api.go:1139](/tmp/dt-review.ZydY0S/internal/api/api.go:1139).

- **Trigger:** At takeover, submit `extend_shift` for Jordan, E01, with `minutes: 4611686018427387903`.
- **Evidence:** Multiplication by 60 occurs before validation. This positive `int64` becomes a duration of **−60 seconds**. It passes both `Minutes < 1` and `d > MaxExtension`. Desk coverage remains an accepted extension effect, and the action moves the shift end backward.
- **Consequence:** Repeated actions can shorten coverage and paid time through an action intended only to extend shifts. More extreme wrapped values can also reach invalid event times. The arithmetic and acceptance path are confirmed; the resulting financial effect or panic needs execution. The normal interface’s bounded input does not protect JSON/API actions.
- **Smallest correction:** Validate the raw minute value against `1..MaxExtension/Minute` before multiplication.
- **Regression:** Exercise zero, negative values, 241, `MaxInt64`, the value above, and valid boundaries through validation and application. Invalid actions must leave coverage, employee state, bookings, and financial state unchanged.

**2. Medium — Shared-staff booking rejects feasible capacity and depends on booking order.**

Reference: [demand.go:251](/tmp/dt-review.ZydY0S/internal/sim/demand.go:251), [established.json:45](/tmp/dt-review.ZydY0S/scenarios/base/established.json:45).

- **Trigger:** Extend Riley on a future Tuesday–Friday through 19:45. At 18:00, XR can use Riley or Sam; CT can use Sam. Book XR 18:00–18:15 before attempting CT 18:00–18:30.
- **Evidence:** `sharedFree` counts the XR appointment against CT because XR’s possible-staff set contains Sam. It rejects CT despite the feasible assignment **XR→Riley, CT→Sam**. Booking CT first allows the overlapping XR appointment because XR has two possible technologists.
- **Consequence:** Paid additional coverage can fail to create usable booking capacity. This weakens conclusions about Riley’s extension and the added-technologist experiment. The size of the throughput effect requires rerunning them.
- **Smallest correction:** Check whether simultaneous appointments can be assigned to qualified staff. Evaluate appointment boundaries as well as coverage boundaries; counting every appointment intersecting a long segment is insufficient.
- **Regression:** Test both booking orders, two consecutive XR appointments overlapping one CT appointment, coverage handovers, and the genuinely infeasible single-technologist case.

Actual imaging dispatch still requires an available technologist; this defect suppresses capacity rather than establishing simultaneous use of one employee.

**3. Medium — Walk-in conversion can distort reported acceptance and lost-demand rates.**

Reference: [flow.go:334](/tmp/dt-review.ZydY0S/internal/sim/flow.go:334), [metrics.go:219](/tmp/dt-review.ZydY0S/internal/sim/metrics.go:219), [metrics.go:260](/tmp/dt-review.ZydY0S/internal/sim/metrics.go:260), [demand.go:409](/tmp/dt-review.ZydY0S/internal/sim/demand.go:409).

- **Trigger:** Decline walk-ins today, successfully book them for another day, then measure that return day.
- **Evidence:** A returning scheduled patient retains `o.walkIn`. Starting that patient’s exam increments `WalkIns.Imaged`, although their original arrival occurred outside the reporting period. `Acceptance = Imaged/Arrived` can therefore exceed 100%. Waiting time starts at the return visit, hiding the intervening appointment delay.
- **Related inconsistency:** Failed walk-in booking offers increment modality `Lost`, but walk-ins do not increment modality `Received`. `LostRate` therefore mixes populations. Rebooking losses can also refer to earlier arrival cohorts. Additionally, `walkInsBooked` increments before an appointment offer succeeds, contrary to the API’s “sent away with an appointment” description.
- **Smallest correction:** Separate same-visit walk-in service from scheduled conversions; distinguish booking attempts from accepted appointments; calculate access rates on a consistent arrival cohort, with pending outcomes explicit.
- **Regression:** Cover cross-day returns, failed offers, periods with no new walk-ins, and rebooking. Assert bounded cohort rates and exactly one completed exam and claim per order.

This is a confirmed **reporting exploit**. It does not establish duplicate throughput or revenue.

**4. Medium — Several report conclusions overstate what the measurements establish.**

Reference: [step-18-report.md:240](/tmp/dt-review.ZydY0S/docs/calibration/step-18-report.md:240), [step-18-report.md:298](/tmp/dt-review.ZydY0S/docs/calibration/step-18-report.md:298), [calib_test.go:213](/tmp/dt-review.ZydY0S/internal/sim/calib_test.go:213), [calib_test.go:284](/tmp/dt-review.ZydY0S/internal/sim/calib_test.go:284).

- **Trigger:** Interpret the report’s loss explanation, lead times, or TAT labels as reconciled causal measurements.
- **Evidence:** “Where the 330 a month go” adds approximately **372**, and mixes arrival-cohort losses with period activity. No-shows are not all terminal losses because some rebook. Exams also include inherited orders, while new arrivals can remain booked beyond the window.
- **Further evidence:** The reported US lead time measures receipt to the **first booked appointment**, including requested lead time; it does not isolate capacity-induced delay. The approximately 32-minute TAT is **all-priority**, not routine-only. Shortest-first reduces median TAT from 34 to 22 minutes—about 35%, not half.
- **Smallest correction:** Correct labels and arithmetic; describe losses separately until an order-level reconciliation exists; distinguish requested lead from additional booking delay; qualify the shortest-first conclusion.
- **Regression:** Use a hand-checkable cohort containing inherited orders, future bookings, cancellations, no-shows, rebookings, and walk-in conversions. Reconcile beginning outstanding orders plus arrivals against completed, terminally lost, and ending outstanding orders.

These errors weaken the explanation, not the underlying observation that throughput is below demand.

**5. Low — Strategy calibration silently tolerates rejected extension actions.**

Reference: [calib_actions_test.go:30](/tmp/dt-review.ZydY0S/internal/sim/calib_actions_test.go:30), [calib_actions_test.go:99](/tmp/dt-review.ZydY0S/internal/sim/calib_actions_test.go:99), [calib_actions_test.go:125](/tmp/dt-review.ZydY0S/internal/sim/calib_actions_test.go:125).

- **Trigger:** Strategies B and D encounter Morgan’s Saturday shifts on October 24 and November 7.
- **Evidence:** The helper promises weekday extensions but iterates all rostered days. It requests an extension from Saturday’s 18:00 close toward 20:00; production validation rejects it. Application results are ignored. Rejection counts are collected but neither asserted nor printed.
- **Smallest correction:** Generate only intended valid actions and assert acceptance, or explicitly report expected rejections.
- **Regression:** Check these Saturdays and representative weekdays; make any unexpected rejection fail the experiment.

The rejected Saturday actions do not themselves increase capacity or expense.

**B. Risks needing reproduction**

**6. Medium — “Shortest-first at no cost” is not established for urgent or long studies.**

Reference: [reading.go:98](/tmp/dt-review.ZydY0S/internal/sim/reading.go:98), [metrics.go:390](/tmp/dt-review.ZydY0S/internal/sim/metrics.go:390), [established-seed1-90d.json:2222](/tmp/dt-review.ZydY0S/testdata/golden/established-seed1-90d.json:2222).

- **Trigger:** Select shortest-first during sustained arrivals, particularly near reading-block or shift boundaries.
- **Evidence:** Only `urgent_then_fifo` applies urgency ranking. FIFO and shortest-first have no urgent override, as documented. Shortest-first uses catalog expected duration, so it can repeatedly favor short routine work over long urgent work.
- **Existing warning:** Even the default seed-1 golden has **19/118 urgent studies over their one-hour target**, urgent p90 approximately **12.1 hours**, and maximum approximately **41 hours**. This does not prove a new policy defect; it shows why aggregate medians are insufficient.
- **Smallest correction now:** Withdraw the unqualified “at no cost” claim and report paired priority/modality tails, unfinished counts, and oldest outstanding age.
- **Regression/reproduction:** Run default, FIFO, and shortest-first against the same arrivals at 30 and 90 days. Include a controlled long urgent study competing with continuing short routine arrivals.

Acquisition-cohort TAT includes unfinished studies at their current age, which prevents the simplest completed-only exclusion trick. Nevertheless, a median can improve while a small subgroup deteriorates severely. No policy change is justified until that tradeoff is measured.

**7. Medium — Sensitivity experiments lack enough saved provenance to verify their numerical conclusions.**

Reference: [calib_test.go:122](/tmp/dt-review.ZydY0S/internal/sim/calib_test.go:122), [step-18-report.md:251](/tmp/dt-review.ZydY0S/docs/calibration/step-18-report.md:251), [step-18-report.md:265](/tmp/dt-review.ZydY0S/docs/calibration/step-18-report.md:265).

- **Trigger:** Reproduce the three-added-technologist, faster-XR, or reduced-reading-allocation experiments.
- **Evidence:** The overlay mechanism exists, but the reviewed calibration artifacts do not preserve those exact overlay inputs and per-seed outputs. Baseline output also lacks embedded engine/scenario/overlay fingerprints.
- **Smallest correction:** Preserve the resolved inputs, fingerprints, seeds, horizons, and per-seed results for each cited experiment.
- **Regression/reproduction:** Reapply each saved overlay twice, verify identical canonical configuration and results, and rerun the staffing experiments after finding 2.

The reported direction is plausible; the precise **7% throughput gain and −18.7% margin** remain unverified sensitivity results.

**C. Modeling limitations**

**8. Medium — Financial evaluation is sensitive to posting boundaries.**

Reference: [evaluation.go:97](/tmp/dt-review.ZydY0S/internal/sim/evaluation.go:97), [evaluation.go:233](/tmp/dt-review.ZydY0S/internal/sim/evaluation.go:233), [finance.go:273](/tmp/dt-review.ZydY0S/internal/sim/finance.go:273), [finance.go:372](/tmp/dt-review.ZydY0S/internal/sim/finance.go:372).

- **Trigger:** Evaluate a partial final day or compare strategies while an hourly shift is still underway.
- **Evidence:** Evaluation allocates the remaining takeover-day fixed/salary expense into the window, but includes the entire final day’s midnight accrual without an equivalent end adjustment. With constant daily expense `F`, a 06:00→06:00 one-day window contains `0.75F + F`. Hourly wages post only when the employee goes home.
- **Consequence:** A strategically timed extension can postpone recognition of the employee’s whole shift expense beyond an intraday snapshot. This is temporary reporting advantage, not avoidance of eventual wages. Published 06:00 endpoints avoid active-hourly-shift exposure, but retain the final-day fixed/salary timing effect.
- **Smallest correction:** Clearly label the existing posting convention. If arbitrary-time profit is a gameplay score, add symmetric endpoint allocation and accrued wages for ongoing shifts.
- **Regression:** Compare equivalent 24-hour windows and snapshots immediately before/after shift end, including an extension crossing the endpoint.

This follows the documented accounting convention rather than demonstrating a broken ledger. Correcting it would not make the baseline profitable.

**9. Low — Walk-in patience has a discontinuity during registration.**

Reference: [flow.go:368](/tmp/dt-review.ZydY0S/internal/sim/flow.go:368).

- **Trigger:** A patient’s single patience deadline occurs while registration is underway.
- **Evidence:** The deadline becomes stale; the patient subsequently remains even if the post-registration wait is long.
- **Assessment:** Documented simplification, not an implementation mismatch. It makes abandonment depend on whether registration happens to straddle one instant.
- **Smallest correction:** No V0.1 behavior change is required; explain the rule. Consider a remaining-patience or renewed-deadline model later.
- **Regression:** Compare otherwise identical patients whose registration finishes just before versus just after the deadline, preserving the intended rule explicitly.

**D. Optional future enhancement**

**10. Low — Decision previews could better explain costs and reading policies.**

Reference: [actions.go:197](/tmp/dt-review.ZydY0S/internal/sim/actions.go:197), [api.go:415](/tmp/dt-review.ZydY0S/internal/api/api.go:415).

- **Trigger:** A player chooses an extension or shortest-first policy.
- **Evidence:** Extension estimates are approximate wage estimates, not full incremental operating expense including benefits and possible subsequent weekly-premium effects. Catalog expected reading times drive shortest-first but are not included in `WorklistItem`.
- **Smallest enhancement:** Label the wage estimate’s scope, show benefit-inclusive estimates where practical, and expose the same catalog expected duration used by the policy.
- **Regression:** Check previews below and across the weekly overtime threshold; changing hidden case-duration draws must not change displayed expected read times.

**Parameter realism and bottlenecks**

The parameter set is coherent as an illustrative scenario, but the calibration is not external validation of a real practice’s rates.

| Area | Assessment |
|---|---|
| Demand and referral mix | Six sources have distinct specialty and priority mixes. Weekday and seasonal factors explain changing volume. Fixed source behavior and no reputation feedback are reasonable V0.1 simplifications. |
| Appointment demand | Requested lead times extend far into the future, alongside morning/afternoon/evening preferences. Therefore substantial receipt-to-appointment lead is intentional, not wholly congestion. |
| Durations | Imaging, documentation, registration, and reading are separate resource demands. Comparing room time alone with available technologist hours understates workload. |
| Attendance | Cancellation, rebooking, and lead-dependent no-show rules are internally consistent. They require attempt-level versus patient-level distinctions when interpreting losses. |
| Staffing | Shared coverage, breaks, registration fallback, and documentation explain substantial technologist occupancy alongside idle rooms. |
| Reading | Allocated reading blocks and short catalog reading times plausibly produce short median TAT at this volume. Block timing and closing gaps explain why tails can remain poor. |
| Reimbursement/costs | The payer mix, expected collections, 25% benefits, supplies, IT, billing fees, and fixed expenses are explicit. They are scenario assumptions, not demonstrated market estimates. |

The configuration evidence is concentrated in [established.json:99](/tmp/dt-review.ZydY0S/scenarios/base/established.json:99), [established.json:198](/tmp/dt-review.ZydY0S/scenarios/base/established.json:198), and [established.json:259](/tmp/dt-review.ZydY0S/scenarios/base/established.json:259).

**XR/CT:** The principal explanation is credible. Technologists perform documentation and sometimes registration while the room is idle, and shared coverage permits only one active task per employee. Imaging dispatch prioritizes urgent patients, due appointments, late fitted-in patients, walk-ins, then early appointments. This disadvantages walk-ins under load by design. Finding 2 adds an unintended booking constraint.

**US:** One technologist, restricted hours, category-specific slots, alternating Monday coverage, and patient tolerance can produce substantial losses. The report’s “45-minute slots” description is incomplete: the regular weekday template mixes seven 45-minute and five 30-minute slots. The quoted utilization ceiling is consequently an approximation, not a hard derived bound.

**MR:** Nine regular-day slots are directly supported by the template. Four later noncontrast slots exist but require extended coverage. Extending a shift cannot invent other categories or additional templates.

**Slot utilization:** `BookableSec` sums individually roster-covered room slots without deducting shared-staff conflicts. Low fill therefore does not establish equivalent unused simultaneous capacity. See [metrics.go:340](/tmp/dt-review.ZydY0S/internal/sim/metrics.go:340).

**Radiologists:** Spare aggregate reading capacity is credible. The precise “145% capacity” interpretation is less secure: block utilization counts reading inside blocks, while work can finish outside them. Capacity, block utilization, and urgent-service availability should remain separate concepts.

**Walk-in policy and extension economics**

The decline-walk-in throughput increase has a legitimate mechanism:

1. A walk-in arrives and receives one order.
2. Policy declines immediate service.
3. The normal appointment search starts from the following day.
4. The patient accepts an available date within tolerance or becomes lost.
5. An accepted booking subsequently follows ordinary attendance behavior.
6. The returning scheduled patient receives appointment priority instead of competing as a walk-in with a short patience deadline.

No special booking bump or duplicate order/study/revenue path was found. Different preferences and the scheduled-versus-walk-in queue position explain the advantage. Repeated policy switching does not itself repeatedly book the same patient.

Thus **more exams after declining immediate service is not inherently an exploit**. It trades immediate access for scheduled access. Finding 3 makes the reported access comparison unreliable.

For ordinary, bounded positive extensions, the implementation checks dates, roster presence, maximum shift length, coverage, and qualifications; updates existing slot eligibility; and pays actual elapsed labor less taken breaks. Repeated extensions accumulate against the shift cap. No duplicate-slot creation or permanent hourly-pay avoidance was found. Salaried extensions having no incremental wage expense, and supervision being compatible with reading, are explicit simplifications.

The action results describe meaningful tradeoffs:

- **B, US/MR extensions:** approximately 68 more exams and $5,000 better profit over 30 days.
- **C, Riley extension:** approximately 31 more exams but worse profit; finding 2 may suppress its benefit.
- **D, combined extensions:** highest throughput, with additional labor consuming much of the financial gain.
- **E, scheduled walk-in conversion:** approximately 31 more exams without better aggregate profit; access changes materially.
- **F, shortest-first:** improves aggregate TAT, with subgroup costs insufficiently measured.

These choices are not universally dominated because throughput, immediate access, urgent service, and profit differ. `accept_always` having little baseline effect is also understandable when physical staffing is already the constraint.

**Financial credibility**

The core accounting is well structured: expected net revenue is recognized at report finalization; supplies at acquisition; later collection differences adjust expected revenue; cash settlements remain separate. Payroll and payable timing explain cash/profit divergence. Inherited collections do not become newly earned operating revenue.

Across all **18 stored seed/window results**, operating profit equals net revenue minus operating expense, and ending cash reconciles to opening cash plus collections minus disbursements. The extra takeover allocation also explains why reported operating expense exceeds the displayed labor/fixed/variable subtotal.

The three-technologist experiment’s weak financial return is economically plausible: labor is paid for additional coverage, while attendance, fixed templates, category mix, and finite demand limit additional completed reports. Full wages and benefits can readily exceed the contribution from a 7% throughput increase. However, the missing overlay provenance and shared-booking defect prevent accepting that exact experiment as a clean measurement.

No evidence supports raising reimbursement merely to restore the retired profitability target. A future profitable preset also need not wait for dynamic referral-growth gameplay: higher static demand is already configurable.

**Cross-seed robustness**

Recalculation of the stored baseline supports the report’s broad characterization:

| Metric | 30-day window | 90-day window |
|---|---:|---:|
| Demand/month, mean | 1,731 | 1,646 |
| Exams/month, mean | 1,381 | 1,314 |
| Exams/month, seed range | 1,350–1,401 | 1,293–1,335 |
| Operating margin, mean | −12.50% | −14.04% |
| Margin, seed range | −16.90% to −11.06% | −15.60% to −13.01% |
| Referral-cohort lost bookings/month, mean | 131 | 127 |
| Mean of seed TAT p90s | 306 min | 122 min |

The 90-day demand and exam totals vary modestly; the struggling-practice conclusion is robust within this parameter set. Lost bookings and abandonment vary more.

The 30-day TAT tail is materially less robust: seed p90s range **119–665 minutes**, and their median is **141 minutes**. Reporting only their mean of 306 minutes obscures the split. Longer observation reduces this variation, but the existing evidence does not isolate seasonality from horizon and calendar effects.

Warm-up testing is useful: [warmup_test.go:388](/tmp/dt-review.ZydY0S/internal/sim/warmup_test.go:388) compares nine-seed 180-day and 365-day profiles and includes a shorter-warm-up failure control. That supports appointment/A/R initialization, not convergence of every policy outcome.

The characterization test is a meaningful broad smoke test, not a balance certificate. Its demand, exam, margin, abandonment, and reconciliation bounds can catch gross changes. However, “US loses the most demand” checks only US versus XR, and “reporting keeps up” permits poor urgent tails. Its morning sample also includes days beyond the report’s Tuesday–Friday population. See [calibration_sanity_test.go:30](/tmp/dt-review.ZydY0S/internal/sim/calibration_sanity_test.go:30).

More random seeds alone are not the highest-value next step. Targeted boundary tests, paired strategy deltas, and small demand/read-time/patience sensitivity changes would improve confidence more.

**Golden, replay, and observation integrity**

The mechanisms inspected are sound:

- Older-engine goldens are not silently relabeled. Engine mismatch is reported; explicitly allowing it still compares outcomes and digests. See [result.go:335](/tmp/dt-review.ZydY0S/internal/session/result.go:335).
- Golden tests request replay verification, including when allowing engine mismatch. See [golden_test.go:46](/tmp/dt-review.ZydY0S/cmd/dreamteam/golden_test.go:46).
- Calibration experiments are build-tagged test code; overlay handling does not enter ordinary simulation builds.
- Scenario merging is deterministic for a fixed input. Array replacement versus ID-based merging must be respected when preserving experiments.
- Reading policy uses catalog expected duration, while actual duration is independently keyed to the case. No hidden future outcome entering ordinary player observations was found.
- The stored seed-1 baseline financial and throughput totals agree with the existing 90-day golden.

**Current dev.18 golden validity, replay execution, and absence of historical parameter drift remain unconfirmed here.** They require a Go-enabled run and, for drift, comparison with the preceding repository revision. Fixes affecting simulation behavior should produce explicitly reviewed golden changes.

**Completion recommendations**

For **V0.1**, prioritize:

1. Overflow validation and flexible shared-staff booking tests.
2. Cross-window walk-in accounting and order-level loss reconciliation.
3. Correct report labels and assert strategy action acceptance.
4. Reproduce urgent/modality TAT tails and affected staffing strategies.
5. Run current-version golden, replay, and accounting checks; preserve exact experiment inputs.

For **V0.2**, improve decision previews, access-cohort reporting, arbitrary-time financial scoring, patience behavior, and the separately designed profitable preset.

The calibration is credible enough to retain the default scenario and its struggling-practice identity. It is not yet strong enough to certify the implementation as exploit-free or every reported intervention as correctly valued.
## Assessment

Verdict accepted: CONDITIONAL GO. Checked against the code:

| # | Finding | Verified | Note |
|---|---|---|---|
| 1 (high) | `extend_shift` minutes overflow | **Confirmed** (`actions.go:141`): `Minutes × 60` is computed before the range check, so 4,611,686,018,427,387,903 wraps to −60 s and passes `Minutes ≥ 1`, `d ≤ 240 min`. Reachable from JSON actions and the CLI actions file; the TUI's ±15 steps cannot reach it. |
| 2 (medium) | Shared-staff booking feasibility | **Confirmed** (`demand.go:251`): `sharedFree` counts every overlapping appointment whose possible staff intersects, without checking for an assignment. In the default roster XR and CT share no one 11:30–16:15 (XR Riley, CT Sam), so it bites only once Riley is extended past 16:15 (XR Riley+Sam, CT Sam) — strategies C and D. |
| 3 (medium) | Walk-in conversion counting | **Confirmed**: `noteExamStart` counts any exam of a walk-in order as a walk-in imaged, including the scheduled return visit (`metrics.go:227`); walk-in booking losses are counted in modality `Lost` while walk-ins are not in `Received`; `walkInsBooked` counts offers, not accepted appointments. Reporting only; no duplicate exam, claim or revenue. |
| 4 (medium) | Report arithmetic and labels | **Confirmed**: the loss decomposition adds to ≈372, not 330, and mixes cohorts; lead time is receipt to first appointment (includes requested lead); the ≈32-minute TAT is all priorities; shortest-first cuts the median ≈35%, not half. |
| 5 (low) | Strategy helper extends Saturdays | **Confirmed**: `extendTo` iterates every rostered day; Saturday extensions are rejected (`after_close`) and the rejections were collected but not shown. No effect on the reported capacity or cost. |
| 6 | Shortest-first urgent tail | Risk; to reproduce with priority and modality tails. The default's own urgent tail (19/118 over target, p90 ≈12 h in the seed-1 90-day golden) is worth reporting regardless. |
| 7 | Experiment provenance | Accepted: the cited overlays and per-seed results should be saved. |
| 8 | Endpoint accrual asymmetry | Modeling limitation as documented (takeover-day allocation; the end of the window takes the posted accruals); label it. |
| 9 | Patience during registration | Documented simplification. |
| 10 | Preview scope | Future enhancement. |

The reviewer confirms the calibration's core characterization (struggling,
capacity-constrained; 90-day exams 1,293–1,335, margin −15.6% to −13.0%
across seeds), the walk-in mechanism as legitimate (no duplicate booking or
revenue path), the extension economics for ordinary inputs, the accounting
reconciliation over all 18 stored seed-windows, and the golden and replay
mechanisms.

## Resolution

Owner direction: keep step 18 at CONDITIONAL GO until the defects are
fixed and the experiments rerun; no redesign. Resolved in 0.1.0-dev.18.1.
Each defect got a regression test first, confirmed failing on `b1a9ffb`,
then the smallest fix.

| # | Resolution | Regression test (failure on the old code) |
|---|---|---|
| 1 | `extend_shift` validates the raw minutes (1–240) before any conversion; the preview's result time is the engine's now. | `TestRegress18ExtensionMinutesRange`: MaxInt64 and 4,611,686,018,427,387,903 were accepted as −1 min (Jordan's shift moved to 16:14). Cases 0, negatives, MinInt64, 240/241 included. |
| 2 | `sharedFree` is an actual feasibility check: appointments are cut at coverage boundaries (handover only there), each piece needs a qualified technologist on its segment, overlapping pieces distinct ones; memoized search over the cluster linked by shared staff. Dispatch unchanged. Default outcomes identical to `b1a9ffb` (ledger and visit/study/appointment hashes, three seeds per scenario); warm-up time unchanged (≈1.5 s). | `TestRegress18SharedStaffFeasibility`: failed on "XR then CT", "two XR then CT" and "handover after 17:00"; both booking orders, consecutive XR over one CT, one-tech infeasible cases included. |
| 3 | Walk-in arrival, same-visit admission, offer (declined by policy or unservable), offer booked or failed, same-visit exam and return exam are separate counters; walk-in-origin booking events (`demand.walkInOrigin`) are kept out of referral demand; acceptance and abandonment are over the cohort of walk-ins who arrived in the period, each classified once as of its end (imaged same visit, converted, left, lost, on site). No clamping. | `TestRegress18WalkInCohorts` (cohorts checked against an independent classification from visits and appointments; cross-day returns, failed offers, a week with no new walk-ins, mid-day on-site walk-ins, rebookings, inherited orders, exactly one exam/study/claim per completed order). On the old code the same seed showed `imaged` counting return exams on 4 of 21 days and acceptance 2/0 on a day with no walk-ins. `TestRegress18FlowCountsFieldsFolded` guards every counter through add/sub. |
| 4 | Report corrected (`docs/calibration/step-18-report.md`, marked [18.1]): order-cohort reconciliation replaces "330"; lead split into requested lead and capacity delay; TAT labeled all-priority, routine reported separately; shortest-first ≈35%. | `TestRegress18OrderCohortReconciles`: one-week mini cohort, received = completed + ended + outstanding by origin, one exam per completed order. |
| 5 | Strategy extensions stop at the day's scheduled close; the harness fails on any rejected action. | `TestCalibrationActions` / `calibrate`: on the old helper, two Saturday `after_close` rejections in each of B and D (seed 1), none now. |
| 6 | Urgent TAT analysed (report section "Urgent turnaround"): paired 30/90-day runs of default, FIFO and shortest-first, by priority and modality. The default's urgent tail is mostly a reading-coverage gap (≈¾ of breaches wait overnight), identical across policies; FIFO adds ≈15 in-hours urgent breaches per 90 days; shortest-first ≈2, but delays long urgent studies (urgent MR p50 99 vs 52 min) and starves them when short routine work never runs out. No policy semantics changed in 18.1; owner decision then made urgency take precedence under every policy (0.1.0-dev.18.2, a gameplay rule change). | `TestRegress18UrgentLongStudyVsShortRoutine` pins the trade-off. |
| 7 | Every experiment is a manifest in `docs/calibration/experiments/` with results (engine, fingerprints, overlay hash, per-seed, aggregates) in `docs/calibration/results/`. Rerun after the fixes: conclusions unchanged (three techs −18.6% vs −18.7%; C, D within one exam and 0.1 margin points). | `go test -tags calibration ./internal/sim -run 'CalibrationExperiments$'` |
| 8–10 | Documented limitations; unchanged. | — |

Goldens: all four regenerated. Simulation outcomes unchanged (log, ledger
and action digests identical); the state digest changed (new order and
counter fields in the canonical state walk) and with it the run digest;
the result JSON changed in walk-in fields, referral `lost`/`booked`
(walk-in-origin events moved out) and the engine name. Scenario
fingerprints unchanged (`b1cbc2a0…`, `bc160655…`).
