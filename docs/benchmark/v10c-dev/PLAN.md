# V0.10c validation feedback against preview information: prospective plan

`v10c-feedback-1` (`protocol.json`, SHA-256 `7a25e9870e8a42e7dea9b97c8a8f97d01ca60d9113e265434977458bf4509843`; schema
`dreamteam.shared-prefix/v3`, specified in [SPEC.md](SPEC.md)). This plan
was committed before any request of this experiment was sent. A change
after that commit is an amendment, recorded at the end with its reason.
None is planned.

**The question.** At a recorded branch point, the state, conversation and
initial staffing proposal are held fixed. Does the engine's full preview
change the model's final decision beyond what the engine's validation
answer for the same actions does?

The primary comparison is preview against validation (C − B). The
neutral continuation (A) is a reference for how much either kind of
feedback changes the decision. The outcome is the immediate decision.

## 1. Models, settings, material

| Item | Value |
|---|---|
| Models | `gemma-4-26B-A4B-it-oQ4e-mtp`, `Qwen3.6-35B-A3B-oQ4e-mtp`, the V0.5c configurations, unchanged; no substitute |
| Server | oMLX 0.7.0 (OpenAPI `67525ae2…`), through `DREAMTEAM_LLM_BASE_URL`, never recorded |
| Request | the recorded proposal request, unchanged (temperature 0, `max_tokens` 4096, json_object, no seed), then the proposal and the arm's message |
| Timeout, retries | 300 s, none |
| Material | development only: the committed V0.8c and V0.9c recordings (seeds 41901, 41902, 42001, 42002). No clinical, reference, reserved or sealed holdout seed. No new live agent trajectory |

## 2. Branch points

**Inventory first** (`points/inventory.json`; [API_INVENTORY.md](API_INVENTORY.md)
§2). The 56 staffing proposals have, by the validation of their staffing
actions at the recorded state and financial evidence v1 of the whole
proposal:

| Model | Accepted | Refused | Credit-supported | Retained obligation | Withdrawable | Not reversible | Displaces bookings | Insufficient funds |
|---|---|---|---|---|---|---|---|---|
| Gemma | 2 (1 distinct context) | 3 | 0 | 2 (the one context) | 0 | 0 | 0 | 1 |
| Qwen | 17 | 34 | 1 | 4 | 8 | 8 | 5 | 5 |

* Only one proposal in all the recordings is credit-supported:
  ss-qwen-pv-s42002-p6.
* Gemma still has one accepted-preview context, as in V0.10b.

**Design** (declared here, from the inventory):
* **Gemma:** every distinct eligible context, 1 accepted and 3 refused.
  These are V0.10b's four, by the same rule.
* **Qwen, 4 accepted:** for each category in turn (credit-supported,
  retained obligation, withdrawable, displaces bookings, not reversible),
  the context with that category and the largest stake not yet taken;
  then the largest stakes.
* **Qwen, 4 refused:** `insufficient_funds` refusals first, then by stake.
* **Stake:** the larger of the preview's funding commitment and financial
  evidence v1's applied commitment.
* **Ties and limits:** ties by SHA-256(`dreamteam-v10c-select/1|<recording>|<point>`);
  a context is taken once; at most 2 points from one recording.
* **Blind to outcomes:** the rule reads no model response of V0.10b or
  V0.10c. It was written after V0.10b's results were known, and V0.10b's
  ss-qwen-pv-s42002-p6 is selected because it is the only credit-supported
  proposal.

**Selected** (`points/manifest.json`, `points/contexts.jsonl` SHA-256
`a27304c9…`):

| Point | Model | Validation | Refusal | Categories | Stake | Context | Proposal | Validate | Preview | Message chars A / B / C | Requests A / B / C |
|---|---|---|---|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | Gemma | accepted | — | retained_obligation | $3,069.98 | `d7635f40ef4a` | `6ba0fbc590fa` | `7e827342ea49` | `a5f57eca4c39` | 305 / 1,087 / 2,554 | `53ab9c30` / `e5f82a7f` / `62b70391` |
| fl-gemma-s41901-p26 | Gemma | refused | invalid_schema | — | $0.00 | `28c04507df84` | `10d0cde8cdd1` | `86a05dd6b3ca` | `f33ce9dd7576` | 305 / 794 / 826 | `bfc071dc` / `bc8d2340` / `e6c511ad` |
| fl-gemma-s41901-p28 | Gemma | refused | invalid_schema | — | $0.00 | `a77458bcace9` | `c97297d1fd60` | `a40f14032cd5` | `fe4b195d728f` | 305 / 777 / 809 | `454f6c4b` / `7a616446` / `196e8c7d` |
| fl-gemma-s41901-p27 | Gemma | refused | insufficient_funds | insufficient_funds | $1,464.51 | `ce614fdbcd84` | `d49a413163f7` | `4d57a3187340` | `49323fdfbce8` | 305 / 1,050 / 1,289 | `86bb6120` / `06fa17a6` / `35510cb8` |
| ss-qwen-pv-s42002-p6 | Qwen | accepted | — | credit_supported, withdrawable, not_reversible | $44,687.30 | `f7f95ac5ecd1` | `984a995945c0` | `f1861486e9cb` | `df279bede2c5` | 305 / 1,567 / 3,637 | `1adf12b6` / `fb83b8cd` / `35780873` |
| ss-qwen-s42002-p7 | Qwen | accepted | — | retained_obligation, not_reversible | $3,228.33 | `b6bd84a6a1ab` | `5e9186112ab8` | `dd59f85ce38e` | `154e61e0dac7` | 305 / 1,191 / 2,663 | `e0d0a3a4` / `dc13ef25` / `dd347b00` |
| fp-qwen-s41901-p26 | Qwen | accepted | — | withdrawable | $9,863.02 | `c4ce47e15a79` | `81f7024330d1` | `64ea21237a65` | `021155893dd5` | 305 / 1,265 / 2,434 | `a3dbe014` / `8b4b8422` / `43d888b9` |
| ss-qwen-s42001-p20 | Qwen | accepted | — | retained_obligation, withdrawable, displaces_bookings | $1,475.80 | `f956014b01c4` | `83c88d784f03` | `460539590a8f` | `ceebabeb2dba` | 305 / 7,886 / 100,937 | `338a5e9b` / `0d5465de` / `10e54b93` |
| fl-qwen-s41901-p9 | Qwen | refused | insufficient_funds | insufficient_funds | $36,978.91 | `8dfa0efade4a` | `25e6d5e2d31c` | `84a1b11dc9fe` | `9d9596f32e7d` | 305 / 2,055 / 2,300 | `3e96dedc` / `79de67a5` / `ed4481ba` |
| fl-qwen-s41901-p11 | Qwen | refused | insufficient_funds | insufficient_funds | $24,657.53 | `d518498303e9` | `318af3f85e2b` | `8c8f14696866` | `cb51987f7474` | 305 / 1,007 / 1,253 | `c4fda00a` / `dd13438d` / `b473a8b0` |
| ss-qwen-s42001-p23 | Qwen | refused | past_date | — | $0.00 | `e34d46458910` | `e67bb6f034cf` | `cea5d7767964` | `5b5418d4f3c6` | 305 / 869 / 905 | `39cb6fb7` / `959aa28b` / `d917d2ad` |
| fp-qwen-s42001-p27 | Qwen | refused | invalid_schema | — | $0.00 | `e3b50b952686` | `2c60c0cf376a` | `ea0e22ab94bd` | `cf94c9dc0df1` | 305 / 936 / 968 | `9a65cedb` / `a1048182` / `756a7624` |

`-pv` marks a V0.9c preview-run recording, whose point is a V0.9c preview
point.

**Overlap with V0.10b:**
* Gemma's four points are V0.10b's four.
* ss-qwen-pv-s42002-p6 is V0.10b's ss-qwen-s42002-p6.
* The other seven Qwen points are new.

**Points with large previews:** ss-qwen-s42001-p20's preview message is
100,937 characters. It is within llm-agent 4's bound, so it is
eligible.

## 3. Arms

From [SPEC.md](SPEC.md) §2, with texts and hashes in `protocol.json`:

| Arm | Message |
|---|---|
| **A, neutral** | V0.10b's control |
| **B, validation** | the engine's exact validate answer for the proposal's staffing actions, in a frame built like llm-agent 4's |
| **C, preview** | llm-agent 4's exact review message with the engine's preview of the same actions |

* All three end with the same final paragraph. The only text added for
  B is its two framing sentences and its operations note. No financial
  advice, heuristic or persuasion is added.
* **No length control.** A length-matched neutral message was
  considered. It would need padding, which is either a new stimulus (a
  repeated or irrelevant text) or invented content. Each arm's message
  length is reported per point instead. Matching length would not make
  the information equal in any case.

## 4. Draws, order and running

* **Draws per arm per point:** Gemma 10, Qwen 20. In V0.10b, Gemma gave
  one decision per arm in all 20 draws at each of its points, and Qwen
  gave 1–5 distinct decisions per arm.
* **Total:** 4 × 3 × 10 + 8 × 3 × 20 = **600 requests**. There is no
  inference preflight; the only preflight is GETs of `/health`,
  `/v1/models` and `/openapi.json`.
* **Order** (`dreamteam-v10c-order/1`):
  * Gemma's 120 requests, then Qwen's 480;
  * each round has every point and arm of the model once, in the order
    of SHA-256(`dreamteam-v10c-order/1|<model>|<round>|<point>|<arm>`);
  * the exact order is `schedule.json` (SHA-256 `99a20032226629b47d75235bcf7e40cf49f2a7381cfea39a1c6a5f58151d64b7`).
* **One request at a time.** The client runs on the Pi under
  `systemd-run --user --scope -p MemoryMax=1G nice -n 10`.
* **Shared service:** no admin call, no model load or unload, no cache
  clear, no restart.
* **Server checks** before the run, each round and after: `/health`, the
  version, the OpenAPI SHA-256, and both models' entries.
* **Stop:**
  * 5 consecutive transport failures;
  * 4 hours of wall clock;
  * an unhealthy `/health`;
  * a material server change.

  A stopped run is reported as stopped and analysed as far as it went. It
  is not resumed or rerun without the owner.
* **Failures:** every request is recorded. A failed request or an invalid
  reply is that draw's outcome. Nothing is retried, replaced or dropped.

## 5. Outcomes

**Primary,** per point and arm, every draw in the denominator:
* kept, revised, abandoned or invalid (llm-agent 4's rule against the
  recorded proposal; llm-agent 3's strict parser);
* the final decision accepted, refused or no action (the engine's
  validation at the recorded state);
* the exact action content (the distribution of canonical action lists).

The primary estimand is the difference in these distributions,
**preview minus validation**, at each point. B − A and C − A are
references.

**Financial,** with financial evidence v1 unchanged, of each distinct
final decision at the recorded state:
* the initial and final commitment and their difference;
* cash-only funding headroom at the point;
* credit-supported commitments;
* retained obligations;
* reversibility;
* direct operational exposure;
* displaced bookings.

**Kept apart for every draw:**
1. the information received (the arm's message length; what B and C
   carry: API_INVENTORY §4);
2. the proposal changed (revised or abandoned);
3. the proposal became valid (a refused proposal's final decision
   accepted);
4. the commitment changed;
5. the direct exposure changed.

No composite score. A smaller commitment is not called better, and a
valid one not wise.

## 6. Statistical reporting

* **Per point first:**
  * counts with Wilson 95% intervals per arm;
  * C − B, B − A and C − A per outcome and final status, with Newcombe
    95% intervals;
  * the round-blocked table per comparison;
  * whether the action-content distributions are identical.
* **Then pooled,** per model and stratum and overall: the per-point
  differences, their unweighted mean, and the number of points with a
  positive or negative difference. No interval over clustered points.
* **Kept separate:**
  * the validation-feedback effect (B − A);
  * the incremental preview effect (C − B);
  * inference variation (distinct values within an arm);
  * branch-point heterogeneity (the spread of per-point differences).
* Each interval treats a point's draws as repetitions of one request.
  Gemma's intervals, if its answers repeat as in V0.10b, describe
  repetition, not sampling.
* No hypothesis test. No population claim about models, prompts or
  practices. No draw is treated as an independent clinical case.

## 7. Continuation

As V0.10b:
* each distinct final decision validated, staffing previewed first;
* applied in order if every batch is valid;
* then every later day advanced with no action.

A refused decision is not submitted.

Recorded: the apply results and their agreement with validation, the
digests, the observed state at the end, and financial evidence v1. This
is a mechanics check. No long-run effect is attributed to the preview.

## 8. Replay

* The analysis reads `live/requests.jsonl`, the points and the engine.
  It contacts no model.
* It checks:
  * every request's and response's SHA-256;
  * every content against its response;
  * the order against the schedule;
  * the validate and preview answers against a fresh engine session.
* CI regenerates the inventory, the points' check, the schedule and the
  analysis, and compares them byte for byte across platforms.
* Live timing, tokens and cache counts (`inference.json`) are kept apart.

## Amendments

None.
