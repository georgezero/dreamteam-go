# V0.10b: live shared-prefix decision experiment

Status: **declared** (0.10.0-dev.2). The protocol is committed before any
live request; results follow in a later commit. V0.10 is unreleased. The
independent evidence review is still PENDING.

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
