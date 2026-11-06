import {connection} from "next/server";
import {notFound} from "next/navigation";
import {ObservationPage} from "@/components/workbench/store-observations/observation-page";
import {StorePublicationDetail} from "@/components/workbench/supply/store-publication-detail";
import {isAcquisitionUUID} from "@/lib/contracts/product-acquisition";
export default async function StoreProductsPage({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}) {
 await connection();const query=await searchParams;
 if(["source","store","record","receipt"].some(k=>query[k]!==undefined)){
  if(process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED!=="true"||process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||Object.keys(query).some(k=>!["source","store","record","receipt"].includes(k)))notFound();
  const read=(k:string)=>{const v=query[k];if(typeof v!=="string"||!isAcquisitionUUID(v))notFound();return v;};
  return <StorePublicationDetail sourceId={read("source")} storeId={read("store")} recordId={read("record")} receiptId={read("receipt")}/>;
 }
 return <ObservationPage kind="products"/>;
}
