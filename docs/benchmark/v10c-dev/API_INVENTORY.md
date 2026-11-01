# What validate and preview tell an agent (V0.10c information inventory)

The engine's two read-only answers about a proposed batch, from the code
and from the 56 staffing proposals of the V0.8c and V0.9c recordings, each
asked at its recorded state. The data is in `points/inventory.json`
(`feedback_experiment.py inventory`), and CI makes it again.

I did not assume that the preview adds useful information. What it adds,
and when, is listed below.

## 1. The two requests

| | `validate` | `preview` |
|---|---|---|
| Handler | `internal/agent/agent.go` `validate` → `check` → `Session.ValidateBatch` | `internal/agent/agent.go` `preview` → `Session.PreviewStaffing` → `World.PreviewStaffing` (`internal/sim/preview.go`) |
| Accepts | any batch of 1–64 actions | staffing actions only (`hire`, `terminate`, `set_roster`, `set_reading_allocation`, `contract_teleradiology`); others → `not_previewable` |
| Changes state | no | no |
| Answer | `{"valid", "results"}` | `{"preview": {"asOf", "ok", "results", …}}` |

`preview.results` is built by the same call as validate's results
(`Results: w.ValidateBatch(actions)`, preview.go:98). In all 56 recorded
proposals the two arrays are identical (`previewResultsEqualValidate`:
56 of 56). `preview.ok` is validate's `valid`.

**When the preview adds more:**
* **`funding`,** the engine's funding test (`commitmentCents`,
  `headroomCents`, `funded`, `basis`). It is present when the scenario
  models solvency and every item is otherwise valid. So it appears for a
  valid batch, and for a batch refused only for `insufficient_funds`.
* **The detail sections,** only when every action is valid:
  * `hires`, `terminations`, `rosters`, `allocations`, `teleradiology`;
  * `slotsOpened`, `slotsClosed`, `displacements`;
  * `guaranteed` and `forecast` money lines.

  A refused batch's preview stops at `results` (and `funding`).

## 2. Per staffing action type, in the 56 proposals

A proposal can hold several types, so a row counts every proposal
containing that type.

| Type | Validation | Proposals | validate fields | Fields only the preview has |
|---|---|---|---|---|
| contract_teleradiology | accepted | 9 | `valid`; results: `id, batch, time, action, ok, code, message, effectiveAt, contractId` | `asOf, ok, funding, guaranteed, forecast, teleradiology`, and with rosters in the batch `rosters, slotsOpened, slotsClosed, displacements` |
| contract_teleradiology | refused | 20 | `valid`; results: `id, batch, time, action, ok, code, message` | `asOf, ok, funding` (the funding test only when the refusal is `insufficient_funds`) |
| hire | refused | 9 | as above | `asOf, ok, funding` |
| set_reading_allocation | accepted | 9 | `valid`; results with `effectiveAt` | `asOf, ok, funding, guaranteed, forecast, allocations`, with terminations in the batch `terminations, displacements` |
| set_reading_allocation | refused | 11 | as above | `asOf, ok, funding` |
| set_roster | accepted | 2 | `valid`; results with `effectiveAt` | `asOf, ok, funding, guaranteed, forecast, rosters, slotsOpened, slotsClosed, displacements` (and `teleradiology` with a contract in the batch) |
| set_roster | refused | 4 | as above | `asOf, ok` |
| terminate | accepted | 4 | `valid`; results with `effectiveAt` | `asOf, ok, funding, guaranteed, forecast, terminations, displacements` (and `allocations`) |
| terminate | refused | 3 | as above | `asOf, ok` |

No hire was accepted in these recordings, so a valid hire's preview
fields (`hires`: offer, tier, rate, lead time, start and productive dates,
training, slots opened) are known from the code only.

**Refusal codes observed:**

| Type | Codes |
|---|---|
| contract_teleradiology | invalid_schema 6, contract_overlap 6, insufficient_funds 3, past_date 3, batch_rejected 2 |
| set_reading_allocation | batch_rejected 7, insufficient_funds 3, commitment_too_short 2, past_date 1 |
| hire | lead_too_short 8, batch_rejected 3 |
| set_roster | batch_rejected 3, not_monday 2, wrong_role 1, notice_too_short 1 |
| terminate | batch_rejected 3, notice_too_short 1 |

**Over the 56 proposals:**
* **Valid (19):** every preview has `funding`; 17 also have detail
  sections; 5 list displaced bookings. Median lengths: validate 655
  characters, preview 2,128.
* **Refused (37):** 31 previews have nothing beyond `results`; 6 add
  `funding` (`insufficient_funds`). Median lengths: validate 575,
  preview 786.

## 3. The items the owner asked about

| Item | `validate` | `preview` |
|---|---|---|
| Refusal codes | `results[].code` | the same array |
| Refusal messages | `results[].message` | the same array |
| Accepted actions' terms | in `results[].message` as text. For example: a teleradiology contract's plan, windows, dates, monthly retainer and read fees; an allocation's site share, reading hours and salary change | the same messages, and structured items (`teleradiology`, `allocations`, `rosters`, `terminations`, `hires`) |
| Financial commitment | not as a number. Only an `insufficient_funds` message states it ("this batch commits $1,465 …") | `funding.commitmentCents` for a valid or funding-refused batch; `guaranteed` money lines (amounts, when, basis) for a valid batch |
| Funding headroom | only in an `insufficient_funds` message ("more than the funding headroom −$2,430 = cash … + undrawn credit … − current liabilities … − reserved …") | `funding.headroomCents` and `basis` (cash, undrawn credit, current liabilities, reserved), valid or funding-refused |
| Credit requirement | not stated. Derivable from an `insufficient_funds` message's breakdown | not stated. Derivable from `funding`: the commitment against the headroom less undrawn credit (as financial evidence v1 does) |
| Reversibility | not stated. `effectiveAt`; teleradiology messages give the earliest end | not stated as a class. Facts: an allocation's `withdrawBy`; a contract's `minimumTermEnd` and `guaranteedCents` to `guaranteedThrough`; a termination's `lastDay` and `severanceCents`; the "Earliest lawful exit" money line for a hire |
| Operational exposure | partly, in messages (an allocation's reading hours) | `slotsOpened`, `slotsClosed`, `displacements` (appointment, exam, priority, date, reason, provisional replacement); rosters' and allocations' before/after minutes; the reporting backlog; a contract's trailing unread demand |
| Displaced appointments | no | `displacements`, valid batches only |
| Shared by both | `valid`/`ok`, every result (codes, messages, effective dates, contract IDs) | |
| Preview only | | `asOf`, `funding`, every detail section, money lines |

Financial evidence v1's classes are not in either answer:
`funded_with_credit`, `reversible_with_retained_obligation`,
`direct_change`. They are its own reading of the preview's facts and the
simulator's rules.

## 4. What reaches the model

* **llm-agent 4** shows the whole preview answer, compact JSON, verbatim,
  when it fits the prompt's bound (`maxPromptChars`). Every field above
  reaches the model. It does not show the validate answer: it validates
  after the review reply.
* **llm-agent 3** shows nothing for a valid proposal. After a refusal it
  sends `Your reply could not be used: refused: action i: code: message;
  …`, which holds the refused results' code and message only.
* **V0.10c's arms:**
  * B shows the whole validate answer for the staffing actions, accepted
    results included;
  * C shows the whole preview answer for the same actions;
  * A shows neither.

So:
* **At a refused proposal,** B and C carry the same results. C adds the
  wrapper, `asOf`, and for an `insufficient_funds` refusal the
  structured funding test. The refusal message already states its
  numbers.
* **At a valid proposal,** C adds the funding test, money lines, item
  details, slots and displacements. It is 1.9 to 12.8 times as long at the
  selected points.
