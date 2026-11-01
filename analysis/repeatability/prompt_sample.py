#!/usr/bin/env python3
"""The V0.10a prompt sample (dreamteam.repeatability-sample/v1): a small,
deterministic sample of recorded model requests from the V0.8c and V0.9c
live evaluations, with the exact request bytes, each message verified
against the SHA-256 the committed recordings hold.

    # select and extract (needs the live runs' model.jsonl, kept locally)
    python3 -I -B analysis/repeatability/prompt_sample.py --live DIR [--live DIR] --out DIR
    # check a committed sample against the committed recordings (no model, no live files)
    python3 -I -B analysis/repeatability/prompt_sample.py --check DIR

The selection reads only committed files: the recordings and the financial
evidence v1 outputs. A recorded attempt is a candidate when the model
answered it (an output is recorded). Its category, in this order:
* preview_review: the reply to an llm-agent 4 preview (input kind preview);
* validation_refusal: validate refused at least one of its actions;
* financially_consequential: financial evidence v1 records a commitment of
  at least $10,000 for one of its batches;
* staffing_proposal: it proposes a staffing action;
* no_action: it proposes no action and asks for no details;
* anything else (operations only, a details request, an unparsable reply)
  is not sampled.
Per model and category the candidates are ranked by the SHA-256 of
"dreamteam-v10a-sample/1|<recording>|<attempt id>" and the first three are
taken. Nothing in it depends on a response to this experiment.

The request is the one the agent sent, taken from the live run's
model.jsonl (never committed: it names the endpoint). Every message is
checked against the recording: the system prompt against the built-in
one, the first user message against promptSha256, each assistant message
against the earlier attempt's output SHA-256, each later user message
against that attempt's input SHA-256. The endpoint is not copied.
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
sys.path.insert(0, os.path.join(ROOT, "analysis", "evidence"))
import llm_agent3 as A3  # noqa: E402  (llm-agent 3, read-only: its constants and helpers)

SCHEMA = "dreamteam.repeatability-sample/v1"
RULE = "dreamteam-v10a-sample/1"
PER_CELL = 3
MIN_COMMITMENT_CENTS = 1_000_000
CATEGORIES = ("no_action", "staffing_proposal", "financially_consequential", "validation_refusal", "preview_review")
RECORDINGS = ("docs/benchmark/v08c-dev/eval/*-decisions.jsonl", "docs/benchmark/v09c-dev/eval/*-decisions.jsonl")
EVIDENCE = ("docs/benchmark/v09b-dev/financial-evidence.json", "docs/benchmark/v09c-dev/financial-evidence.json")
MODELS = {"qwen36-35b-a3b": "Qwen3.6-35B-A3B-oQ4e-mtp", "gemma4-26b-a4b": "gemma-4-26B-A4B-it-oQ4e-mtp"}
SYSTEM_SHA256 = "0efc36e21b592abc4acb4c8b78ee7634b39755580a0f5ac348a5be1f8a80c4e1"


class SampleError(Exception):
    pass


def sha256(b):
    return hashlib.sha256(b if isinstance(b, bytes) else b.encode("utf-8")).hexdigest()


def rel(p):
    return os.path.relpath(os.path.abspath(p), ROOT).replace(os.sep, "/")


def model_of(path):
    name = os.path.basename(path)
    for k in MODELS:
        if f"-{k}-" in name:
            return k
    raise SampleError(f"{name}: no known model in the name")


def load_jsonl(path):
    with open(path, encoding="utf-8") as f:
        return [json.loads(l) for l in f if l.strip()]


def commitments(root=ROOT):
    """{(recording, attempt id): largest commitment in cents} from the
    financial evidence v1 outputs."""
    out = {}
    for p in EVIDENCE:
        with open(os.path.join(root, p), encoding="utf-8") as f:
            doc = json.load(f)
        for r in doc["records"]:
            if r.get("dimension") != "AFFORDABILITY":
                continue
            m = r.get("measurements") or {}
            c = (m.get("engineFunding") or {}).get("commitmentCents")
            if c is None:
                c = m.get("extensionCommitmentCents")
            k = (r["recording"]["path"], (r.get("attempt") or {}).get("id"))
            if c is not None and k[1] is not None:
                out[k] = max(out.get(k, 0), c)
    return out


def category(at, path, commit):
    if not isinstance(at.get("output"), dict):
        return None
    if (at.get("input") or {}).get("kind") == "preview":
        return "preview_review"
    if at.get("refusals"):
        return "validation_refusal"
    if commit.get((path, at.get("id")), 0) >= MIN_COMMITMENT_CENTS:
        return "financially_consequential"
    if not (at.get("parse") or {}).get("ok") or at.get("detailsRequested") is not None:
        return None
    acts = at.get("actions")
    if not isinstance(acts, list):
        return None
    if any(a.get("type") in A3.STAFFING for a in acts):
        return "staffing_proposal"
    return "no_action" if not acts else None


def rank(path, aid):
    return sha256(f"{RULE}|{path}|{aid}")


def candidates(root=ROOT):
    """Every candidate attempt, in a stable order: dicts with the
    recording, its SHA-256, the attempt id, model and category."""
    commit = commitments(root)
    out = []
    for pat in RECORDINGS:
        for p in sorted(glob.glob(os.path.join(root, pat))):
            path = os.path.relpath(p, root).replace(os.sep, "/")
            with open(p, "rb") as f:
                raw = f.read()
            for rec in (json.loads(l) for l in raw.decode("utf-8").splitlines() if l.strip()):
                for at in rec.get("attempts") or []:
                    c = category(at, path, commit)
                    if c:
                        out.append({"recording": path, "recordingSha256": sha256(raw), "point": rec["point"],
                                    "attempt": at["attempt"], "attemptId": at["id"], "model": model_of(path),
                                    "category": c, "rank": rank(path, at["id"])})
    return out


def select(cands):
    chosen = []
    for m in sorted(MODELS):
        for c in CATEGORIES:
            cell = sorted((x for x in cands if x["model"] == m and x["category"] == c), key=lambda x: x["rank"])
            chosen += cell[:PER_CELL]
    for i, x in enumerate(chosen):
        x["id"] = f"p{i + 1:02d}"
    return chosen


def counts(cands, chosen):
    def tab(xs):
        return {m: {c: sum(1 for x in xs if x["model"] == m and x["category"] == c) for c in CATEGORIES}
                for m in sorted(MODELS)}
    return {"candidates": tab(cands), "selected": tab(chosen)}


def chain(rec, k):
    """The expected SHA-256 of each message of attempt k's request, from
    the record: [(role, sha256)]."""
    atts = rec["attempts"]
    want = [("system", SYSTEM_SHA256), ("user", rec["promptSha256"])]
    for j in range(1, k):
        prev, at = atts[j - 1], atts[j]
        kind = (at.get("input") or {}).get("kind")
        if kind == "resend":
            continue
        if not isinstance(prev.get("output"), dict):
            raise SampleError(f"attempt {at['id']}: follows an attempt with no output")
        want += [("assistant", prev["output"]["sha256"]), ("user", at["input"]["sha256"])]
    if want[-1][1] != atts[k - 1]["input"]["sha256"] and (atts[k - 1].get("input") or {}).get("kind") != "resend":
        raise SampleError(f"attempt {atts[k - 1]['id']}: the last message is not its input")
    return want


def verify(entry, rec):
    """Check one sample entry's request against its record."""
    k = entry["attempt"]
    at = rec["attempts"][k - 1]
    p = entry["request"]
    msgs = p["messages"]
    want = chain(rec, k)
    got = [(m["role"], sha256(m["content"])) for m in msgs]
    if got != want:
        raise SampleError(f"{entry['id']}: the request's messages differ from the recording's hashes")
    s = rec.get("promptSettings") or {}
    if p.get("model") != MODELS[entry["model"]] or p.get("temperature") != 0 or p.get("max_tokens") != 4096 \
            or p.get("response_format") != {"type": "json_object"} or set(p) != {"model", "messages", "temperature",
                                                                                 "max_tokens", "response_format"}:
        raise SampleError(f"{entry['id']}: the request's settings are not the recorded agent's")
    if entry["requestSha256"] != sha256(json.dumps(p)):
        raise SampleError(f"{entry['id']}: requestSha256 is not the request's")
    o = entry["original"]
    if sha256(o["content"]) != at["output"]["sha256"]:
        raise SampleError(f"{entry['id']}: the original output is not the recorded one")
    if not s:
        raise SampleError(f"{entry['id']}: the record has no prompt settings")
    return True


def live_runs(dirs):
    """{decisions.jsonl SHA-256: run's policy-evidence dir} over live run
    directories."""
    out = {}
    for d in dirs:
        for pe in sorted(glob.glob(os.path.join(d, "runs", "*", "policy-evidence"))):
            dp = os.path.join(pe, "decisions.jsonl")
            if os.path.exists(dp) and os.path.exists(os.path.join(pe, "model.jsonl")):
                with open(dp, "rb") as f:
                    out[sha256(f.read())] = pe
    return out


def extract(chosen, live, root=ROOT):
    runs = live_runs(live)
    entries = []
    for x in chosen:
        pe = runs.get(x["recordingSha256"])
        if pe is None:
            raise SampleError(f"{x['recording']}: no live run with this recording")
        calls = [c for c in load_jsonl(os.path.join(pe, "model.jsonl"))
                 if c.get("point") == x["point"] and c.get("attempt") == x["attempt"]]
        if len(calls) != 1:
            raise SampleError(f"{x['recording']} {x['attemptId']}: {len(calls)} recorded calls")
        c = calls[0]
        body = json.loads(c["response"])
        ch = (body.get("choices") or [{}])[0]
        e = {k: x[k] for k in ("id", "category", "model", "recording", "recordingSha256", "point", "attempt",
                               "attemptId")}
        e["request"] = c["request"]
        e["requestSha256"] = sha256(json.dumps(c["request"]))
        e["requestChars"] = sum(len(m["content"]) for m in c["request"]["messages"])
        e["messages"] = len(c["request"]["messages"])
        e["original"] = {"content": c["content"], "finishReason": ch.get("finish_reason"),
                         "usage": c.get("usage"), "latencyMs": c.get("latencyMs"),
                         "servedModel": c.get("servedModel")}
        rec = recording_point(root, x["recording"], x["point"])
        verify(e, rec)
        entries.append(e)
    return entries


def recording_point(root, path, point):
    for rec in load_jsonl(os.path.join(root, path)):
        if rec["point"] == point:
            return rec
    raise SampleError(f"{path}: no decision point {point}")


def manifest(cands, chosen, entries):
    return {"schema": SCHEMA, "rule": RULE, "perCell": PER_CELL, "minCommitmentCents": MIN_COMMITMENT_CENTS,
            "categories": list(CATEGORIES), "recordings": list(RECORDINGS), "evidence": list(EVIDENCE),
            "models": MODELS, "systemPromptSha256": SYSTEM_SHA256, "counts": counts(cands, chosen),
            "prompts": [{k: e[k] for k in ("id", "category", "model", "recording", "recordingSha256", "point",
                                           "attempt", "attemptId", "requestSha256", "requestChars", "messages")}
                        for e in entries],
            "promptsFileSha256": None}


def write(out, cands, chosen, entries):
    os.makedirs(out, exist_ok=True)
    # Not sorted: the request keeps the agent's key order, so that its bytes
    # are the ones sent (requestSha256).
    body = "".join(json.dumps(e, ensure_ascii=False) + "\n" for e in entries).encode("utf-8")
    with open(os.path.join(out, "prompts.jsonl"), "wb") as f:
        f.write(body)
    m = manifest(cands, chosen, entries)
    m["promptsFileSha256"] = sha256(body)
    with open(os.path.join(out, "manifest.json"), "w", encoding="utf-8") as f:
        json.dump(m, f, indent=1, sort_keys=True)
        f.write("\n")
    return m


def check(d, root=ROOT):
    """A committed sample: the selection made again from the committed
    files, every entry's request verified, the manifest consistent."""
    with open(os.path.join(d, "manifest.json"), encoding="utf-8") as f:
        m = json.load(f)
    with open(os.path.join(d, "prompts.jsonl"), "rb") as f:
        body = f.read()
    entries = [json.loads(l) for l in body.decode("utf-8").splitlines() if l.strip()]
    if sha256(body) != m["promptsFileSha256"]:
        raise SampleError("prompts.jsonl is not the manifest's")
    cands = candidates(root)
    chosen = select(cands)
    if [(x["id"], x["recording"], x["attemptId"]) for x in chosen] != \
            [(e["id"], e["recording"], e["attemptId"]) for e in entries]:
        raise SampleError("the selection made again differs from the sample")
    for x, e in zip(chosen, entries):
        if x["recordingSha256"] != e["recordingSha256"] or x["category"] != e["category"]:
            raise SampleError(f"{e['id']}: the recording or category differs")
        verify(e, recording_point(root, e["recording"], e["point"]))
    if manifest(cands, chosen, entries) | {"promptsFileSha256": m["promptsFileSha256"]} != m:
        raise SampleError("the manifest is not the one the sample makes")
    return len(entries)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--live", action="append", default=[], help="a live evaluation's output directory")
    ap.add_argument("--out", help="where to write manifest.json and prompts.jsonl")
    ap.add_argument("--check", help="a committed sample directory to check")
    a = ap.parse_args(argv)
    try:
        if a.check:
            n = check(a.check)
            print(f"prompt sample: {n} requests, selection and every message hash verified")
            return 0
        cands = candidates()
        chosen = select(cands)
        m = write(a.out, cands, chosen, extract(chosen, a.live))
        print(f"prompt sample: {len(m['prompts'])} requests from {len(cands)} candidates")
        return 0
    except SampleError as e:
        print(f"prompt sample: {e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
