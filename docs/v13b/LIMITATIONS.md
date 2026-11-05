# V0.13b known limitations

Status: 0.13.0-dev.2. These limitations hold for everything V0.13b
implements. The V0.13a limitations (`docs/v13a/LIMITATIONS.md`) all
still apply; this list adds what implementation showed.

## Projections (class B)

* **One realization.** A projection is exact for one seed, one decision
  point, this engine and the no-further-action continuation. It is not
  what would happen, and not a general effect of terminating.
* **An unrealistic continuation.** No management action follows the
  decision. A real practice keeps deciding, and under any other
  continuation the differences can change in size or sign.
* **The horizon.** Money differences recur weekly and capacity effects
  begin at end dates. The common horizon also moves with the
  alternatives listed: a later sibling end date lengthens every branch.
  Projections are compared only within one set.
* **Oracle information.** Projections use the simulator's realized
  future. They are evaluation-only; the agent audience cannot hold
  them. Showing them to an agent would need its own declared,
  oracle-labelled arm, which is not proposed.
* **Comparability.** The branches share the exogenous world: orders
  received and walk-ins were identical in every case tested. Within
  that world they still diverge through the engine's mechanics:
  * draws used in one branch only (an exam drawn only if it happens);
  * attempt numbers (a rebooking starts a new attempt);
  * draw contexts (the same number applied to another situation).

  These are the alternative's effect, as this engine produces it. The
  shared seed does not make a branch difference a causal estimate
  outside the simulator.
* **Accrual timing, tested.** Two accepted worklist changes that undo
  each other accrue the interval metrics at the decision instant. The
  only differences from no action are the action counts and the log,
  state and run digests that record the batch. The ledger and every
  reported metric are identical (`TestBranchAccrualPoint`). A refused
  batch gives the same result. This was tested for this one case; other
  accepted no-effect batches were not enumerated.

## Slots closed (E4)

* It is capacity, not demand: not every closed slot would have been
  booked, and a closed slot is not a lost patient.
* It counts only the booking horizon's materialized days. A termination
  removes capacity indefinitely; the count stops at the horizon.
* It counts room slots by staffing coverage. Desk capacity (a
  front-desk termination) is not a room slot and counts 0. Shared
  technologist capacity at a given moment is not modelled in the count.
* It is available only for termination-only batches, as the owner
  approved. A batch mixing a termination with other actions has no
  slots-closed report. A roster batch keeps the preview's own
  `slotsClosed`.

## The comparison and the rule

* **Selection.**
  * The rule adjusts a hire by a later start only. V0.13a's
    premium-to-standard tier change is deferred: the action has no tier
    field, so the change would need the offer table.
  * The rule never substitutes another action type (for example a
    reduced roster instead of a termination). The agent may propose
    one as a revision.
  * A proposal that holds the same action twice (which the engine
    refuses) gives two identical leave-one-out candidates. Its comparison
    cannot be branched for evaluation. Fixing this changes the rule, so
    it awaits the owner (review F4).
* **Decoding is structural.** `DecodeAgent` and `DecodeEvaluation`
  check form, classes and sources. Only `VerifyAgent` and
  `VerifyEvaluation` bind a comparison to the engine, by rebuilding it.
* **Keeping.**
  * The weekly guarantee kept under no action is read from the
    proposal's preview and cross-checked against the observed
    severance. Without that line it is listed as unavailable.
  * The observation does not expose the benefits rate (G5 in V0.13a),
    so no independent derivation is attempted.
* **The agent audience is not secret.** The comparison omits the seed
  and the record checksum, but it carries the scenario fingerprint and
  the state digest. These identify the decision point; they do not give
  its future without the simulator, the seed and the record.
* **Citations.** A verified citation shows only that a reply quoted
  shown values correctly and named a difference. It is not evidence
  that the difference caused the decision.
* **Sensitivity pairs are real states.** They differ in more than the
  named fact, and every compared fact that differs is listed.

## Not done in V0.13b

* No live model was contacted, and no agent workflow was added. The
  template has never been shown to a model.
* No clinical suite, reserved or holdout seed, or historical live rerun
  was used.
* No human review of the template, the rule or the scripted cases has
  happened yet. It is required before any live use. The V0.12b 98-item
  review is still pending.
