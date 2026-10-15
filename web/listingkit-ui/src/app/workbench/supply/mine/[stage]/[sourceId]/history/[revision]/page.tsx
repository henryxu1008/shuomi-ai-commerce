import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {completionHistoryNavigation,completionRevision} from "@/lib/contracts/completion-draft-navigation";
import {CompletionHistoryPage} from "@/components/workbench/supply/completion-history-page";

export default async function Page({params,searchParams}:{params:Promise<{stage:string;sourceId:string;revision:string}>;searchParams:Promise<Record<string,string|string[]|undefined>>}){
 await connection();const {stage,sourceId,revision}=await params,saved=completionRevision(revision);
 if(stage!=="missing"||process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED!=="true"||!collectionID.safeParse(sourceId).success||saved===null)notFound();
 const navigation=completionHistoryNavigation(await searchParams,"history");if(!navigation)notFound();
 return <CompletionHistoryPage sourceId={sourceId} revision={saved} {...navigation}/>;
}
