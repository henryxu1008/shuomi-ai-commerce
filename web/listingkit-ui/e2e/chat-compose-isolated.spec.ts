import { createHash } from "node:crypto";
import { encode } from "@auth/core/jwt";
import { expect, test } from "@playwright/test";

const secret = "codex-chat-compose-synthetic-ui-only";
test.skip(process.env.CHAT_COMPOSE_UI_FIXTURE !== "1" || process.env.AUTH_SECRET !== secret, "requires the explicit isolated synthetic UI fixture");
test("retains the input through the original create retry without dispatching a message", async ({page,context,baseURL},info) => {
  const origin=baseURL!,issuer="https://chat-compose.invalid",client="chat-compose-fixture";
  await page.addInitScript(() => {
    const target = window as unknown as {SpeechRecognition: unknown; webkitSpeechRecognition: unknown; voiceFixture: {starts: number; current?: {onresult?: (event: unknown) => void}}};
    target.voiceFixture = {starts:0};
    class FakeSpeechRecognition {
      onresult?: (event: unknown) => void;
      onend?: () => void;
      constructor() { target.voiceFixture.current = this; }
      start() { target.voiceFixture.starts++; }
      stop() { setTimeout(() => this.onend?.(), 0); }
      abort() {}
    }
    target.SpeechRecognition = FakeSpeechRecognition;
    target.webkitSpeechRecognition = FakeSpeechRecognition;
  });
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
      await expect(page.getByRole("button",{name:"开始对话",exact:true})).toBeVisible();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
      await page.screenshot({path:info.outputPath(`chat-compose-${viewport.width}-${theme}.png`)});
    }
  }
  await page.getByLabel("需求").fill("【合成测试】请整理商品描述");
  expect(await page.evaluate(() => (window as unknown as {voiceFixture:{starts:number}}).voiceFixture.starts)).toBe(0);
  await page.getByRole("button",{name:"开始语音输入"}).click();
  await page.evaluate(() => {
    const fixture = (window as unknown as {voiceFixture:{current:{onresult:(event:unknown)=>void}}}).voiceFixture;
    const item = Object.assign([{transcript:"，补充卖点"}], {isFinal:true});
    fixture.current.onresult({resultIndex:0,results:[item]});
  });
  await expect(page.getByLabel("需求")).toHaveValue("【合成测试】请整理商品描述，补充卖点");
  await page.getByRole("button",{name:"停止语音输入"}).click();
  await page.getByLabel("需求").fill("【合成测试】请整理商品描述");
  await page.screenshot({path:info.outputPath("chat-compose-desktop.png")});
  await page.getByRole("button",{name:"开始对话",exact:true}).click();
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

test("routes a store question once and hides cached totals after revoked access",async({page,context,baseURL})=>{
 const origin=baseURL!,issuer="https://chat-compose.invalid",client="chat-compose-fixture";
 const namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32),name=`shuomi.${namespace}.session-token`;
 const token=await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-chat-compose",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"org",userId:"user",userType:"zitadel",roles:[]}}});
 await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"}]);
 const id="11111111-1111-4111-8111-111111111111",messageId="22222222-2222-4222-8222-222222222222",stamp="2026-11-12T00:00:00Z";
 const conversation={ID:id,Scope:{OrganizationID:"org",ActorID:"user"},Title:"",Favorite:false,Archived:false,MetadataRevision:1,NextSequence:1,CreatedAt:stamp,UpdatedAt:stamp,businessScope:{version:"workbench-scope-v1",kind:"general"}};
 let intentCalls=0,modelKey="",revoked=false,answered=false;
 await page.route("**/*",async route=>{
  const req=route.request(),url=new URL(req.url());if(url.origin!==origin)return route.abort();if(!url.pathname.startsWith("/api/"))return route.continue();
  const json=(data:unknown,status=200)=>route.fulfill({status,contentType:"application/json",body:JSON.stringify(data)}),path=url.pathname;
  if(path==="/api/workbench/context")return json({user:{id:"user"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,aiWorkbenchAvailable:true,aiWorkbenchExecutionKind:"product.content.localize",aiWorkbenchExecutionReadiness:"AVAILABLE",aiWorkbenchPlanningReadiness:"AVAILABLE",aiWorkbenchStoreRouteReadiness:"AVAILABLE",organizations:[{id:"org",name:"合成测试企业",roles:["listingkit_admin"],permissions:["workbench.chat.use","workbench.chat.read","workbench.store.read"],capabilities:{"workbench.source_account.manage":false,"workbench.chat.use":true}}]});
  if(path===`/api/workbench/chat/conversations/${id}`&&req.method()==="GET"){
   const ref={ID:messageId,ConversationID:id,Sequence:2,Author:"ASSISTANT",Content:"店铺概况",CreatedAt:stamp,kind:"STORE_OVERVIEW_REF",storeQueryStatus:"ALL_NON_DELETED",...(revoked?{storeOverviewUnavailable:true}:{currentStoreOverview:{total:3,platforms:{shein:2,temu:1},statuses:{active:3},operatingTypeUnknown:3,readStartedAt:stamp}})};
   return json({conversation,messages:answered?[ref]:[],proposals:[],before:""});
  }
  if(path===`/api/workbench/chat/conversations/${id}/intent`&&req.method()==="POST"){
   intentCalls++;if(intentCalls===1){modelKey=req.headers()["idempotency-key"];return json({code:"OUTCOME_UNKNOWN"},503)}
   expect(req.headers()["idempotency-key"]).toBe(modelKey);expect(req.postDataJSON()).toEqual({content:"我有几个店铺？"});answered=true;return json({state:"COMPLETE"});
  }
  return json({code:"NOT_FOUND"},404);
 });
 await page.goto(`/workbench/ai/chat/${id}`);await page.getByLabel("需求").fill("我有几个店铺？");expect(intentCalls).toBe(0);
 await page.getByRole("button",{name:"发送消息"}).click();await page.getByRole("button",{name:"重试同一消息"}).click();
 await expect(page.getByText("当前可查看 3 家店铺。")).toBeVisible();expect(intentCalls).toBe(2);
 revoked=true;await page.evaluate(()=>{window.dispatchEvent(new Event("blur"));window.dispatchEvent(new Event("focus"));});
 await expect(page.getByText("当前可查看 3 家店铺。")).toHaveCount(0);await expect(page.getByText("当前无权查看或店铺数据暂不可用。")).toBeVisible();
});
