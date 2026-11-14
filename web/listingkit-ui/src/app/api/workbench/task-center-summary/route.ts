import {NextRequest} from "next/server";
import {serverAuth} from "@/auth";
import {readZitadelIdentityFromSession} from "@/lib/server/zitadel-auth";
import {readZitadelServerAccessToken} from "@/lib/server/zitadel-server-token";
import {collectTaskSummary} from "@/lib/server/task-center-summary";

export const dynamic="force-dynamic";
export async function GET(request:NextRequest){
 const controller=new AbortController();const abort=()=>controller.abort();
 request.signal.addEventListener("abort",abort,{once:true});if(request.signal.aborted)abort();
 const timer=setTimeout(abort,12000);
 let ended=()=>{};
 const timeout=new Promise<Response>(resolve=>{ended=()=>resolve(Response.json({status:"partial",counts:null,excluded:0,capturedAt:new Date().toISOString(),reasons:["unavailable"]},{headers:{"Cache-Control":"private, no-store"}}));controller.signal.addEventListener("abort",ended,{once:true});if(controller.signal.aborted)ended()});
 try{
  const scoped=new NextRequest(request.url,{method:"GET",headers:request.headers,signal:controller.signal});
  const authenticated=serverAuth(async req=>collectTaskSummary(scoped,readZitadelServerAccessToken(req.auth),String(readZitadelIdentityFromSession(req.auth)?.userId??"")));
  return await Promise.race([authenticated(scoped,{params:Promise.resolve({})}),timeout])??Response.json({code:"AUTHENTICATION_REQUIRED"},{status:401,headers:{"Cache-Control":"private, no-store"}});
 }catch{return Response.json({code:"DEPENDENCY_UNAVAILABLE"},{status:503,headers:{"Cache-Control":"private, no-store"}})}finally{clearTimeout(timer);request.signal.removeEventListener("abort",abort);controller.signal.removeEventListener("abort",ended);controller.abort()}
}
