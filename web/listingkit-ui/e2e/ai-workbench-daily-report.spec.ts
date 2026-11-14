import {createHash} from "node:crypto";
import {readFile} from "node:fs/promises";
import {join} from "node:path";
import {encode} from "@auth/core/jwt";
import {test,expect} from "@playwright/test";

test.skip(!process.env.SCHEDULED_REPORT_BROWSER_FIXTURE_DIR,"requires the existing task-owned Go/PG fixture");
test("saves single-store reports through the real BFF and backend, then reopens after refresh",async({page,context,request,baseURL},info)=>{
 test.setTimeout(180000);
 const seed=JSON.parse(await readFile(join(process.env.SCHEDULED_REPORT_BROWSER_FIXTURE_DIR!,"go.json"),"utf8"));
 const origin=baseURL!,issuer="https://scheduled-ui.invalid",client="scheduled-ui-fixture",secret=process.env.AUTH_SECRET!;
 const namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32),name=`shuomi.${namespace}.session-token`;
 const token=await encode({secret,salt:name,maxAge:1200,token:{accessToken:"synthetic",expiresAt:Math.floor(Date.now()/1000)+1200,identityVersion:3,identity:{tenantId:"daily-org",userId:"daily-user",userType:"zitadel",roles:[]}}});
 await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"},{name:"shuomi_effective_organization",value:"daily-org",url:origin,httpOnly:true,sameSite:"Lax"}]);
 const permissions=["workbench.task.read","workbench.agent.use","workbench.store.read","workbench.cockpit.stores.read","workbench.report.read","workbench.report.manage"];
 const at=new Date().toISOString(),metric={value:null,unknown:1,complete:false,reason:"not_synced"};
 const store={id:seed.storeId,name:seed.storeName,platform:"shein",region:"US",externalStoreId:"",recordStatus:"active",serviceStatus:"pending_activation",serviceStartedAt:null,serviceExpiresAt:null,connectionStatus:"disconnected",version:1,createdAt:at,updatedAt:at,profile:{storeId:seed.storeId,groupId:null,responsibleMemberId:null,lowStockThreshold:null,profileVersion:0,updatedAt:null},attention:"unknown",signals:{connectionStatus:"disconnected",applicationType:"",products:metric,orders:metric,inventory:metric,productSync:"",orderSync:"",syncHealthy:false,observedAt:null,reason:"not_synced"}};
 // Synthetic session/context and the directory presentation use the same
 // canonical fixture Store ID. Every schedule/report request below goes
 // through the real BFF and live backend authorization; no business API mock.
 await page.route("**/*",async route=>{
  const url=new URL(route.request().url());if(url.origin!==origin)return route.abort();
  if(url.pathname==="/api/workbench/context")return route.fulfill({json:{user:{id:"daily-user"},homeOrganizationId:"daily-org",effectiveOrganizationId:"daily-org",effectiveMemberId:"daily-member",selectionRequired:false,aiWorkbenchAvailable:true,aiWorkbenchExecutionKind:"product.content.localize",aiWorkbenchExecutionReadiness:"UNAVAILABLE",organizations:[{id:"daily-org",name:"单店日报合成验证",roles:[],permissions}]}});
  if(url.pathname==="/api/workbench/store-center/catalog")return route.fulfill({json:{schemaVersion:"store-center-v1",organizationId:"daily-org",userId:"daily-user",data:{items:[store],pagination:{page:1,pageSize:100,total:1},metrics:{total:1,normal:null,attention:null,syncHealthy:null,unknown:1,complete:false,observedAt:at},facets:{regions:["US"],applicationTypes:[],groups:[],canManageGroups:false}}}});
  return route.continue();
 });
 let creates=0;page.on("request",req=>{if(new URL(req.url()).pathname==="/api/workbench/schedules"&&req.method()==="POST")creates++});
 for(const frequency of ["once","daily"] as const){
  await page.goto("/workbench/ai/tasks?view=schedules");
  await page.getByRole("button",{name:"创建定时任务",exact:true}).click();
  await page.getByRole("combobox",{name:"店铺",exact:true}).selectOption(seed.storeId);
  if(frequency==="once")await page.getByLabel("执行日期与时间（本机时区）").fill(new Date(Date.now()+120000).toLocaleString("sv-SE").slice(0,16).replace(" ","T"));
  else {await page.getByRole("combobox",{name:"频率",exact:true}).selectOption("daily");await page.getByLabel("最多执行次数").fill("2");await page.getByLabel("执行时刻（计划时区）").fill(new Date(Date.now()+120000).toLocaleTimeString("en-GB",{timeZone:"Asia/Shanghai",hour:"2-digit",minute:"2-digit"}));}
  const before=creates;await page.getByRole("button",{name:"核对计划",exact:true}).click();await expect(page.getByText("每次执行自动保存一份个人报告")).toBeVisible();expect(creates).toBe(before);
  const response=page.waitForResponse(r=>new URL(r.url()).pathname==="/api/workbench/schedules"&&r.request().method()==="POST");await page.getByRole("button",{name:"确认并保存计划"}).click();const saved=await response;expect(saved.status()).toBe(200);const body=await saved.json();const id=body.plan.id;expect(creates).toBe(before+1);
  const dispatch=await request.post(seed.controlOrigin+"/dispatch",{data:{planId:id}});expect(dispatch.status()).toBe(204);
  await page.reload();const reportLink=page.getByRole("link",{name:"查看经营日报",exact:true});await expect(reportLink).toBeVisible();const href=await reportLink.getAttribute("href");expect(href).toMatch(/reportId=/);
  await reportLink.click();await expect(page.getByText("0.00 元").first()).toBeVisible();await page.reload();await expect(page.getByText("0.00 元").first()).toBeVisible();expect(page.url()).toContain(href!.split("?")[1]);
  const download=page.waitForEvent("download");await page.getByRole("button",{name:"下载文本",exact:true}).click();expect((await download).suggestedFilename()).toMatch(/\.txt$/);
  await page.screenshot({path:info.outputPath(`store-report-${frequency}.png`),fullPage:true});
  if(frequency==="once")await page.setViewportSize({width:390,height:844});
 }
 expect(creates).toBe(2);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth)).toBeLessThanOrEqual(391);
});
