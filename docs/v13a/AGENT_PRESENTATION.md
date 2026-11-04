# V0.13a agent presentation proposal

Status: **proposed** in V0.13a (0.13.0-dev.1). It is not implemented,
and no prompt was written or changed. The final text will be fixed,
hashed and locked in V0.13b before any model sees it.

**What is shown:**
* the comparison with audience `agent` (`COMPARISON_CONTRACT.md` §5):
  classes A1, A2 and C only;
* no class B projection.

## 1. Principles

1. **Neutral.**
   * Every alternative is shown with the same sections, in the same
     order and with the same wording.
   * There are no adjectives of value: "costly", "safe", "risky",
     "better", "savings".
   * There is no emphasis, colour or ordering by any value.
2. **Facts apart from estimates.**
   * Engine facts are under one heading.
   * The engine's own estimates are under another, with the engine's
     "estimate" label.
   * Unavailable items are under a third.
3. **No recommendation.**
   * The text never says which alternative to choose, and never implies
     a default.
   * The proposal is listed first only because it is the agent's own.
     This is stated.
4. **No claim that more information helps.**
   * The presentation does not tell the agent that comparing improves
     decisions. That has not been shown.

## 2. Layout per alternative

```
Alternative <n> of <N>: <id>
  Actions: <the exact batch, as JSON>
  Description: <engine-derived description: action types, employee, dates>
  Validation: accepted | refused (<codes>: <engine messages>)

  Engine facts at <decision time>:
    Financial commitments:
      - <label>: <amount> · <when> · <engine basis>
    Operational consequences:
      - <e.g. last day, end date, displaced bookings (count; list by id), voided extensions>
    Reversibility: <engine rule, e.g. "irreversible: notice cannot be moved or withdrawn">
    Obligations that continue: <e.g. weekly contract guarantee: <amount>>

  Engine estimates (labelled "estimate" by the engine):
    - <label>: <amount> · <when> · <engine basis>

  Not available before deciding:
    - <item>: <reason>
```

**Rules:**
* An alternative the engine refused shows only its validation.
* `no_action` shows the obligations that continue (`TERMINATION_VS_KEEP.md`
  §3). It never says "no cost".

## 3. Example (termination versus keeping current staffing)

The values are those of `TERMINATION_VS_KEEP.md` §2 and §3, and the
text is illustrative.

```
Alternative 1 of 2: proposal (your proposal)
  Actions: [{"type":"terminate","employee":"E04","lastDay":"2026-10-27"}]
  Validation: accepted
  Engine facts at 2026-10-13 07:00:
    Financial commitments:
      - Severance · Sam (E04): $3,950.00 · recognized now, paid 2026-10-30
      - Contract pay to the last day · Sam (E04): $4,350.00 minimum · today to 2026-10-27
    Operational consequences:
      - Last day 2026-10-27; not employed from 2026-10-28; notice 14 days
      - Displaced bookings: 0
    Reversibility: irreversible (notice cannot be moved or withdrawn)
  Engine estimates:
    - Contract cost no longer owed · Sam (E04): $1,975.00 per full week from 2026-10-28
      (estimate: the contract guarantee with benefits; overtime and hold-over not modeled)
  Not available before deciding:
    - Bookable slots lost from 2026-10-28: not reported for a termination
    - Revenue and orders affected after 2026-10-28: not projected
    - Whether displaced patients accept replacements: decided at commit

Alternative 2 of 2: no_action
  Actions: []
  Validation: accepted (nothing to apply)
  Engine facts at 2026-10-13 07:00:
    Financial commitments: none new
    Operational consequences: none new; current rosters and bookings unchanged
    Reversibility: a termination remains possible later, with at least 14 days' notice
    Obligations that continue:
      - Contract guarantee · Sam (E04): $1,975.00 per full week, until a later decision
      - Severance if terminated later: two contracted weeks (currently $3,950.00)
  Not available before deciding:
    - Revenue, orders and costs after today: not projected
```

## 4. What the agent is asked

The request states:
* the alternatives listed;
* the three replies allowed (`select:<id>`, `revise`, `abandon`;
  `ALTERNATIVE_SELECTION.md` §5);
* that no reply is a default.

It asks for a short reason, recorded as text only. The reason is never
scored as evidence of comparison (`EVALUATION_CONTRACT.md` §3).

## 5. What is fixed in V0.13b before any use

* The template text and its SHA-256.
* The ordering rule.
* The labels.
* The tests that show:
  * every number shown equals the engine field it names;
  * nothing is shown that the contract does not contain.
* No live model sees the template until an owner-approved, locked
  protocol allows it.
