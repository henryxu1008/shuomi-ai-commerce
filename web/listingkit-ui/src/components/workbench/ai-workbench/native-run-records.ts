import {z} from "zod";
import {configurationRequest,type ConfigurationScope} from "@/lib/api/agent-configuration";
import {recentRunsSchema,type CatalogEntry} from "@/lib/contracts/agent-configuration";
import {imageSetRecentSchema} from "@/lib/contracts/product-image-set";
import {artworkRecentSchema} from "@/lib/contracts/pod-artwork";
import {selectionRecentSchema} from "@/lib/contracts/product-selection";
import {diagnosisRecentSchema} from "@/lib/contracts/shop-diagnosis";

export function canDiscoverNativeRuns(permissions:readonly string[]){
 return permissions.includes("workbench.agent.use") && (permissions.includes("workbench.agent.read")||permissions.includes("workbench.agent.configure"));
}
export type NativeRunRecord={
 key:string;runId:string;agentId:string;title:string;caption:string;summary:string;label:string;state?:string;
 contextKind:string;contextId:string;productKey?:string;targetPlatform:string;at:string;href:string;direct:boolean;
};
export type NativeRunPage={items:NativeRunRecord[];nextCursor:string};
const contexts:Record<string,string>={own_product:"自有商品",acquisition:"源商品",collection:"商品集合",supply:"供应链商品","pod-artwork-v1":"POD 图案",selection:"选品范围",diagnosis:"经营诊断范围","listing-target-record":"上架资料"};
const platforms:Record<string,string>={product:"通用商品",business:"业务范围",sds:"SDS"};
export function nativeRunDestination(scope:ConfigurationScope,agentId:string,contextKind:string,contextId:string,requestKey?:string,targetPlatform?:string){
 const fallback={href:`/workbench/agents/mine/${encodeURIComponent(agentId)}`,direct:false};
 if(!requestKey)return fallback;
 const query=new URLSearchParams({agent_key:requestKey,agent_actor:scope.userId,agent_org:scope.organizationId});
 if(agentId==="product.content.agent"&&contextKind==="own_product")return {href:`/workbench/data/mine/own/${contextId}/optimize?${query}`,direct:true};
 if(agentId==="listing.materials.agent"&&contextKind==="listing-target-record")return {href:`/workbench/supply/materials/${contextId}?${query}`,direct:true};
 if(["product.content.agent","product.title.agent"].includes(agentId)&&contextKind==="acquisition"){
  query.set("agent_platform",targetPlatform??"product");if(agentId==="product.content.agent")query.set("agent_kind","content");
  return {href:`/workbench/supply/acquisition/operation/${contextId}?${query}`,direct:true};
 }
 return fallback;
}
type TextRun=z.infer<typeof recentRunsSchema>["items"][number]|z.infer<typeof selectionRecentSchema>["items"][number]|z.infer<typeof diagnosisRecentSchema>["items"][number];
function textRecord(scope:ConfigurationScope,entry:CatalogEntry,run:TextRun):NativeRunRecord{
 const id=entry.agent.agentId;
 const kind="contextKind" in run?run.contextKind:id==="product.selection.agent"?"selection":"diagnosis";
 const label={running:"执行中",interrupted:"已暂停",human_review_required:"建议已生成",stopped:"运行已停止"}[run.phase];
 const state=({running:"RUNNING",interrupted:"PAUSED",stopped:"ERROR"} as Record<string,string>)[run.phase];
 return {key:`${id}:${run.runId}`,runId:run.runId,agentId:id,title:entry.name,caption:run.productKey||contexts[kind]||"原业务上下文",summary:label,label,state,contextKind:kind,contextId:run.operationId,productKey:run.productKey,targetPlatform:run.targetPlatform,at:run.startedAt,...nativeRunDestination(scope,id,kind,run.operationId,run.requestKey,run.targetPlatform)};
}
type ImageRun=z.infer<typeof imageSetRecentSchema>["items"][number]|z.infer<typeof artworkRecentSchema>["items"][number];
const imagePhases:Record<string,{label:string;state?:string}>={
 planning:{label:"计划准备中",state:"RUNNING"},awaiting_plan_approval:{label:"待确认计划",state:"WAITING_CONFIRMATION"},
 executing:{label:"生成中",state:"RUNNING"},evaluating:{label:"处理结果中",state:"RUNNING"},repairing:{label:"核实原调用中",state:"RUNNING"},
 awaiting_final_approval:{label:"待审核结果",state:"WAITING_CONFIRMATION"},completed:{label:"图片运行已完成",state:"COMPLETED"},
 blocked:{label:"运行受阻",state:"ERROR"},failed:{label:"运行失败",state:"ERROR"},cancelled:{label:"已取消"},
};
function imageRecord(scope:ConfigurationScope,entry:CatalogEntry,run:ImageRun):NativeRunRecord{
 const phase=imagePhases[run.status]??{label:"状态未知"};
 return {key:`${entry.agent.agentId}:${run.runId}`,runId:run.runId,agentId:entry.agent.agentId,title:entry.name,caption:contexts[run.contextKind]??"原业务上下文",summary:phase.label,...phase,contextKind:run.contextKind,contextId:run.contextId,targetPlatform:run.targetPlatform,at:run.createdAt,...nativeRunDestination(scope,entry.agent.agentId,run.contextKind,run.contextId)};
}
export async function readNativeRunPage(scope:ConfigurationScope,entry:CatalogEntry,cursor:string,signal:AbortSignal):Promise<NativeRunPage>{
 const path=`${entry.agent.agentId}/recent-runs?pageSize=20${cursor?`&cursor=${encodeURIComponent(cursor)}`:""}`;
 switch(entry.parameterSchema){
  case "image-config-v1":{const p=await configurationRequest(scope,path,imageSetRecentSchema,{signal});return {...p,items:p.items.map(r=>imageRecord(scope,entry,r))};}
  case "pod-artwork-config-v1":{const p=await configurationRequest(scope,path,artworkRecentSchema,{signal});return {...p,items:p.items.map(r=>imageRecord(scope,entry,r))};}
  case "product-selection-config-v1":{const p=await configurationRequest(scope,path,selectionRecentSchema,{signal});return {...p,items:p.items.map(r=>textRecord(scope,entry,r))};}
  case "shop-operations-diagnosis-config-v1":{const p=await configurationRequest(scope,path,diagnosisRecentSchema,{signal});return {...p,items:p.items.map(r=>textRecord(scope,entry,r))};}
  default:{const p=await configurationRequest(scope,path,recentRunsSchema,{signal});return {...p,items:p.items.map(r=>textRecord(scope,entry,r))};}
 }
}
export const nativeContextLabel=(kind:string)=>contexts[kind]??"原业务上下文";
export const nativePlatformLabel=(platform:string)=>platforms[platform]??platform.toUpperCase();
