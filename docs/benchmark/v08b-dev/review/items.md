# Decision evidence review items

Sample `v08b-review-1` of the frozen `dreamteam.decision-evidence/v1` outputs: 98 items, in a seeded order that mixes rules and outputs. Review each item on its own evidence; label it in form.csv (see PROTOCOL.md). The label is about this rule output, not about the model.

### R001 — FACTUAL_OBSERVATION_MISMATCH · rule output: consistent

* Identifier: `rv1-a3c4c98911b5391c`
* Recording: `docs/benchmark/v06c-dev/compare-30/demand-growth-gemma4-26b-a4b-s41203-decisions.jsonl`, decision point 9 (2026-10-22T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a reading backlog of 13 studies, including 2 overdue 'soon' reports. Dr. Avery starts in 30 minutes (07:30) and will begin addressing the backlog. No immediate operational changes are required.
* Relevant span: «2 overdue 'soon»
* What the rule checked: `{"kind": "overdue_priority", "observed": 2, "observedPath": "worklist.items[overdue, priority soon]", "stated": 2}`
* Prompt excerpt (what the model was shown): `…n point 9: Thu 2026-10-22 07:00 (day 10). Practice phase: closed; hours 08:00–20:00; scheduled close today 20:00. ⏎ Decide what to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - [warn] 2 soon reports overdue (target 4h) · oldest 11h 18m — no radiologist in a site-reading block now · next block 07:30 ⏎ - Reading backlog: 13 unread, 0 being read, 0 urgent, 2 overdue, oldest 793 min; today…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R002 — EXPLICIT_OPPOSITE_ACTION · rule output: consistent

* Identifier: `rv1-806526e5879197b8`
* Recording: `docs/benchmark/v07c-dev/stage-a/demand-growth-credit-gemma4-26b-a4b-s41602-decisions.jsonl`, decision point 8 (2026-10-21T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has 3 overdue 'soon' reports and a growing backlog of 13 unread studies. Since the next radiologist shift starts in 30 minutes (07:30), I will take no action and allow the scheduled shift to address the backlog.
* Relevant span: «I will take no action»
* Proposed actions: `[]`
* What the rule checked: `{"flags": [], "kind": "no_action", "params": {}, "targets": [], "type": null}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R003 — FACTUAL_OBSERVATION_MISMATCH · rule output: consistent

* Identifier: `rv1-d8aa3c3dd6ad79bc`
* Recording: `docs/benchmark/v06c-dev/compare-30/stable-qwen36-35b-a3b-s41202-decisions.jsonl`, decision point 19 (2026-11-01T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is closed on Sunday with no staff on-site and a reading backlog of 11 studies. Dr. Avery's shift starts at 07:30 today, which will address the immediate backlog. Extending shifts today is ineffective as there are no exams to perform or read beyond Dr. Avery's existing block, and the practice is closed. No changes are warranted at this time.
* Relevant span: «reading backlog of 11 studies»
* What the rule checked: `{"kind": "unread", "observed": 11, "observedPath": "worklist.unread", "stated": 11}`
* Prompt excerpt (what the model was shown): `Decision point 19: Sun 2026-11-01 07:00 (day 20). Practice phase: closed; hours ; scheduled close today None. ⏎ Decide what to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - Reading backlog: 11 unread, 0 being read, 0 urgent, 0 overdue, oldest 989 min; today 0 breaches, TAT p90 0 min. ⏎ - Lost demand: 0 today, 124 since the start. ⏎ - Booking pressure (first open slot over 3 days a…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R004 — NEGATIVE_CONTROL · rule output: no_statement

* Identifier: `nc1-07b7ac5881aeb282`
* Recording: `docs/benchmark/v05c-dev/stage-d/qwen36-35b-a3b-s40805-decisions.jsonl`, decision point 80 (2027-01-01T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is closed today with no exams scheduled, so extending shifts or hiring now would incur costs without generating revenue. The new CT technologist starts on Jan 2, and the teleradiology contract is already active for nights/weekends. No immediate action is needed to address the reading backlog as the first radiologist block starts at 07:30 today.
* Proposed actions: `[]`
* Rule output: **no_statement**. The reason mentions an action, but the rules found no explicit statement of intention in it.
* Evidence completeness: `{"prompt": null}`

### R005 — FACTUAL_OBSERVATION_MISMATCH · rule output: consistent

* Identifier: `rv1-0ae1e854f8898f2e`
* Recording: `docs/benchmark/v07c-dev/stage-b/demand-growth-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 5 (2026-10-18T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog of 28 studies, including 12 overdue reports, and is currently losing money. Since no radiologist is currently in a site-reading block and the practice is closed, I will take no immediate action as the current roster and shifts are already set for the day.
* Relevant span: «reading backlog of 28 studies»
* What the rule checked: `{"kind": "unread", "observed": 28, "observedPath": "worklist.unread", "stated": 28}`
* Prompt excerpt (what the model was shown): `…target 1d) · oldest 1d 13h — no radiologist in a site-reading block now ⏎ - [warn] 4 soon reports overdue (target 4h) · oldest 16h 05m — no radiologist in a site-reading block now ⏎ - Reading backlog: 28 unread, 0 being read, 0 urgent, 12 overdue, oldest 2265 min; today 0 breaches, TAT p90 0 min. ⏎ - Lost demand: 0 today, 40 since the start. ⏎ - Operating profit month to date -$2,868 (margin…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R006 — EXPLICIT_OPPOSITE_ACTION · rule output: consistent

* Identifier: `rv1-90a5f6e7145fef0a`
* Recording: `docs/benchmark/v06c-dev/compare-30/stable-gemma4-26b-a4b-s41202-decisions.jsonl`, decision point 10 (2026-10-23T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog of 24 studies, with 7 'soon' reports already overdue. Since the next radiologist block starts at 07:30, I will do nothing now to allow the upcoming shift to address the backlog.
* Relevant span: «I will do nothing»
* Proposed actions: `[]`
* What the rule checked: `{"flags": [], "kind": "no_action", "params": {}, "targets": [], "type": null}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R007 — FINANCIAL_AMOUNT_MISMATCH · rule output: consistent

* Identifier: `rv1-de2614684270fec7`
* Recording: `docs/benchmark/v07c-dev/stage-b/stable-credit-qwen36-35b-a3b-s41701-decisions.jsonl`, decision point 79 (2026-12-31T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is currently closed on New Year's Eve with no active exams or backlogs. Cash is healthy ($187k) and lost demand is stable at 526. With no immediate operational needs and a focus on maintaining financial sustainability, doing nothing is the correct decision.
* Relevant span: «$187k»
* What the rule checked: `{"amount": "$187k", "observed": [{"cents": 18755618, "path": "finance.cashCents"}], "quantity": "cash"}`
* Prompt excerpt (what the model was shown): `…:0,"minutes":0}],"targetMin":[{"priority":"routine","count":0,"minutes":1440},{"priority":"soon","count":0,"minutes":240},{"priority":"urgent","count":0,"minutes":60}]},"finance":{"cashCents":18755618,"cashChangeTodayCents":0,"arCents":36260438,"apCents":14491755,"accruedPayrollCents":2755905,"today":{"label":"Today","from":"2026-12-31","to":"2026-12-31","days":1,"netRevenueCen…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R008 — NEGATIVE_CONTROL · rule output: no_statement

* Identifier: `nc1-7d005104b243b837`
* Recording: `docs/benchmark/v07c-dev/stage-a/staffing-shortage-credit-qwen36-35b-a3b-s41601-decisions.jsonl`, decision point 17 (2026-10-30T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice has a pending teleradiology contract (T00001) set to start on 2026-11-01, which is the earliest possible effective date (14 days notice). Since the contract is already pending and will address the reading capacity issues starting next week, and current cash is healthy ($161k) despite the monthly loss, no immediate operational changes are warranted today. The reading backlog is manageable by Dr. Avery who is on shift today until 16:00, and the single soon-overdue report can be handled within his shift or the incoming telerad coverage.
* Proposed actions: `[]`
* Rule output: **no_statement**. The reason mentions an action, but the rules found no explicit statement of intention in it.
* Evidence completeness: `{"prompt": null}`

### R009 — NEGATIVE_CONTROL · rule output: no_statement

* Identifier: `nc1-ab0093ff15dc6c3d`
* Recording: `docs/benchmark/v05b-dev/llm-decisions.jsonl`, decision point 10 (2026-10-23T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog (26 unread, 7 overdue) and no radiologist is currently reading. Dr. Avery (E07) is the only radiologist on-site today and his shift ends at 16:00. Extending his shift allows him to read the backlog during his extended time. Extending Casey (E05) and Morgan (E06) helps clear the imaging backlog to prevent future reading delays, although the primary constraint is reading capacity. Dr. Blake (E08) is off today.
* Proposed actions: `[{"date": "2026-10-23", "employee": "E07", "minutes": 180, "type": "extend_shift"}, {"date": "2026-10-23", "employee": "E05", "minutes": 120, "type": "extend_shift"}, {"date": "2026-10-23", "employee": "E06", "minutes": 120, "type": "extend_shift"}]`
* Rule output: **no_statement**. The reason mentions an action, but the rules found no explicit statement of intention in it.
* Evidence completeness: `{"prompt": null}`

### R010 — EXPLICIT_ACTION_OMITTED · rule output: consistent

* Identifier: `rv1-d6cd99bb78443600`
* Recording: `docs/benchmark/v05c-dev/stage-c/qwen36-35b-a3b-nomem-s40804-decisions.jsonl`, decision point 8 (2026-10-21T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The previous extend_shift actions exceeded the 12-hour maximum shift limits for the employees. I need to calculate the maximum allowable extension minutes for each employee based on their shift end and cap times. Dr. Avery (E07) works 07:30-16:00, cap 19:30. Max extension is 3h 30m = 210 minutes. Casey (E05) works 08:00-16:30, cap 20:00. Max extension is 3h 30m = 210 minutes. Sam (E04) works 11:30-20:00, cap 23:30. Max extension is 3h 30m = 210 minutes. Riley (E03) works 07:45-16:15, cap 19:45. Max extension is 3h 30m = 210 minutes. I will extend all four by their maximum allowed 210 minutes to maximize throughput and clear the reading backlog.
* Relevant span: «I will extend all four by their maximum allowed 210 minutes to maximize throughput and clear the reading backlog.»
* Proposed actions: `[{"date": "2026-10-21", "employee": "E07", "minutes": 210, "type": "extend_shift"}, {"date": "2026-10-21", "employee": "E05", "minutes": 210, "type": "extend_shift"}, {"date": "2026-10-21", "employee": "E04", "minutes": 210, "type": "extend_shift"}, {"date": "2026-10-21", "employee": "E03", "minutes": 210, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "not_submitted_other_refused", "submitted": false, "validate": {"code": "accepted", "ok": true}}, {"apply": null, "index": 1, "state": "not_submitted_other_refused", "submitted": false, "validate": {"code": "accepted", "ok": true}}, {"apply": null, "index": 2, "state": "refused_at_validate", "submitted": false, "validate": {"code": "after_close", "ok": false}}, {"apply": null, "index": 3, "state": "not_submitted_other_refused", "submitted": false, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": ["unresolved_target"], "kind": "commit", "params": {}, "targets": [], "type": "extend_shift"}`
* Staff in the observation: `{"E03": "Riley", "E04": "Sam", "E05": "Casey", "E07": "Dr. Avery"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R011 — STATED_ACTION_NOT_EXECUTED · rule output: supported

* Identifier: `de1-f3828fc6b488858d`
* Recording: `docs/benchmark/v05c-dev/stage-b/qwen36-35b-a3b-s40802-decisions.jsonl`, decision point 25 (2026-11-07T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The previous attempt to extend shifts by 480 minutes was refused because the maximum allowed extension is 240 minutes. I will extend the shifts of Sam (E04) and Morgan (E06) by the maximum allowed 240 minutes each for today (Saturday) to maximize Saturday capacity. I will also set the walk-in policy to 'accept_always' to capture demand given the booking pressure.
* Relevant span: «I will extend the shifts of Sam (E04) and Morgan (E06) by the maximum allowed 240 minutes each for today (Saturday) to maximize Saturday capacity.»
* Proposed actions: `[{"date": "2026-11-07", "employee": "E04", "minutes": 240, "type": "extend_shift"}, {"date": "2026-11-07", "employee": "E06", "minutes": 240, "type": "extend_shift"}, {"policy": "accept_always", "type": "set_walkin_policy"}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "refused_at_validate", "submitted": false, "validate": {"code": "after_close", "ok": false}}, {"apply": null, "index": 1, "state": "refused_at_validate", "submitted": false, "validate": {"code": "after_close", "ok": false}}, {"apply": null, "index": 2, "state": "not_submitted_other_refused", "submitted": false, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E04"], "type": "extend_shift"}`
* Staff in the observation: `{"E04": "Sam", "E06": "Morgan"}`
* Rule output: **supported** (extend_shift). The stated action was proposed but not executed: refused_at_validate.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R012 — EXPLICIT_ACTION_OMITTED · rule output: consistent

* Identifier: `rv1-760d672d869eaf3e`
* Recording: `docs/benchmark/v07c-dev/stage-b/staffing-shortage-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 50 (2026-12-02T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has 8 unread studies, including 2 overdue 'soon' reports. I will extend Casey's shift to increase MRI capacity and address booking pressure, and extend Morgan's shift to provide more US capacity.
* Relevant span: «extend Morgan's shift to provide more US capacity.»
* Proposed actions: `[{"employee": "E05", "minutes": 120, "type": "extend_shift"}, {"employee": "E06", "minutes": 120, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00030", "message": "Shift extended: Casey today until 18:30 (+2h approved overtime \u00b7 \u2248$90) \u00b7 staffs MR 16:30\u201318:30 \u00b7 opens 2 MR slots for booking.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00031", "message": "Shift extended: Morgan today until 20:00 (+2h approved overtime \u00b7 \u2248$88) \u00b7 staffs US 18:00\u201320:00 \u00b7 opens 3 US slots for booking.", "ok": true}, "index": 1, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E06"], "type": "extend_shift"}`
* Staff in the observation: `{"E05": "Casey", "E06": "Morgan"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R013 — STATED_ACTION_NOT_EXECUTED · rule output: supported

* Identifier: `de1-fc129d7caa6848cd`
* Recording: `docs/benchmark/v06c-dev/compare-90/demand-growth-gemma4-26b-a4b-s41301-decisions.jsonl`, decision point 47 (2026-11-29T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The previous attempt failed due to a missing 'op' field. I am starting a teleradiology contract for Sundays to help manage the growing reading backlog and overdue reports.
* Relevant span: «I am starting a teleradiology contract for Sundays to help manage the growing reading backlog and overdue reports.»
* Proposed actions: `[{"effective": "2026-12-13", "op": "start", "plan": "standard-v1", "requestId": "telerad-sun-backlog", "type": "contract_teleradiology", "windows": ["sunday"]}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "refused_at_validate", "submitted": false, "validate": {"code": "invalid_schema", "ok": false}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"op": "start"}, "targets": [], "type": "contract_teleradiology"}`
* Rule output: **supported** (contract_teleradiology). The stated action was proposed but not executed: refused_at_validate.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R014 — REPEATED_INVALID_PROPOSAL · rule output: supported

* Identifier: `de1-ae4dfc7b913d49bb`
* Recording: `docs/benchmark/v07c-dev/stage-a/staffing-shortage-credit-qwen36-35b-a3b-s41601-decisions.jsonl`, decision point 6 (2026-10-19T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* What the rule checked: `{"code": "after_close", "message": "Dr. Blake's shift today already runs to the scheduled close: an extension after close adds no capacity (staff stay automatically while admitted patients need them).", "occurrences": [{"attempt": 2, "point": 6}, {"attempt": 1, "point": 9}], "type": "extend_shift"}`
* Rule output: **supported** (after_close). The same refused proposal (extend_shift after_close) was refused 2 times: 0 time(s) again within a decision point's retries, 1 time(s) at a later decision point.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R015 — FACTUAL_OBSERVATION_MISMATCH · rule output: consistent

* Identifier: `rv1-d535a2f629f9101b`
* Recording: `docs/benchmark/v05c-dev/stage-a/gemma4-26b-a4b-s40801-decisions.jsonl`, decision point 2 (2026-10-15T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a growing backlog of 11 unread studies, including 3 overdue 'soon' reports. Since the next radiologist block starts at 07:30, no immediate action is required to address the backlog, and no operational changes are warranted at this time.
* Relevant span: «backlog of 11 unread studies»
* What the rule checked: `{"kind": "unread", "observed": 11, "observedPath": "worklist.unread", "stated": 11}`
* Prompt excerpt (what the model was shown): `…de what to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - [warn] 3 soon reports overdue (target 4h) · oldest 12h 11m — no radiologist in a site-reading block now · next block 07:30 ⏎ - Reading backlog: 11 unread, 0 being read, 0 urgent, 3 overdue, oldest 821 min; today 1 breaches, TAT p90 0 min. ⏎ - Lost demand: 0 today, 24 since the start. ⏎ - Operating profit month to date -$2,809 (margin -…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R016 — NEGATIVE_CONTROL · rule output: no_statement

* Identifier: `nc1-f3d52ae78e4bcfd2`
* Recording: `docs/benchmark/v05c-dev/stage-b/gemma4-26b-a4b-s40802-decisions.jsonl`, decision point 19 (2026-11-01T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice is currently closed and has a significant backlog of unread studies (though currently 0, the recent trend shows high pressure). I am not making any changes as the current teleradiology contract (T00001) is already active for Sundays to address the backlog, and other capacity issues like booking pressure for US require more long-term staffing adjustments which cannot be implemented immediately without notice or lead time.
* Proposed actions: `[]`
* Rule output: **no_statement**. The reason mentions an action, but the rules found no explicit statement of intention in it.
* Evidence completeness: `{"prompt": null}`

### R017 — STATED_ACTION_NOT_EXECUTED · rule output: consistent

* Identifier: `rv1-4683c193df6caca1`
* Recording: `docs/benchmark/v07c-dev/stage-b/demand-growth-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 80 (2027-01-01T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog (30 studies) and overdue reports. I will extend the shifts of Riley and Casey to increase technologist capacity for today to help manage the volume and prevent further lost demand.
* Relevant span: «I will extend the shifts of Riley and Casey to increase technologist capacity for today to help manage the volume and prevent further lost demand.»
* Proposed actions: `[{"employee": "E03", "minutes": 120, "type": "extend_shift"}, {"employee": "E05", "minutes": 120, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00022", "message": "Shift extended: Riley today until 18:15 (+2h approved overtime \u00b7 \u2248$108) \u00b7 staffs XR 16:15\u201318:15 \u00b7 no slot newly bookable (already staffed, or no template slots then).", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00023", "message": "Shift extended: Casey today until 18:30 (+2h approved overtime \u00b7 \u2248$135) \u00b7 staffs MR 16:30\u201318:30 \u00b7 no slot newly bookable (already staffed, or no template slots then).", "ok": true}, "index": 1, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E05"], "type": "extend_shift"}`
* Staff in the observation: `{"E03": "Riley", "E05": "Casey"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R018 — NEGATIVE_CONTROL · rule output: no_statement

* Identifier: `nc1-512f4f52680cd757`
* Recording: `docs/benchmark/v07c-dev/stage-a/reporting-backlog-credit-gemma4-26b-a4b-s41601-decisions.jsonl`, decision point 27 (2026-11-09T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant negative operating profit for the month (-$9,740) and no current backlog or overdue reports. No immediate operational changes are required as the current staffing and teleradiology setup is managing the workload effectively.
* Proposed actions: `[]`
* Rule output: **no_statement**. The reason mentions an action, but the rules found no explicit statement of intention in it.
* Evidence completeness: `{"prompt": null}`

### R019 — FACTUAL_OBSERVATION_MISMATCH · rule output: consistent

* Identifier: `rv1-cce531cf23e9250f`
* Recording: `docs/benchmark/v05c-dev/stage-c/qwen36-35b-a3b-nomem-s40803-decisions.jsonl`, decision point 11 (2026-10-24T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is currently closed on a Saturday with no staff on-site and a significant reading backlog (24 unread, 1 urgent overdue). Since the teleradiology contract (T00001) is pending and not effective until Nov 1, and the radiologist allocation change (X00005) is pending for Oct 30, we cannot resolve the immediate reading backlog or open for patients today without internal resources. However, the practice is closed (07:00-10:00) and scheduled to close at 18:00. Opening today would incur significant overtime costs for technologists and potentially radiologists (if they come in) for likely low volume on a Saturday, while the reading bottleneck remains until the teleradiology contract starts. The most financially prudent action is to maintain the closed status for today to avoid unnecessary overtime costs, as the teleradiology contract will begin on Nov 1 to handle the overflow. We should ensure the teleradiology contract is active and the radiologist allocation is set for the upcoming week. Note: The teleradiology contract is already pending effective Nov 1. The radiologist allocation change is pending effective Oct 30. No immediate operational changes are needed for today (Sat Oct 24) as we are closed and cannot read reports without a radiologist on-site or remote coverage (which starts Nov 1). We will do nothing today.
* Relevant span: «The practice is currently closed»
* What the rule checked: `{"kind": "closed", "observed": true, "observedPath": "practice.open", "stated": true}`
* Prompt excerpt (what the model was shown): `Decision point 11: Sat 2026-10-24 07:00 (day 12). Practice phase: closed; hours 10:00–18:00; scheduled close today 18:00. ⏎ Decide what to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - [alert] 1 urge…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R020 — EXPLICIT_ACTION_OMITTED · rule output: consistent

* Identifier: `rv1-20b1cc2fdc3bdd0c`
* Recording: `docs/benchmark/v06c-dev/compare-90/financial-pressure-qwen36-35b-a3b-s41301-decisions.jsonl`, decision point 7 (2026-10-20T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is facing significant financial losses (-$20k MTD) and has a reading backlog. I am extending Dr. Avery's shift today to help clear the 2 unread MR studies immediately, as she is the only radiologist on-site until Dr. Blake arrives. I am also setting the worklist policy to 'oldest_first' to ensure the oldest overdue study is prioritized for reading once Dr. Avery starts.
* Relevant span: «I am extending Dr. Avery's shift today to help clear the 2 unread MR studies immediately, as she is the only radiologist on-site until Dr. Blake arrives.»
* Proposed actions: `[{"date": "2026-10-20", "employee": "E07", "minutes": 60, "type": "extend_shift"}, {"policy": "oldest_first", "type": "set_worklist_policy"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00004", "message": "Shift extended: Dr. Avery today until 17:00 (+1h approved overtime) \u00b7 on site for contrast supervision (reading blocks unchanged) 16:00\u201317:00.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "unchanged", "id": "X00005", "message": "Worklist policy is already oldest_first.", "ok": true}, "index": 1, "state": "accepted_unchanged", "submitted": true, "validate": {"code": "unchanged", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E07"], "type": "extend_shift"}`
* Staff in the observation: `{"E07": "Dr. Avery"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R021 — REPEATED_INVALID_PROPOSAL · rule output: consistent

* Identifier: `rv1-060d77aed75e9a91`
* Recording: `docs/benchmark/v07c-dev/stage-a/demand-growth-credit-gemma4-26b-a4b-s41601-decisions.jsonl`, decision point 22 (2026-11-04T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* What the rule checked: `{"code": "notice_too_short", "message": "Ending a teleradiology contract needs 30 days' notice: the earliest end is <date>.", "occurrences": [{"attempt": 2, "point": 22}], "type": "contract_teleradiology"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R022 — STATED_ACTION_NOT_EXECUTED · rule output: consistent

* Identifier: `rv1-006dc8d782ea1d8d`
* Recording: `docs/benchmark/v06c-dev/compare-90/financial-pressure-qwen36-35b-a3b-s41301-decisions.jsonl`, decision point 7 (2026-10-20T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is facing significant financial losses (-$20k MTD) and has a reading backlog. I am extending Dr. Avery's shift today to help clear the 2 unread MR studies immediately, as she is the only radiologist on-site until Dr. Blake arrives. I am also setting the worklist policy to 'oldest_first' to ensure the oldest overdue study is prioritized for reading once Dr. Avery starts.
* Relevant span: «I am also setting the worklist policy to 'oldest_first' to ensure the oldest overdue study is prioritized for reading once Dr. Avery starts.»
* Proposed actions: `[{"date": "2026-10-20", "employee": "E07", "minutes": 60, "type": "extend_shift"}, {"policy": "oldest_first", "type": "set_worklist_policy"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00004", "message": "Shift extended: Dr. Avery today until 17:00 (+1h approved overtime) \u00b7 on site for contrast supervision (reading blocks unchanged) 16:00\u201317:00.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "unchanged", "id": "X00005", "message": "Worklist policy is already oldest_first.", "ok": true}, "index": 1, "state": "accepted_unchanged", "submitted": true, "validate": {"code": "unchanged", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"policy": "oldest_first"}, "targets": [], "type": "set_worklist_policy"}`
* Staff in the observation: `{"E07": "Dr. Avery"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R023 — STATED_ACTION_NOT_EXECUTED · rule output: consistent

* Identifier: `rv1-9609d8391f631702`
* Recording: `docs/benchmark/v06c-dev/compare-30/staffing-shortage-gemma4-26b-a4b-s41203-decisions.jsonl`, decision point 12 (2026-10-25T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice is closed on Sunday, but there is a significant reading backlog of 23 studies, including one overdue 'soon' report. Since no radiologist is currently assigned to a reading block, I will start a teleradiology contract for Sundays to address the backlog and prevent further breaches, effective in 14 days to allow for the notice period.
* Relevant span: «I will start a teleradiology contract for Sundays to address the backlog and prevent further breaches, effective in 14 days to allow for the notice period.»
* Proposed actions: `[{"effective": "2026-11-08", "op": "start", "plan": "standard-v1", "requestId": "telerad-sun-backlog", "scope": "all", "type": "contract_teleradiology", "windows": ["sunday"]}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00001", "message": "Teleradiology contract T00001 (standard-v1): sunday \u00b7 all priorities \u00b7 from 2026-11-08, open-ended (earliest end 2026-12-08) \u00b7 retainer $800 a month, accrued daily \u00b7 per read XR $12.00, US $30.00, CT $45.00, MR $60.00, +20% claimed 22:00\u201308:00 \u00b7 two lanes for all contracts, 10m handoff \u00b7 payable on the 1st.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"op": "start"}, "targets": [], "type": "contract_teleradiology"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R024 — REPEATED_INVALID_PROPOSAL · rule output: supported

* Identifier: `de1-bdf39c351e83cc49`
* Recording: `docs/benchmark/v07c-dev/stage-b/staffing-shortage-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 11 (2026-10-24T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* What the rule checked: `{"code": "after_close", "message": "Morgan's shift today already runs to the scheduled close: an extension after close adds no capacity (staff stay automatically while admitted patients need them).", "occurrences": [{"attempt": 1, "point": 11}, {"attempt": 1, "point": 25}, {"attempt": 1, "point": 39}, {"attempt": 1, "point": 53}], "type": "extend_shift"}`
* Rule output: **supported** (after_close). The same refused proposal (extend_shift after_close) was refused 4 times: 0 time(s) again within a decision point's retries, 3 time(s) at a later decision point.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R025 — FINANCIAL_AMOUNT_MISMATCH · rule output: unresolved

* Identifier: `de1-20f71477bd1be188`
* Recording: `docs/benchmark/v05c-dev/stage-c/qwen36-35b-a3b-nomem-s40804-decisions.jsonl`, decision point 14 (2026-10-27T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is operating at a significant loss (-376 BP margin) with a pending teleradiology contract that will add substantial fixed costs ($31k/month) starting Nov 6. The current reading backlog is manageable (14 studies, oldest ~16 hours) and does not require immediate external help, especially given the high cost of the TR contract. The pending allocation changes for Dr. Avery and Dr. Blake (effective Nov 5) will increase salary costs significantly (+$44k/month combined) to cover 100% of reading. Given the current financial bleed and lack of urgent capacity or reading bottlenecks, the prudent strategy is to hold steady, avoid adding more fixed costs, and let the existing staff handle the current volume. I will not cancel the TR contract yet as it is already pending and might be needed for weekend/night coverage if volume picks up, but I will not make any new changes today.
* Relevant span: «$31k»
* What the rule checked: `{"amount": "$31k", "observed": [], "quantity": null}`
* Rule output: **unresolved** (unbound). A dollar amount ($31k) that no quantity rule binds: a rate (per month, week, day or hour), not an observed amount.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R026 — STATED_ACTION_NOT_EXECUTED · rule output: supported

* Identifier: `de1-e15a8ce2eafecd73`
* Recording: `docs/benchmark/v06c-dev/compare-90/financial-pressure-qwen36-35b-a3b-s41301-decisions.jsonl`, decision point 1 (2026-10-14T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The previous actions were refused due to shift cap violations. Dr. Avery's shift cap is 19:30 (current end 16:00, so max +3h30m). Dr. Blake's shift cap is 23:30 (current end 20:00, so max +3h30m). I will extend Dr. Avery by 210 minutes (3.5 hours) to 19:30 and Dr. Blake by 210 minutes to 23:30 to maximize reading coverage. For technologists, Riley (E03) cap is 19:45 (current 16:15, max +3h30m), Sam (E04) is already at cap for today (20:00). I will extend Riley by 120 minutes to help with imaging throughput.
* Relevant span: «I will extend Dr. Avery by 210 minutes (3.5 hours) to 19:30 and Dr. Blake by 210 minutes to 23:30 to maximize reading coverage.»
* Proposed actions: `[{"date": "2026-10-14", "employee": "E07", "minutes": 210, "type": "extend_shift"}, {"date": "2026-10-14", "employee": "E08", "minutes": 210, "type": "extend_shift"}, {"date": "2026-10-14", "employee": "E03", "minutes": 120, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "not_submitted_other_refused", "submitted": false, "validate": {"code": "accepted", "ok": true}}, {"apply": null, "index": 1, "state": "refused_at_validate", "submitted": false, "validate": {"code": "after_close", "ok": false}}, {"apply": null, "index": 2, "state": "not_submitted_other_refused", "submitted": false, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E07"], "type": "extend_shift"}`
* Staff in the observation: `{"E03": "Riley", "E07": "Dr. Avery", "E08": "Dr. Blake"}`
* Rule output: **supported** (extend_shift). The stated action was proposed but not executed: not_submitted_other_refused.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R027 — REPEATED_INVALID_PROPOSAL · rule output: consistent

* Identifier: `rv1-09818bf81dfcee8a`
* Recording: `docs/benchmark/v06c-dev/compare-30/staffing-shortage-gemma4-26b-a4b-s41203-decisions.jsonl`, decision point 14 (2026-10-27T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* What the rule checked: `{"code": "contract_overlap", "message": "Teleradiology contracts cannot overlap: T00001 (from <date>, open-ended) and new contract \"telerad-weeknights\" (from <date>). A successor starts on or after its predecessor's end: cancel the predecessor first or in the same batch.", "occurrences": [{"attempt": 1, "point": 14}], "type": "contract_teleradiology"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R028 — EXPLICIT_ACTION_OMITTED · rule output: unresolved

* Identifier: `de1-7c0d1bb97227838e`
* Recording: `docs/benchmark/v06c-dev/compare-30/demand-growth-qwen36-35b-a3b-s41203-decisions.jsonl`, decision point 13 (2026-10-26T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice has a critical reading backlog (32 unread, 32 overdue) and is currently closed with no radiologist reading. Extending Dr. Avery's shift does not help reading speed as reading blocks are fixed, but extending Morgan (US) and Riley (XR/CT) shifts will open capacity for booking and help clear the imaging backlog, while Dr. Avery's reading block starts at 07:30 which is imminent. However, since the practice is closed and no one is reading, the most immediate operational need is to ensure we have coverage for the day's exams and start reading the backlog. Dr. Avery's block starts at 07:30, so he will start reading soon. The backlog is the main issue. I will extend Morgan's shift to open US slots for booking (reducing lost demand) and extend Riley's shift to cover XR/CT. I will also extend Dr. Avery's shift to ensure he is available for contrast supervision if needed, though his reading block is the key. Given the financial loss, I will be conservative with overtime. I will extend Morgan and Riley for 2 hours each to open slots and help with the backlog of exams, which will eventually reduce the reading backlog. I will not extend Dr. Avery's shift as it doesn't increase reading capacity (blocks are fixed) and costs money. I will focus on imaging capacity to reduce future backlog.
* Relevant span: «I will also extend Dr. Avery's shift to ensure he is available for contrast supervision if needed, though his reading block is the key.»
* Proposed actions: `[{"date": "2026-10-26", "employee": "E06", "minutes": 120, "type": "extend_shift"}, {"date": "2026-10-26", "employee": "E03", "minutes": 120, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00020", "message": "Shift extended: Morgan today until 20:00 (+2h approved overtime \u00b7 \u2248$88) \u00b7 staffs US 18:00\u201320:00 \u00b7 opens 3 US slots for booking.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00021", "message": "Shift extended: Riley today until 18:15 (+2h approved overtime \u00b7 \u2248$72) \u00b7 staffs XR 16:15\u201318:15 \u00b7 opens 10 XR slots for booking.", "ok": true}, "index": 1, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": ["hedged"], "kind": "commit", "params": {}, "targets": ["E07"], "type": "extend_shift"}`
* Staff in the observation: `{"E03": "Riley", "E06": "Morgan", "E07": "Dr. Avery"}`
* Rule output: **unresolved** (extend_shift). An action statement with no matching proposed action, but the statement is hedged.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R029 — NEGATIVE_CONTROL · rule output: no_statement

* Identifier: `nc1-027d2e648496255e`
* Recording: `docs/benchmark/v06c-dev/compare-90/stable-gemma4-26b-a4b-s41301-decisions.jsonl`, decision point 9 (2026-10-22T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a reading backlog of 7 unread studies, with the oldest being 753 minutes old. However, the current staff roster includes two radiologists, Dr. Avery and Dr. Blake) whose shifts are about to begin or are already active, and no 'soon' or 'urgent' reports are currently overdue. No immediate operational changes are required to address the backlog or the negative month-to-date.
* Proposed actions: `[]`
* Rule output: **no_statement**. The reason mentions an action, but the rules found no explicit statement of intention in it.
* Evidence completeness: `{"prompt": null}`

### R030 — FACTUAL_OBSERVATION_MISMATCH · rule output: consistent

* Identifier: `rv1-ad4c4a1c38464ffb`
* Recording: `docs/benchmark/v05b-dev/llm-decisions.jsonl`, decision point 4 (2026-10-17T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): Dr. Avery is remote and his reading blocks are fixed; extending his shift does not add reading capacity. Morgan is off today. The practice is closed until 10:00. The reading backlog is 26 studies with 2 overdue soon reports. The next radiologist block starts at 10:00. Since I cannot change the radiologist's reading block via extend_shift, and no technologists are on shift to perform exams today (practice closed), there is no immediate operational action I can take to resolve the reading backlog or lost demand today. The system will open at 10:00 and the radiologist will start reading. I should do nothing now to avoid unnecessary costs.
* Relevant span: «The practice is closed»
* What the rule checked: `{"kind": "closed", "observed": true, "observedPath": "practice.open", "stated": true}`
* Prompt excerpt (what the model was shown): `Decision point 4: Sat 2026-10-17 07:00 (day 5). Practice phase: closed; hours 10:00–18:00; scheduled close today 18:00. ⏎ Decide what to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - [warn] 2 soon re…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R031 — REPEATED_INVALID_PROPOSAL · rule output: supported

* Identifier: `de1-8d6f7a1a2be15742`
* Recording: `docs/benchmark/v05c-dev/stage-b/qwen36-35b-a3b-s40804-decisions.jsonl`, decision point 2 (2026-10-15T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* What the rule checked: `{"code": "after_close", "message": "Dr. Blake's shift today already runs to the scheduled close: an extension after close adds no capacity (staff stay automatically while admitted patients need them).", "occurrences": [{"attempt": 2, "point": 2}, {"attempt": 1, "point": 8}, {"attempt": 1, "point": 15}, {"attempt": 1, "point": 27}], "type": "extend_shift"}`
* Rule output: **supported** (after_close). The same refused proposal (extend_shift after_close) was refused 4 times: 0 time(s) again within a decision point's retries, 3 time(s) at a later decision point.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R032 — FACTUAL_OBSERVATION_MISMATCH · rule output: supported

* Identifier: `de1-45e3429e5f0f74c8`
* Recording: `docs/benchmark/v05b-dev/llm-decisions.jsonl`, decision point 13 (2026-10-26T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is closed today (Sunday) with no staff on-site, making any shift extensions or operational changes ineffective for today. The reading backlog is severe (32 unread, 32 overdue) and the primary constraint is radiologist coverage. Dr. Blake (E08) is scheduled to start at 11:30 today, but his reading block starts at 07:30 (which has already passed or is imminent depending on exact start, but he is off until 11:30). Wait, the observation says 'next block 07:30' but Dr. Blake's shift is 11:30-20:00. The alert says 'no radiologist in a site-reading block now'. Dr. Blake's shift starts at 11:30. The 'next block 07:30' likely refers to a remote block or a misunderstanding in the alert text vs roster. Let's look closer: Dr. Blake's shift is 11:30-20:00. He is not on site until 11:30. The alert says 'next block 07:30'. This might be a remote block? No, 'site-reading block'. If Dr. Blake is not on site until 11:30, he cannot do a site reading block at 07:30. Perhaps the alert is stale or refers to a different radiologist? Dr. Avery is off today. Dr. Blake is the only radiologist on today. His shift starts at 11:30. The facility opens at 08:00. There is a gap. However, since the facility is closed to new patients and no one is on site, extending shifts today is useless. I will do nothing today and prepare for Monday when Dr. Blake starts and can begin reading. The backlog will grow, but no action can fix it today. Doing nothing is the correct decision for a closed day.
* Relevant span: «today (Sunday»
* What the rule checked: `{"kind": "weekday", "observed": "Mon", "observedPath": "weekday", "stated": "Sun"}`
* Prompt excerpt (what the model was shown): `Decision point 13: Mon 2026-10-26 07:00 (day 14). Practice phase: closed; hours 08:00–20:00; scheduled close today 20:00. ⏎ Decide what to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - [warn] 1 soon …`
* Rule output: **supported** (weekday). The reason states weekday 'Sun'; the observation shown has weekday = 'Mon'.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R033 — FINANCIAL_AMOUNT_MISMATCH · rule output: consistent

* Identifier: `rv1-d55f8d8982b4bb03`
* Recording: `docs/benchmark/v05c-dev/stage-b/qwen36-35b-a3b-s40803-decisions.jsonl`, decision point 2 (2026-10-15T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): Operating profit is positive ($438 MTD) and cash is healthy ($164k). The reading backlog (8 studies) is manageable by Dr. Avery during normal hours and does not require overtime or teleradiology yet. Lost demand is low (24 total, 0 today). MR booking pressure is moderate (5 days out). Doing nothing avoids unnecessary overtime costs and allows the practice to stabilize further.
* Relevant span: «$164k»
* What the rule checked: `{"amount": "$164k", "observed": [{"cents": 16442917, "path": "finance.cashCents"}], "quantity": "cash"}`
* Prompt excerpt (what the model was shown): `…:0,"minutes":0}],"targetMin":[{"priority":"routine","count":0,"minutes":1440},{"priority":"soon","count":0,"minutes":240},{"priority":"urgent","count":0,"minutes":60}]},"finance":{"cashCents":16442917,"cashChangeTodayCents":0,"arCents":34853292,"apCents":6123678,"accruedPayrollCents":5211468,"today":{"label":"Today","from":"2026-10-15","to":"2026-10-15","days":1,"netRevenueCent…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R034 — EXPLICIT_ACTION_OMITTED · rule output: unresolved

* Identifier: `de1-ad2ec0b09cecc6f3`
* Recording: `docs/benchmark/v07c-dev/stage-b/demand-growth-credit-qwen36-35b-a3b-s41701-decisions.jsonl`, decision point 1 (2026-10-14T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): There is one overdue soon-priority report (11h old) and a reading backlog of 15 studies. Dr. Avery's next site block starts at 07:30, which is imminent, so no immediate action is needed to start reading. However, to address the reading backlog and prevent future breaches while managing costs, I will set the worklist policy to 'oldest_first' to prioritize clearing the oldest studies. I will also extend Dr. Avery's shift slightly to ensure they can clear the backlog if needed, but given the block starts at 07:30, I will wait to see if the block clears the overdue one. Actually, extending a shift is an operations action that costs money. The most cost-effective immediate step is to ensure the worklist policy helps clear the oldest items. Since Dr. Avery is starting at 07:30, they will likely read the overdue one first due to 'urgent_then_fifo' (soon is higher than routine). The current policy is fine. I will do nothing for now as the radiologist is about to start reading, and the cost of overtime may not be justified for just one overdue soon study. The lost demand is low (7 total). The profit is negative due to fixed costs. Doing nothing is often right.
* Relevant span: «I will set the worklist policy to 'oldest_first' to prioritize clearing the oldest studies.»
* Proposed actions: `[]`
* What the rule checked: `{"flags": ["revised_in_text"], "kind": "commit", "params": {"policy": "oldest_first"}, "targets": [], "type": "set_worklist_policy"}`
* Rule output: **unresolved** (set_worklist_policy). An action statement with no matching proposed action, but the statement is revised_in_text.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R035 — FINANCIAL_AMOUNT_MISMATCH · rule output: consistent

* Identifier: `rv1-07968e6504f3dafd`
* Recording: `docs/benchmark/v06c-dev/compare-30/staffing-shortage-qwen36-35b-a3b-s41202-decisions.jsonl`, decision point 3 (2026-10-16T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is bleeding cash (-$9k MTD) with high overtime and lost demand. The immediate bottleneck is reading capacity (11 unread, 1 overdue) and imaging throughput (CT backlog). I will extend Dr. Avery's shift to start clearing the reading backlog immediately and extend Morgan's shift to open more US slots for booking. I will also hire a premium CT technologist to address the CT booking pressure (next open 4 days out) with a start date of 2026-10-37 (21 days lead time from today 10-16 is 11-06, but earliest start is 10-17, so I will use 10-37? No, 10-16 + 21 days = 11-06. Wait, 10-16 + 21 = 11-06. Let's re-calculate: Oct has 31 days. 16+21 = 37. 37-31 = 6. So Nov 6. I will hire a CT tech with start date 2026-11-06).
* Relevant span: «-$9k»
* What the rule checked: `{"amount": "-$9k", "observed": [{"cents": -924587, "path": "finance.monthToDate.operatingProfitCents"}], "quantity": "profit_mtd"}`
* Prompt excerpt (what the model was shown): `…date","from":"2026-10-01","to":"2026-10-16","days":16,"netRevenueCents":11571717,"laborCents":5649252,"fixedCents":4567737,"variableCents":1979315,"operatingExpenseCents":12496304,"operatingProfitCents":-924587,"marginBP":-799,"collectionsCents":11480003,"disbursementsCents":22620402,"netCashFlowCents":-11140399,"reportsFinal":768,"denialRateBP":420},"trailing30":{"label":"Trailing 30 da…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R036 — STATED_ACTION_NOT_EXECUTED · rule output: consistent

* Identifier: `rv1-e72158c334537967`
* Recording: `docs/benchmark/v05c-dev/stage-c/qwen36-35b-a3b-nomem-s40803-decisions.jsonl`, decision point 10 (2026-10-23T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The previous attempt failed because I tried to extend shifts that already run until the scheduled close (20:00) or are off-shift. Dr. Blake (E08) and Sam (E04) and Alex (E11) are already working until 20:00 or later. Dr. Avery (E07) works until 16:00, so extending him helps. Riley (E03) works until 16:15. Casey (E05) works until 16:30. Morgan (E06) works until 18:00. I will extend the shifts of the staff who have remaining capacity before the 20:00 close to help clear the backlog and handle any new demand. Specifically, I will extend Dr. Avery, Riley, Casey, and Morgan. I will not extend Blake, Sam, or Alex as they are already at close.
* Relevant span: «I will extend Dr. Avery, Riley, Casey, and Morgan.»
* Proposed actions: `[{"date": "2026-10-23", "employee": "E07", "minutes": 120, "type": "extend_shift"}, {"date": "2026-10-23", "employee": "E03", "minutes": 60, "type": "extend_shift"}, {"date": "2026-10-23", "employee": "E05", "minutes": 60, "type": "extend_shift"}, {"date": "2026-10-23", "employee": "E06", "minutes": 60, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00010", "message": "Shift extended: Dr. Avery today until 18:00 (+2h approved overtime) \u00b7 on site for contrast supervision (reading blocks unchanged) 16:00\u201318:00.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00011", "message": "Shift extended: Riley today until 17:15 (+1h approved overtime \u00b7 \u2248$54) \u00b7 staffs XR 16:15\u201317:15 \u00b7 no slot newly bookable (already staffed, or no template slots then).", "ok": true}, "index": 1, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00012", "message": "Shift extended: Casey today until 17:30 (+1h approved overtime \u00b7 \u2248$68) \u00b7 staffs MR 16:30\u201317:30 \u00b7 no slot newly bookable (already staffed, or no template slots then).", "ok": true}, "index": 2, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00013", "message": "Shift extended: Morgan today until 19:00 (+1h approved overtime \u00b7 \u2248$44) \u00b7 staffs US 18:00\u201319:00 \u00b7 opens 1 US slots for booking.", "ok": true}, "index": 3, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E05"], "type": "extend_shift"}`
* Staff in the observation: `{"E03": "Riley", "E05": "Casey", "E06": "Morgan", "E07": "Dr. Avery"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R037 — EXPLICIT_ACTION_OMITTED · rule output: consistent

* Identifier: `rv1-15e079f92e0b9375`
* Recording: `docs/benchmark/v05a-dev/llm-decisions.jsonl`, decision point 6 (2026-10-19T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is operating at a significant loss with a large backlog of 11 overdue reports and 56 items on the recall list. To address the immediate bottleneck in reporting and reduce the backlog, I am extending Dr. Avery's shift today to allow more time for reading studies. I am also extending Riley's shift to ensure coverage for the morning XR/CT volume, which supports throughput.
* Relevant span: «I am extending Dr. Avery's shift today to allow more time for reading studies.»
* Proposed actions: `[{"employee": "E07", "minutes": 120, "type": "extend_shift"}, {"employee": "E03", "minutes": 60, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00002", "message": "Shift extended: Dr. Avery today until 18:00 (+2h approved overtime) \u00b7 on site for contrast supervision (reading blocks unchanged) 16:00\u201318:00.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": null}, {"apply": {"code": "accepted", "id": "X00003", "message": "Shift extended: Riley today until 17:15 (+1h approved overtime \u00b7 \u2248$36) \u00b7 staffs XR 16:15\u201317:15 \u00b7 opens 5 XR slots for booking.", "ok": true}, "index": 1, "state": "accepted", "submitted": true, "validate": null}]`
* What the rule checked: `{"flags": ["unresolved_target"], "kind": "commit", "params": {}, "targets": [], "type": "extend_shift"}`
* Staff in the observation: `{"E03": null, "E07": null}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": false, "prompt": "unavailable", "proposed": true, "reason": true, "refusals": false, "validation": true}`

### R038 — STATED_ACTION_NOT_EXECUTED · rule output: consistent

* Identifier: `rv1-0babd14c08e3d766`
* Recording: `docs/benchmark/v05c-dev/stage-d/gemma4-26b-a4b-s40805-decisions.jsonl`, decision point 5 (2026-10-18T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a growing reading backlog and several overdue reports due to a lack of radiologist coverage. Since it is Sunday and no radiologist is scheduled, I will start a teleradiology contract for Sundays to address the backlog and prevent future breaches.
* Relevant span: «I will start a teleradiology contract for Sundays to address the backlog and prevent future breaches.»
* Proposed actions: `[{"effective": "2026-11-01", "op": "start", "plan": "standard-v1", "requestId": "telerad-sun-backlog", "scope": "all", "type": "contract_teleradiology", "windows": ["sunday"]}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00001", "message": "Teleradiology contract T00001 (standard-v1): sunday \u00b7 all priorities \u00b7 from 2026-11-01, open-ended (earliest end 2026-12-01) \u00b7 retainer $800 a month, accrued daily \u00b7 per read XR $12.00, US $30.00, CT $45.00, MR $60.00, +20% claimed 22:00\u201308:00 \u00b7 two lanes for all contracts, 10m handoff \u00b7 payable on the 1st.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"op": "start"}, "targets": [], "type": "contract_teleradiology"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R039 — REPEATED_INVALID_PROPOSAL · rule output: consistent

* Identifier: `rv1-c009db0083e7dc35`
* Recording: `docs/benchmark/v05c-dev/stage-c/qwen36-35b-a3b-nomem-s40804-decisions.jsonl`, decision point 10 (2026-10-23T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* What the rule checked: `{"code": "invalid_schema", "message": "contract_teleradiology op start needs plan, windows, scope and effective.", "occurrences": [{"attempt": 1, "point": 10}], "type": "contract_teleradiology"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R040 — REPEATED_INVALID_PROPOSAL · rule output: supported

* Identifier: `de1-778ba1671cd736a0`
* Recording: `docs/benchmark/v05c-dev/stage-c/qwen36-35b-a3b-nomem-s40804-decisions.jsonl`, decision point 3 (2026-10-16T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* What the rule checked: `{"code": "invalid_minutes", "message": "Minutes must be 1\u2013240.", "occurrences": [{"attempt": 1, "point": 3}, {"attempt": 1, "point": 3}, {"attempt": 1, "point": 11}, {"attempt": 1, "point": 11}], "type": "extend_shift"}`
* Rule output: **supported** (invalid_minutes). The same refused proposal (extend_shift invalid_minutes) was refused 4 times: 2 time(s) again within a decision point's retries, 1 time(s) at a later decision point.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R041 — REPEATED_INVALID_PROPOSAL · rule output: supported

* Identifier: `de1-02543fd824729aea`
* Recording: `docs/benchmark/v06c-dev/compare-30/financial-pressure-qwen36-35b-a3b-s41203-decisions.jsonl`, decision point 7 (2026-10-20T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* What the rule checked: `{"code": "exceeds_max_shift", "message": "Casey would work until 20:30, past the 12-hour limit at 20:00 (at most +3h 30m more).", "occurrences": [{"attempt": 2, "point": 7}, {"attempt": 1, "point": 8}], "type": "extend_shift"}`
* Rule output: **supported** (exceeds_max_shift). The same refused proposal (extend_shift exceeds_max_shift) was refused 2 times: 0 time(s) again within a decision point's retries, 1 time(s) at a later decision point.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R042 — FINANCIAL_AMOUNT_MISMATCH · rule output: unresolved

* Identifier: `de1-d1edb219c9a1ebaf`
* Recording: `docs/benchmark/v06c-dev/compare-90/staffing-shortage-qwen36-35b-a3b-s41301-decisions.jsonl`, decision point 89 (2027-01-10T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): Cash is critically negative and operating profit is deeply negative. The teleradiology contract T00001 is a significant fixed cost (~$15k/month) that should be cancelled to stop the bleeding, as we have no reading capacity and are losing money. I will cancel the contract effective at its earliest end date. I will also reduce Dr. Blake's reading allocation to 0 to save salary costs, effective at the earliest date.
* Relevant span: «~$15k»
* What the rule checked: `{"amount": "~$15k", "observed": [], "quantity": null}`
* Rule output: **unresolved** (unbound). A dollar amount (~$15k) that no quantity rule binds: a rate (per month, week, day or hour), not an observed amount.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R043 — EXPLICIT_ACTION_OMITTED · rule output: consistent

* Identifier: `rv1-6aa8084bac43737e`
* Recording: `docs/benchmark/v05b-dev/llm-decisions.jsonl`, decision point 6 (2026-10-19T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog (24 unread, 24 overdue) and is operating at a loss. The primary constraint is radiologist reading capacity. Dr. Blake (E08) is on-site until 20:00 and can read. Extending his shift to the cap (23:30) will allow him to read more studies in the evening, helping to clear the backlog. Dr. Avery is remote and extending his shift does not add reading capacity. I will extend Dr. Blake's shift by 210 minutes (3.5 hours) to 23:30 today.
* Relevant span: «I will extend Dr. Blake's shift by 210 minutes (3.5 hours) to 23:30 today.»
* Proposed actions: `[{"date": "2026-10-19", "employee": "E08", "minutes": 210, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "refused_at_validate", "submitted": false, "validate": {"code": "after_close", "ok": false}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E08"], "type": "extend_shift"}`
* Staff in the observation: `{"E08": "Dr. Blake"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R044 — EXPLICIT_ACTION_OMITTED · rule output: consistent

* Identifier: `rv1-42de40f520f5460a`
* Recording: `docs/benchmark/v06c-dev/compare-30/financial-pressure-gemma4-26b-a4b-s41202-decisions.jsonl`, decision point 19 (2026-11-01T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a reading backlog of 9 studies, with the oldest being nearly 17 hours old. Since it is Sunday and no radiologists are currently scheduled to work, I will initiate a teleradiology contract to address the backlog and prevent further delays.
* Relevant span: «I will initiate a teleradiology contract to address the backlog and prevent further delays.»
* Proposed actions: `[{"effective": "2026-11-15", "op": "start", "plan": "standard-v1", "requestId": "telerad-sun-nov", "scope": "all", "type": "contract_teleradiology", "windows": ["sunday"]}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00001", "message": "Teleradiology contract T00001 (standard-v1): sunday \u00b7 all priorities \u00b7 from 2026-11-15, open-ended (earliest end 2026-12-15) \u00b7 retainer $800 a month, accrued daily \u00b7 per read XR $12.00, US $30.00, CT $45.00, MR $60.00, +20% claimed 22:00\u201308:00 \u00b7 two lanes for all contracts, 10m handoff \u00b7 payable on the 1st.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"op": "start"}, "targets": [], "type": "contract_teleradiology"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R045 — FINANCIAL_AMOUNT_MISMATCH · rule output: unresolved

* Identifier: `de1-9f14a21bb8c64846`
* Recording: `docs/benchmark/v07c-dev/stage-b/demand-growth-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 48 (2026-11-30T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice is currently stable with no unread studies or overdue reports, but the monthly profit is significantly negative (-$11,671). Since there is no immediate operational pressure or backlog, no changes are warranted at this time.
* Relevant span: «-$11,671»
* What the rule checked: `{"amount": "-$11,671", "observed": [], "quantity": null}`
* Rule output: **unresolved** (unbound). A dollar amount (-$11,671) that no quantity rule binds: no quantity rule names this amount.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R046 — STATED_ACTION_NOT_EXECUTED · rule output: supported

* Identifier: `de1-bd4764c503b0e55e`
* Recording: `docs/benchmark/v07c-dev/stage-b/reporting-backlog-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 5 (2026-10-18T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The previous attempt failed due to a missing 'op' field. I am starting a teleradiology contract for Sundays to address the growing reading backlog and overdue reports.
* Relevant span: «I am starting a teleradiology contract for Sundays to address the growing reading backlog and overdue reports.»
* Proposed actions: `[{"effective": "2026-11-01", "op": "start", "plan": "standard-v1", "requestId": "telerad-sun-backlog", "type": "contract_teleradiology", "windows": ["sunday"]}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "refused_at_validate", "submitted": false, "validate": {"code": "invalid_schema", "ok": false}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"op": "start"}, "targets": [], "type": "contract_teleradiology"}`
* Rule output: **supported** (contract_teleradiology). The stated action was proposed but not executed: refused_at_validate.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R047 — FACTUAL_OBSERVATION_MISMATCH · rule output: supported

* Identifier: `de1-795746bf6b28834f`
* Recording: `docs/benchmark/v06c-dev/compare-90/staffing-shortage-gemma4-26b-a4b-s41301-decisions.jsonl`, decision point 14 (2026-10-27T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog of 17 unread studies (all routine) and negative operating profit. However, the current unread count is 0, and the next radiologist block starts in 30 minutes. No immediate operational or staffing changes are required at this moment.
* Relevant span: «reading backlog of 17 unread studies»
* What the rule checked: `{"kind": "unread", "observed": 0, "observedPath": "worklist.unread", "stated": 17}`
* Prompt excerpt (what the model was shown): `…ldOver":24,"skippedBreaks":0,"overtimeMin":2021,"overtimeCostCents":96648,"approvedOvertimeMin":0,"approvedOvertimeCostCents":0},"missedNow":0,"noShowsLastClose":3,"waiting":null},"worklist":{"policy":"urgent_then_fifo","unread":0,"reading":0,"urgent":0,"items":null,"oldestMin":0,"byModality":[{"key":"XR","unread":0,"reading":0},{"key":"US","unread":0,"reading":0},{"key":"CT","u…`
* Rule output: **supported** (unread). The reason states unread 17; the observation shown has worklist.unread = 0.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R048 — FINANCIAL_AMOUNT_MISMATCH · rule output: unresolved

* Identifier: `de1-45df36115c881479`
* Recording: `docs/benchmark/v07c-dev/stage-a/reporting-backlog-credit-qwen36-35b-a3b-s41601-decisions.jsonl`, decision point 2 (2026-10-15T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): There is a reading backlog of 15 unread studies with 3 overdue, but the current radiologist (Dr. Avery) is scheduled to start reading at 07:30, which is imminent. Extending shifts today is expensive and Dr. Avery's shift already ends at 16:00 (extended previously to 18:00 yesterday, but today's shift is 07:30-16:00). The most cost-effective immediate action is to ensure Dr. Avery reads as much as possible during their current shift. Since Dr. Avery is already on site and the next reading block starts at 07:30 (which is now), no immediate operational change is needed to start reading. However, to prevent future overdue reports and reduce overtime costs, I should consider extending Dr. Avery's shift today to allow more reading time, or hiring a teleradiology contract for the evening. Given the high cost of radiologist overtime ($462 for 2 hours) vs teleradiology fees, and the fact that Dr. Avery is already present, extending Dr. Avery's shift by a moderate amount (e.g., 60-120 mins) to clear the backlog is a reasonable step. Let's extend Dr. Avery by 60 minutes to 17:00 to clear the most urgent items. Also, Dr. Blake starts at 11:30, so there will be a gap. I will also extend Dr. Blake's shift? No, Dr. Blake is already until 20:00. I will extend Dr. Avery to 17:00.
* Relevant span: «$462 »
* What the rule checked: `{"amount": "$462", "observed": [], "quantity": null}`
* Rule output: **unresolved** (unbound). A dollar amount ($462) that no quantity rule binds: no quantity rule names this amount.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R049 — REPEATED_INVALID_PROPOSAL · rule output: supported

* Identifier: `de1-91c85d886024aa41`
* Recording: `docs/benchmark/v07c-dev/stage-b/staffing-shortage-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 4 (2026-10-17T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* What the rule checked: `{"code": "not_on_shift", "message": "Casey is not rostered today.", "occurrences": [{"attempt": 1, "point": 4}, {"attempt": 1, "point": 40}, {"attempt": 1, "point": 46}, {"attempt": 1, "point": 47}, {"attempt": 1, "point": 53}, {"attempt": 1, "point": 54}, {"attempt": 1, "point": 60}, {"attempt": 1, "point": 74}, {"attempt": 1, "point": 81}, {"attempt": 1, "point": 88}], "type": "extend_shift"}`
* Rule output: **supported** (not_on_shift). The same refused proposal (extend_shift not_on_shift) was refused 10 times: 0 time(s) again within a decision point's retries, 9 time(s) at a later decision point.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R050 — STATED_ACTION_NOT_EXECUTED · rule output: supported

* Identifier: `de1-8f9aab5ad5ca1432`
* Recording: `docs/benchmark/v07c-dev/stage-a/demand-growth-credit-gemma4-26b-a4b-s41601-decisions.jsonl`, decision point 20 (2026-11-02T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The previous attempt failed due to a missing 'scope' field. I am starting a teleradiology contract for weeknights to address booking pressure and workload, ensuring all required fields are included.
* Relevant span: «I am starting a teleradiology contract for weeknights to address booking pressure and workload, ensuring all required fields are included.»
* Proposed actions: `[{"effective": "2026-11-16", "op": "start", "plan": "standard-v1", "requestId": "telerad-weeknights-nov26", "scope": "all", "type": "contract_teleradiology", "windows": ["weeknights"]}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "refused_at_validate", "submitted": false, "validate": {"code": "contract_overlap", "ok": false}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"op": "start"}, "targets": [], "type": "contract_teleradiology"}`
* Rule output: **supported** (contract_teleradiology). The stated action was proposed but not executed: refused_at_validate.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R051 — EXPLICIT_OPPOSITE_ACTION · rule output: consistent

* Identifier: `rv1-0c3404e14995ed19`
* Recording: `docs/benchmark/v05b-dev/llm-decisions.jsonl`, decision point 5 (2026-10-18T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is closed on Sunday with no staff on-site, making immediate operational changes like shift extensions impossible or ineffective for today. The reading backlog is significant but cannot be addressed without radiologist coverage. I will do nothing today and reassess when the practice opens and staff are available.
* Relevant span: «I will do nothing»
* Proposed actions: `[]`
* What the rule checked: `{"flags": ["deferred"], "kind": "no_action", "params": {}, "targets": [], "type": null}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R052 — EXPLICIT_ACTION_OMITTED · rule output: supported

* Identifier: `de1-0fa62e1438c57675`
* Recording: `docs/benchmark/v07c-dev/stage-b/staffing-shortage-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 29 (2026-11-11T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog with urgent and soon reports overdue. I will extend Morgan's shift to increase US capacity and help address the backlog, although the primary issue is the lack of a radiologist in a site-reading block. Since the next block starts at 07:30, I will wait for the radiologist to arrive to clear the backlog.
* Relevant span: «I will extend Morgan's shift to increase US capacity and help address the backlog, although the primary issue is the lack of a radiologist in a site-reading block.»
* Proposed actions: `[]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E06"], "type": "extend_shift"}`
* Staff in the observation: `{"E06": "Morgan"}`
* Rule output: **supported** (extend_shift). The reason states a extend_shift for E06; the same reply proposes no such action (0 proposed).
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R053 — STATED_ACTION_NOT_EXECUTED · rule output: supported

* Identifier: `de1-276b9aa02c057a33`
* Recording: `docs/benchmark/v05c-dev/stage-c/qwen36-35b-a3b-nomem-s40802-decisions.jsonl`, decision point 17 (2026-10-30T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): Dr. Blake (E08) and Sam (E04) are already at their shift caps (20:00), so extending them is invalid. I will extend Dr. Avery (E07) to clear the reading backlog and Riley (E03) to ensure CT/XR capacity. I will also extend Casey (E05) and Alex (E11) to ensure MR capacity, and Morgan (E06) for US.
* Relevant span: «I will also extend Casey (E05) and Alex (E11) to ensure MR capacity, and Morgan (E06) for US.»
* Proposed actions: `[{"date": "2026-10-30", "employee": "E07", "minutes": 120, "type": "extend_shift"}, {"date": "2026-10-30", "employee": "E03", "minutes": 60, "type": "extend_shift"}, {"date": "2026-10-30", "employee": "E05", "minutes": 60, "type": "extend_shift"}, {"date": "2026-10-30", "employee": "E11", "minutes": 60, "type": "extend_shift"}, {"date": "2026-10-30", "employee": "E06", "minutes": 60, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "not_submitted_other_refused", "submitted": false, "validate": {"code": "accepted", "ok": true}}, {"apply": null, "index": 1, "state": "not_submitted_other_refused", "submitted": false, "validate": {"code": "accepted", "ok": true}}, {"apply": null, "index": 2, "state": "not_submitted_other_refused", "submitted": false, "validate": {"code": "accepted", "ok": true}}, {"apply": null, "index": 3, "state": "refused_at_validate", "submitted": false, "validate": {"code": "after_close", "ok": false}}, {"apply": null, "index": 4, "state": "not_submitted_other_refused", "submitted": false, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E05"], "type": "extend_shift"}`
* Staff in the observation: `{"E03": "Riley", "E05": "Casey", "E06": "Morgan", "E07": "Dr. Avery", "E11": "Alex"}`
* Rule output: **supported** (extend_shift). The stated action was proposed but not executed: not_submitted_other_refused.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R054 — EXPLICIT_OPPOSITE_ACTION · rule output: consistent

* Identifier: `rv1-ffaa2b9f9ae30951`
* Recording: `docs/benchmark/v05c-dev/stage-b/qwen36-35b-a3b-s40803-decisions.jsonl`, decision point 4 (2026-10-17T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): Dr. Avery is a remote radiologist; extending his shift does not extend his site-reading blocks, so it cannot clear the reading backlog. The backlog is due to lack of reading capacity during the day. Since the next reading block is at 10:00 and it is currently 07:00, I should wait for the block to start. No immediate action can clear the backlog without a reading block. I will do nothing today and reassess after the 10:00 block.
* Relevant span: «I will do nothing»
* Proposed actions: `[]`
* What the rule checked: `{"flags": [], "kind": "no_action", "params": {}, "targets": [], "type": null}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R055 — FINANCIAL_AMOUNT_MISMATCH · rule output: consistent

* Identifier: `rv1-27966f57b73e0ab6`
* Recording: `docs/benchmark/v07c-dev/stage-b/staffing-shortage-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 62 (2026-12-14T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice is starting a new week with no unread studies or overdue reports, but the month-to-date operating profit is significantly negative (-$10,180). No immediate operational changes are required for today's capacity as the backlog is clear.
* Relevant span: «-$10,180»
* What the rule checked: `{"amount": "-$10,180", "observed": [{"cents": -1018048, "path": "finance.monthToDate.operatingProfitCents"}], "quantity": "profit_mtd"}`
* Prompt excerpt (what the model was shown): `…date","from":"2026-12-01","to":"2026-12-14","days":14,"netRevenueCents":10195121,"laborCents":5429439,"fixedCents":3996770,"variableCents":1641391,"operatingExpenseCents":11213169,"operatingProfitCents":-1018048,"marginBP":-998,"collectionsCents":8623607,"disbursementsCents":18484188,"netCashFlowCents":-9860581,"reportsFinal":689,"denialRateBP":642},"trailing30":{"label":"Trailing 30 days…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R056 — FACTUAL_OBSERVATION_MISMATCH · rule output: insufficient_evidence

* Identifier: `de1-640652e86590c10e`
* Recording: `docs/benchmark/v05a-dev/llm-decisions.jsonl`, decision point 4 (2026-10-17T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): It is Saturday morning with no staff on-site and no appointments scheduled; the practice is closed. There are 3 overdue 'soon' reports, but the radiologist (Dr. Avery) is not on-site until 10:00 and cannot read remotely without a site block. Extending shifts or hiring now would incur costs for a day with zero demand and no capacity to utilize them effectively. The best course is to do nothing and wait for the scheduled opening.
* Relevant span: «the practice is closed»
* What the rule checked: `{"kind": "closed", "observed": null, "observedPath": null, "stated": true}`
* Rule output: **insufficient_evidence** (closed). A factual claim, but the prompt the model saw is not available verified (not rebuildable from this recording).
* Evidence completeness: `{"apply": null, "observation": false, "prompt": "unavailable", "proposed": true, "reason": true, "refusals": false, "validation": true}`

### R057 — REPEATED_INVALID_PROPOSAL · rule output: consistent

* Identifier: `rv1-ec5d29a595a7b745`
* Recording: `docs/benchmark/v05b-dev/llm-decisions.jsonl`, decision point 4 (2026-10-17T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* What the rule checked: `{"code": "no_coverage", "message": "Dr. Avery would cover no room, desk or contrast supervision in the extra time today (a remote radiologist's reading blocks are not extended): nothing to gain.", "occurrences": [{"attempt": 1, "point": 4}], "type": "extend_shift"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R058 — FACTUAL_OBSERVATION_MISMATCH · rule output: supported

* Identifier: `de1-8c247df20413cf57`
* Recording: `docs/benchmark/v07c-dev/stage-b/demand-growth-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 21 (2026-11-03T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog of 35 studies, all of which are overdue. Since the next radiologist block starts in 30 minutes (07:30), no immediate operational changes are required at 07:00.
* Relevant span: «reading backlog of 35 studies»
* What the rule checked: `{"kind": "unread", "observed": 0, "observedPath": "worklist.unread", "stated": 35}`
* Prompt excerpt (what the model was shown): `…dOver":37,"skippedBreaks":0,"overtimeMin":3026,"overtimeCostCents":496795,"approvedOvertimeMin":0,"approvedOvertimeCostCents":0},"missedNow":0,"noShowsLastClose":5,"waiting":null},"worklist":{"policy":"urgent_then_fifo","unread":0,"reading":0,"urgent":0,"items":null,"oldestMin":0,"byModality":[{"key":"XR","unread":0,"reading":0},{"key":"US","unread":0,"reading":0},{"key":"CT","u…`
* Rule output: **supported** (unread). The reason states unread 35; the observation shown has worklist.unread = 0.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R059 — FACTUAL_OBSERVATION_MISMATCH · rule output: supported

* Identifier: `de1-0f128a57209702f0`
* Recording: `docs/benchmark/v05c-dev/stage-b/gemma4-26b-a4b-s40804-decisions.jsonl`, decision point 12 (2026-10-25T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog of 46 studies, including 21 overdue routine reports and 1 overdue 'soon' report. Since it is Sunday and no radiologist is currently scheduled for a site-reading block, I will take no action now to allow the current staff roster to address the backlog when they start their shifts.
* Relevant span: «21 overdue routine»
* What the rule checked: `{"kind": "overdue_priority", "observed": 20, "observedPath": "worklist.items[overdue, priority routine]", "stated": 21}`
* Prompt excerpt (what the model was shown): `Decision point 12: Sun 2026-10-25 07:00 (day 13). Practice phase: closed; hours ; scheduled close today None. ⏎ Decide what to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - [warn] 20 routine reports overdue (target 1d) · oldest 1d 15h — no radiologist in a site-reading block now ⏎ - [warn] Reporting backlog: 46 studies unread or being read — no radiologist in a site-reading block now ⏎ -…`
* Rule output: **supported** (overdue_priority). The reason states overdue priority 21 (routine); the observation shown has worklist.items[overdue, priority routine] = 20.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R060 — EXPLICIT_ACTION_OMITTED · rule output: supported

* Identifier: `de1-e073136cd67bea71`
* Recording: `docs/benchmark/v05c-dev/stage-c/qwen36-35b-a3b-nomem-s40803-decisions.jsonl`, decision point 28 (2026-11-10T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is losing money (-4653 BP margin) with significant fixed costs. I have already scheduled a CT hire (H00001) and reduced radiologist allocations to cut salary costs. To further improve margins, I will terminate the departing technologist's notice early (if possible, but contract requires 14 days, so I will keep the current termination) and reduce the roster of the most expensive underutilized staff (Casey E05 and Alex E11 for MR, and Sam E04 for CT/XR) to match lower demand, while ensuring XR coverage remains. Specifically, I will reduce Casey's and Alex's hours as MR demand is low and they are expensive, and reduce Sam's hours as the new CT tech arrives soon. I will also extend Riley's shift slightly to cover more XR/CT volume efficiently.
* Relevant span: «I will also extend Riley's shift slightly to cover more XR/CT volume efficiently.»
* Proposed actions: `[{"effective": "2026-11-30", "employee": "E05", "op": "set", "type": "set_roster"}, {"effective": "2026-11-30", "employee": "E11", "op": "set", "type": "set_roster"}, {"effective": "2026-11-30", "employee": "E04", "op": "set", "type": "set_roster"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00021", "message": "Roster change: Casey (E05) from Mon Nov 30: rostered 20 h a week (was 40 h a week) \u00b7 contract 40 h unchanged \u00b7 contracted time not rostered is still paid (20 h a week).", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00022", "message": "Roster change: Alex (E11) from Mon Nov 30: rostered 0 h a week (was 40 h a week) \u00b7 contract 40 h unchanged \u00b7 contracted time not rostered is still paid (40 h a week).", "ok": true}, "index": 1, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00023", "message": "Roster change: Sam (E04) from Mon Nov 30: rostered 16 h a week (was 39.5 h a week) \u00b7 contract 39.5 h unchanged \u00b7 contracted time not rostered is still paid (23.5 h a week).", "ok": true}, "index": 2, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E03"], "type": "extend_shift"}`
* Staff in the observation: `{"E03": "Riley", "E04": "Sam", "E05": "Casey", "E11": "Alex"}`
* Rule output: **supported** (extend_shift). The reason states a extend_shift for E03; the same reply proposes no such action (3 proposed).
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R061 — FACTUAL_OBSERVATION_MISMATCH · rule output: supported

* Identifier: `de1-7b21aeb3029a95eb`
* Recording: `docs/benchmark/v07c-dev/stage-a/financial-pressure-credit-gemma4-26b-a4b-s41601-decisions.jsonl`, decision point 21 (2026-11-03T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog of 10 studies, all of which are overdue. However, the current staff roster is about to start their shifts, and no immediate operational changes are required to address the backlog as the radiologists will begin reading shortly.
* Relevant span: «reading backlog of 10 studies»
* What the rule checked: `{"kind": "unread", "observed": 0, "observedPath": "worklist.unread", "stated": 10}`
* Prompt excerpt (what the model was shown): `…dOver":30,"skippedBreaks":0,"overtimeMin":2663,"overtimeCostCents":457599,"approvedOvertimeMin":0,"approvedOvertimeCostCents":0},"missedNow":0,"noShowsLastClose":3,"waiting":null},"worklist":{"policy":"urgent_then_fifo","unread":0,"reading":0,"urgent":0,"items":null,"oldestMin":0,"byModality":[{"key":"XR","unread":0,"reading":0},{"key":"US","unread":0,"reading":0},{"key":"CT","u…`
* Rule output: **supported** (unread). The reason states unread 10; the observation shown has worklist.unread = 0.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R062 — EXPLICIT_ACTION_OMITTED · rule output: unresolved

* Identifier: `de1-77fd2f3fde215dd8`
* Recording: `docs/benchmark/v05c-dev/stage-b/gemma4-26b-a4b-s40802-decisions.jsonl`, decision point 1 (2026-10-14T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog with 3 'soon' priority studies already overdue, and no radiologist is currently on site. I will extend the shifts of the current technologists to ensure they can handle the incoming demand and prevent further backlog growth, although the immediate priority is the radiologist's reading block which starts at 07:30.
* Relevant span: «I will extend the shifts of the current technologists to ensure they can handle the incoming demand and prevent further backlog growth, although the immediate priority is the radiologist's reading block which starts at 07:30.»
* Proposed actions: `[]`
* What the rule checked: `{"flags": ["unresolved_target"], "kind": "commit", "params": {}, "targets": [], "type": "extend_shift"}`
* Rule output: **unresolved** (extend_shift). An action statement with no matching proposed action, but the statement is unresolved_target (no named target).
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R063 — STATED_ACTION_NOT_EXECUTED · rule output: supported

* Identifier: `de1-15a82bda410b211d`
* Recording: `docs/benchmark/v05c-dev/stage-b/gemma4-26b-a4b-s40802-decisions.jsonl`, decision point 7 (2026-10-20T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The previous attempt failed because the 'scope' field was missing. I will start a teleradiology contract for weeknights to address booking pressure, ensuring the effective date respects the 14-day notice period and the scope is set to 'all'.
* Relevant span: «I will start a teleradiology contract for weeknights to address booking pressure, ensuring the effective date respects the 14-day notice period and the scope is set to 'all'.»
* Proposed actions: `[{"effective": "2026-11-03", "op": "start", "plan": "standard-v1", "requestId": "telerad-weeknights-capacity", "scope": "all", "type": "contract_teleradiology", "windows": ["weeknights"]}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "refused_at_validate", "submitted": false, "validate": {"code": "contract_overlap", "ok": false}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"op": "start"}, "targets": [], "type": "contract_teleradiology"}`
* Rule output: **supported** (contract_teleradiology). The stated action was proposed but not executed: refused_at_validate.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R064 — EXPLICIT_ACTION_OMITTED · rule output: unresolved

* Identifier: `de1-54c65138a6896adc`
* Recording: `docs/benchmark/v06c-dev/compare-30/staffing-shortage-gemma4-26b-a4b-s41202-decisions.jsonl`, decision point 5 (2026-10-18T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog with 4 overdue 'soon' reports and no radiologist currently reading. I will extend a teleradiology contract to address the backlog and prevent future breaches, although it will take effect on 2026-11-01.
* Relevant span: «I will extend a teleradiology contract to address the backlog and prevent future breaches, although it will take effect on 2026-11-01.»
* Proposed actions: `[{"effective": "2026-11-01", "op": "start", "plan": "standard-v1", "requestId": "telerad-backlog-fix", "scope": "all", "type": "contract_teleradiology", "windows": ["weeknights", "saturday", "sunday"]}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00002", "message": "Teleradiology contract T00001 (standard-v1): weeknights, saturday, sunday \u00b7 all priorities \u00b7 from 2026-11-01, open-ended (earliest end 2026-12-01) \u00b7 retainer $3,100 a month, accrued daily \u00b7 per read XR $12.00, US $30.00, CT $45.00, MR $60.00, +20% claimed 22:00\u201308:00 \u00b7 two lanes for all contracts, 10m handoff \u00b7 payable on the 1st.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": ["unresolved_target"], "kind": "commit", "params": {}, "targets": [], "type": "extend_shift"}`
* Rule output: **unresolved** (extend_shift). An action statement with no matching proposed action, but the statement is unresolved_target (no named target).
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R065 — FACTUAL_OBSERVATION_MISMATCH · rule output: supported

* Identifier: `de1-a2030ec213af2d64`
* Recording: `docs/benchmark/v06c-dev/compare-90/staffing-shortage-qwen36-35b-a3b-s41301-decisions.jsonl`, decision point 22 (2026-11-04T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is currently closed (07:00, opens 08:00) with no staff on-site, so no extend_shift actions are possible today. The reading backlog is cleared (0 unread, 0 overdue) and exams are flowing normally. The pending allocation change for Dr. Avery takes effect on Nov 7. No immediate operational changes are needed.
* Relevant span: «0 unread»
* What the rule checked: `{"kind": "unread", "observed": 4, "observedPath": "worklist.unread", "stated": 0}`
* Prompt excerpt (what the model was shown): `…ecision point 22: Wed 2026-11-04 07:00 (day 23). Practice phase: closed; hours 08:00–20:00; scheduled close today 20:00. ⏎ Decide what to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - Reading backlog: 4 unread, 0 being read, 0 urgent, 0 overdue, oldest 652 min; today 0 breaches, TAT p90 0 min. ⏎ - Lost demand: 0 today, 241 since the start. ⏎ - Operating profit month to date -$13,946 (margin…`
* Rule output: **supported** (unread). The reason states unread 0; the observation shown has worklist.unread = 4.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R066 — NEGATIVE_CONTROL · rule output: no_statement

* Identifier: `nc1-480338168c51e76e`
* Recording: `docs/benchmark/v05c-dev/stage-b/gemma4-31b-s40802-decisions.jsonl`, decision point 0 (2026-10-13T07:00:00); model gemma-4-31B-it-oQ4e-mtp
* Reason (whole): The practice is starting the day with a small reading backlog (2 studies) and a negative month-to-date profit. No immediate operational or staffing changes are required at this time; the current roster and policies are sufficient to handle today's scheduled demand.
* Proposed actions: `[]`
* Rule output: **no_statement**. The reason mentions an action, but the rules found no explicit statement of intention in it.
* Evidence completeness: `{"prompt": null}`

### R067 — EXPLICIT_ACTION_OMITTED · rule output: supported

* Identifier: `de1-c9470e4e65386ec4`
* Recording: `docs/benchmark/v07c-dev/stage-b/staffing-shortage-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 21 (2026-11-03T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice is facing significant booking pressure for MRI (next open in 14 days) and a negative operating profit for the month. I will extend Morgan's shift today to increase US capacity and address potential demand, although the primary bottleneck is MRI. Since no other immediate operational or staffing changes are valid for today, I will take no action to avoid unnecessary costs while monitoring the MRI gap.
* Relevant span: «I will extend Morgan's shift today to increase US capacity and address potential demand, although the primary bottleneck is MRI.»
* Proposed actions: `[]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E06"], "type": "extend_shift"}`
* Staff in the observation: `{"E06": "Morgan"}`
* Rule output: **supported** (extend_shift). The reason states a extend_shift for E06; the same reply proposes no such action (0 proposed).
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R068 — FINANCIAL_AMOUNT_MISMATCH · rule output: unresolved

* Identifier: `de1-28681922981e5b76`
* Recording: `docs/benchmark/v06c-dev/compare-90/reporting-backlog-gemma4-26b-a4b-s41301-decisions.jsonl`, decision point 7 (2026-10-20T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The reading backlog has been cleared (1 unread, 0 overdue), but the practice is operating at a significant monthly loss (-$9,811). Since the current staff and schedule are managing the workload effectively without further immediate changes are required.
* Relevant span: «-$9,811»
* What the rule checked: `{"amount": "-$9,811", "observed": [], "quantity": null}`
* Rule output: **unresolved** (unbound). A dollar amount (-$9,811) that no quantity rule binds: no quantity rule names this amount.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R069 — FACTUAL_OBSERVATION_MISMATCH · rule output: insufficient_evidence

* Identifier: `de1-0c60b8b04d5747f0`
* Recording: `docs/benchmark/v05a-dev/llm-decisions.jsonl`, decision point 5 (2026-10-18T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): It is Sunday with no demand and closed operations; no immediate actions are needed. The practice has a backlog of 11 unread studies and a TAT warning for 2 overdue 'soon' reports, but since the facility is closed today, I will wait to adjust staffing and reading allocations when operations resume on Monday to ensure efficient coverage and timely reporting.
* Relevant span: «2 overdue 'soon»
* What the rule checked: `{"kind": "overdue_priority", "observed": null, "observedPath": null, "stated": 2}`
* Rule output: **insufficient_evidence** (overdue_priority). A factual claim, but the prompt the model saw is not available verified (not rebuildable from this recording).
* Evidence completeness: `{"apply": null, "observation": false, "prompt": "unavailable", "proposed": true, "reason": true, "refusals": false, "validation": true}`

### R070 — REPEATED_INVALID_PROPOSAL · rule output: supported

* Identifier: `de1-49fd6233606ba280`
* Recording: `docs/benchmark/v06c-dev/compare-30/staffing-shortage-gemma4-26b-a4b-s41201-decisions.jsonl`, decision point 2 (2026-10-15T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* What the rule checked: `{"code": "after_close", "message": "Sam's shift today already runs to the scheduled close: an extension after close adds no capacity (staff stay automatically while admitted patients need them).", "occurrences": [{"attempt": 1, "point": 2}, {"attempt": 1, "point": 29}], "type": "extend_shift"}`
* Rule output: **supported** (after_close). The same refused proposal (extend_shift after_close) was refused 2 times: 0 time(s) again within a decision point's retries, 1 time(s) at a later decision point.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R071 — FINANCIAL_AMOUNT_MISMATCH · rule output: supported

* Identifier: `de1-fb2661ae251a19e6`
* Recording: `docs/benchmark/v07c-dev/stage-b/staffing-shortage-credit-qwen36-35b-a3b-s41701-decisions.jsonl`, decision point 59 (2026-12-11T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): Cash is critically low ($2,698) with a $21,336 payroll due on 12-25 and $45,945 in bills due 01-01, while receivables are $347k. Terminating remaining notice staff (H00002, H00001, E01, E03, E04, E05, E06) immediately reduces future contract pay obligations and avoids further cash drain, though severance is due 12-25. However, since all are already in notice and severance is fixed, the primary lever is to stop any new hiring costs and ensure no overtime is incurred. The most urgent issue is the cash shortfall. Since we cannot change payment dates, we must minimize outflows. All staff are already in notice. H00002's last day is today (12-11). Terminating others now doesn't save immediate cash (severance is same) but stops future weekly pay. Given the extreme cash low, I will terminate all remaining notice staff to stop their weekly contract pay immediately after their last day, effectively minimizing the burn rate until collections arrive. I will also set the walk-in policy to 'accept_if_gap' (already set) and do nothing else as hiring is too slow and costly.
* Relevant span: «$21,336 »
* What the rule checked: `{"amount": "$21,336", "observed": [{"cents": 213356, "path": "finance.nextPayroll.cents"}, {"cents": 213356, "path": "finance.accruedPayrollCents"}], "quantity": "payroll_due"}`
* Prompt excerpt (what the model was shown): `…621},"claims":{"total":11272,"open":2412,"pending":166,"submitted":2128,"denied":118,"paid":7858,"shortPaid":467,"writtenOff":535,"daysInAR":45},"nextPayroll":{"date":"2026-12-25","cents":213356},"nextPayables":{"date":"2027-01-01","cents":4594461},"severanceDue":[{"date":"2026-12-25","cents":2719500}],"balanceSheet":{"cashCents":269762,"receivablesCents":34705839,"assets…`
* Rule output: **supported** (payroll_due). The reason states payroll_due as $21,336 (equal, precision $1); the prompt showed finance.nextPayroll.cents = $2,133; finance.accruedPayrollCents = $2,133 (one of several alternatives).
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R072 — EXPLICIT_ACTION_OMITTED · rule output: unresolved

* Identifier: `de1-9611afd8754ce415`
* Recording: `docs/benchmark/v07c-dev/stage-b/staffing-shortage-credit-gemma4-26b-a4b-s41701-decisions.jsonl`, decision point 6 (2026-10-19T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog with several overdue reports. Since the next radiologist block starts at 07:30, I will extend the shifts of the current technologists to ensure capacity for the day, though the primary issue is the lack of radiologist coverage for the backlog.
* Relevant span: «I will extend the shifts of the current technologists to ensure capacity for the day, though the primary issue is the lack of radiologist coverage for the backlog.»
* Proposed actions: `[]`
* What the rule checked: `{"flags": ["unresolved_target"], "kind": "commit", "params": {}, "targets": [], "type": "extend_shift"}`
* Rule output: **unresolved** (extend_shift). An action statement with no matching proposed action, but the statement is unresolved_target (no named target).
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R073 — REPEATED_INVALID_PROPOSAL · rule output: supported

* Identifier: `de1-c5bcddc0b9c38d8c`
* Recording: `docs/benchmark/v05b-dev/llm-decisions.jsonl`, decision point 1 (2026-10-14T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* What the rule checked: `{"code": "after_close", "message": "Dr. Blake's shift today already runs to the scheduled close: an extension after close adds no capacity (staff stay automatically while admitted patients need them).", "occurrences": [{"attempt": 1, "point": 1}, {"attempt": 1, "point": 3}, {"attempt": 2, "point": 6}, {"attempt": 1, "point": 7}], "type": "extend_shift"}`
* Rule output: **supported** (after_close). The same refused proposal (extend_shift after_close) was refused 4 times: 0 time(s) again within a decision point's retries, 3 time(s) at a later decision point.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R074 — REPEATED_INVALID_PROPOSAL · rule output: supported

* Identifier: `de1-5046fa5a22906d2d`
* Recording: `docs/benchmark/v05c-dev/stage-b/gemma4-26b-a4b-s40802-decisions.jsonl`, decision point 7 (2026-10-20T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* What the rule checked: `{"code": "contract_overlap", "message": "Teleradiology contracts cannot overlap: T00001 (from <date>, open-ended) and new contract \"telerad-weeknights-capacity\" (from <date>). A successor starts on or after its predecessor's end: cancel the predecessor first or in the same batch.", "occurrences": [{"attempt": 2, "point": 7}, {"attempt": 2, "point": 8}, {"attempt": 1, "point": 9}, {"attempt": 2, "point": 10}], "type": "contract_teleradiology"}`
* Rule output: **supported** (contract_overlap). The same refused proposal (contract_teleradiology contract_overlap) was refused 4 times: 0 time(s) again within a decision point's retries, 3 time(s) at a later decision point.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R075 — FACTUAL_OBSERVATION_MISMATCH · rule output: supported

* Identifier: `de1-00d77fddd9dc6b0b`
* Recording: `docs/benchmark/v05c-dev/stage-b/qwen36-35b-a3b-s40802-decisions.jsonl`, decision point 21 (2026-11-03T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is currently closed (Sunday night/Monday morning transition) and the teleradiology contract is active to handle reading volume. The pending increase in Dr. Avery's allocation takes effect on Nov 6. No immediate operational or staffing changes are required today as the practice is closed and existing measures are in place.
* Relevant span: «currently closed (Sunday»
* What the rule checked: `{"kind": "weekday", "observed": "Tue", "observedPath": "weekday", "stated": "Sun"}`
* Prompt excerpt (what the model was shown): `Decision point 21: Tue 2026-11-03 07:00 (day 22). Practice phase: closed; hours 08:00–20:00; scheduled close today 20:00. ⏎ Decide what to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - Lost demand: 0…`
* Rule output: **supported** (weekday). The reason states weekday 'Sun'; the observation shown has weekday = 'Tue'.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R076 — FACTUAL_OBSERVATION_MISMATCH · rule output: supported

* Identifier: `de1-269c4ab7d42958c1`
* Recording: `docs/benchmark/v07c-dev/stage-b/staffing-shortage-credit-qwen36-35b-a3b-s41701-decisions.jsonl`, decision point 7 (2026-10-20T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is currently closed (Sunday) with no staff on shift, making extend_shift actions invalid. The new technologist starts Nov 11 and Dr. Blake's allocation change takes effect Oct 30; no immediate operational changes can be made today. Doing nothing is the correct decision until the practice reopens.
* Relevant span: «currently closed (Sunday»
* What the rule checked: `{"kind": "weekday", "observed": "Tue", "observedPath": "weekday", "stated": "Sun"}`
* Prompt excerpt (what the model was shown): `Decision point 7: Tue 2026-10-20 07:00 (day 8). Practice phase: closed; hours 08:00–20:00; scheduled close today 20:00. ⏎ Decide what to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - Lost demand: 0 t…`
* Rule output: **supported** (weekday). The reason states weekday 'Sun'; the observation shown has weekday = 'Tue'.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R077 — FACTUAL_OBSERVATION_MISMATCH · rule output: supported

* Identifier: `de1-ec85f52d8152ab66`
* Recording: `docs/benchmark/v06c-dev/compare-90/reporting-backlog-gemma4-26b-a4b-s41301-decisions.jsonl`, decision point 41 (2026-11-23T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog of 29 studies, with 29 overdue routine reports and 2 overdue 'soon' reports. A radiologist (Dr. Avery) is scheduled to start a reading block in 30 minutes (07:30), which should address the backlog. No immediate operational changes are required.
* Relevant span: «29 overdue routine»
* What the rule checked: `{"kind": "overdue_priority", "observed": 27, "observedPath": "worklist.items[overdue, priority routine]", "stated": 29}`
* Prompt excerpt (what the model was shown): `…t to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - [warn] 2 soon reports overdue (target 4h) · oldest 1d 16h — no radiologist in a site-reading block now · next block 07:30 ⏎ - [warn] 27 routine reports overdue (target 1d) · oldest 2d 11h — no radiologist in a site-reading block now · next block 07:30 ⏎ - Reading backlog: 29 unread, 0 being read, 0 urgent, 29 overdue, oldest 3590 min; toda…`
* Rule output: **supported** (overdue_priority). The reason states overdue priority 29 (routine); the observation shown has worklist.items[overdue, priority routine] = 27.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R078 — STATED_ACTION_NOT_EXECUTED · rule output: supported

* Identifier: `de1-488a8a3ca55a2d98`
* Recording: `docs/benchmark/v06c-dev/compare-30/staffing-shortage-gemma4-26b-a4b-s41203-decisions.jsonl`, decision point 14 (2026-10-27T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The previous attempt failed because teleradiology contracts cannot overlap. I will cancel the existing Sunday contract (T00001) and start a new contract that covers both Sunday and weeknights to address the reading backlog and booking pressure.
* Relevant span: «I will cancel the existing Sunday contract (T00001) and start a new contract that covers both Sunday and weeknights to address the reading backlog and booking pressure.»
* Proposed actions: `[{"contract": "T00001", "endDate": "2026-12-08", "op": "cancel", "type": "contract_teleradiology"}, {"effective": "2026-11-10", "op": "start", "plan": "standard-v1", "requestId": "telerad-combined", "scope": "all", "type": "contract_teleradiology", "windows": ["sunday", "weeknights"]}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "not_submitted_batch_rejected", "submitted": false, "validate": {"code": "batch_rejected", "ok": false}}, {"apply": null, "index": 1, "state": "refused_at_validate", "submitted": false, "validate": {"code": "contract_overlap", "ok": false}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"contract": "T00001", "op": "cancel"}, "targets": [], "type": "contract_teleradiology"}`
* Rule output: **supported** (contract_teleradiology). The stated action was proposed but not executed: not_submitted_batch_rejected.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R079 — FINANCIAL_AMOUNT_MISMATCH · rule output: unresolved

* Identifier: `de1-ee235e0b82372821`
* Recording: `docs/benchmark/v05c-dev/stage-d/gemma4-26b-a4b-s40805-decisions.jsonl`, decision point 34 (2026-11-16T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has cleared its reading backlog, but the monthly operating profit is significantly negative (-$9,471). No immediate operational or staffing changes are required as the current state is stable and the backlog is zero.
* Relevant span: «-$9,471»
* What the rule checked: `{"amount": "-$9,471", "observed": [], "quantity": null}`
* Rule output: **unresolved** (unbound). A dollar amount (-$9,471) that no quantity rule binds: no quantity rule names this amount.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R080 — FINANCIAL_AMOUNT_MISMATCH · rule output: supported

* Identifier: `de1-c6b2050d2d3612d2`
* Recording: `docs/benchmark/v06c-dev/compare-90/financial-pressure-qwen36-35b-a3b-s41301-decisions.jsonl`, decision point 36 (2026-11-18T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is operating at a significant loss (-$20,976 MTD) with a $150k/month teleradiology retainer that is largely unused (only 6 claims). The reading backlog is small (7 unread, 0 overdue) and can be handled by Dr. Avery's existing shift. Extending shifts is a recurring cost that hasn't solved the profitability issue. The most impactful action is to cancel the underutilized teleradiology contract to stop the bleeding, which is allowed with 30 days notice (effective end date 2026-12-18). No immediate staffing changes are needed as the backlog is manageable.
* Relevant span: «$150k»
* What the rule checked: `{"amount": "$150k", "observed": [{"cents": 150000, "path": "contract_teleradiology.targets.cancellableContracts[T00001].monthlyRetainerCents"}, {"cents": 150000, "path": "contract_teleradiology.targets.plans[standard-v1].retainers[weeknights].monthlyCents"}, {"cents": 80000, "path": "contract_teleradiology.targets.plans[standard-v1].retainers[saturday].monthlyCents"}, {"cents": 80000, "path": "contract_teleradiology.targets.plans[standard-v1].retainers[sunday].monthlyCents"}, {"cents": 230000, "path": "plans[standard-v1] retainers weeknights+saturday"}, {"cents": 230000, "path": "plans[standard-v1] retainers weeknights+sunday"}, {"cents": 160000, "path": "plans[standard-v1] retainers saturday+sunday"}, {"cents": 310000, "path": "plans[standard-v1] retainers weeknights+saturday+sunday"}], "quantity": "teleradiology_retainer"}`
* Prompt excerpt (what the model was shown): `…questId":"tr-urgent-backlog-1016","windows":["weeknights"],"scope":"urgent","effective":"2026-10-30","status":"active","minimumTermEnd":"2026-11-29","earliestEndDate":"2026-12-18","monthlyRetainerCents":150000,"readFees":[{"modality":"XR","cents":1200},{"modal… ⏎ - staff E11: {"id":"E11","name":"Alex","role":"technologist","skills":["MR"],"employment":"terminated"} ⏎ - teleradiology: {"plan…`
* Rule output: **supported** (teleradiology_retainer:cents_read_as_dollars). The reason states teleradiology_retainer as $150k (equal, precision $1,000); the prompt showed contract_teleradiology.targets.cancellableContracts[T00001].monthlyRetainerCents = $1,500; contract_teleradiology.targets.plans[standard-v1].retainers[weeknights].monthlyCents = $1,500; contract_teleradiology.targets.plans[standard-v1].retainers[saturday].monthlyCents = $800; contract_teleradiology.targets.plans[standard-v1].retainers[sunday].monthlyCents = $800 (one of several alternatives); the stated amount is the observed one with cents read as dollars.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R081 — REPEATED_INVALID_PROPOSAL · rule output: supported

* Identifier: `de1-6c0b0efa2ca0afb3`
* Recording: `docs/benchmark/v06c-dev/compare-30/staffing-shortage-gemma4-26b-a4b-s41202-decisions.jsonl`, decision point 26 (2026-11-08T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* What the rule checked: `{"code": "invalid_roster", "message": "roster.A: 37.5 h rostered net of breaks, more than the 20 h contract (a part-time contract cannot carry a longer regular roster)", "occurrences": [{"attempt": 2, "point": 26}, {"attempt": 2, "point": 27}, {"attempt": 2, "point": 28}], "type": "hire"}`
* Rule output: **supported** (invalid_roster). The same refused proposal (hire invalid_roster) was refused 3 times: 0 time(s) again within a decision point's retries, 2 time(s) at a later decision point.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R082 — FINANCIAL_AMOUNT_MISMATCH · rule output: supported

* Identifier: `de1-57407c546dfa94d3`
* Recording: `docs/benchmark/v06c-dev/compare-90/financial-pressure-qwen36-35b-a3b-s41301-decisions.jsonl`, decision point 35 (2026-11-17T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is operating at a significant loss (-$21,812 MTD) with a teleradiology contract (T00001) costing $150k/month retainer that is underutilized (only 5 claims). The backlog is small (1 unread) and MR booking pressure is high (next slot 9 days out). I will cancel the teleradiology contract to stop the bleeding, as the local radiologists can handle the current volume and the contract is not cost-effective. I will also extend Dr. Avery's shift to ensure the single unread MR is read promptly and to maintain service levels.
* Relevant span: «$150k»
* What the rule checked: `{"amount": "$150k", "observed": [{"cents": 150000, "path": "contract_teleradiology.targets.cancellableContracts[T00001].monthlyRetainerCents"}], "quantity": "teleradiology_retainer"}`
* Prompt excerpt (what the model was shown): `…questId":"tr-urgent-backlog-1016","windows":["weeknights"],"scope":"urgent","effective":"2026-10-30","status":"active","minimumTermEnd":"2026-11-29","earliestEndDate":"2026-12-17","monthlyRetainerCents":150000,"readFees":[{"modality":"XR","cents":1200},{"modal… ⏎ - staff E11: {"id":"E11","name":"Alex","role":"technologist","skills":["MR"],"startsAt":"11:30","employment":"notice"} ⏎ - telera…`
* Rule output: **supported** (teleradiology_retainer:cents_read_as_dollars). The reason states teleradiology_retainer as $150k (equal, precision $1,000); the prompt showed contract_teleradiology.targets.cancellableContracts[T00001].monthlyRetainerCents = $1,500; the stated amount is the observed one with cents read as dollars.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R083 — FACTUAL_OBSERVATION_MISMATCH · rule output: supported

* Identifier: `de1-23c64098100f2b96`
* Recording: `docs/benchmark/v05c-dev/stage-b/qwen36-35b-a3b-s40802-decisions.jsonl`, decision point 13 (2026-10-26T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is closed today (Sunday) and the reading backlog is driven by lack of site reading coverage, which cannot be fixed by extending technologist shifts or hiring new staff (too long lead time). The pending teleradiology contract (effective Nov 1) and the increased site reading allocation for Dr. Avery (effective Nov 6) are the planned solutions. No immediate action can clear the backlog or improve finances today.
* Relevant span: «today (Sunday»
* What the rule checked: `{"kind": "weekday", "observed": "Mon", "observedPath": "weekday", "stated": "Sun"}`
* Prompt excerpt (what the model was shown): `Decision point 13: Mon 2026-10-26 07:00 (day 14). Practice phase: closed; hours 08:00–20:00; scheduled close today 20:00. ⏎ Decide what to change now, if anything. ⏎  ⏎ ACTIVE PROBLEMS ⏎ - [warn] 29 rout…`
* Rule output: **supported** (weekday). The reason states weekday 'Sun'; the observation shown has weekday = 'Mon'.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R084 — EXPLICIT_ACTION_OMITTED · rule output: unresolved

* Identifier: `de1-9c7cd97482a178af`
* Recording: `docs/benchmark/v05c-dev/stage-c/qwen36-35b-a3b-nomem-s40802-decisions.jsonl`, decision point 15 (2026-10-28T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The primary issue is a reading backlog with 7 overdue reports and no radiologist currently reading. Dr. Avery's shift ends at 16:00, but the next reading block starts at 07:30 (which has already passed or is imminent depending on exact start, but the alert says 'next block 07:30' implying it's the immediate next opportunity). However, the alert explicitly states 'no radiologist in a site-reading block now'. Extending Dr. Avery's shift ensures they are available to read the backlog immediately after their current shift or during the block. Given the financial loss and overdue reports, extending Dr. Avery's shift by 2 hours (120 mins) to cover the reading block and clear the backlog is the most direct operational fix. I will also extend Dr. Blake's shift slightly to ensure coverage for the later part of the day if needed, but Dr. Avery is the primary reader for the morning block. Actually, looking at the roster, Dr. Avery is on 07:30-16:00. Extending to 18:00 allows them to read during the 16:00-18:00 block mentioned in previous extensions. This directly addresses the 'no radiologist in a site-reading block now' warning.
* Relevant span: «I will also extend Dr. Blake's shift slightly to ensure coverage for the later part of the day if needed, but Dr. Avery is the primary reader for the morning block.»
* Proposed actions: `[{"date": "2026-10-28", "employee": "E07", "minutes": 120, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00017", "message": "Shift extended: Dr. Avery today until 18:00 (+2h approved overtime) \u00b7 on site for contrast supervision (reading blocks unchanged) 16:00\u201318:00.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": ["hedged", "deferred", "revised_in_text"], "kind": "commit", "params": {}, "targets": ["E08"], "type": "extend_shift"}`
* Staff in the observation: `{"E07": "Dr. Avery", "E08": "Dr. Blake"}`
* Rule output: **unresolved** (extend_shift). An action statement with no matching proposed action, but the statement is hedged, deferred, revised_in_text.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R085 — STATED_ACTION_NOT_EXECUTED · rule output: consistent

* Identifier: `rv1-4662210a5a09b14d`
* Recording: `docs/benchmark/v05b-dev/llm-decisions.jsonl`, decision point 7 (2026-10-20T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): Dr. Blake's shift already ends at the scheduled close (20:00), so extending it is invalid. I will extend Morgan (US) and Casey (MR) to increase imaging capacity and help clear the backlog, and extend Dr. Avery (E07) to increase reading capacity for the 2 unread MR studies.
* Relevant span: «I will extend Morgan (US) and Casey (MR) to increase imaging capacity and help clear the backlog, and extend Dr. Avery (E07) to increase reading capacity for the 2 unread MR studies.»
* Proposed actions: `[{"date": "2026-10-20", "employee": "E06", "minutes": 120, "type": "extend_shift"}, {"date": "2026-10-20", "employee": "E05", "minutes": 120, "type": "extend_shift"}, {"date": "2026-10-20", "employee": "E07", "minutes": 120, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00007", "message": "Shift extended: Morgan today until 20:00 (+2h approved overtime \u00b7 \u2248$88) \u00b7 staffs US 18:00\u201320:00 \u00b7 opens 3 US slots for booking.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00008", "message": "Shift extended: Casey today until 18:30 (+2h approved overtime \u00b7 \u2248$90) \u00b7 staffs MR 16:30\u201318:30 \u00b7 no slot newly bookable (already staffed, or no template slots then).", "ok": true}, "index": 1, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00009", "message": "Shift extended: Dr. Avery today until 18:00 (+2h approved overtime) \u00b7 on site for contrast supervision (reading blocks unchanged) 16:00\u201318:00.", "ok": true}, "index": 2, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E06"], "type": "extend_shift"}`
* Staff in the observation: `{"E05": "Casey", "E06": "Morgan", "E07": "Dr. Avery"}`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R086 — STATED_ACTION_NOT_EXECUTED · rule output: supported

* Identifier: `de1-04aa81b4601d8c49`
* Recording: `docs/benchmark/v07c-dev/stage-a/reporting-backlog-credit-qwen36-35b-a3b-s41602-decisions.jsonl`, decision point 27 (2026-11-09T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The previous attempt failed because the teleradiology contract cancellation and new contract must be in the same batch to avoid overlap, and the extension for Dr. Blake was invalid as it was after close. I will cancel the existing teleradiology contract T00001 and start a new one with Saturday coverage effective 2026-11-23 in a single batch. I will not extend shifts today as it is ineffective.
* Relevant span: «I will cancel the existing teleradiology contract T00001 and start a new one with Saturday coverage effective 2026-11-23 in a single batch.»
* Proposed actions: `[{"contract": "T00001", "endDate": "2026-12-09", "op": "cancel", "type": "contract_teleradiology"}, {"effective": "2026-11-23", "op": "start", "plan": "standard-v1", "requestId": "tr-sat-2026-11-09-v2", "scope": "all", "type": "contract_teleradiology", "windows": ["weeknights", "saturday"]}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "not_submitted_batch_rejected", "submitted": false, "validate": {"code": "batch_rejected", "ok": false}}, {"apply": null, "index": 1, "state": "refused_at_validate", "submitted": false, "validate": {"code": "contract_overlap", "ok": false}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"contract": "T00001", "op": "cancel"}, "targets": [], "type": "contract_teleradiology"}`
* Rule output: **supported** (contract_teleradiology). The stated action was proposed but not executed: not_submitted_batch_rejected.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R087 — EXPLICIT_ACTION_OMITTED · rule output: supported

* Identifier: `de1-e3ca4c7dc9fdff91`
* Recording: `docs/benchmark/v06c-dev/compare-30/stable-qwen36-35b-a3b-s41202-decisions.jsonl`, decision point 7 (2026-10-20T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is unprofitable with a reading backlog and lost demand. Extending Dr. Avery's shift has been ineffective as it only adds contrast supervision time without increasing reading capacity. To improve profitability and clear the backlog, I will reduce Dr. Avery's site allocation to the minimum (3000 BP) to lower fixed salary costs, and extend Casey's shift to increase MR exam throughput during the afternoon.
* Relevant span: «I will reduce Dr. Avery's site allocation to the minimum (3000 BP) to lower fixed salary costs, and extend Casey's shift to increase MR exam throughput during the afternoon.»
* Proposed actions: `[{"date": "2026-10-20", "employee": "E05", "minutes": 60, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00011", "message": "Shift extended: Casey today until 17:30 (+1h approved overtime \u00b7 \u2248$45) \u00b7 staffs MR 16:30\u201317:30 \u00b7 no slot newly bookable (already staffed, or no template slots then).", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"direction": "down"}, "targets": ["E07"], "type": "set_reading_allocation"}`
* Staff in the observation: `{"E05": "Casey", "E07": "Dr. Avery"}`
* Rule output: **supported** (set_reading_allocation). The reason states a set_reading_allocation for E07 (direction down); the same reply proposes no such action (1 proposed).
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R088 — FACTUAL_OBSERVATION_MISMATCH · rule output: insufficient_evidence

* Identifier: `de1-385fe408ea146433`
* Recording: `docs/benchmark/v05a-dev/llm-decisions.jsonl`, decision point 5 (2026-10-18T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): It is Sunday with no demand and closed operations; no immediate actions are needed. The practice has a backlog of 11 unread studies and a TAT warning for 2 overdue 'soon' reports, but since the facility is closed today, I will wait to adjust staffing and reading allocations when operations resume on Monday to ensure efficient coverage and timely reporting.
* Relevant span: «backlog of 11 unread studies»
* What the rule checked: `{"kind": "unread", "observed": null, "observedPath": null, "stated": 11}`
* Rule output: **insufficient_evidence** (unread). A factual claim, but the prompt the model saw is not available verified (not rebuildable from this recording).
* Evidence completeness: `{"apply": null, "observation": false, "prompt": "unavailable", "proposed": true, "reason": true, "refusals": false, "validation": true}`

### R089 — EXPLICIT_ACTION_OMITTED · rule output: supported

* Identifier: `de1-d8566d59ad338f3a`
* Recording: `docs/benchmark/v06c-dev/compare-90/financial-pressure-qwen36-35b-a3b-s41301-decisions.jsonl`, decision point 35 (2026-11-17T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice is operating at a significant loss (-$21,812 MTD) with a teleradiology contract (T00001) costing $150k/month retainer that is underutilized (only 5 claims). The backlog is small (1 unread) and MR booking pressure is high (next slot 9 days out). I will cancel the teleradiology contract to stop the bleeding, as the local radiologists can handle the current volume and the contract is not cost-effective. I will also extend Dr. Avery's shift to ensure the single unread MR is read promptly and to maintain service levels.
* Relevant span: «I will cancel the teleradiology contract to stop the bleeding, as the local radiologists can handle the current volume and the contract is not cost-effective.»
* Proposed actions: `[{"date": "2026-11-17", "employee": "E07", "minutes": 60, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00026", "message": "Shift extended: Dr. Avery today until 17:00 (+1h approved overtime) \u00b7 on site for contrast supervision (reading blocks unchanged) 16:00\u201317:00.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {"op": "cancel"}, "targets": [], "type": "contract_teleradiology"}`
* Staff in the observation: `{"E07": "Dr. Avery"}`
* Rule output: **supported** (contract_teleradiology). The reason states a contract_teleradiology (op cancel); the same reply proposes no such action (1 proposed).
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R090 — NEGATIVE_CONTROL · rule output: no_statement

* Identifier: `nc1-df4dc1f8dbb6ad89`
* Recording: `docs/benchmark/v05a-dev/llm-decisions.jsonl`, decision point 2 (2026-10-15T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The previous attempt failed because set_reading_allocation requires the effective date to be after today (2026-10-15). The earliest effective date for Dr. Avery and Dr. Blake is 2026-10-29. Since the goal is to clear the backlog and the radiologists are already scheduled to start reading at their normal times (07:30 and 11:30), and no immediate operational changes like hiring or shifting are needed right now, doing nothing is the correct action. The existing staff and schedules are in place to handle the day's operations.
* Proposed actions: `[]`
* Rule output: **no_statement**. The reason mentions an action, but the rules found no explicit statement of intention in it.
* Evidence completeness: `{"prompt": null}`

### R091 — EXPLICIT_OPPOSITE_ACTION · rule output: unresolved

* Identifier: `de1-0d23d09e4e2d3120`
* Recording: `docs/benchmark/v06c-dev/compare-30/staffing-shortage-qwen36-35b-a3b-s41203-decisions.jsonl`, decision point 2 (2026-10-15T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): Dr. Blake is already at the scheduled close time, so extending his shift adds no reading capacity. The backlog is due to lack of reading blocks. I will set Dr. Blake's reading allocation to 100% (effective 2026-10-29) to maximize his reading time once he is on-site, and also set Dr. Avery's to 100% to ensure full coverage. This addresses the root cause of the reading bottleneck for the future. For today, since I cannot extend shifts past close or cap, and no new radiologists can start, I will do nothing operationally today as the backlog will clear as Dr. Blake reads during his current block.
* Relevant span: «I will do nothing»
* Proposed actions: `[{"effective": "2026-10-29", "employee": "E08", "op": "set", "siteAllocationBP": 10000, "type": "set_reading_allocation"}, {"effective": "2026-10-29", "employee": "E07", "op": "set", "siteAllocationBP": 10000, "type": "set_reading_allocation"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00006", "message": "Allocation change: Dr. Blake (E08) from Thu Oct 29: site share 100% (was 50%) \u00b7 site reading 42.5 h (week A), 46.5 h (week B) (was 21 h 40 min (week A), 24 h (week B)) \u00b7 salary with benefits $49,315 per 30 days (+$24,658) \u00b7 shifts and contrast supervision unchanged.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00007", "message": "Allocation change: Dr. Avery (E07) from Thu Oct 29: site share 100% (was 60%) \u00b7 site reading 46.5 h (week A), 42.5 h (week B) (was 28.5 h (week A), 25 h 50 min (week B)) \u00b7 salary with benefits $49,315 per 30 days (+$19,726) \u00b7 shifts and contrast supervision unchanged.", "ok": true}, "index": 1, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": ["hedged", "says_cannot"], "kind": "no_action", "params": {}, "targets": [], "type": null}`
* Staff in the observation: `{"E07": "Dr. Avery", "E08": "Dr. Blake"}`
* Rule output: **unresolved** (no_action_stated). A statement of no action, but the statement is hedged; the reason also states an action.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R092 — STATED_ACTION_NOT_EXECUTED · rule output: supported

* Identifier: `de1-ebdef140815a8583`
* Recording: `docs/benchmark/v05b-dev/llm-decisions.jsonl`, decision point 6 (2026-10-19T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog (24 unread, 24 overdue) and is operating at a loss. The primary constraint is radiologist reading capacity. Dr. Blake (E08) is on-site until 20:00 and can read. Extending his shift to the cap (23:30) will allow him to read more studies in the evening, helping to clear the backlog. Dr. Avery is remote and extending his shift does not add reading capacity. I will extend Dr. Blake's shift by 210 minutes (3.5 hours) to 23:30 today.
* Relevant span: «I will extend Dr. Blake's shift by 210 minutes (3.5 hours) to 23:30 today.»
* Proposed actions: `[{"date": "2026-10-19", "employee": "E08", "minutes": 210, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": null, "index": 0, "state": "refused_at_validate", "submitted": false, "validate": {"code": "after_close", "ok": false}}]`
* What the rule checked: `{"flags": [], "kind": "commit", "params": {}, "targets": ["E08"], "type": "extend_shift"}`
* Staff in the observation: `{"E08": "Dr. Blake"}`
* Rule output: **supported** (extend_shift). The stated action was proposed but not executed: refused_at_validate.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R093 — FACTUAL_OBSERVATION_MISMATCH · rule output: unresolved

* Identifier: `de1-f97976f0a4c4fae4`
* Recording: `docs/benchmark/v07c-dev/stage-b/financial-pressure-credit-qwen36-35b-a3b-s41701-decisions.jsonl`, decision point 51 (2026-12-03T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice has a reading backlog of 11 studies with 3 overdue, including 1 urgent report overdue by over 10 hours. Dr. Avery (E07) starts at 07:30 and is on-site; extending her shift by 2 hours will allow her to read the backlog after her regular shift ends, preventing further breaches and clearing the urgent report.
* Relevant span: «3 overdue»
* What the rule checked: `{"kind": "overdue", "observed": 11, "observedPath": "worklist.unread", "stated": 3}`
* Prompt excerpt (what the model was shown): `…ogist in a site-reading block now · next block 07:30 ⏎ - [warn] 2 soon reports overdue (target 4h) · oldest 12h 55m — no radiologist in a site-reading block now · next block 07:30 ⏎ - Reading backlog: 11 unread, 0 being read, 1 urgent, 3 overdue, oldest 859 min; today 1 breaches, TAT p90 0 min. ⏎ - Lost demand: 0 today, 283 since the start. ⏎ - Operating profit month to date -$3,29…`
* Rule output: **unresolved** (overdue). A factual claim (overdue 3) that may not mean what the rule reads: a total beside counts by priority in the same sentence.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R094 — FINANCIAL_AMOUNT_MISMATCH · rule output: consistent

* Identifier: `rv1-52637f9340f093df`
* Recording: `docs/benchmark/v07c-dev/stage-a/reporting-backlog-credit-gemma4-26b-a4b-s41601-decisions.jsonl`, decision point 27 (2026-11-09T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant negative operating profit for the month (-$9,740) and no current backlog or overdue reports. No immediate operational changes are required as the current staffing and teleradiology setup is managing the workload effectively.
* Relevant span: «-$9,740»
* What the rule checked: `{"amount": "-$9,740", "observed": [{"cents": -974021, "path": "finance.monthToDate.operatingProfitCents"}], "quantity": "profit_mtd"}`
* Prompt excerpt (what the model was shown): `… to date","from":"2026-11-01","to":"2026-11-09","days":9,"netRevenueCents":6071417,"laborCents":3296944,"fixedCents":2655000,"variableCents":963240,"operatingExpenseCents":7045438,"operatingProfitCents":-974021,"marginBP":-1604,"collectionsCents":6022717,"disbursementsCents":12886312,"netCashFlowCents":-6863595,"reportsFinal":392,"denialRateBP":593},"trailing30":{"label":"Trailing 30 day…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R095 — FINANCIAL_AMOUNT_MISMATCH · rule output: consistent

* Identifier: `rv1-2249043c1cd92527`
* Recording: `docs/benchmark/v06c-dev/compare-90/financial-pressure-gemma4-26b-a4b-s41301-decisions.jsonl`, decision point 54 (2026-12-06T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a reading backlog of 17 studies and a significant booking pressure for Ultrasound (next open in 4 days). However, the current month-to-date operating profit is negative (-$2,876), and no immediate staffing or operational policy changes required to address the current backlog as the radiologist block is expected to handle it. Doing nothing is the most prudent decision to avoid unnecessary costs while the practice recovers.
* Relevant span: «-$2,876»
* What the rule checked: `{"amount": "-$2,876", "observed": [{"cents": -287673, "path": "finance.monthToDate.operatingProfitCents"}], "quantity": "profit_mtd"}`
* Prompt excerpt (what the model was shown): `… to date","from":"2026-12-01","to":"2026-12-06","days":6,"netRevenueCents":4486486,"laborCents":2345465,"fixedCents":1712899,"variableCents":715795,"operatingExpenseCents":4774159,"operatingProfitCents":-287673,"marginBP":-641,"collectionsCents":3417835,"disbursementsCents":12627693,"netCashFlowCents":-9209858,"reportsFinal":283,"denialRateBP":476},"trailing30":{"label":"Trailing 30 days…`
* Rule output: **consistent**. The rule checked this and found the statement and the evidence in agreement.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R096 — FACTUAL_OBSERVATION_MISMATCH · rule output: unresolved

* Identifier: `de1-c3f434994f7f2387`
* Recording: `docs/benchmark/v07c-dev/stage-a/staffing-shortage-credit-gemma4-26b-a4b-s41601-decisions.jsonl`, decision point 27 (2026-11-09T07:00:00); model gemma-4-26B-A4B-it-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog of 17 unread studies, including 2 overdue 'soon' reports and 15 overdue reports. A radiologist block starts in 30 minutes (07:30), which should address the immediate backlog. No operational or staffing changes are required at this moment.
* Relevant span: «15 overdue»
* What the rule checked: `{"kind": "overdue", "observed": 2, "observedPath": "worklist.items[overdue, priority soon]", "stated": 15}`
* Prompt excerpt (what the model was shown): `…st in a site-reading block now · next block 07:30 ⏎ - [warn] 15 routine reports overdue (target 1d) · oldest 1d 18h — no radiologist in a site-reading block now · next block 07:30 ⏎ - Reading backlog: 17 unread, 0 being read, 0 urgent, 17 overdue, oldest 2548 min; today 0 breaches, TAT p90 0 min. ⏎ - Lost demand: 0 today, 353 since the start. ⏎ - Operating profit month to date -$14…`
* Rule output: **unresolved** (overdue). A factual claim (overdue 15) that may not mean what the rule reads: a total beside counts by priority in the same sentence.
* Evidence completeness: `{"apply": null, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R097 — EXPLICIT_OPPOSITE_ACTION · rule output: unresolved

* Identifier: `de1-f35f26cb147fcb07`
* Recording: `docs/benchmark/v07c-dev/stage-a/reporting-backlog-credit-qwen36-35b-a3b-s41601-decisions.jsonl`, decision point 24 (2026-11-06T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): The practice has a significant reading backlog (24 unread, 6 overdue) and is currently closed with no radiologist on-site to read them until 07:30. The teleradiology contract T00001 is active for weekends but incurs a monthly retainer; since there is no current backlog requiring immediate weekend coverage and the backlog is growing during the week, I will not extend shifts or add teleradiology yet. However, the booking pressure is high (US next open 5 days out). To address the backlog and booking pressure, I will extend Dr. Avery's shift today to start reading earlier and cover more, and extend Riley's shift to help with imaging throughput. Wait, Dr. Avery starts at 07:30. Extending her shift *today* from 07:30 is not an 'extend_shift' in the overtime sense if she is already working. The 'extend_shift' action adds overtime. Dr. Avery's shift is 07:30-16:00. Extending it would mean working later. This helps reading. Riley (07:45-16:15) extending helps imaging. Given the profit is negative and we are losing demand, I need to be careful with costs. The backlog is the main issue. Teleradiology is expensive. I will try to clear the backlog with existing staff first by extending shifts slightly to cover the reading and imaging load, but only if it doesn't cost too much. Actually, looking at previous decisions, I did not extend shifts recently. The backlog is growing. I will extend Dr. Avery's shift by 60 minutes to 17:00 to clear the overdue reports, and extend Riley's shift by 60 minutes to 17:15 to help with the imaging backlog. This is a targeted investment to reduce the backlog and potentially reduce lost demand.
* Relevant span: «I will not extend shifts or add teleradiology yet.»
* Proposed actions: `[{"date": "2026-11-06", "employee": "E07", "minutes": 60, "type": "extend_shift"}, {"date": "2026-11-06", "employee": "E03", "minutes": 60, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00006", "message": "Shift extended: Dr. Avery today until 17:00 (+1h approved overtime \u00b7 \u2248$231 at the salary's hourly rate, if worked) \u00b7 on site for contrast supervision (reading blocks unchanged) 16:00\u201317:00.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00007", "message": "Shift extended: Riley today until 17:15 (+1h approved overtime \u00b7 \u2248$54) \u00b7 staffs XR 16:15\u201317:15 \u00b7 no slot newly bookable (already staffed, or no template slots then).", "ok": true}, "index": 1, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": ["revised_in_text", "unresolved_target"], "kind": "negated", "params": {}, "targets": [], "type": "extend_shift"}`
* Staff in the observation: `{"E03": "Riley", "E07": "Dr. Avery"}`
* Rule output: **unresolved** (negated_action). A statement not to act, matched by a proposed action, but the statement is revised_in_text, unresolved_target.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`

### R098 — EXPLICIT_OPPOSITE_ACTION · rule output: unresolved

* Identifier: `de1-d3bd783cf869b657`
* Recording: `docs/benchmark/v07c-dev/stage-a/reporting-backlog-credit-qwen36-35b-a3b-s41602-decisions.jsonl`, decision point 0 (2026-10-13T07:00:00); model Qwen3.6-35B-A3B-oQ4e-mtp
* Reason (whole): Dr. Blake, Sam, and Alex already work until or past the scheduled close time (20:00), so extending their shifts today adds no capacity. Dr. Avery (E07) works until 16:00, Riley (E03) until 16:15, Casey (E05) until 16:30. To address the reading backlog and potential imaging demand, I will extend Dr. Avery's shift to cover more reading time, and extend Riley and Casey to cover more imaging slots. I will not extend staff who are already at or past close.
* Relevant span: «I will not extend staff who are already at or past close.»
* Proposed actions: `[{"date": "2026-10-13", "employee": "E07", "minutes": 120, "type": "extend_shift"}, {"date": "2026-10-13", "employee": "E03", "minutes": 120, "type": "extend_shift"}, {"date": "2026-10-13", "employee": "E05", "minutes": 120, "type": "extend_shift"}]`
* Validation and execution: `[{"apply": {"code": "accepted", "id": "X00001", "message": "Shift extended: Dr. Avery today until 18:00 (+2h approved overtime \u00b7 \u2248$462 at the salary's hourly rate, if worked) \u00b7 on site for contrast supervision (reading blocks unchanged) 16:00\u201318:00.", "ok": true}, "index": 0, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00002", "message": "Shift extended: Riley today until 18:15 (+2h approved overtime \u00b7 \u2248$72) \u00b7 staffs XR 16:15\u201318:15 \u00b7 no slot newly bookable (already staffed, or no template slots then).", "ok": true}, "index": 1, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}, {"apply": {"code": "accepted", "id": "X00003", "message": "Shift extended: Casey today until 18:30 (+2h approved overtime \u00b7 \u2248$90) \u00b7 staffs MR 16:30\u201318:30 \u00b7 no slot newly bookable (already staffed, or no template slots then).", "ok": true}, "index": 2, "state": "accepted", "submitted": true, "validate": {"code": "accepted", "ok": true}}]`
* What the rule checked: `{"flags": ["already", "unresolved_target"], "kind": "negated", "params": {}, "targets": [], "type": "extend_shift"}`
* Staff in the observation: `{"E03": "Riley", "E05": "Casey", "E07": "Dr. Avery"}`
* Rule output: **unresolved** (negated_action). A statement not to act, matched by a proposed action, but the statement is unresolved_target.
* Evidence completeness: `{"apply": true, "observation": true, "prompt": "verified", "proposed": true, "reason": true, "refusals": true, "validation": true}`
