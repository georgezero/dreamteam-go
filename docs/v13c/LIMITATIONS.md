# V0.13c known limitations

## Process

* **The human presentation review is PENDING.** No human reviewer was
  available. The protocol cannot be locked, and no model may see the
  texts, until it is done (PRESENTATION_REVIEW.md).
* **No live inference.** Nothing here measures any model's behaviour.
  Every result in `docs/benchmark/v13c-dev/` comes from the engine or
  from scripted rules.
* **The agent side is not implemented.** The workflow that sends the
  turns, parses the reply (select / revise / abandon / inspect) and
  records the draw is V0.13d's work. Its reply schema, identity and
  SHA-256 are pinned at lock.
* **The draft protocol JSON** (`protocol-draft.json`) was written from
  the committed evidence files by the preparer. Nothing re-derives it
  yet; the lock's generator and check will.

## Design

* **The boundary assumes trusted free-text inputs (V0.13c review F1).**
  The structural and lexical guards cannot tell what a free-text field
  means: an action's `requestId` or a scenario's employee name can carry
  any text the engine accepts. The protocol relies on every such input
  being a recording, a committed and pinned file, or the model's own
  reply (owner decision D7).
* **The framing differs (F7).** Only the control text says "Nothing has
  been applied yet" (D8).

* **The arms differ in more than the comparison's content:**
  * Treatment may spend one call on inspect.
  * Treatment's reply options include the listed alternatives.
  * Control shows only the proposal and its validation.
* **Matched control turn.** With D1 as recommended, control is not
  llm-agent 5 unchanged. It has a confirmation turn that no earlier
  experiment had.
* **`compared` is not observable with presentation v1.** The texts show
  labels, not item keys, and do not describe a citation field (D2, P1).
* **The reply's reason is recorded, never interpreted.** No claim is
  made about reasoning.
* **A change of decision is not an improvement.** Nothing scores
  decisions.

## Branch points

* **The pool is small.** It holds 20 recorded proposals, of which 5 have
  a due turn: 1 Gemma, 4 Qwen.
* **No historical termination without displacement**
  (`termination_keep`). Termination versus keep is covered only by
  constructed members.
* **No valid hire, and no credit-supported commitment where the turn is
  due.** No valid not-due proposal over $10,000 apart from the
  credit-supported one.
* **Prior exposure.** Every point was in an earlier experiment, because
  only those have committed contexts. The rule reads no response of
  those experiments.
* **Refused points.** The turn is due there only if the model corrects
  the proposal validly. Their contribution depends on the model.

## Sensitivity pairs

* **Constructed contexts.** Scripted proposals at no-action trajectories
  of one scenario and seed, with empty memory. No model made those
  proposals, and the contexts are not historical behaviour.
* **No pair isolates one fact.** Every other visible difference is
  listed (SENSITIVITY_CASES.md §3):
  * the best isolated pair is S5, with 6 value differences;
  * S1 changes the whole context, a two-week later state;
  * severance (S3) cannot vary without changing the employee;
  * S2's displacement contrast is a single booking.
* **One scenario (mini) and one seed (52)** for every constructed
  member.

## Engine and versions

* **The engine version is unchanged (0.13.0-dev.2).** V0.13c changes no
  engine, session or simulator file. Bumping the version would change
  the engine string embedded in V0.13b's evidence (V0.13b review F1),
  which the owner asked to keep unchanged. This departs from the
  per-step version bump and is recorded in `docs/PROGRESS.md`.
* **Batch order is part of a batch in rule v2.** Two candidates with the
  same actions in another order are both kept. Treating order as
  irrelevant would need another rule version.
* **V0.13b's F6–F10 notes still apply.** Rule v2 keeps v1's text
  unchanged apart from its name and the duplicates section, including
  the sentence F6 noted.
