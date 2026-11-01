#!/usr/bin/env python3
"""The shared-prefix comparison, offline prototype (shared-prefix/v1;
docs/benchmark/v10a-dev/DESIGN.md).

Two arms branch from one recorded decision point:
* both replay the recording's decisions before it, with the recording
  agent's own checks (every digest, memory, prompt, attempt input and
  validate answer), so both reach the recorded state;
* both rebuild the point's conversation up to the recorded initial
  proposal and take that proposal, verified against its SHA-256, as their
  shared first reply;
* control continues as llm-agent 3 would: the proposal is validated;
* treatment continues as llm-agent 4 would: the proposal's staffing
  actions are previewed and the preview is shown before anything is
  validated;
* every later reply comes from the spec, recorded or scripted, each bound
  to the SHA-256 of the message it answers. No model is called;
* after the branch point both arms submit a declared fixed suffix (no
  action), so the simulator continues deterministically.

    # one arm, as an external benchmark policy
    python3 -I -B analysis/repeatability/shared_prefix.py policy --spec SPEC --arm control|treatment
    # the specs for the V0.9c preview points, and their run configurations
    python3 -I -B analysis/repeatability/shared_prefix.py specs --out DIR
    # the comparison of the arms' runs
    python3 -I -B analysis/repeatability/shared_prefix.py compare --dir DIR --runs DIR --out DIR

A prototype: its scripted replies are not model behaviour, and nothing it
reports is an effect of the preview.
"""

import argparse
import glob
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, os.path.join(ROOT, "agents", "llm"))
import llm_agent3 as A3  # noqa: E402  (unchanged: replay, prompt, parser, batching)
import llm_agent4 as A4  # noqa: E402  (unchanged: the preview step and its replay)
sys.path.insert(0, os.path.join(ROOT, "analysis", "evidence"))
import financial_outcomes as FO  # noqa: E402  (metric helpers, read-only)

VERSION = "shared-prefix/v1"
SPEC_SCHEMA = "dreamteam.shared-prefix-spec/v1"
BRANCH_SCHEMA = "dreamteam.shared-prefix-branch/v1"
COMPARE_SCHEMA = "dreamteam.shared-prefix-comparison/v1"
ARMS = ("control", "treatment")
SCRIPTED_NO_ACTION = json.dumps({"reason": "Scripted reply (shared-prefix/v1 prototype): no action.", "actions": []})
sha256 = A3.sha256


class MissingEvidence(Exception):
    """The branch cannot be made from the recording: a stable code and a
    message. The arm submits nothing it has not verified."""

    def __init__(self, code, message):
        super().__init__(f"{code}: {message}")
        self.code = code


def module_for(rec):
    return A4 if rec.get("schema") == A4.RECORD_SCHEMA else A3


def full_text(at, outputs=None):
    """An attempt's output text: the record's if it holds all of it,
    otherwise the spec's copy (outputs, by attempt id), which must have the
    recorded output's SHA-256."""
    o = at.get("output")
    if not isinstance(o, dict):
        raise MissingEvidence("no_output", f"attempt {at.get('id')} has no recorded output")
    if isinstance(o.get("text"), str) and len(o["text"]) == o.get("chars") and sha256(o["text"]) == o.get("sha256"):
        return o["text"]
    t = (outputs or {}).get(at.get("id"))
    if t is None:
        raise MissingEvidence("output_truncated", f"attempt {at.get('id')}'s output is not recorded in full")
    if sha256(t) != o.get("sha256"):
        raise MissingEvidence("output_differs", f"attempt {at.get('id')}: the spec's output is not the recorded one")
    return t


def proposal_attempt(rec):
    """The recorded initial proposal: the first parsed reply at the point
    with a staffing action (llm-agent 4's preview rule)."""
    for at in rec.get("attempts") or []:
        if A4.preview_due(at, False):
            return at["attempt"]
    raise MissingEvidence("no_staffing_proposal", f"decision point {rec.get('point')} has no staffing proposal")


def prefix_conversation(h, rec, k, obs, omitted, catalog, prompt, outputs=None):
    """The messages up to the one attempt k answered, rebuilt and checked
    against the record, with the agent's state at that attempt:
    (messages, failures so far, details still open)."""
    pset = rec["promptSettings"]
    if sha256(prompt) != rec.get("promptSha256"):
        raise MissingEvidence("prompt_differs", "the rebuilt prompt is not the recorded one")
    msgs = [{"role": "system", "content": A3.SYSTEM_PROMPT}, {"role": "user", "content": prompt}]
    atts, failures, details_open = rec["attempts"], 0, bool(pset.get("allowDetails"))
    for j in range(1, k):
        prev, at = atts[j - 1], atts[j]
        kind = (at.get("input") or {}).get("kind")
        if prev.get("error") is not None:
            failures += 1
        if kind == "resend":
            continue
        out = full_text(prev, outputs)
        if kind == "details":
            text = A3.details_text(h, prev["detailsRequested"], obs, omitted, catalog, pset["maxPromptChars"] // 4)
            details_open = False
        elif kind == "retry" and not at["input"].get("feedbackTruncated"):
            text = A3.retry_text(at["input"]["feedback"])
        else:
            raise MissingEvidence("unsupported_prefix", f"attempt {at.get('id')}: input {kind!r} cannot be rebuilt")
        if sha256(text) != at["input"].get("sha256"):
            raise MissingEvidence("input_differs", f"attempt {at.get('id')}: the rebuilt input is not the recorded one")
        msgs += [{"role": "assistant", "content": out}, {"role": "user", "content": text}]
    return msgs, failures, details_open


class Replies:
    """The spec's replies for one arm, in order, each checked against the
    message it answers."""

    def __init__(self, replies):
        self.replies, self.i = list(replies), 0

    def next(self, last_message):
        if self.i >= len(self.replies):
            raise MissingEvidence("no_reply", f"the spec has no reply {self.i + 1} for this arm")
        r = self.replies[self.i]
        self.i += 1
        if r.get("answers") is not None and r["answers"] != sha256(last_message):
            raise MissingEvidence("reply_not_for_this_message",
                                  f"reply {self.i} answers another message ({r.get('from')})")
        if r.get("sha256") is not None and r["sha256"] != sha256(r["content"]):
            raise MissingEvidence("reply_differs", f"reply {self.i} is not the recorded output ({r.get('from')})")
        return r


def branch(h, arm, rec, obs, omitted, catalog, prompt, clock, replies, max_retries, outputs=None):
    """One arm at the branch point: (batches, trace). The loop is the
    recording agent's (llm-agent 3 for control, 4 for treatment) from the
    proposal attempt on, with the spec's replies for the model's."""
    pset = rec["promptSettings"]
    k = proposal_attempt(rec)
    msgs, failures, details_open = prefix_conversation(h, rec, k, obs, omitted, catalog, prompt, outputs)
    proposal = full_text(rec["attempts"][k - 1], outputs)
    types = {t["type"] for t in catalog.get("types", [])}
    trace = {"proposalAttempt": rec["attempts"][k - 1]["id"], "proposalSha256": sha256(proposal),
             "failuresBefore": failures, "steps": [], "preview": None}
    content, source, previewed, used, prop_actions = proposal, "shared", arm == "control", None, None
    while failures <= max_retries:
        step = {"kind": "reply", "source": source, "answers": sha256(msgs[-1]["content"]),
                "sha256": sha256(content), "chars": len(content), "parse": None, "actions": None,
                "validation": None, "error": None}
        trace["steps"].append(step)
        error, review, asked = None, None, None
        try:
            try:
                d = A3.parse_decision(content, types, details_open)
            except ValueError as e:
                step["parse"] = str(e).split(":", 1)[0]
                raise
            step["parse"] = "ok"
            step["actions"] = d["actions"]
            if prop_actions is None:
                prop_actions = d["actions"]
            if "details" in d:
                asked = d["details"]
            else:
                at = {"parse": {"ok": True}, "detailsRequested": None, "actions": d["actions"]}
                if A4.preview_due(at, previewed):
                    previewed = True
                    pv, review = A4.request_preview(h, d["actions"], clock, pset)
                    trace["preview"] = {"status": pv["status"], "answerSha256": pv["answerSha256"],
                                        "message": pv["message"], "digestsUnchanged": pv["digestsUnchanged"],
                                        "error": pv["error"]}
                if review is None and d["actions"]:
                    step["validation"], refused = [], []
                    for batch, idx in A3.split(d["actions"]):
                        v = h.request("validate", {"actions": batch})
                        step["validation"].append({"valid": bool(v.get("valid")), "refused": sorted(
                            r.get("code") for r in v.get("results") or [] if not r.get("ok"))})
                        if not v.get("valid"):
                            refused += A3.refusals(v, idx, len(trace["steps"]))
                    if refused:
                        raise ValueError("refused: " + "; ".join(
                            f"action {r['action']}: {r['code']}: {r['message']}" for r in refused))
        except A3.ProtocolError as e:
            error = f"invalid_action: validate answered {e.code}: {e.message}"
        except ValueError as e:
            error = str(e)
        step["error"] = error
        if asked is not None:
            details_open = False
            text = A3.details_text(h, asked, obs, omitted, catalog, pset["maxPromptChars"] // 4)
            msgs = msgs + [{"role": "assistant", "content": content}, {"role": "user", "content": text}]
        elif review is not None:
            details_open = False
            msgs = msgs + [{"role": "assistant", "content": content}, {"role": "user", "content": review}]
        elif error is None:
            used = d["actions"]
            break
        else:
            failures += 1
            if failures > max_retries:
                break
            msgs = msgs + [{"role": "assistant", "content": content},
                           {"role": "user", "content": A3.retry_text(error)}]
        r = replies.next(msgs[-1]["content"])
        content, source = r["content"], r["source"]
    batches = [b for b, _ in A3.split(used)] if used else []
    first_after = next((s["actions"] for s in trace["steps"][1:] if s["parse"] == "ok"), None)
    trace.update({"batches": batches,
                  "outcome": "actions" if batches else ("no_action" if used is not None else "fallback"),
                  "messages": [{"role": m["role"], "sha256": sha256(m["content"]), "chars": len(m["content"])}
                               for m in msgs],
                  "response": A4.response_of(prop_actions, first_after) if arm == "treatment" and
                  trace["preview"] and trace["preview"]["status"] == "ok" else None,
                  "final": A4.response_of(prop_actions, used) if used is not None else "fallback"})
    return batches, trace


# --- the policy ----------------------------------------------------------

def load_spec(path):
    with open(path, "rb") as f:
        raw = f.read()
    spec = json.loads(raw)
    if spec.get("schema") != SPEC_SCHEMA:
        raise MissingEvidence("spec", f"{path} is not a {SPEC_SCHEMA} spec")
    return spec, hashlib.sha256(raw).hexdigest()


def policy(spec_path, arm, evidence, stdin=sys.stdin, stdout=sys.stdout):
    spec, spec_sha = load_spec(spec_path)
    rpath = os.path.join(ROOT, spec["recording"])
    with open(rpath, "rb") as f:
        raw = f.read()
    if hashlib.sha256(raw).hexdigest() != spec["recordingSha256"]:
        raise MissingEvidence("recording_differs", f"{spec['recording']} is not the spec's recording")
    first = json.loads(raw.decode("utf-8").splitlines()[0])
    M = module_for(first)
    replayer = A3.Replayer(M.load_recordings([rpath]))
    ev = A3.Evidence(evidence)
    h = A3.Harness(stdin, stdout)
    p, checked, out = spec["branchPoint"], False, None
    while True:
        msg = h.turn()
        if msg["turn"] == "end":
            break
        point = msg["point"]
        if not checked:
            (A4 if arm == "treatment" else A3).check_contract(h.request("hello"))
            checked = True
        if point < p:
            batches = M.replay_point(h, ev, replayer, point)["batches"]
        elif point == p:
            clock = h.request("digests")
            rec = replayer.next(point, clock)  # the recorded state: the digests must be the recording's
            pset = rec["promptSettings"]
            if replayer.memory is None:
                replayer.memory = A3.Memory(pset["memoryDecisions"], pset["memoryMetricPoints"],
                                            pset["memoryMaxChars"])
            obs, omitted, catalog, mem, prompt, _ = A3.prepare(h, replayer.memory, point, pset)
            if sha256(mem) != rec.get("memorySha256"):
                raise MissingEvidence("memory_differs", "the rebuilt memory is not the recorded one")
            batches, trace = branch(h, arm, rec, obs, omitted, catalog, prompt or "", clock,
                                    Replies(spec["replies"][arm]), spec["maxRetries"], spec.get("outputs"))
            out = {"schema": BRANCH_SCHEMA, "version": VERSION, "spec": spec["id"], "specSha256": spec_sha,
                   "arm": arm, "recording": spec["recording"], "recordingSha256": spec["recordingSha256"],
                   "branchPoint": p, "prefix": {"pointsReplayed": p, "digests": clock["digests"],
                                                "memorySha256": sha256(mem), "promptSha256": sha256(prompt)},
                   "suffix": spec["suffix"], **trace}
            ev.write("branch.json", out)
        else:
            batches = []  # the declared suffix: no action
        h.decide(batches)
    if out is None:
        raise MissingEvidence("not_reached", f"the run ended before decision point {p}")
    return out


# --- the specs for the V0.9c preview points ------------------------------

V09C = "docs/benchmark/v09c-dev"
SHORT = {"financial-pressure-credit": "fp", "staffing-shortage-credit": "ss"}
ENTRY_OF = {"financial-pressure-credit": "bench-financial-pressure-ot-credit",
            "staffing-shortage-credit": "bench-staffing-shortage-ot-credit"}


def preview_points(root=ROOT):
    """Every V0.9c decision point at which a preview was shown, in a
    stable order: (recording path, record)."""
    out = []
    for p in sorted(glob.glob(os.path.join(root, V09C, "eval", "*-preview-s*-decisions.jsonl"))):
        with open(p, encoding="utf-8") as f:
            for line in f:
                rec = json.loads(line)
                if (rec.get("preview") or {}).get("status") == "ok":
                    out.append((os.path.relpath(p, root).replace(os.sep, "/"), rec))
    return out


def live_outputs(live, rsha, point):
    """{attempt id: content} of a decision point, from the live run whose
    recording is rsha (its model.jsonl, kept locally)."""
    for d in live:
        for pe in glob.glob(os.path.join(d, "runs", "*", "policy-evidence")):
            dp = os.path.join(pe, "decisions.jsonl")
            if not os.path.exists(dp):
                continue
            with open(dp, "rb") as f:
                if hashlib.sha256(f.read()).hexdigest() != rsha:
                    continue
            return {f"{c['point']}.{c['attempt']}": c["content"] for c in load_jsonl(os.path.join(pe, "model.jsonl"))
                    if c.get("point") == point and c.get("content") is not None}
    return {}


def prior_outputs(prior, rsha, point):
    """{attempt id: content} from a committed spec of this recording and
    point (prior: a prototype directory)."""
    for p in glob.glob(os.path.join(prior, "specs", "*.json")):
        with open(p, encoding="utf-8") as f:
            s = json.load(f)
        if s.get("recordingSha256") == rsha and s.get("branchPoint") == point:
            return s.get("outputs") or {}
    return {}


def make_spec(path, rec, root=ROOT, live=(), prior=None):
    """Treatment: the recorded replies after the proposal, each bound to the
    recorded input it answered and to its recorded SHA-256. Control: one
    scripted no-action reply, used only if the proposal is refused (no
    control reply from this state was ever recorded). Outputs the record
    holds only in part are copied from the live run (outputs), each checked
    against the recorded SHA-256 here and again by the policy."""
    with open(os.path.join(root, path), "rb") as f:
        rsha = hashlib.sha256(f.read()).hexdigest()
    k = proposal_attempt(rec)
    lo = live_outputs(live, rsha, rec["point"]) if live else {}
    if prior:
        lo = {**prior_outputs(prior, rsha, rec["point"]), **lo}
    outputs = {}
    for at in rec["attempts"]:
        o = at.get("output")
        if isinstance(o, dict) and not (isinstance(o.get("text"), str) and len(o["text"]) == o.get("chars")):
            if at["id"] in lo and sha256(lo[at["id"]]) == o.get("sha256"):
                outputs[at["id"]] = lo[at["id"]]
    tr = [{"source": "recorded", "from": at["id"], "answers": at["input"]["sha256"], "sha256": at["output"]["sha256"],
           "content": full_text(at, outputs)}
          for at in rec["attempts"][k:] if isinstance(at.get("output"), dict)]
    entry, seed = entry_seed(path)
    model = "qwen" if "-qwen36-35b-a3b-" in os.path.basename(path) else "gemma"
    sid = f"{SHORT[entry]}-{model}-s{seed}-p{rec['point']}"
    return {"schema": SPEC_SCHEMA, "id": sid, "version": VERSION, "recording": path,
            "recordingSha256": rsha, "branchPoint": rec["point"], "maxRetries": 1, "suffix": "no_action",
            "outputs": dict(sorted(outputs.items())),
            "replies": {"treatment": tr,
                        "control": [{"source": "scripted", "from": None, "answers": None,
                                     "content": SCRIPTED_NO_ACTION}]}}


def entry_seed(path):
    name = os.path.basename(path)
    entry = next(e for e in ENTRY_OF if name.startswith(e + "-"))
    seed = int(name.split("-s")[-1].split("-")[0])
    return entry, seed


def run_config(entry, seed, specs):
    ext = []
    for s in specs:
        for arm in ARMS:
            ext.append({"id": f"{s['id']}-{arm}", "version": "1",
                        "description": f"shared-prefix/v1 prototype, {arm} arm of {s['id']}: the recorded decisions "
                                       f"before point {s['branchPoint']} replayed and checked, the recorded proposal "
                                       f"at it, the {arm} arm's continuation with the spec's replies, then no action; "
                                       "no model is called",
                        "command": ["python3", "-I", "-B", "./analysis/repeatability/shared_prefix.py", "policy",
                                    "--spec", f"./docs/benchmark/v10a-dev/prototype/specs/{s['id']}.json",
                                    "--arm", arm],
                        "timeoutSeconds": 300})
    return {"schema": "dreamteam.benchmark-suite-run/v1", "id": f"v10a-prototype-{entry}-s{seed}",
            "description": f"V0.10a shared-prefix prototype, entry {entry}, seed {seed}: both arms of each V0.9c "
                           "preview point on this run, beside no action. Deterministic: a rerun gives the same "
                           "files. Run from the repository root.",
            "suite": "../../suite/dreamteam-dev-4.json", "entries": [entry], "seeds": [seed],
            "seedPlan": "The V0.9c run's seed, so that the recorded decisions belong to the run.", "days": 30,
            "policies": ["noop"] + [e["id"] for e in ext], "external": ext, "baseline": "noop", "output": "results",
            "evaluation": {"t90": False}}


def write_json(path, obj):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(obj, f, indent=1, ensure_ascii=False)
        f.write("\n")


def specs(out, root=ROOT, live=(), prior=None):
    groups = {}
    for path, rec in preview_points(root):
        s = make_spec(path, rec, root, live, prior)
        write_json(os.path.join(out, "specs", f"{s['id']}.json"), s)
        groups.setdefault(entry_seed(path), []).append(s)
    names = []
    for (entry, seed), ss in sorted(groups.items()):
        n = f"run-{entry}-s{seed}.json"
        write_json(os.path.join(out, n), run_config(entry, seed, ss))
        names.append(n)
    return sum(len(v) for v in groups.values()), names


# --- the comparison ------------------------------------------------------

def load_jsonl(p):
    with open(p, encoding="utf-8") as f:
        return [json.loads(l) for l in f if l.strip()]


def run_dir(runs, spec, arm):
    hits = [d for d in glob.glob(os.path.join(runs, "*", "runs", "*")) if d.endswith(f"-{spec['id']}-{arm}")]
    if len(hits) != 1:
        raise MissingEvidence("no_run", f"{spec['id']} {arm}: {len(hits)} run directories")
    return hits[0]


def digests_by_point(d):
    """The digests at each decision point of a run: the first digests
    answer after each policy turn."""
    out, want = [], False
    for line in load_jsonl(os.path.join(d, "transcript.jsonl")):
        if line.get("op") == "digests":
            want = True
            continue
        if want and "result" in line:
            dg = line["result"].get("digests")
            if not out or out[-1] != dg:
                out.append(dg)
            want = False
    return out


def contract_rows(runs):
    out = {}
    for c in glob.glob(os.path.join(runs, "*", "contract.json")):
        with open(c, encoding="utf-8") as f:
            doc = json.load(f)
        for r in doc["runs"]:
            out[r["id"]] = r
    return out


METRICS = (("financial", "operatingProfitCents"), ("operational", "examsCompleted"),
           ("staffingCapacity", "ordersLost"), ("serviceReliability", "tatBreaches"))


def metric(run, dim, name):
    return FO.metric(run.get("dimensions"), dim, name)


def compare(d, runs):
    rows = contract_rows(runs)
    out = []
    for sp in sorted(glob.glob(os.path.join(d, "specs", "*.json"))):
        spec, ssha = load_spec(sp)
        arms = {}
        for arm in ARMS:
            rd = run_dir(runs, spec, arm)
            with open(os.path.join(rd, "policy-evidence", "branch.json"), encoding="utf-8") as f:
                b = json.load(f)
            if b["specSha256"] != ssha:
                raise MissingEvidence("spec_differs", f"{spec['id']} {arm}: the run used another spec")
            run = rows.get(os.path.basename(rd)) or {}
            arms[arm] = {"branch": b, "digests": digests_by_point(rd),
                         "outcomes": {n: metric(run, dm, n) for dm, n in METRICS}}
        c, t = arms["control"], arms["treatment"]
        cb, tb = c["branch"], t["branch"]
        rec = {r["point"]: r for r in load_jsonl(os.path.join(ROOT, spec["recording"]))}[spec["branchPoint"]]
        div = next((i for i, (x, y) in enumerate(zip(c["digests"], t["digests"])) if x != y), None)
        out.append({
            "spec": spec["id"], "recording": spec["recording"], "branchPoint": spec["branchPoint"],
            "heldFixed": {"prefixDigestsEqual": cb["prefix"] == tb["prefix"],
                          "prefixIsRecorded": cb["prefix"]["digests"] == rec["digests"],
                          "proposalEqual": cb["proposalSha256"] == tb["proposalSha256"],
                          "proposalSha256": cb["proposalSha256"]},
            "differsByDesign": {"treatmentPreview": tb["preview"], "controlMessages": len(cb["messages"]),
                                "treatmentMessages": len(tb["messages"])},
            "replies": {arm: [{"source": s["source"], "parse": s["parse"], "error": (s["error"] or "").split(":", 1)[0] or None}
                              for s in arms[arm]["branch"]["steps"]] for arm in ARMS},
            "decision": {arm: {"outcome": arms[arm]["branch"]["outcome"], "final": arms[arm]["branch"]["final"],
                               "batches": arms[arm]["branch"]["batches"]} for arm in ARMS},
            "treatmentResponse": tb["response"],
            "treatmentEqualsRecording": tb["batches"] == rec["batches"],
            "sameSubmission": cb["batches"] == tb["batches"],
            "firstDivergentPoint": div,
            "runOutcomes": {arm: arms[arm]["outcomes"] for arm in ARMS},
            "runOutcomeDifference": {n: (t["outcomes"][n] - c["outcomes"][n])
                                     if isinstance(t["outcomes"][n], (int, float)) and
                                     isinstance(c["outcomes"][n], (int, float)) else None for _, n in METRICS},
        })
    return {"schema": COMPARE_SCHEMA, "version": VERSION, "points": out,
            "note": "A prototype: the control arm's replies are scripted and the suffix is fixed. Nothing here is "
                    "an effect of the preview."}


def tables(doc):
    L = ["# V0.10a shared-prefix prototype", "", doc["note"], "",
         "| Point | Prefix equal, recorded | Proposal equal | Preview | Treatment response | Control | Treatment | Same submission | Treatment = recording | First divergent point | Profit difference (treatment − control) |",
         "|---|---|---|---|---|---|---|---|---|---|---|"]
    for p in doc["points"]:
        h, dd = p["heldFixed"], p["decision"]
        pv = (p["differsByDesign"]["treatmentPreview"] or {}).get("status")
        prof = p["runOutcomeDifference"]["operatingProfitCents"]
        L.append(f"| {p['spec']} | {h['prefixDigestsEqual'] and h['prefixIsRecorded']} | {h['proposalEqual']} | "
                 f"{pv} | {p['treatmentResponse']} | {dd['control']['final']} ({dd['control']['outcome']}) | "
                 f"{dd['treatment']['final']} ({dd['treatment']['outcome']}) | {p['sameSubmission']} | "
                 f"{p['treatmentEqualsRecording']} | {p['firstDivergentPoint']} | "
                 f"{'' if prof is None else f'${prof / 100:,.2f}'} |")
    return "\n".join(L) + "\n"


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    a1 = sub.add_parser("policy")
    a1.add_argument("--spec", required=True)
    a1.add_argument("--arm", required=True, choices=ARMS)
    a1.add_argument("--evidence", default="evidence")
    a2 = sub.add_parser("specs")
    a2.add_argument("--out", required=True)
    a2.add_argument("--live", action="append", default=[], help="a live run directory, for outputs the record holds in part")
    a2.add_argument("--prior", help="a prototype directory whose specs hold those outputs")
    a3 = sub.add_parser("compare")
    a3.add_argument("--dir", required=True)
    a3.add_argument("--runs", required=True)
    a3.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    try:
        if a.cmd == "policy":
            policy(a.spec, a.arm, a.evidence)
        elif a.cmd == "specs":
            n, names = specs(a.out, live=a.live, prior=a.prior)
            print(f"shared-prefix specs: {n} branch points, {len(names)} run configurations")
        else:
            doc = compare(a.dir, a.runs)
            write_json(os.path.join(a.out, "prototype.json"), doc)
            with open(os.path.join(a.out, "prototype-tables.md"), "w", encoding="utf-8") as f:
                f.write(tables(doc))
            print(f"shared-prefix comparison: {len(doc['points'])} branch points")
        return 0
    except (MissingEvidence, A3.ReplayError, A3.ExchangeError, A3.SettingsError) as e:
        print(f"shared-prefix: {e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
