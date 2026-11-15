import {createHash} from "node:crypto";
import {encode} from "@auth/core/jwt";
import AxeBuilder from "@axe-core/playwright";
import {test,expect} from "@playwright/test";
// Opt-in synthetic UI only; the real PostgreSQL/Temporal tests live in Go.
// Fixture dev server: AUTH_URL must match baseURL; LISTINGKIT_REPORT_CENTER_ENABLED=true.
const secret="codex-scheduled-content-synthetic-ui-only";
test.skip(process.env.SCHEDULES_UI_FIXTURE!=="1"||process.env.AUTH_SECRET!==secret,"requires explicit isolated synthetic UI environment");
const id="11111111-1111-4111-8111-111111111111",taskID="22222222-2222-4222-8222-222222222222",at="2030-11-20T00:00:00Z";
test("creates only store reports while preserving existing content plans and recovery",async({page,context,baseURL},info)=>{
 test.setTimeout(120000);const origin=baseURL!,issuer="https://scheduled-ui.invalid",client="scheduled-ui-fixture";
 const settleTheme=()=>page.evaluate(async()=>{await Promise.all(document.getAnimations().filter(a=>a instanceof CSSTransition).map(a=>a.finished.catch(()=>{})))});
 const namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32),name=`shuomi.${namespace}.session-token`;
 const token=await encode({secret,salt:name,maxAge:600,token:{accessToken:"synthetic-scheduled-ui",expiresAt:Math.floor(Date.now()/1000)+600,identityVersion:3,identity:{tenantId:"org",userId:"user",userType:"zitadel",roles:[]}}});
 await context.addCookies([{name,value:token,url:origin,httpOnly:true,sameSite:"Lax"}]);
 const permissions=["workbench.report.read","workbench.task.read","workbench.agent.use","workbench.collection.read","local_agent.write","workbench.store.read","workbench.cockpit.stores.read"];
 const metric={value:null,unknown:1,complete:false,reason:"not_synced"};
 const store={id,name:"合成测试店",platform:"shein",region:"US",externalStoreId:"",recordStatus:"active",serviceStatus:"pending_activation",serviceStartedAt:null,serviceExpiresAt:null,connectionStatus:"disconnected",version:1,createdAt:at,updatedAt:at,profile:{storeId:id,groupId:null,responsibleMemberId:null,lowStockThreshold:null,profileVersion:0,updatedAt:null},attention:"unknown",signals:{connectionStatus:"disconnected",applicationType:"",products:metric,orders:metric,inventory:metric,productSync:"",orderSync:"",syncHealthy:false,observedAt:null,reason:"not_synced"}};
 const selection={subject:{kind:"own_product",id,catalogVersion:"1",publicationId:id},capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true}};
 const plan={id,deleted:false,revision:"2",state:"ACTIVE",registrationState:"MATCHED",detailsAvailable:true,goalSummary:"完善已保存商品的英语描述",selection,rule:{frequency:"once",timeZone:"Asia/Shanghai",atUtc:at,hour:0,minute:0,weekday:0},maxOccurrences:"1",consumedOccurrences:"0",unit:"ai-micropoint-v1",perRunUpperBoundMicros:"19200000",totalUpperBoundMicros:"19200000",nextAt:at,createdAt:"2026-11-12T00:00:00Z"};
 const originalTask={id:taskID,conversationId:"",proposalId:"",scheduleId:id,scheduledOccurrenceId:taskID,title:"定时内容原任务",goalSummary:plan.goalSummary,createdAt:"2026-11-12T00:00:00Z",kind:"product.content.localize",projectionAvailable:true,state:"WAITING_CONFIRMATION",canStart:false,canReconcile:false,canResume:false,canReview:false,productDetailsAvailable:false};

 const reportID="33333333-3333-4333-8333-333333333333",copyID="44444444-4444-4444-8444-444444444444";
 const report={...plan,id:reportID,kind:"store.operating-daily-report",selection:undefined,report:{storeId:id,version:1,periodPolicy:"previous_natural_day",businessZone:"Asia/Shanghai"},storeName:store.name,goalSummary:"店铺经营日报",unit:"none",perRunUpperBoundMicros:"0",totalUpperBoundMicros:"0",rule:{...plan.rule,frequency:"daily"}};
 let quoteUnavailable=false,created=false,creates=0,quotes=0,edits=0,firstKey="",firstBody="",editKey="",editBody="";
 let copied:typeof report|undefined,reportSaved=false,reportReadable=true;
 const savedReportId="66666666-6666-4666-8666-666666666666";
 const savedSummary={id:savedReportId,ref:{kind:"STORE_DAILY_REPORT",id:taskID,version:"1"},title:"合成测试店 · 经营日报",productKey:"",storeId:id,capturedAt:at,favorite:false,scheduledOrigin:{organizationId:"org",actorId:"user",memberId:"member",planId:reportID,planRevision:2,occurrenceId:taskID,scheduledAt:at,storeId:id,day:"2030-11-19",rendererVersion:1}};
 await page.route("**/*",async route=>{
  const req=route.request(),url=new URL(req.url());if(url.origin!==origin)return route.abort();
  if(!url.pathname.startsWith("/api/"))return route.continue();
  const json=(data:unknown,status=200)=>route.fulfill({status,contentType:"application/json",body:JSON.stringify(data)}),path=url.pathname;
  if(path==="/api/workbench/context")return json({user:{id:"user"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,aiWorkbenchAvailable:true,aiWorkbenchExecutionKind:"product.content.localize",aiWorkbenchExecutionReadiness:"AVAILABLE",organizations:[{id:"org",name:"定时任务隔离 UI",roles:["listingkit_admin"],permissions}]});
  if(path==="/api/workbench/tasks")return json({tasks:[originalTask],next:""});
  if(path===`/api/workbench/tasks/${taskID}`)return json({task:originalTask});
  if(path==="/api/workbench/collections/batches")return json({items:[],total:0});
  if(path==="/api/workbench/store-center/catalog")return json({schemaVersion:"store-center-v1",organizationId:"org",userId:"user",data:{items:[store],pagination:{page:1,pageSize:100,total:1},metrics:{total:1,normal:null,attention:null,syncHealthy:null,unknown:1,complete:false,observedAt:at},facets:{regions:["US"],applicationTypes:[],groups:[],canManageGroups:false}}});
  if(path===`/api/report-center/reports/${savedReportId}`)return json({...savedSummary,content:{schemaVersion:1,sections:[{title:"经营日报",fields:[{label:"资料状态",value:"资料不全"}]}]},digest:"a".repeat(64)});
  if(path==="/api/report-center/reports/summary")return json({saved:1,recent:1,favorites:0,stores:1});
  if(path==="/api/report-center/reports")return json({items:[savedSummary],nextCursor:""});
  if(path==="/api/workbench/schedules/availability")return json({enabled:true,ready:true,contentEnabled:true,storeDailyReportEnabled:true});
  if(path==="/api/workbench/schedules/quote"){
   quotes++;const body=req.postDataJSON();if(!body.scheduleId)expect(body.kind).toBe("store.operating-daily-report");
   if(quoteUnavailable)return json({code:"DEPENDENCY_UNAVAILABLE"},503);
   return json({confirmation:"a".repeat(64),consumedOccurrences:"0",...(body.kind==="store.operating-daily-report"?{kind:body.kind,report:body.report,storeName:store.name,unit:"none",perRunUpperBoundMicros:"0",totalUpperBoundMicros:"0"}:{selection:body.selection??selection,unit:plan.unit,perRunUpperBoundMicros:plan.perRunUpperBoundMicros,totalUpperBoundMicros:plan.totalUpperBoundMicros,executionMode:"default"}),goalSummary:body.goalSummary??report.goalSummary,rule:body.rule,maxOccurrences:body.maxOccurrences,nextAt:body.rule.atUtc??at});
  }
  if(path==="/api/workbench/schedules"&&req.method()==="POST"){
   creates++;const body=req.postDataJSON();expect(body.kind).toBe("store.operating-daily-report");
   if(body.sourceScheduleId){expect(body.sourceScheduleId).toBe(reportID);expect(req.headers()["idempotency-key"]).not.toBe(firstKey);copied={...report,id:copyID,state:"ACTIVE",revision:"2",rule:body.rule,consumedOccurrences:"0"};return json({plan:copied,planSaved:true});}
   if(creates===1){firstKey=req.headers()["idempotency-key"];firstBody=req.postData()!;created=true;report.rule=body.rule;report.maxOccurrences=body.maxOccurrences;return json({code:"DEPENDENCY_UNAVAILABLE"},503);}
   expect(req.headers()["idempotency-key"]).toBe(firstKey);expect(req.postData()).toBe(firstBody);return json({plan:report,planSaved:true});
  }
  if(path==="/api/workbench/schedules")return json({plans:[plan,...(created?[report]:[]),...(copied?[copied]:[])],next:""});
  if(path===`/api/workbench/schedules/${reportID}`)return json({plan:report});
  if(path===`/api/workbench/schedules/${copyID}`)return json({plan:copied});
  if(path===`/api/workbench/schedules/${reportID}/occurrences`)return json({occurrences:reportSaved?[{id:taskID,scheduledAt:at,planRevision:"2",state:"RECONCILED",reason:"REPORT_SAVED",...(reportReadable?{reportId:savedReportId}:{})}]:[],next:""});
  if(path===`/api/workbench/schedules/${copyID}/occurrences`)return json({occurrences:[],next:""});
  if(path===`/api/workbench/schedules/${id}/occurrences`)return json({occurrences:[{id:taskID,scheduledAt:"2026-11-12T00:00:00Z",planRevision:"2",state:"UNKNOWN",reason:"ORIGINAL_RUN_OUTCOME_UNRESOLVED",businessTaskId:taskID}],next:""});
  if(path===`/api/workbench/schedules/${id}`&&req.method()==="GET")return json({plan});
  if(path.endsWith("/pause")||path.endsWith("/resume")){expect(req.postDataJSON().expectedRevision).toBe(plan.revision);plan.revision=String(Number(plan.revision)+2);plan.state=path.endsWith("/pause")?"PAUSED":"ACTIVE";return json({plan,planSaved:true});}
  if(path===`/api/workbench/schedules/${id}`&&req.method()==="PATCH"){
   if(++edits===1){editKey=req.headers()["idempotency-key"];editBody=req.postData()!;const body=req.postDataJSON();plan.goalSummary=body.changes.goalSummary;plan.selection=body.changes.selection;plan.rule=body.rule;plan.revision=String(Number(plan.revision)+2);return json({code:"DEPENDENCY_UNAVAILABLE"},503);}
   expect(req.headers()["idempotency-key"]).toBe(editKey);expect(req.postData()).toBe(editBody);return json({plan,planSaved:true});
  }
  return json({code:"NOT_FOUND"},404);
 });
 const originalPlan=JSON.stringify(plan);
 await page.goto(`/workbench/ai/tasks?view=schedules&schedule_proposal=${id}`);
 await expect(page.getByText("商品内容不再提供新建定时计划，已有计划和执行记录仍可查看")).toBeVisible();
 expect(quotes).toBe(0);expect(creates).toBe(0);expect(JSON.stringify(plan)).toBe(originalPlan);
 await expect(page.getByRole("button",{name:"核对时间与预算"})).toHaveCount(0);
 await page.getByRole("button",{name:"返回定时任务",exact:true}).click();
 await page.getByRole("button",{name:"创建定时任务",exact:true}).click();
 await expect(page.getByRole("combobox",{name:"任务类型",exact:true})).toHaveCount(0);
 await expect(page.getByRole("textbox",{name:"任务目标",exact:true})).toHaveCount(0);
 await page.getByRole("combobox",{name:"店铺",exact:true}).selectOption(id);
 await page.getByRole("combobox",{name:"频率",exact:true}).selectOption("daily");
 await expect(page.getByLabel("最多执行次数",{exact:true})).toHaveValue("30");
 await page.getByRole("button",{name:"最多7次",exact:true}).click();await expect(page.getByLabel("最多执行次数")).toHaveValue("7");
 await page.getByRole("button",{name:"最多30次",exact:true}).click();await page.getByLabel("最多执行次数",{exact:true}).fill("3");
 await page.getByLabel("截止时间（可选，本机时区）",{exact:true}).fill("2030-12-31T23:00");
 for(const width of [1440,1920,390])for(const theme of ["light","dark"]){
  await page.setViewportSize({width,height:900});await page.getByRole("combobox",{name:"显示主题"}).selectOption(theme);await settleTheme();
  await expect(page.getByRole("combobox",{name:"店铺",exact:true})).toHaveValue(id);await expect(page.getByLabel("最多执行次数",{exact:true})).toHaveValue("3");
  await expect(page.getByLabel("截止时间（可选，本机时区）")).toHaveValue("2030-12-31T23:00");
  await expect(page.getByRole("region",{name:"计划摘要",exact:true}).getByText("最多执行 3 次",{exact:true})).toBeVisible();
  expect(await page.evaluate(()=>document.documentElement.scrollWidth)).toBeLessThanOrEqual(width);
  const field=await page.getByRole("combobox",{name:"店铺",exact:true}).boundingBox(),summary=await page.getByRole("region",{name:"计划摘要",exact:true}).boundingBox();
  if(width>=1440){const workspace=await page.getByRole("main").boundingBox();expect(summary!.x).toBeGreaterThan(field!.x+field!.width);expect(workspace!.x+workspace!.width-summary!.x-summary!.width).toBeLessThanOrEqual(32);expect(summary!.width).toBe(320);}else expect(summary!.y).toBeGreaterThan(field!.y);
  const footer=await page.getByRole("group",{name:"计划操作",exact:true}).boundingBox();expect(footer!.y+footer!.height).toBeLessThanOrEqual(901);
  const check=await new AxeBuilder({page}).include(".console-page").withTags(["wcag2a","wcag2aa"]).analyze();expect(check.violations.filter(v=>v.impact==="critical"||v.impact==="serious")).toEqual([]);
  if(width===1920&&theme==="dark")await page.screenshot({path:info.outputPath("schedule-report-only-desktop.png"),fullPage:true});
 }
 quoteUnavailable=true;await page.getByRole("button",{name:"核对计划",exact:true}).click();await expect(page.getByRole("group",{name:"计划操作"}).getByRole("alert")).toBeVisible();await expect(page.getByLabel("最多执行次数")).toHaveValue("3");
 quoteUnavailable=false;await page.getByRole("button",{name:"核对计划",exact:true}).click();await expect(page.getByText("每次执行自动保存一份个人报告")).toBeVisible();expect(creates).toBe(0);
 await page.getByRole("button",{name:"确认并保存计划"}).click();await expect(page.getByRole("heading",{name:"上次操作结果待核实"})).toBeVisible();await page.reload();await page.getByRole("button",{name:"核实原操作"}).click();await expect(page.getByRole("heading",{name:`${store.name} · 店铺经营日报`})).toBeVisible();expect(creates).toBe(2);expect(JSON.stringify(plan)).toBe(originalPlan);
 reportSaved=true;report.consumedOccurrences="1";
 await page.goto("/workbench/ai/tasks?view=schedules");
 for(const width of [1440,1920,390])for(const theme of ["light","dark"]){
  await page.setViewportSize({width,height:900});await page.getByRole("combobox",{name:"显示主题"}).selectOption(theme);await settleTheme();
  const reportLink=page.getByRole("link",{name:"查看日报",exact:true});await expect(reportLink).toHaveAttribute("href",`/workbench/ai/reports?reportId=${savedReportId}`);
  await expect(page.getByText("日报已生成",{exact:true})).toBeVisible();expect(await reportLink.evaluate(node=>node.closest("button")===null)).toBe(true);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth)).toBeLessThanOrEqual(width);
  const check=await new AxeBuilder({page}).include(".console-page").withTags(["wcag2a","wcag2aa"]).analyze();expect(check.violations.filter(v=>v.impact==="critical"||v.impact==="serious")).toEqual([]);
  if(theme==="dark"&&(width===1920||width===390))await page.screenshot({path:info.outputPath(`schedule-recent-${width}.png`),fullPage:true});
 }
 const reportLink=page.getByRole("link",{name:"查看日报",exact:true});await reportLink.focus();await page.keyboard.press("Enter");
 await expect(page).toHaveURL(new RegExp(`reportId=${savedReportId}`));await expect(page.getByText("资料不全",{exact:true})).toBeVisible();
 await page.reload();await expect(page.getByText("资料不全",{exact:true})).toBeVisible();
 await page.goto("/workbench/ai/tasks?view=schedules");await expect(page.getByRole("link",{name:"查看日报",exact:true})).toBeVisible();
 reportReadable=false;await page.getByRole("button",{name:"刷新状态",exact:true}).click();await expect(page.getByRole("link",{name:"查看日报",exact:true})).toHaveCount(0);reportReadable=true;
 await page.goto(`/workbench/ai/tasks?view=schedules&schedule_id=${id}`);await expect(page.getByRole("heading",{name:plan.goalSummary,exact:true})).toBeVisible();await expect(page.getByRole("button",{name:"复制新建",exact:true})).toHaveCount(0);await expect(page.getByRole("link",{name:"查看原任务与人工审核"})).toHaveAttribute("href",`/workbench/ai/tasks/${taskID}`);
 await page.getByRole("button",{name:"暂停未来执行"}).click();await expect(page.getByRole("button",{name:"恢复未来执行"})).toBeVisible();await page.getByRole("button",{name:"恢复未来执行"}).click();await expect(page.getByRole("button",{name:"暂停未来执行"})).toBeVisible();
 await page.getByRole("button",{name:"编辑计划",exact:true}).click();const goal="保留原商品事实、三个规格组合和两个尺码，不补充缺失信息。".repeat(12);await page.getByRole("textbox",{name:"任务目标",exact:true}).fill(goal);await page.getByLabel("执行日期与时间（本机时区）").fill("2030-11-21T10:00");await page.getByRole("button",{name:"核对时间与预算"}).click();await page.getByText("查看完整计划内容",{exact:true}).focus();await page.keyboard.press("Enter");await expect(page.locator("details[open] p")).toHaveText(goal);
 await page.getByRole("button",{name:"确认修改计划"}).click();await expect(page.getByRole("textbox",{name:"任务目标",exact:true})).toHaveValue(goal);await page.reload();await page.getByRole("button",{name:"核实原操作"}).click();expect(edits).toBe(2);await expect(page.getByRole("link",{name:"查看原任务与人工审核"})).toBeVisible();
 for(const action of ["结束计划","删除计划"]){const trigger=page.getByRole("button",{name:action,exact:true});await trigger.click();const dialog=page.getByRole("dialog");await expect(dialog.getByRole("button",{name:"取消"})).toBeFocused();await page.keyboard.press("Escape");await expect(dialog).toHaveCount(0);await expect(trigger).toBeFocused();}
 plan.state="CLOSED";await page.reload();await expect(page.getByRole("button",{name:"复制新建",exact:true})).toHaveCount(0);await expect(page.getByRole("heading",{name:"执行历史"})).toBeVisible();
 report.state="CLOSED";report.rule={...report.rule,frequency:"once",hour:0,minute:0};report.maxOccurrences="1";report.consumedOccurrences="1";
 await page.goto(`/workbench/ai/tasks?view=schedules&schedule_id=${reportID}`);await page.getByRole("button",{name:"复制新建",exact:true}).click();await expect(page.getByLabel("执行日期与时间（本机时区）")).toHaveValue("");await page.getByLabel("执行日期与时间（本机时区）").fill("2030-11-22T10:00");await page.getByRole("button",{name:"核对计划",exact:true}).click();await page.getByRole("button",{name:"确认并保存计划"}).click();await expect(page.getByText("尚无触发记录",{exact:true})).toBeVisible();expect(creates).toBe(3);await page.reload();await expect(page.getByText("尚无触发记录",{exact:true})).toBeVisible();expect(creates).toBe(3);
});
