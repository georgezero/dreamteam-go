# Financial and operational outcomes, paired with no action (V0.8b)

**What this is.** A descriptive analysis of the existing V0.6c and V0.7c
evaluations (`dreamteam.financial-outcomes/v1`,
`analysis/evidence/financial_outcomes.py`). No run was made for it.

**Files:**
* [financial-outcomes.md](financial-outcomes.md): every pair, all 140.
* [financial-outcomes.json](financial-outcomes.json): the data.

**How it pairs.** Every policy's run is paired with the no-action run on
the same entry, seed, horizon and economics, from the committed contract
reports.

**What the replays add.** The replays exist for the LLM runs and their
no-action pairs, and they supply what the contract report lacks:
* overtime cost;
* the most credit drawn and the interest;
* the actions executed, by type.

**What it keeps apart:**
1. financial outcomes;
2. operational outcomes;
3. stated reasons: the explicit commitments the frozen v1 rules read, by
   action type;
4. submitted decisions;
5. executed decisions.

**What it does not do:**
* **Combine economics:** V0.6c (dreamteam-dev 1: overtime not costed, no
  credit line) and V0.7c (dreamteam-dev 3: radiologist overtime paid, a
  $50,000 credit line) are reported separately and never combined.
* **Score, rank or explain:** there is no composite score, no ranking, no
  counterfactual and no causal claim. A difference from no action
  describes two runs over the whole horizon, after every event in them.
  It is not attributed to any one decision.
* **Judge:** an unfavourable difference is not a judgement that a
  decision was irrational.

## Ranges of the differences from no action, by suite run and policy

Each cell is the lowest and the highest difference over the pairs (entries
and seeds) of that suite run, rounded to the nearest dollar.

### V0.6c, 30 days (dreamteam-dev 1; 15 pairs per policy)

| Policy | Profit | Revenue | Expense | Lowest cash | Exams | Lost orders | TAT breaches | TAT p90 (min) |
|---|---|---|---|---|---|---|---|---|
| greedy | −$1,636 to +$694 | +$58 to +$1,926 | +$1,233 to +$1,890 | 0 | 0 | 0 | −27 to 0 | −207 to 0 |
| conservative-v2 | −$567 to +$662 | $0 to +$1,237 | $0 to +$822 | 0 | 0 | 0 | −3 to 0 | −12 to 0 |
| Qwen3.6-35B-A3B | −$30,447 to +$1,070 | $0 to +$8,797 | $0 to +$37,361 | −$9,142 to +$247 | 0 to +51 | −23 to 0 | −80 to +25 | −656 to +108 |
| Gemma 4 26B-A4B | −$3,074 to +$1,169 | $0 to +$3,676 | $0 to +$5,067 | −$843 to 0 | 0 to +24 | −16 to 0 | −41 to +3 | −577 to +10 |

### V0.6c, 90 days (dreamteam-dev 1; 5 pairs per policy)

| Policy | Profit | Revenue | Expense | Lowest cash | Exams | Lost orders | TAT breaches | TAT p90 (min) |
|---|---|---|---|---|---|---|---|---|
| greedy | −$33,940 to −$27,266 | −$40 to +$6,729 | +$32,554 to +$33,995 | −$26,709 to 0 | 0 | 0 | −267 to −42 | −424 to −97 |
| conservative-v2 | −$2,301 to +$2,191 | $0 to +$2,766 | $0 to +$2,339 | 0 | 0 | 0 | −53 to 0 | −118 to 0 |
| Qwen3.6-35B-A3B | −$137,507 to +$1,284 | −$63,378 to +$22,769 | −$19,444 to +$160,276 | −$134,540 to −$725 | −298 to +187 | −113 to +361 | −206 to +11 | −833 to +16 |
| Gemma 4 26B-A4B | $0 to +$2,098 | $0 to +$7,962 | $0 to +$5,864 | 0 | 0 | 0 | −97 to 0 | −156 to 0 |

### V0.7c stage A, 30 days (dreamteam-dev 3; 10 pairs per policy)

| Policy | Profit | Revenue | Expense | Lowest cash | Overtime cost | Exams | Lost orders | TAT breaches |
|---|---|---|---|---|---|---|---|---|
| greedy | −$1,387 to −$30 | +$157 to +$1,202 | +$1,233 to +$1,890 | 0 | (not replayed) | 0 | 0 | −25 to −1 |
| conservative-v2 | 0 | 0 | 0 | 0 | (not replayed) | 0 | 0 | 0 |
| Qwen3.6-35B-A3B | −$20,304 to −$3,197 | $0 to +$3,414 | +$4,200 to +$23,363 | −$10,096 to −$920 | $0 to +$12,923 | 0 to +11 | −10 to 0 | −61 to +1 |
| Gemma 4 26B-A4B | −$2,736 to 0 | −$318 to +$1,881 | $0 to +$3,925 | 0 | 0 | 0 | 0 | −39 to 0 |

No run drew on its credit line in stage A.

### V0.7c stage B, 90 days (dreamteam-dev 3; 5 pairs per policy)

| Policy | Profit | Revenue | Expense | Lowest cash | Overtime cost | Most credit drawn | Exams | Lost orders | TAT breaches |
|---|---|---|---|---|---|---|---|---|---|
| greedy | −$33,550 to −$29,022 | −$52 to +$4,891 | +$32,570 to +$33,957 | −$18,863 to 0 | (not replayed) | (not replayed) | 0 | 0 | −317 to −34 |
| conservative-v2 | −$2,712 to 0 | 0 | $0 to +$2,712 | 0 | (not replayed) | (not replayed) | 0 | 0 | −12 to 0 |
| Qwen3.6-35B-A3B | −$181,589 to −$8,007 | −$138,515 to +$6,768 | +$8,952 to +$53,357 | −$37,228 to −$6,149 | −$2,218 to +$10,477 | $0 to $17,200 | −867 to +11 | −8 to +1,450 | −304 to −32 |
| Gemma 4 26B-A4B | −$22,096 to 0 | $0 to +$12,668 | $0 to +$23,685 | −$6,627 to 0 | $0 to +$4,617 | $0 | 0 to +54 | −53 to 0 | −340 to 0 |

**Where the extremes come from.** They are single pairs. The V0.7c
staffing-shortage Qwen run (seed 41701) holds:
* profit −$181,589;
* revenue −$138,515;
* exams −867;
* lost orders +1,450;
* the only credit drawn, $17,200 (interest $14).

V0.7.0 reported it as one development observation.

**V0.6 and V0.7 values must not be read against each other.** For
example, overtime cost exists only under V0.7's paid-overtime economics,
and so does credit.

## Stated, submitted and executed decisions (LLM runs)

These are totals over each suite run's pairs. Each column counts
something different:
* **Stated:** explicit commitments by action type, as the frozen v1 reader
  finds them ("I will extend …"), counting hedged ones.
* **Submitted:** actions in submitted batches.
* **Executed:** actions the engine accepted with a change. An action
  accepted with nothing to change, such as a policy already in effect or
  an employee already leaving, is not counted.

| Suite run | Model | Stated | Submitted | Executed |
|---|---|---|---|---|
| V0.6c 30 d | Qwen | extend 74, teleradiology 9, worklist 5, hire 7, allocation 2, walk-in 1 | extend 155, teleradiology 12, allocation 9, worklist 4, hire 5, roster 3, walk-in 1 | extend 155, teleradiology 11, allocation 7, worklist 4, hire 5, roster 1, walk-in 1 |
| V0.6c 30 d | Gemma 26B | extend 20, teleradiology 7, hire 1 | extend 24, teleradiology 8 | extend 24, teleradiology 8 |
| V0.6c 90 d | Qwen | extend 48, teleradiology 8, worklist 5, allocation 4, hire 1 | extend 156, teleradiology 8, worklist 7, roster 6, allocation 5, terminate 3, hire 3 | extend 156, teleradiology 7, roster 3, worklist 3, allocation 4, terminate 3, hire 3 |
| V0.6c 90 d | Gemma 26B | teleradiology 3 | teleradiology 1 | teleradiology 1 |
| V0.7c A | Qwen | extend 42, teleradiology 7, allocation 5, worklist 5, hire 2 | extend 85, teleradiology 8, allocation 7, worklist 6, hire 2 | extend 85, teleradiology 8, allocation 6, worklist 6, hire 2 |
| V0.7c A | Gemma 26B | teleradiology 6 | teleradiology 6 | teleradiology 6 |
| V0.7c B | Qwen | extend 32, worklist 8, allocation 5, teleradiology 2, walk-in 2, terminate 2, hire 1 | extend 67, terminate 16, worklist 8, allocation 5, hire 5, teleradiology 3, walk-in 2 | extend 67, terminate 7, hire 5, worklist 4, allocation 4, teleradiology 3, walk-in 1 |
| V0.7c B | Gemma 26B | extend 68, teleradiology 8, hire 1 | extend 80, teleradiology 7 | extend 80, teleradiology 7 |

**What this shows:**
* **Most submitted actions were not explicitly stated.** Qwen submitted
  two to three times as many extensions as it stated in "I will …" form. The rest
  were described ("Extending Dr. Avery's shift will…") or not mentioned.
  This is a limit of what the stated-intent rules can check, not a
  finding of inconsistency.
* **Submitted and executed differ only where the engine had nothing to
  change:** 9 of Qwen's 16 terminations in stage B were of employees
  already leaving, and 4 of its 8 worklist changes set the policy in
  effect. No submitted action of any policy was refused when applied (the
  contract reports' refusal counts are 0 throughout).

## What this analysis does not show

* **Causes:** that any action caused a difference. Each difference spans
  the whole horizon, after every event.
* **Alternatives:** what any other decision would have produced. The only
  comparison is with no action.
* **Quality:** that a policy is better or worse in general. These are
  development seeds and five scenarios, with no significance test.
* **Clinical value:** turnaround and breaches are simulated service
  measures.
