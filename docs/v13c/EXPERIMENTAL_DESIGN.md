# V0.13c experimental design: management alternatives at the decision

**Status:**
* Design, prepared for owner review. No live inference is authorized,
  and none was run.
* The engine side of both arms is implemented and tested
  (`internal/alternatives/arms.go`).
* The agent side, a workflow that sends the turns to a model, is not
  implemented. That is V0.13d's work, after the protocol is locked.

## 1. The question

From the same decision context and the same initial proposal: when a
neutral comparison of engine-reported alternatives is shown before the
final decision, does the decision respond to the actual differences
between the alternatives?

**For example:**
* Does an agent decide differently when a termination closes 2,481
  scheduled slots than when it closes 362?
* Does it decide differently when a termination displaces 18 bookings
  rather than none?
* Does it recognize that keeping an employee carries the weekly
  guarantee and future severance?

**The primary outcome is decision sensitivity:** how much the decision
moves between two decision points that differ in one engine-reported
fact, in treatment, beyond how much it moves in control (§6).

**What it does not establish:**
* A change of decision is not an improvement.
* Nothing here scores a decision economically or clinically.
* An agent's stated reason is recorded and never read as evidence of its
  reasoning.

## 2. Arms

Both arms are the same validation-first workflow, from the same recorded
or constructed context and the same initial proposal. They differ only
in the content of one final-decision turn.

| | A: control | B: treatment |
|---|---|---|
| Before the turn | llm-agent 5's validation-first workflow (`llm-agent-validation-first-workflow/v1`, `a0204f62…`): validation, `llm-agent-validation-feedback/v1` on a refusal, one correction | the same |
| When the turn is due | the reply is split as the agents split it (`SplitReply`, as `llm_agent3.split`); rule v2 applies to its staffing batch, and the engine accepts both its operations batch and its staffing batch. One test, made once (`Steps` on the whole reply), decides both arms | the same test |
| The turn | `dreamteam.alternatives-control/v1` (`331a1542…`): the proposal, its validation ("accepted"), and the reply options `select:proposal`, `revise`, `abandon` | `dreamteam.alternatives-presentation/v1` (`5bf586cf…`): the summary of the agent-audience comparison, then on `inspect:<id>` that alternative's detail; reply options `select:<id>` (any listed), `revise`, `abandon` |
| Turn not due | nothing more: the validated proposal is submitted, as validation-first does | the same |
| After the reply | final validation of what would be submitted; submitted only if accepted | the same |
| Classification | `ClassifyControl` | `Classify` (V0.13b) |

**Why the control has a turn (owner decision D1).** The V0.12b design
compared a workflow with an extra model turn against one without, so its
B − A difference mixed the information shown with the extra turn itself.
Here both arms get one turn at the same points, with the same final
validation and the same decision grammar (select, revise, abandon), so
the difference between them is mainly the turn's content. The framing is
not identical: only the control text says "Nothing has been applied
yet"; the treatment summary does not (V0.13c review F7). Aligning the
wording would change a pinned template, so it is an owner decision (D8)
and a matter for the human review. The cost is that
control is no longer llm-agent 5 unchanged: it is validation-first plus
a fixed confirmation turn. The alternative, llm-agent 5 unchanged as
control, is listed in §8.

**Model calls are not equalized.** Treatment may make one inspect
request before deciding, which costs one call (§5). Calls are counted
per draw and reported beside every outcome.

## 3. The information boundary, in code

Treatment may be sent only the turn's `Message` and its `Inspect`
answers. Both are built by `alternatives.Steps(s, reply)` from the
decision-point session alone:
1. `SelectV2` at the session's time (only reads the session).
2. `BuildAgent`: the agent audience only. Its Go type has no field for a
   projection (V0.13b, E3). The evaluation audience, the branch builder
   and every realized outcome are never built.
3. `Encode`, then `DecodeAgentAt(raw, s, sel)`:
   * unknown fields refused;
   * only classes A1, A2 and C allowed;
   * every class A value's source must be a decision-point request;
   * the comparison rebuilt from the session and required equal, byte
     for byte (V0.13b F2).
4. `Summary(bound)` and `Detail(bound, id)` from the pinned template.
5. `checkShown`, a second guard on every text the step would send:
   * no run of 16 or more hexadecimal characters, in any case
     (digests, seed digests);
   * none of: seed, digest, sha256, branch, projection, projected
     orders, projected outcome, realized, evaluation, continuation,
     orders lost, lost orders, profit, future outcome, hidden.
   * A match makes `Steps` fail; nothing is returned.
6. The session's digests are required unchanged.

**What treatment can see:**

| Class | Content |
|---|---|
| A1, reported | decision-point engine facts: validation, the staffing preview (displacements, guarantees, severance, contract pay, voided rosters and extensions), slots closed (`dreamteam.slots-closed/v1`), observation (payroll, payables, staff) |
| A1/A2 estimates | engine-labelled estimates (contract cost no longer owed per week, the retainer run rate, read fees at trailing demand) |
| A2, derived | existing obligations under no action (the weekly guarantee, severance if terminated later), each with its derivation |
| A2, engine rule | reversibility |
| | the alternatives listed and their validation |
| C, unavailable | explicitly unavailable items: slots beyond the booking horizon, demand for closed slots, future orders and revenue, what is not modeled |

**What the code never adds to a text:**
* realized future orders, lost demand or profit;
* projected branch outcomes or the evaluation audience;
* hidden simulator state;
* seed identifiers or digests.

**The threat model, and what it does not cover (V0.13c review F1).** The
guards above are structural (only the bound agent audience is rendered)
and lexical (the text check). Neither can tell what a free-text field
means. A field the engine accepts and the template prints, such as an
action's `requestId` or an employee's name in the scenario, can carry any
text, an evaluation-only value written in words included. The reviewer
showed this with a valid `requestId`. So the boundary holds only if those
inputs are trusted:
* **Recorded proposals** were written by a model that never saw any
  evaluation value; they are bound to their committed context lines by
  SHA-256.
* **Scripted proposals and the scenario** are committed files, pinned
  at lock.
* **Live replies** come from the model itself, which has only what it
  was shown.

Whether this trusted-input model is acceptable, or the free-text fields
must also be constrained or replaced by neutral identifiers in the turn,
is the owner's decision D7. It is a policy decision, not a code fix,
and the protocol cannot be locked before it.

`TestArmsBoundary` and `TestReviewF1DigestAnyCase` check the structural
and lexical guards. They build the evaluation audience at the same
decision point and require that:
* none of its projected labels appears in any treatment text;
* no record checksum, seed digest, state or run digest, or comparison
  digest appears;
* the text check runs on the summary, on every inspect answer and on
  the control text;
* the step type holds no evaluation value.

**What a driver must do (V0.13d):**
* Send `Message`, and the requested `Inspect` answers, verbatim and
  nothing else from the step.
* Record `Rule`, `TemplateSha256`, `ComparisonSha256` and
  `MessageSha256` in the decision record, never in a message.
* Call `eval.GuardManifest` before opening a session, as every
  evaluation driver does. `Steps` itself never sees a seed.
* Prompt instructions are not relied on for the boundary.

## 4. Decision contexts

There are two kinds, both listed in the draft protocol.

**Historical branch points** ([BRANCH_POINTS.md](BRANCH_POINTS.md)):
* the V0.12b pool of 20 recorded staffing proposals with committed
  contexts, opened on the engine at the recorded state;
* the recorded request and conversation;
* the recorded initial proposal as the model's first reply, as in
  V0.12b.

Nine points are selected by `dreamteam-v13c-select/1`, which reads no
model response.

**Constructed sensitivity pairs** ([SENSITIVITY_CASES.md](SENSITIVITY_CASES.md)):
* six pairs of real engine states (mini, seed 52, reached from the start
  with no action);
* each with a scripted proposal;
* the agent's prompt built at that state by the agent's own `prepare`,
  with empty memory;
* the scripted proposal as the first reply.

No model made these proposals. They are constructed contexts, labelled
as such, and are not historical behaviour.

## 5. The turn and the reply

**Treatment:**
* The summary is sent.
* The model may reply with an inspect request for one or more listed
  IDs. The answer holds those details, in the order requested, as one
  message.
* At most one inspect round; a second is answered with the decision
  request only.
* Then one decision reply: `select:<id>`, `revise` with a batch, or
  `abandon`, with a reason.

**Control:**
* The control text is sent.
* One decision reply: `select:proposal`, `revise` with a batch, or
  `abandon`.
* An inspect request is recorded and answered with the decision request.

**Both arms:**
* The decision reply's JSON form is that of the agents' final response.
  Its exact schema is fixed by V0.13d's workflow, with its own identity
  and SHA-256, before lock.
* An unreadable reply is handled by the workflow's one retry
  (`maxRetries` 1), then classified `no_decision`.
* The operations actions of the reply (llm_agent3.split's operations
  batch) are validated, and the turn is due only if they are accepted
  too. The comparison is of the staffing batch only. Whether abandoning
  or selecting no action also withholds the operations batch is the
  owner's decision D9.

## 6. Observable behaviour

These are defined before any request, per draw.

**Results** (`Classify` for treatment, `ClassifyControl` for control):

| Result | Meaning |
|---|---|
| `retained` | the proposal submitted unchanged (select:proposal, or a revision equal to it), after final validation |
| `selected` | treatment: another listed alternative submitted |
| `selected_no_action` | treatment: no_action selected; nothing submitted |
| `revised` | a new batch, different from every listed alternative, submitted after final validation |
| `abandoned` | abandon; nothing submitted |
| `invalid_alternative` | an unlisted or refused alternative, or a batch the final validation refuses; nothing submitted |
| `no_decision` | no usable or a contradictory reply; nothing submitted, never defaulted |

**Flags:**
* `inspected`: an explicit inspect request for a listed alternative
  other than the proposal. Reading the summary is not inspection.
* `compared`: the reply's structured citations name item values that
  equal the shown values, for two or more alternatives, at least one
  not chosen, with a difference among them. At most three entries,
  fewer than the items shown, and not every compared row: copying the
  table does not count (V0.13b `verifyCitations`).
* `no_observable_comparison`: neither.
* The explanation text is never parsed for comparison or reasoning.

**Shared outcomes, defined for both arms** (owner decision D10):
* **departure**: anything but `retained`. This includes
  `invalid_alternative` and `no_decision`, which are also reported
  separately, so a parsing failure is never read as a deliberate change;
* **no action submitted**: `abandoned` or `selected_no_action`
  (`NoActionSubmitted`);
* **revised**, which is not arm-neutral: the same new batch is
  `revised` in control and `selected` in treatment when it equals a
  listed alternative, so the two are reported together as "submitted a
  different batch";
* **selected another alternative** (treatment only).

**The primary outcome, decision sensitivity.** For a sensitivity pair
(a, b), per model:

> Δ_arm = P_arm(departure | b) − P_arm(departure | a)
>
> sensitivity = Δ_treatment − Δ_control

A positive or negative value means the comparison changed how much the
decision follows the difference between the two contexts. No pair
isolates one fact (SENSITIVITY_CASES.md §3), and the difference in
differences removes only what the control turn also shows, not the other
listed co-differences or their interactions with treatment. The estimand
is sensitivity to the named pair of contexts, with its co-differences
named (V0.13c review N1). It does not mean the decision improved. The
same is computed for "no action submitted" as a secondary outcome.

**At historical points:** P_treatment(departure) − P_control(departure)
per point, and the flags, as secondary outcomes.

## 7. What differs between the arms, besides the comparison

* **The turn's text.** Control names the proposal and its validation;
  treatment shows the alternatives and their facts. Both have the same
  reply options, apart from treatment's extra `select:<id>` and inspect.
* **Calls.** Treatment may use one more call, for inspect. Counted,
  never equalized.
* **Framing and length.** Only control says "Nothing has been applied
  yet" (D8). The treatment text is longer and lists more alternatives.
* **"No action":** control's `abandon`, or treatment's `abandon` or
  `select:no_action`. Both count as "no action submitted".
* **Points where the turn is not due.** The arms are identical there:
  the same workflow, no turn. They are A/A controls: any difference
  there is noise or nondeterminism, not the comparison.

## 8. Decisions for the owner before lock

| # | Decision | Recommendation |
|---|---|---|
| D1 | Control: validation-first with the matched confirmation turn (this design), or llm-agent 5 validation-first unchanged (no turn) | the matched turn: it isolates the comparison's content from the extra turn |
| D2 | `compared` is observable only if the reply can cite item keys. The presentation v1 template shows labels, not keys, and does not describe a citation field, so `compared` would almost never be observed | a presentation v2 that shows each item's key and one neutral sentence describing an optional citation field. It needs the human review (PRESENTATION_REVIEW.md) and a new pin; otherwise report `compared` as not measurable |
| D3 | Draws and budget (draft protocol §5): Gemma 10 and Qwen 20 per arm per unit, 1 per arm at the two A/A points; 964 draws, at most 4,820 calls, an 8-hour stop | as drafted, or fewer draws for Gemma, which gave one decision per arm in every earlier experiment |
| D4 | Constructed sensitivity contexts: prompts built at a scripted state, with empty memory and a scripted proposal | accept, labelled as constructed; no historical context differs in one fact |
| D5 | The engine version for the run: V0.13d will add the agent workflow and bump the version, and the lock pins it | pin at lock |
| D6 | Inspect: one round of any number of IDs (this design), or full detail sent with the summary (no inspect, no `inspected` flag) | one round |
| D7 | The information boundary for free-text fields (§3): accept the trusted-input model (recorded, committed and pinned inputs; live replies from the model itself), or constrain or replace displayed free-text fields such as `requestId` and names | accept the trusted-input model, stated as such in the protocol, with every input pinned at lock; reconsider for any input that is not a recording, a committed file or the model's own reply |
| D8 | Framing: give both arms the same "Nothing has been applied yet" sentence (a new presentation identity and pin), or record the difference | the same sentence in both, decided together with D2 and the human review |
| D9 | Operations batch: when the decision is abandon or no action, is the reply's operations batch still submitted | submitted unchanged in both arms; the shared outcomes concern the staffing batch |
| D10 | The shared outcomes (§6): departure including failures, reported with failures separately; "submitted a different batch" in place of raw `revised` | as stated in §6 |
