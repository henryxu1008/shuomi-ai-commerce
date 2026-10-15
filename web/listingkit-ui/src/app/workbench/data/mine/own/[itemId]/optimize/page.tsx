import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {OwnProductOptimizePage} from "@/components/workbench/collections/own-product-optimize-page";
import {isKnowledgeAvailable} from "@/lib/server/knowledge-availability";

export default async function Page({params}:{params:Promise<{itemId:string}>}){
 await connection();const {itemId}=await params;
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||!collectionID.safeParse(itemId).success)notFound();
 return <OwnProductOptimizePage itemId={itemId} enabled={process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED==="true"} knowledgeAvailable={isKnowledgeAvailable()}/>;
}
