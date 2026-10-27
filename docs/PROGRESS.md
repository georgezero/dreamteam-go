# DreamTeam — Progress

One section per roadmap step (ROADMAP §2), newest first. Each records what
was built, what the player can see, how it was tested, and known
limitations. The version shown in the TUI top bar (`DREAMTEAM v0.1.0-dev.N`)
and by `dreamteam version` names the latest step a build contains.

---

## V0.5b — memory, action guidance and cost reporting (0.5.0-dev.2)

Owner decision 2026-10-27: V0.5a accepted at 3104e67. V0.5b improves the
LLM agent without changing simulator behaviour, the protocol or scoring.
No multi-model comparison (V0.5c).

**Built** (`agents/llm/llm_agent.py`, llm-agent 2; spec
`docs/LLM_AGENT.md`; agent, tests and smoke-test configuration declared in
e30aacf):
* **Bounded memory** (`llm-agent-memory/v1`, §6):
  * what it holds:
    - the agent's recent decisions in full, the earlier ones as totals;
    - the `validate` refusals it met, with their messages;
    - each decision's applied outcome, from the next observation's action
      log;
    - metric rows at recent decision points;
    - the pending changes the observation and catalog show;
  * derived only from public observations, the catalog and the agent's own
    records;
  * bounded by decisions, rows and characters (default 6,000);
  * in the process only, so empty at the start of every run.
* **Targeted prompt** (`llm-agent-prompt/v2`, §4):
  * contents:
    - the time, operating hours and scheduled close;
    - active problems;
    - the memory;
    - a summarized observation;
    - for every action type, its fields (its JSON Schema in one line),
      the catalog's rules, the earliest and latest dates and lead times
      from its targets (shown as days from today), and the targets;
  * where no date is given, the prompt says so;
  * every cut is named, and the model may ask once for details: full
    catalog entries, or left-out paths from the compact or full
    observation;
  * `observation: compact` and `catalog: full` restore V0.5a's content;
  * on V0.5a's seven recorded observations: 59% fewer characters and 56%
    fewer tokens (the server's counts: 21,850–23,016 to 9,465–9,936).
* **Mixed actions** (§7):
  * the investigation: `mixed_batch` is a deliberate session invariant
    (atomic staffing batches versus sequential V0.1 batches,
    V0.2_DESIGN §5.7), not a transport, validation or implementation gap;
  * kept unchanged, and documented in LLM_AGENT §7 and AGENT_PROTOCOL §3;
  * the agent now splits a reply into an operations batch and a staffing
    batch, validates each and submits both at one decision point (the
    decision's documented list of batches);
  * the two are not one atomic unit.
* **Usage and cost** (§9; `summary.json`, `usage.txt`):
  * the model;
  * requests, retries and details requests;
  * tokens, as reported, estimated or unknown;
  * mean, median and maximum latency, and total inference time;
  * valid, refused and fallback decisions;
  * an estimated cost from configured prices per million tokens, labelled
    as an estimate;
  * no endpoint and no secrets.
* **Replay** (§8):
  * a version 2 recording's memory and prompt are rebuilt at every decision
    point and checked against their recorded SHA-256;
  * V0.5a recordings replay as before;
  * other memory or prompt versions are refused with an explicit
    `version incompatible` error.
* **Version:** 0.5.0-dev.2; 0.5.0-dev.1 registered for replay
  (`mini-seed52-v051`, batch mode `v0.5_dev1`).
* **CI:** replays both smoke tests' recordings on every platform and
  compares their manifests.

**Mock acceptance** (`internal/bench/llm_agent_test.go`, twelve mock
behaviours; the owner's ten tests):
1. **The memory updates:** each decision and its applied outcome are carried
   to the next point; refusals and pending allocation changes are shown.
2. **It stays bounded:** a 90-day mock run keeps it at most 1,479 of 1,500
   characters, and a 120-point unit test within 6,000.
3. **It resets:** it is empty at the start of every run.
4. **Pending changes come from public data only:** pending versions and
   staff dates come from the catalog and observation, and fields the
   observation does not have never appear.
5. **Invalid actions still fail safely:** each failure gets exactly one retry,
   then no action, with nothing applied.
6. **Constraints appear:** dates, lead times, operating hours, caps and batch
   rules are in the prompt (unit tests on the recorded observation).
7. **Filtering keeps every action:** every type and every field is in the
   prompt, details return full catalog entries and a path from the full
   observation, and `catalog: full` includes the JSON.
8. **Usage totals are correct:** they match the mock's counts.
9. **Cost estimates are correct:** they match the configured prices.
10. **Replay stays byte-identical:**
    * decisions, sessions, results, digests and the memory at every point
      match the live runs;
    * tampered digests and a tampered memory hash are refused;
    * another memory format is refused.

Mixed operations and staffing actions were applied as two batches at one
decision point. Python unit tests: 26.

**Live smoke test** (`docs/benchmark/v05b-dev/`; sustainable, seed 40701,
14 days, noop and the agent with V0.5a's model on a local server):
* both runs completed, and `verify` passed 24 of 24;
* 14 decisions:
  * 6 with actions: 6 batches, 17 actions, all accepted;
  * 7 no action;
  * 1 fallback (a reply cut off at the output limit, then a refused
    retry);
* `validate` caught every refused proposal. The model re-proposed an
  `after_close` extension at points 3, 6 and 7, although its memory held
  that refusal;
* no staffing action and no details request;
* the memory grew to 5,976 characters and stayed under its 6,000 bound;
* 20 requests: 236,061 prompt and 8,797 completion tokens, all
  reported (first attempts 9,613–12,506 prompt tokens, memory included);
  mean latency 4.3 s, median 4.0 s.

**Replay:**
* byte-identical decisions, sessions, results and final digests to the
  live run, and an identical memory at all 14 points;
* two replays are identical, and reproduce passed 19 of 19;
* the manifest is byte-identical on linux/arm64 and linux/amd64 (Python
  3.11 and 3.13);
* the V0.5a recording still replays byte-identically to its live run.

**Validation:**
* the full uncached suite passes on linux/arm64 (about 3,720 s) and
  linux/amd64 (532 s);
* race passes on bench, agent, session and cmd/dreamteam: 0 data races;
* vet (all tags) and gofmt are clean;
* 36 of 36 session fixtures verify, including the new 0.5.0-dev.1 fixture;
* golden references are identical;
* the Python unit tests pass on Python 3.11 and 3.13;
* the CI benchmark passes verify (86/86) and reproduce (59/59); its 48
  files equal 0.5.0-dev.1's apart from the engine version (and the session
  checksums and manifest hashes that cover it);
* cross-builds succeed for five targets.

**Known limitations** (`docs/LLM_AGENT.md` §12):
* the memory is bounded, informs rather than enforces, and has no
  planning behind it;
* outcomes are read from the action log;
* the operations and staffing batches are not one atomic unit;
* summaries can omit what matters (details on request);
* live runs are not reproducible;
* cost is an estimate from configured prices;
* one model, one seed, 14 days.

---

## V0.5a — the external LLM agent (0.5.0-dev.1)

Owner decision 2026-10-27: V0.4.0 accepted at ced3acb; V0.5 re-slotted
from growth to LLM agents (growth deferred, V0.6 the full agent
benchmark). V0.5a: one minimal model-agnostic LLM agent that takes part
through the existing policy protocol as a separate process.

**Built.**
* **The agent** (`agents/llm/llm_agent.py`, llm-agent v1; spec
  `docs/LLM_AGENT.md`). Standard library Python, an OpenAI-compatible chat
  completions client.
  * **Settings** (`dreamteam.llm-agent-settings/v1`): the model and base
    URL, given directly or by environment variable; the API key from the
    environment only; timeout, temperature, output tokens, retries,
    decision budget, response format, prompt bound, optional system
    prompt file.
  * **Each decision point:** `digests`, the compact observation, the
    catalog; a bounded prompt (`llm-agent-prompt/v1`); a strict
    `{"reason", "actions"}` reply; catalog types checked; the batch
    validated through the protocol and submitted only if valid; bounded
    retries with the error fed back; no action on failure.
  * **Stateless:** no memory, no planning, one batch at most.
* **Evidence:** `agent.json`, `model.jsonl` (every request payload, raw
  response, usage, latency, error), `decisions.jsonl` and `summary.json`,
  without secrets.
* **Replay mode** (`--replay FILE…`): recorded decisions submitted without
  a model, each matched to its decision point and all five digests;
  several recordings are matched by their first digests.
* **Harness:** an external policy's `evidence/` files are kept in the
  run's `policy-evidence/` (≤1,000 files, ≤64 MiB, regular files only),
  listed in the manifest as nondeterministic and not hashed. No other
  harness, protocol, scoring or simulator change; manifests of configurations
  without evidence are unchanged.
* **Version:** 0.5.0-dev.1; 0.4.0 registered for replay
  (`mini-seed52-v040`, batch mode `v0.4.0`).
* **Roadmap re-slot**; CI runs the agent's unit tests and replays the smoke
  test's recorded decisions on every platform.

**Mock acceptance** (`internal/bench/llm_agent_test.go`, a deterministic
mock OpenAI-compatible server; ten behaviours; `TestLLMAgentMock`,
`TestLLMAgentReplay`):
* every run completes;
* the agent asks only for hello, digests, observe, actions and validate,
  and the gate denies nothing;
* a no-action reply is no change, and a valid raise is applied, from a
  plain or fenced reply and on a retry;
* prose, an unknown action, a refused allocation, an extra field, a slow
  server and HTTP 500 each get exactly one retry, then no action, with
  nothing applied;
* the evidence is kept and listed;
* the stand-in API key reaches the server and appears in no result file;
* a replay calls no model and reproduces the live runs' decisions,
  sessions, results and digests; another run's or a tampered recording
  is refused (`policy_crash`).

The Python unit tests (11) cover the settings, parsing, endpoint, prompt
bound and replay matching.

**Live smoke test** (`docs/benchmark/v05a-dev/`; config, settings and agent
committed first in b5ac92f; sustainable, seed 40601, 7 days, noop and the
agent with `Qwen3.6-35B-A3B-oQ4e-mtp` on a local server):
* both runs completed, and `verify` passed 24 of 24;
* 7 decisions: 5 no action, 2 batches (3 shift extensions), all valid when
  submitted;
* 2 proposals refused by `validate` (`past_date`, `after_close`) were each
  corrected on the one retry;
* 9 model calls: 202,084 prompt and 1,225 completion tokens, mean latency
  5.1 s.

**Replay:** byte-identical decisions, sessions, results and final digests
to the live run; reproduce 19 of 19; manifest byte-identical on
linux/arm64 and linux/amd64 (Python 3.11 and 3.13).

**Validation:**
* the full uncached suite passes on linux/arm64 (3,632 s) and linux/amd64
  (532 s);
* race passes on the affected packages (bench, agent, session,
  cmd/dreamteam): 0 data races;
* vet (all tags) and gofmt are clean;
* 35 of 35 session fixtures replay, including the new 0.4.0 fixture;
* golden references are identical;
* the Python agent's unit tests pass on Python 3.11 and 3.13, and the
  Python client and reference policy are unchanged;
* the CI benchmark passes verify (86/86) and reproduce (59/59); its 48
  files equal v0.4.0's apart from the engine version;
* cross-builds succeed for five targets.

**Known limitations** (`docs/LLM_AGENT.md` §9): stateless; one batch per
decision point (V0.1 and staffing actions cannot share one); live runs
are not reproducible; about 22,000 prompt tokens per decision; a trusted
process; no cost accounting beyond tokens and latency; one model, one
seed, 7 days: not a measure of decision quality.

---

## V0.4.0 — release

The V0.4 release on the owner's decision (2026-10-26): V0.4b accepted at
1a88857, no further implementation slice. The agent evaluation harness
(V0.4a and V0.4b). Release preparation only:
* the version (0.4.0) and the replay registration of 0.4.0-dev.2
  (`mini-seed52-v042`, batch mode `v0.4_dev2`);
* README, release notes, roadmap, progress and the benchmark
  specification's status and limitations.

No change to mechanics, randomness, scenarios, actions, accounting,
evaluation, replay, the harness, the policies or the policy protocol.
`python-ref` v1 is unchanged. The 90-day development matrix is not rerun:
the release changes no code it exercises, and CI runs the benchmark
configuration (verify and reproduce) on every platform.

The documented limitations are accepted as nonblocking (`docs/BENCHMARK.md`
§12): trusted, unsandboxed external policies; seed withholding through
the protocol only; wall-clock timeouts; children of a failed policy;
`python-ref` v1 never acting in the development matrix; `greedy` and
`rules` trading about $33,000 of cost for faster reporting; T90 NO GO on
all 20 development runs. Development comparisons are not confirmatory
performance claims. Next: the owner's V0.5 decision.

---

## V0.4b — external agent benchmarking (0.4.0-dev.2)

Owner decision 2026-10-26: V0.4a is accepted at 2ab3564. V0.4b scope:
1. an external policy process adapter;
2. benchmark seeds hidden from agents;
3. a fresh 90-day development comparison;
4. a reproduce command, if straightforward.

No new built-in baseline.

**Built.**

* **External policies** (`internal/bench/external.go`,
  `docs/BENCHMARK.md` §4).
  * A configuration declares them under `external`: ID, version,
    description, command, timeout. `./` and `../` command files are
    resolved from the run directory, and their SHA-256 goes in the
    manifest.
  * One fresh process per run, in an empty working directory outside the
    results, removed afterwards.
  * The policy protocol (`dreamteam.benchmark-policy/v1`), JSON lines
    over stdio:
    * the harness sends `decide` and `end` turns;
    * the policy sends agent/v1 requests, each answered in order through
      the same gate as a built-in policy's;
    * the policy ends each turn with `{"decision":{"batches":[…]}}`.
  * A per-decision timeout, which also covers the exit after `end`.
    Standard error is captured separately (`stderr.txt`, kept and
    listed, not hashed).
  * Failures are classified: `policy_crash` (could not start, ended
    early, exit status not 0), `policy_timeout`, and `policy_error` (a
    decision that cannot be read). The process is killed and reaped, and
    the other runs go on. Requests the protocol refuses are counted as
    request errors (a new count, for built-in policies too).
  * No HTTP or sockets. apply, advance, save and t90 stay the harness's.
* **Seed privacy.**
  * `agent.Handler.Benchmark` is set by the harness. `hello` then omits
    the seed and carries `benchmark: {seedWithheld, permittedOps,
    decision}`.
  * Every other session's `hello` is byte-for-byte unchanged (the Python
    client's run digest is unchanged).
  * The seed stays in the harness's own evidence.
  * The trust boundary (BENCHMARK.md §11): this stops disclosure through
    the protocol only. External policies are trusted development
    participants, not sandboxed. No OS-level isolation is claimed.
* **`clients/python/reference_policy.py`** (`python-ref` version 1):
  * standard library only and protocol only;
  * a weekly review (decision points 0, 7, 14, …) that raises one site
    reading allocation by 5 points, validated first, when the oldest
    unfinished study is over 24 h old;
  * a refused proposal is passed over, a protocol error means no change
    that turn, and the process exits 0 after `end`.

  It was fixed before any run and committed with the configuration.
* **`dreamteam benchmark reproduce RESULTS [-out DIR]`**
  (`internal/bench/reproduce.go`). It reruns the recorded `config.json`
  and compares engine, policies, policy files, metric definitions, run
  statuses, every hashed file and `manifest.json`. Timing and standard
  error are not compared. Exit codes: 0 identical, 3 different, 1 not
  rerunnable (another engine).
* **Report**: external policies are marked; request errors are added to
  the counts table.
* **Replay registration**: 0.4.0-dev.1 (batch mode `v0.4_dev1`, fixture
  `mini-seed52-v041` recorded by the 2ab3564 binary), with its
  MIGRATIONS entry.
* **CI**: the benchmark step includes `python-ref` and runs `reproduce`.
  Its manifest, the external policy's runs included, is compared across
  linux/amd64, linux/arm64 and darwin/arm64.

**Development comparison** (`docs/benchmark/v04b-dev/`):
* sustainable, seeds 40501–40505, noop, greedy, rules and python-ref,
  90 days, T90;
* configuration and policy committed first (0853bcc);
* 20 of 20 completed and replayed, decisions matched, identical initial
  conditions per seed;
* no failures, refusals, protocol errors, denied requests or request
  errors.

| Machine | Result |
|---|---|
| Pi (arm64, Python 3.11) | 2,094 s; `manifest.json` `dd957074…`; verify -t90 226 of 226 |
| horo (amd64, Python 3.13) | the same `manifest.json`; verify -t90 226 of 226; `reproduce` 147 of 147 |

Findings, not claims:
* **python-ref never acted.** Its weekly reviews fall on Tuesdays, while
  the backlog passes 24 h only on Sunday and Monday mornings. Its runs
  equal noop's ("same decisions, identical outcome"). The rule is not
  changed.
* **Greedy and rules made the same 12 raises per seed in the opposite
  order.** Rules never lowered. At 90 days their outcomes differ
  slightly on every seed (TAT p90 by 1–20 min, breaches by 0–13; profit
  equal within $1). The 90-day horizon does not make them meaningfully
  different.
* **Against noop, 5 of 5 seeds:** much lower TAT and fewer breaches, net
  revenue +$0.5k to +$5.3k, expense about +$33k, profit −$27k to −$33k.
* **T90 at day 90 (not projected) is NO GO for all 20 runs.** noop and
  python-ref fail service reliability; greedy and rules fail solvency.

**Compatibility.**
* The V0.4a configuration reruns on 0.4.0-dev.2. All 15 runs have the
  same initial and final digests, metrics, decisions and T90 verdicts
  as V0.4a's, and verify passes 156 of 156.
* The V0.3 Python client passes against 0.4.0-dev.2 with the same run
  digest (`948f6ad0…`). Its transcript is V0.4a's apart from the engine
  version.

**Tests.** `internal/bench`:
* `TestExternalPolicy`: a real subprocess (the test binary as a helper).
  It checks counts (denied 4, request errors 1), byte-identical reruns,
  the manifest's policy entry, `stderr.txt` listed as nondeterministic,
  verify, and that no working directory is left.
* `TestExternalPolicyFailures`: crash, hang (timeout), a malformed
  decision, a process that does not exit, a non-zero exit, and a missing
  policy file refused before running. Each is classified at the right
  decision point, the files verify, and the noop run beside them is
  byte-identical to its run alone.
* `TestSeedIsWithheld`: a built-in probe and an external process call
  every permitted operation, failing calls included, on two scenarios.
  No response carries the seed's digits; `hello` says it is withheld;
  the process's arguments, environment and working directory do not
  contain it; `run.json` keeps it; a non-benchmark `hello` still
  reports it.
* `TestPythonReferencePolicy`: completes, verifies, two runs identical,
  its file hash recorded, standard error empty.

`cmd/dreamteam`: `reproduce` matches a fresh rerun, and a manifest that
records another hash fails (exit 3). `internal/session`: the
0.4.0-dev.1 fixture replays and continues.

Mutation checks: a `hello` that reports the seed under the harness, and
an external policy whose requests bypass the gate, each fail
`TestSeedIsWithheld` and `TestExternalPolicy`.

**Limitations** (BENCHMARK.md §12):
* external policies are trusted, not isolated: a policy could read
  local files or search for the seed from the digests;
* timeouts are wall-clock and include the harness's time;
* only the started process is killed on failure;
* no feedback within a decision point;
* all acting policies use one lever, and python-ref's weekly timing
  never met its trigger here.

---

## V0.4a — the agent evaluation harness (0.4.0-dev.1)

Owner decision 2026-10-26: V0.3.0 is accepted and closed. V0.4 is the
agent evaluation harness; V0.6 keeps the full benchmark. Scope: compare
policies fairly and reproducibly through the released agent protocol,
with no new scoring rule.

**Built.**

* **`internal/bench`, the harness** (`docs/BENCHMARK.md`).
  * A versioned configuration (`dreamteam.benchmark/v1`), validated
    before anything runs. It needs explicit seeds and a seed plan. It
    refuses reference seeds 1001–1020 and sealed holdout seeds (the
    holdout guard gives an error only).
  * The **gate**: a policy reaches the session only through the
    protocol's read and check operations (hello, observe, sections,
    actions, validate, preview, digests). apply, advance, save and t90
    are the harness's, and the gate answers them `not_permitted` without
    touching the session.
  * **One fresh session and policy per run.** For each run the harness
    applies the policy's batches, advances at the configured cadence,
    saves, and asks for T90 where it applies. It then replays the saved
    session and writes its canonical result: `dreamteam replay
    -format json`, byte for byte.
  * **Metrics** are 32 values read by path from that result, with no
    conversion. Failures are classified (`policy_error`,
    `protocol_error`, `evaluation_error`, `replay_error`,
    `harness_error`). Refusals, protocol errors and denied requests are
    counted.
  * **Results.** A results directory with a manifest of SHA-256 hashes,
    in which only `timing.json` is nondeterministic. A report (text and
    JSON) with five parts:
    1. runs;
    2. initial conditions;
    3. per-policy minimum, median and maximum;
    4. paired differences with the baseline, separating same decisions
       and identical outcomes from different decisions, plus counts of
       higher, lower and equal;
    5. failures and replay.
  * **Verification**: hashes and no unlisted files, replay, the result
    rebuilt, metrics, the decisions against the session's batches, the
    report rebuilt, and optionally T90 recomputed.
* **CLI**: `dreamteam benchmark run CONFIG [-out DIR]`, `report RESULTS
  [-format text|json]`, `verify RESULTS [-t90]`.
* **Policies**: `noop`, `greedy` (the V0.3 demonstration, unchanged) and
  `rules` version 1. Rules keeps the reading allocation in step with the
  backlog, with hysteresis: it raises past 24 h oldest unread and lowers
  after three decision points at 8 h or less. Each change is validated
  and previewed first. The thresholds were fixed before any run.
* **`internal/agent`**: `Client.H` is an `Exchanger` interface (a handler
  or the gate), with optional string id prefixes and `Requests()`.
  `Handler.Dir` resolves a relative save path, so transcripts carry no
  absolute path. The protocol's behaviour is unchanged.
* **Replay registration.** The v0.3.0 release is registered for replay:
  batch mode `v0.3.0`, fixture `mini-seed52-v030` recorded by the
  v0.3.0 binary built from the tag. The MIGRATIONS entry is added.
* **Holdout containment.** The containment test reviews the new
  `GuardManifest` call (bench config validation).

**Development run** (`docs/benchmark/v04a-smoke/`):
* sustainable, seeds 40401–40405, three policies, 30 days, T90;
* the seeds were committed alone (b9cc0b6) before any run on them;
* 15 of 15 runs completed; every session replayed and matched its
  decisions; initial conditions are identical per seed;
* the Pi (arm64) and horo (amd64) gave the same `manifest.json`
  (`81ecfaa1…`), so every deterministic file is byte-identical;
* `verify -t90` passed all 171 checks.

Findings, not claims:
* greedy and rules made the same four allocation changes per seed in a
  different order, so with the 14-day lead time their 30-day outcomes are
  nearly identical;
* both finalized more reports, with lower TAT and fewer breaches than
  noop, on 5 of 5 seeds, with mixed profit;
* T90 (v0.2-eval-4, projected) gives CONDITIONAL GO on 2 seeds and NO GO
  on 3.

**Tests.** `internal/bench` covers:
* configuration validation;
* the gate;
* byte-identical reruns;
* initial conditions;
* isolation: a run's files do not depend on the other policies or their
  order;
* the policy's view: the transcript shows only permitted ops;
* policy errors and panics: classified, files verify, the other runs go
  on;
* metrics against the result;
* rules' validate→preview→apply order;
* tamper detection.

`cmd/dreamteam` checks:
* T90 equals `dreamteam t90` (v0.2-eval-4, projected);
* `result.json` equals `dreamteam replay -format json`;
* `verify -t90`, report JSON, write-once output and a tampered file;
* invalid configurations (exit 2, nothing written) and usage errors.

Mutation check: an open gate fails three tests.

**Limitations** (BENCHMARK.md, known limitations):
* built-in policies only; the external-process adapter is designed but
  not built;
* `hello` shows the seed, which an external agent with the binary could
  use to simulate ahead;
* there is no feedback between batches within a decision point;
* a 30-day horizon sees only reading changes set in the first 16 days;
* T90 takes about 40 s per run on a Pi;
* rules is not yet distinct from greedy on this scenario.

---

## V0.3.0 — release

The V0.3 release on the owner's decision: the agent interface (V0.3a and
V0.3b). Release preparation only:
* the version (0.3.0) and the replay registration of 0.3.0-dev.2
  (`mini-seed52-v032`);
* README, release notes, roadmap and protocol status;
* a test fix: the session test of an unregistered engine named 0.3.0,
  which is now this build, so it names 9.9.9 instead.

No change to mechanics, randomness, scenarios, actions, accounting,
evaluation or replay. The documented limitations are accepted as
nonblocking: `t90`'s run time on a Pi, staffing-only previews, and the
multi-seed bound's last digit across processor families. Next: an agent
evaluation harness.

---

## V0.3b — external agent usability (0.3.0-dev.2)

The second V0.3 slice, on the owner's decision after V0.3a: the protocol
made practical for agents outside the process. Not released.
* **`t90`:** the T90 evaluation of the session as it stands, by the
  policy in force (session.T90), the same as `dreamteam t90`. It is only
  ever explicit (never in `observe`) and read-only: no digest changes
  and no randomness is drawn. Stable refusals for unsupported scenarios,
  earlier action rules, not started, sealed seeds.
* **Observation projections:** `observe` takes `sections` and `compact`,
  and the new `sections` op lists what can be selected. A projection is
  cut from the full observation's own JSON (keys left out, order kept),
  so it can show nothing new. `compact` leaves out four declared record
  listings and counts them (≈116 KB → ≈29 KB). Without options, observe
  is byte-for-byte as before.
* **The reference Python client** (`clients/python/`, standard library
  only): a complete session through a real subprocess, checked by
  `dreamteam verify` and `dreamteam t90`; run by the Go suite and in CI,
  where its transcript is compared across platforms.
* **Cross-platform statistics.** T90 reports differed between arm64 and
  amd64 in the last digit of some reported statistics (design effects,
  Wilson bounds): arm64 fuses multiply-add, amd64 does not. The
  evaluation now rounds every product before a sum (`float64(x*y)`, as
  the Go spec provides). amd64's values are unchanged; arm64's now equal
  them, so arm64 T90 output differs in those digits from V0.2.0's. No
  status, verdict or rule changed.
* **Compatibility:** the V0.3a transcript replays byte for byte except
  hello's engine and ops. Registers 0.3.0-dev.1 for replay
  (`mini-seed52-v031`).
* **CI had been failing since V0.2 Step 0C, on two defects of its own,
  found here and fixed:**
  * The golden step compared without the migration manifest that the
    documented check uses (`testdata/golden/README.md`), so the
    deliberate state-schema change of V0.2 Slice 1 read as a mismatch
    on arm64 and macOS. It now passes `-golden-migrations`.
  * `TestV4Neutrality` pinned an arm64 digest that never held on amd64.
    It now pins one digest per processor family. The remaining
    difference is the multi-seed joint gate's exact bound, through the
    Go math library, in the last digit; every status is the same.
  
  The local gates (the full suite on arm64, race on amd64, goldens with
  the manifest, replays) had passed throughout.

---

## V0.3a — the deterministic agent interface (0.3.0-dev.1)

The first V0.3 slice: `dreamteam agent --stdio`, a JSON protocol over
the session API (`docs/AGENT_PROTOCOL.md`), on the owner's decision after
the V0.2.0 release. Not released.
* **Transport-independent.** `internal/agent.Handler.Handle` turns one
  request into one response, as bytes; `ServeLines` carries them one per
  line over stdin and stdout. Only responses go to stdout.
* **Operations:** `hello`, `observe` (the game's observation), `actions`
  (a JSON Schema, constraints and current targets for every action type,
  read from the offer table and the session's views), `validate`,
  `preview` (the staffing preview), `apply` (atomic: a batch is applied
  only if every action is valid, so a refused batch changes no digest),
  `advance` (minutes, to a time, to the next visible event, to the end
  of the day: the session's own semantics), `save`, `digests`.
* **Stable error codes**; invalid JSON, unknown ops and bad arguments
  never stop the process or change the session; ids are echoed.
* **A scripted runner** (`dreamteam agent -policy noop|greedy`) drives
  the same protocol in process, from protocol JSON only.
* **No simulation change.** Registers 0.2.0 (the release) for replay
  (`mini-seed52-v020`).
* **A V0.2 defect fixed: the T90 evaluation of a continued session.**
  V0.2's T90 view refused any session continued from an earlier build
  ("defined for the sustainable scenario only"), which under this build
  would have meant every sustainable session saved by the V0.2.0
  release. A continued session is now evaluated with its earlier
  decisions when they replay under this build's action rules (every
  build from 0.2.0-dev.8 on), and refused with an accurate message
  otherwise. The evaluation of a V0.2.0-saved session equals that of the
  same plan played here (`sustainable-seed906-v020`,
  `TestT90EarlierEngineSession`).
* **Tested:** 30-day noop and greedy sessions valid line by line,
  byte-identical when repeated, saved and replayed; every action type
  discovered and applied from the catalog's targets; invalid batches,
  validate and preview leave every digest unchanged; the agent and the
  session API reach the same states, the observation byte for byte;
  stdout carries only responses, one per request, in order.

---

## V0.2.0 — release

The V0.2 practice-design release, on the owner's decision after the
acceptance pass and the completion of F1 and F2. Release preparation
only: the version (0.2.0), the replay registration of 0.2.0-dev.27
(`mini-seed52-dev27`), README, release notes and roadmap. No change to
mechanics, randomness, scenarios, actions, accounting, evaluation or
replay. The documented P2 limitations are accepted as nonblocking
(`README.md`). Next: V0.3, the agent interface.

---

## V0.2 completion — the T90 evaluation view and the bundled sustainable scenario (0.2.0-dev.27)

The V0.2 acceptance pass (16e5751) found two promised deliverables not
built; the owner decided to build both before the release.
* **F2 — `sustainable` is a bundled scenario.** `scenarios/sustainable.json`
  is the Step 0C shared-staffing control at ×1.30 demand (V0.2_DESIGN
  §5.6, `W130-SHR-MRft`): the established practice with that experiment's
  overlay, byte for byte. By the owner's decision its internal id stays
  `established`, so it keeps the fingerprint the evaluation policy in
  force pins (`e8718dbd…`); it is listed and chosen as `sustainable`.
* **F1 — the T90 evaluation is visible.** `dreamteam t90 FILE` and the
  summary screen's third view (`tab`) evaluate the player's plan under
  v0.2-eval-4, embedded byte for byte: the plan's decisions before T90
  and the no-action baseline on the same seed, both run with no further
  decision to the observation end, judged by `eval.Evaluate`. They show
  the metric vector with the baseline and the difference, every service
  gate (pass, fail or insufficient, with its counts and limit), the plan
  gates and the verdict, and say what is projected (days not yet played),
  what is the fixed-policy tail and what is incomplete. No rule is added;
  "scoring comes later" is gone. Defined for the sustainable scenario
  only; holdout seeds are refused, as by every harness.
* **No engine change:** the evaluation recorder moved unchanged from a
  test file into `internal/sim`; every golden and replay fixture is
  unchanged.
* **Registers 0.2.0-dev.26** for replay (`mini-seed52-dev26`).
* **Tested:** the game's report equals the canonical development
  runner's byte for byte (`TestT90MatchesCanonicalRunner`); pass, fail and
  insufficient gates each displayed as decided; projected vs measured;
  the session unchanged by evaluating; the CLI, the session and the TUI
  view identical; other scenarios refused; the sustainable scenario
  deterministic and replayed, its fingerprint the policy's.

---

## V0.2 v5 protocol lock (0.2.0-dev.26, lock commit; NOT RUN)

The prospective confirmatory protocol for v0.2-eval-5 is locked, on the
owner's conditional GO after the 5C.4o release-gate audit. The lock
record is `docs/calibration/clinical-v5-lock/LOCK.md`, with
`lock.sha256` beside it.
* **Not run, not frozen, not activated.** The reserved run needs the
  owner's separate authorization.
* **The lock changes only what protocol §8 provides:** `familyAllowed`
  admits PCG(20261201, ·); the two refusal assertions become admission
  assertions, constructing nothing; the protocol's status statements
  read LOCKED; its digest is re-pinned (`12d1eda4…0a0d`).
* **Pinned:** parent `7bf1ed6`; `sourceSHA256` `5cd6a438…588b`; the
  test binary built with `-trimpath`, `CGO_ENABLED=0`, go1.27.1
  linux/arm64, `5edee74d…130d` (reproducible across checkouts); the
  candidate, manifest, protocol, policy in force and simulator profile;
  the command, 3,904 lines, T1–T8, the T5b tolerances, the trusted
  in-process scope with the study-swap limitation, one execution, the
  failure handling and no post-run tuning.
* **Verified:** every committed test passes at the lock tree; the
  pinned binary, run on the development family only, records the pinned
  source digest, inputs and engine and reads `development` /
  `not certified`. Nothing ran on the reserved family.

---

## V0.2 deterministic adjudication output, before the protocol lock (0.2.0-dev.26)

The owner accepted O1 and O2 as closed on the targeted remediation
(conditional GO for the lock) and required one change first: no output
byte may depend on Go's map iteration order.
* **The change:** T5a sums its group predictions, and lists its group
  means, in group order instead of map order. No formula, threshold,
  policy, manifest, seed family, simulation behaviour or certification
  rule changed. NM2.1's 200-seed prediction now reads 0.34404 on every
  adjudication (it varied in the 17th digit).
* **Registers 0.2.0-dev.25** for replay (`mini-seed52-dev25`).
* **Tested:** `TestAdjudicationDeterministic` (fails with map order
  restored); the 200-seed development lines re-read three times and two
  trusted 80-seed development executions (NM2, NM2.1), each
  byte-identical, `development` / `not certified`; the full uncached
  suite, vet, gofmt and replay through dev.25.
* **Not locked, not run, not frozen.** The lock is a separate commit.

---

## V0.2 release-gate blocker fixes — audit 5C.4o O1 and O2 (0.2.0-dev.25)

The focused release-gate audit 5C.4o (36d34f2, committed verbatim before
any change) found M1, M3 and M6 closed and returned NO GO on two
certification-critical findings. Under the owner's stopping rule only
those were fixed and only their checks rerun. The maintainer's response
is `docs/audits/slice-5c4o-release-gate-audit-response.md`.
* **Not locked, not run, not frozen.** The owner decides whether the
  fixes close the gate; no further broad audit is scheduled.
* **Unchanged:** the clinical policy, the manifest, the simulator, the
  thresholds, the trust boundary and the reserved seed protocol.
* **Registers 0.2.0-dev.24** for replay (`mini-seed52-dev24`).
* **Nothing reserved was run.**

**Fixed:**
* **O1:** each line's Gate B is reconciled with its own measurements by
  the candidate's rule (`gateBReconciles`): decided statuses need every
  pair matched and, for urgent, a difference on every pair; differences,
  safeguard flags, deteriorated seeds, increase and status must be the
  rule's computation on them, exactly. Near-margin replicates need a
  difference on every pair; no pair injects more than it served or left
  unimaged; a hidden pair construction error is refused. Every relation
  held on all 664 genuine development lines first.
* **O2:** the reader refuses anything after a line's JSON value.
* **O3 (documentation):** the "failing exit" and survivor-equivalence
  claims are narrowed in place; the lock recipe names the two refusal
  assertions the lock inverts.

**Tested:** the auditor's fixtures (their O1 cases and trailing file now
refused); each witness on an otherwise consistent line
(`TestReleaseGateMeasurements`, `TestReleaseGateReader`); the genuine
development lines re-read with no measurement problem and the same
verdicts; 49 mutations (48 caught; the survivor the documented T5a guard); the full uncached suite, vet, gofmt and
replay through dev.24.

---

## V0.2 Slice 5C.4n — final certification pipeline remediation (0.2.0-dev.24)

This slice remediates the independent 5C.4m audit (c456c94, committed
verbatim before any change) under the owner's decisions of 2026-10-20
(`docs/calibration/clinical-5c4n/`). It is the final planned remediation
before the focused 5C.4o release-gate audit. The maintainer's response
is `docs/audits/slice-5c4m-provenance-protocol-lock-audit-response.md`.
* **Not locked, not run, not frozen, not reviewed.**
* **Unchanged:** the clinical policy (`83d481b7…3948`), the simulator's
  behaviour, the severity thresholds, the scenario manifest
  (`250bc7fc…3ef4`) and the reserved seed protocol.
* **Registers 0.2.0-dev.23** for replay (`mini-seed52-dev23`).
* **Nothing reserved was run:** no strategy, reference or holdout seed,
  unseal record or PCG(20261201, ·) stream.

**Release blockers:**
* **M1, the trust boundary.** The reserved execution constructs,
  judges, adjudicates and certifies in one process and writes its files
  afterwards; certification is decided from how the lines were computed
  (trusted, family, manifest seeds), never from a file. A file read back
  is a `non-certifying analysis`: the auditor's fabricated, complete
  3,904-line file, on which every criterion PASSes, is `not certified`.
  The candidate, manifest, protocol, policy in force and simulator
  profile are verified against digests compiled into the build.
* **M2, measurement completeness.** Every line's measurements are
  checked for presence, cardinality, range, finiteness, seed identity
  and consistency before any criterion reads them. T2G (undetermined
  materiality), T5a (no prediction), T8 (matched counts by equality,
  exact per identity-witness kind) and T4 (exact seed sets) no longer
  read missing evidence favourably.
* **M3, the protocol.** It opens with the CERTIFICATION SCOPE: trusted
  in-process evaluation only; the study swap accepted, undetected, only
  there; authenticated order-to-study provenance for any future external
  records.
* **M4, errors.** Malformed settings, truncated or malformed input and
  unserializable results are errors; incomplete coverage fails the exit;
  no partial or development invocation is confirmatory.
* **M6, the validation helper.** It records the scenario's fingerprint
  and refuses records failing integrity; the regenerated access and
  follow-up evidence is byte-identical to 5C.4l's.
* **M5** (shared history with a larger candidate capacity): deferred;
  no manifest scenario has one.

**Corrections:** 20 distinct stable T1 groups, not "0 of 28"; the
near-margin results changed with 5C.4l's shared history (0.35 → 0.30
and 0.31 → 0.33 at +2.1 points; 0.56 → 0.65 at +2 points); "every seed
is valid" and "no bearing on the confirmatory run" corrected in place.

**Development checks** (PCG(20261021, ·), certified path, both
`development` / `not certified`): every line byte-identical to 5C.4l's;
no measurement problem; T8 passes after the per-kind correction found by
the first run; T5a passes on 200 seeds, T5b 0.33 at +2.1 points, 0 at +3.

**Tested:** the auditor's witnesses as regression tests and their own
eight files through the reader, 37 mutations (36 caught, one
equivalent, caught in combination), the full uncached suite, vet, gofmt
and the replay fixtures.

---

## V0.2 Slice 5C.4l — order provenance, inherited safety and reserved protocol closure (0.2.0-dev.23)

This slice remediates the independent 5C.4k audit (d06cdb9, committed
verbatim before any change), under the owner's decisions of 2026-10-20
(`docs/calibration/clinical-5c4l/`). The maintainer's response is
`docs/audits/slice-5c4k-clinical-safety-protocol-lock-audit-response.md`.
* **Not locked, not run, not frozen, not reviewed.** The focused 5C.4m
  review comes next.
* **The clinical policy is unchanged:** no threshold, window, limit,
  ceiling, follow-up or tolerance changed.
* **Registers 0.2.0-dev.22** for replay (`mini-seed52-dev22`).
* **v0.2-eval-4 is byte-identical.** The 5C.4d, e, f, h and j
  candidates keep their own rules, from their exact bytes.
* **Nothing reserved was run:** no strategy, reference or holdout seed,
  unseal record or PCG(20261201, ·) stream.

**Record integrity** (candidate SHA-256 `83d481b7…3948`):
* **K1, order provenance** (`orderProvenance: pairedOrigin`). Every
  order of a candidate's record, inherited or new, eligible or not, must
  reconcile with the paired plan's by origin: the same ID, receipt,
  wanted time, priority, modality and walk-in flag. Outcomes are never
  compared. A difference, a missing order or a candidate-only order
  makes the comparison INVALID. The audit's relink-and-retime bypass is
  INVALID (GO under the 5C.4j candidate).
* **K2, inherited safety** (`orderStudyIntegrity: contractInherited`).
  The inherited exemption is gone: every served order has exactly one
  post-takeover study, and order IDs are unique. Every inherited bypass
  of the audit is INVALID.
* **Not closed, reported for the owner:** an outcome-only study swap
  between two same-request orders around the horizon stays GO, because
  nothing immutable changes (`TestStudySwapLimitation`). Four options
  are set out; the reserved run is not affected (its records are built
  in process).
* **The fidelity model** gained a shared pre-takeover history, so
  paired records hold the same inherited orders, as the simulator's do.

**Adjudicator and protocol:**
* **K4:** no criterion passes with zero evidence; scenarios not run are
  incomplete coverage; the output records what was judged
  (`invocation`: `confirmatory` or `development`). The harness defaults
  to the manifest's seeds. The reserved execution is pinned in protocol
  §8 (expected 3,904 lines), not run.
* **K6:** the promised T1, T2 and T7 reports are emitted; W-ACCESS65
  checks the disclosed loss under a favourable Gate B; the follow-up's
  financial observation is measured: no financial measure or decision
  changed on the development seeds.
* **K5:** the false statement that no simulator partial window failed is
  corrected (seeds 901–903 failed locally; no seed or evaluation verdict
  changed), and local, seed-level and evaluation-level results are now
  reported apart.

**Development checks** (PCG(20261021, ·), seeds 901–910):
* every judged criterion passes; T5b 0.33 at +2.1 points, 0 at +3;
* provenance fails only on the identity witnesses' first seeds;
* 0 of 28 stable groups rejected; no stable seed fails the
  partial-window ceiling;
* the simulator records under the candidate are identical to 5C.4j's.

**Tested:** the audit's K1 and K2 witnesses as regression tests, the
5C.4i and 5C.4k witnesses, 23 new mutations (all caught) and the 5C.4j sets (21 of 21 applicable caught), the full uncached suite, vet,
gofmt and the replay fixtures.

**Next:** the owner's decision on the study swap, and the focused 5C.4m
review. Protocol lock only if clean; the reserved draw is authorized
separately after the owner inspects the lock commit.

---

## V0.2 Slice 5C.4j — clinical safety and reserved protocol remediation (0.2.0-dev.22)

This slice remediates the independent 5C.4i audit (4bead43, committed
verbatim before any change), under the owner's decisions of 2026-10-19
(`docs/calibration/clinical-5c4j/`). The maintainer's response is
`docs/audits/slice-5c4i-boundary-safety-lock-readiness-audit-response.md`.
* **Not locked, not run, not frozen, not reviewed.** The independent
  5C.4k review comes next.
* **Registers 0.2.0-dev.21** for replay (`mini-seed51-dev21`).
* **v0.2-eval-4 is byte-identical.** The 5C.4d, 5C.4e, 5C.4f and 5C.4h
  candidates keep their own rules, from their exact bytes.
* **Nothing reserved was run:** no strategy, reference or holdout seed,
  unseal record or PCG(20261201, ·) stream.

**Clinical policy** (candidate SHA-256 `9efa6e63…29e7c`):
* **I1, partial windows** (`partialWindows: rateCeiling`). Every
  nonempty partial window fails above a 5% observed severe rate,
  whatever its size. At or below 5% with fewer than 100 acquisitions it
  is NOT_JUDGED. 5/6, 5/10, 5/50, 5/99 and 1/6 were GO and are now
  NO GO on the full 40-seed evaluation; 5/100 and 0/6 stay GO.
* **I2, the order–study contract** (`orderStudyIntegrity: contract`).
  Unique study IDs; one order per study, with the same request, served
  and resolved at the acquisition; one study per served new order.
  Inherited work is exempt. The audit's three bypasses are now INVALID.
* **I3, a 45-day follow-up** (`followUpDays: 45`), observed by every
  recorder. Equality at the limit passes, a report due exactly at E and
  unfinished stays INSUFFICIENT, and nothing is fabricated.
* **I4, prefix-stable fidelity draws.** Observing longer changes no
  earlier outcome, in the fidelity model or in the simulator.

**Adjudicator:**
* **I5:** coverage is checked before judgment: exact groups, the
  remainder, replicates, seeds, validity and scanner errors.
* **I6:** raw T7 insufficiency; T2G per priority; witnesses judged by
  their targeted rule; W-ID-CHANGED moves the wanted time.
* **I7:** an exact binomial T5 prediction; the approved tolerances are
  pinned in the manifest (version 2).

**Development checks** (PCG(20261021, ·), seeds 901–910):
* **Results.** Every judged criterion passes. T5b at +2.1 points: 0.31
  (tolerance 0.40); at +3: 0.
* **False rejection.** 0 of 28 stable groups rejected; 0 of 1,120
  stable seeds failed the partial-window ceiling. Not material.
* **Simulator.** Routine reports not yet observable fall from 22 to 3.
  A recorder defect exposed by the contract was repaired: no-shows at
  the takeover with a rebooking queued.

**Tested:** the audit's witnesses as regression tests; 24 mutations, all
caught; the full uncached suite, vet, gofmt and the replay fixtures.

**Next:** the independent 5C.4k review. Protocol lock only if clean,
then one reserved run. Reserved calibration and the freeze stay NO GO.

---

## V0.2 Slice 5C.4h — reserved protocol closure and clinical boundary safety (0.2.0-dev.21)

This slice is the remediation of the independent 5C.4g audit (c55f3cb,
committed verbatim before any change), under the owner's decisions of
2026-10-19 (`docs/calibration/clinical-5c4h/`).
* **Not locked, not run, not frozen.**
* **Registers 0.2.0-dev.20** for replay (`mini-seed50-dev20`).
* **v0.2-eval-4 is byte-identical.** The 5C.4d, 5C.4e and 5C.4f
  candidates keep their own rules.
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream (the harness refuses PCG(20261201, ·)).

**Boundary safety: a clinical-policy revision, pending re-review.**
* **Partial windows** (`partialWindows: countCeiling`). A partial window
  below the minimum fails when it holds more severe reports than a
  compliant minimum window may: 5. A priority with nothing judged is
  INSUFFICIENT, never a vacuous pass. The audit's 640-report witness was
  GO and is now NO GO.
* **Horizon** (`safetyCohort: ordersByHorizon`). The absolute limits
  cover every study of an order wanted by the horizon, observed in the
  follow-up. Each study is in one state: observed violation, unresolved
  overdue, not yet observable (INSUFFICIENT), or completed in the
  follow-up. Unacquired orders are counted, and Gate A judges them.
  * **Witness:** the one-second witness was GO and is now NO GO.
  * **Link:** records carry a study-to-order link, exported without any
    draw.
  * **Integrity:** a post-horizon study without a valid link is INVALID.
* **Thresholds, minimums, anchor, allowance, Gate B and the safeguard:**
  unchanged.

**Protocol (proposed, not locked):**
* **The protocol itself.** `reserved-validation-protocol.md` gives every
  criterion an estimator, interval, family rule, numeric threshold,
  group and remainder handling, and PASS / FAIL / INCONCLUSIVE. A FAIL
  or INCONCLUSIVE is a documented NO GO; a post-result waiver is a
  deviation.
* **T2.** Harmed-seed detection must be at least 0.95. Whole-evaluation
  rejection is reported against the allowance's 0.371.
* **T5.** The tolerances are proposed (0.40 at +2.1 points, 0.01 at +3)
  and **await the owner's approval**.
* **The manifest.** `scenario-manifest.json` defines 49 scenarios:
  * disjoint streams and identical paired demand (no A85-against-B85
    pair);
  * random injections, record-reconciled;
  * exact margin witnesses.

  It is executed by `reserved_manifest_test.go` and judged by
  `reservedCriteria`.

**Reporting:**
* **G6:** occupied weeks are counted correctly, and unavailable unimaged
  fractions are null.
* **T7:** the raw per-gate insufficient seeds are listed.
* **G5:** the allowance interaction is documented
  (`allowance-interaction.md`) and the safeguard's overlap wording
  corrected.
* **G7:** the 40-seed impossibility statement is corrected.

**Development checks** (PCG(20261021, ·), simulator seeds 901–910):
* **Construction.** Every scenario was constructed and paired; all 14
  witnesses met their expectation. One construction defect was found
  and repaired: modality drawn from the booking stream made v4 case
  selection reject 4 of 20 stable groups. After the repair, none.
* **Near the margin.** Gate B accepted 0.32 at +2.1 points (predicted
  0.31) and 0 at +3.
* **Follow-up.** It leaves 1–6 routine studies not yet observable on 9
  of 10 no-action seeds (INSUFFICIENT). A 45-day follow-up is offered to
  the owner.

**Tested:** the boundary, horizon, allowance, reporting, criteria and
manifest witnesses; the mutation checks; the full uncached suite, vet,
gofmt and the replay fixtures.

**Next:** the owner's decisions:
* the T5 tolerances;
* the follow-up length;
* a targeted independent re-review of the boundary-safety revision.

Then the lock commit, and the reserved run exactly once. Reserved
calibration and the freeze stay NO GO until then.

---

## V0.2 Slice 5C.4f — volume-aware clinical severity windows (0.2.0-dev.20)

Each priority's severity window now matches its volume (owner decisions
of 2026-10-19; `docs/calibration/clinical-5c4f/`). The evaluation window
changed; the severity standard did not.

* **Not frozen.**
* **Registers 0.2.0-dev.19** for replay (`mini-seed49-dev19`).
* **v0.2-eval-4 is byte-identical.** The 5C.4d and 5C.4e candidates'
  exact bytes still evaluate under their own rules.
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream.

**Windows.** Each priority's window is set explicitly in the policy
(`weeklyWindowWeeks`):

| Priority | Weekly gate | Minimum | What remains for it |
|---|---|---|---|
| urgent | none | none | the per-seed 5% severe rate and the absolute 24 h limit |
| soon | fixed four-week calendar blocks | 100 | |
| routine | fixed calendar weeks | 100 | |

* **Blocks:** anchored at the first complete Monday, consecutive and
  never slid or pooled.
* **Below the minimum:** a complete window is INSUFFICIENT; a partial
  window is NOT_JUDGED, disclosed and never a pass.
* **Safeguard:** approved at 10 / 6 / 5 points.

**Development evaluability (seeds 901–910, no action):**
* **Urgent:** every seed is evaluable.
* **Soon:** complete blocks hold 127–218 acquisitions (mean 166), and
  none is insufficient.
* **Routine:** none of 200 weeks is insufficient.
* **The no-action practice fails on genuine delays:** soon blocks fail
  on 9 seeds (reports 42–45 h past due, severe rates up to 9.4%), and
  urgent absolute violations occur on 8 seeds.
* **Anchoring:** moving the anchor 1–3 weeks changes which seed fails
  on 1–2 of 10 seeds, never evaluability.

**Tested:** the window witnesses (`severity_window_test.go`), the
policy validation, 11 mutations (all caught), the full uncached suite,
vet, gofmt and the replay fixtures.

**Next:** a narrow independent review of the final v5 policy and its
statistical claims. The confirmatory protocol is complete and not run.
The reserved calibration and the freeze are NO GO until that review is
done.

---

## V0.2 Slice 5C.4e — empirical clinical acceptance and weekly severity closure (0.2.0-dev.19)

The owner's decisions of 2026-10-19 (`docs/calibration/clinical-5c4e/`):
Gate B changes from a population confidence bound to an **empirical
benchmark acceptance rule**, with a per-seed safeguard; the weekly
severity moves to **fixed Monday–Sunday calendar weeks**.

* **Not frozen.**
* **Registers 0.2.0-dev.18** for replay (`mini-seed48-dev18`).
* **v0.2-eval-4 is byte-identical;** the 5C.4d candidate's exact bytes
  still evaluate under their own rules.
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream.

**Gate B (`lostDemand.rule = "empirical"`).**

* PASS when the observed seed-averaged increase on the 40 paired seeds
  is at most 2 points (exact rational), FAIL above it; unmatched, thin
  or too few seeds INSUFFICIENT, never PASS.
* No bound, confidence or population enters the decision or its
  reasons. The `betting` bound is reported beside it as a labelled
  research diagnostic; `t` is refused even for that.
* The seed differences' distribution and both absolute unimaged
  fractions are reported.
* Every comparison states its claims apart: empirical acceptance on
  the paired seeds; population confidence not claimed; absolute
  adequacy not established. Every favourable verdict leads with the
  access limitation and the empirical scope.

**Per-seed safeguard (`maxSeedIncrease` 10 / 6 / 5 points).** A seed
whose own increase exceeds it fails in the shared allowance (4 of 40),
so improvements elsewhere cannot hide concentrated harm. Thresholds
from a prespecified rule on development-only evidence
(`seed-safeguard.md`): z(0.999) times the largest no-effect sd among
the development seeds, floored at 5 points.

**Weekly severity (`weeklyPartition = "calendar"`).** Every complete
week judged on its own: INSUFFICIENT below the minimum, FAIL above 5%;
a partial boundary week below the minimum NOT_JUDGED, never PASS; no
pooling. **Stop condition:** on the development seeds every complete
urgent week (6.7 acquisitions a week against a minimum of 20) is
insufficient, so every seed lacks evidence and no plan can pass; soon
is insufficient on every seed too. Reported for the owner; the
minimums and partition are unchanged.

**Tested:** the 20 required witnesses (`empirical_acceptance_test.go`),
policy validation, the safeguard rule and its development derivation
(`seed_safeguard_test.go`), the mutation checks, the full uncached
suite, vet, gofmt, the replay fixtures.

**Open:** the weekly minimums; the Astra review of the empirical rule
and safeguard; the reserved confirmation (NO GO); the freeze (NO GO).

---

## V0.2 Slice 5C.4d — audit remediation and seed-level lost-demand inference (0.2.0-dev.18)

This slice remediates the independent 5C.4c audit (kept verbatim; the
response is in `docs/audits/slice-5c4c-access-gates-audit-response.md`)
under the owner's decisions (`docs/calibration/clinical-5c4d/`).

* **Not frozen.**
* **Registers 0.2.0-dev.17** for replay (`mini-seed47-dev17`).
* **v0.2-eval-4 is byte-identical.**
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream.

**Fixes:**

* **F2:** the validated named end capture.
* **F3:** unknown outcomes and priorities are INVALID, never a panic or
  a pass.
* **F8:** every clinical fraction is finite and in [0, 1].
* **F9:** weekly spans are given in calendar weeks and dates.

**Identity.** Evaluation orders carry the simulator's immutable
`origin`, drawn from nothing. Under three plans it is unique within a
record and identical, request for request.

**Gate B:**

* now seed-level: one difference per matched seed, estimand their mean;
* the invalid conditional bound is removed;
* two procedures implemented: `betting` (finite-sample) and `t`
  (approximate).
* **No valid bound can pass the 2-point margin at 40 seeds.** The best
  any valid method can reach is 7.2 points; `betting` reaches 20.7 (2.0
  needs 400 seeds).
* So with the guaranteed method **Gate B is INSUFFICIENT for every
  plan.** This is reported for the owner's decision, not weakened.
* **Coverage study:**
  * `betting` covers in every scenario;
  * `t` is near nominal in smooth scenarios but covers only 0.70 (and
    accepts a 3-point increase 30% of the time) under all-or-nothing
    seed shocks.

**Verdict limitation.** Every favourable verdict states "Access
non-deterioration only; absolute access adequacy not established." with
access statements: timeliness, absolute unimaged fractions for
candidate and baseline, and the bound.

**Weekly rule:**

* the strict 5% ceiling and forward pooling are kept;
* backward pooling is reported, with a partition-sensitivity flag;
* a valid partition alone can change the verdict, in synthetic
  witnesses and on 1 of 10 no-action simulator seeds (urgent): a
  **freeze blocker**.

**Protocol.** The reserved protocol is updated: near-margin pairs,
coverage, clusters, identity, every audit witness. It is **not
runnable** until the owner decides the Gate B method and the weekly
partition rule.

---

## V0.2 Slice 5C.4c — clinical access and weekly severity gates (0.2.0-dev.17)

The owner's modified option B and the volume-normalized weekly
severity, in the v5 evaluator
(`docs/calibration/clinical-5c4c/access-gates.md`).

* **Not frozen:** the candidate policy is still a proposal.
* **v0.2-eval-4 is byte-identical** (`TestV4Neutrality`).
* **Earlier behaviour kept:** v5 policies without the new fields (every
  calibration harness) are unchanged, and the committed calibration runs
  reproduce (`TestOCFidelityModelUnchanged`).
* **Registers 0.2.0-dev.16** for replay (batch mode `v0.2_dev16`,
  fixture `mini-seed45-dev16`).
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream.

**The rules:**

* **Gate A (timeliness):** served, open and late-cancelled eligible
  orders, from the original wanted time, at ≥ 95% (urgent 48 h; soon 7
  and routine 30 days, provisional). Lost and left orders are counted
  apart.
* **Gate B (lost demand):** unimaged demand (lost, left or ended by the
  patient) against the no-action baseline, per priority.
  * Paired only on identical eligible demand: identity by receipt and
    wanted times, priority, modality and walk-in. The simulator's demand
    is shown exogenous across plans (`TestDemandExogenousAcrossPlans`).
  * The exact one-sided 95% bound of the pooled increase must be at most
    2 points.
  * Unmatched demand is insufficient (NO GO).
  * It is a non-deterioration gate: absolute fractions are always
    reported, and adequacy is not certified.
* **Weekly severity:** severe-delay rate ≤ 5% in blocks of Monday weeks
  of at least 20, 40 or 100 acquisitions; low volume is pooled, a short
  remainder is joined, and too few is insufficient.
  * Severity is defined apart from the absolute limits: urgent 12 h,
    soon 24 h, routine 72 h past due.
* **Maximum wait:** reported only.

**Synthetic model:**

* the takeover instant is a field, and the aligned takeover gives
  calendar evaluator weeks (the old ones ran Wednesday to Wednesday);
* demand can be drawn from its own stream, so that paired practices
  share it.

**Tests:**

* cases C1–C10 and C7a (lost, delayed, loss against timeliness,
  cancellation manipulation, rebooking, horizon, low-volume weeks,
  overlapping failures, missing baseline, noncomparable demand);
* the updated witnesses W1–W12;
* validation;
* 10 of 10 mutations caught.

**No-action simulator:**

* Gate A passes the baseline: urgent 0.993, soon 0.965, routine 0.979.
* Gate B reports 65% / 38% / 14% lost or left, not certified.
* The weekly 5% rule fails every seed on soon (weekend concentration).

**Next:**

* the reserved protocol is complete but not to be run;
* the Astra High review package is prepared;
* the owner approvals are listed in access-gates §7.

---

## V0.2 Slice 5C.4b closure — clinical acceptance design (0.2.0-dev.16)

The final specification step before v0.2-eval-5
(`docs/calibration/clinical-5c4b/acceptance-design.md`).

* **Not frozen, and blocked on an owner decision:** urgent access
  cannot be justified as specified.
* **No version change:** only tests, test data, a candidate policy
  outside `docs/evaluation` and documents changed.
* **Nothing else was run:** no strategy, reference or holdout seed,
  unseal record or reserved stream. The simulator ran with no action on
  the development seeds.

* **Order lifecycle** (`order-lifecycle.md`): every state, its
  initiator, and what the evaluation record carries and omits.
  * The order and acquisition mixes reconcile through loss: **65% of
    urgent orders are lost at the first offer**. 75% of urgent orders
    tolerate only the same day, and booking reserves nothing for urgent
    work.
  * About two thirds of urgent losses are staffing, the rest booking
    structure. No plan action changes the booking structure.
* **Access denominators** (`access-denominators.md`): rules A–J as
  built, with no survivorship.
  * Lost and left orders are untimely. Open orders are untimely past
    the deadline, undecided before it, and insufficient is not a pass.
  * A patient outcome before the deadline is withdrawn, after it
    untimely.
  * The wanted time is fixed.
  * Inherited orders are timed from the takeover.
  * **The calendar clock is recommended.** The open-hours clock would
    hide weekend delay.
* **No-action simulator access, every eligible order:** urgent 48 h
  **0.33**, soon 7 days 0.58, routine 30 days 0.83. Of served urgent
  orders, 99% are within 48 h.
  * A 90% or 95% urgent threshold rejects every plan.
  * Options A (as built), B (timeliness of served and open orders, plus
    a per-priority loss gate against the baseline: recommended) and C
    are set out for the owner.
* **Reporting limits on the no-action records** (utilization 0.95, no
  weekend reading):
  * urgent 24 h fails 8 of 10 seeds (the weekend gap, as intended);
  * soon 48 h and routine 14 days hold;
  * the weekly count of 5 fails every seed for routine (20–37 in the
    worst week): owner decision.
* **Candidate policy** (`candidate-policy-v5.json`): v0.2-eval-4 plus
  the clinical section, a proposal and not in force.
  `TestCandidatePolicyFile` checks that it changes no v4 rule.
* **Witnesses W1–W12** (`clinical_witness_test.go`): the audit's
  delays, lost, cancelled, open and rescheduled urgent orders, the
  Saturday order, observation boundaries, contradictory data and
  manipulation. Each has its expected classification, denominator,
  numerator and outcome, and all pass.
* **Reserved-stream protocol** (`reserved-validation-protocol.md`): a
  draft, not runnable until the owner's decisions.
  * PCG(20261201, ·), verified never used.
  * It needs one model correction first: the synthetic records'
    evaluator weeks start on Wednesday.

---

## V0.2 Slice 5C.4b, concluded — simulator-fidelity calibration (0.2.0-dev.16)

The owner's final calibration correction before choosing the
v0.2-eval-5 values (`docs/calibration/clinical-5c4b/fidelity-protocol.md`,
committed before any result on its streams; findings in
`findings-fidelity.md`). There is no version change: only test code,
test data and documents changed.

* **No v5 policy and no value chosen.**
* **The simulator was measured, not a strategy.** The evaluation
  scenario's own practice ran with no action on the development seeds,
  through the holdout guard. No reference or holdout seed was touched.
  The profile is `internal/eval/testdata/sim-profile-5c4b.json`.
* **What the profile shows:**
  * 51.7 acquisitions a day, Saturday 5.9%;
  * acquisition mix 1.9 / 11.6 / 86.5% (the order mix is 4.4 / 14.9 /
    80.7%, most urgent orders being lost);
  * reading utilization 0.952;
  * soon breach 0.36;
  * worst delays about 40–45 h.
* **The fidelity model:**
  * the simulator's demand, mix, weekday distribution and acquisition
    hours (slot-timed);
  * its two radiologists in their reading blocks, with stochastic read
    times by modality, the shift-end rule, overrun debt and
    urgent_then_fifo;
  * urgent weeknight lanes and a weekend reader;
  * regimes on the core reading capacity;
  * a 181-day warm start.
  * The earlier models and results are unchanged.
* **Results, 24 scenarios and 36,000 seeds.**
  * **Calibration:** core load 0.85 and 0.92 achieved. Queues are
    stable; takeover backlogs are 1 study at p50.
  * **At 0.85:**
    * no severe delay;
    * lenient and moderate bundles reject nothing, every failure being
      v4's backlog rule;
    * urgent 24 h, soon 48 h and routine absolute limits are admissible
      at zero tolerance.
  * **At 0.92:** 0.2–0.4% of seeds violate urgent 24 h.
  * **Weekend gap:** worst urgent about 40 h at equal staffing and at
    equal utilization. A 24–36 h urgent limit violates about 70% of
    E3-like seeds; 48 h almost none.
  * **Urgent access within 1 day at 95%** fails every stable seed
    (Sunday closure). Within 2 days it fails 2–3%.
  * **Detection:** harmful reading losses are caught by the clinical
    gates; L10 at 0.85 only by v4's soon gate (clinically mild); imaging
    losses are absorbed.
* **The mirror against the simulator.** At the simulator's own load the
  mirror's breach and severe shares are 1.5–3× the simulator's. The
  protocol's validity criterion fails, and no simulator-level validity
  is claimed.
* **Tests:**
  * `TestSimProfileDerived`, `TestOCReaders`, `TestOCReaderRules`,
    `TestOCFidelityCalibration` and
    `TestClinicalOCFidelityAgreesWithEvaluate`;
  * 13 of 13 mutations caught;
  * the full uncached suite, `go vet` and gofmt pass at 18b21d6.

---

## V0.2 Slice 5C.4b, continued — calibration correction (0.2.0-dev.16)

The exploratory operating characteristics were run and reproduced. The
owner then asked to hold the threshold choice and correct their
calibration first. No version change: only test code and documents
changed, and the binary is the same.

* **No v5 policy, no threshold chosen.** Nothing else was run: no
  strategy, development, reference or holdout run, and the confirmatory
  streams are untouched.
* **The original run** (`oc-v0.2-eval-5-exploratory.json`) is committed
  as generated, with joofi and the Pi byte-identical.
  * Its findings carry an erratum. The stable practice measured 0.67
    (0.74 over the last 30 days), not the intended 0.85.
  * Its rate was sized over 66.5 reading hours but it read 70.5,
    cancellations were not counted, and it started empty.
* **The correction** (`docs/calibration/clinical-5c4b/correction-protocol.md`,
  committed before any corrected result):
  * utilization is measured per seed, and the rate is set from the
    target over the practice's own hours;
  * the practice runs 181 days before the takeover and inherits the
    work open then;
  * it reads in the simulator's urgent_then_fifo order;
  * regimes at 0.85 and 0.92, on new streams PCG(20261102, ·).
  * Each change is a field whose zero value is the original model. The
    original run still reproduces byte for byte.
* **Results** (`findings-corrected.md`):
  * **Utilization** measured 0.845–0.849 and 0.914–0.919.
  * **At 0.85:**
    * no v5 clinical gate value fails a stable seed;
    * the bundles' 40-seed rejection is at most 0.002, from v4's own
      gates;
    * urgent absolute limits of 24 h or more, and soon of 48 h or more,
      are admissible at zero tolerance. Soon 24 h is not: one seed of
      2,500 reached 34.9 h.
  * **At 0.92:** v4's 4-hour soon breach gate rejects 85–89% of
    seeds, with severe delays below 0.1%.
  * **Weekend, at equal utilization:** no weekend reading moves the
    worst urgent delay from 14 h to 42–44 h. An urgent limit of 36 h or
    less makes it a violation; 48 h does not at 0.85.
  * **Deterioration:**
    * reading losses that harm (L20, D, K, L10 at 0.92) are caught by
      the clinical components under every bundle;
    * L10 at 0.85 is clinically mild (severe below 0.05%) but rejected
      by v4's soon turnaround gate;
    * imaging losses are absorbed.
* **Against the simulator** (read, not run):
  * **Matched:** the worklist rule, imaging hours, demand weekdays,
    targets, and the warm start.
  * **Not matched:** individual readers and read times, modalities,
    booking, and the Saturday load (12.8% of acquisitions against about
    5%).
  * No simulator-level validity is claimed.
* **Tests.**
  * `TestSimPick` checks the simulator's worklist rule.
  * `TestOCCorrectedCalibration` checks utilization and inherited work
    on the debug stream.
  * `TestClinicalOCCorrectedAgreesWithEvaluate` checks exact agreement
    with the production evaluator.
  * 10 of 10 mutations were caught. One of them is a sentinel bug
    (day −1 as "no slot"), found and fixed before the protocol was
    committed.
  * The full uncached suite passes at d11964f, `TestEvaluateMatrix`
    included, as do `go vet` and gofmt.

---

## V0.2 Slice 5C.4b — the v0.2-eval-5 clinical evaluator (0.2.0-dev.16)

This slice implements the clinical acceptance rules specified in 5C.4a
(`docs/evaluation/proposals/v5-clinical-acceptance.md`, implementation
decisions in its §J). It also fixes the protocol for the combined
synthetic operating characteristics
(`docs/calibration/clinical-5c4b/protocol.md`), committed before any
combined result. Design §13.11.

* **No v5 policy is frozen.** The owner chooses the values from those
  operating characteristics (owner decision: build first, values
  later; the weekend question after the operating characteristics).
* **Unchanged:** v0.2-eval-4, the simulation and every strategy.
* **Nothing else was run:** no strategy, development, reference or
  holdout run.

* **Rules.** These are reached only by a policy of schema
  `dreamteam.evalpolicy/v5` (`internal/eval/clinical.go`):
  * absolute past-due limits by priority, from exact timestamps, with
    unfinished studies censored at the end of observation, at zero
    tolerance;
  * the severe-delay rate and severe delays per Monday week;
  * the timely-acquisition fraction from the wanted time, with open,
    lost and abandoned orders untimely, a patient's withdrawal before
    the limit excluded, and a rebooking never resetting the clock;
  * the maximum wait;
  * inherited work timed from the takeover or from its own due time;
  * one shared seed allowance over service, severity, access and
    backlog, each seed counted once, with the absolute limits kept
    apart (NO GO, clinical safety);
  * the sampled age limit reported only;
  * the M3 fix (an insufficient baseline is not restored);
  * the G10 integrity check (an order imaged before its wanted time).
* **v0.2-eval-4 neutrality.** `TestV4Neutrality` pins the evaluation of
  a varied synthetic input. It covers GO, NO GO, service, backlog and
  case-selection paths, with a digest recorded before any evaluator
  change. The digest is unchanged.
* **Acceptance cases G1–G6 and G10–G14** on synthetic records. Each
  harmful case has its acceptable counterpart and exact boundaries.
  * The audit's witnesses (six urgent reports eight days late; a report
    348 h past due; thirty urgent acquisitions thirty days late) now
    fail under v5 and still pass under v0.2-eval-4.
  * Four seeds failing service plus four others failing backlog pass
    v0.2-eval-4's separate allowances and fail v5's shared one.
* **The combined synthetic practice** (`oc_clinical_test.go`).
  * It adds an imaging and booking stage to the 5C.3 reading queue:
    wanted times, a patient-tolerance loss, cancellations, and
    acquisitions sorted in time (the M1 fix; the 5C.3 queue itself is
    unchanged).
  * It produces complete records, which the production evaluator
    judges.
  * Its tabulation agrees exactly with `planReport` on sampled seeds of
    a stable, a structural and a harmful scenario.
  * The stable practice's reading hours cover its imaging. The
    scenario's own rota failed v0.2-eval-4's service gates on a debug
    stream, so it is kept as a structural variant. Both are recorded in
    the protocol.
* **Session replay.** 0.2.0-dev.15 is registered: batch mode
  `v0.2_dev15`, with fixture `mini-seed43-dev15` recorded by the
  31a2573 binary.

**Tests.**

* New: `TestV4Neutrality`, eleven `TestClinical*` tests,
  `TestClinicalOCAgreesWithEvaluate`, and `TestDev15RecordReplays` and
  `TestDev15RecordContinues`.
* Sixteen targeted mutations of the new rules are each caught. Four
  survived the first pass and were closed with stronger tests and one
  shared severity helper.
* Removing the dev.15 registration fails the replay tests.
* **An existing intermittent failure, fixed.**
  * Since 5C.0b, `TestEvaluateMatrix` shared one strategy-pin map
    across its cases.
  * One case deleted a pin, so in some random map orders another case
    failed. It failed in the first full run of this slice.
  * Each case now gets its own copy: 20 of 20 runs pass, where before
    20 runs gave several failures.

**Limitations.**

* **Values.** The values are not chosen; the exploratory run follows
  in its own commit.
* **Synthetic model.** It is simplified: one modality pool, day-level
  booking, and constant finance.
* **Saturday routine bookings.** In the practice without weekend
  reading, Saturday routine bookings follow the wanted day uniformly,
  which may overstate its routine breach rate.

## V0.2 Slice 5C.4a — audit record, holdout containment, clinical policy proposal (0.2.0-dev.15)

The independent audit of v0.2-eval-4 at 7707dbd gave **NO GO** for the
confirmatory run. This slice preserves the audit, contains the ordinary
tests' use of holdout seeds, and specifies, without activating, the
clinical acceptance rules for a successor policy. Design §13.10.

Unchanged:

* v0.2-eval-4, which stays in force byte for byte, with every earlier
  policy;
* the simulation, scenarios, demand, staffing, economics and TAT
  targets;
* every strategy file and development result.

Nothing was run on a strategy, a reference seed or a holdout seed. There
is no unseal record, and no primary strategy is chosen.

* **The audit** is committed separately (e54cdca), as
  `docs/audits/slice-5c3-eval4-framework-audit.md`.
  * The auditor's read-only findings are kept as delivered.
  * A separate maintainer appendix records the four executed witnesses:
    six urgent reports 8 days late, 30 urgent acquisitions 30 days late,
    a report 348 h past due seen as 334 h, and an insufficient baseline
    counted as restored. Each got the verdict the audit predicted. It
    also records code checks of H3 and H6.
* **Exposure**, recorded in `docs/evaluation/holdout-exposure-5c4a.md`.
  * From Slice 5C.0b, `TestHoldoutRecordersGuarded` built a
    mini-scenario world on the first real holdout seed, ran it to the
    takeover only, and checked that the recorders refuse it.
  * Guard tests also regenerated the seeds in memory.
  * There was seed use, but no outcome was disclosed and no
    confirmatory run took place. The holdout is kept (owner decision).
* **Containment.**
  * The guard tests use synthetic holdout manifests.
  * `NewEvalWorld` refuses a holdout seed before any world is built.
    The development runner and the recorder tests use it.
  * `Authorize` refuses a sealed holdout before regenerating seeds.
  * `TestHoldoutContainment` reads all Go source and fails on any
    unreviewed site that names the manifest, regenerates its seeds or
    authorizes with the repository's paths.
* **Clinical policy proposal**, in
  `docs/evaluation/proposals/v5-clinical-acceptance.md`. It is not
  active.
  * Exact-timestamp severe-delay rules: the age limit made zero
    tolerance, priority maximums, and the incidence and weekly
    concentration of severe delays.
  * Wanted-to-acquisition access, with no survivorship: a timely
    fraction and a maximum wait.
  * One shared four-seed clinical allowance, with absolute limits kept
    apart.
  * v0.2-eval-4's backlog rule kept; its 150% allowance documented as a
    ceiling of 250% of the adjusted reference.
  * Every numerical limit is an owner decision. The proposal states the
    scenario's structural weekend delay (urgent up to about 40.5 h past
    due without weekend reading). It also states the cost of the shared
    allowance: under independence the stable seasonal-bursty queue's
    40-seed false rejection rises from about 0.05 to 0.22.
  * A 15-case synthetic acceptance test plan is set out for 5C.4b.
* **README.** v0.2-eval-4's README states the 250% ceiling and that the
  age limit is sampled, and records the audit and the proposal.
* **Session replay.** 0.2.0-dev.14 is registered: batch mode
  `v0.2_dev14`, with fixture `mini-seed41-dev14` recorded by the 7707dbd
  binary.

**Tests.**

* New: `TestHoldoutContainment`, `TestGuardSeeds`,
  `TestAuthorizeSealedFirst`, and `TestDev14RecordReplays` and
  `TestDev14RecordContinues`.
* Rewritten on synthetic fixtures: `TestHoldoutRecordersGuarded` and
  `TestGuardRepo`.
* `TestHoldoutSealed` keeps only the integrity checks of the real
  manifest.
* Six mutations are each caught:
  * the guard removed from `NewEvalWorld`;
  * a new manifest-path reference;
  * a non-synthetic `HoldoutSeeds` label;
  * a new `Authorize(RepoPaths)` site;
  * the old real-manifest guard test restored;
  * the dev.14 registration removed.
* The pre-fix `Authorize` order fails `TestAuthorizeSealedFirst`.

**Limitations.**

* The guards still regenerate the real seeds in memory in order to
  refuse them.
* The manifest lets anyone with the repository regenerate the seeds.
* The containment test checks names, not data flow.
* Custody, approval binding and run-once are Slice 5C.4c's.

## V0.2 Slice 5C.3 — backlog-gate operating characteristics (0.2.0-dev.14)

The owner's backlog decisions on the Slice 5C.2 findings, and a backlog
rule selected by a prespecified protocol on synthetic queues. Both are
frozen as **policy v0.2-eval-4** (`docs/evaluation/policy-v0.2-eval-4.json`, SHA-256
4e277c72…b58ffb6fb3). Design §13.9; rules in
`docs/evaluation/README.md` §4; protocol, findings and re-evaluation in
`docs/calibration/backlog-5c3`.

Unchanged:

* the breach gates (12% / 32% / 6%), the joint seed rule and its 10%
  allowance;
* case selection, solvency, the dated funding test, and T90 as the
  primary endpoint;
* the simulation, staffing, demand, economics and TAT targets;
* the holdout, which stays sealed. No reference seed was run, and there
  is no unseal record.

v0.2-eval-3 (digest 0cf4884f…), its README, its operating
characteristics and its development report are kept byte for byte. No
primary strategy is chosen.

* **Diagnosis.** v0.2-eval-3 reproduces 5C.2's stable-queue result
  exactly (191 and 263 of 300). Its false failures are the peak alone
  (50% stationary, 27% seasonal) and the peak with the mean (13%, 57%).
  With bursty days, the late-work means also fail.
* **The rule.**
  * level: the workload-adjusted block mean, within max(150%, 2
    studies);
  * overdue, stale and exposure: block medians, at their old
    tolerances;
  * stale cap: 3 studies, in every block from the reference on;
  * age: 14 days past due, unchanged;
  * reported only: the peak, recovery time and the old mean check.
* **How it was selected.**
  * 720 candidates were scored by the production judge (`judgeBacklog`)
    on 18 synthetic scenarios, 1000 fresh-stream seeds each.
  * 84 met T1 (every stable queue's 40-seed rejection ≤ 5%) and T2
    (every deterioration detected on ≥ 90% of seeds).
  * The protocol's ladder and tie-break order chose one.
  * `TestOperatingCharacteristics` checks that the frozen policy is that
    candidate.
* **Operating characteristics.**
  * Stable queues: 0–3.8% of seeds unstable (v0.2-eval-3: 61–92%). The
    worst 40-seed rejection is 0.017.
  * Deterioration scenarios: 97.6–100% unstable.
  * Not detected: a capacity step that leaves the queue stable (10%
    lost: 23%), and slow erosion (46%). A seasonal queue at utilization
    0.92 is rejected at 40 seeds with probability 0.56 (sensitivity).
* **Clustering interpretation** (README §2).
  * Empirical acceptance, population reliability, false-rejection
    probability and power are reported apart.
  * The holdout is a strict empirical acceptance test.
  * The independent-breach figures are a lower bound on false
    rejection, not a guarantee.
* **Development re-evaluation** (`p6-eval4`, seeds 901–910, strategies,
  records and horizon unchanged):
  * D2 is CONDITIONAL GO (transition, not yet repaid, restoring service);
  * E2 and E3 are CONDITIONAL GO (service purchase);
  * F, G and H are NO GO on solvency; B–E are NO GO on service;
  * the no-action plan now fails backlog on its overdue median (3 of 10
    seeds);
  * service, funding and solvency are identical to v0.2-eval-3.
* **D2 against E3.**

  | | D2 | E3 |
  |---|---|---|
  | Pooled urgent | 9.0% | 2.8% |
  | Worst seed urgent | 11.6% | 3.7% |
  | T90 Δ | −$18.8k | −$23.2k |

  Both are backlog-stable on 10 of 10 seeds.
* **Registration.** 0.2.0-dev.13 is registered (batch mode `v0.2_dev13`).
  Its fixture is `testdata/sessions/mini-seed39-dev13.*`, recorded by
  the dev.13 binary from a clean worktree of dc1c043.

**Tests.**
* `TestBacklogStability` covers each protocol case; `TestPolicyFrozen`
  checks the v4 → v3 → v2 → v1 chain.
* `TestSeasonalBacklog`, `TestBacklogSafeguards` and
  `TestBacklogTrendRotaAligned` keep the earlier rules' cases under
  v0.2-eval-3 and assert v0.2-eval-4's outcomes beside them.
* **Fail-before-fix:** 16 targeted mutations, each breaking one new rule
  (informational counted as hard, statistics, the cap and its
  reference, level tolerance and scaling, the median, recovery,
  persistence, policy validation, the superseded schema, the covering
  block). All 16 are caught. "Level statistic ignored" survived the
  first pass and is caught after a mean-versus-median surge case was
  added.

**Verification.**
* `p6-eval4` ran on joofi and reproduced on the Pi byte for byte:
  `run.json`, report and trajectory.
* The operating characteristics were generated on the Pi (old candidate
  format) and on joofi; every number is identical. The backlog part,
  recomputed on the Pi (`DT_OC_BACKLOG=1`), equals joofi's committed
  file byte for byte.
* The full uncached suite passes (12:39); `go vet` is clean for every
  tag (none, calibration, acceptance), and gofmt is clean.
* The dev.13 tests fail without the registration.
* Holdout sealed (no `holdout-unseal.json`); historical references and
  v0.2-eval-3 files unchanged.

**Limitations.**
* The calibration rests on a simple hourly synthetic queue.
* The candidate family was shaped by exploratory sweeps, though the
  targets were fixed before them and the confirmatory run used fresh
  streams.
* One protocol line was corrected after the run: the per-seed limit is
  5.06%, not the hand estimate of 3.4%. The selected rule's worst stable
  scenario (3.8%) meets the actual criterion.
* Development verdicts rest on 10 seeds and are exploratory.

---

## V0.2 Slice 5C.2 — policy feasibility, seasonal backlog and cluster-aware service (0.2.0-dev.13)

The owner's corrections to the rules Slice 5C.1 showed to be infeasible
or statistically misleading, frozen as **policy v0.2-eval-3**
(`docs/evaluation/policy-v0.2-eval-3.json`).

Unchanged:

* the breach gates (12% / 32% / 6%), the joint seed rule and its 10%
  allowance;
* case selection, solvency, and T90 as the primary financial endpoint;
* the simulation, staffing economics, demand, pricing and TAT targets;
* the holdout, which stays sealed. The reference seeds were not run.

v0.2-eval-2 (digest 5dbab462…) and its README are kept byte for byte,
and its 5C.1 reports are unchanged. No primary strategy is chosen.
Design §13.8; rules in `docs/evaluation/README.md`; diagnostics and the
re-evaluation in `docs/calibration/policy-5c2`.

* **Funding: a dated settlement test** (§5). It replaces "cash ≥ the
  whole bridge, 14 days of future wages included, at every instant",
  which the no-action practice failed on 76% of samples while paying
  every bill on time. A seed is funded when:
  * cash after every posting is at or above the floor, and nothing is
    past due (solvency, unchanged);
  * observation runs at least 31 days past the horizon (the longest
    payment cycle: payables on the 1st), so every obligation recognized
    by the horizon is seen paid from actual collections;
  * at the end, cash covers the committed liabilities outside the cycle
    (severance payable, contract minimums, fees in flight).

  Receivables are never forecast. The bridge is reported split by kind,
  with the old 14-day coverage as a liquidity-risk indicator.
* **Backlog: workload-adjusted** (§4). The mean, peak, overdue and
  exposure checks compare a block with the reference scaled by its own
  acquisitions (Little's law), exactly. Stale and persistent work stays
  absolute. Rota alignment, the covering block and every check are kept.
* **Cluster-aware service** (§2). Every gate reports:
  * day (weekday-adjusted) and week design effects;
  * a seed-cluster-robust pooled interval;
  * for the joint rule, the empirical pass fraction and the exact upper
    bound on the population failing-seed fraction.

  The operating characteristics cover independent, variable-volume,
  day-clustered and week-clustered models, with power at unacceptable
  rates. The allowance is read prospectively: a pass is empirical, not
  a confidence statement.
* **Transition, not yet repaid** (§10): CONDITIONAL GO for a plan passing
  every hard gate whose T90 increment is not positive but whose days
  31–90 increment is.
  * T180 is never promoted to financial GO.
  * Every comparison reports its financial evidence: T90, operating
    window and T180, each absolute and incremental, plus transition
    obligations at T90.
  * Service restoration is stated as a fact, whatever the verdict.
* **Development storage.** New runs commit `run.json`, which pins the
  records, report and trajectory by digest, and the generated
  `report.md`. The detailed report and trajectory go to an ignored
  `detail/` directory and are checked against `run.json`.

**Re-evaluation** (set `p5-eval3`: A–H, D2, E2, E3, strategy files
unchanged; `docs/calibration/policy-5c2/reevaluation.md`):

| | v0.2-eval-2 | v0.2-eval-3 |
|---|---|---|
| Funding-gap seeds, every plan | 10 of 10 | 0 |
| Service joint rule | as before | unchanged (by design) |
| Backlog-stable seeds | A 0, D2 2, E2/E3 1 | A 1, D2 2, E2/E3 1 |
| Verdicts | NO GO | NO GO |

* D2, E2 and E3 now fail **only the backlog gate**.
* F, G and H fail solvency.
* A, B, C, D and E also fail service.
* If backlog passed (a counterfactual): D2 would be CONDITIONAL GO, a
  transition restoring service; E2 and E3 CONDITIONAL GO service
  purchases.

**What the operating characteristics show** (`oc-v0.2-eval-3.json`):

* **The backlog rule is too noisy for a stable queue.**
  * A synthetic queue at utilization 0.85 is unstable on 64% of seeds
    with stationary demand and 88% seasonal (v0.2-eval-2: 61% and 93%).
  * The peak check alone fails half of the stable seeds; without it,
    13% and 61%.
  * Alternatives are tabulated, none adopted: the tolerances are an open
    owner decision.
* **Service under clustering.** At the planning rates the frozen design's
  40-seed false rejection is:
  * 0.022 independent;
  * 0.34 at week ICC 0.02;
  * 0.96 across the development range.

  Power at the gates is 1.00 in every model.
* **D2 against E3 without independence.** A week-block bootstrap
  (exploratory): D2 fails a seed 13.7% of the time, almost all urgent,
  so it passes the 40-seed rule in 33% of replicates; E3 and E2 pass
  every one.

**Tests.**

* `TestDatedFunding` (deferral cannot conceal insolvency):
  * a past-due payment with ample cash;
  * negative cash between positive samples;
  * observation short of the settlement window (INVALID), and the
    window binding at a day-175 horizon;
  * severance, contract minimums or fees in flight above end cash;
  * exact coverage passes, and the indicator alone does not gate.
* `TestSeasonalBacklog`:
  * seasonal-stable passes, and fails v0.2-eval-2;
  * deterioration (flat demand, and beyond the workload) fails;
  * a backlog that does not fall when demand falls fails;
  * persistent work, a growing and a proportional stale burden fail;
  * a spike and late-tail deterioration fail;
  * with no workload recorded, the comparison is unscaled.
* `TestClusterStatistics`: design effects, weekday adjustment, robust
  intervals, Clopper–Pearson bounds.
* `TestTransitionClassification`: transition with and without
  restoration, service purchase, no benefit, restoration under NO GO,
  no T180 promotion.
* `TestPolicyFrozen`: v1 and v2 kept and refused; every backlog check
  adjusted or absolute exactly once.
* The operating-characteristics checks.
* `internal/session` `TestDev12Record*`.

**Fail-before-fix.** 20 targeted mutations of the new rules, all caught.
Two survived first and their tests were strengthened:

* the settlement window, invisible at a day-150 horizon;
* scaling stale, invisible below the 2-study floor.

Registers 0.2.0-dev.12 (batch mode `v0.2_dev12`, fixture
`mini-seed37-dev12` from the dev.12 binary).

**Limitations.**

* The backlog tolerances, and the dependence trade-off against the
  planning target, are unresolved owner decisions. Strategy selection
  is NO GO.
* The operating characteristics' models are synthetic. Their dependence
  range comes from 10 development seeds, and the queue is a simple
  hourly model.
* The bootstrap resamples weeks independently and has only 10 seeds of
  between-seed variation.

---

## V0.2 Slice 5C.1 — development-seed strategy comparison (0.2.0-dev.12)

Strategy development on the development seeds 901–910 only, under policy
v0.2-eval-2 unchanged. The holdout is sealed and untouched. The
reference seeds 1001–1020 were not used. Simulation, economics, demand,
durations, thresholds, the policy, the holdout manifest and the holdout
runner are unchanged. Design §13.7; the full record is in
`docs/calibration/strategy-development/README.md`.

* **Harness.**
  * `eval.Development` loads a strategy set in the holdout contract's
    formats: strict decoding, every pin checked, the baseline and
    `teleOnlyOf` validated.
  * It accepts only the policy's development seeds, and its protocol is
    never authorized.
  * `TestStrategyDevelopment` (tag `calibration`) runs the canonical
    runner's loop with the development recorder: the common horizon
    computed first and confirmed, the complete matrix, and the frozen
    evaluation.
  * It writes `run.json`, `report.json` and `trajectory.csv` once; a
    rerun must reproduce them byte for byte.
  * `report.py`, `reconcile.py` and `breach_timing.py` generate every
    table.
* **Phase 1.** A–D reproduce every archived development-seed daily
  sample exactly: 10,800 samples across engines dev.4 and dev.5, in
  profit, cash, reports and backlog. Their T90 breach rates are the same
  under all three rules.
  * D's historical CONDITIONAL GO becomes NO GO under v0.2-eval-2, from
    the rules alone:
    * the joint per-seed rule: soon fails on 6 seeds, urgent on 3;
    * the block backlog rule;
    * the funding gate.
* **Family (fixed before any teleradiology run, from D's breach
  timing):**
  * E: D plus urgent-only teleradiology on weeknights and Saturdays;
  * F: D plus all-priority teleradiology on every window;
  * G: B plus F's contract.

  All are open-ended from the earliest date, Oct 27, with no
  cancellation during the evaluation.
* **Four refinements**, each with a hypothesis written before its run:
  * R1 E2: E at a 65/55% allocation. Kept: it passes the joint service
    rule on all 10 seeds.
  * R2 H: D plus all-priority on weeknights only. Rejected: insolvent
    on 8 seeds.
  * R3 E3: E2 on weeknights only. Kept: the same gates, $2.0k cheaper
    per 90 days.
  * R4 D2: E2 without the contract. Kept: it passes service alone; its
    urgent margin is thin (worst seed 11.6% against 12%).
* **Result: no strategy passes, and none can as the policy is frozen.**
  Every plan, no action included, fails two hard gates on every seed:
  * funding: the 14-day employee commitment, about $53k, exceeds the
    cash headroom from day 1;
  * backlog: the reference block, Dec 7 – Jan 3, falls in the December
    demand low.

  These are policy limitations for the owner. The policy was not
  changed.
* **What passes the other hard gates.** D2, E3 and E2 pass service,
  pooled rates, case selection and solvency on all 10 seeds. They lose
  $18.8k, $23.2k and $25.2k at T90 against no action: a service purchase
  at best.
* **Teleradiology's marginal value.**
  * Urgent-only: urgent about 10% → 2–3%, for $4.4–6.5k per 90 days.
    It adds no reports and no revenue.
  * All-priority: every gate passes, for $27–36k per 90 days. It adds
    at most 23 reports and $3.0k of revenue, and is insolvent on 7–10
    seeds.
* **Recommendation.** No primary strategy under v0.2-eval-2 as frozen.
  * If the funding and backlog rules are revised, the prespecified
    ranking picks D2.
  * E3 is the robust alternative: D2 fails the 40-seed joint rule about
    9% of the time under independence, E3 essentially never.
  * The strategy set is not frozen, and no unseal record exists.
* **Registers 0.2.0-dev.11.** Batch mode `v0.2_dev11`, with a genuine
  fixture from the dev.11 binary (`mini-seed35-dev11`): an urgent
  contract, a `minimum_term` refusal and a cancellation. Without the
  registration the dev.11 tests fail as an engine mismatch.

**Tests.**

* `TestDevelopmentRunnerMatchesCanonical`: on the synthetic protocol,
  the development and canonical runners give byte-identical records and
  the same horizon, including a batch at a sample instant.
* `TestDevelopmentSet`: development seeds only; refuses a reference
  seed, an unpinned strategy, an acting baseline, another schema and an
  invalid `teleOnlyOf`; never authorized.
* `TestDevelopmentSetsCommitted`: every committed set loads strictly,
  and every batch is in [takeover, T90).
* `TestWriteOnce`.
* `internal/session` `TestDev11Record*`.

**Mutations.** 11 targeted mutations, all caught:

* the seed range, pin, schema, baseline and `teleOnlyOf` checks;
* batch-before-sample order, commitment days, horizon and capture
  equivalence;
* `writeOnce`'s two guards.

The batch-order mutation first survived, because the fixture had no
batch at a sample instant; the test now has one.

**Limitations.**

* Ten seeds: a single seed decides the development verdict. The
  results identify mechanisms and credible strategies, not 40-seed pass
  rates.
* Soon breaches are clustered (dispersion 2–9), so the per-seed soon
  gate's operating characteristics are optimistic.
* Every reorganization has a negative T90 increment, so under the frozen
  verdict order the best attainable outcome is CONDITIONAL GO, service
  purchase.
* `eval.Development` calls `GuardRepo` as defence in depth. It is
  redundant with the development-seed check, so no mutation can show it
  necessary.

---

## V0.2 Slice 5C.0b — canonical holdout runner (0.2.0-dev.11)

This step closes the audit's remaining high finding, H6: holdout
protection was a cooperative convention, not an enforced one-time
protocol. The holdout stays sealed and untouched. No strategy was run,
and the policy (v0.2-eval-2) and simulation are unchanged. Design §13.6;
rules in `docs/evaluation/README.md` §9.

* **One path to a holdout seed.**
  * `eval.GuardRepo` now refuses holdout seeds whether or not the
    holdout is unsealed.
  * Its refusal no longer names them. v0.2-eval-1's printed the
    offending seeds, which would have leaked them into a log.
  * Every harness calls it: `parseSeeds`, the session replay check, and
    newly the dynamic-staffing runner per seed and the evaluation
    recorder itself.
  * The only recorder for a holdout seed is `World.NewHoldoutRecorder`.
    It needs an `eval.Protocol` from `eval.Authorize`, which cannot be
    constructed outside the package.
* **The artifact contract** (`internal/eval/protocol.go`):
  * strategy files (`dreamteam.strategy/v1`), each optionally declaring
    a teleradiology-only parent;
  * a strategy set (`dreamteam.strategyset/v1`) pinning each file by
    digest and naming the scenario manifest and the action-free
    baseline;
  * the unseal record (`dreamteam.holdout-unseal/v1`), pinning:
    * the holdout manifest's bytes, the policy and the strategy set;
    * the exact engine;
    * the one primary strategy;
    * the approver, date and statement.
* **Authorize** validates every field and digest, decoding strictly. It
  requires every file committed with no change (`GitRepo`, through
  `git`), and the unseal record committed by its approver.
* **The runner** (`internal/sim` `TestHoldoutRun`, run with
  `DT_HOLDOUT_RUN=1`):
  * it computes the common horizon from the strategies' accepted
    actions before running, and the records confirm it on every seed;
  * it runs every strategy, the baseline included, on all 40 seeds;
  * it refuses a batch at or after T90;
  * it evaluates with the complete seed set and the pinned digests
    required (`eval.Input` `Seeds`, `PlanSHA`), one digest per plan;
  * it names the primary, and every other candidate is reported
    exploratory;
  * it writes `run.json` (every record's digest, the horizon and the
    artifacts) and `report.json` once, read-only, into
    `docs/evaluation/holdout-runs/<holdout>/`.

  A rerun must reproduce both byte for byte, or it fails and writes
  nothing. Without an unseal record the test skips as sealed.
* **The holdout manifest's protocol text** now describes this; its
  method, label and seed digest are unchanged.
* **Version.** 0.2.0-dev.11 registers 0.2.0-dev.10 (batch mode
  `v0.2_dev10`). The fixture `testdata/sessions/mini-seed33-dev10` was
  written by the dev.10 binary from a clean 5f9cb94 worktree. It holds an
  accepted contract, an overlap refusal and an accepted cancellation, and
  replays exactly and continues. MIGRATIONS.json lists the bump; every
  reference is unchanged.
* **Tests.** The protocol runs end to end on a synthetic two-seed
  holdout with its own label and range: the mini practice, the short
  policy, a baseline and a teleradiology-only strategy.
  * Authorization refuses in 14 ways:
    * a missing record;
    * an edited strategy, strategy set, policy or manifest;
    * a primary that is the baseline or unknown;
    * a bad date;
    * no statement;
    * a wrong schema;
    * an unknown field;
    * another committer;
    * uncommitted files;
    * another engine.
  * The run is written and then reproduced. A tampered report fails
    with nothing written, and a directory with other files is not
    written over.
  * A T90 batch, an incomplete matrix, a foreign strategy digest and an
    unauthorized protocol are refused.
  * Holdout seeds are refused by both recorders.
  * The evaluator's matrix rules and roles are tested.
  * `GitRepo` is tested on a scratch repository: committed, changed,
    untracked, nonexistent, first author.
  * The repository's holdout authorizes as sealed.
  * Two dev.10 replay tests.
* **Fail-before-fix.** 29 targeted mutations, each removing one check,
  were all caught. Two first survived and exposed weak tests:
  * the forged-protocol test passed on a nil-pointer panic, not the
    guard; it now has a complete forged protocol and checks the refusal
    message;
  * no test gave a nonexistent path to `Committed`; one does now.

  The dev.10 replay tests fail without the registration.
* **Verification.**
  * The full uncached suite on the Pi passes in 11 min 10 s (sim 639 s,
    session 206 s, cmd 134 s, eval 64 s, tui 37 s). Goldens pass with no
    reference changed, and every session file replays as
    `earlier_build_exact`, the new dev.10 fixture included.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * On the Cortex-A76, with the guarded harness, plan A–D validation
    (56 s) and evaluation (195 s) match the committed archive in every
    field but the engine name. The 180-day session replay check passes
    (165 s).
* **Not done (by design).** No strategy set is frozen and no unseal
  record exists: both are 5C.1's and the owner's. The canonical run has
  never been executed on the holdout. Acceptance was not rerun: it
  exercises only the simulation engine, whose production code is
  unchanged.
* **Remaining before unsealing.** One independent review of the whole
  corrected framework, the frozen strategy set, and the owner's unseal
  record. Open policy decisions are in `docs/evaluation/README.md`.

---

## V0.2 Slice 5C.0a — evaluation policy correctness and anti-gaming (0.2.0-dev.10)

An independent read-only Astra High audit of v0.2-eval-1
(`docs/audits/slice-5c0-eval-policy-audit.md`) gave CONDITIONAL GO for
development and NO GO for unsealing. It found seven high, five medium
and some low findings, and no critical one. The owner decided the open
questions and made this slice a prerequisite for definitive strategy
ranking.

This step corrects the evaluator and freezes a new policy version,
**v0.2-eval-2** (`docs/evaluation/policy-v0.2-eval-2.json`, rationale in
`docs/evaluation/README.md`, design §13.5). No strategy was run, ranked
or tuned. No holdout seed was generated or used. The simulation's
clinical and financial mechanics are unchanged.

* **Versioning.**
  * v0.2-eval-1 (`policy-v0.2.json`, `README-v0.2-eval-1.md`) is kept
    byte for byte and named by digest in the new policy's `supersedes`.
  * `LoadPolicy` refuses it as a policy to evaluate under. No evaluation
    was ever run under it.
  * The new policy lists every changed rule with its finding, rationale,
    old and new value and expected impact on eligibility. It also lists
    the deferred design measures.
* **H5, joint reliability.** A seed fails if any of its five per-seed
  service gates fails or lacks evidence, and at most 10% of seeds may
  fail jointly. Pooled rates still pass per window. The fixture:
  * six seeds each fail a different gate;
  * every gate alone is within its limit, which v0.2-eval-1 passed;
  * the union of six seeds fails.
* **H1, urgent evidence window.** Urgent is decided per seed over the
  whole gated period (t0, tailEnd]. The 12% gate is unchanged, pooled
  urgent is still gated in T90 and the tail, and soon and routine are
  unchanged.
  * The window was chosen by synthetic operating characteristics
    (`oc-v0.2-eval-2.json`, regenerated by
    `DT_OC=1 go test -run OperatingCharacteristics ./internal/eval`), at
    the owner's 8% planning rate, with every per-seed gate and the joint
    rule.
  * False rejection on 40 seeds is 0.030, against 0.981 for v0.2-eval-1's
    windows (independent studies). On 10 development seeds it is 0.069.
  * The Monte Carlo with week-level clustering (ICC 0.02) gives 0.37,
    and a 1-point seed SD gives 0.11. So the ≤ 5% aim holds only near
    independence.
  * That trade-off is reported, not hidden by weakening the gate. A
    per-gate dispersion index measures it on real seeds.
* **H2, common horizon.** One horizon covers every plan's tail rule and
  its latest commitment (a contract's minimum term). A record whose
  horizon or observation end differs, or falls short, makes the
  evaluation INVALID. `TailEnd` and `ObservationEnd` let a harness
  compute both before running.
* **H3, access followed through.** Due cohorts are taken by the original
  wanted time, wherever an order was received, with outcomes at the end
  of observation. Receipt-based loss is also followed to the end.
  Inherited orders are a guarded stratum. The day-89 receipt with a
  day-100 wanted date, and a referral lost after its window, are both
  regressions.
* **H4, backlog.** Every rota-aligned block after the reference, through
  the one covering the horizon, is checked against it for mean and peak
  backlog, overdue, stale (24 h past due) and exposure. No study may be
  more than 14 days past due. The fixtures cover:
  * a persistent stream of old work;
  * old work replacing new;
  * a spike that recovers;
  * late-tail deterioration;
  * an inflated reference.

  Each passes the old rule and fails this one.
* **M1/M2, case selection.**
  * Every threshold is an exact basis-point comparison, so equality at
    2%, 5 and 2 points passes. A non-whole basis point is refused.
  * Strata: per seed by priority and modality; pooled by priority,
    modality, their intersection and inherited orders; acquisitions by
    priority and modality.
  * A rare stratum is insufficient, never a pass, and fails if dropped.
    It blocks GO for priority, modality and inherited strata.
* **H7, verdicts.** GO also needs positive absolute operating profit at
  T90 and T180. Each comparison carries a classification: financial
  superiority, relative improvement, service purchase (only when the
  baseline fails service) or no benefit.
* **M3, money.**
  * The obligation bridge adds 14 days of contracted employee pay; its
    components are disjoint.
  * A funding gap at any capture or daily sample is a hard gate.
  * The recorder follows cash posting by posting and records when the
    minimum fell. On real runs it equals the evaluation window's minimum.
* **M4/M5, record integrity.** These are now integrity failures:
  * a missing, duplicate or mistimed capture;
  * a missing or misdated sample;
  * disordered study timestamps;
  * an order outcome inconsistent with its resolution time;
  * short observation;
  * missing payback data.

  Displacements are counted by event time. Labels name seed fractions;
  worst seeds are paired and identified, including the worst profit
  seed; means are over defined values only.
* **Recorder** (test-only, read-only). Record v2 adds:
  * order resolution and displacement times;
  * daily obligations;
  * the employee commitment;
  * the latest commitment and common horizon.

  A real-run finding: a no-show can call for a new date, so it is the
  one outcome allowed before a later resolution.
* **Version.** 0.2.0-dev.10 registers 0.2.0-dev.9 (batch mode
  `v0.2_dev9`). The fixture `testdata/sessions/mini-seed31-dev9` was
  written by the dev.9 binary from a clean e2ce1fd worktree. It holds an
  accepted contract, a `notice_too_short` refusal and an accepted
  cancellation, and replays exactly and continues. MIGRATIONS.json lists
  the bump; every reference is unchanged.
* **Tests.** 36 evaluator tests. 16 are new adversarial ones:
  * disjoint failing seeds;
  * common horizons;
  * carried-forward and inherited orders;
  * backlog safeguards;
  * exact tolerances and equality end to end;
  * rare strata;
  * a loss-making improvement;
  * an insolvent baseline without restoration;
  * the funding gap through the horizon;
  * record integrity (17 corruptions);
  * displacement timing;
  * worst seeds;
  * the obligation bridge;
  * dispersion.

  Alongside them: the operating-characteristic check, the real-run
  integration test with new assertions, and two dev.9 replay tests.
* **Fail-before-fix.** 27 targeted mutations, each reverting one fix,
  were all caught. The first run had three survivors:
  * the per-seed horizon comparison was redundant with the
    evaluation-wide check, so it was removed;
  * a referral lost after its window lacked a test, now added;
  * the persistent rule lacked a case only it catches (a steady stream
    of studies 20 days late), now added.

  The dev.9 replay tests fail without the registration.
* **Verification.**
  * The full uncached suite on the Pi passes in 11 min 40 s (sim 672 s,
    session 221 s, cmd 132 s, eval 48 s, tui 41 s). Goldens pass with no
    reference changed, and every session file replays as
    `earlier_build_exact`, the new dev.9 fixture included.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * On the Cortex-A76, plan A–D validation (75 s) and evaluation (205 s)
    under dev.10 match the committed archive in every field but the
    engine name, with the harnesses' seed guard now on v0.2-eval-2. The
    180-day session replay check passes (167 s).
  * Evaluator output is byte-identical between runs and input orders
    (`TestEvaluationDeterministic`).
  * No holdout seed was run: the guard refuses them, and no unseal
    record exists.
* **Not done (by design).** No strategy comparison was run, and the
  holdout is sealed and untouched. Acceptance was not rerun: it
  exercises only the simulation engine, whose production code is
  unchanged.
* **Remaining audit findings.**
  * H6, the holdout runner and artifact binding, is Slice 5C.0b.
  * M5's deferred measures are listed, not implemented.
  * The urgent dependence trade-off and the [proposed] numbers are open
    for the owner.

---

## V0.2 Slice 5C.0 — frozen evaluation framework (0.2.0-dev.9)

The owner accepted Slice 5B. This step freezes how V0.2 strategies will be
judged, before any staffing, reading-allocation or teleradiology strategy
is compared. It is a preregistration and measurement milestone: no
strategy was run, selected or tuned. No demand, reimbursement, staffing or
allocation economics, teleradiology price, read duration or action
semantics changed, and no simulation production code changed. Rules,
rationale and open decisions: `docs/evaluation/README.md`; design summary:
`docs/V0.2_DESIGN.md` §13.4.

* **Policy.** `docs/evaluation/policy-v0.2.json` (`dreamteam.evalpolicy/v1`,
  decoded strictly) holds every threshold, window, minimum and verdict rule.
  Reports carry its SHA-256.
* **Evaluator.** `internal/eval` is a pure package with no simulation
  dependency.
  * `Measure` turns a per-seed record into cohorts, access, remote, money
    and backlog measures, and cross-checks its sources: samples against
    studies, remote fees against the ledger, the cash bridge.
  * `Evaluate` applies the policy. It gives per-seed gates, pooled results
    with Wilson intervals and the worst seed, paired t 95% comparisons
    against the baseline (or between candidates), accrual and cash
    payback, and the verdict.
* **Recorder.** `World.NewEvalRecorder` (test-only,
  `internal/sim/export_eval_test.go`) samples a run daily and captures it
  at each horizon. It reads state only: a recorded run's state digest
  equals an unrecorded one's.
* **Service.** The owner's gates are unchanged: urgent ≤ 12%, soon ≤ 32%,
  routine ≤ 6%.
  * They are measured per seed and per priority, in T90 and in the
    frozen-policy tail.
  * A study breaches if it is not final at its due time, observed
    through a 24-hour closure.
  * Empty or thin cohorts are insufficient, never a pass. The minimums are
    urgent 43, soon 88 and routine 23, from a 5-point standard error at
    the gate.
  * At most 10% of seeds may fail or lack evidence, and the pooled rate
    must pass.
* **Cohorts.** All studies acquired in each window, finished or not:
  * completion;
  * completed-only TAT (never gating);
  * unfinished ages;
  * overdue count and study-minutes;
  * breaches and their denominators;
  * the historical as-of-end rate beside the gated one.

  Inherited backlog and orders are a separate population.
* **Backlog.** The single-sample day-90/day-180 rule, which plan D failed
  on a Monday-to-Sunday comparison, is replaced. The new rule compares
  28-day blocks aligned to the rota (week-A Mondays), and growth may be at
  most max(10%, 2 studies). Stages, ages, overdue fraction, flow and slope
  are reported. The old rule is reported as legacy, and the historical
  reports are untouched.
* **Money.**
  * T90 is primary, T30 feedback, D31–90 operating and T180 validation.
  * Solvency is the posting-level minimum cash, which may not fall below
    zero on any seed through the tail.
  * An obligation bridge (payables, payroll, severance, earned unposted
    wages, remote in-flight fees, contract minimums) is listed beside the
    P&L, never in it. Unbilled work is not cash, and there is no
    borrowing.
* **Payback.** Sustained break-even, accrual and cash, right-censored. A
  median is given only with at least half the seeds repaid.
* **Remote reading.** Local and remote reports, claims, handoff wait,
  turnaround by reader, retainer and read-fee expense from the ledger,
  and cost per remote report.
  * Faster reporting, additional reports and additional recognized
    revenue are kept apart.
  * The worst-seed breach rate by priority is set against cash drain,
    the owner's question for teleradiology.
  * A teleradiology-only variant must book identically.
* **Access and case selection.** Due-demand served fraction by priority
  and modality, per seed and pooled, plus acquisitions, lost referrals
  and walk-in abandonment, all against the paired baseline. A plan cannot
  pass the breach gates by declining cases.
* **Seeds.** Development is 901–910. 1001–1020 are exposed and
  reference-only.
  * A new **locked holdout** `v0.2-holdout-1` has 40 seeds from a
    published SHA-256 method, with no overlap with earlier sets. Its
    digest is recorded, and the seeds are listed nowhere.
  * Every calibration harness refuses its seeds (`eval.GuardRepo`) until
    an approved unseal record names the policy and a frozen strategy set.
  * It is run once.
* **Verdicts.**
  * INVALID: an integrity check fails.
  * NO GO: a hard gate fails or lacks evidence.
  * GO: a significant T90 benefit, T180 validation and no funding gap.
  * CONDITIONAL GO: a positive mean without every GO condition, or
    service restored at a stated cost.

  There is no weighted score.
* **Version.** 0.2.0-dev.9 registers 0.2.0-dev.8 (batch mode
  `v0.2_dev8`, the current action set and rules). The fixture
  `testdata/sessions/mini-seed29-dev8` was made by the dev.8 binary. It
  holds an accepted contract, an overlap refusal and a cancellation, and
  replays exactly and continues. MIGRATIONS.json lists the bump; every
  reference is unchanged.
* **Tests.** 19 evaluator tests on synthetic fixtures cover:
  * empty strata;
  * insufficient and undecided cohorts;
  * unfinished overdue studies;
  * rota-aligned backlog;
  * midnight and window boundaries;
  * inherited backlog;
  * remote and local reconciliation;
  * vendor cost;
  * transaction-level minimum cash;
  * payback censoring and the median rule;
  * case selection;
  * historical file hashes;
  * deterministic output;
  * the verdict rules;
  * booking invariance;
  * the frozen policy and holdout;
  * the repository guard;
  * the statistics.

  Alongside them: an integration test on real mini runs with and without
  a contract, and two dev.8 replay tests.
* **Fail-before-fix.** 21 targeted mutations were run:
  * 20 are caught by the intended tests;
  * one, the dev.8 replay calling `Session.Apply` instead of
    `World.Apply`, is equivalent (`Session.Apply` is `World.Apply`). The
    redundant case was removed.
* **Verification.**
  * Full uncached suite on the Pi passes in 11 min 31 s (sim 668 s,
    session 169 s, cmd 95 s, tui 29 s, eval 13 s). Goldens pass with no
    reference changed, and every session file replays as
    `earlier_build_exact`, the new dev.8 fixture included.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * On the Cortex-A76, plan A–D validation (55 s) and evaluation (197 s)
    under dev.9 match the committed archive in every field but the
    engine name. The 180-day session replay check passes (163 s).
* **Not done (by design).** No strategy comparison was run, no strategy
  was chosen, and the holdout is sealed. Acceptance (`-tags acceptance`)
  was not rerun: it exercises only the simulation engine, whose
  production code is unchanged.
* **Open decisions** (owner, before unsealing): breach decided at due
  time; the evidence minimums (the urgent tail cohort is close to its
  minimum on low-volume seeds); the 10% seed limit; the case-selection
  tolerances; the backlog rule; zero-tolerance solvency and funding gap;
  the GO conditions; the holdout's size and protocol; and the §13.3
  `DELTA_*` tolerances, which stay informational.

---

## V0.2 Slice 5B — teleradiology contract action and economics (0.2.0-dev.8)

The owner accepted Slice 5A. This step exposes the remote reading engine
through the player action `contract_teleradiology`, with §8.2's catalog
and §8.4's economics: retainers, read fees locked at claim, payables and
settlement, previews, observation, the TUI and replay. It invents no
price, plan or lane, and changes no demand, reimbursement, wage,
allocation, read duration, service threshold or frozen benchmark plan.
5C (evaluation and any strategy) has not started. Design and as-built
notes: `docs/V0.2_DESIGN.md` §5.13.

* **Action.**
  * `op start`: `requestId`, `plan` (`standard-v1`), `windows`, `scope`
    and `effective`. Activation is at least 14 days ahead and inside the
    booking horizon. The contract is open-ended, `T00001` onward.
  * `op cancel`: `contract` and an exclusive `endDate` at least 30 days
    ahead and at least 30 days after activation. Once fixed, the end
    cannot move.
  * Repeats are `unchanged`. A reused requestId with other terms is
    `request_conflict`, and a moved end `already_cancelled`.
  * Overlaps are judged on the timeline the whole batch leaves, so a
    cancellation and its successor can share a batch.
  * The batch is atomic like the staffing actions: a refusal leaves no
    contract, posting, event or log line.
* **Money**, in integer cents, with two appended expense accounts:
  * **Retainers** accrue per selected window per active day, at 00:00 after
    the day's payments: `floor(M×d/n) − floor(M×(d−1)/n)`. The minimum
    term is owed with no reads.
  * **Read fees** are locked at claim (night +20% at ≥ 22:00 or < 08:00),
    committed while in flight and recognized once at the report's final.
  * **Settlement:** both are paid by the existing first-of-month A/P
    payment, which covers everything accrued before it; an expiry erases
    nothing.
  * **Revenue** stays at report final.
* **Observation** (`Observation.teleradiology`, `Session.Teleradiology`):
  * contracts and their status, dates and terms;
  * coverage now, eligible waiting studies, and both lanes;
  * remote and local reports with their mean turnaround;
  * vendor money: recognized, paid, unpaid with its payment date, and
    in-flight fees.
  * Guaranteed amounts are listed apart from forecasts, and no future
    draw is read.
* **Preview:** `PreviewStaffing` covers the action, showing:
  * the guaranteed retainer through the earliest end, and the
    open-ended run rate;
  * a labelled upper bound from trailing observed demand: the studies
    that waited inside the windows in the last 28 days.
* **TUI:** the staff screen (`m`) has a Teleradiology section: plan and
  rules, contracts, coverage, lanes and the vendor balance. It has rows
  for a new contract (windows, scope, activation) and one cancellation
  row per open-ended contract, each previewed and submitted with enter.
* **Replay and compatibility.**
  * The new state is encoded only when a contract exists. The ledger's
    account arrays encode the two new accounts only when non-zero, so
    with no contract every state, log and ledger digest is as under dev.7
    and every golden reference reproduces.
  * 0.2.0-dev.7 is registered as an earlier build (batch mode
    `v0.2_dev7`). dev.5–dev.7 records replay with Slice 4's action set,
    in which the type was unknown.
  * A genuine dev.7 record (`testdata/sessions/mini-seed27-dev7`) holds
    an `unknown_action` and a `mixed_batch` refusal of the type. It
    replays exactly, fails with this build's action set, and continues.
* **Tests.** 28 targeted tests (27 in `internal/sim/telecontract_test.go`
  and a save/load test in the session package), three dev.7 record tests
  and a TUI test. Money is checked to the cent, including same-instant
  midnight and expiry cases and repeated submissions.
* **Fail-before-fix.**
  * Run against the dev.7 engine with only the new API types, 26 of the
    27 sim tests fail. The 27th, no-contract neutrality, passes before by
    design and is covered by mutation.
  * All 25 targeted mutations (M1–M25) are caught. They cover the rules
    on notice, term, overlap and atomicity; proration, expiry and night
    pricing; fee timing and settlement order; lanes and scope; previews;
    and the digest, mixed-batch and dev.7 registration rules. M19
    first failed to build; rewritten, it is caught by two tests.
* **Functional validation** (development seeds 901–903 only; not a
  strategy search; `docs/calibration/teleradiology-5b`).
  * Steps: a contract on all windows, then cancelled at its earliest end,
    then 90 days.
  * Every seed activates, claims backlog, reports every claim at
    catalog prices, and accrues retainers exactly ($6,100.01).
  * Vendor bills settle completely, and nothing is claimed after the end.
  * Ownership, ledger, payables and cash invariants hold daily. The cash
    bridge reconciles, and bookings are identical to the no-contract run.
  * Total reports equal the no-contract run's: here remote reads
    substituted for later local reads, at $27.7–29.4k of operating profit over 90 days (the vendor cost).
    That is for 5C to weigh, not a verdict.
* **Verification.**
  * Full uncached suite on the Pi: 9 min 58 s (sim 560 s, session 133 s,
    cmd 98 s, tui 28 s). All goldens pass, and all 11 session files
    replay as `earlier_build_exact`.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * Acceptance on the Cortex-A76: 2 h 35 min while shared, 396 subtests
    passed and none failed.
  * Without a contract the benchmark is unchanged. The plan D results
    under dev.8 (development, 66 s; held out, 220 s) match the committed
    archive in every field but the engine name. The 180-day session
    replay check passes (179 s).
  * Engine benchmarks, dev.7 against dev.8, show no measurable
    difference. On the Pi, Mini180 medians are 2.89 vs 2.94 s and
    EstablishedPlay90 1.34 vs 1.34 s. On the Cortex-A76 they are 946 vs
    970 ms and 387 vs 390 ms, inside a 3–12% run-to-run spread.
    Allocations are identical, and EstablishedPlay90 uses 0.4 MB/op less.
* **Limitations.**
  * There is one catalog plan, and a contract can be ended but not
    edited.
  * A batch cannot cancel the contract it starts.
  * The trailing-demand preview is an upper bound, not a lane model.
  * The evaluation window does not yet split reports, turnaround or cost
    by reader kind, or attribute waits to absent coverage. That is 5C's,
    frozen before any strategy.
  * The shift-end lapse of local reading debt remains.

---

## V0.2 Slice 5A — remote reading engine (0.2.0-dev.7)

The owner accepted dev.6 and authorized Slice 5 in three stages with a
checkpoint after each: 5A, the remote reading engine; 5B, contract
economics and the `contract_teleradiology` action; 5C, the evaluation,
with service and solvency rules frozen before any strategy and a new
locked holdout. This step is 5A only. No player action exists, and no
price, retainer, fee or account is added. Demand, reimbursement, wages,
allocations, read durations and service thresholds are unchanged. Local
read debt still lapses at shift end, now documented as a deliberate
limitation. Design and as-built notes: `docs/V0.2_DESIGN.md` §5.12.

* **Ownership.** A study's reader is a tagged, unexported value: none,
  an employed radiologist, or a remote read claim.
  * It encodes as one integer, exactly as before for on-site reads.
  * It is read only through `employee()` and `remote()`, so stale
    employee-index lookups no longer compile.
  * One owner from claim to final, through a shared finalization
    (`finalize`): one report, one revenue posting and one payer claim,
    whoever reads.
  * Employee debt and status stay on the employee; lanes have neither.
* **Two persistent lanes** (`internal/sim/remote.go`) serve every
  contract together.
  * Day start, closes and contract expiry never reset them. A lane is
    released only by its read's end.
  * A claim holds its lane for the ten-minute handoff and the study's own
    keyed read duration. Both ends are generation-checked completions.
  * The claim records its contract, lane, claim time, read-start time and
    night flag, for 5B's settlement.
* **Coverage** is a pure function of time.
  * It is the union of half-open windows (weeknights, saturday, sunday),
    intersected with the contract's dates. Exact starts are in and exact
    ends out.
  * Friday's night runs to Saturday 08:00. Saturday 08:00–14:00 and
    Monday 00:00–08:00 are gaps. Saturday and Sunday join at midnight.
  * Contracts can't overlap, so touching windows add no capacity.
  * Edge wakes fire only where coverage changes. They claim nothing;
    the dispatch after them may.
  * Remote coverage never touches booking.
* **Claiming** is dispatch's last pass, after the unchanged on-site pass
  and employed reading.
  * Idle lanes, in lane order, take the first eligible unread study in
    worklist order.
  * Urgent-only scope checks the original priority, not the overdue
    promotion.
  * There is no shift-end rule for lanes.
  * Reads finish after window end, contract expiry and midnight. The
    report belongs to its finalization date.
  * A successor contract uses only idle lanes.
* **Contracts before 5B.** Contracts have coverage terms and structural
  checks only: windows, a future activation, no overlap and ordered ends.
  Tests reach them only through `export_test.go`.
* **Compatibility.**
  * 0.2.0-dev.6 is registered as batch mode `v0.2_dev6`: this build's
    action set and rules, every digest verified. Earlier modes keep
    dev.5's rules.
  * Genuine dev.6 record `testdata/sessions/mini-seed26-dev6`, from a
    clean 3c71669 build: 100%/95%, a Sunday reduction refused as
    `commitment_too_short`, and a later change.
  * Before registration, the record was refused as an engine mismatch.
    Registered under dev.5's rules, it failed at batch 2.
  * It now replays whole and chunked, and continues as a verbatim prefix.
  * The remote state is encoded only when a contract exists, so every
    golden reproduces in every field.
* **Tests.** 14 in `internal/sim/remote_test.go`, 3 in
  `internal/session/dev6_test.go`. Fixtures are
  `internal/sim/testdata/micro-tele{,-local}.json`. They cover:
  * both lanes busy, a third study waiting, and the handoff holding a
    lane through a mid-handoff dispatch;
  * expiry during handoff and during a read, midnight, past a window's
    end, and the exact end with free lanes;
  * a successor sharing the lanes; adjacent windows and exact edges;
  * urgent-only by original priority;
  * local/remote exclusivity, with the employed reader first at a shared
    instant;
  * one report, revenue entry and payer claim per study;
  * save/reload as replay at a mid-handoff and a mid-read instant, across
    chunkings and event-budget stops;
  * booking invariance; §5.11's local debt conservation with lanes
    reading; edge dispatch neutrality.
* **Fail-before-fix (by mutation).** The 14 guards, from remote-first
  dispatch to an inclusive contract end, were each broken in turn with
  the tests unchanged, and each broken build failed its test (list in
  §5.12).
* **Verification.**
  * The full uncached suite passes on the Pi in 10 min 11 s (sim 589 s,
    session 138 s, cmd 113 s, tui 36 s), with `-timeout 60m`. All ten
    session fixtures, the new dev.6 one included, verify
    `earlier_build_exact`.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * Acceptance (`-tags acceptance`) passes on the Cortex-A76 in 2 h 11 min
    while shared (369 subtests).
  * The frozen plans A–D benchmark (`reading-dev`, `reading-eval`)
    reproduces the committed results in every field but the engine name,
    and the 180-day session replay check passes: no remote capacity
    without a contract.
  * Engine benchmarks, dev.6 against dev.7 interleaved: on the
    Cortex-A76, Mini180 has a median of 947 vs 959 ms/op and
    EstablishedPlay90 392 vs 388 ms/op, within the ≈7% run-to-run spread.
    The Pi results are noisier, with no measurable difference. Bytes and
    allocations per op are identical.
* **Limitations.**
  * No player can create a contract, and no session file can hold one,
    until 5B.
  * Observation names a remote reader only by lane. Remote share,
    coverage waits and turnaround by reader kind are 5B/5C reporting.
  * Worklist alerts describe employed readers only.
* **Next: Slice 5B**, contract economics and `contract_teleradiology`.
  Mode `v0.2_dev6` must then refuse the new type, as dev.6 did. Freeze
  the 5C evaluation rules before choosing any strategy. The owner plans
  an Astra review after 5B.

---

## V0.2 dev.6 — pre-Slice 5 remediation (0.2.0-dev.6)

An independent read-only audit of 13987e6 (0.2.0-dev.5) returned
CONDITIONAL GO, with two verified scheduler defects and an allocation
arbitrage that would distort Slice 5's comparison of internal reading
against outsourced reading. This step fixes those three and nothing else:
`contract_teleradiology` is not built, and demand, reimbursement, wages,
recruiting, read durations, service gates and the frozen plans are
unchanged. Design and as-built notes: `docs/V0.2_DESIGN.md` §5.11.

* **Fix 1: touching reading blocks.** At 100% the cycle's blocks touch.
  The next block's start (class 1) ran before the previous block's end
  (class 5) at the same instant, so the end closed reading and the
  radiologist claimed nothing for the rest of the shift.
  * `blockEnd` now continues reading when the day's next block starts
    exactly where this one ends.
  * The blocks, their sum and their boundaries are unchanged, so no
    capacity is added.
  * Reads stay nonpreemptive. Worklist order and the shift-end claim rule
    are unchanged.
* **Fix 2: overrun debt from time actually spent outside blocks.** dev.5
  charged the overrun only if a read ended outside a block, so a read
  running through a whole other-duty gap owed nothing. This affected
  allocations above 75%; the scenario and plan D allocations were not
  affected.
  * When a block opens while its radiologist is still reading, the gap
    since the last block ended is owed at once.
  * Repayment starts when the read ends, out of block time. What the
    block cannot cover carries forward as before, and debt lapses at the
    shift's end as since V0.1.
* **Fix 3: minimum allocation commitment (the Sunday arbitrage).** A
  version at 30% from a Sunday, restored on Monday, kept every shift's
  reading and cut the Sunday salary by $328.77 for Dr. Blake (about $904
  a Sunday for both radiologists at plan D's shares).
  * An allocation now stays in force at least 14 calendar days
    (`MinCommitDays`) before the next change for that radiologist.
  * The check runs on the timeline the whole batch leaves, after every
    item's own checks, so item order doesn't matter.
  * The refusal is `commitment_too_short` on the first item that makes
    the short pair; the rest of the batch is `batch_rejected` and nothing
    changes. Previews agree.
  * Replacing a pending version's value passes. Under this rule a
    withdrawal can only lengthen a pair.
  * Pairs already on an earlier build's timeline are grandfathered, but a
    withdrawal that would create a new short pair is refused.
  * Notice, effective dates, horizon, bounds, history, no-op rules and
    atomic mixed batches are unchanged. No new charge or account.
* **Tests (fail-before-fix).** All seven new tests fail with the fixes
  disabled and the tests unchanged.
  * `TestTouchingBlocksReadContinuously`: 4 and 5 claims instead of 18,
    for 30- and 20-minute reads.
  * `TestOverrunThroughOtherDutiesIsOwed` (95%, a read claimed at cycle
    minute 94 crossing 95–100, reads ending exactly on a block start or
    end): failed at the dev.5 claim minutes.
  * `TestOverrunThroughSeveralGapsIsRepaidLater` (30%, a 205-minute read
    crossing two gaps): next claim at 11:25 instead of 18:25.
  * `TestReadingTimeConservation`: 12, 12, 17 and 428 claims while
    owing.
  * The three tests in `allocation_commit_test.go`: the arbitrage, 13-
    against 14-day spacing and the earlier-timeline rules were all
    accepted.

  The micro fixture is `internal/sim/testdata/micro-blocks.json`: a
  saturated worklist across seven boundaries.
* **Conservation.** `TestReadingTimeConservation` keeps an independent
  account of every radiologist-day, built from the studies' claim and
  final times and the day's plan. It requires:
  * every minute outside a block borrowed once;
  * repayment, in order, only from later block time spent not reading;
  * no claim while time is owed or outside a block;
  * what is still owed at the shift's end lapses.

  Six weeks of the established practice at 100%/95%: 303,438 s borrowed
  = 297,557 repaid + 5,881 lapsed, with no claim while owing. Under
  dev.5's rules there were 428 claims while 2,790 minutes were owed.
* **Compatibility.** The fixes change outcomes, so this is not a digest
  migration.
  * Every record from an earlier build replays under that build's rules
    (`SetDev5Rules`, set by `session.Replay` until the record's end).
  * 0.2.0-dev.5 is registered as batch mode `v0.2_slice4`, with every
    digest verified.
  * There are two genuine dev.5 records from a clean 13987e6 build
    (`testdata/sessions/mini-seed24-dev5`, `mini-seed25-dev5-reading`).
    Unregistered, both were refused. Registered without dev.5's rules,
    the first fails at batch 2, and the second's log and run digests
    differ.
  * Both replay whole and chunked, and continue as a verbatim prefix
    under this build's rules.
  * A record claiming dev.5 but needing dev.6 rules is refused.
  * Unknown actions, v1 and v2 records, and dev.2–dev.4 records behave
    as before.
  * One existing session test moved a change from Nov 9 to Nov 10, from
    13 days apart to 14.
* **Neutrality.**
  * All seven golden references reproduce. MIGRATIONS.json lists the
    Slice 4 reference, now historical, with no changed field.
  * Plans A–D rerun under dev.6 reproduce `results/reading-dev` and
    `results/reading-eval` in every field but the engine name.
  * All nine session files (the benchmark's dev.4 and dev.5 files and
    the testdata records) verify as `earlier_build_exact`.
  * No plan was retuned and no seed was used to choose anything.
* **Verification.**
  * Full uncached suite on the Pi passes: sim 544 s, session 111 s, cmd
    87 s, tui 24 s.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * Plan session replay, A–D × seeds 901 and 1001 × 180 days, whole and
    in 13-hour chunks: 171 s on joofi.
  * Acceptance on joofi passes: staffing and roster invariants (50 seeds
    × 90 days each, after every transaction), turnover, churn, V0.1
    invariants and 1,000 chunkings. The first run hit a 120-minute
    timeout I had set (Slice 3 needed 128 min), during the V0.1 criteria;
    those were then rerun alone and passed in 42 min.
  * Performance (dev.5 against dev.6, interleaved, joofi): Mini180 917
    against 937 ms/op and EstablishedPlay90 376 against 386 ms/op
    (medians), inside a ≈15% spread. Allocations are identical. Details
    are in MEASUREMENTS.
* **Limitations.**
  * Debt still lapses at shift end, as in V0.1, now measured: 98 minutes
    in six weeks at 100%/95%.
  * The commitment is per radiologist only and does not price early
    exit.
  * The service-gate, backlog-rule and turnaround audit findings (Medium
    and below) are open, for Slice 5's pre-registration.
* **Next: Slice 5, `contract_teleradiology`.** This needs persistent
  remote reading ownership, service and solvency rules set before any
  strategy, liquidity as an independent constraint, and a fresh locked
  holdout. Seeds 1001–1020 stay as reference data.

---

## V0.2 Slice 4 — reading allocation (0.2.0-dev.5)

The owner approved `set_reading_allocation` after the dynamic staffing
benchmark, and nothing else (`contract_teleradiology` comes later).
Design and as-built notes: `docs/V0.2_DESIGN.md` §5.10. Benchmark:
`docs/calibration/dynamic-staffing/` README, "Plan D". No reimbursement,
demand, wage, reading-duration, equipment or opening-hour input changed.

* **Action.** `set_reading_allocation` changes an employed salaried
  radiologist's site share of paid time.
  * **Ops:** `op: set` / `remove`, `employee`, `effective`,
    `siteAllocationBP` 3,000–10,000.
  * **Dates:** any weekday at least 14 days ahead, within the booking
    horizon. Removal needs the same notice.
  * **Versions:** dated, like roster versions. A pending version can be
    replaced or withdrawn; versions in force or past never change; an
    append-only log keeps every operation. A set equal to the allocation
    in force, or the removal of an absent version, is `unchanged`; two
    writes to one radiologist and date in a batch are
    `invalid_parameters`.
  * **Distinct from contracts and rosters:** radiologists have no
    contract hours, and their shifts stay immutable.
* **Reading and supervision.**
  * The allocation in force each day sets that day's site reading blocks
    through the existing 100-minute cycle inside the same shifts.
    Worklist order, overrun debt, one read at a time, the p100-by-shift-
    end claim rule and origin-keyed read times are untouched.
  * Contrast supervision is presence on site and does not move (tested).
  * Booking coverage segments were also cut at reading-block boundaries,
    which allow shared-capacity handovers. They now keep the scenario
    allocation's boundaries (`DayPlan.cutReading`), so an allocation
    never changes technologist booking capacity and displaces nothing.
    The test that fails without this covers days built after the change
    and days a later hire rebuilds.
* **Money.** The existing daily salary line uses the allocation in force
  that day: daily rounding, benefits, and payday settlement through
  accrued payroll. One accrual per radiologist per day, by its key, so no
  time is charged twice. No new account or entry kind.
  * Tested to the cent across two changes, with the cash bridge
    reconciled.
  * Previews show the salary run-rate change per 30 days as a guaranteed
    obligation, with the date until which it can be withdrawn at no
    cost.
* **Batches and compatibility.**
  * The new type mixes atomically with `hire`, `terminate` and
    `set_roster`; any invalid item refuses the whole batch.
  * A `mixed_batch` message names `set_reading_allocation` only when the
    batch holds one, so every recorded message and golden is unchanged.
  * Allocation versions are encoded in the state digest only when
    present, so no-action state digests are unchanged too.
  * 0.2.0-dev.3 and dev.4 records replay with Slice 3's action set (batch
    mode `v0.2_slice3`). A genuine dev.4 record
    (`testdata/sessions/mini-seed23-dev4.*`, from a clean af475fd
    worktree) has the new type refused as `unknown_action` and
    `mixed_batch`. It failed before registration, and failed again under
    this build's action set; it now replays with every digest verified
    and continues as a prefix. The benchmark's dev.4 session files verify
    as `earlier_build_exact`.
* **Interface.**
  * **API:** `Employee.SiteBP` is the allocation in effect;
    `Employee.AllocationChanges` lists pending ones.
    `World/Session.ReadingAllocations()` gives each radiologist's
    timeline with site reading minutes per rota week and salary per 30
    days. `StaffingPreview.Allocations` gives before and after, the
    withdrawal date and the reporting backlog.
  * **CLI:** action files take the new type; golden
    `established-seed1-60d-allocation.json`.
  * **Session v2.**
  * **TUI:** staff screen rows per radiologist (←→ share in 2.5-point
    steps, `,` `.` date, `x` withdraw), the backlog line and a read-only
    preview.
* **Tests (fail-before-fix).** Nine `internal/sim/allocation_test.go`
  tests failed before the action existed. They cover:
  * date boundaries; bounds and shape; versions;
  * salary proration; supervision and booking invariance;
  * reading rules with urgent order; mixed batches; chunking; previews.

  Session tests cover save and replay across effective dates (whole,
  chunked, event-budgeted), a v1 record with the type name and the dev.4
  record. There are also a TUI test and the new golden.
* **Plan D benchmark (20 held-out seeds 1001–1020, 180 days).** D is plan
  C's staffing plus Dr. Avery 62.5% and Dr. Blake 52.5% from Tue Oct 27,
  the earliest legal date. D and its decision rule were fixed before any
  run.
  * **Reproduction:** A, B and C reproduce the original results exactly.
  * **D against C:** +$13.7k radiologist salary, +74 ± 20 reports, $12.5k
    of backlog revenue recovered. Profit −1.5 ± 3.2 $k at T180 and +0.6 ±
    1.2 $k per 30 days in steady state. Imaging and access are identical.
  * **Breach rates days 91–180 (urgent / soon / routine):** D 9.2 / 31.9
    / 4.4%, C 52.7 / 88.2 / 55.3%, no action 10.7 / 38.1 / 7.2%. The
    gates are met on the mean, but on all three gates only on 7/20 (T90)
    and 11/20 (days 91–180) seeds.
  * **Against no action:** +12.6 ± 1.4 $k at T180, break-even day 141
    (113–164) on 20/20 seeds, cash payback by T180 on 1/20.
  * **Decision: CONDITIONAL GO**, by the pre-registered rule. Its one
    failing test compares single backlog samples on day 90 and day 180;
    no action fails it too, and D's 30-day mean backlog is flat.
    Conditions:
    * the gates are met on average only;
    * the soon rate is at its gate;
    * the allocation buys service at about its own cost.

    No other allocation was tried on the held-out seeds.
* **Verification.**
  * The full uncached suite passed on the Pi in 9 min 1 s (sim 537 s,
    session 94 s, cmd 84 s, tui 27 s). An earlier run hit Go's default
    10-minute timeout in `sim` while the Pi was also compiling; rerun with
    `-timeout 60m` it passed.
  * `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * All seven golden references reproduce: the six historical ones under
    `MIGRATIONS.json` as before, the new one byte for byte.
  * `dreamteam verify` replays the three dev.4 benchmark sessions
    (`earlier_build_exact`) and D's.
  * Every plan replays whole and in 13-hour chunks (seeds 901 and 1001,
    180 days).
* **Limitations.**
  * One allocation was tested.
  * Gate pass rates per seed are modest.
  * The rule's backlog test uses point samples.
  * No recruiting uncertainty; ×1.30 demand only; 180 days; arm64 only.
  * Teleradiology, the alternative for marginal and evening reads, is not
    built yet.
* **Next: Slice 5, `contract_teleradiology`.** Then test substitution
  against allocation, on development seeds first.

---

## V0.2 dynamic staffing benchmark (0.2.0-dev.4)

The owner's milestone after Slice 3: can the sustainable scenario's shared
start be improved with the implemented actions alone, and at what
transition cost and payback? Benchmark: `docs/calibration/dynamic-staffing/`
(README, plans, per-seed results, reports, trajectories, sessions). Design:
`docs/V0.2_DESIGN.md` §5.9. **No engine behaviour, economics, demand,
wages, hours or patient behaviour changed, and no action was added.**

* **Plans.** `plans.py` writes them as fixed timed action batches, applied
  through `World.Apply` identically on every seed. Each submits one atomic
  batch at the takeover with standard offers; V0.1 controls stay at their
  defaults.
  * **A, no action.**
  * **B, partial:** convert the MR evening. `hire` a 20 h 16:00–20:00 MR
    technologist and `terminate` Alex (40 h) once it is productive.
  * **C, full:** the closest legal path to Step 0C's `DED-MRpt`. It hires a
    CT day core (40 h), an XR/CT evening (27.5 h) and an MR evening (20 h);
    terminates Sam and Alex; and moves Riley to X-ray only with `set_roster`
    from Monday Nov 23.
  * **Feasibility:** every step is legal (lead, training, notice,
    headcount 8 of 8, Monday effective date, no coverage gap). Only Step
    0C's reading allocation awaits Slice 4.
* **No overfitting.**
  * B was chosen among three candidates (MR conversion, Step 0C's XR/CT
    float, both) on development seeds 901–910. The rule was written into
    `plans.py` before any run: the best T180 increment with the urgent
    breach rate at most 5 points worse. The float candidates failed that
    rule.
  * The evaluation used 20 held-out seeds, 1001–1020, never used before
    (Steps 0–0C used 1–20), for 180 days.
* **Validation, every seed:** all actions were accepted; every lifecycle
  date matched the plan; at most 8 technologists; no inconsistent booking
  at any endpoint; the cash bridge reconciled; labor postings summed to
  the window's labor to the cent.
  * C displaced one CT contrast appointment on 2 of 20 seeds, rebooked the
    same day: a break-relief cover was needed in X-ray.
  * Each plan replays authoritatively from a saved session, whole and in
    chunks, to the same digests.
* **Results (20 held-out seeds, Δ against the paired no-action run).**
  * **B:** costs $9.8k (recruiting 3.0, severance 4.5, training 2.2). Δ
    profit at T30 / T90 / T180 is −8.0 / −2.0 / **+12.9 ± 0.1** $k, and
    +$5.0k a month in steady state. Break-even is **day 101** (100–106) on
    20/20 seeds; cash pays back on 20/20 (median day 114). Service is
    unchanged.
  * **C:** costs $23.7k (recruiting 9.0, severance 8.4, training 6.2). Δ
    profit at T30 / T90 / T180 is −19.0 / −10.4 / **+14.1 ± 2.7** $k, and
    +$8.2k a month. Break-even is **day 134** (113–175) on 20/20 seeds;
    cash pays back on 11/20 by T180. Lowest cash is $20.7k.
  * **C, days 91–180:** imaging +265 exams, lost referrals −179, walk-ins
    leaving −90. But reading cannot keep up:
    * urgent, soon and routine breach rates are 52.7%, 88.2% and 55.3%,
      against 10.7%, 38.1% and 7.2% with no action;
    * the backlog is 93 against 30 and still growing.
  * **Guaranteed pay** is negligible in both plans. **Unrecovered demand:**
    US "not rostered" losses (≈405 per 90 days) are untouched by every
    plan.
* **Decision: CONDITIONAL GO.**
  * Staffing actions give measurable agency with credible payback;
    transitions are complete by day 42 and do not call for a different
    starting state.
  * The full reorganization's service needs `set_reading_allocation`.
  * Next: Slice 4, then rerun the same frozen plans on the same held-out
    seeds with C plus an explicitly defined reading adjustment.
* **Code (test and session compatibility only).**
  * `internal/sim/calib_dynamic_test.go` is the harness (build tag
    `calibration`). It reads `benchmark.json` and `plans/`, samples daily,
    measures endpoints and spans, decomposes labor, and checks
    lifecycles. `calib_export_test.go` is a read-only ledger export for
    tests.
  * `internal/session/calib_dynamic_test.go` saves and replays every plan.
  * `report.py` builds the paired tables, payback and `trajectory.csv`.
* **Version and compatibility.** 0.2.0-dev.4 registers 0.2.0-dev.3 as an
  earlier V0.2 build: batch mode `v0.2_slice3`, the same action set.
  * A genuine dev.3 session file, `testdata/sessions/mini-seed22-slice3.*`,
    built from a clean 5fdfb95 worktree, replays with every digest and
    continues as a prefix (`TestSlice3RecordReplaysAndContinues`). It
    failed as an engine mismatch before the registration (fail-before-fix).
  * The Slice 3 golden is listed in `MIGRATIONS.json` with no changed
    field.
* **Verification.**
  * The full uncached suite passed on the Pi in 9 min 6 s; `go vet` is
    clean for the default, `acceptance` and `calibration` tags; gofmt is
    clean.
  * Benchmark results from the dev.3 and dev.4 binaries, on the Pi and on
    the Cortex-A76, are identical in every field but the engine name.
* **Limitations.**
  * One hand-authored plan per strategy, not optimized; premium offers and
    US coverage untested.
  * No recruiting uncertainty in the model.
  * Demand fixed at ×1.30 (×1.15 not rerun).
  * 180-day horizon.
  * Accrual expenses fees and severance at acceptance, and revenue waits
    for the report.
  * The provisional service gates are missed by every staffing-only plan,
    including no action on soon and routine.
  * amd64 not run.

---

## V0.2 Slice 3 — roster changes (0.2.0-dev.3)

The owner approved `set_roster` (2026-10-15). `set_reading_allocation` and
`contract_teleradiology` are not in this slice. Design: `docs/V0.2_DESIGN.md`
§5.8, with the deviations identified there. Without a `set_roster` action
every earlier outcome is unchanged.

* **Roster versions.**
  * An hourly technologist's or front desk employee's roster is now a
    timeline: the original (scenario or hire, immutable) and dated versions.
    Each version is a full A/B pattern from a Monday, in force until the
    next.
  * `op: set` inserts a version or replaces a pending one; `op: remove`
    withdraws a pending one, and its predecessor resumes. Several
    successive versions are allowed.
  * Versions in effect or past never change. Every operation, including
    replaced, withdrawn and voided versions, stays in the employee's roster
    log.
  * The version for a date is resolved behind the per-day registry view
    that the plan, the runtime and the booking predicates already use.
* **Validation** (in order): shape; employee (hourly technologist or front
  desk, still employed); effective date (`past_date`, `not_monday`, 14 days'
  notice, horizon, before the end of employment); each shift as for a hire
  (rooms of the employee's skills, desk for the front desk, breaks, cap);
  `unchanged` when equal to the roster already in force; then the
  whole-roster rules — at most six days a rota week, **net rostered time
  ≤ the contract** in each rota week (the employee's own original roster
  may always be restored), rest. Then the batch's composed timelines are
  checked as a whole:
  * no version after a termination (in either batch order);
  * rest across every version boundary, withdrawals included;
  * a trainee still completing training within 60 days
    (`training_incomplete`);
  * every approved extension still valid on its new shift
    (`extension_conflict`, naming the date: refused, never silently
    cancelled).
* **Rest rule, one for hiring, rosters and overtime:** 10 hours from a
  shift's cap — the latest permitted release, min(start + 12 h,
  23:59:59) — to the next day's start. Approved extensions and hold-over
  cannot pass the cap, so no approval can break the rule. Tested at the
  day boundary within a pattern (A→B and B→A), across versions, after a
  withdrawal, and at exactly 10 hours.
* **Contract and money.** Contracted hours, rate, accrual weights, notice
  and severance never change with the roster; nothing new is posted. A full
  week under a four-day or a zero-hour roster pays Riley's contract exactly
  ($1,440 + benefits); severance is identical before and after a cut.
* **Atomic batches and displacement.** A batch commits in input order,
  moves trainees' productive dates, then reconciles once, against the final
  timeline, from the earliest affected day: the same survivor and
  re-offer lifecycle as Slice 2. Causes: `employment_ended`,
  `roster_changed`, `training_delayed` (joined by `+`). A coordinated
  exchange (an XR+CT hire and Sam swapping Monday and Tuesday) displaces
  nobody in either order, while each half alone displaces 11 bookings.
  A termination voids pending versions on or after the end (logged), as
  it voids extensions (unchanged).
* **Training.** A roster change before the first working day or during
  training recomputes the productive date from the composed timeline (past
  days are the attended ones). A delay displaces the bookings that relied on
  the trainee (including XR bookings losing a shared technologist); an
  earlier completion opens capacity. Capacity is never granted before
  completion (invariant).
* **Preview.** Rostered hours before and after against the unchanged
  contract, the interval, a trainee's productive date before and after, the
  extensions kept, the batch's slots opened and closed by modality, exact
  displacements with provisional replacements, and the guaranteed weekly
  contract cost apart from forecasts (contracted time not rostered,
  extensions kept, displaced bookings). Pure and independent of the hidden
  future (tested).
* **Interfaces.** `api.Action` gains `op` and `effective`;
  `api.Employee.RosterChanges`; `World/Session.RosterPlan`; CLI actions
  files; session v2; the TUI roster editor (staff screen `m`, enter on a
  roster row): weekly or fortnightly grid, field editing, effective Monday,
  withdrawal, live preview.
* **Compatibility.**
  * State digest still `dreamteam.state/v3`: versions are encoded only when
    present.
  * A genuine 0.2.0-dev.2 save, built with that commit's binary and kept
    in `testdata/sessions`, replays under this build with every digest
    equal. Its batches apply with Slice 2's action set (`ApplySlice2`),
    where `set_roster` was an unknown action. Before the version bump the
    same file failed replay (fail-before-fix). Continued, it becomes the
    prefix of a new v2 record
    (`prefix:earlier_build_exact+native_exact`).
  * v1 → v2 continuation with roster changes replays.
  * The four V0.1 goldens are unchanged under the manifest. The Slice 2
    golden differs only in its action and run digests: its mixed batch's
    refusal message now names `set_roster`. This is recorded in
    `MIGRATIONS.json` and confirmed by diffing the dev.2 and dev.3 action
    records (log, ledger and state identical).
  * A sixth reference, `established-seed1-60d-roster.json`, pins the roster
    plan (36 displacements, all rebooked).
* Verification:
  * Full uncached suite (`go test -count=1 -timeout 60m ./...`) passes on
    the Pi in 8 min 19 s: cmd 86 s, session 88.5 s, sim 488 s, tui 24 s.
    `go vet` is clean for the default, `acceptance` and `calibration`
    tags; gofmt is clean.
  * Acceptance (`-tags acceptance -run Acceptance`) passes on joofi in
    128 min: the V0.1 criteria, staffing turnover (155 hires over 720
    days) and roster churn (340 roster versions accepted over 720 days),
    with the invariants checked daily; 156 tests and subtests, no failure.
  * All six golden references and the Slice 2 session fixture give
    byte-identical outputs on the Pi and on joofi.
  * Fail-before-fix mutations: all 20 guards were caught (not_monday,
    notice, contract, pattern rest, boundary rest, extension conflict,
    extension on the composed view, training projection, trainee capacity
    early, void on termination, composed not_employed, duplicate date
    claim, unchanged, version resolution, reconcile after roster,
    per-item reconcile, guarantee after a roster cut, digest omit-empty,
    Slice 2 batch mode, original-roster restore only when exact).
* Performance (Pi, `docs/MEASUREMENTS.md`): the 90-day run is unchanged
  (best of 8 interleaved: 3.40 s dev.2, 3.41 s dev.3); EstablishedPlay90
  allocations unchanged (78.6 MB, 1.241M). A roster preview costs
  28.2–28.3 ms, a commit 25.3–25.7 ms; the editor renders in 48 ms.
* Limitations: the rest rule is conservative (measured from the cap, see
  deviation 1); break rules are not applied to approved extensions;
  radiologist and supervision displacement remain untestable until the
  reading actions; the result schema is still v1; amd64 and macOS were
  not run; coverage differences are reported per batch, not per change;
  the TUI edits only the first cover of a shift.

---

## V0.2 Slice 2 — hiring and termination (0.2.0-dev.2)

The owner approved two player actions, `hire` and `terminate` (2026-10-15).
`set_roster`, `set_reading_allocation` and `contract_teleradiology` are not
in this slice. Design: `docs/V0.2_DESIGN.md` §5.7, with the deviations
identified there. Without a V0.2 action every V0.1 outcome is unchanged.

* **Registry and identity.**
  * Hires get the first unused `H00001`, `H00002`, … (scenario IDs skipped),
    allocated at commit only.
  * A hire's terms are frozen in state; records are append-only and never
    reused.
  * Staffing sets are growable (no 64-employee mask); the scenario limit is
    a 1,000-employee sanity bound.
  * The state digest encoding is `dreamteam.state/v3`.
* **Hire.**
  * Validated in a fixed order: offer table (role, skills, exact standard or
    premium rate), contract 1–2,400 minutes, start ≥ today + the tier's lead,
    roster rules, 40 h (technologist) or 24 h (desk) of training within 60
    days, and the 8-technologist / 4-desk caps including notice.
  * The recruiting fee posts at acceptance (recruiting / A/P).
  * From the start date the employee is paid (contract guarantee in equal
    sevenths, training as worked time) but works under a training view with
    no capacity: no room, registration, supervision, hold-over or extension.
  * Independent from the midnight after the 40th (24th) attended hour.
    Future slots open from that date.
* **Terminate.**
  * Hourly technologists and front desk only.
  * Last day ≥ 14 days after today and the start; identical repeats are
    `unchanged`; a moved or withdrawn notice is refused.
  * Contract pay continues to the last day, whatever the roster; the last
    week closes on the last day.
  * Severance (two contracted weeks with benefits) is recognized at
    acceptance (severance / severance payable) and paid on the first payday
    after the last day.
  * Approved extensions after the end are voided with a log line.
  * Terminating a pending hire keeps the fee, runs the contract from the
    start and owes severance.
* **Displacement** (when the batch commits; design §5.7):
  * The days after an employee's end are re-planned from the final registry
    (a replacement hired in the same batch counts).
  * Bookings are re-inserted in survivor order with the booking rule's own
    predicates; those that cannot stay are displaced, all released first,
    then offered a same-category slot no earlier than their original time,
    accepted within the patient's tolerance of the original tolerance
    anchor, else lost to displacement.
  * The behavioral attempt, confirmation and no-show lead basis are kept, so
    no new random draw is made.
  * Each displacement has a record and its own counters, outside demand,
    loss, cancellation and no-show figures.
* **Atomic batches.** A batch with a V0.2 action holds only V0.2 actions
  (`mixed_batch`); one refusal refuses all (`batch_rejected`) and commits
  nothing; a valid batch commits in input order and reconciles once. V0.1
  batches are unchanged.
* **Finance.** New accounts `recruiting`, `severance`, `severance_payable`
  (listed only once they hold a balance, so the baseline digest is
  unchanged); statement lines; labor includes severance; the cash bridge
  and cash identity include severance; severance due appears in
  obligations.
* **Previews.** `PreviewStaffing`: validation, terms, timeline, slots a hire
  opens, exact potential displacements with capacity-only provisional
  replacements, guaranteed and forecast money in separate tables. Pure.
* **Session v2.** `dreamteam.session/v2`; v1 read by separate strict DTOs;
  an engine registry (`0.1.*`, `0.2.0-dev.0.*`, `0.2.0-dev.1`) replays v1
  records with outcomes verified; a continued v1 session saves as v2 with
  the v1 record embedded and this build's transition digests.
* **Interfaces.** API (`api.Action` fields, `ValidateBatch`,
  `PreviewStaffing`), CLI actions files and replay, TUI staff screen `m`.
* **Compatibility and neutrality.**
  * The four historical goldens are byte-for-byte unchanged. They still differ
    from this build in exactly `digests.state`, `digests.run` and
    `replay.runDigest`; every outcome field, the log, ledger and action
    digests and the baseline digest are identical (90 days and the V0.1
    action fixture included). `MIGRATIONS.json` now records this build's
    new values (`dreamteam.state/v3`) with the explanation.
  * A fifth reference, `established-seed1-60d-staffing.json` (V0.2 actions,
    written by this build), pins hire and terminate end to end.
  * The Step 0C sustainable start and lead candidate reproduce their
    committed per-seed and aggregate results exactly (9 seeds each).
  * All five references are byte-identical on the Pi 4 (Cortex-A72) and the
    Cortex-A76 machine. amd64 and macOS not run (no machine).
* **Regression tests.** New: `staffing_test.go` (lifecycle and training
  capacity, the §5.4 partial-week example to the cent, notice and severance
  with a zero roster, every rejection code, repeats and conflicts, atomic
  batches, headcount caps with notice, displacement records and preview
  agreement, a replacement in the same batch displacing nobody,
  shared-technologist displacement, tolerance anchor over 12
  terminations, read-only and hidden-future-independent previews, more than
  64 employees, the evaluation endpoint), `empset_test.go`,
  session `staffing_test.go` (save/replay at every lifecycle boundary,
  chunked and event-budgeted replay, v1 → v2 continuation and tampering),
  TUI `staff_test.go`, and two acceptance-tagged checks (invariants after
  every transaction for 50 seeds with the reference plan; 720 days of
  turnover past 64 lifetime employees, reproducible).
  * **Fail-before-fix evidence by mutation**: each guard was disabled in
    turn and its test run; all 17 fail as required — trainee capacity,
    guarantee from the start date, severance posting, severance payday,
    notice pay, reconciliation, attempt preserved, tolerance anchor (at the
    call site and in the rule), shared-technologist survivors, batch
    atomicity, notice in the cap, training required, v1 transition check,
    preview purity, sets past 64, v1 replay semantics for V0.2 type names. The first anchor test did not catch a
    more lenient rule; `TestDisplacementToleranceAnchor` and
    `TestAcceptReplacementAnchor` were added and do.
* **Remaining limitations.**
  * No `set_roster`: a hire's roster is fixed at hire; contracts change only
    by termination and hiring (owner decision).
  * Contrast-supervision displacement cannot arise yet (radiologists cannot
    be terminated); the path is shared with the tested staffing check.
  * The result schema stays `dreamteam.result/v1` with additive fields;
    the T90 vector and gates are not built.
  * Hire names are their IDs. The TUI offers fixed roster patterns, not a
    roster editor.
  * amd64/macOS determinism not yet run.
* **Performance** (MEASUREMENTS.md): about +5% CPU and +7–12% memory on
  a 90-day run (best of ten on a loaded, throttled Pi: 3.20 → 3.49 s); a
  termination preview ≈21–24 ms and its commit ≈30 ms; the staff screen
  renders in ≈25 ms with a live preview.
* **Verification:**
  * Full uncached `go test -count=1 -timeout 30m ./...` passed: 7 packages,
    8.5 min on the loaded Pi (sim 503 s, cmd 83 s, session 75 s). `go vet`
    clean for the default, calibration and acceptance tags; gofmt clean.
  * Goldens: the four historical references match under the migration
    manifest (90 days included); the V0.2 reference is byte-identical;
    `-verify-replay` matched on all five.
  * Acceptance on the Cortex-A76 (`-tags acceptance` test binary), all
    passed, ≈1 h 31 min with the suites sharing four cores: V0.1 criterion
    10 (50 seeds × 90 days, invariants after every transaction) and
    criterion 9 (1,000 chunkings, 20.7 s); new: invariants after every
    transaction for 50 seeds × 90 days with the V0.2 reference plan, and
    720 days of turnover — 155 hires, registry 163, slot staff sets three
    words wide, reproducible (44.5 s).
* **Slice 3 prerequisites (exact).**
  1. Date-aware roster versions: `planFor` composes registry terms by
     date (`dayTerms`); `set_roster` adds dated versions behind it, with the
     §6.4 composition (version keys, insert/replace/remove, tombstones).
  2. Reuse `reconcile` for roster reductions (cause `roster_reduction`)
     and `recoverDays` for additions; extensions that no longer fit a new
     version must reject (§6.4), not be voided.
  3. Training projections must be recomputed on a roster change before
     the productive date (`training_delayed` displacement).
  4. The regular-roster ≤ contract rule and §6.4's legacy-pattern rule for
     initial employees (frozen original week totals).
  5. Session v2 already carries arbitrary V0.2 actions; add the
     `set_roster` DTO fields and canonical comparison.
  6. Then the owner's benchmark: no action, partial and full
     reorganization over the same 20 seeds at T30, T90 and T180.

---

## V0.2 Slice 1 — employee contract and capacity foundation (0.2.0-dev.1)

The owner approved CONDITIONAL GO (2026-10-15). Decisions, design §18:
* The measured 1.5–5% margin band.
* ×1.30 demand.
* A shared-staffing start, with reorganization as the player's objective.
* Contracts kept at takeover; reductions only through termination and hiring.
* Provisional scenario-specific T90 gates.

Slice 1 builds the foundation and **adds no player action**. `established` and
`mini` outcomes are unchanged.

* **Contracts.** Hourly employees have contracted weekly hours. A scenario
  may state them (`contractedWeeklyMinutes`, 1–2,400); otherwise they are
  inferred from the roster as the smaller net rota week: E01/E03/E05 40 h,
  E04/E06 39.5 h. The accrual weights (the original roster by rota week and
  weekday) are frozen at load. Salaried staff have no contract hours; their
  salary is the guarantee.
* **Guaranteed pay.** Worked pay is posted exactly as before. At accounting
  close the week's guarantee earns the day's share, and a signed provision
  tops worked pay up to the contract (wages and benefits). Sunday's close
  finalizes the week. Guaranteed time is never worked time, so it creates no
  overtime and is never paid twice. A payday pays accrued payroll less the
  week's still-open guarantee. The provision stays inside accrued payroll
  rather than a new account, so the evaluation baseline's account list and
  digest are unchanged (design §5.5, revision of §9).
* **Employment registry** (state, digested): stable ID, lifecycle stage
  (only `active` until hiring exists), employed dates, contract and the
  week's pay counters. Checked every transaction. Observations show stage,
  contract minutes, the inferred flag, this week's open guarantee and the
  severance termination would owe (two contracted weeks with benefits).
* **Whole-schedule staffing check** (`World.CheckStaffing`, read-only). For
  a hypothetical roster change from a future day, it re-books every booked
  appointment in survivor order with the booking rule's own predicates. It
  reports each one that would be left `unstaffed`, without supervision
  (`no_supervision`), or competing for shared technologists
  (`shared_capacity`). It cancels, moves and displaces nothing, and makes no
  draw or log entry.
  * About 11 ms over the full horizon.
  * 150 unchanged-roster checks (established, the sustainable start and
    mini; 10 seeds; takeover to day 89) found no inconsistent existing
    booking.
* **Transition analysis** (design §5.6).
  * The shared start (`W130-SHR-MRft`) is built from existing scenario
    inputs. Its takeover contracts are Riley 40 h, Sam 39.5 h and Alex 40 h.
  * The lead staffing is reachable with the five approved actions:
    * `hire` Jamie, plus 27.5 h and 20 h replacements for Sam and Alex.
    * `terminate` Sam and Alex.
    * `set_roster` for Riley.
    * `set_reading_allocation` for the radiologists.
  * The plan reaches exactly the proposed 8-technologist cap during notice.
    At the earliest it is complete around day 41.
  * The saving comes from the part-time conversions: keeping every contract
    and only adding Jamie measured −$1.2k a month in Step 0C.
  * Estimated upfront cost is ≈$20–25k (recruiting, severance, training,
    overlap) against ≈$6k a month of headroom. **The full plan may not pay
    back within T90.** The dynamic benchmark must measure it; gameplay
    documentation does not promise +2.8%.
* **Compatibility.**
  * The state encoding is `dreamteam.state/v2`. Each historical golden
    reference differs in exactly `digests.state`, `digests.run` and
    `replay.runDigest`. Log, ledger, actions and baseline digests and every
    outcome value are identical.
  * The references stay byte-for-byte. `testdata/golden/MIGRATIONS.json`
    records old and new values with the explanation, and `-golden-migrations`
    requires both exactly.
  * Session records from v1-state builds (`0.1.*`, `0.2.0-dev.0.*`) replay
    with outcomes verified (action results, log, ledger, actions, summary)
    and the state change reported. Any other engine still needs every
    digest.
  * The Step 0C sustainable start and the lead candidate reproduce their
    committed per-seed results exactly (9 seeds each).
* **Regression tests.** Written before the implementation and shown failing
  against stubs:
  * The design's worked payroll examples to the cent (no work, 32 of 40,
    45 h with premium, 28 of 20, 40 of 40).
  * Provision reversal, partial week and rounding.
  * Contract inference; the registry in the state digest.
  * Guaranteed pay with a roster below the contract, and payday exclusion.
  * Neutrality of contracts at or below the roster: ledger and log hashes
    equal.
  * Extensions taking the week into overtime; `TestGuaranteeWithOvertime`
    passed against the stub too, so it is a guard.
  * Scenario validation and the start and lead contracts.
  * Unchanged-schedule, lost-coverage, shared-capacity, supervision,
    read-only and rejection staffing checks.
  * Golden migration strictness and v1-state replay. The replay test fails
    with the rule disabled.
* **Remaining for later slices:**
  * Growable staff sets (64-employee bit set).
  * Date-aware rosters and plan versions.
  * Recruiting and severance postings.
  * Displacement.
  * The five actions.
  * Session v2.
  * The named `sustainable` scenario.
  * Service-gate calibration from dynamic plans.
* **Verification:**
  * Full uncached `go test -count=1 ./...` passed: 7 packages, sim 349 s,
    cmd 65 s.
  * `go vet` clean (default, calibration and acceptance tags); gofmt clean.
  * The four goldens are unchanged and match under the migration manifest
    (90 days included).
  * `-verify-replay` matched on every reference.
  * The 90-day run time is unchanged within noise (MEASUREMENTS.md).

---

## V0.2 step 0C — late shifts and staffing architectures (0.2.0-dev.0.2)

Owner option G (2026-10-15): fix the noon shift-start limit, then test
explicitly different, fully costed technologist staffing architectures for a
larger practice. No action code; `established` and `mini` untouched.

* **Late shifts (engine).** A shift's cap, holdover included, is now the
  earlier of start + `maxShiftHours` and accounting close (23:59:59):
  `sim.Policies.ShiftCap`, used by runtime staffing, `extend_shift`, the
  roster view and alerts. Validation rejects shifts longer than
  `maxShiftHours` or ending after 23:59:59, not shifts starting after noon.
  Messages name accounting close when it is the binding limit.
  * Regression tests written failing first: `TestShortLateShifts`
    (16:00–20:00, 17:00–20:00, 11:30–20:00, 19:00–23:59 valid; over-cap and
    past-midnight rejected), `TestShiftCap`, `TestShortLateShiftsRun` (late
    shifts staff their rooms; invariants hold daily).
  * `TestRegress05MidnightBoundary` now runs its fixture: the late patient
    is rescheduled and nobody works past midnight.
  * Every existing roster starts before noon, so caps, goldens, replay and
    determinism are unchanged.
* **Harness** (calibration build tag, read-only): records conflict timing
  (weekday × hour), qualified staffed hours, technologist duty hours,
  reports by modality, estimated radiologist cost, payables, accrued
  payroll, receivables and the lowest daily cash. The test-only loss probe
  records each shared-conflict slot's start. The 20-seed Step 0B candidate
  reproduces exactly.
* **Sweep** (`docs/calibration/feasibility/step0c/`; 74 manifests; 9 paired
  seeds, 20 for the leaders):
  * 62% of CT shared-technologist losses were on Mondays.
  * A part-time MR evening is worth +$4.9k a month with no volume change.
  * Dedicated or float XR/CT coverage removes 80–95% of shared losses.
  * Part-time beats full-time for the same coverage by $2.6–7.8k a month.
  * The extra volume overwhelms reading: urgent p90 188 → 1,046 minutes at
    the base allocation.
* **Lead candidate** (×1.30 demand, 20 seeds): dedicated XR/CT day core,
  part-time evenings, reading at 62.5/52.5%.
  * Margin +2.8% (1.5–5.0%, median 2.6%); profitable on 20/20 seeds at T90
    and in days 31–90; lowest cash ≈$51k.
  * Breach rates better than the shared control: urgent 10.1% vs 10.4%,
    soon 27% vs 37%, routine 4.2% vs 7.0%.
  * 145 fewer lost referrals and 101 fewer walk-ins leaving.
  * At ×1.15 demand every architecture loses money (lead −4.0%); at ×1.50,
    out of range, +6.3%.
* **Verdict: CONDITIONAL GO.** Owner decisions (design §18): accept a
  ≈1.5–5% band; accept ×1.30 as the defining demand; choose the starting
  state (the candidate, or the shared practice with reorganization as the
  player's task); decide how part-time contracts are reached; freeze the
  service tolerances.
* **Housekeeping:** removed a committed `__pycache__` file; `.gitignore`
  ignores `__pycache__/`.
* **Verification:** new and updated regression tests; `go vet -tags
  calibration`; control reproduction; full uncached `go test -count=1 ./...`.

---

## V0.2 step 0B — CT/XR booking losses and templates (0.2.0-dev.0.1)

Owner option B (2026-10-15): find why CT/XR referrals are lost while their
rooms are ≈40% used, then test CT/XR templates as a permitted static lever.
No action, scenario or gameplay change; `established` and `mini` untouched.

* **Booking-loss probe** (diagnostics). Production change, explained: one
  function field `state.lossProbe` (excluded from the state digest, like the
  existing test hooks `dropOrder` and `forceNoShow`) and one nil-checked
  call where `offer` records a loss. The diagnosis itself
  (`internal/sim/lossprobe_export_test.go`, test builds only) classifies
  every compatible slot in the lost patient's acceptance window by the first
  FirstFit check it fails (booked, not rostered, no supervision, notice,
  shared technologist). It assigns one primary cause by documented
  precedence and also records every cause present. It reads state only.
* **Neutrality:** `TestLossProbeIsPassive` (log, ledger, state and action
  digests identical with and without the probe; every loss diagnosed once;
  no unexplained free slot). The 20-seed Step 0 candidate reproduces its
  per-seed results and fingerprint exactly with the probe on. Goldens
  unchanged.
* **Finding:** at the Step 0 best candidate, 72% of lost CT/XR referrals had
  a free, staffed slot in their window, blocked because one technologist
  covers both rooms (Riley mornings and Mondays, Sam evenings). 92% of them
  are urgent or soon patients. The template category mix explains ≈7% of CT
  losses.
* **Templates and coverage** (`docs/calibration/feasibility/step0b/`; 13
  manifests, 9 paired seeds, the best and its reference at 20):
  * 12-minute XR slots: +$1.16k per 90 days (CI +0.36 to +1.96), margin
    +0.4% → +0.5%. Recovered referrals displace walk-ins.
  * XR halved in shared hours: −$2.3k per 30 days.
  * XR grid staggered 5 minutes: −$5.2k (fragments shared time).
  * An XR/CT technologist (day or evening): recovers 145–170 losses and pays
    for itself, no more. Urgent breaches rise ×2–4: reading becomes the
    constraint. Restoring it with the 65/55 allocation: −$5.3k.
* **Verdict:** gate still not passed. The binding constraints are shared
  technologist staffing, low XR contribution, evening reading coverage, and
  fixed costs with salaried reading. Templates are exhausted. The best
  configuration is +0.5% mean T90 margin, 14 of 20 seeds profitable.
* **Tooling:** `report.py` takes a results directory and paired references
  and adds booking-loss cause tables. The calibration harness records loss
  causes per window.
* **Verification:** `TestLossProbeIsPassive`; `go vet -tags calibration`;
  calibration summary unchanged; full uncached `go test -count=1 ./...`.
* **Open:** owner decision on the premise (design §17.2: adopt the
  near-break-even candidate, build reading coverage and test teleradiology
  substitution dynamically, or revisit the staffing model). The noon
  shift-start fix stays a separate prerequisite. NO GO for action code.

---

## V0.2 step 0 — design draft 2 and the feasibility gate (0.2.0-dev.0)

Design and static experiments only: no engine, scenario, action or replay
change. No-action outcomes are those of v0.1.0.

* **V0.1.0 published.** GitHub Release "DreamTeam v0.1.0" from the existing
  tag (89b1779), after CI (run 38663474125) and a full uncached suite on the
  tagged tree passed.
* **Design draft 2** (`docs/V0.2_DESIGN.md`) applies the six owner decisions
  and resolves the 18 review findings (Appendix A): staff registry separate
  from the scenario; contracted, rostered and worked hours with guaranteed
  weekly pay and no double payment (legacy contracts inferred, including
  two 39.5-hour employees); the lifecycle from recruiting to severance; five
  actions with schemas, atomic batches, no-op rules and previews separating
  guaranteed from forecast costs; whole-schedule booking reconciliation and
  a practice-caused displacement lifecycle; employed vs outsourced reading
  with no price floor; finance postings; neutrality and golden migration;
  session v2 with a v1 reader; the T90 metric vector, baseline-relative
  service gates and an exploit table. `docs/ROADMAP.md` re-scoped V0.2–V0.6
  (V0.6 restored), V0.1 sections unchanged.
* **Step 0 feasibility sweep** (`docs/calibration/feasibility/`): 53 static
  overlays (demand ×1.00–1.30 uniform and imaging-weighted, XR/CT, US and MR
  technologists, 40-minute US slots, radiologist allocation 50/40–65/55),
  9 paired seeds at 30 and 90 days, the best three and the base at 20 seeds,
  plus a labelled out-of-range diagnostic (×1.50). Calibration harness:
  `DT_EXP_DIR`/`DT_RES_DIR`, parallel manifests, more aggregated figures;
  historical results untouched (summary regenerates byte-identically).
  `generate.py` writes the manifests, `report.py` the tables.
* **Gate result: not passed.** The best permitted configuration
  (imaging-weighted ×1.30 + MR evening technologist) breaks even: +0.4%
  mean T90 margin (−0.9 to +2.4), T90 profit +$2.8k mean (95% CI −$0.2k to
  +$5.7k), 12 of 20 seeds profitable. Demand ×1.50 gives only +1.6%: the
  practice cannot book it (CT and XR lose hundreds of referrals at ≈40% room
  use; US and MR are single rooms). Fixed costs and salaried reading are
  ≈58% of expenses; cutting the allocation collapses reporting at higher
  volume; US 40-minute slots change nothing; only the MR evening
  technologist pays.
* **Engine constraint found:** no shift may start after 12:00 (validation:
  start + 12 h `maxShiftHours` before midnight), so part-time evening shifts
  cannot be expressed; those cells are blocked (upper bound ≈+2.4–3.8%).
* **Verification:** `go vet -tags calibration ./internal/sim`;
  `TestCalibrationSummary` (historical summary unchanged); the sweep itself
  (four runs, all passing); `go test ./...`.
* **Open:** owner decision on the scenario premise (draft 2 §17.2: accept
  break-even, CT/XR template levers, cost base, reimbursement, teleradiology
  substitution, a second US room); then re-sweep. NO GO for action code.

---

## V0.1.0 — release preparation (v0.1.0)

Release preparation only: no change to mechanics, RNG, scenarios,
actions, accounting or replay.

* **Accepted deviations** (owner, 2026-10-14), recorded in
  `docs/ACCEPTANCE.md`, `docs/ROADMAP.md` and the README: the Pi 4's ≈3.3 s
  headless 90-day run (2 s target waived for V0.1; SHA-256 keyed RNG kept)
  and the naturally small or empty takeover worklist of the struggling
  default practice (no manufactured backlog; the 5–20 target moves to a
  future reporting-constrained scenario).
* **CI**: the Node 20 deprecation came from `actions/checkout@v4`,
  `actions/setup-go@v5`, `actions/upload-artifact@v4` and
  `actions/download-artifact@v4`; now `checkout@v7`, `setup-go@v7`,
  `upload-artifact@v7`, `download-artifact@v8` (all Node 24). Matrix and
  golden checks unchanged.
* **Release notes**: `docs/RELEASE_NOTES.md` (new; newest first).
* **Version** `0.1.0`. The golden references were not regenerated (no
  outcome changed): they still name the engine that wrote them, and the
  comparison reports the engine difference with outcomes identical.

---

## Steps 19–20 — Performance pass, acceptance results, release docs (v0.1.0-dev.20)

**Acceptance results.** Criterion 10: invariants after every transaction,
50 seeds × 90 days — passed (53.5 min on four Cortex-A76 cores; one seed
takes ≈7 min on the Pi 4). Criterion 9: 1,000 random chunkings of the real
simulation — passed. Criterion 8: the first CI run produced byte-identical
golden runs on linux/amd64, linux/arm64 and darwin/arm64. Every criterion
and its evidence: `docs/ACCEPTANCE.md`.

**Performance (criterion 16)**, details in `docs/MEASUREMENTS.md`:

| | Pi 4 (throttled, loaded) | Cortex-A76 (joofi) | Target |
|---|---|---|---|
| Warm-up | 1.44–1.49 s best | 0.53 s | < 1.5 s |
| `run -days 90` | ≈3.3 s | 1.1–1.2 s | < 2 s |
| TUI at 15 min/s | 7.4–8.2% of a core | — | < 15% |
| TUI paused | 0.4% (was 3.0%) | — | — |
| Max speed, worst frame | 52–94 ms | — | `space` within 100 ms |

The Pi 4 misses the 90-day target: its CPU has no SHA-256 instructions and
the keyed draws (the determinism contract) are SHA-256; the profile has no
other dominant hotspot. Changing the draw hash would change every outcome
and digest — an engine-version decision for the owner, not a tune-up. A
per-day free-slot cache for FirstFit was tried (outcomes identical) and
reverted: 3% at best.

**Changes.** The TUI's frame clock stops while paused and the renderer runs
at 20 fps (idle CPU 3.0% → 0.4%; outcomes unaffected). New benchmarks:
`BenchmarkEstablishedPlay90` (sim) and `BenchmarkMaxSpeedFrames` (TUI frame
times at Max). `dreamteam help` lists the `scenario` command.

**Release docs.** `README.md` (what it is, quick start, controls, actions,
scenarios, headless runs, sessions and replay, development, documents,
known limitations) and `docs/ACCEPTANCE.md`.

**Open for the owner.** (1) Accept criterion 16 on the Pi 4 as measured, or
decide on a faster keyed-draw hash for a later engine version. (2)
Criterion 1's "unread backlog at the takeover" does not hold for the
struggling practice (reading is over-provisioned at its volume). (3) Tag
V0.1 when satisfied.

---

## Steps 16–17 — Acceptance pass, part 1: TUI polish and end-to-end replay (v0.1.0-dev.18.3)

The V0.1 acceptance pass (ROADMAP §2), with the owner's list. The
checklist with every criterion's evidence is `docs/ACCEPTANCE.md`.

**Built.**
* **Help screen**: `?` from any screen shows the keys and the worklist
  rule; `?` or `esc` returns where it came from.
* **Event-log filters**: on the log screen `tab` cycles the category,
  `!` the severity (all · warnings & alerts · alerts only); the title shows
  the filter and how many entries match.
* **`run -log events.jsonl`**: the observable log from the takeover, one
  JSON object per line (seq, time, severity, category, text), through an
  output-only sink (never part of the state; the result is byte-identical
  with and without it). `-summary text|json` is an alias of `-format`.
* **NO_COLOR**: colors follow the profile Bubble Tea detects from the
  environment; with `NO_COLOR` no color sequence remains and severity stays
  marked (⚠ ‼). Documented in `dreamteam help`.
* **CI** (`.github/workflows/ci.yml`): vet (all tags), gofmt, the full
  suite and the golden references on linux/amd64, linux/arm64 and
  darwin/arm64, then a byte-for-byte comparison of the three platforms'
  results.
* **Acceptance suite** (`-tags acceptance`): criterion 9 on the real
  simulation (1,000 random chunkings of warm-up, takeover and a day of
  operations) and criterion 10 (invariants after every transaction, 50
  seeds × 90 days).

**Tests.** `TestTUIRunReplaysHeadless` (criterion 8: three randomized TUI
sessions — speeds, frame gaps 10–610 ms, pauses, steps, alert auto-pauses,
actions submitted from the actions screen — each reproduced by a headless
session from its action record and by replay; all five digests),
`TestHelpAndLogFilters`, `TestNoColor` (fails without NO_COLOR),
`TestRunLogJSONL`, `TestScreenSnapshots` (ten screens at 80×24, 120×36,
160×48; `-update-screens` to regenerate), and `TestRenderFitsTerminal` now
also covers 140×40, briefing, actions, help and the end-of-day summary.

**Deviation recorded.** Criterion 1 asks for an unread backlog at the
takeover; the struggling practice usually has none (reading is
over-provisioned for its volume, as characterized in step 18).

---

## Step 18.2 — Urgency always takes precedence on the worklist (v0.1.0-dev.18.2)

**A deliberate gameplay rule change (owner decision), not a calibration
correction.** Step 18.1 showed that shortest-first could postpone a long
urgent study indefinitely while short routine studies kept arriving. Now
urgent studies are offered before all others under every worklist policy,
and the policy orders within each class: default unchanged (urgent and
overdue first, then FIFO), FIFO → urgent first then FIFO, oldest-first →
urgent first then oldest, shortest-first → urgent first then shortest
published expected read. The existing `urgent` priority is the class (no
new severity); reads in progress are never interrupted; hidden read
durations are never used; ties still fall to the study ID. Reading blocks,
staffing, read durations and reimbursement are unchanged.

The TUI actions screen labels the choices "urgent first, then …" and says
"Urgent studies are always read first; the policy orders the rest"; the
worklist screen, API comments and SIMULATION_DESIGN §6.8 say the same.

**Tests.** `TestRegress18UrgentLongStudyVsShortRoutine` (every policy reads
the urgent MR next after the read in progress; before: FIFO and
oldest-first after 5 min, shortest-first never in 7 h) and
`TestRegress18WorklistUrgentFirst` (exact order per policy on a hand-built
worklist, determinism, shortest-first unchanged with no urgent study
waiting) fail on 049b304 and pass now. `TestWorklistPolicyOrdering` checks
shortest-first within each class. Full uncached suite green.

**Effect** (nine seeds, paired; details in the calibration report):
exams, lost bookings, walk-ins, revenue, opex and margin identical in every
run. 90 days: FIFO urgent breaches 32.9 → 15.1 and urgent p90 599 → 282
min (now equal to the default); shortest-first urgent breaches 18.2 → 16.3,
urgent MR median 99 → 52 min, routine p50 still 22 min. The default policy
is unchanged. The remaining urgent tail is the evening/weekend reading
coverage gap; with this rule the calibration phase is closed.

**Goldens.** Only `established-seed1-7d-actions.json` (it sets FIFO) was
regenerated; the three default-policy references are identical (compared
with the engine mismatch allowed). Its changes: reading order (urgent TAT
p50 1,407 → 803 s, max 3,628 → 2,256 s; breaches 13 → 12; one alert fewer;
radiologist busy time shifted between the two), the log digest
(6e34bd068f7f… → 625dfe83e5d8…), the ledger digest (6c18847e1bad… →
4482cec1c8d3…: revenue posts at report final, so entries move in time;
window money and balances are identical), state 72f117978ca0… →
5a626a6308b7…, run 9a3d8861926c… → fb0ccb14a79c…. Patient flow, exams and
money unchanged.

---

## Step 18.1 — Calibration review remediation (v0.1.0-dev.18.1)

The step 18 review (`docs/reviews/step-18-astra-review.md`, CONDITIONAL GO)
found five defects. Each got a regression test that fails on `b1a9ffb`,
then the smallest fix; every experiment was rerun on the fixed engine.

**Fixes.**
* `extend_shift` validates the raw minutes before converting them; huge
  values no longer wrap into a negative extension.
* Shared-staff booking is a real feasibility check: distinct qualified
  technologists for every overlapping appointment, handovers only at
  coverage boundaries, whatever the booking order. Default outcomes are
  identical; dispatch is unchanged; warm-up time is unchanged.
* Walk-ins: arrival, same-visit admission, the offer (declined by policy
  or unservable) and whether it booked, the same-visit exam and a later
  return exam are separate counters. Walk-in-origin booking events are
  out of referral demand (`demand.walkInOrigin`). Acceptance and
  abandonment are over the cohort of walk-ins who arrived in the period,
  each classified once (imaged same visit, converted, left, lost, on
  site). The TUI summary reads "walk-ins N: admitted, offered later
  (booked), left".
* The strategy helper no longer requests extensions past the scheduled
  close; the harness fails on any rejected action.

**Reproducible experiments.** `docs/calibration/experiments/*.json`
(scenario, overlay, strategy, seeds, horizon) run with
`go test -tags calibration ./internal/sim -run 'CalibrationExperiments$'`
and write `docs/calibration/results/<name>.json` (engine, fingerprints,
overlay hash, per-seed windows, aggregates) and `results/summary.md`. The
harness adds an order cohort, requested lead vs capacity delay, approved
overtime and turnaround by priority and modality.

**Results** (nine seeds; full tables in the report):

| | 30 days | 90 days |
|---|---|---|
| Baseline exams / margin | 1,363, −12.5% | 1,296 per 30 d, −14.0% |
| Three more technologists | 1,458, −16.7% | 1,390 per 30 d, −18.6% (step 18: −18.7%) |
| Faster XR (×0.75) | 1,390, −11.6% | 1,321 per 30 d, −13.1% |
| Reading allocation 45%/38% | TAT p50 20 h — unstable | TAT p50 22 h |

Strategies (30 days): B 1,431 exams, −9.4%; C 1,394, −13.4%; D 1,461,
−10.4%; E 1,394, −12.6%; F TAT p50 34 → 22 min (≈35%, not half). C and D
moved by at most one exam and 0.1 points with the booking fix; the
struggling-practice characterization stands.

**Urgent turnaround.** The default's urgent tail is a reading-coverage
gap: ≈¾ of urgent breaches are studies acquired after the last reading
block and read the next morning or Monday (max 31–41 h under every
policy). FIFO adds ≈15 in-hours urgent breaches per 90 days;
shortest-first ≈2, but it delays long urgent studies (urgent MR p50 99 vs
52 min) and, in a controlled micro case, starves them while short routine
work keeps arriving. No worklist semantics changed; the trade-off is
documented for an owner decision.

**Goldens.** All four regenerated: outcomes unchanged (log, ledger and
action digests identical), the state digest changed (new order and counter
fields in the state walk), and the result JSON changed in walk-in fields
and referral `lost`/`booked`.

| Golden | state digest | run digest |
|---|---|---|
| established-seed1-7d | 8a2a0fd92281… → c4d6b46c4d94… | af8fb442c246… → 7b99a6f7ffc1… |
| established-seed1-90d | 8f7de97d1fc0… → 8c6575d35565… | a25871c771cc… → 12157065771c… |
| mini-seed1-7d | 0db92e2b5226… → f2dff6e4d307… | e3b39b7707d0… → 26adad607323… |
| established-seed1-7d-actions | 08ab88ba9b21… → 72f117978ca0… | 9005416cd60a… → 9a3d8861926c… |

**Tests.** Regressions: `TestRegress18ExtensionMinutesRange`,
`TestRegress18SharedStaffFeasibility`, `TestRegress18WalkInCohorts`,
`TestRegress18FlowCountsFieldsFolded`, `TestRegress18OrderCohortReconciles`,
`TestRegress18UrgentLongStudyVsShortRoutine`. Full uncached suite green.

---

## Step 18 — Calibration: a struggling, capacity-constrained practice (v0.1.0-dev.18)

**Owner decisions (2026-10-14).** Calibrate in order — volume and flow,
reporting, economics, cross-seed — without forcing numbers. When the
measurements showed the margin target conflicts with the documented roster,
the owner chose: the default established practice is **struggling and
capacity-constrained**; documented reimbursement, costs, staffing and
operational assumptions stay; its margin is characterized, not forced; the
6–12% band belongs to a separate better-staffed preset, added only if it
can be configured cleanly (it cannot yet — next scenario milestone); the
mini scenario is unchanged; no hiring, roster editing, new actions or
artificial revenue.

**Method.** A calibration suite behind the `calibration` build tag
(`internal/sim/calib_test.go`, `calib_actions_test.go`): nine seeds, the
180-day warm-up, 30- and 90-day windows from the takeover, daily 07:30
worklist snapshots, per-person utilization, lost orders by reason, lead
times by modality; and a scenario overlay (`DT_OVERLAY`, the scenario
system's own merge) for trying candidate parameters without editing files.
The pre-calibration nine-seed record is
`docs/calibration/baseline-0.1.0-dev.15.json`; the full report with every
table is `docs/calibration/step-18-report.md`.

**Baseline diagnosis** (nine seeds, 90 days; 30-day in brackets):

| | Mean (range) |
|---|---|
| Demand: orders + walk-ins a month | 1,423 + 224 = 1,647 |
| Exams a month | 1,314 (1,293–1,335) [1,381] |
| Tue–Fri visits a day | 59.9: XR 27.0, US 10.0, CT 15.1, MR 7.7 |
| Lost bookings a month | 127: US 64, MR 29, CT 17, XR 17 (almost all "declined": earliest slot beyond tolerance) |
| Walk-ins leaving | ≈100 a month, 45% of walk-ins |
| No-shows / cancellations | 6.5% / 8.9% (as documented) |
| Technologists busy (of duty) | Sam 82%, Morgan 82%, Casey 81%, Riley 76% |
| Radiologists reading (of site blocks) | Avery 73%, Blake 63% |
| Worklist at takeover / 07:30 Tue–Fri | ≈0 / mean 3.3, max 9–18 |
| TAT p50 / p90 / over target | 32 min / 122 min / 5.2% |
| Net revenue, labor, fixed, variable a month | $200.4k, $106.8k, $87.8k, $32.7k |
| Operating margin | −14.0% (−15.6 to −13.0) [−12.5% (−16.9 to −11.1)] |

**Bottlenecks.** Demand matches the design; capacity does not. XR and CT
share technologist time most of the day (Sam and Riley saturated, rooms
idle 60%+): walk-ins register, wait 30–112 min and leave. US is one
technologist with 45-min slots for ~37-min exams (69% busy of staffed time,
the most lost demand, lead p50 19 days). MR is one technologist with nine
slots. Reading is over-provisioned for this volume (hence the empty
evening worklist and fast TAT).

**Experiments** (nine seeds; none applied): XR room times ×0.75 → +27
exams a month; walk-in policy `accept_always` → no change; radiologist
site allocation 45%/38% → unstable reading queue (TAT p50 20 h), 55%/45% →
stable, ≈$5k a month cheaper (not applied: documented staffing kept); three
more technologists → lost bookings −70% but exams +7% and margin −18.7%,
because demand, no-shows and cancellations bind. The design's reference
P&L counted ≈1,650 orders as ≈1,650 exams with technologists ~94% busy:
internally inconsistent with its own roster (now noted in §8.6).

**Parameters changed: none.** No change was defensible under the decision
(each candidate either altered documented staffing or only marginally
moved volume), so the established scenario's fingerprint is unchanged
(`b1cbc2a0…`), and the golden references are unchanged: their inputs and
outcomes did not change (rule: update only when they do). What changed is
the characterization: design §5.1, §6.2, §8.6, ROADMAP step 18 and the
V0.1 calibration criteria (now the better-staffed preset's), the briefing
note, and the tests. The four references still name engine 0.1.0-dev.15; under
0.1.0-dev.18 their outcomes reproduce exactly (`TestGoldenReferences`
compares with the engine difference allowed and reported). The CLI without
`-allow-engine-mismatch` reports them as not comparable (exit 5), as
designed; regenerating them would only restamp the engine.

**Actions** (30 days, nine seeds; demand identical across strategies,
asserted):

| Strategy | Exams | Lost | Margin |
|---|---|---|---|
| A no actions | 1,363 | 141 | −12.5% |
| B US + MR extended to close (approved ahead), walk-ins until 19:00 | 1,431 | 94 | −9.4% |
| C Riley extended (2nd XR/CT evenings), accept all walk-ins | 1,394 | 127 | −13.5% |
| D all three extensions, accept all | 1,460 | 79 | −10.5% |
| E decline walk-ins, FIFO | 1,394 | 178 | −12.6% |
| F shortest-first worklist | 1,363 (TAT p50 22, p90 157 min) | 141 | −12.5% |

Actions matter (B recovers a third of lost bookings; F halves TAT; E turns
walk-ins who would leave into appointments) and every added hour is paid
(117–194 h of approved overtime), but none reaches break-even: profitability
needs hiring and demand growth (V0.4). No loophole found: demand is
unchanged by any strategy, declined walk-ins stay counted, extensions are
costed, worklist ordering uses published read times only.

**Targets** (report table): demand, no-shows, MR utilization cap, US
longest lead and most losses, cash/profit divergence — met. Tue–Fri
visits (60 vs 65–85), XR and MR per day just below bands, US 80–90%
utilization (unreachable with 45-min slots, no overbooking), radiologist
capacity 105–115% (≈145%), 07:30 backlog 5–20 (mean 3.3), routine TAT 1–4 h
(32 min), T30 margin 6–12% — missed by the default and assigned to the
better-staffed preset.

**Tests.** `TestEstablishedCharacterization` (three seeds, 30 days, broad
bands: demand 1,450–1,850 a month, exams 1,250–1,500, margin −20% to −6%,
25–65% of walk-ins leave, US loses most, TAT p50 < 2 h and morning backlog
< 40, books reconcile). The calibration suite is tagged (≈30 s and ≈2 min).
Determinism, replay (step 14), warm-up boundary (step 11), closing rules
(step 8), demand independence and goldens are unchanged and pass:
`go test -count=1 ./...`.

**Limitations and next scenario milestone.**
* The better-staffed `moderate` established practice (6–12%) needs more
  staff and more demand; demand growth is V0.4 (referrals). Until then the
  default is the only established practice.
* The P&L workbook and `docs/ASSUMPTIONS.md` (ROADMAP step 18) move with it.
* Reading capacity exceeds this practice's demand; a struggling owner might
  reduce the site allocation — not a V0.1 action.
* Forty-five percent walk-in abandonment is a consequence of the roster;
  patience (30–120 min) is as documented.

---

## Step 15 — Canonical results and golden references (v0.1.0-dev.15)

**Owner decisions (2026-10-14).** Engine-version compatibility stays:
replay and golden comparison against another engine only with
`-allow-engine-mismatch`, and compatibility is always reported apart from
verification. Golden outputs are canonical JSON for machines plus a concise
text summary for people. Golden files hold observable and financial
outcomes and digests, never individual records or hidden state.

**Result schema** (`dreamteam.result/v1`, SIMULATION_DESIGN §4.5):

| Section | Contents | Kind |
|---|---|---|
| identity | schema, engine, scenario ID and fingerprint, seed, epoch date | — |
| `takeover`, `end`, `playDays` | simulation seconds and ISO time | times |
| `actions` | attempted / accepted / rejected, history digest, every result in order (rejections and no-ops included) | history |
| `digests` | log, ledger, actions, state, run | verification |
| `calendar` | each completed day, midnight to midnight (the takeover day flagged): orders, lost, walk-ins, no-shows, exams, reports, unread at close, wait p50, overtime, net revenue, expenses, profit, collections, disbursements, end-of-day cash, alerts | calendar flows (+ end-of-day cash) |
| `window` | from the exact takeover to the end: baseline digest, period metrics (demand, appointments, walk-ins, waits, rooms, staff, reporting, workforce, alert and action counts), window money (baseline-adjusted and as posted, cash bridge) | evaluation-window flows |
| `balances` | cash, A/R, A/P, payroll owed, equity, lowest cash, days below zero | point in time |
| `obligations` | next payroll and bills, open claims and aging, unread / reading, recall list, booked ahead, policies, approved extensions | point in time |
| `alerts` | raised in the window by severity; active at the end | counts |
| `replay` | verified, batches, replayed run digest (with `-verify-replay`) | verification |

Integers only; ratios as numerator and denominator; fixed field order;
arrays in defined order; no wall-clock time, paths or locale. Every field
is compared.

**CLI** (no new execution path: `run` and `replay` gained output flags):

    dreamteam run -scenario established -seed 1 -days 7 -format json            # canonical JSON on stdout
    dreamteam run -days 7 -actions FILE -verify-replay                           # text summary, replay checked
    dreamteam run -days 90 -verify-replay -format json -golden testdata/golden/established-seed1-90d.json
    dreamteam run -days 7 -format json -write-golden FILE [-overwrite]           # explicit, never silent
    dreamteam replay SESSION -format json [-golden FILE]                         # the replayed run's result
    dreamteam verify SESSION                                                     # "ok <run digest>"

Diagnostics go to stderr; with `-format json` stdout holds only the JSON.
Exit codes: 0 ok · 1 error (I/O, scenario) · 2 invalid input · 3 replay
failure · 4 golden mismatch · 5 incompatible or unsupported format (another
schema, engine, scenario, seed or run length; malformed golden; unknown
`-format`). A mismatch report gives compatibility, the number of
differences, the digests that differ and the first differing paths with
both values, e.g. `first: .window.ops.exams: golden 1, run 184`.

**Reference runs** (`testdata/golden/`, README there; each generated with
`-verify-replay`, aggregate outcomes plus digests, 16–52 KB):

| Reference | Exams | Orders / lost | Net revenue | Op. expense | Op. profit | Cash end | Run digest |
|---|---|---|---|---|---|---|---|
| established, seed 1, 7 days | 292 | 329 / 33 | $47,037 | $55,712 | −$8,675 | $144,817 | `af8fb442c246…` |
| established, seed 1, 90 days | 3,887 | 4,171 / 370 | $593,096 | $675,797 | −$82,702 | $16,985 | `a25871c771cc…` |
| mini, seed 1, 7 days | 184 | 360 / 4 | $27,893 | $50,784 | −$22,891 | $137,965 | `e3b39b7707d0…` |
| established, seed 1, 7 days, actions (8: 5 accepted incl. a no-op, 3 rejected) | 294 | 329 / 32 | $47,452 | $55,899 | −$8,447 | $144,707 | `9005416cd60a…` |

(Uncalibrated: the losses are the step 10–11 volume shortfall that step 18
calibration addresses. The 90-day run ends with cash near zero.)

**Tests** (`go test -count=1 ./...` passes):

| Requirement | Test |
|---|---|
| References reproduce (byte-identical when the engine matches) | `TestGoldenReferences` (all four, `-verify-replay`) |
| Repeated runs byte-identical; seeds and scenarios differ | `TestResultsRepeatableAndDistinct` |
| Meaningful changes, missing / unexpected fields, digests, calendar length, engine, fingerprint, seed, run length, schema, malformed / truncated / trailing files, CLI exit codes, no silent overwrite, unsupported format | `TestGoldenComparisonReports` |
| Action history changes detected; rejected actions represented | `TestActionHistoryInResult` |
| Chunk sizes and playback speeds (one jump, 37-min chunks, Max-speed event budgets, alert stops) | `TestResultIndependentOfChunkingAndSpeed` |
| No hidden data | `TestResultHasNoHiddenData` (no record IDs, no latent or queue fields) |
| Financial reconciliation, window boundaries | `TestResultReconciles` (balance sheet, daily cash roll-forward, cash bridge, window = takeover → end, calendar days = play days) |

**Cross-platform.** All four references reproduced byte-identically on a
second Linux arm64 machine (Cortex-A76, via the built binary) as well as
the Raspberry Pi 4 (Cortex-A72) that wrote them. Linux amd64 and macOS
arm64 have **not** been run (no such machine or emulator here); the
procedure is in `testdata/golden/README.md`.

**Performance** (MEASUREMENTS.md): `run -days 7` 1.9 s, `-days 90` 3.1–4.7
s (warm-up included); result build and encode after 90 days 0.54–0.62 s
(mostly the state digest); comparison 18–25 ms. Not in any output.

**Limitations.**
* An engine version bump requires `-allow-engine-mismatch` for old
  references (reported); regenerate to make them byte-identical again.
* Per-day calendar rows make the 90-day file 52 KB; per-patient detail is
  deliberately absent, so a digest-only change (e.g. the order of two
  same-day events) is reported as differing digests without a metric path.
* No amd64 or macOS verification yet.

---

## Step 14 remediation — replay review findings (v0.1.0-dev.14.1)

A focused review of replay correctness (`docs/reviews/step-14-astra-review.md`,
gpt-6-astra at high reasoning, read-only snapshot of `16c53a2`) returned
NO GO with eight findings; all were confirmed, each with a regression test
that fails on `16c53a2` (finding 7 by inspection: no injection point
existed) and passes after the smallest fix. No simulation outcome changed.
The suspected pause-then-act mismatch was ruled out.

| # | Finding | Fix |
|---|---|---|
| 1 (high) | Hour keys like `"+8"` were accepted beside `"08"`: compilation depended on map order | only canonical `"00"`–`"23"` |
| 2 (high) | The run digest did not bind the scenario: future rules could differ under the same digest | run digest includes the scenario fingerprint and seed |
| 3 (medium) | Sessions kept the caller's scenario object: later edits broke saves | private copy |
| 4 (medium) | The state digest included the last-transaction time cache | excluded |
| 5 (medium) | Trailing `]`/`}` after a record, data after a scenario, a second actions array accepted | EOF required |
| 6 (medium) | Re-signed edits to batch numbers, times, scenario ID/name, start/until and summary verified | all checked against what they describe |
| 7 (medium) | Directory-sync failure after rename reported as a durable save | reported; session stays unsaved |
| 8 (low) | Refusals printed "OK … digests match"; mismatch summaries were zeros | OK only when fully checked; real replay position reported |

Records saved by `16c53a2` no longer verify (the run digest changed); they
replay with `-allow-engine-mismatch`. Remaining test gaps are listed in the
review document.

---

## Step 14 — Sessions, replay and digests (v0.1.0-dev.14)

### Part A — playtest (before step 14)

Driven through the real TUI model (scripted keys, every screen read at
80×24) on the established practice, seed 1:

1. **Two days, no actions.** Alerts, queues, worklist and money agree:
   day 1 70 exams, 16 alerts (mostly XR waits behind the shared XR/CT
   technologist), 3h 01m overtime, profit $575; day 2 60 exams, an insight
   flags MR busy 86% of staffed time but staffed 70% of open hours. The
   17:00 dashboard shows self-registration after the desk staff leave;
   "MR room unstaffed — why: Casey on break until 13:00" matches the roster.
2. **US extension a week ahead** (Morgan, Wed Oct 21, +2h): the schedule
   for that day went from 2 to 5 open US slots (18:00, 18:45, 19:30), at
   once. Demand untouched (tests).
3. **Walk-ins: decline, then accept-if-gap.** The 3 walk-ins arriving
   while declining were offered appointments and counted as declined;
   nobody already waiting was removed; switching back took effect for the
   next arrival.
4. **Worklist: priority → FIFO** with an urgent and two routine studies
   unread: the urgent XR dropped from first to last at once; MR and US were
   read first, the urgent one at 38 min; the read in progress continued.

**Defects found and fixed (all in action feedback):**
* An extension's preview did not say what the extra time covers, and an
  extension that would cover nothing was accepted. Now each says
  "staffs US 18:00–20:00", "covers the desk …" or "on site for contrast
  supervision (reading blocks unchanged)", and one covering no room, desk
  or supervision is refused (`no_coverage`). Coverage is evaluated on the
  extended plan, the same plan dispatch and booking use, so the preview,
  the schedule and the effect agree (`TestExtensionWithoutCoverageRefused`).
* It did not say the benefit: it now names the slots it opens ("opens 3
  US slots for booking") or says none would be ("already staffed, or no
  template slots then" — e.g. Riley's XR evening, which Sam covers).
* At 80 columns the preview and result were cut off (the cost and effect
  lost) and dates were ISO. They now wrap over up to three lines ("Done ✓
  …", "If submitted: ✓/✗ …"), dates read "Wed Oct 21", and the next preview
  restarts at +60m instead of showing a confusing rejection.
* Declined walk-ins were cut off the 80-column dashboard: now "walk-ins 3
  (3 declined)".

Owner note: the five-day comparison's +$181 does not make the policy
optimal — 22 h of approved overtime for 8 exams may reverse once step 18
calibrates rates and staffing. That trade-off is for players and agents to
discover.

### Part B — sessions, replay, digests

**Principle.** A saved run is reproduced from its scenario, seed and
timestamped actions; no serialized world is trusted. The session file
(`dreamteam.session/v1`) is a run record: the fully resolved scenario with
its fingerprint, the seed, epoch and takeover, every action batch with its
results (rejections and no-ops too), the saved time, five digests, a
readable summary and a checksum. Format and rules: SIMULATION_DESIGN §4.5
(as built).

| Digest | Covers |
|---|---|
| log | every observable log entry, in order |
| ledger | every journal entry, in order |
| actions | the action record with results |
| state | the whole authoritative state, hidden fields included, and the live pending events; not caches, hooks, processing counters or event sequence numbers |
| run | all of the above and the clock |

**Load = replay.** Strict parse → checksum → schema → scenario fingerprint
→ batch structure → compile, warm up, apply every batch at its time
through `Apply`, compare each result → run to the saved time → compare the
digests. Mismatches name the first differing action (batch, time, action,
recorded vs replayed) or the differing digests, with both summaries.

**Compatibility.** Another schema: refused. Another engine version:
refused unless `-allow-engine-mismatch`, which replays and lets the digests
say whether outcomes agree. Changing the rules (any future gameplay step)
therefore changes which old saves verify: that is the point.

**Changes to existing behavior (no outcome changes).**
* Alerts keep the text they were raised (or escalated) with; before, the
  stored text was refreshed at each evaluation, so a *resolved* alert's
  displayed text depended on when unrelated events happened. Observations
  of active alerts still show the live text.
* A batch of only rejected or no-op actions no longer moves the metrics'
  accrual point or the clock cache; an empty batch is not a batch.
* `Session.Digest()` is now the run digest (log, ledger, actions, state,
  clock).
* Saved and loaded sessions without actions are identical to `08ccc5d`
  (ledger and every visit, study and appointment, established 21 days and
  mini 30 days, seeds 1–3).

**Controls.**
* TUI: `ctrl+s` save, `ctrl+o` reload (replay the saved file and resume),
  `ctrl+v` verify by replay; the top bar shows the seed and "● unsaved".
* CLI:
  * `dreamteam play [-load FILE] [-save FILE]`
  * `dreamteam run [-scenario ID] [-seed N] -days D [-actions FILE] [-save FILE]`
    — headless, prints each action result, the since-takeover figures and
    the digests. Actions file: `[{"time": "2026-10-13T07:00:00", "actions": [...]}]`.
  * `dreamteam replay FILE [-allow-engine-mismatch]` — full report.
  * `dreamteam verify FILE` — `ok <run digest>` or the mismatch, exit 1.

**Tests** (`go test -count=1 ./...` passes; `internal/sim` 158 s,
`internal/session` 21 s):

| Requirement | Test |
|---|---|
| Save/load round trip, resume | `TestSaveLoadRoundTrip` (accepted, rejected, no-op, two multi-action batches, a future-day extension; both sessions continue identically) |
| Replay after restart | `TestRunSaveReplayVerify` (CLI: separate run, file, replay, verify) |
| Midday takeover, cross-day and cross-week, payroll and claims across resume | `TestReplayMiddayTakeoverAcrossPayrollAndWeek` (13:00 takeover, saved Thursday 23:00, continued past Friday payroll and remittances to the next Tuesday) |
| Corrupt, truncated, edited, fingerprint, schema, engine, tampered results and digests, order | `TestCorruptAndIncompatibleFiles` (11 cases) |
| Atomic save failure | `TestAtomicSaveFailure` |
| Chunking | `TestReplayChunkingInvariance` (17 min, 3 h, single step) |
| Independence from unrelated events, reads, previews, rejections | `TestStateDigestIndependentOfNoiseAndReads` |
| Hidden state covered, nothing leaked | `TestStateDigestCoversHiddenState`, `TestSavingHasNoSideEffects` (no latent fields in the file; saving and verifying change nothing) |
| Determinism | `TestStateDigestDeterministic`, `TestCanonicalRoundTrip`, `TestParseISORoundTrip` |
| TUI | `TestSessionKeys` |

**Performance** (MEASUREMENTS.md): warm-up unchanged (1415 ms vs 1359 ms
best, within thermal noise; identical memory); save after 30 days
0.44–0.54 s; load = replay + verify of 30 days 2.1–2.3 s; `run -days 30`
plus save 4.0 s and 48 MB peak; file 35 KB.

**Limitations.**
* No snapshots: loading always replays from the epoch (warm-up included),
  about 1.4 s plus the played days. A snapshot cache verified against the
  digests can come with V0.2 save/load UI.
* The state digest is a reflection walk over the state's fields: a field
  added later is covered automatically, but a new cache must be added to
  its skip list (or the noise test fails, which is how this one was found).
* Records include the seed: fine for game saves; benchmark records will
  escrow it (V0.5).
* Golden 7- and 90-day digests (ROADMAP testing strategy) are not pinned
  yet; the start-state goldens are.

---

## Step 13 — Player actions (v0.1.0-dev.13)

**Owner decisions recorded (2026-10-14).** The four step 12 conventions are
approved. For #3, the window's operating result excluding inherited-claim
collection variance is a *baseline-adjusted* evaluation convention: the
since-takeover screen labels it "baseline-adjusted" and shows the ledger
basis ("as posted (the accrual P&L)") next to it; the calendar statements
and `WindowMoney.Ledger` keep the actual collection variance. A write-down
of an inherited receivable after takeover is visible there.

**What the player can do.** `a` opens the actions screen (auto-pause moved
to `A`): extend a shift today or on a day up to two weeks ahead (`[` `]`),
set the walk-in policy, set the worklist policy. Each row is previewed
live; `enter` submits; results and the latest actions are listed. The
dashboard shows the policies in force and approved extensions.

**One interface.** `World.Apply` / `Session.Apply(actions...)` is the only
way to change the simulation; `Validate` previews without effect. The TUI
calls the same functions an agent will. Batches apply at the current clock
between transactions, in order, each action against the state the previous
left; a rejected action changes nothing; one dispatch and the alert
observer run after a batch that changed something, none after rejections or
no-ops. Every attempt is recorded with a stable ID (`X00001`), batch
number, time, action, result code and message (`ActionLog`,
`Observation.Actions`, and per-period counts in the metrics and the
evaluation window). Schema, codes and rules: SIMULATION_DESIGN §9.5 (as
built).

| Action | Parameters | Effect |
|---|---|---|
| `extend_shift` | `employee`, `minutes` 1–240, `date` (optional; today to the booking horizon) | the scheduled end moves later that day; they stay on regular duty; the day's slots in the extra time become bookable at once; time past the rostered end is approved overtime |
| `set_walkin_policy` | `policy`: `accept_always`, `accept_if_gap`, `accept_until` (+ `cutoff` HH:MM), `schedule_later` | walk-ins arriving from now on; declined ones are offered appointments, counted apart |
| `set_worklist_policy` | `policy`: `urgent_then_fifo`, `fifo`, `oldest_first`, `shortest_first` | the order unread studies are offered; reads in progress continue |

**Extension vs hold-over** (the design issue you raised). Required hold-over
finishes patients already admitted; it starts past the scheduled end and
only while someone needs the person. An approved extension moves the
scheduled end, so the person keeps staffing their room on regular duty: new
capacity for waiting patients, walk-ins and bookings. Approved and required
overtime are recorded apart (`ApprovedOvertimeSec`/`…CostCents` in the flow
counters, staff metrics, workforce metrics and the window). Hard limits stay
hard: the 12-hour cap (rejection says how much is left), no retroactive
approval (`shift_ended`, `past_date`), no extension that would only add
time after scheduled close (`after_close`: hold-over covers that), breaks
and reading blocks unchanged, nobody else extended.

To give extensions something to fill, the established roster's US and MR
templates now continue to the evening (US 18:00–20:00, MR 16:30–19:30).
Those slots are unstaffed — not bookable — unless a shift is extended; runs
without actions are identical to step 12 (checked below).

**Did the decisions matter?** Same seed (1), established practice, 5 days
from takeover, no actions vs. a plan approved at takeover: US and MR
extended to the 20:00 close on every weekday, walk-ins accepted until
19:00 (`TestInterventionsChangeOutcomesNotDemand`):

| 5 days from takeover | no actions | interventions |
|---|---|---|
| orders received / walk-ins arrived | 254 / 39 | 254 / 39 (identical, same attributes) |
| exams | 259 | 267 |
| orders lost | 27 | 22 |
| US staffed / busy, of open hours | 57% / 34% | 71% / 40% |
| MR staffed / busy, of open hours | 58% / 48% | 82% / 52% |
| arrival→exam wait p90 | 52 min | 51 min |
| overtime (of it approved) | 6 h (0) | 28 h (22) |
| net revenue / expenses | $40,558 / $43,094 | $42,641 / $44,997 |
| operating profit (baseline-adjusted) | −$2,536 | −$2,355 |

The first version only extended shifts on the day itself: US and MR were
staffed, but almost nothing was booked into the new time, because bookings
are made days ahead (+4 exams for 22 h of overtime; profit fell to −$3,488).
Hence the `date` parameter: approved ahead, the slots open for booking now.
Uncalibrated figures; the point is that management decisions now change
operations and money while demand stays exogenous.

**Tests** (`go test ./...` passes; `internal/sim` 149 s):

| Requirement | Test |
|---|---|
| Valid and invalid extensions, cap | `TestExtendShiftValidation` (16 cases), `TestExtensionAheadOpensBookings` (dates, horizon, Sunday, cap) |
| Extension vs hold-over; payroll | `TestExtensionVersusHoldOver` (36 min required vs 60 approved; one wage entry = hourly pay of on-duty time; approved cost = overtime cost) |
| Capacity, not demand | `TestExtensionAddsCapacityNotDemand`, `TestExtensionAheadOpensBookings` (evening US slots booked ahead), `TestInterventionsChangeOutcomesNotDemand` |
| Walk-in policies, declines measurable, not retroactive | `TestWalkInPolicies`, `TestWalkInPolicyNotRetroactive` |
| Worklist ordering, read times, no interruption, no double read | `TestWorklistPolicyOrdering` (shortest-first follows the published p50 only) |
| Simultaneous actions | `TestSimultaneousActionsOrdered` (order-dependent results; intermediate policies inert) |
| Invalid-action atomicity | `TestInvalidActionAtomicity` |
| Replay | `TestActionReplayEquivalence` (re-applying the record reproduces log, books and results) |
| No retroactive effects | `TestActionsNotRetroactive` |
| Chunking, unrelated events | `TestActionsChunkingAndNoise` |
| Evaluation window | `TestEvaluationAttributesActions` |
| TUI | `TestActionsScreen` (submit, reject, policy, ahead; 80×24, 120×36) |

Without actions, the ledger and every visit, study and appointment are
identical to `4bbd03f` (established 21 days, mini 30 days, seeds 1–3), and
the pinned start-state digests are unchanged.

**Performance** (MEASUREMENTS.md): warm-up 1402 ms vs 1401–1434 ms (best
runs; the Pi alternates between ~1.4 s and ~2.5 s with throttling); one
preview 4 µs, one applied action with its dispatch 12–14 µs; 30 days after
takeover 405 ms without actions, 380 ms with daily interventions.

**Known limitations.**
* Extensions are whole-shift-end only: no early starts, no extra breaks
  for long shifts (the cap bounds them), cover windows keep their times,
  radiologists' reading blocks are not extended (their supervision is).
* A cost estimate ahead uses the base rate; the ×1.5 depends on the
  week's hours when worked.
* No undo: an approved extension stays (V0.2 `send_home`).
* The evening slots exist only for US and MR in the default roster.
* Rejected actions are in the action record, not the event log.
* Run records and replay files are step 14.

---

## Step 12 — Metrics, alerts, daily summaries (v0.1.0-dev.12)

**What the player sees.** The dashboard answers "what is going wrong, and
why": an **ALERTS** panel (active alerts, most severe first, time raised,
the screen that shows more, and a "why:" line built from observed facts —
"shared technologist Riley busy (documenting P00031)", "no XR technologist
rostered until 09:30") and a **Today** line (time-weighted utilization of
staffed hours by room, arrival→exam wait p50/p90, lost orders). `e` runs
today's operations to operational close and opens the **end-of-day
summary** (TUI_DESIGN §4.10): patients, demand, schedule, waits, room time
three ways (busy of open, busy of staffed, staffed of open), staff,
reporting, money, cash, alerts and rule-based insights; `←→` pages earlier
days, `tab` shows **since takeover**. It fits 80×24 untruncated (short
forms below 110 columns). The history and briefing screens now draw on the
same measurements (room busy time within open hours, so ≤ 100%).

**Reporting windows (owner decision, 2026-10-14).** Two windows:

| Window | Purpose | Definition |
|---|---|---|
| Calendar | TUI, daily history, statements | midnight to midnight, including the takeover day's activity before the takeover |
| Evaluation | future player/agent evaluation | from the exact takeover instant to `evaluationDays` (scenario; 0 = open-ended) |

The baseline is captured at the end of the takeover transaction, after the
opening adjustment, so everything at that instant belongs to the inherited
state and the window is (takeover, end]. It is immutable and digested:
balances and trial balance, open claims by state, unreported studies,
booked appointments, recall list, patients on site, staff on duty, staffed
rooms, ledger and log positions. Window money is the journal after the
baseline (a second day book), with operating performance separated from
cash flow and working capital and a cash bridge that reconciles to the cent.
No weighted score, weights or thresholds: measurements only.

**Metric definitions** (full table: SIMULATION_DESIGN §11.1). One
cumulative tally integrates the state between transactions; a period is the
difference of two snapshots (midnight, takeover, endpoint, or now).

* Rooms: open (scheduled hours); staffed (open with a qualified technologist
  on duty; a shared technologist counts fully for each room); busy (patient
  in the room, within open hours); after-hours busy apart. Utilization of
  open hours = busy/open, of staffed hours = busy/staffed, coverage =
  staffed/open: none can exceed 100%.
* Staff: on duty (paid time, breaks out, overtime in), busy (registration,
  exam, documentation, reading), break, overtime. Technologist utilization =
  busy/on duty per person (shared time counted once). Radiologist reading
  utilization = reads inside site-reading blocks / block time; overrun reads
  apart.
* Appointments (slots starting in the period): fill = booked/bookable slot
  time; no-show rate = no-shows (missed included until accounting close) /
  (scheduled − cancelled); cancellation rate = cancelled/scheduled.
* Demand: received, booked, recall, lost, cancelled; lost rate =
  lost/received. Walk-ins: acceptance = imaged/arrived, abandonment =
  left/arrived.
* Waits (exams started in the period): arrival→registration,
  arrival→exam, exam start past the appointment time.
* Reporting: TAT of studies imaged in the period, unfinished ones censored
  at the period end; TAT of reports finalized (inherited ones counted);
  breaches; backlog at both ends.
* Workforce: overtime and its cost, held-over, skipped breaks, coverage
  reschedules, patient-time waiting to register with no desk staff.
* Ratios keep numerator and denominator (an empty denominator is "—", never
  0%); percentiles are nearest-rank everywhere (the worklist's median and
  p90 now use the same definition).

**Alert behavior.** Ten rules over observable state, from the takeover on:
waits (30 min warn, 60 min alert; measured past the appointment time, or
since arrival for walk-ins and patients fitted in after grace), unstaffed
rooms with patients on site, report TAT overdue (urgent alert, routine
warn, soon tracked since each breach is already logged), held-over staff
(tracked) and the 12-hour cap approaching (alert), closing running late (30
min warn, 60 alert), reporting backlog (≥ 40, clears below 32), lost demand
(≥ 3 per modality per day, expires at midnight), cash low (< $25,000) and
negative (alert; replaces the step 11 shortfall entry, same text). One
raise per episode, at most one escalation, a logged resolution after a
10-minute clear for waits, rooms and backlog. Time-driven changes get one
exact passive wake (no polling); passive events never trigger dispatch, so
they cannot change outcomes. Seed 1 raised 16 alerts on day 1 and 11 on
day 2 (escalations included), mostly waits behind the shared XR/CT
technologist and short unstaffed spells during breaks, plus one late close. Thresholds are configurable (`alerts` in
the scenario) and uncalibrated.

**Conventions to confirm (owner).**
1. Events at the exact takeover instant (midnight payroll or bills, 10:00
   remittances, dispatch) belong to the inherited state, consistent with the
   step 11 opening rule; the window is (takeover, end].
2. The takeover day's midnight accruals (salaries, fixed costs) cover the
   whole day; the window bears the share after the takeover (18/24 at
   06:00), shown as "takeover-day accruals" and in the cash bridge.
3. Collection variance on inherited claims is excluded from window
   operating profit (reported apart); billing fees on collecting them stay
   window expenses (the fee is a cost incurred in the window).
4. Wait alerts measure from the appointment time (an early arrival waiting
   for a punctual exam is not a problem); the wait metrics report both.

**Outcome neutrality.** Metrics, alerts and summaries change nothing in
the simulation. Checked against `23b6ff8`: ledger digest and every visit,
study and appointment identical after 21 days (established) and 30 days
(mini), seeds 1–3; the pinned start-state digests (ledger and log) are
unchanged. The log gains the alert entries.

**Tests** (`go test ./...`: all packages pass; `internal/sim` 157 s on the
Pi). New, on hand-built micro practices with fixed durations where figures
are worked out by hand, and on the mini and established practices:

| Requirement | Test |
|---|---|
| Exact utilization denominators | `TestMetricsRoomAndStaffDenominators` (XR busy 46 min of 60 open, 34 min after hours; Tess 2h06m on duty, 36m overtime, 96m busy; an unused staffed room is 0%, not undefined) |
| Shared technologist | `TestMetricsSharedTechnologist` |
| Staffed vs open | `TestMetricsStaffedVersusOpen` (staffed = open − breaks) |
| Reading blocks | `TestMetricsReadingBlocks` (and TAT percentiles, censored) |
| Percentiles, empty denominators | `TestNearestRankPercentiles`, `TestMetricsEmptyDenominators`, wait percentiles in the denominators test |
| Partial day, midnight boundary | `TestMetricsPartialDay`, `TestMetricsMidnightBoundary` (only the censored TAT ages, by one second, from 23:59:59 to midnight) |
| Exact-timestamp boundary | `TestEvaluationExactTakeoverBoundary` (10:00 takeover with remittances at that instant; window books = journal after the baseline; orders before 10:00 in the calendar day only) |
| Inherited claims paid after takeover | `TestEvaluationInheritedClaims` |
| Imaged before, finalized after | `TestEvaluationPreTakeoverStudyFinalizedAfter` |
| Capital excluded | `TestEvaluationExcludesCapitalTransfers` (+$350,000 opening cash: identical operations and operating results) |
| Accrual allocation, whole-day window | `TestEvaluationAllocatesTakeoverDayAccruals`, `TestWindowEqualsDaysFromMidnightTakeover` |
| Alerts: trigger, escalate, resolve, dedupe, expire | `TestAlertsTriggerEscalateResolve` (exact times 08:20, 08:50, 09:20, 09:40, 09:43), `TestAlertsExpireAndDeduplicate` |
| Same-instant ordering | `TestAlertSameInstantOrdering` (an exam starting at the threshold instant raises nothing) |
| Daily summary reconciliation | `TestDailySummaryReconciliation` (days sum to since-day-1; cash rolls forward) |
| No future leakage | `TestMetricsAndAlertsRevealNoFuture` |
| Chunking, speed, unrelated events | `TestMetricsChunkingInvariance`; the TUI pacing test; `TestRegressOutcomeIndependentOfUnrelatedEvents` now includes both alert fixtures |
| Session, TUI | `TestEndOfDayStopsAtOperationalClose`, `TestEndOfDaySummary`, `TestDashboardAlerts`; render tests cover finance and history |
| Scenario fields | `TestValidationErrors`: 4 new cases (`evaluationDays`, `alerts` ranges and ordering) |

**Performance** (MEASUREMENTS.md): warm-up 1402 ms vs 1366 ms at `23b6ff8`
(best of four alternating runs, Pi at 81 °C, +2.6%); one observation 1.26
ms vs 1.20 ms; the evaluation window 1.3 ms, on request only. Completed
days' summaries are cached.

**Known limitations.**
* All overtime is required hold-over until actions arrive (step 13); the
  approved/required split is not yet reported.
* Alerts start at the takeover: history days have none.
* Thresholds, insight rules and the backlog limit are illustrative
  (calibration, step 18). Insights are two simple rules.
* A day appears in the history from midnight (before: from accounting
  close); after accounting close today's summary is still "so far".
* The baseline digest covers the baseline record, not the full state (the
  full-state digest is step 14). Scenarios leave `evaluationDays` at 0.
* No turnover state in V0.1, so busy = patient in the room.
* The history screen did not gain alert or wait-percentile columns (the
  summary screen pages through days instead).

---

## Step 11 remediation — review findings (v0.1.0-dev.11.1)

The step 11 review (`docs/reviews/step-11-astra-review.md`, gpt-6-astra at
high reasoning, read-only snapshot of `e5863b4`) returned a CONDITIONAL GO
with six confirmed defects and two test gaps. For each defect a regression
test was written first and run on the unfixed code (`e5863b4`); then the
smallest fix was made and the test rerun.

| # | Defect | Regression test | Before (on `e5863b4`) | After | Fix |
|---|---|---|---|---|---|
| 1 (high) | The takeover adjustment ran before same-instant events: a later-queued day start (payroll or bills) at a midnight takeover, 10:00 remittances, and dispatch. Opening cash came out wrong and the snapshot was taken too early. | `TestRegress11TakeoverCollisions` | FAIL: 5 operating entries after the opening at midnight on payday Fri Oct 16 and on Nov 1; 87 at a 10:00 takeover. Opening cash $100,801 (payroll paid after the injection), $27,142 (bills) and $156,239 (collections) instead of $150,000. | PASS: exactly $150,000; snapshot = books when play begins; the opening is the last entry of the instant | The takeover event only marks the adjustment due. It is posted at the end of that instant's dispatch, after every event and dispatch decision. |
| 2 (medium) | The cash chart and daily records showed pro forma (shifted) history as plain cash | `TestRegress11ActualHistoricalCash`, `TestRegress11CashHistoryLabelled` | FAIL: cash history Sep 14 $195,837, actual −$372,992; screens unlabelled | PASS | Actual cash everywhere by default. `DaySummary.ProFormaCashEndCents` is a separate, named field. The history screen says ACTUAL CASH and shows a takeover line ("owner capital +$568,829 … brought cash from −$418,829 to $150,000"). The chart says "Cash (actual)" and notes the capital at takeover. The ledger is untouched. |
| 3 (medium) | `submitTime`/`paymentTime` of 24:00 accepted, scheduling onto the next (weekend) day | `TestValidationErrors` (2 new cases) | FAIL: no validation error | PASS | Point-in-time financial clocks must be before 24:00. |
| 4 (low) | Ledger hash ignored the fixed-cost item | `TestRegress11LedgerHashCoversItem` | FAIL: same hash | PASS | The item is part of each hashed line. The golden ledger digest changed for this reason only; the log digest is unchanged. |
| 5 (low) | The balance line printed "=" between independently rounded dollars (the reported $1) | `TestRegress11BalanceEquationExact` | FAIL: whole dollars | PASS | The balance and equity lines show exact cents: $428,153.55 = $88,071.43 + $340,082.12. |
| 6 (low) | Dates outside years 0–9999 wrapped (10000 → 0000) | `TestRegress11DateFormattingAllYears` | FAIL: "0000-01-01" for 10000, "000/-12-31" for −1 | PASS | The fast path covers 0–9999; others use the previous `fmt` format. |

**The $1 discrepancy.** It was display rounding. The exact balances at
takeover (seed 1) were assets 42,815,355¢, liabilities 8,807,143¢ and
equity 34,008,212¢: an exact match in cents. The ledger also enforces
Σ = 0 on every posting and in the invariants.

**Test gaps closed**
* `TestRegress11SplitEqualsWholePendingState`: building then playing 2 days
  7 h equals one uninterrupted run. It compares the full pending event queue
  and every claim, appointment, study, visit and order, not only digests.
  This passed before as well.
* `TestRegress11ConvergenceRejectsNaN`: the convergence comparator now
  rejects non-finite ratios (0/0 used to pass). It failed before.
* `TestDrawMemoEqualsFreshHash`: memoized draws equal fresh hashes, with
  independent cursors.
* `TestLogDigestIndependentOfReadsAndEviction`: the hash covers every entry
  regardless of reads and ring eviction, and retained order is kept.
* The last two passed before; they confirm the review's finding that the
  optimizations are sound.

**Not changed (recorded):**
* The convergence test remains a nine-seed smoke test. It has no variance
  estimate, the seeds are unpaired across warm-up lengths, and liabilities
  and backlog are not compared. Step 18 measures variance.
* The review's other suggested fixtures remain future work: takeover during
  required overtime; a recall, no-show rebook and stale arrival at
  takeover; one claim in every state; an overnight TAT in history.

**Owner decisions**
1. **Capital stays explicit** (owner, 2026-10-14). The takeover injection
   is shown wherever opening cash is (briefing, log, finance screen,
   history line) and is never treated as a balancing mechanism. Calibration
   is not assumed to shrink it.
2. **Historical cash** (owner, 2026-10-14): actual by default with the
   injection visible. Pro forma only as an explicitly named value. Done.
3. **Scoring windows (open).** "Today" and "since day 1" are calendar-day
   windows from 00:00. A 06:00 takeover includes the 00:00 accruals; a
   13:00 takeover includes the morning. That is fine for display. For
   benchmark scoring (step 12 onward) the recommendation is to score
   post-takeover performance from the exact takeover instant against the
   opening balance sheet, and to keep capital transfers out of operating
   performance.

Verification: the full suite passes, and all step 8 and step 10
regressions pass.

---

## Step 11 — Warm-up, opening state and performance (v0.1.0-dev.11)

**What was built.**
* `dreamteam` now opens the **established practice** by default (bundled
  scenario `established`; `mini` remains).
* It runs the documented 180-day warm-up with the ordinary engine before
  the TUI starts ("Building 180 days of practice history…") and opens on a
  takeover briefing at Tue Oct 13 06:00, day 1.
* New code:
  * `internal/sim/opening.go`: the opening adjustment and snapshot, and
    financing-shortfall tracking;
  * `history.go`: daily records;
  * `draws.go`: the draw memo;
  * `internal/tui/briefing.go`: the briefing and history screens.
* Design: SIMULATION_DESIGN §8.8; TUI_DESIGN §4.7a.

**Warm-up semantics.** Warm-up and play are one continuous run of the same
engine. Nothing is approximated, skipped or reset. These carry over as the
history left them:
* appointments (868 booked from day 1 for seed 1) and the recall list
  (48);
* pending orders and patients on site;
* studies on the worklist and reads in progress;
* claims at every stage, receivables, payables and accrued payroll;
* staff weekly hours, for overtime;
* ID counters.

**Warm-up accounting conventions**
* Cash is 0 at the epoch, and the history's collections, payroll and bills
  accumulate.
* At the start time one journal entry, the **opening equity adjustment**
  (Dr/Cr Cash ↔ Opening equity), sets cash to the scenario's $150,000.
  This is the documented convention (§8.4): owner capital in, or a
  distribution out. It is not revenue or a cash flow.
* Nothing else changes. A/R, A/P, accrued payroll and every open claim
  continue, and future payment events stay queued. The history's operating
  result stays in equity, so losses are not erased.
* Seed 1 at takeover:

| | |
|---|---|
| Cash left by the history | −$418,829 |
| Owner capital added | $568,829 |
| Opening cash | $150,000 |
| Receivables (1,870 open claims) | $278,154 |
| Payables | $48,933 |
| Payroll owed | $39,139 |
| History result (180 days) | −$223,758 |
| Last 30 days: net revenue / expenses / profit (margin) | $192,344 / $222,591 / −$30,247 (−15.7%) |
| Balance sheet at takeover | assets $428,154 = liabilities $88,071 + equity $340,082 |

* The capital is large because the uncalibrated history loses money.
  Alternatives are listed under the decisions below.
* **Revenue recognized vs net revenue.** Revenue recognized is the expected
  net booked when each report is finalized. Net revenue adds the collection
  variance on claims that settled in the period. In step 10's 180-day run
  net revenue ($1,136,003) exceeded revenue recognized ($1,125,060) because
  the claims settled so far paid $10,943 more than expected: fast payments
  in full settle first, while denials and self-pay write-offs settle 30–120
  days later. The TUI rows are now "Revenue recognized", "+ collected vs
  expected" and "Net revenue", with a note.
* **Financing shortfall.** Negative cash stays allowed, as approved, but
  it is now tracked from the start:
  * the lowest balance and when it happened, and days ended below zero;
  * one alert per dip below zero, and a red **FINANCING SHORTFALL** on the
    finance screen and the dashboard.

  No credit line or bankruptcy is modelled.
* **Underlying categories.** Fixed-cost ledger lines carry their item, so
  every period reports fixed cost by item (rent, utilities, insurance, each
  lease and service contract, IT) whatever account it posts to.
* **History boundary.**
  * Days before the start date are history: daily records, ledger and log.
  * Today's counters, demand and statements since day 1, cash change today
    and the financing figures start at day 1. Day 1's 00:00 accruals are
    day 1's.
  * The opening adjustment never appears as today's cash change.
  * The cash history and day records restate the days before takeover by
    the adjustment, so the series is continuous. This is labelled.

**On screen**
* **Briefing** (`b`):
  * money: opening cash, owner capital, last-30-day P&L and cash flow,
    carried-over receivables, payables and payroll, next payroll and bills,
    and the financing note;
  * patients: last 7 days of demand and exams, waits, utilization of open
    and staffed hours, bookings ahead;
  * worklist and TAT;
  * today's roster.
* **Daily history** (`h`): 30 days, with history rows marked "·".
* **Finance:** a balance sheet line with the equity split, a financing
  line, and the clarified revenue rows.

**Determinism and correctness tests** (`internal/sim/warmup_test.go`):

| Requirement | Test |
|---|---|
| Opening balance sheet reconciles | `TestOpeningBalanceSheet`: one opening entry, at the start, after every warm-up entry; adjustment = target − cash before; cash = $150,000; assets = liabilities + equity; equity components match the journal; A/R, A/P and payroll carried over and non-zero; day-1 "today" figures empty; history ends the day before the start, restated to the opening cash |
| Claims, payments, reports, worklist, appointments, weekly overtime survive | `TestStateSurvivesTheBoundary` (13:00 start, so patients on site and the worklist cross the boundary). Every study pending at the start is finalized once. All of the 1,000+ claims open at the start settle within 145 days (the longest path). Warm-up bookings resolve. Riley's weekly paid time at the start equals Monday's pay exactly. |
| Warm-up vs uninterrupted run; chunking; unrelated events | `TestWarmupDeterminism`: a daily-chunked warm-up, a warm-up with live and stale noise events, and a rebuild give identical log and ledger hashes and identical observations; build then 3 days of play = one uninterrupted run; seed 2 differs |
| Golden start state | `TestStartStateGolden` (ledger and log digests, seed 1) |
| No hidden future in the opening state | `TestOpeningStateHidesTheFuture`: forcing every patient from day 2 to no-show and every claim adjudicated after the start to fail leaves the observation, the 180-day history and tomorrow's schedule unchanged; afterwards the books differ |
| Convergence (ROADMAP) | `TestWarmupConvergence`: nine-seed means, 180 vs 365 days. Forward fill to 90 days, A/R and A/R ≤ 60 days within ±5% (largest: 7–30-day fill −4.6%); tails within ±25%. A 90-day warm-up fails (−12.6% fill, no A/R over 90 days). |
| Financing shortfall | `TestFinancingShortfallTracked`: $10k opening cash; an alert per dip; lowest cash and days below zero agree with the daily history |
| Optimizations change nothing | Formatter equivalence tests against `fmt`. Before the step-11 behavior was added, a full warm-up plus 3 days produced byte-identical log text (61,226 entries) and ledger (`a37b36…`) to `32313ac`. |

All step 1–10 tests pass, including the step 8 remediation regressions and
the step 10 finance tests. The full suite takes about 2 minutes on the Pi;
the convergence test is 30 s of that and is skipped with `-short`.

**Performance (before / after).** Warm-up build, alternating old and new
with cool-downs:
* step 10: 1.80–2.38 s/op, 111.6 MB, 1.97M allocations;
* step 11: 1.35–1.97 s/op, 89.7 MB, 1.26M allocations.

That is about 22–25% faster at the median. The spread is thermal: the Pi
ran at 75–79 °C with throttling, and unthrottled builds take 1.34–1.35 s.
Details and the remaining profile are in MEASUREMENTS.md:
* What changed: lazy log time text, a streamed log hash, Sprintf-free
  ID and time formatting, a per-day memo of keyed draws (30% were
  repeats), the claim draw kept on the claim, and reused room staffing
  lists.
* What didn't: no randomness semantics, no event suppression and no
  concurrency.

**Remaining limitations**
* **Target ≤ 1.5 s only when unthrottled.** Throttled builds take
  1.7–1.9 s. The next steps are a lazy structured log, or the start-state
  cache once snapshots exist (step 14).
* **The worklist is empty at 06:00 on day 1.** Monday's studies are all
  read by Monday evening at the current (below-design) volume; the design's
  10–15 unread is a calibration matter (step 18).
* **Large owner capital.** The history loses money, so takeover needs
  $569k of owner capital. This is expected to shrink to a working-capital
  figure after calibration.
* **Utilization is minimal.** The daily records keep only exam time over
  open and staffed hours; the full metric set is step 12.
* **No financing model.** No credit line, interest or insolvency (V0.2).

**Decisions for owner approval**
1. **Opening capital convention** (implemented as documented): one
   adjustment at takeover, with the history's balances and losses kept.
   Alternatives:
   * (a) opening cash at the epoch and no adjustment, so the start cash is
     what the history leaves (−$268,829 for seed 1);
   * (b) no opening capital (−$418,829);
   * (c) keep the adjustment and calibrate so the history is about
     break-even (step 18).

   Recommendation: keep the adjustment, aim for (c).
2. **Restated cash history.** Before day 1 it is shown as if the opening
   capital had always been there (continuous series, labelled). The
   alternative is the raw series with a jump at takeover.
3. **The log digest format changed** (streamed SHA-256). Digests differ
   from step 10 by construction; outcomes do not.

---

## Step 10 — Ledger, claims, payroll and costs (v0.1.0-dev.10)

**What was built.**
* A double-entry ledger in integer cents (`internal/sim/ledger.go`).
* Revenue at report final, the claim lifecycle, payroll, and fixed and
  variable costs (`finance.go`).
* Statements, claims and the journal in the API (`finance_view.go`,
  `api.Finance`), and the TUI finance screen.
* All the assumptions are scenario data: the new `finance` section and
  catalog `baseRateCents` / `supplyCostCents`, taken from §8.2–8.3 of the
  design. They are illustrative and not tuned. Design detail is in
  SIMULATION_DESIGN §8.7.

**Accounting conventions**
* **Accrual for profit, cash for liquidity, one ledger.** Revenue is earned
  when the radiologist finalizes the report: Dr A/R / Cr revenue at expected
  net = base × payer multiplier × NCR. It is never earned at booking,
  arrival, registration, imaging, worklist entry or reading start. Cash
  moves only when a payment arrives or payroll or bills are paid. The
  practice can be profitable and short of cash at the same time.
* **Every entry** balances and is non-empty, and has a stable idempotency
  key (study, claim and payment number, employee and day, or day). It is
  never edited. Corrections are linked reversing entries, which can be
  posted once and never reversed themselves. The incremental check re-sums
  new entries independently, the day books must equal the balances, and
  the accounts sum to zero.
* **Claims.**
  * Pending, then submitted 1–3 business days after final, at 17:00 in one
    batch. Remittance comes after the payer's delay, on a business day at
    10:00.
  * Outcomes:
    * paid in full;
    * short paid at m ± 10 pp;
    * denied, then resubmitted the next business day, then recovered or
      written off 30–60 days after the denial;
    * self-pay unpaid, written off and not counted as a denial.
  * Settlement sends expected − paid to collection variance (contra
    revenue), so a closed claim has zero A/R.
  * Outcome, delay, short fraction, rework time and charge lag come from
    the order's origin key (`claim`, `claim.lag`). The payer class is drawn
    when the order is disclosed.
* **Payroll.**
  * Hourly staff accrue when they go home: worked time minus breaks taken,
    with skipped breaks and held-over time paid. Pay is at the regular rate
    until the week reaches 40 paid hours, then ×1.5 (the step 8 rule).
  * Salaried staff accrue annual ÷ 365 per calendar day. Radiologists
    accrue × their site allocation.
  * Benefits are 25% on everything.
  * Payroll is paid every other Friday from Oct 16 at 00:00 and covers
    everything accrued through Thursday.
* **Costs.**
  * Fixed monthly items accrue daily with exact monthly totals.
  * Supplies (contrast included) and $3 PACS post at acquisition.
  * Billing is 5% of each payment.
  * Accounts payable is paid on the 1st.
* **Statements.**
  * Today, month to date, trailing 30 days and since day 1.
  * Net revenue = revenue − collection variance.
  * Expenses are shown by account, and also as labor, fixed and variable.
  * Operating profit and margin; the margin is undefined at zero revenue.
  * Collections, disbursements and net cash flow.
  * Denial rate, days in A/R and A/R aging.
  * Revenue and direct cost by modality, plus a cash history.

**On screen**
* `f` finance has three views, switched with `tab`:
  * **overview**: periods side by side, cash and the next payroll and
    bills, A/R and denials, the modality table and a cash sparkline;
  * **claims**: open claims, oldest first;
  * **journal**: the latest entries as debit and credit lines.
* The dashboard's second line shows cash ▲/▼ today, revenue, expenses and
  operating profit today, and A/R.
* The report-final log line now ends "· revenue $395".
* The log adds lines for claim batches, payments, short payments, denials,
  write-offs, payroll and bills.
* Patient journeys continue through revenue, claim, denial and
  resubmission, and payment or write-off.

**Tests** (`internal/sim/finance_test.go`, `ledger_internal_test.go`, a TUI
test, and scenario validation cases):

| Requirement | Test |
|---|---|
| Revenue once, only at final; none for cancelled, no-show, left, lost, rescheduled-unperformed or unread exams | `TestRevenueOnlyAtReportFinal`: minute by minute on day 1, then 30 days; non-vacuity checks for each excluded state |
| Correct modality and payer reimbursement | `TestFinanceArithmetic` (NCR 0.950/0.980/0.934/0.400; CT w/ contrast, commercial = $416 allowed, $395.20 expected), `TestClaimLifecycleAndTiming` (every claim's allowed and expected amounts) |
| Claim lifecycle, payment timing, denials, receivables | `TestClaimLifecycleAndTiming`, 150 days. For every claim, the drawn charge lag, remittance date, rework date, state and paid amount match exactly. Non-vacuous for paid, short, denied → paid, denied → written off, self-pay unpaid, open and pending. A/R per claim and cash are rebuilt from the journal alone. |
| Collection variance mean ≈ 0 (ROADMAP) | `TestCollectionVarianceMeanZero`: 30,000 draws per payer; the collected fraction is within 0.6 pp of NCR |
| Balanced entries, idempotency, reversals | `TestLedgerPostingRules`: unbalanced, empty, duplicate, out-of-order and bad lines are rejected with no effect; reversal mirrors and links; no double reversal; tampering is detected. Every simulation test with invariants re-sums the journal. |
| Cash and receivable reconciliation | `checkFinance` invariants (cash = opening + receipts − disbursements from independent totals; A/R = open claims; revenue = Σ expected; claims = final reports; accrued payroll and A/P), checked after every transaction in all invariant-checked tests and at each day boundary in the finance tests |
| Payroll, regular and overtime; weekly threshold crossing | `TestPayrollWeeklyOvertime` (`micro-week.json`): Tue–Thu $630; Fri crosses 40 h after 8.5 h, $690; Sat all premium, $360; Monday resets, $630; unpaid lunch excluded; benefits 25% |
| Required overtime after scheduled close | `TestPayrollRequiredOvertime`: wages = scheduled part + exactly the step 8 overtime cost (36 min held over: $126 = $90 + $36; premium fixture $189 = $135 + $54); one entry per employee and day |
| Fixed and variable cost timing; day, week and month boundaries | `TestCostAndPaymentTiming`, Oct 13 to Dec 1: one fixed accrual per day at 00:00; November totals exactly the monthly sum; A/P paid Nov 1 and Dec 1 equals the balance before; supplies and PACS once per study at acquisition; billing fee = 5% of each payment; payroll only on the four anchored Fridays, equal to the accrual through Thursday; salaries once per employee and day; wages on the day worked |
| Same-instant ordering | The same test checks that at 00:00 bills and payroll are paid before that day's accruals. Claim batches and remittances are class 6, after operations at the same instant. |
| Sparse events, speed, replay | `TestFinanceDeterminism`: 75 days in one jump, daily, or in 13-minute chunks give identical ledger and log hashes. A replay is identical; a different seed differs. |
| Independence from unrelated events | `TestFinanceUnaffectedByUnrelatedEvents` (live and stale events at 10:00, 17:00 and around them for 50 days); the step 8 wake-completeness tests now also compare the ledger hash |
| Outcomes independent of operations | `TestClaimOutcomesIndependentOfOperations`: worklist FIFO vs shortest-first changes finalization times, but no payer, amount or outcome |
| Hidden outcomes not leaked | `TestFinanceOutcomesNotLeaked`: forcing every claim to fail leaves the full observation identical for 15 days (and all journeys on day 15), then changes cash. The latent-field deny-list now covers `api.Finance`. |

Mutation checks (each made the named test fail, then was reverted):
* revenue at exam end;
* claim draw keyed by processing order;
* weekly threshold ignoring earlier days;
* held-over time unpaid;
* payables paid on the 2nd;
* charge lag in calendar days (caught once the test checked the exact
  drawn lag);
* an observation revealing a forced outcome;
* a wrong short-paid amount;
* a cash identity off by a cent.

All step 1–9 tests pass, including the step 8 remediation regressions. The
fixtures gained rates and a `finance` section, but no expectation changed.

**Measured: 180 days of the mini practice, seed 1** (2.1–2.5 s/op on the
loaded Pi, MEASUREMENTS.md):

| | Since day 1 (181 days) | Trailing 30 days |
|---|---|---|
| Revenue earned (expected net) | $1,125,060 | $195,429 |
| Collection variance (favorable) | +$10,943 | +$2,429 |
| Net revenue | $1,136,003 | $197,858 |
| Operating expense | $1,336,861 | $222,177 |
| — labor (wages $508,015 + benefits $127,005) | $635,020 | $103,677 |
| — fixed | $529,192 | $86,692 |
| — variable (supplies $108,567, PACS $58,461, billing $41,498) | $172,649 | $31,809 |
| Operating profit | −$200,858 | −$24,319 |
| Operating margin | −17.7% | −12.3% |
| Cash collected | $829,949 | $179,528 |
| Disbursements | $1,260,067 | $223,235 |

* At the end: cash −$280,118 (opening $150,000); A/R $306,054 (46 days);
  A/P $43,247; accrued payroll $33,547.
* Denial rate: 5.9% (309 of 5,199 insurance adjudications).
* Claims: 7,514. Paid 4,935, short paid 249, written off 270, open 2,060.
* By modality (net revenue / direct cost):
  * XR: 3,538 reports, $182,866 / $34,074;
  * US: 1,216 reports, $213,322 / $30,321;
  * CT: 1,833 reports, $436,681 / $180,790;
  * MR: 927 reports, $303,134 / $223,453.
* Hot spots: SHA-256 22%, dispatch 23%, log formatting 14%, finance about
  10% (adjudication 4%, recognition 3%, posting 2%).

**Why the margin is negative.** Volume, not the financial assumptions.
Monthly-equivalent figures:
* Labor ($106.7k vs the reference $105.1k) and fixed costs ($88.9k vs
  $88.5k) match the §8.6 reference calculation.
* Net revenue is $190.8k vs $261.5k. That is about 1,260 reports a month
  instead of 1,650, the volume shortfall noted since step 7. Revenue per
  report is $151 vs $158.5, from the exam mix.

Nothing was tuned toward the 8.5% target, as instructed. That is step 18.
The cumulative collection variance is favorable (−1.0% of revenue) because
claims that settle early are mostly paid in full. Denials and self-pay
write-offs settle 30–120 days later. Over all outcomes the mean is zero,
as the draw test shows.

**Known limitations**
* The negative margin and cash are a volume artifact until calibration
  (step 18).
* Opening cash is posted at the epoch. The warm-up opening-equity
  adjustment at the start time is step 11.
* No credit line or interest: cash simply goes negative.
* No copays or deductibles (not specified). Bad debt and denials share
  collection variance (split in V0.3).
* No holidays; business days are Mon–Fri.
* No prepaid expenses (rent is paid in arrears on the 1st).
* No payroll tax remittance schedule separate from payroll.
* No reversal flow in V0.1 operations. The mechanism exists and is tested.
* The end-of-day summary's finance line comes with the summary screen
  (later step).
* Performance: finance adds about 0.3–0.4 s per 180 days. The step 11
  warm-up target now depends on the planned mitigations.

**Decisions for owner approval** (each is scenario data or a small,
documented rule; SIMULATION_DESIGN §8.7):
1. **Rework timing.** "Resubmitted after ReworkDelay" is read as denial →
   final outcome in 30–60 days, with the resubmission on the next business
   day.
2. **Remittance clock.** Business days only; submissions in a 17:00 batch,
   payments, denials and write-offs at 10:00.
3. **Self-pay.** A statement, not a claim. Non-payment is written off at
   the due date, never reworked, and excluded from the denial rate.
4. **Chart mapping.** Utilities and insurance go to *Rent & facility*; the
   monthly IT/RIS charge to *IT / PACS*; leases and service to *Equipment
   service & leases*, tagged by modality. No new accounts.
5. **Payables.** All of them (rent included) are paid in arrears on the
   1st, because the chart has no prepaid account. Payroll is paid at 00:00
   on the anchored Friday (Oct 16, then every 14 days) and covers what was
   accrued through Thursday, with benefits paid alongside.
6. **Salaried accrual.** Per calendar day (annual ÷ 365), weekends
   included. Radiologists × site allocation.
7. **US base rates** within the documented $140–170: abdomen $170, pelvis
   $155, venous $160, thyroid and renal $140.
8. **Negative cash** is allowed and shown in red (no financing in V0.1).

---

## Step 9 — Worklist, reading and turnaround (v0.1.0-dev.9)

**Behavior**

* **Studies**: acquired at exam end (imaging complete, TAT start), on the
  worklist once the technologist's documentation is done, then read, then
  final. There is one study per imaged visit, never a report before imaging,
  and never a second finalization.
* **Radiologists read** during their site reading blocks while on regular
  duty. Remote radiologists read too. Each claims the first study in worklist
  order whose longest read (p100) ends by their scheduled shift end, so
  reading never extends a shift or crosses midnight. One read at a time.
  Reads that run past a block's end finish, and the overrun delays the next
  block (§6.7, with a wake-up so the claim happens on time).
* **Supervision** stays presence-based and compatible with reading, so a
  radiologist can read while supervising. The supervisor of an exam in
  progress stays until it ends (step 8 remediation). A remote radiologist
  never supervises. A held-over radiologist claims no new reads.
* **Read times** are per case (catalog `readSec`, keyed by the order's
  origin), so worklist order and policy never change them.
* **Worklist order** (default urgent then FIFO): urgent studies and studies
  already past their TAT target first (no starvation), then time on the
  worklist, with the study ID as the final tie-break. FIFO, oldest-first and
  shortest-first (by the published expected time) are ready for the step-13
  action.
* **Turnaround**: TAT = imaging complete → report final. The figures shown
  are median and p90 over studies acquired today with unfinished ones at
  their age (censored), worklist wait, read time, backlog by modality and
  priority, the oldest unfinished study, and breaches. Targets: urgent 60
  min and routine 24 h (documented), soon 4 h (illustrative). Overdue
  urgent and soon reports are logged once.
* **After operational close** radiologists keep reading in their blocks.
  Operations close without waiting for reports, and the backlog carries into
  the next day (Saturday's afternoon studies are read on Monday).

**On screen**: `w` worklist (reading and unread studies, turnaround,
backlog, latest reports; `enter` opens the patient), a dashboard reporting
line, radiologists shown as reading / ready to read / other duties /
supervising, and journeys that continue to the final report and its TAT.

**Tests** (`internal/sim/reading_test.go`, new micro fixtures):

* A month of the mini practice:
  * exactly one study per imaged visit;
  * timestamps ordered, no duplicate finalization;
  * every claim inside the claimer's site block;
  * every read ends by shift end;
  * no radiologist reads two studies at once;
  * backlog crosses days.
* Priority and ties: the urgent study is read first, and two studies
  entering at the same instant go in ID order.
* TAT definition (acquired 08:23, final 08:29 → 6m).
* Overrun delays the next block to 09:51 (claim exactly then); a read ending
  exactly at block end owes nothing (claim at 09:40, same-instant ordering).
* Radiologist distinctions (ROADMAP): remote reads, never supervises; on-site
  in other duties supervises, claims no reads.
* Read time unchanged when the policy reorders the worklist. A deliberate
  dependency on claim time makes this test fail.
* Observation types carry no latent fields.
* Outcomes unchanged by injected live or stale events (now including the
  reading fixtures).
* All step 8 remediation tests, chunking and speed tests pass.

**Measured**: 180 days of the mini practice in 1.7–1.9 s on the loaded Pi:
7,528 studies, 7,514 final, 14 outstanding at the end. Reading is not a hot
path. See MEASUREMENTS.md.

**Limitations**

* Volumes are below calibration, so the reading backlog is small. Weekday
  TAT medians run 20–40 min, and Tuesday mornings start nearly clear (the
  design expects 10–15 unread from warm-up at full volume, steps 11 and 18).
* No report interruptions, addenda or protocoling (V0.2).
* Revenue at finalization is step 10.

---

## Step 8 remediation — review findings 1–10 (v0.1.0-dev.8.1)

The step 8 review (`docs/reviews/step-08-astra-review.md`, gpt-6-astra
at high reasoning) found 10 defects, all confirmed. For each, a regression
test reproducing the reviewer's scenario was written first and run against
the unfixed code (`ee26417`), then the fix was made and the same test
rerun. Tests are in `internal/sim/regress_test.go`, with fixtures in
`internal/sim/testdata/micro-*.json`.

| # | Defect | Regression test | Before (on `ee26417`) | After | Resolution |
|---|---|---|---|---|---|
| 1 | Held radiologist released while supervising; colleague kept past her cap | `TestRegress01RadiologistHandoverCap` | FAIL: Dr. Bo overtime 0s (want 33m), Dr. Rae 3m (want 0, cap 09:30) | PASS | The supervising radiologist is recorded at exam start and stays until the exam ends. A waiting contrast patient keeps a radiologist only if she could supervise in time and no on-site colleague on regular duty could. New invariant: every contrast exam has its supervisor on duty, on site, within cap. |
| 2 | Held employee kept past usefulness and cap (no wake) | `TestRegress02HeldPastUsefulnessAndCap` | FAIL: Tess overtime 45m, want 0 | PASS | "Needed" now uses the patient's earliest possible start and the full remaining task. A single deadline wake fires when a held employee stops being useful or a patient stops being servable. |
| 3 | Desk staff who can't fit a registration block self-registration; patient stranded | `TestRegress03DeskCannotBlockSelfRegistration` | FAIL: panic, 1 visit still on site at accounting close | PASS | Desk and ops staff block self-registration only while on regular duty and able to fit a registration. They never start registrations after shift end. The deadline wake covers the moment they stop fitting. |
| 4 | Last technologist released while their patient is registering | `TestRegress04HoldForRegisteringPatient` | FAIL: patient rescheduled "no XR technologist for the rest of today" | PASS | Registering patients count as outstanding for hold-over. |
| 5 | Cap at midnight: documentation ends at 00:00, after day start; overtime lost | `TestRegress05MidnightBoundary` | FAIL: overtime 0m recorded, want 15m | PASS (scenario now rejected: cap must end by accounting close) | Day boundary: shift start + max hours must be ≤ 23:59:59. Accounting close sends home anyone past shift end (overtime booked to that day) and fails loudly if anyone is busy. The tradeoff against carrying work across midnight is documented in SIMULATION_DESIGN §6.9. |
| 6 | FirstFit skipped an earlier preferred slot in a later room | `TestRegress06FirstFitPreferredAcrossRooms` | FAIL: booked room 0 at 15:00, want room 2 at 13:00 | PASS | The early exit compares any-window and preferred candidates separately. |
| 7 | Shared capacity checked only at slot start | `TestRegress07SharedCapacityAcrossSlot` | FAIL: XR booked 09:00 on day 7 although its technologist is booked in CT 09:30–10:00 | PASS (XR goes to the next free day) | Slot technologist sets cover the whole slot. `sharedFree` checks every constant-coverage stretch of the slot. |
| 8 | Opening schedules bypassed shared capacity | `TestRegress08OpeningRespectsSharedCapacity` | FAIL: no validation error | PASS | `sim.CheckOpening` books the opening schedule in order under the same rules, and scenario validation reports problems by JSON path. The mini opening Tuesday moved two CT appointments (08:00 → 09:15, 09:00 → 11:00), and `micro-shared.json` now alternates rooms. |
| 9 | A second break overwrote a deferred first break | `TestRegress09DeferredBreaksDoNotOverwrite` | FAIL: break taken 15m, want 30m | PASS | A break falling due while busy or on break waits its turn after the current one. An older waiting break is skipped and counted, never silently dropped. |
| 10 | Negative wait for patients not yet imaged | `TestRegress10WaitWhileRegistering` | FAIL: registering, waited -480 min | PASS | Waited runs to now while waiting, to exam start once imaged, and to departure otherwise. |

**Missing-wake detection.** `TestRegressOutcomeIndependentOfUnrelatedEvents`
reruns ten micro practices with no-op events (handled, so dispatch runs)
every 7 minutes, and with stale events. On `ee26417` four of them changed
outcome (`micro-cap`, `micro-contrast-cap`, `micro-cap-wake`,
`micro-desk-cap`). After the fixes none do. A four-day version on the full
mini practice with invariants on (`TestRegressMiniUnaffectedByUnrelatedEvents`)
passes. It also passed on `ee26417`: the mini practice rarely reaches these
edges, which is why the micro fixtures exist.

**Expected changes to existing tests.** In `TestCoverageExhaustedReschedules`
the patient is now rescheduled at 09:09:01, the second their last feasible
start passes (cap 09:30 minus 21 min). Before, it happened at the next
unrelated event (09:14). The contrast hold-over now logs "supervising a
contrast exam". `TestSharedTechnologistContention` samples at 08:20 because
its bookings now alternate rooms.

**Verification.** The full suite passes. That covers the step 8 closing
tests (acceptance criteria rechecked: patients served after scheduled
close, required overtime costed, explicit coverage reschedules, contrast
after hours, operational close after scheduled close), month-long
invariants, minute-by-minute chunking, and identical results across speeds
and event budgets. 180 days of the mini practice take 1.4–2.0 s on the
loaded Pi (MEASUREMENTS.md, step 11 risk).

---

## Step 8 — Registration, imaging, overtime and closing rules (v0.1.0-dev.8)

**Planning changes first** (your decision): scheduled closing time no
longer cuts off admitted patients. PLAN D21, SIMULATION_DESIGN §3.4.2 (three
closes) and §6.9 (closing time and required overtime), the §6.4–6.5 rule
changes, invariants 7–8a, closing metrics, ROADMAP step 8, acceptance
criterion 5a, an exploit risk row, and the TUI closing status.

**Behavior**

* **Registration**: front desk, then the operations manager (×1.25), then,
  only with nobody desk-capable on duty, the patient's technologist (×1.5).
  The queue is urgent first, then scheduled patients by appointment time,
  then walk-ins.
* **Imaging**: patients across all free rooms are ranked by the queue policy
  (urgent, due by appointment, late fitted in, walk-ins under the walk-in
  policy, early arrivals) and started when a qualified technologist on the
  room is free and can finish by their 12-hour cap. Contrast also needs an
  on-site radiologist whose cap covers the exam. Room time and documentation
  are drawn per exam. The technologist documents for 1–12 min after the
  room is released.
* **Shared technologists**: one person, one exam. Booking now respects it
  too: a slot whose technologist also staffs another room isn't bookable
  while that person is already booked there (schedule shows `¦`).
* **Scheduled close** stops admissions only. Admitted patients are still
  registered and imaged. Staff past their shift end are **held over** while
  admitted patients need them (a technologist for the rooms they had, an
  on-site radiologist for contrast). They are released at the first instant
  they're not needed. Overtime minutes and cost (×1.5 past 40 paid hours a
  week, hourly staff) are recorded per person and day.
* **Coverage runs out** → the patient is rescheduled explicitly with the
  reason (for example "no XR technologist can finish within the 12-hour shift
  limit", "no radiologist on site to supervise contrast"), on arrival or
  while waiting. Nobody is sent home for the clock.
* **Operational close**: the first moment after scheduled close with nobody
  on site and nobody mid-task. It is logged with after-hours exams and
  overtime. Accounting close at 23:59:59 checks nobody is still on site.
* Breaks due while busy are deferred up to 60 min, or skipped (counted).
* **Template change [illustrative]**: Saturday XR/CT reduced to 12 + 7
  staggered slots. Sam works both rooms alone and registers everyone.

**On screen**

* Practice line: `● OPEN 08:00–20:00`, then `◐ CLOSING — 3 patients
  remaining · OT +42m ($38) after 20:00 · held over: Sam`, then
  `○ CLOSED · operations closed 20:47 (scheduled 20:00) · OT +1h 12m ($63)`.
* Rooms show scans in progress (`● Scanning P00614 Head w/o 14/12m Sam`,
  amber when past the expected time) and how many are waiting. Staff show
  their task (registering, exam, documenting) and overtime.
* Patient flow: on site by stage (registering, waiting by modality, in
  exam), longest wait, exams done, after-hours exams, coverage reschedules.
* Journey: registered (by whom, when), exam (technologist, room, wait, time
  taken against the expected time, after scheduled close), departed or
  rescheduled with the reason.

**Tests**

* Micro practices (`internal/sim/testdata/micro-*.json`, fixed durations)
  check exact timelines:
  * waiting patients stay after scheduled close, and an exam starts after it;
  * operations close at 09:36 against a scheduled 09:00;
  * 36 min of required overtime costs $36, or $54 past 40 weekly hours;
  * the shift cap forces an explicit reschedule with its reason;
  * a shared XR/CT technologist never runs two exams at once;
  * an on-site radiologist is held over for contrast after hours, and runs
    out of cap with an explicit reason;
  * patience expiry beats an exam start at the same instant, and a patient
    being registered can't leave.
* Across a month of the mini practice: every visit ends imaged, left, or
  rescheduled with a reason, overtime is recorded, and some days close
  late. Invariants are checked after every transaction: shift cap, one task
  per person, room occupancy, timestamp order, nobody on site at accounting
  close.
* Determinism: minute-by-minute chunks and various speeds and budgets give
  identical results.

**Measured on the mini practice** (28 days, seed 1): wait from arrival to
exam has median 14 min, p90 44 min, max ≈ 3 h (busy Saturdays and
evenings). Overtime averages ≈ 30 min a day (≈ $15). There are a few
after-hours exams a week.

**Limitations**

* No study or report yet. The worklist and radiologist reading are step 9.
  Radiologists only supervise contrast here.
* Held-over staff take no breaks. There is no minimum rest before the next
  shift yet (V0.2), so a late night doesn't thin the next morning.
* `extend_shift` (approved overtime) comes with the actions in step 13.
  Until then all overtime is *required*.
* Performance: 180 days of the mini practice take ≈ 1.3–1.7 s on the loaded
  Pi. Step 11 must apply the warm-up mitigations (MEASUREMENTS.md).

---

## Step 7 — Arrivals, missed appointments, walk-ins, patience (v0.1.0-dev.7)

**Behavior**

* **Scheduled patients arrive** early (≈ 60%, 5–25 min), on time (≈ 28%),
  late (≈ 9%, 5–20 min) or very late (≈ 3%, 20–45 min), or never. The no-show
  rate is a base per modality (XR 4%, US 7%, CT 6%, MR 8%) plus 0.5 points
  per 10 days of booking lead, capped at +6. Whether and when each patient
  comes is drawn by origin key when the appointment's day starts, and stays
  hidden until it happens.
* **Missed → no-show:** at start + 15 min an absent patient's appointment is
  marked *missed* (provisional). If they arrive later the same day they are
  fitted in (late policy `fit_in`) or rescheduled (`reschedule`). At
  accounting close (23:59:59) appointments still missed become *no-shows*.
  40% of no-shows ask for a new date the next morning. A patient who
  arrives after closing is rescheduled.
* **Walk-ins** (≈ 8 per weekday, about 85% X-ray, some venous ultrasound and
  head CT, never MR) arrive only in open hours, peaking late morning and
  late afternoon. Each becomes a patient and an order like any referral. If
  their modality has no technologist for the rest of the day (or the
  walk-in policy is `schedule_later`) they are booked an appointment and
  leave. Otherwise they wait.
* **Patience:** a waiting walk-in leaves after 30–120 min (hidden until it
  happens) and is counted as having left without an exam.
* **Closing time:** no exam starts after closing, so anyone still waiting is
  sent home unseen. **Until imaging exists (step 8) that is everyone who
  arrives**, so the waiting room fills through the day and empties at close.
* **IDs:** visits get `V…` IDs when the patient arrives. Each appointment and
  each order has at most one visit.

**On screen**

* The top bar shows the version: `DREAMTEAM v0.1.0-dev.7`.
* Dashboard **PATIENT FLOW**: waiting now by modality and the longest wait,
  no-shows at the last close, today's arrivals (late, after grace),
  walk-ins, walk-ins who left, provisional misses and patients sent home
  unseen. Each room line shows how many are waiting for it. On 80-column
  terminals the staff summary moves to the practice line, and the log keeps
  5 entries (8 on larger terminals).
* **Patients (`p`)**: a day's list with appointment time, actual arrival
  (with the offset when more than 5 min off), time waited and status.
  Filters (`tab`): all, waiting, missed / no-show, left / unseen, walk-ins.
  `←→` pages days. `enter` opens the **journey**: ordered or walked in,
  booked, missed (provisional), arrived (how early or late), waiting, left,
  sent home, no-show, rescheduled, lost. Only observed facts appear.

**Tests**

* Arrival offsets match the punctuality classes over 8 weeks. No-shows run
  4–10%, are finalized only at accounting close, and never have a visit.
  Patients who were marked missed and then arrived appear in the journey as
  "Missed (provisional) > Arrived".
* Conservation: every appointment of a finished day ended arrived, no-show,
  cancelled or rescheduled, with at most one visit per appointment and per
  order. Invariants (visit ↔ appointment ↔ order states, waiting list) are
  checked after every transaction in the simulation tests.
* Same instant: an arrival beats a grace check and a cancellation whatever
  the push order; grace then cancellation leaves the appointment missed; a
  patience expiry at closing time is handled before closing.
* Walk-ins: 6–12 per weekday, ≥ 75% X-ray, no MR, only eligible exams, only
  in open hours. Patience: departures between 30 and 120 min, counters match.
* Hidden-state noninterference: forcing every later patient to be a no-show
  and removing later orders and walk-ins changes nothing observable before
  it could be seen. A deliberate leak makes this test fail.
* Speed: steps of 1 min, 7 min and 2 h, and an event budget, give identical
  observations and patient lists. The existing booking, coverage and
  determinism tests pass.

**Limitations**

* No registration or imaging yet (step 8), so the waiting room only empties
  at closing time and rooms stay idle with patients waiting.
* `accept_if_gap` and `accept_always` both let walk-ins wait. The gap rule is
  applied when exams are dispatched (step 8).
* A no-show's new-date request comes the next calendar day at 09:00, even on
  Sunday (booking is centralized).
* Performance: 180 days of the mini scenario now take ≈ 0.8 s on the Pi
  (MEASUREMENTS.md). The step-11 warm-up target (< 1.5 s) will need the
  noted mitigations.

---

## Step 6 — Demand and booking (`ff54f40`)

Referral orders from six fictional sources arrive through the day. Counts,
times and attributes are keyed by origin, and IDs are given only when an
order arrives. FirstFit books slots that the roster staffs for the whole
slot (contrast slots also need an on-site radiologist). It tries the
patient's preferred time of day, then any time, within their wait
tolerance. Otherwise the order is lost (declined, or no slot within 120
days). Orders wanted beyond the horizon go to a recall list. Hidden
cancellations free the slot at once, and half the patients ask for a new
date from the next day. The mini scenario has a hand-built Tuesday of 47
appointments. TUI: dashboard DEMAND panel, compact dashboard log, full log
(`l`), schedule (`s`) with 15-minute timelines, slot list, day paging and
demand against capacity. Tests: determinism, demand unchanged by booking
capacity, feasible and non-overlapping bookings, demand accounting, recall
and cancellation paths, ID noninterference.

## Step 5 — Scenarios, rooms, staff, roster (`6ed41f3`)

Scenario JSON with `extends` and preset merging, strict decoding and
validation errors that name the JSON path. The bundled practice and the mini
scenario. Fortnight rota (weeks A/B), breaks, cover windows, radiologist
site reading blocks. Room states closed / unstaffed / idle, with one
staffing rule shared by the runtime and the coverage report
(`dreamteam scenario coverage mini`). TUI rooms and staff panels. Tests:
coverage timelines for Mon/Tue/Sat in both rota weeks, and the runtime
matches the plan minute by minute over a fortnight.

## Step 4 — API, session, TUI shell (`7ef6abf`)

Observation contract, session pacing (advance, event budget, step to the
next visible entry), Bubble Tea TUI with five speeds, pause, step and
auto-pause. The TUI cannot import the simulation (tested). Frame rate and
speed never change outcomes (tested).

## Step 3 — Event engine (`76476b2`)

Timestamp transactions with event classes, stale-event skipping, invariant
hooks. Chunking invariance tested, plus a synthetic 180-day load benchmark.

## Step 2 — Randomness (`0cbcf95`, `214d2d9`)

SHA-256-keyed draws with four fixed lanes per key, ChaCha8 streams,
integer-only uniform, Bernoulli, weighted, quantile and Poisson samplers,
golden values cross-checked against Python.

## Step 1 — Simulated time (`a3b4015`)

Integer seconds, practice calendar, clock and date formatting.
