# V0.12a: high-impact action acknowledgment

Status: **done** (0.12.0-dev.1), awaiting owner review. V0.12.0 is
unreleased. This was an implementation and deterministic-testing
milestone:
* no live model was run;
* no simulator mechanics, action schema, accounting, scoring, scenario,
  existing agent, prompt or workflow identity changed.

* **Design:** [V0.12_ACKNOWLEDGMENT.md](../../V0.12_ACKNOWLEDGMENT.md).
  It covers the contract `dreamteam.action-acknowledgment/v1`, the
  workflow `llm-agent-acknowledgment-workflow/v1`, the record
  `dreamteam.llm-decision-record/v6`, replay and the limitations.
* **Consequence inventory:**
  [V0.12_CONSEQUENCE_INVENTORY.md](../../V0.12_CONSEQUENCE_INVENTORY.md),
  written before the safeguard was implemented.
* **Agent:** llm-agent 6, `agents/llm/llm_agent6.py`. It loads llm-agent
  5, 4 and 3 unchanged.
* **Test evidence:**
  * `agents/llm/test_llm_agent6.py` (31 tests);
  * `internal/bench/llm_agent6_test.go` (the real engine and the mock
    model, live and replayed);
  * [scripted-cases.md](scripted-cases.md) and `scripted-cases.json`,
    made by `analysis/acknowledgment/scripted_cases.py` and compared by
    CI on every platform.
* **Independent review: PENDING.** No reviewer has participated.

**Acknowledgment is not approval.** A matching acknowledgment shows that
the model's reply stated the engine-reported direct consequences of the
exact proposal. It does not establish that the action is clinically or
financially wise. The engine's direct consequences are not all future
outcomes.

## 1. Baseline

These hashes were taken at clean `main` a267cd3 (the V0.11.0 release, tag
`v0.11.0`: object d84e09a → a267cd3, unchanged), before any V0.12a
change. Each aggregate is the SHA-256 of the lines `<path> <sha256>\n`,
in path order. The script is the one used for
`docs/audits/v0.11.0-release-baseline.md`.

| Set | Files | SHA-256 |
|---|---|---|
| Simulator: Go sources, excluding tests, the version package and the session-registration files (internal/session/record*.go); scenarios; go.mod | 127 | `95c8643de0679346758ebb6cc704593c669c33a25355c59d34f9f0750789205b` |
| Session registration (internal/session/record.go, record_v1.go) | 2 | `fdd93aead9abb305ad7a7f999ddaed8cd575707ed75bcc7c20bc92eb76b64b83` |
| Historical recordings, V0.5a-V0.9c (*-decisions.jsonl) | 114 | `336024d8ba6dd490c99b72c7e626b7f0792ad10d312767dba2deacf45b8f2107` |
| llm-agent 1 to 4 (agents/llm/llm_agent.py, llm_agent3.py, llm_agent4.py) | 3 | `30bfec269265d207156b59738126ab8612027bfa9f7546116aba903c9a72e884` |
| llm-agent 5 (agents/llm/llm_agent5.py) | 1 | `57ec93b8fce3523c3b3fd9fe06977f6cbb9404e2f87136f5b4cdae8714e66d73` |
| Frozen evidence contracts (frozen-v1.json, frozen-parameter-v1.json and their checks) | 4 | `48566731ec58d6ba6bfca1dd1be1f482d4558f95bbdb16bceba33969f9a3a3ec` |
| Evidence analyzers (analysis/evidence/*.py) | 23 | `41bfbea46bc948979ebb3cb2b3e66318b8af0befa788ea8a0a49c7231e5ba6b1` |
| Shared-prefix v1-v4 and repeatability code (analysis/repeatability/*.py) | 13 | `de45f76fcde5fe507fbfa05f508cc47d651a9722d9599da081559b3f4bffd492` |
| V0.11a scripted-case generator (analysis/workflow/*.py) | 1 | `a9db37d980125f5a68d808bd704f6302cc1cccbca6d83f48a1b6308c6800718b` |
| V0.10a repeatability evidence (docs/benchmark/v10a-dev/) | 29 | `615cded61342102233c26433df68efa1112d72d132e6f37ee3995ebe5475eb9e` |
| V0.10b experiment (docs/benchmark/v10b-dev/) | 14 | `b95e433fffecb9b22585c776179a3905f5deaa08b460715ad29bebe08ee2306b` |
| V0.10c experiment (docs/benchmark/v10c-dev/) | 16 | `f3885a15dcf797ac1a78ebb2974eefc0dfddd517a4b74f98f0cd9724ed4b6d0b` |
| V0.11a scripted evidence (docs/benchmark/v11a-dev/) | 3 | `69f5170bc14d82c4b5d149265979b626e3781e3460c09b531fcdcb23c7b035a7` |
| V0.11b prospective evidence (docs/benchmark/v11b-dev/) | 14 | `6138f4ea4ff2cd4a74a1520b9a5e5f4f0d1bc273cc545ea73ad56cf02039decc` |
| Benchmark suites (docs/benchmark/suite/) | 4 | `63c007ccedc023a501690aa14ca76d400a18be68024cfa658ec5804bd7265c65` |
| Golden references, without MIGRATIONS.json | 12 | `d84f80718f3ab0463cc3768cf89b5ba09885b654e674a92a87c82c6998a2b25f` |
| All earlier development evidence (docs/benchmark/v0*) | 479 | `e109a4de7e4de385c412695b6c9d7827fd32a9bc82400976d5adbe188b4b732f` |
| Session fixtures (testdata/sessions/*.json) | 122 | `dc9aaaea1223e76af107a83d1ed88f1478e02425f51cd903522fc9011d8bc230` |
| Prompts: llm-agent-prompt/v2 system prompt; llm-agent-preview/v1 treatment | | system `0efc36e21b592abc4acb4c8b78ee7634b39755580a0f5ac348a5be1f8a80c4e1`, treatment `f8b7d787b59c36373d4a18135c1503ccf64e3dd956c749be42fdfb0dbe5c54fa` |
| llm-agent 5 texts: validation feedback, preview message, operations note | | feedback `d0606d729634a6504aac733f200494e53982f3835e0f4a6824d386818b72b5c7`, preview `9e4f676abcc21cacff8ca90d85f4aafc41c17d7c30ba9785ea9f38e0237f0386`, note `4d8fd9854cfc0933e4c50daab3057b5bfc474dc9551bb896aa40a48f51eb4b5d` |
| Decision-record contracts | | `dreamteam.llm-decision-record/v3`, `dreamteam.llm-decision-record/v4`, `dreamteam.llm-decision-record/v5` |
| llm-agent 5 workflow identities | | `llm-agent-workflow/v1` `3ee7d66f4f10fd93e05a908725c792b16b76d01dc29095360dae9d67c0c09d0a`; `llm-agent-validation-first-workflow/v1` `a0204f628c7631d43ebfbb48e3b6e31071b67252daa76f141e3ad3a43ff5f184` |

## 2. What V0.12a changed

Recomputed on the V0.12a tree, every set above is unchanged except
these:

| Set | Files | SHA-256 | Why |
|---|---|---|---|
| Session registration (internal/session/record.go, record_v1.go) | 2 | `f5f75c8c7334b64c6c17375ec9d615a766aaa6466708f49723181a20f90e04ba` | 0.11.0 registered as batch mode `v0.11.0` |
| Session fixtures (testdata/sessions/*.json) | 124 | `ba0b466882b8debb58235738ed9496ef8596a6add47c949faad88f73b2d922db` | `mini-seed52-v0110.{actions,session}.json` added (made from a267cd3) |

`agents/llm/llm_agent5.py` is unchanged
(`f89dd4e1950662c5e5b3d4e1dd48b8ee23b9ed7e3e4080dbb0ed136267c26cb0`),
and so are both of its workflow identities. These files are new:

| File | SHA-256 |
|---|---|
| `agents/llm/llm_agent6.py` | `57ffad6ba6ec33ac769709c57da3d49e03be8d62b4462231568e5c6f16fe86a9` |
| `analysis/acknowledgment/scripted_cases.py` | `6eb4025ac143ebca15b94e707198e757ad93a045f9620a7d337b317a2b2184ec` |
| `docs/benchmark/v12a-dev/scripted-cases.json` | `0a8d3377aabcaa9a6cb0516812a7bff0334c2efc5c7b789fce812953fe5f6742` |

Identities:
* workflow `llm-agent-acknowledgment-workflow/v1`:
  `b9cb79828f2657de91de69b1fa30019752ab73611d80fb6854c8f7a6488595a6`;
* contract `dreamteam.action-acknowledgment/v1`:
  `33c116da979a5a1067a4f19163369a1092394ca9bfe9ec4aade2dab8cf9c4c9b`.

## 3. Quality gates

Run locally on horo (linux/amd64, Go 1.27.0) from the working tree:

| Gate | Result |
|---|---|
| Race, all packages (`go test -race -timeout 90m ./...`) | exit 0; 10 packages ok, no failures |
| `go vet ./...` | ok |
| `gofmt -l` | clean |
| Cross-builds: linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64 | 5 of 5 ok |
| Frozen evidence checks: v08c credit/low-liquidity, v09c credit/staffing-shortage | 4 × 104 checks, 0 failed |
| Shared-prefix v4 points | 14 contexts, selection and requests verified |
| V0.11b workflow analysis | 720 draws replayed without a model; decisions.json, decision-tables.md, inference.json reproduced |
| V0.11a scripted evidence (20 cases) | scripted-cases.json and .md reproduced |
| V0.12a scripted evidence (27 cases) | scripted-cases.json and .md reproduced |
| Python: agents/llm | 138 tests ok (31 for llm-agent 6) |
| Python: analysis/evidence | 147 tests ok |
| Python: analysis/repeatability | 88 tests ok |

On the Pi: `TestLLMAgent6Acknowledgment` (real engine) and `TestV0110Record*` pass.

The CI run on the commit covers the rest, so these were not repeated locally: the full non-race Go suite, golden references, historical replay, benchmark verify/reproduce, the V0.12a evidence regeneration, and the Linux/macOS/Windows matrix.

Not run: live inference, the clinical suites, and historical live reruns. The owner's instructions exclude all three.
