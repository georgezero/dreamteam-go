# V0.13n human presentation review — reviewer packet

**Status: HUMAN REVIEW PENDING.** No reviewer has been assigned, and none is
named here. Only the reviewer may enter an identity, finding or verdict. No
automated or technical review substitutes for this review.

## What the reviewer receives

A self-contained packet built from commit
`9d4626bc51cecb3dd4d21f2efa7a246bbbcf78ca`, with repository-relative paths
preserved, containing only:

| Material | Path |
|---|---|
| Frozen guide, blank record, manifest and its checksum file | `docs/v13i/human-review/` (frozen at `ee987a1`) |
| 93 model-visible transcripts | `docs/v13h/review-package/transcripts/001.json` … `093.json` |
| 7 static assets | `docs/v13h/review-package/sources/ACTIVE/` |
| Package README, failure-feedback disclosure, navigation index | `docs/v13h/review-package/` (`README.md`, `FAILURE_FEEDBACK_DISCLOSURE.md`, `RECORD_ONLY_INDEX.json`, `SHA256SUMS`) |
| The 12-item questionnaire and reviewer instructions | `docs/v13m/human-review/QUESTIONNAIRE.md`, `REVIEWER_INSTRUCTIONS.md` |
| This page | `docs/v13n/human-review/REVIEWER_PACKET.md` |

The packet deliberately **excludes** all technical audits and their
conclusions (`docs/audits/`, every acceptance or independent technical
review, and the implementation). The reviewer forms an initial judgment from
the model-visible material alone. Record-only lifecycle examples are not
model-visible and are not included.

Integrity check, from the packet root:

```sh
sha256sum -c docs/v13i/human-review/MANIFEST.sha256
```

`MANIFEST.json` SHA-256: `31bec875723d2d59643efb47979538eb8cd77ed07583228e9f78ce400b9b8faf`.
In the repository, `python3 -I -B analysis/alternatives/v13i_human_review.py --check`
additionally confirms the 93 transcript and 7 asset hashes.

## Reviewer profile (owner guidance)

One radiologist or radiology operations leader who:

* has experience with staffing, scheduling, capacity or operational
  management;
* has not developed DreamTeam or authored the presentation;
* can assess the actual control and treatment messages without inspecting
  code.

## What the completed record must contain

Use a copy of `docs/v13i/human-review/REVIEW_RECORD.md`. In addition to its
fields, state explicitly:

1. Reviewer identity (name and affiliation).
2. Relevant expertise (staffing, scheduling, capacity or operations
   experience).
3. Independence declaration (no role in developing DreamTeam, the simulator,
   the presentation or the experiment).
4. Responses to all 12 questionnaire items.
5. Specific findings, each naming the affected transcript(s) or asset(s).
6. Overall recommendation: ACCEPT / ACCEPT WITH CHANGES / DO NOT ACCEPT.

The completed record is committed verbatim as a new file. The frozen package
is never edited.

## Disposition of findings

For each finding the project records: the exact concern, the affected
material, a severity, the smallest fix, and whether transcript or asset
hashes would change. A material presentation change requires owner approval
before any change, a new freeze with new hashes, and human re-review. Protocol
lock remains NO GO until this review passes and the bounded technical check
is accepted.
