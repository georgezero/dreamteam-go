# V0.6c: conservative v2 and the frozen-agent comparison

The two phases of [PLAN.md](PLAN.md) (`v06c-plan/1`, committed in 4b9ed5c
before any run on its seeds, with no amendment):
* **Phase 1:** qualification of `conservative-v2`
  ([BENCHMARK_SUITE.md](../../BENCHMARK_SUITE.md) §7.2);
* **Phase 2:** the first standardized comparison of two frozen LLM agents
  with the deterministic baselines, on the five-scenario suite.

**Not a ranking.** There is no score, leaderboard or significance test, and
no claim beyond these seeds, models and settings. "Appropriate" and
"disproportionate" are the owner's to judge; this document gives the
figures and the agents' recorded actions and reasons. It never infers a
model's reasoning from an outcome.

## Files

| File | What it is |
|---|---|
| `PLAN.md` | The plan, as declared |
| `qualify.json`, `compare-30.json`, `compare-90.json` | The suite-run configurations, as run |
| `analyze.py` | The tables of PLAN §4 from a results directory (declared with the plan) |
| `breaches.py` | `conservative-v2`'s daily breach records and money conditions, read back from its own requests in the transcripts (PLAN §2) |
| `<stage>/contract.txt`, `contract.json` | The contract report the runner wrote |
| `<stage>/report.txt`, `report.json`, `manifest.json` | The harness's report and the manifest |
| `<stage>/tables.md` | `analyze.py`'s output |
| `<stage>/breaches.md` | `breaches.py`'s output |
| `compare-*/<entry>-<model>-s<seed>-decisions.jsonl` | Each LLM run's recorded decisions (the agent's `policy-evidence/decisions.jsonl`): prompt and memory hashes, every attempt, refusal and error, the reason, and the batches submitted |
| `compare-*/<entry>-<model>-s<seed>-usage.json` | Each LLM run's usage summary: requests, retries, tokens, latency |
| `compare-*-replay-<entry>.json` | The replay configurations, one per stage and entry |

Kept outside the repository: the run directories (qualification 153 MB,
30-day 316 MB) and the model exchanges (`policy-evidence/model.jsonl`).

## 1. Phase 1: `conservative-v2` qualification

### 1.1 Synthetic qualification

Done before the plan was committed:
`internal/bench/conservative2_test.go` passes all eight requirements of
§7.2 with synthetic public observations served by a fake protocol
endpoint. It also passes the review, counting and real-log tests.

### 1.2 Development check

* **Run:** [qualify.json](qualify.json): `noop` and `conservative-v2`, the
  five entries, seed 41101, 90 days, T90 for reporting backlog; 10 runs
  in 13 minutes on the Pi.
* **Status:** complete, 10 of 10.

**What it did.** It acted in one entry, stable, three times:

| Decided | Change | Effective | Why (by its rules) |
|---|---|---|---|
| 2026-10-19 | Dr. Blake (E08) 52.5% → 57.5% | 2026-11-02 | Breaches sustained, trailing profit ≥ 0, affordable after the preview's +$2,466 per 30 days |
| 2026-11-16 | E08 57.5% → 52.5% | 2026-11-30 | The review after 14 days in effect: breaches fell from about 2.8 to 1.0 per day (the raise helped), but the trailing-30-day profit was −$7,123, so it stepped back |
| 2026-12-16 | E08 52.5% → 57.5% | 2026-12-30 | Sustained again, profit +$4,063, affordable |

**Paired with no action** (stable, seed 41101, 90 days): profit −$3,370
(expenses +$3,370), TAT p90 −49 minutes, breaches −12, lost orders 0. In
the other four entries it was identical to no action.

**Why it did not act elsewhere** ([qualify/breaches.md](qualify/breaches.md),
read back from its own observations):

| Entry | Sustained problem (of 84 points with a week's history) | Trailing-30-day profit below zero (of 90) | Every raise condition met | Outcome |
|---|---|---|---|---|
| financial pressure | 28 | 90 | 0 | no action |
| reporting backlog | 57 | 86 | 0 | no action |
| demand growth | 33 | 79 | 2 | previewed twice; declined by the affordability check (trailing profit $2,039 and $1,079, less the preview's +$2,466 per 30 days) |
| staffing shortage | 6 | 90 | 0 | no action |
| stable | 50 | 43 | 25 | the three changes above |

So its service signal now sees sustained problems in every entry (version
1 saw none). What stops it is the financial guardrail: in four of five
entries the trailing 30 days lose money nearly all the time. That is
the policy as specified, not a defect. Nothing was tuned after the run,
and no version 3 was made.

**Integrity** (linux/amd64, horo): `verify -t90` 108 of 108; `reproduce`
73 of 73, manifest byte-identical; a fresh run's manifest equal to the
Pi's.

## 2. Phase 2: the 30-day comparison

* **Run:** [compare-30.json](compare-30.json): five entries, seeds
  41201–41203, 30 days, `noop`, `greedy`, `conservative-v2`,
  `qwen36-35b-a3b`, `gemma4-26b-a4b`. 75 runs in 2 h 50 min on the Pi
  (limit 6 h). The LLM runs took 70 minutes (Qwen) and 80 minutes (Gemma)
  in all.
* **Status:** complete, 75 of 75. No run failed, no infrastructure error
  (HTTP, connection or timeout), no rerun. The contract report opens
  `complete: 75 of 75 runs`.
* **Starting state** identical across the five policies for every entry
  and seed (15 of 15).

### 2.1 Outcomes, paired with no action

Policy minus `noop`, same entry and seed, for seeds 41201, 41202, 41203 in
that order. Every absolute value is in [compare-30/tables.md](compare-30/tables.md)
and [compare-30/contract.txt](compare-30/contract.txt).

| Entry | Policy | Seeds acting | Δ operating profit | Δ breaches | Δ TAT p90 (min) | Δ lost orders |
|---|---|---|---|---|---|---|
| financial pressure | greedy | 3 of 3 | −$1,175, −$635, +$425 | −1, −4, −5 | −26, −30, −34 | 0, 0, 0 |
| | conservative-v2 | 0 of 3 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 |
| | Qwen | 3 of 3 | $0, +$1, −$1,483 | −7, −2, −7 | −18, −3, −20 | 0, 0, −4 |
| | Gemma | 1 of 3 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 |
| reporting backlog | greedy | 3 of 3 | −$975, −$676, +$237 | −14, −27, −3 | −52, −187, −5 | 0, 0, 0 |
| | conservative-v2 | 1 of 3 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 |
| | Qwen | 3 of 3 | +$1,070, −$16,102, $0 | −29, −70, −5 | −83, −229, −17 | −6, 0, 0 |
| | Gemma | 0 of 3 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 |
| demand growth | greedy | 3 of 3 | −$532, −$470, +$389 | −11, −23, −5 | −61, −206, −11 | 0, 0, 0 |
| | conservative-v2 | 1 of 3 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 |
| | Qwen | 3 of 3 | −$3,471, −$1,435, +$142 | −68, −80, +25 | −199, −263, +108 | 0, 0, −19 |
| | Gemma | 1 of 3 | −$3,074, 0, 0 | −41, 0, 0 | −160, 0, 0 | 0, 0, 0 |
| staffing shortage | greedy | 3 of 3 | −$1,144, −$689, −$925 | 0, −3, −2 | −10, −28, −18 | 0, 0, 0 |
| | conservative-v2 | 0 of 3 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 | 0, 0, 0 |
| | Qwen | 3 of 3 | −$7,683, −$30,447, −$28,564 | −25, −31, −7 | −51, −554, −36 | −16, −8, −23 |
| | Gemma | 3 of 3 | +$1,169, −$1,391, −$573 | +3, −32, −12 | +10, −576, −17 | −12, −16, 0 |
| stable | greedy | 3 of 3 | −$1,636, +$4, +$694 | −4, −2, −7 | 0, −12, −62 | 0, 0, 0 |
| | conservative-v2 | 2 of 3 | −$567, +$662, 0 | −3, −2, 0 | 0, −12, 0 | 0, 0, 0 |
| | Qwen | 3 of 3 | −$160, −$635, −$2,923 | −8, −14, −38 | −24, −55, −655 | −2, 0, 0 |
| | Gemma | 2 of 3 | −$627, −$660, 0 | −18, −26, 0 | −9, −32, 0 | 0, 0, 0 |

"Seeds acting" counts runs with at least one accepted change. A change
effective after the 30th day is counted but has no effect in the window:
* `conservative-v2` in reporting backlog and demand growth, seed 41201:
  raised on 2026-11-06, effective 2026-11-20;
* Gemma in financial pressure, seed 41202: teleradiology from 2026-11-15.

Radiologists are salaried (no overtime pay,
[SIMULATION_DESIGN.md](../../SIMULATION_DESIGN.md)), so extending a
radiologist's shift adds overtime minutes but no expense. That is why
Qwen's financial-pressure runs show service changes with little or no
profit change.

**`conservative-v2`** ([compare-30/breaches.md](compare-30/breaches.md)):
* **Stable:** it acted on seeds 41201 and 41202, each a single raise of
  Dr. Blake to 57.5%.
* **Seed 41203:** it previewed eleven raises and declined each by the
  affordability check.
* **Reporting backlog and demand growth:** it raised once (seed 41201,
  too late to take effect) and declined others by the same check.
* **Financial pressure and staffing shortage:** the trailing profit was
  negative on all 30 points, so it never reached a preview.

### 2.2 LLM agent behaviour

Totals over 15 runs each (450 decision points per model); per run in
[compare-30/tables.md](compare-30/tables.md).

| | Qwen3.6-35B-A3B | Gemma 4 26B-A4B |
|---|---|---|
| Points with actions | 108 | 25 |
| No action | 329 | 412 |
| Fallback (no valid decision after the retry; no action taken) | 13 | 13 |
| First attempt valid | 383 | 423 |
| Changes accepted | 189 | 32 |
| Refusals by `validate` | 116 | 34 |
| Agent errors: refused / invalid decision / invalid JSON / invalid action | 68 / 9 / 2 / 1 | 28 / 1 / 11 / 0 |
| Infrastructure errors | 0 | 0 |
| Details requests | 0 | 0 |
| Requests (retries) | 517 (67) | 477 (27) |
| Prompt / completion tokens | 6,314,023 / 121,230 | 5,272,023 / 43,547 |
| Latency: mean per request, maximum | 6.9 s, 52.4 s | 8.7 s, 28.0 s |
| Estimated cost | none configured | none configured |

**Refusals.** Most of Qwen's were `extend_shift` refusals:
* `after_close`: extending a shift that already ends at the close;
* `exceeds_max_shift`;
* `not_on_shift`;
* `no_coverage`;
* `invalid_minutes`.

It also had `hire lead_too_short`, and teleradiology overlap and schema
errors.

Gemma's came mostly in staffing shortage. On seed 41202 the refusals were
`hire lead_too_short` four times and `invalid_roster` five times, with 7
fallbacks. On seed 41203 they were teleradiology overlap and batch
rejections, with 6 fallbacks and 8 invalid-JSON errors.

**Repeated invalid proposals** (the same type and code seen earlier in the
run): Qwen up to 8 in a run, Gemma up to 11. The per-run figures are in
the tables.

### 2.3 The six questions

Answered from the outcomes and the agents' recorded actions and reasons
(`compare-30/*-decisions.jsonl`). Quotations are the agents' own `reason`
fields.

**1. Does an agent act appropriately under financial pressure?**
* **`conservative-v2`:** no action. The trailing profit was negative
  throughout.
* **Gemma:** acted once in three runs: a Sunday teleradiology contract
  from 2026-11-15, outside the window. Its recurring no-action reason
  named the backlog and then deferred to the next radiologist: "Dr.
  Avery starts in 30 minutes (07:30), which should address the immediate
  backlog … so no immediate action is required". On seed 41201 it also
  said "no immediate action is taken to avoid unnecessary costs".
* **Qwen:** extended radiologist and technologist shifts on 3–10 days per
  run (31 accepted extensions). Its reasons cite the loss ("The practice
  is facing significant financial losses (-$18k MTD) …").
  * On seed 41203 it described the practice as closed and losing money,
    and extended three or four staff by 210 minutes a day "to reopen and
    generate revenue". It did this although it was the 07:00 decision
    point before opening.
  * Paired result: profit −$1,483, breaches −7, lost orders −4, overtime
    +3,245 minutes.
* **Greedy:** cost $635–$1,175 on two seeds and gained $425 on one.

**2. Does it reduce the reporting backlog?**
* **Qwen:** reduced breaches on all three seeds (−29, −70, −5).
  * On seed 41202 it set both radiologists' site allocation to 100% from
    2026-11-02 and contracted teleradiology every night and weekend.
  * That run's result: breaches −70, TAT p90 −229 minutes, profit
    −$16,102.
* **Gemma:** did nothing in any run. Its reasons acknowledge growing
  backlogs ("a significant reading backlog of 30 studies, with 30 overdue
  reports") and then expect the next scheduled block to address them.
* **`conservative-v2`:** one raise, too late to take effect.
* **Greedy:** reduced breaches by 3–27 at a cost of up to $975.

**3. Does it recognize imaging-capacity limits under demand growth?** In
V0.6b, added demand was lost, not imaged.
* **Qwen:** its reasons name lost demand at 7–12 decision points per run.
  On seed 41203 it extended technologist shifts on most days, giving as the reason
  that it "increases imaging throughput to prevent further backlog growth
  and capture lost demand", while noting that this
  "does not resolve the reading bottleneck".
  * That run's result: exams +29, lost orders −19, overtime +2,852
    minutes, profit +$142, but breaches +25 and TAT p90 +108 minutes.
  * On the other two seeds it targeted reading (shift extensions,
    teleradiology), and lost orders were unchanged.
* **Gemma:** mentioned lost demand once in the three runs. It contracted
  Sunday teleradiology on seed 41201 (breaches −41, profit −$3,074), and
  nothing else.
* **The baselines** use only the reading allocation and cannot affect
  imaging capacity.

**4. Does it address the staffing shortage?**
* **Qwen:**
  * hired technologists on all three seeds (`hire` accepted 1–2 times,
    premium lead times);
  * set both radiologists' allocations to 100% (seeds 41202 and 41203),
    contracted teleradiology on all three, and extended shifts on most
    days.
  * Lost orders fell (−16, −8, −23) and breaches fell (−25, −31, −7).
  * Profit fell by $7,683, $30,447 and $28,564; expenses rose $13,287 to
    $37,361.
* **Gemma:**
  * extended technologist shifts on two seeds;
  * contracted teleradiology on two;
  * made repeated hire attempts on seed 41202, all refused for lead time
    or roster errors.
  * Profit changed by +$1,169, −$1,391 and −$573. On seed 41202 breaches
    fell by 32 and TAT p90 by 576 minutes.
* **`conservative-v2`:** no action (loss throughout).

**5. Does it avoid unnecessary intervention in the stable practice?**
* **No action:** the stable practice makes $6,100–$8,900 in 30 days.
* **Qwen:** intervened on every seed: 5, 7 and 11 decision points with
  actions, mostly shift extensions (radiologists and technologists),
  with one teleradiology contract.
  Profit fell on all three (−$160, −$635, −$2,923) while breaches fell
  (−8, −14, −38).
  * One recorded reason does not match the decision: "I will reduce Dr.
    Avery's site allocation to the minimum (3000 BP) to lower fixed salary
    costs, and extend Casey's shift". Only the extension was proposed.
* **Gemma:** a Sunday teleradiology contract on two seeds (profit −$627,
  −$660; breaches −18, −26), nothing on the third.
* **`conservative-v2`:** raised once on two seeds (−$567 and +$662).
* **Greedy:** raised four times on every seed.

**6. Are service improvements accompanied by disproportionate financial
costs?** The figures; the judgement is the owner's.
* **Flag:** `analyze.py` flags "better service, lower profit" on 26 of the
  60 paired runs. Every LLM run with a profit loss over $3,000 is one of
  them:
  * Qwen reporting backlog 41202: −$16,102 for −70 breaches;
  * Qwen staffing shortage, all three seeds: −$7,683 to −$30,447 for −7
    to −31 breaches and −8 to −23 lost orders;
  * Qwen demand growth 41201: −$3,471 for −68 breaches;
  * Gemma demand growth 41201: −$3,074 for −41 breaches.
* **Cost per breach avoided** in the 15 flagged LLM runs ranges from
  about $18 (Qwen, demand growth 41202) to about $4,081 (Qwen's hires in
  staffing shortage, 41203). The median is $51.
* **Without a profit loss:** Qwen reduced breaches in reporting backlog
  41201 (−29, profit +$1,070) and in financial pressure (radiologist
  overtime, unpaid).
* **For comparison:** greedy's largest cost was $1,636 and
  `conservative-v2`'s $567.

## 3. Phase 2: the 90-day stage

* **Run:** [compare-90.json](compare-90.json): the five entries, seed
  41301 (declared in the plan), 90 days, the same five policies, T90 for
  reporting backlog. 25 runs in 4 h 21 min on the Pi (limit 8 h). The LLM
  runs took 16–30 minutes each (Qwen 16–26, Gemma 27–30).
* **Status:** complete, 25 of 25. No run failed, no infrastructure error,
  no rerun. Starting state identical across policies, 5 of 5.
* **T90** (reporting backlog): NO GO for all five policies.

### 3.1 Outcomes, paired with no action

One seed, so one value per cell. Absolute values in
[compare-90/tables.md](compare-90/tables.md).

| Entry | Policy | Changes accepted | Δ operating profit | Δ breaches | Δ TAT p90 (min) | Δ lost orders | Δ overtime (min) |
|---|---|---|---|---|---|---|---|
| financial pressure | greedy | 12 | −$32,549 | −51 | −119 | 0 | 0 |
| | conservative-v2 | 0 | 0 | 0 | 0 | 0 | 0 |
| | Qwen | 56 | −$43,934 | −76 | −107 | +361 | +3,306 |
| | Gemma | 0 | 0 | 0 | 0 | 0 | 0 |
| reporting backlog | greedy | 12 | −$33,940 | −139 | −178 | 0 | 0 |
| | conservative-v2 | 0 | 0 | 0 | 0 | 0 | 0 |
| | Qwen | 19 | +$1,284 | +11 | +17 | −14 | +1,686 |
| | Gemma | 0 | 0 | 0 | 0 | 0 | 0 |
| demand growth | greedy | 12 | −$27,266 | −267 | −423 | 0 | 0 |
| | conservative-v2 | 1 | +$2,191 | −5 | −48 | 0 | 0 |
| | Qwen | 36 | −$2,260 | −5 | +2 | −16 | +3,372 |
| | Gemma | 1 | +$2,098 | −97 | −156 | 0 | 0 |
| staffing shortage | greedy | 12 | −$32,591 | −42 | −96 | 0 | 0 |
| | conservative-v2 | 0 | 0 | 0 | 0 | 0 | 0 |
| | Qwen | 61 | −$137,507 | −67 | −630 | −113 | +3,278 |
| | Gemma | 0 | 0 | 0 | 0 | 0 | 0 |
| stable | greedy | 12 | −$32,655 | −99 | −252 | 0 | 0 |
| | conservative-v2 | 2 | −$2,301 | −53 | −117 | 0 | 0 |
| | Qwen | 16 | −$58,693 | −206 | −832 | +21 | +1,037 |
| | Gemma | 0 | 0 | 0 | 0 | 0 | 0 |

Over 90 days, greedy's twelve raises (to the cap of 100%) cost $27,000–
$34,000 in every entry, against $500–$1,700 in 30 days.

**`conservative-v2`** ([compare-90/breaches.md](compare-90/breaches.md)):
* **Stable:** raised Dr. Blake to 57.5% on 2026-10-29 (effective
  11-12), then retreated to 52.5% at the review on 11-26 (effective 12-10).
  Paired: −$2,301, breaches −53, TAT p90 −117 minutes.
* **Demand growth:** one raise on 2026-12-22 (effective 2027-01-05).
  Paired: +$2,191, breaches −5.
* **The other three entries:** no action. The trailing-30-day profit was
  negative at 81–90 of the 90 points, and every raise condition was met
  on at most 3 points.
* This is the run the 90-day stage was declared for: a raise took effect
  and was reviewed, and the review retreated.

### 3.2 LLM agent behaviour

Totals over 5 runs each (450 decision points per model); per run in
[compare-90/tables.md](compare-90/tables.md).

| | Qwen3.6-35B-A3B | Gemma 4 26B-A4B |
|---|---|---|
| Points with actions | 97 | 1 |
| No action | 345 | 445 |
| Fallback | 8 | 4 |
| First attempt valid | 398 | 445 |
| Changes accepted | 188 | 1 |
| Refusals by `validate` | 89 | 2 |
| Agent errors: refused / invalid decision / invalid JSON | 54 / 6 / 0 | 2 / 1 / 6 |
| Infrastructure errors | 0 | 0 |
| Requests (retries) | 502 (52) | 455 (5) |
| Prompt / completion tokens | 6,412,170 / 101,575 | 4,877,684 / 33,570 |
| Latency: mean per request, maximum | 10.6 s, 48.6 s | 17.3 s, 31.7 s |
| Estimated cost | none configured | none configured |

The refusal pattern is the 30-day one: Qwen's mostly `extend_shift`
`after_close`, `exceeds_max_shift` and `not_on_shift`, plus
`hire lead_too_short`. It repeated an invalid proposal up to 18 times in a
run (staffing shortage).

### 3.3 What the longer horizon shows

From the outcomes and the recorded actions and reasons
(`compare-90/*-decisions.jsonl`):
* **Gemma acted once in 450 decision points:** a Sunday teleradiology
  contract in demand growth (decided 2026-11-30, effective 12-14, after
  "a significant reading backlog of 33 studies, with 33 overdue
  reports"). That run's breaches fell by 97 with profit +$2,098. In the
  other four entries it proposed no change; its only failures
  were two fallbacks after invalid JSON in staffing shortage.
* **Qwen in staffing shortage** made the largest intervention of either
  stage:
  * 53 shift extensions, three technologist hires, weeknight
    teleradiology, two roster changes and two allocation changes;
  * TAT p90 fell from 761 to 131 minutes and lost orders by 113;
  * expenses rose $160,276 and profit fell $137,507;
  * its reasons from 2026-12-01 note "Cash is critically low ($3,005)",
    then "$20,292" and "$26,714", and it continued the extensions.
* **Qwen in financial pressure** terminated a technologist ("terminating
  the most expensive technologist (Alex, E11, MR-only)") and changed
  another technologist's roster on 2026-11-03. Lost orders rose by 361 and revenue fell by
  $63,378.
  * On 11-17 and 11-18 its reasons call the teleradiology retainer
    "$150k/month". The observation gives `monthlyRetainerCents` 150000,
    that is $1,500.
  * The 11-17 reason says "I will cancel the teleradiology contract";
    the accepted change that day was a shift extension only.
* **Qwen in the stable practice:** three allocation changes, two roster
  changes, three teleradiology actions (two starts and a cancellation) and
  eight extensions. TAT p90 fell from 956 to
  123 minutes and breaches by 206, for −$58,693. The practice's no-action
  profit over 90 days is $31,025.
* **Qwen in reporting backlog:** 19 shift extensions, with profit
  +$1,284, breaches +11 and T90 NO GO.

The 30-day patterns of §2.3 recur on this seed: Gemma mostly waits, and
Qwen intervenes often and pays for service. At 90 days the costs of the
staffing levers fall inside the window, so the profit differences are
larger.

## 4. Replay and integrity

| Check | Result |
|---|---|
| `benchmark verify` on compare-30 (linux/amd64), with the suite checks | 689 of 689 |
| Starting state identical across policies (contract report) | 15 of 15 entry-seed pairs |
| Recorded-decision replay, one configuration per entry (`compare-30-replay-<entry>.json`), on linux/amd64 | 5 of 5 suite runs complete, each `verify` 95 of 95 |
| Replayed LLM runs equal to the live runs: `decisions.json`, `session.json`, `result.json`, the agent's `memory.jsonl`, and the initial and final digests | 30 of 30, byte for byte |
| Replay repeated on linux/amd64 | manifest byte-identical, 5 of 5 |
| `benchmark verify -t90` on compare-90 (linux/amd64) | 249 of 249 |
| Starting state identical across policies, compare-90 | 5 of 5 |
| Recorded-decision replay, compare-90 (`compare-90-replay-<entry>.json`) | 5 of 5 suite runs complete, each `verify` 41 of 41 |
| Replayed LLM runs equal to the live runs, compare-90 | 10 of 10, byte for byte |
| Replay repeated, compare-90 | manifest byte-identical, 5 of 5 |
| CI | the replays on linux/amd64, linux/arm64 and darwin/arm64, manifests compared |

In replay, the agent rebuilds each prompt and its memory from the
recording and checks them against the recorded SHA-256 before submitting
the recorded batches. It calls no model. Live inference is not
deterministic; the replay is.

## 5. Limitations

* **Three seeds at 30 days, one at 90.** Differences between seeds are
  large (Qwen in reporting backlog: +$1,070 and −$16,102), so no
  per-model pattern here is a statistical claim.
* **30 days is short for staffing levers.** Hires, allocation changes and
  teleradiology take 14–28 days to start, so their costs and benefits
  are only partly inside the window. Some accepted changes started after
  it. The 90-day stage covers them, on one seed.
* **One prompt and one set of settings.** Both models used the V0.5 agent
  unchanged: summary observation, guided catalog, memory on, temperature 0.
  Other settings could behave differently.
* **Radiologist overtime is free in the simulator** (salaried staff), so
  shift extensions for radiologists cost nothing. A model that extends
  them is not shown a cost.
* **No inference cost.** The local server publishes no prices; tokens and
  latency are reported instead.
* **Latency depends on the local server.** Gemma's mean was 8.7 s per
  request at 30 days and 17.3 s at 90; the server and its load are not
  controlled, so latencies are not comparable across stages.
* **`conservative-v2` is mostly inactive in loss-making entries,** by its
  financial guardrail. It is one conservative behaviour, not the only
  one.
