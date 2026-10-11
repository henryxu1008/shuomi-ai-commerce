import {z} from "zod";
import {catalogItemSchema,catalogSchema,catalogFiltersSchema,groupsSchema,profileSchema,metadataCommandSchema,metadataResultSchema,storeCenterEnvelope,storeCenterID,type MetadataCommand} from "@/lib/api/store-center";
import {readBoundedStrictJSON} from "@/lib/api/strict-json-response";
import {serviceOrigin} from "./account-proxy";
import {hasTrustedSameOriginWrite} from "./same-origin-write";
import {WORKBENCH_COOKIE_NAME} from "./workbench-proxy";
const safeID=(value:string|null):value is string=>typeof value==="string" && /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(value);
const fail=(status:number,code:string)=>Response.json({code},{status,headers:{"Cache-Control":"private, no-store","X-Content-Type-Options":"nosniff"}});
type Endpoint={path:string;schema:z.ZodType;write:boolean;key?:string;store?:string};
function endpoint(url:URL,method:string):Endpoint|null {
 const prefix="/api/workbench/store-center/";if(!url.pathname.startsWith(prefix))return null;
 const path=url.pathname.slice(prefix.length);if(url.pathname.length>2048 || url.search.length>4096)return null;
 try{decodeURIComponent(url.search)}catch{return null}
 if(path==="catalog" && method==="GET"){
  const raw:Record<string,unknown>={page:1,pageSize:20};
  for(const key of url.searchParams.keys()){
   if(url.searchParams.getAll(key).length!==1)return null;
   const value=url.searchParams.get(key)!;
   if(key==="page" || key==="pageSize"){if(!/^[1-9][0-9]*$/.test(value))return null;raw[key]=Number(value)}else raw[key]=value;
  }
  if(!catalogFiltersSchema.safeParse(raw).success || (Number(raw.page)-1)*Number(raw.pageSize)>Number.MAX_SAFE_INTEGER)return null;
  return {path:"store-catalog",schema:catalogSchema,write:false};
 }
 if(url.search || url.href.endsWith("?"))return null;
 if(path==="groups" && method==="GET")return {path:"store-groups",schema:groupsSchema,write:false};
 if(path==="metadata" && method==="POST")return {path:"store-metadata",schema:metadataResultSchema,write:true};
 const parts=path.split("/");
 if(parts.length===2 && storeCenterID.safeParse(parts[1]).success && method==="GET"){
  if(parts[0]==="metadata")return {path:`store-metadata/${parts[1]}`,schema:metadataResultSchema,write:false,key:parts[1]};
  if(parts[0]==="summaries")return {path:`stores/${parts[1]}/summary`,schema:catalogItemSchema,write:false,store:parts[1]};
  if(parts[0]==="profiles")return {path:`stores/${parts[1]}/profile`,schema:profileSchema,write:false,store:parts[1]};
 }
 return null;
}
export async function proxyStoreCenter(request:Request,token:string,user:string):Promise<Response>{
 if(!token || !safeID(user))return fail(401,"AUTHENTICATION_REQUIRED");
 if(request.headers.get("X-Expected-User-ID")!==user)return fail(409,"IDENTITY_CONTEXT_CHANGED");
 const selections=(request.headers.get("cookie")??"").split(";").map(value=>value.trim()).filter(value=>value.startsWith(`${WORKBENCH_COOKIE_NAME}=`));
 let org:string|null=null;try{if(selections.length===1)org=decodeURIComponent(selections[0].slice(WORKBENCH_COOKIE_NAME.length+1))}catch{}
 if(!safeID(org) || org!==request.headers.get("X-Expected-Organization-ID"))return fail(409,"ORGANIZATION_CONTEXT_CHANGED");
 const url=new URL(request.url),selected=endpoint(url,request.method);if(!selected)return fail(400,"INVALID_REQUEST");
 if(selected.write && !hasTrustedSameOriginWrite(request))return fail(403,"PERMISSION_DENIED");
 if(!selected.write && (request.body!==null || request.headers.has("transfer-encoding") || request.headers.has("Idempotency-Key") || request.headers.has("content-length") && request.headers.get("content-length")!=="0")){void request.body?.cancel().catch(()=>undefined);return fail(400,"INVALID_REQUEST")}
 const origin=serviceOrigin();if(!origin)return fail(503,"DEPENDENCY_UNAVAILABLE");
 const controller=new AbortController();const abort=()=>controller.abort();request.signal.addEventListener("abort",abort,{once:true});if(request.signal.aborted)abort();const timer=setTimeout(abort,15000);
 let dispatched=false;
 try{
  const upstream=new URL(`/api/v1/workbench/${selected.path}`,origin);upstream.search=url.search;
  const headers=new Headers({Authorization:`Bearer ${token}`,Accept:"application/json","X-Requested-Organization-ID":org});
  let body:string|undefined,key:string|undefined,command:MetadataCommand|undefined;
  if(selected.write){
   key=request.headers.get("Idempotency-Key")??undefined;
   if(!storeCenterID.safeParse(key).success || !/^application\/json(?:\s*;|$)/i.test(request.headers.get("content-type")??""))return fail(400,"INVALID_REQUEST");
   let raw:unknown;try{raw=await readBoundedStrictJSON(new Response(request.body,{headers:{"Content-Type":"application/json"}}),16384,controller.signal)}catch{return fail(400,"INVALID_REQUEST")}
   const parsed=metadataCommandSchema.safeParse(raw);if(!parsed.success)return fail(400,"INVALID_REQUEST");command=parsed.data;body=JSON.stringify(command);headers.set("Content-Type","application/json");headers.set("Idempotency-Key",key!);
  }
  controller.signal.throwIfAborted();dispatched=true;
  const response=await fetch(upstream,{method:request.method,body,headers,cache:"no-store",redirect:"manual",signal:controller.signal});
  const raw=await readBoundedStrictJSON(response,2<<20,controller.signal);controller.signal.throwIfAborted();
  if(response.status!==200){
   const error=z.object({code:z.enum(["INVALID_REQUEST","AUTHENTICATION_REQUIRED","PERMISSION_DENIED","NOT_FOUND","STORE_NOT_FOUND","VERSION_CONFLICT","STORE_VERSION_CONFLICT","IDEMPOTENCY_KEY_CONFLICT","DEPENDENCY_UNAVAILABLE","CATALOG_SUMMARY_UNAVAILABLE","ORGANIZATION_ACCESS_DENIED","ORGANIZATION_ACCESS_REVOKED","ORGANIZATION_SUSPENDED","ORGANIZATION_CONTEXT_CHANGED"])}).passthrough().safeParse(raw);
   if(!error.success || ![400,401,403,404,409,412,503].includes(response.status))return fail(502,selected.write?"OUTCOME_UNKNOWN":"INVALID_UPSTREAM_RESPONSE");
   return fail(response.status,error.data.code);
  }
  const envelope=storeCenterEnvelope(selected.schema).safeParse(raw);if(!envelope.success)return fail(502,selected.write?"OUTCOME_UNKNOWN":"INVALID_UPSTREAM_RESPONSE");
  if(envelope.data.organizationId!==org || envelope.data.userId!==user)return fail(409,"ORGANIZATION_CONTEXT_CHANGED");
  if(selected.schema===metadataResultSchema){
   const result=metadataResultSchema.parse(envelope.data.data);
   if(result.operationId!==(key??selected.key))return fail(502,"INVALID_UPSTREAM_RESPONSE");
   if(command){const ids=command.stores.map(item=>item.storeId).sort();if(JSON.stringify(ids)!==JSON.stringify(result.profiles.map(item=>item.storeId).sort()) || result.profiles.some(profile=>profile.groupId!==command!.groupId) || command.action==="set_profile" && result.profiles.some(profile=>profile.responsibleMemberId!==command!.responsibleMemberId || profile.lowStockThreshold!==command!.lowStockThreshold))return fail(502,"OUTCOME_UNKNOWN")}
  }
  if(selected.store && (selected.schema===catalogItemSchema?catalogItemSchema.parse(envelope.data.data).id:profileSchema.parse(envelope.data.data).storeId)!==selected.store)return fail(502,"INVALID_UPSTREAM_RESPONSE");
  return Response.json(envelope.data,{headers:{"Cache-Control":"private, no-store","X-Content-Type-Options":"nosniff"}});
 }catch{return fail(controller.signal.aborted?504:502,selected.write && dispatched?"OUTCOME_UNKNOWN":controller.signal.aborted?"DEADLINE_EXCEEDED":"DEPENDENCY_UNAVAILABLE")}
 finally{clearTimeout(timer);request.signal.removeEventListener("abort",abort)}
}
