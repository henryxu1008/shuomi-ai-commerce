import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {MaterialsPage} from "./materials-page";
import {initialSupplyDraft} from "./draft-fields";
import {SupplyAPIError} from "@/lib/api/supply-chain";
const f=vi.hoisted(()=>({record:vi.fn(),rules:vi.fn(),prepare:vi.fn(),start:vi.fn(),read:vi.fn(),resume:vi.fn(),adopt:vi.fn(),adoption:vi.fn(),mode:"default",params:new URLSearchParams(),guards:[] as Array<()=>boolean>}));
const id="550e8400-e29b-41d4-a716-446655440000",key="550e8400-e29b-41d4-a716-446655440001";
const context=vi.hoisted(()=>({user:{id:"actor"},effectiveOrganization:{id:"org",name:"测试企业"},permissions:["workbench.agent.use","local_agent.write","workbench.supply.read"],isLoading:false,isSwitching:false,error:null,blockingError:null,selectionRequired:false,registerOrganizationSwitchGuard:(check:()=>boolean)=>{f.guards.push(check);return()=>{f.guards=f.guards.filter(v=>v!==check)}}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>context}));
vi.mock("next/navigation",()=>({useSearchParams:()=>f.params}));
vi.mock("next/link",()=>({default:({href,children}:React.ComponentProps<"a">)=><a href={href}>{children}</a>}));
vi.mock("@/lib/api/supply-chain",async importOriginal=>({...await importOriginal<typeof import("@/lib/api/supply-chain")>(),readSupplyRecord:(...a:unknown[])=>f.record(...a),supplyRules:(...a:unknown[])=>f.rules(...a)}));
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
 f.rules.mockResolvedValue({rules:{categories:[{category_id:123,product_type_id:456,category_name:"候选类目",last_category:true,children:[]}]}});
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
 f.read.mockResolvedValue(run(original));fireEvent.click(screen.getByRole("button",{name:"核对原运行"}));await screen.findByText("建议差异 · 尚未保存");expect(f.start).toHaveBeenCalledTimes(1);expect(f.read.mock.calls[0][2]).toBe(original);await waitFor(()=>expect(f.guards.every(g=>g())).toBe(true));
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
 const interrupted={...run(),phase:"interrupted",revision:"3",canResume:true,canRequestAdoption:false};
 const input={configurationSnapshotRef:ref,revision:"3",feedback:"原核对意见"};
 f.params=new URLSearchParams({agent_key:key,agent_actor:"actor",agent_org:"org",materials_mutation:JSON.stringify({recordId:id,userId:"actor",organizationId:"org",key,action:"resume",input})});
 f.read.mockResolvedValue(interrupted);f.resume.mockResolvedValue({...run(),revision:"4"});
 render(<MaterialsPage recordId={id}/>);await screen.findByText("等待继续");
 expect(screen.getByRole("textbox")).toBeDisabled();
 fireEvent.click(screen.getByRole("button",{name:"重试原继续请求"}));
 await waitFor(()=>expect(f.resume).toHaveBeenCalledTimes(1));expect(f.resume.mock.calls[0][3]).toEqual(input);
 await screen.findByText("建议差异 · 尚未保存");await waitFor(()=>expect(f.guards.every(g=>g())).toBe(true));
});
it("restores a lost save and only replays the original exact candidate",async()=>{
 const input={configurationSnapshotRef:ref,candidateHash:"c".repeat(64),humanConfirmed:true};
 f.params=new URLSearchParams({agent_key:key,agent_actor:"actor",agent_org:"org",materials_mutation:JSON.stringify({recordId:id,userId:"actor",organizationId:"org",key,action:"adopt",input})});
 f.adopt.mockResolvedValue({record:{...base,id:key,revision:2,input:run().after},replayed:true});
 render(<MaterialsPage recordId={id}/>);await screen.findByText("建议差异 · 尚未保存");
 expect(screen.getByRole("checkbox")).toBeDisabled();expect(f.guards.some(g=>!g())).toBe(true);
 fireEvent.click(screen.getByRole("button",{name:"重试原保存请求"}));await screen.findByText("本地草稿已保存 · v2");
 expect(f.adopt.mock.calls[0][3]).toEqual(input);await waitFor(()=>expect(f.guards.every(g=>g())).toBe(true));
});
it("allows original reads but no writes when a retained mutation is damaged",async()=>{
 f.params=new URLSearchParams({agent_key:key,agent_actor:"actor",agent_org:"org",materials_mutation:"damaged"});
 render(<MaterialsPage recordId={id}/>);await screen.findByText("建议差异 · 尚未保存");
 fireEvent.click(screen.getByRole("checkbox"));
 expect(screen.getByRole("button",{name:"确认保存本地草稿"})).toBeDisabled();
 expect(screen.getByRole("button",{name:"核对原运行"})).toBeEnabled();expect(f.adopt).not.toHaveBeenCalled();
});
it("shows recorded image advice without preparing, generating or adopting",async()=>{
 f.mode="read_only";f.params=new URLSearchParams({agent_key:key,agent_actor:"actor",agent_org:"org"});
 const imageAdvice={
  schemaVersion:"image-advice-v1",
  binding:{productKey:"product",catalogVersion:"1",publicationId:"publication",targetPlatform:"shein",inventoryDigest:"a".repeat(64),target:{recordId:id,revision:"1",storeId:id,site:"shein-us",categoryId:12,ruleDigest:"d".repeat(64),sourceId:id,originalPublicationId:"publication",originalVersion:"1",applyReceiptId:""}},
  evidence:{toolId:"listing.materials.inspect",toolVersion:"v3.0.0",callId:"call",digest:"b".repeat(64)},
  facts:[],checks:[{id:"check",code:"approved_assets_unavailable",status:"unknown",assetIds:[],ruleRef:"",reason:"批准素材尚未准备好，不能据此判断必需图片缺失。"}],
  suggestions:[{kind:"open_existing_images",checkIds:["check"],assetIds:[],reason:"请在原图片入口核对"}],unresolved:["批准素材尚未准备好"]
 };
 f.read.mockResolvedValue({...run(),canRequestAdoption:false,canResume:false,imageAdvice});
 render(<MaterialsPage recordId={id}/>);
 await screen.findByRole("heading",{name:"图片事实与建议"});
 expect(screen.getByText("批准素材尚未准备好，不能据此判断必需图片缺失。")).toBeInTheDocument();
 expect(f.prepare).not.toHaveBeenCalled();expect(f.start).not.toHaveBeenCalled();expect(f.adopt).not.toHaveBeenCalled();
 expect(screen.queryByRole("button",{name:"确认保存本地草稿"})).toBeNull();
});

function queryRun(requestKey=key){
 const {before,after,proposal,candidateHash,...common}=run(requestKey);void before;void after;void proposal;void candidateHash;
 return {...common,purpose:"category-rules",candidateCategoryId:"123",canRequestAdoption:false,canResume:false,queryRules:{binding:{CandidateRules:{BaseRevision:"1",CandidateCategoryID:"123"}},categoryPath:[{id:"123",name:"候选类目",productTypeId:"456"}],fill:{fill_in_standard_list:[{field_key:"name",module:"product",required:null,show:true}],picture_config_list:[]},attributes:{attribute_infos:[]},coverage:[{section:"category",status:"available",reason:"实际类目"},{section:"conditional",status:"not_evaluated",reason:"具体属性关联条件未评价"}],unknowns:["未检查真实图片"],evidence:{readAt:"2026-11-03T00:00:00Z",officialVersionStatus:"unknown"}},ruleExplanation:{schemaVersion:"listing-rule-explanation-v1",summary:"先核对该类目的基础填写模板。",references:[],unknowns:[]}};
}

it("confirms a separate candidate selector without changing the saved draft or offering adoption",async()=>{
 const before=JSON.stringify(base);
 f.prepare.mockImplementation(async(_scope,_record,input)=>({recordId:id,requestKey:input.requestKey,configurationSnapshotRef:ref,modeSnapshotRef:modeRef,modeRevision:"1",templateSelection:null,purpose:"category-rules",candidateCategoryId:input.candidateCategoryId}));
 f.start.mockImplementation(async(_scope,_record,requestKey)=>queryRun(requestKey));
 render(<MaterialsPage recordId={id}/>);
 fireEvent.change(await screen.findByLabelText("本次用途"),{target:{value:"category-rules"}});
 await screen.findByRole("option",{name:"候选类目"});
 fireEvent.change(screen.getByLabelText("候选类目"),{target:{value:"123"}});
 fireEvent.click(screen.getByRole("button",{name:"查询候选类目规则"}));
 await screen.findByRole("heading",{name:"确认查询候选类目规则"});
 fireEvent.click(screen.getByRole("button",{name:"确认查询并解释"}));
 await screen.findByText("先核对该类目的基础填写模板。");
 expect(f.prepare.mock.calls[0][2].candidateCategoryId).toBe("123");
 expect(JSON.stringify(base)).toBe(before);expect(f.adopt).not.toHaveBeenCalled();
 expect(screen.queryByRole("button",{name:"确认保存本地草稿"})).toBeNull();
 expect(screen.queryByRole("button",{name:"核对原保存结果"})).toBeNull();
});

it("restores full-access rule history without refreshing rules or enabling saves",async()=>{
 f.mode="full_access";f.params=new URLSearchParams({agent_key:key,agent_actor:"actor",agent_org:"org"});f.read.mockResolvedValue(queryRun());
 render(<MaterialsPage recordId={id}/>);
 await screen.findByText("先核对该类目的基础填写模板。");
 expect(screen.getByText("具体属性关联条件未评价")).toBeInTheDocument();
 expect(f.rules).not.toHaveBeenCalled();expect(f.prepare).not.toHaveBeenCalled();expect(f.start).not.toHaveBeenCalled();expect(f.adopt).not.toHaveBeenCalled();
 expect(screen.queryByRole("button",{name:"核对原保存结果"})).toBeNull();
});

it("shows the known query repair limit without claiming an explanation was generated",async()=>{
 f.params=new URLSearchParams({agent_key:key,agent_actor:"actor",agent_org:"org"});
 f.read.mockResolvedValue({...queryRun(),stopReason:"repair_limit",ruleExplanation:undefined});
 render(<MaterialsPage recordId={id}/>);
 await screen.findByRole("heading",{name:"规则解释未完成，请核对停止原因"});
 expect(screen.queryByRole("heading",{name:"规则解释已生成，待核对"})).toBeNull();
 expect(screen.queryByRole("heading",{name:"智能体解释"})).toBeNull();
 expect(screen.getByText("停止原因：repair_limit")).toBeInTheDocument();
 expect(f.rules).not.toHaveBeenCalled();expect(f.start).not.toHaveBeenCalled();expect(f.adopt).not.toHaveBeenCalled();
});
it("keeps a historical interrupted run readable without offering Resume",async()=>{
 f.params=new URLSearchParams({agent_key:key,agent_actor:"actor",agent_org:"org"});
 f.read.mockResolvedValue({...run(),phase:"interrupted",canResume:false,canRequestAdoption:false});
 render(<MaterialsPage recordId={id}/>);
 await screen.findByText("等待继续");
 expect(screen.queryByRole("textbox")).toBeNull();expect(f.resume).not.toHaveBeenCalled();expect(f.start).not.toHaveBeenCalled();
 expect(screen.getByText("本次记录未提供图片检查。")).toBeInTheDocument();
});
