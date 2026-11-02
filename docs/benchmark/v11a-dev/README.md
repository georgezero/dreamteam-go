# V0.11a: validation-first agent decision workflow

Status: **done** (0.11.0-dev.1), awaiting owner review. V0.11 is
unreleased. This is an implementation and deterministic-testing
milestone: no live model was run, and no simulator mechanics, accounting,
action schema, scoring, scenario, existing agent or prompt changed.

* **The design:** [V0.11_AGENT_WORKFLOW.md](../../V0.11_AGENT_WORKFLOW.md),
  `llm-agent-workflow/v1`. It covers the workflow, the validation feedback
  specification, the preview eligibility rule, the preview presentation,
  mixed batches, the recording contract, the test evidence and the
  limitations.
* **The agent:** llm-agent 5, `agents/llm/llm_agent5.py`. It loads
  llm-agent 3 and 4 unchanged.
* **The test evidence:**
  * `agents/llm/test_llm_agent5.py` (29 tests);
  * `internal/bench/llm_agent5_test.go` (the real engine and the mock
    model, live and replayed);
  * [scripted-cases.md](scripted-cases.md) and `scripted-cases.json`,
    made by `analysis/workflow/scripted_cases.py` and compared by CI on
    every platform.
* **The reviewer handoff:** [REVIEWER_HANDOFF.md](../../REVIEWER_HANDOFF.md).
  **INDEPENDENT REVIEW: PENDING.** No reviewer has participated, and no
  label, agreement, precision or recall exists.

## 1. Baseline

At clean `main` 728fbf5 (the V0.10.0 release, tag `v0.10.0`: object
8230a31 → 728fbf5, unchanged), before any V0.11a change. Each aggregate
is the SHA-256 of the lines `<path> <sha256>\n`, in path order.

| Set | Files | SHA-256 |
|---|---|---|
| Simulator: Go sources without tests, the version package and the session-registration files (internal/session/record*.go); scenarios; go.mod | 127 | `95c8643de0679346758ebb6cc704593c669c33a25355c59d34f9f0750789205b` |
| Session registration (internal/session/record.go, record_v1.go) | 2 | `0fe053e41dd8ec670a32b549e6258589c4df5ab722f4a4044f28500d4d487a74` |
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
| Session fixtures (testdata/sessions/*.json) | 116 | `e0c494c818286943cbd24621c707e3f8295d4ca5b30f4878143c82a6cfcb3392` |
| Prompts: llm-agent-prompt/v2 system prompt; llm-agent-preview/v1 treatment | | system `0efc36e21b592abc4acb4c8b78ee7634b39755580a0f5ac348a5be1f8a80c4e1`, treatment `f8b7d787b59c36373d4a18135c1503ccf64e3dd956c749be42fdfb0dbe5c54fa` |
| Decision-record contracts | | `dreamteam.llm-decision-record/v3`, `dreamteam.llm-decision-record/v4` (llm-agent 1 and 2: `llm_agent.py`, unchanged) |

**On the V0.11a tree,** every set above is unchanged except two:
* **The session registration:** 0.10.0 is added as batch mode `v0.10.0`,
  with every earlier mode kept. The aggregate is
  `32d13747624e29304e59bdeef1e1fc6076797956c073155a44d0387d9fd11397`.
* **The session fixtures:** 118 files, aggregate
  `56182c935663ef7534f38737656b9af10d88778c17173498ad65002defee4eb9`. The
  two new files are `mini-seed52-v0100.*`, made from a clean 728fbf5
  worktree.

The changed sets therefore exclude llm-agent 1–4, the prompts, the
decision-record contracts v3 and v4, shared-prefix v1–v3, the V0.10
evidence, the frozen contracts, the recordings and the golden references.

## 2. Deterministic test results

[scripted-cases.md](scripted-cases.md) lists the 20 scripted decision
points:

| Behaviour | Cases |
|---|---|
| Valid, submitted without a preview turn | operations_only, not_eligible (preview not eligible) |
| Valid, preview shown, then kept / revised / abandoned | keep, mixed, retained_obligation, displaced / revise_after_preview / abandon_after_preview |
| Refused, then corrected / abandoned | refused_corrected, refused_corrected_then_previewed, mixed_operations_refused / refused_abandoned |
| Refused repeatedly, the budget exhausted (nothing submitted) | repeated_invalid, budget |
| Revised after the preview, then refused, then abandoned | revised_fails |
| Preview failing or too large (submitted as validated) | preview_error, too_large |
| Model timeout, unreadable output | timeout, invalid_output |
| State changed by a read-only request (nothing submitted) | mutate |

In every case where something was submitted, the used attempt's batches
were all valid. No refusal became an accepted action.

## 3. Quality gates

| Gate | Where | Result |
|---|---|---|
| Focused workflow tests (`agents/llm/test_llm_agent5.py`, 29) and the existing agent tests | Pi, horo and CI | `agents/llm`: 89 tests, OK |
| Engine and mock-model tests of llm-agent 3, 4 and 5 (`internal/bench`, `TestLLMAgent*`) | Pi, horo, CI | ok |
| Evidence tests (`analysis/evidence`), repeatability tests (`analysis/repeatability`) | horo and CI | 147 and 74 tests, OK |
| Go suite (`go test ./...`), golden references, historical session replay (new `v0100` tests included), seed guards | CI, three platforms; race run on horo | passed |
| Race (`go test -race -timeout 90m ./...`) | horo, linux/amd64 | exit 0; 10 packages ok, no failure |
| `go vet ./...` (default, calibration, acceptance tags), `gofmt -l` | horo, and CI | vet ok; gofmt clean |
| Five cross-builds (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64) | horo | all ok |
| Historical replays, benchmark verify and reproduce, V0.10 analyses | CI | run by CI on this commit |
| Scripted-case evidence regenerated | Pi, horo and CI | byte-identical |
| Cross-platform comparison (linux/amd64, linux/arm64, darwin/arm64) | CI | run by CI on this commit |

Not run: no live inference, no clinical suite, no historical live
evaluation.

## 4. Known limitations

See [V0.11_AGENT_WORKFLOW.md](../../V0.11_AGENT_WORKFLOW.md) §9. Above
all:
* no live model has run this workflow;
* whether it improves decisions, or still encourages keeping expensive
  commitments, is V0.11b's question.
