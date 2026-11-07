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
import v13g as D
import v13g_fidelity as F
import v13d as OLD

ROOT=Path(D.ROOT)
V=ROOT/'docs/benchmark/v13g-dev'
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
        pins=json.loads((ROOT/'docs/v13g/CONTRACT-PINS.json').read_text())
        for path,want in pins['files'].items():
            self.assertEqual(hashlib.sha256((ROOT/path).read_bytes()).hexdigest(),want,path)
        index=json.loads((ROOT/'docs/v13g/CONTRACT-IDENTITIES.json').read_text())
        for files in index['identities'].values():
            for path,want in files.items():
                self.assertEqual(hashlib.sha256((ROOT/path).read_bytes()).hexdigest(),want,path)
        self.assertEqual(pins['parserVersionSha256'],D.sha256(D.ALT.PARSER_VERSION))
        import re
        go_version=re.search(r'ParserV3ID\s*=\s*"([^"]+)"',(ROOT/'internal/alternatives/validation_v3.go').read_text())[1]
        self.assertEqual(go_version,D.ALT.PARSER_VERSION)
        self.assertEqual(D.ALT.SCHEMAS['parser'],'dreamteam.phase-response-parser/v2')
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
        for line in (ROOT/'docs/v13g/BASELINE-HASHES.sha256').read_text().splitlines():
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
        with tempfile.TemporaryDirectory(prefix='v13g-unverified-') as tmp:
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
        with tempfile.TemporaryDirectory(prefix='v13g-partial-') as tmp:
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
        with tempfile.TemporaryDirectory(prefix='v13g-aborted-') as tmp:
            doc=D.run_schedule(self.proto,'scripted',sched,self.ctxs,self.ctxs.opener(BINARY),{D.QWEN:broken},tmp,'scripted test',now=lambda:0.0,utc=lambda:None)
            r=json.loads(Path(tmp,'live/aborted-draw.json').read_text())
            D.check_run_doc(doc,[],sched,'scripted',self.proto,r)
            with self.assertRaises(D.ExperimentError):D.check_run_doc(doc,[],sched,'scripted',self.proto,None)
            an=D.analyze(BINARY,{},self.ctxs,self.proto,sched,[],doc,'scripted',r)
            self.assertEqual(an['abortedVerification']['status'],'RECONSTRUCTED')
            self.assertFalse(an['armDifferences'])

if __name__=='__main__':unittest.main()

class FinalRemediation(unittest.TestCase):
    setUpClass=classmethod(Engine.setUpClass.__func__)
    tearDownClass=classmethod(Engine.tearDownClass.__func__)
    draw=Engine.draw
    def test_exact_execution_attribution(self):
        ext={'type':'extend_shift','employee':'E03','minutes':30}
        ds=[self.draw('S1-a','control',[F.REVISE([a])]) for a in (ext,dict(ext,date='2026-10-13'))]
        for i,d in enumerate(ds,1):d.update(seq=i,draw=i)
        sched=[{k:d[k] for k in ('seq','context','model','arm','draw')} for d in ds]
        doc={'schema':D.RUN_SCHEMA,'driver':D.DRIVER,'protocol':self.proto['id'],'protocolSha256':'test',
             'status':'COMPLETE','drawsScheduled':2,'drawsCompleted':2,'drawInterrupted':None,'drawAborted':None,
             'drawsNotRun':0,'modelCalls':2,'stopped':None,'resume':'never'}
        an=D.analyze(BINARY,{},self.ctxs,self.proto,sched,ds,doc,'test')
        self.assertEqual(len(an['executions']),2)
        self.assertEqual(ds[0]['record']['submittedAction']['sha256'],ds[1]['record']['submittedAction']['sha256'])
        self.assertNotEqual(an['draws'][0]['executionRecord'],an['draws'][1]['executionRecord'])
        for d,x in zip(ds,an['draws']):
            actual=D.execute(BINARY,{},self.ctxs.by_id['S1-a'],d['batches'])
            self.assertEqual(actual,x['executedAction'])
            ledger=an['executions'][x['executionRecord']]
            self.assertEqual(ledger['draws'],[{k:d[k] for k in ('seq','context','model','arm','draw')}])
            self.assertEqual(ledger['batches'],d['batches'])
            apply=next(e for e in actual['engine'] if e['op']=='apply')
            self.assertEqual(apply['args']['actions'],d['batches'][0])
        aa,bb=[x['executedAction'] for x in an['draws']]
        self.assertNotEqual(aa['digestsAfterApply'],bb['digestsAfterApply'])
        # Persisting sorted record objects must not discard submission key order.
        for d in ds:
            restored=json.loads(json.dumps(d,sort_keys=True))
            self.assertEqual(restored['submittedRepresentation'],d['submittedRepresentation'])
        left=json.dumps([[{'type':'set_walkin_policy','policy':'schedule_later'}]])
        right=json.dumps([[{'policy':'schedule_later','type':'set_walkin_policy'}]])
        self.assertEqual(json.loads(left),json.loads(right))
        self.assertNotEqual(D.sha256(left),D.sha256(right))
        c=self.ctxs.by_id['S1-a']
        self.assertEqual(D.execute(BINARY,{},c,json.loads(left)),D.execute(BINARY,{},c,json.loads(right)))
        self.assertEqual(aa['observationAfterDay']['observation']['finance'],bb['observationAfterDay']['observation']['finance'])
        self.assertNotEqual(aa['observationAfterDay']['observation']['actions'],bb['observationAfterDay']['observation']['actions'])

    def test_initialization_failures_are_durable(self):
        import subprocess
        sched=[{'seq':1,'model':D.QWEN,'context':'S1-a','arm':'control','draw':1}]
        for failure in (RuntimeError('opening failure'),TimeoutError('opening timeout'),
                        FileNotFoundError('engine absent'),subprocess.CalledProcessError(3,['engine']),KeyboardInterrupt()):
            class Broken:
                def __call__(self,c,arm):raise failure
                def close(self):pass
            with self.subTest(failure=type(failure).__name__), tempfile.TemporaryDirectory(prefix='v13g-opening-') as tmp:
                doc=D.run_schedule(self.proto,'test',sched,self.ctxs,Broken(),{D.QWEN:D.scripted([])},tmp,'scripted',now=lambda:0.,utc=lambda:None)
                aborted=json.loads(Path(tmp,'live/aborted-draw.json').read_text())
                self.assertEqual((doc['status'],doc['drawAborted'],doc['modelCalls']),('PARTIAL',1,0))
                self.assertEqual((aborted['failureStage'],aborted['engineInitialization'],aborted['modelCallsCompleted']),('engine_open','FAILED',0))
                self.assertEqual(aborted['engine'],[])
                an=D.analyze(BINARY,{},self.ctxs,self.proto,sched,[],doc,'test',aborted)
                self.assertEqual(an['abortedVerification']['status'],'UNVERIFIED')
                self.assertEqual((an['draws'],an['armDifferences'],an['pairs']),([],[],[]))

    def test_actual_engine_process_open_failure(self):
        sched=[{'seq':1,'model':D.QWEN,'context':'S1-a','arm':'control','draw':1}]
        class FailedProcess:
            def __call__(self,c,arm):return D.Fresh(BINARY,dict(c,scenario='missing-scenario-input.json'))
            def close(self):pass
        with tempfile.TemporaryDirectory(prefix='v13g-process-') as tmp:
            doc=D.run_schedule(self.proto,'test',sched,self.ctxs,FailedProcess(),{D.QWEN:D.scripted([])},tmp,'scripted',now=lambda:0.,utc=lambda:None)
            r=json.loads(Path(tmp,'live/aborted-draw.json').read_text())
            self.assertEqual((r['status'],r['failureStage'],r['modelCallsCompleted']),('aborted','engine_open',0))
            an=D.analyze(BINARY,{},self.ctxs,self.proto,sched,[],doc,'test',r)
            self.assertEqual(an['abortedVerification']['status'],'UNVERIFIED')

    def test_numeric_workflow_and_replay(self):
        for spelling in ('30','30.0','3e1'):
            content='{"reason":"catalog compliant","actions":[{"type":"extend_shift","employee":"E03","minutes":'+spelling+'}]}'
            d=self.draw('S1-a','control',[content])
            self.assertEqual(d['record']['result'],'revised')
            self.assertEqual(len(d['calls']),1)
            self.assertEqual(d['batches'][0][0]['minutes'],30)
            c=self.ctxs.by_id['S1-a']
            D.replay_draw(d,c,self.ctxs.settings[D.QWEN],self.pool,self.ctxs.memory(c,D.QWEN))
        content='{"reason":"precision","actions":[{"type":"set_walkin_policy","policy":"schedule_later","minutes":9007199254740993.0}]}'
        parsed=D.ALT.parse_initial(content,{'set_walkin_policy'},False)
        self.assertEqual(parsed['actions'][0]['minutes'],9007199254740993)

    def test_collection_equivalence_and_large_integers_through_engine(self):
        shift={'start':'08:00','end':'12:00','assignment':'R1','breaks':[],'covers':[]}
        action={'type':'set_roster','op':'set','employee':'E03','effective':'2026-11-02',
                'roster':{'A':{'Mon':shift},'B':{'Mon':copy.deepcopy(shift)}}}
        variants=[action]
        for mode in ('null','omitted'):
            a=copy.deepcopy(action)
            for week in a['roster'].values():
                for field in ('breaks','covers'):
                    if mode=='null':week['Mon'][field]=None
                    else:del week['Mon'][field]
            variants.append(a)
        ds=[self.draw('S1-a','control',[F.REVISE([a])]) for a in variants]
        self.assertTrue(all(d['record']['outcome']['valid'] for d in ds))
        self.assertEqual(len({d['record']['submittedAction']['sha256'] for d in ds}),1)
        c=self.ctxs.by_id['S1-a'];ses=D.Fresh(BINARY,c)
        try:
            h=D.Harness(ses)
            self.assertTrue(h.request('apply',{'actions':[action]})['applied'])
            for a in variants:self.assertEqual(h.request('validate',{'actions':[a]})['results'][0]['code'],'unchanged')
        finally:ses.close()
        ids=[]
        for n in (9007199254740992,9007199254740993):
            d=self.draw('S1-a','control',[F.REVISE([dict(F.OPS,minutes=n)])])
            self.assertTrue(d['record']['outcome']['valid'])
            ids.append(d['record']['submittedAction']['sha256'])
            D.replay_draw(d,c,self.ctxs.settings[D.QWEN],self.pool,self.ctxs.memory(c,D.QWEN))
        self.assertNotEqual(*ids)

    def test_stop_events_bidirectional(self):
        sched=[{'seq':1,'model':D.QWEN,'context':'S1-a','arm':'control','draw':1}]
        for state_fn in (lambda:{'health':{'status':'unhealthy'}},None):
            now=iter([0.,20000.,20000.,20000.,20000.])
            with tempfile.TemporaryDirectory(prefix='v13g-stop-') as tmp:
                doc=D.run_schedule(self.proto,'test',sched,self.ctxs,self.ctxs.opener(BINARY),{D.QWEN:D.scripted([])},tmp,'scripted',state_fn=state_fn,now=lambda:next(now,20000.),utc=lambda:None)
                event=json.loads(Path(tmp,'live/stop-event.json').read_text())
                self.assertEqual(event,doc['stopEvent'])
                D.check_stop_artifact(str(Path(tmp,'live')),doc)
                event_path=Path(tmp,'live/stop-event.json')
                event_path.unlink()
                with self.assertRaises(D.ExperimentError):D.check_stop_artifact(str(Path(tmp,'live')),doc)
                event_path.write_text(json.dumps(dict(event,modelCalls=999)))
                with self.assertRaises(D.ExperimentError):D.check_stop_artifact(str(Path(tmp,'live')),doc)
                D.check_run_doc(doc,[],sched,'test',self.proto)
                for mutate in (lambda d:d.pop('stopEvent'),lambda d:d['stopEvent'].update(nextDraw=dict(sched[0],seq=999)),
                               lambda d:d.update(stopped={'code':'operator'}),lambda d:d.update(stopped={'code':'driver'}),
                               lambda d:d.update(drawsNotRun=-1),lambda d:d.update(modelCalls=1),lambda d:d['stopEvent'].update(modelCalls=False),lambda d:d.update(status='COMPLETE')):
                    bad=copy.deepcopy(doc);mutate(bad)
                    with self.assertRaises(D.ExperimentError):D.check_run_doc(bad,[],sched,'test',self.proto)

class Coverage(unittest.TestCase):
    def test_v11b_mismatch_fails_ci_command(self):
        import subprocess
        script=ROOT/'analysis/alternatives/ci_compare.py'
        workflow=(ROOT/'.github/workflows/ci.yml').read_text()
        self.assertIn('analysis/alternatives/ci_compare.py --generated v11b',workflow)
        self.assertLess(workflow.index('git worktree remove .git/v13e-candidate'),workflow.index('- name: High-impact action acknowledgment'))
        with tempfile.TemporaryDirectory(prefix='v13g-ci-') as tmp:
            root=Path(tmp);gen=root/'gen';ref=root/'ref';gen.mkdir();ref.mkdir()
            files=['decisions.json','decision-tables.md','inference.json']
            for f in files:(gen/f).write_text('same');(ref/f).write_text('same')
            args=[sys.executable,'-I','-B',str(script),'--generated',str(gen),'--reference',str(ref),'--out',str(root/'out'),*files]
            self.assertEqual(subprocess.run(args,capture_output=True).returncode,0)
            for f in files:
                (gen/f).write_text('mismatch')
                self.assertNotEqual(subprocess.run(args,capture_output=True).returncode,0,f)
                (gen/f).write_text('same')
