# V0.13c presentation review package (human review)

**Status: PENDING.** No human reviewer has reviewed this package.

* The owner's decision requires a genuine human review of the
  agent-facing texts before the protocol is locked, and before any live
  inference.
* No reviewer identity is recorded here, because none exists yet. No
  reviewer label has been invented, and no automated review stands in
  for this one.
* The protocol stays unlocked (`PROTOCOL_DRAFT.md`) until a reviewer
  completes §5 and the owner accepts it.

## 1. What is reviewed

| Text | File | SHA-256 |
|---|---|---|
| Treatment: the comparison's summary, and each alternative's detail on `inspect:<id>` (`dreamteam.alternatives-presentation/v1`) | `internal/alternatives/text/presentation-v1.tmpl` | `5bf586cf12c83c157a8f31db05cd8f5cc68c0503b3bf99b1a89102198313b89f` |
| Control: the final-decision turn (`dreamteam.alternatives-control/v1`) | `internal/alternatives/text/control-v1.tmpl` | `331a1542ed79e15fcf10965fb924b134cd5ca206f740be96a51d4ce49f65d5a9` |
| The candidate rule the alternatives come from (`dreamteam.alternative-selection/v2`) | `internal/alternatives/text/selection-v2.txt` | `5f32b2c84a0b8383f1a213ce52745e56f5d5aa8292bc714096d1beec70debcf9` |
| The value labels, bases and derivations the templates print | `internal/alternatives/comparison.go` (`keep`, `practice`, the termination and preview items) | see the snapshot |

**Rendered examples:** every text exactly as a model would receive it.
* Each sensitivity-pair member has its control turn, its treatment
  summary and every inspect answer in
  `docs/benchmark/v13c-dev/turns/S<n>-<a|b>.txt`:

| File | Content |
|---|---|
| S1-a.txt (= S6-a.txt) | terminate Sam with 14 days' notice, Oct 13: 0 displaced, 2,481 slots |
| S1-b.txt | the same, Oct 27: 18 displaced |
| S2-a.txt, S2-b.txt | terminate Riley, Feb 16 or Feb 9: 0 or 1 displaced |
| S3-a.txt, S3-b.txt | terminate Casey or Morgan: severance $4,500.00 or $4,345.00 |
| S4-a.txt, S4-b.txt | terminate Casey, Jan 26 (adjusted listed) or Feb 2 (not listed) |
| S5-a.txt, S5-b.txt | teleradiology, three windows or weeknights only |
| S6-b.txt | terminate Sam, Jan 26: 362 slots |

* The historical points' texts are made from their recorded states at
  run time. Their identities are in `points/inventory.json`
  (`treatmentMessageSha256`, `controlMessageSha256`).
* The V0.13b termination example, with its evaluation-only projections
  shown separately, is in `docs/benchmark/v13b-dev/termination-presentation.txt`.

## 2. What the reviewer is asked to judge

For each item: acceptable as is, acceptable with a stated change, or
not acceptable, with reasons.

1. **Neutral wording.** Does any text favour an alternative: its order,
   "your proposal", "irreversible", "nothing to reverse", "no action
   rehires"?
2. **Readability.** Can a manager read the summary and an inspect answer
   and find a figure? Consider the ISO date-times, raw JSON actions,
   item counts and the "·" separators.
3. **Financial terminology.** Are "severance recognized now", "contract
   pay to the last day (minimum)", "contract cost no longer owed
   (estimate)", "contract guarantee continues", "payroll accrued" and
   "accounts payable" clear and correct?
4. **Missing information.** Is anything a manager would need at the
   decision point absent, that the engine reports?
5. **Reversibility.** Is each alternative's reversibility stated fairly?
6. **Displaced bookings.** Are the count, "booked now" and the
   replacement estimate clear?
7. **Capacity loss.** Is "scheduled slots no longer staffed (booked or
   not) … capacity, not orders" clear enough that it is not read as lost
   patients or revenue?
8. **No-action obligations.** Are the guarantee and later severance
   under no action stated so that keeping is not read as free?
9. **Potentially misleading comparisons.** Could anything be read as a
   saving, a loss, a ranking or a recommendation?
10. **Information leakage.** Does any text reveal the future, hidden
    state, the seed or the evaluation?

## 3. Points the preparer noticed

These were noticed while preparing the package. They are not decided:
the reviewer may accept, change or reject each.

* **P1, the citation format (owner decision D2).** The detail shows
  labels, not item keys, and the reply instruction does not describe how
  to cite values. A model cannot produce the structured citations that
  `compared` requires, so `compared` would not be observable. Possible
  remedy: show each item's key and add one neutral sentence describing
  an optional citation field. That is a template v2 with a new pin.
* **P2, severance twice.** A termination's detail shows severance twice:
  "Severance recognized now" from the termination preview, and
  "Severance … two contracted weeks" from the guarantee list. This may
  give it more weight than other items.
* **P3, a $0.00 estimate.** In S1-b, "Displaced bookings: $0.00 · at
  commit · 13 appointments lose coverage; 13 have a feasible
  replacement…" is an engine estimate of $0.00 next to 13 displaced
  bookings. It may read as "displacement costs nothing".
* **P4, engine jargon.** Some bases name engine internals:
  "already_terminating:", "observe.staff.severanceCents / 2",
  "the run result's bookedAhead".
* **P5, "(your proposal)" and "No alternative is a default."** These
  are neutral by intent. Whether "your proposal" anchors the decision is
  for the reviewer.
* **P6, detail counts.** The summary's "15 engine facts, 1 engine
  estimates…" may suggest that more items mean more consequence. The
  counts also read "1 engine estimates".
* **P7, the control text.** "Your proposal has been validated:
  accepted … Nothing has been applied yet" should not suggest the
  proposal is endorsed.

* **P8, framing (review F7, owner decision D8).** Only the control text
  says "Nothing has been applied yet"; the treatment summary does not.
  Should both say it?
* **P9, free text (review F1, owner decision D7).** Action fields such as
  `requestId`, and employee names, are printed as given. Is that
  acceptable in front of a model, or should the turn show neutral
  identifiers instead?

## 4. Independence

The reviewer must not have written or reviewed the templates or the
code. They should have no stake in the result, and should read the
texts without seeing any model's past decisions on these contexts.

## 5. Review record (to be completed by the reviewer)

| Field | Entry |
|---|---|
| Reviewer (name or stable identifier, as the reviewer chooses) | PENDING |
| Role and relevant experience | PENDING |
| Independence declaration (§4), in the reviewer's words | PENDING |
| Date | PENDING |
| Files reviewed, with SHA-256 (§1) | PENDING |
| Items 1–10: acceptable / with change / not acceptable, with reasons | PENDING |
| P1–P9: accepted / changed / rejected | PENDING |
| Verdict: the texts may be shown to a model as they are / after the stated changes / not | PENDING |

If any text changes, it gets a new identity and SHA-256, and the changed
text is reviewed again before the protocol is locked.
