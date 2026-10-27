#!/usr/bin/env python3
"""The DreamTeam LLM agent (llm-agent, version 1).

An external benchmark policy (docs/BENCHMARK.md §4) that asks a language
model what to do. It runs as a separate process under `dreamteam benchmark
run`, speaking the policy protocol (dreamteam.benchmark-policy/v1) on its
standard input and output, and calls a model through an OpenAI-compatible
chat completions API. Standard library only. Specification:
docs/LLM_AGENT.md.

At each decision point (live mode) it:

  1. reads the clock (`digests`) and the compact public observation;
  2. reads the action catalog;
  3. builds a bounded prompt from them;
  4. asks the model for a decision, {"reason": "...", "actions": [...]};
  5. parses it strictly, checks the action types against the catalog and
     validates the batch through the protocol (`validate`);
  6. submits the batch only if it is valid; after a bounded number of
     retries it falls back to no action;
  7. records every model exchange and the decision in its evidence files.

It is stateless: each decision uses only that decision point's
observation and catalog. It never applies, advances, saves or asks for T90:
the harness does. It is never told the seed.

Replay mode (--replay FILE ...) submits the decisions recorded by an
earlier run, without contacting any model, after checking that each
decision point and its digests are the recorded ones.

Evidence goes to ./evidence/ in the working directory the harness gives
it; the harness keeps it in the run's policy-evidence/ (not hashed).
"""

import argparse
import hashlib
import json
import os
import socket
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

AGENT = "llm-agent"
VERSION = "1"
PROMPT_VERSION = "llm-agent-prompt/v1"
SETTINGS_SCHEMA = "dreamteam.llm-agent-settings/v1"
EVIDENCE_SCHEMA = "dreamteam.llm-agent-evidence/v1"
POLICY_PROTOCOL = "dreamteam.benchmark-policy/v1"
AGENT_PROTOCOL = "dreamteam.agent/v1"
NEEDS = ("hello", "digests", "observe", "actions", "validate")

MAX_ACTIONS = 64  # the protocol's batch limit
MAX_REASON = 2000  # characters
MAX_RESPONSE_KEPT = 1 << 20  # characters of a model response kept in evidence

SYSTEM_PROMPT = """\
You manage the operations of a synthetic outpatient imaging practice in \
the DreamTeam simulator. Everything is synthetic; you make operational \
decisions only, never clinical ones.

Objective: keep the practice financially sustainable while giving timely \
service: exams done without long waits or lost demand, studies read and \
reported promptly (turnaround, breaches), operating profit and cash \
healthy. Changes cost money and some take effect only after a lead time. \
Doing nothing is a valid decision and is often right.

At each decision point you see the practice's current public state and the \
action catalog. You will be asked again at later decision points.

Reply with exactly one JSON object and nothing else:
{"reason": "<one or two sentences>", "actions": [<action objects>]}

Rules:
- Use only action types in the catalog. Each action object must follow its \
type's JSON Schema exactly (no extra fields) and name only listed targets \
and allowed values.
- All your actions form one atomic batch: if any one is refused, none is \
applied.
- extend_shift, set_walkin_policy and set_worklist_policy cannot share a \
batch with hire, terminate, set_roster, set_reading_allocation or \
contract_teleradiology.
- At most 64 actions. Use "actions": [] when no change is warranted.
"""


class SettingsError(Exception):
    """The settings file or the environment is not usable."""


class ProtocolError(Exception):
    """An error response (ok: false)."""

    def __init__(self, op, error):
        self.code = error.get("code")
        self.message = error.get("message")
        super().__init__(f"{op}: {self.code}: {self.message}")


class ExchangeError(Exception):
    """The harness's input ended or broke the protocol."""


class ReplayError(Exception):
    """A recorded decision does not belong to this decision point."""


# --- the policy protocol -------------------------------------------------

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
    """The benchmark's contract, from hello: what this agent relies on."""
    if hello.get("protocol") != AGENT_PROTOCOL:
        raise ExchangeError(f"protocol {hello.get('protocol')!r}")
    bench = hello.get("benchmark") or {}
    missing = [op for op in NEEDS if op not in bench.get("permittedOps", [])]
    if bench.get("decision") != POLICY_PROTOCOL or missing:
        raise ExchangeError(f"not the expected benchmark contract (missing {missing})")


# --- settings ------------------------------------------------------------

DEFAULTS = {
    "requestTimeoutSeconds": 120,
    "temperature": 0,
    "maxOutputTokens": 1024,
    "maxRetries": 1,
    "decisionBudgetSeconds": 600,
    "responseFormat": "json_object",
    "maxPromptChars": 200000,
}
FIELDS = {"schema", "description", "model", "modelEnv", "baseUrl", "baseUrlEnv", "apiKeyEnv",
          "systemPromptPath", *DEFAULTS}


def _number(s, name, lo, hi, integer=True):
    v = s[name]
    if isinstance(v, bool) or not isinstance(v, int if integer else (int, float)) or not lo <= v <= hi:
        raise SettingsError(f"{name} must be {'an integer' if integer else 'a number'} {lo}–{hi}, not {v!r}")


def _either(s, env, name, env_name):
    """A value given directly, or by the name of an environment variable."""
    if (name in s) == (env_name in s):
        raise SettingsError(f"give exactly one of {name} and {env_name}")
    if name in s:
        v = s[name]
    else:
        v = env.get(s[env_name], "")
        if not v:
            raise SettingsError(f"the environment variable {s[env_name]} ({env_name}) is not set")
    if not isinstance(v, str) or not v:
        raise SettingsError(f"{name} must be a non-empty string")
    return v


def load_settings(path, env):
    """The settings file, checked, with its defaults and the values it names
    in the environment. The API key is read from the environment only."""
    try:
        with open(path, encoding="utf-8") as f:
            s = json.load(f)
    except (OSError, json.JSONDecodeError) as e:
        raise SettingsError(f"settings {path}: {e}")
    if not isinstance(s, dict):
        raise SettingsError("the settings must be a JSON object")
    if "apiKey" in s:
        raise SettingsError("an API key is never put in the settings; name its environment variable in apiKeyEnv")
    unknown = sorted(set(s) - FIELDS)
    if unknown:
        raise SettingsError(f"unknown settings fields {unknown}")
    if s.get("schema") != SETTINGS_SCHEMA:
        raise SettingsError(f"schema must be {SETTINGS_SCHEMA!r}")
    out = dict(DEFAULTS)
    out.update({k: v for k, v in s.items() if k in DEFAULTS})
    _number(out, "requestTimeoutSeconds", 1, 3600)
    _number(out, "temperature", 0, 2, integer=False)
    _number(out, "maxOutputTokens", 1, 65536)
    _number(out, "maxRetries", 0, 3)
    _number(out, "decisionBudgetSeconds", 1, 3600)
    _number(out, "maxPromptChars", 1000, 2000000)
    if out["responseFormat"] not in ("json_object", "none"):
        raise SettingsError("responseFormat must be \"json_object\" or \"none\"")
    out["model"] = _either(s, env, "model", "modelEnv")
    base = _either(s, env, "baseUrl", "baseUrlEnv")
    parts = urllib.parse.urlsplit(base)
    if parts.scheme not in ("http", "https") or not parts.hostname:
        raise SettingsError("the base URL must be an http or https URL")
    if parts.username or parts.password:
        raise SettingsError("the base URL must not carry credentials; use apiKeyEnv")
    out["baseUrl"] = base.rstrip("/")
    out["endpoint"] = endpoint_id(base)
    out["apiKey"] = None
    if "apiKeyEnv" in s:
        if not isinstance(s["apiKeyEnv"], str) or not s["apiKeyEnv"]:
            raise SettingsError("apiKeyEnv must name an environment variable")
        out["apiKey"] = env.get(s["apiKeyEnv"]) or None
    out["systemPrompt"], out["systemPromptSource"] = SYSTEM_PROMPT, "built-in"
    if "systemPromptPath" in s:
        p = s["systemPromptPath"]
        if not isinstance(p, str) or not p:
            raise SettingsError("systemPromptPath must be a path")
        full = p if os.path.isabs(p) else os.path.join(os.path.dirname(os.path.abspath(path)), p)
        try:
            with open(full, encoding="utf-8") as f:
                out["systemPrompt"] = f.read()
        except OSError as e:
            raise SettingsError(f"systemPromptPath: {e}")
        out["systemPromptSource"] = os.path.basename(p)
    return out


def endpoint_id(url):
    """The endpoint without credentials, query or fragment."""
    p = urllib.parse.urlsplit(url)
    host = p.hostname or ""
    if ":" in host:
        host = f"[{host}]"
    if p.port:
        host += f":{p.port}"
    return urllib.parse.urlunsplit((p.scheme, host, p.path.rstrip("/"), "", ""))


def sha256(text):
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


# --- the model -----------------------------------------------------------

class ModelClient:
    """POST {baseUrl}/chat/completions. The key, if any, goes in the
    Authorization header only; it is never recorded."""

    def __init__(self, settings):
        self.s = settings

    def payload(self, messages):
        p = {"model": self.s["model"], "messages": messages,
             "temperature": self.s["temperature"], "max_tokens": self.s["maxOutputTokens"]}
        if self.s["responseFormat"] == "json_object":
            p["response_format"] = {"type": "json_object"}
        return p

    def call(self, payload):
        """One request: (status, raw body text, error); error is None on a
        2xx answer."""
        headers = {"Content-Type": "application/json"}
        if self.s["apiKey"]:
            headers["Authorization"] = "Bearer " + self.s["apiKey"]
        req = urllib.request.Request(self.s["baseUrl"] + "/chat/completions",
                                     data=json.dumps(payload).encode("utf-8"), headers=headers, method="POST")
        try:
            with urllib.request.urlopen(req, timeout=self.s["requestTimeoutSeconds"]) as r:
                return r.status, r.read().decode("utf-8", "replace"), None
        except urllib.error.HTTPError as e:
            body = e.read().decode("utf-8", "replace")
            return e.code, body, f"http_status: HTTP {e.code}"
        except (socket.timeout, TimeoutError):
            return None, "", f"timeout: no answer within {self.s['requestTimeoutSeconds']} s"
        except urllib.error.URLError as e:
            if isinstance(e.reason, (socket.timeout, TimeoutError)):
                return None, "", f"timeout: no answer within {self.s['requestTimeoutSeconds']} s"
            return None, "", f"connection: {e.reason}"
        except OSError as e:
            return None, "", f"connection: {e}"


def completion(body):
    """The message content and usage of a chat completion body."""
    try:
        o = json.loads(body)
    except json.JSONDecodeError:
        raise ValueError("response_not_json: the answer is not JSON")
    try:
        content = o["choices"][0]["message"]["content"]
    except (KeyError, IndexError, TypeError):
        raise ValueError("response_shape: no choices[0].message.content")
    if not isinstance(content, str) or not content.strip():
        raise ValueError("empty_content: the model returned no content")
    return content, o.get("usage") if isinstance(o.get("usage"), dict) else None


# --- prompt and decision -------------------------------------------------

def build_prompt(point, obs, catalog, max_chars):
    """The user message: the decision point, the compact observation and the
    catalog. Over max_chars, the largest action schemas are left out first
    (their constraints and targets stay); still over, None."""
    header = (f"Decision point {point}: {obs.get('time')} (day {obs.get('day')}, {obs.get('weekday')}).\n"
              "Decide what to change now, if anything.\n\n")
    types = [dict(t) for t in catalog.get("types", [])]
    cat = {"batch": catalog.get("batch"), "types": types}
    omitted = []

    def render():
        return (header
                + "OBSERVATION (dreamteam.observation/v1, compact: listed history is left out and "
                  "counted in \"omitted\"):\n"
                + json.dumps(obs, separators=(",", ":"))
                + "\n\nACTION CATALOG (batch rules; for each type its JSON Schema, constraints and current targets):\n"
                + json.dumps(cat, separators=(",", ":"))
                + "\n\nReply with the JSON decision object only.")

    text = render()
    for t in sorted(types, key=lambda t: -len(json.dumps(t.get("schema")))):
        if len(text) <= max_chars:
            break
        t["schema"] = "left out for size: follow the constraints and targets"
        omitted.append(t["type"])
        text = render()
    if len(text) > max_chars:
        return None, omitted
    return text, omitted


def strip_wrapping(content):
    """The reply without a leading <think>…</think> block or one enclosing
    Markdown code fence: the only tolerances."""
    c = content.strip()
    if c.startswith("<think>") and "</think>" in c:
        c = c[c.index("</think>") + len("</think>"):].strip()
    if c.startswith("```") and c.endswith("```") and c.count("```") == 2:
        c = c[3:-3]
        if c.startswith("json"):
            c = c[4:]
        c = c.strip()
    return c


def parse_decision(content, types):
    """The decision in a model reply, strictly: one JSON object with exactly
    "reason" (a string) and "actions" (a list of at most 64 objects, each
    with a catalog type). Raises ValueError with a stable code."""
    try:
        d = json.loads(strip_wrapping(content))
    except json.JSONDecodeError as e:
        raise ValueError(f"invalid_json: the reply is not one JSON object ({e.msg})")
    if not isinstance(d, dict):
        raise ValueError("invalid_decision: the reply is not a JSON object")
    if set(d) != {"reason", "actions"}:
        raise ValueError(f"invalid_decision: the object must have exactly \"reason\" and \"actions\", not {sorted(d)}")
    if not isinstance(d["reason"], str):
        raise ValueError("invalid_decision: \"reason\" must be a string")
    if len(d["reason"]) > MAX_REASON:
        raise ValueError(f"invalid_decision: \"reason\" is longer than {MAX_REASON} characters")
    acts = d["actions"]
    if not isinstance(acts, list) or len(acts) > MAX_ACTIONS:
        raise ValueError(f"invalid_decision: \"actions\" must be a list of at most {MAX_ACTIONS} actions")
    for i, a in enumerate(acts):
        if not isinstance(a, dict) or not isinstance(a.get("type"), str):
            raise ValueError(f"invalid_action: action {i} is not an object with a \"type\"")
        if a["type"] not in types:
            raise ValueError(f"unknown_action: action {i} has type {a['type']!r}, not in the catalog")
    return d


def refusals(validation):
    """The refused results of a validate answer, briefly."""
    out = []
    for i, r in enumerate(validation.get("results") or []):
        if not r.get("ok"):
            out.append(f"action {i}: {r.get('code')}: {r.get('message')}")
    return "; ".join(out) or "the batch is not valid"


class Evidence:
    """The agent's evidence files: agent.json (settings, without secrets),
    model.jsonl (every model exchange), decisions.jsonl (every decision
    point) and summary.json (totals)."""

    def __init__(self, path):
        self.dir = path
        os.makedirs(path, exist_ok=True)
        self.calls = []

    def write(self, name, obj):
        with open(os.path.join(self.dir, name), "w", encoding="utf-8") as f:
            json.dump(obj, f, indent=1, sort_keys=True)
            f.write("\n")

    def append(self, name, obj):
        # Keys in their own order: a recorded batch is replayed byte for byte.
        with open(os.path.join(self.dir, name), "a", encoding="utf-8") as f:
            f.write(json.dumps(obj, separators=(",", ":")) + "\n")

    def call(self, rec):
        self.calls.append(rec)
        self.append("model.jsonl", rec)


def ask(h, ev, client, settings, point):
    """One live decision point: the batches to submit and the record."""
    start = time.monotonic()
    clock = h.request("digests")
    rec = {"point": point, "time": clock["time"], "day": clock["day"], "digests": clock["digests"],
           "mode": "live", "attempts": 0, "errors": [], "outcome": "fallback", "reason": None,
           "proposed": None, "validation": None, "batches": []}
    obs = h.request("observe", {"compact": True, "recentLog": 0})["observation"]
    catalog = h.request("actions")
    types = {t["type"] for t in catalog.get("types", [])}
    prompt, omitted = build_prompt(point, obs, catalog, settings["maxPromptChars"])
    rec["schemasLeftOut"] = omitted
    if prompt is None:
        rec["errors"].append(f"prompt_too_large: over {settings['maxPromptChars']} characters")
        return [], rec
    rec["promptChars"], rec["promptSha256"] = len(prompt), sha256(prompt)
    messages = [{"role": "system", "content": settings["systemPrompt"]}, {"role": "user", "content": prompt}]
    for attempt in range(1, settings["maxRetries"] + 2):
        if attempt > 1 and time.monotonic() - start >= settings["decisionBudgetSeconds"]:
            rec["errors"].append(f"budget: {settings['decisionBudgetSeconds']} s spent; no further attempt")
            break
        rec["attempts"] = attempt
        payload = client.payload(messages)
        t0 = time.monotonic()
        status, body, err = client.call(payload)
        call = {"point": point, "attempt": attempt, "model": settings["model"], "endpoint": settings["endpoint"],
                "request": payload, "status": status, "latencyMs": round((time.monotonic() - t0) * 1000),
                "response": body[:MAX_RESPONSE_KEPT], "responseTruncated": len(body) > MAX_RESPONSE_KEPT,
                "usage": None, "content": None, "error": err}
        if err is None:
            try:
                content, call["usage"] = completion(body)
                call["content"] = content
                d = parse_decision(content, types)
                rec["reason"], rec["proposed"] = d["reason"], d["actions"]
                if d["actions"]:
                    v = h.request("validate", {"actions": d["actions"]})
                    rec["validation"] = v
                    if not v.get("valid"):
                        raise ValueError(f"refused: {refusals(v)}")
                    rec["batches"], rec["outcome"] = [d["actions"]], "actions"
                else:
                    rec["outcome"] = "no_action"
            except ProtocolError as e:
                call["error"] = f"invalid_action: validate answered {e.code}: {e.message}"
            except ValueError as e:
                call["error"] = str(e)
        ev.call(call)
        if call["error"] is None:
            return rec["batches"], rec
        rec["errors"].append(f"attempt {attempt}: {call['error']}")
        if call["content"] is not None:  # the model answered: say what was wrong
            messages = messages + [
                {"role": "assistant", "content": call["content"]},
                {"role": "user", "content": f"Your reply could not be used: {call['error']}. Reply again with one "
                                            "JSON object {\"reason\": ..., \"actions\": [...]} that follows the "
                                            "catalog exactly, or with \"actions\": [] for no change."}]
    rec["batches"], rec["outcome"] = [], "fallback"  # never a batch that was not validated
    return [], rec


# --- replay --------------------------------------------------------------

def load_recordings(paths):
    """Each recorded decisions file: (path, SHA-256, records)."""
    out = []
    for p in paths:
        try:
            with open(p, "rb") as f:
                raw = f.read()
            recs = [json.loads(l) for l in raw.decode("utf-8").splitlines() if l.strip()]
        except (OSError, UnicodeDecodeError, json.JSONDecodeError) as e:
            raise SettingsError(f"replay {p}: {e}")
        if not recs or any(not isinstance(r, dict) or "point" not in r or "digests" not in r
                           or not isinstance(r.get("batches"), list) for r in recs):
            raise SettingsError(f"replay {p}: not a recorded decisions file")
        out.append((p, hashlib.sha256(raw).hexdigest(), recs))
    return out


class Replayer:
    """The recorded decisions, checked turn by turn: the recording is the
    one whose first decision point and digests are this run's (several
    recordings may be given for a matrix); each later decision point must
    be the recorded one, with the recorded digests."""

    def __init__(self, recordings):
        self.recordings, self.chosen, self.i = recordings, None, 0

    def next(self, point, clock):
        if self.chosen is None:
            match = [r for r in self.recordings if r[2][0]["point"] == point and r[2][0]["digests"] == clock["digests"]]
            seqs = {json.dumps([(x["point"], x["digests"], x["batches"]) for x in r[2]], sort_keys=True) for r in match}
            if not match:
                raise ReplayError(f"no recording begins at decision point {point} with this run's digests")
            if len(seqs) > 1:
                raise ReplayError(f"{len(match)} different recordings begin with this run's digests")
            self.chosen = match[0]
        recs = self.chosen[2]
        if self.i >= len(recs):
            raise ReplayError(f"decision point {point}: the recording has only {len(recs)} decision points")
        r = recs[self.i]
        if r["point"] != point:
            raise ReplayError(f"decision point {point}: the recording's next one is {r['point']}")
        if r["digests"] != clock["digests"]:
            raise ReplayError(f"decision point {point}: the digests differ from the recording's")
        self.i += 1
        return r

    def finish(self):
        if self.chosen is None:
            return
        left = len(self.chosen[2]) - self.i
        if left:
            raise ReplayError(f"the run ended with {left} recorded decision points not reached")


# --- main ----------------------------------------------------------------

def summary(ev, records):
    def total(key):
        vals = [c["usage"].get(key) for c in ev.calls if c.get("usage") and isinstance(c["usage"].get(key), int)]
        return sum(vals) if vals else None
    lat = [c["latencyMs"] for c in ev.calls]
    outcomes = {}
    for r in records:
        outcomes[r["outcome"]] = outcomes.get(r["outcome"], 0) + 1
    return {"decisionPoints": len(records), "outcomes": outcomes,
            "modelCalls": len(ev.calls), "failedCalls": sum(1 for c in ev.calls if c["error"]),
            "retries": sum(max(r.get("attempts", 0) - 1, 0) for r in records),
            "promptTokens": total("prompt_tokens"), "completionTokens": total("completion_tokens"),
            "totalTokens": total("total_tokens"),
            "latencyMs": {"total": sum(lat), "mean": round(sum(lat) / len(lat)) if lat else None,
                          "max": max(lat) if lat else None}}


def main(argv=None):
    ap = argparse.ArgumentParser(description="The DreamTeam LLM agent (a benchmark policy).")
    ap.add_argument("--settings", help="settings file (dreamteam.llm-agent-settings/v1); required live")
    ap.add_argument("--replay", action="append", default=[], metavar="FILE",
                    help="submit the decisions recorded in FILE (decisions.jsonl) instead of asking a model; repeatable")
    ap.add_argument("--evidence", default="evidence", help="evidence directory (default ./evidence)")
    a = ap.parse_args(argv)
    try:
        if a.replay:
            settings, recordings = None, load_recordings(a.replay)
        elif a.settings:
            settings, recordings = load_settings(a.settings, os.environ), None
        else:
            raise SettingsError("give --settings (live) or --replay (replay)")
    except SettingsError as e:
        print(f"{AGENT}: {e}", file=sys.stderr)
        return 2
    ev = Evidence(a.evidence)
    info = {"schema": EVIDENCE_SCHEMA, "agent": AGENT, "version": VERSION, "promptVersion": PROMPT_VERSION,
            "mode": "replay" if recordings else "live",
            "python": sys.version.split()[0]}
    if settings:
        info.update({k: settings[k] for k in ("model", "endpoint", "temperature", "maxOutputTokens",
                                               "requestTimeoutSeconds", "maxRetries", "decisionBudgetSeconds",
                                               "responseFormat", "maxPromptChars", "systemPromptSource")})
        info["systemPromptSha256"] = sha256(settings["systemPrompt"])
        info["apiKey"] = "set" if settings["apiKey"] else "none"
    else:
        info["replay"] = [{"file": os.path.basename(p), "sha256": s} for p, s, _ in recordings]
    ev.write("agent.json", info)
    client = ModelClient(settings) if settings else None
    replayer = Replayer(recordings) if recordings else None
    h = Harness(sys.stdin, sys.stdout)
    records, checked = [], False
    try:
        while True:
            msg = h.turn()
            if msg["turn"] == "end":
                if replayer:
                    replayer.finish()
                return 0
            point = msg["point"]
            if not checked:
                check_contract(h.request("hello"))
                checked = True
            if replayer:
                clock = h.request("digests")
                r = replayer.next(point, clock)
                batches = r["batches"]
                rec = {"point": point, "time": clock["time"], "day": clock["day"], "digests": clock["digests"],
                       "mode": "replay", "outcome": r.get("outcome"), "reason": r.get("reason"),
                       "batches": batches, "replayed": os.path.basename(replayer.chosen[0])}
            else:
                try:
                    batches, rec = ask(h, ev, client, settings, point)
                except ProtocolError as e:
                    batches, rec = [], {"point": point, "mode": "live", "outcome": "fallback", "batches": [],
                                        "errors": [f"protocol: {e}"], "attempts": 0}
            records.append(rec)
            ev.append("decisions.jsonl", rec)
            ev.write("summary.json", summary(ev, records))
            h.decide(batches)
    except (ExchangeError, ReplayError) as e:
        print(f"{AGENT}: {e}", file=sys.stderr)
        return 1
    finally:
        ev.write("summary.json", summary(ev, records))


if __name__ == "__main__":
    sys.exit(main())
