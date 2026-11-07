# Protocol successor — v13f-alt-1

DRAFT — NOT LOCKED — NOT AUTHORIZED.
Independent re-review: PENDING. Human presentation review: PENDING.
Protocol lock: NO GO. Live inference: NO GO. V0.13.0 remains unreleased.

Authoritative outcomes are OUTCOME_DEFINITIONS.md, implemented together with
the protocol generator in analysis/alternatives/v13f.py. The estimand is the
effect of assignment to the complete alternatives interface on canonical
submitted-action distributions within comparison-exposed completed draws.
The treatment includes information, menu, shortcuts, inspection, additional
decision opportunities and calls, and order. There is no information-only
causal claim. Decisions are not scored for improvement.

| Fixed design component | Allocation |
|---|---|
| Historical contexts | 9, unchanged frozen sources |
| Sensitivity pairs | 6, retaining all 12 members |
| Arms | control and treatment |
| Draws | 262; 84 for the first frozen model and 178 for the second |
| Per-unit repetitions | 3 and 5 respectively; A/A contexts retain 2 |
| Model-call ceiling | 802, unchanged |
| Per-draw guard | 6 |
| Request timeout | 300 seconds |
| Time mechanism | 14,400-second request-start guard, not a hard deadline |

The model identifiers, original settings-file bytes, temperature zero,
4096-token response limit and json_object format remain pinned in the
machine-readable protocol. Both arms use the new shared phase-aware system
instruction. The original built-in instruction is retained only for historical
replay. New public-context wrappers, templates, schemas, parser version,
workflow specification and source identities are pinned separately.

The call ceiling is still the sum of per-context schedule bounds: at most
three final-turn calls after a valid replayed proposal, or four including
one correction after a refused one. The correction itself has no additional
retry once the original refusal consumes the single failure allowance.
All 21 replayed initial proposals satisfy the successor initial schema;
the two recorded correction scripts satisfy its correction schema. No hidden
retry or new decision opportunity increases the bound. A/A contexts retain
the existing conservative slack. No budget change needs owner approval.

Schedule entries preserve the existing deterministic order rule and are
regenerated from the same context IDs, models, arms and counts. Neither
contexts nor alternatives are added to the experiment. Extra fidelity witnesses
are scripted regression cases, not scheduled experimental alternatives.

S1-a and S6-a are identical inputs: same public prompt and proposal. They
remain labelled members of separate pairs but add no independent scenario
diversity. The pairs differ in multiple facts and compare whole contexts.
Three or five repeated observations can reveal a change but cannot establish
stability or absence of variation. Temperature zero does not guarantee
stationarity, repeatability or independent responses. Small per-cell samples
have wide intervals; empirical total variation has small-sample noise.
Historical branch selection is conditional on previously observed proposals.

Per-identity treatment-minus-control rates use Newcombe intervals, with TV
reported for the canonical action distribution. Secondary pair departures
compare submitted identity against the validated proposal identity within
exposed members, then use Newcombe/MOVER as context summaries. There is no
pooling across models or hypothesis test. Inspection and citations are
post-assignment process measures and do not isolate inspection effects.

Partial runs preserve all exchanges, stop evidence and coverage. They yield
no complete-run arm differences or pair estimates, and are never resumed.
Limits count every actual attempt, including corrections, failures and
retries, excluding only the explicitly replayed initial proposals.

Before protocol lock: complete the independent technical re-review by a
different reviewer, complete independent human review of the new full
interaction package, obtain owner disposition and acceptance, and pass final
engineering gates on identified artifacts. Eventual lock does not authorize
inference. The live driver implementation remains absent.
