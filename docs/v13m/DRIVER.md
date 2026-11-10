# V0.13m-lite driver

`dreamteam-v13m-driver/9` wraps the unchanged V0.13h draw, engine operations,
decision workflow, parser, presentation, view, selection, instruction,
canonicalization and primary submitted-action estimand. Only recording
failure handling (fail-closed) and verification change. The engine version
remains `0.13.0-dev.2`.

Successor protocol identity: `dreamteam.alternatives-experiment/v9`,
experiment `v13m-alt-1`, schedule format /4 with a successor protocol
binding (`docs/benchmark/v13m-dev/`). Nine historical contexts, six
sensitivity pairs, 21 contexts, two arms, 262 draws and the 802-call ceiling
are unchanged; the schedule entries are identical to V0.13l's.

The command-line `run` remains refused because the protocol is not locked.
All engineering evidence uses scripted replies and scripted or localhost-only
services. See [recording](RECORDING.md) and the
[acceptance contract](ACCEPTANCE_CONTRACT.md).
