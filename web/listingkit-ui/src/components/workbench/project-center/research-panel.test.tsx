import {beforeEach,afterEach,it,expect,vi} from "vitest";
import {render,screen,fireEvent,waitFor,cleanup} from "@testing-library/react";
import {ResearchPanel} from "./research-panel";
import {ResearchError} from "@/lib/api/market-research";
const mock=vi.hoisted(()=>({read:vi.fn(),context:{permissions:["workbench.collection.read","workbench.project.manage"],registerOrganizationSwitchGuard:()=>()=>{}}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>mock.context}));
vi.mock("@/lib/api/market-research",async original=>({...await original<object>(),researchRequest:mock.read}));
vi.mock("../pod/collection-picker",()=>({CollectionPicker:({onSelect,isEligible}:{onSelect:(i:unknown)=>void;isEligible:(i:unknown)=>boolean})=>isEligible({source:{kind:"amazon_data"},id:"22222222-2222-4222-8222-222222222222"})?<button onClick={()=>onSelect({id:"22222222-2222-4222-8222-222222222222",revision:1,source:{kind:"amazon_data",publicationId:id,version:"1"}})}>核实插件来源</button>:<p>不支持来源已排除</p>}));
vi.mock("./research-comparison",()=>({ResearchComparison:()=>null}));
const id="11111111-1111-4111-8111-111111111111";
const candidate={id,revision:2,state:"TO_RESEARCH",selectedLinkId:id,evidenceDigest:"a".repeat(64),observation:{title:"真实保存商品",site:"us",asin:"B000123456",sourceUrl:"https://www.amazon.com/dp/B000123456",capturedAt:"2026-11-02T00:00:00Z",parserVersion:"amazon-v1",missing:["rating"],ref:{checksum:"original",selection:{itemId:id,revision:1,publicationId:id,version:"1"}},item:{id,revision:1,source:{kind:"amazon_data",version:"1",publicationId:id}}},evidence:[],history:[]};
beforeEach(()=>{sessionStorage.clear();mock.read.mockReset().mockResolvedValue({items:[candidate],incomplete:false});});afterEach(cleanup);
it("shows original missing facts without dispatching a command on open",async()=>{render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false}/>);expect(await screen.findByText("真实保存商品")).toBeInTheDocument();expect(screen.getByText(/rating/)).toBeInTheDocument();expect(mock.read.mock.calls.every(c=>!c[4]?.intent)).toBe(true);});
it("keeps the original unknown write and recovers it by readonly key lookup",async()=>{mock.read.mockImplementation(async(_s,_p,_z,_a,options)=>{if(options?.intent)throw new ResearchError("OUTCOME_UNKNOWN");if(options?.lookup)return {key:options.lookup,candidateIds:[id],replayed:true};return {items:[candidate],incomplete:false};});render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false}/>);fireEvent.click(await screen.findByRole("button",{name:"提交人工审阅"}));await screen.findByRole("button",{name:"核实原操作"});const original=mock.read.mock.calls.find(c=>c[4]?.intent)?.[4].intent;fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));await waitFor(()=>expect(mock.read.mock.calls.some(c=>c[4]?.lookup===original.key)).toBe(true));expect(mock.read.mock.calls.filter(c=>c[4]?.intent)).toHaveLength(1);});
it("masks unavailable evidence and keeps archived projects readonly",async()=>{mock.read.mockResolvedValue({items:[{...candidate,observation:undefined,evidenceDigest:undefined}],incomplete:false});render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived/>);expect(await screen.findByText("来源当前不可访问")).toBeInTheDocument();expect(screen.queryByText("真实保存商品")).not.toBeInTheDocument();expect(screen.queryByRole("button",{name:"提交人工审阅"})).not.toBeInTheDocument();});

it("clears a definite unsupported-origin rejection and lets research continue",async()=>{
 mock.read.mockImplementation(async(_s,_p,_z,_a,options)=>{if(options?.intent)throw new ResearchError("SOURCE_NOT_SUPPORTED");return {items:[candidate],incomplete:false};});
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false}/>);
 await screen.findByText("真实保存商品");fireEvent.click(screen.getByRole("button",{name:"关联已保存资料"}));fireEvent.click(screen.getByRole("button",{name:"核实插件来源"}));
 await screen.findByText(/该资料不是当前支持的服务端 Amazon 结果/);
 await waitFor(()=>expect(screen.queryByRole("button",{name:"核实原操作"})).not.toBeInTheDocument());
 expect(screen.getByRole("button",{name:"提交人工审阅"})).toBeEnabled();expect(screen.getByText("不支持来源已排除")).toBeInTheDocument();
});
