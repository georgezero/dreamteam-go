# Slice 5C.4f: volume-aware clinical severity windows

The owner's decisions of 2026-10-19 (0.2.0-dev.20):
* no urgent weekly gate (the per-seed 5% severe rate and the absolute
  24 h limit remain);
* soon in fixed four-week calendar blocks (minimum 100, provisional);
* routine in fixed calendar weeks (minimum 100);
* the per-seed safeguard approved at 10 / 6 / 5 points.

* **Not frozen.** The candidate policy remains a proposal.
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream.

| File | What it is |
|---|---|
| `severity-windows.md` | The rules, the configuration, the boundaries, the development-only evaluability, the soon block cohorts, FAIL versus INSUFFICIENT, the anchoring and partial-block sensitivity, the witnesses, the mutation checks. |
| `sim-access-5c4f.json` | The no-action simulator records on the development seeds under the candidate: every window, every seed's clinical gates, the per-week soon counts. |
| `reserved-validation-protocol.md` | The confirmatory protocol: complete, not run, to be locked after the independent review. |
