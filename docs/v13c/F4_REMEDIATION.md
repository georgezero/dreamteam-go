# V0.13c F4 remediation: duplicate alternatives

**The finding** (`docs/audits/v0.13b-independent-review.md`, F4):
* A proposal that terminates the same employee twice produced two
  identical leave-one-out candidates under
  `dreamteam.alternative-selection/v1`.
* `BuildAgent` listed both, and `BuildEvaluation` refused the pair.

**The owner's decision** (V0.13b acceptance):
* Fix it narrowly. When two leave-one-out candidates produce identical
  canonical action batches, keep only the first occurrence under the
  existing order.
* Never substitute another candidate, and keep the limit of four.
* Never repair an invalid proposal.
* Give the rule a new identity and hash, and preserve v1 and its
  evidence.

## 1. What was done

| | |
|---|---|
| New rule | `dreamteam.alternative-selection/v2`, `text/selection-v2.txt`, SHA-256 `5f32b2c84a0b8383f1a213ce52745e56f5d5aa8292bc714096d1beec70debcf9` ([SELECTION_RULE_V2.md](SELECTION_RULE_V2.md)) |
| The change | after v1's list is made, a candidate whose canonical batch equals an earlier candidate's is removed, and the removal recorded (`removed`, `sameAs`) |
| v1 | unchanged: `Select`, `text/selection-v1.txt` (`158a6254…`), its code path, and all V0.13b evidence |
| Code | `SelectV2` and `SelectRule`; `dedupe` and `batchKey`; `VerifySelection` replays under the rule named; `Agent.check` accepts v1 and v2 with their own pins, and under v2 refuses a repeated batch |
| Files outside `internal/alternatives` | none. No simulator, session, validation, accounting, scoring or action-schema file changed |

**Scope.** The rule removes any later candidate whose batch equals an
earlier one, not only leave-one-out pairs:
* Under v1, only leave-one-out candidates can coincide. The proposal
  and the adjustment of a one-action batch always differ, and
  `no_action` is the only empty batch.
* So in practice the removal is exactly the leave-one-out case the owner
  named.
* The general wording means a later rule change cannot create an
  unhandled duplicate silently.

**Order.** The canonical batch keeps batch order, as the branch builder
does:
* The engine validates a batch in order, so `[A, B]` and `[B, A]` are
  different batches.
* Only one listed case is affected, `[terminate X, operations action,
  terminate X]`. Both reordered leave-one-out candidates are kept there
  (`TestSelectionV2ActionOrder`).
* This is a choice the reviewers and the owner can see. Treating order
  as irrelevant would merge them, and would need a further rule version.

## 2. Effect

| Proposal (mini, seed 52, Oct 13) | v1 candidates | v2 candidates |
|---|---|---|
| terminate E04 twice, refused | proposal (refused), no_action, without_1, without_2 | proposal (refused), no_action, without_1 |
| terminate E03, E03, E04, refused | proposal (refused), no_action, without_1, without_2 | proposal (refused), no_action, without_1 |

* In both, `without_3` is not listed in either rule. v2 does not
  substitute it for the removed duplicate.
* The refused proposal stays refused, with its codes.
* v2's comparison branches and verifies (`BuildEvaluation`,
  `VerifyEvaluation`).
* v1's comparison of the first case is refused by the branch builder,
  as before.

**The historical inventory:** across the 20 pooled points on the engine
(`docs/benchmark/v13c-dev/points/inventory.json`), v2 removed nothing.
No recorded proposal repeats an irreversible action.

## 3. Tests and mutation checks

**Tests:** the twelve required tests are mapped in
[SELECTION_RULE_V2.md](SELECTION_RULE_V2.md) §3.

**Mutation checks:**
* 17 mutations of the rule, the verifier, the agent check and the arms'
  turns were run (`~/dt-audit-kept/v013c/mutate.py`, kept outside the
  repository). Each was run against its targeted test.
* 15 were caught on the first pass. The two survivors were:
  * **F6** (`VerifySelection` accepts any pin): the replay comparison
    refused the edited selection anyway, so the pin check was
    unobservable. The test now requires the rule error itself.
  * **A3** (inspect answers not checked): the test's probe phrase also
    occurs in the summary, so the summary check caught it first. The
    probe is now a phrase that occurs only in inspect answers.
* All 17 are caught now:

| # | Mutation | Test |
|---|---|---|
| F1 | duplicates not removed | `TestSelectionV2DuplicateTermination` |
| F2 | removal names the wrong first occurrence | `TestSelectionV2DuplicateTermination` |
| F3 | batch order ignored | `TestSelectionV2ActionOrder` |
| F4 | dedupe applied under v1 too | `TestSelectionV2KeepsV1` |
| F5 | identity does not name the rule | `TestSelectionV2DuplicateTermination` |
| F6 | verifier accepts any pin | `TestSelectionV2Stable` |
| F7 | v2 comparison with a repeated batch accepted | `TestSelectionV2Stable` |
| F8 | candidate IDs compared | `TestSelectionV2DuplicateTermination` |
| F9 | rule text changed | `TestSelectionV2PinnedText` |
| F10 | list refilled after a removal | `TestSelectionV2DuplicateLeaveOneOut` |
| A1 | refused proposal gets the decision turn | `TestArmsNotDue` |
| A2 | turn texts not checked | `TestArmsBoundary` |
| A3 | inspect answers not checked | `TestArmsBoundary` |
| A4 | digest check removed | `TestArmsBoundary` |
| A5 | arms use rule v1 | `TestArmsTermination` |
| A6 | control turn names the alternatives | `TestArmsTermination` |
| A7 | treatment text not the comparison's presentation | `TestArmsTermination` |

## 4. Historical compatibility

* `docs/benchmark/v13b-dev/` (five files) is unchanged byte for byte.
  `TestEvidenceTermination` and `TestEvidenceScripted` pass.
* Every V0.13b test passes unchanged: `TestSelectionTermination`,
  `TestSelectionRules` and the rest. They call `Select`, which is
  rule v1.
* The engine version is not changed (0.13.0-dev.2). V0.13c changes no
  engine, session or simulator file, and a version bump would change the
  engine string embedded in V0.13b's evidence (V0.13b review F1). See
  `docs/PROGRESS.md`, V0.13c.
