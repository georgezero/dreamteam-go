# V0.2 Step 0B — CT/XR booking losses and template levers

**Question (owner, 2026-10-15, option B):** Step 0 found CT and XR losing
hundreds of referrals while their rooms were ≈40% used. Why are those
referrals lost, and can CT/XR appointment templates (now a permitted static
lever), alone or with shared-technologist coverage, make the Step 0 best
candidate materially profitable without degrading access or reporting?

**Answer: no.** The losses are not caused by the templates. At the Step 0 best candidate, 72% of
lost CT and XR referrals (57% in the unchanged practice) had a free, staffed, compatible slot inside the patient's
acceptance window that could not be booked because the **one technologist
covering both rooms** was already committed. 92% are urgent or soon
patients, who accept only 0–10 days. The best template, 12-minute XR slots,
gives a statistically real but immaterial gain: +$1.2k over 90 days (95% CI
+$0.4k to +$2.0k, 20 paired seeds), margin +0.4% → +0.5%. Its recovered
referrals displace walk-ins. Adding an XR/CT technologist recovers most
losses but only pays for itself, and moves the bottleneck to reading. The
Step 0 gate still fails.

Tables: `results/feasibility.md` (generated); per-seed results in
`results/`.

## Phase 1 — diagnosing booking losses

**Instrumentation.** A passive probe (`internal/sim/lossprobe_export_test.go`,
test builds only) runs when an order is lost, before the loss is recorded.
The production change is one function field on the simulation state
(`state.lossProbe`, excluded from the state digest like the existing test
hooks) and one nil-checked call in `offer` (`internal/sim/demand.go`). The
probe only reads state: no draws, no mutation.

**Neutrality evidence.**

* `TestLossProbeIsPassive` (normal suite): the established practice, 21
  days after the takeover, with and without the probe. Log, ledger, state
  and action digests are identical. Every lost order is diagnosed exactly
  once, and no diagnosis finds a slot FirstFit should have booked.
* With the probe on, the 20-seed Step 0 best candidate (`W130-MRfull-s20`)
  reproduces Step 0's per-seed profits and exam counts and its scenario
  fingerprint exactly.
* The golden references are unchanged.

**What the probe records.** For each lost order, it examines every slot of the
exam's category on every day of the patient's acceptance window: from the
earliest wanted day (not before today) to earliest + tolerance, capped at the
booking horizon. Each slot has exactly one blocking reason: the first check
FirstFit fails, in FirstFit's own order.

| Reason | Meaning |
|---|---|
| booked | an appointment holds the slot |
| not rostered | no qualified technologist rostered for the whole slot |
| no supervision | contrast slot without an on-site radiologist |
| notice | inside the same-day booking notice |
| shared technologist | free and staffed, but its technologists are needed by appointments already booked in the other room (shared-capacity rule) |

**Primary cause precedence.** A loss can have several causes. The primary
cause is the first that applies, in this order: the cause whose removal alone
would have booked the patient without moving anyone, most specific first.

1. shared_tech
2. not_rostered
3. no_supervision
4. notice
5. template_category: every compatible slot booked while other-category slots in the modality's rooms were free and bookable
6. category_full: every compatible slot booked and nothing else free
7. no_template_slot: no compatible slot in the window

The tables also count every loss with each cause present ("any"), and report
priority and how far beyond the window the first compatible slot was. Patient
preference windows (morning/afternoon/evening) never cause a loss: FirstFit
falls back to any time. Every loss is "first compatible slot after the
patient's tolerance". The booking horizon (120 days) binds only for orders
with no compatible slot at all (≈7 per modality per 90 days).

**Findings** (90 days, 9 seeds; prior best = imaging-weighted ×1.30 + MR
evening technologist):

| Configuration | Modality | Lost | Shared tech | Not rostered | Template category | Category full | No template slot | Urgent / soon / routine |
|---|---|---:|---:|---:|---:|---:|---:|---|
| Unchanged practice | CT | 49 | 27 | 4 | 2 | 4 | 9 | 26 / 19 / 5 |
| Unchanged practice | XR | 49 | 29 | 8 | 0 | 6 | 6 | 32 / 12 / 6 |
| Prior best | CT | 197 | 135 | 7 | 14 | 25 | 11 | 72 / 108 / 18 |
| Prior best | XR | 116 | 90 | 9 | 0 | 11 | 6 | 78 / 31 / 6 |
| Prior best | US | 386 | — | 379 | 0 | 3 | 4 | 23 / 99 / 263 |
| Prior best | MR | 102 | — | — | 45 | 41 | 14 | 19 / 56 / 27 |

* **CT/XR: shared technologist.** On weekdays one technologist staffs both
  X-ray and CT: Riley all Monday and Tuesday–Friday until 11:30, Sam from
  16:15. Both full templates are offered against one person. Routine
  bookings fill that shared time days ahead. Urgent and soon patients, who
  accept 0–2 and 1–10 days, find the remaining free slots blocked. The CT
  and XR rooms show ≈40% utilization because half their template time can
  never be used at once.
* **The template mix is a minor cause.** CT category mismatch causes about
  7% of CT losses. "Not rostered" slots (Monday evenings, breaks) are
  present in most losses but rarely primary.
* **US and MR differ.** US losses are free US slots with no technologist
  rostered (evenings, Monday B weeks), mostly routine patients whose first
  staffed slot came ≈15 days after their tolerance. MR losses with the evening technologist are template category mix
  and full templates.

## Phase 2 — template and coverage variants (9 paired seeds)

All variants are on the prior best candidate. Everything else stays at its
base value: exam and read durations, rooms, qualifications, pay and expense
rates, reading allocation (except the one labelled run), reimbursement,
patient behaviour, demand and opening hours. Every template stays within
opening hours, and the scenario validator and the booking rules (shared
capacity: one technologist, one exam at a time) apply unchanged.
`generate.py` holds the exact slot lists.

| Code | Change |
|---|---|
| `XR12` | XR slots of 12 minutes, 08:00–20:00 weekdays: 60 instead of 48. XR exams occupy the technologist p50 10–13 minutes (room 8–11 + 2–2.5); CT_15 already takes p50 16–17 minutes, so CT cannot be packed closer |
| `XRhalf` | XR slots every 30 minutes in the shared-technologist hours (block boundaries: the shared time is left to CT), every 15 minutes otherwise |
| `STG` | Control: the XR grid offset 5 minutes against CT (staggering) |
| `XRCT` | + XR/CT technologist on CT, 08:00–16:30 (unshares mornings) |
| `XRCTeve` | + XR/CT technologist on X-ray, 11:30–20:00, $38/h (unshares evenings) |
| `A65` | radiologist allocation 65%/55% |

Not tested, with reasons: filling the 15:30–16:00 CT gap (it is Sam's
break, so those slots would be unstaffed); closer CT packing (slots already
shorter than exam time); priority holdback slots (not a template property:
categories are by exam, not priority).

| Variant (90 days) | Margin | Δ profit $k / 30 d vs prior best | XR+CT lost | Walk-ins leaving | Urgent breaches | Routine report p90 |
|---|---:|---:|---:|---:|---:|---:|
| Prior best | +0.5% | — | 313 | 361 | 8 | 17 h |
| `XR12` | +0.7% | +0.4 (−0.8 to +1.6) | 267 | 398 | 9 | 17 h |
| `XRhalf` | −0.4% | −2.3 | 503 | 333 | 6 | 16 h |
| `STG` | −1.7% | −5.2 | 487 | 300 | 4 | 15 h |
| `XRCT` | +0.1% | −1.0 | 168 | 301 | 16 | 26 h |
| `XRCT` + `XR12` | −0.1% | −1.4 | 155 | 330 | 15 | 25 h |
| `XRCTeve` | +0.5% | −0.1 | 143 | 228 | 34 | 35 h |
| `XRCTeve` + `XR12` | +0.3% | −0.4 | 140 | 258 | 26 | 32 h |
| `XRCTeve` + `A65` | −1.6% | −5.3 | 143 | 228 | 17 | 14 h |

What the variants show:

* `XR12` recovers about 45 referrals, but walk-ins then leave instead
  (+37). The shared technologist's time is the same, so completed exams do
  not change.
* `XRhalf` moves losses from CT to XR and costs money. CT's higher
  contribution does not offset losing XR referrals and walk-ins.
* `STG` is worse than the base grid. Each offset XR slot overlaps two CT
  slots, so in the shared hours one booking blocks two slots in the other
  room. Misaligned grids fragment the shared technologist's time;
  staggering creates none.
* An XR/CT technologist recovers most CT/XR losses (+160 to +260 XR+CT exams
  per 90 days) and pays for itself, no more. The extra evening exams then
  wait for the next morning's reading, so urgent breaches and the routine
  tail worsen. Restoring reading with the 65/55 allocation costs more than
  the exams earn.

## Phase 3 — 20 paired seeds

| Configuration | Window | Margin mean (range) | Profit $k mean / median | 95% CI | Profitable seeds |
|---|---|---:|---:|---:|---:|
| Unchanged practice | T90 | −14.0% (−17.0 to −11.3) | −82.9 / −83.1 | −86.2 to −79.7 | 0/20 |
| Prior best | T90 | +0.4% (−0.9 to +2.4) | +2.8 / +1.4 | −0.2 to +5.7 | 12/20 |
| Prior best + `XR12` | T90 | +0.5% (−0.6 to +2.9) | +3.9 / +2.4 | +0.8 to +7.0 | 14/20 |
| Prior best | T30 | −0.1% (−2.2 to +3.8) | −0.1 / −0.1 | −1.8 to +1.5 | 10/20 |
| Prior best + `XR12` | T30 | +0.2% (−1.5 to +4.2) | +0.5 / +0.4 | −1.1 to +2.1 | 11/20 |

Paired `XR12` − prior best: +$1.16k over 90 days (95% CI +$0.36k to +$1.96k;
15 of 20 seeds better); +$0.64k over 30 days (CI +$0.15k to +$1.14k).

Access and reporting with `XR12` (20 seeds, 90 days):

| Measure | `XR12` | Prior best |
|---|---:|---:|
| Referral orders lost | 747 | 790 |
| CT lost / of which shared technologist | 175 / 109 | 190 / 126 |
| XR lost / of which shared technologist | 85 / 59 | 113 / 87 |
| Walk-ins leaving unseen | 405 | 366 |
| Urgent breaches | 9 | 8 |
| Routine report p90 | 17.7 h | 17.4 h |
| Cash at 90 days | $80k–105k | — |

## Phase 4 — evaluation

| Question | Result |
|---|---|
| 1. Recover lost CT/XR referrals? | Partly. `XR12` −45 referral losses; an XR/CT technologist −145 to −170 |
| 2. Completed and reported exams? | `XR12`: no (walk-ins displaced, exams unchanged). XR/CT technologist: yes, +160 to +260 XR+CT exams per 90 days |
| 3. Operating margin? | `XR12`: +0.1 point, statistically real, immaterial. Technologist: none |
| 4. Access and reporting preserved? | `XR12`: walk-in abandonment +11%. Technologist: urgent breaches ×2–4, routine report tail ×1.5–2 |
| 5. Robust across seeds? | The best candidate loses money on 6 of 20 seeds |

**Binding constraints, in order:**

1. **Shared technologist staffing.** One person covers two rooms for half
   the weekday. Templates cannot create technologist time.
2. **Low XR contribution** (≈$42 per exam). An XR/CT technologist's
   recovered work does not cover ≈$8k per month.
3. **Reading coverage after the last reading block.** More evening imaging
   becomes next-morning reports. Fixing that with allocation costs more
   than it earns; teleradiology is the V0.2 action meant for this
   (deferred).
4. **Fixed costs and salaried reading** (≈58% of expenses; Step 0).

Patient behaviour sets which losses occur (urgent and soon tolerances), but
it is not the lever. Template geometry is exhausted: the best template is
worth ≈0.1 margin point.

**Verdict: the gate fails again.** No permitted static configuration is
materially profitable. The best is `XR12` on the prior best candidate:
+0.5% mean T90 margin, 14 of 20 seeds profitable.

## Reproduce

    python3 docs/calibration/feasibility/step0b/generate.py
    DT_EXP_DIR=../../docs/calibration/feasibility/step0b/experiments \
    DT_RES_DIR=../../docs/calibration/feasibility/step0b/results \
    go test -tags calibration ./internal/sim -run 'CalibrationExperiments$' -v -parallel 4 -timeout 600m
    python3 docs/calibration/feasibility/report.py step0b/results diag-W130-MRfull W130-MRfull-s20

About 9 minutes on a Raspberry Pi 4. Results record engine 0.2.0-dev.0
(run with this step's passive probe; outcomes as v0.1.0).
