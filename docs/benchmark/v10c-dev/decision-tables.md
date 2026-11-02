# V0.10c decision distributions: neutral, validation, preview

Each point and arm: every scheduled draw in the denominator (Gemma 10, Qwen 20 per arm). Wilson 95% intervals; differences with Newcombe 95% intervals. Descriptive only. The primary comparison is C − B.

## Per point and arm

| Point | Proposal | Arm | Message chars | Kept | Revised | Abandoned | Invalid | Final accepted / refused / no action | Distinct decisions |
|---|---|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | accepted | A neutral | 305 | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 10 / 0 / 0 | 1 |
| ss-gemma-pv-s42001-p14 | accepted | B validation | 1,087 | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 10 / 0 / 0 | 1 |
| ss-gemma-pv-s42001-p14 | accepted | C preview | 2,554 | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 10 / 0 / 0 | 1 |
| fl-gemma-s41901-p26 | refused | A neutral | 305 | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 0 / 10 / 0 | 1 |
| fl-gemma-s41901-p26 | refused | B validation | 794 | 0/10 [0%, 28%] | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 0 / 10 / 0 | 1 |
| fl-gemma-s41901-p26 | refused | C preview | 826 | 0/10 [0%, 28%] | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 0 / 10 / 0 | 1 |
| fl-gemma-s41901-p28 | refused | A neutral | 305 | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 0 / 10 / 0 | 1 |
| fl-gemma-s41901-p28 | refused | B validation | 777 | 0/10 [0%, 28%] | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 10 / 0 / 0 | 1 |
| fl-gemma-s41901-p28 | refused | C preview | 809 | 0/10 [0%, 28%] | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 10 / 0 / 0 | 1 |
| fl-gemma-s41901-p27 | refused | A neutral | 305 | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 0 / 10 / 0 | 1 |
| fl-gemma-s41901-p27 | refused | B validation | 1,050 | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0 / 0 / 10 | 1 |
| fl-gemma-s41901-p27 | refused | C preview | 1,289 | 0/10 [0%, 28%] | 0/10 [0%, 28%] | 10/10 [72%, 100%] | 0/10 [0%, 28%] | 0 / 0 / 10 | 1 |
| ss-qwen-pv-s42002-p6 | accepted | A neutral | 305 | 12/20 [39%, 78%] | 7/20 [18%, 57%] | 0/20 [0%, 16%] | 1/20 [1%, 24%] | 18 / 1 / 0 | 6 |
| ss-qwen-pv-s42002-p6 | accepted | B validation | 1,567 | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| ss-qwen-pv-s42002-p6 | accepted | C preview | 3,637 | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| ss-qwen-s42002-p7 | accepted | A neutral | 305 | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| ss-qwen-s42002-p7 | accepted | B validation | 1,191 | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| ss-qwen-s42002-p7 | accepted | C preview | 2,663 | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| fp-qwen-s41901-p26 | accepted | A neutral | 305 | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| fp-qwen-s41901-p26 | accepted | B validation | 1,265 | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| fp-qwen-s41901-p26 | accepted | C preview | 2,434 | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| ss-qwen-s42001-p20 | accepted | A neutral | 305 | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| ss-qwen-s42001-p20 | accepted | B validation | 7,886 | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| ss-qwen-s42001-p20 | accepted | C preview | 100,937 | 17/20 [64%, 95%] | 3/20 [5%, 36%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 2 |
| fl-qwen-s41901-p9 | refused | A neutral | 305 | 0/20 [0%, 16%] | 19/20 [76%, 99%] | 0/20 [0%, 16%] | 1/20 [1%, 24%] | 0 / 19 / 0 | 7 |
| fl-qwen-s41901-p9 | refused | B validation | 2,055 | 0/20 [0%, 16%] | 18/20 [70%, 97%] | 2/20 [3%, 30%] | 0/20 [0%, 16%] | 2 / 16 / 2 | 3 |
| fl-qwen-s41901-p9 | refused | C preview | 2,300 | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 4 |
| fl-qwen-s41901-p11 | refused | A neutral | 305 | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0 / 20 / 0 | 1 |
| fl-qwen-s41901-p11 | refused | B validation | 1,007 | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0 / 0 / 20 | 1 |
| fl-qwen-s41901-p11 | refused | C preview | 1,253 | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0 / 0 / 20 | 1 |
| ss-qwen-s42001-p23 | refused | A neutral | 305 | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 3 |
| ss-qwen-s42001-p23 | refused | B validation | 869 | 0/20 [0%, 16%] | 17/20 [64%, 95%] | 3/20 [5%, 36%] | 0/20 [0%, 16%] | 5 / 12 / 3 | 5 |
| ss-qwen-s42001-p23 | refused | C preview | 905 | 0/20 [0%, 16%] | 19/20 [76%, 99%] | 1/20 [1%, 24%] | 0/20 [0%, 16%] | 16 / 3 / 1 | 3 |
| fp-qwen-s42001-p27 | refused | A neutral | 305 | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0 / 20 / 0 | 1 |
| fp-qwen-s42001-p27 | refused | B validation | 936 | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0 / 20 / 0 | 1 |
| fp-qwen-s42001-p27 | refused | C preview | 968 | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0 / 20 / 0 | 1 |

## C preview − B validation (primary)

| Point | Kept | Revised | Abandoned | Invalid | Final accepted | Same action content | Discordant rounds |
|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes | 0/10 |
| fl-gemma-s41901-p26 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes | 0/10 |
| fl-gemma-s41901-p28 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | no | 0/10 |
| fl-gemma-s41901-p27 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes | 0/10 |
| ss-qwen-pv-s42002-p6 | +100 pp [77%, 100%] | -100 pp [-100%, -77%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 20/20 |
| ss-qwen-s42002-p7 | +100 pp [77%, 100%] | -100 pp [-100%, -77%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 20/20 |
| fp-qwen-s41901-p26 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes | 0/20 |
| ss-qwen-s42001-p20 | +85 pp [58%, 95%] | -85 pp [-95%, -58%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 17/20 |
| fl-qwen-s41901-p9 | +0 pp [-16%, 16%] | +10 pp [-8%, 30%] | -10 pp [-30%, 8%] | +0 pp [-16%, 16%] | +90 pp [64%, 97%] | no | 2/20 |
| fl-qwen-s41901-p11 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes | 0/20 |
| ss-qwen-s42001-p23 | +0 pp [-16%, 16%] | +10 pp [-11%, 31%] | -10 pp [-31%, 11%] | +0 pp [-16%, 16%] | +55 pp [24%, 73%] | no | 4/20 |
| fp-qwen-s42001-p27 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 0/20 |

## B validation − A neutral

| Point | Kept | Revised | Abandoned | Invalid | Final accepted | Same action content | Discordant rounds |
|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes | 0/10 |
| fl-gemma-s41901-p26 | -100 pp [-100%, -61%] | +100 pp [61%, 100%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | no | 10/10 |
| fl-gemma-s41901-p28 | -100 pp [-100%, -61%] | +100 pp [61%, 100%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +100 pp [61%, 100%] | no | 10/10 |
| fl-gemma-s41901-p27 | -100 pp [-100%, -61%] | +0 pp [-28%, 28%] | +100 pp [61%, 100%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | no | 10/10 |
| ss-qwen-pv-s42002-p6 | -60 pp [-78%, -33%] | +65 pp [38%, 82%] | +0 pp [-16%, 16%] | -5 pp [-24%, 12%] | +10 pp [-8%, 30%] | no | 13/20 |
| ss-qwen-s42002-p7 | -100 pp [-100%, -77%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 20/20 |
| fp-qwen-s41901-p26 | -100 pp [-100%, -77%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 20/20 |
| ss-qwen-s42001-p20 | -100 pp [-100%, -77%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 20/20 |
| fl-qwen-s41901-p9 | +0 pp [-16%, 16%] | -5 pp [-26%, 15%] | +10 pp [-8%, 30%] | -5 pp [-24%, 12%] | +10 pp [-8%, 30%] | no | 3/20 |
| fl-qwen-s41901-p11 | -100 pp [-100%, -77%] | +0 pp [-16%, 16%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 20/20 |
| ss-qwen-s42001-p23 | +0 pp [-16%, 16%] | -15 pp [-36%, 4%] | +15 pp [-4%, 36%] | +0 pp [-16%, 16%] | -75 pp [-89%, -48%] | no | 3/20 |
| fp-qwen-s42001-p27 | -100 pp [-100%, -77%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 20/20 |

## C preview − A neutral

| Point | Kept | Revised | Abandoned | Invalid | Final accepted | Same action content | Discordant rounds |
|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | yes | 0/10 |
| fl-gemma-s41901-p26 | -100 pp [-100%, -61%] | +100 pp [61%, 100%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | no | 10/10 |
| fl-gemma-s41901-p28 | -100 pp [-100%, -61%] | +100 pp [61%, 100%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | +100 pp [61%, 100%] | no | 10/10 |
| fl-gemma-s41901-p27 | -100 pp [-100%, -61%] | +0 pp [-28%, 28%] | +100 pp [61%, 100%] | +0 pp [-28%, 28%] | +0 pp [-28%, 28%] | no | 10/10 |
| ss-qwen-pv-s42002-p6 | +40 pp [16%, 61%] | -35 pp [-57%, -12%] | +0 pp [-16%, 16%] | -5 pp [-24%, 12%] | +10 pp [-8%, 30%] | no | 8/20 |
| ss-qwen-s42002-p7 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | yes | 0/20 |
| fp-qwen-s41901-p26 | -100 pp [-100%, -77%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 20/20 |
| ss-qwen-s42001-p20 | -15 pp [-36%, 4%] | +15 pp [-4%, 36%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 3/20 |
| fl-qwen-s41901-p9 | +0 pp [-16%, 16%] | +5 pp [-12%, 24%] | +0 pp [-16%, 16%] | -5 pp [-24%, 12%] | +100 pp [77%, 100%] | no | 1/20 |
| fl-qwen-s41901-p11 | -100 pp [-100%, -77%] | +0 pp [-16%, 16%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 20/20 |
| ss-qwen-s42001-p23 | +0 pp [-16%, 16%] | -5 pp [-24%, 12%] | +5 pp [-12%, 24%] | +0 pp [-16%, 16%] | -20 pp [-42%, 0%] | no | 1/20 |
| fp-qwen-s42001-p27 | -100 pp [-100%, -77%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | no | 20/20 |

## Pooled (after the per-point results)

Unweighted means of the per-point differences; the points are clusters, too few for an interval.

| Scope | Points | Comparison | Kept | Revised | Abandoned | Invalid | Points with the same action content |
|---|---|---|---|---|---|---|---|
| gemma/accepted | 1 | preview-validation | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 1 |
| gemma/accepted | 1 | validation-neutral | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 1 |
| gemma/accepted | 1 | preview-neutral | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 1 |
| gemma/refused | 3 | preview-validation | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 2 |
| gemma/refused | 3 | validation-neutral | -100.0 pp | +66.7 pp | +33.3 pp | +0.0 pp | 0 |
| gemma/refused | 3 | preview-neutral | -100.0 pp | +66.7 pp | +33.3 pp | +0.0 pp | 0 |
| gemma/all | 4 | preview-validation | +0.0 pp | +0.0 pp | +0.0 pp | +0.0 pp | 3 |
| gemma/all | 4 | validation-neutral | -75.0 pp | +50.0 pp | +25.0 pp | +0.0 pp | 1 |
| gemma/all | 4 | preview-neutral | -75.0 pp | +50.0 pp | +25.0 pp | +0.0 pp | 1 |
| qwen/accepted | 4 | preview-validation | +71.2 pp | -71.2 pp | +0.0 pp | +0.0 pp | 1 |
| qwen/accepted | 4 | validation-neutral | -90.0 pp | +91.2 pp | +0.0 pp | -1.2 pp | 0 |
| qwen/accepted | 4 | preview-neutral | -18.8 pp | +20.0 pp | +0.0 pp | -1.2 pp | 1 |
| qwen/refused | 4 | preview-validation | +0.0 pp | +5.0 pp | -5.0 pp | +0.0 pp | 1 |
| qwen/refused | 4 | validation-neutral | -50.0 pp | +20.0 pp | +31.2 pp | -1.2 pp | 0 |
| qwen/refused | 4 | preview-neutral | -50.0 pp | +25.0 pp | +26.2 pp | -1.2 pp | 0 |
| qwen/all | 8 | preview-validation | +35.6 pp | -33.1 pp | -2.5 pp | +0.0 pp | 2 |
| qwen/all | 8 | validation-neutral | -70.0 pp | +55.6 pp | +15.6 pp | -1.2 pp | 0 |
| qwen/all | 8 | preview-neutral | -34.4 pp | +22.5 pp | +13.1 pp | -1.2 pp | 1 |
| all/accepted | 5 | preview-validation | +57.0 pp | -57.0 pp | +0.0 pp | +0.0 pp | 2 |
| all/accepted | 5 | validation-neutral | -72.0 pp | +73.0 pp | +0.0 pp | -1.0 pp | 1 |
| all/accepted | 5 | preview-neutral | -15.0 pp | +16.0 pp | +0.0 pp | -1.0 pp | 2 |
| all/refused | 7 | preview-validation | +0.0 pp | +2.9 pp | -2.9 pp | +0.0 pp | 3 |
| all/refused | 7 | validation-neutral | -71.4 pp | +40.0 pp | +32.1 pp | -0.7 pp | 0 |
| all/refused | 7 | preview-neutral | -71.4 pp | +42.9 pp | +29.3 pp | -0.7 pp | 0 |
| all/all | 12 | preview-validation | +23.8 pp | -22.1 pp | -1.7 pp | +0.0 pp | 5 |
| all/all | 12 | validation-neutral | -71.7 pp | +53.8 pp | +18.8 pp | -0.8 pp | 1 |
| all/all | 12 | preview-neutral | -47.9 pp | +31.7 pp | +17.1 pp | -0.8 pp | 2 |

## What changed, per arm (draws)

| Point | Arm | Proposal changed | Became valid | Commitment changed | Exposure changed |
|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | A neutral | 0 | 0 | 0 | 0 |
| ss-gemma-pv-s42001-p14 | B validation | 0 | 0 | 0 | 0 |
| ss-gemma-pv-s42001-p14 | C preview | 0 | 0 | 0 | 0 |
| fl-gemma-s41901-p26 | A neutral | 0 | 0 | 0 | 0 |
| fl-gemma-s41901-p26 | B validation | 10 | 0 | 0 | 0 |
| fl-gemma-s41901-p26 | C preview | 10 | 0 | 0 | 0 |
| fl-gemma-s41901-p28 | A neutral | 0 | 0 | 0 | 0 |
| fl-gemma-s41901-p28 | B validation | 10 | 10 | 10 | 10 |
| fl-gemma-s41901-p28 | C preview | 10 | 10 | 10 | 10 |
| fl-gemma-s41901-p27 | A neutral | 0 | 0 | 0 | 0 |
| fl-gemma-s41901-p27 | B validation | 10 | 0 | 0 | 0 |
| fl-gemma-s41901-p27 | C preview | 10 | 0 | 0 | 0 |
| ss-qwen-pv-s42002-p6 | A neutral | 7 | 0 | 7 | 7 |
| ss-qwen-pv-s42002-p6 | B validation | 20 | 0 | 20 | 20 |
| ss-qwen-pv-s42002-p6 | C preview | 0 | 0 | 0 | 0 |
| ss-qwen-s42002-p7 | A neutral | 0 | 0 | 0 | 0 |
| ss-qwen-s42002-p7 | B validation | 20 | 0 | 20 | 20 |
| ss-qwen-s42002-p7 | C preview | 0 | 0 | 0 | 0 |
| fp-qwen-s41901-p26 | A neutral | 0 | 0 | 0 | 0 |
| fp-qwen-s41901-p26 | B validation | 20 | 0 | 0 | 0 |
| fp-qwen-s41901-p26 | C preview | 20 | 0 | 0 | 0 |
| ss-qwen-s42001-p20 | A neutral | 0 | 0 | 0 | 0 |
| ss-qwen-s42001-p20 | B validation | 20 | 0 | 0 | 0 |
| ss-qwen-s42001-p20 | C preview | 3 | 0 | 0 | 0 |
| fl-qwen-s41901-p9 | A neutral | 19 | 0 | 0 | 0 |
| fl-qwen-s41901-p9 | B validation | 20 | 2 | 0 | 2 |
| fl-qwen-s41901-p9 | C preview | 20 | 20 | 4 | 20 |
| fl-qwen-s41901-p11 | A neutral | 0 | 0 | 0 | 0 |
| fl-qwen-s41901-p11 | B validation | 20 | 0 | 0 | 0 |
| fl-qwen-s41901-p11 | C preview | 20 | 0 | 0 | 0 |
| ss-qwen-s42001-p23 | A neutral | 20 | 20 | 20 | 20 |
| ss-qwen-s42001-p23 | B validation | 20 | 5 | 5 | 5 |
| ss-qwen-s42001-p23 | C preview | 20 | 16 | 16 | 16 |
| fp-qwen-s42001-p27 | A neutral | 0 | 0 | 0 | 0 |
| fp-qwen-s42001-p27 | B validation | 20 | 0 | 0 | 0 |
| fp-qwen-s42001-p27 | C preview | 20 | 0 | 0 | 0 |

## Financial evidence v1 of the distinct final decisions

| Point | Proposal | Draws A / B / C | Outcome | Commitment | Credit-supported | Retained obligations | Reversibility | Direct exposure | Displaced bookings |
|---|---|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| ss-gemma-pv-s42001-p14 | yes | 10 / 10 / 10 | accepted | $3,069.98 | no | 1 | reversible_with_retained_obligation 1 | teleradiology coverage 1 | 0 |
| fl-gemma-s41901-p26 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-gemma-s41901-p26 |  | 0 / 10 / 10 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-gemma-s41901-p26 | yes | 10 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-gemma-s41901-p28 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-gemma-s41901-p28 |  | 0 / 10 / 0 | accepted | $3,023.32 | yes | 1 | reversible_with_retained_obligation 1 | teleradiology coverage 1 | 0 |
| fl-gemma-s41901-p28 |  | 0 / 0 / 10 | accepted | $1,462.90 | yes | 1 | reversible_with_retained_obligation 1 | teleradiology coverage 1 | 0 |
| fl-gemma-s41901-p28 | yes | 10 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-gemma-s41901-p27 |  | 0 / 10 / 10 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-gemma-s41901-p27 | yes | 10 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-pv-s42002-p6 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-pv-s42002-p6 |  | 2 / 0 / 0 | accepted | $44,687.30 | yes | 0 | not_reversible 2, withdrawable 2 | reporting allocation 2, scheduled duty 2 | 0 |
| ss-qwen-pv-s42002-p6 |  | 4 / 0 / 0 | accepted | $303.75 | no | 0 | not_reversible 2 | scheduled duty 2 | 0 |
| ss-qwen-pv-s42002-p6 |  | 2 / 20 / 0 | accepted | $303.75 | no | 0 | not_reversible 2 | scheduled duty 2 | 0 |
| ss-qwen-pv-s42002-p6 |  | 1 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-pv-s42002-p6 | yes | 10 / 0 / 20 | accepted | $44,687.30 | yes | 0 | not_reversible 2, withdrawable 2 | reporting allocation 2, scheduled duty 2 | 0 |
| ss-qwen-s42002-p7 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-s42002-p7 | yes | 20 / 0 / 20 | accepted | $3,228.33 | no | 1 | not_reversible 1, reversible_with_retained_obligation 1 | scheduled duty 1, teleradiology coverage 1 | 0 |
| ss-qwen-s42002-p7 |  | 0 / 20 / 0 | accepted | $135.00 | no | 0 | not_reversible 1 | scheduled duty 1 | 0 |
| fp-qwen-s41901-p26 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fp-qwen-s41901-p26 | yes | 20 / 0 / 0 | accepted | $9,863.02 | no | 0 | withdrawable 1 | reporting allocation 1 | 0 |
| fp-qwen-s41901-p26 |  | 0 / 20 / 20 | accepted | $9,863.02 | no | 0 | withdrawable 1 | reporting allocation 1 | 0 |
| ss-qwen-s42001-p20 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-s42001-p20 |  | 0 / 20 / 0 | accepted | $1,475.80 | no | 1 | reversible_with_retained_obligation 1, withdrawable 6 | rostered time 6, teleradiology coverage 1 | 405 |
| ss-qwen-s42001-p20 |  | 0 / 0 / 3 | accepted | $1,475.80 | no | 1 | reversible_with_retained_obligation 1, withdrawable 6 | rostered time 6, teleradiology coverage 1 | 405 |
| ss-qwen-s42001-p20 | yes | 20 / 0 / 17 | accepted | $1,475.80 | no | 1 | reversible_with_retained_obligation 1, withdrawable 6 | rostered time 6, teleradiology coverage 1 | 405 |
| fl-qwen-s41901-p9 |  | 0 / 2 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p9 |  | 3 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p9 |  | 1 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p9 |  | 2 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p9 | yes | 0 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p9 |  | 10 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p9 |  | 2 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p9 |  | 1 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p9 |  | 0 / 0 / 4 | accepted | $288.45 | yes | 0 | not_reversible 1, withdrawable 1 | reporting allocation 1, scheduled duty 1 | 0 |
| fl-qwen-s41901-p9 |  | 0 / 16 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p9 |  | 0 / 2 / 10 | accepted | $0.00 | no | 0 | withdrawable 1 | reporting allocation 1 | 0 |
| fl-qwen-s41901-p9 |  | 0 / 0 / 2 | accepted | $0.00 | no | 0 | not_reversible 1, withdrawable 1 | reporting allocation 1, technologist capacity 1 | 8 |
| fl-qwen-s41901-p9 |  | 0 / 0 / 4 | accepted | $0.00 | no | 0 | not_reversible 1 | technologist capacity 1 | 8 |
| fl-qwen-s41901-p11 |  | 0 / 20 / 20 | no_action | $0.00 | no | 0 | — | — | 0 |
| fl-qwen-s41901-p11 | yes | 20 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-s42001-p23 |  | 0 / 3 / 1 | no_action | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-s42001-p23 |  | 12 / 0 / 0 | accepted | $547.50 | no | 0 | not_reversible 4 | scheduled duty 4 | 0 |
| ss-qwen-s42001-p23 |  | 0 / 11 / 3 | refused | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-s42001-p23 |  | 1 / 3 / 0 | accepted | $468.75 | no | 0 | not_reversible 3, reversible_no_recorded_cost 1 | reporting order 1, scheduled duty 3 | 0 |
| ss-qwen-s42001-p23 |  | 7 / 2 / 16 | accepted | $468.75 | no | 0 | not_reversible 3 | scheduled duty 3 | 0 |
| ss-qwen-s42001-p23 | yes | 0 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| ss-qwen-s42001-p23 |  | 0 / 1 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fp-qwen-s42001-p27 |  | 0 / 0 / 0 | no_action | $0.00 | no | 0 | — | — | 0 |
| fp-qwen-s42001-p27 |  | 0 / 0 / 20 | refused | $0.00 | no | 0 | — | — | 0 |
| fp-qwen-s42001-p27 |  | 0 / 20 / 0 | refused | $0.00 | no | 0 | — | — | 0 |
| fp-qwen-s42001-p27 | yes | 20 / 0 / 0 | refused | $0.00 | no | 0 | — | — | 0 |

## Funding at the points

| Point | Funding headroom | Of it without credit | Proposal's stake |
|---|---|---|---|
| ss-gemma-pv-s42001-p14 | $103,953.60 | $53,953.60 | $3,069.98 |
| fl-gemma-s41901-p26 | $2,654.01 | −$47,345.99 | $0.00 |
| fl-gemma-s41901-p28 | $8,590.76 | −$41,409.24 | $0.00 |
| fl-gemma-s41901-p27 | −$2,429.59 | −$52,429.59 | $1,464.51 |
| ss-qwen-pv-s42002-p6 | $87,205.37 | $37,205.37 | $44,687.30 |
| ss-qwen-s42002-p7 | $100,306.40 | $50,306.40 | $3,228.33 |
| fp-qwen-s41901-p26 | $68,288.14 | $18,288.14 | $9,863.02 |
| ss-qwen-s42001-p20 | $53,511.19 | $3,511.19 | $1,475.80 |
| fl-qwen-s41901-p9 | $16,724.63 | −$33,275.37 | $36,978.91 |
| fl-qwen-s41901-p11 | $14,047.14 | −$35,952.86 | $24,657.53 |
| ss-qwen-s42001-p23 | $64,126.16 | $14,126.16 | $0.00 |
| fp-qwen-s42001-p27 | $78,386.71 | $28,386.71 | $0.00 |
