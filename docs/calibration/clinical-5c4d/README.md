# Slice 5C.4d: audit remediation and seed-level lost-demand inference

The remediation of the independent 5C.4c audit
(`../../audits/slice-5c4c-access-gates-audit.md`, verbatim; the
maintainer's response is `../../audits/slice-5c4c-access-gates-audit-response.md`),
under the owner's decisions of 2026-10-18 (0.2.0-dev.18).

* **Not frozen.** The candidate policy remains a proposal.
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream.

| File | What it is |
|---|---|
| `remediation.md` | The fixes (F2, F3, F8, F9), the stable order identity, the seed-level Gate B, the verdict limitation, the weekly partition sensitivity, the remaining blockers. |
| `gate-b-statistics.md` | Gate B's estimand, assumptions, the impossibility at 40 seeds, the two procedures, the coverage evidence, the owner's alternatives. |
| `gate-b-coverage.json` | The coverage table (`TestGateBCoverage` with `DT_GATEB_COVERAGE`). |
| `sim-access-5c4d.json` | The no-action access validation under the current candidate (development seeds), with the weekly partitions. |
| `reserved-validation-protocol.md` | The updated confirmatory protocol: not runnable until the owner's two decisions. |
