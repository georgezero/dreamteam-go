# Slice 5C.4c: clinical access and weekly severity gates

The owner's modified option B (2026-10-18) and the volume-normalized
weekly severity are implemented in the v5 evaluator (0.2.0-dev.17).

* **Not frozen.** The candidate policy remains a proposal
  (`../clinical-5c4b/candidate-policy-v5.json`).
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream. The simulator ran with no action on
  the development seeds and on the mini test fixture.

| File | What it is |
|---|---|
| `access-gates.md` | The implemented rules, the lost-demand methodology, the weekly rule, the adversarial tests, the no-action check and the remaining owner decisions. |
| `reserved-validation-protocol.md` | The completed confirmatory protocol; not to be run before the review and the owner's approvals. |
| `astra-review-package.md` | The independent review's scope, questions and constraints. |
| `sim-access-5c4c.json` | The no-action access validation under the current candidate (development seeds). |

Code:

* `internal/eval/clinical.go`: Gate A (lost apart), the weekly blocks,
  the report-only maximum wait.
* `internal/eval/lostdemand.go`: Gate B.
* `internal/eval/clinical_gates_test.go`: cases C1–C10.
* `internal/sim/demand_pairing_test.go`: exogenous demand across plans.
* `internal/eval/oc_fidelity_test.go`: aligned weeks, paired demand
  streams, model unchanged.
