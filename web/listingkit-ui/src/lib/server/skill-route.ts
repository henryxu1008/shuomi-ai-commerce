import {NextRequest} from "next/server";
import {serverAuth} from "@/auth";
import {readZitadelIdentityFromSession} from "./zitadel-auth";
import {readZitadelServerAccessToken} from "./zitadel-server-token";
import {proxySkill,skillFailure,type SkillDispatch} from "./skill-proxy";
export async function handleSkill(request:NextRequest){
 const controller=new AbortController(),abort=()=>controller.abort();request.signal.addEventListener("abort",abort,{once:true});if(request.signal.aborted)abort();const timer=setTimeout(abort,15000);
 const dispatch:SkillDispatch={forwarded:false};const failure=()=>skillFailure(504,dispatch.forwarded?"OUTCOME_UNKNOWN":"DEADLINE_EXCEEDED");let finish=()=>{};
 const ended=new Promise<Response>(resolve=>{finish=()=>resolve(failure());controller.signal.addEventListener("abort",finish,{once:true});if(controller.signal.aborted)finish()});
 try{const authenticated=serverAuth(async(r:NextRequest&{auth?:unknown})=>proxySkill(r,readZitadelServerAccessToken(r.auth as never),String(readZitadelIdentityFromSession(r.auth as never)?.userId??""),dispatch));const scoped=new NextRequest(request.url,{method:request.method,headers:request.headers,body:request.body,signal:controller.signal});return await Promise.race([authenticated(scoped,{params:Promise.resolve({})}),ended])??skillFailure(503,"SKILL_UNAVAILABLE")}
 catch{return controller.signal.aborted?failure():skillFailure(503,dispatch.forwarded?"OUTCOME_UNKNOWN":"SKILL_UNAVAILABLE")}finally{clearTimeout(timer);request.signal.removeEventListener("abort",abort);controller.signal.removeEventListener("abort",finish)}
}
export const rejectSkill=()=>skillFailure(405,"INVALID_REQUEST");
