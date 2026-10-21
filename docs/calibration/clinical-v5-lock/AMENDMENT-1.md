# Lock amendment 1: the executing host, and the owner's authorization

**2026-10-21.** This amends the lock record `LOCK.md` (lock commit
ff5b566) before the reserved run. It changes no rule, input, criterion,
tolerance, stream, scenario, seed count or command logic, and no file
listed in `lock.sha256`.

## 1. The owner's authorization

The owner authorized, on 2026-10-21, **one** reserved execution of
PCG(20261201, ·) under the locked protocol, on the host `horo`, with the
linux/amd64 build of the lock commit's source (§2). The authorization
does not permit additional or replacement reserved executions, strategy
optimization, policy, threshold or manifest changes, post-run tuning,
reference or holdout runs, or v5 activation or freeze.

## 2. The executable

The lock pinned a linux/arm64 test binary. The run executes on `horo`,
an x86-64 host (Intel Core i7-10700, 16 threads, 31 GB), about nine
times faster than the machine of the lock. Its executable is the same
source, built by the same toolchain with the target set:

```
GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 \
  go test -c -trimpath -o eval.test ./internal/eval     # go1.27.1, from the lock commit's tree
```

| Item | Value |
|---|---|
| Source | the lock commit's tree: `sourceSHA256` `5cd6a438a3c2e741f691815ba96e805f022f39f34a51f131d8a2164f9a07588b`, unchanged |
| Toolchain | go1.27.1, cross-compiling for linux/amd64 (`GOAMD64=v1`) |
| linux/amd64 test binary | SHA-256 `bf9e1c46a90022331efa571ff66f83e2d38ae4e73aa3f4c93f63a9677724e088` |
| linux/arm64 test binary (the lock's, not used for the run) | SHA-256 `5edee74db6bb36314fd5dfd8357ac918714c850bbd0be931f5101788a183130d` |

The amd64 binary is reproducible: the same digest was built from the
working tree and from a `git archive` of the lock commit.

**Equivalence, on the development family only.** The amd64 binary on
`horo` ran the 49-scenario, 40-seed development pass three times. Each
time its 144 lines were byte-identical to the arm64 build's
(`d2fe1c0d6d50729a0eebbdb36f2358e9f0941242f0a3d5cfc395826445e20452`),
in 261–272 s against 2,428 s.

## 3. What else is unchanged

The preconditions, the command and the paths of `LOCK.md` §3 apply as
written, with `$HOME` the home directory on `horo` and the binary of §2.
The suite precondition was met at the lock commit (on the machine of the
lock, 3,478 s, every package passing); the run's binary is that commit's
source, unchanged.
