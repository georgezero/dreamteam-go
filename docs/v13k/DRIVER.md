# V0.13k driver

`dreamteam-v13k-driver/7` wraps the unchanged V0.13h draw, engine operations, decision workflow, parser, presentation, view, selection, instruction, canonicalization and primary submitted-action estimand. Only recording, persistence and verification change. The engine version remains `0.13.0-dev.2` because mechanics, session, API, accounting and scoring are unchanged.

The successor protocol identity is `dreamteam.alternatives-experiment/v7`, experiment `v13k-alt-1`; the schedule format remains /4 with a successor protocol binding. Nine historical contexts, six sensitivity pairs, 21 contexts, two arms, 262 draws and the 802-call ceiling remain fixed. Existing recordings retain their original identities and bytes.

The command-line `run` remains refused because the protocol is not locked. Engineering fixtures use scripted replies and mock or localhost-only services. No inference or external model service is part of these gates.

See [recording](RECORDING.md), [inventory](EVIDENCE_BINDINGS.md), [lifecycle](LIFECYCLE_CONTRACT.md) and [known limitations](KNOWN_LIMITATIONS.md).
