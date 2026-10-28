#!/usr/bin/env python3
"""The DreamTeam LLM agent (llm-agent, version 2).

An external benchmark policy (docs/BENCHMARK.md §4) that asks a language
model what to do. It runs as a separate process under `dreamteam benchmark
run`, speaking the policy protocol (dreamteam.benchmark-policy/v1) on its
standard input and output, and calls a model through an OpenAI-compatible
chat completions API. Standard library only. Specification:
docs/LLM_AGENT.md.

At each decision point (live mode) it:

  1. reads the clock (`digests`), the compact public observation and the
     action catalog;
  2. updates its memory from the observation: what became of its last
     decision, and the metrics now;
  3. builds a targeted prompt: the time and operating hours, the active
     problems, its bounded memory, a summary of the observation, and every
     action type with its fields, rules, timing and targets; what it leaves
     out it names, and the model may ask for it once ("details");
  4. asks the model for a decision, {"reason": "...", "actions": [...]};
  5. parses it strictly, checks the action types against the catalog,
     splits the actions into an operations batch and a staffing batch and
     validates each through the protocol (`validate`);
  6. submits the batches only if every one is valid; after a bounded
     number of retries it falls back to no action;
  7. records every model exchange, the decision and its memory in its
     evidence files, with the run's model usage and estimated cost.

Its memory holds its own decisions, the validation results it was given
and what the public observation showed afterwards, within fixed bounds. It
lives in the process, so it begins empty in every run. It never applies,
advances, saves or asks for T90: the harness does. It is never told the
seed.

Replay mode (--replay FILE ...) submits the decisions recorded by an
earlier run, without contacting any model, after checking that each
decision point and its digests are the recorded ones. For a recording of
this version it also rebuilds the memory and the prompt from the same
observations and the recorded decisions, and checks both against the
recording. A V0.5a recording (version 1, no memory) replays its batches as
before; a recording of any other memory or prompt version is refused.

Evidence goes to ./evidence/ in the working directory the harness gives
it; the harness keeps it in the run's policy-evidence/ (not hashed).
"""

import argparse
import copy
import datetime
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
VERSION = "2"
PROMPT_VERSION = "llm-agent-prompt/v2"
MEMORY_FORMAT = "llm-agent-memory/v1"
SETTINGS_SCHEMA = "dreamteam.llm-agent-settings/v1"
EVIDENCE_SCHEMA = "dreamteam.llm-agent-evidence/v2"
POLICY_PROTOCOL = "dreamteam.benchmark-policy/v1"
AGENT_PROTOCOL = "dreamteam.agent/v1"
NEEDS = ("hello", "digests", "observe", "actions", "validate")

MAX_ACTIONS = 64  # the protocol's batch limit
MAX_REASON = 2000  # characters
MAX_DETAILS = 8  # names in one details request
MAX_RESPONSE_KEPT = 1 << 20  # characters of a model response kept in evidence

# The V0.2 staffing actions: they form one atomic batch and cannot share it
# with the V0.1 operations actions (mixed_batch; docs/AGENT_PROTOCOL.md §3).
STAFFING = ("hire", "terminate", "set_roster", "set_reading_allocation", "contract_teleradiology")

SYSTEM_PROMPT = """\
You manage the operations of a synthetic outpatient imaging practice in \
the DreamTeam simulator. Everything is synthetic; you make operational \
decisions only, never clinical ones.

Objective: keep the practice financially sustainable while giving timely \
service: exams done without long waits or lost demand, studies read and \
reported promptly (turnaround, breaches), operating profit and cash \
healthy. Changes cost money and some take effect only after a lead time. \
Doing nothing is a valid decision and is often right.

At each decision point you see the time and operating hours, the active \
problems, your memory of your own earlier decisions in this run and what \
followed them, a summary of the practice's public state, and every action \
type with its fields, rules, timing and current targets. You will be asked \
again at later decision points.

Reply with exactly one JSON object and nothing else:
{"reason": "<one or two sentences>", "actions": [<action objects>]}

Before deciding you may ask once for more: {"reason": "<why>", "actions": \
[], "details": ["<name>", ...]}, where each name is an action type (its \
full JSON Schema and targets) or a path listed under LEFT OUT. You will \
then be asked again with them.

Rules:
- Use only the listed action types, with exactly their listed fields, \
listed targets and allowed values. Fields marked ? are optional.
- Dates are practice-local YYYY-MM-DD and times HH:MM. Respect each \
action's earliest and latest dates and the operating hours shown.
- Operations actions and staffing actions may both be in "actions": they \
are validated and submitted as two batches at the same time, operations \
first. Each batch is all-or-nothing. If either is refused, nothing is \
submitted and you are told why.
- At most 64 actions of each kind. Use "actions": [] when no change is \
warranted.
"""


class SettingsError(Exception):
    """The settings file, a recording or the environment is not usable."""


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


def look(h):
    """The decision point's public inputs: the compact observation, what
    compact left out (path: entries), and the action catalog."""
    o = h.request("observe", {"compact": True, "recentLog": 0})
    return o["observation"], o.get("omitted") or {}, h.request("actions")


# --- settings ------------------------------------------------------------

DEFAULTS = {
    "requestTimeoutSeconds": 120,
    "temperature": 0,
    "maxOutputTokens": 1024,
    "maxRetries": 1,
    "decisionBudgetSeconds": 600,
    "responseFormat": "json_object",
    "maxPromptChars": 200000,
    "observation": "summary",
    "catalog": "guided",
    "allowDetails": True,
    "memoryDecisions": 6,
    "memoryMetricPoints": 6,
    "memoryMaxChars": 6000,
}
FIELDS = {"schema", "description", "model", "modelEnv", "baseUrl", "baseUrlEnv", "apiKeyEnv",
          "systemPromptPath", "pricing", *DEFAULTS}


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


def _pricing(p):
    """Optional prices per million tokens: {"currency", "inputPerMillion",
    "outputPerMillion"}."""
    if not isinstance(p, dict) or set(p) != {"currency", "inputPerMillion", "outputPerMillion"}:
        raise SettingsError("pricing must be {\"currency\", \"inputPerMillion\", \"outputPerMillion\"}")
    c = p["currency"]
    if not isinstance(c, str) or not (len(c) == 3 and c.isalpha() and c.isupper()):
        raise SettingsError("pricing.currency must be a three-letter code such as \"USD\"")
    for k in ("inputPerMillion", "outputPerMillion"):
        v = p[k]
        if isinstance(v, bool) or not isinstance(v, (int, float)) or not 0 <= v <= 100000:
            raise SettingsError(f"pricing.{k} must be a number 0–100000, not {v!r}")
    return dict(p)


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
    _number(out, "memoryDecisions", 0, 50)
    _number(out, "memoryMetricPoints", 0, 30)
    _number(out, "memoryMaxChars", 500, 50000)
    if out["responseFormat"] not in ("json_object", "none"):
        raise SettingsError("responseFormat must be \"json_object\" or \"none\"")
    if out["observation"] not in ("summary", "compact"):
        raise SettingsError("observation must be \"summary\" or \"compact\"")
    if out["catalog"] not in ("guided", "full"):
        raise SettingsError("catalog must be \"guided\" or \"full\"")
    if not isinstance(out["allowDetails"], bool):
        raise SettingsError("allowDetails must be true or false")
    out["pricing"] = _pricing(s["pricing"]) if "pricing" in s else None
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


# The settings that shape the prompt and the memory. A replay uses the
# recording's (decisions.jsonl's "promptSettings"), so it rebuilds the same.
PROMPT_SETTINGS = ("observation", "catalog", "allowDetails", "memoryDecisions", "memoryMetricPoints",
                   "memoryMaxChars", "maxPromptChars")


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


def compact(obj):
    return json.dumps(obj, separators=(",", ":"), ensure_ascii=False)


def clip(text, n):
    text = " ".join(str(text).split())
    return text if len(text) <= n else text[:n - 1] + "…"


def money(cents):
    """Whole dollars, from integer cents."""
    if not isinstance(cents, int) or isinstance(cents, bool):
        return "?"
    return f"{'-' if cents < 0 else ''}${abs(cents) // 100:,}"


def dig(obj, *path, default=None):
    for k in path:
        if isinstance(obj, dict) and k in obj:
            obj = obj[k]
        else:
            return default
    return obj


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
    """The message content, usage and served model of a chat completion
    body."""
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
    usage = o.get("usage") if isinstance(o.get("usage"), dict) else None
    return content, usage, o.get("model") if isinstance(o.get("model"), str) else None


# --- the observation summary ---------------------------------------------

# Paths the summary leaves out of the compact observation: history,
# breakdowns and per-item listings. Each one is named in the prompt, and the
# model may ask for it. (path, how many list entries to keep, or None to
# leave it out entirely.)
PERIODS = ("today", "monthToDate", "trailing30", "sinceStart")
SUMMARY_CUTS = (
    [("finance.cashHistory", None), ("finance.accounts", None), ("finance.opening", None),
     ("finance.claims.aging", None)]
    + [(f"finance.{p}.{k}", None) for p in PERIODS for k in ("byModality", "fixedByItem")]
    + [("today.rooms", None), ("today.staff", None), ("today.demand.byModality", None),
       ("worklist.items", 5), ("alerts.active", 10), ("actions.recent", 5)]
)
MAX_LIST = 20  # any other listing longer than this is cut to it

# Fields the summary keeps: of each finance period, the totals; of each
# worklist item, what identifies and ages it; of each staff member,
# everything but what the shift, the catalog's terminate targets or the
# contract already say.
PERIOD_KEEP = ("label", "from", "to", "days", "netRevenueCents", "laborCents", "fixedCents", "variableCents",
               "operatingExpenseCents", "operatingProfitCents", "marginBP", "collectionsCents", "disbursementsCents",
               "netCashFlowCents", "reportsFinal", "denialRateBP")
ITEM_KEEP = ("study", "modality", "priority", "state", "ageMin", "overdue")
STAFF_DROP = ("startsAt", "contractInferred", "severanceCents")


def _parent(obj, path):
    keys = path.split(".")
    for k in keys[:-1]:
        obj = obj.get(k) if isinstance(obj, dict) else None
    return (obj, keys[-1]) if isinstance(obj, dict) and keys[-1] in obj else (None, None)


def summarize(obs):
    """The observation without the SUMMARY_CUTS (and any other listing cut
    to MAX_LIST), and what was left out: (summary, [(path, note)])."""
    s = copy.deepcopy(obs)
    left = []
    for path, keep in SUMMARY_CUTS:
        parent, key = _parent(s, path)
        if parent is None:
            continue
        v = parent[key]
        if keep is None:
            if v not in (None, [], {}):
                left.append((path, ""))
            del parent[key]
        elif isinstance(v, list) and len(v) > keep:
            left.append((path, f"{len(v)} entries; the first {keep} shown"))
            parent[key] = v[:keep]

    def keep_fields(path, keep=None, drop=()):
        parent, key = _parent(s, path)
        if parent is None:
            return
        rows = parent[key] if isinstance(parent[key], list) else [parent[key]]
        gone = set()
        for r in rows:
            if isinstance(r, dict):
                for k in [k for k in r if (keep is not None and k not in keep) or k in drop]:
                    gone.add(k)
                    del r[k]
        if gone:
            left.append((path, "fields " + ", ".join(sorted(gone))))
    for p in PERIODS:
        keep_fields(f"finance.{p}", keep=PERIOD_KEEP)
    keep_fields("worklist.items", keep=ITEM_KEEP)
    keep_fields("staff", drop=STAFF_DROP)

    def cut(o, path):
        if isinstance(o, dict):
            for k in o:
                cut(o[k], f"{path}.{k}" if path else k)
        elif isinstance(o, list):
            if len(o) > MAX_LIST:
                left.append((path, f"{len(o)} entries; the first {MAX_LIST} shown"))
                del o[MAX_LIST:]
            for i, x in enumerate(o):
                cut(x, f"{path}[{i}]")
    cut(s, "")
    return s, left


def full_path(obs, path):
    """A left-out path's value in the compact observation (None if absent)."""
    parent, key = _parent(obs, path)
    return parent[key] if parent is not None else None


def problems(obs):
    """The active problems, as public facts: alerts, the reading backlog,
    lost demand, coverage and money. Lines of text."""
    out = []
    for a in (dig(obs, "alerts", "active") or [])[:10]:
        why = f" — {clip(a.get('why'), 160)}" if a.get("why") else ""
        out.append(f"[{a.get('severity')}] {clip(a.get('text', ''), 160)}{why}")
    w = obs.get("worklist") or {}
    items = w.get("items") or []
    overdue = sum(1 for i in items if isinstance(i, dict) and i.get("overdue"))
    if w.get("unread") or w.get("reading"):
        out.append(f"Reading backlog: {w.get('unread')} unread, {w.get('reading')} being read, {w.get('urgent')} urgent, "
                   f"{overdue} overdue, oldest {w.get('oldestMin')} min; today {w.get('breachToday')} breaches, "
                   f"TAT p90 {w.get('tatP90Min')} min.")
    mods = dig(obs, "demand", "modalities") or []
    lost_today = sum(dig(m, "today", "lost", default=0) or 0 for m in mods)
    lost = sum(dig(m, "sinceStart", "lost", default=0) or 0 for m in mods)
    if lost or lost_today:
        out.append(f"Lost demand: {lost_today} today, {lost} since the start.")
    full = [f"{m.get('modality')} next open {m.get('nextOpen')}" for m in mods if (m.get("nextOpenDays") or 0) > 3]
    if full:
        out.append("Booking pressure (first open slot over 3 days ahead): " + ", ".join(full) + ".")
    ft = dig(obs, "flow", "today") or {}
    gaps = {k: ft.get(k) for k in ("noCoverage", "missed", "left", "walkInsDeclined") if ft.get(k)}
    if gaps:
        out.append("Today so far: " + ", ".join(f"{k} {v}" for k, v in gaps.items()) + ".")
    f = obs.get("finance") or {}
    cash = f.get("cashCents")
    if isinstance(cash, int) and cash < 0 or dig(f, "financing", "daysBelowZero"):
        out.append(f"Cash {money(cash)}; days below zero {dig(f, 'financing', 'daysBelowZero')}.")
    mtd = dig(f, "monthToDate", "operatingProfitCents")
    if isinstance(mtd, int) and mtd < 0:
        out.append(f"Operating profit month to date {money(mtd)} (margin {dig(f, 'monthToDate', 'marginBP')} BP).")
    return out or ["None shown by the observation."]


# --- the action guidance -------------------------------------------------

def brief(s):
    """A JSON Schema in one line: fields marked ? are optional; fields with
    the same schema are listed together (Mon|Tue: …)."""
    if not isinstance(s, dict):
        return compact(s)
    if "const" in s:
        return compact(s["const"])
    if "enum" in s:
        return "|".join(compact(v) for v in s["enum"])
    if "oneOf" in s:
        return " or ".join(brief(x) for x in s["oneOf"])
    t = s.get("type")
    if t == "object" and "properties" in s:
        req = set(s.get("required", []))
        groups = []
        for k, v in s["properties"].items():
            r = brief(v)
            for g in groups:
                if g[1] == r and g[2] == (k in req):
                    g[0].append(k)
                    break
            else:
                groups.append([[k], r, k in req])
        return "{" + ", ".join(f"{'|'.join(n)}{'' if r_ else '?'}: {r}" for n, r, r_ in groups) + "}"
    if t == "array":
        n = []
        if "minItems" in s:
            n.append(f"min {s['minItems']}")
        if "maxItems" in s:
            n.append(f"max {s['maxItems']}")
        return f"[{brief(s.get('items', {}))}, …]" + (f" ({', '.join(n)})" if n else "")
    out = t if isinstance(t, str) else "value"
    if "minimum" in s or "maximum" in s:
        out += f" {s.get('minimum', '')}–{s.get('maximum', '')}"
    if s.get("description"):
        out += f" ({s['description']})"
    return out


DAYS = ("Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun")


def shift_text(d):
    if d is None:
        return "off"
    t = f"{d['start']}–{d['end']} @{d['assignment']}"
    if d.get("breaks"):
        t += " break " + ",".join(f"{a}–{b}" for a, b in d["breaks"])
    if d.get("covers"):
        t += " covers " + ",".join(f"{c['room']} {c['from']}–{c['until']}" for c in d["covers"])
    return t


def week_text(w):
    """A rota week, days with the same shift together: "Mon–Fri 07:45–16:15
    @desk break 12:00–12:30; Sat,Sun off"."""
    runs = []
    for d in DAYS:
        t = shift_text(w.get(d))
        if runs and runs[-1][1] == t:
            runs[-1][0].append(d)
        else:
            runs.append([[d], t])
    parts = []
    for days, t in runs:
        name = f"{days[0]}–{days[-1]}" if len(days) > 2 else ",".join(days)
        parts.append(f"{name} {t}")
    return "; ".join(parts)


def roster_text(r):
    a, b = week_text(r["A"]), week_text(r["B"])
    return f"A: {a} | B: {'same as A' if b == a else b}"


def roster_targets(targets):
    """set_roster targets with each roster as text (the JSON is the same
    shape as the action's roster field)."""
    out = []
    for t in targets:
        t = dict(t)
        vs = []
        for v in t.get("versions") or []:
            v = dict(v)
            if isinstance(v.get("roster"), dict):
                v["roster"] = roster_text(v["roster"])
            vs.append(v)
        if "versions" in t:
            t["versions"] = vs
        out.append(t)
    return out


def dates_in(obj, path=""):
    """The (path, value) of every date-valued or lead-time field in a
    catalog entry's targets: keys with earliest/latest or ending in Date,
    Day or Days."""
    out = []
    if isinstance(obj, dict):
        for k, v in obj.items():
            p = f"{path}.{k}" if path else k
            if isinstance(v, (dict, list)):
                out += dates_in(v, p)
            elif v is not None and ("earliest" in k.lower() or "latest" in k.lower()
                                    or k.endswith(("Date", "Day", "Days"))):
                out.append((p, v))
    elif isinstance(obj, list):
        for i, x in enumerate(obj):
            label = None
            if isinstance(x, dict):
                label = x.get("employee") or x.get("id") or "/".join([str(x.get("role"))] + [str(k) for k in x.get("skills") or []])
            out += dates_in(x, f"{path}[{label if label and label != 'None' else i}]")
    return out


def days_from(today, date):
    """Whole days from today to a YYYY-MM-DD date (None if not a date)."""
    try:
        return (datetime.date.fromisoformat(date) - datetime.date.fromisoformat(today)).days
    except (TypeError, ValueError):
        return None


def timing(today, dates):
    """Date and lead-time fields grouped by name and value: "earliestLastDay
    2026-10-27 (in 14 days): E01, E03"; lead times as their range."""
    by = {}
    for path, v in dates:
        name = path.rsplit(".", 1)[-1]
        owner = path[:-len(name) - 1] if "." in path else ""
        by.setdefault(name, {}).setdefault(compact(v), []).append(owner)
    out = []
    for name, vals in by.items():
        if all(isinstance(json.loads(v), (int, float)) for v in vals) and len(vals) > 1:
            nums = sorted(json.loads(v) for v in vals)
            out.append(f"{name} {nums[0]}–{nums[-1]}")
            continue
        parts = []
        for v, owners in vals.items():
            d = days_from(today, json.loads(v))
            who = [o.strip("[]") for o in owners if o]
            parts.append(f"{json.loads(v)}" + (f" (in {d} days)" if d is not None else "")
                         + (f" for {', '.join(who[:12])}" + (" …" if len(who) > 12 else "") if who and len(vals) > 1 else ""))
        out.append(f"{name} " + "; ".join(parts))
    return "; ".join(out)


def guidance(obs, catalog):
    """Every action type in the catalog: its batch, summary, fields,
    rules, timing and targets."""
    today = obs.get("date")
    p = obs.get("practice") or {}
    lines = ["Batch rules (catalog): " + " ".join(catalog.get("batch") or [])]
    ops = [t.get("type") for t in catalog.get("types", []) if t.get("type") not in STAFFING]
    staff = [t.get("type") for t in catalog.get("types", []) if t.get("type") in STAFFING]
    lines.append(f"This agent submits operations actions ({', '.join(ops)}) and staffing actions "
                 f"({', '.join(staff)}) as two batches at this decision point, operations first; "
                 "both must validate or neither is submitted. They are not one atomic unit.")
    lines.append(f"Operating hours {p.get('hours')}; scheduled close today {p.get('scheduledClose')}; "
                 f"booking horizon {p.get('bookingHorizonDays')} days. Today is {today} ({obs.get('weekday')}), "
                 f"now {obs.get('clock')}.")
    for t in catalog.get("types", []):
        name = t.get("type")
        lines.append("")
        lines.append(f"## {name} ({'staffing' if name in STAFFING else 'operations'} batch): {t.get('summary', '')}")
        lines.append(f"Fields: {brief(t.get('schema'))}")
        for c in t.get("constraints") or []:
            lines.append(f"Rule: {c}")
        dates = dates_in(t.get("targets"))
        if dates:
            lines.append("Timing (from targets): " + timing(today, dates))
        else:
            lines.append("Timing: the catalog's targets state no dates or lead times for this type; "
                         "the fields and rules above say what is known, and validate decides.")
        targets = t.get("targets")
        if name == "set_roster" and isinstance(targets, list):
            try:
                targets = roster_targets(targets)
                lines.append("(Rosters shown as text: \"Mon–Fri 07:45–16:15 @desk break 12:00–12:30\" is, "
                             "for each of those days, {\"start\":\"07:45\",\"end\":\"16:15\",\"breaks\":[[\"12:00\","
                             "\"12:30\"]],\"assignment\":\"desk\",\"covers\":[]}; \"off\" is null.)")
            except (KeyError, TypeError, ValueError, AttributeError):
                targets = t.get("targets")
        if targets in (None, [], {}):
            lines.append("Targets: none now.")
        else:
            lines.append(f"Targets ({name}): {compact(targets)}")
    return "\n".join(lines)


# --- memory --------------------------------------------------------------

def action_text(a):
    return clip(compact(a), 200)


def pending(obs, catalog):
    """Changes already made that have not yet taken effect, as the public
    observation and catalog show them: pending roster and allocation
    versions, staff not yet active or leaving, cancellable contracts and
    approved extensions."""
    out = []
    for t in catalog.get("types", []):
        if t.get("type") in ("set_roster", "set_reading_allocation") and isinstance(t.get("targets"), list):
            for e in t["targets"]:
                for v in e.get("versions") or []:
                    if isinstance(v, dict) and v.get("pending"):
                        v = dict(v)
                        if isinstance(v.get("roster"), dict):
                            try:
                                v["roster"] = roster_text(v["roster"])
                            except (KeyError, TypeError, AttributeError):
                                pass
                        out.append(f"{t['type']} {e.get('employee')}: {clip(compact(v), 300)}")
        if t.get("type") == "contract_teleradiology" and isinstance(t.get("targets"), dict):
            for c in t["targets"].get("cancellableContracts") or []:
                out.append(f"teleradiology contract: {clip(compact(c), 300)}")
    for s in obs.get("staff") or []:
        if isinstance(s, dict) and s.get("employment") not in (None, "active"):
            keep = {k: v for k, v in s.items() if k in ("id", "name", "role", "skills", "employment")
                    or k.endswith(("Date", "Day")) or "start" in k.lower() or "last" in k.lower()}
            out.append(f"staff {s.get('id')}: {clip(compact(keep), 300)}")
    if obs.get("extensions"):
        out.append(f"approved extensions: {clip(compact(obs['extensions']), 600)}")
    if obs.get("teleradiology"):
        out.append(f"teleradiology: {clip(compact(obs['teleradiology']), 600)}")
    return out


def metrics(point, obs):
    """One row of the memory's metric table."""
    mods = dig(obs, "demand", "modalities") or []
    items = dig(obs, "worklist", "items") or []
    ss = dig(obs, "flow", "sinceStart") or {}
    return {"point": point, "time": str(obs.get("time", ""))[5:16].replace("T", " "),
            "cash": money(dig(obs, "finance", "cashCents")),
            "profitMTD": money(dig(obs, "finance", "monthToDate", "operatingProfitCents")),
            "unread": dig(obs, "worklist", "unread"),
            "overdue": sum(1 for i in items if isinstance(i, dict) and i.get("overdue")),
            "lost": sum(dig(m, "sinceStart", "lost", default=0) or 0 for m in mods),
            "exams": ss.get("examsDone"), "overtimeMin": ss.get("overtimeMin"), "noCoverage": ss.get("noCoverage")}


METRIC_COLUMNS = ("point", "time", "cash", "profitMTD", "unread", "overdue", "lost", "exams", "overtimeMin",
                  "noCoverage")


class Memory:
    """The agent's memory in one run (llm-agent-memory/v1): its recent
    decisions in full, the earlier ones as totals, the validation refusals
    it has met, metric rows at recent decision points, and the pending
    changes the current observation and catalog show. Derived only from the
    public observation, the catalog and its own decision records; bounded
    by the settings (decisions, metric rows, characters)."""

    LESSONS = 8

    def __init__(self, decisions, metric_points, max_chars):
        self.n_decisions, self.n_metrics, self.max_chars = decisions, metric_points, max_chars
        self.entries = []
        self.earlier = None
        self.lessons = []  # [type, code, message, point], newest last, unique by (type, code)
        self.rows = []
        self.counts = None

    def observe(self, point, obs):
        """A decision point's observation: the outcome of the last decision
        (from the action log's counts and results) and a metric row."""
        log = obs.get("actions") or {}
        now = (log.get("accepted") or 0, log.get("rejected") or 0)
        if self.entries and self.entries[-1]["submitted"] and self.entries[-1]["applied"] is None and self.counts:
            last = self.entries[-1]
            results = [r for r in log.get("recent") or [] if isinstance(r, dict) and r.get("time") == last["time"]]
            last["applied"] = {"accepted": now[0] - self.counts[0], "rejected": now[1] - self.counts[1],
                               "results": [f"{r.get('code')}: {clip(r.get('message', ''), 160)}" for r in results[:4]]}
        self.counts = now
        if self.n_metrics:
            self.rows = (self.rows + [metrics(point, obs)])[-self.n_metrics:]

    def record(self, rec):
        """A decision point's record (live, or from a recording)."""
        submitted = [action_text(a) for b in rec.get("batches") or [] for a in b]
        refused = rec.get("refusals") or []
        self.entries.append({"point": rec["point"], "time": rec["time"], "outcome": rec.get("outcome"),
                             "reason": clip(rec.get("reason") or "", 200), "submitted": submitted,
                             "refused": [f"{r.get('type')} {r.get('code')}: {clip(r.get('message', ''), 160)}"
                                         for r in refused[:4]], "errors": len(rec.get("errors") or []),
                             "applied": None})
        for r in refused:
            if r.get("code") == "batch_rejected":
                continue
            self.lessons = [x for x in self.lessons if (x[0], x[1]) != (r.get("type"), r.get("code"))]
            self.lessons.append([r.get("type"), r.get("code"), clip(r.get("message", ""), 200), rec["point"]])
        self.lessons = self.lessons[-self.LESSONS:]
        while len(self.entries) > self.n_decisions:
            self.earlier = self._fold(self.earlier, self.entries.pop(0))

    @staticmethod
    def _fold(e, x):
        e = dict(e or {"from": x["point"], "points": 0, "no_action": 0, "actions": 0, "fallback": 0,
                       "submitted": 0, "accepted": 0, "rejected": 0, "refusedProposals": 0})
        e["to"] = x["point"]
        e["points"] += 1
        e[x["outcome"] if x["outcome"] in ("no_action", "actions", "fallback") else "fallback"] += 1
        e["submitted"] += len(x["submitted"])
        e["refusedProposals"] += len(x["refused"])
        if x["applied"]:
            e["accepted"] += x["applied"]["accepted"]
            e["rejected"] += x["applied"]["rejected"]
        return e

    def _render(self, k, rows, lessons, pend):
        """The memory text with the k most recent entries in full."""
        earlier, entries = self.earlier, self.entries
        for x in entries[:len(entries) - k]:
            earlier = self._fold(earlier, x)
        entries = entries[len(entries) - k:]
        out = [f"MEMORY ({MEMORY_FORMAT}: your own decisions in this run and what the public observation "
               "showed afterwards; bounded)"]
        if not self.entries and not self.earlier:
            out.append("No earlier decisions in this run.")
        if earlier:
            out.append(f"Earlier, points {earlier['from']}–{earlier['to']}: {earlier['points']} decisions "
                       f"({earlier['no_action']} no action, {earlier['actions']} with actions, {earlier['fallback']} "
                       f"fallbacks); {earlier['submitted']} actions submitted, {earlier['accepted']} accepted, "
                       f"{earlier['rejected']} refused when applied; {earlier['refusedProposals']} proposals "
                       "refused by validate.")
        if entries:
            out.append("Recent decisions:")
        for x in entries:
            line = f"- point {x['point']} ({x['time'][5:16].replace('T', ' ')}): {x['outcome']}"
            if x["reason"]:
                line += f" — \"{x['reason']}\""
            out.append(line)
            for r in x["refused"]:
                out.append(f"  refused by validate first: {r}")
            if x["outcome"] == "fallback":
                out.append(f"  no usable decision after {x['errors']} errors; nothing submitted")
            if x["submitted"]:
                out.append("  submitted: " + "; ".join(x["submitted"][:8])
                           + (f"; and {len(x['submitted']) - 8} more" if len(x["submitted"]) > 8 else ""))
                a = x["applied"]
                if a is None:
                    out.append("  applied: not yet observed")
                else:
                    out.append(f"  applied: {a['accepted']} accepted, {a['rejected']} refused"
                               + ("".join(f"; {r}" for r in a["results"])))
        if lessons:
            out.append("Validation refusals met in this run (most recent last):")
            for typ, code, msg, point in lessons:
                out.append(f"- {typ} {code} (point {point}): {msg}")
        out.append("Pending changes now (from the observation and catalog):")
        out += [f"- {p}" for p in pend] or ["- none shown"]
        if rows:
            out.append("Metrics at recent decision points (cash; operating profit month to date; unread studies; "
                       "overdue; lost demand, exams done, overtime minutes, no-coverage events since the start):")
            out.append(" | ".join(METRIC_COLUMNS))
            for r in rows:
                out.append(" | ".join(str(r[c]) for c in METRIC_COLUMNS))
        return "\n".join(out)

    def render(self, pend):
        """The memory text, within max_chars: older entries are folded into
        the totals first, then metric rows, refusals and pending lines are
        dropped, oldest first; a last resort cuts the text. With
        memoryDecisions 0 the memory is off: nothing from earlier decision
        points is shown, only the pending changes now."""
        if not self.n_decisions:
            return "\n".join([f"MEMORY ({MEMORY_FORMAT}): off in this run (memoryDecisions 0); nothing from "
                              "earlier decision points is carried over.",
                              "Pending changes now (from the observation and catalog):"]
                             + ([f"- {p}" for p in list(pend)[:12]] or ["- none shown"]))[:self.max_chars]
        rows, lessons, pend = list(self.rows), list(self.lessons), list(pend)[:12]
        k = len(self.entries)
        while True:
            text = self._render(k, rows, lessons, pend)
            if len(text) <= self.max_chars:
                return text
            if k:
                k -= 1
            elif rows:
                rows.pop(0)
            elif lessons:
                lessons.pop(0)
            elif pend:
                pend.pop()
            else:
                return text[:self.max_chars - 20] + "\n…(memory cut)"


# --- prompt and decision -------------------------------------------------

def build_prompt(point, obs, omitted, catalog, memory_text, settings):
    """The user message: the time, active problems, memory, observation
    summary, action guidance and what was left out. (None, left) when it
    is over maxPromptChars."""
    p = obs.get("practice") or {}
    head = (f"Decision point {point}: {obs.get('weekday')} {obs.get('date')} {obs.get('clock')} "
            f"(day {obs.get('day')}). Practice phase: {p.get('phase')}; hours {p.get('hours')}; scheduled close "
            f"today {p.get('scheduledClose')}.\nDecide what to change now, if anything.")
    left = [(k, f"{v} entries (history; left out by the protocol's compact observation)") for k, v in omitted.items()]
    if settings["observation"] == "summary":
        state, cut = summarize(obs)
        left += cut
        state_title = "STATE (the public observation, dreamteam.observation/v1, summarized: see LEFT OUT)"
    else:
        state = obs
        state_title = "STATE (the compact public observation, dreamteam.observation/v1)"
    parts = [head, "ACTIVE PROBLEMS\n" + "\n".join(f"- {x}" for x in problems(obs)), memory_text,
             state_title + "\n" + compact(state),
             "ACTIONS (every action type in the catalog)\n" + guidance(obs, catalog)]
    if settings["catalog"] == "full":
        parts.append("FULL ACTION CATALOG (JSON Schemas and targets)\n"
                     + compact({"batch": catalog.get("batch"), "types": catalog.get("types")}))
    else:
        left.append(("action JSON Schemas and raw targets", "summarized above as Fields and Targets; ask by "
                     "action type"))
    if left:
        how = "ask with \"details\" (once per decision point)" if settings["allowDetails"] else "not available"
        whole = [a for a, b in left if not b]
        lines = [f"- {a}: {b}" for a, b in left if b]
        if whole:
            lines.append("- left out entirely: " + ", ".join(whole))
        parts.append(f"LEFT OUT ({how}):\n" + "\n".join(lines))
    parts.append("Reply with the JSON decision object only.")
    text = "\n\n".join(parts)
    names = [a for a, _ in left]
    if len(text) > settings["maxPromptChars"]:
        return None, names
    return text, names


def details_text(h, names, obs, omitted, catalog, max_chars):
    """The details a model asked for: full catalog entries and left-out
    observation paths; a path the compact observation leaves out comes from
    the full observation's section (observe without compact). Each is cut
    to max_chars."""
    types = {t.get("type"): t for t in catalog.get("types", [])}
    out = []
    for n in names:
        if n in types:
            out.append(f"{n} (catalog entry): {clip(compact(types[n]), max_chars)}")
        elif n in omitted:
            o = h.request("observe", {"sections": [n.split(".")[0]], "recentLog": 0})["observation"]
            out.append(f"{n} (full observation): {clip(compact(full_path(o, n)), max_chars)}")
        elif full_path(obs, n) is not None:
            out.append(f"{n} (observation): {clip(compact(full_path(obs, n)), max_chars)}")
        else:
            out.append(f"{n}: not an action type or an observation path; nothing to show")
    return "Details you asked for:\n" + "\n".join(out) + "\n\nNow reply with the JSON decision object only " \
        "({\"reason\", \"actions\"}; no further details)."


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


def parse_decision(content, types, details_allowed=False):
    """The decision in a model reply, strictly: one JSON object with exactly
    "reason" (a string) and "actions" (a list of objects, each with a
    catalog type), and, when allowed, "details" (1–8 names, with no
    actions). Raises ValueError with a stable code."""
    try:
        d = json.loads(strip_wrapping(content))
    except json.JSONDecodeError as e:
        raise ValueError(f"invalid_json: the reply is not one JSON object ({e.msg})")
    if not isinstance(d, dict):
        raise ValueError("invalid_decision: the reply is not a JSON object")
    keys = set(d)
    if keys == {"reason", "actions", "details"}:
        if not details_allowed:
            raise ValueError("invalid_decision: \"details\" cannot be asked for now; reply with \"reason\" and "
                             "\"actions\" only")
        n = d["details"]
        if not isinstance(n, list) or not 1 <= len(n) <= MAX_DETAILS or not all(isinstance(x, str) for x in n):
            raise ValueError(f"invalid_decision: \"details\" must be a list of 1–{MAX_DETAILS} names")
        if d["actions"] != []:
            raise ValueError("invalid_decision: ask for \"details\" with \"actions\": []")
    elif keys != {"reason", "actions"}:
        raise ValueError(f"invalid_decision: the object must have exactly \"reason\" and \"actions\", not {sorted(d)}")
    if not isinstance(d["reason"], str):
        raise ValueError("invalid_decision: \"reason\" must be a string")
    if len(d["reason"]) > MAX_REASON:
        raise ValueError(f"invalid_decision: \"reason\" is longer than {MAX_REASON} characters")
    acts = d["actions"]
    if not isinstance(acts, list) or len(acts) > 2 * MAX_ACTIONS:
        raise ValueError(f"invalid_decision: \"actions\" must be a list of at most {MAX_ACTIONS} actions of each kind")
    for i, a in enumerate(acts):
        if not isinstance(a, dict) or not isinstance(a.get("type"), str):
            raise ValueError(f"invalid_action: action {i} is not an object with a \"type\"")
        if a["type"] not in types:
            raise ValueError(f"unknown_action: action {i} has type {a['type']!r}, not in the catalog")
    if sum(a["type"] in STAFFING for a in acts) > MAX_ACTIONS or sum(a["type"] not in STAFFING for a in acts) > MAX_ACTIONS:
        raise ValueError(f"invalid_decision: at most {MAX_ACTIONS} actions of each kind")
    return d


def split(actions):
    """The actions as batches: operations first, then staffing; each with
    the indices of its actions in the reply."""
    out = []
    for staffing in (False, True):
        idx = [i for i, a in enumerate(actions) if (a["type"] in STAFFING) == staffing]
        if idx:
            out.append(([actions[i] for i in idx], idx))
    return out


def refusals(validation, idx, attempt):
    """The refused results of a validate answer: [{attempt, action, type,
    code, message}], action numbered as in the model's reply."""
    out = []
    for k, r in enumerate(validation.get("results") or []):
        if not r.get("ok"):
            a = r.get("action") or {}
            out.append({"attempt": attempt, "action": idx[k] if k < len(idx) else k, "type": a.get("type"),
                        "code": r.get("code"), "message": r.get("message")})
    return out


class Evidence:
    """The agent's evidence files: agent.json (settings, without secrets),
    model.jsonl (every model exchange), decisions.jsonl (every decision
    point), memory.jsonl (the memory at every decision point), summary.json
    and usage.txt (usage and estimated cost)."""

    def __init__(self, path):
        self.dir = path
        os.makedirs(path, exist_ok=True)
        self.calls = []

    def write(self, name, obj):
        with open(os.path.join(self.dir, name), "w", encoding="utf-8") as f:
            json.dump(obj, f, indent=1, sort_keys=True)
            f.write("\n")

    def text(self, name, text):
        with open(os.path.join(self.dir, name), "w", encoding="utf-8") as f:
            f.write(text)

    def append(self, name, obj):
        # Keys in their own order: a recorded batch is replayed byte for byte.
        with open(os.path.join(self.dir, name), "a", encoding="utf-8") as f:
            f.write(json.dumps(obj, separators=(",", ":")) + "\n")

    def call(self, rec):
        self.calls.append(rec)
        self.append("model.jsonl", rec)


def prompt_settings(settings):
    return {k: settings[k] for k in PROMPT_SETTINGS}


def prepare(h, memory, point, pset):
    """The decision point's inputs, memory and prompt (live and replay
    alike): (obs, omitted, catalog, memory text, prompt, left out)."""
    obs, omitted, catalog = look(h)
    memory.observe(point, obs)
    mem = memory.render(pending(obs, catalog))
    prompt, left = build_prompt(point, obs, omitted, catalog, mem, pset)
    return obs, omitted, catalog, mem, prompt, left


def new_record(point, clock, mode, mem, prompt, left, pset):
    rec = {"point": point, "time": clock["time"], "day": clock["day"], "digests": clock["digests"], "mode": mode,
           "agentVersion": VERSION, "promptVersion": PROMPT_VERSION, "memoryFormat": MEMORY_FORMAT,
           "promptSettings": pset, "memorySha256": sha256(mem), "memoryChars": len(mem),
           "promptSha256": sha256(prompt) if prompt else None, "promptChars": len(prompt) if prompt else None,
           "leftOut": left}
    return rec


def ask(h, ev, client, settings, memory, point):
    """One live decision point: the batches to submit and the record."""
    start = time.monotonic()
    clock = h.request("digests")
    pset = prompt_settings(settings)
    obs, omitted, catalog, mem, prompt, left = prepare(h, memory, point, pset)
    rec = new_record(point, clock, "live", mem, prompt, left, pset)
    rec.update({"attempts": 0, "retries": 0, "detailsRequested": None, "errors": [], "refusals": [],
                "outcome": "fallback", "reason": None, "proposed": None, "validation": None, "batches": []})
    ev.append("memory.jsonl", {"point": point, "sha256": rec["memorySha256"], "chars": len(mem), "text": mem})
    if prompt is None:
        rec["errors"].append(f"prompt_too_large: over {settings['maxPromptChars']} characters")
        return [], rec
    types = {t["type"] for t in catalog.get("types", [])}
    messages = [{"role": "system", "content": settings["systemPrompt"]}, {"role": "user", "content": prompt}]
    failures, calls, details_open, turn = 0, 0, settings["allowDetails"], "first"
    while failures <= settings["maxRetries"]:
        if calls and time.monotonic() - start >= settings["decisionBudgetSeconds"]:
            rec["errors"].append(f"budget: {settings['decisionBudgetSeconds']} s spent; no further attempt")
            break
        calls += 1
        rec["attempts"] = calls
        payload = client.payload(messages)
        t0 = time.monotonic()
        status, body, err = client.call(payload)
        call = {"point": point, "attempt": calls, "turn": turn, "model": settings["model"],
                "endpoint": settings["endpoint"], "request": payload, "status": status,
                "latencyMs": round((time.monotonic() - t0) * 1000), "response": body[:MAX_RESPONSE_KEPT],
                "responseTruncated": len(body) > MAX_RESPONSE_KEPT, "usage": None, "servedModel": None,
                "content": None, "error": err}
        asked = None
        if err is None:
            try:
                content, call["usage"], call["servedModel"] = completion(body)
                call["content"] = content
                d = parse_decision(content, types, details_open)
                rec["reason"], rec["proposed"] = d["reason"], d["actions"]
                if "details" in d:
                    asked = d["details"]
                elif d["actions"]:
                    rec["validation"], refused = [], []
                    for batch, idx in split(d["actions"]):
                        v = h.request("validate", {"actions": batch})
                        rec["validation"].append(v)
                        if not v.get("valid"):
                            refused += refusals(v, idx, calls)
                    if refused:
                        rec["refusals"] += refused
                        raise ValueError("refused: " + "; ".join(
                            f"action {r['action']}: {r['code']}: {r['message']}" for r in refused))
                    rec["batches"], rec["outcome"] = [b for b, _ in split(d["actions"])], "actions"
                else:
                    rec["outcome"] = "no_action"
            except ProtocolError as e:
                call["error"] = f"invalid_action: validate answered {e.code}: {e.message}"
            except ValueError as e:
                call["error"] = str(e)
        ev.call(call)
        if asked is not None:
            details_open = False
            rec["detailsRequested"] = asked
            messages = messages + [{"role": "assistant", "content": call["content"]},
                                   {"role": "user", "content": details_text(h, asked, obs, omitted, catalog,
                                                                            settings["maxPromptChars"] // 4)}]
            turn = "details"
            continue
        rec["retries"] = sum(1 for c in ev.calls if c["point"] == point and c["turn"] == "retry")
        if call["error"] is None:
            return rec["batches"], rec
        failures += 1
        turn = "retry"
        rec["errors"].append(f"attempt {calls}: {call['error']}")
        if call["content"] is not None:  # the model answered: say what was wrong
            messages = messages + [
                {"role": "assistant", "content": call["content"]},
                {"role": "user", "content": f"Your reply could not be used: {call['error']}. Reply again with one "
                                            "JSON object {\"reason\": ..., \"actions\": [...]} that follows the "
                                            "action fields, targets and dates exactly, or with \"actions\": [] for "
                                            "no change."}]
    rec["retries"] = sum(1 for c in ev.calls if c["point"] == point and c["turn"] == "retry")
    rec["batches"], rec["outcome"] = [], "fallback"  # never a batch that was not validated
    return [], rec


# --- replay --------------------------------------------------------------

def load_recordings(paths):
    """Each recorded decisions file: (path, SHA-256, records). A recording
    of another memory format or prompt version is refused here."""
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
        for r in recs:
            got = (r.get("memoryFormat"), r.get("promptVersion"))
            if got != (None, None) and got != (MEMORY_FORMAT, PROMPT_VERSION):
                raise SettingsError(
                    f"replay {p}: version incompatible: decision point {r['point']} was recorded with memory format "
                    f"{got[0]} and prompt {got[1]}; this agent (llm-agent {VERSION}) replays {MEMORY_FORMAT} with "
                    f"{PROMPT_VERSION}, and recordings without memory (llm-agent 1, V0.5a)")
        if len({r.get("memoryFormat") for r in recs}) > 1:
            raise SettingsError(f"replay {p}: decision points with and without memory in one recording")
        out.append((p, hashlib.sha256(raw).hexdigest(), recs))
    return out


class Replayer:
    """The recorded decisions, checked turn by turn: the recording is the
    one whose first decision point and digests are this run's (several
    recordings may be given for a matrix); each later decision point must
    be the recorded one, with the recorded digests."""

    def __init__(self, recordings):
        self.recordings, self.chosen, self.i = recordings, None, 0
        self.memory = None  # made from the chosen recording's prompt settings

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
        if r.get("protocolError"):
            raise ReplayError(f"decision point {point} was recorded after a protocol error; it cannot be replayed")
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


def replay_point(h, ev, replayer, point):
    """One replayed decision point: the recorded batches, after checking
    the digests and, for a recording with memory, the memory and prompt
    rebuilt from this run's observation."""
    clock = h.request("digests")
    r = replayer.next(point, clock)
    rec = {"point": point, "time": clock["time"], "day": clock["day"], "digests": clock["digests"],
           "mode": "replay", "outcome": r.get("outcome"), "reason": r.get("reason"), "batches": r["batches"],
           "replayed": os.path.basename(replayer.chosen[0])}
    if r.get("memoryFormat") is None:
        return rec  # V0.5a: no memory
    pset = r.get("promptSettings")
    if not isinstance(pset, dict) or set(pset) != set(PROMPT_SETTINGS):
        raise ReplayError(f"decision point {point}: the recording has no usable promptSettings")
    if replayer.memory is None:
        replayer.memory = Memory(pset["memoryDecisions"], pset["memoryMetricPoints"], pset["memoryMaxChars"])
    memory = replayer.memory
    _, _, _, mem, prompt, left = prepare(h, memory, point, pset)
    ev.append("memory.jsonl", {"point": point, "sha256": sha256(mem), "chars": len(mem), "text": mem})
    if sha256(mem) != r.get("memorySha256"):
        raise ReplayError(f"decision point {point}: the memory rebuilt from this run differs from the recording's "
                          f"({MEMORY_FORMAT}): the memory or its formatting changed without a new format version")
    if (sha256(prompt) if prompt else None) != r.get("promptSha256"):
        raise ReplayError(f"decision point {point}: the prompt rebuilt from this run differs from the recording's "
                          f"({PROMPT_VERSION}): the prompt changed without a new prompt version")
    rec.update(new_record(point, clock, "replay", mem, prompt, left, pset))
    rec.update({"refusals": r.get("refusals") or [], "errors": r.get("errors") or [],
                "replayed": os.path.basename(replayer.chosen[0])})
    memory.record(r)
    return rec


# --- usage ---------------------------------------------------------------

def estimate_tokens(text):
    """A character-based estimate (4 characters a token), not a count."""
    return (len(text) + 3) // 4


def call_usage(c):
    """A model call's tokens: ("reported", prompt, completion) from the
    server, ("estimated", prompt, completion) from characters when the
    server answered without counts, or ("unknown", None, None)."""
    u = c.get("usage") or {}
    pt, ct = u.get("prompt_tokens"), u.get("completion_tokens")
    if isinstance(pt, int) and isinstance(ct, int) and not isinstance(pt, bool) and not isinstance(ct, bool):
        return "reported", pt, ct
    if c.get("content") is not None:
        sent = sum(len(m.get("content") or "") for m in c["request"].get("messages", []))
        return "estimated", (sent + 3) // 4, estimate_tokens(c["content"])
    return "unknown", None, None


def median(xs):
    xs = sorted(xs)
    if not xs:
        return None
    m = len(xs) // 2
    return xs[m] if len(xs) % 2 else round((xs[m - 1] + xs[m]) / 2)


def summary(ev, records, settings):
    """The run's model usage: requests, retries, tokens by how they are
    known, latency, decision outcomes and, with configured prices, the
    estimated cost."""
    calls = ev.calls
    tok = {"reported": {"requests": 0, "prompt": 0, "completion": 0},
           "estimated": {"requests": 0, "prompt": 0, "completion": 0}, "unknown": {"requests": 0}}
    for c in calls:
        kind, pt, ct = call_usage(c)
        tok[kind]["requests"] += 1
        if kind != "unknown":
            tok[kind]["prompt"] += pt
            tok[kind]["completion"] += ct
    for k in ("reported", "estimated"):
        tok[k]["total"] = tok[k]["prompt"] + tok[k]["completion"]
    lat = [c["latencyMs"] for c in calls]
    outcomes = {}
    for r in records:
        outcomes[r.get("outcome")] = outcomes.get(r.get("outcome"), 0) + 1
    out = {"model": settings["model"] if settings else None,
           "servedModels": sorted({c["servedModel"] for c in calls if c.get("servedModel")}),
           "decisionPoints": len(records), "outcomes": outcomes,
           "decisions": {"valid": outcomes.get("actions", 0) + outcomes.get("no_action", 0),
                         "withActions": outcomes.get("actions", 0), "noAction": outcomes.get("no_action", 0),
                         "refused": sum(1 for r in records if r.get("refusals")),
                         "fallback": outcomes.get("fallback", 0)},
           "requests": len(calls), "failedRequests": sum(1 for c in calls if c["error"]),
           "retries": sum(1 for c in calls if c.get("turn") == "retry"),
           "detailRequests": sum(1 for c in calls if c.get("turn") == "details"),
           "tokens": tok,
           "latencyMs": {"total": sum(lat), "mean": round(sum(lat) / len(lat)) if lat else None,
                         "median": median(lat), "max": max(lat) if lat else None},
           "cost": None}
    pr = settings and settings.get("pricing")
    if pr:
        known = tok["reported"]["prompt"] + tok["estimated"]["prompt"], \
            tok["reported"]["completion"] + tok["estimated"]["completion"]
        basis = "reported tokens" if not tok["estimated"]["requests"] else "reported and estimated tokens"
        out["cost"] = {"currency": pr["currency"], "inputPerMillion": pr["inputPerMillion"],
                       "outputPerMillion": pr["outputPerMillion"],
                       "estimated": round((known[0] * pr["inputPerMillion"] + known[1] * pr["outputPerMillion"]) / 1e6, 6),
                       "basis": f"{basis} × configured prices; an estimate, not provider billing",
                       "requestsWithUnknownUsage": tok["unknown"]["requests"]}
    return out


def usage_text(s):
    """summary.json as text."""
    t, lat, d = s["tokens"], s["latencyMs"], s["decisions"]
    lines = [f"Model usage ({AGENT} {VERSION})",
             f"model                 {s['model']}" + (f" (served as {', '.join(s['servedModels'])})"
                                                      if s["servedModels"] and s["servedModels"] != [s["model"]] else ""),
             f"decision points       {s['decisionPoints']}: {d['valid']} valid ({d['withActions']} with actions, "
             f"{d['noAction']} no action), {d['refused']} with a refused proposal, {d['fallback']} fallbacks",
             f"requests              {s['requests']} ({s['retries']} retries, {s['detailRequests']} detail requests, "
             f"{s['failedRequests']} failed)",
             f"tokens, reported      {t['reported']['requests']} requests: prompt {t['reported']['prompt']}, completion "
             f"{t['reported']['completion']}, total {t['reported']['total']}",
             f"tokens, estimated     {t['estimated']['requests']} requests: prompt {t['estimated']['prompt']}, completion "
             f"{t['estimated']['completion']}, total {t['estimated']['total']} (from characters, not counted)",
             f"tokens, unknown       {t['unknown']['requests']} requests",
             f"latency               mean {lat['mean']} ms, median {lat['median']} ms, max {lat['max']} ms; "
             f"inference {lat['total']} ms in all"]
    c = s["cost"]
    lines.append("cost                  no prices configured" if not c else
                 f"cost, estimated       {c['estimated']:.6f} {c['currency']} at {c['inputPerMillion']} input and "
                 f"{c['outputPerMillion']} output per million tokens ({c['basis']}; "
                 f"{c['requestsWithUnknownUsage']} requests of unknown usage not counted)")
    return "\n".join(lines) + "\n"


# --- main ----------------------------------------------------------------

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
            "memoryFormat": MEMORY_FORMAT, "mode": "replay" if recordings else "live",
            "python": sys.version.split()[0]}
    if settings:
        info.update({k: settings[k] for k in ("model", "endpoint", "temperature", "maxOutputTokens",
                                               "requestTimeoutSeconds", "maxRetries", "decisionBudgetSeconds",
                                               "responseFormat", "systemPromptSource", "pricing", *PROMPT_SETTINGS)})
        info["systemPromptSha256"] = sha256(settings["systemPrompt"])
        info["apiKey"] = "set" if settings["apiKey"] else "none"
    else:
        info["replay"] = [{"file": os.path.basename(p), "sha256": s} for p, s, _ in recordings]
    ev.write("agent.json", info)
    client = ModelClient(settings) if settings else None
    replayer = Replayer(recordings) if recordings else None
    memory = Memory(settings["memoryDecisions"], settings["memoryMetricPoints"],
                    settings["memoryMaxChars"]) if settings else None
    h = Harness(sys.stdin, sys.stdout)
    records, checked = [], False

    def finish():
        s = summary(ev, records, settings)
        ev.write("summary.json", s)
        ev.text("usage.txt", usage_text(s))
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
                rec = replay_point(h, ev, replayer, point)
                batches = rec["batches"]
            else:
                try:
                    batches, rec = ask(h, ev, client, settings, memory, point)
                except ProtocolError as e:
                    # Never expected (the agent uses permitted operations
                    # only); recorded as such, and refused by replay.
                    batches, rec = [], {"point": point, "time": None, "mode": "live", "agentVersion": VERSION,
                                        "promptVersion": PROMPT_VERSION, "memoryFormat": MEMORY_FORMAT,
                                        "protocolError": str(e), "outcome": "fallback", "batches": [],
                                        "errors": [f"protocol: {e}"], "attempts": 0, "digests": None}
            records.append(rec)
            if memory and rec.get("time") is not None:
                memory.record(rec)
            ev.append("decisions.jsonl", rec)
            finish()
            h.decide(batches)
    except (ExchangeError, ReplayError) as e:
        print(f"{AGENT}: {e}", file=sys.stderr)
        return 1
    finally:
        finish()


if __name__ == "__main__":
    sys.exit(main())
