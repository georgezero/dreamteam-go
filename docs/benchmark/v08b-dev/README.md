# V0.8b: evidence validation and financial decision outcomes

V0.8b (0.8.0-dev.2), authorized by the owner on 2026-10-30, after
accepting V0.8a.

**Restrictions:** no live LLM inference, no new simulator mechanic, and
no change to recordings, prompts, policies or scoring. No model was
contacted for anything here.

**It answers three questions:**
* Can the consistency findings be trusted?
* Can the evidence pipeline represent a refusal and what follows it?
* Can outcomes be described without mixing correlation, causation and
  stated intent?

## 1. Frozen evidence

`dreamteam.decision-evidence/v1` is frozen as built in V0.8a (80cfff9)
([FREEZE.md](../v08a-dev/FREEZE.md), `analysis/evidence/frozen-v1.json`):

| | |
|---|---|
| Analyzer | `decision-evidence 1`, SHA-256 `1cc9f9ba…` |
| Rules | six, all version 1 |
| Corpus | 90 recordings, 3,972 decision points, fingerprint `846abc4c…` |
| Evidence | `5743bc22…`; findings: supported 187, unresolved 71, insufficient evidence 9 |

**Reproduced under 0.8.0-dev.2:** the analyzer, tables and traces byte
for byte, and the evidence in every field but the engine build
(`check_frozen.py`, in CI).

**Unchanged:** no rule was tuned.

**Defects found and documented, not fixed:**
* **V1-D1:** repeats of refusals whose message states amounts are not
  recognized across decision points. The funding probes found it.
* **V1-D2:** flagged commitments with a matching action are counted as
  consistent.
* **V1-D3:** amount agreement is lenient by one unit of the stated
  precision.
* **V1-D4:** month-to-date profit in the traces resets at a month's
  start.

## 2. Independent review: PENDING

The review package is complete and frozen ([review/](review/)):
* **Sample:** `v08b-review-1`, 98 items, drawn deterministically from the
  frozen outputs and stratified by rule, output, recording group, model
  and prompt completeness:

  | Cell | Items |
  |---|---|
  | supported findings | 38: every A (5) and C (3) finding, and 10 each of D, E and F |
  | unresolved | 17 |
  | insufficient evidence | 3 |
  | consistent checks | 32 |
  | negative controls | 8: reasons mentioning an action, in which no explicit statement was read |

* **Spread:** across V0.5a (5), V0.5b (9), V0.5c (25), V0.6c (29) and
  V0.7c (30), and all three models.
* **Each item** carries:
  * its identifier and the recording reference;
  * the reason and its span;
  * the proposed actions, validation and execution;
  * the observation the rule used: staff names, or the prompt excerpt
    with the observed value;
  * the rule's output;
  * the evidence completeness.
* **The form:** supported, unsupported, ambiguous or
  insufficient_evidence, with comments and corrections
  ([PROTOCOL.md](review/PROTOCOL.md)).

**No independent reviewer was available. The review is not done:**
* No label exists.
* No agreement, precision or recall is reported.
* The developer's own reading of the findings during V0.8a is not
  independent and is not counted.

**What analyses a review:** `analysis/evidence/review_analysis.py`. It
analyses only complete forms that carry a reviewer's independence
declaration, and reports descriptive agreement by cell.

## 3. Independent review findings

None: the review is pending.

## 4. Funding-refusal probes

Deterministic scripted sequences through the existing agent protocol, on
the V0.7b solvency scenarios, seed 41801 ([PROBES.md](PROBES.md)). All 3
probes ran, and every expected code was observed.

| Probe | Event | Observed |
|---|---|---|
| A | unaffordable hire | `insufficient_funds`: $10,457 against $6,910 of headroom |
| B | unaffordable allocation raise | `insufficient_funds`: $19,726 against $1,715 |
| C | unaffordable teleradiology contract | `insufficient_funds`: $3,000 against −$7,322 |
| D | unaffordable paid radiologist extension | `insufficient_funds`: $577 against −$4,184 |
| E | identical hire repeated (two days, and twice in one day) | refused each time |
| F | revised affordable raise | accepted and reserved ($1,479) |
| G | no action after a refusal | nothing committed |
| H | cost reduction after a refusal | accepted with −$18,725 of headroom |
| I | reservations reduce headroom | an extension accepted before F's reservation is refused after it ($577 against $236) |
| J | credit-limit boundary | funded up to the headroom exactly, both ways: no credit, 7401 BP commits $6,909.04 ≤ $6,910.28 < $6,913.97 at 7402 BP; with credit, the headroom is exactly $50,000 more ($56,910.28), with the same exact boundary |

These test the mechanics and the evidence, not any model.

## 5. Refusal-response evidence

A separate extension, `dreamteam.refusal-response/v1` (DECISION_EVIDENCE
§12), classifies what was proposed next after each refusal, by exact
action identity ([refusal-response-tables.md](refusal-response-tables.md)).

**Probes** (attempt-level evidence), seven refusals:
* `identical_repeated` 2;
* `modified_proposal` 1;
* `different_proposal` 3;
* `no_action` 1.

Read in the llm-agent 2 format, 3 of the 7 become `evidence_unavailable`.

**Historical recordings** (V0.5a–V0.7c), 559 refusals, none of them
`insufficient_funds`:

| Category | Refusals |
|---|---|
| identical_repeated | 2 |
| refusal_no_longer_applicable | 0 |
| modified_proposal | 24 |
| different_proposal | 34 |
| no_action | 42 |
| evidence_unavailable | 457 |

* **Why so many are unavailable:** in 452 of the 457 the response is
  recorded, but the refused attempt's actions are not. A refused first
  attempt followed by a corrected second is the usual case, and only the
  final reply's actions are recorded.
* **The difference from V0.8a's repeat count:** V0.8a counted many repeats
  by refusal message (`REPEATED_INVALID_PROPOSAL`). Here only 2 repeats
  are established by exact payload, because the payloads mostly are not
  recorded. The two analyses use different identities, and both figures
  stand.
* **What none of it says:** whether any agent understood a refusal.

## 6. Financial and operational outcomes

Every V0.6c and V0.7c policy run is paired with no action on the same
entry, seed, horizon and economics: 140 pairs
([FINANCIAL_OUTCOMES.md](FINANCIAL_OUTCOMES.md),
[financial-outcomes.md](financial-outcomes.md)).

**What it reports, separately:**
* financial outcomes: profit, revenue, expense, lowest cash, overtime
  cost, credit and interest;
* operational outcomes: exams, lost orders, turnaround, breaches and
  overtime hours;
* stated commitments, submitted actions and executed actions.

**What it never does:** combine V0.6 with V0.7, score, rank, attribute a
cause, or construct a counterfactual.

**What it shows:**
* **The widest single differences are in V0.7c stage B's
  staffing-shortage Qwen run** (−$181,589 profit, $17,200 credit drawn).
* **Gemma 4 26B-A4B's differences are mostly zero,** because it seldom
  acted.
* **Most actions the LLM agents submitted were not stated in an explicit
  "I will" form.**
* **Submitted and executed differ only where the engine had nothing to
  change.** No submitted action of any policy was refused when applied.

## 7. Proposed V0.8c recording contract

`dreamteam.llm-decision-record/v3` is designed, not deployed
([RECORDING_CONTRACT.md](RECORDING_CONTRACT.md)).

**What each attempt would record:**
* its own reason, actions, parse result, validation and refusals;
* the retry feedback;
* the hash of the message it answered.

**Also proposed:**
* the final decision, and a completeness block;
* live metadata kept apart from replay inputs;
* bounded text, and no secrets or infrastructure details.

It would ship with a new agent version, so llm-agent 2 recordings keep
replaying unchanged.

## 8. Reproducibility

* **Frozen v1:** reproduced under 0.8.0-dev.2 (`check_frozen.py`).
* **Committed outputs:** the review package, the financial outcomes, the
  probes, the frozen rules on the probes and the refusal responses are
  each regenerated in CI and must equal the committed files. CI compares
  them across linux/amd64, linux/arm64 and darwin/arm64.
* **Repeatability:** the probes run twice with identical outputs.
* **Review sampling:** no random number generator is used. The order
  comes from SHA-256 of a fixed seed and each identifier.
* **No inference:** no model is called anywhere, and the v1 tests run the
  analysis with sockets disabled.
* **Historical recordings, sessions, golden references and benchmark
  results:** unchanged (PROGRESS, V0.8b gates).

## 9. Known limitations

* **The independent review has not happened.** The trustworthiness of
  the v1 findings beyond the developer's own reading is untested.
* **The probes cover the mechanics on one scenario and seed.**
  `refusal_no_longer_applicable` could not be scripted there, and is
  tested only in unit tests.
* **Refusal responses in the historical recordings** are mostly
  unclassifiable. That is the recording gap V0.8c's contract would close.
* **The outcome analysis is descriptive:**
  * development seeds, five scenarios, one horizon each;
  * replays only for the LLM runs, so overtime cost and credit for the
    baselines are not shown.
