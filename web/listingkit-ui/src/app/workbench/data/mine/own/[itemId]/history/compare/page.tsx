import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID,collectionVersion} from "@/lib/contracts/product-collection";
import {OwnProductHistoryPage} from "@/components/workbench/collections/own-product-history-page";
export default async function Page({params,searchParams}:{params:Promise<{itemId:string}>;searchParams:Promise<Record<string,string|string[]|undefined>>}){
 await connection();const {itemId}=await params,query=await searchParams;
 const from=collectionVersion.safeParse(query.from),to=collectionVersion.safeParse(query.to);
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||!collectionID.safeParse(itemId).success||!from.success||!to.success||from.data===to.data||Object.keys(query).some(key=>key!=="from"&&key!=="to"))notFound();
 return <OwnProductHistoryPage itemId={itemId} from={from.data} to={to.data}/>;
}
