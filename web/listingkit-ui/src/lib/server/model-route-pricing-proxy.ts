import {z} from "zod";
import {routeFact,routeInput,routeCurrent,routeReceipt,routePreview,routeHistory,routeBindings,routeCapabilities,routeCapabilityLabels} from "../contracts/model-route-pricing";
import {readBoundedStrictJSON} from "../api/strict-json-response";
import {hasTrustedSameOriginWrite} from "./same-origin-write";
import {WORKBENCH_COOKIE_NAME} from "./workbench-proxy";
import {hasEmptyBody} from "./members-proxy";
import {configuredAgentConfigurationOrigin} from "./agent-configuration-origin";
export const routeFailure=(status:number,code:string)=>Response.json({code},{status,headers:{"Cache-Control":"private, no-store","X-Content-Type-Options":"nosniff"}});
export async function proxyModelRoutePricing(request:Request,token:string,userId:string):Promise<Response>{
 if(!token||!userId)return routeFailure(401,"AUTHENTICATION_REQUIRED");
 if(request.headers.get("X-Expected-User-ID")!==userId)return routeFailure(409,"IDENTITY_CONTEXT_CHANGED");
 const url=new URL(request.url),path=url.pathname.slice("/api/model-route-pricing".length),parts=path.slice(1).split("/");
 const op=parts[2]??"current",publish=op==="publish",write=request.method==="POST";
 let output:z.ZodType;
 if(!path&&request.method==="GET")output=routeCapabilities;
 else if(parts.length>=2&&["text","image"].includes(parts[0])&&Object.hasOwn(routeCapabilityLabels,parts[1])&&(parts[0]==="image")===(parts[1]==="imageagent.generation")){
  if(parts.length===2&&request.method==="GET")output=routeCurrent;
  else if(parts.length===3&&request.method==="GET"&&["history","bindings"].includes(op))output=op==="history"?routeHistory:routeBindings;
  else if(parts.length===4&&op==="receipts"&&routeFact.safeParse(parts[3]).success&&request.method==="GET")output=routeReceipt;
  else if(parts.length===3&&write&&["validate","publish"].includes(op))output=publish?routeReceipt:routePreview;
  else return routeFailure(400,"INVALID");
 }else return routeFailure(400,"INVALID");
 if(url.search.length>512||request.url.endsWith("?")||request.headers.has("content-encoding"))return routeFailure(400,"INVALID");
 for(const [k,v] of url.searchParams){if(op!=="history"||url.searchParams.getAll(k).length!==1||(k==="cursor"?!routeFact.safeParse(v).success:k==="limit"?!/^[1-9][0-9]?$/.test(v)||Number(v)>50:true))return routeFailure(400,"INVALID")}
 if(write&&!hasTrustedSameOriginWrite(request))return routeFailure(403,"FORBIDDEN");
 const cookies=(request.headers.get("cookie")??"").split(";").map(v=>v.trim()).filter(v=>v.startsWith(WORKBENCH_COOKIE_NAME+"="));let org="";
 try{if(cookies.length===1)org=decodeURIComponent(cookies[0].slice(WORKBENCH_COOKIE_NAME.length+1))}catch{}
 if(!routeFact.safeParse(org).success||org!==request.headers.get("X-Expected-Organization-ID"))return routeFailure(409,"ORGANIZATION_CONTEXT_CHANGED");
 const controller=new AbortController(),abort=()=>controller.abort();request.signal.addEventListener("abort",abort,{once:true});if(request.signal.aborted)abort();const timer=setTimeout(abort,25000);let dispatched=false;
 try{
  const headers=new Headers({Accept:"application/json",Authorization:"Bearer "+token,"X-Requested-Organization-ID":org});let body:string|undefined;
  if(write){
   const payload=await readBoundedStrictJSON(request as unknown as Response,16*1024,controller.signal),parsed=routeInput.safeParse(payload);if(!parsed.success||("pointsPerImage" in parsed.data.rates)!==(parts[0]==="image"))return routeFailure(400,"INVALID");
   if(publish){const key=request.headers.get("Idempotency-Key");if(!routeFact.safeParse(key).success)return routeFailure(400,"INVALID");headers.set("Idempotency-Key",key!)}
   body=JSON.stringify(parsed.data);headers.set("Content-Type","application/json");
  }else if(!(await hasEmptyBody(request,controller.signal)))return routeFailure(400,"INVALID");
  const origin=configuredAgentConfigurationOrigin();if(!origin)return routeFailure(503,"UNAVAILABLE");controller.signal.throwIfAborted();dispatched=true;
  const response=await fetch(origin+"/api/v1/model-route-pricing"+path+url.search,{method:request.method,headers,body,signal:controller.signal,cache:"no-store",redirect:"manual"});
  const payload=await readBoundedStrictJSON(response,512*1024,controller.signal);controller.signal.throwIfAborted();
  if(response.status!==200){const parsed=z.strictObject({code:z.string().regex(/^[A-Z_]{1,80}$/)}).safeParse(payload);return routeFailure(response.status>=400&&response.status<600?response.status:502,publish&&response.status>=500?"OUTCOME_UNKNOWN":parsed.success?parsed.data.code:publish?"OUTCOME_UNKNOWN":"UNAVAILABLE")}
  const parsed=output.parse(payload) as Record<string,unknown>;
  if(output===routeCapabilities){if(parsed.organizationId!==org)throw new Error()}
  const revisions=output===routeHistory?routeHistory.parse(parsed).revisions:output===routeCurrent?[routeCurrent.parse(parsed).revision].filter(v=>!!v):output===routeReceipt?[routeReceipt.parse(parsed).receipt]:[];
  for(const r of revisions){if(!r||r.scope.organizationId!==org||r.scope.capability!==parts[1]||("image" in r.payload)!==(parts[0]==="image"))throw new Error()}
  if(output===routeReceipt){const r=routeReceipt.parse(parsed).receipt;if(r.key!==(publish?headers.get("Idempotency-Key"):parts[3]))throw new Error();if(publish){const in_=routeInput.parse(JSON.parse(body!));if(r.actorId!==userId||r.parent!==in_.expectedRevision||r.payload.bindingId!==in_.bindingId||r.payload.credentialVersion!==in_.credentialVersion||r.rollbackFrom!==(in_.rollbackFrom||undefined))throw new Error()}}
  if(output===routePreview){const in_=routeInput.parse(JSON.parse(body!)),p=routePreview.parse(parsed);if(p.expectedRevision!==in_.expectedRevision||p.payload.bindingId!==in_.bindingId||p.payload.credentialVersion!==in_.credentialVersion)throw new Error()}
  return Response.json(parsed,{headers:{"Cache-Control":"private, no-store","X-Content-Type-Options":"nosniff"}});
 }catch{return routeFailure(controller.signal.aborted?504:502,publish&&dispatched?"OUTCOME_UNKNOWN":dispatched?"UNAVAILABLE":"INVALID")}
 finally{clearTimeout(timer);request.signal.removeEventListener("abort",abort)}
}
