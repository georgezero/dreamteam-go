# Calibration report — established practice (V0.1 step 18)

Nine seeds (1–9), the established practice after its 180-day warm-up,
measured over 30 and 90 days from the takeover (Tue Oct 13 06:00; the 90-day
window runs into January and its December season factor of 0.93).
Reproduce with:

    DT_SEEDS=1-9 DT_DAYS=90 DT_CALIB_OUT=$PWD/docs/calibration/baseline-0.1.0-dev.15.json \
        go test -tags calibration ./internal/sim -run CalibrationReport -v -timeout 60m
    DT_SEEDS=1-9 go test -tags calibration ./internal/sim -run CalibrationActions -v -timeout 60m

`baseline-0.1.0-dev.15.json` holds every seed's figures. It was recorded
before any change was considered; no scenario input changed afterwards
(owner decision 2026-10-14, below), so it also describes the calibrated
default.

## Step 18.1 corrections (0.1.0-dev.18.1)

The step 18 review (`docs/reviews/step-18-astra-review.md`) found
arithmetic, labeling and provenance problems in this report and three
engine defects. All are fixed; every experiment was rerun on the fixed
engine and is now reproducible from a saved manifest:

    go test -tags calibration ./internal/sim -run 'CalibrationExperiments$' -v -timeout 120m

The manifests are `docs/calibration/experiments/*.json` (scenario, overlay,
strategy, seeds, horizon). Each run writes `docs/calibration/results/<name>.json`
(engine version, base and overlaid scenario fingerprints, overlay SHA-256,
per-seed windows, aggregates) and regenerates
`docs/calibration/results/summary.md`. Corrections below are marked
**[18.1]**; the baseline tables above the diagnosis are unchanged (the
fixes changed no outcome of the default practice: log and ledger digests
are identical, and the referral counters never included walk-ins in those
tables).

* **Booking feasibility fixed.** Shared-staff booking now checks for an
  actual assignment of distinct qualified technologists. Default outcomes
  are identical; only strategies that extend Riley past 16:15 (C, D)
  change, by at most 1 exam and 0.1 margin points (table below). The
  struggling-practice characterization stands.
* **Walk-in accounting separated** (arrival, same-visit admission,
  conversion to an appointment, exam). Walk-in losses are no longer in
  referral `lost`: referral lost bookings over 30 days are 129, not 141
  (the 12 were walk-ins whose offered appointment did not suit them).
* **Report figures corrected:** the loss decomposition, lead time, TAT
  labels and the shortest-first effect.

## Owner decision (2026-10-14)

The default established practice is a **struggling, capacity-constrained
practice**. Documented reimbursement, costs, staffing and operational
assumptions are preserved; its margin is not forced into a band but
characterized by these measurements. The 6–12% operating-margin target
belongs to a separate better-staffed established practice (next scenario
milestone, below). The mini scenario is unchanged.

## Baseline (all nine seeds; per month = ×30.4/days)

**[18.1]** "Lead" is receipt → first appointment day, including the
referral's own requested lead (split below); TAT rows are all priorities;
"lost" rows are referral orders only.

### 30 days

| metric                                   |       mean |     median |        min |        max |
|---|---|---|---|---|
| takeover worklist (unread+reading)       |        0.1 |        0.0 |        0.0 |        1.0 |
| exams / month                            |     1381.3 |     1384.2 |     1349.8 |     1401.4 |
| reports final / month                    |     1379.0 |     1381.2 |     1348.7 |     1399.4 |
| referral orders / month                  |     1498.3 |     1500.7 |     1454.1 |     1533.2 |
| walk-ins / month                         |      232.8 |      225.0 |      198.6 |      275.6 |
| XR orders / month                        |      585.9 |      583.7 |      545.2 |      609.0 |
| XR walk-ins / month                      |      190.3 |      187.5 |      154.0 |      240.2 |
| XR lost declined / month                 |       16.6 |       16.2 |       12.2 |       22.3 |
| XR lost no slot / month                  |        0.9 |        1.0 |        0.0 |        2.0 |
| XR lead p50 days                         |        6.4 |        6.0 |        6.0 |        7.0 |
| XR lead p90 days                         |       44.7 |       43.0 |       40.0 |       52.0 |
| XR exams / Tue–Fri day                   |       27.5 |       27.5 |       25.9 |       28.2 |
| XR exams / Monday                        |       19.5 |       19.5 |       17.0 |       22.5 |
| XR exams / Saturday                      |       12.1 |       12.5 |       11.2 |       12.8 |
| XR busy/open %                           |       35.6 |       35.5 |       35.1 |       37.0 |
| XR busy/staffed %                        |       38.9 |       38.9 |       38.4 |       40.3 |
| XR staffed/open %                        |       91.3 |       91.3 |       90.6 |       91.7 |
| XR net revenue / report $                |       51.3 |       51.5 |       50.5 |       51.9 |
| US orders / month                        |      295.3 |      292.9 |      280.7 |      316.2 |
| US walk-ins / month                      |       27.6 |       29.4 |       22.3 |       31.4 |
| US lost declined / month                 |       64.5 |       64.9 |       53.7 |       79.0 |
| US lost no slot / month                  |        1.5 |        2.0 |        0.0 |        3.0 |
| US lead p50 days                         |       19.6 |       19.0 |       18.0 |       22.0 |
| US lead p90 days                         |       50.9 |       50.0 |       48.0 |       55.0 |
| US exams / Tue–Fri day                   |       10.0 |       10.1 |        9.1 |       10.6 |
| US exams / Monday                        |        5.3 |        5.2 |        4.8 |        5.8 |
| US exams / Saturday                      |        3.8 |        4.0 |        3.2 |        4.2 |
| US busy/open %                           |       41.8 |       41.6 |       39.2 |       44.0 |
| US busy/staffed %                        |       69.3 |       69.0 |       65.1 |       72.6 |
| US staffed/open %                        |       60.4 |       60.2 |       59.7 |       61.2 |
| US net revenue / report $                |      174.5 |      174.4 |      170.2 |      181.6 |
| CT orders / month                        |      390.6 |      392.2 |      353.7 |      421.5 |
| CT walk-ins / month                      |       15.0 |       13.2 |        9.1 |       22.3 |
| CT lost declined / month                 |       14.7 |       14.2 |       11.1 |       18.2 |
| CT lost no slot / month                  |        0.5 |        0.0 |        0.0 |        2.0 |
| CT lead p50 days                         |        6.4 |        6.0 |        6.0 |        7.0 |
| CT lead p90 days                         |       47.0 |       47.0 |       42.0 |       52.0 |
| CT exams / Tue–Fri day                   |       16.0 |       16.2 |       14.8 |       17.2 |
| CT exams / Monday                        |        8.8 |        8.8 |        7.2 |       10.5 |
| CT exams / Saturday                      |        6.0 |        6.0 |        5.5 |        6.8 |
| CT busy/open %                           |       35.6 |       35.8 |       32.5 |       37.7 |
| CT busy/staffed %                        |       38.9 |       39.3 |       35.4 |       41.4 |
| CT staffed/open %                        |       91.4 |       91.3 |       90.9 |       91.8 |
| CT net revenue / report $                |      234.1 |      234.1 |      227.6 |      239.5 |
| MR orders / month                        |      226.4 |      228.0 |      205.7 |      255.4 |
| MR walk-ins / month                      |        0.0 |        0.0 |        0.0 |        0.0 |
| MR lost declined / month                 |       31.4 |       30.4 |       23.3 |       45.6 |
| MR lost no slot / month                  |        0.7 |        1.0 |        0.0 |        2.0 |
| MR lead p50 days                         |       12.4 |       13.0 |       10.0 |       15.0 |
| MR lead p90 days                         |       49.0 |       50.0 |       41.0 |       55.0 |
| MR exams / Tue–Fri day                   |        7.7 |        7.6 |        7.4 |        8.2 |
| MR exams / Monday                        |        7.8 |        7.8 |        7.5 |        8.2 |
| MR exams / Saturday                      |        0.0 |        0.0 |        0.0 |        0.0 |
| MR busy/open %                           |       43.1 |       42.6 |       40.1 |       45.8 |
| MR busy/staffed %                        |       71.4 |       70.7 |       67.3 |       76.4 |
| MR staffed/open %                        |       60.4 |       60.4 |       59.6 |       61.0 |
| MR net revenue / report $                |      320.6 |      317.8 |      309.1 |      334.9 |
| Jordan busy (read) % of duty (blocks)    |       40.7 |       40.5 |       39.8 |       41.8 |
| Jordan overtime h / month                |        0.5 |        0.5 |        0.3 |        0.8 |
| Taylor busy (read) % of duty (blocks)    |       26.6 |       26.3 |       25.7 |       28.1 |
| Taylor overtime h / month                |        0.6 |        0.7 |        0.4 |        0.8 |
| Riley busy (read) % of duty (blocks)     |       77.1 |       77.5 |       74.9 |       78.6 |
| Riley overtime h / month                 |        6.0 |        5.3 |        4.1 |        9.7 |
| Sam busy (read) % of duty (blocks)       |       86.4 |       86.7 |       81.4 |       91.4 |
| Sam overtime h / month                   |       14.4 |       15.0 |        8.8 |       17.6 |
| Casey busy (read) % of duty (blocks)     |       80.6 |       80.0 |       76.3 |       86.2 |
| Casey overtime h / month                 |        2.7 |        2.9 |        0.4 |        4.8 |
| Morgan busy (read) % of duty (blocks)    |       81.7 |       81.5 |       77.5 |       85.7 |
| Morgan overtime h / month                |        4.1 |        3.5 |        1.8 |        7.6 |
| Dr. Avery busy (read) % of duty (blocks) |       75.9 |       75.8 |       73.6 |       77.6 |
| Dr. Avery overtime h / month             |        0.2 |        0.1 |        0.0 |        0.6 |
| Dr. Blake busy (read) % of duty (blocks) |       65.5 |       66.1 |       62.8 |       66.7 |
| Dr. Blake overtime h / month             |       10.0 |       10.8 |        4.8 |       11.9 |
| Tue–Fri exams / day (all)                |       61.2 |       61.4 |       59.5 |       62.8 |
| cancel rate %                            |        8.7 |        8.7 |        7.5 |        9.6 |
| no-show rate % (of not cancelled)        |        6.4 |        6.4 |        5.2 |        7.5 |
| walk-ins left / month                    |      109.2 |      104.4 |       90.2 |      132.7 |
| technologist busy/duty %                 |       81.5 |       81.9 |       78.9 |       83.1 |
| radiologist read/block %                 |       71.2 |       71.6 |       68.7 |       72.6 |
| backlog 07:30 Tue–Fri, mean              |        4.3 |        4.2 |        3.4 |        5.4 |
| backlog 07:30 Tue–Fri, max               |       11.3 |       10.0 |        9.0 |       18.0 |
| share of Tue–Fri 07:30 in 5–20 %         |       47.7 |       52.9 |       29.4 |       64.7 |
| unread at close, mean                    |        5.0 |        4.9 |        4.6 |        5.6 |
| TAT p50 min, all priorities (censored)   |       33.8 |       33.0 |       32.0 |       36.0 |
| TAT p90 min, all priorities              |      305.7 |      141.0 |      119.0 |      665.0 |
| TAT over target %, all priorities        |        5.2 |        5.0 |        4.9 |        5.8 |
| net revenue $k / month                   |      208.1 |      209.6 |      198.1 |      211.3 |
| labor $k / month                         |      108.1 |      108.1 |      107.4 |      108.5 |
| fixed $k / month                         |       87.9 |       87.9 |       87.9 |       87.9 |
| variable $k / month                      |       34.3 |       34.4 |       32.5 |       35.1 |
| opex $k / month (adjusted)               |      234.1 |      234.5 |      231.6 |      235.0 |
| operating profit $k / month              |      -26.0 |      -25.4 |      -33.5 |      -23.4 |
| operating margin %                       |      -12.5 |      -12.1 |      -16.9 |      -11.1 |
| collections $k / month                   |      196.6 |      194.4 |      189.2 |      206.3 |
| disbursements $k / month                 |      223.2 |      223.3 |      221.3 |      225.1 |
| A/R change $k                            |       11.8 |       11.5 |        5.0 |       20.1 |
| cash at end $k                           |      123.7 |      122.9 |      115.1 |      131.8 |
| overtime h / month                       |       38.5 |       40.6 |       23.7 |       45.2 |

### 90 days

| metric                                   |       mean |     median |        min |        max |
|---|---|---|---|---|
| takeover worklist (unread+reading)       |        0.1 |        0.0 |        0.0 |        1.0 |
| exams / month                            |     1313.8 |     1312.6 |     1293.0 |     1334.6 |
| reports final / month                    |     1310.3 |     1308.2 |     1289.3 |     1331.2 |
| referral orders / month                  |     1422.7 |     1425.4 |     1379.1 |     1449.7 |
| walk-ins / month                         |      223.8 |      221.9 |      203.3 |      242.9 |
| XR orders / month                        |      559.9 |      556.7 |      539.1 |      581.3 |
| XR walk-ins / month                      |      181.1 |      180.4 |      165.2 |      196.6 |
| XR lost declined / month                 |       14.8 |       15.5 |       11.1 |       16.6 |
| XR lost no slot / month                  |        1.9 |        1.7 |        1.0 |        3.4 |
| XR lead p50 days                         |        6.3 |        6.0 |        6.0 |        7.0 |
| XR lead p90 days                         |       48.2 |       48.0 |       43.0 |       53.0 |
| XR exams / Tue–Fri day                   |       27.0 |       27.0 |       26.1 |       28.7 |
| XR exams / Monday                        |       19.0 |       18.8 |       17.5 |       21.0 |
| XR exams / Saturday                      |       11.9 |       11.8 |       11.2 |       12.5 |
| XR busy/open %                           |       34.5 |       34.5 |       33.7 |       35.6 |
| XR busy/staffed %                        |       37.8 |       37.9 |       37.0 |       39.2 |
| XR staffed/open %                        |       91.1 |       91.1 |       90.8 |       91.3 |
| XR net revenue / report $                |       50.9 |       50.8 |       50.5 |       51.4 |
| US orders / month                        |      282.9 |      286.1 |      260.8 |      291.8 |
| US walk-ins / month                      |       26.3 |       25.0 |       22.3 |       32.1 |
| US lost declined / month                 |       62.5 |       62.5 |       49.7 |       71.6 |
| US lost no slot / month                  |        1.9 |        2.0 |        0.0 |        3.0 |
| US lead p50 days                         |       19.4 |       19.0 |       17.0 |       22.0 |
| US lead p90 days                         |       54.2 |       54.0 |       52.0 |       58.0 |
| US exams / Tue–Fri day                   |       10.0 |       10.0 |        9.8 |       10.3 |
| US exams / Monday                        |        5.3 |        5.2 |        5.1 |        5.5 |
| US exams / Saturday                      |        3.4 |        3.4 |        3.2 |        3.7 |
| US busy/open %                           |       41.4 |       41.5 |       39.9 |       42.9 |
| US busy/staffed %                        |       69.4 |       69.0 |       67.2 |       71.6 |
| US staffed/open %                        |       59.7 |       59.8 |       59.3 |       60.2 |
| US net revenue / report $                |      174.1 |      172.9 |      168.3 |      182.0 |
| CT orders / month                        |      364.8 |      364.8 |      348.2 |      383.7 |
| CT walk-ins / month                      |       16.4 |       15.9 |       12.8 |       19.9 |
| CT lost declined / month                 |       15.1 |       15.5 |       12.5 |       16.9 |
| CT lost no slot / month                  |        1.6 |        1.0 |        0.3 |        3.7 |
| CT lead p50 days                         |        6.4 |        6.0 |        6.0 |        7.0 |
| CT lead p90 days                         |       48.1 |       48.0 |       44.0 |       52.0 |
| CT exams / Tue–Fri day                   |       15.1 |       15.3 |       14.4 |       15.6 |
| CT exams / Monday                        |        8.7 |        8.8 |        7.7 |        9.3 |
| CT exams / Saturday                      |        6.0 |        6.1 |        5.7 |        6.2 |
| CT busy/open %                           |       34.0 |       34.3 |       32.3 |       35.5 |
| CT busy/staffed %                        |       37.3 |       37.7 |       35.6 |       39.0 |
| CT staffed/open %                        |       91.1 |       91.2 |       90.9 |       91.3 |
| CT net revenue / report $                |      234.4 |      234.0 |      231.2 |      237.8 |
| MR orders / month                        |      215.1 |      213.5 |      203.7 |      227.7 |
| MR walk-ins / month                      |        0.0 |        0.0 |        0.0 |        0.0 |
| MR lost declined / month                 |       27.9 |       28.0 |       21.6 |       36.1 |
| MR lost no slot / month                  |        1.5 |        1.7 |        0.7 |        2.7 |
| MR lead p50 days                         |       12.3 |       12.0 |       11.0 |       14.0 |
| MR lead p90 days                         |       51.6 |       52.0 |       47.0 |       56.0 |
| MR exams / Tue–Fri day                   |        7.7 |        7.7 |        7.4 |        8.0 |
| MR exams / Monday                        |        7.7 |        7.7 |        7.4 |        8.2 |
| MR exams / Saturday                      |        0.0 |        0.0 |        0.0 |        0.0 |
| MR busy/open %                           |       42.5 |       42.6 |       40.4 |       44.0 |
| MR busy/staffed %                        |       71.2 |       71.2 |       68.0 |       74.1 |
| MR staffed/open %                        |       59.6 |       59.7 |       59.3 |       59.9 |
| MR net revenue / report $                |      321.8 |      320.1 |      316.1 |      330.1 |
| Jordan busy (read) % of duty (blocks)    |       40.0 |       40.1 |       39.3 |       40.5 |
| Jordan overtime h / month                |        0.4 |        0.4 |        0.3 |        0.5 |
| Taylor busy (read) % of duty (blocks)    |       26.0 |       26.0 |       25.4 |       26.6 |
| Taylor overtime h / month                |        0.6 |        0.7 |        0.3 |        0.8 |
| Riley busy (read) % of duty (blocks)     |       76.1 |       76.2 |       73.3 |       77.5 |
| Riley overtime h / month                 |        5.1 |        4.8 |        4.3 |        6.2 |
| Sam busy (read) % of duty (blocks)       |       82.3 |       82.0 |       80.9 |       84.4 |
| Sam overtime h / month                   |       11.4 |       11.2 |        9.5 |       13.9 |
| Casey busy (read) % of duty (blocks)     |       80.5 |       80.5 |       77.0 |       83.7 |
| Casey overtime h / month                 |        2.8 |        2.9 |        1.8 |        3.5 |
| Morgan busy (read) % of duty (blocks)    |       81.8 |       81.5 |       79.4 |       83.7 |
| Morgan overtime h / month                |        4.2 |        4.5 |        2.9 |        5.6 |
| Dr. Avery busy (read) % of duty (blocks) |       73.4 |       73.2 |       72.6 |       74.8 |
| Dr. Avery overtime h / month             |        0.1 |        0.2 |        0.0 |        0.2 |
| Dr. Blake busy (read) % of duty (blocks) |       63.1 |       63.2 |       61.6 |       64.3 |
| Dr. Blake overtime h / month             |        7.4 |        7.2 |        6.0 |        9.8 |
| Tue–Fri exams / day (all)                |       59.9 |       59.9 |       58.9 |       60.9 |
| cancel rate %                            |        8.9 |        8.9 |        8.2 |        9.6 |
| no-show rate % (of not cancelled)        |        6.5 |        6.6 |        5.5 |        7.4 |
| walk-ins left / month                    |       99.7 |       99.3 |       85.5 |      110.1 |
| technologist busy/duty %                 |       80.2 |       80.3 |       78.8 |       81.1 |
| radiologist read/block %                 |       68.7 |       68.8 |       67.6 |       69.5 |
| backlog 07:30 Tue–Fri, mean              |        3.3 |        3.2 |        2.9 |        4.1 |
| backlog 07:30 Tue–Fri, max               |       12.0 |       12.0 |        9.0 |       18.0 |
| share of Tue–Fri 07:30 in 5–20 %         |       33.6 |       33.3 |       29.4 |       41.2 |
| unread at close, mean                    |        4.5 |        4.4 |        4.2 |        4.9 |
| TAT p50 min, all priorities (censored)   |       31.9 |       32.0 |       30.0 |       33.0 |
| TAT p90 min, all priorities              |      122.1 |      119.0 |      115.0 |      132.0 |
| TAT over target %, all priorities        |        5.2 |        5.1 |        5.0 |        5.7 |
| net revenue $k / month                   |      200.4 |      200.3 |      197.4 |      202.7 |
| labor $k / month                         |      106.8 |      106.8 |      106.4 |      107.0 |
| fixed $k / month                         |       87.8 |       87.8 |       87.8 |       87.8 |
| variable $k / month                      |       32.7 |       32.6 |       32.4 |       33.4 |
| opex $k / month (adjusted)               |      228.5 |      228.3 |      228.1 |      229.5 |
| operating profit $k / month              |      -28.1 |      -27.9 |      -30.8 |      -26.3 |
| operating margin %                       |      -14.0 |      -13.9 |      -15.6 |      -13.0 |
| collections $k / month                   |      194.4 |      193.5 |      191.1 |      200.6 |
| disbursements $k / month                 |      239.7 |      239.6 |      238.8 |      241.0 |
| A/R change $k                            |       11.1 |       15.2 |       -5.4 |       25.0 |
| cash at end $k                           |       16.0 |       13.5 |        8.9 |       30.6 |
| overtime h / month                       |       32.0 |       31.9 |       27.0 |       38.1 |

## Diagnosis

**Demand is not the constraint.** Referral orders plus walk-ins come to
1,647 a month (90-day mean), the design's ~1,650. Exams are 1,314 a month
over 90 days (1,381 over the first 30, before the December season dip).

**Where demand goes [18.1].** The earlier "330 a month" mixed flows (a
month's exams include orders received before the month; its losses
include orders received after it) and its parts summed to ≈372. The
reconciliation is an order cohort: every order received in the window is,
at its end, completed, ended without an exam, or still outstanding — each
counted once (`orderCohort` in the results; a hand-checkable one-week
version is `TestRegress18OrderCohortReconciles`). Nine seeds, 90 days:

| | Received | Completed | Ended without exam | Outstanding at the end |
|---|---|---|---|---|
| Referral orders | 4,212 | 2,785 | 632: lost 376 (earliest slot beyond tolerance, or none in the horizon), cancelled 146, no-show 109 | 795: booked 757, on recall 38 |
| Walk-ins | 662 | 357 | 304: left 295, offered appointment lost 7, later cancelled or no-show 2 | 2 |

Over 30 days the referral cohort is 1,479 received: 704 completed, 197
ended (lost 129, cancelled 41, no-show 27), 577 outstanding — most
referrals are booked days to weeks ahead, so a 30-day cohort is mostly
still waiting at its end. Lost referral bookings per 90 days by modality:
US the most (≈190), then MR, CT and XR.

**Why: technologist time and slot templates, not rooms.**

| Constraint | Evidence (9-seed means) |
|---|---|
| XR/CT on one shared technologist most of the day | Sam 82–86% busy, Riley 76–77%; the XR room is busy only 35–38% of staffed time. Walk-ins who leave registered quickly and then waited 30–112 min for an exam (seed 1 trace). Shorter XR room times (×0.75) add only ~27 exams a month; the walk-in policy (`accept_always`) changes nothing. |
| US: one technologist, 45-min slots for ~37-min exams | 10.0 exams per Tue–Fri day, 69% busy of staffed time (~72% is the ceiling with these slots and 7% no-shows; the 80–90% target needs shorter slots or overbooking); US loses the most demand. **[18.1]** The "lead p50 19 days" in the tables is receipt → first appointment and includes the referral's own requested lead: split, US requested lead p50 ≈9.6 days and capacity delay (earliest wanted day → appointment) p50 ≈3 days; MR 8.3 and ≈1.7; XR and CT ≈5.2 and 0. |
| MR: one technologist, 9 template slots | 7.7 exams per weekday; 42% busy of open hours (cap 66.7%, dark after 16:30). |
| Monday thinner by design | XR 19, CT 8.7, US 5.3 per Monday (Sam off; Morgan alternate weeks). |
| Radiologists over-provisioned for this volume | 63–76% of site-reading block time used; TAT p50 32–34 min (all priorities); 07:30 backlog mean 3.3 (90 d), takeover worklist ≈0. A cut to 45%/38% site allocation made the queue unstable (all-priority TAT p50 22 h; rerun `staffing-reduced-reading`); 55%/45% was stable and saved ≈$5k/month — not applied (documented staffing preserved). |

**Economics follow volume.** Net revenue per exam $153 (reference $158.5,
more XR in the mix); labor $106.8k, fixed $87.8k, variable $32.7k a month —
all as documented. Operating margin −12.5% over 30 days (seeds −16.9% to
−11.1%) and −14.0% over 90 days (−15.6% to −13.0%).

**The design's reference was internally inconsistent.** §8.6 assumed 1,650
*exams* from ~1,650 *orders* with the default roster: no no-shows,
cancellations, declines or abandonment, and technologists ~94% busy. A
better-staffed variant shows the limit: three more technologists (an XR/CT
day technologist and US and MR technologists to the 20:00 close) cut lost
referral bookings by 65% (376 → 130 per 90 days) but raise exams only 7%
(1,390 per 30 days over 90 days, against 1,296), because demand, no-shows
and cancellations bind; the margin falls to −18.6% (−20.0 to −17.4).
**[18.1]** Rerun on the fixed booking check (`staffing-three-techs`):
the same conclusion; the step 18 figures were −18.7% and ≈1,407 a month
(×30.4/30). XR room times ×0.75 (`staffing-faster-xr`): +25 exams per 30
days, margin −13.1%. A moderately profitable established practice
therefore needs more demand (referral growth, V0.4) as well as staff — it
cannot be configured cleanly from staffing alone, so it is documented as
the next scenario milestone rather than added now.

## What the player's three actions can do (30 days, nine seeds) [18.1]

Rerun on the fixed engine (`strategy-b` … `strategy-g`; A is `baseline`).
Strategies are decided at the takeover; extensions are approved for every
day the employee works inside the next 30 days, never past the day's
scheduled close (the step 18 helper also asked for Saturday extensions,
which were rejected; the harness now asserts every action is accepted).
Demand is identical across strategies (asserted for every seed in
`TestCalibrationActions`). Means (range); TAT all priorities; lost =
referral bookings lost.

| strategy | exams | lost | walk-ins left / declined | TAT p50 / p90 min | urgent p90 min / breaches | overtime h (approved) | net revenue $k | opex $k | profit $k | margin % |
|---|---|---|---|---|---|---|---|---|---|---|
| A no actions | 1363 (1332–1383) | 129 (113–144) | 108 / 0 | 34 / 306 | 500 / 6.0 | 38 (0) | 205.4 (195.5–208.5) | 231.0 (228.6–231.9) | -25.6 (-33.1–-23.1) | -12.5 (-16.9–-11.1) |
| B US+MR to 20:00, walk-ins until 19:00 | 1431 (1412–1446) | 85 (72–93) | 108 / 5 | 44 / 762 | 668 / 7.4 | 151 (117) | 220.6 (214.5–223.6) | 241.2 (239.4–241.9) | -20.6 (-24.9–-17.7) | -9.4 (-11.6–-7.9) |
| C 2nd XR/CT tech evenings, accept all | 1394 (1368–1420) | 114 (100–131) | 91 / 0 | 36 / 399 | 205 / 6.3 | 100 (77) | 207.6 (198.5–211.3) | 235.5 (233.4–236.9) | -27.8 (-34.9–-25.0) | -13.4 (-17.6–-11.8) |
| D all three extensions, accept all | 1461 (1433–1484) | 69 (58–74) | 93 / 0 | 49 / 817 | 449 / 9.3 | 214 (194) | 222.5 (217.0–224.8) | 245.7 (244.3–246.9) | -23.2 (-27.3–-20.7) | -10.4 (-12.6–-9.2) |
| E decline walk-ins, FIFO worklist | 1394 (1360–1429) | 152 (140–166) | 0 / 230 | 33 / 362 | 718 / 10.4 | 34 (0) | 205.0 (195.4–209.3) | 230.8 (228.4–231.8) | -25.8 (-32.9–-22.4) | -12.6 (-16.9–-10.7) |
| F shortest-first worklist | 1363 (1332–1383) | 129 (113–144) | 108 / 0 | 22 / 157 | 575 / 7.1 | 38 (0) | 205.4 (195.5–208.6) | 231.0 (228.6–231.9) | -25.6 (-33.1–-23.0) | -12.5 (-16.9–-11.0) |
| G FIFO worklist only | 1363 (1332–1383) | 129 (113–144) | 108 / 0 | 34 / 252 | 603 / 12.1 | 38 (0) | 205.4 (195.5–208.5) | 231.0 (228.6–231.9) | -25.6 (-33.1–-23.1) | -12.5 (-16.9–-11.1) |

Against step 18: B, E and F are identical; C and D moved by at most one
exam and 0.1 margin points (the booking fix: Riley extended past 16:15
shares staff with CT); lost is lower in every row because walk-in losses
are no longer counted as referral losses.

* **Capacity costs labor.** Extending US and MR (B) adds 68 exams and
  recovers 44 lost bookings for 117 h of approved overtime: profit +$5k a
  month, margin −9.4%. Adding the XR/CT evening extension (D) raises volume
  most (+98) but its labor eats the gain. Extending Riley alone (C) loses
  money.
* **Declining walk-ins (E) raises exams** (+31): offered appointments
  mostly come back instead of leaving after an hour; referral losses rise
  because declined walk-ins take slots. Walk-in demand stays measured (230
  declined, counted).
* **Shortest-first reading (F) cuts the all-priority TAT median by about
  35%** (34 → 22 min; p90 306 → 157) at no cost — published expected read
  times only. It is not free for urgent studies: see below.
* **No strategy reaches break-even.** The best is about −9%.

## Urgent turnaround under the worklist policies [18.1]

Paired runs (same seeds and demand), 30 and 90 days: the default
(`urgent_then_fifo`), FIFO only (G) and shortest-first (F). Studies acquired
in the window, as of its end; nine-seed means, ranges for max and
breaches. Breaches are completed over target plus unfinished past it;
"overnight" are those first claimed on a later day than acquired.

| policy | window | urgent studies | unfinished | urgent p50 / p90 min | urgent max min | urgent breaches (overnight) | routine p50 / p90 min | routine oldest unfinished min |
|---|---|---|---|---|---|---|---|---|
| default | 30 d | 39.2 | 0 | 15 / 500 | 1884 (731–2457) | 6 (3–11) (5.0) | 35 / 126 | 361 |
| FIFO (G) | 30 d | 39.2 | 0 | 29 / 603 | 1893 (757–2457) | 12 (8–22) (6.1) | 35 / 123 | 361 |
| shortest-first (F) | 30 d | 39.2 | 0 | 16 / 575 | 1894 (742–2465) | 7 (4–13) (5.3) | 22 / 125 | 460 |
| default | 90 d | 114.8 | 0 | 16 / 282 | 2084 (845–2457) | 15 (8–19) (11.6) | 33 / 112 | 2366 |
| FIFO (G) | 90 d | 114.8 | 0 | 27 / 599 | 2101 (891–2457) | 33 (29–37) (14.2) | 33 / 109 | 2366 |
| shortest-first (F) | 90 d | 114.8 | 0 | 16 / 434 | 2083 (839–2465) | 18 (12–23) (12.4) | 22 / 97 | 2366 |

By modality (90 days): urgent XR p50 10 / 19 / 11 min (default / FIFO /
shortest-first), urgent CT 25 / 54 / 35, urgent US 16 / 47 / 21; urgent MR
(about one a month) 52 / 52 / 99. Full tables: `results/summary.md`.

* **The default's urgent tail is a coverage gap, not ordering.** About
  three quarters of its urgent breaches (11.6 of 15 over 90 days) wait
  overnight: studies acquired after the last reading block (weekday
  evenings to 20:00, Saturday afternoons) are read the next morning or on
  Monday. That is the 5–8 h urgent p90 (≈12 h in the seed-1 90-day golden) and the 31–41 h maximum, and it is the
  same under every policy. Reducing it needs evening or weekend reading
  coverage, which is a staffing question, not a worklist one.
* **Within reading hours, the default protects urgent work**; FIFO adds
  about 15 urgent breaches per 90 days in reading hours (18.8 vs 3.4) and
  doubles the urgent median.
* **Shortest-first costs little for urgent work here** (+3 breaches per 90
  days, 2 of them in reading hours), because the worklist empties often;
  but a long urgent study is offered after every shorter routine one: the
  rare urgent MR waits about twice as long (p50 99 vs 52 min), and the
  controlled micro case (`TestRegress18UrgentLongStudyVsShortRoutine`: one
  radiologist, a routine XR every 140 s, one urgent MR) shows unbounded
  starvation when short routine studies never run out — the default reads
  the urgent study next, FIFO after 5 min, shortest-first not at all in 7
  hours. No policy semantics changed; whether shortest-first should keep
  urgent studies first is an owner decision.

### Gameplay rule: urgency always takes precedence (0.1.0-dev.18.2)

Owner decision after this analysis: urgent studies are offered first under
every worklist policy, and the policy orders within each class (FIFO,
oldest-first, shortest expected read; the default is unchanged). This is a
deliberate gameplay rule change, not a correction to the calibration data:
reading blocks, staffing, read durations, reimbursement and the default
policy are unchanged, so the baseline and every default-policy figure in
this report stand. Rerun of the paired comparison (`baseline`,
`strategy-e`, `strategy-f`, `strategy-g`; nine seeds). Exams, lost
bookings, walk-ins, net revenue, opex and margin are identical before and
after in every run; only reading order changed.

| policy | window | urgent p50 / p90 / max min, before → after | urgent breaches (overnight), before → after | routine p50 / p90 min, before → after |
|---|---|---|---|---|
| default | 30 / 90 d | unchanged: 15 / 500 / 1884; 16 / 282 / 2084 | unchanged: 6.0 (5.0); 15.1 (11.6) | unchanged: 35 / 126; 33 / 112 |
| FIFO (G) | 30 d | 29 / 603 / 1893 → 15 / 496 / 1877 | 12.1 (6.1) → 6.0 (5.0) | 35 / 123 → 35 / 126 |
| FIFO (G) | 90 d | 27 / 599 / 2101 → 16 / 282 / 2076 | 32.9 (14.2) → 15.1 (11.6) | 33 / 109 → 33 / 112 |
| shortest-first (F) | 30 d | 16 / 575 / 1894 → 15 / 565 / 1877 | 7.1 (5.3) → 6.2 (5.1) | 22 / 125 → 22 / 126 |
| shortest-first (F) | 90 d | 16 / 434 / 2083 → 15 / 420 / 2076 | 18.2 (12.4) → 16.3 (12.2) | 22 / 97 → 22 / 98 |
| decline walk-ins + FIFO (E) | 30 d | 43 / 718 / 1410 → 19 / 419 / 1379 | 10.4 (5.6) → 4.9 (3.8) | 33 / 122 → 33 / 126 |

* FIFO now matches the default for urgent work (90 days: 15.1 breaches,
  p90 282 min); the cost is a few minutes on the routine p90.
* Shortest-first keeps its routine gain (p50 22 vs 33–35 min) and loses
  its urgent penalty in reading hours; the urgent MR median is back to
  the default's (90 days: 99 → 52 min).
* What remains is the coverage gap: ≈12 overnight urgent breaches per 90
  days and a 31–41 h maximum under every policy. Evening or weekend
  reading coverage could improve service quality without improving the
  margin — a legitimate management trade-off for a later version; V0.1
  has no action for it.

With this rule, the calibration phase is closed.

## Targets

| Target (V0.1 acceptance / design) | Default established practice | Status |
|---|---|---|
| Demand ~1,650 a month | 1,647 (orders + walk-ins) | met |
| Exams ~1,650 a month | 1,314–1,381 | belongs to the better-staffed preset (milestone) |
| Tue–Fri visits 65–85 | 60–61 | below: capacity |
| XR 28–40, US 10–13, CT 14–20, MR 8–11 per Tue–Fri | 27.0, 10.0, 15.1, 7.7 | XR, MR just below; US, CT in |
| Walk-ins 6–12 per Tue–Fri | ≈8 arrive, ≈5 imaged | arrivals in; abandonment high (capacity) |
| No-show 4–9% | 6.5% | met |
| US staffed utilization 80–90%, longest lead, most lost | 69%; lead p50 19 d; most lost | lead and losses met; 80–90% unreachable with 45-min slots and no overbooking |
| MR open-hours utilization ≤ 65% | 42.5% | met |
| Radiologist capacity 105–115% of reading demand | ≈145% at this volume | over-provisioned (documented allocation kept) |
| Unread backlog at 07:30 Tue–Fri 5–20 | mean 3.3, max 9–18; 34% of mornings in 5–20; takeover ≈0 | below, consistent with over-provisioned reading |
| Routine TAT median 1–4 h | 33 min routine (90 d; all priorities 32) [18.1] | faster than target (same cause) |
| T30 operating margin 6–12% | −12.5% (−16.9 to −11.1) | moved to the better-staffed preset |
| Cash and profit diverge on payroll days | yes | met |
