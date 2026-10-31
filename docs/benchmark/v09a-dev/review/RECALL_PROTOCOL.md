# A recall protocol for the decision-evidence rules (design, not carried out)

**Status: designed in V0.9a, not carried out, and not authorized to run.**
No recall figure exists for any evidence rule. This design is separate
from the frozen review sample, which cannot estimate recall.

## Why the frozen sample cannot estimate recall

Recall asks: of the statements that really are mismatches, how many did
the rules find? Its denominator is every eligible statement that a
careful reader would call a mismatch, **including those the rules never
flagged**.

The frozen sample (`v08b-review-1`) was drawn from the rules' own
outputs: supported findings, unresolved and insufficient outcomes,
consistent checks, and 8 negative controls. A statement the rules never
read at all is in none of those cells. So a mismatch the rules missed has
no chance of being in the sample. Labelling all 98 items gives agreement
and a share of confirmed findings. It gives no recall.

## The design

**1. The population.** Every reason in a fixed corpus, split into
sentences by a declared splitter, each sentence a unit. The unit is the
sentence, not a rule output. Proposed corpus: the V0.8c llm-agent 3
records (270 attempts, every reply's reason and actions recorded),
because only they have every attempt.

**2. Eligibility, decided by people.** For each unit an annotator first
marks whether it contains an explicit, checkable claim of each kind:
* a stated action;
* a parameter (amount, target, date, duration, direction);
* an observed fact.

They then mark the claim's value. They do this without seeing any rule
output.

**3. The reference label.** For each eligible claim, the annotator
compares it with the reply's actions and the observation the agent was
shown, and labels it `match`, `mismatch` or `cannot_tell`.

**4. The sample.**
* A simple random sample of units, without replacement.
* A seed fixed and committed before drawing.
* A size set in advance, from the precision wanted on recall. Mismatches
  are rare: in V0.8a, 187 of 7,265 checked statements (2.6%) were
  supported findings, and most of those were repeats.
  So a useful estimate needs either a large sample, or strata by claim
  kind with known inclusion probabilities. The design is to be finished
  with the owner before any drawing.

**5. Independence and agreement.**
* Two annotators who meet the review protocol's independence conditions
  label every unit separately.
* Disagreements are resolved by a third, or reported unresolved.
* Agreement (Cohen's κ per claim kind) is reported before any recall.

**6. Only then, the comparison with the rules.**
* Recall: rule mismatch findings that match a reference mismatch, over
  all reference mismatches.
* Precision: on the same units, over the rule's findings there.
* Both with confidence intervals (Wilson), and weighted by inclusion
  probability if stratified.
* Abstentions (`unresolved`, `insufficient_evidence`) are counted apart,
  as neither found nor missed.

**7. What stays fixed.**
* The rules under test, frozen beforehand by version and analyzer SHA-256.
* The corpus fingerprint.
* No rule may be changed after the reference labels are seen, without a
  new version and a new sample.

## What it would need from the owner

* Authorization.
* Two or three independent annotators.
* A decision on the corpus and the sample size.

None of this is implemented in V0.9a beyond this design.
