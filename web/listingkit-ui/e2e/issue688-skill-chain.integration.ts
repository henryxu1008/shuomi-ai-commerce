import {createServer,type Server} from "node:http";
import type {AddressInfo} from "node:net";
import {beforeAll,afterAll,it,expect,vi} from "vitest";
import {NextRequest} from "next/server";
const session=vi.hoisted(()=>({userId:"admin",token:"admin"}));
vi.mock("@/auth",()=>({serverAuth:(handler:(r:NextRequest&{auth?:unknown},context:unknown)=>unknown)=>(r:NextRequest,context:unknown)=>{Object.defineProperty(r,"auth",{value:session});return handler(r,context)}}));
vi.mock("@/lib/server/zitadel-auth",async importOriginal=>({...await importOriginal<object>(),readZitadelIdentityFromSession:(s:unknown)=>{if(s!==session)throw Error("fixture session mismatch");return {userId:"admin"}}}));
vi.mock("@/lib/server/zitadel-server-token",()=>({readZitadelServerAccessToken:(s:unknown)=>{if(s!==session)throw Error();return "admin"}}));
import * as skillsRoute from "@/app/api/workbench/skills/[[...path]]/route";
import * as workbenchRoute from "@/app/api/workbench/[...path]/route";
import {handleProductTitleReview} from "@/lib/server/product-title-review-route";
import {skillRequest} from "@/lib/api/skill";
import * as s from "@/lib/contracts/skill";
import {requestProductAgent} from "@/lib/api/product-agent";
import {fetchProductTitleProposal,decideProductTitleProposal,applyProductTitleProposal} from "@/lib/api/product-title-review-client";
const upstream=process.env.SKILL_INTEGRATION_ORIGIN,operation=process.env.SKILL_INTEGRATION_OPERATION,modeRevision=process.env.SKILL_INTEGRATION_MODE_REVISION;
if(!upstream||new URL(upstream).hostname!=="127.0.0.1"||!operation||!modeRevision)throw Error("Run through the owned PostgreSQL TestSkillConsoleClientsUseActualBFFAndOwners fixture");
const nativeFetch=globalThis.fetch;let server:Server,origin:string;
beforeAll(async()=>{
 process.env.LISTINGKIT_SERVICE_API_BASE=upstream+"/api/v1";process.env.PRODUCT_REVIEW_API_ORIGIN=upstream;
 server=createServer(async(req,res)=>{try{
  const chunks:Buffer[]=[];for await(const chunk of req)chunks.push(Buffer.from(chunk));const body=Buffer.concat(chunks).toString("utf8");const headers=new Headers();for(const [k,v] of Object.entries(req.headers))if(v!==undefined)headers.set(k,Array.isArray(v)?v.join(","):v);
  const request=new NextRequest(new URL(req.url!,origin),{method:req.method,headers,...(body?{body}:{})});const path=new URL(req.url!,origin).pathname;
  const response=path.startsWith("/api/workbench/skills/")?await (req.method==="GET"?skillsRoute.GET: req.method==="PUT"?skillsRoute.PUT:skillsRoute.POST)(request):path.startsWith("/api/product/text-proposals")?await handleProductTitleReview(request):await (req.method==="GET"?workbenchRoute.GET:workbenchRoute.POST)(request,{params:Promise.resolve({path:path.slice("/api/workbench/".length).split("/")})});
  res.writeHead(response.status,Object.fromEntries(response.headers));res.end(Buffer.from(await response.arrayBuffer()));
 }catch{res.writeHead(500);res.end()}});
 await new Promise<void>(resolve=>server.listen(0,"127.0.0.1",resolve));origin=`http://127.0.0.1:${(server.address() as AddressInfo).port}`;process.env.LISTINGKIT_PUBLIC_BASE_URL=origin;
 vi.stubGlobal("fetch",(input:RequestInfo|URL,init?:RequestInit)=>{if(typeof input==="string"&&input.startsWith("/")){const headers=new Headers(init?.headers);headers.set("cookie","shuomi_effective_organization=B");if(init?.method&&init.method!=="GET")headers.set("Origin",origin);return nativeFetch(new URL(input,origin),{...init,headers})}return nativeFetch(input,init)});
});
afterAll(async()=>{vi.unstubAllGlobals();if(server)await new Promise<void>((resolve,reject)=>server.close(e=>e?reject(e):resolve()))});
it("uses actual typed clients and authenticated Next routes through governed content, Review and Catalog",async()=>{
 const scope={userId:"admin",organizationId:"B"},personal={userId:"admin"};
 const mutate=<T>(path:string,payload:unknown,schema:import("zod").z.ZodType<T>,revision?:number|string)=>skillRequest(path.startsWith("organization")?scope:personal,path,schema,{method:path.endsWith("admission")?"PUT":"POST",headers:{"Content-Type":"application/json","Idempotency-Key":crypto.randomUUID(),...(revision!==undefined?{"If-Match":`"${revision}"`}:{})},body:JSON.stringify(payload)});
 let official=await mutate("official",{files:[{path:"SKILL.md",text:"---\nname: product-title-normalization\ndescription: Normalize exact saved facts.\n---\nNormalize exact saved facts using [style](references/style.md).\n"},{path:"references/style.md",text:"Keep only saved attributes.\n"}],allowCopy:true,allowExport:true},s.skillReceiptSchema);
 for(const action of ["publish","enable"]){official=await mutate(`official/${official.skillId}/versions/1/${action}`,{},s.skillReceiptSchema,official.revision)}
 const template=await skillRequest(personal,`official/${official.skillId}/versions/1`,s.skillProjectionSchema);
 let copy=await mutate("personal/copy",{source:s.skillRef(template.version),allowCopy:true,allowExport:true},s.skillReceiptSchema);
 for(const action of ["publish","enable"]){copy=await mutate(`personal/${copy.skillId}/versions/1/${action}`,{},s.skillReceiptSchema,copy.revision)}
 const method=(await skillRequest(personal,`personal/${copy.skillId}/versions/1`,s.skillProjectionSchema)).version;
 let admission=await skillRequest(scope,"organization/admission",s.skillAdmissionSchema);expect(admission.policy.allowPersonal).toBe(false);admission=await mutate("organization/admission",{allowPersonal:true},s.skillAdmissionSchema,admission.policy.revision);expect(admission.policy.allowPersonal).toBe(true);
 const selection={skillId:method.skill.id,version:method.version,digest:method.package.digest},key=crypto.randomUUID(),fields={targetLanguage:"en-US",title:true,description:false,sellingPoints:false,seoKeywords:false};
 const start=()=>requestProductAgent("start",operation!,key,scope,new AbortController().signal,undefined,undefined,"product",undefined,undefined,modeRevision!,fields,"content",selection);
 const run=await start();expect("skillVersionRef" in run&&run.skillVersionRef?.digest).toBe(selection.digest);expect((await start()).requestKey).toBe(key);
 const linked=await requestProductAgent("review",operation!,key,scope,new AbortController().signal,undefined,undefined,undefined,undefined,undefined,undefined,undefined,"content");if(!("proposalId" in linked))throw Error();
 const reviewScope={...scope,proposalId:linked.proposalId};let proposal=await fetchProductTitleProposal(reviewScope);expect(proposal.policy).toBe("product-content-review-v1");if(proposal.policy!=="product-content-review-v1")throw Error();expect(proposal.execution.skillVersionRef?.id).toBe(selection.skillId);
 proposal=await decideProductTitleProposal({...reviewScope,idempotencyKey:crypto.randomUUID(),input:{action:"accept",expected_revision:proposal.revision}});proposal=await applyProductTitleProposal({...reviewScope,idempotencyKey:crypto.randomUUID(),input:{expected_revision:proposal.revision}});expect(proposal.state).toBe("applied");expect(proposal.apply_receipt?.product_version).toBe("2");
 const path="references/"+"x".repeat(242)+".md",description="😀".repeat(1024);
 const boundary=await mutate("personal",{files:[{path:"SKILL.md",text:`---\nname: boundary-method\ndescription: ${JSON.stringify(description)}\n---\nUse [reference](${path}).\n`},{path,text:"Reference."}],allowCopy:false,allowExport:true},s.skillReceiptSchema);
 const saved=await skillRequest(personal,`personal/${boundary.skillId}/versions/1`,s.skillProjectionSchema);expect(saved.version.package.description).toBe(description);
 const exported=await skillRequest(personal,`personal/${boundary.skillId}/versions/1/export`,s.skillPackageSchema);expect(exported.files?.[1].path).toBe(path);
 const imported=await mutate("personal",{files:exported.files,allowCopy:false,allowExport:true},s.skillReceiptSchema);
 const reread=await skillRequest(personal,`personal/${imported.skillId}/versions/1`,s.skillProjectionSchema);expect(reread.version.package.digest).toBe(exported.digest);
});
