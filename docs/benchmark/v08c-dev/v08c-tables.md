# V0.8c evaluation tables

`dreamteam.v08c-analysis/v1`; suite {'id': 'dreamteam-dev', 'version': 4, 'split': 'development'}, engine 0.8.0-dev.3, 30 days, seeds 41901, 41902. Every figure describes runs; none is a score. Money to the nearest dollar. Differences are policy minus no action on the same entry and seed.

## Recording completeness (LLM runs)

| Entry | Seed | Policy | Points | Every attempt recorded | Inputs verifiable | Attempts | Attempts per point | Outputs, reasons, actions recorded | Transport failures | Infrastructure-affected points | Truncated fields |
|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41901 | qwen36-35b-a3b | 30 | 30 | 30 | 36 | 1: 24, 2: 6 | 36, 36, 36 | 0 | 0 | 0 |
| financial-pressure-credit | 41901 | gemma4-26b-a4b | 30 | 30 | 30 | 30 | 1: 30 | 30, 30, 30 | 0 | 0 | 0 |
| financial-pressure-credit | 41902 | qwen36-35b-a3b | 30 | 30 | 30 | 36 | 1: 24, 2: 6 | 36, 34, 34 | 0 | 0 | 0 |
| financial-pressure-credit | 41902 | gemma4-26b-a4b | 30 | 30 | 30 | 30 | 1: 30 | 30, 30, 30 | 0 | 0 | 0 |
| financial-pressure-lowliquidity | 41901 | qwen36-35b-a3b | 30 | 30 | 30 | 42 | 1: 18, 2: 12 | 42, 42, 42 | 0 | 0 | 0 |
| financial-pressure-lowliquidity | 41901 | gemma4-26b-a4b | 30 | 30 | 30 | 33 | 1: 27, 2: 3 | 33, 33, 33 | 0 | 0 | 0 |
| financial-pressure-lowliquidity | 41902 | qwen36-35b-a3b | 30 | 30 | 30 | 33 | 1: 27, 2: 3 | 33, 33, 33 | 0 | 0 | 0 |
| financial-pressure-lowliquidity | 41902 | gemma4-26b-a4b | 30 | 30 | 30 | 30 | 1: 30 | 30, 30, 30 | 0 | 0 | 0 |

## Funding refusals and what followed (LLM runs)

Refusals by validate with code `insufficient_funds`, in any attempt, and their responses (dreamteam.refusal-response/v2). Proposed: actions in every parsed attempt.

| Entry | Seed | Policy | Proposed actions | Funding refusals | Repeated | Revised: modified | Revised: different | No action | No longer applicable | Evidence unavailable | All validate refusals |
|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41901 | qwen36-35b-a3b | 28 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | after_close 7, batch_rejected 3, commitment_too_short 2, lead_too_short 1, no_coverage 1 |
| financial-pressure-credit | 41901 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | none |
| financial-pressure-credit | 41902 | qwen36-35b-a3b | 26 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | after_close 6, no_coverage 1, not_on_shift 1 |
| financial-pressure-credit | 41902 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | none |
| financial-pressure-lowliquidity | 41901 | qwen36-35b-a3b | 44 | 8 | 0 | 1 | 3 | 4 | 0 | 0 | after_close 3, batch_rejected 11, commitment_too_short 1, insufficient_funds 8, invalid_schema 1, lead_too_short 4, not_on_shift 1, past_date 1 |
| financial-pressure-lowliquidity | 41901 | gemma4-26b-a4b | 5 | 2 | 0 | 1 | 0 | 1 | 0 | 0 | insufficient_funds 2, invalid_schema 2 |
| financial-pressure-lowliquidity | 41902 | qwen36-35b-a3b | 22 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | after_close 3, already_terminating 1, batch_rejected 2, exceeds_max_shift 2, invalid_minutes 1, no_coverage 1, notice_too_short 1 |
| financial-pressure-lowliquidity | 41902 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | none |

## Refusals of every code and what followed (LLM runs; dreamteam.refusal-response/v2)

| Entry | Seed | Policy | Refusals | Repeated | Modified | Different | No action | No longer applicable | Evidence unavailable |
|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41901 | qwen36-35b-a3b | 11 | 0 | 1 | 3 | 7 | 0 | 0 |
| financial-pressure-credit | 41901 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure-credit | 41902 | qwen36-35b-a3b | 8 | 0 | 0 | 5 | 3 | 0 | 0 |
| financial-pressure-credit | 41902 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure-lowliquidity | 41901 | qwen36-35b-a3b | 19 | 0 | 6 | 4 | 9 | 0 | 0 |
| financial-pressure-lowliquidity | 41901 | gemma4-26b-a4b | 4 | 0 | 2 | 1 | 1 | 0 | 0 |
| financial-pressure-lowliquidity | 41902 | qwen36-35b-a3b | 9 | 1 | 4 | 3 | 1 | 0 | 0 |
| financial-pressure-lowliquidity | 41902 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | 0 | 0 | 0 |

## Actions submitted, accepted and executed (every run; replay apply results)

| Entry | Seed | Policy | Submitted | Accepted at apply | Executed (with a change) | Refused at apply | Executed by type |
|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41901 | noop | 0 | 0 | 0 | 0 | none |
| financial-pressure-credit | 41901 | greedy | 4 | 4 | 4 | 0 | set_reading_allocation 4 |
| financial-pressure-credit | 41901 | conservative-v2 | 0 | 0 | 0 | 0 | none |
| financial-pressure-credit | 41901 | qwen36-35b-a3b | 12 | 12 | 9 | 0 | extend_shift 2, set_reading_allocation 6, terminate 1 |
| financial-pressure-credit | 41901 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | none |
| financial-pressure-credit | 41902 | noop | 0 | 0 | 0 | 0 | none |
| financial-pressure-credit | 41902 | greedy | 4 | 4 | 4 | 0 | set_reading_allocation 4 |
| financial-pressure-credit | 41902 | conservative-v2 | 0 | 0 | 0 | 0 | none |
| financial-pressure-credit | 41902 | qwen36-35b-a3b | 9 | 9 | 9 | 0 | extend_shift 9 |
| financial-pressure-credit | 41902 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | none |
| financial-pressure-lowliquidity | 41901 | noop | 0 | 0 | 0 | 0 | none |
| financial-pressure-lowliquidity | 41901 | greedy | 4 | 2 | 2 | insufficient_funds 2 | set_reading_allocation 2 |
| financial-pressure-lowliquidity | 41901 | conservative-v2 | 0 | 0 | 0 | 0 | none |
| financial-pressure-lowliquidity | 41901 | qwen36-35b-a3b | 11 | 11 | 9 | 0 | extend_shift 5, hire 1, set_reading_allocation 2, terminate 1 |
| financial-pressure-lowliquidity | 41901 | gemma4-26b-a4b | 1 | 1 | 1 | 0 | contract_teleradiology 1 |
| financial-pressure-lowliquidity | 41902 | noop | 0 | 0 | 0 | 0 | none |
| financial-pressure-lowliquidity | 41902 | greedy | 4 | 2 | 2 | insufficient_funds 2 | set_reading_allocation 2 |
| financial-pressure-lowliquidity | 41902 | conservative-v2 | 0 | 0 | 0 | 0 | none |
| financial-pressure-lowliquidity | 41902 | qwen36-35b-a3b | 6 | 6 | 5 | 0 | extend_shift 2, set_worklist_policy 1, terminate 2 |
| financial-pressure-lowliquidity | 41902 | gemma4-26b-a4b | 0 | 0 | 0 | 0 | none |

## Financial outcomes (every run)

| Entry | Seed | Policy | Profit | Profit vs no action | Lowest cash | Most credit drawn | Days drawn | Interest | Shortfall (distress) | Days below zero |
|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41901 | noop | −$20,009 | baseline | $40,571 | $0 | 0 | $0 | $0 | 0 |
| financial-pressure-credit | 41901 | greedy | −$21,280 | −$1,271 | $40,571 | $0 | 0 | $0 | $0 | 0 |
| financial-pressure-credit | 41901 | conservative-v2 | −$20,009 | $0 | $40,571 | $0 | 0 | $0 | $0 | 0 |
| financial-pressure-credit | 41901 | qwen36-35b-a3b | −$41,230 | −$21,221 | $39,957 | $0 | 0 | $0 | $0 | 0 |
| financial-pressure-credit | 41901 | gemma4-26b-a4b | −$20,009 | $0 | $40,571 | $0 | 0 | $0 | $0 | 0 |
| financial-pressure-credit | 41902 | noop | −$13,911 | baseline | $37,780 | $0 | 0 | $0 | $0 | 0 |
| financial-pressure-credit | 41902 | greedy | −$14,509 | −$598 | $37,780 | $0 | 0 | $0 | $0 | 0 |
| financial-pressure-credit | 41902 | conservative-v2 | −$13,911 | $0 | $37,780 | $0 | 0 | $0 | $0 | 0 |
| financial-pressure-credit | 41902 | qwen36-35b-a3b | −$15,610 | −$1,699 | $36,009 | $0 | 0 | $0 | $0 | 0 |
| financial-pressure-credit | 41902 | gemma4-26b-a4b | −$13,911 | $0 | $37,780 | $0 | 0 | $0 | $0 | 0 |
| financial-pressure-lowliquidity | 41901 | noop | −$20,027 | baseline | $0 | $29,429 | 4 | $18 | $0 | 0 |
| financial-pressure-lowliquidity | 41901 | greedy | −$21,298 | −$1,271 | $0 | $29,429 | 4 | $18 | $0 | 0 |
| financial-pressure-lowliquidity | 41901 | conservative-v2 | −$20,027 | $0 | $0 | $29,429 | 4 | $18 | $0 | 0 |
| financial-pressure-lowliquidity | 41901 | qwen36-35b-a3b | −$40,994 | −$20,967 | $0 | $33,668 | 4 | $22 | $0 | 0 |
| financial-pressure-lowliquidity | 41901 | gemma4-26b-a4b | −$20,027 | $0 | $0 | $29,429 | 4 | $18 | $0 | 0 |
| financial-pressure-lowliquidity | 41902 | noop | −$13,934 | baseline | $0 | $32,220 | 4 | $23 | $0 | 0 |
| financial-pressure-lowliquidity | 41902 | greedy | −$14,532 | −$598 | $0 | $32,220 | 4 | $23 | $0 | 0 |
| financial-pressure-lowliquidity | 41902 | conservative-v2 | −$13,934 | $0 | $0 | $32,220 | 4 | $23 | $0 | 0 |
| financial-pressure-lowliquidity | 41902 | qwen36-35b-a3b | −$22,450 | −$8,515 | $0 | $33,951 | 5 | $26 | $0 | 0 |
| financial-pressure-lowliquidity | 41902 | gemma4-26b-a4b | −$13,934 | $0 | $0 | $32,220 | 4 | $23 | $0 | 0 |

## Operational outcomes (every run)

| Entry | Seed | Policy | Exams | vs no action | Lost orders | vs no action | TAT breaches | vs no action | TAT p90 (min) | vs no action |
|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41901 | noop | 1,496 | — | 161 | — | 127 | — | 833 | — |
| financial-pressure-credit | 41901 | greedy | 1,496 | 0 | 161 | 0 | 121 | −6 | 800 | −32 |
| financial-pressure-credit | 41901 | conservative-v2 | 1,496 | 0 | 161 | 0 | 127 | 0 | 833 | 0 |
| financial-pressure-credit | 41901 | qwen36-35b-a3b | 1,454 | −42 | 201 | +40 | 313 | +186 | 2,889 | +2,056 |
| financial-pressure-credit | 41901 | gemma4-26b-a4b | 1,496 | 0 | 161 | 0 | 127 | 0 | 833 | 0 |
| financial-pressure-credit | 41902 | noop | 1,516 | — | 152 | — | 107 | — | 862 | — |
| financial-pressure-credit | 41902 | greedy | 1,516 | 0 | 152 | 0 | 105 | −2 | 842 | −19 |
| financial-pressure-credit | 41902 | conservative-v2 | 1,516 | 0 | 152 | 0 | 107 | 0 | 862 | 0 |
| financial-pressure-credit | 41902 | qwen36-35b-a3b | 1,518 | +2 | 152 | 0 | 109 | +2 | 862 | 0 |
| financial-pressure-credit | 41902 | gemma4-26b-a4b | 1,516 | 0 | 152 | 0 | 107 | 0 | 862 | 0 |
| financial-pressure-lowliquidity | 41901 | noop | 1,496 | — | 161 | — | 127 | — | 833 | — |
| financial-pressure-lowliquidity | 41901 | greedy | 1,496 | 0 | 161 | 0 | 121 | −6 | 800 | −32 |
| financial-pressure-lowliquidity | 41901 | conservative-v2 | 1,496 | 0 | 161 | 0 | 127 | 0 | 833 | 0 |
| financial-pressure-lowliquidity | 41901 | qwen36-35b-a3b | 1,496 | 0 | 161 | 0 | 335 | +208 | 2,506 | +1,672 |
| financial-pressure-lowliquidity | 41901 | gemma4-26b-a4b | 1,496 | 0 | 161 | 0 | 127 | 0 | 833 | 0 |
| financial-pressure-lowliquidity | 41902 | noop | 1,516 | — | 152 | — | 107 | — | 862 | — |
| financial-pressure-lowliquidity | 41902 | greedy | 1,516 | 0 | 152 | 0 | 105 | −2 | 842 | −19 |
| financial-pressure-lowliquidity | 41902 | conservative-v2 | 1,516 | 0 | 152 | 0 | 107 | 0 | 862 | 0 |
| financial-pressure-lowliquidity | 41902 | qwen36-35b-a3b | 1,516 | 0 | 152 | 0 | 111 | +4 | 849 | −12 |
| financial-pressure-lowliquidity | 41902 | gemma4-26b-a4b | 1,516 | 0 | 152 | 0 | 107 | 0 | 862 | 0 |

## Repeated refused proposals: version 2 beside the frozen v1

| Group | Model | Refusals | repeated_refused | repeated_accepted | repeated_unvalidated | not_repeated | unresolved | insufficient_evidence | v2 findings |
|---|---|---|---|---|---|---|---|---|---|
| v08c | Qwen3.6-35B-A3B-oQ4e-mtp | 47 | 1 | 0 | 0 | 46 | 0 | 0 | 1 |
| v08c | gemma-4-26B-A4B-it-oQ4e-mtp | 4 | 0 | 0 | 0 | 4 | 0 | 0 | 0 |

| Group | Model | v1 links | v2 links | both | v1 only: v2 abstains | v1 only: different actions | v2 only: message differs | v2 only: other |
|---|---|---|---|---|---|---|---|---|
| v08c | Qwen3.6-35B-A3B-oQ4e-mtp | 13 | 1 | 1 | 0 | 12 | 0 | 0 |
| v08c | gemma-4-26B-A4B-it-oQ4e-mtp | 0 | 0 | 0 | 0 | 0 | 0 | 0 |

## The frozen rules (dreamteam.decision-evidence/v1) on the llm-agent 2 view

Findings by status: insufficient_evidence 0, supported 17, unresolved 35.

| Rule | Supported | Unresolved | Insufficient evidence |
|---|---|---|---|
| EXPLICIT_ACTION_OMITTED | 0 | 4 | 0 |
| EXPLICIT_OPPOSITE_ACTION | 0 | 0 | 0 |
| FACTUAL_OBSERVATION_MISMATCH | 2 | 0 | 0 |
| FINANCIAL_AMOUNT_MISMATCH | 0 | 31 | 0 |
| REPEATED_INVALID_PROPOSAL | 7 | 0 | 0 |
| STATED_ACTION_NOT_EXECUTED | 8 | 0 | 0 |

The independent review of the v1 findings is PENDING: none of these findings has been independently validated.
