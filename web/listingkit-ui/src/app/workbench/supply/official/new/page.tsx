import {connection} from "next/server";
import {notFound} from "next/navigation";
import {SupplyApplicationForm} from "@/components/workbench/supply-market/applications";
import {marketID} from "@/lib/contracts/supply-market";
export default async function Page({searchParams}:{searchParams:Promise<{item?:string|string[]}>}){await connection();if(process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED!=="true")notFound();const query=await searchParams;const item=marketID.safeParse(query.item);return <SupplyApplicationForm official initialItemId={item.success?item.data:undefined}/>}
