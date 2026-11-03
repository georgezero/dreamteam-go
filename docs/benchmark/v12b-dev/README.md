# V0.12b evidence

**Status:**
* **Stage A** is done: the independent safeguard audit, the F1 repair,
  the focused re-review, and the real-engine displacement and hire
  tests.
* **Stage B:** the owner approved its design, and its driver is
  implemented with fidelity evidence, independently reviewed (GO with
  conditions, no Blocker) and remediated. The lock is next. It is not locked and not run, and **live inference is not
  authorized**.

V0.12.0 is unreleased.

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

## Stage B (design approved; driver implemented; not locked, not run)

| Document | What |
|---|---|
| [PROTOCOL.md](PROTOCOL.md) | The protocol: question, points, arms, limits, outcomes, statistics, replay, lock |
| [DRIVER.md](DRIVER.md) | The driver's specification: how a draw runs, limits, replay, classification, limitations |
| [PROCEDURE.md](PROCEDURE.md) | Run instructions and the partial-run procedure |
| [fidelity/fidelity.md](fidelity/fidelity.md) | 113 scripted cases on the real engine: the driver against the agents' own runs |
| [fidelity/mock-run/decision-tables.md](fidelity/mock-run/decision-tables.md) | A scripted mock run of the real schedule, PARTIAL by its scripted clock |
| `docs/audits/v0.12b-stage-b-driver-review.md` | The independent review of the driver (c9ba751): GO with conditions |
| `docs/audits/v0.12b-stage-b-driver-review-response.md` | The response: D1–D13 answered before the lock |

| File | SHA-256 |
|---|---|
| `protocol.json` (`v12b-ack-1`, final draft) | `70299a411cc4e7d7b3d66d0fcc31f4204b31ff30ed4a4874aabd8c2aea591e22` |
| `points/manifest.json` | `e288fd702d8999b25b84e0b0625ed86d077f6045ace16ddb03bf63fbf1447134` |
| `points/contexts.jsonl` | `759831a3b05629bdac8b08473eae4efd8e8897763285b714e770bda6e2690511` |
| `schedule.json` (420 draws, each with its expected start) | `5067b4cdca2fc888fce5f5380c3666e1fa0c0fa766b506d6dba5625814298cf2` |
| `analysis/acknowledgment/stage_b_points.py` | `e360ca1f10cdea269b4bde8031397dfb393a77bc9135a14e5592af80a6a8a8c7` |
| `analysis/acknowledgment/stage_b.py` (the driver) | `7591b97d6cc0cadeb7c2f8e788ed6fccb86c18301e7c7d334d77b425e2fa56be` |
| `analysis/acknowledgment/stage_b_fidelity.py` | `88d5a432a888ea756025947717328a61ba2b12615cf10491e2fba380fe650e0a` |
| `analysis/acknowledgment/test_stage_b.py` | `3337e65185994e8e8261a7aa678a95d775aae69ba8cccdc7c6ade8968fd3063f` |
| `fidelity/fidelity.json` | `1ab64e67911a22116856bd37d4004acd0654d96117a6d5de1fb20cdc2cba46ca` |
| `fidelity/mock-run/live/draws.jsonl` | `1f1ea78a40ab1fcaa40c1d8f7d199c6efa0400cf5665abfa185362cfd97111a9` |
| `fidelity/mock-run/decisions.json` | `4c7d2639b7e4d867fe5b6f6b176bd494b9225370a33f1a14ec545f669f1a5df2` |

Made with:

    python3 -I -B analysis/acknowledgment/stage_b_points.py points --out docs/benchmark/v12b-dev/points
    python3 -I -B analysis/acknowledgment/stage_b.py schedule --protocol docs/benchmark/v12b-dev/protocol.json --out docs/benchmark/v12b-dev
    python3 -I -B analysis/acknowledgment/stage_b_fidelity.py fidelity --binary ./dt --replays DIR --protocol docs/benchmark/v12b-dev/protocol.json --out docs/benchmark/v12b-dev/fidelity
    python3 -I -B analysis/acknowledgment/stage_b_fidelity.py mockrun --binary ./dt --replays DIR --protocol docs/benchmark/v12b-dev/protocol.json --out docs/benchmark/v12b-dev/fidelity/mock-run

where DIR holds the V0.8c and V0.9c replays (`PROCEDURE.md` §1). CI makes
every one again and compares it on every platform. No model was contacted.
The separate frozen 98-item human evidence review is still PENDING.
