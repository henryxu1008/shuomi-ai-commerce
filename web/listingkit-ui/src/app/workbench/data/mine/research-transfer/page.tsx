import {connection} from "next/server";
import {notFound} from "next/navigation";
import {CollectionPage} from "@/components/workbench/collections/collection-page";

export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}) {
  await connection();
  if (process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true") notFound();
  const {handoff}=await searchParams;
  const token=typeof handoff==="string"&&handoff.length<=64?handoff:"";
  const supplyAvailable=process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED==="true" ||
    process.env.LISTINGKIT_ACQUISITION_PLANS_ENABLED==="true" ||
    process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED==="true";
  return <CollectionPage supplyAvailable={supplyAvailable} researchToken={token}/>;
}
