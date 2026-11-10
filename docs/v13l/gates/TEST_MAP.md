# V0.13l test map

Independent review PENDING. The engineering tests do not authorize protocol lock, live inference or release.

| Requirement | Relationships / bindings | Tests and evidence |
| --- | --- | --- |
| Strict JSON consistency (DT13K-01) | R-JSON-001/002/003, R-RESPONSE-001, B-TRANSPORT-001 | `test_v13l.StrictJson` (false/0, true/1, 1/1.0, 0.0/-0.0, null/absent, numeric strings, large integers, nested objects, reordered keys, duplicate keys, non-finite numbers, BOM); `v13l_dt13k.py` DT13K-01 (ten cases incl. ordinary CLI) and two positive controls; relationship matrix |
| Failure binding (DT13K-02) | R-FAILURE-001/002/003, R-RESPONSE-002 | `v13l_dt13k.py` DT13K-02 (six cases on genuine initial_digests, workflow, actual final_digests, engine_close and raised-transport bases); relationship matrix; relabelled inherited mutations |
| Halt reconstruction (DT13K-03) | R-HALT-001/002, B-STOP-002 | `v13l_dt13k.py` DT13K-03 (three cases); relationship matrix (aborted and interrupted-row halts) |
| Completed-draw handoff (DT13K-04) | R-HANDOFF-001/002, B-HANDOFF-001 | `v13l_preservation.py`: six inherited cases and exhaustive line enumeration (258 sites from completion to durable commit); `v13l_dt13k.py` DT13K-04 (three boundaries); `test_ownership_boundaries_sample` |
| Model-response handoff (DT13K-05) | R-RESPONSE-003, R-DISPATCH-001 | `v13l_preservation.py`: nine write/fsync/close × retention cases and exhaustive line enumeration (62 sites from receipt to durable record); `v13l_dt13k.py` DT13K-05; `test_response_handoff_preserves_a_received_response`; fault matrix and signals check received responses |
| Conservative INVALID (DT13K-06) | lifecycle.postAnswerFailure | `v13l_dt13k.py` DT13K-06; [known limitations](../KNOWN_LIMITATIONS.md) |
| Generated fault contexts (DT13K-07) | R-HANDOFF-001, R-RESPONSE-003 | `CONTEXT_KEYS`/`PRESERVATION_KEYS` in `v13l_faults.py`; `test_v13l.GeneratedContexts`; DT13K-07 witness; full fault matrix |
| Generalized bindings (Phase 9) | all 17 R-* and 35 B-* | `bindings_v2.json`; `lifecycle_v4_spec.relationships()` and `test_lifecycle_v4_spec`; `v13l_binding_matrix.py` (350 + 119 full analyses); `test_v13l.Relationships` |
| Lifecycle, seal, signals, loopback | all lifecycle bindings | full fault matrix; real-process signals; real localhost loopback; `test_v13l.Lifecycle` |
| Independent verifier | all reconstructable bindings | syntax-tree import test; shared contract data only; strict typed replay |
| Historical design and recordings | all protected baseline groups | `v13l_preserved.py` (3,387 protected files); historical replay, fidelity and earlier-stage gates; human-review freeze at `ee987a1` |

## Reproduction

On a host with the built engine and recorded historical replays (`./dt benchmark suite run docs/benchmark/v0{8,9}c-dev/eval-replay-*.json -out llm-v0{8,9}c-replay-*`), set absolute paths:

```sh
R=/path/to/fresh/checkout
A="$R/analysis/alternatives"
B="$R/dt"
O=/path/to/fresh/scratch
mkdir -p "$O/tmp"
export TMPDIR="$O/tmp" PYTHONDONTWRITEBYTECODE=1
python3 -I -B "$A/v13l_dt13k.py" --binary "$B" --implementation v13k --scratch "$O/dt13k-k" --out "$O/before-v13k.json"
python3 -I -B "$A/v13l_dt13k.py" --binary "$B" --implementation v13l --scratch "$O/dt13k-l" --out "$O/after-v13l.json"
python3 -I -B "$A/v13l_binding_matrix.py" --binary "$B" --scratch "$O/matrix" --out "$O/matrix.json"
python3 -I -B "$A/v13l_preservation.py" --binary "$B" --scratch "$O/preservation" --out "$O/preservation.json" --jobs 8
python3 -I -B "$A/v13l_faults.py" run --binary "$B" --scratch "$O/faults" --out "$O/faults.json" --jobs 8
python3 -I -B "$A/v13l_signals.py" run --binary "$B" --scratch "$O/signals" --out "$O/signals.json" --jobs 4
python3 -I -B "$A/v13l_loopback.py" --binary "$B" --scratch "$O/loopback" --out "$O/loopback.json"
python3 -I -B "$A/v13l_regressions.py" --binary "$B" --scratch "$O/regressions" --out "$O/regressions.json"
python3 -I -B "$A/v13l_witnesses.py" --binary "$B" --scratch "$O/witnesses" --out "$O/witnesses.json"
python3 -I -B "$A/v13l_mutations.py" --binary "$B" --scratch "$O/mutations" --out "$O/mutations.json"
python3 -I -B "$A/v13l_fidelity.py" --binary "$B" --contexts "$R/docs/benchmark/v13g-dev/contexts" --replays "$R" --scratch "$O/fidelity" --out "$O/fidelity.json"
DREAMTEAM_BINARY="$B" python3 -B -m unittest discover -s "$A" -p test_v13l.py
python3 -B -m unittest discover -s "$A" -p test_lifecycle_v4_spec.py
python3 -I -B "$A/v13l_preserved.py"
python3 -I -B "$A/v13l_identities.py" --check
```

Use fresh explicitly named scratch directories. Real signals and loopback run on horo; on the Pi run only focused tests under the memory scope. Test archives use declared synthetic provenance and are analysed through `Fixture.analyze`; ordinary command-line analysis rejects synthetic provenance (the DT13K-01 CLI witness uses a genuine, non-fixture recording). No script contacts a model endpoint.
