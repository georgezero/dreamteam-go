# V0.13b candidate selection: `dreamteam.alternative-selection/v1`

Status: implemented in 0.13.0-dev.2, `internal/alternatives/selection.go`.

**The rule's text:**
* `internal/alternatives/text/selection-v1.txt` is embedded in the
  build.
* It is pinned by SHA-256 `158a6254952ee3c19dba0b2ea045d2d02de510e7f7f7537cf435357facc5e358`.
* A test fails if the text changes without the pin.

## 1. When it applies

The rule applies to a proposed batch that holds a `terminate`, a `hire`,
or a `contract_teleradiology` op `start`. It also applies to a staffing
batch whose engine preview reports displaced bookings.

Otherwise no candidate is listed, and `applies` is false with the
reason. An operations batch has no preview, so it is out of scope.

## 2. Candidates, in this order, at most four

| # | ID | Content |
|---|---|---|
| 1 | `proposal` | the batch as proposed, unchanged; always listed |
| 2 | `no_action` | no batch; always listed |
| 3+ | `without_<k>` | for two or more actions: the proposal without its *k*-th action (1-based), for each `terminate`, `hire` or contract start in batch order, while fewer than four are listed |
| 3 | `adjusted` | for one action: one adjustment toward a smaller, later or less binding commitment, listed only when the engine accepts it |

**Adjustments:**

| Action | Adjustment |
|---|---|
| `terminate` | `lastDay` + 14 days |
| `hire` | `startDate` + 14 days |
| `contract_teleradiology` start, scope `all` | the same contract, scope `urgent` |
| `contract_teleradiology` start, scope `urgent` | `effective` + 14 days |
| `set_roster` op `set` | `effective` + 7 days |
| any other | none |

* An adjusted date must be within the booking horizon (today plus the
  horizon days). Otherwise the rule defines no adjustment.
* An adjustment the engine refuses is not listed. It is recorded under
  `omitted` with the engine's codes.

**Not done:**
* No candidate is repaired.
* No other action type is substituted.
* No order is set by value.
* A refused proposal or leave-one-out candidate stays listed, with the
  engine's codes.

**Departure from V0.13a's proposal:**
* V0.13a proposed moving a premium hire to the standard tier.
* The action carries no tier field; the engine infers the tier from the
  offer table.
* An exact tier change would need that table, so the rule uses the later
  start for every hire.
* The tier adjustment is deferred.

## 3. Determinism and replay

**The selection is a pure function of four inputs:**
* the proposal;
* the engine's validation answers;
* today's date;
* the booking horizon.

**Each candidate's identity** is the SHA-256 of the rule's ID, the
candidate ID and its canonical actions.

**`VerifySelection(session, selection)`:**
* recomputes the selection;
* requires it equal in every field;
* refuses another rule or rule digest.

**Checks:**
* Selecting does not change the session's digests.
* Tested:
  * the termination case gives `proposal`, `no_action`, `adjusted`
    (last day +14);
  * two or three terminations give leave-one-out candidates, limited to
    four;
  * an operations batch: the rule does not apply;
  * an invalid proposal stays listed, refused and unrepaired;
  * a refused adjustment is omitted and recorded;
  * an adjustment past the horizon is not defined;
  * a refused leave-one-out candidate stays listed;
  * a reordered selection does not replay.
