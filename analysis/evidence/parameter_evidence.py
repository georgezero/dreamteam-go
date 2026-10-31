#!/usr/bin/env python3
"""Parameter evidence (dreamteam.parameter-evidence/v1): does a parameter
an agent's reason states explicitly agree with the parameter of the action
it submitted in the same reply?

    python3 -I -B analysis/evidence/parameter_evidence.py \
        --corpus docs/benchmark/v09a-dev/corpus.json --replays <dir> --out <dir>

Five rules, each narrow:
* AMOUNT_MISMATCH: an hourly rate stated for a hire (rateCents), or a site
  reading allocation stated as a percentage or basis points
  (siteAllocationBP);
* TARGET_MISMATCH: one named staff member, or one contract ID, against the
  submitted action's employee or contract;
* DATE_MISMATCH: a calendar date, a weekday, today or tomorrow against the
  action's date parameter;
* DURATION_MISMATCH: a shift extension's stated length against its
  minutes;
* DIRECTION_MISMATCH: an explicit increase or decrease of a reading
  allocation against the submitted value and the allocation in effect.

A reply's reason and actions are read together: llm-agent 3 records every
attempt's, llm-agent 2 only the final parsed reply's. A statement is read
only when it names an action of one type, states the value exactly, is not
hedged, negated or about the past, and binds to exactly one submitted
action. Anything less is abstention: `unresolved` with the reason, or
`insufficient_evidence` when an input is missing. No outcome is read, no
parameter is inferred, and no model is contacted.

Separate from dreamteam.decision-evidence/v1, which is frozen and
unchanged. The frozen analyzer's loaders (the corpus, the replay
transcript, the observation) are used read-only.
"""

import argparse
import datetime
import hashlib
import json
import os
import re
import sys
from decimal import Decimal, InvalidOperation

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(HERE, "..", "..", "agents", "llm"))
import decision_evidence as D  # noqa: E402
import llm_agent as A  # noqa: E402

SCHEMA = "dreamteam.parameter-evidence/v1"
ANALYZER = "parameter-evidence 1"
RULES = {"AMOUNT_MISMATCH": 1, "TARGET_MISMATCH": 1, "DATE_MISMATCH": 1, "DURATION_MISMATCH": 1,
         "DIRECTION_MISMATCH": 1}
STATUSES = ("consistent", "supported", "unresolved", "insufficient_evidence")
V3_SCHEMA = "dreamteam.llm-decision-record/v3"

# --- normalization rules (named in every check) ----------------------------
NORMALIZATION = {
    "N-MONEY-HOURLY": "a dollar amount with an hourly marker (/hr, /hour, per hour, an hour, hourly), thousands "
                      "separators removed, at most two decimals, as exact integer cents; compared exactly",
    "N-PERCENT": "a percentage as exact basis points (30% = 3000, 32.5% = 3250); compared exactly",
    "N-BP": "an explicit basis-point figure (3000 BP); compared exactly",
    "N-DATE-ISO": "an ISO calendar date (2026-11-19)",
    "N-DATE-MONTHDAY": "a month and day (Nov 19, 19 November), in the decision point's year, or the next year if "
                       "that would be more than 60 days before the decision point",
    "N-WEEKDAY": "a weekday name, compared with the weekday of the submitted date",
    "N-RELATIVE": "today or tomorrow, from the decision point's own date in the record",
    "N-DURATION": "minutes and hours (2 hours, 90 minutes, 1.5 h, an hour) as exact minutes; compared exactly",
    "N-STAFF": "a staff name or employee ID, matched as a whole word against the names and IDs the observation "
               "lists; a name that fits more than one employee is not resolved",
    "N-CONTRACT": "a teleradiology contract ID (T00001), compared exactly",
    "N-DIRECTION": "an increase or decrease verb governing an allocation, against the submitted value and the "
                   "allocation in effect in the observation's action catalog",
}

MONTHS = {m: i + 1 for i, m in enumerate(["jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct",
                                          "nov", "dec"])}
MONTH_RX = r"(?:Jan(?:uary)?|Feb(?:ruary)?|Mar(?:ch)?|Apr(?:il)?|May|June?|July?|Aug(?:ust)?|Sept?(?:ember)?|" \
           r"Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?)\.?"
WEEKDAYS = ["monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"]

# What a statement names: one action type per clause.
TYPE_RX = {
    "extend_shift": re.compile(r"\bextend(?:s|ed|ing)?\b|\bextension\b", re.I),
    "terminate": re.compile(r"\bterminat\w*|\blet(?:ting)? \w+ go\b|\bgive notice\b|\bgiving notice\b|\blay(?:ing)? off\b"
                            r"|\bdismiss\w*", re.I),
    "hire": re.compile(r"\bhir(?:e|es|ed|ing)\b|\brecruit\w*", re.I),
    "set_reading_allocation": re.compile(r"\ballocat(?:ion|ed)\b", re.I),
    "contract_teleradiology": re.compile(r"\b(?:teleradiology|tele-radiology|telerad)\b[^.;]{0,40}?\bcontracts?\b|"
                                        r"\bcontracts?\b[^.;]{0,40}?\b(?:teleradiology|tele-radiology|telerad)\b|"
                                        r"\bT\d{5}\b", re.I),
    "set_roster": re.compile(r"\broster\b|\brota\b", re.I),
}
# An action phrase: an action verb anchored by a stated intention ("I will",
# "I'll", "I am going to", "I'm …ing") or opening its clause ("Extending …",
# "Reduce …"), up to the first subordinate break, or, for a verb that opens
# the clause, up to its main verb ("… will allow"). Only a phrase is read:
# descriptions ("is currently 60%"), capabilities ("can extend up to") and
# names in other roles ("before Dr. Blake arrives") are not.
VERBS = (r"extend(?:ing)?|hir(?:e|ing)|recruit(?:ing)?|terminat(?:e|ing)|let(?:ting)?|giv(?:e|ing)|lay(?:ing)?|"
         r"dismiss(?:ing)?|set(?:ting)?|reduc(?:e|ing)|increas(?:e|ing)|rais(?:e|ing)|lower(?:ing)?|cut(?:ting)?|"
         r"decreas(?:e|ing)|chang(?:e|ing)|adjust(?:ing)?|restor(?:e|ing)|start(?:ing)?|begin(?:ning)?|sign(?:ing)?|"
         r"cancel(?:l?ing)?|end(?:ing)?|add(?:ing)?|mov(?:e|ing)|shift(?:ing)?|updat(?:e|ing)|correct(?:ing)?|"
         r"resubmit(?:ting)?|submit(?:ting)?|schedul(?:e|ing)|engag(?:e|ing)|bring(?:ing)?|boost(?:ing)?|"
         r"maximi[sz](?:e|ing)|minimi[sz](?:e|ing)|request(?:ing)?|activat(?:e|ing)|initiat(?:e|ing)|put(?:ting)?|"
         r"apply(?:ing)?|us(?:e|ing)|mak(?:e|ing)|establish(?:ing)?|add(?:ing)?|cover(?:ing)?|book(?:ing)?")
COMMIT = (r"\b(?:I|[Ww]e)\s+(?:will|shall)\s+|\b(?:I|[Ww]e)['’]ll\s+|\b(?:I\s+am|I['’]m|[Ww]e\s+are|[Ww]e['’]re)\s+going\s+"
          r"to\s+|\b(?:I|[Ww]e)\s+(?:plan|intend|have decided|decided|am about)\s+to\s+|\b(?:I\s+am|I['’]m|[Ww]e\s+are|"
          r"[Ww]e['’]re)\s+(?=\w+ing\b)")
ADVERBS = r"(?:(?:also|now|then|immediately|again|instead|still|first|additionally|therefore|thus|only|today|simply|" \
          r"just|proactively|consequently|accordingly|further|both)\s+){0,3}"
ANCHOR_COMMIT = re.compile(r"(?:" + COMMIT + r")" + ADVERBS + r"(?P<verb>\b(?:" + VERBS + r"))\b", re.I)
ANCHOR_START = re.compile(r"^\W*(?:(?:Also|Additionally|Then|So|Therefore|Thus|Instead|Finally|First|Next|Now|"
                          r"Meanwhile|Today|Second|Third),?\s+)?(?P<verb>\b(?:" + VERBS + r"))\b", re.I)
PHRASE_END = re.compile(r",?\s+(?:so that|so|because|since|as(?! of\b)(?! well\b)|which|while|before|after|once|whereas|"
                        r"but|given|due to|in order to|thereby|thus|leaving|helping|preventing|ensuring|allowing|"
                        r"providing|giving|who|where|whose|"
                        r"to (?:allow|help|ensure|address|clear|reduce|increase|cover|prevent|provide|maintain|support|"
                        r"handle|improve|avoid|respect|meet|match|free|save|maximi[sz]e|minimi[sz]e|boost|keep|catch|"
                        r"work|read|process|absorb|compensate|offset|stabili[sz]e|manage|lower|cut|bring|get|take|"
                        r"restore|protect|preserve|enable|capture|generate|recover|accommodate|staff|fill|deal|tackle|"
                        r"relieve|ease|mitigate|see|test|use|try|let|give|comply|align|account|stay|remain|avoid|"
                        r"absorb|replace|substitute|backfill|balance))\b"
                        r"|\s+[-—–]\s|\s*\(?\b(?:currently|which is|that is)\b", re.I)
SUBJECT_END = re.compile(r"\s+(?:will|would|should|can|could|may|might|must|is|are|was|were|allows?|helps?|provides?|"
                         r"ensures?|gives?|addresses|keeps|prevents|means|lets|enables|adds|brings|costs|saves|"
                         r"increases|reduces|lowers|raises|improves|creates|frees|maximi[sz]es|minimi[sz]es|"
                         r"requires|needs|aligns|matches|covers|supports)\b", re.I)


def action_phrases(reason, cs, ce):
    """The clause's action phrases: [(start, end)]."""
    text = reason[cs:ce]
    anchors = [(m.start("verb"), False) for m in ANCHOR_COMMIT.finditer(text)]
    m = ANCHOR_START.match(text)
    if m and not any(a == m.start("verb") for a, _ in anchors):
        anchors.append((m.start("verb"), True))
    anchors.sort()
    out = []
    for k, (a, opens) in enumerate(anchors):
        limit = anchors[k + 1][0] if k + 1 < len(anchors) else len(text)
        end = limit
        vm = re.match(r"\w+", text[a:])
        rest_from = a + (vm.end() if vm else 0)
        for rx in ((PHRASE_END, SUBJECT_END) if opens else (PHRASE_END,)):
            m2 = rx.search(text, rest_from, limit)
            if m2 and m2.start() < end:
                end = m2.start()
        out.append((cs + a, cs + end))
    return out


# A phrase that states no exact, current intention is not read.
EXCLUDE = [
    ("hedged", re.compile(r"\b(?:if|unless|could|would|might|may|maybe|perhaps|possibly|consider|considering|either|"
                          r"whether|option|alternatively|try|trying|hope|hoping)\b|\bor\b", re.I)),
    ("negated", re.compile(r"\b(?:not|never|cannot|no longer|instead of|rather than|invalid|unable|without|blocked|"
                           r"infeasible|unnecessary|premature|impossible|pointless|ineffective|wasteful|"
                           r"counterproductive|disallowed|prohibited|unavailable|redundant|too)\b|n['’]t\b", re.I)),
    ("past", re.compile(r"\b(?:was|were|had|has been|have been|already|previous|previously|yesterday|earlier|failed|"
                        r"attempted|last (?:time|point|attempt|week))\b", re.I)),
]
QUALIFIER = re.compile(r"(?:\b(?:at least|at most|up to|no more than|no less than|no later than|no earlier than|"
                       r"not before|not after|"
                       r"about|approximately|around|roughly|nearly|almost|over|under|more than|less than|within)"
                       r"\s+(?:the\s+)?(?:\w+\s+){0,1}|[~≈]\s*)$", re.I)
QUALIFIER_AFTER = re.compile(r"^\s*(?:or (?:more|less|later|earlier|longer|so)|\+)", re.I)

DURATION = re.compile(r"(?P<num>\d+(?:\.\d+)?|an?|one|two|three|four|five|six|seven|eight|half an?)\s*[- ]?"
                      r"(?P<unit>hours?|hrs?|h|minutes?|mins?)\b", re.I)
DURATION_BIND_BEFORE = re.compile(r"\b(?:by|for|an extra|extra|additional|another|of)\s+(?:the\s+)?(?:\w+\s+){0,3}$"
                                  r"|\(\s*$", re.I)
DURATION_BIND_AFTER = re.compile(r"^\s*(?:extension|longer|more|of overtime|overtime|extra)\b", re.I)
WORDNUM = {"a": 1, "an": 1, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8}

PERCENT = re.compile(r"(?P<num>\d+(?:\.\d+)?)\s*%")
BP = re.compile(r"(?P<num>\d{1,5})\s*BP\b")
VALUE_BIND = re.compile(r"\b(?:to|at|of)\s+(?:the\s+)?(?:\w+\s+){0,2}\(?$|=\s*$|\(\s*$", re.I)
CHANGE_BY = re.compile(r"\bby\s+(?:\w+\s+){0,1}$", re.I)
MONEY_HOURLY = re.compile(r"\$\s?(?P<num>\d[\d,]*(?:\.\d+)?)(?P<k>k)?\s*(?:/\s*(?:hr|hour|h)\b|per hour\b|an hour\b|"
                          r"hourly\b)", re.I)

DATE_ISO = re.compile(r"\b(?P<y>20\d\d)-(?P<m>\d\d)-(?P<d>\d\d)\b")
DATE_MD = re.compile(r"\b(?P<mon>" + MONTH_RX + r")\s+(?P<d>\d{1,2})(?:st|nd|rd|th)?\b(?:,?\s+(?P<y>20\d\d))?")
DATE_DM = re.compile(r"\b(?P<d>\d{1,2})(?:st|nd|rd|th)?\s+(?P<mon>" + MONTH_RX + r")(?:\s+(?P<y>20\d\d))?\b")
WEEKDAY = re.compile(r"\b(?P<w>Mon|Tues|Wednes|Thurs|Fri|Satur|Sun)day(?P<pos>['’]s)?\b")
RELATIVE = re.compile(r"\b(?P<w>today|tomorrow)(?P<pos>['’]s)?\b", re.I)
RELATIVE_OPEN = re.compile(r"\b(?:next (?:week|month|weekend)|this (?:week|weekend)|later this week|in \d+ (?:days|weeks)|"
                           r"in (?:a|one|two|three) (?:days?|weeks?)|the day after)\b", re.I)
DATE_BIND = re.compile(r"\b(?:no later than|no earlier than|not before|not after|by|on|from|starting|start(?:s|ing)?(?: on)?|beginning|begins|effective(?: from| on)?|as of|"
                       r"until|till|through|ending|end(?:s|ing)?(?: on)?|for|dated?|date of|last day(?: of| is| on)?|"
                       r"start date(?: of| is)?)\s+(?:the\s+)?(?:earliest\s+)?(?:effective\s+)?(?:date\s+)?\(?$"
                       r"|\b(?:date|day)\s*\(\s*$", re.I)
END_WORDS = re.compile(r"\b(?:until|till|through|ending|end(?:s|ing)?(?: on)?|end date)\s+(?:\w+\s+){0,2}$", re.I)
UP = re.compile(r"\b(?:increas|rais|boost|maximi[sz]|restor)\w*", re.I)
DOWN = re.compile(r"\b(?:decreas|reduc|lower|cut|minimi[sz])\w*", re.I)
ALLOC_OBJECT = re.compile(r"^\s+(?:\S+\s+){0,5}?\S*allocation\b|^\s+(?:it|this)\b", re.I)
CONTRACT_ID = re.compile(r"\bT\d{5}\b")
STAFF_ID = re.compile(r"\b[EH]\d{2,5}\b")

DATE_PARAM = {"extend_shift": "date", "terminate": "lastDay", "hire": "startDate",
              "set_reading_allocation": "effective", "set_roster": "effective"}
TARGET_PARAM = {"extend_shift": "employee", "terminate": "employee", "set_reading_allocation": "employee",
                "set_roster": "employee"}


def sha256_bytes(b):
    return hashlib.sha256(b).hexdigest()


def stable_id(*parts):
    return "pe1-" + sha256_bytes("|".join(str(p) for p in parts).encode("utf-8"))[:16]


# --- the observation ----------------------------------------------------------

def staff_index(obs):
    """lower-case name or ID -> set of employee IDs, from the observation's
    staff list: each ID, each full name, the name without "Dr.", and each
    word of the name. A key that fits more than one employee keeps them
    all, so that it is not resolved."""
    idx = {}
    for s in (obs or {}).get("staff") or []:
        if not isinstance(s, dict) or not s.get("id"):
            continue
        sid, name = s["id"], str(s.get("name") or "")
        keys = {sid.lower()}
        if name:
            n = name.lower()
            keys |= {n, n.replace("dr. ", "dr "), n.replace("dr. ", "")}
            keys |= {w for w in re.split(r"[\s.]+", n) if w and w != "dr"}
        for k in keys:
            idx.setdefault(k, set()).add(sid)
    return idx


NAME_TOKEN = re.compile(r"\b(?:Dr\.?\s+)?[A-Z][a-z]+\b|\b[EH]\d{2,5}\b")


def staff_mentions(text, base, staff):
    """The staff a clause names: [(start, end, ids)], ids a set (several if
    the name fits several employees). Words that are no staff name are
    ignored."""
    out = []
    for m in NAME_TOKEN.finditer(text):
        tok = m.group(0)
        key = re.sub(r"^dr\.?\s+", "", tok.lower())
        ids = staff.get(key)
        if ids:
            out.append((base + m.start(), base + m.end(), set(ids)))
    return out


def decision_date(rec):
    t = rec.get("time")
    try:
        return datetime.date.fromisoformat(str(t)[:10])
    except (TypeError, ValueError):
        return None


def current_allocation(catalog, emp):
    """The allocation in effect, and whether a change is already pending,
    from the observation's action catalog."""
    for t in (catalog or {}).get("types") or []:
        if t.get("type") == "set_reading_allocation" and isinstance(t.get("targets"), list):
            for e in t["targets"]:
                if e.get("employee") == emp and isinstance(e.get("inEffectBP"), int):
                    pend = any(isinstance(v, dict) and v.get("pending") for v in e.get("versions") or [])
                    return e["inEffectBP"], pend
    return None, False


# --- reading a clause -----------------------------------------------------------

def clauses(reason):
    """The reason's clauses: its sentences (split as the frozen analyzer
    splits them), as (start, end, after_semicolon)."""
    out = []
    for s, e in D.sentences(reason):
        prev = reason[:s].rstrip()
        out.append((s, e, prev.endswith(";")))
    return out


def types_in(text):
    found = []
    for t, rx in TYPE_RX.items():
        if rx.search(text):
            found.append(t)
    # A contract ID or "teleradiology" is the contract's; an extension of a
    # teleradiology reader's hours is not a shift extension.
    if "contract_teleradiology" in found and "extend_shift" in found and re.search(r"\bextend\w*\s+(?:the\s+)?"
                                                                                   r"(?:teleradiology\s+)?contract",
                                                                                   text, re.I):
        found.remove("extend_shift")
    return found


def excluded(text):
    for name, rx in EXCLUDE:
        if rx.search(text):
            return name
    return None


def qualified(reason, start, end, date=False):
    before = reason[max(0, start - 40):start]
    return bool(QUALIFIER.search(before) or QUALIFIER_AFTER.match(reason[end:end + 12])
                or date and re.search(r"\bby\s+(?:the\s+)?$", before, re.I))


def durations(reason, cs, ce):
    """Bound shift-extension lengths in a clause: [(start, end, minutes,
    text)]."""
    out = []
    text = reason[cs:ce]
    for m in DURATION.finditer(text):
        s, e = cs + m.start(), cs + m.end()
        before, after = reason[max(cs, s - 40):s], reason[e:min(ce, e + 20)]
        # a clock time (16:00) or a day count is no duration
        if re.search(r"\d:\d\d\s*$", before) or re.match(r"\s*:\d", after):
            continue
        if not (DURATION_BIND_BEFORE.search(before) or DURATION_BIND_AFTER.match(after)
                or re.match(r"\s*[- ]?(?:hour|minute)\s+(?:extension|shift extension)", reason[s:e + 20], re.I)):
            continue
        num, unit = m.group("num").lower(), m.group("unit").lower()
        try:
            n = Decimal(WORDNUM[num]) if num in WORDNUM else Decimal("0.5") if num.startswith("half") else Decimal(num)
        except InvalidOperation:
            continue
        mins = n * 60 if unit.startswith("h") else n
        if mins != mins.to_integral_value():
            continue
        out.append((s, e, int(mins), reason[s:e], qualified(reason, s, e)))
    return out


def percents(reason, cs, ce):
    """Bound allocation values in a clause: [(start, end, bp, text, rule,
    qualified)], and the spans of changes stated "by" an amount."""
    out, by = [], []
    for rx, rule in ((PERCENT, "N-PERCENT"), (BP, "N-BP")):
        for m in rx.finditer(reason[cs:ce]):
            s, e = cs + m.start(), cs + m.end()
            before = reason[max(cs, s - 30):s]
            if CHANGE_BY.search(before):
                by.append((s, e, reason[s:e]))
                continue
            if not VALUE_BIND.search(before):
                continue
            n = Decimal(m.group("num"))
            bp = n * 100 if rule == "N-PERCENT" else n
            if bp != bp.to_integral_value():
                continue
            out.append((s, e, int(bp), reason[s:e], rule, qualified(reason, s, e)))
    return out, by


def hourly_money(reason, cs, ce):
    out = []
    for m in MONEY_HOURLY.finditer(reason[cs:ce]):
        s, e = cs + m.start(), cs + m.end()
        try:
            v = Decimal(m.group("num").replace(",", ""))
        except InvalidOperation:
            continue
        if m.group("k"):
            v *= 1000
        cents = v * 100
        if cents != cents.to_integral_value():
            continue
        out.append((s, e, int(cents), reason[s:e], qualified(reason, s, e)))
    return out


def resolve_md(day, month, year, ddate):
    if year:
        try:
            return datetime.date(int(year), month, day)
        except ValueError:
            return None
    if ddate is None:
        return None
    try:
        d = datetime.date(ddate.year, month, day)
    except ValueError:
        return None
    if d < ddate - datetime.timedelta(days=60):
        try:
            d = datetime.date(ddate.year + 1, month, day)
        except ValueError:
            return None
    return d


def dates(reason, cs, ce, ddate):
    """Bound dates in a clause: [(start, end, kind, value, text, rule,
    end_word, qualified)]; kind "date" (value a date), "weekday" (value
    0-6) or "open" (a relative date the record does not establish)."""
    out, seen = [], []
    text = reason[cs:ce]

    def bound(s, e, possessive=False):
        before = reason[max(cs, s - 45):s]
        return possessive or bool(DATE_BIND.search(before))

    for rx, rule in ((DATE_ISO, "N-DATE-ISO"), (DATE_MD, "N-DATE-MONTHDAY"), (DATE_DM, "N-DATE-MONTHDAY")):
        for m in rx.finditer(text):
            s, e = cs + m.start(), cs + m.end()
            if any(a <= s < b or a < e <= b for a, b in seen):
                continue
            if rule == "N-DATE-ISO":
                try:
                    v = datetime.date(int(m.group("y")), int(m.group("m")), int(m.group("d")))
                except ValueError:
                    continue
            else:
                v = resolve_md(int(m.group("d")), MONTHS[m.group("mon").lower()[:3]], m.group("y"), ddate)
                if v is None:
                    continue
            seen.append((s, e))
            # "today (2027-01-07)" and "Nov 19 (Thursday)" define each other:
            # a date in parentheses after another counts with it.
            par = re.search(r"(?:today|tomorrow|" + MONTH_RX + r"\s+\d{1,2}|\d{4}-\d\d-\d\d|day)\s*\(\s*$",
                            reason[max(cs, s - 25):s], re.I)
            if bound(s, e) or par:
                out.append((s, e, "date", v, reason[s:e], rule, bool(END_WORDS.search(reason[max(cs, s - 30):s])),
                            qualified(reason, s, e, True)))
    for m in WEEKDAY.finditer(text):
        s, e = cs + m.start(), cs + m.end()
        if any(a <= s < b for a, b in seen) or re.match(r"\s*\(\s*$", reason[max(cs, s - 3):s]) and \
                re.search(r"\d\s*\(\s*$", reason[max(cs, s - 5):s]):
            continue  # "2026-11-16 (Monday)": the date says it
        if bound(s, e, bool(m.group("pos"))):
            w = WEEKDAYS.index((m.group("w") + "day").lower())
            out.append((s, e, "weekday", w, reason[s:e], "N-WEEKDAY",
                        bool(END_WORDS.search(reason[max(cs, s - 30):s])), qualified(reason, s, e, True)))
    for m in RELATIVE.finditer(text):
        s, e = cs + m.start(), cs + m.end()
        after = reason[e:e + 20]
        if re.match(r"\s*\(\s*\d{4}-\d\d-\d\d\s*\)", after):
            continue  # "today (2027-01-07)": the date is read instead
        if ddate is None:
            out.append((s, e, "date", None, reason[s:e], "N-RELATIVE", False, False))
            continue
        v = ddate + datetime.timedelta(days=1 if m.group("w").lower() == "tomorrow" else 0)
        out.append((s, e, "date", v, reason[s:e], "N-RELATIVE", False, qualified(reason, s, e, True)))
    for m in RELATIVE_OPEN.finditer(text):
        s, e = cs + m.start(), cs + m.end()
        if bound(s, e):
            out.append((s, e, "open", None, reason[s:e], "N-RELATIVE", False, False))
    out.sort(key=lambda x: x[0])
    return out


def direction(reason, cs, ce):
    """The direction an increase or decrease verb gives an allocation in a
    clause: ("up"|"down"|"both", start, end) or None."""
    found = []
    for rx, d in ((UP, "up"), (DOWN, "down")):
        for m in rx.finditer(reason[cs:ce]):
            s, e = cs + m.start(), cs + m.end()
            if ALLOC_OBJECT.match(reason[e:ce]) or re.match(r"\s+(?:\w+\s+){0,3}?(?:Dr\.?\s+)?[A-Z][a-z]+['’]s\s+"
                                                            r"(?:\w+\s+){0,2}allocation", reason[e:ce]):
                found.append((d, s, e))
    if not found:
        return None
    ds = {d for d, _, _ in found}
    s, e = min(x[1] for x in found), max(x[2] for x in found)
    return ("both" if len(ds) > 1 else ds.pop(), s, e)


# --- the replies of a recording ----------------------------------------------

def replies(recs):
    """Each reply whose reason and actions are recorded together:
    [(rec, attempt or None, reason, actions, per-action validation)].
    llm-agent 3: every parsed attempt that is not a details request.
    llm-agent 2 (and 1): the final parsed reply only."""
    out = []
    for rec in recs:
        if rec.get("schema") == V3_SCHEMA:
            for at in rec.get("attempts") or []:
                if not (at.get("parse") or {}).get("ok") or at.get("detailsRequested"):
                    continue
                acts = at.get("actions") if isinstance(at.get("actions"), list) else []
                val = [None] * len(acts)
                for b in at.get("validation") or []:
                    res = (b.get("answer") or {}).get("results") or []
                    for k, i in enumerate(b.get("indices") or []):
                        if k < len(res) and i < len(val):
                            val[i] = res[k]
                out.append((rec, at, at.get("reason"), acts, val))
        else:
            acts = rec.get("proposed") if isinstance(rec.get("proposed"), list) else None
            if acts is None:
                if isinstance(rec.get("reason"), str):
                    out.append((rec, None, rec.get("reason"), None, []))
                continue
            val = [None] * len(acts)
            answers = rec.get("validation")
            if acts and isinstance(answers, list):
                try:
                    batches = A.split([a if isinstance(a, dict) else {"type": None} for a in acts])
                except Exception:  # noqa: BLE001 - a malformed reply: validation left unknown
                    batches = []
                if len(batches) == len(answers):
                    for (_, idx), ans in zip(batches, answers):
                        res = (ans or {}).get("results") or []
                        for k, i in enumerate(idx):
                            if k < len(res):
                                val[i] = res[k]
            out.append((rec, None, rec.get("reason"), acts, val))
    return out


def validated(v):
    if v is None:
        return None
    return {"ok": bool(v.get("ok")), "code": v.get("code")}


# --- the analysis -------------------------------------------------------------------

class Analysis:
    def __init__(self):
        self.checks = []
        self.cov = {}
        self.by_type = {}

    def count(self, key, field, n=1):
        c = self.cov.setdefault(key, {})
        c[field] = c.get(field, 0) + n


def check(an, ctx, rec, at, reason, rule, status, basis, span, clause_span, action_index, act, param, stated, submitted,
          norm, val, extra=None):
    assert status in STATUSES and rule in RULES
    atid = at.get("id") if at else None
    key = [ctx["recording"]["sha256"], rec.get("point"), atid or "final", rule, RULES[rule],
           f"{span[0]}:{span[1]}", action_index if action_index is not None else "-", param]
    c = {
        "id": stable_id(*key),
        "rule": {"id": rule, "version": RULES[rule]},
        "status": status,
        "basis": basis,
        "recording": ctx["recording"],
        "run": ctx["run"],
        "decision": {"point": rec.get("point"), "time": rec.get("time"), "line": ctx["lines"].get(rec.get("point"))},
        "attempt": {"id": atid, "attempt": at.get("attempt") if at else None, "used": at.get("used") if at else None,
                    "level": "attempt" if at else "final parsed reply"},
        "reasonSpan": {"start": span[0], "end": span[1], "text": reason[span[0]:span[1]]},
        "statement": {"start": clause_span[0], "end": clause_span[1], "text": reason[clause_span[0]:clause_span[1]]},
        "action": None if action_index is None else {
            "ref": f"{rec.get('point')}.{at.get('attempt') if at else 'final'}#{action_index}", "index": action_index,
            "type": (act or {}).get("type"), "validation": validated(val)},
        "parameter": param,
        "stated": stated,
        "submitted": submitted,
        "normalization": norm,
        "completeness": ctx["completeness"](rec, at),
    }
    if extra:
        c.update(extra)
    an.checks.append(c)
    t = (act or {}).get("type") or ctx.get("clause_type") or "-"
    an.count(ctx["group"] + "|" + ctx["format"], rule + ":" + status)
    bt = an.by_type.setdefault(t, {})
    bt[rule + ":" + status] = bt.get(rule + ":" + status, 0) + 1


def candidates(acts, typ):
    return [(i, a) for i, a in enumerate(acts or []) if isinstance(a, dict) and a.get("type") == typ]


def analyze_reply(an, ctx, rec, at, reason, acts, vals, obs, catalog):
    key = ctx["group"] + "|" + ctx["format"]
    an.count(key, "replies")
    if not isinstance(reason, str) or not reason.strip():
        an.count(key, "reasonMissing")
        return
    if acts is None:
        an.count(key, "actionsMissing")
        return
    if D.REVISION.search(reason):
        an.count(key, "revisedReasons")
        return
    staff = staff_index(obs)
    ddate = decision_date(rec)
    prev_types = []
    phrases = []
    for cs0, ce0, after_semicolon in clauses(reason):
        clause_types = types_in(reason[cs0:ce0])
        for ps, pe in action_phrases(reason, cs0, ce0):
            types = types_in(reason[ps:pe])
            if not types and after_semicolon and len(prev_types) == 1 and re.search(r"\b(?:this|it)\b", reason[ps:pe],
                                                                                     re.I):
                types = list(prev_types)  # "…allocated 30%; increasing this to 50%"
            phrases.append((cs0, ce0, ps, pe, types))
        prev_types = clause_types
    # Every employee an action statement of each type names: a statement
    # naming someone else is then about another action, not a mistargeted one.
    named_by_type = {}
    for _, _, ps, pe, types in phrases:
        if len(types) == 1:
            for _, _, ids in staff_mentions(reason[ps:pe], ps, staff):
                if len(ids) == 1:
                    named_by_type.setdefault(types[0], []).append(((ps, pe), next(iter(ids))))
    for clause_start, clause_end, cs, ce, types in phrases:
        text = reason[cs:ce]
        if not types:
            continue
        an.count(key, "statements")
        why = excluded(reason[clause_start:clause_end])
        if why:
            an.count(key, "excluded:" + why)
            continue
        if len(types) > 1:
            an.count(key, "excluded:several action types")
            continue
        typ = types[0]
        ctx["clause_type"] = typ
        cands = candidates(acts, typ)
        if not cands:
            an.count(key, "noActionOfType")
            continue
        clause_span = (cs, ce)
        mentions = staff_mentions(text, cs, staff) if staff else []
        named = [m for m in mentions if len(m[2]) == 1]
        ambiguous_names = [m for m in mentions if len(m[2]) > 1]
        named_ids = []
        for _, _, ids in named:
            i = next(iter(ids))
            if i not in named_ids:
                named_ids.append(i)

        # The actions a value binds to: the named target's, each named
        # target's ("Riley and Casey by 2 hours each"), each of "both" or
        # "all three" when that is every action of the type, or the only one.
        def bind():
            if typ in TARGET_PARAM:
                if ambiguous_names:
                    return None, "a name fits more than one employee"
                if named_ids:
                    hits = [[(i, a) for i, a in cands if a.get(TARGET_PARAM[typ]) == t] for t in named_ids]
                    if all(len(h) == 1 for h in hits):
                        return [h[0] for h in hits], None
                    if len(named_ids) == 1:
                        if not hits[0] and len(cands) == 1:
                            return None, "the named employee is not the submitted action's"
                        return None, "several possible actions" if hits[0] else \
                            "no submitted action for the named employee"
                    return None, "several staff named, not each with one submitted action"
            m = re.search(r"\b(?:both|all (two|three|four|five|six))\b", text, re.I)
            if m and len(cands) == (2 if m.group(0).lower() == "both" else
                                    ["two", "three", "four", "five", "six"].index(m.group(1).lower()) + 2):
                return list(cands), None
            if len(cands) == 1:
                return [cands[0]], None
            return None, "several possible actions"

        bound_list, unbound_why = bind()
        obs_missing = obs is None

        # TARGET_MISMATCH
        if typ in TARGET_PARAM and staff:
            p = TARGET_PARAM[typ]
            if ambiguous_names:
                s, e, ids = ambiguous_names[0]
                check(an, ctx, rec, at, reason, "TARGET_MISMATCH", "unresolved",
                      f"The name fits {len(ids)} employees ({', '.join(sorted(ids))}).", (s, e), clause_span, None,
                      None, p, {"text": reason[s:e], "value": sorted(ids)}, None, "N-STAFF", None)
            elif len(named_ids) == 1:
                s, e, _ = named[0]
                tgt = named_ids[0]
                hit = [(i, a) for i, a in cands if a.get(p) == tgt]
                if hit:
                    i, a = hit[0]
                    check(an, ctx, rec, at, reason, "TARGET_MISMATCH", "consistent",
                          "The named employee is the submitted action's.", (s, e), clause_span, i, a, p,
                          {"text": reason[s:e], "value": tgt}, {"value": a.get(p)}, "N-STAFF", vals[i])
                elif len(cands) == 1 and any(t == cands[0][1].get(p) and sp != (cs, ce)
                                             for sp, t in named_by_type.get(typ, [])):
                    i, a = cands[0]
                    check(an, ctx, rec, at, reason, "TARGET_MISMATCH", "unresolved",
                          f"The submitted {typ} is for {a.get(p)}, whom another statement names; this one names "
                          f"{tgt}, an action not submitted (an omission, which this rule does not judge).", (s, e),
                          clause_span, i, a, p, {"text": reason[s:e], "value": tgt}, {"value": a.get(p)}, "N-STAFF",
                          vals[i])
                elif len(cands) == 1:
                    i, a = cands[0]
                    check(an, ctx, rec, at, reason, "TARGET_MISMATCH", "supported",
                          f"The statement names {tgt}; the only {typ} submitted in this reply is for {a.get(p)}.",
                          (s, e), clause_span, i, a, p, {"text": reason[s:e], "value": tgt}, {"value": a.get(p)},
                          "N-STAFF", vals[i])
                else:
                    check(an, ctx, rec, at, reason, "TARGET_MISMATCH", "unresolved",
                          f"No {typ} for {tgt} among {len(cands)} submitted; which one the statement means is "
                          "not determined.", (s, e), clause_span, None, None, p,
                          {"text": reason[s:e], "value": tgt}, None, "N-STAFF", None)
            elif len(named_ids) > 1:
                for s, e, ids in named:
                    tgt = next(iter(ids))
                    hit = [(i, a) for i, a in cands if a.get(p) == tgt]
                    if hit:
                        i, a = hit[0]
                        check(an, ctx, rec, at, reason, "TARGET_MISMATCH", "consistent",
                              "The named employee is one of the submitted actions'.", (s, e), clause_span, i, a, p,
                              {"text": reason[s:e], "value": tgt}, {"value": a.get(p)}, "N-STAFF", vals[i])
                    else:
                        check(an, ctx, rec, at, reason, "TARGET_MISMATCH", "unresolved",
                              "Several staff are named in the statement; this one has no submitted action, and the "
                              "statement may name them for another reason.", (s, e), clause_span, None, None, p,
                              {"text": reason[s:e], "value": tgt}, None, "N-STAFF", None)
        elif typ in TARGET_PARAM and obs_missing and STAFF_ID.search(text):
            m = STAFF_ID.search(text)
            check(an, ctx, rec, at, reason, "TARGET_MISMATCH", "insufficient_evidence",
                  "The observation (the staff list) is not available.", (cs + m.start(), cs + m.end()), clause_span,
                  None, None, TARGET_PARAM[typ], {"text": m.group(0), "value": None}, None, "N-STAFF", None)
        if typ == "contract_teleradiology":
            ids = sorted(set(CONTRACT_ID.findall(text)))
            withc = [(i, a) for i, a in cands if a.get("contract")]
            if len(ids) == 1 and withc:
                m = CONTRACT_ID.search(text)
                s, e = cs + m.start(), cs + m.end()
                hit = [(i, a) for i, a in withc if a.get("contract") == ids[0]]
                if hit:
                    i, a = hit[0]
                    check(an, ctx, rec, at, reason, "TARGET_MISMATCH", "consistent",
                          "The named contract is the submitted action's.", (s, e), clause_span, i, a, "contract",
                          {"text": ids[0], "value": ids[0]}, {"value": a.get("contract")}, "N-CONTRACT", vals[i])
                elif len(withc) == 1:
                    i, a = withc[0]
                    check(an, ctx, rec, at, reason, "TARGET_MISMATCH", "supported",
                          f"The statement names {ids[0]}; the only contract action naming a contract is for "
                          f"{a.get('contract')}.", (s, e), clause_span, i, a, "contract",
                          {"text": ids[0], "value": ids[0]}, {"value": a.get("contract")}, "N-CONTRACT", vals[i])
                else:
                    check(an, ctx, rec, at, reason, "TARGET_MISMATCH", "unresolved", "Several contract actions.",
                          (s, e), clause_span, None, None, "contract", {"text": ids[0], "value": ids[0]}, None,
                          "N-CONTRACT", None)

        def value_check(rule, param, items, compare, norm_of):
            """items: [(start, end, value, text, qualified)]; one check per
            distinct statement, or one unresolved if they disagree."""
            if not items:
                return
            vals_ = {it[2] for it in items}
            s, e = items[0][0], items[-1][1]
            stated = {"text": "; ".join(it[3] for it in items), "value": items[0][2] if len(vals_) == 1 else None}
            norm = norm_of(items)
            if len(vals_) > 1:
                check(an, ctx, rec, at, reason, rule, "unresolved", "The statement gives several different values.",
                      (s, e), clause_span, None, None, param, dict(stated, values=sorted(map(str, vals_))), None,
                      norm, None)
                return
            if any(it[4] for it in items):
                check(an, ctx, rec, at, reason, rule, "unresolved", "The value is qualified (at least, about, …), "
                      "not stated exactly.", (s, e), clause_span, None, None, param, stated, None, norm, None)
                return
            if bound_list is None:
                check(an, ctx, rec, at, reason, rule, "unresolved", f"Not bound to one action: {unbound_why}.",
                      (s, e), clause_span, None, None, param, stated, None, norm, None)
                return
            for i, a in bound_list:
                if a.get(param) is None:
                    check(an, ctx, rec, at, reason, rule, "unresolved",
                          f"The submitted action does not state {param}; its default is not inferred.", (s, e),
                          clause_span, i, a, param, stated, {"value": None}, norm, vals[i])
                    continue
                ok, sub, note = compare(items[0][2], a.get(param))
                if ok is None:
                    check(an, ctx, rec, at, reason, rule, "unresolved", note, (s, e), clause_span, i, a, param,
                          stated, sub, norm, vals[i])
                else:
                    check(an, ctx, rec, at, reason, rule, "consistent" if ok else "supported", note, (s, e),
                          clause_span, i, a, param, stated, sub, norm, vals[i])

        # DURATION_MISMATCH
        if typ == "extend_shift":
            ds = [(s, e, m, t, q) for s, e, m, t, q in durations(reason, cs, ce)]

            def cmp_minutes(st, sub):
                if not isinstance(sub, int):
                    return None, {"value": sub}, "The submitted minutes are not a number."
                return (st == sub, {"value": sub, "normalized": sub},
                        f"Stated {st} minutes; submitted {sub} minutes.")
            value_check("DURATION_MISMATCH", "minutes", ds, cmp_minutes, lambda _: "N-DURATION")

        # AMOUNT_MISMATCH
        if typ == "set_reading_allocation":
            ps, by = percents(reason, cs, ce)
            for s, e, t in by:
                check(an, ctx, rec, at, reason, "AMOUNT_MISMATCH", "unresolved",
                      "A change stated by an amount, not the value set; percentage points and relative change are "
                      "not told apart.", (s, e), clause_span, None, None, "siteAllocationBP",
                      {"text": t, "value": None}, None, "N-PERCENT", None)

            def cmp_bp(st, sub):
                if not isinstance(sub, int):
                    return None, {"value": sub}, "The submitted allocation is not a number."
                return st == sub, {"value": sub, "normalized": sub}, f"Stated {st} BP; submitted {sub} BP."
            value_check("AMOUNT_MISMATCH", "siteAllocationBP", [(s, e, bp, t, q) for s, e, bp, t, _, q in ps], cmp_bp,
                        lambda items: "+".join(sorted({r for _, _, _, _, r, _ in ps})))
        if typ == "hire":
            ms = hourly_money(reason, cs, ce)

            def cmp_cents(st, sub):
                if not isinstance(sub, int):
                    return None, {"value": sub}, "The submitted rate is not a number."
                note = f"Stated {st} cents an hour; submitted {sub}."
                if st != sub and st == sub * 100:
                    note += " The stated figure is the submitted cents read as dollars."
                return st == sub, {"value": sub, "normalized": sub}, note
            value_check("AMOUNT_MISMATCH", "rateCents", ms, cmp_cents, lambda _: "N-MONEY-HOURLY")

        def date_check(i, a, param, stated, cal, wks, rules_, s, e):
            raw = a.get(param)
            if raw is None:
                check(an, ctx, rec, at, reason, "DATE_MISMATCH", "unresolved",
                      f"The submitted action does not state {param}; its default is not inferred.",
                      (s, e), clause_span, i, a, param, stated, {"value": None}, rules_, vals[i])
                return
            try:
                sub = datetime.date.fromisoformat(str(raw))
            except ValueError:
                sub = None
            if sub is None:
                check(an, ctx, rec, at, reason, "DATE_MISMATCH", "unresolved",
                      "The submitted date is not a calendar date.", (s, e), clause_span, i, a, param,
                      stated, {"value": raw}, rules_, vals[i])
                return
            ok = (not cal or next(iter(cal)) == sub) and (not wks or next(iter(wks)) == sub.weekday())
            check(an, ctx, rec, at, reason, "DATE_MISMATCH", "consistent" if ok else "supported",
                  f"Stated {stated['value'] or stated['weekday']}; submitted {raw} "
                  f"({WEEKDAYS[sub.weekday()]}).", (s, e), clause_span, i, a, param, stated,
                  {"value": raw, "weekday": WEEKDAYS[sub.weekday()]}, rules_, vals[i])

        # DATE_MISMATCH
        dts = dates(reason, cs, ce, ddate)
        if dts:
            open_ = [d for d in dts if d[2] == "open"]
            if open_:
                s, e = open_[0][0], open_[0][1]
                check(an, ctx, rec, at, reason, "DATE_MISMATCH", "unresolved",
                      "A relative date the record does not establish.", (s, e), clause_span, None, None,
                      DATE_PARAM.get(typ, "effective"), {"text": open_[0][4], "value": None}, None, "N-RELATIVE",
                      None)
            exact = [d for d in dts if d[2] != "open"]
            if any(d[3] is None for d in exact):
                d = exact[0]
                check(an, ctx, rec, at, reason, "DATE_MISMATCH", "insufficient_evidence",
                      "The decision point's date is not recorded.", (d[0], d[1]), clause_span, None, None,
                      DATE_PARAM.get(typ, "effective"), {"text": d[4], "value": None}, None, d[5], None)
                exact = []
            if exact and not open_:
                if typ == "contract_teleradiology":
                    ends = {d[6] for d in exact}
                    if len(ends) > 1:
                        param = None
                    elif bound_list and all(a.get("op") == "cancel" for _, a in bound_list):
                        param = "endDate"
                    else:
                        param = "endDate" if ends.pop() else "effective"
                else:
                    param = DATE_PARAM.get(typ)
                if param is None:
                    d = exact[0]
                    check(an, ctx, rec, at, reason, "DATE_MISMATCH", "unresolved",
                          "Dates for both the start and the end; which is which is not bound.", (d[0], d[1]),
                          clause_span, None, None, "effective", {"text": d[4], "value": None}, None, d[5], None)
                else:
                    # Calendar dates and weekdays must agree with each other.
                    cal = {d[3] for d in exact if d[2] == "date"}
                    wks = {d[3] for d in exact if d[2] == "weekday"}
                    s, e = exact[0][0], exact[-1][1]
                    rules_ = "+".join(sorted({d[5] for d in exact}))
                    stated = {"text": "; ".join(d[4] for d in exact),
                              "value": (next(iter(cal)).isoformat() if len(cal) == 1 else None),
                              "weekday": (WEEKDAYS[next(iter(wks))] if len(wks) == 1 else None)}
                    agree = len(cal) <= 1 and len(wks) <= 1 and not (cal and wks and
                                                                     next(iter(cal)).weekday() != next(iter(wks)))
                    if not agree:
                        check(an, ctx, rec, at, reason, "DATE_MISMATCH", "unresolved",
                              "The statement gives several different dates.", (s, e), clause_span, None, None, param,
                              stated, None, rules_, None)
                    elif any(d[7] for d in exact):
                        check(an, ctx, rec, at, reason, "DATE_MISMATCH", "unresolved",
                              "The date is qualified (no later than, …), not stated exactly.", (s, e), clause_span,
                              None, None, param, stated, None, rules_, None)
                    elif bound_list is None:
                        check(an, ctx, rec, at, reason, "DATE_MISMATCH", "unresolved",
                              f"Not bound to one action: {unbound_why}.", (s, e), clause_span, None, None, param,
                              stated, None, rules_, None)
                    else:
                        for i, a in bound_list:
                            date_check(i, a, param, stated, cal, wks, rules_, s, e)

        # DIRECTION_MISMATCH
        if typ == "set_reading_allocation":
            dr = direction(reason, cs, ce)
            if dr:
                d, s, e = dr
                if d == "both":
                    check(an, ctx, rec, at, reason, "DIRECTION_MISMATCH", "unresolved",
                          "Both an increase and a decrease are stated.", (s, e), clause_span, None, None,
                          "siteAllocationBP", {"text": reason[s:e], "value": None}, None, "N-DIRECTION", None)
                elif bound_list is None:
                    check(an, ctx, rec, at, reason, "DIRECTION_MISMATCH", "unresolved",
                          f"Not bound to one action: {unbound_why}.", (s, e), clause_span, None, None,
                          "siteAllocationBP", {"text": reason[s:e], "value": d}, None, "N-DIRECTION", None)
                else:
                    for i, a in bound_list:
                        new = a.get("siteAllocationBP")
                        cur, pending = current_allocation(catalog, a.get("employee"))
                        stated = {"text": reason[s:e], "value": d}
                        if catalog is None or cur is None:
                            check(an, ctx, rec, at, reason, "DIRECTION_MISMATCH", "insufficient_evidence",
                                  "The allocation in effect is not in the observation.", (s, e), clause_span, i, a,
                                  "siteAllocationBP", stated, {"value": new}, "N-DIRECTION", vals[i])
                        elif not isinstance(new, int):
                            check(an, ctx, rec, at, reason, "DIRECTION_MISMATCH", "unresolved",
                                  "The submitted action sets no allocation value.", (s, e), clause_span, i, a,
                                  "siteAllocationBP", stated, {"value": new}, "N-DIRECTION", vals[i])
                        elif pending or new == cur:
                            check(an, ctx, rec, at, reason, "DIRECTION_MISMATCH", "unresolved",
                                  "A change is already pending, or the submitted value equals the allocation in effect.",
                                  (s, e), clause_span, i, a, "siteAllocationBP", stated,
                                  {"value": new, "inEffect": cur, "pending": pending}, "N-DIRECTION", vals[i])
                        else:
                            actual = "up" if new > cur else "down"
                            check(an, ctx, rec, at, reason, "DIRECTION_MISMATCH",
                                  "consistent" if actual == d else "supported",
                                  f"Stated {'an increase' if d == 'up' else 'a decrease'}; submitted {new} BP against "
                                  f"{cur} BP in effect.", (s, e), clause_span, i, a, "siteAllocationBP", stated,
                                  {"value": new, "inEffect": cur, "direction": actual}, "N-DIRECTION", vals[i])
        ctx.pop("clause_type", None)


def order(checks):
    """The output order: recording, decision point, attempt, position in
    the reason, rule, action, identifier."""
    return sorted(checks, key=lambda c: (c["recording"]["group"], c["recording"]["path"], c["decision"]["point"],
                                         c["attempt"]["attempt"] or 0, c["reasonSpan"]["start"], c["rule"]["id"],
                                         (c["action"] or {}).get("index", -1), c["id"]))


# --- the corpus -----------------------------------------------------------------------

def analyze(corpus_path, replays, inputs=None):
    inputs = inputs or D.Inputs(replays)
    corpus = D.load_json(inputs, corpus_path)
    if corpus.get("schema") != "dreamteam.parameter-evidence-corpus/v1":
        raise SystemExit(f"{corpus_path}: not a dreamteam.parameter-evidence-corpus/v1 file")
    an = Analysis()
    recordings = []
    for g in corpus["groups"]:
        for rp in g["replays"]:
            cfg = D.load_json(inputs, rp["config"])
            recmap = D.replay_recordings(cfg)
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
                    rb = json.loads(first).get("replayed") if first.strip() else None
                    if rb in by_base:
                        runs[by_base[rb]] = os.path.join(runs_dir, rid)
            for pid in sorted(recmap):
                for path in recmap[pid]:
                    raw = inputs.read(path)
                    recs = [json.loads(l) for l in raw.decode("utf-8").splitlines() if l.strip()]
                    fmt = ("llm-agent 3" if recs and recs[0].get("schema") == V3_SCHEMA else
                           f"llm-agent {recs[0].get('agentVersion') or 1}" if recs else "empty")
                    run_dir = runs.get(path)
                    pts, run = {}, {}
                    if run_dir:
                        # The replay's own files are derived (and checked by
                        # benchmark verify and reproduce); they carry the
                        # engine build, so they are read but not listed as
                        # inputs, and the output does not depend on the build.
                        with open(os.path.join(run_dir, "run.json"), encoding="utf-8") as f:
                            run = json.load(f)
                        pts = D.transcript_points(os.path.join(run_dir, "transcript.jsonl"),
                                                  [r.get("digests") for r in recs])
                    rec_id = {"path": path, "sha256": sha256_bytes(raw), "group": g["id"], "format": fmt}
                    run_id = {"policy": pid, "model": D.model_of(path, inputs),
                              "scenario": (run.get("scenario") or {}).get("id"), "seed": run.get("seed"),
                              "replayRun": os.path.join(rp["out"], "runs", os.path.basename(run_dir)) if run_dir
                              else None}
                    lines = {r.get("point"): n + 1 for n, r in enumerate(recs)}
                    idx_of = {id(r): n for n, r in enumerate(recs)}

                    def completeness(rec, at, pts=pts, idx_of=idx_of, fmt=fmt):
                        p = pts.get(idx_of.get(id(rec))) or {}
                        return {"attemptLevel": fmt == "llm-agent 3", "reason": True, "actions": True,
                                "validation": (at.get("validation") is not None) if at else
                                rec.get("validation") is not None,
                                "observation": p.get("obs") is not None, "catalog": p.get("catalog") is not None}
                    ctx = {"recording": rec_id, "run": run_id, "lines": lines, "completeness": completeness,
                           "group": g["id"], "format": fmt}
                    key = g["id"] + "|" + fmt
                    an.count(key, "recordings")
                    an.count(key, "points", len(recs))
                    if not run_dir:
                        an.count(key, "replayMissing")
                    recordings.append(path)
                    for rec, at, reason, acts, vals in replies(recs):
                        p = pts.get(idx_of[id(rec)]) or {}
                        if acts:
                            for a in acts:
                                if isinstance(a, dict):
                                    an.count(key, "submitted:" + str(a.get("type")))
                        analyze_reply(an, ctx, rec, at, reason, acts, vals, p.get("obs"), p.get("catalog"))
    an.checks = order(an.checks)
    ids = [c["id"] for c in an.checks]
    if len(ids) != len(set(ids)):
        raise SystemExit("check identifiers are not unique")
    by_rule = {r: {s: sum(1 for c in an.checks if c["rule"]["id"] == r and c["status"] == s) for s in STATUSES}
               for r in RULES}
    for r in by_rule:
        by_rule[r]["eligible"] = sum(by_rule[r].values())
    doc = {
        "schema": SCHEMA, "analyzer": ANALYZER,
        "rules": [{"id": r, "version": v} for r, v in RULES.items()],
        "normalization": NORMALIZATION,
        "corpus": {"path": corpus_path, "id": corpus.get("id")},
        "inputs": [{"path": p, "sha256": h} for p, h in sorted(inputs.files.items())],
        "coverage": {k: dict(sorted(v.items())) for k, v in sorted(an.cov.items())},
        "byActionType": {k: dict(sorted(v.items())) for k, v in sorted(an.by_type.items())},
        "byRule": by_rule,
        "checks": an.checks,
    }
    return doc


# --- tables ------------------------------------------------------------------------------

def tables(doc):
    L = [f"# Parameter evidence: `{doc['schema']}`", "",
         f"Analyzer `{doc['analyzer']}`, corpus `{doc['corpus']['id']}`. Every count describes recorded replies; "
         "none is a measure of model quality. `unresolved` and `insufficient_evidence` are abstentions.", "",
         "## By rule", "", "| Rule | Eligible | Consistent | Supported | Unresolved | Insufficient evidence |",
         "|---|---|---|---|---|---|"]
    for r, c in doc["byRule"].items():
        L.append(f"| {r} | {c['eligible']} | {c['consistent']} | {c['supported']} | {c['unresolved']} | "
                 f"{c['insufficient_evidence']} |")
    L += ["", "## Coverage by group and recording format", "",
          "| Group, format | Recordings | Points | Replies read | Reason missing | Revised | Statements | Excluded "
          "(hedged / negated / past / several types) | No action of the type | Checks |", "|---|---|---|---|---|---|---|---|---|---|"]
    for k, c in doc["coverage"].items():
        checks = sum(v for f, v in c.items() if ":" in f and f.split(":")[0] in RULES)
        ex = " / ".join(str(c.get("excluded:" + w, 0)) for w in ("hedged", "negated", "past", "several action types"))
        L.append(f"| {k.replace('|', ', ')} | {c.get('recordings', 0)} | {c.get('points', 0)} | {c.get('replies', 0)} | "
                 f"{c.get('reasonMissing', 0)} | {c.get('revisedReasons', 0)} | {c.get('statements', 0)} | {ex} | "
                 f"{c.get('noActionOfType', 0)} | {checks} |")
    L += ["", "## Checks by action type and rule", "", "| Action type | Rule | Consistent | Supported | Unresolved | "
          "Insufficient |", "|---|---|---|---|---|---|"]
    for t, c in doc["byActionType"].items():
        for r in RULES:
            row = [c.get(f"{r}:{s}", 0) for s in STATUSES]
            if any(row):
                L.append(f"| {t} | {r} | " + " | ".join(map(str, row)) + " |")
    sup = [c for c in doc["checks"] if c["status"] == "supported"]
    L += ["", f"## Supported mismatches ({len(sup)})", ""]
    for c in sup:
        L += [f"* `{c['id']}` {c['rule']['id']}: {c['recording']['path']} point {c['decision']['point']}, "
              f"{c['attempt']['level']} {c['attempt']['id'] or ''}, action {(c['action'] or {}).get('ref')}: "
              f"stated «{c['reasonSpan']['text']}», submitted {c['parameter']} = "
              f"`{json.dumps((c['submitted'] or {}).get('value'))}`. {c['basis']}"]
    unr = {}
    for c in doc["checks"]:
        if c["status"] in ("unresolved", "insufficient_evidence"):
            k = (c["rule"]["id"], c["status"], re.sub(r"\d+", "N", c["basis"]))
            unr[k] = unr.get(k, 0) + 1
    L += ["", "## Abstentions by reason", "", "| Rule | Status | Basis | Checks |", "|---|---|---|---|"]
    for (r, s, b), n in sorted(unr.items(), key=lambda x: (-x[1], x[0])):
        L.append(f"| {r} | {s} | {b} | {n} |")
    return "\n".join(L) + "\n"


def write_json(path, obj):
    with open(path, "w", encoding="utf-8") as f:
        json.dump(obj, f, indent=1, sort_keys=True, ensure_ascii=False)
        f.write("\n")


def main(argv=None):
    ap = argparse.ArgumentParser(description="Parameter evidence of recorded agent replies (no model is contacted).")
    ap.add_argument("--corpus", required=True)
    ap.add_argument("--replays", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    doc = analyze(a.corpus, os.path.normpath(a.replays))
    os.makedirs(a.out, exist_ok=True)
    write_json(os.path.join(a.out, "parameter-evidence.json"), doc)
    with open(os.path.join(a.out, "parameter-tables.md"), "w", encoding="utf-8") as f:
        f.write(tables(doc))
    br = doc["byRule"]
    print("parameter evidence: " + ", ".join(f"{r} {c['eligible']} ({c['supported']} supported)"
                                             for r, c in br.items()))
    return 0


if __name__ == "__main__":
    sys.exit(main())
