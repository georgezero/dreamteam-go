# v07c-stage-a: suite dreamteam-dev version 3, seeds 41601, 41602, 30 days, engine 0.7.0-dev.3, status complete (50 of 50 runs, 0 failed)

## 1. Starting states (identical across the policies of each entry and seed)

| entry | seed | identical |
|---|---|---|
| financial-pressure-credit | 41601 | yes |
| financial-pressure-credit | 41602 | yes |
| reporting-backlog-credit | 41601 | yes |
| reporting-backlog-credit | 41602 | yes |
| demand-growth-credit | 41601 | yes |
| demand-growth-credit | 41602 | yes |
| staffing-shortage-credit | 41601 | yes |
| staffing-shortage-credit | 41602 | yes |
| stable-credit | 41601 | yes |
| stable-credit | 41602 | yes |

## 2. Outcomes (absolute, each run)

Overtime cost is the harness's (all staff, before benefits); approved is the part from extend_shift. Most drawn is the credit line's largest end-of-day balance; shortfall is the largest unfunded amount (cash below zero with the line fully drawn).

| entry | seed | policy | status | replay | profit | revenue | expenses | TAT p90 | breaches | exams | lost | overtime | overtime cost | approved overtime cost | cash end | min cash | most drawn | interest | shortfall |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41601 | noop | completed | yes | -$16,657 | $236,565 | $253,222 | 837 | 93 | 1,529 | 165 | 3,729 | $6,388 | $0 | $135,518 | $41,873 | $0 | $0 | $0 |
| financial-pressure-credit | 41601 | greedy | completed | yes | -$16,716 | $237,739 | $254,455 | 823 | 92 | 1,529 | 165 | 3,729 | $6,388 | $0 | $135,518 | $41,873 | $0 | $0 | $0 |
| financial-pressure-credit | 41601 | conservative-v2 | completed | yes | -$16,657 | $236,565 | $253,222 | 837 | 93 | 1,529 | 165 | 3,729 | $6,388 | $0 | $135,518 | $41,873 | $0 | $0 | $0 |
| financial-pressure-credit | 41601 | qwen36-35b-a3b | completed | yes | -$32,443 | $236,933 | $269,376 | 797 | 92 | 1,529 | 165 | 7,089 | $19,312 | $12,923 | $125,422 | $31,777 | $0 | $0 | $0 |
| financial-pressure-credit | 41601 | gemma4-26b-a4b | completed | yes | -$16,657 | $236,565 | $253,222 | 837 | 93 | 1,529 | 165 | 3,729 | $6,388 | $0 | $135,518 | $41,873 | $0 | $0 | $0 |
| financial-pressure-credit | 41602 | noop | completed | yes | -$17,598 | $233,115 | $250,713 | 817 | 88 | 1,480 | 182 | 3,125 | $5,200 | $0 | $123,879 | $36,846 | $0 | $0 | $0 |
| financial-pressure-credit | 41602 | greedy | completed | yes | -$17,783 | $234,163 | $251,945 | 806 | 87 | 1,480 | 182 | 3,125 | $5,200 | $0 | $123,879 | $36,846 | $0 | $0 | $0 |
| financial-pressure-credit | 41602 | conservative-v2 | completed | yes | -$17,598 | $233,115 | $250,713 | 817 | 88 | 1,480 | 182 | 3,125 | $5,200 | $0 | $123,879 | $36,846 | $0 | $0 | $0 |
| financial-pressure-credit | 41602 | qwen36-35b-a3b | completed | yes | -$22,281 | $234,771 | $257,052 | 815 | 87 | 1,484 | 181 | 3,416 | $6,004 | $995 | $122,236 | $35,114 | $0 | $0 | $0 |
| financial-pressure-credit | 41602 | gemma4-26b-a4b | completed | yes | -$17,598 | $233,115 | $250,713 | 817 | 88 | 1,480 | 182 | 3,125 | $5,200 | $0 | $123,879 | $36,846 | $0 | $0 | $0 |
| reporting-backlog-credit | 41601 | noop | completed | yes | -$6,395 | $249,722 | $256,117 | 1,031 | 170 | 1,616 | 279 | 4,320 | $6,946 | $0 | $151,761 | $50,450 | $0 | $0 | $0 |
| reporting-backlog-credit | 41601 | greedy | completed | yes | -$6,926 | $250,506 | $257,432 | 984 | 155 | 1,616 | 279 | 4,320 | $6,946 | $0 | $151,761 | $50,450 | $0 | $0 | $0 |
| reporting-backlog-credit | 41601 | conservative-v2 | completed | yes | -$6,395 | $249,722 | $256,117 | 1,031 | 170 | 1,616 | 279 | 4,320 | $6,946 | $0 | $151,761 | $50,450 | $0 | $0 | $0 |
| reporting-backlog-credit | 41601 | qwen36-35b-a3b | completed | yes | -$9,987 | $250,330 | $260,317 | 953 | 109 | 1,617 | 279 | 4,756 | $8,339 | $1,013 | $150,801 | $49,284 | $0 | $0 | $0 |
| reporting-backlog-credit | 41601 | gemma4-26b-a4b | completed | yes | -$7,777 | $249,722 | $257,500 | 988 | 147 | 1,616 | 279 | 4,320 | $6,946 | $0 | $151,761 | $50,450 | $0 | $0 | $0 |
| reporting-backlog-credit | 41602 | noop | completed | yes | -$5,406 | $250,946 | $256,352 | 1,101 | 164 | 1,596 | 294 | 4,138 | $7,065 | $0 | $138,247 | $44,329 | $0 | $0 | $0 |
| reporting-backlog-credit | 41602 | greedy | completed | yes | -$6,792 | $251,450 | $258,242 | 1,047 | 142 | 1,596 | 294 | 4,138 | $7,065 | $0 | $138,247 | $44,329 | $0 | $0 | $0 |
| reporting-backlog-credit | 41602 | conservative-v2 | completed | yes | -$5,406 | $250,946 | $256,352 | 1,101 | 164 | 1,596 | 294 | 4,138 | $7,065 | $0 | $138,247 | $44,329 | $0 | $0 | $0 |
| reporting-backlog-credit | 41602 | qwen36-35b-a3b | completed | yes | -$10,786 | $254,360 | $265,146 | 1,072 | 138 | 1,597 | 291 | 4,706 | $8,459 | $1,637 | $136,343 | $42,425 | $0 | $0 | $0 |
| reporting-backlog-credit | 41602 | gemma4-26b-a4b | completed | yes | -$5,406 | $250,946 | $256,352 | 1,101 | 164 | 1,596 | 294 | 4,138 | $7,065 | $0 | $138,247 | $44,329 | $0 | $0 | $0 |
| demand-growth-credit | 41601 | noop | completed | yes | -$6,261 | $250,080 | $256,341 | 1,036 | 171 | 1,615 | 298 | 4,297 | $7,103 | $0 | $151,761 | $50,450 | $0 | $0 | $0 |
| demand-growth-credit | 41601 | greedy | completed | yes | -$7,209 | $250,447 | $257,656 | 984 | 157 | 1,615 | 298 | 4,297 | $7,103 | $0 | $151,761 | $50,450 | $0 | $0 | $0 |
| demand-growth-credit | 41601 | conservative-v2 | completed | yes | -$6,261 | $250,080 | $256,341 | 1,036 | 171 | 1,615 | 298 | 4,297 | $7,103 | $0 | $151,761 | $50,450 | $0 | $0 | $0 |
| demand-growth-credit | 41601 | qwen36-35b-a3b | completed | yes | -$14,696 | $253,200 | $267,897 | 953 | 128 | 1,621 | 293 | 5,135 | $8,958 | $1,668 | $149,228 | $47,666 | $0 | $0 | $0 |
| demand-growth-credit | 41601 | gemma4-26b-a4b | completed | yes | -$8,997 | $249,762 | $258,759 | 968 | 132 | 1,615 | 298 | 4,297 | $7,103 | $0 | $151,761 | $50,450 | $0 | $0 | $0 |
| demand-growth-credit | 41602 | noop | completed | yes | -$5,018 | $251,295 | $256,314 | 1,105 | 167 | 1,602 | 301 | 4,135 | $7,043 | $0 | $138,247 | $44,329 | $0 | $0 | $0 |
| demand-growth-credit | 41602 | greedy | completed | yes | -$6,010 | $252,194 | $258,204 | 1,046 | 142 | 1,602 | 301 | 4,135 | $7,043 | $0 | $138,247 | $44,329 | $0 | $0 | $0 |
| demand-growth-credit | 41602 | conservative-v2 | completed | yes | -$5,018 | $251,295 | $256,314 | 1,105 | 167 | 1,602 | 301 | 4,135 | $7,043 | $0 | $138,247 | $44,329 | $0 | $0 | $0 |
| demand-growth-credit | 41602 | qwen36-35b-a3b | completed | yes | -$8,215 | $254,518 | $262,733 | 960 | 107 | 1,602 | 301 | 4,135 | $7,043 | $0 | $137,327 | $43,408 | $0 | $0 | $0 |
| demand-growth-credit | 41602 | gemma4-26b-a4b | completed | yes | -$5,018 | $251,295 | $256,314 | 1,105 | 167 | 1,602 | 301 | 4,135 | $7,043 | $0 | $138,247 | $44,329 | $0 | $0 | $0 |
| staffing-shortage-credit | 41601 | noop | completed | yes | -$12,011 | $229,774 | $241,785 | 725 | 79 | 1,526 | 391 | 3,708 | $5,990 | $0 | $142,927 | $46,252 | $0 | $0 | $0 |
| staffing-shortage-credit | 41601 | greedy | completed | yes | -$13,050 | $229,968 | $243,018 | 708 | 77 | 1,526 | 391 | 3,708 | $5,990 | $0 | $142,927 | $46,252 | $0 | $0 | $0 |
| staffing-shortage-credit | 41601 | conservative-v2 | completed | yes | -$12,011 | $229,774 | $241,785 | 725 | 79 | 1,526 | 391 | 3,708 | $5,990 | $0 | $142,927 | $46,252 | $0 | $0 | $0 |
| staffing-shortage-credit | 41601 | qwen36-35b-a3b | completed | yes | -$32,315 | $232,833 | $265,148 | 121 | 46 | 1,537 | 381 | 5,010 | $8,492 | $2,432 | $135,400 | $38,435 | $0 | $0 | $0 |
| staffing-shortage-credit | 41601 | gemma4-26b-a4b | completed | yes | -$12,011 | $229,774 | $241,785 | 725 | 79 | 1,526 | 391 | 3,708 | $5,990 | $0 | $142,927 | $46,252 | $0 | $0 | $0 |
| staffing-shortage-credit | 41602 | noop | completed | yes | -$16,555 | $227,196 | $243,751 | 729 | 77 | 1,518 | 402 | 4,038 | $7,175 | $0 | $130,547 | $43,751 | $0 | $0 | $0 |
| staffing-shortage-credit | 41602 | greedy | completed | yes | -$17,631 | $227,353 | $244,984 | 726 | 76 | 1,518 | 402 | 4,038 | $7,175 | $0 | $130,547 | $43,751 | $0 | $0 | $0 |
| staffing-shortage-credit | 41602 | conservative-v2 | completed | yes | -$16,555 | $227,196 | $243,751 | 729 | 77 | 1,518 | 402 | 4,038 | $7,175 | $0 | $130,547 | $43,751 | $0 | $0 | $0 |
| staffing-shortage-credit | 41602 | qwen36-35b-a3b | completed | yes | -$29,995 | $228,009 | $258,004 | 712 | 75 | 1,520 | 400 | 4,731 | $9,277 | $2,004 | $126,229 | $39,128 | $0 | $0 | $0 |
| staffing-shortage-credit | 41602 | gemma4-26b-a4b | completed | yes | -$18,598 | $229,077 | $247,676 | 132 | 56 | 1,518 | 402 | 4,038 | $7,175 | $0 | $130,547 | $43,751 | $0 | $0 | $0 |
| stable-credit | 41601 | noop | completed | yes | $8,552 | $267,144 | $258,591 | 1,065 | 144 | 1,714 | 211 | 2,986 | $5,204 | $0 | $157,432 | $52,187 | $0 | $0 | $0 |
| stable-credit | 41601 | greedy | completed | yes | $8,400 | $268,306 | $259,906 | 1,033 | 138 | 1,714 | 211 | 2,986 | $5,204 | $0 | $157,432 | $52,187 | $0 | $0 | $0 |
| stable-credit | 41601 | conservative-v2 | completed | yes | $8,552 | $267,144 | $258,591 | 1,065 | 144 | 1,714 | 211 | 2,986 | $5,204 | $0 | $157,432 | $52,187 | $0 | $0 | $0 |
| stable-credit | 41601 | qwen36-35b-a3b | completed | yes | -$848 | $267,518 | $268,366 | 995 | 107 | 1,714 | 211 | 4,784 | $11,663 | $6,580 | $152,896 | $47,555 | $0 | $0 | $0 |
| stable-credit | 41601 | gemma4-26b-a4b | completed | yes | $8,552 | $267,144 | $258,591 | 1,065 | 144 | 1,714 | 211 | 2,986 | $5,204 | $0 | $157,432 | $52,187 | $0 | $0 | $0 |
| stable-credit | 41602 | noop | completed | yes | $1,006 | $257,648 | $256,643 | 843 | 84 | 1,646 | 250 | 2,693 | $4,560 | $0 | $143,376 | $47,059 | $0 | $0 | $0 |
| stable-credit | 41602 | greedy | completed | yes | $975 | $258,851 | $257,876 | 805 | 80 | 1,646 | 250 | 2,693 | $4,560 | $0 | $143,376 | $47,059 | $0 | $0 | $0 |
| stable-credit | 41602 | conservative-v2 | completed | yes | $1,006 | $257,648 | $256,643 | 843 | 84 | 1,646 | 250 | 2,693 | $4,560 | $0 | $143,376 | $47,059 | $0 | $0 | $0 |
| stable-credit | 41602 | qwen36-35b-a3b | completed | yes | -$3,632 | $257,648 | $261,280 | 836 | 85 | 1,646 | 250 | 3,657 | $8,270 | $3,692 | $138,739 | $42,421 | $0 | $0 | $0 |
| stable-credit | 41602 | gemma4-26b-a4b | completed | yes | $1,006 | $257,648 | $256,643 | 843 | 84 | 1,646 | 250 | 2,693 | $4,560 | $0 | $143,376 | $47,059 | $0 | $0 | $0 |

## 3. Paired with no action (policy − noop, same entry and seed; minutes for times)

| entry | seed | policy | Δprofit | Δrevenue | Δexpenses | ΔTAT p90 | Δbreaches | Δexams | Δlost | Δovertime | Δovertime cost | Δcash end | Δmin cash | flag |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41601 | greedy | -$59 | +$1,174 | +$1,233 | -14 | -1 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| financial-pressure-credit | 41601 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| financial-pressure-credit | 41601 | qwen36-35b-a3b | -$15,786 | +$367 | +$16,154 | -41 | -1 | 0 | 0 | +3,360 | +$12,923 | -$10,096 | -$10,096 | better service, lower profit |
| financial-pressure-credit | 41601 | gemma4-26b-a4b | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| financial-pressure-credit | 41602 | greedy | -$185 | +$1,048 | +$1,233 | -12 | -1 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| financial-pressure-credit | 41602 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| financial-pressure-credit | 41602 | qwen36-35b-a3b | -$4,684 | +$1,656 | +$6,340 | -2 | -1 | +4 | -1 | +292 | +$804 | -$1,642 | -$1,732 | better service, lower profit |
| financial-pressure-credit | 41602 | gemma4-26b-a4b | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| reporting-backlog-credit | 41601 | greedy | -$531 | +$784 | +$1,315 | -47 | -15 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| reporting-backlog-credit | 41601 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| reporting-backlog-credit | 41601 | qwen36-35b-a3b | -$3,592 | +$607 | +$4,200 | -78 | -61 | +1 | 0 | +436 | +$1,393 | -$961 | -$1,166 | better service, lower profit |
| reporting-backlog-credit | 41601 | gemma4-26b-a4b | -$1,383 | $0 | +$1,383 | -43 | -23 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| reporting-backlog-credit | 41602 | greedy | -$1,387 | +$504 | +$1,890 | -55 | -22 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| reporting-backlog-credit | 41602 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| reporting-backlog-credit | 41602 | qwen36-35b-a3b | -$5,381 | +$3,414 | +$8,794 | -29 | -26 | +1 | -3 | +568 | +$1,394 | -$1,904 | -$1,904 | better service, lower profit |
| reporting-backlog-credit | 41602 | gemma4-26b-a4b | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| demand-growth-credit | 41601 | greedy | -$948 | +$367 | +$1,315 | -53 | -14 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| demand-growth-credit | 41601 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| demand-growth-credit | 41601 | qwen36-35b-a3b | -$8,435 | +$3,121 | +$11,556 | -83 | -43 | +6 | -5 | +838 | +$1,855 | -$2,534 | -$2,784 | better service, lower profit |
| demand-growth-credit | 41601 | gemma4-26b-a4b | -$2,736 | -$318 | +$2,419 | -68 | -39 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| demand-growth-credit | 41602 | greedy | -$991 | +$899 | +$1,890 | -59 | -25 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| demand-growth-credit | 41602 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| demand-growth-credit | 41602 | qwen36-35b-a3b | -$3,197 | +$3,222 | +$6,419 | -146 | -60 | 0 | 0 | 0 | $0 | -$920 | -$920 | better service, lower profit |
| demand-growth-credit | 41602 | gemma4-26b-a4b | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| staffing-shortage-credit | 41601 | greedy | -$1,038 | +$195 | +$1,233 | -17 | -2 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| staffing-shortage-credit | 41601 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| staffing-shortage-credit | 41601 | qwen36-35b-a3b | -$20,304 | +$3,059 | +$23,363 | -604 | -33 | +11 | -10 | +1,302 | +$2,502 | -$7,527 | -$7,817 | better service, lower profit |
| staffing-shortage-credit | 41601 | gemma4-26b-a4b | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| staffing-shortage-credit | 41602 | greedy | -$1,076 | +$157 | +$1,233 | -3 | -1 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| staffing-shortage-credit | 41602 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| staffing-shortage-credit | 41602 | qwen36-35b-a3b | -$13,440 | +$813 | +$14,253 | -17 | -2 | +2 | -2 | +693 | +$2,102 | -$4,318 | -$4,623 | better service, lower profit |
| staffing-shortage-credit | 41602 | gemma4-26b-a4b | -$2,043 | +$1,881 | +$3,925 | -597 | -21 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| stable-credit | 41601 | greedy | -$152 | +$1,163 | +$1,315 | -31 | -6 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| stable-credit | 41601 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| stable-credit | 41601 | qwen36-35b-a3b | -$9,401 | +$374 | +$9,775 | -69 | -37 | 0 | 0 | +1,798 | +$6,459 | -$4,537 | -$4,632 | better service, lower profit |
| stable-credit | 41601 | gemma4-26b-a4b | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| stable-credit | 41602 | greedy | -$30 | +$1,202 | +$1,233 | -38 | -4 | 0 | 0 | 0 | $0 | $0 | $0 | better service, lower profit |
| stable-credit | 41602 | conservative-v2 | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |
| stable-credit | 41602 | qwen36-35b-a3b | -$4,637 | $0 | +$4,637 | -7 | +1 | 0 | 0 | +965 | +$3,710 | -$4,637 | -$4,637 | better service, lower profit |
| stable-credit | 41602 | gemma4-26b-a4b | $0 | $0 | $0 | 0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |  |

## 4. Actions: proposed, refused, accepted, executed

Proposed counts each action each time it was put forward (each validate attempt; the built-in policies' submissions); `batch_rejected` is counted apart. Executed: accepted and in effect by the end of the window. Funds: refused insufficient_funds (at validate / at apply).

| entry | seed | policy | proposed | refused by validate (+batch) | submitted | accepted | refused at apply (+batch) | executed | executed ops/staffing | funds v/a | accepted by type |
|---|---|---|---|---|---|---|---|---|---|---|---|
| financial-pressure-credit | 41601 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| financial-pressure-credit | 41601 | greedy | 4 | 0 (+0) | 4 | 4 | 0 (+0) | 2 | 0/2 | 0/0 | set_reading_allocation: 4 |
| financial-pressure-credit | 41601 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| financial-pressure-credit | 41601 | qwen36-35b-a3b | 21 | 2 (+0) | 16 | 16 | 0 (+0) | 16 | 16/0 | 0/0 | extend_shift: 16 |
| financial-pressure-credit | 41601 | gemma4-26b-a4b | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| financial-pressure-credit | 41602 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| financial-pressure-credit | 41602 | greedy | 4 | 0 (+0) | 4 | 4 | 0 (+0) | 2 | 0/2 | 0/0 | set_reading_allocation: 4 |
| financial-pressure-credit | 41602 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| financial-pressure-credit | 41602 | qwen36-35b-a3b | 6 | 2 (+0) | 4 | 4 | 0 (+0) | 4 | 3/1 | 0/0 | extend_shift: 3; set_reading_allocation: 1 |
| financial-pressure-credit | 41602 | gemma4-26b-a4b | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| reporting-backlog-credit | 41601 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| reporting-backlog-credit | 41601 | greedy | 4 | 0 (+0) | 4 | 4 | 0 (+0) | 2 | 0/2 | 0/0 | set_reading_allocation: 4 |
| reporting-backlog-credit | 41601 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| reporting-backlog-credit | 41601 | qwen36-35b-a3b | 17 | 6 (+2) | 7 | 7 | 0 (+0) | 7 | 6/1 | 0/0 | contract_teleradiology: 1; extend_shift: 5; set_worklist_policy: 1 |
| reporting-backlog-credit | 41601 | gemma4-26b-a4b | 1 | 0 (+0) | 1 | 1 | 0 (+0) | 1 | 0/1 | 0/0 | contract_teleradiology: 1 |
| reporting-backlog-credit | 41602 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| reporting-backlog-credit | 41602 | greedy | 4 | 0 (+0) | 4 | 4 | 0 (+0) | 2 | 0/2 | 0/0 | set_reading_allocation: 4 |
| reporting-backlog-credit | 41602 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| reporting-backlog-credit | 41602 | qwen36-35b-a3b | 24 | 10 (+1) | 9 | 9 | 0 (+0) | 8 | 6/2 | 0/0 | contract_teleradiology: 2; extend_shift: 6; set_reading_allocation: 1 |
| reporting-backlog-credit | 41602 | gemma4-26b-a4b | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| demand-growth-credit | 41601 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| demand-growth-credit | 41601 | greedy | 4 | 0 (+0) | 4 | 4 | 0 (+0) | 2 | 0/2 | 0/0 | set_reading_allocation: 4 |
| demand-growth-credit | 41601 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| demand-growth-credit | 41601 | qwen36-35b-a3b | 12 | 1 (+0) | 11 | 11 | 0 (+0) | 11 | 10/1 | 0/0 | extend_shift: 9; set_reading_allocation: 1; set_worklist_policy: 1 |
| demand-growth-credit | 41601 | gemma4-26b-a4b | 9 | 4 (+2) | 3 | 3 | 0 (+0) | 1 | 0/1 | 0/0 | contract_teleradiology: 3 |
| demand-growth-credit | 41602 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| demand-growth-credit | 41602 | greedy | 4 | 0 (+0) | 4 | 4 | 0 (+0) | 2 | 0/2 | 0/0 | set_reading_allocation: 4 |
| demand-growth-credit | 41602 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| demand-growth-credit | 41602 | qwen36-35b-a3b | 9 | 5 (+1) | 1 | 1 | 0 (+0) | 1 | 0/1 | 0/0 | contract_teleradiology: 1 |
| demand-growth-credit | 41602 | gemma4-26b-a4b | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| staffing-shortage-credit | 41601 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| staffing-shortage-credit | 41601 | greedy | 4 | 0 (+0) | 4 | 4 | 0 (+0) | 2 | 0/2 | 0/0 | set_reading_allocation: 4 |
| staffing-shortage-credit | 41601 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| staffing-shortage-credit | 41601 | qwen36-35b-a3b | 22 | 4 (+0) | 16 | 16 | 0 (+0) | 16 | 12/4 | 0/0 | contract_teleradiology: 1; extend_shift: 11; hire: 1; set_reading_allocation: 2; set_worklist_policy: 1 |
| staffing-shortage-credit | 41601 | gemma4-26b-a4b | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| staffing-shortage-credit | 41602 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| staffing-shortage-credit | 41602 | greedy | 4 | 0 (+0) | 4 | 4 | 0 (+0) | 2 | 0/2 | 0/0 | set_reading_allocation: 4 |
| staffing-shortage-credit | 41602 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| staffing-shortage-credit | 41602 | qwen36-35b-a3b | 28 | 7 (+0) | 15 | 15 | 0 (+0) | 14 | 11/3 | 0/0 | contract_teleradiology: 1; extend_shift: 10; hire: 1; set_reading_allocation: 2; set_worklist_policy: 1 |
| staffing-shortage-credit | 41602 | gemma4-26b-a4b | 4 | 3 (+0) | 1 | 1 | 0 (+0) | 1 | 0/1 | 0/0 | contract_teleradiology: 1 |
| stable-credit | 41601 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| stable-credit | 41601 | greedy | 4 | 0 (+0) | 4 | 4 | 0 (+0) | 2 | 0/2 | 0/0 | set_reading_allocation: 4 |
| stable-credit | 41601 | conservative-v2 | 1 | 0 (+0) | 1 | 1 | 0 (+0) | 0 | 0/0 | 0/0 | set_reading_allocation: 1 |
| stable-credit | 41601 | qwen36-35b-a3b | 23 | 4 (+0) | 19 | 19 | 0 (+0) | 19 | 18/1 | 0/0 | contract_teleradiology: 1; extend_shift: 17; set_worklist_policy: 1 |
| stable-credit | 41601 | gemma4-26b-a4b | 1 | 0 (+0) | 1 | 1 | 0 (+0) | 0 | 0/0 | 0/0 | contract_teleradiology: 1 |
| stable-credit | 41602 | noop | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| stable-credit | 41602 | greedy | 4 | 0 (+0) | 4 | 4 | 0 (+0) | 2 | 0/2 | 0/0 | set_reading_allocation: 4 |
| stable-credit | 41602 | conservative-v2 | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |
| stable-credit | 41602 | qwen36-35b-a3b | 14 | 3 (+0) | 10 | 10 | 0 (+0) | 9 | 9/0 | 0/0 | contract_teleradiology: 1; extend_shift: 8; set_worklist_policy: 1 |
| stable-credit | 41602 | gemma4-26b-a4b | 0 | 0 (+0) | 0 | 0 | 0 (+0) | 0 | 0/0 | 0/0 | none |

## 5. Refused for funds, and what followed

Each action refused insufficient_funds; later in the run, the decision points at which the policy put forward the same type for the same target, how many of those were refused for funds again, and how many were accepted.

None: no action was refused insufficient_funds.

## 6. LLM decisions

| entry | seed | policy | points | with actions | no action | fallback | first-attempt valid | repeated invalid | details | infrastructure errors |
|---|---|---|---|---|---|---|---|---|---|---|
| demand-growth-credit | 41601 | gemma4-26b-a4b | 30 | 2 | 25 | 3 | 25/30 | 6 | 0 | 0 |
| demand-growth-credit | 41601 | qwen36-35b-a3b | 30 | 4 | 26 | 0 | 29/30 | 0 | 0 | 0 |
| demand-growth-credit | 41602 | gemma4-26b-a4b | 30 | 0 | 30 | 0 | 30/30 | 0 | 0 | 0 |
| demand-growth-credit | 41602 | qwen36-35b-a3b | 30 | 1 | 26 | 3 | 24/30 | 2 | 0 | 0 |
| financial-pressure-credit | 41601 | gemma4-26b-a4b | 30 | 0 | 30 | 0 | 30/30 | 0 | 0 | 0 |
| financial-pressure-credit | 41601 | qwen36-35b-a3b | 30 | 16 | 13 | 1 | 29/30 | 0 | 0 | 0 |
| financial-pressure-credit | 41602 | gemma4-26b-a4b | 30 | 0 | 30 | 0 | 30/30 | 0 | 0 | 0 |
| financial-pressure-credit | 41602 | qwen36-35b-a3b | 30 | 3 | 27 | 0 | 28/30 | 0 | 0 | 0 |
| reporting-backlog-credit | 41601 | gemma4-26b-a4b | 30 | 1 | 29 | 0 | 30/30 | 0 | 0 | 0 |
| reporting-backlog-credit | 41601 | qwen36-35b-a3b | 30 | 4 | 25 | 1 | 23/30 | 5 | 0 | 0 |
| reporting-backlog-credit | 41602 | gemma4-26b-a4b | 30 | 0 | 30 | 0 | 30/30 | 0 | 0 | 0 |
| reporting-backlog-credit | 41602 | qwen36-35b-a3b | 30 | 6 | 20 | 4 | 16/30 | 15 | 0 | 0 |
| stable-credit | 41601 | gemma4-26b-a4b | 30 | 1 | 29 | 0 | 30/30 | 0 | 0 | 0 |
| stable-credit | 41601 | qwen36-35b-a3b | 30 | 15 | 14 | 1 | 22/30 | 5 | 0 | 0 |
| stable-credit | 41602 | gemma4-26b-a4b | 30 | 0 | 30 | 0 | 30/30 | 0 | 0 | 0 |
| stable-credit | 41602 | qwen36-35b-a3b | 30 | 9 | 20 | 1 | 25/30 | 2 | 0 | 0 |
| staffing-shortage-credit | 41601 | gemma4-26b-a4b | 30 | 0 | 30 | 0 | 30/30 | 0 | 0 | 0 |
| staffing-shortage-credit | 41601 | qwen36-35b-a3b | 30 | 9 | 20 | 1 | 23/30 | 4 | 0 | 0 |
| staffing-shortage-credit | 41602 | gemma4-26b-a4b | 30 | 1 | 23 | 6 | 24/30 | 3 | 0 | 0 |
| staffing-shortage-credit | 41602 | qwen36-35b-a3b | 30 | 10 | 17 | 3 | 23/30 | 7 | 0 | 0 |

## 7. Refusals by validate, and agent errors

| entry | seed | policy | refusals (type code: count) | errors (code: count) |
|---|---|---|---|---|
| demand-growth-credit | 41601 | gemma4-26b-a4b | contract_teleradiology batch_rejected: 3; contract_teleradiology contract_overlap: 5; contract_teleradiology invalid_schema: 1; contract_teleradiology notice_too_short: 1; extend_shift no_coverage: 1 | refused: 8 |
| demand-growth-credit | 41601 | qwen36-35b-a3b | extend_shift no_coverage: 1 | refused: 1 |
| demand-growth-credit | 41602 | gemma4-26b-a4b | none | none |
| demand-growth-credit | 41602 | qwen36-35b-a3b | contract_teleradiology batch_rejected: 1; contract_teleradiology invalid_schema: 1; extend_shift after_close: 4; extend_shift exceeds_max_shift: 5; hire batch_rejected: 1; hire lead_too_short: 1 | invalid_decision: 1; invalid_json: 2; refused: 6 |
| financial-pressure-credit | 41601 | gemma4-26b-a4b | none | none |
| financial-pressure-credit | 41601 | qwen36-35b-a3b | extend_shift after_close: 3; extend_shift exceeds_max_shift: 2 | refused: 2 |
| financial-pressure-credit | 41602 | gemma4-26b-a4b | none | none |
| financial-pressure-credit | 41602 | qwen36-35b-a3b | extend_shift after_close: 1; extend_shift no_coverage: 1 | refused: 2 |
| reporting-backlog-credit | 41601 | gemma4-26b-a4b | none | none |
| reporting-backlog-credit | 41601 | qwen36-35b-a3b | contract_teleradiology batch_rejected: 2; contract_teleradiology contract_overlap: 1; contract_teleradiology notice_too_short: 3; contract_teleradiology past_date: 1; extend_shift after_close: 4; extend_shift no_coverage: 1 | refused: 8 |
| reporting-backlog-credit | 41602 | gemma4-26b-a4b | none | none |
| reporting-backlog-credit | 41602 | qwen36-35b-a3b | contract_teleradiology batch_rejected: 3; contract_teleradiology contract_overlap: 4; contract_teleradiology invalid_schema: 1; contract_teleradiology notice_too_short: 1; extend_shift after_close: 14; extend_shift invalid_minutes: 2; extend_shift no_coverage: 1; set_reading_allocation past_date: 1 | invalid_decision: 3; refused: 15 |
| stable-credit | 41601 | gemma4-26b-a4b | none | none |
| stable-credit | 41601 | qwen36-35b-a3b | contract_teleradiology notice_too_short: 1; extend_shift after_close: 6; extend_shift no_coverage: 1; extend_shift not_on_shift: 1 | refused: 9 |
| stable-credit | 41602 | gemma4-26b-a4b | none | none |
| stable-credit | 41602 | qwen36-35b-a3b | extend_shift after_close: 3; extend_shift no_coverage: 1; extend_shift not_on_shift: 1 | invalid_decision: 1; refused: 5 |
| staffing-shortage-credit | 41601 | gemma4-26b-a4b | none | none |
| staffing-shortage-credit | 41601 | qwen36-35b-a3b | contract_teleradiology invalid_schema: 1; contract_teleradiology past_date: 1; extend_shift after_close: 4; extend_shift exceeds_max_shift: 2; extend_shift not_on_shift: 2; hire lead_too_short: 1; set_reading_allocation batch_rejected: 1 | refused: 8 |
| staffing-shortage-credit | 41602 | gemma4-26b-a4b | hire invalid_roster: 3; hire lead_too_short: 2 | invalid_json: 7; refused: 5 |
| staffing-shortage-credit | 41602 | qwen36-35b-a3b | extend_shift after_close: 5; extend_shift no_coverage: 1; extend_shift not_on_shift: 4; hire lead_too_short: 1 | invalid_decision: 2; refused: 8 |

## 8. LLM actions in time order, with cash and credit

Every action an LLM agent submitted, with the outcome, and the cash and credit drawn at the end of the day before; and the agent's recorded reason at each point where an action was refused for funds.


### demand-growth-credit 41601 gemma4-26b-a4b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-18 | contract_teleradiology effective=2026-11-01, op=start, plan=standard-v1, requestId=telerad-sun-oct2… | accepted | yes | $118,383 | $0 |
| 2026-11-06 | contract_teleradiology contract=T00001, endDate=2026-12-06, op=cancel | accepted | no | $102,858 | $0 |
| 2026-11-06 | contract_teleradiology effective=2026-12-07, op=start, plan=standard-v1, requestId=telerad-weeknigh… | accepted | no | $102,858 | $0 |

### demand-growth-credit 41601 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-14 | extend_shift date=2026-10-14, employee=E07, minutes=120 | accepted | yes | $157,816 | $0 |
| 2026-10-14 | extend_shift date=2026-10-14, employee=E03, minutes=60 | accepted | yes | $157,816 | $0 |
| 2026-10-14 | extend_shift date=2026-10-14, employee=E05, minutes=60 | accepted | yes | $157,816 | $0 |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E07, minutes=120 | accepted | yes | $163,406 | $0 |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E03, minutes=60 | accepted | yes | $163,406 | $0 |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E05, minutes=60 | accepted | yes | $163,406 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E07, minutes=120 | accepted | yes | $169,256 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E03, minutes=60 | accepted | yes | $169,256 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E05, minutes=60 | accepted | yes | $169,256 | $0 |
| 2026-10-19 | set_worklist_policy policy=oldest_first | accepted | yes | $116,936 | $0 |
| 2026-10-19 | set_reading_allocation effective=2026-11-02, employee=E08, op=set, siteAllocationBP=10000 | accepted | yes | $116,936 | $0 |

### demand-growth-credit 41602 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-17 | contract_teleradiology effective=2026-10-31, op=start, plan=standard-v1, requestId=tr-2026-10-17-00… | accepted | yes | $125,702 | $0 |

### financial-pressure-credit 41601 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-15 | extend_shift date=2026-10-15, employee=E07, minutes=210 | accepted | yes | $163,754 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E07, minutes=210 | accepted | yes | $168,765 | $0 |
| 2026-10-19 | extend_shift date=2026-10-19, employee=E07, minutes=210 | accepted | yes | $116,734 | $0 |
| 2026-10-20 | extend_shift date=2026-10-20, employee=E07, minutes=210 | accepted | yes | $140,071 | $0 |
| 2026-10-21 | extend_shift date=2026-10-21, employee=E07, minutes=210 | accepted | yes | $148,036 | $0 |
| 2026-10-22 | extend_shift date=2026-10-22, employee=E07, minutes=210 | accepted | yes | $154,555 | $0 |
| 2026-10-23 | extend_shift date=2026-10-23, employee=E07, minutes=210 | accepted | yes | $161,677 | $0 |
| 2026-10-26 | extend_shift date=2026-10-26, employee=E07, minutes=210 | accepted | yes | $171,073 | $0 |
| 2026-10-28 | extend_shift date=2026-10-28, employee=E07, minutes=210 | accepted | yes | $202,429 | $0 |
| 2026-10-29 | extend_shift date=2026-10-29, employee=E07, minutes=210 | accepted | yes | $209,228 | $0 |
| 2026-10-30 | extend_shift date=2026-10-30, employee=E07, minutes=210 | accepted | yes | $215,627 | $0 |
| 2026-11-02 | extend_shift date=2026-11-02, employee=E07, minutes=210 | accepted | yes | $31,777 | $0 |
| 2026-11-04 | extend_shift date=2026-11-04, employee=E07, minutes=210 | accepted | yes | $65,418 | $0 |
| 2026-11-05 | extend_shift date=2026-11-05, employee=E07, minutes=210 | accepted | yes | $72,687 | $0 |
| 2026-11-09 | extend_shift date=2026-11-09, employee=E07, minutes=210 | accepted | yes | $84,741 | $0 |
| 2026-11-11 | extend_shift date=2026-11-11, employee=E07, minutes=210 | accepted | yes | $118,427 | $0 |

### financial-pressure-credit 41602 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-14 | extend_shift date=2026-10-14, employee=E03, minutes=120 | accepted | yes | $157,963 | $0 |
| 2026-10-14 | set_reading_allocation effective=2026-10-28, employee=E07, op=set, siteAllocationBP=8000 | accepted | yes | $157,963 | $0 |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E07, minutes=120 | accepted | yes | $166,991 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E07, minutes=120 | accepted | yes | $174,093 | $0 |

### reporting-backlog-credit 41601 gemma4-26b-a4b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-25 | contract_teleradiology effective=2026-11-08, op=start, plan=standard-v1, requestId=telerad-backlog-… | accepted | yes | $175,461 | $0 |

### reporting-backlog-credit 41601 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-14 | extend_shift date=2026-10-14, employee=E07, minutes=120 | accepted | yes | $157,816 | $0 |
| 2026-10-14 | extend_shift date=2026-10-14, employee=E03, minutes=60 | accepted | yes | $157,816 | $0 |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E07, minutes=60 | accepted | yes | $163,406 | $0 |
| 2026-10-18 | set_worklist_policy policy=oldest_first | accepted | yes | $117,371 | $0 |
| 2026-10-18 | contract_teleradiology effective=2026-11-01, op=start, plan=standard-v1, requestId=tr-weekend-backl… | accepted | yes | $117,371 | $0 |
| 2026-11-06 | extend_shift date=2026-11-06, employee=E07, minutes=60 | accepted | yes | $101,692 | $0 |
| 2026-11-06 | extend_shift date=2026-11-06, employee=E03, minutes=60 | accepted | yes | $101,692 | $0 |

### reporting-backlog-credit 41602 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-13 | extend_shift date=2026-10-13, employee=E07, minutes=120 | accepted | yes | – | – |
| 2026-10-13 | extend_shift date=2026-10-13, employee=E03, minutes=120 | accepted | yes | – | – |
| 2026-10-13 | extend_shift date=2026-10-13, employee=E05, minutes=120 | accepted | yes | – | – |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E07, minutes=120 | accepted | yes | $168,903 | $0 |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E05, minutes=120 | accepted | yes | $168,903 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E07, minutes=120 | accepted | yes | $176,805 | $0 |
| 2026-10-19 | contract_teleradiology effective=2026-11-02, op=start, plan=standard-v1, requestId=tr-2026-10-19-al… | accepted | yes | $124,214 | $0 |
| 2026-10-23 | set_reading_allocation effective=2026-11-06, employee=E08, op=set, siteAllocationBP=8000 | accepted | yes | $169,090 | $0 |
| 2026-11-11 | contract_teleradiology contract=T00001, endDate=2026-12-11, op=cancel | accepted | no | $128,560 | $0 |

### stable-credit 41601 gemma4-26b-a4b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-11-01 | contract_teleradiology effective=2026-11-15, op=start, plan=standard-v1, requestId=telerad-sun-nov-… | accepted | no | $183,208 | $0 |

### stable-credit 41601 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-13 | extend_shift date=2026-10-13, employee=E07, minutes=120 | accepted | yes | – | – |
| 2026-10-13 | extend_shift date=2026-10-13, employee=E03, minutes=60 | accepted | yes | – | – |
| 2026-10-13 | extend_shift date=2026-10-13, employee=E09, minutes=60 | accepted | yes | – | – |
| 2026-10-13 | extend_shift date=2026-10-13, employee=E05, minutes=60 | accepted | yes | – | – |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E07, minutes=120 | accepted | yes | $165,055 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E07, minutes=120 | accepted | yes | $170,695 | $0 |
| 2026-10-18 | set_worklist_policy policy=oldest_first | accepted | yes | $118,291 | $0 |
| 2026-10-18 | contract_teleradiology effective=2026-11-01, op=start, plan=standard-v1, requestId=tr-sunday-backlo… | accepted | yes | $118,291 | $0 |
| 2026-10-19 | extend_shift date=2026-10-19, employee=E07, minutes=120 | accepted | yes | $118,291 | $0 |
| 2026-10-26 | extend_shift date=2026-10-26, employee=E07, minutes=120 | accepted | yes | $178,699 | $0 |
| 2026-10-27 | extend_shift date=2026-10-27, employee=E07, minutes=120 | accepted | yes | $204,889 | $0 |
| 2026-10-28 | extend_shift date=2026-10-28, employee=E07, minutes=120 | accepted | yes | $213,360 | $0 |
| 2026-10-29 | extend_shift date=2026-10-29, employee=E07, minutes=120 | accepted | yes | $222,267 | $0 |
| 2026-10-30 | extend_shift date=2026-10-30, employee=E07, minutes=120 | accepted | yes | $230,913 | $0 |
| 2026-11-03 | extend_shift date=2026-11-03, employee=E07, minutes=120 | accepted | yes | $74,761 | $0 |
| 2026-11-04 | extend_shift date=2026-11-04, employee=E07, minutes=120 | accepted | yes | $81,788 | $0 |
| 2026-11-05 | extend_shift date=2026-11-05, employee=E07, minutes=120 | accepted | yes | $91,693 | $0 |
| 2026-11-06 | extend_shift date=2026-11-06, employee=E07, minutes=120 | accepted | yes | $101,484 | $0 |
| 2026-11-10 | extend_shift date=2026-11-10, employee=E07, minutes=120 | accepted | yes | $135,166 | $0 |

### stable-credit 41602 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-13 | extend_shift date=2026-10-13, employee=E07, minutes=120 | accepted | yes | – | – |
| 2026-10-14 | extend_shift date=2026-10-14, employee=E07, minutes=120 | accepted | yes | $159,182 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E07, minutes=120 | accepted | yes | $176,295 | $0 |
| 2026-10-19 | extend_shift date=2026-10-19, employee=E07, minutes=120 | accepted | yes | $123,860 | $0 |
| 2026-10-20 | extend_shift date=2026-10-20, employee=E07, minutes=120 | accepted | yes | $146,651 | $0 |
| 2026-10-21 | extend_shift date=2026-10-21, employee=E07, minutes=120 | accepted | yes | $152,911 | $0 |
| 2026-10-23 | extend_shift date=2026-10-23, employee=E07, minutes=120 | accepted | yes | $170,091 | $0 |
| 2026-10-26 | extend_shift date=2026-10-26, employee=E07, minutes=120 | accepted | yes | $181,011 | $0 |
| 2026-10-31 | set_worklist_policy policy=oldest_first | accepted | yes | $174,047 | $0 |
| 2026-10-31 | contract_teleradiology effective=2026-11-14, op=start, plan=standard-v1, requestId=tr-2026-10-31-sa… | accepted | no | $174,047 | $0 |

### staffing-shortage-credit 41601 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-13 | extend_shift date=2026-10-13, employee=E03, minutes=120 | accepted | yes | – | – |
| 2026-10-13 | extend_shift date=2026-10-13, employee=E05, minutes=90 | accepted | yes | – | – |
| 2026-10-13 | extend_shift date=2026-10-13, employee=E06, minutes=60 | accepted | yes | – | – |
| 2026-10-14 | extend_shift date=2026-10-14, employee=E05, minutes=120 | accepted | yes | $156,336 | $0 |
| 2026-10-14 | set_reading_allocation effective=2026-10-28, employee=E08, op=set, siteAllocationBP=10000 | accepted | yes | $156,336 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E03, minutes=120 | accepted | yes | $167,219 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E06, minutes=120 | accepted | yes | $167,219 | $0 |
| 2026-10-16 | set_worklist_policy policy=oldest_first | accepted | yes | $167,219 | $0 |
| 2026-10-18 | contract_teleradiology effective=2026-11-01, op=start, plan=standard-v1, requestId=tr-2026-11-01-00… | accepted | yes | $118,318 | $0 |
| 2026-10-18 | set_reading_allocation effective=2026-11-01, employee=E08, op=set, siteAllocationBP=10000 | unchanged | yes | $118,318 | $0 |
| 2026-10-20 | extend_shift date=2026-10-20, employee=E03, minutes=120 | accepted | yes | $138,390 | $0 |
| 2026-10-20 | hire contractedWeeklyMinutes=2400, rateCents=4400, requestId=hire-ct-premium-2026-11-10-001, role=t… | accepted | yes | $138,390 | $0 |
| 2026-10-21 | extend_shift date=2026-10-21, employee=E07, minutes=120 | accepted | yes | $146,580 | $0 |
| 2026-10-22 | extend_shift date=2026-10-22, employee=E07, minutes=120 | accepted | yes | $154,172 | $0 |
| 2026-10-23 | extend_shift date=2026-10-23, employee=E07, minutes=120 | accepted | yes | $162,368 | $0 |
| 2026-10-26 | extend_shift date=2026-10-26, employee=E07, minutes=120 | accepted | yes | $170,768 | $0 |

### staffing-shortage-credit 41602 gemma4-26b-a4b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-20 | contract_teleradiology effective=2026-11-03, op=start, plan=standard-v1, requestId=telerad-mri-fix,… | accepted | yes | $147,095 | $0 |

### staffing-shortage-credit 41602 qwen36-35b-a3b

| date | action | outcome | executed | cash day before | drawn day before |
|---|---|---|---|---|---|
| 2026-10-13 | extend_shift date=2026-10-13, employee=E05, minutes=60 | accepted | yes | – | – |
| 2026-10-13 | hire contractedWeeklyMinutes=2400, rateCents=4900, requestId=hire-mr-tech-urgent-002, role=technolo… | accepted | yes | – | – |
| 2026-10-15 | extend_shift date=2026-10-15, employee=E05, minutes=60 | accepted | yes | $165,403 | $0 |
| 2026-10-15 | set_reading_allocation effective=2026-10-29, employee=E07, op=set, siteAllocationBP=7000 | accepted | yes | $165,403 | $0 |
| 2026-10-15 | set_reading_allocation effective=2026-10-29, employee=E08, op=set, siteAllocationBP=6000 | accepted | yes | $165,403 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E07, minutes=120 | accepted | yes | $171,953 | $0 |
| 2026-10-16 | extend_shift date=2026-10-16, employee=E05, minutes=60 | accepted | yes | $171,953 | $0 |
| 2026-10-29 | extend_shift date=2026-10-29, employee=E07, minutes=60 | accepted | yes | $210,123 | $0 |
| 2026-11-01 | set_worklist_policy policy=oldest_first | accepted | yes | $169,350 | $0 |
| 2026-11-01 | contract_teleradiology effective=2026-11-15, op=start, plan=standard-v1, requestId=telerad-sunday-b… | accepted | no | $169,350 | $0 |
| 2026-11-04 | extend_shift date=2026-11-04, employee=E07, minutes=60 | accepted | yes | $68,458 | $0 |
| 2026-11-06 | extend_shift date=2026-11-06, employee=E07, minutes=60 | accepted | yes | $84,218 | $0 |
| 2026-11-09 | extend_shift date=2026-11-09, employee=E07, minutes=60 | accepted | yes | $89,851 | $0 |
| 2026-11-10 | extend_shift date=2026-11-10, employee=E07, minutes=60 | accepted | yes | $111,317 | $0 |
| 2026-11-11 | extend_shift date=2026-11-11, employee=E07, minutes=60 | accepted | yes | $117,364 | $0 |

## 9. Tokens, latency and cost

| entry | seed | policy | requests | retries | prompt tokens | completion tokens | estimated/unknown | latency mean s | max s | total s | estimated cost |
|---|---|---|---|---|---|---|---|---|---|---|---|
| demand-growth-credit | 41601 | gemma4-26b-a4b | 35 | 5 | 429,243 | 3,366 | 0/0 | 6.7 | 9.5 | 234 | none configured |
| demand-growth-credit | 41601 | qwen36-35b-a3b | 31 | 1 | 358,490 | 4,086 | 0/0 | 3.3 | 4.9 | 102 | none configured |
| demand-growth-credit | 41602 | gemma4-26b-a4b | 30 | 0 | 319,030 | 2,122 | 0/0 | 6.4 | 6.9 | 191 | none configured |
| demand-growth-credit | 41602 | qwen36-35b-a3b | 36 | 6 | 436,153 | 16,864 | 0/0 | 6.4 | 32.2 | 230 | none configured |
| financial-pressure-credit | 41601 | gemma4-26b-a4b | 30 | 0 | 317,869 | 2,063 | 0/0 | 4.9 | 5.5 | 148 | none configured |
| financial-pressure-credit | 41601 | qwen36-35b-a3b | 31 | 1 | 356,629 | 4,557 | 0/0 | 3.3 | 4.5 | 102 | none configured |
| financial-pressure-credit | 41602 | gemma4-26b-a4b | 30 | 0 | 317,078 | 2,045 | 0/0 | 5.2 | 6.0 | 157 | none configured |
| financial-pressure-credit | 41602 | qwen36-35b-a3b | 32 | 2 | 363,514 | 4,020 | 0/0 | 3.2 | 4.3 | 102 | none configured |
| reporting-backlog-credit | 41601 | gemma4-26b-a4b | 30 | 0 | 343,868 | 2,065 | 0/0 | 6.4 | 7.3 | 193 | none configured |
| reporting-backlog-credit | 41601 | qwen36-35b-a3b | 37 | 7 | 465,310 | 6,390 | 0/0 | 3.9 | 8.2 | 145 | none configured |
| reporting-backlog-credit | 41602 | gemma4-26b-a4b | 30 | 0 | 318,824 | 2,076 | 0/0 | 6.5 | 7.1 | 196 | none configured |
| reporting-backlog-credit | 41602 | qwen36-35b-a3b | 44 | 14 | 568,626 | 10,819 | 0/0 | 4.3 | 11.6 | 189 | none configured |
| stable-credit | 41601 | gemma4-26b-a4b | 30 | 0 | 342,402 | 2,208 | 0/0 | 8.0 | 10.9 | 239 | none configured |
| stable-credit | 41601 | qwen36-35b-a3b | 38 | 8 | 493,963 | 6,335 | 0/0 | 3.5 | 5.9 | 135 | none configured |
| stable-credit | 41602 | gemma4-26b-a4b | 30 | 0 | 325,607 | 2,054 | 0/0 | 6.9 | 7.8 | 208 | none configured |
| stable-credit | 41602 | qwen36-35b-a3b | 35 | 5 | 423,873 | 5,593 | 0/0 | 3.9 | 6.8 | 137 | none configured |
| staffing-shortage-credit | 41601 | gemma4-26b-a4b | 30 | 0 | 309,173 | 2,200 | 0/0 | 7.1 | 10.1 | 213 | none configured |
| staffing-shortage-credit | 41601 | qwen36-35b-a3b | 37 | 7 | 484,617 | 8,401 | 0/0 | 4.0 | 8.2 | 149 | none configured |
| staffing-shortage-credit | 41602 | gemma4-26b-a4b | 36 | 6 | 413,951 | 6,403 | 0/0 | 7.7 | 12.1 | 275 | none configured |
| staffing-shortage-credit | 41602 | qwen36-35b-a3b | 37 | 7 | 461,318 | 9,501 | 0/0 | 4.6 | 9.5 | 169 | none configured |
