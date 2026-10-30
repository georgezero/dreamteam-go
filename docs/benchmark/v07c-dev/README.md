# V0.7c: frozen agents under overtime and solvency economics

The two stages of [PLAN.md](PLAN.md) (`v07c-plan/1`, committed in 6182f6c
before any run of either stage, with no amendment).

**Not a ranking.** There is no score, leaderboard or significance test, and
no claim beyond these seeds, models and settings. No result here is a
clinical claim. "Appropriate" is the owner's to judge. This document gives
the figures with the agents' recorded actions and reasons, and it never
infers a model's reasoning from an outcome. Every comparison is within
V0.7, paired with `noop` on the same entry and seed. V0.6c is context only.

## Files

| File | What it is |
|---|---|
| `PLAN.md` | The plan, as declared |
| `stage-a.json`, `stage-b.json` | The suite-run configurations, as run |
| `pipeline-check.json` | The pre-plan pipeline check (PLAN §2); not a result |
| `analyze.py` | The tables of PLAN §5 from a results directory (declared with the plan) |
| `stage-<s>/contract.txt`, `contract.json` | The contract report the runner wrote |
| `stage-<s>/report.txt`, `report.json`, `manifest.json` | The harness's report and the manifest |
| `stage-<s>/tables.md` | `analyze.py`'s output: outcomes, the pairs, actions at every stage, refusals for funds, LLM decisions, the LLM action timeline with cash and credit, inference |
| `stage-<s>/<entry>-<model>-s<seed>-decisions.jsonl` | Each LLM run's recorded decisions (the agent's `policy-evidence/decisions.jsonl`): prompt and memory hashes, every attempt, refusal and error, the reason, and the batches submitted |
| `stage-<s>/<entry>-<model>-s<seed>-usage.json` | Each LLM run's usage summary: requests, retries, tokens, latency |
| `stage-<s>-replay-<entry>.json` | The replay configurations, one per stage and entry |

Kept outside the repository: the run directories and the model exchanges
(`policy-evidence/model.jsonl`).

**Build marker.** The binary reports `0.7.0-dev.3` built from 6182f6c with
`+dirty`. The only difference from the commit was an untracked editor swap
file under `docs/audits/`, which is not part of any build input.

## 1. Runs (deliverables 2 and 3)

| Stage | Seeds | Days | Runs | Completed | Failed | Reruns | Wall clock (Pi 4) | Within limit |
|---|---|---|---|---|---|---|---|---|
| A | 41601, 41602 | 30 | 50 (20 LLM) | 50 | 0 | 0 | 1 h 26 min | yes (6 h) |
| B | 41701 | 90 | 25 (10 LLM) | 25 | 0 | 0 | 3 h 59 min | yes (10 h) |

* **All 75 runs completed**, each replay-verified by the runner. No
  infrastructure error occurred, so the rerun rule (PLAN §3) was never
  used.
* **Starting states** are identical across the five policies of every entry
  and seed.
* **Both models** answered every decision point that did not fall back.
  Fallbacks (no action after the retry) per LLM run: stage A 0–6, stage B
  0–4. They are counted in [stage-a/tables.md](stage-a/tables.md) §6 and
  [stage-b/tables.md](stage-b/tables.md) §6.
* **Stage B** ran after stage A, whatever stage A showed, with the seed,
  horizon and policies of the plan.

## 2. Funding and credit across the 75 runs (deliverable 4)

| | Stage A (50 runs) | Stage B (25 runs) |
|---|---|---|
| Runs that drew on the credit line | 0 | 1 (Qwen, staffing shortage) |
| Interest charged | $0 | $14 (that run) |
| Unfunded shortfall | 0 | 0 |
| Actions refused `insufficient_funds` (validate / apply) | 0 / 0 | 0 / 0 |

This confirms what V0.7b found with the baselines. The five practices
start with enough liquidity that the funding rule did not bind on any
action, in any run, for any policy. The one draw came from a sequence of
decisions described in §5.

**What was drawn.** In that run cash reached $0 on 2026-12-01, and the line
held $4,406 at the end of that day. It reached $0 again on 2027-01-01 to
01-03, at most $17,200 drawn, and was repaid by 2027-01-04. Severance of
$27,195 was paid in the window.

## 3. Stage A: 30 days, seeds 41601 and 41602

**Baselines.**
* `greedy` raised reading allocation 4 times per run (2 in effect within
  30 days). Paired with `noop`: profit −$30 to −$1,387, breaches −1 to
  −25.
* `conservative-v2` made no change, except one raise in stable 41601 that
  was not yet in effect at the end of the window.

**LLM agents, paired with `noop`** (executed = accepted and in effect by
the end of the window; overtime cost includes the approved extensions):

| Entry | Seed | Model | Executed | ΔProfit | ΔTAT p90 (min) | ΔBreaches | ΔOvertime cost | ΔLowest cash |
|---|---|---|---|---|---|---|---|---|
| financial pressure | 41601 | Qwen | 16 | −$15,786 | −41 | −1 | +$12,923 | −$10,096 |
| | | Gemma | 0 | $0 | 0 | 0 | $0 | $0 |
| financial pressure | 41602 | Qwen | 4 | −$4,684 | −2 | −1 | +$804 | −$1,732 |
| | | Gemma | 0 | $0 | 0 | 0 | $0 | $0 |
| reporting backlog | 41601 | Qwen | 7 | −$3,592 | −78 | −61 | +$1,393 | −$1,166 |
| | | Gemma | 1 | −$1,383 | −43 | −23 | $0 | $0 |
| reporting backlog | 41602 | Qwen | 8 | −$5,381 | −29 | −26 | +$1,394 | −$1,904 |
| | | Gemma | 0 | $0 | 0 | 0 | $0 | $0 |
| demand growth | 41601 | Qwen | 11 | −$8,435 | −83 | −43 | +$1,855 | −$2,784 |
| | | Gemma | 1 | −$2,736 | −68 | −39 | $0 | $0 |
| demand growth | 41602 | Qwen | 1 | −$3,197 | −146 | −60 | $0 | −$920 |
| | | Gemma | 0 | $0 | 0 | 0 | $0 | $0 |
| staffing shortage | 41601 | Qwen | 16 | −$20,304 | −604 | −33 | +$2,502 | −$7,817 |
| | | Gemma | 0 | $0 | 0 | 0 | $0 | $0 |
| staffing shortage | 41602 | Qwen | 14 | −$13,440 | −17 | −2 | +$2,102 | −$4,623 |
| | | Gemma | 1 | −$2,043 | −597 | −21 | $0 | $0 |
| stable | 41601 | Qwen | 19 | −$9,401 | −69 | −37 | +$6,459 | −$4,632 |
| | | Gemma | 0 | $0 | 0 | 0 | $0 | $0 |
| stable | 41602 | Qwen | 9 | −$4,637 | −7 | +1 | +$3,710 | −$4,637 |
| | | Gemma | 0 | $0 | 0 | 0 | $0 | $0 |

* **Qwen** acted in all ten runs. Its actions were mostly `extend_shift`
  (in 9 of 10), plus teleradiology, worklist policy, reading allocation and
  one hire per staffing-shortage seed.
* **Gemma** submitted actions in four runs, all teleradiology contracts;
  one in each of three runs was in effect within the window.
* **Exams and lost orders** changed by at most 11 in any stage A pair.

## 4. Stage B: 90 days, seed 41701

**Baselines.**
* `greedy`: 11 raises in effect per entry; profit −$29,022 to −$33,550.
* `conservative-v2`: no change except two raises in stable (profit −$2,712,
  breaches −12).

**LLM agents, paired with `noop`:**

| Entry | Model | Executed | ΔProfit | ΔTAT p90 (min) | ΔBreaches | ΔExams | ΔLost | ΔOvertime cost | Approved overtime cost | ΔLowest cash |
|---|---|---|---|---|---|---|---|---|---|---|
| financial pressure | Qwen | 25 | −$19,149 | −32 | −32 | 0 | 0 | +$10,477 | $10,154 | −$6,298 |
| | Gemma | 0 | $0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |
| reporting backlog | Qwen | 16 | −$8,007 | −35 | −38 | +11 | −8 | +$6,865 | $7,147 | −$8,122 |
| | Gemma | 3 | −$22,096 | −285 | −340 | 0 | 0 | $0 | $0 | −$4,998 |
| demand growth | Qwen | 18 | −$20,914 | −333 | −304 | +6 | −7 | +$2,357 | $2,397 | −$6,149 |
| | Gemma | 28 | −$5,923 | −180 | −108 | +35 | −28 | +$1,875 | $2,562 | −$2,152 |
| staffing shortage | Qwen | 40 | −$181,589 | −635 | −79 | −867 | +1,450 | −$2,218 | $4,786 | −$37,228 |
| | Gemma | 54 | −$3,687 | −17 | −106 | +54 | −53 | +$4,617 | $5,164 | −$6,627 |
| stable | Qwen | 5 | −$51,560 | −805 | −299 | +4 | 0 | +$1,575 | $1,615 | −$9,823 |
| | Gemma | 0 | $0 | 0 | 0 | 0 | 0 | $0 | $0 | $0 |

* **Gemma** made no change in financial pressure or stable. It ran
  teleradiology in reporting backlog. It extended shifts in demand growth
  (27) and staffing shortage (53).
* **Qwen** acted in every entry. The staffing-shortage run is described in
  §5.
* **The flag in tables.md §3** ("better service, lower profit") reads only
  TAT and breaches. For Qwen in staffing shortage, TAT and breaches fell
  while 867 fewer exams were completed and 1,450 more orders were lost.

## 5. The staffing-shortage run that drew on credit (Qwen, stage B)

The timeline is in [stage-b/tables.md](stage-b/tables.md) §8; the reasons
below are quoted from the recorded decisions.

* **October to early November.** Qwen hired two technologists (XR/CT from
  10-14, MR from 11-03), extended shifts 14 times and changed worklist
  policy. Cash at the end of the day stayed between $39,178 and $216,084.
* **2026-11-27.** It terminated the new MR technologist and reduced Dr.
  Blake's reading allocation to 50%: "To address the financial loss and MR
  booking pressure, I will terminate the non-productive MR technologist
  H00002 to save costs".
* **2026-12-01 to 12-07.** On 12-01, cash was paid out at the start of the
  month and the line was drawn. Over these seven days it gave notice to the
  other new hire and then to every remaining technologist and the front
  desk:
  * 12-01: "Cash is $0 and critically low; immediate cost reduction is
    required to avoid insolvency before payroll on 12-11."
  * 12-07: "To prevent insolvency, I must terminate the remaining active
    staff members … though it will halt production. This is a survival
    move."
* **2026-12-11.** "Cash is critically low ($2,698) … while receivables are
  $347k." It resubmitted the seven terminations, which were already in
  notice and recorded as unchanged.
* **Afterwards.** From 12-21, when the last notice ended, the practice had
  no technologist and no front desk until the end of the window; the
  operations manager and the two radiologists remained. Two hires submitted on 2027-01-10 were
  accepted, with start dates after the window.
* **Outcome against `noop`:** profit −$181,589, revenue −$138,515, exams
  −867, lost orders +1,450, lowest cash $0.
* **The credit line.** No recorded reason in this run mentions the credit
  line, although the summary observation carries its limit, the amount
  available and the funding headroom. Across all 30 LLM runs, one
  recorded reason mentions it: Qwen, reporting backlog, stage B, 11-03 ("a
  healthy cash position relative to the credit line").

## 6. Refusals and responses (deliverable 5)

* **Refused `insufficient_funds`:** none, at validate or at apply, in 75
  runs. Whether agents repeat actions refused for funds could not be
  observed.
* **Refused at apply:** none for any policy. Every refusal was at the
  agent's validate step, before submission.
* **Refusal codes** (per run in tables.md §7):
  * Qwen and Gemma were refused for timing: `extend_shift`
    `after_close`, `not_on_shift`, `exceeds_max_shift`, `no_coverage`;
    `contract_teleradiology` `notice_too_short`, `contract_overlap`; `hire`
    `lead_too_short`.
  * They were also refused for form: `invalid_schema`, `invalid_minutes`.
* **Repeated invalid proposals** (the same invalid action again after a
  refusal):
  * Qwen: up to 15 per run in stage A (reporting backlog 41602) and up to
    13 in stage B (staffing shortage).
  * Gemma: up to 27 in stage B staffing shortage (`extend_shift`
    `not_on_shift` 17, `after_close` 12).
  * The agent's memory keeps each refusal as a lesson. The recorded reasons
    often name the earlier refusal and the correction (for example
    "The previous allocation change was refused because it was effective
    too soon … I will reschedule").
* **First-attempt validity:** Qwen 16/30 to 29/30 per run in stage A and
  76/90 to 88/90 in stage B. Gemma 24/30 to 30/30 and 63/90 to 90/90.

## 7. The financial-reasoning questions (PLAN §5.6)

Intent is quoted from recorded reasons only. No answer is a verdict.

**1. Do agents keep extending shifts after the overtime cost is visible?**
* **Overtime cost is in the summary from the first decision.**
* **Qwen** extended shifts in 14 of its 15 runs. Approved overtime in
  stage B was $1,615–$10,154 per run.
* **Its reasons name the cost both ways.**
  * 67 of its no-action decisions cite overtime cost as a reason not to
    extend,
    for example: "Extending shifts today would incur overtime costs
    without adding capacity" (demand growth 41601).
  * 12 of its decisions that acted mention it, for example: "while minimizing
    overtime costs by ke[eping the extension short]" (stable 41601).
* **Gemma** extended shifts only in stage B demand growth and staffing
  shortage, 80 times in all. Its reasons mention overtime in five
  decisions across all its runs, each a decision not to act; one of them
  is in those two runs (staffing shortage, 10-28, after its first
  extensions).

**2. Do they try to spend beyond the funding headroom?** No action by any
policy was refused for funds. The headroom never bound.

**3. Do they repeat actions refused for insufficient funds?** Not
observable: there were none. Repetition after other refusals is in §6.

**4. Do they reduce spending when cash or headroom falls?**
* **Qwen, staffing shortage, stage B:** yes, by terminating staff, with
  the reasons quoted in §5. Its reasons cite cash, payroll and receivables,
  never the credit line.
* **Qwen, financial pressure, stage B:**
  * On 12-02 it recorded "Cash is critically low ($27,499)" and did not
    act.
  * From 12-03 to 12-18 it extended Dr. Avery's shift 11 more times, with
    reasons about the reading backlog. Day-before cash was $34,079 to
    $108,856.
  * On 2027-01-01 it recorded "no immediate operational changes can
    generate revenue or reduce costs fast enough to prevent insolvency"
    and took no action. Day-before cash was $19,740, with $50,000 of
    unused credit.
* **Gemma:** its recorded reasons do not discuss cash. It made no change
  in the two entries with the least cash.

**5. Are service improvements accompanied by their financial cost?**
* **Every pair that improved TAT or breaches lowered operating profit and
  cash,** in both stages and for every policy that acted. No pair improved
  service and raised profit.
* **The size varies widely:**
  * Gemma, demand growth, stage B: −180 min TAT p90 and −108 breaches for
    −$5,923.
  * Qwen, stable, stage B: −805 min and −299 breaches for −$51,560.
* **Qwen, staffing shortage, stage B,** is the exception to "improved":
  service fell in volume (§5).

**6. Do they avoid unnecessary intervention in the stable practice?**
* **Gemma:** no change took effect in its three stable runs. One
  teleradiology contract was accepted in 41601 and not yet in effect.
* **`conservative-v2`:** one raise not in effect (A), two raises in
  effect (B).
* **Qwen** acted in all three:
  * 19, 9 and 5 executed actions;
  * profit −$9,401, −$4,637 and −$51,560;
  * breaches −37, +1 and −299.
* **Qwen's stage B reasons** cite a reading backlog ("2 overdue reports
  and a backlog of 7 unread studies") for a teleradiology contract on
  every scope, two shift extensions and a premium MR hire. Whether that
  intervention is unnecessary is the owner's to judge.

## 8. Inference (live; not replayed)

* **Requests per run:** stage A 30–44, stage B 90–117.
* **Latency:** mean 3.2–17.2 s per request, maximum 52.1 s.
* **Prompt tokens:** about 0.96 to 1.47 million per 90-day run.
* **No infrastructure error.** Cost: none configured.

Figures are in tables.md §9. They depend on the server and are not
deterministic.

## 9. Replay and cross-platform (deliverables 6 and 7)

* **Suite checks:** `benchmark verify` of the live results on the Pi.
  Stage A: 464 checks; stage B: 239 checks; 0 failed.
* **LLM replay without a model (linux/amd64):**
  * Each stage's ten replay configurations replay the 30 LLM runs from
    their recorded decisions. The prompt and memory hashes are checked at
    every point.
  * Every replay suite verifies: stage A 68 checks, stage B 41 per entry;
    0 failed.
  * The replayed `decisions.json`, `result.json` and `session.json` of
    all 30 LLM runs are **byte-identical** to the live linux/arm64 runs.
    That includes the ledger, the credit draws, repayments and interest.
* **Repeatability:**
  * `benchmark reproduce` of every replay suite: 47 checks per stage A suite and 29 per stage B suite, 0 failed, on linux/arm64 (Pi) and linux/amd64.
  * The ten replay manifests are byte-identical on linux/arm64 and
    linux/amd64: all ten identical.
* **CI** runs the ten replay configurations on linux/amd64, linux/arm64
  and darwin/arm64 and compares the manifests byte for byte.

## 10. Limitations

* **One to two seeds per stage,** two models, one prompt; no inference
  about other seeds, models or settings.
* **The funding rule was not exercised.** No action was refused for funds,
  so the refusal-response questions (2 and 3) have no observations. This
  is a property of the five practices' liquidity, not of the agents. No
  lower-cash scenario was added (not approved).
* **Overtime cost is reported before benefits,** as in V0.7a. Manager
  overtime is not modelled.
* **The agents' summary carries the financing block,** but the prompt was
  not changed for V0.7. Whether an agent read a figure can only be seen
  where its recorded reason names it.
* **Live inference figures** (latency, tokens) are not reproducible. The
  decisions they produced are.
