# V0.13c implementation report: alternative comparison protocol preparation

**Status:**
* 0.13.0-dev.2, with the engine unchanged.
* Prepared for owner review. The protocol is **not locked** and live
  inference is **not authorized**. V0.13.0 is not released.

**The owner's decisions:**
* V0.13b accepted and closed;
* GO for V0.13c preparation only;
* the F4 duplicate-candidate repair approved;
* live inference NO GO.

**Not done:**
* No model was contacted.
* No simulator mechanics, accounting, scoring or action schema changed.
* No live agent workflow was written.

## 1. What was built

| Part | Where | Document |
|---|---|---|
| F4: rule `dreamteam.alternative-selection/v2` (removes later duplicate batches; v1 unchanged) | `internal/alternatives/selection.go`, `text/selection-v2.txt` | [F4_REMEDIATION.md](F4_REMEDIATION.md), [SELECTION_RULE_V2.md](SELECTION_RULE_V2.md) |
| Both arms' final-decision turns, the information boundary in code, control-arm classification | `internal/alternatives/arms.go`, `text/control-v1.tmpl` | [EXPERIMENTAL_DESIGN.md](EXPERIMENTAL_DESIGN.md) |
| The branch-point inventory and `dreamteam-v13c-select/1` (the V0.12b pool on the engine, from the V0.8c and V0.9c replays; rebuilt in CI) | `internal/alternatives/inventory_test.go`, `docs/benchmark/v13c-dev/points/` | [BRANCH_POINTS.md](BRANCH_POINTS.md) |
| Six sensitivity pairs of real engine states, every visible difference listed, every turn rendered | `internal/alternatives/v13c_evidence_test.go`, `docs/benchmark/v13c-dev/sensitivity-pairs.json`, `turns/` | [SENSITIVITY_CASES.md](SENSITIVITY_CASES.md) |
| The human presentation review package (review PENDING) | | [PRESENTATION_REVIEW.md](PRESENTATION_REVIEW.md) |
| The draft prospective protocol | `docs/benchmark/v13c-dev/protocol-draft.json` | [PROTOCOL_DRAFT.md](PROTOCOL_DRAFT.md) |
| Limitations | | [LIMITATIONS.md](LIMITATIONS.md) |
| Baseline | | `docs/audits/v0.13c-baseline.md` |

**Scope of the code:**
* Outside `internal/alternatives`, only CI changed: one step rebuilds
  the inventory from the replays.
* `preview.go`, `api.go`, validation, accounting, randomness, the
  session package, the agent protocol, the V0.13b presentation template
  and selection rule v1's text are byte-identical
  (`docs/audits/v0.13c-baseline.md`).

## 2. Results

**F4:**
* The duplicate-termination proposal lists proposal, no_action and
  without_1 under v2 (v1: four, two identical).
* The refused proposal stays refused with its codes, nothing is
  substituted, and the list branches and verifies.
* On the 20 historical points, v2 removed nothing.

**Arms:**
* The turn is due by one test for both arms.
* Treatment's text is the bound agent comparison's presentation, and
  nothing else.
* Every text passes the boundary check.

**Branch points:**
* Nine points are selected.
* Five are due at the recorded proposal (1 Gemma, 4 Qwen), two are
  refused points that qualify after a correction, and two are A/A
  controls.
* `termination_keep` and `valid_expensive_not_due` are unavailable.

**Sensitivity pairs:**
* There are six: displacement across a date and within a state,
  severance, an alternative available or not, the guaranteed
  commitment, and capacity loss.
* None isolates one fact. S5 comes closest, with 6 value differences.
  All are listed.
* Each pair's fact is detected by the scripted rule that reads it.

## 3. Reviews

* **Technical review:** `docs/audits/v0.13c-independent-review.md`
  (verbatim), with the response in `docs/audits/v0.13c-review-response.md`.
* **Human presentation review: PENDING.**

## 4. Quality gates

See `docs/PROGRESS.md`, V0.13c.
