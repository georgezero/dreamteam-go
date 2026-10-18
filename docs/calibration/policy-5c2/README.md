# Slice 5C.2: policy feasibility diagnostics and re-evaluation

Policy v0.2-eval-3 (`docs/evaluation/policy-v0.2-eval-3.json`, rules in
`docs/evaluation/README.md`) corrects four v0.2-eval-2 rules that the
Slice 5C.1 development runs showed infeasible or statistically
misleading. Everything here uses the development seeds 901–910 only,
and is labelled exploratory where it is. The holdout stays sealed, the
reference seeds were not run, and no primary strategy is chosen.

| File | What it is |
|---|---|
| `clustering.md`, `cluster_diagnostics.py` | Day (weekday-adjusted) and week intra-cluster correlation of breaches per plan and priority (exploratory). It bounds the dependence range of the operating characteristics; no rule was chosen from it. |
| `bootstrap.md`, `block_bootstrap.py` | A week-block bootstrap estimate of each plan's chance of passing the 40-seed service rule without assuming independent patients (exploratory). |
| `reevaluation.md`, `compare_versions.py` | Every committed strategy under v0.2-eval-2 (the 5C.1 reports, unchanged) beside v0.2-eval-3 (set `p5-eval3`). |
| `../strategy-development/results/p5-eval3/` | The v0.2-eval-3 development run: `run.json` (record, report and trajectory digests) and `report.md`. The detailed report is regenerated into `detail/` (not committed). |

Reproduce:

    DT_STRAT_SET=p5-eval3 DT_STRAT_RECORDS=/tmp/rec \
      DT_STRAT_PAIRS=E3-urgent-weeknights-alloc65:D2-full-reading-alloc65,E2-urgent-tele-alloc65:E3-urgent-weeknights-alloc65,D2-full-reading-alloc65:D-full-reading,D-full-reading:C-full-reorganization \
      go test -tags calibration ./internal/sim -run 'StrategyDevelopment$' -v -timeout 600m
    python3 docs/calibration/strategy-development/report.py p5-eval3
    python3 docs/calibration/policy-5c2/compare_versions.py > docs/calibration/policy-5c2/reevaluation.md
    python3 docs/calibration/policy-5c2/cluster_diagnostics.py /tmp/rec/p5-eval3 PLAN...
    python3 docs/calibration/policy-5c2/block_bootstrap.py /tmp/rec/p5-eval3 1000 PLAN...
    DT_OC=1 go test ./internal/eval -run 'TestOperatingCharacteristics$' -v -timeout 120m

## Findings

**Funding.** v0.2-eval-2 counted 14 days of future wages as payable at
every instant. Split by kind, the end-of-observation bridge of the
no-action practice is:

| Kind | Amount |
|---|---|
| Cash | $116.8k |
| Accrued liabilities paid in the cycle (payables, payroll) | $87.5k |
| Committed liabilities | $0 |
| Future wages (the 14-day commitment) | $52.9k |
| Receivables (never counted) | $365.7k |

* The practice pays every bill on its date, and cash never goes below
  zero.
* Only the future wages pushed it into "gap", on 76% of daily samples.

Under the dated settlement test, every plan is funded, F, G and H
included: what fails them is solvency (negative cash), as before. The
commitment coverage is still reported. Every plan is in commitment gap
at some instant, and the plans with remote fees or lower cash are in
gap at almost every one.

**Backlog.** The scenario's demand rises from December (0.93) to January
(1.05). The later blocks' workload is 3–4% above the reference's in the
runs (ratio 1.03–1.04).

* Workload adjustment moved no action from 0 to 1 stable seed.
* D2, E2 and E3 stay unstable on 8–9 of 10 seeds, and most of them fail
  the peak check:
  * D2 fails on the peak alone on 4 seeds, E2 on 5, E3 on 6;
  * the workload-adjusted mean fails on 3–4 seeds;
  * none has an overdue, stale, exposure or persistent failure.
* The operating characteristics (`docs/evaluation/oc-v0.2-eval-3.json`,
  `backlog`) show why. The peak check alone fails 50% of a stable
  synthetic queue's seeds, and the frozen rule rejects such a plan's 40
  seeds with probability about 1.

The backlog gate is the one remaining binding rule, and its numbers are
an open owner decision.

**Service.**

* Day and week clustering is real: soon's week design effect is 1.7–5.9
  in the plans near the gate. Under that dependence the frozen design
  rejects plans at the planning rates with high probability.
* Under the run's own week structure, D2's urgent margin is thin:
  * a seed fails 13.7% of the time, 12.3 points of it on urgent;
  * so D2 passes the 40-seed joint rule in 33% of bootstrap
    replicates.
* E3 and E2 fail a seed 0.8% of the time and pass every replicate.

**Classification**, if the backlog gate passed. This is a counterfactual
(`report.md`), not a verdict:

* D2 is CONDITIONAL GO, a transition not yet repaid that restores
  service. Its days 31–90 increment is +$1.9k [−0.7, 4.6], its T90
  increment −$18.8k.
* E3 and E2 are CONDITIONAL GO service purchases. Their operating-window
  increments are −$1.6k and −$3.2k, and their T90 costs $23.2k and
  $25.2k.
* None is financial GO.
