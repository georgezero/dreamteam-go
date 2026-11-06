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
| **V0.7** (released, `v0.7.0`: financial realism, V0.7a–V0.7c, by the owner's release decision 2026-10-30) | Simulator realism: paid radiologist overtime and opt-in financial solvency (credit, interest, distress, affordability of new commitments), versioned benchmark suites, and a frozen-agent evaluation under the new economics (§7a) | V0.6 benchmark |
| **V0.8** (released, `v0.8.0`: auditable decision evidence, V0.8a–V0.8c, by the owner's release decision 2026-10-31) | Auditable decision evidence for practice-management agents: deterministic, versioned consistency rules with abstention, a frozen review package (review pending), paired financial outcomes, refusal probes, complete attempt recording (llm-agent 3) and a prospective funding-refusal evaluation (§7b). Evaluation and auditability, not clinical validation | V0.7 |
| **V0.9** (released, `v0.9.0`: financial decision evidence and preview-based decision support, V0.9a–V0.9c, by the owner's release decision 2026-11-01) | Evidence and evaluation capabilities: frozen parameter evidence, an offline review workflow (review pending), financial decision evidence (affordability, reversibility, operational exposure), an engine-backed preview agent (llm-agent 4) with complete recording, and a prospective preview evaluation that established no financial benefit (§7c). Not clinical validation | V0.8 |
| **V0.10** (released, `v0.10.0`: inference repeatability and controlled decision experiments, V0.10a–V0.10c, by the owner's release decision 2026-11-01) | A methodology and measurement release: live inference repeatability, a versioned shared-prefix method (v1–v3) with model-free exact replay, two prospective decision experiments (preview against neutral, then neutral, validation and preview), and an inventory of what validate and preview tell an agent (§7d). No demonstrated financial or operational benefit of the preview; not clinical validation | V0.9 |
| **V0.11** (released, `v0.11.0`: the validation-first agent workflow, V0.11a–V0.11b, by the owner's release decision 2026-11-02) | Agent workflow and evaluation capabilities: llm-agent 5, validation first with exact refusal feedback and bounded retries; `validation-first` the default and the selective preview opt-in (`--workflow`); complete v5 recording and model-free replay; a prospective decision-level evaluation against llm-agent 3 (§7e). No claim of improved financial decision quality; not clinical validation | V0.10 |
| **V0.12** (released, `v0.12.0`: high-impact action acknowledgment and reproducible decision evidence, V0.12a–V0.12b, by the owner's release decision 2026-11-04) | Decision safeguards: llm-agent 6, engine-backed acknowledgment of high-impact actions, an explicitly selected experimental agent (llm-agent 5's validation-first stays the default); an independent safeguard audit; a prospective live evaluation against llm-agent 5's preview (420 draws): every acknowledgment accurate, none changed a high-impact action (§7f). No decision-quality benefit established; not clinical validation | V0.11 |
| **V0.13** (in progress: V0.13a, V0.13b and V0.13c accepted; V0.13d workflow and experiment preparation done, awaiting review; protocol not locked) | Decision safeguards beyond acknowledgment: the independent human evidence review; explicit comparison of engine-derived consequences of alternatives before irreversible actions; financial decision-quality evaluation; multi-day prospective testing only after the safeguards are reviewed (§7g) | V0.12 |
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

**V0.7c (0.7.0-dev.3)** (owner decision 2026-10-29). The
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

**V0.7c results** ([README](benchmark/v07c-dev/README.md)):
* 75 of 75 runs completed, none rerun;
* every LLM run replays without the model, byte-identical across
  linux/arm64 and linux/amd64;
* no action was refused for funds;
* one run (Qwen, staffing shortage, 90 days) drew on credit after
  terminating its technologists and front desk.

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

**V0.7c accepted; V0.7.0 released** (owner, 2026-10-30; V0.7c at ad6d25c,
CI run 41015378407 green). Every release criterion was met. The release
notes record, as development observations:
* Qwen improved service but frequently reduced profit;
* in one 90-day staffing-shortage run, Qwen's staffing changes, stated as
  avoiding insolvency, left both financial and operational outcomes far
  worse than no action;
* solvency constraints remain untested by agent behaviour: no V0.7c
  action was refused for insufficient funds. Solvency accounting and
  enforcement are shown by deterministic tests and the baseline
  characterization, not by LLM responses to refusals.

No V0.7d. No model retuning or further financial scenarios before the
release.

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

## 7b. V0.8 — agent decision quality (released)

The owner's recommendation after V0.7c. With the simulator's economics
improved, the next weakness is the agents' ability to make coherent
management decisions. The V0.7c staffing-shortage run is the example: the
agent took actions that undermined the objective it said it was pursuing.

| Milestone | Scope | Status |
|---|---|---|
| V0.8a | Deterministic decision-evidence analysis | complete: accepted by the owner 2026-10-30 (80cfff9, CI run 41104143975) |
| V0.8b | Evidence validation and financial outcome analysis | complete: accepted by the owner 2026-10-30 (5b6cdba, CI run 41133370321) |
| V0.8c | Complete decision recording and prospective refusal-response evaluation | complete: accepted by the owner 2026-10-31 (declared ee8aabc, results da52a77, CI run 41171062478) |
| V0.8.0 | Release | released 2026-10-31, tag `v0.8.0` |

**V0.8c accepted; V0.8.0 released** (owner, 2026-10-31). Every release
criterion below was met; the independent review is pending and disclosed.
The release notes record, as development observations:
* complete recording changes what can be measured: repeat evidence v2
  confirms none of frozen v1's 202 historical repeat links, and 1 of its
  13 V0.8c links. v1 stays reproducible, but its repeat counts are not
  confirmed exact-action repeats;
* 10 funding refusals were observed, all in the low-liquidity condition
  on one seed (Qwen 8, Gemma 2). The next proposals: no action 5,
  different 3, modified 2 (the effective date only), exact repeat 0.
  These are counts, not rates;
* Qwen lowered profit against no action in all four of its runs; Gemma's
  outcomes equalled no action. Consistency, financial and operational
  outcomes are kept separate.

No V0.8d. No simulator, agent, prompt or scoring change before the
release.

**V0.8c as authorized** (owner, 2026-10-30, after accepting V0.8b):
* **Recording:** llm-agent 3 records every attempt
  (`dreamteam.llm-decision-record/v3`, [LLM_AGENT.md §13](LLM_AGENT.md)).
  llm-agent 2 and its recordings are unchanged.
* **Repeat rule version 2:** recognizes repeated refused proposals by
  action content, separately versioned. The frozen v1 rules are unchanged.
* **Independent review:** continues separately. Its status stays explicit,
  and V0.8c makes no claim of independent validation.
* **A small controlled live evaluation:**
  * conditions: the existing solvency scenario and one low-liquidity
    variant made of existing settings;
  * models: the two frozen models;
  * seeds and horizon: two fresh development seeds, 30 days;
  * runs: at most eight live LLM runs, with deterministic comparators.
* **Not approved:** prompt or model tuning, new mechanics, changes to the
  frozen v1 rules.
* **Declared before any live run:** [PLAN.md](benchmark/v08c-dev/PLAN.md).
  Built: [docs/benchmark/v08c-dev/](benchmark/v08c-dev/README.md).

**V0.8.0 release criteria** (owner):
* V3 captures every attempt, rejected ones included.
* Historical recordings and replay are unchanged.
* Repeat v2 is versioned apart from the frozen v1.
* All planned runs completed, or their failures documented.
* Every completed run replays without model access.
* Financial and operational outcomes are kept apart from consistency
  findings.
* Cross-platform CI is green.
* The independent review status is reported accurately. Its completion is
  not a release blocker, provided the unvalidated findings are labelled.

**V0.8b as authorized** (owner, 2026-10-30), with no live LLM run, no new
mechanic, and no change to recordings, prompts, policies or scoring:
* **Freeze:** `dreamteam.decision-evidence/v1` is frozen
  ([FREEZE.md](benchmark/v08a-dev/FREEZE.md)). Rule changes need a new
  version and owner approval.
* **Independent review:** the seeded review package is prepared. The
  review stays PENDING until a genuinely independent reviewer completes
  it. The developer's own review does not count.
* **Funding-refusal probes:** deterministic scripted sequences through the
  agent protocol test the mechanics and the evidence, not a model.
* **Refusal responses:** a separate extension,
  `dreamteam.refusal-response/v1`, classifies what was proposed after a
  refusal, without inferring understanding.
* **Outcomes:** descriptive financial and operational outcomes paired with
  no action, never combining V0.6 and V0.7 economics, with no score,
  cause or counterfactual.
* **V0.8c recording contract:** designed, not deployed.

Built: [docs/benchmark/v08b-dev/](benchmark/v08b-dev/README.md).

**V0.8a as authorized** (owner, 2026-10-30, after accepting V0.7.0). It
measures whether an agent's explanation is consistent with its actions,
not whether its decisions were good. It is a read-only, deterministic
analysis of the existing recordings, with no new model run, no simulator
change and no change to the agents.
* **Five stages are kept apart:** intent (the recorded reason), proposal,
  validation, execution and consequence.
* **Six rules:** explicit action omitted, explicit opposite action,
  financial amount mismatch, repeated invalid proposal, stated action not
  executed, factual observation mismatch.
* **Three statuses:** supported, unresolved, insufficient evidence. The
  analysis abstains whenever a statement is not explicit.
* **Built:** the contract `dreamteam.decision-evidence/v1`
  ([DECISION_EVIDENCE.md](DECISION_EVIDENCE.md)) and the analysis of the
  V0.5a–V0.7c recordings
  ([docs/benchmark/v08a-dev/](benchmark/v08a-dev/README.md)).
* **Rejected:** no LLM judge, and no verdict on financial wisdom.

The owner's proposal that preceded it:

**V0.8a would distinguish three things:**
1. **Stated intent:** what the agent says it is trying to accomplish.
2. **Submitted decision:** what it actually proposes.
3. **Observed consequence:** what happens after the actions execute.

| Agent statement | Submitted action | Potential finding |
|---|---|---|
| "Reduce staffing expenses" | hire additional staff | intent/action contradiction |
| "Preserve liquidity" | commit substantial new spending | possible financial inconsistency |
| "Cancel teleradiology" | no cancellation submitted | intent/action mismatch |
| "Avoid insolvency" | terminate revenue-generating staff | requires consequence analysis |
| "Wait for scheduled coverage" | no action | intent/action agreement |

* An action is not labelled irrational because its eventual outcome was
  unfavourable. That needs a separate, outcome-based analysis.
* The aim is a reproducible evidence framework for identifying
  contradictions, not an LLM judge assigning subjective quality scores.
* The suggested start is a read-only analysis of the existing recorded
  decisions (V0.5c–V0.7c), before any new agent experiment. This is
  V0.8a.

**Other candidate directions** (not scheduled):
* dated demand shocks;
* staff absences and turnover;
* additional benchmark scenarios;
* held-out benchmark governance in practice (a held-out set and its
  execution).

V0.8a, V0.8b and V0.8c were authorized and released in V0.8.0.

## 7c. V0.9 — agent decision reliability (released)

The owner's recommendation after V0.8c: shift from building evidence
infrastructure to using it to improve the reliability of management
agents. First, independent review of the existing evidence rules, then
more specific decision/action consistency checks. No further simulator
mechanics and no large LLM benchmark first.

| Milestone | Scope | Status |
|---|---|---|
| V0.9a | Independent review workflow and decision/action parameter consistency | accepted and closed 2026-10-31 (a5e5f9b, 0.9.0-dev.1) |
| V0.9b | Financial decision quality and operational exposure | accepted and closed 2026-10-31 (8ffae8d, 0.9.0-dev.2) |
| V0.9c | Prospective preview evaluation: does showing the simulator's existing preview change management decisions? | accepted and closed 2026-11-01 (declared d4d2f2f, results be537f2, gates a38d031, CI run 41274622273; 0.9.0-dev.3) |
| V0.9.0 | Release | released 2026-11-01, tag `v0.9.0` |

**V0.9a as authorized** (owner, 2026-10-31, after accepting V0.8.0):
* **Independent review:** a separate human task. The implementation may
  prepare, validate, import and analyse labels, but may not claim
  validation without an actual independent reviewer.
* **Precision:** only with a clear denominator and sampling
  qualification.
* **Recall:** never from the 98 items. It would need a separate
  annotation protocol over eligible statements, misses included.
* **Parameter evidence:** `dreamteam.parameter-evidence/v1`, a separate,
  versioned contract:
  * amount, target, date, duration and direction mismatches;
  * explicit statements only;
  * abstention preferred.
* **Not approved:** simulator, agent or prompt changes; live LLM runs;
  scoring; an LLM judge; changes to frozen evidence.
* **Built:** [docs/benchmark/v09a-dev/](benchmark/v09a-dev/README.md).
  * The review is still PENDING.
  * The parameter evidence made 816 eligible checks on V0.5c–V0.8c: 0
    supported mismatches, 54 abstentions.

**V0.9a, accepted** (owner, 2026-10-31). The owner's qualification: the
zero supported parameter mismatches are an in-sample result after the
rules were narrowed on the same corpus. So parameter-evidence/v1 is frozen,
and not tuned against these recordings again.

**V0.9b as authorized** (owner, 2026-10-31): financial decision quality
and operational exposure, a read-only analysis.
* **Three dimensions,** with evidence completeness as a fourth property:
  affordability, reversibility and operational exposure, never combined
  into a score.
* **No financial model on top of the simulator:** use the engine's exact
  values where it gives them, a supported qualitative property where it
  does not, and otherwise abstain.
* **Exposure in three parts:** direct state changes, subsequent observed
  outcomes and paired run-level differences. No outcome is attributed to
  one action because it followed it.
* **Not approved:** simulator, agent or prompt changes; live LLM runs; new
  benchmark scoring.
* **Built:** [docs/benchmark/v09b-dev/](benchmark/v09b-dev/README.md),
  `dreamteam.financial-evidence/v1`:
  * 78 recordings from V0.6c, V0.7c and V0.8c, every engine answer
    reproduced;
  * 2,062 records;
  * case studies of the V0.7c staffing-shortage run, a limited-intervention
    contrast and the V0.8c low-liquidity refusals.

**V0.9b, accepted** (owner, 2026-10-31). The finding that sets V0.9c's
direction: the simulator could state a decision's commitments and direct
effects (such as displaced appointments) when it was made, but the agents
never requested that preview. The preview reports immediate commitments
and direct effects under the simulator's rules; it is not a forecast of
profit or turnaround.

**V0.9c as authorized** (owner, 2026-10-31): a controlled, prospective
comparison of two conditions.
* **Baseline:** llm-agent 3, unchanged.
* **Preview-enabled:** a new agent version that receives the engine's
  preview of a proposed management batch before finalizing it, and may
  keep, revise or abandon it. The preview workflow is the only intended
  treatment difference: no broader prompt redesign, no coaching.
* **Matrix:** Qwen3.6-35B-A3B and Gemma 4 26B-A4B; financial pressure and
  staffing shortage; two fresh development seeds; 30 days; both
  conditions. 16 live LLM runs, with matching no-action runs.
* **Not in V0.9c:** single-decision causal interventions (withholding one
  action changes later observations and decisions, so it is not a clean
  estimate of one action's effect); simulator, accounting or scoring
  changes; tuning against results.
* **Declared:** [docs/benchmark/v09c-dev/PLAN.md](benchmark/v09c-dev/PLAN.md),
  with llm-agent 4 (`agents/llm/llm_agent4.py`, LLM_AGENT.md §14), the
  treatment record [TREATMENT.md](benchmark/v09c-dev/TREATMENT.md), seeds
  42001 and 42002.
* **Evaluated:** [docs/benchmark/v09c-dev/](benchmark/v09c-dev/README.md).
  * 20 of 20 runs; every live run replays exactly.
  * Nine previews were shown. The five that showed a refused action were
    revised or abandoned; the four that showed accepted, funded actions
    were kept.
  * No preview showed an unfunded commitment or displaced bookings.
  * The model server's answers varied at byte-identical requests, so the
    run-level pairs cannot separate the preview from that variability.

The owner plans to release V0.9.0 after V0.9c whatever its result: a
negative result would show that making financial consequences visible is
not, by itself, enough to improve management decisions.

**V0.9c accepted; V0.9.0 released** (owner, 2026-11-01). Every release
criterion was met; the independent review is pending and disclosed. The
owner's assessment:
* V0.9c measured the preview interactions, but it could not identify their
  downstream financial effect. **No financial benefit of preview access
  was established.**
* In the nine interactions, the agents kept proposals the preview showed
  as accepted (4) and revised (4) or abandoned (1) those it showed as
  refused. These are observed interactions, not estimates.
* The larger finding is model nondeterminism: Qwen matched identical
  requests at 0 of 5 compared points, every Qwen pair diverged before any
  preview, and one pair without a preview differed in profit by
  $6,646.03. This is a design limitation of the experiment, not a failure
  of recording or replay.

No V0.9d. No simulator, agent, prompt, model or evaluation change before
the release.

Directions recorded:
* **independent evidence review:** label the frozen 98-item package by an
  independent reviewer, then report agreement, precision and recall
  under its protocol;
* **improved financial decision analysis;**
* **decision/action parameter consistency:** whether stated amounts,
  dates, targets and values agree with the submitted action's
  parameters;
* **broader refusal-response evaluation,** on more seeds and conditions,
  with complete recording;
* **agent decision-quality benchmarking,** without a leaderboard.

Each rule change is a new, separately versioned rule, and earlier
findings stay under their original versions. V0.9a, V0.9b and V0.9c are
accepted and V0.9.0 is released. Later steps each need the owner's GO.

## 7d. V0.10 — experimental validity (released)

The owner's recommendation after V0.9c. DreamTeam can replay an
experiment exactly; the next question is whether an experiment can
distinguish the effect of a decision-support intervention from ordinary
variation in model outputs. V0.10 is about experimental validity rather
than more financial features.

| Milestone | Proposed scope | Status |
|---|---|---|
| V0.10a | Inference repeatability and controlled-comparison design | accepted 2026-11-01 (de8ac79, CI 41326685680) |
| V0.10b | Live shared-prefix decision experiments | accepted 2026-11-01 (ab1d8d1, d3da060; CI 41349684200) |
| V0.10c | Validation feedback against preview information | accepted 2026-11-01 (679483e, 222168e; CI 41378207630) |
| V0.10.0 | Release | released 2026-11-01, tag `v0.10.0` |

Proposed work:
* **independent evidence review:** label the frozen 98-item package by an
  independent reviewer;
* **controlled inference repeatability experiments:** how often identical
  requests give identical outputs, and under what server settings;
* **decision-level intervention evaluation:** comparisons that differ in
  one decision, from a shared recorded prefix;
* **improved preview coverage:** beyond staffing actions;
* **financial consequence communication:** how the preview's facts are
  presented to an agent.

**V0.9.0 accepted; V0.10a as authorized** (owner, 2026-11-01). The main
remaining problem is experimental control, not simulator functionality:
in V0.9c Qwen's answers diverged before the preview, so the financial
differences could not be attributed to it. V0.10a is a methodology and
measurement milestone in two parts:
* **Part A, inference repeatability:** how often the same model, prompt,
  settings and server state give identical responses, at the levels of
  raw text, parsed reply, proposed actions, validated actions and the
  decision. A difference in wording is not necessarily a difference in
  management decisions. Cold and warm, sequential and concurrent
  requests, and prompt-cache behaviour, where the server allows them;
  otherwise marked unavailable.
* **Part B, controlled comparison design:** a versioned shared-prefix
  method that branches from an identical recorded state and initial
  proposal, one branch shown the preview, the other not, with an offline
  prototype on recorded or scripted responses.
* **Not authorized:** a new 30- or 90-day live LLM evaluation; simulator
  mechanics; agent prompt or decision changes; scoring changes; restarting
  the shared inference service.
* **Declared:** [docs/benchmark/v10a-dev/PLAN.md](benchmark/v10a-dev/PLAN.md),
  before any request.
* **Done:** [docs/benchmark/v10a-dev/](benchmark/v10a-dev/README.md).
  * 25 recorded requests, 375 live requests, every one answered. Across 15
    requests of a prompt, the text was identical for 2 of 25 and the
    decision the same for 17 of 25.
  * Sent one at a time, the decision changed at 3 of 25 prompts; sent two
    at a time, at 8. Staffing proposals varied most. `seed` made no
    visible difference, and the prompt cache was nearly always hit.
  * Cache clearing, cache disabling and a cold start were unavailable.
  * The shared-prefix design ([DESIGN.md](benchmark/v10a-dev/DESIGN.md))
    and its offline prototype, on the 9 V0.9c preview points: identical
    prefixes and proposals, the treatment arm equal to the recording.
    Nothing causal is claimed.

**V0.10a accepted; V0.10b authorized** (owner, 2026-11-01): a bounded,
prospectively declared decision-level experiment, not another 30- or
90-day benchmark.
* **Design:** at 8 recorded branch points, the state and initial staffing
  proposal are held fixed. The control arm gets a neutral continuation;
  the treatment arm gets the engine's exact preview. 20 draws per arm per
  point, 320 requests, sequential and in a randomized interleaved order.
  The outcome is keep, revise or abandon; the simulator continues only
  with a fixed no-action suffix.
* **Inventory:** the declared 2 accepted-preview and 2 refused-preview
  points per model was impossible for Gemma, which has one
  accepted-preview context
  ([inventory](audits/v0.10b-branch-point-inventory.md)). The owner
  approved option A: Gemma 1 accepted and 3 refused; Qwen 2 and 2.
* **Declared:** [docs/benchmark/v10b-dev/PLAN.md](benchmark/v10b-dev/PLAN.md)
  and [SPEC.md](benchmark/v10b-dev/SPEC.md) (`shared-prefix/v2`), before
  any request.
* **Done:** [docs/benchmark/v10b-dev/](benchmark/v10b-dev/README.md). 320 of
  320 requests.
  * Where the preview showed a refusal, the treatment changed the
    decision in 99 of 100 draws; the control resubmitted the refused
    proposal in 77.
  * Where it showed nothing refused, the arms were the same at 2 of 3
    points.
  * The control has no validation feedback, so the comparison is the
    preview against no information.
  * Recommended for V0.10c: a validation-feedback control and
    prospective points with financial stakes.

**V0.10b accepted; V0.10c authorized** (owner, 2026-11-01). The preview
must be compared with the agent's actual validation feedback, not only
with a neutral message.
* **Arms:** A neutral, B the engine's validation answer, C the full
  preview. The primary comparison is C − B; A is a reference.
* **Points:** accepted and refused proposals, with financial stakes
  (credit-supported, large, reversible, with retained obligations,
  displacing bookings).
* **Scope:** immediate decisions and the deterministic no-action suffix;
  no multi-day evaluation.
* **Release:** the owner intends V0.10.0 after V0.10c, even if
  validation proves as effective as the preview.
* **Declared:** [docs/benchmark/v10c-dev/PLAN.md](benchmark/v10c-dev/PLAN.md),
  [SPEC.md](benchmark/v10c-dev/SPEC.md) (`shared-prefix/v3`) and
  [API_INVENTORY.md](benchmark/v10c-dev/API_INVENTORY.md), before any
  request. 12 points, 600 requests.
* **Done:** [docs/benchmark/v10c-dev/](benchmark/v10c-dev/README.md). 600
  of 600 requests.
  * **Refused proposals:** validation produced nearly the preview's
    decisions (the same at 5 of 7 points).
  * **Valid proposals:** validation led Qwen to revise every time, while
    the preview mostly kept the proposals, including a $44,687
    credit-supported commitment.
  * **Recommendation:** release V0.10.0 as a methodology and measurement
    release, with no agent change.

**V0.10c accepted; V0.10.0 released** (owner, 2026-11-01), as a
methodology and measurement release, with no change to llm-agent 3 or 4.
The owner's verdicts:

| Component | Decision |
|---|---|
| Inference repeatability | PASS |
| Shared-prefix methodology v1–v3 | PASS |
| Prospective experiments | PASS |
| Information inventory | PASS |
| Recorded replay | PASS |
| Cross-platform reproducibility | PASS |
| Historical compatibility | PASS |
| Causal financial benefit | not established |
| Independent evidence review | PENDING |

* **The central question:** does the full preview give meaningfully
  different decision support from ordinary validation feedback? Yes in
  some contexts, but not consistently, and with no demonstrated financial
  or operational benefit.
* **Validation and preview serve different purposes:** validation
  establishes whether an action is permitted; the preview supplies
  information about its consequences. Neither establishes whether the
  action is a good management decision.
* The findings argue against automatically inserting the full preview
  into every agent decision.

No V0.10d. Each later step needs the owner's GO.

## 7e. V0.11 — agent decision workflow (released)

The owner's recommendation after V0.10.0: shift from methodology
experiments to the agent's decision workflow. Nothing here is approved
or built.

**A proposed workflow,** not yet an empirically validated improvement:
1. always validate a proposed action before submission;
2. return concise refusal feedback when validation fails;
3. offer detailed preview information selectively, for financially
   consequential valid actions;
4. never treat an accepted preview as a recommendation to proceed;
5. preserve the complete decision and feedback record.

**Directions:**
* **independent evidence review:** complete the 98-item review with a
  genuinely independent reviewer before relying on the consistency
  findings for broader model-quality claims (a priority);
* better integration of validation feedback into agents;
* financial consequence communication;
* decision-level intervention evaluation;
* broader prospective studies.

**V0.10.0 accepted; V0.11a authorized** (owner, 2026-11-02). The key
principle: validate first, preview selectively, never interpret an
accepted action as a recommended one. The independent review is to be
prepared and verified but does not block engineering.

| Milestone | Scope | Status |
|---|---|---|
| V0.11a | Validation-first agent workflow | accepted 2026-11-02 (e6bb386, CI 41448551855) |
| V0.11b | Controlled evaluation of the validation-first workflow | accepted 2026-11-02 (562ed93, 26005b3; CI 41498797098) |
| V0.11c | Multi-day evaluation, if justified | not pursued: prospective multi-day testing is proposed for V0.12, after safeguards are independently reviewed |
| V0.11.0 | Release, with the workflow selector (b44384a) | released 2026-11-02, tag `v0.11.0` |

* **V0.11a, done:** llm-agent 5 (`agents/llm/llm_agent5.py`,
  [V0.11_AGENT_WORKFLOW.md](V0.11_AGENT_WORKFLOW.md)).
  * Every proposal is validated first, and a refusal is answered with the
    engine's own results.
  * A valid staffing proposal's preview is shown only when the engine's
    answer shows a direct commitment, obligation, termination,
    displacement or capacity change, with neutral wording.
  * Only fully validated batches are submitted, and every stage is
    recorded (dreamteam.llm-decision-record/v5).
  * Tested with scripted models only; no live run.
  * The reviewer handoff is [REVIEWER_HANDOFF.md](REVIEWER_HANDOFF.md);
    the review is still PENDING.

**V0.11a accepted; V0.11b authorized** (owner, 2026-11-02). It
distinguishes three things:
* **correctness**, established by V0.11a's deterministic tests;
* **behaviour:** whether the workflow changes decisions;
* **decision quality:** whether the changes improve outcomes.

V0.11b measures behaviour. It compares three whole workflows from one
recorded proposal: llm-agent 3 (A), llm-agent 5 without the preview (B)
and llm-agent 5 (C). The eligibility rule stays unchanged.
* **Declared:** [docs/benchmark/v11b-dev/PLAN.md](benchmark/v11b-dev/PLAN.md),
  [SPEC.md](benchmark/v11b-dev/SPEC.md) (`shared-prefix/v4`), before any
  request.
* **Size:** 14 points, 720 draws.
* **Done:** [docs/benchmark/v11b-dev/](benchmark/v11b-dev/README.md). 720
  of 720 draws.
  * **The preview (C − B):** the same decisions at 10 of 14 points. At
    valid proposals it never raised a commitment, kept the $44,687.30
    credit-supported allocation, and cut one teleradiology contract.
  * **Validation-first feedback (B − A):** valid corrections at 3 of 5
    refused Qwen points where llm-agent 3 made none. Two of them
    terminate staff.
  * **Recommendation:** prepare V0.11.0. The owner decides whether B or
    C is the default.

**V0.11b accepted; conditional GO for V0.11.0** (owner, 2026-11-02):
* **The default workflow is validation-first without the preview (B).**
  B produced most of the observed validation improvement. C made more
  model calls without a consistent additional benefit, though it
  occasionally reduced commitments.
* **The selective preview (C) stays available,** opt-in.
* **llm-agent 3 and 4 remain** available and unchanged.
* **Option A, authorized:** a bounded `--workflow` selector for
  llm-agent 5 (b44384a), with its recording, replay and tests, as part of
  the release, not a separate V0.11c. `llm-agent-workflow/v1` keeps its
  identity and is the opt-in `validation-first-preview`. The new
  `llm-agent-validation-first-workflow/v1` is the default.
* **The safety concern:** validation-first turned refused actions into
  valid irreversible terminations, one displacing 100 booked
  appointments. A technically valid correction is not necessarily a
  good management decision. No approval mechanism was added before the
  release. Irreversible and high-exposure actions are disclosed in the
  agent's evidence and the release documentation, and decision
  safeguards are proposed for V0.12.

## 7f. V0.12 — decision safeguards (released)

The owner's recommendation after V0.11.0: decision safeguards,
especially for irreversible staffing decisions. Syntactic validity and
financial affordability are insufficient as management safeguards.
Nothing here is approved or built.
* **Decision safeguards for irreversible actions:** an explicit,
  engine-backed consequence acknowledgment before high-impact actions,
  such as terminations and displaced bookings. This would not be an
  automatic refusal or a new scoring rule.
* **Explicit operational exposure before commitment:** what an action
  displaces, retains or makes irreversible, from the engine's own
  answer.
* **Independent evidence review** (still PENDING).
* **Financial decision-quality evaluation.**
* **Prospective multi-day testing,** only after the safeguards are
  independently reviewed.

Each step needs the owner's GO.

**V0.11.0 accepted; V0.12a authorized** (owner, 2026-11-02), with
scripted models only. The priority is not more valid submissions but
explicit recognition of consequential actions before they are committed.
The safeguard works at the agent workflow layer and does not change the
simulator's action rules. An acknowledgment must name the specific
action and its consequences with exact engine-derived values, so that it
is auditable rather than ceremonial. Acknowledgment is not approval, and
does not establish that an action is wise.

| Milestone | Scope | Status |
|---|---|---|
| V0.12a | Engine-backed consequence acknowledgment | accepted 2026-11-02 (9871e37) |
| V0.12b Stage A | Independent safeguard audit; real-engine displacement test; F1 repair; focused re-review; real-engine hire test | accepted 2026-11-03; completed 2026-11-03 (F1 CLOSED) |
| V0.12b Stage B | Prospective live decision-level evaluation: llm-agent 5 (validation-first-preview) against llm-agent 6 | design approved 2026-11-03; driver implemented, independently reviewed and re-reviewed (GO); protocol locked at 5b8879f (`docs/benchmark/v12b-dev/PROTOCOL.md`); the single authorized run COMPLETE, 420 of 420 draws, 524 model calls (`RESULTS.md`) |
| V0.12c | Prospective multi-day agent evaluation | not started; moved to V0.13, after the safeguards are reviewed |
| V0.12.0 | Release: high-impact action acknowledgment and reproducible decision evidence | released 2026-11-04 (`v0.12.0`) |

* **V0.12a, done:** llm-agent 6 (`agents/llm/llm_agent6.py`,
  [V0.12_ACKNOWLEDGMENT.md](V0.12_ACKNOWLEDGMENT.md)), with the
  consequence inventory written first
  ([V0.12_CONSEQUENCE_INVENTORY.md](V0.12_CONSEQUENCE_INVENTORY.md)).
  * `dreamteam.action-acknowledgment/v1` covers terminations, new
    retained obligations (hires and teleradiology starts) and displaced
    bookings, with exact engine values, a proposal token, fail-closed
    withholding and a final check.
  * The record is v6, and its model-free replay re-runs the workflow.
  * Testing used scripted models and the real engine; no live model was
    run. Evidence: [benchmark/v12a-dev/](benchmark/v12a-dev/README.md).
* **The concern for V0.12b:** a model may acknowledge accurately that a
  termination displaces 100 appointments and still proceed. That is not a
  software defect. V0.12b should test whether the safeguard changes
  decisions, or merely creates a procedural acknowledgment that models
  learn to repeat.

**V0.12a accepted; V0.12b authorized in two stages** (owner, 2026-11-02).
The owner made these decisions:

| Area | Decision |
|---|---|
| Independent safeguard audit | GO |
| Real-engine displacement test | Required |
| Live decision-level experiment | Conditional GO, after the audit |
| Multi-day LLM evaluation | NO GO |
| V0.12.0 release | Not yet |

Security and correctness come before behavioural usefulness. The first
question is whether the safeguard can be bypassed, or certify incorrect
consequences. Only then does it make sense to ask whether seeing, for
example, 100 displaced appointments changes what an agent does.

* **Stage A, done:**
  * **Real-engine displacement test**
    ([benchmark/v12b-dev/DISPLACEMENT.md](benchmark/v12b-dev/DISPLACEMENT.md)).
    Terminations that displace 7 to 22 bookings are acknowledged with the
    engine's exact batch count. A missing or wrong count, or a changed
    proposal, prevents submission. Applying displaces exactly the
    acknowledged number, and replay is byte-identical.
  * **Independent audit**
    ([audits/v0.12b-safeguard-audit.md](audits/v0.12b-safeguard-audit.md)).
    The safeguard is GO, with no blocking finding. Stage B is GO with
    conditions on how its evidence is described. The two Major findings
    are documented, not fixed, under the owner's rule (fix only
    blockers):
    * F1: the engine's preview fields are trusted for presence and type;
    * F2: replay certifies consistency, not authorship.

    The response is
    [audits/v0.12b-safeguard-audit-response.md](audits/v0.12b-safeguard-audit-response.md).
    It puts two decisions to the owner before Stage B.
* **Stage B, not started:** it compares llm-agent 5's validation-first
  preview workflow against llm-agent 6 at identical states and initial
  proposals. It runs only after the owner approves a frozen protocol.
* **Not affected:** the separate frozen 98-item evidence review remains
  PENDING.

**Stage A accepted; F1 to fail closed** (owner, 2026-11-03). The owner's
decisions:
* **F2:** the current provenance is accepted. Replay plus the preserved
  `model.jsonl` is enough, with no response signing and no claim of
  authenticated model origin.
* **Independence:** the separate reviewer instance is accepted for this
  engineering gate. The 98-item human evidence review stays PENDING.
* **Live inference:** not yet authorized.

What followed:
* **F1 repair:** 2c73c3b, 0.12.0-dev.2. The contract, identities and
  token are unchanged.
* **New tests:** 18 F1 regression cases from real engine answers, and a
  real-engine hire test.
* **Focused re-review**
  ([audits/v0.12b-f1-rereview.md](audits/v0.12b-f1-rereview.md)): F1
  CLOSED, no blocking defect, GO for protocol approval.
* **The Stage B protocol, prepared and not run**
  ([benchmark/v12b-dev/PROTOCOL.md](benchmark/v12b-dev/PROTOCOL.md)):
  * llm-agent 5's validation-first preview against llm-agent 6;
  * 12 branch points selected by a deterministic rule from existing
    evidence;
  * 420 draws, at most 1,680 model calls;
  * the central distinction is acknowledged and proceeded against
    acknowledged and reconsidered.

  Two categories are unavailable in the corpus: a valid hire, and a
  termination without displacement. They are reported, not filled.

**Stage B design approved; driver GO** (owner, 2026-11-03). The owner
approved:
* the 12 branch points and the selection rule;
* 420 draws, at most 1,680 calls, sequential, a four-hour hard limit;
* the missing categories, as limitations.

Before live inference, three things are required:
* a driver;
* an independent review of its fidelity;
* a protocol lock.

A partial run must be reported as partial, never silently resumed.

* **The driver** (`analysis/acknowledgment/stage_b.py`,
  [benchmark/v12b-dev/DRIVER.md](benchmark/v12b-dev/DRIVER.md)):
  * it calls the agents' own `ask()` with the frozen settings, the
    agents' own clients and the recordings' memory;
  * the agent's first request must equal the recorded request byte for
    byte, at all 12 points;
  * per point, both arms get the same state, request and proposal;
    model calls are counted per draw, at most 2;
  * the limits are enforced before every request;
  * a stopped run is PARTIAL, never resumed or substituted
    ([benchmark/v12b-dev/PROCEDURE.md](benchmark/v12b-dev/PROCEDURE.md)).
* **Fidelity evidence** (`docs/benchmark/v12b-dev/fidelity/`): on the
  real engine, 113 scripted cases match the agents' own runs exactly, and
  every draw replays without a model. A scripted mock run of the real
  schedule stops PARTIAL and is analysed as partial.
* **Review and lock:**
  * The independent review
    ([audits/v0.12b-stage-b-driver-review.md](audits/v0.12b-stage-b-driver-review.md))
    gave GO with conditions, with no Blocker.
  * The focused re-review
    ([audits/v0.12b-stage-b-driver-rereview.md](audits/v0.12b-stage-b-driver-rereview.md))
    gave GO for the lock.
  * Every finding was answered
    ([audits/v0.12b-stage-b-driver-review-response.md](audits/v0.12b-stage-b-driver-review-response.md)).
  * The protocol was locked at 5b8879f.
* **The run** (authorized by the owner, 2026-11-03): COMPLETE, 420 of 420
  draws, 524 model calls, replayed without a model and reproduced on
  arm64, amd64 and the three CI platforms
  ([benchmark/v12b-dev/RESULTS.md](benchmark/v12b-dev/RESULTS.md)).
  * Arm B: 153 acknowledgment requests, all answered accurately and
    followed by the same action. **No action was revised or abandoned
    after an acknowledgment was shown.** 77 draws needed none.
  * Most revisions in both arms followed the engine's validation
    refusals, before any preview or acknowledgment. Every B − A
    interval contains 0.
  * At one point, 16 of 20 first acknowledgment replies were malformed
    JSON; all recovered on the retry.
* **Owner acceptance (2026-11-04):** Stage B accepted and closed. The
  mechanism works technically in the tested contexts; it does not
  establish better decisions. Here acknowledgment behaved like a
  procedural confirmation, not a decision-changing safeguard.

**V0.12.0 released** (2026-11-04, `v0.12.0`) as high-impact action
acknowledgment and reproducible decision evidence, not as an improved
decision-quality system. llm-agent 6 is an explicitly selected
experimental agent; llm-agent 5's validation-first workflow stays the
default. Release baseline:
[audits/v0.12.0-release-baseline.md](audits/v0.12.0-release-baseline.md).

## 7g. V0.13 — decision safeguards beyond acknowledgment (proposed)

The owner's direction after V0.12.0: do not build another acknowledgment
mechanism. Requiring a model to restate consequences did not make it
reconsider an action. Nothing here is approved or built.

**Priorities:**
1. **Complete the independent human evidence review** (the frozen
   98-item review, still PENDING).
2. **Decision safeguards that go beyond acknowledgment.**
3. **Explicit consequence comparison before irreversible actions.**
   Before a termination, for example, the agent would be shown the
   proposed action, keeping the current arrangement and any other valid
   alternatives, each with its engine-derived consequences. This is a new
   decision-support capability, not another confirmation step. The
   measure is whether the agent evaluates the alternatives, not whether
   it can repeat the numbers.
4. **Financial decision-quality evaluation.**
5. **Multi-day prospective agent testing,** only after the safeguards are
   reviewed.

**The owner's plan for V0.13:**

| Milestone | Scope | Status |
|---|---|---|
| V0.13a | Alternative-action inventory and design | **accepted** (0.13.0-dev.1, `1c75c93`) |
| V0.13b | Deterministic alternative-comparison implementation | **accepted** (0.13.0-dev.2, `701d83e`) |
| V0.13c | Controlled decision-level evaluation: protocol preparation | **accepted** as a protocol-design milestone (0.13.0-dev.2, `9a4808d`) |
| V0.13d | Controlled alternatives workflow and experiment preparation | **done** (0.13.0-dev.2), awaiting owner review; protocol not locked; human presentation review pending |
| V0.13e | The lock; then, only on the owner's authorization, the run | proposed |
| V0.13.0 | Release | pending |

**V0.13a (done, 0.13.0-dev.1):** design and inventory only.
* Documents: `docs/V0.13_MANAGEMENT_ALTERNATIVES.md` and `docs/v13a/`.
* No agent workflow, no model request, and no change to simulator
  mechanics, accounting, schemas or scoring.

**What V0.13a found:**
* Every action has a meaningful no-action alternative.
* A termination's direct terms are exact in the preview.
* Keeping the employee keeps the weekly guarantee and the severance
  exposure.
* The preview does not report capacity lost by a termination-only batch
  (G1).
* Exact branching is possible by rebuilding from the record, not by
  copying in memory. A mechanics check verified it.
* Keyed randomness gives paired branches the same exogenous world.
* Projections use the realized future. They belong to evaluation, not
  to the agent's presentation.

**V0.13a's recommendation for V0.13b (accepted by the owner, with E4
narrowly approved):**
* Deterministic work only:
  * the branch builder, branch identities and comparison reader (E1–E3,
    read-only, in `internal/session`);
  * the comparison contract;
  * the candidate rule;
  * a fixed, hashed presentation template;
  * paired scripted cases.
* No live inference.
* E4, an opt-in `slotsClosed` for termination-only previews, needs the
  owner's decision.

**V0.13b (done, 0.13.0-dev.2):** deterministic implementation; no live
inference, no new agent workflow, and no change to simulator mechanics,
accounting, schemas or scoring.
* Documents: `docs/v13b/` and the V0.13b section of
  `docs/V0.13_MANAGEMENT_ALTERNATIVES.md`.
* **E1, E2:** read-only branches rebuilt from a saved record, with
  verified identities (`internal/session/branch.go`).
* **E4:** an opt-in, separate slots-closed report for termination-only
  batches; the default preview is unchanged (`internal/sim/slotsclosed.go`).
* **E3, the comparison:** `dreamteam.management-alternatives/v1`, with
  the agent and evaluation audiences separated in code; the candidate
  rule; the fixed, pinned template; and the classification of scripted
  replies (`internal/alternatives/`).
* **Evidence:** the termination case and scripted decision-sensitivity
  pairs from real engine states (`docs/benchmark/v13b-dev/`).

**V0.13c (done, 0.13.0-dev.2):** protocol preparation; no live
inference.
* Documents: `docs/v13c/` and the V0.13c section of
  `docs/V0.13_MANAGEMENT_ALTERNATIVES.md`.
* **F4:** `dreamteam.alternative-selection/v2`, which removes duplicate
  batches; v1 is unchanged.
* **The arms:** control and treatment turns built by the engine from one
  test, with the information boundary in code (`internal/alternatives/arms.go`).
* **Branch points and sensitivity pairs:** the V0.12b pool on the engine,
  rebuilt in CI; six pairs of real engine states
  (`docs/benchmark/v13c-dev/`).
* **Reviews:** the independent technical review and its response; the
  human presentation review is pending.
* **The draft protocol:** not locked. It waits on the human review, the
  owner's decisions D1–D10 and V0.13d's workflow and driver.

**V0.13d (done, 0.13.0-dev.2):** the controlled alternatives workflow and
experiment preparation; no live inference.
* Documents: `docs/v13d/` and the V0.13d section of
  `docs/V0.13_MANAGEMENT_ALTERNATIVES.md`.
* **The owner's decisions:** D7 neutral identifiers and a typed
  presentation; D8 symmetric wording; D9 the estimand; D10 operations
  batches preserved.
* **Built:** the typed view and presentation v2; the turn and the reply as
  opt-in read-only protocol operations; the workflow; the driver, its
  scripted fidelity evidence and the revised draft protocol (262 draws).
* **Pending:** the human presentation review; the owner's approval of the
  budget; then a lock (V0.13e). A run needs the owner's authorization.

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


## V0.13e: presentation review preparation (0.13.0-dev.2, engine unchanged)

HUMAN REVIEW PENDING. No reviewer, identity or findings exist. No remediation
or re-review occurs until a human review exists. The rendered, checksum-pinned
[review package](v13e/review-package/README.md) includes both arms and all
inspect details, with record-only headers separated. The owner accepted
V0.13d and approved F7, F3, N1 and the preserved budget. Engineering boundary
tests supplement the existing implementation without changing it.

LOCK CANDIDATE - NOT LOCKED - awaiting human review

[Candidate identities and verifier](v13e/PROTOCOL_LOCK_CANDIDATE.md) preserve
262 scheduled draws, 802 max calls, four hours, nine historical contexts,
six sensitivity pairs and two arms. Actual submitted batches determine the
primary final action; delivered-detail citations are secondary. Pairs do not
isolate single facts. A stale descriptive field in frozen evidence is
[recorded separately](v13e/REVIEW_RESPONSE.md), not silently corrected.

Engine version remains 0.13.0-dev.2; earlier evidence stays byte-identical.
Live inference NO GO, requiring separate owner authorization. No protocol-lock
commit, tag or release. See [run procedure](v13e/RUN_PROCEDURE.md),
[partial-run procedure](v13e/PARTIAL_RUN_PROCEDURE.md), and
[operational incident report](audits/v0.13e-operational-incident.md).
