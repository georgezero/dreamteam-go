# V0.13h scripted fidelity

Workflow `llm-agent-alternatives-workflow/v3` (`9ac657fe9f66…`). Historical contexts included: yes. Failed checks: **0**.

| Case | Context | Arm | Result | Checks passed | Failed |
|---|---|---|---|---|---|
| initial invalid-number feedback disclosure | F-initial-number | control | not_due:validation_first_no_action | 10 | — |
| partial termination commitments | F-two-terms | treatment | selected | 16 | — |
| roster capacity and withdrawal | F-roster | treatment | retained | 15 | — |
| mixed unsupported capacity disclosure | F-term-tele | treatment | retained | 15 | — |
| system schema despite lower template | S1-a | control | revised | 13 | — |
| empty revision | S1-a | treatment | revised | 12 | — |
| invalid correction | F-refused | control | not_due:validation_first_fallback | 10 | — |
| deliberate correction to no action | F-refused | treatment | not_due:validation_first_no_action | 10 | — |
| retained | S1-a | control | retained | 15 | — |
| retained | S1-a | treatment | retained | 14 | — |
| alternative selected | S1-a | treatment | selected | 16 | — |
| no action selected | S1-a | treatment | selected_no_action | 12 | — |
| revision | S1-a | control | revised | 15 | — |
| revision | S1-a | treatment | revised | 14 | — |
| abandonment | S1-a | control | abandoned | 13 | — |
| abandonment | S1-a | treatment | abandoned | 12 | — |
| invalid selection | S1-a | treatment | invalid_alternative | 12 | — |
| invalid selection | S1-a | control | invalid_alternative | 13 | — |
| missing comparison (turn not due) | F-rule | control | not_due:rule | 11 | — |
| missing comparison (turn not due) | F-rule | treatment | not_due:rule | 11 | — |
| malicious free text | F-hostile | treatment | retained | 16 | — |
| mixed operations and staffing: select | F-mixed | treatment | selected | 15 | — |
| mixed operations and staffing: no action | F-mixed | treatment | selected_no_action | 15 | — |
| mixed operations and staffing: revise the whole reply | F-mixed | control | revised | 15 | — |
| mixed operations and staffing: abandon | F-mixed | control | abandoned | 13 | — |
| refused initial proposal | F-refused | treatment | retained | 14 | — |
| retry exhaustion | S1-a | treatment | no_decision | 13 | — |
| second inspect counts as a failure | S1-a | treatment | selected | 16 | — |
| model timeout, then a decision | S1-a | control | retained | 16 | — |
| model timeouts exhaust the retries | S1-a | treatment | no_decision | 13 | — |
| inspect in control | S1-a | control | retained | 15 | — |
| numeric spelling 30 | S1-a | control | revised | 15 | — |
| numeric spelling 30.0 | S1-a | control | revised | 15 | — |
| numeric spelling 3e1 | S1-a | control | revised | 15 | — |
| numeric correction decimal | F-refused | control | not_due:rule | 12 | — |
| numeric correction exponent | F-refused | treatment | not_due:rule | 12 | — |
| 64 operations validated | S1-a | control | revised | 15 | — |
| 65 operations refused | S1-a | control | invalid_alternative | 13 | — |
| 65 operations and valid staffing refused together | S1-a | treatment | invalid_alternative | 12 | — |
| invalid operation and valid staffing refused together | S1-a | control | invalid_alternative | 13 | — |
| numeric refusal 30.1 | S1-a | control | no_decision | 14 | — |
| numeric refusal 9223372036854775808 | S1-a | control | no_decision | 14 | — |
| historical: refused proposal, the recorded correction, then the turn | fl-gemma-s41901-p28 | treatment | selected_no_action | 14 | — |
| historical: refused proposal, the recorded correction, then the turn | fl-gemma-s41901-p28 | control | retained | 15 | — |
| historical: a corrected hire with an operations action | fp-qwen-s42001-p7 | treatment | selected | 14 | — |
| historical: a corrected hire with an operations action | fp-qwen-s42001-p7 | control | abandoned | 13 | — |
| historical: termination with displaced bookings | fl-qwen-s41902-p24 | treatment | retained | 15 | — |
| historical: termination with displaced bookings | fl-qwen-s41902-p24 | control | retained | 15 | — |
| historical: teleradiology guarantee | ss-gemma-pv-s42001-p14 | treatment | selected | 14 | — |
| historical: staffing adjustment with displacement | ss-qwen-s42001-p20 | treatment | selected_no_action | 12 | — |
| historical: not due (A/A) | fp-qwen-s41901-p25 | treatment | not_due:rule | 11 | — |
| complete context transcript | fl-qwen-s41902-p24 | control | retained | 15 | — |
| complete context transcript | fl-qwen-s41902-p24 | treatment | retained | 15 | — |
| complete context transcript | ss-qwen-s42001-p20 | control | retained | 15 | — |
| complete context transcript | ss-qwen-s42001-p20 | treatment | retained | 15 | — |
| complete context transcript | ss-qwen-s42002-p7 | control | retained | 15 | — |
| complete context transcript | ss-qwen-s42002-p7 | treatment | retained | 15 | — |
| complete context transcript | ss-qwen-s42001-p1 | control | retained | 15 | — |
| complete context transcript | ss-qwen-s42001-p1 | treatment | retained | 15 | — |
| complete context transcript | ss-gemma-pv-s42001-p14 | control | retained | 15 | — |
| complete context transcript | ss-gemma-pv-s42001-p14 | treatment | retained | 15 | — |
| complete context transcript | ss-qwen-pv-s42002-p6 | control | not_due:rule | 11 | — |
| complete context transcript | ss-qwen-pv-s42002-p6 | treatment | not_due:rule | 11 | — |
| complete context transcript | fp-qwen-s41901-p25 | control | not_due:rule | 11 | — |
| complete context transcript | fp-qwen-s41901-p25 | treatment | not_due:rule | 11 | — |
| complete context transcript | fl-gemma-s41901-p28 | control | retained | 15 | — |
| complete context transcript | fl-gemma-s41901-p28 | treatment | retained | 15 | — |
| complete context transcript | fp-qwen-s42001-p7 | control | retained | 15 | — |
| complete context transcript | fp-qwen-s42001-p7 | treatment | retained | 15 | — |
| complete context transcript | S1-a | control | retained | 15 | — |
| complete context transcript | S1-a | treatment | retained | 15 | — |
| complete context transcript | S1-b | control | retained | 15 | — |
| complete context transcript | S1-b | treatment | retained | 15 | — |
| complete context transcript | S2-a | control | retained | 15 | — |
| complete context transcript | S2-a | treatment | retained | 15 | — |
| complete context transcript | S2-b | control | retained | 15 | — |
| complete context transcript | S2-b | treatment | retained | 15 | — |
| complete context transcript | S3-a | control | retained | 15 | — |
| complete context transcript | S3-a | treatment | retained | 15 | — |
| complete context transcript | S3-b | control | retained | 15 | — |
| complete context transcript | S3-b | treatment | retained | 15 | — |
| complete context transcript | S4-a | control | retained | 15 | — |
| complete context transcript | S4-a | treatment | retained | 15 | — |
| complete context transcript | S4-b | control | retained | 15 | — |
| complete context transcript | S4-b | treatment | retained | 15 | — |
| complete context transcript | S5-a | control | retained | 15 | — |
| complete context transcript | S5-a | treatment | retained | 15 | — |
| complete context transcript | S5-b | control | retained | 15 | — |
| complete context transcript | S5-b | treatment | retained | 15 | — |
| complete context transcript | S6-a | control | retained | 15 | — |
| complete context transcript | S6-a | treatment | retained | 15 | — |
| complete context transcript | S6-b | control | retained | 15 | — |
| complete context transcript | S6-b | treatment | retained | 15 | — |
| tampered turn message | S1-a | treatment | refused | 1 | — |
| tampered response body | S1-a | treatment | refused | 1 | — |
| stale comparison (another state) | S1-b | treatment | refused | 1 | — |
| interrupted draw replays with its halt | S1-a | treatment | draw_call_limit | 1 | — |
| edited halt cause | S1-a | treatment | refused | 1 | — |
| edited served model | S1-a | treatment | refused | 1 | — |
| edited usage | S1-a | treatment | refused | 1 | — |
| five consecutive transport failures stop the run before a sixth call | S1-a | treatment | PARTIAL after 5 calls | 1 | — |
| duplicate shift extensions in a revision | S1-a | control | invalid_alternative | 13 | — |
| missing engine field | S1-a | treatment | aborted | 1 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | fl-qwen-s41902-p24 | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | ss-qwen-s42001-p20 | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | ss-qwen-s42002-p7 | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | ss-qwen-s42001-p1 | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | ss-gemma-pv-s42001-p14 | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | ss-qwen-pv-s42002-p6 | both | not_due:rule | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | fp-qwen-s41901-p25 | both | not_due:rule | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | fl-gemma-s41901-p28 | both | proposal refused: no turn before a correction | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | fp-qwen-s42001-p7 | both | proposal refused: no turn before a correction | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S1-a | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S1-b | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S2-a | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S2-b | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S3-a | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S3-b | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S4-a | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S4-b | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S5-a | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S5-b | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S6-a | both | due | 3 | — |
| initial request, proposal and state equal across arms; turn due as V0.13c | S6-b | both | due | 3 | — |
| mock run, complete | S1-a, S1-b, S6-b | both | COMPLETE | 1 | — |
| partial experiment (the call budget reached) | S1-a, S1-b, S6-b | both | PARTIAL | 1 | — |
