import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {CompletionDraftPage} from "@/components/workbench/supply/completion-draft-page";

export default async function Page({params}:{params:Promise<{stage:string;sourceId:string}>}){
 await connection();const {stage,sourceId}=await params;
 if(stage!=="missing"||process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED!=="true"||!collectionID.safeParse(sourceId).success)notFound();
 return <CompletionDraftPage sourceId={sourceId}/>;
}
