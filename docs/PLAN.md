# DreamTeam — Project Plan

Status: **pre-implementation decisions settled (2026-10-13); do not implement until the plan is approved.**
Date: 2026-10-13

DreamTeam is a realistic outpatient radiology practice management simulator and
game. It's built on a deterministic simulation engine that will later serve as an
AI-agent benchmark.

| Document | Contents |
|---|---|
| **PLAN.md** (this) | vision, critique of the brief, key decisions, scenarios, AI/benchmark readiness, repository layout, open questions, summary |
| [SIMULATION_DESIGN.md](SIMULATION_DESIGN.md) | domain model, engine, randomness, patients, workforce, finance, observations/actions, invariants, metrics |
| [TUI_DESIGN.md](TUI_DESIGN.md) | screens, ASCII mockups, controls, pacing, layouts |
| [ROADMAP.md](ROADMAP.md) | V0.1–V0.6, implementation sequence, acceptance criteria, testing, risks |

All numbers in these documents are **illustrative assumptions or calibration
targets**, not validated operational data. Patients, staff, referrers and
finances are synthetic. DreamTeam simulates *operations*. It makes no clinical
decisions and generates no clinical content.

---

## 1. Product vision

### 1.1 One engine, three uses

```
                  ┌──────────────────────────────────────────┐
                  │   DreamTeam engine (deterministic DES)   │
                  │ patients · staff · rooms · reading · $   │
                  └───────────────┬──────────────────────────┘
                                  │  api.Observation / api.Action
          ┌───────────────────────┼─────────────────────────┐
          ▼                       ▼                         ▼
   Game (TUI, later Web)   Research (CLI runs,       Benchmark (agents,
   humans manage a         seeds, policies,          baselines, held-out
   practice                metrics, replays)         seeds, scoring)
```

The game, the research simulator and the benchmark are the same engine
behind the same action interface. The game adds presentation, goals and pacing.
Research adds batch runs. The benchmark adds fixed scenarios, held-out seeds and
scoring. Nothing in the engine knows which of the three is driving it.

### 1.2 Core loop

```
 Plan ─────────▶ Operate ─────────▶ Respond ─────────▶ Review & grow
 rosters,        the day runs        delays, walk-ins,  daily summary,
 templates,      under policies;     urgent studies,    monthly P&L,
 policies        you watch flow      gaps, downtime     hire / buy / expand
   ▲                                                         │
   └─────────────────────────────────────────────────────────┘
```

The day runs by itself. The player sets policies and steps in when something
goes wrong. Days pass quickly once the player trusts the policies. Growth
decisions happen monthly.

### 1.3 Player objectives

* **Primary: financial performance.** Sustained operating profit and cash
  health, and in the long run growth in practice value.
* **Constraints and secondary objectives:** patient access (lead time, lost
  demand), patient experience (waits), clinical service (report turnaround,
  urgent handling), workforce (overtime, breaks, turnover), safety (hard rules).

Financial score only counts while quality gates hold (§11). This is the main
defence against "profitable but terrible" strategies.

### 1.4 Progression

1. **Stabilize** (days 1–30): fix the staffing/template mismatch, the US
   bottleneck, and the evening dead zones. Get the backlog under control.
2. **Optimize** (months 2–6): templates, overbooking, reserved add-on capacity,
   denial management, payer mix awareness.
3. **Grow** (months 6+): hire, extend hours (MR evenings), add a US room, invest
   in referral relationships. **[later]** Open a second site.
4. **Automate** **[V0.6]**: introduce automated scheduling, billing and AI
   operations management, then compare.

Progression is unlocked by money and capability, not by experience points.

### 1.5 What makes it engaging

* **Visible flow**: you watch patients and studies move, and you see a
  bottleneck form before the numbers show it.
* **Coupled tradeoffs with delays**: more CT bookings raise revenue now, push the
  reading backlog up next week, and erode referrals next quarter.
* **Cash vs profit**: a profitable practice can still run out of cash.
  Collections lag behind services by weeks.
* **Crises** **[V0.2]**: the CT tech calls out at 07:30 or the MR goes down at
  13:10, and the player has to decide quickly.
* **Explanations**: daily insights point at the levers without giving the
  answer.
* **Fair randomness**: replays and common random numbers make "what if I had…"
  comparisons meaningful.

---

## 2. Critical evaluation of the brief

The brief is a strong foundation. Several of its assumptions are unrealistic or
incomplete. Each item below says what is wrong, why it matters, and what the
plan does about it.

### 2.1 Unrealistic or underspecified assumptions

| # | Assumption | Problem | Resolution |
|---|---|---|---|
| 1 | Four technologists, one per modality, cover 68 h/week | Full coverage needs 272 room-hours/week. Four techs on 8.5 h shifts with unpaid lunch give 160 available hours, a 41% shortfall. Also, credentials don't transfer freely: sonographers (US) and MR techs are separate specialties, and XR↔CT cross-training is the common case | Keep the roster but model it honestly: shared XR/CT tech, dark MR evenings, US 09:30–18:00. Coverage gaps become the first challenge. See SIMULATION_DESIGN §6.2–6.3 |
| 2 | One front desk covers registration | One person can't cover 68 h, and phones, scheduling and check-out compete for the same time | Model desk coverage. The ops manager covers some hours, slowly. After that, techs self-register, which costs scan time. Phone and scheduling workload is V0.2 |
| 3 | Two radiologists working this one center full-time | ~70–85 studies/day is about 8 reading-hours. Two full-time radiologists would sit half idle, and their salaries would make the practice unprofitable. Real centers share radiologists across sites or use teleradiology | Radiologists have an explicit **site allocation** (50–60%). Only that share of cost and time is charged to the practice. Reading is remote-capable and not tied to room hours |
| 4 | Revenue on exam completion | You can't bill until the report is signed. Collection takes weeks and some claims are denied | Recognize revenue at report final, at expected net. Cash comes from claim payments with lag and outcome. SIMULATION_DESIGN §8 |
| 5 | Walk-ins "may require CT or ultrasound" | Walk-in CT/MR is rare in outpatient practice: orders, authorization, MR safety screening and protocoling get in the way. Walk-ins are overwhelmingly XR | XR-dominated walk-ins. CT/US add-ons arrive mainly as urgent same-day orders. MR walk-ins are ~0 |
| 6 | Contrast exams anytime | Contrast administration requires physician supervision. Evening and Saturday contrast exams depend on a radiologist being on duty, or on virtual supervision where allowed | Hard constraint: contrast needs a supervising radiologist on duty and on site. Configurable `Virtual` mode in V0.2 |
| 7 | Exam duration = slot | Room occupancy, tech workload and patient time differ: changing, MR safety screening, IV starts, documentation, room turnover | Separate room time, tech extra time and slot length. SIMULATION_DESIGN §2.4, §5.10 |
| 8 | Daily volumes in the brief (XR 20–35, US 10–18, CT 15–25, MR 8–14) | US 10–18 at 30–45 min doesn't fit inside an 8.5 h US shift. MR 8–14 doesn't fit "61% utilization" unless utilization is measured against open hours | Report utilization two ways, against open hours and against staffed hours. Calibrate volumes to staffed capacity |
| 9 | Prior authorization deferred | Missing authorizations are a major real cause of CT/MR cancellations | Keep it deferred, but bake its effect into the cancellation rates. Explicit auth workflow comes later |
| 10 | Saturday hours with a weekday roster | Saturday staff need a weekday off (Tue–Sat rotation), which thins Mondays | Roster includes Tue–Sat rotations. Monday is visibly thinner |
| 11 | "Established practice with historical performance" | Hand-written history can contradict the engine and cause a visible discontinuity on day 1 | History comes from a **warm-up run** of the same engine (180 days ≥ the longest booking lead time), so schedule, backlog, A/R and trends are all consistent |
| 12 | Both pacing modes "produce identical outcomes" | True only for the same *timestamped* actions. A human acting at different moments gets different outcomes, as they should | Define determinism as scenario + seed + timestamped action log. Guarantee frame-rate and speed independence (chunking invariance) |

### 2.2 Missing operational constraints (and when they arrive)

| Constraint | Version |
|---|---|
| Staff breaks/lunch, finish-in-progress, start-if-it-fits rule | V0.1 |
| Contrast supervision | V0.1 |
| Desk coverage and self-registration penalty | V0.1 |
| Daily QA / warm-up before first patient (CT, MR) | V0.2 |
| Front desk phone/scheduling workload; abandoned calls | V0.2 |
| Radiologist protocoling of CT/MR before scan | V0.2 |
| Prep failures (fasting, labs), incomplete MR (claustrophobia), repeats | V0.2 |
| Staff absences, equipment failures | V0.2 |
| Maximum consecutive hours, weekly hours, fatigue → turnover | V0.1 (hard cap) / V0.4 (turnover) |
| Prior authorization, insurance eligibility | later (≥ V0.4) |
| Critical results communication | later |
| Patient mobility/transport, interpreters, pediatrics | later |

### 2.3 Ways V0.1 is simplified without losing the long-term design

* One exam per order and one order per visit. Entities are still separate, so
  multi-exam visits slot in later.
* Centralized automatic booking, with no scheduler labor (V0.2 adds it).
* Single policy per area with enum switches, no plugin interfaces.
* Scripted downtime only.
* Global billing and a single-expectation revenue model, with real claim timing.
* No persistence beyond run records (JSON). Snapshots exist only in tests.
* Just three player actions, all on the real action path, so replay, validation
  and logging are proven from day one.

---

## 3. Exploits and incentive problems

A financial objective rewards degenerate strategies unless the model closes
them off. Each exploit below becomes a test case.

| Exploit | Why it pays (naively) | Closure |
|---|---|---|
| Book only MR/CT, starve XR/US | Higher revenue per minute | The practice can't choose its orders; referrers send mixed demand. Turning away exams increases lost demand, and in V0.4 lost demand erodes referral relationships across all modalities |
| Overbook aggressively | More exams per day | Waits, late starts, overtime and patient experience suffer. V0.4 reputation feeds back into demand. Quality gates |
| Read only short studies (`ShortestFirst`) | Inflates the count of finalized reports | TAT is measured over *all* acquired studies, with unread ones censored at their age, plus cohort breach shares per modality. Revenue is recognized per study regardless of order, so there's no revenue gain |
| Push work past the benchmark horizon (leave studies unread, reschedule patients beyond the end) | Hidden waits don't count | Evaluation tail: default policy runs N days after the horizon and its outcomes count. Unserved orders and unread studies are scored at their age. Access is measured from the original order |
| Sacrifice a 10% tail to keep p90 low | p90 ignores the worst 10% | Cohort metrics (share breaching target) alongside percentiles |
| Leave reports unread to save cost | Radiologists are salaried, so it saves nothing | Revenue only arrives at finalization, so delays defer revenue and cash. TAT metrics |
| Unlimited overtime | Capacity without hiring | 1.5× cost, 12 h hard cap, skipped breaks and turnover risk (V0.4) |
| Run unstaffed rooms / schedule outside hours | Phantom capacity | Hard constraints. Validation rejects it |
| Cancel low-paying (self-pay/Medicaid) patients | Better payer mix | Practices can't unilaterally cancel booked patients without consequences. The action surface doesn't allow payer-based cancellation. **Open question Q6** covers payer visibility |
| Exploit determinism (save-scum, seed knowledge, seed recovery from observed history) | Knowing the future | Fine in the game. In the benchmark: 256-bit secret test seeds, SHA-256-keyed ChaCha8 streams (seed recovery infeasible), isolated agent processes, no snapshots or records, observations without latents |
| Read the future from IDs or pack names | Leaks undisclosed demand or perturbations | Public IDs allocated at disclosure. Neutral pack names. Perturbation details live in the private part of the pack |
| Step-to-next-internal-event to detect hidden events | Leaks timing | `StepVisible` only stops on observable entries |
| Defer costs by timing | Costs are accrued, not just paid | Accrual accounting. Benchmarks report profit and cash |
| Upcode / order more imaging | Revenue | Orders are exogenous and come from referrers. The practice can't create or change orders (operational AI, not clinical) |
| Starve maintenance **[V0.2]** | Saves downtime now | Failure hazard increases with deferred maintenance |

---

## 4. Key design decisions

Each decision lists the recommendation, the alternatives and the tradeoff.

| # | Decision | Recommendation | Alternatives | Tradeoff |
|---|---|---|---|---|
| D1 | Simulation paradigm | **Discrete-event**, priority queue, integer seconds | fixed time step | DES is exact, fast in fast-forward and free of frame-rate artifacts, but slightly harder to write. Time-stepping makes speed affect outcomes |
| D2 | Entity granularity | **Individual** patients, orders, appointments, visits, studies, claims, employees | aggregate flows | Richer behavior and per-patient inspection at modest cost (~1.5k events/day) |
| D3 | Randomness | **SHA-256-keyed ChaCha8 streams**, one per attribute, keyed by origin keys; integer samplers | one global PCG | Common random numbers, cross-arch determinism, and seed secrecy for benchmarks. About 1 µs per draw, which is negligible |
| D4 | Numbers in engine | **Integers only** (s, ¢, ppm, bp) | float64 | Avoids FMA/arch divergence. Display converts |
| D5 | Duration distributions | **Quantile tables** | lognormal/gamma params | Integer-exact, and the format real RIS data can populate directly. Less elegant parameterization |
| D6 | Event representation | **Plain data structs**, lazy cancellation | closures | Serializable and replayable. Needs a switch-based dispatcher |
| D7 | Dispatch | **Greedy, once per timestamp batch**, fixed entity order | optimization each step | Realistic, cheap and a good baseline. Smarter policies come later as alternatives |
| D8 | Hidden state | **Unexported latents** in `sim`; `api` boundary; TUI may not import `sim` | convention only | The compiler enforces it. Projection code has to be written |
| D9 | Concurrency | **Single-threaded engine**, advanced inside the Bubble Tea `Update` | engine goroutine | No races, exact pausing. Max speed bounded per frame (fine at this scale) |
| D10 | Booking | **Policy-driven automatic** booking with player-tunable templates/policies | player books each patient | Matches "policies, not micromanagement". Manual reschedule is V0.2 |
| D11 | Revenue recognition | **At report final, expected net**; cash on claim payment | cash at exam end | Realistic and closes exploits. Needs a ledger |
| D12 | Accounting | **Minimal double-entry ledger** | running totals | Free invariants and an easy path to V0.3 statements |
| D13 | Established history | **Warm-up run** of the engine (180 days, convergence-tested) | synthetic history | Self-consistent. Costs ~1 s at build |
| D14 | Radiologist model | **Employed, global billing, site allocation %** | per-read teleradiology contract | Keeps a staff-management game. Per-read pricing can be added in V0.4 as an option |
| D15 | Persistence | **JSON** run records (V0.1), JSON snapshots (V0.2), **SQLite** (pure Go) in V0.5 for results/indexes | SQLite from day one | No dependency until it's needed. JSON is diff-friendly |
| D16 | Packages | Few packages: `simtime`, `rng`, `api`, `sim`, `scenario`, `session`, `tui` | many fine-grained packages | Clear boundaries where they matter (hidden state, UI), with no premature splits elsewhere |
| D17 | Speed units | **Sim time per real second** (1 min/s … Max) | "10×" | Unambiguous. 1× real time is useless for a 12 h day |
| D18 | Exam codes | **Internal codes** | CPT | Avoids licensing. Can be mapped later |
| D19 | Time zone | **Practice-local, no DST** | real tz | Removes calendar bugs that add no gameplay value |
| D20 | V0.1 interaction | **3 real actions** + time controls + inspection | view-only V0.1 | Without some interaction V0.1 is a screensaver. Three actions prove the action path end-to-end |

---

## 5. Architecture overview

```
cmd/dreamteam ──▶ tui ───────┐
      │                       ├──▶ session ──▶ sim ──▶ rng, simtime
      └──▶ (run/replay CLI) ──┘        │         │
                                       │         └──▶ api (contract types)
                                       └──▶ scenario ──▶ sim
later:  web ──▶ session        agent/bench ──▶ session        store(SQLite) ◀── session
```

* `sim` owns truth. `api` is the only shape the outside world sees.
* `session` wraps an engine with pacing primitives, the action log, record and
  replay, and digests. Every frontend (TUI, CLI, Web, agents) uses `session`.
* Details: SIMULATION_DESIGN §3.

---

## 6. Domain model (summary)

Order → Appointment → Visit → Study → Report → Claim → Payment, plus Practice,
Room/Equipment, Employee/Shift, ReferralSource, Ledger, Scenario, Event and
LogEntry. Full fields and state machines: SIMULATION_DESIGN §2.

---

## 7. Patients, workforce and finance (summary)

* **Patients**: exogenous Poisson demand by source, hour, weekday and season.
  Lead-time mix from same-day to 6 months. Automatic `FirstFit` booking against
  templates. Lost demand when the wait exceeds tolerance. Latent no-show,
  lateness and cancellation. XR-heavy walk-ins with patience. Details:
  SIMULATION_DESIGN §5.
* **Workforce**: credentials, daily shifts from rosters, breaks, finish-in-progress,
  start-if-it-fits, overtime ×1.5 beyond 40 h, 12 h cap, contrast supervision,
  desk coverage chain. Details: §6.
* **Finance**: global billing; revenue at report final at expected net; charge
  lag; payer-class adjudication with denial and lag; accrued expenses; cash on
  payments; profitability presets as calibration bands. Details: §8.

---

## 8. Scenario system

### 8.1 Format

Scenarios are JSON files embedded in the binary (`scenarios/` with `go:embed`).
User files load with `--scenario path.json`. All quantities are integers with
unit-suffixed keys, so a reader never has to guess units.

```json
{
  "schema": "dreamteam.scenario/v1",
  "id": "established-moderate",
  "name": "DreamTeam Imaging Center — Established",
  "family": "established",
  "extends": "base/established.json",
  "preset": "moderate",
  "startDate": "2026-10-13",
  "startTime": "06:00",
  "warmupDays": 180,
  "openingCashCents": 15000000,
  "hours": {
    "Mon": ["08:00", "20:00"], "Tue": ["08:00", "20:00"], "Wed": ["08:00", "20:00"],
    "Thu": ["08:00", "20:00"], "Fri": ["08:00", "20:00"], "Sat": ["10:00", "18:00"],
    "Sun": null
  },
  "rooms": [
    {"id": "R1", "modality": "XR", "name": "X-ray 1"},
    {"id": "R2", "modality": "US", "name": "Ultrasound 1"},
    {"id": "R3", "modality": "CT", "name": "CT 1"},
    {"id": "R4", "modality": "MR", "name": "MRI 1"}
  ],
  "catalog": [
    {"code": "CT_ABD_PEL_WC", "modality": "CT", "slotMin": 30, "contrast": true,
     "roomTimeSec": {"p0": 1080, "p10": 1260, "p50": 1560, "p90": 1920, "p100": 2400},
     "techExtraSec": {"p0": 60, "p50": 180, "p100": 420},
     "readSec": {"p0": 480, "p50": 840, "p90": 1140, "p100": 1500},
     "baseRateCents": 26000, "supplyCostCents": 5000, "walkInEligible": false}
  ],
  "employees": [
    {"id": "E03", "name": "Riley", "role": "technologist", "skills": ["XR", "CT"],
     "pay": {"type": "hourly", "rateCents": 3600},
     "roster": {"Tue": {"start": "07:45", "end": "16:15",
                        "breaks": [["11:30", "12:00"]], "assignment": "R1",
                        "covers": [{"room": "R3", "until": "11:30"}]}}}
  ],
  "templates": {"R4": {"Tue": [{"start": "08:00", "lenMin": 45, "category": "MR_45"}]}},
  "referralSources": [
    {"id": "SRC1", "name": "Northside Primary Care", "kind": "primary_care",
     "baseDailyMilliOrders": 21000, "urgentBP": 300,
     "examMix": {"XR_CHEST_2V": 30, "US_ABD_COMPLETE": 10, "CT_HEAD_WO": 5}}
  ],
  "demand": {"hourProfile": [], "weekdayFactorBP": {}, "seasonFactorBP": [],
             "leadTimeMix": [], "walkIns": {}},
  "behavior": {"noShowPPM": {}, "arrivalOffset": {}, "cancellation": {},
               "walkInPatienceSec": {}},
  "finance": {"payers": [], "fixedMonthlyCents": {}, "benefitsLoadBP": 2500,
              "billingFeeBP": 500, "chargeLagBusinessDays": [1, 3]},
  "policies": {"walkIn": "accept_if_gap", "worklist": "urgent_then_fifo",
               "late": "fit_in", "graceMin": 15, "overtimeAllowanceMin": 15,
               "maxShiftHours": 12, "contrastSupervision": "on_site"},
  "scriptedEvents": [
    {"at": {"day": 1, "time": "13:10"}, "type": "room_down", "room": "R3",
     "durationMin": 150, "enabled": false}
  ],
  "objectives": []
}
```

* **`extends`**: a scenario can override a base file. Presets are small
  override files (`presets/struggling.json`, …) merged by key path.
* **Synthetic names** come from bundled fictional lists. Display names are cosmetic.

### 8.2 Scenario families

* **Established practice** (default, V0.1): 180-day warm-up under default
  policies, start Tuesday 06:00, populated 4-month forward schedule, backlog
  and A/R. Presets: `struggling`, `break-even`, `moderate` (default), `high`.
  Calibration bands: SIMULATION_DESIGN §8.6.
* **New practice** (V0.4): no warm-up history. Opened 2 weeks ago. Referral
  relationships weak (demand 35–50% of established). Cash runway ~4–6 months
  of fixed costs. Equipment leased. Objective: reach sustainable profitability.
  Needs referral dynamics, so it waits for V0.4.
* **Benchmark scenarios** (V0.2+): an established/new base plus scripted
  perturbations, horizons, decision epochs and objectives (§9.4).

### 8.3 Validation

`dreamteam scenario validate` and every load check:

* Schema version and required fields; units non-negative; quantile tables
  monotonic with p0 ≤ p50 ≤ p100.
* References resolve (rooms, employees, exam codes, sources).
* Hours well-formed. Shifts inside a day. Breaks inside shifts. Cover windows
  inside shifts.
* Skills cover assignments (a tech assigned to MR must hold MR).
* Template slots inside open hours. Categories exist in the catalog.
* Payer mix sums to 10,000 bp. Exam mixes are non-empty.
* **Coverage report** (warning, not error): staffed vs open hours per room,
  desk coverage. This shows the player the gaps.
* Feasibility warnings: template slots during unstaffed hours, contrast slots
  without a radiologist on duty.

Errors name the JSON path (`employees[2].roster.Tue.breaks[0]`).

---

## 9. AI and benchmark readiness

Interfaces are designed now. Implementation is staged (skeleton V0.2,
framework V0.5, LLM V0.6).

### 9.1 Observation and action schemas

Defined in SIMULATION_DESIGN §9.4–9.5 (`dreamteam.observation/v1`,
action catalog by version). A JSON Schema for both is generated from the `api`
types and published under `docs/schema/` in V0.2.

### 9.2 Agent interaction loop

```
reset(benchmark, seed) → obs₀
loop:
    actions = agent.decide(obs, permittedActions)
    results = [session.Apply(a) for a in actions]      # validated, logged
    obs = session.AdvanceToNextDecisionEpoch()          # sim time passes
until horizon reached → metrics report
```

* **Decision epochs**: scheduled (06:00 daily, plus every 30 sim-minutes during
  open hours by default) and event-triggered by observable alerts (urgent
  order, room down, absence). The benchmark pack sets the epoch rule.
* **Time freezes while the agent thinks.** Wall-clock latency never matters.
  Human benchmark players use the same epoch protocol, so the comparison is fair.
* **`permittedActions`**: action types available in this pack, with parameter
  domains (e.g., employees who can be extended today). This doesn't hint at
  outcomes.
* Transport: in-process Go interface for scripted policies; JSONL over stdio
  (V0.2); HTTP (V0.5).

### 9.3 Baseline policies

| Baseline | Purpose |
|---|---|
| `do-nothing` | default policies only. The floor |
| `random-valid` | sanity check that scoring isn't trivially gamed |
| `rules` | hand-written ops-manager heuristics (extend the shift of the room most behind; reassign the cross-trained tech to the longest queue; urgent-first) |
| `greedy` | one-step look-ahead on *expected* values from observable information |
| `optimizer` (V0.5) | template/roster optimization on historical observed demand (MIP or local search) |

### 9.4 Benchmark pack format

```json
{
  "schema": "dreamteam.benchmark/v1",
  "id": "disruption-pack-03",
  "public": {
    "scenario": "established-moderate",
    "perturbationCategories": ["staff_absent", "room_down"],
    "horizonDays": 3,
    "evaluationTailDays": 7,
    "decisionEpochs": {"every": "30m", "openHoursOnly": true, "onAlerts": true},
    "actions": ["reassign_technologist", "call_in_staff", "extend_shift",
                "reschedule_appointment", "set_walkin_policy", "set_worklist_policy"],
    "metrics": ["operating_profit", "wait_p90", "urgent_tat_breach_share",
                "overtime_hours", "patients_rescheduled", "lost_demand",
                "unserved_orders_at_end", "unread_overdue_at_end"],
    "hardConstraints": ["no_unqualified_staff", "max_shift_hours",
                        "contrast_supervision", "open_hours"],
    "softConstraints": {"urgent_tat_minutes": 60, "overtime_hours_per_person": 2},
    "devSeeds": [1, 2, 3, 4, 5]
  },
  "private": {
    "testSeeds": "<32-byte secrets, held by the runner>",
    "perturbations": [{"at": {"day": 1, "time": "07:30"}, "type": "staff_absent",
                       "employee": "E04", "durationDays": 1}]
  }
}
```

Agents see only `public`. The runner holds `private` and never writes it where
an agent process can read it.

### 9.5 Evaluation

* Report metric **vectors**, with mean and 95% CI across seeds, per dimension
  (SIMULATION_DESIGN §11). A composite score is optional and pack-defined, and
  never replaces the vector.
* **Paired comparisons** use common random numbers: the same seeds for every
  agent, with keyed streams so that differences come from decisions.
* Report rejected actions and hard-constraint *attempts* (a safety signal even
  though rejected), plus soft-constraint violations.
* **Evaluation tail**: after the horizon, default policies run
  `evaluationTailDays` more days, and service metrics include them, so work
  pushed past the horizon still counts (SIMULATION_DESIGN §11).
* **Deterministic replay**: every run stores its run record. Any result can be
  reproduced and inspected in the TUI.

### 9.6 Safety constraints

* **Hard** (rejected by validation): unqualified staff, exceeding max shift,
  contrast without supervision, exams outside open hours, cancelling urgent
  studies, changing ordered exams (no clinical decisions).
* **Soft** (measured and gated): urgent TAT breaches, overtime, skipped breaks,
  waits, lost urgent demand.
* **Scope**: operational AI only. Agents never interpret images, choose
  examinations, or prioritize based on clinical content beyond the referrer's
  stated priority.

### 9.7 Hidden-state leakage prevention

Unexported latents. TUI and agents can't import `sim`. `StepVisible`. Secret
256-bit test seeds with cryptographic stream derivation. Public IDs allocated
at disclosure. Public/private pack split. Agents as isolated processes with no
snapshots or records. Validation messages that reveal nothing.
Reflection deny-list and same-past/different-future tests. Details:
SIMULATION_DESIGN §9.2–9.3.

---

## 10. Repository structure and CLI

### 10.1 Layout (target for end of V0.1)

```
dreamteam-go/
├── go.mod                      module github.com/georgezero/dreamteam-go
├── README.md
├── cmd/
│   └── dreamteam/main.go       subcommands, flags; no logic
├── internal/
│   ├── simtime/                Time, Duration, calendar, formatting
│   ├── rng/                    keyed streams, integer samplers
│   ├── api/                    Observation, Action, ActionResult, LogEntry, Metrics (JSON contract)
│   ├── sim/                    State, entities, queue, handlers, dispatch,
│   │                           policies, ledger, actions, invariants, metrics, projection
│   ├── scenario/               JSON types, merge/presets, validate, Build (+warm-up)
│   ├── session/                pacing primitives, action log, record/replay, digest
│   └── tui/                    Bubble Tea v2 app (see TUI_DESIGN §5)
├── scenarios/
│   ├── embed.go                go:embed of *.json
│   ├── base/established.json
│   ├── presets/{struggling,break-even,moderate,high}.json
│   └── established-moderate.json
├── testdata/
│   ├── golden/                 digests, CLI outputs
│   └── records/                sample run records
└── docs/
    ├── PLAN.md  ROADMAP.md  SIMULATION_DESIGN.md  TUI_DESIGN.md
    ├── ASSUMPTIONS.md          (V0.1 step 18: every tunable value, source, label)
    ├── calibration/pnl-model.xlsx  (V0.1 step 18: P&L calibration workbook, ROADMAP §10)
    └── turns/                  short per-change notes (optional)
```

Later: `internal/web/`, `internal/store/` (SQLite), `internal/agent/`,
`internal/bench/`, `benchmarks/` (packs), `docs/schema/`.

`sim` will be large, and that's fine. Split it into sub-packages only when a
boundary becomes real (e.g., `sim/ledger` if finance grows in V0.3).

### 10.2 CLI

```
dreamteam                               # same as `play`
dreamteam play    [--scenario ID|FILE] [--preset NAME] [--seed N] [--fps 10] [--debug-omniscient]
dreamteam run     [--scenario …] [--seed N] --days N [--actions actions.jsonl]
                  [--log events.jsonl] [--summary text|json] [--record out.json] [--check-invariants]
dreamteam replay  record.json [--verify] [--tui]
dreamteam scenario list | show ID | validate FILE | coverage ID
dreamteam version
# V0.2+: dreamteam agent --stdio --benchmark PACK --seed N
# V0.5+: dreamteam bench run PACK --policy rules --seeds dev ; dreamteam web
```

Dependencies (V0.1): standard library, Bubble Tea v2, Lip Gloss v2, optionally
Bubbles v2. Nothing else.

---

## 11. Objectives, scoring and quality gates (game layer)

* **Scenario objectives** (V0.4): e.g., "Achieve ≥ 12% operating margin over a
  quarter with urgent TAT breaches < 2%, wait p90 < 30 min, and no month with
  negative cash."
* **Quality gates**: a month's financial result counts toward the score only if
  gates hold. Gates are configurable per scenario.
* **Practice valuation** (V0.4): trailing-12-month EBITDA × multiple + cash −
  debt. The multiple is reduced when quality gates fail. This gives "financial
  growth" one long-horizon number without hiding the underlying metrics.
* V0.1 has no score. It shows metrics and daily insights only.

---

## 12. Open design questions

### 12.1 Needed before implementation — **decided 2026-10-13**

All five were settled as recommended. Q2 was confirmed explicitly (option A); the rest use the defaults.

| # | Question | Decision |
|---|---|---|
| Q1 | Default profitability preset for the first scenario: struggling, break-even, moderate or high? | **Moderate (~8.5% margin)** with visible inefficiencies. Viable but clearly improvable. Struggling as the "hard" option |
| Q2 | Radiologist model: employed with site allocation (global billing) vs contracted per-read teleradiology | **Employed, site allocation 50–60%** (D14) |
| Q3 | Jurisdiction for labor/payer assumptions | **Generic US** (FLSA weekly OT, Medicare-like reference rates), all configurable |
| Q4 | Include the three V0.1 actions (extend shift, walk-in policy, worklist policy) | **Yes**. Otherwise V0.1 is view-only |
| Q5 | Speed presets and start-paused behavior | **1 min/s, 5, 15, 1 h/s, Max; start paused** |

### 12.2 Can wait (decide by the listed version)

| # | Question | Default / note | By |
|---|---|---|---|
| Q6 | Should schedulers and agents see a patient's payer class? Real schedulers do. It enables payer-based discrimination | Visible to the player. Benchmark packs choose. Measure access disparity by payer | V0.2 |
| Q7 | Save files: allow save-scumming in the game? | Yes in the game. Irrelevant to benchmarks | V0.2 |
| Q8 | Virtual contrast supervision allowed? | Off by default, scenario flag | V0.2 |
| Q9 | Composite game score formula | Valuation with quality gates (§11) | V0.4 |
| Q10 | WebUI stack (templ + Datastar vs alternatives) | templ + Datastar unless something better appears | V0.5 |
| Q11 | Optimizer baseline: external MIP solver or pure-Go local search | Pure-Go local search first (no dependency) | V0.5 |
| Q12 | LLM provider abstraction and cost budget | Provider-agnostic harness, per-run token budget | V0.6 |
| Q13 | Real-data calibration (anonymized RIS timestamps from a partner) | Quantile-table format ready. Needs data governance first | any |
| Q14 | Multiple locations | Out of scope until V0.4 is solid | ≥ V0.7 |

---

## 13. Summary

### 13.1 Executive summary of the architecture

A single-threaded, **deterministic discrete-event engine** in Go (`sim`) owns
the practice's true state. It uses integer time and money, keyed random streams
(SHA-256-keyed ChaCha8) for reproducibility, common random numbers and seed
secrecy, and data-only events processed as atomic timestamp transactions
from a `(time, class, seq)` priority queue. A thin **`api`** package defines the only
view the outside world gets: observations without latent state, and validated
actions. **`session`** wraps the engine with pacing (advance, step to next
visible event, end of day), action logs, run records, replay and digests. The
**Bubble Tea v2 TUI** advances the session inside its update loop at 10 fps,
so rendering and speed never change outcomes. Scenarios are embedded JSON with
presets and validation. Established-practice history comes from a warm-up run of
the same engine. The WebUI and AI agents (later) plug into `session`/`api`
without touching the engine.

### 13.2 Recommended V0.1 scope

A typical Tuesday (continuing into following days) at the established practice
(`moderate` preset): demand, booking, cancellations, early/late/no-show
arrivals, walk-ins, registration with desk coverage, imaging with
credential/coverage/contrast constraints, radiologist reading with site
allocation, report finalization, claims with lag, accrued payroll and expenses,
and cash. TUI dashboard plus schedule, patients/journey, staff, worklist,
finance, log, actions, help and end-of-day screens. Time controls and
auto-pause. Three actions. CLI run/replay/scenario. Run records and replay. Full
detail: ROADMAP §2.

### 13.3 Key design decisions

D1 discrete-event · D3 keyed RNG · D4 integer-only engine · D8 unexported
latents + `api` boundary · D9 single-threaded engine inside the TUI loop · D11
revenue at report final, cash on payment · D13 warm-up history · D14
radiologist site allocation · D20 three real actions in V0.1. See §4.

### 13.4 Decisions made (formerly open questions)

Q1 `moderate` preset (~8.5% margin) · Q2 radiologists employed with site
allocation, global billing (per-read teleradiology possible as a V0.4 option)
· Q3 generic US rules · Q4 three actions in V0.1 · Q5 speeds 1 min/s, 5, 15,
1 h/s, Max, starting paused. See §12.1. The questions in §12.2 remain open
until their listed version.

### 13.5 Proposed implementation sequence (walking skeleton)

1 simtime → 2 rng → 3 queue/timestamp transactions (+chunking invariance,
synthetic-load benchmark) → **4 minimal api/session + TUI shell (first thing
you can watch: clock, controls, event log)** → 5 mini scenario + rooms/staff/
shifts → 6 demand/booking (schedule view) → 7 arrivals/walk-ins → 8
registration/imaging → 9 reading → 10 ledger/claims → 11 warm-up (real
established Tuesday) → 12 alerts/metrics/summary → 13 actions → 14 replay →
15 CLI → 16 TUI polish + golden screens → 17 end-to-end replay → 18
calibration → 19 performance → 20 docs. Details and closing tests: ROADMAP §2.

### 13.6 V0.1 acceptance criteria

Sixteen criteria in ROADMAP §2 across functional play (opens paused on a
populated Tuesday, visible flow, controls, all screens at 80×24 and 140×40,
actions change outcomes, CLI run and replay), determinism (identical digests
across TUI/headless, speeds, frame rates, amd64/arm64; chunking invariance;
invariants over 50 seeds × 90 days; leakage tests), calibration bands (volumes,
no-shows, utilization pattern, backlog, TAT, margin, cash/profit divergence) and
performance on the Raspberry Pi.

### 13.7 Deliberately deferred

Save/load UI, equipment failures, staff absences, phone/scheduling workload,
template and roster editing, rescheduling, overbooking and reserved slots, prep
failures, repeats, protocoling, multi-exam visits, denial rework and full
accounting statements, hiring, equipment purchase, extended hours, referral
dynamics, reputation, competition, the New Practice scenario, multi-site
operations, prior authorization, insurance eligibility, detailed coding,
depreciation, regulatory modelling, mouse, WebUI, SQLite, benchmark packs,
and AI agents. See ROADMAP §2 and the version sections for when each arrives.
