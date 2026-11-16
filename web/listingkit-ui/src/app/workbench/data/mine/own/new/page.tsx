import {connection} from "next/server";
import {notFound} from "next/navigation";
import {OwnProductEntryPage,OfficialOwnProductEntryPage} from "@/components/workbench/collections/own-product-entry-page";

export default async function Page({searchParams}:{searchParams:Promise<{continuation?:string|string[]}>}){
 await connection();
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true")notFound();
 const query=await searchParams;
 if(query.continuation==="official"){if(process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED!=="true")notFound();return <OfficialOwnProductEntryPage/>}
 return <OwnProductEntryPage/>;
}
