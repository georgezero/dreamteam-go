"""The versioned run-lifecycle contract (dreamteam.run-lifecycle/v1), loaded
from lifecycle_v1.json, and the few pure definitions the contract itself
states (health body categories, timestamp validity, fault points).

The recorder (v13i.py) and the verifier (v13i_verify.py) both load this
contract. Neither imports the other: the recorder drives transitions through
the table; the verifier reconstructs the archive state from persisted
evidence and checks it against the same table and invariants.
"""
import datetime
import hashlib
import json
import os
import re

HERE = os.path.dirname(os.path.abspath(__file__))
SPEC_FILE = os.path.join(HERE, "lifecycle_v1.json")
LIFECYCLE_ID = "dreamteam.run-lifecycle/v1"


class SpecError(Exception):
    pass


def spec_bytes():
    with open(SPEC_FILE, "rb") as f:
        return f.read()


def spec_sha256():
    return hashlib.sha256(spec_bytes()).hexdigest()


def load():
    spec = json.loads(spec_bytes().decode("utf-8"))
    validate(spec)
    return spec


def validate(spec):
    """The contract's own consistency: every name it uses is declared."""
    def need(ok, message):
        if not ok:
            raise SpecError(message)
    need(spec.get("id") == LIFECYCLE_ID and spec.get("schema") == "dreamteam.lifecycle-spec/v1", "spec identity")
    states, events = spec["states"], spec["events"]
    seen = set()
    for t in spec["transitions"]:
        need(t["from"] is None or t["from"] in states, "unknown from-state " + str(t["from"]))
        need(t["to"] in states and not states[t["to"]].get("classifierOnly"), "bad to-state " + t["to"])
        need(t["event"] in events, "unknown event " + t["event"])
        need(t["from"] is None or not states[t["from"]]["terminal"], "transition out of a terminal state")
        need((t["from"], t["event"]) not in seen, "duplicate transition")
        seen.add((t["from"], t["event"]))
    need(set(events) == {t["event"] for t in spec["transitions"]}, "an event without a transition")
    reach, frontier = set(), [None]
    while frontier:
        s = frontier.pop()
        for t in spec["transitions"]:
            if t["from"] == s and t["to"] not in reach:
                reach.add(t["to"])
                frontier.append(t["to"])
    need(reach == {s for s, v in states.items() if not v.get("classifierOnly")}, "unreachable state")
    for code, rule in spec["stopCodes"].items():
        for s in rule["states"]:
            need(s in states and (s, "stop") in seen or (s == "CLEANUP" and code in ("driver", "operator")),
                 "stop code " + code + " in a state without a stop transition: " + s)
    ops = spec["faultInjection"]["operations"]
    kinds = spec["faultInjection"]["kinds"]
    for name, b in spec["faultInjection"]["boundaries"].items():
        need(b["operation"] in ops, "boundary operation " + name)
    for op in ops.values():
        for phase, ks in op["phases"].items():
            need(all(k in kinds for k in ks), "unknown fault kind")
    expected = spec["faultInjection"]["expected"]
    need(all(r["expect"] in expected for r in spec["faultInjection"]["expectedRule"]), "unknown expectation")
    need(set(spec["health"]["bodyCategories"].values()) <= set(spec["health"]["outcomes"]), "health outcome")
    need(set(spec["requiredArtifactsByState"]) == {s for s, v in states.items() if not v.get("classifierOnly")},
         "required artifacts for every recorder state")
    for k in ("L%02d" % i for i in range(1, 20)):
        need(k in spec["invariants"], "missing invariant " + k)
    return spec


def transition_table(spec):
    return {(t["from"], t["event"]): t["to"] for t in spec["transitions"]}


def fault_points(spec, operation):
    """(phase, kind) pairs the contract declares for an operation."""
    return [(phase, kind) for phase, kinds in spec["faultInjection"]["operations"][operation]["phases"].items()
            for kind in kinds]


# --- health body categories (spec health.bodyRule) ------------------------------

def health_body_category(raw, max_bytes):
    """The category of a /health response body, from its bytes alone."""
    if not isinstance(raw, (bytes, bytearray)):
        raise SpecError("health body must be bytes")
    if len(raw) > max_bytes:
        return "oversized_body"
    try:
        text = bytes(raw).decode("utf-8", "strict")
    except UnicodeDecodeError:
        return "non_utf8_body"
    try:
        doc = json.loads(text)
    except ValueError:
        return "invalid_json"
    if not isinstance(doc, dict):
        return "not_an_object"
    status = doc.get("status")
    if "error" in doc and status == "healthy":
        return "contradictory_body"
    if not isinstance(status, str) or not status:
        return "missing_status"
    return "healthy" if status == "healthy" else "unhealthy_status"


# --- timestamps (spec timestamps.format) ----------------------------------------

_TS = re.compile(r"^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(\.(\d{1,6}))?Z$")


def parse_timestamp(value):
    """A datetime for a valid UTC RFC 3339 timestamp, else None."""
    if not isinstance(value, str):
        return None
    m = _TS.match(value)
    if not m:
        return None
    frac = (m.group(8) or "").ljust(6, "0")
    try:
        return datetime.datetime(int(m.group(1)), int(m.group(2)), int(m.group(3)), int(m.group(4)),
                                 int(m.group(5)), int(m.group(6)), int(frac or 0), tzinfo=datetime.timezone.utc)
    except ValueError:
        return None
