# V0.13m-lite partial-run procedure

Identity: `dreamteam.alternatives-partial-run/v9`. Protocol lock and live
inference NO GO. Successor of the [V0.13l procedure](../v13l/PARTIAL_RUN_PROCEDURE.md);
steps 1–6 there apply, with:

* FULL analysis through `analysis/alternatives/v13m_verify.py`.
* Every retained model-response body is reparsed strictly (R-JSON-004); an
  ambiguous response makes the archive INVALID.
* Every halt is checked for causal possibility (R-HALT-003): the
  wall-clock guard, its displayed duration and (with SYSTEM_UTC timestamps)
  the structured timestamps.
* A `draw_unrecorded` draw whose retained calls complete the workflow is not
  an acceptable PARTIAL (R-HANDOFF-003, possible evidence loss).
* INVALID archives — including every fail-closed run — are refused and
  excluded from primary analysis; the analysis reports retained evidence and
  `possibleEvidenceLoss` for audit only. No recovery, resumption, retry or
  replacement draw.
