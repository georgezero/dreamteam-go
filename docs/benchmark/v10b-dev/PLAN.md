# V0.10b live shared-prefix decision experiment: prospective plan

`v10b-decision-1` (`protocol.json`, SHA-256 `5bef15f03956a16b0d3eb80c022fd45a368aa89ec2a27efb55c66e75f777db60`; schema
`dreamteam.shared-prefix/v2`, specified in [SPEC.md](SPEC.md)). This plan
was committed before any request of this experiment was sent. A change
after that commit is an amendment, recorded at the end with its reason.
None is planned.

**The question.** At a recorded branch point, the state, the conversation
and the initial staffing proposal are held fixed. Does the model's final
decision (keep, revise or abandon the proposal) have a different
distribution when it is shown the engine's preview of the proposal than
when it is asked for its final decision without it?

The outcome is the immediate decision. Long-run financial outcomes are
not.

## 1. Models and settings

| Item | Value |
|---|---|
| Models | `gemma-4-26B-A4B-it-oQ4e-mtp`, `Qwen3.6-35B-A3B-oQ4e-mtp`, the V0.5c configurations, unchanged; no substitute if one is unavailable |
| Server | oMLX 0.7.0, OpenAPI SHA-256 `67525ae2…` on 2026-11-01, the same as V0.10a; reached through `DREAMTEAM_LLM_BASE_URL`, never recorded |
| Request | the recorded proposal request: `temperature` 0, `max_tokens` 4096, `response_format` json_object, nothing added (no seed). Its messages, then the recorded proposal (assistant) and the arm's message (user). Serialized as the agent serialized it |
| Timeout, retries | 300 s, none |
| System prompt, prompt, memory | the recorded ones (`0efc36e2…`; the prompt and memory as recorded at the point), checked by hash |

## 2. Branch points

* **Design:** option A of
  [docs/audits/v0.10b-branch-point-inventory.md](../../audits/v0.10b-branch-point-inventory.md),
  approved by the owner after the inventory. The declared 2 + 2 per model
  is impossible for Gemma, which has one accepted-preview context. So:
  * Gemma: 1 accepted-preview point and all 3 refused-preview points;
  * Qwen: 2 accepted and 2 refused.
* **Candidates:** every V0.8c and V0.9c decision point with a staffing
  proposal (56).
* **Exclusions:**
  * the proposal's request or reply not verifiable against the
    recording;
  * the engine's preview failing, or too large for llm-agent 4 to show;
  * a context already taken.

  None of the 56 is excluded for evidence. One is a duplicate context
  (Gemma's staffing shortage, s42001, point 14: identical in the baseline
  and preview runs).
* **Stratum:** whether the engine's preview of the proposal's staffing
  actions at the recorded state accepts every action, or refuses one.
* **Selection** (`dreamteam-v10b-select/1`): per model and stratum,
  * first the V0.9c points at which llm-agent 4 showed the preview;
  * then the rest;
  * each group by SHA-256(`dreamteam-v10b-select/1|<recording>|<point>`).
  It reads no response of this experiment.
* **Points:** `points/manifest.json` (all candidates and the selection)
  and `points/contexts.jsonl` (SHA-256 `224585df…`). Each context holds
  the request, the proposal, the preview and both arms' messages. CI
  checks them again from the recordings and the engine
  (`decision_experiment.py points --check`).

| Point | Model | Preview | Refusal | Source | Point | Context | Proposal | Preview | Control request | Treatment request | Treatment message (chars) |
|---|---|---|---|---|---|---|---|---|---|---|---|
| ss-gemma-s42001-p14 | Gemma | accepted | — | V0.9c, shown | 14 | `d7635f40ef4a` | `6ba0fbc590fa` | `a5f57eca4c39` | `53ab9c30825a` | `62b70391cc0c` | 2,554 |
| fl-gemma-s41901-p27 | Gemma | refused | insufficient_funds | V0.8c, computed | 27 | `ce614fdbcd84` | `d49a413163f7` | `49323fdfbce8` | `86bb61205f87` | `35510cb804e1` | 1,289 |
| fl-gemma-s41901-p28 | Gemma | refused | invalid_schema | V0.8c, computed | 28 | `a77458bcace9` | `c97297d1fd60` | `fe4b195d728f` | `454f6c4b21bf` | `196e8c7d3eeb` | 809 |
| fl-gemma-s41901-p26 | Gemma | refused | invalid_schema | V0.8c, computed | 26 | `28c04507df84` | `10d0cde8cdd1` | `f33ce9dd7576` | `bfc071dc8148` | `e6c511ade32e` | 826 |
| ss-qwen-s42002-p6 | Qwen | accepted | — | V0.9c, shown | 6 | `f7f95ac5ecd1` | `984a995945c0` | `df279bede2c5` | `1adf12b6cdd0` | `35780873e02f` | 3,637 |
| ss-qwen-s42001-p1 | Qwen | accepted | — | V0.9c, shown | 1 | `51448e8d80c0` | `2943040fb23f` | `d4a084c596c8` | `2c61053b71e2` | `6a71c92c3a67` | 2,617 |
| ss-qwen-s42002-p26 | Qwen | refused | batch_rejected, notice_too_short | V0.9c, shown | 26 | `91622b395d1f` | `3e6a4a0e734a` | `ca60619689fe` | `2f026fdaadee` | `7ac79879d832` | 5,370 |
| ss-qwen-s42002-p1 | Qwen | refused | invalid_schema | V0.9c, shown | 1 | `10ffe075a34a` | `39b09c3b7a84` | `4be7076160b6` | `61361c917ea5` | `2c9fcb367cb7` | 911 |

Notes on the points:
* "Computed": no preview was shown in that run. The engine's preview at
  the replayed state is deterministic and checked again by CI.
* Gemma's three refused points are consecutive points of one V0.8c run,
  all with a teleradiology contract. Two are refused for a malformed
  action (`invalid_schema`), not for funding.
* At the five V0.9c points the treatment request is, byte for byte, the
  review request llm-agent 4 sent in V0.9c.

## 3. Arms

* **Control:** the neutral continuation, `bab212b0…`, 305 characters.
* **Treatment:** llm-agent 4's review message (`llm-agent-preview/v1`,
  `f8b7d787…`).

Both texts are in [SPEC.md](SPEC.md) §2 and `protocol.json`. Their final
paragraph, which carries the response schema and the request for a final
decision, is the same bytes. No advice, heuristic or persuasion is added.
The other differences are recorded per point (characters, SHA-256): the
preview, its introduction and the operations note.

## 4. Draws, order and running

* **Draws:** 20 per arm per point: 8 × 2 × 20 = **320 requests**. There is
  no inference preflight; the only preflight is GETs of `/health`,
  `/v1/models` and `/openapi.json`.
* **Order** (`dreamteam-v10b-order/1`):
  * Gemma's 160 requests, then Qwen's;
  * per model, 20 rounds, each with every point and arm once, in the
    order of SHA-256(`dreamteam-v10b-order/1|<round>|<point>|<arm>`);
  * the exact order is `schedule.json` (SHA-256 `c5c1e62bed9abb604bb5f4c2dd3ad048601305eb168fb9b080c4e958d99bb325`).
* **One request at a time.** No concurrent request.
* **Client:** `decision_experiment.py run`, on the Pi, under
  `systemd-run --user --scope -p MemoryMax=1G nice -n 10`.
* **Shared service:** no admin call, no model load or unload, no cache
  clear, no restart.
* **Server checks** before the run, before each round and after the run:
  `/health`, the server version, the OpenAPI SHA-256, and both models'
  entries in `/v1/models`. Other models on the shared server are not a
  change.
* **Stop:**
  * 5 consecutive transport failures;
  * 4 hours of wall clock;
  * an unhealthy `/health` before a round;
  * a material server change before a round (version, OpenAPI or either
    model's entry).

  A stopped run is reported as stopped, analysed as far as it went, and
  not resumed or rerun without the owner.
* **Failures:** every request is recorded. A failed request or an invalid
  reply is that draw's outcome. Nothing is retried, replaced or dropped.

## 5. Outcomes

**Primary,** per point and arm over the 20 draws, every draw counted:
* **kept:** the same actions as the proposal, in any order;
* **revised:** other actions;
* **abandoned:** no action;
* **invalid:** a failed request, or a reply llm-agent 3's strict parser
  rejects; details requests are not allowed.

The kept, revised and abandoned classes use llm-agent 4's rule. The
primary comparison is the treatment's distribution minus the control's,
with invalid replies reported apart and counted in every denominator.

Also per point and arm:
* the initial proposal's identity (SHA-256, the same for both arms);
* the final actions;
* the engine's validation (accepted, refused or no action);
* distinct decisions and distinct action proposals;
* latency and tokens.

**Secondary:**
* raw, canonical-JSON and action agreement within an arm;
* the direction and parameters of revisions (actions matched by type
  and employee, role or op; the fields that differ);
* accepted versus refused final decisions;
* financial evidence v1 (unchanged) of each distinct final decision at
  the recorded state:
  * AFFORDABILITY per batch, with the commitment of applied batches and
    credit support;
  * REVERSIBILITY and OPERATIONAL_EXPOSURE per executed action;
  * the commitment change against the proposal.

No composite score. No revised decision is called better because it
changed.

## 6. Statistical reporting

* Per point first:
  * counts and proportions with Wilson 95% intervals;
  * treatment minus control per outcome, with Newcombe hybrid-score 95%
    intervals;
  * the round-blocked table of control and treatment outcomes. The
    schedule puts both arms of a point in each round, so a round is a
    block. It does not make the two draws a natural pair.
* Then pooled, per model and overall: the per-point differences, their
  unweighted mean, and the number of points with a positive or negative
  difference. There is no interval over the points: they are 4 or 8
  clusters.
* Each interval treats a point's draws as repetitions of one request.
  That is an assumption, which V0.10a's variation results bear on.
* No hypothesis test, no population claim, and no small difference read
  as a definitive effect. The assumptions for a local intervention
  comparison are in [SPEC.md](SPEC.md) §8.

## 7. Continuation (fixed suffix)

Each distinct final decision at a point is run on the engine at the
recorded state:
* its batches are validated (staffing previewed first);
* if every batch is valid, each is applied in order;
* then every later day is advanced with no action.

A refused decision is not submitted. There is no retry here, so nothing
is applied.

Recorded: the apply results and their agreement with validation, the
digests after the decision and each day after, and the observed state at
the end. This is a mechanics check. No long-run effect is attributed to
the preview.

## 8. Replay

* `decision_experiment.py analyze` reads `live/requests.jsonl`, the
  points and the engine. It contacts no model.
* It rebuilds every request from the points and checks it against the
  recorded SHA-256, and checks the order against the schedule.
* CI regenerates the points' check, the schedule and the analysis, and
  compares them byte for byte across platforms.
* Live timing, tokens and cache counts (`inference.json`) are kept apart.

## Amendments

None.
