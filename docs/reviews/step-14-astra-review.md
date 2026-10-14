# Step 14 review — gpt-6-astra, high reasoning

* Reviewed: commit `16c53a2` (V0.1 step 14), read-only snapshot copied to a
  temporary directory (deleted afterwards).
* Command: `codex exec -m gpt-6-astra -c model_reasoning_effort="high" -s read-only`
* Prompt: a focused brief on replay correctness — outcome-relevant state
  not reproduced by replay (with the TUI-pause-at-a-shared-instant case as
  the prime suspect), digest completeness and canonicality, tampered records
  that could pass, compatibility and integrity, atomic saves, test gaps.
  No Go toolchain on the review host; review by reading code.
* The findings below are the model's output, unedited. File links point at
  the temporary snapshot; line numbers match `16c53a2`.

## Findings (verbatim)

**Verdict: NO GO**

Static review only; no files changed and no tests executed. The suspected TUI pause defect is **not present**, but accepted scenarios can compile nondeterministically, and verification does not bind the scenario to the run digest.

**Findings — ordered by severity**

**1. High — Accepted hour keys make scenario compilation nondeterministic**

Evidence: [validate_demand.go:288](/tmp/dt-review.FR6WOv/internal/scenario/validate_demand.go:288), [validate_demand.go:500](/tmp/dt-review.FR6WOv/internal/scenario/validate_demand.go:500).

Both hourly-profile parsers accept any two-character string that `strconv.Atoi` parses into 0–23. Consequently, `"08"` and `"+8"` are both accepted and write the same array element. Map iteration determines which value survives, while validation sums both values.

Concrete failure: a resolved scenario with `hourProfileBP: {"08":2500,"+8":7500}` passes the 10,000-share check but compiles hour 8 to either 2,500 or 7,500. Recording and replay can therefore generate different demand from identical scenario JSON, fingerprint, seed and actions.

Minimal fix: require the exact canonical spelling `fmt.Sprintf("%02d", h)` and reject duplicate normalized hours.

Regression: reject signed aliases in both referral and walk-in profiles; repeatedly compile accepted profiles and compare configurations.

**2. High — Identical run digests can represent different future simulation rules**

Evidence: configuration is excluded at [digest.go:42](/tmp/dt-review.FR6WOv/internal/sim/digest.go:42); the run hash omits the scenario fingerprint at [record.go:109](/tmp/dt-review.FR6WOv/internal/session/record.go:109). The fingerprint check only compares the embedded scenario with its accompanying fingerprint at [record.go:274](/tmp/dt-review.FR6WOv/internal/session/record.go:274).

Concrete failure: save `mini` immediately at its 06:00 takeover. Change `finance.perStudyITCents` from 300 to 301, update the scenario fingerprint and record checksum, and leave all five recorded digests unchanged. That parameter has not affected state yet; it is consumed when imaging completes at [finance.go:388](/tmp/dt-review.FR6WOv/internal/sim/finance.go:388). Verification accepts the altered record, although continuing it produces different expenses.

This requires no hash collision. Independently creating those two scenarios also produces identical initial run digests.

Minimal fix: bind the resolved scenario fingerprint into `Run`. If `StateDigest` is intended to identify future behavior independently, bind a canonical configuration digest there too. Version the changed digest encoding.

Regression: compare otherwise identical takeover records differing only in this future cost; require different run digests and rejection when the original digests are retained.

**3. Medium — `FromFile` retains a mutable scenario reference, allowing successful saves that cannot load**

Evidence: [session.go:52](/tmp/dt-review.FR6WOv/internal/session/session.go:52) caches the fingerprint but retains `f` directly. [record.go:131](/tmp/dt-review.FR6WOv/internal/session/record.go:131) later serializes that reference while reusing the old fingerprint.

Concrete failure:

```go
s, _ := session.FromFile(f, seed)
f.OpeningCashCents++
s.Save(path)
```

The save reports success, but loading rejects its scenario fingerprint. Reusing a scenario object to construct benchmark variants can therefore invalidate an already-running session’s saves. There is additional sharing through `EmployeeConfig.Skills` at [validate.go:251](/tmp/dt-review.FR6WOv/internal/scenario/validate.go:251).

Minimal fix: deep-copy the resolved scenario before fingerprinting, compiling and retaining it.

Regression: mutate scalar, map and slice fields of the caller’s scenario after construction; the original session must still save, verify and resume identically.

**4. Medium — The state digest still includes the unrelated-event clock cache**

Evidence: `state.now` records the last transaction time at [world.go:27](/tmp/dt-review.FR6WOv/internal/sim/world.go:27), is updated for every handled event at [world.go:222](/tmp/dt-review.FR6WOv/internal/sim/world.go:222), and is not excluded by [digest.go:41](/tmp/dt-review.FR6WOv/internal/sim/digest.go:41).

Concrete failure: advance two identical mini worlds to 06:00:01, inserting the existing `PushLiveNoise` roster-edge event only in one. The engine clocks and operational outcomes agree, but the ordinary world retains `state.now == 06:00:00` while the noisy world has `06:00:01`. Their state digests differ. Normalizing the metrics snapshot does not normalize this separate field.

This violates the documented unrelated-event invariance; it is not evidence of a normal TUI pause mismatch. The existing noise test compares only after the next midnight transaction, which overwrites both caches: [digest_test.go:42](/tmp/dt-review.FR6WOv/internal/sim/digest_test.go:42).

Minimal fix: exclude or normalize the cached transaction clock in the digest; ensure public projections use the engine clock where appropriate.

Regression: compare digests immediately after an unrelated event in an otherwise event-free interval, before another shared transaction.

**5. Medium — “Strict” decoding accepts trailing data**

Evidence: [record.go:265](/tmp/dt-review.FR6WOv/internal/session/record.go:265) uses `Decoder.More()` as an EOF check. [builtin.go:75](/tmp/dt-review.FR6WOv/internal/scenario/builtin.go:75) and the CLI actions reader at [main.go:169](/tmp/dt-review.FR6WOv/cmd/dreamteam/main.go:169) perform no EOF check.

Concrete failure: append `]` or `}` to a valid saved record. `More()` returns false on a closing delimiter, so the malformed file is accepted with its original checksum. Data following that delimiter is also unchecked. Separately, an actions file containing two concatenated arrays silently executes only the first.

Minimal fix: after decoding the expected value, perform another decode and require `io.EOF`.

Regression: test closing delimiters, another JSON value, malformed suffixes, and whitespace-only suffixes. The current record test covers only appended `{}`, which this implementation happens to reject.

**6. Medium — Verification accepts contradictory batch and descriptive metadata**

Evidence: structural checks at [record.go:346](/tmp/dt-review.FR6WOv/internal/session/record.go:346) require only increasing outer batch numbers. Replay applies actions without using those numbers at [record.go:391](/tmp/dt-review.FR6WOv/internal/session/record.go:391). Final comparison checks digests, not the recorded summary, at [record.go:413](/tmp/dt-review.FR6WOv/internal/session/record.go:413).

Concrete failure: change outer batch numbers from `1,2,3` to `101,102,103`, leaving results unchanged, and recompute the checksum. Verification succeeds although each batch disagrees with its results. Likewise, altered `Scenario.ID/Name`, `Start`, `Until`, batch `Time`, or summary cash/exam counts can pass. The CLI displays some of those unchecked fields as the verified run’s identity and timing.

Minimal fix: validate redundant fields against authoritative values: batch numbers/results, ISO timestamps/seconds, scenario identity/resolved content, and summary/replayed summary. Alternatively, remove redundant stored fields and derive them.

Regression: independently alter each field while recomputing only the checksum; require a specific mismatch.

**7. Medium — Save reports success when directory durability fails**

Evidence: [record.go:238](/tmp/dt-review.FR6WOv/internal/session/record.go:238) ignores both directory-open failure and `d.Sync()` failure. [record.go:194](/tmp/dt-review.FR6WOv/internal/session/record.go:194) then marks the session saved.

Concrete failure: the temporary file sync and rename succeed, but directory synchronization returns an I/O error. `Save` still returns success and clears the unsaved marker, although rename durability was not established.

Minimal fix: handle directory synchronization failures explicitly, distinguishing unsupported operations from real errors. A post-rename failure must report uncertain durability; the “failure leaves the old file untouched” promise cannot cover that stage.

Regression: inject directory-open and directory-sync failures after rename, checking the returned status and saved marker. The existing fault injection stops before rename.

**8. Low — Failed replay reports can claim success or show fictitious zero summaries**

Evidence: [record.go:339](/tmp/dt-review.FR6WOv/internal/session/record.go:339) initializes `Report.OK` to true before compatibility and compilation checks. [record.go:321](/tmp/dt-review.FR6WOv/internal/session/record.go:321) renders that as “digests match.” `play -load` prints this report even on failure at [main.go:85](/tmp/dt-review.FR6WOv/cmd/dreamteam/main.go:85).

Concrete failure: loading an engine-incompatible record prints the incompatibility error followed by an “OK … digests match” report, although nothing was replayed. Action-result mismatches return before `RepSummary` is populated, so their reports display zero values as the replayed summary.

Minimal fix: mark success only after completed verification; distinguish unavailable diagnostics from computed values, and capture the actual replay position on action mismatch.

Regression: assert report contents for engine rejection, compile failure and first-action mismatch.

**Things verified as correct**

- **The prime suspect is ruled out.** `transaction()` drains all events at time T before returning; stop callbacks run afterward. Dispatch and observers run under a phase that forbids pushes at T. Thus auto-pause, Max-budget stops, `StepVisible` and `EndOfDay` cannot leave later same-time events pending through these paths. See [engine.go:156](/tmp/dt-review.FR6WOv/internal/sim/engine.go:156), [engine.go:174](/tmp/dt-review.FR6WOv/internal/sim/engine.go:174), [engine.go:126](/tmp/dt-review.FR6WOv/internal/sim/engine.go:126).
- Replay reconstructs the world and uses `Apply`; it does not deserialize mutable simulation state. Actions within batches remain ordered, with one settle only when something changed.
- All five digest strings are compared in full. Truncation is confined to diagnostic display. Action-result comparison includes the submitted action itself, so behavior-equivalent action edits do not escape comparison.
- A changed seed is covered by `state.seed`. Swapped batches, duplicate outer batch numbers, empty recorded batches and batches beyond `until` are rejected by the relevant checks.
- Pending events retain relative `(At, Class, Seq)` order before encoding. Omitting numerical `Seq` values does **not** erase equal-time event ordering: [engine.go:243](/tmp/dt-review.FR6WOv/internal/sim/engine.go:243).
- Map entries are sorted for hashing; integers use fixed-width encoding; pointers are encoded by contents. The metrics substitution clones its slices and computes the tally at the engine clock without accruing live state.
- Record checksums compact `RawMessage` during marshaling, so save-file indentation alone does not break verification. I found no separate `omitempty` round-trip loss affecting `Compile`.
- Temporary names are unique and in the target directory; ordinary pre-rename failures clean up the temporary file. Saves explicitly set mode `0644`.

**Test gaps**

Beyond the regressions above:

- Save/replay **immediately after each stop mode**, apply a batch at that exact time, and resume. Existing budget tests primarily compare after continued advancement.
- Multiple separate batches at the same timestamp, including rejected/no-op batches and a save between batches.
- Save at takeover and exactly at a configured evaluation endpoint.
- Perturb equal-time event ordering; test passive-event scheduling and stale-event exclusion against continued behavior.
- Compare digests immediately before and after observations, previews and saves, rather than only after later transactions.
- Property-test accepted custom scenarios for canonical round-trip **and deterministic compilation**; current round-trip coverage uses only the two bundled scenarios.
- Exercise every digest independently, checksum-recomputed metadata edits, concurrent saves, post-rename failures, and cross-architecture golden digests.
## Assessment

All eight findings were confirmed in the code and by a regression test that
fails on `16c53a2` (finding 7 by inspection: the old code ignored the
directory-sync error and had no point to inject one, so its test cannot
build there). None changes a simulation outcome; they are about what a
record proves and how faithfully files are read.

* Finding 1 is the most serious for benchmarking: an accepted scenario
  could compile differently on two runs, so identical records could replay
  differently. The bundled scenarios use canonical keys and were not
  affected.
* Finding 2: the state digest deliberately excludes the configuration
  (fingerprinted separately), but the run digest did not bind the
  fingerprint, so a run digest did not identify the rules still to come.
* Findings 4–8 are integrity and reporting gaps the existing tests missed
  because they compared after a later shared transaction, appended only
  `{}`, or checked digests but not the readable copies.

The reviewer confirmed the prime suspect is not present: a transaction
drains every event at its instant before any stop is consulted, and
dispatch and observers cannot push at the current time, so a batch applied
after any pause sees exactly the state replay reconstructs.

## Resolution

Version 0.1.0-dev.14.1. Regression tests run first on `16c53a2`, then the
smallest fix, then the tests again:

| # | Regression test | Before (`16c53a2`) | Fix | After |
|---|---|---|---|---|
| 1 (high) | `TestRegress14HourKeysCanonical` | FAIL: `"+8"` alias accepted for demand and walk-in profiles | hour keys must be exactly `"%02d"` | PASS |
| 2 (high) | `TestRegress14RunDigestBindsScenario` | FAIL: same run digest for different future rules; a swapped-scenario record with old digests verified | the run digest hashes the scenario fingerprint and the seed with the components (separators between them) | PASS |
| 3 (medium) | `TestRegress14FromFileCopiesScenario` | FAIL: save after the caller changed its file did not load (fingerprint mismatch) | `FromFile` keeps a private copy (canonical encode → decode) | PASS |
| 4 (medium) | `TestRegress14DigestRightAfterNoise` | FAIL: digests differ right after an unrelated event | `state.now` (time of the last transaction) excluded from the state digest | PASS |
| 5 (medium) | `TestRegress14RecordRejectsTrailing`, `TestRegress14ScenarioDecodeRejectsTrailing`, `TestRegress14ActionsFileTrailing` | FAIL: `]`/`}` after a record, anything after a scenario, a second array in an actions file accepted | a second decode must return EOF (records, scenarios, actions files) | PASS |
| 6 (medium) | `TestRegress14MetadataChecked` (10 edits) | FAIL: 8 of 10 re-signed edits verified (batch numbers, batch time, scenario ID and name, start, until, summary cash and exams) | replay checks scenario identity against its content, every ISO time against its seconds, every result against its batch, and the summary against the replayed one | PASS |
| 7 (medium) | `TestRegress14DirectorySyncFailure` | (cannot build: no injection point; the error was ignored) | a directory-sync failure after the rename is reported ("written, but its durability is not confirmed") and the session stays unsaved; EINVAL/ENOTSUP (no directory sync on that file system) and Windows are not errors | PASS |
| 8 (low) | `TestRegress14FailureReports` | FAIL: an engine refusal printed "OK … digests match"; an action mismatch showed a zero replay summary | a report is OK only when everything was checked; a mismatch records where the replay stood | PASS |

Full suite (`go test -count=1 ./...`) passes. The run digest of every
session changed with fix 2, so records saved by `16c53a2` no longer verify
(they replay with the engine mismatch allowed and report the run digest);
the engine version moved to 0.1.0-dev.14.1 accordingly.

Not changed, recorded as test gaps for later steps: saving and applying a
batch immediately after each stop mode; several batches at one timestamp
with saves between them; saving exactly at takeover and at an evaluation
endpoint; property tests over generated scenarios for deterministic
compilation; concurrent saves; cross-architecture golden digests (CI).
