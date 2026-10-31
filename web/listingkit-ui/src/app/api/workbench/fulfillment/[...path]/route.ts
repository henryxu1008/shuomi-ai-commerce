import { NextRequest } from "next/server";
import { serverAuth } from "@/auth";
import { readZitadelIdentityFromSession } from "@/lib/server/zitadel-auth";
import { readZitadelServerAccessToken } from "@/lib/server/zitadel-server-token";
import { proxyFulfillment } from "@/lib/server/fulfillment-proxy";
export const dynamic="force-dynamic";
const authenticated=serverAuth(async(request:NextRequest&{auth?:unknown})=>proxyFulfillment(request,readZitadelServerAccessToken(request.auth as never),String(readZitadelIdentityFromSession(request.auth as never)?.userId??"")));
async function handle(request:NextRequest):Promise<Response>{
 try{return await authenticated(request,{params:Promise.resolve({})})??Response.json({code:"FULFILLMENT_UNAVAILABLE"},{status:503,headers:{"Cache-Control":"private, no-store"}})}catch{return Response.json({code:"FULFILLMENT_UNKNOWN"},{status:503,headers:{"Cache-Control":"private, no-store"}})}
}
export const GET=handle;
export const POST=handle;
const reject=()=>Response.json({code:"FULFILLMENT_INVALID"},{status:405,headers:{"Cache-Control":"private, no-store"}});
export const PUT=reject;export const DELETE=reject;export const PATCH=reject;export const HEAD=reject;export const OPTIONS=reject;
