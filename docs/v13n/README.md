# V0.13n — Human review and protocol-lock preparation

**Status: PREPARATION ONLY.** Human presentation review: **PENDING** (no
reviewer assigned). Bounded independent technical check: **PENDING**
(deferred by the owner). Protocol lock: **NO GO**. Live inference: **NO GO**.
V0.13.0 release: **NO GO**.

## Owner decisions carried into V0.13n

| Item | Decision |
|---|---|
| DT13L-01 strict response JSON | Accepted as fixed |
| DT13L-02 wall-clock causality | Accepted as fixed |
| DT13L-03/04 fail-closed policy | Accepted as documented limitations |
| Revised acceptance contract (`docs/v13m/ACCEPTANCE_CONTRACT.md`) | Approved |
| 60-second clock tolerance | Approved provisionally, with conditions below |
| V0.13m-lite engineering milestone | Accepted |
| Further recorder development | Stopped |
| Human presentation review | GO |

Conditions on the 60-second tolerance: it reconciles halt elapsed time with
recorded wall-clock timestamps only and does not weaken the four-hour
request-start threshold; a discrepancy over 60 seconds causes conservative
rejection without retry or replacement; it is pinned in the protocol and is an
operational assumption, not a measured clock-accuracy guarantee. It is
currently pinned in `analysis/alternatives/acceptance_v1.json`
(`toleranceSeconds: 60`), part of identity `dreamteam.acceptance-standard/v1`.
Whether the implementation meets these conditions is one of the items of the
pending bounded technical check.

The V0.13m-lite technical acceptance review (`docs/v13m/ACCEPTANCE_REVIEW.md`)
was performed by the implementer. It is not independent verification.

## Phase 1 — baseline (commit `9d4626bc51cecb3dd4d21f2efa7a246bbbcf78ca`)

| Item | Identity / check |
|---|---|
| Frozen human-review package | `git diff ee987a1 -- docs/v13i/human-review` clean |
| `MANIFEST.json` | `31bec875723d2d59643efb47979538eb8cd77ed07583228e9f78ce400b9b8faf` |
| `MANIFEST.sha256` | `9f2efba95693451f514bb9e6dd06795a990332649493d85a5e2148ece8bee0a5` |
| 93 transcripts / 7 static assets | `v13i_human_review.py --check` and `sha256sum -c MANIFEST.sha256` pass |
| Transcripts 001–093, concatenated in order | `14c414d144efa76f211d6dd14af2e15ea6f5fa8c5f2a25fb476198a78f209655` |
| Static assets, concatenated in name order | `a2dd64b82cb6803f366458eea1a9daec8db87d3440d3d04d5fa33160e2d47032` |
| Simulator and engine source (`git ls-tree -r HEAD internal cmd`) | `1964bd9fbd367b3d6c49c209761db2512bba21edb2ae81b1a6763fe1d8a0dfa1`; engine `0.13.0-dev.2` |
| Protocol draft (`docs/benchmark/v13m-dev/protocol-draft.json`) | `ca120d95dfcefa5c81d7763c2142a0c31fff1aa6a278955f6ab427e9ced1981c`; `maxModelCalls` 802; `wallClockSeconds` 14400 |
| Schedule draft (`docs/benchmark/v13m-dev/schedule-draft.json`) | `1e5bf02010c6fe16ef711056784a3e752507ae9e96cbab14ad5d7aed5e325a13`; 262 entries |
| Acceptance standard (`analysis/alternatives/acceptance_v1.json`) | `c6af89ede52284a023f1413ea7b3d46b38a7da516ccd87da92841efe1195a721` |
| Recorder (`analysis/alternatives/v13m.py`) | `c83bb296367f72ca59b18cdef1c9fa96a81bf34d1893c9dcc5302b399ea509da` |
| Verifier (`analysis/alternatives/v13m_verify.py`) | `327ec2fbf5c3431cd164d48cea2751ca535323bfe8714c5dc2eef8e849648455` |
| All V0.13m contract identities | `docs/v13m/CONTRACT-IDENTITIES.json`, `87e9e77fecfdbf17e322b707ad06fb4d471c7b2c37c2e42c7be9e7c433bc09c2` |

Primary outcome, completion seal and historical evidence are unchanged since
V0.13m-lite (its gate record verifies 3,606 protected files byte-identical);
V0.13n changes no code or evidence.

## Phase 2 — human review

The reviewer packet and requirements are in
[human-review/REVIEWER_PACKET.md](human-review/REVIEWER_PACKET.md). The
packet contains only the frozen model-visible material, the 12-item
questionnaire and the instructions; it excludes every technical audit and its
conclusions. Status: **HUMAN REVIEW PENDING**; no reviewer assigned.

## Phase 3 — disposition

No human findings yet.

## Phase 4 — bounded technical check

**PENDING.** The owner deferred it. Scope when run: strict response JSON;
wall-clock halt causality; fail-closed persistence; no false COMPLETE; no
contradictory PARTIAL; no invalid evidence in primary analysis; no automatic
retry or resume; historical preservation; reproduction of the original
DT13L-01 and DT13L-02 witnesses; the 60-second tolerance conditions.
Read-only, no code changes.

## Phases 5–6 — lock candidate and lock gate

Not started. They require the human review to pass and the bounded technical
check to be accepted. No lock is made automatically.

## Remaining blockers for protocol lock

1. Independent human presentation review (PENDING).
2. Bounded independent technical check (PENDING, deferred).
3. Lock candidate preparation and owner approval (not started).
