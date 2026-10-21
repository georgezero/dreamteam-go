# The v5 T3 failure: S-A85, seed 259

An investigation of the single absolute-limit breach that failed the v5
confirmation, from the preserved evidence only. Nothing was rerun: no
reserved seed was drawn again, and seed 259 was not reconstructed.

## 1. Conclusion

**The preserved evidence is not sufficient to establish the cause.** It
establishes which rule was breached, by which record, on which seed and
priority, and that the breach is a reporting delay, not an imaging-access
delay. It does not contain the breaching study's times, so the wanted
time, the acquisition, documentation and report times, the deadline, the
breach's duration, whether the order was inherited, and the mechanism of
the delay are **unknown**. This is by design: the certified execution
keeps one summary line per group and discards the clinical records
(audit 5C.4m M1; protocol §8), and the line records only the seed that
breached each limit.

## 2. Observed facts

From `../clinical-v5-reserved/reserved-run.jsonl` (S-A85, group 6, seeds
240–279), `reserved-criteria.json`, the locked candidate and the code at
the lock commit:

| Fact | Value | Source |
|---|---|---|
| Scenario | S-A85: stable practice A85 against itself, the candidate side on an independent booking stream; no strategy difference between the two records | manifest |
| Record judged | the candidate side's record (the policy report's per-seed clinical measures) | `clinical.go` `Safety` |
| Breach | `pastDue urgent` on seed 259: one or more urgent studies whose report delay past due exceeds the limit | line `violatingSeedsByLimit` |
| The limit | an urgent study acquired at A is due at A + 60 min (urgent TAT target); the breach is a report later than A + 60 + 1,440 min = **A + 25 h**, or not reported by the end of observation with that delay already exceeded (counted "overdue"). For a study acquired before the takeover the clock starts no earlier than the takeover | candidate `absoluteMaxPastDueMin.urgent` 1,440, `tatTargetMin.urgent` 60; `clinical.go` absolute limits |
| Cohort | every urgent study acquired by the horizon (inherited included) and every study of an urgent order wanted by the horizon acquired in the 45-day follow-up | `safetyCohort ordersByHorizon` |
| Group decision | rejected by `safety`: T1's only rejection of 62 (limit 3), so T1 passes while T3 fails | line, criteria |
| Urgent access on seed 259 | 159 eligible urgent orders; the candidate left **0** unimaged (the baseline 1) | line `urgentSeeds` |
| Gate B on group 6 | PASS in every priority; all 40 pairs matched (no provenance failure) | line |
| Soon reporting on seed 259 | worst soon delay past due **2,211 min (36.9 h)**, the largest of the group's 40 seeds (soon limit 2,880 min, not breached) | line `worstSoonPastDueMin` |
| Severe urgent studies, group 6 | 1 of 6,027 urgent studies acquired in the gated period (severe after 720 min past due), summed over the 40 seeds | line `severeAcquired` |
| Other seeds | no other absolute violation in any stable scenario (S-A85, S-B85, S-C85, S-Cs85; 10,000 seeds) | criteria T3 |
| The model | the reserved run constructs records with the evaluation fidelity model (`internal/eval`), not the operational simulator (`internal/sim`), which did not run | protocol §2, harness |

## 3. What remains unknown

| Asked | Status |
|---|---|
| The urgent order's wanted time | not preserved |
| Acquisition time | not preserved |
| Documentation time | not preserved |
| Report completion time, or whether it was ever reported | not preserved: whether the breach was observed or overdue (censored) is not recorded |
| The deadline | known only as a rule: acquisition + 25 h (§2) |
| The breach's duration | not preserved (the worst delay is computed in memory and not written) |
| Whether the order was inherited | not preserved |
| Whether the reporting obligation was correctly attributed | not checkable from the line; attribution follows the order-study link verified by the record integrity contract, which passed (the group was valid) |
| The mechanism of the delay | not established |
| Whether the severe urgent study of group 6 is seed 259's | plausible (a breach of 1,440 min is beyond the 720-min severe threshold), not established: the count is summed over the group and covers only the gated period |

## 4. Hypotheses (not findings)

Each is consistent with the facts above; none is supported over the
others by them.

1. **A reading-capacity gap in the practice model.** Seed 259 also has
   the group's worst soon reporting delay (36.9 h), which suggests a
   period when reports were delayed across priorities, as a gap in
   reading coverage (a weekend, a holiday or a capacity lapse in A85's
   model) would cause. *Category: operational model behaviour, possibly
   realistic.*
2. **Censoring at the end of observation.** An urgent order wanted by
   the horizon but imaged late in the follow-up and unread by its end
   would count as an overdue breach. *Category: measurement boundary.*
3. **The takeover clock.** A study inherited near the takeover starts
   its clock at the takeover; a backlog at the takeover could breach.
   *Category: evaluation/measurement or model boundary.*
4. **A report-workflow edge case** in the model (a study whose report
   step is skipped or deferred unusually). *Category: report workflow.*
5. **A strategy limitation** is unlikely to be the explanation alone:
   both records are the same practice under the same plan, so a breach
   reflects the practice model, not a candidate strategy. The clinical
   policy's role is only the zero-tolerance rule that judged it.

## 5. What the pre-lock evidence could and could not show

Before the lock, T3 was exercised on development streams at no more
than 200 seeds per stable scenario (800 stable seeds). If the true
per-seed breach rate were 0.0001, that evidence would have shown no
breach 92% of the time, while a zero-violation rule over 10,000 seeds
would pass only 37% of the time. The criterion's operating
characteristic at full scale was not estimated before the lock. This is
an observation about the process, not a reinterpretation of T3.

## 6. Recommended v6 first slice: breach characterization on development streams

The smallest useful slice establishes the facts this investigation could
not, without touching T3 or the consumed family:

1. **Instrument, test-only.** On every absolute-limit violation, record
   the study-level facts in the group line: order ID, priority, wanted,
   acquisition, documentation and report times, due time, delay,
   observed or overdue, inherited or post-horizon, and the reading
   capacity at the time. No rule changes.
2. **Measure on a new development family**, declared in writing before
   it is drawn (never PCG(20261201, ·), never tuned against seed 259):
   the four stable scenarios at their full 2,500 seeds each (about 20
   minutes on `horo`), to estimate the breach rate and collect breaches.
3. **Classify each breach** into one of: operational scheduling defect,
   report workflow defect, evaluation/measurement defect, strategy
   limitation, clinical-policy design issue. Check whether the
   operational simulator (`internal/sim`) shows comparable urgent
   reporting delays on its development seeds.
4. **Stop for an owner decision.** A model or measurement defect is
   repaired as such. If the breaches are realistic, whether a
   zero-tolerance criterion over 10,000 seeds is the right v6 standard
   is a clinical-policy question for the owner, decided explicitly and
   before any new confirmation, never by relaxing T3 silently.

**Lesson for any future confirmation:** dry-run every zero-tolerance
criterion at the full manifest scale on development streams before the
lock, so that its operating characteristic is known in advance.
