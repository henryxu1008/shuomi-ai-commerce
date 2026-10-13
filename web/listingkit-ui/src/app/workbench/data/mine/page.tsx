import { connection } from "next/server";
import { notFound } from "next/navigation";
import { CollectionPage } from "@/components/workbench/collections/collection-page";
import {collectionID} from "@/lib/contracts/product-collection";

export default async function Page({searchParams}:{searchParams:Promise<{batchId?:string|string[];item?:string|string[]}>}) {
  await connection();
  if (process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED !== "true") notFound();
  const {batchId,item}=await searchParams;const parsed=collectionID.safeParse(batchId);const selected=collectionID.safeParse(item);
  return <CollectionPage initialBatchId={parsed.success?parsed.data:undefined} initialItemId={selected.success?selected.data:undefined} agentAvailable={process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED === "true"} supplyAvailable={process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED === "true" || process.env.LISTINGKIT_ACQUISITION_PLANS_ENABLED === "true" || process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED === "true"} />;
}
