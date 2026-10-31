# Refusal-response tables

`dreamteam.refusal-response/v2`. What was proposed next after a refusal; not why.

| Group | Model | Refusals | insufficient_funds | identical_repeated | refusal_no_longer_applicable | modified_proposal | different_proposal | no_action | evidence_unavailable | of which unavailable only for the refused payload |
|---|---|---|---|---|---|---|---|---|---|---|
| probes-v2-format | scripted probe | 7 | 7 | 0 | 0 | 0 | 1 | 1 | 5 | 3 |
| probes | scripted probe | 7 | 7 | 2 | 0 | 1 | 3 | 1 | 0 | 0 |
| v05a | Qwen3.6-35B-A3B-oQ4e-mtp | 2 | 0 | 0 | 0 | 0 | 0 | 0 | 2 | 2 |
| v05b | Qwen3.6-35B-A3B-oQ4e-mtp | 8 | 0 | 0 | 0 | 0 | 0 | 0 | 8 | 7 |
| v05c | Qwen3.6-35B-A3B-oQ4e-mtp | 103 | 0 | 0 | 0 | 0 | 3 | 10 | 90 | 84 |
| v05c | gemma-4-26B-A4B-it-oQ4e-mtp | 22 | 0 | 0 | 0 | 0 | 0 | 1 | 21 | 14 |
| v05c | gemma-4-31B-it-oQ4e-mtp | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 1 | 1 |
| v06c | Qwen3.6-35B-A3B-oQ4e-mtp | 186 | 0 | 0 | 0 | 1 | 6 | 9 | 170 | 159 |
| v06c | gemma-4-26B-A4B-it-oQ4e-mtp | 32 | 0 | 0 | 0 | 1 | 1 | 1 | 29 | 23 |
| v07c | Qwen3.6-35B-A3B-oQ4e-mtp | 137 | 0 | 0 | 0 | 0 | 8 | 11 | 118 | 107 |
| v07c | gemma-4-26B-A4B-it-oQ4e-mtp | 68 | 0 | 0 | 0 | 3 | 2 | 2 | 61 | 55 |

## Probe refusals

| Read as | Recording | Point | Attempt | Refused | Code | Response | Category |
|---|---|---|---|---|---|---|---|
| attempts | `sequence-nocredit.decisions.jsonl` | 0 | 1 | hire | insufficient_funds | point 1 attempt 1 | identical_repeated |
| attempts | `sequence-nocredit.decisions.jsonl` | 1 | 1 | hire | insufficient_funds | point 1 attempt 2 | identical_repeated |
| attempts | `sequence-nocredit.decisions.jsonl` | 1 | 2 | hire | insufficient_funds | point 2 attempt 1 | different_proposal |
| attempts | `sequence-nocredit.decisions.jsonl` | 2 | 1 | set_reading_allocation | insufficient_funds | point 2 attempt 2 | modified_proposal |
| attempts | `sequence-nocredit.decisions.jsonl` | 3 | 1 | extend_shift | insufficient_funds | point 4 attempt 1 | different_proposal |
| attempts | `sequence-nocredit.decisions.jsonl` | 4 | 1 | contract_teleradiology | insufficient_funds | point 5 attempt 1 | no_action |
| attempts | `sequence-nocredit.decisions.jsonl` | 6 | 1 | extend_shift | insufficient_funds | point 6 attempt 2 | different_proposal |
| llm-agent 2 format | `sequence-nocredit.decisions.jsonl` | 0 | 1 | hire | insufficient_funds | point 1 attempt 1 | evidence_unavailable |
| llm-agent 2 format | `sequence-nocredit.decisions.jsonl` | 1 | 1 | hire | insufficient_funds | point 1 attempt 2 | evidence_unavailable |
| llm-agent 2 format | `sequence-nocredit.decisions.jsonl` | 1 | 2 | hire | insufficient_funds | point 2 attempt 1 | evidence_unavailable |
| llm-agent 2 format | `sequence-nocredit.decisions.jsonl` | 2 | 1 | set_reading_allocation | insufficient_funds | point 2 attempt 2 | evidence_unavailable |
| llm-agent 2 format | `sequence-nocredit.decisions.jsonl` | 3 | 1 | extend_shift | insufficient_funds | point 4 attempt 1 | different_proposal |
| llm-agent 2 format | `sequence-nocredit.decisions.jsonl` | 4 | 1 | contract_teleradiology | insufficient_funds | point 5 attempt 1 | no_action |
| llm-agent 2 format | `sequence-nocredit.decisions.jsonl` | 6 | 1 | extend_shift | insufficient_funds | point 6 attempt 2 | evidence_unavailable |
