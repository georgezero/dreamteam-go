# V0.11b: controlled evaluation of the validation-first workflow

Status: **done** (0.11.0-dev.2), awaiting owner review. Declared at
562ed93 before any request (CI 41471597358); run and analysed after.
V0.11 is unreleased. The independent evidence review is still PENDING.

The protocol is in [PLAN.md](PLAN.md), `protocol.json`, `points/` and
`schedule.json`. The method, `shared-prefix/v4` (whole workflows as arms),
is in [SPEC.md](SPEC.md).

From the recorded state and initial staffing proposal at 14 branch points
(Gemma 4, Qwen 10), three decision workflows run, each in full:
* **A:** llm-agent 3;
* **B:** llm-agent 5 with the preview disabled;
* **C:** llm-agent 5.

The primary comparison is C − B and the secondary B − A. There are 720
draws (Gemma 10, Qwen 20 per arm and point), one model request at a time.
The simulator continues only with the fixed no-action suffix.

## 1. Baseline

At clean `main` e6bb386 (V0.11a; `v0.10.0` is still the tag object
8230a31 → 728fbf5). Each aggregate is the SHA-256 of the lines
`<path> <sha256>\n`, in path order.

| Set | Files | SHA-256 |
|---|---|---|
| Simulator: Go sources without tests, the version package and the session-registration files (internal/session/record*.go); scenarios; go.mod | 127 | `95c8643de0679346758ebb6cc704593c669c33a25355c59d34f9f0750789205b` |
| Session registration (internal/session/record.go, record_v1.go) | 2 | `32d13747624e29304e59bdeef1e1fc6076797956c073155a44d0387d9fd11397` |
| Historical recordings, V0.5a-V0.9c (*-decisions.jsonl) | 114 | `336024d8ba6dd490c99b72c7e626b7f0792ad10d312767dba2deacf45b8f2107` |
| llm-agent 1 to 4 (agents/llm/llm_agent.py, llm_agent3.py, llm_agent4.py) | 3 | `30bfec269265d207156b59738126ab8612027bfa9f7546116aba903c9a72e884` |
| Frozen evidence contracts (frozen-v1.json, frozen-parameter-v1.json and their checks) | 4 | `48566731ec58d6ba6bfca1dd1be1f482d4558f95bbdb16bceba33969f9a3a3ec` |
| Evidence analyzers (analysis/evidence/*.py) | 23 | `41bfbea46bc948979ebb3cb2b3e66318b8af0befa788ea8a0a49c7231e5ba6b1` |
| Shared-prefix v1-v3 and repeatability code (analysis/repeatability/*.py) | 10 | `1d00341bf0f5d00ad4abbb5fecd4e6e162ef18d695f41d51e23ba0344dff07d0` |
| V0.10a repeatability evidence (docs/benchmark/v10a-dev/) | 29 | `615cded61342102233c26433df68efa1112d72d132e6f37ee3995ebe5475eb9e` |
| V0.10b experiment (docs/benchmark/v10b-dev/) | 14 | `b95e433fffecb9b22585c776179a3905f5deaa08b460715ad29bebe08ee2306b` |
| V0.10c experiment (docs/benchmark/v10c-dev/) | 16 | `f3885a15dcf797ac1a78ebb2974eefc0dfddd517a4b74f98f0cd9724ed4b6d0b` |
| Benchmark suites (docs/benchmark/suite/) | 4 | `63c007ccedc023a501690aa14ca76d400a18be68024cfa658ec5804bd7265c65` |
| Golden references, without MIGRATIONS.json | 12 | `d84f80718f3ab0463cc3768cf89b5ba09885b654e674a92a87c82c6998a2b25f` |
| All earlier development evidence (docs/benchmark/v0*) | 479 | `e109a4de7e4de385c412695b6c9d7827fd32a9bc82400976d5adbe188b4b732f` |
| Session fixtures (testdata/sessions/*.json) | 118 | `56182c935663ef7534f38737656b9af10d88778c17173498ad65002defee4eb9` |
| Prompts: llm-agent-prompt/v2 system prompt; llm-agent-preview/v1 treatment | | system `0efc36e21b592abc4acb4c8b78ee7634b39755580a0f5ac348a5be1f8a80c4e1`, treatment `f8b7d787b59c36373d4a18135c1503ccf64e3dd956c749be42fdfb0dbe5c54fa` |
| Decision-record contracts | | `dreamteam.llm-decision-record/v3`, `dreamteam.llm-decision-record/v4` (llm-agent 1 and 2: `llm_agent.py`, unchanged) |
| llm-agent 5 (agents/llm/llm_agent5.py) | 1 | `a125129a7768f8614cb0714642bd8faf51f0cfcf3b42f85affd7cd729e995d38` |
| llm-agent 5 workflow and record | | `llm-agent-workflow/v1` `3ee7d66f4f10fd93e05a908725c792b16b76d01dc29095360dae9d67c0c09d0a`, `dreamteam.llm-decision-record/v5` |
| V0.11a evidence (docs/benchmark/v11a-dev/) | 3 | `69f5170bc14d82c4b5d149265979b626e3781e3460c09b531fcdcb23c7b035a7` |

**On the V0.11b tree,** every set above is unchanged except three:
* the session registration (0.11.0-dev.1 added, batch mode `v0.11_dev1`);
* the session fixtures (two `mini-seed52-v0111.*` files added);
* `analysis/repeatability`, with two files added
  (`workflow_experiment.py`, `test_workflow_experiment.py`). The earlier
  files there are unchanged.

## 2. Results in brief

These results are for these 14 points, these two models and this server.
They are descriptive.

* **Completion:** 720 of 720 draws, from 2026-11-02 15:26 to 16:04 UTC.
  * 572 model calls, all HTTP 200, finish reason `stop`.
  * No stop. All 30 server checks found the same configuration.
* **C − B (the selective preview):** no difference at 10 of 14 points.
  * **Valid proposals:** A and B submit without a model call. C showed
    the preview at all 5 Qwen and the 1 Gemma valid points; after it the
    model:
    * **kept** the proposal at 4 points, 20/20 or 10/10, including the
      $44,687.30 credit-supported allocation (ss-qwen-pv-s42002-p6);
    * **dropped the teleradiology contract** at ss-qwen-s42002-p7 in
      16/20 (revised) and abandoned the proposal in 4/20;
    * **dropped a no-op allocation** at fp-qwen-s41901-p26 in 20/20. The
      engine had answered it `unchanged`; the commitment did not change.
  * **Refused proposals:** C and B gave the same decisions at 5 of 7
    points and differed by 5 pp at 2.
* **B − A (validation-first feedback against llm-agent 3's retry loop):**
  no difference at the valid points, which both submit, or at Gemma's
  points. At 3 of 5 refused Qwen points, B made valid corrections where A
  did not:
  * fl-qwen-s41902-p27: 20/20 against 0/20;
  * fl-qwen-s41901-p9: 13/20 against 0/20;
  * fp-qwen-s42001-p24: 5/20 against 0/20.
* **Valid is not good.** At fl-qwen-s41902-p27 and fl-qwen-s41901-p9,
  B's and C's valid corrections terminate a technologist: not reversible,
  displacing 100 and 8 bookings. Under A those workflows ended refused or
  with no action.
* **The nudge test:**
  * At valid proposals the preview did not increase commitments: C's
    commitment was the proposal's or lower at every valid point.
  * It reinforced the credit-supported $44,687.30 proposal no more than
    B's no-review submission did: both submitted it in every draw.
  * Unlike V0.10c, the validation-first workflow shows a valid proposal
    nothing before submitting it. So B cannot cut it, as V0.10c's
    validation message did.
* **Replay:**
  * All 720 draws were replayed without a model: 572 model responses and
    2,299 engine requests, every trace and answer equal.
  * The analysis is byte-identical on the Pi, horo and CI (§10).

## 3. Branch points

14 points: [PLAN.md](PLAN.md) §2 gives fingerprints, categories and
prior exposure. Nine were in V0.10b or V0.10c, five are new.

## 4. Workflow fidelity

* **Tested against the real agents:** before the run,
  `test_workflow_experiment.py` checked on scripted engine and model
  that:
  * the A driver sends exactly llm-agent 3's messages and submits its
    batches;
  * B sends llm-agent 5's when no preview is shown;
  * C sends llm-agent 5's.

  The cases cover refusals, corrections, abandonment, repeated refusals,
  unreadable replies, timeouts, mixed batches, previews kept, revised or
  abandoned, and a failed preview.
* **Checked again in the run:** the replay re-ran every draw's workflow
  and found the same messages (by SHA-256) and the same engine requests.
* **Model calls** (all draws): A 130, B 130, C 312.
  * At a valid proposal, A and B call no model, and C calls it once
    (plus a retry when its revision is refused: 4 at ss-qwen-s42002-p7).
  * At a refused proposal, each arm calls it once (`maxRetries` 1), and C
    once more when a corrected proposal's preview is shown (68 times).

  The arms' inference exposure is therefore not equal. A and B's draws at
  a valid proposal are one deterministic result repeated.

## 5. Decision distributions

kept / revised / abandoned / refused / invalid, of 10 (Gemma) or 20
(Qwen). Then the final decision: accepted / no action / none submitted.
Intervals are in [decision-tables.md](decision-tables.md).

| Point | Proposal | A | B | C | Final A | Final B | Final C |
|---|---|---|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | valid | 10/0/0/0/0 | 10/0/0/0/0 | 10/0/0/0/0 | 10/0/0 | 10/0/0 | 10/0/0 |
| fl-gemma-s41901-p28 | refused | 0/10/0/0/0 | 0/10/0/0/0 | 0/10/0/0/0 | 10/0/0 | 10/0/0 | 10/0/0 |
| fl-gemma-s41901-p27 | refused | 0/0/10/0/0 | 0/0/10/0/0 | 0/0/10/0/0 | 0/10/0 | 0/10/0 | 0/10/0 |
| fl-gemma-s41901-p26 | refused | 0/0/0/10/0 | 0/0/0/10/0 | 0/0/0/10/0 | 0/0/10 | 0/0/10 | 0/0/10 |
| ss-qwen-pv-s42002-p6 | valid | 20/0/0/0/0 | 20/0/0/0/0 | 20/0/0/0/0 | 20/0/0 | 20/0/0 | 20/0/0 |
| ss-qwen-s42002-p7 | valid | 20/0/0/0/0 | 20/0/0/0/0 | 0/16/4/0/0 | 20/0/0 | 20/0/0 | 16/4/0 |
| fl-qwen-s41902-p24 | valid | 20/0/0/0/0 | 20/0/0/0/0 | 20/0/0/0/0 | 20/0/0 | 20/0/0 | 20/0/0 |
| fp-qwen-s41901-p26 | valid | 20/0/0/0/0 | 20/0/0/0/0 | 0/20/0/0/0 | 20/0/0 | 20/0/0 | 20/0/0 |
| fp-qwen-s41901-p25 | valid | 20/0/0/0/0 | 20/0/0/0/0 | 20/0/0/0/0 | 20/0/0 | 20/0/0 | 20/0/0 |
| fl-qwen-s41901-p9 | refused | 0/0/9/10/1 | 0/13/2/5/0 | 0/12/3/5/0 | 0/9/11 | 13/2/5 | 12/3/5 |
| fl-qwen-s41901-p11 | refused | 0/1/19/0/0 | 0/0/20/0/0 | 0/0/20/0/0 | 1/19/0 | 0/20/0 | 0/20/0 |
| fp-qwen-s42001-p7 | refused | 0/20/0/0/0 | 0/20/0/0/0 | 0/20/0/0/0 | 20/0/0 | 20/0/0 | 20/0/0 |
| fl-qwen-s41902-p27 | refused | 0/0/0/20/0 | 0/20/0/0/0 | 0/20/0/0/0 | 0/0/20 | 20/0/0 | 20/0/0 |
| fp-qwen-s42001-p24 | refused | 0/0/0/20/0 | 0/5/0/15/0 | 0/4/0/16/0 | 0/0/20 | 5/0/15 | 4/0/16 |
| **All** | | 110/31/38/60/1 | 110/68/32/30/0 | 70/102/37/31/0 | 141/38/61 | 178/32/30 | 172/37/31 |

The totals add different points and are shown only for the accounting.

**Gemma** gave one decision per arm at every point, and the same one
under A, B and C at all four.

## 6. Validation correction results (refused proposals)

Valid corrections (final accepted) under A / B / C:

| Point | Refusal | A | B | C | B − A | C − B |
|---|---|---|---|---|---|---|
| fl-gemma-s41901-p28 | invalid_schema | 10/10 | 10/10 | 10/10 | 0 | 0 |
| fl-gemma-s41901-p27 | insufficient_funds | 0 (abandoned 10) | 0 (abandoned 10) | 0 (abandoned 10) | 0 | 0 |
| fl-gemma-s41901-p26 | invalid_schema | 0 (refused 10) | 0 (refused 10) | 0 (refused 10) | 0 | 0 |
| fl-qwen-s41901-p9 | insufficient_funds | 0 | 13 | 12 | **+65 pp** [38, 82] | −5 pp |
| fl-qwen-s41901-p11 | insufficient_funds | 1 | 0 | 0 | −5 pp | 0 |
| fp-qwen-s42001-p7 | lead_too_short | 20 | 20 | 20 | 0 | 0 |
| fl-qwen-s41902-p27 | notice_too_short | 0 (refused 20) | 20 | 20 | **+100 pp** [77, 100] | 0 |
| fp-qwen-s42001-p24 | contract_overlap | 0 (refused 20) | 5 | 4 | **+25 pp** [4, 47] | −5 pp |

Abandonment (no action) under A / B / C: fl-qwen-s41901-p9 9 / 2 / 3,
fl-qwen-s41901-p11 19 / 20 / 20. Elsewhere it is the same in every arm.

## 7. The preview nudge test

**Valid proposals** (B submits without a review, C after the preview):

| Point | Kept B / C | Commitment B | Commitment C | Credit-supported B / C |
|---|---|---|---|---|
| ss-gemma-pv-s42001-p14 | 10 / 10 | $3,069.98 | $3,069.98 (10) | 0 / 0 |
| ss-qwen-pv-s42002-p6 | 20 / 20 | $44,687.30 | $44,687.30 (20) | 20 / 20 |
| ss-qwen-s42002-p7 | 20 / 0 | $3,228.33 | $135.00 (10), $300.00 (4), $303.75 (2), $0 (4) | 0 / 0 |
| fl-qwen-s41902-p24 | 20 / 20 | $0.00 | $0.00 (20) | 0 / 0 |
| fp-qwen-s41901-p26 | 20 / 0 | $9,863.02 | $9,863.02 (20) | 0 / 0 |
| fp-qwen-s41901-p25 | 20 / 20 | $9,863.02 | $9,863.02 (20) | 0 / 0 |

**Refused proposals:** see §6. Valid corrections under C against B:
* 12 against 13 at fl-qwen-s41901-p9;
* 4 against 5 at fp-qwen-s42001-p24;
* equal elsewhere.

Abandonment was equal, or within one draw.

Per point:
* **The preview never raised a commitment at a valid proposal.** At 4 of
  6 valid points it
  left the proposal unchanged. At 1 it led to cutting the commitment
  ($3,228.33 to $135–$303.75, or nothing).
* **It did not lower the credit-supported $44,687.30 proposal,** which
  was kept 20/20. B also submitted it 20/20, without asking the model
  again.
* Whether the preview "reinforced" it cannot be told apart from B's
  default, which submits a valid proposal unreviewed. V0.10c's
  validation arm showed the validation answer for a valid proposal and
  led Qwen to cut this one. The validation-first workflow shows nothing
  for a valid proposal.

## 8. Financial commitments and operational exposure

Financial evidence v1 (unchanged) of every distinct final decision is in
[decision-tables.md](decision-tables.md).
* **Commitments changed by a workflow:**
  * ss-qwen-s42002-p7 under C: the retained teleradiology obligation
    dropped, $3,228.33 to $135.00–$303.75;
  * fp-qwen-s42001-p24: B 5 and C 4 draws start a teleradiology contract,
    $2,225.80, with a retained obligation, where A starts none;
  * fp-qwen-s42001-p7: every arm hires, $11,159.83 (A once $11,700.00);
  * fl-gemma-s41901-p28: every arm starts the corrected contract,
    $3,023.32, credit-supported.
* **Credit-supported submissions:** A 30, B 30, C 30 draws: the
  $44,687.30 allocation (20 each) and Gemma's corrected contract (10
  each).
* **Terminations and displaced bookings:**
  * fl-qwen-s41902-p24's valid proposal terminates two employees (6
    bookings displaced) under every arm;
  * B's and C's corrections at fl-qwen-s41902-p27 terminate a
    technologist (100 bookings displaced, 20/20 each);
  * at fl-qwen-s41901-p9, 13 B and 12 C draws do so (8 displaced).

  A's workflows there ended with nothing submitted. Terminations are not
  reversible.
* **Cash-only headroom** was negative at all seven low-liquidity points,
  from −$67,446.87 to −$33,275.37, and positive elsewhere.

A smaller commitment is not called better, a valid correction is not
called good, and an accepted preview is not called good judgment.

## 9. Statistical interpretation

The effects, kept separate:
* **Workflow effects:**
  * C − B is zero at 10 of 14 points; at 2 valid Qwen points it is 100 pp
    and 80 pp of revision.
  * B − A is zero at every valid point (both submit) and at Gemma's
    points. At 3 refused Qwen points it is +25 to +100 pp of valid
    corrections.
* **Inference variability:**
  * Gemma: one decision per arm everywhere.
  * Qwen: 1–5 distinct decisions per arm at refused points, and 1–4
    under C at valid ones.
  * A and B at valid points involve no inference.
* **Branch-point heterogeneity:** every non-zero difference is at a
  particular point. Pooled means (decision-tables.md) average 10 zeros
  with a few large values.
* **Model differences:** the three workflows were indistinguishable for
  Gemma at its four points, and differed for Qwen.
* **Financial exposure:**
  * C lowered or kept commitments;
  * B (and C) made refused proposals valid, sometimes by terminating
    staff;
  * A more often ended with nothing submitted.

**A local causal reading** at a point needs:
* the workflows share everything before the proposal (checked);
* assignment by the schedule (holds);
* inference noise exchangeable between arms, and no cache interference
  (not checked);
* the draws are repetitions of one workflow.

Even then, a workflow's message content cannot be separated from the
extra model call it makes. Nothing generalizes beyond these points,
models and server. No reason for a decision is inferred.

## 10. Replay and cross-platform evidence

* **The live run** (`live/`):
  * `draws.jsonl` holds every draw: its schedule entry, every model call
    (request SHA-256, full response and its SHA-256, content, finish
    reason, usage, latency), the workflow trace and every engine request
    and answer;
  * the server's state before, at each round and after;
  * the endpoint is not recorded.
* **The analysis** (`workflow_experiment.py analyze`):
  * runs every draw's workflow again, with the recorded responses matched
    by request SHA-256, on a fresh engine session at the point. 720
    draws, 572 calls and 2,299 engine requests were checked, all equal;
  * then evaluates each distinct final decision with financial evidence
    v1 and the no-action suffix. The 21 applied ones agreed with their
    validation.
* **Identity:** made on the Pi, regenerated on horo (linux/amd64; twice, and once more after the fix in §11)
  and on every CI platform. All byte-identical (§11).
* **The points:** checked again in CI.
* **Live metadata:** latency, tokens and cache (568 of 572 calls cached)
  are in `inference.json`, apart.

## 11. Quality gates

| Gate | Where | Result |
|---|---|---|
| Focused tests: `analysis/repeatability` (11 new for v4) | Pi, horo and CI | 85 tests, OK |
| Agent tests (`agents/llm`), evidence tests (`analysis/evidence`) | horo and CI | 89 and 147 tests, OK |
| Go suite (`go test ./...`), golden references | CI, three platforms | run by CI on this commit; the new `v0111` session tests also on the Pi |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64 | exit 0; 10 packages ok, no failure |
| `go vet ./...` (default, calibration, acceptance tags), `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64) | horo | all ok |
| Historical replays (V0.8c, V0.9c) | horo and CI | 104 checks, 0 failed, each of 4 replays |
| Benchmark verify and reproduce | CI | run by CI on this commit |
| Branch points made again (selection, engine answers, requests) | horo and CI | 14 contexts verified |
| Every draw replayed without a model, analysis regenerated and compared | Pi, horo (three times) and CI | 720 draws, 572 responses, 2,299 engine requests equal; byte-identical (see the note) |
| Cross-platform CI (linux/amd64, linux/arm64, darwin/arm64) | CI | run by CI on this commit; declaration commit 562ed93 green (run 41471597358) |

**A cross-version fix found by the gates.** The first horo comparison
found one difference in `decisions.json`: a pooled mean of exactly
0.21875 was rounded to 0.2187 on the Pi (Python 3.11) and to 0.2188 on
horo (Python 3.13). Python 3.12 made `sum()` of floats compensated, so
the computed mean differed in its last bit. The pooled means now use
`math.fsum`, which is correctly rounded on every version. Only that one
value changed (to 0.2188), and no per-point result. The fixed analysis
was rerun on horo and equals the committed file. The V0.10b and V0.10c
analyzers, frozen, use the same `sum()`. Their committed outputs are
unaffected, as CI's comparisons show.

Not run: no clinical suite, no historical live evaluation rerun, and no
local repeat of the gates CI runs (Go suite, golden references, verify
and reproduce).

## 12. Known limitations

* **Selection:**
  * Gemma has one valid proposal and no credit-supported one;
  * one credit-supported proposal exists in all the recordings;
  * two valid Qwen points are similar allocations from one run;
  * 9 of 14 points were seen in V0.10b or V0.10c, and the rule was
    written after their results.
* **Unequal exposure:**
  * at valid proposals, A and B make no model call;
  * the arms' inference exposure is unequal and counted, not
    equalized.
* **B is an experiment configuration** of the driver, not a production
  agent.
* **`maxRetries` 1**, as frozen. A larger budget could change A's
  refused-point results.
* **The driver** reproduces the agents' loops; it is not the agent
  process. Its fidelity is tested on scripted cases, and the decision
  budget's wall clock is not applied.
* **Scope:** immediate decisions only. The no-action continuation is a
  mechanics check, with no long-run claim.

## 13. V0.11.0 release recommendation

**Recommend preparing V0.11.0** as the validation-first workflow release,
after owner review. The release would contain:
* llm-agent 5 (unchanged since V0.11a);
* the shared-prefix v4 workflow method;
* this evaluation.

**What V0.11b says:**
* **Validation first:** llm-agent 5's exact feedback is the part with a
  measured effect. At 3 of 5 refused Qwen points it produced valid
  corrections where llm-agent 3's retry loop produced none.

  Whether those corrections are good is open. Two of them terminate
  staff.
* **The selective preview** adds little at refused proposals. At valid
  ones it left 4 of 6 proposals unchanged, cut one commitment, and
  removed one no-op action. It did not raise a valid proposal's
  commitment, and it did not
  lower the $44,687.30 credit-supported one.
* **Not established:** whether any of this improves decisions.

**Decision for the owner:**
* whether llm-agent 5 with the preview (C) or without it (B) should be
  the default;
* B has the same measured validation benefit with fewer model calls and
  less variability;
* C is the one that may review an expensive valid proposal.

The independent evidence review remains PENDING.
