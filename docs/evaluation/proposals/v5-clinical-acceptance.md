# Proposed v0.2-eval-5: clinical acceptance (Slice 5C.4a)

**Status: proposal, not active.**

* **Slice 5C.4b** implements every rule below in `internal/eval`
  (`clinical.go`), reached only by a policy of schema
  `dreamteam.evalpolicy/v5`.
* **No v5 policy file exists yet**: the values are the owner's to set,
  after the combined synthetic operating characteristics
  (`docs/calibration/clinical-5c4b`).
* **Unchanged:** v0.2-eval-4 stays in force, and its evaluations are
  byte-identical (`TestV4Neutrality`).
* **Settled by the implementation:** §J lists where the code makes
  precise a point this proposal left open.

* **What is in force.** v0.2-eval-4 (`../policy-v0.2-eval-4.json`)
  stays in force, unchanged. No evaluator code implements anything here.
* **Where it goes next.** Slice 5C.4b implements this specification
  and validates it on synthetic records. It is frozen only once the
  values marked **OWNER DECISION REQUIRED** are decided.
* **What it does not use.** No limit here is derived from a strategy
  result. D2 and E3 were not rerun, and no development, reference or
  holdout outcome was consulted.

It answers audit findings H1 (severe reporting delays pass every gate)
and H2 (access is not gated), both confirmed by executed witnesses
([`../../audits/slice-5c3-eval4-framework-audit.md`](../../audits/slice-5c3-eval4-framework-audit.md),
Appendix C).

## 0. Owner decisions recorded (2026-10-18)

| Question (audit §9) | Decision |
|---|---|
| Severe reporting delay | Hard, priority-specific severity limits, independent of aggregate breach rates |
| Timely access | Gated: wanted (requested) time to acquisition, unresolved orders included |
| 92% utilization | An explicitly **out-of-calibration stress domain** for now; never evidence of evaluator reliability |
| Clinical failure allowance | **One shared four-seed allowance** (at 40 seeds) across reporting service, access and backlog. Violations of the absolute age limit are **zero-tolerance** in the evaluated records |
| Holdout protection | Cooperative safeguards are insufficient for a one-time confirmatory claim: durable external custody and authenticated approval are required (Slice 5C.4c) |
| Prior holdout seed use | Documented as limited exposure (`../holdout-exposure-5c4a.md`); the holdout is not discarded |
| Clinical limits | Defined prospectively from each priority's clinical meaning and the existing due-time definitions; never chosen to reject a particular witness |
| Access measures | Two complementary measures: a priority-specific maximum wait and a population timely-acquisition rate. An unresolved order keeps accumulating waiting time |
| Backlog | v0.2-eval-4's rule kept until the combined clinical gates are tested; its 150% allowance is documented as up to 250% of the adjusted reference |

**Two kinds of clinical failure** (owner's qualification). v5 keeps them
separate:

* **Absolute safety limits.**
  * One violation in any evaluated record makes the plan **NO GO
    (clinical safety)**. It does not use the seed allowance.
  * These limits must be truly absolute: rare enough in an acceptable
    practice that one occurrence is evidence against the plan.
  * A limit that stochastic simulation crosses now and then in a sound
    practice cannot be zero-tolerance; it is a seed-level gate (§C.3).
* **Seed-level gates.** A seed fails if any seed-level gate fails or
  lacks evidence. The failing seeds share one allowance.

## Definitions

* **Times.** t0 is the takeover; H the common horizon; E the observation
  end (`ObservedUntilSec`, at least H + 31 days). All times are exact
  record timestamps, never daily samples.
* **Study.** Priority p, acquired `AcquiredSec` = a, finalized
  `FinalSec` = f (0 if unfinished at E).
* **Due time.** d = a + TAT_p, with the frozen targets: urgent 60 min,
  soon 240 min, routine 1,440 min.
* **Past-due delay** at time t: L(t) = max(0, min(f, t) − d), with f
  taken as +∞ if the study is unfinished.
  * For a finished study, L = f − d exactly.
  * For an unfinished study, L(E) = E − d is a lower bound (right
    censored). It keeps growing: an unfinished study is never dropped.
  * A limit X is **violated** when L > X, whether the study is finished
    or censored. It is **undecided** when a censored study has L(E) ≤ X.
  * Every study due by H is decided for any X ≤ 31 days, because
    E ≥ H + 31 d.
* **Order.** Priority p, wanted time w (`WantedSec`: the start of the
  earliest day the patient asked for, or a walk-in's arrival; a later
  promise never moves it).
  * Its final outcome is taken at the `end` capture.
  * If served, its acquisition time is s = `ResolvedSec` (exam end).
* **Wait.** W = s − w if served; otherwise W(t) = t − w while the order
  is unresolved, which also keeps growing.
* **Inherited.** A study acquired at or before t0, or an order open at
  t0.

## A. Severe reporting-delay protection

Each rule uses exact due and completion timestamps, and counts completed
and unfinished studies alike.

### A.1 Absolute age limit (all priorities): zero tolerance

* **Rule.** No study may have L > 14 days.
* **Scope.** Every study acquired at or before H, inherited studies
  included (§A.5).
* **The value is kept.** 14 days is v0.2-eval-4's `persistentAfterDueDays`
  (one rota cycle: a study still unread after every reader on the rota
  has been scheduled through a whole cycle). It is still
  **[proposed]**, open decision 2 of the README; confirming it is an
  **OWNER DECISION REQUIRED**.
* **What changes.**
  * It is measured from timestamps instead of 06:00 samples, which
    closes the 348-hour (seen as 334 hours) gap.
  * It is judged through E, not only the covering block.
  * It moves out of the backlog seed allowance into zero tolerance.

### A.2 Priority-specific absolute maximum delay: zero tolerance

* **Rule.** No urgent study may have L > Lmax_urgent, and no soon study
  L > Lmax_soon.
* **Ordering.** Lmax_urgent ≤ Lmax_soon ≤ 14 days, so a more urgent
  priority is never allowed a longer delay.
* **Values: OWNER DECISION REQUIRED.** No clinically validated number
  is proposed. Two facts bound the choice:
  * **The owner's own requirement.** Rejecting six urgent reports eight
    days late is itself a clinical judgment: an urgent report 8 days
    (192 hours) past a 1-hour target is unacceptable. That makes
    Lmax_urgent < 8 days a consequence of the decision, not tuning to
    the witness.
  * **The scenario's structural delays.** These come from the published
    rota, not from any run. They are what the no-action practice does
    by construction, with no teleradiology:
    * Weekday imaging runs to 20:00 and the last reader leaves at 20:00.
      An urgent study acquired just before 20:00 is read from 07:30 the
      next day, about **10.5 h past due**.
    * Saturday imaging runs to 18:00, reading to 14:00, and Sunday is
      closed. An urgent study acquired on Saturday after 14:00 waits for
      Monday 07:30, up to about **40.5 h past due**.
    * A soon study in the same position is about **37.5 h past due**.
  * **The consequence.** Lmax_urgent below about 41 h makes every
    Saturday-afternoon urgent acquisition without weekend reading a
    zero-tolerance violation. That may be clinically right (a closed
    weekend is a real urgent-care gap). But it rejects the no-action
    practice structurally, and it is a choice about what the evaluation
    claims, so the owner must make it explicitly.

### A.3 Severe-delay definition and incidence: seed-level

* **Severe delay.** L > 1,440 minutes (1 day past due). This is
  v0.2-eval-4's `staleAfterDueMin`, still **[proposed]**: the
  evaluation's closure, a breach decided a full day ago. Confirming it,
  or setting it per priority, is an **OWNER DECISION REQUIRED**.
* **Rule.** For priority p, the share of the cohort with a severe delay
  is at most r_p. The values of r_p are an **OWNER DECISION REQUIRED**.
* **Windows.** The same as the breach gates: urgent over the gated
  period (t0, H]; soon and routine separately in T90 and the tail.
* **Denominator.** The breach gates' acquisition cohort: every study
  acquired in the window, finished or not. Censored studies are decided
  as above.
* **Why both A.3 and the breach gate.** A breach rate counts any
  lateness alike: a report 2 minutes late and one 8 days late each count
  once. A.3 counts only lateness beyond the severe threshold, so a low
  breach rate can no longer hide severe cases.

### A.4 Concentrated severe delays: seed-level

* **Rule.** For priority p, at most k_p severe-delay studies acquired in
  any one Monday-based calendar week of the gated period. That is the
  same week as the cluster diagnostics (`stats.go`); partial first and
  last weeks are counted as they are.
* **Values: OWNER DECISION REQUIRED,** both k_p and whether Monday weeks
  or a rolling 7 days are used.
* **Why.** Concentrated failure is the clinical signature that period
  averages and block medians hide: a bad week in an otherwise good
  quarter. Its false-rejection cost under the structural weekend delays
  (§A.2) must be measured before values are set (§G).

### A.5 Inherited studies

* **Rule.** An inherited study already past due at t0 has its delay for
  the absolute limits (A.1, A.2) measured from max(d, t0).
  * The plan is held responsible for the time since the takeover.
  * Its full delay from d is reported.
  * The rule is the same for every plan and the baseline, which share
    the inherited studies.
* **OWNER DECISION REQUIRED.** The alternative is to measure from d,
  which makes a takeover with very old inherited work fail every plan
  alike.
* **Unchanged.** Inherited studies stay a blocking case-selection
  stratum, as now.

## B. Timely access

**Access limit A_p (days).** An order of priority p is:

* **timely** if it is served with W ≤ A_p;
* **untimely** if it is not served by w + A_p, for whatever reason:
  still open, lost (declined, no slot, displaced and lost), left (a
  walk-in who abandoned), or served later;
* **withdrawn** if it ended by patient cancellation or no-show before
  w + A_p without being served. A withdrawn order leaves the
  denominator, because the patient ended it before the practice had
  failed it. A patient-caused end after w + A_p is untimely: the
  failure had already happened.

Under these definitions:

* **No survivorship bias.** Open, lost and late orders stay in the
  denominator, and an unresolved order keeps accumulating waiting time.
* **Displacement cannot reset the clock.** The wait runs from the
  original wanted time.
* **Decidability.** An order is decided by w + A_p ≤ E. Every order
  wanted by H is decided if A_p ≤ 31 days. A larger A_p (routine) needs
  E extended to H + A_p, or a decision that open orders at E are
  untimely once W(E) > A_p.

### B.1 Timely-acquisition fraction: seed-level, and pooled

* **Rule.** For priority p, timely / (timely + untimely) ≥ g_p.
* **Cohort.** Orders whose wanted time falls in the window, walk-ins
  included. Inherited orders form their own stratum (§B.3).
* **Windows.** As for reporting: urgent over (t0, H]; soon and routine
  in T90 and the tail separately. The tail window catches deterioration
  near the horizon; T90 catches it early.
* **Pooled.** The pooled fraction must pass in each window, as the
  breach gates' pooled rates must.
* **Values: OWNER DECISION REQUIRED,** both g_p and the minimum
  decided-order cohort per priority. The latter may reuse the breach
  gates' n = ⌈g(1−g)/0.05²⌉ heuristic, with its known limits (audit,
  Slice 5C.0, H1).

### B.2 Maximum wait

* **Rule.** For priority p, no order may have W > Wmax_p. This includes
  open orders at E, through W(E), and lost and left orders, through
  their resolution time or E.
* **OWNER DECISION REQUIRED:** the values, and whether B.2 is absolute
  (zero tolerance) or seed-level. The recommended default is
  **seed-level**. Patient-chosen dates are anchored in w already, but
  rare scheduling extremes in a sound practice are plausible, so zero
  tolerance must first pass the rare-event test (§C.3).

### B.3 Inherited orders

* **Rule.** The wait is measured from max(w, t0) for gating and from w
  for reporting, for the same reason as §A.5.
* **OWNER DECISION REQUIRED,** with the same alternative as there.

### Candidate anchors (none is clinically validated)

These are listed only to frame the owner's decision:

* **The scenario's patient wait tolerances** (`behavior.waitToleranceDays`):
  urgent 0–2 days, soon 1–10, routine 5–60. These are the waits patients
  accept, behavioural parameters of the simulation, not clinical
  standards.
* **The tolerance as the limit.** Using them as A_p would make timely
  mean "within what this patient would have accepted". That ties the
  clinical gate to a modelling assumption.

## C. Joint clinical reliability

### C.1 One shared seed allowance

A seed **fails clinically** if any seed-level clinical gate fails or
lacks evidence:

* the five per-seed breach gates (unchanged): urgent over the gated
  period; soon and routine in T90 and the tail;
* severe-delay incidence (A.3) and concentration (A.4);
* timely access (B.1), and the maximum wait (B.2) if it is seed-level;
* backlog stability: every v0.2-eval-4 hard block check except the age
  limit, which becomes A.1.

How the allowance works:

* **A seed counts once.** A seed failing several gates is one failing
  seed.
* **Plan rule.** The plan's clinical gate passes if failing seeds ≤
  ⌊0.1 n⌋: 4 of 40, 2 of 20, 1 of 10.
* **Pooled requirements stay separate plan-level gates:** the breach
  rates and the timely-access fractions.
* **Reported:** each seed's failing gates; the count per gate; the union
  count; and the exact one-sided 95% upper bound on the failing share.
  The 40-seed rule is read as before (README §2): it is an empirical
  acceptance test, not a population-reliability claim.
* **What changes.** v0.2-eval-4 has separate four-seed allowances for
  service and backlog, so up to 8 seeds could fail clinically (audit
  §8). v5 allows 4 in all.

### C.2 Absolute safety limits

* **Which limits.** A.1, A.2, and B.2 if the owner makes it absolute.
  Any violation in any evaluated record makes the plan **NO GO
  (clinical safety)**.
* **How it is reported.** The violation is listed by plan, seed and
  limit, with no study timestamps that would identify a holdout
  outcome beyond the verdict.

### C.3 The cost, stated before calibration

**Shared allowance.** Under independence between gates, the union
raises false rejection sharply. The figures are computed from the
committed operating characteristics.

* **Service alone.** The per-seed failure probability that gives the
  frozen 0.030 forty-seed rejection is about 4.4%.
* **Adding backlog failures:**

  | Per-seed backlog-unstable probability | Union per seed | 40-seed rejection, shared allowance | Separate allowances (v4) |
  |---|---|---|---|
  | 0.5% (stable bursty) | 4.9% | 0.044 | 0.030 |
  | 3.8% (stable seasonal and bursty, Cs) | 8.0% | **0.215** | 0.047 |
  | 12.4% (seasonal at utilization 0.92) | 16.3% | **0.80** | 0.58 |

* **Reading it.** The ≤ 5% aim is not met at Cs once the allowance is
  shared, even before access and severity gates are added. This is the
  price of counting every clinically failing seed against one
  allowance. 5C.4b must measure it on combined synthetic records rather
  than assume independence. The owner then chooses: accept the higher
  false rejection, lengthen observation, or revise a component gate in
  a new version.

**Zero tolerance.** A zero-tolerance limit rejects a plan whose per-seed
violation probability is q with probability 1 − (1 − q)^40.

* Keeping that at or below 5% needs **q ≤ 0.13%**.
* At q = 1% it is already 0.33.
* So a limit is admissible as absolute only if combined synthetic
  validation shows q ≤ 0.13% for acceptable practices (a prespecified
  target in §G). Otherwise it is made seed-level, and that is recorded
  as a policy change.

**Stress domain.** Results at utilization 0.92 are reported as a stress
domain, never as evidence that the evaluator is reliable or unreliable
(owner decision).

## D. Backlog

* **v0.2-eval-4's block rule is kept**, with the same blocks, statistics
  and tolerances, until the combined clinical gates are validated (§G).
* **What changes:**
  * the age limit leaves the block checks and becomes A.1;
  * backlog stability counts in the shared allowance (C.1) instead of
    its own.
* **The level allowance, explicitly.**
  * A later block passes the level check when B ≤ w·B_ref +
    max(1.5·w·B_ref, 2 studies): **up to 250% of the workload-adjusted
    reference**, or the reference plus 2 studies, whichever is larger.
  * The 150% is the allowance above the reference, not the ceiling.
  * This is now stated in README §4. It is not a threshold change.
* **No recalibration in v5.** If combined validation shows the backlog
  rule must change, that is a further, separately justified version.

## E. Other corrections carried into v5

* **M3. "Service restored" needs a demonstrated failure.**
  * The baseline must *fail* the service gates the candidate passes.
  * A baseline whose service is only *insufficient* is reported as
    insufficient evidence, not as restored service.
* **L1. Observation end.** README §1's example is corrected: observation
  ends at max(H, T180) + closure, at least H + 31 days, or at the end of
  the covering block. A day-170 horizon gives at least day 201.
* **Validation of new fields.**
  * A record is **INVALID** if:
    * a timestamp is missing, or contradicts one it depends on (f < a;
      a served order with s = 0, or s < w; ResolvedSec before
      ReceivedSec);
    * the end capture's outcomes contradict ResolvedSec;
    * a censored item's state contradicts E.
  * INVALID is never a pass or a fail.

## F. Limit justification

| Limit | Priority | Window | Denominator and censoring | Record fields | Clinical rationale | False-rejection trade-off | Status |
|---|---|---|---|---|---|---|---|
| A.1 age limit, 14 d past due, zero tolerance | all | studies acquired ≤ H, judged through E | each study; censored if unfinished, violated once E − d > 14 d | Study.AcquiredSec, FinalSec, Priority; TATTargetMin; ObservedUntilSec; TakeoverSec | A study unread through a whole rota cycle is abandoned work for any priority | No seed of the stable synthetic queues A, B, C or Cs failed the sampled age check (0 of 1,000 each; `oc-v0.2-eval-4.json`, `attributionEval3`). 0 of 1,000 bounds q at about 0.3% (95%), above the 0.13% needed; the exact-timestamp version must be shown on more seeds (G15) | value kept from v4, **[proposed]**: OWNER DECISION REQUIRED |
| A.2 Lmax_urgent, Lmax_soon, zero tolerance | urgent, soon | as A.1 | as A.1 | as A.1 | The clinical meaning of urgent (1 h) and soon (4 h) targets; must be < 8 d for urgent by the owner's requirement | Below about 41 h, structural weekend delay in a practice without weekend reading violates it (§A.2) | **OWNER DECISION REQUIRED** |
| A.3 severe threshold, 1 d past due | all (owner may set per priority) | as the breach gates | the breach cohort; censored as A.1 | as A.1 | Lateness beyond the closure, the evaluation's own decision horizon | Structural weekend delays make some soon and urgent studies severe every week in the no-action rota | value kept from v4, **[proposed]**: OWNER DECISION REQUIRED |
| A.3 r_p incidence | urgent, soon, routine | urgent (t0, H]; others T90, tail | as above | as above | A rate of severe cases, separate from the breach rate | To be measured on synthetic records with the scenario's hours (§G); must not be set from strategy runs | **OWNER DECISION REQUIRED** |
| A.4 k_p per week | urgent, soon, routine | each Monday week of (t0, H] | studies acquired in the week | as above | A concentrated failure a period average hides | Week counts are noisy at low volume (urgent about 1 a day); measure first | **OWNER DECISION REQUIRED** |
| B.1 A_p, g_p | urgent, soon, routine | urgent (t0, H]; others T90, tail | orders wanted in the window; untimely includes open, lost and left; withdrawn before w + A_p excluded | Order.WantedSec, ResolvedSec, FirstApptSec, DisplacementSec, WalkIn, Priority; end capture outcomes | Patients imaged within a clinically acceptable wait | Lost orders also count in case selection: deliberate, as one seed counts once | **OWNER DECISION REQUIRED** (anchors in §B are behavioural, not clinical) |
| B.2 Wmax_p | urgent, soon, routine | as B.1 | every order; open ones through W(E) | as B.1 | No patient waits past a hard limit | Zero tolerance needs q ≤ 0.13%; default seed-level | **OWNER DECISION REQUIRED** |
| A.5, B.3 inherited, from t0 | all | as above | inherited stratum | + TakeoverSec | The plan answers for time since the takeover | Measuring from d or w fails every plan alike when old work is inherited | **OWNER DECISION REQUIRED** (default from t0) |
| C.1 shared allowance, 4 of 40 | all | per seed | union of seed-level gates | all of the above | Up to 10% of seeds may fail clinically, counted once | Cs: about 0.22 at 40 seeds under independence (§C.3) | owner decided; **trade-off for OWNER DECISION** after 5C.4b |

## G. Synthetic acceptance test plan (for Slice 5C.4b)

**How the tests are run.**

* **Synthetic records only.** All are deterministic synthetic records or
  synthetic queues. None runs a strategy, a reference seed or a holdout
  seed.
* **The full evaluator.** The combined tests run `Evaluate`, not
  `judgeBacklog` alone (audit M1).
* **The ordering defect is fixed first.** Arrivals within an hour are
  sorted chronologically before reading, and the fix is shown to change
  nothing in the committed v4 operating characteristics' selection, or
  the change is reported.
* **Expected outcomes need values.** They assume the owner's values, so
  each row states what must hold *for any value consistent with the
  owner's decisions*.

| # | Case | Construction | Expected | Why |
|---|---|---|---|---|
| G1 | Six urgent reports 8 days late (the audit's witness) | audit Appendix A `report`, all 40 seeds | **NO GO (clinical safety)** by A.2; also an A.4 seed failure | Lmax_urgent < 8 d follows from the owner's requirement |
| G1a | Acceptable: six urgent reports 2 h past due in one week | as G1 | passes A.1–A.4; breaches counted by the breach gate as now | Ordinary lateness is not severe |
| G2 | One report finished 348 h past due (sampled as 334) | audit Appendix A `age` | **NO GO** by A.1 | Exact timestamps close the sampling gap |
| G2a | Acceptable: one report 335 h past due | as G2 | passes A.1 | Just inside the limit, exactly compared |
| G2b | An urgent study unfinished at E, due 20 days before H | censored | **NO GO** by A.1 (and A.2) | An unfinished study is never dropped |
| G3 | First 30 urgent acquisitions delayed 30 days (the audit's witness) | audit Appendix A `access` | **NO GO**: B.1 fails on every seed, so the shared allowance is exceeded; B.2 too | Delayed access is gated |
| G3a | Acceptable: 30 urgent acquisitions moved 2 h later within their wanted day | as G3 | passes B.1, B.2 | Within any A_urgent of at least a day |
| G4 | Unresolved urgent orders: 5 urgent orders per seed never booked | open at E | untimely in B.1; W(E) beyond B.2: seed failure | No survivorship |
| G4a | Urgent orders lost (no slot) | lost before w + A | untimely | Practice-caused loss is not timely |
| G4b | Patient cancels before w + A, and another after | patient outcomes | the first withdrawn (excluded), the second untimely | Censoring rule |
| G5 | Concentrated weekly reporting delays: k_soon + 1 severe soon delays in one week, aggregate rate under r_soon | single week | seed failure by A.4 | Concentration is visible |
| G5a | Acceptable: the same count spread one a week | spread | passes A.4 | Same incidence, not concentrated |
| G6 | Persistent access deterioration: timely fraction falls below g in the tail and stays there | tail only | tail B.1 fails, a seed failure | Deterioration near the horizon |
| G6a | Acceptable: a dip in T30 that recovers, T90 above g | early dip | passes | Window design |
| G7 | Stable high utilization: synthetic seasonal queue at 0.92 | ocQueue, 1000 fresh seeds | **reported** union false rejection and per-gate shares, labelled stress domain; no pass or fail target | Owner decision: out of calibration |
| G8 | Bursty seasonal demand (Cs) | ocQueue, 1000 fresh seeds | 40-seed false rejection of the combined evaluator reported against the ≤ 5% aim; the result goes to the owner whichever side it falls | §C.3 trade-off |
| G9 | Gradual capacity erosion (K) and the 10% step-down (L10) | ocQueue | combined detection reported per gate. The claim that service and access gates compensate for the backlog gate is **accepted only if measured** | Audit §5 |
| G10 | Missing and contradictory timestamps | f < a; served order with s = 0; s < w; missing samples; outcome contradicting ResolvedSec | **INVALID**, never pass or fail | Integrity |
| G11 | Shared allowance | 4 seeds fail service and 4 *other* seeds fail backlog | **NO GO** in v5 (8 > 4); v4 passes | One allowance |
| G11a | Acceptable: 4 seeds each fail service, backlog and access | same seeds | passes (4 failing seeds) | A seed counts once |
| G12 | Absolute against allowance | 1 A.1 violation on 1 seed, every other gate passing | **NO GO (clinical safety)**, allowance untouched | The two kinds stay separate |
| G13 | Inherited study 20 d past due at t0 | finalized on day 2, and another left unresolved 15 more days | the first passes (from t0, default); the second fails A.1 | §A.5 |
| G14 | Insufficient baseline (audit Appendix B) | classification unit test | not "service restored" | M3 |
| G15 | Rare-event admissibility | stable synthetic queues A, B, C, Cs at fresh seeds | per-seed violation probability of every zero-tolerance limit reported, with its exact 95% upper bound; a limit is admissible as absolute only if that bound is at most 0.13%, which needs at least about 2,300 violation-free seeds per scenario | §C.3 |

**Prespecification.** The targets in G7–G9 and G15 (the ≤ 5% aim, and
q ≤ 0.13% for admissibility), the scenarios and the fresh streams are
fixed in 5C.4b's protocol before any combined result is seen. Limits
are not adjusted after it to make a scenario pass.

## H. Owner decisions required before v5 can be frozen

1. **A.1.** Confirm 14 days past due as the absolute age limit.
2. **A.2.** Lmax_urgent and Lmax_soon, and whether a closed weekend
   should be an urgent safety violation.
3. **A.3.** The severe threshold (1 day, or per priority) and r_p.
4. **A.4.** k_p, and Monday weeks or rolling 7 days.
5. **B.1.** A_p, g_p and minimum decided-order cohorts; the observation
   extension for any A_p over 31 days.
6. **B.2.** Wmax_p, and absolute or seed-level.
7. **A.5 and B.3.** Inherited work measured from the takeover (default)
   or from its own due or wanted time.
8. **B.** Withdrawn-before-the-limit exclusion (default), and walk-in
   abandonment counted as untimely (default yes).
9. **C.3.** After 5C.4b's measurement: accept the combined false
   rejection, lengthen observation, or revise a component in a further
   version.

## I. Implementation outline (Slice 5C.4b)

* **Policy file.** Schema `dreamteam.evalpolicy/v5`, file
  `policy-v0.2-eval-5.json`.
  * It supersedes v4, which is kept byte for byte, and records each
    changed rule as `was`, `now`, `rationale` and `expectedImpact`.
  * It adds the sections `severity` (A), `access` (B) and `clinical`
    (the shared allowance and the absolute-limit list).
* **Record.** No new simulator field is required: every rule uses
  existing record fields. The recorder and the engine are unchanged.
* **Measurement.** Exact-timestamp delay and wait computations, with
  censoring, in `internal/eval`. Union seed failure and clinical-safety
  verdicts are added to the report schema.
* **Validation.** G1–G15 as tests, the combined synthetic operating
  characteristics under a prespecified protocol, and mutation tests for
  each new rule.
* **Not in 5C.4b.** Refused-action semantics, approval binding,
  authenticated approval, custody and durable run-once are Slice
  5C.4c's.

## J. Implementation decisions (Slice 5C.4b)

Each of these is a point the sections above left open, made precise in
code and tests. Every one can be changed when the owner freezes v5.

* **Absolute limits (A.1, A.2).**
  * They are one per-priority limit, `absoluteMaxPastDueMin`; routine
    is A.1's age limit.
  * They apply to every study acquired by the horizon, inherited ones
    included, judged at the end of observation.
  * A study's delay is its finish minus its due time, or, if
    unfinished, the end of observation minus its due time.
* **Severe delay (A.3, A.4).**
  * The severe-delay rate is gated per seed only, in the priority's
    per-seed windows. Its pooled rate is reported, never gated.
  * The weekly count uses Monday weeks of the gated period, by
    acquisition.
* **Maximum wait (B.2).**
  * It covers served and open orders only. A lost or abandoned order is
    untimely in B.1, and its loss is case selection's; it has no wait
    to measure.
  * It is measured over the gated period's new orders and the inherited
    orders wanted by the horizon.
* **Inherited orders (B.3).** They are subject to the maximum wait,
    never to a timely-access rate, because they are not new demand.
* **Per-seed windows.** The severity and access gates are per seed
  exactly where the priority's breach gate is (`perSeedWindows`). In
  T180 they are reported only; validation stays the breach rule's.
* **Backlog in v5.**
  * The sampled age limit (`persistent`) must be informational, and is
    reported only.
  * A v4 policy that makes it informational is refused, as before.
* **Restoration (M3).** Under v5, service is restored only when the
  baseline's breach service demonstrably fails. An insufficient
  baseline gives "no benefit" with that reason stated.
* **Record integrity (G10).** A v5 record is invalid if any order was
  imaged before its wanted time.
* **Report.** Schema `dreamteam.evalreport/v5`. The v4 statuses stay
  reported beside the clinical ones.
