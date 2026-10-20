# Slice 5C.4j: clinical safety remediation

These are the clinical-policy changes made in answer to the independent
5C.4i audit
(`docs/audits/slice-5c4i-boundary-safety-lock-readiness-audit.md`),
under the owner's decisions of 2026-10-19. The evaluator is
0.2.0-dev.22. The changes:

* **I1:** the partial-window rule becomes an observed severe-rate
  ceiling.
* **I2:** the order–study integrity contract.
* **I3:** the 45-day follow-up.
* **I4:** follow-up randomness that leaves earlier outcomes unchanged.

**Status:**
* **Not locked, not frozen, not reviewed.** This is a revised clinical
  policy, so no earlier independent component GO carries over to it. It
  needs the independent 5C.4k review before any lock.
* **Unchanged:** the severity thresholds, rates, minimums, absolute
  limits, anchor, Gate A, Gate B and its margin, the safeguard and the
  allowance.
* **Older candidates:** they keep their rules byte for byte, kept as
  `internal/eval/testdata/candidate-policy-v5-5c4{d,e,f,h}.json`.

The candidate's new values:

```json
"partialWindows": "rateCeiling",
"safetyCohort": "ordersByHorizon",
"followUpDays": 45,
"orderStudyIntegrity": "contract"
```

## 1. Partial windows: the observed severe-rate ceiling (I1)

### 1.1 The rule

A partial window is a leading or trailing calendar window not wholly
within the gated period. Each nonempty partial window is judged on its
own acquisitions, never pooled:

| Partial window | Status |
|---|---|
| severe ÷ acquired **above 5%**, whatever its size | **FAIL** |
| at or below 5%, fewer than 100 acquisitions | **NOT_JUDGED**: disclosed, establishes nothing |
| at or below 5%, 100 or more | PASS (judged like a complete window) |
| no acquisitions | NOT_JUDGED |

The rest is unchanged:
* a complete window below 100 is INSUFFICIENT;
* a priority with no judged window and no insufficient one is
  INSUFFICIENT (`noJudgedWindow`);
* a seed's window gate fails when any window fails, and its failures
  count in the shared allowance like any seed-level clinical failure.
  The rule is not an unconditional veto.

**What a FAIL means.** It is a safety veto on an observed concentration,
not a statistical claim that the underlying rate exceeds 5%. It is
deliberately conservative: one severe report among six acquisitions
fails. A NOT_JUDGED window never counts as a pass: the passing gate's
reason names it, "partial windows below the minimum not judged, each at
or below the rate (too few to establish it)".

### 1.2 The owner's table, on the full 40-seed evaluation

`TestRateCeilingFullVerdicts`: every candidate seed's leading partial
week holds n soon acquisitions, s of them 30 h past due. The baseline is
the same practice without the severe reports.

| s / n | Local | Full verdict (5C.4j) | 5C.4h (count ceiling) |
|---|---|---|---|
| 5 / 6 | FAIL | **NO GO** (40 failing seeds) | GO |
| 5 / 10 | FAIL | **NO GO** | GO |
| 5 / 50 | FAIL | **NO GO** | GO |
| 5 / 99 | FAIL | **NO GO** | GO |
| 6 / 99 | FAIL | NO GO | NO GO |
| 5 / 100 | PASS (judged, exactly 5%) | GO | GO |
| 6 / 100 | FAIL | NO GO | NO GO |
| 0 / 6 | NOT_JUDGED | GO | GO |
| 1 / 6 | FAIL | **NO GO** | GO |
| 16 / 24 (the 640-report exploit) | FAIL | NO GO | NO GO (GO under 5C.4f) |

### 1.3 Window cases (`TestRateCeilingWindows`)

| Case | Statuses |
|---|---|
| empty partial windows | not_judged … not_judged |
| two partial windows, 1 of 20 each (exactly 5%) | not_judged pass pass not_judged |
| two partial windows, 1 of 19 and 0 of 19 | fail pass pass not_judged |
| two partial windows, 5 of 6 each | fail pass pass fail |
| not pooled: 1 of 19, then 0 of 81 elsewhere | fail pass pass not_judged |
| a complete window of 99 | insufficient |
| a partial window of 100 at 6% | fail |
| no window judged (short horizon, 1 of 20) | INSUFFICIENT, with its reason |

### 1.4 Stable-practice false rejection

The owner asked for the stricter rule's effect on stable practices,
reported without changing any threshold. Measured on the development
family PCG(20261021, ·) with the final model (45-day follow-up,
prefix-stable draws; `construction-checks.json`):

| Stable practices (S-A, S-B, S-C, S-Cs) | Groups rejected | Seeds failing the partial-window ceiling | Seed-level clinical failures (allowance 4 of 40 per group) |
|---|---|---|---|
| 85% scenarios, 200 seeds each (20 groups) | **0 of 20** | **0 of 800** | 19: timely access (urgent) 16, backlog 3; at most 2 in a group |
| 85% and 92% scenarios, 40 seeds each (8 groups) | **0 of 8** | **0 of 320** | 13: timely access (urgent) 6, soon breach rate 5, backlog 2; at most 4 in a group |
| the same, 5C.4h count ceiling (31-day follow-up) | 0 of 20 and 0 of 8 | 0 | 19 and 10 |

* **No material increase.** Across 1,120 stable seeds, the partial-window
  rate ceiling never fired. The per-gate failures on the 200-seed pass
  are identical to 5C.4h's. The 40-seed pass differs by three seeds, all
  on gates the ceiling does not touch: the longer, prefix-stable
  observation changes the fidelity model's later draws, not the rule.
  So there was no reason to stop for owner review on this count.
* **Where it fires.** Only in harmful scenarios, each already rejected
  under 5C.4h on other rules: Wg85 (28 seeds), WgU85 (22), R085 (29),
  D-85 (16), K-92 (27), K-85 (1), L20-85 (1), and the W-PARTIAL witness
  (40 of 40). The count ceiling fired on 4 of these. No scenario's group
  verdict changed between 5C.4h and 5C.4j (34 of 49 rejected in both).
* **The simulator's no-action practice** (seeds 901–910, §3.1).
  *Corrected in Slice 5C.4l (audit 5C.4k K5); this said "no partial
  window fails", which was wrong.* The leading partial soon window fails
  the ceiling on 3 of 10 seeds: 901 (3 of 42), 902 (3 of 41) and 903
  (4 of 41). Those seeds already failed the soon window gate on complete
  blocks, so no seed's status and no gate's status changed; the
  aggregate comparison hid the new local failures
  (`../clinical-5c4l/clinical-provenance.md` §4).
* **Limitation.** These are the manifest's stable practices on the
  development family, not a population. A practice with sparse,
  front-loaded soon demand would fail on one severe report in a short
  partial week. That is the owner's deliberate conservatism, and the
  shared allowance absorbs isolated cases.

## 2. The order–study contract (I2)

The authoritative relationship, enforced under
`orderStudyIntegrity: contract` (`checkOrderStudies`; only with the
follow-up cohort): a study is the imaging of one order, and the order's
resolution is that study's acquisition. For new work:

* every study acquired after the takeover has a **unique ID** and names
  **exactly one order** of the record;
* that order has **the same priority and modality** (the request);
* its final outcome is **served**;
* **its resolution time is the study's acquisition time**;
* **no two studies name one order;**
* **every order received after the takeover and served has exactly one
  study.**

Any breach makes the record **INVALID**, never GO. The contract has two
exemptions, both inherited work that the records cannot reconcile:
* a study acquired at or before the takeover, whose order may have been
  resolved before it and be absent from the record;
* an order received at or before the takeover and served after it
  without a post-takeover study.

`TestOrderStudyContract` covers the audit's manipulations of a
post-horizon violation, together with further breaches of the contract:

| Manipulation | 5C.4j | 5C.4h |
|---|---|---|
| missing, unknown or ambiguous order link | INVALID | INVALID |
| **duplicated study row** | **INVALID** | NO GO, counted twice |
| **study relinked to an order wanted after the horizon** | **INVALID** | **GO** |
| **study deleted, its order still served** | **INVALID** | **GO** |
| repeated study ID; study of another modality; resolution other than the acquisition; study of an order not served | INVALID | — |
| a served new order with no study | INVALID | — |

Inherited work is valid as the fixture records it
(`TestOrderStudyInheritedExempt`).

* **Simulator.** Its records satisfy the contract once the recorder
  exports rebooked no-shows (§3.1): the development seeds evaluate as
  valid (`sim-access-5c4j.json`).
* **Fidelity model.** It now gives its studies unique IDs.
* **Older candidates do not apply the contract.** The 5C.4h candidate
  keeps its own link check, under which the audit's three bypasses
  reproduce exactly as reported. The 5C.4f and older candidates have
  neither check.

## 3. The 45-day follow-up (I3)

`followUpDays: 45`: every record is observed at least 45 calendar days
past the horizon H, beside the policy's other needs (T180 and closure,
settlement, backlog). The end of observation is E = H + 45 days.

* **Validation.** The policy refuses a follow-up shorter than access
  limit + reporting target + absolute limit for any priority:

  | Priority | Access limit + reporting target + absolute limit |
  |---|---|
  | urgent | 2 d + 1 h + 24 h |
  | soon | 7 d + 4 h + 48 h |
  | routine | **30 d + 1 d + 14 d = 45 d** |

  It also refuses a follow-up without the follow-up cohort.
* **Where it applies.** `observationNeeded` adds the follow-up. Every
  recorder that uses it observes through E:
  * the simulator's evaluation recorder (`eval.ObservationEnd`);
  * the fidelity model;
  * the synthetic fixture.

**Time-boundary conventions** (seconds, exact). The due time is the
acquisition plus the priority's reporting target. The deadline is the
due time plus the absolute limit. E is the end of observation.

| State at E | Condition | Result |
|---|---|---|
| observed violation | finished, finish − due > limit | FAIL |
| compliant | finished, finish − due ≤ limit (**equality passes**) | — |
| unresolved overdue | unfinished, E − due > limit | FAIL |
| not yet observable | unfinished, E − due ≤ limit (**a report due exactly at E and unfinished stays here**) | INSUFFICIENT |
| finished after E | a record claiming it | INVALID (outside the observation) |

**Extending the follow-up fabricates no outcome.** An unfinished report
remains unfinished. An order still unimaged at E has no reporting clock
and is counted `unacquired`; Gate A judges it.

`TestFollowUp45` covers the audit's I3 table at E = H + 45 d:

| Case | Safety |
|---|---|
| acquired H + 1 s, 60 h past due | FAIL |
| acquired before H, reported H + 1 s, 60 h past due | FAIL |
| acquired H + 1 s, exactly 48 h past due | PASS |
| one second beyond | FAIL |
| soon, acquired E − 53 h, unfinished | FAIL (overdue) |
| soon, acquired E − 52 h, unfinished (due exactly at E) | INSUFFICIENT |
| soon, acquired E − 10 h, unfinished | INSUFFICIENT |
| routine wanted at H, acquired at its 30-day access deadline, finished exactly at E | PASS |
| the same, unfinished | INSUFFICIENT |
| routine acquired H + 30 d + 1 s, unfinished | INSUFFICIENT |
| routine acquired H + 29 d, unfinished | FAIL (overdue) |
| a report claimed one second after E | INVALID |

**Censoring that cannot be avoided.** A finite follow-up cannot observe
every eventual deadline:
* a report due exactly at E and unfinished;
* orders imaged after their access limit, late in the follow-up;
* orders still unimaged at E.

These stay INSUFFICIENT or are judged by Gate A, never PASS. They are
limitations of finite observation, not defects.

### 3.1 Effect on the development seeds

`sim-access-5c4j.json`: the no-action practice on seeds 901–910,
recorded under the revised candidate and observed 45 days past H
(31 days in 5C.4h):

| Routine, orders wanted by H | 5C.4h (31 days) | 5C.4j (45 days) |
|---|---|---|
| studies not yet observable (INSUFFICIENT) | 22 on 9 seeds | **3 on 3 seeds** (1 each) |
| orders still unimaged at E (Gate A) | 89 | **24** |
| observed violations | 0 | 0 |

* **Seed-level results unchanged.** Every seed has the same clinical
  status and the same failing gates as in 5C.4h. Urgent absolute
  violations occur on 8 seeds (1–5 each). *Corrected in Slice 5C.4l:
  this said that no partial window fails; the leading soon window fails
  on seeds 901, 902 and 903 (§1.4), each already failing the soon window
  gate.*
* **A recorder defect found and repaired.** The order–study contract
  exposed it. The evaluation recorder exported only the orders open at
  the takeover. An order whose patient missed an appointment just before
  the takeover, with the call for a new date already queued, was left
  out. When that patient later came back and was imaged, the study named
  an order absent from the record: INVALID on 8 of 10 seeds, 1–3 studies
  each. The recorder now exports those no-shows as inherited orders
  (`export_eval_test.go`), reading the queued events and drawing nothing.
  `TestFollowUpPrefixUnchanged` asserts that seed 901's 45-day record
  keeps the contract; with the repair reverted it fails. The recorder is
  test code: no shipped behaviour, digest or replay changes.

## 4. Follow-up randomness (I4)

**The defect.** The fidelity model drew each study's documentation and
read times from the booking stream after drawing every order of the
observation. A longer observation therefore shifted every earlier
study's times: the audit found about 7,800 changed final times.

**The repair** (`prefixStableDraws`, set in every manifest practice):
each study's two draws are mixed from a booking-stream salt and the
order's index. They are fixed per entity, so they no longer depend on
how many orders the observation holds. Booking and demand draws are
made in time order, so they are already prefix-stable.

`TestFidelityFollowUpPrefixStable` compares the A85 manifest practice
observed 31 and 45 days past H. Up to the earlier end, everything
agrees:
* every order's demand, modality, request and resolution;
* every study's acquisition, documentation, read and final time.

The committed model, kept for its pinned digests, is shown changing.

**Simulator.** `TestFollowUpPrefixUnchanged` records development seed
901 to 31 and to 45 days past H. Up to the earlier end everything is
identical:
* every order and its resolution;
* every capture's outcomes;
* every study's timestamps.

The simulator's decisions and random streams do not depend on how long
the recorder watches. Its export draws nothing. Historical replay is
exact.
