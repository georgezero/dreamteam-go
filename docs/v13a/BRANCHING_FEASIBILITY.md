# V0.13a branching feasibility

Status: V0.13a (0.13.0-dev.1), design only. No branching code was
written.

**The question:** can the current engine and session API compare two
alternatives exactly from the same decision point?

**Sources:**
* `internal/session/session.go`, `record.go` (`Record`, `Replay`,
  `Digests`) and `t90.go` (an existing paired run).
* `internal/sim/world.go`, `engine.go` and `digest.go`.
* The CLI `run`, `verify` and `agent`.
* The mechanics check in §3.

## 1. Capabilities

| Capability | Supported | How, or what is missing |
|---|---|---|
| Clone a state | **No, not in memory.** **Yes, by rebuilding.** | Neither `sim.World` nor `session.Session` has a clone, snapshot or restore. A state is rebuilt exactly from its record: the canonical scenario, the seed, and every batch at its instant (`session.Replay`; the same pattern `T90Job.Run` uses with fresh worlds). |
| Apply an action to one copy | Yes | `Session.Apply` on one rebuilt session. |
| Leave another copy unchanged | Yes | A second rebuilt session is a separate `World`. It shares nothing with the first. |
| Advance both deterministically | Yes | `Session.AdvanceTo`, or `Engine().RunUntil`, with no decision. Both are deterministic for the same scenario, seed, batches and engine. |
| Compare their outcomes | **Partly** | Aggregate outcomes: `session.Result` (the `run` JSON: money, operations, balances, obligations), and `eval.Record` with `eval.Evaluate`. The canonical evaluation policy is defined for the `sustainable` scenario only (`ErrT90Scenario`). Per-entity alignment is not exposed (§4). |
| Preserve the original session | Yes | A branch is built from the record, never from the live session. `validate` and `preview` leave every digest unchanged (§3). |
| Reproduce both branches | Yes | `Session.Save`, then `Replay` or `dreamteam verify`. Rerunning a branch reproduces it exactly (§3). |
| Record branch identities | **No** | A saved record has a checksum and digests, but there is no notion of a branch, its parent or its alternative. |
| Verify every digest | Yes, per branch | `Replay` checks the log, ledger, actions, state and run digests, and every action result. That the prefixes match must be checked separately, by comparing both branches' digests at the decision instant. |

**Conclusion:**
* Exact branching is supported by rebuilding from the record.
* It is not supported by copying in memory.
* No approximation is needed, and none is proposed.

## 2. Restrictions

1. **Earlier rules.**
   * A branch re-applies the recorded batches with this build's rules.
   * A record whose batches replay only under earlier rules cannot be
     branched. This applies to the batch modes `legacy`, `slice2` to
     `slice4`, `dev6` and `dev7` (`currentRules`).
   * `T90Job` refuses the same case (`ErrT90Rules`), and a branch
     builder must too.
2. **Engine identity.**
   * Both branches must run on the same engine as each other.
   * The decision point's record must replay exactly on that engine.
   * A record from another engine is refused, not approximated.
3. **Holdout seeds.**
   * A branch must refuse sealed holdout seeds, as every harness does
     (`eval.GuardManifest`).
4. **Cost.**
   * Each branch reruns from the takeover. It cannot start from the
     decision instant.
   * Measured on the Raspberry Pi at `8bbeca2`, 45 days take about 2.2
     seconds and 26 MB (`mini`), or about 2.9 seconds and 56 MB
     (`established`).
   * Two to four branches per decision point are cheap.
   * A large search would not be. `ALTERNATIVE_SELECTION.md` limits
     the number.

## 3. Mechanics check

This check exercises the existing commands. It is not an evaluation,
and its outcome numbers are not evidence about any decision.

**The decision point:**
* `mini`, seed 52, 2026-10-13 07:00.
* Through `dreamteam agent -stdio`: `advance` to 07:00, then `digests`,
  then `validate` and `preview` of `terminate E04, lastDay
  2026-10-27`, then `digests` again.
* Both digest answers were identical. Validation and preview changed
  nothing.

**The branches:**
* `dreamteam run -scenario mini -seed 52 -days 45 -actions <file> -save
  <file> -verify-replay -format json`, with:
  * **A:** one batch, the termination at 07:00;
  * **B:** no batch.

**Results:**
* Both runs saved, replayed and verified (`"verified": true`).
* `dreamteam verify` returned `ok` with the same run digests:
  * A: `808d448b…`;
  * B: `8618a73e…`.
* Rerunning A gave the same result in every field.
* Both branches received the same 2,224 orders: demand arrivals are
  common to the two branches.
* 130 of 204 result fields differed, starting from the termination.

The command lines and file hashes are in
`docs/audits/v0.13a-baseline.md`.

## 4. What is missing for a comparison engine

**E1: a branch builder.** A read-only function in `internal/session`,
following `T90Job`'s pattern:

```
Branch(rec *Record, at simtime.Time, alts [][]api.Action, until simtime.Time) ([]BranchResult, error)
```

What it does:
* It rebuilds a fresh world per alternative from the record's scenario,
  seed and batches before `at`.
* It checks that every branch has the same digests at `at`.
* It applies the alternative's batch, or nothing.
* It advances to `until` with no decision.
* It returns each branch's digests at `at` and at `until`, its record
  and its result.

What it changes: it adds no simulator rule and changes no existing
function.

**E2: a branch identity.**
* SHA-256 over:
  * the contract schema;
  * the engine;
  * the scenario fingerprint;
  * the seed;
  * the checksum of the decision point's record;
  * `at`;
  * the alternative's canonical actions JSON;
  * the continuation identity;
  * `until`.
* The branch's own saved record makes it replayable.

**E3: a comparison reader.**
* It reads the same named metrics from each branch result.
* It does no arithmetic across classes (`COMPARISON_CONTRACT.md`).

**E4 (optional): capacity for a termination.**
* `slotsClosed` for termination-only batches
  (`TERMINATION_VS_KEEP.md` G1).
* Changing `PreviewStaffing`'s default output would change the bytes of
  the preview answers recorded in earlier experiments.
* Historical replay rebuilds prompts from those answers and checks
  their SHA-256. So the field must be opt-in: a new request argument,
  or a new read-only op. It must not be a default change.
* Alternatively, a branch can read the coverage of each later day after
  applying the termination. Coverage depends on the registry and
  rosters only, not on hidden attributes.

**Per-entity alignment (not proposed):**
* Appointment, visit and claim IDs are assigned in sequence and diverge
  between branches.
* Orders are keyed internally by origin (day, source, block, index),
  but the API does not expose that key.
* Comparing individual patients across branches would need a new
  read-only view. It is not needed for aggregate comparison, and is not
  proposed for V0.13b.

## 5. The smallest extension

E1 to E3 are the minimum:
* they are read-only;
* they are additive;
* they live outside `internal/sim`.

E4 is optional and must be opt-in.

None of them changes:
* simulator mechanics;
* accounting;
* action schemas;
* scoring;
* any historical output.
