# v0.2-eval-5 confirmatory cycle: closure

**Status: CLOSED, NOT APPROVED (2026-10-21).** The v0.2-eval-5 clinical
candidate completed its one prospectively locked confirmatory evaluation
and was certified **NO GO**. The owner accepted the verdict as final.
v0.2-eval-5 is not frozen as an approved policy and is not activated;
v0.2-eval-4 remains the policy in force. The reserved family is
consumed and is never rerun. No criterion, threshold, tolerance,
window, scenario or candidate is changed against its streams, and the
failed result is not reinterpreted.

## Record

| Item | Value |
|---|---|
| Locked protocol | commit `ff5b566` (parent `7bf1ed6`, 0.2.0-dev.26); `../clinical-v5-lock/LOCK.md`, `lock.sha256` |
| Authorized execution | amendment 1, commit `38881d4`: the owner's authorization of one run, on `horo`, with the linux/amd64 build |
| Reserved evidence | commit `af45430`: `../clinical-v5-reserved/` (outputs, run record, preconditions record), unchanged |
| Verdict | invocation `confirmatory`; **`certified NO GO: confirmation failed`** |
| Failing criterion | **T3** zero-tolerance admissibility: one seed of 10,000 stable-practice seeds (S-A85, seed 259) breaches the urgent past-due absolute limit |
| Passing mandatory criteria (10) | T1, T2, T2-conformance, T2G, T4, T5a, T5b, T6, T7, T8 |

**Locked inputs** (each verified before the run and recorded by it):

| Input | SHA-256 |
|---|---|
| Candidate policy v0.2-eval-5 | `83d481b7040b69d78a58de68afc938a588b5b37ba396438509f38c56ca163948` |
| Scenario manifest | `250bc7fc835d99140212505e0e79f74f2da1484dbe856fea6a5b93eec9153ef4` |
| Protocol (locked) | `12d1eda4f8543e64cdbf1d5118339b65bc6a620e8ee3f6cb01dfd974d0c91a0d` |
| Policy in force v0.2-eval-4 | `4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3` |
| Simulator profile | `ac8621889813a2d582b4b19674f04871c5711d27d0ac1620839ba01df4596100` |

**Execution and outputs:**

| Item | Value |
|---|---|
| Source (`sourceSHA256`, recorded) | `5cd6a438a3c2e741f691815ba96e805f022f39f34a51f131d8a2164f9a07588b` |
| Executable (linux/amd64, amendment 1) | `bf9e1c46a90022331efa571ff66f83e2d38ae4e73aa3f4c93f63a9677724e088` |
| Lock's linux/arm64 build (not executed for the run) | `5edee74db6bb36314fd5dfd8357ac918714c850bbd0be931f5101788a183130d` |
| Host, time | `horo`; 2026-10-21T15:21:30Z to 17:31:45Z; exit 0 |
| `reserved-run.jsonl` (3,904 lines) | `86826be5fc29b6288ee492dbdf8229ff07ef72b912227dbfd4409e0965808619` |
| `reserved-criteria.json` | `674af45cb3eb5c8dd5365b97861dfd9884781823c815103c72ca334380d2cc7e` |
| `reserved-run.log` | `24fc4370389e1be5591f590acb979025399b7223e1c2d80aad6be133b8fb5e7c` |

## Recorded events

* **A launch aborted before execution.** The first start of the run
  service stopped in its own precondition check (the check for an
  existing reserved service matched itself). The binary was never
  invoked; no output or reserved draw existed. The check was corrected
  and every precondition re-verified before the one execution.
* **Cross-architecture rounding.** Re-reading the lines offline with
  the amd64 binary reproduces every criterion exactly. With the lock's
  arm64 build every status is identical, while some reported statistics
  (interval bounds, T2's orientation-only bound, one cluster standard
  error) differ in their last one or two digits (relative 1e-16 to
  1e-14). No decision depends on them. The certified values are the
  amd64 execution's.
* **No reserved rerun, no post-run tuning, no freeze, no activation.**

## What the result does and does not say

* The evaluator, the protocol and the certification worked as designed:
  a prospectively locked rule was applied once, to trusted in-process
  measurements, and its verdict was accepted.
* One violation in 10,000 seeds does not establish a violation rate
  (exact 95% interval 0.000003 to 0.00056 per seed), nor that the
  practice model or the candidate is generally unsafe. Under the locked
  zero-violation criterion it is unambiguously NO GO.
* The investigation of the failing seed is
  `t3-investigation.md`; any further clinical work is a separate v6
  development stream on new development seeds.
