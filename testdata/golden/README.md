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

`established-7d.actions.json` (input) holds a future-day shift extension,
a walk-in policy change and back, a worklist policy change, a no-op and
three rejected actions (unknown employee, a retroactive extension, too
many minutes). Every reference was produced with `-verify-replay`: the
run's own record was replayed from scratch and matched.

## Check

    go test ./cmd/dreamteam -run Golden              # all four (-short skips 90 days)
    dreamteam run -scenario established -seed 1 -days 7 \
        -verify-replay -format json -golden testdata/golden/established-seed1-7d.json

Exit 0: identical outcomes. Exit 4: a difference — stderr names the first
differing JSON path with both values, the number of differences and the
digests that differ. Exit 5: not comparable (another schema, scenario
fingerprint, seed or run length; or another engine version, unless
`-allow-engine-mismatch`, which compares outcomes and still reports the
engine difference) or a malformed file.

## Update (explicit only)

Golden files change only when gameplay is meant to change. Regenerate with
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

A new engine version with unchanged outcomes needs no regeneration: the
test compares with the engine mismatch allowed and reports it. Regenerate
when convenient so the files name the current engine (they are then also
byte-identical).

## Cross-platform procedure

On each platform (Linux amd64, Linux arm64, macOS arm64), from a clean
checkout of the same commit:

    go build -o dt ./cmd/dreamteam
    for args in "-days 7" "-days 90" "-scenario mini -days 7" \
                "-days 7 -actions testdata/golden/established-7d.actions.json"; do
        ./dt run -seed 1 $args -verify-replay -format json > out.json   # then:
        cmp out.json testdata/golden/<matching file>                     # byte-identical?
    done

or simply `go test ./cmd/dreamteam -run Golden`. Integers only, fixed
field order, sorted maps and 64-bit big-endian digest encodings mean the
results should not depend on the platform, but that is only established
where it has been run:

| Platform | Status |
|---|---|
| Linux arm64, Raspberry Pi 4 (Cortex-A72) | verified: all four byte-identical (the machine that wrote them) |
| Linux arm64, second machine (Cortex-A76) | verified 2026-10-14: all four byte-identical, exit 0 |
| Linux amd64 | not yet run (no amd64 machine or emulator available); CI matrix planned |
| macOS arm64 | not yet run |
