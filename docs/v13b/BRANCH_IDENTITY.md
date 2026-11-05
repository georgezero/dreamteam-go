# V0.13b branch builder and branch identity (E1, E2)

Status: implemented in 0.13.0-dev.2, `internal/session/branch.go`. It is
read-only: it never changes the record or the session it starts from,
and it changes no simulator mechanics.

## 1. What a branch is

A branch is one alternative, played from one decision point:

1. **Rebuild.** The saved decision-point record is replayed from
   scratch in a fresh world (`session.Replay`). Every digest must match
   the record; nothing is copied from a running world.
2. **Check the start.** The rebuilt state's digests must equal the
   decision point's. Every branch therefore starts from the same state.
3. **Apply.** The alternative's batch is applied at the decision
   instant through `Session.Apply`, the only way to change a run. An
   empty batch (`no_action`) applies nothing.
4. **Continue.** The branch runs with no further management action to
   the common horizon (§3).
5. **Record.** The branch is saved as its own `dreamteam.session/v2`
   record: the source's batches, then the alternative's batch. It
   replays independently, like any session file.

`Branch(record, alternatives, options)` builds one set; the
`Session.Branch` method takes the session's own record at its current
time and only reads the session.

## 2. What is refused (fail closed)

Nothing is approximated. `Branch` refuses:

| Condition | Error |
|---|---|
| The record's checksum does not match its content | `ErrCorrupt` |
| A holdout seed (the embedded manifest guard) | the guard's error |
| A record, or a prefix it continues, whose batch mode does not use this build's action set and rules (`currentRules`) | `ErrBranchRules` |
| A record or prefix replayed with another state encoding (`StateChanged`) | `ErrBranchRules` |
| A replay that does not reproduce the record, or a fingerprint or time that disagrees with it | `ErrMismatch` |
| No alternative, more than eight, an empty or repeated ID, two alternatives with the same batch, or an action type outside the eight existing ones | `ErrBranchInput` |
| An unknown continuation, or horizon days out of range | `ErrBranchInput` |

A record from an earlier build that uses this build's action set and
rules (for example 0.12.0, batch mode `v0.12.0`) is branched exactly:
its replay is `earlier_build_exact`, and every digest is verified.

## 3. Continuation and horizon

The only continuation is `dreamteam.continuation/no-further-action-v1`:

* no management action of any kind after the decision instant;
* the policies in force continue.

**The horizon** is common to every branch of a set:

`horizon = max(decision + AfterDecisionDays, latestEffect + AfterLatestEffectDays)`

* `latestEffect` is the latest `World.LatestEffect` over every branch
  after its batch. This is the start of the day of the latest effect:
  an end date, a productive date, or a roster, allocation or contract
  date.
* The default is 30 and 30 days, following the evaluation policy's
  `daysAfterLatestEffect`.
* Pass 1 replays each branch to its batch and reads `latestEffect`.
  Pass 2 replays each branch again and runs it to the horizon.

## 4. Branch identity: `dreamteam.branch-identity/v1`

| Field | Content |
|---|---|
| `sourceChecksum` | the decision-point record's checksum |
| `scenarioFingerprint` | the scenario's fingerprint |
| `seedDigest` | SHA-256 of `dreamteam.seed/v1`, the source checksum and the seed (below) |
| `decisionSec`, `decisionTime` | the decision instant |
| `decisionRunDigest` | the run digest at the decision point |
| `engine` | this build's version |
| `actions`, `actionsSha256` | the alternative's batch, canonical JSON, and its SHA-256 |
| `continuation`, `continuationSha256` | the continuation and the SHA-256 of its JSON |
| `horizonSec`, `horizon` | the common horizon |
| `digest` | SHA-256 of the identity's JSON with `digest` empty |

**Properties, each tested:**
* Identical inputs give identical identities, field for field.
* A different batch, continuation or common horizon gives a different
  identity. The horizon changes with a sibling alternative's later
  effect, even when this branch's batch is unchanged.
* The alternative's ID is a label, not part of the identity. Two
  alternatives with the same batch in one set are refused.

**The seed is not written in clear.**
* The identity binds the seed through `seedDigest`.
* A digest of a small integer cannot hide it from someone who
  enumerates seeds: it binds the seed, it does not make it secret.
* The agent audience carries no identity and no seed.

## 5. Verification

**`VerifyBranch(source, branch)` checks, for one branch:**
* that the identity digest matches;
* that the identity follows from the source record's own fields;
* that the branch's record is the source's run: same scenario, seed and
  takeover, and the source's batches unchanged, then at most the
  alternative's batch at the decision instant, with the recorded results;
* that the record was saved at the identity's horizon, with the
  recorded horizon digests;
* that a replay of the record from scratch reproduces every digest;
* that the horizon is at least what the branch's own effects require.

**`VerifyBranchSet(source, set)` checks, for the whole set:**
* every branch;
* one source, one continuation and one horizon;
* that the horizon equals the continuation's rule applied to the latest
  effect of every replayed branch.

A forged continuation that one branch alone would accept is refused at
the set level.

Tampering is tested: edited identities, actions, results, digests, batch
contents, checksums, a missing record, a forged continuation and a branch
checked against another source are all refused.

## 6. Cost

Measured on the Raspberry Pi (linux/arm64), mini scenario:
* Two branches to a 45-day horizon, with set verification, took about
  6 seconds.
* Three branches to the 59-day horizon of the evidence case, verified,
  took about 19 seconds.

Branches are built one after another; at most one world is held at a
time beyond the source replay.
