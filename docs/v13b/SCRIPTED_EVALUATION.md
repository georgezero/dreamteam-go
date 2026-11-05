# V0.13b scripted evaluation evidence

Status: 0.13.0-dev.2. Every reply here is scripted, and no model was
contacted.

**Where the evidence comes from:**
* The files in `docs/benchmark/v13b-dev/` are produced by
  `internal/alternatives/evidence_test.go`.
* Every run compares them byte for byte, on every CI platform: Linux
  amd64 and arm64, and macOS.
* `DREAMTEAM_V13B_WRITE=1` rewrites them.

| File | Content |
|---|---|
| `termination-selection.json` | the candidate rule's answer for "terminate Sam (E04), last day 2026-10-27" at mini, seed 52, 2026-10-13 07:00 |
| `termination-agent.json` | the agent audience of that decision point |
| `termination-presentation.txt` | the summary and each `inspect:<id>` detail, rendered by the pinned template |
| `termination-evaluation.json` | the evaluation audience: three verified branches to 2026-12-11, projections and differences |
| `scripted-evaluation.json` | the classified scripted cases and the sensitivity pairs |

## 1. Decision cases

All ten cases are at the termination decision point, with the alternatives
`proposal`, `no_action` and `adjusted`:

| Case | Reply | Result | Flags |
|---|---|---|---|
| alternative inspected | `inspect:no_action`, `inspect:adjusted`, then `select:proposal` | retained | inspected |
| alternative compared | `select:proposal`, citing severance for proposal and no action | retained | compared |
| original retained | `select:proposal` | retained | no_observable_comparison |
| another alternative selected | `inspect:adjusted`, `select:adjusted` | selected | inspected |
| no action selected | `inspect:no_action`, `select:no_action`, citing severance | selected_no_action | inspected, compared |
| proposal revised | `revise` to last day 2026-11-03 | revised | no_observable_comparison |
| proposal abandoned | `abandon` | abandoned | no_observable_comparison |
| invalid alternative | `select:without_1`, which is not listed | invalid_alternative | no_observable_comparison |
| no decision | no reply | no_decision | no_observable_comparison |
| whole table copied | `select:proposal`, citing every row of the compared table | retained | no_observable_comparison |

**How a reply is classified:**
* A revision whose batch equals a listed alternative is classified as
  that alternative.
* A select, revise or abandon with contradictory fields is `no_decision`.
* Nothing is submitted for `selected_no_action`, `abandoned`,
  `invalid_alternative` or `no_decision`.
* The batch is revalidated by the engine before it would be submitted,
  and a refusal there is `invalid_alternative`.
* Nothing is ever defaulted.

**When citations count as `compared`:**
* each item is shown for each alternative named;
* each entry names at least two alternatives, one of them not the one
  chosen;
* every value equals the shown value, as a number or as the formatted
  text;
* at least one entry shows a difference;
* there are at most three entries, fewer than the items shown, and not
  every row of the compared table.

The reply's reason text is recorded and never read: a reason claiming a
comparison changes no flag.

## 2. Decision sensitivity

**The scripted rules:** three deterministic reply rules, each a pure
function of the agent comparison, are applied to pairs of real engine
states:
* **always-retain:** select the proposal.
* **no-action-if-displacing:** select no action, citing displaced
  bookings, if the proposal displaces any; otherwise retain.
* **later-last-day-if-listed:** select `adjusted` if it is listed;
  otherwise retain.

**The pairs:** real decision points of mini, seed 52. They differ in the
named fact and, because they are real states, possibly in others, which
are listed.

| Pair | Compared facts that differ | always-retain | no-action-if-displacing | later-last-day-if-listed |
|---|---|---|---|---|
| Displaced bookings: E04 at Oct 13, last day Oct 27, vs E04 at Oct 27, last day Nov 10 | displaced bookings 0 vs 18; slots closed booked now 0 vs 18; booked ahead 0 vs 277 | not sensitive | **sensitive** (retained → selected no action) | not sensitive |
| Severance: E05 vs E06, last day Oct 27 | severance $4,500 vs $4,345; contract pay to the last day; slots closed 684 vs 872 | not sensitive | not sensitive | not sensitive |
| Later-last-day alternative listed: E04, last day Oct 27 vs Feb 1 (beyond the horizon for +14 days) | notice 14 vs 111 days; contract pay to the last day; slots closed 2,481 vs 235 | not sensitive | not sensitive | **sensitive** (selected adjusted → retained) |

**What this shows:**
* The measurement detects a change of result exactly where a rule uses
  the fact that changed, and nowhere else.
* Sensitivity is a property of results across paired states. It is not
  evidence of reasoning, and it does not say that any result is a better
  decision.

**Its limit:** the pairs differ in more than one fact. A rule that
reacts to an unlisted difference would also show as sensitive. The
listed differences say which facts could explain a change.

## 3. Termination versus keep, under the continuation (evaluation only)

These values are **class B**:
* They use the simulator's realized future, which no manager knows at
  the decision point.
* They are not in the agent comparison or its presentation.
* They are exact for mini, seed 52, this engine and
  `no-further-action-v1` only.

Each value is the change from 2026-10-13 07:00 to the 2026-12-11
horizon. The horizon is the adjusted alternative's end date plus 30
days, common to all three branches.

| Metric | proposal | no_action | adjusted |
|---|---|---|---|
| Orders received | 2,881 | 2,881 | 2,881 |
| Walk-ins arrived | 438 | 438 | 438 |
| Orders booked | 2,561 | 2,849 | 2,629 |
| Orders lost | 407 | 125 | 337 |
| Exams | 1,963 | 2,223 | 2,035 |
| Net revenue | $280,733.66 | $337,682.22 | $295,327.78 |
| Wages | $155,468.82 | $165,992.91 | $158,567.04 |
| Severance recognized | $3,950.00 | $0.00 | $3,950.00 |
| Operating profit (window convention) | −$122,414.64 | −$84,478.85 | −$114,337.90 |
| Cash at the horizon | −$134,867.02 | −$141,134.01 | −$134,222.62 |
| Wait to exam, 90th percentile (takeover to horizon) | 2,598 s | 2,850 s | 2,687 s |

**What these values are:**
* Orders received and walk-ins are identical in all three branches: the
  branches share the exogenous world.
* Each difference is the engine's response to the alternative, under no
  further management action.
* Lost orders are realized simulated outcomes, and they are
  evaluation-only.
* None of these values is a saving, a loss or a verdict.

**An earlier figure, reproduced:** the V0.13a mechanics check's 45-day
pair (to 2026-11-27) is reproduced exactly by `TestBranchTerminateVersusKeep`:
* exams 1,481 vs 1,653;
* lost 260 vs 83;
* 2,224 orders in both.

The 177 more lost orders it showed are such an outcome, also
evaluation-only.
