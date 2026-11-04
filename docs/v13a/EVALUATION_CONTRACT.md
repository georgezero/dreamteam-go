# V0.13a evaluation contract

Status: **proposed** in V0.13a (0.13.0-dev.1). There is no experiment,
no protocol and no model request. A later experiment (V0.13c) would need
its own protocol, locked and approved by the owner before any live
inference.

**This document defines:**
* what a decision record must contain;
* how outcomes are classified;
* what counts as observable evidence that the agent compared
  alternatives.

**It never infers internal reasoning from the model's explanation.**

## 1. What is recorded at each decision point

* the comparison shown (`dreamteam.management-alternatives/v1`,
  audience `agent`) and its SHA-256;
* every request the agent made after seeing it, including any
  `inspect:<id>` (§2);
* the reply, exactly as received, and its parse;
* the engine's revalidation of the batch about to be submitted;
* what was submitted and the engine's result;
* after the decision, for evaluation only, optionally the class B
  projections of every listed alternative under the declared
  continuation.

Every record replays without a model, from the recorded replies. The
comparison and the candidate list must be rebuilt identically.

## 2. Making inspection observable

The comparison is delivered in two levels:
1. **A summary.** For each alternative: its ID, actions, validation,
   and the count of facts, estimates and unavailable items.
2. **The detail.** Delivered only when the agent requests it, with
   `inspect:<id>`. Each request is recorded.

Inspection is then an action in the record, not an inference.

**Accepted cost:**
* An agent that never inspects decides from the summary alone.
* An arm that shows full detail without inspection is a different
  treatment, and must be declared as such.

## 3. Outcome classification

Each decision point gets exactly one **result**, from the reply and
what was submitted:

| Code | Result | Evidence |
|---|---|---|
| `retained` | Original proposal retained | `select:proposal`, and the proposal was submitted unchanged |
| `selected` | Another listed alternative selected | `select:<id>`, with `<id>` not `proposal` and not `no_action`; that batch submitted |
| `selected_no_action` | The no-action alternative selected | `select:no_action`; nothing submitted |
| `revised` | Proposal revised | `revise` with a new batch that differs from every listed alternative |
| `abandoned` | Proposal abandoned | `abandon`; nothing submitted |
| `invalid_alternative` | Invalid or unavailable alternative | the reply names an ID that is not listed, or one the engine refused; or the final revalidation refuses the batch. Nothing is submitted for that reply. |
| `no_decision` | No usable reply | missing, unparseable or contradictory reply after the allowed retries; nothing submitted |

A `revise` whose batch equals a listed alternative is classified as
`selected` for that alternative, or `retained` if it equals the
proposal. The reply's label does not decide the result.

Each decision point also gets **behaviour flags**, independent of the
result:

| Flag | Meaning | Evidence |
|---|---|---|
| `inspected` | At least one alternative other than `proposal` was inspected | a recorded `inspect:<id>` request, with `<id>` not `proposal` |
| `compared` | A verified stated comparison (§4) | the reply's `comparison` field passes every check in §4 |
| `no_observable_comparison` | Neither of the above | no inspection of another alternative and no verified comparison |

`no_observable_comparison` is the record's "comparison ignored". It
says only that nothing observable shows a comparison. It does not say
what the model "thought".

## 4. What counts as a stated comparison

The reply may include a structured `comparison`: a list of at most
three entries:

```json
{"item": "<item key from the comparison>",
 "values": {"<id>": <value>, "<id2>": <value>}}
```

**It counts as `compared` only if all of the following hold:**
1. Each item key exists in the comparison shown.
2. Each entry names at least two alternatives. One of them is not the
   alternative finally chosen.
3. Each value equals the shown value exactly.
4. At least one entry's values differ between the alternatives named.
5. The entries cite at most three items, fewer than the items shown.
   An entry list that repeats the whole table does not count.

**What `compared` is not:**
* It is evidence that the agent could cite a difference correctly.
* It is not evidence that the difference caused the decision.
* A free-text reason is recorded but never scored.

## 5. Evidence that comparison affects decisions

A citation can be copied. The stronger, observable evidence is
**decision sensitivity**. It is designed for scripted cases first, with
no live model:
* Pairs of decision points that are identical except for one compared
  fact. For example:
  * displaced bookings 0 versus 100;
  * severance $3,950 versus $39,500;
  * a later-last-day alternative available versus not.
* The fact is set from real engine states where possible. Otherwise
  the case is labelled scripted.

**What sensitivity is:**
* The difference in the distribution of results between the two
  members of a pair.
* A model whose results do not change when a compared fact changes
  shows no observable use of that fact. This holds whatever its
  citations say.

**What would be reported in a later live experiment (V0.13c):**
* per arm:
  * the result distribution;
  * the flag rates;
  * the rate of irreversible actions submitted;
* paired sensitivity per fact.

The arms would be:
* control: the validation-first workflow, with no comparison;
* treatment: comparison with audience `agent`.

**No outcome is called a better decision.** The class B projections of
chosen alternatives are described under the declared continuation and
horizon, as consequences, never as quality.

## 6. What this contract does not allow

* Scoring the model's explanation.
* Counting a citation as reasoning.
* Calling `retained` a failure, or `selected` a success.
* Pooling results across continuations, horizons, engines or prompt
  versions.
* Repeating or substituting draws after seeing results (as in V0.12b).
