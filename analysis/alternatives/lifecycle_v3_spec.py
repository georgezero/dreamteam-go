"""The versioned run-lifecycle and safety contract dreamteam.run-lifecycle/v3,
loaded from bindings_v1.json, its self-consistency check, and the few pure
definitions the contract itself states: health body categories and the
timestamp grammar.

The recorder (v13k.py) and the verifier (v13k_verify.py) both load this
contract. Neither imports the other. Exception normalization (recorder, from
exception objects) and exception-evidence checking (verifier, from recorded
strings against the contract's tables) are deliberately implemented
separately in each.
"""
import datetime
import hashlib
import json
import os
import re

HERE = os.path.dirname(os.path.abspath(__file__))
SPEC_FILE = os.path.join(HERE, "bindings_v1.json")
LIFECYCLE_ID = "dreamteam.run-lifecycle/v3"
SPEC_SCHEMA = "dreamteam.lifecycle-spec/v3"


class SpecError(Exception):
    pass


def spec_bytes():
    with open(SPEC_FILE, "rb") as f:
        return f.read()


def spec_sha256():
    return hashlib.sha256(spec_bytes()).hexdigest()


def load():
    spec = json.loads(spec_bytes().decode("utf-8"))["lifecycle"]
    validate(spec)
    return spec


def validate(spec):
    """The contract's own consistency: every name it uses is declared."""
    def need(ok, message):
        if not ok:
            raise SpecError(message)
    need(spec.get("id") == LIFECYCLE_ID and spec.get("schema") == SPEC_SCHEMA, "spec identity")
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
    need(set(events) == set(spec["transitionPredicates"]), "every event needs a transition predicate")
    sealed = {s for s, v in states.items() if v.get("sealedOnly")}
    need(sealed == {t["to"] for t in spec["transitions"] if t["event"] in spec["sealOnlyEvents"]},
         "seal-only events must be exactly the transitions into sealed-only states")
    reach, frontier = set(), [None]
    while frontier:
        s = frontier.pop()
        for t in spec["transitions"]:
            if t["from"] == s and t["to"] not in reach:
                reach.add(t["to"])
                frontier.append(t["to"])
    need(reach == {s for s, v in states.items() if not v.get("classifierOnly")}, "unreachable state")
    points = spec["safePoints"]["points"]
    for name, p in points.items():
        need(p["kind"] in ("STOP", "NOTE", "RECORD"), "safe point kind " + name)
        need(all(s in states for s in p["states"]), "safe point state " + name)
    need(set(spec["safePoints"]["dispositions"]) == {"STOP", "AFTER_STOP", "AFTER_SCHEDULE"}, "dispositions")
    for code, rule in spec["stopCodes"].items():
        for s in rule["states"]:
            need(s in states and ((s, "stop") in seen or (s == "CLEANUP" and code == "driver")),
                 "stop code " + code + " in a state without a stop transition: " + s)
        need(set(rule["fields"]) == {"failure", "evidence", "request", "healthCheck", "beforeSignature", "safePoint"},
             "stop fields for " + code)
    stop_states = {s for name, p in points.items() if p["kind"] == "STOP" for s in p["states"]}
    need(stop_states == set(spec["stopCodes"]["operator"]["states"]),
         "operator stop states must be exactly the STOP safe points' states")
    stages = spec["abortedDraw"]["stages"]
    fields = None
    for name, st in stages.items():
        need(st["origin"] in ("recorder", "draw"), "aborted stage origin " + name)
        need(fields is None or set(st["fields"]) == fields, "aborted stage fields differ: " + name)
        fields = set(st["fields"])
    ops = spec["faultInjection"]["operations"]
    kinds = spec["faultInjection"]["kinds"]
    for name, b in spec["faultInjection"]["boundaries"].items():
        need(b["operation"] in ops, "boundary operation " + name)
    for op in ops.values():
        for phase, ks in op["phases"].items():
            need(all(k in kinds for k in ks), "unknown fault kind")
    need(set(ops["safe_point"]["phases"]) == set(points), "every safe point is a fault-injection phase")
    expected = spec["faultInjection"]["expected"]
    need(all(r["expect"].split(" ")[0] in expected for r in spec["faultInjection"]["expectedRule"]), "unknown expectation")
    need(set(spec["health"]["bodyCategories"].values()) <= set(spec["health"]["outcomes"]), "health outcome")
    fam = spec["health"]["readFailureFamilies"]
    listed = [t for f in fam.values() for t in f["types"]]
    need(len(listed) == len(set(listed)), "an exception type listed in two families")
    need(not set(fam) & set(spec["health"]["bodyCategories"]), "a category both body and read failure")
    for bad in ("builtins.KeyboardInterrupt", "builtins.ValueError", "builtins.BaseException", "builtins.Exception"):
        need(bad not in listed, "a non-transport exception listed as a health read failure: " + bad)
    need(set(spec["allowedArchiveFiles"]) == set(spec["artifacts"]) - {"pending-draw.json", "persistence-failure.json"}, "artifact list")
    need(set(spec["finalization"]["seal"]["fields"]) >= {"terminalState", "transition", "manifest", "ledger", "calls",
                                                         "health", "sideArtifacts", "runId", "protocol",
                                                         "scheduleSha256", "finalization"}, "seal fields")
    for k in ["S%02d" % i for i in range(1, 5)] + ["L%02d" % i for i in range(1, 21)]:
        need(k in spec["invariants"], "missing invariant " + k)
    return spec


def transition_table(spec):
    return {(t["from"], t["event"]): t["to"] for t in spec["transitions"]}


def fault_points(spec, operation):
    """(phase, kind) pairs the contract declares for an operation."""
    return [(phase, kind) for phase, kinds in spec["faultInjection"]["operations"][operation]["phases"].items()
            for kind in kinds]


def declared_points(spec):
    return {(b, p) for b, v in spec["faultInjection"]["boundaries"].items()
            for p in spec["faultInjection"]["operations"][v["operation"]]["phases"]}


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
        doc = json.loads(text, object_pairs_hook=_unique_pairs, parse_constant=_nonfinite)
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


# --- timestamps (spec timestamps.grammar) ----------------------------------------

_TS = re.compile(r"([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.([0-9]{1,6}))?Z", re.ASCII)


def parse_timestamp(value):
    """A datetime for a timestamp matching the contract's grammar as a whole
    ASCII string with a valid calendar date and time, else None."""
    if not isinstance(value, str) or not value.isascii():
        return None
    m = _TS.fullmatch(value)
    if not m:
        return None
    frac = (m.group(7) or "").ljust(6, "0")
    try:
        return datetime.datetime(int(m.group(1)), int(m.group(2)), int(m.group(3)), int(m.group(4)),
                                 int(m.group(5)), int(m.group(6)), int(frac or 0), tzinfo=datetime.timezone.utc)
    except ValueError:
        return None


def _unique_pairs(items):
    keys = [k for k, _ in items]
    if len(keys) != len(set(keys)):
        raise ValueError("duplicate JSON key")
    return dict(items)


def _nonfinite(value):
    raise ValueError("nonfinite JSON value")


def bindings():
    doc = json.loads(spec_bytes())
    if doc["schema"] != "dreamteam.evidence-bindings/1":
        raise SpecError("bindings identity")
    return {b["id"]: b for b in doc["bindings"]}


def rule(identity):
    return bindings()[identity]["rules"][0]
