# Independent evidence review: reviewer handoff

**INDEPENDENT REVIEW: PENDING.** This note is for a person who has agreed
to review the frozen 98-item decision-evidence package. It changes
nothing in the package. The full instructions are
[benchmark/v09a-dev/review/INSTRUCTIONS.md](benchmark/v09a-dev/review/INSTRUCTIONS.md),
and the label definitions are
[benchmark/v08b-dev/review/PROTOCOL.md](benchmark/v08b-dev/review/PROTOCOL.md).

## Before you start: are you independent?

You may review only if all three hold:
* you did not write, calibrate or tune the decision-evidence rules;
* you did not choose the sample;
* you did not see the developer's review notes.

No script, model or LLM may produce a label.

## 1. Open the offline page

Open `docs/benchmark/v09a-dev/review/review.html` in a current web browser
(double-click it, or File → Open). It runs from the file system. It uses
no network, needs no installation and sends nothing anywhere.

## 2. Inspect the evidence

Select each of the 98 items in the list on the left. Each shows:
* the recording, decision point and model;
* the agent's stated reason, with the span the rule read highlighted;
* the observation the agent had: staff behind employee IDs, and the
  prompt excerpt with the observed value;
* the proposed actions, their validation and their execution;
* what the rule checked, its output and its basis;
* the evidence's completeness.

Judge the item from this evidence only. Do not use outcomes.

## 3. Label each finding

Choose exactly one label per item, as PROTOCOL.md defines them:
* `supported`;
* `unsupported`, with what the output should have been;
* `ambiguous`;
* `insufficient_evidence`.

Write a short explanation for any label other than `supported`.

The page shows no suggestion and no default label. Your progress is kept
in the browser. Save `form.csv` now and then; "Resume from a saved
form.csv" loads it again.

## 4. Export the completed form

At the bottom of the page:
1. Make the declaration: your name or role, and the three independence
   statements. Tick only what is true.
2. Save both files with the page's buttons: `form.csv` and
   `reviewer.json`.
3. Keep them together in one directory.

## 5. Submit it for analysis

Send that directory to the project owner. The owner runs:

    python3 -I -B analysis/evidence/review_metrics.py \
        --package docs/benchmark/v08b-dev/review \
        --form REVIEWER/form.csv --out review-metrics.json

The script checks your form against the frozen package and your
declaration. Only then does it report:
* agreement with each rule's output;
* precision per rule over the reviewed supported findings.

It never reports recall: the sample cannot show what the rules missed.
Until a completed independent form arrives, the status stays PENDING and
no agreement, precision or recall figure exists.
