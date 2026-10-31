# V0.8c: complete decision recording and a prospective funding-refusal evaluation

V0.8c (0.8.0-dev.3), authorized by the owner on 2026-10-30 after accepting
V0.8b. Not released.

**Restrictions:**
* no prompt or model tuning;
* no new simulator mechanic;
* no change to the frozen `dreamteam.decision-evidence/v1` rules;
* at most eight live LLM runs.

**Order:**
* The plan ([PLAN.md](PLAN.md)) was committed before any live run
  (ee8aabc).
* The plan has no amendments.

**It answers three questions:**
* Does llm-agent 3 record every attempt, including rejected ones?
* Can a repeated refused proposal be recognized by what was proposed, not
  by the refusal's words?
* What do the two frozen models propose after a real funding refusal?

## 1. The V3 recording (llm-agent 3)

**What it is:**
* `agents/llm/llm_agent3.py` writes `dreamteam.llm-decision-record/v3`
  ([LLM_AGENT.md §13](../../LLM_AGENT.md)).
* `agents/llm/llm_agent.py` (llm-agent 2) is unchanged, byte for byte
  (SHA-256 `5214e7bf…`).

**What llm-agent 3 keeps unchanged:** llm-agent 2's system prompt, prompt
(`llm-agent-prompt/v2`), memory (`llm-agent-memory/v1`), settings,
retries, details turn, batching and fallback. It sends the model the same
messages. Three checks show this:
* `test_llm_agent3.py`: given the same model answers, versions 2 and 3
  make the same requests, decisions and memory.
* `llm_agent3_test.go`: against a mock server, the same simulated
  practice, digest for digest.
* The frozen v1 analyzer rebuilt the llm-agent 2 prompt at all 240 V0.8c
  decision points and verified it against llm-agent 3's recorded SHA-256:
  240 of 240.

**What every attempt records:**
* its identifier (`point.attempt`), turn and input (by SHA-256; a retry
  also keeps its feedback);
* the transport result and the model's output (bounded);
* the parse, reason and actions;
* every validate answer, the refusals and the error;
* whether it was the attempt used.

**What the decision point records:** the used attempt (`final`), the
submitted batches and a completeness block. Attempts are also appended to
`attempts.jsonl` as they are made.

**What is kept apart:**

| Stage | Where |
|---|---|
| Model output | `output.text` |
| Parsed actions | `actions` |
| Validated actions | `validation` |
| Submitted actions | `batches` |
| Applied actions | the harness's `decisions.json` |

**Excluded:** no endpoint, host, port, key, latency or token count is in
the record. Those stay in the live, uncommitted evidence.

**Completeness in the evaluation** (all 8 live runs):

| | Result |
|---|---|
| Decision points | 240 |
| Every attempt recorded | 240 of 240 |
| Every input verifiable | 240 of 240 |
| Attempts | 270, all with their output; 268 with a parsed reason and actions (two replies could not be parsed) |
| Transport failures | 0 |
| Infrastructure-affected points | 0 |
| Truncated fields | 0 |

## 2. Historical compatibility

* **llm-agent 2 and its recordings are unchanged.** Every historical
  replay (V0.5a–V0.7c) reproduces, and the frozen v1 analyzer, tables,
  traces and evidence reproduce (`check_frozen.py`).
* **Version 3 replays only version 3 records.** It refuses a version 2
  recording, and names `llm_agent.py` as the agent that replays it.
* **Simulation state and model metadata:** the replays of the 8 live runs
  call no model. Their sessions, results and decisions equal the live
  runs' byte for byte, ledger, credit and interest included. Live
  metadata never enters the record.
* **A build number in two frozen V0.8b outputs.** The V0.8b review package
  and probe log record the build that made them (0.8.0-dev.2). They are
  frozen, so they are not regenerated. `check_frozen`'s approach is
  extended to them (`check_v08b.py`): every byte must match except those
  build numbers.

## 3. Repeat rule version 2

**What it is:** `dreamteam.repeat-evidence/v2`
([DECISION_EVIDENCE §14](../../DECISION_EVIDENCE.md)). It recognizes a
repeated refused proposal by the action's exact content, not the refusal
message:
* it abstains explicitly where the evidence is incomplete;
* it counts same-target, other-value proposals as `related`, never as
  repeats.

**Unchanged:** v1's rule and its findings.

**Historical recordings** (559 refusals,
[history/repeat-v2-tables.md](history/repeat-v2-tables.md)):

| Status | Refusals |
|---|---|
| insufficient_evidence (the refused attempt's actions not recorded) | 452 |
| unresolved (a later attempt's actions not recorded) | 93 |
| not_repeated | 14 |
| repeated_refused | 0 |

**Versus v1 on the historical recordings:**
* v1 made 202 repeat links.
* v2 can confirm none of them:
  * at 194, at least one end's actions are not recorded;
  * at 8, both ends are recorded and the actions differ. Different actions
    refused with the same message were merged: a new defect, **V1-D5**, in
    the [freeze record](../v08a-dev/FREEZE.md).

**The V0.8b probes:**
* Read attempt by attempt, v2 finds the repeat across decision points that
  v1 missed (V1-D1).
* Read in the llm-agent 2 format, it abstains on 6 of the 7 refusals.

**V0.8c's complete records** (51 refusals; every attempt recorded, so v2
never abstains):
* *v2:* 1 repeat (the same extension refused `after_close` twice within
  one decision point), 50 not repeated.
* *v1:* 13 repeat links:
  * 1 is v2's;
  * 12 join different actions refused with the same message (V1-D5). The
    commonest case is extensions for different staff or dates refused
    `after_close`.

**Exact identity is strict.** Qwen proposed Dr. Blake's allocation raise
to 100% at four decision points, each time effective from that day.
Gemma proposed a teleradiology contract at two consecutive points, the
second time effective one day later. v2
counts those as `related`, and refusal-response counts them as
`modified_proposal`. Neither is called a repeat, by design. The report
does not decide whether such a proposal is "the same" one.

## 4. Independent review status

**REVIEW STATUS: PENDING.**
* The frozen 98-item package
  ([../v08b-dev/review/](../v08b-dev/review/PROTOCOL.md)) is unchanged:
  not relabelled, not resampled.
* No independent reviewer has completed it, so no review result is
  reported.
* No precision or recall is claimed, and no LLM substitutes for the
  reviewer.

**What it means for the findings:** every v1 finding, historical or V0.8c,
is unvalidated.

## 5. The prospective declaration

[PLAN.md](PLAN.md) (`v08c-plan/1`, ee8aabc, before any live run) and
[eval.json](eval.json):

| | |
|---|---|
| Conditions | control `financial-pressure-credit` (fingerprint `586f3ef7…`): $150,000 opening cash, $50,000 credit line at 10%, the funding rule; low liquidity `financial-pressure-lowliquidity` (fingerprint `3175656e…`): the same with $80,000 opening cash |
| Low-liquidity rule | opening cash lowered by the control's lowest no-action funding headroom on analysis seed 41900, rounded down to $10,000 ([scenario_analysis.py](scenario_analysis.py), [scenario-analysis.json](scenario-analysis.json)); no policy, no model, not changed after any result |
| Seeds | 41901, 41902 (fresh); 41900 for the scenario analysis only |
| Horizon, cadence | 30 days, daily at 07:00 |
| Models | `Qwen3.6-35B-A3B-oQ4e-mtp`, `gemma-4-26B-A4B-it-oQ4e-mtp`, the frozen V0.5c settings |
| Agent | llm-agent 3 (`ef2c17a0…`); prompt v2, memory v1, memory on |
| Deterministic comparators | `noop`, `greedy`, `conservative-v2` |
| Runs | 20, of which 8 live LLM runs |

**Under no action on the analysis seed:**
* the control's funding headroom was $76,093 to $105,357;
* the variant's was $6,080 to $35,357, drawing up to $21,726 of credit,
  with no distress.

## 6. Live evaluation results

All 20 runs completed on the first attempt (2026-10-30, 22:08 to 22:37,
Raspberry Pi 4).
* There were no failures, reruns or transport errors.
* Both models were served.
* `benchmark verify`: 194 checks, 0 failed.

**Funding refusals encountered** (`insufficient_funds` from validate, in
any attempt, out of the actions proposed in every parsed attempt):

| Condition | Seed | Qwen3.6-35B-A3B | Gemma 4 26B-A4B | greedy (refused at apply) |
|---|---|---|---|---|
| control | 41901 | 0 of 28 | 0 of 0 | 0 of 4 |
| control | 41902 | 0 of 26 | 0 of 0 | 0 of 4 |
| low liquidity | 41901 | **8** of 44 | **2** of 5 | 2 of 4 |
| low liquidity | 41902 | 0 of 22 | 0 of 0 | 2 of 4 |

* **Where the refusals occurred:** only in the low-liquidity condition,
  and only on seed 41901 for the LLM agents. Seed 41902's runs met none.
* **Which proposals were refused:**
  * Qwen: four allocation raises (Dr. Blake to 100%; two of them in
    batches that also held hires), three teleradiology contracts and one
    hire;
  * Gemma: two teleradiology contracts.
* **`conservative-v2` and no action** proposed nothing that needed funding.

**Attempts per decision point:**

| Condition | Model | Seed 41901 | Seed 41902 |
|---|---|---|---|
| control | Qwen | 1.20 (36 attempts / 30 points) | 1.20 |
| control | Gemma | 1.00 | 1.00 |
| low liquidity | Qwen | 1.40 (12 points with two attempts) | 1.10 |
| low liquidity | Gemma | 1.10 | 1.00 |

## 7. Refusal-response observations

**The 10 funding refusals** (dreamteam.refusal-response/v2):

| Response | Count |
|---|---|
| Refused and repeated (identical) | 0 |
| Refused and revised: modified (same type and target, other values) | 2 |
| Refused and revised: different proposal | 3 |
| Refused, then no action | 5 |
| Refusal no longer applicable | 0 |
| Evidence unavailable | 0 |

* **Run level:** six of the eight LLM runs met no funding refusal.
* **Every refusal was classifiable,** against 59 of 559 (10.6%) in the
  historical recordings under the same reader.
* **Refusals of every code** (51, all runs): repeated 1, modified 13,
  different 16, no action 21, no longer applicable 0, unavailable 0.

**What the categories describe:**
* *The two "modified" funding responses* changed only the effective date,
  by one day: Qwen's raise and Gemma's teleradiology contract. Both were
  refused again.
* *The "no action" responses* are retries or next points that proposed
  nothing.
* None of this says whether a model understood a refusal. A modified or
  different proposal is not called adaptation, and none is called
  successful.

**The historical recordings, read again (refusal-response v2,
[history/](history/refusal-response-v2-tables.md)):** of 559 refusals:
* 59 are classifiable;
* 500 are evidence-unavailable:
  * 452 because the refused attempt's actions are not recorded;
  * 43 because the next proposal's actions are not recorded;
  * 5 because no proposal follows.

**V0.8b's 102 "classified" includes two kinds of reading that v2 does not
make:**
* *43 refusals* whose response v1 read from the next point's final reply,
  although an earlier proposal at that point had gone unrecorded.
* *2 refusals that are a defect, RR1-D1.* v1 assigned the parsed reply to
  the last attempt even when that attempt's reply could not be parsed, so
  each refusal was compared with itself: the two historical "identical
  repeats".

V0.8b's committed v1 figures are kept as they are. v2 is reported beside
them.

## 8. Financial and operational outcomes

From [v08c-tables.md](v08c-tables.md). Every difference is policy minus no
action on the same condition and seed. It describes two runs over 30
days, not the effect of any decision.

**Financial outcomes** (each figure is profit vs no action, with the most
credit drawn):

| Condition | Seed | No action: profit, most credit drawn | greedy | Qwen | Gemma |
|---|---|---|---|---|---|
| control | 41901 | −$20,009, $0 | −$1,271, $0 | −$21,221, $0 | $0, $0 |
| control | 41902 | −$13,911, $0 | −$598, $0 | −$1,699, $0 | $0, $0 |
| low liquidity | 41901 | −$20,027, $29,429 | −$1,271, $29,429 | −$20,967, $33,668 | $0, $29,429 |
| low liquidity | 41902 | −$13,934, $32,220 | −$598, $32,220 | −$8,515, $33,951 | $0, $32,220 |

* **Distress:** none in any run (shortfall $0, no day below zero).
* **Credit:** drawn in every low-liquidity run, no action included, at the
  month's disbursements. It was drawn on 4 or 5 days, with $18 to $26 of
  interest. It was never drawn in the control.
* **`conservative-v2`** took no action in any run.

**Operational outcomes vs no action:**

| Condition | Seed | Qwen: exams, lost orders, TAT breaches, TAT p90 | greedy: breaches |
|---|---|---|---|
| control | 41901 | −42, +40, +186, +2,056 min | −6 |
| control | 41902 | +2, 0, +2, 0 | −2 |
| low liquidity | 41901 | 0, 0, +208, +1,672 min | −6 |
| low liquidity | 41902 | 0, 0, +4, −12 min | −2 |

* **Gemma:** the same as no action on every operational measure. It
  executed one action in all: a teleradiology contract, accepted on Nov 10
  and effective Nov 24, after the horizon.
* **Qwen, seed 41901, both conditions:** this is where the large service
  differences occurred.
  * Under the control, Qwen lowered both radiologists' site reading
    allocation to 30% from Oct 31. It also terminated a technologist (its
    stated reason: "reduce the cost base by terminating the most expensive
    technologist").
  * Under low liquidity, it lowered Dr. Avery's allocation to 30% from Oct
    31 and Dr. Blake's from Nov 19.
  * These co-occur with the differences above. No causal attribution is
    made.

**Stated, submitted and executed actions** (Qwen; Gemma submitted one):

| Condition | Seed | Stated commitments | Submitted | Executed (with a change) |
|---|---|---|---|---|
| control | 41901 | 6 | 12 | 9 |
| control | 41902 | 5 | 9 | 9 |
| low liquidity | 41901 | 21 | 11 | 9 |
| low liquidity | 41902 | 3 | 6 | 5 |

Every action the LLM agents submitted was accepted at apply. Greedy's four
refusals at apply were all `insufficient_funds`, in the low-liquidity
condition.

**The frozen v1 rules on the V0.8c records** (llm-agent 2 view):
* *Findings:* 17 supported (7 repeated invalid proposal, 8 stated action
  not executed, 2 factual mismatch) and 35 unresolved (31 of them amount
  mismatches the rules could not resolve).
* *Prompts:* 240 of 240 verified.
* *Status:* not independently validated.

## 9. Replay and cross-platform evidence

* **Replays of every run**, the deterministic comparators included, with
  no model:
  * [eval-replay-financial-pressure-credit.json](eval-replay-financial-pressure-credit.json)
  * [eval-replay-financial-pressure-lowliquidity.json](eval-replay-financial-pressure-lowliquidity.json)
  * Both verified: 104 checks each, 0 failed.
* **Equal to the live runs:** sessions, results and decisions match byte
  for byte. So does the financial ledger, with its credit draws,
  repayments and interest. The contract metrics also match (live versus
  replay, `replayMatchesLive`).
* **What llm-agent 3's replay checked:**
  * at every point: the digests, memory and prompt;
  * every attempt's input: 270 inputs;
  * every validate answer, asked again of the engine and compared in full.
* **Reproducibility:**
  * `benchmark reproduce` of both replays gives byte-identical manifests on
    linux/amd64 (horo) and linux/arm64 (Pi).
  * CI replays them on linux/amd64, linux/arm64 and darwin/arm64 and
    compares every output across platforms.
* **Regenerated in CI and compared with the committed files:**
  * the low-liquidity scenario analysis;
  * the historical refusal-response v2 and repeat v2 outputs;
  * the V0.8c analysis.
* **Unchanged:**
  * historical recordings, the frozen v1 evidence, golden references;
  * benchmark fingerprints: suite version 3's entries are unchanged, and
    suite version 4 only adds one.

## 10. Quality gates

See [PROGRESS.md](../../PROGRESS.md), V0.8c.

## 11. Known limitations

**Of the evaluation:**
* *Size:* two seeds, 30 days, one practice in two conditions, two models.
  Funding refusals came from 2 of 8 live runs, all on one seed. Nothing
  here generalizes beyond these runs.
* *What a response category is:* a description of the next proposal, not
  of intent. The "modified" responses changed only a date.
* *Applied actions* come from the harness's record. The agent never sees
  the apply answer.

**Of the evidence:**
* *Historical recordings* remain mostly unclassifiable. llm-agent 3 fixes
  this for new recordings only.
* *The v1 findings* are unvalidated: the independent review is pending.
  V1-D1 to V1-D5 apply to every v1 finding.
