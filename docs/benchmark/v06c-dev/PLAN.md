# V0.6c plan: conservative v2 and the frozen-agent comparison

Plan version `v06c-plan/1`, written 2026-10-28 and committed before any
run on its seeds. V0.6c has two phases:
* **Phase 1:** qualify a second financially conservative baseline,
  `conservative-v2` ([BENCHMARK_SUITE.md](../../BENCHMARK_SUITE.md) §7.2).
* **Phase 2:** run the first standardized comparison of frozen LLM agents
  on the five-scenario suite.

**Neither phase is a ranking:** no composite score, no leaderboard, no
significance test, and no claim beyond these seeds and models. A credible
benchmark must be able to show poor agent performance, and the LLM agents
are not expected to beat the baselines.

Anything this plan does not settle is decided before the stage it affects
runs, and committed as a dated amendment at the end. An amendment never
changes a seed, a model, a setting, a policy or the agent.

## 1. What is fixed

| | |
|---|---|
| Suite | `dreamteam-dev` version 1 ([suite file](../suite/dreamteam-dev-1.json)), all five entries |
| Scenario fingerprints | financial pressure `3dcd153b…`, reporting backlog `e8718dbd…` (T90 applicable), demand growth `0146ac34…`, staffing shortage `6ef37e92…`, stable `d6511389…`; checked by the runner before anything runs |
| Contract | `dreamteam.benchmark-contract/v1`, the contract report as V0.6b writes it |
| Engine | 0.6.0-dev.3 (this commit) |
| Cadence | daily at 07:00 |
| Baselines | `noop` 1, `greedy` 1, `conservative-v2` 2 (this commit); `rules` 1 and `conservative` 1 stay available for reproducing earlier results and are not comparators here |
| LLM agent | `agents/llm/llm_agent.py`, llm-agent 2, prompt `llm-agent-prompt/v2`, memory `llm-agent-memory/v1` (on), unchanged since V0.5.0 |
| Models | `Qwen3.6-35B-A3B-oQ4e-mtp` (policy `qwen36-35b-a3b`) and `gemma-4-26B-A4B-it-oQ4e-mtp` (policy `gemma4-26b-a4b`), chosen by the owner before any result: both acted in V0.5c, and both are far faster than Gemma 4 31B. A pragmatic choice, not a claim that either is better. |
| Model settings | The V0.5c files, unchanged: [llm-qwen36-35b-a3b.json](../v05c-dev/llm-qwen36-35b-a3b.json), [llm-gemma4-26b-a4b.json](../v05c-dev/llm-gemma4-26b-a4b.json). Identical except for the model (tested in V0.5c): temperature 0, `maxOutputTokens` 4,096, one retry, 300 s per request, 900 s per decision, `json_object`, summary observation, guided catalog, details allowed, memory 6 decisions, 6 metric rows, 6,000 characters. |
| Server | The same local omlx server as V0.5c, reached through `DREAMTEAM_LLM_BASE_URL`, never committed; no key, no prices, so no inference cost is estimated (reported as "none configured") |
| Harness timeout | 1,800 s per decision point for the LLM agents |

**Failures.** A model that cannot load or complete the protocol is
recorded and classified. It is never replaced. An infrastructure failure
(HTTP, connection or timeout) is reported, and the run is rerun once in
full into a new directory; both are kept. A second failure stops the
stage, with the rest reported incomplete.

## 2. Phase 1: conservative-v2 qualification

1. **Synthetic qualification** (done before this plan was committed;
   `internal/bench/conservative2_test.go`): the eight requirements in
   §7.2 of the specification, all passing.
2. **Development check:** [qualify.json](qualify.json):
   * no action and `conservative-v2`;
   * the five entries;
   * seed 41101;
   * 90 days (long enough for a raise to take effect and be reviewed);
   * T90 where it applies;
   * 10 runs.

   **Reported:** for each entry, whether `conservative-v2` acted, every
   change with its date, every review and its outcome, its daily breach
   records as read back from its own requests in the transcript, and its
   paired differences from no action.

   **Nothing is tuned after the run.** If it never acts, that is reported.
   No version 3 is made in this milestone.

Phase 2 runs whatever Phase 1 shows: `conservative-v2` is a comparator
either way.

## 3. Phase 2: the comparison

| Stage | Configuration | Seeds | Days | Policies | Runs | Limit |
|---|---|---|---|---|---|---|
| 30-day | [compare-30.json](compare-30.json) | 41201, 41202, 41203 | 30 | noop, greedy, conservative-v2, qwen36-35b-a3b, gemma4-26b-a4b | 75 (30 LLM) | 6 h |
| 90-day | [compare-90.json](compare-90.json) | 41301 | 90 | the same | 25 (10 LLM) | 8 h |

* **30 days first.** T90 is not requested at 30 days (it evaluates the
  90-day window).
* **The 90-day stage** runs only if the 30-day stage completed with no run
  failing twice and no server outage over an hour. It runs on seed 41301,
  fixed now, not chosen from the 30-day outcomes. T90 is evaluated for the
  reporting-backlog entry.
* **At a limit,** the command is stopped. The completed runs are reported,
  and the suite is reported as partial. No horizon is shortened.
* **Order:** one stage at a time on a Raspberry Pi 4 (linux/arm64):
  qualify, compare-30, compare-90.

## 4. What is reported

**Sources:** the contract report and [analyze.py](analyze.py) (committed
with this plan), for each entry, seed and policy:
* operating profit, net revenue, expenses;
* TAT p90, breaches;
* exams completed, lost orders, overtime;
* actions by category (operations or staffing) and type;
* replay verification.

**Also reported:**
* paired differences against no action, with the flag "better service,
  lower profit";
* for the LLM agents: decision outcomes (actions, no action, fallback),
  first-attempt validity, refusals by `validate` by code, repeated invalid
  proposals, details requests, agent errors and infrastructure errors;
  prompt and completion tokens, requests, retries, latency (mean, maximum,
  total), and estimated cost (none configured).

**The six questions, answered from the outcomes and the agents' recorded
decisions and reasons:**
1. Does an agent act appropriately under financial pressure?
2. Does it reduce the reporting backlog?
3. Does it recognize imaging-capacity limits under demand growth
   (V0.6b: added demand was lost, not imaged)?
4. Does it address the staffing shortage?
5. Does it avoid unnecessary intervention in the stable practice?
6. Are service improvements accompanied by disproportionate financial
   costs?

**Rules for answering them:**
* **No inferred reasoning:** a model's reasoning is never inferred from
  outcomes. Behaviour is described from its recorded actions and the
  reasons it gave.
* **No verdict:** "appropriate" and "disproportionate" are left to the
  owner; the figures are given.

## 5. Replay and integrity

* **Checks** for every stage:
  * `benchmark verify` (with `-t90` where T90 ran), including the suite
    checks (scenario fingerprints, the expansion, the contract report);
  * starting-state equality across the policies of each entry and seed,
    from the contract report;
  * `benchmark reproduce` of the deterministic stages (qualification).
* **LLM replay:** every completed LLM run is replayed from its recorded
  decisions without any model, with the memory and prompt rebuilt and
  checked at every decision point.
  * **One configuration per entry:** demand growth and reporting backlog
    share their starting state, and the agent's replay refuses recordings
    it cannot tell apart.
  * **Matching:** the replay's decisions, sessions, results, memory and
    final digests must equal the live runs'.
  * **Repeatability:** the replay must reproduce on linux/amd64 with a
    byte-identical manifest.
* **CI** replays the recordings on three platforms and compares the
  manifests.
* **Deterministic and not:** the replay is deterministic, live inference
  is not. The model exchanges stay outside the repository.

## Amendments

None.
