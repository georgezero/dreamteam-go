# Funding-refusal probes (V0.8b)

**What these probes do.** They are deterministic, scripted decision
sequences through the existing agent protocol, on the V0.7b solvency
scenarios. They test whether the simulator, the protocol and the evidence
pipeline represent a funding refusal and what follows it. **They do not
measure any model:**
* the "agent" is a script;
* whether a real model responds to a refusal is a question for V0.8c.

No model was contacted, and no action, mechanic or scenario was added.

**Files:**
* [probes.json](probes.json): the definitions, with every expected code.
* [probes-out/](probes-out/): the outputs:
  * the log;
  * the decisions in the llm-agent 2 recording format;
  * the attempt-level record.
* [probe-evidence.json](probe-evidence.json): the frozen v1 rules applied
  to the probe recording.
* [refusal-response.json](refusal-response.json) and
  [its tables](refusal-response-tables.md): the responses classified.

**Running them:** `analysis/probes/funding_probes.py`. The runner exits
non-zero if any expected code is not observed.

## How a scripted decision is made

**Submission.** A scripted decision is submitted the way the LLM agent
submits its own (`agents/llm/llm_agent.py`, `split`):
* the actions form an operations batch and a staffing batch;
* both are validated;
* both are applied only if both are valid.

**Refusal.** A refused attempt is followed by the decision's next scripted
attempt. A decision whose last attempt is refused falls back to no
action.

**Recorded for each decision point:**
* the public observation of funding: cash, current liabilities, reserved
  commitments, funding headroom and credit;
* every validate answer;
* the preview's funding object (commitment, headroom) for staffing
  batches;
* every apply answer;
* the code each probe expects and the code observed.

**Setup.**
* *Scenarios:* `bench-financial-pressure-ot-nocredit` and
  `bench-financial-pressure-ot-credit`.
* *Seed:* 41801, fresh: the next block after V0.7c's 41701, and not a
  reference, development, reserved-stream or holdout seed.
* *Dates:* chosen from a no-action scan of the no-credit practice's
  headroom on that seed. It falls from $55,217 on day 1 to $6,910 on
  11-17, and below zero from 11-20.

## Results

All 3 probes ran, and every expectation was met (0 unmet).

**Sequence without credit** (probes A–I), 2026-11-17 to 11-23:

| Point | Date | Funding headroom (reserved) | Probe | Scripted proposal | Validate / apply | Commitment |
|---|---|---|---|---|---|---|
| 0 | Tue 11-17 | $6,910.28 | A | hire an XR technologist (standard, start 2027-01-04) | `insufficient_funds`, decision falls back | $10,457.06 (recruiting fee and earliest lawful exit) |
| 1 | Wed 11-18 | $3,754.36 | E | the identical hire, twice (two attempts) | `insufficient_funds` ×2 | $10,457.06 |
| 2 | Thu 11-19 | $1,715.39 | I (before) | check: Dr. Avery +2 h | `accepted` | |
| | | | B | attempt 1: Dr. Avery's allocation 60% → 100% | `insufficient_funds` | $19,726.02 (30 days' salary increase) |
| | | | F | attempt 2: Dr. Avery's allocation 60% → 63% | `accepted`, applied | $1,479.45 |
| | | ($1,479.45 reserved) | I (after) | check: the same Dr. Avery +2 h | `insufficient_funds` | $577 against $235.94 left |
| 3 | Fri 11-20 | −$4,183.90 ($1,479.45) | D | Dr. Avery +2 h (paid radiologist extension) | `insufficient_funds` | $577 |
| 4 | Sat 11-21 | −$7,322.32 | C | teleradiology contract, all windows | `insufficient_funds` | $3,000 (retainer to the earliest end) |
| 5 | Sun 11-22 | −$13,641.19 | G | no action | no action | — |
| 6 | Mon 11-23 | −$18,724.76 | H | attempt 1: Dr. Avery +2 h | `insufficient_funds` | $577 |
| | | | | attempt 2: Dr. Blake's allocation 50% → 30% (a cost reduction) | `accepted`, applied | none: never refused for funding |

**Credit-limit boundary** (probe J), Tue 11-17, the same seed and day:

| Practice | Funding headroom | Varied action | Last funded | Its commitment | First refused | Its commitment |
|---|---|---|---|---|---|---|
| no credit | $6,910.28 = cash $91,401.86 − liabilities $84,491.58 | Dr. Avery's allocation | 7401 BP | $6,909.04 | 7402 BP | $6,913.97 |
| $50,000 credit line, nothing drawn | $56,910.28 = the same + $50,000 undrawn credit | Dr. Blake's allocation, after the probe A hire, a teleradiology contract and Dr. Avery at 100% in one batch | 9811 BP | $56,908.55 | 9812 BP | $56,913.48 |

**What J shows:**
* **Undrawn credit counts in the headroom:** the two practices' cash and
  liabilities are identical on that day, and their headroom differs by
  exactly the $50,000 undrawn credit.
* **The boundary is exact:** a batch committing up to the headroom is
  funded, and one committing more is refused.
* **Over the boundary, only the crossing action is refused:** Dr. Blake's
  raise is `insufficient_funds`, and the batch's other three actions are
  `batch_rejected`.
* **The probe A hire with credit:** the hire refused without credit is
  accepted with it.

**What the sequence shows:**
* **The four kinds of commitment are refused when unfunded:** A hire, B
  allocation raise, C teleradiology contract and D paid radiologist
  extension.
* **Reservations count:** a revised, affordable commitment (F) is accepted
  and reserved, and the reservation then refuses an extension that fitted
  before it (I).
* **A repeat stays refused:** an identical proposal is refused again (E).
* **No action commits nothing** (G).
* **A cost reduction is accepted with negative headroom** (H), as V0.7b
  designed.

## What the evidence pipeline makes of it

**The refusal-response extension**
(`dreamteam.refusal-response/v1`), reading the attempt-level record,
classifies the seven refusals:

| Refusal | Next proposal | Category |
|---|---|---|
| A hire, point 0 | the identical hire, point 1 attempt 1 | `identical_repeated` |
| E hire, point 1 attempt 1 | the identical hire, attempt 2 | `identical_repeated` |
| E hire, point 1 attempt 2 | Dr. Avery's raise, point 2 | `different_proposal` |
| B raise to 100%, point 2 attempt 1 | the raise to 63%, attempt 2 | `modified_proposal` |
| D extension, point 3 | the teleradiology contract, point 4 | `different_proposal` |
| C contract, point 4 | nothing, point 5 | `no_action` |
| H extension, point 6 attempt 1 | Dr. Blake's reduction, attempt 2 | `different_proposal` |

**The same probes read in the llm-agent 2 format** (only the final reply's
actions recorded) lose 3 of the 7: the refused first attempts of points 1,
2 and 6 become `evidence_unavailable`. That is the historical recordings'
gap, measured on a known sequence ([RECORDING_CONTRACT.md](RECORDING_CONTRACT.md)).

**The frozen v1 rules on the probe recording** (`probe-evidence.json`):
* **Execution states:** refused at validate at points 0, 1, 3 and 4;
  accepted at 2 and 6.
* **`REPEATED_INVALID_PROPOSAL`** finds the repeat within point 1 (the
  two attempts' refusals have the same message). It does not find the
  repeat across points 0 and 1. The two refusals' messages state that
  day's headroom ($6,910 and $3,754), and v1 normalizes dates only. This
  is defect V1-D1 of the [freeze record](../v08a-dev/FREEZE.md), found by
  these probes and documented, not fixed. The refusal-response extension
  identifies actions by payload and does find it.

**No probe tests `refusal_no_longer_applicable`:** the identical proposal
accepted after a refusal. Without credit, this practice's headroom stays
negative after 11-20, through December, so the case cannot be scripted
on this seed without a new mechanic. The unit tests cover it. It
occurred nowhere in the historical recordings.

## Determinism

* **Repeatability:** the probes run twice with identical outputs (the
  probe tests).
* **CI:** reruns them on three platforms, compares their outputs with the
  committed ones, and compares them across platforms.
* **What they depend on:** only the engine build, the scenario, the seed
  and the script.
