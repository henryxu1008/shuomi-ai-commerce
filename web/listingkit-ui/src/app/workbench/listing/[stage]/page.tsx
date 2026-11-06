import { connection } from "next/server";
import { notFound } from "next/navigation";
import { PreparationsPage } from "@/components/workbench/listing/preparations-page";
import { ListingStageTabs } from "@/components/workbench/listing/listing-stage-tabs";
import { listingHome } from "@/lib/workbench/listing-navigation";
import { ListingPage } from "@/components/workbench/listing/listing-page";
import { CompletionDraftsPage } from "@/components/workbench/listing/completion-drafts-page";
import { stageSchema } from "@/lib/contracts/supply-chain";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
import { ConsolePage, ConsoleState } from "@/components/workbench/console/console-page";
import { findConsoleRoute } from "@/lib/workbench/console-navigation";
import { isReportSheinTargetAvailable } from "@/lib/server/report-center-availability";
export default async function Page({params,searchParams}:{params:Promise<{stage:string}>;searchParams:Promise<Record<string,string|string[]|undefined>>}){
 await connection();
 const full=process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED==="true";
 const sourceMaterials=process.env.LISTINGKIT_SOURCE_MATERIALS_ENABLED==="true";
 const preparation=process.env.LISTINGKIT_ACQUISITION_PLANS_ENABLED==="true";
 const completion=process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED==="true";
 const stage=stageSchema.safeParse((await params).stage);if(!stage.success)notFound();
 const query=await searchParams;
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||!full&&(!(preparation||completion)||stage.data!=="waiting")&&(!completion||stage.data!=="missing")){
  const route=findConsoleRoute(listingHome+"/"+stage.data)??findConsoleRoute(listingHome)!;
  const context=Object.fromEntries(["preparation","store","operation","plan"].flatMap(key=>{
   const value=query[key];
   return typeof value==="string"&&isAcquisitionUUID(value)?[[key,value]]:[];
  }));
  return <ConsolePage title="商品刊登" breadcrumbs={route.trail} description="按目标店铺整理商品资料并查看上传结果。">
    <ListingStageTabs current={stage.data} context={context}/>
    <ConsoleState kind="unavailable" title="平台处理尚未配置">此阶段需要正式官方应用、SHEIN 渠道及供应链运行配置。已保存的商品和待上传批次仍保留在原入口。</ConsoleState>
  </ConsolePage>;
 }
 const read=(name:string)=>{const value=query[name];if(value===undefined)return undefined;if(typeof value!=="string"||!isAcquisitionUUID(value))notFound();return value};
 const record=read("record");
 if(record){
  if(stage.data!=="ready"||Object.keys(query).some(key=>key!=="record"))notFound();
  return <ListingPage initialStage={stage.data} initialRecord={record} reportSourcesAvailable={isReportSheinTargetAvailable()}/>;
 }
 if(query.mode!==undefined&&(query.mode!=="platform"||stage.data!=="missing"||!full)&&(query.mode!=="local"||stage.data!=="waiting"||!completion))notFound();
 if(completion&&stage.data==="missing"&&query.mode!=="platform"&&!query.plan)return <CompletionDraftsPage initialPreparation={read("preparation")} initialSource={read("source")} fullAvailable={full}/>;
 if(!full||completion&&stage.data==="waiting"&&query.mode==="local")return <PreparationsPage initialPreparation={read("preparation")} completionAvailable={completion} sourceMaterialsAvailable={sourceMaterials}/>;
 return <ListingPage initialStage={stage.data} initialPreparation={read("preparation")} initialStore={read("store")} initialOperation={read("operation")} initialPlan={read("plan")} completionAvailable={completion} sourceMaterialsAvailable={sourceMaterials} materialsAvailable={full&&process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED==="true"}/>;
}
