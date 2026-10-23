import { AMAZON_CAPTURE_BASE, AMAZON_CAPTURE_MAX_BYTES, AMAZON_CAPTURE_RESPONSE_MAX_BYTES, amazonCaptureSchema, amazonCaptureResultSchema, amazonCaptureErrors, parseAmazonJSON } from "@/lib/contracts/amazon-browser-capture";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
import { hasTrustedSameOriginRecovery, hasTrustedSameOriginWrite } from "./same-origin-write";
import { readBodyWithinLimit } from "./workbench-proxy";

const failure=(status:number,code:string)=>Response.json({code,message:code},{status,headers:{"Cache-Control":"no-store"}});
export async function buildAmazonCaptureRequest(request:Request,token:string,actor:string):Promise<{url:string;init:RequestInit;expectedID?:string;key?:string}|Response>{
 const url=new URL(request.url),suffix=url.pathname.slice(AMAZON_CAPTURE_BASE.length),post=request.method==="POST";
 const valid=url.pathname.startsWith(AMAZON_CAPTURE_BASE) && (post?(suffix===""||suffix==="/verify"):request.method==="GET"&&(/^\/by-key\/[0-9a-f-]{36}$/.test(suffix)||/^\/[0-9a-f-]{36}(?:\/published)?$/.test(suffix)));
 if(!valid || url.search || request.url.split("#")[0]?.endsWith("?"))return failure(404,"INVALID_CAPTURE");
 if(!token || !actor)return failure(401,"AUTHENTICATION_REQUIRED");
 if(request.headers.get("X-Expected-User-ID")!==actor)return failure(409,"IDENTITY_CONTEXT_CHANGED");
 const member=request.headers.get("X-Expected-Member-ID")??"";
 if(!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(member))return failure(409,"IDENTITY_CONTEXT_CHANGED");
 if(!(post?hasTrustedSameOriginWrite(request):hasTrustedSameOriginRecovery(request)))return failure(403,"CAPTURE_FORBIDDEN");
 const cookies=(request.headers.get("cookie")??"").split(";").map(v=>v.trim()).filter(v=>v.startsWith("shuomi_effective_organization="));
 const org=cookies[0]?.slice("shuomi_effective_organization=".length);
 if(cookies.length!==1 || !org || !/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(org) || org!==request.headers.get("X-Expected-Organization-ID"))return failure(409,"ORGANIZATION_CONTEXT_CHANGED");
 const headers=new Headers({Authorization:`Bearer ${token}`,Accept:"application/json","X-Requested-Organization-ID":org,"X-Expected-Member-ID":member});
 let body:string|undefined;let key:string|undefined;
 try{
  const bytes=await readBodyWithinLimit(request.body,AMAZON_CAPTURE_MAX_BYTES,5000,request.signal);
  if(post){
   key=request.headers.get("Idempotency-Key")??"";
   if(!isAcquisitionUUID(key)||!/^application\/json(?:;\s*charset=utf-8)?$/i.test(request.headers.get("content-type")??"")||request.headers.has("content-encoding"))return failure(400,"INVALID_CAPTURE");
   body=new TextDecoder("utf-8",{fatal:true}).decode(bytes);amazonCaptureSchema.parse(parseAmazonJSON(body));headers.set("Idempotency-Key",key);headers.set("Content-Type","application/json");
  }else if(bytes.length || request.headers.has("Idempotency-Key"))return failure(400,"INVALID_CAPTURE");
 }catch{return failure(400,"INVALID_CAPTURE");}
 const parts=suffix.split("/").filter(Boolean);const expectedID=parts[0]!=="by-key"?parts[0]:undefined;
 if(parts.length && parts[0]!=="verify" && !isAcquisitionUUID(parts[0]==="by-key"?parts[1]!:parts[0]!))return failure(400,"INVALID_CAPTURE");
 const root=(process.env.LISTINGKIT_SERVICE_API_BASE?.trim()||"http://localhost:8085/api/v1").replace(/\/+$/,"");
 return {url:`${root}/workbench/sourcing/amazon/browser-captures${suffix}`,init:{method:request.method,headers,body,cache:"no-store",redirect:"error"},expectedID:expectedID==="verify"?undefined:expectedID,key:key??(parts[0]==="by-key"?parts[1]:undefined)};
}
export async function proxyAmazonCapture(request:Request,token:string,actor:string){
 const built=await buildAmazonCaptureRequest(request,token,actor);if(built instanceof Response)return built;
 const timeout=AbortSignal.timeout(22000);const signal=AbortSignal.any([request.signal,timeout]);
 try{
  const upstream=await fetch(built.url,{...built.init,signal});
  if(!/^application\/json(?:;|$)/i.test(upstream.headers.get("content-type")??""))throw new Error();
  const bytes=await readBodyWithinLimit(upstream.body,AMAZON_CAPTURE_RESPONSE_MAX_BYTES,22000,signal);
  const value=parseAmazonJSON(new TextDecoder("utf-8",{fatal:true}).decode(bytes),AMAZON_CAPTURE_RESPONSE_MAX_BYTES);
  if(upstream.status!==200){const code=value&&typeof value==="object"&&"code"in value?value.code:undefined;if(typeof code==="string"&&amazonCaptureErrors[code]===upstream.status)return failure(upstream.status,code);throw new Error();}
  const parsed=amazonCaptureResultSchema.parse(value);
  if(built.expectedID && parsed.captureID!==built.expectedID || built.key && parsed.key!==built.key)throw new Error();
  return Response.json(parsed,{headers:{"Cache-Control":"no-store"}});
 }catch{return failure(503,"OUTCOME_UNKNOWN");}
}
