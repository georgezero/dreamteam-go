# V0.12a scripted cases: high-impact action acknowledgment

llm-agent 6, `llm-agent-acknowledgment-workflow/v1` (`b9cb79828f26…`), contract `dreamteam.action-acknowledgment/v1` (`33c116da979a…`). Scripted model replies and a scripted engine whose preview answers have the real engine's shape; no model and no engine. Made by `analysis/acknowledgment/scripted_cases.py`; the full records are in `scripted-cases.json`.

Inputs: the message each attempt answered. Items: the consequence items the engine's answer required acknowledging. Matches: each acknowledgment's result. Withheld: why a proposal was not submitted. Submitted: actions handed to the harness. Replay: the workflow re-run with the recorded replies, without a model.

| Case | Inputs | Items | Matches | Outcome | Withheld | Submitted | Replay |
|---|---|---|---|---|---|---|---|
| ordinary_allocation | prompt | — | — | actions | — | 1 | identical |
| operations_only | prompt | — | — | actions | — | 1 | identical |
| termination_acknowledged | prompt → acknowledgment | terminate | accepted | actions | — | 1 | identical |
| termination_not_acknowledged | prompt → acknowledgment → acknowledgment_rejected | terminate | missing; missing | fallback | acknowledgment_not_accepted | 0 | identical |
| displaced_acknowledged | prompt → acknowledgment | terminate, staffing_batch | accepted | actions | — | 1 | identical |
| roster_displacement | prompt → acknowledgment | staffing_batch | accepted | actions | — | 1 | identical |
| hire_acknowledged | prompt → acknowledgment | hire | accepted | actions | — | 1 | identical |
| teleradiology_acknowledged | prompt → acknowledgment | contract_teleradiology | accepted | actions | — | 1 | identical |
| multiple_qualifying | prompt → acknowledgment | hire, terminate, terminate, staffing_batch | accepted | actions | — | 3 | identical |
| one_missing | prompt → acknowledgment | terminate, terminate | missing_item | fallback | acknowledgment_not_accepted | 0 | identical |
| wrong_employee | prompt → acknowledgment | terminate | wrong_target | fallback | acknowledgment_not_accepted | 0 | identical |
| wrong_type | prompt → acknowledgment | terminate | wrong_type | fallback | acknowledgment_not_accepted | 0 | identical |
| wrong_severance | prompt → acknowledgment | terminate | wrong_value | fallback | acknowledgment_not_accepted | 0 | identical |
| wrong_displaced_count | prompt → acknowledgment | terminate, staffing_batch | wrong_value | fallback | acknowledgment_not_accepted | 0 | identical |
| stale | prompt → acknowledgment → acknowledgment → acknowledgment_rejected | terminate | stale_proposal,wrong_target,wrong_value; stale_proposal,wrong_target,wrong_value | fallback | acknowledgment_not_accepted | 0 | identical |
| revised_after_acknowledgment | prompt → acknowledgment → acknowledgment | terminate | proposal_changed; accepted | actions | — | 1 | identical |
| generic | prompt → acknowledgment | terminate | malformed | fallback | acknowledgment_not_accepted | 0 | identical |
| agreement_in_prose | prompt → acknowledgment | terminate | missing | fallback | acknowledgment_not_accepted | 0 | identical |
| unparseable_then_acknowledged | prompt → acknowledgment → retry | terminate | accepted | actions | — | 1 | identical |
| timeout_then_acknowledged | prompt → acknowledgment → resend | terminate | accepted | actions | — | 1 | identical |
| consequences_unreadable | prompt | — | — | fallback | consequences_unavailable | 0 | identical |
| preview_unavailable | prompt | — | — | fallback | consequences_unavailable | 0 | identical |
| mixed_acknowledged | prompt → acknowledgment | terminate | accepted | actions | — | 2 | identical |
| mixed_withheld | prompt → acknowledgment | terminate | missing | fallback | acknowledgment_not_accepted | 0 | identical |
| refused | prompt → validation | — | — | no_action | — | 0 | identical |
| retries_exhausted | prompt → acknowledgment → acknowledgment_rejected → acknowledgment_rejected | terminate | wrong_target; wrong_target; wrong_target | fallback | acknowledgment_not_accepted | 0 | identical |
| final_check_disagrees | prompt → acknowledgment | terminate | accepted | fallback | final_check_disagrees | 0 | identical |

Acknowledgment is not approval. A matching acknowledgment shows that the reply stated the engine's reported direct consequences of the exact proposal; it does not show that the action is wise, and the engine's direct consequences are not all future outcomes.
