# Slice 5B functional validation (0.2.0-dev.8)

A check that `contract_teleradiology` works end to end on the dynamic
benchmark's starting practice (`../feasibility/step0c/experiments/W130-SHR-MRft-s20.json`,
the shared ×1.30 start). **It is not a strategy search or an evaluation.**
The contract is one fixed lifecycle, not tuned, and no result chose
anything. It uses only development seeds (901–903). The held-out
reference seeds 1001–1020 are not touched. 5C's evaluation, frozen before
any strategy, will use a new locked holdout.

    go test -tags calibration ./internal/sim -run 'TeleFunctional$' -v

The plan:
* At the takeover (Tue Oct 13), a `standard-v1` contract on every window
  (weeknights, saturday, sunday) for all priorities, effective Oct 27
  (the earliest, 14 days ahead).
* On day 44 (Thu Nov 26, 09:00) it is cancelled at its earliest end,
  Sat Dec 26, 30 days ahead.
* The run goes to day 90.

Each seed is paired with the same seed without the contract. Every
invariant (ledger, cash, payables, remote ownership and the vendor
accounts) is checked at every midnight. Results:
`results/functional-dev.json`.

| Seed | Claims (night) | XR / US / CT / MR | Retainer | Read fees | Paid to vendor | Unpaid at day 90 | Remote TAT (mean) |
|---|---|---|---:|---:|---:|---:|---:|
| 901 | 641 (22) | 185 / 95 / 243 / 118 | $6,100.01 | $23,246.40 | $29,346.41 | $0 | 59 min |
| 902 | 641 (10) | 196 / 77 / 257 / 111 | $6,100.01 | $22,953.60 | $29,053.61 | $0 | 52 min |
| 903 | 610 (21) | 198 / 77 / 221 / 114 | $6,100.01 | $21,619.80 | $27,719.81 | $0 | 52 min |

What it shows, on every seed:
* **The contract activates.** The first claim is at Tue Oct 27 00:00, the
  part of Monday night's window that falls inside the contract period.
  Retainers accrue from Oct 27.
* **Eligible backlog is claimed.** Every claim is reported, and its fee
  equals the catalog price, +20% when claimed at night.
* **Retainers are exact.** Oct 27 to Dec 25 on all three windows comes to
  $6,100.01, equal to the independently computed proration.
* **Vendor bills settle.** The Nov 1, Dec 1 and Jan 1 payments pay every
  retainer and fee accrued before them. Nothing is unpaid at day 90.
* **The cancellation stops new claims.** No claim is made at or after
  Dec 26 00:00. The last ones are on Dec 25 near 20:40. Every read
  finished before midnight, so no read crossed the end on these seeds;
  `TestTeleReadCrossesExpiry` covers that case.
* **One owner per study.** The ownership invariants hold at every midnight.
* **Books reconcile.** The ledger, payables and cash reconcile, and the
  evaluation's cash bridge reconciles to the cent.
* **Bookings are unaffected.** Appointments are identical to the paired
  no-contract run.

One observation is for 5C, not a verdict. The final reports with the
contract (local plus remote) equal those without it: 3,939 + 641 = 4,580
on seed 901, and likewise on 902 and 903. Over 90 days the contract
substituted for reads the employed radiologists would have made later,
at $27.7–29.4k in retainers and fees. Their salary is unchanged, and
operating profit falls by about that much. Whether the earlier reports
are worth it is a turnaround and service question. 5C's frozen criteria
measure that, with whatever allocation changes go with a contract.
