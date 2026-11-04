# V0.12b evidence

**Status:**
* **Stage A** is done: the independent safeguard audit, the F1 repair,
  the focused re-review, and the real-engine displacement and hire
  tests.
* **Stage B: run once, COMPLETE.** The design was approved by the owner.
  The driver was implemented with fidelity evidence, independently
  reviewed and re-reviewed (GO), and remediated, and the protocol locked
  (5b8879f). The owner then authorized the single live run. It completed
  420 of 420 draws with 524 model calls ([RESULTS.md](RESULTS.md)). No
  further run is authorized.

V0.12.0 is unreleased (NO GO until the owner has reviewed the results).

## Stage A

| Item | Where |
|---|---|
| Baseline at 9871e37 | `docs/audits/v0.12b-stage-a-baseline.md` |
| Independent safeguard audit (82cea3f): GO, no blocking finding | `docs/audits/v0.12b-safeguard-audit.md` |
| Real-engine displacement test | [DISPLACEMENT.md](DISPLACEMENT.md), `internal/bench/llm_agent6_displacement_test.go` |
| F1 repair: fail closed on missing or invalid consequence fields (2c73c3b, 0.12.0-dev.2) | `agents/llm/llm_agent6.py`; required fields in `docs/V0.12_CONSEQUENCE_INVENTORY.md` §4 |
| F1 regression tests, 18 cases built from real engine answers | `agents/llm/test_llm_agent6_fields.py` |
| Real-engine hire test (audit F4) | `internal/bench/llm_agent6_hire_test.go` |
| Focused re-review (2c73c3b): F1 CLOSED, no blocking defect, GO | `docs/audits/v0.12b-f1-rereview.md` |
| Maintainer's response, and the owner's decisions | `docs/audits/v0.12b-safeguard-audit-response.md` |

**Identities, unchanged by the repair:**
* workflow `llm-agent-acknowledgment-workflow/v1`:
  `b9cb79828f2657de91de69b1fa30019752ab73611d80fb6854c8f7a6488595a6`;
* contract `dreamteam.action-acknowledgment/v1`:
  `33c116da979a5a1067a4f19163369a1092394ca9bfe9ec4aade2dab8cf9c4c9b`.

The V0.12a scripted evidence regenerates byte-identical.

| File | SHA-256 |
|---|---|
| `agents/llm/llm_agent6.py` (as re-reviewed) | `b0e2914150e9cee18dbc6383f1ddd431a881cd36672353e4b9cbdec86fab98f2` |
| `agents/llm/test_llm_agent6_fields.py` | `e52ab563b3f543b1502a5820d8bf03da26fd15d12bc68899d1bfc07e13c50db6` |
| `internal/bench/llm_agent6_hire_test.go` | `be48a1563214f8d2cf2f3586765556dca73fa0e545a0d3ceb645254a754451d8` |
| `internal/bench/llm_agent6_displacement_test.go` | `4a20c2de7592c9b78c43c12e4fc3bf796b09a6c4c36ab83474f2fcfd8aaaa57a` |

## Stage B (locked at 5b8879f; run once, COMPLETE)

| Document | What |
|---|---|
| [RESULTS.md](RESULTS.md) | The live run's results: execution, acknowledgment classes, preview classes, outcomes, exposure, limitations |
| `live/` | The live run record, as the driver wrote it (committed separately, before the analysis) |
| `decisions.json`, `decision-tables.md`, `inference.json` | The run's analysis, by the pinned driver, replayed without a model; reproduced on arm64 and amd64 and in CI |
| [PROTOCOL.md](PROTOCOL.md) | The locked protocol: question, points, arms, limits, outcomes, statistics, replay, lock |
| [DRIVER.md](DRIVER.md) | The driver's specification: how a draw runs, limits, replay, classification, limitations |
| [PROCEDURE.md](PROCEDURE.md) | Run instructions and the partial-run procedure |
| [fidelity/fidelity.md](fidelity/fidelity.md) | 113 scripted cases on the real engine: the driver against the agents' own runs |
| [fidelity/mock-run/decision-tables.md](fidelity/mock-run/decision-tables.md) | A scripted mock run of the real schedule, PARTIAL by its scripted clock |
| `docs/audits/v0.12b-stage-b-driver-review.md` | The independent review of the driver (c9ba751): GO with conditions |
| `docs/audits/v0.12b-stage-b-driver-rereview.md` | The focused re-review of the remediation (81bb0cf): GO for the lock |
| `docs/audits/v0.12b-stage-b-driver-review-response.md` | The response: D1–D13 and R1–R5 answered before the lock |

**The lock** (`protocol.json` → `lock`, status `LOCKED — NOT RUN`):
* **Files:** the SHA-256 of 48 files the run executes or reads:
  * the driver, its fidelity code and tests;
  * the agents and the fixtures they import;
  * the shared-prefix, feedback, decision, sample and repeat modules, and
    financial evidence v1;
  * the selection, its inputs and the twelve recordings;
  * the replay configurations, the points and the settings files;
  * PROTOCOL.md, DRIVER.md and PROCEDURE.md.
* **Schedule:** pinned by its entries,
  `5f0918131b83cd75b9a2966bd25a3e7e95577b016c34ef9761bcb3fd54569c7e`.
* **Driver:** `d870fd39a69c7e7e4b0d2e42de118a04e2431fe3b85938b336aa6e7bdfa04b6a`.
* **Agents:**
  * A: llm-agent 5, `llm-agent-workflow/v1`
    (`3ee7d66f4f10fd93e05a908725c792b16b76d01dc29095360dae9d67c0c09d0a`);
  * B: llm-agent 6, `llm-agent-acknowledgment-workflow/v1`
    (`b9cb79828f2657de91de69b1fa30019752ab73611d80fb6854c8f7a6488595a6`),
    contract `dreamteam.action-acknowledgment/v1`
    (`33c116da979a5a1067a4f19163369a1092394ca9bfe9ec4aade2dab8cf9c4c9b`).
* **Engine:** 0.12.0-dev.3.
* **Checking it:** `stage_b.py check` verifies every pin against the files
  and exits non-zero if one does not hold. `run` refuses a protocol whose
  lock does not hold, and requires the owner's `--authorization`.

| File | SHA-256 |
|---|---|
| `protocol.json` (`v12b-ack-1`, locked) | `6e8a748885a636c2f02c90f87fb221eccc965a24c1eabb73e27128bece82b783` |
| `points/manifest.json` | `e288fd702d8999b25b84e0b0625ed86d077f6045ace16ddb03bf63fbf1447134` |
| `points/contexts.jsonl` | `759831a3b05629bdac8b08473eae4efd8e8897763285b714e770bda6e2690511` |
| `schedule.json` (420 draws, each with its expected start) | `4c4c555e595ed0cc18116e57a6553f66257eb923832e1f63e059923bb4f04079` |
| `analysis/acknowledgment/stage_b_points.py` | `e360ca1f10cdea269b4bde8031397dfb393a77bc9135a14e5592af80a6a8a8c7` |
| `analysis/acknowledgment/stage_b.py` (the driver) | `d870fd39a69c7e7e4b0d2e42de118a04e2431fe3b85938b336aa6e7bdfa04b6a` |
| `analysis/acknowledgment/stage_b_fidelity.py` | `88d5a432a888ea756025947717328a61ba2b12615cf10491e2fba380fe650e0a` |
| `analysis/acknowledgment/test_stage_b.py` | `7ba5ac48d7ba863f61dbfbf852dae0d7eba6760909f054ffb2ea44cf07f41e13` |
| `fidelity/fidelity.json` | `1ab64e67911a22116856bd37d4004acd0654d96117a6d5de1fb20cdc2cba46ca` |
| `fidelity/mock-run/live/draws.jsonl` | `1f1ea78a40ab1fcaa40c1d8f7d199c6efa0400cf5665abfa185362cfd97111a9` |
| `fidelity/mock-run/decisions.json` | `3ae4a20a8defeec7a78fa1eafa5a9971945b6bfd4982449ce72366f7710d5c94` |

Made with:

    python3 -I -B analysis/acknowledgment/stage_b_points.py points --out docs/benchmark/v12b-dev/points
    python3 -I -B analysis/acknowledgment/stage_b.py schedule --protocol docs/benchmark/v12b-dev/protocol.json --out docs/benchmark/v12b-dev
    python3 -I -B analysis/acknowledgment/stage_b_fidelity.py fidelity --binary ./dt --replays DIR --protocol docs/benchmark/v12b-dev/protocol.json --out docs/benchmark/v12b-dev/fidelity
    python3 -I -B analysis/acknowledgment/stage_b_fidelity.py mockrun --binary ./dt --replays DIR --protocol docs/benchmark/v12b-dev/protocol.json --out docs/benchmark/v12b-dev/fidelity/mock-run

where DIR holds the V0.8c and V0.9c replays (`PROCEDURE.md` §1). CI makes
every one again and compares it on every platform. No model was contacted
for any of these. The live run's analysis is made with

    python3 -I -B analysis/acknowledgment/stage_b.py analyze --binary ./dt --replays DIR --dir docs/benchmark/v12b-dev --protocol docs/benchmark/v12b-dev/protocol.json --out OUT

which replays every draw without a model; CI does so on every platform.
The separate frozen 98-item human evidence review is still PENDING.
