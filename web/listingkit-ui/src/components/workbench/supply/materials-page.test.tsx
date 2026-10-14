import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {MaterialsPage} from "./materials-page";
import {initialSupplyDraft} from "./draft-fields";
import {SupplyAPIError} from "@/lib/api/supply-chain";
const f=vi.hoisted(()=>({record:vi.fn(),prepare:vi.fn(),start:vi.fn(),read:vi.fn(),resume:vi.fn(),adopt:vi.fn(),adoption:vi.fn(),mode:"default",params:new URLSearchParams(),guards:[] as Array<()=>boolean>}));
const id="550e8400-e29b-41d4-a716-446655440000",key="550e8400-e29b-41d4-a716-446655440001";
const context=vi.hoisted(()=>({user:{id:"actor"},effectiveOrganization:{id:"org",name:"测试企业"},permissions:["workbench.agent.use","local_agent.write","workbench.supply.read"],isLoading:false,isSwitching:false,error:null,blockingError:null,selectionRequired:false,registerOrganizationSwitchGuard:(check:()=>boolean)=>{f.guards.push(check);return()=>{f.guards=f.guards.filter(v=>v!==check)}}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>context}));
vi.mock("next/navigation",()=>({useSearchParams:()=>f.params}));
vi.mock("next/link",()=>({default:({href,children}:React.ComponentProps<"a">)=><a href={href}>{children}</a>}));
vi.mock("@/lib/api/supply-chain",async importOriginal=>({...await importOriginal<typeof import("@/lib/api/supply-chain")>(),readSupplyRecord:(...a:unknown[])=>f.record(...a)}));
vi.mock("@/lib/api/listing-materials",()=>({prepareMaterials:(...a:unknown[])=>f.prepare(...a),startMaterials:(...a:unknown[])=>f.start(...a),readMaterials:(...a:unknown[])=>f.read(...a),resumeMaterials:(...a:unknown[])=>f.resume(...a),adoptMaterials:(...a:unknown[])=>f.adopt(...a),readMaterialsAdoption:(...a:unknown[])=>f.adoption(...a)}));
vi.mock("../agents/agent-configuration-hooks",()=>({configurationError:()=>"配置不可用",useConfigurationRead:(_scope:unknown,path:string)=>({data:path.includes("execution-modes")?{mode:f.mode,revision:"1"}:path.includes("templates?")?{items:[]}: {canUse:true,agent:{activation:"ENABLED"},capabilities:[{support:"REQUIRED",readiness:"AVAILABLE"}]}})}));
const draft=initialSupplyDraft({product:{}} as Parameters<typeof initialSupplyDraft>[0]);
const base={id,targetId:id,revision:1,effectiveVersion:"1",input:{sourceId:id,storeId:id,expectedRevision:0,effectiveVersion:"1",draft},merchant:{site:"shein-us"},source:{id},result:{ready_for_upload:false,issues:[]}};
const ref={Kind:"agent-configuration-v1",ID:key,Digest:"a".repeat(64)},modeRef={Kind:"agent-execution-mode-v1",ID:key,Digest:"b".repeat(64)};
function run(requestKey=key){return {runId:key,requestKey,recordId:id,phase:"human_review_required",revision:"2",configurationSnapshotRef:ref,modeSnapshotRef:modeRef,executionMode:f.mode,modeRevision:"1",before:base.input,after:{...base.input,draft:{...draft,product:{...draft.product,brand_code:"fixture-brand"}}},proposal:{brandCode:"fixture-brand",evidenceIds:["observed"]},candidateHash:"c".repeat(64),unresolved:["类目待补齐"],canRequestAdoption:f.mode!=="read_only",steps:[],tokens:30,estimatedCostMicros:20,currency:"CNY",usageStatus:"observed"}}
beforeEach(()=>{
 f.mode="default";f.params=new URLSearchParams();f.guards=[];vi.clearAllMocks();window.history.replaceState(null,"",`/workbench/supply/materials/${id}`);
 Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true}});Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false}});
 f.record.mockResolvedValue(base);
 f.prepare.mockImplementation(async(_scope,_record,input)=>({recordId:id,requestKey:input.requestKey,configurationSnapshotRef:ref,modeSnapshotRef:modeRef,modeRevision:"1",templateSelection:null}));
 f.start.mockImplementation(async(_scope,_record,requestKey)=>run(requestKey));f.read.mockResolvedValue(run());
});
afterEach(()=>cleanup());
async function generate(){fireEvent.click(await screen.findByRole("button",{name:"准备平台资料"}));await screen.findByRole("heading",{name:"确认准备平台资料"});fireEvent.click(screen.getByRole("button",{name:"确认生成建议"}));}
it("shows the exact proposal as unsaved until the original Target receipt is confirmed",async()=>{
 f.adopt.mockResolvedValue({record:{...base,id:key,revision:2,input:run().after},replayed:false});
 render(<MaterialsPage recordId={id}/>);await generate();await screen.findByText("建议差异 · 尚未保存");
 expect(f.adopt).not.toHaveBeenCalled();expect(screen.getByRole("button",{name:"确认保存本地草稿"})).toBeDisabled();
 fireEvent.click(screen.getByRole("checkbox"));fireEvent.click(screen.getByRole("button",{name:"确认保存本地草稿"}));
 await screen.findByText("本地草稿已保存 · v2");expect(screen.queryByText("建议差异 · 尚未保存")).toBeNull();
 const original=f.start.mock.calls[0][2];expect(f.adopt.mock.calls[0][2]).toBe(original);expect(f.adopt.mock.calls[0][3]).toEqual({configurationSnapshotRef:ref,candidateHash:"c".repeat(64),humanConfirmed:true});
});
it("keeps a lost start's exact UUID/configuration and blocks enterprise switch until original read",async()=>{
 f.start.mockRejectedValueOnce(new SupplyAPIError("OUTCOME_UNKNOWN",503));
 render(<MaterialsPage recordId={id}/>);await generate();await screen.findByRole("alert");
 const original=f.start.mock.calls[0][2],stored=JSON.parse(new URL(window.location.href).searchParams.get("materials_intent")!);
 expect(stored.key).toBe(original);expect(stored.prepared.configurationSnapshotRef).toEqual(ref);expect(f.guards.some(g=>!g())).toBe(true);
 f.read.mockResolvedValue(run(original));fireEvent.click(screen.getByRole("button",{name:"核对原运行"}));await screen.findByText("建议差异 · 尚未保存");expect(f.start).toHaveBeenCalledTimes(1);expect(f.read.mock.calls[0][2]).toBe(original);expect(f.guards.every(g=>g())).toBe(true);
});
it("replays a lost unclaimed Start with its original prepared reference after NOT_FOUND",async()=>{
 f.start.mockRejectedValueOnce(new SupplyAPIError("OUTCOME_UNKNOWN",503));f.read.mockRejectedValueOnce(new SupplyAPIError("NOT_FOUND",404));
 render(<MaterialsPage recordId={id}/>);await generate();await screen.findByRole("alert");
 const first=f.start.mock.calls[0];fireEvent.click(screen.getByRole("button",{name:"核对原运行"}));
 await screen.findByText("原结果尚未找到；这不表示可以更换编号重试。");
 fireEvent.click(screen.getByRole("button",{name:"重试原确认与请求"}));await screen.findByText("建议差异 · 尚未保存");
 expect(f.prepare).toHaveBeenCalledTimes(1);expect(f.start).toHaveBeenCalledTimes(2);expect(f.start.mock.calls[1].slice(0,4)).toEqual(first.slice(0,4));
});
it("restores read-only history without generating a model call or offering adoption",async()=>{
 f.mode="read_only";f.params=new URLSearchParams({agent_key:key,agent_actor:"actor",agent_org:"org"});f.read.mockResolvedValue(run());
 render(<MaterialsPage recordId={id}/>);await screen.findByText("建议差异 · 尚未保存");expect(screen.queryByRole("button",{name:"确认保存本地草稿"})).toBeNull();expect(f.start).not.toHaveBeenCalled();expect(f.prepare).not.toHaveBeenCalled();
});
it("does not turn a damaged retained intent into a fresh execution",async()=>{
 f.params=new URLSearchParams({materials_intent:"damaged"});render(<MaterialsPage recordId={id}/>);
 await screen.findByText("原确认资料不可读，请保留原编号核对。");expect(f.prepare).not.toHaveBeenCalled();expect(f.start).not.toHaveBeenCalled();expect(screen.queryByRole("button",{name:"准备平台资料"})).toBeNull();
});
it("does not read a foreign identity's retained record or run",async()=>{
 f.params=new URLSearchParams({agent_key:key,agent_actor:"other",agent_org:"org"});render(<MaterialsPage recordId={id}/>);
 await screen.findByText("原请求属于另一身份或企业");expect(f.record).not.toHaveBeenCalled();expect(f.read).not.toHaveBeenCalled();
});
it("restores a lost resume's exact feedback and revision for retry",async()=>{
 const interrupted={...run(),phase:"interrupted",revision:"3",canRequestAdoption:false};
 const input={configurationSnapshotRef:ref,revision:"3",feedback:"原核对意见"};
 f.params=new URLSearchParams({agent_key:key,agent_actor:"actor",agent_org:"org",materials_mutation:JSON.stringify({recordId:id,userId:"actor",organizationId:"org",key,action:"resume",input})});
 f.read.mockResolvedValue(interrupted);f.resume.mockResolvedValue({...run(),revision:"4"});
 render(<MaterialsPage recordId={id}/>);await screen.findByText("等待继续");
 expect(screen.getByRole("textbox")).toBeDisabled();
 fireEvent.click(screen.getByRole("button",{name:"重试原继续请求"}));
 await waitFor(()=>expect(f.resume).toHaveBeenCalledTimes(1));expect(f.resume.mock.calls[0][3]).toEqual(input);
 await screen.findByText("建议差异 · 尚未保存");expect(f.guards.every(g=>g())).toBe(true);
});
it("restores a lost save and only replays the original exact candidate",async()=>{
 const input={configurationSnapshotRef:ref,candidateHash:"c".repeat(64),humanConfirmed:true};
 f.params=new URLSearchParams({agent_key:key,agent_actor:"actor",agent_org:"org",materials_mutation:JSON.stringify({recordId:id,userId:"actor",organizationId:"org",key,action:"adopt",input})});
 f.adopt.mockResolvedValue({record:{...base,id:key,revision:2,input:run().after},replayed:true});
 render(<MaterialsPage recordId={id}/>);await screen.findByText("建议差异 · 尚未保存");
 expect(screen.getByRole("checkbox")).toBeDisabled();expect(f.guards.some(g=>!g())).toBe(true);
 fireEvent.click(screen.getByRole("button",{name:"重试原保存请求"}));await screen.findByText("本地草稿已保存 · v2");
 expect(f.adopt.mock.calls[0][3]).toEqual(input);expect(f.guards.every(g=>g())).toBe(true);
});
it("allows original reads but no writes when a retained mutation is damaged",async()=>{
 f.params=new URLSearchParams({agent_key:key,agent_actor:"actor",agent_org:"org",materials_mutation:"damaged"});
 render(<MaterialsPage recordId={id}/>);await screen.findByText("建议差异 · 尚未保存");
 fireEvent.click(screen.getByRole("checkbox"));
 expect(screen.getByRole("button",{name:"确认保存本地草稿"})).toBeDisabled();
 expect(screen.getByRole("button",{name:"核对原运行"})).toBeEnabled();expect(f.adopt).not.toHaveBeenCalled();
});
