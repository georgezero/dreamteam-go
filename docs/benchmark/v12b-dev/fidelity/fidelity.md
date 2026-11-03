# V0.12b Stage B driver: fidelity on the real engine (scripted replies)

113 of 113 cases faithful; every apply check equal: yes.

Each case: the driver's draw against the agent's own run (its Harness on its own engine session, its ModelClient payloads, its Evidence), then the draw replayed without the model.

| Point | Arm | Script | Calls | Outcome | Class / preview | Inputs | Model requests | Record | Engine | Batches | Replay |
|---|---|---|---|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | A | keep | 1 | kept | proceeded | preview | same | same | same | same | same |
| ss-gemma-pv-s42001-p14 | A | abandon | 1 | abandoned | abandoned | preview | same | same | same | same | same |
| ss-gemma-pv-s42001-p14 | A | garbage → keep | 2 | invalid | failed | preview → retry | same | same | same | same | same |
| ss-gemma-pv-s42001-p14 | A | timeout → timeout | 2 | invalid | failed | preview → resend | same | same | same | same | same |
| ss-gemma-pv-s42001-p14 | B | ack | 1 | kept | A_proceeded | acknowledgment | same | same | same | same | same |
| ss-gemma-pv-s42001-p14 | B | ack_wrong → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-gemma-pv-s42001-p14 | B | ack_wrong → ack_wrong | 2 | withheld | D_withheld | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-gemma-pv-s42001-p14 | B | abandon | 1 | abandoned | C_abandoned | acknowledgment | same | same | same | same | same |
| ss-gemma-pv-s42001-p14 | B | ack_stale → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-gemma-pv-s42001-p14 | B | timeout → ack | 2 | kept | A_proceeded | acknowledgment → resend | same | same | same | same | same |
| ss-gemma-pv-s42001-p14 | B | ack_missing → garbage | 2 | withheld | D_withheld | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-gemma-pv-s42001-p14 | B | ack_field → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| fl-gemma-s41901-p28 | A | keep | 1 | refused | not_shown | validation | same | same | same | same | same |
| fl-gemma-s41901-p28 | A | abandon | 1 | abandoned | not_shown | validation | same | same | same | same | same |
| fl-gemma-s41901-p28 | A | garbage | 1 | invalid | not_shown | validation | same | same | same | same | same |
| fl-gemma-s41901-p28 | A | http500 → abandon | 1 | invalid | not_shown | validation | same | same | same | same | same |
| fl-gemma-s41901-p28 | B | keep | 1 | refused | E_not_required | validation | same | same | same | same | same |
| fl-gemma-s41901-p28 | B | abandon | 1 | abandoned | E_not_required | validation | same | same | same | same | same |
| fl-gemma-s41901-p28 | B | garbage | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| fl-gemma-s41901-p28 | B | http500 → abandon | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| fl-gemma-s41901-p26 | A | keep | 1 | refused | not_shown | validation | same | same | same | same | same |
| fl-gemma-s41901-p26 | A | abandon | 1 | abandoned | not_shown | validation | same | same | same | same | same |
| fl-gemma-s41901-p26 | A | garbage | 1 | invalid | not_shown | validation | same | same | same | same | same |
| fl-gemma-s41901-p26 | A | http500 → abandon | 1 | invalid | not_shown | validation | same | same | same | same | same |
| fl-gemma-s41901-p26 | B | keep | 1 | refused | E_not_required | validation | same | same | same | same | same |
| fl-gemma-s41901-p26 | B | abandon | 1 | abandoned | E_not_required | validation | same | same | same | same | same |
| fl-gemma-s41901-p26 | B | garbage | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| fl-gemma-s41901-p26 | B | http500 → abandon | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| fl-qwen-s41902-p24 | A | keep | 1 | kept | proceeded | preview | same | same | same | same | same |
| fl-qwen-s41902-p24 | A | abandon | 1 | abandoned | abandoned | preview | same | same | same | same | same |
| fl-qwen-s41902-p24 | A | garbage → keep | 2 | invalid | failed | preview → retry | same | same | same | same | same |
| fl-qwen-s41902-p24 | A | timeout → timeout | 2 | invalid | failed | preview → resend | same | same | same | same | same |
| fl-qwen-s41902-p24 | A | revise_drop | 1 | revised | revised | preview | same | same | same | same | same |
| fl-qwen-s41902-p24 | B | ack | 1 | kept | A_proceeded | acknowledgment | same | same | same | same | same |
| fl-qwen-s41902-p24 | B | ack_wrong → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| fl-qwen-s41902-p24 | B | ack_wrong → ack_wrong | 2 | withheld | D_withheld | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| fl-qwen-s41902-p24 | B | abandon | 1 | abandoned | C_abandoned | acknowledgment | same | same | same | same | same |
| fl-qwen-s41902-p24 | B | ack_stale → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| fl-qwen-s41902-p24 | B | timeout → ack | 2 | kept | A_proceeded | acknowledgment → resend | same | same | same | same | same |
| fl-qwen-s41902-p24 | B | ack_missing → garbage | 2 | withheld | D_withheld | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| fl-qwen-s41902-p24 | B | ack_field → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| fl-qwen-s41902-p24 | B | revise_drop → ack | 2 | revised | B_revised | acknowledgment → acknowledgment | same | same | same | same | same |
| ss-qwen-s42002-p7 | A | keep | 1 | kept | proceeded | preview | same | same | same | same | same |
| ss-qwen-s42002-p7 | A | abandon | 1 | abandoned | abandoned | preview | same | same | same | same | same |
| ss-qwen-s42002-p7 | A | garbage → keep | 2 | invalid | failed | preview → retry | same | same | same | same | same |
| ss-qwen-s42002-p7 | A | timeout → timeout | 2 | invalid | failed | preview → resend | same | same | same | same | same |
| ss-qwen-s42002-p7 | A | revise_drop | 1 | revised | revised | preview | same | same | same | same | same |
| ss-qwen-s42002-p7 | B | ack | 1 | kept | A_proceeded | acknowledgment | same | same | same | same | same |
| ss-qwen-s42002-p7 | B | ack_wrong → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-qwen-s42002-p7 | B | ack_wrong → ack_wrong | 2 | withheld | D_withheld | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-qwen-s42002-p7 | B | abandon | 1 | abandoned | C_abandoned | acknowledgment | same | same | same | same | same |
| ss-qwen-s42002-p7 | B | ack_stale → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-qwen-s42002-p7 | B | timeout → ack | 2 | kept | A_proceeded | acknowledgment → resend | same | same | same | same | same |
| ss-qwen-s42002-p7 | B | ack_missing → garbage | 2 | withheld | D_withheld | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-qwen-s42002-p7 | B | ack_field → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-qwen-s42002-p7 | B | revise_drop → ack | 1 | revised | B_revised | acknowledgment | same | same | same | same | same |
| ss-qwen-s42001-p20 | A | keep | 1 | kept | proceeded | preview | same | same | same | same | same |
| ss-qwen-s42001-p20 | A | abandon | 1 | abandoned | abandoned | preview | same | same | same | same | same |
| ss-qwen-s42001-p20 | A | garbage → keep | 2 | invalid | failed | preview → retry | same | same | same | same | same |
| ss-qwen-s42001-p20 | A | timeout → timeout | 2 | invalid | failed | preview → resend | same | same | same | same | same |
| ss-qwen-s42001-p20 | A | revise_drop | 1 | revised | revised | preview | same | same | same | same | same |
| ss-qwen-s42001-p20 | B | ack | 1 | kept | A_proceeded | acknowledgment | same | same | same | same | same |
| ss-qwen-s42001-p20 | B | ack_wrong → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-qwen-s42001-p20 | B | ack_wrong → ack_wrong | 2 | withheld | D_withheld | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-qwen-s42001-p20 | B | abandon | 1 | abandoned | C_abandoned | acknowledgment | same | same | same | same | same |
| ss-qwen-s42001-p20 | B | ack_stale → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-qwen-s42001-p20 | B | timeout → ack | 2 | kept | A_proceeded | acknowledgment → resend | same | same | same | same | same |
| ss-qwen-s42001-p20 | B | ack_missing → garbage | 2 | withheld | D_withheld | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-qwen-s42001-p20 | B | ack_field → ack | 2 | kept | A_proceeded | acknowledgment → acknowledgment_rejected | same | same | same | same | same |
| ss-qwen-s42001-p20 | B | revise_drop → ack | 2 | revised | B_revised | acknowledgment → acknowledgment | same | same | same | same | same |
| ss-qwen-pv-s42002-p6 | A | keep | 1 | kept | proceeded | preview | same | same | same | same | same |
| ss-qwen-pv-s42002-p6 | A | abandon | 1 | abandoned | abandoned | preview | same | same | same | same | same |
| ss-qwen-pv-s42002-p6 | A | garbage → keep | 2 | invalid | failed | preview → retry | same | same | same | same | same |
| ss-qwen-pv-s42002-p6 | A | timeout → timeout | 2 | invalid | failed | preview → resend | same | same | same | same | same |
| ss-qwen-pv-s42002-p6 | A | revise_drop | 1 | revised | revised | preview | same | same | same | same | same |
| ss-qwen-pv-s42002-p6 | B | — | 0 | kept | E_not_required | — | same | same | same | same | same |
| fl-qwen-s41902-p27 | A | keep | 1 | refused | not_shown | validation | same | same | same | same | same |
| fl-qwen-s41902-p27 | A | abandon | 1 | abandoned | not_shown | validation | same | same | same | same | same |
| fl-qwen-s41902-p27 | A | garbage | 1 | invalid | not_shown | validation | same | same | same | same | same |
| fl-qwen-s41902-p27 | A | http500 → abandon | 1 | invalid | not_shown | validation | same | same | same | same | same |
| fl-qwen-s41902-p27 | B | keep | 1 | refused | E_not_required | validation | same | same | same | same | same |
| fl-qwen-s41902-p27 | B | abandon | 1 | abandoned | E_not_required | validation | same | same | same | same | same |
| fl-qwen-s41902-p27 | B | garbage | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| fl-qwen-s41902-p27 | B | http500 → abandon | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| fl-qwen-s41901-p9 | A | keep | 1 | refused | not_shown | validation | same | same | same | same | same |
| fl-qwen-s41901-p9 | A | abandon | 1 | abandoned | not_shown | validation | same | same | same | same | same |
| fl-qwen-s41901-p9 | A | garbage | 1 | invalid | not_shown | validation | same | same | same | same | same |
| fl-qwen-s41901-p9 | A | http500 → abandon | 1 | invalid | not_shown | validation | same | same | same | same | same |
| fl-qwen-s41901-p9 | B | keep | 1 | refused | E_not_required | validation | same | same | same | same | same |
| fl-qwen-s41901-p9 | B | abandon | 1 | abandoned | E_not_required | validation | same | same | same | same | same |
| fl-qwen-s41901-p9 | B | garbage | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| fl-qwen-s41901-p9 | B | http500 → abandon | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| fp-qwen-s42001-p7 | A | keep | 1 | refused | not_shown | validation | same | same | same | same | same |
| fp-qwen-s42001-p7 | A | abandon | 1 | abandoned | not_shown | validation | same | same | same | same | same |
| fp-qwen-s42001-p7 | A | garbage | 1 | invalid | not_shown | validation | same | same | same | same | same |
| fp-qwen-s42001-p7 | A | http500 → abandon | 1 | invalid | not_shown | validation | same | same | same | same | same |
| fp-qwen-s42001-p7 | B | keep | 1 | refused | E_not_required | validation | same | same | same | same | same |
| fp-qwen-s42001-p7 | B | abandon | 1 | abandoned | E_not_required | validation | same | same | same | same | same |
| fp-qwen-s42001-p7 | B | garbage | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| fp-qwen-s42001-p7 | B | http500 → abandon | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| ss-qwen-s42001-p23 | A | keep | 1 | refused | not_shown | validation | same | same | same | same | same |
| ss-qwen-s42001-p23 | A | abandon | 1 | abandoned | not_shown | validation | same | same | same | same | same |
| ss-qwen-s42001-p23 | A | garbage | 1 | invalid | not_shown | validation | same | same | same | same | same |
| ss-qwen-s42001-p23 | A | http500 → abandon | 1 | invalid | not_shown | validation | same | same | same | same | same |
| ss-qwen-s42001-p23 | B | keep | 1 | refused | E_not_required | validation | same | same | same | same | same |
| ss-qwen-s42001-p23 | B | abandon | 1 | abandoned | E_not_required | validation | same | same | same | same | same |
| ss-qwen-s42001-p23 | B | garbage | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| ss-qwen-s42001-p23 | B | http500 → abandon | 1 | invalid | E_not_required | validation | same | same | same | same | same |
| fp-qwen-s41901-p25 | A | keep | 1 | kept | proceeded | preview | same | same | same | same | same |
| fp-qwen-s41901-p25 | A | abandon | 1 | abandoned | abandoned | preview | same | same | same | same | same |
| fp-qwen-s41901-p25 | A | garbage → keep | 2 | invalid | failed | preview → retry | same | same | same | same | same |
| fp-qwen-s41901-p25 | A | timeout → timeout | 2 | invalid | failed | preview → resend | same | same | same | same | same |
| fp-qwen-s41901-p25 | B | — | 0 | kept | E_not_required | — | same | same | same | same | same |

## Apply at the recorded state: the agent's harness against the analysis continuation

| Point | Arm | Decision | Applied (reference / continuation) | Digests |
|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | B | 7637de24a044 | [True] / [True] | equal |
| fl-qwen-s41902-p24 | B | 839c4b961fad | [True] / [True] | equal |
| ss-qwen-s42002-p7 | B | 1649fda3fc0d | [True] / [True] | equal |
| ss-qwen-s42001-p20 | B | 9b01e75a4331 | [True] / [True] | equal |
| ss-qwen-pv-s42002-p6 | B | 0b26cf33c344 | [True, True] / [True, True] | equal |
| fp-qwen-s41901-p25 | B | 66f3583a0083 | [True] / [True] | equal |
