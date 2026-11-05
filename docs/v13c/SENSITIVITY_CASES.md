# V0.13c sensitivity cases

**The evidence:**
* `docs/benchmark/v13c-dev/sensitivity-pairs.json`.
* Every member's exact control and treatment turns, with every inspect
  answer, are in `docs/benchmark/v13c-dev/turns/S<n>-<a|b>.txt`.
* `TestEvidenceSensitivityPairs` makes them from the engine and compares
  them byte for byte on every CI platform.
* No value in them is typed by hand, and no engine fact is invented.

## 1. Construction

**Each member is a real engine state:**
* mini, seed 52;
* reached from the scenario's start by advancing with no action to 07:00
  on the date given;
* a scripted proposal. No model made it.

**Member a and member b differ in their proposal**, or, in S1, in their
decision date.

**What is listed for each pair:**
* every item the treatment turn renders (summary and every inspect
  answer) that differs between the members, by alternative, section and
  key: the decision time, each alternative's position, actions,
  description and validation as shown, each value's label, value, time,
  basis and derivation, and the order of every list. An accepted action's
  validation message is not rendered, so it is not compared. A proposal
  employee's ID in a key is written `<employee>` so that different
  employees compare item by item. `TestVisibleLedgerComplete` checks that
  each rendered field reaches the ledger (V0.13c review F2);
* every rendered line of the treatment turn that differs, position by
  position (`treatmentLinesThatDiffer`), as a cross-check;
* each difference classed as `[value]`, `[label only]` (the employee's
  name), or `[only in a]`/`[only in b]` (an item or alternative shown
  in one member only);
* every line of the control turn that differs;
* whether the two observations (the whole decision context the agent's
  prompt is built from) are equal.

**Nothing is hidden or equalized.** Where the simulator does not permit
isolating one fact, the other differences are listed.

## 2. The pairs

| Pair | Fact | Member a | Member b | Same state | Differences (value / label only / one member only) |
|---|---|---|---|---|---|
| S1 | displaced bookings, zero versus nonzero, decided two weeks apart | terminate Sam (E04) with 14 days' notice, decided Oct 13: **0 displaced**, 2,481 slots closed | the same with 14 days' notice, decided Oct 27: **18 displaced** (18 booked now), 2,481 slots closed | no | 38 / 0 / 4 |
| S2 | displaced bookings, zero versus nonzero, same state | terminate Riley (E03), last day Feb 16, decided Oct 20: **0 displaced**, 45 slots | last day Feb 9: **1 displaced**, 278 slots | yes | 14 / 0 / 2 |
| S3 | severance | terminate Casey (E05), last day Oct 27: **$4,500.00** severance, guarantee $2,250.00 a week, 684 slots | terminate Morgan (E06): **$4,345.00**, $2,172.50 a week, 872 slots | yes | 20 / 17 / 4 |
| S4 | an alternative available versus unavailable | terminate Casey (E05), last day Jan 26: the later-last-day adjustment **listed** (Feb 9, in the booking horizon) | last day Feb 2: the adjustment **not listed** (Feb 16 is past the booking horizon) | yes | 10 / 0 / 31 |
| S5 | guaranteed commitment | teleradiology start, weeknights, Saturday and Sunday, from Oct 27: retainer to the earliest end **$3,083.34** | weeknights only: **$1,491.94** | yes | 6 / 2 / 0 |
| S6 | capacity loss | terminate Sam (E04), last day Oct 27: **2,481** slots no longer staffed | last day Jan 26: **362** | yes | 26 / 0 / 0 |

## 3. What else differs, pair by pair

**S1:**
* The decision dates differ, so the whole context differs: the
  observation, the booked-ahead count (0 vs 277), payroll accrued
  ($2,133.55 vs $38,675.39) and payables ($2,854.82 vs $50,300.58).
* The termination's own figures are equal: notice, severance, contract
  pay to the last day, weekly guarantee and slots closed. Only their
  dates move by two weeks.
* This is the closest the engine permits to "the same termination with
  and without displacement". It is the V0.13b pair, now with both
  arms' turns.

**S2:**
* The same state and conversation. The last days are a week apart, so
  slots closed (45 vs 278), notice (119 vs 112 days) and contract pay
  differ too.
* The displacement contrast is small, 0 vs 1 booking.

**S3:**
* Different employees: severance moves with the guarantee, contract
  pay, contracted minutes and capacity. Severance is set by each
  contract and cannot be varied alone.
* The modality of the closed slots differs too: MR in a, US in b.
* The severance difference is small, $155.

**S4:**
* The listing differs (the 25 one-member-only items are the adjusted
  alternative's).
* Also: a week of last day, slots closed (99 vs 54), notice and
  contract pay.

**S5:**
* The best isolated pair. Only the windows, the retainer guarantee,
  the run-rate and read-fee estimates and the action text differ;
  nothing about bookings or staff.

**S6:**
* The last day moves three months. Notice (14 vs 105 days), contract
  pay to the last day ($4,350.00 vs $30,025.00) and every date move
  with it.

## 4. The measurement detects what a rule uses

The six scripted reply rules are pure functions of the comparison, and
none is a model. Each pair's own fact is detected by the rule that reads
it, and `always-retain` never differs within a pair:

| Pair | Rule reading the fact | a | b |
|---|---|---|---|
| S1 | no-action-if-displacing | retained | selected_no_action |
| S2 | no-action-if-displacing | retained | selected_no_action |
| S3 | no-action-if-severance-over-4400 | selected_no_action | retained |
| S4 | later-last-day-if-listed | selected:adjusted | retained |
| S5 | no-action-if-guarantee-over-2000 | selected_no_action | retained |
| S6 | no-action-if-slots-over-1000 | selected_no_action | retained |

* The other rules' results are in the evidence file.
* A rule that reads a fact both members share gives the same result in
  both. For example, no-action-if-guarantee-over-2000 selects no action
  in both members of every termination pair, because the severance
  guarantee is over $2,000 in each.
* So a change in a model's decision within a pair cannot, by itself, be
  attributed to the named fact alone. The listed differences say what
  else it could follow. The difference-in-differences against control
  removes only what the control turn also shows: the proposal and its
  validation.

## 5. Use in the protocol

* Every member is a unit of the draft protocol, in both arms, for both
  models.
* The context is the agent's prompt built at the member's state by the
  agent's own `prepare`, with empty memory.
* The proposal is the scripted proposal, as the first reply.
* A pair's primary outcome is decision sensitivity,
  Δ_treatment − Δ_control ([EXPERIMENTAL_DESIGN.md](EXPERIMENTAL_DESIGN.md) §6).
