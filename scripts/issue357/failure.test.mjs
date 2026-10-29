import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,readFile,writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {withFailureEvidence} from './failure.mjs';

test('protected evidence preserves original error identity, causes and prior failures',async()=>{
 const directory=await mkdtemp(join(tmpdir(),'issue357-failure-unit-'));
 const m={directory,runId:'fixture',sourceSha:'a'.repeat(40)};
 const first=new Error('first',{cause:Object.assign(new Error('cause'),{code:'EACCES'})});
 const second=new AggregateError([Object.assign(new Error('child'),{code:'EPERM'})],'second',{cause:first});
 for(const [phase,error] of [['initialize',first],['cleanup',second]]) {
  await assert.rejects(withFailureEvidence(m,phase,async stage=>{stage('fixture-stage');throw error}),caught=>caught===error);
 }
 const evidence=JSON.parse(await readFile(join(directory,'failure-details.json'),'utf8'));
 assert.equal(evidence.failures.length,2);
 assert.equal(evidence.failures[0].error.cause.code,'EACCES');
 assert.equal(evidence.failures[1].error.errors[0].code,'EPERM');
 assert.equal(evidence.failures[1].error.cause.message,'first');
 assert.equal(evidence.failures[0].stage,'fixture-stage');
});

test('unavailable evidence storage never replaces the original exception',async t=>{
 const root=await mkdtemp(join(tmpdir(),'issue357-failure-unit-'));
 const directory=join(root,'not-a-directory');await writeFile(directory,'fixture');
 const output=[];t.mock.method(console,'error',line=>output.push(line));
 const error=new Error('SYNTHETIC_PRIVATE_ORIGINAL');
 await assert.rejects(withFailureEvidence({directory,runId:'fixture'},'cleanup',async()=>{throw error}),caught=>caught===error);
 assert.deepEqual(output,['FAILURE_EVIDENCE_INCOMPLETE phase=cleanup; run=fixture']);
});
