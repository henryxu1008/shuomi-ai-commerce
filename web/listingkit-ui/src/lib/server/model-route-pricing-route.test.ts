// @vitest-environment node
import {NextRequest} from "next/server";
import {afterEach,expect,it,vi} from "vitest";

const calls=vi.hoisted(()=>({proxy:vi.fn()}));
vi.mock("@/auth",()=>({serverAuth:(handler:(request:NextRequest & {auth:unknown})=>Promise<Response>)=>async(request:NextRequest)=>handler(Object.assign(request,{auth:{token:"synthetic-token"}}))}));
vi.mock("./zitadel-server-token",()=>({readZitadelServerAccessToken:()=>"synthetic-token"}));
vi.mock("./zitadel-auth",()=>({readZitadelIdentityFromSession:()=>({userId:"synthetic-user"})}));
vi.mock("./model-route-pricing-proxy",()=>({proxyModelRoutePricing:calls.proxy,routeFailure:(status:number,code:string)=>Response.json({code},{status})}));
import {handleModelRoutePricing} from "./model-route-pricing-route";

afterEach(()=>{calls.proxy.mockReset()});
it.each(["GET","POST"])("preserves %s fields from a framework-wrapped request",async method=>{
 calls.proxy.mockImplementation(async(request:NextRequest,token:string,user:string)=>Response.json({method:request.method,header:request.headers.get("X-Expected-Organization-ID"),body:await request.text(),token,user}));
 const incoming=new NextRequest("https://console.test/api/model-route-pricing/platform-prices/automatic-fx"+(method==="POST"?"/start":""),{method,headers:{"X-Expected-Organization-ID":"fixture-org","Content-Type":"application/json"},...(method==="POST"?{body:"{}"}:{})});
 const wrapped=new Proxy(incoming,{get(target,key){const value=Reflect.get(target,key,target);return typeof value==="function"?value.bind(target):value}});
 const response=await handleModelRoutePricing(wrapped);
 expect(response.status).toBe(200);expect(await response.json()).toEqual({method,header:"fixture-org",body:method==="POST"?"{}":"",token:"synthetic-token",user:"synthetic-user"});
});
