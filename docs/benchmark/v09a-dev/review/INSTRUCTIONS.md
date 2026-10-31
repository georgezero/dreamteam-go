# Independent review: instructions

**Status: PENDING** ([REVIEW_STATUS.md](REVIEW_STATUS.md)). This
directory adds a way to carry out the review. It does not change the
review: the sample, its items, its form and its protocol are the frozen
V0.8b package in [docs/benchmark/v08b-dev/review/](../../v08b-dev/review/),
unchanged.

## Who can review

Only someone who meets the protocol's independence conditions
([PROTOCOL.md](../../v08b-dev/review/PROTOCOL.md), "Independence"):
* they did not write, calibrate or tune the decision-evidence rules;
* they did not choose the sample;
* they did not see the developer's own review notes.

The developer cannot review. No script, model or LLM may produce a label.

## What you need

* This repository, or only two files from it: [review.html](review.html)
  and the protocol.
* A current web browser. The page is self-contained: it uses no network,
  needs no installation and runs from the file system.

## Steps

1. **Read the protocol** first: what each label means, and the rules for
   the reviewer (judge the item, not the model; do not use outcomes; read
   "explicit statement" as the rules define it).
2. **Open `review.html`** in the browser (double-click it, or File → Open).
3. **For each of the 98 items** (the list on the left):
   * read its evidence:
     * the recording, decision point and model;
     * the sources (the recording line, the replay transcript, the apply
       results, the prompt's SHA-256), where the item lists them;
     * the reason, with the span the rule read highlighted;
     * the observation: the staff behind employee IDs, and the prompt
       excerpt where the observed value appears;
     * the proposed actions, their validation and execution;
     * what the rule checked, and its output and basis;
     * the evidence completeness;
   * choose one label: `supported`, `unsupported`, `ambiguous` or
     `insufficient_evidence`;
   * write a brief explanation. It is required for any label other than
     `supported`, and welcome for every item;
   * for `unsupported`, say what the output should have been
     (correction).
4. **Make the declaration** at the bottom: your name or role, and the
   three independence statements. Tick only what is true.
5. **Save both files** with the two buttons: `form.csv` and
   `reviewer.json`. Put them in one directory.
6. **Send that directory** to the project owner.

**Progress** is kept in the browser while you work. To be safe across
browsers or machines, save `form.csv` now and then, and load it again with
"Resume from a saved form.csv". The page accepts only the frozen form's
rows, unchanged.

The page shows no suggestion and no default label. Every item starts
unlabelled.

## What happens to the files

The owner checks and analyses them, with no model:

```
python3 -I -B analysis/evidence/review_metrics.py \
    --package docs/benchmark/v08b-dev/review \
    --form <your directory>/form.csv --out review-metrics.json
```

The script refuses a form, and reports nothing, if:
* any item is unlabelled or carries another label;
* a row's identifier, rule or rule output differs from the frozen form;
* `reviewer.json` is missing, or does not declare independence;
* the declaration names another sample.

It warns about a non-`supported` label without an explanation.

The protocol's own analysis, `review_analysis.py`, also still applies,
unchanged.

**What is reported** from a completed review, and what is not, is set out
in [REVIEW_STATUS.md](REVIEW_STATUS.md). In short:
* agreement with the rules' outputs, cell by cell;
* the share of supported findings confirmed, rule by rule, always with
  its denominator and how the items were sampled;
* never recall, which this sample cannot show
  ([RECALL_PROTOCOL.md](RECALL_PROTOCOL.md)).

Two reviewers may each label the sample independently. Their
disagreements are then reported item by item.
