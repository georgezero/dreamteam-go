# v0.2-eval-5 reserved confirmatory protocol: lock record

**Locked 2026-10-21. NOT RUN. The reserved run is NOT AUTHORIZED by this
lock**: it needs the owner's separate authorization, recorded before it
starts.

This record pins what the one reserved run must be. It instantiates the
locked protocol, `../clinical-5c4n/reserved-validation-protocol.md`
(§8), and changes none of its rules. Where the two differ, the protocol
governs and the difference is a defect of this record.

**Basis.** The owner's conditional GO after the 5C.4o release-gate audit
(2026-10-20): M1, M3 and M6 independently passed; O1 and O2 closed by
targeted maintainer remediation and checks (662a021), not independently
re-audited; the T5a output made independent of map iteration order
before the lock (7bf1ed6).

## 1. Source identity, and what this lock does not hash

A commit cannot contain its own hash, so the lock is pinned by its
**parent** and by the **content** of what it adds:

| Item | Value |
|---|---|
| Parent commit (the audited code plus the two targeted changes) | `7bf1ed6` (0.2.0-dev.26) |
| Lock commit | the commit that adds this file, with `7bf1ed6` as its only parent; its hash is recorded in the run record, not here |
| Engine version recorded by the run | `0.2.0-dev.26` |
| `sourceSHA256` the run must record (`sourceDigest`: every `internal/eval/*.go`, in byte order of name, each as `"<name> <length>\n"` then its bytes, SHA-256) | `5cd6a438a3c2e741f691815ba96e805f022f39f34a51f131d8a2164f9a07588b` |
| Toolchain | `go1.27.1 linux/arm64`, `GOTOOLCHAIN=local` |
| Test binary built by the command in §3 | SHA-256 `5edee74db6bb36314fd5dfd8357ac918714c850bbd0be931f5101788a183130d` |

**Not hashed by this record:** this record itself, and
`lock.sha256` beside it. Neither is read by the execution or compiled
into the binary (the binary and `sourceSHA256` depend only on
`internal/eval` and its imports), so their digests cannot change
anything they pin. `lock.sha256` lists, for `sha256sum -c`, the five
inputs of §2, `go.mod`, `go.sum` and every `internal/eval/*.go` (the
files `sourceSHA256` is computed from), each with its digest at the
lock.

**The binary's digest is reproducible**: `-trimpath` removes the
checkout's path, a test binary carries no VCS stamp, and
`CGO_ENABLED=0` removes the C toolchain. The same digest was obtained
from the working tree and from a `git archive` of the tree in another
directory. These build flags change no code; they are this record's
instantiation of §8's `go test -c` so that the executable can be pinned.

## 2. Pinned inputs

The execution verifies each runtime input against the digest compiled
into the binary (`certifiedInputs`) before it constructs anything, and
the manifest's `policy.sha256` against the candidate's.

| Role | File | SHA-256 |
|---|---|---|
| Candidate policy (v0.2-eval-5) | `docs/calibration/clinical-5c4b/candidate-policy-v5.json` | `83d481b7040b69d78a58de68afc938a588b5b37ba396438509f38c56ca163948` |
| Scenario manifest (version 3) | `docs/calibration/clinical-5c4l/scenario-manifest.json` | `250bc7fc835d99140212505e0e79f74f2da1484dbe856fea6a5b93eec9153ef4` |
| Protocol (locked bytes) | `docs/calibration/clinical-5c4n/reserved-validation-protocol.md` | `12d1eda4f8543e64cdbf1d5118339b65bc6a620e8ee3f6cb01dfd974d0c91a0d` |
| Policy in force (v0.2-eval-4) | `docs/evaluation/policy-v0.2-eval-4.json` | `4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3` |
| Model / fidelity profile (simulator calibration) | `internal/eval/testdata/sim-profile-5c4b.json` | `ac8621889813a2d582b4b19674f04871c5711d27d0ac1620839ba01df4596100` |

The fidelity model itself (shared pre-takeover history, prefix-stable
draws, 45-day follow-up) is code in `internal/eval`, pinned by
`sourceSHA256` and the binary digest.

**The protocol at the lock.** Its pre-lock bytes (7bf1ed6) had SHA-256
`094bdab86eca795223f748be72e862ea4815c3156cfdd6d6a577ca104a658bad`. The
lock changed only its status statements, as §8 provides:
* the title's "(proposed)" to "(locked)", and §8's heading to "locked;
  not run";
* the status block to LOCKED, NOT RUN, naming this record and the basis
  above;
* the two sentences saying the build refuses the reserved family, to
  say that from the lock commit it admits it, for the authorized run.

No rule, criterion, tolerance, stream, scenario or command changed.

## 3. The execution

**Seed family:** PCG(F = **20261201**, ·), every stream as in protocol
§2 and the manifest. Nothing has run on it.

**Paths:** `$BIN` is `$HOME/dt-v5-run/bin`, outside the repository. The
lock directory of §8 is `docs/calibration/clinical-v5-reserved/`.

**Preconditions**, each checked and recorded in the run record:
1. the owner's authorization of the run is recorded;
2. the checkout is the lock commit, clean (`git status --porcelain`
   empty), whose parent is `7bf1ed6`;
3. `sha256sum -c docs/calibration/clinical-v5-lock/lock.sha256` passes;
4. every committed test passes at the lock commit
   (`go test -count=1 -p 1 ./...`), with no `DT_MANIFEST_*` or
   `DT_CRITERIA_OUT` variable set;
5. the binary built below has the digest in §1;
6. neither output file exists.

**Command** (one process, memory-capped; from the repository root):

```
export GOTOOLCHAIN=local CGO_ENABLED=0
go test -c -trimpath -o "$HOME/dt-v5-run/bin/eval.test" ./internal/eval
sha256sum "$HOME/dt-v5-run/bin/eval.test"   # must equal §1
mkdir -p docs/calibration/clinical-v5-reserved
cd internal/eval
env -u DT_MANIFEST_SEEDS -u DT_MANIFEST_SCENARIOS -u DT_MANIFEST_IN \
  DT_MANIFEST_FAMILY=20261201 \
  DT_MANIFEST_OUT=../../docs/calibration/clinical-v5-reserved/reserved-run.jsonl \
  DT_CRITERIA_OUT=../../docs/calibration/clinical-v5-reserved/reserved-criteria.json \
  GOMEMLIMIT=5GiB systemd-run --user --scope -q -p MemoryMax=6G nice -n 10 \
  "$HOME/dt-v5-run/bin/eval.test" -test.run 'TestReservedManifestRun$' -test.v -test.timeout 2000m \
  2>&1 | tee ../../docs/calibration/clinical-v5-reserved/reserved-run.log
```

The memory cap and `nice` are this machine's operating limits; they do
not change the computation.

**Expected:** 49 scenarios at their manifest seeds (61,000 scenario seed
assignments, 56,000 distinct base pairs), **3,904 result lines** (3,896
complete groups and 8 remainders), computed, judged, adjudicated and
certified in memory before either file is written. The invocation must
read `confirmatory`, with `sourceSHA256` and `engine` as in §1 and the
inputs as in §2.

## 4. What decides

**Mandatory criteria** (protocol §5; all must PASS for `certified GO`):

| Criterion | PASS |
|---|---|
| T1 stable-practice false rejection | at most 3 of 62 groups rejected in every stable scenario |
| T2 harmed-seed detection | point estimate at least 0.95, with at least 200 harmed seeds |
| T2-conformance | the deterministic conformance holds in every group |
| T2G material global deterioration | at least 23 of 25 groups rejected in every material scenario |
| T3 zero-tolerance admissibility | no violating seed for any absolute limit in any of its scenarios |
| T4 witnesses | every group of every witness meets its expectation |
| T5a near-margin conformance | every δ's observed rate within 4 cluster standard errors, plus 0.02, of its prediction |
| T5b near-margin tolerance (approved 2026-10-19) | at most **0.40** at δ = 2.1 points and at most **0.01** at δ = 3 points (manifest `t5Tolerances`) |
| T6 safeguard false flags | at most 19 flagged seeds in 10,000 in every priority |
| T7 severity-window and safety evaluability | at most 3 of 62 groups in every one of its scenarios |
| T8 construction integrity | every condition of protocol §5 holds |

**Certification** (protocol §6): only this trusted in-process execution
on the reserved family at every scenario's manifest seeds is
`confirmatory`. It is `certified GO: confirmation succeeded` when every
mandatory criterion PASSes, `certified NO GO: confirmation failed` when
any FAILs, and `certified NO GO: confirmation not achieved` when any is
INCONCLUSIVE.

**Scope** (protocol, CERTIFICATION SCOPE): the certification is valid
only for records generated and adjudicated in that one trusted process.
Externally editable records and imported measurement summaries cannot
certify. **The accepted study-swap limitation**: the evaluator cannot
detect two same-request orders, one wanted by the horizon and one after
it, both imaged after the later wanted time, exchanging their studies
and resolutions inside one record. The owner accepts it only for
trusted in-process evaluation. Any evaluation of external, imported,
edited or untrusted records needs an authenticated order-to-study
mapping, or an equivalent independently trusted event history, before
a favorable v5 verdict can be certified. No clinical rule changes for
it.

## 5. Once, and after

* **One execution.** The reserved family is consumed by the first
  execution that starts constructing on it, whatever its outcome. It is
  never rerun, resumed or repeated, in whole or in part.
* **Failure or partial run.** An execution stopped before it writes
  (a refused input or setting, a crash, a timeout) has no
  certification: it is recorded as **not certified**, with its log. An
  execution that completes with incomplete coverage or a missing or
  malformed measurement writes its files, reads **certified NO GO:
  confirmation not achieved**, and exits failing (protocol §8). Neither
  is retried on the reserved family. A certified NO GO on complete
  coverage is a valid outcome.
* **No post-run tuning.** A failed confirmation is a documented **NO
  GO**. It is never followed by changing a threshold, tolerance, window,
  criterion, scenario or the candidate against these streams, nor by
  retuning against the reserved seeds.
* **Offline analysis is never certification.** Re-reading the written
  lines (`TestReservedCriteria`) reads `non-certifying analysis` and
  `not certified`.
* **Evidence committed afterwards, unchanged:** `reserved-run.jsonl`,
  `reserved-criteria.json` and `reserved-run.log` in
  `docs/calibration/clinical-v5-reserved/`, their SHA-256 digests as the
  execution logged them, the lock commit's hash, the binary's digest and
  the preconditions' record.

## 6. The lock commit's changes

Besides this record, `lock.sha256` and the repository's progress and
measurement logs (`docs/PROGRESS.md`, `docs/MEASUREMENTS.md`, outside
every pinned set), exactly what protocol §8 provides:
* `familyAllowed` admits F = 20261201 (and `reservedFamily`'s comment
  says it is run once, from the lock commit);
* the two assertions of the refusal become assertions of the admission,
  constructing nothing: `TestReservedManifestRefusesReserved` and the
  last check of `TestCertifiedExecutionSmall`;
* the protocol's status statements (§2 above);
* the protocol's digest in `certifiedInputs`, re-pinned to the locked
  bytes.

Nothing in the evaluator, the adjudicator, the fidelity model, the
manifest or the candidate changed.
