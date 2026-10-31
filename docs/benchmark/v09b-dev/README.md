# V0.9b: financial decision quality and operational exposure

Status: **in development** (`0.9.0-dev.2`), authorized by the owner on
2026-10-31 after V0.9a was accepted. V0.9 is not released.

V0.9b is a read-only analysis milestone. It changes no simulator
mechanics, agent, prompt or benchmark score. It ran no live model and no
clinical suite.

| Part | Where |
|---|---|
| The contract and its definitions | [docs/FINANCIAL_EVIDENCE.md](../../FINANCIAL_EVIDENCE.md) |
| The analyzer and its tests | `analysis/evidence/financial_evidence.py`, `analysis/evidence/test_financial_evidence.py` |
| The corpus | [corpus.json](corpus.json) |
| The results | [financial-evidence.json](financial-evidence.json), [financial-tables.md](financial-tables.md) |
| The reversibility matrix | [REVERSIBILITY.md](REVERSIBILITY.md) |
| The case studies | [case-studies.md](case-studies.md) |
| The parameter evidence v1 freeze | `analysis/evidence/frozen-parameter-v1.json`, `analysis/evidence/check_parameter_frozen.py` |

The question this milestone asks: **can financially consequential
management decisions be identified from what was observable when they
were made, before their long-term outcomes are known?**

## 1. Baseline

* **Start:** clean main at a5e5f9b (V0.9a, accepted).
* **Tag:** `v0.8.0` (object db5c147) still points at b803302. No tag was
  moved.
* **Evidence present and unchanged:**

| Evidence | File | SHA-256 at a5e5f9b |
|---|---|---|
| decision evidence v1 (frozen) | docs/benchmark/v08a-dev/evidence.json | `5743bc223965e75d77440021ecf602b74c1e4bb35c6b6a08afe8be08b8955d4d` |
| its freeze record | analysis/evidence/frozen-v1.json | `64a8e015f9b6901447ec6ddbd979e8882047cdfef62911cf6db983db7ff76a78` |
| repeat evidence v2 | docs/benchmark/v08c-dev/history/repeat-v2.json | `a80b704cc62d960c46075b723801ef1c8cd47b682185f3ed27daa72abfde2624` |
| refusal-response v1 | docs/benchmark/v08b-dev/refusal-response.json | `55d22b53d65cc49f6e72a2379238d5ebfa80f7e0027810020fd3a068ff7d3cc9` |
| refusal-response v2 | docs/benchmark/v08c-dev/history/refusal-response-v2.json | `cdb0e1cc0d59495f09733d2eb0a1e8b5e98dc1edfc574cd2010f2a242e6f0aa0` |
| parameter evidence v1 | docs/benchmark/v09a-dev/parameter-evidence.json | `9f556c69ad187c22b9dbb1c4a3149b1e450c368b97815b1e3e75368ae3c01a71` |
| | docs/benchmark/v09a-dev/parameter-tables.md | `2c880e59c8918dad0e20cdde88266d0680d35ec9e8d9476890608ea53d03c474` |
| the review package `v08b-review-1` | docs/benchmark/v08b-dev/review/items.json | `57e332736a5cec3e7bed1a6343237711919cd2c65f515a72e99bf8849f44fd3f` |
| | review/manifest.json | `027f5910b051f9da50c8e00f5e0084204b1670187c2b7b78378790638b932aa6` |
| | review/form.csv | `754b4cbb5889e9e049f1b17781fccd1bec43c9977b236d5ba7e51b1b7fc121dd` |
| | review/PROTOCOL.md | `5121059b4d1fa1b7cce75b46054ba31db942696e573da1c2c7e9d2dd7899f3bf` |
| the review page and status | docs/benchmark/v09a-dev/review/review.html | `4d001f0d28b24837d0991205ce1955b90fcc053034edbbccbb20f746a31a2f50` |
| | review/review-status.json (PENDING) | `36e6efabcb42fae765d52cbe72d7d31ebafc57a232e4f3f7aed4ffd8011bfed4` |

Aggregates (SHA-256 of the lines `<path> <sha256>` in path order):

| Set | Files | Aggregate |
|---|---|---|
| historical recordings, V0.5c–V0.8c (`*-decisions.jsonl`) | 96 | `041a30b0788edc3017b18b2fa07d36087e73f5786af33d33a98559794234b5c3` |
| benchmark suites (docs/benchmark/suite) | 4 | `63c007ccedc023a501690aa14ca76d400a18be68024cfa658ec5804bd7265c65` |
| golden references (testdata/golden, without MIGRATIONS.json) | 12 | `d84f80718f3ab0463cc3768cf89b5ba09885b654e674a92a87c82c6998a2b25f` |
| testdata/golden/MIGRATIONS.json | 1 | `08301b5dc96a7b8750f73d81bbc28b0e10cc4d56cc30e36d7dbe116a063f2b40` |
| session fixtures before V0.9b (testdata/sessions/*.json) | 102 | `db94caa5d2d162b2bf99acb5678cd2c69e8dc0e94c11696f8b412363957af734` |

V0.9b changes none of these files except MIGRATIONS.json. That file
records each engine version: V0.9b sets its engine to 0.9.0-dev.2 and adds
the version's explanation line, as every version does. V0.9b also adds two
session fixture files (`mini-seed52-v091`).

## 2. Parameter evidence v1 is frozen

`dreamteam.parameter-evidence/v1` is frozen as accepted at a5e5f9b.

* **The freeze record:** `analysis/evidence/frozen-parameter-v1.json`
  holds the SHA-256 of:
  * the analyzer and its tests;
  * the corpus and its 96 recordings;
  * both outputs;
  * the counts by rule.
* **The check:** `check_parameter_frozen.py` requires all of them byte for
  byte. CI runs it on every commit, on the parameter evidence it has just
  regenerated.
* **No further tuning.** The rules are not tuned against these recordings
  again. A rule change would be a new version, with its own outputs.
* **In-sample calibration.** Two drafts found 13 and then 7 "supported"
  mismatches, every one a false positive. The rules were narrowed on this
  same corpus.
* **The result:** 762 of 816 eligible checks were consistent, 54 were
  unresolved and 0 supported. That is an in-sample result with
  deliberately limited coverage. It is not evidence that agents never
  misstate parameters, and recall is not measured.

**The independent review is still PENDING.** No reviewer's files exist.

## 3. The financial evidence contract

`dreamteam.financial-evidence/v1` has three dimensions, each a separate
record with its own statuses. Each record carries:
* the recording, scenario and seed, and the policy and model (through its
  run);
* the decision point, the attempt ID where recorded, the action identity,
  the validation and execution outcomes;
* the observation and funding state, the engine's measurements,
  completeness, the rule version and a stable `fe1-` identifier.

The source recordings are only read. The ordering is deterministic.
**There is no composite score.**

**Every number is the engine's.**
* Each replay's transcript is fed back to the engine with read-only
  previews, observations and validations inserted, and every original
  answer must come back identical.
* The preview is the game's staff-screen preview. It gives:
  * the exact commitment, the headroom and the funding result;
  * guaranteed money lines with their timing and basis;
  * the batch's hires, terminations, roster and allocation changes,
    teleradiology, slots and displaced bookings.
* A shift extension's commitment is the reservation the engine added when
  it was applied.

**Result:** all 78 recordings replayed with every answer identical, on
3,540 decision points. The 77 llm-agent 2 final replies that were refused
were validated again, and each reproduced its recorded validation
exactly. One refused reply has no recorded validation.

**Knowable, not known.** Neither llm-agent 2 nor llm-agent 3 ever calls
`preview` (there is no such call in agents/llm). The agents saw:
* the observation's cash, liabilities, reserved commitments, headroom and
  credit;
* the validation answers.

The preview's exact commitments were available to them on request, but
never requested.

## 4. Historical analysis

Each group is its own economics. **Groups are never combined, models are
not ranked, and no row is a quality metric.** Full tables are in
[financial-tables.md](financial-tables.md).

### 4.1 Coverage

| Group | Recordings | Decision points | Submitted batches | Executed actions | Earlier attempts not recorded |
|---|---|---|---|---|---|
| V0.6c (no funding rule, no credit) | 40 | 1,800 | 281 | 396 | 151 |
| V0.7c ($50,000 credit, the funding rule) | 30 | 1,500 | 272 | 291 | 150 |
| V0.8c (as V0.7c, plus low liquidity) | 8 | 240 | 69 | 33 | 0 |

### 4.2 Affordability (denominator: submitted batches)

| Group | Funded without credit | Funded with credit | Refused: insufficient funds | No new commitment | Refused for another reason | Evidence unavailable |
|---|---|---|---|---|---|---|
| V0.6c (281) | 0 | 0 | 0 | 83 | 29 | 169 |
| V0.7c (272) | 180 | 30 | 0 | 22 | 39 | 1 |
| V0.8c (69) | 8 | 9 | 10 | 8 | 32 | 2 |

* **V0.6c** has no funding rule. Each of its 169 committing batches is
  `evidence_unavailable`, by definition, not by gap.
* **V0.7c** had no funding refusal. Of its 210 funded batches, 30 were
  funded only by the credit line: their commitment exceeded the headroom
  without undrawn credit.
  * Its 150 earlier llm-agent 2 attempts are not recorded. None of the
    refusal codes recorded for them is `insufficient_funds`.
* **V0.8c's 10 funding refusals** are all in the low-liquidity entry, on
  seed 41901, as V0.8c reported.
  * Their exact commitments ranged from $1,462.90 to $37,600.28.
  * The headroom at the refusal ranged from −$9,863.66 to $16,724.63.
* **Commitments the engine measured for executed batches:**
  * V0.7c, staffing: $187,948.07 in 29 batches funded without credit, and
    $75,121.76 in 4 funded with credit;
  * V0.7c, extensions: $73,250.87 in 150 batches, and $14,721.17 in 25;
  * V0.8c, staffing: $19,726.04 in 2 batches, and $25,000.47 in 3;
  * V0.8c, extensions: $2,728.23 in 4 batches, and $2,968.94 in 6.
* **Immediate ledger effects** of executed batches:
  * recruiting fees of $24,000.00 (V0.6c, 8 hires), $21,000.00 (V0.7c, 7)
    and $3,000.00 (V0.8c, 1);
  * severance of $10,945.00 (V0.6c, 3 terminations), $27,195.00 (V0.7c,
    7) and $15,600.00 (V0.8c, 4).
* **Recurring obligations** of executed batches:
  * contract guarantees per week;
  * salary allocation changes per 30 days;
  * open-ended teleradiology retainers.

  They are listed in financial-tables.md §2.
* **Credit usage.** Five runs drew on the line:
  * the V0.7c staffing-shortage Qwen run, stage B: at most $17,200.28,
    interest $14.14;
  * all four low-liquidity runs of V0.8c: at most $29,429.43 to
    $33,951.14, interest $17.82 to $25.53.

  No other run drew credit.

### 4.3 Reversibility (denominator: executed actions)

| Group | Withdrawable | Reversible, obligation retained | Not reversible | Reversible, no recorded cost | Unknown |
|---|---|---|---|---|---|
| V0.6c (396) | 15 | 31 | 342 | 8 | 0 |
| V0.7c (291) | 10 | 27 | 243 | 11 | 0 |
| V0.8c (33) | 8 | 2 | 22 | 1 | 0 |

* **Extensions dominate "not reversible".** No action withdraws an
  approved extension: 335, 232 and 18 by group. The others are
  terminations and contract cancellations.
* **The obligations retained on reversal are exact.** Each hire carries:
  * its recruiting fee, which is not refunded;
  * its earliest-exit amount, which totals $77,678.26 (V0.6c, 8 hires),
    $63,178.27 (V0.7c, 7) and $9,114.13 (V0.8c, 1).

  Teleradiology starts carry their guaranteed retainers: $38,945.80
  (V0.6c, 23 starts), $30,946.39 (V0.7c, 20) and $3,023.32 (V0.8c, 1).
* **No reversibility record rests on the rule alone** (720 of 720
  complete). Every executed staffing action had a matching preview item;
  extensions and policies need none.

### 4.4 Operational exposure

* **A, direct changes**, by resource (financial-tables.md §5):
  * extension minutes: 38,700, 28,320 and 1,920 by group;
  * template slots opened by hires;
  * site reading minutes changed by allocations;
  * booked appointments displaced by termination batches: 183 (V0.6c, 2
    batches), 991 (V0.7c, 6) and 72 (V0.8c, 3). The engine's preview
    states each of these figures at the decision.
* **B, subsequent observed state:** every run's timeline at every decision
  point, with no attribution (financial-tables.md §6).
* **C, paired with no action:** each run against no action on the same
  entry, seed, horizon and economics, from its contract report
  (financial-tables.md §7). These are differences between two whole runs,
  never allocated to a single action.

### 4.5 Evidence gaps

* **Earlier llm-agent 2 attempts:** 151 (V0.6c) and 150 (V0.7c) were not
  recorded. Their actions are unknown; their refusal codes are counted in
  financial-tables.md §8.
* **Affordability not established:** 169 (V0.6c: no funding rule), 1
  (V0.7c) and 2 (V0.8c). The last three are extension batches that were
  never applied, so the engine reserved nothing to measure.
* **Missing observations, previews or attempt links:** none.

## 5. Case studies

The full reconstructions are in [case-studies.md](case-studies.md),
generated from the evidence. In each case:
* what was observable at a decision point is shown apart from what was
  observed later;
* no reasoning is inferred;
* no decision is labelled irrational from its outcome.

### 1. V0.7c stage B, staffing shortage, Qwen (90 days, seed 41701)

Known at each decision:

| Date | Decision | What the engine stated or showed then |
|---|---|---|
| 10-14 | Hire an XR/CT technologist | Commitment $10,871.38, funded without credit. The headroom without credit was $55,840.89. |
| 10-16 | Raise Dr. Blake's allocation from 50% to 100% | $24,657.53 more salary per 30 days, withdrawable until that day. |
| 11-03 | Hire an MR technologist (premium) | Commitment $13,150.00. The headroom without credit was −$781.54, so the hire was **funded only by the credit line**. |
| 11-27 to 12-07 | Terminate seven employees in six batches: both new hires, the front desk and the four original technologists | No new commitment. Severance of $27,195.00 recognized at once, and none reversible. The previews stated 0, 47, 0, 141, 260 and 543 booked appointments displaced by the six batches: 991 in all. The headroom without credit was negative at each decision, from −$27,773.61 to −$71,645.52. |
| 01-10 (last point) | Hire two XR/CT technologists | Funded with credit, commitment $21,742.76. Their start (02-07) is after the run ends. |

Observed later, and not attributed:
* After the last termination took effect, no technologist was employed.
* Orders lost so far rose from 556 at 11-30 to 2,479 at the end.
* The line was drawn at most $17,200.28.

Paired with no action (C):
* profit −$181,588.63;
* orders lost +1,450;
* exams completed −867;
* lowest cash −$37,228.50.

These describe two whole runs. Allocating them to the terminations would
need runs that differ only in those decisions, which were not performed.

### 2. Contrast: V0.7c stage A, staffing shortage, Gemma (30 days, seed 41602)

* **One executed action:** a teleradiology contract, with a commitment of
  $3,093.33, funded without credit. Three later hire submissions were
  refused: twice for an invalid roster and once for lead time.
* **Paired with no action:** profit −$2,043.31. Orders lost and exams
  completed were unchanged.
* **Not a paired comparison with case 1:** the seed and the horizon
  differ.

### 3. V0.8c, low liquidity, Qwen (30 days, seed 41901)

* **From the first decision point,** the headroom without credit was
  negative (−$14,782.14 on $80,000.00 cash). Every commitment depended on
  the credit line.
* **Eight batches were refused for insufficient funds,** each shown with
  the engine's exact commitment and headroom at that moment:
  * three hires, each batched with an allocation change: $36,978.91,
    $37,600.28 and $12,114.13;
  * two allocation increases alone: $24,657.53 each;
  * a batch combining a termination, an allocation change and a
    teleradiology start: $3,036.67;
  * two teleradiology starts: $3,029.98 and $1,462.90.
* **What was executed:**
  * a premium CT hire batched with an allocation cut, funded with credit;
  * a second allocation cut batched with one termination;
  * five extensions.

## 6. Evidence limitations

1. **Historical attempts are missing.** llm-agent 2 (V0.6c, V0.7c)
   recorded only its final reply: 301 earlier attempts are unknown except
   for their refusal codes.
2. **Financial observations are absent** in V0.6c's economics: there is no
   funding rule and no credit, so affordability cannot be established
   there.
3. **Action parameters are incomplete** for extensions that were never
   applied: their commitment is not measured (3 batches).
4. **Reversibility cannot be established** beyond the simulator's rules.
   None are assumed. Restoring what a termination or cancellation removed
   would cost a new hire or contract, which is not computed because none
   was decided.
5. **Direct effects cannot be separated** from what followed within a run.
   B is observation only.
6. **Paired comparisons are unavailable** across groups, and between case
   studies with different seeds or horizons. Within a group, the pair is
   the whole run against no action, not one decision.
7. **A causal conclusion would require an unperformed intervention:** runs
   that differ in one decision only. None was run.
8. **The preview was never requested by the agents.** "Knowable" is not
   "known", and the analysis does not infer what an agent understood.
9. **Calibration.** The statuses are definitions applied to engine values,
   not tuned against outcomes. The corpus is in-sample development data,
   and no holdout was used or unsealed.

## 7. Reproducibility

* **Determinism:** two runs on linux/amd64 gave byte-identical outputs.
  CI regenerates them on linux/amd64, linux/arm64 and darwin/arm64,
  compares each with the committed files, and compares the platforms with
  each other.
* **No earlier output changed:**
  * the frozen decision evidence (`check_frozen.py`);
  * the V0.8b review package (`check_v08b.py`);
  * parameter evidence v1 (`check_parameter_frozen.py`);
  * the review page and status;
  * repeat v2 and refusal-response v2;
  * benchmark verify and reproduce;
  * the golden references.
* **No model** was contacted. No clinical suite was rerun.

## 8. What V0.9b answers

**Yes, within the simulator's own terms.** At each decision point, these
were established exactly from what was knowable then:
* the exact commitment and whether it rested on the credit line;
* what a reversal would leave owed;
* which resources the action changed directly. For a termination, that
  includes how many booked appointments it displaces.

In case 1, the credit-funded MR hire and the termination batches' 991
displaced bookings were stated by the engine before any of the losses
were observed.

**What V0.9b cannot answer** is how much of a run's later loss any one
decision caused.

## 9. Recommended V0.9c scope

A prospective evaluation with frozen models that asks whether agents
respond to these constraints when they are shown, without changing the
simulator:
* **Use the existing preview.** Record, with llm-agent 3's complete
  attempts, whether the agent requests a preview before a commitment.
  Compare it with an otherwise identical agent configuration, each
  declared in advance as a separate agent version.
* **Score nothing new.** Report the three dimensions as in V0.9b, by
  scenario and seed, with denominators, against no action.
* **Design single-decision interventions,** declared in advance, to
  measure what this milestone could not:
  * paired runs that differ in one decision only, such as a termination
    batch withheld;
  * on seeds not used before, with the holdout still sealed.
* **Keep the frozen rules frozen.** Any new rule is a new version.
