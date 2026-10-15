# V0.1 acceptance checklist

Status of each V0.1 acceptance criterion (ROADMAP §2) with its evidence:
an automated test, a command, or a recorded measurement. "Met" means the
evidence passes on the current commit; deviations say why.

## Functional

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | `go run ./cmd/dreamteam` opens the TUI paused at Tuesday 06:00 of the established practice, with a populated schedule and an unread backlog | Met, with a calibration deviation | `TestStartsTuesdayMorning`, `TestKeys` (starts paused), `TestOpeningBriefing`. The schedule is populated; the takeover worklist is usually empty (mean 0.1 studies across nine seeds): the struggling practice's reading is over-provisioned for its volume (owner decision, step 18). A backlog builds during the day. |
| 2 | `space` runs the day: patients arrive, register, are imaged, studies are read and finalized; log, rooms, queues, worklist, cash and profit change | Met | `TestKeys`, `TestPacingDoesNotAffectOutcome`, `TestDashboardAlerts`, `TestWorklistScreen`, `TestFinanceScreen` |
| 3 | Five speeds; `n` steps to the next observable entry; `e` completes the day with the summary; play continues into Wednesday | Met | `TestSpeedsIdentical`, `TestStepVisibleLandsOnNewEntry`, `TestEndOfDayStopsAtOperationalClose`, `TestEndOfDaySummary`, `TestKeys` |
| 4 | Every screen renders at 80×24 and 140×40 without overflow; < 60×16 shows the size message | Met | `TestRenderFitsTerminal`: dashboard, schedule, log, patients, worklist, finance, history, briefing, actions, help and end-of-day summary at 80×24, 140×40, 120×36, 160×48 and 60×16; size message at 40×10 |
| 5 | The three actions work from the TUI with validation messages and change outcomes | Met | `TestActionsScreen`, `TestExtendShiftValidation`, `TestWalkInPolicies`, `TestWorklistPolicyOrdering`, `TestInterventionsChangeOutcomesNotDemand`, `TestExtensionAheadOpensBookings` |
| 5a | Admitted patients are served after scheduled close with costed overtime shown; coverage reschedules are explicit with a reason | Met | `TestWaitingPatientsServedAfterClose`, `TestRequiredOvertimeCost`, `TestCoverageExhaustedReschedules`, `TestClosingStatusLine` |
| 6 | `dreamteam run --days 90 --seed 7 --summary text` prints operational and financial summaries; `--log events.jsonl` writes the observable log | Met | `TestRunLogJSONL` (`-summary` text and json; `-log` JSON lines with consecutive sequence numbers from the takeover; the result is identical with and without `-log`) |
| 7 | `dreamteam replay record.json` reproduces the recorded digest | Met | `TestRunSaveReplayVerify`, `TestSaveLoadRoundTrip`, `TestReplayMiddayTakeoverAcrossPayrollAndWeek` |

## Determinism and correctness

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 8 | Same scenario, seed and actions give identical digests: headless vs TUI, any speed sequence, any frame rate, amd64 vs arm64 | Met | `TestTUIRunReplaysHeadless` (random speeds, frame gaps, pauses, steps, alerts and actions from the actions screen; all five digests equal a headless rebuild and a replay), `TestSpeedsIdentical`, `TestResultIndependentOfChunkingAndSpeed`. `.github/workflows/ci.yml` runs the suite and the golden references on linux/amd64, linux/arm64 and darwin/arm64 and compares the three platforms' results byte for byte; first run (75d217c) green: all four golden runs byte-identical on linux/amd64, linux/arm64 and darwin/arm64, and outcome-identical to the references (Go 1.26.0 there, 1.27.1 on the Pi). |
| 9 | Chunking invariance for ≥ 1,000 random chunkings | Met | `TestChunkingInvariance` (engine, 1,000 chunkings); `TestAcceptanceChunking` (`-tags acceptance`: the real simulation, warm-up through one day of operations, 1,000 random chunkings, log and state digests; passed); `TestOperationsChunkingInvariance`, `TestReplayChunkingInvariance`, `TestMetricsChunkingInvariance`, `TestActionsChunkingAndNoise` |
| 10 | All invariants hold for every timestamp over 50 seeds × 90 days | Met | `TestAcceptanceInvariants` (`-tags acceptance`): 50 seeds × 90 days from the takeover, every invariant after every transaction; passed (53.5 min on four Cortex-A76 cores) |
| 11 | Leakage: observation deny-list reflection test; same-past / different-future observation equality; `StepVisible` | Met | `TestObservationHasNoLatentFields`, `TestOpeningStateHidesTheFuture`, `TestMetricsAndAlertsRevealNoFuture`, `TestIDsRevealNoFutureDemand`, `TestFinanceOutcomesNotLeaked`, `TestResultHasNoHiddenData`, `TestStepVisibleLandsOnNewEntry` |

## Realism and calibration

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 12–15 | Volumes, utilization, backlog, TAT and margin bands of the `moderate` preset | Moved to the better-staffed preset (owner decision, step 18) | The default established practice is struggling and capacity-constrained by decision; its nine-seed characterization is `docs/calibration/step-18-report.md` and `docs/calibration/results/`, guarded by `TestEstablishedCharacterization`. Calibration is closed. |

## Performance (Raspberry Pi 4)

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 16 | Warm-up < 1.5 s; `run --days 90` < 2 s; TUI at 10 fps < 15% of a core at 15 min/s; Max speed responds to `space` within 100 ms | Partly met on the Pi 4; met on a Cortex-A76 | Pi 4 (throttled, loaded): warm-up 1.44–1.49 s best (met only at best), `run -days 90` ≈3.3 s (**not met**: no SHA-256 instructions; keyed draws are hashed), TUI 7.4–8.2% of a core at 15 min/s (met; idle 0.4%), Max-speed worst frame 52–94 ms (met). Cortex-A76 (joofi): warm-up 0.53 s, `run -days 90` 1.1–1.2 s. `docs/MEASUREMENTS.md`. Meeting 2 s on a Pi 4 would need a different keyed-draw hash: an engine-version decision for the owner. |

## Owner additions

| Item | Status | Evidence |
|---|---|---|
| Help screen | Met | `?` from any screen, back with `?` or `esc`; `TestHelpAndLogFilters` |
| Event-log filters | Met | Log screen: `tab` cycles the category, `!` the severity (all · warnings & alerts · alerts only); `TestHelpAndLogFilters` |
| `--log events.jsonl` | Met | `TestRunLogJSONL` |
| TUI-to-headless replay test | Met | `TestTUIRunReplaysHeadless` |
| NO_COLOR | Met | Colors follow the terminal's detected profile; `NO_COLOR` removes every color sequence while severity stays marked (⚠, ‼) — `TestNoColor` |
| amd64 CI job | Added | `.github/workflows/ci.yml` |
