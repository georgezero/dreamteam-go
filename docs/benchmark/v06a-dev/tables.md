## Runs

| scenario | seed | policy | status | replay verified | actions match | fingerprint | T90 |
|---|---|---|---|---|---|---|---|
| financial-pressure | 40901 | noop | completed | yes | yes | 3dcd153bb24c… | not_applicable |
| financial-pressure | 40901 | greedy | completed | yes | yes | 3dcd153bb24c… | not_applicable |
| financial-pressure | 40901 | rules | completed | yes | yes | 3dcd153bb24c… | not_applicable |
| financial-pressure | 40902 | noop | completed | yes | yes | 3dcd153bb24c… | not_applicable |
| financial-pressure | 40902 | greedy | completed | yes | yes | 3dcd153bb24c… | not_applicable |
| financial-pressure | 40902 | rules | completed | yes | yes | 3dcd153bb24c… | not_applicable |
| financial-pressure | 40903 | noop | completed | yes | yes | 3dcd153bb24c… | not_applicable |
| financial-pressure | 40903 | greedy | completed | yes | yes | 3dcd153bb24c… | not_applicable |
| financial-pressure | 40903 | rules | completed | yes | yes | 3dcd153bb24c… | not_applicable |
| reporting-backlog | 40901 | noop | completed | yes | yes | e8718dbd1031… | NO GO |
| reporting-backlog | 40901 | greedy | completed | yes | yes | e8718dbd1031… | NO GO |
| reporting-backlog | 40901 | rules | completed | yes | yes | e8718dbd1031… | NO GO |
| reporting-backlog | 40902 | noop | completed | yes | yes | e8718dbd1031… | NO GO |
| reporting-backlog | 40902 | greedy | completed | yes | yes | e8718dbd1031… | NO GO |
| reporting-backlog | 40902 | rules | completed | yes | yes | e8718dbd1031… | NO GO |
| reporting-backlog | 40903 | noop | completed | yes | yes | e8718dbd1031… | NO GO |
| reporting-backlog | 40903 | greedy | completed | yes | yes | e8718dbd1031… | NO GO |
| reporting-backlog | 40903 | rules | completed | yes | yes | e8718dbd1031… | NO GO |
| demand-growth | 40901 | noop | completed | yes | yes | 0146ac340d95… | not_applicable |
| demand-growth | 40901 | greedy | completed | yes | yes | 0146ac340d95… | not_applicable |
| demand-growth | 40901 | rules | completed | yes | yes | 0146ac340d95… | not_applicable |
| demand-growth | 40902 | noop | completed | yes | yes | 0146ac340d95… | not_applicable |
| demand-growth | 40902 | greedy | completed | yes | yes | 0146ac340d95… | not_applicable |
| demand-growth | 40902 | rules | completed | yes | yes | 0146ac340d95… | not_applicable |
| demand-growth | 40903 | noop | completed | yes | yes | 0146ac340d95… | not_applicable |
| demand-growth | 40903 | greedy | completed | yes | yes | 0146ac340d95… | not_applicable |
| demand-growth | 40903 | rules | completed | yes | yes | 0146ac340d95… | not_applicable |
| staffing-shortage | 40901 | noop | completed | yes | yes | 6ef37e929b8d… | not_applicable |
| staffing-shortage | 40901 | greedy | completed | yes | yes | 6ef37e929b8d… | not_applicable |
| staffing-shortage | 40901 | rules | completed | yes | yes | 6ef37e929b8d… | not_applicable |
| staffing-shortage | 40902 | noop | completed | yes | yes | 6ef37e929b8d… | not_applicable |
| staffing-shortage | 40902 | greedy | completed | yes | yes | 6ef37e929b8d… | not_applicable |
| staffing-shortage | 40902 | rules | completed | yes | yes | 6ef37e929b8d… | not_applicable |
| staffing-shortage | 40903 | noop | completed | yes | yes | 6ef37e929b8d… | not_applicable |
| staffing-shortage | 40903 | greedy | completed | yes | yes | 6ef37e929b8d… | not_applicable |
| staffing-shortage | 40903 | rules | completed | yes | yes | 6ef37e929b8d… | not_applicable |
| stable | 40901 | noop | completed | yes | yes | d65113896ce5… | not_applicable |
| stable | 40901 | greedy | completed | yes | yes | d65113896ce5… | not_applicable |
| stable | 40901 | rules | completed | yes | yes | d65113896ce5… | not_applicable |
| stable | 40902 | noop | completed | yes | yes | d65113896ce5… | not_applicable |
| stable | 40902 | greedy | completed | yes | yes | d65113896ce5… | not_applicable |
| stable | 40902 | rules | completed | yes | yes | d65113896ce5… | not_applicable |
| stable | 40903 | noop | completed | yes | yes | d65113896ce5… | not_applicable |
| stable | 40903 | greedy | completed | yes | yes | d65113896ce5… | not_applicable |
| stable | 40903 | rules | completed | yes | yes | d65113896ce5… | not_applicable |

**Starting states** (the harness's initial conditions: every policy's run of a scenario and seed starts from the same state):

| scenario | seed | runs | identical | state digest |
|---|---|---|---|---|
| bench-financial-pressure | 40901 | 3 | yes | 7bb5203c04fc7415… |
| bench-financial-pressure | 40902 | 3 | yes | a91f6c381f429527… |
| bench-financial-pressure | 40903 | 3 | yes | 1814f874baa5b924… |
| sustainable | 40901 | 3 | yes | a8685ac046c9706a… |
| sustainable | 40902 | 3 | yes | 6986b590bc89a0b3… |
| sustainable | 40903 | 3 | yes | b15c41a2ed308a9a… |
| bench-demand-growth | 40901 | 3 | yes | a8685ac046c9706a… |
| bench-demand-growth | 40902 | 3 | yes | 6986b590bc89a0b3… |
| bench-demand-growth | 40903 | 3 | yes | b15c41a2ed308a9a… |
| bench-staffing-shortage | 40901 | 3 | yes | 171dd6eb99bb5e23… |
| bench-staffing-shortage | 40902 | 3 | yes | 4eaa2b6be2d25588… |
| bench-staffing-shortage | 40903 | 3 | yes | 47fc4c6dbcadc0bf… |
| bench-stable | 40901 | 3 | yes | e2e4befab21d7f8f… |
| bench-stable | 40902 | 3 | yes | c630bf48b6d3c6f5… |
| bench-stable | 40903 | 3 | yes | 9131166b63bc7318… |

Scenario fingerprints all differ: yes.
Scenarios sharing a starting state on a seed: demand-growth and reporting-backlog (seed 40901); demand-growth and reporting-backlog (seed 40902); demand-growth and reporting-backlog (seed 40903) (expected: demand growth differs from reporting backlog only from November).

## No-action profile (minimum / median / maximum over the seeds)


**Operational performance**

| scenario | examsCompleted | reportsFinal | tatP50Sec | tatP90Sec | backlogEnd | waitToExamP50Sec | waitToExamP90Sec |
|---|---|---|---|---|---|---|---|
| financial-pressure | 4,247 / 4,301 / 4,341 | 4,237 / 4,293 / 4,335 | 61 min / 63 min / 66 min | 847 min / 855 min / 873 min | 9 / 10 / 11 | 19 min / 19 min / 20 min | 60 min / 61 min / 63 min |
| reporting-backlog | 4,538 / 4,654 / 4,660 | 4,536 / 4,645 / 4,647 | 108 min / 120 min / 135 min | 973 min / 1,015 min / 1,112 min | 11 / 15 / 15 | 20 min / 21 min / 21 min | 68 min / 68 min / 70 min |
| demand-growth | 4,683 / 4,744 / 4,784 | 4,656 / 4,733 / 4,761 | 143 min / 149 min / 168 min | 1,112 min / 1,159 min / 1,176 min | 13 / 25 / 40 | 21 min / 21 min / 21 min | 70 min / 71 min / 71 min |
| staffing-shortage | 4,327 / 4,406 / 4,409 | 4,317 / 4,395 / 4,401 | 41 min / 41 min / 45 min | 723 min / 741 min / 771 min | 8 / 10 / 11 | 18 min / 19 min / 19 min | 65 min / 67 min / 68 min |
| stable | 4,714 / 4,805 / 4,874 | 4,718 / 4,819 / 4,879 | 106 min / 114 min / 135 min | 909 min / 930 min / 987 min | 8 / 10 / 11 | 19 min / 19 min / 19 min | 52 min / 56 min / 57 min |

**Financial performance**

| scenario | netRevenueCents | operatingExpenseCents | operatingProfitCents | margin | operatingCashFlowCents |
|---|---|---|---|---|---|
| financial-pressure | $689,108 / $690,040 / $697,646 | $721,915 / $722,570 / $723,730 | -$33,690 / -$32,807 / -$24,924 | -4.9% / -4.8% / -3.6% | -$107,923 / -$91,296 / -$82,320 |
| reporting-backlog | $738,448 / $738,530 / $750,165 | $732,410 / $733,728 / $733,908 | $4,622 / $6,037 / $16,437 | 0.6% / 0.8% / 2.2% | -$70,228 / -$52,700 / -$52,037 |
| demand-growth | $745,736 / $748,467 / $755,500 | $735,135 / $735,647 / $735,731 | $10,601 / $12,736 / $19,852 | 1.4% / 1.7% / 2.6% | -$74,264 / -$54,694 / -$51,411 |
| staffing-shortage | $662,760 / $665,541 / $675,972 | $694,051 / $694,543 / $695,373 | -$31,784 / -$28,511 / -$19,401 | -4.8% / -4.3% / -2.9% | -$93,210 / -$77,165 / -$71,544 |
| stable | $758,786 / $764,416 / $777,477 | $741,259 / $743,480 / $743,784 | $17,526 / $20,936 / $33,694 | 2.3% / 2.7% / 4.3% | -$53,249 / -$34,049 / -$32,786 |

**Staffing and capacity**

| scenario | ordersReceived | ordersLost | lostRate | overtimeSec | heldOver | skippedBreaks | noCoverage |
|---|---|---|---|---|---|---|---|
| financial-pressure | 4,915 / 4,933 / 4,953 | 486 / 546 / 549 | 9.8% / 11.1% / 11.2% | 8,389 min / 8,501 min / 9,500 min | 103 / 106 / 124 | 0 / 0 / 0 | 0 / 0 / 0 |
| reporting-backlog | 5,548 / 5,582 / 5,589 | 734 / 808 / 814 | 13.1% / 14.6% / 14.6% | 11,550 min / 11,890 min / 12,461 min | 134 / 135 / 144 | 0 / 0 / 0 | 0 / 0 / 2 |
| demand-growth | 6,111 / 6,123 / 6,166 | 963 / 1,038 / 1,045 | 15.6% / 17.0% / 17.1% | 12,745 min / 12,799 min / 13,345 min | 144 / 147 / 152 | 0 / 0 / 0 | 0 / 0 / 0 |
| staffing-shortage | 5,548 / 5,582 / 5,589 | 1,040 / 1,076 / 1,123 | 18.6% / 19.4% / 20.1% | 10,866 min / 11,033 min / 11,212 min | 129 / 133 / 137 | 0 / 0 / 0 | 0 / 0 / 0 |
| stable | 5,548 / 5,582 / 5,589 | 594 / 656 / 684 | 10.6% / 11.8% / 12.3% | 8,201 min / 8,359 min / 9,349 min | 118 / 122 / 130 | 0 / 0 / 0 | 0 / 0 / 0 |

**Service reliability**

| scenario | tatOverTarget | tatBreaches | waitsOver30 | displacedAppointments | alertsRaisedAlert | alertsActiveAtEnd |
|---|---|---|---|---|---|---|
| financial-pressure | 302 / 303 / 307 | 293 / 295 / 298 | 1,279 / 1,304 / 1,377 | – | 269 / 280 / 292 | 1 / 2 / 2 |
| reporting-backlog | 407 / 495 / 528 | 393 / 481 / 519 | 1,491 / 1,542 / 1,619 | – | 304 / 306 / 310 | 2 / 2 / 2 |
| demand-growth | 547 / 575 / 597 | 507 / 552 / 585 | 1,574 / 1,608 / 1,708 | – | 309 / 321 / 332 | 2 / 2 / 2 |
| staffing-shortage | 238 / 248 / 282 | 230 / 239 / 273 | 1,282 / 1,302 / 1,326 | – | 281 / 288 / 288 | 1 / 2 / 2 |
| stable | 320 / 340 / 400 | 312 / 331 / 391 | 1,391 / 1,435 / 1,438 | – | 226 / 235 / 257 | 1 / 2 / 2 |

**Long-term sustainability**

| scenario | minCashCents | shortfallCents | daysBelowZero |
|---|---|---|---|
| financial-pressure | $34,689 / $42,191 / $44,612 | $0 / $0 / $0 | 0 / 0 / 0 |
| reporting-backlog | $48,260 / $52,171 / $52,560 | $0 / $0 / $0 | 0 / 0 / 0 |
| demand-growth | $48,260 / $52,171 / $52,560 | $0 / $0 / $0 | 0 / 0 / 0 |
| staffing-shortage | $42,928 / $52,860 / $56,402 | $0 / $0 / $0 | 0 / 0 / 0 |
| stable | $48,172 / $52,020 / $55,572 | $0 / $0 / $0 | 0 / 0 / 0 |

**Actions**

| scenario | actionsAttempted | actionsAccepted | actionsRejected |
|---|---|---|---|
| financial-pressure | 0 / 0 / 0 | 0 / 0 / 0 | 0 / 0 / 0 |
| reporting-backlog | 0 / 0 / 0 | 0 / 0 / 0 | 0 / 0 / 0 |
| demand-growth | 0 / 0 / 0 | 0 / 0 / 0 | 0 / 0 / 0 |
| staffing-shortage | 0 / 0 / 0 | 0 / 0 / 0 | 0 / 0 / 0 |
| stable | 0 / 0 / 0 | 0 / 0 / 0 | 0 / 0 / 0 |

## Intended conditions (no action, each seed; PROTOCOL.md §2.3)

| seed | scenario | condition | values |
|---|---|---|---|
| 40901 | financial-pressure | holds | profit -$33,690 |
| 40901 | reporting-backlog | holds | TAT p90 1,015 min, breaches 10.4% of final reports |
| 40901 | demand-growth | holds | orders per 30 days 1,956, 2,138, 2,072; received 6,166 vs 5,589 |
| 40901 | staffing-shortage | holds | lost 1,040 vs 734 |
| 40901 | stable | holds | profit $20,936; lost 594 vs 734; TAT p90 909 min vs 1,015 min |
| 40902 | financial-pressure | holds | profit -$32,807 |
| 40902 | reporting-backlog | holds | TAT p90 973 min, breaches 8.7% of final reports |
| 40902 | demand-growth | DOES NOT HOLD | orders per 30 days 2,029, 2,094, 2,000; received 6,123 vs 5,548 |
| 40902 | staffing-shortage | holds | lost 1,076 vs 808 |
| 40902 | stable | holds | profit $17,526; lost 684 vs 808; TAT p90 930 min vs 973 min |
| 40903 | financial-pressure | holds | profit -$24,924 |
| 40903 | reporting-backlog | holds | TAT p90 1,112 min, breaches 11.2% of final reports |
| 40903 | demand-growth | DOES NOT HOLD | orders per 30 days 2,035, 2,128, 1,948; received 6,111 vs 5,582 |
| 40903 | staffing-shortage | holds | lost 1,123 vs 814 |
| 40903 | stable | holds | profit $33,694; lost 656 vs 814; TAT p90 987 min vs 1,112 min |

## Distinctness: primary metrics whose no-action ranges do not overlap

| scenario | scenario | separated | metrics |
|---|---|---|---|
| financial-pressure | reporting-backlog | 6 | operatingProfitCents, operatingExpenseCents, minCashCents, tatP90Sec, tatBreaches, tatOverTarget |
| financial-pressure | demand-growth | 7 | operatingProfitCents, operatingExpenseCents, minCashCents, ordersLost, lostRate, examsCompleted, waitToExamP90Sec |
| financial-pressure | staffing-shortage | 4 | operatingExpenseCents, ordersLost, overtimeSec, waitToExamP90Sec |
| financial-pressure | stable | 5 | operatingProfitCents, operatingExpenseCents, minCashCents, tatP90Sec, ordersLost |
| reporting-backlog | demand-growth | 5 | tatP90Sec, tatOverTarget, ordersLost, lostRate, examsCompleted |
| reporting-backlog | staffing-shortage | 5 | tatP90Sec, tatBreaches, tatOverTarget, ordersLost, overtimeSec |
| reporting-backlog | stable | 4 | tatBreaches, tatOverTarget, operatingProfitCents, ordersLost |
| demand-growth | staffing-shortage | 4 | lostRate, examsCompleted, waitToExamP90Sec, overtimeSec |
| demand-growth | stable | 4 | ordersLost, lostRate, waitToExamP90Sec, tatP90Sec |
| staffing-shortage | stable | 5 | ordersLost, overtimeSec, waitToExamP90Sec, operatingProfitCents, tatP90Sec |

## Baselines paired with no action (policy − noop, same scenario and seed)


**Operational performance**

| scenario | seed | policy | examsCompleted | reportsFinal | tatP50Sec | tatP90Sec | backlogEnd | waitToExamP50Sec | waitToExamP90Sec |
|---|---|---|---|---|---|---|---|---|---|
| financial-pressure | 40901 | greedy | 0 | 0 | -30 min | -117 min | 0 | 0 min | 0 min |
| financial-pressure | 40901 | rules | 0 | 0 | -31 min | -135 min | 0 | 0 min | 0 min |
| financial-pressure | 40902 | greedy | 0 | 0 | -32 min | -108 min | 0 | 0 min | 0 min |
| financial-pressure | 40902 | rules | 0 | 0 | -32 min | -129 min | 0 | 0 min | 0 min |
| financial-pressure | 40903 | greedy | 0 | 0 | -33 min | -111 min | 0 | 0 min | 0 min |
| financial-pressure | 40903 | rules | 0 | 0 | -34 min | -129 min | 0 | 0 min | 0 min |
| reporting-backlog | 40901 | greedy | 0 | +7 | -80 min | -193 min | -7 | 0 min | 0 min |
| reporting-backlog | 40901 | rules | 0 | +7 | -81 min | -197 min | -7 | 0 min | 0 min |
| reporting-backlog | 40902 | greedy | 0 | +5 | -69 min | -184 min | -5 | 0 min | 0 min |
| reporting-backlog | 40902 | rules | 0 | +5 | -69 min | -188 min | -5 | 0 min | 0 min |
| reporting-backlog | 40903 | greedy | 0 | 0 | -87 min | -221 min | 0 | 0 min | 0 min |
| reporting-backlog | 40903 | rules | 0 | 0 | -90 min | -237 min | 0 | 0 min | 0 min |
| demand-growth | 40901 | greedy | 0 | +16 | -101 min | -280 min | -16 | 0 min | 0 min |
| demand-growth | 40901 | rules | 0 | +16 | -102 min | -285 min | -16 | 0 min | 0 min |
| demand-growth | 40902 | greedy | 0 | +30 | -108 min | -374 min | -30 | 0 min | 0 min |
| demand-growth | 40902 | rules | 0 | +30 | -109 min | -378 min | -30 | 0 min | 0 min |
| demand-growth | 40903 | greedy | 0 | +2 | -120 min | -286 min | -2 | 0 min | 0 min |
| demand-growth | 40903 | rules | 0 | +2 | -122 min | -297 min | -2 | 0 min | 0 min |
| staffing-shortage | 40901 | greedy | 0 | 0 | -15 min | -70 min | 0 | 0 min | 0 min |
| staffing-shortage | 40901 | rules | 0 | 0 | -15 min | -114 min | 0 | 0 min | 0 min |
| staffing-shortage | 40902 | greedy | 0 | 0 | -15 min | -92 min | 0 | 0 min | 0 min |
| staffing-shortage | 40902 | rules | 0 | 0 | -16 min | -136 min | 0 | 0 min | 0 min |
| staffing-shortage | 40903 | greedy | 0 | 0 | -18 min | -105 min | 0 | 0 min | 0 min |
| staffing-shortage | 40903 | rules | 0 | 0 | -18 min | -119 min | 0 | 0 min | 0 min |
| stable | 40901 | greedy | 0 | 0 | -54 min | -163 min | 0 | 0 min | 0 min |
| stable | 40901 | rules | 0 | 0 | -55 min | -173 min | 0 | 0 min | 0 min |
| stable | 40902 | greedy | 0 | 0 | -63 min | -198 min | 0 | 0 min | 0 min |
| stable | 40902 | rules | 0 | 0 | -63 min | -204 min | 0 | 0 min | 0 min |
| stable | 40903 | greedy | 0 | 0 | -81 min | -231 min | 0 | 0 min | 0 min |
| stable | 40903 | rules | 0 | 0 | -81 min | -244 min | 0 | 0 min | 0 min |

**Financial performance**

| scenario | seed | policy | netRevenueCents | operatingExpenseCents | operatingProfitCents | margin | operatingCashFlowCents |
|---|---|---|---|---|---|---|---|
| financial-pressure | 40901 | greedy | +$10 | +$32,656 | -$32,646 | – | -$28,756 |
| financial-pressure | 40901 | rules | +$31 | +$32,677 | -$32,646 | – | -$28,355 |
| financial-pressure | 40902 | greedy | +$35 | +$32,587 | -$32,552 | – | -$28,418 |
| financial-pressure | 40902 | rules | +$40 | +$32,600 | -$32,560 | – | -$28,158 |
| financial-pressure | 40903 | greedy | +$4 | +$32,716 | -$32,711 | – | -$29,294 |
| financial-pressure | 40903 | rules | +$16 | +$32,645 | -$32,629 | – | -$28,984 |
| reporting-backlog | 40901 | greedy | +$621 | +$33,947 | -$33,326 | – | -$28,885 |
| reporting-backlog | 40901 | rules | +$621 | +$33,947 | -$33,326 | – | -$28,887 |
| reporting-backlog | 40902 | greedy | +$450 | +$33,916 | -$33,467 | – | -$29,470 |
| reporting-backlog | 40902 | rules | +$450 | +$33,916 | -$33,467 | – | -$29,470 |
| reporting-backlog | 40903 | greedy | +$16 | +$32,753 | -$32,737 | – | -$28,560 |
| reporting-backlog | 40903 | rules | +$16 | +$32,753 | -$32,737 | – | -$28,561 |
| demand-growth | 40901 | greedy | +$2,422 | +$33,949 | -$31,527 | – | -$28,832 |
| demand-growth | 40901 | rules | +$2,422 | +$33,949 | -$31,527 | – | -$28,832 |
| demand-growth | 40902 | greedy | +$5,392 | +$33,950 | -$28,558 | – | -$28,804 |
| demand-growth | 40902 | rules | +$5,392 | +$33,950 | -$28,559 | – | -$28,812 |
| demand-growth | 40903 | greedy | +$235 | +$32,733 | -$32,498 | – | -$28,973 |
| demand-growth | 40903 | rules | +$235 | +$32,733 | -$32,498 | – | -$28,973 |
| staffing-shortage | 40901 | greedy | $0 | +$32,548 | -$32,548 | – | -$29,180 |
| staffing-shortage | 40901 | rules | $0 | +$32,548 | -$32,548 | – | -$29,180 |
| staffing-shortage | 40902 | greedy | -$34 | +$32,554 | -$32,588 | – | -$29,055 |
| staffing-shortage | 40902 | rules | -$34 | +$32,554 | -$32,588 | – | -$29,055 |
| staffing-shortage | 40903 | greedy | +$24 | +$32,577 | -$32,553 | – | -$28,617 |
| staffing-shortage | 40903 | rules | +$110 | +$32,584 | -$32,474 | – | -$28,473 |
| stable | 40901 | greedy | +$50 | +$32,599 | -$32,548 | – | -$28,206 |
| stable | 40901 | rules | +$50 | +$32,599 | -$32,548 | – | -$28,206 |
| stable | 40902 | greedy | +$19 | +$33,238 | -$33,219 | – | -$29,225 |
| stable | 40902 | rules | +$19 | +$33,238 | -$33,219 | – | -$29,225 |
| stable | 40903 | greedy | +$4 | +$32,719 | -$32,715 | – | -$29,268 |
| stable | 40903 | rules | +$4 | +$32,637 | -$32,633 | – | -$29,199 |

**Staffing and capacity**

| scenario | seed | policy | ordersReceived | ordersLost | lostRate | overtimeSec | heldOver | skippedBreaks | noCoverage |
|---|---|---|---|---|---|---|---|---|---|
| financial-pressure | 40901 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| financial-pressure | 40901 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| financial-pressure | 40902 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| financial-pressure | 40902 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| financial-pressure | 40903 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| financial-pressure | 40903 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| reporting-backlog | 40901 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| reporting-backlog | 40901 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| reporting-backlog | 40902 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| reporting-backlog | 40902 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| reporting-backlog | 40903 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| reporting-backlog | 40903 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| demand-growth | 40901 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| demand-growth | 40901 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| demand-growth | 40902 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| demand-growth | 40902 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| demand-growth | 40903 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| demand-growth | 40903 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| staffing-shortage | 40901 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| staffing-shortage | 40901 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| staffing-shortage | 40902 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| staffing-shortage | 40902 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| staffing-shortage | 40903 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| staffing-shortage | 40903 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| stable | 40901 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| stable | 40901 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| stable | 40902 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| stable | 40902 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| stable | 40903 | greedy | 0 | 0 | – | 0 min | 0 | 0 | 0 |
| stable | 40903 | rules | 0 | 0 | – | 0 min | 0 | 0 | 0 |

**Service reliability**

| scenario | seed | policy | tatOverTarget | tatBreaches | waitsOver30 | displacedAppointments | alertsRaisedAlert | alertsActiveAtEnd |
|---|---|---|---|---|---|---|---|---|
| financial-pressure | 40901 | greedy | -47 | -47 | 0 | – | -3 | +1 |
| financial-pressure | 40901 | rules | -51 | -51 | 0 | – | -2 | +1 |
| financial-pressure | 40902 | greedy | -41 | -41 | 0 | – | -6 | 0 |
| financial-pressure | 40902 | rules | -59 | -59 | 0 | – | -5 | 0 |
| financial-pressure | 40903 | greedy | -45 | -45 | 0 | – | -4 | 0 |
| financial-pressure | 40903 | rules | -51 | -51 | 0 | – | -5 | 0 |
| reporting-backlog | 40901 | greedy | -171 | -165 | 0 | – | -3 | -1 |
| reporting-backlog | 40901 | rules | -173 | -167 | 0 | – | -3 | -1 |
| reporting-backlog | 40902 | greedy | -113 | -108 | 0 | – | -3 | 0 |
| reporting-backlog | 40902 | rules | -113 | -108 | 0 | – | -3 | 0 |
| reporting-backlog | 40903 | greedy | -185 | -185 | 0 | – | -3 | 0 |
| reporting-backlog | 40903 | rules | -191 | -191 | 0 | – | -3 | 0 |
| demand-growth | 40901 | greedy | -253 | -238 | 0 | – | -1 | 0 |
| demand-growth | 40901 | rules | -260 | -245 | 0 | – | -1 | 0 |
| demand-growth | 40902 | greedy | -252 | -222 | 0 | – | -2 | -1 |
| demand-growth | 40902 | rules | -252 | -222 | 0 | – | -1 | -1 |
| demand-growth | 40903 | greedy | -252 | -250 | 0 | – | -3 | 0 |
| demand-growth | 40903 | rules | -255 | -253 | 0 | – | -3 | 0 |
| staffing-shortage | 40901 | greedy | -18 | -18 | 0 | – | 0 | 0 |
| staffing-shortage | 40901 | rules | -25 | -25 | 0 | – | 0 | 0 |
| staffing-shortage | 40902 | greedy | -21 | -21 | 0 | – | -3 | 0 |
| staffing-shortage | 40902 | rules | -30 | -30 | 0 | – | -2 | 0 |
| staffing-shortage | 40903 | greedy | -41 | -41 | 0 | – | -1 | 0 |
| staffing-shortage | 40903 | rules | -46 | -46 | 0 | – | -1 | 0 |
| stable | 40901 | greedy | -43 | -43 | 0 | – | 0 | 0 |
| stable | 40901 | rules | -52 | -52 | 0 | – | -1 | 0 |
| stable | 40902 | greedy | -73 | -73 | 0 | – | -7 | 0 |
| stable | 40902 | rules | -75 | -75 | 0 | – | -7 | 0 |
| stable | 40903 | greedy | -116 | -116 | 0 | – | -6 | 0 |
| stable | 40903 | rules | -126 | -126 | 0 | – | -7 | 0 |

**Long-term sustainability**

| scenario | seed | policy | minCashCents | shortfallCents | daysBelowZero |
|---|---|---|---|---|---|
| financial-pressure | 40901 | greedy | -$29,016 | $0 | 0 |
| financial-pressure | 40901 | rules | -$29,031 | $0 | 0 |
| financial-pressure | 40902 | greedy | -$12,425 | $0 | 0 |
| financial-pressure | 40902 | rules | -$12,353 | $0 | 0 |
| financial-pressure | 40903 | greedy | -$21,904 | $0 | 0 |
| financial-pressure | 40903 | rules | -$21,819 | $0 | 0 |
| reporting-backlog | 40901 | greedy | -$7,574 | $0 | 0 |
| reporting-backlog | 40901 | rules | -$7,577 | $0 | 0 |
| reporting-backlog | 40902 | greedy | -$828 | $0 | 0 |
| reporting-backlog | 40902 | rules | -$828 | $0 | 0 |
| reporting-backlog | 40903 | greedy | $0 | $0 | 0 |
| reporting-backlog | 40903 | rules | $0 | $0 | 0 |
| demand-growth | 40901 | greedy | -$8,974 | $0 | 0 |
| demand-growth | 40901 | rules | -$8,974 | $0 | 0 |
| demand-growth | 40902 | greedy | -$1,060 | $0 | 0 |
| demand-growth | 40902 | rules | -$1,060 | $0 | 0 |
| demand-growth | 40903 | greedy | $0 | $0 | 0 |
| demand-growth | 40903 | rules | $0 | $0 | 0 |
| staffing-shortage | 40901 | greedy | -$23,219 | $0 | 0 |
| staffing-shortage | 40901 | rules | -$23,174 | $0 | 0 |
| staffing-shortage | 40902 | greedy | -$11,303 | $0 | 0 |
| staffing-shortage | 40902 | rules | -$11,303 | $0 | 0 |
| staffing-shortage | 40903 | greedy | -$21,418 | $0 | 0 |
| staffing-shortage | 40903 | rules | -$21,274 | $0 | 0 |
| stable | 40901 | greedy | $0 | $0 | 0 |
| stable | 40901 | rules | $0 | $0 | 0 |
| stable | 40902 | greedy | $0 | $0 | 0 |
| stable | 40902 | rules | $0 | $0 | 0 |
| stable | 40903 | greedy | $0 | $0 | 0 |
| stable | 40903 | rules | $0 | $0 | 0 |

**Actions**

| scenario | seed | policy | actionsAttempted | actionsAccepted | actionsRejected |
|---|---|---|---|---|---|
| financial-pressure | 40901 | greedy | +12 | +12 | 0 |
| financial-pressure | 40901 | rules | +12 | +12 | 0 |
| financial-pressure | 40902 | greedy | +12 | +12 | 0 |
| financial-pressure | 40902 | rules | +12 | +12 | 0 |
| financial-pressure | 40903 | greedy | +12 | +12 | 0 |
| financial-pressure | 40903 | rules | +12 | +12 | 0 |
| reporting-backlog | 40901 | greedy | +12 | +12 | 0 |
| reporting-backlog | 40901 | rules | +12 | +12 | 0 |
| reporting-backlog | 40902 | greedy | +12 | +12 | 0 |
| reporting-backlog | 40902 | rules | +12 | +12 | 0 |
| reporting-backlog | 40903 | greedy | +12 | +12 | 0 |
| reporting-backlog | 40903 | rules | +12 | +12 | 0 |
| demand-growth | 40901 | greedy | +12 | +12 | 0 |
| demand-growth | 40901 | rules | +12 | +12 | 0 |
| demand-growth | 40902 | greedy | +12 | +12 | 0 |
| demand-growth | 40902 | rules | +12 | +12 | 0 |
| demand-growth | 40903 | greedy | +12 | +12 | 0 |
| demand-growth | 40903 | rules | +12 | +12 | 0 |
| staffing-shortage | 40901 | greedy | +12 | +12 | 0 |
| staffing-shortage | 40901 | rules | +12 | +12 | 0 |
| staffing-shortage | 40902 | greedy | +12 | +12 | 0 |
| staffing-shortage | 40902 | rules | +12 | +12 | 0 |
| staffing-shortage | 40903 | greedy | +12 | +12 | 0 |
| staffing-shortage | 40903 | rules | +12 | +12 | 0 |
| stable | 40901 | greedy | +12 | +12 | 0 |
| stable | 40901 | rules | +12 | +12 | 0 |
| stable | 40902 | greedy | +12 | +12 | 0 |
| stable | 40902 | rules | +12 | +12 | 0 |
| stable | 40903 | greedy | +12 | +12 | 0 |
| stable | 40903 | rules | +12 | +12 | 0 |

**Accepted actions by type**

| scenario | seed | policy | operations | staffing | by type |
|---|---|---|---|---|---|
| financial-pressure | 40901 | greedy | 0 | 12 | set_reading_allocation: 12 |
| financial-pressure | 40901 | rules | 0 | 12 | set_reading_allocation: 12 |
| financial-pressure | 40902 | greedy | 0 | 12 | set_reading_allocation: 12 |
| financial-pressure | 40902 | rules | 0 | 12 | set_reading_allocation: 12 |
| financial-pressure | 40903 | greedy | 0 | 12 | set_reading_allocation: 12 |
| financial-pressure | 40903 | rules | 0 | 12 | set_reading_allocation: 12 |
| reporting-backlog | 40901 | greedy | 0 | 12 | set_reading_allocation: 12 |
| reporting-backlog | 40901 | rules | 0 | 12 | set_reading_allocation: 12 |
| reporting-backlog | 40902 | greedy | 0 | 12 | set_reading_allocation: 12 |
| reporting-backlog | 40902 | rules | 0 | 12 | set_reading_allocation: 12 |
| reporting-backlog | 40903 | greedy | 0 | 12 | set_reading_allocation: 12 |
| reporting-backlog | 40903 | rules | 0 | 12 | set_reading_allocation: 12 |
| demand-growth | 40901 | greedy | 0 | 12 | set_reading_allocation: 12 |
| demand-growth | 40901 | rules | 0 | 12 | set_reading_allocation: 12 |
| demand-growth | 40902 | greedy | 0 | 12 | set_reading_allocation: 12 |
| demand-growth | 40902 | rules | 0 | 12 | set_reading_allocation: 12 |
| demand-growth | 40903 | greedy | 0 | 12 | set_reading_allocation: 12 |
| demand-growth | 40903 | rules | 0 | 12 | set_reading_allocation: 12 |
| staffing-shortage | 40901 | greedy | 0 | 12 | set_reading_allocation: 12 |
| staffing-shortage | 40901 | rules | 0 | 12 | set_reading_allocation: 12 |
| staffing-shortage | 40902 | greedy | 0 | 12 | set_reading_allocation: 12 |
| staffing-shortage | 40902 | rules | 0 | 12 | set_reading_allocation: 12 |
| staffing-shortage | 40903 | greedy | 0 | 12 | set_reading_allocation: 12 |
| staffing-shortage | 40903 | rules | 0 | 12 | set_reading_allocation: 12 |
| stable | 40901 | greedy | 0 | 12 | set_reading_allocation: 12 |
| stable | 40901 | rules | 0 | 12 | set_reading_allocation: 12 |
| stable | 40902 | greedy | 0 | 12 | set_reading_allocation: 12 |
| stable | 40902 | rules | 0 | 12 | set_reading_allocation: 12 |
| stable | 40903 | greedy | 0 | 12 | set_reading_allocation: 12 |
| stable | 40903 | rules | 0 | 12 | set_reading_allocation: 12 |

## T90

| scenario | seed | policy | status | verdict | classification |
|---|---|---|---|---|---|
| financial-pressure | 40901 | noop | not_applicable | – | – |
| financial-pressure | 40901 | greedy | not_applicable | – | – |
| financial-pressure | 40901 | rules | not_applicable | – | – |
| financial-pressure | 40902 | noop | not_applicable | – | – |
| financial-pressure | 40902 | greedy | not_applicable | – | – |
| financial-pressure | 40902 | rules | not_applicable | – | – |
| financial-pressure | 40903 | noop | not_applicable | – | – |
| financial-pressure | 40903 | greedy | not_applicable | – | – |
| financial-pressure | 40903 | rules | not_applicable | – | – |
| reporting-backlog | 40901 | noop | evaluated | NO GO | not judged |
| reporting-backlog | 40901 | greedy | evaluated | NO GO | not judged |
| reporting-backlog | 40901 | rules | evaluated | NO GO | not judged |
| reporting-backlog | 40902 | noop | evaluated | NO GO | not judged |
| reporting-backlog | 40902 | greedy | evaluated | NO GO | not judged |
| reporting-backlog | 40902 | rules | evaluated | NO GO | not judged |
| reporting-backlog | 40903 | noop | evaluated | NO GO | not judged |
| reporting-backlog | 40903 | greedy | evaluated | NO GO | not judged |
| reporting-backlog | 40903 | rules | evaluated | NO GO | not judged |
| demand-growth | 40901 | noop | not_applicable | – | – |
| demand-growth | 40901 | greedy | not_applicable | – | – |
| demand-growth | 40901 | rules | not_applicable | – | – |
| demand-growth | 40902 | noop | not_applicable | – | – |
| demand-growth | 40902 | greedy | not_applicable | – | – |
| demand-growth | 40902 | rules | not_applicable | – | – |
| demand-growth | 40903 | noop | not_applicable | – | – |
| demand-growth | 40903 | greedy | not_applicable | – | – |
| demand-growth | 40903 | rules | not_applicable | – | – |
| staffing-shortage | 40901 | noop | not_applicable | – | – |
| staffing-shortage | 40901 | greedy | not_applicable | – | – |
| staffing-shortage | 40901 | rules | not_applicable | – | – |
| staffing-shortage | 40902 | noop | not_applicable | – | – |
| staffing-shortage | 40902 | greedy | not_applicable | – | – |
| staffing-shortage | 40902 | rules | not_applicable | – | – |
| staffing-shortage | 40903 | noop | not_applicable | – | – |
| staffing-shortage | 40903 | greedy | not_applicable | – | – |
| staffing-shortage | 40903 | rules | not_applicable | – | – |
| stable | 40901 | noop | not_applicable | – | – |
| stable | 40901 | greedy | not_applicable | – | – |
| stable | 40901 | rules | not_applicable | – | – |
| stable | 40902 | noop | not_applicable | – | – |
| stable | 40902 | greedy | not_applicable | – | – |
| stable | 40902 | rules | not_applicable | – | – |
| stable | 40903 | noop | not_applicable | – | – |
| stable | 40903 | greedy | not_applicable | – | – |
| stable | 40903 | rules | not_applicable | – | – |

## Missing metrics

* imaging capacity utilization (no room-utilization metric in dreamteam.result/v1)
* trend within the run (every metric is a whole-window total; the T90 vector's later-window figures exist only for the reporting-backlog scenario)
* action categories (operations versus staffing) are counted from decisions.json by the report, not by a result metric
* model usage and latency (only for policies that record them, such as the LLM agent's policy-evidence)
