import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {render,screen,fireEvent,waitFor,cleanup,act} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {ResearchAnalysisFlow} from "./research-analysis-flow";
import {ResearchPanel} from "./research-panel";
import {MarketAPIError} from "@/lib/api/supply-market";
import type {ResearchCandidate} from "@/lib/contracts/market-research";
const mock=vi.hoisted(()=>({page:vi.fn(),inspect:vi.fn(),prepare:vi.fn(),execute:vi.fn(),preparation:vi.fn(),run:vi.fn(),recent:vi.fn(),config:vi.fn(),context:{effectiveMemberId:"member",permissions:["workbench.collection.read","workbench.project.manage"],registerOrganizationSwitchGuard:()=>()=>{}}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>mock.context}));
vi.mock("@/lib/api/market-research",async original=>({...await original<object>(),researchRequest:mock.page}));
vi.mock("@/lib/api/research-analysis",()=>({inspectResearch:mock.inspect,prepareResearch:mock.prepare,executeResearch:mock.execute,readResearchPreparation:mock.preparation,readResearchRun:mock.run,recentResearchRuns:mock.recent}));
vi.mock("@/lib/api/agent-configuration",async original=>({...await original<object>(),configurationRequest:mock.config}));
const project="11111111-1111-4111-8111-111111111111",key="22222222-2222-4222-8222-222222222222",other="33333333-3333-4333-8333-333333333333",hash="a".repeat(64);
const selected=[key,other].map<ResearchCandidate>(id=>({id,revision:1,state:"TO_RESEARCH",selectedLinkId:id,evidenceDigest:hash,evidence:[],history:[],observation:{title:"Saved sample",site:"us",asin:"B000123456",batchId:project,sourceUrl:"https://www.amazon.com/dp/B000123456",capturedAt:"2026-11-02T00:00:00Z",parserVersion:"amazon-v1",missing:[],query:{mode:"keyword",keyword:"sample",asinCount:0,limit:2,fields:["title"]},ref:{selection:{itemId:id,revision:1,publicationId:id,version:"1"},source:{kind:"amazon_data",productKey:id,publicationId:id,operationId:id,version:"1"},jobId:id,checksum:hash},item:{id,batchId:project,revision:1,createdAt:"2026-11-02T00:00:00Z",source:{kind:"amazon_data",productKey:id,publicationId:id,operationId:id,version:"1"}}}}));
const facts={version:"market-research-inspection-v1",projectId:project,candidates:selected.map((c,n)=>({ref:{ItemID:c.id,ItemRevision:"1",ProductKey:c.id,PublicationID:c.id,CatalogVersion:"1"},research:{CandidateID:c.id,CandidateRevision:"1",SelectedLinkID:c.id,EvidenceDigest:hash},observation:{site:"us",asin:"B000123456",capturedAt:"2026-11-02T00:00:00Z",parserVersion:"amazon-v1",checksum:hash,query:{mode:"keyword",keyword:"sample",asinCount:0,limit:2,fields:["title"]}},facts:[{id:"fact:"+(n?"b":"a").repeat(64),path:"title",value:n?"Sample B":"Sample A"}],missing:["sales","landed_cost"],warnings:[]})),summary:{selected:2,unpriced:2,prices:[]}};
const prepared={contextId:project,requestKey:key,configurationSnapshotRef:{Kind:"agent-configuration-v1",ID:key,Digest:hash},modeSnapshotRef:{Kind:"agent-execution-mode-v1",ID:key,Digest:hash},modeRevision:"0",maximumTokens:10000,maximumCostMicros:1000000,maximumCallCostMicros:500000,referencePriceVersion:"reference-v1",currency:"CNY"};
const result={runId:other,contextId:project,requestKey:key,phase:"human_review_required",revision:"4",configurationSnapshotRef:prepared.configurationSnapshotRef,modeSnapshotRef:prepared.modeSnapshotRef,modeRevision:"0",executionMode:"default",goalSummary:"Saved goal",validated:true,inspection:facts,proposal:{overview:"Saved advisory overview",overviewEvidenceIds:[facts.candidates[0].facts[0].id],assessments:[{candidateId:key,recommendation:"RESEARCH_MORE",advice:"Verify complete costs",evidenceIds:[facts.candidates[0].facts[0].id]}]},unresolved:[],tokens:60,costMicros:30,currency:"CNY",usageStatus:"observed"};
let queries:QueryClient;
function open(){queries=new QueryClient({defaultOptions:{queries:{retry:false}}});return render(<QueryClientProvider client={queries}><ResearchAnalysisFlow scope={{userId:"user",organizationId:"org"}} projectId={project} selected={selected} archived={false} view="selection"/></QueryClientProvider>);}
beforeEach(()=>{localStorage.clear();for(const f of [mock.inspect,mock.prepare,mock.execute,mock.preparation,mock.run,mock.recent,mock.config])f.mockReset();mock.config.mockImplementation(async(_s,path)=>path.startsWith("execution-modes")?{mode:"default",revision:"0"}:{agent:{activation:"ENABLED"},canUse:true,canReadRuns:true,capabilities:[{id:"text.generate",readiness:"AVAILABLE"}]});mock.inspect.mockResolvedValue(facts);mock.prepare.mockImplementation(async(_s,intent)=>({...prepared,requestKey:intent.key}));mock.preparation.mockResolvedValue(prepared);mock.run.mockRejectedValue(new MarketAPIError("NOT_FOUND",404,true));mock.recent.mockResolvedValue({items:[],nextCursor:""});mock.execute.mockImplementation(async(_s,intent)=>({...result,requestKey:intent.key}));});
afterEach(()=>{cleanup();queries?.clear();localStorage.clear();vi.unstubAllGlobals();});
async function prepare(){fireEvent.click(screen.getByRole("button",{name:"查看所选商品比较"}));await waitFor(()=>expect(screen.getByRole("button",{name:"准备AI分析"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"准备AI分析"}));}
it("requires a fact read, explicit preparation and separate cost confirmation before starting",async()=>{
 open();await screen.findByRole("button",{name:"准备AI分析"});expect(mock.inspect).not.toHaveBeenCalled();expect(mock.prepare).not.toHaveBeenCalled();expect(mock.execute).not.toHaveBeenCalled();
 await prepare();await screen.findByRole("heading",{name:"确认本次AI分析"});expect(screen.getByText(/费用按此版本估算，不代表厂商最终账单/)).toHaveTextContent("reference-v1");expect(mock.execute).not.toHaveBeenCalled();await waitFor(()=>expect(screen.getByRole("button",{name:"确认生成AI建议"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"确认生成AI建议"}));await screen.findByText("Saved advisory overview");expect(mock.execute).toHaveBeenCalledOnce();expect(screen.getByText("建议进一步了解 · Sample A")).toBeInTheDocument();expect(screen.queryByRole("button",{name:"批准测试"})).toBeNull();
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
it("releases only a fresh start whose original response proves it did not claim a run",async()=>{
 let revision="0";
 mock.config.mockImplementation(async(_scope,path)=>path.startsWith("execution-modes")?{mode:"default",revision}:{agent:{activation:"ENABLED"},canUse:true,canReadRuns:true,capabilities:[{id:"text.generate",readiness:"AVAILABLE"}]});
 mock.execute.mockImplementationOnce(async()=>{
  revision="1";
  throw new MarketAPIError("CONFIGURATION_CHANGED",409,false);
 });
 open();await prepare();
 await waitFor(()=>expect(screen.getByRole("button",{name:"确认生成AI建议"})).toBeEnabled());
 fireEvent.click(screen.getByRole("button",{name:"确认生成AI建议"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"查看所选商品比较"})).toBeEnabled());
 expect(screen.queryByRole("button",{name:"按原内容重试分析"})).toBeNull();
 fireEvent.click(screen.getByRole("button",{name:"查看所选商品比较"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"准备AI分析"})).toBeEnabled());
 fireEvent.click(screen.getByRole("button",{name:"准备AI分析"}));
 await waitFor(()=>expect(mock.prepare).toHaveBeenCalledTimes(2));
 expect(mock.prepare.mock.calls[0][1].key).not.toBe(mock.prepare.mock.calls[1][1].key);
 expect(mock.prepare.mock.calls[1][1].body.expectedModeRevision).toBe("1");
 expect(mock.execute).toHaveBeenCalledOnce();
});

it("keeps an earlier uncertain start when its same-key retry is rejected before claim",async()=>{
 mock.execute.mockRejectedValueOnce(new MarketAPIError("OUTCOME_UNKNOWN",503));
 mock.execute.mockRejectedValueOnce(new MarketAPIError("CONFIGURATION_CHANGED",409,false));
 open();await prepare();
 await waitFor(()=>expect(screen.getByRole("button",{name:"确认生成AI建议"})).toBeEnabled());
 fireEvent.click(screen.getByRole("button",{name:"确认生成AI建议"}));
 const retry=await screen.findByRole("button",{name:"按原内容重试分析"});
 await waitFor(()=>expect(retry).toBeEnabled());
 fireEvent.click(retry);
 await waitFor(()=>expect(mock.execute).toHaveBeenCalledTimes(2));
 expect(mock.execute.mock.calls[1][1].key).toBe(mock.execute.mock.calls[0][1].key);
 expect(screen.getByRole("button",{name:"查看所选商品比较"})).toBeDisabled();
 expect(screen.getByRole("button",{name:"查询本次分析结果"})).toBeVisible();
 expect(mock.prepare).toHaveBeenCalledOnce();
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

const firstPageSamples = [
  {asin:"TPG0000006", price:24.99, checksum:"78d0044fc82e9ca70404418ef0f51e5849dec04b414df9cf22492f089686b44a"},
  {asin:"TPG0000028", price:28.99, checksum:"92f259a00e1b11e7336d4483ac29fa8ece77c8577a46168c0c62e43939a58618"},
];
const readableSamples = selected.map((sample,index) => {
  const original = firstPageSamples[index];
  return {
    ...sample,
    evidenceDigest:String(index+1).repeat(64),
    observation:{
      ...sample.observation!,
      ...original,
      currency:"USD",
      title:original.asin,
      ref:{...sample.observation!.ref,checksum:original.checksum},
    },
  };
});
const sampleInspection = {
  ...facts,
  candidates:facts.candidates.map((candidate,index) => ({
    ...candidate,
    research:{...candidate.research,EvidenceDigest:readableSamples[index].evidenceDigest!},
    observation:{...candidate.observation,...firstPageSamples[index],currency:"USD"},
    facts:[{...candidate.facts[0],value:firstPageSamples[index].asin}],
  })),
  summary:{selected:2,unpriced:0,prices:[{
    currency:"USD",basis:"synthetic_listing_offer",count:2,minimum:24.99,maximum:28.99,
  }]},
};

async function openIncompletePage(view:"research"|"selection"="research") {
  const unreadable = Array.from({length:48},(_,index) => ({
    ...selected[0],
    id:"44444444-4444-4444-8444-"+String(index).padStart(12,"0"),
    observation:undefined,
    evidenceDigest:undefined,
  }));
  mock.page.mockReset().mockResolvedValue({items:[...readableSamples,...unreadable],incomplete:true});
  const real = await vi.importActual<typeof import("@/lib/api/research-analysis")>("@/lib/api/research-analysis");
  mock.inspect.mockImplementation(real.inspectResearch);
  queries = new QueryClient({defaultOptions:{queries:{retry:false}}});
  render(<QueryClientProvider client={queries}>
    <ResearchPanel scope={{userId:"user",organizationId:"org"}} projectId={project} archived={false} view={view}/>
  </QueryClientProvider>);
  fireEvent.click(await screen.findByRole("checkbox",{name:"TPG0000006"}));
  fireEvent.click(screen.getByRole("checkbox",{name:"TPG0000028"}));
  return screen.getByRole("button",{name:"查看所选商品比较"});
}

it("compares the two readable originals through the existing endpoint when the other 48 time out",async()=>{
  const fetcher = vi.fn(async (_path:string,_init:RequestInit) => Response.json(sampleInspection));
  vi.stubGlobal("fetch",fetcher);
  const comparison = await openIncompletePage("selection");
  expect(screen.getByText("部分商品资料暂时无法确认，请刷新后再查看。")).toBeVisible();
  expect(screen.getAllByRole("checkbox",{name:"这份资料暂时无法查看"})).toHaveLength(48);
  expect(comparison).toBeEnabled();
  fireEvent.click(comparison);
  expect(await screen.findByText("本次比较 2 件")).toBeVisible();
  expect(screen.getByText(/USD 页面报价：24.99–28.99/)).toBeVisible();
  expect(fetcher).toHaveBeenCalledOnce();
  const [path,init] = fetcher.mock.calls[0];
  expect(path).toBe("/api/workbench/research/projects/"+project+"/analysis/inspect");
  expect(init.method).toBe("POST");
  expect(JSON.parse(String(init.body)).candidates).toEqual(readableSamples.map(sample => ({
    candidateId:sample.id,expectedRevision:"1",linkId:sample.selectedLinkId,evidenceDigest:sample.evidenceDigest,
    selection:{itemId:sample.id,expectedRevision:"1",originalPublicationId:sample.id,originalVersion:"1"},
  })));
  expect(screen.getByRole("button",{name:"准备AI分析"})).toBeDisabled();
  expect(mock.prepare).not.toHaveBeenCalled();
  expect(mock.execute).not.toHaveBeenCalled();
});

it.each([
  ["version changed",409,"AGENT_CONFLICT"],
  ["digest changed",409,"AGENT_CONFLICT"],
  ["permission revoked",403,"PERMISSION_DENIED"],
] as const)("hides the existing comparison when the endpoint reports %s for a selected original",async (_reason,status,code)=>{
  const fetcher = vi.fn()
    .mockResolvedValueOnce(Response.json(sampleInspection))
    .mockResolvedValueOnce(Response.json({code},{status}));
  vi.stubGlobal("fetch",fetcher);
  const comparison = await openIncompletePage();
  fireEvent.click(comparison);
  await screen.findByText("本次比较 2 件");
  fireEvent.click(comparison);
  await screen.findByText("部分商品资料暂时无法确认，请刷新后重新选择。");
  expect(screen.queryByText("本次比较 2 件")).not.toBeInTheDocument();
  expect(screen.queryByRole("table")).not.toBeInTheDocument();
  expect(fetcher).toHaveBeenCalledTimes(2);
});
it.each(["version","digest"] as const)("rejects a successful response with the wrong selected original %s",async changed=>{
  const [first,second] = sampleInspection.candidates;
  const inconsistent = changed==="version"
    ? {...first,ref:{...first.ref,CatalogVersion:"2"}}
    : {...first,research:{...first.research,EvidenceDigest:"f".repeat(64)}};
  const fetcher = vi.fn().mockResolvedValue(Response.json({...sampleInspection,candidates:[inconsistent,second]}));
  vi.stubGlobal("fetch",fetcher);
  fireEvent.click(await openIncompletePage());
  expect(await screen.findByText("所选商品资料已变化，请刷新后重新选择。")).toBeVisible();
  expect(screen.queryByText("本次比较 2 件")).not.toBeInTheDocument();
  expect(screen.queryByRole("table")).not.toBeInTheDocument();
  expect(fetcher).toHaveBeenCalledOnce();
});