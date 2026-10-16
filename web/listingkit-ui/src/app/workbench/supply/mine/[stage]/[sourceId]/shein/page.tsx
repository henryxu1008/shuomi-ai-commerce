import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {sheinImportNavigation} from "@/lib/contracts/shein-import-navigation";
import {SheinImportPage} from "@/components/workbench/supply/shein-import-page";
export default async function Page({params,searchParams}:{params:Promise<{stage:string;sourceId:string}>;searchParams:Promise<Record<string,string|string[]|undefined>>}){
 await connection();const {stage,sourceId}=await params;
 if(stage!=="missing"||process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED!=="true"||process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED!=="true"||!collectionID.safeParse(sourceId).success)notFound();
 const navigation=sheinImportNavigation(await searchParams);if(!navigation)notFound();return <SheinImportPage sourceId={sourceId} {...navigation}/>;
}
