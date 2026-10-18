# Slice 5C.3: backlog-gate calibration protocol

This protocol was fixed before the confirmatory operating-characteristic
run (`docs/evaluation/oc-v0.2-eval-4.json`, `backlog`), which uses
random streams no exploratory run used. Synthetic queues only: no
development or strategy result enters any target, scenario or selection
step.

## Exploration before it

Four exploratory sweeps, 200–300 seeds each, on their own streams,
shaped the candidate family. Their results are not the evidence; the
confirmatory run is.

1. **Reproduction and first sweep.**
   * The 5C.2 result reproduces exactly under v0.2-eval-3: 191 and 263
     of 300 stable seeds unstable.
   * Only the peak and the mean ever fail a stable queue.
   * The mean's 2-study floor, not its 10% fraction, decides most of
     its failures.
2. **Realistic burstiness.**
   * The development runs' realized weekday acquisitions vary less than
     Poisson arrivals would.
   * With a day factor of CV 0.15, no candidate whose overdue, stale
     and exposure checks use block means meets T1 (10–20% of bursty
     seeds unstable). Occasional multi-day excursions dominate a 28-day
     mean of late work.
3. **Medians of the daily samples.**
   * Medians of overdue, stale and exposure never failed a stable queue.
   * They detect late-tail deterioration less often, because late work
     appears in less than half of the covering block's days.
4. **Wider level tolerances.**
   * A stable queue's level rises faster than its workload as
     utilization rises. The scenario's December reference (season 0.93)
     and January block (1.05) at utilization 0.85 put ρ/(1−ρ) 2.1 times
     apart for a workload 1.1 times apart.
   * A level tolerance near +100–200% beyond the workload-scaled
     reference is what such a swing needs.
   * Late-tail deterioration is then detected mainly by the level check.
     The block mean detects it more often than the block median.

The family below was fixed after these sweeps, and the targets were
fixed before the first.

## Owner decisions taken as given

These are the owner's 5C.3 decisions:

* The individual 28-day peak becomes informational. It is still measured
  and reported.
* Time to recover after a spike is reported only.
* These stay hard:
  * the workload-adjusted 28-day mean;
  * stale studies;
  * maximum overdue age (no study more than 14 days past due);
  * overdue count and exposure;
  * every later block, intermediate and covering.
* Rota alignment, the reference block, the covering block and the common
  observation horizon are unchanged.
* Stale studies and the overdue-age limit are never workload-normalized.

## Scenarios

The queue is the Slice 5C.2 queue (`internal/eval/oc_queue_test.go`):
* 52 studies a day, with the scenario's weekday, hourly and monthly
  season profiles;
* reading Mon–Fri 07:30–20:00 and Sat 10:00–14:00 at a utilization of
  0.85 of the season-1 demand;
* reading by priority, first in first out;
* 181 days from Tue 2026-10-13.

Its block means (about 9 studies) are of the same order as the
development runs' (9–18). That was checked for scale only.

### Stable: must pass (false rejection)

| Id | Queue |
|---|---|
| A | Stationary demand, utilization 0.85 |
| B | Scenario seasonality, utilization 0.85 |
| C | Stationary demand, bursty days: a gamma day factor on arrivals with CV 0.15 |
| Cs | Scenario seasonality, bursty days (CV 0.15) |

**Why CV 0.15.** In the no-action development runs, realized weekday
acquisitions vary with a CV of 0.04–0.09, below the Poisson CV of about
0.12. A day factor of 0.15 on top of Poisson arrivals gives a total of
about 0.19, two to four times the realized day-to-day variation.

### Deteriorating: must fail (detection)

| Id | Departure from B |
|---|---|
| D | Sustained capacity deterioration: reading capacity 20% lower from day 100. Utilization is 1.06 at season 1, so the queue cannot clear |
| E | Persistent inherited overdue work: 10 routine studies acquired before the takeover, unread until day 120 |
| F | Rising age without count growth: from day 100, the first routine arrival of each weekday is held back 12 days before it joins the queue. About 8 old studies, never 14 days past due |
| G | Intermediate deterioration, then recovery: capacity 25% lower on days 90–117, restored after |
| H | Late-tail deterioration: capacity 20% lower from day 140, inside the covering block only |
| I | Reference inflation: capacity 30% lower in the reference block (days 55–82), restored, then 20% lower from day 111 |

### Sensitivity: reported, not targets

* Stationary queues at utilizations 0.75 and 0.92.
* A seasonal queue at utilization 0.92.
* A seasonal bursty queue with a day CV of 0.30.
* Slow erosion: capacity 0.2% lower each day from day 83, reaching
  utilization 1.0 by about day 158. It was added before the confirmatory
  run started.
* A capacity-loss ladder: 5%, 10% and 15% lower from day 100, seasonal.
  * At 5% and 10% the queue is still stable, at utilization 0.89 and
    0.94 (0.94 and 0.99 in January). It runs at a new, higher level.
    Whether such a step change must fail is a judgement, not a target.
  * The ladder measures sensitivity.

## Targets

**T1, false rejection.** For every stable scenario, the probability that
more than floor(10% × 40) = 4 of 40 holdout seeds are unstable is at
most 0.05. The criterion is the rejection probability itself, which the
code computes exactly. Equivalently, the per-seed unstable fraction is
at most 5.06%.

*Correction after the run.* This line first restated the limit as
"about 3.4%", a hand estimate that was wrong. The selected rule's worst
stable scenario (Cs, 3.8%) is within the criterion, at a rejection of
0.017, but it is above that misstated figure.

**T2, detection.** For every deteriorating scenario, the per-seed
unstable fraction is at least 0.90. Rejection at 40 seeds and at 10
seeds is then above 0.999.

Both are judged from 1000 simulated seeds per scenario. The exact
binomial (Clopper–Pearson) 95% bounds are reported beside each estimate.

## Candidate family

**Fixed structure (the owner's).**
* Hard checks:
  * the workload-adjusted level;
  * the workload-adjusted overdue count and exposure;
  * the absolute stale count;
  * the 14-day overdue-age limit;
  * every later block, intermediate and covering.
* Informational: the peak, recovery time, and the v0.2-eval-3 mean
  check (10% / 2 studies).

**Parameters.**

| Parameter | Values |
|---|---|
| Level statistic (daily backlog over the block) | mean, median |
| Level tolerance fraction (beyond the workload-scaled reference) | 0.10, 0.25, 0.50, 1.00, 1.25, 1.50, 1.75, 2.00, 2.50, 3.00 |
| Level tolerance floor | 2, 5 studies |
| Late-work statistic (overdue, stale, exposure, stale cap) | mean, median |
| Late-work tolerance fraction (floors unchanged: 2 studies, 2880 study-minutes) | 0.10, 0.25, 0.50 |
| Stale cap: no block from the reference on may exceed it, whatever the reference's | none, 2, 3 studies |

That is 720 candidates. v0.2-eval-3, and v0.2-eval-3 without the peak
check, are reported beside them.

**Selection.**
1. Keep the candidates meeting T1 and T2.
2. Among them, choose the highest mean detection on the 5% and 10%
   capacity-loss rungs (the most sensitive).
3. Break ties by the smallest departure from v0.2-eval-3, in this order:
   * the mean as the level statistic;
   * the mean as the late-work statistic;
   * the smaller level fraction;
   * the smaller level floor;
   * the smaller late-work fraction;
   * no cap before a cap, then the larger cap.

**Outcome.**
* If no candidate meets T1 and T2, no policy is frozen. v0.2-eval-3
  stays active, and the trade-off is reported.
* If one does, it becomes v0.2-eval-4, and the operating-characteristic
  test checks that the frozen policy is the selected candidate.
