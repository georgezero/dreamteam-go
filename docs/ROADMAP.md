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
| **V0.2** | Operations control: interactive scheduling & staffing, disruptions, save/load, agent-interface skeleton | V0.1 |
| **V0.3** | Financial management: claims workflow, denials, collections, payroll cycles, monthly close | V0.1 (V0.2 for save) |
| **V0.4** | Growth: hiring, equipment, extended hours, referrals & reputation, New Practice scenario, goals | V0.2, V0.3 |
| **V0.5** | WebUI on the same engine + benchmark framework (scenario packs, baselines, scoring) | V0.2 (API), V0.4 for growth benchmarks |
| **V0.6** | AI-managed practice: LLM agent harness, human-vs-AI, safety evaluation, automation levels | V0.5 |

Change from the brief's sequence: a **minimal observation/action JSON loop and
scripted-policy runner** moves forward into V0.2. It is small, it keeps the
action API honest from the start, and it lets us run baseline policies against
balance changes long before agents exist. The full benchmark framework stays
in V0.5.

```
V0.1 ──▶ V0.2 ──┬──▶ V0.4 ──▶ V0.5 ──▶ V0.6
   └──▶ V0.3 ───┘
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
| 16 | TUI polish: responsive layouts (all breakpoints), help, filters, theme / `NO_COLOR` | all screens at every size | **golden screen snapshots at 80×24, 120×36, 160×48** (layout now stable) |
| 17 | End-to-end: TUI action → run record → headless replay | — | identical digests TUI vs headless |
| 18 | Calibration pass for `moderate` (per weekday type); preset bands; `docs/ASSUMPTIONS.md`; **P&L calibration workbook** (§10) | — | calibration suite (build tag); engine T30 margin within ±2 pp of the workbook for the same inputs |
| 19 | Performance pass on the Pi (build, 90-day run, TUI CPU, Max-speed responsiveness) | — | acceptance criterion 16 |
| 20 | README, docs sync, change notes | — | — |

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

### Deferred out of V0.1 (explicit)

Save/load UI · stochastic equipment failures · staff absences · front-desk phone
and scheduling workload · template editing · rescheduling actions · overbooking
& reserved add-on slots · prep failures, incomplete exams, repeats · protocoling ·
multi-exam visits · claim rework, denial management · payroll cycle UI · hiring,
equipment purchase, extended hours · referrals dynamics, reputation, competition ·
New Practice scenario · mouse · WebUI · SQLite · agents.

---

## 3. V0.2 — Operations control

Deliverables

* Actions: `reassign_technologist`, `call_in_staff`, `send_home`,
  `approve_overtime_week`, `change_roster`,
  `edit_template`, `reserve_addon_slots`, `set_overbooking`, `set_late_policy`,
  `reschedule_appointment`, `offer_earlier_slot`, `schedule_maintenance`.
* Weekly staff roster editor (shifts, Saturday rotation) with coverage preview
  and labor cost estimate.
* Template editor (per room, per weekday) with expected-capacity preview.
* Disruptions: stochastic equipment failures with noisy vendor ETA, staff
  absences (call-outs at 06:00–07:30), late-running emergencies.
* Front desk scheduling workload (phone requests consume desk time; abandonment).
* Protocoling task for CT/MR add-ons (radiologist task with priority).
* Save/load: versioned JSON snapshots (contains hidden state; game-only).
* Scripted scenarios: "CT tech calls out 07:30", "MR down at 13:10".
* Agent interface skeleton: `dreamteam agent --stdio` (JSONL observation →
  action list → advance to next decision epoch); a scripted `greedy` policy
  in Go using the same API.
* Mouse support in TUI.

Acceptance

* Every action validated in `sim`, logged, replayable; rejected actions logged.
* Save → load → continue gives the same digest as uninterrupted play.
* The "CT tech sick" scenario is solvable via reassignment/call-in, and
  `greedy` beats `do-nothing` on waits in ≥ 80% of 50 seeds.
* Agent stdio loop runs 30 days headless with the scripted policy; observation
  JSON validates against the published JSON Schema.

## 4. V0.3 — Financial management

Deliverables

* Claims workflow: charge lag, submission, adjudication, partial payments,
  denials with reason categories, rework queue and policy, write-offs.
* Self-pay policies (estimate/collect at time of service).
* Payroll cycles (biweekly), payables calendar, monthly close, monthly P&L,
  balance sheet, cash-flow statement.
* KPIs: days in A/R, denial rate, net collection rate, cost per exam,
  contribution margin per modality/exam.
* Finance screen: statements, A/R aging, payer view, cash forecast from
  *known* obligations (no peeking at latent payment dates).
* Accounting review with the **Finance** plugin (§10): chart of accounts, accrual
  and close procedure, statement layout, reconciliation checks.

Acceptance

* Statements reconcile (balance sheet balances; cash-flow ties to cash).
* Preset calibration bands hold over 6 simulated months.
* Contribution-margin view makes the MR evening-capacity opportunity
  discoverable without hidden data.

## 5. V0.4 — Growth and progression

Deliverables

* Hiring (with recruiting lead time, onboarding productivity ramp), termination
  with notice, raises; workload-driven turnover risk.
* Equipment: lease/purchase scanners or a second US room, construction lead
  time, financing (loan/lease), service contracts.
* Extended hours / Sunday opening.
* Referral sources with dynamic relationship strength driven by TAT, access
  (lead time), patient experience, and outreach spending (demand thinning keeps
  common random numbers).
* Competition events (a competitor opens, payer rate cut).
* **New Practice** scenario (low demand, thin cash, ramping referrals, no
  warm-up history).
* Goals and progression: scenario objectives (e.g., "reach 12% margin for a
  quarter with no urgent TAT breaches"), monthly review screen, practice
  valuation proxy with quality gates.
* Investment-case review with the **financial-analysis** plugin (§10):
  valuation proxy (EBITDA multiple vs DCF), lease-vs-buy for scanners, payback of
  expansion options, financing terms.

Acceptance

* New Practice is winnable (scripted heuristic reaches profitability within
  18 simulated months on ≥ 70% of seeds) and losable.
* Growth exploits from PLAN §3 are covered by tests (e.g., unchecked overbooking
  degrades referrals and revenue over a quarter).

## 6. V0.5 — WebUI and benchmark framework

Deliverables

* WebUI (templ + Datastar or equivalent, decided then) using `session` and
  `api` only; feature parity with TUI dashboard, schedule, worklist, finance.
* SQLite store (pure-Go driver) for saves index, run records, benchmark results,
  long event logs.
* Benchmark packs: scenario + seeds (public dev seeds, 256-bit secret test
  seeds) + horizon + evaluation tail + decision epochs + scripted perturbations +
  metric spec + constraints. Split into a public part and a private part
  (seeds, perturbation timing). Agents run as isolated processes (PLAN §9.7).
* Baselines: do-nothing, rules, greedy, random-valid, simple look-ahead
  optimizer (optionally a MIP via an external solver process, decided then).
* Scoring report: metric vectors with confidence intervals across seeds,
  constraint violations, rejected actions; optional composite score.

Acceptance

* Two baselines × 100 seeds × 30 days run in < 5 min on the Pi.
* Results reproducible from stored run records.
* WebUI and TUI produce identical digests for the same action log.

## 7. V0.6 — AI-managed practice

Deliverables

* LLM agent harness over the stdio/HTTP agent API (provider-agnostic; budget
  and token accounting; tool-call style actions).
* Human-vs-agent leaderboard on benchmark packs; human play recorded through
  the same decision-epoch protocol.
* Safety evaluation: hard-constraint attempt rate, urgent-case handling,
  workforce welfare metrics.
* Automation levels as a game mechanic: manual → automated scheduling &
  billing → AI operations manager; compare outcomes.

Acceptance

* An LLM agent completes a 30-day pack without harness errors; all actions
  replay; no hidden-state access paths exist (audited).

---

## 8. Testing strategy (all versions)

| Layer | What | When |
|---|---|---|
| Unit | `simtime`, `rng` samplers, queue, ledger, policies | every commit |
| Table tests | action validation, state transitions | every commit |
| Property | chunking invariance; invariants over random seeds; conservation; replay equality | every commit (short), nightly (long) |
| Golden | start-state digest, 7-day and 90-day digests per preset; CLI output; TUI views | every commit; regenerate only with intentional gameplay changes |
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
| V0.3 | Anthropic **Finance** plugin (journal entries, reconciliation, statements, variance analysis, month-end close) | review ledger design, close procedure, statement reconciliation | review notes in a turn report; fixes in `sim` |
| V0.4 | Anthropic **financial-analysis** plugin (DCF, three-statement models) | valuation proxy, lease-vs-buy, expansion payback | investment-case notes; parameters in scenario JSON |

Limits: these plugins are general corporate finance. Imaging-specific numbers
(reimbursement, denial rates, days in A/R, staffing ratios, radiologist
compensation) need healthcare sources such as MGMA, HFMA and CMS fee schedules.
Those sources and the label for each number go in `docs/ASSUMPTIONS.md`.
Install each plugin only when its version starts.
