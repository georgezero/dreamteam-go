# V0.13d revised protocol draft: `v13d-alt-1`

**Status: DRAFT — NOT LOCKED — NOT AUTHORIZED.** It is prepared for the
owner. It is not frozen and must not be run: `run` refuses any protocol
that is not `LOCKED — NOT RUN`, and needs the owner's authorization text
as well. The 964-draw V0.13c draft (`v13c-alt-1`) was not approved and is
replaced by this one.

**Files** (`docs/benchmark/v13d-dev/`): `protocol-draft.json`
(`57aed043…` before the review; see §8 for the hashes at commit),
`schedule-draft.json`, `contexts/contexts.jsonl`. Both JSON files are made
by the driver from the committed inputs alone (`v13d.py protocol`,
`schedule`), and `test_v13d.py` requires them identical to what the inputs
make. This fixes V0.13c's limitation that nothing re-derived the draft.

## 1. Models and settings

| Model | Identifier | Settings file |
|---|---|---|
| Gemma | `gemma-4-26B-A4B-it-oQ4e-mtp` | `docs/benchmark/v05c-dev/llm-gemma4-26b-a4b.json` |
| Qwen | `Qwen3.6-35B-A3B-oQ4e-mtp` | `docs/benchmark/v05c-dev/llm-qwen36-35b-a3b.json` |

The settings are frozen and loaded by the agent's own loader, unchanged:
temperature 0, `max_tokens` 4096, `json_object`, request timeout 300 s,
`maxRetries` 1, details allowed, the built-in system prompt (`0efc36e2…`).
Gemma runs first, then Qwen; one model's draws at a time.

## 2. Texts and identities

| Identity | SHA-256 |
|---|---|
| Workflow `llm-agent-alternatives-workflow/v1` | `1e360348d2e689952d1911aa0e9d4323e8fbbb6c74edd751a833c2d7bf8a8ccb` |
| Its base, `llm-agent-validation-first-workflow/v1` | `a0204f628c7631d43ebfbb48e3b6e31071b67252daa76f141e3ad3a43ff5f184` |
| Presentation v2 template | `8327f254ed2e25865db781f5f1cdcc70db129ea0df72ab6d0ce924ddf79ce496` |
| View table (`dreamteam.alternatives-view/v1`) | `ce7f0e238c2d8cf71bf87cf720f12b56bd667035971308ae9abae7399d66aba0` |
| Selection rule v2 | `5f32b2c84a0b8383f1a213ce52745e56f5d5aa8292bc714096d1beec70debcf9` |
| Arms | `dreamteam.alternatives-arms/v2` |
| Driver | `dreamteam-v13d-driver/1` |

`requests` in the protocol gives, per context and model, the SHA-256 of
the first request (the one the initial proposal answers). Every draw's
first request must be that one, byte for byte.

## 3. Contexts and arms

* **Historical (9):** the V0.13c selection (`dreamteam-v13c-select/1`, from
  the V0.12b pool), each with the model that made it. Five are due at the
  recorded proposal (1 Gemma, 4 Qwen); two are refused proposals whose turn
  is due only after the model corrects them (1 each); two are A/A controls
  where the turn is never due (Qwen).
* **Constructed (12):** the six sensitivity pairs, both members, run with
  both models.
* **Arms:** A, control: validation-first, then the shared head and foot.
  B, treatment: validation-first, then the shared head, the verified
  comparison (one inspect round) and the shared foot. The arms share the
  state, the request, the proposal, the settings, the memory, the decision
  cadence and every instruction before the turn.

## 4. Draws, calls and time: a smaller, bounded design

| | Gemma | Qwen | Draws |
|---|---|---|---|
| Draws per arm per unit | 3 | 5 | |
| Draws per arm at A/A contexts | — | 2 | |
| Historical draws | 12 | 58 | 70 |
| Constructed draws | 72 | 120 | 192 |
| **All draws** | **84** | **178** | **262** |

* **Model calls.** A draw's bound is 3 calls after its initial proposal at a
  valid proposal (the turn's reply, a reply after inspect, one retry) and
  5 at a refused one. The run's budget is the sum of every scheduled
  draw's bound: **818 calls**. That is a ceiling that should never bind; it
  is the stop condition.
* **Expected calls:** about 1 to 2 per due draw (a decision, and one more
  when the model inspects); none at the A/A draws, where the turn is never
  due. Roughly 300 to 450 in all.
* **Per-draw guard:** 6 calls.
* **Time limit:** 4 hours on the wall clock. A request starts only if its
  whole 300 s timeout fits. V0.12b's 420 draws made 524 calls in 3,633 s on
  the same server, so a run of this size should take well under the
  limit.

**Why this budget (V0.10–V0.12 repeatability evidence):**
* **V0.10a:** of 25 prompts sent 11 times one at a time, the decision
  changed for 3. Across all 15 requests of each prompt (every condition,
  concurrent ones included), the decision had one value for 8 of 10 Gemma
  prompts and 9 of 15 Qwen prompts.
* **V0.12b Stage B:** at every Gemma point, all 10 draws of each arm made
  the same decision. At Qwen's valid points, 18 to 20 of 20 draws made the
  same decision; the most variation was at refused points.
* So at temperature 0, repeated draws of one unit mostly repeat one
  decision. Extra draws per unit mainly estimate a rate that is nearly 0 or
  1. They add little to what the experiment can show, which depends on the
  number of distinct contexts.
* **Gemma 3 draws** are enough to see whether a unit varies at all.
  **Qwen 5 draws** allow a little more resolution where Qwen has varied.
  **A/A contexts, 2 draws each:** they check that the arms are identical
  where no turn is shown, and need no precision.
* The V0.13c draft had 964 draws (4,820 calls). This one has 262 (818 at
  most): 27% of the draws and 17% of the call ceiling. More draws per unit
  would narrow per-unit intervals but would not add contexts, so they have
  no clear methodological benefit here.

**What this budget cannot do:**
* It cannot estimate small per-unit effects: with 3 or 5 draws per arm, a
  single unit's interval is wide.
* With 6 pairs, a pair-level summary has very little power. The pairs
  are reported one by one, with no hypothesis test.

## 5. Schedule

`dreamteam-v13d-order/1` (DRIVER.md §5): per model, rounds; within a
round, every unit of that model once, ordered by a SHA-256 of the
protocol's order rule, model, round, context and arm. The arms are
interleaved within each round. The schedule is a pure function of the
protocol and the contexts.

## 6. Analysis definitions (fixed before any request)

**Primary estimand (owner decision D9).** For each context and model, the
difference between the arms' distributions of the immediate final action:
`retained`, `selected`, `selected_no_action`, `revised`, `abandoned`,
`invalid_alternative`, `no_decision` (and `not_due` where no turn was
shown). Reported per category as treatment minus control, with a Newcombe
hybrid score interval (95%), and as the total variation distance between
the two distributions. The two arms' outcome sets differ: control cannot
select another alternative or no action, because it lists only the
proposal. That is part of the information the treatment adds, and is not
corrected for.

**The three questions the owner asked to keep apart:**
1. **Information exposure:** was the comparison presented (`presented`:
   the turn was due in the treatment arm)?
2. **Observable comparison:** did the model inspect an alternative other
   than its proposal (`inspected`), or cite verified differences
   (`compared`)? Neither: `no_observable_comparison`.
3. **Decision sensitivity:** did the final action differ between the arms,
   and, for the pairs, between the members?

None of these establishes a better financial or operational decision.

**Secondary, per sensitivity pair and model:** the departure rate (anything
but `retained`) in member b minus member a, per arm, with a Newcombe
interval; treatment minus control, with a MOVER interval. A context pair,
not one fact (SENSITIVITY_CASES.md).

**Also reported, per cell:** final valid, submitted, executed, partial,
calls and retries.

**Not done:** no hypothesis test, no pooling across models, no scoring of
decisions. The reason text is never read.

## 7. Stop conditions, failures and partial runs

* **Stops:** the call budget; the wall clock; 5 consecutive transport
  failures; an unhealthy server or a material server change before a
  round; any driver exception (DRIVER.md §3).
* **Failure handling inside a draw:** an unreadable reply, a transport
  failure or a second inspect uses the turn's one retry; when it is spent
  the result is `no_decision` and nothing is submitted. Validation-first
  keeps its own retries.
* **Partial run:** the interrupted draw is recorded and replayed, not
  classified. `run.json` says PARTIAL. The analysis reports only what was
  observed, with no pooled summary and no pair estimates.
* **Never resumed:** any continuation needs the owner's separate
  authorization and a documented amendment.

## 8. Replay requirements

Every recorded draw must replay without a model: the same contexts and
engine, the agent's own code, each recorded response only for its request
hash and only if its body still has its hash, every recorded response
used, and every deterministic field identical. Executed batches must give
the same digests when applied again.

## 9. Statistical limitations

* Draws repeat one unit at temperature 0 on one server: the intervals
  describe that server's repeat variation, not a population of contexts.
* 9 historical contexts and 12 constructed members: results do not
  generalize beyond them.
* The constructed contexts are scripted, with an empty memory, and are not
  historical behaviour.
* Every historical point was seen in an earlier experiment.
* The pairs do not isolate one fact.

## 10. Before a lock

* The human presentation review, complete and accepted (PENDING).
* The independent technical review's confirmed findings resolved
  (`docs/audits/v0.13d-review-response.md`).
* The owner's approval of this budget, or a change to it.
* The engine version for the run (it is still 0.13.0-dev.2; see
  LIMITATIONS.md) and the server's identity.
* Then a lock: every pinned file's SHA-256, the status `LOCKED — NOT RUN`,
  the schedule's hash. A lock does not authorize a run; the owner does.
