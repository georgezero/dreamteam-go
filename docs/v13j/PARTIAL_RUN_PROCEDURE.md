# Archive verification and the partial-run procedure — dreamteam.alternatives-partial-run/v6

DRAFT, NOT LOCKED. Analysis schema `dreamteam.alternatives-analysis/v7`.

[v13j_verify.py](../../analysis/alternatives/v13j_verify.py) classifies an
archive COMPLETE, PARTIAL or INVALID from its persisted files and the
contract (`lifecycle_v2.json`). It does not import the recorder and never
takes a claimed state as truth.

1. **Files** (L16, S01): exactly the admissible regular files; no temporary,
   marker or stray file, no directory; side artifacts are JSON objects (a
   present JSON null is a contradiction, never absence); strict JSON (no
   duplicate keys, no NaN). The call journal must end in a complete record
   and every record must be in canonical form. Without `SEAL.json` the
   archive is unsealed: INVALID, whatever the manifest says.
2. **Journal** (L14, L17, L19): chained legal transitions, no seal-only
   event, ending in FINALIZING; every transition predicate recomputed
   (`schedule_done` only after the last scheduled draw completed; `continue`
   only with draws remaining; each required health visit once; `begin_draw`
   only for the next scheduled draw after its required visit; reconciled
   commits followed by the stop).
3. **Ledger** (L01, L02): schedule prefix, complete evidence of completed
   rows (process evidence equal to the record), halt evidence of an
   interrupted last row.
4. **Call journal** (L04): dispatch/outcome pairs in order, bound to their
   draws, each draw's entries equal one-to-one to its recorded calls; no
   unresolved dispatch in a sealed archive; no recorded call without its
   dispatch; dispatched calls of a draw no artifact keeps are a lost-call
   contradiction (except `draw_unrecorded`, whose calls are null).
5. **Aborted draw** (L05, L06): every field fixed by its failure stage;
   process evidence recomputed from the artifact's own engine log and calls.
6. **Health** (L07): every kept observation re-checked independently against
   the closed exception families and the body rule; rejected values must be
   invalid; visits exactly as the schedule requires.
7. **Stop, cleanup, interrupts** (L06, L08, L19, L20): every stop field per
   code and state; the failure event recomputed; cleanup equal to its
   transition; interrupt events consistent with the stop and the schedule.
8. **Counters** (L03, L04): recomputed; `modelCalls` is the dispatch count.
9. **Terminal state** (L09, L10): the state the evidence supports.
10. **Seal** (S02–S04): identities, every hash recomputed from the bytes on
    disk (null meaning absent), transition and terminal state equal to the
    derived state.
11. **Timestamps** (L13): the strict grammar everywhere; required order,
    including dispatch <= sent <= outcome for every call.

Analysis refuses INVALID. For COMPLETE and PARTIAL it replays every
completed and interrupted row model-free through the unchanged V0.13h
machinery, executes canonical submissions, and reconstructs an aborted
workflow draw where its evidence allows (a raised transport call, or an
operator stop at the next dispatch boundary); the external cause stays
UNVERIFIED. PARTIAL produces no arm comparison or pair summary.

The `preserved` section of every classification reports the durable ledger
rows, completed rows, dispatched calls and unresolved calls, so that the
evidence of a rejected run is never silently lost.
