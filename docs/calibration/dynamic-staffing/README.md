# V0.2 dynamic staffing reorganization benchmark

**Question (owner, 2026-10-15):** starting from the sustainable scenario's
shared-staffing practice, can a player improve the practice with the
implemented actions alone (`hire`, `terminate`, `set_roster` and the V0.1
controls)? What does the transition cost, how much does it improve
operations and service, and when does it pay back?

**Answer: yes, financially, but service needs the reading actions.** Both
reorganizations pay back within 180 days on all 20 held-out seeds:

* **Partial (convert the MR evening to part-time):** costs $9.8k up front,
  earns +$5.0k a month from day 42, and breaks even around **day 101**
  (100–106). Service is unchanged.
* **Full (the closest legal path to Step 0C's lead staffing):** costs
  $23.7k, earns +$8.2k a month, and breaks even around **day 134**
  (113–175). On cash it pays back on only 11 of 20 seeds by day 180.
  * It images 265 more patients and loses 179 fewer referrals per 90 days.
  * Reading capacity does not keep up. The urgent-report breach rate rises
    from 11% to 53%, and the reporting backlog roughly triples (30 → 93
    studies) and is still growing at T180.

Recommendation: **CONDITIONAL GO**. Staffing actions give real, measurable
agency at a credible payback. A service-acceptable full reorganization
needs `set_reading_allocation` (see *Decision*).

**Follow-up (Slice 4, plan D):** the same full reorganization with the
lead's 62.5/52.5% reading allocation from its earliest legal date (Oct 27)
brings breach rates back within the provisional gates on average and
below no action. It costs about $2.5k a month in salary and keeps C's
profit (+$12.6k at T180, break-even day 141). The pre-registered rule
gives **CONDITIONAL GO** (see *Plan D*).

All figures are from `results/eval/report.md`, which `report.py` generates
from the per-seed result files. The plans were fixed on development seeds
before any held-out seed was run.

## Method

### Starting practice

The sustainable scenario's starting state, as decided by the owner on
2026-10-15. It is the Step 0C control `W130-SHR-MRft`:

* shared XR/CT technologists (Riley and Sam);
* a full-time evening MR technologist, Alex (E11);
* 12-minute XR slots;
* imaging-weighted referral demand ×1.30, the documented sustainable demand.

It is loaded unchanged from
`../feasibility/step0c/experiments/W130-SHR-MRft-s20.json`:

| | |
|---|---|
| Scenario fingerprint (merged) | `e8718dbd1031feb326b4934d291f81eb14fd867d4a8fa6acc1673a75b55b76f8` |
| Base (`established`) | `b1cbc2a0108f22e34082a0d39f72499678a3eb2f8b2501d483f3525a39ad806a` |
| Overlay SHA-256 | `7484fb0296a78beac4a2da416f31cc5a62fc43191a77d1335a411372745c9637` |
| Engine | `0.2.0-dev.4` (identical outcomes under `0.2.0-dev.3`, see *Verification*) |

Nothing in the model changed for this benchmark: economics, patient
behaviour, demand, wage rates, opening hours and templates are all as
before. There are no new actions.

### Plans

`plans.py` writes `plans/*.json` and `benchmark.json`. A plan is a list of
timed action batches. Every seed gets the same actions at the same
practice-local instant through `World.Apply`. There is no seed-dependent
choice, no static overlay, and nothing the player could not see.

Every plan submits its actions in one atomic batch at the takeover
(Tuesday 2026-10-13 06:00). It uses standard offers only: XR+CT at $38/h,
MR at $45/h, a 28-day recruiting lead and a $3,000 fee per hire.

| Plan | Actions |
|---|---|
| **A — no action** | none |
| **B — partial** (`B-partial`, selected as below; identical to `B1-mr-evening`) | `hire` an MR technologist, 20 h, Mon–Fri 16:00–20:00, start Nov 10; `terminate` Alex (40 h), last day Mon Nov 23 |
| **C — full reorganization** | `hire` a dedicated CT day technologist (40 h, 07:45–16:15 on CT, relieving X-ray's 11:30 break); `hire` an XR/CT evening technologist (27.5 h, Mon–Fri 16:00–20:00 on CT covering X-ray from 16:15, Sat 10:00–18:00); `hire` an MR evening technologist (20 h, Mon–Fri 16:00–20:00), all starting Nov 10; `terminate` Sam (39.5 h, last day Fri Nov 20) and Alex (last day Mon Nov 23); `set_roster` Riley to X-ray only, relieving CT's 12:30 break, effective Mon Nov 23 |

Plan C is the Step 0C lead staffing (`W130-DED-MRpt`) reached legally.
It respects every constraint the brief lists:

* **Recruiting lead:** 28 days, so the earliest start is Nov 10.
* **Training:** 40 attended hours on the hire's own roster. The CT core is
  productive Nov 17, the XR/CT evening Nov 21 and the MR evening Nov 24.
* **Contracts:** the starting contracts are immutable, so Sam's and Alex's
  part-time evenings need a termination plus a new hire, not a roster cut
  (a cut keeps paying the contract).
* **Notice:** at least 14 days, with severance of two contracted weeks plus
  benefits, recognized at acceptance.
* **Headcount:** 8 technologists at the peak (5 + 3), exactly the cap.
* **Roster changes:** Riley's change takes effect on a Monday at least 14
  days ahead, after the CT core is productive.
* **No coverage gap:** each leaver works until the replacement is
  productive.
* **Atomic batch, no approved overtime.**

**Every step of the target staffing is legal; nothing was out of reach.**
The one part of the Step 0C lead that cannot be reached is its reading
allocation (62.5/52.5%), because `set_reading_allocation` is not
implemented. Plan C is therefore `DED-MRpt` at the base allocation.

The partial plan was chosen on development seeds 901–910 among three
candidates, all with standard offers and actions at the takeover:

* **B1:** the MR evening conversion.
* **B2:** Step 0C's cross-trained XR/CT float, 24 h: Monday CT day,
  Tuesday–Friday X-ray evenings, with no termination.
* **B3:** B1 and B2 together.

The rule was written in `plans.py` before any run. Pick the highest mean
T180 cumulative profit over no action, among candidates whose days-91–180
urgent breach rate is at most 5 points above no action's.

| Dev candidate (10 seeds) | Actions | Δ T180 $k | Δ urgent breach rate D91–180 | Eligible |
|---|---|---|---|---|
| B1 MR evening conversion | 2 | +13.0 | +0.6 pts | yes, **selected** |
| B2 float | 1 | +8.0 | +12.3 pts | no |
| B3 float + MR conversion | 3 | +21.0 | +12.4 pts | no |

The float adds imaging, but reading cannot absorb it, the same failure as
C (`results/dev/report.md`). No V0.1 control is used in any plan: the
policies stay at their defaults, so differences come from staffing alone.

### Seeds, horizon and measurement

* **Seeds:** development 901–910 (plan validation and the partial choice);
  held-out evaluation 1001–1020. Neither set was used by Step 0, 0B or 0C
  (seeds 1–20).
* **Horizon:** 180 days from the takeover. Endpoints T30, T90 and T180;
  spans days 31–90 and 91–180.
* **Pairing:** every comparison is against the no-action run of the same
  seed, with a 95% paired t interval.
* **Harness:** `internal/sim/calib_dynamic_test.go` (build tag
  `calibration`). It is read-only and adds a test-only ledger export
  (`calib_export_test.go`). Each day at 06:00 it samples cumulative
  evaluation profit, cash, receivables, liabilities, exams, reports,
  backlog and technologist headcount. At the endpoints it records the
  calibration metrics, displacement, utilization counters, the worklist,
  and a whole-schedule staffing check (no booked appointment may have lost
  its coverage without being displaced).
* **Labor decomposition:** labor is decomposed from the ledger's own
  postings and must equal the window's labor to the cent.
* **Profit basis:** the evaluation window's baseline-adjusted accrual
  operating profit. Recruiting fees and severance are expensed when the
  action is accepted. Cash is reported separately.
* **Payback:** the first day from which the cumulative increment over the
  paired no-action run stays at or above zero through day 180.

Reproduce (about 7 minutes per set on a Raspberry Pi 4, 3–4 on a
Cortex-A76):

    python3 docs/calibration/dynamic-staffing/plans.py
    DT_DYN_SET=dev  go test -tags calibration ./internal/sim -run 'DynamicStaffing$' -v -parallel 3 -timeout 300m
    DT_DYN_SET=eval go test -tags calibration ./internal/sim -run 'DynamicStaffing$' -v -parallel 3 -timeout 300m
    python3 docs/calibration/dynamic-staffing/report.py dev
    python3 docs/calibration/dynamic-staffing/report.py eval
    DT_DYN_SEEDS=901,1001 DT_DYN_DAYS=180 go test -tags calibration ./internal/session -run DynamicPlansReplay -v

## Validation (Phase 2)

On all 30 seeds (development and held out):

* **Actions:** every action was accepted.
* **Lifecycle:** every lifecycle date matched the plan's expectation. The
  hires started Nov 10 and were productive Nov 17, 21 and 24; the last days
  were Nov 20 and 23; Riley's roster version took effect Nov 23.
* **Headcount:** at most 8 technologists.
* **Bookings:** the whole-schedule check found no inconsistent booking at
  any endpoint.
* **Cash:** the cash bridge reconciled at every endpoint.
* **Labor:** labor postings (worked, guaranteed, salaried and severance)
  summed to the window's labor exactly. Recruiting and severance equal the
  fees and two contracted weeks with benefits.
* **Determinism:** each plan replays authoritatively from a saved session,
  whole and in 13-hour chunks, to the same digests at day 180, on seeds 901
  and 1001 (`internal/session/calib_dynamic_test.go`).

**Displacement.** Plan C displaced **one appointment on each of 2 of the 20
held-out seeds** (1006 and 1020), and none on the development seeds. Both
were rebooked the same day, an hour later.

The cause is a booked CT contrast study at 12:30 on a Thursday, during the
CT technologist's break. Riley is rostered to relieve it, but Riley was
also needed in X-ray by another booking (shared capacity). The log names
all three staffing changes as the cause.

Nothing was stranded silently. The finding is that **break-relief covers
between dedicated technologists recreate a small shared-technologist
conflict**.

## Results (20 held-out seeds)

### Finance

Operating profit, $k, mean (range). Δ is against the paired no-action
run, mean ± 95% CI.

| Plan | T30 | T90 | T180 | Days 31–90 | Days 91–180 | Margin days 91–180 |
|---|---|---|---|---|---|---|
| A no action | 0.5 | 3.1 | 3.9 (−8.1 to 15.9) | 2.6 | 0.8 | 0.1% (−1.4 to 1.5) |
| B partial | −7.5 | 1.1 | 16.8 (4.7 to 28.8) | 8.6 | 15.7 | 2.1% (0.6 to 3.4) |
| C full | −18.5 | −7.3 | 18.0 (3.2 to 30.3) | 11.3 | 25.3 | 3.3% (1.0 to 4.9) |
| **Δ B** | −8.0 ± 0.0 | −2.0 ± 0.1 | **+12.9 ± 0.1** | +5.9 | +14.9 | +2.0 pts |
| **Δ C** | −19.0 ± 0.0 | −10.4 ± 1.2 | **+14.1 ± 2.7** | +8.6 | +24.5 | +3.2 pts |

B is ahead of no action on 0/20 seeds at T90 and on 20/20 at T180; C
likewise (0/20 at T90, 20/20 at T180).

The source of the improvement, days 91–180 against no action:

* **B** is a pure labor saving: −$15.0k of wages and benefits, with
  revenue unchanged. The 4-hour MR evening keeps the MR coverage that
  mattered.
* **C** earns from volume:
  * net revenue +$28.1k, against variable cost +$6.5k;
  * labor −$2.9k: the three new contracts (87.5 h at lower rates) cost
    about what Sam's and Alex's 79.5 h did, but there is less paid run-over
    and overtime (173 h against 219 h in the span).

### Transition costs and obligations

Cumulative $k at T180 (all incurred by day 42):

| Plan | Recruiting | Severance (with benefits) | Training (paid hours before productive) | Overlap | Guaranteed pay for time not worked | Total |
|---|---|---|---|---|---|---|
| B | 3.0 | 4.5 | 2.2 | 0.0 | 0.06 | **9.8** |
| C | 9.0 | 8.4 | 6.2 | 0.0 | 0.18 | **23.7** |

**Overlap** counts a leaver's pay on days its designated replacement is
already productive. It is zero because each last day is the eve of the
replacement's first productive day. C still pays Sam from Nov 17 to 20,
alongside the CT core, for coverage the core already gives, but Sam's
designated replacement is the evening hire.

**Guaranteed pay** for contracted time not worked is negligible: rosters
match their contracts. **Severance** is payable on the first payday after
the last day: at T30 it is $4.5k (B) and $8.4k (C), and $0 by T90. Total
liabilities (payables, accrued payroll and severance payable) at T90 and
T180 match no action's within $1.5k.

### Payback (accrual and cash)

| Plan | Steady state Δ / 30 days (days 91–180) | Accrual break-even day, median (range) | Paid back by T90 / T180 | Cash break-even day, median | Cash paid back by T180 |
|---|---|---|---|---|---|
| B | +5.0 ± 0.0 | **101** (100–106) | 0 / 20 of 20 | 114 | 20 of 20 |
| C | +8.2 ± 1.0 | **134** (113–175) | 0 / 20 of 20 | 176 | 11 of 20 |

Mean cumulative increment ($k, accrual / cash), from
`results/eval/report.md`:

| Day | 7 | 28 | 42 | 56 | 70 | 90 | 105 | 120 | 135 | 150 | 165 | 180 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| B | −7.5 / 0 | −7.5 / −3.0 | −9.8 / −3.7 | −7.5 / −9.1 | −5.3 / −6.8 | −2.0 / −2.2 | 0.5 / 0.1 | 3.1 / 2.3 | 5.6 / 4.7 | 8.1 / 6.9 | 10.8 / 9.3 | 12.9 / 11.5 |
| C | −17.4 / 0 | −17.4 / −9.0 | −26.0 / −11.4 | −21.6 / −26.7 | −17.5 / −25.4 | −10.4 / −23.2 | −6.3 / −17.9 | −2.1 / −14.5 | 2.1 / −9.4 | 6.4 / −6.4 | 10.2 / −1.6 | 14.1 / 0.4 |

* **The accrual trough** is at day 42, once the hires are productive:
  −$9.8k for B, −$26.0k for C.
* **Cash** lags accrual. Fees are paid in arrears (about day 20),
  severance on the first payday after the last day (day 49), and C's
  extra volume builds receivables (+$13.6k at T180).
* **Lowest cash** over 180 days: no action $32.8k; B $38.5k; C $20.7k (mean
  of minima $35.9k). Cash is never negative, so there is no financing
  shortfall.
* **Every seed paid back within 180 days**, so no extrapolation is needed.
  For reference, labelled as extrapolation: at the steady-state rates, C's
  cumulative lead over B (+$1.2k at T180) grows by about $3.2k a month.

### Patient service and reporting (Phase 5)

Days 91–180, mean (paired Δ against no action ± 95% CI):

| | A no action | B partial | C full |
|---|---|---|---|
| Exams (XR / US / CT / MR) | 2,082 / 677 / 1,217 / 741 | same | 2,224 / 677 / 1,342 / 741 (**+265 ± 10**) |
| Final reports | 4,705 | +0 ± 1 | **+214 ± 19** |
| Lost referrals (orders received in the span) | 853 | +0 | **−179 ± 9** (XR 99→39, CT 218→99) |
| Walk-ins leaving unseen | 428 | +0 | −90 ± 6 |
| Booking lead p50 XR / CT (days) | 8 / 9 | 8 / 9 | 6 / 7 |
| Urgent / soon / routine breach rate | 10.7 / 38.1 / 7.2% | 10.2 / 37.9 / 7.2% | **52.7 / 88.2 / 55.3%** (+42 ± 13 / +50 ± 6 / +48 ± 13 pts) |
| Urgent / soon / routine report p90 (min) | 140 / 967 / 1,100 | 134 / 968 / 1,098 | **2,900 / 3,286 / 3,597** |
| Backlog at T180, unread + reading (oldest) | 30 (2,132 min) | 30 | **93** (3,173 min); 30 at T90 |
| Radiologist read ÷ site block | 85.5% | 85.6% | 88.9% |
| Technologist busy ÷ duty | 78.0% | 86.3% | 78.9% |

**C's imaging recovery is real.** Unsharing removes most shared-technologist
losses (XR 71 → 19, CT 139 → 20 per 90 days), and Monday evening CT becomes
staffed.

**Reading is now the bottleneck.** C's reports trail its exams (+214
against +265), so the backlog keeps growing. Urgent studies acquired after
the site reading blocks wait for the next day. This matches Step 0C's
static `DED-MRpt` (urgent breach 29% at T90); over 180 days it compounds.
The unreported backlog at T180 holds about $10k of revenue not yet
recognized (63 extra studies at ≈$159 a report).

**B changes no measured service figure.**

**Neither staffing plan reaches the provisional service gates.** These
gates (urgent ≤ 12%, soon ≤ 32%, routine ≤ 6%) were frozen from Step 0C's
lead candidate with reading restored. No action itself misses the soon
and routine gates (38% and 7.2%), since those depend on reading coverage.

### Why the results are what they are

| Cause | B | C |
|---|---|---|
| Transition expense | $9.8k, the whole T90 deficit | $23.7k, more than the T90 deficit |
| Delayed productive start | no saving before day 42 (Alex is paid until Nov 23) | no benefit before day 35–42; days 1–41 carry the old contracts plus trainees |
| Excess guaranteed labor | negligible ($60) | negligible ($180) |
| Unrecovered demand | unchanged: US (≈405 lost referrals per 90 days, "not rostered" after 18:00 and on Morgan's alternate Mondays) is the largest single loss, outside both plans and Step 0C's architecture | US unchanged. CT category-mix losses rise (20 → 45) once technologists stop binding: the CT template becomes the next limit |
| Reading capacity | not binding | binding: +51 studies a quarter left unread; service collapses; revenue is deferred |

## Decision

**CONDITIONAL GO.** Staffing actions alone demonstrate player agency with
a credible financial trajectory:

* **Spread:** the three plans span $13–14k of T180 profit and $5–8k a
  month in steady state (2–3 margin points) on the same seeds.
* **Payback:** every reorganization pays back within six months on every
  held-out seed.
* **Transitions:** neither too slow nor too expensive. Everything is in
  place by day 42, and the costs are $10–24k, so the starting state does
  not need revisiting.

**The service condition fails for the larger reorganization.** The
operational improvement (the full plan's imaging and access gains) breaks
report turnaround, which only reading actions can address. The partial
plan keeps service but is purely a cost saving.

Per the owner's plan, proceed with **`set_reading_allocation` (Slice 4)**.
Then rerun these same frozen plans on the same held-out seeds, adding
**C + an explicitly defined reading adjustment**, for example 62.5/52.5%
from its earliest legal effective date (Step 0C's `A62`). That isolates the
value of radiologist management before teleradiology.

Do not tune the plans further on seeds 1001–1020.

## Plan D: full staffing with reading allocation (Slice 4, 0.2.0-dev.5)

The owner's follow-up after Slice 4 (`set_reading_allocation`, V0.2_DESIGN
§5.10): rerun the frozen plans on the same held-out seeds and add plan C
with a defined reading adjustment. This section's method and decision rule
were written on 2026-10-16, before any run of plan D. The original plans,
seeds, results and conclusions above are unchanged.

### Method (fixed before any run)

* **Plan D (`plans/D-full-reading.json`)** is plan C's six staffing actions,
  unchanged, in the same takeover batch. Two allocation changes follow
  them: Dr. Avery (E07) 60% → 62.5% and Dr. Blake (E08) 50% → 52.5%. These
  are the Step 0C lead's `A62` levels.
* **Effective date: Tuesday 2026-10-27.** It is the earliest legal date:
  an allocation change needs 14 days' notice, on any weekday, from the
  takeover on Tuesday 2026-10-13.
* **No tuning.** D is not adjusted on any result. If it does not restore
  acceptable reporting, no other allocation is tried on seeds 1001–1020;
  that belongs to a separate development-seed experiment.
* **Validation.** D was run on development seeds 901–910 (`reading-dev`)
  with A and C, for acceptance and lifecycle only; nothing was chosen.
* **Evaluation.** A, B, C and D ran on held-out seeds 1001–1020
  (`reading-eval`) for 180 days under 0.2.0-dev.5. A, B and C must
  reproduce the original `results/eval` outcomes exactly: no allocation
  action, no change. D is compared with C by paired difference.
* **Measured:** incremental reading expense (radiologist salary), final
  reports, revenue held in the reporting backlog (imaged, not yet billed),
  breach rates, backlog size and age, imaging and losses, cumulative
  operating profit, lowest cash, liabilities, and accrual and cash payback
  against no action.

### Decision rule (fixed before any run)

"Acceptable reporting" means two things, judged on means across seeds:

* the owner's provisional gates (§13.3: urgent ≤ 12%, soon ≤ 32%,
  routine ≤ 6% breach rate) are met over T90 and over days 91–180;
* the reporting backlog is not growing: at T180 it is at most 10% above
  its T90 level.

No action itself misses the soon and routine gates, so the paired
difference against no action is reported alongside.

* **GO:** D's reporting is acceptable, and D keeps the staffing plan's
  economic benefit: its T180 cumulative profit over no action is positive,
  with a 95% interval above zero.
* **CONDITIONAL GO:** D significantly improves reporting over C (the
  paired urgent-breach reduction has a 95% interval below zero) and keeps
  a positive benefit, but at least one gate or the backlog condition
  fails. More reading capacity or teleradiology is needed.
* **NO GO:** D does not significantly improve reporting over C, or its
  cost removes the economic benefit (T180 profit over no action ≤ 0).

### Validation

* **Development seeds 901–910 (`results/reading-dev/`):** D's eight
  actions were accepted on every seed, and every lifecycle date and
  allocation version matched the plan. A and C reproduced the original
  development results exactly. Nothing was chosen.
* **Held-out seeds (`results/reading-eval/`):**
  * **A, B and C** under 0.2.0-dev.5 reproduce `results/eval` exactly,
    in every per-seed trajectory, window, span, labor figure and check,
    apart from the engine name.
  * **Seed 1001 sessions:** the session replay check gives the same run
    digests as the committed 0.2.0-dev.4 session files.
  * **D** was accepted on every seed, with its lifecycle and allocations
    as planned. It displaced the same two appointments as C, both
    rebooked. Every endpoint reconciled to the cent, and no booking was
    inconsistent.
  * **Replay:** every plan, D included, saves and replays whole and in
    13-hour chunks to identical digests (seeds 901 and 1001, 180 days).
    `sessions/D-full-reading-seed1001.session.json` is D on seed 1001.

### Results (20 held-out seeds, D − C paired unless stated)

Generated tables: `results/reading-eval/reading.md` (D against C and no
action) and `results/reading-eval/report.md` (all four plans, as for the
original evaluation).

| | T30 | T90 | days 91–180 | T180 |
|---|---|---|---|---|
| Reading expense (radiologist salary with benefits), $k | +1.4 | +6.3 | +7.4 | +13.7 |
| Final reports | +5 ± 2 | +18 ± 10 | +55 ± 22 | +74 ± 20 |
| Net revenue, $k | +0.9 ± 0.4 | +3.2 ± 1.8 | +9.4 ± 3.8 | +12.6 ± 3.3 |
| Operating profit, $k | −0.5 ± 0.4 | −3.3 ± 1.7 | +1.8 ± 3.7 | −1.5 ± 3.2 |
| Revenue held in the backlog at the endpoint (C → D), $k | 3.6 → 2.7 | 4.4 → 1.3 | — | 14.9 → 2.4 |

Imaging, access, losses and displacement are identical to C's (Δ 0 on
every seed), as designed: an allocation changes reading only.

**Reporting service** (breach rate urgent / soon / routine, %):

| | T90 | days 91–180 | backlog T90 / T180 | seeds meeting all three gates, T90 / days 91–180 |
|---|---|---|---|---|
| A, no action | 11.8 / 36.4 / 7.0 | 10.7 / 38.1 / 7.2 | 18 / 30 | 0 / 1 of 20 |
| C, full staffing | 20.1 / 61.1 / 16.4 | 52.7 / 88.2 / 55.3 | 30 / 93 | 0 / 0 of 20 |
| **D, full staffing + reading** | **11.3 / 31.6 / 5.1** | **9.2 / 31.9 / 4.4** | 12 / 20 | 7 / 11 of 20 |

Against C, D cuts the breach rates by 43, 56 and 51 points in days
91–180. Against no action, D is better on every rate: urgent −1.4 ± 2.2,
soon −6.2 ± 3.8 and routine −2.8 ± 0.4 points. The urgent p90 TAT is 150
minutes, against 2,900 for C. The backlog's p90 age at T180 is 816
minutes, against 2,638 for C and 1,781 for no action.

**Economics against no action:**

| | Δ profit T30 / T90 / T180, $k | steady state / 30 d | accrual break-even | cash paid back by T180 | lowest cash (mean, min) | margin days 91–180 |
|---|---|---|---|---|---|---|
| C | −19.0 / −10.4 / +14.1 ± 2.7 | +8.2 ± 1.0 | day 134 (113–175), 20/20 | 11 of 20 | 35.9, 20.7 | 3.3% |
| **D** | −19.5 / −13.7 / **+12.6 ± 1.4** | **+8.8 ± 0.6** | **day 141 (113–164), 20/20** | **1 of 20** | 33.3, 18.2 | 3.5% |
| D − C (the allocation alone) | −0.5 / −3.3 / −1.5 ± 3.2 | +0.6 ± 1.2 | — | — | — | — |

* **Cost of the allocation:** the 5 extra points cost $2.47k per 30 days
  with benefits, $13.7k by T180. Radiologist salary rises from $54.2k to
  $56.7k per 30 days.
* **Payback:** the allocation roughly pays for itself in steady state. It
  recovers the revenue C leaves in the backlog ($12.5k at T180) and adds
  55 reports per 90 days. Over 180 days it is not significantly different
  from C (−1.5 ± 3.2 $k).
* **Cash:** cash payback is later. The salary is paid every payday, while
  the revenue it brings forward is collected only after the claim and
  payment lags: D's lowest cash is $2.6k below C's.
* **Obligations:** liabilities are at C's level ($83k at T180). The
  allocation is a recurring commitment of about $2.5k per 30 days until
  changed, on 14 days' notice.

### Decision

**Rule outcome: CONDITIONAL GO.** The rule is the one fixed before any
run, applied as written:

* Gates met (means) over T90 and days 91–180: **yes**.
* T180 profit over no action positive with a 95% interval above zero:
  **yes**, +12.6 ± 1.4 $k.
* Significant urgent-breach reduction against C: **yes**.
* Backlog not growing (T180 at most 1.1 × T90): **no**, 12 → 20.

The failing condition is the backlog sample. The rule compares single
06:00 samples on day 90 (a Monday) and day 180 (a Sunday), and no action
fails the same test (18 → 30). In 30-day blocks D's mean daily backlog is
flat, below no action's in every block; C's grows from 35 to 85 (see
`reading.md`, "Supplementary"). The rule outcome stands, but the reporting
evidence is that `set_reading_allocation` restores service to the
provisional gates.

The remaining conditions are real:

1. **The gates are met on the mean only.** All three are met on 7 of 20
   seeds over T90 and 11 of 20 over days 91–180. The soon breach rate
   (31.6–31.9%) sits at its 32% gate.
2. **The allocation does not pay for itself.** It trades about $2.5k a
   month of salary for restored turnaround and the revenue that comes
   with it. D's profit advantage over no action is C's, not more. On cash,
   D pays back by T180 on 1 of 20 seeds.
3. **There is no headroom for growth.** More imaging, or a higher
   allocation, raises salary in 2.5-point steps of $1.2k a month per
   radiologist. Teleradiology's per-read pricing is the untested
   alternative for the marginal and evening reads.

Per the rule, no other allocation was tried on seeds 1001–1020. Whether
a higher allocation, or teleradiology instead of part of it, gives
better service per dollar is a separate development-seed experiment.

## Limitations

* **One hand-authored plan per strategy, not optimized.**
  * Premium offers, staggered start dates and other partial plans (e.g. a
    part-time US evening for the US losses) were not tested.
  * The partial choice used 10 development seeds and a pre-declared rule.
* **No reading action in A–C.** Plan C is the Step 0C lead without its
  reading allocation, so its service result is not the lead candidate's;
  plan D adds it (Slice 4).
* **Plan D tries one allocation.** The design fixed it in advance and
  nothing was tuned. The backlog test in its decision rule uses
  single-day samples.
* **The model has no recruiting uncertainty.** An accepted hire always
  starts and trains on schedule; no candidate declines and no one quits.
  Real transitions would be slower and riskier.
* **Demand is the documented ×1.30 and stationary.** The ×1.15 sensitivity
  (where Step 0C lost money on every architecture) was not rerun
  dynamically.
* **Horizon.** 180 days, with nothing extrapolated except the labelled
  comparison. The full plan's backlog was still growing at T180.
* **Accrual and cash timing.** Accrual profit expenses severance and fees
  at acceptance; revenue is recognized when a report is final, so C's
  reading backlog defers revenue.
* **Overlap definition.** Overlap counts only a designated replacement.
  C's CT core overlapping Sam's midday for four days is in labor, not in
  overlap.
* **Platforms.** Results were produced on two arm64 machines (Pi 4 and a
  Cortex-A76) and compared across them; amd64 was not run.

## Files

* **Inputs:** `plans.py` (plans, `benchmark.json`, selection rule),
  `plans/*.json` (the frozen plans; `B-partial` = `B1-mr-evening`
  renamed), `benchmark.json`.
* **Tools:** `report.py` (tables, payback, `trajectory.csv`);
  `report_reading.py` (plan D against C and no action, the reproduction
  check against the original results, the decision rule).
* **Sessions:** `sessions/`, the evaluation plans on seed 1001 played to
  day 180 as session files (A–C written by 0.2.0-dev.4, which this build
  replays with every digest verified; D by 0.2.0-dev.5). `dreamteam verify FILE` replays one;
  `dreamteam play -load FILE` opens it in the TUI to inspect the staff,
  rosters, worklist and finances.
* **Results:** `results/eval/` and `results/dev/` (original, 0.2.0-dev.4);
  `results/reading-eval/` and `results/reading-dev/` (with plan D,
  0.2.0-dev.5, plus `reading.md`). Each holds per-plan
  result JSON (provenance, every action's result, lifecycle, daily
  trajectories, endpoint windows, spans, labor decomposition, checks),
  `report.md` and `trajectory.csv` (per seed and day: cumulative
  incremental profit and cash).
