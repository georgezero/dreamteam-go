# Independent review status

**INDEPENDENT REVIEW: PENDING.**

* **The sample:** `v08b-review-1`, 98 items of the frozen
  `dreamteam.decision-evidence/v1` outputs
  ([docs/benchmark/v08b-dev/review/](../../v08b-dev/review/)). It is
  frozen: items SHA-256 `57e332736a5cec3e7bed1a6343237711919cd2c65f515a72e99bf8849f44fd3f`,
  unchanged since V0.8b, neither resampled nor relabelled.
* **The protocol:** unchanged.
* **Labels:** none. No independent reviewer has completed the review. The
  developer has not labelled the items and would not count as
  independent. No model labelled them, and none may.
* **Reported:** nothing. No agreement, confirmation share, precision or
  recall exists. No v1 rule is independently validated.

The machine-readable status is
[review-status.json](review-status.json). CI regenerates it with
`review_metrics.py` and no form, and requires it to say PENDING.

## Added in V0.9a, to make the review possible

* **[review.html](review.html):** an offline review page built from the
  frozen package.
* **[INSTRUCTIONS.md](INSTRUCTIONS.md):** the reviewer's steps.
* **`analysis/evidence/review_metrics.py`:** the checks and metrics for
  completed forms.
* **[RECALL_PROTOCOL.md](RECALL_PROTOCOL.md):** a separate design for
  measuring recall, not carried out.

They were tested with synthetic forms only, which the tests generate and
discard. None is committed as a review.

## What a completed review will report

Per reviewer:
* **Number reviewed:** 98, or the form is refused.
* **Coverage by rule** and rule output: the sample's allocation (below).
* **Agreement with the rules' outputs:** the items labelled `supported`,
  overall and per cell. `supported` means the evidence supports the
  rule's output as stated: a finding that holds, a check that is
  consistent, or an abstention that is right.
* **Supported findings:** confirmed, unsupported, ambiguous and
  insufficient evidence, among the 38 supported findings in the sample.
* **Precision, per rule:** confirmed over that rule's reviewed supported
  findings. Its denominator and qualification are always given:

  | Rule | Supported findings reviewed | v1 supported findings | Qualification |
  |---|---|---|---|
  | EXPLICIT_ACTION_OMITTED | 5 | 5 | census |
  | FINANCIAL_AMOUNT_MISMATCH | 3 | 3 | census |
  | REPEATED_INVALID_PROPOSAL | 10 | 107 | deterministic stratified sample |
  | STATED_ACTION_NOT_EXECUTED | 10 | 45 | deterministic stratified sample |
  | FACTUAL_OBSERVATION_MISMATCH | 10 | 27 | deterministic stratified sample |

  EXPLICIT_OPPOSITE_ACTION has no supported finding. For the census rules
  the share is that of every v1 finding, as judged by that reviewer. For
  the sampled rules it describes the reviewed findings only: the sample
  is stratified and not random, so it is not an estimate for the others.
  The figures are small and carry wide uncertainty.
* **Between reviewers,** if there are two or more: the items on which all
  agree, and each disagreement.

**Never reported: recall.** The sample was drawn from the rules' outputs.
It holds no labelled population of eligible statements, so it cannot show
what the rules missed. The negative controls show only whether the
sampled abstentions were right. A recall estimate needs the separate
protocol in [RECALL_PROTOCOL.md](RECALL_PROTOCOL.md).

**The sample's allocation** (from its manifest):

| Rule | Supported | Unresolved | Insufficient evidence | Consistent |
|---|---|---|---|---|
| EXPLICIT_ACTION_OMITTED | 5 | 6 | – | 6 |
| EXPLICIT_OPPOSITE_ACTION | – | 3 | – | 4 |
| FINANCIAL_AMOUNT_MISMATCH | 3 | 6 | – | 6 |
| REPEATED_INVALID_PROPOSAL | 10 | – | – | 4 |
| STATED_ACTION_NOT_EXECUTED | 10 | – | – | 6 |
| FACTUAL_OBSERVATION_MISMATCH | 10 | 2 | 3 | 6 |

There are also 8 negative controls (`no_statement`): 98 items in all.
