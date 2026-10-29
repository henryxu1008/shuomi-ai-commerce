import {join} from 'node:path';
import {json,readJSON} from './io.mjs';

function errorDetails(error,seen=new Set()) {
 if(error===null||typeof error!=='object')return {thrown:error};
 if(seen.has(error))return {circular:true};
 seen.add(error);
 const result={};
 for(const key of ['name','message','stack','code','errno','syscall','address','port','path','dest','privateOutput'])if(error[key]!==undefined)result[key]=error[key];
 if(error.cause!==undefined)result.cause=errorDetails(error.cause,seen);
 if(error instanceof AggregateError)result.errors=error.errors.map(item=>errorDetails(item,seen));
 return result;
}

// Called only inside the existing protected run directory and owner lock.
// Keep raw errors separate from the public manifest and console. This file is
// retained evidence, so cleanup must not include it in the private-file deletion list.
export async function withFailureEvidence(m,phase,operation) {
 let stage='begin';
 try{return await operation(value=>{stage=value})}catch(error){
  try {
   const file=join(m.directory,'failure-details.json');
   const evidence=await readJSON(file).catch(e=>{
    if(e.code!=='ENOENT')throw e;
    return {schemaVersion:'issue357-failure-v1',runId:m.runId,sourceSha:m.sourceSha,failures:[]};
   });
   evidence.failures.push({phase,stage,at:new Date().toISOString(),error:errorDetails(error)});
   await json(file,evidence);
  }catch{
   // Never mask the operation error or leak a diagnostic write error's private details.
   console.error(`FAILURE_EVIDENCE_INCOMPLETE phase=${phase}; run=${m.runId}`);
  }
  throw error;
 }
}
