import {z} from "zod";
import {readBoundedStrictJSON} from "@/lib/api/strict-json-response";
import {parseWorkbenchContextPayload} from "@/lib/api/workbench-context";
import {aiResponseSchemas} from "@/lib/contracts/ai-workbench";
import {catalogPageSchema} from "@/lib/contracts/agent-configuration";
import {isAcquisitionUUID} from "@/lib/contracts/product-acquisition";
import {parseProductTitleProposalList} from "@/lib/api/product-title-review";
import {taskCenterSummarySchema,summarizeTaskRecords,type TaskCenterSummary} from "@/lib/contracts/task-center-summary";
import {canDiscoverNativeRuns,readNativeRunPage,type NativeRunRecord} from "@/components/workbench/ai-workbench/native-run-records";
import {buildWorkbenchBrowserResponse,buildWorkbenchUpstreamRequest} from "./workbench-proxy";
import {proxyAgentConfiguration} from "./agent-configuration-proxy";
import {proxyProductTitleReview} from "./product-title-review-proxy";
import {isAgentConfigurationAvailable} from "./agent-configuration-origin";
import {configuredProductReviewOrigin} from "./product-title-review-request";

class SummaryLimit extends Error {}
type Options={tasks:boolean;native:boolean;reviews:boolean};
export async function collectTaskSummary(request:Request,token:string,userId:string,options?:Options):Promise<Response>{
 const json=(body:unknown,status=200)=>Response.json(body,{status,headers:{"Cache-Control":"private, no-store","X-Content-Type-Options":"nosniff"}});
 const query=new URL(request.url).searchParams;
 if(request.method!=="GET"||request.body!==null||[...query.keys()].some(k=>!["scopeKind","projectId","storeId"].includes(k)||query.getAll(k).length!==1))return json({code:"INVALID_REQUEST"},400);
 const kind=query.get("scopeKind")??"all";
 if(!["all","general","project","store","unrecorded"].includes(kind)||((kind==="project")!==query.has("projectId"))||((kind==="store")!==query.has("storeId"))||["projectId","storeId"].some(k=>query.has(k)&&!isAcquisitionUUID(query.get(k)!)))return json({code:"INVALID_REQUEST"},400);
 if(!token||!userId)return json({code:"AUTHENTICATION_REQUIRED"},401);
 if(request.headers.get("X-Expected-User-ID")!==userId)return json({code:"IDENTITY_CONTEXT_CHANGED"},409);
 const org=request.headers.get("X-Expected-Organization-ID");
 if(!org)return json({code:"ORGANIZATION_CONTEXT_CHANGED"},409);
 const child=(path:string)=>new Request(new URL(path,request.url),{method:"GET",headers:request.headers,signal:request.signal});
 let calls=0,records=0;
 const budget=()=>{request.signal.throwIfAborted();if(++calls>100)throw new SummaryLimit()};
 const readJSON=(response:Response)=>{if(!response.ok)throw Error("source unavailable");return readBoundedStrictJSON(response,1024*1024,request.signal)};
 const work=async(path:string)=>{
  budget();const req=child(`/api/workbench/${path}`);
  const built=await buildWorkbenchUpstreamRequest(req,path.split("?")[0].split("/"),token,userId);
  if(built instanceof Response)return readJSON(built);
  const response=await fetch(built.url,{...built.init,redirect:"manual",signal:request.signal});
  return readJSON(await buildWorkbenchBrowserResponse(response,built.responseContract,built.expectedStoreId,{signal:request.signal}));
 };
 const context=async()=>{const parsed=parseWorkbenchContextPayload(await work("context"));if(!parsed.success||parsed.data.user.id!==userId||parsed.data.effectiveOrganizationId!==org||parsed.data.selectionRequired)throw Error("context changed");return parsed.data};
 const signature=(value:Awaited<ReturnType<typeof context>>)=>{const selected=value.organizations.find(o=>o.id===org);return JSON.stringify([value.user.id,value.effectiveOrganizationId,value.effectiveMemberId,[...(selected?.permissions??[])].sort(),[...(selected?.roles??[])].sort(),value.aiWorkbenchAvailable])};
 const config=async<T>(_:unknown,path:string,schema:z.ZodType<T>):Promise<T>=>{budget();return schema.parse(await readJSON(await proxyAgentConfiguration(child(`/api/workbench/agents/${path}`),token,userId)))};
 async function walk<T>(read:(cursor:string)=>Promise<{items:T[];next:string}>):Promise<T[]>{
  const out:T[]=[];const seen=new Set<string>();let cursor="";
  do{if(seen.has(cursor))throw new SummaryLimit();seen.add(cursor);const page=await read(cursor);records+=page.items.length;if(records>5000)throw new SummaryLimit();out.push(...page.items);cursor=page.next}while(cursor);
  return out;
 }
 const failed=(reason:"unavailable"|"limit"|"changed")=>json(taskCenterSummarySchema.parse({status:"partial",counts:null,excluded:0,capturedAt:new Date().toISOString(),reasons:[reason]}));
 try{
  const before=await context();const permissions=before.organizations.find(o=>o.id===org)?.permissions??[];
  const deployed=options??{tasks:before.aiWorkbenchAvailable===true,native:isAgentConfigurationAvailable(),reviews:configuredProductReviewOrigin()!==null};
  const enabled={tasks:deployed.tasks&&permissions.includes("workbench.task.read"),native:deployed.native&&canDiscoverNativeRuns(permissions),reviews:deployed.reviews&&permissions.includes("local_agent.write")};
  const tasks=enabled.tasks?await walk(async cursor=>{const q=new URLSearchParams(query);q.set("scopeKind",kind);q.set("limit","50");if(cursor)q.set("after",cursor);const page=aiResponseSchemas["task-list"].parse(await work(`tasks?${q}`));if(page.incomplete)throw Error("task coverage incomplete");return {items:page.tasks,next:page.next}}):[];
  const unscoped=kind==="all"||kind==="unrecorded";
  const reviews=enabled.reviews&&unscoped?await walk(async cursor=>{budget();const q=new URLSearchParams({view:"actionable",limit:"100"});if(cursor)q.set("cursor",cursor);const response=await proxyProductTitleReview(child(`/api/product/text-proposals?${q}`),token,{forwarded:false});const page=parseProductTitleProposalList(await readJSON(response));if(!page)throw Error("invalid review page");return {items:page.items,next:page.next_cursor??""}}):[];
  const runs:NativeRunRecord[]=[];
  if(enabled.native&&unscoped){
   const entries=await walk(async cursor=>{const page=await config(null,`market?pageSize=100${cursor?`&cursor=${encodeURIComponent(cursor)}`:""}`,catalogPageSchema);return {items:page.items,next:page.nextCursor}});
   const readable=Array.from(new Map(entries.filter(e=>e.canReadRuns&&e.parameterSchema!=="market-research-config-v1").map(e=>[e.agent.agentId,e])).values());
   for(let i=0;i<readable.length;i+=4){const batch=await Promise.all(readable.slice(i,i+4).map(entry=>walk(async cursor=>{const page=await readNativeRunPage({userId,organizationId:org},entry,cursor,request.signal,config);return {items:page.items,next:page.nextCursor}})));runs.push(...batch.flat())}
  }
  const summary=summarizeTaskRecords(tasks,reviews,runs);
  if(signature(before)!==signature(await context()))return failed("changed");
  const result:TaskCenterSummary={status:"complete",...summary,capturedAt:new Date().toISOString(),reasons:[]};
  return json(taskCenterSummarySchema.parse(result));
 }catch(error){return failed(error instanceof SummaryLimit?"limit":"unavailable")}
}
