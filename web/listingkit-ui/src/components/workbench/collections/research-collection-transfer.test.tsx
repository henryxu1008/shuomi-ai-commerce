import {act,cleanup,fireEvent,render as renderWithProviders,screen,waitFor} from "@testing-library/react";
import {AppRouterContext,type AppRouterInstance} from "next/dist/shared/lib/app-router-context.shared-runtime";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {useSyncExternalStore,type ReactElement} from "react";
import {ResearchCollectionTransfer} from "./research-collection-transfer";
import {CollectionPage} from "./collection-page";
import {ResearchPanel} from "../project-center/research-panel";
import {ResearchSelectionGate} from "../project-center/research-selection-gate";
import {ResearchOpportunity} from "../project-center/research-opportunity";
import {createResearchHandoff,handoffToken,researchScopeKey} from "@/lib/workbench/research-handoff";
import {loadSupplyIntent,saveSupplyIntent} from "@/lib/api/supply-intent";
import {SupplyAPIError,type SupplyIntent} from "@/lib/api/supply-chain";
import type {DataJob} from "@/lib/contracts/data-services";
import type {ResearchCandidate} from "@/lib/contracts/market-research";

const f=vi.hoisted(()=>({
  batch:vi.fn(),item:vi.fn(),job:vi.fn(),project:vi.fn(),research:vi.fn(),inspect:vi.fn(),authority:vi.fn(),
  send:vi.fn(),recover:vi.fn(),push:vi.fn(),subscribe:()=>()=>{},
}));
const router:AppRouterInstance={
  back:vi.fn(),forward:vi.fn(),refresh:vi.fn(),push:f.push,
  replace:vi.fn(),prefetch:vi.fn(),bfcacheId:"research-transfer-test",
};
function render(element:ReactElement) {
  return renderWithProviders(element,{wrapper:({children})=><AppRouterContext.Provider value={router}>{children}</AppRouterContext.Provider>});
}
vi.mock("@/lib/api/product-collection",()=>({readCollectionBatch:f.batch,readCollectionItem:f.item}));
vi.mock("@/lib/api/data-services",()=>({dataRequest:f.job}));
vi.mock("@/lib/api/project-center",()=>({projectRequest:f.project}));
vi.mock("@/lib/api/market-research",()=>({researchRequest:f.research}));
vi.mock("@/lib/api/research-analysis",()=>({inspectResearch:f.inspect}));
vi.mock("@/lib/api/workbench-context",()=>({fetchWorkbenchContext:f.authority}));
vi.mock("@/lib/api/supply-chain",async importOriginal=>({
  ...await importOriginal<typeof import("@/lib/api/supply-chain")>(),
  supplyCommand:f.send,readSupplyCommand:f.recover,
}));
vi.mock("../project-center/research-analysis-flow",async importOriginal=>({
  ...await importOriginal<typeof import("../project-center/research-analysis-flow")>(),ResearchAnalysisFlow:()=>null,
}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>context}));
vi.mock("../project-center/research-handoff-provider",()=>({
  researchHandoffScope:()=>researchScopeKey(context.user.id,context.effectiveOrganization.id,context.effectiveMemberId,context.permissions),
  useResearchHandoff:()=>({store,draft:useSyncExternalStore(store.subscribe,store.getSnapshot,store.getSnapshot)}),
}));

const id=(n:number)=>"92000000-0000-4000-8000-"+String(n).padStart(12,"0");
const scope={userId:"user",organizationId:"org"};
const permissions=["workbench.project.manage","workbench.project.read","workbench.collection.read","workbench.supply.manage"];
const scopeKey=researchScopeKey(scope.userId,scope.organizationId,"member",permissions);
const projectId=id(1),A=id(2),B=id(3),batchId=id(4),jobId=id(5),key=id(6);
const query={site:"us",mode:"keyword",keyword:"under sink organizer",limit:5,fields:["attributes","price","currency"]} as const;
const job={id:jobId,query:{...query,fields:[...query.fields]},batchId,saved:2,state:"SUCCEEDED",
  commandKey:id(7),createdAt:"2026-11-06T12:00:00Z",deadline:"2026-11-06T12:00:00Z",
  discovered:true,canceled:false,failed:0,pending:0,confirmedFen:0,pendingFen:0} satisfies DataJob;
const nextJob={...job,id:id(8),batchId:id(9),query:{...job.query,keyword:"under sink mat"}};
const choices=[A,B].map(candidateId=>({candidateId,expectedRevision:"1",linkId:candidateId,evidenceDigest:"a".repeat(64),
  selection:{itemId:candidateId,expectedRevision:"2",originalPublicationId:candidateId,originalVersion:candidateId===A?"3":"1"}}));
const rows=choices.map((choice,index)=>({
  ref:{ItemID:choice.selection.itemId,ItemRevision:"2",ProductKey:choice.candidateId,PublicationID:choice.selection.originalPublicationId,CatalogVersion:choice.selection.originalVersion},
  research:{CandidateID:choice.candidateId,CandidateRevision:"1",SelectedLinkID:choice.linkId,EvidenceDigest:choice.evidenceDigest},
  observation:{asin:index===0?"B000000001":"B000000002",price:index===0?29.99:24.99,currency:"USD",priceBasis:"displayed_listing_offer"},
  facts:[{id:"fact:"+String(index+1).repeat(64),path:"attributes.0",value:{Name:"宽度",Value:index===0?"38–55 cm":"42 cm"}}],missing:[],
}));
const inspection={projectId,candidates:rows};
const candidates=choices.map((choice,index)=>({
  id:choice.candidateId,revision:1,selectedLinkId:choice.linkId,evidenceDigest:choice.evidenceDigest,evidence:[],history:[],
  state:index===0?"APPROVED_FOR_TEST":"TO_RESEARCH",
  observation:{site:"us",batchId,query:{...job.query,asinCount:0},
    asin:rows[index].observation.asin,title:index===0?"Adjustable organizer":"Fixed organizer",missing:[],
    ref:{jobId,selection:{itemId:choice.selection.itemId,revision:2,publicationId:choice.selection.originalPublicationId,version:choice.selection.originalVersion}}},
})) as unknown as ResearchCandidate[];
const currentBatch={id:batchId,name:"Q1",revision:4,count:2,supplyTransferSupported:true};
const originalItem={id:A,batchId,title:"Adjustable organizer",revision:2,source:{publicationId:A,version:"3"}};
let store:ReturnType<typeof createResearchHandoff>;
const context={
  user:{id:scope.userId},effectiveOrganization:{id:scope.organizationId},effectiveMemberId:"member",permissions, collectionIntentReady:true,pendingCollectionIntent:null,supplyIntentReady:true,
  pendingSupplyIntent:null as SupplyIntent|null,registerOrganizationSwitchGuard:f.subscribe,
  setPendingSupplyIntent(intent:SupplyIntent|null) {
    if (!saveSupplyIntent(intent)) return false;
    context.pendingSupplyIntent=intent;
    return true;
  },
};

beforeEach(()=>{
  vi.resetAllMocks();sessionStorage.clear();context.pendingSupplyIntent=null;store=createResearchHandoff();
  context.user={id:scope.userId};context.effectiveOrganization={id:scope.organizationId};context.effectiveMemberId="member";context.permissions=permissions;
  f.batch.mockResolvedValue(currentBatch);f.item.mockResolvedValue({item:originalItem});
  f.job.mockImplementation((_scope,path:string)=>Promise.resolve(path==="amazon/jobs"?[job,nextJob]:job));
  f.project.mockResolvedValue({id:projectId,kind:"PRODUCT_RESEARCH",archived:false});
  f.research.mockResolvedValue({items:candidates,incomplete:false});
  f.inspect.mockResolvedValue(inspection);
  f.authority.mockImplementation(async()=>({user:context.user,effectiveOrganizationId:context.effectiveOrganization.id,
    effectiveMemberId:context.effectiveMemberId,organizations:[{id:context.effectiveOrganization.id,permissions:context.permissions}],selectionRequired:false}));
});
afterEach(cleanup);

function approved() {
  const identity=researchScopeKey(context.user.id,context.effectiveOrganization.id,context.effectiveMemberId,context.permissions);
  const draft=store.start(identity,projectId,job);
  const active={...draft,confirmed:{representative:A,approvalRevision:1,choices,after:"",confirmedAt:"2026-11-06T12:00:00Z",
    requirements:{use:"水槽下收纳",specs:[{factId:rows[0].facts[0].id,required:"可调宽",difference:"最小 40 cm"}],missing:"",targetPrice:""}},
    transfer:{key,revision:4}};
  store.replace(draft,active);
  return handoffToken(active);
}

async function confirmButton() {
  const button=await screen.findByRole("button",{name:"确认加入所选商品"});
  await waitFor(()=>expect(button).toBeEnabled());
  return button;
}

it("W1 lets an operator without collection execution access read a saved job and open the same project",async()=>{
  const research=render(<ResearchPanel scope={scope} projectId={projectId} archived={false} view="research"/>);
  const jobs=await screen.findByLabelText("本次查询");
  await waitFor(()=>expect(jobs).toBeEnabled());
  fireEvent.change(jobs,{target:{value:jobId}});
  fireEvent.click(screen.getByRole("checkbox",{name:"Adjustable organizer"}));
  fireEvent.click(screen.getByRole("checkbox",{name:"Fixed organizer"}));
  const inspect=screen.getByRole("button",{name:"核对所选原件"});
  await waitFor(()=>expect(inspect).toBeEnabled());
  fireEvent.click(inspect);
  fireEvent.change(await screen.findByLabelText("代表候选"),{target:{value:A}});
  expect(screen.getByText("样本展示价 USD 24.99–29.99")).toBeVisible();
  fireEvent.change(screen.getByLabelText("商品用途"),{target:{value:"水槽下收纳"}});
  fireEvent.click(screen.getByRole("button",{name:"添加必要规格"}));
  fireEvent.change(screen.getByLabelText("第 1 条规格依据"),{target:{value:rows[0].facts[0].id}});
  fireEvent.change(screen.getByLabelText("必要规格"),{target:{value:"可调宽"}});
  fireEvent.change(screen.getByLabelText("可接受差异"),{target:{value:"最小 40 cm"}});
  fireEvent.click(screen.getByRole("button",{name:"确认并进入选品"}));
  await waitFor(()=>expect(f.push).toHaveBeenCalledWith("/workbench/supply/selection/"+projectId+"?handoff="+handoffToken(store.getSnapshot()!)));
  expect(store.getSnapshot()?.confirmed).toMatchObject({representative:A,choices,
    requirements:{specs:[{factId:rows[0].facts[0].id,required:"可调宽",difference:"最小 40 cm"}]}});
  expect(screen.queryByRole("link",{name:"前往数据市场"})).toBeNull();
  expect(f.job.mock.calls.every(call=>call[3]===undefined)).toBe(true);
  research.unmount();
  render(<ResearchSelectionGate scope={scope} projectId={projectId} archived={false} token={handoffToken(store.getSnapshot()!)}>
    {()=> <p>当前项目已接收选品方向</p>}
  </ResearchSelectionGate>);
  expect(await screen.findByText("当前项目已接收选品方向")).toBeVisible();
});

it("W1 sends only A after live validation, then keeps the original key lookup-only after lost response and refresh",async()=>{
  const token=approved();
  f.send.mockRejectedValue(new SupplyAPIError("OUTCOME_UNKNOWN",503));
  f.recover.mockRejectedValue(new SupplyAPIError("NOT_FOUND",404));
  const view=render(<ResearchCollectionTransfer scope={scope} token={token} supplyAvailable/>);
  const button=await confirmButton();
  expect(screen.queryByText(/整批转入|清除预填/)).toBeNull();
  fireEvent.click(button);fireEvent.click(button);
  await waitFor(()=>expect(f.send).toHaveBeenCalledTimes(1));
  const intent={...scope,key,route:"transfer",lookupOnly:true,command:{batchId,expectedRevision:4,itemIds:[A]}};
  expect(f.send).toHaveBeenCalledWith(intent,expect.any(AbortSignal));
  expect(f.authority).toHaveBeenCalledTimes(1);
  expect(f.inspect).toHaveBeenCalledWith(scope,projectId,{candidates:choices},expect.any(AbortSignal));
  view.unmount();
  store=createResearchHandoff();
  context.pendingSupplyIntent=loadSupplyIntent();
  expect(context.pendingSupplyIntent).toEqual(intent);
  render(<ResearchCollectionTransfer scope={scope} token={token} supplyAvailable/>);
  fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));
  await screen.findByText(/尚未读取到原操作/);
  expect(f.recover).toHaveBeenCalledWith(intent,expect.any(AbortSignal));
  expect(screen.queryByRole("button",{name:"重试原请求"})).toBeNull();
  expect(screen.queryByRole("button",{name:/确认加入/})).toBeNull();
  expect(f.send).toHaveBeenCalledTimes(1);
});

it("consumes a successful handoff so reopening the same entry cannot create a second request",async()=>{
  const token=approved();
  f.send.mockResolvedValue({preparation:{id:id(10),sourceBatchId:batchId,sourceRevision:4,name:"Q1",count:1,
    revision:1,createdAt:"2026-11-06T12:00:00Z"},replayed:false});
  const first=render(<ResearchCollectionTransfer scope={scope} token={token} supplyAvailable/>);
  fireEvent.click(await confirmButton());
  expect(await screen.findByRole("link",{name:"查看待上传商品"})).toHaveAttribute("href","/workbench/listing/waiting?preparation="+id(10));
  expect(context.pendingSupplyIntent).toBeNull();
  first.unmount();
  render(<ResearchCollectionTransfer scope={scope} token={token} supplyAvailable/>);
  expect(screen.getByText(/本次确认已使用/)).toBeVisible();
  expect(screen.queryByRole("button",{name:/确认加入/})).toBeNull();
  expect(f.send).toHaveBeenCalledTimes(1);
});

const completed={preparation:{id:id(10),sourceBatchId:batchId,sourceRevision:4,name:"Q1",count:1,
  revision:1,createdAt:"2026-11-06T12:00:00Z"},replayed:false};

function switchContext(kind:"organization"|"member"|"permissions") {
  if (kind==="organization") context.effectiveOrganization={id:"another-org"};
  if (kind==="member") context.effectiveMemberId="another-member";
  if (kind==="permissions") context.permissions=context.permissions.filter(permission=>permission!=="workbench.supply.manage");
}

it.each(["organization","member","permissions"] as const)("masks a successful transfer receipt when %s changes on the same route",async(kind)=>{
  context.permissions=[...permissions,"workbench.data-market.use"];
  const token=approved();
  f.send.mockResolvedValue(completed);
  const page=()=> <CollectionPage researchToken={token} supplyAvailable/>;
  const view=render(page());
  fireEvent.click(await confirmButton());
  expect(await screen.findByRole("link",{name:"查看待上传商品"})).toHaveAttribute("href","/workbench/listing/waiting?preparation="+id(10));
  switchContext(kind);
  view.rerender(page());
  expect(screen.queryByRole("link",{name:"查看待上传商品"})).toBeNull();
  expect(screen.queryByText("已转入 1 件。")).toBeNull();
  expect(screen.queryByRole("button",{name:/确认加入/})).toBeNull();
});

it("ignores a late lookup response from the previous member without resending the pending command",async()=>{
  context.permissions=[...permissions,"workbench.data-market.use"];
  const token=approved();
  const intent={...scope,key,route:"transfer",lookupOnly:true,command:{batchId,expectedRevision:4,itemIds:[A]}} as const;
  context.setPendingSupplyIntent(intent);
  let complete!:()=>void;
  f.recover.mockImplementation(()=>new Promise(resolve=>{complete=()=>resolve(completed);}));
  const page=()=> <CollectionPage researchToken={token} supplyAvailable/>;
  const view=render(page());
  fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));
  await waitFor(()=>expect(f.recover).toHaveBeenCalledTimes(1));
  switchContext("member");
  view.rerender(page());
  await act(async()=>complete());
  expect(screen.queryByRole("link",{name:"查看待上传商品"})).toBeNull();
  expect(loadSupplyIntent()).toEqual(intent);
  expect(f.send).not.toHaveBeenCalled();
});

it.each(["query","permission","approval","original","missing-token"] as const)("W2 stops a stale transfer after %s changes",async(reason)=>{
  const token=approved();
  if (reason==="missing-token") {
    render(<ResearchCollectionTransfer scope={scope} token="" supplyAvailable/>);
    expect(screen.getByRole("alert")).toHaveTextContent("交接已失效");
    expect(screen.queryByRole("button",{name:/确认加入/})).toBeNull();
    expect(f.send).not.toHaveBeenCalled();
    return;
  }
  const view=render(<ResearchCollectionTransfer scope={scope} token={token} supplyAvailable/>);
  const button=await confirmButton();
  if (reason==="permission") f.authority.mockResolvedValue({user:{id:scope.userId},effectiveOrganizationId:scope.organizationId,
    effectiveMemberId:"member",organizations:[{id:scope.organizationId,permissions:permissions.slice(1)}],selectionRequired:false});
  if (reason==="approval") f.research.mockResolvedValue({items:[{...candidates[0],state:"TO_RESEARCH"},candidates[1]],incomplete:false});
  if (reason==="original") f.inspect.mockResolvedValue({...inspection,candidates:[{...rows[0],ref:{...rows[0].ref,CatalogVersion:"4"}},rows[1]]});
  if (reason==="query") {
    let complete!:()=>void;
    f.authority.mockImplementation(()=>new Promise(resolve=>{complete=()=>resolve({user:{id:scope.userId},effectiveOrganizationId:scope.organizationId,
      effectiveMemberId:"member",organizations:[{id:scope.organizationId,permissions}],selectionRequired:false});}));
    fireEvent.click(button);
    await waitFor(()=>expect(complete).toBeTypeOf("function"));
    await act(async()=>{store.start(scopeKey,projectId,nextJob);complete();});
  } else fireEvent.click(button);
  await screen.findByRole("alert");
  expect(f.send).not.toHaveBeenCalled();
  expect(screen.queryByRole("button",{name:/确认加入/})).toBeNull();
  view.unmount();
});

it("W3 identifies missing evidence before any handoff",async()=>{
  const onlyTitle={...candidates[0],observation:{...candidates[0].observation,missing:["attributes","price"]}} as ResearchCandidate;
  render(<ResearchOpportunity scope={scope} scopeKey={scopeKey} projectId={projectId} store={store}
    draft={store.start(scopeKey,projectId,job)} selected={[onlyTitle]} after="" blocked={false} onJobChange={()=>{}}/>);
  expect(await screen.findByText(/待补证/)).toBeVisible();
  expect(screen.getByRole("button",{name:"核对所选原件"})).toBeDisabled();
  expect(screen.queryByRole("button",{name:"确认并进入选品"})).toBeNull();
  expect(f.inspect).not.toHaveBeenCalled();
});
