# Golden references

Canonical results (`dreamteam.result/v1`) of pinned runs. They contain
aggregate outcomes and verification digests only: no individual patient,
study, claim or order records, no hidden state (that is only in the state
digest).

| File | Run |
|---|---|
| `established-seed1-7d.json` | established practice, seed 1, 7 days from the takeover (Tue Oct 13 06:00 → Tue Oct 20 06:00) |
| `established-seed1-90d.json` | established practice, seed 1, 90 days from the takeover |
| `mini-seed1-7d.json` | mini practice, seed 1, 7 days from the takeover |
| `established-seed1-7d-actions.json` | established, seed 1, 7 days, with `established-7d.actions.json` |
| `established-seed1-60d-staffing.json` | **V0.2** (0.2.0-dev.2): established, seed 1, 60 days, with `established-staffing.actions.json` |

`established-7d.actions.json` (input) holds a future-day shift extension,
a walk-in policy change and back, a worklist policy change, a no-op and
three rejected actions (unknown employee, a retroactive extension, too
many minutes). Every reference was produced with `-verify-replay`: the
run's own record was replayed from scratch and matched.

`established-staffing.actions.json` (input, V0.2 Slice 2) terminates Sam and
hires an XR+CT replacement on his roster in one atomic batch (premium lead,
independent the day after his last day: nobody is displaced); terminates
Morgan with a gap before a US hire becomes independent (her US bookings are
displaced: rebooked or lost); hires a front desk clerk and terminates her
before her start; and includes a V0.1 extension in a separate batch at the
same instant, a refused mixed batch, an identical repeat (`unchanged`) and
a termination of someone who has already left (`not_employed`).

## Check

    go test ./cmd/dreamteam -run Golden              # all four (-short skips 90 days)
    dreamteam run -scenario established -seed 1 -days 7 \
        -verify-replay -format json -golden testdata/golden/established-seed1-7d.json \
        -golden-migrations testdata/golden/MIGRATIONS.json -allow-engine-mismatch

## Historical references and the migration manifest

The four V0.1 files are historical compatibility fixtures: they stay
byte-for-byte as their V0.1 engine wrote them. The V0.2 reference is
written by the build named in it and compared byte for byte by that build. V0.2 Slice 1 changed the state
digest's encoding (`dreamteam.state/v2`: the employment registry and
guaranteed-pay counters) without changing any outcome, so exactly three fields
differ in each: `digests.state`, `digests.run` and `replay.runDigest`. Slice 2
changed it again (`dreamteam.state/v3`: hires, terminations, displacement,
growable staff sets, the new accounts; and the run digest names
`dreamteam.session/v2`); the same three fields differ, with new values.
`MIGRATIONS.json` records each one's old and new value per reference, with the
explanation. Under `-golden-migrations` a listed field must hold exactly its
old value in the reference and its new value in the run; every other field is
compared as before. A later state-schema change updates the manifest's new
values and explanation, never the historical files.

Exit 0: identical outcomes. Exit 4: a difference — stderr names the first
differing JSON path with both values, the number of differences and the
digests that differ. Exit 5: not comparable (another schema, scenario
fingerprint, seed or run length; or another engine version, unless
`-allow-engine-mismatch`, which compares outcomes and still reports the
engine difference) or a malformed file.

## Update (explicit only)

Golden files change only when gameplay is meant to change; a state-encoding
change with identical outcomes goes in `MIGRATIONS.json` instead. Regenerate with
the same command and `-write-golden FILE -overwrite`, review the diff, and
say why in the commit message:

    dreamteam run -scenario established -seed 1 -days 7 -verify-replay \
        -format json -write-golden testdata/golden/established-seed1-7d.json -overwrite
    dreamteam run -scenario established -seed 1 -days 90 -verify-replay \
        -format json -write-golden testdata/golden/established-seed1-90d.json -overwrite
    dreamteam run -scenario mini -seed 1 -days 7 -verify-replay \
        -format json -write-golden testdata/golden/mini-seed1-7d.json -overwrite
    dreamteam run -scenario established -seed 1 -days 7 -actions testdata/golden/established-7d.actions.json \
        -verify-replay -format json -write-golden testdata/golden/established-seed1-7d-actions.json -overwrite
    dreamteam run -scenario established -seed 1 -days 60 -actions testdata/golden/established-staffing.actions.json \
        -verify-replay -format json -write-golden testdata/golden/established-seed1-60d-staffing.json -overwrite

A new engine version with unchanged outcomes needs no regeneration: the
test compares with the engine mismatch allowed and reports it. Since V0.2
the four files are kept as historical references (owner decision), not
regenerated to name the current engine.

## Cross-platform procedure

On each platform (Linux amd64, Linux arm64, macOS arm64), from a clean
checkout of the same commit:

    go build -o dt ./cmd/dreamteam
    for args in "-days 7" "-days 90" "-scenario mini -days 7" \
                "-days 7 -actions testdata/golden/established-7d.actions.json"; do
        ./dt run -seed 1 $args -verify-replay -format json > out.json   # then:
        cmp out.json testdata/golden/<matching file>                     # byte-identical?
    done

(byte-identical only with the build that wrote the files; with a later one,
compare with `-golden FILE -golden-migrations testdata/golden/MIGRATIONS.json
-allow-engine-mismatch`, exit 0) or simply `go test ./cmd/dreamteam -run Golden`. Integers only, fixed
field order, sorted maps and 64-bit big-endian digest encodings mean the
results should not depend on the platform, but that is only established
where it has been run:

| Platform | Status |
|---|---|
| Linux arm64, Raspberry Pi 4 (Cortex-A72) | verified: all four byte-identical (the machine that wrote them) |
| Linux arm64, second machine (Cortex-A76) | verified 2026-10-14: all four byte-identical, exit 0 |
| Linux amd64 | not yet run (no amd64 machine or emulator available); CI matrix planned |
| macOS arm64 | not yet run |
