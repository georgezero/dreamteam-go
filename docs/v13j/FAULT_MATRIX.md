# V0.13j fault verification

Independent technical re-review PENDING. Every check uses scripted model
replies, scripted HTTP byte streams, genuine exception objects, real signals
to real processes, or localhost-only HTTP servers. No model endpoint, no
inference. All results below were generated on horo (linux/amd64, Python
3.13) and are regenerated and compared byte for byte by the gates.

## Fault matrix — dreamteam.v13j-fault-matrix/v1

[v13j_faults.py](../../analysis/alternatives/v13j_faults.py) runs nine base
scenarios with a recording hook. Every point the recorder reaches must be a
declared `(boundary, phase)` of the contract; each reached point is combined
with every fault kind the contract declares for it, the run is repeated with
that single fault, and the observed outcome is compared with the outcome the
contract's `expectedRule` predicts from the base trace (the state, committed
and completed rows, dispatched calls, pending draw and stop status at the
point, and the next safe point the base reaches). The expected values are
computed from the trace and the declared base outcome, never from the faulted
run.

| Base | Scenario (declared outcome) |
|---|---|
| A | two completed draws, server observed (COMPLETE) |
| B | the round health check before draw 2 answers unhealthy (PARTIAL health at HEALTH_CHECK) |
| C | engine opening fails at draw 2 (PARTIAL driver at DRAW_ACTIVE, engine_open) |
| D | per-draw call limit 0 (PARTIAL draw_call_limit at DRAW_RECORDED) |
| E | run call budget 1 (PARTIAL call_budget at DRAW_RECORDED) |
| F | one transport failure stops the next request (PARTIAL transport at DRAW_RECORDED) |
| G | the wall clock passes during draw 1 (PARTIAL wall_clock at DRAW_RECORDED) |
| H | the server signature changes before draw 2 (PARTIAL server_change at HEALTH_CHECK) |
| I | the pooled session release fails after both draws (PARTIAL driver at CLEANUP, cleanup FAILED) |

Fault kinds: `exception` (a recorder fault), `signal` (a real SIGINT sent to
the recording process with os.kill at the point), `persistence` (one EIO from
the filesystem operation), `persistent` (every later write, sync, replace,
rename, unlink and append in the archive directory fails, including any
recovery operation), `transport` and `guard_halt` (base A only, since they
change a scenario's own mechanics).

Result (`docs/v13j/fault-matrix/matrix.json`): **4,916 cases at 1,511
points; 4,916 agree; 0 false COMPLETE**; no declared point unreached.
Kinds: exception 1,509, signal 1,509, persistence 684, persistent 1,207,
transport 5, guard halt 2. Classifications: COMPLETE 36, PARTIAL 2,037,
INVALID 2,843. Rules: INTERRUPT 1,482, FAIL_CLOSED 2,375, NO_ARCHIVE 468,
HANDLED 486, COMMITTED 90, CLEANUP_FAILED 8, HEALTH_STOP 3, GUARD_STOP 2,
WORKFLOW_EVIDENCE 2.

Coverage is measured from the finalized archives, not counted from the
contract: all 19 transitions observed (including both seal-only terminal
transitions); all 22 admissible stop combinations observed (every stop code
in every admissible state, and the operator stop at each of its seven STOP
safe-point/state combinations); dispositions STOP, AFTER_STOP and
AFTER_SCHEDULE observed. Every case also checks, independently of its
expectation: no FALSE COMPLETE; analysis accepts exactly the valid archives
and suppresses comparisons for PARTIAL; the recorder's sealed state equals
the independent classification; a recording failure never leaves a sealed or
accepted archive; no durable row or dispatched call present at the point is
missing afterwards.

## Real-process signals and termination — dreamteam.v13j-signal-tests/v1

[v13j_signals.py](../../analysis/alternatives/v13j_signals.py): a child
process records a base; the parent delivers the signal. Result
(`docs/v13j/signals/results.json`): **326 of 326 agree, 0 false COMPLETE**.

| Kind | Cases | What |
|---|---|---|
| sigint | 32 | external SIGINT while the child is blocked at the point: every safe point, the transport windows, persistence phases, every finalization and seal phase |
| sigint-process-group | 4 | SIGINT to the child's whole process group (as a terminal Ctrl-C): engines are isolated and the draw completes; the stop is at the next safe point |
| sigint-line | 4 | real SIGINT on the four DT13I-01 source lines inside the unchanged V0.12b transport and V0.13h draw bookkeeping: no call or draw lost; the stop is at after_draw_persisted with modelCalls equal to the transport's actual invocations |
| sigkill-self | 276 | SIGKILL at every point of base A and every stop-sequence point of bases B and C: INVALID (unsealed) before the seal rename, sealed and classified by content from the rename on; durable rows and journal entries kept |
| sigkill-external | 7 | SIGKILL from the parent at a dispatch, a model call, the ledger replace, the final manifest write and three seal phases |
| sigterm-self | 3 | SIGTERM at a model call and around the seal rename |

## Loopback — dreamteam.v13j-loopback/v1

[v13j_loopback.py](../../analysis/alternatives/v13j_loopback.py): a local
HTTP server on localhost (ephemeral port) answers /health; the production
probe reads it; actual UTC timestamps and a fresh UUID; the verifier refuses
synthetic provenance. Result (`docs/v13j/loopback/results.json`, horo):
**12 of 12 agree** — healthy, explicit unhealthy, connection refused, closed
early, incomplete response, delayed within timeout, timeout, malformed JSON,
HTTP 503, healthy-with-error, server disappearing between checks, bad status
line. Recorder state, seal, verifier and analysis agree in every case.

## Regression witnesses — dreamteam.v13j-witnesses/v1

[v13j_witnesses.py](../../analysis/alternatives/v13j_witnesses.py):
**76 of 76 agree** (`docs/v13j/witnesses/results.json`): every DT13I-01, -02,
-03 and -05 counterexample of the V0.13i review (DT13I-04 and -06 are archive
forgeries, below), the V0.13h DT13H-01/-03 shapes and the V0.13g lifecycle
edges and partial probes. Each also checks that the archive's call count
equals the transport's actual invocations and that pooled sessions were
released exactly once.

## Contradictory archives — dreamteam.v13j-mutations/v1

[v13j_mutations.py](../../analysis/alternatives/v13j_mutations.py): ten
genuine bases (COMPLETE, health stop, aborted workflow, halt row, cleanup
failure, engine-open failure, operator stop before dispatch, operator stop
between draws, post-schedule interrupt, unrecorded draw); each mutant
recomputes commit bindings, the failure-event mirror and every seal hash,
count and binding, so only the targeted contradiction remains. **125 of 125
as expected** (`docs/v13j/mutations/results.json`): 123 rejected with a
targeted invariant and refused by analysis; one consistent erasure of a
completed call (row and journal, counts recomputed) rejected by model-free
replay; one positive control (call times shifted within their valid
intervals) accepted, confirming that timestamps stay outside replay identity.

## Record-only examples — dreamteam.v13j-lifecycle-examples/v1

[v13j_examples.py](../../analysis/alternatives/v13j_examples.py): 16
byte-reproducible archives under `docs/benchmark/v13j-dev/lifecycle-examples/`
(synthetic, labelled provenance), each with its classification, the public
result's interrupt summary and, when valid, the analysis; compared across
platforms by CI.
