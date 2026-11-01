# Financial decision evidence: tables

`dreamteam.financial-evidence/v1`, analyzer `financial-evidence 1`. Three dimensions, each counted on its own; nothing here is a score, a ranking or a cause. Money is exact, in dollars and cents, as the engine states it. Each group has its own economics and is never combined with another.

## 1. Coverage

| Group | Economics | Recordings | Replayed through the engine | Submissions | Executed actions | Earlier attempts not recorded |
|---|---|---|---|---|---|---|
| v09c | dreamteam-dev 4 (V0.9c, unchanged since V0.8c): radiologist overtime paid, $50,000 credit line at 10%, the V0.7b funding rule | 16 | 16 | 159 | 133 | 0 |

Earlier attempts not recorded: llm-agent 2 (V0.6c, V0.7c) records only its final reply; the earlier attempts' actions are unknown, their refusal codes are listed in §8.

## 2. Affordability (one record per submitted batch)

| Group | funded_without_credit | funded_with_credit | refused_insufficient_funds | no_new_commitment | not_tested | evidence_unavailable | Total |
|---|---|---|---|---|---|---|---|
| v09c | 79 | 3 | 0 | 7 | 68 | 2 | 159 |

Actions by type: submitted, accepted by validation, refused, and executed with a change.

| Group | Type | Submitted | Accepted | Refused | Executed |
|---|---|---|---|---|---|
| v09c | contract_teleradiology | 39 | 14 | 25 | 12 |
| v09c | extend_shift | 220 | 152 | 68 | 102 |
| v09c | hire | 11 | 6 | 5 | 4 |
| v09c | set_reading_allocation | 3 | 3 | 0 | 3 |
| v09c | set_roster | 17 | 10 | 7 | 8 |
| v09c | set_worklist_policy | 7 | 7 | 0 | 2 |
| v09c | terminate | 3 | 2 | 1 | 2 |

Commitments the engine measured for executed batches (the funding test's commitment for a staffing batch; the reservation an applied extension batch added), by funding status:

| Group | Kind | Status | Batches | Total commitment | Largest |
|---|---|---|---|---|---|
| v09c | extension | funded_with_credit | 2 | $1,916.24 | $958.12 |
| v09c | extension | funded_without_credit | 67 | $32,840.09 | $1,009.61 |
| v09c | staffing | funded_with_credit | 1 | $44,383.55 | $44,383.55 |
| v09c | staffing | funded_without_credit | 12 | $92,780.59 | $24,850.00 |

Refused for insufficient funds (the engine's commitment and headroom):

| Recording | Point | Actions | Commitment | Headroom | Headroom without credit |
|---|---|---|---|---|---|

Recurring obligations and immediate ledger effects the engine states for executed staffing batches:

| Group | Line | Lines | Total |
|---|---|---|---|
| v09c | immediate: Recruiting fee (recognized now, paid) | 6 | $18,000.00 |
| v09c | immediate: Severance (recognized now, paid) | 2 | $9,000.00 |
| v09c | recurring: Contract guarantee per full week (from) | 6 | $13,650.00 |
| v09c | recurring: Salary allocation change (per 30 days) | 3 | $34,520.53 |
| v09c | recurring: Teleradiology retainer, open-ended (per month) | 9 | $18,400.00 |

## 3. Credit usage

Observed at the decision points (B), and the most drawn at any day's end in the run with interest charged (from the replay's calendar, C). V0.6c has no credit line.

| Recording | Most drawn at a decision point | Most drawn at a day's end | Interest |
|---|---|---|---|

Recordings not listed drew no credit.

## 4. Reversibility (one record per executed action)

| Group | Action | withdrawable | reversible_with_retained_obligation | not_reversible | reversible_no_recorded_cost | unknown | Effective within the run | After the run ends |
|---|---|---|---|---|---|---|---|---|
| v09c | contract_teleradiology:cancel | 0 | 0 | 3 | 0 | 0 | 0 | 3 |
| v09c | contract_teleradiology:start | 0 | 9 | 0 | 0 | 0 | 7 | 2 |
| v09c | extend_shift | 0 | 0 | 102 | 0 | 0 | 99 | 3 |
| v09c | hire | 0 | 4 | 0 | 0 | 0 | 4 | 0 |
| v09c | set_reading_allocation | 3 | 0 | 0 | 0 | 0 | 3 | 0 |
| v09c | set_roster | 8 | 0 | 0 | 0 | 0 | 0 | 8 |
| v09c | set_worklist_policy | 0 | 0 | 0 | 2 | 0 | 0 | 2 |
| v09c | terminate | 0 | 0 | 2 | 0 | 0 | 2 | 0 |

The rule behind each class is in the reversibility matrix (docs/benchmark/v09b-dev/REVERSIBILITY.md); the amounts are per record in financial-evidence.json.

## 5. Operational exposure: direct state changes (A)

| Group | Resource | Actions (batches, for booked appointments) | Detail |
|---|---|---|---|
| v09c | booked appointments (termination batches) | 1 | 69 booked appointments displaced (counted once per batch) |
| v09c | reporting allocation | 3 | +3,630 site reading minutes per two rota weeks |
| v09c | reporting order | 2 | — |
| v09c | rostered time | 8 | — |
| v09c | scheduled duty | 102 | 12,690 extension minutes |
| v09c | technologist capacity | 6 | — |
| v09c | teleradiology coverage | 12 | — |

## 6. Subsequent observed state (B): first and last decision point of each run

Observed, never attributed to an action.

| Recording | Cash | Funding headroom | Credit drawn | Orders lost since start | Unread studies | Headcount (employed: active, notice, onboarding) |
|---|---|---|---|---|---|---|
| financial-pressure-credit-gemma4-26b-a4b-preview-s42001-decisions.jsonl | $150,000.00 → $131,362.33 | $106,024.83 → $87,273.20 | $0.00 → $0.00 | 174 | 1 → 10 | 9 → 9 |
| financial-pressure-credit-gemma4-26b-a4b-preview-s42002-decisions.jsonl | $150,000.00 → $119,048.89 | $106,874.05 → $75,284.98 | $0.00 → $0.00 | 180 | 4 → 9 | 9 → 9 |
| financial-pressure-credit-gemma4-26b-a4b-s42001-decisions.jsonl | $150,000.00 → $131,362.33 | $106,024.83 → $87,273.20 | $0.00 → $0.00 | 174 | 1 → 10 | 9 → 9 |
| financial-pressure-credit-gemma4-26b-a4b-s42002-decisions.jsonl | $150,000.00 → $119,048.89 | $106,874.05 → $75,284.98 | $0.00 → $0.00 | 180 | 4 → 9 | 9 → 9 |
| financial-pressure-credit-qwen36-35b-a3b-preview-s42001-decisions.jsonl | $150,000.00 → $127,114.01 | $106,024.83 → $80,744.49 | $0.00 → $0.00 | 173 | 1 → 0 | 9 → 9 |
| financial-pressure-credit-qwen36-35b-a3b-preview-s42002-decisions.jsonl | $150,000.00 → $113,268.23 | $106,874.05 → $66,619.77 | $0.00 → $0.00 | 180 | 4 → 9 | 9 → 9 |
| financial-pressure-credit-qwen36-35b-a3b-s42001-decisions.jsonl | $150,000.00 → $124,788.68 | $106,024.83 → $91,720.89 | $0.00 → $0.00 | 374 | 1 → 6 | 9 → 7 |
| financial-pressure-credit-qwen36-35b-a3b-s42002-decisions.jsonl | $150,000.00 → $118,183.53 | $106,874.05 → $73,265.80 | $0.00 → $0.00 | 180 | 4 → 9 | 9 → 9 |
| staffing-shortage-credit-gemma4-26b-a4b-preview-s42001-decisions.jsonl | $150,000.00 → $132,482.61 | $108,751.29 → $88,075.73 | $0.00 → $0.00 | 373 | 0 → 0 | 8 → 8 |
| staffing-shortage-credit-gemma4-26b-a4b-preview-s42002-decisions.jsonl | $150,000.00 → $126,039.51 | $109,042.78 → $84,987.10 | $0.00 → $0.00 | 407 | 0 → 9 | 8 → 8 |
| staffing-shortage-credit-gemma4-26b-a4b-s42001-decisions.jsonl | $150,000.00 → $132,482.61 | $108,751.29 → $88,075.73 | $0.00 → $0.00 | 373 | 0 → 0 | 8 → 8 |
| staffing-shortage-credit-gemma4-26b-a4b-s42002-decisions.jsonl | $150,000.00 → $126,039.51 | $109,042.78 → $84,987.10 | $0.00 → $0.00 | 407 | 0 → 9 | 8 → 8 |
| staffing-shortage-credit-qwen36-35b-a3b-preview-s42001-decisions.jsonl | $150,000.00 → $122,324.09 | $108,751.29 → $63,446.77 | $0.00 → $0.00 | 369 | 0 → 0 | 8 → 10 |
| staffing-shortage-credit-qwen36-35b-a3b-preview-s42002-decisions.jsonl | $150,000.00 → $123,478.96 | $109,042.78 → $35,416.35 | $0.00 → $0.00 | 388 | 0 → 0 | 8 → 8 |
| staffing-shortage-credit-qwen36-35b-a3b-s42001-decisions.jsonl | $150,000.00 → $122,080.85 | $108,751.29 → $64,864.55 | $0.00 → $0.00 | 407 | 0 → 14 | 8 → 10 |
| staffing-shortage-credit-qwen36-35b-a3b-s42002-decisions.jsonl | $150,000.00 → $123,341.90 | $109,042.78 → $76,163.71 | $0.00 → $0.00 | 392 | 0 → 0 | 8 → 8 |

## 7. Paired with no action (C): the run's own contract report

Policy minus no action on the same entry, seed, horizon and economics. A difference describes two runs; it is not allocated to any action.

| Recording | Days | Profit | Lowest cash | Orders lost | Exams completed | Overtime cost |
|---|---|---|---|---|---|---|
| financial-pressure-credit-gemma4-26b-a4b-preview-s42001-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-credit-gemma4-26b-a4b-preview-s42002-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-credit-gemma4-26b-a4b-s42001-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-credit-gemma4-26b-a4b-s42002-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| financial-pressure-credit-qwen36-35b-a3b-preview-s42001-decisions.jsonl | 30 | −$1,896.09 | −$4,248.32 | −1 | +1 | +$3,305.86 |
| financial-pressure-credit-qwen36-35b-a3b-preview-s42002-decisions.jsonl | 30 | −$8,665.21 | −$5,780.66 | 0 | 0 | +$6,932.19 |
| financial-pressure-credit-qwen36-35b-a3b-s42001-decisions.jsonl | 30 | −$27,548.48 | −$6,573.65 | +211 | −118 | +$771.36 |
| financial-pressure-credit-qwen36-35b-a3b-s42002-decisions.jsonl | 30 | −$2,019.18 | −$865.36 | 0 | 0 | +$1,615.35 |
| staffing-shortage-credit-gemma4-26b-a4b-preview-s42001-decisions.jsonl | 30 | +$1,233.00 | $0.00 | 0 | 0 | $0.00 |
| staffing-shortage-credit-gemma4-26b-a4b-preview-s42002-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| staffing-shortage-credit-gemma4-26b-a4b-s42001-decisions.jsonl | 30 | +$1,233.00 | $0.00 | 0 | 0 | $0.00 |
| staffing-shortage-credit-gemma4-26b-a4b-s42002-decisions.jsonl | 30 | $0.00 | $0.00 | 0 | 0 | $0.00 |
| staffing-shortage-credit-qwen36-35b-a3b-preview-s42001-decisions.jsonl | 30 | −$16,735.73 | −$10,203.52 | −7 | +13 | +$1,475.85 |
| staffing-shortage-credit-qwen36-35b-a3b-preview-s42002-decisions.jsonl | 30 | −$17,301.57 | −$2,820.55 | −19 | +17 | +$583.33 |
| staffing-shortage-credit-qwen36-35b-a3b-s42001-decisions.jsonl | 30 | −$11,507.00 | −$10,741.76 | +44 | +64 | +$3,783.50 |
| staffing-shortage-credit-qwen36-35b-a3b-s42002-decisions.jsonl | 30 | −$2,872.38 | −$2,787.61 | −16 | +14 | +$2,380.10 |

## 8. Evidence gaps

| Group | Gap | Count |
|---|---|---|
| v09c | earlier attempts not recorded (llm-agent 2) | 0 |
| v09c | affordability not established (evidence_unavailable) | 2 |
