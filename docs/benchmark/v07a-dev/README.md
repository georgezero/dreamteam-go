# V0.7a: characterizing radiologist overtime costing

This is the characterization in [PLAN.md](PLAN.md) (`v07a-plan/1`). It was
committed in 790972c with the overtime model, before any run on its
seeds, and has no amendments. It runs the five V0.6 practices under two
economics:
* **Version 1**, the historical economics: suite `dreamteam-dev` 1, with
  radiologist overtime unpaid.
* **Version 2**: suite `dreamteam-dev` 2, with an on-site radiologist's
  time on duty past the rostered end paid at the salary's hourly rate
  ([V0.7_OVERTIME_DESIGN.md](../../V0.7_OVERTIME_DESIGN.md)).

The policies are `noop`, `greedy` and `conservative-v2`. **Not a
ranking:** there is no score and no significance test, and no policy was
tuned. No LLM agent was run.

## Files

| File | What it is |
|---|---|
| `PLAN.md` | The plan, as declared |
| `characterize-v1.json`, `characterize-v2.json` | The suite-run configurations, as run |
| `ci-suite-run.json` | CI's 21-day smoke test of suite version 2 (not part of the characterization) |
| `analyze.py` | The tables from the two results directories (declared with the plan) |
| `tables.md` | `analyze.py`'s output |
| `characterize-v<N>/contract.txt`, `contract.json` | The contract reports |
| `characterize-v<N>/report.txt`, `report.json`, `manifest.json` | The harness's reports and the manifests |

## 1. The runs

| | |
|---|---|
| Seeds | 41401, 41402, 41403, the same for both versions |
| Horizon | 90 days, decisions daily at 07:00; T90 requested for version 1 (reporting backlog) |
| Runs | 45 per version, 90 in all, on linux/amd64 in 9 min 43 s (both versions in parallel) |
| Status | complete, 45 of 45 each; no failure, no rerun |
| Engine | 0.7.0-dev.1 (790972c) |

## 2. What changes

**Radiologist overtime is not rare, even with no action.** No baseline
extends a shift. All radiologist overtime here is **hold-over**:
* a radiologist stays past the scheduled end to supervise a contrast exam;
* or contrast patients wait with no other on-site radiologist.

With no action, the two radiologists were held over **39–69 hours in 90
days**, depending on the practice and seed. Under version 2 that costs:

| Entry | Radiologist hold-over, 90 days (no action) | Charge (before benefits) | Operating profit, version 1 → version 2 (no action) |
|---|---|---|---|
| Financial pressure | 39–50 h | $9,035–$11,445 | −$36,907 → −$51,214; −$42,109 → −$53,403; −$40,751 → −$52,514 |
| Reporting backlog | 61–66 h | $14,088–$15,335 | $2,598 → −$16,358; −$6,924 → −$24,534; −$3,006 → −$22,175 |
| Demand growth | 63–69 h | $14,515–$15,831 | $13,761 → −$5,039; $2,130 → −$16,014; $8,723 → −$11,066 |
| Staffing shortage | 58–68 h | $13,433–$15,744 | −$18,508 → −$38,188; −$33,834 → −$50,625; −$28,478 → −$46,724 |
| Stable | 40–55 h | $9,338–$12,773 | $26,593 → $10,626; $16,349 → $4,676; $17,397 → $4,264 |

Profit values are per seed, in seed order.

**Expense.**
* The charge is $230.77 an hour, with benefits (25%) on top.
* Operating expense rises by $11,294–$19,788 per run. That is the charge
  plus benefits, and every other line is unchanged.
* In every run whose operations are unchanged, the harness's overtime cost
  rises by exactly the charge.

**Practices that broke even now lose money.**
* Reporting backlog and demand growth, profitable or near break-even with
  no action under version 1, lose $5,000–$24,500 over 90 days under
  version 2.
* The stable practice stays profitable, but by $4,264–$10,626 instead of
  $16,349–$26,593.

**Cash.**
* Cash at the end falls by $12,268–$21,130.
* That is more than the profit change, by $974–$1,729 per run. The
  overtime of the warm-up days before the takeover was accrued in the
  opening state and paid at the first payroll.

## 3. Changes in management actions

**`noop` and `greedy`** took exactly the same actions at the same times
under both versions, in all 30 run pairs:
* their exams, reports, TAT, breaches, orders lost and radiologist
  overtime minutes are identical;
* their revenue is identical;
* their profit differs by exactly the charge with benefits;
* greedy's paired difference from no action is the same in both versions
  (−$29,431 to −$33,049).

Greedy raises allocations whatever the money. The radiologist overtime is
the same with or without its raises, because hold-over is driven by
contrast patients, not reading.

**`conservative-v2` acted differently in 7 of its 15 run pairs.** Its
guardrails read the trailing 30 days' profit, which now includes the
charge:

| Entry, seed | Version 1 (accepted allocation changes) | Version 2 | Δ breaches | Δ TAT p90 |
|---|---|---|---|---|
| Reporting backlog 41401 | raise 12-17 | none | +11 | +17 min |
| Reporting backlog 41402 | raise 11-19, undo 12-17 | none | +19 | +11 min |
| Demand growth 41401 | raise 12-16 | raise 12-31 | +13 | +48 min |
| Demand growth 41402 | raise 11-19, undo 12-17 | none | +40 | +39 min |
| Stable 41401 | raise 10-20, raise 11-17, back 12-15 | raise 11-04, undo 12-02 | +17 | +70 min |
| Stable 41402 | raise 10-20, raise 11-17, back 12-15 | raise 11-18, undo 12-16 | +30 | +74 min |
| Stable 41403 | raise 10-27, undo 11-24 | none | +14 | +37 min |

* **Fewer raises, later, and more undos.** Under version 2,
  `conservative-v2` never raised past 57.5% (version 1 reached 62.5%
  twice). In the stable practice its reviews undid each raise, where
  version 1's had kept or extended them. (Its rule retreats when the
  trailing profit is negative or service is clear.) Where the practice
  now loses money, its profit guardrail allowed no raise.
* **Service was correspondingly worse** in those runs: 11–40 more
  breaches, and TAT p90 11–74 minutes longer.
* **No change in exams.** Revenue is $1–$1,975 lower: fewer reports were
  final by the end, and revenue is earned at report final.
* **Versus no action, in version 2:**
  * stable 41401: −$2,301 profit for 12 fewer breaches (version 1:
    −$7,898 for 29);
  * stable 41402: −$2,301 for 5 (version 1: −$8,052 for 35);
  * stable 41403: identical to no action;
  * reporting backlog and demand growth, every seed: identical to no
    action.
* The other 8 run pairs (financial pressure and staffing shortage, where
  it never acts, plus seed 41403 in reporting backlog and demand growth)
  had identical actions.

This is the designed behaviour, not a tuning target. The rule was not
changed after these results.

## 4. Starting states

**Version 1 and version 2 never start from the same state** (15 of 15
entry-seed pairs differ).
* The warm-up days run under each scenario's own rule.
* Version 2's opening ledger carries the warm-up's radiologist overtime as
  accrued payroll.
* Opening cash, staff, patients and schedule are the same. The operations
  that follow are identical for `noop` and `greedy`.

The paired comparison is therefore valid for operations and actions. The
money difference includes the warm-up carry-over, $974–$1,729 of cash per
run, as well as the charge in the window.

## 5. Integrity

| Check | Result |
|---|---|
| `benchmark verify -t90`, version 1 (linux/amd64) | 437 of 437 |
| `benchmark verify`, version 2 (linux/amd64) | 419 of 419 |
| `benchmark reproduce` on linux/amd64 | version 1: 290 of 290; version 2: 281 of 281 |
| `benchmark reproduce` on linux/arm64 (Raspberry Pi) | version 1: 290 of 290; version 2: 281 of 281 |
| Starting states identical across policies (contract reports) | 15 of 15 entry-seed pairs, each version |
| T90, version 1 reporting backlog | NO GO, 9 of 9 (as in V0.6); not applicable in version 2 |
| Fingerprints checked by the runner | the five version 1 and five version 2 scenarios |

## 6. Limitations

* **Three seeds, deterministic baselines only.** No statistical claim.
  Because no baseline extends a shift, all the overtime here is hold-over;
  how agents that extend shifts respond is for V0.7c.
* **Idle time inside an extension is paid.** The engine has no early
  release, so a radiologist stays to the scheduled end.
* **Radiologists only.** The salaried operations manager's overtime is
  still unpaid.
* **Straight time.** No premium, call pay or payroll tiers; benefits at
  the scenario's existing 25%.
* **Still no insolvency consequence.** Several practices now lose money
  throughout, and nothing stops them (V0.7b).
* **Extensions add no reading blocks.** A radiologist's extension adds
  on-site presence (contrast supervision), not reading time. That was
  true before V0.7a, and is now documented.
