import {beforeEach,afterEach,it,expect,vi} from "vitest";
import {render,screen,fireEvent,waitFor,cleanup} from "@testing-library/react";
import {ResearchPanel} from "./research-panel";
import {ResearchError} from "@/lib/api/market-research";
const mock=vi.hoisted(()=>({read:vi.fn(),context:{permissions:["workbench.collection.read","workbench.project.manage"],registerOrganizationSwitchGuard:()=>()=>{}}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>mock.context}));
vi.mock("@/lib/api/market-research",async original=>({...await original<object>(),researchRequest:mock.read}));
vi.mock("../pod/collection-picker",()=>({CollectionPicker:({onSelect,isEligible}:{onSelect:(i:unknown)=>void;isEligible:(i:unknown)=>boolean})=>isEligible({source:{kind:"amazon_data"},id:"22222222-2222-4222-8222-222222222222"})?<button onClick={()=>onSelect({id:"22222222-2222-4222-8222-222222222222",revision:1,source:{kind:"amazon_data",publicationId:id,version:"1"}})}>核实插件来源</button>:<p>不支持来源已排除</p>}));
vi.mock("./research-analysis-flow",async original=>({...await original<object>(),ResearchAnalysisFlow:()=> <h3>AI分析建议</h3>}));
vi.mock("./research-comparison",()=>({ResearchComparison:()=>null}));
const id="11111111-1111-4111-8111-111111111111";
const candidate={id,revision:2,state:"TO_RESEARCH",selectedLinkId:id,evidenceDigest:"a".repeat(64),observation:{title:"真实保存商品",site:"us",asin:"B000123456",sourceUrl:"https://www.amazon.com/dp/B000123456",capturedAt:"2026-11-02T00:00:00Z",parserVersion:"amazon-v1",missing:["rating"],ref:{checksum:"original",selection:{itemId:id,revision:1,publicationId:id,version:"1"}},item:{id,revision:1,source:{kind:"amazon_data",version:"1",publicationId:id}}},evidence:[],history:[]};
beforeEach(()=>{sessionStorage.clear();mock.read.mockReset().mockResolvedValue({items:[candidate],incomplete:false});});afterEach(cleanup);
it("places optional analysis before the independent human decision in selection review",async()=>{
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="selection"/>);
 const decision=await screen.findByRole("button",{name:"开始评估"});
 expect(screen.getByRole("heading",{name:"AI分析建议"}).compareDocumentPosition(decision)&Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
});
it("shows original missing facts without dispatching a command on open",async()=>{render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false}/>);expect(await screen.findByText("真实保存商品")).toBeInTheDocument();expect(screen.getByText(/rating/)).toBeInTheDocument();expect(mock.read.mock.calls.every(c=>!c[4]?.intent)).toBe(true);});
it("focuses the requested function while keeping selected candidates and avoiding commands between steps",async()=>{
 const scope={userId:"u",organizationId:"o"};
 const view=render(<ResearchPanel scope={scope} projectId={id} archived={false} view="selection" workflowStep="decisions"/>);
 await screen.findByRole("button",{name:"开始评估"});
 await waitFor(()=>expect(document.getElementById("research-decisions")).toHaveFocus());
 fireEvent.click(screen.getByRole("checkbox"));expect(screen.getByRole("checkbox")).toBeChecked();
 view.rerender(<ResearchPanel scope={scope} projectId={id} archived={false} view="selection" workflowStep="transfer"/>);
 await waitFor(()=>expect(document.getElementById("research-transfer")).toHaveFocus());
 expect(screen.getByRole("checkbox")).toBeChecked();expect(mock.read.mock.calls.every(c=>!c[4]?.intent)).toBe(true);
});
it("keeps the original unknown write and recovers it by readonly key lookup",async()=>{mock.read.mockImplementation(async(_s,_p,_z,_a,options)=>{if(options?.intent)throw new ResearchError("OUTCOME_UNKNOWN");if(options?.lookup)return {key:options.lookup,candidateIds:[id],replayed:true};return {items:[candidate],incomplete:false};});render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false}/>);fireEvent.click(await screen.findByRole("button",{name:"开始评估"}));await screen.findByRole("button",{name:"查询操作结果"});const original=mock.read.mock.calls.find(c=>c[4]?.intent)?.[4].intent;fireEvent.click(screen.getByRole("button",{name:"查询操作结果"}));await waitFor(()=>expect(mock.read.mock.calls.some(c=>c[4]?.lookup===original.key)).toBe(true));expect(mock.read.mock.calls.filter(c=>c[4]?.intent)).toHaveLength(1);});
it("masks unavailable evidence and keeps archived projects readonly",async()=>{mock.read.mockResolvedValue({items:[{...candidate,observation:undefined,evidenceDigest:undefined}],incomplete:false});render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived/>);expect(await screen.findByText("这份资料暂时无法查看")).toBeInTheDocument();expect(screen.queryByText("真实保存商品")).not.toBeInTheDocument();expect(screen.queryByRole("button",{name:"开始评估"})).not.toBeInTheDocument();});

it("clears a definite unsupported-origin rejection and lets research continue",async()=>{
 mock.read.mockImplementation(async(_s,_p,_z,_a,options)=>{if(options?.intent)throw new ResearchError("SOURCE_NOT_SUPPORTED");return {items:[candidate],incomplete:false};});
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false}/>);
 await screen.findByText("真实保存商品");fireEvent.click(screen.getByRole("button",{name:"添加已有商品资料"}));fireEvent.click(screen.getByRole("button",{name:"核实插件来源"}));
 await screen.findByText(/这份资料暂不能用于调研/);
 await waitFor(()=>expect(screen.queryByRole("button",{name:"查询操作结果"})).not.toBeInTheDocument());
 expect(screen.getByRole("button",{name:"开始评估"})).toBeEnabled();expect(screen.getByText("不支持来源已排除")).toBeInTheDocument();
});

it("carries exact selected observations into the other center and rejects stale handoff",async()=>{
 const other="33333333-3333-4333-8333-333333333333",second={...candidate,id:other,selectedLinkId:other,observation:{...candidate.observation,title:"第二项观察",ref:{...candidate.observation.ref,selection:{...candidate.observation.ref.selection,itemId:other}}}};
 mock.read.mockResolvedValue({items:[candidate,second],incomplete:false});
 const view=render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="research"/>);
 await screen.findByText("真实保存商品");for(const checkbox of screen.getAllByRole("checkbox"))fireEvent.click(checkbox);
 const href=screen.getByRole("link",{name:"进入智能选品"}).getAttribute("href")!,sample=new URL(href,"https://app.example").searchParams.get("sample")!;
 expect(JSON.parse(sample).map((c:{candidateId:string})=>c.candidateId)).toEqual([id,other]);expect(JSON.parse(sample)[0].evidenceDigest).toBe(candidate.evidenceDigest);
 view.unmount();mock.read.mockResolvedValue({items:[{...candidate,revision:3},second],incomplete:false});
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="selection" sample={sample}/>);
 await screen.findByText("所选商品资料已更新或暂时无法查看，请重新选择。");expect(screen.getAllByRole("checkbox").every(v=>!(v as HTMLInputElement).checked)).toBe(true);
});
