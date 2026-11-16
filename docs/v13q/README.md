# V0.13q — Human presentation review and protocol lock candidate

**Status: PREPARATION ONLY.** Human presentation review: **PENDING** (no
reviewer named). Lock candidate: **PREPARED, NOT LOCKED**. Protocol lock:
**NO GO**. Live inference: **NO GO**. V0.13.0 release: **NO GO**.

Starting commit: `70184c5dadbd0616df80134923959d801cfe6cb1` (clean main,
equal to `origin/main`; V0.13p accepted, CI green). No implementation,
recorder, verifier, presentation, protocol or evidence file changes in V0.13q.

## Phase 1 — Baseline (verified at `70184c5`)

| Check | Result |
|---|---|
| Frozen human-review package, `git diff ee987a1 -- docs/v13i/human-review` | empty (unchanged) |
| `MANIFEST.json` / `MANIFEST.sha256` | `31bec875…` / `9f2efba9…`; `sha256sum -c`: all 100 entries OK |
| `v13i_human_review.py --check` | 93 transcripts, 7 static assets; HUMAN REVIEW PENDING |
| Transcripts 001–093 concatenated | `14c414d1…` (identical to the V0.13n baseline) |
| 7 static assets concatenated | `a2dd64b8…` (identical to the V0.13n baseline) |
| Engine and simulator source (`git ls-tree -r HEAD internal cmd`) | `1964bd9f…` (identical to the V0.13n baseline); engine `0.13.0-dev.2` |
| `v13e_package.py --check` | 65 files; PASS |
| `v13g_identities.py`, `v13k_identities.py`, `v13l_identities.py`, `v13m_identities.py --check` | PASS |
| `v13m_preserved.py` | 3,606 protected files and all experiment/presentation groups: PASS |
| `docs/v13o/IDENTITIES.json` (10), `docs/v13p/IDENTITIES.json` (16) | all match |
| Protocol `v13o-alt-1` | `202548ad…`, status `DRAFT — NOT LOCKED — NOT AUTHORIZED` |
| Schedule | `2be938b6…`, 262 entries, `protocolSha256` = protocol |
| Call ceiling / request-start guard | 802 / `stop.wallClockSeconds` 14400 with `request.timeoutSeconds` 300, no tolerance |
| Clock tolerance | integer 60 s, pinned in the protocol; applied by the verifier (R-PIN-001) |
| Primary outcome (`analysis`) | identical to V0.13m-lite's protocol |

Golden references, the Go test suite and the full per-version gates are
covered by CI, which passed on `70184c5` (Ubuntu, macOS, ARM,
cross-platform, offline).

## Phase 2 — Human review: PENDING

No reviewer has been named. None is invented. No identity, expertise,
independence declaration, questionnaire response, finding or recommendation
is recorded ([human-review/STATUS.json](human-review/STATUS.json)).

What a reviewer receives, unchanged: the V0.13n packet
([../v13n/human-review/REVIEWER_PACKET.md](../v13n/human-review/REVIEWER_PACKET.md);
archive `dreamteam-human-review-packet-27b04bd.tar.gz`, sha256 `9a93ae1a…`).
It contains only the frozen model-visible material (`ee987a1`), the 12-item
questionnaire and instructions, and no technical audits. The package it
contains is byte-identical to the baseline above.

Owner guidance on the reviewer: a radiologist or radiology operations
leader who has not taken part in DreamTeam's development, judging whether
the alternatives are understandable, balanced and operationally faithful,
not whether the code is correct.

When a reviewer completes it, the record is committed verbatim with:

- reviewer identity and relevant expertise;
- the independence declaration;
- the 12 questionnaire responses and any other specific findings;
- the overall recommendation (ACCEPT / ACCEPT WITH CHANGES / DO NOT ACCEPT);
- the exact package identity reviewed (commit, `MANIFEST.json` SHA-256,
  `v13i_human_review.py --check` output).

## Phase 3 — Findings: none yet

No human findings exist. When they do, each is recorded without altering
its meaning and classified as one of:

| Class | Meaning | Consequence |
|---|---|---|
| DOC | Documentation clarification | No model-visible byte changes. |
| PRESENTATION | Presentation correction | Changes model-visible material: owner approval, new hashes, a new freeze, human re-review, and a new lock candidate. |
| PROTOCOL | Protocol reconsideration | Owner decision, a new protocol identity and a new lock candidate. |

Model-visible material is never edited silently.

## Phase 4 — Lock candidate

[LOCK-CANDIDATE.json](LOCK-CANDIDATE.json), schema
`dreamteam.lock-candidate/v1`, status `CANDIDATE — NOT LOCKED — NOT
AUTHORIZED`. It pins the 12 items of
[../v13p/LOCK-REQUIREMENTS.md](../v13p/LOCK-REQUIREMENTS.md):

1. Protocol `202548ad…` (`v13o-alt-1`).
2. Acceptance standard `c6af89ed…`, equal to the protocol's pin.
3. Recorder: entry `v13p.py` and every module a fresh interpreter loads for
   it (`v13p`, `v13o`, `v13m`, `v13h`, `lifecycle_v4_spec`), each by SHA-256;
   the contract identities file; driver; lifecycle; engine version and
   source tree.
4. Verifier: entry `v13p_verify.py` and every module it loads (`v13p_verify`,
   `v13o_verify`, `v13m_verify`, `v13h`), each by SHA-256, plus its data files.
5. Schedule `2be938b6…`, 262 entries.
6. Presentation: the protocol's text identities, system prompt and workflow
   contracts, and the human-review freeze (`ee987a1`, manifest, transcript
   and asset hashes). Human review: PENDING.
7. Primary outcome: the protocol's `analysis` block.
8. Model settings: models, order, settings files (hash-checked), retries,
   request timeout, per-draw call bound.
9. Allocation: 262 draws, per-unit draws, contexts file.
10. Call ceiling 802.
11. Request-start guard 14400 s / 300 s, without tolerance.
12. Clock tolerance 60 s with its statements, and the value the verifier
    applies.

`analysis/alternatives/v13q_candidate.py --check` re-derives every value
from the repository (strict parsing, recorder binding check and independent
verifier pin check) and requires the committed candidate to equal it byte
for byte. It also refuses if the design values differ from 262 / 802 /
14400 / 60 or the package from 93 / 7. CI runs it. It records nothing and
contacts nothing.

Lock candidate SHA-256: see the final report and `git log`; the candidate
is re-derived on every CI run.

### Protocol-hash enforcement plan (DT13O-02)

Current state: **not yet enforced.** Analysis (`v13p_verify.py`) requires
the archive's protocol SHA-256 to equal the SHA-256 of the protocol file it
is given (L15, `v13m_verify.py:517`), and the seal binds it. Nothing yet
requires that file to be the locked protocol. A re-pinned protocol with the
same id is analysed under its own terms (recheck `scratch/s3arch/repinned-tol120`).

Plan, to be implemented and checked **before the lock is accepted**:

1. **Lock manifest.** The owner-approved lock manifest
   (`dreamteam.protocol-lock/v1`) is the lock candidate with blockers cleared,
   status `LOCKED`, the locked commit, the engine binary SHA-256 and the
   owner approval record. It is strict JSON and committed.
2. **Locked-run analysis entry.** A successor analysis entry (wrapping
   `v13p_verify.py` unchanged) takes `--lock MANIFEST` as a required argument
   and refuses, with the archive INVALID, unless:
   - the manifest parses strictly and has status `LOCKED`;
   - SHA-256(protocol bytes) equals the manifest's item 1;
   - the archive's `protocolSha256` and seal protocol equal it;
   - the acceptance file and the verifier modules equal items 2 and 4.

   With no manifest, it refuses: there is no fallback to an unlocked
   protocol.
3. **Witnesses.** Through the ordinary command line, these cases are
   refused, and the genuine locked archive is accepted:
   - a re-pinned protocol with the same id;
   - an archive recorded under the draft (unlocked) protocol;
   - a tampered manifest;
   - a missing manifest.
4. **Runner side (DT13O-05).** The future runner refuses unless the protocol
   bytes equal the lock manifest's item 1 before calling `v13p.run_schedule`.
5. **Review.** Covered by the owner's lock gate; it is small and bounded.

This plan requires owner authorization: it is implementation, and V0.13q is
preparation only.

## Remaining blockers (also in the candidate)

| Id | Blocker |
|---|---|
| LC-01 | Human presentation review PENDING: no reviewer named. |
| LC-02 | DT13O-02 enforcement not implemented (plan above). |
| LC-03 | Engine binary SHA-256 to be recorded from the locked commit at lock time. |
| LC-04 | Owner lock-gate approval (never automatic) and separate run authorization. |
