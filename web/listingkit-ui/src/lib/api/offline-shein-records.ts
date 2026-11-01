import {z} from "zod";
import {offlineCreate,offlineReceipt,offlineSaved,offlineIntent,type OfflineIntent} from "@/lib/contracts/offline-shein-records";
import type {ReportScope} from "@/lib/contracts/report-center";
import {readBoundedStrictJSON} from "./strict-json-response";
import {parseOfflineRecordFailure} from "./offline-shein-record-errors";
export class OfflineRecordError extends Error{constructor(public code:string){super(code)}}
export async function offlineRequest<T>(scope:ReportScope,path:string,schema:z.ZodType<T>,signal?:AbortSignal,intent?:OfflineIntent):Promise<T>{
 let dispatched=false;
 try{
 const headers=new Headers({Accept:"application/json","X-Expected-User-ID":scope.userId,"X-Expected-Organization-ID":scope.organizationId});
 if(intent){offlineIntent.parse(intent);headers.set("Content-Type","application/json");headers.set("Idempotency-Key",intent.key);}
 const timeout=AbortSignal.timeout(15000),bound=signal?AbortSignal.any([signal,timeout]):timeout;bound.throwIfAborted();dispatched=true;
 const response=await fetch(`/api/offline-shein-records${path}`,{method:intent?"POST":"GET",headers,body:intent?JSON.stringify(offlineCreate.parse({collectionItemId:intent.collectionItemId,input:intent.input})):undefined,signal:bound,credentials:"same-origin",cache:"no-store",redirect:"error"});
 const raw=await readBoundedStrictJSON(response,(2<<20)+4096,bound);
 if(response.status!==(intent?201:200)){const failure=parseOfflineRecordFailure(raw,response.status);throw new OfflineRecordError(intent&&response.status>=500?"OUTCOME_UNKNOWN":failure?failure.code:intent?"OUTCOME_UNKNOWN":"DEPENDENCY_UNAVAILABLE");}
 const result=schema.safeParse(raw);if(!result.success)throw new OfflineRecordError(intent?"OUTCOME_UNKNOWN":"INVALID_UPSTREAM_RESPONSE");
 if(intent)offlineReceipt.parse(result.data);else if(path!==""&&!path.startsWith("?")){const saved=offlineSaved.parse(result.data);if(path.startsWith("/commands/")?saved.operationId!==path.slice(10):saved.recordId!==path.slice(1))throw new OfflineRecordError("INVALID_UPSTREAM_RESPONSE");}
 return result.data;
 }catch(e){if(e instanceof OfflineRecordError)throw e;throw new OfflineRecordError(intent&&dispatched?"OUTCOME_UNKNOWN":"DEPENDENCY_UNAVAILABLE");}
}
const pendingKey=(s:ReportScope)=>`offline-shein-intent:${JSON.stringify([s.userId,s.organizationId])}`;
export function readOfflineIntent(s:ReportScope):OfflineIntent|null{try{const raw=sessionStorage.getItem(pendingKey(s));return raw?offlineIntent.parse(JSON.parse(raw)):null;}catch{throw new OfflineRecordError("INTENT_STORAGE_UNAVAILABLE");}}
export function writeOfflineIntent(s:ReportScope,v:OfflineIntent|null){try{if(v)sessionStorage.setItem(pendingKey(s),JSON.stringify(offlineIntent.parse(v)));else sessionStorage.removeItem(pendingKey(s));}catch{throw new OfflineRecordError("INTENT_STORAGE_UNAVAILABLE");}}
