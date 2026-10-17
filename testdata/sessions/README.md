# Session fixtures

Session files written by earlier builds, kept byte-for-byte as those builds
wrote them, to test that this build replays them authoritatively.

| File | Written by | Run |
|---|---|---|
| `mini-seed21-slice2.session.json` | 0.2.0-dev.2 (V0.2 Slice 2, commit 2da900d) | mini, seed 21, 9 days, with `mini-seed21-slice2.actions.json` |
| `mini-seed22-slice3.session.json` | 0.2.0-dev.3 (V0.2 Slice 3, commit 5fdfb95) | mini, seed 22, 9 days, with `mini-seed22-slice3.actions.json` |
| `mini-seed23-dev4.session.json` | 0.2.0-dev.4 (dynamic staffing benchmark, commit af475fd) | mini, seed 23, 9 days, with `mini-seed23-dev4.actions.json` |
| `mini-seed24-dev5.session.json` | 0.2.0-dev.5 (V0.2 Slice 4, commit 13987e6) | mini, seed 24, 30 days, with `mini-seed24-dev5.actions.json` |
| `mini-seed25-dev5-reading.session.json` | 0.2.0-dev.5 (V0.2 Slice 4, commit 13987e6) | mini, seed 25, 20 days, with `mini-seed25-dev5-reading.actions.json` |
| `mini-seed26-dev6.session.json` | 0.2.0-dev.6 (pre-Slice 5 remediation, commit 3c71669) | mini, seed 26, 25 days, with `mini-seed26-dev6.actions.json` |
| `mini-seed27-dev7.session.json` | 0.2.0-dev.7 (V0.2 Slice 5A, commit 855acfe) | mini, seed 27, 25 days, with `mini-seed27-dev7.actions.json` |

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

The two 0.2.0-dev.5 files were made the same way from 13987e6, with
`-seed 24 -days 30` and `-seed 25 -days 20`. They are there because
0.2.0-dev.6 changed outcomes, not only an encoding (V0.2_DESIGN §5.11):

* `mini-seed24-dev5` sets Dr. Avery to 100% and Dr. Blake to 95% from
  Oct 27, Dr. Blake to 30% from Sat Oct 31 and back on Mon Nov 2, Dr.
  Avery to 30% from Sat Nov 7 and back to 100% on Mon Nov 9, then
  withdraws Dr. Blake's Oct 31 change. dev.5 accepted every action;
  dev.6 refuses the short versions (`commitment_too_short`) and would
  read differently at 100% and 95%.
* `mini-seed25-dev5-reading` has only the 100%/95% batch, which both
  builds accept; its log and run digests reproduce only under dev.5's
  reading rules.

Both replay in batch mode `v0.2_slice4` (this build's action set under
dev.5's rules) with every digest verified (`earlier_build_exact`), and
the first continues under this build's rules with the file embedded as
the new record's prefix (`internal/session/dev5_test.go`). Before
0.2.0-dev.5 was registered, this build refused both as an engine
mismatch; registered without dev.5's rules, the first is refused at its
second batch.

The 0.2.0-dev.6 file was made the same way from 3c71669, with `-seed 26
-days 25`. Its batches are:

* Dr. Avery at 100% and Dr. Blake at 95% from Oct 27, accepted;
* Dr. Blake at 30% from Sun Nov 1 and back on Mon Nov 2, refused as
  `commitment_too_short`, with the rest of the batch `batch_rejected`;
* Dr. Avery back to 60% from Nov 10, accepted.

It is there because 0.2.0-dev.7 changed no outcome but must replay dev.6
records under dev.6's rules, not dev.5's as every earlier mode does. It
replays in batch mode `v0.2_dev6` (this build's action set and rules) with
every digest verified (`earlier_build_exact`), and continues with the file
embedded as the new record's prefix (`internal/session/dev6_test.go`).
Before 0.2.0-dev.6 was registered, this build refused it as an engine
mismatch. Registered under dev.5's rules, it fails at its second batch,
which dev.5's allocation validation accepts.

The 0.2.0-dev.7 file was made the same way from 855acfe, with `-seed 27
-days 25`. Its batches are:

* a `contract_teleradiology` start alone, refused as `unknown_action`
  (dev.7 had no such action);
* Dr. Avery at 80% beside a `contract_teleradiology` start, refused as
  `mixed_batch` with the Slice 4 message, the allocation `batch_rejected`;
* Dr. Blake at 70% from Oct 30, accepted.

It is there because 0.2.0-dev.8 makes `contract_teleradiology` a real
action. Records of 0.2.0-dev.5 to dev.7 replay with Slice 4's action set,
in which the type is unknown, as they recorded it. This one replays in
batch mode `v0.2_dev7` with every digest verified
(`earlier_build_exact`), and continues with the file embedded as the new
record's prefix (`internal/session/dev7_test.go`). Before 0.2.0-dev.7 was
registered this build refused it as an engine mismatch. Replayed with this
build's action set, it fails at its first batch, where the start is now
refused as `invalid_schema` (it has no plan).
