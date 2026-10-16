# Session fixtures

Session files written by earlier builds, kept byte-for-byte as those builds
wrote them, to test that this build replays them authoritatively.

| File | Written by | Run |
|---|---|---|
| `mini-seed21-slice2.session.json` | 0.2.0-dev.2 (V0.2 Slice 2, commit 2da900d) | mini, seed 21, 9 days, with `mini-seed21-slice2.actions.json` |

It was made with that build's binary:

    git worktree add /tmp/dev2 2da900d && (cd /tmp/dev2 && go build -o dreamteam-dev2 ./cmd/dreamteam)
    dreamteam-dev2 run -scenario mini -seed 21 -days 9 \
        -actions testdata/sessions/mini-seed21-slice2.actions.json \
        -save testdata/sessions/mini-seed21-slice2.session.json

Its batches include a hire, a termination, a V0.1 extension and, written
before `set_roster` existed, two actions of type `set_roster`. One is
refused as an unknown action. The other, in a batch with a termination, is
refused as `mixed_batch` with Slice 2's message naming only hire and
terminate. This build replays the file with Slice 2's action set and every
digest verified (`dreamteam verify` reports `earlier_build_exact`), and
continues it with the file embedded as the new record's prefix
(`internal/session/roster_test.go`).
