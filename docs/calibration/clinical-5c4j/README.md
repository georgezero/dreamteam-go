# Slice 5C.4j: clinical safety and reserved protocol remediation

This slice answers the independent 5C.4i audit
(`docs/audits/slice-5c4i-boundary-safety-lock-readiness-audit.md`,
committed verbatim before any of this) under the owner's decisions of
2026-10-19. The evaluator is 0.2.0-dev.22. The maintainer's response is
`docs/audits/slice-5c4i-boundary-safety-lock-readiness-audit-response.md`.

* **Not locked, not run, not frozen, not reviewed.** The revised
  clinical policy and the adjudicator need the independent 5C.4k review.
  No earlier component GO carries over.
* **Nothing reserved was run.** No reference or holdout seed, unseal
  record or reserved stream PCG(20261201, ·) was used; the harness
  refuses that family. No strategy was evaluated. The simulator ran with
  no action on the development seeds 901–910 only.
* **History unchanged:**
  * v0.2-eval-4 is byte-identical;
  * the 5C.4d, 5C.4e, 5C.4f and 5C.4h candidates keep their rules (kept
    byte for byte in `internal/eval/testdata/`);
  * replay through dev.21 is exact.

| File | Contents |
|---|---|
| `clinical-safety.md` | The clinical-policy changes: the observed severe-rate ceiling on partial windows (I1), the order–study contract (I2), the 45-day follow-up and its time conventions (I3), prefix-stable randomness (I4) |
| `reserved-validation-protocol.md` | The revised confirmatory protocol: coverage before judgment, the remainder, per-priority materiality, targeted witnesses, the exact T5 prediction, the approved tolerances |
| `scenario-manifest.json` | Version 2: the T5 tolerances pinned, W-ID-CHANGED as specified, the practices with prefix-stable draws |
| `construction-checks.json` | The manifest on the development family PCG(20261021, ·): all 49 scenarios, 40 seeds each; and the stable and near-margin scenarios, 200 seeds each |
| `sim-access-5c4j.json` | The no-action simulator records on the development seeds, observed 45 days past the horizon, under the revised candidate |

The candidate policy is `../clinical-5c4b/candidate-policy-v5.json`,
SHA-256 `9efa6e63f38de9cfbfdb829ac1a82eff114d6e5c523d32df5d46f53b8c429e7c`:

```json
"partialWindows": "rateCeiling",
"safetyCohort": "ordersByHorizon",
"followUpDays": 45,
"orderStudyIntegrity": "contract"
```

## Development results (not the protocol's verdicts)

On the development family PCG(20261021, ·), judged by `reservedCriteria`
(`construction-checks.json`). The protocol judges 62 groups on the
reserved family; these checks judge 1 or 5 groups, so T2 (below 200
harmed seeds) and T5a (on 40 seeds) are INCONCLUSIVE by construction.

| Criterion | 49 scenarios × 40 seeds | stable and near-margin × 200 seeds |
|---|---|---|
| T1 stable rejection | pass (0 of 4 groups) | pass (0 of 20) |
| T2 harmed detection | inconclusive (15 of 15 flagged; too few) | — |
| T2 conformance, T2G | pass | — |
| T3 absolute limits, stable | pass (no violating seed) | pass (none of 800) |
| T4 witnesses | pass (all 14 by their targeted rule) | — |
| T5a near-margin conformance | inconclusive (one group) | pass |
| T5b tolerances (0.40 at +2.1, 0.01 at +3) | pass (0.35, 0) | pass (**0.31**, **0**) |
| T6 safeguard on stable | pass | pass (0 of 800 flagged) |
| T7 stable insufficiency | pass | pass |
| T8 coverage and validity | pass | pass |

* **Near the margin**, Gate B accepted 0.56 at +2 points (exact binomial
  prediction 0.508) and 0.31 at +2.1 (0.303).
* **False rejection** under the rate ceiling: none (`clinical-safety.md`
  §1.4).
* **Runs:** 2,116 s and 3,031 s, peak 1.2 GB each, memory-capped, one at
  a time.
