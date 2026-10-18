# v0.2-eval-4 evaluation-framework audit (Astra High)

**Audited revision:** commit 7707dbd (0.2.0-dev.14, policy v0.2-eval-4, digest `4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3`), 2026-10-18. Line references are to 7707dbd.

The audit was independent and read-only, and covered the whole V0.2 evaluation framework. The auditor read a git-archive snapshot of the repository with no Go toolchain. It ran no test, simulator or strategy. It ran no reference seed 1001–1020 and generated no holdout seed.

* **Sections 1–12 and Appendices A–B** are the auditor's report as delivered, with only the file links rewritten to be repository-relative. Its evidence labels ("independently computed", "verified by reading implementation", "reported", "inferred", "unexecuted") are the auditor's own.
* **Appendix C** comes from the maintainer, after the audit. It covers witness tests the maintainer ran and code checks the maintainer made. It is kept separate, so the auditor's claims can be told apart from what was later executed.
* **The owner's decisions** on the open questions in section 9 are recorded in `docs/evaluation/proposals/v5-clinical-acceptance.md` (Slice 5C.4a, a later commit), not here.

---

**1. Overall verdict: NO GO for the one-time confirmatory evaluation at this revision.**

The implementation has substantial strengths: exact threshold comparisons, reconciled study counts, transaction-level cash tracking, common observation horizons, explicit exploratory comparisons, and reproducible calibration arithmetic. Nevertheless, the combined clinical gates admit materially harmful delays, and the holdout safeguards do not enforce the claimed execution and approval boundaries.

E3 remains a reasonable **provisional service-purchase candidate**. This audit does not establish that E3 produces the adverse outcomes described below; it establishes that the evaluator can accept them.

No files were changed. No Go tests, simulator runs, strategies, reference evaluations, or holdout seed derivations were executed. Appendix tests are proposed, unexecuted witnesses. “Computed” below means arithmetic on committed calibration counts or explicitly synthetic numbers.

**2. Ranked findings**

No Critical defect was established. The following High findings block acceptance.

| ID | Severity | Finding | Blocks primary freeze? | Blocks unseal? |
|---|---|---|---|---|
| H1 | High | Severe reporting delays can pass every clinical gate | Yes | Yes |
| H2 | High | Access gates protect eventual throughput, not timely access | Yes | Yes |
| H3 | High | Sealed holdout execution is not excluded; a default test uses a real holdout seed | Not candidate selection alone | Yes |
| H4 | High | Approval does not bind the exact executable implementation | Yes, final approval | Yes |
| H5 | High | First-run immutability and authorized reproduction rely on cooperation | No | Yes |
| H6 | High | Canonical runner silently tolerates refused strategy actions | Yes, execution semantics | Yes |
| M1 | Medium | Calibration queue differs materially from the simulator and has an arrival-order defect | Yes, calibration acceptance | Yes |
| M2 | Medium | Selection uncertainty and near-capacity false rejection remain unresolved | Owner decision required | Yes |
| M3 | Medium | “Service restored” includes an insufficient-evidence baseline | No for the reported E3 comparison | Correct before release |
| L1 | Low | Observation-end documentation omits settlement extension | No | No |

**H1 — Severe reporting delays can pass every clinical gate. Confirmed acceptance gap; engine reachability inferred.**

The level check permits

\[
B_{\text{later}}\leq wB_{\text{reference}}
+\max(1.5wB_{\text{reference}},2).
\]

Thus “150% allowance” means **up to 250% of the workload-adjusted reference**, not 150% of it. Late-work checks use medians, while the older 10% mean check and peak are informational. See [measure.go:976](../../internal/eval/measure.go:976) and [policy-v0.2-eval-4.json:96](../../docs/evaluation/policy-v0.2-eval-4.json:96).

A concrete synthetic witness uses one urgent acquisition daily, with six consecutive urgent studies acquired on days 112–117 each reported **eight days later**:

- Urgent breach rate over 150 days: **6/150 = 4%**.
- Urgent tail rate: **6/60 = 10%**.
- T180 urgent rate: **6/180 = 3.33%**.
- Additional backlog in block 111–138: **48 study-days / 28 = 1.714 studies**, below the two-study floor.
- Backlog is positive on only 13 mornings. Median overdue count, stale count and overdue exposure are all **zero**.
- No study exceeds the fourteen-day persistence threshold.

These numbers were independently computed. Appendix A supplies a fully reconciled record witness expected to receive **GO** when its synthetic finances are favorable. This demonstrates that service gates do not generally compensate for the median-based backlog blind spot.

The repository already explicitly expects ten routine studies delayed eight days to pass the v4 backlog rule when the affected mornings are a minority: [adversarial_test.go:465](../../internal/eval/adversarial_test.go:465).

There is also a narrower age-cap discrepancy. Persistence is checked only at daily samples, and only through the final covering block: [measure.go:895](../../internal/eval/measure.go:895). A study finalized **348 hours after its due time** can have a largest sampled overdue age of **334 hours**, escaping the nominal 336-hour cap. Appendix A includes this witness. At plan level, even detected persistence contributes to the backlog failure allowance; it is not an unconditional zero-tolerance rejection across all forty seeds.

**Consequence:** a passing result does not establish acceptable severity or duration of reporting delays.

**Minimal correction:** obtain owner-approved limits for severe delay and concentrated service failure; evaluate the maximum overdue interval directly from study timestamps, including completed studies, rather than sampling it. Specify whether persistence permits any failing seeds. Preserve v4 and issue a successor policy if acceptance rules change.

**H2 — Eventual service can conceal substantial deterioration in access. Confirmed acceptance gap; engine reachability inferred.**

Demand cohorts are correctly anchored to `WantedSec`, but their gated served outcome is taken at the **end of observation**. `ServedByWindowEnd`, booking lead and displacement measures do not establish a hard waiting-time requirement: [measure.go:674](../../internal/eval/measure.go:674). The comparison gates protect served fractions, acquisitions and losses: [compare.go:315](../../internal/eval/compare.go:315).

A candidate can shift the first thirty urgent acquisitions—and their appointments and reports—thirty days later, entirely within T90:

- Same wanted demand.
- Same eventual service.
- Same T90 and tail acquisition totals.
- Same acquisition-to-report TAT.
- No additional overnight reporting backlog.

The first month’s urgent patients wait an extra month, yet the current gates can all pass. Appendix A provides the expected-GO witness.

This omission is disclosed in [policy-v0.2-eval-4.json:191](../../docs/evaluation/policy-v0.2-eval-4.json:191). Disclosure makes it an explicit limitation, but does not make the framework adequate for the requested clinical acceptance claim.

**Consequence:** reducing imaging capacity or postponing appointments can worsen patient access without breaching reporting or case-selection gates.

**Minimal correction:** gate wanted-to-actual-service delay, with priority-specific limits and unresolved orders included. Alternatively, explicitly narrow the confirmatory claim to exclude timely access; that would not satisfy the primary question as presently framed.

**H3 — Holdout protection starts after execution is already possible. Confirmed code defect and governance limitation.**

`TestHoldoutRecordersGuarded` loads the repository’s real holdout seed list, creates a **mini-scenario** world with its first seed, and advances that world to takeover **before** asserting that the ordinary recorder refuses it: [holdout_protocol_test.go:283](../../internal/sim/holdout_protocol_test.go:283).

This is not evidence that a full confirmatory strategy evaluation occurred. It is direct evidence that the ordinary test suite contains a sealed-holdout-seed execution path.

Other boundaries are also incomplete:

- `LoadHoldout` returns the seed list independently of owner authorization; `Authorize` invokes it before checking whether the unseal record exists: [protocol.go:157](../../internal/eval/protocol.go:157).
- Ordinary sessions construct and advance worlds without holdout authorization: [session.go:38](../../internal/session/session.go:38).
- The recorder checks a **caller-supplied seed label**, not its identity against the world’s actual random stream: [export_eval_test.go:34](../../internal/sim/export_eval_test.go:34).

The commitment mechanism establishes a reproducible set; it does not make that set inaccessible to a developer with the repository.

**Consequence:** canonical-run exclusivity and “never previously exercised” cannot be established from these guards. The documented full-suite executions do not resolve this concern.

**Minimal correction:** replace the guard test’s real manifest with a synthetic fixture; enforce authorization before world construction on supported entry paths; bind recorder identity to the world. Separately obtain an owner-reviewed exposure accounting. If resistance to deliberate developer access is required, custody must sit outside the freely readable repository.

**H4 — Approval binds artifact labels and selected files, not the exact implementation. Confirmed.**

Authorization compares the approved engine string with `version.Number`, a release constant. It does not bind the source revision or executable digest: [holdout_runner_test.go:244](../../internal/sim/holdout_runner_test.go:244), [version.go:10](../../internal/version/version.go:10).

The committed-file check covers manifests, policy and strategy files, but not the evaluator, recorder or simulation source: [protocol.go:212](../../internal/eval/protocol.go:212). Modified code retaining `0.2.0-dev.14` can therefore satisfy this identity check.

Approval provenance is also weaker than its wording suggests:

- It compares `approvedBy` with the **first author name** of the unseal file.
- It does not authenticate an owner identity or verify approval of the file’s current bytes.
- Later committed edits retain that first-author provenance.

See [protocol.go:250](../../internal/eval/protocol.go:250) and [protocol.go:386](../../internal/eval/protocol.go:386).

Finally, an authorized `Protocol` exposes mutable policy, seed, strategy and approval fields; `Authorized()` only checks a boolean: [protocol.go:95](../../internal/eval/protocol.go:95).

**Consequence:** approval of one implementation can be presented as approval of a different implementation or subsequently altered configuration.

**Minimal correction:** bind approval to a complete source-tree/build digest, current manifest bytes and authenticated owner approval; reject modified builds; make the authorized configuration immutable or revalidate it before execution.

**H5 — Run-once enforcement is a local artifact convention. Confirmed.**

`CommitRun` compares existing files byte for byte, which is useful. However, the first publication uses ordinary `os.WriteFile` calls after a separate existence check, without exclusive creation, a lock or atomic publication of the pair: [protocol.go:326](../../internal/eval/protocol.go:326).

Consequences include:

- Concurrent first runs can race.
- Removing the directory or using a fresh checkout resets the local “first run.”
- A crash between report and manifest writes leaves an incomplete publication.
- A mismatching rerun is rejected **after** all outcomes have been recomputed, not before exposure: [holdout_runner_test.go:185](../../internal/sim/holdout_runner_test.go:185).
- File mode `0444` does not provide an immutable record against the file owner.

Successful artifacts also contain literal seed identifiers: [protocol.go:276](../../internal/eval/protocol.go:276), [evaluate.go:64](../../internal/eval/evaluate.go:64). Normal success logging is restrained, but artifact confidentiality is not equivalent to seed-free logging.

**Minimal correction:** preflight reproduction against the approved implementation and existing manifest before simulation; publish under exclusive locking with atomic completion; record first execution in a durable owner-controlled registry. Decide whether post-unseal seed identifiers may appear in distributed artifacts.

**H6 — Canonical execution can silently differ from the approved action file. Confirmed.**

Both `strategyTail` and `runStrategy` ignore unsuccessful action results, recording only the time of successful actions: [holdout_runner_test.go:114](../../internal/sim/holdout_runner_test.go:114), [holdout_runner_test.go:165](../../internal/sim/holdout_runner_test.go:165).

The development runner instead treats refused actions as errors: [strategy_dev_test.go:84](../../internal/sim/strategy_dev_test.go:84).

**Consequence:** a partially executed strategy can be stamped with the complete frozen strategy’s hash and judged as that strategy. Development/canonical equivalence tests cover successful execution, not this semantic difference.

**Minimal correction:** share action-application logic; abort on unexpected refusal; record action outcomes. If refusal is intended behavior, its treatment must be specified before freeze.

**M1 — Calibration does not validate the combined evaluator on operationally equivalent queues. Confirmed mismatch; impact magnitude unmeasured.**

The synthetic queue uses aggregate hourly capacity and strict urgent→soon→routine priority. The actual default simulator promotes overdue work alongside urgent work and otherwise orders by worklist entry time: [oc_queue_test.go:60](../../internal/eval/oc_queue_test.go:60), [reading.go:125](../../internal/sim/reading.go:125).

The real engine also includes individual readers, allocation blocks, breaks, shift-fit constraints, stochastic read durations, documentation delay and remote-reading mechanics. Those affect which priority absorbs capacity loss.

There is a specific synthetic ordering defect: arrivals within an hour receive random acquisition times but are appended without sorting. Reading examines only each priority queue’s head. A later arrival can therefore block an earlier arrival behind it: [oc_queue_test.go:128](../../internal/eval/oc_queue_test.go:128).

Most importantly, backlog calibration calls `judgeBacklog`; it does not apply the complete service/access evaluator to those same deterioration records: [oc_backlog_test.go:300](../../internal/eval/oc_backlog_test.go:300). Service clustering calibration separately generates breach indicators: [oc_cluster_test.go:124](../../internal/eval/oc_cluster_test.go:124).

**Minimal correction:** fix chronological queue ordering and prospectively validate the **combined** gates on fresh, non-holdout synthetic operational cases. Include the actual priority policy, concentrated delays, access deterioration and late-horizon effects. Do not interpret existing counts as measured compensation by service gates.

**M2 — Calibration selection is reproducible, but its uncertainty and applicability are narrower than the headline acceptance claim.**

The final streams are fresh relative to exploratory sweeps, but the same final counts select the winner and estimate its operating characteristics: [oc_backlog_test.go:348](../../internal/eval/oc_backlog_test.go:348). This is a prespecified selection procedure, not an independent validation of the selected rule.

The candidate family followed exploratory sweeps. That creates model-selection risk even if no candidate’s final strategy result entered the selection formula. The snapshot cannot establish the chronology claimed by the protocol.

Near-capacity seasonal behavior is explicitly outside the acceptance targets: [protocol.md:100](../../docs/calibration/backlog-5c3/protocol.md:100). Its roughly 56% forty-seed false rejection is therefore not a selection-code error, but it is a substantial applicability restriction.

**Minimal correction:** owner acceptance of the intended utilization/seasonality domain, followed by fresh validation of the selected policy. Use uncertainty bounds, not only point estimates, when claiming the false-rejection target is established.

**M3 — Insufficient evidence can be described as service restoration. Confirmed.**

`ServiceRestored` and the service-purchase branch use `base.ServiceStatus != Pass`, which includes `Insufficient`: [compare.go:442](../../internal/eval/compare.go:442).

**Consequence:** a candidate can be classified as restoring service when the baseline’s service failure has not been established.

**Minimal correction:** distinguish a demonstrated failure from insufficient evidence. This does not overturn the reported E3 comparison, whose baseline is reported to fail measured service gates.

**L1 — Observation-end example is stale. Confirmed.**

[README.md:103](../../docs/evaluation/README.md:103) says a day-170 horizon ends observation on day 194. The implementation also requires `H + 31`, giving at least **day 201**: [measure.go:334](../../internal/eval/measure.go:334).

**Minimal correction:** update the formula and example. The implementation provides the longer protection.

**3. Policy integrity and reproducibility**

The independently computed SHA-256 of the exact v4 policy bytes is:

```text
4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3
```

It matches the full pin in [p6-eval4/run.json:5](../../docs/calibration/strategy-development/results/p6-eval4/run.json:5) and the abbreviated policy identifiers found in the accompanying report and progress documentation.

Additional byte checks passed:

| Check | Independently verified result |
|---|---|
| Historical policies | v1–v3 files present; successor `supersedes` hashes match |
| Protected history | All 48 v4 `historicalReferences` digests match |
| Development sets | All six set-file digests match their `run.json` manifests |
| Strategy artifacts | All 43 strategy entries across those sets match their files |
| Available detailed reports | p1–p4 `report.json` digests match their manifests |
| Tele-only declarations | All five declared tele-only relationships have identical non-teleradiology actions and times |

The loader rejects superseded policy schemas: [policy.go:208](../../internal/eval/policy.go:208). The canonical path binds the approved policy digest, strategy-set digest and individual strategy hashes.

However:

- A policy’s self-description as frozen is not an immutable current-policy digest check.
- The supplied archive cannot independently establish commit ancestry, freeze chronology, owner provenance or absence of post-freeze edits.
- p5/p6 detailed reports and trajectories are not committed here. Their manifest hashes cannot independently establish the underlying outcomes without those bytes.
- Historical file preservation is verified; historical A–D simulation neutrality is reported and covered by test source, not independently executed.

Session registration includes dev.13 and preserves its replay mode: [record.go:415](../../internal/session/record.go:415). Tests specify exact replay across step sizes and continuation with the original record embedded: [dev13_test.go:22](../../internal/session/dev13_test.go:22). The migration declaration identifies dev.14: [MIGRATIONS.json:3](../../testdata/golden/MIGRATIONS.json:3).

Cross-machine byte equality and sixteen caught mutations are **reported**, not independently reproduced. The cited machines provide useful ARM-machine replication, not demonstrated portability across all operating systems and architectures: [MEASUREMENTS.md:493](../../docs/MEASUREMENTS.md:493).

**4. Specific judgment on the v4 backlog gate**

The rule is implemented consistently with its main numerical specification, but it is **not sufficient as a clinical safety gate**.

| Element | Audit judgment |
|---|---|
| 28-day, rota-aligned blocks | Correctly implemented |
| Reference block | For H=150, days 55–82 |
| Intermediate/final blocks | Days 83–110, 111–138 and 139–166; final block covers H |
| Workload adjustment | Later/reference acquisitions; zero workload falls back to an unscaled comparison |
| Level | Mean; permits adjusted reference plus 150% or two studies |
| Late work | Medians; can suppress substantial minority-of-days harm |
| Stale cap | Median stale count ≤3, including reference; not a daily maximum |
| Persistence | Sampled age test; incomplete as a continuous fourteen-day maximum |
| Missing data | Missing samples or inconsistent acquisition/backlog counts make records invalid |
| Reference inflation | Severe persistent stale inflation is constrained; inflated younger queues and brief stale bursts remain possible |
| Near-horizon deterioration | Covering block helps, but averaging dilutes late changes; persistence adjudication stops at its end |
| Plan-level allowance | Up to four unstable seeds at forty, including persistence failures |

Block construction and adjudication are at [measure.go:822](../../internal/eval/measure.go:822) and [measure.go:895](../../internal/eval/measure.go:895); reconciliation is at [measure.go:1119](../../internal/eval/measure.go:1119).

The gate can detect major sustained accumulation, inherited studies left unresolved beyond the sampled age cap, and sufficiently prolonged replacement of young backlog with older work. It cannot establish that a queue is clinically acceptable merely because its growth and median old-work burden are within these limits.

An unacquired inherited order is additionally outside the study-age safeguard. Its eventual-service protection does not supply a maximum access wait.

**5. Do service gates compensate for the capacity-deterioration blind spots?**

**No general compensation has been demonstrated, and the synthetic witnesses show that compensation is not guaranteed.**

The service implementation correctly uses:

- Urgent ≤12%, soon ≤32%, routine ≤6%.
- Five joint per-seed checks: urgent over the whole gated period; soon and routine separately over T90 and tail.
- At most four failing or insufficient seeds at forty.
- Separate pooled T90/tail requirements.
- Minimum cohorts of 43 urgent, 88 soon and 23 routine studies.
- Due-time classification: a report finalized by its due time passes, including after a window boundary; an unfinished study whose deadline has passed breaches.

See [policy-v0.2-eval-4.json:36](../../docs/evaluation/policy-v0.2-eval-4.json:36) and [measure.go:556](../../internal/eval/measure.go:556).

But utilization does not determine breach rates. With demand at 85% of initial capacity, a 10% capacity reduction gives:

\[
\rho'=0.85/0.90=94.44\%.
\]

With the stated 1.05 seasonal factor, utilization becomes **99.17%**.

For an illustrative urgent-first fluid allocation using the synthetic 2%/12%/86% mix:

- Urgent plus soon consume \(0.85\times0.14=0.119\) initial-capacity units.
- Remaining post-cut capacity is \(0.9-0.119=0.781\).
- Routine demand is \(0.85\times0.86=0.731\).
- Routine utilization of the residual capacity is **93.60%**.

With sufficiently smooth arrivals, all priorities can still meet their deadlines: **0%, 0%, 0% breaches is possible**, despite the capacity reduction. With bursts and overnight closures, breach rates can be much higher. Neither result can be inferred quantitatively from utilization alone.

Therefore, it would be unjustified to assert that the 12/32/6% gates necessarily trip. Conversely, capacity reduction by itself is not proof of clinically unacceptable service.

There is an important favorable correction to the owner’s concern: **22.8% and 45.9% are per-seed detection rates, not forty-seed rejection rates.** Under the calibration model, their forty-seed rejection probabilities are about **96.7%** and **99.9999%**, respectively. The main unresolved issue is clinical coverage and model validity, rather than weak forty-seed power against those two particular synthetic scenarios.

**6. Operating characteristics and support for the real simulation**

I independently parsed all **720** candidate rows, recalculated T1, T2 and ladder scores, and applied the stated tie-break order.

- **84** candidates qualify.
- Selected parameters: **mean level, 150% growth, two-study floor, median late work, 10% late growth, stale cap three**.
- Ladder score: \((0.002+0.228)/2=0.115\).
- This matches the frozen rule and committed selection.

The selection arithmetic is correct. See [protocol.md:114](../../docs/calibration/backlog-5c3/protocol.md:114), [protocol.md:160](../../docs/calibration/backlog-5c3/protocol.md:160), and [oc-v0.2-eval-4.json:5247](../../docs/evaluation/oc-v0.2-eval-4.json:5247).

For \(q\), the probability that a synthetic seed is unstable, I recomputed:

\[
R_{40}(q)=P\{\operatorname{Binomial}(40,q)>4\}.
\]

| Scenario | Unstable seeds / 1,000 | Recomputed forty-seed rejection | Approximate 95% range from transforming reported binomial bounds |
|---|---:|---:|---:|
| Stable stationary | 0 | 0% point estimate | Upper bound about 0.000040% |
| Stable seasonal | 1 | 0.000000064% | Effectively near zero |
| Stable bursty | 5 | 0.000178% | 0.00000071–0.00997% |
| Stable seasonal, bursty | 38 | **1.724%** | **0.432–5.437%** |
| 5% capacity loss | 2 | 0.00000199% | Near zero |
| 10% capacity loss | 228 | **96.705%** | **92.892–98.661%** |
| Gradual erosion | 459 | **99.999885%** | **99.999335–99.999983%** |
| Stable seasonal, 92% utilization | 124 | **56.377%** | **40.563–71.371%** |

Underlying counts and bounds are in [oc-v0.2-eval-4.json:4422](../../docs/evaluation/oc-v0.2-eval-4.json:4422). These ranges describe Monte Carlo uncertainty under the synthetic model, not uncertainty about real-world applicability. Selection makes ordinary marginal intervals optimistic as validation statements.

Monte Carlo standard errors for the per-seed estimates are approximately:

- Cs: **0.60 percentage points**.
- L10: **1.33 points**.
- Erosion: **1.58 points**.
- B92: **1.04 points**.

The Cs point estimate meets the 5% forty-seed target, but its transformed upper bound exceeds 5%. Zero observed failures likewise does not establish zero risk.

The protocol’s post-run correction from “about 3.4%” to approximately **5.06%** is mathematically justified for the unchanged forty-seed criterion. Its timing remains documented rather than independently verifiable.

**Judgment:** the calibration supports a reproducible choice within a specified synthetic family. It does **not yet support the broader claim that this combined evaluator reliably accepts clinically sound real-simulator practices and rejects clinically harmful ones.**

**7. Canonical-runner enforceability and strategy fairness**

The canonical path does enforce several useful conditions:

- Approved policy and action-file digests.
- A no-action baseline.
- One named primary candidate; other candidates labeled exploratory.
- Complete plan-by-seed records through `Evaluate` and `NewRun`.
- Common horizon and observation end across the matrix.
- Action batches strictly before T90.
- Scenario matching and paired seed matching.
- Teleradiology-only appointment invariance.

See [evaluate.go:281](../../internal/eval/evaluate.go:281), [protocol.go:293](../../internal/eval/protocol.go:293), and [holdout_runner_test.go:92](../../internal/sim/holdout_runner_test.go:92).

The committed E3 non-remote actions match D2’s; its extra batch is urgent weeknight teleradiology. All committed development strategies have fixed action batches at takeover. I found no action-file adaptation to future realized cases. The actual reading selector uses published expected duration rather than the hidden realized read duration: [reading.go:149](../../internal/sim/reading.go:149).

Case-selection protections are meaningful but incomplete:

- Priority and modality margins are protected.
- Sufficiently populated intersections are checked.
- Rare intersections may remain unjudged.
- Payer/referral-source and access distributions are explicitly deferred.
- Same seed labels and hashes are not independent proof that arbitrary supplied records came from the claimed world.

**Judgment:** the runner is enforceable as a cooperative evidence-production workflow after H6 is fixed. It is not presently an exclusive execution boundary or durable run-once system.

D2 must remain exploratory even if its eventual result looks better. It must not replace a failed E3 primary after exposure.

**8. Statistical interpretation of the forty-seed protocol**

The rule is defensible as a **prespecified empirical acceptance rule**, provided the owner accepts its operating characteristics and its limited claim.

It does not demonstrate, at 95% confidence, that the population seed-failure probability is below 10%.

Independently computed one-sided 95% upper bounds are:

| Failures among forty | Upper bound on population seed-failure probability |
|---:|---:|
| 0 | 7.22% |
| 1 | 11.32% |
| 4 | 21.44% |

At a true failure probability of 10%, the rule accepts about **62.90%** of evaluations.

The implementation explicitly reports this distinction: [evaluate.go:126](../../internal/eval/evaluate.go:126), [stats.go:304](../../internal/eval/stats.go:304).

Patient-level independence is not required for a binomial model of independently generated **seed-level outcomes**. Within-seed clustering changes the seed-failure probability, however. The committed service calibration illustrates that sensitivity: at the planning breach rates, the independently computed forty-seed rejection rises from about **2.2%** in the independent model to about **97.2%** in the week-ICC 0.05 model.

The implementation calculates weekday-adjusted day and Monday-based week diagnostics and seed-clustered pooled-rate intervals: [stats.go:226](../../internal/eval/stats.go:226). These are informative diagnostics; they do not themselves alter acceptance or guarantee finite-sample coverage under every heterogeneous seed distribution.

Also, service and backlog have separate four-seed allowances. Four service-failing seeds and four different backlog-failing seeds could produce **eight seeds with some clinical failure**, while both plan-level allowances pass. Thus “90% jointly clinically reliable” is not the current rule.

**9. Financial correctness and remaining owner decisions**

The financial implementation is comparatively strong.

| Requirement | Code-reading judgment |
|---|---|
| Minimum cash | Scanned after every ledger posting, including intermediate same-time postings |
| Payment-date solvency | Payroll, payables and severance are paid when due, even into negative cash |
| Settlement observation | At least H+31 days, plus other observation requirements |
| Past-due obligations | Zero is justified by the current engine’s compulsory-payment mechanics |
| Receivables | Never treated as cash |
| Severance/contract commitments | Included in terminal committed liabilities |
| Remote in-flight fees | Included |
| Double counting | Accrued amounts, future minimum retainers and in-flight fees are separated |
| Financial GO | Requires positive absolute T90/T180 means, positive lower T90 incremental interval, and T180 validation |
| Transition/service purchase | Kept distinct from financial superiority |

Supporting implementation: [export_eval_test.go:147](../../internal/sim/export_eval_test.go:147), [finance.go:275](../../internal/sim/finance.go:275), [measure.go:334](../../internal/eval/measure.go:334), [compare.go:486](../../internal/eval/compare.go:486).

I found no supported strategy action that postpones an obligation already recognized by H beyond the fixed settlement cycle to evade the cash gate.

There are limits:

- `PastDueCents=0` is an engine invariant, not an independent dated-obligation audit in the generic record evaluator.
- Terminal funding excludes ordinary accrued liabilities not yet due and future wages from its hard coverage test.
- At H=150, observation ends at day 181. This is not thirty-one days of settlement after T180.
- Obligations arising from continued operation after H can therefore mature beyond observation.
- Passing finite-horizon solvency does not establish continuing financial viability.

These are material scope limits, not demonstrated payment-deferral defects in the current engine.

The committed development documentation reports E3’s T90 absolute profit as **−$19.5k**, incremental profit **−$23.2k**, operating-window increment **−$1.6k**, and T180 increment **−$10.8k**. D2’s reported operating-window increment is positive but its interval crosses zero. These figures support the owner’s terminology—E3 as a service purchase, D2 as an exploratory economic comparator—not financial GO. They were not independently reproduced: [reevaluation.md:19](../../docs/calibration/backlog-5c3/reevaluation.md:19).

Remaining owner decisions are:

1. What severe urgent/soon/routine delay is unacceptable even when aggregate breach rates pass?
2. What timely-access claim must the evaluation establish?
3. Is the 92%-utilization seasonal false-rejection risk acceptable for the intended practice?
4. Is acceptance empirical, or must it support a population-reliability confidence claim?
5. Should all clinical failures share one seed allowance, and should persistence be zero-tolerance?
6. What constitutes an affordable service purchase, including ongoing funding beyond the observation horizon?
7. Is cooperative protocol compliance sufficient, or is independent holdout custody required?
8. Does the identified mini-scenario holdout-seed use require an exposure declaration or replacement protocol?

**10. Minimal remediation plan**

Ordered to avoid consuming the holdout while resolving acceptance:

1. **Before any further ordinary suite execution:** replace real-holdout guard fixtures with synthetic fixtures and account for prior exposure. Do not derive or inspect holdout outcomes.
2. **Before freeze:** run the appendix witnesses in a disposable copy. Decide the clinical severity, access and age requirements. Preserve v4 unchanged; version any revised acceptance rules.
3. **Before freeze:** make refused-action behavior identical in development and canonical execution.
4. **Before policy acceptance:** fix synthetic arrival ordering; validate combined service/access/backlog behavior on fresh non-holdout cases, including near-capacity seasonality and clustered severe delays. Freeze the validation protocol before inspecting its results.
5. **Before unseal:** bind approval to exact source/build and artifact bytes; authenticate current owner approval; prevent mutation of authorized configuration.
6. **Before unseal:** add durable first-run registration, exclusive/atomic artifact publication and pre-execution reproduction checks.
7. **Before unseal:** freeze one primary and the comparator roles, the financial interpretation, uncertainty wording and failure-handling procedure. Correct the restoration predicate and observation documentation.

None of these steps requires executing a strategy or exposing holdout results during this audit.

**11. Explicit decisions**

| Proposed action | Decision |
|---|---|
| Freeze E3 now as the confirmatory primary | **NO GO** |
| Retain E3 as the provisional primary candidate | **GO**, with service-purchase interpretation |
| Keep D2 as an exploratory economic comparator | **GO** |
| Keep E2 archived | **GO** |
| Commit the unseal record now | **NO GO** |
| Execute the holdout now | **NO GO** |

**12. Verified versus reported**

| Item | Evidence status |
|---|---|
| Policy/strategy/set/history byte digests | Independently computed |
| Candidate selection and forty-seed probability transforms | Independently computed from committed OC counts |
| Synthetic clinical-witness arithmetic | Independently computed |
| Gate logic, recorder accounting, protocol bypasses | Verified by reading implementation |
| Appendix expected outcomes | Inferred from implementation; unexecuted |
| OC counts themselves | Committed results; not regenerated |
| E3/D2 development outcomes | Reported in committed documentation |
| Cross-machine reproduction, mutation-test success, full-suite success | Reported; test implementations inspected |
| Exact commit ancestry and pre-freeze chronology | Not verifiable from this archive |
| Absence of prior holdout exposure | Not established |
| Go test/simulator execution during this audit | None |

**Appendix A — Synthetic clinical acceptance witnesses**

Proposed filename: `internal/eval/zz_audit_clinical_blindspots_test.go`.

This uses the existing synthetic record helpers. It performs no simulation and does not load a holdout manifest. The seed labels are synthetic record identifiers.

**Expected current outcome:** all three subtests pass while asserting a `GO` verdict. That outcome would confirm the acceptance gaps, not clinical safety. These records demonstrate evaluator behavior; they do not prove that a particular permitted E3 action sequence generates them.

```go
package eval

import "testing"

func TestAuditClinicalBlindSpots(t *testing.T) {
	for _, mode := range []string{"report", "access", "age"} {
		t.Run(mode, func(t *testing.T) {
			p, digest := testPolicy(t)
			in := Input{
				Baseline: "A",
				Primary:  "B",
				Plans:    map[string][]Record{},
			}
			want := map[string]int{"report": 6, "access": 30, "age": 1}

			for seed := int64(1); seed <= 40; seed++ {
				in.Seeds = append(in.Seeds, seed)
				a := base("A", seed)
				a.breachPM = map[string]int{}
				in.Plans["A"] = append(in.Plans["A"], a.record())

				b := base("B", seed)
				b.breachPM = map[string]int{}
				b.profitPD = 30_000
				r := b.record()
				changed := 0

				for i := range r.Studies {
					s := &r.Studies[i]
					if s.Priority != "urgent" {
						continue
					}
					day := (s.AcquiredSec - t0) / daySec
					switch {
					case mode == "report" && day >= 112 && day < 118:
						s.FinalSec = s.AcquiredSec + 8*daySec
						s.ClaimedSec = s.FinalSec - 60
						s.ReadStartSec = s.ClaimedSec
						changed++
					case mode == "access" && day < 30:
						s.AcquiredSec += 30 * daySec
						s.EnteredSec += 30 * daySec
						s.ClaimedSec += 30 * daySec
						s.ReadStartSec += 30 * daySec
						s.FinalSec += 30 * daySec
						changed++
					case mode == "age" && day == 149:
						due := s.AcquiredSec + testTargets["urgent"]*60
						s.FinalSec = due + 14*daySec + 12*hour
						s.ClaimedSec = s.FinalSec - 60
						s.ReadStartSec = s.ClaimedSec
						changed++
					}
				}
				if changed != want[mode] {
					t.Fatalf("changed %d studies, want %d", changed, want[mode])
				}

				if mode == "access" {
					for i := range r.Orders {
						o := &r.Orders[i]
						if o.Priority == "urgent" &&
							o.ResolvedSec < t0+30*daySec {
							o.FirstApptSec += 30 * daySec
							o.ResolvedSec += 30 * daySec
						}
					}
					for i := range r.Captures {
						c := &r.Captures[i]
						for j, o := range r.Orders {
							switch {
							case o.ReceivedSec > c.AtSec:
								c.OrderOutcomes[j] = OutNotDisclosed
							case o.ResolvedSec > c.AtSec:
								c.OrderOutcomes[j] = OutOpen
							default:
								c.OrderOutcomes[j] = OutServed
							}
						}
					}
				}

				for i := range r.Samples {
					s := r.Samples[i]
					r.Samples[i] = b.sample(&r, s.Day, s.AtSec, b.money(s.AtSec))
				}
				in.Plans["B"] = append(in.Plans["B"], r)
			}

			rep := Evaluate(p, digest, in)
			pr := plan(rep, "B")
			c := comparison(t, rep, "B")
			if !pr.Valid || !pr.Eligible || !c.Valid || c.Verdict != GO {
				t.Fatalf("verdict=%s plan=%v comparison=%v reasons=%v",
					c.Verdict, pr.Integrity, c.Integrity, c.Reasons)
			}
			if pr.BacklogStable != 40 {
				t.Fatalf("stable seeds=%d", pr.BacklogStable)
			}
			if mode == "age" {
				bm := pr.PerSeed[0].Measures.Backlog
				if bm.Persistent != 0 || bm.WorstPastDueHr != 334 {
					t.Fatalf("sampled age result: %+v", bm)
				}
			}
		})
	}
}
```

The three expected acceptance witnesses are:

| Subtest | Harm encoded | Why current gates admit it |
|---|---|---|
| `report` | Six urgent studies each take eight days | Low aggregate breach rate; 1.714 mean backlog; zero late-work medians |
| `access` | Thirty urgent patients acquire imaging thirty days later | Same eventual service and broad-window acquisitions; unchanged reporting TAT |
| `age` | One urgent study is finalized 348 hours past due | Daily samples observe at most 334 overdue hours; low aggregate burden |

**Appendix B — Insufficient baseline classified as restored service**

Proposed filename: `internal/eval/zz_audit_service_restoration_test.go`.

**Expected current outcome:** passes, confirming that insufficient evidence is accepted as the basis for a service-purchase classification. This is a classification unit test, not a complete record evaluation.

```go
package eval

import "testing"

func TestAuditInsufficientBaselineCountsAsRestored(t *testing.T) {
	p, _ := testPolicy(t)
	cand := &PlanReport{
		Valid:          true,
		ServiceStatus:  Pass,
		ServiceJoint:   JointGate{Status: Pass},
		BacklogStatus:  Pass,
		SolvencyStatus: Pass,
		FundingStatus:  Pass,
	}
	baseline := &PlanReport{Valid: true, ServiceStatus: Insufficient}
	c := Comparison{
		Valid: true,
		Profit: map[string]Interval{
			WinT90: {N: 40, Mean: -100},
			WinOps: {N: 40, Mean: -10},
		},
	}
	verdict(p, &c, cand, baseline)
	if !c.ServiceRestored ||
		c.Verdict != CONDITIONAL ||
		c.Classification != ClassServicePurchase {
		t.Fatalf("%+v", c)
	}
}
```
**Appendix C — Maintainer verification (after the audit, not part of the auditor's read-only findings)**

All checks were run on 2026-10-18 against 7707dbd, in a disposable copy extracted from `git archive 7707dbd` and kept outside the repository. No repository file was changed. The witness files were not committed.

| Check | How | Result |
|---|---|---|
| Appendix A `report` (six urgent reports eight days late) | `go test ./internal/eval -run TestAudit -count=1` | **PASS**: verdict GO, 40/40 seeds backlog-stable. H1 confirmed. |
| Appendix A `access` (thirty urgent acquisitions thirty days later) | same | **PASS**: verdict GO, 40/40 backlog-stable. H2 confirmed. |
| Appendix A `age` (report 348 h past due) | same | **PASS**: verdict GO. The worst sampled past-due age is 334 h, with no persistent study, so the age gap in H1 is confirmed. |
| Appendix B (insufficient baseline counted as restored service) | same | **PASS**: `ServiceRestored`, CONDITIONAL GO, service purchase. M3 confirmed. |
| H6 (refused actions) | reading [`holdout_runner_test.go`](../../internal/sim/holdout_runner_test.go) `strategyTail` and `runStrategy` against [`strategy_dev_test.go`](../../internal/sim/strategy_dev_test.go) | Confirmed. The canonical runner skips a refused action without reporting it; the development runner returns an error. |
| H3 (real holdout seed in an ordinary test) | reading `TestHoldoutRecordersGuarded` in [`holdout_protocol_test.go`](../../internal/sim/holdout_protocol_test.go) | Confirmed. The test loads the real holdout manifest and builds a **mini-scenario** world on its first seed, run up to takeover only. It records no outcome and prints no seed, and the recorder refuses the seed as intended. The exposure is accounted for in `docs/evaluation/holdout-exposure-5c4a.md` (Slice 5C.4a). |

The other findings were not re-executed: H4, H5, M1 (including the arrival-ordering defect), M2 and L1.
