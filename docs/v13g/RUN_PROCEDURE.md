# Future run procedure

Protocol lock: NO GO. Live inference: NO GO. Independent re-review: PENDING.
No live run command was invoked. The successor driver's run command has no
live implementation and refuses execution even if supplied a nominal lock.

Before any future run, complete independent technical re-review on the final
snapshot, independent human review of the complete successor package, owner
acceptance and all required engineering gates. Any live implementation and
live authorization are separate work. Neither code completion nor a future
protocol lock authorizes inference.

Offline analysis uses:

```
python3 -I -B analysis/alternatives/v13g.py analyze --binary PATH_TO_BINARY --protocol docs/benchmark/v13g-dev/protocol-draft.json --contexts docs/benchmark/v13g-dev/contexts --replays PATH_TO_REPLAYS --dir PATH_TO_SCRIPTED_RUN --out PATH_TO_ANALYSIS
```

Use dedicated directories. Never resume or replace draws in a partial run.
Preserve every request, response, abort and stop record. The four-hour
mechanism is a request-start guard; it does not bound blocked operations.
Keep the existing 262-draw design and 802-call ceiling unchanged.
