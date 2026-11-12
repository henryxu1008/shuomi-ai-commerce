import { createHash } from "node:crypto";
import { encode } from "@auth/core/jwt";
import { expect, test } from "@playwright/test";

const secret = "codex-chat-compose-synthetic-ui-only";
test.skip(process.env.CHAT_COMPOSE_UI_FIXTURE !== "1" || process.env.AUTH_SECRET !== secret, "requires the explicit isolated synthetic UI fixture");
test("retains the input through the original create retry without dispatching a message", async ({page,context,baseURL},info) => {
  const origin=baseURL!,issuer="https://chat-compose.invalid",client="chat-compose-fixture";
  const namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32),name=`shuomi.${namespace}.session-token`;
  const token=await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-chat-compose",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"org",userId:"user",userType:"zitadel",roles:[]}}});
  await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"}]);
  const id="11111111-1111-4111-8111-111111111111",confirmation="a".repeat(64);
  const conversation={ID:id,Scope:{OrganizationID:"org",ActorID:"user"},Title:"",Favorite:false,Archived:false,MetadataRevision:1,NextSequence:1,CreatedAt:"2026-11-12T00:00:00Z",UpdatedAt:"2026-11-12T00:00:00Z",businessScope:{version:"workbench-scope-v1",kind:"general"}};
  let creates=0,firstKey="",firstBody="",messages=0;
  await page.route("**/*",async route=>{
    const req=route.request(),url=new URL(req.url()); if(url.origin!==origin)return route.abort();
    if(!url.pathname.startsWith("/api/"))return route.continue();
    const json=(data:unknown,status=200)=>route.fulfill({status,contentType:"application/json",body:JSON.stringify(data)}),path=url.pathname;
    if(path==="/api/workbench/context")return json({user:{id:"user"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,aiWorkbenchAvailable:true,aiWorkbenchExecutionKind:"product.content.localize",aiWorkbenchExecutionReadiness:"AVAILABLE",aiWorkbenchPlanningReadiness:"AVAILABLE",organizations:[{id:"org",name:"合成测试企业",roles:["listingkit_admin"],permissions:["workbench.chat.use","workbench.chat.read"],capabilities:{"workbench.source_account.manage":false,"workbench.chat.use":true}}]});
    if(path==="/api/workbench/chat/conversations/scopes") {const kind=url.searchParams.get("kind")!,projectId=url.searchParams.get("projectId"),storeId=url.searchParams.get("storeId");return json({options:[{scope:{version:"workbench-scope-v1",kind,...(projectId?{projectId}:{}),...(storeId?{storeId}:{})},confirmation,...(projectId?{projectName:"合成项目",storeName:"合成关联店铺"}:storeId?{storeName:"合成店铺"}:{})}],next:"",incomplete:false});}
    if(path==="/api/workbench/chat/conversations"&&req.method()==="POST"){creates++;if(creates===1){firstKey=req.headers()["idempotency-key"];firstBody=req.postData()!;return json({code:"OUTCOME_UNKNOWN"},503);}expect(req.headers()["idempotency-key"]).toBe(firstKey);expect(req.postData()).toBe(firstBody);return json({conversation,replay:true});}
    if(path===`/api/workbench/chat/conversations/${id}`&&req.method()==="GET")return json({conversation,messages:[],proposals:[],before:""});
    if(path.endsWith("/messages")){messages++;return json({code:"NOT_FOUND"},404);}
    return json({code:"NOT_FOUND"},404);
  });
  await page.goto("/workbench/ai/chat/new");
  for (const viewport of [{width:1440,height:900},{width:390,height:844}]) {
    await page.setViewportSize(viewport);
    for (const theme of ["light", "dark"]) {
      await page.evaluate(value => localStorage.setItem("listingkit-theme", value), theme);
      await page.reload();
      await expect(page.locator("html")).toHaveClass(new RegExp(theme));
      await expect(page.getByLabel("需求")).toBeVisible();
      await expect(page.getByRole("button",{name:"创建会话",exact:true})).toBeVisible();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
      await page.screenshot({path:info.outputPath(`chat-compose-${viewport.width}-${theme}.png`)});
    }
  }
  await page.getByLabel("需求").fill("【合成测试】请整理商品描述");
  await page.screenshot({path:info.outputPath("chat-compose-desktop.png")});
  await page.getByRole("button",{name:"创建会话",exact:true}).click();
  await expect(page.getByRole("button",{name:"核对原创建请求"})).toBeVisible();
  await page.reload(); await expect(page.getByLabel("需求")).toHaveValue("【合成测试】请整理商品描述");
  await page.getByRole("button",{name:"核对原创建请求"}).click();
  await page.waitForURL(`**/workbench/ai/chat/${id}`);
  await expect(page.getByLabel("需求")).toHaveValue("【合成测试】请整理商品描述");
  expect(creates).toBe(2); expect(messages).toBe(0);
  await page.setViewportSize({width:390,height:844});
  await page.goto("/workbench/ai/chat/new?projectId=fixture-project");
  await expect(page.getByText("工作背景：合成项目 · 合成关联店铺")).toBeVisible();
  await page.screenshot({path:info.outputPath("chat-compose-project-mobile.png")});
  await page.goto("/workbench/ai/chat/new?storeId=fixture-store");
  await expect(page.getByText("工作背景：合成店铺")).toBeVisible();
  expect(messages).toBe(0); expect(creates).toBe(2);
});
