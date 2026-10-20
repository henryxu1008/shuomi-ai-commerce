import {connection} from "next/server";
import {notFound} from "next/navigation";
import {MaterialsPage} from "@/components/workbench/supply/materials-page";
import {collectionID} from "@/lib/contracts/product-collection";
export default async function Page({params}:{params:Promise<{recordId:string}>}){
 await connection();
 const {recordId}=await params;
 if(process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED!=="true"||process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED!=="true"||!collectionID.safeParse(recordId).success)notFound();
 return <MaterialsPage recordId={recordId}/>;
}
