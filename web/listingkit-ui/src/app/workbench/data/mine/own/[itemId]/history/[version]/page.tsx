import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID,collectionVersion} from "@/lib/contracts/product-collection";
import {OwnProductHistoryPage} from "@/components/workbench/collections/own-product-history-page";
export default async function Page({params}:{params:Promise<{itemId:string;version:string}>}){
 await connection();const {itemId,version}=await params;
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||!collectionID.safeParse(itemId).success||!collectionVersion.safeParse(version).success)notFound();
 return <OwnProductHistoryPage itemId={itemId} version={version}/>;
}
