import {connection} from "next/server";
import {notFound} from "next/navigation";
import {SupplyApplicationsPage} from "@/components/workbench/supply-market/applications";
import {requireAccountUserId} from "@/lib/server/account-page-auth";
export default async function Page({searchParams}:{searchParams:Promise<{kind?:string|string[]}>}){await connection();if(process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED!=="true")notFound();const expectedUserId=await requireAccountUserId("/workbench/supply/operator");const query=await searchParams;return <SupplyApplicationsPage admin expectedUserId={expectedUserId} initialKind={query.kind==="official"?"official":undefined}/>}
