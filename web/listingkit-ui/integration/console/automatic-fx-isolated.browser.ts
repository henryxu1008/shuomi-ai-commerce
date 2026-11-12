import {createHash} from "node:crypto";
import {encode} from "@auth/core/jwt";
import {chromium,expect,test} from "@playwright/test";

// Synthetic browser sessions and a shared simulated backend only. The real
// Temporal concurrency and lost-ack tests run separately in a no-network container.
const secret="synthetic-fx-browser-only";
const issuer="https://fx-browser.invalid",client="fx-browser-fixture";
test.skip(process.env.FX_BROWSER_ISOLATED!=="1"||process.env.AUTH_SECRET!==secret||!process.env.FX_CHROME_PATH||!process.env.FX_EDGE_PATH,"requires the explicit isolated FX fixture and two browser executables");

test("two separate browsers do not create a second initialization after a lost response",async()=>{
 const origin="http://127.0.0.1:39211",namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32),name=`shuomi.${namespace}.session-token`;
 const token=await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-only",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"fixture-org",userId:"maintainer",userType:"zitadel",roles:[]}}});
 const chrome=await chromium.launch({executablePath:process.env.FX_CHROME_PATH,headless:true}),edge=await chromium.launch({executablePath:process.env.FX_EDGE_PATH,headless:true});
 const contexts=await Promise.all([chrome.newContext(),edge.newContext()]);let initialized=false,starts=0,commits=0;
 try{
  const pages=await Promise.all(contexts.map(async context=>{
   await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"}]);const page=await context.newPage();
   await page.route("**/*",async route=>{const req=route.request(),url=new URL(req.url());if(url.origin!==origin)return route.abort();if(!url.pathname.startsWith("/api/"))return route.continue();
    const json=(value:unknown,status=200)=>route.fulfill({status,contentType:"application/json",body:JSON.stringify(value)});
    if(url.pathname==="/api/workbench/context")return json({user:{id:"maintainer"},homeOrganizationId:"fixture-org",effectiveOrganizationId:"fixture-org",effectiveMemberId:"member",selectionRequired:false,organizations:[{id:"fixture-org",name:"合成测试企业",roles:["listingkit_admin"],permissions:["listingkit.platform_admin","listingkit.admin.write"]}]});
    if(url.pathname==="/api/model-route-pricing/platform-prices")return json({status:"UNCONFIGURED",revision:null,preview:null});
    if(url.pathname==="/api/model-route-pricing/platform-prices/history")return json({revisions:[],nextCursor:""});
    if(url.pathname==="/api/model-route-pricing/platform-prices/automatic-fx")return initialized?json({code:"OUTCOME_UNKNOWN"},503):json({state:"PREPARED",checkedAt:"2026-11-12T00:00:00Z",stopRequested:false});
    if(url.pathname==="/api/model-route-pricing/platform-prices/automatic-fx/start"&&req.method()==="POST"){
     starts++;if(!initialized){initialized=true;commits++;return route.abort("failed")}return json({code:"OUTCOME_UNKNOWN"},503);
    }
    return json({code:"NOT_FOUND"},404);
   });return page;
  }));
  await Promise.all(pages.map(p=>p.goto(origin+"/workbench/account/organization/model-route-pricing")));
  for(const p of pages)await expect(p.getByRole("button",{name:"启用每周自动更新"})).toBeVisible();
  await pages[0].getByRole("button",{name:"启用每周自动更新"}).click();await expect(pages[0].getByText(/启用结果待核实/)).toBeVisible();
  await pages[1].getByRole("button",{name:"启用每周自动更新"}).click();await expect(pages[1].getByText(/启用结果待核实/)).toBeVisible();
  expect(starts).toBe(1);expect(commits).toBe(1);
  await Promise.all(pages.map(p=>p.reload()));
  for(const p of pages){await expect(p.getByText(/未启用或状态未知/)).toBeVisible();await expect(p.getByRole("button",{name:"启用每周自动更新"})).toHaveCount(0)}
  expect(starts).toBe(1);expect(commits).toBe(1);
 }finally{await Promise.all(contexts.map(c=>c.close()));await Promise.all([chrome.close(),edge.close()])}
});
