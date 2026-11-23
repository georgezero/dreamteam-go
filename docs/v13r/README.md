# V0.13r — Locked-protocol enforcement (LC-02)

**Status: IMPLEMENTED, TEST ONLY fixtures.** No protocol is locked: there is
no LOCKED manifest in this repository and no lock commit. Human
presentation review: **PENDING**. Protocol lock: **NO GO**. Live inference:
**NO GO**. V0.13.0 release: **NO GO**.

Starting commit: `aabda48c2e95563e5fef294d4b13d0d255511061` (clean main,
equal to `origin/main`; V0.13q accepted; owner GO for LC-02).

## Phase 1 — Baseline (verified at `aabda48`)

| Check | Result |
|---|---|
| Frozen human-review package (`ee987a1`) | unchanged; `MANIFEST.json` `31bec875…`, `MANIFEST.sha256` 100 entries OK; `v13i_human_review.py --check`: 93 transcripts, 7 static assets |
| Transcripts / static assets, concatenated | `14c414d1…` / `a2dd64b8…` |
| Engine and simulator source (`git ls-tree -r HEAD internal cmd`) | `1964bd9f…`, engine `0.13.0-dev.2` |
| Protocol `v13o-alt-1` / schedule / acceptance standard | `202548ad…` / `2be938b6…` (262 entries) / `c6af89ed…` |
| Design | 802-call ceiling; 14,400 s request-start guard (300 s timeout, no tolerance); 60 s clock tolerance; primary action estimand (`analysis`) unchanged |
| Lock candidate `docs/v13q/LOCK-CANDIDATE.json` | `e55b0812…`; `v13q_candidate.py --check`: 12 items re-derived and equal |
| Identities and history | `v13e_package`, `v13g/k/l/m_identities --check`, `v13m_preserved.py` (3,606 files), `docs/v13o` and `docs/v13p` identity files: PASS |

The local engine binary was stale: an Nov 5 build of `e40e6c0` that lacked
the `alternatives.v3.canonical` operation. Earlier witnesses halt before the
first draw, so they never exercised it. It was rebuilt from HEAD exactly as
CI builds it (`go build -o dt ./cmd/dreamteam`). CI always builds its own
binary.

## Authority: content identity is not authorization

A manifest's hashes establish *what* is locked. They do not establish that
the owner locked it: a `status: LOCKED` field or an approval string in an
editable file confers nothing. Authority comes from the owner's **lock
commit**, an immutable Git identity. The checks:

1. The operator passes the lock commit in full (40 hex) with
   `--lock-commit`. It must be in the history of the repository's HEAD.
2. The commit has exactly one parent, the manifest's `creation.baseCommit`.
3. It **adds** the manifest at `creation.lockFile`. It changes nothing except
   the lock files: the manifest, the approval record, the human-review
   record, and the locked protocol and schedule.
4. The manifest presented to the tool is byte-identical to the one in that
   commit. The protocol, schedule, approval record and human-review record
   in that commit's tree have the pinned SHA-256 values.
5. The commit's author is the manifest's `approval.owner`. The approval
   record states the lock id, the protocol SHA-256, the decision and the
   owner, each on its own line.
6. The engine source tree at that commit has the pinned hash.

The owner records the lock commit id in a separate decision, the same way
releases are recorded. Analyses and runs use that id. The trust root is the
owner-controlled `main` history; no signing infrastructure is introduced.

## Lock manifest schema: `dreamteam.protocol-lock/v1`

Strict JSON (`dreamteam.strict-json/v1`, with its own parser in
`v13r_lock.py`; a three-way differential test shows it agrees with the
recorder and verifier parsers). Every object has an exact key set: a missing
or unexpected field is refused.

| Field | Content |
|---|---|
| `schema`, `status`, `purpose`, `lockId`, `notice` | `dreamteam.protocol-lock/v1`; `LOCKED`; `RUN` or `TEST ONLY`; lock id; the fixed notice for the purpose |
| `protocol` | `file`, `sha256` of the exact bytes, `id`. The locked protocol is the draft with status `LOCKED — NOT RUN` (the only status `v13p.py run` accepts). |
| `acceptance` | `file`, `sha256` (must equal the protocol's pin) |
| `schedule` | `file`, `sha256`, `entries`. Its `protocolSha256` must be the locked protocol's, and it must equal the schedule computed from the protocol and contexts. |
| `recorder` | entry `v13r_preflight.py`; `modules`: every repository module a fresh recorder process loads, by path and SHA-256 (24 today, including `v13p`, `v13o`, `v13m`, `v13h`, `stage_b`, the `agents/llm` workflow) |
| `verifier` | entry `v13p_verify.py`; `modules` (21 today); `data` (`acceptance_v1.json`, `bindings_v2.json`) |
| `analysis` | entry `v13r_analyze.py`; `modules` loaded beyond the verifier's (`v13r_analyze`, `v13r_lock`) |
| `engine` | `version`, `sourceTreeSha256` (at the lock commit), `binarySha256` (the run host's binary, built from the base commit; LC-03), `platform` |
| `models` | `models`, `modelOrder`, `settings` (files hash-checked on disk), `request`, `maxModelCallsPerDraw`, all equal to the protocol's |
| `presentation` | `texts` (files hash-checked), `systemPromptSha256`, `workflow`, `humanReviewFreeze` (`ee987a1`, manifest and content hashes), `humanReview` (`verdict`, `record` file and SHA-256) |
| `outcome` | `analysisSchema`, `definition` (the protocol's `analysis` block, the primary action estimand), and `sha256` of its canonical JSON |
| `design` | `draws`, `maxModelCalls`, `wallClockSeconds`, `timeoutSeconds`, `toleranceSeconds`. A RUN lock must pin 262 / 802 / 14400 / 300 / 60. |
| `approval` | `owner` (`name`, `email`), `decision`, `date`, `record` (file and SHA-256) |
| `creation` | `tool`, `candidateSha256` (the V0.13q candidate it came from), `baseCommit`, `lockFile`, `created` |

Purpose rules:

- **RUN** requires `approval.decision` = `APPROVE PROTOCOL LOCK`,
  `humanReview.verdict` = `ACCEPT`, and the approved design values.
- **TEST ONLY** requires the fixed markers `TEST ONLY — NOT AN OWNER APPROVAL`
  and `TEST ONLY — NO HUMAN REVIEW`. A TEST ONLY manifest that claims owner
  approval or a review verdict is refused.
- A `dreamteam.lock-candidate/v1` manifest is never a lock.

## Locked analysis: `v13r_analyze.py`

```sh
python3 -I -B analysis/alternatives/v13r_analyze.py analyze \
  --lock LOCK.json --lock-commit <40-hex> --protocol <locked protocol> --schedule <locked schedule> \
  --contexts DIR --binary ./dt --dir RUN --replays DIR --out DIR      # classify: same, without --replays/--out
```

`--lock` and `--lock-commit` are required. There is no fallback to unlocked
analysis and no option to skip a check. In order, before any verifier code
runs:

1. **Strict parse and structure.** `LK-PARSE`, `LK-SCHEMA`, `LK-STATUS`,
   `LK-PURPOSE`. A TEST ONLY lock needs `--test-only`, and `--test-only`
   accepts nothing else.
2. **Lock commit identity.** `LK-COMMIT`, `LK-APPROVAL`, `LK-REVIEW`,
   `LK-ENGINE` (source tree).
3. **Modules** (`LK-MODULE`). Every repository module loaded in the process
   must be exactly the lock's verifier and analysis modules, byte for byte.
   This is checked again after classification or analysis, before anything is
   written.
4. **Protocol** (`LK-PROTOCOL`). The exact bytes, locked status, id and
   contexts; R-PROTO-001 applies.
5. **Acceptance standard** (`LK-ACCEPTANCE`). The exact bytes, then R-POLICY-001
   and R-PIN-001.
6. **Schedule** (`LK-SCHEDULE`).
7. **Engine binary** (`LK-ENGINE`).
8. **The rest of the locked configuration.** Model settings (`LK-MODEL`),
   presentation (`LK-PRESENTATION`), outcome definition (`LK-OUTCOME`) and
   design values (`LK-DESIGN`).
9. **Archive** (`LK-ARCHIVE`). `run.json` and the completion seal name the
   locked protocol id and SHA-256, and the run's `authorization` begins with
   this lock's binding:

   ```
   dreamteam.lock-authorization/v1 lock=<manifest sha256> commit=<lock commit> protocol=<sha256> purpose=<purpose>;
   ```

   This binding is recorded only by `record_after_preflight`.

An unreadable input is refused with `LK-IO`.

**On refusal:**

- `classify` prints `{"state": "INVALID", "lockRefused": true, ...}` and
  exits 1.
- `analyze` exits non-zero with `LOCK REFUSED <code>` and writes nothing.

**On success:** `v13p_verify.classify` / `analyze` run unchanged, with the same
arguments as the unlocked path. The output gains a `lock` block (lock id,
manifest SHA-256, lock commit, protocol SHA-256, marker). A TEST ONLY lock is
marked `TEST ONLY — NOT A PRIMARY RESULT`. The primary estimand is not
modified.

## Runner preflight: `v13r_preflight.py`

`preflight(...)` runs the same lock checks with the recorder's modules.
- **Protocol:** the exact protocol bytes, R-PROTO-001 and the V0.13o
  acceptance binding via `v13p`.
- **Locked inputs:** the acceptance standard, schedule, engine binary and
  model configuration must all be the locked ones.
- **Modules:** a RUN lock requires the loaded repository modules to be
  exactly the recorder set.
- **Result:** a `Preflight` object.

`record_after_preflight(pf, ...)` is the only path to `v13p.run_schedule`
under a lock. It writes the lock binding at the start of the run's
authorization. In V0.13r it refuses a RUN lock (`LK-RUNNER`): no live runner
is built or authorized, and nothing contacts a model.

```sh
python3 -I -B analysis/alternatives/v13r_preflight.py check --lock LOCK.json --lock-commit <40-hex> \
  --protocol <locked protocol> --schedule <locked schedule> --contexts DIR --binary ./dt
```

## Adversarial witnesses (`docs/v13r/witness/results.json`, deterministic)

`v13r_witness.py` builds a scratch Git repository containing:
- the engine source tree, taken from `git archive` of HEAD;
- a **TEST ONLY** lock commit authored by a TEST ONLY identity, pinning a
  synthetic single-context protocol.

The valid archive is recorded with scripted model replies through the
preflight, so no endpoint is involved. Every adversarial variant is its own
commit. Each case runs through the ordinary command line.

| Case | Result |
|---|---|
| Valid TEST ONLY lock | reaches the verifier; COMPLETE; locked analysis **equals the unlocked V0.13p analysis** of the same archive apart from the `lock` block; marked TEST ONLY |
| Byte-identical module copies | reaches the verifier; COMPLETE |
| Missing `--lock` / missing `--lock-commit` | refused (usage) |
| TEST ONLY lock without `--test-only` | LK-PURPOSE |
| V0.13q candidate instead of a lock | LK-SCHEMA |
| Status not LOCKED | LK-STATUS |
| Missing approval; missing engine field | LK-SCHEMA |
| Duplicate manifest key; nested duplicate key | LK-PARSE |
| TEST ONLY lock claiming owner approval | LK-APPROVAL |
| Lock commit not in HEAD's history; unknown commit; abbreviated id; manifest differing from the commit; commit changing other files | LK-COMMIT |
| Commit not authored by `approval.owner`; approval record altered after the lock; record without the decision line | LK-APPROVAL |
| Re-pinned protocol with the same id | LK-PROTOCOL |
| Archive under the re-pinned protocol; draft-protocol archive; draft archive relabelled to the locked protocol with all hashes and the seal recomputed | LK-ARCHIVE |
| Changed acceptance file | LK-ACCEPTANCE |
| Changed schedule | LK-SCHEDULE |
| Changed verifier module; changed analysis module | LK-MODULE |
| Wrong engine binary | LK-ENGINE |
| Changed model settings file pin; changed request settings | LK-MODEL |
| Changed outcome definition | LK-OUTCOME |
| **Fully forged archive**: draft archive relabelled, lock binding written into its authorization, all hashes and the seal recomputed | **reaches the verifier: limitation, below** |

All 30 refusal cases are refused before the verifier, and none writes an
analysis.

Runner preflight results:
- **Passes:** the valid lock.
- **Refused:**
  - a candidate instead of a lock (LK-SCHEMA);
  - a re-pinned protocol or the draft protocol (LK-PROTOCOL);
  - a changed schedule (LK-SCHEDULE);
  - the wrong binary (LK-ENGINE);
  - a changed recorder module (LK-MODULE);
  - a changed acceptance file (LK-ACCEPTANCE).
- **No RUN recording:** `record_after_preflight` refuses a RUN lock
  (LK-RUNNER).

Unit tests (`test_v13r.py`, 13 tests):
- every manifest field is required, and unexpected fields are refused;
- 21 malformed or rule-breaking manifests are refused with their codes;
- strict parsing rejects duplicate keys, non-finite numbers, invalid UTF-8
  and trailing content;
- the lock parser, the recorder parser and the verifier parser agree on the
  66-input corpus;
- refusals never call the verifier (it is replaced by a trap);
- there is no fallback option;
- `v13r_analyze` imports no recorder, and `v13r_preflight` imports no
  verifier.

## Limitation: archive authenticity (owner decision needed)

Archives are sealed with hashes, not signatures. Anyone who can rewrite an
archive can:
- relabel it to the locked protocol;
- write the lock binding into its `authorization`;
- recompute every hash and the seal.

Content checks cannot tell this from a genuine locked run. The witness
demonstrates it. Every weaker forgery is refused, including the relabelled
and resealed archive that lacks the binding.

The proposed mitigation is a procedure, with no new cryptography. Immediately
after a locked run, the owner commits a custody record holding the archive's
`SEAL.json` SHA-256 and the run id. The locked analysis would then also
require that custody commit (`--custody-commit`), checked the same way as the
lock commit. This is not implemented in V0.13r; it is listed as a remaining
lock requirement.

## Lock creation procedure (for the owner; not performed)

1. **Preconditions:** the human presentation review is complete, with verdict
   ACCEPT and the record committed verbatim (LC-01); and the owner decides to
   lock.
2. **Locked protocol and schedule.** Write the locked protocol (the draft with
   status `LOCKED — NOT RUN`) and its schedule with `v13p.py schedule`. Commit
   neither yet.
3. **Engine binary.** Build the engine on the run host from the base commit
   (`go build -o dt ./cmd/dreamteam`) and record its SHA-256 (LC-03).
4. **Approval record.** The owner writes the approval record. It states
   `Lock:`, `Protocol SHA-256:`, `Decision: APPROVE PROTOCOL LOCK` and
   `Owner: <name> <email>`. The tool never writes it.
5. **Manifest.** Generate it with `v13r_lock.create(purpose="RUN", ...)` from
   the V0.13q candidate.
6. **Lock commit.** The owner makes a dedicated lock commit containing only
   the lock files, authored as `approval.owner`, and pushes it to `main`.
7. **Verify.** `v13r_lock.py check --lock … --lock-commit <id>`, then
   `v13r_preflight.py check …`.
8. **Record the id.** The owner records the lock commit id in a separate
   decision. A run additionally requires the owner's run authorization.

## Preservation

New files only: `v13r_lock.py`, `v13r_analyze.py`, `v13r_preflight.py`,
`v13r_witness.py` and `test_v13r.py`, plus `docs/v13r/`, a CI step and a
PROGRESS entry. Nothing else changed: not the simulator, accounting,
scoring, session, decision workflow, alternative selection, presentation,
primary outcome, historical evidence, frozen review package or schedule. No
Go file changed.

## Remaining before an owner lock gate

| Id | Requirement |
|---|---|
| LC-01 | Independent human presentation review (PENDING; the owner nominates and contacts the reviewer; the packet is ready). |
| LC-02 | **Implemented here.** Archive custody (above) is an open owner decision. |
| LC-03 | Engine binary SHA-256 recorded from the run host at lock time. |
| LC-04 | Owner lock-gate approval in a lock commit, and a separate run authorization. |
| — | The live runner itself (not built; separate authorization). |
