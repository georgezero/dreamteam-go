# Slice 5C.1: development-seed strategy comparison

**Question.** Which financially feasible strategies meet the frozen
clinical service and access gates of v0.2-eval-2, using the development
seeds 901–910 only? This is strategy development, not a confirmatory
evaluation. The 40-seed holdout stays sealed. The reference seeds
1001–1020 are not used for anything here.

**Answer: no strategy passes the frozen development rules, and none
can.** Two hard gates fail on every seed for every plan, the no-action
baseline included. Neither failure is something a strategy can fix:

* **Funding.** The obligation bridge counts payables, accrued payroll
  and the 14-day forward employee commitment. It exceeds cash at some
  daily sample on all 10 seeds, from day 1, for every plan.
  * No action alone is in gap on 76% of its samples, by up to $40.9k.
  * Its forward commitment is $52.9k on its own, larger than that worst
    gap. The practice's $350k of receivables never counts.
* **Backlog.** The reference block (days 55–82, Dec 7 – Jan 3) falls in
  the scenario's seasonal demand low:
  * the season factor is 0.93 in December, 1.05 in January, 1.00 in
    February and March;
  * the later blocks get 3–4% more acquisitions a day;
  * no action's mean backlog then "grows" from 15.6 to 17.9, 17.2 and
    19.3 studies, and it is unstable on 10 of 10 seeds;
  * plans that read faster fail mostly on 28-day peaks.

  Only F, G and H, which read almost everything remotely at night, are
  stable on most seeds.

Within those two failures, the development results do separate the
strategies. **Three plans pass every other hard gate on all 10 seeds:**
the joint service rule, the pooled rates, case selection and solvency.
All three come from refinement:

* **D2:** D's staffing at a 65/55% allocation, with no contract;
* **E3:** D2 plus urgent-only weeknight teleradiology;
* **E2:** the same contract on weeknights and Saturdays.

Each loses money at T90 against no action: −$18.8k, −$23.2k and
−$25.2k. So each would at best be a **service purchase**: no action
fails service, and these restore it.

**Recommendation.** No primary strategy under v0.2-eval-2 as frozen.
The funding and backlog rules are owner decisions (*Policy
limitations*). If they are revised so that the baseline-wide failures
no longer bind:

* the prespecified ranking picks **D2**, which has the highest T90
  increment. D2 − E3 is +$4.4k [4.3, 4.5] at T90, so the simplicity
  tie-break does not apply;
* **E3** is the robust alternative:
  * D2's urgent rate is 9.0% pooled and up to 11.6% on a seed, against
    the 12% gate. Under independence it fails the 40-seed joint rule
    about 9% of the time; clustering would raise that;
  * E3's is 2.8% pooled and 3.7% at worst, with essentially no risk.

Choosing between D2 and E3 is the owner's call. The difference is about
$4.4k of contract cost per 90 days against the urgent margin.

## Results

All tables are generated from `results/<set>/report.json`
(`report.md` in each). A plan that appears in more than one set has
identical records in each (same digests), so its figures are the same
throughout.

### Development-seed comparison (seeds 901–910, H = day 150)

Service rates are pooled over the gated period: urgent per seed over
(t0, H], soon and routine per seed in T90 and the tail. Money is $k, the
mean over seeds; Δ is paired against A.

| Plan | Failing seeds (joint, limit 1) | Breach U / S / R % | Backlog stable | Lowest cash (insolvent seeds) | Funding-gap seeds | Profit T90 / T180 | Δ T90 / Δ T180 | Verdict |
|---|---|---|---|---|---|---|---|---|
| A no action | 10 | 10.2 / 35.6 / 7.1 | 0/10 | 36.2 | 10 | 3.7 / 5.9 | — | baseline |
| B partial | 10 | 10.5 / 36.0 / 7.1 | 1/10 | 44.4 | 10 | 1.7 / 18.9 | −2.0 / +13.0 | NO GO |
| C full | 10 | 22.7 / 67.1 / 17.5 | 3/10 | 26.8 | 10 | −4.8 / 23.3 | −8.5 / +17.4 | NO GO |
| D full + reading | 8 | 10.9 / 28.9 / 4.7 | 1/10 | 24.3 | 10 | −8.8 / 17.9 | −12.5 / +12.1 | NO GO |
| E D + urgent tele | 4 | 3.4 / 27.4 / 4.5 | 1/10 | 14.8 | 10 | −15.3 / 3.7 | −19.0 / −2.2 | NO GO |
| F D + broad tele | **0** | 2.8 / 2.7 / 0.6 | 10/10 | −38.9 (10) | 10 | −41.8 / −54.3 | −45.5 / −60.2 | NO GO |
| G B + broad tele | **0** | 2.7 / 3.0 / 0.6 | 10/10 | −23.4 (7) | 10 | −31.5 / −56.5 | −35.2 / −62.3 | NO GO |
| *R1* E2 (E at 65/55) | **0** | 1.9 / 20.9 / 3.9 | 1/10 | 2.1 | 10 | −21.5 / −9.4 | −25.2 / −15.3 | NO GO |
| *R2* H (D + weeknight broad) | **0** | 3.7 / 5.7 / 3.5 | 8/10 | −24.2 (8) | 10 | −35.7 / −40.3 | −39.4 / −46.2 | NO GO |
| *R3* E3 (E2, weeknights only) | **0** | 2.8 / 20.9 / 3.9 | 1/10 | 6.2 | 10 | −19.5 / −4.9 | −23.2 / −10.8 | NO GO |
| *R4* D2 (E2 without contract) | **0** | 9.0 / 21.5 / 3.9 | 2/10 | 15.2 | 10 | −15.1 / 5.1 | −18.8 / −0.8 | NO GO |

**Case selection passes for every candidate.** No stratum fails and no
blocking stratum lacks evidence:

* served fractions are 72.3% (A, B, G) and 74.9% (C–F and the
  refinements) of T90 due demand;
* lost referrals fall from 13.9% to 12.3% with the full reorganization;
* inherited orders are served alike (10,808–10,810 of 12,019).

The service gains are therefore not bought by selecting cases: the
reorganized plans serve more demand, and the reading changes alter no
booking. Every `teleOnlyOf` pair booked identically on every seed.

### Per-seed service eligibility

Each failing seed's binding gate (from each set's `report.md`):

| Seed | A | B | C | D | E | E2, E3, D2, F, G, H |
|---|---|---|---|---|---|---|
| 901 | S T90, R T90, R tail | S, R | S, R, U | S T90 40.3% | S T90 37.1% | pass |
| 902 | S, R, U | S, R | S, R, U | S T90 32.6% | pass | pass |
| 903 | S, R | S, R | S, R, U | U 14.0% | pass | pass |
| 904 | S, R | S, R, U | S, R, U | U 12.2% | pass | pass |
| 905 | S, R | S, R | S, R, U | pass | pass | pass |
| 906 | S, R | S, R | S, R | S T90 35.0% | S T90 34.3% | pass |
| 907 | S, R | S, R, U | S, R, U | S tail 32.3% | pass | pass |
| 908 | S, R | S, R | S, R, U | S tail 41.6% | S tail 40.7% | pass |
| 909 | S, R | S, R | S, R, U | S T90 33.1%, U 12.3% | S T90 32.8% | pass |
| 910 | S, R, U | S, R, U | S, R, U | pass | pass | pass |

**D's and E's soon failures are clustered.**

* *Cause:* a day of 80–90 acquisitions builds a daytime backlog of
  30–45 studies, which takes days to clear.
* *Effect:* single weeks breach almost every soon study, for example 48
  of 49 in week 6 on seed 901.
* *Measured:* the soon dispersion index is 2–9 for D and E, against
  about 1 for independent breaches. Urgent's is 0.4–1.6.
* *What fixes it:* one more allocation step (R1, R4). The urgent
  contract does not.

### Teleradiology: marginal value (paired, per seed, 90 days)

| Contract (against its parent) | Remote reports T90 | Retainer + fees T90 | Cost per remote report | Δ reports | Δ revenue | Δ breach T90 U / S / R pts | Δ profit T90 |
|---|---|---|---|---|---|---|---|
| E − D: urgent, weeknights + Sat | 26 | $5.8k + $0.7k | $252 | 0 | $0.0k | −6.3 / −1.1 / −0.2 | −$6.5k |
| E3 − D2: urgent, weeknights | 24 | $3.8k + $0.6k | $187 | 0 | $0.0k | −5.4 / −0.5 / −0.0 | −$4.4k |
| F − D: all, every window | 736 | $7.8k + $26.2k | $46 | +10 | +$1.1k | −6.8 / −25.6 / −4.1 | −$33.0k |
| H − D: all, weeknights | 611 | $3.8k + $23.1k | $44 | +0 | +$0.2k | −5.6 / −22.3 / −1.3 | −$26.9k |
| G − B: all, every window | 788 | $7.8k + $28.3k | $46 | +23 | +$3.0k | −6.3 / −30.8 / −6.1 | −$33.2k |

The three effects a contract could have, kept apart:

* **Earlier reports.** This is the whole of the benefit:
  * an urgent-only contract takes urgent from about 10% to 2–3%. Remote
    urgent TAT is 23–26 min at p50 and 57–68 min at p90;
  * a broad contract takes soon from 29% to 3–6%.
* **Additional reports.** At most 23 per seed per 90 days, about 0.3%
  of reports. They are studies that would still have been unread at the
  endpoint.
* **Additional recognized revenue.** At most $3.0k per seed per 90 days,
  against $34k of cost.

Remote reports otherwise substitute for local reads (F: 7,364 remote,
local 47,442 → 40,183). The radiologists' salary is unchanged, so every
fee is a net cost.

87% of F's fees are on weeknights, and most claim routine studies that
the morning's local readers would have finished in time. All-priority
coverage is insolvent on 7–10 of 10 seeds. Urgent-only coverage costs
mostly its retainer ($1,500–2,300 a month) and is solvent.

### Financial and cash comparison

From each set's `report.md`, "Cash, obligations and payback":

* **Transition.** Every reorganization loses money through T90, by
  $8.5–12.5k for C and D (−$2.0k for B). No plan's T90 accrual payback
  is reached on any seed.
* **Steady state (the tail, days 91–150).**
  * D: +$17.8k, against D2 +$13.9k and E3 +$10.3k;
  * B: +$10.1k;
  * broad teleradiology turns it negative: F −$7.6k, G −$17.5k.
* **T180.**
  * Paid back on accrual by day 180: B on 10 of 10 seeds (day 101), C
    on 10 (day 120), D on 10 (day 134).
  * E and D2 pay back by T180 on 3 of 10 seeds; E2 and E3 on none.
    Their T180 increments run from −$0.8k (D2) to −$15.3k (E2).
* **Cash.**
  * Lowest cash after any posting: no action $36.2k; D $24.3k; D2
    $15.2k; E3 $6.2k; E2 $2.1k.
  * F, G and H go below zero on 7–10 seeds.
  * Cash payback by T180: B on 10 of 10 seeds, C on 9, D on 1, none of
    the refinements.
* **Obligations.** At T90 the bridge holds $107–116k per seed (mean).
  About half is the forward employee commitment: $50.6–59.8k at
  the captures of seed 906, depending on the staffing. Contract
  minimums add at most $3.1k.

### Binding constraints

For every failed plan, the gates that bind (beyond funding and
backlog, which bind for all):

| Plan | Binding constraint |
|---|---|
| A | soon and routine on 9–10 seeds (breaches after the evening reading ends, and Saturday imaging); urgent on 2 |
| B | as A; it changes no service figure |
| C | reading capacity: every priority, every seed; the backlog grows to 58 by days 139–166 |
| D | soon on 6 seeds (clustered daytime bursts); urgent on 3 (evening acquisitions) |
| E | soon on 4 seeds (the same bursts; the contract fixes urgent only) |
| F, G, H | solvency: remote fees of $23–28k per 90 days drive cash below zero; T90 and T180 profit far below A |
| E2, E3, D2 | none beyond funding and backlog; absolute and incremental profitability negative at T90 (service purchase at best) |

### Phase 1: the historical plans under v0.2-eval-2

`results/p1-historical/reconciliation.md` (`reconcile.py`).

* **Same simulation.** Every archived daily sample of A–D on 901–910 is
  reproduced exactly: 10,800 samples across six archived files (engines
  0.2.0-dev.4 and dev.5), in profit, cash, final reports and backlog.
* **Same T90 measurement.** The archived T90 breach rates, the
  evaluation's legacy rule and v0.2-eval-2's due-time rule agree at T90
  to the study (A 9.7 / 35.9 / 7.2%; D 10.0 / 30.2 / 5.2%), and so do
  the window profits.
* **What changes is the rules:**

  | Historical rule (dynamic-staffing README) | v0.2-eval-2 | Effect on D |
  |---|---|---|
  | Gates on the mean across seeds, T90 and days 91–180 | Joint per-seed rule (a seed fails on any of five gates, at most 1 of 10) plus pooled rates | Mean gates met → 8 of 10 seeds fail |
  | Tail = days 91–180 | Tail = days 91–150 (H) | Urgent 11.0 → 11.8%, soon 30.1 → 27.1% |
  | Backlog: day-180 sample ≤ 1.1 × day-90 sample | Every 28-day block against the Dec 7 – Jan 3 reference: mean, peak, overdue, stale, exposure | Failed either way; now on 9 of 10 seeds |
  | No funding test | Obligations (with the 14-day employee commitment) ≤ cash at every sample | Fails on every seed |
  | CONDITIONAL GO: T180 increment positive with CI | GO needs a positive T90 interval and own profit | T90 increment −$12.5k |

  The historical CONDITIONAL GO for D becomes NO GO under v0.2-eval-2.
  None of the change comes from the simulation.

### Policy limitations found (for the owner; the policy is unchanged)

1. **The funding gate is infeasible for this practice.** It fails the
   baseline on every seed, from day 1.
   * The 14-day employee commitment ([proposed], `employeeCommitmentDays`)
     is about $53k.
   * The practice runs on about $50–220k of cash, with $350k of
     receivables that the bridge never counts.
   * No player action can pass it, so as frozen it makes every
     comparison NO GO.
2. **The backlog reference block is seasonal.** The last whole rota
   block ending by T90 is days 55–82, in the December demand low.
   * Every later block compares higher-demand months against it.
   * No action is unstable on 10 of 10 seeds. D2, E2 and E3 have the
     lowest backlogs of the local-reading plans (reference means of 7–9
     studies), no persistent study and a worst lateness of 39 hours.
     Yet they are stable on only 1–2 seeds, failing on +2-study means
     and 28-day peaks.
   * The 10% / 2-study tolerances and the 28-day peak were set from
     units, with no operating characteristics. On 10 development seeds
     the rule rejects the baseline every time.
3. **Soon breaches are clustered.** The dispersion index of the soon
   gates is 2–9 for D and E, the plans near the gate. The per-seed soon gate's
   operating characteristics assume independence. The owner's open
   urgent decision has a soon counterpart.
4. **T90 primary versus transitions.** Every reorganization carries
   $10–24k of transition cost before day 42, so no plan with staffing
   changes can have a positive T90 increment.
   * Under the frozen order, the best outcome open to any candidate here
     is CONDITIONAL GO, service purchase.
   * B, C and D are profitable by T180, but only after T90.
5. **Ten seeds.** At the joint limit of 1 of 10, a single seed decides
   the development verdict. These results identify mechanisms and
   credible strategies; they do not estimate 40-seed pass rates.

## Method

### What is fixed

Nothing in the model, the policy, the holdout manifest or the holdout
runner changed:

* simulation economics, demand, exam and reading durations;
* clinical thresholds;
* `docs/evaluation/policy-v0.2-eval-2.json`;
* `holdout-v0.2.json`;
* `TestHoldoutRun`.

Every strategy uses only supported player actions (`hire`, `terminate`,
`set_roster`, `set_reading_allocation`, `contract_teleradiology`),
applied through `World.Apply` at fixed practice-local instants. Every
seed gets the same actions, and no strategy has a rule of its own.

### Files and runner

* **Strategy files.** `strategies.py` writes `strategies/*.json`
  (`dreamteam.strategy/v1`) and the development sets `sets/*.json`
  (`dreamteam.strategyset/v1`, every strategy pinned by SHA-256).
  * These are the holdout contract's own formats, so a frozen set needs
    no conversion.
  * Plans A–D copy the frozen dynamic-staffing plans' actions
    unchanged.
* **Runner.** `internal/sim` `TestStrategyDevelopment` (build tag
  `calibration`) runs a set on the policy's development seeds.
  * It loads the set through `eval.Development`. That function decodes
    the set strictly, checks every pin, the baseline and `teleOnlyOf`,
    and refuses any seed that is not a development seed.
  * The protocol it returns is never authorized, so
    `NewHoldoutRecorder` refuses it.
  * It runs exactly as the canonical runner does: the same scenario,
    batch order, daily samples, captures, common horizon (computed
    before running, confirmed by every record) and evaluation. Only the
    recorder differs, the development one.
  * `TestDevelopmentRunnerMatchesCanonical` shows that both runners
    produce byte-identical records and the same horizon, on the
    synthetic protocol.
* **Outputs** (`results/<set>/`):
  * `run.json`: the set and policy digests, the engine, the seeds, the
    horizon, every record's digest and the report's digest;
  * `report.json`: the frozen policy's evaluation;
  * `trajectory.csv`: the daily samples.

  They are written once. A rerun must reproduce all three byte for
  byte, or it fails.
* **Tables.** `report.py` generates `report.md` from `report.json`, and
  `reconcile.py` generates Phase 1's reconciliation. `breach_timing.py`
  reads records written with `DT_STRAT_RECORDS`; the records themselves
  are not committed.

Reproduce (about 7 minutes per 40 runs on a Raspberry Pi 4). Each run's
outputs already exist, so a rerun checks them byte for byte:

    python3 docs/calibration/strategy-development/strategies.py
    DT_STRAT_SET=p1-historical go test -tags calibration ./internal/sim -run 'StrategyDevelopment$' -v -timeout 600m
    DT_STRAT_SET=p2-family DT_STRAT_PAIRS=D-full-reading:C-full-reorganization,F-broad-tele:E-urgent-tele,G-partial-broad-tele:F-broad-tele \
      go test -tags calibration ./internal/sim -run 'StrategyDevelopment$' -v -timeout 600m
    DT_STRAT_SET=p3-refine DT_STRAT_PAIRS=E2-urgent-tele-alloc65:E-urgent-tele,H-weeknight-tele:F-broad-tele,H-weeknight-tele:E2-urgent-tele-alloc65,E-urgent-tele:D-full-reading,F-broad-tele:D-full-reading \
      go test -tags calibration ./internal/sim -run 'StrategyDevelopment$' -v -timeout 600m
    DT_STRAT_SET=p4-refine DT_STRAT_PAIRS=E3-urgent-weeknights-alloc65:E2-urgent-tele-alloc65,D2-full-reading-alloc65:E2-urgent-tele-alloc65 \
      go test -tags calibration ./internal/sim -run 'StrategyDevelopment$' -v -timeout 600m
    python3 docs/calibration/strategy-development/report.py p1-historical p2-family p3-refine p4-refine
    python3 docs/calibration/strategy-development/reconcile.py

### Seeds, horizon, pairing

* **Seeds:** 901–910, the policy's development seeds. Every comparison is
  paired on the seed.
* **Common horizon:** H = day 150 for every set here. No strategy's
  effect or commitment reaches past day 120:
  * the latest staffing effect is Nov 24 (day 42);
  * contracts activate on Oct 27 with an earliest end of Nov 26.
* **Observation:** to day 181 (T180 + the 1,440-minute closure). The
  backlog blocks are days 55–82 (the reference), 83–110, 111–138 and
  139–166.
* **Endpoints:**
  * T30, feedback;
  * T90, primary;
  * D31–90, operating performance;
  * the tail (days 91–150), gated;
  * T180, validation.

### Phase 2: the strategy family (fixed before any teleradiology run)

| Plan | Actions | Purpose |
|---|---|---|
| **A-no-action** | none | Baseline |
| **B-partial** | the frozen B: MR evening converted to part-time | Low-cost financial improvement |
| **C-full-reorganization** | the frozen C: dedicated XR/CT day core, part-time evenings | Imaging capacity |
| **D-full-reading** | C plus the 62.5/52.5% reading allocation from Oct 27 | Internal reading capacity |
| **E-urgent-tele** | D plus an **urgent-only** contract, **weeknights + saturday** | Urgent turnaround |
| **F-broad-tele** | D plus an **all-priority** contract on **every window** | Whether more remote capacity is worth it |
| **G-partial-broad-tele** | B plus F's contract | A lower-cost staffing base with remote coverage |

**Why these contracts.** D's breaches on the development seeds
(`results/p1-historical/breach-timing.md`), by the windows that overlap
each breaching study's wait:

* **urgent:** 196 of 250 are reachable on weeknights and 22 on
  Saturday. They were acquired 17:00–20:00, after Dr. Avery's 16:00 end.
* **soon:** 2,321 of 2,933 are reachable on weeknights and 371 on
  Saturday.
* **routine:** 2,913 of 3,248 are Saturday imaging, reachable on
  Saturday and Sunday.

E puts urgent-only reading where the urgent breaches are. F is the
broadest contract the catalog offers.

G tests the user's caution not to assume D is the right base. B is the
cheapest staffing change with a positive steady state: it saves about
$5k a month and leaves service as in A, whose soon and routine
breaches are also evening and weekend ones. If remote reading can carry
those, B plus a contract might match D's service without the
reorganization's $23.7k transition or the allocation's $2.5k a month.

**Contract terms.** All three contracts share these terms (catalog
`standard-v1`; nothing invented):

| | E | F, G |
|---|---|---|
| Accepted | takeover, Tue 2026-10-13 06:00, in its own batch after the staffing batch | same |
| Activation | Tue 2026-10-27 (the earliest, 14 days' notice) | same |
| Windows | weeknights (Mon–Fri 18:00 to 08:00), saturday (14:00–24:00) | weeknights, saturday, sunday (00:00–24:00) |
| Scope | urgent (by original priority) | all priorities |
| Retainer | $1,500 + $800 = **$2,300 a month** | $1,500 + $800 + $800 = **$3,100 a month** |
| Per-read fee | XR $12, US $30, CT $45, MR $60; +20% claimed 22:00–08:00 | same |
| Minimum commitment | 30 days from activation, to Nov 26 ($2,300) | to Nov 26 ($3,100) |
| End | open-ended | open-ended |
| Cancellation plan | **none during the evaluation.** The contract runs through the observation. Cancelling needs 30 days' notice and takes effect no earlier than Nov 26. A cancellation would move the horizon, and the gated tail would then judge the plan without its contract. | same |

Remote reports earn the same revenue as local ones, once each. A
contract adds revenue only by reporting studies that would otherwise
still be unreported at an endpoint.

### Phase 9: the recommendation rule (fixed before Phase 2's results)

1. **Eligible:** the evaluation's verdict against A is not NO GO or
   INVALID. This means every hard gate passes on the 10 development
   seeds:
   * service (joint seed rule and pooled rates);
   * case selection;
   * backlog;
   * solvency;
   * funding.
2. **Among eligible plans:**
   * GO ranks above CONDITIONAL GO (relative improvement), which ranks
     above CONDITIONAL GO (service purchase);
   * within a class, the highest mean T90 incremental profit wins;
   * if the paired 95% interval between the top two includes zero, the
     simpler one wins: fewer actions and no contract, or the smaller
     contract minimum.
3. **No eligible plan:** no primary strategy is recommended. The plans
   that pass every hard gate the no-action baseline itself passes are
   listed apart, with the gates they fail, for the owner's decision.
   This is not a verdict.

The 10-seed verdicts are exploratory: at a true 8% urgent rate the
frozen design rejects 6.9% of the time on 10 seeds, even with
independent breaches (`docs/evaluation/README.md` §2).

### Phase 8: refinement rules

* At most **four** revisions, on development seeds only.
* Each revision has:
  * a stated hypothesis, written before its run;
  * an exact change to one action file;
  * a paired before/after comparison;
  * a reason it was kept or rejected.
* Every trial is kept in `sets/` and `results/`, whatever its outcome.

### Phase 8: refinements as run

Each hypothesis was written here before its run. The parents are rerun in
the same set, so every comparison is paired within one evaluation.

**R1, `E2-urgent-tele-alloc65`** (set `p3-refine`, against E).
* *Observation:* E's remaining failures are soon breaches on four seeds
  (901, 906, 908, 909).
  * They cluster in single weeks: for example 48 of 49 soon studies in
    week 6 on seed 901, and 62 of 62 in week 20 on seed 908.
  * A day of 80–90 acquisitions builds a daytime backlog of 30–45
    studies that takes several days to clear. The model has no
    absences.
* *Hypothesis:* one more 2.5-point allocation step (Avery 65.0%,
  Blake 55.0%, about $2.5k per 30 days) clears these bursts faster. The
  failing soon seeds then fall to at most one, with no new failure; the
  urgent contract keeps urgent where E has it.
* *Change:* the two `siteAllocationBP` values of E (6250 → 6500, 5250 →
  5500), nothing else.
* *Keep if:* the joint service rule passes, and no gate passing in E
  fails.

**R2, `H-weeknight-tele`** (set `p3-refine`, against F).
* *Observation:* 87% of F's remote claims and fees are on weeknights.
  * Sunday has no claims, because Saturday night clears the backlog.
  * D passes routine on every seed without weekend coverage.
* *Hypothesis:* F's contract on the weeknight window alone keeps F's
  service: soon and urgent pass on every seed, routine passes as in D.
  It saves the $1,600 a month of weekend retainers and Saturday's fees.
  The question is whether all-priority coverage can then be solvent.
* *Change:* the contract's windows, `["weeknights", "saturday",
  "sunday"]` → `["weeknights"]` (request ID `weeknight-tele`).
* *Keep if:* the joint service rule and solvency both pass.

**R3, `E3-urgent-weeknights-alloc65`** (set `p4-refine`, against E2;
written after R1 and R2's results).
* *Observation:* only 22 of D's 250 urgent breaches were reachable on
  Saturday alone. E2's Saturday retainer is $800 a month.
* *Hypothesis:* E2's urgent contract on the weeknight window alone keeps
  the urgent per-seed gate on every seed, for $2.4k less per 90 days.
* *Change:* the contract's windows, `["weeknights", "saturday"]` →
  `["weeknights"]`.
* *Keep if:* every hard gate that passes in E2 still passes.

**R4, `D2-full-reading-alloc65`** (set `p4-refine`, against E2).
* *Observation:* D's urgent breaches were acquired 17:00–20:00, after Dr.
  Avery's 16:00 end, while Dr. Blake reads until 20:00. A higher
  allocation lengthens Blake's site reading too.
* *Hypothesis:* the 65/55 allocation alone, with no contract, keeps
  urgent within its gate on at least 9 of 10 seeds and soon as in E2.
  That saves the whole contract ($2.3k a month plus fees).
* *Change:* E2 without its contract batch.
* *Keep if:* every hard gate that passes in E2 still passes.
