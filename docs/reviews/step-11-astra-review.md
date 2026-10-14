# Step 11 review — gpt-6-astra, high reasoning

* Reviewed: commit `e5863b4` (V0.1 step 11), read-only snapshot copied to a
  temporary directory (deleted afterwards).
* Command: `codex exec -m gpt-6-astra -c model_reasoning_effort="high" -s read-only`
* Prompt: the owner's review brief (warm-up boundary, financial
  reconciliation, owner capital, historical cash presentation, convergence,
  performance optimizations, accounting boundaries, test quality), plus a
  note on the environment (no Go toolchain; review by reading code) and the
  origin of the quoted balance-sheet figures.
* The findings below are the model's output, unedited. File links point at the
  temporary snapshot path used for the review; the line numbers match
  `e5863b4`.

## Findings (verbatim)

**CONDITIONAL GO for Step 12.** Fix takeover ordering and historical-cash presentation before building further metrics on them. The cent-based accounting reconciles by construction; the reported $1 difference comes from independently rounded display totals.

This is a code-tracing review of snapshot `e5863b4`. No tests were executed, toolchains downloaded, or files changed. Recorded test and performance results below are repository evidence, not independently reproduced results.

**A. Confirmed defects from code tracing**

1. **High — Capitalization and the opening snapshot occur before the takeover timestamp has finished processing.**

   References: [opening.go:37](/tmp/dt-review.5y1iMy/internal/sim/opening.go:37), [world.go:81](/tmp/dt-review.5y1iMy/internal/sim/world.go:81), [world.go:239](/tmp/dt-review.5y1iMy/internal/sim/world.go:239), [finance.go:277](/tmp/dt-review.5y1iMy/internal/sim/finance.go:277), [engine.go:152](/tmp/dt-review.5y1iMy/internal/sim/engine.go:152).

   **Trigger:** A positive warm-up ending at midnight on a payday or first of the month, with accrued liabilities. Alternatively, takeover at the claim-payment clock with a payment due.

   `New` queues the opening event immediately, with sequence number 2 and `ClassDayBoundary`. Future day-start events are queued incrementally and therefore have later sequence numbers. At a midnight takeover after warm-up, **opening runs before day start**, contrary to its comment.

   Cash is first set to the target. Payroll and bills then reduce it. If their balances are `P` and `B`, the returned opening cash is `target − P − B`. At a payment-time takeover, class-6 collections instead increase cash after capitalization. The opening snapshot also precedes same-instant report completions, recall releases, appointments and dispatch decisions.

   This does not break double entry or chunking determinism: both uninterrupted and split runs execute the same incorrect ordering. It breaks the promised opening balance and snapshot semantics.

   **Smallest fix:** Finalize capitalization and capture the opening snapshot after the ordinary takeover transaction, including dispatch, has completed, before returning control. Merely moving it after financial events still leaves dispatch-generated wage accruals and state changes outstanding.

   **Regression:** Use a tiny warmed practice with payroll and payables due at midnight; separately place a claim payment and report completion exactly at takeover. Assert target cash, snapshot/current-state agreement, one capital entry, and unchanged operating postings. Repeat uninterrupted, split and budgeted runs. Current tests use ordinary 06:00 and 13:00 boundaries and do not explicitly cover these collisions.

2. **Medium — Pro forma historical cash is presented as actual historical cash.**

   References: [history.go:112](/tmp/dt-review.5y1iMy/internal/sim/history.go:112), [finance_view.go:59](/tmp/dt-review.5y1iMy/internal/sim/finance_view.go:59), [briefing.go:135](/tmp/dt-review.5y1iMy/internal/tui/briefing.go:135), [finance.go:195](/tmp/dt-review.5y1iMy/internal/tui/finance.go:195).

   **Trigger:** Any nonzero takeover adjustment followed by viewing history or the cash sparkline.

   Both projections calculate:

   `displayed historical cash = actual historical cash + takeover adjustment`

   That arithmetic correctly implements the chosen constant-shift convention. The ledger preserves actual cash. However, the screens say **“CASH END”** and **“Cash, last … days”**, without identifying the values as restated or pro forma. The historical marker means only “before day 1.” The documentation’s assertion that the restatement is labelled is false.

   The adjustment is disclosed in the briefing and opening log, but the chart removes its jump and supplies no annotation. Consequently, ordinary cash presentation can imply that hypothetical capital existed throughout the history.

   **Smallest fix:** Label both displays explicitly as pro forma and explain the shift. Preserve an explicitly named actual-cash series for consumers; alternatively, show actual cash by default and retain the takeover jump. No ledger changes are needed.

   **Regression:** Use known historical cash and a nonzero injection. Check actual cash against the journal, adjusted cash against actual plus adjustment, and both rendered labels. Existing tests assert shifted values—including yesterday equalling opening cash in the default case—but do not test disclosure or an actual series.

3. **Medium — Accepted `24:00` financial clocks bypass business-day scheduling.**

   References: [validate_finance.go:83](/tmp/dt-review.5y1iMy/internal/scenario/validate_finance.go:83), [simtime.go:252](/tmp/dt-review.5y1iMy/internal/simtime/simtime.go:252), [finance.go:332](/tmp/dt-review.5y1iMy/internal/sim/finance.go:332), [finance.go:445](/tmp/dt-review.5y1iMy/internal/sim/finance.go:445).

   **Trigger:** Configure `paymentTime: "24:00"` and a remittance whose adjusted due date is Friday.

   Validation accepts 24:00. Scheduling first checks Friday as a business day, then constructs Friday plus 24 hours: **Saturday 00:00**. Submission clocks have the same problem; their delayed handler also uses the new day’s `st.day`, potentially including claims due that new day.

   This violates the documented Monday–Friday rule. It is an existing validation/scheduling defect exposed by reviewing unusual financial boundaries; the bundled 10:00/17:00 clocks avoid it.

   **Smallest fix:** Require financial point-in-time clocks to be strictly below 24 hours. Retain 24:00 support where appropriate for interval endpoints.

   **Regression:** Reject 24:00 submission/payment clocks with field-specific errors; accept 00:00 and 23:59:59. Current lifecycle tests exercise default clocks, not this accepted edge.

4. **Low — Ledger hashing omits the new fixed-cost item identity.**

   References: [ledger.go:292](/tmp/dt-review.5y1iMy/internal/sim/ledger.go:292), [finance_view.go:262](/tmp/dt-review.5y1iMy/internal/sim/finance_view.go:262), [session.go:130](/tmp/dt-review.5y1iMy/internal/session/session.go:130).

   **Trigger:** Two fixed-cost items use the same account and modality. Compare configurations where only item A has a monthly amount versus only item B having that same amount.

   Zero accrual lines are skipped. The resulting postings can have identical times, keys, account, modality and amount, while differing in `line.item`. The ledger hashes match because `item` is omitted, although journal item labels and `FixedByItem` differ.

   This does not change balances; it weakens the digest’s ability to detect reporting changes.

   **Smallest fix:** Include the item index in canonical line hashing and intentionally update affected golden digests.

   **Regression:** Hash two otherwise identical balanced entries differing only in item tag; hashes must differ. The existing ledger test checks that adding a posting changes the hash, not that every meaningful line field participates.

5. **Low — The balance-sheet display writes an exact equation using independently rounded values.**

   References: [finance.go:23](/tmp/dt-review.5y1iMy/internal/tui/finance.go:23), [finance.go:167](/tmp/dt-review.5y1iMy/internal/tui/finance.go:167), [PROGRESS.md:59](/tmp/dt-review.5y1iMy/docs/PROGRESS.md:59).

   **Trigger:** Balanced cent totals whose fractional dollars produce different rounding for the total and its components—the supplied figures are such a case.

   `usd` rounds each amount independently using `(abs(cents)+50)/100`, restores the sign, and inserts separators. The screen then prints an unqualified `=`. Thus:

   `$428,154 = $88,071 + $340,082`

   is a false equation in displayed whole dollars even when the underlying cents balance exactly.

   **Smallest fix:** Show cents on the reconciliation line, or explicitly mark it as rounded with totals potentially differing. Do not adjust ledger balances or invent a balancing dollar.

   **Regression:** Supply balanced cent totals whose rounded components differ by $1 from the rounded total. Assert exact underlying equality and truthful presentation. Current opening tests check cents, not the rendered equation.

6. **Low — Optimized date formatting silently wraps years above 9999.**

   References: [simtime.go:155](/tmp/dt-review.5y1iMy/internal/simtime/simtime.go:155), [format_test.go:10](/tmp/dt-review.5y1iMy/internal/simtime/format_test.go:10).

   **Trigger:** Start on the accepted date `9999-12-31` and advance into the next year.

   The formatter takes the year modulo 10000, so year 10000 becomes `0000`. The previous `%04d` behavior expands to five digits. This changes observable dates and log text. The equivalence test only exercises dates around 1999–2002.

   **Smallest fix:** Keep the fast path for years 0–9999 and use a general formatter outside that range.

   **Regression:** Compare date and ISO formatting against the previous formatting for years −1, 0, 9999 and 10000. Existing formatter coverage does not include these boundaries.

**Financial reconciliation assessment**

[observeBalance](/tmp/dt-review.5y1iMy/internal/sim/finance_view.go:313) builds the statement entirely in integer cents:

- Assets = cash + A/R.
- Liabilities = A/P + accrued payroll.
- Equity = owner capital + historical net income + income since day 1.

Revenue, collection variance and expenses partition the remaining ledger accounts. Because each posting sums to zero, these definitions imply **Assets − Liabilities − Equity = 0**. [checkFinance](/tmp/dt-review.5y1iMy/internal/sim/finance.go:546) additionally reconciles cash movements, open-claim A/R, recognized revenue, payroll and payables against independent operational totals.

The reviewed postings are consistent:

| Event | Accounting effect |
|---|---|
| Report final | Increase A/R and expected-net revenue |
| Collection | Increase cash, reduce A/R; accrue billing fee |
| Settlement/write-off | Clear residual A/R through collection variance |
| Expense accrual | Increase expense and the appropriate liability |
| Payroll/bill payment | Reduce cash and liability without expensing twice |
| Capitalization | Change cash and owner equity only |

Given the user’s clarification that the three quoted totals came from **one observation**, their $1 discrepancy is **display rounding, accompanied by a misleading reporting equation**, not evidence of ledger imbalance. The raw cent triplet is not supplied or recoverable from the golden hashes, so this review does not independently reproduce its exact seed-1 values.

**Owner capital and warm-up continuity**

Cash starts at zero in [initFinance](/tmp/dt-review.5y1iMy/internal/sim/finance.go:262). Capitalization posts `target − cashBefore` against equity; it does not touch revenue, expenses, historical losses or claims. For the ordinary established 06:00 opening, this produces exactly 15,000,000 cents.

There is one scheduled opening event. Normal continuation consumes it once, and rebuilding starts a fresh ledger. Duplicate posting keys are rejected. A zero adjustment intentionally produces no empty journal entry. **Persisted resume is not implemented here**, so snapshot/resume safety cannot yet be certified.

The bundled `mini` scenario inherits the cash target with zero warm-up; test micro scenarios can use zero capital. New-practice and other planned scenario families do not yet provide additional implemented capitalization rules to verify.

Operational continuity is structurally strong: `session.New` advances the same world that gameplay subsequently uses. There is no copying or rebuilding of appointments, recalls, visits, studies, claims, liabilities, IDs or queued events at takeover. Weekly paid hours survive; they reset on Monday, not on capitalization. Employee generations reset during ordinary day materialization, not during opening.

The Step 8 protections remain relevant and present: explicit deadline wakes, registering-patient hold-over, supervisor commitments, deferred-break handling, and rejection of shifts whose caps reach midnight. I found no regression in those mechanisms caused by the staffing-buffer reuse.

Ordinary accounting boundaries also trace consistently:

- Payroll and first-of-month payables settle prior accruals before new daily accruals.
- Monthly cumulative rounding sums exactly to the monthly cost.
- Claim charge lag counts weekdays; remittance/rework dates roll weekends forward, subject to finding 3.
- Historical report TAT uses absolute acquisition-to-final timestamps and a finalization cursor, so an overnight-unread study can contribute its full elapsed TAT when finalized.

**B. Plausible risks requiring reproduction**

**Medium assurance risk — Existing equality tests do not establish equality of all pending state.**

References: [warmup_test.go:197](/tmp/dt-review.5y1iMy/internal/sim/warmup_test.go:197), [warmup_test.go:113](/tmp/dt-review.5y1iMy/internal/sim/warmup_test.go:113), [session.go:130](/tmp/dt-review.5y1iMy/internal/session/session.go:130).

The uninterrupted comparison checks log and ledger hashes after three gameplay days. Those hashes do not cover the full queue, entity generations, recall state or future appointment state. The longer survival test provides useful evidence, but aggregates pending studies and claims rather than requiring every lifecycle stage and a held-over employee to be present.

A defect affecting a recall or settlement months later could therefore escape the short equality comparison. This is a coverage limitation, **not a demonstrated lost-event defect**.

The smallest improvement is a deterministic mixed-state fixture with explicitly asserted pending categories, full state/queue comparison for split versus uninterrupted runs, and continuation through their resolutions. Noise variants should compare semantic outcomes while accounting for intentionally different event sequence numbers.

**C. Design and modeling limitations**

The [convergence test](/tmp/dt-review.5y1iMy/internal/sim/warmup_test.go:303) measures only forward booking fill and A/R balances by age. Its recorded nine-seed means show:

| Comparison | Recorded criterion/result |
|---|---|
| 180 versus 365 days | Bulk measures within 5%; thin tails within 25% |
| Largest reported bulk difference | 7–30-day booking fill: −4.6% |
| 90 versus 365 days | Fails, including missing A/R over 90 days |

This is useful regression evidence, but not statistical equivalence:

- No variances or confidence intervals are calculated.
- Nine seeds are not justified by a precision or power calculation.
- Changing warm-up length changes epoch-relative random keys, so equal seed numbers do not pair the same calendar-date demand.
- Aggregate means can conceal modality-specific or individual-seed bias.
- The negative control need fail only one metric.
- Payables, payroll liabilities, weekly hours, recalls, backlog and backlog age are not tested for convergence.
- A consistently empty or otherwise biased worklist can pass completely.
- The comparator should explicitly reject non-finite ratios; `0/0` otherwise produces a NaN that passes its comparisons.

The recorded 540-day comparison is additional documentation evidence, not part of the committed convergence test. Treat nine seeds as a smoke test until uncertainty is measured; determine the required sample from observed variance.

Two boundary conventions also need explicit ownership before benchmark scoring:

- [Design §8.4](/tmp/dt-review.5y1iMy/docs/SIMULATION_DESIGN.md:1437) says history ends at the exact start time. The implemented “since day 1” periods and counters instead start at midnight, as later documentation and tests explicitly specify. A 13:00 takeover therefore includes morning activity; even 06:00 includes daily accruals. Calendar-day reporting is reasonable, but those fields cannot silently serve as post-takeover scores.
- Equalizing opening cash compensates weaker histories with larger capital injections. This preserves losses correctly, but cross-scenario comparisons must report capitalization and inherited net assets separately. Score post-takeover performance against a fixed opening baseline; exclude capital transfers from operating performance.

Weekend-only business calendars, no holidays/DST, salaries accrued over a fixed 365-day denominator, arrears billing, and no financing interest or insolvency are modeling conventions rather than defects. The empty default opening worklist and historical losses are documented calibration limitations.

**Additional high-value tests**

The finding-specific regressions above should be supplemented with:

| Minimal setup | Expected outcome and invariant | Current coverage |
|---|---|---|
| Takeover during required overtime; employee already near 40 weekly paid hours | Same departure, overtime premium and single wage accrual as uninterrupted play | Weekly overtime and ordinary survival tested separately |
| One recall entering the horizon at takeover; one pending no-show rebook; one cancelled appointment with stale arrival | Each resolves once; no resurrected appointment, duplicate visit or changed ID allocation | General lifecycle coverage, no explicit combined takeover fixture |
| One claim in each pending/submitted/denied/resubmitted state | Same submission, settlement, variance and residual A/R across takeover | Aggregate open-claim survival and ordinary lifecycle tests |
| Study acquired Monday evening, finalized Tuesday after takeover | Full overnight TAT appears once in Tuesday history; Monday backlog remains recorded | Same-day TAT and monthly reading tests, no exact historical-summary assertion |
| More than 2,000 log entries, with digest/observation calls interspersed | Complete canonical hash unchanged by reads or ring eviction; retained entries ordered correctly | Long-run hashes exercise rollover indirectly; no direct retention oracle |
| Repeated memoized draws, interleaved lane consumption and day reset | Values equal fresh hashes; independent draw cursors; no observable latent values | Golden and leakage tests, no direct cache-equivalence test |

**Performance recommendation and release conditions**

Keep streamed log hashing, daily draw memoization, cached claim draws and staffing-list reuse. The log hashes each entry before retention eviction; reading a digest does not reset the stream. Memoization stores immutable lanes and returns fresh draw objects. Staffing changes clone the scratch buffer, avoiding room-list aliasing. I found no reason to revert these optimizations.

The log digest change is intentional and documented; old digest values are incompatible. Future replay/snapshot support must preserve or version the hash format and serialized hash state, and carry the opening marker, pending events and generations. Full event-log retention is not provided by the 2,000-entry ring.

Recorded builds improved to 1.35–1.97 seconds with lower memory and allocations. That supports the optimization direction, but does not establish an unconditional sub-1.5-second gate. Re-measure under controlled thermal/load conditions before adding another optimization or cache.

Proceed with Step 12 after resolving findings 1–2 and adding their regressions. Fix the accepted financial-clock edge as part of boundary hardening. The remaining low-severity issues are small, targeted corrections. Owner decisions are limited to actual versus explicitly pro forma cash presentation and calendar-day versus exact-takeover scoring windows; neither requires changing the underlying capitalization convention.
## Assessment

Each finding was checked against the code at `e5863b4` and reproduced by a
regression test before any fix.

| # | Severity | Verdict | Notes |
|---|---|---|---|
| 1 | High | **Confirmed** | The takeover event (queued second, class 0) ran before a later-queued day start at a midnight takeover, and before class-6 remittances at a 10:00 takeover. Opening cash then came out as target − payroll or bills, or target + collections. The snapshot also preceded same-instant work and dispatch. |
| 2 | Medium | **Confirmed** | The cash chart and daily records showed shifted (pro forma) history as plain cash. The documentation claimed the shift was labelled; it wasn't. |
| 3 | Medium | **Confirmed (edge)** | `submitTime`/`paymentTime` of 24:00 passed validation and scheduled onto the next (possibly weekend) day. |
| 4 | Low | **Confirmed** | `ledger.hash` omitted the fixed-cost item tag added in step 11. |
| 5 | Low | **Confirmed** | The balance line printed an exact "=" between independently rounded whole-dollar figures. The reported $1 ($428,154 vs $88,071 + $340,082) is this display rounding. The exact cents were 42,815,355 = 8,807,143 + 34,008,212, and the ledger was balanced. |
| 6 | Low | **Confirmed (edge)** | The fast date formatter wrapped years outside 0–9999. |
| B | — | Coverage gap, accepted | The split-vs-whole comparison checked only log and ledger hashes. It now also compares the pending event queue and all claims, appointments, studies, visits and orders. |
| C | — | Test limitation, accepted in part | The convergence comparator let a 0/0 ratio pass as NaN; fixed. The broader statistical points (no variance estimate, unpaired seeds, unmeasured liabilities and backlog) are recorded as limitations for calibration (step 18). |

No finding was rejected. The review found no ledger imbalance, no changed
randomness or log content from the optimizations, no leakage and no lost
events at the boundary.

## Resolution

All six defects were fixed in the step 11 remediation pass (version
`0.1.0-dev.11.1`). Each has a regression test that fails on `e5863b4` and
passes after the fix (tests in `internal/sim/regress11_test.go`,
`ledger_internal_test.go`, `internal/simtime/format_test.go`,
`internal/scenario/scenario_test.go` and `internal/tui/tui_test.go`). The
before and after outputs are in `docs/PROGRESS.md` ("Step 11
remediation"). The owner decisions raised (scoring windows; actual vs pro
forma cash) are recorded there and in SIMULATION_DESIGN §8.8.
