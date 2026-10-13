import { connection } from "next/server";
import { notFound } from "next/navigation";
import { PreparationsPage } from "@/components/workbench/supply/preparations-page";
import { SupplyPage } from "@/components/workbench/supply/supply-page";
import { CompletionDraftsPage } from "@/components/workbench/supply/completion-drafts-page";
import { stageSchema } from "@/lib/contracts/supply-chain";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
import { ConsolePage, ConsoleState } from "@/components/workbench/console/console-page";
import { findConsoleRoute } from "@/lib/workbench/console-navigation";
export default async function Page({params,searchParams}:{params:Promise<{stage:string}>;searchParams:Promise<Record<string,string|string[]|undefined>>}){
 await connection();
 const full=process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED==="true";
 const preparation=process.env.LISTINGKIT_ACQUISITION_PLANS_ENABLED==="true";
 const completion=process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED==="true";
 const stage=stageSchema.safeParse((await params).stage);if(!stage.success||stage.data==="all")notFound();
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||!full&&(!(preparation||completion)||stage.data!=="waiting")&&(!completion||stage.data!=="missing")){
  const route=findConsoleRoute(`/workbench/supply/mine/${stage.data}`)!;
  return <ConsolePage title={route.node.label} breadcrumbs={route.trail} description="按目标平台整理商品并完成发布。"><ConsoleState kind="unavailable" title="平台处理尚未配置">此阶段需要正式官方应用、SHEIN 渠道及供应链运行配置。已保存的商品和待适配批次仍保留在原入口。</ConsoleState></ConsolePage>;
 }
 const query=await searchParams;const read=(name:string)=>{const value=query[name];if(value===undefined)return undefined;if(typeof value!=="string"||!isAcquisitionUUID(value))notFound();return value};
 if(query.mode!==undefined&&(query.mode!=="platform"||stage.data!=="missing"||!full)&&(query.mode!=="local"||stage.data!=="waiting"||!completion))notFound();
 if(completion&&stage.data==="missing"&&query.mode!=="platform")return <CompletionDraftsPage initialPreparation={read("preparation")} initialSource={read("source")} fullAvailable={full}/>;
 if(!full||completion&&stage.data==="waiting"&&query.mode==="local")return <PreparationsPage initialPreparation={read("preparation")} completionAvailable={completion}/>;
 return <SupplyPage initialStage={stage.data} initialPreparation={read("preparation")} initialStore={read("store")} initialOperation={read("operation")} completionAvailable={completion} materialsAvailable={full&&process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED==="true"}/>;
}
