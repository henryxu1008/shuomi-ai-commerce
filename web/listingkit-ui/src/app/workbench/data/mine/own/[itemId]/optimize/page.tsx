import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {OwnProductOptimizePage} from "@/components/workbench/collections/own-product-optimize-page";
import {parseSkillLaunchRecord} from "@/lib/contracts/skill-launch";
import {isKnowledgeAvailable} from "@/lib/server/knowledge-availability";

export default async function Page({params,searchParams}:{params:Promise<{itemId:string}>;searchParams:Promise<Record<string,string|string[]|undefined>>}){
 await connection();const {itemId}=await params;
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||!collectionID.safeParse(itemId).success)notFound();
 return <OwnProductOptimizePage launch={parseSkillLaunchRecord(await searchParams)} itemId={itemId} enabled={process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED==="true"} knowledgeAvailable={isKnowledgeAvailable()}/>;
}
