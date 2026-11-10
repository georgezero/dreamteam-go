# V0.13m-lite engineering gates

Human presentation review PENDING. Protocol lock, live inference and V0.13.0 release NO GO. These are engineering checks; the bounded technical acceptance review that interprets them is [ACCEPTANCE_REVIEW.md](../ACCEPTANCE_REVIEW.md) (implementer-performed; not an independent or human review).

Starting commit: `133e21d2181c10f910da00c0339fce77257d51f9`. Commits: `811dc0f` (baseline and acceptance standard), `15cdeed` (implementation and successor gates), `4dca48d` (evidence, contracts, human-review preparation, CI steps), then the gate record and review. Heavy checks ran on horo in `/home/george/dt-v013m` (Linux amd64, Go 1.27.0, Python 3.13.5) from a transfer of the tracked and untracked non-ignored working tree (never the protected swap file or dt). The Pi ran only focused tests in the 2 GiB memory scope at reduced priority. No model endpoint was contacted; all model replies are scripted.

| Location | Gate | Result |
| --- | --- | --- |
| Pi | Baseline identity | PASS; HEAD = origin/main = 133e21d; V0.13l and V0.13k preservation and identity checks pass; 3,606 protected files recorded |
| Pi | Focused successor tests | PASS; `test_v13m` non-binary classes (strict response JSON, acceptance standard, strict JSON, contract) |
| horo | Full Go suite | PASS; 11 packages |
| horo | Race suite | PASS; 11 packages |
| horo | Vet default, calibration and acceptance | PASS; three modes |
| horo | gofmt | PASS; 333 source files |
| horo | Cross-builds | PASS; linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64 |
| horo | Golden references | PASS; four references, outcomes identical |
| horo | Benchmark verify/reproduce | PASS; 86 verification checks, 59 reproduced outcomes |
| horo | Historical replay inputs | PASS; the four recorded V0.08c/V0.09c replay suites regenerated from scratch and verified (40 sessions, every digest; recorded sessions only); four targeted historical Go tests |
| horo | Frozen evidence | PASS; 141 recorded frozen-evidence replays verified; decision and parameter evidence reproduced |
| horo | V0.12b analysis | PASS; 420 draws, outputs byte-identical |
| horo | V0.12a and V0.11b comparisons | PASS; 27 scripted cases; 720 draws, 572 calls, 2,299 engine requests |
| horo | V0.13d/f/g/h fidelity, protocol, schedule, checks and packages | PASS; byte comparisons passed |
| horo | V0.13i witnesses, mutations, matrix sample | PASS; 71, 69, 61/601 |
| horo | V0.13j deterministic evidence | PASS; 4,916 fault cases at 1,511 points, 326 real signals, 12 loopback; byte-identical |
| horo | V0.13k deterministic evidence (unchanged modules) | PASS; 350-case binding matrix, fidelity, witnesses, mutations, regressions, examples, identities |
| horo | V0.13l deterministic evidence (unchanged modules) | PASS; protocol, schedule, check, examples, 350 + 119 binding matrix, 16 regressions, 125 mutations, fidelity byte-identical; DT13K witnesses 27/27; 76/76 witnesses agree (their committed file lacks the later `journalOutcomeKs`/`receivedKs` observation fields, as the V0.13l reviewer also found; every other field identical); preservation and identities |
| horo | Python suites | PASS; 742 tests, no failures: 689 preserved (agents-llm 168, acknowledgment 51, evidence 147, probes 3, repeatability 88, historical alternatives 29, lifecycle specs 4/5/5/8, v13g 21, v13h 35, v13i 21, v13j 25, v13k 34, v13l 45) and 53 successor (`test_v13m`, with the engine binary) |
| horo | Successor protocol, schedule and check | PASS; 21 contexts, 262 scheduled draws (entries identical to V0.13l), 802-call ceiling |
| horo | DT13L witnesses before/after (`v13m_dt13l.py`) | V0.13l 24/56 and the pre-fix successor 24/56 meet the remediation expectation; V0.13m-lite 57/57 (including one verifier-defense case) |
| horo | Review probes (`v13m_review_probes.py`) | V0.13l 7/14; V0.13m-lite 14/14 |
| horo | Reviewer's original V0.13l CLI archives | V0.13l verifier: both accepted (exit 0); V0.13m-lite verifier: both refused (exit 1) on R-JSON-004 / R-HALT-003 |
| horo | Binding matrix | PASS; 350 inherited (35 accepted, 315 rejected) + 119 relationship cases (17 valid accepted, 102 contradictions rejected, 17 persistence interruptions INVALID with evidence preserved); zero survivors; byte-identical on regeneration |
| horo | Durable ownership (`v13m_preservation.py`) | PASS; 6 inherited, 9 response-handoff, 258 completed-draw line boundaries, 62 response line boundaries |
| horo | Successor lifecycle fault matrix | PASS; 5,288 cases at 1,619 points, all agree; zero false COMPLETE; full transition and stop-state coverage |
| horo | Successor real-process signals and termination | PASS; 353/353 |
| horo | Successor real localhost loopback sockets | PASS; 12/12 |
| horo | Successor regressions, witnesses, mutations | PASS; 16/16, 76/76, 125/125 (byte-identical on regeneration) |
| horo | Successor lifecycle examples | PASS; 17 (the V0.13l 16 plus `unrecorded-draw-retained-calls-complete-refused`, whose full analysis is refused under R-HANDOFF-003) |
| horo | Successor fidelity with historical replay inputs | PASS; 66 units, 62 scripted calls, 834 engine events; full analysis COMPLETE/VERIFIED; byte-identical on regeneration |
| horo | DT13K witnesses on the successor | 35/35 meet the V0.13l remediation expectation |
| horo / Pi | Protected evidence, seal contract, human-review freeze, identities | PASS; 3,606 protected files; finalization subtree equal; 93 transcripts and seven assets (`v13i_human_review.py --check`, `MANIFEST.sha256`); `v13m_identities.py --check`, `v13l_identities.py --check`, `v13k_identities.py --check` |
| CI | Both cross-platform workflows (ubuntu-latest, ubuntu-24.04-arm, macos-latest) | Run IDs and results are recorded in the owner handoff; the triggering commit cannot contain its own results |

Development record: the first examples, binding-matrix and witness runs used the V0.13l `draw_unrecorded` base, whose single retained call already completes the workflow; under R-HANDOFF-003 that base is refused, so the successor base uses a first reply that needs a correction call (the retained call cannot complete the workflow), and the V0.13l construction is kept as the refused example. The first review-probe run changed a transport-failure (empty) body in three bases and was corrected (see the review). Historical gates were first started in a tree copied from the V0.13l gate tree whose generated outputs already existed; those outputs (explicitly named) were removed and the gates rerun from scratch. CI for the evidence push `4dca48d`: both Linux jobs of each workflow passed; the macOS jobs were cancelled by the hosted-runner service ("not acquired ... capacity constraints"), not by a failure. Superseded runs are excluded from the counts above.

External retained evidence: `/home/george/dt-audit-kept/v013m/` (gate archive, scripts, final-commit snapshot and checksums).
