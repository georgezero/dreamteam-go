# V0.2 Step 0C — staffing architectures for the sustainable practice

**Question (owner option G, 2026-10-15):** can a larger outpatient practice
with a different, fully costed technologist staffing architecture reach
sustainable profitability under the existing simulation economics?

**Answer: yes at the primary demand case, narrowly, and not at the lower
sensitivity.** At imaging-weighted referral demand ×1.30, a **dedicated XR
and CT day core with part-time evening coverage and reading restored to a
62.5/52.5% allocation** (`W130-DED-MRpt-A62`) earns **+2.8% mean T90
operating margin (1.5% to 5.0%, median 2.6%) on all 20 paired seeds**, and
is profitable on all 20 in days 31–90. Its access and reporting service is
better than the shared-staffing control on every measured rate. The margin
is low single digits, not 6–12%. At ×1.15 demand every architecture loses
money (best −2.8%); break-even for the lead candidate is near ×1.24.
Recommendation: **CONDITIONAL GO** (see *Decision*).

All figures come from `results/step0c.md` (generated) and the per-seed
result files in `results/`.

## Method

* **Prerequisite fixed first (engine, outcome-neutral):** a shift's cap is
  now the earlier of start + `maxShiftHours` and accounting close
  (23:59:59), so a 16:00–20:00 shift is valid; validation checks shift
  length against the cap and the end against the day boundary
  (`sim.Policies.ShiftCap`, `internal/scenario/validate.go`). Existing
  rosters all start before noon, so their caps are unchanged; goldens,
  replay and determinism tests pass unchanged, and the 20-seed control below
  reproduces Step 0B's per-seed results exactly.
* **Harness** (`internal/sim/calib_test.go`, build tag `calibration`,
  read-only): adds when the shared-technologist conflicts fall (weekday ×
  hour), qualified staffed hours per room, technologist duty hours,
  reports by modality, an estimate of the radiologists' salary cost,
  payables, accrued payroll and receivables at the window end, and the
  lowest cash in the window (sampled daily).
* **Manifests:** `experiments/*.json` from `generate.py` (74). Every one is
  the established practice plus a static overlay: imaging-weighted demand,
  Step 0B's 12-minute XR slots, and the architecture's rosters. Each result
  records the merged scenario fingerprint and the overlay's SHA-256.
* **Runs:** no actions, default policies, from the takeover; T30 and T90.
  Nine paired seeds for every manifest; the control and the leading
  candidates at 20 paired seeds (`-s20`). Engine `0.2.0-dev.0.2`.
* **Report:** `report.py` writes `results/step0c.md`. Every Δ is paired by
  seed against the control at the same demand (`<demand>-SHR-MRft`, the
  Step 0B best configuration) with a 95% paired t interval.

Reproduce (about 40 minutes on a Raspberry Pi 4):

    python3 docs/calibration/feasibility/step0c/generate.py
    DT_EXP_DIR=../../docs/calibration/feasibility/step0c/experiments \
    DT_RES_DIR=../../docs/calibration/feasibility/step0c/results \
    go test -tags calibration ./internal/sim -run 'CalibrationExperiments$' -v -parallel 4 -timeout 600m
    python3 docs/calibration/feasibility/step0c/report.py

`python3 docs/calibration/feasibility/step0c/generate.py contracts` prints
the contract table below.

### Fixed

Reimbursement, payer mix and behaviour, wage and benefit rates, exam and
read durations, patient behaviour, rooms and equipment, opening hours,
templates (other than Step 0B's XR grid on every row), fixed expenses and
opening cash. Radiologists' rosters are unchanged; only their salaried site
allocation moves (Phase 4).

### Where the conflicts were (measured, control, 9 seeds, 90 days)

Lost referrals with a shared-technologist conflict, each spread over the
conflicting slots' weekday and hour:

* **CT 115:** Monday 72 (12:00–16:00 alone 57: Riley staffs X-ray and the
  CT contrast block), Tuesday–Friday mornings ≈11, Tuesday–Friday
  16:00–18:00 ≈20.
* **XR 61:** Monday 25, Tuesday–Friday 17:00–20:00 ≈30.

### Architectures (all hourly staff at documented rates)

| Code | Design |
|---|---|
| `SHR` (A, control) | Base: Riley (XR/CT, $36) 07:45–16:15 Mon–Fri on X-ray, covering CT all Monday and until 11:30 Tue–Fri; Sam (CT/XR, $40) 11:30–20:00 Tue–Fri on CT, covering X-ray from 16:15, and Saturday |
| `DED` (B) | Dedicated day core: Riley X-ray only, new Jamie (CT/XR, $38) CT only, both 07:45–16:15 Mon–Fri, each relieving the other's break; Sam the shared evening technologist 16:00–20:00 Mon–Fri (Monday evening newly staffed) plus Saturday |
| `DEDft` (E pair of B) | The same core; Sam keeps the full 11:30–20:00 shift |
| `DPT` (C) | Dedicated core plus part-time evenings: Sam on CT and new Robin (XR/CT, $38) on X-ray, both 16:00–20:00 Mon–Fri |
| `DPTft` (E pair of C) | The same with full 11:30–20:00 shifts for Sam and Robin |
| `FLT` (D) | Base shared rosters plus a cross-trained float, Jamie ($38), in the measured conflict periods: Monday 07:45–16:15 on CT, Tue–Fri 16:00–20:00 on X-ray (24 h/week) |
| `FLTft` (E pair of D) | The float as a 40-hour week: Monday as above, Tue–Fri 11:30–20:00 on X-ray |
| `MRft` / `MRpt` | Evening MR technologist Alex ($45): Step 0's full 11:30–20:00 shift, or part-time 16:00–20:00 (Casey works 08:00–16:30) |
| `A62` `A65` `A70` | Radiologist site allocation 62.5/52.5%, 65/55%, 70/60% (base 60/50%) |

**Contracts and cost.** Hourly staff are paid for every rostered minute
except unpaid breaks, plus run-over, ×1.5 past 40 hours a week, plus 25%
benefits and payroll tax: the same rules as every existing employee.
Contracted weekly hours are the roster's paid hours (the static payroll
obligation; the contract lifecycle is V0.2 action work). Recruiting and
onboarding are not modelled: these are steady-state staffing patterns.

| Architecture | Technologists (paid h/week) | XR/CT h/week | Technologist cost $/month |
|---|---|---|---|
| SHR-MRft (control) | Riley 40.0, Sam 39.5, Casey 40.0, Morgan 39.8, Alex 40.0 | 79.5 | 45,332 |
| SHR-MRpt | … Alex 20.0 | 79.5 | 40,457 |
| DED-MRpt | Riley 40.0, Sam 27.5, Casey 40.0, Morgan 39.8, Jamie 40.0, Alex 20.0 | 107.5 | 46,090 |
| DEDft-MRpt | Sam 39.5, Jamie 40.0, Alex 20.0, others as base | 119.5 | 48,690 |
| DPT-MRpt | Sam 27.5, Jamie 40.0, Robin 20.0, Alex 20.0 | 127.5 | 50,207 |
| DPTft-MRpt | Sam 39.5, Jamie 40.0, Robin 40.0, Alex 20.0 | 159.5 | 56,924 |
| FLT-MRpt | Sam 39.5, Jamie 24.0, Alex 20.0 | 103.5 | 45,397 |
| FLTft-MRpt | Sam 39.5, Jamie 40.0, Alex 20.0 | 119.5 | 48,690 |

Each `-MRft` variant costs $4,875 a month more than its `-MRpt` pair.
Radiologists: $480k a year each × site allocation × 1.25 (≈$55.0k a month at
60/50%, +$2.5k per 2.5 points on both).

## Results

### Architectures at ×1.30 demand (9 paired seeds, T90, base allocation)

| Architecture | MR evening full-time: margin, Δ $k/30 d | MR evening part-time: margin, Δ $k/30 d | Δ exams / 30 d (MRpt) | Δ lost referrals / 90 d (MRpt) |
|---|---|---|---|---|
| SHR (control) | +0.7% | +2.7%, +4.9 | 0 | 0 |
| DED | +1.9%, +3.3 | +3.9%, +8.3 | +64 | −149 |
| DEDft | +0.2%, −1.2 | +2.1%, +3.7 | +61 | −108 |
| DPT | +0.9%, +0.7 | +2.8%, +5.6 | +90 | −173 |
| DPTft | −2.1%, −7.1 | −0.2%, −2.1 | +92 | −139 |
| FLT | +1.5%, +2.2 | +3.5%, +7.1 | +57 | −125 |
| FLTft | +0.5%, −0.4 | +2.4%, +4.5 | +73 | −126 |

1. **Part-time MR evening is worth $4.9k a month with no volume change.**
   The full shift's 11:30–16:30 overlap with Casey (two technologists, one
   room) adds nothing measurable.
2. **Unsharing XR/CT works operationally.** Shared-technologist losses
   fall from CT 115 / XR 61 to 17/16 (DED), 9/0 (DPT) and 4/4 (FLT); FLT
   leaves the unstaffed Monday evening (46–52 CT losses become "not
   rostered").
3. **Part-time beats full-time for the same qualified coverage, every
   time:** by $4.5k (DED vs DEDft), $7.8k (DPT vs DPTft) and $2.6k (FLT vs
   FLTft) a month, with similar volume (the full-time pairs add 61–92 exams
   a month against 57–90). Full-time evening shifts pay for midday hours
   when both rooms are already covered.
4. **The added volume overwhelms reading.** At the base allocation, DED's
   urgent report p90 rises from 188 to 1,046 minutes and its urgent breach
   rate from 10% to 29% (20 seeds); DPT is worse (urgent p90 2,487 min).
   The extra imaging, especially Monday's (XR 18 → 41 and CT 9 → 23 exams a
   day), arrives faster than the radiologists' site blocks clear it.

### Reading capacity (Phase 4; ×1.30, part-time MR)

Site allocation scales the reading blocks linearly (`sim.ReadingBlocks`):
each 2.5 points on both radiologists adds ≈4.6% reading time for ≈$2.5k a
month.

| Run (20 seeds) | Margin mean (range) | Median | Profitable T90 / days 31–90 | Δ $k/30 d (95% CI) | Breach rate urgent / soon / routine | Urgent p90 | Routine p90 | Lost referrals / 90 d | Walk-ins left | Lowest cash $k |
|---|---|---|---|---|---|---|---|---|---|---|
| SHR-MRft (control) | 0.5% (−0.6–2.9) | 0.3% | 14/20 · 15/20 | — | 10.4 / 37.3 / 7.0% | 188 | 1,061 | 747 | 405 | 55.8 |
| SHR-MRpt | 2.5% (1.4–4.8) | 2.4% | 20/20 · 20/20 | +4.9 (4.9–5.0) | 10.9 / 37.4 / 7.1% | 188 | 1,061 | 747 | 405 | 60.9 |
| DED-MRpt | 3.8% (2.4–5.7) | 3.9% | 20/20 · 20/20 | +8.5 (8.1–8.9) | 28.9 / 70.8 / 27.5% | 1,046 | 2,207 | 601 | 304 | 60.7 |
| **DED-MRpt-A62** | **2.8% (1.5–5.0)** | **2.6%** | **20/20 · 20/20** | **+5.9 (5.4–6.5)** | **10.1 / 27.3 / 4.2%** | **158** | **933** | **601** | **304** | **58.9** |
| DED-MRpt-A65 | 1.8% (0.4–3.9) | 1.5% | 20/20 · 17/20 | +3.3 (2.6–3.9) | 9.3 / 20.5 / 3.6% | 125 | 795 | 601 | 304 | 56.9 |
| FLT-MRpt | 3.4% (2.2–5.5) | 3.0% | 20/20 · 20/20 | +7.2 (6.9–7.6) | 13.9 / 55.2 / 10.3% | 191 | 1,510 | 624 | 310 | 59.9 |
| FLT-MRpt-A62 | 2.3% (1.1–4.3) | 2.1% | 20/20 · 20/20 | +4.5 (4.1–5.0) | 9.9 / 27.7 / 4.6% | 124 | 934 | 624 | 310 | 58.1 |
| FLT-MRpt-A65 | 1.2% (−0.0–3.2) | 1.1% | 19/20 · 17/20 | +1.8 (1.3–2.3) | 8.6 / 22.5 / 3.9% | 88 | 843 | 624 | 310 | 56.0 |

Breach rate: breaches ÷ studies acquired in the window, per priority.
Dedicated coverage books more urgent patients (≈145 against 86 in 90
days), so raw breach counts rise even when the rate falls. Lowest cash is
reached around the third week (payroll before collections) and never
approaches zero. Payables plus accrued payroll at T90 are ≈$55k in every
row; receivables $360–375k.

At 70/60% reading improves little further and every architecture loses
money (9 seeds: −0.2% to −1.4%).

### Demand sensitivity (9 seeds, T90)

| Run | ×1.15 | ×1.30 | ×1.50 (diagnostic, out of range) |
|---|---|---|---|
| SHR-MRft (control) | −5.1%, 0/9 profitable | +0.7% | +2.4% |
| SHR-MRpt | −3.0%, 0/9 | +2.7% | +4.4% |
| DED-MRpt | −2.8%, 0/9 | +3.9% (reading collapses) | +3.6% (urgent p90 ≈ 7 days) |
| **DED-MRpt-A62** | **−4.0%, 0/9** | **+2.9%** | **+6.3% (urgent p90 2,269 min)** |
| FLT-MRpt-A62 | −4.3%, 0/9 | +2.4% | +5.9% (urgent p90 71 min) |

At ×1.15 no architecture is profitable: the rooms are no longer
constrained, so unsharing technologists recovers little, while the fixed
costs and salaried reading stay. Linear interpolation puts the lead
candidate's break-even near ×1.24. At ×1.50, reports drive revenue: more
reading raises profit (A62 beats A60 by $7.7k a month for DED).

## Binding constraints

1. **Demand level.** The economics work only near the documented upper
   bound (×1.30). Fixed costs ($88.5k a month) and salaried reading
   (≈$57k) are ≈60% of expenses.
2. **Reading capacity** is now the operational bottleneck: every recovered
   exam must be read, and evening and Monday imaging lands after the reading
   blocks. Restoring service costs ≈$2.5k a month per 2.5 allocation points.
3. **Low XR contribution** (≈$42 an exam) limits what added X-ray coverage
   can earn; CT recovery pays for the dedicated core.

## Player headroom (static, ×1.30)

The same practice spans **−2.1% to +3.9% margin** across these staffing
plans (≈$15k a month), and service-acceptable plans span +1.2% to +2.8%. The
decisions have real trade-offs: part-time versus full-time coverage
($2.6–7.8k a month); dedicated versus float coverage (access versus Monday
evening); and reading allocation, where 2.5 points cost ≈$2.6k a month and
take urgent p90 from 1,046 to 158 minutes. Starting from the shared control
(+0.5%), the best service-preserving plan adds +$5.9k a month (CI 5.4–6.5)
while improving every breach rate. This is static steady state: the managed
path (recruiting, notice, contract changes, transition losses) is V0.2
step 8 work.

## Decision

Against the owner's five tests, for `W130-DED-MRpt-A62`:

1. **Consistently positive 90-day results:** yes at ×1.30 (20/20 seeds,
   T90 and days 31–90; lowest 1.5%); no at ×1.15.
2. **Headroom for player decisions:** yes, ≈$6k a month from the shared
   starting point with better service, and visible service-for-money
   trade-offs.
3. **Acceptable access and urgent reporting:** better than the matched
   control on every breach rate, lost referrals −145 and walk-ins leaving
   −101. The formal §13.3 tolerances are not yet frozen.
4. **Realistic, fully accounted staffing:** yes, at documented rates with
   benefits, run-over and overtime; static contracts, no recruiting costs.
5. **Reproducible:** 20 paired seeds, fingerprints and overlays recorded.

**CONDITIONAL GO**, conditional on owner decisions:

* accept a **≈1.5–5% margin band** (mean ≈2.8%) as "sustainable", not 6–12%;
* accept **×1.30 imaging-weighted demand** as the sustainable scenario's
  defining demand, with the demand risk declared (break-even ≈×1.24;
  ×1.15 loses ≈4%);
* choose the **starting state**: the profitable candidate itself, or the
  shared-staffing practice at ×1.30 (≈break-even), with reorganization as
  the player's task (larger, measurable agency);
* the reorganization uses `hire`, `set_roster` and
  `set_reading_allocation`, but reducing Sam to 27.5 hours and Alex to a
  4-hour evening changes contracted hours, which the adopted scope has no
  separate action for: it must go through terminate and hire, with their
  costs, or the starting contracts must already be part-time;
* freeze the §13.3 service tolerances from these matched references before
  action code.

Not proposed: forcing the 6–12% band with demand beyond ×1.30, or cutting
reading allocation below the level that keeps urgent reporting at the
control's rate.
