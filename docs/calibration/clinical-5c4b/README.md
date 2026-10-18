# Slice 5C.4b: the v0.2-eval-5 clinical evaluator

This slice implements the clinical acceptance rules proposed in Slice
5C.4a (`docs/evaluation/proposals/v5-clinical-acceptance.md`, with its
implementation decisions in §J).

* **No v5 policy is frozen.** The owner chooses the values from
  synthetic operating characteristics (owner decision, 2026-10-18:
  build first, values later; the weekend question decided after the
  operating characteristics).
* **v0.2-eval-4 stays in force,** and its evaluations are byte-identical.
* **Nothing else was run:** no strategy, development, reference or
  holdout run.

| File | What it is |
|---|---|
| `protocol.md` | The combined operating-characteristics protocol: the model, scenarios, streams, value grids, bundles and targets. It was committed before any combined result. |
| `../../evaluation/oc-v0.2-eval-5-exploratory.json` | The exploratory run's output, joofi and the Pi byte-identical. |
| `findings.md` | What the exploratory run shows, with its erratum (§7): the stable practice read at about 0.67–0.74, not 0.85. |
| `correction-protocol.md` | The calibration correction: measured utilization, practices at 0.85 and 0.92, a steady state at the takeover, the simulator's worklist; its scenarios, streams and reporting. Committed before any corrected result. |
| `../../evaluation/oc-v0.2-eval-5-exploratory-corrected.json` | The corrected run's output (joofi; A85, Wg85 and L10-92 reproduced identically on the Pi). |
| `findings-corrected.md` | What the corrected run shows: utilization 0.85 and 0.92 as measured, false rejection, the weekend comparison at equal utilization, deterioration with its consequence, the full bundle, and the remaining limitations. |
| `fidelity-protocol.md` | The final calibration correction: the simulator profile (no action, development seeds), demand, mix and hours from it, individual readers in the simulator's blocks, regimes on core capacity, the mirror's validation; committed before any fidelity result. |
| `../../../internal/eval/testdata/sim-profile-5c4b.json` | The measured simulator profile the fidelity model reads. |
| `../../evaluation/oc-v0.2-eval-5-exploratory-fidelity.json` | The fidelity run's output (joofi; A85, Wg85 and M reproduced identically on the Pi). |
| `findings-fidelity.md` | What the fidelity run shows: weekday and mix, readers, utilization, warm start, weekend comparison, delay tails, full bundles, the mirror against the simulator, limitations, and the owner's provisional values. |
| `acceptance-design.md` | The acceptance-design closure: the provisional policy, the stopping point (urgent access cannot be justified as specified), the simulator check of the reporting limits, the witnesses, external validity and the owner decision table. |
| `candidate-policy-v5.json` | The candidate v5 policy: a proposal, not in force, not frozen. |
| `order-lifecycle.md` | The simulator's order lifecycle and the reconciliation of orders with acquisitions. |
| `access-denominators.md` | Access rules A–J, gaming, the calendar and service-opportunity clocks, no-action access. |
| `sim-access-5c4b.json` | The no-action access validation (development seeds; `internal/sim/clinical_access_test.go`). |
| `reserved-validation-protocol.md` | The reserved-stream confirmatory protocol: a draft, not runnable until the owner's decisions. |

Code:

* **`internal/eval/clinical.go`:** measures (exact-timestamp delays,
  censoring, weekly concentration, access with no survivorship,
  absolute limits), seed gates, the shared allowance and clinical
  safety.
* **`internal/eval/clinical_test.go`:** the synthetic acceptance cases
  G1–G6 and G10–G14 of the proposal's §G, each harmful case with an
  acceptable counterpart and exact boundaries.
* **`internal/eval/neutrality_test.go`:** the v0.2-eval-4 evaluation of
  a varied synthetic input, pinned by its digest from before any change.
* **`internal/eval/oc_clinical_test.go`:** the synthetic practice
  (imaging and reading), the tabulation, and its exact agreement with
  the production evaluator.
* **`internal/eval/oc_correction_test.go`:** the correction: measured
  utilization, the corrected scenarios, the consequence measures and
  bundle components, the Monte Carlo intervals, and the tests of the
  calibration and of the simulator's worklist rule.

Reproduce the runs:

    DT_OC_CLINICAL=1 go test ./internal/eval -run 'TestClinicalOperatingCharacteristics$' -v -timeout 600m
    DT_OC_CORRECTED=1 go test ./internal/eval -run 'TestClinicalOCCorrected$' -v -timeout 600m
    DT_SIM_PROFILE=<file> go test ./internal/sim -run 'TestSimulatorClinicalProfile$' -timeout 60m
    DT_OC_FIDELITY=1 go test ./internal/eval -run 'TestClinicalOCFidelity$' -v -timeout 600m
    DT_SIM_ACCESS=<file> go test ./internal/sim -run 'TestSimulatorAccessValidation$' -timeout 60m
