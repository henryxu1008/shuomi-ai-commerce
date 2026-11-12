import {createHash} from "node:crypto";
import {encode} from "@auth/core/jwt";
import {chromium,expect,test} from "@playwright/test";

// Opt-in only. Identity/grants and the FX source are synthetic; the Next BFF,
// mounted Go HTTP routes, authorization checks and isolated Temporal are real.
const secret=process.env.FX_CHAIN_SECRET??"",issuer="https://fx-chain.invalid",client="fx-chain-fixture";
test.skip(process.env.FX_CHAIN_BROWSER!=="1"||!secret||process.env.AUTH_SECRET!==secret||!process.env.FX_CHROME_PATH||!process.env.FX_EDGE_PATH,"requires the dedicated isolated FX chain fixture");

test("two browsers use the actual BFF and Temporal after the first response is lost",async()=>{
 const origin="http://127.0.0.1:39211",namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32),name=`shuomi.${namespace}.session-token`;
 const token=await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-only",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"fixture-org",userId:"maintainer",userType:"zitadel",roles:[]}}});
 const chrome=await chromium.launch({executablePath:process.env.FX_CHROME_PATH,headless:true}),edge=await chromium.launch({executablePath:process.env.FX_EDGE_PATH,headless:true});
 const contexts=await Promise.all([chrome.newContext(),edge.newContext()]);let starts=0,actualBFFStatus=0;
 try{
  const pages=await Promise.all(contexts.map(async context=>{
   await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"},{name:"shuomi_effective_organization",value:"fixture-org",url:origin,sameSite:"Lax"}]);
   // Verify the listener can decrypt this run's randomly keyed synthetic
   // session before sending any state-changing request to it.
   const session=await context.request.get(origin+"/api/auth/session");expect(session.status()).toBe(200);expect((await session.json()).identity?.userId).toBe("maintainer");
   const page=await context.newPage();await page.route("**/*",async route=>{const url=new URL(route.request().url());if(url.origin!==origin)return route.abort();
    if(url.pathname==="/api/workbench/context")return route.fulfill({json:{user:{id:"maintainer"},homeOrganizationId:"fixture-org",effectiveOrganizationId:"fixture-org",effectiveMemberId:"fixture-member",selectionRequired:false,organizations:[{id:"fixture-org",name:"合成测试企业",roles:["listingkit_admin"],permissions:["listingkit.platform_admin","listingkit.admin.write"]}]}});
    return route.continue();
   });return page;
  }));
  const prepared=await contexts[0].request.post(origin+"/api/model-route-pricing/platform-prices/automatic-fx/prepare",{headers:{Origin:origin,"Sec-Fetch-Site":"same-origin","X-Expected-User-ID":"maintainer","X-Expected-Organization-ID":"fixture-org"},data:{}});
  expect(prepared.status(),await prepared.text()).toBe(200);expect(await prepared.json()).toEqual({state:"PREPARED"});
  await Promise.all(pages.map(p=>p.goto(origin+"/workbench/account/organization/model-route-pricing")));
  for(const p of pages)await expect(p.getByRole("button",{name:"启用每周自动更新"})).toBeVisible();
  await pages[0].route("**/api/model-route-pricing/platform-prices/automatic-fx/start",async route=>{starts++;const response=await route.fetch();actualBFFStatus=response.status();expect(await response.json()).toEqual({state:"STARTED"});await route.abort("failed")});
  await pages[1].route("**/api/model-route-pricing/platform-prices/automatic-fx/start",async route=>{starts++;await route.continue()});
  await pages[0].getByRole("button",{name:"启用每周自动更新"}).click();await expect(pages[0].getByText(/启用结果待核实/)).toBeVisible();expect(actualBFFStatus).toBe(200);
  await pages[1].getByRole("button",{name:"启用每周自动更新"}).click();await expect(pages[1].getByText(/启用结果待核实/)).toBeVisible();expect(starts).toBe(1);
  await Promise.all(pages.map(p=>p.reload()));for(const p of pages)await expect(p.getByRole("button",{name:"启用每周自动更新"})).toHaveCount(0);
  expect(starts).toBe(1);
 }finally{await Promise.all(contexts.map(c=>c.close()));await Promise.all([chrome.close(),edge.close()])}
});
