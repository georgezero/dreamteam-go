# Session fixtures

Session files written by earlier builds, kept byte-for-byte as those builds
wrote them, to test that this build replays them authoritatively.

| File | Written by | Run |
|---|---|---|
| `mini-seed21-slice2.session.json` | 0.2.0-dev.2 (V0.2 Slice 2, commit 2da900d) | mini, seed 21, 9 days, with `mini-seed21-slice2.actions.json` |
| `mini-seed22-slice3.session.json` | 0.2.0-dev.3 (V0.2 Slice 3, commit 5fdfb95) | mini, seed 22, 9 days, with `mini-seed22-slice3.actions.json` |
| `mini-seed23-dev4.session.json` | 0.2.0-dev.4 (dynamic staffing benchmark, commit af475fd) | mini, seed 23, 9 days, with `mini-seed23-dev4.actions.json` |

Each was made with its build's binary, from a clean worktree of that commit:

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

The Slice 3 file was made the same way from 5fdfb95, with
`-seed 22 -actions testdata/sessions/mini-seed22-slice3.actions.json`. Its
batches are a hire with a roster version for Riley, a termination, a V0.1
extension, and a batch that withdraws Riley's version and sets another a
week later. Every action was accepted. Slice 3's action set is this
build's, so it replays in batch mode `v0.2_slice3` with every digest
verified (`earlier_build_exact`) and continues with the file embedded as
the new record's prefix (`TestSlice3RecordReplaysAndContinues`). Before
0.2.0-dev.3 was registered, this build refused it as an engine mismatch.

The 0.2.0-dev.4 file was made the same way from af475fd, with `-seed 23
-actions testdata/sessions/mini-seed23-dev4.actions.json`. It was written
before `set_reading_allocation` existed, and its batches are:

* a `set_reading_allocation` alone, refused as an unknown action;
* the same type beside a hire, refused as `mixed_batch` with dev.4's
  message naming hire, terminate and set_roster;
* an accepted hire with a roster version for Riley;
* a V0.1 extension.

Under this build's action set the first would be a no-op and the second
would be accepted. The file therefore replays with Slice 3's action set in
batch mode `v0.2_slice3`, with every digest verified
(`earlier_build_exact`). It continues with a real allocation change, with
the file embedded as the new record's prefix
(`TestDev4RecordReplaysAndContinues`). Before 0.2.0-dev.4 was registered,
this build refused it as an engine mismatch. The benchmark's session files
in `docs/calibration/dynamic-staffing/sessions/` are also dev.4 records,
and `dreamteam verify` replays them the same way.
