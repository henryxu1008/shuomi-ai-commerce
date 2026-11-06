import { redirect } from "next/navigation";
import { fulfillmentID } from "@/lib/api/fulfillment";

export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){
 const params=await searchParams;
 const query=new URLSearchParams({step:"purchase"});
 for(const key of ["storeId","sourceId"]){
  const value=params[key];
  if(typeof value==="string"&&fulfillmentID.safeParse(value).success)query.set(key,value);
 }
 if(params.kind==="platform_supply"||params.kind==="consumer_order")query.set("kind",params.kind);
 redirect("/workbench/store-orders/fulfill?"+query);
}
