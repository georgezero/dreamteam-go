# Slice 5C.3: backlog-gate operating characteristics

Policy v0.2-eval-4 (`docs/evaluation/policy-v0.2-eval-4.json`, rules in
`docs/evaluation/README.md` §4) recalibrates the backlog gate only.

* **How it was chosen.** The owner decided that the peak is reported
  only and that the sustained safeguards stay hard. The tolerances and
  statistics were then selected by a prespecified protocol on synthetic
  queues.
* **What is unchanged.** The service thresholds, the joint seed rule,
  funding, solvency and every other rule.
* **The holdout.** It stays sealed. No reference seed was run, and no
  primary strategy is chosen.

| File | What it is |
|---|---|
| `protocol.md` | The calibration protocol: owner decisions, scenarios, targets T1 and T2, the 720-candidate family and the selection rule. It also records the exploration that shaped the family, and one correction made after the run. |
| `../../evaluation/oc-v0.2-eval-4.json` (`backlog`) | The confirmatory run: 1000 seeds per scenario on fresh streams; every candidate's unstable seeds per scenario; v0.2-eval-3 beside it, with its failures attributed by check; the selection. |
| `reevaluation.md`, `compare_versions.py` | Every committed strategy under v0.2-eval-3 (`p5-eval3`, unchanged) beside v0.2-eval-4 (`p6-eval4`), and D2 against E3. |
| `../strategy-development/results/p6-eval4/` | The v0.2-eval-4 development run: `run.json` (record, report and trajectory digests) and `report.md`. The detailed report is regenerated into `detail/`, which is not committed. |

Reproduce:

    DT_OC=1 go test ./internal/eval -run 'TestOperatingCharacteristics$' -v -timeout 240m
    DT_OC_BACKLOG=1 go test ./internal/eval -run 'TestOperatingCharacteristics$' -v -timeout 120m   # backlog part only, byte for byte (about 40 min on a Pi 4)
    DT_STRAT_SET=p6-eval4 \
      DT_STRAT_PAIRS=E3-urgent-weeknights-alloc65:D2-full-reading-alloc65,E2-urgent-tele-alloc65:E3-urgent-weeknights-alloc65,D2-full-reading-alloc65:D-full-reading,D-full-reading:C-full-reorganization \
      go test -tags calibration ./internal/sim -run 'StrategyDevelopment$' -v -timeout 600m
    python3 docs/calibration/strategy-development/report.py p6-eval4
    python3 docs/calibration/backlog-5c3/compare_versions.py > docs/calibration/backlog-5c3/reevaluation.md

## Findings

**1. Why v0.2-eval-3 fails stable queues.** On the 5C.2 streams it
reproduces exactly: 63.7% of stationary and 87.7% of seasonal stable
seeds are unstable. Broken down by the checks that failed:

| | Peak alone | Peak and mean | Mean alone | Any other check |
|---|---|---|---|---|
| Stationary | 50.3% | 12.7% | 0.7% | 0 |
| Seasonal | 27.0% | 57.0% | 3.7% | 0 |

* With bursty days, the overdue and exposure checks on block means also
  failed 4–12% of seeds, mostly in intermediate blocks.
* The mean's 2-study floor, not its 10% fraction, decided most
  stationary failures.
* Seasonally, a stable queue's level rises faster than its workload.
  Utilization goes from 0.79 (December reference) to 0.89 (January),
  and ρ/(1−ρ) rises 2.1 times for a 1.1-times workload.

**2. The selected rule** (the protocol's selection among the 84 of 720
candidates meeting both targets):

* **level:** the workload-adjusted block mean, within max(150%, 2
  studies) of the scaled reference;
* **overdue, stale, exposure:** block medians, at v0.2-eval-3's
  tolerances;
* **stale cap:** 3 studies, in every block from the reference on;
* **age:** no study more than 14 days past due (unchanged);
* **reported only:** the peak, recovery time and v0.2-eval-3's mean.

On the selection itself:

* **Ties.** Candidates with floor 5 or other late-work fractions tie on
  ladder detection. The tie-break order picks the smallest change from
  v0.2-eval-3.
* **Without a cap.** No candidate without a cap meets T2: reference
  inflation (I) is then detected on 84–88% of seeds by those that meet T1.

**3. Operating characteristics** (unstable seeds of 1000):

| | A | B | C | Cs | D | E | F | G | H | I |
|---|---|---|---|---|---|---|---|---|---|---|
| v0.2-eval-3 | 60.5% | 88.9% | 69.5% | 92.2% | 100% | 100% | 100% | 100% | 100% | 88.7% |
| v0.2-eval-4 | 0.0% | 0.1% | 0.5% | 3.8% | 100% | 100% | 100% | 100% | 98.5% | 97.6% |

* The worst stable rejection at 40 seeds is 0.017; every deterioration
  scenario is rejected with probability 1.00.
* Ladder and sensitivity scenarios:

  | Scenario | Unstable seeds | Rejection at 40 seeds |
  |---|---|---|
  | Capacity 5% lower | 0.2% | |
  | Capacity 10% lower | 22.8% | |
  | Capacity 15% lower | 96.6% | |
  | Slow erosion | 45.9% | |
  | Stationary at utilization 0.92 | 0.2% | |
  | Seasonal at utilization 0.92 | 12.4% | 0.56 |
  | Day CV 0.30 | 36.7% | |

**4. Development seeds 901–910** (exploratory; strategy files, records
and horizon unchanged):

| Plan | Backlog-stable seeds, v0.2-eval-3 → v0.2-eval-4 | v0.2-eval-4 verdict |
|---|---|---|
| D2 | 2 → 10 | CONDITIONAL GO, transition not yet repaid, restoring service |
| E3 | 1 → 10 | CONDITIONAL GO, service purchase |
| E2 | 1 → 10 | CONDITIONAL GO, service purchase |
| A (no action) | 1 → 7 | backlog gate fails: 3 seeds fail the overdue median |
| F, G, H | | NO GO on solvency, as before |
| B, C, D, E | | NO GO on service, as before |

Service, funding and solvency are identical in both versions.

**5. D2 against E3** (development seeds; T90 Δ against no action):

| | Urgent, gated | Worst seed urgent | Joint failing seeds | Backlog stable | T90 Δ, $k |
|---|---|---|---|---|---|
| D2 | 9.0% | 11.6% | 0 | 10 | −18.8 [−21.2, −16.3] |
| E3 | 2.8% | 3.7% | 0 | 10 | −23.2 [−25.7, −20.7] |

E3 costs about $4.4k more per 90 days.

* **D2's urgent margin is thin.** Its worst seed sits 0.4 points under
  the 12% gate. The 5C.2 week-block bootstrap put its 40-seed pass
  probability at about 0.33 (exploratory).
* **E3's margin is wide.** Its worst seed is 8 points under the gate.

## Limitations

* **Synthetic queue.** The calibration rests on a simple hourly queue
  shaped by the scenario. Its scale was checked against the development
  runs (block means 9–18 studies), its burstiness against the realized
  weekday dispersion, and nothing else.
* **The exploration shaped the family.**
  * The family was fixed after four exploratory sweeps; the targets were
    fixed before them.
  * The confirmatory run used fresh streams.
  * The family was extended twice, to block medians and to wide level
    tolerances, after an exploratory sweep showed the original family
    could not meet T1.
* **Not detected** (README §4, open decision 6):
  * step changes that leave a queue stable at a higher level;
  * slow erosion before the horizon;
  * a seasonal queue at utilization 0.92, which is rejected at 40 seeds
    with probability 0.56.
