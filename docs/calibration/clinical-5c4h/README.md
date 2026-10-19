# Slice 5C.4h: reserved protocol closure and clinical boundary safety

This slice answers the independent 5C.4g audit
(`docs/audits/slice-5c4g-final-clinical-policy-audit.md`, committed
verbatim before any of this) under the owner's decisions of 2026-10-19.
The evaluator is 0.2.0-dev.21.

* **Not locked, not run, not frozen.**
  * **Protocol:** proposed, not locked.
  * **T5:** the tolerances await the owner's approval.
  * **Boundary rules:** the revision awaits a targeted independent
    re-review.
* **Nothing reserved was run.** No reference or holdout seed, unseal
  record or reserved stream PCG(20261201, ·) was used; the harness
  refuses that family. No strategy was evaluated. The simulator ran
  with no action on the development seeds 901–910 only.
* **History unchanged:**
  * v0.2-eval-4 is byte-identical;
  * the 5C.4d, 5C.4e and 5C.4f candidates keep their rules (the 5C.4f
    bytes are kept as `internal/eval/testdata/candidate-policy-v5-5c4f.json`);
  * replay through dev.20 is exact.

| File | Contents |
|---|---|
| `reserved-validation-protocol.md` | The proposed complete confirmatory protocol: pinned items, streams, the decision, the criteria T1–T8 with estimators, intervals, thresholds and consequences, the T5 tolerance proposal, and limitations |
| `scenario-manifest.json` | The versioned executable manifest: 49 scenarios with constructions, streams, injections, effects, metrics, acceptance rules and failure consequences |
| `construction-checks.json` | The manifest run on the development family PCG(20261021, ·), 40 seeds a scenario: every scenario constructed, paired and judged. The near-margin and stable scenarios have 200 seeds |
| `boundary-safety.md` | The clinical-policy revision: the partial-window count ceiling, and the absolute limits' follow-up cohort with its four states |
| `allowance-interaction.md` | How the shared clinical allowance, case selection, Gate B and the absolute limits combine (audit G5) |
| `sim-access-5c4h.json` | The no-action simulator records on the development seeds under the revised candidate |

The candidate policy is `../clinical-5c4b/candidate-policy-v5.json`,
SHA-256 `e41d53bac22c00920257c9437bba641b425c7fd2234ff4dddde45dd23ebdce0e`.

## Results (development and synthetic only)

**Boundary safety** (`boundary-safety.md`):
* **The audit's witnesses are now NO GO.** The 640-report partial-window
  witness and the one-second horizon witness were both GO under 5C.4f.
* **Partial windows on the no-action simulator seeds.** The count
  ceiling fails no partial window, and every seed's clinical status is
  unchanged (all FAIL on genuine delays).
* **Follow-up on the no-action simulator seeds.** The follow-up cohort
  adds no violation. It finds routine studies not yet observable at the
  end of observation on 9 of 10 seeds: they were imaged 1.5–2.6 days
  before it, and nearly all were already past the 30-day access limit.
  Safety is then INSUFFICIENT, as specified. An owner option, a 45-day
  follow-up, is set out in `boundary-safety.md` §2.5.

**Scenario construction** (`construction-checks.json`, PCG(20261021, ·)):
* **Every scenario is constructed, paired and judged.**
  * All 49 scenarios on 40 seeds; no construction error.
  * Every pair matched on every priority, except each identity witness's
    own seed.
  * All 14 witnesses met their expectation.
* **The checks found one construction defect, now repaired.** The
  fidelity model drew each study's modality from the booking stream.
  Two runs of one practice on independent booking streams then differed
  by 2–3% in MR acquisitions, and v4 case selection's pooled acquired
  drop (2%) rejected **4 of 20** stable 40-seed groups. In the manifest's
  practices modality is now part of the exogenous order
  (`exogenousModality`), as in the simulator. The committed model is
  unchanged (`TestOCFidelityModelUnchanged`). After the fix **0 of 20**
  stable groups are rejected.
* **Stable practices, 200 seeds each.** No rejection, no absolute
  violation, no seed not yet observable, no insufficient window, and no
  safeguard flag in 800 seeds.
* **Near the margin (5 groups × 20 replicates).**

  | Increase | Gate B urgent accepted | Prediction |
  |---|---|---|
  | +2.0 points | 0.58 | 0.517 |
  | +2.1 points | 0.32 | 0.308 |
  | +3.0 points | 0 | 0 |

* **Harm.** Every harmed seed was flagged (15 of 15). In the harm
  scenarios v4 case selection usually rejects the group before the
  allowance does, so whole-evaluation rejection is reported by rule.
* **Unequal volumes (HV).** Every seed paired. At 60% volume, complete
  soon blocks fall below the minimum of 100: 12 of 40 seeds lack
  evidence (reported only).

## Owner approvals still required

1. **T5 tolerances** (protocol §5): recommended τ₂.₁ = 0.40, τ₃ = 0.01.
2. **The boundary-safety revision.** It passes its witnesses, but as a
   change to the clinical policy it needs a targeted independent
   re-review before any lock.
3. **The follow-up's length** (`boundary-safety.md` §2.5): keep 31 days
   (INSUFFICIENT whenever a report is still unobservable), or require
   45.
4. **The lock itself:** a commit pinning the candidate, the manifest,
   this protocol and the tolerances.

## Mutation checks

Each rule was broken alone and its witnesses rerun. **All 20 were
caught.**

**The partial-window rule:**
* the count ceiling turned off;
* the ceiling taken against the window's own size;
* the ceiling taken against one severe report fewer;
* complete windows below the minimum put under the ceiling;
* the vacuous pass allowed again.

**The follow-up:**
* post-horizon studies dropped;
* orders wanted after the horizon kept;
* not-yet-observable studies passing;
* unresolved overdue studies treated as not yet observable;
* the link integrity check turned off.

**Reporting:**
* occupied weeks set to the span;
* unavailable fractions set to zero;
* raw insufficiency counted only on seeds that do not fail.

**The criteria:**
* T1 allowing 4 of 62 groups;
* T6 allowing 20 flags;
* T2 accepting a detection rate of 0.94;
* T5b judged without approved tolerances;
* T4 accepting any rejection;
* the reserved family allowed;
* the pairing check turned off.
