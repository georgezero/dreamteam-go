# V0.13c draft prospective protocol: management alternatives at the decision

**Status: DRAFT — NOT LOCKED — NOT AUTHORIZED.**
* The machine-readable draft is
  `docs/benchmark/v13c-dev/protocol-draft.json` (`v13c-alt-1`).
* **It may be locked only when all of the following hold:**
  1. the technical review is clean (`docs/audits/v0.13c-independent-review.md`
     and the response);
  2. the human presentation review is complete and accepted
     (`PRESENTATION_REVIEW.md`, now PENDING);
  3. the owner has decided D1–D10 (`EXPERIMENTAL_DESIGN.md` §8), D7, the
     threat model for free-text fields, among them;
  4. V0.13d has implemented and reviewed the agent workflow and driver
     this draft names.
* **A lock does not authorize inference.** Only the owner's explicit,
  quoted authorization does.
* No model was contacted to prepare this draft.

## 1. Question and primary outcome

The question is in [EXPERIMENTAL_DESIGN.md](EXPERIMENTAL_DESIGN.md) §1.

**Primary outcome:** decision sensitivity per sensitivity pair and model,
Δ_treatment − Δ_control of P(departure) (§7).

**Secondary outcomes:**
* the same for "no action submitted";
* per historical point, the treatment − control difference in
  departure;
* the `inspected` and `compared` flags;
* model calls and tokens.

## 2. Models and settings

| | |
|---|---|
| Models | `gemma-4-26B-A4B-it-oQ4e-mtp` (gemma4-26b-a4b), `Qwen3.6-35B-A3B-oQ4e-mtp` (qwen36-35b-a3b), on the oMLX server of V0.11b and V0.12b, reached through `DREAMTEAM_LLM_BASE_URL` (never recorded or committed); no substitute model |
| Settings files | `docs/benchmark/v05c-dev/llm-gemma4-26b-a4b.json` (`1a1edae5…`), `docs/benchmark/v05c-dev/llm-qwen36-35b-a3b.json` (`e19a47dc…`), loaded by `llm_agent3.load_settings`, unchanged |
| Request | temperature 0, `max_tokens` 4096, `response_format` json_object, no seed; timeout 300 s; no retries beyond the workflow's own |
| Frozen workflow settings | `maxRetries` 1, `allowDetails` true |
| System prompt | `llm-agent-prompt/v2`, SHA-256 `0efc36e2…` (unchanged) |
| Server | no admin call, load, unload, cache clear or restart; `/health`, `/v1/models`, `/openapi.json` before the run, before each round and after; a material change stops the run |

## 3. Agent versions and arms

| Arm | Agent and workflow | Turn text |
|---|---|---|
| A, control | llm-agent 7 (V0.13d, not yet written): llm-agent 5's validation-first workflow (`llm-agent-validation-first-workflow/v1`, `a0204f62…`) and the control turn | `dreamteam.alternatives-control/v1`, `331a1542…` |
| B, treatment | llm-agent 7: the same workflow and the alternatives turn | `dreamteam.alternatives-presentation/v1`, `5bf586cf…` (or v2, if D2 is decided so) |

* Both arms' turns come from `alternatives.Steps` (engine 0.13.0-dev.2
  at this draft, pinned at lock), using rule
  `dreamteam.alternative-selection/v2` (`5f32b2c8…`), with the
  information boundary in code and its trusted-input threat model
  (EXPERIMENTAL_DESIGN.md §3, D7). The turn is due only when the engine
  accepts the reply's operations batch as well as its staffing batch.
* The workflow identity (version and SHA-256) and the driver's are
  pinned at lock.
* If the owner chooses D1's alternative, A is llm-agent 5's
  validation-first workflow unchanged.

## 4. Units

**Historical branch points** (`points/manifest.json`, `dreamteam-v13c-select/1`):
* nine points;
* the recorded state, request and conversation;
* the recorded proposal as the model's first reply;
* each point run with its recording's model only.

| Point | Model | Category | Turn due at the recorded proposal |
|---|---|---|---|
| fl-qwen-s41902-p24 | Qwen | termination_displacement | yes |
| ss-qwen-s42001-p20 | Qwen | staffing_adjustment_displacement | yes |
| ss-qwen-s42002-p7 | Qwen | retained_obligation | yes |
| ss-qwen-s42001-p1 | Qwen | retained_obligation | yes |
| ss-gemma-pv-s42001-p14 | Gemma | retained_obligation | yes |
| fp-qwen-s42001-p7 | Qwen | refused_qualifying | after a valid correction |
| fl-gemma-s41901-p28 | Gemma | refused_qualifying | after a valid correction |
| ss-qwen-pv-s42002-p6 | Qwen | credit_supported_not_due | no (A/A) |
| fp-qwen-s41901-p25 | Qwen | nonqualifying_control | no (A/A) |

**Constructed sensitivity members** (`sensitivity-pairs.json`):
* twelve members (S1–S6, a and b) on mini, seed 52;
* each run with both models;
* the agent's prompt built at the state by `prepare` with empty memory;
* the scripted proposal as the first reply.

**Seeds:** only development seeds: the recordings' 41901, 41902, 42001
and 42002, and mini 52. No reference, holdout, reserved or clinical
seed. The driver calls `eval.GuardManifest` before opening any session.

## 5. Draws, schedule and budget

**Draws per arm per unit:**

| Units | Gemma | Qwen |
|---|---|---|
| Historical points where the arms can differ (7) | 10 | 20 |
| Historical A/A points (2; no model call is possible: the validated proposal is submitted) | — | 1 |
| Constructed members (12) | 10 | 20 |

**Totals:**
* historical: 2 × 2 × 10 + 5 × 2 × 20 + 2 × 2 × 1 = 244;
* constructed: 12 × 2 × (10 + 20) = 720;
* **964 draws**.

**Calls per draw:**
* a validation feedback turn (at a refused proposal), one inspect round,
  one decision reply, and the workflow's one retry: at most 4;
* the limit of **5** per draw is a guard: if it is reached, the run
  stops;
* the run's limit is **4,820 calls**.

**Expected:** about 1,500 to 2,000 calls. Due units need 1–3, A/A
points none.

**The order (`dreamteam-v13c-order/1`):**
* per model, Gemma first, then Qwen;
* rounds; in each round every unit and arm of the model with draws left
  once, in the order of
  SHA-256(`dreamteam-v13c-order/1|<model>|<round>|<unit>|<arm>`);
* the exact schedule is made and pinned at lock;
* one draw and one model request at a time; no concurrency.

**Time limit:**
* the run stops at **8 hours**: a request starts only if its whole
  300 s timeout fits;
* no token cap beyond `max_tokens`; tokens recorded per call.

## 6. Running, failures and partial runs

**Client:** the Pi, under `systemd-run --user --scope -p MemoryMax=3G
nice -n 10`. That covers up to about 33 engine sessions (one per unit
and arm of the model being run) and the driver. The exact figure is
measured by the V0.13d mock run.

**Failures:**
* Every request and answer is recorded.
* A failure is handled only by the arm's own workflow (its one retry),
  never rerun outside it.

**Stop conditions:**
* the per-draw or run call limit;
* the time limit;
* 5 consecutive transport failures;
* an unhealthy `/health`;
* a material server change;
* a driver failure;
* the operator.

**A stopped run is PARTIAL:**
* the interrupted draw is recorded and replayed, not classified;
* `run.json` says PARTIAL, with the coverage;
* the analysis reports only what was observed, with no Δ_treatment −
  Δ_control for a pair missing any cell;
* nothing is resumed, substituted or rerun. Any continuation needs the
  owner's authorization and a recorded amendment.

**Authorization:** `run` requires the owner's authorization, quoted
verbatim and recorded in `run.json`.

## 7. Analysis (fixed before any request)

**Per unit, model and arm:**
* the result distribution (§6 of the design);
* P(departure) and P(no action submitted), each with a Wilson 95%
  interval;
* the flags, calls and tokens.

**Historical points:** P_B − P_A per point, with Newcombe's hybrid
score 95% interval.

**Sensitivity pairs, per model:**
* Δ_arm = P_arm(b) − P_arm(a), with Newcombe's interval;
* sensitivity Δ_B − Δ_A, with a MOVER interval (Zou and Donner)
  built from the two Newcombe intervals.

**What is not done:**
* No hypothesis test, composite score, ranking or pooling across pairs,
  points or models.
* Gemma's draws may be identical at temperature 0, which gives
  degenerate intervals. They are reported as observed.

**What it can say:**
* how much the decision followed the named fact with the comparison,
  beyond without it;
* for each pair, the other listed differences the decision could also
  have followed (SENSITIVITY_CASES.md §3).

**What it cannot say:** that a decision was better, cheaper, safer or
clinically appropriate.

## 8. Replay

**Every recorded draw is replayed without a model:**
* the same session pattern, every engine request and answer identical,
  and every model reply taken from the record;
* everything but send times and latencies identical;
* the analysis made again from the replay, equal to the committed one;
* CI replays the run on every platform, as for V0.12b.

## 9. Limitations

See [LIMITATIONS.md](LIMITATIONS.md).
