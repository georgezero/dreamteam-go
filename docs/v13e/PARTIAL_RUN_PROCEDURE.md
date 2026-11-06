# Partial run procedure

A stopped or interrupted run remains PARTIAL. Preserve the entire run
and aborted-draw artifacts in its dedicated directory. Never resume,
overwrite, extend, or replace its draws. Report scheduled and completed
coverage and the exact stop cause, calls, elapsed time and failure streak.

The existing offline analysis invocation is:

```
python3 -I analysis/alternatives/v13d.py analyze --binary PATH_TO_BINARY --protocol PATH_TO_PROTOCOL --contexts docs/benchmark/v13d-dev/contexts --replays PATH_TO_REPLAYS --dir PATH_TO_PARTIAL_RUN --out PATH_TO_ANALYSIS
```

Model-free replay must reconstruct every request and engine answer,
verify response hashes and metadata, and reproduce any halt from its
evidence. Only observed cells may be reported. No complete-run arm
comparisons, pair estimates, replacement draws or pooled conclusions are
permitted from partial coverage. Preserve execution and partial application
separately from selection. Any later experiment requires a new authorization
and a new directory; it cannot repair the missing coverage of this run.
