# V0.7b: characterizing financial solvency

This is the characterization in [PLAN.md](PLAN.md) (`v07b-plan/1`). It was
committed in 28b9a27 with the solvency model, before any run on its seeds.
It has one amendment, which fixes the analysis script and was made after
the runs and before any finding was written.

It runs the five practices three ways:
* **Version 2** (suite `dreamteam-dev` 2): radiologist overtime paid, no
  credit line or funding rule. This is the comparison.
* **Version 3, `-credit`:** a $50,000 credit line at 10% a year, and the
  funding rule ([V0.7_SOLVENCY_DESIGN.md](../../V0.7_SOLVENCY_DESIGN.md)).
* **Version 3, `-nocredit`:** no credit, and the funding rule.

The policies are `noop`, `greedy` and `conservative-v2`. **Not a
ranking:** there is no score and no significance test, and no policy was
tuned. No LLM agent was run.

## Files

| File | What it is |
|---|---|
| `PLAN.md` | The plan, as declared, with amendment 1 |
| `characterize-v2.json`, `characterize-v3.json` | The suite-run configurations, as run |
| `ci-suite-run.json` | CI's 21-day smoke test of suite version 3 (not part of the characterization) |
| `analyze.py` | The tables from the two results directories |
| `tables.md` | `analyze.py`'s output |
| `characterize-v<N>/contract.txt`, `contract.json` | The contract reports |
| `characterize-v<N>/report.txt`, `report.json`, `manifest.json` | The harness's reports and the manifests |

## 1. The runs

| | |
|---|---|
| Seeds | 41501, 41502, 41503, the same for every version |
| Horizon | 90 days, decisions daily at 07:00; no T90 |
| Runs | 45 (version 2) and 90 (version 3), 135 in all, on linux/amd64 in 17 min (both in parallel) |
| Status | complete, all 135; every run replay-verified; no failure, no rerun |
| Engine | 0.7.0-dev.2 (28b9a27) |

## 2. Where the model binds

A run is bound by the model if anything was refused for funding, anything
was drawn or charged, or cash went below zero.

| | `-credit` ($50,000) | `-nocredit` |
|---|---|---|
| `noop` | never | never |
| `conservative-v2` | never | never |
| `greedy` | financial pressure only: 3 refusals in 3 runs; one draw of $12,584, $8 of interest | every practice: 65 refusals in 14 of 15 runs; one distress |

**It never binds for `noop` or `conservative-v2`.**
* **`noop`** commits nothing. Its cash never fell below $15,919, so it
  never borrowed. Its 30 runs are identical to version 2: actions,
  operations, money and starting states.
* **`conservative-v2`** changed allocations only where its own guardrail
  allowed: one change in demand growth 41501, and one to three in each
  stable run. All were funded.
  * Its rule asks whether trailing profit covers the raise. That is
    stricter than whether the practice can fund it. In financial pressure,
    staffing shortage and reporting backlog it never asks for a raise, so
    the funding rule is never reached.
  * In the practices where it does ask, the headroom was ample.
  * As the owner anticipated, its profitability guardrail is more
    restrictive than the simulator's financial capacity.

**`greedy` is the only policy that binds.** It raises a radiologist's site
allocation by 5 points every week, whatever the money.
* **With a $50,000 line**, only financial pressure binds:
  * the last two raises refused in seed 41502, the last one in 41503;
    41501 unaffected;
  * in 41502, cash would have fallen to −$12,584 on Jan 8. The line drew
    exactly that, cash stayed at $0, and the line was repaid by
    collections: $8 of interest in all.
* **With no credit**, every practice binds:
  * 65 refusals, each a raise refused once the headroom (cash less what
    is already owed) could not cover 30 days of the higher salary;
  * the first refusals came from mid-November (financial pressure) to
    the last fortnight (stable);
  * stable seed 41501 is the only run unaffected; in stable 41503 only the
    last day's raise was refused.
  * `greedy` retries the refused raise at each review. The refusal code
    tells it why, but its rule does not change.

## 3. Cash, credit and distress

| Practice, seed (`greedy`) | Version 2: lowest cash | `-credit`: lowest cash, most drawn, interest | `-nocredit`: lowest cash |
|---|---|---|---|
| Financial pressure 41501 | $9,108 | $9,108, $0, $0 | $13,875 |
| Financial pressure 41502 | **−$12,584** | $0, **$12,584**, **$8** | **−$1,981 (distress)** |
| Financial pressure 41503 | $1,623 | $1,623, $0, $0 | $12,271 |

* **Version 2** let financial pressure 41502 go to −$12,584 on Jan 8 with
  nothing to finance it: the shortfall V0.6 recorded and ignored.
* **With credit**, that payroll was financed: the line drew $12,584,
  collections repaid it from the next midnight, and the window's cash
  bridge reconciled to the cent.
* **With no credit**, the funding rule had already refused 9 raises. The
  practice still could not fund a payroll it already owed: $1,981 short
  at 00:00 on a payday, recovered the same day. The payroll was paid and
  the shortfall recorded. This is the one distress in 135 runs. It shows
  that refusing new commitments does not guarantee liquidity: the
  obligations already incurred can exceed cash on their own.
* No other run drew credit or went below zero.

## 4. Profit and service

The funding rule only removes spending, so where it binds profit rises and
service falls, relative to version 2 (`greedy`, `-nocredit`):

| Practice | Δ operating profit (3 seeds) | Δ breaches | Δ TAT p90 |
|---|---|---|---|
| Financial pressure | +$6,164 to +$12,658 | +4 to +6 | +11 to +24 min |
| Reporting backlog | +$740 to +$3,781 | 0 to +4 | 0 to +5 min |
| Demand growth | +$740 to +$1,973 | 0 | 0 to +1 min |
| Staffing shortage | +$1,973 to +$6,164 | 0 to +1 | +1 to +8 min |
| Stable | $0 to +$740 | 0 | 0 |

* Revenue and exams are unchanged in every pair.
* The raises refused were the latest ones. Their reading time would have
  come after most of the window.
* With a $50,000 line, `greedy`'s profit changed by +$75 in one run (41502:
  the two refused raises, less $8 of interest), and by $0 elsewhere.
* **Versus no action**, `greedy` still loses $19,895 to $32,656 over 90
  days in every practice, and has 32 to 329 fewer breaches. The funding
  rule limits its spending, but most of it falls before the headroom runs
  out.

## 5. Starting states and pairing

**All 15 entry-seed pairs start from the same state in all three
variants.**
* The warm-up runs without solvency (the model starts at the takeover).
* A version 3 run with nothing drawn or reserved encodes exactly as
  version 2's.

So every paired difference here is the model's effect alone. Where the
actions differ, the service change belongs to the refused actions.

## 6. Integrity

| Check | Result |
|---|---|
| `benchmark verify`, version 2 and version 3 (linux/amd64) | 419 of 419; 824 of 824 |
| `benchmark reproduce` on linux/amd64 | 281 of 281; 551 of 551 |
| `benchmark reproduce` on linux/arm64 (Raspberry Pi) | 281 of 281; 551 of 551 |
| Starting states identical across versions and policies | 15 of 15 entry-seed pairs |
| Window cash bridge (collections, disbursements, capital, credit) | reconciled in all 135 runs |
| Fingerprints checked by the runner | the five version 2 and ten version 3 scenarios |

## 7. Limitations

* **Three seeds, deterministic baselines only.** No statistical claim.
* **Little borrowing.** With these practices' opening cash (about
  $150,000) and 90 days, only one run needed the line. The credit
  mechanics are exercised mainly by the financial tests.
* **Refusals came late.** In these practices the headroom shrinks slowly,
  so the rule refused only the latest raises. A practice with less
  opening cash, or a longer horizon, would bind sooner.
* **Funding is a 30-day test.** A raise funded now still commits the
  practice to salary indefinitely (V0.7_SOLVENCY_DESIGN §12).
* **Distress has no consequence beyond refusals.** Payroll is always paid
  and nothing shuts down.
* **The baselines cannot read the refusal.** `greedy` retries refused
  raises. How agents respond to the funding rule is for V0.7c.
