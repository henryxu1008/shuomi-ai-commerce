import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {OwnProductHistoryPage} from "@/components/workbench/collections/own-product-history-page";
export default async function Page({params}:{params:Promise<{itemId:string}>}){
 await connection();const {itemId}=await params;
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||!collectionID.safeParse(itemId).success)notFound();
 return <OwnProductHistoryPage itemId={itemId}/>;
}
