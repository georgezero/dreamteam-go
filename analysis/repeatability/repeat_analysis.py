#!/usr/bin/env python3
"""The V0.10a repeatability analysis (dreamteam.repeatability-analysis/v1):
the live run's responses compared at five levels (PLAN.md §4).

    python3 -I -B analysis/repeatability/repeat_analysis.py --binary ./dt \
        --dir docs/benchmark/v10a-dev --replays DIR --out DIR

* A. raw: the content's bytes;
* B. parsed: the content parsed as llm-agent 3 parses it, canonical JSON;
* C. actions: llm-agent 3's strict reading of the reply (its actions, a
  details request, or the error code it gives), canonical JSON;
* D. validation: each distinct action list split into llm-agent 3's
  batches and validated by the engine at the recorded decision point, on
  a replay of the recording (`--replays`: the recorded-decision replays CI
  makes); the outcome is each batch's valid flag and refusal codes;
* E. decision: no_action, actions, refused, details or unparsable.

Every response is compared with the prompt's first response in the same
condition and with the original recorded response. Timings are reported
apart, as live metadata: nothing else depends on them. The output is a
function of the committed files and the engine.
"""

import argparse
import collections
import glob
import hashlib
import json
import os
import statistics
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, os.path.join(ROOT, "agents", "llm"))
sys.path.insert(0, HERE)
import llm_agent3 as A3  # noqa: E402  (llm-agent 3, read-only: its parser and batching)
import prompt_sample as S  # noqa: E402

SCHEMA = "dreamteam.repeatability-analysis/v1"
ANALYZER = "repeatability-analysis 1"
LEVELS = ("A_raw", "B_parsed", "C_actions", "D_validation", "E_decision")


class AnalysisError(Exception):
    pass


def canon(x):
    return json.dumps(x, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def sha(s):
    return hashlib.sha256(s.encode("utf-8")).hexdigest()


def load_jsonl(path):
    with open(path, encoding="utf-8") as f:
        return [json.loads(l) for l in f if l.strip()]


# --- reading a reply -----------------------------------------------------

def details_allowed(rec, k):
    """Whether llm-agent 3 (or 4) would accept a details request as the
    reply to attempt k: only before a details turn or a preview at the
    point."""
    for at in rec["attempts"][:k]:
        if (at.get("input") or {}).get("kind") in ("details", "preview"):
            return False
    return bool((rec.get("promptSettings") or {}).get("allowDetails"))


def parsed(content):
    """B: the reply as one JSON object, canonical; or an unparsable mark."""
    if content is None:
        return "no_content"
    try:
        d = json.loads(A3.strip_wrapping(content))
    except ValueError:
        return "unparsable"
    return canon(d) if isinstance(d, dict) else "not_an_object"


def reading(content, types, allow):
    """C: llm-agent 3's strict reading: {"actions": [...]}, {"details":
    [...]} or {"error": code}."""
    if content is None:
        return {"error": "no_content"}
    try:
        d = A3.parse_decision(content, types, allow)
    except ValueError as e:
        return {"error": str(e).split(":", 1)[0]}
    if "details" in d:
        return {"details": d["details"]}
    return {"actions": d["actions"]}


def batches(actions):
    return [b for b, _ in A3.split(actions)]


def outcome(answers):
    """D: per batch, valid and the refused results' codes."""
    out = []
    for v in answers:
        if v.get("protocolError"):
            out.append({"protocolError": v["protocolError"]})
            continue
        out.append({"valid": bool(v.get("valid")),
                    "refused": sorted(r.get("code") for r in v.get("results") or [] if not r.get("ok"))})
    return out


def decision(read, val):
    if "error" in read:
        return "unparsable"
    if "details" in read:
        return "details"
    if not read["actions"]:
        return "no_action"
    return "actions" if all(b.get("valid") for b in val) else "refused"


# --- the engine at a recorded decision point ---------------------------

def replay_runs(replays):
    """{recording basename: replay run directory}, from each replay run's
    own record of the recording it replayed."""
    out = {}
    for ev in sorted(glob.glob(os.path.join(replays, "*", "runs", "*", "policy-evidence", "decisions.jsonl"))):
        with open(ev, encoding="utf-8") as f:
            first = f.readline()
        rb = json.loads(first).get("replayed") if first.strip() else None
        if rb:
            out[rb] = os.path.dirname(os.path.dirname(ev))
    return out


def validate_at(binary, run_dir, targets):
    """Feed the replay's transcript to the engine, and right after the first
    digests answer equal to a target's digests (the decision point's start,
    before any batch of it is applied) ask the catalog and validate each of
    the target's batches. Every original answer must come back identical.
    targets: {digests: [batch]}. Returns {digests: {"types": [...],
    "answers": [answer per batch]}}."""
    with open(os.path.join(run_dir, "run.json"), encoding="utf-8") as f:
        run = json.load(f)
    pairs, req = [], None
    with open(os.path.join(run_dir, "transcript.jsonl"), encoding="utf-8") as f:
        for line in f:
            d = json.loads(line)
            if "op" in d:
                req = d
            elif req is not None:
                pairs.append((req, d))
                req = None
    scratch = tempfile.TemporaryDirectory(prefix="v10a-validate-")
    proc = subprocess.Popen([os.path.abspath(binary), "agent", "--stdio", "-scenario", run["scenario"]["ref"],
                             "-seed", str(run["seed"])], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL, text=True, encoding="utf-8", cwd=scratch.name)

    def ask(r):
        proc.stdin.write(json.dumps(r) + "\n")
        proc.stdin.flush()
        line = proc.stdout.readline()
        if not line:
            raise AnalysisError("the engine closed the session")
        return json.loads(line)
    out, left = {}, dict(targets)
    try:
        for n, (r, want) in enumerate(pairs, 1):
            if not left:
                break
            got = ask(r)
            if r.get("op") == "hello":
                same = ((got.get("result") or {}).get("clock") == (want.get("result") or {}).get("clock") and
                        ((got.get("result") or {}).get("scenario") or {}).get("fingerprint") ==
                        ((want.get("result") or {}).get("scenario") or {}).get("fingerprint"))
            else:
                same = got == want
            if not same:
                raise AnalysisError(f"{run['id']}: request {n} ({r.get('op')}) answered differently")
            dg = (got.get("result") or {}).get("digests") if r.get("op") == "digests" and got.get("ok") else None
            key = canon(dg) if dg is not None else None
            if key in left:
                cat = ask({"id": "v10a-actions", "op": "actions"})
                types = sorted(t.get("type") for t in (cat.get("result") or {}).get("types") or [])
                answers = []
                for b in left.pop(key):
                    v = ask({"id": "v10a-validate", "op": "validate", "args": {"actions": b}})
                    answers.append(v.get("result") if v.get("ok") else
                                   {"protocolError": (v.get("error") or {}).get("code")})
                out[key] = {"types": types, "answers": answers}
    finally:
        proc.stdin.close()
        proc.wait()
        scratch.cleanup()
    if left:
        raise AnalysisError(f"{run['id']}: {len(left)} decision points not reached in the transcript")
    return out


# --- the analysis --------------------------------------------------------

def analyze(binary, d, replays):
    with open(os.path.join(d, "protocol.json"), "rb") as f:
        proto_raw = f.read()
    proto = json.loads(proto_raw)
    sdir = os.path.join(d, proto["sample"])
    with open(os.path.join(sdir, "manifest.json"), encoding="utf-8") as f:
        man = json.load(f)
    prompts = load_jsonl(os.path.join(sdir, "prompts.jsonl"))
    with open(os.path.join(d, "live", "run.json"), encoding="utf-8") as f:
        run = json.load(f)
    rows = load_jsonl(os.path.join(d, "live", "requests.jsonl"))
    if run["protocolSha256"] != S.sha256(proto_raw) or run["promptsFileSha256"] != man["promptsFileSha256"]:
        raise AnalysisError("the live run is not of this protocol and sample")
    recs = {p["id"]: S.recording_point(ROOT, p["recording"], p["point"]) for p in prompts}
    runs = replay_runs(replays)

    # The engine's catalog and validation answers, one engine replay per run.
    by_run = collections.defaultdict(dict)
    pending = collections.defaultdict(list)
    contents = collections.defaultdict(list)
    for p in prompts:
        contents[p["id"]].append(p["original"]["content"])
    for r in rows:
        contents[r["prompt"]].append(r.get("content"))
    # First pass: the catalog only (an empty target list), then every
    # distinct action list read with it.
    for p in prompts:
        rec = recs[p["id"]]
        base = os.path.basename(p["recording"])
        if base not in runs:
            raise AnalysisError(f"{p['id']}: no replay of {base} in {replays}")
        by_run[base].setdefault(canon(rec["digests"]), [])
    cats = {}
    for base, targets in sorted(by_run.items()):
        got = validate_at(binary, runs[base], {k: [] for k in targets})
        for k, v in got.items():
            cats[(base, k)] = v["types"]
    readings = {}
    for p in prompts:
        rec, base = recs[p["id"]], os.path.basename(p["recording"])
        types = set(cats[(base, canon(rec["digests"]))])
        allow = details_allowed(rec, p["attempt"])
        for c in contents[p["id"]]:
            rd = reading(c, types, allow)
            readings[(p["id"], c)] = rd
            if "actions" in rd and rd["actions"]:
                for b in batches(rd["actions"]):
                    if b not in pending[(base, canon(rec["digests"]))]:
                        pending[(base, canon(rec["digests"]))].append(b)
    answers = {}
    for base in sorted({b for b, _ in pending}):
        targets = {k: v for (b, k), v in pending.items() if b == base}
        got = validate_at(binary, runs[base], targets)
        for k, v in got.items():
            for b, a in zip(targets[k], v["answers"]):
                answers[(base, k, canon(b))] = a

    def levels(p, c):
        rec, base = recs[p["id"]], os.path.basename(p["recording"])
        rd = readings[(p["id"], c)]
        val = outcome([answers[(base, canon(rec["digests"]), canon(b))] for b in batches(rd["actions"])]) \
            if rd.get("actions") else []
        return {"A_raw": c if c is not None else "\0no_content", "B_parsed": parsed(c), "C_actions": canon(rd),
                "D_validation": canon(val), "E_decision": decision(rd, val)}

    out_prompts = []
    for p in prompts:
        orig = levels(p, p["original"]["content"])
        prow = [r for r in rows if r["prompt"] == p["id"]]
        conds = {}
        for cnd in proto["conditions"]:
            rs = sorted((r for r in prow if r["condition"] == cnd["id"]), key=lambda r: r["seq"])
            answered = [r for r in rs if r.get("content") is not None]
            lv = [levels(p, r["content"]) for r in answered]
            entry = {"requests": len(rs), "answered": len(answered),
                     "transportFailures": sum(1 for r in rs if r.get("status") is None),
                     "httpErrors": sum(1 for r in rs if r.get("status") is not None and r.get("error")),
                     "finishReasons": dict(sorted(collections.Counter(str(r.get("finishReason"))
                                                                      for r in answered).items())),
                     "levels": {}}
            for L in LEVELS:
                vals = [x[L] for x in lv]
                entry["levels"][L] = {
                    "distinct": len(set(vals)),
                    "allIdentical": len(set(vals)) == 1 if vals else None,
                    "sameAsFirst": sum(1 for v in vals if v == vals[0]) if vals else 0,
                    "sameAsOriginal": sum(1 for v in vals if v == orig[L]),
                }
            entry["decisions"] = dict(sorted(collections.Counter(x["E_decision"] for x in lv).items()))
            conds[cnd["id"]] = entry
        every = [levels(p, r["content"]) for r in prow if r.get("content") is not None]
        allc = {L: len(set(x[L] for x in every)) for L in LEVELS}
        first = min(prow, key=lambda r: r["seq"]) if prow else None
        out_prompts.append({
            "id": p["id"], "model": p["model"], "category": p["category"], "recording": p["recording"],
            "attemptId": p["attemptId"], "requestSha256": p["requestSha256"], "original": orig,
            "conditions": conds, "allConditionsDistinct": allc,
            "allConditionsDecisions": dict(sorted(collections.Counter(x["E_decision"] for x in every).items())),
            "firstInStudy": {"seq": first["seq"], "condition": first["condition"],
                             "cachedTokens": cached(first), "promptTokens": (first.get("usage") or {}).get("prompt_tokens"),
                             "sameAsOriginal": {L: levels(p, first.get("content"))[L] == orig[L] for L in LEVELS}}
            if first else None,
            "distinctActionLists": sorted({x["C_actions"] for x in every} | {orig["C_actions"]}),
        })
    return {"schema": SCHEMA, "analyzer": ANALYZER, "protocol": proto["id"], "protocolSha256": run["protocolSha256"],
            "promptsFileSha256": man["promptsFileSha256"], "requestsFileSha256": file_sha(os.path.join(d, "live", "requests.jsonl")),
            "run": {k: run[k] for k in ("requestsScheduled", "requestsSent", "stopped")},
            "levels": list(LEVELS), "prompts": out_prompts, "summary": summarize(out_prompts, proto),
            "exploratory": exploratory(rows, prompts, proto), "inference": inference(rows, proto)}


def exploratory(rows, prompts, proto):
    """Not declared in the plan, reported as exploratory: raw-text
    agreement of the two requests sent at once; of sequential responses
    with and without reported cached tokens; and of the seeded and back-
    to-back sequences."""
    out = {"note": "exploratory: not a measure declared in PLAN.md", "concurrentPairs": {}, "sequentialByCache": {},
           "seededEqualsBackToBack": {}}
    for m in proto["modelOrder"]:
        ps = [p for p in prompts if p["model"] == m]
        same = total = 0
        cache = collections.Counter()
        seq_same = 0
        for p in ps:
            rs = [r for r in rows if r["prompt"] == p["id"]]
            for rd in sorted({r["round"] for r in rs if r["condition"] == "concurrent"}):
                pair = [r.get("content") for r in rs if r["condition"] == "concurrent" and r["round"] == rd]
                total += 1
                same += len(pair) == 2 and pair[0] == pair[1]
            seq = sorted((r for r in rs if r["condition"] != "concurrent"), key=lambda r: r["seq"])
            mode = collections.Counter(r.get("content") for r in seq).most_common(1)[0][0] if seq else None
            for r in seq:
                k = "noCachedTokens" if not cached(r) else "cachedTokens"
                cache[(k, r.get("content") == mode)] += 1
            a = [r.get("content") for r in sorted((r for r in rs if r["condition"] == "seeded"), key=lambda r: r["seq"])]
            b = [r.get("content") for r in sorted((r for r in rs if r["condition"] == "back_to_back"),
                                                  key=lambda r: r["seq"])]
            seq_same += a == b
        out["concurrentPairs"][m] = {"pairs": total, "identicalRaw": same}
        out["sequentialByCache"][m] = {k: {"requests": cache[(k, True)] + cache[(k, False)],
                                           "sameAsPromptsMostCommon": cache[(k, True)]}
                                       for k in ("cachedTokens", "noCachedTokens")}
        out["seededEqualsBackToBack"][m] = {"prompts": len(ps), "identicalSequences": seq_same}
    return out


def file_sha(p):
    with open(p, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def cached(r):
    return ((r.get("usage") or {}).get("prompt_tokens_details") or {}).get("cached_tokens")


def summarize(ps, proto):
    """Per model and condition, and per category and condition: prompts
    whose responses are all identical at each level, with the
    denominator."""
    out = {"byModel": {}, "byCategory": {}, "allConditions": {}}
    for key, group in (("byModel", "model"), ("byCategory", "category")):
        for g in sorted({p[group] for p in ps}):
            sub = [p for p in ps if p[group] == g]
            out[key][g] = {}
            for c in proto["conditions"]:
                es = [p["conditions"][c["id"]] for p in sub if p["conditions"][c["id"]]["answered"]]
                row = {"prompts": len(es), "requests": sum(e["requests"] for e in es),
                       "answered": sum(e["answered"] for e in es)}
                for L in LEVELS:
                    row[L] = {"allIdentical": sum(1 for e in es if e["levels"][L]["allIdentical"]),
                              "sameAsFirst": sum(e["levels"][L]["sameAsFirst"] for e in es),
                              "sameAsOriginal": sum(e["levels"][L]["sameAsOriginal"] for e in es)}
                out[key][g][c["id"]] = row
            out["allConditions"].setdefault(key, {})[g] = {
                "prompts": len(sub),
                **{L: {"oneValue": sum(1 for p in sub if p["allConditionsDistinct"][L] == 1)} for L in LEVELS}}
    return out


def inference(rows, proto):
    """Live metadata: tokens, cached tokens and latency, per model and
    condition. Reported, not compared."""
    out = {}
    for m in proto["modelOrder"]:
        out[m] = {}
        for c in proto["conditions"]:
            rs = [r for r in rows if r["model"] == m and r["condition"] == c["id"] and r.get("usage")]
            lat = [r["latencyMs"] for r in rs]
            ct = [cached(r) for r in rs if cached(r) is not None]
            pt = [(r["usage"] or {}).get("prompt_tokens") for r in rs]
            out[m][c["id"]] = {
                "answered": len(rs),
                "latencyMs": {"median": statistics.median(lat) if lat else None, "max": max(lat) if lat else None},
                "completionTokens": {"median": statistics.median([(r["usage"] or {}).get("completion_tokens") or 0
                                                                 for r in rs]) if rs else None},
                "promptTokens": {"total": sum(x or 0 for x in pt)},
                "cachedTokens": {"total": sum(ct), "requestsWithNone": sum(1 for x in ct if x == 0)},
            }
    return out


# --- tables --------------------------------------------------------------

NAMES = {"A_raw": "raw", "B_parsed": "parsed", "C_actions": "actions", "D_validation": "validation",
         "E_decision": "decision"}


def tables(doc):
    lines = ["# V0.10a repeatability tables", "",
             f"Generated by `{doc['analyzer']}` from `live/requests.jsonl` ({doc['run']['requestsSent']} of "
             f"{doc['run']['requestsScheduled']} requests). Counts, each with its denominator; no rate is "
             "generalized beyond these prompts.", ""]
    s = doc["summary"]
    lines += ["## Prompts whose responses were all identical, by model and condition", "",
              "| Model | Condition | Prompts | Requests | " + " | ".join(NAMES[L] for L in LEVELS) + " |",
              "|---|---|---|---|" + "---|" * len(LEVELS)]
    for m, cs in s["byModel"].items():
        for c, row in cs.items():
            lines.append(f"| {m} | {c} | {row['prompts']} | {row['answered']} | " +
                         " | ".join(f"{row[L]['allIdentical']} of {row['prompts']}" for L in LEVELS) + " |")
    lines += ["", "## Requests equal to the original recorded response, by model and condition", "",
              "| Model | Condition | Requests | " + " | ".join(NAMES[L] for L in LEVELS) + " |",
              "|---|---|---|" + "---|" * len(LEVELS)]
    for m, cs in s["byModel"].items():
        for c, row in cs.items():
            lines.append(f"| {m} | {c} | {row['answered']} | " +
                         " | ".join(f"{row[L]['sameAsOriginal']} of {row['answered']}" for L in LEVELS) + " |")
    lines += ["", "## Prompts with one value across all 15 requests (every condition)", "",
              "| Group | Prompts | " + " | ".join(NAMES[L] for L in LEVELS) + " |", "|---|---|" + "---|" * len(LEVELS)]
    for key in ("byModel", "byCategory"):
        for g, row in s["allConditions"][key].items():
            lines.append(f"| {g} | {row['prompts']} | " +
                         " | ".join(f"{row[L]['oneValue']}" for L in LEVELS) + " |")
    lines += ["", "## Each prompt: distinct values across all conditions, and the decisions", "",
              "| Prompt | Model | Category | " + " | ".join(NAMES[L] for L in LEVELS) +
              " | Decisions (all requests) | Original decision |", "|---|---|---|" + "---|" * len(LEVELS) + "---|---|"]
    for p in doc["prompts"]:
        dec = ", ".join(f"{k} {v}" for k, v in p["allConditionsDecisions"].items())
        lines.append(f"| {p['id']} | {p['model']} | {p['category']} | " +
                     " | ".join(str(p["allConditionsDistinct"][L]) for L in LEVELS) +
                     f" | {dec} | {p['original']['E_decision']} |")
    x = doc["exploratory"]
    lines += ["", "## Exploratory (not declared in the plan)", "",
              "| Model | Concurrent pairs with identical raw text | Sequential requests with cached tokens: equal to the prompt's most common text | Without cached tokens: equal | Prompts whose seeded and back-to-back sequences are identical |",
              "|---|---|---|---|---|"]
    for m in x["concurrentPairs"]:
        c, q, sd = x["concurrentPairs"][m], x["sequentialByCache"][m], x["seededEqualsBackToBack"][m]
        lines.append(f"| {m} | {c['identicalRaw']} of {c['pairs']} | "
                     f"{q['cachedTokens']['sameAsPromptsMostCommon']} of {q['cachedTokens']['requests']} | "
                     f"{q['noCachedTokens']['sameAsPromptsMostCommon']} of {q['noCachedTokens']['requests']} | "
                     f"{sd['identicalSequences']} of {sd['prompts']} |")
    lines += ["", "## Inference (live metadata, not compared)", "",
              "| Model | Condition | Answered | Median latency (s) | Max latency (s) | Median completion tokens | Prompt tokens | Cached tokens | Requests with no cached tokens |",
              "|---|---|---|---|---|---|---|---|---|"]
    for m, cs in doc["inference"].items():
        for c, r in cs.items():
            med = r["latencyMs"]["median"]
            mx = r["latencyMs"]["max"]
            lines.append(f"| {m} | {c} | {r['answered']} | {med / 1000:.1f} | {mx / 1000:.1f} | "
                         f"{r['completionTokens']['median']} | {r['promptTokens']['total']:,} | "
                         f"{r['cachedTokens']['total']:,} | {r['cachedTokens']['requestsWithNone']} |"
                         if med is not None else f"| {m} | {c} | 0 | | | | | | |")
    return "\n".join(lines) + "\n"


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--dir", required=True)
    ap.add_argument("--replays", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    try:
        doc = analyze(a.binary, a.dir, a.replays)
    except AnalysisError as e:
        print(f"repeatability analysis: {e}", file=sys.stderr)
        return 1
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, "repeatability.json"), "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True, ensure_ascii=False)
        f.write("\n")
    with open(os.path.join(a.out, "repeatability-tables.md"), "w", encoding="utf-8") as f:
        f.write(tables(doc))
    print(f"repeatability analysis: {len(doc['prompts'])} prompts, {doc['run']['requestsSent']} requests")
    return 0


if __name__ == "__main__":
    sys.exit(main())
