# Financial and operational outcomes, paired with no action

`dreamteam.financial-outcomes/v1`. Each row compares one policy's run with the no-action run on the same entry, seed, horizon and economics. A difference describes two runs; it is not attributed to any one decision, and no row is a score. The suite runs are never combined. Money is rounded to the nearest dollar. Overtime cost, credit and executed actions come from the replays, which exist for the LLM runs and their no-action pairs only (— for the other policies); n/a: no credit line in that economics.

## v06c-compare-30: dreamteam-dev 1 (V0.6): overtime not costed, no credit line

Contract `docs/benchmark/v06c-dev/compare-30/contract.json`, engine 0.6.0-dev.3, 30 days.

### 1. Financial outcomes (policy minus no action)

| Entry | Seed | Policy | Profit | Revenue | Expense | Lowest cash (policy) | Lowest cash difference | Overtime cost | Most credit drawn | Interest |
|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure | 41201 | greedy | −$1,175 | +$58 | +$1,233 | $43,006 | $0 | — | n/a | n/a |
| financial-pressure | 41201 | conservative-v2 | $0 | $0 | $0 | $43,006 | $0 | — | n/a | n/a |
| financial-pressure | 41201 | qwen36-35b-a3b | $0 | $0 | $0 | $43,006 | $0 | $0 | n/a | n/a |
| financial-pressure | 41201 | gemma4-26b-a4b | $0 | $0 | $0 | $43,006 | $0 | $0 | n/a | n/a |
| financial-pressure | 41202 | greedy | −$635 | +$598 | +$1,233 | $47,878 | $0 | — | n/a | n/a |
| financial-pressure | 41202 | conservative-v2 | $0 | $0 | $0 | $47,878 | $0 | — | n/a | n/a |
| financial-pressure | 41202 | qwen36-35b-a3b | +$1 | +$68 | +$68 | $47,811 | −$68 | +$33 | n/a | n/a |
| financial-pressure | 41202 | gemma4-26b-a4b | $0 | $0 | $0 | $47,878 | $0 | $0 | n/a | n/a |
| financial-pressure | 41203 | greedy | +$425 | +$1,658 | +$1,233 | $55,388 | $0 | — | n/a | n/a |
| financial-pressure | 41203 | conservative-v2 | $0 | $0 | $0 | $55,388 | $0 | — | n/a | n/a |
| financial-pressure | 41203 | qwen36-35b-a3b | −$1,483 | +$684 | +$2,168 | $53,379 | −$2,009 | +$1,239 | n/a | n/a |
| financial-pressure | 41203 | gemma4-26b-a4b | $0 | $0 | $0 | $55,388 | $0 | $0 | n/a | n/a |
| reporting-backlog | 41201 | greedy | −$975 | +$916 | +$1,890 | $51,085 | $0 | — | n/a | n/a |
| reporting-backlog | 41201 | conservative-v2 | $0 | $0 | $0 | $51,085 | $0 | — | n/a | n/a |
| reporting-backlog | 41201 | qwen36-35b-a3b | +$1,070 | +$2,831 | +$1,761 | $48,356 | −$2,728 | +$206 | n/a | n/a |
| reporting-backlog | 41201 | gemma4-26b-a4b | $0 | $0 | $0 | $51,085 | $0 | $0 | n/a | n/a |
| reporting-backlog | 41202 | greedy | −$676 | +$1,214 | +$1,890 | $55,466 | $0 | — | n/a | n/a |
| reporting-backlog | 41202 | conservative-v2 | $0 | $0 | $0 | $55,466 | $0 | — | n/a | n/a |
| reporting-backlog | 41202 | qwen36-35b-a3b | −$16,102 | +$3,554 | +$19,656 | $55,713 | +$247 | $0 | n/a | n/a |
| reporting-backlog | 41202 | gemma4-26b-a4b | $0 | $0 | $0 | $55,466 | $0 | $0 | n/a | n/a |
| reporting-backlog | 41203 | greedy | +$237 | +$1,552 | +$1,315 | $62,957 | $0 | — | n/a | n/a |
| reporting-backlog | 41203 | conservative-v2 | $0 | $0 | $0 | $62,957 | $0 | — | n/a | n/a |
| reporting-backlog | 41203 | qwen36-35b-a3b | $0 | $0 | $0 | $62,957 | $0 | $0 | n/a | n/a |
| reporting-backlog | 41203 | gemma4-26b-a4b | $0 | $0 | $0 | $62,957 | $0 | $0 | n/a | n/a |
| demand-growth | 41201 | greedy | −$532 | +$1,358 | +$1,890 | $51,085 | $0 | — | n/a | n/a |
| demand-growth | 41201 | conservative-v2 | $0 | $0 | $0 | $51,085 | $0 | — | n/a | n/a |
| demand-growth | 41201 | qwen36-35b-a3b | −$3,471 | +$3,381 | +$6,852 | $51,016 | −$69 | +$21 | n/a | n/a |
| demand-growth | 41201 | gemma4-26b-a4b | −$3,074 | $0 | +$3,074 | $51,085 | $0 | $0 | n/a | n/a |
| demand-growth | 41202 | greedy | −$470 | +$1,420 | +$1,890 | $55,466 | $0 | — | n/a | n/a |
| demand-growth | 41202 | conservative-v2 | $0 | $0 | $0 | $55,466 | $0 | — | n/a | n/a |
| demand-growth | 41202 | qwen36-35b-a3b | −$1,435 | +$933 | +$2,368 | $54,802 | −$664 | $0 | n/a | n/a |
| demand-growth | 41202 | gemma4-26b-a4b | $0 | $0 | $0 | $55,466 | $0 | $0 | n/a | n/a |
| demand-growth | 41203 | greedy | +$389 | +$1,704 | +$1,315 | $62,957 | $0 | — | n/a | n/a |
| demand-growth | 41203 | conservative-v2 | $0 | $0 | $0 | $62,957 | $0 | — | n/a | n/a |
| demand-growth | 41203 | qwen36-35b-a3b | +$142 | +$3,299 | +$3,158 | $60,826 | −$2,131 | +$1,678 | n/a | n/a |
| demand-growth | 41203 | gemma4-26b-a4b | $0 | $0 | $0 | $62,957 | $0 | $0 | n/a | n/a |
| staffing-shortage | 41201 | greedy | −$1,144 | +$88 | +$1,233 | $49,222 | $0 | — | n/a | n/a |
| staffing-shortage | 41201 | conservative-v2 | $0 | $0 | $0 | $49,222 | $0 | — | n/a | n/a |
| staffing-shortage | 41201 | qwen36-35b-a3b | −$7,683 | +$5,604 | +$13,287 | $42,662 | −$6,560 | +$1,923 | n/a | n/a |
| staffing-shortage | 41201 | gemma4-26b-a4b | +$1,169 | +$2,290 | +$1,121 | $48,379 | −$843 | +$563 | n/a | n/a |
| staffing-shortage | 41202 | greedy | −$689 | +$544 | +$1,233 | $56,573 | $0 | — | n/a | n/a |
| staffing-shortage | 41202 | conservative-v2 | $0 | $0 | $0 | $56,573 | $0 | — | n/a | n/a |
| staffing-shortage | 41202 | qwen36-35b-a3b | −$30,447 | +$3,045 | +$33,492 | $49,439 | −$7,134 | +$586 | n/a | n/a |
| staffing-shortage | 41202 | gemma4-26b-a4b | −$1,391 | +$3,676 | +$5,067 | $55,975 | −$598 | +$352 | n/a | n/a |
| staffing-shortage | 41203 | greedy | −$925 | +$308 | +$1,233 | $63,203 | $0 | — | n/a | n/a |
| staffing-shortage | 41203 | conservative-v2 | $0 | $0 | $0 | $63,203 | $0 | — | n/a | n/a |
| staffing-shortage | 41203 | qwen36-35b-a3b | −$28,564 | +$8,797 | +$37,361 | $54,061 | −$9,142 | +$814 | n/a | n/a |
| staffing-shortage | 41203 | gemma4-26b-a4b | −$573 | $0 | +$573 | $63,203 | $0 | $0 | n/a | n/a |
| stable | 41201 | greedy | −$1,636 | +$255 | +$1,890 | $52,410 | $0 | — | n/a | n/a |
| stable | 41201 | conservative-v2 | −$567 | +$255 | +$822 | $52,410 | $0 | — | n/a | n/a |
| stable | 41201 | qwen36-35b-a3b | −$160 | +$208 | +$368 | $52,059 | −$351 | +$176 | n/a | n/a |
| stable | 41201 | gemma4-26b-a4b | −$627 | $0 | +$627 | $52,410 | $0 | $0 | n/a | n/a |
| stable | 41202 | greedy | +$4 | +$1,237 | +$1,233 | $53,633 | $0 | — | n/a | n/a |
| stable | 41202 | conservative-v2 | +$662 | +$1,237 | +$575 | $53,633 | $0 | — | n/a | n/a |
| stable | 41202 | qwen36-35b-a3b | −$635 | $0 | +$635 | $52,998 | −$635 | +$402 | n/a | n/a |
| stable | 41202 | gemma4-26b-a4b | −$660 | +$358 | +$1,018 | $53,633 | $0 | $0 | n/a | n/a |
| stable | 41203 | greedy | +$694 | +$1,926 | +$1,233 | $66,703 | $0 | — | n/a | n/a |
| stable | 41203 | conservative-v2 | $0 | $0 | $0 | $66,703 | $0 | — | n/a | n/a |
| stable | 41203 | qwen36-35b-a3b | −$2,923 | +$3,663 | +$6,587 | $64,377 | −$2,327 | $0 | n/a | n/a |
| stable | 41203 | gemma4-26b-a4b | $0 | $0 | $0 | $66,703 | $0 | $0 | n/a | n/a |

### 2. Operational outcomes (policy minus no action)

| Entry | Seed | Policy | Exams completed | Orders lost | Report TAT p90 (min) | TAT breaches | Overtime (h) |
|---|---|---|---|---|---|---|---|
| financial-pressure | 41201 | greedy | 0 | 0 | −25 | −1 | 0 |
| financial-pressure | 41201 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41201 | qwen36-35b-a3b | 0 | 0 | −18 | −7 | +24 |
| financial-pressure | 41201 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41202 | greedy | 0 | 0 | −30 | −4 | 0 |
| financial-pressure | 41202 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41202 | qwen36-35b-a3b | +1 | 0 | −3 | −2 | +6 |
| financial-pressure | 41202 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41203 | greedy | 0 | 0 | −34 | −5 | 0 |
| financial-pressure | 41203 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41203 | qwen36-35b-a3b | +7 | −4 | −19 | −7 | +54 |
| financial-pressure | 41203 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41201 | greedy | 0 | 0 | −51 | −14 | 0 |
| reporting-backlog | 41201 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41201 | qwen36-35b-a3b | +5 | −6 | −83 | −29 | +4 |
| reporting-backlog | 41201 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41202 | greedy | 0 | 0 | −187 | −27 | 0 |
| reporting-backlog | 41202 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41202 | qwen36-35b-a3b | 0 | 0 | −228 | −70 | +10 |
| reporting-backlog | 41202 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41203 | greedy | 0 | 0 | −5 | −3 | 0 |
| reporting-backlog | 41203 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41203 | qwen36-35b-a3b | 0 | 0 | −17 | −5 | +6 |
| reporting-backlog | 41203 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41201 | greedy | 0 | 0 | −61 | −11 | 0 |
| demand-growth | 41201 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41201 | qwen36-35b-a3b | +5 | 0 | −199 | −68 | +7 |
| demand-growth | 41201 | gemma4-26b-a4b | 0 | 0 | −159 | −41 | 0 |
| demand-growth | 41202 | greedy | 0 | 0 | −206 | −23 | 0 |
| demand-growth | 41202 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41202 | qwen36-35b-a3b | 0 | 0 | −262 | −80 | +8 |
| demand-growth | 41202 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41203 | greedy | 0 | 0 | −11 | −5 | 0 |
| demand-growth | 41203 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41203 | qwen36-35b-a3b | +29 | −19 | +108 | +25 | +47 |
| demand-growth | 41203 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41201 | greedy | 0 | 0 | −10 | 0 | 0 |
| staffing-shortage | 41201 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41201 | qwen36-35b-a3b | +13 | −16 | −51 | −25 | +68 |
| staffing-shortage | 41201 | gemma4-26b-a4b | +12 | −12 | +10 | +3 | +12 |
| staffing-shortage | 41202 | greedy | 0 | 0 | −28 | −3 | 0 |
| staffing-shortage | 41202 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41202 | qwen36-35b-a3b | +10 | −8 | −553 | −31 | +16 |
| staffing-shortage | 41202 | gemma4-26b-a4b | +24 | −16 | −576 | −32 | +4 |
| staffing-shortage | 41203 | greedy | 0 | 0 | −18 | −2 | 0 |
| staffing-shortage | 41203 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41203 | qwen36-35b-a3b | +51 | −23 | −36 | −7 | +19 |
| staffing-shortage | 41203 | gemma4-26b-a4b | 0 | 0 | −17 | −12 | 0 |
| stable | 41201 | greedy | 0 | 0 | 0 | −4 | 0 |
| stable | 41201 | conservative-v2 | 0 | 0 | 0 | −3 | 0 |
| stable | 41201 | qwen36-35b-a3b | +4 | −2 | −23 | −8 | +11 |
| stable | 41201 | gemma4-26b-a4b | 0 | 0 | −9 | −18 | 0 |
| stable | 41202 | greedy | 0 | 0 | −11 | −2 | 0 |
| stable | 41202 | conservative-v2 | 0 | 0 | −12 | −2 | 0 |
| stable | 41202 | qwen36-35b-a3b | 0 | 0 | −55 | −14 | +20 |
| stable | 41202 | gemma4-26b-a4b | 0 | 0 | −32 | −26 | 0 |
| stable | 41203 | greedy | 0 | 0 | −61 | −7 | 0 |
| stable | 41203 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| stable | 41203 | qwen36-35b-a3b | 0 | 0 | −655 | −38 | +12 |
| stable | 41203 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |

### 3–5. Stated reasons, submitted and executed decisions (LLM runs), and the contract's action counts (every policy)

Stated: explicit commitments by action type, as the frozen rules read them (a commitment can be hedged). Submitted: actions in submitted batches. Executed: actions the engine accepted with a change (replay apply results).

| Entry | Seed | Policy | Stated commitments | Submitted | Executed | Contract report: operations, staffing, refused |
|---|---|---|---|---|---|---|
| financial-pressure | 41201 | greedy | not an LLM | — | — | 0, 4, 0 |
| financial-pressure | 41201 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| financial-pressure | 41201 | qwen36-35b-a3b | extend_shift 7 | extend_shift 8 | extend_shift 8 | 8, 0, 0 |
| financial-pressure | 41201 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| financial-pressure | 41202 | greedy | not an LLM | — | — | 0, 4, 0 |
| financial-pressure | 41202 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| financial-pressure | 41202 | qwen36-35b-a3b | extend_shift 1 | extend_shift 4 | extend_shift 4 | 4, 0, 0 |
| financial-pressure | 41202 | gemma4-26b-a4b | contract_teleradiology 1 | contract_teleradiology 1 | contract_teleradiology 1 | 0, 1, 0 |
| financial-pressure | 41203 | greedy | not an LLM | — | — | 0, 4, 0 |
| financial-pressure | 41203 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| financial-pressure | 41203 | qwen36-35b-a3b | extend_shift 4 | extend_shift 19 | extend_shift 19 | 19, 0, 0 |
| financial-pressure | 41203 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| reporting-backlog | 41201 | greedy | not an LLM | — | — | 0, 4, 0 |
| reporting-backlog | 41201 | conservative-v2 | not an LLM | — | — | 0, 1, 0 |
| reporting-backlog | 41201 | qwen36-35b-a3b | contract_teleradiology 2, extend_shift 2 | contract_teleradiology 1, extend_shift 3, set_reading_allocation 1 | contract_teleradiology 1, extend_shift 3, set_reading_allocation 1 | 3, 2, 0 |
| reporting-backlog | 41201 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| reporting-backlog | 41202 | greedy | not an LLM | — | — | 0, 4, 0 |
| reporting-backlog | 41202 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| reporting-backlog | 41202 | qwen36-35b-a3b | extend_shift 2 | contract_teleradiology 1, extend_shift 3, set_reading_allocation 2 | contract_teleradiology 1, extend_shift 3, set_reading_allocation 2 | 3, 3, 0 |
| reporting-backlog | 41202 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| reporting-backlog | 41203 | greedy | not an LLM | — | — | 0, 4, 0 |
| reporting-backlog | 41203 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| reporting-backlog | 41203 | qwen36-35b-a3b | extend_shift 3 | extend_shift 3 | extend_shift 3 | 3, 0, 0 |
| reporting-backlog | 41203 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| demand-growth | 41201 | greedy | not an LLM | — | — | 0, 4, 0 |
| demand-growth | 41201 | conservative-v2 | not an LLM | — | — | 0, 1, 0 |
| demand-growth | 41201 | qwen36-35b-a3b | extend_shift 3, set_walkin_policy 1, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 5, set_walkin_policy 1, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 5, set_walkin_policy 1, set_worklist_policy 1 | 7, 1, 0 |
| demand-growth | 41201 | gemma4-26b-a4b | contract_teleradiology 1 | contract_teleradiology 1 | contract_teleradiology 1 | 0, 1, 0 |
| demand-growth | 41202 | greedy | not an LLM | — | — | 0, 4, 0 |
| demand-growth | 41202 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| demand-growth | 41202 | qwen36-35b-a3b | contract_teleradiology 5 | contract_teleradiology 4, extend_shift 4 | contract_teleradiology 3, extend_shift 4 | 4, 4, 0 |
| demand-growth | 41202 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| demand-growth | 41203 | greedy | not an LLM | — | — | 0, 4, 0 |
| demand-growth | 41203 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| demand-growth | 41203 | qwen36-35b-a3b | contract_teleradiology 1, extend_shift 24, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 30, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 30, set_worklist_policy 1 | 31, 1, 0 |
| demand-growth | 41203 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| staffing-shortage | 41201 | greedy | not an LLM | — | — | 0, 4, 0 |
| staffing-shortage | 41201 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| staffing-shortage | 41201 | qwen36-35b-a3b | extend_shift 5, hire 2, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 20, hire 1, set_roster 1 | contract_teleradiology 1, extend_shift 20, hire 1, set_roster 1 | 20, 3, 0 |
| staffing-shortage | 41201 | gemma4-26b-a4b | extend_shift 8 | extend_shift 13 | extend_shift 13 | 13, 0, 0 |
| staffing-shortage | 41202 | greedy | not an LLM | — | — | 0, 4, 0 |
| staffing-shortage | 41202 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| staffing-shortage | 41202 | qwen36-35b-a3b | extend_shift 5, hire 5 | contract_teleradiology 1, extend_shift 10, hire 2, set_reading_allocation 2 | contract_teleradiology 1, extend_shift 10, hire 2, set_reading_allocation 2 | 10, 5, 0 |
| staffing-shortage | 41202 | gemma4-26b-a4b | extend_shift 12, hire 1 | contract_teleradiology 1, extend_shift 11 | contract_teleradiology 1, extend_shift 11 | 11, 1, 0 |
| staffing-shortage | 41203 | greedy | not an LLM | — | — | 0, 4, 0 |
| staffing-shortage | 41203 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| staffing-shortage | 41203 | qwen36-35b-a3b | extend_shift 7, set_reading_allocation 1, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 13, hire 2, set_reading_allocation 4, set_roster 2, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 13, hire 2, set_reading_allocation 2, set_worklist_policy 1 | 14, 9, 0 |
| staffing-shortage | 41203 | gemma4-26b-a4b | contract_teleradiology 3 | contract_teleradiology 3 | contract_teleradiology 3 | 0, 3, 0 |
| stable | 41201 | greedy | not an LLM | — | — | 0, 4, 0 |
| stable | 41201 | conservative-v2 | not an LLM | — | — | 0, 1, 0 |
| stable | 41201 | qwen36-35b-a3b | extend_shift 2 | extend_shift 9 | extend_shift 9 | 9, 0, 0 |
| stable | 41201 | gemma4-26b-a4b | contract_teleradiology 1 | contract_teleradiology 1 | contract_teleradiology 1 | 0, 1, 0 |
| stable | 41202 | greedy | not an LLM | — | — | 0, 4, 0 |
| stable | 41202 | conservative-v2 | not an LLM | — | — | 0, 1, 0 |
| stable | 41202 | qwen36-35b-a3b | extend_shift 7, set_reading_allocation 1 | extend_shift 13 | extend_shift 13 | 13, 0, 0 |
| stable | 41202 | gemma4-26b-a4b | contract_teleradiology 1 | contract_teleradiology 1 | contract_teleradiology 1 | 0, 1, 0 |
| stable | 41203 | greedy | not an LLM | — | — | 0, 4, 0 |
| stable | 41203 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| stable | 41203 | qwen36-35b-a3b | contract_teleradiology 1, extend_shift 2, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 11, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 11, set_worklist_policy 1 | 12, 1, 0 |
| stable | 41203 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |

## v06c-compare-90: dreamteam-dev 1 (V0.6): overtime not costed, no credit line

Contract `docs/benchmark/v06c-dev/compare-90/contract.json`, engine 0.6.0-dev.3, 90 days.

### 1. Financial outcomes (policy minus no action)

| Entry | Seed | Policy | Profit | Revenue | Expense | Lowest cash (policy) | Lowest cash difference | Overtime cost | Most credit drawn | Interest |
|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure | 41301 | greedy | −$32,549 | +$5 | +$32,554 | $13,170 | −$26,709 | — | n/a | n/a |
| financial-pressure | 41301 | conservative-v2 | $0 | $0 | $0 | $39,878 | $0 | — | n/a | n/a |
| financial-pressure | 41301 | qwen36-35b-a3b | −$43,934 | −$63,378 | −$19,444 | $37,719 | −$2,159 | +$654 | n/a | n/a |
| financial-pressure | 41301 | gemma4-26b-a4b | $0 | $0 | $0 | $39,878 | $0 | $0 | n/a | n/a |
| reporting-backlog | 41301 | greedy | −$33,940 | −$27 | +$33,913 | $42,684 | −$2,371 | — | n/a | n/a |
| reporting-backlog | 41301 | conservative-v2 | $0 | $0 | $0 | $45,055 | $0 | — | n/a | n/a |
| reporting-backlog | 41301 | qwen36-35b-a3b | +$1,284 | +$2,607 | +$1,323 | $44,331 | −$725 | +$561 | n/a | n/a |
| reporting-backlog | 41301 | gemma4-26b-a4b | $0 | $0 | $0 | $45,055 | $0 | $0 | n/a | n/a |
| demand-growth | 41301 | greedy | −$27,266 | +$6,729 | +$33,995 | $43,920 | −$1,136 | — | n/a | n/a |
| demand-growth | 41301 | conservative-v2 | +$2,191 | +$2,766 | +$575 | $45,055 | $0 | — | n/a | n/a |
| demand-growth | 41301 | qwen36-35b-a3b | −$2,260 | +$2,203 | +$4,463 | $43,345 | −$1,710 | +$2,568 | n/a | n/a |
| demand-growth | 41301 | gemma4-26b-a4b | +$2,098 | +$7,962 | +$5,864 | $45,055 | $0 | $0 | n/a | n/a |
| staffing-shortage | 41301 | greedy | −$32,591 | −$25 | +$32,567 | $26,043 | −$22,643 | — | n/a | n/a |
| staffing-shortage | 41301 | conservative-v2 | $0 | $0 | $0 | $48,686 | $0 | — | n/a | n/a |
| staffing-shortage | 41301 | qwen36-35b-a3b | −$137,507 | +$22,769 | +$160,276 | −$85,854 | −$134,540 | +$1,374 | n/a | n/a |
| staffing-shortage | 41301 | gemma4-26b-a4b | $0 | $0 | $0 | $48,686 | $0 | $0 | n/a | n/a |
| stable | 41301 | greedy | −$32,655 | −$40 | +$32,615 | $47,058 | $0 | — | n/a | n/a |
| stable | 41301 | conservative-v2 | −$2,301 | +$37 | +$2,339 | $47,058 | $0 | — | n/a | n/a |
| stable | 41301 | qwen36-35b-a3b | −$58,693 | −$5,074 | +$53,619 | $39,861 | −$7,197 | +$882 | n/a | n/a |
| stable | 41301 | gemma4-26b-a4b | $0 | $0 | $0 | $47,058 | $0 | $0 | n/a | n/a |

### 2. Operational outcomes (policy minus no action)

| Entry | Seed | Policy | Exams completed | Orders lost | Report TAT p90 (min) | TAT breaches | Overtime (h) |
|---|---|---|---|---|---|---|---|
| financial-pressure | 41301 | greedy | 0 | 0 | −118 | −51 | 0 |
| financial-pressure | 41301 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41301 | qwen36-35b-a3b | −298 | +361 | −107 | −76 | +55 |
| financial-pressure | 41301 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41301 | greedy | 0 | 0 | −177 | −139 | 0 |
| reporting-backlog | 41301 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41301 | qwen36-35b-a3b | +15 | −14 | +16 | +11 | +28 |
| reporting-backlog | 41301 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41301 | greedy | 0 | 0 | −423 | −267 | 0 |
| demand-growth | 41301 | conservative-v2 | 0 | 0 | −47 | −5 | 0 |
| demand-growth | 41301 | qwen36-35b-a3b | +13 | −16 | +2 | −5 | +56 |
| demand-growth | 41301 | gemma4-26b-a4b | 0 | 0 | −155 | −97 | 0 |
| staffing-shortage | 41301 | greedy | 0 | 0 | −96 | −42 | 0 |
| staffing-shortage | 41301 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41301 | qwen36-35b-a3b | +187 | −113 | −630 | −67 | +54 |
| staffing-shortage | 41301 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| stable | 41301 | greedy | 0 | 0 | −251 | −99 | 0 |
| stable | 41301 | conservative-v2 | 0 | 0 | −117 | −53 | 0 |
| stable | 41301 | qwen36-35b-a3b | −21 | +21 | −832 | −206 | +17 |
| stable | 41301 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |

### 3–5. Stated reasons, submitted and executed decisions (LLM runs), and the contract's action counts (every policy)

Stated: explicit commitments by action type, as the frozen rules read them (a commitment can be hedged). Submitted: actions in submitted batches. Executed: actions the engine accepted with a change (replay apply results).

| Entry | Seed | Policy | Stated commitments | Submitted | Executed | Contract report: operations, staffing, refused |
|---|---|---|---|---|---|---|
| financial-pressure | 41301 | greedy | not an LLM | — | — | 0, 12, 0 |
| financial-pressure | 41301 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| financial-pressure | 41301 | qwen36-35b-a3b | contract_teleradiology 2, extend_shift 13, set_worklist_policy 4 | contract_teleradiology 4, extend_shift 41, set_roster 2, set_worklist_policy 6, terminate 3 | contract_teleradiology 3, extend_shift 41, set_roster 1, set_worklist_policy 2, terminate 3 | 47, 9, 0 |
| financial-pressure | 41301 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| reporting-backlog | 41301 | greedy | not an LLM | — | — | 0, 12, 0 |
| reporting-backlog | 41301 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| reporting-backlog | 41301 | qwen36-35b-a3b | extend_shift 6 | extend_shift 19 | extend_shift 19 | 19, 0, 0 |
| reporting-backlog | 41301 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| demand-growth | 41301 | greedy | not an LLM | — | — | 0, 12, 0 |
| demand-growth | 41301 | conservative-v2 | not an LLM | — | — | 0, 1, 0 |
| demand-growth | 41301 | qwen36-35b-a3b | extend_shift 12, set_worklist_policy 1 | extend_shift 35, set_worklist_policy 1 | extend_shift 35, set_worklist_policy 1 | 36, 0, 0 |
| demand-growth | 41301 | gemma4-26b-a4b | contract_teleradiology 3 | contract_teleradiology 1 | contract_teleradiology 1 | 0, 1, 0 |
| staffing-shortage | 41301 | greedy | not an LLM | — | — | 0, 12, 0 |
| staffing-shortage | 41301 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| staffing-shortage | 41301 | qwen36-35b-a3b | contract_teleradiology 2, extend_shift 12, set_reading_allocation 2 | contract_teleradiology 1, extend_shift 53, hire 3, set_reading_allocation 2, set_roster 2 | contract_teleradiology 1, extend_shift 53, hire 3, set_reading_allocation 2, set_roster 1 | 53, 8, 0 |
| staffing-shortage | 41301 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| stable | 41301 | greedy | not an LLM | — | — | 0, 12, 0 |
| stable | 41301 | conservative-v2 | not an LLM | — | — | 0, 2, 0 |
| stable | 41301 | qwen36-35b-a3b | contract_teleradiology 4, extend_shift 5, hire 1, set_reading_allocation 2 | contract_teleradiology 3, extend_shift 8, set_reading_allocation 3, set_roster 2 | contract_teleradiology 3, extend_shift 8, set_reading_allocation 2, set_roster 1 | 8, 8, 0 |
| stable | 41301 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |

## v07c-stage-a: dreamteam-dev 3 (V0.7): radiologist overtime paid, $50,000 credit line at 10%

Contract `docs/benchmark/v07c-dev/stage-a/contract.json`, engine 0.7.0-dev.3, 30 days.

### 1. Financial outcomes (policy minus no action)

| Entry | Seed | Policy | Profit | Revenue | Expense | Lowest cash (policy) | Lowest cash difference | Overtime cost | Most credit drawn | Interest |
|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41601 | greedy | −$59 | +$1,174 | +$1,233 | $41,873 | $0 | — | — | — |
| financial-pressure-credit | 41601 | conservative-v2 | $0 | $0 | $0 | $41,873 | $0 | — | — | — |
| financial-pressure-credit | 41601 | qwen36-35b-a3b | −$15,786 | +$367 | +$16,154 | $31,777 | −$10,096 | +$12,923 | $0 | $0 |
| financial-pressure-credit | 41601 | gemma4-26b-a4b | $0 | $0 | $0 | $41,873 | $0 | $0 | $0 | $0 |
| financial-pressure-credit | 41602 | greedy | −$185 | +$1,048 | +$1,233 | $36,846 | $0 | — | — | — |
| financial-pressure-credit | 41602 | conservative-v2 | $0 | $0 | $0 | $36,846 | $0 | — | — | — |
| financial-pressure-credit | 41602 | qwen36-35b-a3b | −$4,684 | +$1,656 | +$6,340 | $35,114 | −$1,732 | +$804 | $0 | $0 |
| financial-pressure-credit | 41602 | gemma4-26b-a4b | $0 | $0 | $0 | $36,846 | $0 | $0 | $0 | $0 |
| reporting-backlog-credit | 41601 | greedy | −$531 | +$784 | +$1,315 | $50,450 | $0 | — | — | — |
| reporting-backlog-credit | 41601 | conservative-v2 | $0 | $0 | $0 | $50,450 | $0 | — | — | — |
| reporting-backlog-credit | 41601 | qwen36-35b-a3b | −$3,593 | +$607 | +$4,200 | $49,284 | −$1,166 | +$1,393 | $0 | $0 |
| reporting-backlog-credit | 41601 | gemma4-26b-a4b | −$1,383 | $0 | +$1,383 | $50,450 | $0 | $0 | $0 | $0 |
| reporting-backlog-credit | 41602 | greedy | −$1,387 | +$504 | +$1,890 | $44,329 | $0 | — | — | — |
| reporting-backlog-credit | 41602 | conservative-v2 | $0 | $0 | $0 | $44,329 | $0 | — | — | — |
| reporting-backlog-credit | 41602 | qwen36-35b-a3b | −$5,381 | +$3,414 | +$8,794 | $42,425 | −$1,904 | +$1,394 | $0 | $0 |
| reporting-backlog-credit | 41602 | gemma4-26b-a4b | $0 | $0 | $0 | $44,329 | $0 | $0 | $0 | $0 |
| demand-growth-credit | 41601 | greedy | −$948 | +$367 | +$1,315 | $50,450 | $0 | — | — | — |
| demand-growth-credit | 41601 | conservative-v2 | $0 | $0 | $0 | $50,450 | $0 | — | — | — |
| demand-growth-credit | 41601 | qwen36-35b-a3b | −$8,435 | +$3,121 | +$11,556 | $47,666 | −$2,784 | +$1,855 | $0 | $0 |
| demand-growth-credit | 41601 | gemma4-26b-a4b | −$2,736 | −$318 | +$2,419 | $50,450 | $0 | $0 | $0 | $0 |
| demand-growth-credit | 41602 | greedy | −$991 | +$899 | +$1,890 | $44,329 | $0 | — | — | — |
| demand-growth-credit | 41602 | conservative-v2 | $0 | $0 | $0 | $44,329 | $0 | — | — | — |
| demand-growth-credit | 41602 | qwen36-35b-a3b | −$3,197 | +$3,222 | +$6,419 | $43,408 | −$920 | $0 | $0 | $0 |
| demand-growth-credit | 41602 | gemma4-26b-a4b | $0 | $0 | $0 | $44,329 | $0 | $0 | $0 | $0 |
| staffing-shortage-credit | 41601 | greedy | −$1,038 | +$195 | +$1,233 | $46,252 | $0 | — | — | — |
| staffing-shortage-credit | 41601 | conservative-v2 | $0 | $0 | $0 | $46,252 | $0 | — | — | — |
| staffing-shortage-credit | 41601 | qwen36-35b-a3b | −$20,304 | +$3,059 | +$23,363 | $38,435 | −$7,817 | +$2,502 | $0 | $0 |
| staffing-shortage-credit | 41601 | gemma4-26b-a4b | $0 | $0 | $0 | $46,252 | $0 | $0 | $0 | $0 |
| staffing-shortage-credit | 41602 | greedy | −$1,076 | +$157 | +$1,233 | $43,751 | $0 | — | — | — |
| staffing-shortage-credit | 41602 | conservative-v2 | $0 | $0 | $0 | $43,751 | $0 | — | — | — |
| staffing-shortage-credit | 41602 | qwen36-35b-a3b | −$13,440 | +$813 | +$14,253 | $39,128 | −$4,623 | +$2,102 | $0 | $0 |
| staffing-shortage-credit | 41602 | gemma4-26b-a4b | −$2,043 | +$1,881 | +$3,925 | $43,751 | $0 | $0 | $0 | $0 |
| stable-credit | 41601 | greedy | −$152 | +$1,163 | +$1,315 | $52,187 | $0 | — | — | — |
| stable-credit | 41601 | conservative-v2 | $0 | $0 | $0 | $52,187 | $0 | — | — | — |
| stable-credit | 41601 | qwen36-35b-a3b | −$9,401 | +$374 | +$9,775 | $47,555 | −$4,632 | +$6,459 | $0 | $0 |
| stable-credit | 41601 | gemma4-26b-a4b | $0 | $0 | $0 | $52,187 | $0 | $0 | $0 | $0 |
| stable-credit | 41602 | greedy | −$30 | +$1,202 | +$1,233 | $47,059 | $0 | — | — | — |
| stable-credit | 41602 | conservative-v2 | $0 | $0 | $0 | $47,059 | $0 | — | — | — |
| stable-credit | 41602 | qwen36-35b-a3b | −$4,637 | $0 | +$4,637 | $42,421 | −$4,637 | +$3,710 | $0 | $0 |
| stable-credit | 41602 | gemma4-26b-a4b | $0 | $0 | $0 | $47,059 | $0 | $0 | $0 | $0 |

### 2. Operational outcomes (policy minus no action)

| Entry | Seed | Policy | Exams completed | Orders lost | Report TAT p90 (min) | TAT breaches | Overtime (h) |
|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41601 | greedy | 0 | 0 | −13 | −1 | 0 |
| financial-pressure-credit | 41601 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure-credit | 41601 | qwen36-35b-a3b | 0 | 0 | −40 | −1 | +56 |
| financial-pressure-credit | 41601 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| financial-pressure-credit | 41602 | greedy | 0 | 0 | −11 | −1 | 0 |
| financial-pressure-credit | 41602 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure-credit | 41602 | qwen36-35b-a3b | +4 | −1 | −2 | −1 | +4 |
| financial-pressure-credit | 41602 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog-credit | 41601 | greedy | 0 | 0 | −47 | −15 | 0 |
| reporting-backlog-credit | 41601 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog-credit | 41601 | qwen36-35b-a3b | +1 | 0 | −77 | −61 | +7 |
| reporting-backlog-credit | 41601 | gemma4-26b-a4b | 0 | 0 | −43 | −23 | 0 |
| reporting-backlog-credit | 41602 | greedy | 0 | 0 | −54 | −22 | 0 |
| reporting-backlog-credit | 41602 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog-credit | 41602 | qwen36-35b-a3b | +1 | −3 | −29 | −26 | +9 |
| reporting-backlog-credit | 41602 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| demand-growth-credit | 41601 | greedy | 0 | 0 | −52 | −14 | 0 |
| demand-growth-credit | 41601 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| demand-growth-credit | 41601 | qwen36-35b-a3b | +6 | −5 | −82 | −43 | +13 |
| demand-growth-credit | 41601 | gemma4-26b-a4b | 0 | 0 | −67 | −39 | 0 |
| demand-growth-credit | 41602 | greedy | 0 | 0 | −58 | −25 | 0 |
| demand-growth-credit | 41602 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| demand-growth-credit | 41602 | qwen36-35b-a3b | 0 | 0 | −145 | −60 | 0 |
| demand-growth-credit | 41602 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage-credit | 41601 | greedy | 0 | 0 | −16 | −2 | 0 |
| staffing-shortage-credit | 41601 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage-credit | 41601 | qwen36-35b-a3b | +11 | −10 | −604 | −33 | +21 |
| staffing-shortage-credit | 41601 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage-credit | 41602 | greedy | 0 | 0 | −3 | −1 | 0 |
| staffing-shortage-credit | 41602 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage-credit | 41602 | qwen36-35b-a3b | +2 | −2 | −16 | −2 | +11 |
| staffing-shortage-credit | 41602 | gemma4-26b-a4b | 0 | 0 | −596 | −21 | 0 |
| stable-credit | 41601 | greedy | 0 | 0 | −31 | −6 | 0 |
| stable-credit | 41601 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| stable-credit | 41601 | qwen36-35b-a3b | 0 | 0 | −69 | −37 | +29 |
| stable-credit | 41601 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| stable-credit | 41602 | greedy | 0 | 0 | −38 | −4 | 0 |
| stable-credit | 41602 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| stable-credit | 41602 | qwen36-35b-a3b | 0 | 0 | −7 | +1 | +16 |
| stable-credit | 41602 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |

### 3–5. Stated reasons, submitted and executed decisions (LLM runs), and the contract's action counts (every policy)

Stated: explicit commitments by action type, as the frozen rules read them (a commitment can be hedged). Submitted: actions in submitted batches. Executed: actions the engine accepted with a change (replay apply results).

| Entry | Seed | Policy | Stated commitments | Submitted | Executed | Contract report: operations, staffing, refused |
|---|---|---|---|---|---|---|
| financial-pressure-credit | 41601 | greedy | not an LLM | — | — | 0, 4, 0 |
| financial-pressure-credit | 41601 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| financial-pressure-credit | 41601 | qwen36-35b-a3b | none | extend_shift 16 | extend_shift 16 | 16, 0, 0 |
| financial-pressure-credit | 41601 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| financial-pressure-credit | 41602 | greedy | not an LLM | — | — | 0, 4, 0 |
| financial-pressure-credit | 41602 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| financial-pressure-credit | 41602 | qwen36-35b-a3b | extend_shift 1, set_reading_allocation 1 | extend_shift 3, set_reading_allocation 1 | extend_shift 3, set_reading_allocation 1 | 3, 1, 0 |
| financial-pressure-credit | 41602 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| reporting-backlog-credit | 41601 | greedy | not an LLM | — | — | 0, 4, 0 |
| reporting-backlog-credit | 41601 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| reporting-backlog-credit | 41601 | qwen36-35b-a3b | extend_shift 7, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 5, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 5, set_worklist_policy 1 | 6, 1, 0 |
| reporting-backlog-credit | 41601 | gemma4-26b-a4b | contract_teleradiology 1 | contract_teleradiology 1 | contract_teleradiology 1 | 0, 1, 0 |
| reporting-backlog-credit | 41602 | greedy | not an LLM | — | — | 0, 4, 0 |
| reporting-backlog-credit | 41602 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| reporting-backlog-credit | 41602 | qwen36-35b-a3b | contract_teleradiology 3, extend_shift 7, set_reading_allocation 1 | contract_teleradiology 2, extend_shift 6, set_reading_allocation 1 | contract_teleradiology 2, extend_shift 6, set_reading_allocation 1 | 6, 3, 0 |
| reporting-backlog-credit | 41602 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| demand-growth-credit | 41601 | greedy | not an LLM | — | — | 0, 4, 0 |
| demand-growth-credit | 41601 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| demand-growth-credit | 41601 | qwen36-35b-a3b | extend_shift 7, set_worklist_policy 1 | extend_shift 9, set_reading_allocation 1, set_worklist_policy 1 | extend_shift 9, set_reading_allocation 1, set_worklist_policy 1 | 10, 1, 0 |
| demand-growth-credit | 41601 | gemma4-26b-a4b | contract_teleradiology 3 | contract_teleradiology 3 | contract_teleradiology 3 | 0, 3, 0 |
| demand-growth-credit | 41602 | greedy | not an LLM | — | — | 0, 4, 0 |
| demand-growth-credit | 41602 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| demand-growth-credit | 41602 | qwen36-35b-a3b | contract_teleradiology 1 | contract_teleradiology 1 | contract_teleradiology 1 | 0, 1, 0 |
| demand-growth-credit | 41602 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| staffing-shortage-credit | 41601 | greedy | not an LLM | — | — | 0, 4, 0 |
| staffing-shortage-credit | 41601 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| staffing-shortage-credit | 41601 | qwen36-35b-a3b | contract_teleradiology 1, extend_shift 9, hire 2, set_reading_allocation 3, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 11, hire 1, set_reading_allocation 2, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 11, hire 1, set_reading_allocation 1, set_worklist_policy 1 | 12, 4, 0 |
| staffing-shortage-credit | 41601 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| staffing-shortage-credit | 41602 | greedy | not an LLM | — | — | 0, 4, 0 |
| staffing-shortage-credit | 41602 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| staffing-shortage-credit | 41602 | qwen36-35b-a3b | contract_teleradiology 1, extend_shift 6, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 10, hire 1, set_reading_allocation 2, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 10, hire 1, set_reading_allocation 2, set_worklist_policy 1 | 11, 4, 0 |
| staffing-shortage-credit | 41602 | gemma4-26b-a4b | contract_teleradiology 1 | contract_teleradiology 1 | contract_teleradiology 1 | 0, 1, 0 |
| stable-credit | 41601 | greedy | not an LLM | — | — | 0, 4, 0 |
| stable-credit | 41601 | conservative-v2 | not an LLM | — | — | 0, 1, 0 |
| stable-credit | 41601 | qwen36-35b-a3b | extend_shift 3 | contract_teleradiology 1, extend_shift 17, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 17, set_worklist_policy 1 | 18, 1, 0 |
| stable-credit | 41601 | gemma4-26b-a4b | contract_teleradiology 1 | contract_teleradiology 1 | contract_teleradiology 1 | 0, 1, 0 |
| stable-credit | 41602 | greedy | not an LLM | — | — | 0, 4, 0 |
| stable-credit | 41602 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| stable-credit | 41602 | qwen36-35b-a3b | contract_teleradiology 1, extend_shift 2, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 8, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 8, set_worklist_policy 1 | 9, 1, 0 |
| stable-credit | 41602 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |

## v07c-stage-b: dreamteam-dev 3 (V0.7): radiologist overtime paid, $50,000 credit line at 10%

Contract `docs/benchmark/v07c-dev/stage-b/contract.json`, engine 0.7.0-dev.3, 90 days.

### 1. Financial outcomes (policy minus no action)

| Entry | Seed | Policy | Profit | Revenue | Expense | Lowest cash (policy) | Lowest cash difference | Overtime cost | Most credit drawn | Interest |
|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41701 | greedy | −$32,565 | +$11 | +$32,576 | $5,603 | −$18,863 | — | — | — |
| financial-pressure-credit | 41701 | conservative-v2 | $0 | $0 | $0 | $24,466 | $0 | — | — | — |
| financial-pressure-credit | 41701 | qwen36-35b-a3b | −$19,149 | +$1,115 | +$20,263 | $18,168 | −$6,298 | +$10,477 | $0 | $0 |
| financial-pressure-credit | 41701 | gemma4-26b-a4b | $0 | $0 | $0 | $24,466 | $0 | $0 | $0 | $0 |
| reporting-backlog-credit | 41701 | greedy | −$33,550 | +$407 | +$33,957 | $39,081 | −$6,082 | — | — | — |
| reporting-backlog-credit | 41701 | conservative-v2 | $0 | $0 | $0 | $45,163 | $0 | — | — | — |
| reporting-backlog-credit | 41701 | qwen36-35b-a3b | −$8,007 | +$945 | +$8,952 | $37,041 | −$8,122 | +$6,865 | $0 | $0 |
| reporting-backlog-credit | 41701 | gemma4-26b-a4b | −$22,096 | +$1,589 | +$23,685 | $40,165 | −$4,998 | $0 | $0 | $0 |
| demand-growth-credit | 41701 | greedy | −$29,022 | +$4,891 | +$33,913 | $39,026 | −$6,082 | — | — | — |
| demand-growth-credit | 41701 | conservative-v2 | $0 | $0 | $0 | $45,108 | $0 | — | — | — |
| demand-growth-credit | 41701 | qwen36-35b-a3b | −$20,914 | +$6,768 | +$27,682 | $38,959 | −$6,149 | +$2,357 | $0 | $0 |
| demand-growth-credit | 41701 | gemma4-26b-a4b | −$5,923 | +$10,139 | +$16,061 | $42,957 | −$2,152 | +$1,875 | $0 | $0 |
| staffing-shortage-credit | 41701 | greedy | −$32,622 | −$52 | +$32,570 | $24,153 | −$13,075 | — | — | — |
| staffing-shortage-credit | 41701 | conservative-v2 | $0 | $0 | $0 | $37,229 | $0 | — | — | — |
| staffing-shortage-credit | 41701 | qwen36-35b-a3b | −$181,589 | −$138,515 | +$43,074 | $0 | −$37,229 | −$2,218 | $17,200 | $14 |
| staffing-shortage-credit | 41701 | gemma4-26b-a4b | −$3,687 | +$12,668 | +$16,355 | $30,602 | −$6,627 | +$4,617 | $0 | $0 |
| stable-credit | 41701 | greedy | −$32,483 | +$110 | +$32,593 | $47,046 | $0 | — | — | — |
| stable-credit | 41701 | conservative-v2 | −$2,712 | $0 | +$2,712 | $47,046 | $0 | — | — | — |
| stable-credit | 41701 | qwen36-35b-a3b | −$51,560 | +$1,796 | +$53,357 | $37,224 | −$9,823 | +$1,575 | $0 | $0 |
| stable-credit | 41701 | gemma4-26b-a4b | $0 | $0 | $0 | $47,046 | $0 | $0 | $0 | $0 |

### 2. Operational outcomes (policy minus no action)

| Entry | Seed | Policy | Exams completed | Orders lost | Report TAT p90 (min) | TAT breaches | Overtime (h) |
|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41701 | greedy | 0 | 0 | −162 | −86 | 0 |
| financial-pressure-credit | 41701 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure-credit | 41701 | qwen36-35b-a3b | 0 | 0 | −31 | −32 | +45 |
| financial-pressure-credit | 41701 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog-credit | 41701 | greedy | 0 | 0 | −342 | −233 | 0 |
| reporting-backlog-credit | 41701 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog-credit | 41701 | qwen36-35b-a3b | +11 | −8 | −34 | −38 | +42 |
| reporting-backlog-credit | 41701 | gemma4-26b-a4b | 0 | 0 | −285 | −340 | 0 |
| demand-growth-credit | 41701 | greedy | 0 | 0 | −457 | −317 | 0 |
| demand-growth-credit | 41701 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| demand-growth-credit | 41701 | qwen36-35b-a3b | +6 | −7 | −332 | −304 | +17 |
| demand-growth-credit | 41701 | gemma4-26b-a4b | +35 | −28 | −180 | −108 | +43 |
| staffing-shortage-credit | 41701 | greedy | 0 | 0 | −89 | −34 | 0 |
| staffing-shortage-credit | 41701 | conservative-v2 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage-credit | 41701 | qwen36-35b-a3b | −867 | +1,450 | −635 | −79 | −41 |
| staffing-shortage-credit | 41701 | gemma4-26b-a4b | +54 | −53 | −16 | −106 | +97 |
| stable-credit | 41701 | greedy | 0 | 0 | −180 | −76 | 0 |
| stable-credit | 41701 | conservative-v2 | 0 | 0 | −27 | −12 | 0 |
| stable-credit | 41701 | qwen36-35b-a3b | +4 | 0 | −805 | −299 | +5 |
| stable-credit | 41701 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 |

### 3–5. Stated reasons, submitted and executed decisions (LLM runs), and the contract's action counts (every policy)

Stated: explicit commitments by action type, as the frozen rules read them (a commitment can be hedged). Submitted: actions in submitted batches. Executed: actions the engine accepted with a change (replay apply results).

| Entry | Seed | Policy | Stated commitments | Submitted | Executed | Contract report: operations, staffing, refused |
|---|---|---|---|---|---|---|
| financial-pressure-credit | 41701 | greedy | not an LLM | — | — | 0, 12, 0 |
| financial-pressure-credit | 41701 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| financial-pressure-credit | 41701 | qwen36-35b-a3b | contract_teleradiology 1, extend_shift 2, set_reading_allocation 2 | contract_teleradiology 1, extend_shift 22, set_reading_allocation 2 | contract_teleradiology 1, extend_shift 22, set_reading_allocation 1 | 22, 3, 0 |
| financial-pressure-credit | 41701 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
| reporting-backlog-credit | 41701 | greedy | not an LLM | — | — | 0, 12, 0 |
| reporting-backlog-credit | 41701 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| reporting-backlog-credit | 41701 | qwen36-35b-a3b | extend_shift 6 | extend_shift 14, set_walkin_policy 1, set_worklist_policy 1 | extend_shift 14, set_worklist_policy 1 | 16, 0, 0 |
| reporting-backlog-credit | 41701 | gemma4-26b-a4b | contract_teleradiology 4 | contract_teleradiology 3 | contract_teleradiology 3 | 0, 3, 0 |
| demand-growth-credit | 41701 | greedy | not an LLM | — | — | 0, 12, 0 |
| demand-growth-credit | 41701 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| demand-growth-credit | 41701 | qwen36-35b-a3b | extend_shift 12, set_worklist_policy 2 | contract_teleradiology 1, extend_shift 15, set_reading_allocation 1, set_worklist_policy 1 | contract_teleradiology 1, extend_shift 15, set_reading_allocation 1, set_worklist_policy 1 | 16, 2, 0 |
| demand-growth-credit | 41701 | gemma4-26b-a4b | contract_teleradiology 3, extend_shift 13 | contract_teleradiology 3, extend_shift 27 | contract_teleradiology 3, extend_shift 27 | 27, 3, 0 |
| staffing-shortage-credit | 41701 | greedy | not an LLM | — | — | 0, 12, 0 |
| staffing-shortage-credit | 41701 | conservative-v2 | not an LLM | — | — | 0, 0, 0 |
| staffing-shortage-credit | 41701 | qwen36-35b-a3b | extend_shift 9, set_reading_allocation 3, set_walkin_policy 1, set_worklist_policy 6, terminate 2 | extend_shift 14, hire 4, set_reading_allocation 2, set_worklist_policy 6, terminate 16 | extend_shift 14, hire 4, set_reading_allocation 2, set_worklist_policy 2, terminate 7 | 20, 22, 0 |
| staffing-shortage-credit | 41701 | gemma4-26b-a4b | contract_teleradiology 1, extend_shift 55, hire 1 | contract_teleradiology 1, extend_shift 53 | contract_teleradiology 1, extend_shift 53 | 53, 1, 0 |
| stable-credit | 41701 | greedy | not an LLM | — | — | 0, 12, 0 |
| stable-credit | 41701 | conservative-v2 | not an LLM | — | — | 0, 2, 0 |
| stable-credit | 41701 | qwen36-35b-a3b | contract_teleradiology 1, extend_shift 3, hire 1, set_walkin_policy 1 | contract_teleradiology 1, extend_shift 2, hire 1, set_walkin_policy 1 | contract_teleradiology 1, extend_shift 2, hire 1, set_walkin_policy 1 | 3, 2, 0 |
| stable-credit | 41701 | gemma4-26b-a4b | none | none | none | 0, 0, 0 |
