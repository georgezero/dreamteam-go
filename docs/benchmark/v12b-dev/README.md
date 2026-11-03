# V0.12b evidence

**Status:**
* **Stage A** is done: the independent safeguard audit, the F1 repair,
  the focused re-review, and the real-engine displacement and hire
  tests.
* **Stage B** is proposed. It is not frozen and not run, and **live
  inference is not authorized**.

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

## Stage B (proposed)

The protocol is [PROTOCOL.md](PROTOCOL.md). It needs three things before
any request: the owner's approval, then the driver and its fidelity tests
(PROTOCOL.md §8), then a freezing commit.

| File | SHA-256 |
|---|---|
| `protocol.json` (`v12b-ack-1`, draft) | `b53d469c8ac7e68a9c612691f98c8b73c04d1721e8ad69b8610b74071dd57278` |
| `points/manifest.json` | `e288fd702d8999b25b84e0b0625ed86d077f6045ace16ddb03bf63fbf1447134` |
| `points/contexts.jsonl` | `759831a3b05629bdac8b08473eae4efd8e8897763285b714e770bda6e2690511` |
| `schedule.json` (420 draws) | `3714d792b2a823e58816ab4504c72bec1f1d3e6f641186af80f9723129989e2c` |
| `analysis/acknowledgment/stage_b_points.py` | `e360ca1f10cdea269b4bde8031397dfb393a77bc9135a14e5592af80a6a8a8c7` |

Made with:

    python3 -I -B analysis/acknowledgment/stage_b_points.py points --out docs/benchmark/v12b-dev/points
    python3 -I -B analysis/acknowledgment/stage_b_points.py schedule --protocol docs/benchmark/v12b-dev/protocol.json --out docs/benchmark/v12b-dev

CI checks both on every platform. No model was contacted to prepare
them. The separate frozen 98-item human evidence review is still PENDING.
