# V0.13i limited human presentation review

HUMAN REVIEW PENDING.

* [REVIEWER_GUIDE.md](REVIEWER_GUIDE.md) — scope, exact material, reading order, questionnaire.
* [REVIEW_RECORD.md](REVIEW_RECORD.md) — blank record for the reviewer.
* [MANIFEST.json](MANIFEST.json), [MANIFEST.sha256](MANIFEST.sha256) — the frozen 93 transcripts and 7 active static assets (path and SHA-256).

Verify: `python3 -I -B analysis/alternatives/v13i_human_review.py --check`
(or `sha256sum -c docs/v13i/human-review/MANIFEST.sha256` from the repository root).

The material is the unchanged model-visible content of the V0.13h package.
V0.13i changes only run recording and verification; if any listed byte ever
changes, the affected transcripts require human re-review. The human review
does not cover, and cannot substitute for, the technical review of the run
lifecycle. Protocol lock and live inference: NO GO.
