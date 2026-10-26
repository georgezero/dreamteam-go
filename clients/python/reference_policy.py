#!/usr/bin/env python3
"""The reference external benchmark policy (python-ref, version 1).

A DreamTeam benchmark policy run as a separate process by `dreamteam
benchmark run` (docs/BENCHMARK.md §4). It speaks the policy protocol
(dreamteam.benchmark-policy/v1) on its standard input and output: at each
decision point the harness writes a turn; the policy sends agent/v1
requests (each answered by one response line) and ends the turn with its
decision. After the last turn it exits with status 0. Standard library
only; it uses nothing but the protocol.

Its behaviour is deliberately simple, a weekly review, fixed before any
benchmark run and not tuned to outcomes:

  * It reviews at decision points 0, 7, 14, ... and does nothing between.
  * At a review, if the oldest unfinished study is more than 24 hours old,
    it proposes raising one radiologist's site reading allocation by 5
    points (at most the maximum), from the earliest lawful date: the
    editable radiologist with the lowest allocation in effect, room to
    grow and no pending change (ties: the catalog's order).
  * It validates each proposal first. A refused proposal is passed over
    for the next candidate; it submits at most one change, and only a
    valid one.
  * A protocol error during a turn ends that turn with no change (noted on
    standard error); a broken exchange ends the process with status 1.
"""

import json
import sys

POLICY_PROTOCOL = "dreamteam.benchmark-policy/v1"
AGENT_PROTOCOL = "dreamteam.agent/v1"
NEEDS = ("hello", "observe", "actions", "validate")

REVIEW_EVERY = 7  # decision points between reviews
PRESSURE_MIN = 24 * 60  # oldest unfinished study, minutes
STEP_BP = 500  # basis points: 5 percentage points


class ProtocolError(Exception):
    """An error response (ok: false)."""

    def __init__(self, op, error):
        self.code = error.get("code")
        super().__init__(f"{op}: {self.code}: {error.get('message')}")


class ExchangeError(Exception):
    """The harness's input ended or broke the protocol."""


class Harness:
    """The policy's side of the exchange: turns in, requests and responses,
    decisions out. Request ids are "r1", "r2", ... for the whole run."""

    def __init__(self, inp, out):
        self._in, self._out, self._next = inp, out, 0

    def _write(self, obj):
        self._out.write(json.dumps(obj, separators=(",", ":")) + "\n")
        self._out.flush()

    def _read(self):
        line = self._in.readline()
        if not line:
            raise ExchangeError("the harness closed the exchange")
        try:
            return json.loads(line)
        except json.JSONDecodeError as e:
            raise ExchangeError(f"a line that is not JSON: {e}")

    def turn(self):
        msg = self._read()
        if msg.get("protocol") != POLICY_PROTOCOL or msg.get("turn") not in ("decide", "end"):
            raise ExchangeError(f"expected a turn, got {msg!r}")
        return msg

    def request(self, op, args=None):
        self._next += 1
        rid = f"r{self._next}"
        req = {"id": rid, "op": op}
        if args is not None:
            req["args"] = args
        self._write(req)
        resp = self._read()
        if resp.get("protocol") != AGENT_PROTOCOL or resp.get("id") != rid:
            raise ExchangeError(f"expected the response to {rid}, got {resp!r}")
        if not resp.get("ok"):
            raise ProtocolError(op, resp.get("error") or {})
        return resp["result"]

    def decide(self, batches):
        self._write({"decision": {"batches": batches}})


def check_contract(hello):
    """The benchmark's contract, from hello: what this policy relies on."""
    if hello.get("protocol") != AGENT_PROTOCOL:
        raise ExchangeError(f"protocol {hello.get('protocol')!r}")
    bench = hello.get("benchmark") or {}
    missing = [op for op in NEEDS if op not in bench.get("permittedOps", [])]
    if bench.get("decision") != POLICY_PROTOCOL or missing:
        raise ExchangeError(f"not the expected benchmark contract (missing {missing})")


def candidates(catalog):
    """Radiologists whose site reading allocation can be raised now,
    lowest allocation in effect first."""
    targets = []
    for t in catalog["types"]:
        if t["type"] == "set_reading_allocation":
            targets = t["targets"]
    out = [a for a in targets
           if a["editable"] and a["inEffectBP"] < a["maxBP"]
           and not any(v["pending"] for v in a["versions"])]
    return sorted(out, key=lambda a: a["inEffectBP"])


def review(h, point):
    """One decision point's batches."""
    if point % REVIEW_EVERY != 0:
        return []
    obs = h.request("observe", {"sections": ["worklist"], "compact": True, "recentLog": 0})
    if obs["observation"]["worklist"]["oldestMin"] <= PRESSURE_MIN:
        return []
    for a in candidates(h.request("actions")):
        action = {"type": "set_reading_allocation", "op": "set", "employee": a["employee"],
                  "effective": a["earliestEffective"],
                  "siteAllocationBP": min(a["maxBP"], a["inEffectBP"] + STEP_BP)}
        if h.request("validate", {"actions": [action]})["valid"]:
            return [[action]]
    return []


def main():
    h = Harness(sys.stdin, sys.stdout)
    checked = False
    try:
        while True:
            msg = h.turn()
            if msg["turn"] == "end":
                return 0
            point = msg["point"]
            try:
                if not checked:
                    check_contract(h.request("hello"))
                    checked = True
                batches = review(h, point)
            except ProtocolError as e:
                print(f"decision point {point}: {e}; no change", file=sys.stderr)
                batches = []
            h.decide(batches)
    except ExchangeError as e:
        print(f"python-ref: {e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
