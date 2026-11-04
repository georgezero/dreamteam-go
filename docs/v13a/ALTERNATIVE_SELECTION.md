# V0.13a alternative selection: `dreamteam.alternative-selection/v1`

Status: **proposed** in V0.13a (0.13.0-dev.1). It is not implemented.

**The rule decides:**
* which alternatives are compared at a decision point;
* their order.

**How it behaves:**
* It is deterministic, small and built only from existing actions.
* The model never generates an alternative. It can only:
  * choose among the alternatives listed;
  * submit its own revision, which is handled like any other proposal
    (§5).

## 1. When it applies

The rule applies to a proposed staffing batch that contains at least
one action that is irreversible at commit and consequential:
* `terminate`;
* `hire`;
* `contract_teleradiology` with op `start`.

It also applies to a staffing batch whose preview displaces bookings.

These are the V0.12 safeguard's conditions (`V0.12_ACKNOWLEDGMENT.md`)
without the acknowledgment. Operations batches are out of scope,
because they have no preview (`ACTION_INVENTORY.md` §3).

## 2. Candidates

There are at most **four** alternatives, in this order.

| # | ID | Source | Content |
|---|---|---|---|
| 1 | `proposal` | the agent | The batch as proposed. Always present. |
| 2 | `no_action` | fixed | No batch: every existing obligation continues. Always present and always valid. |
| 3 | `without_<k>` | leave one out | Only for a batch of two or more actions: the proposal without its *k*-th irreversible action (`terminate`, `hire`, `contract_teleradiology` start), for the first such action in batch order, then the second, while fewer than four alternatives are listed. |
| 3 or 4 | `adjusted` | bounded adjustment | Only for a batch of one action: the single variant in §3, if the action type has one. |

**Withdrawing or reversing something not yet committed:**
* The proposal itself is not yet committed. Withdrawing it is
  `no_action`.
* When the proposal is itself a withdrawal, the alternative "keep the
  pending item" is also `no_action`. Examples:
  * `set_roster` or `set_reading_allocation` with op `remove`;
  * `contract_teleradiology` with op `cancel`.
* No separate candidate is needed.

**Validation of candidates:**
* Every candidate is validated by the engine (`validate`).
* A candidate the engine refuses stays in the list, with its codes, and
  counts against the four.
* It is never repaired or replaced.

**Why leave one out:** displacement is computed for the whole batch
(`V0.12_CONSEQUENCE_INVENTORY.md` §2). Removing one action shows how
much of the batch's displacement and cost it carries, using the engine's
own exact preview.

## 3. Bounded adjustments

Each action type has one adjustment. It moves toward a smaller, later
or less binding commitment. It never moves toward a larger one, and
never invents a field the proposal does not have.

| Action | Adjustment | Rule |
|---|---|---|
| `terminate` | later last day | `lastDay + 14 days`, if that is within the booking horizon; otherwise none |
| `hire` at the premium tier | standard tier | the offer's standard rate, with `startDate = max(proposed, today + standard lead)`; all other fields unchanged |
| `hire` at the standard tier | later start | `startDate + 14 days`, if within the horizon; otherwise none |
| `contract_teleradiology` start, scope `all` | urgent only | the same plan, windows and effective date, with scope `urgent` |
| `contract_teleradiology` start, scope `urgent` | later activation | `effective + 14 days`, if within the horizon; otherwise none |
| `set_roster` (displacing) | one week later | `effective + 7 days`, the next Monday, if within the horizon; otherwise none |
| `set_reading_allocation` | none | the change is already withdrawable at no cost until its withdraw-by date |

**Not proposed: substituting a different action type.** For example,
reducing hours with `set_roster` instead of terminating:
* `set_roster` needs a whole two-week roster.
* No deterministic rule can choose one without designing it.
* That would be the system inventing a management decision.
* The agent may still propose it itself (§5).

## 4. Determinism

**The candidate list is a pure function of three inputs:**
* the proposal's actions;
* the engine's `validate` answers;
* the decision point's date and booking horizon.

The list is recorded with:
* the rule's identity;
* the SHA-256 of the rule's text;
* the inputs.

**In replay,** the list is rebuilt and must match the recorded list
exactly. Otherwise the replay is refused.

## 5. What the agent may do

After the comparison, the agent returns exactly one of:
* `select:<id>`: one of the listed alternatives;
* `revise`: a new proposal;
* `abandon`: no action.

**For `revise`:**
* The new batch goes through validation and, if it qualifies (§1), one
  new comparison.
* A second revision is not compared. It is submitted only if valid,
  and recorded as such.
* This bounds the work at a decision point.

**For `select:no_action`:** this is recorded as selecting no action. It
is distinct from `abandon` (`EVALUATION_CONTRACT.md`).

The system never selects for the agent. If the reply is invalid, or no
reply arrives, nothing is submitted. Nothing is defaulted to any
alternative, the proposal included.

## 6. Size

* There are at most four alternatives, and one comparison plus one
  revised comparison per decision point.
* Class B projections, if run for evaluation, need one branch per
  alternative: four to eight runs per point.
* That costs about 2 to 3 seconds each on the Raspberry Pi
  (`BRANCHING_FEASIBILITY.md` §2).
* There is no search over parameters.
