# V0.10c: validation feedback against preview information

Status: **declared** (0.10.0-dev.3). The protocol is committed before any
live request; results follow in a later commit. V0.10 is unreleased. The
independent evidence review is still PENDING.

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
