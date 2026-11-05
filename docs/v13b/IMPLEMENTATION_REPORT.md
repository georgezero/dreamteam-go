# V0.13b implementation report: deterministic management alternatives

Status: 0.13.0-dev.2. Done and awaiting owner review. V0.13.0 is not
released.

**The owner's decision:**
* V0.13a was accepted and closed.
* GO for V0.13b, with E1–E3 approved and E4 approved narrowly: `slotsClosed`
  describes only the exact capacity removed from the existing schedule.
  It does not estimate future lost orders, and does not claim to measure
  the full consequences of termination.
* Live inference: NO GO.

**Not done:**
* No live model was contacted, and no agent workflow was added.
* No simulator mechanics, financial accounting, action schemas or scoring
  changed.
* The default staffing preview is unchanged byte for byte.

## 1. What was built

| Part | Where | Specification |
|---|---|---|
| E1 branch builder, E2 branch identity, verification | `internal/session/branch.go` | [BRANCH_IDENTITY.md](BRANCH_IDENTITY.md) |
| E4 slots closed (opt-in, separate request) | `internal/sim/slotsclosed.go`; `Session.PreviewSlotsClosed` | [SLOTS_CLOSED.md](SLOTS_CLOSED.md) |
| Candidate rule `dreamteam.alternative-selection/v1` (pinned) | `internal/alternatives/selection.go`, `text/selection-v1.txt` | [CANDIDATE_SELECTION.md](CANDIDATE_SELECTION.md) |
| E3 comparison `dreamteam.management-alternatives/v1`: agent and evaluation audiences | `internal/alternatives/comparison.go`, `evaluation.go` | [COMPARISON_CONTRACT.md](COMPARISON_CONTRACT.md) |
| Presentation `dreamteam.alternatives-presentation/v1` (pinned) | `internal/alternatives/present.go`, `text/presentation-v1.tmpl` | [PRESENTATION_TEMPLATE.md](PRESENTATION_TEMPLATE.md) |
| Classification of scripted replies | `internal/alternatives/decide.go` | [SCRIPTED_EVALUATION.md](SCRIPTED_EVALUATION.md) |
| Evidence | `docs/benchmark/v13b-dev/` | [SCRIPTED_EVALUATION.md](SCRIPTED_EVALUATION.md) |
| Registration of 0.13.0-dev.1 (batch mode `v0.13_dev1`) | `internal/session/record*.go`, `v0131_test.go`, `testdata/sessions/mini-seed52-v0131.*` | `testdata/sessions/README.md` |

**Everything new only reads.**
* `internal/alternatives` uses the session API alone.
* No existing simulator or session source file was edited, apart from
  the version and the registration.
* `preview.go`, `api.go`, validation, accounting and randomness are
  byte-identical (`docs/audits/v0.13b-baseline.md`).

## 2. Results

**Termination versus keep** (mini, seed 52, 2026-10-13 07:00; terminate
Sam (E04), last day 2026-10-27).

The candidates were `proposal`, `no_action` and `adjusted` (last day
2026-11-10).

The agent audience shows, for the termination:
* the last day and end date;
* 14 days' notice;
* severance of $3,950.00, recognized now and paid 2026-10-30;
* contract pay to the last day, $4,350.00;
* 0 displaced bookings;
* 0 voided rosters or extensions;
* 2,481 scheduled slots no longer staffed (1,095 XR, 1,386 CT), from
  2026-10-28 to 2027-02-10, 0 of them booked now;
* the engine's estimate of $1,975.00 a week no longer owed.

For keeping, it shows:
* the employment stage;
* contracted and rostered minutes;
* the weekly guarantee of $1,975.00, which continues;
* severance of $3,950.00 if terminated later;
* pending rosters and extensions;
* payroll accrued, payables and appointments booked ahead.

Nothing under keep is shown as cost-free, and no saving is computed.

**Projections (evaluation only),** to 2026-12-11:
* Orders received were 2,881 in every branch.
* Orders lost were 407 under the proposal, 125 under no action and 337
  under the adjusted alternative.
* The V0.13a 45-day pair is reproduced exactly: 177 more lost orders.

These are realized simulated outcomes. They are not in the agent
audience, and are labelled as differences under the continuation and
horizon, not as gains, losses or verdicts.

**Randomness:**
* Orders and walk-ins were identical across branches, in total and day
  by day.
* Cancellations, exams and waits differ: draws used only where an exam
  happens, attempt numbers and draw contexts.
* An accepted batch that changes nothing, and a refused batch, leave the
  ledger and every reported metric equal to no action.

**Slots closed:** in nine cases on the established scenario, the report
equalled the slots a committed termination closes, slot for slot. Six of
the cases were run by the reviewer: every employee at a state with
bookings, and one batch of two terminations. A front-desk termination
closes no room slot.

**Scripted evaluation:**
* Ten decision cases cover every result and flag.
* Three decision-sensitivity pairs from real engine states show results
  changing exactly where a scripted rule uses the changed fact.

## 3. Safety tests

| # | Requirement | Test |
|---|---|---|
| 1 | Source state unchanged | `TestBranchTerminateVersusKeep`, `TestSelectionTermination`, `TestAgentTerminationVersusKeep`, `TestSlotsClosedExact` (digests); reviewer W3 (bytes) |
| 2 | Branch identity stable | `TestBranchIdentity` |
| 3 | Action changes identity | `TestBranchIdentity` |
| 4 | Horizon changes identity | `TestBranchIdentity` (continuation and sibling horizon) |
| 5 | Replay exact | `TestBranchRecordSavesAndReplays`, `VerifyBranchSet` in every evaluation |
| 6 | Historical rules refused | `TestBranchHistoricalRules` (Slice 2, continued Slice 2, dev.6 two prefixes down) |
| 7 | Shared outside demand | `TestBranchTerminateVersusKeep` (totals and per day) |
| 8 | Accrual-point differences | `TestBranchAccrualPoint` |
| 9 | Termination versus keep | `TestBranchTerminateVersusKeep`, `TestAgentTerminationVersusKeep` |
| 10 | Missing required field | `TestAgentRefusals` |
| 11 | Invalid preview | `TestAgentRefusals` (a refused proposal shows only validation) |
| 12 | Invalid candidate | `TestSelectionRules` (refused proposal, leave-one-out and adjustment) |
| 13 | Candidate limit enforced | `TestSelectionRules` |
| 14 | Deterministic ordering | `TestSelectionTermination` (repeat, reorder refused) |
| 15 | No-action obligations | `TestAgentTerminationVersusKeep` |
| 16 | Slots-closed exactness | `TestSlotsClosedExact`, `TestSlotsClosedRuleMatchesRosterPreview` |
| 17 | Slots-closed opt-in only | `TestSlotsClosedOptInOnly`, `TestSlotsClosedRefusals` |
| 18 | No default preview change | `TestSlotsClosedOptInOnly`; `preview.go` unchanged |
| 19 | No future information in agent audience | `TestAgentAudienceBoundary` |
| 20 | Evaluation audience includes projections | `TestAgentAudienceBoundary`, `TestEvidenceTermination` |
| 21 | Tampered branch rejected | `TestBranchTamper` |
| 22 | Tampered comparison rejected | `TestAgentRefusals`, `TestAgentAudienceBoundary` |
| 23 | Cross-platform determinism | the evidence files, compared byte for byte by CI on Linux amd64 and arm64 and macOS, and on horo |
| 24 | Historical compatibility | `TestV0131Record*`, every earlier fixture test, golden references |

**Mutation checks.**
* 22 mutations of the safety boundaries were run
  (`~/dt-audit-kept/v013b/mutate.py`, kept outside the repository).
* In the first pass 18 were caught. The four survivors exposed weak
  tests, and those tests were strengthened:
  * the horizon was varied only together with the continuation;
  * a class B fact was caught by the digest, not by the class rule;
  * the compared table had four rows, so the copied-table rule never
    ran;
  * the number check matched substrings, so the template can now hold
    no digit at all.
* All 22 are caught now.

## 4. Independent review

The review (`docs/audits/v0.13b-independent-review.md`, verbatim) found
no BLOCKER.

**Fixed** (`docs/audits/v0.13b-review-response.md`):
* F1: evidence generated before the version bump;
* F2: the agent decoder accepted a relabelled projection;
* F3: the rules check covered one prefix only;
* F5: the citation-check order.

**Awaiting the owner:**
* F4: identical leave-one-out candidates. Collapsing them changes the
  rule.

**Recorded:** the notes F6–F10, including the hire adjustment's
departure from V0.13a (F7). It uses a later start, because the action
has no tier field.

**Found by the gates, after the review:**
* `TestHoldoutContainment` did not list the branch builder's call to the
  embedded holdout guard.
* The call refuses a holdout seed before anything runs, like the T90 and
  benchmark sites, and its error never names a seed.
* It is now a reviewed site.

## 5. Quality gates

See `docs/PROGRESS.md`, V0.13b, for the table of gates, where each ran,
and the results.
