# Human presentation review — reviewer instructions

**Status: HUMAN REVIEW PENDING.** No reviewer is assigned.

1. Who may review: a person independent of the design of the simulator, the
   presentation and the experiment, who declares that independence in the
   record. An automated or technical review does not count.
2. Before starting, at the commit you review, run the three checks in
   [README.md](README.md). If any fails, stop: the material changed and must
   be re-frozen before review.
3. Read the frozen guide `docs/v13i/human-review/REVIEWER_GUIDE.md` (exact
   material, suggested reading order, what is out of scope). Every scripted
   reply is labelled SCRIPTED-MODEL OUTPUT; no model was sampled.
4. Answer all 12 items of the [questionnaire](QUESTIONNAIRE.md) with written
   findings, in a copy of `docs/v13i/human-review/REVIEW_RECORD.md`: your
   name, identity/affiliation, independence declaration, date, the exact
   repository commit, the `MANIFEST.json` SHA-256
   (`31bec875723d2d59643efb47979538eb8cd77ed07583228e9f78ce400b9b8faf`), the
   `v13i_human_review.py --check` result, per-item findings and a verdict.
5. Return the completed record separately. It is committed verbatim as a new
   file; the frozen package is not edited. Until then the status remains
   HUMAN REVIEW PENDING.
6. Scope: presentation neutrality and clarity only. Not the recorder,
   verifier, lifecycle, protocol, schedule, estimand or analysis plan; not an
   authorization of protocol lock or live inference.
