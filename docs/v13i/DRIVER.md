# Driver — dreamteam-v13i-driver/5

DRAFT, NOT LOCKED. The live run is not implemented: `run` refuses even with a
nominal lock and authorization. No command contacts a model.

The successor driver keeps every V0.13h context, draw, replay, schedule,
classification and execution function unchanged (it imports them from
`v13h.py`). It replaces only the enclosing run recorder with the lifecycle
recorder and adds the structured health probe.

The successor protocol `v13i-alt-1` (`dreamteam.alternatives-experiment/v5`)
is V0.13h's protocol with only `schema`, `id`, `driver` and a new `lifecycle`
binding `{id, file, sha256}` changed; every other field, the nine historical
contexts, six context pairs, 21 contexts, two arms, 262 scheduled draws,
802-call ceiling and four-hour request-start guard are equal to the
predecessor. The schedule's 262 entries are identical; only its protocol
identity and hash change.

    python3 -I -B analysis/alternatives/v13i.py protocol --contexts docs/benchmark/v13g-dev/contexts --out FILE
    python3 -I -B analysis/alternatives/v13i.py schedule --contexts docs/benchmark/v13g-dev/contexts --protocol FILE --out FILE
    python3 -I -B analysis/alternatives/v13i.py check --binary ./dt --contexts docs/benchmark/v13g-dev/contexts --protocol FILE --replays .
