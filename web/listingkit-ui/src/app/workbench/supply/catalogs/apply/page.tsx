import {connection} from "next/server";
import {notFound} from "next/navigation";
import {SupplyCatalogsPage} from "@/components/workbench/supply-market/catalogs";
import {SupplyApplicationsPage} from "@/components/workbench/supply-market/applications";
export default async function Page({searchParams=Promise.resolve({})}:{searchParams?:Promise<Record<string,string|string[]|undefined>>}={}){await connection();if(process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED!=="true")notFound();const {view}=await searchParams;if(view!==undefined&&view!=="progress"&&view!=="records")notFound();return view?<SupplyApplicationsPage connection view={view}/>:<SupplyCatalogsPage view="apply"/>}
