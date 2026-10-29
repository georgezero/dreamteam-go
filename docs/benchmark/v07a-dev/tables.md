# V0.7a characterization: v07a-characterize-v1 (suite dreamteam-dev 1) and v07a-characterize-v2 (suite dreamteam-dev 2)

Seeds 41401, 41402, 41403, 90 days. Version 1: complete (45 of 45); version 2: complete (45 of 45). Engines 0.7.0-dev.1, 0.7.0-dev.1.

## 1. Starting states

| entry | seed | v1 start | v2 start | identical |
|---|---|---|---|---|
| financial-pressure | 41401 | 992f0c2c7b32… | e4e23fca967e… | no |
| financial-pressure | 41402 | 89b9765daaf0… | 1028365b8cf8… | no |
| financial-pressure | 41403 | 84871d24651f… | 211b28d32cdc… | no |
| reporting-backlog | 41401 | 8c327fb0e83c… | dff1dbc11f5f… | no |
| reporting-backlog | 41402 | a3fd277bbeec… | 382ccab68a6a… | no |
| reporting-backlog | 41403 | 332a7fba3902… | 63a5e4bdb76d… | no |
| demand-growth | 41401 | 8c327fb0e83c… | dff1dbc11f5f… | no |
| demand-growth | 41402 | a3fd277bbeec… | 382ccab68a6a… | no |
| demand-growth | 41403 | 332a7fba3902… | 63a5e4bdb76d… | no |
| staffing-shortage | 41401 | 93aeabf7a3fe… | 47b8cafdf04a… | no |
| staffing-shortage | 41402 | ae3fc3e04e77… | a51f4e392cf4… | no |
| staffing-shortage | 41403 | 266f1448f673… | 56de649419d7… | no |
| stable | 41401 | 81130a6d7809… | d6d82bc8cb67… | no |
| stable | 41402 | e6f7ec90424f… | 52f458af5cf3… | no |
| stable | 41403 | ddbecd67215e… | 3124bf8a96e2… | no |

## 2. Each run, version 1 and version 2

| entry | seed | policy | status v1/v2 | replay | profit v1 | profit v2 | radiologist OT h v1 | radiologist OT h v2 | accepted v1 | accepted v2 | T90 v1 | T90 v2 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure | 41401 | noop | completed/completed | yes | -$36,907 | -$51,214 | 50 | 50 | 0 | 0 | not applicable | not applicable |
| financial-pressure | 41401 | greedy | completed/completed | yes | -$69,454 | -$83,761 | 50 | 50 | 12 | 12 | not applicable | not applicable |
| financial-pressure | 41401 | conservative-v2 | completed/completed | yes | -$36,907 | -$51,214 | 50 | 50 | 0 | 0 | not applicable | not applicable |
| financial-pressure | 41402 | noop | completed/completed | yes | -$42,109 | -$53,403 | 39 | 39 | 0 | 0 | not applicable | not applicable |
| financial-pressure | 41402 | greedy | completed/completed | yes | -$74,657 | -$85,951 | 39 | 39 | 12 | 12 | not applicable | not applicable |
| financial-pressure | 41402 | conservative-v2 | completed/completed | yes | -$42,109 | -$53,403 | 39 | 39 | 0 | 0 | not applicable | not applicable |
| financial-pressure | 41403 | noop | completed/completed | yes | -$40,751 | -$52,514 | 41 | 41 | 0 | 0 | not applicable | not applicable |
| financial-pressure | 41403 | greedy | completed/completed | yes | -$73,299 | -$85,062 | 41 | 41 | 12 | 12 | not applicable | not applicable |
| financial-pressure | 41403 | conservative-v2 | completed/completed | yes | -$40,751 | -$52,514 | 41 | 41 | 0 | 0 | not applicable | not applicable |
| reporting-backlog | 41401 | noop | completed/completed | yes | $2,598 | -$16,358 | 66 | 66 | 0 | 0 | NO GO | not applicable |
| reporting-backlog | 41401 | greedy | completed/completed | yes | -$29,724 | -$48,679 | 66 | 66 | 12 | 12 | NO GO | not applicable |
| reporting-backlog | 41401 | conservative-v2 | completed/completed | yes | $3,069 | -$16,358 | 66 | 66 | 1 | 0 | NO GO | not applicable |
| reporting-backlog | 41402 | noop | completed/completed | yes | -$6,924 | -$24,534 | 61 | 61 | 0 | 0 | NO GO | not applicable |
| reporting-backlog | 41402 | greedy | completed/completed | yes | -$38,327 | -$55,937 | 61 | 61 | 12 | 12 | NO GO | not applicable |
| reporting-backlog | 41402 | conservative-v2 | completed/completed | yes | -$9,226 | -$24,534 | 61 | 61 | 2 | 0 | NO GO | not applicable |
| reporting-backlog | 41403 | noop | completed/completed | yes | -$3,006 | -$22,175 | 66 | 66 | 0 | 0 | NO GO | not applicable |
| reporting-backlog | 41403 | greedy | completed/completed | yes | -$32,622 | -$51,791 | 66 | 66 | 12 | 12 | NO GO | not applicable |
| reporting-backlog | 41403 | conservative-v2 | completed/completed | yes | -$3,006 | -$22,175 | 66 | 66 | 0 | 0 | NO GO | not applicable |
| demand-growth | 41401 | noop | completed/completed | yes | $13,761 | -$5,039 | 65 | 65 | 0 | 0 | not applicable | not applicable |
| demand-growth | 41401 | greedy | completed/completed | yes | -$15,670 | -$34,470 | 65 | 65 | 12 | 12 | not applicable | not applicable |
| demand-growth | 41401 | conservative-v2 | completed/completed | yes | $14,668 | -$5,039 | 65 | 65 | 1 | 1 | not applicable | not applicable |
| demand-growth | 41402 | noop | completed/completed | yes | $2,130 | -$16,014 | 63 | 63 | 0 | 0 | not applicable | not applicable |
| demand-growth | 41402 | greedy | completed/completed | yes | -$27,325 | -$45,470 | 63 | 63 | 12 | 12 | not applicable | not applicable |
| demand-growth | 41402 | conservative-v2 | completed/completed | yes | -$179 | -$16,014 | 63 | 63 | 2 | 0 | not applicable | not applicable |
| demand-growth | 41403 | noop | completed/completed | yes | $8,723 | -$11,066 | 69 | 69 | 0 | 0 | not applicable | not applicable |
| demand-growth | 41403 | greedy | completed/completed | yes | -$21,801 | -$41,589 | 69 | 69 | 12 | 12 | not applicable | not applicable |
| demand-growth | 41403 | conservative-v2 | completed/completed | yes | $8,723 | -$11,066 | 69 | 69 | 0 | 0 | not applicable | not applicable |
| staffing-shortage | 41401 | noop | completed/completed | yes | -$18,508 | -$38,188 | 68 | 68 | 0 | 0 | not applicable | not applicable |
| staffing-shortage | 41401 | greedy | completed/completed | yes | -$51,057 | -$70,737 | 68 | 68 | 12 | 12 | not applicable | not applicable |
| staffing-shortage | 41401 | conservative-v2 | completed/completed | yes | -$18,508 | -$38,188 | 68 | 68 | 0 | 0 | not applicable | not applicable |
| staffing-shortage | 41402 | noop | completed/completed | yes | -$33,834 | -$50,625 | 58 | 58 | 0 | 0 | not applicable | not applicable |
| staffing-shortage | 41402 | greedy | completed/completed | yes | -$66,382 | -$83,173 | 58 | 58 | 12 | 12 | not applicable | not applicable |
| staffing-shortage | 41402 | conservative-v2 | completed/completed | yes | -$33,834 | -$50,625 | 58 | 58 | 0 | 0 | not applicable | not applicable |
| staffing-shortage | 41403 | noop | completed/completed | yes | -$28,478 | -$46,724 | 63 | 63 | 0 | 0 | not applicable | not applicable |
| staffing-shortage | 41403 | greedy | completed/completed | yes | -$61,073 | -$79,319 | 63 | 63 | 12 | 12 | not applicable | not applicable |
| staffing-shortage | 41403 | conservative-v2 | completed/completed | yes | -$28,478 | -$46,724 | 63 | 63 | 0 | 0 | not applicable | not applicable |
| stable | 41401 | noop | completed/completed | yes | $26,593 | $10,626 | 55 | 55 | 0 | 0 | not applicable | not applicable |
| stable | 41401 | greedy | completed/completed | yes | -$6,457 | -$22,423 | 55 | 55 | 12 | 12 | not applicable | not applicable |
| stable | 41401 | conservative-v2 | completed/completed | yes | $18,694 | $8,324 | 55 | 55 | 3 | 2 | not applicable | not applicable |
| stable | 41402 | noop | completed/completed | yes | $16,349 | $4,676 | 40 | 40 | 0 | 0 | not applicable | not applicable |
| stable | 41402 | greedy | completed/completed | yes | -$16,200 | -$27,872 | 40 | 40 | 12 | 12 | not applicable | not applicable |
| stable | 41402 | conservative-v2 | completed/completed | yes | $8,297 | $2,374 | 40 | 40 | 3 | 2 | not applicable | not applicable |
| stable | 41403 | noop | completed/completed | yes | $17,397 | $4,264 | 46 | 46 | 0 | 0 | not applicable | not applicable |
| stable | 41403 | greedy | completed/completed | yes | -$15,051 | -$28,184 | 46 | 46 | 12 | 12 | not applicable | not applicable |
| stable | 41403 | conservative-v2 | completed/completed | yes | $15,096 | $4,264 | 46 | 46 | 2 | 0 | not applicable | not applicable |

## 3. Version 2 minus version 1 (same entry, seed and policy)

Overtime cost is the harness's overtime cost (all staff, before benefits); its change is the radiologist overtime charge where operations are unchanged. Actions: whether the accepted actions (and when they were decided) are identical.

| entry | seed | policy | actions same | Δ overtime cost | Δ expense | Δ profit | Δ revenue | Δ cash end | Δ min cash | Δ TAT p90 min | Δ breaches | Δ exams | Δ lost | Δ radiologist OT min |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure | 41401 | noop | yes | +$11,445 | +$14,307 | -$14,307 | $0 | -$14,469 | -$10,116 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41401 | greedy | yes | +$11,445 | +$14,307 | -$14,307 | $0 | -$14,469 | -$14,469 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41401 | conservative-v2 | yes | +$11,445 | +$14,307 | -$14,307 | $0 | -$14,469 | -$10,116 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41402 | noop | yes | +$9,035 | +$11,294 | -$11,294 | $0 | -$12,268 | -$10,001 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41402 | greedy | yes | +$9,035 | +$11,294 | -$11,294 | $0 | -$12,268 | -$12,268 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41402 | conservative-v2 | yes | +$9,035 | +$11,294 | -$11,294 | $0 | -$12,268 | -$10,001 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41403 | noop | yes | +$9,411 | +$11,763 | -$11,763 | $0 | -$12,862 | -$10,269 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41403 | greedy | yes | +$9,411 | +$11,763 | -$11,763 | $0 | -$12,862 | -$12,862 | 0 | 0 | 0 | 0 | 0 |
| financial-pressure | 41403 | conservative-v2 | yes | +$9,411 | +$11,763 | -$11,763 | $0 | -$12,862 | -$10,269 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41401 | noop | yes | +$15,164 | +$18,955 | -$18,955 | $0 | -$20,198 | -$5,450 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41401 | greedy | yes | +$15,164 | +$18,955 | -$18,955 | $0 | -$20,198 | -$16,892 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41401 | conservative-v2 | **no** | +$15,164 | +$17,969 | -$19,427 | -$1,458 | -$19,541 | -$5,450 | +17 | +11 | 0 | 0 | 0 |
| reporting-backlog | 41402 | noop | yes | +$14,088 | +$17,610 | -$17,610 | $0 | -$19,313 | -$5,604 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41402 | greedy | yes | +$14,088 | +$17,610 | -$17,610 | $0 | -$19,313 | -$16,554 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41402 | conservative-v2 | **no** | +$14,088 | +$15,306 | -$15,307 | -$1 | -$17,057 | -$5,604 | +11 | +19 | 0 | 0 | 0 |
| reporting-backlog | 41403 | noop | yes | +$15,335 | +$19,169 | -$19,169 | $0 | -$20,517 | -$5,818 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41403 | greedy | yes | +$15,335 | +$19,169 | -$19,169 | $0 | -$20,517 | -$20,517 | 0 | 0 | 0 | 0 | 0 |
| reporting-backlog | 41403 | conservative-v2 | yes | +$15,335 | +$19,169 | -$19,169 | $0 | -$20,517 | -$5,818 | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41401 | noop | yes | +$15,040 | +$18,800 | -$18,800 | $0 | -$19,719 | -$5,450 | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41401 | greedy | yes | +$15,040 | +$18,800 | -$18,800 | $0 | -$19,719 | -$15,865 | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41401 | conservative-v2 | **no** | +$15,040 | +$17,731 | -$19,706 | -$1,975 | -$18,979 | -$5,450 | +48 | +13 | 0 | 0 | 0 |
| demand-growth | 41402 | noop | yes | +$14,515 | +$18,144 | -$18,144 | $0 | -$19,873 | -$5,604 | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41402 | greedy | yes | +$14,515 | +$18,144 | -$18,144 | $0 | -$19,873 | -$16,194 | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41402 | conservative-v2 | **no** | +$14,515 | +$15,830 | -$15,835 | -$5 | -$17,824 | -$5,604 | +39 | +40 | 0 | 0 | 0 |
| demand-growth | 41403 | noop | yes | +$15,831 | +$19,788 | -$19,788 | $0 | -$20,911 | -$5,818 | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41403 | greedy | yes | +$15,831 | +$19,788 | -$19,788 | $0 | -$20,911 | -$20,911 | 0 | 0 | 0 | 0 | 0 |
| demand-growth | 41403 | conservative-v2 | yes | +$15,831 | +$19,788 | -$19,788 | $0 | -$20,911 | -$5,818 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41401 | noop | yes | +$15,744 | +$19,680 | -$19,680 | $0 | -$21,130 | -$6,434 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41401 | greedy | yes | +$15,744 | +$19,680 | -$19,680 | $0 | -$21,130 | -$21,130 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41401 | conservative-v2 | yes | +$15,744 | +$19,680 | -$19,680 | $0 | -$21,130 | -$6,434 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41402 | noop | yes | +$13,433 | +$16,791 | -$16,791 | $0 | -$18,219 | -$10,131 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41402 | greedy | yes | +$13,433 | +$16,791 | -$16,791 | $0 | -$18,219 | -$18,219 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41402 | conservative-v2 | yes | +$13,433 | +$16,791 | -$16,791 | $0 | -$18,219 | -$10,131 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41403 | noop | yes | +$14,597 | +$18,246 | -$18,246 | $0 | -$19,249 | -$16,010 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41403 | greedy | yes | +$14,597 | +$18,246 | -$18,246 | $0 | -$19,249 | -$19,249 | 0 | 0 | 0 | 0 | 0 |
| staffing-shortage | 41403 | conservative-v2 | yes | +$14,597 | +$18,246 | -$18,246 | $0 | -$19,249 | -$16,010 | 0 | 0 | 0 | 0 | 0 |
| stable | 41401 | noop | yes | +$12,773 | +$15,967 | -$15,967 | $0 | -$17,116 | -$4,130 | 0 | 0 | 0 | 0 | 0 |
| stable | 41401 | greedy | yes | +$12,773 | +$15,967 | -$15,967 | $0 | -$17,116 | -$4,130 | 0 | 0 | 0 | 0 | 0 |
| stable | 41401 | conservative-v2 | **no** | +$12,773 | +$10,213 | -$10,370 | -$157 | -$11,691 | -$4,130 | +70 | +17 | 0 | 0 | 0 |
| stable | 41402 | noop | yes | +$9,338 | +$11,673 | -$11,673 | $0 | -$13,337 | -$5,114 | 0 | 0 | 0 | 0 | 0 |
| stable | 41402 | greedy | yes | +$9,338 | +$11,673 | -$11,673 | $0 | -$13,337 | -$5,114 | 0 | 0 | 0 | 0 | 0 |
| stable | 41402 | conservative-v2 | **no** | +$9,338 | +$5,906 | -$5,923 | -$16 | -$8,179 | -$5,114 | +74 | +30 | 0 | 0 | 0 |
| stable | 41403 | noop | yes | +$10,506 | +$13,133 | -$13,133 | $0 | -$14,593 | -$4,756 | 0 | 0 | 0 | 0 | 0 |
| stable | 41403 | greedy | yes | +$10,506 | +$13,133 | -$13,133 | $0 | -$14,593 | -$4,756 | 0 | 0 | 0 | 0 | 0 |
| stable | 41403 | conservative-v2 | **no** | +$10,506 | +$10,793 | -$10,831 | -$38 | -$13,048 | -$4,756 | +37 | +14 | 0 | 0 | 0 |

## 4. Paired with no action, each version (policy − noop, same entry and seed)

| entry | seed | policy | Δ profit v1 | Δ breaches v1 | Δ profit v2 | Δ breaches v2 |
|---|---|---|---|---|---|---|
| financial-pressure | 41401 | greedy | -$32,547 | -43 | -$32,547 | -43 |
| financial-pressure | 41401 | conservative-v2 | $0 | 0 | $0 | 0 |
| financial-pressure | 41402 | greedy | -$32,548 | -34 | -$32,548 | -34 |
| financial-pressure | 41402 | conservative-v2 | $0 | 0 | $0 | 0 |
| financial-pressure | 41403 | greedy | -$32,548 | -35 | -$32,548 | -35 |
| financial-pressure | 41403 | conservative-v2 | $0 | 0 | $0 | 0 |
| reporting-backlog | 41401 | greedy | -$32,321 | -105 | -$32,321 | -105 |
| reporting-backlog | 41401 | conservative-v2 | +$471 | -11 | $0 | 0 |
| reporting-backlog | 41402 | greedy | -$31,403 | -115 | -$31,403 | -115 |
| reporting-backlog | 41402 | conservative-v2 | -$2,303 | -19 | $0 | 0 |
| reporting-backlog | 41403 | greedy | -$29,616 | -90 | -$29,616 | -90 |
| reporting-backlog | 41403 | conservative-v2 | $0 | 0 | $0 | 0 |
| demand-growth | 41401 | greedy | -$29,431 | -210 | -$29,431 | -210 |
| demand-growth | 41401 | conservative-v2 | +$907 | -13 | $0 | 0 |
| demand-growth | 41402 | greedy | -$29,455 | -169 | -$29,455 | -169 |
| demand-growth | 41402 | conservative-v2 | -$2,309 | -40 | $0 | 0 |
| demand-growth | 41403 | greedy | -$30,524 | -223 | -$30,524 | -223 |
| demand-growth | 41403 | conservative-v2 | $0 | 0 | $0 | 0 |
| staffing-shortage | 41401 | greedy | -$32,549 | -27 | -$32,549 | -27 |
| staffing-shortage | 41401 | conservative-v2 | $0 | 0 | $0 | 0 |
| staffing-shortage | 41402 | greedy | -$32,548 | -21 | -$32,548 | -21 |
| staffing-shortage | 41402 | conservative-v2 | $0 | 0 | $0 | 0 |
| staffing-shortage | 41403 | greedy | -$32,595 | -19 | -$32,595 | -19 |
| staffing-shortage | 41403 | conservative-v2 | $0 | 0 | $0 | 0 |
| stable | 41401 | greedy | -$33,049 | -46 | -$33,049 | -46 |
| stable | 41401 | conservative-v2 | -$7,898 | -29 | -$2,301 | -12 |
| stable | 41402 | greedy | -$32,548 | -49 | -$32,548 | -49 |
| stable | 41402 | conservative-v2 | -$8,052 | -35 | -$2,301 | -5 |
| stable | 41403 | greedy | -$32,448 | -52 | -$32,448 | -52 |
| stable | 41403 | conservative-v2 | -$2,301 | -14 | $0 | 0 |

## 5. Where the accepted actions differ

* reporting-backlog 41401 conservative-v2:
  * version 1: [('2026-12-17', 'set_reading_allocation', 5500)]
  * version 2: []
* reporting-backlog 41402 conservative-v2:
  * version 1: [('2026-11-19', 'set_reading_allocation', 5500), ('2026-12-17', 'set_reading_allocation', 5000)]
  * version 2: []
* demand-growth 41401 conservative-v2:
  * version 1: [('2026-12-16', 'set_reading_allocation', 5500)]
  * version 2: [('2026-12-31', 'set_reading_allocation', 5500)]
* demand-growth 41402 conservative-v2:
  * version 1: [('2026-11-19', 'set_reading_allocation', 5500), ('2026-12-17', 'set_reading_allocation', 5000)]
  * version 2: []
* stable 41401 conservative-v2:
  * version 1: [('2026-10-20', 'set_reading_allocation', 5750), ('2026-11-17', 'set_reading_allocation', 6250), ('2026-12-15', 'set_reading_allocation', 5750)]
  * version 2: [('2026-11-04', 'set_reading_allocation', 5750), ('2026-12-02', 'set_reading_allocation', 5250)]
* stable 41402 conservative-v2:
  * version 1: [('2026-10-20', 'set_reading_allocation', 5750), ('2026-11-17', 'set_reading_allocation', 6250), ('2026-12-15', 'set_reading_allocation', 5750)]
  * version 2: [('2026-11-18', 'set_reading_allocation', 5750), ('2026-12-16', 'set_reading_allocation', 5250)]
* stable 41403 conservative-v2:
  * version 1: [('2026-10-27', 'set_reading_allocation', 5750), ('2026-11-24', 'set_reading_allocation', 5250)]
  * version 2: []
