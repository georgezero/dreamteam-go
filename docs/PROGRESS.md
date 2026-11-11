# DreamTeam — Progress

One section per roadmap step (ROADMAP §2), newest first. Each records what
was built, what the player can see, how it was tested, and known
limitations. The version shown in the TUI top bar (`DREAMTEAM v0.1.0-dev.N`)
and by `dreamteam version` names the latest step a build contains.

---

## V0.13n — human review and protocol-lock preparation (0.13.0-dev.2, engine unchanged)

The owner accepted V0.13m-lite as an engineering milestone: DT13L-01 and
DT13L-02 accepted as fixed, DT13L-03/04 accepted as documented fail-closed
limitations, the revised acceptance contract approved, the 60-second clock
tolerance approved provisionally, and recorder development stopped. The
V0.13m-lite technical acceptance review was implementer-performed, not
independent.

**What was done (preparation only, no code or evidence changed):**

* Baseline at `9d4626b` recorded with exact hashes in
  [v13n/README.md](v13n/README.md): the frozen human-review package
  (`MANIFEST.json` `31bec875…`, 93 transcripts, 7 static assets, all checks
  pass), simulator source, protocol and schedule drafts (262 draws, 802-call
  ceiling, 14,400-second request-start guard), acceptance standard (60-second
  tolerance pinned), recorder, verifier and contract identities.
* Human reviewer packet defined in
  [v13n/human-review/REVIEWER_PACKET.md](v13n/human-review/REVIEWER_PACKET.md):
  frozen model-visible material, the 12-item questionnaire and instructions
  only; no technical audit conclusions. Required record contents: identity,
  expertise, independence declaration, 12 responses, specific findings,
  overall recommendation.

**Status:** human presentation review PENDING (no reviewer assigned; none
invented). Bounded independent technical check PENDING (deferred by the
owner). Lock candidate and lock gate not started. Protocol lock: NO GO. Live
inference: NO GO. V0.13.0 release: NO GO.

---

## V0.13m-lite — experimental readiness closure (0.13.0-dev.2, engine unchanged)

The owner ended the open-ended recorder-hardening cycle after the V0.13l
independent review (NO GO; DT13L-01..02 BLOCKER, DT13L-03..04 MAJOR) and
directed a bounded remediation: fix the two scientific-integrity blockers and
adopt a conservative fail-closed persistence policy for the double-fault
findings — an explicit, owner-approved change to the acceptance standard.
[Remediation and dispositions](v13m/DT13L_REMEDIATION.md). Bounded technical
acceptance review: performed by the implementer (not an independent review,
not a human review) — [record](v13m/ACCEPTANCE_REVIEW.md). Human
presentation review: PENDING (frozen package `docs/v13i/human-review` at
`ee987a1`, unchanged; [questionnaire and instructions](v13m/human-review/README.md)).
Protocol lock: NO GO. Live inference: NO GO. V0.13.0 release: NO GO.

**What was built:**

* [Revised acceptance contract](v13m/ACCEPTANCE_CONTRACT.md)
  (`dreamteam.acceptance-standard/v1`, machine form
  `analysis/alternatives/acceptance_v1.json`): BLOCKING — false COMPLETE,
  contradictory PARTIAL accepted, invalid evidence in analysis, automatic
  retries or replacement draws; NON-BLOCKING IF DISCLOSED — a genuine failed
  run classified INVALID, evidence loss in an unrecoverable double failure
  provided the run is rejected, no automatic crash recovery, no power-loss
  guarantee. DT13L-03/04 are recorded as a policy disposition, not as
  resolved.
* DT13L-01 fixed (R-JSON-004): the verifier reparses every retained model
  response body (draw calls and call-journal outcomes) under strict JSON;
  duplicate keys at any depth (including escaped-equivalent), NaN/Infinity,
  overflow and any disagreement with the workflow parser's reading make the
  archive INVALID; metadata comes from the strict reading. Bytes, workflow
  and parser unchanged.
* DT13L-02 fixed (R-HALT-003): a wall-clock halt's displayed duration must be
  able to have triggered the four-hour request-start guard
  (`threshold − 0.5 < shown ≤ elapsed + 0.5`) and, with system timestamps,
  every halt's elapsed time must agree with the structured journal timestamps
  (60 s tolerance). No hard deadline is claimed.
* Fail-closed persistence (DT13L-03/04): a failure while recovering a draw's
  result poisons the archive and stops the run unsealed (INVALID), disclosing
  `evidenceLoss`; the failure-report fallback keeps the response held at
  receipt and discloses the secondary failure
  (`failed-persistence-retention/v3`); the verifier refuses a
  `draw_unrecorded` PARTIAL whose retained calls complete the workflow
  (R-HANDOFF-003). No retry, resumption, replacement draw or new recovery
  machinery.
* Successor identities: driver /9, run /9, analysis /10, partial-run /9,
  experiment /9 (`v13m-alt-1`, schedule entries identical to V0.13l).
  Lifecycle /4, evidence bindings /2, the completion seal and its
  process-crash commit point are unchanged.

**Engineering evidence** ([gate record](v13m/gates/README.md), [test map](v13m/gates/TEST_MAP.md)):
DT13L witnesses before/after on genuine archives (V0.13l and the pre-fix
successor: 24/56 cases meet the remediation expectation; V0.13m-lite: 57/57,
including one verifier-defense case), including ordinary-CLI
reproductions; 14 additional review probes (V0.13l 7/14, V0.13m-lite 14/14);
the reviewer's original V0.13l CLI archives accepted by V0.13l and rejected
by V0.13m-lite on the semantic relation. Successor regeneration: 350 + 119
binding-matrix cases (zero survivors), 5,288 fault cases at 1,619 points
(zero false COMPLETE), 353 real signals, 12 real loopback, 6 + 9 + 258 + 62
ownership cases, 16 regressions, 76 witnesses, 125 mutations, 17 lifecycle
examples, 66-unit fidelity (COMPLETE/VERIFIED); DT13K witnesses on V0.13m
(35/35). Full Go, race, vet, gofmt, five cross-builds, golden
references, benchmark verify/reproduce, historical replays, frozen evidence,
V0.12b/V0.12a/V0.11b and every earlier V0.13 stage (including V0.13l's
unchanged modules) pass on horo; 742 Python tests pass (689 preserved, 53
successor). Both cross-platform workflows cover the successor; final-push
results are in the owner handoff.

**Preserved design:** simulator, accounting, session, API, scoring, decision
workflow, parser, presentation, alternative selection, primary estimand,
nine historical contexts, six sensitivity pairs, 262 scheduled draws,
802-call ceiling, 93 transcripts and seven static assets. All 3,606
protected starting files (every V0.13l and earlier file) are byte-identical
([baseline](v13m/BASELINE.md)); only CI workflows and bookkeeping documents
changed in place.

**Baseline:** `133e21d2181c10f910da00c0339fce77257d51f9` (V0.13l `7b21d93`
plus its verbatim independent review). The protected swap file and dt are
excluded from stages and archives.

**Known limitations:** [details](v13m/KNOWN_LIMITATIONS.md) — evidence can
still be lost in an unrecoverable double failure (the run is then rejected
and the loss disclosed); fail-closed verification rejects some genuine runs
(availability); inherited V0.13l limitations. No clinical suite, historical
live rerun, reserved/holdout seed, model endpoint, protocol lock, live
inference, tag or release.

---

## V0.13l — canonical evidence and durable handoff closure (0.13.0-dev.2, engine unchanged)

The owner directed V0.13l after the V0.13k independent review (NO GO;
DT13K-01..03 BLOCKER, DT13K-04..05 MAJOR, DT13K-06 MINOR, DT13K-07 NOTE),
with a change of approach: contract-driven rather than witness-driven.
[Per-finding remediation](v13l/DT13K_REMEDIATION.md). Separate independent
review: PENDING. Human presentation review: PENDING (separate,
`docs/v13i/human-review` at `ee987a1`, unchanged). Protocol lock: NO GO.
Live inference: NO GO. V0.13.0 release: NO GO.

**What was built:**

* A generalized, executable [evidence inventory](v13l/EVIDENCE_BINDINGS.md)
  (`dreamteam.evidence-bindings/2`, committed first in `b13dc42`): three
  guarantees (canonical, causal, ownership), strict JSON semantics
  (`dreamteam.strict-json/v1`) and 17 cross-field relationships — raw versus
  structured JSON, dispatch versus journal, response versus transport record,
  halt versus reconstructed workflow, failure versus lifecycle event,
  completed draw versus pending record, pending record versus ledger, ledger
  versus seal, call counts versus limits — each planned under seven kinds
  (valid agreement, missing evidence, contradiction, duplicate evidence,
  incorrect identity, incorrect type, persistence interruption).
* Canonical consistency (DT13K-01): the verifier compares every retained JSON
  representation with strict typed equality (boolean, integer, float, string,
  null and absence distinct; exact integers; bitwise floats; duplicate keys
  and non-finite numbers refused); request bytes are reparsed independently
  and compared with the structured request; replay compares request bytes and
  objects. A self-consistent seal never overrides a mismatch.
* Causal consistency (DT13K-02, DT13K-03): closed failed-engine entries only
  as the last engine event and equal to the draw failure; raised transport
  outcomes terminal and equal to the draw failure; stop messages derived from
  code and fields; reconstructed replay failures compared; halts
  reconstructed from retained calls, limits and workflow phase (stage, draw
  and run counts, transport-failure streak, guard order and message), with
  only the wall-clock measurement external.
* Durable ownership (DT13K-04, DT13K-05): the completed draw is owned in the
  statement that receives it, its handoff is inside the recovery scope, and
  run_draw bookkeeping after the final digests is recovered and replayed; a
  received model response is owned at receipt and consumed only after its
  outcome record is durable (`dreamteam.response-handoff/v1`), otherwise the
  draw ends with `ResponseNotRecorded`, no replacement call is made and the
  response is retained (`dreamteam.failed-persistence-retention/v2`, also in
  the structured recording error). The journal close is injectable.
* DT13K-06 stays a conservative INVALID, documented as an availability
  limitation. DT13K-07: generated fault contexts keep `pendingRetained`,
  `receivedUnrecorded` and every preservation key; the oracle fails a context
  that drops one.
* Successor recording (driver/8, run/8), lifecycle /4, partial-run /8 and
  analysis /9. The verifier shares only binding data with the recorder.

**Engineering evidence** ([gate record](v13l/gates/README.md), [test map](v13l/gates/TEST_MAP.md)):
the original witnesses reproduced before and after remediation on genuine
archives (V0.13k: 9/27 witness cases meet the remediation expectation; the
pre-fix successor: the same 9/27; V0.13l: 27/27 with 8/8 positive controls
accepted); 350 inherited and 119 relationship full-analysis matrix cases
(zero surviving contradictions; 17 real persistence interruptions INVALID
with evidence preserved); 6 + 9 ownership fault cases and exhaustive
line-boundary enumeration (258 completed-draw and 62 received-response
sites, all retaining their evidence, no replacement call); 5,288 lifecycle
fault cases at 1,619 points (zero false COMPLETE); 353 real-process signal
and 12 real loopback cases; 16 DT13J regressions, 76 witnesses, 125 re-sealed
mutations; 16 lifecycle examples and the 66-unit fidelity fixture. Full Go,
race, vet, gofmt, five cross-builds, golden references, benchmark
verify/reproduce, historical replays, frozen evidence, V0.12b/V0.12a/V0.11b
and every earlier V0.13 stage pass on horo; 689 Python tests pass. Both
cross-platform workflows cover the successor; final-push results are in the
owner handoff.

**Preserved design:** simulator mechanics, accounting, session, API,
scoring, decision workflow, parser, presentation, view, alternative
selection, system instruction, canonicalization and primary canonical
submitted-action estimand. Nine historical contexts, six sensitivity pairs,
21 contexts, 262 scheduled draws, the 802-call ceiling, 93 transcripts and
seven static assets retain their hashes. All 3,387 protected starting files
(including every V0.13k module and evidence file) are byte-identical; the
completion seal format and process-crash commit point are unchanged. See
[baseline](v13l/BASELINE.md) and [preservation](v13l/PRESERVATION.md).

**Baseline:** `ea7bf4bc4718397b2d73568e7a5279104fc795b4` on main (V0.13k
`105fba3` plus its verbatim independent review). The protected swap file and
dt are excluded from stages and source archives.

**The engine version remains 0.13.0-dev.2.** Recording, verification,
analysis, lifecycle, binding and evidence contracts receive
[successor identities](v13l/CONTRACT-IDENTITIES.json).

**Known limitations:** DT13K-06 conservative INVALID; recorder-internal
faults between receipt and durable record are INVALID with the response
retained; no recovery or resumption of INVALID archives; no power-loss or
SIGKILL recovery; storage that refuses every later write leaves the evidence
in the structured error only; the seal is a self-hash, not an authenticity
anchor. [Details](v13l/KNOWN_LIMITATIONS.md).

**Review preparation only:** PENDING repository stubs, an external brief and
a read-only archive of the final commit prepare the separate independent
review (Astra, Extra High). No independent review, human acceptance,
clinical suite, historical live rerun, reserved/holdout seed, protocol lock,
live inference, tag or release was run or authorized.

---

## V0.13k — evidence binding and verification closure (0.13.0-dev.2, engine unchanged)

The owner directed V0.13k after the V0.13j independent review (NO GO;
DT13J-01..05). All five dispositions are FIX; exception identity is fixed
narrowly. [Per-finding remediation](v13k/DT13J_REMEDIATION.md). Separate
independent review: PENDING. Human presentation review: PENDING (separate,
`docs/v13i/human-review` at `ee987a1`, unchanged). Protocol lock: NO GO.
Live inference: NO GO. V0.13.0 release: NO GO.

**What was built:**

* One explicit [evidence inventory](v13k/EVIDENCE_BINDINGS.md), schema
  `dreamteam.evidence-bindings/1`, with 35 bindings, 293 named fixed fields
  and recursively compared domain descendants. The baseline and inventory
  were committed first (214c35a), before recording or verifier changes.
  Types, requiredness, lifecycle stages, related evidence, invalidity,
  replay, absence and permitted external-only UNVERIFIED scope are explicit.
* Successor recording (driver/7, run/7), lifecycle /3, independent partial-run
  verification /7 and full analysis /8. The verifier shares only binding
  JSON data with the recorder; it has its own timestamp and health-body
  validation and reconstructs engine/workflow events through frozen code.
  External failures never waive retained decisions, submissions, responses,
  counts, call bounds, exchanges or state-digest checks. Unknown call arrays
  still require request reconstruction from the retained journal prefix.
* Exactly ordered cleanup/finalization visits, interval and interrupt
  reconciliation, context bounds derived from scheduled contexts independently
  of nullable draw calls, lowercase hexadecimal hash grammar and the absolute
  802-call ceiling. The existing completion seal, finalization contract and
  process-crash rename commit point are preserved.
* A durable completed-result handoff, retained failed ledger/handoff
  candidates, pending-aware error reconciliation and exact failure evidence.
  Six real filesystem/return/error-handling cases retain and replay the
  completed result. Invalid archives are retained for audit, never silently
  repaired into accepted runs. Actual trusted health exception class objects
  are matched by identity; same-text impostors are refused. Duplicate keys
  and nonfinite health JSON use the existing invalid_json category.
* Generated engineering evidence: 350 full-analysis binding cases (35 matching
  cases accepted, 315 contradictory cases rejected, no surviving contradiction);
  5,288 lifecycle fault cases at 1,619 points, all agreeing,
  zero false COMPLETE and full declared transition/stop coverage;
  353 real-process signal/termination cases; 12 real loopback cases;
  16 focused regressions; 76 inherited witnesses; 125 re-sealed mutations
  including one positive control; six completed-result preservation cases;
  16 deterministic lifecycle examples; and 66 successor fidelity
  draws across all scheduled context/model/arm units, including nine historical
  contexts, all recorded and independently replayed.

**Inventory amendments during implementation:** fixed descendant types and
conditional nullability are explicit; the latency field spelling is corrected;
canonical JSON comparison preserves boolean/numeric type identity. Retained
journal outcomes without status require transport-error evidence and no body.
The final inventory drives field validation, stage shapes, cleanup order,
hash grammar and call bounds. The additional journal-prefix regression closes
an accepted valid-hex false request in the unknown-call-array path.

**Preserved design:** simulator mechanics, accounting, session, API, scoring,
decision workflow, parser, presentation, view, alternative selection, system
instruction, canonicalization and primary canonical submitted-action estimand.
Nine historical contexts, six sensitivity pairs, 21 contexts, 262 scheduled
draws, the 802-call ceiling, 93 transcripts and seven static assets retain
their hashes. All 3,172 protected starting files are byte-identical; earlier
source, recordings and independent reports remain unchanged. See
[baseline](v13k/BASELINE.md) and [preservation](v13k/PRESERVATION.md).

**Baseline:** `cb5ac6832e7b43ef68212355ff395cf5958fff11` on main, verified
ancestor (V0.13j plus its verbatim independent review). The protected swap
file and dt are excluded from stages and source archives.

**The engine version remains 0.13.0-dev.2.** Engine behavior and model-visible
material do not change. Recording, verification, analysis, lifecycle, binding
and evidence contracts receive [successor identities](v13k/CONTRACT-IDENTITIES.json).

**Quality gates:** [gate record](v13k/gates/README.md) and
[test map](v13k/gates/TEST_MAP.md). Heavy Go/Python, historical replay,
fidelity, real signals and loopback run on horo. Focused Pi checks use the
memory-limited scope. Both cross-platform workflows cover the successor.
Exact final-push workflow results are recorded in the owner handoff.

**Known limitations:** no recovery or resumption of INVALID archives; no
claim to recover arbitrary power loss or in-memory evidence lost to forced
termination before its first durable handoff. If storage rejects every
subsequent write, existing candidates are retained and the caller receives
the pending result and error evidence. Only external causes may remain
UNVERIFIED. The self-hash seal is not an authenticity anchor. These bounded
availability limitations do not permit false COMPLETE or contradictory
PARTIAL. [Details](v13k/KNOWN_LIMITATIONS.md).

**Review preparation only:** PENDING repository stubs, an external brief and
a read-only archive of the final commit prepare the separate independent
review. It must reproduce all five original findings, generate additional
cross-field contradictions, recompute hashes/seals and use FULL analysis.
No independent review, human acceptance, clinical suite, historical live
rerun, reserved/holdout seed, protocol lock, live inference, tag or release
was run or authorized.

---

## V0.13j — safe-point interrupts and sealed run completion (0.13.0-dev.2, engine unchanged)

The owner directed V0.13j after the V0.13i independent review (NO GO;
DT13I-01..06): the final seal must be the only evidence that a run completed;
never falsely COMPLETE; contradictory PARTIAL evidence must be rejected;
known calls and completed draws must not be lost; a genuine failure safely
classified INVALID is an acceptable, documented availability limitation.
[Per-finding response](audits/v0.13j-audit-response.md). Independent technical
re-review (Extra High): PENDING. Human presentation review: PENDING
(separate; docs/v13i/human-review at ee987a1, unchanged). Protocol lock: NO
GO. Live inference: NO GO. V0.13.0 release: NO GO.

**What was built:**

* A safety contract, [dreamteam.run-lifecycle/v2](v13j/SAFETY_CONTRACT.md)
  (machine-readable `analysis/alternatives/lifecycle_v2.json`), committed before
  any recorder change (8b37738): COMPLETE/PARTIAL/INVALID acceptance rules, the
  owner's severity hierarchy, ten declared safe points, the durable call
  journal, the completion seal and finalization protocol with an explicit
  durability boundary, per-stage evidence fields, transition predicates,
  closed health exception families, the strict timestamp grammar, invariants
  S01–S04 and L01–L20, and the declared fault-injection operations.
* A recorder (driver/6, run/6) whose SIGINT handler only counts; interrupts
  act only at safe points; every model call is journalled durably before
  dispatch and on return; the completion seal, renamed into place after every
  bound file is durable, is the commit point; engine sessions run in their
  own process group. The public result reports every interrupt.
* An independent verifier and analyzer (partial-run/6, analysis/7) that
  requires a valid seal, never treats it as sufficient, and recomputes every
  binding from the persisted files; it does not import the recorder.
* Generated evidence: a fault matrix generated from the contract (4,916 cases at 1,511 points over nine bases, all agreeing, no false COMPLETE, every transition and every admissible stop observed); 326 real-process signal and termination tests (external and process-group SIGINT, SIGINT on the DT13I-01 source lines, SIGKILL at every reached point, SIGTERM); 12 real loopback tests on horo; 76 regression witnesses, including every DT13I counterexample;
  125 re-sealed contradictory-archive checks. See [fault verification](v13j/FAULT_MATRIX.md).

**Contract amendments during implementation** (inside the contract file,
recorded here rather than rewritten into 8b37738; compare
`git diff 8b37738 -- analysis/alternatives/lifecycle_v2.json`): the per-call
order is dispatch <= sent <= outcome (the intent precedes the draw's own send
time; the specification commit had the first two reversed); a fault before an
outcome record poisons the archive and a deferred recorder fault still
unresolved when the run stops fails closed (`safePoints.windows`, with the
expected-outcome rule refined to match); an explicit threat model (the seal
is a self-hash, not a signature).

**Durability boundary:** process-crash consistency is claimed and tested
with real SIGKILL at every reached operation boundary; power-loss durability
is not claimed (design intent only: fsync ordering before the seal rename;
an uncertain case can only revert to INVALID).

**Preserved design:** nine historical contexts, six context pairs, 21
contexts, two arms, 262 scheduled draws, 802 maximum calls, four-hour
request-start guard, canonical submitted-action primary outcome, unchanged
decision workflow, parser, presentation, view, selection, system
instruction, canonicalization, simulator, session, API, scoring and draw
contract (draw/v4; `v13h.py` and `stage_b.py` byte-identical). All 2,984
starting files other than PROGRESS and the two workflows retain their bytes
([baseline](v13j/BASELINE.json)); the 93 transcripts and 7 active static
assets are byte-identical to the frozen human-review manifest.

**Baseline:** 72dc35aa7e73bccf6ca64a3857bc3d58dfabfd43 on main (V0.13i plus
the verbatim independent review). The protected swap file and dt are
excluded from stages and source archives.

**The engine version remains 0.13.0-dev.2.** Simulator, session, API,
mechanics, accounting, scoring and decision operations are unchanged; only
run recording and verification receive [successor identities](v13j/CONTRACT-IDENTITIES.json).

**Quality gates:** see [the gate record](v13j/gates/README.md) and the
[test map](v13j/gates/TEST_MAP.md); retained logs and the review's witnesses
are under /home/george/dt-audit-kept/v013j/. Heavy checks, real signals and
loopback tests run on horo; the Pi runs memory-limited focused checks.

**Known limitations (availability, documented):** a recorder fault inside
the stop, cleanup or finalization sequence (other than the session release),
a fault before an outcome record, any persistence failure and a hard kill
before the seal each leave an unsealed INVALID archive with its durable rows
and call journal intact; there is no recovery or resumption. A SIGINT during
a model call or health read takes effect when that call returns (at most the
request timeout). A writer able to rewrite every archive file consistently
can remove evidence that leaves no other trace; an external anchor of the
seal would be an owner decision.

**Not run or authorized:** independent technical re-review, human review,
clinical suites, historical live reruns, reserved/holdout seeds, protocol
lock, live inference, tag or release.

---

## V0.13i — run lifecycle state machine and systematic fault verification (0.13.0-dev.2, engine unchanged)

The owner accepted V0.13h as a remediation milestone (not technical
clearance) and directed a lifecycle redesign for DT13H-01 to DT13H-04 and
timestamp validation for DT13H-05. [Per-finding response](audits/v0.13i-audit-response.md).
Independent technical re-review (Extra High): PENDING. Human presentation
review: PENDING, limited to the 93 unchanged model-visible transcripts.
Protocol lock: NO GO. Live inference: NO GO. V0.13.0 release: NO GO.

**What was built:**

* A versioned lifecycle contract, [dreamteam.run-lifecycle/v1](v13i/LIFECYCLE.md)
  (machine-readable `analysis/alternatives/lifecycle_v1.json`), committed before
  any recorder change: 10 recorder states plus verifier-only INVALID, 19 legal
  transitions, stop codes per state, an aborted-draw stage table, structured
  health categories, required artifacts, counters, timestamp rules, interrupt
  and fail-closed semantics, finalization steps, 19 invariants and the declared
  fault-injection operations, phases, kinds and expected-outcome rule.
* A recorder (driver/5, run/5) that changes state only through that table,
  with a structured health probe, ledger reconciliation on interrupts, SIGINT
  deferred outside interruptible regions, and FINALIZING-marker finalization
  that fails closed.
* An independent classifier and analyzer (partial-run/5, analysis/6) that loads
  the same contract, does not import the recorder, and derives the state from
  the persisted artifacts; PARTIAL runs never produce complete-run comparisons.
* A fault matrix generated from the contract (601 cases at 240 points, all
  agreeing), 19 real-process signal tests, 11 real loopback tests on horo, 71
  regression witnesses from the V0.13g/V0.13h reviews and 69 re-sealed
  contradictory-evidence mutants. See [fault verification](v13i/FAULT_MATRIX.md).
* A frozen, limited human presentation review package (commit ee987a1):
  [human-review](v13i/human-review/README.md).

**Contract amendments during implementation** (all inside the contract file,
recorded here rather than rewritten into the earlier specification commit
44efcb1): the marker-removal operation has no reachable `after` phase; the
os_error read-failure category admits any other OSError subclass (by class);
ordinary runs must observe the server, only checked fixtures may not; a SIGINT
handled in the recorder's own frame inside a region is deferred.

**Preserved design:** nine historical contexts, six context pairs, 21 contexts,
two arms, 262 scheduled draws, 802 maximum calls, four-hour request-start
guard, canonical submitted-action primary outcome, unchanged decision workflow,
parser, presentation, selection and draw contract (draw/v4, V0.13h `run_draw`).
All 2,884 starting files other than PROGRESS and the two workflows retain their
bytes ([baseline](v13i/BASELINE.json)); the 93 transcripts and 7 active static
assets are byte-identical to the frozen human-review manifest.

**Baseline:** a4d7204c14f36977fee044b3d1a17518d58cc02c on main.
The protected swap file and dt are excluded from stages and source archives.

**The engine version remains 0.13.0-dev.2.** Simulator, session, API,
mechanics, accounting, scoring and decision operations are unchanged; only run
recording and verification receive [successor identities](v13i/CONTRACT-IDENTITIES.json).

**Quality gates:** see [the gate record](v13i/gates/README.md); retained logs
and before-fix witnesses are under /home/george/dt-audit-kept/v013i/. Heavy
checks and loopback tests run on horo; the Pi runs memory-limited focused checks.

**Not run or authorized:** independent technical re-review, human review,
clinical suites, historical live reruns, reserved/holdout seeds, protocol
lock, live inference, tag or release. No post-hoc recovery of hard-killed
archives exists (they are INVALID); whether one is wanted is an owner decision.

---

## V0.13h — run-level failure recording and verification closure (0.13.0-dev.2, engine unchanged)

The owner accepted V0.13g with conditions and limited remediation to DT13G-01,
DT13G-02 and DT13G-03 disclosure. [Per-finding response](audits/v0.13h-audit-response.md).
Independent technical re-review: PENDING. Human presentation review: PENDING.
Protocol lock: NO GO. Live inference: NO GO. V0.13.0 release: NO GO.

**What was built:**

* Successor health evidence accepts supported failed reads without inventing
  responses, retaining scheduled boundary, attempted request, timestamp and
  model-call accounting. External causes remain UNVERIFIED.
* A guarded run lifecycle creates a PARTIAL manifest before fallible work,
  checkpoints progress and preserves primary failures alongside cleanup errors.
* Critical JSON uses same-directory atomic replacement and supported fsyncs.
  Unfinished or inconsistent checkpoints never pass as COMPLETE.
* Partial verification checks event identities, stages, counts, artifacts and
  cleanup; incomplete runs have no complete-run arm or pair comparisons.
* A successor human-review package preserves all previous delivered messages
  and adds the disclosed initial invalid-number feedback interaction and
  record-only lifecycle examples. Human review of the exact package is pending.

**Preserved design:** nine historical contexts, six context pairs, two arms,
262 scheduled draws, 802 maximum calls, four-hour request-start guard, canonical
submitted-action primary outcome and unchanged decision workflow/presentation.
All pinned predecessor evidence and three independent audits retain their bytes.

**Baseline:** 24df940db349657bcf088ae56f480526d015a945 on main.
The protected swap file and dt are excluded from stages and source archives.

**The engine version remains 0.13.0-dev.2.** Simulator, session, API, mechanics,
accounting, scoring and decision operations are unchanged. Only recording and
verification receive [successor identities](v13h/CONTRACT-IDENTITIES.json).

**Quality gates:** see [the gate record](v13h/gates/README.md); full retained
logs and before/after witnesses are under /home/george/dt-audit-kept/v013h/.
Heavy checks run on horo; the Pi runs memory-limited focused checks only.

**Not run or authorized:** focused independent re-review, human acceptance,
clinical suites, historical live reruns, reserved/holdout seeds, protocol lock,
live inference, tag or release.

---

## V0.13g — final technical remediation (0.13.0-dev.2, engine unchanged)

The owner accepted DT13F-01 through DT13F-07 from the independent re-review as
remediation baseline. [Per-finding response](audits/v0.13g-audit-response.md).
Independent technical re-review: PENDING. Human presentation review: PENDING.
Protocol lock: NO GO. Live inference: NO GO. V0.13.0 release: NO GO.

**What was built:**

* Versioned final validation reuses the authoritative protocol batch decoder;
  known operations failures submit neither operations nor staffing.
* Exact submitted representations have their own execution identities and draw
  attribution. Canonical submitted actions remain the primary outcome.
* Initialization failures retain aborted skeletons. Stop reasons and events are
  checked in both directions, with explicit UNVERIFIED external evidence.
* Integral numeric spellings parse exactly; successor canonicalization preserves
  integer precision and normalizes engine-equivalent empty collections.
* Historical candidate CI runs in its original checkout. All current-tree checks,
  V0.12a/V0.11b comparisons, artifact upload and platform comparisons remain active.
* A successor complete human-review package includes corrected failure and numeric
  paths and real scripted partial-run examples. Human review remains PENDING.

**Preserved design:** nine historical contexts, six pairs, two arms, 262 draws,
802 maximum calls and the four-hour request-start guard. The intervention is
the complete interface; no claim of decision-quality improvement is made.
Predecessor evidence, contracts, historical agents, golden references and both
committed independent audits retain their bytes. Successor identities and pins
are under [v13g](v13g/CONTRACT-IDENTITIES.json).

**Baseline:** cf9a12bf12fc94089e8a99f73dea12edd0cfd019 on main.
The protected untracked swap file and dt are excluded from stages and archives.

**The engine version remains 0.13.0-dev.2.** Simulator mechanics, session,
accounting, scoring, action schema and preview behavior are unchanged. The
exported protocol decoder helper delegates to the unchanged original decoder;
existing callers and historical operations retain their behavior. Versioned
successor parsing, recording and verification receive separate identities.

**Quality gates:** final results are recorded in [the gate record](v13g/gates/README.md),
with full logs and before/after witnesses under ~/dt-audit-kept/v013g/.

**Not run or authorized:** independent re-review, human acceptance review,
live inference, clinical suites, historical live reruns, reserved/holdout
seeds, protocol lock, tag or release.

---

## V0.13f — independent audit remediation preparation (0.13.0-dev.2, engine unchanged)

The owner accepted DT13E-01 through DT13E-13 as the deterministic remediation
baseline. [Per-finding response](audits/v0.13f-audit-response.md).
Independent re-review: PENDING. Human presentation review: PENDING.
Protocol lock: NO GO. Live inference: NO GO. V0.13.0 remains unreleased.

**What was built:**

* Canonical submitted action identities are PRIMARY. Intended, validated,
  submitted and executed actions have separate fields. Five abandonments
  versus five no_action selections produce primary TV exactly zero; original
  response routes and seven F7 labels remain secondary.
* Side-by-side successors: shared phase-aware system instruction, workflow v2,
  driver v2, presentation v3 and view v2. The system schema takes precedence;
  compliant reason/actions final revisions, including empty ones, are accepted.
* Continuing employment obligations use a common no-action baseline and
  incremental differences, once per employee. Conditional severance is not
  payable-now severance. Exact unavailable guarantees remain unavailable.
* Closed slots, displaced bookings and replacement capacity retain units and
  horizons. The 2,481-slots/zero-bookings witness and 2,354-slot roster witness
  are reproduced. Withdrawal follows the engine notice rule; historical fee
  proxies retain study counts, history limits and lane context.
* Eligibility, exposure, correction failure, deliberate inaction, process
  failures, calls and retries are separate. Partial runs verify aborted draw
  identity, schedule/count conservation, call bounds, responses and stops.
  Reconstructible aborts replay; unsupported reconstruction is UNVERIFIED.
* Numeric citations use typed values and explicit unit separation. The
  estimand covers the complete interface bundle. Duplicate S1-a/S6-a inputs
  remain; small samples cannot establish stability. The four-hour mechanism
  is accurately a request-start guard.
* Complete scripted interactions are packaged for subsequent human review,
  with ACTIVE and LEGACY-INACTIVE sources and evaluation-only records kept
  outside model-visible messages. Reviewer fields remain PENDING.

**Preserved design:** nine historical contexts, six sensitivity pairs (all
12 members), two arms, 262 draws and 802 maximum model calls. The budget does
not change. No alternative is added or reordered. Historical evidence,
contracts, templates, workflows and agents remain byte-identical; the old
operations retain their behavior. The opt-in command registers additional
read-only successor operations.

**Baseline:** main e584107c1eacbcca10163aafbf8372f22745e0d8; v0.12.0 still
peels to 8bbeca29550da67c70d69efd313baadca6438244. The 1,173 scoped file hashes
are in v13f/BASELINE-HASHES.sha256. The protected untracked swap file and dt
are excluded from every stage and archive.

**The engine version remains 0.13.0-dev.2.** No simulator mechanics, session,
accounting, scoring, action schema, validation or preview behavior changes.
As in V0.13c–V0.13e, a version bump would alter embedded historical evidence.
The extension and presentation successors receive their own identities.

**Quality gates:** focused Go regressions and historical V0.13d evidence
reproduction pass locally; successor workflow/driver and baseline-byte checks
pass. Full Go, race, vet, format, five cross-builds, golden references,
benchmark verify/reproduce, historical replays, V0.12b analysis and both
fidelity paths are recorded in [the gate record](v13f/gates/README.md),
with retained logs under ~/dt-audit-kept/v013f/. All remote component gates
pass. The format walk was scoped to committed Go files after scratch caches
were found by the broad walk; no source formatting changed. The historical V0.13e CI
verifier's exact-HEAD-tree requirement conflicts with successor additions;
a narrow invocation exception is requested and not assumed.

**Not run or authorized:** independent re-review, human acceptance review,
live inference, clinical suites, historical live reruns, reserved/holdout
seeds, protocol lock, tag or release.

---

## V0.13e — rendered presentation review and candidate preparation (0.13.0-dev.2, engine unchanged)

The owner accepted V0.13d and approved F7, F3, N1, 262 scheduled draws,
802 max calls and a four-hour limit. Human presentation review remains the
substantive approval dependency. Live inference NO GO. V0.13.0 unreleased.

**What was built:**
* Self-contained [rendered review package](v13e/review-package/README.md):
  twelve members, both arms and every inspect detail, with record-only
  headers isolated. All text has SHA-256 identities; generation is
  reproducible and checked byte for byte in CI.
* Blank twelve-item questionnaire; HUMAN REVIEW PENDING. No reviewer,
  identity, findings, remediation or re-review exists.
* F7/F3 boundary tests supplement existing implementation. Submitted
  batches determine seven mutually exclusive categories; choosing,
  validation, execution and verified citations stay separate. Citations
  are secondary process measures; a changed action is not an improvement.
* Candidate file manifest and committed-tree verifier, run/partial-run
  procedures, limitations and read-only incident investigation.
* LOCK CANDIDATE - NOT LOCKED - awaiting human review.
  No protocol-lock commit. No live model contacted.

**Engineering observations:** the frozen draft's descriptive estimand field
still names process labels; the implementation and approved scale use actual
final actions. A stale frozen limitations paragraph also predates the N1
fix. Both are recorded in v13e/REVIEW_RESPONSE.md; neither was silently
changed. The six pairs compare contexts, not isolated single facts.

**The engine version string is unchanged at 0.13.0-dev.2.** Bumping it
would alter the embedded engine identity in V0.13b–V0.13d evidence.
All earlier evidence is byte-identical, verified against fb8cf3a by Git
trees, individual hashes and the candidate checker.

**Quality gates** (engineering checks only; local logs under
~/dt-audit-kept/v013e/):

| Where | Gate | Result / log |
|---|---|---|
| Pi | Alternatives, presentation, information boundary, F7/F3, selection | PASS; alternatives.log |
| Pi | Session branch tests | PASS; session-branch.log |
| Pi | Python driver tests (isolated mode) | 15 tests, OK, 5 engine tests deferred to CI; python-alternatives.log |
| Pi | Package byte reproduction | PASS; package.log |
| Pi | Committed candidate hashes, frozen trees | PASS, 1,964 hashes / 10 trees at first preparation commit; candidate-check.log |
| horo / Pi retained copy | Prior gate script | Remote path absent; local retained copy inspected and adapted; horo-gate-blocker.log / replay-gates.sh |
| horo | Dedicated race (90m timeout), five cross-builds, vet, format | Inspected replacement script launched on the preparation source tree; completion recorded in retained race.status / gates.status and race.log, cross-build.log, vet.log, format.log |
| horo | Golden, historical replay, benchmark verify/reproduce, frozen evidence, V0.12b check/analysis, V0.13d fidelity with historical replays, Python suites | Adapted retained script after race/build gate; logs/replay-gates.log and replay-gates.status |
| CI | Full Go suite, golden, historical replay, benchmark verify/reproduce, frozen evidence, V0.12b check/analysis, V0.13d fidelity with historical replays, package check, cross-platform comparison | Run by CI on this commit; CI job logs |

V0.13d committed fidelity reference: 65 cases, zero failed checks. Its
reproduction including historical replays is required in CI; the Pi's
pure tests alone do not establish that reproduction. The historical nine
contexts, six sensitivity pairs, two arms, 262 draws, 802 calls and
14,400-second budget are unchanged and verified by offline protocol/schedule
tests. The run parser refusal is unit-tested; its live implementation is absent.

**Not run:** live inference, clinical suites, historical live reruns, reserved
or holdout seeds. No protocol approval or human review was substituted.

**Not changed:** simulator, session, accounting, action schema, scoring,
validation, preview, pinned templates, view table, earlier workflows,
recordings, evidence or golden references. No tag or release.

Incident investigation: no matching deletion command in available histories,
no matching root-log writer/reference within the scoped scan, no matching
archive entry. The prior run directory is absent; losses remain unknown.
Dedicated run directories and deletion of only explicitly named temporary
artifacts are required. See docs/audits/v0.13e-operational-incident.md.

---

## V0.13d — controlled alternatives workflow and experiment preparation (0.13.0-dev.2, engine unchanged)

On 2026-11-05 the owner accepted V0.13c, with conditions, as a
protocol-design milestone. They gave GO for V0.13d preparation, with
decisions D7–D10, and did not approve the 964-draw budget. Protocol lock:
NO GO. Live inference: NO GO. V0.13.0 remains unreleased.

**What was built** (`docs/v13d/IMPLEMENTATION_REPORT.md`):
* **D7, the typed boundary.** `dreamteam.alternatives-view/v1`
  (`internal/alternatives/view.go`, table pinned `ce7f0e23…`) is built from
  the bound comparison:
  * neutral references (E, H, C, P), by proposal order;
  * closed tables of labels, forms and bases, parsed exactly and rebuilt;
  * no source text, no refusal codes;
  * anything unexpected fails closed (`ErrView`), and both arms then get
    `not_due`.
* **D8, presentation v2** (`presentation-v2.tmpl`, `257b4377…`). The head
  and foot are byte-identical in both arms, "Nothing has been applied yet"
  included. The treatment adds the comparison, the detail and one inspect
  round.
* **The workflow** `llm-agent-alternatives-workflow/v1`. It runs
  llm-agent 5's validation-first unchanged, then the engine's turn,
  through the opt-in protocol operations `alternatives.turn` and
  `alternatives.decide` (`dt agent --stdio -alternatives`). `hello` is
  unchanged.
* **D9, the outcome.** One final-action scale for both arms
  (`original_retained`, `listed_alternative`, `no_staffing_action`,
  `other_batch`, `abandoned`, `invalid`, `no_decision`), made from the
  submitted batches only. The arm's own labels are process measures.
* **D10, the operations batch.**
  * Select keeps the batch, unchanged; a revision replaces it; abandon
    submits nothing.
  * Each batch is validated alone with the protocol's batch validation.
  * Partial application is recorded.
* **The driver** `dreamteam-v13d-driver/1` (`analysis/alternatives/v13d.py`).
  * Both arms start from the same state, request and proposal.
  * Every draw replays without a model, with call metadata and halts
    rechecked.
  * Analysis keeps an execution ledger. A run that is not COMPLETE gives
    no arm differences.
  * `run` refuses anything not locked and authorized, and has no live
    implementation.
* **Fidelity** (`docs/v13d/FIDELITY.md`): 65 scripted cases,
  0 failed checks. They cover the 9 historical contexts (recorded V0.12b
  corrections as scripts), the 12 sensitivity members and the review's
  cases. No model was contacted.
* **The sensitivity pairs** are context pairs, with every difference
  counted (`docs/v13d/SENSITIVITY_CASES.md`).
* **The draft protocol** `v13d-alt-1` (`docs/v13d/PROTOCOL_DRAFT.md`):
  * 262 draws, at most 802 model calls, a 4-hour stop;
  * justified by the V0.10–V0.12 repeatability evidence;
  * DRAFT, NOT LOCKED, NOT AUTHORIZED.

  Hashes:
  * `protocol-draft.json`: `13498b55cffdfa35892cb114b2931195f2357cc7e2f8b689b113d64783836752`
  * `schedule-draft.json`: `81f40c75cec5a05a9e0546a301c8510729012bd0636d1b2982eaccfd334bd54b`
  * `contexts.jsonl`: `6af6cfd54fb4a9cf3e9eab5447611d7a001b7403627591cbacc6e2d5e2a340e3`
* **Human presentation review:** PENDING
  (`docs/v13d/PRESENTATION_REVIEW.md`). No reviewer exists, and none is
  invented.

**Independent technical review** (`docs/audits/v0.13d-independent-review.md`,
verbatim, `e40e6c0`): do not lock yet. It found 9 MAJOR, 3 MINOR and
2 NOTE.
* **Fixed:** F1–F12 and N1, plus one defect the preparer found (the
  `inspected` flag on a whole revision).
* **For the owner to confirm:** F7, the final-action scale; F3,
  `compared` (only details actually sent count); N1, the identical foot.

The response is in `docs/audits/v0.13d-review-response.md`.

**Mutation checks:** 23 on the fixed tree. 21 are caught; 2 are
equivalent (another check refuses the input).

**The engine version is not bumped.** A bump would change the engine
string in V0.13b and V0.13c's evidence, which must stay byte for byte.

**Quality gates:**

| Where | Gate | Result |
|---|---|---|
| horo (linux/amd64), reviewed tree `2f89a03` | Race, all packages, 90 min | exit 0; 11 of 11 packages with tests ok |
| horo, reviewed and fixed trees | Five cross-builds; vet (default, calibration, acceptance); gofmt | 5 of 5 ok; vet ok; gofmt clean |
| horo, reviewed and fixed trees | Golden references (four, through the migration manifest) | outcomes identical |
| horo, reviewed and fixed trees | Benchmark verify and reproduce | 86 and 59 checks, 0 failed |
| horo, reviewed and fixed trees | V0.8c and V0.9c replays | 4 × 104 checks, 0 failed |
| horo, reviewed and fixed trees | V0.13c inventory; V0.13b–V0.13d evidence, byte for byte | ok |
| horo, reviewed and fixed trees | V0.12b Stage B check, and the live run analysis replayed without a model | 24 point-arm starts at the recorded state, lock holds; decisions, tables and inference reproduced |
| horo, reviewed and fixed trees | V0.13d: driver tests, contexts, protocol and schedule made again, check, fidelity | 15 tests OK; contexts 21 identical; protocol and schedule identical; check 21 contexts, 0 failed; fidelity 65 cases, 0 failed (reviewed tree: 59, before the review cases) |
| horo, reviewed and fixed trees | Python: agents/llm, analysis/evidence, analysis/repeatability, analysis/acknowledgment, analysis/alternatives | OK |
| horo, fixed tree | Race, all packages, 90 min | exit 0; 11 of 11 packages with tests ok |
| Pi (linux/arm64), fixed tree | `internal/alternatives`, `internal/agent`, `cmd/dreamteam`; driver tests; constructed fidelity | ok; 15 OK; 47 cases, 0 failed |
| CI | Full Go suite, golden references, historical replay, benchmark verify and reproduce, frozen evidence, the V0.13c and V0.13d steps, on Linux amd64, Linux arm64 and macOS | run by CI on this commit |

**Not run:** live inference, the clinical suites, historical live reruns,
and any reserved or holdout seed.

**Not changed:**
* simulator mechanics, accounting, action schemas, scoring, validation,
  preview;
* the session API;
* every earlier agent, prompt, recording and frozen contract;
* V0.13b and V0.13c's evidence (`docs/audits/v0.13d-baseline.md`).

The agent protocol gains only the opt-in extension operations
(`docs/AGENT_PROTOCOL.md` §3.3).

---

## V0.13c — alternative comparison protocol preparation (0.13.0-dev.2, engine unchanged)

On 2026-11-05 the owner accepted and closed V0.13b, approved the narrow
F4 repair, and gave GO for V0.13c preparation only. Live inference: NO
GO. V0.13.0 remains unreleased.

**What was built** (`docs/v13c/IMPLEMENTATION_REPORT.md`):
* **F4.** `dreamteam.alternative-selection/v2` (pinned `5f32b2c8…`) is
  v1's list with later duplicate canonical batches removed and recorded.
  * It never substitutes, refills, repairs or reorders anything.
  * v1, its pin and all V0.13b evidence are unchanged.
* **The arms.** `alternatives.Steps(session, reply)` builds the control
  and treatment final-decision turns from one selection:
  * the turn is due only when the engine accepts the whole reply;
  * treatment shows only the bound agent comparison, through the pinned
    template, with a text check on every message and inspect answer;
  * `ClassifyControl` classifies control replies against the step's own
    proposal.
* **The branch-point inventory.** The V0.12b pool of 20 proposals was
  opened on the engine at the recorded states, from the V0.8c and V0.9c
  replays, and `dreamteam-v13c-select/1` was applied. Nine points are
  selected, and CI rebuilds both files.
* **Sensitivity pairs.** Six pairs of real engine states (mini, seed 52),
  with every rendered difference listed and every turn text committed.
* **Documents:**
  * the experimental design, with the owner's decisions D1–D10;
  * the presentation review package (PENDING);
  * the draft protocol (not locked);
  * limitations and the baseline.

**Independent technical review** (`docs/audits/v0.13c-independent-review.md`,
verbatim, `6cf962a`): do not lock yet. It found 1 BLOCKER, 2 MAJOR,
6 MINOR and 1 NOTE.
* **Fixed:**
  * F1 in part: the digest check is case-insensitive;
  * F2: the sensitivity ledger holds everything rendered;
  * F3: candidate identities are checked structurally;
  * F4: trailing JSON is refused;
  * F5: the point's read-only tail is checked;
  * F6: the turn is due only for a fully accepted reply;
  * F8: control classification is bound to its step;
  * F9: the budget is consistent.
* **Awaiting the owner:**
  * F1, the rest: meaning carried in free-text fields, under a
    trusted-input threat model (D7);
  * F7: the framing sentence (D8);
  * N1: the estimand, the operations batch and the shared outcomes
    (D9, D10).

**Mutation checks:** 26 of 26 caught. That is 17 for F4 and the arms
(15 caught first; the two weak tests were strengthened) and 9 for the
review fixes.

**The engine version is not bumped.** V0.13c changes no engine, session
or simulator file. A bump would change the engine string embedded in
V0.13b's evidence, which the owner asked to keep unchanged. This
departs from the usual per-step version bump.

**Quality gates:**

| Where | Gate | Result |
|---|---|---|
| horo (linux/amd64), reviewed tree `e8470c0` | Race, all packages, 90 min | exit 0; 11 of 11 packages with tests ok |
| horo, reviewed tree | Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64); vet (default, calibration, acceptance); gofmt | 5 of 5 ok; vet ok; gofmt clean |
| horo, reviewed tree | Golden references (four, through the migration manifest) | outcomes identical |
| horo, reviewed tree | Benchmark verify and reproduce (`testdata/benchmark/ci.json`) | 86 and 59 checks, 0 failed |
| horo, reviewed tree | V0.8c and V0.9c replays (historical replay, frozen evidence inputs) | 4 × 104 checks, 0 failed |
| horo, reviewed tree | V0.12b Stage B check, and the live run analysis replayed without a model | 24 point-arm starts at the recorded state, lock holds; decisions, tables and inference reproduced |
| horo, reviewed tree | Python: agents/llm, analysis/evidence, analysis/repeatability, analysis/acknowledgment | OK |
| horo, fixed tree | vet, gofmt, five cross-builds | ok; clean; 5 of 5 |
| horo, fixed tree | Race: `internal/alternatives` with the replay-backed inventory | ok |
| horo, fixed tree | Race: `internal/agent`, `internal/eval` | ok (exit 0) |
| horo, fixed tree | The inventory regenerated and compared | byte-identical to the reviewed tree's |
| Pi (linux/arm64), fixed tree | Every `internal/alternatives` test; `internal/agent` | ok |
| CI | Full Go suite, golden references, historical replay, benchmark verify and reproduce, frozen evidence, the V0.13c inventory step, on Linux amd64, Linux arm64 and macOS | run by CI on this commit |

**Not run:** live inference, the clinical suites, historical live reruns,
and any reserved or holdout seed.

**Not changed:**
* simulator mechanics, accounting, action schemas, scoring;
* the session API and agent protocol;
* every agent, prompt, recording and frozen contract;
* V0.13b's evidence, presentation template and rule v1
  (`docs/audits/v0.13c-baseline.md`).

---

## V0.13b — deterministic management alternatives (0.13.0-dev.2)

On 2026-11-04 the owner accepted and closed V0.13a and gave GO for
V0.13b:
* E1–E3 approved;
* E4 approved narrowly: exact scheduled capacity only, never future lost
  orders;
* live inference: NO GO.

V0.13.0 remains unreleased.

**What was built** (`docs/v13b/IMPLEMENTATION_REPORT.md`):
* **Branching (E1, E2).** `session.Branch` rebuilds every alternative
  from the same saved decision-point record:
  * a fresh replay, verified;
  * the alternative's batch at the decision instant;
  * no further action to a common horizon
    (`dreamteam.continuation/no-further-action-v1`).

  Every branch is its own replayable record with a
  `dreamteam.branch-identity/v1` identity. `VerifyBranch` and
  `VerifyBranchSet` check identities, records, the horizon rule and a
  fresh replay. Records under earlier rules or another state encoding
  are refused, anywhere in their prefix chain.
* **Slots closed (E4).** `PreviewSlotsClosed` is a separate, opt-in,
  read-only `dreamteam.slots-closed/v1` report for termination-only
  batches. It counts the template slots, booked or not, that staffing
  makes bookable now and the termination leaves unstaffed, from the end
  date to the last materialized day. It is capacity, not demand.
  `PreviewStaffing` is unchanged.
* **The comparison (E3).** `internal/alternatives` holds:
  * the candidate rule `dreamteam.alternative-selection/v1` (pinned;
    at most four candidates);
  * `dreamteam.management-alternatives/v1`, with an agent audience that
    cannot hold class B, decoded with refusal of unknown fields, of
    classes other than A1, A2 and C, and of sources that are not
    decision-point requests, and bound to the session by `VerifyAgent`;
  * an evaluation audience with the class B projections of verified
    branches and paired differences;
  * the presentation template `dreamteam.alternatives-presentation/v1`
    (pinned; summary, then detail on `inspect:<id>`);
  * the classification of scripted replies.
* **Evidence** (`docs/benchmark/v13b-dev/`): the termination case in both
  audiences and as presented; ten scripted decision cases; three
  decision-sensitivity pairs from real engine states. CI compares the
  files byte for byte.
* **Registration:** 0.13.0-dev.1 as batch mode `v0.13_dev1`, with a fixture
  made from a clean archive of `1c75c93`.

**Termination versus keep** (mini, seed 52, 2026-10-13 07:00, Sam
(E04), last day 2026-10-27).

The agent sees:
* notice of 14 days;
* severance of $3,950.00, recognized now and paid 2026-10-30;
* contract pay to the last day, $4,350.00;
* 0 displaced bookings;
* 2,481 slots no longer staffed to 2027-02-10, 0 of them booked now;
* the engine's estimate of $1,975.00 a week no longer owed.

For keeping, the agent sees:
* the weekly guarantee of $1,975.00, which continues;
* severance if terminated later;
* the employment stage, contracted and rostered minutes;
* payroll, payables and appointments booked ahead.

Evaluation only, to 2026-12-11:
* orders received were 2,881 in every branch;
* orders lost were 407 (proposal), 125 (no action) and 337 (adjusted).

**Tests:**
* 24 safety requirements are mapped to tests in the implementation
  report, §3.
* 22 mutations of the safety boundaries are all caught. The first pass
  caught 18, and the four tests that missed were strengthened.
* An accepted no-effect batch, and a refused batch, leave the ledger
  and every reported metric equal to no action. The V0.13a accrual
  question is closed for that case.

**Independent review** (`docs/audits/v0.13b-independent-review.md`,
verbatim, `c79f800`): no BLOCKER.
* **Fixed:**
  * F1: evidence regenerated after the version bump;
  * F2: the agent decoder refuses non-decision-point sources, and
    `DecodeAgentAt` binds;
  * F3: the whole prefix chain is checked;
  * F5: citation checks are in a fixed order.
* **Awaiting the owner:** F4, identical leave-one-out candidates. Fixing
  it changes the rule.
* **Recorded:** F6–F10 (`docs/audits/v0.13b-review-response.md`).

**A finding from the gates:**
* The race run found that the holdout containment test does not list
  `branchSource`, which calls the embedded manifest's guard.
* The call is of the same kind as the reviewed T90 and benchmark sites.
  It refuses a holdout seed before anything runs, and its error never
  names a seed.
* It is now listed in `holdoutReach`, with that reason.

**Quality gates:**

| Where | Gate | Result |
|---|---|---|
| horo (linux/amd64, Go 1.27.0), reviewed tree | Race, all packages, 90 min | 11 of 13 packages ok; `internal/alternatives` failed (F1, stale evidence) and `internal/eval` failed (`TestHoldoutContainment`, above); both fixed |
| horo, fixed tree | Race: `internal/alternatives`, `internal/session`, `internal/sim` | exit 0; 3 of 3 packages ok, no failure |
| Pi (linux/arm64) | `TestHoldoutContainment` after the fix; the V0.13b focused tests (`TestBranch*`, `TestSlotsClosed*`, `TestV0131*`, every `internal/alternatives` test) | ok |
| horo, both trees | Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64), vet, gofmt | 5 of 5 ok; vet ok; gofmt clean |
| horo | Frozen evidence (V0.8c credit and low liquidity, V0.9c credit and staffing shortage) | 4 × 104 checks, 0 failed |
| horo | Shared-prefix v4 points; V0.11b analysis replayed without a model (720 draws); V0.11a and V0.12a scripted evidence | verified; reproduced |
| horo | Python: agents/llm, analysis/evidence, analysis/repeatability | 156, 147 and 88 tests ok |
| horo, fixed tree | V0.13b evidence on amd64 (inside the race run of `internal/alternatives`) | byte-identical |
| CI | Full Go suite, golden references, historical replay, benchmark verify and reproduce, Linux, macOS and arm64 | run by CI on this commit |

**Not run:** live inference, the clinical suites, historical live reruns,
and any reserved or holdout seed.

**Not changed:**
* simulator mechanics, accounting, action schemas, scoring, the default
  preview;
* every historical recording, agent, prompt and frozen contract
  (`docs/audits/v0.13b-baseline.md`).

---

## V0.13a — engine-backed management alternatives: inventory and design (0.13.0-dev.1)

On 2026-11-04 the owner accepted and closed V0.12.0, and gave GO for
V0.13a.

**What V0.13a is:**
* a design and inventory milestone only;
* no agent workflow, no live model, and no change to simulator
  mechanics, financial accounting, action schemas, scoring or
  historical evidence.

V0.13.0 remains unreleased.

**The question:** can the existing engine represent and compare,
exactly, the alternatives to an irreversible or financially
consequential action before it is taken? Termination versus keeping
the employee came first.

**What was built:**
* **Documents:**
  * `docs/V0.13_MANAGEMENT_ALTERNATIVES.md`, the overview and the
    recommendation;
  * `docs/v13a/`, with nine documents:
    * the action inventory;
    * the termination-versus-keep case;
    * the comparison contract `dreamteam.management-alternatives/v1`
      (proposed);
    * branching feasibility;
    * the continuation design
      (`dreamteam.continuation/no-further-action-v1`, proposed);
    * alternative selection (`dreamteam.alternative-selection/v1`,
      proposed);
    * the agent presentation proposal;
    * the evaluation contract;
    * the known limitations.
* **The baseline:** `docs/audits/v0.13a-baseline.md`.
  * Hashes at `8bbeca2` for the simulator, action schemas, validation,
    preview, session API, accounting, randomness, the evaluation policy,
    llm-agent 5 and 6, the acknowledgment contract, the recordings, the
    frozen evidence and the goldens.
  * Every set from the V0.12.0 release baseline is unchanged.
* **The version:** `0.13.0-dev.1`. 0.12.0 is registered for replay
  (batch mode `v0.12.0`), with the fixture `mini-seed52-v0120`, made
  from a clean archive of `8bbeca2`, and
  `internal/session/v0120_test.go`.

**Findings:**
1. Every existing action has a meaningful no-action alternative. Every
   alternative proposed is an existing action with existing parameters.
2. A termination's direct terms are exact engine facts:
   * last day, end date and notice;
   * severance and payday;
   * contract pay to the last day;
   * displaced bookings;
   * voided items.
3. Keeping the employee is not free. The weekly guarantee and the
   severance exposure continue, and the engine has no resignation.
4. The preview does not report capacity lost by a termination-only
   batch. In the example, 0 bookings were displaced, while the 45-day
   projection lost 177 more orders.
5. Exact branching is possible by rebuilding from the record, not by
   copying in memory. A mechanics check saved, replayed and verified
   both branches, and reran one identically.
6. The engine's keyed randomness gives paired branches the same
   exogenous world: 2,224 orders in both.
7. Projections use the realized future. They are evaluation material,
   not information to show the agent.

**Design validation:**
* Read-only inspection of the engine.
* One decision-point probe (`validate`, `preview`, digests unchanged).
* One pair of branch runs on `mini`, seed 52 (a development seed).
* The session-registration tests and the golden tests.
* No model request, no reserved or holdout seed, no clinical suite.
* Cross-platform CI runs the full suite on the commit.

**Recommendation:** GO for a deterministic V0.13b, with no live
inference:
* E1–E3: the branch builder, identities and comparison reader;
* the contract;
* the candidate rule;
* the hashed template;
* paired scripted cases.

E4 (an opt-in `slotsClosed` for termination previews) needs an owner
decision.

**Stopped for owner review.**

---

## V0.12.0 — release: high-impact action acknowledgment and reproducible decision evidence (0.12.0)

The owner accepted and closed V0.12b Stage B on 2026-11-04 and gave GO
for the V0.12.0 release, a release-only milestone: no change to
simulator mechanics, agent workflows, acknowledgment rules, prompts,
models, scoring or experimental evidence.

**Released as** high-impact action acknowledgment and reproducible
decision evidence, **not** as an improved decision-quality system.
llm-agent 6 is an explicitly selected experimental agent; llm-agent 5's
validation-first workflow stays the default.

**What changed:**
* the version, `0.12.0`, and `testdata/golden/MIGRATIONS.json`;
* 0.12.0-dev.3 registered for replay (batch mode `v0.12_dev3`, every
  earlier mode kept), with the fixture `mini-seed52-v0123` made from a
  clean archive of 59676a3 and `internal/session/v0123_test.go`;
* documentation: README, RELEASE_NOTES, ROADMAP (V0.12 released, V0.13
  proposed, §7g), LLM_AGENT §21, BENCHMARK, V0.12_ACKNOWLEDGMENT (§10,
  the live evaluation) and the Stage B README's status;
* the release baseline, `docs/audits/v0.12.0-release-baseline.md`.

**Preserved byte for byte:** the simulator, llm-agent 1 to 6, every
prompt, workflow identity and contract, the historical recordings, the
frozen evidence, the golden references, the V0.12a scripted evidence and
the V0.12b locked protocol, run record and results (the lock's 48 pins
hold).

**Development findings, as released:** 420 completed draws, 524 model
calls, 12 branch points, two local models; 153 accurate
acknowledgments, 153 acknowledged and proceeded, 0 revised, 0 abandoned,
0 failures after retries, 77 draws needing none. Acknowledgment did not
change any high-impact action after it was shown; most revisions
followed validation refusals earlier; no decision-quality benefit was
established.

**Gates:** all passed; listed in `docs/audits/v0.12.0-release-baseline.md`. No live model was contacted,
and no clinical suite or historical live experiment was rerun.

---

## V0.12b — Stage B live run: COMPLETE (results; 0.12.0-dev.3, unchanged)

The owner authorized the single Stage B run of the locked protocol
`v12b-ack-1` (lock commit 5b8879f) on 2026-11-03. Nothing else was
authorized: no amendment, no tuning, no repeated or substituted draws.

**Before the run:**
* HEAD was the lock commit, and the engine was built from it
  (`0.12.0-dev.3`).
* All 48 pinned files, the schedule's entries and both settings files
  were checked independently, and `stage_b.py check` reported all 24
  point-arm starts at the recorded state and `lock: holds`.
* The server answered healthy and listed both protocol models with the
  entries V0.11b recorded. It reported oMLX 0.7.3 (V0.11b: 0.7.0); the
  protocol does not pin the version.
* The output directory was new, and no Stage B run record existed.

**The run** (`docs/benchmark/v12b-dev/live/`, committed by itself):
the exact command of `PROCEDURE.md` §2, with the owner's authorization
verbatim. **COMPLETE**: 420 of 420 draws, no stop; 524 model calls (limit
1,680), all HTTP 200; 3,633 s (limit 14,400 s); 30 server checks, all
healthy and unchanged.

**The analysis** (`decisions.json`, `decision-tables.md`,
`inference.json`, [RESULTS.md](benchmark/v12b-dev/RESULTS.md)):
* every draw, model call and engine request replayed without a model,
  identical; the same analysis on the Pi (arm64) and horo (amd64);
* arm B: 153 acknowledgment requests, all followed by an accurate
  acknowledgment and the same action (class A, "acknowledged and
  proceeded"); no revision, abandonment or withholding after a request;
  77 draws needed none (E). At one termination point, 16 of 20 first
  acknowledgment replies were malformed JSON and were accepted on the
  retry;
* arm A: of 170 shown previews, 166 proceeded, 3 revised and 1 failed;
* both arms' revisions and abandonments otherwise follow the engine's
  validation refusals, with very similar distributions; every B − A
  interval contains 0;
* the same high-impact actions were executed in both arms (terminations
  at three points, an unfunded hire, a $44,687.30 allocation
  commitment).

**Not claimed:** no financial or clinical improvement, no model ranking,
and acknowledgment is not read as approval.

**CI** now also replays the live run and compares its analysis on every
platform. The agents, the simulator, the scoring and the locked
documents are unchanged. The version is not bumped: the run's engine is
the locked `0.12.0-dev.3`.

---

## V0.12b — Stage B preparation: the driver, its review, and the protocol lock (LOCKED — NOT RUN; 0.12.0-dev.3)

The owner approved the Stage B design on 2026-11-03. That covers:
* the 12 branch points and `dreamteam-v12b-select/1`;
* 420 draws, at most 1,680 calls, sequential, a four-hour hard limit;
* the unavailable categories, as limitations.

Two requirements came with it:
* the arms must start identically, while their model-call counts stay
  visible;
* a run stopped by the limit must be reported as partial.

Live inference is not authorized.

**The driver** (`analysis/acknowledgment/stage_b.py`,
`docs/benchmark/v12b-dev/DRIVER.md`):
* **How a draw runs:** each draw calls the arm's agent's own `ask()`,
  llm-agent 5 with `validation-first-preview` or llm-agent 6. It uses:
  * the model's frozen settings file, loaded by the agent's own loader;
  * the agent's own `ModelClient` payloads;
  * the recording's memory text, its SHA-256 checked.

  The agent's first request must equal the recorded request byte for
  byte, and its reply is the recorded proposal. Nothing of either
  workflow is reimplemented.
* **Engine sessions:** one per point and arm, the arms never sharing a
  process. Each draw checks the digests at its start and records them at
  its end. Each arm reads the engine's lines with its agent's own reader.
* **Limits:** before every model request: 4 calls per draw, 1,680 per
  run, and the four-hour stop, under which a request starts only if its
  300 s timeout fits.
* **Stops:** a stopped run is PARTIAL, with no resume. A run also needs
  `--authorization`, the owner's text, recorded verbatim.
* **Replay:** every recorded draw is replayed without a model, with
  everything but send times and latencies identical.
* **Classification:** the outcomes, B's classes A–E with "accurate
  before a change", A's preview classes, and the engine-reported
  exposure of each distinct decision.

**Fidelity evidence:**
* **The dry start** (`stage_b.py check`): at all 24 point-arm
  combinations, the engine is at the recorded state, and the agent's
  first request equals the recorded request byte for byte. That checks
  the rebuilt prompt, memory and settings.
* **The real engine** (`docs/benchmark/v12b-dev/fidelity/`): 113
  scripted cases, each run through the driver and as the agent itself
  (its own `Harness` class on its own engine session). All 113 are equal
  in:
  * every model request, byte for byte;
  * every engine exchange;
  * the decision record and the batches.

  Every driver draw replays without a model. At the 6 points with a
  submitted decision, applying it through the agent's harness gives the
  same digests as the analysis continuation.
* **A defect the evidence caught:** the first run labelled unchanged
  proposals "revised". The Stage B contexts file stores actions with
  sorted keys, and llm-agent 4's rule compares compact JSON. The
  classification now compares against the agent's own parse of the
  recorded proposal, with a regression test.
* **The scripted engines** (`analysis/acknowledgment/test_stage_b.py`,
  39 tests at first, 47 after the review), with mock inference only:
  * every V0.12a scripted case and V0.11b's arm C cases, the driver
    against the agent's own `ask()`;
  * the owner's 25 scenarios;
  * the call bound: every sequence of three reply kinds at five starts
    gives at most 2 calls per draw.
* **The mock run** (`fidelity/mock-run/`): the real schedule, with
  scripted replies and a scripted clock. MOCKRESULT

**Independent review** (`docs/audits/v0.12b-stage-b-driver-review.md`,
verbatim at 1fb619c):
* **Verdict:** GO for locking the protocol, with conditions; no Blocker.
* **Clean areas:** fidelity, initial-state equality under pooled
  sessions (checked on the real engine against fresh sessions), arm
  isolation, ordering, the 2-call bound, limits and replay.
* **Findings:** D1 (Major) and D2–D13 (Minor and Note). Every one is
  answered in `docs/audits/v0.12b-stage-b-driver-review-response.md`:
  * D1: one rule for "unchanged", with reorders flagged;
  * D2: the final proposal's validity;
  * D3: wording;
  * D4: a reason for every class D draw;
  * D5: body-derived metadata checked in replay;
  * D6, D7: aborted draws recorded and counted;
  * D8: wording;
  * D9: the lock's contents and order;
  * D10: `check`'s exit code;
  * D11: `run.json` checked against the rows;
  * D12: the protocol field;
  * D13: recorded.
* **Tests:** 8 regression tests, each failing before the fix.

**Focused re-review** (`docs/audits/v0.12b-stage-b-driver-rereview.md`,
verbatim at 75038a3): GO for writing the lock; D1–D13 closed (D9 by the
lock). New: R1 (Major: key order inside an action counted as a change),
R2–R4 (Minor), R5 (Note), all fixed before the lock with 4 regression
tests; 51 tests in all.

**Version:** 0.12.0-dev.3. 0.12.0-dev.2 is registered as batch mode
`v0.12_dev2`, with the fixture `mini-seed52-v0122`.

**Known limitations:**
* **Session reuse:** sessions are reused per point and arm. A fresh
  engine per draw costs 5–36 s on the Pi and would not fit the four
  hours. The digests are checked at every draw.
* **Memory:** the memory text is the recording's replay's, checked by
  hash and by the rebuilt prompt.
* **Real-engine coverage:** the real-engine cases cover acknowledgment,
  revision, abandonment, refusal and failures. Correcting a refused
  proposal into a valid one is covered only on the scripted engines,
  since no correction is known in advance at the real points.

**The lock:** the protocol was locked in a commit of its own after the
review, the re-review and their fixes. Its status is `LOCKED — NOT RUN`.
* **Pinned:** 48 files (the driver, its fidelity code and tests; the
  agents, modules, points, recordings, settings and documents), plus the
  schedule's entries (`5f0918131b83…`), the driver (`d870fd39a69c…`), both
  agents' identities, the outcome definitions, the limits and the engine
  version.
* **Checked:** `stage_b.py check` on the locked tree reports `lock: holds`
  at all 24 point-arm starts. The mock run was regenerated for the locked
  protocol.
* **Not authorized:** the lock does not authorize live inference. Stage B
  awaits the owner's explicit authorization.

---

## V0.12b — Stage A completed (F1 repaired and re-reviewed); Stage B protocol proposed (0.12.0-dev.2)

The owner accepted Stage A on 2026-11-03 and made these decisions:
* **F1:** fail closed.
* **F2:** accept the current provenance (replay plus `model.jsonl`, no
  signing, no claim of authenticated model origin).
* **Independence:** the separate reviewer instance is accepted for this
  engineering gate. The 98-item human evidence review stays PENDING.
* **Live inference:** not authorized.

**F1 repair** (2c73c3b):
* **Requirement:** llm-agent 6 requires every preview field that a
  consequence is read from or identified by to be present, not `null`,
  and of the engine's type: an integer ≥ 0 (not a boolean or float), a
  real date, or non-empty text.
* **Consistency:** each termination's batch count must equal the
  displacement list, no entry may be left that the batch does not
  account for, and no key may be given twice inside the preview.
* **Reader:** llm-agent 6's own reader marks duplicate keys and refuses
  `NaN` and `Infinity`.
* **On failure:** the proposal is withheld as `consequences_unavailable`
  before any acknowledgment request.
* **Unchanged:** the contract, matching, token and identities. The
  V0.12a evidence regenerates byte-identical, and all 13 recorded real
  previews read cleanly.
* **Fields:** `docs/V0.12_CONSEQUENCE_INVENTORY.md` §4.

**Tests:**
* `agents/llm/test_llm_agent6_fields.py`: 18 cases from real recorded
  engine answers. Before the repair, 15 failed (confirmed by the
  reviewer's reconstruction); after it, all pass.
* `TestLLMAgent6Hire`: a real-engine hire, on two seeds.
  * The fee is 300000 and the earliest exit 745706, both the engine's
    own.
  * An empty or wrong acknowledgment is withheld, and the session ends as
    the no-op run's.
  * The accurate acknowledgment is submitted and applied as H00001.
  * The replay is byte-identical.

**Version:** 0.12.0-dev.2. 0.12.0-dev.1 is registered as batch mode
`v0.12_dev1`, with the fixture `mini-seed52-v0121`.

**Focused re-review** (`docs/audits/v0.12b-f1-rereview.md`, verbatim at
aa22530):
* F1 CLOSED; no blocking defect; GO for protocol approval.
* R1, Minor: the duplicate-key check covers objects inside `preview`,
  not a duplicated `preview` key. The docs are narrowed.
* R2, Minor: the maintainer's before-fix log came from a test draft; the
  reviewer's log stands.
* The agent's code is unchanged from the reviewed revision.

**Stage B, proposed and not run** (`docs/benchmark/v12b-dev/PROTOCOL.md`):
* **Arms:** A, llm-agent 5 validation-first-preview; B, llm-agent 6.
* **Branch points:** 12, from the 20 pooled proposals that are eligible
  under shared-prefix/v4 and already have committed contexts. Selected by
  `dreamteam-v12b-select/1`, which reads no response.
* **Unavailable:** no valid hire, and no termination without
  displacement, exist in the corpus.
* **Size:** 420 draws (Gemma 10, Qwen 20 per arm per point), at most
  1,680 calls, a 4-hour stop.
* **Central classes:** acknowledged and proceeded / revised / abandoned;
  failed acknowledgment, withheld; no acknowledgment required.
* **Statistics:** per point, Wilson and Newcombe intervals, with no
  composite score.
* **Before any request:** the driver and its fidelity tests, then a
  freezing commit.
* **Files:** the draft protocol `protocol.json`, its points and schedule,
  with hashes in `docs/benchmark/v12b-dev/README.md`. CI checks them.

**Quality gates:**

| Where | Gate | Result |
|---|---|---|
| horo (linux/amd64, Go 1.27.0), 2c73c3b | Race, all packages, 90 min | exit 0, 10 packages ok |
| horo | vet, gofmt | ok, clean |
| horo | Five cross-builds | 5 of 5 ok |
| horo | Frozen evidence | 4 × 104 checks, 0 failed |
| horo | V0.11b | 720 draws replayed, all files reproduced |
| horo | V0.11a and V0.12a scripted evidence | reproduced |
| horo | Python suites | 156, 147, 88 tests ok |
| Pi | Real-engine llm-agent 6 tests | `TestLLM*`, displacement and hire pass |
| Pi | Session and golden tests | `TestV0121*`, `TestV0110*`, `cmd/dreamteam` golden, `internal/agent`, `internal/tui` pass |
| CI on 2c73c3b | Full suite, golden, historical replay, benchmark verify/reproduce, cross-platform | green (41645751085) |

* **Not repeated locally:** the follow-up documentation and protocol
  commit, which changes no code; CI covers it.
* **Not run:** live inference, the clinical suites, and historical live
  experiments.
* **Still PENDING:** the separate frozen 98-item human evidence review.

---

## V0.12b Stage A — independent safeguard audit and real-engine displacement (0.12.0-dev.1)

The owner accepted V0.12a (9871e37, CI 41564875041) on 2026-11-02 and
authorized V0.12b in two stages. Stage A comes first: an independent
audit and a real-engine displacement test. The live evaluation (Stage B)
needs separate approval of a frozen protocol. Multi-day LLM evaluation
is NO GO. V0.12.0 is not released.

**Baseline** (`docs/audits/v0.12b-stage-a-baseline.md`):
* main was clean at 9871e37;
* `v0.11.0` is unchanged (tag object d84e09a, commit a267cd3);
* hashes are recorded for the simulator, llm-agent 1–6, the
  acknowledgment contract and workflow identities, the record schemas,
  the V0.12a evidence, the historical recordings, the frozen evidence
  contracts and the golden references.

**Real-engine displacement** (82cea3f,
`internal/bench/llm_agent6_displacement_test.go`,
`docs/benchmark/v12b-dev/DISPLACEMENT.md`):
* **Setup:** the mini scenario on development seeds 739182645 and 5. At
  the third decision point, terminations displace 7 to 22 bookings. Five
  mock-model modes, ten runs.
* **Exact count:** the batch item states the engine's exact count, over
  every action. Each termination is its own item.
* **Missing or wrong count:** leaving out the displacement
  acknowledgment, or stating one fewer, gets 3 of 3 rejections
  (`missing_item`, `wrong_value`). Nothing is submitted, and the session
  ends byte-identical to the no-op run.
* **Changed proposal:** a proposal changed after acknowledgment (10
  displaced, then 7) is submitted only on its own acknowledgment.
* **Final check and apply:** the final check's hashes equal the
  acknowledged ones. Applying displaces exactly the acknowledged count.
* **Replay:** byte-identical, with no model call.
* **Sensitivity check:** making the agent ignore the displaced count
  makes the test fail.

**Independent audit** (`docs/audits/v0.12b-safeguard-audit.md`,
committed verbatim at ca6b705):
* **How it was done:** a separate reviewer worked on a read-only snapshot
  of 82cea3f, with witnesses built in a work copy (82 scripted cases and
  a real-engine witness test). The snapshot was verified before and
  after.
* **Verdicts:** GO for the safeguard; GO for Stage B, with two
  conditions on how its evidence is described.
* **No Blocking finding.**
* **F1, Major:** the engine's preview fields are trusted for presence
  and type. A field missing from the engine's answer would be listed as
  `null` and could be acknowledged. The engine at this revision always
  sends these fields.
* **F2, Major:** replay certifies consistency, not authorship. A record
  regenerated consistently from hand-written replies replays as
  accepted.
* **F3, Minor:** stale documentation.
* **F4, Minor:** thin coverage of hires on the real engine.

**Response** (`docs/audits/v0.12b-safeguard-audit-response.md`):
* **Code:** none changed (fix only blockers). llm-agent 6, its
  identities, the V0.12a evidence and the simulator are unchanged.
* **Specification:** `docs/V0.12_ACKNOWLEDGMENT.md` is corrected:
  * §2: the engine-contract reliance (F1);
  * §7: what replay proves (F2);
  * §8: the recorded answers reach 100 displaced bookings, not 8;
  * §9: the limitations.
* **For the owner:** two decisions before Stage B. Should F1's hardening
  be applied (which would need a focused re-review)? Is replay, as now
  documented, enough for Stage B evidence (F2)?

**Version:** still 0.12.0-dev.1. Stage A adds only tests and documents;
the engine and the agents did not change.

**Quality gates:**
* the displacement test, run locally on the Pi and again by the reviewer;
* all `TestLLM*` bench tests on the Pi, which pass;
* the reviewer's Python agent suites and historical-replay Go tests,
  which pass;
* CI on 82cea3f, which also covers the other gates the milestone lists.

No race, cross-build or full-suite run was repeated locally for the
documentation-only follow-up commit, because CI covers them. No live
inference, clinical suite or historical live experiment was run. The
separate frozen 98-item evidence review is still PENDING.

---

## V0.12a — high-impact action acknowledgment (0.12.0-dev.1)

V0.11.0 was accepted and closed (a267cd3, tag `v0.11.0`, CI 41534888421),
and V0.12a was authorized with scripted models only. This was an
implementation and deterministic-testing milestone: no live model was
run, and the engine is unchanged.

**Built:**
* **The consequence inventory**
  (`docs/V0.12_CONSEQUENCE_INVENTORY.md`), written first. For every
  consequence, it records what the engine's validate and preview
  requests report exactly (A), what can be derived without changing
  engine semantics (B), and what is unavailable (C).
  * The three conditions are identifiable:
    * termination (A);
    * displaced bookings (A), as a whole-batch count;
    * new retained obligations: a hire's fee and earliest exit (B, from
      the engine's documented line order, label-checked), or a
      teleradiology start's guaranteed retainer (A).
  * Allocation changes (withdrawable), roster changes that displace
    nothing, teleradiology cancellations and operations actions are not
    high-impact.
* **The contract** `dreamteam.action-acknowledgment/v1` (`33c116da979a…`).
  * Items carry exact engine values, with their sources.
  * A payload hash (canonical JSON) and a proposal token.
  * Strict matching: the action, type, target, consequence types and
    every value must be exact, with the same JSON type. Generic,
    malformed, missing, incomplete, other-action, other-proposal (stale)
    and changed-proposal acknowledgments are not accepted.
* **llm-agent 6** (`agents/llm/llm_agent6.py`,
  `llm-agent-acknowledgment-workflow/v1`, `b9cb79828f26…`). It loads
  llm-agent 5 unchanged. The steps are:
  1. propose;
  2. validate (version 5's feedback);
  3. read the consequences from the staffing batch's preview;
  4. request acknowledgment;
  5. match;
  6. run the final check (validate again, preview again, digests);
  7. submit.

  It fails closed: anything not acknowledged is withheld, with the
  reason, and the proposal's operations batch is not submitted on its
  own. There is no threshold, score or recommendation.
* **Record** `dreamteam.llm-decision-record/v6`. It holds the proposal,
  the validation requests and answers, the consequence source and values,
  the payload hash, the request (by hash), the model's acknowledgment
  (its full reply), the parsed acknowledgment, the match, the final
  check, the withheld batches, the submitted batches, the digests, and
  the stages: proposed, validated, previewed, acknowledgment requested,
  acknowledged, submitted (executed at the next point).
* **Replay** re-runs the workflow against the engine with the recorded
  replies, without a model, and requires the identical record. Any
  tampered field is named and refused.
* **Docs:**
  * `docs/V0.12_ACKNOWLEDGMENT.md`;
  * `docs/benchmark/v12a-dev/` (baseline, changes, scripted evidence,
    gates);
  * LLM_AGENT §21, BENCHMARK §11f, ROADMAP §7f (the V0.12 table: V0.12a
    done; V0.12b and V0.12c proposed; V0.12.0 pending), README.

**Tested:**
* `agents/llm/test_llm_agent6.py` (31). Tests 01–29 are the 29 named
  behaviours, from an ordinary action to cross-platform determinism. The
  suite also reads items from 13 distinct real engine preview answers
  recorded in V0.11b (6 terminations, 5 displacement items, 4
  teleradiology starts, 1 hire), and pins the identities. Every scripted
  case also replays without the model.
* `internal/bench/llm_agent6_test.go`: the real engine and the mock
  model, two seeds:
  * a termination and a teleradiology start, acknowledged and applied;
  * several terminations in one batch;
  * a generic "I understand." that withholds the termination;
  * every item checked against the engine's answer;
  * a byte-identical model-free replay;
  * tampered value, acknowledgment, action and v5 records refused.
* `docs/benchmark/v12a-dev/scripted-cases.{json,md}`: 27 cases, made by
  `analysis/acknowledgment/scripted_cases.py` and compared by CI across
  platforms.

**Quality gates:**
Run locally on horo (linux/amd64, Go 1.27.0) from the working tree:

| Gate | Result |
|---|---|
| Race, all packages (`go test -race -timeout 90m ./...`) | exit 0; 10 packages ok, no failures |
| `go vet ./...` | ok |
| `gofmt -l` | clean |
| Cross-builds: linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64 | 5 of 5 ok |
| Frozen evidence checks: v08c credit/low-liquidity, v09c credit/staffing-shortage | 4 × 104 checks, 0 failed |
| Shared-prefix v4 points | 14 contexts, selection and requests verified |
| V0.11b workflow analysis | 720 draws replayed without a model; decisions.json, decision-tables.md, inference.json reproduced |
| V0.11a scripted evidence (20 cases) | scripted-cases.json and .md reproduced |
| V0.12a scripted evidence (27 cases) | scripted-cases.json and .md reproduced |
| Python: agents/llm | 138 tests ok (31 for llm-agent 6) |
| Python: analysis/evidence | 147 tests ok |
| Python: analysis/repeatability | 88 tests ok |

On the Pi: `TestLLMAgent6Acknowledgment` (real engine) and `TestV0110Record*` pass.

The CI run on the commit covers the rest, so these were not repeated locally: the full non-race Go suite, golden references, historical replay, benchmark verify/reproduce, the V0.12a evidence regeneration, and the Linux/macOS/Windows matrix.

Not run: live inference, the clinical suites, and historical live reruns. The owner's instructions exclude all three.

**Not changed:**
* the simulator, actions, accounting or scoring;
* llm-agent 1–5, their workflow identities and hashes, or the frozen
  llm-agent 3 settings schema;
* any recording or evidence.

No live inference, clinical suite or historical live experiment was run.
The independent review is still PENDING.

**Known limitations:**
* scripted models only;
* restating exact values is not understanding;
* displacement is a batch fact;
* the hire amounts are class B;
* the mini scenario displaces no bookings, so real-engine displacement
  is covered by the recorded V0.11b answers;
* direct consequences only;
* no cross-batch atomicity.

The details are in `docs/V0.12_ACKNOWLEDGMENT.md` §9.

**Engine:** unchanged. 0.11.0 is registered for replay
(`mini-seed52-v0110`, batch mode `v0.11.0`).

---

## V0.11.0 — release: the validation-first agent workflow (0.11.0)

V0.11b was accepted (26005b3, CI 41498797098), with a conditional GO for
V0.11.0 from the owner. The default is validation-first without the
preview (B), and the preview (C) is opt-in. The owner authorized one
bounded change for the release (Option A): a workflow selector for
llm-agent 5.

**The selector** (b44384a, a separate commit before the release):
* `llm_agent5.py --workflow validation-first | validation-first-preview`.
  When omitted, the workflow is validation-first.
* **Identities:**
  * `llm-agent-workflow/v1` (`3ee7d66f…`) is unchanged byte for byte. It
    is now the opt-in `validation-first-preview`.
  * `llm-agent-validation-first-workflow/v1` (`a0204f62…`) is new: the
    version, the unchanged feedback text and the no-preview rule.
* **The only behavioural difference** between the two is whether the
  selective preview step occurs. Validation, refusal feedback, retries
  and submission are shared.
* **Recording:** every live record and `agent.json` name the workflow.
* **Replay** takes the workflow from the recording. A record without a
  selection must be `llm-agent-workflow/v1` and replays with the preview.
  Unknown identities, contradictory selections, mixed recordings and a
  contradicting `--workflow` are refused.
* `ask()` keeps its V0.11a default for callers that name no workflow.
  So the V0.11a scripted evidence and the V0.11b fidelity tests are
  unchanged, and CI regenerates the evidence equal.
* **Tests:**
  * `agents/llm/test_llm_agent5_workflows.py` (18);
  * `analysis/repeatability/test_workflow_selector.py` (3; B and C equal
    to the V0.11b drivers);
  * `internal/bench/llm_agent5_test.go`: the existing tests name the
    preview workflow, and `TestLLMAgent5ValidationFirst` covers the
    default and explicit validation-first against the real engine, with
    replay.

**The release:**
* Version 0.11.0.
* 0.11.0-dev.2 registered for replay: batch mode `v0.11_dev2`, the
  `mini-seed52-v0112` fixture made from a clean b44384a worktree, and
  `internal/session/v0112_test.go`. Every earlier batch mode is kept.
* Release notes, README, ROADMAP (V0.11 released; V0.12 proposed, not
  approved), LLM_AGENT §18–20, BENCHMARK §11e, FINANCIAL_EVIDENCE §10,
  V0.11_AGENT_WORKFLOW §10, and the release baseline
  (`docs/audits/v0.11.0-release-baseline.md`).
* The safety disclosure: validation success does not establish decision
  quality. It covers the V0.11b terminations (100, 8 and 6 displaced
  bookings), retained obligations, funding approval, and the preview not
  being a recommendation.

**Quality gates:**
| Gate | Where | Result |
|---|---|---|
| Python tests: agents (`agents/llm`, including the 18 new workflow tests), evidence (`analysis/evidence`), repeatability (`analysis/repeatability`, including the 3 new selector tests) | horo and CI | 107, 147 and 88 tests, OK |
| Go suite (`go test ./...`, including `TestLLMAgent5ValidationFirst` and the new `v0112` session tests), golden references, seed guards (`internal/eval`), historical session replay | CI, three platforms; race run on horo | passed |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64 | exit 0; 10 packages ok, no failure |
| `go vet ./...`, `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64) | horo | all ok |
| Historical model-free replays (V0.8c financial-pressure credit and low-liquidity; V0.9c financial-pressure and staffing-shortage credit) with the 0.11.0 binary | horo | 4 × 104 checks, 0 failed |
| V0.11a scripted evidence regenerated after the selector | horo | `scripted-cases.json` and `.md` byte-identical |
| V0.11b analysis reproduced (720 draws, 572 model calls, 2,299 engine requests replayed) | horo | `decisions.json`, `decision-tables.md`, `inference.json` byte-identical |
| Benchmark verify and reproduce, benchmark fingerprints, frozen-evidence checks | CI | passed on the release commit |
| Cross-platform comparison (linux/amd64, linux/arm64, darwin/arm64) | CI | passed on the release commit |
| Selector commit `b44384a` | CI run 41522392152, three platforms | passed |

Not run locally (covered by CI): the full Go suite, golden references and
benchmark reproduction. Not run at all: live inference, the clinical
suite, and historical live experiments.

**No change to:** prompts, simulator mechanics, action schemas,
financial accounting, scoring, llm-agent 1–4, the settings schema, or
any recording or evidence. No live inference was rerun, and no clinical
suite was run.

---

## V0.11b — controlled evaluation of the validation-first workflow (0.11.0-dev.2): results

The experiment declared at 562ed93 (below), run and analysed. Report:
`docs/benchmark/v11b-dev/README.md`.

**Live run:** 2026-11-02, 15:26–16:04 UTC.
* 720 of 720 draws and 572 model calls, all HTTP 200; no stop.
* 30 server checks, all the same configuration.
* Model calls by arm: A 130, B 130, C 312.

**Results** (per point; Gemma of 10, Qwen of 20):
* **C − B:** the same decisions at 10 of 14 points.
  * At valid proposals, C kept 4 of 6, including the $44,687.30
    credit-supported allocation, 20/20.
  * At ss-qwen-s42002-p7, C dropped the teleradiology contract (16/20
    revised, 4/20 abandoned; $3,228.33 to $135.00–$303.75).
  * At fp-qwen-s41901-p26, C dropped a no-op allocation (20/20, the
    commitment unchanged).
  * At refused proposals, C and B are within 5 pp.
* **B − A:** valid corrections where llm-agent 3 made none:
  * fl-qwen-s41902-p27: 20/20 against 0;
  * fl-qwen-s41901-p9: 13 against 0;
  * fp-qwen-s42001-p24: 5 against 0.

  The first two terminate a technologist (100 and 8 bookings displaced).
  No difference at the valid points (both submit) or for Gemma.
* **Gemma:** identical decisions under A, B and C at all four points.
* **Credit-supported submissions:** 30 per arm.

**Replay:** all 720 draws were run again without a model (572 responses,
2,299 engine requests, all equal). The analysis is byte-identical on the
Pi, horo and CI.

**Quality gates:**
| Gate | Where | Result |
|---|---|---|
| Focused tests: `analysis/repeatability` (11 new for v4) | Pi, horo and CI | 85 tests, OK |
| Agent tests (`agents/llm`), evidence tests (`analysis/evidence`) | horo and CI | 89 and 147 tests, OK |
| Go suite (`go test ./...`), golden references | CI, three platforms | run by CI on this commit; the new `v0111` session tests also on the Pi |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64 | exit 0; 10 packages ok, no failure |
| `go vet ./...` (default, calibration, acceptance tags), `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64) | horo | all ok |
| Historical replays (V0.8c, V0.9c) | horo and CI | 104 checks, 0 failed, each of 4 replays |
| Benchmark verify and reproduce | CI | run by CI on this commit |
| Branch points made again (selection, engine answers, requests) | horo and CI | 14 contexts verified |
| Every draw replayed without a model, analysis regenerated and compared | Pi, horo (three times) and CI | 720 draws, 572 responses, 2,299 engine requests equal; byte-identical (see the note) |
| Cross-platform CI (linux/amd64, linux/arm64, darwin/arm64) | CI | run by CI on this commit; declaration commit 562ed93 green (run 41471597358) |

**A cross-version fix found by the gates.** The first horo comparison
found one difference in `decisions.json`: a pooled mean of exactly
0.21875 was rounded to 0.2187 on the Pi (Python 3.11) and to 0.2188 on
horo (Python 3.13). Python 3.12 made `sum()` of floats compensated, so
the computed mean differed in its last bit. The pooled means now use
`math.fsum`, which is correctly rounded on every version. Only that one
value changed (to 0.2188), and no per-point result. The fixed analysis
was rerun on horo and equals the committed file. The V0.10b and V0.10c
analyzers, frozen, use the same `sum()`. Their committed outputs are
unaffected, as CI's comparisons show.

Not run: no clinical suite, no historical live evaluation rerun, and no
local repeat of the gates CI runs (Go suite, golden references, verify
and reproduce).

**No change to:** the simulator, llm-agent 1–5, prompts, settings,
scoring, the frozen evidence rules or any recording. No historical live
evaluation was rerun, and no clinical suite was run.

**Recommendation:** prepare V0.11.0 after owner review. Choose B (fewer
calls, the same measured validation benefit) or C (it can review an
expensive valid proposal) as the default.

---

## V0.11b — controlled evaluation of the validation-first workflow (0.11.0-dev.2): declaration

V0.11a was accepted (e6bb386, CI 41448551855), and V0.11b authorized by
the owner. This commit declares the experiment before any live request.
Report: `docs/benchmark/v11b-dev/README.md`.

**Protocol** (`v11b-workflow-1`, `dreamteam.shared-prefix/v4`; `PLAN.md`,
`SPEC.md`, `protocol.json`, `points/`, `schedule.json`):
* **Arms:** three whole workflows from one recorded state and proposal:
  * A, llm-agent 3;
  * B, llm-agent 5 with the preview disabled (the experiment
    configuration `v11b-B-validation-only`);
  * C, llm-agent 5.

  They run with the frozen `maxRetries` 1 and details turn. Model calls
  differ by workflow and are counted.
* **Points:** 14 from the V0.10c inventory.
  * Gemma: all 4 of its eligible contexts.
  * Qwen, 5 valid: credit-supported, retained obligation, termination,
    allocation, then stake.
  * Qwen, 5 refused: insufficient funds 2, timing 2, other 1.
  * Two candidates with an earlier refusal at the point are excluded.
  * Prior exposure is recorded: 9 points were in V0.10b or V0.10c.
* **Draws:** Gemma 10 and Qwen 20 per arm and point; 720 draws, one model
  request at a time, in hash-ordered rounds.
* **Analysis:**
  * primary C − B and secondary B − A, per point;
  * financial evidence v1 and the preview-nudge tables;
  * a fixed no-action continuation.

**Code:** `analysis/repeatability/workflow_experiment.py`. Its arm driver
follows each agent's loop with the agents' own functions. It also does
the selection, the schedule, the sequential live client (one engine
session per point), and the analysis, which replays every draw's workflow
without a model. `test_workflow_experiment.py` (11 tests) checks that A
sends llm-agent 3's messages and submits its batches, B llm-agent 5's
without a preview, and C llm-agent 5's.

**Engine:** unchanged. 0.11.0-dev.1 is registered for replay
(`mini-seed52-v0111`, batch mode `v0.11_dev1`).

---

## V0.11a — validation-first agent decision workflow (0.11.0-dev.1)

V0.10.0 was accepted, and the owner authorized V0.11a: implementation and
deterministic testing, with no live inference. Report:
`docs/benchmark/v11a-dev/README.md`; design:
`docs/V0.11_AGENT_WORKFLOW.md`.

**llm-agent 5** (`agents/llm/llm_agent5.py`, `llm-agent-workflow/v1`)
loads llm-agent 3 and 4 unchanged. It keeps their parser, batching,
prompt, memory, preview request and replay infrastructure.
* **Validate first:** every batch of every parsed proposal.
* **Refused:** the model gets the engine's results for the refused
  actions, verbatim (`llm-agent-validation-feedback/v1`). It revises or
  abandons within `maxRetries`; a refusal never becomes an accepted
  action.
* **Valid with staffing:** one read-only preview per decision point,
  shown only when the engine's answer shows a direct effect
  (`llm-agent-preview-eligibility/v1`).
  * The reasons: a measured commitment, a new guaranteed obligation, a
    termination, displaced bookings or a capacity change.
  * There is no dollar threshold. Nothing in the fields justifies one:
    17 of the 19 recorded valid staffing previews qualify, and the other
    2 change nothing.
* **The preview message** (`llm-agent-selective-preview/v1`) says the
  proposal passed validation and that the preview gives direct
  commitments and effects. It says neither establishes the decision is
  beneficial, and offers keep, revise or abandon. The reply is validated
  again.
* **Submit only validated batches,** after checking that the digests are
  unchanged.
* **Mixed batches:** operations and staffing batches stay separate and are
  validated independently, with no cross-batch atomicity. The next
  point's action log records each batch as applied or not; a refused
  batch's results are in the harness's record.
* **The record** (`dreamteam.llm-decision-record/v5`) keeps proposed,
  validated, previewed, submitted and executed apart, with every input,
  answer and message by SHA-256. Replay checks it all without a model.

**Tests** (scripted models only):
* `test_llm_agent5.py`, 29 tests (the agents' suite now 89);
* `internal/bench/llm_agent5_test.go`, the real engine with the mock
  model, live and replayed byte for byte, tampering refused;
* the scripted-case evidence (20 cases), regenerated by CI on every
  platform.

**Independent review:** `docs/REVIEWER_HANDOFF.md` added; the package is
unchanged. **INDEPENDENT REVIEW: PENDING.**

**Engine:** unchanged. 0.10.0 is registered for replay
(`mini-seed52-v0100`, batch mode `v0.10.0`).

**Quality gates:**
| Gate | Where | Result |
|---|---|---|
| Focused workflow tests (`agents/llm/test_llm_agent5.py`, 29) and the existing agent tests | Pi, horo and CI | `agents/llm`: 89 tests, OK |
| Engine and mock-model tests of llm-agent 3, 4 and 5 (`internal/bench`, `TestLLMAgent*`) | Pi, horo, CI | ok |
| Evidence tests (`analysis/evidence`), repeatability tests (`analysis/repeatability`) | horo and CI | 147 and 74 tests, OK |
| Go suite (`go test ./...`), golden references, historical session replay (new `v0100` tests included), seed guards | CI, three platforms; race run on horo | passed |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64 | exit 0; 10 packages ok, no failure |
| `go vet ./...` (default, calibration, acceptance tags), `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64) | horo | all ok |
| Historical replays, benchmark verify and reproduce, V0.10 analyses | CI | run by CI on this commit |
| Scripted-case evidence regenerated | Pi, horo and CI | byte-identical |
| Cross-platform comparison (linux/amd64, linux/arm64, darwin/arm64) | CI | run by CI on this commit |

Not run: no live inference, no clinical suite, no historical live
evaluation.

**Not done:** no live inference, no clinical suite, no historical live
evaluation.

---

## V0.10.0 — release: inference repeatability and controlled decision experiments

V0.10c was accepted (679483e, 222168e; CI 41378207630), and the owner
gave GO for the V0.10.0 release. This is release-only: no change to
mechanics, actions, scoring, scenarios, agents, prompts, models or any
experimental result.

* **Version** 0.10.0.
* **Replay:** 0.10.0-dev.3 (V0.10c) is registered for historical replay
  in batch mode `v0.10_dev3`, with the `mini-seed52-v0103` fixture (made
  from a clean 222168e worktree) and its tests. Every earlier batch mode
  is kept.
* **Release notes** (`docs/RELEASE_NOTES.md`): the sixteen capabilities;
  the V0.10a, V0.10b and V0.10c findings, financial findings and
  limitations. V0.10c is not an untouched confirmatory holdout.
* **Documentation:**
  * README, ROADMAP (§7d released; §7e V0.11 proposed, not approved);
  * the status lines of BENCHMARK and LLM_AGENT.
* **Release baseline:** `docs/audits/v0.10.0-release-baseline.md`. Every
  set is unchanged except the session registration and two added
  fixture files.
* **Preserved byte for byte:** the V0.10a, V0.10b and V0.10c development
  evidence.

**Quality gates:**
| Gate | Where | Result |
|---|---|---|
| Python tests: agents (`agents/llm`), evidence (`analysis/evidence`), repeatability (`analysis/repeatability`) | horo and CI | 60, 147 and 74 tests, OK |
| Go suite (`go test ./...`), golden references, seed guards (`internal/eval`), historical session replay (`internal/session`, the new `v0103` tests included) | CI, three platforms; race run on horo | passed |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64 | exit 0; 10 packages ok, no failure |
| `go vet ./...` (default, calibration, acceptance tags), `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64) | horo | all ok |
| Historical replays, benchmark verify and reproduce, benchmark fingerprints | CI | passed on the release commit |
| V0.10a, V0.10b and V0.10c analyses regenerated (sample, points, inventory, schedules, analyses, prototype, financial evidence, continuation) and compared | CI, three platforms | byte-identical to the committed files |
| Cross-platform comparison (linux/amd64, linux/arm64, darwin/arm64) | CI | passed on the release commit |

Not run: no live inference, no clinical suite, no historical live
evaluation.

**Not done:** no live inference, no clinical suite, no historical live
evaluation.

---

## V0.10c — validation feedback against preview information (0.10.0-dev.3): results

The experiment declared at 679483e (below), run and analysed. Report:
`docs/benchmark/v10c-dev/README.md`.

**Live run:** 2026-11-01 23:46 to 2026-11-02 00:26 UTC.
* 600 of 600 requests, all HTTP 200, one at a time; no stop.
* 30 server checks, all the same configuration.
* 2 invalid replies (neutral).

**Decisions** (kept / revised / abandoned / invalid; Gemma of 10, Qwen
of 20):

| Point | Proposal | A neutral | B validation | C preview |
|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | accepted | 10/0/0/0 | 10/0/0/0 | 10/0/0/0 |
| fl-gemma-s41901-p26 | refused | 10/0/0/0 | 0/10/0/0 | 0/10/0/0 |
| fl-gemma-s41901-p28 | refused | 10/0/0/0 | 0/10/0/0 | 0/10/0/0 |
| fl-gemma-s41901-p27 | refused | 10/0/0/0 | 0/0/10/0 | 0/0/10/0 |
| ss-qwen-pv-s42002-p6 | accepted | 12/7/0/1 | 0/20/0/0 | 20/0/0/0 |
| ss-qwen-s42002-p7 | accepted | 20/0/0/0 | 0/20/0/0 | 20/0/0/0 |
| fp-qwen-s41901-p26 | accepted | 20/0/0/0 | 0/20/0/0 | 0/20/0/0 |
| ss-qwen-s42001-p20 | accepted | 20/0/0/0 | 0/20/0/0 | 17/3/0/0 |
| fl-qwen-s41901-p9 | refused | 0/19/0/1 | 0/18/2/0 | 0/20/0/0 |
| fl-qwen-s41901-p11 | refused | 20/0/0/0 | 0/0/20/0 | 0/0/20/0 |
| ss-qwen-s42001-p23 | refused | 0/20/0/0 | 0/17/3/0 | 0/19/1/0 |
| fp-qwen-s42001-p27 | refused | 20/0/0/0 | 0/20/0/0 | 0/20/0/0 |

* **C − B, primary:** the same decisions at 7 of 12 points.
  * At 3 accepted Qwen points the preview kept what validation revised.
  * At 2 refused Qwen points the preview's revisions were more often
    valid: 20 vs 2 and 16 vs 5 of 20.
* **B − A:** under validation the model revised or abandoned every
  refused proposal in every draw, and revised every accepted Qwen
  proposal. Under A it resubmitted at 5 of the 7 refused points and kept
  at the accepted ones.
* **Money:**
  * Validation dropped ss-qwen-pv-s42002-p6's $44,687.30 credit-supported
    allocations (to $303.75) and ss-qwen-s42002-p7's teleradiology
    contract. The preview kept both.
  * Credit-supported commitments submitted: A 12, B 10, C 34.
* **Repeatability:** Gemma gave one decision per arm everywhere. Qwen
  varied most under A.

**Continuation:** 53 distinct decisions were run with the no-action
suffix. The 21 applied ones agreed with their validation, 21 of 21.

**Reproducibility:** the analysis contacts no model. It checks every
request and response hash and the engine's answers, and is byte-identical
on the Pi, horo and CI.

**Quality gates:**
| Gate | Where | Result |
|---|---|---|
| Focused shared-prefix tests (`analysis/repeatability`; 17 new for v3) | Pi, horo and CI | 74 tests, OK |
| Agent tests (`agents/llm`), evidence tests (`analysis/evidence`) | horo and CI | 60 and 147 tests, OK |
| Go suite (`go test ./...`), golden references | CI, three platforms | run by CI on this commit; the new `v0102` session tests also on the Pi |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64 | exit 0; 10 packages ok, no failure |
| `go vet ./...` (default, calibration, acceptance tags), `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64) | horo | all ok |
| Historical replays (V0.8c, V0.9c) | horo and CI | 104 checks, 0 failed, each of 4 replays |
| Benchmark verify and reproduce | CI | run by CI on this commit |
| Inventory and branch points made again (selection, engine answers, requests) | horo and CI | 12 contexts verified |
| Analysis regenerated and compared | Pi, horo (twice) and CI | byte-identical |
| Cross-platform CI (linux/amd64, linux/arm64, darwin/arm64) | CI | run by CI on this commit; declaration commit 679483e green (run 41363727129) |

Not run: no clinical suite, no historical live evaluation rerun, and no
local repeat of the gates CI runs (Go suite, golden references, verify
and reproduce).

**No change to:** the simulator, the agents, prompts, settings, scoring,
the frozen evidence rules or any earlier recording. No historical live
evaluation was rerun, and no clinical suite was run.

**Recommendation:** release V0.10.0 as a methodology and measurement
release, after owner review.

---

## V0.10c — validation feedback against preview information (0.10.0-dev.3): declaration

V0.10b was accepted (CI 41349684200) and V0.10c authorized by the owner.
This commit declares the experiment before any live request. Report:
`docs/benchmark/v10c-dev/README.md`.

**Information inventory** (`API_INVENTORY.md`, `points/inventory.json`).
The engine's validate and preview answers for the 56 recorded staffing
proposals, each asked at its recorded state:
* the preview's `results` equal validate's in 56 of 56;
* a refused proposal's preview adds `asOf` and, for `insufficient_funds`,
  the funding test, whose numbers the refusal message already states;
* a valid proposal's preview adds the structured funding test, money
  lines, item details, slots and displacements.

Neither answer states a credit requirement or a reversibility class.

**Protocol** (`v10c-feedback-1`, `dreamteam.shared-prefix/v3`; `PLAN.md`,
`SPEC.md`, `protocol.json`, `points/`, `schedule.json`):
* **Points:** 12. Gemma has all 4 of its contexts (1 accepted, 3
  refused). Qwen has 4 accepted, by financial-category coverage and
  stake, and 4 refused, insufficient funds first.
* **Arms:**
  * A, V0.10b's neutral control;
  * B, the engine's exact validate answer;
  * C, llm-agent 4's exact preview message.

  All three end with the same final paragraph.
* **Draws:** Gemma 10 and Qwen 20 per arm per point; 600 requests, one at
  a time, in hash-ordered rounds.
* **Analysis:** primary C − B per point, with B − A and C − A as
  references; financial evidence v1; the five distinctions kept apart
  (information, change, validity, commitment, exposure).
* **No length control.** Padding would be a new stimulus or invented
  content; the lengths are reported.

**Code:** `analysis/repeatability/feedback_experiment.py` imports
shared-prefix/v2's module unchanged. It covers the inventory, the points
and their CI check, the schedule, the sequential live client, and the
analysis, which contacts no model. Tests: `test_feedback_experiment.py`,
17 tests.

**Engine:** unchanged. 0.10.0-dev.2 is registered for replay
(`mini-seed52-v0102`, batch mode `v0.10_dev2`).

---

## V0.10b — live shared-prefix decision experiment (0.10.0-dev.2): results

The experiment declared at ab1d8d1 (below), run and analysed. Report:
`docs/benchmark/v10b-dev/README.md`.

**Live run:** 2026-11-01, 19:11–19:25 UTC.
* 320 of 320 requests, all HTTP 200, one at a time; no stop.
* 40 server checks, all the same configuration.
* 7 invalid replies (6 control, 1 treatment), counted in every
  denominator.

**Decisions** (kept / revised / abandoned / invalid, of 20 per arm):

| Point | Preview | Control | Treatment |
|---|---|---|---|
| ss-gemma-s42001-p14 | accepted | 20 / 0 / 0 / 0 | 20 / 0 / 0 / 0 |
| fl-gemma-s41901-p27 | refused | 20 / 0 / 0 / 0 | 0 / 0 / 20 / 0 |
| fl-gemma-s41901-p28 | refused | 20 / 0 / 0 / 0 | 0 / 20 / 0 / 0 |
| fl-gemma-s41901-p26 | refused | 20 / 0 / 0 / 0 | 0 / 20 / 0 / 0 |
| ss-qwen-s42002-p6 | accepted | 11 / 8 / 0 / 1 | 20 / 0 / 0 / 0 |
| ss-qwen-s42001-p1 | accepted | 20 / 0 / 0 / 0 | 20 / 0 / 0 / 0 |
| ss-qwen-s42002-p26 | refused | 1 / 7 / 7 / 5 | 0 / 0 / 20 / 0 |
| ss-qwen-s42002-p1 | refused | 16 / 4 / 0 / 0 | 0 / 19 / 0 / 1 |

* **Refused-preview points:** the treatment changed the decision in 99 of
  100 draws, and the control resubmitted the refused proposal in 77.
  Two revisions corrected a malformed contract, which was then accepted.
* **Accepted-preview points:** the same at 2 of 3. At ss-qwen-s42002-p6
  the treatment kept a credit-supported $44,687.30 commitment 20/20,
  against 11/20 under control.
* **Credit-supported commitments submitted:** control 11, treatment 40.
* **Repeatability:**
  * Gemma: one decision per arm at every point.
  * Qwen: 5–17 distinct texts per control arm, 2–4 per treatment arm.
  * At the V0.9c points the treatment reproduced V0.9c's recorded
    decision in 99 of 100 draws.
* **Interpretation:** the control has no validation feedback, and the
  treatment message is longer. This is a local comparison of the preview
  against no information, at these 8 points, with intervals that are
  descriptive.

**Continuation:** 26 distinct decisions were run with the no-action
suffix. The 8 applied ones agreed with their validation, 8 of 8.

**Reproducibility:** the analysis contacts no model and rebuilds every
request. It is byte-identical on the Pi, horo and CI.

**Quality gates:**
| Gate | Where | Result |
|---|---|---|
| Focused tests: `analysis/repeatability` (25 new) | Pi, horo and CI | 57 tests, OK |
| Python tests: `agents/llm`, `analysis/evidence` | CI | run by CI on this commit (unchanged code) |
| Go suite (`go test ./...`), golden references | CI, three platforms | run by CI on this commit; the new `v0101` session tests also run on the Pi |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64 | exit 0; 10 packages ok, no failure |
| `go vet ./...` (default, calibration, acceptance tags), `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64) | horo | all ok |
| Historical replays (V0.8c, V0.9c) | horo and CI | 104 checks, 0 failed, each of 4 replays |
| Benchmark verify and reproduce | CI | run by CI on this commit |
| Branch points checked again (selection, previews, requests) | Pi, horo and CI | 8 contexts verified |
| Analysis regenerated and compared | Pi, horo (twice) and CI | byte-identical |
| Cross-platform CI (linux/amd64, linux/arm64, darwin/arm64) | CI | run by CI on this commit; declaration commit ab1d8d1 green (run 41340455061) |

Not run: no clinical suite, no historical live evaluation rerun, and no
local repeat of the gates CI runs (Go suite, golden references, verify
and reproduce).

**No change to:** the simulator, the agents, prompts, settings, scoring,
the frozen evidence rules or any earlier recording. No historical live
evaluation was rerun, and no clinical suite was run.

**Recommended V0.10c:**
* a control with the agents' validation feedback;
* a length-matched or length-reported control;
* prospective branch points with financial stakes;
* draws fixed from V0.10b's variation.

---

## V0.10b — live shared-prefix decision experiment (0.10.0-dev.2): declaration

V0.10a was accepted (de8ac79, CI 41326685680) and V0.10b authorized by the
owner. This commit declares the experiment before any live request.
Report: `docs/benchmark/v10b-dev/README.md`.

**Inventory and the owner's decision**
(`docs/audits/v0.10b-branch-point-inventory.md`):
* 56 V0.8c and V0.9c decision points have a staffing proposal. All are
  verifiable. The engine's preview at each recorded state accepts the
  proposal at 19 and refuses it at 37.
* Gemma has only one accepted-preview context. Its staffing shortage,
  s42001, point 14 is the same bytes in the baseline and preview runs. So
  the declared 2 + 2 per model was impossible. The run stopped for the
  owner, who approved option A: Gemma 1 accepted and 3 refused (all its
  contexts), Qwen 2 and 2.

**Protocol** (`v10b-decision-1`, `dreamteam.shared-prefix/v2`; `PLAN.md`,
`SPEC.md`, `protocol.json`, `points/`, `schedule.json`):
* **Arms,** after the same recorded conversation and initial proposal:
  * control: a neutral continuation, whose final paragraph is the
    treatment message's own;
  * treatment: llm-agent 4's review message with the engine's exact
    preview.
* **Draws:** one reply each, no retry; 20 draws per arm per point; 320
  requests, one at a time. Rounds are ordered by hash, Gemma then Qwen.
* **Outcome:** kept, revised, abandoned or invalid, every draw in the
  denominator. Wilson and Newcombe intervals per point before any pooled
  summary.
* **Continuation:** a fixed no-action suffix. Financial evidence v1 is
  applied to each distinct decision.
* **Stops:** server checks before each round.

**Code** (`analysis/repeatability/decision_experiment.py`):
* points: candidates, selection, contexts, and the CI check;
* the schedule and the sequential live client;
* the analysis, which contacts no model.

Tests: `test_decision_experiment.py`, 25 tests (the repeatability suite
now has 57), against a scripted engine and a stub server.

**Engine:** unchanged. 0.10.0-dev.1 is registered for replay
(`mini-seed52-v0101`, batch mode `v0.10_dev1`).

---

## V0.10a — inference repeatability and controlled comparison design (0.10.0-dev.1): results

The protocol declared at 41b16c9 (below), run, analysed and prototyped.
Report: `docs/benchmark/v10a-dev/README.md`.

**Live repeatability** (2026-11-01, 14:23–14:48 UTC; 375 of 375 requests
answered, no stop, no rerun):
* **Text:** identical across all 15 requests of a prompt for 2 of 10 Gemma
  prompts and 0 of 15 Qwen prompts. Parsed replies varied exactly as the
  text did: no difference was only formatting.
* **Actions:** identical for 7 of 10 and 6 of 15. **Decision:** identical
  for 8 of 10 and 9 of 15.
* **Sequential vs concurrent:** requests sent one at a time (11 per
  prompt) changed the decision at 3 of 25 prompts, and two at a time (4
  per prompt) at 8. Concurrent requests were the least repeatable at every
  level. Two identical requests sent at once had identical text in 11 of
  50 pairs (exploratory).
* **Seed and cache:** `"seed": 0` gave the same counts as without it. The
  prompt cache was hit by 365 of 375 requests, including most of those
  whose answers differed. Cache clearing, cache disabling and a cold start
  were unavailable, and none is claimed.
* **Most variable:** staffing proposals, which alternated between
  submitting, being refused and doing nothing.

**Shared-prefix design** (`DESIGN.md`, `shared-prefix/v1`):
* the arms share the recorded state and the recorded initial proposal,
  then continue as llm-agent 3 (control) and 4 (treatment, preview
  shown), with a declared suffix;
* what is held fixed, what differs by design, what remains uncontrolled,
  and what can and cannot be identified; the assumptions a causal reading
  needs.

**Offline prototype** (`analysis/repeatability/shared_prefix.py`,
`prototype/`): an external benchmark policy, one spec per V0.9c preview
point (9):
* prefixes and proposals were identical in both arms and equal to the
  recording at 9 of 9 points;
* the treatment arm submitted what the recording submitted at 9 of 9;
* where both arms submitted the same batches (5), the runs never
  diverged.

The control arm's replies are scripted, so nothing it reports is an
effect.

**New:**
* `analysis/repeatability/`: `prompt_sample.py`, `repeat_run.py`,
  `repeat_analysis.py`, `shared_prefix.py`, with 32 tests (14 for the
  sample, the runner and the analysis levels, 18 for the prototype);
* `docs/benchmark/v10a-dev/`: `live/`, `repeatability.json`,
  `repeatability-tables.md`, `DESIGN.md`, `prototype/`, `README.md`;
* a CI step that checks the sample, rebuilds the schedule and the
  prototype's specs, and regenerates the analysis and both prototype arms
  and compares them;
* LLM_AGENT §15, BENCHMARK §11a.

**Quality gates:**
| Gate | Where | Result |
|---|---|---|
| Focused tests: `analysis/repeatability` | Pi and CI | 32 tests, OK |
| Python tests: `agents/llm`, `analysis/evidence` | Pi and CI | 60 and 147 tests, OK |
| Go suite (`go test ./...`), golden files | CI, three platforms | run by CI on this commit |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64, Go 1.27.0 | exit 0; 10 packages ok, no failure |
| `go vet ./...`, `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64) | horo | all ok |
| Historical replays (V0.8c, V0.9c recordings) | Pi, horo and CI | 104 checks, 0 failed, each of 4 replays |
| Benchmark verify and reproduce; prototype runs verified | CI | run by CI on this commit; prototype runs 41, 113 and 77 checks, 0 failed |
| Analysis, specs, prototype and comparison regenerated and compared byte for byte | Pi, horo and CI | identical |
| Cross-platform CI (linux, macOS, Windows) | CI | run by CI on this commit |

Not run: no clinical suite, no historical live evaluation rerun, and no
local repeat of the gates CI runs (Go suite, golden files, verify and
reproduce), per the project's practice of running only the race and the
cross-builds locally.

**No change to:**
* simulator behaviour, actions, scenarios, scoring or accounting;
* llm-agent 1–4, their prompts or settings;
* any recording, session, golden reference, suite or benchmark result;
* the frozen evidence.

The independent review is still PENDING. No historical live experiment
was rerun.

---

## V0.10a — inference repeatability and controlled comparison design (0.10.0-dev.1): declaration

Owner decision 2026-11-01: V0.9.0 is accepted and closed (7fa9557, tag
`v0.9.0`, CI run 41307369300). V0.10a is authorized: a methodology and
measurement milestone. No simulator mechanics, agent prompt or decision
change, scoring change, or multi-day live benchmark.

**Declared before any request** (`docs/benchmark/v10a-dev/`):
* **The protocol** `v10a-repeatability-1` (`PLAN.md`, `protocol.json`):
  * the 25 sampled requests sent again, unchanged, to the same two
    models;
  * four conditions: `interleaved` (5 rounds), `back_to_back` (3 in a
    row), `concurrent` (2 rounds of 2 at once) and `seeded` (3 in a row
    with `"seed": 0`). 375 requests;
  * the exact order in `schedule.json`;
  * stop conditions, resource limits, and the five comparison levels
    (raw, parsed, actions, validation, decision);
  * unavailable and not invented: cache cleared, cache disabled and
    cold start. They need admin access or would affect the shared
    server's other users.
* **The prompt sample** (`sample/`, `analysis/repeatability/prompt_sample.py`,
  `dreamteam.repeatability-sample/v1`):
  * a deterministic, hash-ranked sample from the V0.8c and V0.9c
    recordings in five categories (no action, staffing, financially
    consequential, validation refusal, preview review);
  * every message of every request verified against the recorded SHA-256;
  * re-checked in CI.
* **The server record** (`SERVER.md`, `server-declared.json`): oMLX 0.7.0;
  what its public endpoints show, and what only the admin API would.
* **The live client** (`analysis/repeatability/repeat_run.py`): the
  endpoint is never recorded, nothing is retried or dropped; tests with a
  stub server.
* **Version** 0.10.0-dev.1; 0.9.0 registered for replay
  (`mini-seed52-v090`, made from a clean 7fa9557 worktree; batch mode
  `v0.9.0`; `internal/session/v090_test.go`).
* **Baseline** at 7fa9557 (`docs/benchmark/v10a-dev/README.md` §1).

No change to the simulator's behaviour, the agents, their prompts or
settings, any recording, evidence rule, suite or golden reference.

---

## V0.9.0 — release: financial decision evidence and preview-based decision support

Owner decision 2026-11-01:
* V0.9c accepted and closed (d4d2f2f, be537f2, a38d031; CI run
  41274622273 green);
* GO for the V0.9.0 release, release-only;
* no V0.9d; no simulator, agent, prompt, model, scoring, scenario or
  evaluation change, and no evaluation rerun;
* V0.10 proposed (ROADMAP §7d), not approved.

The owner's finding: V0.9c measured the preview interactions but could
not identify their downstream financial effect. **No financial benefit of
preview access was established.**

**Baseline** (`docs/audits/v0.9.0-release-baseline.md`): clean `main` at
a38d031; the tags `v0.5.0` to `v0.8.0` unchanged, locally and on the
remote; hashes of the historical recordings (114), the frozen evidence
contracts, the review package and page, the suites, the golden
references, the agents (llm-agent 1–4), the V0.9c plan and results, and
all development evidence. On the release tree only the session fixtures
changed: two added.

**Changed:**
* Version 0.9.0. 0.9.0-dev.3 is registered for replay
  (`mini-seed52-v093`, made from a clean a38d031 worktree; batch mode
  `v0.9_dev3`; `internal/session/v093_test.go`). Every earlier batch mode
  is kept.
* Release notes: V0.9.0, with the thirteen capabilities, the V0.9a, V0.9b
  and V0.9c findings with their denominators, the staffing-shortage case
  study (direct consequences apart from run outcomes), nondeterminism and
  the four properties (simulator replay, recorded-inference replay, live
  inference, causal identification), the pending review and the known
  limitations.
* README (V0.9 current; limitations), ROADMAP (V0.9 released; §7d, V0.10
  proposed), DECISION_EVIDENCE, PARAMETER_EVIDENCE, FINANCIAL_EVIDENCE and
  LLM_AGENT (status).
* The development evidence (`docs/benchmark/v09a-dev/`, `v09b-dev/`,
  `v09c-dev/`) is unchanged.

No change to mechanics, randomness, scenarios, actions, accounting,
evaluation, scoring, the harness, the protocols, the policies, the LLM
agents, their prompts or settings, or any evidence rule. No evaluation
was rerun, and no model was contacted.

**Release gates** (linux/amd64 unless noted):
* Clean release tree (only release changes; baseline unchanged but for
  the two added fixtures).
* Full default Go suite on the release tree (horo): all packages pass,
  including `v093_test.go`, the golden references and the committed seed
  guards (`TestCommittedConfigs`, `TestCommittedSuiteRuns`, the holdout
  guards).
* The V0.9 fixture tests (`TestV091*`, `TestV092*`, `TestV093*`) on
  linux/arm64 (the Pi): pass.
* Five cross-builds: linux/amd64, linux/arm64, darwin/arm64,
  darwin/amd64, windows/amd64.
* CI on the release commit (linux/amd64, linux/arm64, darwin/arm64, and
  the cross-platform comparison): vet (three tag sets), gofmt, the full
  suite, golden references, the Python agent, evidence and probe tests,
  benchmark verify and reproduce, every recorded-LLM replay (llm-agent 3
  and 4 included), the frozen evidence checks, and every V0.8 and V0.9
  pipeline regenerated and compared with the committed outputs.
* **Not rerun for the release,** by the owner's decision: the race
  detector and the separate linux/amd64 rerun of the replays and
  pipelines. Both passed on a38d031 (V0.9c), and the release changes no
  concurrent code: the version constant, one batch-mode constant and one
  map entry.

No live LLM evaluation and no clinical suite was rerun.

---

## V0.9c — prospective preview evaluation (0.9.0-dev.3): results

The evaluation declared at d4d2f2f (below), run and analysed
(`docs/benchmark/v09c-dev/README.md`). The plan was not amended. The
independent review is still PENDING, and V0.9 is unreleased.

**Run:**
* 20 of 20 runs on the Pi, in about 70 minutes: 16 live (8 pairs) and 4
  no action.
* All 555 model requests were answered: no infrastructure-affected point,
  no failure, no rerun.

**Found:**
* **The preview step.** 9 previews were shown (Qwen 8, Gemma 1), in 9
  eligible points out of 240, with no failures.
  * The 5 that showed a refused action: 4 revised, 1 abandoned. In four of
    the five, the model's next reason describes the refusal as a failed
    attempt.
  * The 4 that showed accepted, funded actions were kept. One of them,
    Qwen's allocation raises ($44,383.55), was funded only with the credit
    line.
  * No preview showed an unfunded commitment, a termination or displaced
    bookings.
* **Model variability.** The server's answers differed at byte-identical
  requests at temperature 0:
  * Qwen: 0 of 5 the same;
  * Gemma: 85 of 91 the same output, 91 of 91 the same proposal.

  Every Qwen pair diverged at point 0 or 1, at or before its first
  preview. One Qwen pair with no preview shown still differs by −$6,646.03
  in operating profit, so the run-level pairs cannot isolate the preview.
* **Gemma's four pairs** end identical in every run outcome.
* **No run** had financial distress, drew credit, or had a batch refused
  for funds.
* **Financial evidence v1** runs unchanged on both conditions (425 records,
  every validated batch linked), through two documented adapters: the v3
  view of a v4 record, and the transcript without the agent's own digests
  check after a preview (FINANCIAL_EVIDENCE.md §9).

**New:**
* `analysis/evidence/preview_evaluation.py`
  (`dreamteam.preview-evaluation/v1`) and its tests;
* the evaluation's recordings, usage files and contract report
  (`docs/benchmark/v09c-dev/eval/`), with one replay configuration per
  entry and the corpus;
* the outputs: `preview-evaluation.json`, `preview-tables.md`,
  `financial-evidence.json`, `financial-tables.md`;
* a CI step that replays the evaluation and compares the outputs;
* LLM_AGENT.md §14.5 and FINANCIAL_EVIDENCE.md §9.

**Quality gates:**

Run on linux/amd64 (horo) unless noted:
* **Focused tests:**
  * agent tests (60, including the 20 llm-agent 4 tests);
  * evidence tests (147, including the 8 preview evaluation tests);
  * probe tests (3, against the built binary).
  All pass.
* **Go:**
  * full default suite: all packages pass, including
    `TestLLMAgent4Preview`, `TestLLMAgent4Replay` and the new
    `mini-seed52-v092` fixture (0.9.0-dev.2, batch mode `v0.9_dev2`);
  * race (`go test -race -timeout 90m ./...`): pass;
  * vet (three tag sets) and gofmt: clean.
* **Historical replay:** every fixture replays with every digest
  verified; every recording in the corpus was replayed at 0.9.0-dev.3.
* **Benchmark verify and reproduce:** 70 of 70, 0 failed, including the
  two V0.9c entries (104 checks each).
* **Earlier pipelines:** the V0.8c pipeline, run twice, is byte-identical
  and equal to the committed files; `check_v08b.py` reproduces the V0.8b
  review package and probe log, with build numbers set aside; the V0.9a
  outputs are rebuilt identically and the review status is PENDING.
* **Parameter evidence v1 freeze:** the frozen analyzer, corpus,
  recordings and outputs are reproduced.
* **Financial evidence (V0.9b):** run twice, 2,062 records from 78
  recordings, byte-identical and equal to the committed files.
* **Preview evaluation (V0.9c):** the 16 LLM runs replayed and the
  analysis run twice: 8 pairs, 425 financial evidence records,
  byte-identical and equal to the committed files.
* **Earlier evidence, frozen files, baseline agents (llm-agent 1–3),
  suites and golden references:** unchanged; no deletions.
* **Five cross-builds:** linux/amd64, linux/arm64, darwin/arm64,
  darwin/amd64, windows/amd64: all pass.
* **CI** (run 41268808888, on be537f2): linux/amd64, linux/arm64 and
  darwin/arm64, with the cross-platform comparison and the new
  prospective preview evaluation step: all pass.
* **Clinical suites:** not run, as specified.

**No change to:**
* simulator behaviour, actions, scenarios, scoring or accounting;
* llm-agent 1, 2 or 3, their prompts or settings;
* any earlier recording, session, golden reference or benchmark result;
* the frozen evidence: decision evidence v1, repeat v2, refusal-response
  v1 and v2, parameter evidence v1, and V0.9b's financial evidence outputs.

No historical live experiment was rerun.

---

## V0.9c — prospective preview evaluation (0.9.0-dev.3): declaration

Owner decision 2026-10-31: V0.9b is accepted and closed (8ffae8d, CI run
41251637831). V0.9c is authorized: a controlled, prospective comparison of
llm-agent 3 (unchanged) and a preview-enabled agent version, on frozen
models. No simulator mechanics, accounting or scoring changes, and no
tuning against results.

**Declared before any live run** (`docs/benchmark/v09c-dev/`):
* **llm-agent 4** (`agents/llm/llm_agent4.py`, LLM_AGENT.md §14):
  * llm-agent 3, loaded unchanged, with one step, `llm-agent-preview/v1`.
    The first parsed proposal at a decision point that has a staffing
    action is previewed with the protocol's existing `preview`.
  * The model is shown the answer exactly as returned and replies with its
    decision: kept, revised or abandoned. That decision is validated and
    submitted as version 3's would be.
  * A failed or oversized preview is recorded and not shown, and the
    proposal goes to validation unchanged.
  * The record is `dreamteam.llm-decision-record/v4`, and the replay asks
    every preview again.
* **The plan** (`PLAN.md`, `v09c-plan/1`):
  * financial-pressure-credit and staffing-shortage-credit;
  * seeds 42001 and 42002, 30 days;
  * no action, both models, both conditions: 20 runs, 16 of them live;
  * the 20 outcomes, analyses A–G, stop conditions and replay
    requirements.
* **The treatment isolation record** (`TREATMENT.md`): what is identical,
  the only added text (SHA-256 `f8b7d787…`), and the unavoidable
  differences (one more model request per preview shown, a longer
  conversation at that point).
* **The baseline** (`README.md` §1): hashes at 8ffae8d of the agents,
  settings, frozen evidence, recordings, suites, golden references and
  fixtures.

**Tests:**
* `agents/llm/test_llm_agent4.py` (20 tests): the 16 required cases.
* `internal/bench/llm_agent4_test.go`, against the engine:
  * kept proposals give version 3's decisions and session byte for byte,
    so the preview alone changes nothing;
  * every first request equals version 3's;
  * replay equals live;
  * tampering is refused.
* The configuration ran for one day on seed 777 against a stub server (not
  kept).

**Version:** 0.9.0-dev.3, with no engine change. 0.9.0-dev.2 is registered
for replay (`mini-seed52-v092`, batch mode `v0.9_dev2`).

---

## V0.9b — financial decision quality and operational exposure (0.9.0-dev.2)

Owner decision 2026-10-31:
* V0.9a accepted and closed (a5e5f9b, CI run 41230474654).
* V0.9b is authorized: a read-only analysis of what recorded decisions
  committed the practice to, how reversible they are and which
  capabilities they put at risk.
* No simulator, agent, prompt or scoring change, and no live LLM run.

**Built:**
* **Parameter evidence v1 frozen**
  (`analysis/evidence/frozen-parameter-v1.json`,
  `check_parameter_frozen.py`, run in CI). Its in-sample calibration and
  limited recall are documented in PARAMETER_EVIDENCE.md and DECISION_EVIDENCE
  §18.
* **Financial evidence** `dreamteam.financial-evidence/v1`
  ([FINANCIAL_EVIDENCE.md](FINANCIAL_EVIDENCE.md),
  `analysis/evidence/financial_evidence.py`). Three dimensions, never
  combined:
  * **affordability:** one record per submitted batch, under the engine's
    funding rule;
  * **reversibility:** one record per executed action, under the
    simulator's own rules ([the matrix](benchmark/v09b-dev/REVERSIBILITY.md));
  * **operational exposure:** direct state changes (A), kept apart from
    the observed timeline (B) and the pairing with no action (C).
* **Engine-sourced values.**
  * Each replay's transcript is fed back to the engine with read-only
    previews, observations and validations inserted.
  * Every original answer must be identical, or the analysis stops.
  * Commitments, money lines, notice, severance, retained obligations and
    displaced bookings are the engine's preview values.
  * A shift extension's commitment is the reservation it added.
* **The corpus** ([benchmark/v09b-dev/corpus.json](benchmark/v09b-dev/corpus.json)):
  V0.6c, V0.7c and V0.8c, each group with its own economics and contract
  reports, and three case studies.
* **CI** checks the parameter evidence freeze. It runs the financial
  evidence on the replays it already makes, compares the outputs with the
  committed files, and compares them across platforms.
* **Version 0.9.0-dev.2.** It registers 0.9.0-dev.1 for replay
  (`mini-seed52-v091`, made from a5e5f9b; batch mode `v0.9_dev1`;
  `internal/session/v091_test.go`).

**Results** ([benchmark/v09b-dev/](benchmark/v09b-dev/README.md)):
* **The engine replay:** 78 recordings and 3,540 decision points, every
  original answer identical. 77 refused llm-agent 2 final replies were
  validated again, and each reproduced its recorded validation exactly.
* **Affordability,** by submitted batches:
  * V0.6c (281): no funding rule, so 169 are `evidence_unavailable` by
    definition.
  * V0.7c (272): 180 funded without credit, 30 only with credit, and no
    funding refusal.
  * V0.8c (69): 8 funded without credit, 9 with credit, and 10 refused for
    insufficient funds, all in the low-liquidity entry.
* **Reversibility,** by executed actions: 720 actions, none resting on
  the rule alone (every staffing action had a matching preview item).
  * V0.6c (396): 342 not reversible, 31 reversible with a retained
    obligation, 15 withdrawable, 8 policies.
  * V0.7c (291): 243, 27, 10 and 11.
  * V0.8c (33): 22, 2, 8 and 1.

  Extensions make up most of "not reversible".
* **Credit:** five runs drew on the line: the V0.7c staffing-shortage Qwen
  run (at most $17,200.28) and the four V0.8c low-liquidity runs.
* **The case studies.**
  * V0.7c staffing shortage, Qwen: the MR hire was funded only by the
    credit line, and the engine stated that the termination batches
    displaced 991 booked appointments. Both were knowable at the decision,
    before any loss was observed.
  * A limited-intervention contrast.
  * The V0.8c low-liquidity refusals.
* **Agents never called `preview`.** The exact commitments were knowable
  on request, not shown.

**Quality gates:**

Run on linux/amd64 unless noted:
* **Focused tests:**
  * agent tests (40);
  * evidence tests (139, including the 31 financial evidence tests, which
    cover the 18 required cases, and the parameter evidence freeze check);
  * probe tests (3, against the built binary).
  All pass.
* **Go:**
  * full default suite: all packages pass, including the new
    `mini-seed52-v091` fixture (0.9.0-dev.1, batch mode `v0.9_dev1`);
  * race (`go test -race -timeout 90m ./...`): pass;
  * vet (three tag sets) and gofmt: clean.
* **Historical replay:** every fixture replays with every digest
  verified; every recording in the corpus was replayed at 0.9.0-dev.2.
* **Benchmark verify and reproduce:** 66 of 66, 0 failed.
* **Earlier pipelines:** the V0.8c pipeline, run twice, is byte-identical
  and equal to the committed files; `check_v08b.py` reproduces the V0.8b
  review package and probe log, with build numbers set aside; the V0.9a
  outputs are rebuilt identically and the review status is PENDING.
* **Parameter evidence v1 freeze:** the frozen analyzer, corpus,
  recordings and outputs are reproduced.
* **Financial evidence:** run twice, 2,062 records from 78 recordings,
  byte-identical and equal to the committed files. Every engine replay
  answered identically to its recording.
* **Earlier evidence, suites and golden references:** unchanged; no
  deletions.
* **Five cross-builds:** all pass.
* **CI:** linux/amd64, linux/arm64 and darwin/arm64, with the
  cross-platform comparison and the new financial evidence step.

**No change to:**
* simulator behaviour, actions, scenarios, scoring, agents, prompts or
  settings;
* any recording, session, golden reference or benchmark result;
* the frozen rules and outputs: decision evidence v1, repeat v2,
  refusal-response v1 and v2, parameter evidence v1;
* the review package, its protocol, the review page and status.

No model was contacted.

**Known limitations:**
* **The independent review is pending.** No rule is independently
  validated.
* **Knowable is not known.** The agents never requested the previews this
  analysis uses.
* **llm-agent 2 records only the final reply.** 301 earlier attempts are
  counted but unknown.
* **No causal attribution.** No later outcome is attributed to one
  decision. That would need runs that differ in one decision only, which
  were not performed.
* **In-sample, development recordings only.**

---

## V0.9a — evidence validation and decision/action parameter consistency (0.9.0-dev.1)

Owner decision 2026-10-31:
* V0.8.0 accepted and closed (b803302, tag v0.8.0, CI run 41211327314).
* V0.9a is authorized: the independent review as a separate human task,
  and narrowly scoped, versioned parameter-consistency evidence on
  existing recordings.
* No simulator, agent, prompt or scoring change, and no live LLM run.

**Built:**
* **The independent review workflow**
  ([benchmark/v09a-dev/review/](benchmark/v09a-dev/review/INSTRUCTIONS.md)).
  The frozen V0.8b package and its protocol are unchanged.
  * `review.html`: one offline, self-contained page built from the
    package by `analysis/evidence/review_tool.py`. A reviewer reads each
    item's evidence, labels it, explains, declares independence, and
    saves `form.csv` and `reviewer.json`.
  * `review_metrics.py` (`dreamteam.evidence-review-metrics/v1`): the
    checks and metrics for completed forms. Precision is given with
    denominator and sampling; recall never.
  * `INSTRUCTIONS.md`, `REVIEW_STATUS.md` and `review-status.json`
    (**PENDING**).
  * `RECALL_PROTOCOL.md`: a design, not run.
* **Parameter evidence** `dreamteam.parameter-evidence/v1`
  ([PARAMETER_EVIDENCE.md](PARAMETER_EVIDENCE.md),
  `analysis/evidence/parameter_evidence.py`):
  * AMOUNT, TARGET, DATE, DURATION and DIRECTION mismatches between an
    explicit statement in an action phrase and the action submitted in
    the same reply;
  * deterministic normalization, with no tolerance;
  * explicit abstention.
* **The corpus** ([benchmark/v09a-dev/corpus.json](benchmark/v09a-dev/corpus.json)):
  V0.5c–V0.7c (llm-agent 2) and V0.8c (llm-agent 3), on the replays CI
  already runs.
* **CI** rebuilds the review page and the PENDING status, runs the
  parameter evidence twice, compares each with the committed file, and
  compares them across platforms.
* **Version 0.9.0-dev.1.** It registers 0.8.0 for replay
  (`mini-seed52-v080`, made from b803302; batch mode `v0.8.0`;
  `internal/session/v080_test.go`).

**Results** ([benchmark/v09a-dev/](benchmark/v09a-dev/README.md)):
* **Independent review:** PENDING. Nothing was labelled. The workflow was
  tested with synthetic labels only, which the tests generate and
  discard.
* **Parameter evidence:** 96 recordings, 4,191 decision points.
  * 816 eligible checks: 762 consistent, **0 supported**, 54 unresolved,
    0 insufficient evidence.
  * By rule (eligible): target 493, date 146, duration 142, amount 21,
    direction 14.
  * Older recordings are checked on the final parsed reply only.
  * The developer's calibration removed 13, then 7, false positives (all
    documented). The rules have found mismatches only in synthetic tests.

**Quality gates:**

Run on linux/amd64 unless noted:
* **Focused tests:**
  * agent tests (40);
  * evidence tests (111, including the 40 parameter evidence tests and the
    9 review workflow tests, one of which runs the review page's own
    logic under Node);
  * probe tests (3, against the built binary).
  All pass.
* **Go:**
  * full default suite: all 10 packages pass, including the new
    `mini-seed52-v080` fixture (0.8.0, batch mode `v0.8.0`);
  * race (`go test -race -timeout 90m ./...`): pass;
  * vet (three tag sets) and gofmt: clean.
* **Historical replay:** every fixture replays with every digest
  verified; every recording in the corpus was replayed at 0.9.0-dev.1.
* **Benchmark verify and reproduce:** 66 of 66, 0 failed.
* **The V0.8c pipeline:** run twice, byte-identical and equal to the
  committed files. `check_v08b.py` reproduces the V0.8b review package
  and probe log, with build numbers set aside.
* **V0.9a outputs:** the review page and its PENDING status are rebuilt
  identically. The parameter evidence, run twice, is byte-identical.
  The gate run caught one stale committed file: `parameter-evidence.json`
  had been written before replay `run.json` and `transcript.jsonl` were
  dropped from the hashed inputs, so its `inputs` list still named 192 of
  them. Every check, count and table was identical; the file was
  regenerated, and it now equals both runs.
* **Five cross-builds:** all pass.
* **CI:** linux/amd64, linux/arm64 and darwin/arm64, with the
  cross-platform comparison and the new review and parameter evidence
  step.

**No change to:**
* simulator behaviour, actions, scenarios, scoring, agents, prompts or
  settings;
* any recording, session, golden reference or benchmark result;
* the frozen v1 rules and outputs, repeat v2, refusal-response v1 and v2;
* the review package and its protocol.

No model was contacted.

**Known limitations:**
* **The independent review is pending.** No rule is independently
  validated.
* **The parameter rules are calibrated on the corpus they describe,**
  with no held-out set, and are narrow by design.
* **llm-agent 2 records only the final parsed reply.**
* **No hire stated an hourly rate,** so `rateCents` is untested on
  recordings.

---

## V0.8.0 — release: auditable decision evidence for practice-management agents

Owner decision 2026-10-31:
* V0.8c accepted and closed (ee8aabc, da52a77; CI run 41171062478 green);
* GO for the V0.8.0 release, release-only;
* no V0.8d; no simulator, agent, prompt, model, scoring or scenario
  change;
* V0.9 proposed (ROADMAP §7c), not approved.

**Changed:**
* Version 0.8.0. 0.8.0-dev.3 is registered for replay
  (`mini-seed52-v083`, made from da52a77; batch mode `v0.8_dev3`;
  `internal/session/v083_test.go`). Every earlier batch mode is kept.
* Release notes: V0.8.0, with the release summary (fifteen items), the
  development findings with their denominators, the evidence corrections
  (the four versions; V1-D1 to V1-D5 and RR1-D1), the pending
  independent review, the funding refusals and the paired outcomes, and
  the known limitations.
* README (V0.8 current, findings, limitations), ROADMAP (V0.8 released;
  §7c, V0.9 proposed), DECISION_EVIDENCE (status; §16, the versions and
  how to read them, and the one reproducibility exception), BENCHMARK_SUITE
  and LLM_AGENT (status and limitations).
* `check_v08b.py` now also requires the regenerated review manifest's
  `itemsSha256` to be its own `items.json`'s SHA-256. Its exception,
  the build-version fields of the two frozen V0.8b artifacts only, is
  documented narrowly in DECISION_EVIDENCE §16 and the release notes.
* The development evidence (`docs/benchmark/v08a-dev/`, `v08b-dev/`,
  `v08c-dev/`) is unchanged.

No change to mechanics, randomness, scenarios, actions, accounting,
evaluation, scoring, the harness, the protocols, the policies, the LLM
agents, their prompts or settings, or any evidence rule. No evaluation
was rerun, and no model was contacted.

**Release gates** (linux/amd64 unless noted):
* Clean release tree (only release changes).
* Full default Go suite: all 10 packages with tests pass, including
  `v083_test.go`. The tagged clinical evaluation suites were not rerun
  (unaffected).
* Race: `go test -race -timeout 90m ./...` passes.
* vet (three tag sets) and gofmt: clean.
* Python: agent tests (40, llm-agent 2 and 3), evidence tests (62) and
  probe tests (3, against the built 0.8.0 binary) pass.
* Historical session replay: every fixture, including
  `mini-seed52-v083` (0.8.0-dev.3, batch mode `v0.8_dev3`), replays with
  every digest verified.
* Golden references: outcomes identical (through the migration manifest).
* Suite verify and reproduce: the V0.4 CI benchmark (T90 recomputed), the
  V0.6a suite, the V0.6b, V0.7a and V0.7b suite runs.
* Recorded LLM replay: V0.5a, V0.5b, V0.5c stages a–d, the ten V0.6c and
  ten V0.7c replays (the financial ledger, credit draws, repayments and
  interest included), and the two V0.8c llm-agent 3 replays, every
  attempt's input and validate answer checked. With the suites: 66 of 66
  verify and reproduce commands, 0 checks failed.
* Frozen v1: `check_frozen.py` reproduces the analyzer, tables and traces
  byte for byte, and the evidence in every field but the engine build.
* Frozen V0.8b artifacts: `check_v08b.py` reproduces the review package
  and probe log with only the documented build-version fields set aside.
* The V0.8b and V0.8c pipelines (financial outcomes, probes, probe
  evidence, refusal-response v1 and v2, repeat v2, the scenario analysis
  and the V0.8c analysis), run twice under 0.8.0: byte-identical, and
  equal to the committed files.
* Five cross-builds: linux/amd64, linux/arm64, darwin/arm64,
  darwin/amd64, windows/amd64.
* Committed seed guards (`TestCommittedConfigs`, `TestCommittedSuiteRuns`,
  the holdout guard): pass.
* CI on the release commit: green on linux/amd64, linux/arm64 and
  darwin/arm64, with the cross-platform comparison.

No live LLM evaluation was rerun.

---

## V0.8c — complete decision recording and prospective refusal-response evaluation (0.8.0-dev.3)

Owner decision 2026-10-30: V0.8b accepted and closed (5b6cdba, CI run
41133370321). V0.8c is authorized, including a small controlled live LLM
evaluation (at most eight runs), and repeat detection as a separately
versioned rule. Prompts, models, mechanics and the frozen v1 rules are not
to be changed.

**Declared before any live run** (ee8aabc):
* **llm-agent 3** (`agents/llm/llm_agent3.py`) and its record
  `dreamteam.llm-decision-record/v3` ([LLM_AGENT.md §13](LLM_AGENT.md)):
  * every attempt, its input by SHA-256, output, parse, reason, actions,
    validate answers, refusals and error, the used attempt, and the
    record's completeness;
  * a replay that rebuilds every input and asks validate again;
  * the same messages and decisions as llm-agent 2 on the same answers.
* **The low-liquidity condition:** scenario
  `bench-financial-pressure-ot-lowliquidity`, suite `dreamteam-dev`
  version 4. It is the V0.7c financial-pressure-credit practice with
  $80,000 opening cash, chosen by a deterministic rule on analysis seed
  41900 ([scenario_analysis.py](benchmark/v08c-dev/scenario_analysis.py)).
* **The plan:** [PLAN.md](benchmark/v08c-dev/PLAN.md) and
  [eval.json](benchmark/v08c-dev/eval.json). Two conditions, seeds 41901
  and 41902, 30 days, the two frozen models and three deterministic
  comparators: 20 runs, 8 of them live.
* **Version:** 0.8.0-dev.3, with no engine behaviour changed. 0.8.0-dev.2
  is registered for replay (`mini-seed52-v082`, batch mode `v0.8_dev2`).

**Built after the declaration:**
* **Refusal-response version 2** (`dreamteam.refusal-response/v2`,
  DECISION_EVIDENCE §13):
  * a corrected reader;
  * the response is the next parsed proposal;
  * it reads llm-agent 3 directly;
  * version 1's outputs are unchanged.
  * It found **RR1-D1:** both historical `identical_repeated` refusals of
    V0.8b were a proposal compared with itself.
* **Repeat rule version 2** (`dreamteam.repeat-evidence/v2`,
  DECISION_EVIDENCE §14): repeats by exact action content, with explicit
  abstention, compared with v1's links. It found **V1-D5:** v1 merges
  different actions refused with the same message.
* **The frozen rules on llm-agent 3's records** through an llm-agent 2 view
  (DECISION_EVIDENCE §15), checked field by field against llm-agent 2's
  own records.
* **The evaluation's analysis** (`analysis/evidence/v08c_analysis.py`):
  recording completeness, the plan's 15 primary outcomes with
  denominators, the response categories, and outcomes paired with no
  action.
* **`check_v08b.py`:** the frozen V0.8b review package and probe log
  record their build number. They are checked byte for byte with the
  build numbers set aside, rather than regenerated.

**Results** ([benchmark/v08c-dev/](benchmark/v08c-dev/README.md)):
* **Runs:** 20 of 20 completed on the first attempt, 8 of them live LLM
  runs, with no transport failures.
* **Recording:** every attempt was recorded and every input is verifiable
  at 240 of 240 decision points (270 attempts).
* **Funding refusals:** 10, all in the low-liquidity condition on seed
  41901 (Qwen 8, Gemma 2). There were none in the control or on seed
  41902.
* **Responses to those 10:** repeated 0, modified 2 (only the effective
  date changed), different 3, no action 5, unavailable 0.
* **Financial:** no distress anywhere. Credit was drawn under every
  policy in the low-liquidity condition, no action included.
* **Replay:** every live run replays without the model and equals the live
  run byte for byte, every attempt's input and validate answer included.
* **Historical recordings, read again:**
  * refusal-response v2 classifies 59 of 559 refusals, not 102;
  * repeat v2 can confirm none of v1's 202 repeat links: 194 lack the
    payload, and 8 join different actions.
* **Independent review:** PENDING. No v1 finding is independently
  validated.

**Quality gates:**

Run on linux/amd64 unless noted:
* **Focused tests:**
  * agent tests (40, llm-agent 2 and 3, including the llm-agent 3 record,
    its replay and tamper checks, and the same decisions as llm-agent 2);
  * evidence tests (62, including the 9 new ones: repeat v2's cases and
    abstention, refusal-response v2 and RR1-D1, and the llm-agent 2 view);
  * probe tests (3, against the built binary).
  All pass.
* **Go:**
  * full default suite: all 10 packages pass, including the llm-agent 3
    tests against a mock model;
  * race (`go test -race -timeout 90m ./...`): pass;
  * vet (three tag sets) and gofmt: clean.
* **Historical replay:** every fixture, including the new
  `mini-seed52-v082` (0.8.0-dev.2, batch mode `v0.8_dev2`), replays with
  every digest verified. Every llm-agent 1 and 2 recording replays
  unchanged with `llm_agent.py`.
* **Golden references:** outcomes identical.
* **Benchmark verify and reproduce:** 66 of 66, 0 failed: V0.8b's 62 and
  the two llm-agent 3 replays of the evaluation.
* **Frozen v1:** `check_frozen.py` reproduces it byte for byte.
* **V0.8b package:** `check_v08b.py` reproduces the review package and
  probe log, with build numbers set aside.
* **The V0.8c pipeline:** the scenario analysis, both evaluation replays,
  refusal-response v2, repeat v2 and the evaluation analysis, run twice,
  are byte-identical and equal the committed files. No model was
  contacted.
* **Cross-platform:** the evaluation replays' manifests are identical on
  linux/amd64 and linux/arm64.
* **Five cross-builds:** all pass.
* **CI:** green on linux/amd64, linux/arm64 and darwin/arm64, with the
  cross-platform comparison.

**No change to:**
* simulator behaviour, actions, scoring, llm-agent 1 and 2, prompts or
  settings;
* any earlier recording, session, golden reference or benchmark result;
* the frozen v1 rules or the V0.8b review package.

**Known limitations:**
* **The independent review is pending,** so no v1 finding is
  independently validated, and no precision or recall is claimed.
* **The evaluation is small:** two seeds, two conditions, two models, and
  funding refusals on one seed only. It shows what happened, not rates.
* **Historical recordings** remain mostly unclassifiable; only llm-agent 3
  records every attempt.
* **The outcome analysis is descriptive only.**

---

## V0.8b — evidence validation and financial decision outcomes (0.8.0-dev.2)

Owner decision 2026-10-30: V0.8a accepted and closed (80cfff9, CI run
41104143975). V0.8b is authorized. There is no live LLM inference, no new
mechanic, and no change to recordings, prompts, policies or scoring.

**Built:**
* **The freeze** of `dreamteam.decision-evidence/v1`:
  * `analysis/evidence/frozen-v1.json` holds the analyzer's SHA-256, the
    rules, the source commit, the corpus fingerprint and the output
    hashes;
  * `analysis/evidence/check_frozen.py` checks a later build against it;
  * `docs/benchmark/v08a-dev/FREEZE.md` records the unsupported claim
    types, the coverage limitations and the defects V1-D1 to V1-D4.
* **The independent review package** (`analysis/evidence/review_package.py`,
  `docs/benchmark/v08b-dev/review/`):
  * a seeded, stratified sample of 98 items, with a protocol and a form;
  * `review_analysis.py` analyses complete, declared-independent forms
    only.
  * **Status: PENDING.** No independent reviewer was available, so no
    label exists.
* **Funding-refusal probes** (`analysis/probes/funding_probes.py`,
  `docs/benchmark/v08b-dev/probes.json`, `PROBES.md`): probes A–J are
  scripted through the agent protocol on the solvency scenarios, seed
  41801 (fresh). All expectations were met.
* **The frozen rules on the probes:** `analysis/probes/probe_evidence.py`.
* **The extension `dreamteam.refusal-response/v1`**
  (`analysis/evidence/refusal_response.py`, DECISION_EVIDENCE §12),
  applied to the probes and to the historical recordings.
* **Paired financial and operational outcomes**
  (`analysis/evidence/financial_outcomes.py`, `FINANCIAL_OUTCOMES.md`):
  140 V0.6c and V0.7c pairs.
* **The proposed V0.8c recording contract**
  (`docs/benchmark/v08b-dev/RECORDING_CONTRACT.md`): designed, not
  deployed.
* **CI** regenerates every V0.8b output from the replays and the probes,
  requires each to equal the committed one, and compares them across
  platforms. The frozen v1 output is checked with `check_frozen.py`.
* **Version** 0.8.0-dev.2. It registers 0.8.0-dev.1 for historical replay
  (batch mode `v0.8_dev1`, `mini-seed52-v081`, `v081_test.go`).

**Results:**
* **Frozen v1:** reproduced under 0.8.0-dev.2: the analyzer, tables and
  traces byte for byte, and the evidence in every field but the engine
  build.
* **Probes:**
  * The four commitment kinds are refused when unfunded.
  * A revised affordable commitment is accepted and reserved, and the
    reservation then refuses an extension that fitted before it.
  * A cost reduction is accepted with negative headroom.
  * The funding boundary is exact, and with credit it is exactly $50,000
    higher.
  * v1's repeat rule misses the identical hire refused on two days
    (V1-D1).
* **Refusal responses:**
  * Probes: 7 classified as scripted. Read in the llm-agent 2 format, 3 of
    them are `evidence_unavailable`.
  * History: 559 refusals, none `insufficient_funds`. 457 are
    `evidence_unavailable`, 452 of them only because the refused attempt's
    actions are not recorded.
* **Outcomes:** described per suite run, never combined. Stated intent,
  submitted actions and executed actions are counted separately.

**Quality gates:**

Run on linux/amd64 unless noted:
* **Focused tests:** the evidence tests (53, including the 11 new ones:
  the freeze guard, normalization, every refusal-response category, both
  recording formats, sampling determinism and the review analysis's
  refusals) and the probe tests (3, against the built binary) pass.
* **Go:**
  * full default suite: all 10 packages pass;
  * race (`go test -race -timeout 90m ./...`): pass;
  * vet (three tag sets) and gofmt: clean.
* **Python:** agent tests (28) pass.
* **Historical replay:** every fixture, including the new
  `mini-seed52-v081` (0.8.0-dev.1, batch mode `v0.8_dev1`), replays with
  every digest verified.
* **Golden references:** outcomes identical.
* **Benchmark verify and reproduce:** 62 of 62, 0 failed, covering the
  same suites and all 26 recorded-LLM replays as V0.8a.
* **Frozen v1:** `check_frozen.py` reproduces the analyzer, tables and
  traces byte for byte, and the normalized evidence exactly.
* **The V0.8b pipeline:** the review package, financial outcomes, probes,
  probe evidence and refusal responses, run twice, are byte-identical and
  equal the committed files. Every probe expectation is met. No model
  was contacted.
* **Five cross-builds:** all pass.
* **CI:** green on linux/amd64, linux/arm64 and darwin/arm64, with the
  cross-platform comparison of the evidence and the V0.8b outputs.

**No change to:**
* simulator behaviour, actions, scenarios, scoring, agents, prompts or
  settings;
* any recording, session, golden reference or benchmark result;
* the frozen v1 rules.

No model was contacted.

**Known limitations:**
* **The independent review is pending,** so the v1 findings' correctness
  is not independently assessed.
* **The probes** cover one scenario and seed. `refusal_no_longer_applicable`
  is tested only in unit tests.
* **Historical refusal responses** are mostly unclassifiable until the
  recording contract changes.
* **The outcome analysis is descriptive only.**

---

## V0.8a — agent decision consistency and evidence analysis (0.8.0-dev.1)

Owner decision 2026-10-30: V0.7.0 accepted and closed (08ae85f, tag
v0.7.0, CI run 41056460493). V0.8a is authorized as a read-only,
deterministic framework for analysing historical agent decisions and their
stated reasons. It measures whether an explanation is consistent with the
actions, not whether the decisions were good. There is no new LLM run, no
simulator change, no agent change and no LLM judge.

**Built:**
* **`analysis/evidence/decision_evidence.py`** (analyzer `decision-evidence
  1`, standard-library Python), with its tests
  `analysis/evidence/test_decision_evidence.py` (42).
  * **Inputs:** the committed recordings, and the deterministic replays CI
    already runs.
  * **Prompts:** it rebuilds every prompt and memory text from the replay
    transcript with the agent's own functions, read only, and checks both
    against the recorded SHA-256.
  * **Rules:** it applies six rules with three statuses.
  * **Outputs:** `evidence.json`, `traces.jsonl` and `tables.md`.
* **The contract `dreamteam.decision-evidence/v1`, the evidence inventory,
  the coverage matrix, the rules and the abstention flags:**
  `docs/DECISION_EVIDENCE.md`.
* **The corpus** (`docs/benchmark/v08a-dev/corpus.json`): V0.5c, V0.6c and
  V0.7c primary, V0.5a and V0.5b secondary. 90 recordings, 3,972 decision
  points.
* **The results:** `docs/benchmark/v08a-dev/` (README, `evidence.json`,
  `tables.md`).
* **CI** runs the tests and the analysis twice after the replays. It
  compares the output with itself and with the committed evidence, and
  compares the evidence and the traces across the three platforms.
* **The version** is 0.8.0-dev.1. It registers 0.7.0 for historical replay
  (batch mode `v0.7.0`, `testdata/sessions/mini-seed52-v070`,
  `internal/session/v070_test.go`).

**Evidence inventory:**
* Every llm-agent 2 recording (V0.5b–V0.7c) records the same fields: the
  reason and proposed actions of the final parsed reply, validation,
  refusals, errors, outcome, batches, digests, and the prompt and memory
  SHA-256.
* **Prompts:** 3,965 of 3,965 rebuilt and verified.
* **V0.5a** (llm-agent 1) has no memory or refusal fields, and its prompt
  cannot be rebuilt.
* **Not in the repository:** earlier attempts' replies.
* **Details requests:** none was made in any recording.

**Results** (DECISION_EVIDENCE §6, the report §2):

| Rule | Statements checked | Consistent | Supported | Unresolved | Insufficient evidence |
|---|---|---|---|---|---|
| Explicit action omitted | 525 | 480 | 5 | 40 | 0 |
| Explicit opposite action | 672 | 669 | 0 | 3 | 0 |
| Financial amount mismatch | 235 | 206 | 3 | 26 | 0 |
| Repeated invalid proposal | 342 | 235 | 107 | 0 | 0 |
| Stated action not executed | 352 | 307 | 45 | 0 | 0 |
| Factual observation mismatch | 5,139 | 5,101 | 27 | 2 | 9 |

* **Opposite actions:** none was found.
* **Amounts:** the V0.6c retainer misread ($150k for `monthlyRetainerCents`
  150000) is found mechanically, as cents read as dollars.
* **Stated actions that failed** were refused by validate, or not submitted
  because another action in the reply was refused. None was refused when
  applied.
* **Repeats:** most repeats at a later point (146 of 159, memory on)
  happened while the refusal was in the memory shown. These are patterns
  only, not a claim about why.
* **The V0.7c staffing-shortage run** shows no omitted, opposite or
  unexecuted commitment. It shows one payroll amount and one weekday that
  disagree with the prompt.

No count is a measure of model quality.

**Quality gates:**

Run on linux/amd64 unless noted:
* **Focused tests:** the analysis's tests (42) pass, including:
  * each rule with negative controls: similar but distinct targets,
    qualified and scoped statements of no action, subsets and earlier
    values;
  * cents read as dollars;
  * missing observation and missing reason;
  * the V0.5a format;
  * stable identifiers and ordering;
  * no change to any input, and no socket opened.
* **Go:**
  * full default suite: all 10 packages pass;
  * race (`go test -race -timeout 90m ./...`): pass;
  * vet (three tag sets) and gofmt: clean.
* **Python:** agent tests (28) pass.
* **Historical replay:** every fixture, including the new
  `mini-seed52-v070` (0.7.0, batch mode `v0.7.0`), replays with every
  digest verified.
* **Golden references:** outcomes identical.
* **Benchmark verify and reproduce:** 62 of 62, 0 failed. They cover the
  V0.4 CI benchmark (T90 recomputed), the V0.6a and V0.6b suites, the
  V0.7a and V0.7b suite runs, and all 26 recorded-LLM replays (V0.5a,
  V0.5b, V0.5c a–d, ten V0.6c, ten V0.7c).
* **The analysis:** run twice, its evidence, traces and tables are
  byte-identical, and they equal the committed `evidence.json` and
  `tables.md`. It made no model call.
* **Five cross-builds:** all pass.
* **CI:** green on linux/amd64, linux/arm64 and darwin/arm64, with the
  cross-platform comparison of the decision evidence and traces.

No clinical suite or live LLM evaluation was run.

**No change to:**
* simulator behaviour, actions, scenarios or scoring;
* the agents, their prompts or their settings;
* any recording, session, golden reference or benchmark result.

No model was contacted.

**Known limitations:**
* **Calibration:** the rules were calibrated on this corpus and reviewed
  by the author, not independently, and no held-out recordings exist.
* **Reach:** the patterns are narrow. Recall is traded for precision, so
  a hedge or a revision blocks a statement.
* **Coverage:**
  * only the final parsed reply is checked;
  * observation checks compare with the prompt's summary;
  * repeat identity uses the engine's message with dates normalized.
* **Consequences** are next-day observations, never attributed.

---

## V0.7.0 — release: financial realism for practice-management simulation

Owner decision 2026-10-30:
* V0.7c accepted and closed (CI green on ad6d25c, run 41015378407);
* GO for the V0.7.0 release, release-only;
* no V0.7d, no model retuning and no further financial scenarios before
  the release;
* V0.8 proposed as agent decision quality (ROADMAP §7b), not approved.

**Changed:**
* Version 0.7.0. 0.7.0-dev.3 is registered for replay
  (`mini-seed52-v073`, made from ad6d25c; batch mode `v0.7_dev3`;
  `internal/session/v073_test.go`). Every earlier batch mode is kept.
* Release notes: V0.7.0, with the release summary, the V0.7c development
  findings (the 90-day staffing-shortage run as one observation), the
  solvency limitation (no `insufficient_funds` refusal in any V0.7c run,
  so agents' responses to funding refusals are unobserved) and the known
  limitations.
* README (V0.7 current, findings, limitations), ROADMAP (V0.7 released;
  §7b, V0.8 proposed), the overtime and solvency design notes (status,
  and what V0.7c showed), BENCHMARK, BENCHMARK_SUITE and LLM_AGENT
  (status and limitations).
* The development evidence (`docs/benchmark/v07a-dev/`, `v07b-dev/`,
  `v07c-dev/`) is unchanged.

No change to mechanics, randomness, scenarios, actions, accounting,
evaluation, scoring, the harness, the protocols, the policies, the LLM
agent, its prompt or its settings. No evaluation was rerun.

**Release gates** (linux/amd64 unless noted):
* Clean release tree.
* Full default Go suite: all 10 packages with tests pass. The tagged
  clinical evaluation suites were not rerun (unaffected).
* Race: `go test -race -timeout 90m ./...` passes.
* vet (three tag sets) and gofmt: clean.
* Python agent tests: 28 pass.
* Historical session replay: every fixture, including
  `mini-seed52-v073` (0.7.0-dev.3, batch mode `v0.7_dev3`), replays with
  every digest verified.
* Golden references: outcomes identical (four references, through the
  migration manifest). The V0.6 suite and scenario files, and the V0.5 and
  V0.6 evidence, are unchanged since `v0.6.0`.
* Suite verify and reproduce: V0.4 CI benchmark (T90 recomputed), V0.6a
  suite, V0.6b suite run, V0.7a suite run (version 2), V0.7b suite run
  (version 3). 0 failed.
* Recorded LLM replay: V0.5a, V0.5b, V0.5c stages a–d, the ten V0.6c
  per-entry replays and the ten V0.7c per-stage, per-entry replays (the
  financial ledger, credit draws, repayments and interest included), each
  verified and reproduced: 52 of 52 commands, 0 checks failed.
* Five cross-builds: linux/amd64, linux/arm64, darwin/arm64,
  darwin/amd64, windows/amd64.
* Committed seed guards (`TestCommittedConfigs`, `TestCommittedSuiteRuns`,
  the holdout guard): pass.
* CI on the release commit: green on linux/amd64, linux/arm64 and
  darwin/arm64, with the cross-platform comparison.

No live LLM evaluation was rerun.

---

## V0.7c — frozen agents under overtime and solvency economics (0.7.0-dev.3)

Owner decision 2026-10-29:
* V0.7b is accepted, once CI on 9e73520 was green (run 40962984422).
* V0.7c is the last development evaluation before V0.7.0. No simulator
  mechanic is added.

**Declared before any run** ([plan](benchmark/v07c-dev/PLAN.md),
`v07c-plan/1`):
* **Suite:** `dreamteam-dev` version 3, the five `-credit` entries:
  radiologist overtime paid, a $50,000 credit line at 10%, the funding rule.
* **Policies:**
  * `noop`, `greedy` and `conservative-v2`;
  * the LLM agent (llm-agent 2, prompt v2, memory on) with
    Qwen3.6-35B-A3B and with Gemma 4 26B-A4B, both with the V0.6c settings,
    unchanged.
* **Stages:**
  * A: seeds 41601 and 41602, 30 days, 50 runs;
  * B: seed 41701, 90 days, 25 runs, independent of stage A.
* **Pipeline check:** seed 41600, 3 days, one entry, before the plan. Not
  a result.
* **Analysis:** paired with no action. Actions are counted as proposed,
  refused, accepted and executed, with refusals for funds and what the
  agents did next.
* **Version 0.7.0-dev.3:** no engine change. 0.7.0-dev.2 is registered for
  replay (`mini-seed52-v072`, batch mode `v0.7_dev2`).

**Results** ([README](benchmark/v07c-dev/README.md), tables per stage):
* **Runs:** 75 of 75 completed on the Pi, with no failure and no rerun:
  * stage A: 50 runs in 1 h 26 min;
  * stage B: 25 runs in 3 h 59 min.
* **Starting states** were identical across the five policies of every
  entry and seed.
* **Funding:** no action by any policy was refused `insufficient_funds`.
  * One run of 75 drew on the credit line: Qwen, staffing shortage, 90
    days. At most $17,200 was drawn, with $14 interest and no unfunded
    shortfall.
  * This repeats V0.7b's finding: the practices' liquidity leaves the
    funding rule mostly unexercised.
* **Baselines:**
  * `greedy`: lower profit with better TAT throughout ($30–$1,387 over 30
    days, $29,022–$33,550 over 90).
  * `conservative-v2`: changes only in stable.
* **Qwen** acted in all 15 runs, mostly by extending shifts (14 runs).
  Paired with no action:
  * every run lowered profit and lowest cash;
  * all but one lowered breaches.
  * In stable it executed 19, 9 and 5 actions, for profit −$9,401,
    −$4,637 and −$51,560.
* **Qwen, staffing shortage, 90 days:**
  * It hired two technologists, then from 11-27 gave notice to them and
    to every other technologist and the front desk. Its recorded reasons
    were "Cash is $0 and critically low … avoid insolvency" and "This is a
    survival move".
  * Paired with no action: profit −$181,589, exams −867, lost orders
    +1,450.
  * No reason in that run mentions the credit line in the summary.
* **Gemma** changed nothing in financial pressure or stable. It used
  teleradiology, and in stage B extended shifts in demand growth (27) and
  staffing shortage (53). Its paired profit was $0 to −$22,096.
* **Every pair that improved service lowered profit.** No pair improved
  service and raised profit.
* **Refusals:** every refusal was at the agents' validate step, for timing
  or form. Repeated invalid proposals reached 15 (Qwen) and 27 (Gemma)
  in a run. There were no agent infrastructure errors.
* **Replay:**
  * all 30 LLM runs replay without a model;
  * the replays are byte-identical to the live runs (arm64 live, amd64
    replay): decisions, sessions and results, with the ledger and credit;
  * replay verify: 68 checks per stage A entry, 41 per stage B entry, 0
    failed, on both platforms;
  * reproduce: 47 checks per stage A suite and 29 per stage B suite, 0 failed, on linux/arm64 (Pi) and linux/amd64;
  * the ten replay manifests are byte-identical on linux/arm64 and
    linux/amd64.
* **Live results:** `verify` on the Pi: stage A 464 checks, stage B 239,
  0 failed.

**Gates:**
* `go test ./...` on linux/amd64: 10 packages pass. That includes:
  * golden references identical under 0.7.0-dev.3;
  * every historical session replay, `v071` and `v072` included;
  * the committed configurations and seed guards.
* `go test -race -timeout 90m ./...`: 10 packages pass.
* `go vet` (default, `calibration` and `acceptance` tags) and `gofmt`:
  clean.
* Cross-builds: linux/amd64, linux/arm64, darwin/arm64, darwin/amd64 and
  windows/amd64.
* Python tests pass (28).
* CI replays the ten V0.7c configurations on linux/amd64, linux/arm64 and
  darwin/arm64 and compares their manifests.
* The tagged clinical evaluation suites were not rerun (unaffected), and
  no historical LLM experiment was rerun.

**Limitations:**
* two models, one prompt, three seeds;
* the funding rule bound nowhere, so the questions about refusals for
  funds have no observations;
* intent is reported only where a recorded reason states it;
* live latency and tokens are not reproducible, but the decisions are.

## V0.7b — financial solvency (0.7.0-dev.2)

Owner decision 2026-10-29:
* V0.7a is accepted and closed.
* V0.7b models a practice's ability to finance operations, not bankruptcy
  at zero cash: available cash, a credit facility and financial distress.
* Consequences go through existing constraints (refusing discretionary
  commitments), never a shutdown.
* Manager overtime is deferred.

**Design review** ([V0.7_SOLVENCY_DESIGN.md](V0.7_SOLVENCY_DESIGN.md)):
* **Cash timing.** Cash leaves only at 00:00: payroll every 14 days,
  severance on paydays, payables on the 1st. It arrives only with payer
  remittances.
* **Committed versus discretionary.** Current liabilities (accrued
  payroll, payables, severance payable) are on the ledger. Each staffing
  action's preview already lists the money it guarantees. So committed
  obligations and discretionary commitments are separable.
* **Roster changes.** A roster can never exceed the contract, whose
  guarantee is already owed, so `set_roster` commits no new money. This
  was found while testing and recorded in the design.
* **A measurement before the design** (seed 41401, no action): cash stays
  positive in all five practices. Cash less what is owed starts near
  $60,000 and falls to $2,504 by day 90 in financial pressure. The limit
  on spending is money already owed, not overdraft.

**Declared first** (28b9a27, before any run on seeds 41501–41503):
* **The scenario setting** `finance.solvency` (`creditLimitCents` 0 to
  $1,000,000, `creditRateBP` 0–5,000), compiled to finance configuration.
  From the takeover:
  * **Draws.** A disbursement that would leave cash below zero draws on
    the line up to its limit. Beyond the limit, cash goes negative:
    financial distress, logged, with the obligation still paid.
  * **Interest** accrues daily on the balance drawn, rounded half up, to
    payables. Cash above zero repays the line at midnight.
  * **Funding rule.** A hire, salary increase, teleradiology contract or
    paid extension is refused `insufficient_funds` unless it fits in the
    funding headroom: cash plus undrawn credit, less current liabilities
    and reserved commitments.
  * **Reservations** are released evenly over each commitment's period.
  * **One check.** Preview, validation and application share it.
* **Historical states are untouched.**
  * The solvency state is one pointer and two appended accounts, encoded
    only when used.
  * Credit flows are financing, outside collections and disbursements.
    The window's cash bridge includes them.
  * New observation, preview and result fields are omitted when empty.
* **Variants.** Ten bundled variants (`-ot-credit`: $50,000 at 10%;
  `-ot-nocredit`) and suite `dreamteam-dev` version 3.
* **Tests** (`internal/sim/solvency_test.go`, `internal/session/solvency_test.go`):
  * funding: from cash, exactly and a cent short; from credit; with zero
    credit;
  * atomic batches; cuts at negative headroom; a reservation's release day
    by day;
  * extensions;
  * the hire and teleradiology commitments equal the preview's
    guaranteed lines;
  * a 60-day run with no cash and a $20,000 line: the limit reached once,
    distress logged, payroll identical to the run without the line, bills
    plus interest, one interest entry a day to the cent, repayment, every
    invariant at every midnight;
  * a non-binding line identical to no line, digest for digest;
  * save, load and continue;
  * scenario validation, and each variant equal to its base except for the
    setting;
  * suite version 3.
* **The characterization plan** (`v07b-plan/1`) and analysis script.
* **Version.** 0.7.0-dev.2. 0.7.0-dev.1 is registered for replay
  (`mini-seed52-v071`, batch mode `v0.7_dev1`).

**Characterization** ([benchmark/v07b-dev/](benchmark/v07b-dev/README.md);
`noop`, `greedy` and `conservative-v2`; suite version 2 and both version 3
variants; seeds 41501–41503; 90 days; 135 runs, all complete):
* **Amendment 1** (after the runs, before the findings) fixed the
  analysis script's status string. It also measures distress by the
  lowest cash, because the results keep no log.
* **`noop` and `conservative-v2` never bind.** Their 60 version 3 runs are
  identical to version 2.
  * `noop` commits nothing and never fell below $15,919.
  * `conservative-v2`'s profit guardrail is stricter than the funding
    rule.
* **`greedy` binds:**
  * **with a $50,000 line**, only in financial pressure: 3 late raises
    refused, and one draw of $12,584, repaid at $8 of interest, where
    version 2 went to −$12,584 unfinanced;
  * **with no credit**, in every practice: 65 refusals in 14 of 15 runs,
    profit +$740 to +$12,658, up to 6 more breaches and 24 minutes on TAT
    p90.
* **One distress** (financial pressure 41502, no credit): a payroll
  already owed exceeded cash by $1,981 even after 9 refusals. It was paid
  and recorded, and recovered the same day.
* **Starting states** were identical in all 15 entry-seed pairs across
  the three variants. Every pair is the model's effect alone.
* **Integrity:**
  * `verify`: 419 of 419 and 824 of 824;
  * `reproduce`: 281 of 281 and 551 of 551 on both linux/amd64 and linux/arm64 (Raspberry Pi).

**Gates:**
* `go test ./...` on linux/amd64: 10 packages pass. That includes:
  * golden references identical under 0.7.0-dev.2;
  * every historical session replay, `v060` and `v071` included;
  * the committed configurations and seed guards.
* `go test -race -timeout 90m ./...`: 10 packages pass.
* `go vet` (default, `calibration` and `acceptance` tags) and `gofmt`:
  clean.
* Cross-builds: linux/amd64, linux/arm64, darwin/arm64, darwin/amd64 and
  windows/amd64.
* Python tests pass.
* CI green on the declaration commit (28b9a27). CI now also runs a 21-day
  suite version 3 smoke test with `verify` and `reproduce` on every
  platform.
* The tagged clinical evaluation suites were not rerun (unaffected).
* No LLM agent was run.

**Limitations:**
* development seeds and deterministic baselines only;
* the credit line is drawn in one run of 135, so the credit mechanics rest
  mainly on the financial tests;
* the funding test covers 30 days of a raise;
* distress only refuses new commitments;
* manager overtime is still unpaid.

## V0.7a — radiologist overtime costing (0.7.0-dev.1)

Owner decision 2026-10-29:
* V0.6.0 accepted and closed.
* V0.7 is simulator realism: V0.7a overtime economics, V0.7b solvency,
  V0.7c re-characterization and frozen-agent evaluation, then V0.7.0.
* V0.7a starts with a compatibility design review, and is implemented
  only if historical behaviour can be preserved.

**Design review** ([V0.7_OVERTIME_DESIGN.md](V0.7_OVERTIME_DESIGN.md)):
* **What the engine already records.** Overtime is the time on duty past
  the rostered end, recorded per employee and day when they go home. It
  includes approved extensions and hold-over, and an extension that is
  never worked records nothing.
* **What an extension adds.** Radiologist extensions add on-site presence
  (contrast supervision), not reading blocks.
* **The model chosen: worked overtime, in the engine's own measure.** It
  is the same time hourly staff are paid for. Idle time inside an
  extension is paid, because there is no early release.
* **The rate.** The salary's hourly equivalent, annual ÷ (52 ×
  `weeklyOvertimeHours`), straight time, with no new rate parameter.
* **Compatibility.** A compatible design exists, so implementation went
  ahead.

**Declared first** (790972c, before any run on seeds 41401–41403):
* **The scenario policy** `radiologistOvertime`: `"unpaid"` (or omitted,
  as in every scenario before V0.7), or `"salary_rate"`.
  * It compiles to finance configuration, outside the state digest.
  * A first attempt in `Policies` moved every state digest, because the
    state keeps a digested copy of the operating policies. The historical
    session tests caught it, and it was moved.
* **The charge.** Floor per radiologist-day, plus benefits, posted when
  they go home (entry kind `rad_overtime`) and paid at payroll. The
  overtime cost metrics include it. The `extend_shift` message gives an
  estimate.
* **Five bundled `-ot` variants and suite `dreamteam-dev` version 2.**
  Version 1 is unchanged, with the same fingerprints.
* **Tests:**
  * financial tests (`internal/sim/radovertime_test.go`): no extension; an
    extension worked, to the cent; two extensions charged once; an
    extension past close, partly idle; two weeks with two radiologists,
    every charge reconciled; revenue, salary and hourly wages unchanged;
    cash after payroll;
  * a session saved and replayed under `"salary_rate"`, and `"unpaid"`
    identical to omitted;
  * scenario validation, and each variant equal to its base except for the
    policy;
  * suite version 2.
* **The characterization plan** (`v07a-plan/1`) and analysis script.
* **Version.** 0.7.0-dev.1. 0.6.0 is registered for replay
  (`mini-seed52-v060`, batch mode `v0.6.0`).

**Characterization** ([benchmark/v07a-dev/](benchmark/v07a-dev/README.md);
`noop`, `greedy` and `conservative-v2`, both suite versions, seeds
41401–41403, 90 days, 90 runs, all complete):
* **Hold-over is common, even with no action.** Radiologists are held
  over (contrast supervision) for 39–69 hours in 90 days. Under version 2
  that costs $9,035–$15,831, plus benefits.
* **Profit with no action falls by $11,294–$19,788.** Reporting backlog
  and demand growth go from near break-even to losses of $5,039–$24,534.
  The stable practice keeps $4,264–$10,626 instead of $16,349–$26,593.
* **`noop` and `greedy`** took identical actions, with identical
  operations, service and revenue. Only money differs.
* **`conservative-v2`** acted differently in 7 of 15 run pairs:
  * fewer, later raises and more undos, as its profit guardrail and review
    respond to the lower profit;
  * 11–40 more breaches in those runs.
* **Starting states.** Versions 1 and 2 never start from the identical
  state: the warm-up's overtime is in version 2's opening accrued payroll
  ($974–$1,729 of cash).
* **T90** NO GO for all nine version 1 runs; not applicable in version 2.
* **Integrity:**
  * `verify`: 437 of 437 (version 1, with T90) and 419 of 419 (version 2);
  * `reproduce` on linux/amd64: 290 of 290 and 281 of 281;
  * `reproduce` on linux/arm64 (Raspberry Pi): 290 of 290 and 281 of 281.

**Gates:**
* `go test ./...` on linux/amd64: 10 packages pass. That includes:
  * golden references identical under 0.7.0-dev.1;
  * every historical session replay, `v060` included;
  * the committed configurations and seed guards.
* `go test -race -timeout 90m ./...`: 10 packages pass.
* `go vet` (default, `calibration` and `acceptance` tags) and `gofmt`:
  clean.
* Cross-builds: linux/amd64, linux/arm64, darwin/arm64, darwin/amd64 and
  windows/amd64.
* Python tests pass.
* CI green on the declaration commit. CI now also runs a 21-day suite
  version 2 smoke test with `verify` and `reproduce` on every platform.
* The tagged clinical evaluation suites were not rerun (unaffected).
* No LLM agent was run.

**Limitations:**
* development seeds and deterministic baselines only;
* no baseline extends a shift, so agents' response to the cost is V0.7c's;
* idle time inside an extension is paid;
* radiologists only (the operations manager's overtime is still unpaid);
* straight time;
* still no insolvency consequence (V0.7b).

---

## V0.6.0 — release: a reproducible benchmark for practice-management agents

Owner decision 2026-10-29:
* V0.6c accepted (CI green on 132413c);
* GO for the V0.6.0 release, release-only;
* no V0.6d, and no tuning of models, prompts, policies, scenarios or
  metrics;
* V0.7 proposed as simulator realism and agent decision-quality
  evaluation (ROADMAP §7a), not approved.

**Changed:**
* Version 0.6.0. 0.6.0-dev.3 is registered for replay
  (`mini-seed52-v063`, batch mode `v0.6_dev3`,
  `internal/session/v063_test.go`).
* README, release notes, ROADMAP (V0.6 released, V0.7 proposed), and the
  status and known limitations of BENCHMARK_SUITE (§6 now opens with the
  release's limitations), BENCHMARK and LLM_AGENT.
* The development evidence is unchanged, including V0.6a's failed
  demand-growth check and V0.6b's prospectively declared diagnostic.

No change to mechanics, randomness, scenarios, actions, accounting,
evaluation, scoring, the harness, the protocols, the policies, the LLM
agent, its prompt or its settings. No evaluation was rerun.

**Release gates** (linux/amd64 unless noted):
* Clean release tree.
* Full default Go suite: all 10 packages with tests pass. The tagged
  clinical evaluation suites were not rerun (unaffected).
* Race: `go test -race -timeout 90m ./...` passes.
* vet (three tag sets) and gofmt: clean.
* Python agent tests: pass.
* Historical session replay: every fixture, including
  `mini-seed52-v063`, replays with every digest verified.
* Golden references: outcomes identical (four references, through the
  migration manifest).
* Suite verify and reproduce: V0.4 CI benchmark, V0.6a suite, V0.6b suite
  run. 0 failed.
* Recorded LLM replay: V0.5a, V0.5b, V0.5c stages a–d and the ten V0.6c
  per-entry replays, each verified and reproduced. 38 of 38 checks
  passed, 0 failed.
* Five cross-builds.
* Committed seed guards (`TestCommittedConfigs`, `TestCommittedSuiteRuns`,
  the holdout guard): pass.
* CI on the release commit: green on linux/amd64, linux/arm64 and
  darwin/arm64, with the cross-platform comparison.

No live LLM evaluation was rerun.

---

## V0.6c — conservative baseline v2 and frozen-agent evaluation (0.6.0-dev.3)

Owner decision 2026-10-28: V0.6b accepted (CI green on 8bd21a3), with
`conservative` v1's inactivity outstanding. V0.6c:
* adds a second, distinct financially conservative baseline, keeping v1
  as a historical baseline;
* evaluates two frozen LLM agents across the suite (Qwen3.6-35B-A3B and
  Gemma 4 26B-A4B; the V0.5 agent, `llm-agent-prompt/v2`, settings
  identical except the model).

The LLMs need not outperform the baselines.

**Declared first** (4b9ed5c, CI green, before any run on its seeds):
* `conservative-v2`, its specification (BENCHMARK_SUITE §7.2) and
  synthetic qualification;
* the plan `v06c-plan/1` with every seed, model, setting and stage
  ([benchmark/v06c-dev/PLAN.md](benchmark/v06c-dev/PLAN.md)), and
  `analyze.py`;
* version 0.6.0-dev.3 (0.6.0-dev.2 registered for replay:
  `mini-seed52-v062`, batch mode `v0.6_dev2`).

No amendment.

**Built:**
* **`conservative-v2`** (`internal/bench/conservative2.go`):
  * **Signal:** reads urgent and soon report breaches from the public log
    once a day, and acts on a sustained problem (urgent on 3 of 7 days, or
    soon on 5 of 7).
  * **Guardrails:** trailing-30-day profit at or above zero after the
    preview's added salary, cash at least $50,000, at most 10 points above
    the scenario levels, one change at a time, from the earliest lawful
    date, never displacing a booked patient.
  * **Review:** 14 days after a raise takes effect it checks whether
    breaches per day fell by a quarter. If not, it steps the raise back and
    raises no more.
  * **Step back:** when the practice loses money or service is clear.
  * **Qualification:** all eight requirements in
    `internal/bench/conservative2_test.go`, with synthetic observations
    from a fake protocol endpoint.
* **Analysis:** `analyze.py` (tables) and `breaches.py` (the baseline's
  breach records, read back from its transcripts).
* **CI** replays the comparison's recorded decisions, one suite run per
  stage and entry, and compares the manifests on three platforms.

**Phase 1, qualification** (seed 41101, 90 days, no action and
`conservative-v2`):
* **Stable:** it acted only here. It raised Dr. Blake 52.5% → 57.5%. The
  review found the raise had helped, but the practice was losing money,
  so it stepped back; it raised again a month later.
  * Paired with no action: profit −$3,370, TAT p90 −49 minutes,
    breaches −12.
* **Elsewhere:** sustained problems on 6–57 of 84 points, but the trailing
  30 days lost money on 79–90 of 90, so the guardrail held. In demand
  growth the affordability check declined two previewed raises.
* **Integrity:** `verify -t90` 108 of 108; `reproduce` on linux/amd64 73
  of 73, manifest byte-identical.

**Phase 2, 30 days** (seeds 41201–41203; noop, greedy, `conservative-v2`,
Qwen, Gemma; 75 runs in 2 h 50 min):
* **Completion:** 75 of 75; no failure, no infrastructure error, no rerun.
* **Qwen:** acted at 108 of 450 decision points (189 changes accepted,
  116 refusals, 13 fallbacks).
  * It reduced breaches in 14 of 15 runs.
  * At a large cost in staffing shortage (−$7,683 to −$30,447, with
    hires, full radiologist allocations and teleradiology) and once in
    reporting backlog (−$16,102).
  * It intervened on every stable seed (−$160 to −$2,923).
* **Gemma:** acted at 25 points (32 changes, 34 refusals, 13 fallbacks).
  * No action at all in reporting backlog, and none effective in
    financial pressure.
  * Its usual reason: the next scheduled radiologist "should address the
    immediate backlog".
  * Staffing shortage: shift extensions and teleradiology, plus repeated
    refused hires.
* **`conservative-v2`:** acted in stable on two seeds, and once each in
  reporting backlog and demand growth (too late to take effect); otherwise
  held back by its guardrails.
* **Integrity:** `verify` 689 of 689. Every LLM run (30 of 30) replayed
  without a model with byte-identical decisions, session, result, memory
  and final digests; replay manifests byte-identical on a rerun.

**Phase 2, 90 days** (seed 41301; the same five policies; T90 for
reporting backlog; 25 runs in 4 h 21 min):
* **Completion:** 25 of 25; no failure, no infrastructure error, no rerun.
  T90 NO GO for every policy.
* **Greedy:** its twelve raises cost $27,266–$33,940 in every entry.
* **`conservative-v2`:**
  * Stable: raised, then retreated at the review (−$2,301, breaches −53).
  * Demand growth: raised once (+$2,191, breaches −5).
  * The other three entries: no action.
* **Qwen:** 188 changes accepted.
  * Staffing shortage: −$137,507 for breaches −67, TAT p90 761 → 131
    minutes and lost orders −113. It kept extending shifts with its reason
    noting cash of $3,005.
  * Financial pressure: it terminated a technologist (lost orders +361,
    −$43,934). Its reasons misstate the teleradiology retainer as
    "$150k/month" (the observation: 150,000 cents).
  * Stable: −$58,693 for breaches −206.
* **Gemma:** one change in 450 decision points: a Sunday teleradiology
  contract in demand growth (+$2,098, breaches −97).
* **Integrity:** `verify -t90` 249 of 249. All 10 LLM runs
  replayed byte-identically; replay manifests identical on a rerun.

**Gates:**
* Full default Go suite on linux/amd64: every package passes.
* Race on linux/amd64 (`-timeout 90m`): `internal/bench`,
  `internal/scenario`, `internal/session`, `internal/agent`,
  `cmd/dreamteam` pass. The default 10-minute limit is too short under the
  race detector.
* vet (three tag sets) and gofmt: clean.
* Python agent tests: pass.
* Five cross-builds.
* Committed configurations load through the holdout guard.
* CI on the declaration commit green.

**Limitations:**
* development seeds only (three at 30 days, one at 90);
* one prompt and one set of settings;
* radiologist overtime is unpaid in the simulator;
* no inference cost (no prices configured);
* `conservative-v2` is mostly held back by its profit guardrail in
  loss-making entries.

---

## V0.6b — benchmark suite runner and standard reports (0.6.0-dev.2)

Owner decision 2026-10-28:
* V0.6a accepted (CI green on a1afb41), demand growth provisionally and
  financial pressure as limited;
* insolvency mechanics and demand shocks deferred until after V0.6.0;
* V0.6b makes the suite easy to run, verify and interpret, with one new,
  distinct baseline and a prospective demand diagnostic;
* no new mechanics.

**Declared first** (8556766, before any run on seeds 41001–41003): the
code, the conservative baseline, the diagnostic, the protocol and
configurations, version 0.6.0-dev.2 (0.6.0-dev.1 registered for replay:
`mini-seed52-v061`, batch mode `v0.6_dev1`).

**Built:**
* **`dreamteam benchmark suite run`** (`internal/bench/suite.go`):
  * a suite-run configuration (`dreamteam.benchmark-suite-run/v1`) is
    checked: every entry's scenario against its recorded fingerprint, and
    T90 applicability;
  * it is expanded into an ordinary benchmark configuration and played
    through the V0.4 harness; run files and harness reports are byte for
    byte `benchmark run`'s;
  * the results add `suite-run.json`, `suite.json`, `contract.json` and
    `contract.txt`, hashed in the manifest's new optional `suite` list
    (ordinary manifests unchanged);
  * a failed run makes the suite `partial`, stated in the report;
  * exit status as `benchmark run`.
* **The contract report** (`internal/bench/contract.go`,
  `dreamteam.benchmark-contract/v1`):
  * per run: entry, scenario and fingerprint, policy and version, seed,
    horizon, starting-state digest, replay status, actions by category and
    type, T90 applicability and verdict, and the six dimensions with
    absent values marked and explained;
  * paired differences against the baseline;
  * starting-state identity per entry and seed;
  * missing measures.

  `benchmark report` prints it for suite results (`-harness` for the
  harness's own). `verify` checks the suite files, fingerprints, the
  expansion and the rebuilt report; `reproduce` reruns from the stored
  files.
* **`conservative` v1** (`internal/bench/conservative.go`; specification
  BENCHMARK_SUITE §7):
  * raises a site reading allocation only after three days in a row with
    the oldest study over 24 hours;
  * guardrails: trailing-30-day profit at or above zero after the
    change's added salary (from the preview), cash at least $50,000, at
    most 5 points above the scenario levels, 14 days between changes;
  * steps back when profit is negative or the backlog has been clear for
    a week.

  `noop`, `greedy` and `rules` are unchanged.
* **`dreamteam.demand-diagnostic/v1`**
  (`docs/benchmark/v06b-dev/demand_diagnostic.py`): generated demand
  (referral orders as they arrive, and walk-ins), completed exams and lost
  orders per operating day by 30-day window. It checks that generated
  demand is independent of policy (R1), and has declared conditions G1
  and G2. V0.6a's failed check stays recorded.
* **Tests:**
  * suite run equals the standalone matrix, verifies, reproduces, catches
    tampering;
  * partial suites;
  * invalid suite runs;
  * committed suite runs load through the holdout guard;
  * every conservative rule and guardrail, and its run behaviour;
  * the CLI.

**Development matrix** (60 runs, 76 min on the Pi): complete; `verify -t90` 578 of 578;
`reproduce` on linux/amd64 383 of 383, and a fresh linux/amd64 run's
manifest and contract report byte-identical to the Pi's. CI's suite-run
smoke test is byte-identical on three platforms.
* **Conservative:** no action in all 15 runs, identical to no action.
  The oldest study never passed 24 hours on three mornings in a row (at
  most two: Sunday and Monday). The trailing-30-day profit was negative at
  12–90 of 90 points.
* **Demand diagnostic:** R1 holds on 15 of 15 entry-seed pairs; G1 and G2
  hold on all three seeds. Under demand growth, orders per operating day
  rose 2–11% while completed exams stayed flat and lost orders rose.
* **The other baselines:** `greedy` and `rules` as in V0.6a.

**Gates:**
* Focused first: `internal/bench` (suite runner, contract, conservative,
  committed configurations), `internal/scenario`, `cmd/dreamteam` (the
  suite command, goldens, migrations) and the session registration, on
  the Pi.
* Full default Go suite on linux/amd64: every package passes.
* Race on linux/amd64: `internal/bench`, `internal/scenario`,
  `internal/session`, `internal/agent`, `cmd/dreamteam` pass.
* vet (three tag sets) and gofmt: clean.
* Python agent tests: pass.
* Five cross-builds.
* Session fixtures and golden references: identical outcomes under
  0.6.0-dev.2.
* CI green on the declaration commit, including the suite-run smoke
  test byte-identical on three platforms.
* The clinical evaluation suites, unaffected, were not run.

**Limitations:**
* `conservative` v1's service signal (the oldest study at 07:00) does not
  see the suite's soon and urgent breaches, so the baselines still offer
  only two behaviours;
* its profit guardrail cannot fund capacity whose shortage causes the
  loss;
* three development seeds;
* no LLM agent run.

---

## V0.6a — benchmark scenario suite and evaluation contract (0.6.0-dev.1)

Owner decision 2026-10-28: V0.5.0 accepted and closed; V0.6a establishes
the benchmark (scenarios, contract, governance) before any large model
evaluation. The V0.4 harness is reused unchanged. Specification:
`docs/BENCHMARK_SUITE.md`; characterization: `docs/benchmark/v06a-dev/`.

**Declared first** (a1ce029, before any run on the characterization
seeds): the scenarios, the suite file, the specification, the
characterization protocol and configuration, the analysis script, the CI
smoke test, and version 0.6.0-dev.1 (0.5.0 registered for replay:
`mini-seed52-v050`, batch mode `v0.5.0`).

**Built:**
* **Capability inventory** (specification §1): what the scenario format,
  demand, imaging and reading capacity, staffing, finance, actions and
  evaluation can and cannot represent. Not supported: demand trends or
  shocks, an injected starting state, absence or turnover, radiologist
  hiring, insolvency consequences, T90 beyond the sustainable scenario.
* **Five development scenarios** (suite `dreamteam-dev` 1), each a bundled
  scenario from a documented source:

  | Scenario | Built from |
  |---|---|
  | `bench-financial-pressure` | Step 0C `W115-SHR-MRft` |
  | `sustainable` (reporting backlog) | unchanged, so T90 still applies |
  | `bench-demand-growth` | sustainable, November–January demand at October's ×1.05/×1.10/×1.15 |
  | `bench-staffing-shortage` | Step 0 `W130` |
  | `bench-stable` | Step 0C `W130-DED-MRpt-A62` |

  Demand growth and financial pressure are partly supported (month
  factors as a trend; no insolvency); the smallest extensions are
  proposed, not built.
* **Evaluation contract** `dreamteam.benchmark-contract/v1`:
  * six dimensions of existing metrics, with primary metrics per
    scenario;
  * paired differences against no action;
  * no score, weighting or ranking;
  * T90 only for the reporting-backlog scenario, `not_applicable`
    elsewhere;
  * missing metrics named.
* **Development and held-out governance:** the held-out set is a future,
  separately versioned set, created by the owner and committed only as a
  hash, separate from the clinical holdout. None was created.
* **Reporting contract:** the standard result fields, mapped to the
  harness's files; what V0.6b must add.
* **Tests:**
  * `TestSuiteScenarios` checks each scenario against its source, its
    fingerprint, distinctness and T90 applicability;
  * `TestSuiteDimensions` checks that every contract metric exists;
  * the bundled-list test now names seven scenarios.

**Design disclosure:** the scenarios' intended conditions were checked
with no action on seed 1 only. Demand growth failed once and was redefined
once (relative to October's level). A 5-day test of the script showed that
demand growth and reporting backlog share their starting state by
construction; the protocol was corrected before it was committed.

**Characterization** (45 runs: five scenarios × seeds 40901–40903 × noop,
greedy, rules; 90 days; 57 min on the Pi):
* all completed and replayed;
* `verify -t90` 429 of 429;
* `reproduce` on linux/amd64 286 of 286, and a fresh linux/amd64 run's
  manifest byte-identical;
* every scenario's fingerprint differs.

**Observed** (no action, medians):

| Scenario | Profit | Lost | TAT p90 | Breaches |
|---|---|---|---|---|
| Financial pressure | −$32.8k | 11% | 855 min | 295 |
| Reporting backlog | $6.0k | 15% | 1,015 min | 481 |
| Demand growth | $12.7k | 17% | 1,159 min | 552 |
| Staffing shortage | −$28.5k | 19% | 741 min | 239 |
| Stable | $20.9k | 12% | 930 min | 331 |

* **Intended conditions:** four scenarios hold their condition on every
  seed. Demand growth held on one seed of three: the raw 30-day totals
  are confounded by weekday counts. Per weekday its demand rose on every
  seed, about 17% above reporting backlog by days 61–90.
* **Distinctness:** each pair of scenarios is separated on 4–7 primary
  metrics.
* **Baselines:** `greedy` and `rules` made the same 12 allocation raises
  everywhere (+$32.5k–$34k). The payoff differed by scenario: TAT p90
  −280 to −378 min under demand growth, −70 to −136 under staffing
  shortage. In the stable practice the same spend took profit to about
  break-even.
* **Cash:** no scenario ran out of cash.
* **T90:** NO GO for all nine reporting-backlog runs.

**Gates:**
* Focused first, on the Pi: `internal/scenario`, `internal/bench` (suite,
  dimensions, committed configurations) and the session registration.
* Full default Go suite on linux/amd64 (horo): every package passes.
* Race on horo: `internal/bench`, `internal/scenario`, `internal/session`,
  `internal/agent`, `cmd/dreamteam` pass.
* vet (default, calibration and acceptance tags) and gofmt: clean.
* Python agent tests: pass (unaffected).
* Scenario validation: all seven bundled scenarios.
* Session fixtures and golden references: identical outcomes, under
  0.6.0-dev.1.
* Benchmark verify and reproduce: as above.
* Cross-builds: five targets.
* CI: green on the declaration commit, with the suite smoke test's
  manifest byte-identical on three platforms.
* The clinical evaluation suites (calibration and acceptance tags,
  operating-characteristic runs), unaffected, were not run.

**Limitations:**
* five practices of one family;
* the baselines use only the reading allocation, so the staffing and
  demand challenges are not yet exercised;
* the sustainability dimension does not discriminate within 90 days;
* T90 covers one scenario;
* three development seeds.

---

## V0.5.0 — release

The V0.5 release on the owner's decision (2026-10-28): V0.5c accepted at
e495789, its CI (run 40643845200) green; no V0.5d and no tuning of the agent
or the models. LLM agents (V0.5a–V0.5c). Release preparation only:
* the version (0.5.0) and the replay registration of 0.5.0-dev.3
  (`mini-seed52-v053`, batch mode `v0.5_dev3`, recorded by a binary built
  from e495789);
* README, release notes, roadmap (with the owner's V0.6 starting scope),
  progress, and the LLM agent's and the benchmark's status and
  limitations.

No change to mechanics, randomness, scenarios, actions, accounting,
evaluation, scoring, the harness, the protocols, the agent, its prompt or
its settings. The 26 live model runs are not rerun: the release changes no
code they exercise, and CI replays all 18 recordings on every platform.
The clinical evaluation suites, unaffected, are not rerun.

The documented limitations are accepted as nonblocking (README, LLM_AGENT
§12): nondeterministic live inference and replay only from recordings;
trusted, unsandboxed external agents; operations and staffing batches not
jointly atomic; repeated invalid proposals despite memory; behaviour
dependent on prompt and configuration; frequent inaction; service bought
at substantial cost to profit; development results that establish neither
superiority nor generality. No model is clinically validated or
benchmark-certified. Next: V0.6a, the benchmark scenario suite and
evaluation contract.

## V0.5c — multi-model development evaluation (0.5.0-dev.3)

Owner decision 2026-10-27: V0.5b accepted at d476257. V0.5c runs the frozen
agent with several local models on fresh seeds, with one memory-off
control. No prompt tuning, no new scoring, no simulator, protocol or
scoring change. Report: `docs/benchmark/v05c-dev/README.md`.

**Declared first** (8c62526, before any run on the seeds): the plan
(`v05c-plan/1`), the three models, one settings file per model (identical
but for the model; `maxOutputTokens` 4,096 for all), the four stage
configurations, seeds 40801–40805, resource limits and stop rules.
`TestCommittedConfigs` loads every committed configuration through the
holdout guard.

**Built:**
* **Memory off:** with `memoryDecisions` 0 the MEMORY section says the
  memory is off and lists only the pending changes from the current
  observation. Memory on is byte for byte unchanged (the V0.5b recording
  replays with every memory and prompt hash matching). Tests:
  `MemoryTests.test_off`, `TestLLMAgentMemoryOff` (live mock and replay).
* **Version:** 0.5.0-dev.3; 0.5.0-dev.2 registered for replay
  (`mini-seed52-v052`, batch mode `v0.5_dev2`).
* **Evidence:** per stage, the report, manifest, recorded decisions, usage
  summaries, replay configuration and replay manifest; `analyze.py` and
  `tables.md` for the planned tables.
* **CI:** replays all four stages on every platform and compares the
  manifests.

**Ran** (Raspberry Pi 4, against a local omlx server; one stage at a time):

| Stage | Seeds, days | Runs | Wall clock |
|---|---|---|---|
| A | 40801, 7 | 4/4 | 4.9 min |
| B | 40802–40804, 30 | 12/12 | 77 min |
| C (memory off) | 40802–40804, 30 | 6/6 | 15 min |
| D | 40805, 90 | 4/4 | 2 h 33 min |

No infrastructure error, no rerun, no amendment, no horizon shortened.

**Observed** (descriptions of these runs, not claims):
* Qwen3.6 35B-A3B acted at 5–7 of 30 points per seed, used operations and
  staffing actions, and repeated refused shift extensions 1–5 times per
  run; it improved turnaround and breaches against noop on every seed at a
  lower operating profit (−$161 to −$34,565).
* Gemma 4 26B-A4B acted only on 40802 and 40805; on 40802 it was refused
  for teleradiology 13 times, repeatedly, and fell back at 6 points.
* Gemma 4 31B acted at 2 of 187 points; its outcomes equal noop's
  elsewhere. Slowest: 22–84 s mean per request.
* Memory off (Qwen, three seeds): more refusals (14/16/55 against
  6/5/9), more repeated invalid proposals (11/5/39 against 2/1/5), more
  accepted actions, and on two seeds about $17,600 less operating profit
  than noop. No causal claim.
* No model asked for details. No batch was refused when applied.

**Replay:** all 18 LLM recordings replay without a model (linux/amd64,
Python 3.13): decisions, sessions, results, memory and final digests
byte-identical to the live runs; two replays give identical manifests;
`verify` and `reproduce` pass for every stage.

**Gates:**
* Python agent tests: 28 OK.
* `go vet` (default, calibration and acceptance tags) and `gofmt`: clean.
* Go tests on the Pi: `internal/bench`, `internal/session`, `cmd/dreamteam`
  (goldens, migrations, committed configurations, memory off): pass.
* Race (`-race`) on horo: `internal/bench`, `internal/session`,
  `internal/agent`, `cmd/dreamteam`: pass. The Pi's kernel does not
  support the race detector.
* Cross-builds: linux/amd64, linux/arm64, darwin/arm64, darwin/amd64,
  windows/amd64.
* CI steps on horo: the four golden references identical; the CI
  benchmark verifies (86) and reproduces (59); V0.5a, V0.5b and all four
  V0.5c replays verify and reproduce, and the V0.5c replay manifests equal
  the committed ones. The V0.5a and V0.5b replay manifests now differ from
  their committed copies only through the engine name and the agent's
  hash, as expected; CI compares replays across platforms, not with them.
* The expensive clinical suites, unaffected, were not rerun.

**Limitations:** one scenario, five seeds, three models on one shared
server, each live run once; outcomes are the simulator's; the memory
comparison cannot attribute outcome differences to the memory. See the
report §11.

---

## V0.5b — memory, action guidance and cost reporting (0.5.0-dev.2)

Owner decision 2026-10-27: V0.5a accepted at 3104e67. V0.5b improves the
LLM agent without changing simulator behaviour, the protocol or scoring.
No multi-model comparison (V0.5c).

**Built** (`agents/llm/llm_agent.py`, llm-agent 2; spec
`docs/LLM_AGENT.md`; agent, tests and smoke-test configuration declared in
e30aacf):
* **Bounded memory** (`llm-agent-memory/v1`, §6):
  * what it holds:
    - the agent's recent decisions in full, the earlier ones as totals;
    - the `validate` refusals it met, with their messages;
    - each decision's applied outcome, from the next observation's action
      log;
    - metric rows at recent decision points;
    - the pending changes the observation and catalog show;
  * derived only from public observations, the catalog and the agent's own
    records;
  * bounded by decisions, rows and characters (default 6,000);
  * in the process only, so empty at the start of every run.
* **Targeted prompt** (`llm-agent-prompt/v2`, §4):
  * contents:
    - the time, operating hours and scheduled close;
    - active problems;
    - the memory;
    - a summarized observation;
    - for every action type, its fields (its JSON Schema in one line),
      the catalog's rules, the earliest and latest dates and lead times
      from its targets (shown as days from today), and the targets;
  * where no date is given, the prompt says so;
  * every cut is named, and the model may ask once for details: full
    catalog entries, or left-out paths from the compact or full
    observation;
  * `observation: compact` and `catalog: full` restore V0.5a's content;
  * on V0.5a's seven recorded observations: 59% fewer characters and 56%
    fewer tokens (the server's counts: 21,850–23,016 to 9,465–9,936).
* **Mixed actions** (§7):
  * the investigation: `mixed_batch` is a deliberate session invariant
    (atomic staffing batches versus sequential V0.1 batches,
    V0.2_DESIGN §5.7), not a transport, validation or implementation gap;
  * kept unchanged, and documented in LLM_AGENT §7 and AGENT_PROTOCOL §3;
  * the agent now splits a reply into an operations batch and a staffing
    batch, validates each and submits both at one decision point (the
    decision's documented list of batches);
  * the two are not one atomic unit.
* **Usage and cost** (§9; `summary.json`, `usage.txt`):
  * the model;
  * requests, retries and details requests;
  * tokens, as reported, estimated or unknown;
  * mean, median and maximum latency, and total inference time;
  * valid, refused and fallback decisions;
  * an estimated cost from configured prices per million tokens, labelled
    as an estimate;
  * no endpoint and no secrets.
* **Replay** (§8):
  * a version 2 recording's memory and prompt are rebuilt at every decision
    point and checked against their recorded SHA-256;
  * V0.5a recordings replay as before;
  * other memory or prompt versions are refused with an explicit
    `version incompatible` error.
* **Version:** 0.5.0-dev.2; 0.5.0-dev.1 registered for replay
  (`mini-seed52-v051`, batch mode `v0.5_dev1`).
* **CI:** replays both smoke tests' recordings on every platform and
  compares their manifests.

**Mock acceptance** (`internal/bench/llm_agent_test.go`, twelve mock
behaviours; the owner's ten tests):
1. **The memory updates:** each decision and its applied outcome are carried
   to the next point; refusals and pending allocation changes are shown.
2. **It stays bounded:** a 90-day mock run keeps it at most 1,479 of 1,500
   characters, and a 120-point unit test within 6,000.
3. **It resets:** it is empty at the start of every run.
4. **Pending changes come from public data only:** pending versions and
   staff dates come from the catalog and observation, and fields the
   observation does not have never appear.
5. **Invalid actions still fail safely:** each failure gets exactly one retry,
   then no action, with nothing applied.
6. **Constraints appear:** dates, lead times, operating hours, caps and batch
   rules are in the prompt (unit tests on the recorded observation).
7. **Filtering keeps every action:** every type and every field is in the
   prompt, details return full catalog entries and a path from the full
   observation, and `catalog: full` includes the JSON.
8. **Usage totals are correct:** they match the mock's counts.
9. **Cost estimates are correct:** they match the configured prices.
10. **Replay stays byte-identical:**
    * decisions, sessions, results, digests and the memory at every point
      match the live runs;
    * tampered digests and a tampered memory hash are refused;
    * another memory format is refused.

Mixed operations and staffing actions were applied as two batches at one
decision point. Python unit tests: 26.

**Live smoke test** (`docs/benchmark/v05b-dev/`; sustainable, seed 40701,
14 days, noop and the agent with V0.5a's model on a local server):
* both runs completed, and `verify` passed 24 of 24;
* 14 decisions:
  * 6 with actions: 6 batches, 17 actions, all accepted;
  * 7 no action;
  * 1 fallback (a reply cut off at the output limit, then a refused
    retry);
* `validate` caught every refused proposal. The model re-proposed an
  `after_close` extension at points 3, 6 and 7, although its memory held
  that refusal;
* no staffing action and no details request;
* the memory grew to 5,976 characters and stayed under its 6,000 bound;
* 20 requests: 236,061 prompt and 8,797 completion tokens, all
  reported (first attempts 9,613–12,506 prompt tokens, memory included);
  mean latency 4.3 s, median 4.0 s.

**Replay:**
* byte-identical decisions, sessions, results and final digests to the
  live run, and an identical memory at all 14 points;
* two replays are identical, and reproduce passed 19 of 19;
* the manifest is byte-identical on linux/arm64 and linux/amd64 (Python
  3.11 and 3.13);
* the V0.5a recording still replays byte-identically to its live run.

**Validation:**
* the full uncached suite passes on linux/arm64 (about 3,720 s) and
  linux/amd64 (532 s);
* race passes on bench, agent, session and cmd/dreamteam: 0 data races;
* vet (all tags) and gofmt are clean;
* 36 of 36 session fixtures verify, including the new 0.5.0-dev.1 fixture;
* golden references are identical;
* the Python unit tests pass on Python 3.11 and 3.13;
* the CI benchmark passes verify (86/86) and reproduce (59/59); its 48
  files equal 0.5.0-dev.1's apart from the engine version (and the session
  checksums and manifest hashes that cover it);
* cross-builds succeed for five targets.

**Known limitations** (`docs/LLM_AGENT.md` §12):
* the memory is bounded, informs rather than enforces, and has no
  planning behind it;
* outcomes are read from the action log;
* the operations and staffing batches are not one atomic unit;
* summaries can omit what matters (details on request);
* live runs are not reproducible;
* cost is an estimate from configured prices;
* one model, one seed, 14 days.

---

## V0.5a — the external LLM agent (0.5.0-dev.1)

Owner decision 2026-10-27: V0.4.0 accepted at ced3acb; V0.5 re-slotted
from growth to LLM agents (growth deferred, V0.6 the full agent
benchmark). V0.5a: one minimal model-agnostic LLM agent that takes part
through the existing policy protocol as a separate process.

**Built.**
* **The agent** (`agents/llm/llm_agent.py`, llm-agent v1; spec
  `docs/LLM_AGENT.md`). Standard library Python, an OpenAI-compatible chat
  completions client.
  * **Settings** (`dreamteam.llm-agent-settings/v1`): the model and base
    URL, given directly or by environment variable; the API key from the
    environment only; timeout, temperature, output tokens, retries,
    decision budget, response format, prompt bound, optional system
    prompt file.
  * **Each decision point:** `digests`, the compact observation, the
    catalog; a bounded prompt (`llm-agent-prompt/v1`); a strict
    `{"reason", "actions"}` reply; catalog types checked; the batch
    validated through the protocol and submitted only if valid; bounded
    retries with the error fed back; no action on failure.
  * **Stateless:** no memory, no planning, one batch at most.
* **Evidence:** `agent.json`, `model.jsonl` (every request payload, raw
  response, usage, latency, error), `decisions.jsonl` and `summary.json`,
  without secrets.
* **Replay mode** (`--replay FILE…`): recorded decisions submitted without
  a model, each matched to its decision point and all five digests;
  several recordings are matched by their first digests.
* **Harness:** an external policy's `evidence/` files are kept in the
  run's `policy-evidence/` (≤1,000 files, ≤64 MiB, regular files only),
  listed in the manifest as nondeterministic and not hashed. No other
  harness, protocol, scoring or simulator change; manifests of configurations
  without evidence are unchanged.
* **Version:** 0.5.0-dev.1; 0.4.0 registered for replay
  (`mini-seed52-v040`, batch mode `v0.4.0`).
* **Roadmap re-slot**; CI runs the agent's unit tests and replays the smoke
  test's recorded decisions on every platform.

**Mock acceptance** (`internal/bench/llm_agent_test.go`, a deterministic
mock OpenAI-compatible server; ten behaviours; `TestLLMAgentMock`,
`TestLLMAgentReplay`):
* every run completes;
* the agent asks only for hello, digests, observe, actions and validate,
  and the gate denies nothing;
* a no-action reply is no change, and a valid raise is applied, from a
  plain or fenced reply and on a retry;
* prose, an unknown action, a refused allocation, an extra field, a slow
  server and HTTP 500 each get exactly one retry, then no action, with
  nothing applied;
* the evidence is kept and listed;
* the stand-in API key reaches the server and appears in no result file;
* a replay calls no model and reproduces the live runs' decisions,
  sessions, results and digests; another run's or a tampered recording
  is refused (`policy_crash`).

The Python unit tests (11) cover the settings, parsing, endpoint, prompt
bound and replay matching.

**Live smoke test** (`docs/benchmark/v05a-dev/`; config, settings and agent
committed first in b5ac92f; sustainable, seed 40601, 7 days, noop and the
agent with `Qwen3.6-35B-A3B-oQ4e-mtp` on a local server):
* both runs completed, and `verify` passed 24 of 24;
* 7 decisions: 5 no action, 2 batches (3 shift extensions), all valid when
  submitted;
* 2 proposals refused by `validate` (`past_date`, `after_close`) were each
  corrected on the one retry;
* 9 model calls: 202,084 prompt and 1,225 completion tokens, mean latency
  5.1 s.

**Replay:** byte-identical decisions, sessions, results and final digests
to the live run; reproduce 19 of 19; manifest byte-identical on
linux/arm64 and linux/amd64 (Python 3.11 and 3.13).

**Validation:**
* the full uncached suite passes on linux/arm64 (3,632 s) and linux/amd64
  (532 s);
* race passes on the affected packages (bench, agent, session,
  cmd/dreamteam): 0 data races;
* vet (all tags) and gofmt are clean;
* 35 of 35 session fixtures replay, including the new 0.4.0 fixture;
* golden references are identical;
* the Python agent's unit tests pass on Python 3.11 and 3.13, and the
  Python client and reference policy are unchanged;
* the CI benchmark passes verify (86/86) and reproduce (59/59); its 48
  files equal v0.4.0's apart from the engine version;
* cross-builds succeed for five targets.

**Known limitations** (`docs/LLM_AGENT.md` §9): stateless; one batch per
decision point (V0.1 and staffing actions cannot share one); live runs
are not reproducible; about 22,000 prompt tokens per decision; a trusted
process; no cost accounting beyond tokens and latency; one model, one
seed, 7 days: not a measure of decision quality.

---

## V0.4.0 — release

The V0.4 release on the owner's decision (2026-10-26): V0.4b accepted at
1a88857, no further implementation slice. The agent evaluation harness
(V0.4a and V0.4b). Release preparation only:
* the version (0.4.0) and the replay registration of 0.4.0-dev.2
  (`mini-seed52-v042`, batch mode `v0.4_dev2`);
* README, release notes, roadmap, progress and the benchmark
  specification's status and limitations.

No change to mechanics, randomness, scenarios, actions, accounting,
evaluation, replay, the harness, the policies or the policy protocol.
`python-ref` v1 is unchanged. The 90-day development matrix is not rerun:
the release changes no code it exercises, and CI runs the benchmark
configuration (verify and reproduce) on every platform.

The documented limitations are accepted as nonblocking (`docs/BENCHMARK.md`
§12): trusted, unsandboxed external policies; seed withholding through
the protocol only; wall-clock timeouts; children of a failed policy;
`python-ref` v1 never acting in the development matrix; `greedy` and
`rules` trading about $33,000 of cost for faster reporting; T90 NO GO on
all 20 development runs. Development comparisons are not confirmatory
performance claims. Next: the owner's V0.5 decision.

---

## V0.4b — external agent benchmarking (0.4.0-dev.2)

Owner decision 2026-10-26: V0.4a is accepted at 2ab3564. V0.4b scope:
1. an external policy process adapter;
2. benchmark seeds hidden from agents;
3. a fresh 90-day development comparison;
4. a reproduce command, if straightforward.

No new built-in baseline.

**Built.**

* **External policies** (`internal/bench/external.go`,
  `docs/BENCHMARK.md` §4).
  * A configuration declares them under `external`: ID, version,
    description, command, timeout. `./` and `../` command files are
    resolved from the run directory, and their SHA-256 goes in the
    manifest.
  * One fresh process per run, in an empty working directory outside the
    results, removed afterwards.
  * The policy protocol (`dreamteam.benchmark-policy/v1`), JSON lines
    over stdio:
    * the harness sends `decide` and `end` turns;
    * the policy sends agent/v1 requests, each answered in order through
      the same gate as a built-in policy's;
    * the policy ends each turn with `{"decision":{"batches":[…]}}`.
  * A per-decision timeout, which also covers the exit after `end`.
    Standard error is captured separately (`stderr.txt`, kept and
    listed, not hashed).
  * Failures are classified: `policy_crash` (could not start, ended
    early, exit status not 0), `policy_timeout`, and `policy_error` (a
    decision that cannot be read). The process is killed and reaped, and
    the other runs go on. Requests the protocol refuses are counted as
    request errors (a new count, for built-in policies too).
  * No HTTP or sockets. apply, advance, save and t90 stay the harness's.
* **Seed privacy.**
  * `agent.Handler.Benchmark` is set by the harness. `hello` then omits
    the seed and carries `benchmark: {seedWithheld, permittedOps,
    decision}`.
  * Every other session's `hello` is byte-for-byte unchanged (the Python
    client's run digest is unchanged).
  * The seed stays in the harness's own evidence.
  * The trust boundary (BENCHMARK.md §11): this stops disclosure through
    the protocol only. External policies are trusted development
    participants, not sandboxed. No OS-level isolation is claimed.
* **`clients/python/reference_policy.py`** (`python-ref` version 1):
  * standard library only and protocol only;
  * a weekly review (decision points 0, 7, 14, …) that raises one site
    reading allocation by 5 points, validated first, when the oldest
    unfinished study is over 24 h old;
  * a refused proposal is passed over, a protocol error means no change
    that turn, and the process exits 0 after `end`.

  It was fixed before any run and committed with the configuration.
* **`dreamteam benchmark reproduce RESULTS [-out DIR]`**
  (`internal/bench/reproduce.go`). It reruns the recorded `config.json`
  and compares engine, policies, policy files, metric definitions, run
  statuses, every hashed file and `manifest.json`. Timing and standard
  error are not compared. Exit codes: 0 identical, 3 different, 1 not
  rerunnable (another engine).
* **Report**: external policies are marked; request errors are added to
  the counts table.
* **Replay registration**: 0.4.0-dev.1 (batch mode `v0.4_dev1`, fixture
  `mini-seed52-v041` recorded by the 2ab3564 binary), with its
  MIGRATIONS entry.
* **CI**: the benchmark step includes `python-ref` and runs `reproduce`.
  Its manifest, the external policy's runs included, is compared across
  linux/amd64, linux/arm64 and darwin/arm64.

**Development comparison** (`docs/benchmark/v04b-dev/`):
* sustainable, seeds 40501–40505, noop, greedy, rules and python-ref,
  90 days, T90;
* configuration and policy committed first (0853bcc);
* 20 of 20 completed and replayed, decisions matched, identical initial
  conditions per seed;
* no failures, refusals, protocol errors, denied requests or request
  errors.

| Machine | Result |
|---|---|
| Pi (arm64, Python 3.11) | 2,094 s; `manifest.json` `dd957074…`; verify -t90 226 of 226 |
| horo (amd64, Python 3.13) | the same `manifest.json`; verify -t90 226 of 226; `reproduce` 147 of 147 |

Findings, not claims:
* **python-ref never acted.** Its weekly reviews fall on Tuesdays, while
  the backlog passes 24 h only on Sunday and Monday mornings. Its runs
  equal noop's ("same decisions, identical outcome"). The rule is not
  changed.
* **Greedy and rules made the same 12 raises per seed in the opposite
  order.** Rules never lowered. At 90 days their outcomes differ
  slightly on every seed (TAT p90 by 1–20 min, breaches by 0–13; profit
  equal within $1). The 90-day horizon does not make them meaningfully
  different.
* **Against noop, 5 of 5 seeds:** much lower TAT and fewer breaches, net
  revenue +$0.5k to +$5.3k, expense about +$33k, profit −$27k to −$33k.
* **T90 at day 90 (not projected) is NO GO for all 20 runs.** noop and
  python-ref fail service reliability; greedy and rules fail solvency.

**Compatibility.**
* The V0.4a configuration reruns on 0.4.0-dev.2. All 15 runs have the
  same initial and final digests, metrics, decisions and T90 verdicts
  as V0.4a's, and verify passes 156 of 156.
* The V0.3 Python client passes against 0.4.0-dev.2 with the same run
  digest (`948f6ad0…`). Its transcript is V0.4a's apart from the engine
  version.

**Tests.** `internal/bench`:
* `TestExternalPolicy`: a real subprocess (the test binary as a helper).
  It checks counts (denied 4, request errors 1), byte-identical reruns,
  the manifest's policy entry, `stderr.txt` listed as nondeterministic,
  verify, and that no working directory is left.
* `TestExternalPolicyFailures`: crash, hang (timeout), a malformed
  decision, a process that does not exit, a non-zero exit, and a missing
  policy file refused before running. Each is classified at the right
  decision point, the files verify, and the noop run beside them is
  byte-identical to its run alone.
* `TestSeedIsWithheld`: a built-in probe and an external process call
  every permitted operation, failing calls included, on two scenarios.
  No response carries the seed's digits; `hello` says it is withheld;
  the process's arguments, environment and working directory do not
  contain it; `run.json` keeps it; a non-benchmark `hello` still
  reports it.
* `TestPythonReferencePolicy`: completes, verifies, two runs identical,
  its file hash recorded, standard error empty.

`cmd/dreamteam`: `reproduce` matches a fresh rerun, and a manifest that
records another hash fails (exit 3). `internal/session`: the
0.4.0-dev.1 fixture replays and continues.

Mutation checks: a `hello` that reports the seed under the harness, and
an external policy whose requests bypass the gate, each fail
`TestSeedIsWithheld` and `TestExternalPolicy`.

**Limitations** (BENCHMARK.md §12):
* external policies are trusted, not isolated: a policy could read
  local files or search for the seed from the digests;
* timeouts are wall-clock and include the harness's time;
* only the started process is killed on failure;
* no feedback within a decision point;
* all acting policies use one lever, and python-ref's weekly timing
  never met its trigger here.

---

## V0.4a — the agent evaluation harness (0.4.0-dev.1)

Owner decision 2026-10-26: V0.3.0 is accepted and closed. V0.4 is the
agent evaluation harness; V0.6 keeps the full benchmark. Scope: compare
policies fairly and reproducibly through the released agent protocol,
with no new scoring rule.

**Built.**

* **`internal/bench`, the harness** (`docs/BENCHMARK.md`).
  * A versioned configuration (`dreamteam.benchmark/v1`), validated
    before anything runs. It needs explicit seeds and a seed plan. It
    refuses reference seeds 1001–1020 and sealed holdout seeds (the
    holdout guard gives an error only).
  * The **gate**: a policy reaches the session only through the
    protocol's read and check operations (hello, observe, sections,
    actions, validate, preview, digests). apply, advance, save and t90
    are the harness's, and the gate answers them `not_permitted` without
    touching the session.
  * **One fresh session and policy per run.** For each run the harness
    applies the policy's batches, advances at the configured cadence,
    saves, and asks for T90 where it applies. It then replays the saved
    session and writes its canonical result: `dreamteam replay
    -format json`, byte for byte.
  * **Metrics** are 32 values read by path from that result, with no
    conversion. Failures are classified (`policy_error`,
    `protocol_error`, `evaluation_error`, `replay_error`,
    `harness_error`). Refusals, protocol errors and denied requests are
    counted.
  * **Results.** A results directory with a manifest of SHA-256 hashes,
    in which only `timing.json` is nondeterministic. A report (text and
    JSON) with five parts:
    1. runs;
    2. initial conditions;
    3. per-policy minimum, median and maximum;
    4. paired differences with the baseline, separating same decisions
       and identical outcomes from different decisions, plus counts of
       higher, lower and equal;
    5. failures and replay.
  * **Verification**: hashes and no unlisted files, replay, the result
    rebuilt, metrics, the decisions against the session's batches, the
    report rebuilt, and optionally T90 recomputed.
* **CLI**: `dreamteam benchmark run CONFIG [-out DIR]`, `report RESULTS
  [-format text|json]`, `verify RESULTS [-t90]`.
* **Policies**: `noop`, `greedy` (the V0.3 demonstration, unchanged) and
  `rules` version 1. Rules keeps the reading allocation in step with the
  backlog, with hysteresis: it raises past 24 h oldest unread and lowers
  after three decision points at 8 h or less. Each change is validated
  and previewed first. The thresholds were fixed before any run.
* **`internal/agent`**: `Client.H` is an `Exchanger` interface (a handler
  or the gate), with optional string id prefixes and `Requests()`.
  `Handler.Dir` resolves a relative save path, so transcripts carry no
  absolute path. The protocol's behaviour is unchanged.
* **Replay registration.** The v0.3.0 release is registered for replay:
  batch mode `v0.3.0`, fixture `mini-seed52-v030` recorded by the
  v0.3.0 binary built from the tag. The MIGRATIONS entry is added.
* **Holdout containment.** The containment test reviews the new
  `GuardManifest` call (bench config validation).

**Development run** (`docs/benchmark/v04a-smoke/`):
* sustainable, seeds 40401–40405, three policies, 30 days, T90;
* the seeds were committed alone (b9cc0b6) before any run on them;
* 15 of 15 runs completed; every session replayed and matched its
  decisions; initial conditions are identical per seed;
* the Pi (arm64) and horo (amd64) gave the same `manifest.json`
  (`81ecfaa1…`), so every deterministic file is byte-identical;
* `verify -t90` passed all 171 checks.

Findings, not claims:
* greedy and rules made the same four allocation changes per seed in a
  different order, so with the 14-day lead time their 30-day outcomes are
  nearly identical;
* both finalized more reports, with lower TAT and fewer breaches than
  noop, on 5 of 5 seeds, with mixed profit;
* T90 (v0.2-eval-4, projected) gives CONDITIONAL GO on 2 seeds and NO GO
  on 3.

**Tests.** `internal/bench` covers:
* configuration validation;
* the gate;
* byte-identical reruns;
* initial conditions;
* isolation: a run's files do not depend on the other policies or their
  order;
* the policy's view: the transcript shows only permitted ops;
* policy errors and panics: classified, files verify, the other runs go
  on;
* metrics against the result;
* rules' validate→preview→apply order;
* tamper detection.

`cmd/dreamteam` checks:
* T90 equals `dreamteam t90` (v0.2-eval-4, projected);
* `result.json` equals `dreamteam replay -format json`;
* `verify -t90`, report JSON, write-once output and a tampered file;
* invalid configurations (exit 2, nothing written) and usage errors.

Mutation check: an open gate fails three tests.

**Limitations** (BENCHMARK.md, known limitations):
* built-in policies only; the external-process adapter is designed but
  not built;
* `hello` shows the seed, which an external agent with the binary could
  use to simulate ahead;
* there is no feedback between batches within a decision point;
* a 30-day horizon sees only reading changes set in the first 16 days;
* T90 takes about 40 s per run on a Pi;
* rules is not yet distinct from greedy on this scenario.

---

## V0.3.0 — release

The V0.3 release on the owner's decision: the agent interface (V0.3a and
V0.3b). Release preparation only:
* the version (0.3.0) and the replay registration of 0.3.0-dev.2
  (`mini-seed52-v032`);
* README, release notes, roadmap and protocol status;
* a test fix: the session test of an unregistered engine named 0.3.0,
  which is now this build, so it names 9.9.9 instead.

No change to mechanics, randomness, scenarios, actions, accounting,
evaluation or replay. The documented limitations are accepted as
nonblocking: `t90`'s run time on a Pi, staffing-only previews, and the
multi-seed bound's last digit across processor families. Next: an agent
evaluation harness.

---

## V0.3b — external agent usability (0.3.0-dev.2)

The second V0.3 slice, on the owner's decision after V0.3a: the protocol
made practical for agents outside the process. Not released.
* **`t90`:** the T90 evaluation of the session as it stands, by the
  policy in force (session.T90), the same as `dreamteam t90`. It is only
  ever explicit (never in `observe`) and read-only: no digest changes
  and no randomness is drawn. Stable refusals for unsupported scenarios,
  earlier action rules, not started, sealed seeds.
* **Observation projections:** `observe` takes `sections` and `compact`,
  and the new `sections` op lists what can be selected. A projection is
  cut from the full observation's own JSON (keys left out, order kept),
  so it can show nothing new. `compact` leaves out four declared record
  listings and counts them (≈116 KB → ≈29 KB). Without options, observe
  is byte-for-byte as before.
* **The reference Python client** (`clients/python/`, standard library
  only): a complete session through a real subprocess, checked by
  `dreamteam verify` and `dreamteam t90`; run by the Go suite and in CI,
  where its transcript is compared across platforms.
* **Cross-platform statistics.** T90 reports differed between arm64 and
  amd64 in the last digit of some reported statistics (design effects,
  Wilson bounds): arm64 fuses multiply-add, amd64 does not. The
  evaluation now rounds every product before a sum (`float64(x*y)`, as
  the Go spec provides). amd64's values are unchanged; arm64's now equal
  them, so arm64 T90 output differs in those digits from V0.2.0's. No
  status, verdict or rule changed.
* **Compatibility:** the V0.3a transcript replays byte for byte except
  hello's engine and ops. Registers 0.3.0-dev.1 for replay
  (`mini-seed52-v031`).
* **CI had been failing since V0.2 Step 0C, on two defects of its own,
  found here and fixed:**
  * The golden step compared without the migration manifest that the
    documented check uses (`testdata/golden/README.md`), so the
    deliberate state-schema change of V0.2 Slice 1 read as a mismatch
    on arm64 and macOS. It now passes `-golden-migrations`.
  * `TestV4Neutrality` pinned an arm64 digest that never held on amd64.
    It now pins one digest per processor family. The remaining
    difference is the multi-seed joint gate's exact bound, through the
    Go math library, in the last digit; every status is the same.
  
  The local gates (the full suite on arm64, race on amd64, goldens with
  the manifest, replays) had passed throughout.

---

## V0.3a — the deterministic agent interface (0.3.0-dev.1)

The first V0.3 slice: `dreamteam agent --stdio`, a JSON protocol over
the session API (`docs/AGENT_PROTOCOL.md`), on the owner's decision after
the V0.2.0 release. Not released.
* **Transport-independent.** `internal/agent.Handler.Handle` turns one
  request into one response, as bytes; `ServeLines` carries them one per
  line over stdin and stdout. Only responses go to stdout.
* **Operations:** `hello`, `observe` (the game's observation), `actions`
  (a JSON Schema, constraints and current targets for every action type,
  read from the offer table and the session's views), `validate`,
  `preview` (the staffing preview), `apply` (atomic: a batch is applied
  only if every action is valid, so a refused batch changes no digest),
  `advance` (minutes, to a time, to the next visible event, to the end
  of the day: the session's own semantics), `save`, `digests`.
* **Stable error codes**; invalid JSON, unknown ops and bad arguments
  never stop the process or change the session; ids are echoed.
* **A scripted runner** (`dreamteam agent -policy noop|greedy`) drives
  the same protocol in process, from protocol JSON only.
* **No simulation change.** Registers 0.2.0 (the release) for replay
  (`mini-seed52-v020`).
* **A V0.2 defect fixed: the T90 evaluation of a continued session.**
  V0.2's T90 view refused any session continued from an earlier build
  ("defined for the sustainable scenario only"), which under this build
  would have meant every sustainable session saved by the V0.2.0
  release. A continued session is now evaluated with its earlier
  decisions when they replay under this build's action rules (every
  build from 0.2.0-dev.8 on), and refused with an accurate message
  otherwise. The evaluation of a V0.2.0-saved session equals that of the
  same plan played here (`sustainable-seed906-v020`,
  `TestT90EarlierEngineSession`).
* **Tested:** 30-day noop and greedy sessions valid line by line,
  byte-identical when repeated, saved and replayed; every action type
  discovered and applied from the catalog's targets; invalid batches,
  validate and preview leave every digest unchanged; the agent and the
  session API reach the same states, the observation byte for byte;
  stdout carries only responses, one per request, in order.

---

## V0.2.0 — release

The V0.2 practice-design release, on the owner's decision after the
acceptance pass and the completion of F1 and F2. Release preparation
only: the version (0.2.0), the replay registration of 0.2.0-dev.27
(`mini-seed52-dev27`), README, release notes and roadmap. No change to
mechanics, randomness, scenarios, actions, accounting, evaluation or
replay. The documented P2 limitations are accepted as nonblocking
(`README.md`). Next: V0.3, the agent interface.

---

## V0.2 completion — the T90 evaluation view and the bundled sustainable scenario (0.2.0-dev.27)

The V0.2 acceptance pass (16e5751) found two promised deliverables not
built; the owner decided to build both before the release.
* **F2 — `sustainable` is a bundled scenario.** `scenarios/sustainable.json`
  is the Step 0C shared-staffing control at ×1.30 demand (V0.2_DESIGN
  §5.6, `W130-SHR-MRft`): the established practice with that experiment's
  overlay, byte for byte. By the owner's decision its internal id stays
  `established`, so it keeps the fingerprint the evaluation policy in
  force pins (`e8718dbd…`); it is listed and chosen as `sustainable`.
* **F1 — the T90 evaluation is visible.** `dreamteam t90 FILE` and the
  summary screen's third view (`tab`) evaluate the player's plan under
  v0.2-eval-4, embedded byte for byte: the plan's decisions before T90
  and the no-action baseline on the same seed, both run with no further
  decision to the observation end, judged by `eval.Evaluate`. They show
  the metric vector with the baseline and the difference, every service
  gate (pass, fail or insufficient, with its counts and limit), the plan
  gates and the verdict, and say what is projected (days not yet played),
  what is the fixed-policy tail and what is incomplete. No rule is added;
  "scoring comes later" is gone. Defined for the sustainable scenario
  only; holdout seeds are refused, as by every harness.
* **No engine change:** the evaluation recorder moved unchanged from a
  test file into `internal/sim`; every golden and replay fixture is
  unchanged.
* **Registers 0.2.0-dev.26** for replay (`mini-seed52-dev26`).
* **Tested:** the game's report equals the canonical development
  runner's byte for byte (`TestT90MatchesCanonicalRunner`); pass, fail and
  insufficient gates each displayed as decided; projected vs measured;
  the session unchanged by evaluating; the CLI, the session and the TUI
  view identical; other scenarios refused; the sustainable scenario
  deterministic and replayed, its fingerprint the policy's.

---

## V0.2 v5 protocol lock (0.2.0-dev.26, lock commit; NOT RUN)

The prospective confirmatory protocol for v0.2-eval-5 is locked, on the
owner's conditional GO after the 5C.4o release-gate audit. The lock
record is `docs/calibration/clinical-v5-lock/LOCK.md`, with
`lock.sha256` beside it.
* **Not run, not frozen, not activated.** The reserved run needs the
  owner's separate authorization.
* **The lock changes only what protocol §8 provides:** `familyAllowed`
  admits PCG(20261201, ·); the two refusal assertions become admission
  assertions, constructing nothing; the protocol's status statements
  read LOCKED; its digest is re-pinned (`12d1eda4…0a0d`).
* **Pinned:** parent `7bf1ed6`; `sourceSHA256` `5cd6a438…588b`; the
  test binary built with `-trimpath`, `CGO_ENABLED=0`, go1.27.1
  linux/arm64, `5edee74d…130d` (reproducible across checkouts); the
  candidate, manifest, protocol, policy in force and simulator profile;
  the command, 3,904 lines, T1–T8, the T5b tolerances, the trusted
  in-process scope with the study-swap limitation, one execution, the
  failure handling and no post-run tuning.
* **Verified:** every committed test passes at the lock tree; the
  pinned binary, run on the development family only, records the pinned
  source digest, inputs and engine and reads `development` /
  `not certified`. Nothing ran on the reserved family.

---

## V0.2 deterministic adjudication output, before the protocol lock (0.2.0-dev.26)

The owner accepted O1 and O2 as closed on the targeted remediation
(conditional GO for the lock) and required one change first: no output
byte may depend on Go's map iteration order.
* **The change:** T5a sums its group predictions, and lists its group
  means, in group order instead of map order. No formula, threshold,
  policy, manifest, seed family, simulation behaviour or certification
  rule changed. NM2.1's 200-seed prediction now reads 0.34404 on every
  adjudication (it varied in the 17th digit).
* **Registers 0.2.0-dev.25** for replay (`mini-seed52-dev25`).
* **Tested:** `TestAdjudicationDeterministic` (fails with map order
  restored); the 200-seed development lines re-read three times and two
  trusted 80-seed development executions (NM2, NM2.1), each
  byte-identical, `development` / `not certified`; the full uncached
  suite, vet, gofmt and replay through dev.25.
* **Not locked, not run, not frozen.** The lock is a separate commit.

---

## V0.2 release-gate blocker fixes — audit 5C.4o O1 and O2 (0.2.0-dev.25)

The focused release-gate audit 5C.4o (36d34f2, committed verbatim before
any change) found M1, M3 and M6 closed and returned NO GO on two
certification-critical findings. Under the owner's stopping rule only
those were fixed and only their checks rerun. The maintainer's response
is `docs/audits/slice-5c4o-release-gate-audit-response.md`.
* **Not locked, not run, not frozen.** The owner decides whether the
  fixes close the gate; no further broad audit is scheduled.
* **Unchanged:** the clinical policy, the manifest, the simulator, the
  thresholds, the trust boundary and the reserved seed protocol.
* **Registers 0.2.0-dev.24** for replay (`mini-seed52-dev24`).
* **Nothing reserved was run.**

**Fixed:**
* **O1:** each line's Gate B is reconciled with its own measurements by
  the candidate's rule (`gateBReconciles`): decided statuses need every
  pair matched and, for urgent, a difference on every pair; differences,
  safeguard flags, deteriorated seeds, increase and status must be the
  rule's computation on them, exactly. Near-margin replicates need a
  difference on every pair; no pair injects more than it served or left
  unimaged; a hidden pair construction error is refused. Every relation
  held on all 664 genuine development lines first.
* **O2:** the reader refuses anything after a line's JSON value.
* **O3 (documentation):** the "failing exit" and survivor-equivalence
  claims are narrowed in place; the lock recipe names the two refusal
  assertions the lock inverts.

**Tested:** the auditor's fixtures (their O1 cases and trailing file now
refused); each witness on an otherwise consistent line
(`TestReleaseGateMeasurements`, `TestReleaseGateReader`); the genuine
development lines re-read with no measurement problem and the same
verdicts; 49 mutations (48 caught; the survivor the documented T5a guard); the full uncached suite, vet, gofmt and
replay through dev.24.

---

## V0.2 Slice 5C.4n — final certification pipeline remediation (0.2.0-dev.24)

This slice remediates the independent 5C.4m audit (c456c94, committed
verbatim before any change) under the owner's decisions of 2026-10-20
(`docs/calibration/clinical-5c4n/`). It is the final planned remediation
before the focused 5C.4o release-gate audit. The maintainer's response
is `docs/audits/slice-5c4m-provenance-protocol-lock-audit-response.md`.
* **Not locked, not run, not frozen, not reviewed.**
* **Unchanged:** the clinical policy (`83d481b7…3948`), the simulator's
  behaviour, the severity thresholds, the scenario manifest
  (`250bc7fc…3ef4`) and the reserved seed protocol.
* **Registers 0.2.0-dev.23** for replay (`mini-seed52-dev23`).
* **Nothing reserved was run:** no strategy, reference or holdout seed,
  unseal record or PCG(20261201, ·) stream.

**Release blockers:**
* **M1, the trust boundary.** The reserved execution constructs,
  judges, adjudicates and certifies in one process and writes its files
  afterwards; certification is decided from how the lines were computed
  (trusted, family, manifest seeds), never from a file. A file read back
  is a `non-certifying analysis`: the auditor's fabricated, complete
  3,904-line file, on which every criterion PASSes, is `not certified`.
  The candidate, manifest, protocol, policy in force and simulator
  profile are verified against digests compiled into the build.
* **M2, measurement completeness.** Every line's measurements are
  checked for presence, cardinality, range, finiteness, seed identity
  and consistency before any criterion reads them. T2G (undetermined
  materiality), T5a (no prediction), T8 (matched counts by equality,
  exact per identity-witness kind) and T4 (exact seed sets) no longer
  read missing evidence favourably.
* **M3, the protocol.** It opens with the CERTIFICATION SCOPE: trusted
  in-process evaluation only; the study swap accepted, undetected, only
  there; authenticated order-to-study provenance for any future external
  records.
* **M4, errors.** Malformed settings, truncated or malformed input and
  unserializable results are errors; incomplete coverage fails the exit;
  no partial or development invocation is confirmatory.
* **M6, the validation helper.** It records the scenario's fingerprint
  and refuses records failing integrity; the regenerated access and
  follow-up evidence is byte-identical to 5C.4l's.
* **M5** (shared history with a larger candidate capacity): deferred;
  no manifest scenario has one.

**Corrections:** 20 distinct stable T1 groups, not "0 of 28"; the
near-margin results changed with 5C.4l's shared history (0.35 → 0.30
and 0.31 → 0.33 at +2.1 points; 0.56 → 0.65 at +2 points); "every seed
is valid" and "no bearing on the confirmatory run" corrected in place.

**Development checks** (PCG(20261021, ·), certified path, both
`development` / `not certified`): every line byte-identical to 5C.4l's;
no measurement problem; T8 passes after the per-kind correction found by
the first run; T5a passes on 200 seeds, T5b 0.33 at +2.1 points, 0 at +3.

**Tested:** the auditor's witnesses as regression tests and their own
eight files through the reader, 37 mutations (36 caught, one
equivalent, caught in combination), the full uncached suite, vet, gofmt
and the replay fixtures.

---

## V0.2 Slice 5C.4l — order provenance, inherited safety and reserved protocol closure (0.2.0-dev.23)

This slice remediates the independent 5C.4k audit (d06cdb9, committed
verbatim before any change), under the owner's decisions of 2026-10-20
(`docs/calibration/clinical-5c4l/`). The maintainer's response is
`docs/audits/slice-5c4k-clinical-safety-protocol-lock-audit-response.md`.
* **Not locked, not run, not frozen, not reviewed.** The focused 5C.4m
  review comes next.
* **The clinical policy is unchanged:** no threshold, window, limit,
  ceiling, follow-up or tolerance changed.
* **Registers 0.2.0-dev.22** for replay (`mini-seed52-dev22`).
* **v0.2-eval-4 is byte-identical.** The 5C.4d, e, f, h and j
  candidates keep their own rules, from their exact bytes.
* **Nothing reserved was run:** no strategy, reference or holdout seed,
  unseal record or PCG(20261201, ·) stream.

**Record integrity** (candidate SHA-256 `83d481b7…3948`):
* **K1, order provenance** (`orderProvenance: pairedOrigin`). Every
  order of a candidate's record, inherited or new, eligible or not, must
  reconcile with the paired plan's by origin: the same ID, receipt,
  wanted time, priority, modality and walk-in flag. Outcomes are never
  compared. A difference, a missing order or a candidate-only order
  makes the comparison INVALID. The audit's relink-and-retime bypass is
  INVALID (GO under the 5C.4j candidate).
* **K2, inherited safety** (`orderStudyIntegrity: contractInherited`).
  The inherited exemption is gone: every served order has exactly one
  post-takeover study, and order IDs are unique. Every inherited bypass
  of the audit is INVALID.
* **Not closed, reported for the owner:** an outcome-only study swap
  between two same-request orders around the horizon stays GO, because
  nothing immutable changes (`TestStudySwapLimitation`). Four options
  are set out; the reserved run is not affected (its records are built
  in process).
* **The fidelity model** gained a shared pre-takeover history, so
  paired records hold the same inherited orders, as the simulator's do.

**Adjudicator and protocol:**
* **K4:** no criterion passes with zero evidence; scenarios not run are
  incomplete coverage; the output records what was judged
  (`invocation`: `confirmatory` or `development`). The harness defaults
  to the manifest's seeds. The reserved execution is pinned in protocol
  §8 (expected 3,904 lines), not run.
* **K6:** the promised T1, T2 and T7 reports are emitted; W-ACCESS65
  checks the disclosed loss under a favourable Gate B; the follow-up's
  financial observation is measured: no financial measure or decision
  changed on the development seeds.
* **K5:** the false statement that no simulator partial window failed is
  corrected (seeds 901–903 failed locally; no seed or evaluation verdict
  changed), and local, seed-level and evaluation-level results are now
  reported apart.

**Development checks** (PCG(20261021, ·), seeds 901–910):
* every judged criterion passes; T5b 0.33 at +2.1 points, 0 at +3;
* provenance fails only on the identity witnesses' first seeds;
* 0 of 28 stable groups rejected; no stable seed fails the
  partial-window ceiling;
* the simulator records under the candidate are identical to 5C.4j's.

**Tested:** the audit's K1 and K2 witnesses as regression tests, the
5C.4i and 5C.4k witnesses, 23 new mutations (all caught) and the 5C.4j sets (21 of 21 applicable caught), the full uncached suite, vet,
gofmt and the replay fixtures.

**Next:** the owner's decision on the study swap, and the focused 5C.4m
review. Protocol lock only if clean; the reserved draw is authorized
separately after the owner inspects the lock commit.

---

## V0.2 Slice 5C.4j — clinical safety and reserved protocol remediation (0.2.0-dev.22)

This slice remediates the independent 5C.4i audit (4bead43, committed
verbatim before any change), under the owner's decisions of 2026-10-19
(`docs/calibration/clinical-5c4j/`). The maintainer's response is
`docs/audits/slice-5c4i-boundary-safety-lock-readiness-audit-response.md`.
* **Not locked, not run, not frozen, not reviewed.** The independent
  5C.4k review comes next.
* **Registers 0.2.0-dev.21** for replay (`mini-seed51-dev21`).
* **v0.2-eval-4 is byte-identical.** The 5C.4d, 5C.4e, 5C.4f and 5C.4h
  candidates keep their own rules, from their exact bytes.
* **Nothing reserved was run:** no strategy, reference or holdout seed,
  unseal record or PCG(20261201, ·) stream.

**Clinical policy** (candidate SHA-256 `9efa6e63…29e7c`):
* **I1, partial windows** (`partialWindows: rateCeiling`). Every
  nonempty partial window fails above a 5% observed severe rate,
  whatever its size. At or below 5% with fewer than 100 acquisitions it
  is NOT_JUDGED. 5/6, 5/10, 5/50, 5/99 and 1/6 were GO and are now
  NO GO on the full 40-seed evaluation; 5/100 and 0/6 stay GO.
* **I2, the order–study contract** (`orderStudyIntegrity: contract`).
  Unique study IDs; one order per study, with the same request, served
  and resolved at the acquisition; one study per served new order.
  Inherited work is exempt. The audit's three bypasses are now INVALID.
* **I3, a 45-day follow-up** (`followUpDays: 45`), observed by every
  recorder. Equality at the limit passes, a report due exactly at E and
  unfinished stays INSUFFICIENT, and nothing is fabricated.
* **I4, prefix-stable fidelity draws.** Observing longer changes no
  earlier outcome, in the fidelity model or in the simulator.

**Adjudicator:**
* **I5:** coverage is checked before judgment: exact groups, the
  remainder, replicates, seeds, validity and scanner errors.
* **I6:** raw T7 insufficiency; T2G per priority; witnesses judged by
  their targeted rule; W-ID-CHANGED moves the wanted time.
* **I7:** an exact binomial T5 prediction; the approved tolerances are
  pinned in the manifest (version 2).

**Development checks** (PCG(20261021, ·), seeds 901–910):
* **Results.** Every judged criterion passes. T5b at +2.1 points: 0.31
  (tolerance 0.40); at +3: 0.
* **False rejection.** 0 of 28 stable groups rejected; 0 of 1,120
  stable seeds failed the partial-window ceiling. Not material.
* **Simulator.** Routine reports not yet observable fall from 22 to 3.
  A recorder defect exposed by the contract was repaired: no-shows at
  the takeover with a rebooking queued.

**Tested:** the audit's witnesses as regression tests; 24 mutations, all
caught; the full uncached suite, vet, gofmt and the replay fixtures.

**Next:** the independent 5C.4k review. Protocol lock only if clean,
then one reserved run. Reserved calibration and the freeze stay NO GO.

---

## V0.2 Slice 5C.4h — reserved protocol closure and clinical boundary safety (0.2.0-dev.21)

This slice is the remediation of the independent 5C.4g audit (c55f3cb,
committed verbatim before any change), under the owner's decisions of
2026-10-19 (`docs/calibration/clinical-5c4h/`).
* **Not locked, not run, not frozen.**
* **Registers 0.2.0-dev.20** for replay (`mini-seed50-dev20`).
* **v0.2-eval-4 is byte-identical.** The 5C.4d, 5C.4e and 5C.4f
  candidates keep their own rules.
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream (the harness refuses PCG(20261201, ·)).

**Boundary safety: a clinical-policy revision, pending re-review.**
* **Partial windows** (`partialWindows: countCeiling`). A partial window
  below the minimum fails when it holds more severe reports than a
  compliant minimum window may: 5. A priority with nothing judged is
  INSUFFICIENT, never a vacuous pass. The audit's 640-report witness was
  GO and is now NO GO.
* **Horizon** (`safetyCohort: ordersByHorizon`). The absolute limits
  cover every study of an order wanted by the horizon, observed in the
  follow-up. Each study is in one state: observed violation, unresolved
  overdue, not yet observable (INSUFFICIENT), or completed in the
  follow-up. Unacquired orders are counted, and Gate A judges them.
  * **Witness:** the one-second witness was GO and is now NO GO.
  * **Link:** records carry a study-to-order link, exported without any
    draw.
  * **Integrity:** a post-horizon study without a valid link is INVALID.
* **Thresholds, minimums, anchor, allowance, Gate B and the safeguard:**
  unchanged.

**Protocol (proposed, not locked):**
* **The protocol itself.** `reserved-validation-protocol.md` gives every
  criterion an estimator, interval, family rule, numeric threshold,
  group and remainder handling, and PASS / FAIL / INCONCLUSIVE. A FAIL
  or INCONCLUSIVE is a documented NO GO; a post-result waiver is a
  deviation.
* **T2.** Harmed-seed detection must be at least 0.95. Whole-evaluation
  rejection is reported against the allowance's 0.371.
* **T5.** The tolerances are proposed (0.40 at +2.1 points, 0.01 at +3)
  and **await the owner's approval**.
* **The manifest.** `scenario-manifest.json` defines 49 scenarios:
  * disjoint streams and identical paired demand (no A85-against-B85
    pair);
  * random injections, record-reconciled;
  * exact margin witnesses.

  It is executed by `reserved_manifest_test.go` and judged by
  `reservedCriteria`.

**Reporting:**
* **G6:** occupied weeks are counted correctly, and unavailable unimaged
  fractions are null.
* **T7:** the raw per-gate insufficient seeds are listed.
* **G5:** the allowance interaction is documented
  (`allowance-interaction.md`) and the safeguard's overlap wording
  corrected.
* **G7:** the 40-seed impossibility statement is corrected.

**Development checks** (PCG(20261021, ·), simulator seeds 901–910):
* **Construction.** Every scenario was constructed and paired; all 14
  witnesses met their expectation. One construction defect was found
  and repaired: modality drawn from the booking stream made v4 case
  selection reject 4 of 20 stable groups. After the repair, none.
* **Near the margin.** Gate B accepted 0.32 at +2.1 points (predicted
  0.31) and 0 at +3.
* **Follow-up.** It leaves 1–6 routine studies not yet observable on 9
  of 10 no-action seeds (INSUFFICIENT). A 45-day follow-up is offered to
  the owner.

**Tested:** the boundary, horizon, allowance, reporting, criteria and
manifest witnesses; the mutation checks; the full uncached suite, vet,
gofmt and the replay fixtures.

**Next:** the owner's decisions:
* the T5 tolerances;
* the follow-up length;
* a targeted independent re-review of the boundary-safety revision.

Then the lock commit, and the reserved run exactly once. Reserved
calibration and the freeze stay NO GO until then.

---

## V0.2 Slice 5C.4f — volume-aware clinical severity windows (0.2.0-dev.20)

Each priority's severity window now matches its volume (owner decisions
of 2026-10-19; `docs/calibration/clinical-5c4f/`). The evaluation window
changed; the severity standard did not.

* **Not frozen.**
* **Registers 0.2.0-dev.19** for replay (`mini-seed49-dev19`).
* **v0.2-eval-4 is byte-identical.** The 5C.4d and 5C.4e candidates'
  exact bytes still evaluate under their own rules.
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream.

**Windows.** Each priority's window is set explicitly in the policy
(`weeklyWindowWeeks`):

| Priority | Weekly gate | Minimum | What remains for it |
|---|---|---|---|
| urgent | none | none | the per-seed 5% severe rate and the absolute 24 h limit |
| soon | fixed four-week calendar blocks | 100 | |
| routine | fixed calendar weeks | 100 | |

* **Blocks:** anchored at the first complete Monday, consecutive and
  never slid or pooled.
* **Below the minimum:** a complete window is INSUFFICIENT; a partial
  window is NOT_JUDGED, disclosed and never a pass.
* **Safeguard:** approved at 10 / 6 / 5 points.

**Development evaluability (seeds 901–910, no action):**
* **Urgent:** every seed is evaluable.
* **Soon:** complete blocks hold 127–218 acquisitions (mean 166), and
  none is insufficient.
* **Routine:** none of 200 weeks is insufficient.
* **The no-action practice fails on genuine delays:** soon blocks fail
  on 9 seeds (reports 42–45 h past due, severe rates up to 9.4%), and
  urgent absolute violations occur on 8 seeds.
* **Anchoring:** moving the anchor 1–3 weeks changes which seed fails
  on 1–2 of 10 seeds, never evaluability.

**Tested:** the window witnesses (`severity_window_test.go`), the
policy validation, 11 mutations (all caught), the full uncached suite,
vet, gofmt and the replay fixtures.

**Next:** a narrow independent review of the final v5 policy and its
statistical claims. The confirmatory protocol is complete and not run.
The reserved calibration and the freeze are NO GO until that review is
done.

---

## V0.2 Slice 5C.4e — empirical clinical acceptance and weekly severity closure (0.2.0-dev.19)

The owner's decisions of 2026-10-19 (`docs/calibration/clinical-5c4e/`):
Gate B changes from a population confidence bound to an **empirical
benchmark acceptance rule**, with a per-seed safeguard; the weekly
severity moves to **fixed Monday–Sunday calendar weeks**.

* **Not frozen.**
* **Registers 0.2.0-dev.18** for replay (`mini-seed48-dev18`).
* **v0.2-eval-4 is byte-identical;** the 5C.4d candidate's exact bytes
  still evaluate under their own rules.
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream.

**Gate B (`lostDemand.rule = "empirical"`).**

* PASS when the observed seed-averaged increase on the 40 paired seeds
  is at most 2 points (exact rational), FAIL above it; unmatched, thin
  or too few seeds INSUFFICIENT, never PASS.
* No bound, confidence or population enters the decision or its
  reasons. The `betting` bound is reported beside it as a labelled
  research diagnostic; `t` is refused even for that.
* The seed differences' distribution and both absolute unimaged
  fractions are reported.
* Every comparison states its claims apart: empirical acceptance on
  the paired seeds; population confidence not claimed; absolute
  adequacy not established. Every favourable verdict leads with the
  access limitation and the empirical scope.

**Per-seed safeguard (`maxSeedIncrease` 10 / 6 / 5 points).** A seed
whose own increase exceeds it fails in the shared allowance (4 of 40),
so improvements elsewhere cannot hide concentrated harm. Thresholds
from a prespecified rule on development-only evidence
(`seed-safeguard.md`): z(0.999) times the largest no-effect sd among
the development seeds, floored at 5 points.

**Weekly severity (`weeklyPartition = "calendar"`).** Every complete
week judged on its own: INSUFFICIENT below the minimum, FAIL above 5%;
a partial boundary week below the minimum NOT_JUDGED, never PASS; no
pooling. **Stop condition:** on the development seeds every complete
urgent week (6.7 acquisitions a week against a minimum of 20) is
insufficient, so every seed lacks evidence and no plan can pass; soon
is insufficient on every seed too. Reported for the owner; the
minimums and partition are unchanged.

**Tested:** the 20 required witnesses (`empirical_acceptance_test.go`),
policy validation, the safeguard rule and its development derivation
(`seed_safeguard_test.go`), the mutation checks, the full uncached
suite, vet, gofmt, the replay fixtures.

**Open:** the weekly minimums; the Astra review of the empirical rule
and safeguard; the reserved confirmation (NO GO); the freeze (NO GO).

---

## V0.2 Slice 5C.4d — audit remediation and seed-level lost-demand inference (0.2.0-dev.18)

This slice remediates the independent 5C.4c audit (kept verbatim; the
response is in `docs/audits/slice-5c4c-access-gates-audit-response.md`)
under the owner's decisions (`docs/calibration/clinical-5c4d/`).

* **Not frozen.**
* **Registers 0.2.0-dev.17** for replay (`mini-seed47-dev17`).
* **v0.2-eval-4 is byte-identical.**
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream.

**Fixes:**

* **F2:** the validated named end capture.
* **F3:** unknown outcomes and priorities are INVALID, never a panic or
  a pass.
* **F8:** every clinical fraction is finite and in [0, 1].
* **F9:** weekly spans are given in calendar weeks and dates.

**Identity.** Evaluation orders carry the simulator's immutable
`origin`, drawn from nothing. Under three plans it is unique within a
record and identical, request for request.

**Gate B:**

* now seed-level: one difference per matched seed, estimand their mean;
* the invalid conditional bound is removed;
* two procedures implemented: `betting` (finite-sample) and `t`
  (approximate).
* **No valid bound can pass the 2-point margin at 40 seeds.** The best
  any valid method can reach is 7.2 points; `betting` reaches 20.7 (2.0
  needs 400 seeds).
* So with the guaranteed method **Gate B is INSUFFICIENT for every
  plan.** This is reported for the owner's decision, not weakened.
* **Coverage study:**
  * `betting` covers in every scenario;
  * `t` is near nominal in smooth scenarios but covers only 0.70 (and
    accepts a 3-point increase 30% of the time) under all-or-nothing
    seed shocks.

**Verdict limitation.** Every favourable verdict states "Access
non-deterioration only; absolute access adequacy not established." with
access statements: timeliness, absolute unimaged fractions for
candidate and baseline, and the bound.

**Weekly rule:**

* the strict 5% ceiling and forward pooling are kept;
* backward pooling is reported, with a partition-sensitivity flag;
* a valid partition alone can change the verdict, in synthetic
  witnesses and on 1 of 10 no-action simulator seeds (urgent): a
  **freeze blocker**.

**Protocol.** The reserved protocol is updated: near-margin pairs,
coverage, clusters, identity, every audit witness. It is **not
runnable** until the owner decides the Gate B method and the weekly
partition rule.

---

## V0.2 Slice 5C.4c — clinical access and weekly severity gates (0.2.0-dev.17)

The owner's modified option B and the volume-normalized weekly
severity, in the v5 evaluator
(`docs/calibration/clinical-5c4c/access-gates.md`).

* **Not frozen:** the candidate policy is still a proposal.
* **v0.2-eval-4 is byte-identical** (`TestV4Neutrality`).
* **Earlier behaviour kept:** v5 policies without the new fields (every
  calibration harness) are unchanged, and the committed calibration runs
  reproduce (`TestOCFidelityModelUnchanged`).
* **Registers 0.2.0-dev.16** for replay (batch mode `v0.2_dev16`,
  fixture `mini-seed45-dev16`).
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream.

**The rules:**

* **Gate A (timeliness):** served, open and late-cancelled eligible
  orders, from the original wanted time, at ≥ 95% (urgent 48 h; soon 7
  and routine 30 days, provisional). Lost and left orders are counted
  apart.
* **Gate B (lost demand):** unimaged demand (lost, left or ended by the
  patient) against the no-action baseline, per priority.
  * Paired only on identical eligible demand: identity by receipt and
    wanted times, priority, modality and walk-in. The simulator's demand
    is shown exogenous across plans (`TestDemandExogenousAcrossPlans`).
  * The exact one-sided 95% bound of the pooled increase must be at most
    2 points.
  * Unmatched demand is insufficient (NO GO).
  * It is a non-deterioration gate: absolute fractions are always
    reported, and adequacy is not certified.
* **Weekly severity:** severe-delay rate ≤ 5% in blocks of Monday weeks
  of at least 20, 40 or 100 acquisitions; low volume is pooled, a short
  remainder is joined, and too few is insufficient.
  * Severity is defined apart from the absolute limits: urgent 12 h,
    soon 24 h, routine 72 h past due.
* **Maximum wait:** reported only.

**Synthetic model:**

* the takeover instant is a field, and the aligned takeover gives
  calendar evaluator weeks (the old ones ran Wednesday to Wednesday);
* demand can be drawn from its own stream, so that paired practices
  share it.

**Tests:**

* cases C1–C10 and C7a (lost, delayed, loss against timeliness,
  cancellation manipulation, rebooking, horizon, low-volume weeks,
  overlapping failures, missing baseline, noncomparable demand);
* the updated witnesses W1–W12;
* validation;
* 10 of 10 mutations caught.

**No-action simulator:**

* Gate A passes the baseline: urgent 0.993, soon 0.965, routine 0.979.
* Gate B reports 65% / 38% / 14% lost or left, not certified.
* The weekly 5% rule fails every seed on soon (weekend concentration).

**Next:**

* the reserved protocol is complete but not to be run;
* the Astra High review package is prepared;
* the owner approvals are listed in access-gates §7.

---

## V0.2 Slice 5C.4b closure — clinical acceptance design (0.2.0-dev.16)

The final specification step before v0.2-eval-5
(`docs/calibration/clinical-5c4b/acceptance-design.md`).

* **Not frozen, and blocked on an owner decision:** urgent access
  cannot be justified as specified.
* **No version change:** only tests, test data, a candidate policy
  outside `docs/evaluation` and documents changed.
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream. The simulator ran with no action on
  the development seeds.

* **Order lifecycle** (`order-lifecycle.md`): every state, its
  initiator, and what the evaluation record carries and omits.
  * The order and acquisition mixes reconcile through loss: **65% of
    urgent orders are lost at the first offer**. 75% of urgent orders
    tolerate only the same day, and booking reserves nothing for urgent
    work.
  * About two thirds of urgent losses are staffing, the rest booking
    structure. No plan action changes the booking structure.
* **Access denominators** (`access-denominators.md`): rules A–J as
  built, with no survivorship.
  * Lost and left orders are untimely. Open orders are untimely past
    the deadline, undecided before it, and insufficient is not a pass.
  * A patient outcome before the deadline is withdrawn, after it
    untimely.
  * The wanted time is fixed.
  * Inherited orders are timed from the takeover.
  * **The calendar clock is recommended.** The open-hours clock would
    hide weekend delay.
* **No-action simulator access, every eligible order:** urgent 48 h
  **0.33**, soon 7 days 0.58, routine 30 days 0.83. Of served urgent
  orders, 99% are within 48 h.
  * A 90% or 95% urgent threshold rejects every plan.
  * Options A (as built), B (timeliness of served and open orders, plus
    a per-priority loss gate against the baseline: recommended) and C
    are set out for the owner.
* **Reporting limits on the no-action records** (utilization 0.95, no
  weekend reading):
  * urgent 24 h fails 8 of 10 seeds (the weekend gap, as intended);
  * soon 48 h and routine 14 days hold;
  * the weekly count of 5 fails every seed for routine (20–37 in the
    worst week): owner decision.
* **Candidate policy** (`candidate-policy-v5.json`): v0.2-eval-4 plus
  the clinical section, a proposal and not in force.
  `TestCandidatePolicyFile` checks that it changes no v4 rule.
* **Witnesses W1–W12** (`clinical_witness_test.go`): the audit's
  delays, lost, cancelled, open and rescheduled urgent orders, the
  Saturday order, observation boundaries, contradictory data and
  manipulation. Each has its expected classification, denominator,
  numerator and outcome, and all pass.
* **Reserved-stream protocol** (`reserved-validation-protocol.md`): a
  draft, not runnable until the owner's decisions.
  * PCG(20261201, ·), verified never used.
  * It needs one model correction first: the synthetic records'
    evaluator weeks start on Wednesday.

---

## V0.2 Slice 5C.4b, concluded — simulator-fidelity calibration (0.2.0-dev.16)

The owner's final calibration correction before choosing the
v0.2-eval-5 values (`docs/calibration/clinical-5c4b/fidelity-protocol.md`,
committed before any result on its streams; findings in
`findings-fidelity.md`). There is no version change: only test code,
test data and documents changed.

* **No v5 policy and no value chosen.**
* **The simulator was measured, not a strategy.** The evaluation
  scenario's own practice ran with no action on the development seeds,
  through the holdout guard. No reference or holdout seed was touched.
  The profile is `internal/eval/testdata/sim-profile-5c4b.json`.
* **What the profile shows:**
  * 51.7 acquisitions a day, Saturday 5.9%;
  * acquisition mix 1.9 / 11.6 / 86.5% (the order mix is 4.4 / 14.9 /
    80.7%, most urgent orders being lost);
  * reading utilization 0.952;
  * soon breach 0.36;
  * worst delays about 40–45 h.
* **The fidelity model:**
  * the simulator's demand, mix, weekday distribution and acquisition
    hours (slot-timed);
  * its two radiologists in their reading blocks, with stochastic read
    times by modality, the shift-end rule, overrun debt and
    urgent_then_fifo;
  * urgent weeknight lanes and a weekend reader;
  * regimes on the core reading capacity;
  * a 181-day warm start.
  * The earlier models and results are unchanged.
* **Results, 24 scenarios and 36,000 seeds.**
  * **Calibration:** core load 0.85 and 0.92 achieved. Queues are
    stable; takeover backlogs are 1 study at p50.
  * **At 0.85:**
    * no severe delay;
    * lenient and moderate bundles reject nothing, every failure being
      v4's backlog rule;
    * urgent 24 h, soon 48 h and routine absolute limits are admissible
      at zero tolerance.
  * **At 0.92:** 0.2–0.4% of seeds violate urgent 24 h.
  * **Weekend gap:** worst urgent about 40 h at equal staffing and at
    equal utilization. A 24–36 h urgent limit violates about 70% of
    E3-like seeds; 48 h almost none.
  * **Urgent access within 1 day at 95%** fails every stable seed
    (Sunday closure). Within 2 days it fails 2–3%.
  * **Detection:** harmful reading losses are caught by the clinical
    gates; L10 at 0.85 only by v4's soon gate (clinically mild); imaging
    losses are absorbed.
* **The mirror against the simulator.** At the simulator's own load the
  mirror's breach and severe shares are 1.5–3× the simulator's. The
  protocol's validity criterion fails, and no simulator-level validity
  is claimed.
* **Tests:**
  * `TestSimProfileDerived`, `TestOCReaders`, `TestOCReaderRules`,
    `TestOCFidelityCalibration` and
    `TestClinicalOCFidelityAgreesWithEvaluate`;
  * 13 of 13 mutations caught;
  * the full uncached suite, `go vet` and gofmt pass at 18b21d6.

---

## V0.2 Slice 5C.4b, continued — calibration correction (0.2.0-dev.16)

The exploratory operating characteristics were run and reproduced. The
owner then asked to hold the threshold choice and correct their
calibration first. No version change: only test code and documents
changed, and the binary is the same.

* **No v5 policy, no threshold chosen.** Nothing else was run: no
  strategy, development, reference or holdout run, and the confirmatory
  streams are untouched.
* **The original run** (`oc-v0.2-eval-5-exploratory.json`) is committed
  as generated, with joofi and the Pi byte-identical.
  * Its findings carry an erratum. The stable practice measured 0.67
    (0.74 over the last 30 days), not the intended 0.85.
  * Its rate was sized over 66.5 reading hours but it read 70.5,
    cancellations were not counted, and it started empty.
* **The correction** (`docs/calibration/clinical-5c4b/correction-protocol.md`,
  committed before any corrected result):
  * utilization is measured per seed, and the rate is set from the
    target over the practice's own hours;
  * the practice runs 181 days before the takeover and inherits the
    work open then;
  * it reads in the simulator's urgent_then_fifo order;
  * regimes at 0.85 and 0.92, on new streams PCG(20261102, ·).
  * Each change is a field whose zero value is the original model. The
    original run still reproduces byte for byte.
* **Results** (`findings-corrected.md`):
  * **Utilization** measured 0.845–0.849 and 0.914–0.919.
  * **At 0.85:**
    * no v5 clinical gate value fails a stable seed;
    * the bundles' 40-seed rejection is at most 0.002, from v4's own
      gates;
    * urgent absolute limits of 24 h or more, and soon of 48 h or more,
      are admissible at zero tolerance. Soon 24 h is not: one seed of
      2,500 reached 34.9 h.
  * **At 0.92:** v4's 4-hour soon breach gate rejects 85–89% of
    seeds, with severe delays below 0.1%.
  * **Weekend, at equal utilization:** no weekend reading moves the
    worst urgent delay from 14 h to 42–44 h. An urgent limit of 36 h or
    less makes it a violation; 48 h does not at 0.85.
  * **Deterioration:**
    * reading losses that harm (L20, D, K, L10 at 0.92) are caught by
      the clinical components under every bundle;
    * L10 at 0.85 is clinically mild (severe below 0.05%) but rejected
      by v4's soon turnaround gate;
    * imaging losses are absorbed.
* **Against the simulator** (read, not run):
  * **Matched:** the worklist rule, imaging hours, demand weekdays,
    targets, and the warm start.
  * **Not matched:** individual readers and read times, modalities,
    booking, and the Saturday load (12.8% of acquisitions against about
    5%).
  * No simulator-level validity is claimed.
* **Tests.**
  * `TestSimPick` checks the simulator's worklist rule.
  * `TestOCCorrectedCalibration` checks utilization and inherited work
    on the debug stream.
  * `TestClinicalOCCorrectedAgreesWithEvaluate` checks exact agreement
    with the production evaluator.
  * 10 of 10 mutations were caught. One of them is a sentinel bug
    (day −1 as "no slot"), found and fixed before the protocol was
    committed.
  * The full uncached suite passes at d11964f, `TestEvaluateMatrix`
    included, as do `go vet` and gofmt.

---

## V0.2 Slice 5C.4b — the v0.2-eval-5 clinical evaluator (0.2.0-dev.16)

This slice implements the clinical acceptance rules specified in 5C.4a
(`docs/evaluation/proposals/v5-clinical-acceptance.md`, implementation
decisions in its §J). It also fixes the protocol for the combined
synthetic operating characteristics
(`docs/calibration/clinical-5c4b/protocol.md`), committed before any
combined result. Design §13.11.

* **No v5 policy is frozen.** The owner chooses the values from those
  operating characteristics (owner decision: build first, values
  later; the weekend question after the operating characteristics).
* **Unchanged:** v0.2-eval-4, the simulation and every strategy.
* **Nothing else was run:** no strategy, development, reference or
  holdout run.

* **Rules.** These are reached only by a policy of schema
  `dreamteam.evalpolicy/v5` (`internal/eval/clinical.go`):
  * absolute past-due limits by priority, from exact timestamps, with
    unfinished studies censored at the end of observation, at zero
    tolerance;
  * the severe-delay rate and severe delays per Monday week;
  * the timely-acquisition fraction from the wanted time, with open,
    lost and abandoned orders untimely, a patient's withdrawal before
    the limit excluded, and a rebooking never resetting the clock;
  * the maximum wait;
  * inherited work timed from the takeover or from its own due time;
  * one shared seed allowance over service, severity, access and
    backlog, each seed counted once, with the absolute limits kept
    apart (NO GO, clinical safety);
  * the sampled age limit reported only;
  * the M3 fix (an insufficient baseline is not restored);
  * the G10 integrity check (an order imaged before its wanted time).
* **v0.2-eval-4 neutrality.** `TestV4Neutrality` pins the evaluation of
  a varied synthetic input. It covers GO, NO GO, service, backlog and
  case-selection paths, with a digest recorded before any evaluator
  change. The digest is unchanged.
* **Acceptance cases G1–G6 and G10–G14** on synthetic records. Each
  harmful case has its acceptable counterpart and exact boundaries.
  * The audit's witnesses (six urgent reports eight days late; a report
    348 h past due; thirty urgent acquisitions thirty days late) now
    fail under v5 and still pass under v0.2-eval-4.
  * Four seeds failing service plus four others failing backlog pass
    v0.2-eval-4's separate allowances and fail v5's shared one.
* **The combined synthetic practice** (`oc_clinical_test.go`).
  * It adds an imaging and booking stage to the 5C.3 reading queue:
    wanted times, a patient-tolerance loss, cancellations, and
    acquisitions sorted in time (the M1 fix; the 5C.3 queue itself is
    unchanged).
  * It produces complete records, which the production evaluator
    judges.
  * Its tabulation agrees exactly with `planReport` on sampled seeds of
    a stable, a structural and a harmful scenario.
  * The stable practice's reading hours cover its imaging. The
    scenario's own rota failed v0.2-eval-4's service gates on a debug
    stream, so it is kept as a structural variant. Both are recorded in
    the protocol.
* **Session replay.** 0.2.0-dev.15 is registered: batch mode
  `v0.2_dev15`, with fixture `mini-seed43-dev15` recorded by the
  31a2573 binary.

**Tests.**

* New: `TestV4Neutrality`, eleven `TestClinical*` tests,
  `TestClinicalOCAgreesWithEvaluate`, and `TestDev15RecordReplays` and
  `TestDev15RecordContinues`.
* Sixteen targeted mutations of the new rules are each caught. Four
  survived the first pass and were closed with stronger tests and one
  shared severity helper.
* Removing the dev.15 registration fails the replay tests.
* **An existing intermittent failure, fixed.**
  * Since 5C.0b, `TestEvaluateMatrix` shared one strategy-pin map
    across its cases.
  * One case deleted a pin, so in some random map orders another case
    failed. It failed in the first full run of this slice.
  * Each case now gets its own copy: 20 of 20 runs pass, where before
    20 runs gave several failures.

**Limitations.**

* **Values.** The values are not chosen; the exploratory run follows
  in its own commit.
* **Synthetic model.** It is simplified: one modality pool, day-level
  booking, and constant finance.
* **Saturday routine bookings.** In the practice without weekend
  reading, Saturday routine bookings follow the wanted day uniformly,
  which may overstate its routine breach rate.

## V0.2 Slice 5C.4a — audit record, holdout containment, clinical policy proposal (0.2.0-dev.15)

The independent audit of v0.2-eval-4 at 7707dbd gave **NO GO** for the
confirmatory run. This slice preserves the audit, contains the ordinary
tests' use of holdout seeds, and specifies, without activating, the
clinical acceptance rules for a successor policy. Design §13.10.

Unchanged:

* v0.2-eval-4, which stays in force byte for byte, with every earlier
  policy;
* the simulation, scenarios, demand, staffing, economics and TAT
  targets;
* every strategy file and development result.

Nothing was run on a strategy, a reference seed or a holdout seed. There
is no unseal record, and no primary strategy is chosen.

* **The audit** is committed separately (e54cdca), as
  `docs/audits/slice-5c3-eval4-framework-audit.md`.
  * The auditor's read-only findings are kept as delivered.
  * A separate maintainer appendix records the four executed witnesses:
    six urgent reports 8 days late, 30 urgent acquisitions 30 days late,
    a report 348 h past due seen as 334 h, and an insufficient baseline
    counted as restored. Each got the verdict the audit predicted. It
    also records code checks of H3 and H6.
* **Exposure**, recorded in `docs/evaluation/holdout-exposure-5c4a.md`.
  * From Slice 5C.0b, `TestHoldoutRecordersGuarded` built a
    mini-scenario world on the first real holdout seed, ran it to the
    takeover only, and checked that the recorders refuse it.
  * Guard tests also regenerated the seeds in memory.
  * There was seed use, but no outcome was disclosed and no
    confirmatory run took place. The holdout is kept (owner decision).
* **Containment.**
  * The guard tests use synthetic holdout manifests.
  * `NewEvalWorld` refuses a holdout seed before any world is built.
    The development runner and the recorder tests use it.
  * `Authorize` refuses a sealed holdout before regenerating seeds.
  * `TestHoldoutContainment` reads all Go source and fails on any
    unreviewed site that names the manifest, regenerates its seeds or
    authorizes with the repository's paths.
* **Clinical policy proposal**, in
  `docs/evaluation/proposals/v5-clinical-acceptance.md`. It is not
  active.
  * Exact-timestamp severe-delay rules: the age limit made zero
    tolerance, priority maximums, and the incidence and weekly
    concentration of severe delays.
  * Wanted-to-acquisition access, with no survivorship: a timely
    fraction and a maximum wait.
  * One shared four-seed clinical allowance, with absolute limits kept
    apart.
  * v0.2-eval-4's backlog rule kept; its 150% allowance documented as a
    ceiling of 250% of the adjusted reference.
  * Every numerical limit is an owner decision. The proposal states the
    scenario's structural weekend delay (urgent up to about 40.5 h past
    due without weekend reading). It also states the cost of the shared
    allowance: under independence the stable seasonal-bursty queue's
    40-seed false rejection rises from about 0.05 to 0.22.
  * A 15-case synthetic acceptance test plan is set out for 5C.4b.
* **README.** v0.2-eval-4's README states the 250% ceiling and that the
  age limit is sampled, and records the audit and the proposal.
* **Session replay.** 0.2.0-dev.14 is registered: batch mode
  `v0.2_dev14`, with fixture `mini-seed41-dev14` recorded by the 7707dbd
  binary.

**Tests.**

* New: `TestHoldoutContainment`, `TestGuardSeeds`,
  `TestAuthorizeSealedFirst`, and `TestDev14RecordReplays` and
  `TestDev14RecordContinues`.
* Rewritten on synthetic fixtures: `TestHoldoutRecordersGuarded` and
  `TestGuardRepo`.
* `TestHoldoutSealed` keeps only the integrity checks of the real
  manifest.
* Six mutations are each caught:
  * the guard removed from `NewEvalWorld`;
  * a new manifest-path reference;
  * a non-synthetic `HoldoutSeeds` label;
  * a new `Authorize(RepoPaths)` site;
  * the old real-manifest guard test restored;
  * the dev.14 registration removed.
* The pre-fix `Authorize` order fails `TestAuthorizeSealedFirst`.

**Limitations.**

* The guards still regenerate the real seeds in memory in order to
  refuse them.
* The manifest lets anyone with the repository regenerate the seeds.
* The containment test checks names, not data flow.
* Custody, approval binding and run-once are Slice 5C.4c's.

## V0.2 Slice 5C.3 — backlog-gate operating characteristics (0.2.0-dev.14)

The owner's backlog decisions on the Slice 5C.2 findings, and a backlog
rule selected by a prespecified protocol on synthetic queues. Both are
frozen as **policy v0.2-eval-4** (`docs/evaluation/policy-v0.2-eval-4.json`, SHA-256
4e277c72…b58ffb6fb3). Design §13.9; rules in
`docs/evaluation/README.md` §4; protocol, findings and re-evaluation in
`docs/calibration/backlog-5c3`.

Unchanged:

* the breach gates (12% / 32% / 6%), the joint seed rule and its 10%
  allowance;
* case selection, solvency, the dated funding test, and T90 as the
  primary endpoint;
* the simulation, staffing, demand, economics and TAT targets;
* the holdout, which stays sealed. No reference seed was run, and there
  is no unseal record.

v0.2-eval-3 (digest 0cf4884f…), its README, its operating
characteristics and its development report are kept byte for byte. No
primary strategy is chosen.

* **Diagnosis.** v0.2-eval-3 reproduces 5C.2's stable-queue result
  exactly (191 and 263 of 300). Its false failures are the peak alone
  (50% stationary, 27% seasonal) and the peak with the mean (13%, 57%).
  With bursty days, the late-work means also fail.
* **The rule.**
  * level: the workload-adjusted block mean, within max(150%, 2
    studies);
  * overdue, stale and exposure: block medians, at their old
    tolerances;
  * stale cap: 3 studies, in every block from the reference on;
  * age: 14 days past due, unchanged;
  * reported only: the peak, recovery time and the old mean check.
* **How it was selected.**
  * 720 candidates were scored by the production judge (`judgeBacklog`)
    on 18 synthetic scenarios, 1000 fresh-stream seeds each.
  * 84 met T1 (every stable queue's 40-seed rejection ≤ 5%) and T2
    (every deterioration detected on ≥ 90% of seeds).
  * The protocol's ladder and tie-break order chose one.
  * `TestOperatingCharacteristics` checks that the frozen policy is that
    candidate.
* **Operating characteristics.**
  * Stable queues: 0–3.8% of seeds unstable (v0.2-eval-3: 61–92%). The
    worst 40-seed rejection is 0.017.
  * Deterioration scenarios: 97.6–100% unstable.
  * Not detected: a capacity step that leaves the queue stable (10%
    lost: 23%), and slow erosion (46%). A seasonal queue at utilization
    0.92 is rejected at 40 seeds with probability 0.56 (sensitivity).
* **Clustering interpretation** (README §2).
  * Empirical acceptance, population reliability, false-rejection
    probability and power are reported apart.
  * The holdout is a strict empirical acceptance test.
  * The independent-breach figures are a lower bound on false
    rejection, not a guarantee.
* **Development re-evaluation** (`p6-eval4`, seeds 901–910, strategies,
  records and horizon unchanged):
  * D2 is CONDITIONAL GO (transition, not yet repaid, restoring service);
  * E2 and E3 are CONDITIONAL GO (service purchase);
  * F, G and H are NO GO on solvency; B–E are NO GO on service;
  * the no-action plan now fails backlog on its overdue median (3 of 10
    seeds);
  * service, funding and solvency are identical to v0.2-eval-3.
* **D2 against E3.**

  | | D2 | E3 |
  |---|---|---|
  | Pooled urgent | 9.0% | 2.8% |
  | Worst seed urgent | 11.6% | 3.7% |
  | T90 Δ | −$18.8k | −$23.2k |

  Both are backlog-stable on 10 of 10 seeds.
* **Registration.** 0.2.0-dev.13 is registered (batch mode `v0.2_dev13`).
  Its fixture is `testdata/sessions/mini-seed39-dev13.*`, recorded by
  the dev.13 binary from a clean worktree of dc1c043.

**Tests.**
* `TestBacklogStability` covers each protocol case; `TestPolicyFrozen`
  checks the v4 → v3 → v2 → v1 chain.
* `TestSeasonalBacklog`, `TestBacklogSafeguards` and
  `TestBacklogTrendRotaAligned` keep the earlier rules' cases under
  v0.2-eval-3 and assert v0.2-eval-4's outcomes beside them.
* **Fail-before-fix:** 16 targeted mutations, each breaking one new rule
  (informational counted as hard, statistics, the cap and its
  reference, level tolerance and scaling, the median, recovery,
  persistence, policy validation, the superseded schema, the covering
  block). All 16 are caught. "Level statistic ignored" survived the
  first pass and is caught after a mean-versus-median surge case was
  added.

**Verification.**
* `p6-eval4` ran on joofi and reproduced on the Pi byte for byte:
  `run.json`, report and trajectory.
* The operating characteristics were generated on the Pi (old candidate
  format) and on joofi; every number is identical. The backlog part,
  recomputed on the Pi (`DT_OC_BACKLOG=1`), equals joofi's committed
  file byte for byte.
* The full uncached suite passes (12:39); `go vet` is clean for every
  tag (none, calibration, acceptance), and gofmt is clean.
* The dev.13 tests fail without the registration.
* Holdout sealed (no `holdout-unseal.json`); historical references and
  v0.2-eval-3 files unchanged.

**Limitations.**
* The calibration rests on a simple hourly synthetic queue.
* The candidate family was shaped by exploratory sweeps, though the
  targets were fixed before them and the confirmatory run used fresh
  streams.
* One protocol line was corrected after the run: the per-seed limit is
  5.06%, not the hand estimate of 3.4%. The selected rule's worst stable
  scenario (3.8%) meets the actual criterion.
* Development verdicts rest on 10 seeds and are exploratory.

---

## V0.2 Slice 5C.2 — policy feasibility, seasonal backlog and cluster-aware service (0.2.0-dev.13)

The owner's corrections to the rules Slice 5C.1 showed to be infeasible
or statistically misleading, frozen as **policy v0.2-eval-3**
(`docs/evaluation/policy-v0.2-eval-3.json`).

Unchanged:

* the breach gates (12% / 32% / 6%), the joint seed rule and its 10%
  allowance;
* case selection, solvency, and T90 as the primary financial endpoint;
* the simulation, staffing economics, demand, pricing and TAT targets;
* the holdout, which stays sealed. The reference seeds were not run.

v0.2-eval-2 (digest 5dbab462…) and its README are kept byte for byte,
and its 5C.1 reports are unchanged. No primary strategy is chosen.
Design §13.8; rules in `docs/evaluation/README.md`; diagnostics and the
re-evaluation in `docs/calibration/policy-5c2`.

* **Funding: a dated settlement test** (§5). It replaces "cash ≥ the
  whole bridge, 14 days of future wages included, at every instant",
  which the no-action practice failed on 76% of samples while paying
  every bill on time. A seed is funded when:
  * cash after every posting is at or above the floor, and nothing is
    past due (solvency, unchanged);
  * observation runs at least 31 days past the horizon (the longest
    payment cycle: payables on the 1st), so every obligation recognized
    by the horizon is seen paid from actual collections;
  * at the end, cash covers the committed liabilities outside the cycle
    (severance payable, contract minimums, fees in flight).

  Receivables are never forecast. The bridge is reported split by kind,
  with the old 14-day coverage as a liquidity-risk indicator.
* **Backlog: workload-adjusted** (§4). The mean, peak, overdue and
  exposure checks compare a block with the reference scaled by its own
  acquisitions (Little's law), exactly. Stale and persistent work stays
  absolute. Rota alignment, the covering block and every check are kept.
* **Cluster-aware service** (§2). Every gate reports:
  * day (weekday-adjusted) and week design effects;
  * a seed-cluster-robust pooled interval;
  * for the joint rule, the empirical pass fraction and the exact upper
    bound on the population failing-seed fraction.

  The operating characteristics cover independent, variable-volume,
  day-clustered and week-clustered models, with power at unacceptable
  rates. The allowance is read prospectively: a pass is empirical, not
  a confidence statement.
* **Transition, not yet repaid** (§10): CONDITIONAL GO for a plan passing
  every hard gate whose T90 increment is not positive but whose days
  31–90 increment is.
  * T180 is never promoted to financial GO.
  * Every comparison reports its financial evidence: T90, operating
    window and T180, each absolute and incremental, plus transition
    obligations at T90.
  * Service restoration is stated as a fact, whatever the verdict.
* **Development storage.** New runs commit `run.json`, which pins the
  records, report and trajectory by digest, and the generated
  `report.md`. The detailed report and trajectory go to an ignored
  `detail/` directory and are checked against `run.json`.

**Re-evaluation** (set `p5-eval3`: A–H, D2, E2, E3, strategy files
unchanged; `docs/calibration/policy-5c2/reevaluation.md`):

| | v0.2-eval-2 | v0.2-eval-3 |
|---|---|---|
| Funding-gap seeds, every plan | 10 of 10 | 0 |
| Service joint rule | as before | unchanged (by design) |
| Backlog-stable seeds | A 0, D2 2, E2/E3 1 | A 1, D2 2, E2/E3 1 |
| Verdicts | NO GO | NO GO |

* D2, E2 and E3 now fail **only the backlog gate**.
* F, G and H fail solvency.
* A, B, C, D and E also fail service.
* If backlog passed (a counterfactual): D2 would be CONDITIONAL GO, a
  transition restoring service; E2 and E3 CONDITIONAL GO service
  purchases.

**What the operating characteristics show** (`oc-v0.2-eval-3.json`):

* **The backlog rule is too noisy for a stable queue.**
  * A synthetic queue at utilization 0.85 is unstable on 64% of seeds
    with stationary demand and 88% seasonal (v0.2-eval-2: 61% and 93%).
  * The peak check alone fails half of the stable seeds; without it,
    13% and 61%.
  * Alternatives are tabulated, none adopted: the tolerances are an open
    owner decision.
* **Service under clustering.** At the planning rates the frozen design's
  40-seed false rejection is:
  * 0.022 independent;
  * 0.34 at week ICC 0.02;
  * 0.96 across the development range.

  Power at the gates is 1.00 in every model.
* **D2 against E3 without independence.** A week-block bootstrap
  (exploratory): D2 fails a seed 13.7% of the time, almost all urgent,
  so it passes the 40-seed rule in 33% of replicates; E3 and E2 pass
  every one.

**Tests.**

* `TestDatedFunding` (deferral cannot conceal insolvency):
  * a past-due payment with ample cash;
  * negative cash between positive samples;
  * observation short of the settlement window (INVALID), and the
    window binding at a day-175 horizon;
  * severance, contract minimums or fees in flight above end cash;
  * exact coverage passes, and the indicator alone does not gate.
* `TestSeasonalBacklog`:
  * seasonal-stable passes, and fails v0.2-eval-2;
  * deterioration (flat demand, and beyond the workload) fails;
  * a backlog that does not fall when demand falls fails;
  * persistent work, a growing and a proportional stale burden fail;
  * a spike and late-tail deterioration fail;
  * with no workload recorded, the comparison is unscaled.
* `TestClusterStatistics`: design effects, weekday adjustment, robust
  intervals, Clopper–Pearson bounds.
* `TestTransitionClassification`: transition with and without
  restoration, service purchase, no benefit, restoration under NO GO,
  no T180 promotion.
* `TestPolicyFrozen`: v1 and v2 kept and refused; every backlog check
  adjusted or absolute exactly once.
* The operating-characteristics checks.
* `internal/session` `TestDev12Record*`.

**Fail-before-fix.** 20 targeted mutations of the new rules, all caught.
Two survived first and their tests were strengthened:

* the settlement window, invisible at a day-150 horizon;
* scaling stale, invisible below the 2-study floor.

Registers 0.2.0-dev.12 (batch mode `v0.2_dev12`, fixture
`mini-seed37-dev12` from the dev.12 binary).

**Limitations.**

* The backlog tolerances, and the dependence trade-off against the
  planning target, are unresolved owner decisions. Strategy selection
  is NO GO.
* The operating characteristics' models are synthetic. Their dependence
  range comes from 10 development seeds, and the queue is a simple
  hourly model.
* The bootstrap resamples weeks independently and has only 10 seeds of
  between-seed variation.

---

## V0.2 Slice 5C.1 — development-seed strategy comparison (0.2.0-dev.12)

Strategy development on the development seeds 901–910 only, under policy
v0.2-eval-2 unchanged. The holdout is sealed and untouched. The
reference seeds 1001–1020 were not used. Simulation, economics, demand,
durations, thresholds, the policy, the holdout manifest and the holdout
runner are unchanged. Design §13.7; the full record is in
`docs/calibration/strategy-development/README.md`.

* **Harness.**
  * `eval.Development` loads a strategy set in the holdout contract's
    formats: strict decoding, every pin checked, the baseline and
    `teleOnlyOf` validated.
  * It accepts only the policy's development seeds, and its protocol is
    never authorized.
  * `TestStrategyDevelopment` (tag `calibration`) runs the canonical
    runner's loop with the development recorder: the common horizon
    computed first and confirmed, the complete matrix, and the frozen
    evaluation.
  * It writes `run.json`, `report.json` and `trajectory.csv` once; a
    rerun must reproduce them byte for byte.
  * `report.py`, `reconcile.py` and `breach_timing.py` generate every
    table.
* **Phase 1.** A–D reproduce every archived development-seed daily
  sample exactly: 10,800 samples across engines dev.4 and dev.5, in
  profit, cash, reports and backlog. Their T90 breach rates are the same
  under all three rules.
  * D's historical CONDITIONAL GO becomes NO GO under v0.2-eval-2, from
    the rules alone:
    * the joint per-seed rule: soon fails on 6 seeds, urgent on 3;
    * the block backlog rule;
    * the funding gate.
* **Family (fixed before any teleradiology run, from D's breach
  timing):**
  * E: D plus urgent-only teleradiology on weeknights and Saturdays;
  * F: D plus all-priority teleradiology on every window;
  * G: B plus F's contract.

  All are open-ended from the earliest date, Oct 27, with no
  cancellation during the evaluation.
* **Four refinements**, each with a hypothesis written before its run:
  * R1 E2: E at a 65/55% allocation. Kept: it passes the joint service
    rule on all 10 seeds.
  * R2 H: D plus all-priority on weeknights only. Rejected: insolvent
    on 8 seeds.
  * R3 E3: E2 on weeknights only. Kept: the same gates, $2.0k cheaper
    per 90 days.
  * R4 D2: E2 without the contract. Kept: it passes service alone; its
    urgent margin is thin (worst seed 11.6% against 12%).
* **Result: no strategy passes, and none can as the policy is frozen.**
  Every plan, no action included, fails two hard gates on every seed:
  * funding: the 14-day employee commitment, about $53k, exceeds the
    cash headroom from day 1;
  * backlog: the reference block, Dec 7 – Jan 3, falls in the December
    demand low.

  These are policy limitations for the owner. The policy was not
  changed.
* **What passes the other hard gates.** D2, E3 and E2 pass service,
  pooled rates, case selection and solvency on all 10 seeds. They lose
  $18.8k, $23.2k and $25.2k at T90 against no action: a service purchase
  at best.
* **Teleradiology's marginal value.**
  * Urgent-only: urgent about 10% → 2–3%, for $4.4–6.5k per 90 days.
    It adds no reports and no revenue.
  * All-priority: every gate passes, for $27–36k per 90 days. It adds
    at most 23 reports and $3.0k of revenue, and is insolvent on 7–10
    seeds.
* **Recommendation.** No primary strategy under v0.2-eval-2 as frozen.
  * If the funding and backlog rules are revised, the prespecified
    ranking picks D2.
  * E3 is the robust alternative: D2 fails the 40-seed joint rule about
    9% of the time under independence, E3 essentially never.
  * The strategy set is not frozen, and no unseal record exists.
* **Registers 0.2.0-dev.11.** Batch mode `v0.2_dev11`, with a genuine
  fixture from the dev.11 binary (`mini-seed35-dev11`): an urgent
  contract, a `minimum_term` refusal and a cancellation. Without the
  registration the dev.11 tests fail as an engine mismatch.

**Tests.**

* `TestDevelopmentRunnerMatchesCanonical`: on the synthetic protocol,
  the development and canonical runners give byte-identical records and
  the same horizon, including a batch at a sample instant.
* `TestDevelopmentSet`: development seeds only; refuses a reference
  seed, an unpinned strategy, an acting baseline, another schema and an
  invalid `teleOnlyOf`; never authorized.
* `TestDevelopmentSetsCommitted`: every committed set loads strictly,
  and every batch is in [takeover, T90).
* `TestWriteOnce`.
* `internal/session` `TestDev11Record*`.

**Mutations.** 11 targeted mutations, all caught:

* the seed range, pin, schema, baseline and `teleOnlyOf` checks;
* batch-before-sample order, commitment days, horizon and capture
  equivalence;
* `writeOnce`'s two guards.

The batch-order mutation first survived, because the fixture had no
batch at a sample instant; the test now has one.

**Limitations.**

* Ten seeds: a single seed decides the development verdict. The
  results identify mechanisms and credible strategies, not 40-seed pass
  rates.
* Soon breaches are clustered (dispersion 2–9), so the per-seed soon
  gate's operating characteristics are optimistic.
* Every reorganization has a negative T90 increment, so under the frozen
  verdict order the best attainable outcome is CONDITIONAL GO, service
  purchase.
* `eval.Development` calls `GuardRepo` as defence in depth. It is
  redundant with the development-seed check, so no mutation can show it
  necessary.

---

## V0.2 Slice 5C.0b — canonical holdout runner (0.2.0-dev.11)

This step closes the audit's remaining high finding, H6: holdout
protection was a cooperative convention, not an enforced one-time
protocol. The holdout stays sealed and untouched. No strategy was run,
and the policy (v0.2-eval-2) and simulation are unchanged. Design §13.6;
rules in `docs/evaluation/README.md` §9.

* **One path to a holdout seed.**
  * `eval.GuardRepo` now refuses holdout seeds whether or not the
    holdout is unsealed.
  * Its refusal no longer names them. v0.2-eval-1's printed the
    offending seeds, which would have leaked them into a log.
  * Every harness calls it: `parseSeeds`, the session replay check, and
    newly the dynamic-staffing runner per seed and the evaluation
    recorder itself.
  * The only recorder for a holdout seed is `World.NewHoldoutRecorder`.
    It needs an `eval.Protocol` from `eval.Authorize`, which cannot be
    constructed outside the package.
* **The artifact contract** (`internal/eval/protocol.go`):
  * strategy files (`dreamteam.strategy/v1`), each optionally declaring
    a teleradiology-only parent;
  * a strategy set (`dreamteam.strategyset/v1`) pinning each file by
    digest and naming the scenario manifest and the action-free
    baseline;
  * the unseal record (`dreamteam.holdout-unseal/v1`), pinning:
    * the holdout manifest's bytes, the policy and the strategy set;
    * the exact engine;
    * the one primary strategy;
    * the approver, date and statement.
* **Authorize** validates every field and digest, decoding strictly. It
  requires every file committed with no change (`GitRepo`, through
  `git`), and the unseal record committed by its approver.
* **The runner** (`internal/sim` `TestHoldoutRun`, run with
  `DT_HOLDOUT_RUN=1`):
  * it computes the common horizon from the strategies' accepted
    actions before running, and the records confirm it on every seed;
  * it runs every strategy, the baseline included, on all 40 seeds;
  * it refuses a batch at or after T90;
  * it evaluates with the complete seed set and the pinned digests
    required (`eval.Input` `Seeds`, `PlanSHA`), one digest per plan;
  * it names the primary, and every other candidate is reported
    exploratory;
  * it writes `run.json` (every record's digest, the horizon and the
    artifacts) and `report.json` once, read-only, into
    `docs/evaluation/holdout-runs/<holdout>/`.

  A rerun must reproduce both byte for byte, or it fails and writes
  nothing. Without an unseal record the test skips as sealed.
* **The holdout manifest's protocol text** now describes this; its
  method, label and seed digest are unchanged.
* **Version.** 0.2.0-dev.11 registers 0.2.0-dev.10 (batch mode
  `v0.2_dev10`). The fixture `testdata/sessions/mini-seed33-dev10` was
  written by the dev.10 binary from a clean 5f9cb94 worktree. It holds an
  accepted contract, an overlap refusal and an accepted cancellation, and
  replays exactly and continues. MIGRATIONS.json lists the bump; every
  reference is unchanged.
* **Tests.** The protocol runs end to end on a synthetic two-seed
  holdout with its own label and range: the mini practice, the short
  policy, a baseline and a teleradiology-only strategy.
  * Authorization refuses in 14 ways:
    * a missing record;
    * an edited strategy, strategy set, policy or manifest;
    * a primary that is the baseline or unknown;
    * a bad date;
    * no statement;
    * a wrong schema;
    * an unknown field;
    * another committer;
    * uncommitted files;
    * another engine.
  * The run is written and then reproduced. A tampered report fails
    with nothing written, and a directory with other files is not
    written over.
  * A T90 batch, an incomplete matrix, a foreign strategy digest and an
    unauthorized protocol are refused.
  * Holdout seeds are refused by both recorders.
  * The evaluator's matrix rules and roles are tested.
  * `GitRepo` is tested on a scratch repository: committed, changed,
    untracked, nonexistent, first author.
  * The repository's holdout authorizes as sealed.
  * Two dev.10 replay tests.
* **Fail-before-fix.** 29 targeted mutations, each removing one check,
  were all caught. Two first survived and exposed weak tests:
  * the forged-protocol test passed on a nil-pointer panic, not the
    guard; it now has a complete forged protocol and checks the refusal
    message;
  * no test gave a nonexistent path to `Committed`; one does now.

  The dev.10 replay tests fail without the registration.
* **Verification.**
  * The full uncached suite on the Pi passes in 11 min 10 s (sim 639 s,
    session 206 s, cmd 134 s, eval 64 s, tui 37 s). Goldens pass with no
    reference changed, and every session file replays as
    `earlier_build_exact`, the new dev.10 fixture included.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * On the Cortex-A76, with the guarded harness, plan A–D validation
    (56 s) and evaluation (195 s) match the committed archive in every
    field but the engine name. The 180-day session replay check passes
    (165 s).
* **Not done (by design).** No strategy set is frozen and no unseal
  record exists: both are 5C.1's and the owner's. The canonical run has
  never been executed on the holdout. Acceptance was not rerun: it
  exercises only the simulation engine, whose production code is
  unchanged.
* **Remaining before unsealing.** One independent review of the whole
  corrected framework, the frozen strategy set, and the owner's unseal
  record. Open policy decisions are in `docs/evaluation/README.md`.

---

## V0.2 Slice 5C.0a — evaluation policy correctness and anti-gaming (0.2.0-dev.10)

An independent read-only Astra High audit of v0.2-eval-1
(`docs/audits/slice-5c0-eval-policy-audit.md`) gave CONDITIONAL GO for
development and NO GO for unsealing. It found seven high, five medium
and some low findings, and no critical one. The owner decided the open
questions and made this slice a prerequisite for definitive strategy
ranking.

This step corrects the evaluator and freezes a new policy version,
**v0.2-eval-2** (`docs/evaluation/policy-v0.2-eval-2.json`, rationale in
`docs/evaluation/README.md`, design §13.5). No strategy was run, ranked
or tuned. No holdout seed was generated or used. The simulation's
clinical and financial mechanics are unchanged.

* **Versioning.**
  * v0.2-eval-1 (`policy-v0.2.json`, `README-v0.2-eval-1.md`) is kept
    byte for byte and named by digest in the new policy's `supersedes`.
  * `LoadPolicy` refuses it as a policy to evaluate under. No evaluation
    was ever run under it.
  * The new policy lists every changed rule with its finding, rationale,
    old and new value and expected impact on eligibility. It also lists
    the deferred design measures.
* **H5, joint reliability.** A seed fails if any of its five per-seed
  service gates fails or lacks evidence, and at most 10% of seeds may
  fail jointly. Pooled rates still pass per window. The fixture:
  * six seeds each fail a different gate;
  * every gate alone is within its limit, which v0.2-eval-1 passed;
  * the union of six seeds fails.
* **H1, urgent evidence window.** Urgent is decided per seed over the
  whole gated period (t0, tailEnd]. The 12% gate is unchanged, pooled
  urgent is still gated in T90 and the tail, and soon and routine are
  unchanged.
  * The window was chosen by synthetic operating characteristics
    (`oc-v0.2-eval-2.json`, regenerated by
    `DT_OC=1 go test -run OperatingCharacteristics ./internal/eval`), at
    the owner's 8% planning rate, with every per-seed gate and the joint
    rule.
  * False rejection on 40 seeds is 0.030, against 0.981 for v0.2-eval-1's
    windows (independent studies). On 10 development seeds it is 0.069.
  * The Monte Carlo with week-level clustering (ICC 0.02) gives 0.37,
    and a 1-point seed SD gives 0.11. So the ≤ 5% aim holds only near
    independence.
  * That trade-off is reported, not hidden by weakening the gate. A
    per-gate dispersion index measures it on real seeds.
* **H2, common horizon.** One horizon covers every plan's tail rule and
  its latest commitment (a contract's minimum term). A record whose
  horizon or observation end differs, or falls short, makes the
  evaluation INVALID. `TailEnd` and `ObservationEnd` let a harness
  compute both before running.
* **H3, access followed through.** Due cohorts are taken by the original
  wanted time, wherever an order was received, with outcomes at the end
  of observation. Receipt-based loss is also followed to the end.
  Inherited orders are a guarded stratum. The day-89 receipt with a
  day-100 wanted date, and a referral lost after its window, are both
  regressions.
* **H4, backlog.** Every rota-aligned block after the reference, through
  the one covering the horizon, is checked against it for mean and peak
  backlog, overdue, stale (24 h past due) and exposure. No study may be
  more than 14 days past due. The fixtures cover:
  * a persistent stream of old work;
  * old work replacing new;
  * a spike that recovers;
  * late-tail deterioration;
  * an inflated reference.

  Each passes the old rule and fails this one.
* **M1/M2, case selection.**
  * Every threshold is an exact basis-point comparison, so equality at
    2%, 5 and 2 points passes. A non-whole basis point is refused.
  * Strata: per seed by priority and modality; pooled by priority,
    modality, their intersection and inherited orders; acquisitions by
    priority and modality.
  * A rare stratum is insufficient, never a pass, and fails if dropped.
    It blocks GO for priority, modality and inherited strata.
* **H7, verdicts.** GO also needs positive absolute operating profit at
  T90 and T180. Each comparison carries a classification: financial
  superiority, relative improvement, service purchase (only when the
  baseline fails service) or no benefit.
* **M3, money.**
  * The obligation bridge adds 14 days of contracted employee pay; its
    components are disjoint.
  * A funding gap at any capture or daily sample is a hard gate.
  * The recorder follows cash posting by posting and records when the
    minimum fell. On real runs it equals the evaluation window's minimum.
* **M4/M5, record integrity.** These are now integrity failures:
  * a missing, duplicate or mistimed capture;
  * a missing or misdated sample;
  * disordered study timestamps;
  * an order outcome inconsistent with its resolution time;
  * short observation;
  * missing payback data.

  Displacements are counted by event time. Labels name seed fractions;
  worst seeds are paired and identified, including the worst profit
  seed; means are over defined values only.
* **Recorder** (test-only, read-only). Record v2 adds:
  * order resolution and displacement times;
  * daily obligations;
  * the employee commitment;
  * the latest commitment and common horizon.

  A real-run finding: a no-show can call for a new date, so it is the
  one outcome allowed before a later resolution.
* **Version.** 0.2.0-dev.10 registers 0.2.0-dev.9 (batch mode
  `v0.2_dev9`). The fixture `testdata/sessions/mini-seed31-dev9` was
  written by the dev.9 binary from a clean e2ce1fd worktree. It holds an
  accepted contract, a `notice_too_short` refusal and an accepted
  cancellation, and replays exactly and continues. MIGRATIONS.json lists
  the bump; every reference is unchanged.
* **Tests.** 36 evaluator tests. 16 are new adversarial ones:
  * disjoint failing seeds;
  * common horizons;
  * carried-forward and inherited orders;
  * backlog safeguards;
  * exact tolerances and equality end to end;
  * rare strata;
  * a loss-making improvement;
  * an insolvent baseline without restoration;
  * the funding gap through the horizon;
  * record integrity (17 corruptions);
  * displacement timing;
  * worst seeds;
  * the obligation bridge;
  * dispersion.

  Alongside them: the operating-characteristic check, the real-run
  integration test with new assertions, and two dev.9 replay tests.
* **Fail-before-fix.** 27 targeted mutations, each reverting one fix,
  were all caught. The first run had three survivors:
  * the per-seed horizon comparison was redundant with the
    evaluation-wide check, so it was removed;
  * a referral lost after its window lacked a test, now added;
  * the persistent rule lacked a case only it catches (a steady stream
    of studies 20 days late), now added.

  The dev.9 replay tests fail without the registration.
* **Verification.**
  * The full uncached suite on the Pi passes in 11 min 40 s (sim 672 s,
    session 221 s, cmd 132 s, eval 48 s, tui 41 s). Goldens pass with no
    reference changed, and every session file replays as
    `earlier_build_exact`, the new dev.9 fixture included.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * On the Cortex-A76, plan A–D validation (75 s) and evaluation (205 s)
    under dev.10 match the committed archive in every field but the
    engine name, with the harnesses' seed guard now on v0.2-eval-2. The
    180-day session replay check passes (167 s).
  * Evaluator output is byte-identical between runs and input orders
    (`TestEvaluationDeterministic`).
  * No holdout seed was run: the guard refuses them, and no unseal
    record exists.
* **Not done (by design).** No strategy comparison was run, and the
  holdout is sealed and untouched. Acceptance was not rerun: it
  exercises only the simulation engine, whose production code is
  unchanged.
* **Remaining audit findings.**
  * H6, the holdout runner and artifact binding, is Slice 5C.0b.
  * M5's deferred measures are listed, not implemented.
  * The urgent dependence trade-off and the [proposed] numbers are open
    for the owner.

---

## V0.2 Slice 5C.0 — frozen evaluation framework (0.2.0-dev.9)

The owner accepted Slice 5B. This step freezes how V0.2 strategies will be
judged, before any staffing, reading-allocation or teleradiology strategy
is compared. It is a preregistration and measurement milestone: no
strategy was run, selected or tuned. No demand, reimbursement, staffing or
allocation economics, teleradiology price, read duration or action
semantics changed, and no simulation production code changed. Rules,
rationale and open decisions: `docs/evaluation/README.md`; design summary:
`docs/V0.2_DESIGN.md` §13.4.

* **Policy.** `docs/evaluation/policy-v0.2.json` (`dreamteam.evalpolicy/v1`,
  decoded strictly) holds every threshold, window, minimum and verdict rule.
  Reports carry its SHA-256.
* **Evaluator.** `internal/eval` is a pure package with no simulation
  dependency.
  * `Measure` turns a per-seed record into cohorts, access, remote, money
    and backlog measures, and cross-checks its sources: samples against
    studies, remote fees against the ledger, the cash bridge.
  * `Evaluate` applies the policy. It gives per-seed gates, pooled results
    with Wilson intervals and the worst seed, paired t 95% comparisons
    against the baseline (or between candidates), accrual and cash
    payback, and the verdict.
* **Recorder.** `World.NewEvalRecorder` (test-only,
  `internal/sim/export_eval_test.go`) samples a run daily and captures it
  at each horizon. It reads state only: a recorded run's state digest
  equals an unrecorded one's.
* **Service.** The owner's gates are unchanged: urgent ≤ 12%, soon ≤ 32%,
  routine ≤ 6%.
  * They are measured per seed and per priority, in T90 and in the
    frozen-policy tail.
  * A study breaches if it is not final at its due time, observed
    through a 24-hour closure.
  * Empty or thin cohorts are insufficient, never a pass. The minimums are
    urgent 43, soon 88 and routine 23, from a 5-point standard error at
    the gate.
  * At most 10% of seeds may fail or lack evidence, and the pooled rate
    must pass.
* **Cohorts.** All studies acquired in each window, finished or not:
  * completion;
  * completed-only TAT (never gating);
  * unfinished ages;
  * overdue count and study-minutes;
  * breaches and their denominators;
  * the historical as-of-end rate beside the gated one.

  Inherited backlog and orders are a separate population.
* **Backlog.** The single-sample day-90/day-180 rule, which plan D failed
  on a Monday-to-Sunday comparison, is replaced. The new rule compares
  28-day blocks aligned to the rota (week-A Mondays), and growth may be at
  most max(10%, 2 studies). Stages, ages, overdue fraction, flow and slope
  are reported. The old rule is reported as legacy, and the historical
  reports are untouched.
* **Money.**
  * T90 is primary, T30 feedback, D31–90 operating and T180 validation.
  * Solvency is the posting-level minimum cash, which may not fall below
    zero on any seed through the tail.
  * An obligation bridge (payables, payroll, severance, earned unposted
    wages, remote in-flight fees, contract minimums) is listed beside the
    P&L, never in it. Unbilled work is not cash, and there is no
    borrowing.
* **Payback.** Sustained break-even, accrual and cash, right-censored. A
  median is given only with at least half the seeds repaid.
* **Remote reading.** Local and remote reports, claims, handoff wait,
  turnaround by reader, retainer and read-fee expense from the ledger,
  and cost per remote report.
  * Faster reporting, additional reports and additional recognized
    revenue are kept apart.
  * The worst-seed breach rate by priority is set against cash drain,
    the owner's question for teleradiology.
  * A teleradiology-only variant must book identically.
* **Access and case selection.** Due-demand served fraction by priority
  and modality, per seed and pooled, plus acquisitions, lost referrals
  and walk-in abandonment, all against the paired baseline. A plan cannot
  pass the breach gates by declining cases.
* **Seeds.** Development is 901–910. 1001–1020 are exposed and
  reference-only.
  * A new **locked holdout** `v0.2-holdout-1` has 40 seeds from a
    published SHA-256 method, with no overlap with earlier sets. Its
    digest is recorded, and the seeds are listed nowhere.
  * Every calibration harness refuses its seeds (`eval.GuardRepo`) until
    an approved unseal record names the policy and a frozen strategy set.
  * It is run once.
* **Verdicts.**
  * INVALID: an integrity check fails.
  * NO GO: a hard gate fails or lacks evidence.
  * GO: a significant T90 benefit, T180 validation and no funding gap.
  * CONDITIONAL GO: a positive mean without every GO condition, or
    service restored at a stated cost.

  There is no weighted score.
* **Version.** 0.2.0-dev.9 registers 0.2.0-dev.8 (batch mode
  `v0.2_dev8`, the current action set and rules). The fixture
  `testdata/sessions/mini-seed29-dev8` was made by the dev.8 binary. It
  holds an accepted contract, an overlap refusal and a cancellation, and
  replays exactly and continues. MIGRATIONS.json lists the bump; every
  reference is unchanged.
* **Tests.** 19 evaluator tests on synthetic fixtures cover:
  * empty strata;
  * insufficient and undecided cohorts;
  * unfinished overdue studies;
  * rota-aligned backlog;
  * midnight and window boundaries;
  * inherited backlog;
  * remote and local reconciliation;
  * vendor cost;
  * transaction-level minimum cash;
  * payback censoring and the median rule;
  * case selection;
  * historical file hashes;
  * deterministic output;
  * the verdict rules;
  * booking invariance;
  * the frozen policy and holdout;
  * the repository guard;
  * the statistics.

  Alongside them: an integration test on real mini runs with and without
  a contract, and two dev.8 replay tests.
* **Fail-before-fix.** 21 targeted mutations were run:
  * 20 are caught by the intended tests;
  * one, the dev.8 replay calling `Session.Apply` instead of
    `World.Apply`, is equivalent (`Session.Apply` is `World.Apply`). The
    redundant case was removed.
* **Verification.**
  * Full uncached suite on the Pi passes in 11 min 31 s (sim 668 s,
    session 169 s, cmd 95 s, tui 29 s, eval 13 s). Goldens pass with no
    reference changed, and every session file replays as
    `earlier_build_exact`, the new dev.8 fixture included.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * On the Cortex-A76, plan A–D validation (55 s) and evaluation (197 s)
    under dev.9 match the committed archive in every field but the
    engine name. The 180-day session replay check passes (163 s).
* **Not done (by design).** No strategy comparison was run, no strategy
  was chosen, and the holdout is sealed. Acceptance (`-tags acceptance`)
  was not rerun: it exercises only the simulation engine, whose
  production code is unchanged.
* **Open decisions** (owner, before unsealing): breach decided at due
  time; the evidence minimums (the urgent tail cohort is close to its
  minimum on low-volume seeds); the 10% seed limit; the case-selection
  tolerances; the backlog rule; zero-tolerance solvency and funding gap;
  the GO conditions; the holdout's size and protocol; and the §13.3
  `DELTA_*` tolerances, which stay informational.

---

## V0.2 Slice 5B — teleradiology contract action and economics (0.2.0-dev.8)

The owner accepted Slice 5A. This step exposes the remote reading engine
through the player action `contract_teleradiology`, with §8.2's catalog
and §8.4's economics: retainers, read fees locked at claim, payables and
settlement, previews, observation, the TUI and replay. It invents no
price, plan or lane, and changes no demand, reimbursement, wage,
allocation, read duration, service threshold or frozen benchmark plan.
5C (evaluation and any strategy) has not started. Design and as-built
notes: `docs/V0.2_DESIGN.md` §5.13.

* **Action.**
  * `op start`: `requestId`, `plan` (`standard-v1`), `windows`, `scope`
    and `effective`. Activation is at least 14 days ahead and inside the
    booking horizon. The contract is open-ended, `T00001` onward.
  * `op cancel`: `contract` and an exclusive `endDate` at least 30 days
    ahead and at least 30 days after activation. Once fixed, the end
    cannot move.
  * Repeats are `unchanged`. A reused requestId with other terms is
    `request_conflict`, and a moved end `already_cancelled`.
  * Overlaps are judged on the timeline the whole batch leaves, so a
    cancellation and its successor can share a batch.
  * The batch is atomic like the staffing actions: a refusal leaves no
    contract, posting, event or log line.
* **Money**, in integer cents, with two appended expense accounts:
  * **Retainers** accrue per selected window per active day, at 00:00 after
    the day's payments: `floor(M×d/n) − floor(M×(d−1)/n)`. The minimum
    term is owed with no reads.
  * **Read fees** are locked at claim (night +20% at ≥ 22:00 or < 08:00),
    committed while in flight and recognized once at the report's final.
  * **Settlement:** both are paid by the existing first-of-month A/P
    payment, which covers everything accrued before it; an expiry erases
    nothing.
  * **Revenue** stays at report final.
* **Observation** (`Observation.teleradiology`, `Session.Teleradiology`):
  * contracts and their status, dates and terms;
  * coverage now, eligible waiting studies, and both lanes;
  * remote and local reports with their mean turnaround;
  * vendor money: recognized, paid, unpaid with its payment date, and
    in-flight fees.
  * Guaranteed amounts are listed apart from forecasts, and no future
    draw is read.
* **Preview:** `PreviewStaffing` covers the action, showing:
  * the guaranteed retainer through the earliest end, and the
    open-ended run rate;
  * a labelled upper bound from trailing observed demand: the studies
    that waited inside the windows in the last 28 days.
* **TUI:** the staff screen (`m`) has a Teleradiology section: plan and
  rules, contracts, coverage, lanes and the vendor balance. It has rows
  for a new contract (windows, scope, activation) and one cancellation
  row per open-ended contract, each previewed and submitted with enter.
* **Replay and compatibility.**
  * The new state is encoded only when a contract exists. The ledger's
    account arrays encode the two new accounts only when non-zero, so
    with no contract every state, log and ledger digest is as under dev.7
    and every golden reference reproduces.
  * 0.2.0-dev.7 is registered as an earlier build (batch mode
    `v0.2_dev7`). dev.5–dev.7 records replay with Slice 4's action set,
    in which the type was unknown.
  * A genuine dev.7 record (`testdata/sessions/mini-seed27-dev7`) holds
    an `unknown_action` and a `mixed_batch` refusal of the type. It
    replays exactly, fails with this build's action set, and continues.
* **Tests.** 28 targeted tests (27 in `internal/sim/telecontract_test.go`
  and a save/load test in the session package), three dev.7 record tests
  and a TUI test. Money is checked to the cent, including same-instant
  midnight and expiry cases and repeated submissions.
* **Fail-before-fix.**
  * Run against the dev.7 engine with only the new API types, 26 of the
    27 sim tests fail. The 27th, no-contract neutrality, passes before by
    design and is covered by mutation.
  * All 25 targeted mutations (M1–M25) are caught. They cover the rules
    on notice, term, overlap and atomicity; proration, expiry and night
    pricing; fee timing and settlement order; lanes and scope; previews;
    and the digest, mixed-batch and dev.7 registration rules. M19
    first failed to build; rewritten, it is caught by two tests.
* **Functional validation** (development seeds 901–903 only; not a
  strategy search; `docs/calibration/teleradiology-5b`).
  * Steps: a contract on all windows, then cancelled at its earliest end,
    then 90 days.
  * Every seed activates, claims backlog, reports every claim at
    catalog prices, and accrues retainers exactly ($6,100.01).
  * Vendor bills settle completely, and nothing is claimed after the end.
  * Ownership, ledger, payables and cash invariants hold daily. The cash
    bridge reconciles, and bookings are identical to the no-contract run.
  * Total reports equal the no-contract run's: here remote reads
    substituted for later local reads, at $27.7–29.4k of operating profit over 90 days (the vendor cost).
    That is for 5C to weigh, not a verdict.
* **Verification.**
  * Full uncached suite on the Pi: 9 min 58 s (sim 560 s, session 133 s,
    cmd 98 s, tui 28 s). All goldens pass, and all 11 session files
    replay as `earlier_build_exact`.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * Acceptance on the Cortex-A76: 2 h 35 min while shared, 396 subtests
    passed and none failed.
  * Without a contract the benchmark is unchanged. The plan D results
    under dev.8 (development, 66 s; held out, 220 s) match the committed
    archive in every field but the engine name. The 180-day session
    replay check passes (179 s).
  * Engine benchmarks, dev.7 against dev.8, show no measurable
    difference. On the Pi, Mini180 medians are 2.89 vs 2.94 s and
    EstablishedPlay90 1.34 vs 1.34 s. On the Cortex-A76 they are 946 vs
    970 ms and 387 vs 390 ms, inside a 3–12% run-to-run spread.
    Allocations are identical, and EstablishedPlay90 uses 0.4 MB/op less.
* **Limitations.**
  * There is one catalog plan, and a contract can be ended but not
    edited.
  * A batch cannot cancel the contract it starts.
  * The trailing-demand preview is an upper bound, not a lane model.
  * The evaluation window does not yet split reports, turnaround or cost
    by reader kind, or attribute waits to absent coverage. That is 5C's,
    frozen before any strategy.
  * The shift-end lapse of local reading debt remains.

---

## V0.2 Slice 5A — remote reading engine (0.2.0-dev.7)

The owner accepted dev.6 and authorized Slice 5 in three stages with a
checkpoint after each: 5A, the remote reading engine; 5B, contract
economics and the `contract_teleradiology` action; 5C, the evaluation,
with service and solvency rules frozen before any strategy and a new
locked holdout. This step is 5A only. No player action exists, and no
price, retainer, fee or account is added. Demand, reimbursement, wages,
allocations, read durations and service thresholds are unchanged. Local
read debt still lapses at shift end, now documented as a deliberate
limitation. Design and as-built notes: `docs/V0.2_DESIGN.md` §5.12.

* **Ownership.** A study's reader is a tagged, unexported value: none,
  an employed radiologist, or a remote read claim.
  * It encodes as one integer, exactly as before for on-site reads.
  * It is read only through `employee()` and `remote()`, so stale
    employee-index lookups no longer compile.
  * One owner from claim to final, through a shared finalization
    (`finalize`): one report, one revenue posting and one payer claim,
    whoever reads.
  * Employee debt and status stay on the employee; lanes have neither.
* **Two persistent lanes** (`internal/sim/remote.go`) serve every
  contract together.
  * Day start, closes and contract expiry never reset them. A lane is
    released only by its read's end.
  * A claim holds its lane for the ten-minute handoff and the study's own
    keyed read duration. Both ends are generation-checked completions.
  * The claim records its contract, lane, claim time, read-start time and
    night flag, for 5B's settlement.
* **Coverage** is a pure function of time.
  * It is the union of half-open windows (weeknights, saturday, sunday),
    intersected with the contract's dates. Exact starts are in and exact
    ends out.
  * Friday's night runs to Saturday 08:00. Saturday 08:00–14:00 and
    Monday 00:00–08:00 are gaps. Saturday and Sunday join at midnight.
  * Contracts can't overlap, so touching windows add no capacity.
  * Edge wakes fire only where coverage changes. They claim nothing;
    the dispatch after them may.
  * Remote coverage never touches booking.
* **Claiming** is dispatch's last pass, after the unchanged on-site pass
  and employed reading.
  * Idle lanes, in lane order, take the first eligible unread study in
    worklist order.
  * Urgent-only scope checks the original priority, not the overdue
    promotion.
  * There is no shift-end rule for lanes.
  * Reads finish after window end, contract expiry and midnight. The
    report belongs to its finalization date.
  * A successor contract uses only idle lanes.
* **Contracts before 5B.** Contracts have coverage terms and structural
  checks only: windows, a future activation, no overlap and ordered ends.
  Tests reach them only through `export_test.go`.
* **Compatibility.**
  * 0.2.0-dev.6 is registered as batch mode `v0.2_dev6`: this build's
    action set and rules, every digest verified. Earlier modes keep
    dev.5's rules.
  * Genuine dev.6 record `testdata/sessions/mini-seed26-dev6`, from a
    clean 3c71669 build: 100%/95%, a Sunday reduction refused as
    `commitment_too_short`, and a later change.
  * Before registration, the record was refused as an engine mismatch.
    Registered under dev.5's rules, it failed at batch 2.
  * It now replays whole and chunked, and continues as a verbatim prefix.
  * The remote state is encoded only when a contract exists, so every
    golden reproduces in every field.
* **Tests.** 14 in `internal/sim/remote_test.go`, 3 in
  `internal/session/dev6_test.go`. Fixtures are
  `internal/sim/testdata/micro-tele{,-local}.json`. They cover:
  * both lanes busy, a third study waiting, and the handoff holding a
    lane through a mid-handoff dispatch;
  * expiry during handoff and during a read, midnight, past a window's
    end, and the exact end with free lanes;
  * a successor sharing the lanes; adjacent windows and exact edges;
  * urgent-only by original priority;
  * local/remote exclusivity, with the employed reader first at a shared
    instant;
  * one report, revenue entry and payer claim per study;
  * save/reload as replay at a mid-handoff and a mid-read instant, across
    chunkings and event-budget stops;
  * booking invariance; §5.11's local debt conservation with lanes
    reading; edge dispatch neutrality.
* **Fail-before-fix (by mutation).** The 14 guards, from remote-first
  dispatch to an inclusive contract end, were each broken in turn with
  the tests unchanged, and each broken build failed its test (list in
  §5.12).
* **Verification.**
  * The full uncached suite passes on the Pi in 10 min 11 s (sim 589 s,
    session 138 s, cmd 113 s, tui 36 s), with `-timeout 60m`. All ten
    session fixtures, the new dev.6 one included, verify
    `earlier_build_exact`.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * Acceptance (`-tags acceptance`) passes on the Cortex-A76 in 2 h 11 min
    while shared (369 subtests).
  * The frozen plans A–D benchmark (`reading-dev`, `reading-eval`)
    reproduces the committed results in every field but the engine name,
    and the 180-day session replay check passes: no remote capacity
    without a contract.
  * Engine benchmarks, dev.6 against dev.7 interleaved: on the
    Cortex-A76, Mini180 has a median of 947 vs 959 ms/op and
    EstablishedPlay90 392 vs 388 ms/op, within the ≈7% run-to-run spread.
    The Pi results are noisier, with no measurable difference. Bytes and
    allocations per op are identical.
* **Limitations.**
  * No player can create a contract, and no session file can hold one,
    until 5B.
  * Observation names a remote reader only by lane. Remote share,
    coverage waits and turnaround by reader kind are 5B/5C reporting.
  * Worklist alerts describe employed readers only.
* **Next: Slice 5B**, contract economics and `contract_teleradiology`.
  Mode `v0.2_dev6` must then refuse the new type, as dev.6 did. Freeze
  the 5C evaluation rules before choosing any strategy. The owner plans
  an Astra review after 5B.

---

## V0.2 dev.6 — pre-Slice 5 remediation (0.2.0-dev.6)

An independent read-only audit of 13987e6 (0.2.0-dev.5) returned
CONDITIONAL GO, with two verified scheduler defects and an allocation
arbitrage that would distort Slice 5's comparison of internal reading
against outsourced reading. This step fixes those three and nothing else:
`contract_teleradiology` is not built, and demand, reimbursement, wages,
recruiting, read durations, service gates and the frozen plans are
unchanged. Design and as-built notes: `docs/V0.2_DESIGN.md` §5.11.

* **Fix 1: touching reading blocks.** At 100% the cycle's blocks touch.
  The next block's start (class 1) ran before the previous block's end
  (class 5) at the same instant, so the end closed reading and the
  radiologist claimed nothing for the rest of the shift.
  * `blockEnd` now continues reading when the day's next block starts
    exactly where this one ends.
  * The blocks, their sum and their boundaries are unchanged, so no
    capacity is added.
  * Reads stay nonpreemptive. Worklist order and the shift-end claim rule
    are unchanged.
* **Fix 2: overrun debt from time actually spent outside blocks.** dev.5
  charged the overrun only if a read ended outside a block, so a read
  running through a whole other-duty gap owed nothing. This affected
  allocations above 75%; the scenario and plan D allocations were not
  affected.
  * When a block opens while its radiologist is still reading, the gap
    since the last block ended is owed at once.
  * Repayment starts when the read ends, out of block time. What the
    block cannot cover carries forward as before, and debt lapses at the
    shift's end as since V0.1.
* **Fix 3: minimum allocation commitment (the Sunday arbitrage).** A
  version at 30% from a Sunday, restored on Monday, kept every shift's
  reading and cut the Sunday salary by $328.77 for Dr. Blake (about $904
  a Sunday for both radiologists at plan D's shares).
  * An allocation now stays in force at least 14 calendar days
    (`MinCommitDays`) before the next change for that radiologist.
  * The check runs on the timeline the whole batch leaves, after every
    item's own checks, so item order doesn't matter.
  * The refusal is `commitment_too_short` on the first item that makes
    the short pair; the rest of the batch is `batch_rejected` and nothing
    changes. Previews agree.
  * Replacing a pending version's value passes. Under this rule a
    withdrawal can only lengthen a pair.
  * Pairs already on an earlier build's timeline are grandfathered, but a
    withdrawal that would create a new short pair is refused.
  * Notice, effective dates, horizon, bounds, history, no-op rules and
    atomic mixed batches are unchanged. No new charge or account.
* **Tests (fail-before-fix).** All seven new tests fail with the fixes
  disabled and the tests unchanged.
  * `TestTouchingBlocksReadContinuously`: 4 and 5 claims instead of 18,
    for 30- and 20-minute reads.
  * `TestOverrunThroughOtherDutiesIsOwed` (95%, a read claimed at cycle
    minute 94 crossing 95–100, reads ending exactly on a block start or
    end): failed at the dev.5 claim minutes.
  * `TestOverrunThroughSeveralGapsIsRepaidLater` (30%, a 205-minute read
    crossing two gaps): next claim at 11:25 instead of 18:25.
  * `TestReadingTimeConservation`: 12, 12, 17 and 428 claims while
    owing.
  * The three tests in `allocation_commit_test.go`: the arbitrage, 13-
    against 14-day spacing and the earlier-timeline rules were all
    accepted.

  The micro fixture is `internal/sim/testdata/micro-blocks.json`: a
  saturated worklist across seven boundaries.
* **Conservation.** `TestReadingTimeConservation` keeps an independent
  account of every radiologist-day, built from the studies' claim and
  final times and the day's plan. It requires:
  * every minute outside a block borrowed once;
  * repayment, in order, only from later block time spent not reading;
  * no claim while time is owed or outside a block;
  * what is still owed at the shift's end lapses.

  Six weeks of the established practice at 100%/95%: 303,438 s borrowed
  = 297,557 repaid + 5,881 lapsed, with no claim while owing. Under
  dev.5's rules there were 428 claims while 2,790 minutes were owed.
* **Compatibility.** The fixes change outcomes, so this is not a digest
  migration.
  * Every record from an earlier build replays under that build's rules
    (`SetDev5Rules`, set by `session.Replay` until the record's end).
  * 0.2.0-dev.5 is registered as batch mode `v0.2_slice4`, with every
    digest verified.
  * There are two genuine dev.5 records from a clean 13987e6 build
    (`testdata/sessions/mini-seed24-dev5`, `mini-seed25-dev5-reading`).
    Unregistered, both were refused. Registered without dev.5's rules,
    the first fails at batch 2, and the second's log and run digests
    differ.
  * Both replay whole and chunked, and continue as a verbatim prefix
    under this build's rules.
  * A record claiming dev.5 but needing dev.6 rules is refused.
  * Unknown actions, v1 and v2 records, and dev.2–dev.4 records behave
    as before.
  * One existing session test moved a change from Nov 9 to Nov 10, from
    13 days apart to 14.
* **Neutrality.**
  * All seven golden references reproduce. MIGRATIONS.json lists the
    Slice 4 reference, now historical, with no changed field.
  * Plans A–D rerun under dev.6 reproduce `results/reading-dev` and
    `results/reading-eval` in every field but the engine name.
  * All nine session files (the benchmark's dev.4 and dev.5 files and
    the testdata records) verify as `earlier_build_exact`.
  * No plan was retuned and no seed was used to choose anything.
* **Verification.**
  * Full uncached suite on the Pi passes: sim 544 s, session 111 s, cmd
    87 s, tui 24 s.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * Plan session replay, A–D × seeds 901 and 1001 × 180 days, whole and
    in 13-hour chunks: 171 s on joofi.
  * Acceptance on joofi passes: staffing and roster invariants (50 seeds
    × 90 days each, after every transaction), turnover, churn, V0.1
    invariants and 1,000 chunkings. The first run hit a 120-minute
    timeout I had set (Slice 3 needed 128 min), during the V0.1 criteria;
    those were then rerun alone and passed in 42 min.
  * Performance (dev.5 against dev.6, interleaved, joofi): Mini180 917
    against 937 ms/op and EstablishedPlay90 376 against 386 ms/op
    (medians), inside a ≈15% spread. Allocations are identical. Details
    are in MEASUREMENTS.
* **Limitations.**
  * Debt still lapses at shift end, as in V0.1, now measured: 98 minutes
    in six weeks at 100%/95%.
  * The commitment is per radiologist only and does not price early
    exit.
  * The service-gate, backlog-rule and turnaround audit findings (Medium
    and below) are open, for Slice 5's pre-registration.
* **Next: Slice 5, `contract_teleradiology`.** This needs persistent
  remote reading ownership, service and solvency rules set before any
  strategy, liquidity as an independent constraint, and a fresh locked
  holdout. Seeds 1001–1020 stay as reference data.

---

## V0.2 Slice 4 — reading allocation (0.2.0-dev.5)

The owner approved `set_reading_allocation` after the dynamic staffing
benchmark, and nothing else (`contract_teleradiology` comes later).
Design and as-built notes: `docs/V0.2_DESIGN.md` §5.10. Benchmark:
`docs/calibration/dynamic-staffing/` README, "Plan D". No reimbursement,
demand, wage, reading-duration, equipment or opening-hour input changed.

* **Action.** `set_reading_allocation` changes an employed salaried
  radiologist's site share of paid time.
  * **Ops:** `op: set` / `remove`, `employee`, `effective`,
    `siteAllocationBP` 3,000–10,000.
  * **Dates:** any weekday at least 14 days ahead, within the booking
    horizon. Removal needs the same notice.
  * **Versions:** dated, like roster versions. A pending version can be
    replaced or withdrawn; versions in force or past never change; an
    append-only log keeps every operation. A set equal to the allocation
    in force, or the removal of an absent version, is `unchanged`; two
    writes to one radiologist and date in a batch are
    `invalid_parameters`.
  * **Distinct from contracts and rosters:** radiologists have no
    contract hours, and their shifts stay immutable.
* **Reading and supervision.**
  * The allocation in force each day sets that day's site reading blocks
    through the existing 100-minute cycle inside the same shifts.
    Worklist order, overrun debt, one read at a time, the p100-by-shift-
    end claim rule and origin-keyed read times are untouched.
  * Contrast supervision is presence on site and does not move (tested).
  * Booking coverage segments were also cut at reading-block boundaries,
    which allow shared-capacity handovers. They now keep the scenario
    allocation's boundaries (`DayPlan.cutReading`), so an allocation
    never changes technologist booking capacity and displaces nothing.
    The test that fails without this covers days built after the change
    and days a later hire rebuilds.
* **Money.** The existing daily salary line uses the allocation in force
  that day: daily rounding, benefits, and payday settlement through
  accrued payroll. One accrual per radiologist per day, by its key, so no
  time is charged twice. No new account or entry kind.
  * Tested to the cent across two changes, with the cash bridge
    reconciled.
  * Previews show the salary run-rate change per 30 days as a guaranteed
    obligation, with the date until which it can be withdrawn at no
    cost.
* **Batches and compatibility.**
  * The new type mixes atomically with `hire`, `terminate` and
    `set_roster`; any invalid item refuses the whole batch.
  * A `mixed_batch` message names `set_reading_allocation` only when the
    batch holds one, so every recorded message and golden is unchanged.
  * Allocation versions are encoded in the state digest only when
    present, so no-action state digests are unchanged too.
  * 0.2.0-dev.3 and dev.4 records replay with Slice 3's action set (batch
    mode `v0.2_slice3`). A genuine dev.4 record
    (`testdata/sessions/mini-seed23-dev4.*`, from a clean af475fd
    worktree) has the new type refused as `unknown_action` and
    `mixed_batch`. It failed before registration, and failed again under
    this build's action set; it now replays with every digest verified
    and continues as a prefix. The benchmark's dev.4 session files verify
    as `earlier_build_exact`.
* **Interface.**
  * **API:** `Employee.SiteBP` is the allocation in effect;
    `Employee.AllocationChanges` lists pending ones.
    `World/Session.ReadingAllocations()` gives each radiologist's
    timeline with site reading minutes per rota week and salary per 30
    days. `StaffingPreview.Allocations` gives before and after, the
    withdrawal date and the reporting backlog.
  * **CLI:** action files take the new type; golden
    `established-seed1-60d-allocation.json`.
  * **Session v2.**
  * **TUI:** staff screen rows per radiologist (←→ share in 2.5-point
    steps, `,` `.` date, `x` withdraw), the backlog line and a read-only
    preview.
* **Tests (fail-before-fix).** Nine `internal/sim/allocation_test.go`
  tests failed before the action existed. They cover:
  * date boundaries; bounds and shape; versions;
  * salary proration; supervision and booking invariance;
  * reading rules with urgent order; mixed batches; chunking; previews.

  Session tests cover save and replay across effective dates (whole,
  chunked, event-budgeted), a v1 record with the type name and the dev.4
  record. There are also a TUI test and the new golden.
* **Plan D benchmark (20 held-out seeds 1001–1020, 180 days).** D is plan
  C's staffing plus Dr. Avery 62.5% and Dr. Blake 52.5% from Tue Oct 27,
  the earliest legal date. D and its decision rule were fixed before any
  run.
  * **Reproduction:** A, B and C reproduce the original results exactly.
  * **D against C:** +$13.7k radiologist salary, +74 ± 20 reports, $12.5k
    of backlog revenue recovered. Profit −1.5 ± 3.2 $k at T180 and +0.6 ±
    1.2 $k per 30 days in steady state. Imaging and access are identical.
  * **Breach rates days 91–180 (urgent / soon / routine):** D 9.2 / 31.9
    / 4.4%, C 52.7 / 88.2 / 55.3%, no action 10.7 / 38.1 / 7.2%. The
    gates are met on the mean, but on all three gates only on 7/20 (T90)
    and 11/20 (days 91–180) seeds.
  * **Against no action:** +12.6 ± 1.4 $k at T180, break-even day 141
    (113–164) on 20/20 seeds, cash payback by T180 on 1/20.
  * **Decision: CONDITIONAL GO**, by the pre-registered rule. Its one
    failing test compares single backlog samples on day 90 and day 180;
    no action fails it too, and D's 30-day mean backlog is flat.
    Conditions:
    * the gates are met on average only;
    * the soon rate is at its gate;
    * the allocation buys service at about its own cost.

    No other allocation was tried on the held-out seeds.
* **Verification.**
  * The full uncached suite passed on the Pi in 9 min 1 s (sim 537 s,
    session 94 s, cmd 84 s, tui 27 s). An earlier run hit Go's default
    10-minute timeout in `sim` while the Pi was also compiling; rerun with
    `-timeout 60m` it passed.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * All seven golden references reproduce: the six historical ones under
    `MIGRATIONS.json` as before, the new one byte for byte.
  * `dreamteam verify` replays the three dev.4 benchmark sessions
    (`earlier_build_exact`) and D's.
  * Every plan replays whole and in 13-hour chunks (seeds 901 and 1001,
    180 days).
* **Limitations.**
  * One allocation was tested.
  * Gate pass rates per seed are modest.
  * The rule's backlog test uses point samples.
  * No recruiting uncertainty; ×1.30 demand only; 180 days; arm64 only.
  * Teleradiology, the alternative for marginal and evening reads, is not
    built yet.
* **Next: Slice 5, `contract_teleradiology`.** Then test substitution
  against allocation, on development seeds first.

---

## V0.2 dynamic staffing benchmark (0.2.0-dev.4)

The owner's milestone after Slice 3: can the sustainable scenario's shared
start be improved with the implemented actions alone, and at what
transition cost and payback? Benchmark: `docs/calibration/dynamic-staffing/`
(README, plans, per-seed results, reports, trajectories, sessions). Design:
`docs/V0.2_DESIGN.md` §5.9. **No engine behaviour, economics, demand,
wages, hours or patient behaviour changed, and no action was added.**

* **Plans.** `plans.py` writes them as fixed timed action batches, applied
  through `World.Apply` identically on every seed. Each submits one atomic
  batch at the takeover with standard offers; V0.1 controls stay at their
  defaults.
  * **A, no action.**
  * **B, partial:** convert the MR evening. `hire` a 20 h 16:00–20:00 MR
    technologist and `terminate` Alex (40 h) once it is productive.
  * **C, full:** the closest legal path to Step 0C's `DED-MRpt`. It hires a
    CT day core (40 h), an XR/CT evening (27.5 h) and an MR evening (20 h);
    terminates Sam and Alex; and moves Riley to X-ray only with `set_roster`
    from Monday Nov 23.
  * **Feasibility:** every step is legal (lead, training, notice,
    headcount 8 of 8, Monday effective date, no coverage gap). Only Step
    0C's reading allocation awaits Slice 4.
* **No overfitting.**
  * B was chosen among three candidates (MR conversion, Step 0C's XR/CT
    float, both) on development seeds 901–910. The rule was written into
    `plans.py` before any run: the best T180 increment with the urgent
    breach rate at most 5 points worse. The float candidates failed that
    rule.
  * The evaluation used 20 held-out seeds, 1001–1020, never used before
    (Steps 0–0C used 1–20), for 180 days.
* **Validation, every seed:** all actions were accepted; every lifecycle
  date matched the plan; at most 8 technologists; no inconsistent booking
  at any endpoint; the cash bridge reconciled; labor postings summed to
  the window's labor to the cent.
  * C displaced one CT contrast appointment on 2 of 20 seeds, rebooked the
    same day: a break-relief cover was needed in X-ray.
  * Each plan replays authoritatively from a saved session, whole and in
    chunks, to the same digests.
* **Results (20 held-out seeds, Δ against the paired no-action run).**
  * **B:** costs $9.8k (recruiting 3.0, severance 4.5, training 2.2). Δ
    profit at T30 / T90 / T180 is −8.0 / −2.0 / **+12.9 ± 0.1** $k, and
    +$5.0k a month in steady state. Break-even is **day 101** (100–106) on
    20/20 seeds; cash pays back on 20/20 (median day 114). Service is
    unchanged.
  * **C:** costs $23.7k (recruiting 9.0, severance 8.4, training 6.2). Δ
    profit at T30 / T90 / T180 is −19.0 / −10.4 / **+14.1 ± 2.7** $k, and
    +$8.2k a month. Break-even is **day 134** (113–175) on 20/20 seeds;
    cash pays back on 11/20 by T180. Lowest cash is $20.7k.
  * **C, days 91–180:** imaging +265 exams, lost referrals −179, walk-ins
    leaving −90. But reading cannot keep up:
    * urgent, soon and routine breach rates are 52.7%, 88.2% and 55.3%,
      against 10.7%, 38.1% and 7.2% with no action;
    * the backlog is 93 against 30 and still growing.
  * **Guaranteed pay** is negligible in both plans. **Unrecovered demand:**
    US "not rostered" losses (≈405 per 90 days) are untouched by every
    plan.
* **Decision: CONDITIONAL GO.**
  * Staffing actions give measurable agency with credible payback;
    transitions are complete by day 42 and do not call for a different
    starting state.
  * The full reorganization's service needs `set_reading_allocation`.
  * Next: Slice 4, then rerun the same frozen plans on the same held-out
    seeds with C plus an explicitly defined reading adjustment.
* **Code (test and session compatibility only).**
  * `internal/sim/calib_dynamic_test.go` is the harness (build tag
    `calibration`). It reads `benchmark.json` and `plans/`, samples daily,
    measures endpoints and spans, decomposes labor, and checks
    lifecycles. `calib_export_test.go` is a read-only ledger export for
    tests.
  * `internal/session/calib_dynamic_test.go` saves and replays every plan.
  * `report.py` builds the paired tables, payback and `trajectory.csv`.
* **Version and compatibility.** 0.2.0-dev.4 registers 0.2.0-dev.3 as an
  earlier V0.2 build: batch mode `v0.2_slice3`, the same action set.
  * A genuine dev.3 session file, `testdata/sessions/mini-seed22-slice3.*`,
    built from a clean 5fdfb95 worktree, replays with every digest and
    continues as a prefix (`TestSlice3RecordReplaysAndContinues`). It
    failed as an engine mismatch before the registration (fail-before-fix).
  * The Slice 3 golden is listed in `MIGRATIONS.json` with no changed
    field.
* **Verification.**
  * The full uncached suite passed on the Pi in 9 min 6 s; `go vet` is
    clean for the default, `acceptance` and `calibration` tags; gofmt is
    clean.
  * Benchmark results from the dev.3 and dev.4 binaries, on the Pi and on
    the Cortex-A76, are identical in every field but the engine name.
* **Limitations.**
  * One hand-authored plan per strategy, not optimized; premium offers and
    US coverage untested.
  * No recruiting uncertainty in the model.
  * Demand fixed at ×1.30 (×1.15 not rerun).
  * 180-day horizon.
  * Accrual expenses fees and severance at acceptance, and revenue waits
    for the report.
  * The provisional service gates are missed by every staffing-only plan,
    including no action on soon and routine.
  * amd64 not run.

---

## V0.2 Slice 3 — roster changes (0.2.0-dev.3)

The owner approved `set_roster` (2026-10-15). `set_reading_allocation` and
`contract_teleradiology` are not in this slice. Design: `docs/V0.2_DESIGN.md`
§5.8, with the deviations identified there. Without a `set_roster` action
every earlier outcome is unchanged.

* **Roster versions.**
  * An hourly technologist's or front desk employee's roster is now a
    timeline: the original (scenario or hire, immutable) and dated versions.
    Each version is a full A/B pattern from a Monday, in force until the
    next.
  * `op: set` inserts a version or replaces a pending one; `op: remove`
    withdraws a pending one, and its predecessor resumes. Several
    successive versions are allowed.
  * Versions in effect or past never change. Every operation, including
    replaced, withdrawn and voided versions, stays in the employee's roster
    log.
  * The version for a date is resolved behind the per-day registry view
    that the plan, the runtime and the booking predicates already use.
* **Validation** (in order): shape; employee (hourly technologist or front
  desk, still employed); effective date (`past_date`, `not_monday`, 14 days'
  notice, horizon, before the end of employment); each shift as for a hire
  (rooms of the employee's skills, desk for the front desk, breaks, cap);
  `unchanged` when equal to the roster already in force; then the
  whole-roster rules — at most six days a rota week, **net rostered time
  ≤ the contract** in each rota week (the employee's own original roster
  may always be restored), rest. Then the batch's composed timelines are
  checked as a whole:
  * no version after a termination (in either batch order);
  * rest across every version boundary, withdrawals included;
  * a trainee still completing training within 60 days
    (`training_incomplete`);
  * every approved extension still valid on its new shift
    (`extension_conflict`, naming the date: refused, never silently
    cancelled).
* **Rest rule, one for hiring, rosters and overtime:** 10 hours from a
  shift's cap — the latest permitted release, min(start + 12 h,
  23:59:59) — to the next day's start. Approved extensions and hold-over
  cannot pass the cap, so no approval can break the rule. Tested at the
  day boundary within a pattern (A→B and B→A), across versions, after a
  withdrawal, and at exactly 10 hours.
* **Contract and money.** Contracted hours, rate, accrual weights, notice
  and severance never change with the roster; nothing new is posted. A full
  week under a four-day or a zero-hour roster pays Riley's contract exactly
  ($1,440 + benefits); severance is identical before and after a cut.
* **Atomic batches and displacement.** A batch commits in input order,
  moves trainees' productive dates, then reconciles once, against the final
  timeline, from the earliest affected day: the same survivor and
  re-offer lifecycle as Slice 2. Causes: `employment_ended`,
  `roster_changed`, `training_delayed` (joined by `+`). A coordinated
  exchange (an XR+CT hire and Sam swapping Monday and Tuesday) displaces
  nobody in either order, while each half alone displaces 11 bookings.
  A termination voids pending versions on or after the end (logged), as
  it voids extensions (unchanged).
* **Training.** A roster change before the first working day or during
  training recomputes the productive date from the composed timeline (past
  days are the attended ones). A delay displaces the bookings that relied on
  the trainee (including XR bookings losing a shared technologist); an
  earlier completion opens capacity. Capacity is never granted before
  completion (invariant).
* **Preview.** Rostered hours before and after against the unchanged
  contract, the interval, a trainee's productive date before and after, the
  extensions kept, the batch's slots opened and closed by modality, exact
  displacements with provisional replacements, and the guaranteed weekly
  contract cost apart from forecasts (contracted time not rostered,
  extensions kept, displaced bookings). Pure and independent of the hidden
  future (tested).
* **Interfaces.** `api.Action` gains `op` and `effective`;
  `api.Employee.RosterChanges`; `World/Session.RosterPlan`; CLI actions
  files; session v2; the TUI roster editor (staff screen `m`, enter on a
  roster row): weekly or fortnightly grid, field editing, effective Monday,
  withdrawal, live preview.
* **Compatibility.**
  * State digest still `dreamteam.state/v3`: versions are encoded only when
    present.
  * A genuine 0.2.0-dev.2 save, built with that commit's binary and kept
    in `testdata/sessions`, replays under this build with every digest
    equal. Its batches apply with Slice 2's action set (`ApplySlice2`),
    where `set_roster` was an unknown action. Before the version bump the
    same file failed replay (fail-before-fix). Continued, it becomes the
    prefix of a new v2 record
    (`prefix:earlier_build_exact+native_exact`).
  * v1 → v2 continuation with roster changes replays.
  * The four V0.1 goldens are unchanged under the manifest. The Slice 2
    golden differs only in its action and run digests: its mixed batch's
    refusal message now names `set_roster`. This is recorded in
    `MIGRATIONS.json` and confirmed by diffing the dev.2 and dev.3 action
    records (log, ledger and state identical).
  * A sixth reference, `established-seed1-60d-roster.json`, pins the roster
    plan (36 displacements, all rebooked).
* Verification:
  * Full uncached suite (`go test -count=1 -timeout 60m ./...`) passes on
    the Pi in 8 min 19 s: cmd 86 s, session 88.5 s, sim 488 s, tui 24 s.
    `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * Acceptance (`-tags acceptance -run Acceptance`) passes on joofi in
    128 min: the V0.1 criteria, staffing turnover (155 hires over 720
    days) and roster churn (340 roster versions accepted over 720 days),
    with the invariants checked daily; 156 tests and subtests, no failure.
  * All six golden references and the Slice 2 session fixture give
    byte-identical outputs on the Pi and on joofi.
  * Fail-before-fix mutations: all 20 guards were caught (not_monday,
    notice, contract, pattern rest, boundary rest, extension conflict,
    extension on the composed view, training projection, trainee capacity
    early, void on termination, composed not_employed, duplicate date
    claim, unchanged, version resolution, reconcile after roster,
    per-item reconcile, guarantee after a roster cut, digest omit-empty,
    Slice 2 batch mode, original-roster restore only when exact).
* Performance (Pi, `docs/MEASUREMENTS.md`): the 90-day run is unchanged
  (best of 8 interleaved: 3.40 s dev.2, 3.41 s dev.3); EstablishedPlay90
  allocations unchanged (78.6 MB, 1.241M). A roster preview costs
  28.2–28.3 ms, a commit 25.3–25.7 ms; the editor renders in 48 ms.
* Limitations: the rest rule is conservative (measured from the cap, see
  deviation 1); break rules are not applied to approved extensions;
  radiologist and supervision displacement remain untestable until the
  reading actions; the result schema is still v1; amd64 and macOS were
  not run; coverage differences are reported per batch, not per change;
  the TUI edits only the first cover of a shift.

---

## V0.2 Slice 2 — hiring and termination (0.2.0-dev.2)

The owner approved two player actions, `hire` and `terminate` (2026-10-15).
`set_roster`, `set_reading_allocation` and `contract_teleradiology` are not
in this slice. Design: `docs/V0.2_DESIGN.md` §5.7, with the deviations
identified there. Without a V0.2 action every V0.1 outcome is unchanged.

* **Registry and identity.**
  * Hires get the first unused `H00001`, `H00002`, … (scenario IDs skipped),
    allocated at commit only.
  * A hire's terms are frozen in state; records are append-only and never
    reused.
  * Staffing sets are growable (no 64-employee mask); the scenario limit is
    a 1,000-employee sanity bound.
  * The state digest encoding is `dreamteam.state/v3`.
* **Hire.**
  * Validated in a fixed order: offer table (role, skills, exact standard or
    premium rate), contract 1–2,400 minutes, start ≥ today + the tier's lead,
    roster rules, 40 h (technologist) or 24 h (desk) of training within 60
    days, and the 8-technologist / 4-desk caps including notice.
  * The recruiting fee posts at acceptance (recruiting / A/P).
  * From the start date the employee is paid (contract guarantee in equal
    sevenths, training as worked time) but works under a training view with
    no capacity: no room, registration, supervision, hold-over or extension.
  * Independent from the midnight after the 40th (24th) attended hour.
    Future slots open from that date.
* **Terminate.**
  * Hourly technologists and front desk only.
  * Last day ≥ 14 days after today and the start; identical repeats are
    `unchanged`; a moved or withdrawn notice is refused.
  * Contract pay continues to the last day, whatever the roster; the last
    week closes on the last day.
  * Severance (two contracted weeks with benefits) is recognized at
    acceptance (severance / severance payable) and paid on the first payday
    after the last day.
  * Approved extensions after the end are voided with a log line.
  * Terminating a pending hire keeps the fee, runs the contract from the
    start and owes severance.
* **Displacement** (when the batch commits; design §5.7):
  * The days after an employee's end are re-planned from the final registry
    (a replacement hired in the same batch counts).
  * Bookings are re-inserted in survivor order with the booking rule's own
    predicates; those that cannot stay are displaced, all released first,
    then offered a same-category slot no earlier than their original time,
    accepted within the patient's tolerance of the original tolerance
    anchor, else lost to displacement.
  * The behavioral attempt, confirmation and no-show lead basis are kept, so
    no new random draw is made.
  * Each displacement has a record and its own counters, outside demand,
    loss, cancellation and no-show figures.
* **Atomic batches.** A batch with a V0.2 action holds only V0.2 actions
  (`mixed_batch`); one refusal refuses all (`batch_rejected`) and commits
  nothing; a valid batch commits in input order and reconciles once. V0.1
  batches are unchanged.
* **Finance.** New accounts `recruiting`, `severance`, `severance_payable`
  (listed only once they hold a balance, so the baseline digest is
  unchanged); statement lines; labor includes severance; the cash bridge
  and cash identity include severance; severance due appears in
  obligations.
* **Previews.** `PreviewStaffing`: validation, terms, timeline, slots a hire
  opens, exact potential displacements with capacity-only provisional
  replacements, guaranteed and forecast money in separate tables. Pure.
* **Session v2.** `dreamteam.session/v2`; v1 read by separate strict DTOs;
  an engine registry (`0.1.*`, `0.2.0-dev.0.*`, `0.2.0-dev.1`) replays v1
  records with outcomes verified; a continued v1 session saves as v2 with
  the v1 record embedded and this build's transition digests.
* **Interfaces.** API (`api.Action` fields, `ValidateBatch`,
  `PreviewStaffing`), CLI actions files and replay, TUI staff screen `m`.
* **Compatibility and neutrality.**
  * The four historical goldens are byte-for-byte unchanged. They still differ
    from this build in exactly `digests.state`, `digests.run` and
    `replay.runDigest`; every outcome field, the log, ledger and action
    digests and the baseline digest are identical (90 days and the V0.1
    action fixture included). `MIGRATIONS.json` now records this build's
    new values (`dreamteam.state/v3`) with the explanation.
  * A fifth reference, `established-seed1-60d-staffing.json` (V0.2 actions,
    written by this build), pins hire and terminate end to end.
  * The Step 0C sustainable start and lead candidate reproduce their
    committed per-seed and aggregate results exactly (9 seeds each).
  * All five references are byte-identical on the Pi 4 (Cortex-A72) and the
    Cortex-A76 machine. amd64 and macOS not run (no machine).
* **Regression tests.** New: `staffing_test.go` (lifecycle and training
  capacity, the §5.4 partial-week example to the cent, notice and severance
  with a zero roster, every rejection code, repeats and conflicts, atomic
  batches, headcount caps with notice, displacement records and preview
  agreement, a replacement in the same batch displacing nobody,
  shared-technologist displacement, tolerance anchor over 12
  terminations, read-only and hidden-future-independent previews, more than
  64 employees, the evaluation endpoint), `empset_test.go`,
  session `staffing_test.go` (save/replay at every lifecycle boundary,
  chunked and event-budgeted replay, v1 → v2 continuation and tampering),
  TUI `staff_test.go`, and two acceptance-tagged checks (invariants after
  every transaction for 50 seeds with the reference plan; 720 days of
  turnover past 64 lifetime employees, reproducible).
  * **Fail-before-fix evidence by mutation**: each guard was disabled in
    turn and its test run; all 17 fail as required — trainee capacity,
    guarantee from the start date, severance posting, severance payday,
    notice pay, reconciliation, attempt preserved, tolerance anchor (at the
    call site and in the rule), shared-technologist survivors, batch
    atomicity, notice in the cap, training required, v1 transition check,
    preview purity, sets past 64, v1 replay semantics for V0.2 type names. The first anchor test did not catch a
    more lenient rule; `TestDisplacementToleranceAnchor` and
    `TestAcceptReplacementAnchor` were added and do.
* **Remaining limitations.**
  * No `set_roster`: a hire's roster is fixed at hire; contracts change only
    by termination and hiring (owner decision).
  * Contrast-supervision displacement cannot arise yet (radiologists cannot
    be terminated); the path is shared with the tested staffing check.
  * The result schema stays `dreamteam.result/v1` with additive fields;
    the T90 vector and gates are not built.
  * Hire names are their IDs. The TUI offers fixed roster patterns, not a
    roster editor.
  * amd64/macOS determinism not yet run.
* **Performance** (MEASUREMENTS.md): about +5% CPU and +7–12% memory on
  a 90-day run (best of ten on a loaded, throttled Pi: 3.20 → 3.49 s); a
  termination preview ≈21–24 ms and its commit ≈30 ms; the staff screen
  renders in ≈25 ms with a live preview.
* **Verification:**
  * Full uncached `go test -count=1 -timeout 30m ./...` passed: 7 packages,
    8.5 min on the loaded Pi (sim 503 s, cmd 83 s, session 75 s). `go vet`
    clean for the default, calibration and acceptance tags; gofmt clean.
  * Goldens: the four historical references match under the migration
    manifest (90 days included); the V0.2 reference is byte-identical;
    `-verify-replay` matched on all five.
  * Acceptance on the Cortex-A76 (`-tags acceptance` test binary), all
    passed, ≈1 h 31 min with the suites sharing four cores: V0.1 criterion
    10 (50 seeds × 90 days, invariants after every transaction) and
    criterion 9 (1,000 chunkings, 20.7 s); new: invariants after every
    transaction for 50 seeds × 90 days with the V0.2 reference plan, and
    720 days of turnover — 155 hires, registry 163, slot staff sets three
    words wide, reproducible (44.5 s).
* **Slice 3 prerequisites (exact).**
  1. Date-aware roster versions: `planFor` composes registry terms by
     date (`dayTerms`); `set_roster` adds dated versions behind it, with the
     §6.4 composition (version keys, insert/replace/remove, tombstones).
  2. Reuse `reconcile` for roster reductions (cause `roster_reduction`)
     and `recoverDays` for additions; extensions that no longer fit a new
     version must reject (§6.4), not be voided.
  3. Training projections must be recomputed on a roster change before
     the productive date (`training_delayed` displacement).
  4. The regular-roster ≤ contract rule and §6.4's legacy-pattern rule for
     initial employees (frozen original week totals).
  5. Session v2 already carries arbitrary V0.2 actions; add the
     `set_roster` DTO fields and canonical comparison.
  6. Then the owner's benchmark: no action, partial and full
     reorganization over the same 20 seeds at T30, T90 and T180.

---

## V0.2 Slice 1 — employee contract and capacity foundation (0.2.0-dev.1)

The owner approved CONDITIONAL GO (2026-10-15). Decisions, design §18:
* The measured 1.5–5% margin band.
* ×1.30 demand.
* A shared-staffing start, with reorganization as the player's objective.
* Contracts kept at takeover; reductions only through termination and hiring.
* Provisional scenario-specific T90 gates.

Slice 1 builds the foundation and **adds no player action**. `established` and
`mini` outcomes are unchanged.

* **Contracts.** Hourly employees have contracted weekly hours. A scenario
  may state them (`contractedWeeklyMinutes`, 1–2,400); otherwise they are
  inferred from the roster as the smaller net rota week: E01/E03/E05 40 h,
  E04/E06 39.5 h. The accrual weights (the original roster by rota week and
  weekday) are frozen at load. Salaried staff have no contract hours; their
  salary is the guarantee.
* **Guaranteed pay.** Worked pay is posted exactly as before. At accounting
  close the week's guarantee earns the day's share, and a signed provision
  tops worked pay up to the contract (wages and benefits). Sunday's close
  finalizes the week. Guaranteed time is never worked time, so it creates no
  overtime and is never paid twice. A payday pays accrued payroll less the
  week's still-open guarantee. The provision stays inside accrued payroll
  rather than a new account, so the evaluation baseline's account list and
  digest are unchanged (design §5.5, revision of §9).
* **Employment registry** (state, digested): stable ID, lifecycle stage
  (only `active` until hiring exists), employed dates, contract and the
  week's pay counters. Checked every transaction. Observations show stage,
  contract minutes, the inferred flag, this week's open guarantee and the
  severance termination would owe (two contracted weeks with benefits).
* **Whole-schedule staffing check** (`World.CheckStaffing`, read-only). For
  a hypothetical roster change from a future day, it re-books every booked
  appointment in survivor order with the booking rule's own predicates. It
  reports each one that would be left `unstaffed`, without supervision
  (`no_supervision`), or competing for shared technologists
  (`shared_capacity`). It cancels, moves and displaces nothing, and makes no
  draw or log entry.
  * About 11 ms over the full horizon.
  * 150 unchanged-roster checks (established, the sustainable start and
    mini; 10 seeds; takeover to day 89) found no inconsistent existing
    booking.
* **Transition analysis** (design §5.6).
  * The shared start (`W130-SHR-MRft`) is built from existing scenario
    inputs. Its takeover contracts are Riley 40 h, Sam 39.5 h and Alex 40 h.
  * The lead staffing is reachable with the five approved actions:
    * `hire` Jamie, plus 27.5 h and 20 h replacements for Sam and Alex.
    * `terminate` Sam and Alex.
    * `set_roster` for Riley.
    * `set_reading_allocation` for the radiologists.
  * The plan reaches exactly the proposed 8-technologist cap during notice.
    At the earliest it is complete around day 41.
  * The saving comes from the part-time conversions: keeping every contract
    and only adding Jamie measured −$1.2k a month in Step 0C.
  * Estimated upfront cost is ≈$20–25k (recruiting, severance, training,
    overlap) against ≈$6k a month of headroom. **The full plan may not pay
    back within T90.** The dynamic benchmark must measure it; gameplay
    documentation does not promise +2.8%.
* **Compatibility.**
  * The state encoding is `dreamteam.state/v2`. Each historical golden
    reference differs in exactly `digests.state`, `digests.run` and
    `replay.runDigest`. Log, ledger, actions and baseline digests and every
    outcome value are identical.
  * The references stay byte-for-byte. `testdata/golden/MIGRATIONS.json`
    records old and new values with the explanation, and `-golden-migrations`
    requires both exactly.
  * Session records from v1-state builds (`0.1.*`, `0.2.0-dev.0.*`) replay
    with outcomes verified (action results, log, ledger, actions, summary)
    and the state change reported. Any other engine still needs every
    digest.
  * The Step 0C sustainable start and the lead candidate reproduce their
    committed per-seed results exactly (9 seeds each).
* **Regression tests.** Written before the implementation and shown failing
  against stubs:
  * The design's worked payroll examples to the cent (no work, 32 of 40,
    45 h with premium, 28 of 20, 40 of 40).
  * Provision reversal, partial week and rounding.
  * Contract inference; the registry in the state digest.
  * Guaranteed pay with a roster below the contract, and payday exclusion.
  * Neutrality of contracts at or below the roster: ledger and log hashes
    equal.
  * Extensions taking the week into overtime; `TestGuaranteeWithOvertime`
    passed against the stub too, so it is a guard.
  * Scenario validation and the start and lead contracts.
  * Unchanged-schedule, lost-coverage, shared-capacity, supervision,
    read-only and rejection staffing checks.
  * Golden migration strictness and v1-state replay. The replay test fails
    with the rule disabled.
* **Remaining for later slices:**
  * Growable staff sets (64-employee bit set).
  * Date-aware rosters and plan versions.
  * Recruiting and severance postings.
  * Displacement.
  * The five actions.
  * Session v2.
  * The named `sustainable` scenario.
  * Service-gate calibration from dynamic plans.
* **Verification:**
  * Full uncached `go test -count=1 ./...` passed: 7 packages, sim 349 s,
    cmd 65 s.
  * `go vet` clean (default, calibration and acceptance tags); gofmt clean.
  * The four goldens are unchanged and match under the migration manifest
    (90 days included).
  * `-verify-replay` matched on every reference.
  * The 90-day run time is unchanged within noise (MEASUREMENTS.md).

---

## V0.2 step 0C — late shifts and staffing architectures (0.2.0-dev.0.2)

Owner option G (2026-10-15): fix the noon shift-start limit, then test
explicitly different, fully costed technologist staffing architectures for a
larger practice. No action code; `established` and `mini` untouched.

* **Late shifts (engine).** A shift's cap, holdover included, is now the
  earlier of start + `maxShiftHours` and accounting close (23:59:59):
  `sim.Policies.ShiftCap`, used by runtime staffing, `extend_shift`, the
  roster view and alerts. Validation rejects shifts longer than
  `maxShiftHours` or ending after 23:59:59, not shifts starting after noon.
  Messages name accounting close when it is the binding limit.
  * Regression tests written failing first: `TestShortLateShifts`
    (16:00–20:00, 17:00–20:00, 11:30–20:00, 19:00–23:59 valid; over-cap and
    past-midnight rejected), `TestShiftCap`, `TestShortLateShiftsRun` (late
    shifts staff their rooms; invariants hold daily).
  * `TestRegress05MidnightBoundary` now runs its fixture: the late patient
    is rescheduled and nobody works past midnight.
  * Every existing roster starts before noon, so caps, goldens, replay and
    determinism are unchanged.
* **Harness** (calibration build tag, read-only): records conflict timing
  (weekday × hour), qualified staffed hours, technologist duty hours,
  reports by modality, estimated radiologist cost, payables, accrued
  payroll, receivables and the lowest daily cash. The test-only loss probe
  records each shared-conflict slot's start. The 20-seed Step 0B candidate
  reproduces exactly.
* **Sweep** (`docs/calibration/feasibility/step0c/`; 74 manifests; 9 paired
  seeds, 20 for the leaders):
  * 62% of CT shared-technologist losses were on Mondays.
  * A part-time MR evening is worth +$4.9k a month with no volume change.
  * Dedicated or float XR/CT coverage removes 80–95% of shared losses.
  * Part-time beats full-time for the same coverage by $2.6–7.8k a month.
  * The extra volume overwhelms reading: urgent p90 188 → 1,046 minutes at
    the base allocation.
* **Lead candidate** (×1.30 demand, 20 seeds): dedicated XR/CT day core,
  part-time evenings, reading at 62.5/52.5%.
  * Margin +2.8% (1.5–5.0%, median 2.6%); profitable on 20/20 seeds at T90
    and in days 31–90; lowest cash ≈$51k.
  * Breach rates better than the shared control: urgent 10.1% vs 10.4%,
    soon 27% vs 37%, routine 4.2% vs 7.0%.
  * 145 fewer lost referrals and 101 fewer walk-ins leaving.
  * At ×1.15 demand every architecture loses money (lead −4.0%); at ×1.50,
    out of range, +6.3%.
* **Verdict: CONDITIONAL GO.** Owner decisions (design §18): accept a
  ≈1.5–5% band; accept ×1.30 as the defining demand; choose the starting
  state (the candidate, or the shared practice with reorganization as the
  player's task); decide how part-time contracts are reached; freeze the
  service tolerances.
* **Housekeeping:** removed a committed `__pycache__` file; `.gitignore`
  ignores `__pycache__/`.
* **Verification:** new and updated regression tests; `go vet -tags
  calibration`; control reproduction; full uncached `go test -count=1 ./...`.

---

## V0.2 step 0B — CT/XR booking losses and templates (0.2.0-dev.0.1)

Owner option B (2026-10-15): find why CT/XR referrals are lost while their
rooms are ≈40% used, then test CT/XR templates as a permitted static lever.
No action, scenario or gameplay change; `established` and `mini` untouched.

* **Booking-loss probe** (diagnostics). Production change, explained: one
  function field `state.lossProbe` (excluded from the state digest, like the
  existing test hooks `dropOrder` and `forceNoShow`) and one nil-checked
  call where `offer` records a loss. The diagnosis itself
  (`internal/sim/lossprobe_export_test.go`, test builds only) classifies
  every compatible slot in the lost patient's acceptance window by the first
  FirstFit check it fails (booked, not rostered, no supervision, notice,
  shared technologist). It assigns one primary cause by documented
  precedence and also records every cause present. It reads state only.
* **Neutrality:** `TestLossProbeIsPassive` (log, ledger, state and action
  digests identical with and without the probe; every loss diagnosed once;
  no unexplained free slot). The 20-seed Step 0 candidate reproduces its
  per-seed results and fingerprint exactly with the probe on. Goldens
  unchanged.
* **Finding:** at the Step 0 best candidate, 72% of lost CT/XR referrals had
  a free, staffed slot in their window, blocked because one technologist
  covers both rooms (Riley mornings and Mondays, Sam evenings). 92% of them
  are urgent or soon patients. The template category mix explains ≈7% of CT
  losses.
* **Templates and coverage** (`docs/calibration/feasibility/step0b/`; 13
  manifests, 9 paired seeds, the best and its reference at 20):
  * 12-minute XR slots: +$1.16k per 90 days (CI +0.36 to +1.96), margin
    +0.4% → +0.5%. Recovered referrals displace walk-ins.
  * XR halved in shared hours: −$2.3k per 30 days.
  * XR grid staggered 5 minutes: −$5.2k (fragments shared time).
  * An XR/CT technologist (day or evening): recovers 145–170 losses and pays
    for itself, no more. Urgent breaches rise ×2–4: reading becomes the
    constraint. Restoring it with the 65/55 allocation: −$5.3k.
* **Verdict:** gate still not passed. The binding constraints are shared
  technologist staffing, low XR contribution, evening reading coverage, and
  fixed costs with salaried reading. Templates are exhausted. The best
  configuration is +0.5% mean T90 margin, 14 of 20 seeds profitable.
* **Tooling:** `report.py` takes a results directory and paired references
  and adds booking-loss cause tables. The calibration harness records loss
  causes per window.
* **Verification:** `TestLossProbeIsPassive`; `go vet -tags calibration`;
  calibration summary unchanged; full uncached `go test -count=1 ./...`.
* **Open:** owner decision on the premise (design §17.2: adopt the
  near-break-even candidate, build reading coverage and test teleradiology
  substitution dynamically, or revisit the staffing model). The noon
  shift-start fix stays a separate prerequisite. NO GO for action code.

---

## V0.2 step 0 — design draft 2 and the feasibility gate (0.2.0-dev.0)

Design and static experiments only: no engine, scenario, action or replay
change. No-action outcomes are those of v0.1.0.

* **V0.1.0 published.** GitHub Release "DreamTeam v0.1.0" from the existing
  tag (89b1779), after CI (run 38663474125) and a full uncached suite on the
  tagged tree passed.
* **Design draft 2** (`docs/V0.2_DESIGN.md`) applies the six owner decisions
  and resolves the 18 review findings (Appendix A): staff registry separate
  from the scenario; contracted, rostered and worked hours with guaranteed
  weekly pay and no double payment (legacy contracts inferred, including
  two 39.5-hour employees); the lifecycle from recruiting to severance; five
  actions with schemas, atomic batches, no-op rules and previews separating
  guaranteed from forecast costs; whole-schedule booking reconciliation and
  a practice-caused displacement lifecycle; employed vs outsourced reading
  with no price floor; finance postings; neutrality and golden migration;
  session v2 with a v1 reader; the T90 metric vector, baseline-relative
  service gates and an exploit table. `docs/ROADMAP.md` re-scoped V0.2–V0.6
  (V0.6 restored), V0.1 sections unchanged.
* **Step 0 feasibility sweep** (`docs/calibration/feasibility/`): 53 static
  overlays (demand ×1.00–1.30 uniform and imaging-weighted, XR/CT, US and MR
  technologists, 40-minute US slots, radiologist allocation 50/40–65/55),
  9 paired seeds at 30 and 90 days, the best three and the base at 20 seeds,
  plus a labelled out-of-range diagnostic (×1.50). Calibration harness:
  `DT_EXP_DIR`/`DT_RES_DIR`, parallel manifests, more aggregated figures;
  historical results untouched (summary regenerates byte-identically).
  `generate.py` writes the manifests, `report.py` the tables.
* **Gate result: not passed.** The best permitted configuration
  (imaging-weighted ×1.30 + MR evening technologist) breaks even: +0.4%
  mean T90 margin (−0.9 to +2.4), T90 profit +$2.8k mean (95% CI −$0.2k to
  +$5.7k), 12 of 20 seeds profitable. Demand ×1.50 gives only +1.6%: the
  practice cannot book it (CT and XR lose hundreds of referrals at ≈40% room
  use; US and MR are single rooms). Fixed costs and salaried reading are
  ≈58% of expenses; cutting the allocation collapses reporting at higher
  volume; US 40-minute slots change nothing; only the MR evening
  technologist pays.
* **Engine constraint found:** no shift may start after 12:00 (validation:
  start + 12 h `maxShiftHours` before midnight), so part-time evening shifts
  cannot be expressed; those cells are blocked (upper bound ≈+2.4–3.8%).
* **Verification:** `go vet -tags calibration ./internal/sim`;
  `TestCalibrationSummary` (historical summary unchanged); the sweep itself
  (four runs, all passing); `go test ./...`.
* **Open:** owner decision on the scenario premise (draft 2 §17.2: accept
  break-even, CT/XR template levers, cost base, reimbursement, teleradiology
  substitution, a second US room); then re-sweep. NO GO for action code.

---

## V0.1.0 — release preparation (v0.1.0)

Release preparation only: no change to mechanics, RNG, scenarios,
actions, accounting or replay.

* **Accepted deviations** (owner, 2026-10-14), recorded in
  `docs/ACCEPTANCE.md`, `docs/ROADMAP.md` and the README: the Pi 4's ≈3.3 s
  headless 90-day run (2 s target waived for V0.1; SHA-256 keyed RNG kept)
  and the naturally small or empty takeover worklist of the struggling
  default practice (no manufactured backlog; the 5–20 target moves to a
  future reporting-constrained scenario).
* **CI**: the Node 20 deprecation came from `actions/checkout@v4`,
  `actions/setup-go@v5`, `actions/upload-artifact@v4` and
  `actions/download-artifact@v4`; now `checkout@v7`, `setup-go@v7`,
  `upload-artifact@v7`, `download-artifact@v8` (all Node 24). Matrix and
  golden checks unchanged.
* **Release notes**: `docs/RELEASE_NOTES.md` (new; newest first).
* **Version** `0.1.0`. The golden references were not regenerated (no
  outcome changed): they still name the engine that wrote them, and the
  comparison reports the engine difference with outcomes identical.

---

## Steps 19–20 — Performance pass, acceptance results, release docs (v0.1.0-dev.20)

**Acceptance results.** Criterion 10: invariants after every transaction,
50 seeds × 90 days — passed (53.5 min on four Cortex-A76 cores; one seed
takes ≈7 min on the Pi 4). Criterion 9: 1,000 random chunkings of the real
simulation — passed. Criterion 8: the first CI run produced byte-identical
golden runs on linux/amd64, linux/arm64 and darwin/arm64. Every criterion
and its evidence: `docs/ACCEPTANCE.md`.

**Performance (criterion 16)**, details in `docs/MEASUREMENTS.md`:

| | Pi 4 (throttled, loaded) | Cortex-A76 (joofi) | Target |
|---|---|---|---|
| Warm-up | 1.44–1.49 s best | 0.53 s | < 1.5 s |
| `run -days 90` | ≈3.3 s | 1.1–1.2 s | < 2 s |
| TUI at 15 min/s | 7.4–8.2% of a core | — | < 15% |
| TUI paused | 0.4% (was 3.0%) | — | — |
| Max speed, worst frame | 52–94 ms | — | `space` within 100 ms |

The Pi 4 misses the 90-day target: its CPU has no SHA-256 instructions and
the keyed draws (the determinism contract) are SHA-256; the profile has no
other dominant hotspot. Changing the draw hash would change every outcome
and digest — an engine-version decision for the owner, not a tune-up. A
per-day free-slot cache for FirstFit was tried (outcomes identical) and
reverted: 3% at best.

**Changes.** The TUI's frame clock stops while paused and the renderer runs
at 20 fps (idle CPU 3.0% → 0.4%; outcomes unaffected). New benchmarks:
`BenchmarkEstablishedPlay90` (sim) and `BenchmarkMaxSpeedFrames` (TUI frame
times at Max). `dreamteam help` lists the `scenario` command.

**Release docs.** `README.md` (what it is, quick start, controls, actions,
scenarios, headless runs, sessions and replay, development, documents,
known limitations) and `docs/ACCEPTANCE.md`.

**Open for the owner.** (1) Accept criterion 16 on the Pi 4 as measured, or
decide on a faster keyed-draw hash for a later engine version. (2)
Criterion 1's "unread backlog at the takeover" does not hold for the
struggling practice (reading is over-provisioned at its volume). (3) Tag
V0.1 when satisfied.

---

## Steps 16–17 — Acceptance pass, part 1: TUI polish and end-to-end replay (v0.1.0-dev.18.3)

The V0.1 acceptance pass (ROADMAP §2), with the owner's list. The
checklist with every criterion's evidence is `docs/ACCEPTANCE.md`.

**Built.**
* **Help screen**: `?` from any screen shows the keys and the worklist
  rule; `?` or `esc` returns where it came from.
* **Event-log filters**: on the log screen `tab` cycles the category,
  `!` the severity (all · warnings & alerts · alerts only); the title shows
  the filter and how many entries match.
* **`run -log events.jsonl`**: the observable log from the takeover, one
  JSON object per line (seq, time, severity, category, text), through an
  output-only sink (never part of the state; the result is byte-identical
  with and without it). `-summary text|json` is an alias of `-format`.
* **NO_COLOR**: colors follow the profile Bubble Tea detects from the
  environment; with `NO_COLOR` no color sequence remains and severity stays
  marked (⚠ ‼). Documented in `dreamteam help`.
* **CI** (`.github/workflows/ci.yml`): vet (all tags), gofmt, the full
  suite and the golden references on linux/amd64, linux/arm64 and
  darwin/arm64, then a byte-for-byte comparison of the three platforms'
  results.
* **Acceptance suite** (`-tags acceptance`): criterion 9 on the real
  simulation (1,000 random chunkings of warm-up, takeover and a day of
  operations) and criterion 10 (invariants after every transaction, 50
  seeds × 90 days).

**Tests.** `TestTUIRunReplaysHeadless` (criterion 8: three randomized TUI
sessions — speeds, frame gaps 10–610 ms, pauses, steps, alert auto-pauses,
actions submitted from the actions screen — each reproduced by a headless
session from its action record and by replay; all five digests),
`TestHelpAndLogFilters`, `TestNoColor` (fails without NO_COLOR),
`TestRunLogJSONL`, `TestScreenSnapshots` (ten screens at 80×24, 120×36,
160×48; `-update-screens` to regenerate), and `TestRenderFitsTerminal` now
also covers 140×40, briefing, actions, help and the end-of-day summary.

**Deviation recorded.** Criterion 1 asks for an unread backlog at the
takeover; the struggling practice usually has none (reading is
over-provisioned for its volume, as characterized in step 18).

---

## Step 18.2 — Urgency always takes precedence on the worklist (v0.1.0-dev.18.2)

**A deliberate gameplay rule change (owner decision), not a calibration
correction.** Step 18.1 showed that shortest-first could postpone a long
urgent study indefinitely while short routine studies kept arriving. Now
urgent studies are offered before all others under every worklist policy,
and the policy orders within each class: default unchanged (urgent and
overdue first, then FIFO), FIFO → urgent first then FIFO, oldest-first →
urgent first then oldest, shortest-first → urgent first then shortest
published expected read. The existing `urgent` priority is the class (no
new severity); reads in progress are never interrupted; hidden read
durations are never used; ties still fall to the study ID. Reading blocks,
staffing, read durations and reimbursement are unchanged.

The TUI actions screen labels the choices "urgent first, then …" and says
"Urgent studies are always read first; the policy orders the rest"; the
worklist screen, API comments and SIMULATION_DESIGN §6.8 say the same.

**Tests.** `TestRegress18UrgentLongStudyVsShortRoutine` (every policy reads
the urgent MR next after the read in progress; before: FIFO and
oldest-first after 5 min, shortest-first never in 7 h) and
`TestRegress18WorklistUrgentFirst` (exact order per policy on a hand-built
worklist, determinism, shortest-first unchanged with no urgent study
waiting) fail on 049b304 and pass now. `TestWorklistPolicyOrdering` checks
shortest-first within each class. Full uncached suite green.

**Effect** (nine seeds, paired; details in the calibration report):
exams, lost bookings, walk-ins, revenue, opex and margin identical in every
run. 90 days: FIFO urgent breaches 32.9 → 15.1 and urgent p90 599 → 282
min (now equal to the default); shortest-first urgent breaches 18.2 → 16.3,
urgent MR median 99 → 52 min, routine p50 still 22 min. The default policy
is unchanged. The remaining urgent tail is the evening/weekend reading
coverage gap; with this rule the calibration phase is closed.

**Goldens.** Only `established-seed1-7d-actions.json` (it sets FIFO) was
regenerated; the three default-policy references are identical (compared
with the engine mismatch allowed). Its changes: reading order (urgent TAT
p50 1,407 → 803 s, max 3,628 → 2,256 s; breaches 13 → 12; one alert fewer;
radiologist busy time shifted between the two), the log digest
(6e34bd068f7f… → 625dfe83e5d8…), the ledger digest (6c18847e1bad… →
4482cec1c8d3…: revenue posts at report final, so entries move in time;
window money and balances are identical), state 72f117978ca0… →
5a626a6308b7…, run 9a3d8861926c… → fb0ccb14a79c…. Patient flow, exams and
money unchanged.

---

## Step 18.1 — Calibration review remediation (v0.1.0-dev.18.1)

The step 18 review (`docs/reviews/step-18-astra-review.md`, CONDITIONAL GO)
found five defects. Each got a regression test that fails on `b1a9ffb`,
then the smallest fix; every experiment was rerun on the fixed engine.

**Fixes.**
* `extend_shift` validates the raw minutes before converting them; huge
  values no longer wrap into a negative extension.
* Shared-staff booking is a real feasibility check: distinct qualified
  technologists for every overlapping appointment, handovers only at
  coverage boundaries, whatever the booking order. Default outcomes are
  identical; dispatch is unchanged; warm-up time is unchanged.
* Walk-ins: arrival, same-visit admission, the offer (declined by policy
  or unservable) and whether it booked, the same-visit exam and a later
  return exam are separate counters. Walk-in-origin booking events are
  out of referral demand (`demand.walkInOrigin`). Acceptance and
  abandonment are over the cohort of walk-ins who arrived in the period,
  each classified once (imaged same visit, converted, left, lost, on
  site). The TUI summary reads "walk-ins N: admitted, offered later
  (booked), left".
* The strategy helper no longer requests extensions past the scheduled
  close; the harness fails on any rejected action.

**Reproducible experiments.** `docs/calibration/experiments/*.json`
(scenario, overlay, strategy, seeds, horizon) run with
`go test -tags calibration ./internal/sim -run 'CalibrationExperiments$'`
and write `docs/calibration/results/<name>.json` (engine, fingerprints,
overlay hash, per-seed windows, aggregates) and `results/summary.md`. The
harness adds an order cohort, requested lead vs capacity delay, approved
overtime and turnaround by priority and modality.

**Results** (nine seeds; full tables in the report):

| | 30 days | 90 days |
|---|---|---|
| Baseline exams / margin | 1,363, −12.5% | 1,296 per 30 d, −14.0% |
| Three more technologists | 1,458, −16.7% | 1,390 per 30 d, −18.6% (step 18: −18.7%) |
| Faster XR (×0.75) | 1,390, −11.6% | 1,321 per 30 d, −13.1% |
| Reading allocation 45%/38% | TAT p50 20 h — unstable | TAT p50 22 h |

Strategies (30 days): B 1,431 exams, −9.4%; C 1,394, −13.4%; D 1,461,
−10.4%; E 1,394, −12.6%; F TAT p50 34 → 22 min (≈35%, not half). C and D
moved by at most one exam and 0.1 points with the booking fix; the
struggling-practice characterization stands.

**Urgent turnaround.** The default's urgent tail is a reading-coverage
gap: ≈¾ of urgent breaches are studies acquired after the last reading
block and read the next morning or Monday (max 31–41 h under every
policy). FIFO adds ≈15 in-hours urgent breaches per 90 days;
shortest-first ≈2, but it delays long urgent studies (urgent MR p50 99 vs
52 min) and, in a controlled micro case, starves them while short routine
work keeps arriving. No worklist semantics changed; the trade-off is
documented for an owner decision.

**Goldens.** All four regenerated: outcomes unchanged (log, ledger and
action digests identical), the state digest changed (new order and counter
fields in the state walk), and the result JSON changed in walk-in fields
and referral `lost`/`booked`.

| Golden | state digest | run digest |
|---|---|---|
| established-seed1-7d | 8a2a0fd92281… → c4d6b46c4d94… | af8fb442c246… → 7b99a6f7ffc1… |
| established-seed1-90d | 8f7de97d1fc0… → 8c6575d35565… | a25871c771cc… → 12157065771c… |
| mini-seed1-7d | 0db92e2b5226… → f2dff6e4d307… | e3b39b7707d0… → 26adad607323… |
| established-seed1-7d-actions | 08ab88ba9b21… → 72f117978ca0… | 9005416cd60a… → 9a3d8861926c… |

**Tests.** Regressions: `TestRegress18ExtensionMinutesRange`,
`TestRegress18SharedStaffFeasibility`, `TestRegress18WalkInCohorts`,
`TestRegress18FlowCountsFieldsFolded`, `TestRegress18OrderCohortReconciles`,
`TestRegress18UrgentLongStudyVsShortRoutine`. Full uncached suite green.

---

## Step 18 — Calibration: a struggling, capacity-constrained practice (v0.1.0-dev.18)

**Owner decisions (2026-10-14).** Calibrate in order — volume and flow,
reporting, economics, cross-seed — without forcing numbers. When the
measurements showed the margin target conflicts with the documented roster,
the owner chose: the default established practice is **struggling and
capacity-constrained**; documented reimbursement, costs, staffing and
operational assumptions stay; its margin is characterized, not forced; the
6–12% band belongs to a separate better-staffed preset, added only if it
can be configured cleanly (it cannot yet — next scenario milestone); the
mini scenario is unchanged; no hiring, roster editing, new actions or
artificial revenue.

**Method.** A calibration suite behind the `calibration` build tag
(`internal/sim/calib_test.go`, `calib_actions_test.go`): nine seeds, the
180-day warm-up, 30- and 90-day windows from the takeover, daily 07:30
worklist snapshots, per-person utilization, lost orders by reason, lead
times by modality; and a scenario overlay (`DT_OVERLAY`, the scenario
system's own merge) for trying candidate parameters without editing files.
The pre-calibration nine-seed record is
`docs/calibration/baseline-0.1.0-dev.15.json`; the full report with every
table is `docs/calibration/step-18-report.md`.

**Baseline diagnosis** (nine seeds, 90 days; 30-day in brackets):

| | Mean (range) |
|---|---|
| Demand: orders + walk-ins a month | 1,423 + 224 = 1,647 |
| Exams a month | 1,314 (1,293–1,335) [1,381] |
| Tue–Fri visits a day | 59.9: XR 27.0, US 10.0, CT 15.1, MR 7.7 |
| Lost bookings a month | 127: US 64, MR 29, CT 17, XR 17 (almost all "declined": earliest slot beyond tolerance) |
| Walk-ins leaving | ≈100 a month, 45% of walk-ins |
| No-shows / cancellations | 6.5% / 8.9% (as documented) |
| Technologists busy (of duty) | Sam 82%, Morgan 82%, Casey 81%, Riley 76% |
| Radiologists reading (of site blocks) | Avery 73%, Blake 63% |
| Worklist at takeover / 07:30 Tue–Fri | ≈0 / mean 3.3, max 9–18 |
| TAT p50 / p90 / over target | 32 min / 122 min / 5.2% |
| Net revenue, labor, fixed, variable a month | $200.4k, $106.8k, $87.8k, $32.7k |
| Operating margin | −14.0% (−15.6 to −13.0) [−12.5% (−16.9 to −11.1)] |

**Bottlenecks.** Demand matches the design; capacity does not. XR and CT
share technologist time most of the day (Sam and Riley saturated, rooms
idle 60%+): walk-ins register, wait 30–112 min and leave. US is one
technologist with 45-min slots for ~37-min exams (69% busy of staffed time,
the most lost demand, lead p50 19 days). MR is one technologist with nine
slots. Reading is over-provisioned for this volume (hence the empty
evening worklist and fast TAT).

**Experiments** (nine seeds; none applied): XR room times ×0.75 → +27
exams a month; walk-in policy `accept_always` → no change; radiologist
site allocation 45%/38% → unstable reading queue (TAT p50 20 h), 55%/45% →
stable, ≈$5k a month cheaper (not applied: documented staffing kept); three
more technologists → lost bookings −70% but exams +7% and margin −18.7%,
because demand, no-shows and cancellations bind. The design's reference
P&L counted ≈1,650 orders as ≈1,650 exams with technologists ~94% busy:
internally inconsistent with its own roster (now noted in §8.6).

**Parameters changed: none.** No change was defensible under the decision
(each candidate either altered documented staffing or only marginally
moved volume), so the established scenario's fingerprint is unchanged
(`b1cbc2a0…`), and the golden references are unchanged: their inputs and
outcomes did not change (rule: update only when they do). What changed is
the characterization: design §5.1, §6.2, §8.6, ROADMAP step 18 and the
V0.1 calibration criteria (now the better-staffed preset's), the briefing
note, and the tests. The four references still name engine 0.1.0-dev.15; under
0.1.0-dev.18 their outcomes reproduce exactly (`TestGoldenReferences`
compares with the engine difference allowed and reported). The CLI without
`-allow-engine-mismatch` reports them as not comparable (exit 5), as
designed; regenerating them would only restamp the engine.

**Actions** (30 days, nine seeds; demand identical across strategies,
asserted):

| Strategy | Exams | Lost | Margin |
|---|---|---|---|
| A no actions | 1,363 | 141 | −12.5% |
| B US + MR extended to close (approved ahead), walk-ins until 19:00 | 1,431 | 94 | −9.4% |
| C Riley extended (2nd XR/CT evenings), accept all walk-ins | 1,394 | 127 | −13.5% |
| D all three extensions, accept all | 1,460 | 79 | −10.5% |
| E decline walk-ins, FIFO | 1,394 | 178 | −12.6% |
| F shortest-first worklist | 1,363 (TAT p50 22, p90 157 min) | 141 | −12.5% |

Actions matter (B recovers a third of lost bookings; F halves TAT; E turns
walk-ins who would leave into appointments) and every added hour is paid
(117–194 h of approved overtime), but none reaches break-even: profitability
needs hiring and demand growth (V0.4). No loophole found: demand is
unchanged by any strategy, declined walk-ins stay counted, extensions are
costed, worklist ordering uses published read times only.

**Targets** (report table): demand, no-shows, MR utilization cap, US
longest lead and most losses, cash/profit divergence — met. Tue–Fri
visits (60 vs 65–85), XR and MR per day just below bands, US 80–90%
utilization (unreachable with 45-min slots, no overbooking), radiologist
capacity 105–115% (≈145%), 07:30 backlog 5–20 (mean 3.3), routine TAT 1–4 h
(32 min), T30 margin 6–12% — missed by the default and assigned to the
better-staffed preset.

**Tests.** `TestEstablishedCharacterization` (three seeds, 30 days, broad
bands: demand 1,450–1,850 a month, exams 1,250–1,500, margin −20% to −6%,
25–65% of walk-ins leave, US loses most, TAT p50 < 2 h and morning backlog
< 40, books reconcile). The calibration suite is tagged (≈30 s and ≈2 min).
Determinism, replay (step 14), warm-up boundary (step 11), closing rules
(step 8), demand independence and goldens are unchanged and pass:
`go test -count=1 ./...`.

**Limitations and next scenario milestone.**
* The better-staffed `moderate` established practice (6–12%) needs more
  staff and more demand; demand growth is V0.4 (referrals). Until then the
  default is the only established practice.
* The P&L workbook and `docs/ASSUMPTIONS.md` (ROADMAP step 18) move with it.
* Reading capacity exceeds this practice's demand; a struggling owner might
  reduce the site allocation — not a V0.1 action.
* Forty-five percent walk-in abandonment is a consequence of the roster;
  patience (30–120 min) is as documented.

---

## Step 15 — Canonical results and golden references (v0.1.0-dev.15)

**Owner decisions (2026-10-14).** Engine-version compatibility stays:
replay and golden comparison against another engine only with
`-allow-engine-mismatch`, and compatibility is always reported apart from
verification. Golden outputs are canonical JSON for machines plus a concise
text summary for people. Golden files hold observable and financial
outcomes and digests, never individual records or hidden state.

**Result schema** (`dreamteam.result/v1`, SIMULATION_DESIGN §4.5):

| Section | Contents | Kind |
|---|---|---|
| identity | schema, engine, scenario ID and fingerprint, seed, epoch date | — |
| `takeover`, `end`, `playDays` | simulation seconds and ISO time | times |
| `actions` | attempted / accepted / rejected, history digest, every result in order (rejections and no-ops included) | history |
| `digests` | log, ledger, actions, state, run | verification |
| `calendar` | each completed day, midnight to midnight (the takeover day flagged): orders, lost, walk-ins, no-shows, exams, reports, unread at close, wait p50, overtime, net revenue, expenses, profit, collections, disbursements, end-of-day cash, alerts | calendar flows (+ end-of-day cash) |
| `window` | from the exact takeover to the end: baseline digest, period metrics (demand, appointments, walk-ins, waits, rooms, staff, reporting, workforce, alert and action counts), window money (baseline-adjusted and as posted, cash bridge) | evaluation-window flows |
| `balances` | cash, A/R, A/P, payroll owed, equity, lowest cash, days below zero | point in time |
| `obligations` | next payroll and bills, open claims and aging, unread / reading, recall list, booked ahead, policies, approved extensions | point in time |
| `alerts` | raised in the window by severity; active at the end | counts |
| `replay` | verified, batches, replayed run digest (with `-verify-replay`) | verification |

Integers only; ratios as numerator and denominator; fixed field order;
arrays in defined order; no wall-clock time, paths or locale. Every field
is compared.

**CLI** (no new execution path: `run` and `replay` gained output flags):

    dreamteam run -scenario established -seed 1 -days 7 -format json            # canonical JSON on stdout
    dreamteam run -days 7 -actions FILE -verify-replay                           # text summary, replay checked
    dreamteam run -days 90 -verify-replay -format json -golden testdata/golden/established-seed1-90d.json
    dreamteam run -days 7 -format json -write-golden FILE [-overwrite]           # explicit, never silent
    dreamteam replay SESSION -format json [-golden FILE]                         # the replayed run's result
    dreamteam verify SESSION                                                     # "ok <run digest>"

Diagnostics go to stderr; with `-format json` stdout holds only the JSON.
Exit codes: 0 ok · 1 error (I/O, scenario) · 2 invalid input · 3 replay
failure · 4 golden mismatch · 5 incompatible or unsupported format (another
schema, engine, scenario, seed or run length; malformed golden; unknown
`-format`). A mismatch report gives compatibility, the number of
differences, the digests that differ and the first differing paths with
both values, e.g. `first: .window.ops.exams: golden 1, run 184`.

**Reference runs** (`testdata/golden/`, README there; each generated with
`-verify-replay`, aggregate outcomes plus digests, 16–52 KB):

| Reference | Exams | Orders / lost | Net revenue | Op. expense | Op. profit | Cash end | Run digest |
|---|---|---|---|---|---|---|---|
| established, seed 1, 7 days | 292 | 329 / 33 | $47,037 | $55,712 | −$8,675 | $144,817 | `af8fb442c246…` |
| established, seed 1, 90 days | 3,887 | 4,171 / 370 | $593,096 | $675,797 | −$82,702 | $16,985 | `a25871c771cc…` |
| mini, seed 1, 7 days | 184 | 360 / 4 | $27,893 | $50,784 | −$22,891 | $137,965 | `e3b39b7707d0…` |
| established, seed 1, 7 days, actions (8: 5 accepted incl. a no-op, 3 rejected) | 294 | 329 / 32 | $47,452 | $55,899 | −$8,447 | $144,707 | `9005416cd60a…` |

(Uncalibrated: the losses are the step 10–11 volume shortfall that step 18
calibration addresses. The 90-day run ends with cash near zero.)

**Tests** (`go test -count=1 ./...` passes):

| Requirement | Test |
|---|---|
| References reproduce (byte-identical when the engine matches) | `TestGoldenReferences` (all four, `-verify-replay`) |
| Repeated runs byte-identical; seeds and scenarios differ | `TestResultsRepeatableAndDistinct` |
| Meaningful changes, missing / unexpected fields, digests, calendar length, engine, fingerprint, seed, run length, schema, malformed / truncated / trailing files, CLI exit codes, no silent overwrite, unsupported format | `TestGoldenComparisonReports` |
| Action history changes detected; rejected actions represented | `TestActionHistoryInResult` |
| Chunk sizes and playback speeds (one jump, 37-min chunks, Max-speed event budgets, alert stops) | `TestResultIndependentOfChunkingAndSpeed` |
| No hidden data | `TestResultHasNoHiddenData` (no record IDs, no latent or queue fields) |
| Financial reconciliation, window boundaries | `TestResultReconciles` (balance sheet, daily cash roll-forward, cash bridge, window = takeover → end, calendar days = play days) |

**Cross-platform.** All four references reproduced byte-identically on a
second Linux arm64 machine (Cortex-A76, via the built binary) as well as
the Raspberry Pi 4 (Cortex-A72) that wrote them. Linux amd64 and macOS
arm64 have **not** been run (no such machine or emulator here); the
procedure is in `testdata/golden/README.md`.

**Performance** (MEASUREMENTS.md): `run -days 7` 1.9 s, `-days 90` 3.1–4.7
s (warm-up included); result build and encode after 90 days 0.54–0.62 s
(mostly the state digest); comparison 18–25 ms. Not in any output.

**Limitations.**
* An engine version bump requires `-allow-engine-mismatch` for old
  references (reported); regenerate to make them byte-identical again.
* Per-day calendar rows make the 90-day file 52 KB; per-patient detail is
  deliberately absent, so a digest-only change (e.g. the order of two
  same-day events) is reported as differing digests without a metric path.
* No amd64 or macOS verification yet.

---

## Step 14 remediation — replay review findings (v0.1.0-dev.14.1)

A focused review of replay correctness (`docs/reviews/step-14-astra-review.md`,
gpt-6-astra at high reasoning, read-only snapshot of `16c53a2`) returned
NO GO with eight findings; all were confirmed, each with a regression test
that fails on `16c53a2` (finding 7 by inspection: no injection point
existed) and passes after the smallest fix. No simulation outcome changed.
The suspected pause-then-act mismatch was ruled out.

| # | Finding | Fix |
|---|---|---|
| 1 (high) | Hour keys like `"+8"` were accepted beside `"08"`: compilation depended on map order | only canonical `"00"`–`"23"` |
| 2 (high) | The run digest did not bind the scenario: future rules could differ under the same digest | run digest includes the scenario fingerprint and seed |
| 3 (medium) | Sessions kept the caller's scenario object: later edits broke saves | private copy |
| 4 (medium) | The state digest included the last-transaction time cache | excluded |
| 5 (medium) | Trailing `]`/`}` after a record, data after a scenario, a second actions array accepted | EOF required |
| 6 (medium) | Re-signed edits to batch numbers, times, scenario ID/name, start/until and summary verified | all checked against what they describe |
| 7 (medium) | Directory-sync failure after rename reported as a durable save | reported; session stays unsaved |
| 8 (low) | Refusals printed "OK … digests match"; mismatch summaries were zeros | OK only when fully checked; real replay position reported |

Records saved by `16c53a2` no longer verify (the run digest changed); they
replay with `-allow-engine-mismatch`. Remaining test gaps are listed in the
review document.

---

## Step 14 — Sessions, replay and digests (v0.1.0-dev.14)

### Part A — playtest (before step 14)

Driven through the real TUI model (scripted keys, every screen read at
80×24) on the established practice, seed 1:

1. **Two days, no actions.** Alerts, queues, worklist and money agree:
   day 1 70 exams, 16 alerts (mostly XR waits behind the shared XR/CT
   technologist), 3h 01m overtime, profit $575; day 2 60 exams, an insight
   flags MR busy 86% of staffed time but staffed 70% of open hours. The
   17:00 dashboard shows self-registration after the desk staff leave;
   "MR room unstaffed — why: Casey on break until 13:00" matches the roster.
2. **US extension a week ahead** (Morgan, Wed Oct 21, +2h): the schedule
   for that day went from 2 to 5 open US slots (18:00, 18:45, 19:30), at
   once. Demand untouched (tests).
3. **Walk-ins: decline, then accept-if-gap.** The 3 walk-ins arriving
   while declining were offered appointments and counted as declined;
   nobody already waiting was removed; switching back took effect for the
   next arrival.
4. **Worklist: priority → FIFO** with an urgent and two routine studies
   unread: the urgent XR dropped from first to last at once; MR and US were
   read first, the urgent one at 38 min; the read in progress continued.

**Defects found and fixed (all in action feedback):**
* An extension's preview did not say what the extra time covers, and an
  extension that would cover nothing was accepted. Now each says
  "staffs US 18:00–20:00", "covers the desk …" or "on site for contrast
  supervision (reading blocks unchanged)", and one covering no room, desk
  or supervision is refused (`no_coverage`). Coverage is evaluated on the
  extended plan, the same plan dispatch and booking use, so the preview,
  the schedule and the effect agree (`TestExtensionWithoutCoverageRefused`).
* It did not say the benefit: it now names the slots it opens ("opens 3
  US slots for booking") or says none would be ("already staffed, or no
  template slots then" — e.g. Riley's XR evening, which Sam covers).
* At 80 columns the preview and result were cut off (the cost and effect
  lost) and dates were ISO. They now wrap over up to three lines ("Done ✓
  …", "If submitted: ✓/✗ …"), dates read "Wed Oct 21", and the next preview
  restarts at +60m instead of showing a confusing rejection.
* Declined walk-ins were cut off the 80-column dashboard: now "walk-ins 3
  (3 declined)".

Owner note: the five-day comparison's +$181 does not make the policy
optimal — 22 h of approved overtime for 8 exams may reverse once step 18
calibrates rates and staffing. That trade-off is for players and agents to
discover.

### Part B — sessions, replay, digests

**Principle.** A saved run is reproduced from its scenario, seed and
timestamped actions; no serialized world is trusted. The session file
(`dreamteam.session/v1`) is a run record: the fully resolved scenario with
its fingerprint, the seed, epoch and takeover, every action batch with its
results (rejections and no-ops too), the saved time, five digests, a
readable summary and a checksum. Format and rules: SIMULATION_DESIGN §4.5
(as built).

| Digest | Covers |
|---|---|
| log | every observable log entry, in order |
| ledger | every journal entry, in order |
| actions | the action record with results |
| state | the whole authoritative state, hidden fields included, and the live pending events; not caches, hooks, processing counters or event sequence numbers |
| run | all of the above and the clock |

**Load = replay.** Strict parse → checksum → schema → scenario fingerprint
→ batch structure → compile, warm up, apply every batch at its time
through `Apply`, compare each result → run to the saved time → compare the
digests. Mismatches name the first differing action (batch, time, action,
recorded vs replayed) or the differing digests, with both summaries.

**Compatibility.** Another schema: refused. Another engine version:
refused unless `-allow-engine-mismatch`, which replays and lets the digests
say whether outcomes agree. Changing the rules (any future gameplay step)
therefore changes which old saves verify: that is the point.

**Changes to existing behavior (no outcome changes).**
* Alerts keep the text they were raised (or escalated) with; before, the
  stored text was refreshed at each evaluation, so a *resolved* alert's
  displayed text depended on when unrelated events happened. Observations
  of active alerts still show the live text.
* A batch of only rejected or no-op actions no longer moves the metrics'
  accrual point or the clock cache; an empty batch is not a batch.
* `Session.Digest()` is now the run digest (log, ledger, actions, state,
  clock).
* Saved and loaded sessions without actions are identical to `08ccc5d`
  (ledger and every visit, study and appointment, established 21 days and
  mini 30 days, seeds 1–3).

**Controls.**
* TUI: `ctrl+s` save, `ctrl+o` reload (replay the saved file and resume),
  `ctrl+v` verify by replay; the top bar shows the seed and "● unsaved".
* CLI:
  * `dreamteam play [-load FILE] [-save FILE]`
  * `dreamteam run [-scenario ID] [-seed N] -days D [-actions FILE] [-save FILE]`
    — headless, prints each action result, the since-takeover figures and
    the digests. Actions file: `[{"time": "2026-10-13T07:00:00", "actions": [...]}]`.
  * `dreamteam replay FILE [-allow-engine-mismatch]` — full report.
  * `dreamteam verify FILE` — `ok <run digest>` or the mismatch, exit 1.

**Tests** (`go test -count=1 ./...` passes; `internal/sim` 158 s,
`internal/session` 21 s):

| Requirement | Test |
|---|---|
| Save/load round trip, resume | `TestSaveLoadRoundTrip` (accepted, rejected, no-op, two multi-action batches, a future-day extension; both sessions continue identically) |
| Replay after restart | `TestRunSaveReplayVerify` (CLI: separate run, file, replay, verify) |
| Midday takeover, cross-day and cross-week, payroll and claims across resume | `TestReplayMiddayTakeoverAcrossPayrollAndWeek` (13:00 takeover, saved Thursday 23:00, continued past Friday payroll and remittances to the next Tuesday) |
| Corrupt, truncated, edited, fingerprint, schema, engine, tampered results and digests, order | `TestCorruptAndIncompatibleFiles` (11 cases) |
| Atomic save failure | `TestAtomicSaveFailure` |
| Chunking | `TestReplayChunkingInvariance` (17 min, 3 h, single step) |
| Independence from unrelated events, reads, previews, rejections | `TestStateDigestIndependentOfNoiseAndReads` |
| Hidden state covered, nothing leaked | `TestStateDigestCoversHiddenState`, `TestSavingHasNoSideEffects` (no latent fields in the file; saving and verifying change nothing) |
| Determinism | `TestStateDigestDeterministic`, `TestCanonicalRoundTrip`, `TestParseISORoundTrip` |
| TUI | `TestSessionKeys` |

**Performance** (MEASUREMENTS.md): warm-up unchanged (1415 ms vs 1359 ms
best, within thermal noise; identical memory); save after 30 days
0.44–0.54 s; load = replay + verify of 30 days 2.1–2.3 s; `run -days 30`
plus save 4.0 s and 48 MB peak; file 35 KB.

**Limitations.**
* No snapshots: loading always replays from the epoch (warm-up included),
  about 1.4 s plus the played days. A snapshot cache verified against the
  digests can come with V0.2 save/load UI.
* The state digest is a reflection walk over the state's fields: a field
  added later is covered automatically, but a new cache must be added to
  its skip list (or the noise test fails, which is how this one was found).
* Records include the seed: fine for game saves; benchmark records will
  escrow it (V0.5).
* Golden 7- and 90-day digests (ROADMAP testing strategy) are not pinned
  yet; the start-state goldens are.

---

## Step 13 — Player actions (v0.1.0-dev.13)

**Owner decisions recorded (2026-10-14).** The four step 12 conventions are
approved. For #3, the window's operating result excluding inherited-claim
collection variance is a *baseline-adjusted* evaluation convention: the
since-takeover screen labels it "baseline-adjusted" and shows the ledger
basis ("as posted (the accrual P&L)") next to it; the calendar statements
and `WindowMoney.Ledger` keep the actual collection variance. A write-down
of an inherited receivable after takeover is visible there.

**What the player can do.** `a` opens the actions screen (auto-pause moved
to `A`): extend a shift today or on a day up to two weeks ahead (`[` `]`),
set the walk-in policy, set the worklist policy. Each row is previewed
live; `enter` submits; results and the latest actions are listed. The
dashboard shows the policies in force and approved extensions.

**One interface.** `World.Apply` / `Session.Apply(actions...)` is the only
way to change the simulation; `Validate` previews without effect. The TUI
calls the same functions an agent will. Batches apply at the current clock
between transactions, in order, each action against the state the previous
left; a rejected action changes nothing; one dispatch and the alert
observer run after a batch that changed something, none after rejections or
no-ops. Every attempt is recorded with a stable ID (`X00001`), batch
number, time, action, result code and message (`ActionLog`,
`Observation.Actions`, and per-period counts in the metrics and the
evaluation window). Schema, codes and rules: SIMULATION_DESIGN §9.5 (as
built).

| Action | Parameters | Effect |
|---|---|---|
| `extend_shift` | `employee`, `minutes` 1–240, `date` (optional; today to the booking horizon) | the scheduled end moves later that day; they stay on regular duty; the day's slots in the extra time become bookable at once; time past the rostered end is approved overtime |
| `set_walkin_policy` | `policy`: `accept_always`, `accept_if_gap`, `accept_until` (+ `cutoff` HH:MM), `schedule_later` | walk-ins arriving from now on; declined ones are offered appointments, counted apart |
| `set_worklist_policy` | `policy`: `urgent_then_fifo`, `fifo`, `oldest_first`, `shortest_first` | the order unread studies are offered; reads in progress continue |

**Extension vs hold-over** (the design issue you raised). Required hold-over
finishes patients already admitted; it starts past the scheduled end and
only while someone needs the person. An approved extension moves the
scheduled end, so the person keeps staffing their room on regular duty: new
capacity for waiting patients, walk-ins and bookings. Approved and required
overtime are recorded apart (`ApprovedOvertimeSec`/`…CostCents` in the flow
counters, staff metrics, workforce metrics and the window). Hard limits stay
hard: the 12-hour cap (rejection says how much is left), no retroactive
approval (`shift_ended`, `past_date`), no extension that would only add
time after scheduled close (`after_close`: hold-over covers that), breaks
and reading blocks unchanged, nobody else extended.

To give extensions something to fill, the established roster's US and MR
templates now continue to the evening (US 18:00–20:00, MR 16:30–19:30).
Those slots are unstaffed — not bookable — unless a shift is extended; runs
without actions are identical to step 12 (checked below).

**Did the decisions matter?** Same seed (1), established practice, 5 days
from takeover, no actions vs. a plan approved at takeover: US and MR
extended to the 20:00 close on every weekday, walk-ins accepted until
19:00 (`TestInterventionsChangeOutcomesNotDemand`):

| 5 days from takeover | no actions | interventions |
|---|---|---|
| orders received / walk-ins arrived | 254 / 39 | 254 / 39 (identical, same attributes) |
| exams | 259 | 267 |
| orders lost | 27 | 22 |
| US staffed / busy, of open hours | 57% / 34% | 71% / 40% |
| MR staffed / busy, of open hours | 58% / 48% | 82% / 52% |
| arrival→exam wait p90 | 52 min | 51 min |
| overtime (of it approved) | 6 h (0) | 28 h (22) |
| net revenue / expenses | $40,558 / $43,094 | $42,641 / $44,997 |
| operating profit (baseline-adjusted) | −$2,536 | −$2,355 |

The first version only extended shifts on the day itself: US and MR were
staffed, but almost nothing was booked into the new time, because bookings
are made days ahead (+4 exams for 22 h of overtime; profit fell to −$3,488).
Hence the `date` parameter: approved ahead, the slots open for booking now.
Uncalibrated figures; the point is that management decisions now change
operations and money while demand stays exogenous.

**Tests** (`go test ./...` passes; `internal/sim` 149 s):

| Requirement | Test |
|---|---|
| Valid and invalid extensions, cap | `TestExtendShiftValidation` (16 cases), `TestExtensionAheadOpensBookings` (dates, horizon, Sunday, cap) |
| Extension vs hold-over; payroll | `TestExtensionVersusHoldOver` (36 min required vs 60 approved; one wage entry = hourly pay of on-duty time; approved cost = overtime cost) |
| Capacity, not demand | `TestExtensionAddsCapacityNotDemand`, `TestExtensionAheadOpensBookings` (evening US slots booked ahead), `TestInterventionsChangeOutcomesNotDemand` |
| Walk-in policies, declines measurable, not retroactive | `TestWalkInPolicies`, `TestWalkInPolicyNotRetroactive` |
| Worklist ordering, read times, no interruption, no double read | `TestWorklistPolicyOrdering` (shortest-first follows the published p50 only) |
| Simultaneous actions | `TestSimultaneousActionsOrdered` (order-dependent results; intermediate policies inert) |
| Invalid-action atomicity | `TestInvalidActionAtomicity` |
| Replay | `TestActionReplayEquivalence` (re-applying the record reproduces log, books and results) |
| No retroactive effects | `TestActionsNotRetroactive` |
| Chunking, unrelated events | `TestActionsChunkingAndNoise` |
| Evaluation window | `TestEvaluationAttributesActions` |
| TUI | `TestActionsScreen` (submit, reject, policy, ahead; 80×24, 120×36) |

Without actions, the ledger and every visit, study and appointment are
identical to `4bbd03f` (established 21 days, mini 30 days, seeds 1–3), and
the pinned start-state digests are unchanged.

**Performance** (MEASUREMENTS.md): warm-up 1402 ms vs 1401–1434 ms (best
runs; the Pi alternates between ~1.4 s and ~2.5 s with throttling); one
preview 4 µs, one applied action with its dispatch 12–14 µs; 30 days after
takeover 405 ms without actions, 380 ms with daily interventions.

**Known limitations.**
* Extensions are whole-shift-end only: no early starts, no extra breaks
  for long shifts (the cap bounds them), cover windows keep their times,
  radiologists' reading blocks are not extended (their supervision is).
* A cost estimate ahead uses the base rate; the ×1.5 depends on the
  week's hours when worked.
* No undo: an approved extension stays (V0.2 `send_home`).
* The evening slots exist only for US and MR in the default roster.
* Rejected actions are in the action record, not the event log.
* Run records and replay files are step 14.

---

## Step 12 — Metrics, alerts, daily summaries (v0.1.0-dev.12)

**What the player sees.** The dashboard answers "what is going wrong, and
why": an **ALERTS** panel (active alerts, most severe first, time raised,
the screen that shows more, and a "why:" line built from observed facts —
"shared technologist Riley busy (documenting P00031)", "no XR technologist
rostered until 09:30") and a **Today** line (time-weighted utilization of
staffed hours by room, arrival→exam wait p50/p90, lost orders). `e` runs
today's operations to operational close and opens the **end-of-day
summary** (TUI_DESIGN §4.10): patients, demand, schedule, waits, room time
three ways (busy of open, busy of staffed, staffed of open), staff,
reporting, money, cash, alerts and rule-based insights; `←→` pages earlier
days, `tab` shows **since takeover**. It fits 80×24 untruncated (short
forms below 110 columns). The history and briefing screens now draw on the
same measurements (room busy time within open hours, so ≤ 100%).

**Reporting windows (owner decision, 2026-10-14).** Two windows:

| Window | Purpose | Definition |
|---|---|---|
| Calendar | TUI, daily history, statements | midnight to midnight, including the takeover day's activity before the takeover |
| Evaluation | future player/agent evaluation | from the exact takeover instant to `evaluationDays` (scenario; 0 = open-ended) |

The baseline is captured at the end of the takeover transaction, after the
opening adjustment, so everything at that instant belongs to the inherited
state and the window is (takeover, end]. It is immutable and digested:
balances and trial balance, open claims by state, unreported studies,
booked appointments, recall list, patients on site, staff on duty, staffed
rooms, ledger and log positions. Window money is the journal after the
baseline (a second day book), with operating performance separated from
cash flow and working capital and a cash bridge that reconciles to the cent.
No weighted score, weights or thresholds: measurements only.

**Metric definitions** (full table: SIMULATION_DESIGN §11.1). One
cumulative tally integrates the state between transactions; a period is the
difference of two snapshots (midnight, takeover, endpoint, or now).

* Rooms: open (scheduled hours); staffed (open with a qualified technologist
  on duty; a shared technologist counts fully for each room); busy (patient
  in the room, within open hours); after-hours busy apart. Utilization of
  open hours = busy/open, of staffed hours = busy/staffed, coverage =
  staffed/open: none can exceed 100%.
* Staff: on duty (paid time, breaks out, overtime in), busy (registration,
  exam, documentation, reading), break, overtime. Technologist utilization =
  busy/on duty per person (shared time counted once). Radiologist reading
  utilization = reads inside site-reading blocks / block time; overrun reads
  apart.
* Appointments (slots starting in the period): fill = booked/bookable slot
  time; no-show rate = no-shows (missed included until accounting close) /
  (scheduled − cancelled); cancellation rate = cancelled/scheduled.
* Demand: received, booked, recall, lost, cancelled; lost rate =
  lost/received. Walk-ins: acceptance = imaged/arrived, abandonment =
  left/arrived.
* Waits (exams started in the period): arrival→registration,
  arrival→exam, exam start past the appointment time.
* Reporting: TAT of studies imaged in the period, unfinished ones censored
  at the period end; TAT of reports finalized (inherited ones counted);
  breaches; backlog at both ends.
* Workforce: overtime and its cost, held-over, skipped breaks, coverage
  reschedules, patient-time waiting to register with no desk staff.
* Ratios keep numerator and denominator (an empty denominator is "—", never
  0%); percentiles are nearest-rank everywhere (the worklist's median and
  p90 now use the same definition).

**Alert behavior.** Ten rules over observable state, from the takeover on:
waits (30 min warn, 60 min alert; measured past the appointment time, or
since arrival for walk-ins and patients fitted in after grace), unstaffed
rooms with patients on site, report TAT overdue (urgent alert, routine
warn, soon tracked since each breach is already logged), held-over staff
(tracked) and the 12-hour cap approaching (alert), closing running late (30
min warn, 60 alert), reporting backlog (≥ 40, clears below 32), lost demand
(≥ 3 per modality per day, expires at midnight), cash low (< $25,000) and
negative (alert; replaces the step 11 shortfall entry, same text). One
raise per episode, at most one escalation, a logged resolution after a
10-minute clear for waits, rooms and backlog. Time-driven changes get one
exact passive wake (no polling); passive events never trigger dispatch, so
they cannot change outcomes. Seed 1 raised 16 alerts on day 1 and 11 on
day 2 (escalations included), mostly waits behind the shared XR/CT
technologist and short unstaffed spells during breaks, plus one late close. Thresholds are configurable (`alerts` in
the scenario) and uncalibrated.

**Conventions to confirm (owner).**
1. Events at the exact takeover instant (midnight payroll or bills, 10:00
   remittances, dispatch) belong to the inherited state, consistent with the
   step 11 opening rule; the window is (takeover, end].
2. The takeover day's midnight accruals (salaries, fixed costs) cover the
   whole day; the window bears the share after the takeover (18/24 at
   06:00), shown as "takeover-day accruals" and in the cash bridge.
3. Collection variance on inherited claims is excluded from window
   operating profit (reported apart); billing fees on collecting them stay
   window expenses (the fee is a cost incurred in the window).
4. Wait alerts measure from the appointment time (an early arrival waiting
   for a punctual exam is not a problem); the wait metrics report both.

**Outcome neutrality.** Metrics, alerts and summaries change nothing in
the simulation. Checked against `23b6ff8`: ledger digest and every visit,
study and appointment identical after 21 days (established) and 30 days
(mini), seeds 1–3; the pinned start-state digests (ledger and log) are
unchanged. The log gains the alert entries.

**Tests** (`go test ./...`: all packages pass; `internal/sim` 157 s on the
Pi). New, on hand-built micro practices with fixed durations where figures
are worked out by hand, and on the mini and established practices:

| Requirement | Test |
|---|---|
| Exact utilization denominators | `TestMetricsRoomAndStaffDenominators` (XR busy 46 min of 60 open, 34 min after hours; Tess 2h06m on duty, 36m overtime, 96m busy; an unused staffed room is 0%, not undefined) |
| Shared technologist | `TestMetricsSharedTechnologist` |
| Staffed vs open | `TestMetricsStaffedVersusOpen` (staffed = open − breaks) |
| Reading blocks | `TestMetricsReadingBlocks` (and TAT percentiles, censored) |
| Percentiles, empty denominators | `TestNearestRankPercentiles`, `TestMetricsEmptyDenominators`, wait percentiles in the denominators test |
| Partial day, midnight boundary | `TestMetricsPartialDay`, `TestMetricsMidnightBoundary` (only the censored TAT ages, by one second, from 23:59:59 to midnight) |
| Exact-timestamp boundary | `TestEvaluationExactTakeoverBoundary` (10:00 takeover with remittances at that instant; window books = journal after the baseline; orders before 10:00 in the calendar day only) |
| Inherited claims paid after takeover | `TestEvaluationInheritedClaims` |
| Imaged before, finalized after | `TestEvaluationPreTakeoverStudyFinalizedAfter` |
| Capital excluded | `TestEvaluationExcludesCapitalTransfers` (+$350,000 opening cash: identical operations and operating results) |
| Accrual allocation, whole-day window | `TestEvaluationAllocatesTakeoverDayAccruals`, `TestWindowEqualsDaysFromMidnightTakeover` |
| Alerts: trigger, escalate, resolve, dedupe, expire | `TestAlertsTriggerEscalateResolve` (exact times 08:20, 08:50, 09:20, 09:40, 09:43), `TestAlertsExpireAndDeduplicate` |
| Same-instant ordering | `TestAlertSameInstantOrdering` (an exam starting at the threshold instant raises nothing) |
| Daily summary reconciliation | `TestDailySummaryReconciliation` (days sum to since-day-1; cash rolls forward) |
| No future leakage | `TestMetricsAndAlertsRevealNoFuture` |
| Chunking, speed, unrelated events | `TestMetricsChunkingInvariance`; the TUI pacing test; `TestRegressOutcomeIndependentOfUnrelatedEvents` now includes both alert fixtures |
| Session, TUI | `TestEndOfDayStopsAtOperationalClose`, `TestEndOfDaySummary`, `TestDashboardAlerts`; render tests cover finance and history |
| Scenario fields | `TestValidationErrors`: 4 new cases (`evaluationDays`, `alerts` ranges and ordering) |

**Performance** (MEASUREMENTS.md): warm-up 1402 ms vs 1366 ms at `23b6ff8`
(best of four alternating runs, Pi at 81 °C, +2.6%); one observation 1.26
ms vs 1.20 ms; the evaluation window 1.3 ms, on request only. Completed
days' summaries are cached.

**Known limitations.**
* All overtime is required hold-over until actions arrive (step 13); the
  approved/required split is not yet reported.
* Alerts start at the takeover: history days have none.
* Thresholds, insight rules and the backlog limit are illustrative
  (calibration, step 18). Insights are two simple rules.
* A day appears in the history from midnight (before: from accounting
  close); after accounting close today's summary is still "so far".
* The baseline digest covers the baseline record, not the full state (the
  full-state digest is step 14). Scenarios leave `evaluationDays` at 0.
* No turnover state in V0.1, so busy = patient in the room.
* The history screen did not gain alert or wait-percentile columns (the
  summary screen pages through days instead).

---

## Step 11 remediation — review findings (v0.1.0-dev.11.1)

The step 11 review (`docs/reviews/step-11-astra-review.md`, gpt-6-astra at
high reasoning, read-only snapshot of `e5863b4`) returned a CONDITIONAL GO
with six confirmed defects and two test gaps. For each defect a regression
test was written first and run on the unfixed code (`e5863b4`); then the
smallest fix was made and the test rerun.

| # | Defect | Regression test | Before (on `e5863b4`) | After | Fix |
|---|---|---|---|---|---|
| 1 (high) | The takeover adjustment ran before same-instant events: a later-queued day start (payroll or bills) at a midnight takeover, 10:00 remittances, and dispatch. Opening cash came out wrong and the snapshot was taken too early. | `TestRegress11TakeoverCollisions` | FAIL: 5 operating entries after the opening at midnight on payday Fri Oct 16 and on Nov 1; 87 at a 10:00 takeover. Opening cash $100,801 (payroll paid after the injection), $27,142 (bills) and $156,239 (collections) instead of $150,000. | PASS: exactly $150,000; snapshot = books when play begins; the opening is the last entry of the instant | The takeover event only marks the adjustment due. It is posted at the end of that instant's dispatch, after every event and dispatch decision. |
| 2 (medium) | The cash chart and daily records showed pro forma (shifted) history as plain cash | `TestRegress11ActualHistoricalCash`, `TestRegress11CashHistoryLabelled` | FAIL: cash history Sep 14 $195,837, actual −$372,992; screens unlabelled | PASS | Actual cash everywhere by default. `DaySummary.ProFormaCashEndCents` is a separate, named field. The history screen says ACTUAL CASH and shows a takeover line ("owner capital +$568,829 … brought cash from −$418,829 to $150,000"). The chart says "Cash (actual)" and notes the capital at takeover. The ledger is untouched. |
| 3 (medium) | `submitTime`/`paymentTime` of 24:00 accepted, scheduling onto the next (weekend) day | `TestValidationErrors` (2 new cases) | FAIL: no validation error | PASS | Point-in-time financial clocks must be before 24:00. |
| 4 (low) | Ledger hash ignored the fixed-cost item | `TestRegress11LedgerHashCoversItem` | FAIL: same hash | PASS | The item is part of each hashed line. The golden ledger digest changed for this reason only; the log digest is unchanged. |
| 5 (low) | The balance line printed "=" between independently rounded dollars (the reported $1) | `TestRegress11BalanceEquationExact` | FAIL: whole dollars | PASS | The balance and equity lines show exact cents: $428,153.55 = $88,071.43 + $340,082.12. |
| 6 (low) | Dates outside years 0–9999 wrapped (10000 → 0000) | `TestRegress11DateFormattingAllYears` | FAIL: "0000-01-01" for 10000, "000/-12-31" for −1 | PASS | The fast path covers 0–9999; others use the previous `fmt` format. |

**The $1 discrepancy.** It was display rounding. The exact balances at
takeover (seed 1) were assets 42,815,355¢, liabilities 8,807,143¢ and
equity 34,008,212¢: an exact match in cents. The ledger also enforces
Σ = 0 on every posting and in the invariants.

**Test gaps closed**
* `TestRegress11SplitEqualsWholePendingState`: building then playing 2 days
  7 h equals one uninterrupted run. It compares the full pending event queue
  and every claim, appointment, study, visit and order, not only digests.
  This passed before as well.
* `TestRegress11ConvergenceRejectsNaN`: the convergence comparator now
  rejects non-finite ratios (0/0 used to pass). It failed before.
* `TestDrawMemoEqualsFreshHash`: memoized draws equal fresh hashes, with
  independent cursors.
* `TestLogDigestIndependentOfReadsAndEviction`: the hash covers every entry
  regardless of reads and ring eviction, and retained order is kept.
* The last two passed before; they confirm the review's finding that the
  optimizations are sound.

**Not changed (recorded):**
* The convergence test remains a nine-seed smoke test. It has no variance
  estimate, the seeds are unpaired across warm-up lengths, and liabilities
  and backlog are not compared. Step 18 measures variance.
* The review's other suggested fixtures remain future work: takeover during
  required overtime; a recall, no-show rebook and stale arrival at
  takeover; one claim in every state; an overnight TAT in history.

**Owner decisions**
1. **Capital stays explicit** (owner, 2026-10-14). The takeover injection
   is shown wherever opening cash is (briefing, log, finance screen,
   history line) and is never treated as a balancing mechanism. Calibration
   is not assumed to shrink it.
2. **Historical cash** (owner, 2026-10-14): actual by default with the
   injection visible. Pro forma only as an explicitly named value. Done.
3. **Scoring windows (open).** "Today" and "since day 1" are calendar-day
   windows from 00:00. A 06:00 takeover includes the 00:00 accruals; a
   13:00 takeover includes the morning. That is fine for display. For
   benchmark scoring (step 12 onward) the recommendation is to score
   post-takeover performance from the exact takeover instant against the
   opening balance sheet, and to keep capital transfers out of operating
   performance.

Verification: the full suite passes, and all step 8 and step 10
regressions pass.

---

## Step 11 — Warm-up, opening state and performance (v0.1.0-dev.11)

**What was built.**
* `dreamteam` now opens the **established practice** by default (bundled
  scenario `established`; `mini` remains).
* It runs the documented 180-day warm-up with the ordinary engine before
  the TUI starts ("Building 180 days of practice history…") and opens on a
  takeover briefing at Tue Oct 13 06:00, day 1.
* New code:
  * `internal/sim/opening.go`: the opening adjustment and snapshot, and
    financing-shortfall tracking;
  * `history.go`: daily records;
  * `draws.go`: the draw memo;
  * `internal/tui/briefing.go`: the briefing and history screens.
* Design: SIMULATION_DESIGN §8.8; TUI_DESIGN §4.7a.

**Warm-up semantics.** Warm-up and play are one continuous run of the same
engine. Nothing is approximated, skipped or reset. These carry over as the
history left them:
* appointments (868 booked from day 1 for seed 1) and the recall list
  (48);
* pending orders and patients on site;
* studies on the worklist and reads in progress;
* claims at every stage, receivables, payables and accrued payroll;
* staff weekly hours, for overtime;
* ID counters.

**Warm-up accounting conventions**
* Cash is 0 at the epoch, and the history's collections, payroll and bills
  accumulate.
* At the start time one journal entry, the **opening equity adjustment**
  (Dr/Cr Cash ↔ Opening equity), sets cash to the scenario's $150,000.
  This is the documented convention (§8.4): owner capital in, or a
  distribution out. It is not revenue or a cash flow.
* Nothing else changes. A/R, A/P, accrued payroll and every open claim
  continue, and future payment events stay queued. The history's operating
  result stays in equity, so losses are not erased.
* Seed 1 at takeover:

| | |
|---|---|
| Cash left by the history | −$418,829 |
| Owner capital added | $568,829 |
| Opening cash | $150,000 |
| Receivables (1,870 open claims) | $278,154 |
| Payables | $48,933 |
| Payroll owed | $39,139 |
| History result (180 days) | −$223,758 |
| Last 30 days: net revenue / expenses / profit (margin) | $192,344 / $222,591 / −$30,247 (−15.7%) |
| Balance sheet at takeover | assets $428,154 = liabilities $88,071 + equity $340,082 |

* The capital is large because the uncalibrated history loses money.
  Alternatives are listed under the decisions below.
* **Revenue recognized vs net revenue.** Revenue recognized is the expected
  net booked when each report is finalized. Net revenue adds the collection
  variance on claims that settled in the period. In step 10's 180-day run
  net revenue ($1,136,003) exceeded revenue recognized ($1,125,060) because
  the claims settled so far paid $10,943 more than expected: fast payments
  in full settle first, while denials and self-pay write-offs settle 30–120
  days later. The TUI rows are now "Revenue recognized", "+ collected vs
  expected" and "Net revenue", with a note.
* **Financing shortfall.** Negative cash stays allowed, as approved, but
  it is now tracked from the start:
  * the lowest balance and when it happened, and days ended below zero;
  * one alert per dip below zero, and a red **FINANCING SHORTFALL** on the
    finance screen and the dashboard.

  No credit line or bankruptcy is modelled.
* **Underlying categories.** Fixed-cost ledger lines carry their item, so
  every period reports fixed cost by item (rent, utilities, insurance, each
  lease and service contract, IT) whatever account it posts to.
* **History boundary.**
  * Days before the start date are history: daily records, ledger and log.
  * Today's counters, demand and statements since day 1, cash change today
    and the financing figures start at day 1. Day 1's 00:00 accruals are
    day 1's.
  * The opening adjustment never appears as today's cash change.
  * The cash history and day records restate the days before takeover by
    the adjustment, so the series is continuous. This is labelled.

**On screen**
* **Briefing** (`b`):
  * money: opening cash, owner capital, last-30-day P&L and cash flow,
    carried-over receivables, payables and payroll, next payroll and bills,
    and the financing note;
  * patients: last 7 days of demand and exams, waits, utilization of open
    and staffed hours, bookings ahead;
  * worklist and TAT;
  * today's roster.
* **Daily history** (`h`): 30 days, with history rows marked "·".
* **Finance:** a balance sheet line with the equity split, a financing
  line, and the clarified revenue rows.

**Determinism and correctness tests** (`internal/sim/warmup_test.go`):

| Requirement | Test |
|---|---|
| Opening balance sheet reconciles | `TestOpeningBalanceSheet`: one opening entry, at the start, after every warm-up entry; adjustment = target − cash before; cash = $150,000; assets = liabilities + equity; equity components match the journal; A/R, A/P and payroll carried over and non-zero; day-1 "today" figures empty; history ends the day before the start, restated to the opening cash |
| Claims, payments, reports, worklist, appointments, weekly overtime survive | `TestStateSurvivesTheBoundary` (13:00 start, so patients on site and the worklist cross the boundary). Every study pending at the start is finalized once. All of the 1,000+ claims open at the start settle within 145 days (the longest path). Warm-up bookings resolve. Riley's weekly paid time at the start equals Monday's pay exactly. |
| Warm-up vs uninterrupted run; chunking; unrelated events | `TestWarmupDeterminism`: a daily-chunked warm-up, a warm-up with live and stale noise events, and a rebuild give identical log and ledger hashes and identical observations; build then 3 days of play = one uninterrupted run; seed 2 differs |
| Golden start state | `TestStartStateGolden` (ledger and log digests, seed 1) |
| No hidden future in the opening state | `TestOpeningStateHidesTheFuture`: forcing every patient from day 2 to no-show and every claim adjudicated after the start to fail leaves the observation, the 180-day history and tomorrow's schedule unchanged; afterwards the books differ |
| Convergence (ROADMAP) | `TestWarmupConvergence`: nine-seed means, 180 vs 365 days. Forward fill to 90 days, A/R and A/R ≤ 60 days within ±5% (largest: 7–30-day fill −4.6%); tails within ±25%. A 90-day warm-up fails (−12.6% fill, no A/R over 90 days). |
| Financing shortfall | `TestFinancingShortfallTracked`: $10k opening cash; an alert per dip; lowest cash and days below zero agree with the daily history |
| Optimizations change nothing | Formatter equivalence tests against `fmt`. Before the step-11 behavior was added, a full warm-up plus 3 days produced byte-identical log text (61,226 entries) and ledger (`a37b36…`) to `32313ac`. |

All step 1–10 tests pass, including the step 8 remediation regressions and
the step 10 finance tests. The full suite takes about 2 minutes on the Pi;
the convergence test is 30 s of that and is skipped with `-short`.

**Performance (before / after).** Warm-up build, alternating old and new
with cool-downs:
* step 10: 1.80–2.38 s/op, 111.6 MB, 1.97M allocations;
* step 11: 1.35–1.97 s/op, 89.7 MB, 1.26M allocations.

That is about 22–25% faster at the median. The spread is thermal: the Pi
ran at 75–79 °C with throttling, and unthrottled builds take 1.34–1.35 s.
Details and the remaining profile are in MEASUREMENTS.md:
* What changed: lazy log time text, a streamed log hash, Sprintf-free
  ID and time formatting, a per-day memo of keyed draws (30% were
  repeats), the claim draw kept on the claim, and reused room staffing
  lists.
* What didn't: no randomness semantics, no event suppression and no
  concurrency.

**Remaining limitations**
* **Target ≤ 1.5 s only when unthrottled.** Throttled builds take
  1.7–1.9 s. The next steps are a lazy structured log, or the start-state
  cache once snapshots exist (step 14).
* **The worklist is empty at 06:00 on day 1.** Monday's studies are all
  read by Monday evening at the current (below-design) volume; the design's
  10–15 unread is a calibration matter (step 18).
* **Large owner capital.** The history loses money, so takeover needs
  $569k of owner capital. This is expected to shrink to a working-capital
  figure after calibration.
* **Utilization is minimal.** The daily records keep only exam time over
  open and staffed hours; the full metric set is step 12.
* **No financing model.** No credit line, interest or insolvency (V0.2).

**Decisions for owner approval**
1. **Opening capital convention** (implemented as documented): one
   adjustment at takeover, with the history's balances and losses kept.
   Alternatives:
   * (a) opening cash at the epoch and no adjustment, so the start cash is
     what the history leaves (−$268,829 for seed 1);
   * (b) no opening capital (−$418,829);
   * (c) keep the adjustment and calibrate so the history is about
     break-even (step 18).

   Recommendation: keep the adjustment, aim for (c).
2. **Restated cash history.** Before day 1 it is shown as if the opening
   capital had always been there (continuous series, labelled). The
   alternative is the raw series with a jump at takeover.
3. **The log digest format changed** (streamed SHA-256). Digests differ
   from step 10 by construction; outcomes do not.

---

## Step 10 — Ledger, claims, payroll and costs (v0.1.0-dev.10)

**What was built.**
* A double-entry ledger in integer cents (`internal/sim/ledger.go`).
* Revenue at report final, the claim lifecycle, payroll, and fixed and
  variable costs (`finance.go`).
* Statements, claims and the journal in the API (`finance_view.go`,
  `api.Finance`), and the TUI finance screen.
* All the assumptions are scenario data: the new `finance` section and
  catalog `baseRateCents` / `supplyCostCents`, taken from §8.2–8.3 of the
  design. They are illustrative and not tuned. Design detail is in
  SIMULATION_DESIGN §8.7.

**Accounting conventions**
* **Accrual for profit, cash for liquidity, one ledger.** Revenue is earned
  when the radiologist finalizes the report: Dr A/R / Cr revenue at expected
  net = base × payer multiplier × NCR. It is never earned at booking,
  arrival, registration, imaging, worklist entry or reading start. Cash
  moves only when a payment arrives or payroll or bills are paid. The
  practice can be profitable and short of cash at the same time.
* **Every entry** balances and is non-empty, and has a stable idempotency
  key (study, claim and payment number, employee and day, or day). It is
  never edited. Corrections are linked reversing entries, which can be
  posted once and never reversed themselves. The incremental check re-sums
  new entries independently, the day books must equal the balances, and
  the accounts sum to zero.
* **Claims.**
  * Pending, then submitted 1–3 business days after final, at 17:00 in one
    batch. Remittance comes after the payer's delay, on a business day at
    10:00.
  * Outcomes:
    * paid in full;
    * short paid at m ± 10 pp;
    * denied, then resubmitted the next business day, then recovered or
      written off 30–60 days after the denial;
    * self-pay unpaid, written off and not counted as a denial.
  * Settlement sends expected − paid to collection variance (contra
    revenue), so a closed claim has zero A/R.
  * Outcome, delay, short fraction, rework time and charge lag come from
    the order's origin key (`claim`, `claim.lag`). The payer class is drawn
    when the order is disclosed.
* **Payroll.**
  * Hourly staff accrue when they go home: worked time minus breaks taken,
    with skipped breaks and held-over time paid. Pay is at the regular rate
    until the week reaches 40 paid hours, then ×1.5 (the step 8 rule).
  * Salaried staff accrue annual ÷ 365 per calendar day. Radiologists
    accrue × their site allocation.
  * Benefits are 25% on everything.
  * Payroll is paid every other Friday from Oct 16 at 00:00 and covers
    everything accrued through Thursday.
* **Costs.**
  * Fixed monthly items accrue daily with exact monthly totals.
  * Supplies (contrast included) and $3 PACS post at acquisition.
  * Billing is 5% of each payment.
  * Accounts payable is paid on the 1st.
* **Statements.**
  * Today, month to date, trailing 30 days and since day 1.
  * Net revenue = revenue − collection variance.
  * Expenses are shown by account, and also as labor, fixed and variable.
  * Operating profit and margin; the margin is undefined at zero revenue.
  * Collections, disbursements and net cash flow.
  * Denial rate, days in A/R and A/R aging.
  * Revenue and direct cost by modality, plus a cash history.

**On screen**
* `f` finance has three views, switched with `tab`:
  * **overview**: periods side by side, cash and the next payroll and
    bills, A/R and denials, the modality table and a cash sparkline;
  * **claims**: open claims, oldest first;
  * **journal**: the latest entries as debit and credit lines.
* The dashboard's second line shows cash ▲/▼ today, revenue, expenses and
  operating profit today, and A/R.
* The report-final log line now ends "· revenue $395".
* The log adds lines for claim batches, payments, short payments, denials,
  write-offs, payroll and bills.
* Patient journeys continue through revenue, claim, denial and
  resubmission, and payment or write-off.

**Tests** (`internal/sim/finance_test.go`, `ledger_internal_test.go`, a TUI
test, and scenario validation cases):

| Requirement | Test |
|---|---|
| Revenue once, only at final; none for cancelled, no-show, left, lost, rescheduled-unperformed or unread exams | `TestRevenueOnlyAtReportFinal`: minute by minute on day 1, then 30 days; non-vacuity checks for each excluded state |
| Correct modality and payer reimbursement | `TestFinanceArithmetic` (NCR 0.950/0.980/0.934/0.400; CT w/ contrast, commercial = $416 allowed, $395.20 expected), `TestClaimLifecycleAndTiming` (every claim's allowed and expected amounts) |
| Claim lifecycle, payment timing, denials, receivables | `TestClaimLifecycleAndTiming`, 150 days. For every claim, the drawn charge lag, remittance date, rework date, state and paid amount match exactly. Non-vacuous for paid, short, denied → paid, denied → written off, self-pay unpaid, open and pending. A/R per claim and cash are rebuilt from the journal alone. |
| Collection variance mean ≈ 0 (ROADMAP) | `TestCollectionVarianceMeanZero`: 30,000 draws per payer; the collected fraction is within 0.6 pp of NCR |
| Balanced entries, idempotency, reversals | `TestLedgerPostingRules`: unbalanced, empty, duplicate, out-of-order and bad lines are rejected with no effect; reversal mirrors and links; no double reversal; tampering is detected. Every simulation test with invariants re-sums the journal. |
| Cash and receivable reconciliation | `checkFinance` invariants (cash = opening + receipts − disbursements from independent totals; A/R = open claims; revenue = Σ expected; claims = final reports; accrued payroll and A/P), checked after every transaction in all invariant-checked tests and at each day boundary in the finance tests |
| Payroll, regular and overtime; weekly threshold crossing | `TestPayrollWeeklyOvertime` (`micro-week.json`): Tue–Thu $630; Fri crosses 40 h after 8.5 h, $690; Sat all premium, $360; Monday resets, $630; unpaid lunch excluded; benefits 25% |
| Required overtime after scheduled close | `TestPayrollRequiredOvertime`: wages = scheduled part + exactly the step 8 overtime cost (36 min held over: $126 = $90 + $36; premium fixture $189 = $135 + $54); one entry per employee and day |
| Fixed and variable cost timing; day, week and month boundaries | `TestCostAndPaymentTiming`, Oct 13 to Dec 1: one fixed accrual per day at 00:00; November totals exactly the monthly sum; A/P paid Nov 1 and Dec 1 equals the balance before; supplies and PACS once per study at acquisition; billing fee = 5% of each payment; payroll only on the four anchored Fridays, equal to the accrual through Thursday; salaries once per employee and day; wages on the day worked |
| Same-instant ordering | The same test checks that at 00:00 bills and payroll are paid before that day's accruals. Claim batches and remittances are class 6, after operations at the same instant. |
| Sparse events, speed, replay | `TestFinanceDeterminism`: 75 days in one jump, daily, or in 13-minute chunks give identical ledger and log hashes. A replay is identical; a different seed differs. |
| Independence from unrelated events | `TestFinanceUnaffectedByUnrelatedEvents` (live and stale events at 10:00, 17:00 and around them for 50 days); the step 8 wake-completeness tests now also compare the ledger hash |
| Outcomes independent of operations | `TestClaimOutcomesIndependentOfOperations`: worklist FIFO vs shortest-first changes finalization times, but no payer, amount or outcome |
| Hidden outcomes not leaked | `TestFinanceOutcomesNotLeaked`: forcing every claim to fail leaves the full observation identical for 15 days (and all journeys on day 15), then changes cash. The latent-field deny-list now covers `api.Finance`. |

Mutation checks (each made the named test fail, then was reverted):
* revenue at exam end;
* claim draw keyed by processing order;
* weekly threshold ignoring earlier days;
* held-over time unpaid;
* payables paid on the 2nd;
* charge lag in calendar days (caught once the test checked the exact
  drawn lag);
* an observation revealing a forced outcome;
* a wrong short-paid amount;
* a cash identity off by a cent.

All step 1–9 tests pass, including the step 8 remediation regressions. The
fixtures gained rates and a `finance` section, but no expectation changed.

**Measured: 180 days of the mini practice, seed 1** (2.1–2.5 s/op on the
loaded Pi, MEASUREMENTS.md):

| | Since day 1 (181 days) | Trailing 30 days |
|---|---|---|
| Revenue earned (expected net) | $1,125,060 | $195,429 |
| Collection variance (favorable) | +$10,943 | +$2,429 |
| Net revenue | $1,136,003 | $197,858 |
| Operating expense | $1,336,861 | $222,177 |
| — labor (wages $508,015 + benefits $127,005) | $635,020 | $103,677 |
| — fixed | $529,192 | $86,692 |
| — variable (supplies $108,567, PACS $58,461, billing $41,498) | $172,649 | $31,809 |
| Operating profit | −$200,858 | −$24,319 |
| Operating margin | −17.7% | −12.3% |
| Cash collected | $829,949 | $179,528 |
| Disbursements | $1,260,067 | $223,235 |

* At the end: cash −$280,118 (opening $150,000); A/R $306,054 (46 days);
  A/P $43,247; accrued payroll $33,547.
* Denial rate: 5.9% (309 of 5,199 insurance adjudications).
* Claims: 7,514. Paid 4,935, short paid 249, written off 270, open 2,060.
* By modality (net revenue / direct cost):
  * XR: 3,538 reports, $182,866 / $34,074;
  * US: 1,216 reports, $213,322 / $30,321;
  * CT: 1,833 reports, $436,681 / $180,790;
  * MR: 927 reports, $303,134 / $223,453.
* Hot spots: SHA-256 22%, dispatch 23%, log formatting 14%, finance about
  10% (adjudication 4%, recognition 3%, posting 2%).

**Why the margin is negative.** Volume, not the financial assumptions.
Monthly-equivalent figures:
* Labor ($106.7k vs the reference $105.1k) and fixed costs ($88.9k vs
  $88.5k) match the §8.6 reference calculation.
* Net revenue is $190.8k vs $261.5k. That is about 1,260 reports a month
  instead of 1,650, the volume shortfall noted since step 7. Revenue per
  report is $151 vs $158.5, from the exam mix.

Nothing was tuned toward the 8.5% target, as instructed. That is step 18.
The cumulative collection variance is favorable (−1.0% of revenue) because
claims that settle early are mostly paid in full. Denials and self-pay
write-offs settle 30–120 days later. Over all outcomes the mean is zero,
as the draw test shows.

**Known limitations**
* The negative margin and cash are a volume artifact until calibration
  (step 18).
* Opening cash is posted at the epoch. The warm-up opening-equity
  adjustment at the start time is step 11.
* No credit line or interest: cash simply goes negative.
* No copays or deductibles (not specified). Bad debt and denials share
  collection variance (split in V0.3).
* No holidays; business days are Mon–Fri.
* No prepaid expenses (rent is paid in arrears on the 1st).
* No payroll tax remittance schedule separate from payroll.
* No reversal flow in V0.1 operations. The mechanism exists and is tested.
* The end-of-day summary's finance line comes with the summary screen
  (later step).
* Performance: finance adds about 0.3–0.4 s per 180 days. The step 11
  warm-up target now depends on the planned mitigations.

**Decisions for owner approval** (each is scenario data or a small,
documented rule; SIMULATION_DESIGN §8.7):
1. **Rework timing.** "Resubmitted after ReworkDelay" is read as denial →
   final outcome in 30–60 days, with the resubmission on the next business
   day.
2. **Remittance clock.** Business days only; submissions in a 17:00 batch,
   payments, denials and write-offs at 10:00.
3. **Self-pay.** A statement, not a claim. Non-payment is written off at
   the due date, never reworked, and excluded from the denial rate.
4. **Chart mapping.** Utilities and insurance go to *Rent & facility*; the
   monthly IT/RIS charge to *IT / PACS*; leases and service to *Equipment
   service & leases*, tagged by modality. No new accounts.
5. **Payables.** All of them (rent included) are paid in arrears on the
   1st, because the chart has no prepaid account. Payroll is paid at 00:00
   on the anchored Friday (Oct 16, then every 14 days) and covers what was
   accrued through Thursday, with benefits paid alongside.
6. **Salaried accrual.** Per calendar day (annual ÷ 365), weekends
   included. Radiologists × site allocation.
7. **US base rates** within the documented $140–170: abdomen $170, pelvis
   $155, venous $160, thyroid and renal $140.
8. **Negative cash** is allowed and shown in red (no financing in V0.1).

---

## Step 9 — Worklist, reading and turnaround (v0.1.0-dev.9)

**Behavior**

* **Studies**: acquired at exam end (imaging complete, TAT start), on the
  worklist once the technologist's documentation is done, then read, then
  final. There is one study per imaged visit, never a report before imaging,
  and never a second finalization.
* **Radiologists read** during their site reading blocks while on regular
  duty. Remote radiologists read too. Each claims the first study in worklist
  order whose longest read (p100) ends by their scheduled shift end, so
  reading never extends a shift or crosses midnight. One read at a time.
  Reads that run past a block's end finish, and the overrun delays the next
  block (§6.7, with a wake-up so the claim happens on time).
* **Supervision** stays presence-based and compatible with reading, so a
  radiologist can read while supervising. The supervisor of an exam in
  progress stays until it ends (step 8 remediation). A remote radiologist
  never supervises. A held-over radiologist claims no new reads.
* **Read times** are per case (catalog `readSec`, keyed by the order's
  origin), so worklist order and policy never change them.
* **Worklist order** (default urgent then FIFO): urgent studies and studies
  already past their TAT target first (no starvation), then time on the
  worklist, with the study ID as the final tie-break. FIFO, oldest-first and
  shortest-first (by the published expected time) are ready for the step-13
  action.
* **Turnaround**: TAT = imaging complete → report final. The figures shown
  are median and p90 over studies acquired today with unfinished ones at
  their age (censored), worklist wait, read time, backlog by modality and
  priority, the oldest unfinished study, and breaches. Targets: urgent 60
  min and routine 24 h (documented), soon 4 h (illustrative). Overdue
  urgent and soon reports are logged once.
* **After operational close** radiologists keep reading in their blocks.
  Operations close without waiting for reports, and the backlog carries into
  the next day (Saturday's afternoon studies are read on Monday).

**On screen**: `w` worklist (reading and unread studies, turnaround,
backlog, latest reports; `enter` opens the patient), a dashboard reporting
line, radiologists shown as reading / ready to read / other duties /
supervising, and journeys that continue to the final report and its TAT.

**Tests** (`internal/sim/reading_test.go`, new micro fixtures):

* A month of the mini practice:
  * exactly one study per imaged visit;
  * timestamps ordered, no duplicate finalization;
  * every claim inside the claimer's site block;
  * every read ends by shift end;
  * no radiologist reads two studies at once;
  * backlog crosses days.
* Priority and ties: the urgent study is read first, and two studies
  entering at the same instant go in ID order.
* TAT definition (acquired 08:23, final 08:29 → 6m).
* Overrun delays the next block to 09:51 (claim exactly then); a read ending
  exactly at block end owes nothing (claim at 09:40, same-instant ordering).
* Radiologist distinctions (ROADMAP): remote reads, never supervises; on-site
  in other duties supervises, claims no reads.
* Read time unchanged when the policy reorders the worklist. A deliberate
  dependency on claim time makes this test fail.
* Observation types carry no latent fields.
* Outcomes unchanged by injected live or stale events (now including the
  reading fixtures).
* All step 8 remediation tests, chunking and speed tests pass.

**Measured**: 180 days of the mini practice in 1.7–1.9 s on the loaded Pi:
7,528 studies, 7,514 final, 14 outstanding at the end. Reading is not a hot
path. See MEASUREMENTS.md.

**Limitations**

* Volumes are below calibration, so the reading backlog is small. Weekday
  TAT medians run 20–40 min, and Tuesday mornings start nearly clear (the
  design expects 10–15 unread from warm-up at full volume, steps 11 and 18).
* No report interruptions, addenda or protocoling (V0.2).
* Revenue at finalization is step 10.

---

## Step 8 remediation — review findings 1–10 (v0.1.0-dev.8.1)

The step 8 review (`docs/reviews/step-08-astra-review.md`, gpt-6-astra
at high reasoning) found 10 defects, all confirmed. For each, a regression
test reproducing the reviewer's scenario was written first and run against
the unfixed code (`ee26417`), then the fix was made and the same test
rerun. Tests are in `internal/sim/regress_test.go`, with fixtures in
`internal/sim/testdata/micro-*.json`.

| # | Defect | Regression test | Before (on `ee26417`) | After | Resolution |
|---|---|---|---|---|---|
| 1 | Held radiologist released while supervising; colleague kept past her cap | `TestRegress01RadiologistHandoverCap` | FAIL: Dr. Bo overtime 0s (want 33m), Dr. Rae 3m (want 0, cap 09:30) | PASS | The supervising radiologist is recorded at exam start and stays until the exam ends. A waiting contrast patient keeps a radiologist only if she could supervise in time and no on-site colleague on regular duty could. New invariant: every contrast exam has its supervisor on duty, on site, within cap. |
| 2 | Held employee kept past usefulness and cap (no wake) | `TestRegress02HeldPastUsefulnessAndCap` | FAIL: Tess overtime 45m, want 0 | PASS | "Needed" now uses the patient's earliest possible start and the full remaining task. A single deadline wake fires when a held employee stops being useful or a patient stops being servable. |
| 3 | Desk staff who can't fit a registration block self-registration; patient stranded | `TestRegress03DeskCannotBlockSelfRegistration` | FAIL: panic, 1 visit still on site at accounting close | PASS | Desk and ops staff block self-registration only while on regular duty and able to fit a registration. They never start registrations after shift end. The deadline wake covers the moment they stop fitting. |
| 4 | Last technologist released while their patient is registering | `TestRegress04HoldForRegisteringPatient` | FAIL: patient rescheduled "no XR technologist for the rest of today" | PASS | Registering patients count as outstanding for hold-over. |
| 5 | Cap at midnight: documentation ends at 00:00, after day start; overtime lost | `TestRegress05MidnightBoundary` | FAIL: overtime 0m recorded, want 15m | PASS (scenario now rejected: cap must end by accounting close) | Day boundary: shift start + max hours must be ≤ 23:59:59. Accounting close sends home anyone past shift end (overtime booked to that day) and fails loudly if anyone is busy. The tradeoff against carrying work across midnight is documented in SIMULATION_DESIGN §6.9. |
| 6 | FirstFit skipped an earlier preferred slot in a later room | `TestRegress06FirstFitPreferredAcrossRooms` | FAIL: booked room 0 at 15:00, want room 2 at 13:00 | PASS | The early exit compares any-window and preferred candidates separately. |
| 7 | Shared capacity checked only at slot start | `TestRegress07SharedCapacityAcrossSlot` | FAIL: XR booked 09:00 on day 7 although its technologist is booked in CT 09:30–10:00 | PASS (XR goes to the next free day) | Slot technologist sets cover the whole slot. `sharedFree` checks every constant-coverage stretch of the slot. |
| 8 | Opening schedules bypassed shared capacity | `TestRegress08OpeningRespectsSharedCapacity` | FAIL: no validation error | PASS | `sim.CheckOpening` books the opening schedule in order under the same rules, and scenario validation reports problems by JSON path. The mini opening Tuesday moved two CT appointments (08:00 → 09:15, 09:00 → 11:00), and `micro-shared.json` now alternates rooms. |
| 9 | A second break overwrote a deferred first break | `TestRegress09DeferredBreaksDoNotOverwrite` | FAIL: break taken 15m, want 30m | PASS | A break falling due while busy or on break waits its turn after the current one. An older waiting break is skipped and counted, never silently dropped. |
| 10 | Negative wait for patients not yet imaged | `TestRegress10WaitWhileRegistering` | FAIL: registering, waited -480 min | PASS | Waited runs to now while waiting, to exam start once imaged, and to departure otherwise. |

**Missing-wake detection.** `TestRegressOutcomeIndependentOfUnrelatedEvents`
reruns ten micro practices with no-op events (handled, so dispatch runs)
every 7 minutes, and with stale events. On `ee26417` four of them changed
outcome (`micro-cap`, `micro-contrast-cap`, `micro-cap-wake`,
`micro-desk-cap`). After the fixes none do. A four-day version on the full
mini practice with invariants on (`TestRegressMiniUnaffectedByUnrelatedEvents`)
passes. It also passed on `ee26417`: the mini practice rarely reaches these
edges, which is why the micro fixtures exist.

**Expected changes to existing tests.** In `TestCoverageExhaustedReschedules`
the patient is now rescheduled at 09:09:01, the second their last feasible
start passes (cap 09:30 minus 21 min). Before, it happened at the next
unrelated event (09:14). The contrast hold-over now logs "supervising a
contrast exam". `TestSharedTechnologistContention` samples at 08:20 because
its bookings now alternate rooms.

**Verification.** The full suite passes. That covers the step 8 closing
tests (acceptance criteria rechecked: patients served after scheduled
close, required overtime costed, explicit coverage reschedules, contrast
after hours, operational close after scheduled close), month-long
invariants, minute-by-minute chunking, and identical results across speeds
and event budgets. 180 days of the mini practice take 1.4–2.0 s on the
loaded Pi (MEASUREMENTS.md, step 11 risk).

---

## Step 8 — Registration, imaging, overtime and closing rules (v0.1.0-dev.8)

**Planning changes first** (your decision): scheduled closing time no
longer cuts off admitted patients. PLAN D21, SIMULATION_DESIGN §3.4.2 (three
closes) and §6.9 (closing time and required overtime), the §6.4–6.5 rule
changes, invariants 7–8a, closing metrics, ROADMAP step 8, acceptance
criterion 5a, an exploit risk row, and the TUI closing status.

**Behavior**

* **Registration**: front desk, then the operations manager (×1.25), then,
  only with nobody desk-capable on duty, the patient's technologist (×1.5).
  The queue is urgent first, then scheduled patients by appointment time,
  then walk-ins.
* **Imaging**: patients across all free rooms are ranked by the queue policy
  (urgent, due by appointment, late fitted in, walk-ins under the walk-in
  policy, early arrivals) and started when a qualified technologist on the
  room is free and can finish by their 12-hour cap. Contrast also needs an
  on-site radiologist whose cap covers the exam. Room time and documentation
  are drawn per exam. The technologist documents for 1–12 min after the
  room is released.
* **Shared technologists**: one person, one exam. Booking now respects it
  too: a slot whose technologist also staffs another room isn't bookable
  while that person is already booked there (schedule shows `¦`).
* **Scheduled close** stops admissions only. Admitted patients are still
  registered and imaged. Staff past their shift end are **held over** while
  admitted patients need them (a technologist for the rooms they had, an
  on-site radiologist for contrast). They are released at the first instant
  they're not needed. Overtime minutes and cost (×1.5 past 40 paid hours a
  week, hourly staff) are recorded per person and day.
* **Coverage runs out** → the patient is rescheduled explicitly with the
  reason (for example "no XR technologist can finish within the 12-hour shift
  limit", "no radiologist on site to supervise contrast"), on arrival or
  while waiting. Nobody is sent home for the clock.
* **Operational close**: the first moment after scheduled close with nobody
  on site and nobody mid-task. It is logged with after-hours exams and
  overtime. Accounting close at 23:59:59 checks nobody is still on site.
* Breaks due while busy are deferred up to 60 min, or skipped (counted).
* **Template change [illustrative]**: Saturday XR/CT reduced to 12 + 7
  staggered slots. Sam works both rooms alone and registers everyone.

**On screen**

* Practice line: `● OPEN 08:00–20:00`, then `◐ CLOSING — 3 patients
  remaining · OT +42m ($38) after 20:00 · held over: Sam`, then
  `○ CLOSED · operations closed 20:47 (scheduled 20:00) · OT +1h 12m ($63)`.
* Rooms show scans in progress (`● Scanning P00614 Head w/o 14/12m Sam`,
  amber when past the expected time) and how many are waiting. Staff show
  their task (registering, exam, documenting) and overtime.
* Patient flow: on site by stage (registering, waiting by modality, in
  exam), longest wait, exams done, after-hours exams, coverage reschedules.
* Journey: registered (by whom, when), exam (technologist, room, wait, time
  taken against the expected time, after scheduled close), departed or
  rescheduled with the reason.

**Tests**

* Micro practices (`internal/sim/testdata/micro-*.json`, fixed durations)
  check exact timelines:
  * waiting patients stay after scheduled close, and an exam starts after it;
  * operations close at 09:36 against a scheduled 09:00;
  * 36 min of required overtime costs $36, or $54 past 40 weekly hours;
  * the shift cap forces an explicit reschedule with its reason;
  * a shared XR/CT technologist never runs two exams at once;
  * an on-site radiologist is held over for contrast after hours, and runs
    out of cap with an explicit reason;
  * patience expiry beats an exam start at the same instant, and a patient
    being registered can't leave.
* Across a month of the mini practice: every visit ends imaged, left, or
  rescheduled with a reason, overtime is recorded, and some days close
  late. Invariants are checked after every transaction: shift cap, one task
  per person, room occupancy, timestamp order, nobody on site at accounting
  close.
* Determinism: minute-by-minute chunks and various speeds and budgets give
  identical results.

**Measured on the mini practice** (28 days, seed 1): wait from arrival to
exam has median 14 min, p90 44 min, max ≈ 3 h (busy Saturdays and
evenings). Overtime averages ≈ 30 min a day (≈ $15). There are a few
after-hours exams a week.

**Limitations**

* No study or report yet. The worklist and radiologist reading are step 9.
  Radiologists only supervise contrast here.
* Held-over staff take no breaks. There is no minimum rest before the next
  shift yet (V0.2), so a late night doesn't thin the next morning.
* `extend_shift` (approved overtime) comes with the actions in step 13.
  Until then all overtime is *required*.
* Performance: 180 days of the mini practice take ≈ 1.3–1.7 s on the loaded
  Pi. Step 11 must apply the warm-up mitigations (MEASUREMENTS.md).

---

## Step 7 — Arrivals, missed appointments, walk-ins, patience (v0.1.0-dev.7)

**Behavior**

* **Scheduled patients arrive** early (≈ 60%, 5–25 min), on time (≈ 28%),
  late (≈ 9%, 5–20 min) or very late (≈ 3%, 20–45 min), or never. The no-show
  rate is a base per modality (XR 4%, US 7%, CT 6%, MR 8%) plus 0.5 points
  per 10 days of booking lead, capped at +6. Whether and when each patient
  comes is drawn by origin key when the appointment's day starts, and stays
  hidden until it happens.
* **Missed → no-show:** at start + 15 min an absent patient's appointment is
  marked *missed* (provisional). If they arrive later the same day they are
  fitted in (late policy `fit_in`) or rescheduled (`reschedule`). At
  accounting close (23:59:59) appointments still missed become *no-shows*.
  40% of no-shows ask for a new date the next morning. A patient who
  arrives after closing is rescheduled.
* **Walk-ins** (≈ 8 per weekday, about 85% X-ray, some venous ultrasound and
  head CT, never MR) arrive only in open hours, peaking late morning and
  late afternoon. Each becomes a patient and an order like any referral. If
  their modality has no technologist for the rest of the day (or the
  walk-in policy is `schedule_later`) they are booked an appointment and
  leave. Otherwise they wait.
* **Patience:** a waiting walk-in leaves after 30–120 min (hidden until it
  happens) and is counted as having left without an exam.
* **Closing time:** no exam starts after closing, so anyone still waiting is
  sent home unseen. **Until imaging exists (step 8) that is everyone who
  arrives**, so the waiting room fills through the day and empties at close.
* **IDs:** visits get `V…` IDs when the patient arrives. Each appointment and
  each order has at most one visit.

**On screen**

* The top bar shows the version: `DREAMTEAM v0.1.0-dev.7`.
* Dashboard **PATIENT FLOW**: waiting now by modality and the longest wait,
  no-shows at the last close, today's arrivals (late, after grace),
  walk-ins, walk-ins who left, provisional misses and patients sent home
  unseen. Each room line shows how many are waiting for it. On 80-column
  terminals the staff summary moves to the practice line, and the log keeps
  5 entries (8 on larger terminals).
* **Patients (`p`)**: a day's list with appointment time, actual arrival
  (with the offset when more than 5 min off), time waited and status.
  Filters (`tab`): all, waiting, missed / no-show, left / unseen, walk-ins.
  `←→` pages days. `enter` opens the **journey**: ordered or walked in,
  booked, missed (provisional), arrived (how early or late), waiting, left,
  sent home, no-show, rescheduled, lost. Only observed facts appear.

**Tests**

* Arrival offsets match the punctuality classes over 8 weeks. No-shows run
  4–10%, are finalized only at accounting close, and never have a visit.
  Patients who were marked missed and then arrived appear in the journey as
  "Missed (provisional) > Arrived".
* Conservation: every appointment of a finished day ended arrived, no-show,
  cancelled or rescheduled, with at most one visit per appointment and per
  order. Invariants (visit ↔ appointment ↔ order states, waiting list) are
  checked after every transaction in the simulation tests.
* Same instant: an arrival beats a grace check and a cancellation whatever
  the push order; grace then cancellation leaves the appointment missed; a
  patience expiry at closing time is handled before closing.
* Walk-ins: 6–12 per weekday, ≥ 75% X-ray, no MR, only eligible exams, only
  in open hours. Patience: departures between 30 and 120 min, counters match.
* Hidden-state noninterference: forcing every later patient to be a no-show
  and removing later orders and walk-ins changes nothing observable before
  it could be seen. A deliberate leak makes this test fail.
* Speed: steps of 1 min, 7 min and 2 h, and an event budget, give identical
  observations and patient lists. The existing booking, coverage and
  determinism tests pass.

**Limitations**

* No registration or imaging yet (step 8), so the waiting room only empties
  at closing time and rooms stay idle with patients waiting.
* `accept_if_gap` and `accept_always` both let walk-ins wait. The gap rule is
  applied when exams are dispatched (step 8).
* A no-show's new-date request comes the next calendar day at 09:00, even on
  Sunday (booking is centralized).
* Performance: 180 days of the mini scenario now take ≈ 0.8 s on the Pi
  (MEASUREMENTS.md). The step-11 warm-up target (< 1.5 s) will need the
  noted mitigations.

---

## Step 6 — Demand and booking (`ff54f40`)

Referral orders from six fictional sources arrive through the day. Counts,
times and attributes are keyed by origin, and IDs are given only when an
order arrives. FirstFit books slots that the roster staffs for the whole
slot (contrast slots also need an on-site radiologist). It tries the
patient's preferred time of day, then any time, within their wait
tolerance. Otherwise the order is lost (declined, or no slot within 120
days). Orders wanted beyond the horizon go to a recall list. Hidden
cancellations free the slot at once, and half the patients ask for a new
date from the next day. The mini scenario has a hand-built Tuesday of 47
appointments. TUI: dashboard DEMAND panel, compact dashboard log, full log
(`l`), schedule (`s`) with 15-minute timelines, slot list, day paging and
demand against capacity. Tests: determinism, demand unchanged by booking
capacity, feasible and non-overlapping bookings, demand accounting, recall
and cancellation paths, ID noninterference.

## Step 5 — Scenarios, rooms, staff, roster (`6ed41f3`)

Scenario JSON with `extends` and preset merging, strict decoding and
validation errors that name the JSON path. The bundled practice and the mini
scenario. Fortnight rota (weeks A/B), breaks, cover windows, radiologist
site reading blocks. Room states closed / unstaffed / idle, with one
staffing rule shared by the runtime and the coverage report
(`dreamteam scenario coverage mini`). TUI rooms and staff panels. Tests:
coverage timelines for Mon/Tue/Sat in both rota weeks, and the runtime
matches the plan minute by minute over a fortnight.

## Step 4 — API, session, TUI shell (`7ef6abf`)

Observation contract, session pacing (advance, event budget, step to the
next visible entry), Bubble Tea TUI with five speeds, pause, step and
auto-pause. The TUI cannot import the simulation (tested). Frame rate and
speed never change outcomes (tested).

## Step 3 — Event engine (`76476b2`)

Timestamp transactions with event classes, stale-event skipping, invariant
hooks. Chunking invariance tested, plus a synthetic 180-day load benchmark.

## Step 2 — Randomness (`0cbcf95`, `214d2d9`)

SHA-256-keyed draws with four fixed lanes per key, ChaCha8 streams,
integer-only uniform, Bernoulli, weighted, quantile and Poisson samplers,
golden values cross-checked against Python.

## Step 1 — Simulated time (`a3b4015`)

Integer seconds, practice calendar, clock and date formatting.
