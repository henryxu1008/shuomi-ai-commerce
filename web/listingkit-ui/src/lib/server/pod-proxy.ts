import {diagnosisEndpoint} from "../contracts/shop-diagnosis";
import {selectionEndpoint} from "../contracts/product-selection";
import {artworkEndpoint} from "../contracts/pod-artwork";
import {podEndpoint,podID} from "../contracts/pod";
import {readBoundedStrictJSON} from "../api/strict-json-response";
import {hasTrustedSameOriginWrite} from "./same-origin-write";
import {hasEmptyBody} from "./members-proxy";
import {WORKBENCH_COOKIE_NAME} from "./workbench-proxy";
const headers={"Cache-Control":"private, no-store","X-Content-Type-Options":"nosniff"};
export const podFailure=(status:number,code:string)=>Response.json({code},{status,headers});
export async function proxyPOD(request:Request,token:string,userId:string){
 const diagnosis=new URL(request.url).pathname.startsWith("/api/workbench/shop-diagnosis/");
 const selection=new URL(request.url).pathname.startsWith("/api/workbench/product-selection/");
 const artwork=new URL(request.url).pathname.startsWith("/api/workbench/pod-artwork/");
 if(!diagnosis&&!selection&&!artwork&&process.env.LISTINGKIT_SDS_POD_ENABLED!=="true")return podFailure(503,"DEPENDENCY_UNAVAILABLE");
 if(!token||!userId)return podFailure(401,"AUTHENTICATION_REQUIRED");if(request.headers.get("X-Expected-User-ID")!==userId)return podFailure(409,"IDENTITY_CONTEXT_CHANGED");
 let org="";const cookies=(request.headers.get("cookie")??"").split(";").map(s=>s.trim()).filter(s=>s.startsWith(WORKBENCH_COOKIE_NAME+"="));try{if(cookies.length===1)org=decodeURIComponent(cookies[0]!.slice(WORKBENCH_COOKIE_NAME.length+1))}catch{}
 if(!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(org)||org!==request.headers.get("X-Expected-Organization-ID"))return podFailure(409,"ORGANIZATION_CONTEXT_CHANGED");
 const url=new URL(request.url),route=diagnosis?diagnosisEndpoint(url,request.method):selection?selectionEndpoint(url,request.method):artwork?artworkEndpoint(url,request.method):podEndpoint(url,request.method),write=request.method!=="GET";if(!route||request.url.endsWith("?")||request.headers.has("content-encoding")||request.headers.has("If-Match"))return podFailure(400,"INVALID_REQUEST");if(write&&!hasTrustedSameOriginWrite(request))return podFailure(403,"PERMISSION_DENIED");
 let origin:string;try{const base=new URL(process.env.LISTINGKIT_SERVICE_API_BASE??"");if(!["https:","http:"].includes(base.protocol)||base.username||base.password||base.search||base.hash||!["/api/v1","/api/v1/"].includes(base.pathname))throw new Error();origin=base.origin}catch{return podFailure(503,"DEPENDENCY_UNAVAILABLE")}
 const upstream=new Headers({Authorization:"Bearer "+token,Accept:"application/json","X-Requested-Organization-ID":org}),controller=new AbortController(),abort=()=>controller.abort();request.signal.addEventListener("abort",abort,{once:true});if(request.signal.aborted)abort();const timer=setTimeout(abort,(selection||diagnosis)?125000:33000);let sent=false;
 try{let body:string|undefined;if(write){const key=request.headers.get("Idempotency-Key");if(!podID.safeParse(key).success)return podFailure(400,"INVALID_REQUEST");const raw=await readBoundedStrictJSON(new Response(request.body,{headers:{"Content-Type":request.headers.get("Content-Type")??""}}),65536,controller.signal),input=route.input?.safeParse(raw);if(!input?.success||("approval" in route&&route.approval)&&(input.data as {actionId:string}).actionId!==key)return podFailure(400,"INVALID_REQUEST");if((selection||diagnosis)&&route.path==="prepare"&&(input.data as {requestKey:string}).requestKey!==key)return podFailure(400,"INVALID_REQUEST");if((selection||diagnosis)&&route.path.endsWith("/resume")&&route.path.split("/")[3]!==key)return podFailure(400,"INVALID_REQUEST");body=JSON.stringify(input.data);upstream.set("Idempotency-Key",key!);upstream.set("Content-Type","application/json")}else if(!(await hasEmptyBody(request,controller.signal)))return podFailure(400,"INVALID_REQUEST");
  controller.signal.throwIfAborted();sent=true;const response=await fetch(origin+(diagnosis?"/api/v1/workbench/shop-diagnosis/":selection?"/api/v1/workbench/product-selection/":artwork?"/api/v1/workbench/pod-artwork/":"/api/v1/workbench/pod/")+route.path+url.search,{method:request.method,headers:upstream,body,cache:"no-store",redirect:"manual",signal:controller.signal});const raw=await readBoundedStrictJSON(response,2*1024*1024,controller.signal);controller.signal.throwIfAborted();
  if(!response.ok){const code=(raw as {code?:unknown})?.code;if(write&&response.status>=500)return podFailure(response.status,"OUTCOME_UNKNOWN");return podFailure(response.status>=400&&response.status<600?response.status:502,typeof code==="string"&&/^[A-Z_]{1,80}$/.test(code)?code:"DEPENDENCY_UNAVAILABLE")}
  const parsed=route.output.safeParse(raw);if(response.status!==200||!parsed.success)throw new Error();const data=parsed.data as {id?:string;actionId?:string;template?:{id:string};item?:{itemId:string};runId?:string;requestKey?:string;contextId?:string;manifest?:{variantId:string}},p=route.path.split("/");
  if(selection){
   if(p[0]==="contexts"&&(data.contextId!==p[1]||data.requestKey!==(p[3]??request.headers.get("Idempotency-Key"))))throw new Error();
   if(p[0]==="prepare"&&data.requestKey!==request.headers.get("Idempotency-Key"))throw new Error();
  }
  if(artwork){
   if(p[0]==="runs"&&data.runId!==p[1]||p[0]==="by-key"&&(data.contextId!==p[1]||data.requestKey!==p[2])||p[0]==="prepare"&&data.requestKey!==request.headers.get("Idempotency-Key"))throw new Error();
  }
  if(("approval" in route&&route.approval)&&data.actionId!==request.headers.get("Idempotency-Key"))throw new Error();
  if(p[0]==="designs"&&request.method==="GET"&&data.id!==p[1]||p[0]==="templates"&&p.length===2&&request.method==="GET"&&data.template?.id!==p[1]||p[0]==="artwork"&&p.length===2&&data.item?.itemId!==p[1]||p[0]==="artwork"&&p.length===4&&data.actionId!==p[3]||p[0]==="manifests"&&data.manifest?.variantId!==url.searchParams.get("variant"))throw new Error();return Response.json(parsed.data,{headers});
 }catch{return podFailure(sent?502:400,write&&sent?"OUTCOME_UNKNOWN":sent?"DEPENDENCY_UNAVAILABLE":"INVALID_REQUEST")}finally{clearTimeout(timer);request.signal.removeEventListener("abort",abort)}
}
