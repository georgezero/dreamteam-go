# Slice 5C.4l: order provenance, inherited safety and reserved protocol closure

This slice answers the independent 5C.4k audit
(`docs/audits/slice-5c4k-clinical-safety-protocol-lock-audit.md`,
committed verbatim before any of this) under the owner's decisions of
2026-10-20. The evaluator is 0.2.0-dev.23. The maintainer's response is
`docs/audits/slice-5c4k-clinical-safety-protocol-lock-audit-response.md`.

* **Not locked, not run, not frozen, not reviewed.** The focused 5C.4m
  review comes next; the owner decides on the study-swap limitation
  (`clinical-provenance.md` §2.3) before any lock.
* **Nothing reserved was run.** No reference or holdout seed, unseal
  record or reserved stream PCG(20261201, ·) was used; the harness
  refuses that family. No strategy was evaluated. The simulator ran with
  no action on the development seeds 901–910 only.
* **The clinical policy is unchanged.** Only record integrity changes:
  `orderStudyIntegrity: contractInherited` and `orderProvenance:
  pairedOrigin`.
* **History unchanged:**
  * v0.2-eval-4 is byte-identical;
  * the 5C.4d, 5C.4e, 5C.4f, 5C.4h and 5C.4j candidates keep their rules
    (kept byte for byte in `internal/eval/testdata/`);
  * replay through dev.22 is exact.

| File | Contents |
|---|---|
| `clinical-provenance.md` | The provenance contract (K1, K2), what it does not establish (candidate-only orders, both records edited, the study swap), inherited safety, the partial-window reporting levels (K5), the follow-up's financial observation (K6) |
| `reserved-validation-protocol.md` | The revised confirmatory protocol: provenance in the decision, zero-evidence adjudication, the invocation record, the promised reports, the pinned execution (§8, not run) |
| `scenario-manifest.json` | Version 3: the candidate's digest pinned, shared pre-takeover history, the identity witnesses targeted at provenance, W-ACCESS65's disclosure |
| `construction-checks.json` | The manifest on the development family PCG(20261021, ·): all 49 scenarios, 40 seeds each; the stable and near-margin scenarios, 200 seeds each (development checks: `invocation` reads `development`) |
| `sim-access-5c4l.json` | The no-action simulator records on the development seeds under the revised candidate |
| `followup-finance.json` | The no-action practice's financial measures observed to H + 31 d and to H + 45 d |

The candidate policy is `../clinical-5c4b/candidate-policy-v5.json`,
SHA-256 `83d481b7040b69d78a58de68afc938a588b5b37ba396438509f38c56ca163948`.

## Development results (not the protocol's verdicts)

On the development family PCG(20261021, ·), judged by `reservedCriteria`
(`construction-checks.json`). The protocol judges 62 groups on the
reserved family; these checks judge 1 or 5 groups, so the adjudicator's
`invocation` entry reads `development` for both passes.

| Criterion | 49 scenarios × 40 seeds | stable and near-margin × 200 seeds |
|---|---|---|
| T1 stable rejection | pass (0 of 4 judged; the four stress groups, reported, 0 too) | pass (0 of 20) |
| T2 harmed detection | inconclusive (15 of 15 flagged; fewer than 200 harmed) | inconclusive (not run) |
| T2-conformance, T2G | pass | **inconclusive (not run: zero-evidence rule)** |
| T3 absolute limits, stable | pass | pass (none of 800 seeds) |
| T4 witnesses | pass (all 14 by their targeted rule; the identity witnesses by order provenance on their first seed alone) | **inconclusive (not run)** |
| T5a near-margin conformance | inconclusive (one group) | pass |
| T5b (0.40 at +2.1, 0.01 at +3) | pass (0.30, 0) | pass (**0.33**, **0**) |
| T6, T7 | pass | pass |
| T8 | pass | **inconclusive (not run)** |

* **Provenance:** every stable, harmful and near-margin pair reconciles
  by origin; failures occur only on the three identity witnesses' first
  seeds, as specified.
* **Stable practices:** 0 of 28 groups rejected; 22 seed-level failures
  in all, on urgent timely access and backlog stability (and one soon
  breach rate); **no stable seed fails the partial-window ceiling**.
  This shows no regression on the development family; it does not
  establish the T1 target (audit 5C.4k K7).
* **Near the margin**, Gate B accepted 0.65 at +2 points (exact
  binomial prediction 0.549) and 0.33 at +2.1 (0.344), 0 at +3.
* **The 200-seed pass** shows the K4 repair: the eleven deterioration,
  five harm and fourteen witness scenarios it does not run make T2G,
  T2-conformance, T4 and T8 INCONCLUSIVE, where the 5C.4j adjudicator
  reported T4 and T2G PASS with no group.
* **Runs:** 2,370 s and 3,954 s, peak 1.13 and 1.15 GB, memory-capped,
  one at a time.
