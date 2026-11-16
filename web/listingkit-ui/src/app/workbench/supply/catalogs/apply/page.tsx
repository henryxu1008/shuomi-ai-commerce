import {connection} from "next/server";
import {notFound,redirect} from "next/navigation";
import {SupplyApplicationsPage} from "@/components/workbench/supply-market/applications";
export default async function Page({searchParams=Promise.resolve({})}:{searchParams?:Promise<Record<string,string|string[]|undefined>>}={}){await connection();if(process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED!=="true")notFound();const {view}=await searchParams;if(view!==undefined&&view!=="progress"&&view!=="records")notFound();if(!view)redirect("/workbench/supply/catalogs");return <SupplyApplicationsPage connection view={view}/>}
