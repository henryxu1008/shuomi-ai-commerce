import {createHash} from "node:crypto";
import {encode} from "@auth/core/jwt";
import AxeBuilder from "@axe-core/playwright";
import {test,expect} from "@playwright/test";
// Opt-in development UI fixture: normal Auth.js cipher and middleware, a
// synthetic local secret, intercepted local APIs, no IdP/platform/model calls.
// Run with RESEARCH_CENTERS_UI_FIXTURE=1, AUTH_SECRET set to the constant below,
// AUTH_URL=LISTINGKIT_PUBLIC_BASE_URL=http://127.0.0.1:3210,
// ZITADEL_ISSUER_URL=https://research-ui.invalid and ZITADEL_CLIENT_ID=research-ui-fixture.
const secret="codex-research-centers-synthetic-ui-secret-only";
test.skip(process.env.RESEARCH_CENTERS_UI_FIXTURE!=="1"||process.env.AUTH_SECRET!==secret,"requires the explicit synthetic local UI fixture environment");
const project="11111111-1111-4111-8111-111111111111",ids=["22222222-2222-4222-8222-222222222222","33333333-3333-4333-8333-333333333333"],hash="a".repeat(64),at="2026-11-02T00:00:00Z";
test("the four native pages share a project and preserve evidence through analysis and human selection",async({page,context,baseURL},info)=>{
 test.setTimeout(120000);
 const origin=baseURL!,issuer="https://research-ui.invalid",client="research-ui-fixture";
 const namespace=createHash("sha256").update(JSON.stringify([origin,issuer,client])).digest("hex").slice(0,32),name=`shuomi.${namespace}.session-token`;
 const value=await encode({secret,salt:name,maxAge:3600,token:{accessToken:"synthetic-ui-access",expiresAt:Math.floor(Date.now()/1000)+3600,identityVersion:3,identity:{tenantId:"org",userId:"user",userType:"zitadel",roles:[]}}});
 await context.addCookies([{name,value,url:origin,httpOnly:true,sameSite:"Lax"}]);
 await page.addInitScript(()=>localStorage.setItem("listingkit-theme","light"));
 const permissions=["workbench.project.read","workbench.project.manage","workbench.collection.read","workbench.agent.use","workbench.agent.local.write","workbench.agent.configure"];
 const currentProject={id:project,title:"美国站桌面收纳研究",goal:"核对美国站桌面收纳盒的保存样本、页面报价和规格缺项，再决定哪些候选值得测试。",kind:"PRODUCT_RESEARCH",dueDate:"",revision:1,archived:false,createdAt:at,updatedAt:at,storeScope:false,references:[],taskTotal:0,taskCompleted:0,taskPending:0,taskSummaryAvailable:true};
 const candidates=ids.map((id,n)=>({id,revision:1,state:"TO_RESEARCH",selectedLinkId:id,evidenceDigest:hash,evidence:[],history:[] as unknown[],observation:{title:n?"模块化桌面收纳盒":"双层桌面文件收纳架",batchId:project,site:"us",asin:n?"B000000002":"B000000001",sourceUrl:"https://www.amazon.com/dp/"+(n?"B000000002":"B000000001"),capturedAt:at,parserVersion:"amazon-v1",currency:"USD",price:19.99+n*4,priceBasis:"displayed_listing_offer",missing:["rating","review_count"],query:{mode:"keyword",keyword:"desk organizer",asinCount:0,limit:2,fields:["title","price"]},ref:{selection:{itemId:id,revision:1,publicationId:id,version:"1"},source:{kind:"amazon_data",productKey:id,publicationId:id,operationId:id,version:"1"},jobId:id,checksum:hash},item:{id,batchId:project,revision:1,title:n?"模块化桌面收纳盒":"双层桌面文件收纳架",createdAt:at,source:{kind:"amazon_data",productKey:id,publicationId:id,operationId:id,version:"1"}}}}));
 const inspection={version:"market-research-inspection-v1",projectId:project,candidates:candidates.map((c,n)=>({ref:{ItemID:c.id,ItemRevision:"1",ProductKey:c.id,CatalogVersion:"1",PublicationID:c.id},research:{CandidateID:c.id,CandidateRevision:String(c.revision),SelectedLinkID:c.id,EvidenceDigest:hash},observation:{site:"us",asin:c.observation.asin,capturedAt:at,parserVersion:"amazon-v1",checksum:hash,query:c.observation.query,currency:"USD",price:c.observation.price,priceBasis:"displayed_listing_offer"},facts:[{id:"fact:"+(n?"b":"a").repeat(64),path:"title",value:c.observation.title}],missing:["sales","market_demand","shipping","landed_cost"],warnings:["SAVED_OBSERVATIONS_ONLY"]})),summary:{selected:2,unpriced:0,prices:[{currency:"USD",basis:"displayed_listing_offer",count:2,minimum:19.99,maximum:23.99}]}};
 let prepared:Record<string,unknown>|undefined,result:Record<string,unknown>|undefined,starts=0,prepares=0;
 await page.route("**/*",async route=>{
 const request=route.request(),url=new URL(request.url());if(url.origin!==origin){await route.abort();return;}
 if(!url.pathname.startsWith("/api/")){await route.continue();return;}
 const json=(data:unknown,status=200)=>route.fulfill({status,contentType:"application/json",body:JSON.stringify(data)}),path=url.pathname,method=request.method();
 if(path==="/api/workbench/context")return json({user:{id:"user"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,projectCenterAvailable:true,marketResearchAvailable:true,organizations:[{id:"org",name:"开发测试企业",roles:[],permissions}]});
 if(path.startsWith("/api/workbench/projects")){if(method!=="GET")return json({id:project,revision:1,replayed:false});return path==="/api/workbench/projects"?json({projects:[currentProject],next:""}):json(currentProject);}
 if(path==="/api/workbench/agents/product.research.agent")return json({agent:{agentId:"product.research.agent",activation:"ENABLED",revision:"1",activationEpoch:"1",defaultTemplate:null,updatedAt:at},name:"AI分析建议智能体",description:"核对明确所选的研究证据",definitionVersion:"v1.0.0",parameterSchema:"market-research-config-v1",canConfigure:true,canUse:true,canReadRuns:true,capabilities:[{id:"text.generate",support:"REQUIRED",readiness:"AVAILABLE",reason:"本地模拟",observedAt:at}]});
 if(path.includes("/execution-modes/"))return json({subject:{kind:"registered_agent",id:"product.research.agent"},definition:{id:"product.research.agent",version:"v1.0.0"},mode:"default",revision:"0",policyVersion:"agent-execution-policy-v1",canConfigure:true,updatedAt:at,updatedBy:"user"});
 const base=`/api/workbench/research/projects/${project}`;
 if(path===base)return json({items:candidates,incomplete:false});
 if(path===base+"/commands"&&method==="POST"){
 const command=request.postDataJSON(),c=candidates.find(c=>c.id===command.candidateId)!;expect(command.expectedRevision).toBe(c.revision);c.revision++;c.state=command.action==="review"?"PENDING_CONFIRMATION":command.state;c.history.push({state:c.state,revision:c.revision,at,reason:command.reason,evidenceDigest:hash});return json({key:request.headers()["idempotency-key"],candidateIds:[c.id],revisions:{[c.id]:c.revision},replayed:false});
 }
 if(path===base+"/analysis/inspect")return json(inspection);
 if(path===base+"/analysis/prepare"){
 prepares++;const input=request.postDataJSON();prepared={contextId:project,requestKey:input.requestKey,configurationSnapshotRef:{Kind:"agent-configuration-v1",ID:input.requestKey,Digest:hash},modeSnapshotRef:{Kind:"agent-execution-mode-v1",ID:input.requestKey,Digest:hash},modeRevision:"0",maximumTokens:10000,maximumCostMicros:1000000,maximumCallCostMicros:500000,currency:"CNY"};return json(prepared);
 }
 if(path===base+"/analysis/runs"&&method==="POST"){
 starts++;result={runId:ids[1],contextId:project,requestKey:prepared!.requestKey,phase:"human_review_required",revision:"4",configurationSnapshotRef:prepared!.configurationSnapshotRef,modeSnapshotRef:prepared!.modeSnapshotRef,executionMode:"default",modeRevision:"0",goalSummary:"比较所选保存样本",inspection,validated:true,proposal:{overview:"两项样本均需补齐完整成本，建议先进一步核实。",overviewEvidenceIds:[inspection.candidates[0].facts[0].id],assessments:[{candidateId:ids[0],recommendation:"RESEARCH_MORE",advice:"补充到岸成本与当前供货条件后再决定测试。",evidenceIds:[inspection.candidates[0].facts[0].id]}]},unresolved:[],tokens:60,costMicros:30,currency:"CNY",usageStatus:"observed"};return json(result);
 }
 if(path===base+"/analysis/runs")return json({items:[],nextCursor:""});
 if(path.includes("/analysis/runs/"))return result?json(result):json({code:"NOT_FOUND",contextId:project,requestKey:prepared?.requestKey,operationMayExist:true},404);
 return json({code:"NOT_FOUND"},404);
 });
 await page.setViewportSize({width:1440,height:1000});
 await page.goto("/workbench/data/research");await expect(page.getByRole("heading",{name:"市场调研业务流程"})).toBeVisible();await page.screenshot({path:info.outputPath("research-home-light-desktop.png"),fullPage:true});await page.getByRole("link",{name:/开始调研/}).click();await expect(page.getByRole("heading",{name:"市场调研",level:1})).toBeVisible();await expect(page.getByRole("heading",{name:currentProject.title})).toBeVisible();
 await expect(page.locator("html")).toHaveClass(/light/);
 const listA11y=await new AxeBuilder({page}).include(".console-page").withTags(["wcag2a","wcag2aa"]).analyze();expect(listA11y.violations.filter(v=>v.impact==="serious"||v.impact==="critical")).toEqual([]);
 await page.screenshot({path:info.outputPath("research-list-light-desktop.png"),fullPage:true});
 await page.getByRole("link",{name:"查看调研",exact:true}).click();await expect(page.getByText(candidates[0].observation.title,{exact:true})).toBeVisible();expect(starts).toBe(0);
 await page.screenshot({path:info.outputPath("research-detail-light-desktop.png"),fullPage:true});
 await page.getByRole("checkbox").nth(0).check();await page.getByRole("checkbox").nth(1).check();await page.getByRole("button",{name:"查看所选商品比较"}).click();await expect(page.getByText("本次比较 2 件")).toBeVisible();
 await page.getByRole("link",{name:"进入智能选品"}).click();await expect(page.getByRole("heading",{name:"智能选品",level:1})).toBeVisible();await expect(page.getByRole("checkbox").nth(0)).toBeChecked();await expect(page.getByRole("checkbox").nth(1)).toBeChecked();
 await page.getByRole("button",{name:"查看所选商品比较"}).click();await expect(page.getByRole("button",{name:"准备AI分析"})).toBeEnabled();expect(starts).toBe(0);
 await page.getByRole("button",{name:"准备AI分析"}).click();await expect(page.getByRole("heading",{name:"确认本次AI分析"})).toBeVisible();expect(prepares).toBe(1);expect(starts).toBe(0);
 await page.getByRole("button",{name:"确认生成AI建议"}).click();await expect(page.getByText("两项样本均需补齐完整成本，建议先进一步核实。")).toBeVisible();expect(starts).toBe(1);
 await page.screenshot({path:info.outputPath("selection-review-light-desktop.png"),fullPage:true});
 await page.setViewportSize({width:390,height:844});await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth)).toBeLessThanOrEqual(391);await page.screenshot({path:info.outputPath("selection-review-light-mobile.png"),fullPage:true});
 await page.getByRole("combobox",{name:"显示主题"}).selectOption("dark");await expect(page.locator("html")).toHaveClass(/dark/);await page.screenshot({path:info.outputPath("selection-review-dark-mobile.png"),fullPage:true});
 const accessibility=await new AxeBuilder({page}).include(".console-page").withTags(["wcag2a","wcag2aa"]).analyze();expect(accessibility.violations.filter(v=>v.impact==="critical"||v.impact==="serious")).toEqual([]);
 await page.getByRole("button",{name:"开始评估"}).first().click();await page.getByLabel("选择理由").fill("已独立核对保存资料，批准进一步测试。");await page.getByRole("button",{name:"批准测试",exact:true}).click();await expect(page.getByText("批准测试",{exact:true}).first()).toBeVisible();
 const transfer=new URL((await page.getByRole("link",{name:"批次 1 · 核对 1 件商品"}).getAttribute("href"))!,origin);expect(JSON.parse(transfer.searchParams.get("transfer")!)).toEqual([candidates[0].observation.ref.selection]);expect(starts).toBe(1);
 await page.getByRole("combobox",{name:"显示主题"}).selectOption("light");await expect(page.locator("html")).toHaveClass(/light/);
 await page.setViewportSize({width:1440,height:1000});await page.goto("/workbench/supply/selection");await expect(page.getByRole("heading",{name:"智能选品业务流程"})).toBeVisible();await page.screenshot({path:info.outputPath("selection-home-light-desktop.png"),fullPage:true});await page.getByRole("link",{name:/挑选重点商品/}).click();await expect(page.getByRole("heading",{name:"智能选品",level:1})).toBeVisible();await page.screenshot({path:info.outputPath("selection-list-light-desktop.png"),fullPage:true});
});
