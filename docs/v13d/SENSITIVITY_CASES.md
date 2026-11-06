# V0.13d sensitivity cases: context pairs, not single-fact interventions

**Owner decision:** accept the six V0.13c pairs as context pairs. They do
not isolate individual facts, and nothing here claims sensitivity to one
fact when other visible facts differ.

**Evidence:** `docs/benchmark/v13d-dev/sensitivity-pairs.json` (every
difference), `sensitivity-summary.md`, `turns/S<n>-<a|b>.txt` (every text),
made by `TestEvidenceV13d` (`internal/alternatives/v13d_evidence_test.go`)
and required byte for byte.

## 1. The pairs, unchanged

The members are V0.13c's: real engine states of mini, seed 52, reached
from the start with no action, each with a scripted proposal. No model made
these proposals.

| Pair | Named contrast | Member a | Member b |
|---|---|---|---|
| S1 | displaced bookings, decided two weeks apart | terminate E04 Oct 13, last day Oct 27 (0 displaced) | terminate E04 Oct 27, last day Nov 10 (18 displaced) |
| S2 | displaced bookings, the same state | terminate E03, last day Feb 16 (0) | last day Feb 9 (1) |
| S3 | severance | terminate E05 ($4,500.00) | terminate E06 ($4,345.00) |
| S4 | an alternative listed or not | terminate E05, last day Jan 26 (adjusted listed) | last day Feb 2 (adjusted not listed) |
| S5 | guaranteed commitment | teleradiology, three windows ($3,083.34 to the earliest end) | weeknights only ($1,491.94) |
| S6 | capacity loss | terminate E04, last day Oct 27 (2,481 slots) | last day Jan 26 (362 slots) |

## 2. Every visible difference, through presentation v2

Counted by the evidence test over the typed view (each shown item by
alternative, section and key, and each list's order) and over the rendered
lines:

| Pair | Items with different values | Items shown for one member only | Control lines that differ | Treatment lines that differ |
|---|---|---|---|---|
| S1 | 38 | 6 | 1 | 83 |
| S2 | 15 | 3 | 0 | 49 |
| S3 | 14 | 4 | 0 | 14 |
| S4 | 8 | 39 | 0 | 93 |
| S5 | 6 | 0 | 0 | 8 |
| S6 | 22 | 0 | 0 | 24 |

**What the typed view changed.** In V0.13c, S3's members differed in 17
labels only because the employees' names differed. With neutral references
both employees are E1, so those differences are gone: S3 now differs in
values only (severance, contract pay, the guarantee, slots and the
modality list). The model still knows which employee its own reply names.

**What each pair still changes besides its named contrast:**
* **S1** changes the whole context: a state two weeks later (time, booked
  ahead, payroll, payables), every date, and the displacement lines.
* **S2** changes the last day, the notice, the slots, the contract pay and
  the estimate, besides one displaced booking.
* **S3** changes the employee, and with it every employee-dependent value.
* **S4** changes the last day and its consequences, besides whether the
  adjusted alternative is listed (39 items exist for one member only).
* **S5** changes the windows, and with them the retainer and the run rate.
  It is the closest to isolating its contrast.
* **S6** changes the last day, and with it the notice, the dates, the
  contract pay and the slots.

## 3. The estimand these pairs support

For a pair (a, b) and a model: the departure rate (anything but
`retained`) in b minus that in a, in each arm, and the treatment's
difference minus the control's. This is **sensitivity to the named pair of
contexts, with all of the differences above**. It is not sensitivity to
the named contrast alone, and it is not an improvement. The difference of
differences removes only what the control turn also shows (control shows
the time only).

## 4. Further isolated cases: none designed

The owner allowed further isolated cases only if they can be constructed
from real engine states without fabricating information, and without new
simulator mechanics. None was found that isolates one fact:
* A termination's last day sets its notice, its end date, its severance
  payday, its contract pay, its slots and its displacements together.
* A different employee changes every employee-dependent value.
* The same proposal at two times changes the whole context.
* A teleradiology scope or window change moves the retainer and the run
  rate together.

Any such pair would be one more context pair, with its differences listed.
Adding one is possible later, but it would not change what the pairs can
show.
