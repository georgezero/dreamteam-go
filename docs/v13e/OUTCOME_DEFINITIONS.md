# Outcome and observable-comparison definitions

The approved primary scale is determined by actual submitted batches:

| Category | Definition |
|---|---|
| original_retained | Original operations and staffing batches unchanged |
| listed_alternative | Original operations with another accepted candidate's staffing batch |
| no_staffing_action | Original operations only |
| other_batch | Any other submitted batches, including changed or removed operations |
| abandoned | Nothing submitted by choice |
| invalid | Nothing submitted because selection or final validation failed |
| no_decision | Nothing submitted because no usable reply exists |

The last three take precedence for their respective failure/choice states;
otherwise operations equality precedes staffing equality, then empty staffing,
then candidate equality. The implementation returns exactly one category.
not_due is a separate eligibility stratum, not an eighth decision category.
The exact batch hash is retained. Selecting an alternative, validation,
submission, execution and partial execution remain distinct process records.
A changed action does not establish an improved decision.

F3 is a secondary process measure only. Only detail actually delivered in an
inspect answer counts. Cite keys and exact displayed values for at least two
alternatives with a difference and at least one alternative not chosen.
At most three items, fewer than all shown items, and no copied whole table.
A whole-number JSON value may equal its displayed whole-number text; strings
must match exactly. Paraphrase and undelivered detail do not count. No internal
reasoning is inferred, and reason text is never scored.

Reused coverage: TestReviewV13dF7FinalAction, TestDecideOperations,
TestReviewV13dF3UnseenCitation, TestDecideCitations, TestTurnSymmetry,
analysis/alternatives/test_v13d.py and the scripted fidelity suite.
Added engineering boundaries: TestV13eFinalActionBoundaries (all seven
categories in both arms with mixed batches, changed operations and no
execution) and TestV13eCitationBoundaries (positive, wrong value, paraphrase,
undelivered detail, copied short table).
