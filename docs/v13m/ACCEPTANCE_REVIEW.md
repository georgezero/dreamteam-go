# V0.13m-lite bounded technical acceptance review

**Implementer-performed bounded technical acceptance review (owner Phase 9).
This is not an independent review and not a human review.** The owner
decided that no separate independent review is run for V0.13m-lite; the
implementer performed this one bounded review after implementation and
gates. It does not satisfy, and is not a substitute for, the pending human
presentation review. It authorizes nothing: protocol lock NO GO, live
inference NO GO, V0.13.0 release NO GO.

Scope (exactly the owner's five items plus the original DT13L witnesses):
1. strict response JSON; 2. wall-clock halt causality; 3. fail-closed
persistence; 4. complete-run evidence; 5. historical preservation; and the
original DT13L-01..04 witnesses. Standard: the owner's
[revised acceptance contract](ACCEPTANCE_CONTRACT.md). Rule: if any defect
permits a false COMPLETE or an accepted contradictory PARTIAL, NO GO;
otherwise separate availability limitations from scientific-integrity
failures.

Method. Adversarial archives are genuine recordings of the implementation
under test, mutated with every dependent hash, journal copy, workflow
delivery copy, row commitment, failure mirror, counter and seal binding
recomputed, then assessed by structural classification and FULL analysis;
genuine-provenance archives through the ordinary command line
(`v13m_verify.py analyze`, exit status). Every case is run against V0.13l
(and, for the DT13L witnesses, the pre-fix successor) as well, so each
rejection is shown to be caused by the change. Scripted model replies and
scripted health only; no model endpoint was contacted. Reviewed commit: the
V0.13m-lite implementation commit chain on top of `133e21d` (the final
commit is in the owner handoff; a commit cannot name its own hash).

## 1. Strict response JSON (DT13L-01)

| Evidence | V0.13l | V0.13m-lite |
| --- | --- | --- |
| [DT13L-01 witnesses](dt13l/after-v13m.json): 15 re-sealed and 17 genuine-wire archives, ordinary CLI duplicate `choices` | 12/33 meet expectation (every ambiguous body accepted as COMPLETE) | 33/33: 20 ambiguous archives (duplicate, escaped-equivalent, nested, deep-nested and usage duplicates; NaN, Infinity, -Infinity, 1e999, -1e400) INVALID and refused, including ordinary CLI (exit 1); 11 controls (Unicode literal/escape, 2^80+1, signed zero and exponent forms, valid usage) accepted; trailing-content and truncated bodies accepted only as unparsed replies with null metadata |
| [Review probes](review/probes-after-v13m.json): ambiguous body only in the call-journal copy; in an aborted draw; in an interrupted (halted) row; in a `draw_unrecorded` journal outcome | accepted PARTIAL in 3 of 4 | all refused (R-JSON-004) |
| Probe: ambiguous body only in the workflow delivery copy | refused by replay | refused by replay |
| Probe: response nested beyond the parser recursion limit | INVALID | INVALID (never COMPLETE with that response read) |
| Reviewer's original V0.13l `duplicate-response` CLI archive | accepted, exit 0 | refused, exit 1 (R-JSON-004 reported besides the identity mismatch) |
| Unit tests `test_v13m.StrictResponse` | — | pass |

Finding: no ambiguous response contributes to COMPLETE or PARTIAL analysis in
any retained location exercised. The frozen workflow, parser and
model-visible bytes are unchanged (protected-file check).

## 2. Wall-clock halt causality (DT13L-02)

| Evidence | V0.13l | V0.13m-lite |
| --- | --- | --- |
| Draw-start and request-start "0 s elapsed" (synthetic fixtures) | PARTIAL, accepted | INVALID, refused (R-HALT-003) |
| Message one second below the guard threshold (both stages) | PARTIAL, accepted | INVALID, refused |
| Message exactly at the threshold (legitimate rounding) | accepted | accepted |
| Ordinary CLI, real UTC: "0 s" message; elapsed 10^9 s with a seconds-long timestamp span | PARTIAL, accepted (exit 0) | INVALID, refused (exit 1) |
| Ordinary CLI, linked clocks: valid halts just past and 1 h past the threshold | accepted | accepted |
| Ordinary CLI, linked clocks: elapsed beyond the timestamps; message below threshold | accepted | refused |
| Probe: ordinary CLI draw-call-limit halt with elapsed inflated by 3 h | accepted | refused (timestamp disagreement) |
| Reviewer's original V0.13l `wall-message` CLI archive | accepted, exit 0 | refused, exit 1 (R-HALT-003) |

Finding: a halt whose recorded elapsed time could not have triggered the
guard, or which disagrees with the structured timestamps, is refused; the
four-hour request-start guard is unchanged and no hard deadline is claimed.

## 3. Fail-closed persistence (DT13L-03/04, policy disposition)

| Evidence | V0.13l | V0.13m-lite |
| --- | --- | --- |
| Reviewer's nested recovery failures (bookkeeping; recovery entry) | sealed `draw_unrecorded` PARTIAL accepted; completed result silently lost | `RecordingError`, unsealed, INVALID, refused; `evidenceLoss.possible = true` in `persistence-failure.json` and the structured error; transport invocations = journal dispatches; no later draw begun |
| Reviewer's retention-construction failure | INVALID; known response lost | INVALID; response retained in the archive and the structured error; loss disclosed |
| Recorder without the fail-closed guard (verifier defense) | — | PARTIAL classification but FULL analysis refused (R-HANDOFF-003) |
| Probes: third fault while disclosing the loss; KeyboardInterrupt as the secondary fault; nested failure in draw 2 after draw 1 committed | accepted PARTIAL in all three | fail closed in all three (unsealed, INVALID, refused, no retry, no replacement; draw 1 kept in the ledger) |
| Persistent storage failure at the first ledger commit | INVALID, refused | INVALID, refused, no retry, no replacement draw |
| Successor fault matrix | — | 5,288 cases at 1,619 points agree; zero false COMPLETE |

Finding: every exercised persistence or recovery failure stops the run,
yields an unsealed INVALID archive excluded from analysis, makes no retry or
replacement call/draw, and discloses possible evidence loss. Evidence loss in
an unrecoverable double failure remains possible (a completed result held
only in the failed draw's frame); under the owner's revised standard this is
non-blocking because the run is rejected and the loss reported. DT13L-03/04
are therefore a policy disposition, not resolved defects.

## 4. Complete-run evidence

* Positive controls: complete fixtures, ordinary-CLI genuine recordings,
  aborted-workflow, interrupted-row and `draw_unrecorded` controls are
  accepted with exact replay of every valid draw.
* 66-unit fidelity across all 21 contexts (including the nine historical
  contexts restored from recorded replay states): COMPLETE, every completed
  draw VERIFIED, byte-identical on regeneration.
* Binding matrix: 350 inherited + 119 relationship cases, zero surviving
  contradictions; 17 lifecycle examples; 76 witnesses; 125 mutations; 16
  regressions; DT13K witnesses on V0.13m-lite 35/35 (earlier fixes kept).
* Primary-outcome counterexample unchanged: control abandonment and treatment
  `select:no_action` both submit `[]`; total variation and per-identity
  difference are zero.
* No false COMPLETE anywhere in the fault matrix, signals (353), loopback
  (12) or ownership enumeration (6 + 9 + 258 + 62).

## 5. Historical preservation

`v13m_preserved.py`: all 3,606 protected files at `133e21d` byte-identical
(every V0.13l and earlier module, recording, audit and the human-review
freeze); 262-draw schedule identical to V0.13l; 802-call ceiling; nine
historical contexts; six sensitivity pairs; 93 transcripts and seven static
assets (`v13i_human_review.py --check`, `MANIFEST.sha256`). Completion seal
and finalization contract unchanged. Engine `0.13.0-dev.2`. Only
`.github/workflows/{ci,v13f}.yml`, `docs/PROGRESS.md`, `docs/BENCHMARK.md`
and `docs/ROADMAP.md` changed in place. Historical, V0.13l and earlier
stage gates: see the [gate record](gates/README.md).

## Development record

The first run of the review probes picked the first retained call body,
which in three bases was a scripted transport failure (empty body); those
three probes therefore did not test what they named (one appeared to be an
accepted ambiguous `draw_unrecorded` PARTIAL). The probes were corrected to
change a 200 response body and to fail loudly if none exists; the corrected
probes are the recorded ones. No implementation change resulted.

## Separation of findings

**Scientific-integrity failures found:** none. No exercised case produces a
false COMPLETE, an accepted contradictory PARTIAL, invalid evidence in
analysis, or an automatic retry or replacement draw.

**Availability limitations (non-blocking, disclosed):** evidence loss in an
unrecoverable double failure (run rejected, loss reported); genuine runs
rejected by fail-closed verification (a `draw_unrecorded` draw whose retained
calls complete the workflow; a server response with duplicate keys,
non-finite or overflowing numbers; a monotonic/UTC clock step above 60 s); no
crash recovery or resumption; no power-loss or SIGKILL guarantee; the seal is
a self-hash. See [known limitations](KNOWN_LIMITATIONS.md).

**Limits of this review:** it was performed by the implementer, not
independently; it is finite engineering evidence; a writer able to rewrite
every trace consistently can still forge an archive (threat model unchanged).

## Verdict

Within the five items and the original DT13L witnesses: **technical
acceptance GO under the owner's revised acceptance standard** (no
scientific-integrity failure found; availability limitations disclosed).
Human presentation review: **PENDING**. Protocol lock: **NO GO** (human
review pending; owner decision required). Live inference: **NO GO**.
