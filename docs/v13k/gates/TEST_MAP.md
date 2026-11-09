# V0.13k test map

Independent review PENDING. The engineering tests do not authorize protocol lock, live inference or release.

| Requirement | Bindings | Tests and evidence |
| --- | --- | --- |
| Every persisted field and cross-field relationship | All 35 IDs; 293 named fixed fields and recursive descendants | `bindings_v1.json`; `test_inventory_covers_every_named_field_and_mutation_kind`; 350 full-analysis matrix cases with exact targets |
| Aborted-stage evidence shape and replay | B-ABORT-001/002, B-ENGINE-001/002, B-DECISION-001/002, B-BATCH-001 | `v13k_regressions.py` DT13J-01; inherited all-stage witnesses; matrix; journal-prefix reconstruction when calls are null |
| Cleanup order, multiplicity, stage, interval and seal | B-CLEANUP-001, B-INTERRUPT-001, B-SEAL-002 | DT13J-02 regressions; cleanup and interrupt matrix rows; real signal tests |
| Scheduled context identity and limits with unknown calls | B-CALL-001/002/003/004, B-LEDGER-001 | DT13J-03 null-array over-bound, valid-hex false request, nonhex request, impossible response regressions; full lifecycle guard bases; matrix |
| Completed results survive handled persistence failures | B-HANDOFF-001, B-LEDGER-002, B-JOURNAL-002, B-FAILURE-001 | Six cases in `v13k_preservation.py`; `test_completed_draw_preservation`; pending-aware complete fault matrix; handoff real signals |
| Trusted exception class identity | B-HEALTH-004, B-HEALTH-002 | Every trusted class and impostor, wrapped/unwrapped, overloaded class equality in `test_v13k.py`; DT13J-05 regressions |
| Health bytes/category and initial mirror | B-HEALTH-003/005 | Duplicate-key/nonfinite supplementary regressions; nested JSON boolean/numeric and mirror tests; loopback sockets |
| Existing completion seal and commit behavior | B-SEAL-001/002/003/004 | Exact old finalization subtree check; lifecycle fault and real termination cases; re-sealed mutations |
| Independent verifier | All reconstructable bindings | Syntax-tree import test; shared contract data only; typed canonical replay; full analysis for every matrix case |
| Historical design and recordings | All protected baseline groups | `v13k_preserved.py`; 3,172 protected file hashes; historical replay and fidelity gates; unchanged human-review material at ee987a1 |

## Reproduction

On a host with the built engine and recorded historical replays, set absolute paths:

```sh
R=/home/george/dt-v013k
A="$R/analysis/alternatives"
B="$R/dt"
O=/home/george/dt-audit-kept/v013k/reproduction
mkdir -p "$O/tmp"
export TMPDIR="$O/tmp" PYTHONDONTWRITEBYTECODE=1
python3 -I -B "$A/v13k_fidelity.py" --binary "$B" --contexts "$R/docs/benchmark/v13g-dev/contexts" --replays "$R" --scratch "$O/fidelity" --out "$O/fidelity.json"
python3 -I -B "$A/v13k_binding_matrix.py" --binary "$B" --scratch "$O/matrix" --out "$O/matrix.json"
python3 -I -B "$A/v13k_regressions.py" --binary "$B" --scratch "$O/regressions" --out "$O/regressions.json"
python3 -I -B "$A/v13k_preservation.py" --binary "$B" --scratch "$O/preservation" --out "$O/preservation.json"
python3 -I -B "$A/v13k_faults.py" run --binary "$B" --scratch "$O/faults" --out "$O/faults.json" --jobs 4
python3 -I -B "$A/v13k_signals.py" run --binary "$B" --scratch "$O/signals" --out "$O/signals.json" --jobs 4
python3 -I -B "$A/v13k_loopback.py" --binary "$B" --scratch "$O/loopback" --out "$O/loopback.json"
python3 -I -B "$A/v13k_witnesses.py" --binary "$B" --scratch "$O/witnesses" --out "$O/witnesses.json"
python3 -I -B "$A/v13k_mutations.py" --binary "$B" --scratch "$O/mutations" --out "$O/mutations.json"
DREAMTEAM_BINARY="$B" python3 -B -m unittest discover -s "$A" -p test_v13k.py
python3 -B -m unittest discover -s "$A" -p test_lifecycle_v3_spec.py
python3 -I -B "$A/v13k_preserved.py"
python3 -I -B "$A/v13k_identities.py" --check
```

Use fresh explicitly named scratch directories. Real signals and loopback run on horo. On the Pi run only focused tests under the owner's memory scope. Test archives use declared synthetic provenance; their full analysis is invoked through `Fixture.analyze`, which explicitly admits fixtures. Ordinary command-line analysis continues to reject synthetic provenance. None of these scripts contacts a model endpoint.
