# V0.10a: inference repeatability and controlled comparison design

Status: **declared** (0.10.0-dev.1). The protocol ([PLAN.md](PLAN.md),
`protocol.json`), the prompt sample (`sample/`), the request schedule
(`schedule.json`) and the server record ([SERVER.md](SERVER.md),
`server-declared.json`) are committed before any request of this study.
Results follow in a later commit.

V0.10a is a methodology and measurement milestone. It changes no
simulator mechanics, agent, prompt, setting or scoring, and runs no
multi-day live benchmark.

## 1. Baseline

At clean `main` 7fa9557 (the V0.9.0 release, tag `v0.9.0`, object
a5d7e32), before any V0.10a change. Each aggregate is the SHA-256 of the
lines `<path> <sha256>\n`, in path order; a single file's own SHA-256 is
given instead.

| Set | Files | SHA-256 |
|---|---|---|
| Simulator: Go sources without tests and the version package; scenarios; go.mod | 129 | `cbbad4231e26de3a25e74e4e08ceb2476795af1d6e3b7ba3ec0e417ac76e8e68` |
| llm-agent 1 and 2: agents/llm/llm_agent.py | 1 | `5214e7bfb456075f38aca92a5c45a6db3141d4042fb1cd5822c8c1c65c286d87 (the file)` |
| llm-agent 3: agents/llm/llm_agent3.py | 1 | `ef2c17a06749d7c287cda918f77ff0e518d79475048e87fd3404848c2fb0015c (the file)` |
| llm-agent 4: agents/llm/llm_agent4.py | 1 | `4cfc6a6906556b0cda50e95fa39263089d56e848adcef9bd04342e7f6aff998f (the file)` |
| Model configurations: docs/benchmark/v05c-dev/llm-*.json | 4 | `b0a1ff334b58ba0c8769fecc23dcacaa0a1472e16020c0b3e89e3b4a3bc9ecd2` |
| Historical recordings, V0.5a-V0.9c (*-decisions.jsonl) | 114 | `336024d8ba6dd490c99b72c7e626b7f0792ad10d312767dba2deacf45b8f2107` |
| Evidence contracts: frozen-v1.json, frozen-parameter-v1.json, the analyzers and their checks | 25 | `15c04973c677828a9fc23dede93f90ffc59864ae6bdecb1edd489194e6774c7b` |
| Review package and page (docs/benchmark/v08b-dev/review/, v09a-dev/review/) | 10 | `b3826aac8cc147af67894a08cb173b0f231afdbaa84c683041164de7b157ddd6` |
| Benchmark suites (docs/benchmark/suite/) | 4 | `63c007ccedc023a501690aa14ca76d400a18be68024cfa658ec5804bd7265c65` |
| Golden references, without MIGRATIONS.json | 12 | `d84f80718f3ab0463cc3768cf89b5ba09885b654e674a92a87c82c6998a2b25f` |
| All development evidence (docs/benchmark/v0*) | 479 | `e109a4de7e4de385c412695b6c9d7827fd32a9bc82400976d5adbe188b4b732f` |
| Session fixtures (testdata/sessions/*.json) | 108 | `7f50a0ac535eb9b5f91667b2876bbae698265bde1a25bef10c04e976e780ded6` |
| Prompt version | | `llm-agent-prompt/v2`, memory `llm-agent-memory/v1` |
| Built-in system prompt | | `0efc36e21b592abc4acb4c8b78ee7634b39755580a0f5ac348a5be1f8a80c4e1` |
| Preview treatment | | `llm-agent-preview/v1`, `f8b7d787b59c36373d4a18135c1503ccf64e3dd956c749be42fdfb0dbe5c54fa` |

**After the V0.10a changes,** every set is unchanged except:
* the simulator aggregate, through the replay registration of 0.9.0 only
  (`internal/session/record.go`, `record_v1.go`; batch mode `v0.9.0`);
* the session fixtures: `mini-seed52-v090.actions.json` and
  `mini-seed52-v090.session.json` added, made from a clean 7fa9557
  worktree.

The agents, prompts, model configurations, recordings, evidence contracts,
suites and golden references are unchanged.
