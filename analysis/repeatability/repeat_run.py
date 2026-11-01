#!/usr/bin/env python3
"""The V0.10a live repeatability run (dreamteam.repeatability-protocol/v1):
the sample's recorded requests sent again, unchanged, under the protocol's
declared conditions, every response recorded.

    python3 -I -B analysis/repeatability/repeat_run.py --protocol docs/benchmark/v10a-dev/protocol.json \
        --schedule-only --out DIR          # the schedule, no request
    DREAMTEAM_LLM_BASE_URL=... python3 -I -B analysis/repeatability/repeat_run.py \
        --protocol docs/benchmark/v10a-dev/protocol.json --out DIR

The schedule (schedule.json) is a function of the protocol and the sample
alone, and is committed with them before any request. A live run writes,
in DIR/live/: requests.jsonl (one line per request, written as it ends),
server-before.json, server-after.json and run.json. The endpoint's address
is never written. No request is retried, and nothing is selected or
dropped after the fact: a failed request is recorded as failed.
"""

import argparse
import datetime
import hashlib
import json
import os
import sys
import threading
import time
import urllib.error
import urllib.request

SCHEMA = "dreamteam.repeatability-run/v1"
SCHEDULE_SCHEMA = "dreamteam.repeatability-schedule/v1"


class RunError(Exception):
    pass


def sha256(b):
    return hashlib.sha256(b if isinstance(b, bytes) else b.encode("utf-8")).hexdigest()


def load(protocol_path):
    with open(protocol_path, "rb") as f:
        raw = f.read()
    proto = json.loads(raw)
    d = os.path.join(os.path.dirname(os.path.abspath(protocol_path)), proto["sample"])
    with open(os.path.join(d, "manifest.json"), encoding="utf-8") as f:
        man = json.load(f)
    with open(os.path.join(d, "prompts.jsonl"), "rb") as f:
        body = f.read()
    if sha256(body) != man["promptsFileSha256"]:
        raise RunError("prompts.jsonl is not the manifest's")
    prompts = [json.loads(l) for l in body.decode("utf-8").splitlines() if l.strip()]
    return proto, sha256(raw), man, prompts


def order_key(rule, *parts):
    return sha256("|".join([rule, *map(str, parts)]))


def schedule(proto, prompts):
    """The requests in the order they are sent: a list of steps, each a
    list of requests sent at once (one, or the concurrency)."""
    rule, steps = proto["orderRule"], []
    for m in proto["modelOrder"]:
        ps = [p for p in prompts if p["model"] == m]
        for c in proto["conditions"]:
            cid = c["id"]
            if cid == "interleaved":
                for r in range(1, c["rounds"] + 1):
                    for p in sorted(ps, key=lambda p: order_key(rule, cid, r, p["id"])):
                        steps.append([(m, cid, p["id"], r, 1)])
            elif c["concurrency"] > 1:
                for p in sorted(ps, key=lambda p: order_key(rule, cid, p["id"])):
                    for r in range(1, c["rounds"] + 1):
                        steps.append([(m, cid, p["id"], r, k) for k in range(1, c["concurrency"] + 1)])
            else:
                for p in sorted(ps, key=lambda p: order_key(rule, cid, p["id"])):
                    for r in range(1, c["repeats"] + 1):
                        steps.append([(m, cid, p["id"], r, 1)])
    out, n = [], 0
    for i, st in enumerate(steps, 1):
        row = []
        for (m, cid, pid, r, k) in st:
            n += 1
            row.append({"seq": n, "step": i, "model": m, "condition": cid, "prompt": pid, "round": r, "slot": k})
        out.append(row)
    return out


def schedule_doc(proto, proto_sha, man, sched):
    return {"schema": SCHEDULE_SCHEMA, "protocol": proto["id"], "protocolSha256": proto_sha,
            "promptsFileSha256": man["promptsFileSha256"], "steps": len(sched),
            "requests": sum(len(s) for s in sched), "schedule": sched}


def body_of(prompt, cond):
    req = dict(prompt["request"])
    if cond.get("seed") is not None:
        req["seed"] = cond["seed"]
    return json.dumps(req).encode("utf-8")


def post(base, data, timeout):
    req = urllib.request.Request(base.rstrip("/") + "/chat/completions", data=data,
                                 headers={"Content-Type": "application/json"}, method="POST")
    t0 = time.monotonic()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.status, r.read().decode("utf-8", "replace"), None, time.monotonic() - t0
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", "replace"), f"http_status: HTTP {e.code}", time.monotonic() - t0
    except Exception as e:  # noqa: BLE001  (a transport failure of any kind is recorded, not raised)
        name = "timeout" if "timed out" in str(e) else "connection"
        return None, "", f"{name}: {type(e).__name__}", time.monotonic() - t0


def get(base, path, timeout=20):
    root = base.rstrip("/")
    if root.endswith("/v1"):
        root = root[:-3]
    try:
        with urllib.request.urlopen(root + path, timeout=timeout) as r:
            return json.loads(r.read().decode("utf-8"))
    except Exception as e:  # noqa: BLE001
        return {"error": f"{type(e).__name__}"}


def server_state(base):
    """What the public endpoints say about the server (no admin call)."""
    api = get(base, "/openapi.json")
    models = get(base, "/v1/models")
    return {"time": utc(), "health": get(base, "/health"),
            "serverVersion": (api.get("info") or {}).get("version"), "serverTitle": (api.get("info") or {}).get("title"),
            "openapiSha256": sha256(json.dumps(api, sort_keys=True)) if "error" not in api else None,
            "models": [{"id": m.get("id"), "ownedBy": m.get("owned_by"), "maxModelLen": m.get("max_model_len")}
                       for m in models.get("data") or []]}


def utc():
    return datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ")


def response_fields(body):
    try:
        d = json.loads(body)
    except ValueError:
        return {"parsedBody": False}
    ch = (d.get("choices") or [{}])[0]
    msg = ch.get("message") or {}
    return {"parsedBody": True, "content": msg.get("content"), "reasoning": msg.get("reasoning_content") or msg.get("reasoning"),
            "finishReason": ch.get("finish_reason"), "usage": d.get("usage"), "servedModel": d.get("model"),
            "systemFingerprint": d.get("system_fingerprint"), "responseId": d.get("id")}


def run(proto, proto_sha, man, prompts, out, base, post_fn=post, state_fn=server_state, now=time.monotonic):
    sched = schedule(proto, prompts)
    by_id = {p["id"]: p for p in prompts}
    conds = {c["id"]: c for c in proto["conditions"]}
    live = os.path.join(out, "live")
    os.makedirs(live, exist_ok=True)
    stop = proto["stop"]
    before = state_fn(base)
    with open(os.path.join(live, "server-before.json"), "w", encoding="utf-8") as f:
        json.dump(before, f, indent=1, sort_keys=True)
        f.write("\n")
    t_start, started, fails, n_done, stopped = now(), utc(), 0, 0, None
    block, health = None, []
    with open(os.path.join(live, "requests.jsonl"), "w", encoding="utf-8") as f:
        for st in sched:
            b = (st[0]["model"], st[0]["condition"])
            if b != block:
                block = b
                h = state_fn(base)["health"]
                health.append({"model": b[0], "condition": b[1], "time": utc(), "health": h})
                if (h or {}).get("status") != "healthy":
                    stopped = f"health: the server did not answer healthy before {b[0]} {b[1]}"
                    break
            if now() - t_start > stop["wallClockSeconds"]:
                stopped = "wall clock"
                break
            results = [None] * len(st)

            def one(i, r):
                p, c = by_id[r["prompt"]], conds[r["condition"]]
                data = body_of(p, c)
                t = utc()
                status, body, err, lat = post_fn(base, data, proto["request"]["timeoutSeconds"])
                results[i] = dict(r, sent=t, requestSha256=sha256(data), status=status, error=err,
                                  latencyMs=round(lat * 1000), body=body, **response_fields(body))
            if len(st) == 1:
                one(0, st[0])
            else:
                ts = [threading.Thread(target=one, args=(i, r)) for i, r in enumerate(st)]
                for t in ts:
                    t.start()
                for t in ts:
                    t.join()
            for res in results:
                f.write(json.dumps(res, sort_keys=True, ensure_ascii=False) + "\n")
                f.flush()
                n_done += 1
                fails = fails + 1 if res["error"] and res["status"] is None else 0
            if fails >= stop["consecutiveTransportFailures"]:
                stopped = f"{fails} consecutive transport failures"
                break
    after = state_fn(base)
    with open(os.path.join(live, "server-after.json"), "w", encoding="utf-8") as f:
        json.dump(after, f, indent=1, sort_keys=True)
        f.write("\n")
    doc = {"schema": SCHEMA, "protocol": proto["id"], "protocolSha256": proto_sha,
           "promptsFileSha256": man["promptsFileSha256"], "started": started, "ended": utc(),
           "requestsScheduled": sum(len(s) for s in sched), "requestsSent": n_done, "stopped": stopped,
           "healthBeforeBlocks": health, "endpoint": "from " + proto["endpointEnv"] + " (not recorded)",
           "client": {"python": sys.version.split()[0]}}
    with open(os.path.join(live, "run.json"), "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=1, sort_keys=True)
        f.write("\n")
    return doc


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--protocol", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--schedule-only", action="store_true")
    a = ap.parse_args(argv)
    try:
        proto, proto_sha, man, prompts = load(a.protocol)
        sched = schedule(proto, prompts)
        os.makedirs(a.out, exist_ok=True)
        with open(os.path.join(a.out, "schedule.json"), "w", encoding="utf-8") as f:
            json.dump(schedule_doc(proto, proto_sha, man, sched), f, indent=1, sort_keys=True)
            f.write("\n")
        if a.schedule_only:
            print(f"schedule: {sum(len(s) for s in sched)} requests in {len(sched)} steps")
            return 0
        base = os.environ.get(proto["endpointEnv"])
        if not base:
            raise RunError(f"set {proto['endpointEnv']}")
        doc = run(proto, proto_sha, man, prompts, a.out, base)
        print(f"repeatability run: {doc['requestsSent']} of {doc['requestsScheduled']} requests"
              + (f"; stopped: {doc['stopped']}" if doc["stopped"] else ""))
        return 0 if not doc["stopped"] else 1
    except RunError as e:
        print(f"repeatability run: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
