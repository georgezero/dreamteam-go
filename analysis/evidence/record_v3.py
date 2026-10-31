"""Readers of llm-agent 3's record (dreamteam.llm-decision-record/v3) for
the analyses, which were written for the llm-agent 2 format.

* v2_view: the record as llm-agent 2 would have written it for the same
  decision point, field for field, so that the frozen decision evidence
  (dreamteam.decision-evidence/v1) reads it unchanged.
* attempt_view: its attempts in the attempt-level form the funding probes
  write, for the refusal-response extension.
* attempts: its attempts in the common form of repeat rule version 2.

Nothing here changes a record, and no model is contacted.
"""

RECORD_SCHEMA = "dreamteam.llm-decision-record/v3"


def is_v3(rec):
    return isinstance(rec, dict) and rec.get("schema") == RECORD_SCHEMA


def proposal(at):
    """An attempt's proposed actions: its parsed actions, [] for a parsed
    reply without actions or a reply that could not be used as a decision,
    None for a details request (not a decision)."""
    if at.get("detailsRequested"):
        return None
    return at["actions"] if isinstance(at.get("actions"), list) else []


def v2_view(rec):
    """llm-agent 2's record of the same decision point: the summary fields
    llm-agent 3 keeps (outcome, reason, refusals, errors, batches), and
    those it no longer keeps at the top (attempts as a count, retries,
    detailsRequested, proposed, validation), set as llm-agent 2 set them:
    proposed is the last parsed reply's actions, validation the answers of
    the last attempt whose actions were validated."""
    v = {k: rec[k] for k in ("point", "time", "day", "digests", "mode", "agentVersion", "promptVersion", "memoryFormat",
                            "promptSettings", "memorySha256", "memoryChars", "promptSha256", "promptChars", "leftOut")
         if k in rec}
    atts = rec.get("attempts") or []
    parsed = [a for a in atts if (a.get("parse") or {}).get("ok")]
    validated = [a for a in atts if a.get("validation")]
    details = [a for a in atts if a.get("detailsRequested")]
    v.update({"attempts": len(atts), "retries": sum(1 for a in atts if a.get("turn") == "retry"),
              "detailsRequested": details[-1]["detailsRequested"] if details else None,
              "errors": rec.get("errors") or [], "refusals": rec.get("refusals") or [],
              "outcome": rec.get("outcome"), "reason": rec.get("reason"),
              "proposed": parsed[-1]["actions"] if parsed else None,
              "validation": [b["answer"] for b in validated[-1]["validation"]] if validated else None,
              "batches": rec.get("batches") or []})
    if "replayed" in rec:
        v["replayed"] = rec["replayed"]
    return v


def attempt_view(rec):
    """The decision point's attempts as the funding probes record them:
    {"attempts": [{"attempt", "actions", "validation": [{"results"}]}]},
    without details requests."""
    out = []
    for at in rec.get("attempts") or []:
        acts = proposal(at)
        if acts is None:
            continue
        out.append({"attempt": at["attempt"], "actions": acts,
                    "validation": [{"results": (b.get("answer") or {}).get("results") or []}
                                   for b in at.get("validation") or []]})
    return {"point": rec.get("point"), "attempts": out}
