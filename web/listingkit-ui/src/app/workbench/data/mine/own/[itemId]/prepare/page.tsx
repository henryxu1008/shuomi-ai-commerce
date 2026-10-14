import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {OwnProductPreparePage} from "@/components/workbench/collections/own-product-prepare-page";

export default async function Page({params}:{params:Promise<{itemId:string}>}){
 await connection();const {itemId}=await params;
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED!=="true"||!collectionID.safeParse(itemId).success)notFound();
 return <OwnProductPreparePage itemId={itemId}/>;
}
