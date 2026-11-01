# V0.10b decision distributions

Each point and arm: 20 scheduled draws, all in the denominator. Wilson 95% intervals; differences are treatment − control with Newcombe 95% intervals. Descriptive only.

## Per point

| Point | Model | Preview | Arm | Kept | Revised | Abandoned | Invalid | Final accepted / refused / no action | Distinct decisions |
|---|---|---|---|---|---|---|---|---|---|
| ss-gemma-s42001-p14 | gemma | accepted | control | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| ss-gemma-s42001-p14 | gemma | accepted | treatment | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| fl-gemma-s41901-p27 | gemma | refused | control | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0 / 20 / 0 | 1 |
| fl-gemma-s41901-p27 | gemma | refused | treatment | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0 / 0 / 20 | 1 |
| fl-gemma-s41901-p28 | gemma | refused | control | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0 / 20 / 0 | 1 |
| fl-gemma-s41901-p28 | gemma | refused | treatment | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| fl-gemma-s41901-p26 | gemma | refused | control | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0 / 20 / 0 | 1 |
| fl-gemma-s41901-p26 | gemma | refused | treatment | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0 / 20 / 0 | 1 |
| ss-qwen-s42002-p6 | qwen | accepted | control | 11/20 [34%, 74%] | 8/20 [22%, 61%] | 0/20 [0%, 16%] | 1/20 [1%, 24%] | 19 / 0 / 0 | 5 |
| ss-qwen-s42002-p6 | qwen | accepted | treatment | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| ss-qwen-s42001-p1 | qwen | accepted | control | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| ss-qwen-s42001-p1 | qwen | accepted | treatment | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20 / 0 / 0 | 1 |
| ss-qwen-s42002-p26 | qwen | refused | control | 1/20 [1%, 24%] | 7/20 [18%, 57%] | 7/20 [18%, 57%] | 5/20 [11%, 47%] | 0 / 8 / 7 | 6 |
| ss-qwen-s42002-p26 | qwen | refused | treatment | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 20/20 [84%, 100%] | 0/20 [0%, 16%] | 0 / 0 / 20 | 1 |
| ss-qwen-s42002-p1 | qwen | refused | control | 16/20 [58%, 92%] | 4/20 [8%, 42%] | 0/20 [0%, 16%] | 0/20 [0%, 16%] | 0 / 20 / 0 | 3 |
| ss-qwen-s42002-p1 | qwen | refused | treatment | 0/20 [0%, 16%] | 19/20 [76%, 99%] | 0/20 [0%, 16%] | 1/20 [1%, 24%] | 19 / 0 / 0 | 2 |

## Differences (treatment − control)

| Point | Kept | Revised | Abandoned | Invalid | Discordant rounds |
|---|---|---|---|---|---|
| ss-gemma-s42001-p14 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | 0/20 |
| fl-gemma-s41901-p27 | -100 pp [-100%, -77%] | +0 pp [-16%, 16%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | 20/20 |
| fl-gemma-s41901-p28 | -100 pp [-100%, -77%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | 20/20 |
| fl-gemma-s41901-p26 | -100 pp [-100%, -77%] | +100 pp [77%, 100%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | 20/20 |
| ss-qwen-s42002-p6 | +45 pp [20%, 66%] | -40 pp [-61%, -16%] | +0 pp [-16%, 16%] | -5 pp [-24%, 12%] | 9/20 |
| ss-qwen-s42001-p1 | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | +0 pp [-16%, 16%] | 0/20 |
| ss-qwen-s42002-p26 | -5 pp [-24%, 12%] | -35 pp [-57%, -12%] | +65 pp [38%, 82%] | -25 pp [-47%, -4%] | 13/20 |
| ss-qwen-s42002-p1 | -80 pp [-92%, -53%] | +75 pp [46%, 88%] | +0 pp [-16%, 16%] | +5 pp [-12%, 24%] | 16/20 |

## Pooled (after the per-point results)

Unweighted means of the per-point differences; the points are clusters, and there are too few for an interval over them.

| Scope | Points | Kept | Revised | Abandoned | Invalid | Points with kept higher / lower under treatment |
|---|---|---|---|---|---|---|
| gemma | 4 | -75.0 pp | +50.0 pp | +25.0 pp | +0.0 pp | 0 / 3 |
| qwen | 4 | -10.0 pp | +0.0 pp | +16.2 pp | -6.2 pp | 1 / 2 |
| all | 8 | -42.5 pp | +25.0 pp | +20.6 pp | -3.1 pp | 1 / 5 |

## Agreement within an arm (distinct values / modal count)

| Point | Arm | Raw | Canonical JSON | Actions |
|---|---|---|---|---|
| ss-gemma-s42001-p14 | control | 1 / 20 | 1 / 20 | 1 / 20 |
| ss-gemma-s42001-p14 | treatment | 1 / 20 | 1 / 20 | 1 / 20 |
| fl-gemma-s41901-p27 | control | 1 / 20 | 1 / 20 | 1 / 20 |
| fl-gemma-s41901-p27 | treatment | 2 / 19 | 2 / 19 | 1 / 20 |
| fl-gemma-s41901-p28 | control | 1 / 20 | 1 / 20 | 1 / 20 |
| fl-gemma-s41901-p28 | treatment | 1 / 20 | 1 / 20 | 1 / 20 |
| fl-gemma-s41901-p26 | control | 1 / 20 | 1 / 20 | 1 / 20 |
| fl-gemma-s41901-p26 | treatment | 1 / 20 | 1 / 20 | 1 / 20 |
| ss-qwen-s42002-p6 | control | 17 / 2 | 17 / 2 | 5 / 11 |
| ss-qwen-s42002-p6 | treatment | 3 / 12 | 3 / 12 | 1 / 20 |
| ss-qwen-s42001-p1 | control | 5 / 6 | 5 / 6 | 1 / 20 |
| ss-qwen-s42001-p1 | treatment | 3 / 17 | 3 / 17 | 1 / 20 |
| ss-qwen-s42002-p26 | control | 16 / 5 | 15 / 5 | 5 / 7 |
| ss-qwen-s42002-p26 | treatment | 4 / 11 | 4 / 11 | 1 / 20 |
| ss-qwen-s42002-p1 | control | 9 / 5 | 9 / 5 | 3 / 16 |
| ss-qwen-s42002-p1 | treatment | 2 / 19 | 2 / 19 | 2 / 19 |

## Financial evidence v1 of the distinct final decisions

| Point | Proposal | Draws (control / treatment) | Outcome | Commitment | Credit-supported | Reversibility | Direct exposure |
|---|---|---|---|---|---|---|---|
| ss-gemma-s42001-p14 |  | 0 / 0 | no_action | $0.00 | no | — | — |
| ss-gemma-s42001-p14 | yes | 20 / 20 | accepted | $3,069.98 | no | reversible_with_retained_obligation 1 | teleradiology coverage 1 |
| fl-gemma-s41901-p27 |  | 0 / 20 | no_action | $0.00 | no | — | — |
| fl-gemma-s41901-p27 | yes | 20 / 0 | refused | $0.00 | no | — | — |
| fl-gemma-s41901-p28 |  | 0 / 0 | no_action | $0.00 | no | — | — |
| fl-gemma-s41901-p28 |  | 0 / 20 | accepted | $1,462.90 | yes | reversible_with_retained_obligation 1 | teleradiology coverage 1 |
| fl-gemma-s41901-p28 | yes | 20 / 0 | refused | $0.00 | no | — | — |
| fl-gemma-s41901-p26 |  | 0 / 0 | no_action | $0.00 | no | — | — |
| fl-gemma-s41901-p26 |  | 0 / 20 | refused | $0.00 | no | — | — |
| fl-gemma-s41901-p26 | yes | 20 / 0 | refused | $0.00 | no | — | — |
| ss-qwen-s42002-p6 |  | 0 / 0 | no_action | $0.00 | no | — | — |
| ss-qwen-s42002-p6 |  | 3 / 0 | accepted | $303.75 | no | not_reversible 2 | scheduled duty 2 |
| ss-qwen-s42002-p6 |  | 4 / 0 | accepted | $303.75 | no | not_reversible 2 | scheduled duty 2 |
| ss-qwen-s42002-p6 | yes | 11 / 20 | accepted | $44,687.30 | yes | not_reversible 2, withdrawable 2 | reporting allocation 2, scheduled duty 2 |
| ss-qwen-s42002-p6 |  | 1 / 0 | accepted | $303.75 | no | not_reversible 2, reversible_no_recorded_cost 1 | reporting order 1, scheduled duty 2 |
| ss-qwen-s42001-p1 |  | 0 / 0 | no_action | $0.00 | no | — | — |
| ss-qwen-s42001-p1 | yes | 20 / 20 | accepted | $2,867.02 | no | not_reversible 1, reversible_with_retained_obligation 1 | scheduled duty 1, teleradiology coverage 1 |
| ss-qwen-s42002-p26 |  | 7 / 20 | no_action | $0.00 | no | — | — |
| ss-qwen-s42002-p26 |  | 6 / 0 | refused | $0.00 | no | — | — |
| ss-qwen-s42002-p26 |  | 1 / 0 | refused | $0.00 | no | — | — |
| ss-qwen-s42002-p26 | yes | 1 / 0 | refused | $0.00 | no | — | — |
| ss-qwen-s42002-p1 |  | 0 / 0 | no_action | $0.00 | no | — | — |
| ss-qwen-s42002-p1 |  | 3 / 0 | refused | $0.00 | no | — | — |
| ss-qwen-s42002-p1 |  | 0 / 19 | accepted | $1,797.30 | no | not_reversible 2, reversible_with_retained_obligation 1 | scheduled duty 2, teleradiology coverage 1 |
| ss-qwen-s42002-p1 | yes | 16 / 0 | refused | $0.00 | no | — | — |
| ss-qwen-s42002-p1 |  | 1 / 0 | refused | $0.00 | no | — | — |
