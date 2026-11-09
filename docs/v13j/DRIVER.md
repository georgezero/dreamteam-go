# Driver — dreamteam-v13j-driver/6

DRAFT, NOT LOCKED. The live run is not implemented: `run` refuses even with a
nominal lock and authorization. No command contacts a model.

The successor driver keeps every V0.13h context, draw, replay, schedule,
classification and execution function unchanged (it imports them from
`v13h.py`; `v13h.py` and V0.12b's `stage_b.py` are byte-identical to the
starting commit). It replaces the enclosing run recorder with the
safe-point, call-journal and completion-seal recorder, adds the
exception-family health probe, and spawns engine sessions with V0.13h's
exact arguments in their own session (process group), so that a terminal
SIGINT cannot terminate an engine mid-draw.

The successor protocol `v13j-alt-1` (`dreamteam.alternatives-experiment/v6`)
is V0.13h's protocol with only `schema`, `id`, `driver` and the `lifecycle`
binding `{id, file, sha256}` changed; every other field, the nine historical
contexts, six context pairs, 21 contexts, two arms, 262 scheduled draws,
802-call ceiling and four-hour request-start guard are equal to the
predecessor. The schedule's 262 entries are identical to V0.13h's and
V0.13i's; only its protocol identity and hash change.

    python3 -I -B analysis/alternatives/v13j.py protocol --contexts docs/benchmark/v13g-dev/contexts --out FILE
    python3 -I -B analysis/alternatives/v13j.py schedule --contexts docs/benchmark/v13g-dev/contexts --protocol FILE --out FILE
    python3 -I -B analysis/alternatives/v13j.py check --binary ./dt --contexts docs/benchmark/v13g-dev/contexts --protocol FILE --replays .
