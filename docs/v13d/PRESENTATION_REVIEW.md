# V0.13d human presentation review package

**Status: PENDING.** No human reviewer has reviewed this package.
* No reviewer identity is recorded, because none exists yet. No reviewer
  label has been invented, and no automated review stands in for this one.
  The independent technical audit (`docs/audits/v0.13d-independent-review.md`)
  is not a presentation review.
* The protocol stays unlocked, and no model may be shown these texts, until
  a reviewer completes §5 and the owner accepts it.
* V0.13c's package (`docs/v13c/PRESENTATION_REVIEW.md`) was never reviewed.
  This package replaces it: every text it covered has changed.

## 1. What is reviewed: the exact final agent-facing texts

| Text | File | SHA-256 |
|---|---|---|
| Presentation v2: the shared head and foot (both arms), the comparison and the detail (treatment), the inspect, once-only and retry answers | `internal/alternatives/text/presentation-v2.tmpl` (also `docs/benchmark/v13d-dev/presentation/presentation-v2.tmpl`) | `8327f254ed2e25865db781f5f1cdcc70db129ea0df72ab6d0ce924ddf79ce496` |
| The closed tables: every item label, form and basis; every "not known before deciding" reason; the reversibility texts; the reference descriptions | `docs/benchmark/v13d-dev/presentation/view-table.txt` | `ce7f0e238c2d8cf71bf87cf720f12b56bd667035971308ae9abae7399d66aba0` |
| The candidate rule the alternatives come from | `internal/alternatives/text/selection-v2.txt` | `5f32b2c84a0b8383f1a213ce52745e56f5d5aa8292bc714096d1beec70debcf9` |

**Rendered examples, exactly as a model would receive them:**
`docs/benchmark/v13d-dev/turns/S<n>-<a|b>.txt`. Each file holds the control
text, the treatment text and every inspect answer for one sensitivity-pair
member. The first lines of each file name the member and the record-only
mapping; those lines are never sent.

| File | Content |
|---|---|
| S1-a.txt (also S6-a) | terminate one employee with 14 days' notice, decided Oct 13: 0 displaced bookings, 2,481 scheduled slots no longer staffed |
| S1-b.txt | the same employee, decided Oct 27: 18 displaced bookings |
| S2-a.txt, S2-b.txt | terminate another employee, last day Feb 16 or Feb 9: 0 or 1 displaced |
| S3-a.txt, S3-b.txt | terminate one of two employees: severance $4,500.00 or $4,345.00 |
| S4-a.txt, S4-b.txt | a later-last-day alternative listed (Jan 26) or not (Feb 2) |
| S5-a.txt, S5-b.txt | teleradiology, three windows or weeknights only |
| S6-b.txt | terminate, last day Jan 26: 362 slots |

## 2. What the reviewer is asked to judge

For each item: acceptable as is, acceptable with a stated change, or not
acceptable, with reasons.

1. **Neutral wording.** Does any text favour an alternative: its order,
   "(your proposal)", "irreversible", "nothing to reverse", "no action
   re-hires", the reply instructions?
2. **Readability.** Can a manager read the summary and an inspect answer
   and find a figure? Consider the neutral labels (E1), the bracketed item
   keys, the ISO date-times, the "·" separators and the action notation
   (`action 1: terminate E1, lastDay 2026-10-27`).
3. **Financial terminology.** Are "severance recognized now", "contract
   pay to the last day (minimum)", "contract cost no longer owed
   (estimate)", "contract guarantee continues", "payroll accrued, next
   payday" and "accounts payable" clear and correct?
4. **Target identification.** With neutral labels, can a reader tell which
   employee, hire or contract each line is about, and match each
   alternative to the actions of the reply?
5. **Capacity loss.** Owner's question: **could a manager understand the
   difference between 2,481 scheduled slots no longer staffed and zero
   bookings currently displaced, without reading either as a prediction of
   future demand?** See S1-a.txt (proposal detail: `[slots_closed]`,
   `[slots_closed_booked_now]`, `[displaced_bookings]`, and "Demand for the
   closed slots" under "Not known before deciding").
6. **Displaced bookings.** Are the count, "of these, booked now", and "K of
   N displaced bookings have a feasible replacement by capacity" clear,
   and is it clear that acceptance is decided only at commit?
7. **Existing obligations.** Are the guarantee and later severance under
   no action stated so that keeping is not read as free, and the practice's
   payroll and payables as the same under every alternative?
8. **The no-action alternative.** Is `no_action` described fairly, and is
   its relation to `abandon` clear (select no_action keeps operations
   actions; abandon submits nothing)?
9. **Missing information.** Is anything a manager would need at the
   decision point absent, that the engine reports? (V0.13d removed the
   validation messages, the descriptions and the engine's own bases.)
10. **Potentially misleading values.** Could anything be read as a saving,
    a loss, a ranking or a recommendation? Consider "$0.00" values, counts
    without units, and "contract cost no longer owed".
11. **Information leakage.** Does any text reveal the future, hidden state,
    the seed, an evaluation value, an internal digest or infrastructure?

## 3. Points the preparer noticed (not decided)

* **R1, the bracketed keys.** Each detail line starts with its key
  (`[severance_recognized_now.E1]`), so that a reply can cite it. They may
  make the detail harder to read.
* **R2, a "$0.00" severance under no action** (`Severance recognized now ·
  E1: $0.00 · no termination is committed, so none is recognized`). It is
  correct, and may read as "no cost".
* **R3, "Contract cost no longer owed".** An engine estimate of the weekly
  guarantee no longer paid after the last day. It could be read as a
  saving.
* **R4, the per-modality slot counts** (XR, CT, US, MR) are shown under the
  total. They may draw attention away from "Of these, booked now".
* **R5, control and treatment length.** The treatment text is several times
  longer than the control text. The difference is the comparison itself.
* **R6, "(your proposal)".** The proposal is listed first and labelled.
  Whether that anchors the decision is for the reviewer.
* **R7, the cite instruction is in both arms.** The control arm has no
  detail to cite. The sentence is there so the foot is the same text in
  both arms.

## 4. Independence

The reviewer must not have written or reviewed the templates or the code,
should have no stake in the result, and should read the texts without
seeing any model's past decisions on these contexts.

## 5. Review record (to be completed by the reviewer)

| Field | Entry |
|---|---|
| Reviewer (name or stable identifier, as the reviewer chooses) | PENDING |
| Role and relevant experience | PENDING |
| Independence declaration (§4), in the reviewer's words | PENDING |
| Date | PENDING |
| Files reviewed, with SHA-256 (§1) | PENDING |
| Items 1–11: acceptable / with change / not acceptable, with written findings | PENDING |
| R1–R7: accepted / changed / rejected | PENDING |
| Verdict: the texts may be shown to a model as they are / after the stated changes / not | PENDING |

If any text changes, it gets a new identity and SHA-256, and the changed
text is reviewed again before the protocol is locked.
