# V0.13o — DT13M-01: acceptance standard pinned in the protocol

**Status: FIX IMPLEMENTED; NARROW INDEPENDENT RECHECK PENDING.** Human
presentation review: **PENDING**. Protocol lock: **NO GO**. Live inference:
**NO GO**. V0.13.0 release: **NO GO**.

## Finding

The V0.13n bounded independent technical check
([docs/audits/v0.13n-bounded-technical-check.md](../audits/v0.13n-bounded-technical-check.md),
commit `4567e3c`) returned NOT ACCEPT / Technical NO GO on one MAJOR finding:

> DT13M-01: the protocol does not pin the clock-reconciliation acceptance
> policy. The generated protocol omits an acceptance-standard identity/hash
> and the 60-second operational premise; ordinary analysis loads the ambient
> acceptance file.

The tolerance itself was implemented as approved: it reconciles timestamps
only, and a discrepancy over 60 s is rejected. The third condition, "pinned in
the protocol", was not met.

## Fix (successor; V0.13m-lite unchanged)

| File | Role |
|---|---|
| `analysis/alternatives/v13o.py` | Protocol successor `dreamteam.alternatives-experiment/v10`, id `v13o-alt-1`: the V0.13m-lite protocol plus an `acceptance` field. The recorder (`v13m.py`) is imported unchanged; `run_schedule`, `schedule`, `check` and `run` refuse a protocol whose binding does not match the acceptance file. |
| `analysis/alternatives/v13o_verify.py` | Verifier successor: the V0.13m-lite verifier (`v13m_verify.py`) unchanged, behind rule **R-PIN-001**. It imports no recorder. |
| `docs/benchmark/v13o-dev/protocol-draft.json`, `schedule-draft.json` | The draft protocol and schedule (262 entries, identical to V0.13m-lite's; 802-call ceiling; 14,400 s). |
| `analysis/alternatives/v13o_pin.py`, `docs/v13o/pin/results.json` | Before/after witnesses (genuine-provenance recordings, ordinary command line). |
| `analysis/alternatives/test_v13o.py` | Unit tests. |
| `docs/v13o/IDENTITIES.json` | SHA-256 pins. |

The protocol's `acceptance` field:

```json
{
 "id": "dreamteam.acceptance-standard/v1",
 "file": "analysis/alternatives/acceptance_v1.json",
 "sha256": "c6af89ede52284a023f1413ea7b3d46b38a7da516ccd87da92841efe1195a721",
 "clockReconciliation": {
  "rule": "R-HALT-003",
  "toleranceSeconds": 60,
  "scope": "reconciles each halt's evidence elapsedSeconds with the recorded wall-clock (SYSTEM_UTC) journal and call-journal timestamps only; it does not change the four-hour request-start guard (stop.wallClockSeconds, request.timeoutSeconds), which is applied without tolerance",
  "onExcess": "INVALID: a discrepancy above toleranceSeconds rejects the archive, with no retry, resumption or replacement draw",
  "basis": "operational assumption that the monotonic and UTC clocks of the recording host agree within toleranceSeconds over the run; it is not a measured clock-accuracy guarantee"
 }
}
```

The protocol differs from the V0.13m-lite draft only in `schema`, `id` and
this field. `acceptance_v1.json` is unchanged (`c6af89ed…`).

**R-PIN-001.** A run archive binds its protocol by SHA-256, so it binds the
acceptance standard through the protocol. Before classifying or analysing,
the verifier requires that:

- the protocol names the standard and file;
- its hash equals the exact bytes of the acceptance file that the verifier
  reads;
- those bytes parse strictly to the same relationships that the V0.13m-lite
  verifier applies;
- the stated tolerance is the applied integer;
- the scope, excess and operational-assumption statements match verbatim.

If any of these differ, classification is INVALID and analysis is refused.

## Witnesses (`docs/v13o/pin/results.json`, deterministic)

These are genuine-provenance wall-clock halts: a scripted model, a monotonic
clock and a linked SYSTEM_UTC clock, with no model endpoint. Each is analysed
through the ordinary command line. Every mutation recomputes every hash and
the seal.

| Case | V0.13m-lite | V0.13o |
|---|---|---|
| Genuine wall-clock halt (control) | PARTIAL, accepted | PARTIAL, accepted |
| elapsedSeconds 59.9 s beyond the timestamp bound | PARTIAL, accepted | PARTIAL, accepted |
| elapsedSeconds 60.1 s beyond the timestamp bound | INVALID (R-HALT-003), refused | INVALID (R-HALT-003), refused |
| Reviewer's diagnostic: byte-identical verifier copies, ambient acceptance data 120 s, 60.1 s archive | **PARTIAL, accepted** | **INVALID (R-PIN-001), refused** |
| Protocol without the binding | — | INVALID (R-PIN-001), refused; recorder refuses the protocol |
| Protocol pinning another acceptance hash | — | INVALID (R-PIN-001), refused; recorder refuses |
| Protocol stating 120 s | — | INVALID (R-PIN-001), refused; recorder refuses |
| Protocol weakening the operational-assumption statement | — | INVALID (R-PIN-001), refused; recorder refuses |
| Protocol widening the scope to the four-hour guard | — | INVALID (R-PIN-001), refused; recorder refuses |

In the unit tests, 16 protocol alterations are refused by both the recorder
and the verifier. These include a float, boolean or string tolerance, extra
or missing keys, an upper-case hash and a retry-on-excess statement. Altered
acceptance files are also refused, including a duplicate key, a trailing
newline and a 120 s value.

## Preservation

There are no edits to existing files except `.github/workflows/ci.yml` (new
V0.13o step), `docs/PROGRESS.md` and `docs/v13n/README.md`. The following
pass:

- `v13m_preserved.py`: 3,606 protected files.
- `v13m_identities.py --check`.

The human-review freeze (`ee987a1`) is unchanged. Model-visible content does
not change.

## Remaining

1. A narrow independent recheck of DT13M-01 (this binding only).
2. The independent human presentation review (PENDING).
3. Lock candidate and owner lock gate (not started). Nothing is locked
   automatically.
