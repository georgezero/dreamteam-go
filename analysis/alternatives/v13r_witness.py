#!/usr/bin/env python3
"""V0.13r locked-protocol enforcement witnesses (LC-02): adversarial cases
through the ordinary command line. TEST ONLY.

A scratch Git repository holds the engine source tree (git archive of HEAD's
internal/ and cmd/) and a TEST ONLY lock commit. Its approval and
human-review records carry fixed TEST ONLY markers and are authored by a
TEST ONLY identity, never the owner. The lock pins a synthetic single-context
protocol (scripted model replies, linked clocks, no model endpoint). Each
adversarial variant is its own commit, checked out for its case. Nothing
produced here is a protocol lock.

Every invalid case must be refused by v13r_analyze.py (or the runner
preflight) before the existing verifier is reached. The valid fixture must
reach v13p_verify unchanged: its locked analysis equals the unlocked V0.13p
analysis of the same archive, apart from the added ``lock`` block.

    python3 -I -B analysis/alternatives/v13r_witness.py --binary ./dt --scratch DIR --out FILE
"""
import argparse
import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[1]
sys.path.insert(0, str(HERE))
import v13m_dt13l as D  # noqa: E402  (V0.13m-lite witness helpers)
import v13o_pin as B  # noqa: E402  (V0.13o witness helpers)
import v13p  # noqa: E402
import v13r_lock as L  # noqa: E402
import v13r_preflight as RP  # noqa: E402

SCHEMA = "dreamteam.v13r-lock-witnesses/v1"
OWNER = {"name": "TEST ONLY fixture", "email": "test-only@fixture.invalid"}
DATE = "2000-01-01"
LOCK_ID = "TEST-ONLY-fixture-lock"
ANALYSIS_COPY = ("v13r_analyze.py", "v13r_lock.py", "v13p_verify.py", "v13o_verify.py", "v13m_verify.py",
                 "acceptance_v1.json", "bindings_v2.json")
RECORDER_COPY = ("v13r_preflight.py", "v13r_lock.py", "v13p.py", "v13o.py", "v13m.py", "lifecycle_v4_spec.py",
                 "acceptance_v1.json", "bindings_v2.json")
GIT_ENV = {"GIT_AUTHOR_DATE": "2000-01-01T00:00:00Z", "GIT_COMMITTER_DATE": "2000-01-01T00:00:00Z",
           "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.devnull}


class Repo:
    def __init__(self, path):
        self.path = Path(path)

    def git(self, *args, author=OWNER, data=None):
        env = dict(os.environ, **GIT_ENV, GIT_AUTHOR_NAME=author["name"], GIT_AUTHOR_EMAIL=author["email"],
                   GIT_COMMITTER_NAME=author["name"], GIT_COMMITTER_EMAIL=author["email"])
        r = subprocess.run(["git", "-C", str(self.path), *args], capture_output=True, env=env, input=data)
        if r.returncode:
            raise SystemExit(f"git {args[0]}: {r.stderr.decode()}")
        return r.stdout.decode().strip()

    def commit(self, files, message, branch, base, author=OWNER):
        """A commit on `branch` (from `base`) writing `files` {path: bytes}; checked out afterwards."""
        self.git("checkout", "-q", "-f", "-B", branch, base)
        for p, data in files.items():
            (self.path / p).parent.mkdir(parents=True, exist_ok=True)
            (self.path / p).write_bytes(data)
            self.git("add", "--", p)
        self.git("commit", "-q", "--no-gpg-sign", "-m", message, author=author)
        return self.git("rev-parse", "HEAD")


def fixture_repo(scratch):
    repo = Repo(scratch / "lock-repo")
    if repo.path.exists():
        shutil.rmtree(repo.path)
    repo.path.mkdir(parents=True)
    repo.git("init", "-q")
    tree = subprocess.run(["git", "-C", str(ROOT), "archive", "HEAD", "internal", "cmd"], capture_output=True,
                          check=True).stdout
    subprocess.run(["tar", "-x", "-f", "-", "-C", str(repo.path)], input=tree, check=True)
    repo.git("add", "-A")
    repo.git("commit", "-q", "--no-gpg-sign", "-m", "TEST ONLY fixture base: engine source tree")
    return repo, repo.git("rev-parse", "HEAD")


def schedule_bytes(K, raw, proto, sched):
    doc = {"schema": K.H.SCHEDULE_SCHEMA, "protocol": proto["id"], "protocolSha256": L.sha(raw),
           "orderRule": K.H.ORDER_RULE, "draws": len(sched), "schedule": sched}
    return L.encode(doc)


def record(impl, binary, cli, out, recorder, authorization):
    """A COMPLETE scripted run (no endpoint): linked monotonic and SYSTEM_UTC clocks."""
    K, S = impl.K, impl.S
    m = D.Mono()
    utc = D.LinkedClock(m)
    recorder(cli["ctx"], cli["ctx"].opener(binary), {K.QWEN: S.Model(cli["replies"])}, str(out), authorization,
             state_fn=S.Health(utc), utc=utc, now=m)
    return out


def copies(scratch, name, files, alter=None):
    d = scratch / ("copy-" + name)
    if d.exists():
        shutil.rmtree(d)
    d.mkdir(parents=True)
    for f in files:
        shutil.copyfile(HERE / f, d / f)
    if alter:
        f, fn = alter
        (d / f).write_bytes(fn((d / f).read_bytes()))
    return d


def entry(script, copy_dir=None):
    if copy_dir is None:
        return [sys.executable, "-I", "-B", str(HERE / script)]
    # The copies simulate tampered module files in this repository, so the copied lock module is told the
    # real repository root (it would otherwise derive it from the copy's location).
    boot = ("import sys,runpy;sys.path[:0]=[%r,%r];import v13r_lock;v13r_lock.ROOT=%r;sys.argv=[%r]+sys.argv[1:];"
            "runpy.run_path(%r,run_name='__main__')"
            % (str(copy_dir), str(HERE), str(ROOT), str(copy_dir / script), str(copy_dir / script)))
    return [sys.executable, "-I", "-B", "-c", boot]


def locked_cli(args, analysis_out, copy_dir=None, omit=()):
    """classify then analyze through v13r_analyze.py; whether the verifier was reached."""
    def argv(cmd):
        out = []
        for k, v in args.items():
            if k in omit:
                continue
            out += [k] if v is True else [k, str(v)]
        return entry("v13r_analyze.py", copy_dir) + [cmd] + out
    c = subprocess.run(argv("classify"), capture_output=True, text=True, timeout=900)
    try:
        v = json.loads(c.stdout)
    except ValueError:
        v = None
    if v is None:
        cls = {"reachedVerifier": False, "refusal": "usage" if c.returncode == 2 else "error", "state": None}
    elif v.get("lockRefused"):
        cls = {"reachedVerifier": False, "refusal": v["violations"][0]["invariant"], "state": "INVALID"}
    else:
        cls = {"reachedVerifier": "preserved" in v, "refusal": None, "state": v.get("state")}
    if analysis_out.exists():
        shutil.rmtree(analysis_out)
    analysis_out.mkdir(parents=True)
    r = subprocess.run(argv("analyze") + ([] if "--out" in omit else ["--out", str(analysis_out)]),
                       capture_output=True, text=True, timeout=900)
    refusal = None
    for line in r.stderr.splitlines():
        if line.startswith("LOCK REFUSED "):
            refusal = line.split()[2].rstrip(":")
    written = (analysis_out / "analysis.json").exists()
    return {"classify": cls, "analyze": {"exit": r.returncode, "analysisWritten": written,
                                         "refusal": refusal if refusal else ("usage" if r.returncode == 2 else None)}}


def preflight_cli(args, copy_dir=None):
    argv = entry("v13r_preflight.py", copy_dir) + ["check"]
    for k, v in args.items():
        argv += [k] if v is True else [k, str(v)]
    r = subprocess.run(argv, capture_output=True, text=True, timeout=900)
    try:
        doc = json.loads(r.stdout)
    except ValueError:
        return {"preflight": "ERROR", "code": None}
    return {"preflight": doc["preflight"], "code": doc.get("code")}


def run(binary, scratch):
    scratch = Path(scratch)
    scratch.mkdir(parents=True, exist_ok=True)
    impl = D.Impl("v13m", scratch)
    K = impl.K
    draft = B.setup(impl, scratch, "v13r-draft", v13p.draft_protocol)
    locked = B.finish(impl, scratch, "v13r-locked", L.locked_protocol(draft["p"]), draft["rows"], draft["cd"])
    praw = locked["path"].read_bytes()
    sraw = schedule_bytes(K, praw, locked["p"], locked["sched"])
    repo, base = fixture_repo(scratch)
    files = {"lock/protocol-locked.json": praw, "lock/schedule-locked.json": sraw,
             "lock/HUMAN-REVIEW-TEST-ONLY.md": ("TEST ONLY fixture. No human review was performed.\nVerdict: %s\n"
                                                % L.REVIEW[L.TEST]).encode()}
    files["lock/APPROVAL-TEST-ONLY.md"] = L.approval_text(LOCK_ID, L.sha(praw), L.TEST, OWNER,
                                                          "This is not an owner approval.\n").encode()
    candidate = (ROOT / "docs/v13q/LOCK-CANDIDATE.json").read_bytes()

    def manifest(fs, **over):
        doc = L.create(purpose=L.TEST, lock_id=LOCK_ID, protocol_file="lock/protocol-locked.json",
                       protocol_raw=fs["lock/protocol-locked.json"], schedule_file="lock/schedule-locked.json",
                       schedule_raw=fs["lock/schedule-locked.json"], binary=binary, platform="test-fixture",
                       candidate_raw=candidate, base_commit=base, lock_file="lock/LOCK.json", owner=OWNER, date=DATE,
                       approval_file="lock/APPROVAL-TEST-ONLY.md", approval_raw=fs["lock/APPROVAL-TEST-ONLY.md"],
                       review_file="lock/HUMAN-REVIEW-TEST-ONLY.md", review_raw=fs["lock/HUMAN-REVIEW-TEST-ONLY.md"],
                       acceptance_path=str(HERE / "acceptance_v1.json"), repo=str(repo.path))
        return doc

    good = manifest(files)
    lock_raw = L.encode(good)
    commit = repo.commit(dict(files, **{"lock/LOCK.json": lock_raw}), "TEST ONLY fixture lock", "lock", base)
    # Inputs are read from copies outside the fixture working tree, so checking out a variant never removes them.
    inputs = scratch / "inputs"
    inputs.mkdir(exist_ok=True)
    lock_path = inputs / "LOCK.json"
    lock_path.write_bytes(lock_raw)
    sched_path = inputs / "schedule-locked.json"
    sched_path.write_bytes(sraw)

    # Archives. The valid fixture is recorded only after preflight, with the lock binding.
    pf = RP.preflight(str(lock_path), commit, str(locked["path"]), str(sched_path),
                      str(draft["cd"]), binary, str(repo.path), test_only=True)
    valid = record(impl, binary, locked, scratch / "arc-valid",
                   lambda *a, **k: RP.record_after_preflight(pf, *a, **k), "TEST ONLY fixture run")
    draft_arc = record(impl, binary, draft, scratch / "arc-draft",
                       lambda ctx, op, tr, out, auth, **k: v13p.run_schedule(draft["path"].read_bytes(), draft["sched"],
                                                                             ctx, op, tr, out, auth, **k),
                       "TEST ONLY draft run")
    rp = copy.deepcopy(locked["p"])
    rp["stop"]["consecutiveTransportFailures"] = 6
    repinned = B.finish(impl, scratch, "v13r-repinned", rp, draft["rows"], draft["cd"])
    repinned_arc = record(impl, binary, repinned, scratch / "arc-repinned",
                          lambda ctx, op, tr, out, auth, **k: K.run_schedule(rp, repinned["ph"], repinned["sched"],
                                                                             ctx, op, tr, out, auth, **k),
                          L.binding(pf.identity, L.sha(praw)) + " TEST ONLY")
    psha = L.sha(praw)

    def relabel(a, with_binding):
        a.doc["protocolSha256"] = psha
        a.doc["protocol"] = locked["p"]["id"]
        if with_binding:
            a.doc["authorization"] = L.binding(pf.identity, psha) + " TEST ONLY forged"
        if a.seal is not None:
            a.seal["protocol"] = {"id": locked["p"]["id"], "sha256": psha}
    relabelled = scratch / "arc-draft-relabelled-resealed"
    forged = scratch / "arc-draft-fully-forged"
    for out, wb in ((relabelled, False), (forged, True)):
        if out.exists():
            shutil.rmtree(out)
        D.mutate(impl, draft_arc, out, lambda a, wb=wb: relabel(a, wb))

    base_args = {"--lock": lock_path, "--lock-commit": commit, "--repo": repo.path, "--protocol": locked["path"],
                 "--schedule": sched_path, "--contexts": draft["cd"], "--binary": binary,
                 "--dir": valid, "--test-only": True}
    results = []

    def case(name, expected, args=None, copy_dir=None, omit=(), checkout="lock", note=None):
        repo.git("checkout", "-q", "-f", checkout)
        r = locked_cli(dict(base_args, **(args or {})), scratch / "analysis" / name, copy_dir, omit)
        r.update(case=name, expected=expected)
        if note:
            r["note"] = note
        results.append(r)
        return r

    def variant(name, fs=None, doc_fn=None, raw=None, author=OWNER):
        fs = dict(files, **(fs or {}))
        if raw is None:
            d = manifest(fs)
            if doc_fn:
                doc_fn(d)
            raw = L.encode(d)
        c = repo.commit(dict(fs, **{"lock/LOCK.json": raw}), "TEST ONLY variant " + name, "v-" + name, base, author)
        p = inputs / ("LOCK-" + name + ".json")
        p.write_bytes(raw)
        return {"--lock-commit": c, "--lock": p}

    # --- valid fixture -----------------------------------------------------------------
    r = case("valid-test-only-lock", "reaches verifier")
    unlocked = subprocess.run([sys.executable, "-I", "-B", str(HERE / "v13p_verify.py"), "analyze", "--binary", binary,
                               "--protocol", str(locked["path"]), "--contexts", str(draft["cd"]), "--dir", str(valid),
                               "--out", str(scratch / "analysis" / "unlocked")], capture_output=True, text=True)
    a1 = json.loads((scratch / "analysis" / "valid-test-only-lock" / "analysis.json").read_text())
    lock_block = a1.pop("lock")
    a0 = json.loads((scratch / "analysis" / "unlocked" / "analysis.json").read_text()) if unlocked.returncode == 0 else None
    r["sameAsUnlockedV13pAnalysis"] = a0 is not None and a0 == a1
    r["outputMarker"] = lock_block["marker"]

    # --- invocation ----------------------------------------------------------------------
    case("missing-lock", "refused", omit=("--lock",))
    case("missing-lock-commit", "refused", omit=("--lock-commit",))
    case("test-lock-without-test-only-flag", "refused", omit=("--test-only",))
    case("candidate-instead-of-lock", "refused", {"--lock": ROOT / "docs/v13q/LOCK-CANDIDATE.json"})
    # --- manifest structure ----------------------------------------------------------------
    c = variant("status-candidate", doc_fn=lambda d: d.update(status="CANDIDATE"))
    case("status-not-locked", "refused", c, checkout="v-status-candidate")
    c = variant("missing-approval", doc_fn=lambda d: d.pop("approval"))
    case("missing-approval", "refused", c, checkout="v-missing-approval")
    c = variant("missing-field", doc_fn=lambda d: d["engine"].pop("binarySha256"))
    case("missing-engine-field", "refused", c, checkout="v-missing-field")
    dup = lock_raw.replace(b'\n "lockId":', b'\n "lockId": "x",\n "lockId":', 1)
    c = variant("duplicate-key", raw=dup)
    case("duplicate-manifest-key", "refused", c, checkout="v-duplicate-key")
    nested = lock_raw.replace(b'"binarySha256":', b'"binarySha256": "0",\n  "binarySha256":', 1)
    c = variant("nested-duplicate-key", raw=nested)
    case("nested-duplicate-manifest-key", "refused", c, checkout="v-nested-duplicate-key")
    c = variant("claims-owner-approval", doc_fn=lambda d: d["approval"].update(decision=L.DECISION[L.RUN]))
    case("test-lock-claiming-owner-approval", "refused", c, checkout="v-claims-owner-approval")
    # --- Git identity and approval ----------------------------------------------------------
    case("lock-commit-not-in-history", "refused", checkout=base)
    case("unknown-lock-commit", "refused", {"--lock-commit": "0" * 40})
    case("abbreviated-lock-commit", "refused", {"--lock-commit": commit[:12]})
    edited = scratch / "LOCK-edited.json"
    edited.write_bytes(lock_raw.replace(b'"lockId": "TEST-ONLY-fixture-lock"', b'"lockId": "TEST-ONLY-other"'))
    case("manifest-differs-from-lock-commit", "refused", {"--lock": edited})
    c = variant("extra-file", fs={"lock/EXTRA.txt": b"x\n"})
    case("lock-commit-changes-other-files", "refused", c, checkout="v-extra-file")
    c = variant("other-author", author={"name": "Someone Else", "email": "else@fixture.invalid"})
    case("lock-commit-not-by-approval-owner", "refused", c, checkout="v-other-author")
    alt = dict(files)
    alt["lock/APPROVAL-TEST-ONLY.md"] = files["lock/APPROVAL-TEST-ONLY.md"] + b"Altered after approval.\n"
    good_alt_manifest = L.encode(manifest(files))   # manifest still pins the original approval bytes
    c = variant("altered-approval", fs=alt, raw=good_alt_manifest)
    case("altered-approval-record", "refused", c, checkout="v-altered-approval")
    noline = dict(files)
    noline["lock/APPROVAL-TEST-ONLY.md"] = files["lock/APPROVAL-TEST-ONLY.md"].replace(b"Decision:", b"Note:")
    c = variant("approval-without-decision", fs=noline)
    case("approval-record-without-decision", "refused", c, checkout="v-approval-without-decision")
    # --- content ---------------------------------------------------------------------------
    case("re-pinned-protocol-same-id", "refused", {"--protocol": repinned["path"]})
    case("archive-under-re-pinned-protocol", "refused", {"--dir": repinned_arc})
    case("draft-protocol-archive", "refused", {"--dir": draft_arc})
    case("relabelled-archive-recomputed-hashes-and-seal", "refused", {"--dir": relabelled})
    case("fully-forged-archive-binding-hashes-and-seal", "reaches verifier (limitation)", {"--dir": forged},
         note="unsigned archives: rewriting every field including the lock binding and recomputing every hash and "
              "the seal is not detectable by content checks; archive custody is required (docs/v13r/README.md)")
    d = copies(scratch, "acceptance", ANALYSIS_COPY, ("acceptance_v1.json",
                                                      lambda b: b.replace(b'"toleranceSeconds": 60',
                                                                          b'"toleranceSeconds": 120')))
    case("changed-acceptance-file", "refused", copy_dir=d)
    bad_sched = scratch / "schedule-changed.json"
    bad_sched.write_bytes(sraw.replace(b'"orderRule": "', b'"orderRule": "x', 1))
    case("changed-schedule", "refused", {"--schedule": bad_sched})
    d = copies(scratch, "verifier", ANALYSIS_COPY, ("v13o_verify.py", lambda b: b + b"# changed\n"))
    case("changed-verifier-module", "refused", copy_dir=d)
    d = copies(scratch, "analysis", ANALYSIS_COPY, ("v13r_analyze.py", lambda b: b + b"# changed\n"))
    case("changed-analysis-module", "refused", copy_dir=d)
    d = copies(scratch, "identical", ANALYSIS_COPY)
    case("byte-identical-module-copies", "reaches verifier", copy_dir=d)
    wrong = scratch / "dt-wrong"
    wrong.write_bytes(Path(binary).read_bytes() + b"\0")
    wrong.chmod(0o755)
    case("wrong-engine-binary", "refused", {"--binary": wrong})

    def other_settings(d):
        f = d["models"]["settings"]["files"]
        f[sorted(f)[0]] = dict(f[sorted(f)[0]], sha256="0" * 64)
    c = variant("model-settings", doc_fn=other_settings)
    case("changed-model-settings", "refused", c, checkout="v-model-settings")
    c = variant("model-request", doc_fn=lambda d: d["models"]["request"].update(timeoutSeconds=301))
    case("changed-model-request-settings", "refused", c, checkout="v-model-request")
    c = variant("outcome", doc_fn=lambda d: d["outcome"].update(
        definition=dict(d["outcome"]["definition"], estimand="changed"),
        sha256=L.sha(L.canon(dict(d["outcome"]["definition"], estimand="changed")).encode())))
    case("changed-outcome-definition", "refused", c, checkout="v-outcome")

    # --- runner preflight (fresh processes; strict module set) -------------------------------
    repo.git("checkout", "-q", "-f", "lock")
    pargs = {"--lock": lock_path, "--lock-commit": commit, "--repo": repo.path, "--protocol": locked["path"],
             "--schedule": sched_path, "--contexts": draft["cd"], "--binary": binary,
             "--test-only": True}
    pre = []
    for name, expected, over, cdir in (
            ("valid", "PASSED", {}, None),
            ("candidate-instead-of-lock", "REFUSED", {"--lock": ROOT / "docs/v13q/LOCK-CANDIDATE.json"}, None),
            ("re-pinned-protocol-same-id", "REFUSED", {"--protocol": repinned["path"]}, None),
            ("draft-protocol", "REFUSED", {"--protocol": draft["path"]}, None),
            ("changed-schedule", "REFUSED", {"--schedule": bad_sched}, None),
            ("wrong-engine-binary", "REFUSED", {"--binary": wrong}, None),
            ("changed-recorder-module", "REFUSED", {}, copies(scratch, "recorder", RECORDER_COPY,
                                                              ("v13m.py", lambda b: b + b"# changed\n"))),
            ("changed-acceptance-file", "REFUSED", {}, copies(scratch, "rec-acceptance", RECORDER_COPY,
                                                              ("acceptance_v1.json", lambda b: b + b"\n")))):
        r = preflight_cli(dict(pargs, **over), cdir)
        r.update(case=name, expected=expected)
        pre.append(r)
    runner_run_lock = None
    try:
        fake = RP.Preflight(praw, locked["p"], locked["sched"], draft["rows"], dict(pf.identity, purpose=L.RUN),
                            pf.binding, [])
        RP.record_after_preflight(fake, None, None, {}, str(scratch / "never"), "x")
    except L.LockError as x:
        runner_run_lock = x.code
    return results, pre, runner_run_lock, unlocked.returncode


def meets(r):
    if r["expected"].startswith("reaches verifier"):
        ok = r["classify"]["reachedVerifier"] and r["analyze"]["analysisWritten"]
        if r["case"] == "valid-test-only-lock":
            ok = ok and r["classify"]["state"] == "COMPLETE" and r.get("sameAsUnlockedV13pAnalysis") is True \
                and r.get("outputMarker") == "TEST ONLY — NOT A PRIMARY RESULT"
        return ok
    return (not r["classify"]["reachedVerifier"] and r["classify"]["refusal"] is not None
            and not r["analyze"]["analysisWritten"] and r["analyze"]["refusal"] is not None
            and r["analyze"]["exit"] != 0)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    results, pre, runner_run_lock, unlocked_exit = run(str(Path(a.binary).resolve()), a.scratch)
    for r in results:
        r["meetsExpectation"] = meets(r)
    for r in pre:
        r["meetsExpectation"] = r["preflight"] == r["expected"]
    doc = {"schema": SCHEMA, "notice": "TEST ONLY. Synthetic fixtures; no protocol lock is created, no model is "
                                       "contacted.",
           "lockedAnalysis": results, "runnerPreflight": pre,
           "recordUnderRunLock": {"refusal": runner_run_lock, "meetsExpectation": runner_run_lock == "LK-RUNNER"},
           "unlockedV13pAnalysisExit": unlocked_exit}
    doc["allMeetExpectation"] = all(r["meetsExpectation"] for r in results + pre) and \
        doc["recordUnderRunLock"]["meetsExpectation"]
    D.write(a.out, doc)
    for r in results:
        print(f"{'ok ' if r['meetsExpectation'] else 'BAD'} analysis  {r['case']:48} "
              f"{r['classify']['refusal'] or r['classify']['state']!s:14} {r['analyze']['refusal'] or 'written'}")
    for r in pre:
        print(f"{'ok ' if r['meetsExpectation'] else 'BAD'} preflight {r['case']:48} {r['preflight']} {r['code'] or ''}")
    print(f"record_after_preflight under a RUN lock: {runner_run_lock}")
    if not doc["allMeetExpectation"]:
        raise SystemExit("a V0.13r witness does not meet its expectation")


if __name__ == "__main__":
    main()
