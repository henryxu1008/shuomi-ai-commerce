import {z} from "zod";
import * as s from "@/lib/contracts/skill";
import {readBoundedStrictJSON} from "@/lib/api/strict-json-response";
import {hasTrustedSameOriginWrite} from "./same-origin-write";
import {hasEmptyBody} from "./members-proxy";
import {WORKBENCH_COOKIE_NAME} from "./workbench-proxy";
const safeHeaders={"Cache-Control":"private, no-store","X-Content-Type-Options":"nosniff"};
export const skillFailure=(status:number,code:string)=>Response.json({code},{status,headers:safeHeaders});
export type SkillDispatch={forwarded:boolean};
function endpoint(url:URL,method:string){
 const base="/api/workbench/skills/";
 if(!url.pathname.startsWith(base)||url.search.length>1600)return null;
 const p=url.pathname.slice(base.length).split("/"),space=s.skillScopeSchema.safeParse(p[0]);if(!space.success)return null;
 let output:z.ZodType,input:z.ZodType|undefined,match=false,zero=false;
 const get=method==="GET",post=method==="POST",uuid=(v:string)=>s.skillSelectionSchema.shape.skillId.safeParse(v).success;
 const v=(n:string)=>/^[1-9][0-9]*$/.test(n)&&Number.isSafeInteger(Number(n));
 if(p.length===1&&get)output=s.skillPageSchema;
 else if(p.length===2&&p[1]==="info"&&get)output=s.skillInfoSchema;
 else if(p.length===1&&post){input=s.skillInputSchema;output=s.skillReceiptSchema}
 else if(p.length===2&&p[1]==="validate"&&post){input=s.skillInputSchema;output=s.skillValidationSchema}
 else if(p.length===2&&p[1]==="copy"&&post){input=z.strictObject({source:s.skillRefSchema,allowCopy:z.boolean(),allowExport:z.boolean()});output=s.skillReceiptSchema}
 else if(p.length===2&&p[1]==="shares"&&p[0]!=="official"&&get)output=s.skillSharesSchema;
 else if(p.length===2&&p[1]==="shares"&&p[0]==="organization"&&post){input=s.skillSelectionSchema.extend({consent:z.literal(true)});output=s.skillShareSchema}
 else if(p[1]==="shares"&&uuid(p[2])&&p.length===3&&p[0]==="organization"&&get)output=s.skillVersionSchema;
 else if(p[1]==="shares"&&uuid(p[2])&&p.length===4&&post&&(p[0]==="personal"&&p[3]==="withdraw"||p[0]==="organization"&&["approve","reject"].includes(p[3]))){input=z.strictObject({});output=s.skillShareSchema}
 else if(p.length===2&&p[1]==="candidates"&&p[0]==="organization"&&get)output=s.skillCandidatesSchema;
 else if(p.length===2&&p[1]==="admission"&&p[0]==="organization"&&(get||method==="PUT")){output=s.skillAdmissionSchema;if(!get){input=z.strictObject({allowPersonal:z.boolean()});match=true;zero=true}}
 else if(uuid(p[1])&&p[2]==="versions"){
  if(p.length===3&&get)output=s.skillVersionsSchema;
  else if(p.length===3&&post){output=s.skillReceiptSchema;input=s.skillInputSchema;match=true}
  else if(v(p[3])&&p.length===4&&get)output=s.skillProjectionSchema;
  else if(v(p[3])&&p.length===5&&p[4]==="export"&&get)output=s.skillPackageSchema;
  else if(v(p[3])&&p.length===5&&post&&["publish","enable","disable","revoke"].includes(p[4])){output=s.skillReceiptSchema;input=z.strictObject({});match=true}
  else return null;
 }else return null;
 for(const [k,value] of url.searchParams){if(p.length!==1||!get||url.searchParams.getAll(k).length!==1)return null;if(k==="limit"){if(!/^[1-9][0-9]*$/.test(value)||Number(value)>50)return null}else if(k!=="cursor"||!/^[A-Za-z0-9_-]{1,1400}$/.test(value))return null}
 return {p,path:p.join("/"),space:space.data,output,input,match,zero};
}
export async function proxySkill(request:Request,token:string,user:string,dispatch:SkillDispatch={forwarded:false}):Promise<Response>{
 if(!token||!user)return skillFailure(401,"AUTHENTICATION_REQUIRED");
 if(request.headers.get("X-Expected-User-ID")!==user)return skillFailure(409,"IDENTITY_CONTEXT_CHANGED");
 const u=new URL(request.url),route=endpoint(u,request.method);
 if(!route||request.url.endsWith("?")||request.headers.has("content-encoding"))return skillFailure(400,"INVALID_REQUEST");
 const write=!!route.input;if(write&&!hasTrustedSameOriginWrite(request))return skillFailure(403,"FORBIDDEN");
 let org="";
 if(route.space==="organization"){
  const cookies=(request.headers.get("cookie")??"").split(";").map(v=>v.trim()).filter(v=>v.startsWith(WORKBENCH_COOKIE_NAME+"="));
  try{if(cookies.length===1)org=decodeURIComponent(cookies[0].slice(WORKBENCH_COOKIE_NAME.length+1))}catch{}
  if(!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(org)||org!==request.headers.get("X-Expected-Organization-ID"))return skillFailure(409,"ORGANIZATION_CONTEXT_CHANGED");
 }
 const signal=AbortSignal.any([request.signal,AbortSignal.timeout(12000)]);
 try{
  const headers=new Headers({Accept:"application/json",Authorization:`Bearer ${token}`});if(org)headers.set("X-Requested-Organization-ID",org);
  let body:string|undefined;
  if(write){
   const value=await readBoundedStrictJSON(new Response(request.body,{headers:{"Content-Type":request.headers.get("Content-Type")??""}}),98304,signal);
   const parsed=route.input!.safeParse(value);if(!parsed.success)return skillFailure(400,"INVALID_REQUEST");body=JSON.stringify(parsed.data);headers.set("Content-Type","application/json");
   if(route.p[1]!=="validate"){
    const key=request.headers.get("Idempotency-Key");if(!s.skillSelectionSchema.shape.skillId.safeParse(key).success)return skillFailure(400,"INVALID_REQUEST");headers.set("Idempotency-Key",key!);
   }
   if(route.match){const match=request.headers.get("If-Match");if(!match||!(route.zero?/^"(0|[1-9][0-9]*)"$/:/^"[1-9][0-9]*"$/).test(match)||!Number.isSafeInteger(Number(match.slice(1,-1))))return skillFailure(428,"PRECONDITION_REQUIRED");headers.set("If-Match",match)}
   else if(request.headers.has("If-Match")||request.headers.has("If-None-Match"))return skillFailure(400,"INVALID_REQUEST");
  }else if(!await hasEmptyBody(request,signal))return skillFailure(400,"INVALID_REQUEST");
  let origin:string;try{const api=new URL(process.env.LISTINGKIT_SERVICE_API_BASE??"");if(!["http:","https:"].includes(api.protocol)||api.username||api.password||api.search||api.hash||!["/api/v1","/api/v1/"].includes(api.pathname))throw Error();origin=api.origin}catch{return skillFailure(503,"SKILL_UNAVAILABLE")}
  signal.throwIfAborted();dispatch.forwarded=write&&route.p[1]!=="validate";
  const r=await fetch(`${origin}/api/v1/workbench/skills/${route.path}${u.search}`,{method:request.method,headers,body,cache:"no-store",redirect:"manual",signal});
  const payload=await readBoundedStrictJSON(r,262144,signal);signal.throwIfAborted();
  if(r.status!==200){const e=z.strictObject({code:z.string().regex(/^[A-Z_]{1,80}$/)}).safeParse(payload);return skillFailure(r.status>=400&&r.status<600?r.status:502,dispatch.forwarded&&r.status>=500?"OUTCOME_UNKNOWN":e.success?e.data.code:dispatch.forwarded?"OUTCOME_UNKNOWN":"INVALID_UPSTREAM_RESPONSE")}
  const value=route.output.parse(payload);
  const checkVersion=(v:s.SkillVersion)=>{
   const o=v.skill.owner;if(route.space==="personal"&&(o.kind!=="USER"||o.id!==user)||route.space==="organization"&&route.p[1]!=="candidates"&&route.p[1]!=="shares"&&(o.kind!=="ORGANIZATION"||o.id!==org)||route.space==="official"&&(o.kind!=="PLATFORM"||o.id!=="official"))throw Error();
   if(route.p[2]==="versions"&&(v.skill.id!==route.p[1]||route.p.length>=4&&v.version!==Number(route.p[3])))throw Error();
  };
  if(route.output===s.skillProjectionSchema)checkVersion(s.skillProjectionSchema.parse(value).version);
  if(route.output===s.skillPageSchema)for(const item of s.skillPageSchema.parse(value).items)checkVersion(item.version);
  if(route.output===s.skillVersionsSchema)for(const item of s.skillVersionsSchema.parse(value).items)checkVersion(item);
  if(route.output===s.skillCandidatesSchema)for(const item of s.skillCandidatesSchema.parse(value).items){const o=item.skill.owner;if(item.state!=="ENABLED"||o.kind==="USER"&&o.id!==user||o.kind==="ORGANIZATION"&&o.id!==org||o.kind==="PLATFORM"&&o.id!=="official")throw Error()}
  if(route.output===s.skillSharesSchema)for(const item of s.skillSharesSchema.parse(value).items){if(route.space==="personal"?item.submittedBy!==user:item.organizationId!==org)throw Error()}
  if(route.output===s.skillShareSchema){const item=s.skillShareSchema.parse(value);if(route.space==="personal"?item.submittedBy!==user:item.organizationId!==org)throw Error();if(route.p[2]&&item.id!==route.p[2])throw Error();if(route.p.length===2){const wanted=s.skillSelectionSchema.extend({consent:z.literal(true)}).parse(JSON.parse(body!));if(item.source.id!==wanted.skillId||item.source.version!==wanted.version||item.source.digest!==wanted.digest||item.submittedBy!==user)throw Error()}}
  if(route.output===s.skillReceiptSchema){const v=s.skillReceiptSchema.parse(value),op=route.p.length===1?"create":route.p[1]==="copy"?"copy":route.p.length===3?"save-version":route.p[4];if(v.operation!==op||route.p[2]==="versions"&&v.skillId!==route.p[1]||route.p.length===5&&v.version!==Number(route.p[3]))throw Error()}
  return Response.json(value,{headers:safeHeaders});
 }catch{return skillFailure(dispatch.forwarded?502:request.method==="GET"?503:400,dispatch.forwarded?"OUTCOME_UNKNOWN":request.method==="GET"?"SKILL_UNAVAILABLE":"INVALID_REQUEST")}
}
