# V0.11b: controlled evaluation of the validation-first workflow

Status: **declared** (0.11.0-dev.2). The protocol is committed before any
live request; results follow in a later commit. V0.11 is unreleased. The
independent evidence review is still PENDING.

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
