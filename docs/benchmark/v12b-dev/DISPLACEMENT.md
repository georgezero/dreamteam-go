# V0.12b Stage A: real-engine displacement test

`TestLLMAgent6Displacement` (`internal/bench/llm_agent6_displacement_test.go`)
runs llm-agent 6 against the real simulator in a case where a termination
displaces booked appointments. The V0.12a real-engine test did not cover
this, because the mini scenario's first decision point has no bookings past
the earliest last day.

## Setup

* **Runs:** the mini scenario, 4 days, decision points daily at 07:00.
* **Seeds:** development seeds 739182645 and 5. These are not reference,
  holdout or reserved seeds.
* **Decision point:** the third (last day 2026-10-29), where terminations
  displace bookings. A probe of the earlier points showed the first point
  displaces none and the second only a few.
* **The model:** the test's mock server (`ackdisp*`,
  `internal/bench/llm_agent_test.go`). It decides only from the messages
  it receives.
* **Unchanged:** the simulator, llm-agent 6 and its contract.

| Mode | Proposal | Its acknowledgment |
|---|---|---|
| `ackdisp` | terminate the first active technologist | every listed item, as listed |
| `ackdisp2` | terminate the first two technologists, in one batch | every listed item |
| `ackdispnone` | as `ackdisp` | every item except the displaced bookings |
| `ackdispoff` | as `ackdisp` | the displaced-booking count minus one |
| `ackdispswap` | as `ackdisp`, then the second technologist instead | the first request is answered with the changed proposal and the first proposal's acknowledgment; the next request is acknowledged properly |

## What the test checks against the engine's answers

| Requirement | Check |
|---|---|
| The preview reports the exact count | The batch item's count equals the number of `displacements` in the engine's preview answer. Each termination's `displaced` field, which covers the whole batch, has the same value. |
| The acknowledgment includes the batch-level displacement | Exactly one `staffing_batch` item, whose `actions` list every action of the batch. No termination item carries a displacement consequence. |
| Each qualifying action is identified on its own | One `terminate` item per termination, in order. Each has its action index, its employee, and the engine's last day and severance. |
| A missing acknowledgment prevents submission | `ackdispnone`: 3 of 3 acknowledgments fail with `missing_item`, the result is withheld (`acknowledgment_not_accepted`) and no batch reaches the harness. |
| A wrong displacement count prevents submission | `ackdispoff`: 3 of 3 acknowledgments fail with `wrong_value`, and the result is withheld the same way. |
| A changed proposal invalidates the previous acknowledgment | `ackdispswap`: the first acknowledgment is not accepted. The failures include `proposal_changed` or `wrong_proposal`. The new proposal is submitted only at a later attempt, on its own accepted acknowledgment. |
| The final re-preview matches what was acknowledged | For every submission, the final check agrees. Its engine-answer hash and consequence-payload hash equal those of the acknowledged assessment. |
| Validation and preview change nothing | Every assessment reports the digests unchanged. Runs that are withheld end with final digests identical to the no-op policy's run on the same seed. |
| The accepted action displaces what was acknowledged | The run's displaced-appointment events (`displacedAppointments`) equal the acknowledged preview's count. The no-op runs displace none. |
| Replay reproduces the result | The recordings replay without the model, which receives no calls. Final digests, `result.json`, the session and the harness decisions are byte-identical to the live runs. |

## Results

From `go test ./internal/bench -run TestLLMAgent6Displacement -v` at
9871e37 plus this test:

| Run | Displaced (proposal) | Outcome |
|---|---|---|
| mini-s739182645-ackdisp | 10 | E03 submitted at attempt 2, accepted; 10 displaced when applied |
| mini-s739182645-ackdisp2 | 17 (E03 and E04 together) | both submitted at attempt 2, accepted; 17 displaced |
| mini-s739182645-ackdispnone | 10 | 3 rejected (`missing_item`); withheld; final digests equal the no-op run's |
| mini-s739182645-ackdispoff | 10 | 3 rejected (`wrong_value`); withheld; final digests equal the no-op run's |
| mini-s739182645-ackdispswap | 10, then 7 for E04 | E04 submitted at attempt 3 on its own acknowledgment; 7 displaced |
| mini-s5-ackdisp | 15 | E03 submitted at attempt 2; 15 displaced |
| mini-s5-ackdisp2 | 22 | both submitted at attempt 2; 22 displaced |
| mini-s5-ackdispnone | 15 | 3 rejected (`missing_item`); withheld; final digests equal the no-op run's |
| mini-s5-ackdispoff | 15 | 3 rejected (`wrong_value`); withheld; final digests equal the no-op run's |
| mini-s5-ackdispswap | 15, then 7 for E04 | E04 submitted at attempt 3; 7 displaced |

`ackdispswap` shows why an acknowledgment is tied to its proposal. The
first acknowledgment stated 10 displaced bookings, but the changed
proposal displaces 7. That acknowledgment was refused, and 7 was then
acknowledged before anything was submitted.

## Sensitivity check

The test's checks were confirmed to fail on a real defect. I made
llm-agent 6 temporarily ignore the value of `displaced_bookings` when
matching. With that change, `ackdispoff`'s wrong count was accepted and
the test failed (`an acknowledgment {Accepted:true Failures:[]}`). The
agent was then restored from git; the change was never committed.

## Limits

* The mini scenario is small: 7 to 22 displacements, not the roughly 100
  of V0.11b's point fl-qwen-s41902-p24. The count is compared exactly, so
  the size matters only to the evidence, not to the check.
* Displacement is reported for the batch as a whole. The engine does not
  attribute bookings to one termination, and neither does the agent:
  `ackdisp2`'s 17 is the engine's count for the whole batch. Here it
  equals the sum of the single-person counts (10 + 7, and 15 + 7 for seed
  5), but the agent does not compute or assume that sum.
* The model is scripted. These runs show how the workflow handles the
  engine's answers, not how a real model behaves.
