"""Versioned alternatives workflow, scripted transports only in this milestone."""
import hashlib
import json
import os
import time
from pathlib import Path
import llm_agent_alternatives as LEGACY

A5, A3 = LEGACY.A5, LEGACY.A3
ProtocolError=A3.ProtocolError
SettingsError=A3.SettingsError
sha256=A3.sha256
compact=A3.compact
HERE=Path(__file__).resolve().parent
TEXT=HERE/"text"
SYSTEM_VERSION="llm-agent-alternatives-system/v2"
WORKFLOW_VERSION="llm-agent-alternatives-workflow/v2"
RECORD_SCHEMA="dreamteam.alternatives-decision-record/v2"
PARSER_VERSION="dreamteam.phase-response-parser/v2"
SYSTEM_PROMPT=(TEXT/"alternatives-system-v2.txt").read_text()
INITIAL_TEMPLATE=(TEXT/"initial-v2.txt").read_text()
CORRECTION_TEMPLATE=(TEXT/"correction-v2.txt").read_text()
SCHEMAS=json.loads((TEXT/"response-schemas-v2.json").read_text())
CONTRACT_PINS={p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(TEXT.glob("*-v2.*"))}
CONTRACT_PINS["parserVersion"]=sha256(PARSER_VERSION)
WORKFLOW_SHA256=sha256(json.dumps(CONTRACT_PINS,sort_keys=True))
ARMS=("control","treatment")
TURN_RETRIES=1
INSPECT_ROUNDS=1
WORKFLOWS=A5.WORKFLOWS
def new_record(*args):
    rec=A5.new_record(*args)
    rec.update(schema="dreamteam.phase-proposal-record/v2",mode="supplied-transport",agentVersion="alternatives-v2",baseAgentVersion="5",promptVersion="llm-agent-prompt/v3",
               workflowVersion="llm-agent-phase-validation-first/v2",workflowSha256=WORKFLOW_SHA256,
               systemPromptVersion=SYSTEM_VERSION,systemPromptSha256=sha256(SYSTEM_PROMPT))
    return rec
execution_of=A5.execution_of
validate_all=A5.validate_all
refused_items=A5.refused_items
completeness=A5.completeness
stages=A5.stages
preview_due=A5.preview_due
request_preview=A5.request_preview

def feedback_text(items):
    return CORRECTION_TEMPLATE.format(results="\n".join(compact(x) for x in items))

def prepare(h,memory,point,pset):
    obs,omitted,catalog,mem,prompt,left=A3.prepare(h,memory,point,pset)
    return obs,omitted,catalog,mem,INITIAL_TEMPLATE.format(context=prompt) if prompt else None,left

def parse_initial(content,types,details_allowed):
    def pairs(xs):
        d={}
        for k,v in xs:
            if k in d: raise ValueError("duplicate response key")
            d[k]=v
        return d
    obj=json.loads(content,object_pairs_hook=pairs)
    return A3.parse_decision(json.dumps(obj),types,details_allowed)

def ask_proposal(h, ev, client, settings, memory, point, prev=None, *, workflow="validation-first"):
    """One supplied-transport proposal under the versioned shared contract.
    Every attempt is recorded in attempts.jsonl and the decision record.
    The alternatives workflow requests validation-first explicitly."""
    if workflow not in WORKFLOWS:
        raise SettingsError(f"unknown workflow {workflow!r}")
    preview_on = WORKFLOWS[workflow]["preview"]
    start = time.monotonic()
    clock = h.request("digests")
    pset = A3.prompt_settings(settings)
    obs, omitted, catalog, mem, prompt, left = prepare(h, memory, point, pset)
    rec = new_record(point, clock, "live", mem, prompt, left, pset, workflow)
    attempts, cut, stopped, used = [], [], None, None
    rec.update({"attempts": attempts, "final": None, "batches": [], "outcome": "fallback", "reason": None,
                "refusals": [], "errors": [], "previousExecution": execution_of(obs, prev)})
    ev.append("memory.jsonl", {"point": point, "sha256": rec["memorySha256"], "chars": len(mem), "text": mem})
    if prompt is None:
        rec["errors"].append(f"prompt_too_large: over {settings['maxPromptChars']} characters")
        stopped = "prompt_too_large"
    else:
        types = {t["type"] for t in catalog.get("types", [])}
        messages = [{"role": "system", "content": settings["systemPrompt"]}, {"role": "user", "content": prompt}]
        failures, details_open, turn = 0, settings["allowDetails"], "first"
        inp = {"kind": "prompt", "sha256": sha256(prompt), "chars": len(prompt)}
        previewed = False
        while failures <= settings["maxRetries"]:
            if attempts and time.monotonic() - start >= settings["decisionBudgetSeconds"]:
                rec["errors"].append(f"budget: {settings['decisionBudgetSeconds']} s spent; no further attempt")
                stopped = "budget"
                break
            k = len(attempts) + 1
            at = A3.new_attempt(point, k, turn, inp)
            at["preview"] = None
            payload = client.payload(messages)
            t0 = time.monotonic()
            status, body, err = client.call(payload)
            call = {"point": point, "attempt": k, "turn": turn, "model": settings["model"],
                    "endpoint": settings["endpoint"], "request": payload, "status": status,
                    "latencyMs": round((time.monotonic() - t0) * 1000), "response": body[:A3.MAX_RESPONSE_KEPT],
                    "responseTruncated": len(body) > A3.MAX_RESPONSE_KEPT, "usage": None, "servedModel": None,
                    "content": None, "error": err}
            at["transport"] = "ok" if err is None else err.split(":", 1)[0]
            asked, review, refused = None, None, []
            if err is None:
                try:
                    content, call["usage"], call["servedModel"] = A3.completion(body)
                    call["content"] = content
                    at["output"] = {"sha256": sha256(content), "chars": len(content),
                                    "text": A3.bounded(content, A3.MAX_OUTPUT_TEXT, f"attempt {k} output", cut)}
                    try:
                        d = parse_initial(content, types, details_open)
                    except ValueError as e:
                        at["parse"] = {"ok": False, "code": str(e).split(":", 1)[0],
                                       "message": A3.clip(str(e), A3.MAX_MESSAGE)}
                        raise
                    at["parse"] = {"ok": True}
                    at["reason"], at["actions"] = d["reason"], d["actions"]
                    rec["reason"] = d["reason"]
                    if "details" in d:
                        asked = at["detailsRequested"] = d["details"]
                    else:
                        if d["actions"]:
                            refused = validate_all(h, at, d["actions"], k)
                        else:
                            at["validation"] = []  # nothing to validate
                        if refused:
                            at["refusals"] = refused
                            rec["refusals"] += refused
                            raise ValueError("refused: " + "; ".join(
                                f"action {r['action']}: {r['code']}: {r['message']}" for r in refused))
                        if preview_on and preview_due(at, previewed):
                            previewed = True
                            at["preview"], review = request_preview(h, d["actions"], clock, pset, obs)
                except ProtocolError as e:
                    call["error"] = f"invalid_action: validate answered {e.code}: {e.message}"
                except ValueError as e:
                    call["error"] = str(e)
                    if at["output"] is None:
                        at["transport"] = str(e).split(":", 1)[0]
            ev.call(call)
            if call["error"] is not None:
                at["error"] = A3.bounded(call["error"], A3.MAX_FEEDBACK, f"attempt {k} error", cut)
            elif asked is None and review is None:
                at["used"] = True
            attempts.append(at)
            ev.append("attempts.jsonl", at)
            if asked is not None:
                details_open = False
                text = A3.details_text(h, asked, obs, omitted, catalog, settings["maxPromptChars"] // 4)
                messages = messages + [{"role": "assistant", "content": call["content"]},
                                       {"role": "user", "content": text}]
                turn, inp = "details", {"kind": "details", "sha256": sha256(text), "chars": len(text)}
                continue
            if review is not None:
                details_open = False
                messages = messages + [{"role": "assistant", "content": call["content"]},
                                       {"role": "user", "content": review}]
                turn, inp = "review", {"kind": "preview", "sha256": sha256(review), "chars": len(review)}
                continue
            if at["used"]:
                used = at
                break
            failures += 1
            turn = "retry"
            rec["errors"].append(f"attempt {k}: {call['error']}")
            if at["refusals"]:  # refused by validation: the simulator's results, exactly
                text = feedback_text(refused_items(at["validation"]))
                details_open = False
                messages = messages + [{"role": "assistant", "content": call["content"]},
                                       {"role": "user", "content": text}]
                inp = {"kind": "validation", "sha256": sha256(text), "chars": len(text)}
            elif call["content"] is not None:  # unreadable or a protocol error: version 3's retry message
                text = A3.retry_text(call["error"])
                messages = messages + [{"role": "assistant", "content": call["content"]},
                                       {"role": "user", "content": text}]
                inp = {"kind": "retry", "sha256": sha256(text), "chars": len(text), "feedback": at["error"],
                       "feedbackTruncated": len(call["error"]) > A3.MAX_FEEDBACK}
            else:  # no answer: the same messages again
                inp = {"kind": "resend", "sha256": inp["sha256"], "chars": inp["chars"]}
    if used is not None:
        rec["batches"] = [b for b, _ in A3.split(used["actions"])] if used["actions"] else []
        rec["outcome"] = "actions" if rec["batches"] else "no_action"
    else:
        rec["batches"], rec["outcome"] = [], "fallback"  # never a batch that was not validated
    # Validation and preview change nothing: the state must still be the decision point's.
    after = h.request("digests").get("digests")
    rec["digestsBeforeSubmit"] = {"unchanged": after == clock["digests"]}
    if after != clock["digests"] and rec["batches"]:
        rec["errors"].append("state_changed: the digests changed during the decision; nothing is submitted")
        rec["batches"], rec["outcome"] = [], "fallback"
    rec["final"] = {"attempt": used["attempt"] if used else None, "outcome": rec["outcome"]}
    rec["stages"] = stages(attempts, rec["batches"], rec["outcome"])
    rec["completeness"] = completeness(attempts, stopped, cut)
    return rec["batches"], rec



class Capture(LEGACY.Capture):
    def __init__(self, model):
        super().__init__(model)
        self.deliveries=[]
    def call(self,payload):
        status,body,err=super().call(payload)
        self.completionError=None
        if err is None and self.content is None:
            try:A3.completion(body)
            except ValueError as e:self.completionError=str(e)
        self.deliveries.append({"messages":json.loads(json.dumps(payload["messages"])),"status":status,"body":body,"error":err,"content":self.content})
        return status,body,err

def ask(h,ev,model,settings,memory,point,prev=None,*,arm):
    if arm not in ARMS: raise SettingsError("unknown arm")
    settings=dict(settings,systemPrompt=SYSTEM_PROMPT)
    cap=Capture(model)
    batches,base=ask_proposal(h,ev,cap,settings,memory,point,prev,workflow="validation-first")
    atts=base["attempts"]
    used=LEGACY.used_attempt(base)
    correction=bool(len(atts)>1 and (atts[0].get("refusals") or atts[0].get("protocolError")))
    rec={"schema":RECORD_SCHEMA,"workflowVersion":WORKFLOW_VERSION,"workflowSha256":WORKFLOW_SHA256,
         "contractPins":CONTRACT_PINS,"arm":arm,"validationFirst":base,"turn":None,"exchanges":[],"outcome":None,
         "result":"not_due:validation_first_"+base["outcome"],"batches":batches,"errors":[],"inspected":[],
         "eligibility":{"eligible":None,"reason":"comparison not reached"},"exposure":False,
         "validationFirstCorrection":correction,"correctionFailure":correction and used is None,
         "deliberateNoAction":base["outcome"]=="no_action","processFailure":any((a.get("parse") or {}).get("ok") is False or a.get("transport") not in (None,"ok") for a in atts) or bool((base.get("completeness") or {}).get("stopped")),
         "retryRequests":sum(a["input"]["kind"] in ("validation","retry","resend") for a in atts[1:]),"deliveries":cap.deliveries,
         "intendedAction":None if not atts or atts[-1].get("actions") is None else [b for b,_ in A3.split(atts[-1]["actions"])],
         "validatedAction":batches if used else None,"executedAction":{"status":"NOT_EXECUTED"}}
    if base["outcome"]=="actions" and used and cap.messages is not None and cap.content is not None:
        actions=used["actions"]
        rec["proposalSubmittedAction"]=h.request("alternatives.v2.canonical",{"batches":batches})
        turn=h.request("alternatives.v2.turn",{"arm":arm,"actions":actions})
        rec["turn"]=turn
        rec["eligibility"]={"eligible":bool(turn["due"]),"reason":turn["why"]}
        if not turn["due"]: rec["result"]=turn["why"]
        else:
            rec["exposure"]=True
            clock=h.request("digests")
            messages=list(cap.messages)+[{"role":"assistant","content":cap.content},{"role":"user","content":turn["message"]}]
            inspected,rounds,failures,outcome=[],0,0,None
            while outcome is None and failures<=TURN_RETRIES:
                if rec["exchanges"]: rec["retryRequests"]+=int(rec["exchanges"][-1]["kind"] in ("unreadable","inspect_again","transport","parse"))
                status,body,err=cap.call(cap.payload(messages))
                content=cap.content
                x={"n":len(rec["exchanges"])+1,"kind":"transport","status":status,"error":err or cap.completionError}
                rec["exchanges"].append(x)
                if content is None:
                    x["kind"]="transport" if err is not None else "parse"
                    rec["processFailure"]=True;failures+=1;continue
                ans=h.request("alternatives.v2.decide",{"arm":arm,"actions":actions,"inspected":inspected,"round":rounds,"content":content})
                x.update(kind=ans["kind"],parse=ans["parse"],answerSha256=ans["answerSha256"])
                if ans["kind"]=="decision": outcome=ans["outcome"];break
                if ans["kind"]=="inspect":
                    rounds+=1
                    inspected += [i for i in ans["inspected"] if i not in inspected]
                else:
                    rec["processFailure"]=True;failures+=1
                messages=messages+[{"role":"assistant","content":content},{"role":"user","content":ans["answer"]}]
            rec["inspected"]=inspected
            rec["outcome"]=outcome
            if outcome is None:
                rec["responseRoute"]=(rec["exchanges"][-1].get("parse") or {}).get("kind", "transport")
                batches=[];rec["result"]="no_decision";rec["validatedAction"]=None;rec["intendedAction"]=None
            else:
                rec["result"]=outcome["result"]
                rec["responseRoute"]=outcome["responseRoute"]
                rec["intendedAction"]=outcome["intendedAction"]
                rec["validatedAction"]=outcome["validatedAction"]
                batches=outcome["batches"] if outcome["valid"] else []
                rec["deliberateNoAction"]=bool(outcome["valid"] and not batches and outcome["responseRoute"] in ("abandon","revise","select"))
            after=h.request("digests")["digests"]
            rec["digestsBeforeSubmit"]={"unchanged":after==clock["digests"]}
            if after!=clock["digests"]:
                rec["errors"].append("state changed before submission");rec["processFailure"]=True;batches=[]
    elif used is not None:
        rec["eligibility"]={"eligible":False,"reason":"valid no action before comparison"}
    rec["batches"]=batches
    rec["submittedAction"]=h.request("alternatives.v2.canonical",{"batches":batches})
    return batches,rec
