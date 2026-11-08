# Verification and partial-run analysis — dreamteam.alternatives-partial-run/v5, analysis/v6

DRAFT, NOT LOCKED. Independent technical re-review PENDING.

[v13i_verify.py](../../analysis/alternatives/v13i_verify.py) is the
authoritative invariant classifier. It loads the lifecycle contract and the
archive files only; it does not import the recorder and never takes the
claimed status as truth.

## Classification

`classify` returns COMPLETE, PARTIAL or INVALID with every violated invariant
(L01–L19, contract `invariants`). It:

1. checks the file set (no marker, no stray file, manifest and ledger
   readable) and identities (schema, driver, lifecycle SHA-256, protocol,
   schedule, UUID, provenance, no resume);
2. replays the journal through the transition table (L14);
3. checks the ledger as a schedule prefix with complete evidence and binds
   each `commit_draw` to its row's canonical SHA-256 (L01, L02, L17);
4. checks the aborted artifact against the stage table (L05, L06);
5. re-classifies every health observation from its body bytes or exception
   class and checks that health checks occur exactly where the schedule
   requires them (L07);
6. derives the stop, the failure event and cleanup from the evidence and
   requires the manifest's copies to equal them (L06, L08, L19);
7. recomputes counters and call evidence (L03, L04);
8. derives the terminal state the evidence supports and compares it with
   the claim (L09, L10);
9. checks finalization hashes and every required timestamp (L16, L13).

## Analysis

`analyze` refuses any INVALID archive (L12). For a valid archive it replays
every committed row without a model (completed and interrupted rows must
match exactly: VERIFIED), recomputes canonical submitted identities in the
engine, executes each exact submitted representation once, and reports the
aborted draw by the stage table: transport replay when a workflow failure's
last recorded call failed without a body, otherwise UNVERIFIED; external
causes are always UNVERIFIED. Health stops, operator interrupts and driver
failures are reported as `failureVerification` UNVERIFIED with the stop.

COMPLETE archives produce the per-context arm differences (canonical
submitted actions within comparison-exposed draws) and pair summaries,
unchanged from V0.13h. PARTIAL archives produce none (L11). The primary
estimand is unchanged.

## Procedure

    python3 -I -B analysis/alternatives/v13i_verify.py classify --protocol P --contexts docs/benchmark/v13g-dev/contexts --dir RUN
    python3 -I -B analysis/alternatives/v13i_verify.py analyze --binary ./dt --protocol P --contexts docs/benchmark/v13g-dev/contexts --replays . --dir RUN --out OUT

A run is never resumed, extended, overwritten or patched. An archive the
verifier classifies INVALID is reported with its violations; it is not
analysed and not repaired.
