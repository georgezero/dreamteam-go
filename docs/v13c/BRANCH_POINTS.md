# V0.13c branch-point inventory and selection (`dreamteam-v13c-select/1`)

**The evidence:**
* `docs/benchmark/v13c-dev/points/inventory.json` (SHA-256
  `786837ad…`) and `manifest.json` (`e8d29a96…`).
* Both are made by `TestInventoryV13c`
  (`internal/alternatives/inventory_test.go`) from the V0.8c and V0.9c
  replays. CI builds those replays and rebuilds both files, byte for
  byte, on Linux amd64, Linux arm64 and macOS.
* Nothing in either file was written by hand.

## 1. The pool

**What it is:** the V0.12b pool, unchanged
(`docs/benchmark/v12b-dev/points/manifest.json`, `pool`). It holds the
20 recorded staffing proposals of the V0.10c/V0.11b inventory that are
eligible under shared-prefix/v4 and have a committed context
(`docs/benchmark/v10b-dev`, `v10c-dev` or `v11b-dev`
`points/contexts.jsonl`).

**What is not done:**
* No trajectory is made or extended, and no context is rebuilt from
  material outside the repository.
* A category the pool cannot fill is reported as unavailable (§4),
  never filled from elsewhere.

**Each point on the engine:**
1. The context line is found by its SHA-256 (the pool's
   `sourceLineSha256`).
2. The recording's replay is found by the replay's own record of what it
   replayed (as `decision_experiment.replay_runs`).
3. A session is opened on the run's scenario and seed. The recorded
   requests are fed through the agent protocol (`agent.Handler`), and
   every answer must equal the recorded one: for hello, its clock and
   scenario fingerprint. Feeding stops at the digests answer equal to
   the context's digests, as the V0.12b driver's `to_point` does.
4. The recorded reply is split as the agents split it
   (`llm_agent3.split`): its operations batch, then its staffing batch.
   Both are validated, and the result must equal the recorded
   validation (the pool's, where a V0.10b line has none).
5. Rule v2 and both arms' turns are built for the staffing batch
   (`Steps`). The two arms must agree on whether the turn is due.
6. The displaced count is recorded (staffing preview), and so are the
   slots closed when the batch is terminations only.

## 2. Categories (first match, in this order)

| Category | Test, at the recorded state |
|---|---|
| `termination_keep` | turn due; the batch holds a termination; 0 displaced |
| `termination_displacement` | turn due; a termination; displaced > 0 |
| `staffing_adjustment_displacement` | turn due; displaced > 0; a roster or allocation change |
| `retained_obligation` | turn due; a teleradiology start or a hire |
| `credit_supported_not_due` | valid; turn not due; the inventory's tag `credit_supported` |
| `valid_expensive_not_due` | valid; turn not due; stake ≥ $10,000 |
| `nonqualifying_control` | valid; turn not due |
| `refused_qualifying` | refused; rule v2 applies to the proposal as made, so a correction that keeps its action would make the turn due |
| `refused_other` | refused; the rule does not apply |

## 3. The rule `dreamteam-v13c-select/1`

**What it takes:**
* every point in the four "turn due" categories. These are the only
  points where the arms differ;
* one point of each valid not-due category, as A/A controls;
* two `refused_qualifying` points, at most one per model;
* no `refused_other` point.

**Limits:**
* Each point is taken once, with at most two per recording.
* Within a category, the order is SHA-256(`dreamteam-v13c-select/1|<point id>`).

**What it reads:** no model response other than the recorded proposal
itself. It does not prefer points where a model changed its decision in
an earlier experiment, and it was written knowing those experiments'
results exist. Every pooled point was in an earlier experiment (prior
exposure below), because only those have committed contexts.

## 4. The inventory

| Point | Model | At | Staffing batch (+ operations) | Validation | Rule applies | Alternatives | Displaced | Slots closed | Turn | Category | Selected | Prior exposure |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| fp-qwen-s41901-p25 | Qwen | 2026-11-07 | allocation (E08) | accepted | no | — | 0 | — | not due | nonqualifying_control | **yes** | V11B |
| fp-qwen-s41901-p26 | Qwen | 2026-11-08 | allocation (E07, E08) | accepted | no | — | 0 | — | not due | nonqualifying_control | | V10C, V11B |
| fl-gemma-s41901-p26 | Gemma | 2026-11-08 | teleradiology | refused: invalid_schema | no | — | 0 | — | not due | refused_other | | V10B, V10C, V11B |
| fl-gemma-s41901-p27 | Gemma | 2026-11-09 | teleradiology | refused: insufficient_funds | yes | proposal, no_action | 0 | — | not due (refused) | refused_qualifying | | V10B, V10C, V11B |
| fl-gemma-s41901-p28 | Gemma | 2026-11-10 | teleradiology | refused: invalid_schema | yes | proposal, no_action | 0 | — | not due (refused) | refused_qualifying | **yes** | V10B, V10C, V11B |
| fl-qwen-s41901-p9 | Qwen | 2026-10-22 | hire; allocation (E08) | refused: batch_rejected, insufficient_funds | yes | proposal, no_action, without_1 | 0 | — | not due (refused) | refused_qualifying | | V10C, V11B |
| fl-qwen-s41901-p11 | Qwen | 2026-10-24 | allocation (E08) | refused: insufficient_funds | no | — | 0 | — | not due | refused_other | | V10C, V11B |
| fl-qwen-s41902-p24 | Qwen | 2026-11-06 | terminate E11; terminate E01 | accepted | yes | proposal, no_action, without_1, without_2 | 6 | 300 | **due** | termination_displacement | **yes** | V11B |
| fl-qwen-s41902-p27 | Qwen | 2026-11-09 | terminate E04; terminate E01 | refused: notice_too_short, batch_rejected | yes | proposal, no_action, without_1, without_2 | 0 | — | not due (refused) | refused_qualifying | | V11B |
| fp-qwen-s42001-p7 | Qwen | 2026-10-20 | hire (+ extend_shift) | refused: lead_too_short | yes | proposal, no_action, adjusted | 0 | — | not due (refused) | refused_qualifying | **yes** | V11B |
| fp-qwen-s42001-p24 | Qwen | 2026-11-06 | teleradiology | refused: contract_overlap | yes | proposal, no_action | 0 | — | not due (refused) | refused_qualifying | | V11B |
| fp-qwen-s42001-p27 | Qwen | 2026-11-09 | teleradiology; terminate E01 | refused: invalid_schema, batch_rejected | yes | proposal, no_action, without_2 | 0 | — | not due (refused) | refused_qualifying | | V10C |
| ss-gemma-pv-s42001-p14 | Gemma | 2026-10-27 | teleradiology start | accepted | yes | proposal, no_action, adjusted | 0 | — | **due** | retained_obligation | **yes** | V10B, V10C, V11B |
| ss-qwen-s42001-p1 | Qwen | 2026-10-14 | teleradiology start (+ extend_shift) | accepted | yes | proposal, no_action, adjusted | 0 | — | **due** | retained_obligation | **yes** | V10B |
| ss-qwen-s42002-p1 | Qwen | 2026-10-14 | teleradiology (+ 2 extend_shift) | refused: invalid_schema | yes | proposal, no_action | 0 | — | not due (refused) | refused_qualifying | | V10B |
| ss-qwen-pv-s42002-p6 | Qwen | 2026-10-19 | allocation (E07, E08) (+ 2 extend_shift) | accepted | no | — | 0 | — | not due | credit_supported_not_due | **yes** | V10B, V10C, V11B |
| ss-qwen-s42002-p26 | Qwen | 2026-11-08 | five roster changes | refused: notice_too_short, batch_rejected | no | — | 0 | — | not due | refused_other | | V10B |
| ss-qwen-s42001-p20 | Qwen | 2026-11-02 | six roster changes; teleradiology start | accepted | yes | proposal, no_action, without_7 | 405 | — | **due** | staffing_adjustment_displacement | **yes** | V10C |
| ss-qwen-s42001-p23 | Qwen | 2026-11-05 | teleradiology (+ 3 extend_shift) | refused: past_date | yes | proposal, no_action | 0 | — | not due (refused) | refused_qualifying | | V10C |
| ss-qwen-s42002-p7 | Qwen | 2026-10-20 | teleradiology start (+ extend_shift) | accepted | yes | proposal, no_action, adjusted | 0 | — | **due** | retained_obligation | **yes** | V10C, V11B |

* Rule v2 removed no candidate at any point: no recorded proposal
  repeats an irreversible action.
* Five refused teleradiology points list the adjustment as omitted: the
  engine refused the adjusted contract too (`inventory.json`, `omitted`).

## 5. Selected (9) and unavailable

| Category | Selected |
|---|---|
| termination_keep | **unavailable** |
| termination_displacement | fl-qwen-s41902-p24 |
| staffing_adjustment_displacement | ss-qwen-s42001-p20 |
| retained_obligation | ss-qwen-s42002-p7, ss-qwen-s42001-p1, ss-gemma-pv-s42001-p14 |
| credit_supported_not_due | ss-qwen-pv-s42002-p6 |
| valid_expensive_not_due | **unavailable** |
| nonqualifying_control | fp-qwen-s41901-p25 |
| refused_qualifying | fl-gemma-s41901-p28 (Gemma), fp-qwen-s42001-p7 (Qwen) |

**Unavailable or thin categories,** reported as limits and not filled
with new trajectories:
* **termination_keep.** No valid recorded termination has zero
  displaced bookings: the only valid termination, fl-qwen-s41902-p24,
  displaces 6. "Termination versus keep" without displacement is
  covered only by the constructed pairs S1 (member a), S3 and S6
  ([SENSITIVITY_CASES.md](SENSITIVITY_CASES.md)).
* **valid_expensive_not_due.** The one valid not-due proposal with a
  stake over $10,000 (ss-qwen-pv-s42002-p6, $44,687.30) is
  credit-supported and taken in that category first. The other two
  (fp-qwen-s41901-p25 and p26) are $9,863.02 each.
* **Credit-supported commitments where the turn is due.** None.
  ss-qwen-pv-s42002-p6 is an allocation change: neither irreversible nor
  displacing, so rule v2 does not apply and both arms are identical
  there.
* **A valid hire.** None, as in V0.12b. fp-qwen-s42001-p7 reaches a
  hire's turn only if the model corrects the refused hire.
* **Gemma.** One point with a due turn (ss-gemma-pv-s42001-p14) and one
  refused qualifying point.
* **No-action alternatives.** `no_action` is listed at every due point
  by the rule, so it is not a separate category.
