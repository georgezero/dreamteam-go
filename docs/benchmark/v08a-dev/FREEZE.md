# Freeze record: dreamteam.decision-evidence/v1

Frozen 2026-10-30, in V0.8b (0.8.0-dev.2), by the owner's decision.

The machine-readable record is
[analysis/evidence/frozen-v1.json](../../../analysis/evidence/frozen-v1.json).
It is checked by `analysis/evidence/check_frozen.py`, in CI and in the
tests.

## What is frozen

| | |
|---|---|
| Contract | `dreamteam.decision-evidence/v1` ([DECISION_EVIDENCE.md](../../DECISION_EVIDENCE.md)) |
| Analyzer | `decision-evidence 1`: `analysis/evidence/decision_evidence.py`, SHA-256 `1cc9f9ba…` (full value in the record) |
| Rules | EXPLICIT_ACTION_OMITTED, EXPLICIT_OPPOSITE_ACTION, FINANCIAL_AMOUNT_MISMATCH, REPEATED_INVALID_PROPOSAL, STATED_ACTION_NOT_EXECUTED, FACTUAL_OBSERVATION_MISMATCH, all version 1 |
| Source commit | `80cfff9` (V0.8a, CI run 41104143975), engine 0.8.0-dev.1 |
| Corpus | `docs/benchmark/v08a-dev/corpus.json`: 90 recordings, 3,972 decision points; recordings fingerprint `846abc4c…` (SHA-256 of the recordings' `sha256sum` lines, sorted by path) |
| Agent read | llm-agent 2, `llm-agent-prompt/v2`, `llm-agent-memory/v1`, `agents/llm/llm_agent.py` SHA-256 `5214e7bf…` |
| Outputs | `evidence.json` (`5743bc22…`), `tables.md`, and the decision traces (not committed; their SHA-256 is recorded) |
| Findings | supported 187, unresolved 71, insufficient evidence 9 |

The full hashes are in the record.

**Preserved:**
* the rule identifiers;
* the matching and abstention semantics;
* the finding identifiers (`de1-…`) and the output order;
* the committed artifacts in this directory, which are never rewritten.

## How a later build is checked against it

Each later build replays the corpus and runs the frozen analyzer again.
`check_frozen.py` then requires:
* the analyzer source, byte for byte;
* `tables.md` and the traces, byte for byte;
* `evidence.json` equal in every field but two. The excluded fields name
  the engine build the replays were made with:
  * each finding's `run.engine`;
  * the SHA-256 of the replay files, which record the engine version in
    their hello and run records.

Under 0.8.0-dev.2, V0.8b's build, the check passes: every finding,
identifier, status, count and table is unchanged.

## Changes

**Rules are not tuned against this corpus any more.**
* A behavioural change to a rule needs a new rule version (2) and the
  owner's approval.
* A v2 rule would run beside v1, never replace its outputs.

## Unsupported claim types (v1)

These are not checked: no v1 rule reads them.
* parameters of a matched action (minutes, dates, basis points);
* timing claims (shift times, next-slot days, durations);
* claims about the agent's own earlier decisions and refusals;
* causal and purpose claims ("this will clear the backlog");
* implicit intentions (gerunds, "I need to", "I should");
* any statement in a sentence with a hedge, deferral, question or "cannot",
  or in a reason that revises itself;
* amounts other than cash, month-to-date profit, payroll due,
  receivables, bills due and the teleradiology retainer; rates;
  "monthly" profit;
* facts other than unread, overdue (total and by priority), lost demand
  since the start, the weekday and "closed".

## Known coverage limitations

* **Only the final parsed reply** of each decision point has its reason
  and actions recorded. Earlier attempts appear only through their
  refusals' type, code and message.
* **V0.5a** (llm-agent 1): no rebuildable prompt, no refusal records.
* **Observation rules** compare with the agent's summary prompt, not the
  full observation.
* **The rules were calibrated on this corpus;** there is no held-out set.

## Known defects (documented, not fixed)

**V1-D1. Refusals whose message carries amounts are not recognized as
repeats across decision points.**
* *Cause:* `REPEATED_INVALID_PROPOSAL` identifies a refused proposal by
  type, code and message with ISO dates normalized. An `insufficient_funds`
  message states the commitment and that day's funding headroom ("…
  commits $10,457 …, more than the funding headroom $6,910 …").
* *Effect:* the identical proposal refused on two days has two messages,
  so two keys.
* *Shown by the V0.8b funding probes:* the identical hire refused at
  points 0 and 1 is one repeat within point 1 (two attempts with the same
  message) but not a repeat across the points
  ([probe-evidence.json](../v08b-dev/probe-evidence.json)).
* *Historical effect:* none, because no historical refusal was
  `insufficient_funds`. Other codes whose message changes with the day,
  for example a lead time stated against a moving earliest date, are
  affected the same way.
* *Elsewhere:* the refusal-response extension, which identifies actions
  by payload, does not have this defect.
* *A fix* would be rule version 2 (identity by payload where recorded),
  and needs owner approval.

**V1-D2. Flagged commitments with a matching action count as
consistent.**
* *Cause:* when a commitment carries an abstention flag (hedged, deferred,
  …) and the reply proposes a matching action, `EXPLICIT_ACTION_OMITTED`
  counts the statement as consistent.
* *Effect:* the 480 consistent A checks include such statements. No
  finding is affected; only the count is.

**V1-D3. Amount agreement is lenient by up to one unit of the stated
precision.**
* *Cause:* a stated amount agrees with an observed one within one unit of
  its stated precision, on either side. "$11k" agrees with any value from
  $10,001 to $11,999.
* *Effect:* this errs towards consistency, as intended, but it is wider
  than rounding.

**V1-D4. Month-to-date profit in the traces resets at a month's start.**
* *Cause:* the decision traces' `changeSinceThisPoint` includes
  month-to-date profit.
* *Effect:* across the first day of a month its change compares two
  different months. Only the traces are affected, not any finding.
