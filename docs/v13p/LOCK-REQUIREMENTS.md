# V0.13p — Requirements for the eventual protocol lock and live runner

**Status: REQUIREMENTS ONLY.** This file does not create a lock, a lock
candidate or a runner. Protocol lock: **NO GO**. Live inference: **NO GO**.
V0.13.0 release: **NO GO**. Human presentation review: **PENDING**.

Owner decisions on the V0.13o recheck
([docs/audits/v0.13o-dt13m-01-recheck.md](../audits/v0.13o-dt13m-01-recheck.md)):
DT13O-02 is to be required in the lock manifest, DT13O-05 is mandatory, and
DT13O-03 is accepted as a documented limitation. This file records those
decisions as requirements the lock gate must check.

## 1. The lock manifest must pin (DT13O-02)

Each item must be pinned by exact SHA-256 of the file bytes (or, for values,
by the exact value in the pinned protocol). The values below are the current
drafts. The lock gate re-derives every one; it does not copy them from here.

| # | Item | Current draft identity | Source |
|---|---|---|---|
| 1 | Exact protocol file | `docs/benchmark/v13o-dev/protocol-draft.json`, id `v13o-alt-1`, schema `dreamteam.alternatives-experiment/v10`, SHA-256 `202548ad613d2c95b639b345e4fdff262386879d0a17a723fabf50d11e1b7b34` | file bytes |
| 2 | Acceptance file | `analysis/alternatives/acceptance_v1.json`, `dreamteam.acceptance-standard/v1`, SHA-256 `c6af89ede52284a023f1413ea7b3d46b38a7da516ccd87da92841efe1195a721` | file bytes; must equal the protocol's `acceptance.sha256` |
| 3 | Recorder identity | entry point `analysis/alternatives/v13p.py` and the modules it runs unchanged: `v13o.py`, `v13m.py`, and the V0.13m-lite recorder contract (`docs/v13m/CONTRACT-IDENTITIES.json`, checked by `v13m_identities.py --check`); engine binary built from the locked commit (engine `0.13.0-dev.2`) | `docs/v13p/IDENTITIES.json` |
| 4 | Verifier identity | entry point `analysis/alternatives/v13p_verify.py` and the modules it runs unchanged: `v13o_verify.py`, `v13m_verify.py`, `bindings_v2.json` (`dreamteam.run-lifecycle/v4`, `e93b3a5b…`) | `docs/v13p/IDENTITIES.json` |
| 5 | Schedule identity | `docs/benchmark/v13o-dev/schedule-draft.json`, SHA-256 `2be938b639baa1531e3a577c96317b7103530ae13c736cc795e6cf8ed7c7701a`; order rule `dreamteam-v13d-order/1`; its `protocolSha256` equals item 1 | file bytes |
| 6 | Presentation identity | protocol `texts`: presentation `dreamteam.alternatives-presentation/v3` (`19caebcb…`), view table `dreamteam.alternatives-view/v2` (`f96e2c4f…`), selection rule `dreamteam.alternative-selection/v2` (`5f32b2c8…`), arms `dreamteam.alternatives-arms/v3`; system prompt `d7dc7e0f…`; the human-review freeze `ee987a1` (`docs/v13i/human-review/MANIFEST.json`, `31bec875…`) and the outcome of that review | protocol fields; manifest |
| 7 | Primary outcome definition | protocol `analysis.estimand` (per context and model: the difference between the arms' distributions of the canonical submitted action batches within comparison-exposed draws), with `analysis.notClaimed`, `observable` and `pairs` (secondary) | protocol field |
| 8 | Model settings | protocol `models`, `modelOrder`, `settings.files` (`llm-gemma4-26b-a4b.json` `1a1edae5…`, `llm-qwen36-35b-a3b.json` `e19a47dc…`), `settings.maxRetries` 1, `settings.allowDetails`, `request.timeoutSeconds` 300, `workflow.contracts`, driver `dreamteam-v13m-driver/9` | protocol fields; settings file bytes |
| 9 | 262-draw allocation | protocol `draws` 262, `drawsPerUnit`, `contexts`, `contextsFileSha256` `b26c81ab…`, `duplicateInputs` | protocol fields |
| 10 | 802-call ceiling | protocol `maxModelCalls` 802, `maxModelCallsPerDraw` 6 | protocol fields |
| 11 | Four-hour request-start guard | protocol `stop.wallClockSeconds` 14400 with `request.timeoutSeconds` 300: a request starts only if its whole timeout fits before the stop; no tolerance applies to it | protocol fields |
| 12 | 60-second tolerance | protocol `acceptance.clockReconciliation` (R-HALT-003, integer `toleranceSeconds` 60, scope, onExcess, basis), equal to the pinned acceptance file's R-HALT-003 | protocol field; item 2 |

Further requirements:

- **The analysis must require the locked protocol SHA-256.** The verifier
  takes the expected acceptance hash from the protocol it is given
  (DT13O-02), so the lock record is what fixes the protocol. Analysis of a
  locked run is valid only for an archive whose bound protocol SHA-256 equals
  item 1 as recorded in the lock manifest. Archives under any other protocol
  are out of scope, whatever their id.
- **Strict parsing.** The lock gate must read the protocol, the acceptance
  file, the schedule and the manifest with `dreamteam.strict-json/v1` (no
  duplicate key at any depth, no non-finite number). It must refuse the
  lock if any of them fails.
- **Version changes.** Changing any pinned item requires a new protocol id
  or a new pinned identity, and a new lock. The lock never relabels an
  existing identity.

## 2. Future live runner (DT13O-05)

- Any future live runner **must** record through `v13p.run_schedule`, or
  through a named successor that wraps it with the same checks. It passes the
  protocol file's exact bytes, which the entry point:
  - parses strictly (R-PROTO-001);
  - hashes;
  - checks against the strictly parsed acceptance standard (R-POLICY-001, then
    the V0.13o binding).
- It must not call `v13m.run_schedule`, `v13m.Recorder` or
  `v13o.run_schedule` directly. Those remain as preserved history. They do
  not parse strictly, and V0.13m-lite records under any protocol.
- It must also refuse to start unless the protocol is the locked one (item 1)
  and the owner's run authorization is present. The current `v13p.py run`
  refuses in every case, because the live run is not implemented.
- No runner is implemented in V0.13p, and no model endpoint is contacted.

## 3. Documented limitation (DT13O-03, accepted)

R-HALT-003 can only detect a clock discrepancy that the recorded timestamps
around the halt actually show. If the UTC clock jumps ahead by more than
60 s and no timestamp is recorded between the jump and the halt, the archive
is accepted. The same gap with a tight bracket of timestamps is rejected.

The 60 s tolerance is an operational assumption, not a measured clock-accuracy
guarantee (the protocol's `basis`). This limitation predates V0.13o, and the
owner accepts it as documented.
