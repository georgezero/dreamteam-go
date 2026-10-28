# DreamTeam benchmark scenario suite and evaluation contract (V0.6a)

Status: V0.6b, development (0.6.0-dev.2). Not released. The suite is
`dreamteam-dev`, version 1
([benchmark/suite/dreamteam-dev-1.json](benchmark/suite/dreamteam-dev-1.json));
the contract is `dreamteam.benchmark-contract/v1` (§3). V0.6a defined them;
V0.6b runs a suite in one command and reports it under the contract (§9),
adds a financially conservative baseline (§7) and a demand diagnostic
(§8).

This document defines what DreamTeam's agent benchmark measures:
* five practices, each a different management problem;
* the metrics reported for them, grouped into dimensions;
* how development and held-out material are kept apart;
* the standard result an evaluation reports.

It builds on the V0.4 harness ([BENCHMARK.md](BENCHMARK.md)) and adds no
engine. Every scenario is a bundled scenario in the existing format, and
every metric is one the harness already reports. Running agents across
the suite is V0.6b and V0.6c.

**Scenario difficulty is not agent capability.** A scenario names a
problem and the constraints behind it. It does not claim that a good
outcome is reachable, and it encodes no preferred strategy. A practice
near capacity may not be improvable without losing money; a stable one
may be best left alone. The contract therefore never rewards action as
such, and reports each dimension separately.

## 1. Capability inventory

What the simulator can represent today, from the scenario format
(`dreamteam.scenario/v1`, `internal/scenario`) and the action set.

| Area | Supported | Not supported |
|---|---|---|
| **Format and identity** | One JSON file per bundled scenario, `extends` and `preset` overlays merged by key and by `id`, strict decoding, validation (`dreamteam scenario validate`), a coverage report (`dreamteam scenario coverage`), and a fingerprint: the SHA-256 of the resolved canonical JSON. | Deleting an inherited employee, room or source through an overlay (arrays merge by `id`; a roster day can be removed with `null`). |
| **Starting state** | A warm-up (`warmupDays`, 180 in every suite scenario) runs the practice from empty to the takeover, so bookings, the worklist and receivables are the practice's own. Opening cash is set at the takeover (`openingCashCents`). | Injecting a starting state directly, such as a given backlog, unpaid bills or a vacancy: every state comes from the warm-up. |
| **Demand** | Referral sources with base daily rates and priority and exam mixes; hour-of-day, weekday and month (`seasonFactorBP`) factors; booking lead times and windows; walk-ins with their own rate and mix; patient wait tolerance, cancellation and no-shows. | A trend or a shock. A rising demand can only be expressed through the month factors, which step on the 1st and repeat every year. The month factors apply to referrals, not walk-ins. |
| **Imaging capacity** | Rooms by modality, opening hours by weekday, appointment templates (slot lengths and categories), technologist rosters with room assignments and cover windows, fortnight rotas, contrast supervision. | A second scanner of a modality in one room, equipment downtime, room or scanner purchase. |
| **Reading capacity** | Radiologists' rosters (on site or remote) and site reading allocation; worklist policy; teleradiology contracts (an action). | Hiring a radiologist (not in the offer table); reading outside rostered blocks without a contract. No suite scenario has Sunday reading. |
| **Staffing** | Employees by role and skills, hourly or salaried pay, contracted weekly minutes, overtime rules, maximum shift, breaks. | Absence, sickness, resignation or turnover; staff performance differences. |
| **Finance** | Payers (mix, multipliers, delays, denials, short payment), charge lag, benefits, billing fee, per-study IT cost, payroll cycle, fixed monthly costs, opening cash. | A credit line or insolvency: cash may go below zero, and the shortfall is tracked (`shortfallCents`, `daysBelowZero`) but has no consequence. Prices and fixed costs do not change during a run. |
| **Management actions** and their lead times | `extend_shift` (today), `set_walkin_policy` and `set_worklist_policy` (immediate); `set_reading_allocation` (effective at least 14 days ahead); `contract_teleradiology` (start at least 14 days ahead, a 30-day minimum term, cancellation at least 30 days ahead); `hire` (technologists 28 days' recruiting lead at the standard rate, 21 at the premium rate; front desk 14 or 7; then paid training); `set_roster` (14 days' notice); `terminate` (14 days' notice). | New actions are out of scope. |
| **Evaluation** | The harness's metrics ([BENCHMARK.md](BENCHMARK.md) §6), the canonical result (`dreamteam.result/v1`) and the T90 evaluation under policy `v0.2-eval-4`. | T90 for any practice other than the bundled sustainable scenario: the policy pins that scenario's fingerprint and the harness reports every other scenario as `not_applicable`. |

## 2. The development suite (`dreamteam-dev`, version 1)

Five scenarios. Four are bundled scenarios added in V0.6a; the fifth is
the existing `sustainable` scenario, unchanged. Each comes from a source
documented before V0.6a, or adds one documented change to one.
`TestSuiteScenarios` (`internal/bench/suite_test.go`) checks that each
file is exactly its source, that its fingerprint is the recorded one, and
where T90 applies.

**Common to all five:**
* the established practice: four rooms (XR, US, CT, MR), opening Mon–Fri
  08:00–20:00 and Sat 10:00–18:00, closed Sunday;
* the same payers, exams, templates (except where noted), fixed costs of
  $88,500 a month, and $150,000 opening cash at the takeover on
  2026-10-13 06:00;
* two salaried radiologists ($480,000 a year each) with weekday reading
  blocks and alternate Saturday remote blocks; no Sunday reading;
* a 180-day warm-up;
* 90-day evaluation horizon, decisions daily at 07:00 (the suite's
  default; a benchmark configuration may choose otherwise).

Demand multipliers below are the calibration's imaging-weighted referral
demand relative to the established practice (×1.00 = 63.0 referral
orders a day before weekday and month factors).

### 2.1 Financial pressure (`financial-pressure`, version 1)

| | |
|---|---|
| Bundled scenario | `bench-financial-pressure` |
| Fingerprint | `3dcd153bb24c031fb32b2e1c5e1b6e63ef2eeaf9368dd1f111e373fefd1fb391` |
| Source | Step 0C `W115-SHR-MRft` (V0.2_DESIGN §4.7) |
| Demand | Referral demand ×1.15 (72.5 orders a day), base month factors |
| Staffing and coverage | The sustainable practice's: one XR/CT technologist shared across two rooms on weekday days, a CT/XR technologist Tue–Sat late, day and evening MR technologists, a US technologist, a front desk and an operations manager; radiologist site allocations 60% and 50% |
| Finance | As common; the staff costs of a practice sized for ×1.30 demand |
| Challenge | Costs exceed what the volume earns. Balancing service improvements against operating cost. |
| Underlying constraints | Step 0C found every staffing architecture loses money at ×1.15. Most fixed costs cannot be changed by any action; staff costs can, with notice. Improving service costs money. |
| T90 | Not applicable |
| Limitations | No insolvency mechanics: a negative balance is tracked, not enforced. Demand does not respond to service quality beyond patients' wait tolerance. |

### 2.2 Reporting backlog (`reporting-backlog`, version 1)

| | |
|---|---|
| Bundled scenario | `sustainable`, unchanged (internal id `established`) |
| Fingerprint | `e8718dbd1031feb326b4934d291f81eb14fd867d4a8fa6acc1673a75b55b76f8` |
| Source | Step 0C `W130-SHR-MRft` (V0.2_DESIGN §4.7; `TestSustainableIsThePolicyScenario`) |
| Demand | ×1.30 (81.9 orders a day), base month factors |
| Staffing and coverage | As financial pressure (81% of open room-hours staffed over the fortnight); radiologist site allocations 60% and 50% |
| Finance | As common |
| Challenge | Reading coverage leaves report turnaround well past the urgent and soon targets (60 and 240 minutes). Reducing reporting delay without excessive expenditure. |
| Underlying constraints | Reading capacity is set by two radiologists' blocks and allocations; there is no Sunday reading, so weekend work waits. More reading costs money (a higher allocation is salaried cost; teleradiology a retainer and per-study fees) and takes 14 days to start. The practice is near break-even. |
| T90 | Applicable: `v0.2-eval-4`, the evaluation policy in force, defined for exactly this scenario |
| Limitations | The backlog is structural (when reading happens), not a large unread queue at the takeover. |

### 2.3 Demand growth (`demand-growth`, version 1)

| | |
|---|---|
| Bundled scenario | `bench-demand-growth` |
| Fingerprint | `0146ac340d95c10457eaea37e06aaf24988b3cce23c651261e5bf5e7e2183fff` |
| Source | The sustainable scenario, with one change: the November, December and January month factors are October's ×1.05, ×1.10 and ×1.15 (10,710, 11,220 and 11,730; base 10,000, 9,300 and 10,500) |
| Demand | ×1.30 at the takeover, rising 5% a month from November; by January about the calibration's ×1.50 case |
| Staffing and coverage | As reporting backlog |
| Finance | As common |
| Challenge | Demand outgrows imaging and reading capacity. Anticipating capacity needs, given recruiting and allocation lead times, rather than reacting once losses mount. |
| Underlying constraints | Capacity added by hiring arrives 21–28 days after the decision plus training; reading allocation and teleradiology 14 days. Lost referrals are lost revenue, but capacity costs money before the demand arrives. |
| T90 | Not applicable (not the policy's scenario) |
| Limitations | Growth is a step on the 1st of each month, through the month factors, so it repeats in later years and does not apply to walk-ins. Within the 90-day horizon only November, December and 11 days of January are affected. Agents see growth only through orders and bookings; there is no forecast. |

### 2.4 Staffing shortage (`staffing-shortage`, version 1)

| | |
|---|---|
| Bundled scenario | `bench-staffing-shortage` |
| Fingerprint | `6ef37e929b8d3780635ca98274c2605a54d1bfe7c265c4d7b7c2b3510c0d1780` |
| Source | Step 0 `W130` (V0.2_DESIGN §4.5, §4.6) |
| Demand | ×1.30, base month factors |
| Staffing and coverage | The established practice's original rosters: no evening MR technologist, one technologist shared between X-ray and CT, a CT technologist Tue–Sat late, a US technologist; 74% of open room-hours staffed over the fortnight, against 81% in the sustainable practice (`dreamteam scenario coverage`) |
| Finance | As common, with one technologist fewer than the sustainable practice |
| Challenge | Insufficient coverage for the demand. Managing coverage, recruiting lead times and service reliability. |
| Underlying constraints | Step 0B found most lost CT and XR referrals had a free, staffed slot that could not be booked because the one shared technologist was committed; an added technologist "only pays for itself" and moves the bottleneck to reading. Hiring takes 21–28 days plus training; shift extensions are bounded by the maximum shift. |
| T90 | Not applicable |
| Limitations | Shortage means too few rostered staff, not absence: there is no sickness, resignation or vacancy mechanic. |

### 2.5 Stable practice (`stable`, version 1)

| | |
|---|---|
| Bundled scenario | `bench-stable` |
| Fingerprint | `d65113896ce59fd161a42627520efbf511346ebd0d64bda59d4df622b707ee0f` |
| Source | Step 0C `W130-DED-MRpt-A62`, the lead candidate (V0.2_DESIGN §4.7) |
| Demand | ×1.30, base month factors |
| Staffing and coverage | Dedicated X-ray and CT day technologists, a shared evening technologist Mon–Fri plus Saturday, day and part-time evening MR, US; radiologist site allocations 62.5% and 52.5% |
| Finance | As common |
| Challenge | A practice that is profitable at the takeover, with better access and reporting than the others. Recognizing when no action is appropriate, since needless changes cost money or disrupt service. |
| Underlying constraints | Step 0C: +2.8% mean T90 operating margin on 20 paired seeds (1.5% to 5.0%), profitable on all 20 in days 31–90. 85% of open room-hours staffed over the fortnight. The margin is thin, so a modest added cost can remove it. Reporting is still slower at weekends (no Sunday reading). |
| T90 | Not applicable (T90 is defined for the sustainable scenario only; the Step 0C margin was measured with the calibration harness, not the evaluation policy) |
| Limitations | "Stable" is relative: it still has weekend reporting breaches and some lost referrals. |

### 2.6 Unsupported or partly supported

No category was dropped. Two are partial:
* **Demand growth** uses the month factors as a trend. The smallest
  faithful extension would be a demand trend: a dated multiplier
  schedule for referral sources and walk-ins, validated like the month
  factors and included in the fingerprint. Not implemented; it needs an
  owner decision.
* **Financial pressure** has no consequence for running out of cash. The
  smallest extension would be a credit line with interest, or a
  cash-floor rule refusing discretionary spending. Not implemented.

A **reporting backlog at the takeover**, as distinct from slow
turnaround, would need either an initial-state injection or a dated
capacity change. Neither exists; the suite uses the structural backlog
of the sustainable practice instead.

## 3. Evaluation contract (`dreamteam.benchmark-contract/v1`)

**Unit.** One run: a scenario, a seed, a policy and its version, a horizon
and a decision cadence, through the V0.4 harness. Every run of a
scenario and seed starts from the same state (the report's
`initialConditions`).

**Dimensions.** Each run is reported on six dimensions. The metrics are
the harness's own definitions ([BENCHMARK.md](BENCHMARK.md) §6), from the
replayed session's canonical result; none is new.

| Dimension | Metrics |
|---|---|
| Operational performance | `examsCompleted`, `reportsFinal`, `tatP50Sec`, `tatP90Sec`, `backlogEnd`, `waitToExamP50Sec`, `waitToExamP90Sec` |
| Financial performance | `netRevenueCents`, `operatingExpenseCents`, `operatingProfitCents`, `margin`, `operatingCashFlowCents` |
| Staffing and capacity | `ordersReceived`, `ordersLost`, `lostRate`, `overtimeSec`, `heldOver`, `skippedBreaks`, `noCoverage`; staffing changes as accepted actions by type (from `decisions.json`) |
| Service reliability | `tatOverTarget`, `tatBreaches`, `waitsOver30`, `displacedAppointments`, `alertsRaisedAlert`, `alertsActiveAtEnd` |
| Long-term sustainability | `minCashCents`, `shortfallCents`, `daysBelowZero`; the T90 verdict and vector where applicable |
| Actions | `actionsAttempted`, `actionsAccepted`, `actionsRejected`; by category (operations or staffing) and type, from `decisions.json` |

The suite file lists the same grouping (`dimensions`), and
`TestSuiteDimensions` checks that every metric is one the harness
reports.

**Primary metrics by scenario.** What each scenario's challenge is about.
They are listed first in a report; nothing is dropped and nothing is
weighted.

| Scenario | Primary | Also watched |
|---|---|---|
| Financial pressure | `operatingProfitCents`, `operatingExpenseCents`, `minCashCents`, `daysBelowZero` | `tatP90Sec`, `ordersLost` |
| Reporting backlog | `tatP90Sec`, `tatBreaches`, `tatOverTarget`, `backlogEnd`; T90 | `operatingExpenseCents`, `operatingProfitCents` |
| Demand growth | `ordersLost`, `lostRate`, `examsCompleted`, `waitToExamP90Sec` | `operatingProfitCents`, `tatP90Sec` |
| Staffing shortage | `ordersLost`, `noCoverage`, `overtimeSec`, `waitToExamP90Sec`, staffing changes | `operatingProfitCents`, `tatBreaches` |
| Stable practice | `operatingProfitCents`, `tatP90Sec`, `ordersLost` against no action; actions taken | every dimension |

**Comparisons.** A policy is compared with a baseline (no action by
default) on the same scenario and seed: the difference of every metric,
policy minus baseline. Differences are reported per dimension and never
combined. A better figure in one dimension is not traded against a worse
one in another; both are shown.

**No score.** No composite score, weighting, leaderboard or ranking.
Spreads over seeds are minimum, median and maximum. No test of
significance is claimed at development scale.

**Service gates.** The only service gates in force are T90's, under
`v0.2-eval-4`, for the reporting-backlog scenario. For the other four the
contract reports T90 as `not_applicable`. It does not apply T90's gates or
thresholds to scenarios the policy was not defined for, and adds no
clinical policy.

**Missing metrics** (reported as missing, not estimated):
* imaging capacity utilization: no room-utilization metric exists;
* trends within a run: every metric is a whole-window total;
* model usage and latency: only from policies that record them (the LLM
  agent's `policy-evidence`).

## 4. Development and held-out material

**Development** (`split: development`): public scenario definitions and
public development seeds, for building and inspecting agents. Anything
may be run, read and published. `dreamteam-dev` version 1 and the V0.6a
characterization seeds 40901–40903 are development material.

**Held-out** (a future, separately versioned set, for example
`dreamteam-heldout-1`): the material on which a frozen agent is evaluated
once. V0.6a defines its governance and creates none of it.
1. **Separate from the clinical holdout.** It uses none of the clinical
   evaluation's sealed holdout, its manifest, its reserved seed streams,
   its unseal process or its guard. Its seeds must also pass the clinical
   holdout guard, as every benchmark seed does.
2. **Created by the owner, after the development suite is frozen.** The
   seeds (and any held-out scenario variants) are generated outside the
   repository by a procedure the owner chooses. Only a commitment, the
   SHA-256 of the set, is committed before any evaluation.
3. **Not seen in development.** No development run, test, CI job or
   agent setting may use a held-out seed or variant. The repository holds
   no held-out seed until it is revealed.
4. **Frozen agents only.** An agent is evaluated on the held-out set
   only after its version, prompt and settings are committed. Each agent
   version is evaluated once per held-out version; a rerun is reported as
   a rerun.
5. **Revealed after use.** After an evaluation the set is revealed, its
   commitment checked, and the set retired. The next evaluation uses a
   new version.
6. **Same contract.** Held-out results are reported under this contract
   and the same dimensions, with no score.

## 5. Reporting contract

A suite result, for every run, holds the following. Most of it is
already in the harness's files.

| Field | Source today |
|---|---|
| Scenario identity: suite entry, bundled scenario and fingerprint | Suite file; `result.json` `scenario.fingerprint` (not yet in `report.json`) |
| Policy identity and version | `report.json` `runs[].policy`; external programs' SHA-256 in `manifest.json` |
| Seed | `report.json` `runs[].seed` |
| Horizon and cadence | `report.json` `days`, `cadence` |
| Starting-state identity | `report.json` `initialConditions` (state digest per scenario and seed, and whether every run's is identical) |
| Action counts and categories | `report.json` `runs[].counts`; by type from `decisions.json`; categories as in [LLM_AGENT.md](LLM_AGENT.md) (operations: `extend_shift`, `set_walkin_policy`, `set_worklist_policy`; staffing: the rest) |
| Operational, financial, staffing and capacity, service reliability and sustainability outcomes | `report.json` `runs[].metrics`, grouped by §3 |
| T90 applicability and verdict | `report.json` `runs[].t90` (`not_applicable`, `not_requested`, or the verdict) |
| Replay verification | `report.json` `runs[].replayVerified`, `actionsMatchSession`, `replay`; `benchmark verify` |
| Model usage and latency, if available | The policy's evidence (`policy-evidence/summary.json`) |
| Paired comparisons by scenario and seed | `report.json` `pairs[]` (policy minus baseline) |
| Missing metrics | Listed per §3 |

V0.6a produced this view with `docs/benchmark/v06a-dev/analyze.py`. Since
V0.6b, `dreamteam benchmark suite run` writes it as the contract report
and `dreamteam benchmark report` prints it (§9).

## 6. Known limitations

* **Five practices from one family.** All five share the established
  practice's rooms, exams, payers and fixed costs; they differ in demand,
  staffing and reading allocation. They test different problems, not
  different kinds of practice.
* **The baselines use one lever.** `greedy` and `rules` only change the
  site reading allocation, so they cannot show what the staffing, demand
  or financial scenarios ask for. The characterization checks the
  scenarios, not the baselines.
* **Partly supported categories** (§2.6): demand growth through month
  factors; no insolvency consequence; no starting backlog.
* **T90 covers one scenario.**
* **Development seeds only.** Three characterization seeds show that the
  scenarios differ; they are not enough for statistical claims.
* **What the characterization found** ([v06a-dev/README.md](benchmark/v06a-dev/README.md)):
  * No scenario runs out of cash in 90 days with no action, so the
    sustainability dimension does not yet separate them.
  * Demand growth adds about 17% demand per weekday over reporting backlog
    by days 61–90. Its declared within-run condition (raw 30-day totals)
    held on one seed of three: the last window has two fewer weekdays,
    and the base profile's December decline absorbs part of the growth.
  * Financial pressure and staffing shortage both lose money with no
    action; they differ in expenses, lost demand, overtime and waits.
* **The conservative baseline never acted in the V0.6b development matrix**
  ([v06b-dev/README.md](benchmark/v06b-dev/README.md)). Its service signal,
  the oldest unfinished study at the 07:00 decision point, passed 24 hours
  on at most two mornings in a row (after the day with no reading), so it
  found no sustained problem in any scenario. The baselines therefore
  still offer two behaviours: no action, and twelve allocation raises.
* **The conservative baseline cannot invest its way out of a loss.** Its
  profit guardrail forbids raising reading capacity while the trailing 30
  days lose money, even when the loss comes from unbilled reports in the
  backlog (§7).
* **Design used seed 1.** The scenarios' intended conditions were checked
  with no-action runs on seed 1 before the characterization seeds were
  declared ([v06a-dev/PROTOCOL.md](benchmark/v06a-dev/PROTOCOL.md) §3).

## 7. The conservative baseline (`conservative`, version 1)

A built-in policy (`internal/bench/conservative.go`), added in V0.6b
beside `noop`, `greedy` and `rules`, which are unchanged. Its purpose is
to avoid unnecessary spending while still responding to a sustained
reporting problem, and to choose no action otherwise. Its rules and
thresholds were fixed and committed before it ran on any development or
suite seed. They were not derived from the V0.6a results.

**What it reads** (public, through the agent gate; no seed, no hidden
state):
* the compact observation's `worklist.oldestMin` (age of the oldest
  unfinished study);
* `finance.cashCents`;
* `finance.trailing30.operatingProfitCents`;
* the date;
* the action catalog's `set_reading_allocation` targets (allocation in
  effect, scenario allocation, maximum, earliest effective date, pending
  changes);
* the staffing preview's `beforeSalary30Cents` and `afterSalary30Cents`
  (the change's salary with benefits per 30 days).

**Its one lever** is the site reading allocation, the only one the
existing baselines use, so that what differs is the decision rule, not the
action set.

**At each decision point, in order:**

| Step | Rule |
|---|---|
| Track | Counts the consecutive decision points with the oldest unfinished study over 24 hours (*pressure*) and at most 8 hours (*clear*) |
| Wait | Does nothing while any radiologist's allocation change is pending, or within 14 days of the effective date of its own last change |
| Step back | If an allocation is above its scenario level and either the trailing-30-day operating profit is below zero or *clear* has lasted 7 decision points: the radiologist furthest above it loses 5 points, never below the scenario level |
| Raise | If *pressure* has lasted 3 decision points, cash is at least $50,000, the trailing-30-day operating profit is at least zero, and no more than 5 points are already in effect above the scenario levels in all: the radiologist with the lowest allocation in effect and room to grow gains 5 points from the earliest lawful date, but only if the trailing-30-day profit less the preview's added salary per 30 days stays at or above zero |
| Otherwise | No action |

Every change is checked with `validate` and the staffing `preview` and is
submitted only if valid, accepted and displacing no booked patient. At
most one change per decision point.

**Guardrails, summarized:**
* **Financial:** no raise while the practice loses money over the
  trailing 30 days, none that its own cost would turn into a loss, none
  below $50,000 cash; steps back when money is lost.
* **Frequency and size:** one change at a time, 14 days apart (the
  allocation's notice), at most 5 points above the scenario levels in
  all.
* **Lead times:** changes are made from the earliest lawful date, and the
  next waits until the last has taken effect and run for 14 days.

**Why these numbers:**
* 24 and 8 hours and 5 points are the `rules` policy's.
* 3 decision points in a row is three days, so a weekend without reading
  (Sunday and Monday mornings) is not enough.
* 7 is a week; 14 days is the allocation's notice.
* $50,000 and the 5-point cap are round numbers.

**Known behaviour, from its unit-test runs** (test seed 7, not a
development seed):
* In the sustainable practice the oldest study passes 24 hours only on
  Sunday and Monday mornings, so the policy finds no sustained problem
  and does nothing.
* Where a backlog makes the practice lose money (revenue is billed on
  final reports), the profit guardrail forbids the very spending that
  might restore revenue. That is its stance, not a defect, and is
  reported as a limitation (§6).

**Tests** (`internal/bench/conservative_test.go`):
* every rule, threshold and guardrail, one at a time;
* a test-only overloaded practice (not a suite scenario), where it must
  raise;
* the financial-pressure scenario, where it must not;
* over each run: valid changes only, one per decision point, 14 days
  apart, within the cap.

## 8. Demand diagnostic (`dreamteam.demand-diagnostic/v1`)

V0.6a's characterization judged demand growth on raw 30-day totals of
referral orders, which depend on how many weekdays a window holds. That
check failed on two seeds of three and stays recorded as failed
([v06a-dev/README.md](benchmark/v06a-dev/README.md)). V0.6b adds a
separate, prospective diagnostic, declared before its first evaluation
([v06b-dev/PROTOCOL.md](benchmark/v06b-dev/PROTOCOL.md)).

**What it measures**, for each entry, seed and 30-day window from the
takeover, per **operating day** (a day the scenario's opening hours are
not closed; Sunday in every suite scenario):

| Quantity | Source | Kind |
|---|---|---|
| Generated demand: referral orders | the result calendar's `orders`, the simulator's count of orders as they arrive, before booking | demand |
| Generated demand: walk-ins | the calendar's `walkIns` (arrivals) | demand |
| Completed imaging | the calendar's `exams` | outcome |
| Lost orders | the calendar's `lost` | outcome |

These are different quantities. Generated demand is never inferred from
completed imaging or lost orders, which depend on capacity and decisions.
Orders per weekday (Monday to Friday) are reported as a secondary figure,
and orders arriving on closed days are counted, not hidden.

**Reliability check (R1).** Generated demand, day by day, must be
identical across the policies of an entry and seed. If it is not, the
diagnostic reports that demand depends on decisions and is not a clean
measure.

**Declared conditions,** for the demand-growth entry with no action:
* **G1:** referral orders per operating day are higher in days 61–90 than
  in days 1–30, on every seed.
* **G2:** the ratio of demand growth's referral orders per operating day
  to reporting backlog's (same seed and window) rises from days 1–30 to
  days 61–90, and is above 1 in days 31–60 and 61–90, on every seed.

**Script:** `docs/benchmark/v06b-dev/demand_diagnostic.py RESULTS` reads a
suite run's results (its contract report, results and saved sessions).
Nothing in the simulator or the scenarios changes.

## 9. Running a suite and the contract report

**A suite run** is a configuration (`dreamteam.benchmark-suite-run/v1`)
naming a suite file and choosing from it:

| Field | Meaning |
|---|---|
| `schema`, `id`, `description` | As a benchmark configuration |
| `suite` | The suite file, relative to the configuration's directory |
| `entries` | Suite entry IDs to run, in order (default: all, in the suite's order) |
| `seeds`, `seedPlan` | Development seeds, and how they were chosen; the sealed holdout guard applies |
| `days`, `cadence` | Horizon and decision cadence (default: the suite's) |
| `policies`, `external`, `baseline` | As a benchmark configuration |
| `output`, `evaluation` | As a benchmark configuration; T90 is evaluated only where the evaluation policy in force applies |

    dreamteam benchmark suite run docs/benchmark/v06b-dev/suite-run.json -out results
    dreamteam benchmark report results                # the contract report
    dreamteam benchmark report results -harness       # the harness's own report
    dreamteam benchmark report results -format json   # the contract report as JSON
    dreamteam benchmark verify results [-t90]
    dreamteam benchmark reproduce results

**How it runs.** The runner checks the suite file before anything runs:
* its header and dimensions;
* every selected entry's bundled scenario against the recorded
  fingerprint;
* T90 claimed exactly where the evaluation policy in force applies.

It then expands the suite run into an ordinary benchmark configuration
(`config.json` in the results) and plays it through the V0.4 harness, the
same code as `benchmark run`. Every run's files, and the harness's
`report.json` and `report.txt`, are byte for byte those of `benchmark run`
on that configuration (`TestSuiteRunIsTheMatrix`).

**Results** add four files to an ordinary results directory, all hashed in
the manifest's `suite` list:

| File | Contents |
|---|---|
| `suite-run.json` | The suite-run configuration, verbatim |
| `suite.json` | The suite file, verbatim |
| `contract.json` | The contract report (`dreamteam.benchmark-contract/v1`) |
| `contract.txt` | The same, as text |

**Exit status** (as `benchmark run`):
* 0: every run completed;
* 1: a run failed, or the results could not be written;
* 2: the configuration or suite is invalid, and nothing ran.

A failed run is recorded and classified like any other. The contract
report then says `"status": "partial"`, lists the failed runs, and its
text opens with `PARTIAL SUITE: k of N runs completed`. An interrupted
run leaves no manifest, so its directory is not a results directory at
all.

**Verify and reproduce.** `benchmark verify` adds, for a suite run:
* the suite files' hashes;
* the suite file still matching the bundled scenarios;
* `config.json` being the expansion of `suite-run.json`;
* every run's scenario fingerprint being its entry's;
* the contract report rebuilt from the run records.

`benchmark reproduce` reruns the suite run from its stored files and
compares every file, the suite files included. Ordinary results gain no
`suite` field, so their manifests are unchanged.

**The contract report** (`contract.json`):

| Field | Contents |
|---|---|
| `schema`, `engine`, `suite` (`id`, `version`, `split`), `suiteRun` | Identity |
| `status`, `expectedRuns`, `completedRuns`, `failedRuns` | Completeness |
| `seeds`, `days`, `cadence`, `policies` (id and version), `baseline` | The matrix |
| `entries[]` | Entry ID, version, category, bundled scenario, fingerprint, T90 applicability; per seed the starting state (`stateDigest`), how many runs started from it, and whether all did identically |
| `missing` | The contract's missing measures, from the suite file |
| `runs[]` | Per run: entry, scenario, fingerprint, seed, policy, horizon, starting-state digest, status, replay verification, actions (operations, staffing, by type, in refused batches), T90 (applicability, status, verdict, classification), and the six `dimensions`, each metric with its recorded `value` or `absent` and the reason |
| `pairs[]` | Per entry, seed and policy against the baseline: per dimension, policy − baseline for every integer metric (`null` when either is absent or a ratio), and the difference in operations and staffing actions |

`contract.txt` shows the same content in ten sections:
* scenarios and starting states;
* runs;
* the six dimensions (minimum, median and maximum over completed runs per
  entry and policy);
* paired differences (median and range over seeds);
* missing measures, and every metric absent in any run with its reason.

Nothing is combined: no score, weighting, leaderboard or ranking.
