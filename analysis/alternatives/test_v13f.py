"""Deterministic remediation regressions; transport replies are scripted."""
import collections
import copy
import hashlib
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0,str(Path(__file__).resolve().parent))
import v13f as D
import v13f_fidelity as F
import v13d as OLD

ROOT=Path(D.ROOT)
V=ROOT/'docs/benchmark/v13f-dev'
BINARY=os.environ.get('DREAMTEAM_BINARY')

class Pure(unittest.TestCase):
    def test_protocol_budget_and_duplicate_inputs(self):
        rows,csha=D.load_contexts(str(V/'contexts'))
        proto,sched=D.draft_protocol(rows,csha)
        self.assertEqual((len(rows),len(sched),proto['maxModelCalls']),(21,262,802))
        self.assertEqual(proto,json.loads((V/'protocol-draft.json').read_text()))
        self.assertEqual(sched,json.loads((V/'schedule-draft.json').read_text())['schedule'])
        self.assertEqual(sum(c['kind']=='historical' for c in rows),9)
        a,b=({c['id']:c for c in rows}[k] for k in ('S1-a','S6-a'))
        self.assertEqual((a['prompt'],a['proposal']),(b['prompt'],b['proposal']))
        self.assertEqual(proto['duplicateInputs'],[['S1-a','S6-a']])
        self.assertIn('complete alternatives interface',proto['intervention'])
        self.assertIn('request-start guard',proto['stop']['timeGuarantee'])
        self.assertIn('canonical submitted',proto['analysis']['estimand'])
        for c in rows:
            D.ALT.parse_initial(c['proposal']['content'],set(a['type'] for a in c['proposal']['actions']),False)
    def test_contract_pins(self):
        pins=json.loads((ROOT/'docs/v13f/CONTRACT-PINS.json').read_text())
        for path,want in pins['files'].items():
            self.assertEqual(hashlib.sha256((ROOT/path).read_bytes()).hexdigest(),want,path)
        index=json.loads((ROOT/'docs/v13f/CONTRACT-IDENTITIES.json').read_text())
        for files in index['identities'].values():
            for path,want in files.items():
                self.assertEqual(hashlib.sha256((ROOT/path).read_bytes()).hexdigest(),want,path)
        self.assertEqual(pins['parserVersionSha256'],D.sha256(D.ALT.PARSER_VERSION))
        import re
        go_version=re.search(r'ParserV2ID\s*=\s*"([^"]+)"',(ROOT/'internal/alternatives/turn_v2.go').read_text())[1]
        self.assertEqual(go_version,D.ALT.PARSER_VERSION)
        self.assertEqual(D.ALT.SCHEMAS['parser'],D.ALT.PARSER_VERSION)
        self.assertEqual(D.ALT.SCHEMAS['phases']['final_decision'],['select','revise','abandon','whole_reply_revision'])
    def test_request_start_guard(self):
        now=[0.0]
        lim=D.Limits(6,802,14400,300,now=lambda:now[0])
        now[0]=14100;lim.before_call(0)
        now[0]=14100.1
        with self.assertRaises(D.Halt) as e:lim.before_call(0)
        self.assertEqual(e.exception.code,'wall_clock')
        now[0]=14100.01
        with self.assertRaises(D.Halt) as precise:lim.before_call(0)
        self.assertGreater(precise.exception.evidence['elapsedSeconds'],14100)
        D.check_halt({'code':precise.exception.code,'evidence':precise.exception.evidence},
                     {'maxModelCallsPerDraw':6,'maxModelCalls':802,'request':{'timeoutSeconds':300},'stop':{'wallClockSeconds':14400,'consecutiveTransportFailures':5}})
        # No watchdog interrupts an engine operation: this is documented as
        # a request-start guard rather than a hard end-to-end deadline.
        self.assertEqual(lim.calls,0)
    def test_primary_excludes_nonexposed(self):
        c=[{'exposure':True,'batchesSha256':'empty'}]*5+[{'exposure':False,'batchesSha256':'other'}]*9
        t=[{'exposure':True,'batchesSha256':'empty'}]*5
        got=D.primary_comparison(c,t)
        self.assertEqual((got['controlExposed'],got['treatmentExposed'],got['totalVariation']),(5,5,0))
        self.assertIsNone(D.primary_comparison([],t))
    def test_legacy_partial_counterexample(self):
        doc={'status':'PARTIAL','drawsCompleted':1,'drawsScheduled':2,'drawInterrupted':None,'drawAborted':999,'drawsNotRun':-77,'modelCalls':999,'stopped':{'code':'invented'},'resume':'never'}
        rows=[{'seq':1,'status':'completed'}]
        OLD.check_run_doc(doc,rows,[{'seq':1},{'seq':2}])
        with self.assertRaises(D.ExperimentError):D.check_run_doc(doc,rows,[{'seq':1},{'seq':2}])
    def test_historical_bytes(self):
        for line in (ROOT/'docs/v13f/BASELINE-HASHES.sha256').read_text().splitlines():
            want,path=line.split('  ',1)
            self.assertEqual(hashlib.sha256((ROOT/path).read_bytes()).hexdigest(),want,path)

@unittest.skipUnless(BINARY,'DREAMTEAM_BINARY is not set')
class Engine(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        rows,csha=D.load_contexts(str(V/'contexts'))
        cls.proto,_=D.draft_protocol(rows,csha)
        env={D.SB.SETTINGS_ENV:D.SB.REPLAY_URL}
        s=D.A3.load_settings(str(ROOT/D.settings_files()[D.QWEN]['file']),env)
        for c in F.extra_contexts():
            prompt,dg=D.prompt_at(BINARY,c,s)
            c.update(prompt=prompt,promptSha256=D.sha256(prompt),digests=dg)
            rows.append(c)
        cls.ctxs=D.Contexts(cls.proto,[c for c in rows if c['kind']=='constructed'],None,env)
        cls.pool=cls.ctxs.opener(BINARY)
    @classmethod
    def tearDownClass(cls):cls.pool.close()
    def draw(self,cid,arm,replies):
        c=self.ctxs.by_id[cid]
        entry={'seq':1,'model':D.QWEN,'context':cid,'arm':arm,'draw':1}
        return D.run_draw(entry,c,self.ctxs.settings[D.QWEN],self.pool,self.ctxs.memory(c,D.QWEN),D.scripted(replies),None)
    def test_fidelity_record_and_combined_inspect(self):
        for case in F.CASES:
            if case[0] in ('partial termination commitments','refused initial proposal'):
                result,_=F.run_case(BINARY,{},self.ctxs,self.pool,case)
                self.assertEqual(result['failed'],[],result)

    def test_unverifiable_aborted_exchange_is_labelled(self):
        sched=[{'seq':1,'model':D.QWEN,'context':'S1-a','arm':'control','draw':1}]
        class BrokenPool:
            def __call__(inner,c,arm):return F.DropField(self.pool(c,arm),'message')
            def close(inner):pass
        with tempfile.TemporaryDirectory(prefix='v13f-unverified-') as tmp:
            doc=D.run_schedule(self.proto,'scripted',sched,self.ctxs,BrokenPool(),{D.QWEN:D.scripted([])},tmp,'scripted test',now=lambda:0.0,utc=lambda:None)
            r=json.loads(Path(tmp,'live/aborted-draw.json').read_text())
            an=D.analyze(BINARY,{},self.ctxs,self.proto,sched,[],doc,'scripted',r)
            self.assertEqual(an['abortedVerification']['status'],'UNVERIFIED')

    def test_exact_audit_action_counterexample(self):
        a=self.draw('S1-a','control',[F.ABANDON])
        b=self.draw('S1-a','treatment',[F.SEL('no_action')])
        r=self.draw('S1-a','control',[F.REVISE([])])
        ca,cb,cr=map(D.classify,(a,b,r))
        self.assertEqual({x['batchesSha256'] for x in (ca,cb,cr)},{ca['batchesSha256']})
        self.assertEqual(D.primary_comparison([ca]*5,[cb]*5)['totalVariation'],0)
        self.assertNotEqual(ca['secondaryRoute'],cb['secondaryRoute'])
        self.assertTrue(cr['deliberateNoAction'])
        self.assertFalse(cr['processFailure'])
    def test_completion_parse_failure_is_separate(self):
        c=self.ctxs.by_id['S1-a']
        calls=[(200,'{',None),(200,json.dumps({'choices':[{'message':{'content':F.SEL('proposal')}}]}),None)]
        def transport(payload):return calls.pop(0)
        entry={'seq':1,'model':D.QWEN,'context':'S1-a','arm':'control','draw':1}
        d=D.run_draw(entry,c,self.ctxs.settings[D.QWEN],self.pool,self.ctxs.memory(c,D.QWEN),transport,None)
        self.assertEqual(d['record']['exchanges'][0]['kind'],'parse')
        self.assertEqual(d['record']['retryRequests'],1)
        self.assertTrue(d['record']['processFailure'])
        self.assertTrue(d['record']['exposure'])
        D.replay_draw(d,c,self.ctxs.settings[D.QWEN],self.pool,self.ctxs.memory(c,D.QWEN))

    def test_full_workflow_system_schema_precedence(self):
        d=self.draw('S1-a','control',[json.dumps({'reason':'system compliant','actions':[]})])
        self.assertEqual(d['record']['result'],'revised')
        self.assertEqual(len(d['calls']),1)
        self.assertEqual(d['record']['responseRoute'],'revise')
        system=d['calls'][0]['request']['messages'][0]['content']
        lower=d['calls'][0]['request']['messages'][-1]['content']
        self.assertIn('takes precedence',system)
        self.assertIn('"decision": "revise"',lower)
        self.assertIn('Whole-reply revision',system)
    def test_correction_failure_is_not_inaction(self):
        bad=self.draw('F-refused','control',['not json'])
        good=self.draw('F-refused','treatment',[json.dumps({'reason':'no change','actions':[]})])
        a,b=map(D.classify,(bad,good))
        self.assertTrue(a['validationFirstCorrection'] and a['correctionFailure'] and a['processFailure'])
        self.assertFalse(a['deliberateNoAction'] or a['exposure'])
        self.assertTrue(b['validationFirstCorrection'] and b['deliberateNoAction'])
        self.assertFalse(b['correctionFailure'] or b['processFailure'] or b['exposure'])
        self.assertEqual((a['modelCalls'],a['retries'],b['modelCalls'],b['retries']),(1,1,1,1))
        invalid=self.draw('F-refused','control',[json.dumps({'reason':'still invalid','actions':[{'type':'terminate','employee':'E04','lastDay':'2026-10-14'}]})])
        x=D.classify(invalid)
        self.assertTrue(x['correctionFailure'])
        self.assertFalse(x['processFailure'] or x['deliberateNoAction'] or x['exposure'])
    def test_partial_run_metadata_and_suppression(self):
        c=self.ctxs.by_id['S1-a']
        proto=copy.deepcopy(self.proto)
        proto['id']='scripted-partial'
        proto['maxModelCalls']=1
        sched=[{'seq':i+1,'model':D.QWEN,'context':'S1-a','arm':arm,'draw':1} for i,arm in enumerate(D.ARMS)]
        with tempfile.TemporaryDirectory(prefix='v13f-partial-') as tmp:
            doc=D.run_schedule(proto,'scripted',sched,self.ctxs,self.ctxs.opener(BINARY),{D.QWEN:D.scripted([F.SEL('proposal')])},tmp,'scripted test',now=lambda:0.0,utc=lambda:None)
            rows=D.SB.load_jsonl(os.path.join(tmp,'live/draws.jsonl'))
            D.check_run_doc(doc,rows,sched,'scripted',proto)
            an=D.analyze(BINARY,{},self.ctxs,proto,sched,rows,doc,'scripted')
            self.assertEqual((an['runStatus'],an['armDifferences'],an['pairs']),('PARTIAL',[],[]))
            for field,value in (('drawAborted',999),('drawsNotRun',-77),('modelCalls',999),('stopped',{'code':'invented'}),('status','COMPLETE'),('schema','invented'),('driver','invented')):
                bad=copy.deepcopy(doc);bad[field]=value
                with self.assertRaises(D.ExperimentError,msg=field):D.check_run_doc(bad,rows,sched,'scripted',proto)
            bad_doc=copy.deepcopy(doc);bad_rows=copy.deepcopy(rows)
            bad_doc['stopped']['code']='transport';bad_doc['stopped']['evidence']['consecutiveTransportFailures']=5
            bad_rows[-1]['halt']=copy.deepcopy(bad_doc['stopped'])
            with self.assertRaises(D.ExperimentError):D.check_run_doc(bad_doc,bad_rows,sched,'scripted',proto)
    def test_aborted_artifact_and_reconstruction(self):
        sched=[{'seq':1,'model':D.QWEN,'context':'S1-a','arm':'control','draw':1}]
        def broken(payload):raise RuntimeError('scripted transport abort')
        with tempfile.TemporaryDirectory(prefix='v13f-aborted-') as tmp:
            doc=D.run_schedule(self.proto,'scripted',sched,self.ctxs,self.ctxs.opener(BINARY),{D.QWEN:broken},tmp,'scripted test',now=lambda:0.0,utc=lambda:None)
            r=json.loads(Path(tmp,'live/aborted-draw.json').read_text())
            D.check_run_doc(doc,[],sched,'scripted',self.proto,r)
            with self.assertRaises(D.ExperimentError):D.check_run_doc(doc,[],sched,'scripted',self.proto,None)
            an=D.analyze(BINARY,{},self.ctxs,self.proto,sched,[],doc,'scripted',r)
            self.assertEqual(an['abortedVerification']['status'],'RECONSTRUCTED')
            self.assertFalse(an['armDifferences'])

if __name__=='__main__':unittest.main()
