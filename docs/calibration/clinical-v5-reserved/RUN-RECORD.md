# v0.2-eval-5 reserved confirmatory run: record

**Outcome: `certified NO GO: confirmation failed`.** The one authorized
execution of PCG(20261201, ·) completed with full coverage; ten of the
eleven mandatory criteria PASS and **T3 FAILs** (one seed of 10,000
stable-practice seeds breaches the urgent past-due absolute limit). Under
the locked protocol this is a documented NO GO. It is not rerun, and
nothing is tuned against it.

The outputs in this directory are exactly as the execution wrote them.

## 1. Identity

| Item | Value |
|---|---|
| Lock commit | `ff5b566` (parent `7bf1ed6`, 0.2.0-dev.26) |
| Checked-out commit | `38881d4` (lock amendment 1: host and authorization; changes no file in `lock.sha256`) |
| Owner authorization | 2026-10-21, one execution, on `horo` with the linux/amd64 build (`../clinical-v5-lock/AMENDMENT-1.md`) |
| Host | `horo`: Intel Core i7-10700, 16 threads, 31 GB, Linux 6.12.86+deb13-amd64 x86_64 |
| Executable | linux/amd64 test binary, SHA-256 `bf9e1c46a90022331efa571ff66f83e2d38ae4e73aa3f4c93f63a9677724e088` (go1.27.1, `-trimpath`, `CGO_ENABLED=0`, `GOAMD64=v1`, from the lock commit's source) |
| `sourceSHA256` (recorded by the run) | `5cd6a438a3c2e741f691815ba96e805f022f39f34a51f131d8a2164f9a07588b` |
| Engine (recorded) | `0.2.0-dev.26` |
| Family (recorded) | 20261201 |
| Inputs (recorded) | candidate `83d481b7…3948`, manifest `250bc7fc…3ef4`, protocol `12d1eda4…0a0d`, policy in force `4e277c72…b3`, simulator profile `ac862188…4100`: each the pinned digest |
| Start / end (UTC) | 2026-10-21T15:21:30Z / 2026-10-21T17:31:45Z (7,815 s in the test) |
| Exit status | 0 (the service: `Result=success`) |

**Command** (`preconditions/run.sh`): `LOCK.md` §3's run command, as
written (memory cap 6 GB, `GOMEMLIMIT=5GiB`, `nice -n 10`,
`-test.timeout 2000m`, `DT_MANIFEST_SEEDS`, `DT_MANIFEST_SCENARIOS` and
`DT_MANIFEST_IN` unset, `DT_MANIFEST_FAMILY=20261201`), with the
prebuilt binary of amendment 1 in place of the build step, its digest
checked first, run as a user service on `horo`.

## 2. Outputs

| File | SHA-256 (as the execution logged it) |
|---|---|
| `reserved-run.jsonl` (3,904 lines) | `86826be5fc29b6288ee492dbdf8229ff07ef72b912227dbfd4409e0965808619` |
| `reserved-criteria.json` | `674af45cb3eb5c8dd5365b97861dfd9884781823c815103c72ca334380d2cc7e` |
| `reserved-run.log` (the test's output) | `24fc4370389e1be5591f590acb979025399b7223e1c2d80aad6be133b8fb5e7c` |

**Complete.** 3,904 lines, as expected (3,896 complete groups and 8
remainders); the execution reported no coverage or measurement problem.
Invocation `confirmatory`; certification
`certified NO GO: confirmation failed`, failing `["T3"]`, none
INCONCLUSIVE.

## 3. The criteria

| Criterion | Result | Detail |
|---|---|---|
| T1 stable false rejection | PASS | S-A85 1 of 62 (by `safety`), S-B85, S-C85, S-Cs85 0 of 62; limit 3 |
| T2 harmed-seed detection | PASS | 447 of 447 harmed seeds flagged (point estimate 1.00 ≥ 0.95, at least 200 harmed) |
| T2-conformance | PASS | |
| T2G material deterioration | PASS | every material scenario (D-85, IL40-85, K-92, L20-85) 25 of 25 rejected |
| **T3 zero-tolerance admissibility** | **FAIL** | **S-A85: one seed violates `pastDue urgent` (seed 259, group 6, seeds 240–279)**; S-B85, S-C85, S-Cs85 none |
| T4 witnesses | PASS | all 14 witnesses, 25 of 25 groups each |
| T5a near-margin conformance | PASS | NM0 1.00 (pred. 1.00), NM1 1.00 (1.00), NM2 0.49 (0.506), NM2.1 0.288 (0.301), NM3 0 (0) |
| T5b tolerance | PASS | 0.288 at +2.1 points (≤ 0.40), 0 at +3 points (≤ 0.01) |
| T6 safeguard false flags | PASS | 0 flags in 10,000 seeds, every priority (limit 19) |
| T7 evaluability | PASS | 0 insufficient groups in every stable scenario (limit 3) |
| T8 construction integrity | PASS | |

**Clinical safety.** On the stable practices the candidate made one
admissibility breach: seed 259 of S-A85 has an urgent order past its
absolute limit. The candidate itself rejected that group under its
safety rule (T1's only rejection), but T3 admits no violating seed at
all, so the confirmation fails. Every harmful scenario was detected
(T2, T2G), every witness behaved as specified (T4), and the safeguard
raised no false flag (T6).

**Statistical acceptance.** T1, T5a, T5b, T6 and T7 PASS within their
pre-registered limits; the near-margin acceptance rates conform to
their predictions.

## 4. Verification after the run (no evaluation changed)

* **Digests:** the three files' SHA-256 on `horo` and after copying
  match the execution's log.
* **Offline re-read** (`TestReservedCriteria`, non-certifying):
  * by the amd64 binary on `horo`: every criterion identical to
    `reserved-criteria.json`, byte for byte in value;
  * by the lock's arm64 binary (`5edee74d…130d`): every status
    identical; some reported statistics in T1, T2 and T5a differ in the
    last one or two digits (relative 1e-16 to 1e-14: confidence-interval
    bounds, T2's orientation-only lower bound, NM2's cluster standard
    error). Their special functions round differently on the two
    architectures. No decision depends on them at that scale. The
    certified values are the amd64 execution's.
* **Determinism before the run:** the amd64 binary's development lines
  were byte-identical to the arm64 build's in three passes
  (amendment 1).

## 5. Deviations and events

* **The host and executable** differ from `LOCK.md`'s, by owner-approved
  amendment 1 before the run (linux/amd64 on `horo`).
* **An aborted launch, before execution.** The first start of the run
  service (2026-10-21T15:20:25Z) stopped inside its own precondition
  check: the check for an existing reserved service matched the service
  itself. The binary was never invoked; no output, start marker or
  reserved draw existed (`preconditions/preflight-aborted-1.log`,
  `service-journal.log`). The check was replaced by one on the run's
  start marker (`preconditions/preflight.sh` is the corrected version),
  every precondition was re-checked, and the one execution started at
  15:21:30Z.
* **Cross-architecture digits** in reported statistics (§4).
* No interruption, restart, filter, or change to the running
  evaluation. No other reserved, reference or holdout execution.

## 6. Preconditions record

`preconditions/`: `preflight.log` (every precondition of `LOCK.md` §3
and the owner's run instructions, all met, at 15:21:30Z),
`preflight-aborted-1.log`, `preflight.sh`, `run.sh`, `start`, `end`,
`exit` and the service journal.

## 7. Disposition

Under the locked protocol a failed confirmation is a documented **NO
GO**: v0.2-eval-5 is not frozen or activated on this result, the
reserved family is consumed, and no threshold, tolerance, window,
criterion, scenario or candidate is changed against these streams. The
v5 freeze decision is the owner's.
