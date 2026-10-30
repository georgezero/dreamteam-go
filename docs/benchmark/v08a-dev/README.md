# V0.8a: decision evidence in the recorded agent decisions

V0.8a (0.8.0-dev.1). This is a read-only, deterministic analysis of every
committed LLM agent recording, V0.5a to V0.7c. It uses the specification
[docs/DECISION_EVIDENCE.md](../../DECISION_EVIDENCE.md)
(`dreamteam.decision-evidence/v1`, analyzer `decision-evidence 1`).

**How it was produced:**
* **No new run:** no model was called, and nothing in the simulator, the
  agents, the recordings or the scoring contract was changed.
* **Inputs:** each recording was replayed by its existing replay
  configuration, without any model, under 0.8.0-dev.1. The analysis read
  the recordings and the replays' transcripts and apply results.
* **What the model saw:** for each decision point it rebuilt the prompt and
  checked it against the recorded SHA-256.

**Files:**
* [corpus.json](corpus.json): the recordings, the replay configurations
  and the replay output names.
* [evidence.json](evidence.json): every finding, the coverage, and the
  count of every check.
* [tables.md](tables.md): all tables, including every supported finding
  with its basis.

**The question.** Does what an agent says agree with what it does? This is
a different question from whether its decisions were good.

**How to read the counts:**
* A count of findings is not a measure of model quality. An agent that
  says more, and does more, has more statements to check.
* Every count below stands beside the number of statements checked.
* Nothing here is a ranking. Nothing generalizes beyond these recordings,
  seeds, two models (a third in V0.5c) and one agent version.

## 1. Evidence coverage

| Group | Role | Recordings | Decision points | Reason missing | Prompt rebuilt and verified | Observation | Points with submitted actions | Apply results |
|---|---|---|---|---|---|---|---|---|
| V0.5a | secondary | 1 | 7 | 0 | 0 | 0 | 2 | 2 |
| V0.5b | secondary | 1 | 14 | 0 | 14 | 14 | 6 | 6 |
| V0.5c | primary | 18 | 651 | 0 | 651 | 651 | 63 | 63 |
| V0.6c | primary | 40 | 1,800 | 8 | 1,800 | 1,800 | 231 | 231 |
| V0.7c | primary | 30 | 1,500 | 4 | 1,500 | 1,500 | 213 | 213 |

**Prompts.** Every prompt of every llm-agent 2 recording (3,965 of 3,965)
was rebuilt and matched its recorded SHA-256. So the observation checks
compare each reason with exactly the text its model was shown.

**V0.5a** (llm-agent 1, 7 points) cannot be rebuilt, and its replay
requests no observation. Its observation checks are therefore
`insufficient_evidence`.

**Not available anywhere:** the earlier attempts' replies. They are not
in the repository (DECISION_EVIDENCE §3.1).

## 2. Checks and findings

### 2.1 By rule, all groups

| Rule | Statements checked | Consistent | Supported | Unresolved | Insufficient evidence |
|---|---|---|---|---|---|
| A. EXPLICIT_ACTION_OMITTED | 525 | 480 | 5 | 40 | 0 |
| B. EXPLICIT_OPPOSITE_ACTION | 672 | 669 | 0 | 3 | 0 |
| C. FINANCIAL_AMOUNT_MISMATCH | 235 | 206 | 3 | 26 | 0 |
| D. REPEATED_INVALID_PROPOSAL | 342 | 235 | 107 | 0 | 0 |
| E. STATED_ACTION_NOT_EXECUTED | 352 | 307 | 45 | 0 | 0 |
| F. FACTUAL_OBSERVATION_MISMATCH | 5,139 | 5,101 | 27 | 2 | 9 |

**How each rule counts:**
* **A, B and E:** explicit statements of intention. A and E count
  commitments; B counts refusals to act and statements of no action, plus
  directional commitments.
* **C:** dollar amounts.
* **D:** distinct refused proposals.
* **F:** specific factual claims.

### 2.2 Primary corpus (V0.5c, V0.6c, V0.7c), by model

| Model | Decision points | Rule | Statements checked | Consistent | Supported | Unresolved |
|---|---|---|---|---|---|---|
| Qwen3.6-35B-A3B | 1,927 | A | 369 | 333 | 3 | 33 |
| | | B | 92 | 89 | 0 | 3 |
| | | C | 220 | 203 | 3 | 14 |
| | | D | 272 | 190 | 82 | 0 |
| | | E | 224 | 191 | 33 | 0 |
| | | F | 2,987 | 2,973 | 13 | 1 |
| Gemma 4 26B-A4B | 1,837 | A | 140 | 131 | 2 | 7 |
| | | B | 578 | 578 | 0 | 0 |
| | | C | 15 | 3 | 0 | 12 |
| | | D | 62 | 38 | 24 | 0 |
| | | E | 117 | 106 | 11 | 0 |
| | | F | 2,005 | 1,991 | 13 | 1 |
| Gemma 4 31B (V0.5c only) | 187 | A | 2 | 2 | 0 | 0 |
| | | D | 1 | 1 | 0 | 0 |
| | | E | 2 | 2 | 0 | 0 |
| | | F | 114 | 114 | 0 | 0 |

**Qwen** includes V0.5c stage C's three memory-off recordings (§3).

**Gemma 4 26B's B count** is almost entirely statements of no action:
578, all consistent. The model rarely acted.

### 2.3 What was found, rule by rule

**A. Explicit action omitted: 5 supported.** In each case, a reply states a
specific action and proposes none like it:

| Model | Recording | Point | The statement |
|---|---|---|---|
| Qwen | V0.5c stage C, s40803 (memory off) | 28 | "I will also extend Riley's shift slightly"; the reply proposes three roster changes |
| Qwen | V0.6c 30-day stable, s41202 | 7 | "I will reduce Dr. Avery's site allocation to the minimum (3000 BP)"; the reply proposes only Casey's extension |
| Qwen | V0.6c 90-day financial pressure, s41301 | 35 | "I will cancel the teleradiology contract"; the reply proposes only an extension (the cancellation was proposed at the next point) |
| Gemma 26B | V0.7c stage B staffing shortage, s41701 | 21 | "I will extend Morgan's shift today"; later "I will take no action"; no action proposed |
| Gemma 26B | V0.7c stage B staffing shortage, s41701 | 29 | "I will extend Morgan's shift"; then "I will wait"; no action proposed |

**B. Explicit opposite action: none supported.** In 672 statements:
* no allocation stated as raised was lowered, and none stated as lowered
  was raised;
* no contract stated as started was only cancelled, or the reverse;
* no refusal to act was contradicted by the same reply;
* no unqualified statement of no action came with proposed actions.

**C. Financial amount mismatch: 3 supported.**

| Model | Recording | Point | Stated | Shown in the prompt |
|---|---|---|---|---|
| Qwen | V0.6c 90-day financial pressure, s41301 | 35 | "$150k/month retainer" | the contract's `monthlyRetainerCents` 150000 ($1,500). The stated amount is the cents read as dollars. |
| Qwen | the same | 36 | "$150k/month teleradiology retainer" | the same |
| Qwen | V0.7c stage B staffing shortage, s41701 | 59 | "a $21,336 payroll due on 12-25" | `nextPayroll.cents` 213356 ($2,133) |

* **206 bound amounts agreed** with the observation at their stated
  precision. For example, "operating at a significant loss (-$11,201 MTD)"
  when the prompt showed that month-to-date profit.
* **26 amounts are unresolved:**
  * 25 bound to no quantity (weekly savings, overtime costs, "monthly
    operating profit", which may mean month to date or trailing 30 days);
  * 1 because the stated figure appears elsewhere in the prompt.

**D. Repeated invalid proposals: 107 supported.** These are repetition
patterns (§3).

**E. Stated action not executed: 45 supported.** Each is an unflagged
statement whose matching proposed action did not take effect:

| How it failed | Qwen | Gemma 26B | V0.5b |
|---|---|---|---|
| Refused by validate | 15 | 10 | 1 |
| Valid, but another action in the reply was refused, so nothing was submitted | 15 | 0 | 0 |
| Refused with `batch_rejected` beside a refused action in its batch | 3 | 1 | 0 |
| Refused when applied | 0 | 0 | 0 |

No stated action that was submitted was refused when applied.

**F. Factual observation mismatch: 27 supported** (26 in the primary
corpus, one in V0.5b).

| Claim | Qwen | Gemma 26B | Example |
|---|---|---|---|
| Weekday | 8 | 0 | "currently closed (Sunday" on a Tuesday |
| Unread studies | 2 | 5 | "a reading backlog of 22 studies" when the worklist shown has `"unread":0` and no items |
| Overdue, total | 3 | 1 | "13 studies, 12 overdue" when the worklist shown is empty |
| Overdue by priority | 0 | 7 | "8 overdue routine reports" when the alert reads "6 routine reports overdue" |

* **Consistent:** 5,101 claims agreed with the observation shown. Most
  of the claims checked were backlog counts (unread, overdue) and
  statements that the practice was closed at 07:00.
* **Insufficient evidence:** the 9 V0.5a claims.

### 2.4 Unresolved and insufficient evidence

**Unresolved: 71.** Each is a candidate the rules could not decide:

| Rule | Unresolved | Why |
|---|---|---|
| A | 40 | The statement is hedged, deferred, a question, revised in the text, said elsewhere to be impossible, about a setting already in effect, or has no single named target. In 19 of them the reason revises itself ("Wait, …"). |
| B | 3 | One hedged statement of no action beside a stated action; two refusals to act whose target could not be resolved |
| C | 26 | See C above |
| F | 2 | A total beside counts by priority in the same sentence, which may be a remainder |

**Insufficient evidence: 9**, all the V0.5a claims (no prompt to compare
with).

### 2.5 How the rules were checked

The rules were written and calibrated on this corpus; there was no
held-out set.

**Calibration.**
* While the rules were developed, every supported A, B, C and F finding
  was read beside its rebuilt prompt.
* Each false positive led to a rule change, made before these figures
  were produced:
  * a question read as a commitment;
  * a sentence or later sentence that retracts the action;
  * "already set";
  * a modality subset read as the total;
  * a remainder read as the total;
  * an amount that appears elsewhere in the prompt.

**After the last change:**
* Every supported A, B, C and F finding (35) was read again.
* So was a seeded sample of E and D findings.
* So was a seeded random sample of 30 of the 352 commitments judged
  consistent: all 30 matched the right action type and target.

This is the author's own review, not an independent one.

## 3. Repeated invalid proposals

**What counts as the same proposal.** A refused proposal is identified by
its action type, refusal code and message, with dates normalized. So
"Casey is not rostered today." refused on ten different days is one
proposal refused ten times. Primary corpus:

| | Qwen, memory on | Qwen, memory off (V0.5c stage C) | Gemma 26B |
|---|---|---|---|
| Recordings | 40 | 3 | 40 |
| Validate refusals (`batch_rejected` excluded) | 352 | 74 | 122 |
| Distinct refused proposals | 236 | 36 | 62 |
| Refused more than once | 66 | 16 | 24 |
| Refusals in those repeated groups | 182 | 54 | 84 |
| Most repeats of one proposal | 7 | 6 | 10 |
| Repeats within a decision point's retries | 16 | 6 | 1 |
| Repeats at a later decision point | 100 | 32 | 59 |
| … with that refusal in the memory shown at that point | 93 | 0 | 53 |
| … with that refusal not shown | 7 | 32 | 6 |
| Repeated proposals with a schema code | 6 | 1 | 4 |

**What this observes:**
* **Memory on:** most repeats at a later decision point (146 of 159)
  happened while the earlier refusal was in the memory shown to the model.
* **Memory off:** none of the 32 repeats had it shown, as designed.
* **Schema or rules:** most repeated proposals were refused under the
  practice's rules, not the action schema:
  * rule codes, such as `after_close`, `not_on_shift`,
    `exceeds_max_shift`, `lead_too_short`, `contract_overlap` and
    `notice_too_short`: 95 of 106;
  * schema codes (`invalid_minutes`, `invalid_roster`, `invalid_schema`):
    11.

**What this does not show:**
* why a model repeated a proposal;
* whether it "retained" or "understood" anything.

V0.7c's figure of up to 27 repeated invalid proposals counted something
else: proposals refused by validate in a run. The figure here counts one
identical refused proposal.

## 4. Representative examples

Each example can be reproduced:
1. Run its replay configuration (`benchmark suite run`, from the
   repository root, as in CI) into the output directory the corpus names.
2. Run the analysis.
3. Open the finding by its `id` in `evidence.json`.

The finding's `sources` name the recording line, the transcript, the
apply results and the prompt's SHA-256.

**1. Stated, not proposed** (A, `de1-c9470e4e65386ec4`)
* **Recording:** V0.7c stage B, staffing shortage, Gemma 4 26B, seed 41701
  (`docs/benchmark/v07c-dev/stage-b/staffing-shortage-credit-gemma4-26b-a4b-s41701-decisions.jsonl`,
  line 22, point 21).
* **Replay:** `stage-b-replay-staffing-shortage-credit.json`, run
  `bench-staffing-shortage-ot-credit-s41701-gemma4-26b-a4b`. Prompt
  `0963f932…` verified.
* **Reason:** "I will extend Morgan's shift today to increase US capacity
  and address potential demand, although the primary bottleneck is MRI.
  Since no other immediate operational or staffing changes are valid for
  today, I will take no action to avoid unnecessary costs while monitoring
  the MRI gap."
* **Proposed:** nothing (`no_action`).

**2. Two inconsistencies in one reply** (A `de1-d8566d59ad338f3a`, C
`de1-57407c546dfa94d3`)
* **Recording:** V0.6c 90 days, financial pressure, Qwen, seed 41301
  (line 36, point 35). Prompt `e6a58ce4…` verified.
* **Reason:** "…a teleradiology contract (T00001) costing $150k/month
  retainer that is underutilized … I will cancel the teleradiology
  contract to stop the bleeding … I will also extend Dr. Avery's shift…"
* **Proposed:** only the extension.
* **The amount:** the prompt's contract shows `"monthlyRetainerCents":150000`,
  that is $1,500. The stated $150k is those cents read as dollars.
* **The next point (36):** the same reply cites the same $150k and
  proposes the cancellation (`de1-c6b2050d2d3612d2`). This is the misread
  V0.6c reported. Here it is found mechanically.

**3. Stated, proposed, refused** (E, `de1-8f9aab5ad5ca1432`)
* **Recording:** V0.7c stage A, demand growth, Gemma 4 26B, seed 41601
  (line 21, point 20).
* **Reason:** "I am starting a teleradiology contract for weeknights…"
* **What happened:** the proposed start was refused by validate
  (`contract_overlap`), the decision fell back to no action, and nothing
  was submitted.
* **The finding records** the action, the validate code, and
  `submitted: false`.

**4. A claim the observation contradicts** (F, `de1-2203082f2de06f76`)
* **Recording:** V0.6c 90 days, demand growth, Gemma 4 26B, seed 41301
  (line 36, point 35). Prompt `8371561d…` verified.
* **Reason:** "The practice has a significant reading backlog of 22
  studies, with 21 routine reports and 1 'soon' report overdue."
* **Prompt:** the worklist reads `"unread":0,"reading":0,…,"items":null`,
  and the active problems list no backlog.

**5. The V0.7c staffing-shortage case** (Qwen, stage B, seed 41701)

V0.7.0 recorded this run as one development observation. Against no
action, its profit was −$181,589, after terminations from 11-27.

*What the evidence separates:*
* **Commitments and execution:** in the whole run, no commitment was
  found omitted, opposite or not executed (A, B and E). The terminations
  it stated were proposed and accepted:
  * points 45 to 55: `accepted`, with the engine's messages, for example
    "Casey (E05) terminated: last day Sat Dec 19, 14 days' notice with
    contract pay continuing ($2,250/week guaranteed)";
  * point 59, where the seven employees were already leaving:
    `accepted_unchanged`.
* **A misstated amount:** at point 59 the reason says "a $21,336 payroll
  due on 12-25". The prompt showed `nextPayroll.cents` 213356 ($2,133)
  (C, `de1-fb2661ae251a19e6`).
* **A misstated weekday:** at point 7 the reason calls the day Sunday; it
  was Tuesday (F, `de1-269c4ab7d42958c1`).
* **Repeats:** two refused proposals were repeated: a hire `lead_too_short`
  three times, and an extension `after_close` four times.

*What the evidence does not say:*
* The harmful outcome is not an inconsistency. As far as these rules
  read, the agent did what it said.
* Two of its stated facts disagree with what it was shown.

*Consequence links.* The decision traces give each point's apply results
(direct) and the next point's observation (subsequent, not attributed).
For example, point 53 terminated Casey; the next observation showed
`examsSinceStart` +26 and `unread` +11 over the day. The traces present
that as an observation after the day's events, not as an effect of the
termination.

**6. Repetition while the refusal is shown** (D, `de1-91c85d886024aa41`)
* **Recording:** V0.7c stage B, staffing shortage, Gemma 4 26B, seed
  41701.
* **The proposal:** an extension for Casey refused `not_on_shift` ("Casey
  is not rostered today.") ten times, at points 4, 40, 46, 47, 53, 54, 60,
  74, 81 and 88.
* **The memory:** at eight of the nine repeats, the memory shown at that
  point held the refusal.

**7. Abstention** (A, unresolved)
* **Reason:** "I will also extend Dr. Blake's shift? No, Dr. Blake is
  already until 20:00. I will extend Dr. Avery to 17:00." (V0.7c stage A,
  reporting backlog, Qwen, seed 41601, point 2.)
* **Outcome:** a question, followed by a revision. The rules do not read
  it as a commitment to extend Dr. Blake.
* **Another kind:** "I will start the process of expanding teleradiology
  coverage …, but since … I cannot start a new one yet" (`says_cannot`)
  is unresolved too.

**8. Insufficient evidence.** V0.5a's 7 decision points (llm-agent 1)
have no rebuildable prompt. Its 9 factual claims are
`insufficient_evidence`, and no observation finding is made for it.

## 5. The question V0.8a set

**Can the evidence tell a bad decision from an inaccurate description?**

**For explicit commitments, yes, within the rules' reach:**
* Across 525 explicit commitments, the agents' stated actions were
  proposed in 480. Only 5 were plainly omitted, and none was contradicted
  by the opposite action.
* When a stated action failed (45), the failure was a validate refusal or
  a refused batch partner, recorded as such. It was not a stated action
  silently replaced by another.
* The V0.7c staffing-shortage run is an example of a harmful sequence of
  decisions that was consistent with its stated intent.

**The descriptions of the state are a separate matter:**
* The models misstated facts the prompt showed: 3 amounts and 27 claims
  (weekdays, backlog counts, overdue counts).
* Some cited a backlog the observation did not show. The analysis does
  not say where such a figure came from.

**Repetition is a third pattern:**
* A refused proposal was put forward again, mostly while the refusal was
  in the memory shown.

**What remains outside the evidence:**
* most of what the reasons say: objectives, causal claims, timing claims,
  implicit intentions;
* the parameters of matched actions;
* all earlier attempts.

The counts are small and depend on how much each model says. They are not
a comparison of the models.

## 6. Known limitations

* **Calibration:** the rules were calibrated on this corpus, and their
  precision was reviewed by the author, not independently. No held-out
  recordings exist.
* **Narrow reach:** the patterns are narrow. Recall is traded for
  precision, so absence of findings is not evidence of consistency. A
  hedge anywhere in a sentence, or a revision anywhere in a reason, blocks
  a statement.
* **Only the final reply is checked:** only the final parsed reply's
  reason is recorded. Earlier attempts' reasons are outside the
  repository.
* **Observation checks compare with the prompt the model saw**, the
  agent's summary. A value the summary left out cannot be checked.
* **Repeat identity:** it uses the engine's message with dates
  normalized. A reworded refusal of the same payload counts as a
  different proposal.
* **Consequences** are next-day observations, never attributed. There is
  no counterfactual.
* **The corpus:** two models (three in V0.5c), one agent and prompt
  version, development seeds, English reasons.

## 7. Reproducing it

```
# from the repository root, after building dt
dt benchmark run docs/benchmark/v05a-dev/replay.json -out R/llm-replay-v05a
# … every replay in corpus.json, into R/<out>
python3 -I -B analysis/evidence/decision_evidence.py \
    --corpus docs/benchmark/v08a-dev/corpus.json --replays R --out E
cmp E/evidence.json docs/benchmark/v08a-dev/evidence.json
```

CI does this on linux/amd64, linux/arm64 and darwin/arm64, and compares
the evidence and the decision traces across them.
