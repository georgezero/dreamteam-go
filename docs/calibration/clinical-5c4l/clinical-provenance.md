# Slice 5C.4l: order provenance and inherited safety

These are the record-integrity changes made in answer to the independent
5C.4k audit
(`docs/audits/slice-5c4k-clinical-safety-protocol-lock-audit.md`, K1 and
K2), under the owner's decisions of 2026-10-20. The evaluator is
0.2.0-dev.23.

**Status:**
* **Not locked, not frozen, not reviewed.** It needs the focused 5C.4m
  review.
* **The clinical policy is unchanged.** Thresholds, windows, minimums,
  the partial-window ceiling, the 45-day follow-up, the absolute limits,
  Gate A, Gate B, the safeguard, the allowance and the T5 tolerances are
  as reviewed in 5C.4k. Only record integrity changes:

```json
"orderStudyIntegrity": "contractInherited",
"orderProvenance": "pairedOrigin"
```

* **Older candidates keep their rules**, byte for byte:
  `internal/eval/testdata/candidate-policy-v5-5c4{d,e,f,h,j}.json`.

## 1. The owner's decisions (K2)

1. Inherited orders keep their original identity and wanted time. None
   escapes the safety cohort by changing its priority, modality, wanted
   time or origin.
2. Inherited orders are reconciled against the paired baseline; so is
   every post-takeover order (K1). Every shared order must match its
   origin's attributes.
3. The inherited-study exemption is removed. An inherited order acquired
   after the takeover has its study, and its reporting obligation is
   evaluated.
4. Outcomes are not required to match. Booking, acquisition,
   cancellation and reporting may legitimately differ; only immutable
   order attributes are matched.
5. No baseline counterpart is manufactured for a candidate-only order.
   Its identity must come from a trusted generation record; where none
   exists, the result is INVALID or INSUFFICIENT, never GO.

## 2. The provenance contract (K1, K2)

### 2.1 The rule

Under `orderProvenance: pairedOrigin` (`provenance.go`,
`orderProvenance`), every comparison reconciles each seed's candidate
record with the paired plan's record of the same seed:

* every order of either record has a nonempty origin, unique in its
  record;
* the two records hold **the same origins**: inherited and new, eligible
  for Gate A or B or not;
* each origin's **immutable request** is the same in both: the order's
  ID, receipt time, wanted time, priority, modality and walk-in flag.

Any difference makes the comparison **INVALID** (reason `order
provenance`). Outcomes are never compared: first appointment,
resolution, displacement, the outcome at each capture, studies and
reports may all differ.

With it, `orderStudyIntegrity: contractInherited` (`checkOrderStudies`)
extends the order–study contract to inherited work:

* every order the record holds as **served**, inherited or new, has
  exactly one post-takeover study, of its request, acquired at its
  resolution;
* every order ID is unique in its record;
* only a study acquired at or before the takeover stays exempt: its
  order was resolved before the takeover and is not in the record.

### 2.2 Why the paired record can authenticate an order

An order's origin is the demand draw that created it: its day, source,
hour block and index in the simulator (`originID`), its creation index
in the fidelity model. Under paired exogenous demand both plans of a
seed receive the same orders:

* **Simulator.** No plan action creates, removes or alters an order:
  referrals, walk-ins and the opening schedule are keyed exogenous
  draws, and the takeover state precedes every plan.
  `TestDemandExogenousAcrossPlans` now checks every order of the record,
  inherited included, with its ID and request, under three plans
  (no action, a teleradiology contract, other walk-in and worklist
  policies). The recorder's inherited orders (open at the takeover, and
  no-shows with a rebooking queued) are the same in every plan.
* **Fidelity model.** Demand comes from a stream shared by both plans,
  and IDs and origins are creation indexes. Before 5C.4l each practice
  of a pair had its own history before the takeover (its own bookings
  from its own booking stream), so the inherited orders differed. The
  model now has a **shared history** (`sharedHistory`, set in every
  manifest practice): the candidate's bookings before the takeover are
  the baseline's, from the baseline's practice and booking stream; its
  own practice and stream decide from the takeover on.
  `TestSharedHistoryReconciles` shows every pair of seven scenarios
  reconciling, with the same studies acquired before the takeover;
  without the shared history the reconciliation fails.

So every legitimate order of a candidate has its counterpart, and an
order of one record alone is a provenance failure.

### 2.3 What the contract does not establish

These are stated, not assumed.

* **Candidate-only orders.** No trusted generation record exists outside
  the records, and no plan in the simulator or the fidelity model
  creates demand. A candidate-only order therefore makes the comparison
  INVALID. If a future plan action can create demand, a trusted
  generation record (the demand draws, outside the plan's control) must
  exist before such orders can be accepted.
* **Both records edited alike** (K3). The paired record is trusted as
  produced by the harness: in process for the reserved run, by committed
  record digests for a file-based evaluation. Identical edits to both
  records are not detectable by a pure evaluator.
* **The study swap.** A study's link to its order is an outcome of the
  candidate's record, and the paired record cannot check it: its
  bookings and studies legitimately differ. Two orders of the same
  priority and modality can exchange their studies and resolutions when
  both were imaged after the later order's wanted time:
  * order O, wanted by the horizon, imaged after it with a report 60 h
    past due (a violation in the follow-up cohort);
  * order X, wanted after the horizon, imaged after its wanted time with
    a report on time;
  * swapped: O's study and resolution become X's and X's become O's.
    Every request still reconciles, both records satisfy the contract,
    and the violation now belongs to an order wanted after the horizon,
    outside the absolute cohort.

  `TestStudySwapLimitation` shows it: **GO**, where the unswapped control
  is NO GO. It needs only the outcomes of one record, so neither the
  contract nor provenance can close it without an authority for the
  link.
  * **Bearing on the confirmatory run:** none. Its records are
    constructed in process by the harness and judged without being
    written or edited in between.
  * **Bearing on other evaluations:** any evaluation of records that an
    adversary could edit (a file-based evaluation) relies on the record
    digests the harness commits.
  * **Options for the owner**, each a decision rather than an
    implementation detail:
    1. accept it as a stated limitation, with the reliance on harness
       provenance written into the protocol (as now);
    2. attribute post-horizon studies conservatively: for each priority
       and modality, the absolute cohort takes the worst-delayed
       post-horizon studies, as many as there are orders wanted by the
       horizon and served after it (the swap cannot change that count).
       This is link-independent but a clinical-policy change: it can
       charge an order wanted after the horizon with another's delay;
    3. widen the absolute cohort to every study acquired by the end of
       observation, also a policy change;
    4. authenticate the link outside the record (a harness-signed record
       digest at production), an architectural change.

## 3. Inherited orders (K2)

`TestInheritedSafety`: 40 inherited soon orders in both plans, received
the day before the takeover, open at it and served after it with their
studies; the target is one of them. The 5C.4j candidate (inherited
exemption, no provenance) is shown beside the 5C.4l candidate.

| Case (inherited order INH0) | 5C.4l | 5C.4j |
|---|---|---|
| inherited orders open at the takeover, served after it, on time | GO | GO |
| control: imaged day 1, reported 60 h past due | NO GO (safety) | NO GO |
| **its study deleted, the order still served** | **INVALID** | GO |
| **order and study relabelled routine** | **INVALID** (provenance) | GO |
| control: imaged H + 1 s, reported 60 h past due | NO GO (safety) | NO GO |
| **the same, study deleted** | **INVALID** | GO |
| **the same, wanted time moved to H + 1 s** | **INVALID** (provenance) | GO |
| control: imaged day 1, unfinished at E (overdue) | NO GO (safety) | NO GO |
| **the same, study deleted** | **INVALID** | GO |
| imaged late, still awaiting its report at E | NO GO (pending: INSUFFICIENT) | NO GO |
| its study duplicated (a second row) | INVALID | INVALID |
| its study relinked to another inherited order | INVALID | INVALID |
| **the inherited order deleted from the candidate's record** | **INVALID** (provenance) | GO |

Every bypass of the audit's K2 table is INVALID under the 5C.4l
candidate and reproduced as GO under the 5C.4j candidate.

`TestOrderProvenance` does the same for K1 and the brief's attacks on
post-takeover orders, starting from the audit's violation (the last soon
order wanted by the horizon, imaged H + 1 s, reported 60 h past due):

| Manipulation | 5C.4l | 5C.4j |
|---|---|---|
| control | NO GO (safety) | NO GO |
| **K1: relinked to a same-request order wanted after H, re-timed consistently, the original lost** | **INVALID** | **GO** (the audit's bypass) |
| the order's wanted time moved to H + 1 s | INVALID | NO GO (Gate B unmatched) |
| the order's origin changed | INVALID | — |
| order and study relabelled routine, or to another modality | INVALID | NO GO (Gate B) |
| two same-request orders' IDs swapped with their studies' links | INVALID | — |
| an order missing; an order duplicated under a new ID | INVALID | — |
| a candidate-only order | INVALID | — |
| an order outside Gate A/B eligibility (routine, wanted after H) re-timed | INVALID | — |
| a legitimately different outcome (the report on time instead) | GO | GO |
| a legitimately different outcome (another order lost instead of served) | NO GO (safety, as the control) | NO GO |

`TestContractInheritedUnit` checks the contract on one record: a served
inherited order without its study is refused under `contractInherited`
and accepted under `contract`; a study acquired before the takeover
without its order stays accepted; a repeated order ID is refused.

## 4. Partial-window reporting (K5)

The 5C.4j documents said that no partial window failed on the
simulator's development seeds. **That was wrong** (corrected in place in
`../clinical-5c4j/clinical-safety.md` §1.4 and §3.1, marked as
corrections). Reported at three levels:

| Level | Development seeds 901–910, no action, 5C.4j candidate |
|---|---|
| **Local window failures** | the leading partial soon window fails the 5% ceiling on 3 seeds: 901 (3 of 42 severe), 902 (3 of 41), 903 (4 of 41); 904 and 907 sit just below (2 of 42, 2 of 46: NOT_JUDGED) |
| **Seed-level clinical failures** | unchanged: seeds 901, 902 and 903 already fail the soon window gate on complete blocks (2, 2 and 5 failing blocks), so the gate's status and the seed's do not change |
| **Whole-evaluation verdicts** | unchanged: every seed of the no-action practice fails other gates |

The 5% ceiling is unchanged. The lesson is about reporting: an aggregate
comparison of seed or evaluation verdicts can hide newly triggered local
failures, so the construction checks now list each stable scenario's
partial-ceiling seeds by priority (T1 and T7 detail), apart from the
seed-level and group-level results.

## 5. The follow-up's financial observation (K6)

With `followUpDays: 45` the end of observation, and the `end` capture,
move from H + 31 d to H + 45 d. The financial measures that use it:

* **solvency:** the minimum cash and the largest past-due payable,
  through the end of observation (`Measure`: `MinCash`, `PastDue`);
* **funding:** committed liabilities against cash at the `end` capture
  (`FundingGap`);
* **liquidity at `end`**, reported.

The finance rules (floors, allowances) are unchanged, so a longer
observation can only add evidence: a later minimum or past-due payable,
and the funding test at a later instant.

**Measured** (`followup-finance.json`, `TestFollowUpFinancialObservation`:
the no-action practice on seeds 901–910, recorded to H + 31 d and to
H + 45 d, measured under the candidate's finance rules):

| | H + 31 d | H + 45 d |
|---|---|---|
| minimum cash | the same on every seed (reached 18 or 170 days after the takeover, the latter H + 21 d: before either end) | the same |
| largest past-due payable | 0 on every seed | 0 |
| funding gap | 0 on every seed | 0 |
| solvent, funded | every seed | every seed |
| cash at the end | $98,964 to $128,418 | $160,043 to $195,305 |

**No financial measure or decision changed** on these seeds. Only the
reported cash at the end differs (fourteen more days of receipts).
Limitation: these are the no-action practice's records. A strategy whose
cash falls, or whose committed liabilities grow, in the fourteen extra
days would be judged on them; that is the intended effect of observing
to the end of the follow-up, not a new rule, and no finance rule was
changed to accommodate it.

## 6. The simulator and replay

* **Operational simulator: unchanged.** No file of the simulator's
  shipped code changes; the evaluation recorder is test code and is
  unchanged in 5C.4l.
* **Simulator records under the candidate:** identical to 5C.4j's on all ten development seeds
(`sim-access-5c4l.json` against `../clinical-5c4j/sim-access-5c4j.json`:
every per-seed field equal): the records satisfy the contract with
inherited work, every seed is valid, and the clinical results, gates and
statuses are unchanged. `TestDemandExogenousAcrossPlans` now checks every
order of the record, inherited included, with its ID and request, under
three plans: identical.
* **Replay:** 0.2.0-dev.22 is registered (`mini-seed52-dev22`); replay
  through dev.22 is exact.
