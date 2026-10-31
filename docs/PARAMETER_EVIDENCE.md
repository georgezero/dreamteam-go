# Parameter evidence (dreamteam.parameter-evidence/v1)

Status: built in V0.9a (0.9.0-dev.1). Not released. Not independently
validated. A new, separately versioned contract: it changes nothing in
`dreamteam.decision-evidence/v1` (frozen), `dreamteam.repeat-evidence/v2`
or the refusal-response extensions.

**Files:**
* the analysis: `analysis/evidence/parameter_evidence.py` (analyzer
  `parameter-evidence 1`);
* its tests: `analysis/evidence/test_parameter_evidence.py`;
* its corpus:
  [docs/benchmark/v09a-dev/corpus.json](benchmark/v09a-dev/corpus.json);
* its results: [docs/benchmark/v09a-dev/](benchmark/v09a-dev/README.md).

## 1. The question

When an agent's reason **states a parameter explicitly**, does it agree
with the parameter of the action submitted in the **same reply**?
* "Extend Morgan by 2 hours", and the reply submits Casey's extension.
* "Reduce Dr. Blake's allocation to 30% from Nov 19", and the reply
  submits 40% from Nov 20.

**What it is:**
* specific, source-referenced mismatches between a stated value and a
  submitted value;
* read-only and deterministic, on committed recordings and their
  deterministic replays;
* run with no model, no LLM judge and no network.

**What it is not:**
* not an interpretation of what an agent meant;
* not a judgement of whether the decision was good;
* not a reading of outcomes;
* not a score.

Abstaining is preferred to a false finding.

## 2. Inputs

| Input | Source |
|---|---|
| The reply's reason and actions | llm-agent 3 (`dreamteam.llm-decision-record/v3`): every parsed attempt that is not a details request. llm-agent 2: the final parsed reply only; its `reason` and `proposed` come from the same reply (`llm_agent.py`) |
| Each action's validation | llm-agent 3: the attempt's validate answers, by recorded index. llm-agent 2: the answers in batch order, mapped back with the agent's own split (operations, then staffing) |
| Staff names and IDs | the observation the agent received, from the replay transcript (`staff[].id`, `staff[].name`) |
| The allocation in effect | the action catalog the agent received (`set_reading_allocation` targets: `inEffectBP`, pending versions) |
| The decision point's date | the record's `time` |

The frozen v1 analyzer's corpus, replay and transcript readers are used
read-only. Nothing is written to any input.

**Not checkable from older recordings.** llm-agent 2 (V0.5c–V0.7c) did
not record an earlier attempt's reason or actions. A refused proposal is
therefore checked only when it was the final parsed reply, and nothing
said in an earlier attempt can be checked. llm-agent 1 (V0.5a) is outside
the corpus.

## 3. The rules

**A statement** is an **action phrase**, in one of the reason's sentences
(split as the frozen analyzer splits them).

* **Its anchor** is an action verb that either:
  * follows a stated intention ("I will", "I'll", "we will", "I am going
    to", "I plan to", "I'm …ing"); or
  * opens the sentence ("Extending Dr. Avery's shift …", "Reduce …",
    "Also, set …").
* **It ends** at the first subordinate break: so, because, since, as,
  which, while, before, after, once, but, given, due to, in order to, "to
  allow / help / ensure / cover / …", helping, ensuring, a dash,
  "currently". A verb that opens the sentence is its subject, and the
  phrase also ends at the main verb ("Extending her shift by 2 hours
  **will** allow …").
* **Only what is inside the phrase is read:** the action type, the named
  staff and the stated values. Descriptions ("Dr. Blake's allocation is
  currently 60%"), capabilities ("Dr. Avery can extend up to 19:30") and
  names in another role ("… before Dr. Blake arrives") are not
  statements.
* **One action type per phrase.** A phrase after a semicolon that refers
  back with "this" or "it" takes the previous sentence's type
  ("…allocated 30%; increasing this to 50%").

The action types:

| Type | Named by |
|---|---|
| `extend_shift` | extend, extension |
| `terminate` | terminate, let … go, give notice, lay off, dismiss |
| `hire` | hire, recruit |
| `set_reading_allocation` | allocation, allocated |
| `contract_teleradiology` | "teleradiology … contract" (either order), or a contract ID |
| `set_roster` | roster, rota |

**A statement is not read** (it is counted as excluded, by reason) when
**its whole sentence** is:
* **hedged:** if, unless, could, would, might, may, maybe, perhaps,
  possibly, consider, either, whether, option, alternatively, try, hope,
  or;
* **negated or rejected:** not, never, cannot, no longer, instead of,
  rather than, invalid, unable, without, blocked, infeasible,
  unnecessary, premature, impossible, pointless, ineffective, wasteful,
  counterproductive, disallowed, prohibited, unavailable, redundant,
  too, n't. So "Extending Dr. Blake's shift **is not feasible**" is not
  read;
* **about the past:** was, were, had, has/have been, already, previous,
  previously, yesterday, earlier, failed, attempted, last time;
* **about several action types** at once;
* **in a reason that revises itself** (Wait, Actually, …; the frozen
  analyzer's marker): the whole reason is not read.

A statement with no submitted action of its type is not a parameter
check. That is the frozen v1 rules' question (an omission).

**Binding.** A stated value binds to the submitted actions of the type:
* **the named employee's action,** when the statement names one;
* **each named employee's action,** when it names several and each has
  exactly one ("extend Riley and Casey by 2 hours each");
* **each action of the type,** when "both" or "all three" (and so on)
  counts them exactly;
* **otherwise the only action of the type** in the reply.

Anything else is `unresolved`, with the reason: several possible actions,
several staff not each with one action, a name that fits several
employees, or the named employee is not the submitted action's.

**A target is `supported` only** when:
* the statement names one employee;
* the reply has exactly one action of the type, for someone else;
* and no other statement in the reply names that someone else.

If another statement names them ("I will extend Dr. Avery's shift by 120
minutes. I will also extend Dr. Blake's shift."), the statement about the
other employee concerns an action that was not submitted. That is an
omission, and is `unresolved` here.

| Rule | Stated | Submitted parameter | Notes |
|---|---|---|---|
| `AMOUNT_MISMATCH` | a dollar amount with an hourly marker, in a hire statement | `hire.rateCents` | other amounts (annual cost, savings) are not bound |
| | a percentage or basis points after "to", "at", "of" or "=", in an allocation statement | `set_reading_allocation.siteAllocationBP` | "by 20%" is `unresolved`: percentage points and relative change are not told apart |
| `TARGET_MISMATCH` | one staff name or ID | the action's `employee` | `supported` only when the reply has exactly one action of the type, and it is for someone else |
| | one contract ID (T00001) | `contract_teleradiology.contract` | |
| `DATE_MISMATCH` | a date bound by on, from, starting, effective, as of, until, through, ending, for, date of, last day, start date; a weekday so bound or possessive ("Monday's"); today, tomorrow | `extend_shift.date`, `terminate.lastDay`, `hire.startDate`, `set_reading_allocation.effective`, `set_roster.effective`, `contract_teleradiology.effective` or `.endDate` | a contract's end date is read only after until, through or ending, or for a cancellation |
| `DURATION_MISMATCH` | a length bound by by, for, extra, additional, another or of, or followed by extension, longer or more | `extend_shift.minutes` | clock times ("to 18:00") are not durations |
| `DIRECTION_MISMATCH` | increase, raise, boost, maximize, restore / decrease, reduce, lower, cut, minimize, governing an allocation | `siteAllocationBP` against `inEffectBP` | `unresolved` if a change is pending, or the value equals the one in effect |

**Never inferred:**
* **A parameter the action omits** (an `extend_shift` with no `date`):
  the check is `unresolved`, and the default is not assumed.
* **A value the statement qualifies** (at least, at most, up to, about,
  no later than, by a date, or more): `unresolved`.
* **A statement giving several different values** (2 hours for one
  person, 4 for another; Nov 19 and Monday the 23rd): `unresolved`.

## 4. Statuses

| Status | Meaning |
|---|---|
| `consistent` | the stated value and the bound action's value agree exactly |
| `supported` | they differ: the finding's evidence is the reason span, the action reference and both values |
| `unresolved` | the statement was read but cannot be bound or compared; the basis says why |
| `insufficient_evidence` | an input is missing: the observation (staff list), the catalog (allocation in effect), or the decision point's date |

The **eligible checks** are all four statuses together.

## 5. Normalization

Each check names its rule (`normalization`).

| Rule | Handling |
|---|---|
| `N-MONEY-HOURLY` | "$44/hour", "$45.50 per hour", "$4,400 an hour": separators removed, as exact integer cents (Decimal arithmetic). Compared exactly: **no one-unit tolerance.** A stated figure equal to the submitted cents is flagged as cents read as dollars |
| `N-PERCENT` | "30%", "32.5%" as exact basis points (3000, 3250); compared exactly |
| `N-BP` | "3000 BP" as basis points; compared exactly |
| `N-DATE-ISO` | "2026-11-19" |
| `N-DATE-MONTHDAY` | "Nov 19", "November 19th", "19 Nov", with a year if given; otherwise the decision point's year, or the next if that date would be more than 60 days before the decision point |
| `N-WEEKDAY` | "Monday" compared with the weekday of the submitted date. Which week is not decided; a date and a weekday in one statement must agree with each other |
| `N-RELATIVE` | today and tomorrow, from the record's own date. "Next week", "in 3 days", "this weekend": `unresolved` |
| `N-DURATION` | "2 hours", "90 minutes", "1.5 h", "an hour", "half an hour" as exact whole minutes |
| `N-STAFF` | a whole-word name or ID matched against the observation's staff: the ID, the full name, the name without "Dr.", and each word of it. A name that fits more than one employee is not resolved ("Avery" with Dr. Avery and Avery Jones) |
| `N-CONTRACT` | a T-number, exactly |
| `N-DIRECTION` | the verb governs an allocation when "allocation" (or "it", "this") follows within six words, or a possessive name and "allocation" |

## 6. The contract

`parameter-evidence.json`:
* the schema, analyzer and rule versions;
* the normalization rules;
* the corpus and every committed input file with its SHA-256: the
  recordings, usage files, replay configurations and the corpus. The
  replay outputs (`run.json`, `transcript.jsonl`) are read but not
  listed. They are derived, they are checked by benchmark verify and
  reproduce, and they carry the engine build. Leaving them out means the
  output contains no build number and needs no normalization;
* coverage, by group and recording format: recordings, points, replies
  read, missing reasons, revised reasons, statements, exclusions by
  reason, statements with no action of the type, and checks by rule and
  status;
* checks by action type;
* `byRule`: eligible, consistent, supported, unresolved and insufficient
  evidence;
* `checks`: every check, in a fixed order (recording, point, attempt,
  position in the reason, rule, action, identifier).

**Each check:**

| Field | Contents |
|---|---|
| `id` | stable: `pe1-` and 16 hex of the SHA-256 of the recording's content, point, attempt, rule and version, span, action index and parameter |
| `rule`, `status`, `basis` | the rule and version, the status, and why |
| `recording` | path, SHA-256, group, format (llm-agent 2 or 3) |
| `run` | policy, model, scenario, seed, replay run |
| `decision` | point, time, recording line |
| `attempt` | the attempt ID (`point.attempt`) and number, whether it was used, and level: `attempt` (llm-agent 3) or `final parsed reply` (llm-agent 2, no ID) |
| `reasonSpan` | the stated value's span and text |
| `statement` | the statement's span and text |
| `action` | reference (`point.attempt#index`), index, type, and its validation (ok, code) |
| `parameter` | the action parameter |
| `stated`, `submitted` | the text and normalized value of each |
| `normalization` | the rule(s) applied |
| `completeness` | attempt level, validation recorded, observation and catalog available |

`parameter-tables.md` summarizes it: the counts by rule, coverage,
checks by action type, every supported mismatch, and the abstentions by
reason.

## 7. Reproducibility

* Two runs on the same replays give byte-identical outputs. CI runs it,
  compares it with the committed output and across the three platforms.
* No input is changed.
* The tests open no socket, and the analysis has no model client.

## 8. Known limitations

* **Narrow by design.** Most reasons state no exact parameter, and many
  statements are excluded by a single hedge word. A low count means few
  checkable statements, not good consistency.
* **Calibrated on this corpus,** by the developer, and **not
  independently reviewed.** Its findings carry no precision or recall.
  The developer read every `supported` check of each draft:
  * the first draft had 13, all false positives (descriptions,
    subordinate names and dates, a capability);
  * the second draft, reading action phrases only, had 7, all false
    positives (gerund subjects rejected in the predicate, and one hedged
    omission).

  The final rules have **no** supported mismatch in the corpus. They have
  found mismatches only in the synthetic tests, never in a recording.
  With no held-out corpus, the absence of findings shows that no
  explicit, bound parameter disagreed with its action. It does not show
  that the rules would find one.
* **llm-agent 2 records only the final parsed reply.** Older recordings
  support far fewer checks than llm-agent 3's.
* **English only;** dates in slash form (11/19) are not read.
* **A weekday is compared as a weekday,** not as a specific date.
* **Direction covers allocations only.** Hiring against terminating,
  starting against cancelling, are the frozen v1 opposite-action rule's.
* **A name used for another reason** ("to cover for Casey") can be read as
  a target. When several staff are named, such a name is `unresolved`,
  never `supported`.
* **No consequence and no cause** is read or claimed.
