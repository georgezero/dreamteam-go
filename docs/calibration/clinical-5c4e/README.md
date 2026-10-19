# Slice 5C.4e: empirical clinical acceptance and weekly severity closure

The owner's decisions of 2026-10-19 (0.2.0-dev.19): Gate B as an
empirical benchmark acceptance rule without a population-confidence
claim, its per-seed safeguard, and the weekly severity over fixed
calendar weeks.

* **Not frozen.** The candidate policy remains a proposal
  (`../clinical-5c4b/candidate-policy-v5.json`).
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream.
* **Stop condition reported:** the weekly minimum cohorts make the
  urgent rule unevaluable on every development seed
  (`empirical-acceptance.md` §3.1). The minimums are unchanged.

| File | What it is |
|---|---|
| `empirical-acceptance.md` | The empirical Gate B, its claims kept apart, the per-seed safeguard's place in the allowance, the calendar weekly rule, the insufficiency rates, the witnesses, the mutation checks. |
| `seed-safeguard.md` | The per-seed threshold's prespecified rule, its development-only derivation and the synthetic evidence. |
| `seed-safeguard-evidence.json` | That evidence (`TestSeedSafeguardEvidence` with `DT_SEED_SAFEGUARD`). |
| `sim-access-5c4e.json` | The no-action simulator records on the development seeds under the candidate: Gate B's eligible demand, the calendar weeks, every seed's clinical gates. |
| `reserved-validation-protocol.md` | The updated confirmatory protocol: Gate B stated as empirical without coverage; not runnable until the weekly minimums are decided. |
