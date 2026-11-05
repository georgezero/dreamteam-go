# V0.13b comparison contract: `dreamteam.management-alternatives/v1` as implemented

Status: implemented in 0.13.0-dev.2, `internal/alternatives/comparison.go`
and `evaluation.go`. It implements the V0.13a proposal
(`docs/v13a/COMPARISON_CONTRACT.md`) with the changes listed in §6.

## 1. Classes

| Class | Meaning | Audience |
|---|---|---|
| A1 | read from an engine field (`source` names it) | agent, evaluation |
| A2 | computed from engine fields by a stated rule (`derivation`) | agent, evaluation |
| B | read from a branch under a declared continuation | evaluation only |
| C | relevant, not available: key and reason, no value | agent, evaluation |

Every value has a `key`, a `label`, a `class`, a `unit` (cents, count,
days, seconds, date, text) and a `source`. A2 values have a
`derivation`. Engine forecast lines are kept as **estimates**, class
A1, with the engine's own "estimate" basis.

## 2. The audience boundary, in code

**The agent audience (`Agent`):**
* It is its own Go type.
* No field can hold a projection, a continuation, a horizon, a branch,
  the seed or the record checksum.
* `BuildAgent(session, selection)` reads only the decision point:
  * the engine's validation;
  * `PreviewStaffing`;
  * `PreviewSlotsClosed`;
  * the observation, roster plans and schedule.
* It never builds or reads a branch, and it checks that the session's
  digests are unchanged.

**`DecodeAgent`:**
* refuses any unknown field, so an injected `projected`, `continuation`
  or `seed` is refused;
* refuses a class other than A1, A2 or C, even when the digest was
  recomputed to match;
* refuses a missing source, label, unit or derivation;
* refuses a refused alternative that shows values;
* refuses an action type outside the eight existing ones;
* refuses a validation that contradicts its results;
* refuses any key or label with a ranking or verdict word: score, rank,
  optimal, best, recommend, preferred, better, worse, saving, harm;
* requires the proposal first and no action second;
* requires a `compared` table that follows from the alternatives;
* requires a matching SHA-256.

**`VerifyAgent`:** rebuilds the comparison from the session and
requires it equal, byte for byte.

**The evaluation audience (`Evaluation`):**
* It is built only after the decision, by `BuildEvaluation(record,
  agent, options)`.
* It holds the agent comparison exactly as shown, then:
  * the source (seed, record checksum, compatibility, digests);
  * the continuation and its SHA-256;
  * a fixed note on what a projection is not;
  * the horizon;
  * each branch's identity, replay verification, digests at both ends,
    class B projections and class C items;
  * paired differences.
* `DecodeEvaluation` and `VerifyEvaluation` apply the same rules, and
  require every branch to be replay-verified, to match its alternative,
  and to share one continuation, horizon and source.
* Building the evaluation does not change the agent comparison. A test
  rebuilds it after branching and compares.

## 3. Termination versus keep

**Terminate (A1 unless marked):**
* last day;
* end date;
* notice days;
* severance recognized now, with its payday;
* pending roster versions voided (A2, a count);
* approved extensions voided (A2, a count);
* displaced bookings (A2, the number of displacements listed);
* every guaranteed money line;
* slots closed: total, booked now and by modality, with the window;
* the engine's estimate "contract cost no longer owed" (an estimate).

**Keep (`no_action`):**
* severance recognized now: 0 (A2: no termination is committed);
* displaced bookings: 0 (A2);
* slots closed: 0 (A2);
* current employment stage;
* contracted weekly minutes;
* rostered minutes in weeks A and B of the roster in effect;
* pending roster versions;
* approved extensions.

**Obligations that continue under no action:**
* severance if terminated later (A1, `observe.staff[].severanceCents`);
* the weekly contract guarantee (A2): read from the proposal's preview
  forecast line, and cross-checked against the observed severance
  divided by two, within one cent. If the line is absent or the check
  fails, the guarantee is listed as unavailable, never estimated.

**For every accepted alternative:**
* appointments booked ahead (A2);
* payroll accrued, with its next payday (A1);
* accounts payable (A1).

These are the same under every alternative, so no alternative is shown
as cost-free.

**Not available (C):**
* replacement acceptance, when bookings are displaced;
* slots beyond the booking horizon;
* demand for closed slots;
* future orders and revenue;
* what the engine does not model.

No speculative future saving is computed.

## 4. Compared values

`compared` lists every immediate number that every accepted alternative
reports under the same key and unit. A1 and A2 values compare: both are
exact at the decision point.

In the termination case the table has four rows:
* severance recognized now;
* displaced bookings;
* slots closed;
* appointments booked ahead.

A refused alternative reports nothing and takes no part. A missing value
is never treated as zero; zero appears only where the engine reports it
or a stated derivation gives it.

## 5. Projections and differences (evaluation only)

**Flows** (decision to horizon: the horizon value minus the decision
value of the cumulative window metric):
* orders received, booked and lost;
* appointments cancelled;
* walk-ins arrived;
* exams;
* reports finalized;
* net revenue, wages, benefits, severance, operating expense and
  operating profit (the evaluation window's convention).

**Waits** (50th and 90th percentile to exam): over the window from the
takeover to the horizon, because percentiles do not subtract.

**At the horizon:**
* cash;
* lowest cash since day 1;
* days below zero;
* payables;
* severance payable;
* appointments booked ahead;
* studies unread.

If the scenario's evaluation window ends before the horizon, the flows
are listed as unavailable.

**Differences:**
* Each projected number is paired with the reference branch (`no_action`
  when listed): every branch's value, and its value minus the
  reference's.
* The label states the continuation and the horizon, and that the
  difference is not a gain, a loss or a verdict.
* No metric is combined with another.

## 6. Changes from the V0.13a proposal

1. **Values.** V0.13a's `immediate` and `projected` lists became `facts`,
   `estimates` and `obligations` (agent) and `projected` (evaluation), to
   match the presentation's sections.
2. **Compared table.** `comparisons.immediate` became `compared`; A1 and
   A2 compare with each other.
3. **Decision point.** The agent's decision point carries the state
   digest, not the seed or the record checksum, which are
   evaluation-only.
4. **Provenance.** Every engine request (`observe`, `validate <id>`,
   `preview <id>`, `slotsClosed <id>`, `rosterPlan <id>`) is recorded
   with the SHA-256 of its answer.
