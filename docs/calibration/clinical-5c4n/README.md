# Slice 5C.4n: final certification pipeline remediation

This slice answers the independent 5C.4m audit
(`docs/audits/slice-5c4m-provenance-protocol-lock-audit.md`, committed
verbatim before any of this) under the owner's decisions of 2026-10-20.
It is the final planned remediation before the focused 5C.4o
release-gate audit. The evaluator is 0.2.0-dev.24. The maintainer's
response is
`docs/audits/slice-5c4m-provenance-protocol-lock-audit-response.md`.

* **Amended by the release-gate fixes** (0.2.0-dev.25, audit 5C.4o O1
  and O2: Gate B reconciled with its own measurements; nothing after a
  line's value): `certification.md` §8. The development lines below
  re-read under the fixes with no measurement problem and the same
  verdicts.
* **Not locked, not run, not frozen.**
* **Nothing reserved was run.** No reference or holdout seed, unseal
  record or reserved stream PCG(20261201, ·) was used; the build refuses
  that family. No strategy was evaluated or optimized.
* **The clinical policy is unchanged**, and so are the simulator's
  behaviour, the severity thresholds, the scenario manifest and the
  reserved seed protocol:
  * candidate SHA-256
    `83d481b7040b69d78a58de68afc938a588b5b37ba396438509f38c56ca163948`
    (`../clinical-5c4b/candidate-policy-v5.json`);
  * manifest SHA-256
    `250bc7fc835d99140212505e0e79f74f2da1484dbe856fea6a5b93eec9153ef4`
    (`../clinical-5c4l/scenario-manifest.json`).
* **The study swap** is accepted only for trusted in-process evaluation
  (protocol, CERTIFICATION SCOPE).
* **History unchanged:** v0.2-eval-4 is byte-identical; replay through
  dev.23 is exact.

| File | Contents |
|---|---|
| `certification.md` | What changed and why: the in-process trust boundary (M1), measurement completeness and the repaired criteria (M2), the protocol (M3), errors (M4), the validation helper (M6), the corrections, M5 deferred |
| `reserved-validation-protocol.md` | The confirmatory protocol, superseding 5C.4l's: the CERTIFICATION SCOPE, the runtime inputs, measurement completeness, certification, the single pinned execution (§8, not run) |
| `construction-checks.json` | The manifest on the development family PCG(20261021, ·), executed by the certified path: all 49 scenarios at 40 seeds; the stable and near-margin scenarios at 200 seeds. Both `development`, `not certified` |
| `sim-access-5c4n.json` | The no-action simulator records on the development seeds, now integrity-valid; byte-identical to 5C.4l's |
| `followup-finance.json` | The follow-up's financial observation, regenerated on valid records; byte-identical to 5C.4l's |

## Development results (not the protocol's verdicts)

On the development family PCG(20261021, ·), executed by the certified
path (`TestReservedManifestRun` → `runCertified`) and recorded in
`construction-checks.json`. Both passes read `invocation: development`,
`certification: not certified`, as they must: they are neither the
reserved family nor the manifest's seeds.

| Criterion | 49 scenarios × 40 seeds | stable and near-margin × 200 seeds |
|---|---|---|
| T1 stable rejection | pass (0 of 4; stress groups reported, 0) | pass (0 of 20) |
| T2 harmed detection | inconclusive (15 of 15 flagged; fewer than 200 harmed) | inconclusive (not run) |
| T2-conformance, T2G | pass | inconclusive (not run) |
| T3 absolute limits, stable | pass | pass |
| T4 witnesses | pass | inconclusive (not run) |
| T5a near-margin conformance | inconclusive (one group each) | pass (0.65 against 0.549 at +2; 0.33 at +2.1) |
| T5b (0.40 at +2.1, 0.01 at +3) | pass (0.30, 0) | pass (0.33, 0) |
| T6, T7 | pass | pass |
| T8 | **pass** (after the per-kind correction below) | inconclusive (not run) |

* **No measurement changed.** Every line of both passes is
  byte-identical to 5C.4l's: the remediation moved the trust boundary
  and the checks, not the simulator or the clinical rules.
* **No coverage problem** on the 40-seed pass: every genuine line meets
  the measurement completeness checks; exit 0. The 200-seed pass, which
  runs 9 of 49 scenarios by design, exits non-zero on its incomplete
  coverage (40 problems, one for each scenario not run, the first
  "D-85: no seeds under this invocation: not run") and certifies
  nothing: the M4 behaviour.
* **T8 correction during this slice.** The first 40-seed run on this
  build read T8 INCONCLUSIVE ("W-ID-CHANGED group 0: soon matched 40 of
  39"): the new equality expected the changed-urgent witness's first
  seed unmatched in every priority, but only its urgent priority is.
  The old floor (`<`) had hidden the distinction. The expectation is
  now exact per witness kind (`idUnmatched`; protocol §2), guarded by a
  mutation, and both passes were repeated on the corrected build.
* **Runs:** 2,428 s and 4,099 s, peak 1.11 and 1.15 GB, memory-capped,
  one at a time. The executions logged the SHA-256 of their outputs:
  * 40 seeds: lines
    `d2fe1c0d6d50729a0eebbdb36f2358e9f0941242f0a3d5cfc395826445e20452`,
    adjudication
    `b86e5a74d92a42e98bec7181ce81385a55cfac6ef96dc1fb69178608cda44c44`;
  * 200 seeds: lines
    `58e2f1b40ee0fce2f25e255e0cf0c08dd5e71addc3158c484ee8e8e775778cfa`,
    adjudication
    `b1eb7da61000f263e9a6bf113d9da6c012d74946aef6e18f6a523abaf6ae7037`.
