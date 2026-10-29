# V0.7b plan: characterizing financial solvency

Plan version `v07b-plan/1`, written 2026-10-29. It is committed with the
solvency model, before any run on its seeds.

**Purpose.** V0.7b adds a credit line and a funding rule, in new scenario
versions ([V0.7_SOLVENCY_DESIGN.md](../../V0.7_SOLVENCY_DESIGN.md)):
* the practice borrows, within a fixed limit, at a deterministic cost;
* a discretionary commitment is accepted only if it fits in the funding
  headroom: cash plus undrawn credit, less current liabilities and
  reserved commitments.

This characterization measures what that does to the five practices under
the three deterministic baselines. **It is not a ranking.** There is no
composite score and no significance test, and no policy is tuned on the
results. No LLM agent is run.

Anything this plan does not settle is decided before the run it affects,
and committed as a dated amendment at the end. An amendment never changes
a seed, a policy, a scenario or the solvency model.

## 1. What is fixed

| | |
|---|---|
| Solvency model | [V0.7_SOLVENCY_DESIGN.md](../../V0.7_SOLVENCY_DESIGN.md) |
| Comparison (no solvency) | suite `dreamteam-dev` 2 ([file](../suite/dreamteam-dev-2.json)), unchanged: the V0.7a `-ot` scenarios |
| Solvency | suite `dreamteam-dev` 3 ([file](../suite/dreamteam-dev-3.json)), ten entries. Each `-ot` scenario comes with a $50,000 credit line at 10% a year (`-credit`) and with no credit (`-nocredit`). Opening cash and everything else are the `-ot` scenario's. |
| Fingerprints | `-credit`: financial pressure `586f3ef7…`, reporting backlog `1fc90da2…`, demand growth `28c16bf3…`, staffing shortage `5b49b4ca…`, stable `cfae9ca5…`; `-nocredit`: `ee8c7cc2…`, `6c9ee3db…`, `9144c8a4…`, `c500a61a…`, `74612d88…` |
| T90 | not applicable (version 1's sustainable scenario only) |
| Contract | `dreamteam.benchmark-contract/v1`, unchanged |
| Engine | 0.7.0-dev.2 (this commit) |
| Policies | `noop` 1, `greedy` 1, `conservative-v2` 2, unchanged since V0.6c |
| Seeds | 41501, 41502, 41503: fresh, the next block after V0.7a's 41401–41403. Not reference, development, reserved-stream or holdout seeds. The same seeds for every suite version. |
| Horizon and cadence | 90 days; decisions daily at 07:00 |
| Configurations | [characterize-v2.json](characterize-v2.json) (45 runs), [characterize-v3.json](characterize-v3.json) (90 runs): 135 in all |

**Failures.** A run that fails is reported with its status. An
infrastructure failure (the machine, not the simulator) is rerun once
into a new directory, and both are kept. Nothing is rerun to change a
result.

## 2. What is compared

Each version 3 run is paired with the version 2 run on the same practice,
seed and policy. The two `-credit` and `-nocredit` runs of a practice are
also compared with each other.
* **Cash trajectories.** Cash at the end of each day, the lowest cash and
  the cash at the end.
* **Credit utilization.** The balance drawn at the end of each day, and
  the most drawn. Cash only reaches the line when it would go below zero.
* **Borrowing costs.** Interest charged in the window.
* **Financial distress.** Days ending with cash below zero (with the line
  fully drawn), and distress lines in the log.
* **Refused actions.** Actions refused `insufficient_funds`, with the
  policy and date. Whether the accepted actions, and when they were
  decided, are identical to version 2's.
* **Operating profit** and its change from version 2.
* **Service outcomes.** TAT p90, breaches, exams completed and orders
  lost.
* **Within each version:** policy minus `noop` on the same entry and seed,
  operating profit and breaches.

**Where the comparison is valid.** Without a draw, interest or
reservation, the solvency state is empty and the run encodes as
version 2's. Starting states should therefore be identical across
versions. That is checked, not assumed. Where the actions differ, the
service change belongs to the actions, and the analysis says which.

**Binding.** For every practice, variant and policy, the analysis reports
whether the solvency model bound: any refusal, draw, interest or
distress. A practice where it never binds is reported as such.

**Expected, not assumed:**
* `noop` commits nothing and borrows only if cash would fall below zero.
  V0.7a's lowest cash in 90 days was $35,725, so `noop` should be
  identical to version 2 everywhere.
* `greedy` raises allocations whenever its rule says so. Its raises
  commit salary, so it may be refused where the headroom is small
  (financial pressure, staffing shortage), and more often with no credit.
* `conservative-v2`'s own guardrail (trailing profit less the raise) is
  stricter than the funding rule where the practice loses money, so it
  may never reach the rule.

Whatever the runs show is reported as it is.

## 3. Verification

* `benchmark verify` and `benchmark reproduce` on both results, on a
  second platform as well (linux/arm64).
* The contract reports' starting-state check and scenario fingerprints.
* The results' manifests are kept in the repository with the contract
  reports and the analysis.

## 4. Reporting

* [analyze.py](analyze.py) writes the tables from the two results
  directories:
  * starting states;
  * each run in the three variants;
  * the binding summary;
  * refusals;
  * paired differences;
  * policy minus `noop`.
* The README gives the findings and states the limitations.

## Amendments

**Amendment 1 (2026-10-29, after the runs, before any finding was
written).** Two fixes to [analyze.py](analyze.py); no seed, policy,
scenario, run or model changes.
* It compared each run's status with `complete`. The contract report
  writes `completed`, so every run was marked for checking. It now uses
  `completed`, and it lists any run that did not complete or replay.
* §2 named distress lines in the log as a measure. The results keep no
  log. Under solvency, cash is below zero only when the credit line is
  fully drawn, so the window's lowest cash below zero is the distress
  record. Its magnitude is the largest unfunded shortfall. It is now
  reported per run, and as runs in distress per practice and variant.
  Days ending below zero are no longer reported: distress at a 00:00
  disbursement can end the same day, when collections arrive.

