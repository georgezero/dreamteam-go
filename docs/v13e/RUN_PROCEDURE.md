# Future run procedure

LOCK CANDIDATE - NOT LOCKED - awaiting human review

HUMAN REVIEW PENDING. Live inference NO GO. Nothing in this procedure
authorizes a model call. No protocol-lock commit exists.

Before any future run: obtain the completed independent human questionnaire,
owner acceptance, all final engineering gates, a separately prepared
protocol with the approved status, and separate explicit live authorization.
Verify committed file identities with python3 -I analysis/alternatives/lock_check.py.
Use a new dedicated run directory. Preserve every call and abort record.

The actual driver's currently supported run invocation would be:

```
python3 -I analysis/alternatives/v13d.py run --protocol docs/v13e/protocol-approved.json --authorization 'OWNER_AUTHORIZATION_TEXT'
```

The path and authorization above are future placeholders; that protocol
file does not exist. These are the run parser's only arguments. Do not add
binary, contexts, replays or output flags to run; it does not accept them.
Even after both status and authorization pass, the current driver refuses:
its live run implementation does not exist. Future implementation requires
a separately authorized milestone. This command was not executed here.
Refusal is tested through test_run_refuses_a_draft with no network client.

Preserve nine historical contexts, six sensitivity context pairs, control
and treatment, 262 scheduled draws, 802 maximum calls and 14,400 seconds.
Use the exact frozen model identifiers and settings files pinned in the
candidate. Calls must be sequential, without silent retries. No replacement
draws or automatic resume. A stop leaves PARTIAL. Record every response,
metadata, engine exchange and stop cause, then replay without any model.
