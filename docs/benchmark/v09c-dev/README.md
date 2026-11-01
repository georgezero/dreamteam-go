# V0.9c: prospective preview evaluation

**Status: evaluated.** The plan is [PLAN.md](PLAN.md) (`v09c-plan/1`),
declared at d4d2f2f before any live run and not amended. The treatment
isolation record is [TREATMENT.md](TREATMENT.md). The tables are
[preview-tables.md](preview-tables.md) (`dreamteam.preview-evaluation/v1`,
[preview-evaluation.json](preview-evaluation.json)), and the financial
evidence of both conditions is [financial-tables.md](financial-tables.md)
([financial-evidence.json](financial-evidence.json)). The independent
review is still PENDING, and V0.9 is unreleased.

**Question:** does showing the agent the simulator's existing preview
before it submits a staffing proposal change the management decisions it
proposes and executes?

**Answer, within these 16 runs:**
* **At the decision itself, yes, in one specific way.** Nine previews were
  shown, all to proposals with a staffing action:
  * the five that showed a refused action were revised (4) or abandoned
    (1);
  * the four that showed every action accepted and funded were kept.
* **What the previews did not test.** No preview showed an unfunded
  commitment or a termination's displaced bookings, because no agent
  proposed one in the preview-enabled runs. So the consequences V0.9b
  identified were never put in front of a model.
* **The run-level comparison cannot separate the preview from the model's
  own variability.**
  * The server did not return the same answer to byte-identical requests
    at temperature 0. That held for Qwen at every matched point (0 of 5)
    and for Gemma at 6 of 91.
  * Every Qwen pair diverged at point 0 or 1, before or at its first
    preview.
  * One Qwen pair had no preview at all, and still differed by
    −$6,646.03 in operating profit and 9 more submitted actions.
* **Gemma** proposed a staffing action at one point in its four
  preview-enabled runs, kept it after the preview, and its run outcomes
  equal the baseline's in all four pairs.

## 1. Baseline

* **Start:** clean main at 8ffae8d (V0.9b, accepted).
* **Tag:** `v0.8.0` (object db5c147) still points at b803302. No tag was
  moved.
* **Present and unchanged at 8ffae8d:**

| Item | File | SHA-256 |
|---|---|---|
| llm-agent 1 and 2 | agents/llm/llm_agent.py | `5214e7bfb456075f38aca92a5c45a6db3141d4042fb1cd5822c8c1c65c286d87` |
| llm-agent 3 (the baseline) | agents/llm/llm_agent3.py | `ef2c17a06749d7c287cda918f77ff0e518d79475048e87fd3404848c2fb0015c` |
| model settings (both conditions) | docs/benchmark/v05c-dev/llm-qwen36-35b-a3b.json | `e19a47dce948a8acbd3262ef5053b89f1e23c32603860e03628d38b184acbf5d` |
| | docs/benchmark/v05c-dev/llm-gemma4-26b-a4b.json | `1a1edae5a492b67a72086bec51faecc8ed0930800bdb6e548412b65fa444aa6a` |
| financial evidence v1 | analysis/evidence/financial_evidence.py | `8158addf8a8236b5b0426d7e520b13325291aa8b7804d81c8f8959de51cbe4b8` |
| | docs/benchmark/v09b-dev/financial-evidence.json | `bf61212f832eef1a76bca3f9fd827af34251aad6d3d55c5719a85f2a52341b1a` |
| parameter evidence v1 (frozen) | analysis/evidence/frozen-parameter-v1.json | `27e13a79f7787512a9a4a36fcf11fdb437823a608bad02b210ee51aece948619` |
| | docs/benchmark/v09a-dev/parameter-evidence.json | `9f556c69ad187c22b9dbb1c4a3149b1e450c368b97815b1e3e75368ae3c01a71` |
| decision evidence v1 (frozen) | docs/benchmark/v08a-dev/evidence.json | `5743bc223965e75d77440021ecf602b74c1e4bb35c6b6a08afe8be08b8955d4d` |
| | analysis/evidence/frozen-v1.json | `64a8e015f9b6901447ec6ddbd979e8882047cdfef62911cf6db983db7ff76a78` |
| repeat evidence v2 | docs/benchmark/v08c-dev/history/repeat-v2.json | `a80b704cc62d960c46075b723801ef1c8cd47b682185f3ed27daa72abfde2624` |
| refusal-response v1 | docs/benchmark/v08b-dev/refusal-response.json | `55d22b53d65cc49f6e72a2379238d5ebfa80f7e0027810020fd3a068ff7d3cc9` |
| refusal-response v2 | docs/benchmark/v08c-dev/history/refusal-response-v2.json | `cdb0e1cc0d59495f09733d2eb0a1e8b5e98dc1edfc574cd2010f2a242e6f0aa0` |
| review status (PENDING) | docs/benchmark/v09a-dev/review/review-status.json | `36e6efabcb42fae765d52cbe72d7d31ebafc57a232e4f3f7aed4ffd8011bfed4` |

Aggregates (SHA-256 of the lines `<path> <sha256>` in path order):

| Set | Files | Aggregate at 8ffae8d |
|---|---|---|
| simulator: Go sources without tests and the version package, scenarios, go.mod | 129 | `2316df3d8bd1a3016806e7af6c398bc611b3af6b2cc9e21981ceab8c2bb92f58` |
| historical recordings, V0.5c–V0.8c (`*-decisions.jsonl`) | 96 | `041a30b0788edc3017b18b2fa07d36087e73f5786af33d33a98559794234b5c3` |
| benchmark suites (docs/benchmark/suite) | 4 | `63c007ccedc023a501690aa14ca76d400a18be68024cfa658ec5804bd7265c65` |
| golden references (testdata/golden, without MIGRATIONS.json) | 12 | `d84f80718f3ab0463cc3768cf89b5ba09885b654e674a92a87c82c6998a2b25f` |
| testdata/golden/MIGRATIONS.json | 1 | `d0e29c0f96d68607dacadc6bd7129553f793a236ee870ad2c6a85a4674713599` |
| session fixtures (testdata/sessions/*.json) | 104 | `ecc3931fb4697d0ddf356f44034179ae61c541f0c8b5ee311ec97476b3444845` |

**What the declaration changes among them,** as every version does:
* the version, 0.9.0-dev.3;
* the registration of 0.9.0-dev.2 for replay. That is two lines in
  `internal/session/record.go` and one in `record_v1.go`, which change the
  simulator aggregate; no behaviour changes.
* MIGRATIONS.json's engine and explanation line;
* two new session fixture files (`mini-seed52-v092`).

Nothing else in the table changes. llm-agent 3 is not edited: llm-agent 4
loads it.

## 2. The preview-enabled agent

llm-agent 4 (`agents/llm/llm_agent4.py`, [LLM_AGENT.md §14](../../LLM_AGENT.md)):
* **The base is unchanged.** It loads llm-agent 3 from the same file
  (SHA-256 `ef2c17a0…`, unchanged), with one step added.
* **When it applies.** At a decision point, the first parsed proposal that
  has a staffing action is previewed with the protocol's existing
  `preview`.
* **What the model sees and does.** It is shown the answer exactly as
  returned, in the fixed review message `llm-agent-preview/v1` (SHA-256
  `f8b7d787…`). It then keeps, revises or abandons the proposal, and that
  decision is validated and submitted as version 3's would be.
* **When the preview cannot be shown.** A failed or oversized preview is
  recorded, not shown, and the proposal is validated unchanged. This
  happened at no point.
* **Recording and replay.** The record is
  `dreamteam.llm-decision-record/v4`. The replay asks every preview again
  and rebuilds the review message.

**Compatibility:** llm-agent 3 and its recordings are unchanged (§1), and
every earlier recording replays as before (§12). Version 4 replays only
version 4 records.

## 3. The prospective declaration

* **Committed before any live run:** d4d2f2f (`v09c-plan/1`), with CI run
  41258183735 green on linux/amd64, linux/arm64 and darwin/arm64.
* **What it fixed:**
  * the models and settings;
  * the agent hashes and the treatment text;
  * the scenario fingerprints and the seeds (42001, 42002);
  * the horizon and run order;
  * the resource limit and stop conditions;
  * the 20 outcomes, analyses A–G and the replay requirements.
* **Not amended.** The analysis code was completed after the runs, as the
  plan allows, and implements its definitions.
* **Two analysis fixes, both found before any result was used:**
  * financial evidence v1's commitment helper returns a pair: unpacked;
  * the transcript adapter of §7.

## 4. Live evaluation

| | |
|---|---|
| Runs | 20 of 20 completed: 16 live LLM runs (8 pairs) and 4 no-action runs |
| Where | Raspberry Pi 4 (linux/arm64), one suite run, memory-capped, the only heavy job |
| Time | 2026-11-01 00:46:51 to about 02:00 UTC, about 70 minutes (limit 6 hours) |
| Engine | 0.9.0-dev.3, built from a clean worktree at d4d2f2f |
| Models | both served under their declared names; neither was unavailable or replaced |
| Model requests | 555, every one answered (no transport failure, no infrastructure-affected point, no decision budget stop); 93 replies could not be used (parse failures or validate refusals) and were retried as the frozen settings say |
| Failures and reruns | none; nothing was rerun or replaced |

## 5. Preview behaviour (B; outcomes 1–6)

Denominators: decision points (30 a run); eligible points (a first parsed
proposal with a staffing action); previews shown.

| | Qwen | Gemma | All |
|---|---|---|---|
| Decision points, preview-enabled runs | 120 | 120 | 240 |
| Eligible (first proposal with a staffing action) | 8 | 1 | 9 |
| Previews shown | 8 | 1 | 9 |
| Preview failures (`failed`, `too_large`) | 0, 0 | 0, 0 | 0, 0 |
| Shown with an action the simulator refused | 5 | 0 | 5 |
| Kept | 3 | 1 | 4 |
| Revised | 4 | 0 | 4 |
| Abandoned | 1 | 0 | 1 |
| Final submitted against the proposal: kept, revised, abandoned, fallback | 3, 3, 1, 1 | 1, 0, 0, 0 | 4, 3, 1, 1 |
| Review requests added | 8 | 1 | 9 |
| Review message characters (largest) | 26,953 (5,825) | 2,554 | 29,507 |

**Every preview shown** ([preview-tables.md §2](preview-tables.md)):

| Run, point | Proposal's staffing part | What the preview said | Response | Final |
|---|---|---|---|---|
| Qwen financial 42001, 13 | teleradiology | refused: `invalid_schema` | revised | revised |
| Qwen staffing 42001, 0 | two premium hires | refused: `lead_too_short` | revised | revised |
| Qwen staffing 42001, 1 | teleradiology | accepted; commitment $2,290.11, headroom $84,102.15, funded | kept | kept |
| Qwen staffing 42001, 26 | two rosters, teleradiology | refused: `not_monday` | revised | fallback |
| Qwen staffing 42001, 27 | two rosters, teleradiology cancellation | accepted; commitment $0.00, headroom $50,490.57, funded | kept | kept |
| Gemma staffing 42001, 14 | teleradiology | accepted; commitment $3,069.98, headroom $103,953.60, funded | kept | kept |
| Qwen staffing 42002, 1 | teleradiology | refused: `invalid_schema` | revised | revised |
| Qwen staffing 42002, 6 | two allocation raises | accepted; commitment $44,383.55, headroom $87,205.37, funded | kept | kept |
| Qwen staffing 42002, 26 | five rosters | refused: `notice_too_short` | abandoned | abandoned |

**What the records show:**
* **A refused action meant a revised or abandoned proposal (5 of 5).**
  After four of them the model's reason describes the refusal as a failed
  attempt. For example: "The previous hire attempts failed because the
  start date was too early for the premium lead time (21 days)." Here the
  preview did what validate's feedback does for version 3, one step
  earlier.
* **An accepted, funded preview meant a kept proposal (4 of 4).**
* **The credit line.** The allocation raises at Qwen staffing 42002,
  point 6, were funded only with the credit line: headroom without credit
  was below the $44,383.55 commitment, so financial evidence v1 classes
  the batch `funded_with_credit`. The preview's funding line states that
  basis. The model kept the proposal, and its recorded reason says: "The
  preview confirms the reading allocation changes are valid and funded."
  No reasoning beyond that is inferred.
* **Not tested.** No preview showed an unfunded commitment, a
  termination, a hire's guaranteed costs together with a refusal for
  funds, or displaced bookings. In the preview-enabled runs no agent
  proposed a termination, and no proposal failed the funding test.

## 6. Matched decision points and model variability (A)

The two conditions' first messages are built by the same code. Wherever
the practice and the memory are the same, the requests are byte-identical:
same model, temperature 0, same messages
([preview-tables.md §9](preview-tables.md)).

| Pair | Points with the same first message | … same first output | … same first proposal | First submitted difference |
|---|---|---|---|---|
| Qwen financial 42001 | 1 | 0 | 0 | point 0 |
| Qwen financial 42002 | 2 | 0 | 1 | point 1 |
| Qwen staffing 42001 | 1 | 0 | 0 | point 0 |
| Qwen staffing 42002 | 1 | 0 | 0 | point 0 |
| Gemma financial 42001 | 16 | 15 | 16 | none |
| Gemma financial 42002 | 30 | 30 | 30 | none |
| Gemma staffing 42001 | 15 | 14 | 15 | none |
| Gemma staffing 42002 | 30 | 26 | 30 | none |

**The server's answers varied at identical inputs.**
* *Qwen:* 0 of 5 identical requests got the same output. At Qwen
  financial 42001, point 0, before any preview, one request proposed two
  terminations and an allocation change; the identical request in the
  other condition proposed four shift extensions.
* *Gemma:* 85 of 91 got the same output, and 91 of 91 the same proposal.
* *A likely source, not established:* the second of two identical requests
  was served largely from the server's prompt cache (9,592 of 9,599
  prompt tokens at that point), and cached and uncached computation need
  not agree to the bit.

**Consequences for the comparison:**
* Every Qwen pair diverged at point 0 or 1, at or before its first
  preview. Qwen financial 42002's preview-enabled run was never shown a
  preview, yet the pair differs by 9 submitted actions and −$6,646.03 of
  operating profit.
* The Qwen pairs therefore measure the model's variability together with
  any effect of the preview, and cannot separate them. The decision-level
  evidence of §5 is not affected: there, each proposal is compared with
  the same model's reply to its own preview.
* Gemma's pairs are informative but small. Gemma acted at one point (the
  same teleradiology contract in both conditions), and its four pairs end
  identical in every run outcome.

## 7. Financial commitments (C; outcomes 10–11)

`dreamteam.financial-evidence/v1`, unchanged, on both conditions (425
records; [financial-tables.md](financial-tables.md)). Every batch the
records say was validated has its affordability record (checked by the
analysis).

**How the v4 recordings are read:**
* *The v3 view:* the attempt fields v1 links are the same, and the run list
  keeps each file's own SHA-256 and format.
* *The preview checks removed:* llm-agent 4 asks for the digests after
  each preview, to check that nothing changed. v1 starts a decision point
  at a digests request carrying the point's digests, so that check would
  end the point early and drop everything after it. The analysis gives v1
  the transcript without those requests (9, one per preview shown). They
  are read-only, and v1's engine replay still requires every remaining
  answer to be identical.

Denominator: submitted batches.

| Run (baseline / preview-enabled) | Batches | Funded without credit | Funded with credit | Refused for funds | Funded commitments |
|---|---|---|---|---|---|
| Qwen financial 42001 | 32 / 12 | 5 / 9 | 0 / 0 | 0 / 0 | $3,404.56 / $6,301.29 |
| Qwen financial 42002 | 13 / 18 | 4 / 15 | 0 / 0 | 0 / 0 | $2,019.18 / $8,653.65 |
| Qwen staffing 42001 | 35 / 15 | 19 / 7 | 0 / 0 | 0 / 0 | $35,459.83 / $53,889.60 |
| Qwen staffing 42002 | 19 / 13 | 13 / 5 | 0 / 3 | 0 / 0 | $7,044.06 / $49,008.34 |
| Gemma staffing 42001 | 1 / 1 | 1 / 1 | 0 / 0 | 0 / 0 | $3,069.98 / $3,069.98 |
| Gemma financial 42001, 42002; Gemma staffing 42002 | 0 / 0 | — | — | — | — |

* **No batch was refused for insufficient funds,** in either condition.
* **Credit-supported batches:** only the three in Qwen staffing 42002's
  preview-enabled run. One of them, the allocation raises, was the
  previewed and kept proposal (§5).
* **The credit line was never drawn,** in any run.

## 8. Reversibility (D; outcome 12)

Denominator: executed actions.

| Run (baseline / preview-enabled) | Executed | Withdrawable | Obligation stays | Not reversible | No recorded cost |
|---|---|---|---|---|---|
| Qwen financial 42001 | 10 / 11 | 1 / 0 | 2 / 1 | 7 / 10 | 0 / 0 |
| Qwen financial 42002 | 5 / 15 | 0 / 0 | 0 / 0 | 4 / 15 | 1 / 0 |
| Qwen staffing 42001 | 41 / 11 | 6 / 2 | 3 / 3 | 32 / 6 | 0 / 0 |
| Qwen staffing 42002 | 18 / 20 | 0 / 2 | 1 / 1 | 17 / 16 | 0 / 1 |
| Gemma staffing 42001 | 1 / 1 | 0 / 0 | 1 / 1 | 0 / 0 | 0 / 0 |

Shift extensions make up most of "not reversible" in both conditions, as
in V0.9b. No action was of an unknown type.

## 9. Direct operational exposure (E; outcomes 13–14)

| Run (baseline / preview-enabled) | Direct changes by type | Displaced bookings stated by the engine |
|---|---|---|
| Qwen financial 42001 | teleradiology 4, extensions 3, allocation 1, terminations 2 / teleradiology 1, extensions 10 | 69 (one termination batch) / 0 |
| Qwen financial 42002 | extensions 4, worklist policy 1 / extensions 15 | 0 / 0 |
| Qwen staffing 42001 | teleradiology 1, extensions 32, hires 2, rosters 6 / teleradiology 2, extensions 5, hires 2, rosters 2 | 0 / 0 |
| Qwen staffing 42002 | teleradiology 1, extensions 17 / teleradiology 1, extensions 16, allocations 2, worklist policy 1 | 0 / 0 |
| Gemma staffing 42001 | teleradiology 1 / the same | 0 / 0 |

The only terminations in the evaluation were the baseline's at Qwen
financial 42001, point 0, which displaced 69 bookings. That point's
identical request in the preview-enabled condition proposed shift
extensions instead (§6), so no termination was ever previewed.

## 10. Paired financial outcomes (G; outcomes 15–16)

Preview-enabled minus baseline, same model, entry, seed, horizon and
economics. Whole-run differences: none is attributed to one decision or to
the preview, for the reasons of §6.

| Pair | Operating profit | Funded commitments | Credit-supported batches | Most credit drawn | Shortfall, days below zero |
|---|---|---|---|---|---|
| Qwen financial 42001 | +$25,652.39 | +$2,896.73 | 0 | $0.00 | none in either |
| Qwen financial 42002 (no preview shown) | −$6,646.03 | +$6,634.47 | 0 | $0.00 | none in either |
| Qwen staffing 42001 | −$5,228.73 | +$18,429.77 | 0 | $0.00 | none in either |
| Qwen staffing 42002 | −$14,429.19 | +$41,964.28 | +3 | $0.00 | none in either |
| Gemma, all four pairs | $0.00 | $0.00 | 0 | $0.00 | none in either |

**Against no action** (operating profit; [preview-tables.md §8](preview-tables.md)):

| Pair | Baseline | Preview-enabled |
|---|---|---|
| Qwen financial 42001 | −$27,548.48 | −$1,896.09 |
| Qwen financial 42002 | −$2,019.18 | −$8,665.21 |
| Qwen staffing 42001 | −$11,507.00 | −$16,735.73 |
| Qwen staffing 42002 | −$2,872.38 | −$17,301.57 |
| Gemma staffing 42001 | +$1,233.00 | +$1,233.00 |

Gemma's other three pairs equal no action in both conditions. No run had
financial distress (no shortfall, no day below zero), and no run drew on
the credit line.

## 11. Paired operational outcomes (F; outcomes 17–20)

Preview-enabled minus baseline:

| Pair | Exams | Orders lost | TAT breaches | TAT p90 (min) |
|---|---|---|---|---|
| Qwen financial 42001 | +119 | −212 | +35 | +65 |
| Qwen financial 42002 (no preview shown) | 0 | 0 | +2 | −18 |
| Qwen staffing 42001 | −51 | −51 | −69 | −754 |
| Qwen staffing 42002 | +3 | −3 | +17 | −497 |
| Gemma, all four pairs | 0 | 0 | 0 | 0 |

The differences go both ways and are as large in the pair with no preview
as some of the others. The observed state at every decision point is in
the financial evidence's run timelines (B), with no attribution.

## 12. Replay and cross-platform reproducibility

Every completed live run was replayed without a model: llm-agent 3's
recordings by its replay, llm-agent 4's by its own, beside no action run
again. One replay configuration per entry: `eval-replay-*.json`.

**Checked in replay, at every decision point** (`replayChecked`, summed
over the 16 LLM runs):
* 555 model inputs rebuilt and matched by SHA-256. That is every attempt:
  first prompts, details answers, retries and the 9 review messages.
* 159 validate answers, asked again of the engine and compared in full.
* 9 previews, asked again at exactly the attempt the rule puts them, and
  compared in full, with the digests unchanged after each.
* The memory, the prompt, the submitted batches and each point's preview
  summary.

| Check | Result |
|---|---|
| Live and replay, byte for byte | 60 of 60 files equal: `decisions.json`, `session.json` and `result.json` of all 20 runs. The financial ledger, credit and interest are in `session.json` and `result.json` |
| `benchmark verify` | 104 checks, 0 failed, for each entry, on linux/arm64 and linux/amd64 |
| `benchmark reproduce` | 0 failed for each entry, on both, with byte-identical manifests |
| Manifests, linux/arm64 (Pi) and linux/amd64 (horo) | identical: financial pressure `4a0ef67c04e8cc4fa77ec30dce0262072a615968763faba96bc13bd12cc84c4c`, staffing shortage `725f1d30c18d189b4f7099be05424615a3051bcf163d7dbbc43c14dc5b8e6720` |
| The analysis, run twice | byte-identical: `preview-evaluation.json`, `preview-tables.md`, `financial-evidence.json`, `financial-tables.md` |
| CI | replays the recordings and reruns the analysis on linux/amd64, linux/arm64 and darwin/arm64, compares each output with the committed one, and compares the platforms |

**Historical recordings, frozen evidence, golden references and benchmark
fingerprints:** unchanged (§13). No historical live experiment was rerun.

## 13. Quality gates

Run on linux/amd64 (horo) unless noted:
* **Focused tests:**
  * agent tests (60, including the 20 llm-agent 4 tests);
  * evidence tests (147, including the 8 preview evaluation tests);
  * probe tests (3, against the built binary).
  All pass.
* **Go:**
  * full default suite: all packages pass, including
    `TestLLMAgent4Preview`, `TestLLMAgent4Replay` and the new
    `mini-seed52-v092` fixture (0.9.0-dev.2, batch mode `v0.9_dev2`);
  * race (`go test -race -timeout 90m ./...`): pass;
  * vet (three tag sets) and gofmt: clean.
* **Historical replay:** every fixture replays with every digest
  verified; every recording in the corpus was replayed at 0.9.0-dev.3.
* **Benchmark verify and reproduce:** 70 of 70, 0 failed, including the
  two V0.9c entries (104 checks each).
* **Earlier pipelines:** the V0.8c pipeline, run twice, is byte-identical
  and equal to the committed files; `check_v08b.py` reproduces the V0.8b
  review package and probe log, with build numbers set aside; the V0.9a
  outputs are rebuilt identically and the review status is PENDING.
* **Parameter evidence v1 freeze:** the frozen analyzer, corpus,
  recordings and outputs are reproduced.
* **Financial evidence (V0.9b):** run twice, 2,062 records from 78
  recordings, byte-identical and equal to the committed files.
* **Preview evaluation (V0.9c):** the 16 LLM runs replayed and the
  analysis run twice: 8 pairs, 425 financial evidence records,
  byte-identical and equal to the committed files.
* **Earlier evidence, frozen files, baseline agents (llm-agent 1–3),
  suites and golden references:** unchanged; no deletions.
* **Five cross-builds:** linux/amd64, linux/arm64, darwin/arm64,
  darwin/amd64, windows/amd64: all pass.
* **CI** (run 41268808888, on be537f2): linux/amd64, linux/arm64 and
  darwin/arm64, with the cross-platform comparison and the new
  prospective preview evaluation step: all pass.
* **Clinical suites:** not run, as specified.

## 14. Limitations

* **Model variability.** The server's answers differed at byte-identical
  requests at temperature 0 (§6). Run-level pairs therefore cannot isolate
  the preview, and a repeat of this evaluation would not reproduce its
  live decisions. The recordings and their replays are exact.
* **Little exposure to the treatment.** Nine previews in 240 decision
  points, eight of them Qwen's. None showed an unfunded commitment or
  displaced bookings.
* **The preview doubles as validation.** Each action's result is part of
  the preview. Five of nine previews showed a refusal, and the model
  treated it as validate's feedback. V0.9c cannot separate the effect of
  seeing the financial lines from the effect of seeing the refusal
  earlier.
* **Two seeds, two scenarios, two models, 30 days:** a development set.
  No significance test is run and no generalization is claimed.
* **Knowable is not known, again.** A preview shown is evidence of what
  the model was shown. Its reply is quoted, and no understanding is
  inferred.
* **Financial evidence v1 reads v4 through an adapter** (§7): the v3 view
  and the preview checks left out of its segmentation. v1's rules are
  unchanged.
* **The independent review is still PENDING.**

## 15. V0.9.0 release recommendation

**Recommendation: release V0.9.0 after the owner accepts V0.9c,** as the
owner planned whatever the result. The release would be documentation and
the version only: 0.9.0, with 0.9.0-dev.3 registered for replay. No agent,
rule, analysis or evidence would change.

**Why:**
* **V0.9a–V0.9c give one consistent body of evidence:**
  * a frozen parameter evidence v1;
  * an engine-backed financial evidence v1;
  * a preview-enabled agent whose recordings replay exactly;
  * a prospective evaluation declared before it ran.

  Each is versioned, deterministic, and checked on three platforms.
* **The V0.9c result is informative, though not the result hoped for.**
  * Showing the preview changed decisions when it showed a refusal, and
    otherwise left them alone.
  * The consequential financial information (an unfunded commitment,
    displaced bookings) never came up.
  * The run-level comparison is limited by the model server's
    variability.

  Each of these is a finding a release should state.
* **Nothing in V0.9c needs a fix before release.**
  * Both analysis defects were found and corrected before any result was
    used.
  * The adapter for financial evidence v1 is documented and checked.

**What the release notes must say:**
* the independent review is PENDING; no rule is independently validated;
* V0.9c's conclusions are limited to these 16 runs, and its run-level
  pairs do not isolate the preview (§6);
* the preview is not a forecast.

**Not recommended:** rerunning the evaluation, changing the treatment, or
choosing other seeds to obtain a different result.

**Possible later work,** each needing the owner's GO:
* **Measure the server's variability directly,** before any further
  run-level comparison. Repeat identical requests, and use a serving
  setting without prompt-cache reuse if one exists.
* **Make the treatment meet its subject.** Run the preview step where
  commitments often do not fit, such as V0.8c's low-liquidity entry, so
  that the funding test is shown failing. This would use the same declared
  treatment, not a new one.
* **Separate the two things the preview shows.** Showing only the
  financial lines, without each action's result, would be a different
  treatment with its own version.

