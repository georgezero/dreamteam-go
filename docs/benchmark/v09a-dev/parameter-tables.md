# Parameter evidence: `dreamteam.parameter-evidence/v1`

Analyzer `parameter-evidence 1`, corpus `v09a-parameter-corpus-1`. Every count describes recorded replies; none is a measure of model quality. `unresolved` and `insufficient_evidence` are abstentions.

## By rule

| Rule | Eligible | Consistent | Supported | Unresolved | Insufficient evidence |
|---|---|---|---|---|---|
| AMOUNT_MISMATCH | 21 | 20 | 0 | 1 | 0 |
| TARGET_MISMATCH | 493 | 489 | 0 | 4 | 0 |
| DATE_MISMATCH | 146 | 105 | 0 | 41 | 0 |
| DURATION_MISMATCH | 142 | 136 | 0 | 6 | 0 |
| DIRECTION_MISMATCH | 14 | 12 | 0 | 2 | 0 |

## Coverage by group and recording format

| Group, format | Recordings | Points | Replies read | Reason missing | Revised | Statements | Excluded (hedged / negated / past / several types) | No action of the type | Checks |
|---|---|---|---|---|---|---|---|---|---|
| v05c, llm-agent 2 | 18 | 651 | 651 | 0 | 4 | 203 | 83 / 40 / 3 / 0 | 6 | 114 |
| v06c, llm-agent 2 | 40 | 1800 | 1792 | 0 | 12 | 676 | 178 / 252 / 25 / 7 | 7 | 323 |
| v07c, llm-agent 2 | 30 | 1500 | 1496 | 0 | 14 | 608 | 178 / 200 / 19 / 1 | 9 | 293 |
| v08c, llm-agent 3 | 8 | 240 | 268 | 0 | 10 | 131 | 27 / 38 / 3 / 0 | 5 | 86 |

## Checks by action type and rule

| Action type | Rule | Consistent | Supported | Unresolved | Insufficient |
|---|---|---|---|---|---|
| contract_teleradiology | TARGET_MISMATCH | 11 | 0 | 0 | 0 |
| contract_teleradiology | DATE_MISMATCH | 10 | 0 | 7 | 0 |
| extend_shift | TARGET_MISMATCH | 430 | 0 | 4 | 0 |
| extend_shift | DATE_MISMATCH | 73 | 0 | 33 | 0 |
| extend_shift | DURATION_MISMATCH | 136 | 0 | 6 | 0 |
| hire | DATE_MISMATCH | 7 | 0 | 0 | 0 |
| set_reading_allocation | AMOUNT_MISMATCH | 20 | 0 | 1 | 0 |
| set_reading_allocation | TARGET_MISMATCH | 27 | 0 | 0 | 0 |
| set_reading_allocation | DATE_MISMATCH | 15 | 0 | 1 | 0 |
| set_reading_allocation | DIRECTION_MISMATCH | 12 | 0 | 2 | 0 |
| terminate | TARGET_MISMATCH | 21 | 0 | 0 | 0 |

## Supported mismatches (0)


## Abstentions by reason

| Rule | Status | Basis | Checks |
|---|---|---|---|
| DATE_MISMATCH | unresolved | The submitted action does not state date; its default is not inferred. | 27 |
| DATE_MISMATCH | unresolved | Not bound to one action: several possible actions. | 11 |
| DURATION_MISMATCH | unresolved | Not bound to one action: several possible actions. | 4 |
| TARGET_MISMATCH | unresolved | Several staff are named in the statement; this one has no submitted action, and the statement may name them for another reason. | 4 |
| DURATION_MISMATCH | unresolved | The statement gives several different values. | 2 |
| AMOUNT_MISMATCH | unresolved | Not bound to one action: several possible actions. | 1 |
| DATE_MISMATCH | unresolved | A relative date the record does not establish. | 1 |
| DATE_MISMATCH | unresolved | Not bound to one action: several staff named, not each with one submitted action. | 1 |
| DATE_MISMATCH | unresolved | The statement gives several different dates. | 1 |
| DIRECTION_MISMATCH | unresolved | A change is already pending, or the submitted value equals the allocation in effect. | 1 |
| DIRECTION_MISMATCH | unresolved | Not bound to one action: several possible actions. | 1 |
