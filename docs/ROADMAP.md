# DreamTeam — Roadmap

Status: **draft for review**.
Companion documents: [PLAN.md](PLAN.md), [SIMULATION_DESIGN.md](SIMULATION_DESIGN.md),
[TUI_DESIGN.md](TUI_DESIGN.md).

Each version is a playable build. Nothing ships as "framework only".

---

## 1. Milestones at a glance

| Version | Theme | Depends on |
|---|---|---|
| **V0.1** (released, `v0.1.0`) | Engine + playable TUI: a typical Tuesday at an established practice | — |
| **V0.2** (released, `v0.2.0`) | Practice design: sustainable-scenario gate, five staffing/reading actions, contracts and lifecycle, T90 evaluation | V0.1; Step 0 gate |
| **V0.3** (released, `v0.3.0`: the agent interface, by the owner's release contract; the operations-control items below remain planned) | Operations control: scheduling, disruptions, QA/prep, save/load, agent-interface skeleton | V0.2 |
| **V0.4** (released, `v0.4.0`: the agent evaluation harness, V0.4a and V0.4b, by the owner's release decision 2026-10-26) | Agent evaluation harness: policies, built in or external processes over stdio, compared on identical scenarios and seeds through the agent protocol with the seed withheld; decisions and outcomes recorded, replay verified, reruns reproduced ([BENCHMARK.md](BENCHMARK.md)). The financial-management scope formerly planned here (§5) awaits a new slot | V0.3 agent interface |
| **V0.5** (released, `v0.5.0`: LLM agents, V0.5a–V0.5c, by the owner's release decision 2026-10-28) | LLM agents: a model-agnostic agent that manages a practice through the existing policy protocol as a separate process, calling an OpenAI-compatible API; bounded memory within a run, action guidance and targeted prompts; model evidence, usage and estimated cost recorded; recorded decisions replayable ([LLM_AGENT.md](LLM_AGENT.md), §5a) | V0.4 harness |
| **V0.6** (released, `v0.6.0`: the agent benchmark, V0.6a–V0.6c, by the owner's release decision 2026-10-29) | A reproducible benchmark for practice-management agents: a five-scenario suite with versioned fingerprints, one-command suite runs, a six-dimensional contract report paired against no action, deterministic baselines, external LLM agents, recorded-decision replay ([BENCHMARK_SUITE.md](BENCHMARK_SUITE.md), §7) | V0.5 agents |
| **V0.7** (in development: V0.7a, radiologist overtime costing, and V0.7b, financial solvency, accepted; V0.7c, the frozen-agent evaluation, 0.7.0-dev.3; owner decision 2026-10-29) | Simulator realism: the financial consequences of management actions, then re-characterization and frozen-agent evaluation (§7a) | V0.6 benchmark |
| **Future** (deferred) | Growth and expansion (§6, formerly V0.5); financial management (§5, formerly V0.4) | — |

**Owner decision 2026-10-27:** V0.5 is re-slotted from growth to LLM agents,
and V0.6 is the full agent benchmark. V0.5 builds one LLM agent that manages
a practice through the existing interface; V0.6 evaluates agents across
models, scenarios, seeds and baselines. Growth is kept as a deferred
milestone, not removed.

**V0.2–V0.6: proposed in V0.2 design draft 2, pending the Step 0 gate** (first
sweep 2026-10-14 and Step 0B 2026-10-15 did not pass; Step 0C 2026-10-15, a
revised staffing architecture, passes conditionally at ×1.30 demand with a
≈2.8% margin: CONDITIONAL GO, approved by the owner 2026-10-15, design §18).
V0.2 is being implemented in slices; Slice 1, the employee contract and
capacity foundation with no new action, is built (design §5.5); Slice 2,
`hire` and `terminate` with practice-caused displacement and session v2, is
built (design §5.7); Slice 3, `set_roster` with dated roster versions, is
built (design §5.8). The owner's dynamic staffing benchmark (design §5.9,
`docs/calibration/dynamic-staffing/`) found staffing reorganizations paying
back within 180 days on 20/20 held-out seeds, but the full one breaks report
turnaround without reading actions: CONDITIONAL GO. Slice 4,
`set_reading_allocation`, is built (design §5.10). Rerunning the frozen
plans with the full reorganization plus the lead's 62.5/52.5% allocation
(plan D) returned reporting within the provisional gates on average, at
about the allocation's own salary cost: CONDITIONAL GO. 0.2.0-dev.6 fixed
the audit's touching-block and overrun-debt scheduler defects and added a
14-day minimum allocation commitment (design §5.11), with every frozen
result reproduced. Slice 5A, the remote reading engine (0.2.0-dev.7, design
§5.12), is built: tagged reader ownership, two persistent teleradiology
lanes, half-open coverage windows and a common finalization path, with
no player action yet. Slice 5B, `contract_teleradiology` (0.2.0-dev.8, design
§5.13), is built: the `standard-v1` catalog, start and cancel with notice,
minimum term and successor rules, daily prorated retainers, read fees locked
at claim, first-of-month vendor settlement, previews, the TUI and replay of
earlier records. Slice 5C.0 (0.2.0-dev.9, design §13.4) froze the
evaluation before any strategy is compared: per-seed service gates with
evidence minimums, fixed acquisition cohorts, a rota-aligned backlog rule,
transaction-level solvency, censored payback, case-selection guards, a
40-seed sealed holdout and explicit verdict rules (`docs/evaluation`).
After an independent audit, Slice 5C.0a (0.2.0-dev.10, design §13.5)
corrected it as policy v0.2-eval-2: joint seed reliability, an urgent
evidence window chosen by operating characteristics, common paired
horizons, access followed to resolution, backlog aging safeguards, exact
case selection, absolute profit for GO and service-purchase
classification. Slice 5C.0b (0.2.0-dev.11, design §13.6) built the
canonical holdout runner: holdout seeds refused by every other path, an
artifact contract pinning strategies, policy, engine and the owner's
approval by digest, the complete seed and strategy matrix, and a
one-time run that may afterwards only be reproduced byte for byte. Slice
5C.1 (0.2.0-dev.12, design §13.7) compared a prespecified strategy family
and four documented refinements on the development seeds
(`docs/calibration/strategy-development`).
* **No strategy passes v0.2-eval-2.** The funding gate (with its 14-day
  employee commitment) and the backlog rule (a reference block in the
  December demand low) fail on every seed for every plan, no action
  included.
* **Three plans pass every other hard gate on all 10 seeds:** D's
  staffing at a 65/55% reading allocation (D2), alone or with an
  urgent-only weeknight teleradiology contract (E3, E2). Each is a
  service purchase at T90.
* **All-priority teleradiology** restores service but is insolvent.

Slice 5C.2 (0.2.0-dev.13, design §13.8) froze policy v0.2-eval-3:
* funding as a dated settlement test;
* workload-adjusted backlog checks;
* cluster-aware service reporting;
* a CONDITIONAL GO class for unrepaid transitions.

Re-evaluated on the development seeds, funding no longer fails any plan,
and D2, E2 and E3 failed only the backlog gate.

Slice 5C.3 (0.2.0-dev.14, design §13.9) froze policy v0.2-eval-4:
* the peak is informational;
* the workload-adjusted mean has a calibrated tolerance;
* late work is judged on block medians;
* a stale cap holds in every block from the reference on;
* everything was selected by a prespecified protocol on synthetic
  queues.

On the development seeds D2 is CONDITIONAL GO (transition, not yet
repaid), and E2 and E3 are CONDITIONAL GO (service purchase). Next: one
independent review of the whole corrected framework, then the frozen
strategy set and the unseal record.

That review (2026-10-18, at 7707dbd) gave NO GO for the confirmatory
run: severe reporting delays and delayed access could pass every
clinical gate, and the holdout safeguards relied on cooperation.
Remediation is in three slices:

* **5C.4a** (0.2.0-dev.15, design §13.10): the audit is preserved, the
  ordinary tests' holdout exposure is contained and recorded, and the
  clinical acceptance rules for v0.2-eval-5 are specified, not
  activated.
* **5C.4b** (0.2.0-dev.16, design §13.11): the v0.2-eval-5 clinical
  evaluator. Its acceptance tests are in place, v0.2-eval-4 is pinned
  neutral, and the combined synthetic protocol is fixed. The
  exploratory run informs the owner's values, and a confirmatory run
  validates them before v0.2-eval-5 is frozen.
* **5C.4c:** holdout integrity, meaning refused-action semantics,
  executable identity, authenticated approval and durable run-once.

D2 and E3 are not rerun until v0.2-eval-5 is validated.
See [V0.2_DESIGN.md](V0.2_DESIGN.md) for the binding owner decisions and detailed
contracts. V0.1 scope and acceptance history below remain unchanged.

The staffing-first re-scope moves initial hiring/termination and the roster
editor into V0.2. Exactly five new actions are included: `hire`, `terminate`,
`set_roster`, `set_reading_allocation`, `contract_teleradiology`. Operations
control and the minimal observation/action transport move to V0.3, finance to
V0.4, and growth to V0.5. V0.6 retains both the former WebUI/benchmark scope and
the managed-practice scope, staged framework first and harness second. No
milestone is silently dropped.

```
V0.1 ──▶ V0.2 ──▶ V0.3 ──▶ V0.4 ──▶ V0.5 ──▶ V0.6
         Step 0 gate                         framework → harness
```

---

## 2. V0.1 — A typical Tuesday

### Scope

* Deterministic discrete-event engine with keyed RNG, integer time and money.
* Established-practice scenario (`moderate` preset) with 180-day warm-up, start
  **Tuesday 06:00**; the engine continues through subsequent days with
  Mon–Sat hours (multi-day play works; balance is tuned for weekdays).
* 4 rooms, 8 employees, default roster with shared techs and coverage gaps.
* Demand → booking (centralized, automatic) → cancellations → arrivals
  (early/late/no-show) → walk-ins → registration → imaging → reading → report
  final → claim → payment.
* Simplified finance: revenue recognized at report final, claims with lag and
  outcome, payroll and fixed/variable costs accrued, cash on payments.
* Policies: walk-in, worklist, late-arrival, imaging-queue (defaults).
* Three player actions: `extend_shift`, `set_walkin_policy`,
  `set_worklist_policy`.
* Scenario-scripted downtime (exercise path; off in the default Tuesday).
* Observable log, alerts, metrics, end-of-day summary.
* TUI: dashboard, schedule, patients (+journey), staff, worklist, finance,
  event log, actions, help, summary. Time controls and auto-pause.
* CLI: `play`, `run`, `replay`, `scenario validate|list|show`, `version`.
* Run records (action log + digest) and replay.

### Implementation sequence — walking skeleton

Each step ends with green tests and is a reviewable commit. The TUI appears at
**step 4** and every later step adds something visible. Scope, acceptance
criteria and determinism tests are the same as before; only the order changed.

| # | Step | Visible in the TUI after this step | Tests that close it |
|---|---|---|---|
| 1 | Module, `simtime` (time, calendar, formatting) | — | calendar arithmetic, formatting |
| 2 | `rng`: SHA-256-keyed ChaCha8 streams, uniform, Bernoulli ppm, weighted choice, quantile tables, integer-exp Poisson tables | — | byte-exact key derivation vectors; distribution moments within tolerance; fixed golden sequences |
| 3 | Event queue, timestamp transactions (classes, microsteps, no backward insertion), lazy cancellation with `Gen`, `RunUntil`; **synthetic-load benchmark** (≈ 270k events, the size of a 180-day warm-up) | — | ordering, ties, stale-event noninterference, **chunking invariance** property test; benchmark recorded on the Pi as the first warm-up performance evidence |
| 4 | **Minimal `api` + `session` + TUI shell**: Advance/StepVisible/Observe, a scripted demo source emitting observable log entries; TUI with top/bottom bars, clock, run/pause, five speeds, next-event, event log | clock runs, pauses, steps; log scrolls | import-boundary test (TUI may not import `sim`); pacing digest test (fake wall clock, varied fps/speeds ⇒ identical digest); key-handling update tests. **No golden screen snapshots yet** |
| 5 | Scenario types (minimal subset first), embedded **mini scenario** (one Tuesday, no warm-up, hand-built opening state), `extends`/preset merge, validation, coverage report; rooms (incl. `Unstaffed`), employees, fortnight roster → daily shifts, breaks, cover windows, reading blocks, open/close | **staff panel and room states** (closed / unstaffed / idle) change through the day | valid/invalid fixtures; coverage timelines for Mon/Tue/Sat of both rota weeks |
| 6 | Demand generation (origin-keyed, IDs at disclosure), orders, `FirstFit` booking against staffed templates, recall list, cancellations, lost demand | **schedule view** (timelines + appointment list) | booking feasibility, no double-booking, lost-demand accounting, ID noninterference |
| 7 | Arrivals, missed → no-show lifecycle, late policy, walk-ins, patience | **patients arriving and waiting**, no-shows in the log | conservation invariants, late-arrival paths, arrival distribution |
| 8 | Registration (desk / ops / self-reg), imaging dispatch, eligibility wakes, exam timing, start/admission rules, contrast supervision; **closing rules (SIMULATION_DESIGN §6.9)**: admitted patients served after scheduled close, required overtime (hold-over) within the shift cap, recorded overtime time and cost, explicit rescheduling when safe coverage runs out, operational close | **rooms scanning, queues**; patient journey; **`CLOSING — N patients remaining · OT +42m after 20:00`** status | resource invariants, shift-cap guarantee, wake-up independence; waiting patients kept after scheduled close; exams after close; required overtime and cost; shared XR/CT contention; coverage exhaustion → explicit reschedule; contrast after hours; operational close after scheduled close; determinism across speeds and chunking |
| 9 | Worklist, reading blocks with overrun, worklist policies, TAT | **worklist and backlog** panel | backlog behavior, policy ordering; radiologist distinctions test (below) |
| 10 | Ledger, revenue at expected net, claims outcome model and postings, payments, payroll, fixed/variable costs | **cash / revenue / profit** panel and finance view | ledger balance, cash identity, A/R identity, collection-variance mean ≈ 0 |
| 11 | Warm-up build (full operations + finance), opening-cash adjustment, history boundary; default scenario switches from the mini scenario to the **established practice** | **the real populated Tuesday** | deterministic build; start-state digest golden; warm-up convergence test; **build-time benchmark against the < 1.5 s target** (if missed: profile, then cache the built start state per scenario+seed) |
| 12 | Log entries with running log hash, alerts, metric accumulators (interval utilization, censored TAT), operational day end, end-of-day summary | **alerts and end-of-day summary** | metric definitions on hand-built micro-scenarios |
| 13 | Full projection, action batches + validation; the three V0.1 actions | **actions menu** with validation results | leakage tests, action validation table tests, batch-boundary replay |
| 14 | Snapshots (test-only), run records, replay, digest | — | replay equality, snapshot round-trip, StepVisible never stops on hidden events |
| 15 | CLI `run`, `replay`, `scenario` | — | CLI golden output |
| 16 | TUI polish: responsive layouts (all breakpoints), help, filters, theme / `NO_COLOR` | all screens at every size | **golden screen snapshots at 80×24, 120×36, 160×48** (layout now stable). **As done (dev.18.3):** help `?`, log filters, NO_COLOR, `TestScreenSnapshots` |
| 17 | End-to-end: TUI action → run record → headless replay | — | identical digests TUI vs headless. **As done (dev.18.3):** `TestTUIRunReplaysHeadless`; CI on amd64/arm64/macOS |
| 18 | Calibration pass for `moderate` (per weekday type); preset bands; `docs/ASSUMPTIONS.md`; **P&L calibration workbook** (§10) | — | calibration suite (build tag); engine T30 margin within ±2 pp of the workbook for the same inputs. **As done:** the default became a struggling, capacity-constrained practice (owner decision); nine-seed report and calibration suite (`-tags calibration`); the `moderate` better-staffed preset, the workbook and `ASSUMPTIONS.md` moved to the next scenario milestone. **18.1:** review remediation (extension overflow, booking feasibility, walk-in cohorts), reproducible experiment manifests and results, urgent-turnaround analysis. **18.2:** urgency takes precedence under every worklist policy (gameplay rule); calibration phase closed |
| 19 | Performance pass on the Pi (build, 90-day run, TUI CPU, Max-speed responsiveness) | — | acceptance criterion 16. **As done (dev.20):** measured and profiled; idle TUI 3.0% → 0.4%; Pi 4 misses `run -days 90` < 2 s (≈3.3 s, software SHA-256), a Cortex-A76 meets all targets (`docs/MEASUREMENTS.md`) |
| 20 | README, docs sync, change notes | — | — **As done (dev.20):** `README.md`, `docs/ACCEPTANCE.md` |

Notes:

* **The `api` grows with the screens.** Observation types start minimal at step
  4 and gain fields only when a screen needs them. Expect churn until step 16.
  That's why golden screen snapshots wait until then, while behaviour tests
  (keys, pacing, determinism) run from step 4.
* **The mini scenario** (step 5) is a hand-built Tuesday: today's schedule,
  a small unread backlog and an opening cash figure, with no warm-up and no
  A/R history. It stays in the repo as a fast test fixture after step 11.
* **The full-operations warm-up (step 11)** comes after everything it
  exercises. Steps 6–10 test against the mini scenario and short from-epoch
  runs.
* **Radiologist distinctions (step 9 test)** must stay separate in code and
  tests:

  | Concept | Meaning | Drives |
  |---|---|---|
  | Paid allocation | share of salary charged to this site | payroll cost |
  | Site-reading blocks | when they read this site's worklist | reading capacity |
  | Location (on site / remote) | per shift | contrast supervision only |
  | On duty | within their shift | supervision eligibility (with on site) |

  Test case: a remote radiologist in a site-reading block can read but can't
  supervise contrast, and an on-site radiologist in an other-duty block can
  supervise but doesn't claim reads.
* **Calibration caveat.** Matching the P&L workbook (step 18) is a
  *consistency* check between two implementations of the same assumptions,
  not evidence of real-world validity. External validity needs independent
  operational data (see Risks and §10).

### V0.1 acceptance criteria

Functional

1. `go run ./cmd/dreamteam` opens the TUI, paused, at Tuesday 06:00 of the
   established practice, with a populated schedule and an unread backlog.
   *Accepted V0.1 deviation (owner, 2026-10-14):* the default established
   practice is the struggling, imaging-constrained one; its takeover
   worklist is naturally small or empty (reading has spare capacity at its
   volume). No backlog is manufactured; the 5–20-study takeover target is
   deferred to a future reporting-constrained scenario.
2. Pressing `space` runs the day; patients arrive, register, are imaged,
   studies are read and finalized; the log, rooms, queues, worklist, cash and
   profit visibly change.
3. All five speeds work; `n` advances to the next observable log entry; `e`
   completes the day and shows the end-of-day summary; play continues into
   Wednesday.
4. Every screen in TUI_DESIGN §4 (except mouse) renders at 80×24 and 140×40
   without overflow; < 60×16 shows the size message.
5. The three actions work from the TUI with validation messages, and change
   outcomes (e.g., approving overtime for the US tech reduces US waits).
5a. Patients admitted before scheduled close are served after it, with
   overtime shown (`CLOSING — N patients remaining · OT +Xm`) and costed.
   No one is sent home because of the clock. Coverage reschedules are
   explicit and give a reason.
6. `dreamteam run --days 90 --seed 7 --summary text` completes and prints
   operational and financial summaries; `--log events.jsonl` writes the
   observable log.
7. `dreamteam replay record.json` reproduces the recorded digest.

Determinism and correctness

8. Same scenario + seed + action log ⇒ identical digest across: headless vs TUI
   play, any speed sequence, any frame rate, and `amd64` vs `arm64` (CI matrix).
9. Chunking invariance property test passes for ≥ 1,000 random chunkings.
10. All invariants (SIMULATION_DESIGN §10) hold for every timestamp over
    50 seeds × 90 days.
11. Leakage tests pass: observation deny-list reflection test; same-past /
    different-future observation equality test; `StepVisible` test.

Realism / calibration (default `moderate` preset, 20 seeds, **Tue–Fri** of days
1–30 unless stated; targets per weekday type in SIMULATION_DESIGN §5.1)

*Step 18:* criteria 12–15 describe the better-staffed `moderate` preset
(next scenario milestone). The default established practice is the
struggling one; its measured characterization (nine seeds) is in
`docs/calibration/step-18-report.md` and guarded by
`TestEstablishedCharacterization`.

12. Tue–Fri visits 65–85; walk-ins 6–12; no-show 4–9%; same-day cancellations
    visible. Monday and Saturday volumes within their own bands.
13. US staffed-hours utilization 80–90%, with the longest lead time and most lost
    demand of any modality. MR open-hours utilization ≤ 65% (cap 66.7%: dark
    after 16:30). Radiologist site-reading capacity 105–115% of same-day reading
    demand.
14. Unread backlog at 07:30 Tue–Fri: 5–20 studies (higher on Tuesday after the
    weekend); routine TAT median 1–4 h.
15. T30 operating margin within the `moderate` band (6–12%); cash and profit
    diverge on at least one payroll day.

Performance (Raspberry Pi dev machine)

16. Scenario build (180-day warm-up) < 1.5 s; `run --days 90` < 2 s; TUI at 10 fps uses < 15% of one
    core while running at 15 min/s; Max speed stays responsive to `space` within
    100 ms.
    *Accepted V0.1 deviation (owner, 2026-10-14):* on the Raspberry Pi 4 the
    headless `run --days 90` takes ≈3.3 s (no SHA-256 instructions; every
    keyed draw is hashed); the 2 s target is waived for V0.1 on that machine.
    The SHA-256 keyed RNG is kept: an alternative is a separately versioned
    engine experiment. A Cortex-A76 meets every target.

### Deferred out of V0.1 (explicit)

Save/load UI · stochastic equipment failures · staff absences · front-desk phone
and scheduling workload · template editing · rescheduling actions · overbooking
& reserved add-on slots · prep failures, incomplete exams, repeats · protocoling ·
multi-exam visits · claim rework, denial management · payroll cycle UI · hiring,
equipment purchase, extended hours · referrals dynamics, reputation, competition ·
New Practice scenario · mouse · WebUI · SQLite · agents.

---

## 3. V0.2 — Practice design

**Proposed in V0.2 design draft 2, pending the Step 0 gate.**

Deliverables

* **Step 0, before action code:** static scenario-overlay feasibility sweep of
  documented demand, technologist coverage, US packing and reading-allocation
  levers. At least nine paired seeds at T30/T90; promising candidates expanded
  to twenty. Fixed reimbursement, payer/patient behavior, exam/read durations
  and expense rates. Resolve the 16:00–20:00 shift-validator limitation before
  claiming part-time results. Stop if the allowed levers cannot sustain a
  profitable practice with measured service and funding constraints.
* Separate candidate `sustainable` scenario with measured achievable baseline
  and useful remaining management decisions. Keep the struggling `established`
  default and historical V0.1 fixtures. No forced 6–12% margin or inherited
  MR-dark-after-16:30/backlog targets for the new scenario.
* Five actions: `hire`, `terminate`, `set_roster`, `set_reading_allocation`,
  `contract_teleradiology` (including contract cancellation). Atomic V0.2
  batches; deterministic effective dates, previews, errors and no-ops.
* Employee registry separate from scenario configuration; stable IDs; recruiting,
  paid onboarding, active service, notice and termination. Mandatory hourly
  contracts establish guaranteed weekly compensation and notice/severance;
  rostered coverage and actual worked hours are distinct. No contract amendment
  action. Weekly overtime premium applies only above the actual-hours threshold
  (40 hours, ×1.5 in the supplied scenario).
* `set_roster` only for hourly technologists/front desk. Whole-schedule booking
  feasibility and explicit practice-caused displacement/reassignment/rescheduling
  with retained patient history and financial consequences.
* Employed radiologist allocation changes salary charge and reading blocks,
  retaining fixed shifts and contrast presence. Teleradiology has tagged remote
  readers, persistent lanes, coverage windows, locked retainers/per-read fees,
  explicit settlement and service implications. No artificial outsourcing price
  floor; compare total salary/vendor costs and service.
* TUI staff/reading controls and coverage/obligation previews separating
  guaranteed commitments from forecast incremental costs. Complete posting and
  settlement for wages, guarantees, recruiting, severance and vendor costs.
* Session v2 for new records, authoritative v1 reader with separately reported
  outcome compatibility. Preserve historical golden references; record changed
  state/run encodings and generate separate native references.
* T90 primary management vector: operating margin, cash/financing, completed
  exams, lost demand, access, urgent breaches, routine TAT and workforce
  obligations. Baseline-relative minimum service gates and fixed-policy tail;
  raw metrics visible on failure. T30 feedback, T180 optional validation.

Acceptance

* Step 0 evidence and design prerequisites close before action implementation;
  action-plan calibration subsequently includes recruiting/training ramp and
  upfront costs. Static profitability alone is insufficient release evidence.
* All five actions validate, preview, commit and replay through sim/API/session,
  headless and TUI paths; atomic failures/no-ops have no operational side effects.
* No V0.2 actions on existing scenarios preserves legacy patient, reporting,
  accounting, log and observable outcomes. State/run digest changes are explicit;
  old fixtures remain unchanged.
* Service/funding gates catch report deferral, patient displacement/refusal,
  eliminated coverage, endpoint obligations and selective high-margin service;
  no weighted composite conceals the metric vector.
* Payroll and cash reconcile across departure and warm-up; remote reading works
  through midnight/expiry; whole-schedule/shared-tech invariants hold.

## 4. V0.3 — Operations control

**Proposed in V0.2 design draft 2, pending the Step 0 gate.**

Deliverables

* Actions: `reassign_technologist`, `call_in_staff`, `send_home`,
  `approve_overtime_week`, `edit_template`, `reserve_addon_slots`,
  `set_overbooking`, `set_late_policy`, `reschedule_appointment`,
  `offer_earlier_slot`, `schedule_maintenance`. Initial roster editing remains
  V0.2 `set_roster`; no duplicate `change_roster` action.
* Template editor with capacity preview; discretionary rescheduling builds on
  V0.2's patient-history and whole-schedule reconciliation contracts.
* Equipment failures with noisy ETA, staff absences and late emergencies;
  scripted “CT tech calls out 07:30” and “MR down at 13:10” scenarios.
* Front-desk phone/scheduling work and abandonment; CT/MR add-on protocoling;
  QA, preparation failures, incomplete exams and repeats.
* Versioned save/load snapshots with hidden state restricted to the game/session
  boundary; acceleration through saved state, not altered simulation semantics.
* `dreamteam agent --stdio`: observation/action JSONL and scripted-policy runner
  using the same API. Mouse support in TUI.

Acceptance

* Actions validated, logged and replayable, including rejection records;
  guarantee and displacement obligations remain binding under send-home/call-in.
* Save/load continuation matches uninterrupted play; snapshots do not leak
  through observations. Scripted-policy loop runs 30 days with schema-valid JSON.
* The sick-tech scenario is solvable; a scripted greedy policy improves waits
  over no action on ≥80% of 50 seeds while retaining service/financial gates.

## 5. Financial management (formerly V0.4)

**Owner decision 2026-10-26: V0.4 is the agent evaluation harness
([BENCHMARK.md](BENCHMARK.md)); V0.6 keeps the full benchmark. This
financial-management scope is unscheduled until the owner places it.**

**Proposed in V0.2 design draft 2, pending the Step 0 gate.**

Deliverables

* Claims workflow: charge/submission lag, adjudication, partial payment, denials
  with reasons, rework policies and write-offs; self-pay estimates/collection.
* Payroll/payables calendar UI, monthly close, P&L, balance sheet, cash-flow
  statements and financing. This expands existing accrual/settlement and V0.2
  obligations; it does not defer initial guaranteed pay or severance to V0.4.
* Days in A/R, denial/net-collection rates, cost per exam, modality/exam
  contribution; statements, aging, payer views and cash forecasts from known
  obligations and observable collection estimates, without latent-date access.
* Accounting review of chart of accounts, close procedure, reconciliation and
  statement presentation (§10).

Acceptance

* Statements reconcile, including guarantee/severance/vendor liabilities and
  financing; cash-flow ties to cash. Scenario-specific calibration holds over
  six months; contribution views reveal measured capacity opportunities.

## 5a. V0.5 — LLM agents

**Owner decision 2026-10-27.** One model-agnostic LLM agent first: get one
local model making valid management decisions before memory, planning or
more benchmark infrastructure.

* **V0.5a (0.5.0-dev.1): the external LLM agent** (`agents/llm/`,
  [LLM_AGENT.md](LLM_AGENT.md)). A separate process speaking
  `dreamteam.benchmark-policy/v1`; it calls a model through an
  OpenAI-compatible API (no provider-specific dependency). Stateless: each
  decision uses the current compact observation and the action catalog. A
  strict `{"reason", "actions"}` decision, validated through the protocol
  before submission; bounded retries; no action on failure. Every model
  exchange recorded (not hashed); recorded decisions replay through the
  protocol without the model and give the live run's simulator outcome
  exactly. Tested against a mock server; one live 7-day smoke test.
  Accepted by the owner (2026-10-27, 3104e67).
* **V0.5b (0.5.0-dev.2): memory, action guidance and cost reporting**
  (llm-agent 2; owner decision 2026-10-27). No simulator, protocol or
  scoring change.
  * Bounded memory within a run, from public observations and the agent's
    own decisions only (`llm-agent-memory/v1`).
  * A targeted prompt (`llm-agent-prompt/v2`): the time and operating
    hours, active problems, a summarized observation, and every action type
    with its fields, rules, earliest dates and lead times. What is left out
    is named, and the model may ask once for details. 56% fewer tokens on
    the same recorded observations.
  * Operations and staffing actions in one reply, submitted as two
    validated batches at one decision point. The `mixed_batch` invariant is
    kept and documented (LLM_AGENT §7).
  * Per-run usage (reported, estimated and unknown tokens; latency) and an
    estimated cost from configured prices.
  * Replay rebuilds and checks the memory and prompt. V0.5a recordings
    replay; other versions are refused explicitly.
  * Mock tests, including a 90-day memory bound; one live 14-day smoke
    test.
  Accepted by the owner (d476257).
* **V0.5c (0.5.0-dev.3): multi-model development evaluation** (owner
  decision; [benchmark/v05c-dev/](benchmark/v05c-dev/README.md)). The V0.5b
  agent frozen; three local models (Qwen3.6 35B-A3B, Gemma 4 26B-A4B,
  Gemma 4 31B) under identical settings on seeds 40801–40805, declared and
  committed before any run. Stage A, 7 days, one seed; B, 30 days, three
  seeds; C, the memory-off control (`memoryDecisions` 0, the agent's one
  change) on B's seeds; D, 90 days, one seed. All 26 runs completed with
  no infrastructure failure; all 18 LLM recordings replay byte for byte,
  in CI on three platforms. No score, ranking or generalization. Accepted
  by the owner (e495789).
* **V0.5.0** (released 2026-10-28): V0.5a–V0.5c, with no further slice
  (no V0.5d, no tuning). The full benchmark framework is V0.6.

Acceptance (V0.5a): the agent takes part in a complete, replayable session:
valid structured decisions, no unauthorized operation, no action when
none is warranted, failures that change nothing. V0.5b: the agent keeps a
bounded, public-only memory across decision points and a smaller prompt
with explicit constraints, and its runs still replay exactly. V0.5.0 (owner
criteria): at least two local models complete sessions through the
external agent; their decisions are recorded and replayable; outcomes,
refusals, usage and latency can be compared; memory on and off can be
compared; existing simulator and benchmark behaviour is unchanged.

## 6. Growth and progression (formerly V0.5; deferred)

**Owner decision 2026-10-27: deferred to a future milestone, not removed.**

**Proposed in V0.2 design draft 2, pending the Step 0 gate.**

Deliverables

* Raises, workload-driven turnover/fatigue and richer onboarding; initial
  hiring, termination and guaranteed-hours contracts remain V0.2 foundations.
* Scanner/room lease or purchase, construction, financing, service contracts;
  extended hours and Sunday opening.
* Dynamic referral relationships driven by access, TAT, experience and outreach;
  demand thinning preserves common random numbers. Competition and payer shocks.
* New Practice scenario with low demand, thin cash, ramping referrals and no
  warm-up; goals, monthly reviews and valuation with measured service gates.
* Investment-case review: valuation, lease-vs-buy, expansion payback and terms
  (§10). Scenario goals use achievable service baselines rather than assuming
  zero urgent breaches or a universal 12% margin.

Acceptance

* New Practice is winnable and losable: a scripted heuristic reaches sustained
  profitability within 18 simulated months on ≥70% of seeds, meeting its gates.
* Growth exploits from PLAN §3 are tested; overloaded service degrades modeled
  referrals and finances without erasing workforce or patient obligations.

## 7. V0.6 — WebUI, benchmarks and managed practice

**Owner decision 2026-10-27: V0.6 is the full agent benchmark**, evaluating
agents across models, scenarios, seeds and baselines. The language-model
agent itself (item 3) starts in V0.5, over stdio.

**Owner direction 2026-10-28 (V0.6 starting scope, not yet a plan).** From
building agents to evaluating their capabilities: operational, financial
and staffing management, long-term planning, and robustness across
practice conditions. V0.6a starts with a small suite of distinct practice
scenarios from documented simulator capabilities (for example financially
constrained, a reporting backlog, rising demand, staffing shortages, and a
stable practice where needless intervention is costly), with known initial
conditions and reproducible seeds, and an evaluation contract. Development
scenarios are kept apart from held-out benchmark scenarios, and both from
the sealed clinical holdouts. Releasable with a documented scenario suite,
reproducible multi-agent evaluation, the existing operational and
financial metrics and a standard report; no leaderboard, composite score
or statistical ranking required. Not another round of prompt tuning.

**Owner decision 2026-10-28: V0.6 slices.** The V0.4 harness stays the
foundation; no second benchmark engine.
* **V0.6a (0.6.0-dev.1): scenario suite and evaluation contract**
  ([BENCHMARK_SUITE.md](BENCHMARK_SUITE.md)). Five development scenarios
  from existing mechanics and documented calibration sources (financial
  pressure, reporting backlog, demand growth, staffing shortage, stable
  practice); the contract `dreamteam.benchmark-contract/v1` (six dimensions
  of existing metrics, no score; T90 only where its policy applies);
  development and held-out governance (no held-out seeds created); a
  reporting contract; a baseline characterization with no action, `greedy`
  and `rules` on three declared seeds
  ([benchmark/v06a-dev/](benchmark/v06a-dev/README.md)).
* **V0.6a** accepted (owner, 2026-10-28; CI green on a1afb41): demand
  growth provisionally, financial pressure as limited. Insolvency
  mechanics and dated demand shocks deferred until after V0.6.0.
* **V0.6b (0.6.0-dev.2): suite runner and standard reports.**
  * `dreamteam benchmark suite run` and the contract report
    (`dreamteam.benchmark-contract/v1`) on the harness, unchanged;
  * a financially conservative built-in baseline, `conservative` v1;
  * a prospective demand diagnostic, `dreamteam.demand-diagnostic/v1`;
  * a 60-run development matrix
    ([benchmark/v06b-dev/](benchmark/v06b-dev/README.md)).

  `conservative` v1 took no action in any run there.
* **V0.6c (0.6.0-dev.3): conservative baseline v2 and frozen-agent
  evaluation.**
  * `conservative-v2`: acts only on sustained breaches it can afford, with
    a review and retreat;
  * Qwen3.6-35B-A3B and Gemma 4 26B-A4B (the V0.5 agent, unchanged)
    against the baselines on all five entries, at 30 days (three seeds)
    and 90 days (one seed);
  * every LLM run replayed without a model
    ([benchmark/v06c-dev/](benchmark/v06c-dev/README.md)).
* **V0.6c** accepted and **V0.6.0 released** (owner, 2026-10-29): a
  documented, reproducible benchmark for practice-management agents. The
  known limitations are documented prominently
  ([BENCHMARK_SUITE.md](BENCHMARK_SUITE.md) §6). Unpaid radiologist shift
  extensions are flagged as a simulator-modeling limitation, not fixed in
  the release.

**Proposed in V0.2 design draft 2, pending the Step 0 gate.**

Deliverables, staged within the milestone

1. WebUI through `session`/`api`, with TUI dashboard/schedule/worklist/finance
   parity; SQLite saves index, run records, benchmark results and long logs.
2. Benchmark packs: scenarios, public development/private test seeds, horizons,
   evaluation tails, decision epochs, perturbations, metric specifications and
   constraints. Public/private split and isolated processes per PLAN §9.7.
   Baselines: no action, rules, greedy, random-valid and simple look-ahead.
   Reports retain vectors, confidence intervals, constraints and rejected
   actions; management eligibility remains gate-based, without a weighted score.
3. Provider-independent language-model agent harness over stdio/HTTP with budget
   and token accounting; human-versus-agent comparisons on the same decision protocol; hidden-state
   audits, hard-constraint attempts, urgent service and workforce evaluation.
   Automation levels: manual, automated scheduling/billing, operations manager.

Acceptance

* Two baselines ×100 seeds ×30 days in <5 minutes on the benchmark Pi target;
  results reproduced from records, with version/schema compatibility explicit.
* WebUI and TUI produce identical native digests for identical actions.
* A managed-practice agent completes a pack without harness errors, with all
  actions replayed and no hidden-state access. Human and agent comparisons use
  the same service gates, commitment accounting and evaluation tail.

---

## 7a. V0.7 — simulator realism and agent decision-quality evaluation

**Owner decision 2026-10-29:**
* V0.6.0 is accepted and closed.
* V0.7 is approved as simulator realism.
* V0.7a begins with a compatibility design review, and is implemented
  only if historical behaviour can be preserved.

| Milestone | Scope |
|---|---|
| V0.7a | Overtime economics and historical compatibility ([V0.7_OVERTIME_DESIGN.md](V0.7_OVERTIME_DESIGN.md)) |
| V0.7b | Financial solvency and cash constraints ([V0.7_SOLVENCY_DESIGN.md](V0.7_SOLVENCY_DESIGN.md)) |
| V0.7c | Frozen-agent evaluation under overtime and solvency economics ([benchmark/v07c-dev/](benchmark/v07c-dev/PLAN.md)) |
| V0.7.0 | Release |

Decision/rationale consistency, dated demand shocks and staffing
disruptions can follow in later versions.

**V0.7a (0.7.0-dev.1), in development:**
* the design note;
* a scenario policy, `radiologistOvertime`, under which an on-site
  salaried radiologist's time on duty past the rostered end is paid at the
  salary's hourly rate;
* suite `dreamteam-dev` version 2 (the five practices with it on);
* a prospective characterization with the deterministic baselines
  ([benchmark/v07a-dev/](benchmark/v07a-dev/PLAN.md)).

**V0.7a accepted and closed** (owner, 2026-10-29). Manager overtime is
deferred until the solvency model is stable.

**V0.7b (0.7.0-dev.2), in development** (owner decision 2026-10-29). It
models a practice's ability to finance operations, not bankruptcy at zero
cash. It has three concepts:
* **available cash** (cash plus undrawn credit);
* **a credit facility** (a fixed limit, deterministic interest);
* **financial distress** (obligations that cannot be financed, recorded).

Its consequences work through existing constraints:
* a discretionary commitment the practice cannot fund is refused;
* committed obligations are always honoured;
* there is no shutdown.

Its deliverables:
* the design note;
* the opt-in setting `finance.solvency`;
* suite `dreamteam-dev` version 3 (the five practices, with a $50,000
  credit line and with none);
* a prospective characterization
  ([benchmark/v07b-dev/](benchmark/v07b-dev/PLAN.md)).

Not approved in V0.7b: manager overtime and V0.7c's LLM evaluations.

**V0.7b accepted** (owner, 2026-10-29, once CI on 9e73520 was green).
The finding kept: the five practices generally have enough liquidity that
solvency is not their dominant management problem. No scenario is made to
force insolvency.

**V0.7c (0.7.0-dev.3), in development** (owner decision 2026-10-29). The
last development evaluation before V0.7.0:
* V0.6c's two frozen LLM agents (Qwen3.6-35B-A3B, Gemma 4 26B-A4B), with
  the V0.6c prompt, memory and settings unchanged;
* no action, greedy and conservative-v2 as comparators;
* the five practices with overtime paid and solvency on (suite version 3,
  `-credit`);
* stage A, two seeds for 30 days (50 runs), and stage B, one seed for 90
  days (25 runs), both declared before stage A
  ([plan](benchmark/v07c-dev/PLAN.md));
* comparisons paired within V0.7 only; V0.6 results are context.

Not approved in V0.7c:
* new low-cash or longer-horizon scenarios;
* new management actions;
* manager overtime;
* prompt tuning;
* conservative-v3;
* new scoring.

**V0.7.0 release criteria** (owner):
* overtime and solvency accounting stay deterministic;
* historical sessions and V0.7 development sessions replay;
* existing fingerprints and golden references are unchanged;
* all 75 V0.7c runs complete or have documented failures;
* completed LLM runs replay without model access;
* financial and operational outcomes are reported together;
* cross-platform CI is green.

The agents need not perform well: poor financial decisions are legitimate
findings.

Not approved in V0.7a: solvency constraints, new staffing mechanics,
demand shocks, agent prompt changes, retuning the baselines, or rerunning
the V0.6 LLM comparison.

The original proposal follows.
V0.6c showed that an agent can greatly improve reporting while spending
far more than a practice should tolerate. Before the benchmark grows
(more models, seeds or scenarios), the simulator should represent the
financial consequences of management actions faithfully.

Candidate areas, in the owner's order of priority:
1. **Financial consequences of extended shifts.** Radiologist shift
   extensions are currently free (salaried staff), which creates
   unrealistic incentives.
2. **Financial solvency constraints.** Negative cash should eventually
   have operational consequences.
3. **Decision and rationale consistency.** Compare what an agent says it
   intends with what it submits, and with the financial constraints it
   can observe.
4. **Dated demand shocks.** Test whether agents anticipate and respond to
   changing conditions.
5. **Staffing absences and turnover:** operational disruptions.
6. **Agent responses to changing practice conditions.**

The owner suggests beginning with the first two, but only after reviewing
their impact on historical compatibility and deterministic replay. No
further model experiments are planned until the simulator's economics are
improved. Nothing here is implemented in V0.6.0.

## 8. Testing strategy (all versions)

| Layer | What | When |
|---|---|---|
| Unit | `simtime`, `rng` samplers, queue, ledger, policies | every commit |
| Table tests | action validation, state transitions | every commit |
| Property | chunking invariance; invariants over random seeds; conservation; replay equality | every commit (short), nightly (long) |
| Golden | start-state digest, 7-day and 90-day digests per preset; CLI output; TUI views | every commit; preserve historical references and record separate native references for intentional gameplay or state-schema changes |
| Cross-arch | golden digests on `amd64` and `arm64` (GitHub Actions has native arm64 runners) | CI |
| Leakage | observation deny-list reflection; same-past/different-future; StepVisible | every commit |
| Calibration | multi-seed KPI bands per preset (`-tags calibration`, minutes) | before release, after balance changes |
| Fuzz | action JSON decode + validation; scenario JSON validation | nightly |
| Benchmarks | `go test -bench` for build, 90-day run, projection | before release |
| Static | `go vet`, `staticcheck`, a grep test forbidding `time.Now`, `math/rand` (v1), `range` over maps in `sim` | every commit |
| Manual | release smoke checklist: 80×24, resize, NO_COLOR, Max speed responsiveness, quitting | before release |

---

## 9. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Overtime as an exploit (admit everything, run late, reschedule at the cap) | medium | overtime cost in the P&L, workforce and access metrics scored, coverage reschedules counted as access failures, evaluation tail (SIMULATION_DESIGN §6.9) |
| Scope creep before anything is playable | high | walking skeleton: TUI at step 4, every step adds something visible; deferral list is binding |
| Realism without data | high | every number labelled; quantile-table format ready for real RIS data; calibration targets rather than "truth"; workbook match is consistency only; independent calibration against anonymized operational data (and advisor review) before claiming realism |
| Warm-up too slow on the Pi | medium | synthetic-load benchmark at step 3; build benchmark at step 11; fallback: cache built start states |
| Determinism bugs (maps, floats, ordering) | high | integer engine, keyed RNG, cross-arch golden tests, static checks |
| Hidden-state leakage into agents | high | unexported latents, `api` boundary, leakage tests, step-to-visible |
| Balance: game boring (watching a screensaver) | medium | three meaningful actions in V0.1; insights in daily summary; V0.2 priority on interaction |
| Balance: financial exploits | medium | exploit list (PLAN §3) as test cases; multi-objective metrics |
| TUI performance on Raspberry Pi | medium | 10 fps, cached observation, bounded frame work, benchmarks |
| Bubble Tea / Lip Gloss v2 API churn | low–med | pin versions; thin wrapper for layout helpers |
| Over-abstraction | medium | no interfaces until second implementation; one `sim` package until it hurts |
| Perceived as clinical advice | low | clear disclaimer: operational sim with synthetic data, no clinical decisions |
| CPT/licensing issues | low | internal exam codes only |

---

## 10. External finance tooling

Optional helpers for the business side. Neither is a dependency of the Go
code, and neither replaces imaging-specific data.

| When | Tool | Use | Output |
|---|---|---|---|
| V0.1 step 18 | Spreadsheet (xlsx) | **P&L calibration workbook**: monthly volume × payer mix × rates − payroll − fixed − variable costs, with one input sheet per profitability preset. Lets assumptions be tuned and sanity-checked without running the sim | `docs/calibration/pnl-model.xlsx`; inputs mirror scenario JSON keys |
| V0.4 | Anthropic **Finance** plugin (journal entries, reconciliation, statements, variance analysis, month-end close) | review ledger design, close procedure, statement reconciliation | review notes in a turn report; fixes in `sim` |
| Growth (formerly V0.5) | Anthropic **financial-analysis** plugin (DCF, three-statement models) | valuation proxy, lease-vs-buy, expansion payback | investment-case notes; parameters in scenario JSON |

Limits: these plugins are general corporate finance. Imaging-specific numbers
(reimbursement, denial rates, days in A/R, staffing ratios, radiologist
compensation) need healthcare sources such as MGMA, HFMA and CMS fee schedules.
Those sources and the label for each number go in `docs/ASSUMPTIONS.md`.
Install each plugin only when its version starts.
