#!/usr/bin/env python3
"""Real loopback health tests: a local HTTP server on localhost (an
ephemeral port) answers /health in each mode; the recorder reads it with the
production probe (dreamteam.health-observation/v2, exception families by
exact class), records the two-draw fixture with actual UTC timestamps and a
fresh UUID, seals it, and the independent verifier classifies and analyses
the archive with synthetic provenance refused. Recorder (sealed state),
verifier and analysis must agree. No model endpoint and no inference: model
replies are scripted.

    python3 -I -B analysis/alternatives/v13l_loopback.py --binary ./dt --scratch DIR --out FILE
"""
import argparse
import json
import os
from pathlib import Path
import platform
import shutil
import socket
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13l as J  # noqa: E402
import v13l_scenarios as S  # noqa: E402
import v13l_verify as V  # noqa: E402

SCHEMA = "dreamteam.v13l-loopback/v1"
HOST = "localhost"
TIMEOUT = 0.5
BODY = {"healthy": b'{"status":"healthy"}', "unhealthy": b'{"status":"unhealthy"}', "malformed": b'{',
        "contradictory": b'{"status":"healthy","error":"URLError"}'}


class Server:
    def __init__(self, mode):
        self.mode, self.requests = mode, 0
        outer = self

        class Handler(BaseHTTPRequestHandler):
            protocol_version = "HTTP/1.1"

            def do_GET(self):
                outer.requests += 1
                m = outer.mode
                if m == "closed_early":
                    self.close_connection = True
                    return
                if m == "bad_status_line":
                    self.wfile.write(b"NOT-HTTP garbage\r\n\r\n")
                    self.wfile.flush()
                    self.close_connection = True
                    return
                if m == "delayed":
                    time.sleep(TIMEOUT / 3)
                if m == "timeout":
                    time.sleep(TIMEOUT * 3)
                if m == "http_503":
                    body = BODY["unhealthy"]
                    self.send_response(503)
                else:
                    body = BODY.get(m, BODY["healthy"])
                    self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", "100" if m == "incomplete" else str(len(body)))
                self.end_headers()
                try:
                    self.wfile.write(body[:10] if m == "incomplete" else body)
                    self.wfile.flush()
                except (BrokenPipeError, ConnectionResetError):
                    pass
                if m == "incomplete":
                    self.close_connection = True

            def log_message(self, *a):
                pass
        self.httpd = ThreadingHTTPServer((HOST, 0), Handler)
        self.httpd.daemon_threads = True
        self.port = self.httpd.server_address[1]
        self.thread = threading.Thread(target=self.httpd.serve_forever, daemon=True)
        self.thread.start()

    def stop(self):
        self.httpd.shutdown()
        self.httpd.server_close()
        self.thread.join()


def free_port():
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind((HOST, 0))
    port = s.getsockname()[1]
    s.close()
    return port


# (name, server mode, reads before the server disappears, expected)
CASES = [
    ("healthy", "healthy", None, {"recorderState": "COMPLETE", "stopCode": None, "category": None, "completed": 2}),
    ("explicit_unhealthy", "unhealthy", None, {"recorderState": "PARTIAL", "stopCode": "health", "category": "unhealthy_status", "completed": 0}),
    ("connection_refused", "refused", None, {"recorderState": "PARTIAL", "stopCode": "health", "category": "connection_refused", "completed": 0}),
    ("connection_closed_early", "closed_early", None, {"recorderState": "PARTIAL", "stopCode": "health", "category": "remote_disconnected", "completed": 0}),
    ("incomplete_http_response", "incomplete", None, {"recorderState": "PARTIAL", "stopCode": "health", "category": "incomplete_read", "completed": 0}),
    ("delayed_response_within_timeout", "delayed", None, {"recorderState": "COMPLETE", "stopCode": None, "category": None, "completed": 2}),
    ("timeout", "timeout", None, {"recorderState": "PARTIAL", "stopCode": "health", "category": "timeout", "completed": 0}),
    ("malformed_json", "malformed", None, {"recorderState": "PARTIAL", "stopCode": "health", "category": "invalid_json", "completed": 0}),
    ("http_error_status", "http_503", None, {"recorderState": "PARTIAL", "stopCode": "health", "category": "http_status", "completed": 0}),
    ("healthy_with_error_field", "contradictory", None, {"recorderState": "PARTIAL", "stopCode": "health", "category": "contradictory_body", "completed": 0}),
    ("server_disappears_between_checks", "healthy", 2, {"recorderState": "PARTIAL", "stopCode": "health", "category": "connection_refused", "completed": 1}),
    ("bad_status_line", "bad_status_line", None, {"recorderState": "PARTIAL", "stopCode": "health", "category": "http_protocol", "completed": 0}),
]


def one(binary, scratch, case):
    name, mode, vanish_after, expected = case
    fixture = S.Fixture(binary)
    out = os.path.join(scratch, name)
    shutil.rmtree(out, ignore_errors=True)
    server = None if mode == "refused" else Server(mode)
    base = f"http://{HOST}:{server.port if server else free_port()}"
    reads = [0]

    def state():
        reads[0] += 1
        nonlocal server
        if vanish_after is not None and reads[0] == vanish_after + 1 and server is not None:
            server.stop()
            server = None
        return {"health": J.probe_health(base, timeout=TIMEOUT)}
    try:
        result = S.attempt(lambda: J.run_schedule(
            fixture.proto, fixture.proto_sha, fixture.sched, fixture.ctx, fixture.ctx.opener(binary),
            {S.D.QWEN: S.Model([S.F.ABANDON] * 6)}, out, "loopback health test", state_fn=state))
    finally:
        if server is not None:
            server.stop()
    live = Path(out) / "live"
    doc = json.loads((live / "run.json").read_text()) if (live / "run.json").exists() else {}
    seal = json.loads((live / J.SEAL).read_text()) if (live / J.SEAL).exists() else {}
    stop = doc.get("stop") or {}
    hc = stop.get("healthCheck")
    exc = doc["healthChecks"][hc]["serverState"]["health"]["exception"] if hc is not None \
        and doc["healthChecks"][hc]["serverState"] else None
    obs = {"recorder": "returned" if result["returned"] else result["error"], "recorderState": result.get("state"),
           "sealState": seal.get("terminalState"), "manifestState": doc.get("state"), "exception": exc,
           "stopCode": stop.get("code"), "stopState": stop.get("state"),
           "category": doc["healthChecks"][hc]["serverState"]["health"]["category"] if hc is not None
           and doc["healthChecks"][hc]["serverState"] else None,
           "completed": (doc.get("counters") or {}).get("drawsCompleted"), "healthReads": reads[0],
           "provenance": doc.get("provenance")}
    verdict = V.classify(out, fixture.proto, fixture.proto_sha, fixture.sched, fixture.ctx.by_id)  # synthetic refused
    obs["verifierState"] = verdict["state"]
    obs["violations"] = verdict["violations"]
    try:
        a = V.analyze(binary, {}, fixture.ctx, fixture.proto, fixture.sched, out, fixture.proto_sha)
        obs["analysis"] = {"runStatus": a["runStatus"], "completedVerified": len(a["draws"]),
                           "comparisonsSuppressed": a["armDifferences"] == [] and a["pairs"] == [],
                           "failureVerification": (a["failureVerification"] or {}).get("status")}
    except S.D.ExperimentError as x:
        obs["analysis"] = {"refused": str(x)[:300]}
    bad = [k for k, v in expected.items() if obs.get(k) != v]
    if obs["verifierState"] != obs["recorderState"] or obs["sealState"] != obs["recorderState"]:
        bad.append("recorder, seal and verifier disagree")
    if obs["analysis"].get("runStatus") != obs["recorderState"]:
        bad.append("analysis disagrees with the recorder")
    shutil.rmtree(out, ignore_errors=True)
    return {"name": name, "serverMode": mode, "expected": expected, "observed": obs, "agree": not bad, "mismatch": bad}


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    os.makedirs(a.scratch, exist_ok=True)
    results = [one(os.path.abspath(a.binary), os.path.abspath(a.scratch), c) for c in CASES]
    S.D.write_json(a.out, {"schema": SCHEMA, "host": f"{platform.system().lower()}/{platform.machine()}",
                           "python": platform.python_version(), "server": "http.server on localhost, ephemeral port",
                           "notice": "Loopback only; no model endpoint; scripted model replies; actual timestamps and UUIDs.",
                           "cases": results, "summary": {"cases": len(results), "agree": sum(r["agree"] for r in results)}})
    print(f"loopback: {sum(r['agree'] for r in results)} of {len(results)} agree")
    for r in results:
        if not r["agree"]:
            print("DISAGREE", r["name"], r["mismatch"], json.dumps(r["observed"], sort_keys=True))
    if not all(r["agree"] for r in results):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
