import {isOfflineSheinRecordsAvailable} from "@/lib/server/offline-shein-records-availability";
import { connection } from "next/server";
import { notFound } from "next/navigation";
import { CollectionPage } from "@/components/workbench/collections/collection-page";
import {researchTransfer} from "@/lib/contracts/market-research";
import {collectionID} from "@/lib/contracts/product-collection";

export default async function Page({searchParams}:{searchParams:Promise<{batchId?:string|string[];item?:string|string[];transfer?:string|string[];view?:string|string[]}>}) {
  await connection();
  if (process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED !== "true") notFound();
  const {batchId,item,view,transfer}=await searchParams;const parsed=collectionID.safeParse(batchId);const selected=collectionID.safeParse(item);
  let initialTransferSelection;try{if(typeof transfer==="string"&&transfer.length<=8192){const result=researchTransfer.safeParse(JSON.parse(transfer));if(result.success&&parsed.success)initialTransferSelection=result.data;else notFound();}else if(transfer!==undefined)notFound();}catch{notFound();}
  return <CollectionPage offlineAvailable={isOfflineSheinRecordsAvailable()} agentAvailable={process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED === "true"} initialTransferSelection={initialTransferSelection} initialView={view==="own"?"own":"batches"} initialBatchId={parsed.success?parsed.data:undefined} initialItemId={selected.success?selected.data:undefined} supplyAvailable={process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED === "true" || process.env.LISTINGKIT_ACQUISITION_PLANS_ENABLED === "true" || process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED === "true"} />;
}
