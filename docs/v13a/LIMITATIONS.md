# V0.13a known limitations and safety

Status: V0.13a (0.13.0-dev.1), design only. These limitations apply to
any comparison built on this design. Most cannot be removed by better
presentation.

## 1. Safety

**Irreversible terminations.**
* A termination cannot be moved or withdrawn once committed.
* No action rehires the employee.
* A comparison shown before the decision is the only point at which
  the alternatives are still open.
* After commit, the record can describe the consequences but cannot
  undo them.

**Displaced bookings.**
* The count is exact, and it belongs to the whole batch.
* Whether each patient accepts a replacement is decided at commit by a
  hidden attribute.
* Neither a displaced booking nor a lost one is a measure of patient
  harm. The engine does not model clinical outcomes.
* A termination that displaces nothing can still remove capacity that
  was never booked. The preview does not report this (G1 in
  `TERMINATION_VS_KEEP.md`).

**Retained obligations.**
* Keeping an employee keeps the weekly guarantee and the severance
  exposure. It is never shown as cost-free.
* A hire and a teleradiology start create obligations that last beyond
  the decision. The engine reports them as guaranteed.

**Credit-supported commitments.**
* Under a credit line, a commitment may be funded by credit, not cash.
* The funding test says whether the commitment fits the headroom. It
  does not say whether borrowing is wise.
* A comparison must show the funding answer as the engine gives it,
  with no judgment.

**Human review.**
* No comparison result, projection or experiment from this design
  authorizes a management decision in a real practice.
* Before any later milestone draws conclusions, a person must review:
  * the presentation template;
  * the candidate rule;
  * the scripted cases;
  * every reported result.
* The V0.12b 98-item review is still pending. V0.13 does not replace
  it.

**No substitution of decisions.**
* The system never selects, defaults to, or reorders by merit any
  alternative.
* If the agent's reply is invalid, nothing is submitted.

## 2. Evidence and method

**Horizon dependence.**
* Money differences between alternatives recur weekly, and capacity
  effects begin at end dates.
* A projection's sign and size can change with the horizon.
* Every projection names its horizon.

**Counterfactual uncertainty.**
* A class B projection is exact for one seed and one continuation.
* It is not what would happen.
* The no-further-action continuation is unrealistic.
* No causal claim beyond the simulator is made.

**Oracle information.**
* Projections use the simulator's realized future.
* Showing them to an agent would give it information no manager has.
* They are evaluation material only (`COMPARISON_CONTRACT.md` §5).

**Simulation-model limitations.** The engine does not model:
* staff morale, workload, resignation, sickness or absence: an hourly
  contract ends only by termination;
* clinical outcomes, patient harm, reputation or referrers' reactions;
* labour law beyond its own notice, severance and guarantee rules;
* taxes or financing beyond the scenario's credit line.

**Deterministic replay.**
* Branches are exact only on the same engine and only for records under
  this build's rules.
* Records under earlier rules cannot be branched (`currentRules`).
* No approximation is offered for them.

**Evidence provenance.**
* Every comparison records:
  * the engine;
  * the scenario fingerprint;
  * the seed;
  * the decision-point record checksum and digests;
  * the candidate rule;
  * every engine request with its response SHA-256.
* A projection records each branch's digests at the decision instant
  and at the horizon.
* Nothing is accepted without a digest match.

**One example.**
* The values in these documents come from one decision point (`mini`,
  seed 52) and one pair of branch runs.
* They show what the engine can report.
* They are not evidence about terminations, about agents, or about
  decision quality.

**Not yet verified.**
* Whether an accepted action with no material effect changes any
  reported metric through the accrual point
  (`CONTINUATION_DESIGN.md` §3). V0.13b must test this.
