import {z} from "zod";
import {readBoundedStrictJSON} from "./strict-json-response";
import {isAcquisitionUUID} from "../contracts/product-acquisition";
import {analysisMethods,analysisMatches,analysisRequestSchema,type AnalysisRoute,type AnalysisBinding} from "../contracts/product-image-analysis";

export type AnalysisScope={userId:string;organizationId:string;kind:"acquisition"|"supply"|"own_product";contextId:string};
export class AnalysisError extends Error{constructor(public code:string,public status=500){super(code)}}
export async function analysisRequest<T>(scope:AnalysisScope,action:AnalysisRoute,schema:z.ZodType<T>,options:{requestKey?:string;body?:unknown;cursor?:string;signal?:AbortSignal}={}):Promise<T>{
 if(!scope.userId||!scope.organizationId||!isAcquisitionUUID(scope.contextId)||!["acquisition","supply","own_product"].includes(scope.kind))throw new AnalysisError("INVALID_REQUEST",400);
 const binding:AnalysisBinding={contextKind:scope.kind==="supply"?"collection":scope.kind,contextId:scope.contextId,...["prepare","read","confirm"].includes(action)?{requestKey:options.requestKey}:{}};
 const method=analysisMethods[action],mutation=method==="POST";
 if(["prepare","read","confirm"].includes(action)&&!isAcquisitionUUID(options.requestKey??""))throw new AnalysisError("INVALID_REQUEST",400);
 const body=mutation?analysisRequestSchema(action).safeParse(options.body):undefined;
 if(mutation&&!body?.success||options.cursor&&(action!=="recent"||!isAcquisitionUUID(options.cursor)))throw new AnalysisError("INVALID_REQUEST",400);
 let path=scope.kind==="acquisition"?`/api/workbench/sourcing/1688/acquisitions/${scope.contextId}`:scope.kind==="supply"?`/api/workbench/supply-preparations/sources/${scope.contextId}`:`/api/workbench/collections/items/${scope.contextId}`;
 path+="/images/analysis"+(action==="sources"?"":action==="prepare"?"/prepare":action==="recent"?"/runs":`/by-key/${options.requestKey}`+(action==="confirm"?"/confirm":""));
 if(options.cursor)path+="?cursor="+options.cursor;
 const controller=new AbortController(),abort=()=>controller.abort();if(options.signal?.aborted)abort();options.signal?.addEventListener("abort",abort,{once:true});const deadline=setTimeout(abort,130000);
 try{
  const headers=new Headers({Accept:"application/json","X-Expected-User-ID":scope.userId,"X-Expected-Organization-ID":scope.organizationId});
  if(mutation)headers.set("Content-Type","application/json");if(action==="prepare")headers.set("Idempotency-Key",options.requestKey!);
  const response=await fetch(path,{method,headers,body:body?.success?JSON.stringify(body.data):undefined,credentials:"same-origin",cache:"no-store",redirect:"error",signal:controller.signal});
  const payload=await readBoundedStrictJSON(response,128*1024,controller.signal);
  if(!response.ok){const error=z.object({code:z.string()}).safeParse(payload);throw new AnalysisError(error.success?error.data.code:mutation?"OUTCOME_UNKNOWN":"DEPENDENCY_UNAVAILABLE",response.status)}
  const parsed=schema.safeParse(payload);if(response.status!==200||!parsed.success||!analysisMatches(parsed.data,binding))throw new AnalysisError(mutation?"OUTCOME_UNKNOWN":"DEPENDENCY_UNAVAILABLE",mutation?503:502);
  controller.signal.throwIfAborted();return parsed.data;
 }catch(error){if(error instanceof AnalysisError)throw error;throw new AnalysisError(mutation?"OUTCOME_UNKNOWN":"DEPENDENCY_UNAVAILABLE",mutation?503:502)}
 finally{clearTimeout(deadline);options.signal?.removeEventListener("abort",abort)}
}
