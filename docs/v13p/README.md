# V0.13p — Protocol parsing and lock hygiene

**Status: IMPLEMENTED (pre-lock corrections only).** Human presentation
review: **PENDING**. Protocol lock: **NO GO**. Live inference: **NO GO**.
V0.13.0 release: **NO GO**.

Starting commit: `4373fd2`, the V0.13o independent recheck: DT13M-01
RESOLVED, Technical GO.

## Scope (owner decision)

| Finding | Owner decision | V0.13p |
|---|---|---|
| DT13O-01: the recorder and verifier disagree on duplicate-key acceptance files (MINOR) | Fix before lock | **FIXED**: strict policy parsing, R-POLICY-001 |
| DT13O-04: the protocol file was parsed loosely (NOTE) | Fix alongside DT13O-01 | **FIXED**: strict protocol parsing, R-PROTO-001 |
| DT13O-02: protection rests on the protocol hash binding (NOTE) | Require in the lock manifest | **DOCUMENTED** ([LOCK-REQUIREMENTS.md](LOCK-REQUIREMENTS.md) §1) |
| DT13O-05: the future runner's entry point (NOTE) | Mandatory | **DOCUMENTED** (§2). `v13p.run_schedule` takes the exact protocol bytes. No runner is implemented. |
| DT13O-03: timestamp observability (NOTE) | Accepted as a documented limitation | **DOCUMENTED** (§3) |

There is no other recorder development.

## Changes (successor modules; V0.13o and earlier unchanged)

| File | Role |
|---|---|
| `analysis/alternatives/v13p.py` | Recorder entry point. It wraps `v13o.py`, unchanged, and has its own implementation of `dreamteam.strict-json/v1`. The acceptance file must be a well-formed policy (R-POLICY-001) and the protocol file a strict JSON object (R-PROTO-001). `run_schedule` takes the protocol's exact bytes. Commands: `protocol`, `schedule`, `check`, `run`. |
| `analysis/alternatives/v13p_verify.py` | Verifier entry point. It wraps `v13o_verify.py`, unchanged, and uses the V0.13m-lite verifier's strict parser. It applies R-PROTO-001, then R-POLICY-001, then R-PIN-001. It imports no recorder. |
| `analysis/alternatives/test_v13p.py` | Unit tests. |
| `analysis/alternatives/v13p_pin.py`, `docs/v13p/pin/results.json` | Before/after witnesses through the ordinary command line. The output is deterministic. |
| `docs/v13p/LOCK-REQUIREMENTS.md` | Lock-manifest and runner requirements. |
| `docs/v13p/IDENTITIES.json` | SHA-256 pins. |

### Accepted syntax (identical in both components)

Both components use `dreamteam.strict-json/v1`:

- strict UTF-8;
- no duplicate object key at any nesting level, including escaped-equivalent
  keys;
- no `NaN`, `Infinity` or `-Infinity`, and no number that overflows to
  infinity;
- integers stay exact;
- no trailing content.

The recorder and verifier implement the parser separately, so the verifier
stays independent. A differential corpus checks that they agree: 66 inputs
are accepted or refused identically, and every accepted input gives the same
value under strict typed equality.

### R-POLICY-001 (acceptance file structure, both components)

The acceptance file must be an object with:

- `schema` `dreamteam.acceptance-standard/v1`;
- a non-empty `relationships` list of objects, each with a non-empty string
  `id`;
- unique ids;
- R-HALT-003 with a positive integer `toleranceSeconds`.

Any other file is refused. The recorder raises an error; the verifier marks
the archive INVALID and refuses analysis. Only after this check does the
verifier apply R-PIN-001: the exact-bytes hash, strict equality to the
applied standard, the integer tolerance and the verbatim statements.

### R-PROTO-001 (protocol file, both components)

The protocol file must parse strictly to an object. V0.13o read it with
`json.load`, which keeps the last of duplicated keys. Now:

- the recorder's `schedule`, `check`, `run` and `run_schedule` refuse such a
  file;
- the verifier's `classify` reports INVALID (R-PROTO-001), and `analyze`
  refuses.

## Protocol identity: unchanged

The protocol is `v13o-alt-1`, schema `dreamteam.alternatives-experiment/v10`,
SHA-256 `202548ad…`. The schedule is `2be938b6…` and the acceptance file
`c6af89ed…`.

- **Same reading.** The committed protocol and acceptance file parse strictly
  to exactly the values V0.13o read.
- **Byte-identical regeneration.** `v13p.py protocol` and `schedule` reproduce
  both files byte for byte.
- **No version change needed.** The protocol document's content and meaning
  do not change. Strict parsing is a property of the entry points, and the
  lock manifest pins those by identity (LOCK-REQUIREMENTS §1, items 3–4).

## Witnesses (`docs/v13p/pin/results.json`)

These are genuine-provenance wall-clock halts recorded with a scripted model
and linked clocks, with no endpoint. They are classified and analysed through
the ordinary command line. Every mutation recomputes every hash and the seal.

The two DT13O-01 cases use byte-identical verifier copies placed next to the
substituted acceptance file.

| Case | V0.13o | V0.13p |
|---|---|---|
| **DT13O-01 (a)**: duplicate R-HALT-003 (60 then 120); protocol re-pinned to it with 120 s; archive 60.1 s beyond the timestamp bound | recorder **refuses**; verifier **accepts, 120 s applied** (disagreement) | recorder refuses; verifier INVALID (R-POLICY-001), refused |
| **DT13O-01 (b)**: duplicate top-level key; protocol re-pinned | recorder **accepts**; verifier INVALID (R-PIN-001) (disagreement) | recorder refuses; verifier INVALID (R-POLICY-001), refused |
| **DT13O-04**: protocol text `toleranceSeconds` 120 then 60, bound by SHA-256 to the archive | recorder and verifier **accept** (last key wins) | `run_schedule` refuses the bytes; verifier INVALID (R-PROTO-001), refused |
| **DT13M-01** original witness: ambient acceptance 120 s, 60.1 s archive | INVALID (R-PIN-001), refused | INVALID (R-PIN-001), refused |
| Control: genuine halt, recorded through `v13p.run_schedule` | — | PARTIAL, accepted |
| 59.9 s beyond the timestamp bound | — | PARTIAL, accepted |
| 60.1 s beyond the timestamp bound | — | INVALID (R-HALT-003), refused |

Unit tests (`test_v13p.py`, 14 tests) cover:

- **Refused by both components (R-POLICY-001):** 30 acceptance-file variants:
  - both DT13O-01 counterexamples;
  - duplicate rule ids;
  - duplicate top-level and nested keys at several depths;
  - `NaN`, `Infinity`, `-Infinity` and `1e999`;
  - invalid UTF-8 and trailing content;
  - invalid structures: wrong schema, absent, empty or non-list
    relationships, a non-object rule, a missing, empty or non-string id,
    R-HALT-003 absent, and a tolerance that is absent, a float, a boolean, a
    string, zero or negative.

  Protocols re-pinned to each of these files are refused by both as well.
- **Accepted by both:** 4 well-formed files, including the committed one and
  a re-serialization.
- **Protocol variants (R-PROTO-001):** 10, including nested and deep
  duplicates and non-finite numbers.
- **Unchanged R-PIN-001 refusals:** the 16 from V0.13o.
- **Command-line refusals.**
- **The runner entry point:** it refuses before recording and passes the exact
  protocol hash.

## Preservation

- **New files only.** No module, protocol, schedule, evidence or witness from
  V0.13o or earlier changes. The only edits are to `.github/workflows/ci.yml`
  (a new V0.13p step), `docs/PROGRESS.md` and the status line of
  `docs/v13o/README.md`.
- **Checks that pass:**
  - `v13m_preserved.py` (3,606 protected files);
  - `v13m_identities.py --check`;
  - `docs/v13o/IDENTITIES.json`;
  - `test_v13o.py`;
  - the V0.13o witnesses, byte-identical;
  - `docs/v13p/IDENTITIES.json`.
- **Unchanged material:** the human-review freeze (`ee987a1`) and all
  model-visible content.

## Remaining

1. The independent human presentation review (PENDING; the owner names the
   reviewer).
2. Lock candidate and owner lock gate, against
   [LOCK-REQUIREMENTS.md](LOCK-REQUIREMENTS.md). Not started; nothing is
   locked automatically.
