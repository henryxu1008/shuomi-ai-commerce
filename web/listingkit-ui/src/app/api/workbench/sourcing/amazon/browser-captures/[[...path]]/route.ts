import { serverAuth } from "@/auth";
import { readZitadelIdentityFromSession } from "@/lib/server/zitadel-auth";
import { readZitadelServerAccessToken } from "@/lib/server/zitadel-server-token";
import { proxyAmazonCapture } from "@/lib/server/amazon-browser-capture";
export const dynamic="force-dynamic";
const handler=serverAuth(async(request)=>{
 const token=readZitadelServerAccessToken(request.auth as never)??"";
 const subject=readZitadelIdentityFromSession(request.auth as never)?.userId;
 const actor=typeof subject==="string"?subject:"";
 return proxyAmazonCapture(request,token,actor);
});
export {handler as GET,handler as POST};
