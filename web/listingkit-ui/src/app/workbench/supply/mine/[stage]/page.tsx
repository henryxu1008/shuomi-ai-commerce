import { connection } from "next/server";
import { notFound } from "next/navigation";
import { PreparationsPage } from "@/components/workbench/supply/preparations-page";
import { SupplyPage } from "@/components/workbench/supply/supply-page";
import { stageSchema } from "@/lib/contracts/supply-chain";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
export default async function Page({params,searchParams}:{params:Promise<{stage:string}>;searchParams:Promise<Record<string,string|string[]|undefined>>}){
 await connection();
 const full=process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED==="true";
 const preparation=process.env.LISTINGKIT_ACQUISITION_PLANS_ENABLED==="true";
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||!full&&!preparation)notFound();
 const stage=stageSchema.safeParse((await params).stage);if(!stage.success||stage.data==="all")notFound();
 if(!full&&stage.data!=="waiting")notFound();
 const query=await searchParams;const read=(name:string)=>{const value=query[name];if(value===undefined)return undefined;if(typeof value!=="string"||!isAcquisitionUUID(value))notFound();return value};
 if(!full)return <PreparationsPage initialPreparation={read("preparation")}/>;
 return <SupplyPage initialStage={stage.data} initialPreparation={read("preparation")} initialStore={read("store")} initialOperation={read("operation")}/>;
}
