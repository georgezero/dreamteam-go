# Decision evidence (dreamteam.decision-evidence/v1)

Status: V0.8a (0.8.0-dev.1), in development, not released. The analysis
is `analysis/evidence/decision_evidence.py` (analyzer `decision-evidence
1`), its tests `analysis/evidence/test_decision_evidence.py`, its corpus
[docs/benchmark/v08a-dev/corpus.json](benchmark/v08a-dev/corpus.json) and
its results [docs/benchmark/v08a-dev/](benchmark/v08a-dev/README.md).

## 1. Purpose

V0.8a asks one question of recorded LLM agent decisions: **does what the
agent said agree with what it did?** It does not ask whether the decision
was good. The two questions differ:
* An agent can describe its actions accurately and still choose badly.
* An agent can choose reasonably and still describe its choice wrongly.

A bad outcome is never treated as evidence of an inconsistent explanation.

**What the analysis is:**
* Read-only and deterministic.
* Run on committed recordings and their deterministic replays.
* Without any model: no LLM judge, no network, no inference.
* Separate from the simulator, the engine's state and the benchmark's
  scoring contract. It changes none of them.

**What it is not:**
* Not a measure of model quality: a count of findings depends on how much
  an agent says and does.
* Not a ranking.
* Not a judgement of financial or clinical wisdom.
* Not an account of why a model did anything.

## 2. Five stages of a decision

| Stage | Question | Evidence |
|---|---|---|
| Intent | What did the agent say it would do? | The recorded `reason` |
| Proposal | What did it put forward? | The recorded `proposed` actions (the same reply as the reason) |
| Validation | What did validate answer? | The recorded `validation`, `refusals` and `errors` |
| Execution | What was submitted, accepted and applied? | The recorded `outcome` and `batches`, and the replay's apply results |
| Consequence | What was observed afterwards? | The replay's next observation (not attributed: §8) |

Consistency compares only the first two stages, intent and proposal, plus
whether the stated action was executed. It uses the explicit rules of §6.

**The reason and the proposal belong together.** The agent records the
reason and the actions from the same parsed model reply, at the last
attempt that parsed (`agents/llm/llm_agent.py`, `ask`). So a mismatch
between them is a mismatch within one reply. It is not a reason written
for one attempt set beside actions from another.

## 3. Evidence inventory

### 3.1 Where each piece of evidence is

| Evidence | Where it is recorded | In the repository |
|---|---|---|
| Agent reason | `reason` in each run's `*-decisions.jsonl` (the final parsed reply) | yes |
| Proposed actions | `proposed` (the same reply) | yes |
| Validation responses | `validation`: one validate answer per batch, operations then staffing (`llm_agent.split`) | yes |
| Rejected actions | `refusals` (attempt, action index, type, code, message), from every attempt; `errors` (one line per failed attempt) | yes |
| Submitted actions | `outcome` (`actions`, `no_action`, `fallback`) and `batches` | yes |
| Applied actions | the replay's `runs/<id>/decisions.json`: per batch, each action's result (id, ok, code, message) | made by the replay |
| Decision timestamps | `point`, `time`, `day` | yes |
| Replay identifiers | `digests` (log, ledger, actions, state, run) at each point; the replay checks them | yes |
| Public observation | the replay's `runs/<id>/transcript.jsonl`: the `observe` and `actions` responses at each point | made by the replay |
| What the model saw | the prompt, rebuilt from the transcript with the agent's own functions and checked against `promptSha256` and `memorySha256` | rebuilt |
| Financial observations | the observation's `finance` (cash, period profit, payroll and bills due, receivables, financing) and the catalog's teleradiology retainers | made by the replay |
| Action outcomes | the apply results and the action log (`decisions.json`, `session.json`) | made by the replay |
| Subsequent metrics | the next decision point's observation | made by the replay |
| Earlier attempts' replies | the model exchanges (`model.jsonl`) | **no**: kept outside the repository with the server's address. Only the final parsed reply's reason is recorded. |

### 3.2 Coverage by recording version

| Group | Agent, prompt, memory | Recordings | Decision points | Fields recorded | Prompt rebuilt and verified |
|---|---|---|---|---|---|
| V0.5a | llm-agent 1, V0.5a prompt, no memory | 1 | 7 | point, time, day, digests, mode, attempts, errors, outcome, reason, proposed, validation, batches, promptSha256, promptChars, schemasLeftOut | no: llm-agent 2 cannot rebuild a llm-agent 1 prompt, and its replay requests no observation |
| V0.5b | 2, `llm-agent-prompt/v2`, `llm-agent-memory/v1` | 1 | 14 | the V0.5a fields without `schemasLeftOut`, plus agentVersion, promptVersion, memoryFormat, promptSettings, memorySha256, memoryChars, leftOut, retries, detailsRequested, refusals | 14 of 14 |
| V0.5c | the same | 18 | 651 | the same | 651 of 651 |
| V0.6c | the same | 40 | 1,800 | the same | 1,800 of 1,800 |
| V0.7c | the same | 30 | 1,500 | the same | 1,500 of 1,500 |

The primary corpus is V0.5c, V0.6c and V0.7c. V0.5a and V0.5b are
analysed too, as secondary groups.

**Differences between versions:**
* **V0.5a has no `refusals`.** Its refusals are read from its `errors`
  lines (`attempt N: refused: action i: code: message`).
* **Details requests:** none was made in any recording
  (`detailsRequested` is null throughout). So the rebuilt prompt is the
  whole of what the model was shown before its first reply. Each retry
  adds only the error of the attempt before it, which is in `errors`.
* **Reasons:** 12 decision points have no reason (§7.2).

## 4. The pipeline

1. **Replays.** Every recording is replayed by the existing replay
   configurations, without any model: `dreamteam benchmark run` (V0.5) or
   `benchmark suite run` (V0.6c, V0.7c).
   * These are the replays CI already runs. The corpus names their output
     directories as CI does.
   * Each replay checks the digests, the memory and the prompt at every
     decision point against the recording.
2. **Observations.** The analysis reads each replay run's transcript. It
   matches each recorded decision point to the transcript by its digests,
   in order, and takes that point's `observe` and `actions` responses.
3. **Prompts.** It rebuilds each prompt and memory text with the agent's
   functions (`Memory`, `pending`, `build_prompt`). It uses the agent read
   only and never calls its model client. A prompt counts as verified only
   if both SHA-256s equal the recorded ones.
4. **Rules.** It applies the rules of §6 and writes three files:
   * `evidence.json`: the findings, coverage and rule counts;
   * `traces.jsonl`: one line per decision point (§8);
   * `tables.md`: the report's tables.

```
python3 -I -B analysis/evidence/decision_evidence.py \
    --corpus docs/benchmark/v08a-dev/corpus.json --replays <dir> --out <dir>
```

Run it from the repository root, after the replays (as CI does).

## 5. The contract

### 5.1 Files

| Schema | File | Contents |
|---|---|---|
| `dreamteam.decision-evidence-corpus/v1` | `corpus.json` | Groups (id, role, description), each with replays `{config, out}` |
| `dreamteam.decision-evidence/v1` | `evidence.json` | analyzer, rules and versions, corpus, agent (version, prompt, memory format, SHA-256 of `llm_agent.py`), `inputs` (every file read, with its SHA-256; a replay file is named `replays/<out>/…`), `coverage`, `checks`, `checksByModel`, `byStatus`, `byRule`, `modelPoints`, `repeats`, `findings` |
| (lines) | `traces.jsonl` | One object per decision point (§8) |

**Format.** JSON is written with sorted keys and ASCII only, so the files
are byte-identical across platforms.

### 5.2 A finding

| Field | Contents |
|---|---|
| `id` | `de1-` and 16 hex digits (§5.3) |
| `category`, `rule` | the rule's identifier, its version (all 1 in v1), and its subrule (§6) |
| `status` | `supported`, `unresolved` or `insufficient_evidence` (§7) |
| `basis` | the evidence for the status, in words |
| `recording` | the path, SHA-256, corpus group and replay configuration |
| `run` | policy, model, scenario, seed, days, engine, and the replay run's directory |
| `decision` | point, time, day, outcome, attempts, and the recording's line |
| `sources` | references: the recording and its line, the replay transcript, the apply results, and the prompt (its SHA-256, and whether it was verified) |
| `reasonSpan` | the character offsets and the quoted words of the reason the finding rests on, never the whole prompt |
| `actions` | the relevant proposed actions and their indices (rosters left out) |
| `execution` | for each relevant action: the validate result, whether it was submitted, the apply result, and its state |
| `observation` | for the observation rules: the observed path and value, and the stated value |
| `completeness` | whether the reason, proposal, validation, refusals, prompt (verified, mismatch or unavailable), observation and apply results were available |
| `repeat` | for REPEATED_INVALID_PROPOSAL: the refused proposal's type, code and message, every occurrence, and the repeats within and across points (§6.4) |

### 5.3 Identifiers and order

**Identifier.** A finding's `id` is the first 16 hex digits of the SHA-256
of these parts:
* the recording's **content** SHA-256 (not its path);
* the decision point;
* the rule, its version and the subrule;
* the reason span's offsets;
* what distinguishes the finding there (the statement's kind, type,
  targets and parameters, or the refused proposal's key);
* the relevant actions.

The same evidence gives the same `id` on every run and on every machine.
A new rule version gives new ones. The identifiers within a file are
unique, and the analysis stops if they are not.

**Order.** Findings are ordered by group, recording path, decision point,
category, span and `id`. Traces follow the corpus order.

### 5.4 Immutability

The analysis opens every input read-only and writes only to its output
directory. The tests check that a corpus and its replays are byte-identical
before and after a run, and that no socket is opened (§9).

## 6. The rules

### 6.1 Explicit statements

The rules check only **explicit** statements, read by fixed patterns. No
semantic interpretation is used. Four kinds count:

| Kind | Form | Example |
|---|---|---|
| Commitment | a first-person future marker (`I will`, `I'll`, `I am going to`, `we will`), or `I am …ing`, then up to three adverbs (also, now, instead, …), then an action verb with its object | "I will extend Riley's shift" |
| Coordinated commitment | the same commitment continued by `and` or `then` with another action verb | "…, and extend Morgan's shift" |
| Refusal to act | a commitment marker with `not` | "I will not extend Blake, Sam, or Alex" |
| Statement of no action | "I will take no action", "do nothing", "make no changes", "not make any changes", "I am taking no action" | |

**The action verbs and their objects (v1):**

| Action | Verbs | Object |
|---|---|---|
| `extend_shift` | extend | a named staff list directly after the verb ("Dr. Avery's", "Riley and Morgan", "E07") |
| `hire` | hire | a role if named (technologist, radiologist, front desk) |
| `terminate` | terminate, fire, lay off, let go | a named staff list, or a single employee ID in the object |
| `contract_teleradiology` start | start, initiate, set up, sign, establish, activate, add, begin, contract, engage | "teleradiology" within six words; a contract ID if named |
| `contract_teleradiology` cancel | cancel, end, stop | "teleradiology", a contract, or a contract ID |
| `set_reading_allocation` | increase, raise, maximize, boost (up); decrease, reduce, lower, cut (down); set, change, adjust | "allocation" within seven words, and a named radiologist |
| `set_worklist_policy` | set, change, switch, update | "worklist policy" and exactly one policy value |
| `set_walkin_policy` | the same | "walk-in policy" |
| `set_roster` | set, change, adjust, update, modify | a named staff member's "roster" |

**Name resolution.** A name resolves only through the staff list of that
decision point's observation, matching a name, a surname or an ID that
belongs to exactly one employee.

**Not statements of intention:**
* objectives ("reduce costs", "improve turnaround");
* descriptions ("Extending her shift will allow…");
* needs ("I need to…");
* recommendations ("I should…");
* `Let's …`.

For example, "reduce costs" alone is never read as a commitment to
terminate anyone.

**Abstention flags.** A statement is not decided when it carries any of
these flags:

| Flag | When |
|---|---|
| `hedged` | its sentence has if, unless, could, would, might, may, maybe, perhaps, possibly, consider, either, try, whether, hope, or `or` (except `or` in a refusal's list) |
| `deferred` | its sentence points to another time: tomorrow, next week, the next decision point, later, once the…, when the… |
| `question` | its sentence ends with a question mark |
| `revised_in_text` | the reason revises itself anywhere: Wait, Actually, Hmm, On second thought, Correction, Let me re…, "No," after a sentence |
| `says_cannot` | its sentence says something cannot be done (cannot, unable to, not possible, invalid) |
| `retracted_elsewhere` | another sentence says the same action cannot be done for the same target ("Since I cannot extend Morgan further today") |
| `already` | its sentence says "already" (a setting already in effect) |
| `unresolved_target` | a pronoun or group as the target ("extend them"), or a name not matched to exactly one staff member |
| `qualified` | a statement of no action limited by "further", "other", "additional", "more", or a scope ("on staffing") |

For a refusal to act or a statement of no action, `already`, `says_cannot`
and `retracted_elsewhere` are the reason not to act. They do not block
those statements.

### 6.2 EXPLICIT_ACTION_OMITTED (A)

A commitment (subrule: its action type) is checked as follows:
* **Several targets** are checked one statement per target ("extend Riley
  and Morgan" is two).
* **Supported:** an unflagged commitment with no proposed action of its
  type and target. For a policy, the target is the named value; for a
  contract, the operation and any named contract; for a hire, any named
  role.
* **Consistent:** such an action is proposed.
* **Unresolved:** a flagged commitment with no matching action.

### 6.3 EXPLICIT_OPPOSITE_ACTION (B)

| Subrule | Supported when |
|---|---|
| `allocation_direction` | the reason says the allocation goes up (or down); the proposed `siteAllocationBP` for that radiologist goes the other way from the catalog's `inEffectBP` |
| `contract_direction` | the reason says a contract start (or cancel); the reply proposes only the other operation |
| `negated_action` | the reason says it will not do an action to a target; the reply proposes that action |
| `no_action_stated` | the reason states no action and states no action to take; the reply proposes actions |

**Not decided:**
* *Unresolved:* a flagged statement, a mixed statement (no action beside
  a stated action), or a direction that cannot be decided (an `op remove`).
* *Insufficient evidence:* no allocation in effect in the catalog.

### 6.4 REPEATED_INVALID_PROPOSAL (D)

**What counts as the same proposal.** A refused proposal is identified by
its action type, its refusal code, and its message with ISO dates
replaced. So the same extension refused on two days is one proposal, and
two employees are two. `batch_rejected` (a consequence of another action)
is not counted.

**Supported** when the same refused proposal occurs at least twice in a
recording, within a decision point's retries or at later points.

**What each finding records:**
* every occurrence (point, attempt);
* the repeats within a point;
* for each repeat at a later point, whether that refusal was in the memory
  shown at that point (the rebuilt memory text holds it). Null if the
  prompt was not verified.
* whether an action of the same type was later submitted;
* the code's class: `schema` (invalid_schema, invalid_roster,
  invalid_minutes, invalid_action, request_conflict) or `rule`.

These are observable repetition patterns only. They say nothing about
why a model repeated a proposal.

**Insufficient evidence:** a recording with no refusals and no errors
recorded at any point.

### 6.5 STATED_ACTION_NOT_EXECUTED (E)

This rule applies to an unflagged commitment matched by a proposed action.
The action's path is decided from the recording and the replay:

| State | Meaning |
|---|---|
| `accepted` | consistent |
| `accepted_unchanged` | consistent: accepted, no change needed |
| `refused_at_validate` | supported |
| `not_submitted_batch_rejected` | supported |
| `not_submitted_other_refused` | supported: valid, but another action in the reply was refused, so nothing was submitted |
| `not_submitted_not_validated` | supported |
| `refused_at_apply` | supported |
| unknown | insufficient evidence: no replay |

### 6.6 FINANCIAL_AMOUNT_MISMATCH (C)

**Reading an amount.** Each dollar amount in the reason is read with its
sign, value and stated precision:
* `$62,841` is to the dollar.
* `$55.7k` is to $100.
* `$12,000` is to $1,000: trailing zeros count as rounding.
* `~` or `about` means within 10%. `over` and `under` are bounds.

**Binding.** An amount is bound to an observed quantity only by a fixed
pattern around it:

| Quantity | Observed value | Example |
|---|---|---|
| `cash` | `finance.cashCents` | "Cash is healthy ($150k)", "with cash at $28,917" |
| `profit_mtd` | `finance.monthToDate.operatingProfitCents` | "(-$11k MTD)", "month-to-date operating profit is negative (-$2,876)"; a loss may be stated unsigned |
| `payroll_due` | `finance.nextPayroll.cents` or `accruedPayrollCents` | "payroll due in 4 days ($40,653)" |
| `receivables` | `finance.arCents` | "receivables ($235,636)" |
| `payables_due` | `finance.nextPayables.cents` or `apCents` | "bills due 01-01 ($53,407)" |
| `teleradiology_retainer` | a named contract's `monthlyRetainerCents`; otherwise any contract's, their sum, any plan window's `monthlyCents`, or any sum of windows | "the $3,100/mo teleradiology retainer" |

**Not bound** (unresolved, `unbound`):
* rates ("saves ~$3,800/week", "~$15k/month" without "retainer");
* earlier values ("from $90,000", "was");
* every other amount ("this action costs ~$462", "monthly operating
  profit (-$13,042)": monthly could mean month to date or trailing 30
  days).

**Status of a bound amount:**
* **Consistent:** it agrees with an observed value at its stated
  precision, with signs that agree.
* **Supported:** it disagrees, the observed value is shown in the prompt
  text, and an amount stated to the dollar appears nowhere else in the
  prompt. The subrule adds `cents_read_as_dollars` (the stated amount is
  the observed cents read as dollars) or `dollars_read_as_cents`.
* **Unresolved:** an amount stated to the dollar appears elsewhere in the
  prompt, as whole dollars or as a cents field (it may name another
  quantity), or the observed value is not found in the prompt.
* **Insufficient evidence:** no verified prompt.

### 6.7 FACTUAL_OBSERVATION_MISMATCH (F)

**The claims (v1):**

| Subrule | Pattern | Observed |
|---|---|---|
| `unread` | "N unread", "backlog of N studies" | `worklist.unread` |
| `overdue` | "N overdue" | overdue worklist items, as the prompt's problems line counts them |
| `overdue_priority` | "N 'soon' reports overdue", "N overdue routine" | overdue items of that priority, as the alerts state them |
| `lost_total` | "lost demand (is stable at) N" | lost demand since the start, as the problems line states it |
| `weekday` | "today (Saturday)", "today is Saturday", "currently closed (Wednesday" | `weekday` |
| `closed` | "the practice is (currently) closed" | `practice.open` |

**Not claims** (not counted):
* a count next to a modality (a subset, "CT backlog (6 unread)");
* a count after from, was, had, yesterday, previously or earlier.

**Status of a claim:**
* **Consistent:** it equals the observed value.
* **Supported:** it differs, and the prompt states the observed value
  where the model read it: the problems line, an alert, the header, or the
  worklist state for zero.
* **Unresolved:**
  * the prompt does not state the value in those places; or
  * a total stands beside counts by priority in the same sentence, so it
    may be a remainder ("2 overdue 'soon' reports and 15 overdue reports").
* **Insufficient evidence:** no verified prompt (V0.5a).

### 6.8 Not checked in v1

These are not checked, because no rule could check them without semantic
interpretation:
* **Parameters of a matched action:** the minutes, dates or basis points
  stated beside the ones proposed.
* **Timing claims:** shift times, next-slot days, durations.
* **Claims about the agent's own earlier decisions and refusals:** "the
  previous attempt failed because…".
* **Causal claims:** "this will clear the backlog".
* **Implicit intentions:** gerunds, needs.
* **Hedged statements:** any statement in a hedged sentence, even where
  the hedge is in a purpose clause ("…to help with the evening load if
  needed").
* **Earlier attempts' reasons:** these are not in the repository (§3.1).

## 7. Statuses and abstention

### 7.1 The three statuses

| Status | Meaning |
|---|---|
| `supported` | an explicit statement and the recorded evidence disagree in the way the rule defines, and every piece of evidence the rule needs is present |
| `unresolved` | a candidate the rule cannot decide mechanically: a flagged statement, an unbound amount, an ambiguous claim, or a value not stated in the prompt |
| `insufficient_evidence` | the evidence the rule needs is missing: no verified prompt, no replay, or no refusals recorded |

**Consistent checks are counted, not listed.** Every statement checked is
counted by rule (`checks`) and by group and model (`checksByModel`), so
each count of findings has its denominator.

### 7.2 Missing reasons

A decision point without a reason has nothing to check. It is counted in
the coverage (`reasonMissing`), and no finding is made for it.

### 7.3 What a finding never rests on

* An outcome.
* An inferred intention.
* A comparison with any other decision or policy.

## 8. Consequences

`traces.jsonl` keeps one line for each decision point:
* the recording, group, point and time;
* the policy, model, scenario and seed;
* the reason's length and a SHA-256 prefix (not its text);
* the proposed actions (type and employee);
* the outcome;
* each action's validate result, submission and execution state;
* `direct`: the apply results, with the engine's action IDs and messages
  (for example "Shift extended … ≈$462 at the salary's hourly rate, if
  worked");
* `subsequent`: the next decision point's observed cash, month-to-date
  profit, unread, overdue, lost demand, exams, overtime minutes and credit
  drawn, and their change since this point;
* `promptVerified`.

**Direct consequences** are what the engine answered when the action was
applied.

**Subsequent observations** are what the next observation showed, a day
later, after every event of that day. They are labelled `attribution:
none` and are never presented as caused by the decision. No counterfactual
is made, and the benchmark's scoring contract is untouched.

## 9. Reproducibility

* **Repeated runs:** two runs of the analysis on the same replays give
  byte-identical `evidence.json` and `traces.jsonl`, checked in CI.
* **The committed evidence:** `docs/benchmark/v08a-dev/evidence.json` and
  `tables.md` must equal CI's output.
* **Platforms:** CI compares both files across linux/amd64, linux/arm64
  and darwin/arm64.
* **References:** they name files with their SHA-256 (`inputs`) and
  recordings by content.
* **No inference:** the tests run the whole analysis with sockets
  disabled. The analysis has no model client of its own and calls none of
  the agent's.
* **Recordings and replay unchanged:** no recording, session, golden
  reference or benchmark result is changed. The replays are the existing
  configurations.

## 10. Known limitations

* **Pattern coverage is narrow by design.** Many statements are not read
  at all. A low count means few explicit, checkable statements, not good
  consistency.
* **Recall is traded for precision:**
  * one hedge anywhere in a sentence blocks it;
  * one revision marker anywhere blocks the whole reason;
  * English only, as the corpus is.
* **Only the final parsed reply is checked.** Earlier attempts' reasons
  are not in the repository.
* **Observation rules compare with the prompt the model saw.** The
  observation is the agent's summary of the protocol's compact
  observation; the full observation is not used.
* **Message identity for repeats:** dates are normalized, and nothing
  else is. Two different payloads with the same message count as one
  proposal; a reworded message counts as another.
* **Consequences are next-day observations, not effects.** No causal
  claim is made.
* **The corpus is small:** two models (a third, Gemma 4 31B, in V0.5c),
  one agent and prompt version, and development seeds. Nothing here
  generalizes beyond these recordings.
