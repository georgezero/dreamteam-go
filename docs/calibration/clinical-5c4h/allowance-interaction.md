# Slice 5C.4h: how the allowances interact (audit G5)

The owner retained every allowance's semantics. This note states what
they jointly permit. Nothing is combined or changed.

## The rules and their allowances

| Rule | Allowance | What it counts |
|---|---|---|
| **Shared clinical allowance** (`clinicalJointWithSafeguard`) | at most 4 of 40 seeds failing or lacking evidence | each seed once, over every seed-level clinical gate: v4 service, backlog stability, Gate A timely access, per-seed severe rate, local severity windows (including the 5C.4h partial-window ceiling), and the Gate B per-seed safeguard |
| **v4 case selection, per seed** (`served_fraction_drop_per_seed`) | its own: at most 4 of 40 seeds per stratum | seeds whose due-demand served fraction drops more than 5 points against the paired baseline, by priority, modality and the inherited stratum |
| **Gate B's mean rule** | **none**: outside every allowance | the seed-averaged unimaged increase must be at most 2 points. One unmatched or thin seed makes it INSUFFICIENT |
| **Absolute limits** (clinical safety) | **none**: zero tolerance, overriding every allowance | any study beyond its absolute past-due limit on any seed. Under 5C.4h this includes the follow-up of orders wanted by the horizon. A study not yet observable makes safety INSUFFICIENT |

## Consequences (`internal/eval/boundary_safety_test.go`, `TestAllowanceInteraction`)

* **The 4-of-40 cap is not universal.** In the witness:
  * seeds 1–4 each lose 9 of 150 urgent orders. That is 6 points:
    under the safeguard's 10, over case selection's 5;
  * seeds 5–8 each fail a soon four-week block.

  The clinical allowance holds seeds 5–8 and case selection holds seeds
  1–4. **Eight distinct seeds have adverse outcomes, and the verdict is
  GO.**
* **Separate allowances may cover disjoint seeds.** That is the
  witness above. When both rules fail on the **same** four seeds, each
  allowance spends its four places on them: GO.
* **Within one allowance, a seed counts once.** A seed failing several
  clinical gates (service, severity, safeguard) takes one place.
* **A fifth clinical seed fails it.** Seeds 1–4 fail case selection,
  and seeds 5–9 fail soon blocks: NO GO on clinical seed reliability.
* **Gate B's mean stays outside the allowances.** Four urgent orders lost
  on every seed is 2.67 points: Gate B fails and the verdict is NO GO,
  although no seed reaches the safeguard and the clinical allowance
  passes.
* **Absolute violations override.** One soon report more than 48 h past
  due, on one seed and in a complete block, is NO GO, while the
  clinical allowance passes (`TestBoundaryAbsoluteOverride`).

## How results must be stated

A favourable verdict under v0.2-eval-5 does **not** mean that at most 4
of 40 seeds had an adverse outcome. It means all of the following:
* at most 4 seeds failed or lacked evidence on the clinical gates;
* separately, each case-selection stratum had at most 4 failing seeds;
* Gate B's mean held;
* no absolute limit was violated or left unobservable.

The protocol (`reserved-validation-protocol.md` §6) carries this
statement into the result. Combining the allowances into one universal
cap would be a policy change, which the owner has not made.
