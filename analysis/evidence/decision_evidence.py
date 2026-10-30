#!/usr/bin/env python3
"""Decision evidence: a read-only, deterministic analysis of recorded LLM
agent decisions (dreamteam.decision-evidence/v1).

    python3 -I -B analysis/evidence/decision_evidence.py \
        --corpus docs/benchmark/v08a-dev/corpus.json --replays DIR --out DIR

Run from the repository root. It reads, and never writes:
  * the committed recordings (each run's decisions.jsonl): the agent's
    reason and proposed actions, validation, refusals and errors;
  * the deterministic replays of those recordings (DIR/<replay>/runs/<id>/):
    the protocol transcript, which holds the public observation and action
    catalog of every decision point, and the harness's record of every
    submitted batch and its apply results.

From these it rebuilds, with the agent's own functions, the exact prompt of
every decision point and checks it against the recorded SHA-256, so an
observation-based check compares the reason with what the model was shown.
No model is called, no network is used, and nothing is executed beyond this
process. The analysis separates five things: what the agent said (the
reason), what it proposed, what validate answered, what was applied, and
what was observed afterwards. It checks only narrowly defined, explicit
statements, and abstains otherwise (docs/DECISION_EVIDENCE.md).

Writes evidence.json (the findings, coverage and rule summary),
traces.jsonl (one line per decision point: the five stages and the
consequence links) and tables.md (the tables of the analysis report).
"""

import argparse
import hashlib
import json
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, "..", "..", "agents", "llm"))
import llm_agent as A  # noqa: E402  (read-only use: prompt, memory, split)

SCHEMA = "dreamteam.decision-evidence/v1"
CORPUS_SCHEMA = "dreamteam.decision-evidence-corpus/v1"
ANALYZER = "decision-evidence 1"

RULES = {
    "EXPLICIT_ACTION_OMITTED": 1,
    "EXPLICIT_OPPOSITE_ACTION": 1,
    "FINANCIAL_AMOUNT_MISMATCH": 1,
    "REPEATED_INVALID_PROPOSAL": 1,
    "STATED_ACTION_NOT_EXECUTED": 1,
    "FACTUAL_OBSERVATION_MISMATCH": 1,
}
STATUSES = ("supported", "unresolved", "insufficient_evidence")


def sha256_bytes(b):
    return hashlib.sha256(b).hexdigest()


def stable_id(*parts):
    return "de1-" + sha256_bytes("|".join(str(p) for p in parts).encode("utf-8"))[:16]


# --- the reason as text --------------------------------------------------

ABBREV = re.compile(r"(?:\bDr|\bMr|\bMs|\bMrs|\be\.g|\bi\.e|\bvs|\betc|\bapprox|\bNo)\.$")


def sentences(text):
    """The reason's sentences, as (start, end) spans: split after . ! ? or ;
    followed by space, except after an abbreviation (Dr.) or inside a
    number ($55.7k)."""
    out, start = [], 0
    for m in re.finditer(r"[.!?;](?=\s+|$)", text):
        end = m.end()
        if ABBREV.search(text[start:end]):
            continue
        if text[start:end].strip():
            out.append((start, end))
        start = end
    if text[start:].strip():
        out.append((start, len(text)))
    trimmed = []
    for s, e in out:
        while s < e and text[s].isspace():
            s += 1
        trimmed.append((s, e))
    return trimmed


# A text that revises itself ("Wait, ...", "Actually, ...") states no single
# intention: every commitment in it is unresolved.
REVISION = re.compile(r"(?:^|[.!?;:]\s+|\(\s*|,\s+)(?:(?:Wait|Actually|Hmm|On second thought|Correction|Let me re|"
                      r"Let's re|Re-evaluating|Reconsidering|Scratch that)\b|No(?=,))")
# A clause that says the action cannot be done states no intention to do it.
NEGCAP = re.compile(r"\b(?:cannot|can't|can not|unable to|not able to|could not|couldn't|not possible|impossible|"
                    r"not valid|invalid)\b", re.I)
NEGCAP_VERB = re.compile(r"\b(?:cannot|can't|can not|unable to|not able to|could not|couldn't)\s+(?:\w+\s+){0,2}?"
                         r"(?=extend|hire|terminat|cancel|start|set|increas|reduc|decreas|rais|lower)", re.I)
# A setting already in effect: no change is needed to keep it.
ALREADY = re.compile(r"\balready\b", re.I)
# Hedges and conditions: a clause with one states no explicit intention.
HEDGE_WORDS = r"\b(?:if|unless|could|would|might|may|maybe|perhaps|possibly|consider|considering|either|" \
              r"try|trying|whether|hope|hoping)\b"
HEDGE = re.compile(HEDGE_WORDS + r"|\bor\b", re.I)
# "I will not extend A, B or C" excludes all three: "or" is no hedge there.
HEDGE_NEGATED = re.compile(HEDGE_WORDS, re.I)
# A later decision point: an intention for another time is not checked now.
DEFERRED = re.compile(r"\b(?:tomorrow|next (?:week|decision|point|time|step|day)|later|subsequent|in a future|"
                      r"at the next|once the|once it|once they|when the|after the (?:contract|hire|change))\b", re.I)

COMMIT = r"(?:\bI will|\bI['’]ll|\bI am going to|\bI['’]m going to|\b[Ww]e will|\b[Ww]e['’]ll)"
ADVERBS = r"(?:(?:also|now|then|immediately|again|instead|still|first|additionally|therefore|thus|only|today|" \
          r"simply|just|proactively|consequently|accordingly)\s+){0,3}"
PROGRESSIVE = r"(?:\bI am|\bI['’]m)\s+(?:also\s+|now\s+)?"

NAME_ITEM = (r"(?:(?:Dr\.?\s+)?[A-Z][a-z]+(?:\s*\((?:[EH]\d{2,5})\))?|[EH]\d{2,5}(?:\s*\([A-Z][a-z. ]+\))?)"
             r"(?:'s|’s)?")
NAME_LIST = NAME_ITEM + r"(?:\s*,\s*(?:(?:and|or)\s+)?" + NAME_ITEM + r"|\s+(?:and|or)\s+" + NAME_ITEM + r")*"
NAME_TOKEN = re.compile(r"(?:Dr\.?\s+)?[A-Z][a-z]+|[EH]\d{2,5}")
PRONOUN = re.compile(r"^(?:them|him|her|it|their|his|these|those|both|all|everyone|staff|shifts?|the shifts?)\b",
                     re.I)

# Verbs, as the start of the object after a commitment marker. Each is
# (kind, regex on the text after the marker). The regex must match at the
# verb; its groups say the object.
VERBS = [
    ("extend_shift", re.compile(r"extend(?:ing)?\s+(?:the\s+shifts?\s+(?:of|for)\s+|shifts?\s+(?:of|for)\s+)?(?P<obj>.*)",
                                re.S)),
    ("hire", re.compile(r"hir(?:e|ing)\b(?P<obj>.*)", re.S)),
    ("terminate", re.compile(r"(?:terminat(?:e|ing)|fir(?:e|ing)|lay(?:ing)? off|let(?:ting)? go(?: of)?)\s+(?P<obj>.*)",
                             re.S)),
    ("telerad_cancel", re.compile(r"(?:cancel(?:l?ing)?|end(?:ing)?|stop(?:ping)?)\s+(?P<obj>.*)", re.S)),
    ("telerad_start", re.compile(r"(?:start(?:ing)?|initiat(?:e|ing)|set(?:ting)? up|sign(?:ing)?|establish(?:ing)?|"
                                 r"activat(?:e|ing)|add(?:ing)?|begin(?:ning)?|contract(?:ing)?|engag(?:e|ing))\s+"
                                 r"(?P<obj>.*)", re.S)),
    ("allocation", re.compile(r"(?P<verb>increas(?:e|ing)|rais(?:e|ing)|maximi[sz](?:e|ing)|boost(?:ing)?|"
                              r"decreas(?:e|ing)|reduc(?:e|ing)|lower(?:ing)?|cut(?:ting)?|set(?:ting)?|"
                              r"chang(?:e|ing)|adjust(?:ing)?)\s+(?P<obj>.*)", re.S)),
    ("policy", re.compile(r"(?:set(?:ting)?|chang(?:e|ing)|switch(?:ing)?|updat(?:e|ing))\s+(?P<obj>.*)", re.S)),
    ("roster", re.compile(r"(?:set(?:ting)?|chang(?:e|ing)|adjust(?:ing)?|updat(?:e|ing)|modify(?:ing)?)\s+(?P<obj>.*)",
                          re.S)),
]
UP = {"increas", "rais", "maximi", "boost"}
DOWN = {"decreas", "reduc", "lower", "cut"}
TELERAD = re.compile(r"\b(?:teleradiology|tele-radiology|tel-rad|telerad)\b", re.I)
CONTRACT_ID = re.compile(r"\bT\d{5}\b")
ROLE = re.compile(r"\b(technologists?|techs?|radiologists?|front[- ]desk)\b", re.I)
WORKLIST = ("urgent_then_fifo", "fifo", "oldest_first", "shortest_first")

NO_ACTION = re.compile(
    COMMIT + r"\s+" + ADVERBS + r"(?:take no (?P<q1>immediate |further |additional |other |new |more )?actions?|"
    r"do nothing|make no (?P<q2>immediate |further |additional |other |new |more )?changes?|"
    r"not (?:make|take) any (?P<q3>immediate |further |additional |other |new |more )?(?:changes?|actions?))"
    r"|" + PROGRESSIVE + r"taking no (?P<q4>immediate |further |additional |other |new |more )?actions?")
NO_ACTION_SCOPE = re.compile(r"^\s*(?:on|regarding|about|to the|with respect to|for (?!now\b|today\b|the moment\b|"
                             r"this point\b|the time being\b))", re.I)
NEGATED = re.compile(COMMIT + r"\s+" + ADVERBS + r"not\s+(?:attempt to\s+|try to\s+)?(?P<rest>.*)", re.S)


def staff_index(obs):
    """Names and IDs of the practice's staff, as the observation shows
    them: lower-case key -> set of employee IDs."""
    idx = {}
    for s in (obs or {}).get("staff") or []:
        if not isinstance(s, dict) or not s.get("id"):
            continue
        sid = s["id"]
        keys = {sid.lower()}
        name = str(s.get("name") or "")
        if name:
            keys.add(name.lower())
            keys.add(name.lower().replace("dr. ", "dr "))
            last = name.split()[-1].lower()
            keys.add(last)
        for k in keys:
            idx.setdefault(k, set()).add(sid)
    return idx


def resolve_names(text, staff):
    """The employee IDs a name list names: (ids, unresolved names)."""
    ids, bad = [], []
    for m in NAME_TOKEN.finditer(text):
        tok = m.group(0)
        key = tok.lower().replace("dr. ", "dr ")
        hit = staff.get(key) or staff.get(key.split()[-1])
        if hit and len(hit) == 1:
            i = next(iter(hit))
            if i not in ids:
                ids.append(i)
        else:
            bad.append(tok)
    return ids, bad


def name_list_at(obj):
    """The name list at the start of an object, or None."""
    m = re.match(NAME_LIST, obj)
    return m.group(0) if m else None


def statement(kind, span, clause, **kw):
    s = {"kind": kind, "span": span, "text": clause, "type": None, "targets": [], "unresolvedNames": [],
         "params": {}, "flags": []}
    s.update(kw)
    return s


def parse_object(kind, verb_m, staff):
    """The action type, targets and parameters an object names, or None if
    the verb is not this kind of action here."""
    obj = verb_m.group("obj")
    head = re.split(r"\s+(?:to|so|in order|because|since|as|which|that|while|before|until)\s+|[,;:]", obj, maxsplit=1)[0]
    if kind == "extend_shift":
        if PRONOUN.match(obj):
            return {"type": "extend_shift", "unresolved": "pronoun or group as the target"}
        names = name_list_at(obj)
        if not names:
            return {"type": "extend_shift", "unresolved": "no named target"}
        ids, bad = resolve_names(names, staff)
        return {"type": "extend_shift", "targets": ids, "bad": bad}
    if kind == "hire":
        if TELERAD.search(head) or re.search(r"teleradiolog", head, re.I):
            return {"type": "hire", "unresolved": "a teleradiology reader, not a staff hire"}
        role = ROLE.search(head)
        r = None
        if role:
            w = role.group(1).lower()
            r = "radiologist" if w.startswith("radiolog") else "front_desk" if w.startswith("front") else "technologist"
        return {"type": "hire", "params": {"role": r} if r else {}}
    if kind == "terminate":
        if PRONOUN.match(obj):
            return {"type": "terminate", "unresolved": "pronoun or group as the target"}
        names = name_list_at(obj)
        if names:
            ids, bad = resolve_names(names, staff)
            return {"type": "terminate", "targets": ids, "bad": bad}
        idm = re.findall(r"\b[EH]\d{2,5}\b", head)
        if len(set(idm)) == 1:
            ids, bad = resolve_names(idm[0], staff)
            return {"type": "terminate", "targets": ids, "bad": bad}
        return {"type": "terminate", "unresolved": "no named target"}
    if kind in ("telerad_cancel", "telerad_start"):
        window = " ".join(obj.split()[:6])
        if not (TELERAD.search(window) or (kind == "telerad_cancel" and re.match(
                r"(?:the\s+|it\b|this\s+|that\s+)?(?:\w+\s+){0,3}contract\b|T\d{5}", obj))):
            return None
        p = {"op": "cancel" if kind == "telerad_cancel" else "start"}
        c = CONTRACT_ID.search(" ".join(obj.split()[:8]))
        if c:
            p["contract"] = c.group(0)
        if kind == "telerad_cancel" and re.match(r"it\b", obj):
            return {"type": "contract_teleradiology", "params": p, "unresolved": "pronoun as the contract"}
        return {"type": "contract_teleradiology", "params": p}
    if kind == "allocation":
        if "allocation" not in " ".join(obj.split()[:7]).lower():
            return None
        v = verb_m.group("verb").lower()
        direction = "up" if any(v.startswith(x) for x in UP) else "down" if any(v.startswith(x) for x in DOWN) else None
        names = re.match(r"(?:the\s+(?:site\s+|reading\s+|site reading\s+)?allocation\s+(?:of|for)\s+)?(" + NAME_LIST + ")",
                         obj)
        if not names or "allocation" in names.group(1).lower():
            return {"type": "set_reading_allocation", "params": {"direction": direction},
                    "unresolved": "no named target"}
        ids, bad = resolve_names(names.group(1), staff)
        return {"type": "set_reading_allocation", "targets": ids, "bad": bad, "params": {"direction": direction}}
    if kind == "policy":
        w = " ".join(obj.split()[:8]).lower()
        if "worklist" in w and "policy" in w:
            vals = [x for x in WORKLIST if re.search(r"\b" + x + r"\b", obj.split(".")[0])]
            if len(vals) == 1:
                return {"type": "set_worklist_policy", "params": {"policy": vals[0]}}
            return {"type": "set_worklist_policy", "unresolved": "no single policy value named"}
        if re.search(r"walk-?in", w) and "policy" in w:
            return {"type": "set_walkin_policy"}
        return None
    if kind == "roster":
        m = re.match(r"(" + NAME_LIST + r")\s+(?:roster|rota|schedule)\b", obj)
        if not m:
            return None
        ids, bad = resolve_names(m.group(1), staff)
        return {"type": "set_roster", "targets": ids, "bad": bad}
    return None


def verb_at(rest, staff):
    """The action a commitment's text names at its start, or None."""
    for kind, rx in VERBS:
        m = rx.match(rest)
        if m:
            p = parse_object(kind, m, staff)
            if p is not None:
                return p, m
    return None, None


def extract(reason, staff):
    """The reason's explicit statements of intention (dreamteam
    .decision-evidence/v1 statements): commitments to an action, explicit
    refusals to act ("I will not extend Blake"), and statements of no
    action. Everything else is not a statement here: objectives ("reduce
    costs"), descriptions ("Extending her shift will..."), needs ("I need
    to"). Returns (statements, revised)."""
    out = []
    if not reason:
        return out, False
    revised = bool(REVISION.search(reason))
    for s, e in sentences(reason):
        clause = reason[s:e]
        flags = []
        if HEDGE.search(clause):
            flags.append("hedged")
        if clause.rstrip().endswith("?"):
            flags.append("question")
        if NEGCAP.search(clause):
            flags.append("says_cannot")
        if ALREADY.search(clause):
            flags.append("already")
        if DEFERRED.search(clause):
            flags.append("deferred")
        if revised:
            flags.append("revised_in_text")
        for m in NO_ACTION.finditer(clause):
            q = next((g for g in m.groups() if g), None)
            scoped = bool(NO_ACTION_SCOPE.match(clause[m.end():]))
            out.append(statement("no_action", [s + m.start(), s + m.end()], clause[m.start():m.end()],
                                 flags=flags + (["qualified"] if q or scoped else [])))
        nflags = [f for f in flags if f != "hedged"] + (["hedged"] if HEDGE_NEGATED.search(clause) else [])
        for m in NEGATED.finditer(clause):
            p, vm = verb_at(m.group("rest"), staff)
            if p:
                out.append(_from_parse("negated", p, s + m.start(), s + m.end(), clause, m.start(), nflags))
        starts = [m for m in re.finditer(COMMIT + r"\s+" + ADVERBS, clause)] + \
                 [m for m in re.finditer(PROGRESSIVE, clause)]
        for m in sorted(starts, key=lambda x: x.start()):
            rest = clause[m.end():]
            if rest.startswith("not ") or rest.startswith("not\n"):
                continue
            p, vm = verb_at(rest, staff)
            if not p:
                continue
            st = _from_parse("commit", p, s + m.start(), s + len(clause), clause, m.start(), flags)
            out.append(st)
            # Verbs coordinated with this commitment: "..., and extend Morgan's shift".
            for c in re.finditer(r"(?:,\s*|\s)(?:and|then)\s+(?:also\s+)?(?=\w)", rest):
                p2, _ = verb_at(rest[c.end():], staff)
                if p2:
                    out.append(_from_parse("commit", p2, s + m.end() + c.end(), s + len(clause), clause,
                                           m.end() + c.end(), flags))
    # An action the reason says elsewhere cannot be done ("Since I cannot
    # extend Morgan further today") is retracted there.
    cannot = []
    for m in NEGCAP_VERB.finditer(reason):
        p, _ = verb_at(reason[m.end():], staff)
        if p and p.get("type"):
            cannot.append((p["type"], set(p.get("targets") or [])))
    for st in out:
        if st["kind"] == "commit" and any(t == st["type"] and (not ids or not st["targets"] or ids & set(st["targets"]))
                                          for t, ids in cannot):
            st["flags"].append("retracted_elsewhere")
    # One statement per kind, type, targets and start.
    seen, uniq = set(), []
    for st in out:
        k = (st["kind"], st["type"], tuple(st["targets"]), st["span"][0], json.dumps(st["params"], sort_keys=True))
        if k not in seen:
            seen.add(k)
            uniq.append(st)
    uniq.sort(key=lambda x: (x["span"][0], x["kind"], x["type"] or ""))
    return uniq, revised


def _from_parse(kind, p, start, end, clause, local_start, flags):
    text = clause[local_start:].strip()
    st = statement(kind, [start, min(end, start + len(text))], text[:300], type=p.get("type"),
                   targets=list(p.get("targets") or []), unresolvedNames=list(p.get("bad") or []),
                   params=dict(p.get("params") or {}), flags=list(flags))
    if p.get("unresolved"):
        st["flags"].append("unresolved_target")
        st["why"] = p["unresolved"]
    if st["unresolvedNames"]:
        st["flags"].append("unresolved_target")
        st["why"] = "names not matched to one staff member: " + ", ".join(st["unresolvedNames"])
    return st


# --- matching statements with the proposed actions ------------------------

def matches(st, a):
    """Whether proposed action a is the action statement st names."""
    t = st["type"]
    if a.get("type") != t:
        return False
    p = st["params"]
    if t in ("extend_shift", "terminate", "set_roster", "set_reading_allocation"):
        return not st["targets"] or a.get("employee") in st["targets"]
    if t == "contract_teleradiology":
        if a.get("op") != p.get("op"):
            return False
        return not p.get("contract") or p.get("op") != "cancel" or a.get("contract") == p["contract"]
    if t == "set_worklist_policy":
        return not p.get("policy") or a.get("policy") == p["policy"]
    if t == "hire":
        return not p.get("role") or a.get("role") == p["role"]
    return True


def per_target(st):
    """A statement with several named targets checked as one statement per
    target: "extend Riley and Morgan" is two intentions."""
    if len(st["targets"]) <= 1:
        return [st]
    out = []
    for t in st["targets"]:
        x = dict(st)
        x["targets"] = [t]
        out.append(x)
    return out


def current_allocation(catalog, emp):
    for t in (catalog or {}).get("types") or []:
        if t.get("type") == "set_reading_allocation" and isinstance(t.get("targets"), list):
            for e in t["targets"]:
                if e.get("employee") == emp and isinstance(e.get("inEffectBP"), int):
                    return e["inEffectBP"]
    return None


# --- execution: validation, submission, apply -----------------------------

def execution(rec, applied):
    """Each proposed action's path: validate, submission, apply. applied is
    the harness's record of this decision point's batches (None if no
    replay is available). Returns a list aligned with rec["proposed"]."""
    proposed = rec.get("proposed") or []
    out = [{"index": i, "validate": None, "submitted": False, "apply": None, "state": "unknown"}
           for i in range(len(proposed))]
    if not proposed:
        return out
    try:
        parts = A.split(proposed)
    except (KeyError, TypeError):
        return out
    val = rec.get("validation")
    if isinstance(val, list):
        for j, (_, idx) in enumerate(parts):
            if j < len(val) and isinstance(val[j], dict):
                for k, r in enumerate(val[j].get("results") or []):
                    if k < len(idx):
                        out[idx[k]]["validate"] = {"ok": bool(r.get("ok")), "code": r.get("code")}
    if rec.get("outcome") == "actions":
        for j, (_, idx) in enumerate(parts):
            for k, i in enumerate(idx):
                out[i]["submitted"] = True
                if applied is not None and j < len(applied):
                    res = applied[j].get("results") or []
                    if k < len(res):
                        r = res[k]
                        out[i]["apply"] = {"id": r.get("id"), "ok": bool(r.get("ok")), "code": r.get("code"),
                                           "message": r.get("message")}
        for x in out:
            if x["apply"] is None:
                x["state"] = "submitted_apply_unknown"
            elif x["apply"]["ok"]:
                x["state"] = "accepted" if x["apply"]["code"] != "unchanged" else "accepted_unchanged"
            else:
                x["state"] = "refused_at_apply"
        return out
    for x in out:
        v = x["validate"]
        if v is None:
            x["state"] = "not_submitted_not_validated"
        elif not v["ok"]:
            x["state"] = "refused_at_validate" if v["code"] != "batch_rejected" else "not_submitted_batch_rejected"
        else:
            x["state"] = "not_submitted_other_refused"
    return out


EXECUTED = ("accepted", "accepted_unchanged")


# --- amounts -------------------------------------------------------------

AMOUNT = re.compile(r"(?P<sign>[-−–]\s?)?(?P<approx>[~≈]\s?)?(?P<sign2>[-−–])?\$\s?(?P<sign3>-)?"
                    r"(?P<num>\d{1,3}(?:,\d{3})+|\d+)(?P<dec>\.\d+)?\s?(?P<suf>[kKmM](?![A-Za-z]))?")
APPROX_WORDS = re.compile(r"\b(?:about|approximately|around|roughly|nearly|almost|some|circa)\s*\(?\s*$", re.I)
OVER_WORDS = re.compile(r"\b(?:over|more than|above|exceeding|at least)\s*\(?\s*$", re.I)
UNDER_WORDS = re.compile(r"\b(?:under|less than|below|at most)\s*\(?\s*$", re.I)
HISTORIC = re.compile(r"\b(?:from|was|were|had been|previously|yesterday|earlier|last week|initially)\b[^$]{0,20}$",
                      re.I)

QUANTITIES = [
    # (quantity, left-context regex at the end, right-context regex at the start)
    ("profit_mtd", re.compile(r"(?:(?:operating\s+)?(?:profit|loss|losses|result)\s+(?:month[- ]to[- ]date|MTD|this month|"
                              r"for the month)|(?:month[- ]to[- ]date|MTD)\s+(?:operating\s+)?(?:profit|loss|result))"
                              r"(?:\s+(?:is|of|at|stands at|was|currently|significantly|heavily|deeply|negative|"
                              r"positive))*\s*[(:]?\s*$", re.I),
     re.compile(r"^\s*\)?\s*(?:MTD\b|month[- ]to[- ]date|this month\b|so far this month)", re.I)),
    ("cash", re.compile(r"\bcash(?:\s+(?:balance|position|on hand|reserves?))?"
                        r"(?:\s+(?:is|of|at|stands at|remains|sits at|has (?:fallen|dropped|declined|risen) to|now|"
                        r"currently|still|only|critically|very|extremely|relatively|dangerously|healthy|low|stable|"
                        r"strong|positive|negative|tight|limited|depleted|adequate|sufficient|good|solid|at))*"
                        r"\s*[(:]?\s*$", re.I),
     re.compile(r"^\s*(?:in cash|of cash|cash on hand|cash balance)\b", re.I)),
    ("payroll_due", re.compile(r"\bpayroll(?:\s+(?:is\s+)?due)?(?:\s+(?:in \d+ days?|on [\w-]+|soon|tomorrow|today|"
                               r"this week))?(?:\s+(?:of|is|at))?\s*[(:]?\s*$", re.I),
     re.compile(r"^\s*(?:in\s+)?payroll\b", re.I)),
    ("receivables", re.compile(r"\breceivables(?:\s+(?:of|are|is|at|outstanding|totaling|totalling))*\s*[(:]?\s*$",
                               re.I),
     re.compile(r"^\s*(?:in|of)\s+(?:outstanding\s+)?receivables\b", re.I)),
    ("payables_due", re.compile(r"\b(?:bills|payables)(?:\s+(?:are\s+)?due)?(?:\s+(?:on\s+)?\d{1,4}[-/]\d{1,2}"
                                r"(?:[-/]\d{1,2})?|\s+on [\w-]+|\s+in \d+ days?|\s+soon)?"
                                r"(?:\s+(?:of|is|are|at))?\s*[(:]?\s*$", re.I),
     re.compile(r"^\s*(?:in\s+)?(?:bills|payables)\b", re.I)),
    ("teleradiology_retainer", re.compile(r"\bretainers?\b[^$.;]{0,40}$", re.I),
     re.compile(r"^\s*(?:/\s*mo(?:nth)?|per month|a month|monthly)?\s*\)?\s*(?:teleradiology\s+)?retainers?\b", re.I)),
]
PERIOD_SUFFIX = re.compile(r"^\s*\)?\s*(?:/\s*(?:mo|month|wk|week|day|yr|year|hr|hour|h)\b|per\s+\w+|a\s+(?:month|week|"
                           r"day|year)|monthly|weekly|daily|annually)", re.I)


def amounts(reason):
    """Every dollar amount in the reason: its span, value in cents, the
    precision it was stated with, and any comparator."""
    out = []
    for m in AMOUNT.finditer(reason or ""):
        num = int(m.group("num").replace(",", ""))
        dec = m.group("dec") or ""
        suf = (m.group("suf") or "").lower()
        mult = {"k": 1000, "m": 1000000}.get(suf, 1)
        frac = int(dec[1:]) if dec else 0
        cents = num * 100 * mult + (frac * 100 * mult) // (10 ** len(dec[1:])) if dec else num * 100 * mult
        if dec:
            prec = max(1, 100 * mult // (10 ** len(dec[1:])))
        elif suf:
            prec = 100 * mult
        else:
            tz = len(m.group("num").replace(",", "")) - len(m.group("num").replace(",", "").rstrip("0"))
            prec = 100 * 10 ** tz if num else 100
        neg = bool(m.group("sign") or m.group("sign2") or m.group("sign3"))
        left = reason[max(0, m.start() - 90):m.start()]
        left = re.split(r"[.;]\s|\$[\d,]+", left)[-1]
        comp = "approx" if m.group("approx") or APPROX_WORDS.search(left) else \
            "over" if OVER_WORDS.search(left) else "under" if UNDER_WORDS.search(left) else "equal"
        out.append({"span": [m.start(), m.end()], "text": m.group(0).strip(), "cents": -cents if neg else cents,
                    "negative": neg, "precisionCents": prec, "comparator": comp,
                    "left": left, "right": reason[m.end():m.end() + 40]})
    return out


def bind(a, clause_text):
    """The observed quantity an amount is stated as, or (None, why)."""
    if PERIOD_SUFFIX.match(a["right"]) and not re.search(r"retainer", a["left"] + a["right"], re.I):
        return None, "a rate (per month, week, day or hour), not an observed amount"
    if HISTORIC.search(a["left"]):
        return None, "an earlier value (from, was, previously), not the current observation"
    for q, lrx, rrx in QUANTITIES:
        if q == "teleradiology_retainer":
            if lrx.search(a["left"]) or rrx.match(a["right"]):
                return q, None
            continue
        if rrx.match(a["right"]) or lrx.search(a["left"]):
            if q == "cash" and re.search(r"cash\s*flow|bleeding cash|burn", a["left"], re.I):
                continue
            return q, None
    return None, "no quantity rule names this amount"


def observed_amounts(q, obs, catalog, clause_text):
    """The observed value(s) for a quantity, in cents, with their
    observation paths. A list: more than one value is a set of
    alternatives (the retainer of each plan window, or of each contract)."""
    f = (obs or {}).get("finance") or {}
    if q == "cash":
        v = f.get("cashCents")
        return [("finance.cashCents", v)] if isinstance(v, int) else []
    if q == "profit_mtd":
        v = A.dig(f, "monthToDate", "operatingProfitCents")
        return [("finance.monthToDate.operatingProfitCents", v)] if isinstance(v, int) else []
    if q == "payroll_due":
        out = []
        v = A.dig(f, "nextPayroll", "cents")
        if isinstance(v, int):
            out.append(("finance.nextPayroll.cents", v))
        v = f.get("accruedPayrollCents")
        if isinstance(v, int):
            out.append(("finance.accruedPayrollCents", v))
        return out
    if q == "receivables":
        v = f.get("arCents")
        return [("finance.arCents", v)] if isinstance(v, int) else []
    if q == "payables_due":
        out = []
        v = A.dig(f, "nextPayables", "cents")
        if isinstance(v, int):
            out.append(("finance.nextPayables.cents", v))
        v = f.get("apCents")
        if isinstance(v, int):
            out.append(("finance.apCents", v))
        return out
    if q == "teleradiology_retainer":
        targets = None
        for t in (catalog or {}).get("types") or []:
            if t.get("type") == "contract_teleradiology":
                targets = t.get("targets")
        if not isinstance(targets, dict):
            return []
        named = CONTRACT_ID.findall(clause_text)
        contracts = targets.get("cancellableContracts") or []
        out = []
        for c in contracts:
            if isinstance(c.get("monthlyRetainerCents"), int) and (not named or c.get("id") in named):
                out.append((f"contract_teleradiology.targets.cancellableContracts[{c.get('id')}].monthlyRetainerCents",
                            c["monthlyRetainerCents"]))
        if named:
            return out
        if len(out) > 1:
            out.append(("sum of cancellable contracts' monthlyRetainerCents", sum(v for _, v in out)))
        for p in targets.get("plans") or []:
            rs = [(r.get("window"), r.get("monthlyCents")) for r in p.get("retainers") or []
                  if isinstance(r.get("monthlyCents"), int)]
            for w, v in rs:
                out.append((f"contract_teleradiology.targets.plans[{p.get('id')}].retainers[{w}].monthlyCents", v))
            n = len(rs)
            for mask in range(1, 1 << n):
                if bin(mask).count("1") > 1:
                    ws = [rs[i] for i in range(n) if mask >> i & 1]
                    out.append((f"plans[{p.get('id')}] retainers {'+'.join(w for w, _ in ws)}", sum(v for _, v in ws)))
        return out
    return []


def agrees(a, v, signless=False):
    """Whether stated amount a agrees with observed cents v, at the
    precision it was stated with (rounded or truncated). With signless (a
    profit), a loss may be stated without its sign; otherwise the signs
    must agree."""
    s = abs(a["cents"])
    o = abs(v)
    if a["negative"] and v > 0:
        return False
    if not signless and not a["negative"] and v < 0 and s:
        return False
    p = a["precisionCents"]
    if a["comparator"] == "over":
        return o >= s - p
    if a["comparator"] == "under":
        return o <= s + p
    tol = p
    if a["comparator"] == "approx":
        tol = max(p, s // 10)
    return abs(o - s) < tol


def unit_error(a, v):
    """A stated amount that is the observed one with cents read as dollars
    (x100) or dollars as cents (/100)."""
    s = abs(a["cents"])
    for f in (100, 0.01):
        x = dict(a)
        x["cents"] = int(round(s / f))
        x["precisionCents"] = max(1, int(round(a["precisionCents"] / f)))
        x["negative"] = False
        if v and agrees(x, abs(v)):
            return "cents_read_as_dollars" if f == 100 else "dollars_read_as_cents"
    return None


def shown(prompt, path, value):
    """Whether the prompt shows this observed value: as raw cents in the
    state ("cashCents":N) or as whole dollars in the problems."""
    if not prompt:
        return False
    key = path.rsplit(".", 1)[-1] if "." in path else path
    key = re.sub(r"\[.*?\]", "", key)
    if f'"{key}":{value}' in prompt:
        return True
    if f"{value}" in prompt and ("retainer" in path.lower() or "plans[" in path or "cents" in key.lower()):
        return True
    return A.money(value) in prompt


def stated_in_prompt(a, prompt):
    """Where the prompt shows a stated amount exactly (as whole dollars, or
    as cents in the state), or None. Only amounts stated to the dollar."""
    if not prompt or a["precisionCents"] > 100 or not a["cents"]:
        return None
    c = abs(a["cents"])
    for text in (A.money(c), A.money(-c)):
        if text in prompt:
            return text
    for n in (c, -c):
        m = re.search(r'"(\w+)":' + str(n) + r"[,}]", prompt)
        if m:
            return m.group(1)
    return None


# --- facts ---------------------------------------------------------------

DAYNAMES = {"monday": "Mon", "tuesday": "Tue", "wednesday": "Wed", "thursday": "Thu", "friday": "Fri",
            "saturday": "Sat", "sunday": "Sun"}
MODS = r"(?:XR|CT|MR|MRI|US|X-ray|ultrasound)"
FACTS = [
    ("unread", re.compile(r"\b(?P<n>\d+)\s+unread\b(?!\s+" + MODS + r")(?!\s*(?:XR|CT|MR|US))")),
    ("unread", re.compile(r"\b(?:reading\s+)?backlog\s+of\s+(?P<n>\d+)(?:\s+unread)?\s+(?:studies|reports|exams)\b")),
    ("overdue", re.compile(r"(?<![-'\w])(?P<n>\d+)\s+overdue\b(?!\s+'?(?:soon|routine|urgent))")),
    ("overdue_priority", re.compile(r"\b(?P<n>\d+)\s+'?(?P<p>soon|routine|urgent)'?(?:\s+priority)?\s+(?:reports?|studies)"
                                    r"\s+(?:are\s+|is\s+)?overdue\b")),
    ("overdue_priority", re.compile(r"\b(?P<n>\d+)\s+overdue\s+'?(?P<p>soon|routine|urgent)'?\b")),
    ("lost_total", re.compile(r"\blost demand\s+(?:is\s+|remains\s+|has been\s+)?(?:stable\s+|steady\s+|unchanged\s+)?"
                              r"(?:at\s+|of\s+)?\(?(?P<n>\d+)\b(?!\s*(?:today|%))", re.I)),
    ("weekday", re.compile(r"\b[Tt]oday\s*\((?P<d>Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\b")),
    ("weekday", re.compile(r"\b[Tt]oday is (?:a\s+)?(?P<d>Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\b")),
    ("weekday", re.compile(r"\bcurrently closed\s*\((?P<d>Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\b")),
    ("closed", re.compile(r"\bthe practice is (?:currently\s+)?closed\b", re.I)),
]
SUBSET = re.compile(MODS + r"\s*(?:backlog|studies)?\s*(?:of|has|with|:)?\s*$")
PAST = re.compile(r"\b(?:yesterday|previous|previously|earlier|last (?:week|time|point)|was|were|had|from)\s+"
                  r"(?:[\w']+\s+)?$", re.I)


def facts(reason):
    """Specific factual claims in the reason that one observed field
    decides: (kind, value, span)."""
    out = []
    for kind, rx in FACTS:
        for m in rx.finditer(reason or ""):
            left = reason[max(0, m.start() - 30):m.start()]
            if kind in ("unread", "overdue", "overdue_priority", "lost_total"):
                if SUBSET.search(left) or PAST.search(left) or re.search(r"\b" + MODS + r"\b", left[-40:]):
                    continue
            v = m.groupdict()
            out.append({"kind": kind, "span": [m.start(), m.end()], "text": m.group(0),
                        "value": int(v["n"]) if v.get("n") else DAYNAMES[v["d"].lower()] if v.get("d") else True,
                        "priority": v.get("p")})
    uniq = []
    for x in sorted(out, key=lambda x: (x["span"][0], x["kind"])):
        # One claim per kind where two patterns read the same words.
        if any(u["kind"] == x["kind"] and u["span"][0] < x["span"][1] and x["span"][0] < u["span"][1] for u in uniq):
            continue
        uniq.append(x)
    # A total beside counts by priority in one sentence may be the remainder
    # ("2 overdue 'soon' reports and 15 overdue reports").
    for s, e in sentences(reason or ""):
        inside = [x for x in uniq if s <= x["span"][0] < e]
        if any(x["kind"] == "overdue_priority" for x in inside):
            for x in inside:
                if x["kind"] == "overdue":
                    x["ambiguous"] = "a total beside counts by priority in the same sentence"
    return uniq


def fact_shown(f, v, prompt):
    """Whether the prompt states the observed value in the words the model
    read: the active problems, an alert, or the header."""
    if not prompt:
        return False
    k = f["kind"]
    if k == "unread":
        return f"Reading backlog: {v} unread" in prompt or (v == 0 and "Reading backlog:" not in prompt and bool(
            re.search(r'"worklist":\{"policy":"[a-z_]+","unread":0,', prompt)))
    empty = v == 0 and "Reading backlog:" not in prompt and bool(
        re.search(r'"worklist":\{"policy":"[a-z_]+","unread":0,"reading":0,', prompt))
    if k == "overdue":
        return f" {v} overdue," in prompt or empty
    if k == "overdue_priority":
        return f"{v} {f['priority']} reports overdue" in prompt or f"{v} {f['priority']} report overdue" in prompt \
            or empty
    if k == "lost_total":
        return f"{v} since the start." in prompt
    if k == "weekday":
        return prompt.startswith("Decision point ") and f": {v} " in prompt.split("\n", 1)[0]
    if k == "closed":
        return "Practice phase: " in prompt.split("\n", 1)[0]
    return False


def observed_fact(f, obs):
    """(value, path, why-not) for a fact's observed field."""
    if not obs:
        return None, None, "no observation"
    w = obs.get("worklist") or {}
    items = [i for i in w.get("items") or [] if isinstance(i, dict)]
    complete = isinstance(w.get("unread"), int) and len(items) >= (w.get("unread") or 0) + (w.get("reading") or 0)
    if f["kind"] == "unread":
        return w.get("unread"), "worklist.unread", None
    if f["kind"] == "overdue":
        if not complete:
            return None, None, "the observation lists fewer worklist items than are unread or being read"
        return sum(1 for i in items if i.get("overdue")), "worklist.items[overdue]", None
    if f["kind"] == "overdue_priority":
        if not complete:
            return None, None, "the observation lists fewer worklist items than are unread or being read"
        return sum(1 for i in items if i.get("overdue") and i.get("priority") == f["priority"]), \
            f"worklist.items[overdue, priority {f['priority']}]", None
    if f["kind"] == "lost_total":
        mods = A.dig(obs, "demand", "modalities") or []
        return sum(A.dig(m, "sinceStart", "lost", default=0) or 0 for m in mods), \
            "demand.modalities[].sinceStart.lost (sum)", None
    if f["kind"] == "weekday":
        return obs.get("weekday"), "weekday", None
    if f["kind"] == "closed":
        p = obs.get("practice") or {}
        return (not p.get("open")) if isinstance(p.get("open"), bool) else None, "practice.open", None
    return None, None, "unknown fact"


# --- the corpus ----------------------------------------------------------

class Inputs:
    """Every input file read, with its SHA-256: evidence references are
    stable because each names a file and its content. A replay file is
    named relative to the replays directory, so that the names do not
    depend on where the replays were made."""

    def __init__(self, replays=None):
        self.files = {}
        self.replays = replays.rstrip("/") + "/" if replays else None

    def name(self, path):
        if self.replays and path.startswith(self.replays):
            return "replays/" + path[len(self.replays):]
        return path

    def read(self, path):
        with open(path, "rb") as f:
            b = f.read()
        self.files[self.name(path)] = sha256_bytes(b)
        return b


def load_json(inputs, path):
    return json.loads(inputs.read(path).decode("utf-8"))


def replay_recordings(cfg):
    """Each external policy's recordings in a replay configuration: policy
    id -> [paths], and the paths' basenames -> path."""
    out = {}
    for e in cfg.get("external") or []:
        cmd = e.get("command") or []
        paths = [cmd[i + 1] for i, x in enumerate(cmd) if x == "--replay" and i + 1 < len(cmd)]
        out[e.get("id")] = [os.path.normpath(p) for p in paths]
    return out


def model_of(rec_path, inputs):
    """The model a recording came from: its usage file, else the run's
    settings file beside it."""
    base = rec_path[:-len("decisions.jsonl")]
    for cand in (base + "usage.json", os.path.join(os.path.dirname(rec_path), "llm-usage.json"),
                 os.path.join(os.path.dirname(rec_path), "llm-settings.json")):
        if os.path.exists(cand):
            try:
                m = load_json(inputs, cand).get("model")
            except (ValueError, OSError):
                m = None
            if m:
                return m
    return None


def transcript_points(path, wanted):
    """The observation and catalog the agent received at each decision
    point, from the replay's protocol transcript: {point index: {obs,
    omitted, catalog}}, matched to the recorded decision points by their
    digests, in order."""
    out = {}
    i, cur, prev = 0, None, None
    with open(path, encoding="utf-8") as f:
        for line in f:
            d = json.loads(line)
            if "op" in d:
                prev = d
                continue
            if prev is None:
                continue
            op, prev = prev["op"], None
            if not d.get("ok"):
                continue
            res = d.get("result") or {}
            if op == "digests":
                cur = None
                if i < len(wanted) and res.get("digests") == wanted[i]:
                    cur = out.setdefault(i, {})
                    i += 1
            elif op == "observe" and cur is not None and "obs" not in cur:
                cur["obs"] = res.get("observation")
                cur["omitted"] = res.get("omitted") or {}
            elif op == "actions" and cur is not None and "catalog" not in cur:
                cur["catalog"] = res
    return out


def rebuild_prompts(recs, pts):
    """Each decision point's prompt and memory text, rebuilt with the
    agent's functions from the transcript and checked against the recorded
    SHA-256: {i: (prompt, memory, verified)}."""
    out = {}
    if not recs or recs[0].get("memoryFormat") != A.MEMORY_FORMAT or recs[0].get("promptVersion") != A.PROMPT_VERSION:
        return out
    pset = recs[0].get("promptSettings")
    if not isinstance(pset, dict):
        return out
    mem = A.Memory(pset["memoryDecisions"], pset["memoryMetricPoints"], pset["memoryMaxChars"])
    for i, r in enumerate(recs):
        p = pts.get(i)
        if not p or "obs" not in p or "catalog" not in p:
            break
        mem.observe(r["point"], p["obs"])
        mt = mem.render(A.pending(p["obs"], p["catalog"]))
        prompt, _ = A.build_prompt(r["point"], p["obs"], p["omitted"], p["catalog"], mt, r.get("promptSettings") or pset)
        ok = prompt is not None and A.sha256(prompt) == r.get("promptSha256") and A.sha256(mt) == r.get("memorySha256")
        out[i] = (prompt, mt, ok)
        mem.record(r)
    return out


# --- the analysis of one recording ---------------------------------------

def seen_text(rec, prompt):
    """What the model was shown before its final reply: the prompt and the
    retry messages (each earlier attempt's error)."""
    errs = rec.get("errors") or []
    return (prompt or "") + "\n" + "\n".join(str(e) for e in errs)


def short_action(a):
    """An action without its long fields (rosters, windows)."""
    return {k: v for k, v in a.items() if k not in ("roster",)}


class Analysis:
    def __init__(self):
        self.findings = []
        self.checks = {r: {"statements": 0, "consistent": 0, "supported": 0, "unresolved": 0,
                           "insufficient_evidence": 0} for r in RULES}
        self.traces = []
        self.repeat_summary = []
        self.by = {}  # (group, model, rule) -> counts, the denominators of each count of findings
        self.cur = None

    def _count(self, rule, what):
        self.checks[rule][what] += 1
        if self.cur:
            k = "|".join(self.cur + (rule,))
            c = self.by.setdefault(k, {"statements": 0, "consistent": 0, "supported": 0, "unresolved": 0,
                                       "insufficient_evidence": 0})
            c[what] += 1

    def add(self, ctx, rule, status, basis, rec=None, span=None, reason=None, actions=None, exec_=None,
            observation=None, extra=None, sub="", key=None):
        assert status in STATUSES
        # The identifier: the recording's content, the decision point, the
        # rule and its version, the reason span and what distinguishes the
        # finding there (a statement's target, a refused proposal's key).
        key = [ctx["recording"]["sha256"], rec["point"] if rec else "-", rule, RULES[rule], sub,
               f"{span[0]}:{span[1]}" if span else "-", json.dumps(key, sort_keys=True),
               json.dumps(actions or [], sort_keys=True)]
        f = {
            "id": stable_id(*key),
            "category": rule,
            "rule": {"id": rule, "version": RULES[rule], "subrule": sub or None},
            "status": status,
            "basis": basis,
            "recording": ctx["recording"],
            "run": ctx["run"],
            "decision": None if rec is None else {
                "point": rec.get("point"), "time": rec.get("time"), "day": rec.get("day"),
                "outcome": rec.get("outcome"), "attempts": rec.get("attempts"),
                "line": ctx["lines"].get(rec.get("point"))},
            "sources": ctx["sources"](rec),
            "reasonSpan": None if span is None or reason is None else {
                "start": span[0], "end": span[1], "text": reason[span[0]:span[1]]},
            "actions": actions or [],
            "execution": exec_ or [],
            "observation": observation,
            "completeness": ctx["completeness"](rec),
        }
        if extra:
            f.update(extra)
        self.findings.append(f)
        self._count(rule, status)

    def consistent(self, rule):
        self._count(rule, "consistent")

    def statement(self, rule):
        self._count(rule, "statements")


def analyze_recording(an, ctx, recs, pts, prompts, applied_by_point, cov):
    """Every rule on one recording, and its decision traces."""
    an.cur = (ctx["recording"]["group"], ctx["run"]["model"] or "?")
    rebuilt = {i: v for i, v in prompts.items()}
    for i, rec in enumerate(recs):
        reason = rec.get("reason") if isinstance(rec.get("reason"), str) else None
        proposed = rec.get("proposed") if isinstance(rec.get("proposed"), list) else []
        p = pts.get(i) or {}
        obs, catalog = p.get("obs"), p.get("catalog")
        prompt, memory_text, verified = rebuilt.get(i, (None, None, False))
        applied = applied_by_point.get(rec.get("point"))
        ex = execution(rec, applied)
        staff = staff_index(obs)
        cov["points"] += 1
        if not reason:
            cov["reasonMissing"] += 1
        if verified:
            cov["promptVerified"] += 1
        elif i in rebuilt:
            cov["promptMismatch"] += 1
        if obs is not None:
            cov["observation"] += 1
        if rec.get("outcome") == "actions":
            cov["submittedPoints"] += 1
            if applied is not None:
                cov["applyResults"] += 1
        acts = [dict(short_action(a), index=k) for k, a in enumerate(proposed)]

        stmts, revised = extract(reason, staff) if reason else ([], False)
        positive = [s for s in stmts if s["kind"] == "commit"]
        for st0 in stmts:
            for st in per_target(st0):
                check_statement(an, ctx, rec, reason, st, proposed, ex, catalog, positive, obs)
        if reason:
            check_amounts(an, ctx, rec, reason, obs, catalog, prompt, verified, i in rebuilt)
            check_facts(an, ctx, rec, reason, obs, prompt, verified, i in rebuilt)
        an.traces.append(trace(ctx, rec, i, recs, pts, acts, ex, reason, verified, applied))
    check_repeats(an, ctx, recs, rebuilt)


def check_statement(an, ctx, rec, reason, st, proposed, ex, catalog, positive, obs):
    span = st["span"]
    sk = {"kind": st["kind"], "type": st["type"], "targets": st["targets"], "params": st["params"]}
    add = lambda *a, **k: an.add(*a, key=sk, **k)
    blockers = [f for f in st["flags"] if f in ("hedged", "deferred", "revised_in_text", "unresolved_target",
                                                  "qualified", "question", "says_cannot", "already",
                                                  "retracted_elsewhere")]
    exec_for = lambda idx: [dict(ex[k], action=short_action(proposed[k])) for k in idx]
    if st["kind"] in ("no_action", "negated"):
        # "Already" or "cannot" is the reason not to act: it does not weaken
        # a statement of no action.
        blockers = [b for b in blockers if b not in ("already", "says_cannot", "retracted_elsewhere")]
    if st["kind"] == "no_action":
        rule = "EXPLICIT_OPPOSITE_ACTION"
        an.statement(rule)
        if not proposed:
            an.consistent(rule)
            return
        why = []
        if blockers:
            why.append("the statement is " + ", ".join(blockers))
        if positive:
            why.append("the reason also states an action")
        acts = [dict(short_action(a), index=k) for k, a in enumerate(proposed)]
        if why:
            add(ctx, rule, "unresolved", "A statement of no action, but " + "; ".join(why) + ".", rec, span,
                   reason, acts, exec_for(range(len(proposed))), sub="no_action_stated")
        else:
            add(ctx, rule, "supported",
                   f"The reason states no action (\"{reason[span[0]:span[1]]}\") and states no action to take; "
                   f"the same reply proposes {len(proposed)} action(s).", rec, span, reason, acts,
                   exec_for(range(len(proposed))), sub="no_action_stated")
        return
    if st["kind"] == "negated":
        rule = "EXPLICIT_OPPOSITE_ACTION"
        an.statement(rule)
        hit = [k for k, a in enumerate(proposed) if matches(st, a)]
        if not hit:
            an.consistent(rule)
            return
        acts = [dict(short_action(proposed[k]), index=k) for k in hit]
        if blockers or not st["targets"] and st["type"] in ("extend_shift", "terminate", "set_roster"):
            add(ctx, rule, "unresolved", "A statement not to act, matched by a proposed action, but the statement "
                   "is " + (", ".join(blockers) or "without a named target") + ".", rec, span, reason, acts,
                   exec_for(hit), sub="negated_action")
        else:
            add(ctx, rule, "supported", f"The reason states it will not {st['type']} "
                   f"{', '.join(st['targets']) or ''}; the same reply proposes that action.".replace("  ", " "),
                   rec, span, reason, acts, exec_for(hit), sub="negated_action")
        return
    # A commitment.
    rule_a, rule_b, rule_e = "EXPLICIT_ACTION_OMITTED", "EXPLICIT_OPPOSITE_ACTION", "STATED_ACTION_NOT_EXECUTED"
    an.statement(rule_a)
    hit = [k for k, a in enumerate(proposed) if matches(st, a)]
    if blockers:
        if not hit:
            add(ctx, rule_a, "unresolved", "An action statement with no matching proposed action, but the "
                   "statement is " + ", ".join(blockers) + (f" ({st['why']})" if st.get("why") else "") + ".",
                   rec, span, reason, [], [], sub=st["type"] or "")
        else:
            an.consistent(rule_a)
        return
    # Direction: an allocation raised or lowered, a contract started or cancelled.
    if st["type"] == "set_reading_allocation" and st["params"].get("direction") and hit:
        an.statement(rule_b)
        for k in hit:
            a = proposed[k]
            cur = current_allocation(catalog, a.get("employee"))
            bp = a.get("siteAllocationBP")
            if a.get("op") != "set" or not isinstance(bp, int) or cur is None:
                add(ctx, rule_b, "insufficient_evidence" if cur is None else "unresolved",
                       "A directional allocation statement; the proposed change's direction cannot be decided "
                       "(no allocation in effect in the catalog, or not a set).", rec, span, reason,
                       [dict(short_action(a), index=k)], exec_for([k]),
                       observation={"path": "set_reading_allocation.targets[].inEffectBP", "value": cur},
                       sub="allocation_direction")
                continue
            d = "up" if bp > cur else "down" if bp < cur else "none"
            if d != "none" and d != st["params"]["direction"]:
                add(ctx, rule_b, "supported", f"The reason states the allocation goes {st['params']['direction']}; "
                       f"the proposed siteAllocationBP {bp} is {d} from {cur} in effect.", rec, span, reason,
                       [dict(short_action(a), index=k)], exec_for([k]),
                       observation={"path": f"set_reading_allocation.targets[{a.get('employee')}].inEffectBP",
                                    "value": cur}, sub="allocation_direction")
            else:
                an.consistent(rule_b)
    if not hit:
        opposite = []
        if st["type"] == "contract_teleradiology":
            other = "start" if st["params"].get("op") == "cancel" else "cancel"
            opposite = [k for k, a in enumerate(proposed) if a.get("type") == "contract_teleradiology"
                        and a.get("op") == other]
        if opposite:
            an.statement(rule_b)
            add(ctx, rule_b, "supported", f"The reason states a teleradiology contract {st['params'].get('op')}; "
                   f"the same reply proposes only a contract {other}.", rec, span, reason,
                   [dict(short_action(proposed[k]), index=k) for k in opposite], exec_for(opposite),
                   sub="contract_direction")
            return
        same_type = [k for k, a in enumerate(proposed) if a.get("type") == st["type"]]
        add(ctx, rule_a, "supported",
               f"The reason states a {st['type']}" + (f" for {', '.join(st['targets'])}" if st["targets"] else "")
               + (f" ({', '.join(f'{k} {v}' for k, v in sorted(st['params'].items()) if v)})" if any(
                   st["params"].values()) else "")
               + f"; the same reply proposes no such action ({len(proposed)} proposed"
               + (f", {len(same_type)} of that type with other targets or values" if same_type else "") + ").",
               rec, span, reason, [dict(short_action(proposed[k]), index=k) for k in same_type], exec_for(same_type),
               sub=st["type"])
        return
    an.consistent(rule_a)
    # Executed?
    an.statement(rule_e)
    done = [k for k in hit if ex[k]["state"] in EXECUTED]
    if done:
        an.consistent(rule_e)
        return
    states = sorted({ex[k]["state"] for k in hit})
    if any(s in ("unknown", "submitted_apply_unknown") for s in states):
        add(ctx, rule_e, "insufficient_evidence", "The stated action was proposed; whether it was applied is not "
               "in the available evidence (" + ", ".join(states) + ").", rec, span, reason,
               [dict(short_action(proposed[k]), index=k) for k in hit], exec_for(hit), sub=st["type"])
        return
    add(ctx, rule_e, "supported", "The stated action was proposed but not executed: " + ", ".join(states) + ".",
           rec, span, reason, [dict(short_action(proposed[k]), index=k) for k in hit], exec_for(hit),
           sub=st["type"])


def check_amounts(an, ctx, rec, reason, obs, catalog, prompt, verified, rebuildable):
    rule = "FINANCIAL_AMOUNT_MISMATCH"
    for a in amounts(reason):
        q, why = bind(a, reason)
        an.statement(rule)
        span = a["span"]
        if q is None:
            an.add(ctx, rule, "unresolved", f"A dollar amount ({a['text']}) that no quantity rule binds: {why}.",
                   rec, span, reason, sub="unbound")
            continue
        if not verified or obs is None:
            an.add(ctx, rule, "insufficient_evidence", f"The amount is stated as {q}, but the prompt the model saw "
                   "is not available verified (" + ("rebuilt but its SHA-256 differs" if rebuildable else
                                                     "not rebuildable from this recording") + ").",
                   rec, span, reason, sub=q)
            continue
        vals = observed_amounts(q, obs, catalog, reason[max(0, span[0] - 120):span[1] + 60])
        vals = [(p, v) for p, v in vals if isinstance(v, int)]
        if not vals:
            an.add(ctx, rule, "insufficient_evidence", f"The amount is stated as {q}; the observation has no such "
                   "value at this decision point.", rec, span, reason, sub=q)
            continue
        if any(agrees(a, v, signless=q == "profit_mtd") for _, v in vals):
            an.consistent(rule)
            continue
        if not any(shown(prompt, p, v) for p, v in vals):
            an.add(ctx, rule, "unresolved", f"The amount differs from {q}, but the observed value is not found in the "
                   "prompt text.", rec, span, reason, observation={"quantity": q, "values": [
                       {"path": p, "cents": v} for p, v in vals]}, sub=q)
            continue
        elsewhere = stated_in_prompt(a, prompt)
        if elsewhere:
            an.add(ctx, rule, "unresolved", f"The amount differs from {q}, but the prompt shows the stated amount "
                   f"elsewhere ({elsewhere}): it may name another quantity.", rec, span, reason,
                   observation={"quantity": q, "values": [{"path": p, "cents": v} for p, v in vals]}, sub=q)
            continue
        unit = next((unit_error(a, v) for _, v in vals if unit_error(a, v)), None)
        sub = q + (":" + unit if unit else "")
        an.add(ctx, rule, "supported",
               f"The reason states {q} as {a['text']} ({a['comparator']}, precision ${a['precisionCents'] // 100:,}); "
               "the prompt showed " + "; ".join(f"{p} = {A.money(v)}" for p, v in vals[:4])
               + (" (one of several alternatives)" if len(vals) > 1 else "")
               + (f"; the stated amount is the observed one with {unit.replace('_', ' ')}" if unit else "") + ".",
               rec, span, reason, observation={"quantity": q, "stated": {"cents": a["cents"],
                                                                          "precisionCents": a["precisionCents"],
                                                                          "comparator": a["comparator"]},
                                               "values": [{"path": p, "cents": v} for p, v in vals]},
               sub=sub)


def check_facts(an, ctx, rec, reason, obs, prompt, verified, rebuildable):
    rule = "FACTUAL_OBSERVATION_MISMATCH"
    for f in facts(reason):
        an.statement(rule)
        span = f["span"]
        if not verified or obs is None:
            an.add(ctx, rule, "insufficient_evidence", "A factual claim, but the prompt the model saw is not available "
                   "verified (" + ("rebuilt but its SHA-256 differs" if rebuildable else
                                   "not rebuildable from this recording") + ").", rec, span, reason, sub=f["kind"])
            continue
        if f.get("ambiguous"):
            an.add(ctx, rule, "unresolved", f"A factual claim ({f['kind']} {f['value']!r}) that may not mean what "
                   f"the rule reads: {f['ambiguous']}.", rec, span, reason, sub=f["kind"])
            continue
        v, path, why = observed_fact(f, obs)
        if v is None:
            an.add(ctx, rule, "insufficient_evidence", f"A factual claim ({f['kind']}); {why or 'no observed value'}.",
                   rec, span, reason, sub=f["kind"])
            continue
        if v == f["value"]:
            an.consistent(rule)
            continue
        if not fact_shown(f, v, prompt):
            an.add(ctx, rule, "unresolved", f"The claim ({f['kind']} {f['value']!r}) differs from {path} = {v!r}, "
                   "but the prompt text does not state that value where this rule looks for it.", rec, span, reason,
                   observation={"path": path, "value": v, "stated": f["value"]}, sub=f["kind"])
            continue
        an.add(ctx, rule, "supported", f"The reason states {f['kind'].replace('_', ' ')} {f['value']!r}"
               + (f" ({f['priority']})" if f.get("priority") else "") + f"; the observation shown has {path} = {v!r}.",
               rec, span, reason, observation={"path": path, "value": v, "stated": f["value"]}, sub=f["kind"])


DATE = re.compile(r"\b\d{4}-\d{2}-\d{2}\b")


def refusal_key(r):
    return (r.get("type"), r.get("code"), DATE.sub("<date>", str(r.get("message") or "")))


def refusals_of(rec):
    """The validate refusals of a decision point: the recorded list, or
    for a recording without one (llm-agent 1), parsed from its errors.
    None if neither is recorded."""
    if "refusals" in rec:
        return rec["refusals"] if isinstance(rec["refusals"], list) else []
    if not isinstance(rec.get("errors"), list):
        return None
    out = []
    for e in rec.get("errors") or []:
        m = re.match(r"attempt (\d+): refused: (.*)", str(e))
        if not m:
            continue
        for part in m.group(2).split("; action "):
            pm = re.match(r"(?:action )?(\d+): ([a-z_]+): (.*)", part)
            if pm:
                out.append({"attempt": int(m.group(1)), "action": int(pm.group(1)), "type": None,
                            "code": pm.group(2), "message": pm.group(3)})
    return out


SCHEMA_CODES = {"invalid_schema", "invalid_roster", "invalid_minutes", "invalid_action", "request_conflict"}


def check_repeats(an, ctx, recs, rebuilt):
    """REPEATED_INVALID_PROPOSAL: the same refused action (type, code and
    message, dates normalized) refused again, within a decision point's
    retries or at a later one."""
    rule = "REPEATED_INVALID_PROPOSAL"
    groups, missing = {}, 0
    for i, rec in enumerate(recs):
        rs = refusals_of(rec)
        if rs is None:
            missing += 1
            continue
        for r in rs:
            if r.get("code") == "batch_rejected":
                continue
            groups.setdefault(refusal_key(r), []).append((i, rec["point"], r.get("attempt"), r))
    summary = {"recording": ctx["recording"]["path"], "groups": 0, "repeated": 0, "maxRepeats": 0,
               "withinPoint": 0, "acrossPoints": 0, "acrossShownInMemory": 0, "acrossNotShown": 0,
               "schemaCodes": 0, "ruleCodes": 0, "refusals": sum(len(v) for v in groups.values())}
    if missing == len(recs) and recs:
        an.statement(rule)
        an.add(ctx, rule, "insufficient_evidence", "The recording keeps no validate refusals for any decision point.",
               None, sub="no_refusals_recorded")
        an.repeat_summary.append(summary)
        return
    for key in sorted(groups, key=lambda k: (groups[k][0][1], json.dumps(k))):
        occ = groups[key]
        summary["groups"] += 1
        an.statement(rule)
        if len(occ) < 2:
            an.consistent(rule)
            continue
        within = sum(1 for a, b in zip(occ, occ[1:]) if a[1] == b[1])
        across = []
        for a, b in zip(occ, occ[1:]):
            if a[1] != b[1]:
                mem = rebuilt.get(b[0], (None, None, False))
                text = mem[1] if mem[2] else None
                vis = None if text is None else (f"- {key[0]} {key[1]} (point {a[1]})" in text
                                                 or f"{key[0]} {key[1]}:" in text)
                across.append({"from": a[1], "to": b[1], "refusalShownInMemory": vis})
        later_ok = False
        last = occ[-1][0]
        for rec in recs[last + 1:]:
            if rec.get("outcome") == "actions" and any(a.get("type") == key[0]
                                                       for b in rec.get("batches") or [] for a in b):
                later_ok = True
                break
        summary["repeated"] += 1
        summary["maxRepeats"] = max(summary["maxRepeats"], len(occ))
        summary["withinPoint"] += within
        summary["acrossPoints"] += len(across)
        summary["acrossShownInMemory"] += sum(1 for x in across if x["refusalShownInMemory"])
        summary["acrossNotShown"] += sum(1 for x in across if x["refusalShownInMemory"] is False)
        summary["schemaCodes" if key[1] in SCHEMA_CODES else "ruleCodes"] += 1
        first = recs[occ[0][0]]
        an.add(ctx, rule, "supported",
               f"The same refused proposal ({key[0]} {key[1]}) was refused {len(occ)} times: {within} time(s) again "
               f"within a decision point's retries, {len(across)} time(s) at a later decision point.",
               first, None, None, [], [], extra={"repeat": {
                   "type": key[0], "code": key[1], "message": key[2], "count": len(occ),
                   "occurrences": [{"point": p, "attempt": at} for _, p, at, _ in occ],
                   "withinPoint": within, "acrossPoints": across,
                   "sameTypeLaterSubmitted": later_ok,
                   "codeClass": "schema" if key[1] in SCHEMA_CODES else "rule"}},
               sub=f"{key[1]}", key=list(key))
    an.repeat_summary.append(summary)


def metrics_of(obs):
    if not obs:
        return None
    f = obs.get("finance") or {}
    items = A.dig(obs, "worklist", "items") or []
    mods = A.dig(obs, "demand", "modalities") or []
    ss = A.dig(obs, "flow", "sinceStart") or {}
    return {"cashCents": f.get("cashCents"),
            "profitMTDCents": A.dig(f, "monthToDate", "operatingProfitCents"),
            "unread": A.dig(obs, "worklist", "unread"),
            "overdue": sum(1 for i in items if isinstance(i, dict) and i.get("overdue")),
            "lostSinceStart": sum(A.dig(m, "sinceStart", "lost", default=0) or 0 for m in mods),
            "examsSinceStart": ss.get("examsDone"), "overtimeMinSinceStart": ss.get("overtimeMin"),
            "creditDrawnCents": A.dig(f, "financing", "solvency", "creditDrawnCents")}


def trace(ctx, rec, i, recs, pts, acts, ex, reason, verified, applied):
    """One decision point's five stages and consequence links."""
    now = metrics_of((pts.get(i) or {}).get("obs"))
    nxt = metrics_of((pts.get(i + 1) or {}).get("obs")) if i + 1 < len(recs) else None
    delta = None
    if now and nxt:
        delta = {k: (nxt[k] - now[k]) if isinstance(nxt[k], int) and isinstance(now[k], int) else None for k in now}
    return {
        "recording": ctx["recording"]["path"], "group": ctx["recording"]["group"], "point": rec.get("point"),
        "time": rec.get("time"), "policy": ctx["run"]["policy"], "model": ctx["run"]["model"],
        "scenario": ctx["run"]["scenario"], "seed": ctx["run"]["seed"],
        "intent": {"reasonChars": len(reason or ""), "reasonSha256": sha256_bytes((reason or "").encode())[:16]
                   if reason else None},
        "proposed": [{"index": a["index"], "type": a.get("type"), "employee": a.get("employee")} for a in acts],
        "outcome": rec.get("outcome"),
        "validated": [x["validate"] for x in ex],
        "submitted": [x["submitted"] for x in ex],
        "executed": [x["state"] for x in ex],
        "direct": [x["apply"] for x in ex if x["apply"]],
        "promptVerified": verified,
        "subsequent": None if delta is None else {
            "nextPoint": recs[i + 1].get("point"), "observed": nxt, "changeSinceThisPoint": delta,
            "attribution": "none: the next decision point's observation, after every event of the day, "
                           "not a consequence of this decision alone"},
    }


# --- the corpus run ------------------------------------------------------

def analyze(corpus_path, replays, inputs=None):
    inputs = inputs or Inputs(replays)
    corpus = load_json(inputs, corpus_path)
    if corpus.get("schema") != CORPUS_SCHEMA:
        raise SystemExit(f"{corpus_path}: not a {CORPUS_SCHEMA} file")
    an = Analysis()
    coverage = []
    recordings_seen = []
    for g in corpus["groups"]:
        cov = {"group": g["id"], "role": g.get("role"), "recordings": 0, "points": 0, "reasonMissing": 0,
               "promptVerified": 0, "promptMismatch": 0, "observation": 0, "submittedPoints": 0, "applyResults": 0,
               "fields": {}, "agentVersions": [], "replayMissing": 0}
        fields_seen = {}
        agents = set()
        for rp in g["replays"]:
            cfg = load_json(inputs, rp["config"])
            recmap = replay_recordings(cfg)
            out = os.path.join(replays, rp["out"])
            runs_dir = os.path.join(out, "runs")
            by_base = {os.path.basename(p): p for ps in recmap.values() for p in ps}
            runs = {}
            if os.path.isdir(runs_dir):
                for rid in sorted(os.listdir(runs_dir)):
                    ev = os.path.join(runs_dir, rid, "policy-evidence", "decisions.jsonl")
                    if not os.path.exists(ev):
                        continue
                    with open(ev, encoding="utf-8") as f:
                        first = f.readline()
                    rp_base = json.loads(first).get("replayed") if first.strip() else None
                    if rp_base in by_base:
                        runs[by_base[rp_base]] = os.path.join(runs_dir, rid)
            for pid in sorted(recmap):
                for path in recmap[pid]:
                    recordings_seen.append(path)
                    raw = inputs.read(path)
                    lines = raw.decode("utf-8").splitlines()
                    recs = [json.loads(l) for l in lines if l.strip()]
                    for r in recs:
                        for k in r:
                            fields_seen[k] = fields_seen.get(k, 0) + 1
                        agents.add(f"{r.get('agentVersion') or '1'} / {r.get('promptVersion') or 'llm-agent 1 prompt'}"
                                   f" / {r.get('memoryFormat') or 'no memory'}")
                    run_dir = runs.get(path)
                    run = {}
                    pts, applied_by_point, prompts = {}, {}, {}
                    if run_dir:
                        run = load_json(inputs, os.path.join(run_dir, "run.json"))
                        tp = os.path.join(run_dir, "transcript.jsonl")
                        inputs.read(tp)
                        pts = transcript_points(tp, [r.get("digests") for r in recs])
                        dj = load_json(inputs, os.path.join(run_dir, "decisions.json"))
                        applied_by_point = {d.get("point"): d.get("batches") or [] for d in dj}
                        prompts = rebuild_prompts(recs, pts)
                    else:
                        cov["replayMissing"] += 1
                    cov["recordings"] += 1
                    rec_id = {"path": path, "sha256": sha256_bytes(raw), "group": g["id"],
                              "replayConfig": rp["config"]}
                    run_id = {"policy": pid, "model": model_of(path, inputs),
                              "scenario": (run.get("scenario") or {}).get("id"), "seed": run.get("seed"),
                              "days": run.get("days"), "engine": run.get("engine"),
                              "replayRun": os.path.join(rp["out"], "runs", os.path.basename(run_dir)) if run_dir
                              else None}
                    line_of = {r.get("point"): n + 1 for n, r in enumerate(recs)}

                    def sources(rec, path=path, run_dir=run_dir, rp=rp, prompts=prompts, recs=recs, line_of=line_of):
                        out = [{"kind": "recording", "path": path,
                                "line": line_of.get(rec.get("point")) if rec else None}]
                        if run_dir:
                            rel = os.path.join(rp["out"], "runs", os.path.basename(run_dir))
                            out.append({"kind": "replay", "path": rel + "/transcript.jsonl",
                                        "point": rec.get("point") if rec else None})
                            out.append({"kind": "apply", "path": rel + "/decisions.json",
                                        "point": rec.get("point") if rec else None})
                        if rec is not None:
                            i = recs.index(rec)
                            if i in prompts:
                                out.append({"kind": "prompt", "sha256": rec.get("promptSha256"),
                                            "verified": prompts[i][2]})
                        return out

                    def completeness(rec, prompts=prompts, recs=recs, run_dir=run_dir, pts=pts):
                        if rec is None:
                            return {"reason": None, "proposed": None, "validation": None, "refusals": None,
                                    "prompt": None, "observation": None, "apply": None}
                        i = recs.index(rec)
                        pv = prompts.get(i)
                        return {"reason": isinstance(rec.get("reason"), str) and bool(rec.get("reason")),
                                "proposed": isinstance(rec.get("proposed"), list),
                                "validation": rec.get("validation") is not None or not rec.get("proposed"),
                                "refusals": "refusals" in rec,
                                "prompt": "verified" if pv and pv[2] else "mismatch" if pv else "unavailable",
                                "observation": (pts.get(i) or {}).get("obs") is not None,
                                "apply": None if rec.get("outcome") != "actions" else bool(run_dir)}

                    ctx = {"recording": rec_id, "run": run_id, "lines": line_of, "sources": sources,
                           "completeness": completeness}
                    analyze_recording(an, ctx, recs, pts, prompts, applied_by_point, cov)
        cov["fields"] = {k: fields_seen[k] for k in sorted(fields_seen)}
        cov["agentVersions"] = sorted(agents)
        coverage.append(cov)
    an.findings.sort(key=lambda f: (f["recording"]["group"], f["recording"]["path"],
                                    (f["decision"] or {}).get("point") if f["decision"] else -1,
                                    f["category"], (f["reasonSpan"] or {}).get("start", -1), f["id"]))
    ids = [f["id"] for f in an.findings]
    if len(ids) != len(set(ids)):
        raise SystemExit("finding identifiers are not unique")
    by_status = {s: sum(1 for f in an.findings if f["status"] == s) for s in STATUSES}
    by_rule = {r: {s: sum(1 for f in an.findings if f["category"] == r and f["status"] == s) for s in STATUSES}
               for r in RULES}
    doc = {
        "schema": SCHEMA, "analyzer": ANALYZER, "rules": [{"id": r, "version": v} for r, v in RULES.items()],
        "corpus": {"path": corpus_path, "id": corpus.get("id")},
        "agent": {"version": A.VERSION, "promptVersion": A.PROMPT_VERSION, "memoryFormat": A.MEMORY_FORMAT,
                  "sha256": sha256_file(A.__file__)},
        "inputs": [{"path": p, "sha256": h} for p, h in sorted(inputs.files.items())],
        "coverage": coverage,
        "checks": an.checks,
        "checksByModel": {k: an.by[k] for k in sorted(an.by)},
        "byStatus": by_status,
        "byRule": by_rule,
        "repeats": an.repeat_summary,
        "modelPoints": model_points(an.traces),
        "findings": an.findings,
    }
    return doc, an.traces, recordings_seen


def sha256_file(path):
    with open(path, "rb") as f:
        return sha256_bytes(f.read())


def tables(doc):
    """The analysis report's tables, in Markdown."""
    L = ["# Decision evidence tables", "",
         f"Generated by `{doc['analyzer']}` ({doc['schema']}) from corpus `{doc['corpus']['id']}`. "
         "Counts of statements checked, not a measure of model quality.", "",
         "## Coverage", "",
         "| Group | Role | Recordings | Decision points | Reason missing | Prompt rebuilt and verified | "
         "Observation | Points with submitted actions | Apply results |",
         "|---|---|---|---|---|---|---|---|---|"]
    for c in doc["coverage"]:
        L.append(f"| {c['group']} | {c['role']} | {c['recordings']} | {c['points']} | {c['reasonMissing']} | "
                 f"{c['promptVerified']} | {c['observation']} | {c['submittedPoints']} | {c['applyResults']} |")
    L += ["", "## Checks by rule", "",
          "| Rule | Statements checked | Consistent | Supported | Unresolved | Insufficient evidence |",
          "|---|---|---|---|---|---|"]
    for r, c in doc["checks"].items():
        L.append(f"| {r} | {c['statements']} | {c['consistent']} | {c['supported']} | {c['unresolved']} | "
                 f"{c['insufficient_evidence']} |")
    L += ["", "## Findings by group, rule and status", "",
          "| Group | Rule | Supported | Unresolved | Insufficient evidence |", "|---|---|---|---|---|"]
    groups = [c["group"] for c in doc["coverage"]]
    for g in groups:
        for r in RULES:
            n = [sum(1 for f in doc["findings"] if f["recording"]["group"] == g and f["category"] == r
                     and f["status"] == s) for s in STATUSES]
            if any(n):
                L.append(f"| {g} | {r} | {n[0]} | {n[1]} | {n[2]} |")
    L += ["", "## Checks by group, model and rule", "",
          "Each count of findings beside the statements checked (and the decision points of the model's "
          "recordings in the group).", "",
          "| Group | Model | Decision points | Rule | Statements checked | Consistent | Supported | Unresolved | "
          "Insufficient evidence |", "|---|---|---|---|---|---|---|---|---|"]
    for k, c in doc["checksByModel"].items():
        g, m, r = k.split("|")
        if c["statements"]:
            L.append(f"| {g} | {m} | {doc['modelPoints'].get(g + '|' + m, '?')} | {r} | {c['statements']} | "
                     f"{c['consistent']} | {c['supported']} | {c['unresolved']} | {c['insufficient_evidence']} |")
    L += ["", "## Repeated invalid proposals", "",
          "| Recording | Refusals | Distinct refused proposals | Repeated | Most repeats | Within retries | "
          "At a later point | of which the refusal was in the memory shown | not shown |",
          "|---|---|---|---|---|---|---|---|---|"]
    for s in doc["repeats"]:
        if s["repeated"]:
            L.append(f"| `{s['recording']}` | {s['refusals']} | {s['groups']} | {s['repeated']} | {s['maxRepeats']} | "
                     f"{s['withinPoint']} | {s['acrossPoints']} | {s['acrossShownInMemory']} | {s['acrossNotShown']} |")
    L += ["", "## Supported findings", "",
          "| ID | Rule | Model | Recording | Point | Reason span | Basis |", "|---|---|---|---|---|---|---|"]
    for f in doc["findings"]:
        if f["status"] != "supported":
            continue
        span = (f["reasonSpan"] or {}).get("text") or ""
        span = span.replace("|", "\\|").replace("\n", " ")
        if len(span) > 160:
            span = span[:159] + "…"
        L.append(f"| `{f['id']}` | {f['category']}" + (f" ({f['rule']['subrule']})" if f["rule"]["subrule"] else "")
                 + f" | {f['run']['model']} | `{os.path.basename(f['recording']['path'])}` | "
                 f"{(f['decision'] or {}).get('point', '–')} | {span} | {f['basis'].replace('|', '/')} |")
    return "\n".join(L) + "\n"


def write_json(path, obj):
    with open(path, "w", encoding="utf-8") as f:
        json.dump(obj, f, indent=1, sort_keys=True, ensure_ascii=True)
        f.write("\n")


def main(argv=None):
    ap = argparse.ArgumentParser(description="Decision evidence analysis of recorded LLM agent decisions "
                                             "(read-only; no model is called).")
    ap.add_argument("--corpus", required=True, help="a dreamteam.decision-evidence-corpus/v1 file")
    ap.add_argument("--replays", required=True, help="the directory holding the corpus's replay outputs")
    ap.add_argument("--out", required=True, help="the output directory (created; files overwritten)")
    a = ap.parse_args(argv)
    doc, traces, _ = analyze(a.corpus, a.replays.rstrip("/"))
    os.makedirs(a.out, exist_ok=True)
    md = tables(doc)
    write_json(os.path.join(a.out, "evidence.json"), doc)
    with open(os.path.join(a.out, "traces.jsonl"), "w", encoding="utf-8") as f:
        for t in traces:
            f.write(json.dumps(t, sort_keys=True, separators=(",", ":")) + "\n")
    with open(os.path.join(a.out, "tables.md"), "w", encoding="utf-8") as f:
        f.write(md)
    c = doc["coverage"]
    print(f"decision evidence: {sum(x['recordings'] for x in c)} recordings, {sum(x['points'] for x in c)} decision "
          f"points, prompts verified {sum(x['promptVerified'] for x in c)}; findings: "
          + ", ".join(f"{k} {v}" for k, v in doc["byStatus"].items()))
    return 0


def model_points(traces):
    """Decision points per group and model: the denominator beside each
    count of findings."""
    out = {}
    for t in traces:
        k = t["group"] + "|" + (t["model"] or "?")
        out[k] = out.get(k, 0) + 1
    return {k: out[k] for k in sorted(out)}


if __name__ == "__main__":
    sys.exit(main())
