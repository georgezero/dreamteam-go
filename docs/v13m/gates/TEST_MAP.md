# V0.13m-lite test map

The engineering tests do not authorize protocol lock, live inference or release. Human presentation review PENDING.

| Requirement | Relationships | Tests and evidence |
| --- | --- | --- |
| Strict response JSON (DT13L-01) | R-JSON-004, R-JSON-003 | `test_v13m.StrictResponse` (duplicate, escaped-equivalent, nested and deep-nested keys; NaN/Infinity/-Infinity; 1e999/-1e400; trailing content; truncated; BOM; Unicode literal/escape; 2^80+1; signed zero and exponent forms; ordinary responses); `v13m_dt13l.py` DT13L-01 (15 resealed archives, 17 genuine wire recordings, ordinary CLI) |
| Wall-clock causality (DT13L-02) | R-HALT-003, R-HALT-001 | `v13m_dt13l.py` DT13L-02 (draw-start and request-start: zero message, one below threshold, at threshold, above elapsed; ordinary CLI with real UTC and with linked clocks: valid halts, elapsed beyond timestamps, message below threshold); `test_v13m.DT13L` sample |
| Fail-closed persistence (DT13L-03/04) | R-HANDOFF-003, acceptance standard | `v13m_dt13l.py` nested recovery faults (bookkeeping, entry, retention construction), recorder without the guard (verifier defense), persistent storage failure; `test_v13m.AcceptanceStandard`, `test_v13m.DT13L`; lifecycle example `unrecorded-draw-retained-calls-complete-refused` |
| No false COMPLETE / contradictory PARTIAL | all | successor fault matrix (zero false COMPLETE), binding matrix (zero survivors), witnesses, mutations, regressions, signals, loopback |
| No retry / replacement draw | fail-closed policy | transport invocations = journal dispatches and no later draw in every DT13L-03/04 and policy case; fault matrix and witness call-count checks |
| Exact replay of valid draws | B-DECISION, B-ENGINE | positive controls; 66-unit fidelity; lifecycle examples |
| Primary-outcome counterexample | estimand | `v13m_dt13l.py` primary case: control abandonment and treatment `select:no_action` both submit `[]`; total variation and per-identity difference zero |
| Earlier fixes preserved | DT13K-01..07 | `v13l_dt13k.py` run on the V0.13m implementation ([on-v13m](../dt13k/on-v13m.json)); `test_v13m.Remediation` |
| Historical design and recordings | baseline groups | `v13m_preserved.py` (3,606 protected files); V0.13l and earlier stage gates; human-review freeze |

## Reproduction

With the built engine and the recorded historical replays (`./dt benchmark suite run docs/benchmark/v0{8,9}c-dev/eval-replay-*.json -out llm-v0{8,9}c-replay-*`), absolute paths and fresh scratch directories:

```sh
python3 -I -B "$A/v13m_dt13l.py" --binary "$B" --implementation v13l   --scratch "$O/l" --out "$O/before-v13l.json"
python3 -I -B "$A/v13m_dt13l.py" --binary "$B" --implementation prefix --scratch "$O/p" --out "$O/before-prefix-successor.json"
python3 -I -B "$A/v13m_dt13l.py" --binary "$B" --implementation v13m   --scratch "$O/m" --out "$O/after-v13m.json"
python3 -I -B "$A/v13m_binding_matrix.py" --binary "$B" --scratch "$O/bm" --out "$O/bm.json"
python3 -I -B "$A/v13m_preservation.py" --binary "$B" --scratch "$O/pres" --out "$O/pres.json" --jobs 8
python3 -I -B "$A/v13m_faults.py" run --binary "$B" --scratch "$O/faults" --out "$O/faults.json" --jobs 8
python3 -I -B "$A/v13m_signals.py" run --binary "$B" --scratch "$O/sig" --out "$O/sig.json" --jobs 4
python3 -I -B "$A/v13m_loopback.py" --binary "$B" --scratch "$O/lb" --out "$O/lb.json"
python3 -I -B "$A/v13m_regressions.py" --binary "$B" --scratch "$O/reg" --out "$O/reg.json"
python3 -I -B "$A/v13m_witnesses.py" --binary "$B" --scratch "$O/wit" --out "$O/wit.json"
python3 -I -B "$A/v13m_mutations.py" --binary "$B" --scratch "$O/mut" --out "$O/mut.json"
python3 -I -B "$A/v13m_fidelity.py" --binary "$B" --contexts "$R/docs/benchmark/v13g-dev/contexts" --replays "$R" --scratch "$O/fid" --out "$O/fid.json"
DREAMTEAM_BINARY="$B" python3 -B -m unittest discover -s "$A" -p test_v13m.py
python3 -I -B "$A/v13m_preserved.py"
python3 -I -B "$A/v13m_identities.py" --check
```

Real signals and loopback run on a host that permits sockets (horo). No script contacts a model endpoint.
