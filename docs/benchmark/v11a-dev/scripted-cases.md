# llm-agent 5: scripted cases

`llm-agent-workflow/v1` (`3ee7d66f4f10…`), one decision point per case, scripted model replies and scripted engine answers (`agents/llm/test_llm_agent5.py`). Made by `analysis/workflow/scripted_cases.py`.

| Case | Inputs, in order | Preview | Eligibility reasons | Refusals | Outcome | Actions submitted | State before submission |
|---|---|---|---|---|---|---|---|
| abandon_after_preview | prompt → preview | shown | commitment, obligation | — | no_action | 0 | unchanged |
| budget | prompt → retry | — | — | — | fallback | 0 | unchanged |
| displaced | prompt → preview | shown | obligation, termination, displacement | — | actions | 1 | unchanged |
| invalid_output | prompt → retry | — | — | — | actions | 1 | unchanged |
| keep | prompt → preview | shown | commitment, obligation | — | actions | 1 | unchanged |
| mixed | prompt → preview | shown | commitment, obligation | — | actions | 2 | unchanged |
| mixed_operations_refused | prompt → validation | not_eligible | — | set_walkin_policy:invalid_args | actions | 1 | unchanged |
| mutate | prompt → preview | shown | commitment, obligation | — | fallback | 0 | changed (nothing submitted) |
| not_eligible | prompt | not_eligible | — | — | actions | 1 | unchanged |
| operations_only | prompt | — | — | — | actions | 1 | unchanged |
| preview_error | prompt | failed | — | — | actions | 1 | unchanged |
| refused_abandoned | prompt → validation | — | — | set_reading_allocation:insufficient_funds | no_action | 0 | unchanged |
| refused_corrected | prompt → validation | not_eligible | — | set_reading_allocation:insufficient_funds | actions | 1 | unchanged |
| refused_corrected_then_previewed | prompt → validation → preview | shown | commitment, obligation | set_reading_allocation:insufficient_funds | actions | 1 | unchanged |
| repeated_invalid | prompt → validation | — | — | set_reading_allocation:insufficient_funds | fallback | 0 | unchanged |
| retained_obligation | prompt → preview | shown | obligation | — | actions | 1 | unchanged |
| revise_after_preview | prompt → preview | shown | commitment, obligation | — | actions | 1 | unchanged |
| revised_fails | prompt → preview → validation | shown | commitment, obligation | set_reading_allocation:insufficient_funds | no_action | 0 | unchanged |
| timeout | prompt → resend | — | — | — | actions | 1 | unchanged |
| too_large | prompt | too_large | commitment, obligation | — | actions | 1 | unchanged |
