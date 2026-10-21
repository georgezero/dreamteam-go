# Slice 5C.4n: the certification pipeline

This is the final planned remediation before the release-gate review. It
answers the release blockers of the independent 5C.4m audit
(`docs/audits/slice-5c4m-provenance-protocol-lock-audit.md`, M1–M4 and
M6) and defers M5. The clinical policy, the simulator's behaviour, the
severity thresholds, the scenario manifest and the reserved seed
protocol are unchanged. The evaluator is 0.2.0-dev.24.

## 1. The trust boundary (M1)

**Before (5C.4l).** The reserved run wrote one JSON summary a group to a
file and dropped its records; a second invocation read that file and
judged it. The file was the only bridge between construction and
judgment, and its rows were assertions: the 5C.4m auditor wrote 3,904
rows by script, with no simulation, and the adjudicator certified them
`confirmatory` with every criterion PASS.

**Now.** One process does everything, and decides before it writes
(`TestReservedManifestRun` → `runCertified`):

1. it reads its settings strictly and refuses a malformed one (§4);
2. it verifies every runtime input (the candidate, the manifest, the
   protocol, the policy in force, the simulator profile) against the
   digest compiled into the build, and the manifest's pin of the
   candidate (`verifyInputs`);
3. it refuses the reserved family in this build (`familyAllowed`) and
   refuses to overwrite an output file;
4. it constructs every pair (the fidelity model), judges every group
   (`protocolDecision`, `gateBDecision`) and keeps the group lines in
   memory, discarding the records;
5. it adjudicates every criterion (`reservedCriteria`) and certifies
   (`certify`) from how the lines were computed: trusted, the family,
   the seeds;
6. only then it writes the lines and the adjudication, and logs the
   invocation, the certification and both files' SHA-256.

**What certifies.** `certify` reads the invocation as `confirmatory`
only for a trusted execution on the reserved family with every scenario
at its manifest seeds; the certification is then `certified GO` when
every mandatory criterion PASSes and `certified NO GO` otherwise. Every
other execution is `development` and `not certified`.

**What cannot certify.** A file of lines, read back by
`TestReservedCriteria` (now an offline analysis, `analyzeFile`), is
always `non-certifying analysis` and `not certified`, whatever its rows
say and whatever the settings (`TestFabricatedResultsCannotCertify`: a
fabricated, complete 3,904-line file on which every mandatory criterion
PASSes, under three settings). Nothing written by the execution is read
by it.

**The auditor's own files** (kept outside the repository), read by the
real reader of this build: each of the eight (fabricated, duplicated,
invalid, missing replicate, missing scenario, truncated at a line
boundary, truncated mid-line, empty) ends in a failing exit; none is
certified; `DT_MANIFEST_SEEDS=nonsense` and `-1` are refused before
anything is written. The fabricated file also fails the new measurement
checks (§2): its rows lack measurements the criteria read.

**Identity recorded.** A trusted execution's `invocation` records the
family, the verified input digests, the engine version and the SHA-256
of the package's Go sources as found at run time (`sourceSHA256`), so
that the run can be checked against the lock commit's tree. The test
binary itself is not hashed: the protocol's preconditions (a clean
checkout of the lock commit, every test passing there) bind it.

**What remains a file.** The written lines and adjudication are records
of the certified decision. Editing them afterwards cannot change it,
and a copy that does not match the logged digests is not the run's.

## 2. Measurement completeness (M2)

`lineProblems` checks every judged line before any criterion reads it;
a problem is incomplete coverage, so every criterion of that scenario is
INCONCLUSIVE unless it already FAILs:

* **presence and cardinality:** Gate B's status, increase, matched and
  deteriorated seeds, exactly one each a priority; the access claims;
  severe and acquired studies, one pair a priority; one worst soon delay
  a pair; for a near-margin replicate, at least one urgent seed
  difference;
* **ranges:** statuses known; matched seeds within the pairs; severe at
  most acquired; delays and counts nonnegative; served at most eligible;
  unimaged fractions in [0, 1]; urgent counts within their orders;
* **finite numbers:** increases, fractions, differences;
* **seed identities:** every seed a list names is one of the group's
  pairs, named once (a seed pending in two priorities is named twice,
  its form); at most as many urgent seed differences as matched seeds;
* **cross-field:** rejected exactly when a known rule is listed, no rule
  twice.

**The criteria themselves** no longer turn missing evidence into a
favourable reading:

* **T2G:** with no acquired soon or routine study, no worst delay or a
  missing increase, materiality is undetermined: INCONCLUSIVE, never
  non-material;
* **T5a:** a group with no urgent seed difference has no prediction
  (`predictAcceptance` is NaN, not 0): INCONCLUSIVE;
* **T8:** matched seeds must equal the group's pairs, never merely
  reach them. On an identity witness's first seed exactly that seed is
  unmatched: in every priority for a removed or repeated origin (the
  record's identity evidence is broken), in urgent alone for a changed
  urgent request (`idUnmatched`). The first development run of this
  build found the equality too strict there: W-ID-CHANGED's soon and
  routine priorities legitimately match all 40 pairs, which the old
  floor (`<`) had hidden. The expectation was made exact per kind, and
  the development runs were repeated on the corrected build;
* **T4:** "every seed" means exactly the group's pairs, each named, not
  as many names.

Genuine lines pass these checks: `TestReservedManifestConstruction`
checks every scenario's first line on the debug family, and the
development runs (`construction-checks.json`) report no measurement
problem on any of their 664 lines (the 200-seed pass's only coverage
problems are the scenarios it does not run).

## 3. The protocol (M3)

`reserved-validation-protocol.md` opens with the **CERTIFICATION SCOPE**:
certification only for trusted in-process generation and adjudication;
externally editable records and imported measurement summaries cannot
certify; the accepted study-swap limitation, stated as undetected and
accepted only in that scope; authenticated order-to-study provenance
required for any future external-record evaluation; no clinical rule
changed for it. §2, §4, §6 and §8 describe the executable behaviour
above. The 5C.4l documents are marked superseded or corrected, and the
clinical-policy alternatives for the swap are recorded as deferred, not
pursued.

## 4. Errors (M4)

* **Settings:** `DT_MANIFEST_SEEDS` must be a positive integer;
  `DT_MANIFEST_SCENARIOS` known and unrepeated IDs; `DT_MANIFEST_FAMILY`
  an unsigned integer. Anything else is an error, in the execution and
  in the reader, never a default (`manInvocation`).
* **Input:** an unreadable file, a truncated line, malformed JSON or an
  unknown field is an error before anything is written.
* **Output:** the adjudication is serialized with its error checked: an
  unserializable result (a NaN) is an error and writes no file; T5's
  statistics are no longer computed from no groups, so they cannot be
  NaN.
* **Exit:** incomplete coverage, or a missing or malformed measurement,
  fails the execution and the reader after their files are written; a
  certified NO GO on complete coverage is a valid outcome.
* **Never confirmatory:** a partial, reduced, filtered, development or
  file-read invocation (§1).

`TestReaderErrors` covers each, including an empty file and a missing
near-margin scenario, whose adjudications parse back.

## 5. The simulator validation helper (M6)

`recordBaselineWith` recorded its records with an empty scenario
fingerprint; the candidate's integrity check refuses them, so the
statement that every seed was valid was not established by them. It now
records the evaluation scenario's own fingerprint (computed from the
overlaid scenario, equal to the policy's) and refuses to return any
record that fails integrity (`TestValidationHelperIntegrity`).

Regenerated so, the simulator access evidence (`sim-access-5c4n.json`)
and the follow-up's financial observation (`followup-finance.json`) are
**byte-identical** to 5C.4l's: the numbers never depended on the
fingerprint, and every record behind them now passes integrity.

## 6. Corrections

* **Stable groups (audit 5C.4m).** "0 of 28 stable groups" counted the
  first pass's four T1 groups again as the second pass's group 0 and
  added the four stress groups. There were **20 distinct T1 groups**,
  none rejected, sharing their demand across scenarios. This shows no
  regression on the development family and does not establish T1.
* **Near-margin results changed with the shared history.** 5C.4l's
  fidelity-model change moved the measurements: at +2.1 points,
  0.35 → 0.30 (40 seeds) and 0.31 → 0.33 (200 seeds); at +2 points,
  0.56 → 0.65 (200 seeds); predictions 0.297 → 0.300, 0.303 → 0.344 and
  0.508 → 0.549. No threshold or tolerance changed.
* **"Every seed is valid"** (5C.4l, simulator records): corrected (§5).
* **"No bearing on the confirmatory run"** (5C.4l, the swap): corrected;
  the 5C.4l adjudication's file handoff was exactly such a bearing (§1).

Each is marked in place in the 5C.4l documents and the 5C.4k response.

## 7. Deferred: M5

The fidelity model's shared history is not invariant to a candidate
with more imaging capacity than its baseline before the takeover: the
slot grid takes the larger capacity, so inherited bookings can differ.
**No manifest scenario has such a candidate**, so no scenario is
invalidated; it is documented in protocol §7 and must be repaired before
the model serves such a practice. No scope was added for it.

## 8. The release-gate fixes (audit 5C.4o, 0.2.0-dev.25)

The release-gate audit (`docs/audits/slice-5c4o-release-gate-audit.md`)
found M1, M3 and M6 closed and two certification-critical gaps in M2 and
M4. Under the owner's stopping rule only those were fixed; nothing in
the trust boundary, the clinical policy, the manifest or the seed
protocol changed.

* **O1, measurements present but partial or contradictory.** §2's checks
  were of shape, not of agreement: a line could keep one urgent seed
  difference of forty, state a difference its counts contradict, state
  a 90-point increase under a PASS, inject more orders than a pair
  served, or carry a pair construction error the line did not, and pass.
  Each line's Gate B is now reconciled with its own measurements by the
  candidate's rule (`gateBReconciles`): a PASS or FAIL is decided with
  every pair matched, urgent's with a difference on every pair; each
  difference is its counts', each safeguard flag and the deteriorated
  seeds follow from them, the increase is their mean and the status is
  the mean's against the margin, exactly. A near-margin replicate
  carries a difference for every pair, and no pair injects more than it
  served or left unimaged. Every relation held on all 664 genuine lines
  of the development runs before it was adopted, and they still pass.
  Soon and routine keep no per-seed differences in a line, so only their
  matched pairs and status are reconciled.
* **O2, data after a line's value.** The reader decoded one value a line
  and ignored the rest; it now refuses anything but whitespace after it.
* **O3, documentation.** A complete, consistent fabricated file exits
  normally and is `not certified`: the certification decides, not the
  exit (protocol §8). The T5a survivor is equivalent only for the pinned
  manifest's validated inputs. The lock recipe now names the two test
  assertions of the reserved family's refusal that the lock inverts.
  Each is corrected in place.

The test fixtures that stood in for runs (`fakeRun`, `fabricatedRun`)
carried one urgent seed difference a line for speed; they now carry the
rule's own computation (`gateBUrgent`, `failUrgent`, `flagUrgent`), so
the criteria tests exercise consistent lines.

**Deterministic output (0.2.0-dev.26).** At the owner's direction before
the lock, T5a's group predictions and group means are taken in group
order, not Go's map iteration order, so that an adjudication's bytes do
not vary from run to run. The formula is unchanged; only the order of a
floating-point sum is fixed (`TestAdjudicationDeterministic`).

## 9. What this slice does not claim

* **Not audited.** The focused 5C.4o release-gate audit comes next.
* **No lock, reserved draw, freeze or activation.** Nothing ran on
  PCG(20261201, ·); no reference or holdout seed or unseal record was
  used.
* **The study swap is not closed.** It is accepted only for trusted
  in-process evaluation.
* **Not a guarantee of the reserved outcome.** The candidate may fail
  the locked criteria; that is an accepted, documented NO GO.
