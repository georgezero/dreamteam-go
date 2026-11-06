# V0.13d presentation contract: `dreamteam.alternatives-presentation/v2`

**Status: not reviewed by a human (PENDING).** No model may be shown
these texts until the human presentation review (PRESENTATION_REVIEW.md)
is complete and the owner accepts it.

## 1. Identities

| Text | File | SHA-256 |
|---|---|---|
| The template: both arms' shared head and foot, the treatment's comparison and detail, the inspect and retry answers | `internal/alternatives/text/presentation-v2.tmpl` | `PresentationV2Sha256` (in `turn.go`; see PROTOCOL_DRAFT.md §2) |
| The closed tables: every item's label, forms and basis; every unavailable item; every reversibility text; every reference description; the vocabularies | `ViewTableText()` in `view.go`, committed as `docs/benchmark/v13d-dev/presentation/view-table.txt` | `ViewTableSha256` |
| The arms | `dreamteam.alternatives-arms/v2` | |
| The view | `dreamteam.alternatives-view/v1` (INFORMATION_BOUNDARY.md) | |

V0.13c's texts are unchanged and still pinned: presentation v1
(`5bf586cf…`), control v1 (`331a1542…`), selection rule v2 (`5f32b2c8…`).
`TestTurnPinnedTexts` checks all of them.

## 2. Structure of the turn

```
head        (both arms, the same text)
comparison  (treatment only)
foot        (both arms, the same text apart from the IDs a reply may select)
```

**Head** (owner decision D8): "FINAL DECISION at <time>. The staffing
actions in your reply passed validation: the simulator would accept them
now. Nothing has been applied yet."

**Foot:** the reply schema, the same in both arms:
* `{"decision": "select", "id": ID, "reason": TEXT}`, with the IDs listed;
* `{"decision": "revise", "actions": [...], "reason": TEXT}`;
* `{"decision": "abandon", "reason": TEXT}`;
* when the reply has operations actions, one sentence on how each decision
  treats them (D10);
* an optional `"cite"` of up to three items, with the values as shown.

The IDs line is the only line of the foot that differs between the arms.
Control lists `proposal (your proposal)`; treatment lists every listed
alternative. That list is part of the alternatives information.

**Comparison** (treatment only): the alternatives in the fixed order, each
reference defined by the reply's action that names it, each alternative's
actions and validation, then one sentence on what a detail holds and how
to ask for it: `{"inspect": [ID, ...]}`, once.

**Detail** (on inspect): the alternative's actions and validation, then
facts, estimates, obligations that continue, reversibility, and what is
not known before deciding. Each item is shown with its key in brackets
(`[slots_closed]`), so that a reply can cite it.

**Fixed answers:** the inspect answer (the details asked for, in the order
asked, then "Reply with your decision"), the answer to an inspect asking
for no listed ID, the control arm's answer to an inspect ("There are no
details to inspect"), the once-only notice, and the retry text. All are in
the template.

`TestTurnSymmetry` checks, at every due member: the heads are equal; the
treatment is the control with the comparison inserted; and the feet differ
in the IDs line only.

## 3. Changes from presentation v1, and why

| Change | Why |
|---|---|
| The typed view replaces every free-text field | D7 |
| "Nothing has been applied yet" in both arms | D8 (V0.13c review F7) |
| The summary no longer prints item counts ("15 engine facts, 1 engine estimates") | P6: counts may suggest that more items mean more consequence |
| Each detail item shows its key | P1 and D2: citations become possible, so `compared` can be observed |
| A termination's severance is shown once; the repeated money line is checked equal and not shown again | P2 |
| The displaced-bookings line is shown as "K of N displaced bookings have a feasible replacement by capacity", not as a $0.00 estimate | P3 |
| Bases no longer name engine internals (`already_terminating:`, `observe.staff.severanceCents / 2`, `bookedAhead`) | P4 |
| Validation messages are not shown; refusals show the engine's code and the action | they embed names |
| Counts have thousands separators (2,481) | readability |
| The no-action reversibility reads "nothing to reverse · no batch is applied" | the v1 sentence mentioned a termination at non-termination points |
| The reply schema is one JSON object with a `decision` field | the same schema in both arms; replaces v1's `select:<id>` wording |

## 4. Wording rules

* No ranking, recommendation, score or verdict: V0.13b's forbidden words
  still apply to every key and label of the comparison, and the view's
  fixed texts were written to the same rule.
* The order is fixed and stated as saying nothing about any alternative;
  no alternative is a default.
* Capacity is stated as capacity: "Scheduled slots no longer staffed,
  booked or not … this counts capacity, not patients, orders or revenue",
  with "Of these, booked now" next to it, and the demand for closed slots
  listed as not known before deciding.
* Engine estimates are labelled as estimates.

Whether these rules are met is for the human reviewer to judge.
