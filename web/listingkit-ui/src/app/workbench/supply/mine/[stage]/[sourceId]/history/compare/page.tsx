import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {completionHistoryNavigation} from "@/lib/contracts/completion-draft-navigation";
import {CompletionHistoryPage} from "@/components/workbench/supply/completion-history-page";

export default async function Page({params,searchParams}:{params:Promise<{stage:string;sourceId:string}>;searchParams:Promise<Record<string,string|string[]|undefined>>}){
 await connection();const {stage,sourceId}=await params;
 if(stage!=="missing"||process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED!=="true"||!collectionID.safeParse(sourceId).success)notFound();
 const navigation=completionHistoryNavigation(await searchParams,"compare");if(!navigation)notFound();
 return <CompletionHistoryPage sourceId={sourceId} {...navigation}/>;
}
