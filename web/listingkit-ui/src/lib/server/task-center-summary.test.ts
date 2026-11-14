import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {collectTaskSummary} from "./task-center-summary";
const mocks=vi.hoisted(()=>({config:vi.fn(),review:vi.fn()}));
vi.mock("./agent-configuration-proxy",()=>({proxyAgentConfiguration:mocks.config}));
vi.mock("./product-title-review-proxy",()=>({proxyProductTitleReview:mocks.review}));
vi.mock("./workbench-proxy",()=>({buildWorkbenchUpstreamRequest:async(req:Request)=>({url:req.url,init:{method:"GET"},responseContract:"context"}),buildWorkbenchBrowserResponse:async(response:Response)=>response}));
const id=(n:number)=>`00000000-0000-4000-8000-${String(n).padStart(12,"0")}`;
const time="2026-11-14T00:00:00Z";
const context=(permissions=["workbench.agent.use","workbench.agent.read","workbench.task.read","local_agent.write"])=>({user:{id:"actor"},homeOrganizationId:"org",effectiveOrganizationId:"org",effectiveMemberId:"member",selectionRequired:false,aiWorkbenchAvailable:true,organizations:[{id:"org",name:"Org",roles:[],permissions}]});
const task=(n:number,state="WAITING_CONFIRMATION")=>({id:id(n),conversationId:id(100),proposalId:id(101),title:"Task",goalSummary:"Goal",createdAt:time,projectionAvailable:true,state,kind:"product.content.localize",agentRunId:id(n+1000),canStart:false,canReconcile:false,canResume:false,canReview:false,productDetailsAvailable:false});
const req=(query="scopeKind=all")=>new Request(`https://ui.test/api/workbench/task-center-summary?${query}`,{headers:{"X-Expected-User-ID":"actor","X-Expected-Organization-ID":"org",cookie:"shuomi_effective_organization=org"}});
const entry={agent:{agentId:"product.content.agent",activation:"DISABLED",revision:"1",activationEpoch:"1",defaultTemplate:null,updatedAt:time},name:"Content",description:"",definitionVersion:"1",parameterSchema:"content-config-v1",canUse:false,canConfigure:false,canReadRuns:true,capabilities:[]};
beforeEach(()=>{mocks.config.mockReset();mocks.review.mockReset();vi.stubGlobal("fetch",vi.fn(async(url:string)=>Response.json(new URL(url).pathname.endsWith("context")?context():{tasks:[task(1)],next:""})));});
afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs()});
it("aggregates every authorized task page independently from the browser pagination",async()=>{
 vi.mocked(fetch).mockImplementation(async url=>{const u=new URL(String(url));return Response.json(u.pathname.endsWith("context")?context():u.searchParams.has("after")?{tasks:[task(2,"PAUSED")],next:""}:{tasks:[task(1)],next:id(1)})});
 const out=await (await collectTaskSummary(req(),"token","actor",{tasks:true,native:false,reviews:false})).json();
 expect(out.status).toBe("complete");expect(out.counts.WAITING_CONFIRMATION).toBe(1);expect(out.counts.PAUSED).toBe(1);expect(out.counts.ERROR).toBe(0);
 expect(vi.mocked(fetch).mock.calls.every(([,init])=>init?.method==="GET")).toBe(true);
});
it("keeps generated suggestions outside the five states without hiding known counts",async()=>{
 mocks.config.mockImplementation(async(request:Request)=>Response.json(request.url.includes("/market")?{items:[entry],nextCursor:""}:{items:[{runId:id(5),operationId:id(50),contextKind:"own_product",productKey:"shirt",targetPlatform:"product",phase:"human_review_required",revision:"1",requestKey:id(51),startedAt:time}],nextCursor:""}));
 const out=await (await collectTaskSummary(req(),"token","actor",{tasks:true,native:true,reviews:false})).json();
 expect(out.status).toBe("complete");expect(out.counts.WAITING_CONFIRMATION).toBe(1);expect(out.excluded).toBe(1);
});
it("discards counts after current permissions change",async()=>{
 let contexts=0;vi.mocked(fetch).mockImplementation(async url=>Response.json(String(url).endsWith("context")?context(++contexts===1?["workbench.agent.use"]:[]):{tasks:[task(1)],next:""}));
 const out=await (await collectTaskSummary(req(),"token","actor",{tasks:true,native:false,reviews:false})).json();expect(out.counts).toBeNull();expect(out.reasons).toContain("changed");
});
it("does not represent a failed later page as a precise count",async()=>{
 vi.mocked(fetch).mockImplementation(async url=>{const u=new URL(String(url));return u.pathname.endsWith("context")?Response.json(context()):u.searchParams.has("after")?Response.json({}, {status:403}):Response.json({tasks:[task(1)],next:id(1)})});
 const out=await (await collectTaskSummary(req(),"token","actor",{tasks:true,native:false,reviews:false})).json();expect(out.status).toBe("partial");expect(out.counts).toBeNull();
});
it("bounds repeated cursors and request exhaustion",async()=>{
 vi.mocked(fetch).mockImplementation(async url=>Response.json(String(url).endsWith("context")?context():{tasks:[],next:id(1)}));
 const out=await (await collectTaskSummary(req(),"token","actor",{tasks:true,native:false,reviews:false})).json();expect(out.reasons).toContain("limit");expect(fetch).toHaveBeenCalledTimes(3);
});
it("does not read unscoped sources for a project filter",async()=>{
 const out=await (await collectTaskSummary(req(`scopeKind=project&projectId=${id(20)}`),"token","actor",{tasks:true,native:true,reviews:true})).json();
 expect(out.status).toBe("complete");expect(mocks.config).not.toHaveBeenCalled();expect(mocks.review).not.toHaveBeenCalled();
 expect(String(vi.mocked(fetch).mock.calls[1][0])).toContain(`projectId=${id(20)}`);
});
it("rejects a mismatched identity before any source request",async()=>{
 const result=await collectTaskSummary(req(),"token","other");expect(result.status).toBe(409);expect(fetch).not.toHaveBeenCalled();
});
it("caps unique empty pages instead of scanning an unbounded history",async()=>{
 let page=0;vi.mocked(fetch).mockImplementation(async url=>Response.json(String(url).endsWith("context")?context():{tasks:[],next:id(++page)}));
 const out=await (await collectTaskSummary(req(),"token","actor",{tasks:true,native:false,reviews:false})).json();expect(out.counts).toBeNull();expect(out.reasons).toContain("limit");expect(fetch).toHaveBeenCalledTimes(100);
});
it("only reads sources permitted to a viewer using production capability gates",async()=>{
 vi.stubEnv("PRODUCT_REVIEW_API_ORIGIN","https://review.test");vi.stubEnv("LISTINGKIT_AGENT_CONFIGURATION_ENABLED","true");vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","https://service.test/api/v1");
 vi.mocked(fetch).mockImplementation(async url=>Response.json(String(url).endsWith("context")?context(["workbench.task.read"]):{tasks:[task(1)],next:""}));
 const out=await (await collectTaskSummary(req(),"token","actor")).json();expect(out.status).toBe("complete");expect(out.counts.WAITING_CONFIRMATION).toBe(1);expect(mocks.review).not.toHaveBeenCalled();expect(mocks.config).not.toHaveBeenCalled();
});
