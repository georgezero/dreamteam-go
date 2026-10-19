# Slice 5C.4f: volume-aware clinical severity windows

The owner's decisions of 2026-10-19, implemented in the v5 evaluator
(0.2.0-dev.20). Each priority's severity window is set to its volume.
The evaluation window changes; the severity standard does not.

* **Not frozen.** The candidate policy remains a proposal
  (`../clinical-5c4b/candidate-policy-v5.json`, SHA-256
  `d901fe5877684567c6c2d19ad1c063e8cf36ce269ba6fd44ba54299f052e348b`).
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream. The simulator ran with no action on
  the development seeds 901–910 only.
* **History unchanged:** v0.2-eval-4 is byte-identical. The 5C.4d and
  5C.4e candidates' exact bytes still load and keep their rules
  (`TestOldV5PolicyCompatibility`, `TestOld5C4ECompatibility`).

## 1. The rules

| Priority | Severe | Severity window | Minimum per complete window | Severe rate | Absolute limit (zero tolerance) | Per-seed severe rate |
|---|---|---|---|---|---|---|
| urgent | > 12 h past due | **none** (no weekly gate) | — | — | 24 h | ≤ 5% (kept) |
| soon | > 24 h | **fixed four-week calendar blocks** | 100 (provisional) | ≤ 5% per block | 48 h | ≤ 5% (kept) |
| routine | > 72 h | fixed Monday–Sunday weeks (unchanged) | 100 | ≤ 5% per week | 14 days | ≤ 5% (kept) |

Gate A (≥ 95% within 48 h / 7 d / 30 d) is unchanged. So are Gate B (an
empirical mean increase of at most 2 points, with no population claim
and the non-deterioration limitation) and the per-seed safeguard (now
approved: 10 / 6 / 5 points). The safeguard is an empirical benchmark
safeguard, not a calibrated false-flag guarantee. Its derivation
assumes an independence that may not hold in the simulator.

## 2. Configuration

The windows are explicit policy values for each priority, not
hard-coded exceptions:

```json
"weeklyPartition": "calendar",
"weeklyWindowWeeks": {"urgent": 0, "soon": 4, "routine": 1},
"maxWeeklySevereRate": {"soon": 0.05, "routine": 0.05},
"minWeeklyCohort": {"soon": 100, "routine": 100}
```

* **`weeklyWindowWeeks`** gives each priority's window in calendar
  weeks. The value 0 means no weekly gate for that priority, so it is
  neither measured nor gated. Every priority must be named. The field
  is allowed only with the calendar partition, from 0 to 13 weeks.
* **A disabled priority** has no rate or minimum. An enabled one has
  both.
* **When `weeklyWindowWeeks` is absent**, every priority uses one week:
  the 5C.4e behaviour, byte for byte.

## 3. Windows and boundaries

* **Anchor:** Monday 00:00 of the first complete week. That is the
  first Monday at or after the takeover; if the takeover is itself
  Monday 00:00, the anchor is the takeover.
* **Windows:** consecutive, fixed, non-overlapping windows of L weeks
  from the anchor (28 days for soon). Boundaries are never slid,
  optimized or chosen after the data are seen, and windows are never
  pooled on outcomes.
* **Leading partial window:** the takeover's own week, before the
  anchor.
* **Trailing partial window:** the window the horizon cuts. Studies
  acquired after the horizon are not counted.

| Window | Below the minimum | At or above it |
|---|---|---|
| **complete** (wholly within the gated period) | **INSUFFICIENT**: the seed lacks evidence; it never silently passes | PASS, or FAIL when severe ÷ acquired > rate (exact) |
| **partial** (leading or trailing) | **NOT_JUDGED**: disclosed with its dates and counts, never a pass, not held against the seed | judged like a complete window |

A partial window's acquisitions still count toward the per-seed severe
rates and the absolute limits. Every window is reported with:
- its exact start and end dates;
- the number of weeks (for multi-week windows);
- complete or partial;
- cohort, severe count, rate and status.

A seed's gate fails when any judged window fails. Otherwise it is
INSUFFICIENT when any complete window is below the minimum, and PASS
otherwise.

Urgent loses only its weekly gate. Its per-seed severe rate and its
absolute 24-hour limit are evaluated as before
(`TestUrgentWeeklyGateDisabled`).

## 4. Development-only evaluability (`sim-access-5c4f.json`)

The no-action practice on development seeds 901–910 runs for 150 days.
The takeover is Tuesday 2026-10-13, so the anchor is Monday 2026-10-19.
Each seed has one leading partial week, five complete soon blocks and
one trailing partial block (from 2027-03-08). For routine it has 20
complete weeks and two partial ones.

### 4.1 Evaluability

| Priority | Complete windows | Insufficient | Seeds with an insufficient window |
|---|---|---|---|
| urgent | none (no weekly gate) | 0 urgent seed gates insufficient | **0 of 10: every seed evaluable** |
| soon | 50 blocks | **0** | **0 of 10** |
| routine | 200 weeks | 0 | 0 of 10 |

The stop condition does not apply: no complete soon block is below 100,
and no seed is insufficient for its expected volume.

### 4.2 Soon block cohorts

* **Complete blocks:** mean 166.0 acquisitions, minimum 127, p10 147,
  median 167, maximum 218. The smallest block is 27% above the
  minimum.
* **Severe rate per complete block:** 1.8% to 9.4%, median 4.3%.
* **Partial blocks:** the leading week has 24–46 acquisitions and the
  trailing block 21–37. All 20 are NOT_JUDGED and disclosed.

### 4.3 FAIL versus INSUFFICIENT

Every no-action seed is clinically FAIL, and none is INSUFFICIENT on a
severity window. The failing rules on 10 seeds:

| Rule | Seeds |
|---|---|
| service (v4 breach gates) | 10 FAIL |
| soon four-week block above 5% | **9 FAIL** (17 of 50 complete blocks) |
| soon per-seed severe rate (T90 or tail) | 5 FAIL |
| urgent per-seed severe rate | 2 FAIL |
| backlog stability | 3 unstable |
| soon timely access | 1 FAIL |
| urgent absolute 24 h limit | 8 seeds with 1–5 violations (clinical safety) |

The soon failures are **genuine reporting delays**. Every seed's worst
soon report is 2,517–2,708 minutes (42–45 h) past due: severe beyond
24 h, but within the 48 h absolute limit. The failing blocks have
severe rates of 5.3–9.4%. The no-action practice is not required to
pass; it fails on its delays, not on its volume.

### 4.4 Anchoring sensitivity (reported only)

The per-week soon counts (`soonCalendarWeeks`) were re-summed into
four-week blocks with the anchor moved 0–3 weeks later. Offset 0
reproduces the evaluator exactly.

| Anchor offset | Judged blocks: PASS / FAIL | NOT_JUDGED | Seeds failing | Seeds insufficient |
|---|---|---|---|---|
| 0 (the rule) | 33 / 17 | 20 | 9 | 0 |
| 1 week | 33 / 17 | 10 | 8 | 0 |
| 2 weeks | 36 / 21 | 3 | 10 | 0 |
| 3 weeks | 30 / 20 | 10 | 10 | 0 |

* **Evaluability does not depend on the anchor.** No anchor leaves a
  seed insufficient.
* **The verdict on the no-action practice does not depend on it.** It
  fails on 8–10 seeds under every anchor.
* **Which seed fails can depend on it** (1–2 of 10), because a block of
  about 166 acquisitions is about 8 severe reports from the 5% line.
  This is why the anchor is fixed in advance: the rule is not
  re-anchored after outcomes are seen.

### 4.5 Partial blocks and one more severe report

* **Partial blocks:** all 20 partial soon windows are below 100 and are
  NOT_JUDGED. Excluding them changes no seed's status, because their
  studies stay in the per-seed severe rates. A partial window of 100 or
  more would be judged (`TestSoonBlockPartials`).
* **One more severe report:** of 33 passing complete blocks, 6 would
  fail with one more severe report (they sit within one report of 5%).
  At the 100 minimum, 5 of 100 passes and 6 of 100 fails
  (`TestSoonBlockMinimumAndRate`).

## 5. Witnesses (`internal/eval/severity_window_test.go`)

| Case | Result |
|---|---|
| urgent weekly gate disabled | no urgent window measured, no urgent weekly seed gate; soon and routine present |
| urgent per-seed severe rate retained | 10 of 150 urgent 13 h past due: severe rate FAIL, NO GO |
| urgent absolute maximum retained | six urgent reports 8 days late: clinical safety FAIL on every seed |
| soon 28-day boundaries | the block's last second and the next block's first second fall in different blocks; exact dates |
| soon layout | the leading partial week, two complete 28-day blocks, a trailing partial; a Monday 00:00 takeover has no leading partial |
| soon minimum exactly 100 | judged |
| soon 99 (complete) | INSUFFICIENT; 99 followed by 101 is not pooled |
| soon rate exactly 5% / 6 of 100 | PASS / FAIL (one more severe report) |
| sparse block (empty, complete) | INSUFFICIENT |
| complete versus partial | a partial block of 100 with 6 severe FAILs; a partial block of 50 is NOT_JUDGED, never insufficient |
| horizon truncation | studies after the horizon not counted; the cut block is partial and holds only those before it |
| routine weekly rule unchanged | identical to the 5C.4e candidate's result, field for field |
| missing or contradictory records | a soon study of unknown priority, or a soon order with an empty terminal outcome: INVALID, no panic |
| policy validation | window fields only with the calendar partition; every priority named; 0–13 weeks; a disabled priority without a rate or minimum, an enabled one with both |
| old v5 policies | the 5C.4d and 5C.4e candidates' exact bytes load and keep their rules |
| v4 | byte-identical (`TestV4Neutrality`) |
| historical replay | dev.15–dev.19 sessions exact; golden references exact |

## 6. Mutation checks

Each rule was turned off alone and the window witnesses rerun. **All 11
were caught:**

* the disabled urgent priority measured anyway, and its gate still
  emitted;
* the blocks anchored at the takeover's week, or slid by a week;
* partial blocks below the minimum made insufficient;
* complete blocks below the minimum passing;
* a block above the rate not failing;
* the horizon ignored for completeness;
* the windows defaulting to one week;
* validation accepting a disabled priority with a rate, or a missing
  priority.
