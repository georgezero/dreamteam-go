# V0.13c candidate selection: `dreamteam.alternative-selection/v2`

Status: implemented in 0.13.0-dev.2 (V0.13c), in
`internal/alternatives/selection.go`. It is used by both arms' decision
turns (`internal/alternatives/arms.go`).

**The text:**
* `internal/alternatives/text/selection-v2.txt` is embedded in the build
  and pinned by SHA-256
  `5f32b2c84a0b8383f1a213ce52745e56f5d5aa8292bc714096d1beec70debcf9`.
* `TestSelectionV2PinnedText` fails if the text changes without its pin.
  It also fails if the text stops being v1's text with only its name
  changed and one section appended.

**v1 is unchanged:**
* `dreamteam.alternative-selection/v1`
  (`158a6254952ee3c19dba0b2ea045d2d02de510e7f7f7537cf435357facc5e358`) is
  still `Select`. Its text, code path and pin are as in V0.13b.
* Every V0.13b selection, comparison and evidence file replays as it
  did: `docs/benchmark/v13b-dev/` is unchanged byte for byte, and its
  tests pass.

## 1. The only change

v2 makes v1's list exactly as v1 does, including v1's limit of four.
Then it removes every candidate whose actions are identical to those of
an earlier candidate in the list, and records each removal.

| | v1 | v2 |
|---|---|---|
| When it applies, the order, leave-one-out, the adjustments, the limit | as specified | the same, unchanged |
| Two candidates with identical batches | both listed | the later one removed, recorded in `removed` with `sameAs` the earlier |
| A removed candidate's place | n/a | left empty: never refilled, never substituted |
| Candidate IDs | `without_<k>` by batch position | the same IDs (no renumbering) |
| Candidate identity (`sha256`) | SHA-256 of {rule v1, ID, actions} | SHA-256 of {rule v2, ID, actions} |

**Identical batches.**
* Two batches are identical when their canonical form is byte for byte
  equal. The canonical form is each action decoded and encoded again,
  with the actions in batch order.
* This is the identity the branch builder gives a batch
  (`session.checkAlternatives`, which refuses two branches with the same
  batch). So a v2 list can always be branched for evaluation.
* IDs, labels, sources and descriptions are never compared.

**Order is part of the batch.**
* The engine validates and applies a batch in order: its per-action
  results are in batch order, and a repeated target is refused at its
  second position.
* So two batches with the same actions in a different order are
  distinct.
* This occurs among listed candidates only when a non-irreversible
  action sits between two identical irreversible ones (for example
  `[terminate E04, worklist, terminate E04]`). There, `without_1` and
  `without_3` are both kept.

**What it never does:**
* add a candidate, or list one after a removal that v1 did not list;
* reorder, renumber or change a kept candidate, its validation or its
  actions;
* change the proposal, its validation or its codes;
* repair a refused proposal: it stays listed, refused, with the engine's
  codes;
* change the meaning of `no_action`, which is the empty batch, always
  second, and never removed. Only an earlier empty batch could remove it,
  and the proposal is never empty when the rule applies;
* change simulator action semantics, validation or any engine answer.
  It reads the same engine answers as v1.

**Recording:**
* `Selection.removed` is absent when nothing was removed. v1 never
  writes it.
* The agent audience records the rule's name and pin. Under v2,
  `Agent.check` refuses a comparison that lists one batch twice, even
  if its digest is recomputed.
* `VerifySelection` replays a selection under the rule it names, and
  refuses a pin that is not that rule's.

## 2. Examples (`docs/benchmark/v13c-dev/selection-v2.json`)

All cases are on mini, seed 52, on 2026-10-13.

| Proposal | v1 | v2 | Removed |
|---|---|---|---|
| terminate E04 | proposal, no_action, adjusted | the same | none |
| terminate E04 twice (F4); refused | proposal, no_action, without_1, without_2 | proposal, no_action, without_1 | without_2 = without_1 |
| terminate E03, E03, E04; refused | proposal, no_action, without_1, without_2 | proposal, no_action, without_1 (no without_3) | without_2 = without_1 |
| terminate E04, worklist, terminate E04 | proposal, no_action, without_1, without_3 | the same: reordered batches are distinct | none |
| terminate E04 Oct 27, E04 Nov 10 | 4 candidates | the same: different values | none |
| terminate E03, E04 | 4 candidates | the same | none |

In the F4 case:
* v1's comparison lists two identical alternatives.
* `BuildEvaluation` refuses it (`session.ErrBranchInput`: two
  alternatives with the same batch).
* v2's comparison branches, and the evaluation verifies.

## 3. Tests (`internal/alternatives/selection_v2_test.go`)

| # | Required | Test |
|---|---|---|
| 1 | Duplicate termination | `TestSelectionV2DuplicateTermination` |
| 2 | Duplicate leave-one-out | `TestSelectionV2DuplicateLeaveOneOut` |
| 3 | Different labels, identical actions | `TestSelectionV2LabelsNotCompared`, `TestSelectionV2DuplicateTermination` (`without_1`/`without_2`) |
| 4 | Different action order | `TestSelectionV2ActionOrder` |
| 5 | Different action values | `TestSelectionV2DifferentValues` |
| 6 | Invalid original proposal | `TestSelectionV2DuplicateTermination` (refused, codes kept, not repaired) |
| 7 | No-action duplicate | `TestSelectionV2NoAction` |
| 8 | Candidate limit | `TestSelectionV2DuplicateLeaveOneOut` (80 proposals of two or three terminations, and two longer batches: v2 ⊆ v1, at most 4, never refilled) |
| 9 | Stable ordering | `TestSelectionV2Stable` |
| 10 | Stable identities | `TestSelectionV2Stable` |
| 11 | Historical v1 compatibility | `TestSelectionV2KeepsV1`; every V0.13b test and `docs/benchmark/v13b-dev/` byte for byte |
| 12 | Cross-platform determinism | `TestEvidenceSelectionV2` (`selection-v2.json`) and `TestEvidenceSensitivityPairs`, byte for byte in CI on Linux amd64 and arm64 and macOS |

Every v2 case goes through one check, `sameButDuplicates`. It requires:
* v2's list to be v1's with exactly the recorded removals;
* every kept candidate to equal v1's apart from its rule-naming
  identity;
* every removal to name an earlier kept candidate with the same batch;
* the kept batches to be distinct.

Mutation checks are listed in `docs/v13c/F4_REMEDIATION.md` §3.
