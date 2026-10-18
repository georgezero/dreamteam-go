# Slice 5C.4b closure: the v0.2-eval-5 acceptance design

The final specification step before v0.2-eval-5 is selected and
frozen.

* **Status: not frozen, and blocked on one owner decision** (§2).
* **Candidate policy:** `candidate-policy-v5.json`. It is a proposal,
  not in force, kept outside `docs/evaluation`. It is v0.2-eval-4
  unchanged plus the clinical section, with the persistent check
  reported only.
* **Companion documents:** `order-lifecycle.md`,
  `access-denominators.md`, `reserved-validation-protocol.md`.
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal or reserved stream. The simulator ran with no action on the
  development seeds only.

## 1. The provisional clinical policy

**Reporting** (exact timestamps; unfinished studies counted to the end
of observation; hours **past the existing due time**, not
acquisition-to-report turnaround):

| Rule | Candidate | Status |
|---|---|---|
| Urgent absolute maximum | 24 h past due | provisional (owner) |
| Soon absolute maximum | 48 h past due | provisional (owner) |
| Routine absolute maximum | 14 days past due | provisional; needs clinical justification |
| Severe delay | more than 24 h past due, every priority | provisional; the severity definition is the owner's |
| Severe-delay rate | at most 5% per priority, per seed, in the per-seed windows | provisional |
| Severe delays a week | at most 5 per priority | provisional; **see §3: unsuited to routine volume** |
| Hard maximum violations | zero tolerance on every evaluated seed, outside the allowance | provisional (owner) |

**Access** (calendar clock from the original wanted time, every
eligible order: `access-denominators.md`):

| Rule | Candidate | Status |
|---|---|---|
| Urgent timely acquisition | 48 h, at 90% (variant 95%) | **BLOCKED** (§2) |
| Soon timely acquisition | 7 days, at 90% (placeholder) | **OWNER DECISION REQUIRED** |
| Routine timely acquisition | 30 days, at 90% (placeholder) | **OWNER DECISION REQUIRED** |
| Maximum wait | urgent 7, soon 21, routine 31 days (placeholder) | OWNER DECISION REQUIRED |
| Inherited orders | from max(wanted, takeover); maximum wait only | as built |

**Reliability:**

* **Shared allowance:** at most 4 of 40 seeds failing any of service,
  severity, access or backlog. Each seed is counted once, and an
  insufficient gate counts as failing.
* **Absolute violations** invalidate the plan regardless of the
  allowance.
* The 40-seed decision is an empirical acceptance rule. It is not a
  population-level confidence claim.
* **Unchanged:** v4's service thresholds and backlog rules, and every
  historical policy file and digest.

## 2. The stopping point: urgent access cannot be justified as specified

The owner asked to stop and report if a required clinical decision
cannot be justified from the evidence. This one cannot.

**The evidence, no action, ten development seeds**
(`access-denominators.md` §4):

* 65% of urgent orders are **lost**, declined at the first offer. 75%
  of urgent orders tolerate only the same day, and booking reserves
  nothing for urgent work.
* Over all eligible urgent orders the timely fraction at 48 hours is
  **0.33** (0.30–0.35 per seed). Among the urgent orders served, 99%
  are within 48 hours.
* **No plan action can reserve urgent capacity** or change the booking
  templates. Plans can change staffing, which accounts for about two
  thirds of the urgent losses. Even recovering all of those, urgent
  timely access would be about 0.76.
* **So a 90% or 95% threshold over all eligible urgent orders rejects
  every plan.** It would measure the scenario's booking structure, not
  the plan.
* The synthetic models reserve capacity and lose almost no urgent
  orders, so they cannot inform this choice.

**Options for the owner** (the evidence does not choose between them):

| Option | What it gates | Consequence |
|---|---|---|
| **A. Absolute access over all eligible orders** (as built) | timely ÷ every eligible urgent order, at a threshold reachable here | **Any threshold reachable today (≤ 0.35) has no clinical meaning.** A meaningful one (≥ 0.9) makes every plan ineligible until the scenario's booking changes, and that is a simulation change outside this slice. |
| **B. Two gates** | (i) the **timeliness** of served and open eligible orders, open ones untimely past the limit, at an absolute threshold (90% within 48 h is met by served urgent orders today); (ii) **lost and left demand**, per priority and per seed, no worse than the paired baseline (strengthening v4's pooled overall +2 points), and reported absolutely | The audit's witnesses still fail: 30 urgent orders 30 days late fail (i); 30 urgent orders lost (6.5 points) fail (ii). Absolute loss becomes a reported measure, not a gate. Needs an evaluator change and the owner's tolerance for the loss gate. |
| **C. Absolute access with the scenario's loss as the baseline** | timely ÷ eligible, no more than k points below the paired baseline | This is a relative rule: it adds little over B (ii) and loses the absolute timeliness check. |

**Recommendation: B.** It keeps both of the owner's concerns:

* **no survivorship:** open and deferred orders count, and every loss
  counts;
* **the audit's month-late urgent witness fails.**

It also does not reject every plan for a booking structure that no plan
can change. B is a policy-design change: it is **not implemented** here,
pending the owner's decision.

## 3. Simulator no-action check of the reporting limits

The candidate's reporting measures on the ten no-action records
(`sim-access-5c4b.json`). This baseline reads at utilization 0.95,
without weekend reading, and already fails v4's soon breach gate (0.36).

| Measure | Seeds outside the candidate limit | Worst values |
|---|---|---|
| Urgent absolute 24 h | **8 of 10** (1–5 studies each) | 38–45 h past due: urgent studies acquired on Saturday afternoon, read on Monday |
| Soon absolute 48 h | 0 | 42–45 h (3–6 h of margin) |
| Routine absolute 14 days | 0 | 40–43 h |
| Severe rate 5%: urgent / soon / routine | 0 / **5** / 0 (soon in T90 or the tail) | soon 7.0%; urgent 3.5%; routine 3.0% |
| Severe a week at most 5: urgent / soon / routine | 0 / 3 / **10** | routine 20–37 in the worst week; soon 6–7 |

* **The routine weekly count of 5 fails every simulator seed.** Routine
  work is 86% of acquisitions, about 310 a week. A routine severe rate
  of 1–3% gives 3–9 a week on average, and 20–37 in the worst week.
* **The fixed weekly count does not scale with volume.** It was
  calibrated on synthetic practices with no routine severe delays.
* **Owner decision required:** a weekly count proportional to the
  priority's weekly volume, or a larger routine count. The urgent and
  soon counts are not affected.
* **The urgent 24-hour limit fails this baseline.** This is the weekend
  gap the synthetic run showed: Saturday-afternoon urgent studies wait
  until Monday. It is not a calibration defect: the owner's choice is
  that such a practice must change.

## 4. Witnesses (`internal/eval/clinical_witness_test.go`)

All under the candidate's values. All pass, meaning each produces its
expected classification and outcome.

| # | Witness | Classification | Denominator | Numerator | Gate outcome |
|---|---|---|---|---|---|
| W1 | 30 urgent acquisitions 30 days late | served late, untimely | every urgent order wanted in the gated period | others timely | timely access and maximum wait fail every seed: **NO GO**; v4 **GO** |
| W2 | 30 urgent lost | untimely | all eligible | −30 | **fail** |
| W2 | 30 cancelled by the patient after 48 h | untimely | all eligible | −30 | **fail** |
| W2 | 30 cancelled within 48 h | withdrawn | −30 | unchanged | pass (the cause is not recorded: §2 of access-denominators) |
| W3 | 30 urgent open past the deadline | untimely (never dropped) | unchanged | −30 | **NO GO**; maximum wait fails |
| W3 | open, within the limit at the end | undecided | — | — | gate insufficient (not a pass) |
| W4 | 10 displaced three times, never imaged; 10 displaced twice, imaged 5 days late; 10 displaced twice, imaged within 40 h | 20 untimely, 10 timely | unchanged | −20 | **NO GO**; rescheduling within the limit is not a failure |
| W5 | urgent wanted Saturday 17:00, imaged Monday 08:30 | 39.5 calendar h, 1.5 open h | 1 | timely at 48 h, untimely at 24 h | the open-hours clock would hide it; lost instead, untimely under both |
| W6 | patient cancellation at 2 h / 47 h / 60 h; rebooked no-show with no time | withdrawn / withdrawn / untimely / untimely | 4 | 0 | ambiguity marked |
| W7 | wanted 1 h before the horizon and imaged 10 h after; wanted 1 h after; wanted before and never imaged; received before the takeover | in, timely / out / in, untimely / inherited, timely | +2 gated, +1 inherited | +1 | as specified |
| W8 | one report 348 h past due | routine absolute violation | — | — | **NO GO** (clinical safety) |
| W9 | six urgent reports 8 days late in one week | urgent absolute and weekly count | — | — | **NO GO** |
| W10 | four service-failing and four other backlog-failing seeds | 8 seeds failing | — | — | **NO GO** (8 > 4); v4 passes |
| W11 | imaged before wanted; served without a resolution time; an outcome contradicting a later one | — | — | — | record **INVALID** (never a pass or fail) |
| W12 | 30 urgent orders relabelled routine | invisible to the clinical gates | — | — | **NO GO** through v4's case selection |
| W12 | 20 urgent appointments deferred past the horizon | served late, untimely | unchanged | −20 | **NO GO** |

## 5. External validity (Phase 7)

**A. Supported by clinical rationale** (the owner's, not validated
standards):

* the urgent 24-hour and soon 48-hour reporting maxima;
* zero tolerance for them;
* the calendar access clock;
* counting lost and open demand;
* one shared allowance.

**B. Supported by synthetic calibration** (fidelity run, regime 0.85):

* every reporting limit costs no stable seed: urgent 24 h, soon 48 h
  and routine 14 days are admissible at zero tolerance, and the 5%
  rate and the weekly count of 5 cost nothing;
* the weekend gap fails 70% of E3-like seeds at 24–36 h;
* harmful reading losses are detected.

Caveats:

* The synthetic model's tails are heavier than the simulator's at its
  own load (findings-fidelity §8), and unvalidated at 0.85 and 0.92.
  Its rare-violation probabilities (0 of 10,000; 0.2–0.4% at 0.92) are
  model results, not simulator facts.
* Its synthetic weekly counts used evaluator weeks offset from the
  calendar (synthetic takeover instant; reserved-validation-protocol
  §3).

**C. Supported by simulator observation** (no action, 10 seeds, at 0.95
without weekend reading):

* the soon 48-hour and routine 14-day maxima hold;
* the urgent 24-hour maximum fails the baseline's weekend;
* the routine weekly count fails;
* urgent access is loss-dominated.

**D. Lacking adequate support:**

* the urgent access threshold (§2);
* the soon and routine access limits and fractions (no clinical
  rationale given, and the simulator's are loss-dominated: 0.58 and
  0.83);
* the maximum waits;
* the routine weekly count (§3);
* the severity definition (24 h for every priority);
* the 0.92 regime's zero-tolerance brittleness (8–16% rejection over 40
  seeds): to be accepted explicitly or not.

**The other caveats:**

* **Simplified booking and the weekday mix:** they affect access
  (provisional) more than reporting.
* **Weekend coverage:** the structural ~40-hour gap is robust across
  the synthetic models and the simulator.
* **High-utilization stress:** a stress test, never a stable reference.

## 6. Owner decision table

| Decision | Proposed | Status |
|---|---|---|
| Urgent reporting maximum | 24 h past due, zero tolerance | provisional; fails the no-action baseline's weekend (intended) |
| Soon reporting maximum | 48 h past due, zero tolerance | provisional; supported (B, C) |
| Routine reporting maximum | 14 days past due | provisional; clinical justification needed |
| Severe-delay rate | 5% per priority | provisional; severity definition needed |
| Weekly severe-delay count | 5 per priority | **OWNER DECISION REQUIRED**: unsuited to routine volume (§3) |
| Urgent access clock | calendar, from the original wanted time | recommended; open-hours time reported at most |
| Urgent access threshold | 48 h at 90% / 95% | **BLOCKED**: choose option A, B or C (§2) |
| Soon access threshold | 7 days, 90% (placeholder) | OWNER DECISION REQUIRED |
| Routine access threshold | 30 days, 90% (placeholder) | OWNER DECISION REQUIRED |
| Cancellation treatment | a patient outcome before the deadline is withdrawn; at or after it, untimely; the cause is unrecorded | recommended |
| Lost demand | always untimely (option A), or a per-priority loss gate against the baseline (option B) | **BLOCKED** with urgent access |
| Zero-tolerance scope | the reporting maxima, every evaluated seed, completed and unfinished studies | provisional; the 0.92 brittleness to be accepted explicitly |
| Shared seed allowance | 4 of 40 across service, access, severity and backlog | provisional |
| Intended utilization domain | stable reference 0.85 core load; 0.92 stress only | provisional |
