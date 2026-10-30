# v07c-stage-b: suite dreamteam-dev version 3, seeds 41701, 90 days, engine 0.7.0-dev.3, status complete (25 of 25 runs, 0 failed)

## 1. Starting states (identical across the policies of each entry and seed)

| entry | seed | identical |
|---|---|---|
| financial-pressure-credit | 41701 | yes |
| reporting-backlog-credit | 41701 | yes |
| demand-growth-credit | 41701 | yes |
| staffing-shortage-credit | 41701 | yes |
| stable-credit | 41701 | yes |

## 2. Outcomes (absolute, each run)

Overtime cost is the harness's (all staff, before benefits); approved is the part from extend_shift. Most drawn is the credit line's largest end-of-day balance; shortfall is the largest unfunded amount (cash below zero with the line fully drawn).

| entry | seed | policy | status | replay | profit | revenue | expenses | TAT p90 | breaches | exams | lost | overtime | overtime cost | approved overtime cost | cash end | min cash | most drawn | interest | shortfall |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41701 | noop | completed | yes | -$38,654 | $698,029 | $736,683 | 925 | 355 | 4,387 | 494 | 8,897 | $13,967 | $0 | $41,798 | $24,466 | $0 | $0 | $0 |
| financial-pressure-credit | 41701 | greedy | completed | yes | -$71,219 | $698,040 | $769,259 | 762 | 269 | 4,387 | 494 | 8,897 | $13,967 | $0 | $13,164 | $5,602 | $0 | $0 | $0 |
| financial-pressure-credit | 41701 | conservative-v2 | completed | yes | -$38,654 | $698,029 | $736,683 | 925 | 355 | 4,387 | 494 | 8,897 | $13,967 | $0 | $41,798 | $24,466 | $0 | $0 | $0 |
| financial-pressure-credit | 41701 | qwen36-35b-a3b | completed | yes | -$57,803 | $699,144 | $756,947 | 893 | 323 | 4,387 | 494 | 11,621 | $24,444 | $10,154 | $25,459 | $18,168 | $0 | $0 | $0 |
| financial-pressure-credit | 41701 | gemma4-26b-a4b | completed | yes | -$38,654 | $698,029 | $736,683 | 925 | 355 | 4,387 | 494 | 8,897 | $13,967 | $0 | $41,798 | $24,466 | $0 | $0 | $0 |
| reporting-backlog-credit | 41701 | noop | completed | yes | -$2,904 | $750,360 | $753,263 | 1,190 | 593 | 4,700 | 749 | 12,038 | $19,979 | $0 | $82,056 | $45,163 | $0 | $0 | $0 |
| reporting-backlog-credit | 41701 | greedy | completed | yes | -$36,453 | $750,767 | $787,220 | 847 | 360 | 4,700 | 749 | 12,038 | $19,979 | $0 | $53,348 | $39,081 | $0 | $0 | $0 |
| reporting-backlog-credit | 41701 | conservative-v2 | completed | yes | -$2,904 | $750,360 | $753,263 | 1,190 | 593 | 4,700 | 749 | 12,038 | $19,979 | $0 | $82,056 | $45,163 | $0 | $0 | $0 |
| reporting-backlog-credit | 41701 | qwen36-35b-a3b | completed | yes | -$10,910 | $751,305 | $762,215 | 1,155 | 555 | 4,711 | 741 | 14,587 | $26,843 | $7,147 | $73,834 | $37,041 | $0 | $0 | $0 |
| reporting-backlog-credit | 41701 | gemma4-26b-a4b | completed | yes | -$25,000 | $751,948 | $776,948 | 904 | 253 | 4,700 | 749 | 12,038 | $19,979 | $0 | $66,160 | $40,165 | $0 | $0 | $0 |
| demand-growth-credit | 41701 | noop | completed | yes | -$24 | $758,231 | $758,255 | 1,325 | 656 | 4,839 | 997 | 13,683 | $22,503 | $0 | $82,234 | $45,108 | $0 | $0 | $0 |
| demand-growth-credit | 41701 | greedy | completed | yes | -$29,047 | $763,121 | $792,168 | 867 | 339 | 4,839 | 997 | 13,683 | $22,503 | $0 | $52,626 | $39,026 | $0 | $0 | $0 |
| demand-growth-credit | 41701 | conservative-v2 | completed | yes | -$24 | $758,231 | $758,255 | 1,325 | 656 | 4,839 | 997 | 13,683 | $22,503 | $0 | $82,234 | $45,108 | $0 | $0 | $0 |
| demand-growth-credit | 41701 | qwen36-35b-a3b | completed | yes | -$20,939 | $764,999 | $785,938 | 992 | 352 | 4,845 | 990 | 14,711 | $24,860 | $2,397 | $58,315 | $38,959 | $0 | $0 | $0 |
| demand-growth-credit | 41701 | gemma4-26b-a4b | completed | yes | -$5,947 | $768,369 | $774,317 | 1,145 | 548 | 4,874 | 969 | 16,297 | $24,378 | $2,562 | $71,084 | $42,957 | $0 | $0 | $0 |
| staffing-shortage-credit | 41701 | noop | completed | yes | -$33,007 | $681,949 | $714,956 | 785 | 269 | 4,471 | 1,029 | 11,538 | $19,544 | $0 | $59,585 | $37,228 | $0 | $0 | $0 |
| staffing-shortage-credit | 41701 | greedy | completed | yes | -$65,629 | $681,897 | $747,526 | 695 | 235 | 4,471 | 1,029 | 11,538 | $19,544 | $0 | $30,818 | $24,153 | $0 | $0 | $0 |
| staffing-shortage-credit | 41701 | conservative-v2 | completed | yes | -$33,007 | $681,949 | $714,956 | 785 | 269 | 4,471 | 1,029 | 11,538 | $19,544 | $0 | $59,585 | $37,228 | $0 | $0 | $0 |
| staffing-shortage-credit | 41701 | qwen36-35b-a3b | completed | yes | -$214,596 | $543,434 | $758,030 | 150 | 190 | 3,604 | 2,479 | 9,035 | $17,326 | $4,786 | $20,049 | $0 | $17,200 | $14 | $0 |
| staffing-shortage-credit | 41701 | gemma4-26b-a4b | completed | yes | -$36,694 | $694,616 | $731,310 | 768 | 163 | 4,525 | 976 | 17,381 | $24,162 | $5,164 | $50,133 | $30,602 | $0 | $0 | $0 |
| stable-credit | 41701 | noop | completed | yes | $14,685 | $771,580 | $756,894 | 917 | 354 | 4,878 | 595 | 8,534 | $14,005 | $0 | $107,024 | $47,046 | $0 | $0 | $0 |
| stable-credit | 41701 | greedy | completed | yes | -$17,798 | $771,690 | $789,487 | 737 | 278 | 4,878 | 595 | 8,534 | $14,005 | $0 | $78,726 | $47,046 | $0 | $0 | $0 |
| stable-credit | 41701 | conservative-v2 | completed | yes | $11,973 | $771,580 | $759,606 | 889 | 342 | 4,878 | 595 | 8,534 | $14,005 | $0 | $104,311 | $47,046 | $0 | $0 | $0 |
| stable-credit | 41701 | qwen36-35b-a3b | completed | yes | -$36,875 | $773,376 | $810,251 | 112 | 55 | 4,882 | 595 | 8,892 | $15,581 | $1,615 | $61,141 | $37,224 | $0 | $0 | $0 |
| stable-credit | 41701 | gemma4-26b-a4b | completed | yes | $14,685 | $771,580 | $756,894 | 917 | 354 | 4,878 | 595 | 8,534 | $14,005 | $0 | $107,024 | $47,046 | $0 | $0 | $0 |

## 3. Paired with no action (policy − noop, same entry and seed; minutes for times)

| entry | seed | policy | Δprofit | Δrevenue | Δexpenses | ΔTAT p90 | Δbreaches | Δexams | Δlost | Δovertime | Δovertime cost | Δcash end | Δmin cash | flag |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41701 | greedy | -$32,565 | +$11 | +$32,576 | -162 | -86 | 0 | 0 | 0 | $0 | -$28,634 | -$18,863 | better service, lower profit |
| financial-pressure-credit | 41701 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| financial-pressure-credit | 41701 | qwen36-35b-a3b | -$19,149 | +$1,115 | +$20,263 | -32 | -32 | 0 | 0 | +2,724 | +$10,477 | -$16,339 | -$6,298 | better service, lower profit |
| financial-pressure-credit | 41701 | gemma4-26b-a4b | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| reporting-backlog-credit | 41701 | greedy | -$33,550 | +$407 | +$33,957 | -343 | -233 | 0 | 0 | 0 | $0 | -$28,708 | -$6,082 | better service, lower profit |
| reporting-backlog-credit | 41701 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| reporting-backlog-credit | 41701 | qwen36-35b-a3b | -$8,007 | +$945 | +$8,952 | -35 | -38 | +11 | -8 | +2,549 | +$6,865 | -$8,222 | -$8,122 | better service, lower profit |
| reporting-backlog-credit | 41701 | gemma4-26b-a4b | -$22,096 | +$1,589 | +$23,685 | -285 | -340 | 0 | 0 | 0 | $0 | -$15,896 | -$4,998 | better service, lower profit |
| demand-growth-credit | 41701 | greedy | -$29,022 | +$4,891 | +$33,913 | -458 | -317 | 0 | 0 | 0 | $0 | -$29,608 | -$6,082 | better service, lower profit |
| demand-growth-credit | 41701 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| demand-growth-credit | 41701 | qwen36-35b-a3b | -$20,914 | +$6,768 | +$27,682 | -333 | -304 | +6 | -7 | +1,028 | +$2,357 | -$23,919 | -$6,149 | better service, lower profit |
| demand-growth-credit | 41701 | gemma4-26b-a4b | -$5,923 | +$10,139 | +$16,061 | -180 | -108 | +35 | -28 | +2,614 | +$1,875 | -$11,149 | -$2,152 | better service, lower profit |
| staffing-shortage-credit | 41701 | greedy | -$32,622 | -$52 | +$32,570 | -90 | -34 | 0 | 0 | 0 | $0 | -$28,767 | -$13,075 | better service, lower profit |
| staffing-shortage-credit | 41701 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| staffing-shortage-credit | 41701 | qwen36-35b-a3b | -$181,589 | -$138,515 | +$43,074 | -635 | -79 | -867 | +1,450 | -2,503 | -$2,218 | -$39,536 | -$37,228 | better service, lower profit |
| staffing-shortage-credit | 41701 | gemma4-26b-a4b | -$3,687 | +$12,668 | +$16,355 | -17 | -106 | +54 | -53 | +5,843 | +$4,617 | -$9,452 | -$6,627 | better service, lower profit |
| stable-credit | 41701 | greedy | -$32,483 | +$110 | +$32,593 | -180 | -76 | 0 | 0 | 0 | $0 | -$28,297 | $0 | better service, lower profit |
| stable-credit | 41701 | conservative-v2 | -$2,712 | $0 | +$2,712 | -28 | -12 | 0 | 0 | 0 | $0 | -$2,712 | $0 | better service, lower profit |
| stable-credit | 41701 | qwen36-35b-a3b | -$51,560 | +$1,796 | +$53,357 | -805 | -299 | +4 | 0 | +358 | +$1,575 | -$45,883 | -$9,823 | better service, lower profit |
| stable-credit | 41701 | gemma4-26b-a4b | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |

## 4. Actions: proposed, refused, accepted, executed

Proposed counts each action each time it was put forward (each validate attempt; the built-in policies' submissions); `batch_rejected` is counted apart. Executed: accepted and in effect by the end of the window. Funds: refused insufficient_funds (at validate / at apply).

| entry | seed | policy | proposed | refused by validate (+batch) | submitted | accepted | refused at apply (+batch) | executed | executed ops/staffing | funds v/a | accepted by type |
|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41701 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| financial-pressure-credit | 41701 | greedy | 12 | 0 (+0) | 12 | 12 | 0 (+0) | 11 | 0/11 | 0/0 | set_reading_allocation: 12 |
| financial-pressure-credit | 41701 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| financial-pressure-credit | 41701 | qwen36-35b-a3b | 27 | 2 (+0) | 25 | 25 | 0 (+0) | 25 | 22/3 | 0/0 | contract_teleradiology: 1; extend_shift: 22; set_reading_allocation: 2 |
| financial-pressure-credit | 41701 | gemma4-26b-a4b | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| reporting-backlog-credit | 41701 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| reporting-backlog-credit | 41701 | greedy | 12 | 0 (+0) | 12 | 12 | 0 (+0) | 11 | 0/11 | 0/0 | set_reading_allocation: 12 |
| reporting-backlog-credit | 41701 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| reporting-backlog-credit | 41701 | qwen36-35b-a3b | 21 | 4 (+0) | 16 | 16 | 0 (+0) | 16 | 16/0 | 0/0 | extend_shift: 14; set_walkin_policy: 1; set_worklist_policy: 1 |
| reporting-backlog-credit | 41701 | gemma4-26b-a4b | 8 | 4 (+1) | 3 | 3 | 0 (+0) | 3 | 0/3 | 0/0 | contract_teleradiology: 3 |
| demand-growth-credit | 41701 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| demand-growth-credit | 41701 | greedy | 12 | 0 (+0) | 12 | 12 | 0 (+0) | 11 | 0/11 | 0/0 | set_reading_allocation: 12 |
| demand-growth-credit | 41701 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| demand-growth-credit | 41701 | qwen36-35b-a3b | 19 | 1 (+0) | 18 | 18 | 0 (+0) | 18 | 16/2 | 0/0 | contract_teleradiology: 1; extend_shift: 15; set_reading_allocation: 1; set_worklist_policy: 1 |
| demand-growth-credit | 41701 | gemma4-26b-a4b | 39 | 9 (+0) | 30 | 30 | 0 (+0) | 28 | 27/1 | 0/0 | contract_teleradiology: 3; extend_shift: 27 |
| staffing-shortage-credit | 41701 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| staffing-shortage-credit | 41701 | greedy | 12 | 0 (+0) | 12 | 12 | 0 (+0) | 11 | 0/11 | 0/0 | set_reading_allocation: 12 |
| staffing-shortage-credit | 41701 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| staffing-shortage-credit | 41701 | qwen36-35b-a3b | 62 | 9 (+10) | 42 | 42 | 0 (+0) | 40 | 20/20 | 0/0 | extend_shift: 14; hire: 4; set_reading_allocation: 2; set_worklist_policy: 6; terminate: 16 |
| staffing-shortage-credit | 41701 | gemma4-26b-a4b | 79 | 25 (+0) | 54 | 54 | 0 (+0) | 54 | 53/1 | 0/0 | contract_teleradiology: 1; extend_shift: 53 |
| stable-credit | 41701 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| stable-credit | 41701 | greedy | 12 | 0 (+0) | 12 | 12 | 0 (+0) | 11 | 0/11 | 0/0 | set_reading_allocation: 12 |
| stable-credit | 41701 | conservative-v2 | 2 | 0 (+0) | 2 | 2 | 0 (+0) | 2 | 0/2 | 0/0 | set_reading_allocation: 2 |
| stable-credit | 41701 | qwen36-35b-a3b | 14 | 7 (+0) | 5 | 5 | 0 (+0) | 5 | 3/2 | 0/0 | contract_teleradiology: 1; extend_shift: 2; hire: 1; set_walkin_policy: 1 |
| stable-credit | 41701 | gemma4-26b-a4b | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |

## 5. Refused for funds, and what followed

Each action refused insufficient_funds; later in the run, the decision points at which the policy put forward the same type for the same target, how many of those were refused for funds again, and how many were accepted.

None: no action was refused insufficient_funds.

## 6. LLM decisions

| entry | seed | policy | points | with actions | no action | fallback | first-attempt valid | repeated invalid | details | infrastructure errors |
|---|---|---|---|---|---|---|---|---|---|---|
| demand-growth-credit | 41701 | gemma4-26b-a4b | 90 | 15 | 74 | 1 | 80/90 | 8 | 0 | 0 |
| demand-growth-credit | 41701 | qwen36-35b-a3b | 90 | 11 | 78 | 1 | 88/90 | 0 | 0 | 0 |
| financial-pressure-credit | 41701 | gemma4-26b-a4b | 90 | 0 | 90 | 0 | 90/90 | 0 | 0 | 0 |
| financial-pressure-credit | 41701 | qwen36-35b-a3b | 90 | 24 | 64 | 2 | 85/90 | 1 | 0 | 0 |
| reporting-backlog-credit | 41701 | gemma4-26b-a4b | 90 | 3 | 84 | 3 | 85/90 | 4 | 0 | 0 |
| reporting-backlog-credit | 41701 | qwen36-35b-a3b | 90 | 9 | 79 | 2 | 81/90 | 4 | 0 | 0 |
| stable-credit | 41701 | gemma4-26b-a4b | 90 | 0 | 90 | 0 | 90/90 | 0 | 0 | 0 |
| stable-credit | 41701 | qwen36-35b-a3b | 90 | 4 | 84 | 2 | 81/90 | 5 | 0 | 0 |
| staffing-shortage-credit | 41701 | gemma4-26b-a4b | 90 | 44 | 43 | 3 | 63/90 | 27 | 0 | 0 |
| staffing-shortage-credit | 41701 | qwen36-35b-a3b | 90 | 21 | 65 | 4 | 76/90 | 13 | 0 | 0 |

## 7. Refusals by validate, and agent errors

| entry | seed | policy | refusals (type code: count) | errors (code: count) |
|---|---|---|---|---|
| demand-growth-credit | 41701 | gemma4-26b-a4b | contract_teleradiology contract_overlap: 1; contract_teleradiology invalid_schema: 2; contract_teleradiology notice_too_short: 1; extend_shift after_close: 2; extend_shift not_on_shift: 8 | invalid_decision: 1; refused: 10 |
| demand-growth-credit | 41701 | qwen36-35b-a3b | extend_shift after_close: 1; extend_shift no_coverage: 1; extend_shift not_on_shift: 3 | refused: 3 |
| financial-pressure-credit | 41701 | gemma4-26b-a4b | none | none |
| financial-pressure-credit | 41701 | qwen36-35b-a3b | contract_teleradiology batch_rejected: 1; extend_shift after_close: 1; extend_shift invalid_minutes: 1; extend_shift no_coverage: 2; set_reading_allocation notice_too_short: 1 | invalid_decision: 2; refused: 5 |
| reporting-backlog-credit | 41701 | gemma4-26b-a4b | contract_teleradiology batch_rejected: 2; contract_teleradiology contract_overlap: 4; contract_teleradiology invalid_schema: 3; extend_shift after_close: 1 | refused: 8 |
| reporting-backlog-credit | 41701 | qwen36-35b-a3b | extend_shift after_close: 5; extend_shift exceeds_max_shift: 2; extend_shift no_coverage: 1; extend_shift not_on_shift: 1 | invalid_decision: 3; invalid_json: 1; refused: 7 |
| stable-credit | 41701 | gemma4-26b-a4b | none | none |
| stable-credit | 41701 | qwen36-35b-a3b | contract_teleradiology invalid_schema: 1; extend_shift after_close: 4; extend_shift exceeds_max_shift: 2; extend_shift invalid_minutes: 1; hire lead_too_short: 3; set_reading_allocation past_date: 1 | invalid_decision: 2; refused: 9 |
| staffing-shortage-credit | 41701 | gemma4-26b-a4b | contract_teleradiology invalid_schema: 1; extend_shift after_close: 12; extend_shift not_on_shift: 17; hire invalid_roster: 1; hire lead_too_short: 2 | invalid_action: 1; refused: 29 |
| staffing-shortage-credit | 41701 | qwen36-35b-a3b | extend_shift after_close: 5; extend_shift not_on_shift: 3; hire batch_rejected: 6; hire lead_too_short: 6; hire request_conflict: 1; set_reading_allocation batch_rejected: 2; set_reading_allocation past_date: 1; terminate already_terminating: 1; terminate batch_rejected: 5 | invalid_action: 3; invalid_decision: 1; refused: 14 |

## 8. LLM actions in time order, with cash and credit

Every action an LLM agent submitted, with the outcome, and the cash and credit drawn at the end of the day before; and the agent's recorded reason at each point where an action was refused for funds.


### demand-growth-credit 41701 gemma4-26b-a4b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-11-08 | contract_teleradiology effective=2026-11-22, op=start, plan=standard-v1, requestId=telerad-backlog-… | accepted | yes | $100,490 | $0 |
| 2026-12-13 | contract_teleradiology contract=T00001, endDate=2027-01-12, op=cancel | accepted | no | $84,816 | $0 |
| 2026-12-13 | contract_teleradiology effective=2027-02-11, op=start, plan=standard-v1, requestId=telerad-sunday-f… | accepted | no | $84,816 | $0 |
| 2026-12-14 | extend_shift employee=E03, minutes=120 | accepted | yes | $84,816 | $0 |
| 2026-12-14 | extend_shift employee=E05, minutes=120 | accepted | yes | $84,816 | $0 |
| 2026-12-16 | extend_shift employee=E03, minutes=120 | accepted | yes | $120,436 | $0 |
| 2026-12-16 | extend_shift employee=E05, minutes=120 | accepted | yes | $120,436 | $0 |
| 2026-12-17 | extend_shift employee=E03, minutes=120 | accepted | yes | $130,821 | $0 |
| 2026-12-17 | extend_shift employee=E05, minutes=120 | accepted | yes | $130,821 | $0 |
| 2026-12-18 | extend_shift employee=E03, minutes=120 | accepted | yes | $137,217 | $0 |
| 2026-12-18 | extend_shift employee=E05, minutes=120 | accepted | yes | $137,217 | $0 |
| 2026-12-23 | extend_shift employee=E03, minutes=120 | accepted | yes | $180,735 | $0 |
| 2026-12-23 | extend_shift employee=E05, minutes=120 | accepted | yes | $180,735 | $0 |
| 2026-12-24 | extend_shift employee=E03, minutes=120 | accepted | yes | $188,597 | $0 |
| 2026-12-24 | extend_shift employee=E05, minutes=120 | accepted | yes | $188,597 | $0 |
| 2026-12-25 | extend_shift employee=E03, minutes=120 | accepted | yes | $196,877 | $0 |
| 2026-12-25 | extend_shift employee=E05, minutes=120 | accepted | yes | $196,877 | $0 |
| 2026-12-30 | extend_shift employee=E03, minutes=120 | accepted | yes | $182,904 | $0 |
| 2026-12-30 | extend_shift employee=E05, minutes=120 | accepted | yes | $182,904 | $0 |
| 2026-12-31 | extend_shift employee=E03, minutes=120 | accepted | yes | $191,153 | $0 |
| 2026-12-31 | extend_shift employee=E05, minutes=120 | accepted | yes | $191,153 | $0 |
| 2027-01-01 | extend_shift employee=E03, minutes=120 | accepted | yes | $196,962 | $0 |
| 2027-01-01 | extend_shift employee=E05, minutes=120 | accepted | yes | $196,962 | $0 |
| 2027-01-06 | extend_shift employee=E03, minutes=120 | accepted | yes | $99,498 | $0 |
| 2027-01-06 | extend_shift employee=E05, minutes=120 | accepted | yes | $99,498 | $0 |
| 2027-01-07 | extend_shift employee=E03, minutes=120 | accepted | yes | $109,575 | $0 |
| 2027-01-07 | extend_shift employee=E05, minutes=120 | accepted | yes | $109,575 | $0 |
| 2027-01-08 | extend_shift employee=E03, minutes=120 | accepted | yes | $123,156 | $0 |
| 2027-01-08 | extend_shift employee=E05, minutes=120 | accepted | yes | $123,156 | $0 |
| 2027-01-08 | extend_shift employee=E06, minutes=120 | accepted | yes | $123,156 | $0 |

### demand-growth-credit 41701 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-15 | extend_shift date=2026-10-15, employee=E03, minutes=60 | accepted | yes | $162,048 | $0 |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E06, minutes=60 | accepted | yes | $162,048 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E03, minutes=60 | accepted | yes | $169,500 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E06, minutes=60 | accepted | yes | $169,500 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E05, minutes=60 | accepted | yes | $169,500 | $0 |
| 2026-10-18 | set_worklist_policy policy=oldest_first | accepted | yes | $120,071 | $0 |
| 2026-10-18 | contract_teleradiology effective=2026-11-01, op=start, plan=standard-v1, requestId=telerad-sun-back… | accepted | yes | $120,071 | $0 |
| 2026-10-19 | extend_shift date=2026-10-19, employee=E03, minutes=60 | accepted | yes | $120,071 | $0 |
| 2026-10-19 | extend_shift date=2026-10-19, employee=E05, minutes=60 | accepted | yes | $120,071 | $0 |
| 2026-11-19 | extend_shift date=2026-11-19, employee=E03, minutes=60 | accepted | yes | $137,699 | $0 |
| 2026-11-19 | extend_shift date=2026-11-19, employee=E05, minutes=60 | accepted | yes | $137,699 | $0 |
| 2026-11-20 | extend_shift date=2026-11-20, employee=E03, minutes=60 | accepted | yes | $144,459 | $0 |
| 2026-11-20 | extend_shift date=2026-11-20, employee=E05, minutes=60 | accepted | yes | $144,459 | $0 |
| 2026-12-04 | extend_shift date=2026-12-04, employee=E07, minutes=120 | accepted | yes | $62,901 | $0 |
| 2026-12-10 | extend_shift date=2026-12-10, employee=E07, minutes=120 | accepted | yes | $118,999 | $0 |
| 2026-12-11 | extend_shift date=2026-12-11, employee=E07, minutes=120 | accepted | yes | $128,309 | $0 |
| 2026-12-12 | set_reading_allocation effective=2026-12-26, employee=E07, op=set, siteAllocationBP=10000 | accepted | yes | $78,380 | $0 |
| 2026-12-18 | extend_shift date=2026-12-18, employee=E07, minutes=120 | accepted | yes | $130,579 | $0 |

### financial-pressure-credit 41701 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-14 | extend_shift employee=E07, minutes=120 | accepted | yes | $155,700 | $0 |
| 2026-10-15 | extend_shift employee=E07, minutes=120 | accepted | yes | $160,982 | $0 |
| 2026-10-16 | extend_shift employee=E07, minutes=120 | accepted | yes | $166,285 | $0 |
| 2026-10-19 | extend_shift employee=E07, minutes=120 | accepted | yes | $116,051 | $0 |
| 2026-11-23 | extend_shift employee=E07, minutes=120 | accepted | yes | $132,011 | $0 |
| 2026-11-25 | extend_shift employee=E07, minutes=120 | accepted | yes | $159,980 | $0 |
| 2026-11-26 | extend_shift employee=E07, minutes=120 | accepted | yes | $167,356 | $0 |
| 2026-11-27 | extend_shift employee=E07, minutes=120 | accepted | yes | $176,409 | $0 |
| 2026-11-30 | extend_shift employee=E07, minutes=120 | accepted | yes | $126,584 | $0 |
| 2026-12-03 | extend_shift employee=E07, minutes=120 | accepted | yes | $34,079 | $0 |
| 2026-12-04 | extend_shift employee=E07, minutes=120 | accepted | yes | $42,244 | $0 |
| 2026-12-07 | extend_shift employee=E07, minutes=120 | accepted | yes | $53,199 | $0 |
| 2026-12-09 | extend_shift employee=E07, minutes=120 | accepted | yes | $84,632 | $0 |
| 2026-12-10 | extend_shift employee=E07, minutes=120 | accepted | yes | $94,741 | $0 |
| 2026-12-11 | extend_shift employee=E07, minutes=120 | accepted | yes | $103,010 | $0 |
| 2026-12-14 | extend_shift employee=E07, minutes=120 | accepted | yes | $52,355 | $0 |
| 2026-12-15 | extend_shift employee=E07, minutes=120 | accepted | yes | $75,547 | $0 |
| 2026-12-16 | extend_shift employee=E07, minutes=120 | accepted | yes | $83,886 | $0 |
| 2026-12-17 | extend_shift employee=E07, minutes=120 | accepted | yes | $95,066 | $0 |
| 2026-12-18 | extend_shift employee=E07, minutes=120 | accepted | yes | $100,398 | $0 |
| 2026-12-20 | set_reading_allocation effective=2027-01-03, employee=E07, op=set, siteAllocationBP=10000 | accepted | yes | $108,856 | $0 |
| 2026-12-20 | contract_teleradiology effective=2027-01-03, op=start, plan=standard-v1, requestId=start-sunday-tel… | accepted | yes | $108,856 | $0 |
| 2026-12-21 | set_reading_allocation effective=2027-01-04, employee=E07, op=set, siteAllocationBP=10000 | unchanged | yes | $108,856 | $0 |
| 2027-01-06 | extend_shift employee=E07, minutes=120 | accepted | yes | $59,364 | $0 |
| 2027-01-07 | extend_shift employee=E07, minutes=120 | accepted | yes | $69,108 | $0 |

### reporting-backlog-credit 41701 gemma4-26b-a4b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-19 | contract_teleradiology effective=2026-11-02, op=start, plan=standard-v1, requestId=telerad-sun-001,… | accepted | yes | $120,131 | $0 |
| 2026-11-05 | contract_teleradiology contract=T00001, endDate=2026-12-05, op=cancel | accepted | yes | $83,040 | $0 |
| 2026-11-09 | contract_teleradiology effective=2026-12-06, op=start, plan=standard-v1, requestId=telerad-expanded… | accepted | yes | $100,490 | $0 |

### reporting-backlog-credit 41701 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-13 | extend_shift date=2026-10-13, employee=E07, minutes=210 | accepted | yes | – | – |
| 2026-10-13 | extend_shift date=2026-10-13, employee=E03, minutes=195 | accepted | yes | – | – |
| 2026-10-13 | extend_shift date=2026-10-13, employee=E05, minutes=210 | accepted | yes | – | – |
| 2026-10-13 | extend_shift date=2026-10-13, employee=E06, minutes=210 | accepted | yes | – | – |
| 2026-10-13 | extend_shift date=2026-10-13, employee=E01, minutes=195 | accepted | yes | – | – |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E07, minutes=210 | accepted | yes | $162,048 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E07, minutes=210 | accepted | yes | $169,500 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E06, minutes=210 | accepted | yes | $169,500 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E03, minutes=210 | accepted | yes | $169,500 | $0 |
| 2026-10-18 | set_worklist_policy policy=oldest_first | accepted | yes | $118,132 | $0 |
| 2026-10-18 | set_walkin_policy policy=accept_if_gap | unchanged | yes | $118,132 | $0 |
| 2026-10-21 | extend_shift date=2026-10-21, employee=E07, minutes=210 | accepted | yes | $145,970 | $0 |
| 2026-10-22 | extend_shift date=2026-10-22, employee=E07, minutes=210 | accepted | yes | $152,449 | $0 |
| 2026-10-23 | extend_shift date=2026-10-23, employee=E07, minutes=210 | accepted | yes | $160,022 | $0 |
| 2026-10-29 | extend_shift date=2026-10-29, employee=E07, minutes=180 | accepted | yes | $211,183 | $0 |
| 2026-11-13 | extend_shift date=2026-11-13, employee=E07, minutes=180 | accepted | yes | $141,280 | $0 |

### stable-credit 41701 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-14 | contract_teleradiology effective=2026-10-28, op=start, plan=standard-v1, requestId=tr-2026-10-14-al… | accepted | yes | $157,642 | $0 |
| 2026-10-16 | extend_shift employee=E07, minutes=210 | accepted | yes | $169,904 | $0 |
| 2026-10-19 | extend_shift employee=E07, minutes=210 | accepted | yes | $120,288 | $0 |
| 2026-11-08 | set_walkin_policy policy=accept_always | accepted | yes | $99,395 | $0 |
| 2026-11-08 | hire contractedWeeklyMinutes=2400, rateCents=4900, requestId=hire-mr-premium-20261108-v2, role=tech… | accepted | yes | $99,395 | $0 |

### staffing-shortage-credit 41701 gemma4-26b-a4b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-15 | extend_shift employee=E05, minutes=120 | accepted | yes | $162,681 | $0 |
| 2026-10-15 | extend_shift employee=E06, minutes=120 | accepted | yes | $162,681 | $0 |
| 2026-10-16 | extend_shift employee=E05, minutes=120 | accepted | yes | $169,803 | $0 |
| 2026-10-16 | extend_shift employee=E06, minutes=120 | accepted | yes | $169,803 | $0 |
| 2026-10-18 | contract_teleradiology effective=2026-11-01, op=start, plan=standard-v1, requestId=telerad-sun-01, … | accepted | yes | $123,819 | $0 |
| 2026-10-21 | extend_shift employee=E06, minutes=120 | accepted | yes | $149,160 | $0 |
| 2026-10-22 | extend_shift employee=E06, minutes=120 | accepted | yes | $155,963 | $0 |
| 2026-10-23 | extend_shift employee=E06, minutes=120 | accepted | yes | $164,033 | $0 |
| 2026-10-26 | extend_shift employee=E06, minutes=120 | accepted | yes | $171,361 | $0 |
| 2026-10-29 | extend_shift employee=E06, minutes=120 | accepted | yes | $209,043 | $0 |
| 2026-10-30 | extend_shift employee=E06, minutes=120 | accepted | yes | $215,920 | $0 |
| 2026-11-04 | extend_shift employee=E06, minutes=120 | accepted | yes | $70,522 | $0 |
| 2026-11-05 | extend_shift employee=E06, minutes=120 | accepted | yes | $78,532 | $0 |
| 2026-11-06 | extend_shift employee=E06, minutes=120 | accepted | yes | $85,260 | $0 |
| 2026-11-09 | extend_shift employee=E06, minutes=120 | accepted | yes | $94,545 | $0 |
| 2026-11-10 | extend_shift employee=E06, minutes=120 | accepted | yes | $116,016 | $0 |
| 2026-11-12 | extend_shift employee=E06, minutes=120 | accepted | yes | $131,038 | $0 |
| 2026-11-13 | extend_shift employee=E06, minutes=120 | accepted | yes | $138,008 | $0 |
| 2026-11-16 | extend_shift employee=E05, minutes=120 | accepted | yes | $91,825 | $0 |
| 2026-11-18 | extend_shift employee=E06, minutes=120 | accepted | yes | $119,491 | $0 |
| 2026-11-19 | extend_shift employee=E06, minutes=120 | accepted | yes | $126,942 | $0 |
| 2026-11-20 | extend_shift employee=E06, minutes=120 | accepted | yes | $132,082 | $0 |
| 2026-11-23 | extend_shift employee=E05, minutes=120 | accepted | yes | $140,936 | $0 |
| 2026-11-24 | extend_shift employee=E05, minutes=120 | accepted | yes | $162,407 | $0 |
| 2026-11-26 | extend_shift employee=E05, minutes=120 | accepted | yes | $178,395 | $0 |
| 2026-11-27 | extend_shift employee=E05, minutes=120 | accepted | yes | $187,948 | $0 |
| 2026-11-27 | extend_shift employee=E06, minutes=120 | accepted | yes | $187,948 | $0 |
| 2026-11-30 | extend_shift employee=E05, minutes=120 | accepted | yes | $140,659 | $0 |
| 2026-12-01 | extend_shift employee=E05, minutes=120 | accepted | yes | $159,626 | $0 |
| 2026-12-02 | extend_shift employee=E05, minutes=120 | accepted | yes | $38,917 | $0 |
| 2026-12-02 | extend_shift employee=E06, minutes=120 | accepted | yes | $38,917 | $0 |
| 2026-12-03 | extend_shift employee=E05, minutes=120 | accepted | yes | $47,321 | $0 |
| 2026-12-03 | extend_shift employee=E06, minutes=120 | accepted | yes | $47,321 | $0 |
| 2026-12-04 | extend_shift employee=E05, minutes=120 | accepted | yes | $53,776 | $0 |
| 2026-12-04 | extend_shift employee=E06, minutes=120 | accepted | yes | $53,776 | $0 |
| 2026-12-07 | extend_shift employee=E05, minutes=120 | accepted | yes | $60,810 | $0 |
| 2026-12-07 | extend_shift employee=E06, minutes=120 | accepted | yes | $60,810 | $0 |
| 2026-12-08 | extend_shift employee=E05, minutes=120 | accepted | yes | $84,913 | $0 |
| 2026-12-08 | extend_shift employee=E06, minutes=120 | accepted | yes | $84,913 | $0 |
| 2026-12-09 | extend_shift employee=E05, minutes=120 | accepted | yes | $94,498 | $0 |
| 2026-12-09 | extend_shift employee=E06, minutes=120 | accepted | yes | $94,498 | $0 |
| 2026-12-10 | extend_shift employee=E05, minutes=120 | accepted | yes | $102,066 | $0 |
| 2026-12-10 | extend_shift employee=E06, minutes=120 | accepted | yes | $102,066 | $0 |
| 2026-12-11 | extend_shift employee=E05, minutes=120 | accepted | yes | $108,719 | $0 |
| 2026-12-22 | extend_shift employee=E05, minutes=120 | accepted | yes | $144,560 | $0 |
| 2026-12-28 | extend_shift employee=E05, minutes=120 | accepted | yes | $121,995 | $0 |
| 2026-12-29 | extend_shift employee=E05, minutes=120 | accepted | yes | $147,034 | $0 |
| 2026-12-30 | extend_shift employee=E05, minutes=120 | accepted | yes | $155,726 | $0 |
| 2026-12-31 | extend_shift employee=E05, minutes=120 | accepted | yes | $163,899 | $0 |
| 2027-01-01 | extend_shift employee=E05, minutes=120 | accepted | yes | $170,117 | $0 |
| 2027-01-05 | extend_shift employee=E05, minutes=120 | accepted | yes | $72,740 | $0 |
| 2027-01-06 | extend_shift employee=E05, minutes=120 | accepted | yes | $80,311 | $0 |
| 2027-01-07 | extend_shift employee=E05, minutes=120 | accepted | yes | $89,951 | $0 |
| 2027-01-08 | extend_shift employee=E05, minutes=120 | accepted | yes | $97,308 | $0 |

### staffing-shortage-credit 41701 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-14 | extend_shift date=2026-10-14, employee=E03, minutes=60 | accepted | yes | $157,202 | $0 |
| 2026-10-14 | hire contractedWeeklyMinutes=2400, rateCents=3800, requestId=hire-xrct-std-002, role=technologist, … | accepted | yes | $157,202 | $0 |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E03, minutes=60 | accepted | yes | $162,681 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E03, minutes=60 | accepted | yes | $169,803 | $0 |
| 2026-10-16 | set_reading_allocation effective=2026-10-30, employee=E08, op=set, siteAllocationBP=10000 | accepted | yes | $169,803 | $0 |
| 2026-10-19 | extend_shift date=2026-10-19, employee=E07, minutes=120 | accepted | yes | $123,983 | $0 |
| 2026-10-19 | set_worklist_policy policy=oldest_first | accepted | yes | $123,983 | $0 |
| 2026-10-23 | extend_shift date=2026-10-23, employee=E07, minutes=120 | accepted | yes | $164,197 | $0 |
| 2026-10-28 | set_worklist_policy policy=oldest_first | unchanged | yes | $200,645 | $0 |
| 2026-10-28 | extend_shift date=2026-10-29, employee=E07, minutes=120 | accepted | yes | $200,645 | $0 |
| 2026-10-30 | set_worklist_policy policy=oldest_first | unchanged | yes | $216,084 | $0 |
| 2026-10-30 | extend_shift date=2026-10-30, employee=E07, minutes=120 | accepted | yes | $216,084 | $0 |
| 2026-11-01 | set_worklist_policy policy=oldest_first | unchanged | yes | $169,231 | $0 |
| 2026-11-01 | extend_shift date=2026-11-02, employee=E07, minutes=120 | accepted | yes | $169,231 | $0 |
| 2026-11-02 | set_worklist_policy policy=oldest_first | unchanged | yes | $39,178 | $0 |
| 2026-11-02 | extend_shift date=2026-11-03, employee=E07, minutes=120 | accepted | yes | $39,178 | $0 |
| 2026-11-03 | extend_shift date=2026-11-04, employee=E07, minutes=120 | accepted | yes | $60,919 | $0 |
| 2026-11-03 | extend_shift date=2026-11-05, employee=E07, minutes=120 | accepted | yes | $60,919 | $0 |
| 2026-11-03 | extend_shift date=2026-11-06, employee=E07, minutes=120 | accepted | yes | $60,919 | $0 |
| 2026-11-03 | hire contractedWeeklyMinutes=2400, rateCents=4900, requestId=hire-mr-prem-003, role=technologist, r… | accepted | yes | $60,919 | $0 |
| 2026-11-09 | extend_shift date=2026-11-09, employee=E07, minutes=120 | accepted | yes | $90,904 | $0 |
| 2026-11-09 | extend_shift date=2026-11-09, employee=E05, minutes=60 | accepted | yes | $90,904 | $0 |
| 2026-11-09 | set_worklist_policy policy=urgent_then_fifo | accepted | yes | $90,904 | $0 |
| 2026-11-27 | terminate employee=H00002, lastDay=2026-12-11 | accepted | yes | $168,496 | $0 |
| 2026-11-27 | set_reading_allocation effective=2026-12-11, employee=E08, op=set, siteAllocationBP=5000 | accepted | yes | $168,496 | $0 |
| 2026-12-01 | terminate employee=H00001, lastDay=2026-12-15 | accepted | yes | $124,907 | $0 |
| 2026-12-02 | terminate employee=H00001, lastDay=2026-12-15 | unchanged | yes | $8,430 | $4,406 |
| 2026-12-03 | terminate employee=H00002, lastDay=2026-12-11 | unchanged | yes | $12,145 | $0 |
| 2026-12-04 | terminate employee=E01, lastDay=2026-12-18 | accepted | yes | $18,539 | $0 |
| 2026-12-05 | terminate employee=E05, lastDay=2026-12-19 | accepted | yes | $25,636 | $0 |
| 2026-12-06 | terminate employee=E03, lastDay=2026-12-20 | accepted | yes | $25,636 | $0 |
| 2026-12-07 | terminate employee=E04, lastDay=2026-12-21 | accepted | yes | $25,636 | $0 |
| 2026-12-07 | terminate employee=E06, lastDay=2026-12-21 | accepted | yes | $25,636 | $0 |
| 2026-12-11 | terminate employee=H00002, lastDay=2026-12-11 | unchanged | yes | $74,017 | $0 |
| 2026-12-11 | terminate employee=H00001, lastDay=2026-12-15 | unchanged | yes | $74,017 | $0 |
| 2026-12-11 | terminate employee=E01, lastDay=2026-12-18 | unchanged | yes | $74,017 | $0 |
| 2026-12-11 | terminate employee=E03, lastDay=2026-12-20 | unchanged | yes | $74,017 | $0 |
| 2026-12-11 | terminate employee=E04, lastDay=2026-12-21 | unchanged | yes | $74,017 | $0 |
| 2026-12-11 | terminate employee=E05, lastDay=2026-12-19 | unchanged | yes | $74,017 | $0 |
| 2026-12-11 | terminate employee=E06, lastDay=2026-12-21 | unchanged | yes | $74,017 | $0 |
| 2027-01-10 | hire contractedWeeklyMinutes=2400, rateCents=3800, requestId=hire-xrct-std-003, role=technologist, … | accepted | no | $20,049 | $0 |
| 2027-01-10 | hire contractedWeeklyMinutes=2400, rateCents=3800, requestId=hire-xrct-std-004, role=technologist, … | accepted | no | $20,049 | $0 |

## 9. Tokens, latency and cost

| entry | seed | policy | requests | retries | prompt tokens | completion tokens | estimated/unknown | latency mean s | max s | total s | estimated cost |
|---|---|---|---|---|---|---|---|---|---|---|---|
| demand-growth-credit | 41701 | gemma4-26b-a4b | 100 | 10 | 1,230,172 | 8,123 | 0/0 | 16.7 | 26.4 | 1,668 | none configured |
| demand-growth-credit | 41701 | qwen36-35b-a3b | 92 | 2 | 1,172,051 | 11,941 | 0/0 | 10.1 | 24.4 | 929 | none configured |
| financial-pressure-credit | 41701 | gemma4-26b-a4b | 90 | 0 | 961,141 | 5,998 | 0/0 | 6.3 | 7.0 | 563 | none configured |
| financial-pressure-credit | 41701 | qwen36-35b-a3b | 95 | 5 | 1,125,450 | 12,592 | 0/0 | 4.9 | 10.8 | 462 | none configured |
| reporting-backlog-credit | 41701 | gemma4-26b-a4b | 95 | 5 | 1,199,087 | 7,584 | 0/0 | 17.2 | 24.9 | 1,634 | none configured |
| reporting-backlog-credit | 41701 | qwen36-35b-a3b | 99 | 9 | 1,157,380 | 19,578 | 0/0 | 10.2 | 26.8 | 1,009 | none configured |
| stable-credit | 41701 | gemma4-26b-a4b | 90 | 0 | 984,235 | 6,201 | 0/0 | 15.4 | 18.5 | 1,390 | none configured |
| stable-credit | 41701 | qwen36-35b-a3b | 99 | 9 | 1,337,500 | 16,159 | 0/0 | 10.7 | 35.9 | 1,062 | none configured |
| staffing-shortage-credit | 41701 | gemma4-26b-a4b | 117 | 27 | 1,473,979 | 10,437 | 0/0 | 15.4 | 25.3 | 1,801 | none configured |
| staffing-shortage-credit | 41701 | qwen36-35b-a3b | 104 | 14 | 1,236,946 | 34,187 | 0/0 | 14.5 | 52.1 | 1,511 | none configured |
