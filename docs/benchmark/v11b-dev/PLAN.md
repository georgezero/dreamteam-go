# V0.11b controlled evaluation of the validation-first workflow: prospective plan

`v11b-workflow-1` (`protocol.json`, SHA-256 `6de900d7a7a21666fe53c492c25516a33daf9962d378f2dd0f3ed6bab6695408`; schema
`dreamteam.shared-prefix/v4`, specified in [SPEC.md](SPEC.md)). This plan
was committed before any request of this experiment was sent. A change
after that commit is an amendment, recorded at the end with its reason.
None is planned.

**The question.** From the same recorded state and initial staffing
proposal, do three whole decision workflows lead to different immediate
decisions?
* **A:** llm-agent 3's validation and retry.
* **B:** llm-agent 5's validation-first workflow with the preview
  disabled.
* **C:** llm-agent 5 in full.

Above all: does the selective preview (C − B) change decisions? At valid
proposals, does it reinforce financially consequential ones? At refused
proposals, does it help produce valid corrections?

Success does not depend on llm-agent 5 being better. It is enough to
distinguish the three workflows reliably.

## 1. Models, settings, agents, material

| Item | Value |
|---|---|
| Models | `gemma-4-26B-A4B-it-oQ4e-mtp`, `Qwen3.6-35B-A3B-oQ4e-mtp`, the V0.5c configurations, unchanged; no substitute |
| Server | oMLX 0.7.0, through `DREAMTEAM_LLM_BASE_URL`, never recorded |
| Request | the recorded proposal request, unchanged (temperature 0, `max_tokens` 4096, json_object, no seed), its messages followed by the workflow's turns |
| Agent settings | `maxRetries` 1, `allowDetails` true (the frozen settings files); the decision budget's wall clock is not applied (it is not reached by one point's workflow, and is not reproducible) |
| Agents | llm-agent 3 (`ef2c17a0…`), llm-agent 5 (`llm-agent-workflow/v1`, `3ee7d66f…`) and the prompt `llm-agent-prompt/v2` (system prompt `0efc36e2…`), all unchanged. B is the experiment configuration `v11b-B-validation-only` of the driver |
| Material | development only: the committed V0.8c and V0.9c recordings (seeds 41901, 41902, 42001, 42002). No clinical, reference, reserved or sealed holdout seed. No new live agent trajectory |

## 2. Branch points

* **Inventory:** the 56 V0.8c and V0.9c staffing proposals of the V0.10c
  inventory, with financial evidence v1 and preview fields for each,
  checked again by its CI step.
* **Excluded:** 2 proposals with a refusal before them at their decision
  point, whose arms would differ before the branch.
* **Selection** (`dreamteam-v11b-select/1`; `points/manifest.json`), no
  model response read:
  * **Gemma:** every distinct eligible context: 1 valid and 3 refused,
    all it has.
  * **Qwen, 5 valid:** for each category in turn (credit-supported,
    retained obligation, termination, allocation change), the context
    with that category and the largest stake not yet taken; then the
    largest stake.
  * **Qwen, 5 refused:** insufficient funds (2), timing (2: lead time,
    notice, date, Monday, commitment length), any other refusal (1).
    Each by stake.
  * **Ties and limits:** ties by SHA-256(`dreamteam-v11b-select/1|<recording>|<point>`);
    a context is taken once; at most 2 points from one recording.
* **Large guaranteed commitments:** covered by the stake ordering within
  each category.
* **The $44,687.30 credit-supported proposal** (ss-qwen-pv-s42002-p6),
  which remains complete, is selected as the only credit-supported
  proposal in the recordings.

**Selected** (`points/contexts.jsonl` SHA-256 `9d4e7644198d…`):

| Point | Model | Proposal | Category | Refusal | Stake | Prior exposure | Context | Proposal | Prompt | Failures before |
|---|---|---|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | Gemma | accepted | retained_obligation | — | $3,069.98 | V10B, V10C | `d7635f40ef4a` | `6ba0fbc590fa` | `f407b0c43d0f` | 0 |
| fl-gemma-s41901-p28 | Gemma | refused | other | invalid_schema | $0.00 | V10B, V10C | `a77458bcace9` | `c97297d1fd60` | `5d372cee04a0` | 0 |
| fl-gemma-s41901-p27 | Gemma | refused | insufficient_funds | insufficient_funds | $1,464.51 | V10B, V10C | `ce614fdbcd84` | `d49a413163f7` | `bb68ea98d3da` | 0 |
| fl-gemma-s41901-p26 | Gemma | refused | other | invalid_schema | $0.00 | V10B, V10C | `28c04507df84` | `10d0cde8cdd1` | `4d16d5f60bcc` | 0 |
| ss-qwen-pv-s42002-p6 | Qwen | accepted | credit_supported | — | $44,687.30 | V10B, V10C | `f7f95ac5ecd1` | `984a995945c0` | `444cf671c2a7` | 0 |
| ss-qwen-s42002-p7 | Qwen | accepted | retained_obligation | — | $3,228.33 | V10C | `b6bd84a6a1ab` | `5e9186112ab8` | `cf28e6389f34` | 0 |
| fl-qwen-s41902-p24 | Qwen | accepted | termination | — | $0.00 | none | `bf538ed7e6a3` | `af9ed6d0e93f` | `1058510a6aa1` | 0 |
| fp-qwen-s41901-p26 | Qwen | accepted | allocation | — | $9,863.02 | V10C | `c4ce47e15a79` | `81f7024330d1` | `c640deb912cf` | 0 |
| fp-qwen-s41901-p25 | Qwen | accepted | allocation | — | $9,863.02 | none | `19ab1f70c5c0` | `5f270860dba8` | `7e5bd57fa965` | 0 |
| fl-qwen-s41901-p9 | Qwen | refused | insufficient_funds | insufficient_funds | $36,978.91 | V10C | `8dfa0efade4a` | `25e6d5e2d31c` | `f53cfc659c53` | 0 |
| fl-qwen-s41901-p11 | Qwen | refused | insufficient_funds | insufficient_funds | $24,657.53 | V10C | `d518498303e9` | `318af3f85e2b` | `227bd826c9ed` | 0 |
| fp-qwen-s42001-p7 | Qwen | refused | timing | lead_too_short | $0.00 | none | `e2842f486904` | `393ec51b99bb` | `4b562de8c11f` | 0 |
| fl-qwen-s41902-p27 | Qwen | refused | timing | notice_too_short | $0.00 | none | `fe71c99abdfe` | `3dce2c264320` | `63f9319c1b01` | 0 |
| fp-qwen-s42001-p24 | Qwen | refused | other | contract_overlap | $0.00 | none | `d2a61dae8eee` | `98de6e2ef0b0` | `81077f665aab` | 0 |

The scenario fingerprints are the recordings' replays' (checked by
digests at each point).

**Prior exposure.** Nine points were in V0.10b or V0.10c:
* Gemma's four, by its rule;
* the credit-supported proposal;
* four other Qwen points, by stake.

Five are new. The rule reads no response of V0.10b or V0.10c. It does not
select points by whether a difference appeared there, but it was written
after their results were known.

**Limitations of the corpus:**
* Gemma has one valid proposal and no credit-supported one.
* The only valid termination selected has no measured commitment, though
  it has severance and displaced bookings.
* Two of the five valid Qwen points are similar allocations from one run
  (`fp-qwen-s41901-p25` and `-p26`).

## 3. Arms

As [SPEC.md](SPEC.md) §2. Every arm starts with:
* the recorded state;
* the recorded conversation;
* the recorded initial proposal as the model's first reply;
* the same model, settings and action catalog.

**The primary comparison** is C − B; the secondary B − A.

**Model calls** differ by workflow and are counted, not equalized. At a
valid proposal, A and B call no model and submit the proposal; their
draws repeat one deterministic result.

## 4. Draws, order and running

* **Draws per arm per point:** Gemma 10 and Qwen 20, as in V0.10c. In
  V0.10b and V0.10c, Gemma gave one decision per arm at every point, and
  Qwen gave 1–6 distinct decisions per arm in 20 draws.
* **Totals:** 4 × 3 × 10 + 10 × 3 × 20 = **720 draws**, at most 4 model
  calls each (1 + `maxRetries` + a preview review + a details answer), so
  at most 2,880 calls.
* **Order** (`dreamteam-v11b-order/1`):
  * Gemma's draws, then Qwen's;
  * per model, each round has every point and arm once, in the order of
    SHA-256(`dreamteam-v11b-order/1|<model>|<round>|<point>|<arm>`);
  * the exact order is `schedule.json` (SHA-256 `1df609865f9557b7fae9ca69fa8c1d2956e8ef72e993376fe09ab8bde9c8516a`).
* **One draw, and one model request, at a time.** No concurrency.
* **Engine:** one session per point on the recording's replay transcript,
  at the recorded digests. Only read-only requests: validate, preview,
  observe, digests.
* **Client:** the Pi, under `systemd-run --user --scope -p MemoryMax=2G
  nice -n 10`.
* **Shared server:** no admin call, no load or unload, no cache clear, no
  restart.
* **Server checks** before the run, each round and after.
* **Stop:**
  * 5 consecutive transport failures;
  * 4 hours of wall clock;
  * an unhealthy `/health`;
  * a material server change.

  A stopped run is reported and analysed as far as it went. It is not
  resumed without the owner.
* **Failures:** every request is recorded. A failure is handled only by
  the arm's own workflow (a resend within its retries), never rerun
  outside it.

## 5. Outcomes

**Primary,** per point, model and arm, every draw counted:
* **the decision against the proposal:** kept, revised, abandoned,
  refused or invalid;
* **the final decision:** accepted, no action, or none submitted;
* **the workflow:** retries and model calls; preview shown, its status
  and eligibility reasons; the response after a shown preview;
* **the final submitted actions:** the distribution of canonical action
  content.

**Comparisons:** primary C − B, secondary B − A (and C − A), per point.

**Financial,** with financial evidence v1 unchanged, for each final
decision:
* initial and final commitment, and the change;
* cash-only funding headroom at the point;
* credit-supported commitments;
* retained obligations;
* severance and reversibility (REVERSIBILITY records);
* displaced bookings;
* direct capacity changes.

**Preview nudge** (per point, before any pooled result):
* at valid proposals: kept under B against C, the commitment under B
  against C, and credit support under B against C;
* at refused proposals: valid corrections (final accepted) and
  abandonment, under B against C.

No composite score. No reason for a decision is inferred. A smaller
commitment is not called better, and an accepted preview is not called
good judgment.

## 6. Statistics

* **Per point first:**
  * Wilson 95% intervals per arm;
  * Newcombe 95% intervals for C − B, B − A and C − A, per outcome,
    final status and credit support;
  * whether the action-content distributions are identical.
* **Then pooled,** per model and proposal validity and overall:
  per-point differences and their unweighted mean, with no interval over
  clustered points.
* **Kept separate:**
  * workflow effects (between arms);
  * inference variability (within an arm; none for A and B at a valid
    proposal);
  * branch-point heterogeneity;
  * model differences (Gemma and Qwen apart);
  * financial exposure changes.
* **Descriptive only.** No hypothesis test. Draws are repetitions of one
  workflow, not independent practice cases. Nothing is generalized beyond
  these points. The assumptions for a local causal reading are in
  [SPEC.md](SPEC.md) §5.

## 7. Continuation and replay

* **Continuation:** each distinct final decision is applied, if valid, at
  the recorded state, followed by the no-action suffix (shared-prefix/v2)
  with financial evidence v1. It is a mechanics check, with no long-run
  claim.
* **Replay:** every draw's workflow is run again without the model:
  * the recorded responses, each matched to its request's SHA-256;
  * a fresh engine session;
  * the trace and every engine answer must equal the recorded ones.
* **Cross-platform:** CI regenerates the points' check, the schedule and
  the analysis and compares them on every platform. Latency, tokens and
  cache are reported apart.

## Amendments

None.
