# Slice 5C.4h: clinical boundary safety

This is a narrow revision of the candidate clinical policy. It
implements the owner's decision of 2026-10-19 on audit finding G4: close
the partial-window loophole and the horizon loophole rather than only
disclosing them. The evaluator is 0.2.0-dev.21.

* **Not frozen, and not reviewed.** The revision changes the clinical
  policy, so the 5C.4g component GO verdicts do not carry over to it. It
  needs a targeted independent re-review before any protocol lock.
* **Thresholds unchanged.** The severity thresholds, rates, minimums,
  absolute limits, anchor, Gate A, Gate B, the safeguard and the
  allowance are all unchanged. The revision changes only:
  * how a partial window below the minimum is treated;
  * which studies the absolute limits cover.
* **Two explicit policy fields carry the revision**, so the 5C.4d, 5C.4e
  and 5C.4f candidates keep their own rules byte for byte:
  ```json
  "partialWindows": "countCeiling",
  "safetyCohort": "ordersByHorizon"
  ```

## 1. Partial windows: the count ceiling

### 1.1 The loophole (G4.1, reproduced)

`TestPartialWindowConcentration` rebuilds the audit's witness. In the
leading partial week, 16 of 24 soon reports are 30 hours past due on
every seed:
* that is 640 severe reports across 40 seeds;
* the per-seed soon rate is 16 of 558 (2.9%);
* every complete block passes;
* no report breaches the 48 h absolute limit.

Under the 5C.4f candidate the partial week is NOT_JUDGED and the verdict
is **GO**.

### 1.2 The rule

A partial window (leading or trailing) with fewer acquisitions than the
minimum cohort fails when its severe count alone exceeds what a
compliant window of the minimum size may hold:

$$
\text{FAIL} \iff \frac{\text{severe}}{\text{minimum}} > \text{rate}
\quad(\text{exactly: } \text{severe} \times 10\,000 > \text{rate}_{bp} \times \text{minimum}).
$$

At the candidate's values (5%, minimum 100) the ceiling is ⌊5% × 100⌋
= **5 severe reports**, for soon blocks and routine weeks alike.

| Window | Below the minimum | At or above it |
|---|---|---|
| complete | INSUFFICIENT (unchanged) | PASS, or FAIL above the rate (unchanged) |
| partial | **FAIL** when the severe count exceeds the ceiling; otherwise NOT_JUDGED | judged like a complete window (unchanged) |

What the rule preserves and adds:

* **No sparse partial window is penalized for its size.** Whatever the
  missing volume, a partial window that fails would fail even if it were
  padded to the minimum with on-time reports. A window with no severe
  report never fails, however small.
* **Interpretable at sparse boundaries.** The ceiling is a count
  ("at most 5 severe reports in a boundary window"), the same for every
  plan. It depends neither on the window's size nor on the outcomes.
* **No outcome-dependent pooling, and a fixed anchor.** Windows are
  still the fixed calendar windows from the first Monday. Nothing is
  merged, slid or re-anchored.
* **The per-seed rate and the absolute maximum are unchanged.** Partial
  windows' studies still count in both.
* **NOT_JUDGED never counts as a pass.** A passing seed gate states the
  partial windows it did not judge, for example: "5 judged 4-week
  calendar blocks pass; 2 partial windows below the minimum not judged,
  each within the count ceiling of 5 severe". The plan report gives
  `partialSevereCeiling`, `partialFailingWeeks` and every window's
  counts.
* **No vacuous pass.** When a priority has no judged window and no
  insufficient one (a horizon too short for any complete window, with
  partial windows within the ceiling), its gate is INSUFFICIENT
  (`noJudgedWindow`). The 5C.4f rule passed this case.

### 1.3 Witnesses (`internal/eval/boundary_safety_test.go`)

| Case | 5C.4f | 5C.4h |
|---|---|---|
| G4.1: 16 of 24 severe in the leading partial week, every seed | GO | **NO GO**: 40 seeds fail the soon gate |
| 5 of 99, leading partial | not judged | not judged |
| 6 of 99, leading partial | not judged | **fail** |
| 6 of 24, leading partial | not judged | **fail** |
| 6 of 50, trailing partial | not judged | **fail** |
| 5 of 100 / 6 of 100, partial (judged) | pass / fail | pass / fail |
| complete 99 / complete 6 of 100 | insufficient / fail | insufficient / fail |
| routine leading partial week, 6 of 60 | not judged | **fail** |
| no complete window, partial within the ceiling | pass (vacuous) | **insufficient** |

### 1.4 Effect on the development seeds

**No-action simulator practice** (seeds 901–910, `sim-access-5c4h.json`):
the ceiling fails no partial window.
* **Soon:** the leading partial weeks hold 24–46 acquisitions with 0–4
  severe; the trailing partial blocks hold 21–37 with none.
* **Routine:** no partial week exceeds 5 severe.

Every seed's clinical status is unchanged: all FAIL, as under 5C.4f, on
genuine delays.

**Fidelity practices** (`construction-checks.json`):
* **Stable:** the ceiling flags no seed of the stable practices (20
  groups, 800 seeds).
* **Deteriorations:** it fires only where reports are genuinely late
  (K-85, D-85, L20-85, K-92). There the windows also fail on complete
  blocks.
* **Witness:** in the W-PARTIAL witness it fails every seed.

## 2. Horizon safety: the follow-up cohort

### 2.1 The loophole (G4.2, reproduced)

`TestHorizonCrossingAcquisition` rebuilds it. On every seed, one soon
order wanted the day before the horizon H is imaged within its 7-day
access limit and reported 60 hours past due (12 hours beyond the 48 h
absolute limit).

| Acquisition | 5C.4f | 5C.4h |
|---|---|---|
| at H | NO GO (safety) | NO GO (safety) |
| at H + 1 s | **GO** | **NO GO** (safety: the violation is observed in the follow-up) |

### 2.2 The cohort, origins and deadlines

The absolute past-due limits now cover:
* every study acquired by the horizon H, as before;
* every study of an order wanted by H, wherever its acquisition falls
  in the follow-up.

The follow-up runs from H to the end of observation E. E is at least
H + 31 days (the policy's `settlementDays`) and at least T180 plus
closure.

| Quantity | Definition |
|---|---|
| time origin | the study's due time: acquisition plus the priority's reporting target (urgent 1 h, soon 4 h, routine 24 h). Inherited studies acquired before the takeover are timed from the takeover, as before |
| deadline | due time plus the absolute limit (urgent 24 h, soon 48 h, routine 14 days) |
| cohort membership | acquired by H; or acquired after H with its order wanted by H. The study's `order` link names the order, and the order's original wanted time decides membership |

At E each cohort study is in exactly one state:

| State | Condition | Effect |
|---|---|---|
| 1. **observed violation** | finished, finish − due > limit | violation: safety FAIL (zero tolerance, overrides the allowance) |
| 2. **unresolved overdue** | unfinished at E, E − due > limit | violation: the obligation is already broken, whatever follows |
| 3. **not yet observable** | unfinished at E, E − due ≤ limit | **safety INSUFFICIENT** (never a pass) unless a violation fails it |
| 4. **completed after H, observed in the follow-up** | acquired after H (order wanted by H), finished by E | judged as state 1 or compliant, from its own due time |
| compliant | finished, finish − due ≤ limit | — |

Two further cases:

* **An order wanted by H that is still open at E without a study** is
  counted as `unacquired`. No report is yet due: the reporting clock
  starts at acquisition, and no outcome is invented for an unobserved
  report. Its access is judged by Gate A. E is at least 31 days past H,
  beyond every access limit, so Gate A counts it untimely.
* **A study of an order wanted after H** stays outside the cohort,
  however late its report. That order is not part of the evaluated
  period.

For studies acquired by H, the policy's validation (every limit at most
`settlementDays`) keeps every deadline within E, so state 3 never arises
for them and their accounting is unchanged.

The plan report gives each seed's past-due count by state: `observed`,
`overdue`, `pending`, `postHorizon` and `unacquired`. Safety lists the
not-yet-observable seeds (`pending`), and the verdict gives one reason
for each.

### 2.3 Record integrity

A study acquired after H must name exactly one order of its own
priority. If the link is missing, unknown, ambiguous or of another
priority, the record is **INVALID**: the cohort cannot be decided.

* The simulator's evaluation export writes the link (`order`) from the
  study's existing order index, without drawing any random number.
* Records written before 0.2.0-dev.21 have no link. Under the revision
  they are invalid only if they contain a post-horizon study. The older
  candidates do not need the link.

### 2.4 Witnesses

The follow-up states are tested on a soon order wanted by H and imaged
after it; E is the end of observation.

| Case | 5C.4f | 5C.4h |
|---|---|---|
| acquired 60 h before E, unfinished (56 h past due) | GO | **NO GO**: unresolved overdue |
| acquired 10 h before E, unfinished (6 h past due) | GO | **NO GO**: safety insufficient (not yet observable) |
| acquired 60 h before E, reported exactly at the limit | GO | GO |
| the same, one second beyond | GO | **NO GO**: observed violation |
| acquired an hour before H, reported after H, 60 h past due | NO GO | NO GO |
| an order wanted by H, open at E, no study | — | GO; `unacquired` 1; Gate A one more untimely order |
| an order wanted after H, reported 60 h past due | — | GO (outside the cohort) |
| post-H study link missing, unknown, ambiguous, other priority | GO (link not needed) | **INVALID** |

### 2.5 Development evidence, and the follow-up's length

**No-action simulator practice** (seeds 901–910, `sim-access-5c4h.json`),
under the follow-up cohort:
* **Post-horizon studies.** 7–27 soon and 160–201 routine studies a seed
  are acquired after H for orders wanted by H. A few urgent ones also
  fall in the cohort (two on seed 907).
* **Violations.** They are unchanged from 5C.4f: urgent only, 1–5 a
  seed on 8 seeds. The follow-up adds none, and records 5–15 routine
  orders a seed as unacquired (judged by Gate A).
* **Not yet observable.** **9 of 10 seeds have 1–6 routine studies not
  yet observable**, so safety is INSUFFICIENT. That is the only safety
  shortfall on seed 908, which has no violation.
  * **When they were imaged.** 28–29.5 days after H, which is 1.5–2.6
    days before E. They were still unread, because the no-action
    practice's routine reporting runs that far behind.
  * **How late they already were.** All but one were imaged more than
    30 days after their wanted time, past routine's access limit and
    untimely under Gate A. The exception was imaged 29.6 days after its
    wanted time.

**Fidelity practices** (`construction-checks.json`): no stable seed is
not yet observable (T7: 0 of 20 groups).

The rule behaves as the owner specified. A follow-up that cannot
establish the state gives INSUFFICIENT, never PASS. It arises where
routine waits approach the 30-day access limit.

**An owner option (not implemented):** require E ≥ H + 45 days, against
the present 31 (`settlementDays`). The 45 days are
max(access limit + reporting target + absolute limit), which is routine's
30 d + 1 d + 14 d. Every study of an order wanted by H and imaged
within its access limit would then be decidable. INSUFFICIENT would
remain only for orders imaged more than 30 days past H, which are
already untimely under Gate A. The option lengthens the evaluation
record's observation, not the simulator's behaviour.

### 2.6 What is unchanged

These remain acquisition cohorts of the gated period:
* the per-seed severe rates;
* the local severity windows.

A plan could still move a severe report that does not breach the limit
past H, out of those rates. Such a report stays within the absolute
limit, which now covers it. The protocol carries this as a stated
limitation.

## 3. Determinism and history

* **Simulator and replay.** The simulator's behaviour and random
  streams are unchanged; the export adds the link only. Historical
  replay is exact (dev.15 to dev.20).
* **v4.** v0.2-eval-4's evaluation is byte-identical
  (`TestV4Neutrality`).
* **Older candidates.** The 5C.4d, 5C.4e and 5C.4f candidates' exact
  bytes still load and keep their rules:
  * `TestOldV5PolicyCompatibility`, `TestOld5C4ECompatibility`;
  * the 5C.4f columns above.
