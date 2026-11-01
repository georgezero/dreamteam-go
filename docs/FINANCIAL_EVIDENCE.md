# Financial decision evidence (`dreamteam.financial-evidence/v1`)

V0.9b. A deterministic, read-only account of what each recorded
management decision committed the practice to, how far the simulator lets
it be undone, and which simulator-controlled resources it changed
directly. Three dimensions, kept apart. **No composite score, ranking or
quality metric**, and no cause is attributed to a single decision.

* Analyzer: `analysis/evidence/financial_evidence.py` (`financial-evidence 1`).
* Tests: `analysis/evidence/test_financial_evidence.py`.
* Corpus: `docs/benchmark/v09b-dev/corpus.json`
  (`dreamteam.financial-evidence-corpus/v1`).
* Outputs: `docs/benchmark/v09b-dev/` — `financial-evidence.json`,
  `financial-tables.md`, `case-studies.md`; the reversibility matrix in
  `REVERSIBILITY.md`; the milestone report in `README.md`.

```
python3 -I -B analysis/evidence/financial_evidence.py --binary ./dt \
    --corpus docs/benchmark/v09b-dev/corpus.json --replays <dir> --out <dir> [--jobs N]
```

`--replays` is the directory of the recorded-decision replays CI makes
(the V0.9a corpus's replays). `--jobs` runs engine replays in parallel and
does not change the output.

## 1. Where every number comes from

The analysis does not model money. Every value is the engine's own:

1. **The decision point's observation**, from the replay's protocol
   transcript: cash, receivables, and under the V0.7b funding rule the
   current liabilities, reserved commitments, funding headroom and credit
   (`finance.financing.solvency`). This is what the agent was shown.
2. **The engine replay.** The replay's own transcript is fed back to the
   same engine (`dreamteam agent --stdio` on the run's scenario and seed)
   with read-only requests inserted:
   * a `preview` before each staffing batch's `validate`, and before each
     staffing `apply` that no `validate` preceded (the llm-agent 2 replays
     apply the recorded batches directly);
   * an `observe` after each applied batch;
   * for an llm-agent 2 final reply that was refused (and so not in the
     transcript), a `validate`, and a `preview` if staffing, at its
     decision point. The fresh answer is compared with the one the
     recording holds (`source.recordedValidation`).

   Every original answer must come back byte-identical, or the analysis
   stops (`ReplayMismatch`). So the inserted requests change nothing, and
   the previews describe exactly the state the agent decided in.
3. **The preview** is the game's staff-screen preview: the funding test
   (`commitmentCents`, `headroomCents`, `funded`), the guaranteed and
   forecast money lines with their timing and basis, and the hires,
   terminations, roster and allocation changes, teleradiology, slots and
   displaced bookings of the batch.

The agent could have asked for the same preview at the same moment
(`preview` is a permitted operation). Neither llm-agent 2 nor llm-agent 3
calls it. A preview is therefore evidence of **what was knowable** at the
decision, not of what the agent read.

The hashed inputs are committed files only: the corpus, the replay
configurations, the recordings, their usage files and the contract
reports. The replay's own files are read but not hashed, because they
embed the engine build. The output names no build.

## 2. The record

Each record is one dimension of one submission or executed action:

| Field | Meaning |
|---|---|
| `id` | `fe1-` + 16 hex: SHA-256 of recording path, decision point, event index, action indices, dimension and rule version |
| `dimension`, `rule` | `AFFORDABILITY`, `REVERSIBILITY` or `OPERATIONAL_EXPOSURE`, version 1 |
| `status` | one of the dimension's statuses (§3–§5) |
| `recording` | path and group; the run list carries its SHA-256 and format |
| `run` | scenario and seed; the run list carries the policy, model, horizon, economics, replay directory and the engine-replay check |
| `decision` | decision point index, recorded point and time |
| `attempt` | the llm-agent 3 attempt ID; `final` for an llm-agent 2 final reply; `null` when not linked |
| `submission` | the submission's number at its point |
| `actions` | index, the index in the reply (`attemptIndex`), type, op, target, date, and the validation code |
| `validation`, `execution` | batch validity and codes; `applied`, `not_applied` or `refused` |
| `measurements` / `facts` / `directChange` | the dimension's engine values |
| `completeness` | attempt-level recording, attempt linked, observation, catalog, preview |

Records are ordered by group, recording path, decision point, submission,
dimension and action index. Identifiers are checked unique.

## 3. Affordability (one record per submitted batch)

The engine's funding test is per batch (V0.7_SOLVENCY_DESIGN §3), so
affordability is too:

    funding headroom = cash + undrawn credit − current liabilities − reserved commitments

Receivables are never counted, and neither is profit. The headroom
without undrawn credit is `funding headroom − credit available`.

| Status | When |
|---|---|
| `funded_without_credit` | the engine accepted the batch and its commitment is at most the headroom without undrawn credit ("affordable when proposed") |
| `funded_with_credit` | the engine accepted it and its commitment exceeds the headroom without undrawn credit: it is funded only by the credit line ("accepted under available credit") |
| `refused_insufficient_funds` | the engine refused it with `insufficient_funds` |
| `no_new_commitment` | the funding test measures no commitment (a termination, a decrease, a roster change), or an operations batch with no paid extension |
| `not_tested` | refused for another reason before the funding test (the codes are listed); no affordability statement is made |
| `evidence_unavailable` | the economics has no funding rule (V0.6c), the observed state is not the state tested, or an extension batch was never applied (below) |

**The commitment.**
* **Staffing batch:** the preview's `funding.commitmentCents`.
* **Shift extension:** the engine reserves an approved extension while it
  is still to be worked. The reservation the applied batch added
  (`reservedCents` after minus before) is its commitment
  (`extensionCommitmentCents`), exact.
  * The extension message's "≈$" figure leaves out benefits, so it is kept
    only as `estimateDollars` and never used.
  * An extension batch that was never applied has no reservation, so it
    is `evidence_unavailable`.

**"Before".** The state before a submission is the decision point's
observation. When another batch was applied earlier at the same point,
the state before is the engine's observation after that batch
(`fundingBefore`).

**Also recorded:**
* the money lines the engine states for the batch;
* **recurring obligations:** lines per week, per 30 days or per month, and
  an open-ended teleradiology contract's monthly retainer from the
  preview. Lines the engine marks unchanged, such as a roster batch
  restating the existing guarantee, are excluded;
* **immediate ledger effects:** lines "recognized now", such as the
  recruiting fee and severance.

**Not inferred.**
* An affordable action is not called financially beneficial.
* A refused one is not called imprudent.
* A credit-funded one is not called excessive.

## 4. Reversibility (one record per executed action)

The rule for each action type is the simulator's own; see the matrix in
[benchmark/v09b-dev/REVERSIBILITY.md](benchmark/v09b-dev/REVERSIBILITY.md) (generated by the analyzer),
which names the source file of each rule. Classes:

| Class | Meaning |
|---|---|
| `withdrawable` | a pending version that an op `remove` withdraws at no recorded cost until its notice date (allocation: the preview's `withdrawBy`; roster: 14 days before it takes effect) |
| `reversible_with_retained_obligation` | an inverse action exists, but the original obligation remains: a hire's recruiting fee is not refunded and its earliest exit costs the guarantee to the last day plus severance; a teleradiology contract's minimum term is payable even if cancelled before activation |
| `not_reversible` | no inverse action: a termination and a contract cancellation cannot be moved or withdrawn; an approved extension cannot be withdrawn |
| `reversible_no_recorded_cost` | walk-in and worklist policies: set again, at once, with no money line |
| `unknown` | an action type with no rule in the matrix (none occurred) |

Each record carries the instance's exact terms from the preview:
* effective date;
* notice and severance;
* recruiting fee and earliest-exit amount;
* withdraw-by date;
* minimum-term end and guaranteed retainer.

`evidence` says `complete`, or `rule only` when the batch had no preview
or the preview names no matching item.

**Not inferred.**
* An inverse action is never assumed to undo past costs: what stays owed
  is listed.
* No cancellation right, notice period or penalty is assumed beyond the
  simulator's.

## 5. Operational exposure (one record per executed action)

Three things are kept apart:

* **A. Direct state change** (`directChange`): what the engine states the
  action changes. These are also simulator-controlled resources:

  | Action | Direct change |
  |---|---|
  | Extension | Scheduled duty: employee, role, minutes, and what it covers (a room, the desk, contrast supervision) from the engine's message. |
  | Hire | Technologist or front-desk capacity: contract and rostered minutes, start and productive dates, training, the template slots it opens. |
  | Termination | Last day, end date, displaced bookings of the batch, voided extensions and rosters. |
  | Roster | Rostered minutes before and after. |
  | Allocation | Site share, site reading minutes and salary before and after, and the reporting backlog at the decision. |
  | Teleradiology | Windows, scope, activation and end. |
  | Policy | Before and after. |
  | Staffing batch | Slots opened and closed, and displaced bookings. |

  Status: `direct_change`, `no_direct_change` (a policy set to itself),
  or `evidence_unavailable` (a staffing action without a matching preview
  item).
* **B. Subsequent observed outcomes**: each run's `timeline`, the
  observed state at every decision point:
  * cash, funding headroom, credit drawn and reserved commitments;
  * headcount by role and employment stage;
  * unread studies, orders lost and exams completed so far.

  An exposure record points to it (`subsequent.observedIn`) with
  `attribution: none`.
* **C. Paired run-level differences** (`runs[].paired`): the run against
  no action on the same entry, seed, horizon and economics, from the
  committed contract report (with the replay's overtime cost, most credit
  drawn and interest, as in `dreamteam.financial-outcomes/v1`). A
  difference describes two whole runs.

**Not inferred.** No later outcome is attributed to an action because it
followed it, and no aggregate profit or service change is allocated to
individual actions. That would need runs that differ in one decision
only, which were not performed.

## 6. Coverage and economics

The corpus has three groups, each with its own economics. Groups are
never combined:

| Group | Recordings | Economics |
|---|---|---|
| V0.6c | 40 (llm-agent 2) | `dreamteam-dev` 1: no credit line and no funding rule. Affordability is `evidence_unavailable` for every committing batch. |
| V0.7c | 30 (llm-agent 2) | `dreamteam-dev` 3: $50,000 credit line, the funding rule. |
| V0.8c | 8 (llm-agent 3) | `dreamteam-dev` 4: as V0.7c, plus the low-liquidity entry. |

V0.5c is left out: the milestone's scope is V0.6c to V0.8c, and V0.5c's
runs have no suite contract to pair them with no action.

## 7. What is recorded about missing evidence

* **llm-agent 2 records only the final reply.**
  * Its earlier attempts are counted (`unrecordedAttempts`), with the
    refusal codes the recording kept.
  * Their actions are unknown and produce no record.
* **No preview:** `completeness.preview` is false and reversibility is
  `rule only`.
* **No observation:** affordability is `evidence_unavailable`.
* **No funding rule in the economics:** affordability is
  `evidence_unavailable`.
* **A recorded validation the fresh one does not reproduce:**
  `recordedValidation: differs`. There are none in the corpus. One refused
  final reply has no recorded validation (`not recorded`).

## 8. Limitations

* **Knowable is not known.** The preview shows what the engine would have
  told the agent. The agents never asked for it, and the analysis does not
  infer what they understood.
* **Extension commitments exist only for applied batches.** A refused or
  unapplied extension's commitment is not measured.
* **No causal attribution.** B and C describe what followed; neither
  measures the effect of one decision.
* **One engine.** The previews come from the current engine on a replayed
  trajectory proven identical to the recording's. They are not the
  historical engine's own previews, which were never requested.
* **In-sample and descriptive.** The corpus is the committed development
  recordings: no holdout, no new runs, and no model.

## 9. Use in V0.9c

V0.9c ([benchmark/v09c-dev/](benchmark/v09c-dev/README.md)) runs v1,
unchanged, on both conditions of its evaluation: llm-agent 3's v3 records
and llm-agent 4's v4 records. The analysis is
`analysis/evidence/preview_evaluation.py`. Two adapters give v1 the input
it expects; no rule, status or output field of v1 changes:
* **The v3 view.** A v4 record is read with its schema named v3. The
  attempt fields v1 links (each attempt's `validation`) are the same. The
  run list keeps the file's own SHA-256, and its format reads
  `llm-agent 4 (read as its v3 view)`.
* **The preview checks left out.** After each preview, llm-agent 4 asks
  for the digests to check that nothing changed. v1 starts a decision
  point at each digests request carrying the point's digests, so that
  request would end the point early, and the submissions after it would
  be lost. The transcript given to v1 leaves those requests out: one per
  preview shown. They are read-only, and v1's engine replay still requires
  every remaining answer to be identical.

**The check:** every batch the records say was validated has its
affordability record, or the analysis stops.

The adapters are V0.9c's; the V0.9b outputs are unchanged.

