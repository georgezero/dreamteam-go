# V0.13d implementation report: controlled alternatives workflow and experiment preparation

**Status:**
* 0.13.0-dev.2, with the engine version unchanged (LIMITATIONS.md).
* Prepared for owner review. The protocol is a draft, **not locked**;
  live inference is **not authorized**; V0.13.0 is not released.

**The owner's decisions** (2026-11-05): V0.13c accepted as a
protocol-design milestone; GO for V0.13d preparation; D7 neutral
identifiers and a typed presentation; D8 symmetric wording; D9 the
estimand; D10 operations batches preserved; the 964-draw budget not
approved; protocol lock NO GO; live inference NO GO.

**Not done:** no model was contacted; no simulator mechanics, accounting,
scoring, validation, preview or action schema changed; no earlier agent
changed.

## 1. What was built

| Part | Where | Document |
|---|---|---|
| The typed view: neutral references, closed tables, fail closed | `internal/alternatives/view.go` | [INFORMATION_BOUNDARY.md](INFORMATION_BOUNDARY.md) |
| Presentation v2: the shared head and foot, the comparison, the detail and the fixed answers | `internal/alternatives/text/presentation-v2.tmpl`, `turn.go` | [PRESENTATION_V2.md](PRESENTATION_V2.md) |
| Both arms' turns, the reply reader and the outcome with operations batches | `internal/alternatives/turn.go` | [WORKFLOW.md](WORKFLOW.md) |
| The protocol operations `alternatives.turn` and `alternatives.decide`, opt-in | `internal/alternatives/ops.go`, `internal/agent/agent.go` (`Extensions`), `cmd/dreamteam/agent.go` (`-alternatives`) | `docs/AGENT_PROTOCOL.md` §3.3 |
| The workflow `llm-agent-alternatives-workflow/v1` | `agents/llm/llm_agent_alternatives.py` | [WORKFLOW.md](WORKFLOW.md) |
| The driver `dreamteam-v13d-driver/1` | `analysis/alternatives/v13d.py` | [DRIVER.md](DRIVER.md) |
| Scripted fidelity | `analysis/alternatives/v13d_fidelity.py`, `docs/benchmark/v13d-dev/fidelity/` | [FIDELITY.md](FIDELITY.md) |
| Evidence: contexts, draft protocol and schedule, rendered turns, the view table, the pair ledgers | `docs/benchmark/v13d-dev/` | [SENSITIVITY_CASES.md](SENSITIVITY_CASES.md), [PROTOCOL_DRAFT.md](PROTOCOL_DRAFT.md) |
| The human review package | | [PRESENTATION_REVIEW.md](PRESENTATION_REVIEW.md) (PENDING) |
| CI: one step rebuilds and checks everything above without a model | `.github/workflows/ci.yml` | |

## 2. Results

* **D7.** No source string reaches a turn. Hostile employee names and
  request IDs are in the session but in no text the engine makes; twelve
  edits to a bound comparison each make the view fail closed.
* **D8.** At every due member, the arms' heads are equal, their feet are
  equal byte for byte, and the treatment is the control with the
  comparison inserted.
* **D9.** Every outcome has a final action on one scale for both arms
  (`original_retained`, `listed_alternative`, `no_staffing_action`,
  `other_batch`, `abandoned`, `invalid`, `no_decision`), made from the
  submitted batches only; the arm's own result labels are process
  measures.
* **D10.** Select keeps the operations batch first and unchanged
  (`no_action` included); a revision replaces it; abandon submits nothing;
  every batch is validated alone with the protocol's own batch validation;
  execution records partial application.
* **The due test** is V0.13c's at every member and every historical point.
* **Fidelity:** 65 scripted cases, 0 failed checks, with the
  historical contexts: every required case and the review's cases, each
  replayed without a model.
* **The draft protocol:** 262 draws, at most 802 model calls, 4 hours, made
  from the committed inputs alone.

## 3. Reviews

* **Technical review:** `docs/audits/v0.13d-independent-review.md`
  (verbatim, `e40e6c0`): do not lock yet; 9 MAJOR, 3 MINOR, 2 NOTE. Every
  finding is fixed; F7 (the final-action scale), F3 (`compared`) and N1
  (the identical foot) are listed for the owner to confirm. Response:
  `docs/audits/v0.13d-review-response.md`.
* **Mutation checks:** 23 on the fixed tree; 21 caught, 2 equivalent.
* **Human presentation review: PENDING.**

## 4. Quality gates

See `docs/PROGRESS.md`, V0.13d.
