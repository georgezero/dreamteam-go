# V0.2 evaluation framework (Slice 5C.0)

This directory freezes how V0.2 strategies are judged, before any
staffing, reading-allocation or teleradiology strategy is compared. It is
a preregistration: the rules were written from the owner's decisions, the
design (`V0.2_DESIGN.md` §13) and the arithmetic of the gates. No
candidate strategy was run to set them. The no-action baseline's
acquisition volumes were read once, from the archived development
results, only to confirm that the evidence minimums can be met.

| File | What it is |
|---|---|
| `policy-v0.2.json` | The machine-readable policy (`dreamteam.evalpolicy/v1`), decoded strictly by `internal/eval`. Its SHA-256 is the policy digest every report carries. |
| `holdout-v0.2.json` | The locked holdout manifest. It holds the method and a digest, not the seeds. |
| `holdout-unseal.json` | Not present. Committing it, with the owner's approval, is what unseals the holdout. |

The code is `internal/eval`, a pure package that reads records and never
a simulation:

* `Measure` turns one seed's record into its measures.
* `Evaluate` applies the policy to a set of plans and gives the gates,
  paired comparisons and verdicts.

The recorder that produces records from a run is test-only, in
`internal/sim/export_eval_test.go` (`World.NewEvalRecorder`). Recording
reads state only: the integration test checks that a recorded run's state
digest equals an unrecorded one's.

Rules marked **[proposed]** fill a gap the design left open. They are
conservative explicit choices, frozen here so that nothing is chosen
after results exist. The owner should confirm or change them before the
holdout is unsealed (see *Open decisions*). Values marked **[owner]** are
the owner's decisions and are unchanged.

## 1. Horizons and windows

| Window | Span | Role |
|---|---|---|
| T30 | (t0, t0+30 d] | Feedback. Informational. |
| T90 | (t0, t0+90 d] | **Primary** [owner]. Gated. |
| D31–90 | (t0+30 d, t0+90 d] | Operating performance after the transition. Informational. |
| Tail | (T90, tailEnd] | The frozen-policy tail. Gated. |
| T180 | (t0, t0+180 d] | **Validation**, labelled separately [owner]. Part of the GO condition, never merged into T90. |

* t0 is the takeover instant (Tuesday 2026-10-13 06:00). Every window is
  half-open, (a, b]: an event exactly at a window's end belongs to it,
  and one exactly at its start does not. This is the existing
  evaluation-window convention (`internal/sim/evaluation.go`).
* **Tail** [owner]: `tailEnd = max(T90 + 60 d, latestAcceptedEffect + 30 d)`.
  Demand, dispatch, accrual, collections and settlement continue as
  normal, and no action is accepted. A record with an action at or after
  T90 is invalid. Future obligations stay on the books: contract minimums
  and vendor bills are listed at every capture.
* **Closure** [proposed]: each cohort is observed for 1,440 minutes (the
  longest TAT target) after its window ends. That decides every study's
  breach status (§3). A gated window observed for less than this is an
  integrity failure, and its undecided studies are insufficient evidence.
* **Daily samples** are taken at the takeover's clock time (06:00), days
  1, 2, … from t0. That is before the day's reading starts and after the
  00:00 settlement, accrual and payment transaction. The clock time is
  the same every day, so samples are comparable across weekdays.

## 2. Service reliability gates

The breach gates are the owner's provisional T90 gates of 2026-10-15,
unchanged [owner]:

| Priority | Breach rate gate |
|---|---|
| Urgent | ≤ 12% |
| Soon | ≤ 32% |
| Routine | ≤ 6% |

* **Per seed and per priority.** Each seed's breach rate is measured
  separately for each priority in each gated window (T90 and tail). A
  seed passes service only if all three priorities pass in both windows.
* **Evidence minimum** [proposed]: a seed's priority cohort is evidence
  only with at least `n_min = ⌈g(1 − g) / 0.05²⌉` acquisitions, where g
  is the gate. This keeps the binomial standard error at the gate within
  5 points. The minimums are **urgent 43, soon 88, routine 23**, derived
  from the gates, not from any result (`TestPolicyFrozen` recomputes
  them).
  * Feasibility, checked against the no-action development results: T90
    has urgent 71–107, soon 478–582 and routine about 3,900–4,070
    acquisitions per seed.
  * A 60-day tail holds about 47–71 urgent acquisitions at the no-action
    rate, so the urgent tail minimum is close on low-volume seeds. That
    is a known risk, recorded under *Open decisions*; it is not loosened.
* **Insufficient evidence** [proposed]: a cohort with too few
  acquisitions, an empty cohort, or one with studies still undecided when
  observation ended is *insufficient*. It is never a pass and is never
  treated as a 0% rate. Insufficient seeds count against the seed-failure
  limit like failing ones.
* **Maximum failing seeds** [proposed]: at most `floor(0.10 × seeds)`
  seeds may fail or lack evidence on any gate, which is 4 of 40 on the
  holdout. The **pooled** rate (all seeds' breaches over all their
  acquisitions) must also pass. If more seeds lack evidence than the
  limit allows, or the pooled rate is undefined, the gate is
  *insufficient* and the plan cannot be GO.
* **Uncertainty** is reported for every gate:
  * the Wilson 95% interval of the failing-seed fraction;
  * the worst seed and its rate;
  * the mean;
  * and, for comparisons, paired t 95% intervals of each priority's
    breach-rate difference.
* **Due-demand volume and case mix** are guarded by the case-selection
  gates (§8), so a plan cannot pass on a smaller or easier caseload.
* **Inherited backlog** (studies unfinished and orders open at t0) is
  its own population. It is never in a new-demand cohort and never in a
  gate denominator, and its resolution is reported. It is not gated:
  every plan inherits the same backlog, and no action takes effect at the
  takeover instant.

## 3. Fixed acquisition cohorts

A cohort is **every study acquired in the window**, imaging end in (a, b],
finished or not. For each priority it reports:

| Field | Definition |
|---|---|
| acquired | Studies acquired in the window |
| finalized / unfinished | Finalized by b, or not |
| completionFraction | finalized ÷ acquired |
| completedTatMin | p50, p90 and max over studies final by b. Completed-only. It never decides a gate. |
| unfinishedAgeMin | p50, p90 and max age at b of the unfinished |
| overdueUnfinished | Unfinished at b and past target |
| overdueStudyMinutes | For every study, the time from its due time to the earlier of its final and b |
| breaches, breachRate | A study breaches if it is not final by `acquired + target`. Breaches ÷ acquired. |
| unresolved | Due after observation ended, and not final. Its breach is undecided. |
| legacyBreaches, legacyBreachRate | The rule of the historical reports (`calib_test.go` `tatByPriority`): as of b, completed over target plus unfinished past it |

**Breach is decided at each study's due time** [proposed], not at the
window's end. The historical as-of-end rule counts a study acquired an
hour before the end and still unread at the end as on time, whatever
happens next. Deciding at due time, observed through the closure,
removes that artefact. The rule is the same for every plan and the
baseline, the thresholds are unchanged, and the legacy rate is reported
beside it so the two can be compared. Unfinished studies are never
excluded: a study unfinished at its due time is a breach.

## 4. Backlog stability

This replaces the old endpoint rule, "backlog at the day-180 sample at
most 1.1 times the day-90 sample". That rule compared single samples on
different weekdays: day 90 is a Monday and day 180 a Sunday. Plan D failed
it (12 → 20) while its 30-day block means were flat (15.1, 13.4, 14.9).
The old rule and its failures stay in the historical reports
(`dynamic-staffing/results/*/reading.md`), byte for byte. The new
evaluation reports the old rule's outcome beside the new one, labelled
legacy and never gated.

**Rule** [proposed]:

* The daily samples are cut into whole **28-day blocks**: two 14-day rota
  cycles, each starting on a week-A Monday (`rotaWeekA` = 2026-10-12).
* The **reference block** is the last whole block ending by T90. The
  **final block** is the last whole block ending by tailEnd. The two
  never overlap, because the tail is at least 60 days.
* A seed is **stable** when the final block's mean backlog exceeds the
  reference block's mean by no more than max(10% of the reference mean,
  2 studies).
  * The 10% is the old rule's 1.1 factor, kept.
  * The 2-study floor is the measurement unit: it keeps a backlog of a
    few studies from failing on one study.
* The plan's backlog gate passes when at most 10% of its seeds are
  unstable or lack a whole pair of blocks.

Reported per seed, without gating:

* 28-day trailing means at every rota-cycle boundary;
* the least-squares slope from the reference block's start to the final
  block's end;
* each block's mean by stage;
* age p50 and p90 (means of the daily values) and the maximum;
* the overdue fraction (overdue ÷ backlog over the block's samples);
* acquisition against finalization flow.

The stages are distinguished as follows:

| Stage | Meaning |
|---|---|
| Awaiting documentation | Imaged, technologist documentation in progress |
| Unread | On the worklist |
| Remote handoff | Claimed by a remote lane, in its 10-minute handoff |
| Local reading | Claimed by an employed radiologist |
| Remote reading | Being read remotely |
| Finalized | Counted in the flow |

Backlog age and overdue exposure are not separate gates. The tail's
breach gates already judge every study acquired in the tail, so a second
age threshold would invent a clinical standard the design does not have.

## 5. Financial eligibility

Raw profit is the posting-based evaluation window's operating profit,
baseline-adjusted, as in every earlier report. T90 is primary, T30 is
feedback, D31–90 is operating performance, and T180 is validation. Each
capture (T30, T90, T180, tailEnd, end) lists the following.

| Measure | Source |
|---|---|
| Absolute accrual profit, net revenue, operating expense, margin | Evaluation window. The margin is undefined unless revenue > 0. |
| Incremental profit over the paired baseline | Per window, with a paired t 95% interval and the worst seed |
| Cash, **minimum cash and when** | The lowest balance after **any posting** since t0 (`noteCash`): every transaction, not a daily sample |
| Receivables, payables, accrued payroll, severance payable | Ledger balances |
| Earned but unposted wages | Hourly pay with benefits of shifts in progress, which is posted when the employee goes home |
| Remote in-flight fees | Fees locked by remote claims not yet final |
| Remaining minimum contract obligations | Retainers still owed to each contract's fixed end, or its earliest end if open-ended. Not yet accrued. |
| Unbilled completed or acquired work | The expected net revenue of studies imaged but not reported. **Future revenue, never cash and never profit.** |
| Past-due mandatory payments | 0 in V0.2 by construction. No action can defer payroll, payables, severance or vendor bills; they are paid when due, into negative cash if need be. |

* **Commitments are not expenses.** Contract minimums and in-flight fees
  are listed beside the P&L, never added to it. Accrued expenses are in
  the P&L, and payments are in cash. The **obligation bridge** at a
  capture is payables + accrued payroll + severance payable + earned
  unposted wages + in-flight fees + contract minimums + past due. Its
  excess over cash is the **funding gap**. There is no borrowing and no
  credit line, and future receivables are never counted as cash.
* **Solvency** [owner, made exact]: the owner's gate is "no missed
  mandatory obligation, no negative available cash". A seed is insolvent
  if its posting-level minimum cash, from t0 through the end of
  observation (the tail included), falls below 0 cents, or anything is
  past due. **No insolvent seed is allowed** [proposed: zero tolerance,
  because a financing shortfall is not a service rate].
* **Funding gap at the tail's end** [proposed]: a plan whose obligations
  exceed its cash at tailEnd on any seed is *financially incomplete*. It
  cannot be GO, at best CONDITIONAL GO.

## 6. Payback

* **Accrual payback** uses daily cumulative evaluation profit, and
  **cash payback** the daily cash balance. Both are measured against the
  paired baseline.
* The payback day is the **sustained break-even**: the first day from
  which the increment stays at or above zero through the horizon (90 and
  180 days). A later dip below zero resets it, the rule of the earlier
  reports.
* A seed still below zero at the horizon is **right-censored**. It is
  reported as censored, never as the horizon day.
* Each horizon reports the number repaid, the number censored, the
  repaid fraction with its Wilson interval, and the earliest and latest
  payback days.
* The **median** is reported only when at least half the seeds have
  repaid. It is the smallest day by which ⌈n/2⌉ seeds have repaid, the
  Kaplan–Meier median when every censoring falls at the horizon.
  Otherwise the report says it is not reached by the horizon.

## 7. Remote reading metrics

Per window, from the reports finalized in it:

* local and remote report counts;
* remote claims;
* handoff wait;
* remote and local completed turnaround;
* retainer and read-fee expense, from the ledger as differences between
  captures;
* the sum of the remote reports' locked fees, which must equal the
  ledger's read-fee expense, or the record is invalid;
* total remote cost and cost per remote report.

Against the paired baseline, these three are kept apart:

* **Faster reporting**: the paired breach-rate difference and the
  difference in completed p90 by priority.
* **Additional reporting**: the incremental count of reports finalized
  in the window.
* **Additional revenue**: incremental **recognized** net revenue in the
  window, from the ledger.

Revenue is recognized at report final, so a faster report earns revenue
inside the horizon only if the ledger records it there. The change in
unbilled work is listed separately and never added. The question the
owner flagged for teleradiology, its effect on the **worst seed's soon
breach rate** against its **cash drain**, reads directly from
`worstSeedBreachRate` (candidate against baseline, per window and
priority), `minCashDeltaCents` and `incrementalCashCents`.

**Booking invariance**: a record may declare that it differs from
another plan only by teleradiology actions (`teleOnlyOf`). Its
appointment set (identity, room, day, slot, state and booking time) must
then hash equal on every seed. Otherwise the evaluation is invalid,
because remote coverage must not change technologist booking. The
integration test checks this on real runs.

## 8. Access and case selection

Access is measured per window over the demand received in it:

* received, due (original wanted time at or before the window's end; a
  later promise never moves it) and not yet due;
* served (imaged);
* lost referrals (practice-caused);
* walk-in abandonment;
* patient-caused cancellations and no-shows;
* unresolved due orders;
* displaced orders and events;
* booking lead time.

Due counts and served counts are kept by priority and by modality.

**Case-selection gates** [proposed] are applied in every gated window,
against the paired baseline:

| Gate | Rule |
|---|---|
| Served fraction, per seed | In each priority with at least 30 due orders on the baseline seed, the due-demand served fraction falls by at most 5 points. At most 10% of seeds may exceed that. |
| Served fraction, pooled | Each priority and each modality falls by at most 2 points. A stratum the baseline served that disappears fails. |
| Acquisitions, pooled | Each priority's acquired studies fall by at most 2% |
| Lost referrals, pooled | The rate (÷ referrals received) rises by at most 2 points |
| Walk-in abandonment, pooled | The rate (÷ walk-ins received) rises by at most 2 points |

They stop a plan from passing the breach gates by serving fewer, easier
or more profitable cases: declining routine work, turning away urgent
cases, letting walk-ins leave. A plan that serves more is never
penalized. The tolerances are anti-gaming margins, not clinical
standards. They were not measured from paired noise. They are small
explicit margins: tight pooled, looser per seed, where a single seed's
draw varies more.

## 9. Seeds

| Set | Seeds | Use |
|---|---|---|
| Development | 901–910 | Strategy development and selection. The only seeds strategies may be tuned on. |
| Historical reference | 1001–1020 | Exposed by the dynamic staffing and plan D evaluations. Reference only, never for selection or tuning. |
| **Locked holdout** | 40 seeds, `v0.2-holdout-1` | Sealed |

**How the holdout seeds are made**:

* Seed i (i = 0, 1, …) is `100000 + (first 8 bytes of
  SHA-256("dreamteam/v0.2/eval-holdout-1/" + i), big-endian) mod 900000`.
  Repeats and earlier sets (1–100, 901–910, 1001–1020) are skipped, until
  40 seeds are drawn.
* The manifest records the method, the label, the size and the SHA-256
  of the seed list. The list itself is not written anywhere in the
  repository.
* `eval.LoadHoldout` regenerates the list and checks it against the
  digest. The method was cross-checked independently: a separate
  implementation gives the same digest.

**Fixed size: 40** [proposed]. That is twice the earlier 20. With n = 40
the failing-seed limit is 4, and one failing seed moves the fraction by
2.5 points.

**Sealed.** Every calibration harness calls `eval.GuardRepo` on its seeds
(`parseSeeds` in `internal/sim`, and the session harness). A holdout seed
is refused until `holdout-unseal.json` is committed, naming:

* the holdout;
* the current policy's SHA-256;
* the SHA-256 of the frozen strategy set;
* the owner's approval.

**One-time protocol** (in the manifest):

1. Freeze the strategies on development seeds only.
2. Have the owner confirm the policy, then audit it.
3. Commit the strategy files and the unseal record.
4. Run every frozen strategy and the baseline once on all 40 seeds, and
   commit the records and report as they come out.

A rerun may only reproduce the same bytes. If anything changes after
unsealing, the holdout becomes reference-only and a new one is generated
under a new label.

## 10. Verdicts

There is no weighted score. Profit never compensates for a failed gate.
The rules are applied in this order (policy `verdict`):

1. **INVALID**: any integrity check fails. The checks are:
   * the schema;
   * the scenario fingerprint;
   * the TAT targets (frozen in the policy, so a scenario cannot pass by
     changing them);
   * an action at or after T90;
   * the tail rule;
   * observation through the closure;
   * the cash bridge;
   * samples reconciling with studies (backlog, acquisitions and finals
     on every day);
   * remote fees reconciling with the ledger;
   * identical seed sets, engine, scenario and takeover;
   * booking invariance for teleradiology-only variants.

   An invalid evaluation is not evidence.
2. **NO GO**: a hard gate fails or lacks evidence. The hard gates are the
   service breach gates (T90 and tail, per seed and pooled), case
   selection, backlog stability and solvency.
3. **GO**: every hard gate passes, and all of the following hold:
   * the T90 paired 95% interval of incremental profit lies above zero;
   * T180 validation holds: its service gates pass and its mean
     increment is positive;
   * no seed has a funding gap at tailEnd.
4. **CONDITIONAL GO**: every hard gate passes, and either:
   * the mean T90 increment is positive without every GO condition; or
   * it is not positive, but the baseline fails a hard gate the candidate
     passes, so service is restored at a stated cost.
5. **NO GO**: every hard gate passes, but there is no T90 profit while
   the baseline already passes every gate.

Paired comparisons between two candidates (for example "D + tele"
against "D") report the same paired differences without a verdict.
Every report also gives each gate's worst seed.

## 11. Records

A record (`dreamteam.evalrecord/v1`) is one plan on one seed. It holds:

* every study unfinished at t0 or acquired after it, with its
  timestamps, reader kind and locked fee;
* every order open at t0 or received after it, with its wanted time,
  first appointment and displacements;
* daily samples (backlog by stage, ages, overdue, cumulative flow, profit
  and cash);
* captures at T30, T90, T180, tailEnd and end, with order outcomes and
  money;
* the appointment-set hash.

A 5C.1 harness runs each plan to `max(tailEnd, T180) + closure`. It
samples every day and captures at each horizon instant with the
recorder, then calls `Finish`. The world must be at the takeover, with
an open-ended evaluation window (`EvaluationDays = 0`), when recording
starts.

## Open decisions

These need the owner's confirmation before the holdout is unsealed:

1. **Breach decided at due time** instead of the historical as-of-end
   rule, with the thresholds unchanged (§3).
2. **Evidence minimums** from a 5-point standard error at the gate
   (urgent 43, soon 88, routine 23).
   * The urgent tail cohort is close to its minimum on low-volume seeds,
     so a plan can be *insufficient* rather than failing.
   * If that happens, the conservative remedies are a longer fixed tail
     or pooling the tail over seeds. Either needs a policy change before
     unsealing, not after.
3. **At most 10% of seeds** failing or lacking evidence per gate, with
   the pooled rate also required to pass.
4. **Case-selection tolerances**: 5 points per seed; 2 points pooled
   for the served fraction, losses and abandonment; 2% for acquisitions.
5. **Backlog rule**: 28-day rota-aligned blocks, growth within max(10%,
   2 studies), at most 10% of seeds unstable, and no separate age gate.
6. **Solvency with zero tolerance** on any seed, through the tail. A
   funding gap at tailEnd blocks GO.
7. **GO needs a significant T90 benefit**, a positive T180 and no funding
   gap. A service-restoring plan at a cost is CONDITIONAL GO.
8. **Holdout of 40 seeds**, its generation method and the one-time
   protocol.
9. **Still informational in v1**: the design's other baseline-relative
   tolerances (`DELTA_*` in §13.3) for access p90, displacement, coverage
   and workforce are reported but not gated, because no calibrated value
   exists. Promoting any of them needs a policy v2 before unsealing.
10. **Not modelled**: remote read quality. Remote and local reports are
    assumed equivalent beyond turnaround, cost and coverage. Payer
    strata are not used, because a refused patient's payer is never
    drawn.
