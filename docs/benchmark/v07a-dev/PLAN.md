# V0.7a plan: characterizing radiologist overtime costing

Plan version `v07a-plan/1`, written 2026-10-29. It is committed with the
overtime model, before any run on its seeds.

**Purpose.** V0.7a changes one economic rule in new scenario versions:
* an on-site salaried radiologist's time on duty past the rostered end is
  paid at the salary's hourly equivalent
  ([V0.7_OVERTIME_DESIGN.md](../../V0.7_OVERTIME_DESIGN.md)).

This characterization measures what that rule does to the five V0.6
practices under the three deterministic baselines. **It is not a
ranking.** There is no composite score and no significance test, and no
policy is tuned on the results. No LLM agent is run.

Anything this plan does not settle is decided before the run it affects,
and committed as a dated amendment at the end. An amendment never changes
a seed, a policy, a scenario or the overtime model.

## 1. What is fixed

| | |
|---|---|
| Overtime model | [V0.7_OVERTIME_DESIGN.md](../../V0.7_OVERTIME_DESIGN.md) §3: time on duty past the rostered end × annual salary ÷ (52 × `weeklyOvertimeHours`), straight time, floor per radiologist-day, plus the scenario's benefits (25%), posted at going home, paid at the next payroll; on-site salaried radiologists only |
| Version 1 (historical economics) | suite `dreamteam-dev` 1 ([file](../suite/dreamteam-dev-1.json)), unchanged: financial pressure `3dcd153b…`, reporting backlog `e8718dbd…` (T90 applies), demand growth `0146ac34…`, staffing shortage `6ef37e92…`, stable `d6511389…` |
| Version 2 (overtime paid) | suite `dreamteam-dev` 2 ([file](../suite/dreamteam-dev-2.json)): the `-ot` variants, financial pressure `43f2729d…`, reporting backlog `366c6373…`, demand growth `bc157649…`, staffing shortage `9436e988…`, stable `5bb6aa65…`; T90 not applicable (the evaluation policy in force names version 1's sustainable scenario only) |
| Contract | `dreamteam.benchmark-contract/v1`, the contract report as V0.6b writes it |
| Engine | 0.7.0-dev.1 (this commit) |
| Policies | `noop` 1, `greedy` 1, `conservative-v2` 2, unchanged since V0.6c. None of them extends a shift. |
| Seeds | 41401, 41402, 41403: fresh, the next block after V0.6c's 41301. Not reference, development, reserved-stream or holdout seeds. The same seeds for both versions. |
| Horizon and cadence | 90 days; decisions daily at 07:00 |
| Configurations | [characterize-v1.json](characterize-v1.json) (T90 on), [characterize-v2.json](characterize-v2.json) (T90 off): 45 runs each, 90 in all |

**Failures.** A run that fails is reported with its status. An
infrastructure failure (the machine, not the simulator) is rerun once
into a new directory, and both are kept. Nothing is rerun to change a
result.

## 2. What is compared

The two versions differ only in the overtime rule. Each run is therefore
paired with the other version's run on the same entry, seed and policy.
* **Version 2 minus version 1, per run:**
  * **Operating expense:** the change in the harness's overtime cost
    (the radiologist charge before benefits, where operations are
    unchanged), operating expense and operating profit.
  * **Management actions:** whether the accepted actions, and the times
    they were decided, are identical. Where they differ, both lists.
    `conservative-v2`'s guardrails read the trailing profit, so it may act
    differently.
  * **Service outcomes:** TAT p90, breaches, exams completed, orders lost,
    and radiologist overtime minutes.
  * **Cash and profit:** operating profit, cash at the end, minimum cash.
* **Within each version:** policy minus `noop` on the same entry and seed,
  operating profit and breaches, as the contract report gives them.
* **Starting states:** whether version 1 and version 2 start from the same
  state (the warm-up runs under the scenario's own rule).

**Where the comparison is valid.** The paired difference attributes a
change to the economics only where both runs start from the same state.
If the actions also differ, the service change belongs to the actions,
and the analysis says which. Versions are not pooled. V0.6 results are
not restated: they remain results of the historical economics.

**Expected, not assumed:**
* `noop` and `greedy` take no action that depends on money. Their
  operations should be identical between versions, and their profit
  lower in version 2 by the overtime charge with benefits.
* Radiologist overtime here comes from hold-over (contrast supervision
  past the scheduled end), since no baseline extends a shift.

Whatever the runs show is reported as it is.

## 3. Verification

* `benchmark verify` on both results (with `-t90` for version 1), and
  `benchmark reproduce` on both, on a second platform as well.
* The contract reports' starting-state check and scenario fingerprints.
* The results' manifests are kept in the repository with the contract
  reports and the analysis.

## 4. Reporting

* [analyze.py](analyze.py) writes the tables from the two results
  directories: starting states, each run under both versions, the paired
  differences, policy minus `noop` in each version, and where actions
  differ.
* The README gives the findings: expense increases, changes in
  management actions, service changes, and cash and profit. It also
  states the limitations.
