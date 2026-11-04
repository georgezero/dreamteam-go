# V0.13a comparison contract: `dreamteam.management-alternatives/v1`

Status: **proposed** in V0.13a (0.13.0-dev.1). It is not implemented,
and no code reads or writes it.

The contract describes:
* a set of alternatives at one decision point;
* what is known about each, separated by how it is known.

It is a record of facts and projections. It is not a recommendation.

## 1. The three classes

Every value in the contract carries exactly one class.

### Class A: exact immediate

**What it is:**
* A value the engine reports at the decision point, or derives from
  that report by a stated rule.
* It uses only information available to the caller at that moment.

**Sub-classes:**
* **A1, reported:** the value is read directly from an engine field.
  This was "class A" in `V0.12_CONSEQUENCE_INVENTORY.md`.
* **A2, derived:** the value is computed from engine fields by a rule
  stated in the item's `derivation`. This was "class B" there.

**Examples:**
* severance;
* last day;
* displaced bookings;
* the weekly guarantee that B keeps.

### Class B: deterministic projection

**What it is:**
* A value read from a branch that is advanced under a declared
  continuation (`CONTINUATION_DESIGN.md`).
* It is exact for that continuation, that seed and that engine.
* It is **not** a prediction of what will happen, and not a
  counterfactual truth.

**What it requires:**
* a `continuation` block;
* a horizon.

**Example:** orders lost in 45 days with no further management action.

**Warning:**
* A class B value is computed from the simulator's own future: future
  demand, patient attributes and draws.
* No manager has this information at the decision point.
* Its default audience is therefore **evaluation only** (§5).

### Class C: unavailable or uncertain

**What it is:**
* A consequence that is relevant but cannot be given exactly. Either:
  * the engine does not model it, for example morale, clinical
    outcome, or the employee's own later choices; or
  * it is decided only at commit by hidden attributes, for example
    whether a displaced patient accepts a replacement; or
  * it depends on later decisions that the continuation does not
    contain.

**How it is recorded:** it is listed by name, with the reason. It has
no value.

## 2. Structure

```json
{
  "schema": "dreamteam.management-alternatives/v1",
  "decisionPoint": {
    "engine": "0.13.0-dev.N",
    "scenarioFingerprint": "…",
    "seed": 52,
    "time": "2026-10-13T07:00:00",
    "recordChecksum": "…",
    "digests": {"log": "…", "ledger": "…", "actions": "…", "state": "…", "run": "…"}
  },
  "selection": {"rule": "dreamteam.alternative-selection/v1", "ruleSha256": "…"},
  "alternatives": [
    {
      "id": "proposal",
      "source": "proposal",
      "actions": [{"type": "terminate", "employee": "E04", "lastDay": "2026-10-27"}],
      "description": "Terminate Sam (E04), last day 2026-10-27",
      "validation": {"ok": true, "results": ["…engine ActionResult…"]},
      "reversibility": {"class": "A1", "value": "irreversible", "basis": "already_terminating: notice cannot be moved or withdrawn"},
      "immediate": [
        {"key": "severance", "label": "Severance", "class": "A1", "cents": 395000,
         "when": "recognized now, paid 2026-10-30",
         "source": "preview.terminations[0].severanceCents",
         "basis": "two contracted weeks (39.5 h) at the base rate with benefits"}
      ],
      "projected": null,
      "unavailable": [
        {"key": "replacement_acceptance", "class": "C", "reason": "decided at commit by each patient's tolerance, a hidden attribute"}
      ]
    },
    {
      "id": "no_action",
      "source": "no_action",
      "actions": [],
      "description": "Keep current staffing",
      "validation": {"ok": true, "results": []},
      "immediate": [
        {"key": "weekly_guarantee", "label": "Contract guarantee continues", "class": "A2", "cents": 197500,
         "when": "per full week, until a later decision",
         "source": "preview(proposal).forecast[label='Contract cost no longer owed · Sam (E04)'].cents",
         "derivation": "the engine's weekly guarantee for E04, read from the proposal's preview; the guarantee continues under no action. Cross-check: observe.staff[E04].severanceCents / 2, within 1 cent of rounding"}
      ]
    }
  ],
  "continuation": null,
  "comparisons": {
    "immediate": [
      {"key": "severance", "unit": "cents", "class": "A1", "values": {"proposal": 395000, "no_action": 0}}
    ],
    "projected": []
  },
  "provenance": {"engineRequests": ["…the validate, preview and observe requests, with response SHA-256…"]}
}
```

## 3. Field rules

**Alternatives:**
* `alternatives` is ordered by the selection rule, not by any measure
  of merit (`ALTERNATIVE_SELECTION.md`).
* The proposal is always first and no action always second, so an
  order effect is the same for every decision point.

**Validation and actions:**
* `validation` is the engine's own `validate` answer for that exact
  batch.
* An alternative that the engine refuses is kept, with
  `validation.ok: false` and its codes. It has no `immediate` values.
* `actions` holds existing action types only. A type that is not in
  `docs/v13a/ACTION_INVENTORY.md` §1 makes the whole record invalid.

**Immediate values:**
* Every `immediate` item has a `source`: an engine field path.
* An A2 item also has a `derivation`.
* Money is in integer cents. Dates are `YYYY-MM-DD`.
* A missing or mistyped source makes the item unavailable (class C).
  Nothing is defaulted. This is the V0.12b fail-closed rule.

**Comparisons:**
* `comparisons.immediate` lists only keys that every alternative
  reports, in the same unit and class.
* An item that one alternative lacks is listed in that alternative's
  `unavailable`. It is never treated as zero.
* Zero is a value only when the engine reports zero.

**Projections:**
* `projected` items appear only with a `continuation` block. Each names
  the horizon, the metric's definition and the branch digests at both
  ends.
* `comparisons.projected` compares alternatives under the **same**
  continuation and horizon only.

**Unavailable items:** each lists the reason in plain words.

## 4. What the contract forbids

1. **No composite score.**
   * There is no field that adds classes, units or metrics together.
   * There is no weighting.
2. **No ranking or verdict.**
   * No `optimal`, `best`, `recommended`, `preferred`, `rank` or `score`
     field.
   * No ordering by any value.
3. **No harm inference.**
   * Displaced bookings are reported as bookings.
   * No field converts them into patient harm, delay severity or
     clinical risk.
4. **No savings claim.**
   * A difference in money between two class B projections is labelled
     as a difference under the stated continuation and horizon.
   * It is not a saving, a gain or evidence of a good decision.
   * The engine's forecast lines keep the engine's own "estimate"
     label.
5. **No mixing of classes.**
   * A class B projection is never subtracted from a class A value.
   * A value is never shown without its class.

## 5. Audience

Each record has an `audience`:

| Audience | Contains | Use |
|---|---|---|
| `agent` | A1, A2 and C only | shown to the agent before it decides (`AGENT_PRESENTATION.md`) |
| `evaluation` | A1, A2, B and C | written after the decision, for analysis and human review; never shown to the agent at the decision point |

**Why class B is not shown to the agent by default:**
* Showing it would give the agent knowledge of the simulator's realized
  future.
* That would make the benchmark measure use of oracle information, not
  management.

**Showing class B to an agent:**
* It requires its own experimental arm.
* That arm must be declared in a locked protocol.
* Its results must be labelled as oracle-informed.
* No such arm is proposed for V0.13b.

## 6. Versioning

**The identity:**
* The contract is versioned by its schema string.
* Its SHA-256 is that of this document's §2 to §5, as committed when
  the contract is implemented.
* The identity is fixed at implementation (V0.13b), not now.

**Changes:**
* A change to any rule in §3 to §5 is a new version.
* Records under an earlier version stay readable and are never
  rewritten.
