# V0.7b characterization: v07b-characterize-v2 (suite dreamteam-dev 2) and v07b-characterize-v3 (suite dreamteam-dev 3)

Seeds 41501, 41502, 41503, 90 days. Version 2: complete (45 of 45); version 3: complete (90 of 90). Engines 0.7.0-dev.2, 0.7.0-dev.2.

## 1. Starting states

| entry | seed | v2 start | v3 (both variants) identical |
|---|---|---|---|
| financial-pressure | 41501 | 1b48c51ab52c… | yes |
| financial-pressure | 41502 | 0096b50a2646… | yes |
| financial-pressure | 41503 | 3d0a3072d696… | yes |
| reporting-backlog | 41501 | b3578ec60270… | yes |
| reporting-backlog | 41502 | cd1b88dbd16d… | yes |
| reporting-backlog | 41503 | e4fee0e6443a… | yes |
| demand-growth | 41501 | b3578ec60270… | yes |
| demand-growth | 41502 | cd1b88dbd16d… | yes |
| demand-growth | 41503 | e4fee0e6443a… | yes |
| staffing-shortage | 41501 | 487f60be0c82… | yes |
| staffing-shortage | 41502 | f029412674e2… | yes |
| staffing-shortage | 41503 | 1a4220e3c725… | yes |
| stable | 41501 | 48f8d45955b6… | yes |
| stable | 41502 | d0d3e4050df0… | yes |
| stable | 41503 | 6f10a216f07d… | yes |

## 2. Binding: any refusal, draw, interest or distress

| entry | policy | variant | binds | refusals (3 seeds) | most drawn | interest (3 seeds) | runs in distress | largest shortfall | actions as v2 |
|---|---|---|---|---|---|---|---|---|---|
| financial-pressure | noop | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| financial-pressure | noop | nocredit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| financial-pressure | greedy | credit | **yes** | 3 | $12,584 | $8 | 0 | $0 | 1 of 3 |
| financial-pressure | greedy | nocredit | **yes** | 23 | $0 | $0 | 1 | $1,981 | 0 of 3 |
| financial-pressure | conservative-v2 | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| financial-pressure | conservative-v2 | nocredit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| reporting-backlog | noop | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| reporting-backlog | noop | nocredit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| reporting-backlog | greedy | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| reporting-backlog | greedy | nocredit | **yes** | 12 | $0 | $0 | 0 | $0 | 0 of 3 |
| reporting-backlog | conservative-v2 | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| reporting-backlog | conservative-v2 | nocredit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| demand-growth | noop | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| demand-growth | noop | nocredit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| demand-growth | greedy | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| demand-growth | greedy | nocredit | **yes** | 11 | $0 | $0 | 0 | $0 | 0 of 3 |
| demand-growth | conservative-v2 | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| demand-growth | conservative-v2 | nocredit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| staffing-shortage | noop | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| staffing-shortage | noop | nocredit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| staffing-shortage | greedy | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| staffing-shortage | greedy | nocredit | **yes** | 15 | $0 | $0 | 0 | $0 | 0 of 3 |
| staffing-shortage | conservative-v2 | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| staffing-shortage | conservative-v2 | nocredit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| stable | noop | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| stable | noop | nocredit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| stable | greedy | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| stable | greedy | nocredit | **yes** | 4 | $0 | $0 | 0 | $0 | 1 of 3 |
| stable | conservative-v2 | credit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |
| stable | conservative-v2 | nocredit | no | 0 | $0 | $0 | 0 | $0 | 3 of 3 |

## 3. Each run

| entry | seed | policy | profit v2 | min cash v2 | accepted v2 | profit credit | min cash credit | accepted credit | refused credit | most drawn credit | interest credit | shortfall credit | status credit | profit nocredit | min cash nocredit | accepted nocredit | refused nocredit | most drawn nocredit | interest nocredit | shortfall nocredit | status nocredit |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure | 41501 | noop | -$29,456 | $38,063 | 0 | -$29,456 | $38,063 | 0 | 0 | $0 | $0 | $0 | ok | -$29,456 | $38,063 | 0 | 0 | $0 | $0 | $0 | ok |
| financial-pressure | 41501 | greedy | -$59,123 | $9,108 | 12 | -$59,123 | $9,108 | 12 | 0 | $0 | $0 | $0 | ok | -$52,959 | $13,875 | 6 | 6 | $0 | $0 | $0 | ok |
| financial-pressure | 41501 | conservative-v2 | -$29,456 | $38,063 | 0 | -$29,456 | $38,063 | 0 | 0 | $0 | $0 | $0 | ok | -$29,456 | $38,063 | 0 | 0 | $0 | $0 | $0 | ok |
| financial-pressure | 41502 | noop | -$56,065 | $15,919 | 0 | -$56,065 | $15,919 | 0 | 0 | $0 | $0 | $0 | ok | -$56,065 | $15,919 | 0 | 0 | $0 | $0 | $0 | ok |
| financial-pressure | 41502 | greedy | -$88,697 | -$12,584 | 12 | -$88,623 | $0 | 10 | 2 | $12,584 | $8 | $0 | ok | -$76,040 | -$1,981 | 4 | 9 | $0 | $0 | $1,981 | ok |
| financial-pressure | 41502 | conservative-v2 | -$56,065 | $15,919 | 0 | -$56,065 | $15,919 | 0 | 0 | $0 | $0 | $0 | ok | -$56,065 | $15,919 | 0 | 0 | $0 | $0 | $0 | ok |
| financial-pressure | 41503 | noop | -$45,897 | $27,276 | 0 | -$45,897 | $27,276 | 0 | 0 | $0 | $0 | $0 | ok | -$45,897 | $27,276 | 0 | 0 | $0 | $0 | $0 | ok |
| financial-pressure | 41503 | greedy | -$78,450 | $1,623 | 12 | -$78,450 | $1,623 | 11 | 1 | $0 | $0 | $0 | ok | -$65,792 | $12,271 | 4 | 8 | $0 | $0 | $0 | ok |
| financial-pressure | 41503 | conservative-v2 | -$45,897 | $27,276 | 0 | -$45,897 | $27,276 | 0 | 0 | $0 | $0 | $0 | ok | -$45,897 | $27,276 | 0 | 0 | $0 | $0 | $0 | ok |
| reporting-backlog | 41501 | noop | -$5,637 | $50,242 | 0 | -$5,637 | $50,242 | 0 | 0 | $0 | $0 | $0 | ok | -$5,637 | $50,242 | 0 | 0 | $0 | $0 | $0 | ok |
| reporting-backlog | 41501 | greedy | -$35,562 | $33,846 | 12 | -$35,562 | $33,846 | 12 | 0 | $0 | $0 | $0 | ok | -$34,823 | $34,174 | 9 | 3 | $0 | $0 | $0 | ok |
| reporting-backlog | 41501 | conservative-v2 | -$5,637 | $50,242 | 0 | -$5,637 | $50,242 | 0 | 0 | $0 | $0 | $0 | ok | -$5,637 | $50,242 | 0 | 0 | $0 | $0 | $0 | ok |
| reporting-backlog | 41502 | noop | -$21,851 | $47,444 | 0 | -$21,851 | $47,444 | 0 | 0 | $0 | $0 | $0 | ok | -$21,851 | $47,444 | 0 | 0 | $0 | $0 | $0 | ok |
| reporting-backlog | 41502 | greedy | -$53,441 | $22,856 | 12 | -$53,441 | $22,856 | 12 | 0 | $0 | $0 | $0 | ok | -$49,660 | $25,568 | 7 | 5 | $0 | $0 | $0 | ok |
| reporting-backlog | 41502 | conservative-v2 | -$21,851 | $47,444 | 0 | -$21,851 | $47,444 | 0 | 0 | $0 | $0 | $0 | ok | -$21,851 | $47,444 | 0 | 0 | $0 | $0 | $0 | ok |
| reporting-backlog | 41503 | noop | -$19,164 | $45,932 | 0 | -$19,164 | $45,932 | 0 | 0 | $0 | $0 | $0 | ok | -$19,164 | $45,932 | 0 | 0 | $0 | $0 | $0 | ok |
| reporting-backlog | 41503 | greedy | -$51,720 | $35,711 | 12 | -$51,720 | $35,711 | 12 | 0 | $0 | $0 | $0 | ok | -$49,747 | $36,944 | 8 | 4 | $0 | $0 | $0 | ok |
| reporting-backlog | 41503 | conservative-v2 | -$19,164 | $45,932 | 0 | -$19,164 | $45,932 | 0 | 0 | $0 | $0 | $0 | ok | -$19,164 | $45,932 | 0 | 0 | $0 | $0 | $0 | ok |
| demand-growth | 41501 | noop | -$3,754 | $50,242 | 0 | -$3,754 | $50,242 | 0 | 0 | $0 | $0 | $0 | ok | -$3,754 | $50,242 | 0 | 0 | $0 | $0 | $0 | ok |
| demand-growth | 41501 | greedy | -$30,965 | $32,696 | 12 | -$30,965 | $32,696 | 12 | 0 | $0 | $0 | $0 | ok | -$30,226 | $33,024 | 9 | 3 | $0 | $0 | $0 | ok |
| demand-growth | 41501 | conservative-v2 | -$1,181 | $50,242 | 1 | -$1,181 | $50,242 | 1 | 0 | $0 | $0 | $0 | ok | -$1,181 | $50,242 | 1 | 0 | $0 | $0 | $0 | ok |
| demand-growth | 41502 | noop | -$16,767 | $47,444 | 0 | -$16,767 | $47,444 | 0 | 0 | $0 | $0 | $0 | ok | -$16,767 | $47,444 | 0 | 0 | $0 | $0 | $0 | ok |
| demand-growth | 41502 | greedy | -$43,760 | $25,210 | 12 | -$43,760 | $25,210 | 12 | 0 | $0 | $0 | $0 | ok | -$41,787 | $26,443 | 8 | 4 | $0 | $0 | $0 | ok |
| demand-growth | 41502 | conservative-v2 | -$16,767 | $47,444 | 0 | -$16,767 | $47,444 | 0 | 0 | $0 | $0 | $0 | ok | -$16,767 | $47,444 | 0 | 0 | $0 | $0 | $0 | ok |
| demand-growth | 41503 | noop | -$9,805 | $45,932 | 0 | -$9,805 | $45,932 | 0 | 0 | $0 | $0 | $0 | ok | -$9,805 | $45,932 | 0 | 0 | $0 | $0 | $0 | ok |
| demand-growth | 41503 | greedy | -$40,149 | $35,092 | 12 | -$40,149 | $35,092 | 12 | 0 | $0 | $0 | $0 | ok | -$38,176 | $36,325 | 8 | 4 | $0 | $0 | $0 | ok |
| demand-growth | 41503 | conservative-v2 | -$9,805 | $45,932 | 0 | -$9,805 | $45,932 | 0 | 0 | $0 | $0 | $0 | ok | -$9,805 | $45,932 | 0 | 0 | $0 | $0 | $0 | ok |
| staffing-shortage | 41501 | noop | -$37,809 | $42,387 | 0 | -$37,809 | $42,387 | 0 | 0 | $0 | $0 | $0 | ok | -$37,809 | $42,387 | 0 | 0 | $0 | $0 | $0 | ok |
| staffing-shortage | 41501 | greedy | -$69,339 | $13,567 | 12 | -$69,339 | $13,567 | 12 | 0 | $0 | $0 | $0 | ok | -$67,366 | $14,800 | 8 | 4 | $0 | $0 | $0 | ok |
| staffing-shortage | 41501 | conservative-v2 | -$37,809 | $42,387 | 0 | -$37,809 | $42,387 | 0 | 0 | $0 | $0 | $0 | ok | -$37,809 | $42,387 | 0 | 0 | $0 | $0 | $0 | ok |
| staffing-shortage | 41502 | noop | -$46,051 | $37,024 | 0 | -$46,051 | $37,024 | 0 | 0 | $0 | $0 | $0 | ok | -$46,051 | $37,024 | 0 | 0 | $0 | $0 | $0 | ok |
| staffing-shortage | 41502 | greedy | -$78,683 | $8,298 | 12 | -$78,683 | $8,298 | 12 | 0 | $0 | $0 | $0 | ok | -$74,902 | $11,011 | 7 | 5 | $0 | $0 | $0 | ok |
| staffing-shortage | 41502 | conservative-v2 | -$46,051 | $37,024 | 0 | -$46,051 | $37,024 | 0 | 0 | $0 | $0 | $0 | ok | -$46,051 | $37,024 | 0 | 0 | $0 | $0 | $0 | ok |
| staffing-shortage | 41503 | noop | -$41,692 | $42,705 | 0 | -$41,692 | $42,705 | 0 | 0 | $0 | $0 | $0 | ok | -$41,692 | $42,705 | 0 | 0 | $0 | $0 | $0 | ok |
| staffing-shortage | 41503 | greedy | -$74,249 | $17,118 | 12 | -$74,249 | $17,118 | 12 | 0 | $0 | $0 | $0 | ok | -$68,085 | $21,885 | 6 | 6 | $0 | $0 | $0 | ok |
| staffing-shortage | 41503 | conservative-v2 | -$41,692 | $42,705 | 0 | -$41,692 | $42,705 | 0 | 0 | $0 | $0 | $0 | ok | -$41,692 | $42,705 | 0 | 0 | $0 | $0 | $0 | ok |
| stable | 41501 | noop | $22,269 | $56,748 | 0 | $22,269 | $56,748 | 0 | 0 | $0 | $0 | $0 | ok | $22,269 | $56,748 | 0 | 0 | $0 | $0 | $0 | ok |
| stable | 41501 | greedy | -$6,891 | $56,748 | 12 | -$6,891 | $56,748 | 12 | 0 | $0 | $0 | $0 | ok | -$6,891 | $56,748 | 12 | 0 | $0 | $0 | $0 | ok |
| stable | 41501 | conservative-v2 | $21,567 | $56,748 | 3 | $21,567 | $56,748 | 3 | 0 | $0 | $0 | $0 | ok | $21,567 | $56,748 | 3 | 0 | $0 | $0 | $0 | ok |
| stable | 41502 | noop | $3,030 | $51,041 | 0 | $3,030 | $51,041 | 0 | 0 | $0 | $0 | $0 | ok | $3,030 | $51,041 | 0 | 0 | $0 | $0 | $0 | ok |
| stable | 41502 | greedy | -$29,626 | $44,006 | 12 | -$29,626 | $44,006 | 12 | 0 | $0 | $0 | $0 | ok | -$28,886 | $44,335 | 9 | 3 | $0 | $0 | $0 | ok |
| stable | 41502 | conservative-v2 | $2,208 | $51,041 | 1 | $2,208 | $51,041 | 1 | 0 | $0 | $0 | $0 | ok | $2,208 | $51,041 | 1 | 0 | $0 | $0 | $0 | ok |
| stable | 41503 | noop | $12,000 | $49,471 | 0 | $12,000 | $49,471 | 0 | 0 | $0 | $0 | $0 | ok | $12,000 | $49,471 | 0 | 0 | $0 | $0 | $0 | ok |
| stable | 41503 | greedy | -$20,558 | $49,471 | 12 | -$20,558 | $49,471 | 12 | 0 | $0 | $0 | $0 | ok | -$20,558 | $49,471 | 11 | 1 | $0 | $0 | $0 | ok |
| stable | 41503 | conservative-v2 | $11,178 | $49,471 | 1 | $11,178 | $49,471 | 1 | 0 | $0 | $0 | $0 | ok | $11,178 | $49,471 | 1 | 0 | $0 | $0 | $0 | ok |

## 4. Version 3 minus version 2 (same practice, seed and policy)

| entry | seed | policy | variant | actions same | Δ expense | Δ profit | Δ revenue | Δ cash end | Δ min cash | Δ TAT p90 min | Δ breaches | Δ exams | Δ lost |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure | 41501 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41501 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41501 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41501 | greedy | nocredit | **no** | -$6,164 | +$6,164 | $0 | +$4,767 | +$4,767 | +11 | +5 | 0 | 0 |
| financial-pressure | 41501 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41501 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41502 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41502 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41502 | greedy | credit | **no** | -$75 | +$75 | $0 | +$7,685 | +$12,584 | 0 | 0 | 0 | 0 |
| financial-pressure | 41502 | greedy | nocredit | **no** | -$12,658 | +$12,658 | $0 | +$10,603 | +$10,603 | +24 | +4 | 0 | 0 |
| financial-pressure | 41502 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41502 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41503 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41503 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41503 | greedy | credit | **no** | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41503 | greedy | nocredit | **no** | -$12,658 | +$12,658 | $0 | +$10,603 | +$10,648 | +12 | +6 | 0 | 0 |
| financial-pressure | 41503 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41503 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41501 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41501 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41501 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41501 | greedy | nocredit | **no** | -$740 | +$740 | $0 | +$329 | +$329 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41501 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41501 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41502 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41502 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41502 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41502 | greedy | nocredit | **no** | -$3,781 | +$3,781 | $0 | +$2,712 | +$2,712 | +5 | +4 | 0 | 0 |
| reporting-backlog | 41502 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41502 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41503 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41503 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41503 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41503 | greedy | nocredit | **no** | -$1,973 | +$1,973 | $0 | +$1,233 | +$1,233 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41503 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41503 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41501 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41501 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41501 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41501 | greedy | nocredit | **no** | -$740 | +$740 | $0 | +$329 | +$329 | 0 | 0 | 0 | 0 |
| demand-growth | 41501 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41501 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41502 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41502 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41502 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41502 | greedy | nocredit | **no** | -$1,973 | +$1,973 | $0 | +$1,233 | +$1,233 | 0 | 0 | 0 | 0 |
| demand-growth | 41502 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41502 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41503 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41503 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41503 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41503 | greedy | nocredit | **no** | -$1,973 | +$1,973 | $0 | +$1,233 | +$1,233 | +1 | 0 | 0 | 0 |
| demand-growth | 41503 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| demand-growth | 41503 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41501 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41501 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41501 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41501 | greedy | nocredit | **no** | -$1,973 | +$1,973 | $0 | +$1,233 | +$1,233 | +1 | 0 | 0 | 0 |
| staffing-shortage | 41501 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41501 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41502 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41502 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41502 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41502 | greedy | nocredit | **no** | -$3,781 | +$3,781 | $0 | +$2,712 | +$2,712 | +8 | 0 | 0 | 0 |
| staffing-shortage | 41502 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41502 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41503 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41503 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41503 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41503 | greedy | nocredit | **no** | -$6,164 | +$6,164 | $0 | +$4,767 | +$4,767 | +2 | +1 | 0 | 0 |
| staffing-shortage | 41503 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41503 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41501 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41501 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41501 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41501 | greedy | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41501 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41501 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41502 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41502 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41502 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41502 | greedy | nocredit | **no** | -$740 | +$740 | $0 | +$329 | +$329 | 0 | 0 | 0 | 0 |
| stable | 41502 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41502 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41503 | noop | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41503 | noop | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41503 | greedy | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41503 | greedy | nocredit | **no** | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41503 | conservative-v2 | credit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |
| stable | 41503 | conservative-v2 | nocredit | yes | $0 | $0 | $0 | $0 | $0 | 0 | 0 | 0 | 0 |

## 5. Paired with no action (policy − noop, same entry and seed)

| entry | seed | policy | Δ profit v2 | Δ breaches v2 | Δ profit credit | Δ breaches credit | Δ profit nocredit | Δ breaches nocredit |
|---|---|---|---|---|---|---|---|---|
| financial-pressure | 41501 | greedy | -$29,667 | -106 | -$29,667 | -106 | -$23,502 | -101 |
| financial-pressure | 41501 | conservative-v2 | $0 | 0 | $0 | 0 | $0 | 0 |
| financial-pressure | 41502 | greedy | -$32,632 | -56 | -$32,558 | -56 | -$19,975 | -52 |
| financial-pressure | 41502 | conservative-v2 | $0 | 0 | $0 | 0 | $0 | 0 |
| financial-pressure | 41503 | greedy | -$32,553 | -56 | -$32,553 | -56 | -$19,895 | -50 |
| financial-pressure | 41503 | conservative-v2 | $0 | 0 | $0 | 0 | $0 | 0 |
| reporting-backlog | 41501 | greedy | -$29,925 | -184 | -$29,925 | -184 | -$29,185 | -184 |
| reporting-backlog | 41501 | conservative-v2 | $0 | 0 | $0 | 0 | $0 | 0 |
| reporting-backlog | 41502 | greedy | -$31,590 | -131 | -$31,590 | -131 | -$27,809 | -127 |
| reporting-backlog | 41502 | conservative-v2 | $0 | 0 | $0 | 0 | $0 | 0 |
| reporting-backlog | 41503 | greedy | -$32,556 | -126 | -$32,556 | -126 | -$30,584 | -126 |
| reporting-backlog | 41503 | conservative-v2 | $0 | 0 | $0 | 0 | $0 | 0 |
| demand-growth | 41501 | greedy | -$27,211 | -329 | -$27,211 | -329 | -$26,471 | -329 |
| demand-growth | 41501 | conservative-v2 | +$2,574 | -23 | +$2,574 | -23 | +$2,574 | -23 |
| demand-growth | 41502 | greedy | -$26,993 | -196 | -$26,993 | -196 | -$25,020 | -196 |
| demand-growth | 41502 | conservative-v2 | $0 | 0 | $0 | 0 | $0 | 0 |
| demand-growth | 41503 | greedy | -$30,344 | -213 | -$30,344 | -213 | -$28,371 | -213 |
| demand-growth | 41503 | conservative-v2 | $0 | 0 | $0 | 0 | $0 | 0 |
| staffing-shortage | 41501 | greedy | -$31,530 | -33 | -$31,530 | -33 | -$29,558 | -33 |
| staffing-shortage | 41501 | conservative-v2 | $0 | 0 | $0 | 0 | $0 | 0 |
| staffing-shortage | 41502 | greedy | -$32,632 | -33 | -$32,632 | -33 | -$28,852 | -33 |
| staffing-shortage | 41502 | conservative-v2 | $0 | 0 | $0 | 0 | $0 | 0 |
| staffing-shortage | 41503 | greedy | -$32,558 | -33 | -$32,558 | -33 | -$26,393 | -32 |
| staffing-shortage | 41503 | conservative-v2 | $0 | 0 | $0 | 0 | $0 | 0 |
| stable | 41501 | greedy | -$29,160 | -162 | -$29,160 | -162 | -$29,160 | -162 |
| stable | 41501 | conservative-v2 | -$701 | -75 | -$701 | -75 | -$701 | -75 |
| stable | 41502 | greedy | -$32,656 | -60 | -$32,656 | -60 | -$31,916 | -60 |
| stable | 41502 | conservative-v2 | -$822 | -4 | -$822 | -4 | -$822 | -4 |
| stable | 41503 | greedy | -$32,558 | -91 | -$32,558 | -91 | -$32,558 | -91 |
| stable | 41503 | conservative-v2 | -$822 | -3 | -$822 | -3 | -$822 | -3 |

## 6. Refusals and differing actions

* financial-pressure 41501 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500)]
  * refused insufficient_funds: [('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E07', 8000), ('2026-12-14', 'set_reading_allocation', 'E07', 8000), ('2026-12-21', 'set_reading_allocation', 'E07', 8000), ('2026-12-28', 'set_reading_allocation', 'E07', 8000), ('2027-01-04', 'set_reading_allocation', 'E07', 8000)]
* financial-pressure 41502 greedy credit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500)]
  * refused insufficient_funds: [('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E07', 9000)]
* financial-pressure 41502 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000)]
  * refused insufficient_funds: [('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-22', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E07', 7500), ('2026-11-30', 'set_reading_allocation', 'E07', 7500), ('2026-12-07', 'set_reading_allocation', 'E07', 7500), ('2026-12-14', 'set_reading_allocation', 'E07', 7500), ('2026-12-21', 'set_reading_allocation', 'E07', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 7500), ('2027-01-04', 'set_reading_allocation', 'E07', 7500)]
* financial-pressure 41503 greedy credit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000)]
  * refused insufficient_funds: [('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
* financial-pressure 41503 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000)]
  * refused insufficient_funds: [('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E07', 7500), ('2026-11-30', 'set_reading_allocation', 'E07', 7500), ('2026-12-07', 'set_reading_allocation', 'E07', 7500), ('2026-12-14', 'set_reading_allocation', 'E07', 7500), ('2026-12-21', 'set_reading_allocation', 'E07', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 7500), ('2027-01-04', 'set_reading_allocation', 'E07', 7500)]
* reporting-backlog 41501 greedy nocredit:
  * version 2 accepted: [('2026-10-18', 'set_reading_allocation', 'E07', 6500), ('2026-10-19', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-18', 'set_reading_allocation', 'E07', 6500), ('2026-10-19', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500)]
  * refused insufficient_funds: [('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E07', 9000)]
* reporting-backlog 41502 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000)]
  * refused insufficient_funds: [('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E07', 8500), ('2026-12-28', 'set_reading_allocation', 'E07', 8500), ('2027-01-04', 'set_reading_allocation', 'E07', 8500)]
* reporting-backlog 41503 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000)]
  * refused insufficient_funds: [('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E07', 8500), ('2026-12-28', 'set_reading_allocation', 'E07', 8500), ('2027-01-04', 'set_reading_allocation', 'E07', 8500)]
* demand-growth 41501 greedy nocredit:
  * version 2 accepted: [('2026-10-18', 'set_reading_allocation', 'E07', 6500), ('2026-10-19', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-18', 'set_reading_allocation', 'E07', 6500), ('2026-10-19', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500)]
  * refused insufficient_funds: [('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E07', 9000)]
* demand-growth 41502 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000)]
  * refused insufficient_funds: [('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E07', 8500), ('2026-12-28', 'set_reading_allocation', 'E07', 8500), ('2027-01-04', 'set_reading_allocation', 'E07', 8500)]
* demand-growth 41503 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000)]
  * refused insufficient_funds: [('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E07', 8500), ('2026-12-28', 'set_reading_allocation', 'E07', 8500), ('2027-01-04', 'set_reading_allocation', 'E07', 8500)]
* staffing-shortage 41501 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000)]
  * refused insufficient_funds: [('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E07', 8500), ('2026-12-28', 'set_reading_allocation', 'E07', 8500), ('2027-01-04', 'set_reading_allocation', 'E07', 8500)]
* staffing-shortage 41502 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000)]
  * refused insufficient_funds: [('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E07', 8500), ('2026-12-28', 'set_reading_allocation', 'E07', 8500), ('2027-01-04', 'set_reading_allocation', 'E07', 8500)]
* staffing-shortage 41503 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500), ('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E08', 7000), ('2026-12-14', 'set_reading_allocation', 'E07', 8500), ('2026-12-21', 'set_reading_allocation', 'E08', 7500), ('2026-12-28', 'set_reading_allocation', 'E07', 9000), ('2027-01-04', 'set_reading_allocation', 'E08', 8000)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6500), ('2026-10-26', 'set_reading_allocation', 'E08', 5500), ('2026-11-02', 'set_reading_allocation', 'E07', 7000), ('2026-11-09', 'set_reading_allocation', 'E08', 6000), ('2026-11-16', 'set_reading_allocation', 'E07', 7500), ('2026-11-23', 'set_reading_allocation', 'E08', 6500)]
  * refused insufficient_funds: [('2026-11-30', 'set_reading_allocation', 'E07', 8000), ('2026-12-07', 'set_reading_allocation', 'E07', 8000), ('2026-12-14', 'set_reading_allocation', 'E07', 8000), ('2026-12-21', 'set_reading_allocation', 'E07', 8000), ('2026-12-28', 'set_reading_allocation', 'E07', 8000), ('2027-01-04', 'set_reading_allocation', 'E07', 8000)]
* stable 41502 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6750), ('2026-10-26', 'set_reading_allocation', 'E08', 5750), ('2026-11-02', 'set_reading_allocation', 'E07', 7250), ('2026-11-09', 'set_reading_allocation', 'E08', 6250), ('2026-11-16', 'set_reading_allocation', 'E07', 7750), ('2026-11-23', 'set_reading_allocation', 'E08', 6750), ('2026-11-30', 'set_reading_allocation', 'E07', 8250), ('2026-12-07', 'set_reading_allocation', 'E08', 7250), ('2026-12-14', 'set_reading_allocation', 'E07', 8750), ('2026-12-21', 'set_reading_allocation', 'E08', 7750), ('2026-12-28', 'set_reading_allocation', 'E07', 9250), ('2027-01-04', 'set_reading_allocation', 'E08', 8250)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6750), ('2026-10-26', 'set_reading_allocation', 'E08', 5750), ('2026-11-02', 'set_reading_allocation', 'E07', 7250), ('2026-11-09', 'set_reading_allocation', 'E08', 6250), ('2026-11-16', 'set_reading_allocation', 'E07', 7750), ('2026-11-23', 'set_reading_allocation', 'E08', 6750), ('2026-11-30', 'set_reading_allocation', 'E07', 8250), ('2026-12-07', 'set_reading_allocation', 'E08', 7250), ('2026-12-14', 'set_reading_allocation', 'E07', 8750)]
  * refused insufficient_funds: [('2026-12-21', 'set_reading_allocation', 'E08', 7750), ('2026-12-28', 'set_reading_allocation', 'E07', 9250), ('2027-01-04', 'set_reading_allocation', 'E07', 9250)]
* stable 41503 greedy nocredit:
  * version 2 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6750), ('2026-10-26', 'set_reading_allocation', 'E08', 5750), ('2026-11-02', 'set_reading_allocation', 'E07', 7250), ('2026-11-09', 'set_reading_allocation', 'E08', 6250), ('2026-11-16', 'set_reading_allocation', 'E07', 7750), ('2026-11-23', 'set_reading_allocation', 'E08', 6750), ('2026-11-30', 'set_reading_allocation', 'E07', 8250), ('2026-12-07', 'set_reading_allocation', 'E08', 7250), ('2026-12-14', 'set_reading_allocation', 'E07', 8750), ('2026-12-21', 'set_reading_allocation', 'E08', 7750), ('2026-12-28', 'set_reading_allocation', 'E07', 9250), ('2027-01-04', 'set_reading_allocation', 'E08', 8250)]
  * version 3 accepted: [('2026-10-19', 'set_reading_allocation', 'E07', 6750), ('2026-10-26', 'set_reading_allocation', 'E08', 5750), ('2026-11-02', 'set_reading_allocation', 'E07', 7250), ('2026-11-09', 'set_reading_allocation', 'E08', 6250), ('2026-11-16', 'set_reading_allocation', 'E07', 7750), ('2026-11-23', 'set_reading_allocation', 'E08', 6750), ('2026-11-30', 'set_reading_allocation', 'E07', 8250), ('2026-12-07', 'set_reading_allocation', 'E08', 7250), ('2026-12-14', 'set_reading_allocation', 'E07', 8750), ('2026-12-21', 'set_reading_allocation', 'E08', 7750), ('2026-12-28', 'set_reading_allocation', 'E07', 9250)]
  * refused insufficient_funds: [('2027-01-04', 'set_reading_allocation', 'E08', 8250)]
