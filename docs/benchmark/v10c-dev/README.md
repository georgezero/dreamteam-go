# V0.10c: validation feedback against preview information

Status: **done** (0.10.0-dev.3), awaiting owner review. Declared at
679483e before any request (CI 41363727129); run and analysed after.
V0.10 is unreleased. The independent evidence review is still PENDING.

The protocol is in [PLAN.md](PLAN.md), `protocol.json`, `points/` and
`schedule.json`. The method, `shared-prefix/v3`, is in [SPEC.md](SPEC.md),
and what each engine answer contains is in
[API_INVENTORY.md](API_INVENTORY.md).

At 12 recorded branch points (Gemma 4, Qwen 8), every arm holds fixed the
recorded state, conversation and initial staffing proposal. Then one
message:
* **A:** a neutral request for the final decision;
* **B:** the engine's exact validation of the proposal's staffing
  actions;
* **C:** the engine's exact preview of the same actions.

The primary comparison is C − B. Draws per arm per point: Gemma 10,
Qwen 20; 600 requests, one at a time, in a randomized interleaved order.
The simulator continues only with the fixed no-action suffix.

**The inventory's main finding, before any request:** the preview's
`results` are the validation's, in all 56 recorded staffing proposals. A
refused proposal's preview adds only `asOf` and, for an
`insufficient_funds` refusal, the funding test. A valid proposal's
preview adds the funding test, money lines, item details, slots and
displacements.

No simulator mechanics, accounting, action schema, scoring, agent,
prompt or historical evidence changes.

## 1. Baseline

At clean `main` d3da060 (V0.10b), before any V0.10c change. `v0.9.0` is
still the tag object a5d7e32 → 7fa9557. Each aggregate is the SHA-256 of
the lines `<path> <sha256>\n`, in path order; a single file's own SHA-256
is given instead.

| Set | Files | SHA-256 |
|---|---|---|
| Simulator: Go sources without tests and the version package; scenarios; go.mod | 129 | `75c7eb6018c077710a37083cebfa67c6b17f07c8fa9342979df2eceff5e30935` |
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
| Session fixtures (testdata/sessions/*.json) | 112 | `572d1dc3acf2d56cc534cb79171ef31726faf76e7f455e8b218da55009da0a8a` |
| Prompt version | | `llm-agent-prompt/v2`, memory `llm-agent-memory/v1` |
| Built-in system prompt | | `0efc36e21b592abc4acb4c8b78ee7634b39755580a0f5ac348a5be1f8a80c4e1` |
| Preview treatment | | `llm-agent-preview/v1`, `f8b7d787b59c36373d4a18135c1503ccf64e3dd956c749be42fdfb0dbe5c54fa` |
| V0.10a repeatability evidence (docs/benchmark/v10a-dev/) | 29 | `615cded61342102233c26433df68efa1112d72d132e6f37ee3995ebe5475eb9e` |
| Repeatability and shared-prefix code (analysis/repeatability/*.py) | 8 | `3831be3ad33f159f4a48aaec38b8bebb6906fd615765c4969df3564906a80935` |
| V0.10b decision experiment evidence (docs/benchmark/v10b-dev/) | 14 | `b95e433fffecb9b22585c776179a3905f5deaa08b460715ad29bebe08ee2306b` |

Against the V0.10b baseline (de8ac79), only V0.10b's own additions
changed:
* the simulator aggregate, through the 0.10.0-dev.1 registration in
  `internal/session`;
* the session fixtures (`mini-seed52-v0101`);
* the repeatability code.

shared-prefix/v1 and v2, the V0.10a and V0.10b evidence, llm-agent 1–4,
the frozen evidence contracts, the historical sessions and the golden
references are preserved. V0.10c adds a new module and imports v2's.

## 2. Results in brief

These results are for these 12 points, these two models and this server.
They are descriptive.

* **Completion:** 600 of 600 requests, all HTTP 200, finish reason
  `stop`, one at a time, from 2026-11-01 23:46 to 2026-11-02 00:26 UTC.
  * No stop. All 30 server checks found the same configuration.
  * 2 invalid replies, both neutral.
* **Refused proposals (7 points): validation did what the preview did.**
  * The kept, revised and abandoned distributions of C and B are equal
    at 5 points and within 10 pp at the other 2.
  * The final decision was more often valid under the preview at 2 Qwen
    points: fl-qwen-s41901-p9, insufficient funds, 20 of 20 against 2 of
    20; ss-qwen-s42001-p23, a past date, 16 of 20 against 5 of 20.
  * At the 5 other refused points, the final status was the same.
* **Accepted proposals (5 points): validation and preview pulled in
  opposite directions for Qwen.**
  * With validation, Qwen revised the proposal in 20 of 20 draws at all
    four of its accepted points.
  * With the preview it kept it in 20 of 20 at two (ss-qwen-pv-s42002-p6,
    ss-qwen-s42002-p7) and 17 of 20 at a third (ss-qwen-s42001-p20).
  * At the fourth (fp-qwen-s41901-p26), both arms dropped the same no-op
    action, which the engine had answered `unchanged`.
  * Gemma kept its one accepted proposal under every arm.
* **Money:**
  * Under validation, Qwen dropped ss-qwen-pv-s42002-p6's two reading
    allocations, 20 of 20. That took the commitment from $44,687.30
    (credit-supported) to $303.75. The preview kept $44,687.30, 20 of 20.
  * At ss-qwen-s42002-p7, validation dropped the teleradiology contract
    ($3,228.33 to $135.00); the preview kept it.
  * Neither direction is called better.
* **Against no feedback (A):**
  * The kept/revised/abandoned/invalid distribution differed from A's at
    11 of 12 points under validation and 10 of 12 under the preview.
  * Overall, A kept the proposal in 152 of 200 draws, B in 10 and C
    in 67.
* **Repeatability:** Gemma gave one decision per arm at every point
  (raw text identical except one arm). Qwen varied most under the neutral
  arm.
* **Continuation:** 53 distinct final decisions were run with the
  no-action suffix. The 21 applied ones agreed with their validation, 21
  of 21.
* **Replay and cross-platform:** the analysis contacts no model. It
  checks every request and response hash, and asks the engine for the
  validate and preview answers again. It is byte-identical on the Pi,
  horo and CI (§10).

## 3. What each arm received

From [API_INVENTORY.md](API_INVENTORY.md):
* **B and C carry the same `results`** (every action's code and message)
  in all 56 recorded proposals.
* **At a refused proposal** the preview adds only `asOf`, plus the
  funding test for an `insufficient_funds` refusal. That refusal's
  message already states the commitment and headroom.
* **At a valid proposal** the preview adds:
  * the funding test, including `funded` and the headroom;
  * guaranteed and forecast money lines;
  * item details;
  * slots and displacements.
* **Message lengths at the selected points** (A / B / C):
  * refused: 305 / 777–2,055 / 809–2,300;
  * accepted: 305 / 1,087–7,886 / 2,434–100,937.

## 4. Live experiment completion

| Model | Points | Arms | Draws per arm | Requests | HTTP 200 | Invalid replies |
|---|---|---|---|---|---|---|
| Gemma | 4 | 3 | 10 | 120 | 120 | 0 |
| Qwen | 8 | 3 | 20 | 480 | 480 | 2 (neutral) |

Every request reported cached prompt tokens except 3.

## 5. Three-arm decision distributions

Counts of draws: kept / revised / abandoned / invalid; then the final
decision accepted / refused / no action. Wilson intervals are in
[decision-tables.md](decision-tables.md).

| Point | Proposal | A neutral | B validation | C preview | Final A | Final B | Final C |
|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | accepted | 10/0/0/0 | 10/0/0/0 | 10/0/0/0 | 10/0/0 | 10/0/0 | 10/0/0 |
| fl-gemma-s41901-p26 | refused | 10/0/0/0 | 0/10/0/0 | 0/10/0/0 | 0/10/0 | 0/10/0 | 0/10/0 |
| fl-gemma-s41901-p28 | refused | 10/0/0/0 | 0/10/0/0 | 0/10/0/0 | 0/10/0 | 10/0/0 | 10/0/0 |
| fl-gemma-s41901-p27 | refused | 10/0/0/0 | 0/0/10/0 | 0/0/10/0 | 0/10/0 | 0/0/10 | 0/0/10 |
| ss-qwen-pv-s42002-p6 | accepted | 12/7/0/1 | 0/20/0/0 | 20/0/0/0 | 18/1/0 | 20/0/0 | 20/0/0 |
| ss-qwen-s42002-p7 | accepted | 20/0/0/0 | 0/20/0/0 | 20/0/0/0 | 20/0/0 | 20/0/0 | 20/0/0 |
| fp-qwen-s41901-p26 | accepted | 20/0/0/0 | 0/20/0/0 | 0/20/0/0 | 20/0/0 | 20/0/0 | 20/0/0 |
| ss-qwen-s42001-p20 | accepted | 20/0/0/0 | 0/20/0/0 | 17/3/0/0 | 20/0/0 | 20/0/0 | 20/0/0 |
| fl-qwen-s41901-p9 | refused | 0/19/0/1 | 0/18/2/0 | 0/20/0/0 | 0/19/0 | 2/16/2 | 20/0/0 |
| fl-qwen-s41901-p11 | refused | 20/0/0/0 | 0/0/20/0 | 0/0/20/0 | 0/20/0 | 0/0/20 | 0/0/20 |
| ss-qwen-s42001-p23 | refused | 0/20/0/0 | 0/17/3/0 | 0/19/1/0 | 20/0/0 | 5/12/3 | 16/3/1 |
| fp-qwen-s42001-p27 | refused | 20/0/0/0 | 0/20/0/0 | 0/20/0/0 | 0/20/0 | 0/20/0 | 0/20/0 |

A revision changes the actions. A revised refused proposal can still be
refused.

**What the revisions were** (actions matched by type and employee, role or
op):
* **Gemma, refused points:** B and C made the same kind of revision.
  * p26: replaced the malformed contract (still refused).
  * p28: added the contract's windows; B chose $3,023.32 of commitment
    and C $1,462.90, both accepted and credit-supported.
  * p27: abandoned, with no action.
* **ss-qwen-pv-s42002-p6:** B dropped both reading allocations, 20/20. A
  did so in 7/20 (some also adding a policy or extension).
* **ss-qwen-s42002-p7:** B dropped the teleradiology contract, 20/20.
* **fp-qwen-s41901-p26:** B and C dropped the second allocation, 20/20
  each. The engine had answered it `unchanged` (already 50%).
* **ss-qwen-s42001-p20:** B changed the six rosters, 20/20; C did so in
  3/20. The engine's messages said contracted time not rostered is still
  paid.
* **fl-qwen-s41901-p9:**
  * B dropped the hire (16/20), still refused for insufficient funds;
  * C dropped the hire and lowered the allocation's site share (20/20,
    in four variants), accepted;
  * A changed the hire's dates or rate, still refused.
* **ss-qwen-s42001-p23:**
  * B moved the contract's effective date (12/20), still refused;
  * C dropped the contract (16/20), accepted;
  * A dropped the contract and added shift extensions.
* **fl-qwen-s41901-p11:** B and C abandoned (20/20 each).
* **fp-qwen-s42001-p27:** B and C changed the contract's end date
  (20/20 each), still refused.

## 6. Preview against validation (primary, C − B)

Per point, from [decision-tables.md](decision-tables.md), with Newcombe
95% intervals:

| Point | Proposal | Kept | Revised | Final accepted | Same action content | Discordant rounds |
|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | accepted | 0 | 0 | 0 | yes | 0/10 |
| fl-gemma-s41901-p26 | refused | 0 | 0 | 0 | yes | 0/10 |
| fl-gemma-s41901-p28 | refused | 0 | 0 | 0 | no (different windows) | 0/10 |
| fl-gemma-s41901-p27 | refused | 0 | 0 | 0 | yes | 0/10 |
| ss-qwen-pv-s42002-p6 | accepted | **+100 pp** [77, 100] | −100 pp | 0 | no | 20/20 |
| ss-qwen-s42002-p7 | accepted | **+100 pp** [77, 100] | −100 pp | 0 | no | 20/20 |
| fp-qwen-s41901-p26 | accepted | 0 | 0 | 0 | yes | 0/20 |
| ss-qwen-s42001-p20 | accepted | **+85 pp** [58, 95] | −85 pp | 0 | no | 17/20 |
| fl-qwen-s41901-p9 | refused | 0 | +10 pp | **+90 pp** [64, 97] | no | 2/20 |
| fl-qwen-s41901-p11 | refused | 0 | 0 | 0 | yes | 0/20 |
| ss-qwen-s42001-p23 | refused | 0 | +10 pp | **+55 pp** [24, 73] | no | 4/20 |
| fp-qwen-s42001-p27 | refused | 0 | 0 | 0 | no (different end dates) | 0/20 |

* At 7 of 12 points the preview and the validation answer gave the same
  outcome distribution and the same final status.
* At 5 they differed:
  * 3 accepted Qwen points, where the preview kept what validation
    revised;
  * 2 refused Qwen points, where the preview's revisions were valid more
    often.

## 7. Financial commitments

Financial evidence v1 (unchanged), per distinct final decision, is in
[decision-tables.md](decision-tables.md). The applied commitment of the
final decision, in draws:

| Point | Proposal's stake | A | B | C |
|---|---|---|---|---|
| ss-qwen-pv-s42002-p6 | $44,687.30, credit-supported | $44,687.30 ×12; $303.75 ×6; $0 ×1 (refused); invalid ×1 | $303.75 ×20 | $44,687.30 ×20 |
| ss-qwen-s42002-p7 | $3,228.33 | $3,228.33 ×20 | $135.00 ×20 | $3,228.33 ×20 |
| fl-gemma-s41901-p28 | refused | $0 ×10 (refused) | $3,023.32 ×10, credit-supported | $1,462.90 ×10, credit-supported |
| fl-qwen-s41901-p9 | refused ($36,978.91 tested) | $0 (refused ×19, invalid ×1) | $0 (refused ×16, no action ×2, valid at $0 ×2) | $288.45 ×4 (credit-supported); $0 ×16 (valid, cost-neutral) |
| ss-qwen-s42001-p23 | refused | $547.50 ×12; $468.75 ×8 | $0 ×15; $468.75 ×5 | $468.75 ×16; $0 ×4 |

At the other points every arm's commitment was the proposal's: ss-gemma
p14 $3,069.98; fp-qwen-s41901-p26 $9,863.02; ss-qwen-s42001-p20
$1,475.80. It was $0 at Gemma p26 and p27, fl-qwen-s41901-p11 and
fp-qwen-s42001-p27.

**Credit-supported commitments submitted:** A 12, B 10, C 34.

**Kept apart per draw** (counts of draws in [decision-tables.md](decision-tables.md)):

| | A | B | C |
|---|---|---|---|
| More information received | none | the validate answer | the preview answer |
| Proposal changed | 46 + 2 invalid | 190 | 133 |
| Became valid (a refused proposal's final decision accepted) | 20 | 17 | 46 |
| Commitment changed | 27 | 55 | 30 |
| Direct exposure changed | 27 | 57 | 46 |

**Cash-only headroom at the points** (headroom without undrawn credit):
* negative at all five refused low-liquidity points, from −$52,429.59
  to −$33,275.37;
* positive elsewhere, from $3,511.19 at ss-qwen-s42001-p20 to $53,953.60.

A smaller commitment is not called better, and a valid one not wise.

## 8. Operational exposure

* **Retained obligations** (teleradiology starts):
  * kept by every arm at ss-gemma p14 and ss-qwen-s42001-p20;
  * dropped by B at ss-qwen-s42002-p7;
  * added by B and C at fl-gemma p28.
* **Displaced bookings:**
  * ss-qwen-s42001-p20's rosters displace 405 bookings in every arm and
    every variant: the revisions under B and C kept the displacement;
  * C's variants at fl-qwen-s41901-p9 that terminate a technologist
    displace 8 (6 of 20 draws).
* **Not reversible** (shift extensions): kept or added under every arm.
  B and C did not remove them at ss-qwen-pv-s42002-p6 or
  ss-qwen-s42002-p7.
* **Direct exposure changed** with the commitment at the points above,
  and also at fl-qwen-s41901-p9 and ss-qwen-s42001-p23 (the revisions
  there).

## 9. Statistical interpretation

The four effects, kept separate:
* **Validation-feedback effect** (B − A): large and consistent where
  the proposal was refused. Under A the model resubmitted the refused
  proposal at 5 of 7 refused points, every draw, and revised it at the
  other 2. Under B it revised or abandoned it at all 7, in every draw.
  * It was also large where the proposal was valid: B made Qwen revise
    at 4 of 4 accepted points, where A kept.
  * Gemma's accepted point was unaffected.
* **Incremental preview effect** (C − B):
  * none at 7 of 12 points;
  * where the proposal was valid, the preview kept what validation
    revised (3 Qwen points, +85 to +100 pp kept);
  * where it was refused, the preview's revisions were more often valid
    at 2 Qwen points (+90 and +55 pp).

  The preview's extra content is the funding test, money lines and
  details at a valid proposal, and `asOf` (today's date) or the
  structured funding test at a refused one. That content and the
  message's length differ together and cannot be separated.
* **Inference variation:**
  * Gemma: one decision per arm at every point.
  * Qwen: distinct raw texts per arm 1–15 (most under A: 15 at p6, 13 at
    p9, 11 at p23); distinct action lists 1–6.
  * Several C − B differences are 20 of 20 against 0 of 20, larger than
    the within-arm variation.
* **Branch-point heterogeneity:** the direction depends on the point.
  Pooling would hide this: across 12 points the mean C − B in kept is
  +23.8 pp, made of +100, +100 and +85 at three points and 0 at nine.

**Identified,** under SPEC §6's assumptions (one designed difference,
checked; assignment by the schedule; exchangeable noise and no cache
interference, not checked): at each point, the difference in the
immediate-decision distribution between the arms' messages, for that
state, conversation and proposal.

**Not identified or uncontrolled:**
* which part of the preview (the funding test, the money lines, `asOf`,
  the length) caused a difference;
* why validation of a valid proposal led to revisions;
* the shared server's cache and load;
* any population of states, prompts, models or practices;
* any long-run outcome.

The repetitions are of one request each, not independent cases.

## 10. Replay and cross-platform evidence

* **The live run** (`live/`): `requests.jsonl` holds every request's and
  response's SHA-256, status, latency and full body, plus the server's
  state before, at each round and after. The endpoint is not recorded.
* **The analysis** (`feedback_experiment.py analyze`) contacts no model.
  It checks:
  * the order against the schedule;
  * every request rebuilt from the points;
  * every response against its hash;
  * every content against its response;
  * the validate and preview answers, asked again of a fresh engine
    session at each point.

  It then replays each distinct decision with the no-action suffix.
* **Outputs:** `decisions.json`, `decision-tables.md`, `inference.json`.
* **Identity:** made on the Pi, regenerated twice on horo (linux/amd64)
  and on every CI platform. All byte-identical (§11).
* **The inventory and the points:** made again from the recordings and
  the engine on horo and in CI, and equal.
* **Historical recordings, V0.10a and V0.10b evidence:** unchanged. No
  earlier live evaluation was rerun.

## 11. Quality gates

| Gate | Where | Result |
|---|---|---|
| Focused shared-prefix tests (`analysis/repeatability`; 17 new for v3) | Pi, horo and CI | 74 tests, OK |
| Agent tests (`agents/llm`), evidence tests (`analysis/evidence`) | horo and CI | 60 and 147 tests, OK |
| Go suite (`go test ./...`), golden references | CI, three platforms | run by CI on this commit; the new `v0102` session tests also on the Pi |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64 | exit 0; 10 packages ok, no failure |
| `go vet ./...` (default, calibration, acceptance tags), `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64) | horo | all ok |
| Historical replays (V0.8c, V0.9c) | horo and CI | 104 checks, 0 failed, each of 4 replays |
| Benchmark verify and reproduce | CI | run by CI on this commit |
| Inventory and branch points made again (selection, engine answers, requests) | horo and CI | 12 contexts verified |
| Analysis regenerated and compared | Pi, horo (twice) and CI | byte-identical |
| Cross-platform CI (linux/amd64, linux/arm64, darwin/arm64) | CI | run by CI on this commit; declaration commit 679483e green (run 41363727129) |

Not run: no clinical suite, no historical live evaluation rerun, and no
local repeat of the gates CI runs (Go suite, golden references, verify
and reproduce).

## 12. Known limitations

* **Selection:**
  * Gemma has one accepted-proposal context and no credit-supported one;
  * one credit-supported proposal exists in all the recordings;
  * the selection rule was written after V0.10b's results were known,
    though it reads no response.
* **B is not llm-agent 3's feedback.** B shows the whole validate answer
  for the staffing actions, accepted results included. llm-agent 3 shows
  only refused results' codes and messages after a refusal, and nothing
  for a valid proposal.
* **No length control.** C is up to 12.8 times as long as B at accepted
  points.
* **Replies:** one reply, no retry; a refused decision is not submitted.
* **Server and statistics:**
  * the shared server's cache and load are observed, not controlled;
  * draws are repetitions of one request;
  * Gemma's 10 draws per arm repeat one answer.
* **Scope:**
  * staffing actions only;
  * the no-action suffix is a mechanics check.

## 13. V0.10.0 release recommendation

**Recommend releasing V0.10.0** after owner review of V0.10c, as a
methodology and measurement release. Unchanged:
* the engine;
* the agents;
* the prompts;
* the scoring.

**What V0.10 delivers:**
* measured inference repeatability (V0.10a);
* a versioned shared-prefix method with live replay and cross-platform
  identity (v1 to v3);
* two prospectively declared decision experiments;
* the information inventory of validate and preview.

**What V0.10c says for agent design:**
* **Refused proposals:** ordinary validation feedback produced nearly all
  of the change the preview produced. The preview helped beyond it at 2
  Qwen points, by making revisions valid.
* **Valid proposals:** the preview mostly preserved them, including a
  large credit-supported commitment. The validation answer led Qwen to
  cut them.

So more information did not simply improve decisions; it moved them in
directions that depend on the information and the point. That is
evidence against building more elaborate preview-driven workflows on the
strength of V0.10b alone.

**Before release:**
* the owner's acceptance;
* the release baseline, notes and tag by the owner's decision;
* the independent evidence review stays PENDING.
