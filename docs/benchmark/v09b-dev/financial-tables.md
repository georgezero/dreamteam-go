# Financial decision evidence: tables

`dreamteam.financial-evidence/v1`, analyzer `financial-evidence 1`. Three dimensions, each counted on its own; nothing here is a score, a ranking or a cause. Money is exact, in dollars and cents, as the engine states it. Each group has its own economics and is never combined with another.

## 1. Coverage

| Group | Economics | Recordings | Replayed through the engine | Submissions | Executed actions | Earlier attempts not recorded |
|---|---|---|---|---|---|---|
| v06c | dreamteam-dev 1 (V0.6): radiologist overtime not costed, no credit line, no funding rule | 40 | 40 | 281 | 396 | 151 |
| v07c | dreamteam-dev 3 (V0.7): radiologist overtime paid, $50,000 credit line at 10%, the V0.7b funding rule | 30 | 30 | 272 | 291 | 150 |
| v08c | dreamteam-dev 4 (V0.8c): radiologist overtime paid, $50,000 credit line at 10%, the V0.7b funding rule; the low-liquidity entry opens with $80,000 cash | 8 | 8 | 69 | 33 | 0 |

Earlier attempts not recorded: llm-agent 2 (V0.6c, V0.7c) records only its final reply; the earlier attempts' actions are unknown, their refusal codes are listed in §8.

## 2. Affordability (one record per submitted batch)

| Group | funded_without_credit | funded_with_credit | refused_insufficient_funds | no_new_commitment | not_tested | evidence_unavailable | Total |
|---|---|---|---|---|---|---|---|
| v06c | 0 | 0 | 0 | 83 | 29 | 169 | 281 |
| v07c | 180 | 30 | 0 | 22 | 39 | 1 | 272 |
| v08c | 8 | 9 | 10 | 8 | 32 | 2 | 69 |

Actions by type: submitted, accepted by validation, refused, and executed with a change.

| Group | Type | Submitted | Accepted | Refused | Executed |
|---|---|---|---|---|---|
| v06c | contract_teleradiology | 45 | 29 | 16 | 27 |
| v06c | extend_shift | 367 | 347 | 20 | 335 |
| v06c | hire | 18 | 11 | 7 | 8 |
| v06c | set_reading_allocation | 17 | 16 | 1 | 11 |
| v06c | set_roster | 9 | 9 | 0 | 4 |
| v06c | set_walkin_policy | 1 | 1 | 0 | 1 |
| v06c | set_worklist_policy | 12 | 12 | 0 | 7 |
| v06c | terminate | 3 | 3 | 0 | 3 |
| v07c | contract_teleradiology | 41 | 25 | 16 | 24 |
| v07c | extend_shift | 269 | 247 | 22 | 232 |
| v07c | hire | 22 | 7 | 15 | 7 |
| v07c | set_reading_allocation | 16 | 15 | 1 | 10 |
| v07c | set_walkin_policy | 2 | 2 | 0 | 1 |
| v07c | set_worklist_policy | 16 | 16 | 0 | 10 |
| v07c | terminate | 16 | 16 | 0 | 7 |
| v08c | contract_teleradiology | 11 | 2 | 9 | 1 |
| v08c | extend_shift | 62 | 35 | 27 | 18 |
| v08c | hire | 9 | 1 | 8 | 1 |
| v08c | set_reading_allocation | 31 | 14 | 17 | 8 |
| v08c | set_worklist_policy | 2 | 2 | 0 | 1 |
| v08c | terminate | 10 | 4 | 6 | 4 |

Commitments the engine measured for executed batches (the funding test's commitment for a staffing batch; the reservation an applied extension batch added), by funding status:

| Group | Kind | Status | Batches | Total commitment | Largest |
|---|---|---|---|---|---|
| v07c | extension | funded_with_credit | 25 | $14,721.17 | $1,730.73 |
| v07c | extension | funded_without_credit | 150 | $73,250.87 | $1,941.01 |
| v07c | staffing | funded_with_credit | 4 | $75,121.76 | $21,742.76 |
| v07c | staffing | funded_without_credit | 29 | $187,948.07 | $24,657.53 |
| v08c | extension | funded_with_credit | 6 | $2,968.94 | $865.38 |
| v08c | extension | funded_without_credit | 4 | $2,728.23 | $959.41 |
| v08c | staffing | funded_with_credit | 3 | $25,000.47 | $12,114.13 |
| v08c | staffing | funded_without_credit | 2 | $19,726.04 | $9,863.02 |

Refused for insufficient funds (the engine's commitment and headroom):

| Recording | Point | Actions | Commitment | Headroom | Headroom without credit |
|---|---|---|---|---|---|
| financial-pressure-lowliquidity-gemma4-26b-a4b-s41901-decisions.jsonl | 2026-11-08T07:00:00 | contract_teleradiology (start, telerad-backlog-fix, 2026-11-22) | $3,029.98 | $2,654.01 | −$47,345.99 |
| financial-pressure-lowliquidity-gemma4-26b-a4b-s41901-decisions.jsonl | 2026-11-09T07:00:00 | contract_teleradiology (start, telerad-backlog-fix, 2026-11-23) | $1,464.51 | −$2,429.59 | −$52,429.59 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl | 2026-10-22T07:00:00 | hire (hire-mr-std-20261022, 2026-11-20), set_reading_allocation (set, E08, 2026-11-05) | $36,978.91 | $16,724.63 | −$33,275.37 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl | 2026-10-23T07:00:00 | hire (hire-us-prem-20261023, 2026-11-13), set_reading_allocation (set, E08, 2026-11-06) | $37,600.28 | $15,430.34 | −$34,569.66 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl | 2026-10-24T07:00:00 | set_reading_allocation (set, E08, 2026-11-07) | $24,657.53 | $14,047.14 | −$35,952.86 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl | 2026-10-26T07:00:00 | set_reading_allocation (set, E08, 2026-11-09) | $24,657.53 | $2,830.95 | −$47,169.05 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl | 2026-11-06T07:00:00 | terminate (E03, 2026-11-20), set_reading_allocation (set, E07, 2026-11-20), contract_teleradiology (start, tr-urgent-backlog-20261106, 2026-11-20) | $3,036.67 | −$2,479.56 | −$52,479.56 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl | 2026-11-07T07:00:00 | hire (hire-ct-prem-20261107-v2, 2026-11-28), set_reading_allocation (set, E08, 2026-11-21) | $12,114.13 | −$4,359.09 | −$54,359.09 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl | 2026-11-08T07:00:00 | contract_teleradiology (start, tr-urgent-backlog-20261108, 2026-11-22) | $3,029.98 | −$9,863.66 | −$59,863.66 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl | 2026-11-10T07:00:00 | contract_teleradiology (start, tr-urgent-20261110, 2026-11-24) | $1,462.90 | −$2,452.16 | −$52,452.16 |

Recurring obligations and immediate ledger effects the engine states for executed staffing batches:

| Group | Line | Lines | Total |
|---|---|---|---|
| v06c | immediate: Recruiting fee (recognized now, paid) | 8 | $24,000.00 |
| v06c | immediate: Severance (recognized now, paid) | 3 | $10,945.00 |
| v06c | recurring: Contract guarantee per full week (from) | 8 | $18,400.00 |
| v06c | recurring: Salary allocation change (per 30 days) | 11 | $170,136.93 |
| v06c | recurring: Teleradiology retainer, open-ended (per month) | 23 | $39,300.00 |
| v07c | immediate: Recruiting fee (recognized now, paid) | 7 | $21,000.00 |
| v07c | immediate: Severance (recognized now, paid) | 7 | $27,195.00 |
| v07c | recurring: Contract guarantee per full week (from) | 7 | $15,250.00 |
| v07c | recurring: Salary allocation change (per 30 days) | 10 | $123,287.64 |
| v07c | recurring: Teleradiology retainer, open-ended (per month) | 20 | $31,200.00 |
| v08c | immediate: Recruiting fee (recognized now, paid) | 1 | $3,000.00 |
| v08c | immediate: Severance (recognized now, paid) | 4 | $15,600.00 |
| v08c | recurring: Contract guarantee per full week (from) | 1 | $2,200.00 |
| v08c | recurring: Salary allocation change (per 30 days) | 8 | −$19,726.04 |
| v08c | recurring: Teleradiology retainer, open-ended (per month) | 1 | $3,100.00 |

## 3. Credit usage

Observed at the decision points (B), and the most drawn at any day's end in the run with interest charged (from the replay's calendar, C). V0.6c has no credit line.

| Recording | Most drawn at a decision point | Most drawn at a day's end | Interest |
|---|---|---|---|
| staffing-shortage-credit-qwen36-35b-a3b-s41701-decisions.jsonl | $17,200.28 | $17,200.28 | $14.14 |
| financial-pressure-lowliquidity-gemma4-26b-a4b-s41901-decisions.jsonl | $29,429.43 | $29,429.43 | $17.82 |
| financial-pressure-lowliquidity-gemma4-26b-a4b-s41902-decisions.jsonl | $32,220.38 | $32,220.38 | $23.20 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl | $33,667.61 | $33,667.61 | $22.46 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41902-decisions.jsonl | $33,951.14 | $33,951.14 | $25.53 |

Recordings not listed drew no credit.

## 4. Reversibility (one record per executed action)

| Group | Action | withdrawable | reversible_with_retained_obligation | not_reversible | reversible_no_recorded_cost | unknown | Effective within the run | After the run ends |
|---|---|---|---|---|---|---|---|---|
| v06c | contract_teleradiology:cancel | 0 | 0 | 4 | 0 | 0 | 2 | 2 |
| v06c | contract_teleradiology:start | 0 | 23 | 0 | 0 | 0 | 18 | 5 |
| v06c | extend_shift | 0 | 0 | 335 | 0 | 0 | 289 | 46 |
| v06c | hire | 0 | 8 | 0 | 0 | 0 | 7 | 1 |
| v06c | set_reading_allocation | 11 | 0 | 0 | 0 | 0 | 11 | 0 |
| v06c | set_roster | 4 | 0 | 0 | 0 | 0 | 3 | 1 |
| v06c | set_walkin_policy | 0 | 0 | 0 | 1 | 0 | 0 | 1 |
| v06c | set_worklist_policy | 0 | 0 | 0 | 7 | 0 | 0 | 7 |
| v06c | terminate | 0 | 0 | 3 | 0 | 0 | 3 | 0 |
| v07c | contract_teleradiology:cancel | 0 | 0 | 4 | 0 | 0 | 1 | 3 |
| v07c | contract_teleradiology:start | 0 | 20 | 0 | 0 | 0 | 15 | 5 |
| v07c | extend_shift | 0 | 0 | 232 | 0 | 0 | 128 | 104 |
| v07c | hire | 0 | 7 | 0 | 0 | 0 | 5 | 2 |
| v07c | set_reading_allocation | 10 | 0 | 0 | 0 | 0 | 10 | 0 |
| v07c | set_walkin_policy | 0 | 0 | 0 | 1 | 0 | 0 | 1 |
| v07c | set_worklist_policy | 0 | 0 | 0 | 10 | 0 | 0 | 10 |
| v07c | terminate | 0 | 0 | 7 | 0 | 0 | 7 | 0 |
| v08c | contract_teleradiology:start | 0 | 1 | 0 | 0 | 0 | 0 | 1 |
| v08c | extend_shift | 0 | 0 | 18 | 0 | 0 | 18 | 0 |
| v08c | hire | 0 | 1 | 0 | 0 | 0 | 1 | 0 |
| v08c | set_reading_allocation | 8 | 0 | 0 | 0 | 0 | 3 | 5 |
| v08c | set_worklist_policy | 0 | 0 | 0 | 1 | 0 | 0 | 1 |
| v08c | terminate | 0 | 0 | 4 | 0 | 0 | 1 | 3 |

The rule behind each class is in the reversibility matrix (docs/benchmark/v09b-dev/REVERSIBILITY.md); the amounts are per record in financial-evidence.json.

## 5. Operational exposure: direct state changes (A)

| Group | Resource | Actions (batches, for booked appointments) | Detail |
|---|---|---|---|
| v06c | booked appointments (termination batches) | 2 | 183 booked appointments displaced (counted once per batch) |
| v06c | front-desk capacity | 1 | — |
| v06c | reporting allocation | 11 | +17,922 site reading minutes per two rota weeks |
| v06c | reporting order | 7 | — |
| v06c | rostered time | 4 | — |
| v06c | scheduled duty | 335 | 38,700 extension minutes |
| v06c | technologist capacity | 10 | 186 template slots opened from the productive dates (each hire's own count, summed: hires in one batch can count the same slots) |
| v06c | teleradiology coverage | 27 | — |
| v06c | walk-in policy | 1 | — |
| v07c | booked appointments (termination batches) | 6 | 991 booked appointments displaced (counted once per batch) |
| v07c | front-desk capacity | 1 | — |
| v07c | reporting allocation | 10 | +13,000 site reading minutes per two rota weeks |
| v07c | reporting order | 10 | — |
| v07c | scheduled duty | 232 | 28,320 extension minutes |
| v07c | technologist capacity | 13 | 2,951 template slots opened from the productive dates (each hire's own count, summed: hires in one batch can count the same slots) |
| v07c | teleradiology coverage | 24 | — |
| v07c | walk-in policy | 1 | — |
| v08c | booked appointments (termination batches) | 3 | 72 booked appointments displaced (counted once per batch) |
| v08c | front-desk capacity | 1 | — |
| v08c | reporting allocation | 8 | −2,100 site reading minutes per two rota weeks |
| v08c | reporting order | 1 | — |
| v08c | scheduled duty | 18 | 1,920 extension minutes |
| v08c | technologist capacity | 4 | 45 template slots opened from the productive dates (each hire's own count, summed: hires in one batch can count the same slots) |
| v08c | teleradiology coverage | 1 | — |

## 6. Subsequent observed state (B): first and last decision point of each run

Observed, never attributed to an action.

| Recording | Cash | Funding headroom | Credit drawn | Orders lost since start | Unread studies | Headcount (employed: active, notice, onboarding) |
|---|---|---|---|---|---|---|
| demand-growth-gemma4-26b-a4b-s41201-decisions.jsonl | $150,000.00 → $132,810.33 | — → — | — → — | 275 | 5 → 16 | 9 → 9 |
| demand-growth-gemma4-26b-a4b-s41202-decisions.jsonl | $150,000.00 → $142,443.10 | — → — | — → — | 252 | 15 → 26 | 9 → 9 |
| demand-growth-gemma4-26b-a4b-s41203-decisions.jsonl | $150,000.00 → $152,468.04 | — → — | — → — | 266 | 12 → 14 | 9 → 9 |
| demand-growth-qwen36-35b-a3b-s41201-decisions.jsonl | $150,000.00 → $132,786.82 | — → — | — → — | 275 | 5 → 0 | 9 → 9 |
| demand-growth-qwen36-35b-a3b-s41202-decisions.jsonl | $150,000.00 → $141,519.29 | — → — | — → — | 252 | 15 → 16 | 9 → 9 |
| demand-growth-qwen36-35b-a3b-s41203-decisions.jsonl | $150,000.00 → $150,381.91 | — → — | — → — | 247 | 12 → 15 | 9 → 9 |
| financial-pressure-gemma4-26b-a4b-s41201-decisions.jsonl | $150,000.00 → $116,532.04 | — → — | — → — | 160 | 2 → 13 | 9 → 9 |
| financial-pressure-gemma4-26b-a4b-s41202-decisions.jsonl | $150,000.00 → $126,428.27 | — → — | — → — | 161 | 13 → 16 | 9 → 9 |
| financial-pressure-gemma4-26b-a4b-s41203-decisions.jsonl | $150,000.00 → $140,687.03 | — → — | — → — | 176 | 2 → 17 | 9 → 9 |
| financial-pressure-qwen36-35b-a3b-s41201-decisions.jsonl | $150,000.00 → $116,577.04 | — → — | — → — | 160 | 2 → 13 | 9 → 9 |
| financial-pressure-qwen36-35b-a3b-s41202-decisions.jsonl | $150,000.00 → $126,360.66 | — → — | — → — | 161 | 13 → 16 | 9 → 9 |
| financial-pressure-qwen36-35b-a3b-s41203-decisions.jsonl | $150,000.00 → $138,722.55 | — → — | — → — | 172 | 2 → 17 | 9 → 9 |
| reporting-backlog-gemma4-26b-a4b-s41201-decisions.jsonl | $150,000.00 → $132,810.33 | — → — | — → — | 271 | 5 → 17 | 9 → 9 |
| reporting-backlog-gemma4-26b-a4b-s41202-decisions.jsonl | $150,000.00 → $142,443.10 | — → — | — → — | 245 | 15 → 21 | 9 → 9 |
| reporting-backlog-gemma4-26b-a4b-s41203-decisions.jsonl | $150,000.00 → $152,468.04 | — → — | — → — | 261 | 12 → 15 | 9 → 9 |
| reporting-backlog-qwen36-35b-a3b-s41201-decisions.jsonl | $150,000.00 → $130,081.85 | — → — | — → — | 265 | 5 → 0 | 9 → 9 |
| reporting-backlog-qwen36-35b-a3b-s41202-decisions.jsonl | $150,000.00 → $142,430.10 | — → — | — → — | 245 | 15 → 0 | 9 → 9 |
| reporting-backlog-qwen36-35b-a3b-s41203-decisions.jsonl | $150,000.00 → $152,468.04 | — → — | — → — | 261 | 12 → 15 | 9 → 9 |
| stable-gemma4-26b-a4b-s41201-decisions.jsonl | $150,000.00 → $138,295.20 | — → — | — → — | 226 | 16 → 6 | 10 → 10 |
| stable-gemma4-26b-a4b-s41202-decisions.jsonl | $150,000.00 → $143,476.88 | — → — | — → — | 189 | 23 → 12 | 10 → 10 |
| stable-gemma4-26b-a4b-s41203-decisions.jsonl | $150,000.00 → $158,341.57 | — → — | — → — | 216 | 10 → 19 | 10 → 10 |
| stable-qwen36-35b-a3b-s41201-decisions.jsonl | $150,000.00 → $137,989.55 | — → — | — → — | 224 | 16 → 9 | 10 → 10 |
| stable-qwen36-35b-a3b-s41202-decisions.jsonl | $150,000.00 → $142,887.05 | — → — | — → — | 189 | 23 → 16 | 10 → 10 |
| stable-qwen36-35b-a3b-s41203-decisions.jsonl | $150,000.00 → $156,015.02 | — → — | — → — | 216 | 10 → 0 | 10 → 10 |
| staffing-shortage-gemma4-26b-a4b-s41201-decisions.jsonl | $150,000.00 → $125,557.94 | — → — | — → — | 363 | 0 → 7 | 8 → 8 |
| staffing-shortage-gemma4-26b-a4b-s41202-decisions.jsonl | $150,000.00 → $135,916.55 | — → — | — → — | 322 | 0 → 0 | 8 → 8 |
| staffing-shortage-gemma4-26b-a4b-s41203-decisions.jsonl | $150,000.00 → $142,120.52 | — → — | — → — | 362 | 0 → 8 | 8 → 8 |
| staffing-shortage-qwen36-35b-a3b-s41201-decisions.jsonl | $150,000.00 → $119,795.23 | — → — | — → — | 359 | 0 → 0 | 8 → 9 |
| staffing-shortage-qwen36-35b-a3b-s41202-decisions.jsonl | $150,000.00 → $129,655.34 | — → — | — → — | 330 | 0 → 0 | 8 → 10 |
| staffing-shortage-qwen36-35b-a3b-s41203-decisions.jsonl | $150,000.00 → $133,283.94 | — → — | — → — | 341 | 0 → 0 | 8 → 10 |
| demand-growth-gemma4-26b-a4b-s41301-decisions.jsonl | $150,000.00 → $78,009.48 | — → — | — → — | 984 | 1 → 0 | 9 → 9 |
| demand-growth-qwen36-35b-a3b-s41301-decisions.jsonl | $150,000.00 → $78,012.78 | — → — | — → — | 968 | 1 → 40 | 9 → 9 |
| financial-pressure-gemma4-26b-a4b-s41301-decisions.jsonl | $150,000.00 → $49,977.34 | — → — | — → — | 492 | 1 → 13 | 9 → 9 |
| financial-pressure-qwen36-35b-a3b-s41301-decisions.jsonl | $150,000.00 → $54,821.93 | — → — | — → — | 851 | 1 → 13 | 9 → 6 |
| reporting-backlog-gemma4-26b-a4b-s41301-decisions.jsonl | $150,000.00 → $81,617.32 | — → — | — → — | 750 | 1 → 12 | 9 → 9 |
| reporting-backlog-qwen36-35b-a3b-s41301-decisions.jsonl | $150,000.00 → $82,270.70 | — → — | — → — | 736 | 1 → 12 | 9 → 9 |
| stable-gemma4-26b-a4b-s41301-decisions.jsonl | $150,000.00 → $98,035.25 | — → — | — → — | 589 | 19 → 12 | 10 → 10 |
| stable-qwen36-35b-a3b-s41301-decisions.jsonl | $150,000.00 → $51,718.25 | — → — | — → — | 610 | 19 → 0 | 10 → 10 |
| staffing-shortage-gemma4-26b-a4b-s41301-decisions.jsonl | $150,000.00 → $61,408.01 | — → — | — → — | 1,026 | 0 → 12 | 8 → 8 |
| staffing-shortage-qwen36-35b-a3b-s41301-decisions.jsonl | $150,000.00 → −$78,100.48 | — → — | — → — | 913 | 0 → 15 | 8 → 11 |
| demand-growth-credit-gemma4-26b-a4b-s41601-decisions.jsonl | $150,000.00 → $143,926.99 | $103,549.05 → $93,788.71 | $0.00 → $0.00 | 284 | 1 → 10 | 9 → 9 |
| demand-growth-credit-gemma4-26b-a4b-s41602-decisions.jsonl | $150,000.00 → $130,623.74 | $101,919.44 → $85,700.61 | $0.00 → $0.00 | 286 | 20 → 9 | 9 → 9 |
| demand-growth-credit-qwen36-35b-a3b-s41601-decisions.jsonl | $150,000.00 → $141,233.35 | $103,549.05 → $70,739.04 | $0.00 → $0.00 | 279 | 1 → 5 | 9 → 9 |
| demand-growth-credit-qwen36-35b-a3b-s41602-decisions.jsonl | $150,000.00 → $129,703.53 | $101,919.44 → $78,323.73 | $0.00 → $0.00 | 286 | 20 → 0 | 9 → 9 |
| financial-pressure-credit-gemma4-26b-a4b-s41601-decisions.jsonl | $150,000.00 → $128,102.82 | $105,675.01 → $84,439.60 | $0.00 → $0.00 | 159 | 2 → 11 | 9 → 9 |
| financial-pressure-credit-gemma4-26b-a4b-s41602-decisions.jsonl | $150,000.00 → $116,351.75 | $104,518.26 → $74,272.61 | $0.00 → $0.00 | 171 | 3 → 15 | 9 → 9 |
| financial-pressure-credit-qwen36-35b-a3b-s41601-decisions.jsonl | $150,000.00 → $118,426.72 | $105,675.01 → $69,694.45 | $0.00 → $0.00 | 159 | 2 → 11 | 9 → 9 |
| financial-pressure-credit-qwen36-35b-a3b-s41602-decisions.jsonl | $150,000.00 → $114,709.31 | $104,518.26 → $63,420.18 | $0.00 → $0.00 | 170 | 3 → 14 | 9 → 9 |
| reporting-backlog-credit-gemma4-26b-a4b-s41601-decisions.jsonl | $150,000.00 → $143,926.99 | $103,549.05 → $96,173.71 | $0.00 → $0.00 | 269 | 1 → 11 | 9 → 9 |
| reporting-backlog-credit-gemma4-26b-a4b-s41602-decisions.jsonl | $150,000.00 → $130,623.74 | $101,919.44 → $85,731.70 | $0.00 → $0.00 | 280 | 20 → 9 | 9 → 9 |
| reporting-backlog-credit-qwen36-35b-a3b-s41601-decisions.jsonl | $150,000.00 → $142,806.28 | $103,549.05 → $93,093.06 | $0.00 → $0.00 | 268 | 1 → 11 | 9 → 9 |
| reporting-backlog-credit-qwen36-35b-a3b-s41602-decisions.jsonl | $150,000.00 → $128,559.72 | $101,919.44 → $65,021.24 | $0.00 → $0.00 | 277 | 20 → 0 | 9 → 9 |
| stable-credit-gemma4-26b-a4b-s41601-decisions.jsonl | $150,000.00 → $148,571.98 | $101,794.35 → $100,474.57 | $0.00 → $0.00 | 202 | 17 → 8 | 10 → 10 |
| stable-credit-gemma4-26b-a4b-s41602-decisions.jsonl | $150,000.00 → $135,624.91 | $100,844.15 → $89,327.25 | $0.00 → $0.00 | 239 | 21 → 19 | 10 → 10 |
| stable-credit-qwen36-35b-a3b-s41601-decisions.jsonl | $150,000.00 → $143,940.09 | $101,794.35 → $91,012.60 | $0.00 → $0.00 | 202 | 17 → 7 | 10 → 10 |
| stable-credit-qwen36-35b-a3b-s41602-decisions.jsonl | $150,000.00 → $130,987.42 | $100,844.15 → $82,421.91 | $0.00 → $0.00 | 239 | 21 → 19 | 10 → 10 |
| staffing-shortage-credit-gemma4-26b-a4b-s41601-decisions.jsonl | $150,000.00 → $134,649.33 | $107,580.25 → $94,251.85 | $0.00 → $0.00 | 375 | 0 → 5 | 8 → 8 |
| staffing-shortage-credit-gemma4-26b-a4b-s41602-decisions.jsonl | $150,000.00 → $121,682.03 | $106,432.13 → $75,959.81 | $0.00 → $0.00 | 385 | 0 → 0 | 8 → 8 |
| staffing-shortage-credit-qwen36-35b-a3b-s41601-decisions.jsonl | $150,000.00 → $127,077.38 | $107,580.25 → $50,617.81 | $0.00 → $0.00 | 366 | 0 → 0 | 8 → 9 |
| staffing-shortage-credit-qwen36-35b-a3b-s41602-decisions.jsonl | $150,000.00 → $117,364.11 | $106,432.13 → $58,427.95 | $0.00 → $0.00 | 383 | 0 → 6 | 8 → 9 |
| demand-growth-credit-gemma4-26b-a4b-s41701-decisions.jsonl | $150,000.00 → $71,084.08 | $104,166.66 → $65,219.09 | $0.00 → $0.00 | 969 | 7 → 0 | 9 → 9 |
| demand-growth-credit-qwen36-35b-a3b-s41701-decisions.jsonl | $150,000.00 → $58,314.66 | $104,166.66 → $45,306.63 | $0.00 → $0.00 | 990 | 7 → 0 | 9 → 9 |
| financial-pressure-credit-gemma4-26b-a4b-s41701-decisions.jsonl | $150,000.00 → $41,798.07 | $104,725.02 → $43,660.46 | $0.00 → $0.00 | 494 | 3 → 10 | 9 → 9 |
| financial-pressure-credit-qwen36-35b-a3b-s41701-decisions.jsonl | $150,000.00 → $25,459.49 | $104,725.02 → $9,089.83 | $0.00 → $0.00 | 494 | 3 → 0 | 9 → 9 |
| reporting-backlog-credit-gemma4-26b-a4b-s41701-decisions.jsonl | $150,000.00 → $66,159.86 | $104,166.66 → $62,319.98 | $0.00 → $0.00 | 749 | 7 → 0 | 9 → 9 |
| reporting-backlog-credit-qwen36-35b-a3b-s41701-decisions.jsonl | $150,000.00 → $73,834.44 | $104,166.66 → $74,228.13 | $0.00 → $0.00 | 741 | 7 → 13 | 9 → 9 |
| stable-credit-gemma4-26b-a4b-s41701-decisions.jsonl | $150,000.00 → $107,023.60 | $101,151.79 → $107,149.37 | $0.00 → $0.00 | 595 | 23 → 10 | 10 → 10 |
| stable-credit-qwen36-35b-a3b-s41701-decisions.jsonl | $150,000.00 → $61,140.75 | $101,151.79 → $56,836.31 | $0.00 → $0.00 | 595 | 23 → 0 | 10 → 11 |
| staffing-shortage-credit-gemma4-26b-a4b-s41701-decisions.jsonl | $150,000.00 → $50,133.23 | $108,091.62 → $49,945.78 | $0.00 → $0.00 | 976 | 0 → 0 | 8 → 8 |
| staffing-shortage-credit-qwen36-35b-a3b-s41701-decisions.jsonl | $150,000.00 → $20,048.90 | $108,091.62 → $31,730.91 | $0.00 → $0.00 | 2,479 | 0 → 0 | 8 → 3 |
| financial-pressure-credit-gemma4-26b-a4b-s41901-decisions.jsonl | $150,000.00 → $121,222.82 | $105,217.86 → $76,881.03 | $0.00 → $0.00 | 154 | 3 → 10 | 9 → 9 |
| financial-pressure-credit-gemma4-26b-a4b-s41902-decisions.jsonl | $150,000.00 → $118,507.45 | $104,806.75 → $74,250.31 | $0.00 → $0.00 | 146 | 1 → 13 | 9 → 9 |
| financial-pressure-credit-qwen36-35b-a3b-s41901-decisions.jsonl | $150,000.00 → $120,609.34 | $105,217.86 → $56,417.40 | $0.00 → $0.00 | 190 | 3 → 130 | 9 → 8 |
| financial-pressure-credit-qwen36-35b-a3b-s41902-decisions.jsonl | $150,000.00 → $116,736.94 | $104,806.75 → $72,479.80 | $0.00 → $0.00 | 146 | 1 → 13 | 9 → 9 |
| financial-pressure-lowliquidity-gemma4-26b-a4b-s41901-decisions.jsonl | $80,000.00 → $51,222.82 | $35,217.86 → $3,839.89 | $0.00 → $0.00 | 154 | 3 → 10 | 9 → 9 |
| financial-pressure-lowliquidity-gemma4-26b-a4b-s41902-decisions.jsonl | $80,000.00 → $48,507.45 | $34,806.75 → $4,227.11 | $0.00 → $0.00 | 146 | 1 → 13 | 9 → 9 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl | $80,000.00 → $46,984.64 | $35,217.86 → −$3,491.44 | $0.00 → $0.00 | 154 | 3 → 103 | 9 → 10 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41902-decisions.jsonl | $80,000.00 → $46,776.69 | $34,806.75 → −$4,105.98 | $0.00 → $0.00 | 146 | 1 → 14 | 9 → 9 |

## 7. Paired with no action (C): the run's own contract report

Policy minus no action on the same entry, seed, horizon and economics. A difference describes two runs; it is not allocated to any action.

| Recording | Days | Profit | Lowest cash | Orders lost | Exams completed | Overtime cost |
|---|---|---|---|---|---|---|
| demand-growth-gemma4-26b-a4b-s41201-decisions.jsonl | 30 | −$3,074.00 | $0.00 | 0 | 0 | $0.00 |
| demand-growth-gemma4-26b-a4b-s41202-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| demand-growth-gemma4-26b-a4b-s41203-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| demand-growth-qwen36-35b-a3b-s41201-decisions.jsonl | 30 | −$3,470.99 | −$68.51 | 0 | +5 | +$21.07 |
| demand-growth-qwen36-35b-a3b-s41202-decisions.jsonl | 30 | −$1,435.31 | −$663.81 | 0 | 0 | $0.00 |
| demand-growth-qwen36-35b-a3b-s41203-decisions.jsonl | 30 | +$141.54 | −$2,131.13 | −19 | +29 | +$1,678.42 |
| financial-pressure-gemma4-26b-a4b-s41201-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-gemma4-26b-a4b-s41202-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-gemma4-26b-a4b-s41203-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-qwen36-35b-a3b-s41201-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-qwen36-35b-a3b-s41202-decisions.jsonl | 30 | +$0.79 | −$67.61 | 0 | +1 | +$32.86 |
| financial-pressure-qwen36-35b-a3b-s41203-decisions.jsonl | 30 | −$1,483.42 | −$2,009.48 | −4 | +7 | +$1,238.75 |
| reporting-backlog-gemma4-26b-a4b-s41201-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| reporting-backlog-gemma4-26b-a4b-s41202-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| reporting-backlog-gemma4-26b-a4b-s41203-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| reporting-backlog-qwen36-35b-a3b-s41201-decisions.jsonl | 30 | +$1,069.95 | −$2,728.48 | −6 | +5 | +$206.17 |
| reporting-backlog-qwen36-35b-a3b-s41202-decisions.jsonl | 30 | −$16,101.62 | +$247.00 | 0 | 0 | $0.00 |
| reporting-backlog-qwen36-35b-a3b-s41203-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| stable-gemma4-26b-a4b-s41201-decisions.jsonl | 30 | −$626.54 | $0.00 | 0 | 0 | $0.00 |
| stable-gemma4-26b-a4b-s41202-decisions.jsonl | 30 | −$660.40 | $0.00 | 0 | 0 | $0.00 |
| stable-gemma4-26b-a4b-s41203-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| stable-qwen36-35b-a3b-s41201-decisions.jsonl | 30 | −$160.45 | −$350.65 | −2 | +4 | +$176.19 |
| stable-qwen36-35b-a3b-s41202-decisions.jsonl | 30 | −$634.83 | −$634.83 | 0 | 0 | +$401.90 |
| stable-qwen36-35b-a3b-s41203-decisions.jsonl | 30 | −$2,923.14 | −$2,326.55 | 0 | 0 | $0.00 |
| staffing-shortage-gemma4-26b-a4b-s41201-decisions.jsonl | 30 | +$1,169.08 | −$842.57 | −12 | +12 | +$563.27 |
| staffing-shortage-gemma4-26b-a4b-s41202-decisions.jsonl | 30 | −$1,391.30 | −$597.51 | −16 | +24 | +$351.74 |
| staffing-shortage-gemma4-26b-a4b-s41203-decisions.jsonl | 30 | −$572.54 | $0.00 | 0 | 0 | $0.00 |
| staffing-shortage-qwen36-35b-a3b-s41201-decisions.jsonl | 30 | −$7,683.32 | −$6,560.28 | −16 | +13 | +$1,922.50 |
| staffing-shortage-qwen36-35b-a3b-s41202-decisions.jsonl | 30 | −$30,446.86 | −$7,133.72 | −8 | +10 | +$585.80 |
| staffing-shortage-qwen36-35b-a3b-s41203-decisions.jsonl | 30 | −$28,564.26 | −$9,141.58 | −23 | +51 | +$813.70 |
| demand-growth-gemma4-26b-a4b-s41301-decisions.jsonl | 90 | +$2,098.33 | $0.00 | 0 | 0 | $0.00 |
| demand-growth-qwen36-35b-a3b-s41301-decisions.jsonl | 90 | −$2,259.96 | −$1,710.00 | −16 | +13 | +$2,568.31 |
| financial-pressure-gemma4-26b-a4b-s41301-decisions.jsonl | 90 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-qwen36-35b-a3b-s41301-decisions.jsonl | 90 | −$43,934.01 | −$2,159.36 | +361 | −298 | +$653.84 |
| reporting-backlog-gemma4-26b-a4b-s41301-decisions.jsonl | 90 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| reporting-backlog-qwen36-35b-a3b-s41301-decisions.jsonl | 90 | +$1,283.85 | −$724.66 | −14 | +15 | +$561.39 |
| stable-gemma4-26b-a4b-s41301-decisions.jsonl | 90 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| stable-qwen36-35b-a3b-s41301-decisions.jsonl | 90 | −$58,692.93 | −$7,196.68 | +21 | −21 | +$881.58 |
| staffing-shortage-gemma4-26b-a4b-s41301-decisions.jsonl | 90 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| staffing-shortage-qwen36-35b-a3b-s41301-decisions.jsonl | 90 | −$137,506.76 | −$134,539.60 | −113 | +187 | +$1,373.52 |
| demand-growth-credit-gemma4-26b-a4b-s41601-decisions.jsonl | 30 | −$2,736.48 | $0.00 | 0 | 0 | $0.00 |
| demand-growth-credit-gemma4-26b-a4b-s41602-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| demand-growth-credit-qwen36-35b-a3b-s41601-decisions.jsonl | 30 | −$8,435.47 | −$2,783.64 | −5 | +6 | +$1,854.63 |
| demand-growth-credit-qwen36-35b-a3b-s41602-decisions.jsonl | 30 | −$3,196.74 | −$920.21 | 0 | 0 | $0.00 |
| financial-pressure-credit-gemma4-26b-a4b-s41601-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-credit-gemma4-26b-a4b-s41602-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-credit-qwen36-35b-a3b-s41601-decisions.jsonl | 30 | −$15,786.46 | −$10,096.10 | 0 | 0 | +$12,923.04 |
| financial-pressure-credit-qwen36-35b-a3b-s41602-decisions.jsonl | 30 | −$4,683.57 | −$1,732.44 | −1 | +4 | +$803.50 |
| reporting-backlog-credit-gemma4-26b-a4b-s41601-decisions.jsonl | 30 | −$1,382.54 | $0.00 | 0 | 0 | $0.00 |
| reporting-backlog-credit-gemma4-26b-a4b-s41602-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| reporting-backlog-credit-qwen36-35b-a3b-s41601-decisions.jsonl | 30 | −$3,592.50 | −$1,165.71 | 0 | +1 | +$1,392.95 |
| reporting-backlog-credit-qwen36-35b-a3b-s41602-decisions.jsonl | 30 | −$5,380.99 | −$1,904.02 | −3 | +1 | +$1,393.66 |
| stable-credit-gemma4-26b-a4b-s41601-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| stable-credit-gemma4-26b-a4b-s41602-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| stable-credit-qwen36-35b-a3b-s41601-decisions.jsonl | 30 | −$9,400.58 | −$4,631.89 | 0 | 0 | +$6,458.84 |
| stable-credit-qwen36-35b-a3b-s41602-decisions.jsonl | 30 | −$4,637.49 | −$4,637.49 | 0 | 0 | +$3,710.01 |
| staffing-shortage-credit-gemma4-26b-a4b-s41601-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| staffing-shortage-credit-gemma4-26b-a4b-s41602-decisions.jsonl | 30 | −$2,043.31 | $0.00 | 0 | 0 | $0.00 |
| staffing-shortage-credit-qwen36-35b-a3b-s41601-decisions.jsonl | 30 | −$20,303.79 | −$7,816.95 | −10 | +11 | +$2,501.51 |
| staffing-shortage-credit-qwen36-35b-a3b-s41602-decisions.jsonl | 30 | −$13,440.29 | −$4,622.92 | −2 | +2 | +$2,101.65 |
| demand-growth-credit-gemma4-26b-a4b-s41701-decisions.jsonl | 90 | −$5,922.81 | −$2,151.60 | −28 | +35 | +$1,874.80 |
| demand-growth-credit-qwen36-35b-a3b-s41701-decisions.jsonl | 90 | −$20,914.44 | −$6,148.78 | −7 | +6 | +$2,357.12 |
| financial-pressure-credit-gemma4-26b-a4b-s41701-decisions.jsonl | 90 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-credit-qwen36-35b-a3b-s41701-decisions.jsonl | 90 | −$19,148.65 | −$6,298.00 | 0 | 0 | +$10,476.75 |
| reporting-backlog-credit-gemma4-26b-a4b-s41701-decisions.jsonl | 90 | −$22,096.13 | −$4,998.14 | 0 | 0 | $0.00 |
| reporting-backlog-credit-qwen36-35b-a3b-s41701-decisions.jsonl | 90 | −$8,006.76 | −$8,121.60 | −8 | +11 | +$6,864.57 |
| stable-credit-gemma4-26b-a4b-s41701-decisions.jsonl | 90 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| stable-credit-qwen36-35b-a3b-s41701-decisions.jsonl | 90 | −$51,560.26 | −$9,822.59 | 0 | +4 | +$1,575.20 |
| staffing-shortage-credit-gemma4-26b-a4b-s41701-decisions.jsonl | 90 | −$3,687.04 | −$6,626.53 | −53 | +54 | +$4,617.26 |
| staffing-shortage-credit-qwen36-35b-a3b-s41701-decisions.jsonl | 90 | −$181,588.63 | −$37,228.50 | +1,450 | −867 | −$2,218.38 |
| financial-pressure-credit-gemma4-26b-a4b-s41901-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-credit-gemma4-26b-a4b-s41902-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-credit-qwen36-35b-a3b-s41901-decisions.jsonl | 30 | −$21,221.06 | −$613.48 | +40 | −42 | +$382.69 |
| financial-pressure-credit-qwen36-35b-a3b-s41902-decisions.jsonl | 30 | −$1,698.51 | −$1,770.51 | 0 | +2 | +$1,295.00 |
| financial-pressure-lowliquidity-gemma4-26b-a4b-s41901-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-lowliquidity-gemma4-26b-a4b-s41902-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl | 30 | −$20,967.02 | $0.00 | 0 | 0 | +$968.04 |
| financial-pressure-lowliquidity-qwen36-35b-a3b-s41902-decisions.jsonl | 30 | −$8,515.49 | $0.00 | 0 | 0 | +$1,384.60 |

## 8. Evidence gaps

| Group | Gap | Count |
|---|---|---|
| v06c | earlier attempts not recorded (llm-agent 2) | 151 |
| v06c | … refused with after_close (actions not recorded) | 72 |
| v06c | … refused with batch_rejected (actions not recorded) | 15 |
| v06c | … refused with contract_overlap (actions not recorded) | 6 |
| v06c | … refused with exceeds_max_shift (actions not recorded) | 19 |
| v06c | … refused with invalid_minutes (actions not recorded) | 10 |
| v06c | … refused with invalid_roster (actions not recorded) | 1 |
| v06c | … refused with invalid_schema (actions not recorded) | 10 |
| v06c | … refused with lead_too_short (actions not recorded) | 14 |
| v06c | … refused with no_coverage (actions not recorded) | 12 |
| v06c | … refused with not_monday (actions not recorded) | 1 |
| v06c | … refused with not_on_shift (actions not recorded) | 33 |
| v06c | … refused with notice_too_short (actions not recorded) | 2 |
| v06c | … refused with request_conflict (actions not recorded) | 1 |
| v06c | … refused with wrong_role (actions not recorded) | 2 |
| v06c | affordability not established (evidence_unavailable) | 169 |
| v07c | earlier attempts not recorded (llm-agent 2) | 150 |
| v07c | … refused with after_close (actions not recorded) | 59 |
| v07c | … refused with already_terminating (actions not recorded) | 1 |
| v07c | … refused with batch_rejected (actions not recorded) | 16 |
| v07c | … refused with contract_overlap (actions not recorded) | 9 |
| v07c | … refused with exceeds_max_shift (actions not recorded) | 11 |
| v07c | … refused with invalid_minutes (actions not recorded) | 3 |
| v07c | … refused with invalid_roster (actions not recorded) | 1 |
| v07c | … refused with invalid_schema (actions not recorded) | 10 |
| v07c | … refused with lead_too_short (actions not recorded) | 10 |
| v07c | … refused with no_coverage (actions not recorded) | 10 |
| v07c | … refused with not_on_shift (actions not recorded) | 39 |
| v07c | … refused with notice_too_short (actions not recorded) | 5 |
| v07c | … refused with past_date (actions not recorded) | 5 |
| v07c | affordability not established (evidence_unavailable) | 1 |
| v08c | earlier attempts not recorded (llm-agent 2) | 0 |
| v08c | affordability not established (evidence_unavailable) | 2 |
