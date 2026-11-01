# V0.10b: live shared-prefix decision experiment

Status: **done** (0.10.0-dev.2), awaiting owner review. Declared at
ab1d8d1 before any request; run and analysed after. V0.10 is unreleased.
The independent evidence review is still PENDING.

The protocol is in [PLAN.md](PLAN.md), `protocol.json`, `points/` and
`schedule.json`. The method, `shared-prefix/v2`, is in [SPEC.md](SPEC.md).
The design is option A of
[the branch-point inventory](../../audits/v0.10b-branch-point-inventory.md),
approved by the owner.

At 8 recorded branch points, both arms hold fixed the recorded state, the
conversation and the initial staffing proposal. The model then gets one
more message:
* **control:** a neutral request for its final decision;
* **treatment:** the engine's exact preview, then the same request.

There are 20 draws per arm per point, 320 requests in all, one at a time
in a randomized interleaved order. The outcome is the immediate decision
(keep, revise or abandon). The simulator continues with a fixed no-action
suffix as a mechanics check.

No simulator mechanics, scoring, agent, prompt or frozen evidence rule
changes.

## 1. Baseline

At clean `main` de8ac79 (V0.10a), before any V0.10b change. `v0.9.0` is
still the tag object a5d7e32 → 7fa9557. Each aggregate is the SHA-256 of
the lines `<path> <sha256>\n`, in path order; a single file's own SHA-256
is given instead.

| Set | Files | SHA-256 |
|---|---|---|
| Simulator: Go sources without tests and the version package; scenarios; go.mod | 129 | `d92e7336e4b8947a93dfece448ea70e10ab98677d1207d941554cf97f344a878` |
| llm-agent 1 and 2: agents/llm/llm_agent.py | 1 | `5214e7bfb456075f38aca92a5c45a6db3141d4042fb1cd5822c8c1c65c286d87` |
| llm-agent 3: agents/llm/llm_agent3.py | 1 | `ef2c17a06749d7c287cda918f77ff0e518d79475048e87fd3404848c2fb0015c` |
| llm-agent 4: agents/llm/llm_agent4.py | 1 | `4cfc6a6906556b0cda50e95fa39263089d56e848adcef9bd04342e7f6aff998f` |
| Model configurations: docs/benchmark/v05c-dev/llm-*.json | 4 | `b0a1ff334b58ba0c8769fecc23dcacaa0a1472e16020c0b3e89e3b4a3bc9ecd2` |
| Historical recordings, V0.5a-V0.9c (*-decisions.jsonl) | 114 | `336024d8ba6dd490c99b72c7e626b7f0792ad10d312767dba2deacf45b8f2107` |
| Evidence contracts: frozen-v1.json, frozen-parameter-v1.json, the analyzers and their checks | 25 | `15c04973c677828a9fc23dede93f90ffc59864ae6bdecb1edd489194e6774c7b` |
| Review package and page (docs/benchmark/v08b-dev/review/, v09a-dev/review/) | 10 | `b3826aac8cc147af67894a08cb173b0f231afdbaa84c683041164de7b157ddd6` |
| Benchmark suites (docs/benchmark/suite/) | 4 | `63c007ccedc023a501690aa14ca76d400a18be68024cfa658ec5804bd7265c65` |
| Golden references, without MIGRATIONS.json | 12 | `d84f80718f3ab0463cc3768cf89b5ba09885b654e674a92a87c82c6998a2b25f` |
| All development evidence (docs/benchmark/v0*) | 479 | `e109a4de7e4de385c412695b6c9d7827fd32a9bc82400976d5adbe188b4b732f` |
| Session fixtures (testdata/sessions/*.json) | 110 | `ae29b02f0fcdbbfab78ef8accc6a89ec088694804155a7a66dc181c09389431f` |
| Prompt version | | `llm-agent-prompt/v2`, memory `llm-agent-memory/v1` |
| Built-in system prompt | | `0efc36e21b592abc4acb4c8b78ee7634b39755580a0f5ac348a5be1f8a80c4e1` |
| Preview treatment | | `llm-agent-preview/v1`, `f8b7d787b59c36373d4a18135c1503ccf64e3dd956c749be42fdfb0dbe5c54fa` |
| V0.10a repeatability evidence (docs/benchmark/v10a-dev/) | 29 | `615cded61342102233c26433df68efa1112d72d132e6f37ee3995ebe5475eb9e` |
| Repeatability and shared-prefix code (analysis/repeatability/*.py) | 6 | `ba5abcf3b59b8a70c073a75cc3d564e1767e7be9e2871851783cb434e7f65615` |

The rows are unchanged from the V0.10a results baseline.

## 2. Results in brief

These results are for these 8 points, these two models and this server.
They are descriptive.

* **Completion:** 320 of 320 requests, all HTTP 200, one at a time, from
  2026-11-01 19:11:21 to 19:24:59 UTC.
  * No stop. All 40 server checks found the same configuration.
  * 7 invalid replies, all in the denominator: 6 control, 1 treatment.
* **Refused-preview points, where the preview showed a refusal:**
  * Over 5 points and 100 draws per arm, the treatment arm changed the
    decision in 99 (the 100th was invalid).
  * The control arm resubmitted the refused proposal in 77.
  * At every refused-preview point, the 95% interval of the difference
    in at least one outcome excludes zero.
* **Accepted-preview points, where the preview showed nothing refused:**
  * At 2 of 3 points the arms were identical: kept, 20 of 20 each.
  * At the third (ss-qwen-s42002-p6) the treatment kept the proposal 20
    of 20 and the control 11 of 20. The control otherwise dropped two
    reading allocations, which cut the commitment from $44,687.30
    (credit-supported) to $303.75.
* **The comparison is with no information,** not with the agents' own
  loop. The control arm sees no validation, so a refused proposal stays
  as it is. The agents would validate it and retry with the refusal
  message. What the treatment adds is, at the refused points, information
  the agent would otherwise get after validating. Two of the five
  refused-preview points are malformed actions (`invalid_schema`), not
  funding.
* **Repeatability:**
  * Gemma gave one decision per arm at every point: one distinct action
    list in each of 8 arms. The raw text was identical within each arm,
    except one draw.
  * Qwen varied more under control (5–17 distinct raw texts per arm)
    than under treatment (2–4).
  * At the five V0.9c points the treatment reproduced V0.9c's recorded
    decision in 99 of 100 draws; the 100th was invalid.
* **Continuation:** 26 distinct final decisions were run on the engine
  with the no-action suffix. The 8 that were applied agreed with their
  validation, 8 of 8. Nothing else was applied: every other decision was
  refused or no action.
* **Replay and cross-platform:** the analysis contacts no model. It
  rebuilds every request from the points and checks its SHA-256, and
  replays every engine answer from the recordings. It is byte-identical
  on the Pi, horo and the CI platforms (§9).

## 3. Selected branch points

See [PLAN.md](PLAN.md) §2 for the fingerprints. In short:

| Point | Model | Preview | What the preview showed |
|---|---|---|---|
| ss-gemma-s42001-p14 | Gemma | accepted | a teleradiology contract, accepted |
| fl-gemma-s41901-p26 | Gemma | refused | a teleradiology contract with a malformed field (`invalid_schema`) |
| fl-gemma-s41901-p27 | Gemma | refused | a teleradiology contract the practice cannot fund (`insufficient_funds`) |
| fl-gemma-s41901-p28 | Gemma | refused | a teleradiology contract missing its windows (`invalid_schema`) |
| ss-qwen-s42001-p1 | Qwen | accepted | a teleradiology contract, accepted |
| ss-qwen-s42002-p6 | Qwen | accepted | two reading allocations, accepted (with the proposal's two shift extensions, a credit-supported commitment of $44,687.30) |
| ss-qwen-s42002-p1 | Qwen | refused | a teleradiology contract with a malformed field (`invalid_schema`) |
| ss-qwen-s42002-p26 | Qwen | refused | roster changes with too little notice (`notice_too_short`) |

## 4. Control and treatment construction

* Both arms send the recorded proposal request unchanged, with the
  proposal as the model's reply. The control and treatment requests share
  every message but the last, and every setting, byte for byte.
* **Control:** 305 characters (SPEC §2).
* **Treatment:** llm-agent 4's review message, 809 to 5,370 characters.
  The two share the final paragraph.
* The difference is therefore the preview and its introduction, about
  500 to 5,000 characters more. No control equalizes the length. The
  treatment's effect cannot be separated from the presence of more text,
  only from the existence of a second turn and the final request.

## 5. Decision distributions by point and arm

From [decision-tables.md](decision-tables.md). Counts of 20, with Wilson
95% intervals there. Final: the engine's validation of the submitted
actions.

| Point | Preview | Arm | Kept | Revised | Abandoned | Invalid | Final accepted / refused / no action |
|---|---|---|---|---|---|---|---|
| ss-gemma-s42001-p14 | accepted | control | 20 | 0 | 0 | 0 | 20 / 0 / 0 |
| | | treatment | 20 | 0 | 0 | 0 | 20 / 0 / 0 |
| fl-gemma-s41901-p27 | refused | control | 20 | 0 | 0 | 0 | 0 / 20 / 0 |
| | | treatment | 0 | 0 | 20 | 0 | 0 / 0 / 20 |
| fl-gemma-s41901-p28 | refused | control | 20 | 0 | 0 | 0 | 0 / 20 / 0 |
| | | treatment | 0 | 20 | 0 | 0 | 20 / 0 / 0 |
| fl-gemma-s41901-p26 | refused | control | 20 | 0 | 0 | 0 | 0 / 20 / 0 |
| | | treatment | 0 | 20 | 0 | 0 | 0 / 20 / 0 |
| ss-qwen-s42002-p6 | accepted | control | 11 | 8 | 0 | 1 | 19 / 0 / 0 |
| | | treatment | 20 | 0 | 0 | 0 | 20 / 0 / 0 |
| ss-qwen-s42001-p1 | accepted | control | 20 | 0 | 0 | 0 | 20 / 0 / 0 |
| | | treatment | 20 | 0 | 0 | 0 | 20 / 0 / 0 |
| ss-qwen-s42002-p26 | refused | control | 1 | 7 | 7 | 5 | 0 / 8 / 7 |
| | | treatment | 0 | 0 | 20 | 0 | 0 / 0 / 20 |
| ss-qwen-s42002-p1 | refused | control | 16 | 4 | 0 | 0 | 0 / 20 / 0 |
| | | treatment | 0 | 19 | 0 | 1 | 19 / 0 / 0 |
| **All** | | control | 128 | 19 | 7 | 6 | 59 / 88 / 7 |
| | | treatment | 60 | 59 | 40 | 1 | 99 / 20 / 40 |

The totals add points with different states and are shown only for the
accounting; see §7.

**What the revisions were** (actions matched by type and employee, role or
op):
* fl-gemma-p28, treatment 20/20: the contract's `windows` field added.
  The contract was then accepted.
* fl-gemma-p26, treatment 20/20: the contract replaced by another. It was
  still refused.
* ss-qwen-s42002-p1, treatment 19/20: the contract's `windows` field
  changed. It was then accepted.
* ss-qwen-s42002-p6, control 8/20: both reading allocations dropped (4
  also added a worklist policy).
* ss-qwen-s42002-p26, control 7/20: shift extensions (6) or reading
  allocations (1) added to the refused rosters. All were still refused.
* ss-qwen-s42002-p1, control 4/20: a shift extension added or removed.
  All were still refused.

**Invalid replies:**
* control: 5 at ss-qwen-s42002-p26 (3 `invalid_decision`, 2
  `invalid_json`, one of which stopped at the 4,096-token limit) and 1 at
  ss-qwen-s42002-p6 (`invalid_decision`);
* treatment: 1 at ss-qwen-s42002-p1 (`invalid_decision`).

## 6. Financial and operational exposure

Financial evidence v1 (unchanged) of each distinct final decision at the
recorded state is in [decision-tables.md](decision-tables.md). A refused
decision is not submitted, so it commits nothing.
* **Credit-supported commitments submitted:** control 11 draws, treatment
  40.
  * ss-qwen-s42002-p6: the proposal's $44,687.30, kept by the treatment
    20/20 and the control 11/20.
  * fl-gemma-p28: the treatment's corrected contract, $1,462.90, 20/20.
    The control's refused proposal committed nothing.
* **Commitment changes against the proposal:**
  * ss-qwen-s42002-p6 control: −$44,383.55 in the 8 revised draws;
  * fl-gemma-p28 treatment: +$1,462.90;
  * ss-qwen-s42002-p1 treatment: +$1,797.30, because the refused proposal
    committed nothing.
* **Reversibility** of executed actions, per the simulator's rules:
  * teleradiology starts are reversible with a retained obligation;
  * shift extensions are not reversible;
  * reading allocations are withdrawable.
* **Direct operational exposure:**
  * teleradiology coverage (ss-gemma-p14, fl-gemma-p28, ss-qwen-s42001-p1,
    ss-qwen-s42002-p1);
  * scheduled duty (the shift extensions);
  * reporting allocation (ss-qwen-s42002-p6).
* No revised decision is called better or worse. Accepted is not good,
  and a larger commitment is not bad.

## 7. Statistical interpretation

* **Per point.** Each point is its own context. The differences where the
  95% interval of "kept" excludes zero:
  * all five refused-preview points: −100 pp at each Gemma point; −80 pp
    at ss-qwen-s42002-p1; −5 pp in kept but +65 pp in abandoned at
    ss-qwen-s42002-p26;
  * one accepted-preview point: +45 pp at ss-qwen-s42002-p6.

  Two points show no difference at all.
* **Blocked.** Both arms of a point are in each of its 20 rounds. The
  outcomes differed in 20/20 rounds at each refused Gemma point, 16/20
  and 13/20 at the refused Qwen points, 9/20 at ss-qwen-s42002-p6, and
  0/20 at the other two. The two draws of a round are not a natural pair:
  they are separate requests.
* **Pooled,** after the per-point results: unweighted means of the
  per-point differences in kept are −75 pp for Gemma, −10 pp for Qwen and
  −42.5 pp overall. Kept was lower under treatment at 5 of 8 points and
  higher at 1. There is no interval over 8 clustered points.
* **What the comparison supports,** under SPEC §8's assumptions: at these
  points, with this state, conversation and proposal, showing the preview
  changed the distribution of the immediate decision, strongly where the
  preview showed a refusal.

  The assumptions:
  * the arms differ only by the last message (checked);
  * assignment by the schedule (holds);
  * exchangeable server noise between arms, and no interference through
    the cache (both unchecked).

  The Gemma differences are deterministic in practice, the same in all
  20 draws. Their interval reflects 20 identical repetitions, not
  sampling.
* **What it does not support:**
  * an effect on other states, prompts, models or practices;
  * an effect of the preview's content apart from its length and wording;
  * a comparison with the agents' validate-and-retry loop, which this
    control does not have;
  * any long-run effect.

## 8. Inference repeatability observations

Live metadata ([inference.json](inference.json)), apart from the analysis:
* Every request reported cached prompt tokens (320/320).
* Median latency per arm was 1.0–1.6 s for Gemma and 1.7–3.2 s for
  Qwen, except Qwen's control arm at ss-qwen-s42002-p26 (median 19.7 s,
  up to 33.6 s). One of its replies hit the 4,096-token limit (`length`).
* **Gemma:** raw-identical within every arm except fl-gemma-p27
  treatment (2 texts, one action list). Gemma was more repeatable here
  than in V0.10a's sample, at different points.
* **Qwen:** distinct raw texts per arm were 5, 9, 16 and 17 under control
  and 2–4 under treatment. Distinct action lists were 1–5 under control
  and 1–2 under treatment.
* **Treatment draws at the V0.9c points against V0.9c's recorded review
  reply** (exploratory, from the committed recordings and
  `decisions.json`): the same actions in 20, 20, 20, 20 and 19 draws (the
  20th of p1 was invalid). The same raw text in 20 (Gemma p14), 12, 0, 2
  and 19 draws.

## 9. Replay and cross-platform evidence

* **The live run** (`live/`): `requests.jsonl` (SHA-256 `e3266346…`)
  holds every request's SHA-256, status, latency and full response body,
  plus the server state before, at each round and after. The endpoint is
  not recorded.
* **The analysis** (`decision_experiment.py analyze`):
  * reads the recorded responses, the points and the engine, never a
    model;
  * checks the order against the schedule;
  * rebuilds each request from the points and checks its SHA-256;
  * replays each recording's transcript, checking every recorded answer.
* **Outputs:** `decisions.json` (`ab58de06…`), `decision-tables.md`
  (`58264bb7…`), `inference.json` (`d8e591c5…`).
* **Identity:** made on the Pi, regenerated twice on horo (linux/amd64)
  and on every CI platform. All byte-identical (§10).
* **The points:** checked again from the recordings and the engine on
  the Pi, on horo and in CI.
* **Historical recordings:** unchanged (the baseline rows in §1). No
  earlier live evaluation was rerun.

## 10. Quality gates

| Gate | Where | Result |
|---|---|---|
| Focused tests: `analysis/repeatability` (25 new) | Pi, horo and CI | 57 tests, OK |
| Python tests: `agents/llm`, `analysis/evidence` | CI | run by CI on this commit (unchanged code) |
| Go suite (`go test ./...`), golden references | CI, three platforms | run by CI on this commit; the new `v0101` session tests also run on the Pi |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64 | exit 0; 10 packages ok, no failure |
| `go vet ./...` (default, calibration, acceptance tags), `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64) | horo | all ok |
| Historical replays (V0.8c, V0.9c) | horo and CI | 104 checks, 0 failed, each of 4 replays |
| Benchmark verify and reproduce | CI | run by CI on this commit |
| Branch points checked again (selection, previews, requests) | Pi, horo and CI | 8 contexts verified |
| Analysis regenerated and compared | Pi, horo (twice) and CI | byte-identical |
| Cross-platform CI (linux/amd64, linux/arm64, darwin/arm64) | CI | run by CI on this commit; declaration commit ab1d8d1 green (run 41340455061) |

Not run: no clinical suite, no historical live evaluation rerun, and no
local repeat of the gates CI runs (Go suite, golden references, verify
and reproduce).

## 11. Known limitations

* **Selection:**
  * the design is unbalanced: Gemma has one accepted-preview context;
  * Gemma's three refused points are consecutive points of one run with
    the same kind of action;
  * the 8 points are a declared selection, not a sample.
* **Control:**
  * neutral, not length-matched;
  * no validation feedback, so it is not llm-agent 3's loop.
* **Replies:** one reply, no retry; a refused decision is not submitted.
* **Server:**
  * the shared server's cache and load are observed, not controlled;
  * Gemma and Qwen ran in separate blocks.
* **Statistics:**
  * 20 draws per arm cannot estimate rare outcomes;
  * Gemma's identical repetitions make its intervals a statement about
    20 identical answers.
* **Continuation:**
  * the no-action suffix is a mechanics check;
  * long-run financial outcomes are not attributed to the preview.
* **Preview:** staffing only, as in llm-agent 4.

## 12. Recommended V0.10c scope

* **Primary:** a control with the agents' own validation feedback. The
  control arm validates the proposal and, if refused, shows llm-agent 3's
  refusal message before the final reply. This separates the preview
  from error feedback, which the refused-point results here cannot do.
* **Arms:**
  * add a length-matched neutral control, or report the preview's length
    as a covariate;
  * keep sequential, interleaved execution.
* **Points:** prospective branch points with financial stakes, for
  example credit-supported commitments like ss-qwen-s42002-p6. That
  needs new recordings, since Gemma has one accepted-preview context in
  the existing ones. Declare the selection rule and the point count
  before generating them.
* **Statistics:**
  * fix the number of draws per arm from V0.10b's variation: Qwen's
    control arms needed most;
  * report per point and keep the fixed suffix as the primary
    continuation.
* **Deferred:** a live multi-day continuation, until the decision-level
  comparison is settled.
