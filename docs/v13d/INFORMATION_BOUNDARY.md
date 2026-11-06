# V0.13d information boundary: the typed view (`dreamteam.alternatives-view/v1`)

**Owner decision D7:** use neutral identifiers and an allowlisted typed
presentation. Do not let arbitrary text reach the agent-facing
comparison. Do not use a general text sanitizer.

**Code:** `internal/alternatives/view.go` (the view and its closed
tables), `turn.go` (rendering), `text/presentation-v2.tmpl` (the fixed
text). **Tests:** `turn_test.go`.

## 1. What changed from V0.13c

In V0.13c, the treatment text was rendered from the agent audience of the
comparison (`dreamteam.management-alternatives/v1`). That comparison holds
free text: employee names and IDs, hire request IDs, plan and contract
IDs, the engine's own labels and bases (which embed names), and the
engine's validation messages. The V0.13c review (F1) showed that a valid
`requestId` carries any text into every turn.

In V0.13d a model is never shown the comparison. It is shown a **view**,
made from the comparison after `DecodeAgentAt` has bound the comparison
to the decision point, and holding only typed fields.

## 2. What a view can hold

| Field | What it is | Where the shown text comes from |
|---|---|---|
| Neutral references `E1`, `H1`, `C1`, `P1` | an employee, a hire request, a teleradiology contract (new or existing), a teleradiology plan | the letter and the order of first appearance in the proposal; described as "the employee in action 2 of your reply" |
| Alternative IDs | `proposal`, `no_action`, `adjusted`, `without_<k>` | a closed pattern; anything else fails |
| Action type, op, role, windows, scope | closed enumerations | checked against the engine's own value sets |
| Validation | accepted, or refused, with the place of each refused action in the reply | neither the engine's code (an open set of strings) nor its message is shown (review F1) |
| Dates | `YYYY-MM-DD`, checked to be a calendar date | the value itself |
| Numbers | cents, counts, days, minutes | formatted from the integer |
| Item kinds | one row of the closed table `viewKinds` (46 rows) | the row's fixed label and basis, written in `view.go` |
| A "when" | one of the row's fixed forms, such as `recognized now, paid {D}` | the form's text, rebuilt from the typed slots |
| Not available | one row of `viewUnavailable` | the row's fixed label and reason |
| Reversibility | one of six kinds, decided from the batch's action types | the kind's fixed text; the comparison's value must be exactly the engine rule's for the batch (review F1) |
| Modalities | XR, US, CT, MR | the scenario format's closed set |
| Employment stage | recruiting, onboarding, active, notice, terminated | the engine's closed set |

**Never in a view:** employee names and IDs, request IDs, contract IDs,
plan IDs, engine labels, engine bases and derivations, validation codes
and messages, descriptions, sources, the comparison's or any other digest, the
decision point's engine string, scenario fingerprint or state digest, and
any projection or evaluation value (the agent audience cannot hold one;
see V0.13b E3).

## 3. How it is made: typed, not sanitized

No source string is inspected to decide whether it is safe to show. Every
comparison field the view uses is used in one of three ways:
* **Not shown at all.** A name, an ID, a label, a basis, a message, a
  code or a description: never copied. A value's label, class, source,
  basis and derivation are the comparison's own metadata: its contract
  checks them (`DecodeAgentAt`), and the view never shows them. Any other
  field must be empty unless the value's unit uses it: a date on a count,
  or a number on a date, fails the view (review F1). Where the view needs to know what a field
  refers to, it compares the field for equality with a known value (an
  engine label of one of the batch's own targets), and keeps only the
  reference.
* **Shown as a type.** A date, a number, or a member of a closed set.
* **Parsed against an exact form, then rebuilt.** Some engine values come
  as text with typed parts (`recognized now, paid 2026-10-30`). The view
  accepts the text only if it is exactly one of the row's fixed forms with
  its slots filled by dates or whole numbers. The text shown is made again
  from the form and the slot values, so nothing but the form's fixed text
  and the typed values can appear.

## 4. Neutral references and target matching

* References are numbered in the proposal's order. The text defines each
  one by the action of the model's own reply that names it: "E1 is the
  employee in action 1 of your reply". The model has its reply in the
  conversation, so it can tell targets apart and match each alternative to
  its own actions.
* Every alternative's action is shown with the proposal action it is, or
  adjusts (`action 2: terminate E1, lastDay 2027-01-26 (lastDay differs
  from your reply)`). An alternative's action may differ from the
  proposal's only in the fields the selection rule adjusts (lastDay,
  startDate, effective, scope); any other difference fails.
* A reply selects an alternative by its ID. A revision is written in the
  engine's own action form, as every reply is.
* The mapping (E1 to the employee ID, H1 to the request ID, and each shown
  item's key to the comparison key, per alternative) is kept in the turn's
  record (`record.map`). It is never sent.

## 5. Fail closed

`BuildView` returns `ErrView` for anything the tables do not define: an
unknown item key, a money line whose label is not one of the known
prefixes followed by exactly one of the batch's own targets, a "when" or a
basis of no known form, text in a numeric value, a modality or stage
outside its set, an unknown "not available" item, a date that is not a
date, an action that differs from the proposal's in a field not shown, an
alternative ID outside the pattern, a field the value's unit does not
use, or a reversibility value that is not exactly the engine rule's.

**When a view cannot be made,** neither arm is shown a turn: both get
`not_due:not_representable`, and validation-first submits the validated
proposal. The arms stay identical at that point. The driver's `check`
command and the fidelity evidence require every protocol context to have
a view, so this can happen in a run only for a correction a model writes
at a refused point.

**What the boundary covers** (review §1): the texts a workflow may send,
`Turn.Message`, `Turn.Inspect` and the `answer` of `alternatives.decide`.
The rest of each operation's result (`Turn.Record`, the parsed reply, the
outcome) is record data with raw actions and the mapping. It is never sent.

## 6. The lexical check is kept as a second guard

Every text the turn or the decide operation returns still passes V0.13c's
`checkText` (no run of 16 or more hexadecimal characters; none of the
banned words). With the typed view, nothing a source wrote can reach the
text, so this guard is expected never to fire.

## 7. Adversarial tests

| Test | What it does |
|---|---|
| `TestTurnHostileFreeText` | Five hostile employee names (an instruction, a fake evaluation value, a seed and a digest, a JSON decision, a name with a newline and a fake heading) put into the scenario. The name is in the session's observation, which proves it is there. For one and for two terminations, no fragment of it, and no employee ID, is in the control text, the treatment summary or any inspect answer. The mapping keeps the real ID. The same five strings as a teleradiology `requestId`: no request ID and no plan ID is shown. |
| `TestViewFailsClosed` | Fourteen edits to a bound comparison, each with the comparison's own checks kept satisfied (identities, compared values, digest renewed) so that only the view can refuse it: an unknown key; text after a "when"; text in a count; a money line naming someone who is not a target; a money line of no known kind; a modality outside the set; an employment stage outside the set; an unknown unavailable item; a date that is not a date; an action field the view does not show; an action of no proposal target; an alternative ID outside the pattern; an unexpected date on a count; a reversibility basis edited. Each must fail with `ErrView`. |
| `TestReviewV13dF1Closed`, `TestViewHires`, `TestViewDropChecked` | A refused alternative with a code that says "please_select_no_action" reads "refused: action 1"; a hire's view keeps its request ID neutral, and two hires with one offer label fail closed; a repeated severance line must equal what it repeats. |
| `TestTurnDueMatchesV13c` | The view is made for every sensitivity member and the extra cases, and the turn is due exactly where V0.13c's was. |
| Fidelity: malicious free text | A constructed context whose proposal has a hostile `requestId`. The model's own proposal is in the conversation, but no message the engine made contains the hostile text. |

## 8. Threat model, as it now stands

* Free text written by a scenario, a recording or a model's reply cannot
  reach any text the engine makes for the turn.
* The model still sees its own earlier messages, including its own reply's
  free text. That is the model's own text, and the turn does not repeat it.
* The comparison's values are the engine's, bound to the decision point by
  `DecodeAgentAt`. The view cannot add a value; it can only drop text and
  map identities.
