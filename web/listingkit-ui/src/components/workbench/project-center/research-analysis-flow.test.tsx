import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {render,screen,fireEvent,waitFor,cleanup,act} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {ResearchAnalysisFlow} from "./research-analysis-flow";
import {MarketAPIError} from "@/lib/api/supply-market";
import type {ResearchCandidate} from "@/lib/contracts/market-research";
const mock=vi.hoisted(()=>({inspect:vi.fn(),prepare:vi.fn(),execute:vi.fn(),preparation:vi.fn(),run:vi.fn(),recent:vi.fn(),config:vi.fn(),context:{effectiveMemberId:"member",registerOrganizationSwitchGuard:()=>()=>{}}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>mock.context}));
vi.mock("@/lib/api/research-analysis",()=>({inspectResearch:mock.inspect,prepareResearch:mock.prepare,executeResearch:mock.execute,readResearchPreparation:mock.preparation,readResearchRun:mock.run,recentResearchRuns:mock.recent}));
vi.mock("@/lib/api/agent-configuration",async original=>({...await original<object>(),configurationRequest:mock.config}));
const project="11111111-1111-4111-8111-111111111111",key="22222222-2222-4222-8222-222222222222",other="33333333-3333-4333-8333-333333333333",hash="a".repeat(64);
const selected=[key,other].map<ResearchCandidate>(id=>({id,revision:1,state:"TO_RESEARCH",selectedLinkId:id,evidenceDigest:hash,evidence:[],history:[],observation:{title:"Saved sample",site:"us",asin:"B000123456",batchId:project,sourceUrl:"https://www.amazon.com/dp/B000123456",capturedAt:"2026-11-02T00:00:00Z",parserVersion:"amazon-v1",missing:[],query:{mode:"keyword",keyword:"sample",asinCount:0,limit:2,fields:["title"]},ref:{selection:{itemId:id,revision:1,publicationId:id,version:"1"},source:{kind:"amazon_data",productKey:id,publicationId:id,operationId:id,version:"1"},jobId:id,checksum:hash},item:{id,batchId:project,revision:1,createdAt:"2026-11-02T00:00:00Z",source:{kind:"amazon_data",productKey:id,publicationId:id,operationId:id,version:"1"}}}}));
const facts={version:"market-research-inspection-v1",projectId:project,candidates:selected.map((c,n)=>({ref:{ItemID:c.id,ItemRevision:"1",ProductKey:c.id,PublicationID:c.id,CatalogVersion:"1"},research:{CandidateID:c.id,CandidateRevision:"1",SelectedLinkID:c.id,EvidenceDigest:hash},observation:{site:"us",asin:"B000123456",capturedAt:"2026-11-02T00:00:00Z",parserVersion:"amazon-v1",checksum:hash,query:{mode:"keyword",keyword:"sample",asinCount:0,limit:2,fields:["title"]}},facts:[{id:"fact:"+(n?"b":"a").repeat(64),path:"title",value:n?"Sample B":"Sample A"}],missing:["sales","landed_cost"],warnings:[]})),summary:{selected:2,unpriced:2,prices:[]}};
const prepared={contextId:project,requestKey:key,configurationSnapshotRef:{Kind:"agent-configuration-v1",ID:key,Digest:hash},modeSnapshotRef:{Kind:"agent-execution-mode-v1",ID:key,Digest:hash},modeRevision:"0",maximumTokens:10000,maximumCostMicros:1000000,maximumCallCostMicros:500000,currency:"CNY"};
const result={runId:other,contextId:project,requestKey:key,phase:"human_review_required",revision:"4",configurationSnapshotRef:prepared.configurationSnapshotRef,modeSnapshotRef:prepared.modeSnapshotRef,modeRevision:"0",executionMode:"default",goalSummary:"Saved goal",validated:true,inspection:facts,proposal:{overview:"Saved advisory overview",overviewEvidenceIds:[facts.candidates[0].facts[0].id],assessments:[{candidateId:key,recommendation:"RESEARCH_MORE",advice:"Verify complete costs",evidenceIds:[facts.candidates[0].facts[0].id]}]},unresolved:[],tokens:60,costMicros:30,currency:"CNY",usageStatus:"observed"};
let queries:QueryClient;
function open(){queries=new QueryClient({defaultOptions:{queries:{retry:false}}});return render(<QueryClientProvider client={queries}><ResearchAnalysisFlow scope={{userId:"user",organizationId:"org"}} projectId={project} selected={selected} archived={false} view="selection"/></QueryClientProvider>);}
beforeEach(()=>{localStorage.clear();for(const f of [mock.inspect,mock.prepare,mock.execute,mock.preparation,mock.run,mock.recent,mock.config])f.mockReset();mock.config.mockImplementation(async(_s,path)=>path.startsWith("execution-modes")?{mode:"default",revision:"0"}:{agent:{activation:"ENABLED"},canUse:true,canReadRuns:true,capabilities:[{id:"text.generate",readiness:"AVAILABLE"}]});mock.inspect.mockResolvedValue(facts);mock.prepare.mockImplementation(async(_s,intent)=>({...prepared,requestKey:intent.key}));mock.preparation.mockResolvedValue(prepared);mock.run.mockRejectedValue(new MarketAPIError("NOT_FOUND",404,true));mock.recent.mockResolvedValue({items:[],nextCursor:""});mock.execute.mockImplementation(async(_s,intent)=>({...result,requestKey:intent.key}));});
afterEach(()=>{cleanup();queries?.clear();localStorage.clear();});
async function prepare(){fireEvent.click(screen.getByRole("button",{name:"查看所选商品比较"}));await waitFor(()=>expect(screen.getByRole("button",{name:"准备AI分析"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"准备AI分析"}));}
it("requires a fact read, explicit preparation and separate cost confirmation before starting",async()=>{
 open();await screen.findByRole("button",{name:"准备AI分析"});expect(mock.inspect).not.toHaveBeenCalled();expect(mock.prepare).not.toHaveBeenCalled();expect(mock.execute).not.toHaveBeenCalled();
 await prepare();await screen.findByRole("heading",{name:"确认本次AI分析"});expect(mock.execute).not.toHaveBeenCalled();await waitFor(()=>expect(screen.getByRole("button",{name:"确认生成AI建议"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"确认生成AI建议"}));await screen.findByText("Saved advisory overview");expect(mock.execute).toHaveBeenCalledOnce();expect(screen.getByText("建议进一步了解 · Sample A")).toBeInTheDocument();expect(screen.queryByRole("button",{name:"批准测试"})).toBeNull();
});
it("keeps unreadable preparation uncertainty and verifies the original receipt without preparing again",async()=>{
 mock.prepare.mockRejectedValue(new MarketAPIError("PERMISSION_DENIED",403,true));open();await prepare();await screen.findByRole("heading",{name:"查询本次分析结果"});const intent=mock.prepare.mock.calls[0][1];mock.preparation.mockRejectedValueOnce(new MarketAPIError("PERMISSION_DENIED",403,true));
 fireEvent.click(screen.getByRole("button",{name:"查询本次分析结果"}));await waitFor(()=>expect(mock.preparation).toHaveBeenCalledOnce());expect(mock.preparation.mock.calls[0][2]).toBe(intent.key);expect(mock.prepare).toHaveBeenCalledOnce();expect(screen.queryByRole("button",{name:"准备AI分析"})).toBeNull();
 expect(screen.queryByRole("button",{name:"取消本次准备并重新选择"})).toBeNull();
 mock.preparation.mockResolvedValue({...prepared,requestKey:intent.key});fireEvent.click(screen.getByRole("button",{name:"查询本次分析结果"}));await screen.findByRole("heading",{name:"确认本次AI分析"});expect(mock.prepare).toHaveBeenCalledOnce();expect(mock.execute).not.toHaveBeenCalled();
});
it("invalidates an old no-receipt proof when a new lookup cannot establish absence",async()=>{
 mock.prepare.mockRejectedValue(new MarketAPIError("CONFLICT",409,false));open();await prepare();await screen.findByRole("heading",{name:"查询本次分析结果"});
 mock.preparation.mockRejectedValueOnce(new MarketAPIError("NOT_FOUND",404,false));fireEvent.click(screen.getByRole("button",{name:"查询本次分析结果"}));await screen.findByRole("button",{name:"取消本次准备并重新选择"});
 mock.preparation.mockRejectedValue(new MarketAPIError("PERMISSION_DENIED",403,true));fireEvent.click(screen.getByRole("button",{name:"查询本次分析结果"}));
 await waitFor(()=>expect(mock.preparation).toHaveBeenCalledTimes(2));expect(screen.queryByRole("button",{name:"取消本次准备并重新选择"})).toBeNull();
 expect(screen.queryByRole("button",{name:"准备AI分析"})).toBeNull();expect(mock.execute).not.toHaveBeenCalled();
});
it("never releases a pending model start through the unsubmitted preparation action",async()=>{
 mock.execute.mockRejectedValue(new MarketAPIError("OUTCOME_UNKNOWN",503,true));open();await prepare();await waitFor(()=>expect(screen.getByRole("button",{name:"确认生成AI建议"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"确认生成AI建议"}));await screen.findByRole("heading",{name:"查询本次分析结果"});
 mock.run.mockRejectedValue(new MarketAPIError("NOT_FOUND",404,false));fireEvent.click(screen.getByRole("button",{name:"查询本次分析结果"}));await screen.findByText(/尚未查到这次操作记录/);
 expect(screen.queryByRole("button",{name:"取消本次准备并重新选择"})).toBeNull();expect(screen.getByRole("button",{name:"分析另一组商品"})).toBeDisabled();expect(mock.execute).toHaveBeenCalledOnce();
});
it("releases a rejected preparation only after original-key lookup proves no receipt exists",async()=>{
 mock.prepare.mockRejectedValue(new MarketAPIError("CONFLICT",409,false));open();await prepare();await screen.findByRole("heading",{name:"查询本次分析结果"});
 const intent=mock.prepare.mock.calls[0][1];expect(screen.queryByRole("button",{name:"取消本次准备并重新选择"})).toBeNull();
 mock.preparation.mockRejectedValue(new MarketAPIError("NOT_FOUND",404,false));fireEvent.click(screen.getByRole("button",{name:"查询本次分析结果"}));
 fireEvent.click(await screen.findByRole("button",{name:"取消本次准备并重新选择"}));expect(mock.preparation.mock.calls[0][2]).toBe(intent.key);expect(mock.execute).not.toHaveBeenCalled();
 expect(await screen.findByRole("button",{name:"准备AI分析"})).toBeDisabled();mock.prepare.mockImplementation(async(_s,next)=>({...prepared,requestKey:next.key}));
 await prepare();await screen.findByRole("heading",{name:"确认本次AI分析"});expect(mock.prepare.mock.calls[1][1].key).not.toBe(intent.key);expect(mock.execute).not.toHaveBeenCalled();
});
it("masks cached cross-candidate output immediately after a history authorization failure",async()=>{
 open();await prepare();await waitFor(()=>expect(screen.getByRole("button",{name:"确认生成AI建议"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"确认生成AI建议"}));await screen.findByText("Saved advisory overview");mock.run.mockRejectedValue(new MarketAPIError("PERMISSION_DENIED",403,true));
 await act(async()=>{await queries.invalidateQueries({predicate:q=>q.queryKey.includes("run")});});
 await screen.findByText(/本次分析使用的部分资料现在无法查看/);expect(screen.queryByText("Saved advisory overview")).toBeNull();expect(screen.queryByRole("table")).toBeNull();expect(mock.execute).toHaveBeenCalledOnce();
});
it("keeps an UNKNOWN run from opening another analysis",async()=>{
 mock.execute.mockImplementation(async(_s,intent)=>({...result,requestKey:intent.key,phase:"stopped",validated:false,proposal:undefined,usageStatus:"unknown_reserved",stopReason:"usage_unknown"}));open();await prepare();await waitFor(()=>expect(screen.getByRole("button",{name:"确认生成AI建议"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"确认生成AI建议"}));await screen.findByText("这次分析的结果或费用还未确认，请查询操作结果。");expect(screen.getByRole("button",{name:"分析另一组商品"})).toBeDisabled();expect(mock.prepare).toHaveBeenCalledOnce();expect(mock.execute).toHaveBeenCalledOnce();
});
it("masks cached analysis during a parent evidence refresh and reauthorizes the original key",async()=>{
 const screenView=open();await prepare();await waitFor(()=>expect(screen.getByRole("button",{name:"确认生成AI建议"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"确认生成AI建议"}));await screen.findByText("Saved advisory overview");
 const original=mock.execute.mock.calls[0][1].key;mock.run.mockRejectedValue(new MarketAPIError("PERMISSION_DENIED",403,true));
 const props={scope:{userId:"user",organizationId:"org"},projectId:project,archived:false,view:"selection" as const};
 screenView.rerender(<QueryClientProvider client={queries}><ResearchAnalysisFlow {...props} selected={[]} evidenceReadKey="refresh-1" evidenceReadable={false}/></QueryClientProvider>);
 expect(screen.queryByText("Saved advisory overview")).toBeNull();expect(screen.queryByRole("table")).toBeNull();
 screenView.rerender(<QueryClientProvider client={queries}><ResearchAnalysisFlow {...props} selected={[]} evidenceReadKey="refresh-1" evidenceReadable/></QueryClientProvider>);
 await waitFor(()=>expect(mock.run.mock.calls.some(c=>c[2]===original)).toBe(true));await screen.findByText(/本次分析使用的部分资料现在无法查看/);expect(screen.queryByText("Saved advisory overview")).toBeNull();
 mock.run.mockResolvedValue({...result,requestKey:original});
 screenView.rerender(<QueryClientProvider client={queries}><ResearchAnalysisFlow {...props} selected={selected.map(c=>({...c,revision:2}))} evidenceReadKey="refresh-2" evidenceReadable/></QueryClientProvider>);
 await screen.findByText("Saved advisory overview");expect(mock.execute).toHaveBeenCalledOnce();
});
