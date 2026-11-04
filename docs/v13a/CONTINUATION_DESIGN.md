# V0.13a continuation design

Status: V0.13a (0.13.0-dev.1), design only. Nothing here was
implemented or run beyond the mechanics check in
`BRANCHING_FEASIBILITY.md` §3.

A projection (class B in `COMPARISON_CONTRACT.md`) is exact only for a
declared continuation: what happens after the decision point. This
document proposes one fixed continuation for later testing. It also
states what makes paired branches comparable, and what does not.

## 1. Proposed continuation: `dreamteam.continuation/no-further-action-v1`

| Element | Value |
|---|---|
| Starting state | The decision point's record, rebuilt exactly (`BRANCHING_FEASIBILITY.md` E1). Every branch's digests at the decision instant must be equal. |
| Scenario and seed | The decision point's, unchanged. Holdout seeds are refused. |
| Engine | One engine for every branch: the build that rebuilt the decision point. |
| Alternative | The alternative's batch, applied at the decision instant, or no batch for `no_action`. |
| Later decisions | **None.** No management action of any kind after the decision instant. The policies in force continue. |
| Horizon | A common horizon for all branches at the point: `until = max(at + 30 days, latestEffect + 30 days)`. `latestEffect` is the latest effect date of any branch (`World.LatestEffect`): an end date, effective date or start date. The 30 days follow the evaluation policy's `daysAfterLatestEffect`. |
| Metrics | A fixed list (§2), read at `until` from each branch's result over the window from the decision instant. |
| Identity | The continuation ID and its SHA-256, recorded in every projection. |

**Why "no further action":**
* It is the only continuation that needs no model and no policy.
* It cannot make a later decision differ between branches.

**Its cost:** it is unrealistic. A real practice keeps deciding. Every
projection under it must say so in its label.

### Possible sensitivity continuation (not proposed for V0.13b)

**The option:**
* An existing scripted policy (`dreamteam agent -policy greedy` or
  `noop`) could drive both branches after the decision instant.

**The problem:**
* The policy reacts to each branch's own state.
* Later decisions would then differ between branches.
* The difference would mix the alternative's effect with the policy's
  response to it.

If ever used, it needs its own declared identity, and its results must
never be pooled with the no-further-action projections.

## 2. Metrics

Each metric is read from `session.Result` over the window from the
decision instant to `until`. Each has the unit and definition the
engine already uses. No new metric is defined.

| Group | Metrics |
|---|---|
| Money | net revenue; wages; benefits; severance; operating profit (as the result reports it); cash at `until`; lowest cash; days below zero |
| Demand | orders received; booked; lost; cancelled |
| Service | exams performed; wait to exam (50th and 90th percentile); reports finalized; reporting backlog at `until` |
| Obligations | bookings ahead at `until`; payables at `until` |

**Rules:**
* No metric is combined with another.
* No metric is called a gain, a loss, a benefit or a harm.
* A difference is reported as "B minus A under this continuation and
  horizon", with both values.

## 3. Randomness

**How the engine draws:**
* The simulator has no global random stream.
* Every random attribute is a pure function of the seed, a domain name,
  and keys that do not depend on player actions: `rng.Hash(seed,
  domain, keys…)` (`internal/rng`, `internal/sim/draws.go`).

**The domains:**

| Domain | Keys | Depends on actions? |
|---|---|---|
| `demand.count` | day, source, hour block | no |
| `order.core`, `order.ext`, `patient` | the order's origin (day, source, block, index) | no |
| `walkin.count`, `walkin` | day, block, index | no |
| `attempt` | origin, booking attempt number | the attempt number does (below) |
| `attempt.exam`, `registration` | origin, attempt number | the attempt number does |
| `read` | origin | no |
| `claim`, `claim.lag` | origin | no |

**What follows:**
* An action cannot change which number a key receives.
* Paired branches therefore share every exogenous draw:
  * the same orders arrive, in the same blocks, with the same
    patients, exams, priorities, payers and tolerances;
  * the same walk-ins arrive.
* The mechanics check confirmed this: both branches received the same
  2,224 orders.

**Where the branches can still diverge after an action:**
1. **Whether a draw is used at all.** An exam's duration is drawn only
   if the exam happens. A lost order consumes none of its later draws.
2. **The attempt number.** A patient who cancels and rebooks starts a
   new attempt with new draws. If a branch changes whether or when a
   patient books, that patient's later attempts can differ. (A
   displacement replacement keeps the original attempt by design, so
   displacement itself does not shift draws.)
3. **Context.**
   * The same uniform number applies to a different situation in each
     branch.
   * For example, the no-show probability depends on the booking lead,
     and a cancellation's lead time is measured from a different slot.
4. **Accrual points.**
   * An accepted staffing batch accrues the interval metrics at the
     decision instant (`beforeChange`). A no-action branch does not.
   * Whether this alone changes any reported value has not been
     checked. V0.13b must test it with an accepted action that changes
     nothing material.

## 4. Are paired branches comparable?

**Yes, as paired simulations of one realization.**
* Both branches face the same exogenous world.
* Each difference between them follows from the alternative, through
  the engine's own mechanics.
* This is a common-random-numbers design built into the engine.

**What a paired difference is not:**
1. **Not a general effect.**
   * One seed is one realization.
   * A claim about "the effect of terminating" needs many decision
     points and seeds, and an analysis declared in advance.
2. **Not independent of the continuation.**
   * Under any other continuation the difference can be different, in
     size or in sign.
3. **Not a causal identification for the real world.**
   * Within the simulator, the branch difference is exactly the
     difference this engine produces.
   * Reading it as a cause outside the simulator assumes that the
     simulator models every relevant mechanism. It does not (§5).
   * No such assumption is made.
4. **Not information the agent had.**
   * The projection uses the realized future.
   * It is evaluation material (`COMPARISON_CONTRACT.md` §5).

**What V0.13b must check before relying on any projection:**
1. **Identity.** With the same alternative in two branches, every
   digest is identical at `until`.
2. **No-op sensitivity.** An accepted action with no material effect
   must leave every reported metric equal, or the difference must be
   explained (accrual, §3).
3. **Replay.** Every branch saves, replays and verifies, on arm64 and
   amd64.
4. **Common exogenous draws.** Orders received and walk-ins are equal
   across branches at every point tested.

## 5. Known limits of any projection

* **Single realization:** one seed and one decision point.
* **Unrealistic continuation:** no further action.
* **Horizon dependence:** a termination's effects begin at its end
  date. Some of its costs and savings recur weekly, so the money
  difference grows or shrinks with the horizon.
* **Unmodeled consequences:**
  * staff morale, workload and resignation;
  * clinical outcomes and patient harm;
  * reputation and referral behaviour;
  * labour law beyond the engine's notice and severance rules.
