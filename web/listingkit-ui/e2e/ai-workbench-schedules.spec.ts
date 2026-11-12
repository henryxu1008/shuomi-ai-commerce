import {createHash} from "node:crypto";
import {encode} from "@auth/core/jwt";
import AxeBuilder from "@axe-core/playwright";
import {test,expect} from "@playwright/test";
// Opt-in synthetic UI only; the real PostgreSQL/Temporal tests live in Go.
const secret="codex-scheduled-content-synthetic-ui-only";
test.skip(process.env.SCHEDULES_UI_FIXTURE!=="1"||process.env.AUTH_SECRET!==secret,"requires explicit isolated synthetic UI environment");
const id="11111111-1111-4111-8111-111111111111",taskID="22222222-2222-4222-8222-222222222222",at="2030-11-20T00:00:00Z";
test("confirms a Chat schedule, recovers a 503 with the same key, and manages future execution",async({page,context,baseURL},info)=>{
 test.setTimeout(120000);const origin=baseURL!,issuer="https://scheduled-ui.invalid",client="scheduled-ui-fixture";
 const namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32),name=`shuomi.${namespace}.session-token`;
 const token=await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-scheduled-ui",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"org",userId:"user",userType:"zitadel",roles:[]}}});
 await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"}]);
 const permissions=["workbench.task.read","workbench.agent.use","workbench.collection.read","local_agent.write"];
 const selection={subject:{kind:"own_product",id,catalogVersion:"1",publicationId:id},capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true}};
 const plan={id,revision:"2",state:"ACTIVE",registrationState:"MATCHED",detailsAvailable:true,goalSummary:"完善已保存商品的英语描述",selection,rule:{frequency:"once",timeZone:"Asia/Shanghai",atUtc:at,hour:0,minute:0,weekday:0},maxOccurrences:"1",consumedOccurrences:"0",unit:"ai-micropoint-v1",perRunUpperBoundMicros:"19200000",totalUpperBoundMicros:"19200000",nextAt:at,createdAt:"2026-11-12T00:00:00Z"};
 let created=false,creates=0,firstKey="",firstBody="",rejectNextControl=false;
 await page.route("**/*",async route=>{
  const req=route.request(),url=new URL(req.url());if(url.origin!==origin)return route.abort();
  if(!url.pathname.startsWith("/api/"))return route.continue();
  const json=(data:unknown,status=200)=>route.fulfill({status,contentType:"application/json",body:JSON.stringify(data)}),path=url.pathname;
  if(path==="/api/workbench/context")return json({user:{id:"user"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,aiWorkbenchAvailable:true,aiWorkbenchExecutionKind:"product.content.localize",aiWorkbenchExecutionReadiness:"AVAILABLE",organizations:[{id:"org",name:"定时任务隔离 UI",roles:["listingkit_admin"],permissions}]});
  if(path==="/api/workbench/tasks")return json({tasks:[],next:""});
  if(path==="/api/workbench/schedules/availability")return json({enabled:true,ready:true});
  if(path==="/api/workbench/schedules/quote"){const body=req.postDataJSON();return json({confirmation:"a".repeat(64),selection,goalSummary:plan.goalSummary,rule:body.rule,maxOccurrences:body.maxOccurrences,unit:plan.unit,perRunUpperBoundMicros:plan.perRunUpperBoundMicros,totalUpperBoundMicros:plan.totalUpperBoundMicros,nextAt:body.rule.atUtc??at,executionMode:"default"});}
  if(path==="/api/workbench/schedules"&&req.method()==="POST"){
   creates++;if(creates===1){firstKey=req.headers()["idempotency-key"];firstBody=req.postData()!;created=true;return json({code:"DEPENDENCY_UNAVAILABLE"},503);}
   expect(req.headers()["idempotency-key"]).toBe(firstKey);expect(req.postData()).toBe(firstBody);return json({plan,planSaved:true});
  }
  if(path==="/api/workbench/schedules")return json({plans:created?[plan]:[],next:""});
  if(path===`/api/workbench/schedules/${id}/occurrences`)return json({occurrences:[{id:taskID,scheduledAt:"2026-11-12T00:00:00Z",planRevision:"2",state:"UNKNOWN",reason:"ORIGINAL_RUN_OUTCOME_UNRESOLVED",businessTaskId:taskID}],next:""});
  if(path===`/api/workbench/schedules/${id}`&&req.method()==="GET")return json({plan});
  if(path.endsWith("/pause")||path.endsWith("/resume")){if(rejectNextControl){rejectNextControl=false;return json({code:"SCHEDULE_COMMAND_REJECTED",commandAccepted:false},409)}expect(req.postDataJSON().expectedRevision).toBe(plan.revision);plan.revision=String(Number(plan.revision)+2);plan.state=path.endsWith("/pause")?"PAUSED":"ACTIVE";return json({plan,planSaved:true});}
  if(path===`/api/workbench/schedules/${id}`&&req.method()==="PATCH"){plan.rule=req.postDataJSON().rule;plan.nextAt=plan.rule.atUtc;plan.revision=String(Number(plan.revision)+2);return json({plan,planSaved:true});}
  return json({code:"NOT_FOUND"},404);
 });
 await page.goto(`/workbench/ai/tasks?view=schedules&schedule_proposal=${id}`);
 await expect(page.getByRole("heading",{name:"将 Chat 方案设为定时任务"})).toBeVisible();
 await page.getByLabel("执行日期与时间（本机时区）").fill("2030-11-20T09:00");
 await page.getByRole("button",{name:"核对时间与预算"}).click();
 await expect(page.getByText("19.2 AI 积分").first()).toBeVisible();expect(creates).toBe(0);
 await page.screenshot({path:info.outputPath("schedule-confirm-desktop.png"),fullPage:true});
 await page.getByRole("button",{name:"确认并保存计划"}).click();await expect(page.getByRole("heading",{name:"上次操作结果待核实"})).toBeVisible();
 await page.reload();await page.getByRole("button",{name:"核实原操作"}).click();await expect(page.getByRole("heading",{name:plan.goalSummary})).toBeVisible();expect(creates).toBe(2);
 await expect(page.getByRole("link",{name:"查看原任务与人工审核"})).toHaveAttribute("href",`/workbench/ai/tasks/${taskID}`);
 await page.getByRole("button",{name:"暂停未来执行"}).click();await expect(page.getByRole("button",{name:"恢复未来执行"})).toBeVisible();
 await page.getByRole("button",{name:"恢复未来执行"}).click();await expect(page.getByRole("button",{name:"暂停未来执行"})).toBeVisible();
 await page.getByRole("button",{name:"修改时间"}).click();await page.getByLabel("执行日期与时间（本机时区）").fill("2030-11-21T10:00");await page.getByRole("button",{name:"核对时间与预算"}).click();await page.getByRole("button",{name:"确认修改时间"}).click();
 await expect(page.getByRole("heading",{name:plan.goalSummary})).toBeVisible();
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth)).toBeLessThanOrEqual(391);
 const accessibility=await new AxeBuilder({page}).include(".console-page").withTags(["wcag2a","wcag2aa"]).analyze();expect(accessibility.violations.filter(v=>v.impact==="critical"||v.impact==="serious")).toEqual([]);
 await page.screenshot({path:info.outputPath("schedule-history-mobile.png"),fullPage:true});
 rejectNextControl=true;await page.getByRole("button",{name:"暂停未来执行"}).click();
 await expect(page.locator(".console-page").getByRole("alert")).toContainText("已核实原操作没有提交");await expect(page.getByRole("button",{name:"暂停未来执行"})).toBeEnabled();await expect(page.getByRole("heading",{name:"上次操作结果待核实"})).toHaveCount(0);
});
