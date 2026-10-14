import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {OwnProductDetailPage} from "@/components/workbench/collections/own-product-detail-page";

export default async function Page({params}:{params:Promise<{itemId:string}>}){
 await connection();
 const {itemId}=await params;
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||!collectionID.safeParse(itemId).success)notFound();
 return <OwnProductDetailPage itemId={itemId}/>;
}
