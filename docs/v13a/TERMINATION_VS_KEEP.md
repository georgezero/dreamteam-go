# V0.13a termination versus keeping the employee

Status: V0.13a (0.13.0-dev.1), design only.

**The decision:**
* **A:** terminate an hourly employee.
* **B:** keep current staffing.

**Sources:**
* The engine at `8bbeca2`.
* One read-only probe through the agent protocol: `validate`, `preview`
  and `digests`, with no `apply`.
* One pair of branch runs (§5). Their provenance is in
  `docs/audits/v0.13a-baseline.md`.

**The example:**
* scenario `mini`, seed 52 (a development seed outside the sealed
  holdout);
* the takeover at 2026-10-13 06:00, advanced to 07:00;
* employee E04 (Sam), an XR/CT technologist at $40.00/h with a
  contract of 2,370 minutes a week.

These values illustrate the inventory. They are not evidence about
whether terminating anyone is a good decision.

## 1. What each side is

**A, terminate:**
* An accepted `terminate` with `lastDay` 2026-10-27, the earliest
  allowed (14 days).
* It is irreversible at commit (`already_terminating` refuses any later
  change).

**B, keep:**
* No staffing action. Every existing obligation continues.
* B is a decision too:
  * **B is not cost-free.** The contract guarantee and the severance
    exposure continue.
  * **B is not risk-free.** Its later consequences are as uncertain as
    A's.

## 2. A: exact values at the decision point

All are class **A1**, exact immediate values reported by the engine (see
`COMPARISON_CONTRACT.md`).

| Item | Value | Engine field |
|---|---|---|
| Employee identity | E04, Sam | `terminations[].employee`, `.name` |
| Last working day | 2026-10-27 | `terminations[].lastDay` |
| End date (first day not employed) | 2026-10-28 | `terminations[].endDate`, `results[].effectiveAt` |
| Notice period | 14 days | `terminations[].noticeDays` |
| Severance | $3,950.00, recognized at commit | `terminations[].severanceCents`; guaranteed line `Severance · Sam (E04)` |
| Severance payment date | 2026-10-30 | `terminations[].severancePayday` |
| Contract pay to the last day | $4,350.00 minimum (worked pay replaces it) | guaranteed line `Contract pay to the last day · Sam (E04)` |
| Displaced appointments | 0 | `terminations[].displaced`, `len(displacements)` |
| Voided rosters and extensions | none | `voidedRosters`, `voidedExtensions` (absent when empty) |
| Funding test | none: no test, because a termination commits no new spending | `funding` absent |
| Roster changes | the employee leaves every roster from 2026-10-28 | derived from `endDate`; there are no per-day roster entries |
| Capacity changes | **not reported** | `slotsClosed` is absent for a termination-only batch (§4) |
| Existing obligations | the guarantee and pay to the last day as above; other obligations are unchanged | `guaranteed[]` |
| "Contract cost no longer owed" | $1,975.00 per full week from 2026-10-28 | `forecast[]`, labelled "estimate" by the engine |

The forecast line is the only money line about after the end date. The
engine labels it an estimate, and it omits:
* overtime and hold-over (the engine says so);
* revenue from the exams the employee would have performed;
* the work that others must absorb.

It must not be presented as a saving (§6).

## 3. B: exact values at the decision point

Class **A1**, read from the observation the agent already receives
(`observe`, section `staff`) and from the roster.

| Item | Value | Source |
|---|---|---|
| Current employment status | `active` | `staff[].employment` |
| Current roster | 11:30–20:00 today; the two-week rota | `staff[].shift`, `rosterEnd`; `RosterPlan(E04)` |
| Rate and contract | $40.00/h; 2,370 min a week | `staff[].hourlyCents`, `contractMin` |
| Existing salary obligation | $1,975.00 per full week, guaranteed with benefits | class A2: the engine's weekly guarantee, read from A's `forecast[]` line (the observation gives the rate and contract, but not the benefits rate, so B cannot recompute it exactly; see G5) |
| Severance exposure | $3,950.00, owed if the employee is ever terminated | `staff[].severanceCents` |
| Current capacity | the employee's rostered shifts and the rooms they staff | `RosterPlan`, `schedule` |
| Existing booked appointments | those in rooms the employee covers stay covered | `schedule(offset)`; nothing is displaced under B |
| Existing commitments | approved extensions and pending rosters stay in force | `extensions`, `RosterPlan` |

**Three points about B:**
1. **B keeps the guarantee.** Under B, $1,975.00 a week continues
   until some later decision. This is a recurring obligation, not a
   saving.
2. **B defers severance but does not remove it.** The engine has no
   resignation, retirement or attrition: an hourly contract ends only
   by `terminate`. If the practice terminates later, it pays the same
   two weeks' severance, recomputed from the contract then in force.
3. **The two sides are not symmetrical in time.** A creates new events:
   severance posted now, paid 2026-10-30, and coverage ending
   2026-10-28. B creates nothing new. Its costs are the continuation of
   those already reported.

## 4. Gaps

**G1: capacity lost by a termination is not previewed.**
* `PreviewStaffing` computes the plans under the proposed registry for
  every batch.
* It turns them into `slotsOpened` and `slotsClosed` only inside
  `previewRosters`, so only for batches that change a roster.
* For a termination-only batch, the slots that stop being bookable in
  the booking horizon are therefore not reported. `displacements` shows
  only the slots already booked.
* In the example, 0 bookings are displaced, but every unbooked slot
  that E04 alone staffed from 2026-10-28 on is lost.

**G2: displacement resolution is known only at commit.**
* Whether each displaced patient accepts a replacement depends on the
  patient's tolerance, a hidden attribute.
* The preview's provisional date assumes every patient accepts. This is
  correct and labelled, but it is not the outcome.

**G3: no downstream money for either side.**
* Neither the preview nor the observation projects revenue, lost
  demand or overtime after the end date.
* This is by design: the engine reports direct consequences only
  (`V0.12_CONSEQUENCE_INVENTORY.md`, class C).

**G4: operations alternatives are not previewable.**
* An `extend_shift` alternative, for example "cover the gap with
  overtime instead", can be validated but has no preview.

**G5: a kept employee's weekly guarantee is not a field.**
* The engine computes it (`weeklyGuarantee`), but reports it only inside
  money lines:
  * the termination preview's forecast line;
  * a roster preview's guaranteed line.
* The observation gives the rate, the contract and `severanceCents`,
  but not the benefits rate.
* `severanceCents / 2` agrees with the guarantee only to within one
  cent of rounding.
* B's value must therefore be read from the proposal's preview (class
  A2, with the derivation stated), not recomputed.

**What can close G1:**
* Option 1: a read-only, opt-in preview field.
* Option 2: a branch, which applies the termination on a copy and reads
  the coverage of each later day.

The coverage is a deterministic function of the registry and the
roster. Reading it uses no hidden attribute.

G2 and G3 can be closed only by projecting the future (§5 and
`CONTINUATION_DESIGN.md`).

## 5. A deterministic projection of both sides

**The runs:**
* Both sides were run on the existing CLI, `dreamteam run -scenario
  mini -seed 52 -days 45`:
  * A with the termination at 07:00;
  * B with no action.
* Neither run took any later decision.
* Both were saved, replayed and verified (`dreamteam verify`: ok).
* Rerunning A reproduced its result exactly.

**Provenance:**
* `docs/audits/v0.13a-baseline.md` §3 has the command lines and SHA-256
  hashes.
* The working files are outside the repository and are not evidence.

**Status of these numbers:**
* They are class **B**: deterministic consequences under a declared
  continuation ("no further management action for 45 days").
* That continuation is unrealistic, because no manager would take no
  action for 45 days.
* It is a single seed and a single scenario.
* The numbers show what a projection can contain. They are not a
  finding about terminations.

| Over the 45 days | B, keep | A, terminate |
|---|---|---|
| Orders received | 2,224 | 2,224 (identical: §6 of `CONTINUATION_DESIGN.md`) |
| Exams performed | 1,653 | 1,481 |
| Orders lost | 83 | 260 |
| Net revenue | $247,672.70 | $212,074.00 |
| Labor (wages) | $126,719.22 | $119,566.89 |
| Severance | — | $3,950.00 |
| Operating profit (baseline-adjusted, as the CLI reports it) | −$76,278.06 | −$101,144.50 |
| Cash at the end | −$31,110.51 | −$27,852.47 |
| Wait to exam, 90th percentile | 48 min 38 s | 43 min 21 s |
| Displaced at commit | — | 0 |

These numbers do not say which side is better, and they must never be
presented as if they did:
* A ends with more cash and shorter waits.
* B performs more exams, loses fewer orders and has a higher operating
  profit.

"0 displaced" at the decision point did not mean no later difference:
* 177 more orders were lost under A in this projection.
* The cause is capacity that was never booked (G1), not displacement.

## 6. Rules carried into the design

1. Never call keeping an employee cost-free. B's guarantee and
   severance exposure are shown with the same prominence as A's
   severance.
2. Never present "Contract cost no longer owed" as a saving.
   * It is an engine estimate per week.
   * It has no horizon and omits lost revenue.
   * A saving may be stated only as a difference between two class-B
     projections, with the horizon, the continuation and the accounting
     basis named.
3. Never infer patient harm from displaced bookings alone.
   * A displaced booking may be rebooked.
   * An unbooked slot lost may matter more.
   * The engine does not model clinical outcomes.
4. Never treat the cheaper side as the better decision.
