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

Reproduce the run:

    DT_OC_CLINICAL=1 go test ./internal/eval -run 'TestClinicalOperatingCharacteristics$' -v -timeout 600m
