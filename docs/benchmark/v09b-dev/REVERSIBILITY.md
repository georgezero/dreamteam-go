# Reversibility matrix

The simulator's own rules for undoing each management action, as `dreamteam.financial-evidence/v1` applies them (`MATRIX` in analysis/evidence/financial_evidence.py; this page is generated from it). Each rule names the engine source it comes from. Nothing here is a judgement of the action: a reversible action is not harmless, and an irreversible one is not a mistake.

The per-action amounts (fees, severance, earliest exit, guaranteed retainer, withdraw-by dates) are the engine's own preview values, per record in financial-evidence.json.

| Action | Class | Effective | Notice | Inverse action | What remains owed | Cost of reversing | Source | Executed (v06c / v07c / v08c) |
|---|---|---|---|---|---|---|---|---|
| `hire` | reversible_with_retained_obligation | the start date (at least the offer's lead time ahead) | a termination's last day is at least 14 days after both today and the start date | terminate (also before the start date) | the recruiting fee (recognized at acceptance, not refunded), the contract guarantee to the last day, and two contracted weeks' severance with benefits | the preview's "Earliest lawful exit" line: the guarantee from the start to the earliest last day plus severance; the recruiting fee in addition | internal/sim/staffing.go, offers.go (NoticeDays = 14) | 8 / 7 / 1 |
| `terminate` | not_reversible | employment ends at the midnight after the last day | at least 14 days; contract pay continues to the last day | none: a termination cannot be moved or withdrawn (already_terminating); capacity returns only through a new hire, with its own fee and lead time | severance (two contracted weeks, recognized at acceptance) and contract pay to the last day | not applicable | internal/sim/staffing.go | 3 / 7 / 4 |
| `set_reading_allocation` | withdrawable | a dated version, at least 14 days ahead | 14 days; once in force an allocation stays in force at least 14 days | op remove of the pending version, until the preview's withdrawBy date (the same notice); after that, a new set with 14 days' notice | nothing while withdrawable; once in force, the salary accrued under it | none recorded for a withdrawal | internal/sim/allocation.go (NoticeDays = 14, MinCommitDays = 14) | 11 / 10 / 8 |
| `set_roster` | withdrawable | a Monday at least 14 days ahead | 14 days, also to remove a pending version | op remove of the pending version up to 14 days before it takes effect; a version in effect or past never changes | the contract guarantee is owed whatever is rostered: no roster change saves or costs pay | none recorded | internal/sim/roster.go | 4 / 0 / 0 |
| `contract_teleradiology:start` | reversible_with_retained_obligation | the activation, at least 14 days ahead | a cancellation ends at least 30 days ahead and at least 30 days after activation | contract_teleradiology op cancel | the retainer to the minimum-term end, payable even if cancelled before activation | the preview's guaranteed retainer (GuaranteedCents through GuaranteedThrough) | internal/sim/telecontract.go | 23 / 20 / 1 |
| `contract_teleradiology:cancel` | not_reversible | the exclusive end date | at least 30 days | none: a fixed end cannot be moved or withdrawn (already_cancelled); coverage returns only through a new contract, with 14 days' notice | the retainer to the end; claims already made complete at their fees | not applicable | internal/sim/telecontract.go | 4 / 4 / 0 |
| `extend_shift` | not_reversible | the extension's date | none | none: no action withdraws an approved extension; a later roster change or termination can void it | the extra time is paid as worked | not applicable | internal/sim/actions.go | 335 / 232 / 18 |
| `set_walkin_policy` | reversible_no_recorded_cost | at once | none | set_walkin_policy again, at once | no money line | none recorded | internal/sim/actions.go | 1 / 1 / 0 |
| `set_worklist_policy` | reversible_no_recorded_cost | at once | none | set_worklist_policy again, at once | no money line | none recorded | internal/sim/actions.go | 7 / 10 / 1 |

Notes:

* **Withdrawal versus reversal.**
  * An allocation or roster version is withdrawable only while it is pending, and only with the same notice.
  * Once in effect, it changes only by a new version with notice. An allocation then stays in force at least 14 days (`MinCommitDays`).
  * An executed op `remove` is itself classed `withdrawable`: setting the version again restores it, under the same notice.
* **A hire.**
  * It can be ended before its start date by `terminate`, but the recruiting fee is not refunded and the contract runs from the start (staffing.go).
  * The cheapest exit is the preview's "Earliest lawful exit" amount.
* **A teleradiology start** carries the retainer to the minimum-term end. It is payable even if the contract is cancelled before activation (telecontract.go).
* **A termination and a cancellation** cannot be moved or withdrawn (`already_terminating`, `already_cancelled`). What they removed returns only through a new hire or contract, with its own fee, lead time and notice. Those costs are not computed here, because no such hire or contract was decided.
* **An approved extension** has no withdrawal action. A later roster change or termination can void it, and the engine then releases its reservation.
* **`unknown`** is the class for any action type the matrix does not cover. None occurred in the corpus.
