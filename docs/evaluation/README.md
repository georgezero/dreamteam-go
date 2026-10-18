# V0.2 evaluation framework (v0.2-eval-4, Slice 5C.3)

This directory freezes how V0.2 strategies are judged. It is a
preregistration: the rules come from the owner's decisions, the design
(`V0.2_DESIGN.md` §13), the arithmetic of the gates and synthetic
operating characteristics. No candidate strategy was run to set them.

**v0.2-eval-4** (Slice 5C.3) supersedes v0.2-eval-3. It recalibrates
only the backlog gate (§4, §12), from the owner's decisions and a
prespecified protocol on synthetic queues
(`docs/calibration/backlog-5c3/protocol.md`):

* the individual 28-day peak is reported, never gated;
* the level check keeps the workload-adjusted 28-day mean, with its own
  tolerance;
* the overdue, stale and exposure checks judge block medians;
* an absolute stale cap holds in every block from the reference on.

It also states how the service rule's 10% allowance is to be read under
clustering (§2). Every other rule of v0.2-eval-3 is unchanged:
* the breach gates and the joint seed rule;
* case selection, solvency and the dated funding test;
* T90 as the primary financial endpoint.

v0.2-eval-3 and its README (`README-v0.2-eval-3.md`) are kept byte for
byte, and its development report is unchanged.

**Independent audit and the proposed successor** (Slice 5C.4a).
* **Audit.** The read-only audit of this framework at 7707dbd
  ([`../audits/slice-5c3-eval4-framework-audit.md`](../audits/slice-5c3-eval4-framework-audit.md))
  found it **NO GO for the confirmatory run**. Executed witnesses show
  that severe reporting delays (H1) and delayed access (H2) can pass
  every clinical gate.
* **Proposal.** A successor, v0.2-eval-5, is proposed in
  [`proposals/v5-clinical-acceptance.md`](proposals/v5-clinical-acceptance.md).
  It is **not active**.
* **v0.2-eval-4 stays in force.** It is unchanged, byte for byte.
* **Exposure.** The limited use of holdout seeds by ordinary tests is
  recorded in [`holdout-exposure-5c4a.md`](holdout-exposure-5c4a.md).

**v0.2-eval-3** (Slice 5C.2) superseded v0.2-eval-2. It made four
corrections:
* funding became a dated settlement test;
* the backlog checks became workload-adjusted;
* service reliability became reported cluster-aware;
* an unrepaid transition became CONDITIONAL GO.

**v0.2-eval-2** superseded v0.2-eval-1 (Slice 5C.0). It corrected the
loopholes found by the independent audit of v0.2-eval-1
(`docs/audits/slice-5c0-eval-policy-audit.md`). No evaluation was ever
run under v0.2-eval-1.

| File | What it is |
|---|---|
| `policy-v0.2-eval-4.json` | The policy in force (`dreamteam.evalpolicy/v4`), decoded strictly by `internal/eval`. Its SHA-256 is the policy digest every report carries. Its `changes` list every rule changed from v0.2-eval-3, with the finding, the old and new definitions, the rationale and the expected impact. |
| `oc-v0.2-eval-4.json` | The operating characteristics (§2, §4): the exact independent design, the service rule under day and week clustering, and the backlog family on synthetic queues with the protocol's selection. `TestOperatingCharacteristics` recomputes its exact part and checks the rest, including that the frozen backlog rule is the selected candidate. |
| `policy-v0.2-eval-3.json`, `oc-v0.2-eval-3.json`, `README-v0.2-eval-3.md` | v0.2-eval-3, superseded and unchanged. `LoadPolicy` refuses it. |
| `policy-v0.2-eval-2.json`, `oc-v0.2-eval-2.json`, `README-v0.2-eval-2.md` | v0.2-eval-2, superseded and unchanged. `LoadPolicy` refuses it. |
| `policy-v0.2.json`, `README-v0.2-eval-1.md` | v0.2-eval-1, superseded and unchanged. `LoadPolicy` refuses it. |
| `holdout-v0.2.json` | The locked holdout manifest: method and digest, not the seeds. Unchanged. |
| `holdout-unseal.json` | Not present. Committing it, with the owner's approval, is what unseals the holdout. |

The code is `internal/eval`, a pure package that reads records and never
a simulation:

* `Measure` turns one seed's record into its measures and integrity
  checks.
* `Evaluate` applies the policy to a set of plans. It gives the gates,
  paired comparisons, verdicts and classifications.
* `TailEnd` and `ObservationEnd` tell a harness the common horizon and
  how long to observe, before it runs.

The recorder that produces records from a run is test-only, in
`internal/sim/export_eval_test.go` (`World.NewEvalRecorder`). It only
reads state: the integration test checks that a recorded run's state
digest equals an unrecorded one's.

Every threshold is compared **exactly**. A policy fraction is a whole
number of basis points, and `DecodePolicy` refuses one that is not. A
rate is compared through integer cross products, so a value exactly at a
tolerance is within it. No floating-point subtraction decides a gate.

Values marked **[owner]** are the owner's decisions. Values marked
**[proposed]** fill a gap the design left open. Each is set from the
measurement units, not from results.

## 1. Horizons and windows

| Window | Span | Role |
|---|---|---|
| T30 | (t0, t0+30 d] | Feedback. Informational. |
| T90 | (t0, t0+90 d] | **Primary** [owner]. Gated (pooled rates; soon and routine per seed). |
| D31–90 | (t0+30 d, t0+90 d] | Operating performance after the transition. Informational. |
| Tail | (T90, H] | Gated (pooled rates; soon and routine per seed). |
| Gated period | (t0, H] | T90 and the tail together: the urgent per-seed cohort (§2). |
| T180 | (t0, t0+180 d] | **Validation** [owner]: per seed and pooled, part of the GO condition. |

* t0 is the takeover instant (Tuesday 2026-10-13 06:00). Windows are
  half-open, (a, b].
* **Common horizon H** [owner, 2026-10-17].
  * Each plan's own tail rule is `max(T90 + 60 d, latestEffect + 30 d,
    latestCommitment)` [owner].
  * The latest commitment is the end of the latest minimum obligation
    accepted by the last action: a teleradiology contract's minimum
    term, or the end a cancellation would give it.
  * H is the latest of these over every plan of the evaluation (the
    frozen strategy set). Every plan and the baseline are observed over
    the same calendar intervals and to the same instant.
  * A record whose horizon or observation end differs from the others,
    or whose horizon falls short of its own tail rule, makes the
    evaluation **INVALID**. Unequal tail lengths are never compared
    under one metric name.
* **Closure** [proposed]: every cohort is observed 1,440 minutes (the
  longest TAT target) past its window, which decides every study's
  breach status.
* **Observation end**: `max(H, T180) + closure`, or the end of the
  backlog block covering H if that is later (§4). For a day-150 horizon
  that is day 181; for day 170 it is day 194.
* **Daily samples** are taken at 06:00, days 1, 2, …, every day through
  the end of observation.

## 2. Service reliability

The breach gates are the owner's provisional T90 gates of 2026-10-15,
unchanged [owner]: urgent ≤ 12%, soon ≤ 32%, routine ≤ 6%.

**Joint seed reliability** [owner, H5]:

* A seed **fails** if any of its per-seed service gates fails or lacks
  evidence. Its per-seed gates are urgent over the gated period, plus
  soon and routine in each of T90 and the tail: five gates.
* **At most 10% of seeds may fail jointly**: 4 of 40, 2 of 20, 1 of 10.
  If more seeds lack evidence than that, service is *insufficient*.
* v0.2-eval-1 applied the 10% to each gate separately. Up to 24 of 40
  seeds could then fail some gate while every gate passed.

**Pooled rates** (all seeds' breaches over all their acquisitions) must
also pass for every priority in T90 and in the tail, separately. A
pooled rate is undefined, and the gate insufficient, if any seed's
cohort is missing or undecided.

**Urgent decided per seed over the whole gated period** [owner design,
proposed window, H1]. The 12% gate is unchanged. The evidence window
was redesigned and chosen by its operating characteristics
(`oc-v0.2-eval-4.json`, `exact`, as in v0.2-eval-2), never by results:

* **Model.** Synthetic only:
  * the planning target is the owner's: urgent 8% underlying;
  * soon (21%) and routine (4%) are planned at about two-thirds of
    their gates;
  * each seed's daily volume is drawn from the no-action development
    range (urgent 0.79–1.19 a day, soon 5.3–6.5, routine 43–45);
  * all five per-seed gates and the joint rule are applied.
* **Exact results** (independent studies, seeds alike in their rates).
  Each cell is the probability that an evaluation rejects a plan whose
  urgent rate is 8%.

  | Design | 10 seeds | 20 seeds | 40 seeds |
  |---|---|---|---|
  | v0.2-eval-1: urgent per seed in T90 and the tail | 0.756 | 0.903 | **0.981** |
  | **v0.2-eval-2: urgent per seed over (t0, H]** | 0.069 | 0.056 | **0.030** |
  | Alternative: urgent over (t0, H], 90-day tail | 0.038 | 0.024 | 0.008 |
  | Alternative: urgent over (t0, H], 120-day tail | 0.021 | 0.010 | 0.002 |

  * **Power.** At a true 10% the frozen design rejects 0.93 of the
    time on 40 seeds; at 12% and above, it always rejects.
  * The joint 10% rule makes the per-seed gate demanding: a plan must be
    well under 12% on nearly every seed.
* **The target is met under this model**: at most 5% false rejection on
  the 40-seed holdout (0.030) with the owner's tail. It is not met on 10
  development seeds (0.069). Development-seed verdicts are exploratory.
* **Dependence** (Monte Carlo, 4,000 evaluations of 40 seeds each, in
  the same file). Breaches clustered by week (intra-week correlation
  ρ) or seeds that differ in their true rate (SD τ) raise false
  rejection sharply:

  | 40 seeds, urgent 8% | ρ = 0 | ρ = 0.02 | ρ = 0.05 |
  |---|---|---|---|
  | τ = 0 | 0.029 | 0.374 | 0.962 |
  | τ = 0.01 | 0.107 | 0.678 | 0.991 |
  | τ = 0.02 | 0.899 | 0.992 | 1.000 |

  * Lengthening the tail to 120 days brings the ρ = 0.02, τ = 0 case to
    0.044, but helps little elsewhere.
  * Seeds that differ in their true rate are partly a real failure, not
    a false one: a seed truly above 12% should fail.
* **The trade-off is not resolved by weakening the gate.** Under
  dependence, the ≤ 5% target is infeasible without:
  * much longer observation;
  * a larger seed allowance;
  * or a per-seed rule judged by evidence, such as a confidence bound
    against 12%.

  Each of these is the owner's decision (*Open decisions*).
* **Measured dependence.** Every gate reports a **dispersion index**:
  the variance of the seeds' rates over the binomial variance their
  pooled rate and sizes imply. It is about 1 for independent studies
  and above 1 for seed variation or clustering. It is reported, never
  gated, so that the development seeds show whether the independence
  assumption holds.

**How the holdout result is to be read** [owner, 5C.3]. The
thresholds are kept: urgent ≤ 12%, soon ≤ 32%, routine ≤ 6%, and at
most 10% of seeds failing jointly. Four quantities are kept apart, and
no report may substitute one for another:

| Quantity | What it is | Where it is reported |
|---|---|---|
| **Empirical acceptance** | The prespecified rule's verdict on these seeds: at most 4 of 40 fail. This, and only this, is what passes or fails. | `serviceJoint.status`, `passFraction` |
| **Population reliability** | What the passing fraction shows about all seeds the plan could meet: the exact one-sided 95% upper bound on the failing share. Passing with 4 of 40 failing bounds it at 21.4%, not 10%. | `failingFractionUpperBound` |
| **False-rejection probability** | The chance that a plan whose true rates are acceptable fails the rule. It depends on how breaches cluster, not only on the rates. | `oc-v0.2-eval-4.json`, `cluster` |
| **Statistical power** | The chance that a plan at or beyond a gate fails the rule: 1.00 at every gate under every dependence model studied. | the same |

* The holdout is a **strict empirical acceptance test**. It is not a
  95%-confidence demonstration that a plan's population failing share is
  below 10%. The choice favours clinical reliability over statistical
  power, and the risk of rejecting an acceptable strategy is explicit
  (the tables below).
* **Independent patient breaches do not model the run.** The
  development runs cluster breaches by week (soon design effect
  1.7–5.9). The independent-breach operating characteristics, and the
  statement that the planning rates (urgent 8%, soon 21%, routine 4%)
  are rejected at most 5% of the time on 40 seeds, are kept only as a
  lower bound on false rejection under independence. They are not a
  guarantee of reliable passage. Under the clustering observed, a plan
  at the planning rates is rejected most of the time; a plan needs
  margin well below them.

**Cluster-aware evaluation** [owner, 5C.2]. The development runs showed
breaches clustered in time (`docs/calibration/policy-5c2/clustering.md`):

* soon: week design effect 1.7–5.9 for plans near the gate;
* urgent: 1.0–1.3;
* routine: about 3–4, from its volume.

The thresholds, the joint rule and its 10% allowance are unchanged: the
gates are empirical and assume nothing about independence. What assumed
independence was every *probability* attached to them. v0.2-eval-3
therefore reports, never gates:

* per gate, each seed's day (weekday-adjusted) and week design effects,
  and their means;
* each pooled rate with a **seed-cluster-robust 95% interval**: seeds
  are the independent units, so the interval holds whatever the
  dependence inside a seed;
* for the joint rule, three quantities kept apart:
  * the **empirical pass fraction**, which is what the rule judges;
  * the **exact one-sided upper bound** (95%) on the population share
    of failing seeds. This is the *confidence in population
    reliability*;
  * the **power** to detect unacceptable service, from the operating
    characteristics.

**The 10% allowance, read prospectively.**

* Passing means that at most floor(10% × seeds) seeds failed. It does
  *not* establish that the population's failing share is within 10%:
  * with 40 seeds and 4 failing, the 95% upper bound is 21.4%;
  * with 0 of 40 it is 7.2%;
  * with 0 of 10 it is 25.9%.
* A GO is an empirical pass of a prespecified rule, not a confidence
  statement.

**Operating characteristics under dependence** (`oc-v0.2-eval-4.json`,
`cluster`). The frozen per-seed rule is simulated day by day over the
150-day gated period. The figures are the probability of rejecting a
plan whose underlying rates are the planning rates (urgent 8%, soon 21%,
routine 4%):

| Model | Per-seed failure | 10 seeds | 40 seeds |
|---|---|---|---|
| Independent | 0.041 | 0.059 | 0.022 |
| Variable volume (weekday profile, day CV 0.25) | 0.048 | 0.080 | 0.041 |
| Day-clustered, ICC 0.05 | 0.050 | 0.086 | 0.048 |
| Day-clustered, ICC 0.15 | 0.152 | 0.465 | 0.751 |
| Week-clustered, ICC 0.02 | 0.096 | 0.248 | 0.338 |
| Week-clustered, ICC 0.05 | 0.233 | 0.715 | 0.972 |
| Week-clustered, ICC 0.10 | 0.365 | 0.928 | 1.000 |
| Development range (day 0.15; week urgent 0.03, soon 0.11, routine 0.012) | 0.225 | 0.695 | 0.964 |

* **Power is high**, whatever the model: urgent at 12%, soon at 32% or
  routine at 6% is rejected with probability 1.00 on 40 seeds.
* **False rejection is not.** The ≤ 5% target at the planning rates
  holds only for independent, volume-variable or mildly day-clustered
  breaches.
  * At week ICC 0.02 the planning rates are rejected 34% of the time.
  * Across the development range they almost always are.
* **What passes reliably under dependence** (40 seeds, ≤ 5% rejection;
  the others at their planning rates) is far below the gates:
  * at week ICC 0.02, nothing on the grid of the varied priority
    passes, because soon at 21% is itself rejected about 34% of the
    time;
  * a plan near the gates needs margin well beyond the independence
    design's.

The ≤ 5% aim of v0.2-eval-2 (§2 above) is therefore met only under
independence. Under the dependence the development runs show, the
frozen design is stricter than its planning target. This was not
weakened: thresholds and the allowance are the owner's.

**A cluster-aware estimate for a candidate** is exploratory, not a
gate (`docs/calibration/policy-5c2/block_bootstrap.py`). It resamples
whole weeks within seeds, and seeds within the 10 development seeds, so
the run's own dependence is kept.

A time-local urgent failure confined to the tail of one seed is no
longer caught per seed. It is caught when it moves the tail's pooled
urgent rate, or by the backlog checks (§4).

**Evidence minimum** [proposed]: urgent 43, soon 88, routine 23
acquisitions per cohort, `⌈g(1 − g)/0.05²⌉`. This is an **evidence
floor**, not a precision guarantee. 5 breaches of 43 passes, and its
Wilson 95% interval is 5.1%–24.5%. The gated-period urgent cohort holds
about 118–178 studies, far above the floor. A cohort under the floor,
empty, or with studies still undecided is *insufficient*: never a pass,
never a 0% rate.

**T180 validation**: the same joint rule and pooled rates over T180,
with every priority per seed.

## 3. Fixed acquisition cohorts

Unchanged from v0.2-eval-1. A cohort is every study acquired in the
window, finished or not:

* the counts acquired, finalized and unfinished;
* completed-only turnaround (never gated);
* unfinished ages;
* overdue unfinished studies and overdue study-minutes;
* breaches decided at each study's due time;
* unresolved studies;
* the legacy as-of-end rate, beside it.

## 4. Backlog

The rota-aligned 28-day blocks are kept [owner]. Each starts on a
week-A Monday (`rotaWeekA` 2026-10-12). The **reference block** is the
last whole block ending by T90 (days 55–82 for this takeover).

**Every later block, through the one covering H**, is checked against
the reference [proposed, H4]. They follow it end to end: 83–110,
111–138 and 139–166 for a day-150 horizon. The last one closes the
residual days after the last block ending by H.

**Workload adjustment** [owner, 5C.2]. The workload of a block is the
number of studies acquired in it.

* *Why:* the reference block of this takeover (Dec 7 – Jan 3) falls in
  the scenario's December demand low. The season factor is 0.93 in
  December, against 1.05 in January and 1.00 in February and March.
* *The rule:* the workload-adjusted checks compare a block with the
  reference scaled by w, the block's workload over the reference's.
  * This is Little's law: at a stable time in system, the backlog is
    proportional to the workload. A seasonal rise that a stable queue
    follows in proportion is therefore not instability.
  * Only the plan's own observed acquisitions are used.
  * The comparison is exact: the block's sum × the reference's
    acquisitions against the reference's sum × the block's acquisitions.
  * A block or reference with no acquisitions is compared unscaled.
* *What never scales:* stale and persistent studies are clinically
  important delays whatever the demand, so they stay absolute
  (`absoluteChecks`).

**v0.2-eval-4** [owner, 5C.3]. The owner decided that the peak is
reported only and that every sustained safeguard stays hard. The
tolerances and statistics were then selected by a prespecified protocol
on synthetic queues (`docs/calibration/backlog-5c3/protocol.md`, below).

The checks of each later block against the reference:

| Check | Statistic of the block's daily 06:00 samples | Scaling | Tolerance | Role |
|---|---|---|---|---|
| **Level** | Mean unfinished count | × w | max(**150%** of the scaled reference, 2 studies) | hard |
| **Overdue** | **Median** count of unfinished studies past their due time | × w | max(10%, 2 studies) | hard |
| **Stale** | **Median** count more than 1,440 minutes past due | absolute | max(10%, 2 studies) | hard |
| **Overdue exposure** | **Median** Σ minutes past due of the unfinished | × w | max(10%, 2,880 study-minutes) | hard |
| **Stale cap** | Median stale count, in **every block from the reference on**, the reference included | absolute | at most **3 studies** | hard |
| **Persistent** (maximum overdue age) | Any study unfinished at a sample **more than 14 days past its due time** | absolute | none, on any sample from day 1 through the covering block | hard |
| Mean (the v0.2-eval-3 mean check) | Mean unfinished count | × w | max(10%, 2 studies) | reported |
| Peak | Largest daily count | × w | max(10%, 2 studies) | reported |
| Recovery | Days from the block's peak to the first sample at or below the scaled reference mean (−1: not recovered through the covering block) | — | — | reported |

* **The level allowance in full** [audit H1, Slice 5C.4a].
  * The check passes when B_later ≤ w·B_ref + max(1.5·w·B_ref, 2 studies).
  * So "150%" is the allowance *above* the workload-adjusted reference.
    The ceiling is **250% of it** (2.5 × w·B_ref), or the reference plus
    2 studies, whichever is larger.
  * A block mean may therefore reach two and a half times the adjusted
    reference and still pass.
* **The age limit is sampled.** It is checked at the daily 06:00
  samples through the covering block, not from each study's timestamps.
  A study finished between samples can exceed 14 days past due by up to
  a day without being seen (audit H1; executed witness: 348 hours past
  due, 334 observed).
* **Seeds.** A seed is *stable* when every hard check passes. The plan's
  backlog gate passes if at most 10% of seeds are unstable.
* **Never normalized:** stale studies, the cap and the age limit
  (`DecodePolicy` refuses a policy that scales them, or that makes the
  age limit informational).
* **No new clinical deadline is invented.** Every check counts lateness
  from each study's own due time (its priority's TAT target).
  * *Stale* uses the closure, the evaluation's own decision horizon: a
    stale study's breach was decided a full day ago.
  * *Persistent* uses one rota cycle (14 days): a study still unread
    after every reader on the rota has been scheduled through a whole
    cycle has not been delayed by the rota phase.
* **Inherited and carried-over studies are in every sample and every
  check.**
* **A record missing its workload or its overdue-age data is never
  evidence.** That covers:
  * a missing daily sample;
  * samples that omit acquisitions;
  * a missing TAT target;
  * a study without a priority or an acquisition time.

  Each is an integrity failure: the comparison is INVALID. A block with
  no recorded acquisitions is compared unscaled.

**Why each choice.**

* *Peak informational.* The maximum of 28 samples measures one burst.
  Under v0.2-eval-3 it alone made about half of stable stationary queues
  unstable.
* *Level: the mean, with a wide tolerance.* The level is the leading
  indicator of a queue that cannot clear. When demand exceeds capacity
  the backlog grows without bound and passes any fixed multiple of the
  reference.
  * A stable queue's level rises faster than its workload as
    utilization rises. This takeover's December reference (season 0.93)
    and January block (1.05), at utilization 0.85, put ρ/(1−ρ) 2.1 times
    apart for a workload only 1.1 times apart.
  * The 10% / 2-study tolerance therefore failed 59% of stable seasonal
    queues, and most stationary failures came from its 2-study floor.
  * The mean detects late-tail deterioration more often than the median
    does.
* *Late work: medians.* A median of 28 daily samples moves only when late
  work is present on most mornings of a block: sustained, not a burst.
  Block means of late work failed 5–15% of stable bursty queues.
* *Stale cap.* Every relative check compares with the reference, so an
  inflated reference licenses the same stale burden later. The cap does
  not.

**Operating characteristics** (`oc-v0.2-eval-4.json`, `backlog`; 1000
synthetic seeds a scenario; the queue of 5C.2 with the stated
departures). Each figure is the share of seeds unstable:

| Scenario | Role | v0.2-eval-3 | v0.2-eval-3 without the peak | **v0.2-eval-4** |
|---|---|---|---|---|
| A stable stationary | stable | 60.5% | 14.2% | **0.0%** |
| B stable seasonal | stable | 88.9% | 59.9% | **0.1%** |
| C stable, bursty days (CV 0.15) | stable | 69.5% | 44.4% | **0.5%** |
| Cs stable seasonal, bursty days | stable | 92.2% | 83.9% | **3.8%** |
| D capacity 20% lower from day 100 | deteriorating | 100% | 100% | **100%** |
| E 10 inherited studies unread to day 120 | deteriorating | 100% | 100% | **100%** |
| F one study a weekday held 12 days | deteriorating | 100% | 100% | **100%** |
| G capacity 25% lower, days 90–117, then recovery | deteriorating | 100% | 100% | **100%** |
| H capacity 20% lower from day 140 (late tail) | deteriorating | 100% | 100% | **98.5%** |
| I reference inflated, then 20% lower from day 111 | deteriorating | 88.7% | 88.7% | **97.6%** |
| L5 capacity 5% lower (still stable) | ladder | 96.7% | 87.0% | 0.2% |
| L10 capacity 10% lower (still stable) | ladder | 99.9% | 99.6% | 22.8% |
| L15 capacity 15% lower (utilization 1.0) | sensitivity | 100% | 100% | 96.6% |
| K capacity eroding 0.2% a day from day 83 | sensitivity | 99.9% | 99.7% | 45.9% |
| A92 stationary, utilization 0.92 | sensitivity | 71.0% | 47.8% | 0.2% |
| B92 seasonal, utilization 0.92 | sensitivity | 98.2% | 95.9% | 12.4% |
| Cx seasonal, bursty days (CV 0.30) | sensitivity | 93.2% | 91.3% | 36.7% |

**What the rejection probabilities mean.**
* **Stable queues pass the holdout.** The worst stable scenario is
  rejected at 40 seeds with probability **0.017** (Cs; the target is ≤
  0.05).
* **Every deteriorating scenario is detected.** Each is unstable on at
  least 97.6% of seeds (target ≥ 90%), so it is rejected with
  probability 1.00.
* **The trade-off is explicit.**
  * A step change that leaves a queue stable at a higher utilization is
    mostly not unstable: 0.2% of seeds at 5% capacity lost, 22.8% at
    10%.
  * Slow erosion is detected on 46% of seeds by the horizon, because
    late work and growth appear only near its end.
  * A seasonal queue already at utilization 0.92 is unstable on 12.4% of
    seeds, a 40-seed rejection of 0.56. That is outside the targets
    (sensitivity), and it is the main false-rejection risk.

**v0.2-eval-3's false failures, attributed** (the 5C.2 streams, 300
seeds; reproduced exactly):

| | Unstable | Peak alone | Peak and mean | Mean alone |
|---|---|---|---|---|
| Stationary | 63.7% | 50.3% | 12.7% | 0.7% |
| Seasonal | 87.7% | 27.0% | 57.0% | 3.7% |

No other check failed a stable unbursty queue. With bursty days, overdue
and exposure also failed 4–12% of seeds, in intermediate blocks more
than in the covering one (`attributionEval3`).

**Tests.**
* `TestBacklogStability` (v0.2-eval-4) covers each case of the protocol,
  beside v0.2-eval-3 where they differ:
  * bursty workload and seasonal rises pass;
  * temporary spikes pass, the peak and recovery reported;
  * growth without bound, persistent and aging old work, late-tail
    growth and an inflated reference fail;
  * a record missing its workload or overdue-age data is INVALID.
* `TestBacklogSafeguards` and `TestSeasonalBacklog` keep the earlier
  rules' cases under v0.2-eval-3's rule.
* **Legacy.** The historical single-sample rule (day 180 ≤ 1.1 × day 90)
  is still reported, never gated.
* **Also reported:** stage means and medians, ages, rolling 28-day
  means, slope and flow.

## 5. Financial eligibility

Raw profit is the posting-based evaluation window's operating profit,
baseline-adjusted. T90 is primary, T30 feedback, D31–90 operating
performance and T180 validation.

**Obligation bridge** (`Money.Obligations`, at every capture and every
daily sample). Each component starts where the one before stops, so
nothing is counted twice, and none is added to the P&L:

| Component | Covers |
|---|---|
| Payables, accrued payroll (with the guarantee provision), severance payable | Liabilities recognized, from the ledger |
| Earned unposted wages | Hourly pay of shifts in progress |
| **Employee commitment** [owner, M3] | Contracted pay with benefits for the **14 days after** the instant's day, for everyone employed then. Hourly staff count their contract day guarantee, salaried staff their daily salary at the allocation in force. A leaver counts to the last day and a hire from the first. 14 days is the notice an employer must give (`NoticeDays`). Today's pay is in accrued or earned pay already. |
| Remote in-flight fees | Fees locked by remote claims not yet final |
| Contract minimums | Retainers still owed to each contract's fixed end, or its earliest end if cancelled now |
| Past due | 0 in V0.2: nothing can be deferred |

Receivables and unbilled work are never set against the bridge, and are
never cash.

Since v0.2-eval-3 the bridge is split by kind (`Liquidity`, at the
horizon and at the end of observation):

| Kind | Components |
|---|---|
| Payable now | Past due |
| Accrued, paid in the recurring cycle | Payables, accrued payroll, earned unposted wages |
| Committed, outside the cycle | Severance payable, contract minimums, remote fees in flight |
| Future wages, not yet earned | The 14-day employee commitment |
| Cash | Transaction-level cash |
| Receivables | Listed only |

* **Solvency** [owner]: the lowest cash after any posting, through the
  end of observation, may not fall below 0 on any seed.
  * The recorder follows the ledger entry by entry, from the takeover.
  * It records the minimum and its posting time.
  * It checks that the minimum equals the evaluation window's own.
* **Funding: a dated settlement test** [owner, 5C.2]. It replaces
  v0.2-eval-2's requirement that cash cover the whole bridge, the
  forward wages included, at every instant. **No seed may fail it**, and
  it is a hard gate. A seed is funded when all four parts hold:
  1. **Paid when due.** Every payment the model makes falls on its
     date: payroll every 14 days, payables on the 1st, severance on the
     first payday after the last day. Cash after every posting stays at
     or above the floor through the end of observation (solvency).
  2. **Nothing deferred.** Nothing is past due at any capture
     (solvency).
  3. **Settled inside the run.** Observation runs at least
     `settlementDays` (31) past the horizon, so every obligation
     recognized by H falls due and is paid inside the observed run.
     * 31 days is the model's longest recurring payment cycle: payables
       are paid on the 1st for everything accrued before it.
     * The receipts that pay these obligations are the collections the
       simulation actually makes. **Future receivables are never
       forecast or counted as cash**, so no forecast uncertainty
       enters.
  4. **Covered at the end.** At the end of observation, cash covers the
     committed liabilities outside the recurring cycle and anything past
     due.
     * The cycle's accrued liabilities are excluded: the observed run has
       just shown them paid from the practice's own receipts for at least
       31 days.

  Future wages are not payable until the work is done.
* **Liquidity-risk indicator** (reported, never gated): the 14-day
  commitment coverage of v0.2-eval-2. It reports, per plan:
  * the seeds whose whole bridge exceeded cash at some capture or
    sample;
  * the largest gap;
  * the share of instants in gap.
* **Deferral cannot conceal insolvency** (`TestDatedFunding`). Each of
  these fails:
  * a payment past due with ample cash;
  * cash below zero at one posting between positive samples;
  * observation stopping before the settlement window ends (INVALID);
  * severance payable, contract minimums or fees in flight above cash
    at the end.

  The commitment gap alone is reported and does not gate.

## 6. Payback

Unchanged rules: sustained break-even, accrual and cash, right-censored
at the horizon. A median is given only when at least half have repaid.

A seed without daily samples through a horizon is no longer left out
silently. It invalidates the comparison.

## 7. Remote reading metrics

Unchanged. Faster reporting, additional reports and additional
recognized revenue are reported separately.

Worst-seed reporting is now explicit, per window and priority:

* each plan's worst seed rate, with its seed;
* the largest paired increase, with its seed;
* the worst incremental-profit seed.

The teleradiology question (worst-seed soon breach against cash drain)
reads from these, `minCashDeltaCents` and `incrementalCashCents`.

## 8. Access and case selection

**Due cohorts followed to the end of observation** [owner, H3]:

* A window's due demand is every new order (received after t0) whose
  **original wanted time** falls in the window, received in it or
  before it.
* Its outcome is taken at the end of observation, the same instant for
  every plan. An order still open then is not served.
* The day-89 receipt with a day-100 wanted date is due in the tail. A
  plan that leaves it unresolved fails.
* Received demand (loss and abandonment rates) is every order received
  in the window, also followed to the end. A referral received on day
  85 and lost on day 92 counts in T90.
* Also reported:
  * orders carried in (due before the window and open at its start)
    and where they ended;
  * the served count by the window's end;
  * displacements, counted in the window in which they happened (M5).
* A no-show may call for a new date. It is the one outcome that can be
  undone, and records may show it before a later resolution.

**Inherited orders** (open at t0) are their own stratum. They are
followed to the end of observation, with the wait past their wanted
time, and guarded like every other stratum.

**Strata** [proposed, M2]:

* by priority;
* by modality;
* by priority and modality together;
* the inherited orders.

**Case-selection gates**, against the paired baseline, exact [proposed]:

| Gate | Strata | Rule |
|---|---|---|
| Served fraction, per seed | Each priority and modality; inherited | Falls by at most 5 points on a seed with ≥ 30 due on the baseline. At most 10% of seeds may exceed that. |
| Served fraction, pooled | Every stratum | Falls by at most 2 points |
| Acquisitions, pooled | Each priority and modality | Fall by at most 2% |
| Lost referrals, pooled | Overall | Rate rises by at most 2 points |
| Walk-in abandonment, pooled | Overall | Rate rises by at most 2 points |

Denominators:

* served ÷ due, an order open at the end of observation being not
  served;
* acquired ÷ the baseline's acquired;
* lost referrals ÷ referrals received;
* walk-ins left ÷ walk-ins received.

The tolerances are anti-gaming margins, not clinical standards.

**Rare and empty strata** [proposed]: a stratum is never treated as
passing just because it is missing.

* **Fewer than 30 due on the baseline** (pooled, or on a seed for the
  per-seed gate):
  * the stratum is *insufficient* and reported, never a pass;
  * it **fails if the candidate drops it**: the baseline served some
    and the candidate none.
  * Priorities, modalities and inherited orders are **blocking** kinds:
    their insufficiency blocks GO.
  * Priority-and-modality intersections are not blocking: they are
    listed as "reported, not judged".
* **No demand on either plan**: *empty*, reported, nothing to protect.

## 9. Seeds and the holdout

The seed sets are unchanged from v0.2-eval-1:

* development seeds 901–910;
* reference seeds 1001–1020 (exposed);
* holdout `v0.2-holdout-1`: 40 seeds, sealed. Only its digest is
  recorded; its method, label and digest are untouched.

**The canonical runner** (Slice 5C.0b) is the only path that runs a
holdout seed.

* **Sealed for every harness.** `eval.GuardRepo` refuses holdout seeds
  whether or not the holdout is unsealed, and its refusal does not name
  them. Every calibration harness calls it:
  * `parseSeeds`;
  * the dynamic-staffing runner for each seed;
  * the session replay check;
  * the evaluation recorder itself (`World.NewEvalRecorder`).
* **The runner** is `internal/sim` `TestHoldoutRun`. It is the only
  caller of `World.NewHoldoutRecorder`, which needs a protocol from
  `eval.Authorize`; one cannot be constructed outside the package.
* **Other paths.** A run by any other path, the CLI included, produces no
  evaluation record and is not holdout evidence.
* **Development.** Slice 5C.1's runner (`TestStrategyDevelopment`,
  `docs/calibration/strategy-development`) loads a strategy set through
  `eval.Development`.
  * It accepts only the policy's development seeds.
  * Its protocol is never authorized.
  * It records with `World.NewEvalRecorder`, and its records are
    byte-identical to the canonical runner's.

**The artifact contract.** Every file is pinned by SHA-256.

| File | Schema | What it pins |
|---|---|---|
| A strategy file (`docs/evaluation/strategies/<name>.json`) | `dreamteam.strategy/v1` | Name, description, its dated batches of actions, and optionally `teleOnlyOf` (the plan it differs from only by teleradiology) |
| The strategy set | `dreamteam.strategyset/v1` | The scenario manifest, the baseline (a strategy with no batches), and every strategy file's path and digest |
| `holdout-unseal.json` | `dreamteam.holdout-unseal/v1` | Fields listed below |
| `holdout-runs/v0.2-holdout-1/run.json` | `dreamteam.holdoutrun/v1` | The unseal record's digest, the artifacts, the common horizon, every plan × seed record's digest, and the report's digest |
| `holdout-runs/v0.2-holdout-1/report.json` | `dreamteam.evalreport/v2` | The evaluation |

The unseal record names:

* the holdout and its manifest's digest;
* the policy and its digest;
* the strategy set and its digest;
* the exact engine;
* the one **primary** strategy;
* the approver;
* the approval date;
* a statement.

**`eval.Authorize`** refuses (as sealed) unless all of the following hold:

* the manifest regenerates to its digest;
* the unseal record exists, decodes strictly, and is complete;
* every digest it names matches the file;
* every strategy file matches its pin and decodes strictly;
* the baseline takes no action;
* the primary is a candidate, not the baseline;
* `teleOnlyOf` names another strategy;
* the engine is this build;
* every one of these files is committed with no change (`git`);
* the **unseal record was committed by its approver** (the author of
  the commit that added it).

**The run** works through these steps:

1. **Common horizon.** The runner computes the horizon from each
   strategy's accepted actions before running. The records confirm it
   on every seed; if a later seed reaches further, every plan runs again
   with the longer horizon.
2. **Every plan on every holdout seed**, the baseline included. A batch
   at or after T90 is refused.
3. **Evaluation** with the frozen policy, requiring:
   * the complete seed set;
   * every record at its pinned strategy digest, with one digest a plan;
   * the primary named. Every other candidate is reported
     **exploratory**.
4. **Commit, once.**
   * The first run writes `run.json` and `report.json`, read-only, into
     an empty directory.
   * A rerun must reproduce both byte for byte. Otherwise it fails and
     writes nothing: the holdout is then exposed and reference-only, and
     a new one is generated under a new label.
   * The records themselves are not committed. Their digests are, and a
     rerun regenerates them.

The test suite drives the whole protocol on a synthetic two-seed
holdout of its own label, the mini practice and the short policy:
* authorization and the 14 ways it refuses;
* the written run, its reproduction, and a tampered output;
* refused recorders.

The repository's holdout is checked to authorize as sealed.

## 10. Verdicts and classifications

No weighted score. Profit never compensates for a failed gate. The rules
are applied in order (policy `verdict`):

1. **INVALID**: any record, comparison or evaluation integrity check
   fails.
   * The record checks are:
     * the schema, scenario and TAT targets;
     * an action at or after T90;
     * the common horizon, and that it covers the tail rule;
     * observation through every closure and the covering block;
     * each capture exactly once, at its instant;
     * one sample a day, at 06:00 on its date;
     * study timestamps in order;
     * order outcomes consistent with resolution times;
     * the cash bridge and minimum-cash consistency;
     * samples reconciling with studies (backlog, overdue, flow);
     * remote fees reconciling with the ledger.
   * The comparison checks are:
     * identical seed sets, engine, scenario and takeover;
     * one horizon for all records;
     * payback data complete;
     * teleradiology-only booking invariance.
2. **NO GO**: a hard gate fails or lacks evidence. The hard gates are:
   * service (the joint seed rule and the gated pooled rates);
   * case selection (failing strata, and blocking strata lacking
     evidence);
   * backlog;
   * solvency;
   * funding.
3. **GO, financial superiority**: every hard gate passes, and all of
   the following hold:
   * the plan's own mean operating profit is positive at T90 and T180
     [owner];
   * the T90 paired 95% interval of incremental profit lies above zero;
   * T180 validation holds: its joint rule and pooled rates pass, and
     its mean increment is positive.
4. **CONDITIONAL GO, relative improvement**: every hard gate passes and
   the mean T90 increment is positive, without every GO condition. A
   plan that beats a loss-making baseline while losing money itself
   lands here, with the missing conditions listed.
5. **CONDITIONAL GO, transition not yet repaid** [owner, 5C.2]: every
   hard gate passes, the mean T90 increment is not positive, and the
   mean increment over the operating window (days 31–90, the policy's
   post-transition window) is positive.
   * This is a solvent, clinically eligible reorganization that has not
     repaid its transition cost by T90.
   * T180 profit and payback are reported, and never promoted to
     financial GO.
   * If the baseline fails service, the restoration and its T90 cost are
     stated beside it (`serviceRestored`).
6. **CONDITIONAL GO, service purchase** [owner]: every hard gate passes,
   the mean T90 and operating-window increments are not positive, and
   the baseline **fails the service gates** the candidate passes.
   Service is restored at the stated cost.
   * The candidate is financially feasible, since solvency and funding
     are hard gates.
   * A baseline that is insolvent or unfunded but whose service passes
     is not a service restoration.
7. **NO GO, no benefit**: every hard gate passes, there is no T90 or
   operating-window increment, and the baseline's service already
   passes.

Every comparison reports its financial evidence apart (`financial`):

* T90 absolute and incremental profit;
* the operating window's absolute and incremental profit;
* T180 absolute and incremental profit;
* the transition obligations outstanding at T90 (severance payable and
  contract minimums);
* beside them, the accrual and cash paybacks.

Candidate-to-candidate pairs report the same differences without a
verdict.

**Selection** [owner]: the holdout should confirm **one prespecified
primary strategy**. Alternatives are reported as exploratory unless a
multiplicity procedure is frozen before unsealing.

## 11. Records

A record is `dreamteam.evalrecord/v2`. Beyond v1 it adds:

* the common horizon and the latest commitment;
* each order's resolution time and displacement times;
* each sample's obligations;
* each capture's employee commitment and posting-level minimum-cash
  time.

A harness (5C.1 on development seeds, or the canonical runner) proceeds
as follows:

1. Compute H with `TailEnd` over every plan before running.
2. Run every plan to `ObservationEnd`.
3. Sample every day and capture at T30, T90, H (tailEnd), T180 and the
   end.
4. Call `Finish` with H.

## 12. Changes from v0.2-eval-3 (Slice 5C.3)

The owner's backlog decisions on the Slice 5C.2 findings, and the
calibration protocol (`docs/calibration/backlog-5c3/`). Every change is
listed in the policy's `changes`, with its old and new definitions.

| Finding | Rule | Expected impact |
|---|---|---|
| The peak alone made 50% of stable stationary synthetic queues unstable | The peak reported, never gated; recovery reported (§4) | Temporary spikes never fail a seed |
| The mean at max(10%, 2 studies) failed 15–59% of stable queues; a stable queue's level rises faster than its workload | The level check keeps the workload-adjusted mean, with max(150%, 2 studies) | Stable seasonal and bursty queues pass; growth without bound fails |
| Block means of late work failed stable bursty queues | Overdue, stale and exposure on block medians | Bursts of late work that clear pass; sustained late work fails |
| An inflated reference licenses a stale burden | An absolute stale cap of 3 in every block from the reference on | Reference inflation fails |

Holdout reading (§2): the holdout is a strict empirical acceptance test,
and the independent-breach operating characteristics are a lower bound
on false rejection, not a guarantee. The thresholds and the allowance
are unchanged.

Adversarial tests: `TestBacklogStability`. Fail-before-fix evidence:
16 targeted mutations, every one caught (`docs/PROGRESS.md`).

## 12a. Changes from v0.2-eval-2 (Slice 5C.2)

The owner's decisions on the Slice 5C.1 development findings
(`docs/calibration/strategy-development/README.md`, *Policy
limitations*). Every change is listed in the policy's `changes`.

| Finding (development seeds) | Rule | Expected impact |
|---|---|---|
| Funding failed every plan, no action included, from day 1. The 14-day forward wages (about $53k) exceeded the cash headroom of a practice paying every bill on time. | Dated settlement test (§5); the old coverage is reported only | Plans that pay every obligation when due pass. Deferral, negative cash, truncated observation and unfunded committed liabilities still fail. |
| The backlog reference block falls in the December demand low; no action was unstable on 10 of 10 seeds | Workload-adjusted mean, peak, overdue and exposure; absolute stale and persistent (§4) | Proportional seasonal growth passes. Growth beyond the workload, stale and persistent work, spikes and late-tail deterioration still fail. |
| Soon breaches cluster by week (design effects 1.7–5.9) | Cluster-aware reporting and operating characteristics (§2); thresholds and allowance unchanged | No verdict changes by itself. Pass probabilities are stated without assuming independent patients. |
| Every reorganization carries $10–24k of transition cost before day 42 | CONDITIONAL GO, transition not yet repaid (§10) | Solvent, eligible reorganizations whose baseline passes service are CONDITIONAL GO instead of NO GO. Nothing becomes GO. |

Adversarial tests: `TestDatedFunding`, `TestSeasonalBacklog`,
`TestTransitionClassification`, `TestClusterStatistics`. Fail-before-fix
evidence is targeted mutations of each rule (`docs/PROGRESS.md`).

## 12b. Changes in v0.2-eval-2 (from v0.2-eval-1)

| Finding | Rule | Expected impact on eligibility |
|---|---|---|
| H5 | Joint seed reliability over all per-seed service gates | Stricter: plans that fail different gates on different seeds no longer pass |
| H1 | Urgent per seed over the gated period; pooled per window kept | Fewer false rejections: 0.98 → 0.03 at 8% urgent, under independence |
| H2 | One common horizon and observation end; unequal ones invalid | A late-effect candidate's baseline is gated over the same tail |
| H3 | Due cohorts by wanted time, outcomes at the end of observation; inherited stratum | Leaving future-due, difficult or inherited orders unresolved fails |
| H4 | Every later block through the covering one: mean, peak, overdue, stale, exposure; no persistent study | Stricter; may lengthen observation to the covering block |
| M1 | Exact comparisons in basis points | Values exactly at a tolerance pass |
| M2 | Per-seed modality, pooled intersections, rare = insufficient, dropped = fail | Rare strata are reported, never passed silently |
| H7 | GO needs absolute profit; four classifications; service purchase only for a baseline failing service | Loss-making improvements are at most CONDITIONAL GO |
| M3 | Forward employee commitment in the bridge; funding gap at every capture and sample, hard | Stricter |
| M4, M5 | Complete-record integrity; displacements by event time | Incomplete records are INVALID |
| Low | Seed-fraction labels, paired worst seeds, worst profit seed, defined-only means, dispersion index | None |

The adversarial fixtures are in `internal/eval/adversarial_test.go`. Each
fix was shown necessary by reverting it alone: 27 targeted mutations,
every one caught.

## 13. Deferred measures

These design measures are **not measured** by v0.2-eval-4. They are
listed in the policy's `deferredMeasures`, so that nothing implies they
are:

* payer and referral-source strata;
* workforce obligations: contracted, rostered and worked hours,
  training, overrun, premium hours, break entitlement and rest deficits;
* sustainable coverage capacity-hours;
* wanted-to-booked and wanted-to-service distributions for new demand
  (booking lead, and inherited orders' wait past wanted, are reported);
* contribution by modality, and the guarantee provision shown apart;
* coverage-gap exposure and outcome tails by reader kind;
* every baseline-relative `DELTA_*` tolerance of §13.3. None is
  calibrated or gated [owner: informational unless promoted before
  selection].

Every metric used in eligibility or in a substantive claim is measured.

## Open decisions

Decided by the owner on 2026-10-17 and frozen here:

* joint reliability;
* the urgent planning target and the ≤ 5% aim;
* common horizons;
* old and inherited work;
* absolute profit for GO;
* service purchase;
* employee commitments and the full-horizon funding test;
* rare strata;
* one confirmatory strategy;
* informational metrics.

Decided by the owner on 2026-10-18 and frozen in v0.2-eval-3: the dated
funding test, the workload-adjusted backlog, cluster-aware reporting and
the transition classification (§12a).

Decided by the owner on 2026-10-18 and frozen in v0.2-eval-4: the peak
informational and the backlog safeguards kept; the thresholds, the joint
rule and its allowance kept, with the holdout read as a strict empirical
acceptance test (§2). The backlog tolerances are the protocol's
selection (§4).

Still open:

1. **Dependence trade-off** (§2). The frozen design meets ≤ 5% at the
   planning rates only if breaches are close to independent, and the
   development diagnostics show they are not (soon most of all). The
   options are:
   * a longer fixed observation;
   * a larger seed allowance;
   * an evidence-based per-seed rule.

   Each needs a new policy version before unsealing.
2. **The [proposed] numbers**:
   * the stale (1,440 min) and persistent (14 days) limits;
   * the exposure floor (2,880 study-minutes);
   * the blocking strata kinds;
   * the 14-day employee commitment (since v0.2-eval-3 a reported
     indicator only).
3. **Development-seed verdicts** (10 seeds) have 6.9% false rejection at
   8% urgent even under independence. They are exploratory, not
   selection evidence on their own.
4. **Before unsealing**: one independent review of the whole corrected
   framework (v0.2-eval-4 and the canonical runner, built in Slice
   5C.0b), then the frozen strategy set and the owner's committed unseal
   record.
   * **Reviewed 2026-10-18 at 7707dbd: NO GO.** See
     `../audits/slice-5c3-eval4-framework-audit.md`.
   * Remediation is in three slices: 5C.4a, exposure containment and
     the clinical policy specification; 5C.4b, the clinical evaluator
     and its calibration; 5C.4c, holdout integrity.
5. **Not modelled**: remote read quality. Remote and local reports are
   assumed equivalent beyond turnaround, cost and coverage.
6. **Backlog trade-off** (§4). The selected rule meets the protocol's
   targets. What it does not detect is explicit:
   * a step change that leaves the queue stable at a higher level;
   * slow erosion before it produces growth or late work (46%);
   * a seasonal queue already at utilization 0.92 is rejected at 40
     seeds with probability 0.56.

   Whether that balance is right is the owner's to confirm before the
   independent review.
7. **Clinical acceptance** (audit H1, H2).
   * **Decided by the owner on 2026-10-18** (recorded in
     `proposals/v5-clinical-acceptance.md`):
     * hard severity limits by priority;
     * a gated wanted-to-acquisition access measure;
     * one shared four-seed clinical allowance;
     * zero tolerance for absolute age-limit violations;
     * the 92% utilization case as an out-of-calibration stress domain.
   * **Still open:** the numerical limits, which that proposal lists as
     owner decisions.
7. **Dependence and the planning target** (§2). Under the development
   range of clustering, the service design rejects a plan at the
   planning rates with probability 0.96. In v0.2-eval-4 the planning
   rates are no longer a guarantee of passage: a plan needs margin below
   them (owner, 5C.3). Option 1 remains available only through a new
   policy version.
