# Response to the Slice 5C.4k clinical safety and protocol lock audit (maintainer)

This is the maintainer's account of what was done with each finding of
`slice-5c4k-clinical-safety-protocol-lock-audit.md`.

* **The audit is unchanged.** It was committed alone and verbatim
  (d06cdb9) before any remediation, and this document is separate from
  it.
* **The remediation is Slice 5C.4l** (0.2.0-dev.23), under the owner's
  decisions of 2026-10-20:
  * pair inherited orders with the baseline and remove the
    inherited-order exemption (K2);
  * reconcile every order present at the takeover or after it with the
    baseline by origin, not only orders eligible for Gate A or Gate B;
  * match immutable order attributes, never outcomes;
  * never manufacture a counterpart for a candidate-only order: without
    a trusted generation record, INVALID or INSUFFICIENT, never GO.
* **The clinical policy is unchanged.** No threshold, window, minimum,
  limit, the follow-up, the partial-window ceiling or a tolerance
  changed. Only record integrity, the adjudicator, the fidelity model's
  pre-takeover history, reporting and documents changed.
* **The auditor's witnesses** (kept outside the repository) are
  reproduced as regression tests in
  `internal/eval/remediation_5c4l_test.go`.

**Candidate digest (0.2.0-dev.23):**
`83d481b7040b69d78a58de68afc938a588b5b37ba396438509f38c56ca163948`
(pinned in manifest version 3).

## Findings K1–K7

| Finding | Status | Where |
|---|---|---|
| **K1** (High): relink and re-time | **Closed for every attack on immutable attributes; one outcome-only form remains, reported for an owner decision.** Every order of a candidate's record must reconcile with the baseline's by origin, with the same ID, receipt, wanted time, priority, modality and walk-in flag (`orderProvenance: pairedOrigin`). The audit's bypass (the re-timed order) is now INVALID; under the 5C.4j candidate it is reproduced as GO. Also INVALID: a changed origin, priority, modality or wanted time, swapped IDs, a missing order, a duplicate order, a candidate-only order, a re-timed order outside Gate A/B eligibility. Different legitimate outcomes keep their verdicts. **Not closed:** two same-request orders, one wanted by the horizon and one after it, both imaged after the later wanted time, exchanging their studies and resolutions. Nothing immutable changes, so it stays GO (`TestStudySwapLimitation`). It is stated as a limitation with four options (`clinical-provenance.md` §2.3) | `provenance.go`; `compare.go`; `TestOrderProvenance`, `TestOrderProvenanceUnit`, `TestStudySwapLimitation` |
| **K2** (High): inherited orders | **Closed.** The exemption is removed: every served order in a record, inherited or new, has exactly one post-takeover study; order IDs are unique (`orderStudyIntegrity: contractInherited`). Inherited orders are reconciled by origin with the baseline's. Every case of the audit's table is INVALID, and so are a deleted inherited order, a duplicated or relinked study; a pending report stays INSUFFICIENT. The fidelity model gained a shared pre-takeover history so its paired records hold the same inherited orders, as the simulator's do | `checkOrderStudies`; `oc_fidelity_test.go` (`sharedHistory`); `TestInheritedSafety`, `TestContractInheritedUnit`, `TestSharedHistoryReconciles` |
| **K3**: consistent re-timing, both sides edited | **Stated.** The paired record is trusted as the harness produced it; edits made alike to both records are not detectable by a pure evaluator. The reserved run constructs its records in process | protocol §7; `clinical-provenance.md` §2.3 |
| **K4** (Medium): zero-evidence PASS, invocation | **Closed.** No criterion passes with nothing judged, in T1–T8: a criterion with no scenario of its kind, or with a scenario without a complete group (or seed) is INCONCLUSIVE; T8 with no line is INCONCLUSIVE; a scenario declared absent by the invocation is incomplete coverage. The output records each scenario's judged seeds against the manifest's (`invocation`: `confirmatory` only at the manifest's seeds). The harness now defaults to the manifest's seeds. The reserved run and its adjudication are pinned as commands (protocol §8), not run | `reservedCriteria`, `manCoverage`; `TestReservedCriteriaZeroEvidence`, `TestProvenanceCoverage` |
| **K5**: the partial-window statement | **Corrected** in place in the 5C.4j documents, marked as corrections. The leading soon window failed on seeds 901 (3/42), 902 (3/41) and 903 (4/41); seed and evaluation verdicts did not change. Local, seed-level and evaluation-level results are now reported apart | `clinical-provenance.md` §4; `../calibration/clinical-5c4j/clinical-safety.md` |
| **K6**: promised reports, W-ACCESS65, financial observation | **Closed.** T1 reports each stable scenario's partial-ceiling seeds and the stress practices' rates; T7 the partial-ceiling seeds; T2 each scenario's detection and lower bound, the conditional prediction, and the H10 and OH10 rates. W-ACCESS65 now needs Gate B PASS disclosed with both plans' urgent unimaged fractions (at least 0.65) and the claims. The follow-up's financial effect is measured and documented (below) | `reservedCriteria`; `witnessMet`; `clinical-provenance.md` §5 |
| **K7**: statistical limitations | **Carried.** 0 of 28 stable groups shows no regression on the development family, not the T1 target; T5a's first test at its own size is the reserved run | protocol §7 |

## Validation

**Regression witnesses** (`remediation_5c4l_test.go`), each run under
the 5C.4l candidate and, where it shows the change, the 5C.4j candidate:
* **K1:** `TestOrderProvenance`: the audit's relink-and-retime bypass and
  ten further attacks on immutable attributes, INVALID; two
  legitimately different outcomes, not.
* **Provenance, field by field:** `TestOrderProvenanceUnit`: every field
  and every missing, extra, repeated or empty origin, from either side.
* **K2:** `TestInheritedSafety` (the audit's table and five more cases)
  and `TestContractInheritedUnit`.
* **The limitation:** `TestStudySwapLimitation` pins the swap as GO, so
  that any change to it is noticed.
* **The fidelity model:** `TestSharedHistoryReconciles`: seven
  scenarios, and a practice with half the capacity before the takeover,
  reconcile; independent histories do not.
* **K4:** `TestReservedCriteriaZeroEvidence`, `TestProvenanceCoverage`;
  `TestReservedCriteriaRules` with the new witness rules (W-ID by
  provenance on the stated seed, W-ACCESS65's disclosure).
* **Pinning:** `TestManifestPinsCandidate`, `TestPolicyProvenanceValidation`.

**The earlier witnesses:** every 5C.4i and 5C.4j regression passes. The
tests whose expected outcome the owner's decisions changed now expect
INVALID by order provenance under the 5C.4l candidate and keep their
earlier outcome under the 5C.4j candidate:
* `TestGateBMissingIdentity`, `TestGateNoncomparableDemand`,
  `TestWitnessManipulation`: NO GO before, INVALID now;
* `TestOrderStudyInheritedExempt`: the exemption, now 5C.4j's only.

**Mutation checks:** 23 new mutations, one at a time, all caught:
* provenance (9): off; not applied in comparisons; ID, wanted time or
  priority not compared; candidate-only or missing orders accepted; an
  empty origin accepted; its policy validation relaxed;
* the contract (3): the inherited exemption kept, repeated IDs accepted,
  the strict contract not dispatched;
* the adjudicator (7): zero evidence passing; scenarios declared absent
  counted complete; the invocation always confirmatory; T8 passing with
  no line; W-ID by Gate B alone; W-ACCESS65 without the disclosed loss;
  unexpected provenance failures counted complete;
* the fidelity model (4): the shared history's booking stream and its
  capacity ignored, the manifest practices without it, prefix-stable
  draws off.

The 5C.4j sets were re-run: 21 of 21 applicable caught. The three that
target since-changed code are covered by the new ones (the served-order
study rule, the identity witnesses' rule, prefix stability).

**Development runs** (`clinical-5c4l/construction-checks.json`), on the
development family only:
* every judged criterion passes;
* provenance fails only on the identity witnesses' first seeds;
* 0 of 28 stable groups are rejected, and no stable seed fails the
  partial-window ceiling;
* T5b: 0.33 at +2.1 points, 0 at +3;
* the 200-seed pass makes every criterion with an unrun scenario
  INCONCLUSIVE (K4).

**Simulator, development seeds 901–910, no action:**
* records under the candidate are identical to 5C.4j's;
* `TestDemandExogenousAcrossPlans` checks every order, inherited
  included, with its ID and request, under three plans;
* the follow-up changes no financial measure or decision
  (`followup-finance.json`).

**Compatibility:**
* v0.2-eval-4 is byte-identical (`TestV4Neutrality`);
* the 5C.4d, e, f, h and j candidates evaluate under their own rules,
  from their exact bytes;
* 0.2.0-dev.22 is registered (`mini-seed52-dev22`); replay through
  dev.22 is exact;
* no operational simulator code changed.

**Suite:** the full uncached suite passes in every package (one package
at a time, memory-capped: eval 1,905 s, sim 544 s, session 182 s; 2,696 s
in all, peak 662 MB). `go vet` and `gofmt` are clean.

## Owner decision requested

**The study swap** (`clinical-provenance.md` §2.3). It does not bear on
the reserved run, whose records are built in process. It bears on any
evaluation of records that could be edited. The options are:
1. accept it as a stated limitation (the protocol states it now);
2. attribute post-horizon studies conservatively, per priority and
   modality (a clinical-policy change);
3. widen the absolute cohort to every study acquired by the end of
   observation (a clinical-policy change);
4. authenticate the link outside the record (an architectural change).

## What this response does not claim

* **The study swap is not closed.** It is an owner decision (above).
* **No independent review yet.** The focused 5C.4m review must judge
  the provenance contract, inherited-order safety, zero-evidence
  adjudication and reporting.
* **No lock, reserved draw, freeze or activation.** Nothing was run on
  PCG(20261201, ·); no reference or holdout seed or unseal record was
  used. The reserved command is documented, not run.
