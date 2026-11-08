import {NextRequest} from "next/server";
import {serverAuth} from "@/auth";
import {readZitadelIdentityFromSession} from "@/lib/server/zitadel-auth";
import {readZitadelServerAccessToken} from "@/lib/server/zitadel-server-token";
import {proxyCommercialBilling} from "@/lib/server/commercial-billing-proxy";
import {withCommercialReadDeadline} from "@/lib/server/commercial-route";
import {workbenchProtocolError} from "@/lib/server/workbench-proxy";
const handle=withCommercialReadDeadline(serverAuth(async(request:NextRequest & {auth?:unknown})=>{
 const path=new URL(request.url).pathname.replace("/api/workbench/commercial/data-row-pricing","");
 const allowed=request.method==="GET"?path===""||path==="/history"||/^\/receipts\/[A-Za-z0-9._:-]{1,128}$/.test(path):request.method==="POST"&&path==="/publish";
 if(!allowed)return workbenchProtocolError(405,"INVALID_REQUEST","Method is not allowed");
 return proxyCommercialBilling(request,readZitadelServerAccessToken(request.auth as never),String(readZitadelIdentityFromSession(request.auth as never)?.userId??""));
}));
export const GET=handle;export const POST=handle;
export {PUT,PATCH,DELETE,HEAD,OPTIONS} from "@/lib/server/commercial-route";
