# V0.11b decision distributions: A llm-agent 3, B validation only, C validation + preview

Each point and arm: every scheduled draw (Gemma 10, Qwen 20), each one whole workflow. Wilson 95% intervals; differences with Newcombe 95% intervals. Descriptive only. The primary comparison is C − B; the secondary B − A.

## Per point and arm

| Point | Proposal | Category | Arm | Kept | Revised | Abandoned | Refused | Invalid | Final accepted / no action / none | Model calls | Retries | Preview shown (kept / revised / abandoned after) | Credit-supported |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | accepted | retained_obligation | A | 10/10 | 0/10 | 0/10 | 0/10 | 0/10 | 10 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 0 |
| ss-gemma-pv-s42001-p14 | accepted | retained_obligation | B | 10/10 | 0/10 | 0/10 | 0/10 | 0/10 | 10 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 0 |
| ss-gemma-pv-s42001-p14 | accepted | retained_obligation | C | 10/10 | 0/10 | 0/10 | 0/10 | 0/10 | 10 / 0 / 0 | 10 | 0 | 10 (10 / 0 / 0) | 0 |
| fl-gemma-s41901-p28 | refused | other | A | 0/10 | 10/10 | 0/10 | 0/10 | 0/10 | 10 / 0 / 0 | 10 | 10 | 0 (0 / 0 / 0) | 10 |
| fl-gemma-s41901-p28 | refused | other | B | 0/10 | 10/10 | 0/10 | 0/10 | 0/10 | 10 / 0 / 0 | 10 | 10 | 0 (0 / 0 / 0) | 10 |
| fl-gemma-s41901-p28 | refused | other | C | 0/10 | 10/10 | 0/10 | 0/10 | 0/10 | 10 / 0 / 0 | 20 | 10 | 10 (10 / 0 / 0) | 10 |
| fl-gemma-s41901-p27 | refused | insufficient_funds | A | 0/10 | 0/10 | 10/10 | 0/10 | 0/10 | 0 / 10 / 0 | 10 | 10 | 0 (0 / 0 / 0) | 0 |
| fl-gemma-s41901-p27 | refused | insufficient_funds | B | 0/10 | 0/10 | 10/10 | 0/10 | 0/10 | 0 / 10 / 0 | 10 | 10 | 0 (0 / 0 / 0) | 0 |
| fl-gemma-s41901-p27 | refused | insufficient_funds | C | 0/10 | 0/10 | 10/10 | 0/10 | 0/10 | 0 / 10 / 0 | 10 | 10 | 0 (0 / 0 / 0) | 0 |
| fl-gemma-s41901-p26 | refused | other | A | 0/10 | 0/10 | 0/10 | 10/10 | 0/10 | 0 / 0 / 10 | 10 | 10 | 0 (0 / 0 / 0) | 0 |
| fl-gemma-s41901-p26 | refused | other | B | 0/10 | 0/10 | 0/10 | 10/10 | 0/10 | 0 / 0 / 10 | 10 | 10 | 0 (0 / 0 / 0) | 0 |
| fl-gemma-s41901-p26 | refused | other | C | 0/10 | 0/10 | 0/10 | 10/10 | 0/10 | 0 / 0 / 10 | 10 | 10 | 0 (0 / 0 / 0) | 0 |
| ss-qwen-pv-s42002-p6 | accepted | credit_supported | A | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 20 |
| ss-qwen-pv-s42002-p6 | accepted | credit_supported | B | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 20 |
| ss-qwen-pv-s42002-p6 | accepted | credit_supported | C | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 20 | 0 | 20 (20 / 0 / 0) | 20 |
| ss-qwen-s42002-p7 | accepted | retained_obligation | A | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 0 |
| ss-qwen-s42002-p7 | accepted | retained_obligation | B | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 0 |
| ss-qwen-s42002-p7 | accepted | retained_obligation | C | 0/20 | 16/20 | 4/20 | 0/20 | 0/20 | 16 / 4 / 0 | 24 | 4 | 20 (0 / 16 / 4) | 0 |
| fl-qwen-s41902-p24 | accepted | termination | A | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 0 |
| fl-qwen-s41902-p24 | accepted | termination | B | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 0 |
| fl-qwen-s41902-p24 | accepted | termination | C | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 20 | 0 | 20 (20 / 0 / 0) | 0 |
| fp-qwen-s41901-p26 | accepted | allocation | A | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 0 |
| fp-qwen-s41901-p26 | accepted | allocation | B | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 0 |
| fp-qwen-s41901-p26 | accepted | allocation | C | 0/20 | 20/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 20 | 0 | 20 (0 / 20 / 0) | 0 |
| fp-qwen-s41901-p25 | accepted | allocation | A | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 0 |
| fp-qwen-s41901-p25 | accepted | allocation | B | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 0 | 0 | 0 (0 / 0 / 0) | 0 |
| fp-qwen-s41901-p25 | accepted | allocation | C | 20/20 | 0/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 20 | 0 | 20 (20 / 0 / 0) | 0 |
| fl-qwen-s41901-p9 | refused | insufficient_funds | A | 0/20 | 0/20 | 9/20 | 10/20 | 1/20 | 0 / 9 / 11 | 20 | 20 | 0 (0 / 0 / 0) | 0 |
| fl-qwen-s41901-p9 | refused | insufficient_funds | B | 0/20 | 13/20 | 2/20 | 5/20 | 0/20 | 13 / 2 / 5 | 20 | 20 | 0 (0 / 0 / 0) | 0 |
| fl-qwen-s41901-p9 | refused | insufficient_funds | C | 0/20 | 12/20 | 3/20 | 5/20 | 0/20 | 12 / 3 / 5 | 34 | 20 | 14 (11 / 1 / 2) | 0 |
| fl-qwen-s41901-p11 | refused | insufficient_funds | A | 0/20 | 1/20 | 19/20 | 0/20 | 0/20 | 1 / 19 / 0 | 20 | 20 | 0 (0 / 0 / 0) | 0 |
| fl-qwen-s41901-p11 | refused | insufficient_funds | B | 0/20 | 0/20 | 20/20 | 0/20 | 0/20 | 0 / 20 / 0 | 20 | 20 | 0 (0 / 0 / 0) | 0 |
| fl-qwen-s41901-p11 | refused | insufficient_funds | C | 0/20 | 0/20 | 20/20 | 0/20 | 0/20 | 0 / 20 / 0 | 20 | 20 | 0 (0 / 0 / 0) | 0 |
| fp-qwen-s42001-p7 | refused | timing | A | 0/20 | 20/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 20 | 20 | 0 (0 / 0 / 0) | 0 |
| fp-qwen-s42001-p7 | refused | timing | B | 0/20 | 20/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 20 | 20 | 0 (0 / 0 / 0) | 0 |
| fp-qwen-s42001-p7 | refused | timing | C | 0/20 | 20/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 40 | 20 | 20 (20 / 0 / 0) | 0 |
| fl-qwen-s41902-p27 | refused | timing | A | 0/20 | 0/20 | 0/20 | 20/20 | 0/20 | 0 / 0 / 20 | 20 | 20 | 0 (0 / 0 / 0) | 0 |
| fl-qwen-s41902-p27 | refused | timing | B | 0/20 | 20/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 20 | 20 | 0 (0 / 0 / 0) | 0 |
| fl-qwen-s41902-p27 | refused | timing | C | 0/20 | 20/20 | 0/20 | 0/20 | 0/20 | 20 / 0 / 0 | 40 | 20 | 20 (20 / 0 / 0) | 0 |
| fp-qwen-s42001-p24 | refused | other | A | 0/20 | 0/20 | 0/20 | 20/20 | 0/20 | 0 / 0 / 20 | 20 | 20 | 0 (0 / 0 / 0) | 0 |
| fp-qwen-s42001-p24 | refused | other | B | 0/20 | 5/20 | 0/20 | 15/20 | 0/20 | 5 / 0 / 15 | 20 | 20 | 0 (0 / 0 / 0) | 0 |
| fp-qwen-s42001-p24 | refused | other | C | 0/20 | 4/20 | 0/20 | 16/20 | 0/20 | 4 / 0 / 16 | 24 | 20 | 4 (4 / 0 / 0) | 0 |

## C − B (primary)

| Point | Kept | Revised | Abandoned | Refused | Invalid | Final accepted | Credit-supported | Same action content |
|---|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| fl-gemma-s41901-p28 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| fl-gemma-s41901-p27 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| fl-gemma-s41901-p26 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| ss-qwen-pv-s42002-p6 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| ss-qwen-s42002-p7 | -100 pp [-100%, -77%] | +80 pp [53%, 92%] | +20 pp [-0%, 42%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | -20 pp [-42%, 0%] | +0 pp [-16%, 16%] | no |
| fl-qwen-s41902-p24 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| fp-qwen-s41901-p26 | -100 pp [-100%, -77%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no |
| fp-qwen-s41901-p25 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| fl-qwen-s41901-p9 | +0 pp [-16%, 16%] | -5 pp [-32%, 23%] | +5 pp [-17%, 27%] | +0 pp [-26%, 26%] | +0 pp [-16%, 16%] | -5 pp [-32%, 23%] | +0 pp [-16%, 16%] | no |
| fl-qwen-s41901-p11 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| fp-qwen-s42001-p7 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| fl-qwen-s41902-p27 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| fp-qwen-s42001-p24 | +0 pp [-16%, 16%] | -5 pp [-30%, 21%] | +0 pp [-16%, 16%] | +5 pp [-21%, 30%] | +0 pp [-16%, 16%] | -5 pp [-30%, 21%] | +0 pp [-16%, 16%] | no |

## B − A (secondary)

| Point | Kept | Revised | Abandoned | Refused | Invalid | Final accepted | Credit-supported | Same action content |
|---|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| fl-gemma-s41901-p28 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| fl-gemma-s41901-p27 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| fl-gemma-s41901-p26 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| ss-qwen-pv-s42002-p6 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| ss-qwen-s42002-p7 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| fl-qwen-s41902-p24 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| fp-qwen-s41901-p26 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| fp-qwen-s41901-p25 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| fl-qwen-s41901-p9 | +0 pp [-16%, 16%] | +65 pp [38%, 82%] | -35 pp [-57%, -7%] | -25 pp [-49%, 5%] | -5 pp [-24%, 12%] | +65 pp [38%, 82%] | +0 pp [-16%, 16%] | no |
| fl-qwen-s41901-p11 | +0 pp [-16%, 16%] | -5 pp [-24%, 12%] | +5 pp [-12%, 24%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | -5 pp [-24%, 12%] | +0 pp [-16%, 16%] | no |
| fp-qwen-s42001-p7 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no |
| fl-qwen-s41902-p27 | +0 pp [-16%, 16%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | -100 pp [-100%, -77%] | +0 pp [-16%, 16%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | no |
| fp-qwen-s42001-p24 | +0 pp [-16%, 16%] | +25 pp [4%, 47%] | +0 pp [-16%, 16%] | -25 pp [-47%, -4%] | +0 pp [-16%, 16%] | +25 pp [4%, 47%] | +0 pp [-16%, 16%] | no |

## C − A

| Point | Kept | Revised | Abandoned | Refused | Invalid | Final accepted | Credit-supported | Same action content |
|---|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| fl-gemma-s41901-p28 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| fl-gemma-s41901-p27 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| fl-gemma-s41901-p26 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes |
| ss-qwen-pv-s42002-p6 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| ss-qwen-s42002-p7 | -100 pp [-100%, -77%] | +80 pp [53%, 92%] | +20 pp [-0%, 42%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | -20 pp [-42%, 0%] | +0 pp [-16%, 16%] | no |
| fl-qwen-s41902-p24 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| fp-qwen-s41901-p26 | -100 pp [-100%, -77%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no |
| fp-qwen-s41901-p25 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes |
| fl-qwen-s41901-p9 | +0 pp [-16%, 16%] | +60 pp [33%, 78%] | -30 pp [-53%, -2%] | -25 pp [-49%, 5%] | -5 pp [-24%, 12%] | +60 pp [33%, 78%] | +0 pp [-16%, 16%] | no |
| fl-qwen-s41901-p11 | +0 pp [-16%, 16%] | -5 pp [-24%, 12%] | +5 pp [-12%, 24%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | -5 pp [-24%, 12%] | +0 pp [-16%, 16%] | no |
| fp-qwen-s42001-p7 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no |
| fl-qwen-s41902-p27 | +0 pp [-16%, 16%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | -100 pp [-100%, -77%] | +0 pp [-16%, 16%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | no |
| fp-qwen-s42001-p24 | +0 pp [-16%, 16%] | +20 pp [-0%, 42%] | +0 pp [-16%, 16%] | -20 pp [-42%, 0%] | +0 pp [-16%, 16%] | +20 pp [-0%, 42%] | +0 pp [-16%, 16%] | no |

## Pooled (after the per-point results)

Unweighted means of the per-point differences; no interval.

| Scope | Points | Comparison | Kept | Revised | Abandoned | Refused | Invalid | Points with the same action content |
|---|---|---|---|---|---|---|---|---|
| gemma/accepted | 1 | C-B | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 1 |
| gemma/accepted | 1 | B-A | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 1 |
| gemma/accepted | 1 | C-A | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 1 |
| gemma/refused | 3 | C-B | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 3 |
| gemma/refused | 3 | B-A | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 3 |
| gemma/refused | 3 | C-A | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 3 |
| gemma/all | 4 | C-B | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 4 |
| gemma/all | 4 | B-A | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 4 |
| gemma/all | 4 | C-A | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 4 |
| qwen/accepted | 5 | C-B | -40.0 pp | +36.0 pp | +4.0 pp | +0.0 pp | +0.0 pp | 3 |
| qwen/accepted | 5 | B-A | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 5 |
| qwen/accepted | 5 | C-A | -40.0 pp | +36.0 pp | +4.0 pp | +0.0 pp | +0.0 pp | 3 |
| qwen/refused | 5 | C-B | +0.0 pp | -2.0 pp | +1.0 pp | +1.0 pp | +0.0 pp | 3 |
| qwen/refused | 5 | B-A | +0.0 pp | +37.0 pp | -6.0 pp | -30.0 pp | -1.0 pp | 0 |
| qwen/refused | 5 | C-A | +0.0 pp | +35.0 pp | -5.0 pp | -29.0 pp | -1.0 pp | 0 |
| qwen/all | 10 | C-B | -20.0 pp | +17.0 pp | +2.5 pp | +0.5 pp | +0.0 pp | 6 |
| qwen/all | 10 | B-A | +0.0 pp | +18.5 pp | -3.0 pp | -15.0 pp | -0.5 pp | 5 |
| qwen/all | 10 | C-A | -20.0 pp | +35.5 pp | -0.5 pp | -14.5 pp | -0.5 pp | 3 |
| all/accepted | 6 | C-B | -33.3 pp | +30.0 pp | +3.3 pp | +0.0 pp | +0.0 pp | 4 |
| all/accepted | 6 | B-A | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 6 |
| all/accepted | 6 | C-A | -33.3 pp | +30.0 pp | +3.3 pp | +0.0 pp | +0.0 pp | 4 |
| all/refused | 8 | C-B | +0.0 pp | -1.2 pp | +0.6 pp | +0.6 pp | +0.0 pp | 6 |
| all/refused | 8 | B-A | +0.0 pp | +23.1 pp | -3.8 pp | -18.8 pp | -0.6 pp | 3 |
| all/refused | 8 | C-A | +0.0 pp | +21.9 pp | -3.1 pp | -18.1 pp | -0.6 pp | 3 |
| all/all | 14 | C-B | -14.3 pp | +12.1 pp | +1.8 pp | +0.4 pp | +0.0 pp | 10 |
| all/all | 14 | B-A | +0.0 pp | +13.2 pp | -2.1 pp | -10.7 pp | -0.4 pp | 9 |
| all/all | 14 | C-A | -14.3 pp | +25.4 pp | -0.4 pp | -10.4 pp | -0.4 pp | 7 |

## Financial evidence v1 of the distinct final decisions

| Point | Proposal | Draws A / B / C | Outcome | Commitment | Credit-supported | Retained obligations | Reversibility | Direct exposure | Displaced bookings |
|---|---|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| ss-gemma-pv-s42001-p14 | yes | 10 / 10 / 10 | accepted | $3,069.98 | no | 1 | reversible_with_retained_obligation 1 | teleradiology coverage 1 | 0 |
| fl-gemma-s41901-p28 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-gemma-s41901-p28 | yes | 0 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-gemma-s41901-p28 |  | 10 / 10 / 10 | accepted | $3,023.32 | yes | 1 | reversible_with_retained_obligation 1 | teleradiology coverage 1 | 0 |
| fl-gemma-s41901-p27 |  | 10 / 10 / 10 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-gemma-s41901-p27 | yes | 0 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-gemma-s41901-p26 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-gemma-s41901-p26 | yes | 0 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-pv-s42002-p6 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-pv-s42002-p6 | yes | 20 / 20 / 20 | accepted | $44,687.30 | yes | 0 | not_reversible 2, withdrawable 2 | reporting allocation 2, scheduled duty 2 | 0 |
| ss-qwen-s42002-p7 |  | 0 / 0 / 4 | no_action | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-s42002-p7 |  | 0 / 0 / 2 | accepted | $303.75 | no | 0 | not_reversible 2 | scheduled duty 2 | 0 |
| ss-qwen-s42002-p7 |  | 0 / 0 / 4 | accepted | $300.00 | no | 0 | not_reversible 2 | scheduled duty 2 | 0 |
| ss-qwen-s42002-p7 | yes | 20 / 20 / 0 | accepted | $3,228.33 | no | 1 | not_reversible 1, reversible_with_retained_obligation 1 | scheduled duty 1, teleradiology coverage 1 | 0 |
| ss-qwen-s42002-p7 |  | 0 / 0 / 10 | accepted | $135.00 | no | 0 | not_reversible 1 | scheduled duty 1 | 0 |
| fl-qwen-s41902-p24 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41902-p24 | yes | 20 / 20 / 20 | accepted | $0.00 | no | 0 | not_reversible 2 | front-desk capacity 1, technologist capacity 1 | 6 |
| fp-qwen-s41901-p26 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fp-qwen-s41901-p26 | yes | 20 / 20 / 0 | accepted | $9,863.02 | no | 0 | withdrawable 1 | reporting allocation 1 | 0 |
| fp-qwen-s41901-p26 |  | 0 / 0 / 20 | accepted | $9,863.02 | no | 0 | withdrawable 1 | reporting allocation 1 | 0 |
| fp-qwen-s41901-p25 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fp-qwen-s41901-p25 | yes | 20 / 20 / 20 | accepted | $9,863.02 | no | 0 | withdrawable 1 | reporting allocation 1 | 0 |
| fl-qwen-s41901-p9 |  | 9 / 2 / 3 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p9 | yes | 0 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p9 |  | 0 / 8 / 0 | accepted | $0.00 | no | 0 | not_reversible 1 | technologist capacity 1 | 8 |
| fl-qwen-s41901-p9 |  | 0 / 2 / 3 | accepted | $0.00 | no | 0 | not_reversible 1, withdrawable 1 | reporting allocation 1, technologist capacity 1 | 8 |
| fl-qwen-s41901-p9 |  | 0 / 3 / 9 | accepted | $0.00 | no | 0 | not_reversible 1 | technologist capacity 1 | 8 |
| fl-qwen-s41901-p11 |  | 19 / 20 / 20 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p11 |  | 1 / 0 / 0 | accepted | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p11 | yes | 0 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fp-qwen-s42001-p7 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fp-qwen-s42001-p7 |  | 1 / 0 / 0 | accepted | $11,700.00 | no | 1 | reversible_with_retained_obligation 1 | technologist capacity 1 | 0 |
| fp-qwen-s42001-p7 | yes | 0 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fp-qwen-s42001-p7 |  | 5 / 20 / 20 | accepted | $11,159.83 | no | 1 | not_reversible 1, reversible_with_retained_obligation 1 | scheduled duty 1, technologist capacity 1 | 0 |
| fp-qwen-s42001-p7 |  | 14 / 0 / 0 | accepted | $11,159.83 | no | 1 | not_reversible 1, reversible_with_retained_obligation 1 | scheduled duty 1, technologist capacity 1 | 0 |
| fl-qwen-s41902-p27 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41902-p27 | yes | 0 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41902-p27 |  | 0 / 20 / 20 | accepted | $0.00 | no | 0 | not_reversible 1 | technologist capacity 1 | 100 |
| fp-qwen-s42001-p24 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fp-qwen-s42001-p24 |  | 0 / 1 / 0 | accepted | $2,225.80 | no | 1 | not_reversible 1, reversible_with_retained_obligation 1 | teleradiology coverage 2 | 0 |
| fp-qwen-s42001-p24 |  | 0 / 4 / 4 | accepted | $2,225.80 | no | 1 | not_reversible 1, reversible_with_retained_obligation 1 | teleradiology coverage 2 | 0 |
| fp-qwen-s42001-p24 | yes | 0 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |

## Funding at the points

| Point | Funding headroom | Of it without credit | Proposal's stake |
|---|---|---|---|
| ss-gemma-pv-s42001-p14 | $103,953.60 | $53,953.60 | $3,069.98 |
| fl-gemma-s41901-p28 | $8,590.76 | −$41,409.24 | $0.00 |
| fl-gemma-s41901-p27 | −$2,429.59 | −$52,429.59 | $1,464.51 |
| fl-gemma-s41901-p26 | $2,654.01 | −$47,345.99 | $0.00 |
| ss-qwen-pv-s42002-p6 | $87,205.37 | $37,205.37 | $44,687.30 |
| ss-qwen-s42002-p7 | $100,306.40 | $50,306.40 | $3,228.33 |
| fl-qwen-s41902-p24 | $73.54 | −$49,926.46 | $0.00 |
| fp-qwen-s41901-p26 | $68,288.14 | $18,288.14 | $9,863.02 |
| fp-qwen-s41901-p25 | $83,618.38 | $33,618.38 | $9,863.02 |
| fl-qwen-s41901-p9 | $16,724.63 | −$33,275.37 | $36,978.91 |
| fl-qwen-s41901-p11 | $14,047.14 | −$35,952.86 | $24,657.53 |
| fp-qwen-s42001-p7 | $93,805.19 | $43,805.19 | $0.00 |
| fl-qwen-s41902-p27 | −$17,446.87 | −$67,446.87 | $0.00 |
| fp-qwen-s42001-p24 | $91,627.79 | $41,627.79 | $0.00 |
