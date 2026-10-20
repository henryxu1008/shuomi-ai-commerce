import {z} from "zod";
import {readBoundedStrictJSON} from "./strict-json-response";
import {SupplyAPIError,type SupplyScope} from "./supply-chain";
import {collectionID} from "../contracts/product-collection";
import {materialsRunSchema,materialsAdoptionSchema} from "../contracts/supply-chain";
import {materialsPrepareSchema,materialsPreparedSchema,materialsStartSchema,materialsResumeSchema,materialsAdoptSchema} from "../contracts/listing-materials";

export type MaterialsPrepared=z.infer<typeof materialsPreparedSchema>;
export type MaterialsPrepare=z.infer<typeof materialsPrepareSchema>;
export type MaterialsAdopt=z.infer<typeof materialsAdoptSchema>;
async function send<T extends z.ZodType>(scope:SupplyScope,recordId:string,suffix:string,schema:T,signal:AbortSignal,input?:unknown,key?:string):Promise<z.output<T>> {
 const base=`/api/workbench/supply-preparations/records/${collectionID.parse(recordId)}/materials-agent`;
 const controller=new AbortController(),abort=()=>controller.abort();
 signal.addEventListener("abort",abort,{once:true});if(signal.aborted)abort();
 const timer=setTimeout(abort,130000);
 try {
  const headers=new Headers({"X-Expected-User-ID":scope.userId,"X-Expected-Organization-ID":scope.organizationId,Accept:"application/json"});
  if(input!==undefined)headers.set("Content-Type","application/json");if(key)headers.set("Idempotency-Key",collectionID.parse(key));
  const response=await fetch(base+suffix,{method:input===undefined?"GET":"POST",headers,body:input===undefined?undefined:JSON.stringify(input),credentials:"same-origin",redirect:"error",cache:"no-store",signal:controller.signal});
  const raw=await readBoundedStrictJSON(response,2*1024*1024,controller.signal);
  if(!response.ok){const code=raw&&typeof raw==="object"&&"code" in raw&&typeof raw.code==="string"?raw.code:"OUTCOME_UNKNOWN";throw new SupplyAPIError(code,response.status)}
  const value=schema.safeParse(raw);
  if(response.status!==200||!value.success)throw new SupplyAPIError(input===undefined?"DEPENDENCY_UNAVAILABLE":"OUTCOME_UNKNOWN",502);
  return value.data;
 }catch(error){if(error instanceof SupplyAPIError)throw error;throw new SupplyAPIError(input===undefined?"DEPENDENCY_UNAVAILABLE":"OUTCOME_UNKNOWN",503)}
 finally{clearTimeout(timer);signal.removeEventListener("abort",abort)}
}
function exactRun<T extends {recordId:string;requestKey:string}>(result:T,recordId:string,key:string):T {
 if(result.recordId!==recordId||result.requestKey!==key)throw new SupplyAPIError("OUTCOME_UNKNOWN",502);return result;
}
export async function prepareMaterials(scope:SupplyScope,recordId:string,input:MaterialsPrepare,signal:AbortSignal){return exactRun(await send(scope,recordId,"/prepare",materialsPreparedSchema,signal,materialsPrepareSchema.parse(input)),recordId,input.requestKey)}
export async function startMaterials(scope:SupplyScope,recordId:string,key:string,prepared:MaterialsPrepared,signal:AbortSignal){
 if(prepared.recordId!==recordId||prepared.requestKey!==key)throw new SupplyAPIError("INVALID_REQUEST",400);
 const result=exactRun(await send(scope,recordId,"/runs",materialsRunSchema,signal,materialsStartSchema.parse({configurationSnapshotRef:prepared.configurationSnapshotRef}),key),recordId,key);
 if(JSON.stringify(result.configurationSnapshotRef)!==JSON.stringify(prepared.configurationSnapshotRef)||JSON.stringify(result.modeSnapshotRef)!==JSON.stringify(prepared.modeSnapshotRef))throw new SupplyAPIError("OUTCOME_UNKNOWN",502);
 return result;
}
export async function readMaterials(scope:SupplyScope,recordId:string,key:string,signal:AbortSignal){return exactRun(await send(scope,recordId,`/runs/${collectionID.parse(key)}`,materialsRunSchema,signal),recordId,key)}
export async function resumeMaterials(scope:SupplyScope,recordId:string,key:string,input:z.infer<typeof materialsResumeSchema>,signal:AbortSignal){return exactRun(await send(scope,recordId,`/runs/${collectionID.parse(key)}/resume`,materialsRunSchema,signal,materialsResumeSchema.parse(input)),recordId,key)}
export const adoptMaterials=(scope:SupplyScope,recordId:string,key:string,input:MaterialsAdopt,signal:AbortSignal)=>send(scope,recordId,`/runs/${collectionID.parse(key)}/adopt`,materialsAdoptionSchema,signal,materialsAdoptSchema.parse(input));
export const readMaterialsAdoption=(scope:SupplyScope,recordId:string,key:string,signal:AbortSignal)=>send(scope,recordId,`/runs/${collectionID.parse(key)}/adoption`,materialsAdoptionSchema,signal);
