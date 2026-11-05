# V0.13b presentation template: `dreamteam.alternatives-presentation/v1`

Status: implemented in 0.13.0-dev.2, `internal/alternatives/present.go`.

**The template:**
* `internal/alternatives/text/presentation-v1.tmpl` is embedded in the
  build.
* It is pinned by SHA-256 `5bf586cf12c83c157a8f31db05cd8f5cc68c0503b3bf99b1a89102198313b89f`.

**It is not shown to any model.** No agent uses it, and no protocol
request returns it. A later milestone may show it only under an
owner-approved, locked protocol.

## 1. Two levels

1. **Summary** (`Summary(agent)`). For each alternative:
   * its ID;
   * its actions as JSON;
   * the engine's validation;
   * the number of facts, estimates, obligations and unavailable items.

   The summary ends with the replies allowed.
2. **Detail** (`Detail(agent, id)`). The full sections of one
   alternative, delivered on `inspect:<id>`. Inspection is therefore an
   observable request, not an inference.

## 2. Sections, identical for every alternative

* Action
* Description
* Validation
* Engine facts
* Engine estimates
* Obligations that continue
* Reversibility
* Not available before deciding

**How each section is filled:**
* An empty section says "none" or "none listed".
* Each line is the value's label, its value, then its `when`, its
  `basis` and, for A2, its `derivation`.
* A refused alternative shows the engine's codes and messages, and its
  sections hold nothing.
* No action shows the obligations that continue. It never says "no
  cost".

## 3. Neutral language

* The order is the rule's: the proposal first, then no action, then the
  others. The text says that the order means nothing and that no
  alternative is a default.
* There are no adjectives of value and no recommendation. A test refuses
  recommend, best, better, worse, optimal, safe, risky, costly, saving
  and should anywhere in the rendered text.
* The text never says that more information improves a decision.

## 4. Fidelity tests

* **No number in the template text.** The template holds no digit
  outside its identifying comment, so it cannot show a number of its
  own.
* **Every number traced.** Every number token in the rendered summary
  and detail is a whole number token of an identified field of the
  agent comparison. The fields include:
  * a formatted value;
  * its `when` or basis;
  * the actions' JSON;
  * the engine's messages;
  * the decision time;
  * the structural counts.
* **Mutation check.** A template with a hard-coded number, or with an
  adjective of value, fails the test.

**Money** is shown as dollars and cents from integer cents. Counts and
days are integers, and dates are shown as the engine gives them.

The rendered termination case is
`docs/benchmark/v13b-dev/termination-presentation.txt`.
