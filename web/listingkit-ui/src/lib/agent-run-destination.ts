import type {ConfigurationScope} from "@/lib/api/agent-configuration";

// Serving assertions only; the destination still authorizes the original scope.
export type NativeRunDestinations={ownProduct?:boolean;acquisition?:boolean;materials?:boolean};
export function nativeRunDestination(scope:ConfigurationScope,agentId:string,contextKind:string,contextId:string,requestKey?:string,targetPlatform?:string,available:NativeRunDestinations={}){
 const fallback={href:`/workbench/agents/mine/${encodeURIComponent(agentId)}`,direct:false};
 if(!requestKey)return fallback;
 const query=new URLSearchParams({agent_key:requestKey,agent_actor:scope.userId,agent_org:scope.organizationId});
 if(available.ownProduct&&agentId==="product.content.agent"&&contextKind==="own_product")return {href:`/workbench/data/mine/own/${contextId}/optimize?${query}`,direct:true};
 if(available.materials&&agentId==="listing.materials.agent"&&contextKind==="listing-target-record")return {href:`/workbench/supply/materials/${contextId}?${query}`,direct:true};
 if(available.acquisition&&["product.content.agent","product.title.agent"].includes(agentId)&&contextKind==="acquisition"){
  query.set("agent_platform",targetPlatform??"product");if(agentId==="product.content.agent")query.set("agent_kind","content");
  return {href:`/workbench/supply/acquisition/operation/${contextId}?${query}`,direct:true};
 }
 return fallback;
}
