# Financial decision evidence: case studies

Generated from financial-evidence.json; every number is the engine's. For each decision point the record shows only what was observable then (the decision point's observation and the engine's answers to the submitted batch); what happened later is in a separate table and is not attributed to any decision. No agent reasoning is inferred, and no decision is labelled good or bad from its outcome.

## 1. V0.7c stage B, staffing shortage, Qwen (90 days): hires, terminations and credit

The run V0.7c described as drawing on credit (docs/benchmark/v07c-dev/README.md §5). Its eventual losses are already known; this reconstruction keeps them out of each decision point and shows what was observable when each hire, termination, allocation change and extension was decided.

* Recording: `docs/benchmark/v07c-dev/stage-b/staffing-shortage-credit-qwen36-35b-a3b-s41701-decisions.jsonl` (llm-agent 2), policy qwen36-35b-a3b, model Qwen3.6-35B-A3B-oQ4e-mtp.
* Run: bench-staffing-shortage-ot-credit, seed 41701, 90 days; dreamteam-dev 3 (V0.7): radiologist overtime paid, $50,000 credit line at 10%, the V0.7b funding rule.
* Engine replay: 389 requests, every original answer identical; 90 of 90 decision points matched.

* Earlier attempts not recorded: 14; their refused actions' codes: after_close 5, already_terminating 1, batch_rejected 8, lead_too_short 3, not_on_shift 3, past_date 1 (an attempt can be lost to a parse or protocol error, which has no refusal code).

### Decision points with a submission

Each point shows what was observable when the decision was made, then the engine's answer. Nothing later is shown here.

**Point 0 (2026-10-13T07:00:00).** Known: cash $150,000.00; funding headroom $108,091.62; credit drawn $0.00; reserved $0.00; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4; unread studies 0; orders lost so far 0.

* Submitted hire (hire-xrct-premium-1, 2026-10-14) → lead_too_short; affordability **not_tested** (refused for another reason before or instead of the funding test: lead_too_short); refused.

**Point 1 (2026-10-14T07:00:00).** Known: cash $157,201.74; funding headroom $105,908.39; credit drawn $0.00; reserved $0.00; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4; unread studies 10; orders lost so far 9.

* Submitted extend_shift (E03, 2026-10-14) → accepted; affordability **funded_without_credit** (the reservation the engine added is within the headroom without undrawn credit); applied; commitment $67.50.
  * Executed extend_shift (E03, 2026-10-14): reversibility **not_reversible**; effective 2026-10-14.
  * Direct change: {"covers": "staffs XR", "date": "2026-10-14", "employee": "E03", "minutes": 60, "resource": "scheduled duty", "role": "technologist"}.
* Submitted hire (hire-xrct-std-002, 2026-11-11) → accepted; affordability **funded_without_credit** (commitment within the headroom without undrawn credit); applied; commitment $10,871.38; Recruiting fee · XR+CT technologist: $3,000.00 (recognized now, paid 2026-11-01); Contract guarantee per full week · XR+CT technologist: $1,900.00 from 2026-11-11.
  * Executed hire (hire-xrct-std-002, 2026-11-11): reversibility **reversible_with_retained_obligation**; effective 2026-11-11, startDate 2026-11-11, recruitingFeeCents $3,000.00, earliestExitCents $7,871.38, earliestExitWhen last day 2026-11-25.
  * Direct change: {"contractMin": 2400, "offer": "XR+CT technologist", "productiveDate": "2026-11-18", "resource": "technologist capacity", "rosterMinA": 2400, "rosterMinB": 2400, "startDate": "2026-11-11", "tier": "standard", "trainingDays": 7}.

**Point 2 (2026-10-15T07:00:00).** Known: cash $162,681.39; funding headroom $91,810.97; credit drawn $0.00; reserved $7,871.38; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting); unread studies 6; orders lost so far 25.

* Submitted extend_shift (E03, 2026-10-15) → accepted; affordability **funded_without_credit** (the reservation the engine added is within the headroom without undrawn credit); applied; commitment $67.50.
  * Executed extend_shift (E03, 2026-10-15): reversibility **not_reversible**; effective 2026-10-15.
  * Direct change: {"covers": "staffs XR", "date": "2026-10-15", "employee": "E03", "minutes": 60, "resource": "scheduled duty", "role": "technologist"}.

**Point 3 (2026-10-16T07:00:00).** Known: cash $117,543.88; funding headroom $89,977.39; credit drawn $0.00; reserved $7,871.38; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting); unread studies 14; orders lost so far 37.

* Submitted extend_shift (E03, 2026-10-16) → accepted; affordability **funded_without_credit** (the reservation the engine added is within the headroom without undrawn credit); applied; commitment $67.50.
  * Executed extend_shift (E03, 2026-10-16): reversibility **not_reversible**; effective 2026-10-16.
  * Direct change: {"covers": "staffs XR", "date": "2026-10-16", "employee": "E03", "minutes": 60, "resource": "scheduled duty", "role": "technologist"}.
* Submitted set_reading_allocation (set, E08, 2026-10-30) → accepted; affordability **funded_without_credit** (commitment within the headroom without undrawn credit); applied; commitment $24,657.53; Salary allocation change · Dr. Blake (E08): $24,657.53 per 30 days from 2026-10-30 until changed (14 days' notice).
  * Executed set_reading_allocation (set, E08, 2026-10-30): reversibility **withdrawable**; effective 2026-10-30, withdrawBy 2026-10-16.
  * Direct change: {"afterBP": 10000, "afterSalary30Cents": 4931506, "afterSiteMinA": 2550, "afterSiteMinB": 2790, "backlogAtDecision": {"studies": 14, "urgent": 0}, "beforeBP": 5000, "beforeSalary30Cents": 2465753, "beforeSiteMinA": 1300, "beforeSiteMinB": 1440, "effective": "2026-10-30", "employee": "E08", "op": "set", "resource": "reporting allocation"}.

**Point 6 (2026-10-19T07:00:00).** Known: cash $123,982.94; funding headroom $52,113.79; credit drawn $0.00; reserved $32,528.91; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting); unread studies 13; orders lost so far 54.

* Submitted extend_shift (E07, 2026-10-19), set_worklist_policy → accepted; affordability **funded_without_credit** (the reservation the engine added is within the headroom without undrawn credit); applied; commitment $576.91.
  * Executed extend_shift (E07, 2026-10-19): reversibility **not_reversible**; effective 2026-10-19.
  * Executed set_worklist_policy: reversibility **reversible_no_recorded_cost**; no dated terms (in effect at once).
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-10-19", "employee": "E07", "minutes": 120, "resource": "scheduled duty", "role": "radiologist"}.
  * Direct change: {"after": "oldest_first", "before": "urgent_then_fifo", "resource": "reporting order"}.

**Point 10 (2026-10-23T07:00:00).** Known: cash $164,197.36; funding headroom $57,245.16; credit drawn $0.00; reserved $32,528.91; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting); unread studies 8; orders lost so far 119.

* Submitted extend_shift (E07, 2026-10-23) → accepted; affordability **funded_without_credit** (the reservation the engine added is within the headroom without undrawn credit); applied; commitment $576.91.
  * Executed extend_shift (E07, 2026-10-23): reversibility **not_reversible**; effective 2026-10-23.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-10-23", "employee": "E07", "minutes": 120, "resource": "scheduled duty", "role": "radiologist"}.

**Point 15 (2026-10-28T07:00:00).** Known: cash $200,644.54; funding headroom $55,570.14; credit drawn $0.00; reserved $32,528.91; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting); unread studies 10; orders lost so far 181.

* Submitted set_worklist_policy, extend_shift (E07, 2026-10-29) → accepted, unchanged; affordability **funded_without_credit** (the reservation the engine added is within the headroom without undrawn credit); applied; commitment $576.91.
  * Executed extend_shift (E07, 2026-10-29): reversibility **not_reversible**; effective 2026-10-29.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-10-29", "employee": "E07", "minutes": 120, "resource": "scheduled duty", "role": "radiologist"}.

**Point 17 (2026-10-30T07:00:00).** Known: cash $162,119.29; funding headroom $52,137.40; credit drawn $0.00; reserved $31,707.00; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting); unread studies 10; orders lost so far 225.

* Submitted set_worklist_policy, extend_shift (E07, 2026-10-30) → accepted, unchanged; affordability **funded_without_credit** (the reservation the engine added is within the headroom without undrawn credit); applied; commitment $576.91.
  * Executed extend_shift (E07, 2026-10-30): reversibility **not_reversible**; effective 2026-10-30.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-10-30", "employee": "E07", "minutes": 120, "resource": "scheduled duty", "role": "radiologist"}.

**Point 19 (2026-11-01T07:00:00).** Known: cash $39,178.19; funding headroom $43,395.90; credit drawn $0.00; reserved $30,063.16; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting); unread studies 12; orders lost so far 241.

* Submitted set_worklist_policy, extend_shift (E07, 2026-11-02) → accepted, unchanged; affordability **funded_with_credit** (the reservation the engine added exceeds the headroom without undrawn credit); applied; commitment $576.91.
  * Executed extend_shift (E07, 2026-11-02): reversibility **not_reversible**; effective 2026-11-02.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-11-02", "employee": "E07", "minutes": 120, "resource": "scheduled duty", "role": "radiologist"}.

**Point 20 (2026-11-02T07:00:00).** Known: cash $39,178.19; funding headroom $37,735.43; credit drawn $0.00; reserved $29,818.15; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting); unread studies 12; orders lost so far 241.

* Submitted set_worklist_policy, extend_shift (E07, 2026-11-03) → accepted, unchanged; affordability **funded_with_credit** (the reservation the engine added exceeds the headroom without undrawn credit); applied; commitment $576.91.
  * Executed extend_shift (E07, 2026-11-03): reversibility **not_reversible**; effective 2026-11-03.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-11-03", "employee": "E07", "minutes": 120, "resource": "scheduled duty", "role": "radiologist"}.

**Point 21 (2026-11-03T07:00:00).** Known: cash $60,919.33; funding headroom $50,949.19; credit drawn $0.00; reserved $28,996.24; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting); unread studies 0; orders lost so far 253.

* Submitted extend_shift (E07, 2026-11-04), extend_shift (E07, 2026-11-05), extend_shift (E07, 2026-11-06) → accepted; affordability **funded_with_credit** (the reservation the engine added exceeds the headroom without undrawn credit); applied; commitment $1,730.73.
  * Executed extend_shift (E07, 2026-11-04): reversibility **not_reversible**; effective 2026-11-04.
  * Executed extend_shift (E07, 2026-11-05): reversibility **not_reversible**; effective 2026-11-05.
  * Executed extend_shift (E07, 2026-11-06): reversibility **not_reversible**; effective 2026-11-06.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-11-04", "employee": "E07", "minutes": 120, "resource": "scheduled duty", "role": "radiologist"}.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-11-05", "employee": "E07", "minutes": 120, "resource": "scheduled duty", "role": "radiologist"}.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-11-06", "employee": "E07", "minutes": 120, "resource": "scheduled duty", "role": "radiologist"}.
* Submitted hire (hire-mr-prem-003, 2026-11-24) → accepted; affordability **funded_with_credit** (commitment exceeds the headroom without undrawn credit); applied; commitment $13,150.00; Recruiting fee · MR technologist: $3,000.00 (recognized now, paid 2026-12-01); Contract guarantee per full week · MR technologist: $2,450.00 from 2026-11-24.
  * Executed hire (hire-mr-prem-003, 2026-11-24): reversibility **reversible_with_retained_obligation**; effective 2026-11-24, startDate 2026-11-24, recruitingFeeCents $3,000.00, earliestExitCents $10,150.00, earliestExitWhen last day 2026-12-08.
  * Direct change: {"contractMin": 2400, "offer": "MR technologist", "productiveDate": "2026-12-01", "resource": "technologist capacity", "rosterMinA": 2400, "rosterMinB": 2400, "startDate": "2026-11-24", "tier": "premium", "trainingDays": 7}.

**Point 27 (2026-11-09T07:00:00).** Known: cash $90,903.78; funding headroom $18,555.01; credit drawn $0.00; reserved $33,637.82; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+2 recruiting); unread studies 16; orders lost so far 329.

* Submitted extend_shift (E07, 2026-11-09), extend_shift (E05, 2026-11-09), set_worklist_policy → accepted; affordability **funded_with_credit** (the reservation the engine added exceeds the headroom without undrawn credit); applied; commitment $661.29.
  * Executed extend_shift (E07, 2026-11-09): reversibility **not_reversible**; effective 2026-11-09.
  * Executed extend_shift (E05, 2026-11-09): reversibility **not_reversible**; effective 2026-11-09.
  * Executed set_worklist_policy: reversibility **reversible_no_recorded_cost**; no dated terms (in effect at once).
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-11-09", "employee": "E07", "minutes": 120, "resource": "scheduled duty", "role": "radiologist"}.
  * Direct change: {"covers": "staffs MR", "date": "2026-11-09", "employee": "E05", "minutes": 60, "resource": "scheduled duty", "role": "technologist"}.
  * Direct change: {"after": "urgent_then_fifo", "before": "oldest_first", "resource": "reporting order"}.

**Point 45 (2026-11-27T07:00:00).** Known: cash $100,116.89; funding headroom $22,226.39; credit drawn $0.00; reserved $8,265.26; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 onboarding); unread studies 1; orders lost so far 533.

* Submitted terminate (H00002, 2026-12-11), set_reading_allocation (set, E08, 2026-12-11) → accepted; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00; Severance · H00002 (MR technologist): $4,900.00 (recognized now, paid 2026-12-25); Salary allocation change · Dr. Blake (E08): −$24,657.53 per 30 days from 2026-12-11 until changed (14 days' notice).
  * Executed terminate (H00002, 2026-12-11): reversibility **not_reversible**; effective 2026-12-12, lastDay 2026-12-11, endDate 2026-12-12, noticeDays 14, severanceCents $4,900.00, severancePayday 2026-12-25.
  * Executed set_reading_allocation (set, E08, 2026-12-11): reversibility **withdrawable**; effective 2026-12-11, withdrawBy 2026-11-27.
  * Direct change: {"displacedBookingsInBatch": 0, "employee": "H00002", "endDate": "2026-12-12", "lastDay": "2026-12-11", "resource": "technologist capacity", "role": "technologist", "voidedExtensions": 0}.
  * Direct change: {"afterBP": 5000, "afterSalary30Cents": 2465753, "afterSiteMinA": 1300, "afterSiteMinB": 1440, "backlogAtDecision": {"studies": 1, "urgent": 0}, "beforeBP": 10000, "beforeSalary30Cents": 4931506, "beforeSiteMinA": 2550, "beforeSiteMinB": 2790, "effective": "2026-12-11", "employee": "E08", "op": "set", "resource": "reporting allocation"}.

**Point 49 (2026-12-01T07:00:00).** Known: cash $0.00; funding headroom $12,728.40; credit drawn $4,405.66; reserved $4,736.67; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 notice); unread studies 0; orders lost so far 567.

* Submitted terminate (H00001, 2026-12-15) → accepted; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00; Severance · H00001 (XR+CT technologist): $3,800.00 (recognized now, paid 2026-12-25).
  * Executed terminate (H00001, 2026-12-15): reversibility **not_reversible**; effective 2026-12-16, lastDay 2026-12-15, endDate 2026-12-16, noticeDays 14, severanceCents $3,800.00, severancePayday 2026-12-25.
  * Direct change: {"displacedBookingsInBatch": 47, "employee": "H00001", "endDate": "2026-12-16", "lastDay": "2026-12-15", "resource": "technologist capacity", "role": "technologist", "voidedExtensions": 0}.

**Point 50 (2026-12-02T07:00:00).** Known: cash $4,024.54; funding headroom $7,703.24; credit drawn $0.00; reserved $4,060.00; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+2 notice); unread studies 3; orders lost so far 574.

* Submitted terminate (H00001, 2026-12-15) → unchanged; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00.

**Point 51 (2026-12-03T07:00:00).** Known: cash $12,144.97; funding headroom $5,868.40; credit drawn $0.00; reserved $3,383.34; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+2 notice); unread studies 3; orders lost so far 578.

* Submitted terminate (H00002, 2026-12-11) → unchanged; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00.

**Point 52 (2026-12-04T07:00:00).** Known: cash $18,538.93; funding headroom $2,528.65; credit drawn $0.00; reserved $2,706.67; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+2 notice); unread studies 2; orders lost so far 587.

* Submitted terminate (E01, 2026-12-18) → accepted; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00; Severance · Jordan (E01): $2,100.00 (recognized now, paid 2026-12-25).
  * Executed terminate (E01, 2026-12-18): reversibility **not_reversible**; effective 2026-12-19, lastDay 2026-12-18, endDate 2026-12-19, noticeDays 14, severanceCents $2,100.00, severancePayday 2026-12-25.
  * Direct change: {"displacedBookingsInBatch": 0, "employee": "E01", "endDate": "2026-12-19", "lastDay": "2026-12-18", "resource": "front-desk capacity", "role": "front_desk", "voidedExtensions": 0}.

**Point 53 (2026-12-05T07:00:00).** Known: cash $25,635.95; funding headroom −$2,084.14; credit drawn $0.00; reserved $2,030.00; staff: front_desk 0 (+1 notice), ops_manager 1, radiologist 2, technologist 4 (+2 notice); unread studies 3; orders lost so far 601.

* Submitted terminate (E05, 2026-12-19) → accepted; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00; Severance · Casey (E05): $4,500.00 (recognized now, paid 2026-12-25).
  * Executed terminate (E05, 2026-12-19): reversibility **not_reversible**; effective 2026-12-20, lastDay 2026-12-19, endDate 2026-12-20, noticeDays 14, severanceCents $4,500.00, severancePayday 2026-12-25.
  * Direct change: {"displacedBookingsInBatch": 141, "employee": "E05", "endDate": "2026-12-20", "lastDay": "2026-12-19", "resource": "technologist capacity", "role": "technologist", "voidedExtensions": 0}.

**Point 54 (2026-12-06T07:00:00).** Known: cash $25,635.95; funding headroom −$12,911.89; credit drawn $0.00; reserved $1,353.34; staff: front_desk 0 (+1 notice), ops_manager 1, radiologist 2, technologist 3 (+3 notice); unread studies 14; orders lost so far 609.

* Submitted terminate (E03, 2026-12-20) → accepted; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00; Severance · Riley (E03): $3,600.00 (recognized now, paid 2026-12-25).
  * Executed terminate (E03, 2026-12-20): reversibility **not_reversible**; effective 2026-12-21, lastDay 2026-12-20, endDate 2026-12-21, noticeDays 14, severanceCents $3,600.00, severancePayday 2026-12-25.
  * Direct change: {"displacedBookingsInBatch": 260, "employee": "E03", "endDate": "2026-12-21", "lastDay": "2026-12-20", "resource": "technologist capacity", "role": "technologist", "voidedExtensions": 0}.

**Point 55 (2026-12-07T07:00:00).** Known: cash $25,635.95; funding headroom −$21,645.52; credit drawn $0.00; reserved $676.67; staff: front_desk 0 (+1 notice), ops_manager 1, radiologist 2, technologist 2 (+4 notice); unread studies 14; orders lost so far 609.

* Submitted terminate (E04, 2026-12-21), terminate (E06, 2026-12-21) → accepted; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00; Severance · Sam (E04): $3,950.00 (recognized now, paid 2026-12-25); Severance · Morgan (E06): $4,345.00 (recognized now, paid 2026-12-25).
  * Executed terminate (E04, 2026-12-21): reversibility **not_reversible**; effective 2026-12-22, lastDay 2026-12-21, endDate 2026-12-22, noticeDays 14, severanceCents $3,950.00, severancePayday 2026-12-25.
  * Executed terminate (E06, 2026-12-21): reversibility **not_reversible**; effective 2026-12-22, lastDay 2026-12-21, endDate 2026-12-22, noticeDays 14, severanceCents $4,345.00, severancePayday 2026-12-25.
  * Direct change: {"displacedBookingsInBatch": 543, "employee": "E04", "endDate": "2026-12-22", "lastDay": "2026-12-21", "resource": "technologist capacity", "role": "technologist", "voidedExtensions": 0}.
  * Direct change: {"displacedBookingsInBatch": 543, "employee": "E06", "endDate": "2026-12-22", "lastDay": "2026-12-21", "resource": "technologist capacity", "role": "technologist", "voidedExtensions": 0}.

**Point 59 (2026-12-11T07:00:00).** Known: cash $2,697.62; funding headroom −$22,575.55; credit drawn $0.00; reserved $0.00; staff: front_desk 0 (+1 notice), ops_manager 1, radiologist 2, technologist 0 (+6 notice); unread studies 1; orders lost so far 797.

* Submitted terminate (H00002, 2026-12-11), terminate (H00001, 2026-12-15), terminate (E01, 2026-12-18), terminate (E03, 2026-12-20), terminate (E04, 2026-12-21), terminate (E05, 2026-12-19), terminate (E06, 2026-12-21) → unchanged; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00.

**Point 86 (2027-01-07T07:00:00).** Known: cash $33,001.04; funding headroom $30,624.28; credit drawn $0.00; reserved $0.00; staff: ops_manager 1, radiologist 2; unread studies 0; orders lost so far 2,309.

* Submitted hire (hire-xrct-std-001, 2027-02-04), hire (hire-xrct-std-002, 2027-02-04), hire (hire-us-prem-001, 2027-01-14), set_reading_allocation (set, E08, 2027-01-22) → batch_rejected, request_conflict; affordability **not_tested** (refused for another reason before or instead of the funding test: request_conflict); refused.

**Point 87 (2027-01-08T07:00:00).** Known: cash $11,401.66; funding headroom $33,492.87; credit drawn $0.00; reserved $0.00; staff: ops_manager 1, radiologist 2; unread studies 0; orders lost so far 2,384.

* Submitted hire (hire-xrct-std-001, 2027-02-04), hire (hire-xrct-std-002, 2027-02-04) → batch_rejected, lead_too_short; affordability **not_tested** (refused for another reason before or instead of the funding test: lead_too_short); refused.

**Point 88 (2027-01-09T07:00:00).** Known: cash $20,048.90; funding headroom $36,719.31; credit drawn $0.00; reserved $0.00; staff: ops_manager 1, radiologist 2; unread studies 0; orders lost so far 2,461.

* Submitted hire (hire-xrct-std-003, 2027-02-05), hire (hire-xrct-std-004, 2027-02-05) → batch_rejected, lead_too_short; affordability **not_tested** (refused for another reason before or instead of the funding test: lead_too_short); refused.

**Point 89 (2027-01-10T07:00:00).** Known: cash $20,048.90; funding headroom $31,730.91; credit drawn $0.00; reserved $0.00; staff: ops_manager 1, radiologist 2; unread studies 0; orders lost so far 2,479.

* Submitted hire (hire-xrct-std-003, 2027-02-07), hire (hire-xrct-std-004, 2027-02-07) → accepted; affordability **funded_with_credit** (commitment exceeds the headroom without undrawn credit); applied; commitment $21,742.76; Recruiting fee · XR+CT technologist: $3,000.00 (recognized now, paid 2027-02-01); Recruiting fee · XR+CT technologist: $3,000.00 (recognized now, paid 2027-02-01); Contract guarantee per full week · XR+CT technologist: $1,900.00 from 2027-02-07; Contract guarantee per full week · XR+CT technologist: $1,900.00 from 2027-02-07.
  * Executed hire (hire-xrct-std-003, 2027-02-07): reversibility **reversible_with_retained_obligation**; effective 2027-02-07, startDate 2027-02-07, recruitingFeeCents $3,000.00, earliestExitCents $7,871.38, earliestExitWhen last day 2027-02-21, amountsAre batch lines; the batch has several hires.
  * Executed hire (hire-xrct-std-004, 2027-02-07): reversibility **reversible_with_retained_obligation**; effective 2027-02-07, startDate 2027-02-07, recruitingFeeCents $3,000.00, earliestExitCents $7,871.38, earliestExitWhen last day 2027-02-21, amountsAre batch lines; the batch has several hires.
  * Direct change: {"contractMin": 2400, "offer": "XR+CT technologist", "opensSlots": [{"count": 832, "modality": "XR"}, {"count": 624, "modality": "CT"}], "productiveDate": "2027-02-20", "resource": "technologist capacity", "rosterMinA": 2400, "rosterMinB": 0, "startDate": "2027-02-07", "tier": "standard", "trainingDays": 13}.
  * Direct change: {"contractMin": 2400, "offer": "XR+CT technologist", "opensSlots": [{"count": 832, "modality": "XR"}, {"count": 624, "modality": "CT"}], "productiveDate": "2027-02-20", "resource": "technologist capacity", "rosterMinA": 2400, "rosterMinB": 0, "startDate": "2027-02-07", "tier": "standard", "trainingDays": 13}.

### Observed later (B): not attributed to any decision

| Point | Time | Cash | Headroom | Credit drawn | Reserved | Staff (active, + notice/onboarding) | Unread | Orders lost so far |
|---|---|---|---|---|---|---|---|---|
| 0 | 2026-10-13T07:00:00 | $150,000.00 | $108,091.62 | $0.00 | $0.00 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 0 | 0 |
| 7 | 2026-10-20T07:00:00 | $141,590.23 | $61,447.63 | $0.00 | $32,528.91 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting) | 0 | 70 |
| 14 | 2026-10-27T07:00:00 | $192,108.97 | $55,754.45 | $0.00 | $32,528.91 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting) | 0 | 164 |
| 21 | 2026-11-03T07:00:00 | $60,919.33 | $50,949.19 | $0.00 | $28,996.24 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 recruiting) | 0 | 253 |
| 28 | 2026-11-10T07:00:00 | $112,375.14 | $30,992.83 | $0.00 | $32,815.90 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+2 recruiting) | 0 | 342 |
| 35 | 2026-11-17T07:00:00 | $95,480.54 | $29,472.52 | $0.00 | $23,389.17 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 onboarding, 1 recruiting) | 0 | 421 |
| 42 | 2026-11-24T07:00:00 | $142,604.57 | $23,086.31 | $0.00 | $13,285.78 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 onboarding) | 0 | 487 |
| 49 | 2026-12-01T07:00:00 | $0.00 | $12,728.40 | $4,405.66 | $4,736.67 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 notice) | 0 | 567 |
| 56 | 2026-12-08T07:00:00 | $50,405.52 | −$14,959.83 | $0.00 | $0.00 | front_desk 0 (+1 notice), ops_manager 1, radiologist 2, technologist 0 (+6 notice) | 0 | 660 |
| 63 | 2026-12-15T07:00:00 | $35,997.78 | −$18,006.85 | $0.00 | $0.00 | front_desk 0 (+1 notice), ops_manager 1, radiologist 2, technologist 0 (+5 notice) | 0 | 926 |
| 70 | 2026-12-22T07:00:00 | $94,865.33 | −$13,539.77 | $0.00 | $0.00 | ops_manager 1, radiologist 2 | 0 | 1,292 |
| 77 | 2026-12-29T07:00:00 | $78,661.29 | $5,164.86 | $0.00 | $0.00 | ops_manager 1, radiologist 2 | 0 | 1,712 |
| 84 | 2027-01-05T07:00:00 | $17,608.72 | $25,978.38 | $0.00 | $0.00 | ops_manager 1, radiologist 2 | 0 | 2,122 |
| 89 | 2027-01-10T07:00:00 | $20,048.90 | $31,730.91 | $0.00 | $0.00 | ops_manager 1, radiologist 2 | 0 | 2,479 |

### Paired with no action (C)

Same entry, seed (41701), 90 days and economics, from `docs/benchmark/v07c-dev/stage-b/contract.json`: profit −$181,588.63, lowest cash −$37,228.50, orders lost +1,450, exams completed −867, overtime cost −$2,218.38, most credit drawn $17,200.28 (no action: $0.00). These are differences between two whole runs; no part of them is allocated to a single decision.

Limits: llm-agent 2 recorded only the final reply at each point, so earlier refused attempts are counted but not shown. The later losses are observed run-level differences (C) and are not attributed to the hires or terminations; doing so would need runs that differ only in one decision, which were not performed.

## 2. Contrast: V0.7c stage A, staffing shortage, Gemma (30 days): limited intervention

The same scenario and economics with one executed action. It is a contrast in what was done, not a paired comparison with case 1: the horizon (30 days) and the seed (41602) differ.

* Recording: `docs/benchmark/v07c-dev/stage-a/staffing-shortage-credit-gemma4-26b-a4b-s41602-decisions.jsonl` (llm-agent 2), policy gemma4-26b-a4b, model gemma-4-26B-A4B-it-oQ4e-mtp.
* Run: bench-staffing-shortage-ot-credit, seed 41602, 30 days; dreamteam-dev 3 (V0.7): radiologist overtime paid, $50,000 credit line at 10%, the V0.7b funding rule.
* Engine replay: 126 requests, every original answer identical; 30 of 30 decision points matched.

* Earlier attempts not recorded: 6; their refused actions' codes: invalid_roster 1, lead_too_short 2 (an attempt can be lost to a parse or protocol error, which has no refusal code).

### Decision points with a submission

Each point shows what was observable when the decision was made, then the engine's answer. Nothing later is shown here.

**Point 7 (2026-10-20T07:00:00).** Known: cash $147,095.16; funding headroom $102,167.96; credit drawn $0.00; reserved $0.00; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4; unread studies 0; orders lost so far 88.

* Submitted contract_teleradiology (start, telerad-mri-fix, 2026-11-03) → accepted; affordability **funded_without_credit** (commitment within the headroom without undrawn credit); applied; commitment $3,093.33; Teleradiology retainer, open-ended: $3,100.00 per month from 2026-11-03 until a cancellation fixes the end.
  * Executed contract_teleradiology (start, telerad-mri-fix, 2026-11-03): reversibility **reversible_with_retained_obligation**; effective 2026-11-03, minimumTermEnd 2026-12-03, monthlyRetainerCents $3,100.00, guaranteedCents $3,093.33, guaranteedThrough 2026-12-03.
  * Direct change: {"effective": "2026-11-03", "op": "start", "plan": "standard-v1", "resource": "teleradiology coverage", "scope": "all", "windows": ["weeknights", "saturday", "sunday"]}.

**Point 25 (2026-11-07T07:00:00).** Known: cash $94,213.79; funding headroom $78,051.29; credit drawn $0.00; reserved $2,577.78; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4; unread studies 0; orders lost so far 350.

* Submitted hire (hire-mr-tech-01, 2026-11-28) → invalid_roster; affordability **not_tested** (refused for another reason before or instead of the funding test: invalid_roster); refused.

**Point 26 (2026-11-08T07:00:00).** Known: cash $94,213.79; funding headroom $71,428.63; credit drawn $0.00; reserved $2,474.67; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4; unread studies 0; orders lost so far 356.

* Submitted hire (hire-mr-tech-01, 2026-11-28) → lead_too_short; affordability **not_tested** (refused for another reason before or instead of the funding test: lead_too_short); refused.

**Point 27 (2026-11-09T07:00:00).** Known: cash $94,213.79; funding headroom $66,344.80; credit drawn $0.00; reserved $2,371.56; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4; unread studies 0; orders lost so far 356.

* Submitted hire (hire-mr-tech-02, 2026-11-30) → invalid_roster; affordability **not_tested** (refused for another reason before or instead of the funding test: invalid_roster); refused.

### Observed later (B): not attributed to any decision

| Point | Time | Cash | Headroom | Credit drawn | Reserved | Staff (active, + notice/onboarding) | Unread | Orders lost so far |
|---|---|---|---|---|---|---|---|---|
| 0 | 2026-10-13T07:00:00 | $150,000.00 | $106,432.13 | $0.00 | $0.00 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 0 | 0 |
| 2 | 2026-10-15T07:00:00 | $165,402.65 | $103,227.25 | $0.00 | $0.00 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 10 | 42 |
| 4 | 2026-10-17T07:00:00 | $125,596.65 | $99,221.65 | $0.00 | $0.00 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 7 | 66 |
| 6 | 2026-10-19T07:00:00 | $125,596.65 | $88,491.85 | $0.00 | $0.00 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 11 | 68 |
| 8 | 2026-10-21T07:00:00 | $153,057.96 | $96,238.52 | $0.00 | $3,093.33 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 11 | 102 |
| 10 | 2026-10-23T07:00:00 | $167,699.04 | $92,943.79 | $0.00 | $3,093.33 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 12 | 146 |
| 12 | 2026-10-25T07:00:00 | $175,452.94 | $85,220.38 | $0.00 | $3,093.33 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 13 | 169 |
| 14 | 2026-10-27T07:00:00 | $198,327.10 | $94,590.78 | $0.00 | $3,093.33 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 0 | 185 |
| 16 | 2026-10-29T07:00:00 | $210,372.62 | $89,089.36 | $0.00 | $3,093.33 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 5 | 212 |
| 18 | 2026-10-31T07:00:00 | $170,926.54 | $83,876.48 | $0.00 | $3,093.33 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 8 | 261 |
| 20 | 2026-11-02T07:00:00 | $43,751.06 | $72,991.24 | $0.00 | $3,093.33 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 11 | 266 |
| 22 | 2026-11-04T07:00:00 | $72,820.56 | $84,402.54 | $0.00 | $2,887.11 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 0 | 296 |
| 24 | 2026-11-06T07:00:00 | $88,581.23 | $81,407.00 | $0.00 | $2,680.89 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 0 | 329 |
| 26 | 2026-11-08T07:00:00 | $94,213.79 | $71,428.63 | $0.00 | $2,474.67 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 0 | 356 |
| 28 | 2026-11-10T07:00:00 | $115,680.05 | $79,276.02 | $0.00 | $2,268.45 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 0 | 369 |
| 29 | 2026-11-11T07:00:00 | $121,682.03 | $75,959.81 | $0.00 | $2,165.34 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 | 0 | 385 |

### Paired with no action (C)

Same entry, seed (41602), 30 days and economics, from `docs/benchmark/v07c-dev/stage-a/contract.json`: profit −$2,043.31, lowest cash $0.00, orders lost 0, exams completed 0, overtime cost $0.00, most credit drawn $0.00 (no action: $0.00). These are differences between two whole runs; no part of them is allocated to a single decision.

Limits: a different seed and horizon from case 1, so no difference between the two cases is a measured effect of intervening less.

## 3. V0.8c, low liquidity, Qwen (30 days): affordability refusals

The run with the most funding refusals (8 of V0.8c's 10). llm-agent 3 records every attempt, so each refused batch is shown with the engine's commitment and headroom at that moment.

* Recording: `docs/benchmark/v08c-dev/eval/financial-pressure-lowliquidity-qwen36-35b-a3b-s41901-decisions.jsonl` (llm-agent 3), policy qwen36-35b-a3b, model Qwen3.6-35B-A3B-oQ4e-mtp.
* Run: bench-financial-pressure-ot-lowliquidity, seed 41901, 30 days; dreamteam-dev 4 (V0.8c): radiologist overtime paid, $50,000 credit line at 10%, the V0.7b funding rule; the low-liquidity entry opens with $80,000 cash.
* Engine replay: 162 requests, every original answer identical; 30 of 30 decision points matched.

### Decision points with a submission

Each point shows what was observable when the decision was made, then the engine's answer. Nothing later is shown here.

**Point 0 (2026-10-13T07:00:00).** Known: cash $80,000.00; funding headroom $35,217.86; credit drawn $0.00; reserved $0.00; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5; unread studies 3; orders lost so far 0.

* Submitted extend_shift (E07, 2026-10-13), extend_shift (E05, 2026-10-13) → accepted; affordability **funded_with_credit** (the reservation the engine added exceeds the headroom without undrawn credit); applied; commitment $372.83.
  * Executed extend_shift (E07, 2026-10-13): reversibility **not_reversible**; effective 2026-10-13.
  * Executed extend_shift (E05, 2026-10-13): reversibility **not_reversible**; effective 2026-10-13.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-10-13", "employee": "E07", "minutes": 60, "resource": "scheduled duty", "role": "radiologist"}.
  * Direct change: {"covers": "staffs MR", "date": "2026-10-13", "employee": "E05", "minutes": 60, "resource": "scheduled duty", "role": "technologist"}.

**Point 1 (2026-10-14T07:00:00).** Known: cash $87,544.29; funding headroom $32,756.55; credit drawn $0.00; reserved $0.00; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5; unread studies 8; orders lost so far 7.

* Submitted extend_shift (E07, 2026-10-14) → accepted; affordability **funded_with_credit** (the reservation the engine added exceeds the headroom without undrawn credit); applied; commitment $288.45.
  * Executed extend_shift (E07, 2026-10-14): reversibility **not_reversible**; effective 2026-10-14.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-10-14", "employee": "E07", "minutes": 60, "resource": "scheduled duty", "role": "radiologist"}.

**Point 2 (2026-10-15T07:00:00).** Known: cash $92,849.39; funding headroom $28,281.34; credit drawn $0.00; reserved $0.00; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5; unread studies 8; orders lost so far 13.

* Submitted extend_shift (E07, 2026-10-15) → accepted; affordability **funded_with_credit** (the reservation the engine added exceeds the headroom without undrawn credit); applied; commitment $288.45.
  * Executed extend_shift (E07, 2026-10-15): reversibility **not_reversible**; effective 2026-10-15.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-10-15", "employee": "E07", "minutes": 60, "resource": "scheduled duty", "role": "radiologist"}.

**Point 3 (2026-10-16T07:00:00).** Known: cash $43,846.23; funding headroom $26,749.23; credit drawn $0.00; reserved $0.00; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5; unread studies 5; orders lost so far 19.

* Submitted extend_shift (E07, 2026-10-16) → accepted; affordability **funded_with_credit** (the reservation the engine added exceeds the headroom without undrawn credit); applied; commitment $288.45.
  * Executed extend_shift (E07, 2026-10-16): reversibility **not_reversible**; effective 2026-10-16.
  * Direct change: {"covers": "on site for contrast supervision (reading blocks unchanged)", "date": "2026-10-16", "employee": "E07", "minutes": 60, "resource": "scheduled duty", "role": "radiologist"}.

**Point 4 (2026-10-17T07:00:00).** Known: cash $53,812.48; funding headroom $26,920.03; credit drawn $0.00; reserved $0.00; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5; unread studies 5; orders lost so far 30.

* Submitted hire (hire-ct-prem-20261017, 2026-10-24), set_reading_allocation (set, E07, 2026-10-31) → batch_rejected, lead_too_short; affordability **not_tested** (refused for another reason before or instead of the funding test: lead_too_short); refused.
* Submitted hire (hire-ct-prem-20261017-v2, 2026-11-07), set_reading_allocation (set, E07, 2026-10-31) → accepted; affordability **funded_with_credit** (commitment exceeds the headroom without undrawn credit); applied; commitment $12,114.13; Recruiting fee · CT technologist: $3,000.00 (recognized now, paid 2026-11-01); Contract guarantee per full week · CT technologist: $2,200.00 from 2026-11-07; Salary allocation change · Dr. Avery (E07): −$14,794.53 per 30 days from 2026-10-31 until changed (14 days' notice).
  * Executed hire (hire-ct-prem-20261017-v2, 2026-11-07): reversibility **reversible_with_retained_obligation**; effective 2026-11-07, startDate 2026-11-07, recruitingFeeCents $3,000.00, earliestExitCents $9,114.13, earliestExitWhen last day 2026-11-21.
  * Executed set_reading_allocation (set, E07, 2026-10-31): reversibility **withdrawable**; effective 2026-10-31, withdrawBy 2026-10-17.
  * Direct change: {"contractMin": 2400, "offer": "CT technologist", "opensSlots": [{"count": 45, "modality": "CT"}], "productiveDate": "2026-11-24", "resource": "technologist capacity", "rosterMinA": 2250, "rosterMinB": 0, "startDate": "2026-11-07", "tier": "premium", "trainingDays": 17}.
  * Direct change: {"afterBP": 3000, "afterSalary30Cents": 1479451, "afterSiteMinA": 890, "afterSiteMinB": 800, "backlogAtDecision": {"studies": 5, "urgent": 0}, "beforeBP": 6000, "beforeSalary30Cents": 2958904, "beforeSiteMinA": 1710, "beforeSiteMinB": 1550, "effective": "2026-10-31", "employee": "E07", "op": "set", "resource": "reporting allocation"}.

**Point 5 (2026-10-18T07:00:00).** Known: cash $53,812.48; funding headroom $9,006.85; credit drawn $0.00; reserved $9,114.13; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting); unread studies 15; orders lost so far 30.

* Submitted extend_shift (E07, 2026-10-18) → not_on_shift; affordability **not_tested** (refused for another reason before or instead of the funding test: not_on_shift); refused.

**Point 6 (2026-10-19T07:00:00).** Known: cash $53,812.48; funding headroom $4,018.45; credit drawn $0.00; reserved $9,114.13; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting); unread studies 15; orders lost so far 30.

* Submitted extend_shift (E07, 2026-10-19), extend_shift (E05, 2026-10-19) → accepted; affordability **evidence_unavailable** (funded, but never applied, so the engine reserved nothing that measures its commitment); not_applied.
* Submitted set_reading_allocation (set, E07, 2026-10-19) → past_date; affordability **not_tested** (refused for another reason before or instead of the funding test: past_date); refused.
* Submitted extend_shift (E07, 2026-10-19) → accepted; affordability **evidence_unavailable** (funded, but never applied, so the engine reserved nothing that measures its commitment); not_applied.
* Submitted set_reading_allocation (set, E07, 2026-11-02) → commitment_too_short; affordability **not_tested** (refused for another reason before or instead of the funding test: commitment_too_short); refused.

**Point 9 (2026-10-22T07:00:00).** Known: cash $93,567.18; funding headroom $16,724.63; credit drawn $0.00; reserved $9,114.13; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting); unread studies 6; orders lost so far 43.

* Submitted hire (hire-mr-std-20261022, 2026-11-20), set_reading_allocation (set, E08, 2026-11-05) → batch_rejected, insufficient_funds; affordability **refused_insufficient_funds** (the engine refused the batch with insufficient_funds); refused; commitment $36,978.91.

**Point 10 (2026-10-23T07:00:00).** Known: cash $101,164.15; funding headroom $15,430.34; credit drawn $0.00; reserved $9,114.13; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting); unread studies 6; orders lost so far 45.

* Submitted hire (hire-us-std-20261023, 2026-10-24), set_reading_allocation (set, E08, 2026-11-06) → batch_rejected, lead_too_short; affordability **not_tested** (refused for another reason before or instead of the funding test: lead_too_short); refused.
* Submitted hire (hire-us-prem-20261023, 2026-11-13), set_reading_allocation (set, E08, 2026-11-06) → batch_rejected, insufficient_funds; affordability **refused_insufficient_funds** (the engine refused the batch with insufficient_funds); refused; commitment $37,600.28.

**Point 11 (2026-10-24T07:00:00).** Known: cash $109,604.97; funding headroom $14,047.14; credit drawn $0.00; reserved $9,114.13; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting); unread studies 17; orders lost so far 53.

* Submitted set_reading_allocation (set, E08, 2026-11-07) → insufficient_funds; affordability **refused_insufficient_funds** (the engine refused the batch with insufficient_funds); refused; commitment $24,657.53.
* Submitted set_reading_allocation (set, E07, 2026-11-07) → unchanged; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00.

**Point 13 (2026-10-26T07:00:00).** Known: cash $109,604.97; funding headroom $2,830.95; credit drawn $0.00; reserved $9,114.13; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting); unread studies 27; orders lost so far 55.

* Submitted set_reading_allocation (set, E08, 2026-11-09) → insufficient_funds; affordability **refused_insufficient_funds** (the engine refused the batch with insufficient_funds); refused; commitment $24,657.53.

**Point 17 (2026-10-30T07:00:00).** Known: cash $88,149.91; funding headroom $1,472.11; credit drawn $0.00; reserved $9,114.13; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting); unread studies 11; orders lost so far 84.

* Submitted set_reading_allocation (set, E07, 2026-11-13) → unchanged; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00.

**Point 23 (2026-11-05T07:00:00).** Known: cash $4,042.30; funding headroom $2,820.34; credit drawn $0.00; reserved $9,114.13; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting); unread studies 35; orders lost so far 120.

* Submitted set_reading_allocation (set, E08, 2026-11-19), terminate (E11, 2026-11-19) → accepted; affordability **no_new_commitment** (the engine's funding test measures no commitment); applied; commitment $0.00; Severance · Alex (E11): $4,500.00 (recognized now, paid 2026-11-27); Salary allocation change · Dr. Blake (E08): −$9,863.02 per 30 days from 2026-11-19 until changed (14 days' notice).
  * Executed set_reading_allocation (set, E08, 2026-11-19): reversibility **withdrawable**; effective 2026-11-19, withdrawBy 2026-11-05.
  * Executed terminate (E11, 2026-11-19): reversibility **not_reversible**; effective 2026-11-20, lastDay 2026-11-19, endDate 2026-11-20, noticeDays 14, severanceCents $4,500.00, severancePayday 2026-11-27.
  * Direct change: {"afterBP": 3000, "afterSalary30Cents": 1479451, "afterSiteMinA": 800, "afterSiteMinB": 890, "backlogAtDecision": {"studies": 35, "urgent": 0}, "beforeBP": 5000, "beforeSalary30Cents": 2465753, "beforeSiteMinA": 1300, "beforeSiteMinB": 1440, "effective": "2026-11-19", "employee": "E08", "op": "set", "resource": "reporting allocation"}.
  * Direct change: {"displacedBookingsInBatch": 9, "employee": "E11", "endDate": "2026-11-20", "lastDay": "2026-11-19", "resource": "technologist capacity", "role": "technologist", "voidedExtensions": 0}.

**Point 24 (2026-11-06T07:00:00).** Known: cash $12,079.33; funding headroom −$2,479.56; credit drawn $0.00; reserved $9,114.13; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 notice, 1 recruiting); unread studies 55; orders lost so far 130.

* Submitted terminate (E03, 2026-11-20), set_reading_allocation (set, E07, 2026-11-20), contract_teleradiology (start, tr-urgent-backlog-20261106, 2026-11-20) → batch_rejected, invalid_schema; affordability **not_tested** (refused for another reason before or instead of the funding test: invalid_schema); refused.
* Submitted terminate (E03, 2026-11-20), set_reading_allocation (set, E07, 2026-11-20), contract_teleradiology (start, tr-urgent-backlog-20261106, 2026-11-20) → batch_rejected, insufficient_funds; affordability **refused_insufficient_funds** (the engine refused the batch with insufficient_funds); refused; commitment $3,036.67.

**Point 25 (2026-11-07T07:00:00).** Known: cash $18,733.17; funding headroom −$4,359.09; credit drawn $0.00; reserved $8,506.53; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 notice, 1 onboarding); unread studies 66; orders lost so far 144.

* Submitted extend_shift (E04, 2026-11-07) → after_close; affordability **not_tested** (refused for another reason before or instead of the funding test: after_close); refused.
* Submitted hire (hire-ct-prem-20261107-v1, 2026-11-14), set_reading_allocation (set, E08, 2026-11-21) → batch_rejected, lead_too_short; affordability **not_tested** (refused for another reason before or instead of the funding test: lead_too_short); refused.
* Submitted hire (hire-ct-prem-20261107-v2, 2026-11-28), set_reading_allocation (set, E08, 2026-11-21) → batch_rejected, insufficient_funds; affordability **refused_insufficient_funds** (the engine refused the batch with insufficient_funds); refused; commitment $12,114.13.

**Point 26 (2026-11-08T07:00:00).** Known: cash $18,733.17; funding headroom −$9,863.66; credit drawn $0.00; reserved $7,898.92; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 notice, 1 onboarding); unread studies 75; orders lost so far 145.

* Submitted contract_teleradiology (start, tr-urgent-backlog-20261108, 2026-11-22) → insufficient_funds; affordability **refused_insufficient_funds** (the engine refused the batch with insufficient_funds); refused; commitment $3,029.98.

**Point 28 (2026-11-10T07:00:00).** Known: cash $38,881.01; funding headroom −$2,452.16; credit drawn $0.00; reserved $6,683.70; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 notice, 1 onboarding); unread studies 84; orders lost so far 151.

* Submitted extend_shift (E04, 2026-11-10) → after_close; affordability **not_tested** (refused for another reason before or instead of the funding test: after_close); refused.
* Submitted contract_teleradiology (start, tr-urgent-20261110, 2026-11-24) → insufficient_funds; affordability **refused_insufficient_funds** (the engine refused the batch with insufficient_funds); refused; commitment $1,462.90.

**Point 29 (2026-11-11T07:00:00).** Known: cash $46,984.64; funding headroom −$3,491.44; credit drawn $0.00; reserved $6,076.09; staff: front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 notice, 1 onboarding); unread studies 103; orders lost so far 154.

* Submitted extend_shift (E04, 2026-11-11) → after_close; affordability **not_tested** (refused for another reason before or instead of the funding test: after_close); refused.
* Submitted hire (hire-ct-prem-20261111-v1, 2026-11-18), set_reading_allocation (set, E08, 2026-11-19) → batch_rejected, lead_too_short; affordability **not_tested** (refused for another reason before or instead of the funding test: lead_too_short); refused.

### Observed later (B): not attributed to any decision

| Point | Time | Cash | Headroom | Credit drawn | Reserved | Staff (active, + notice/onboarding) | Unread | Orders lost so far |
|---|---|---|---|---|---|---|---|---|
| 0 | 2026-10-13T07:00:00 | $80,000.00 | $35,217.86 | $0.00 | $0.00 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 | 3 | 0 |
| 2 | 2026-10-15T07:00:00 | $92,849.39 | $28,281.34 | $0.00 | $0.00 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 | 8 | 13 |
| 4 | 2026-10-17T07:00:00 | $53,812.48 | $26,920.03 | $0.00 | $0.00 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 | 5 | 30 |
| 6 | 2026-10-19T07:00:00 | $53,812.48 | $4,018.45 | $0.00 | $9,114.13 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting) | 15 | 30 |
| 8 | 2026-10-21T07:00:00 | $85,579.98 | $17,819.10 | $0.00 | $9,114.13 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting) | 13 | 36 |
| 10 | 2026-10-23T07:00:00 | $101,164.15 | $15,430.34 | $0.00 | $9,114.13 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting) | 6 | 45 |
| 12 | 2026-10-25T07:00:00 | $109,604.97 | $7,819.36 | $0.00 | $9,114.13 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting) | 27 | 55 |
| 14 | 2026-10-27T07:00:00 | $125,849.12 | $10,434.82 | $0.00 | $9,114.13 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting) | 3 | 61 |
| 16 | 2026-10-29T07:00:00 | $137,687.81 | $3,737.41 | $0.00 | $9,114.13 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting) | 9 | 75 |
| 18 | 2026-10-31T07:00:00 | $96,521.99 | $918.90 | $0.00 | $9,114.13 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting) | 5 | 98 |
| 20 | 2026-11-02T07:00:00 | $0.00 | −$8,999.37 | $33,667.61 | $9,114.13 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting) | 16 | 98 |
| 22 | 2026-11-04T07:00:00 | $0.00 | $2,990.00 | $5,072.94 | $9,114.13 | front_desk 1, ops_manager 1, radiologist 2, technologist 5 (+1 recruiting) | 21 | 110 |
| 24 | 2026-11-06T07:00:00 | $12,079.33 | −$2,479.56 | $0.00 | $9,114.13 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 notice, 1 recruiting) | 55 | 130 |
| 26 | 2026-11-08T07:00:00 | $18,733.17 | −$9,863.66 | $0.00 | $7,898.92 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 notice, 1 onboarding) | 75 | 145 |
| 28 | 2026-11-10T07:00:00 | $38,881.01 | −$2,452.16 | $0.00 | $6,683.70 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 notice, 1 onboarding) | 84 | 151 |
| 29 | 2026-11-11T07:00:00 | $46,984.64 | −$3,491.44 | $0.00 | $6,076.09 | front_desk 1, ops_manager 1, radiologist 2, technologist 4 (+1 notice, 1 onboarding) | 103 | 154 |

### Paired with no action (C)

Same entry, seed (41901), 30 days and economics, from `docs/benchmark/v08c-dev/eval/contract.json`: profit −$20,967.02, lowest cash $0.00, orders lost 0, exams completed 0, overtime cost +$968.04, most credit drawn $33,667.61 (no action: $29,429.43). These are differences between two whole runs; no part of them is allocated to a single decision.

Limits: one seed of one scenario. What the agent did after each refusal is described by refusal-response v2 (V0.8c) and is not re-analysed here.
