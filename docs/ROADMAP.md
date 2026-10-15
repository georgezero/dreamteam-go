# DreamTeam — Roadmap

Status: **draft for review**.
Companion documents: [PLAN.md](PLAN.md), [SIMULATION_DESIGN.md](SIMULATION_DESIGN.md),
[TUI_DESIGN.md](TUI_DESIGN.md).

Each version is a playable build. Nothing ships as "framework only".

---

## 1. Milestones at a glance

| Version | Theme | Depends on |
|---|---|---|
| **V0.1** | Engine + playable TUI: a typical Tuesday at an established practice | — |
| **V0.2** | Practice design: sustainable-scenario gate, five staffing/reading actions, contracts and lifecycle, T90 evaluation | V0.1; Step 0 gate |
| **V0.3** | Operations control: scheduling, disruptions, QA/prep, save/load, agent-interface skeleton | V0.2 |
| **V0.4** | Financial management: claims, denials, statements, monthly close and financing | V0.2 obligations; V0.3 controls |
| **V0.5** | Growth: equipment, extended hours, referrals, New Practice, progression and richer workforce dynamics | V0.3, V0.4 |
| **V0.6** | WebUI/SQLite, benchmark framework, managed-practice harness, comparisons and automation levels | V0.3 API, V0.4 statements, V0.5 growth benchmarks |

**V0.2–V0.6: proposed in V0.2 design draft 2, pending the Step 0 gate** (first
sweep 2026-10-14 and Step 0B 2026-10-15 did not pass; Step 0C 2026-10-15, a
revised staffing architecture, passes conditionally at ×1.30 demand with a
≈2.8% margin: CONDITIONAL GO, approved by the owner 2026-10-15, design §18).
V0.2 is being implemented in slices; Slice 1, the employee contract and
capacity foundation with no new action, is built (design §5.5).
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

## 5. V0.4 — Financial management

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

## 6. V0.5 — Growth and progression

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
| V0.5 | Anthropic **financial-analysis** plugin (DCF, three-statement models) | valuation proxy, lease-vs-buy, expansion payback | investment-case notes; parameters in scenario JSON |

Limits: these plugins are general corporate finance. Imaging-specific numbers
(reimbursement, denial rates, days in A/R, staffing ratios, radiologist
compensation) need healthcare sources such as MGMA, HFMA and CMS fee schedules.
Those sources and the label for each number go in `docs/ASSUMPTIONS.md`.
Install each plugin only when its version starts.
