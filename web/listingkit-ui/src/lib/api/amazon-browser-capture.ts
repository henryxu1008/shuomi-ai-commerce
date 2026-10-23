import { AMAZON_CAPTURE_BASE, AMAZON_CAPTURE_RESPONSE_MAX_BYTES, amazonCaptureResultSchema, amazonCaptureErrors, parseAmazonJSON, type AmazonCaptureResult } from "@/lib/contracts/amazon-browser-capture";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
import type { AmazonScope } from "@/app/capture/amazon/capture-marker";

export class AmazonCaptureError extends Error{constructor(public code:string){super(code);}}
export async function amazonCaptureRequest(scope:AmazonScope,key:string,body:string|undefined,signal:AbortSignal):Promise<AmazonCaptureResult>{
 if(!isAcquisitionUUID(key))throw new AmazonCaptureError("INVALID_CAPTURE");
 const headers=new Headers({Accept:"application/json","X-Expected-User-ID":scope.userId,"X-Expected-Organization-ID":scope.organizationId,"X-Expected-Member-ID":scope.memberId});
 if(body!==undefined){headers.set("Content-Type","application/json");headers.set("Idempotency-Key",key);}
 const combined=AbortSignal.any([signal,AbortSignal.timeout(25000)]);
 try{
  const response=await fetch(`${AMAZON_CAPTURE_BASE}${body===undefined?`/by-key/${key}`:""}`,{method:body===undefined?"GET":"POST",headers,body,credentials:"same-origin",cache:"no-store",redirect:"error",signal:combined});
  if(!/^application\/json(?:;|$)/i.test(response.headers.get("content-type")??""))throw new Error();
  const reader=response.body?.getReader();if(!reader)throw new Error();const chunks:Uint8Array[]=[];let count=0;
  const abort=()=>void reader.cancel().catch(()=>undefined);combined.addEventListener("abort",abort,{once:true});
  try{while(true){combined.throwIfAborted();const{done,value}=await reader.read();combined.throwIfAborted();if(done)break;count+=value.byteLength;if(count>AMAZON_CAPTURE_RESPONSE_MAX_BYTES){abort();throw new Error();}chunks.push(value);}}finally{reader.releaseLock();combined.removeEventListener("abort",abort);}
  const bytes=new Uint8Array(count);let offset=0;for(const chunk of chunks){bytes.set(chunk,offset);offset+=chunk.length;}
  const value=parseAmazonJSON(new TextDecoder("utf-8",{fatal:true}).decode(bytes),AMAZON_CAPTURE_RESPONSE_MAX_BYTES);
  if(response.status!==200){const code=value&&typeof value==="object"&&"code"in value?value.code:undefined;if(typeof code==="string"&&amazonCaptureErrors[code]===response.status)throw new AmazonCaptureError(code);throw new Error();}
  const result=amazonCaptureResultSchema.parse(value);if(result.key!==key)throw new Error();return result;
 }catch(error){if(error instanceof AmazonCaptureError)throw error;throw new AmazonCaptureError("OUTCOME_UNKNOWN");}
}
