import {StrictMode} from "react";
import {act,cleanup,fireEvent,render,screen,waitFor,within} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {ProductImageSetPanel} from "./product-image-set-panel";
import {imageTemplateSchema} from "@/lib/contracts/image-set-configuration";
import {imageSetRunSchema} from "@/lib/contracts/product-image-set";
import {genericImagePresets} from "@/lib/image-template-presets";
import {contentProposalFixture} from "@/test/product-content-review-fixture";
const calls=vi.hoisted(()=>({context:{} as Record<string,unknown>,mode:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>calls.context}));
vi.mock("@/lib/api/agent-configuration",async original=>{const actual=await original<typeof import("@/lib/api/agent-configuration")>();return {...actual,configurationRequest:(...args:Parameters<typeof actual.configurationRequest>)=>args[1].startsWith("execution-modes/")?calls.mode(...args):actual.configurationRequest(...args)}});
const operation="11111111-1111-4111-8111-111111111111",runId="22222222-2222-4222-8222-222222222222",templateId="33333333-3333-4333-8333-333333333333";
const sha="a".repeat(64),digest="b".repeat(64),now="2026-10-09T00:00:00Z";
const template={templateId,agentId:"product.image.agent",lifecycle:"ACTIVE",revision:"1",version:"1",schemaVersion:"image-config-v1",name:"Two groups",targetPlatform:"product",createdAt:now,image:{schema:"image-config-v1",mode:"standard",shareOriginals:true,background:"white",language:"en",carousel:[{id:"main",purpose:"product_identity"}],detail:[{id:"detail",purpose:"detail_closeup"}]}};
const entry={agent:{agentId:"product.image.agent",activation:"ENABLED",revision:"1",activationEpoch:"1",defaultTemplate:{templateId,revision:"1"},updatedAt:now},name:"Images",description:"Full images",definitionVersion:"v1.0.0",parameterSchema:"image-config-v1",canConfigure:true,canUse:true,canReadRuns:true,capabilities:[{id:"image.generate",support:"REQUIRED",readiness:"AVAILABLE",reason:"",observedAt:now}]};
const source={ContextKind:"acquisition",ProductID:"p",OperationID:operation,OriginalPublicationID:"publication",OriginalVersion:"1",EffectiveVersion:"1"};

function genericEntry(index:number,{hasDefault=false,failDefault=false,pending=false}={}){
 const preset=genericImagePresets[index]!,selection={skillId:preset.skillId,version:2,digest:"f".repeat(64)};
 const date="2026-10-18T00:00:00Z";
 const method={skill:{id:selection.skillId,owner:{kind:"PLATFORM",id:"official"},creator:"maintainer",headVersion:3,revision:3,createdAt:date,updatedAt:date},version:2,package:{name:"product-image-method",displayName:"已选图片方法",description:"保持商品真实事实",target:"product-image",digest:selection.digest,files:null},state:"ENABLED",visibility:"SHARED",allowCopy:true,allowExport:true,createdAt:date};
 const exact=imageTemplateSchema.parse({...template,templateId:"44444444-4444-4444-8444-444444444444",name:"企业自定义名称",revision:"8",version:"7",image:structuredClone(preset.image)});
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((input,init)=>{
  const url=new URL(String(input),"https://app.test");
  if(url.pathname.endsWith("/skills/organization/image-candidates"))return Promise.resolve(Response.json({items:[method]}));
  if(url.pathname.endsWith("/agents/product.image.agent"))return Promise.resolve(Response.json({...entry,agent:{...entry.agent,defaultTemplate:hasDefault?entry.agent.defaultTemplate:null}}));
  if(url.pathname.endsWith("/templates"))return Promise.resolve(Response.json(url.searchParams.has("cursor")?{items:[exact],nextCursor:""}:{items:[{...template,name:preset.name}],nextCursor:"exact-next"}));
  if(failDefault&&url.pathname.includes("/revisions/"))return Promise.resolve(Response.json({code:"DEPENDENCY_UNAVAILABLE"},{status:503}));
  return real(input,init);
 });
 const storage=`product-image-set:actor:org:acquisition:${operation}:intent`;
 const raw=JSON.stringify({action:"confirm",runId,body:{actionId:operation,planRevision:1,planDigest:sha,quoteDigest:digest}});
 if(pending)localStorage.setItem(storage,raw);
 return {selection,exact,storage,raw};
}
function projection(status="awaiting_plan_approval",confirmationActionId=""){
 return {executionModeAvailable:true,generationAllowed:true,executionMode:"default",modeRevision:"0",modeSnapshotRef:{Kind:"agent-execution-mode-v1",ID:runId,Digest:sha},runId,status,template:{templateId,revision:"1"},confirmationActionId,generationAdmitted:!!confirmationActionId,planRevision:1,planDigest:sha,quoteDigest:digest,images:2,points:20,settledPoints:status==="awaiting_plan_approval"?0:20,resultDigest:status==="awaiting_plan_approval"?"":sha,approvalAvailable:status==="awaiting_final_approval",candidateSelectionAvailable:["awaiting_final_approval","completed","failed","blocked","cancelled"].includes(status),regenerationAvailable:status==="awaiting_final_approval",plan:{Source:source,Target:{Platform:"product",StoreID:"",Site:"",CategoryID:0}},slots:["main","detail"].map((slotId,i)=>({slotId,status:status==="awaiting_plan_approval"?"pending":"accepted",attempt:status==="awaiting_plan_approval"?0:1,errorCode:"",recipe:{Purpose:i?"detail_closeup":"product_identity",Background:"white",Language:"en",Placement:{Group:i?"detail":"carousel",Order:1},References:[{AssetID:"original"}],Quote:{Points:10}},candidates:status==="awaiting_plan_approval"?[]:[{assetId:`generated-${i}`,url:`https://images.test/generated-${i}.png`,width:1024,height:1024}],closure:status==="awaiting_plan_approval"?null:{Kind:"generated",Points:10}})),originals:[{ID:"original",DisplayURL:"https://images.test/original.png",Width:1024,Height:1024}],recoverableEffects:null,pendingCommand:null,block:null};
}
let state:ReturnType<typeof projection>,approved:boolean,fetch:ReturnType<typeof vi.fn<(input:unknown,init?:RequestInit)=>Promise<Response>>>;
beforeEach(()=>{
 Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true}});
 Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false}});
 calls.mode.mockReset();calls.mode.mockResolvedValue({subject:{kind:"registered_agent",id:"product.image.agent"},definition:{id:"product.image.agent",version:"v1.0.0"},mode:"default",revision:"0",policyVersion:"agent-execution-policy-v1",canConfigure:false,updatedAt:"2026-10-12T00:00:00Z",updatedBy:""});
 localStorage.clear();approved=false;state=projection();
 calls.context={user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["listingkit.image_agent.read","listingkit.image_agent.write"],registerOrganizationSwitchGuard:vi.fn(()=>vi.fn())};
 fetch=vi.fn(async(input:unknown,init?:RequestInit)=>{
  const url=String(input),body=init?.body?JSON.parse(String(init.body)):undefined;
  if(url.endsWith("/skills/organization/image-candidates"))return Response.json({items:[]});
  if(url.includes("/agents/"))return Response.json(url.includes("/revisions/")?template:url.includes("/templates")?{items:[template],nextCursor:""}:entry);
  const parsed=new URL(url,"http://localhost");
  if(parsed.pathname.endsWith("/images/sources"))return Response.json({contextKind:"acquisition",contextId:operation,source:{...source,EffectiveVersion:parsed.searchParams.get("effectiveCatalogVersion")??source.EffectiveVersion,...parsed.searchParams.get("applyReceiptId")?{ApplyReceiptID:parsed.searchParams.get("applyReceiptId")}:{}},manualReplacementAvailable:false,originals:[{id:"original",displayUrl:"https://images.test/original.png",width:1024,height:1024}],evidence:{}});
  if(url.endsWith("/images/runs"))return Response.json({items:[],nextCursor:""});
  if(url.endsWith("/prepare"))return Response.json(state,{status:201});
  if(url.includes("/by-key/"))return Response.json(state);
  if(url.endsWith("/confirm")){state=projection("awaiting_final_approval",body.actionId);return Response.json({runId,status:"accepted"},{status:202})}
  if(url.endsWith("/preview"))return Response.json({digest,head:{action_id:"",payload_hash:""},assets:[{id:"generated-0",role:"main",url:"https://images.test/generated-0.png"},{id:"generated-1",role:"detail",url:"https://images.test/generated-1.png"}]});
  if(url.endsWith("/approve")){approved=true;state={...state,status:"completed",approvalAvailable:false};return Response.json({runId,status:"accepted"},{status:202})}
  if(url.endsWith("/inventory"))return Response.json({target:{approval_action_id:approved?templateId:"",head:approved?{action_id:templateId,payload_hash:sha}:{action_id:"",payload_hash:""},assets:approved?[{id:"generated-0",role:"main",url:"https://images.test/generated-0.png",presentation:{group:"carousel",order:1}},{id:"generated-1",role:"detail",url:"https://images.test/generated-1.png",presentation:{group:"detail",order:1}}]:[]},generic:null});
  if(url.endsWith(`/runs/${runId}`))return Response.json(state);
  throw new Error(`unexpected ${url}`);
 });vi.stubGlobal("fetch",fetch);
});
afterEach(()=>{cleanup();vi.restoreAllMocks();vi.unstubAllGlobals();localStorage.clear()});
async function prepare(){
 fireEvent.click(await screen.findByRole("checkbox",{name:"共用原始素材 素材 1"}));
 const button=screen.getByRole("button",{name:"准备整套图片计划（2 项）"});await waitFor(()=>expect(button).toBeEnabled());fireEvent.click(button);
}

it("sends the exact VIP selection in a new plan and retains it with the original unknown request",async()=>{
 const original=fetch.getMockImplementation()!,route={BindingID:"image-vip",RevisionID:"route-vip"};
 fetch.mockImplementation((url,init)=>String(url).endsWith("/model-options")?Promise.resolve(Response.json({contextKind:"acquisition",contextId:operation,options:[{model:"qwen-image-3.0-pro",quality:"auto",resolution:"1024x1024",points:1},{route,model:"gpt-image-2-vip",quality:"medium",resolution:"1024x1024",points:2}]})):String(url).endsWith("/prepare")?Promise.resolve(Response.json({code:"OUTCOME_UNKNOWN"},{status:503})):original(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 await screen.findByRole("option",{name:"GPT Image VIP · 2 点/张"});
 fireEvent.change(screen.getByRole("combobox",{name:"本次计划使用的图片模型"}),{target:{value:JSON.stringify(route)}});
 await prepare();await screen.findByRole("button",{name:"核实原请求"});
 const sent=fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"))!;
 expect(JSON.parse(String(sent[1]!.body)).route).toEqual(route);
 expect(JSON.parse(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)!).body.route).toEqual(route);
 expect(screen.getByRole("combobox",{name:"本次计划使用的图片模型"})).toBeDisabled();
 expect(fetch.mock.calls.filter(([url])=>String(url).endsWith("/prepare"))).toHaveLength(1);
});

it("disables preparing a new plan until a restored unavailable VIP is explicitly reselected",async()=>{
 const route={BindingID:"image-vip",RevisionID:"route-old"};Object.assign(state.plan,{Route:route});localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 const original=fetch.getMockImplementation()!;fetch.mockImplementation((url,init)=>String(url).endsWith("/model-options")?Promise.resolve(Response.json({contextKind:"acquisition",contextId:operation,options:[{model:"qwen-image-3.0-pro",quality:"auto",resolution:"1024x1024",points:1}]})):original(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("checkbox",{name:"共用原始素材 素材 1"}));
 await screen.findByText(/原选版本不可用或尚未核实/);
 expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeDisabled();
 fireEvent.change(screen.getByRole("combobox",{name:"本次计划使用的图片模型"}),{target:{value:"default"}});
 await waitFor(()=>expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeEnabled());
});

it("keeps model switching and new preparation locked while the original run has unresolved effects",async()=>{
 state=projection("blocked");Object.assign(state,{recoverableEffects:[{SlotID:"main",Attempt:1,Code:"provider_outcome_unknown"}]});localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("checkbox",{name:"共用原始素材 素材 1"}));
 expect(screen.getByRole("combobox",{name:"本次计划使用的图片模型"})).toBeDisabled();
 expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeDisabled();
 expect(screen.getByRole("button",{name:"核实原调用"})).toBeEnabled();
});

it.each(["prepare","regenerate"])("restores the exact pending VIP for %s and archives a definite route refusal before unlocking",async action=>{
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`,route={BindingID:"image-vip",RevisionID:"route-vip"};
 const raw=JSON.stringify({action,...action==="regenerate"?{runId}:{},requestKey:templateId,body:{route,text:{mode:"none"},expectedModeRevision:"0",target:{Platform:"product"}}},null,2);localStorage.setItem(key,raw);
 const nextRoute={...route,RevisionID:"route-new"};let reads=0,finishRead!:(value:Response)=>void;
 const refreshed=new Promise<Response>(resolve=>{finishRead=resolve});
 const options=(selected:typeof route)=>({contextKind:"acquisition",contextId:operation,options:[{model:"qwen-image-3.0-pro",quality:"auto",resolution:"1024x1024",points:1},{route:selected,model:"gpt-image-2-vip",quality:"medium",resolution:"1024x1024",points:2}]});
 const original=fetch.getMockImplementation()!;fetch.mockImplementation((url,init)=>String(url).endsWith("/model-options")?(++reads===1?Promise.resolve(Response.json(options(route))):refreshed):String(url).endsWith(`/${action}`)?Promise.resolve(Response.json({code:"IMAGE_ROUTE_CHANGED"},{status:409})):original(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 const picker=await screen.findByRole("combobox",{name:"本次计划使用的图片模型"});expect(picker).toHaveValue(JSON.stringify(route));expect(picker).toBeDisabled();
 await screen.findByText(/gpt-image-2-vip · medium · 原始输出/);
 fireEvent.click(screen.getByRole("button",{name:"继续原操作"}));
 await screen.findByText(/所选图片模型版本已不可用/);
 await waitFor(()=>expect(localStorage.getItem(key)).toBeNull());
 expect(Object.keys(localStorage).some(k=>localStorage.getItem(k)===raw)).toBe(true);
 expect(screen.getByRole("combobox",{name:"本次计划使用的图片模型"})).toBeEnabled();
 expect(screen.queryByRole("button",{name:"继续原操作"})).toBeNull();
 expect(fetch.mock.calls.filter(([url])=>String(url).endsWith(`/${action}`))).toHaveLength(1);
 await waitFor(()=>expect(reads).toBe(2));
 fireEvent.click(screen.getByRole("checkbox",{name:"共用原始素材 素材 1"}));
 const prepareButton=screen.getByRole("button",{name:"准备整套图片计划（2 项）"});
 expect(prepareButton).toBeDisabled();
 fireEvent.click(prepareButton);expect(fetch.mock.calls.filter(([url])=>String(url).endsWith(`/${action}`))).toHaveLength(1);
 await act(async()=>finishRead(Response.json(options(nextRoute))));
 await screen.findByRole("option",{name:/原选版本 route-vip/});expect(prepareButton).toBeDisabled();
 fireEvent.change(screen.getByRole("combobox",{name:"本次计划使用的图片模型"}),{target:{value:JSON.stringify(nextRoute)}});
 await waitFor(()=>expect(prepareButton).toBeEnabled());
 expect(Object.keys(localStorage).some(k=>localStorage.getItem(k)===raw)).toBe(true);
});

it.each(["fresh","restored","sources-unavailable"])("retains a %s refused confirmation and keeps that immutable plan disabled after refresh",async scenario=>{
 const route={BindingID:"image-vip",RevisionID:"route-old"},nextRoute={...route,RevisionID:"route-new"};Object.assign(state.plan,{Route:route});
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`;localStorage.setItem(key.replace(/:intent$/,":run"),runId);
 if(scenario==="restored")localStorage.setItem(key,JSON.stringify({action:"confirm",runId,body:{actionId:operation,planRevision:1,planDigest:sha,quoteDigest:digest}},null,2));
 let reads=0,raw="",sourcesUnavailable=false;const original=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>sourcesUnavailable&&String(url).endsWith("/sources")?Promise.resolve(Response.json({code:"IMAGE_UNAVAILABLE"},{status:503})):String(url).endsWith("/model-options")?Promise.resolve(Response.json({contextKind:"acquisition",contextId:operation,options:[{model:"qwen-image-3.0-pro",quality:"auto",resolution:"1024x1024",points:1},{route:++reads===1?route:nextRoute,model:"gpt-image-2-vip",quality:"medium",resolution:"1024x1024",points:2}]})):String(url).endsWith("/confirm")?(raw=localStorage.getItem(key)!,Promise.resolve(Response.json({code:"IMAGE_ROUTE_CHANGED"},{status:409}))):original(url,init));
 const view=render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 await screen.findByText(/gpt-image-2-vip · medium · 原始输出/);
 fireEvent.click(screen.getByRole("button",{name:scenario==="restored"?"继续原确认":"确认点数并生成"}));
 await screen.findByText(/所选图片模型版本已不可用/);
 await waitFor(()=>expect(localStorage.getItem(key)).toBeNull());
 expect(Object.keys(localStorage).some(k=>localStorage.getItem(k)===raw)).toBe(true);
 await waitFor(()=>expect(reads).toBe(2));
 const picker=screen.getByRole("combobox",{name:"本次计划使用的图片模型"});expect(picker).toHaveValue(JSON.stringify(route));
 expect(screen.getByRole("button",{name:"确认点数并生成"})).toBeDisabled();
 fireEvent.change(picker,{target:{value:JSON.stringify(nextRoute)}});
 fireEvent.click(screen.getByRole("checkbox",{name:"共用原始素材 素材 1"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeEnabled());
 expect(screen.getByRole("button",{name:"确认点数并生成"})).toBeDisabled();
 view.unmount();sourcesUnavailable=scenario==="sources-unavailable";render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 await screen.findByRole("button",{name:"确认点数并生成"});
 if(!sourcesUnavailable)await screen.findByRole("option",{name:/原选版本 route-old/});
 expect(screen.getByRole("button",{name:"确认点数并生成"})).toBeDisabled();
 expect(fetch.mock.calls.filter(([url])=>String(url).endsWith("/confirm"))).toHaveLength(1);
 expect(Object.keys(localStorage).some(k=>localStorage.getItem(k)===raw)).toBe(true);
});

it("archives a definite image-review rejection and unlocks correction without generating",async()=>{
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/prepare")?Promise.resolve(Response.json({code:"IMAGE_TEXT_REVIEW_CHANGED"},{status:409})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);await prepare();
 await screen.findByText(/审核译文已变化/);
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`;
 const sent=fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"))!;
 const requestKey=new Headers(sent[1]!.headers).get("Idempotency-Key");
 expect(localStorage.getItem(key)).toBeNull();
 expect(Object.keys(localStorage).some(k=>k.includes(":verified-intent:")&&JSON.parse(localStorage.getItem(k)!).requestKey===requestKey)).toBe(true);
 expect(screen.queryByRole("button",{name:"核实原请求"})).toBeNull();
 expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeDisabled();
 fireEvent.change(screen.getByRole("combobox",{name:"图片文字方式"}),{target:{value:"none"}});
 expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeEnabled();
 fireEvent.click(screen.getByRole("checkbox",{name:"商品识别主图"}));expect(screen.getByRole("button",{name:"准备整套图片计划（1 项）"})).toBeDisabled();
 fireEvent.click(screen.getByRole("checkbox",{name:"商品识别主图"}));expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeDisabled();
 fireEvent.change(screen.getByRole("combobox",{name:"图片文字方式"}),{target:{value:"none"}});expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeEnabled();
 expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(1);
});

it.each(["prepare","regenerate"])("preserves restored %s bytes when the original replay gets a definite review rejection",async action=>{
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`;
 const raw=JSON.stringify({action,...action==="regenerate"?{runId}:{},requestKey:templateId,body:{expectedModeRevision:"0",target:{Platform:"product"},skill:{skillId:operation,version:2,digest:sha},text:{mode:"none"}}},null,2);
 localStorage.setItem(key,raw);const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith(`/${action}`)?Promise.resolve(Response.json({code:"IMAGE_TEXT_REVIEW_CHANGED"},{status:409})):real(url,init));
 const view=render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"继续原操作"}));
 await screen.findByText(/审核译文已变化/);
 expect(localStorage.getItem(key)).toBeNull();
 expect(Object.keys(localStorage).some(k=>k.includes(":verified-intent:")&&localStorage.getItem(k)===raw)).toBe(true);
 const sent=fetch.mock.calls.find(([url])=>String(url).endsWith(`/${action}`))!;
 expect(new Headers(sent[1]!.headers).get("Idempotency-Key")).toBe(templateId);expect(JSON.parse(String(sent[1]!.body))).toEqual(JSON.parse(raw).body);
 view.unmount();render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 const history=await screen.findByText("已核实的原请求");
 expect(history.closest("section")!.querySelector("pre")!.textContent).toBe(raw);
 if(action==="prepare"){
  fireEvent.click(screen.getByRole("checkbox",{name:"共用原始素材 素材 1"}));
  expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeDisabled();
  fireEvent.change(screen.getByRole("combobox",{name:"图片文字方式"}),{target:{value:"none"}});
  expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeEnabled();
 }
 expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(1);
});

it("requires a freshly confirmed Review after a definite initial preparation rejection",async()=>{
 const fixture=contentProposalFixture(),reviewId=fixture.proposal_id,selection={targetLanguage:"en",title:true,sellingPoints:true},values=[{field:"title",value:"Women's organizer"},{field:"selling_points",value:'["Size: 13\\\" × 15\\\""]'}];
 const accepted={...fixture,input:{product_key:"p",base_version:"1"},state:"accepted",content:{...fixture.content,selection,before:values,after:values},execution:{...fixture.execution,binding:{...fixture.execution.binding,ProductKey:"p"},ContentSelection:selection}};
 let reviewRevision="2",preparations=0;const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith(`/text-proposals/${reviewId}`)?Promise.resolve(Response.json({...accepted,revision:reviewRevision})):String(url).endsWith("/prepare")?Promise.resolve(++preparations===1?Response.json({code:"IMAGE_TEXT_REVIEW_CHANGED"},{status:409}):Response.json(state,{status:201})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 const initialCopy=within(await screen.findByRole("region",{name:"图片文字语言"}));
 fireEvent.change(initialCopy.getByRole("combobox",{name:"图片文字方式"}),{target:{value:"reviewed"}});
 fireEvent.change(initialCopy.getByLabelText("文案审核编号"),{target:{value:reviewId}});fireEvent.click(initialCopy.getByRole("button",{name:"读取审核译文"}));await initialCopy.findByText("本次图片文案 · 最多两条");
 fireEvent.click(initialCopy.getByRole("checkbox",{name:"确认目标语言图片文案"}));await prepare();
 await screen.findByText(/审核译文已变化/);
 const button=screen.getByRole("button",{name:"准备整套图片计划（2 项）"}),archived=Object.keys(localStorage).find(key=>key.includes(":verified-intent:"))!,raw=localStorage.getItem(archived)!;
 expect(JSON.parse(raw).body.text.review).toEqual({proposalId:reviewId,revision:"2"});expect(button).toBeDisabled();expect(preparations).toBe(1);
 reviewRevision="3";
 const copy=within(screen.getByRole("region",{name:"图片文字语言"}));
 const read=copy.getByRole("button",{name:"读取审核译文"});await waitFor(()=>expect(read).toBeEnabled());fireEvent.click(read);await copy.findByText("本次图片文案 · 最多两条");expect(button).toBeDisabled();
 fireEvent.click(copy.getByRole("checkbox",{name:"确认目标语言图片文案"}));await waitFor(()=>expect(button).toBeEnabled());fireEvent.click(button);
 await waitFor(()=>expect(preparations).toBe(2));
 const request=fetch.mock.calls.filter(([url])=>String(url).endsWith("/prepare"))[1]!;
 expect(JSON.parse(String(request[1]!.body)).text).toEqual({mode:"reviewed",review:{proposalId:reviewId,revision:"3"},taskIds:["main","detail"]});
 expect(localStorage.getItem(archived)).toBe(raw);expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(2);
});

it("keeps a rejected preparation locked if its original request cannot be archived",async()=>{
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`,real=fetch.getMockImplementation()!;
 const write=Storage.prototype.setItem;
 vi.spyOn(Storage.prototype,"setItem").mockImplementation(function(this:Storage,k,v){if(k.includes(":verified-intent:"))throw new Error("storage full");return write.call(this,k,v)});
 fetch.mockImplementation((url,init)=>String(url).endsWith("/prepare")?Promise.resolve(Response.json({code:"IMAGE_TEXT_REVIEW_CHANGED"},{status:409})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);await prepare();
 await screen.findByText(/原请求未能完整保留/);
 expect(localStorage.getItem(key)).toBeTruthy();
 expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeDisabled();
 expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(1);
});

it.each(["fresh","restored","no-text"])("uses explicitly corrected copy for %s rejected regeneration while preserving its parent and original request",async scenario=>{
 const parentTemplateId="55555555-5555-4555-8555-555555555555",reviewId="12345678-1234-4234-8234-123456789abc";
 const parentTemplate={...template,templateId:parentTemplateId,version:"2",revision:"2",name:"Frozen French",image:{...template.image,language:"fr"}};
 const skill={skillId:operation,version:2,digest:sha},oldText={mode:"reviewed",review:{proposalId:reviewId,revision:"2"},taskIds:["detail"]};
 state=projection("awaiting_final_approval",operation);
 state.template={templateId:parentTemplateId,revision:"2"};
 Object.assign(state.plan,{Method:{Skill:{kind:"skill-version-v1",id:skill.skillId,ownerKind:"PLATFORM",ownerId:"official",version:skill.version,digest:skill.digest,policyVersion:"skill-method-v1"},PolicyVersion:"image-method-visual-brief-v4"},Text:{PolicyVersion:"image-reviewed-text-v1",Mode:"reviewed",Language:"fr",TaskIDs:["detail"],Copy:{Review:oldText.review,ProductID:"p",PublicationID:"publication",Version:"1",Language:"fr",OriginRunID:runId,ModeSnapshot:{Kind:"agent-execution-mode-v1",ID:runId,Digest:sha},Lines:["Ancien texte"]}}});
 const frozenParent=JSON.stringify(state.plan),storage=`product-image-set:actor:org:acquisition:${operation}`,real=fetch.getMockImplementation()!;
 const rejected={action:"regenerate",runId,requestKey:templateId,body:{expectedModeRevision:"0",template:{templateId:parentTemplateId,revision:"2"},target:{Platform:"product"},effectiveCatalogVersion:"1",sharedOriginalIds:["original"],selectedTaskIds:["detail"],skill,text:oldText}};
 const restoredRaw=JSON.stringify(rejected,null,2);
 localStorage.setItem(storage+":run",runId);if(scenario==="restored")localStorage.setItem(storage+":intent",restoredRaw);
 const fixture=contentProposalFixture(),selection={targetLanguage:"fr",title:true,sellingPoints:true},values=[{field:"title",value:"Boîte de rangement"},{field:"selling_points",value:'["Pour les objets du quotidien"]'}];
 const reviewed={...fixture,proposal_id:reviewId,input:{product_key:"p",base_version:"1"},state:"accepted",revision:"3",content:{...fixture.content,selection,before:values,after:values},execution:{...fixture.execution,binding:{...fixture.execution.binding,ProductKey:"p"},ContentSelection:selection}};
 let regenerations=0;
 fetch.mockImplementation((url,init)=>String(url).includes(`/templates/${parentTemplateId}/revisions/2`)?Promise.resolve(Response.json(parentTemplate)):String(url).endsWith(`/text-proposals/${reviewId}`)?Promise.resolve(Response.json(reviewed)):String(url).endsWith("/regenerate")?Promise.resolve(++regenerations===1?Response.json({code:"IMAGE_TEXT_REVIEW_CHANGED"},{status:409}):Response.json(projection(),{status:201})):real(url,init));
 let view=render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 if(scenario==="restored")fireEvent.click(await screen.findByRole("button",{name:"继续原操作"}));
 else{fireEvent.click((await screen.findAllByRole("checkbox",{name:"重新生成此项，另行确认点数"}))[1]);const button=screen.getByRole("button",{name:"准备所选 1 项的新计划"});await waitFor(()=>expect(button).toBeEnabled());fireEvent.click(button)}
 await screen.findByText(/审核译文已变化/);
 const archived=Object.keys(localStorage).find(key=>key.includes(":verified-intent:"))!,raw=localStorage.getItem(archived)!;
 expect(JSON.parse(raw).body).toMatchObject(rejected.body);
 expect(localStorage.getItem(storage+":intent")).toBeNull();if(scenario==="restored")expect(raw).toBe(restoredRaw);
 // Correction must also survive a reload, without replaying or rewriting the rejected request.
 view.unmount();view=render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click((await screen.findAllByRole("checkbox",{name:"重新生成此项，另行确认点数"}))[1]);
 const correction=within(await screen.findByRole("region",{name:"重新生成文案纠正"}));
 expect(correction.getByText("目标语言：法语（fr）")).toBeVisible();
 const button=screen.getByRole("button",{name:"准备所选 1 项的新计划"});expect(button).toBeDisabled();expect(regenerations).toBe(1);
 if(scenario==="no-text")fireEvent.change(correction.getByRole("combobox",{name:"图片文字方式"}),{target:{value:"none"}});
 else{
  fireEvent.change(correction.getByLabelText("文案审核编号"),{target:{value:reviewId}});fireEvent.click(correction.getByRole("button",{name:"读取审核译文"}));
  await correction.findByText("本次图片文案 · 最多两条");expect(button).toBeDisabled();
  fireEvent.click(correction.getByRole("checkbox",{name:"确认目标语言图片文案"}));
 }
 await waitFor(()=>expect(button).toBeEnabled());
 if(scenario==="no-text"){
  // A different slot subset requires a fresh explicit text choice.
  fireEvent.click(screen.getAllByRole("checkbox",{name:"重新生成此项，另行确认点数"})[0]);expect(screen.getByRole("button",{name:"准备所选 2 项的新计划"})).toBeDisabled();
  fireEvent.click(screen.getAllByRole("checkbox",{name:"重新生成此项，另行确认点数"})[0]);expect(button).toBeDisabled();
  fireEvent.change(within(screen.getByRole("region",{name:"重新生成文案纠正"})).getByRole("combobox",{name:"图片文字方式"}),{target:{value:"none"}});
 }
 fireEvent.click(button);
 await waitFor(()=>expect(regenerations).toBe(2));
 const request=fetch.mock.calls.filter(([url])=>String(url).endsWith("/regenerate"))[1]!;
 expect(JSON.parse(String(request[1]!.body))).toMatchObject({...rejected.body,text:scenario==="no-text"?{mode:"none"}:{...oldText,review:{proposalId:reviewId,revision:"3"}}});
 expect(new Headers(request[1]!.headers).get("Idempotency-Key")).not.toBe(JSON.parse(raw).requestKey);
 expect(JSON.stringify(state.plan)).toBe(frozenParent);expect(localStorage.getItem(archived)).toBe(raw);
 expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(2);
});

it.each(["OUTCOME_UNKNOWN","IMAGE_NOT_FOUND"])("does not offer correction for an unresolved regeneration returning %s",async code=>{
 state=projection("awaiting_final_approval",operation);
 const storage=`product-image-set:actor:org:acquisition:${operation}`,raw=JSON.stringify({action:"regenerate",runId,requestKey:templateId,body:{expectedModeRevision:"0",target:{Platform:"product"},text:{mode:"reviewed",review:{proposalId:operation,revision:"2"},taskIds:["detail"]}}});
 localStorage.setItem(storage+":run",runId);localStorage.setItem(storage+":intent",raw);const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/regenerate")?Promise.resolve(Response.json({code},{status:code==="OUTCOME_UNKNOWN"?503:404})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"继续原操作"}));await screen.findByRole("alert");
 expect(localStorage.getItem(storage+":intent")).toBe(raw);expect(Object.keys(localStorage).some(key=>key.includes(":verified-intent:"))).toBe(false);
 expect(screen.queryByRole("region",{name:"重新生成文案纠正"})).toBeNull();expect(screen.getByRole("button",{name:"准备所选 0 项的新计划"})).toBeDisabled();
 expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(1);
});

it("shows frozen translated copy and language on restored plans without translation or model writes",async()=>{
 Object.assign(state.plan,{Method:{Skill:{kind:"skill-version-v1",id:operation,ownerKind:"PLATFORM",ownerId:"official",version:2,digest:sha,policyVersion:"skill-method-v1"},PolicyVersion:"image-method-visual-brief-v4"},Text:{PolicyVersion:"image-reviewed-text-v1",Mode:"reviewed",Language:"en",TaskIDs:["detail"],Copy:{Review:{proposalId:operation,revision:"2"},ProductID:"p",PublicationID:"publication",Version:"1",Language:"en",OriginRunID:runId,ModeSnapshot:{Kind:"agent-execution-mode-v1",ID:runId,Digest:sha},Lines:["Fabric storage box","Holds everyday items"]}}});
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 const text=await screen.findByRole("region",{name:"已固定的图片文案"});
 expect(within(text).getByText("目标语言：英语（en）")).toBeVisible();
 expect(within(text).getByText("Fabric storage box")).toBeVisible();
 expect(within(text).queryByRole("textbox")).toBeNull();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it("shows frozen literal detail copy before confirmation without allowing edits or generation",async()=>{
 Object.assign(state.plan,{Method:{Skill:{kind:"skill-version-v1",id:operation,ownerKind:"PLATFORM",ownerId:"official",version:2,digest:sha,policyVersion:"skill-method-v1"},PolicyVersion:"image-method-visual-brief-v3"}});
 state.slots[1].recipe.Purpose="key_benefits";
 Object.assign(state.slots[1].recipe,{VisibleText:["IKEA DRÖNA 深灰色布艺收纳盒","产品尺寸：宽33厘米，深38厘米，高33厘米"]});
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 const copy=await screen.findByRole("region",{name:"本次详情图文案"});
 expect(within(copy).getByText("IKEA DRÖNA 深灰色布艺收纳盒")).toBeVisible();
 expect(within(copy).getByText("产品尺寸：宽33厘米，深38厘米，高33厘米")).toBeVisible();
 expect(within(copy).queryByRole("textbox")).toBeNull();
 expect(screen.getByRole("button",{name:"确认点数并生成"})).toBeEnabled();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it("explains a text-free detail plan when no whole short fact is available",async()=>{
 Object.assign(state.plan,{Method:{Skill:{kind:"skill-version-v1",id:operation,ownerKind:"PLATFORM",ownerId:"official",version:2,digest:sha,policyVersion:"skill-method-v1"},PolicyVersion:"image-method-visual-brief-v3"}});
 state.slots[1].recipe.Purpose="key_benefits";
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 const copy=await screen.findByRole("region",{name:"本次详情图文案"});
 expect(within(copy).getByText("没有合适的已确认短文案，本图不加字。")).toBeVisible();
});

it("retains literal copy in read-only historical details without an execution or adoption action",async()=>{
 state=projection("awaiting_final_approval",templateId);
 Object.assign(state,{generationAllowed:false,approvalAvailable:false,candidateSelectionAvailable:false,regenerationAvailable:false});
 Object.assign(state.plan,{Method:{Skill:{kind:"skill-version-v1",id:operation,ownerKind:"PLATFORM",ownerId:"official",version:2,digest:sha,policyVersion:"skill-method-v1"},PolicyVersion:"image-method-visual-brief-v3"}});
 state.slots[1].recipe.Purpose="key_benefits";
 Object.assign(state.slots[1].recipe,{VisibleText:["已确认原文"]});
 calls.context={...calls.context,permissions:["listingkit.image_agent.read"]};
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 expect(within(await screen.findByRole("region",{name:"本次详情图文案"})).getByText("已确认原文")).toBeVisible();
 expect(screen.queryByRole("button",{name:"确认点数并生成"})).toBeNull();
 expect(screen.queryByRole("button",{name:"人工批准并保存素材"})).toBeNull();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it("bounds literal copy by two labels and Unicode code points while parsing old omitted fields",()=>{
 const p=projection();
 expect(imageSetRunSchema.safeParse(p).success).toBe(true);
 Object.assign(p.slots[1].recipe,{VisibleText:["🟢".repeat(32)]});
 expect(imageSetRunSchema.safeParse(p).success).toBe(true);
 Object.assign(p.slots[1].recipe,{VisibleText:["🟢".repeat(33)]});
 expect(imageSetRunSchema.safeParse(p).success).toBe(false);
 Object.assign(p.slots[1].recipe,{VisibleText:["一","二","三"]});
 expect(imageSetRunSchema.safeParse(p).success).toBe(false);
});

it("opens the original and generated full images without losing an explicit selection or mutating a task",async()=>{
 state=projection("awaiting_final_approval",templateId);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 const choices=await screen.findAllByRole("button",{name:"选择采用"});
 fireEvent.click(choices[0]);
 await waitFor(()=>expect(screen.getByRole("button",{name:"预览完整选择"})).toBeEnabled());
 fireEvent.click(screen.getAllByRole("button",{name:/查看.*生成结果.*大图/})[0]);
 const dialog=await screen.findByRole("dialog",{name:"图片预览"});
 expect(within(dialog).getByRole("img")).toHaveAttribute("src","https://images.test/generated-0.png");
 fireEvent.click(within(dialog).getByRole("button",{name:"关闭"}));
 expect(screen.queryByRole("dialog")).toBeNull();
 expect(screen.getByRole("button",{name:"预览完整选择"})).toBeEnabled();
 fireEvent.click(screen.getAllByRole("button",{name:/查看原图.*大图/})[0]);
 const originalDialog=await screen.findByRole("dialog",{name:"图片预览"});
 expect(within(originalDialog).getByRole("img")).toHaveAttribute("src","https://images.test/original.png");
 fireEvent(originalDialog,new Event("cancel",{cancelable:true}));
 expect(screen.queryByRole("dialog")).toBeNull();
 expect(screen.getByRole("img",{name:/拟采用：/})).toHaveAttribute("src","https://images.test/generated-0.png");
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it("labels custom results by their image group and order while preserving exact approval identities",async()=>{
 state=projection("awaiting_final_approval",templateId);
 state.slots=state.slots.map((slot,i)=>({...slot,slotId:`custom-${i}`,recipe:{...slot.recipe,Purpose:"custom"}}));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 expect(await screen.findByRole("heading",{name:"主图 / 轮播 1"})).toBeVisible();
 expect(screen.getByRole("heading",{name:"详情图 1"})).toBeVisible();
 const choices=screen.getAllByRole("button",{name:"选择采用"});choices.forEach(button=>fireEvent.click(button));
 expect(screen.getByRole("combobox",{name:"主图 / 轮播 1 素材组"})).toHaveValue("carousel");
 expect(screen.getByRole("combobox",{name:"详情图 1 素材组"})).toHaveValue("detail");
 fireEvent.click(screen.getByRole("button",{name:"预览完整选择"}));
 fireEvent.click(await screen.findByRole("button",{name:"人工批准并保存素材"}));
 await screen.findByRole("heading",{name:"已保存正式素材"});
 const command=JSON.parse(String(fetch.mock.calls.find(([url])=>String(url).endsWith("/approve"))![1]!.body));
 expect(command.choices.map((v:{slot_id:string;asset_id:string})=>[v.slot_id,v.asset_id])).toEqual([["custom-0","generated-0"],["custom-1","generated-1"]]);
 expect(command.choices.map((v:{presentation:unknown})=>v.presentation)).toEqual([{group:"carousel",order:1},{group:"detail",order:1}]);
 fireEvent.click(screen.getByRole("button",{name:"查看正式素材 1大图"}));
 expect(within(await screen.findByRole("dialog",{name:"图片预览"})).getByRole("img")).toHaveAttribute("src","https://images.test/generated-0.png");
});

it("shows a fully released no-generation result while retaining the original failure without recovery actions",async()=>{
 state=projection("blocked",templateId);
 Object.assign(state,{settledPoints:0,resultDigest:"",approvalAvailable:false,candidateSelectionAvailable:false,regenerationAvailable:false,releaseSummary:{releasedPoints:20},block:{Code:"slot_provider_outcome_unknown"},recoverableEffects:[{SlotID:"main",Attempt:1,Code:"slot_provider_outcome_unknown"}]});
 state.slots=state.slots.map(slot=>({...slot,status:"blocked",candidates:[],closure:null,errorCode:"slot_provider_outcome_unknown"}));
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/images/runs")?Promise.resolve(Response.json({items:[{runId,contextKind:"acquisition",contextId:operation,status:"blocked",targetPlatform:"product",createdAt:now}],nextCursor:""})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByRole("heading",{name:"未生成，点数已释放"});
 expect(screen.getByRole("option",{name:/未生成，点数已释放/})).toBeInTheDocument();
 expect(screen.getByText("本次未生成图片，原预留 20 点已释放。")).toBeVisible();
 const disclosure=screen.getByText("原始失败记录与核实依据").closest("details")!;
 fireEvent.click(within(disclosure).getByText("原始失败记录与核实依据"));
 expect(within(disclosure).getByText("原始状态：已阻止")).toBeInTheDocument();
 expect(within(disclosure).getAllByText(/slot_provider_outcome_unknown/).length).toBeGreaterThan(0);
 expect(screen.queryByText("原调用待核实，点数预留保留。")).toBeNull();
 expect(screen.queryByRole("button",{name:"核实原调用"})).toBeNull();
 expect(screen.queryByRole("button",{name:"取消剩余工作"})).toBeNull();
 expect(screen.getByRole("button",{name:"刷新原任务"})).toBeEnabled();
 expect(state.status).toBe("blocked");
 expect(state.slots[0].errorCode).toBe("slot_provider_outcome_unknown");
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it.each(["unknown","partial","unsettled","executing"])("does not accept a contradictory release summary for %s",scenario=>{
 const original=projection("blocked",templateId);
 const released={...original,settledPoints:0,resultDigest:"",releaseSummary:{releasedPoints:20},slots:original.slots.map(slot=>({...slot,status:"blocked",candidates:[] as typeof slot.candidates,closure:null,errorCode:"slot_provider_outcome_unknown"}))};
 if(scenario==="unknown")released.releaseSummary.releasedPoints=0;
 if(scenario==="partial")released.slots[0]={...original.slots[0],closure:null};
 if(scenario==="unsettled")released.settledPoints=10;
 if(scenario==="executing")released.status="executing";
 expect(imageSetRunSchema.safeParse(released).success).toBe(false);
});

it.each(["recover","cancel","restart","confirm","resume"] as const)("archives the settled %s request and unlocks task switching and a new plan after GET verification",async action=>{
 state=projection("blocked",templateId);
 Object.assign(state,{settledPoints:0,resultDigest:"",releaseSummary:{releasedPoints:20},approvalAvailable:false,candidateSelectionAvailable:false,regenerationAvailable:false,block:{Code:"slot_provider_outcome_unknown"},recoverableEffects:[{SlotID:"main",Attempt:1,Code:"slot_provider_outcome_unknown"}]});
 state.slots=state.slots.map(slot=>({...slot,status:"blocked",candidates:[],closure:null,errorCode:"slot_provider_outcome_unknown"}));
 const body=action==="restart"?{planRevision:1,planDigest:sha,quoteDigest:digest}:action==="recover"?{actionId:templateId,planRevision:1,slotId:"main",attempt:1}:action==="confirm"?{actionId:templateId,planRevision:1,planDigest:sha,quoteDigest:digest}:action==="cancel"?{actionId:templateId,planRevision:1}:{actionId:templateId};
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`,saved=JSON.stringify({action,runId,body});
 localStorage.setItem(key,saved);
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>new URL(String(url),"https://app.test").pathname.endsWith("/images/runs")?Promise.resolve(Response.json({items:[runId,operation].map(id=>({runId:id,contextKind:"acquisition",contextId:operation,status:"blocked",targetPlatform:"product",createdAt:now})),nextCursor:String(url).includes("cursor=")?"":"older"})):String(url).endsWith(`/runs/${operation}`)?Promise.resolve(Response.json({...projection("awaiting_final_approval",templateId),runId:operation})):String(url).endsWith("/prepare")?Promise.resolve(Response.json({...projection(),runId:templateId},{status:201})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByRole("heading",{name:"未生成，点数已释放"});
 await waitFor(()=>expect(screen.getByRole("combobox",{name:"本商品最近任务"})).toBeDisabled());
 expect(screen.queryByRole("button",{name:/继续原操作|继续原确认/})).toBeNull();
 expect(screen.getByText(/原请求和编号保留，仅支持只读核实/)).toBeVisible();
 fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(screen.getByRole("combobox",{name:"本商品最近任务"})).toBeEnabled());
 expect(localStorage.getItem(key)).toBeNull();
 expect(Object.keys(localStorage).some(record=>record.startsWith(key.replace(/:intent$/,":verified-intent:"))&&localStorage.getItem(record)===saved)).toBe(true);
 expect(screen.getByRole("button",{name:"加载更多图片任务"})).toBeEnabled();
 fireEvent.click(screen.getByRole("button",{name:"加载更多图片任务"}));
 await waitFor(()=>expect(screen.queryByRole("button",{name:"加载更多图片任务"})).toBeNull());
 const retained=screen.getByText("已核实的原请求").closest("section")!;
 expect(within(retained).getByText(saved)).toBeInTheDocument();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
 expect(screen.queryByRole("button",{name:/继续原操作|继续原确认/})).toBeNull();
 fireEvent.change(screen.getByRole("combobox",{name:"本商品最近任务"}),{target:{value:operation}});
 await waitFor(()=>expect(screen.getByRole("combobox",{name:"本商品最近任务"})).toHaveValue(operation));
 await prepare();
 const posts=fetch.mock.calls.filter(([,init])=>init?.method==="POST");
 expect(posts).toHaveLength(1);expect(String(posts[0][0])).toMatch(/\/prepare$/);
 expect(within(retained).getByText(saved)).toBeInTheDocument();
});

it.each(["another_run","new_regeneration"])("does not suppress an independent pending %s because the selected run was released",async scenario=>{
 state=projection("blocked",templateId);
 Object.assign(state,{settledPoints:0,resultDigest:"",releaseSummary:{releasedPoints:20},approvalAvailable:false,candidateSelectionAvailable:false,regenerationAvailable:false});
 state.slots=state.slots.map(slot=>({...slot,status:"blocked",candidates:[],closure:null}));
 const intent=scenario==="another_run"?{action:"cancel",runId:operation,body:{actionId:templateId,planRevision:1}}:{action:"regenerate",runId,requestKey:operation,body:{expectedModeRevision:"0",target:{Platform:"product"}}};
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`,saved=JSON.stringify(intent);
 localStorage.setItem(key,saved);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByRole("heading",{name:"未生成，点数已释放"});
 expect(screen.getByRole("button",{name:"继续原操作"})).toBeVisible();
 expect(localStorage.getItem(key)).toBe(saved);
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

function savedReleasedRequest(){
 const released=projection("blocked",templateId);
 Object.assign(released,{settledPoints:0,resultDigest:"",releaseSummary:{releasedPoints:20},approvalAvailable:false,candidateSelectionAvailable:false,regenerationAvailable:false,block:{Code:"slot_provider_outcome_unknown"},recoverableEffects:[{SlotID:"main",Attempt:1,Code:"slot_provider_outcome_unknown"}]});
 released.slots=released.slots.map(slot=>({...slot,status:"blocked",candidates:[],closure:null,errorCode:"slot_provider_outcome_unknown"}));
 state=released;
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`;
 const raw=JSON.stringify({action:"recover",runId,body:{actionId:templateId,planRevision:1,slotId:"main",attempt:1}},null,2)+"\n";
 localStorage.setItem(key,raw);return {key,raw};
}

it("retains the exact request across remount and a later uncertain preparation without unlocking that new request",async()=>{
 const {key,raw}=savedReleasedRequest();
 let view=render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(localStorage.getItem(key)).toBeNull());
 const archive=Object.keys(localStorage).find(k=>k.includes(":verified-intent:"))!;
 expect(localStorage.getItem(archive)).toBe(raw);
 view.unmount();view=render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 const retained=await screen.findByText("已核实的原请求");
 expect(retained.closest("section")!.querySelector("pre")!.textContent).toBe(raw);
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/prepare")?Promise.resolve(Response.json({code:"OUTCOME_UNKNOWN"},{status:503})):real(url,init));
 await prepare();
 await screen.findByText("本次请求结果尚未确认，请核实原请求。");
 const newPending=localStorage.getItem(key)!;expect(JSON.parse(newPending).action).toBe("prepare");
 expect(localStorage.getItem(archive)).toBe(raw);
 view.unmount();render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByText("已核实的原请求");
 expect(screen.getByRole("combobox",{name:"图片模板"})).toBeDisabled();
 expect(screen.getByRole("button",{name:"继续原操作"})).toBeVisible();
 expect(localStorage.getItem(key)).toBe(newPending);
 expect(localStorage.getItem(archive)).toBe(raw);
 expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(1);
});

it.each(["quota","collision"])("preserves the pending request and operation lock when archive %s fails",async failure=>{
 const {key,raw}=savedReleasedRequest();
 if(failure==="collision")localStorage.setItem(key.replace(/:intent$/,`:verified-intent:${runId}:recover:${templateId}`),raw.replace('"main"','"detail"'));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByRole("heading",{name:"未生成，点数已释放"});
 if(failure==="quota"){
  const set=Storage.prototype.setItem;
  vi.spyOn(Storage.prototype,"setItem").mockImplementation(function(this:Storage,k:string,v:string){if(k.includes(":verified-intent:"))throw new DOMException("full","QuotaExceededError");set.call(this,k,v)});
 }
 fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));
 await screen.findByText("原请求未能完整保留，请继续核实；任务切换和新计划暂未解锁。");
 expect(localStorage.getItem(key)).toBe(raw);
 expect(screen.getByRole("combobox",{name:"图片模板"})).toBeDisabled();
 if(failure==="collision")expect(Object.keys(localStorage).filter(k=>k.includes(":verified-intent:")).map(k=>localStorage.getItem(k))).toEqual([raw.replace('"main"','"detail"')]);
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it.each([403,404,"unknown"] as const)("does not detach a pending request after unsuccessful or unknown read verification: %s",async response=>{
 const {key,raw}=savedReleasedRequest();
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByRole("heading",{name:"未生成，点数已释放"});
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith(`/runs/${runId}`)?Promise.resolve(response==="unknown"?Response.json({...state,releaseSummary:undefined}):Response.json({code:response===403?"FORBIDDEN":"IMAGE_NOT_FOUND"},{status:response})):real(url,init));
 fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"核实原请求"})).toBeEnabled());
 expect(localStorage.getItem(key)).toBe(raw);
 expect(Object.keys(localStorage).some(k=>k.includes(":verified-intent:"))).toBe(false);
 expect(screen.getByRole("combobox",{name:"图片模板"})).toBeDisabled();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it("does not display another enterprise's archived request or let it unlock a pending request",async()=>{
 const {key,raw}=savedReleasedRequest();
 const view=render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(localStorage.getItem(key)).toBeNull());
 const otherKey=key.replace(":org:",":other-org:");localStorage.setItem(otherKey,raw);
 calls.context={...calls.context,effectiveOrganization:{id:"other-org"}};
 view.rerender(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByRole("heading",{name:"未生成，点数已释放"});
 expect(screen.queryByText("已核实的原请求")).toBeNull();
 expect(screen.getByRole("combobox",{name:"图片模板"})).toBeDisabled();
 expect(localStorage.getItem(otherKey)).toBe(raw);
 expect(Object.keys(localStorage).filter(k=>k.includes(":verified-intent:")).map(k=>localStorage.getItem(k))).toEqual([raw]);
});

it("can retry an interrupted detach without duplicating or losing the retained request",async()=>{
 const {key,raw}=savedReleasedRequest();
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByRole("heading",{name:"未生成，点数已释放"});
 const remove=Storage.prototype.removeItem;let interrupted=true;
 vi.spyOn(Storage.prototype,"removeItem").mockImplementation(function(this:Storage,k:string){if(k===key&&interrupted)throw new Error("interrupted");remove.call(this,k)});
 fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));
 await screen.findByText("原请求未能完整保留，请继续核实；任务切换和新计划暂未解锁。");
 expect(localStorage.getItem(key)).toBe(raw);
 expect(Object.keys(localStorage).filter(k=>k.includes(":verified-intent:")).map(k=>localStorage.getItem(k))).toEqual([raw]);
 interrupted=false;fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(localStorage.getItem(key)).toBeNull());
 expect(Object.keys(localStorage).filter(k=>k.includes(":verified-intent:")).map(k=>localStorage.getItem(k))).toEqual([raw]);
 expect(screen.getByRole("combobox",{name:"图片模板"})).toBeEnabled();
});

it("ignores a released verification response after switching enterprise",async()=>{
 const {key,raw}=savedReleasedRequest();
 const view=render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByRole("heading",{name:"未生成，点数已释放"});
 let finish!:(response:Response)=>void;const late=new Promise<Response>(resolve=>{finish=resolve});
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith(`/runs/${runId}`)?late:real(url,init));
 fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));
 calls.context={...calls.context,effectiveOrganization:{id:"other-org"}};
 view.rerender(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 await screen.findByRole("combobox",{name:"图片模板"});
 await act(async()=>finish(Response.json(state)));
 expect(localStorage.getItem(key)).toBe(raw);
 expect(Object.keys(localStorage).some(k=>k.includes(":verified-intent:"))).toBe(false);
 expect(screen.queryByText("已核实的原请求")).toBeNull();
});

it("keeps a saved approval pending rather than inferring its receipt from the run release",async()=>{
 const {key}=savedReleasedRequest();
 const raw=JSON.stringify({action:"approve",runId,body:{actionId:templateId,planRevision:1,selectionDigest:digest,expectedHead:{action_id:"",payload_hash:""},resultDigest:sha,choices:[{kind:"source",source_id:"original",presentation:{group:"carousel",order:1}}]}});
 localStorage.setItem(key,raw);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByRole("heading",{name:"未生成，点数已释放"});
 fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"核实原请求"})).toBeEnabled());
 expect(localStorage.getItem(key)).toBe(raw);
 expect(Object.keys(localStorage).some(k=>k.includes(":verified-intent:"))).toBe(false);
 expect(screen.getByRole("combobox",{name:"图片模板"})).toBeDisabled();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it("unlocks a confirmed released request without depending on optional inventory hydration",async()=>{
 const {key,raw}=savedReleasedRequest();
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByRole("heading",{name:"未生成，点数已释放"});
 await waitFor(()=>expect(screen.getByRole("button",{name:"核实原请求"})).toBeEnabled());
 const inventoryReads=fetch.mock.calls.filter(([url])=>String(url).endsWith("/inventory")).length;
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/inventory")?Promise.resolve(Response.json({code:"IMAGE_UNAVAILABLE"},{status:503})):real(url,init));
 fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(localStorage.getItem(key)).toBeNull());
 expect(screen.getByRole("combobox",{name:"图片模板"})).toBeEnabled();
 expect(Object.keys(localStorage).filter(k=>k.includes(":verified-intent:")).map(k=>localStorage.getItem(k))).toEqual([raw]);
 expect(fetch.mock.calls.filter(([url])=>String(url).endsWith("/inventory"))).toHaveLength(inventoryReads);
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it("allows method inspection with an authorized empty source while model readiness keeps preparation closed",async()=>{
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/agents/product.image.agent")?Promise.resolve(Response.json({...entry,canUse:false,capabilities:[{...entry.capabilities[0],readiness:"UNAVAILABLE",reason:"模型配置尚未就绪"}]})):String(url).endsWith("/images/sources")?Promise.resolve(Response.json({contextKind:"acquisition",contextId:operation,source,manualReplacementAvailable:false,originals:[],evidence:{}})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 await screen.findByLabelText("使用的方法/技能（可选）");
 expect(screen.getByText("模型配置尚未就绪")).toBeVisible();
 expect(screen.getByText("当前商品没有原始图片，补充真实素材后才能准备生成计划。")).toBeVisible();
 expect(fetch.mock.calls.some(([url,init])=>String(url).endsWith("/prepare")||init?.method==="POST")).toBe(false);
});

it("does not offer image method selection without image write permission",async()=>{
 calls.context={...calls.context,permissions:["listingkit.image_agent.read"]};
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 await screen.findByText("商品图片智能体");
 await waitFor(()=>expect(screen.queryByText("正在读取企业配置与真实商品素材…")).toBeNull());
 expect(screen.queryByLabelText("使用的方法/技能（可选）")).toBeNull();
 expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/skills/organization/image-candidates"))).toBe(false);
});

it("does not mount image readers or show an unavailable card without image read permission",async()=>{
 calls.context={...calls.context,permissions:["workbench.supply.read"]};
 render(<ProductImageSetPanel kind="supply" contextId={operation}/>);
 expect(screen.queryByText("商品图片智能体")).toBeNull();
 expect(fetch).not.toHaveBeenCalled();
 expect(calls.mode).not.toHaveBeenCalled();
});

it.each([["image-method-prompt-v1",1],["image-method-visual-brief-v2",2],["image-method-visual-brief-v3",2]] as const)("displays the exact selected image method under %s",async(policy,version)=>{
 const skillId="2bb819bd-af13-4e68-95e0-1cc86130fb1f",skillDigest="f".repeat(64),date="2026-10-18T00:00:00Z";
 const method={skill:{id:skillId,owner:{kind:"PLATFORM",id:"official"},creator:"maintainer",headVersion:version,revision:3,createdAt:date,updatedAt:date},version,package:{name:"product-image-standard",displayName:"标准商品套图",description:"围绕已选任务组织表达",target:"product-image",digest:skillDigest,files:null},state:"ENABLED",visibility:"SHARED",allowCopy:true,allowExport:true,createdAt:date};
 const binding={Skill:{kind:"skill-version-v1",id:skillId,ownerKind:"PLATFORM",ownerId:"official",version,digest:skillDigest,policyVersion:"skill-method-v1"},PolicyVersion:policy};
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/skills/organization/image-candidates")?Promise.resolve(Response.json({items:[method]})):String(url).endsWith("/prepare")?Promise.resolve(Response.json({...state,plan:{...state.plan,Method:binding}},{status:201})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 await screen.findByRole("option",{name:/标准商品套图/});fireEvent.change(screen.getByLabelText("使用的方法/技能（可选）"),{target:{value:`${skillId}:${version}`}});
 await prepare();await screen.findByText(/计划使用的方法：标准商品套图/);
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"));expect(JSON.parse(String(request?.[1]?.body)).skill).toEqual({skillId,version,digest:skillDigest});
 if(policy==="image-method-prompt-v1"){expect(screen.getByRole("button",{name:"确认点数并生成"})).toBeDisabled();expect(screen.getByText(/该计划保留历史图片方法/)).toBeInTheDocument()}else expect(screen.getByRole("button",{name:"确认点数并生成"})).toBeEnabled();
});
it("preselects the library image version and passes it with explicitly selected existing originals",async()=>{
 const skillId="2bb819bd-af13-4e68-95e0-1cc86130fb1f",date="2026-10-18T00:00:00Z",selection={skillId,version:2,digest:"f".repeat(64)};
 const method={skill:{id:skillId,owner:{kind:"PLATFORM",id:"official"},creator:"maintainer",headVersion:3,revision:3,createdAt:date,updatedAt:date},version:2,package:{name:"product-image-standard",displayName:"标准商品套图",description:"原图方法",target:"product-image",digest:selection.digest,files:null},state:"ENABLED",visibility:"SHARED",allowCopy:true,allowExport:true,createdAt:date};
 const real=fetch.getMockImplementation()!;fetch.mockImplementation((url,init)=>String(url).endsWith("/skills/organization/image-candidates")?Promise.resolve(Response.json({items:[method]})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialSkill={selection}/>);await screen.findByRole("option",{name:/标准商品套图/});expect(screen.getByLabelText("使用的方法/技能（可选）")).toHaveValue(`${skillId}:2`);expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);await prepare();await screen.findByRole("button",{name:"确认点数并生成"});const request=fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"))!;expect(JSON.parse(String(request[1]!.body))).toMatchObject({skill:selection,sharedOriginalIds:["original"]});
});
it.each([0,1,2])("discovers preset %s beyond a same-name mismatch and prepares its exact persisted version with the original skill",async index=>{
 const {selection,exact}=genericEntry(index);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialSkill={selection}/>);
 await waitFor(()=>expect(screen.getByRole("combobox",{name:"图片模板"})).toHaveValue(`${exact.templateId}:7`));
 expect(screen.getByLabelText("使用的方法/技能（可选）")).toHaveValue(`${selection.skillId}:2`);
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
 fireEvent.click(await screen.findByRole("checkbox",{name:"共用原始素材 素材 1"}));
 const prepare=screen.getByRole("button",{name:"准备整套图片计划（1 项）"});await waitFor(()=>expect(prepare).toBeEnabled());fireEvent.click(prepare);
 await screen.findByRole("button",{name:"确认点数并生成"});
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"))!;
 expect(JSON.parse(String(request[1]!.body))).toMatchObject({template:{templateId:exact.templateId,revision:"7"},skill:selection,sharedOriginalIds:["original"],selectedTaskIds:[...exact.image.carousel,...exact.image.detail].map(t=>t.id)});
 expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(1);
});
it("preserves the enterprise default's pinned version and tasks instead of a skill preset",async()=>{
 const {selection}=genericEntry(0,{hasDefault:true});
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialSkill={selection}/>);
 await waitFor(()=>expect(screen.getByRole("combobox",{name:"图片模板"})).toHaveValue(`${templateId}:1`));
 expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeInTheDocument();
 expect(fetch.mock.calls.some(([url])=>String(url).includes("cursor=exact-next"))).toBe(false);
});
it("does not hide an unreadable configured default by selecting a generic template",async()=>{
 const {selection}=genericEntry(0,{hasDefault:true,failDefault:true});
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialSkill={selection}/>);
 const select=await screen.findByRole("combobox",{name:"图片模板"});expect(select).toHaveValue("");
 expect(fetch.mock.calls.some(([url])=>String(url).includes("cursor=exact-next"))).toBe(false);
});
it.each(["archived","wrong-version"])("keeps an %s default unavailable without substituting a generic version",async mismatch=>{
 const {selection}=genericEntry(0,{hasDefault:true});const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).includes("/revisions/")?Promise.resolve(Response.json({...template,...(mismatch==="archived"?{lifecycle:"ARCHIVED"}:{version:"2"})})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialSkill={selection}/>);
 await screen.findByText(/配置已变化/);
 expect(screen.getByRole("combobox",{name:"图片模板"})).toHaveValue("");
 expect(fetch.mock.calls.some(([url])=>String(url).includes("cursor=exact-next"))).toBe(false);
});
it("keeps the original pending request byte-for-byte and skips generic discovery",async()=>{
 const {selection,storage,raw}=genericEntry(0,{pending:true});
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialSkill={selection}/>);
 await screen.findByRole("button",{name:"核实原请求"});
 expect(localStorage.getItem(storage)).toBe(raw);
 expect(fetch.mock.calls.some(([url])=>String(url).includes("cursor=exact-next"))).toBe(false);
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
it("does not overwrite a manual template choice with delayed generic discovery",async()=>{
 const {selection,exact}=genericEntry(0);
 const real=fetch.getMockImplementation()!;let release!:(response:Response)=>void;
 fetch.mockImplementation((url,init)=>String(url).includes("cursor=exact-next")?new Promise(resolve=>{release=resolve}):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialSkill={selection}/>);
 await waitFor(()=>expect(release).toBeTypeOf("function"));
 fireEvent.change(screen.getByRole("combobox",{name:"图片模板"}),{target:{value:`${templateId}:1`}});
 await act(async()=>release(Response.json({items:[exact],nextCursor:""})));
 await screen.findByLabelText("使用的方法/技能（可选）");
 expect(screen.getByRole("combobox",{name:"图片模板"})).toHaveValue(`${templateId}:1`);
 expect(screen.queryByText(/已带入通用模板/)).toBeNull();
});
it("does not rediscover a template when restoring an existing frozen run",async()=>{
 const {selection}=genericEntry(0);localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialSkill={selection}/>);
 await screen.findByRole("button",{name:"确认点数并生成"});
 expect(fetch.mock.calls.some(([url])=>String(url).includes("cursor=exact-next"))).toBe(false);
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
it("bounds automatic template discovery and leaves manual pagination available",async()=>{
 const {selection}=genericEntry(0);const real=fetch.getMockImplementation()!;let pages=0;
 fetch.mockImplementation((url,init)=>new URL(String(url),"https://app.test").pathname.endsWith("/templates")?Promise.resolve(Response.json({items:[],nextCursor:`next-${++pages}`})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialSkill={selection}/>);
 await screen.findByText(/未找到已保存的“通用商品主图”/);
 expect(pages).toBe(20);expect(screen.getByRole("button",{name:"读取更多模板"})).toBeEnabled();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
it("does not unlock or overwrite an uncertain original request when a library skill arrives",async()=>{
 const selection={skillId:templateId,version:2,digest:sha},storage=`product-image-set:actor:org:acquisition:${operation}:intent`;
 const raw=JSON.stringify({action:"confirm",runId,body:{actionId:operation,planRevision:1,planDigest:sha,quoteDigest:digest}});localStorage.setItem(storage,raw);state=projection("executing",operation);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialSkill={selection}/>);await screen.findByRole("button",{name:"核实原请求"});expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeDisabled();expect(localStorage.getItem(storage)).toBe(raw);expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
it.each(["prepare","regenerate"] as const)("refreshes the mode after a definite %s rejection before creating a new intent",async action=>{
 const initial=await calls.mode();calls.mode.mockClear();let complete!:(value:unknown)=>void;
 calls.mode.mockResolvedValueOnce(initial).mockImplementationOnce(()=>new Promise(resolve=>{complete=resolve}));
 const real=fetch.getMockImplementation()!;let rejected=false;
 fetch.mockImplementation((input,init)=>{if(String(input).endsWith(`/${action}`)){if(!rejected){rejected=true;return Promise.resolve(Response.json({code:"IMAGE_CONFIGURATION_CHANGED"},{status:409}))}throw new Error("new plan acknowledgement lost")}return real(input,init)});
 if(action==="regenerate")state=projection("awaiting_final_approval",templateId);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={action==="regenerate"?runId:undefined}/>);
 const name=action==="prepare"?"准备整套图片计划（2 项）":"准备所选 1 项的新计划";
 if(action==="prepare")await prepare();else{
  const checks=await screen.findAllByRole("checkbox",{name:"重新生成此项，另行确认点数"});fireEvent.click(checks[0]);
  const button=screen.getByRole("button",{name});await waitFor(()=>expect(button).toBeEnabled());fireEvent.click(button);
 }
 await screen.findByRole("alert");await waitFor(()=>expect(calls.mode).toHaveBeenCalledTimes(2));
 const button=screen.getByRole("button",{name});expect(button).toBeDisabled();
 const attempts=()=>fetch.mock.calls.filter(v=>String(v[0]).endsWith(`/${action}`));expect(attempts()).toHaveLength(1);
 await act(async()=>complete({...initial,mode:"full_access",revision:"1"}));await waitFor(()=>expect(button).toBeEnabled());
 fireEvent.click(button);await waitFor(()=>expect(attempts()).toHaveLength(2));
 expect(JSON.parse(String(attempts()[0][1]?.body)).expectedModeRevision).toBe("0");expect(JSON.parse(String(attempts()[1][1]?.body)).expectedModeRevision).toBe("1");
 expect(new Headers(attempts()[1][1]?.headers).get("Idempotency-Key")).not.toBe(new Headers(attempts()[0][1]?.headers).get("Idempotency-Key"));
});
it.each(["acquisition","supply"] as const)("binds historical %s originals to the run while fresh plans keep current sources",async(kind)=>{
 const historical={...source,ContextKind:kind,EffectiveVersion:"2",...kind==="supply"?{ApplyReceiptID:operation}:{}};
 const current={...historical,EffectiveVersion:"3",...kind==="supply"?{ApplyReceiptID:templateId}:{}};
 state={...projection("awaiting_final_approval",templateId),plan:{...projection().plan,Source:historical}};
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((input,init)=>{
  const url=new URL(String(input),"http://localhost");
  if(url.pathname.endsWith("/images/sources")){
   const pinned=url.searchParams.get("effectiveCatalogVersion")==="2",binding=pinned?historical:current;
   return Promise.resolve(Response.json({contextKind:kind,contextId:operation,source:binding,manualReplacementAvailable:false,evidence:{},originals:[{id:"original",displayUrl:"https://images.test/original.png",width:1024,height:1024},{id:pinned?"historical-only":"current-only",displayUrl:`https://images.test/${pinned?"historical":"current"}.png`,width:1024,height:1024}]}));
  }
  return real(input,init);
 });
 render(<ProductImageSetPanel kind={kind} contextId={operation} effectiveVersion="3" applyReceiptId={kind==="supply"?templateId:undefined} initialRunId={runId}/>);
 fireEvent.click(await screen.findByRole("button",{name:"选择原图 2"}));
 expect(screen.getByRole("img",{name:"拟采用：原图 2"})).toHaveAttribute("src","https://images.test/historical.png");
 fireEvent.click(screen.getByRole("button",{name:"预览完整选择"}));await screen.findByRole("button",{name:"人工批准并保存素材"});
 const preview=fetch.mock.calls.find(([url])=>String(url).endsWith("/preview"))!;
 expect(preview[0]).toContain(`/runs/${runId}/preview`);expect(JSON.parse(String(preview[1]!.body)).choices[0].source_id).toBe("historical-only");
 const pinned=new URL(String(fetch.mock.calls.find(([url])=>String(url).includes("/images/sources?effectiveCatalogVersion=2"))![0]),"http://localhost");
 expect(pinned.searchParams.get("applyReceiptId")).toBe(kind==="supply"?operation:null);
 fireEvent.click(screen.getByRole("checkbox",{name:"共用原始素材 素材 2"}));
 const button=screen.getByRole("button",{name:"准备整套图片计划（2 项）"});await waitFor(()=>expect(button).toBeEnabled());fireEvent.click(button);
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/prepare"))).toBe(true));
 const fresh=JSON.parse(String(fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"))![1]!.body));
 expect(fresh).toMatchObject({effectiveCatalogVersion:"3",sharedOriginalIds:["current-only"]});
 if(kind==="supply")expect(fresh.applyReceiptId).toBe(templateId);
 expect(fetch.mock.calls.some(([url])=>/\/(confirm|regenerate|approve)$/.test(String(url)))).toBe(false);
});
it.each(["unavailable","mismatched"])("keeps restored results without falling back to current originals when pinned sources are %s",async(mode)=>{
 const historical={...source,EffectiveVersion:"2"},current={...source,EffectiveVersion:"3"};
 state={...projection("awaiting_final_approval",templateId),plan:{...projection().plan,Source:historical}};
 let ready=false;const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((input,init)=>{
  const url=new URL(String(input),"http://localhost");
  if(url.pathname.endsWith("/images/sources")){
   const pinned=url.searchParams.get("effectiveCatalogVersion")==="2";
   if(pinned&&!ready&&mode==="unavailable")return Promise.resolve(Response.json({code:"IMAGE_UNAVAILABLE"},{status:503}));
   return Promise.resolve(Response.json({contextKind:"acquisition",contextId:operation,source:pinned&&ready?historical:current,manualReplacementAvailable:false,evidence:{},originals:[{id:pinned&&ready?"historical":"current",displayUrl:"https://images.test/source.png",width:1024,height:1024}]}));
  }return real(input,init);
 });
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} effectiveVersion="3" initialRunId={runId}/>);
 expect(await screen.findByRole("alert")).toHaveTextContent(mode==="unavailable"?"当前图片执行或依赖不可用":"原任务或版本已变化");
 expect(await screen.findAllByRole("button",{name:"选择采用"})).toHaveLength(2);
 expect(screen.queryByRole("button",{name:/选择原图/})).not.toBeInTheDocument();
 ready=true;fireEvent.click(screen.getByRole("button",{name:"刷新原任务"}));fireEvent.click(await screen.findByRole("button",{name:"选择原图 1"}));
 expect(screen.getByRole("img",{name:"拟采用：原图 1"})).toHaveAttribute("src","https://images.test/source.png");
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
it("restores a historical run even when the current source version is unavailable",async()=>{
 const historical={...source,ContextKind:"supply",EffectiveVersion:"2",ApplyReceiptID:operation};state={...projection("awaiting_final_approval",templateId),plan:{...projection().plan,Source:historical}};
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((input,init)=>{
  const url=new URL(String(input),"http://localhost");
  if(url.pathname.endsWith("/images/sources"))return Promise.resolve(url.searchParams.get("effectiveCatalogVersion")==="2"?Response.json({contextKind:"supply",contextId:operation,source:historical,manualReplacementAvailable:false,evidence:{},originals:[{id:"historical",displayUrl:"https://images.test/historical.png",width:1024,height:1024}]}):Response.json({code:"IMAGE_UNAVAILABLE"},{status:503}));
  if(url.pathname.endsWith("/regenerate"))return Promise.resolve(Response.json(state,{status:201}));
  return real(input,init);
 });
 render(<ProductImageSetPanel kind="supply" contextId={operation} effectiveVersion="3" applyReceiptId={templateId} initialRunId={runId}/>);
 fireEvent.click(await screen.findByRole("button",{name:"选择原图 1"}));
 expect(screen.getByRole("img",{name:"拟采用：原图 1"})).toHaveAttribute("src","https://images.test/historical.png");
 expect(screen.queryByRole("button",{name:/准备整套图片计划/})).not.toBeInTheDocument();expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
 fireEvent.click(screen.getAllByRole("checkbox",{name:"重新生成此项，另行确认点数"})[0]);const regenerate=screen.getByRole("button",{name:"准备所选 1 项的新计划"});await waitFor(()=>expect(regenerate).toBeEnabled());fireEvent.click(regenerate);
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/regenerate"))).toBe(true));
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith("/regenerate"))!;expect(request[0]).toContain(`/runs/${runId}/regenerate`);
 expect(JSON.parse(String(request[1]!.body))).toMatchObject({effectiveCatalogVersion:"2",applyReceiptId:operation,selectedTaskIds:["main"],sharedOriginalIds:["original"],template:{templateId,revision:"1"}});
 expect(fetch.mock.calls.some(([url])=>/\/(prepare|confirm|approve)$/.test(String(url)))).toBe(false);
});
it("ignores a late historical source response after switching back to another run",async()=>{
 const secondId="44444444-4444-4444-8444-444444444444";
 state=projection("awaiting_final_approval",templateId);const second={...state,runId:secondId,plan:{...state.plan,Source:{...source,EffectiveVersion:"2"}}};
 let resolve:(response:Response)=>void=()=>{};const late=new Promise<Response>(r=>{resolve=r});const real=fetch.getMockImplementation()!;
 const payload=(version:string)=>({contextKind:"acquisition",contextId:operation,source:{...source,EffectiveVersion:version},manualReplacementAvailable:false,evidence:{},originals:[{id:`version-${version}`,displayUrl:`https://images.test/version-${version}.png`,width:1024,height:1024}]});
 fetch.mockImplementation((input,init)=>{
  const url=new URL(String(input),"http://localhost");
  if(url.pathname.endsWith("/images/sources"))return url.searchParams.get("effectiveCatalogVersion")==="2"?late:Promise.resolve(Response.json(payload("1")));
  if(url.pathname.endsWith(`/runs/${secondId}`))return Promise.resolve(Response.json(second));
  if(url.pathname.endsWith("/images/runs"))return Promise.resolve(Response.json({items:[state,second].map(p=>({runId:p.runId,contextKind:"acquisition",contextId:operation,status:p.status,targetPlatform:"product",createdAt:now})),nextCursor:""}));
  return real(input,init);
 });
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 const selector=await screen.findByLabelText("本商品最近任务");fireEvent.change(selector,{target:{value:secondId}});
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).includes("/images/sources?effectiveCatalogVersion=2"))).toBe(true));
 expect(screen.queryByRole("button",{name:"选择原图 1"})).not.toBeInTheDocument();
 fireEvent.change(selector,{target:{value:runId}});fireEvent.click(await screen.findByRole("button",{name:"选择原图 1"}));
 await act(async()=>resolve(Response.json(payload("2"))));
 expect(selector).toHaveValue(runId);expect(screen.getByRole("img",{name:"拟采用：原图 1"})).toHaveAttribute("src","https://images.test/version-1.png");
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
it.each(["disabled","unavailable","read-only"])("keeps admitted runs visible and refreshable when generation is %s",async(mode)=>{
 state=projection("awaiting_final_approval",templateId);const real=fetch.getMockImplementation()!;
 const unavailable={...entry,canUse:false,agent:{...entry.agent,activation:mode==="disabled"?"DISABLED":"ENABLED"},capabilities:entry.capabilities.map(c=>({...c,readiness:mode==="unavailable"?"UNAVAILABLE":c.readiness}))};
 if(mode==="read-only")calls.context.permissions=["listingkit.image_agent.read"];
 fetch.mockImplementation((url,init)=>String(url).endsWith("/product.image.agent")?Promise.resolve(Response.json(unavailable)):String(url).endsWith("/images/runs")?Promise.resolve(Response.json({items:[{runId,contextKind:"acquisition",contextId:operation,status:state.status,targetPlatform:"product",createdAt:now}],nextCursor:""})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 expect(await screen.findByRole("heading",{name:"商品识别主图"})).toBeInTheDocument();
 expect(await screen.findByLabelText("本商品最近任务")).toHaveValue(runId);
 expect(screen.queryByRole("button",{name:/准备整套图片计划/})).not.toBeInTheDocument();
 const reads=fetch.mock.calls.filter(([url])=>String(url).endsWith(`/runs/${runId}`)).length;
 fireEvent.click(screen.getByRole("button",{name:"刷新原任务"}));
 await waitFor(()=>expect(fetch.mock.calls.filter(([url])=>String(url).endsWith(`/runs/${runId}`)).length).toBeGreaterThan(reads));
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
 if(mode==="disabled"){
  fireEvent.click(screen.getAllByRole("button",{name:"选择采用"})[0]);fireEvent.click(screen.getByRole("button",{name:"预览完整选择"}));
  fireEvent.click(await screen.findByRole("button",{name:"人工批准并保存素材"}));
  await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/approve"))).toBe(true));
  expect(fetch.mock.calls.some(([url])=>/\/(prepare|regenerate|confirm)$/.test(String(url)))).toBe(false);
 }
});
it.each([
 "{bad JSON",
 JSON.stringify({action:"read",body:{}}),
 JSON.stringify({action:"confirm",runId,body:{actionId:operation,planRevision:"bad"}}),
 ...[undefined,"bad-key"].map(requestKey=>JSON.stringify({action:"prepare",requestKey,body:{target:{Platform:"product"}}})),
 ...[undefined,"bad-run"].map(runId=>JSON.stringify({action:"confirm",runId,body:{actionId:operation,planRevision:1,planDigest:sha,quoteDigest:digest}})),
 JSON.stringify({action:"regenerate",requestKey:operation,body:{target:{Platform:"product"}}}),
])("discards invalid saved intent %s while restoring the known run with GET only",async(saved)=>{
 state=projection("awaiting_final_approval",templateId);
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`;
 localStorage.setItem(key,saved);
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 expect(await screen.findAllByRole("button",{name:"选择采用"})).toHaveLength(2);
 await waitFor(()=>expect(localStorage.getItem(key)).toBeNull());
 expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/sources"))).toBe(true);
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/images/runs"))).toBe(true));
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
it("shows all eight exact references used by a generated slot",async()=>{
 state=projection("awaiting_final_approval",templateId);
 const originals=Array.from({length:8},(_,i)=>({ID:i?`original-${i+1}`:"original",DisplayURL:`https://images.test/reference-${i+1}.png`,Width:1024,Height:1024}));
 state={...state,originals,slots:state.slots.map((slot,i)=>i?slot:{...slot,recipe:{...slot.recipe,References:originals.map(image=>({AssetID:image.ID}))}})};
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 const heading=await screen.findByRole("heading",{name:"商品识别主图"});
 const images=within(heading.closest("article")!).getAllByRole("img",{name:"本图使用的原始素材"});
 expect(images.map(image=>image.getAttribute("src"))).toEqual(originals.map(image=>image.DisplayURL));
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
it("loads the next bounded recent-run page and reopens an older run with GET only",async()=>{
 state=projection("awaiting_final_approval",templateId);
 const cursor="older&after=20",item=(id:string)=>({runId:id,contextKind:"acquisition",contextId:operation,status:"awaiting_final_approval",targetPlatform:"product",createdAt:now});
 const first=Array.from({length:20},(_,i)=>item(`00000000-0000-4000-8000-${String(i+1).padStart(12,"0")}`));
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation(async(url,init)=>{
  const parsed=new URL(String(url),"https://app.test");
  if(parsed.pathname.endsWith("/images/runs"))return Response.json(parsed.searchParams.has("cursor")?{items:[first[0],item(runId)],nextCursor:""}:{items:first,nextCursor:cursor});
  return real(url,init);
 });
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"加载更多图片任务"}));
 const select=screen.getByRole("combobox",{name:"本商品最近任务"}) as HTMLSelectElement;
 await waitFor(()=>expect(select.options).toHaveLength(22));
 expect(screen.queryByRole("button",{name:"加载更多图片任务"})).not.toBeInTheDocument();
 fireEvent.change(select,{target:{value:runId}});
 expect(await screen.findAllByRole("button",{name:"选择采用"})).toHaveLength(2);
 const pages=fetch.mock.calls.filter(([url])=>new URL(String(url),"https://app.test").pathname.endsWith("/images/runs"));
 expect(pages).toHaveLength(2);expect(new URL(String(pages[1][0]),"https://app.test").searchParams.get("cursor")).toBe(cursor);
 expect(pages.every(([,init])=>init?.method==="GET")).toBe(true);
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
it("keeps the original recent-run cursor and list after a page failure and retries the same GET",async()=>{
 const cursor="original-page-cursor",item={runId,contextKind:"acquisition",contextId:operation,status:"awaiting_final_approval",targetPlatform:"product",createdAt:now};
 const real=fetch.getMockImplementation()!;let pages=0;
 fetch.mockImplementation(async(url,init)=>{
  const parsed=new URL(String(url),"https://app.test");
  if(parsed.pathname.endsWith("/images/runs")){
   if(!parsed.searchParams.has("cursor"))return Response.json({items:[item],nextCursor:cursor});
   if(++pages===1)return Response.json({code:"IMAGE_UNAVAILABLE"},{status:503});
   return Response.json({items:[],nextCursor:""});
  }
  return real(url,init);
 });
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"加载更多图片任务"}));
 await screen.findByRole("alert");
 const select=screen.getByRole("combobox",{name:"本商品最近任务"}) as HTMLSelectElement;
 expect(select.options).toHaveLength(2);
 const more=screen.getByRole("button",{name:"加载更多图片任务"});await waitFor(()=>expect(more).toBeEnabled());fireEvent.click(more);
 await waitFor(()=>expect(screen.queryByRole("button",{name:"加载更多图片任务"})).not.toBeInTheDocument());
 expect(select.options).toHaveLength(2);
 const requests=fetch.mock.calls.filter(([url])=>new URL(String(url),"https://app.test").searchParams.has("cursor"));
 expect(requests).toHaveLength(2);expect(requests.map(([url])=>new URL(String(url),"https://app.test").searchParams.get("cursor"))).toEqual([cursor,cursor]);
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
it("aborts a recent-run page and ignores its late items after switching enterprise",async()=>{
 const item={runId,contextKind:"acquisition",contextId:operation,status:"awaiting_final_approval",targetPlatform:"product",createdAt:now};
 let resolve!:(response:Response)=>void;
 const late=new Promise<Response>(ok=>{resolve=ok}),real=fetch.getMockImplementation()!;
 fetch.mockImplementation(async(url,init)=>{
  const parsed=new URL(String(url),"https://app.test");
  if(parsed.pathname.endsWith("/images/runs")){
   if(parsed.searchParams.has("cursor"))return late;
   return Response.json(new Headers(init?.headers).get("X-Expected-Organization-ID")==="other"?{items:[],nextCursor:""}:{items:[item],nextCursor:"older-page"});
  }
  return real(url,init);
 });
 const view=render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"加载更多图片任务"}));
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>new URL(String(url),"https://app.test").searchParams.has("cursor"))).toBe(true));
 const request=fetch.mock.calls.find(([url])=>new URL(String(url),"https://app.test").searchParams.has("cursor"))!;
 calls.context={...calls.context,effectiveOrganization:{id:"other"}};view.rerender(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 expect(request[1]!.signal?.aborted).toBe(true);
 await act(async()=>resolve(Response.json({items:[item],nextCursor:"another-old-page"})));
 await screen.findByRole("button",{name:"准备整套图片计划（2 项）"});
 expect(screen.queryByRole("combobox",{name:"本商品最近任务"})).not.toBeInTheDocument();
 expect(screen.queryByRole("button",{name:"加载更多图片任务"})).not.toBeInTheDocument();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
it("can select an untouched source original outside the generation references without generating again",async()=>{
 state=projection("awaiting_final_approval",templateId);
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation(async(url,init)=>new URL(String(url),"http://localhost").pathname.endsWith("/images/sources")?Response.json({contextKind:"acquisition",contextId:operation,source,manualReplacementAvailable:false,originals:[{id:"original",displayUrl:"https://images.test/original.png",width:1024,height:1024},{id:"untouched",displayUrl:"https://images.test/untouched.png",width:1024,height:1024}],evidence:{}}):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 fireEvent.click(await screen.findByRole("button",{name:"选择原图 2"}));
 expect(screen.getByRole("img",{name:"拟采用：原图 2"})).toHaveAttribute("src","https://images.test/untouched.png");
 fireEvent.click(screen.getByRole("button",{name:"预览完整选择"}));
 fireEvent.click(await screen.findByRole("button",{name:"人工批准并保存素材"}));
 await screen.findByText("本次批准已保存为正式商品素材。");
 for(const action of ["preview","approve"]){
  const request=fetch.mock.calls.find(([url])=>String(url).endsWith(`/${action}`))!;
  const body=JSON.parse(String(request[1]!.body));
  expect(body.choices).toEqual([{kind:"source",source_id:"untouched",presentation:{group:"carousel",order:1}}]);
  expect(body.planRevision).toBe(1);expect(body.resultDigest).toBe(sha);
 }
 expect(fetch.mock.calls.some(([url])=>/\/(prepare|regenerate|confirm)$/.test(String(url)))).toBe(false);
});
it("runs both groups from shared originals and saves only a previewed explicit complete selection",async()=>{
 const saved=vi.fn();render(<StrictMode><ProductImageSetPanel kind="acquisition" contextId={operation} onSaved={saved}/></StrictMode>);
 await prepare();fireEvent.click(await screen.findByRole("button",{name:"确认点数并生成"}));
 const choices=await screen.findAllByRole("button",{name:"选择采用"});expect(choices).toHaveLength(2);choices.forEach(button=>fireEvent.click(button));
 fireEvent.click(screen.getByRole("button",{name:"预览完整选择"}));fireEvent.click(await screen.findByRole("button",{name:"人工批准并保存素材"}));
 await screen.findByText("本次批准已保存为正式商品素材。");expect(saved).toHaveBeenCalledOnce();
 const preparation=fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"))!;
 const request=JSON.parse(String(preparation[1]!.body));expect(request.sharedOriginalIds).toEqual(["original"]);expect(request.selectedTaskIds).toEqual(["main","detail"]);
 const approval=fetch.mock.calls.find(([url])=>String(url).endsWith("/approve"))!;
 const selected=JSON.parse(String(approval[1]!.body));expect(selected.selectionDigest).toBe(digest);expect(selected.choices.map((v:{presentation:unknown})=>v.presentation)).toEqual([{group:"carousel",order:1},{group:"detail",order:1}]);
 expect(selected.choices.every((v:Record<string,unknown>)=>!("url" in v))).toBe(true);
});
it("retains an uncertain preparation across reload and verifies its original key with GET only",async()=>{
 const real=fetch.getMockImplementation()!;fetch.mockImplementation(async(url,init)=>{if(String(url).endsWith("/prepare"))throw new Error("lost acknowledgement");return real(url,init)});
 const view=render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);await prepare();await screen.findByRole("button",{name:"核实原请求"});
 const first=fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"))!;const key=new Headers(first[1]!.headers).get("Idempotency-Key");
 view.unmount();render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));await screen.findByRole("button",{name:"确认点数并生成"});
 expect(fetch.mock.calls.filter(([url])=>String(url).endsWith("/prepare"))).toHaveLength(1);
 expect(fetch.mock.calls.find(([url])=>String(url).endsWith(`/by-key/${key}`))?.[1]?.method).toBe("GET");
 expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).toBeNull();
});
it("does not treat another confirmation's status as proof of the frozen original command",async()=>{
 state=projection("executing",templateId);
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:intent`,JSON.stringify({action:"confirm",runId,body:{actionId:operation,planRevision:1,planDigest:sha,quoteDigest:digest}}));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith(`/runs/${runId}`))).toBe(true));
 expect(screen.getByRole("button",{name:"核实原请求"})).toBeInTheDocument();expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).not.toBeNull();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it("discards a late preparation response after the user changes enterprise",async()=>{
 let resolve!:(response:Response)=>void;
 const late=new Promise<Response>(ok=>{resolve=ok}),real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/prepare")?late:real(url,init));
 const view=render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);await prepare();
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/prepare"))).toBe(true));
 const first=fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"))!;
 calls.context={...calls.context,effectiveOrganization:{id:"other"}};view.rerender(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 expect(first[1]!.signal?.aborted).toBe(true);await act(async()=>resolve(Response.json(state,{status:201})));
 expect(screen.queryByRole("button",{name:"确认点数并生成"})).not.toBeInTheDocument();
 expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).not.toBeNull();
 expect(localStorage.getItem(`product-image-set:actor:other:acquisition:${operation}:run`)).toBeNull();
});

it("keeps an admitted result reviewable after enterprise activation is disabled",async()=>{
 state=projection("awaiting_final_approval",templateId);localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 const real=fetch.getMockImplementation()!;fetch.mockImplementation((url,init)=>String(url).endsWith("/agents/product.image.agent")?Promise.resolve(Response.json({...entry,agent:{...entry.agent,activation:"DISABLED"}})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 expect(await screen.findAllByRole("button",{name:"选择采用"})).toHaveLength(2);
 expect(screen.queryByRole("button",{name:"准备整套图片计划（2 项）"})).not.toBeInTheDocument();
});

for (const status of ["awaiting_plan_approval","executing"]) {
 it(`retains the original confirmation after GET reports ${status} and can resume the exact request`,async()=>{
  const body={actionId:operation,planRevision:1,planDigest:sha,quoteDigest:digest};
  state=projection(status,status==="executing"?operation:"");
  localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:intent`,JSON.stringify({action:"confirm",runId,body}));
  render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
  fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
  await waitFor(()=>expect(screen.getByRole("button",{name:"核实原请求"})).toBeEnabled());
  expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).not.toBeNull();
  fireEvent.click(screen.getByRole("button",{name:"继续原确认"}));
  await waitFor(()=>expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).toBeNull());
  const requests=fetch.mock.calls.filter(([url])=>String(url).endsWith("/confirm"));
  expect(requests).toHaveLength(1);expect(JSON.parse(String(requests[0][1]!.body))).toEqual(body);
 });
}
it("prepares the displayed immutable default content version even after the template head advances",async()=>{
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).includes("/revisions/")?Promise.resolve(Response.json({...template,revision:"2",version:"1"})):String(url).includes("/templates?")?Promise.resolve(Response.json({items:[{...template,revision:"2",version:"2"}],nextCursor:""})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);await prepare();
 await screen.findByRole("button",{name:"确认点数并生成"});
 const select=screen.getByLabelText("图片模板") as HTMLSelectElement;
 expect(select.selectedOptions[0].textContent).toBe("Two groups v1");
 await screen.findByRole("button",{name:"确认点数并生成"});
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"))!;
 expect(JSON.parse(String(request[1]!.body)).template).toEqual({templateId,revision:"1"});
});
for (const parentStatus of ["awaiting_final_approval","completed","failed","blocked","cancelled"]) {
it(`can combine new subset output with successful ${parentStatus} parent outputs without another generation`,async()=>{
 const parentId="44444444-4444-4444-8444-444444444444";
 const parent={...projection(parentStatus,templateId),runId:parentId};
 const child={...projection("awaiting_final_approval",operation),images:1,slots:projection("awaiting_final_approval").slots.slice(0,1),plan:{...state.plan,Regeneration:{RunID:parentId}}};
 state=child;localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith(`/runs/${parentId}`)?Promise.resolve(Response.json(parent)):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click((await screen.findAllByRole("button",{name:"选择采用"}))[0]);
 const original=await screen.findAllByRole("button",{name:"采用原任务结果"});expect(original).toHaveLength(2);expect(original[1]).toBeEnabled();fireEvent.click(original[1]);
 fireEvent.click(screen.getByRole("button",{name:"预览完整选择"}));
 await screen.findByRole("button",{name:"人工批准并保存素材"});
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith("/preview"))!;
 const choices=JSON.parse(String(request[1]!.body)).choices;
 expect(choices.map((v:{run_id:string,slot_id:string})=>[v.run_id,v.slot_id])).toEqual([[runId,"main"],[parentId,"detail"]]);
 expect(fetch.mock.calls.some(([url])=>/\/(prepare|regenerate|confirm)$/.test(String(url)))).toBe(false);
});
}

it("restores and prepares the exact large catalog version without numeric conversion",async()=>{
 const version="9223372036854775807",largeSource={...source,OriginalVersion:"9007199254740993",EffectiveVersion:version};
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation(async(url,init)=>{
  if(String(url).includes("/sources"))return Response.json({contextKind:"acquisition",contextId:operation,source:largeSource,manualReplacementAvailable:false,originals:[{id:"original",displayUrl:"https://images.test/original.png",width:1024,height:1024}],evidence:{}});
  if(String(url).endsWith("/prepare"))return Response.json({...state,plan:{...state.plan,Source:largeSource}},{status:201});
  return real(url,init);
 });
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);await prepare();
 await screen.findByRole("button",{name:"确认点数并生成"});
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"))!;
 expect(JSON.parse(String(request[1]!.body)).effectiveCatalogVersion).toBe(version);
});
it("uses the restored custom parent template rather than the current default for subset regeneration",async()=>{
 const parentTemplateId="55555555-5555-4555-8555-555555555555";
 const parentTemplate={...template,templateId:parentTemplateId,name:"Old custom",revision:"4",version:"2",image:{...template.image,mode:"custom",carousel:[{id:"old-custom",purpose:"custom",brief:"Original custom instruction"}],detail:[]}};
 state={...projection("awaiting_final_approval",operation),template:{templateId:parentTemplateId,revision:"2"},slots:[{...projection("awaiting_final_approval").slots[0]!,slotId:"old-custom"}]} as typeof state;
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).includes(`/templates/${parentTemplateId}/revisions/2`)?Promise.resolve(Response.json(parentTemplate)):String(url).endsWith("/regenerate")?Promise.resolve(Response.json(state,{status:201})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("checkbox",{name:"重新生成此项，另行确认点数"}));
 fireEvent.click(screen.getByRole("checkbox",{name:"共用原始素材 素材 1"}));
 const button=screen.getByRole("button",{name:"准备所选 1 项的新计划"});await waitFor(()=>expect(button).toBeEnabled());fireEvent.click(button);
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/regenerate"))).toBe(true));
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith("/regenerate"))!;
 expect(JSON.parse(String(request[1]!.body)).template).toEqual({templateId:parentTemplateId,revision:"2"});
 expect(JSON.parse(String(request[1]!.body)).selectedTaskIds).toEqual(["old-custom"]);
});
it("loads both legal near-limit templates through bounded pages",async()=>{
 const image={schema:"image-config-v1",mode:"custom",shareOriginals:true,background:"",language:"en",carousel:Array.from({length:32},(_,i)=>({id:`task-${i}`,purpose:"custom",brief:"x".repeat(1990)})),detail:[]};
 image.background="x".repeat(65520-new TextEncoder().encode(JSON.stringify(image)).length);
 const rows=["Large first","Large second"].map((name,i)=>imageTemplateSchema.parse({...template,templateId:`45a227bb-b572-4138-8e2a-5f1a0be9861${i}`,name:`${name} ${"😀".repeat(100)}`,image}));
 expect(new TextEncoder().encode(JSON.stringify({items:rows,nextCursor:""})).length).toBeGreaterThan(128*1024);
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation(async(input,init)=>{
  const url=new URL(String(input),"https://app.test");if(!url.pathname.endsWith("/templates"))return real(input,init);
  const page=url.searchParams.get("pageSize")==="1"?{items:[rows[url.searchParams.has("cursor")?1:0]],nextCursor:url.searchParams.has("cursor")?"":"next-page"}:{items:rows,nextCursor:""};
  return new TextEncoder().encode(JSON.stringify(page)).length>128*1024?Response.json({code:"DEPENDENCY_UNAVAILABLE"},{status:503}):Response.json(page);
 });
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 await screen.findByRole("option",{name:/Large first/});
 fireEvent.click(screen.getByRole("button",{name:"读取更多模板"}));
 await screen.findByRole("option",{name:/Large second/});
 fireEvent.change(screen.getByRole("combobox",{name:"图片模板"}),{target:{value:`${rows[1]!.templateId}:1`}});
 fireEvent.click(screen.getByRole("checkbox",{name:"共用原始素材 素材 1"}));
 expect(screen.getByRole("button",{name:"准备整套图片计划（32 项）"})).toBeEnabled();
 expect(fetch.mock.calls.filter(([url])=>String(url).includes("/templates?")).map(([url])=>new URL(String(url),"https://app.test").searchParams.get("pageSize"))).toEqual(["1","1"]);
});
for(const failure of ["templates","recent"]){
 it(`restores the saved run before optional ${failure} discovery fails`,async()=>{
  state=projection("awaiting_final_approval",templateId);
  localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
  const real=fetch.getMockImplementation()!;
  fetch.mockImplementation(async(url,init)=>String(url).includes(failure==="templates"?"/templates?":"/images/runs")&&!String(url).includes(`/runs/${runId}`)?Response.json({code:"DEPENDENCY_UNAVAILABLE"},{status:503}):real(url,init));
  render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
  const choices=await screen.findAllByRole("button",{name:"选择采用"});
  await screen.findByRole("alert");
  expect(choices).toHaveLength(2);choices.forEach(button=>expect(button).toBeEnabled());
  expect(screen.getByRole("button",{name:"选择原图 1"})).toBeEnabled();
  expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(0);
 });
}
for(const failure of ["read","inventory"]){
 it(`keeps current template preparation available when original run ${failure} fails`,async()=>{
  state=projection("awaiting_final_approval",templateId);
  const key=`product-image-set:actor:org:acquisition:${operation}:run`;localStorage.setItem(key,runId);
  const real=fetch.getMockImplementation()!;
  fetch.mockImplementation(async(url,init)=>String(url).endsWith(failure==="read"?`/runs/${runId}`:"/inventory")?Response.json({code:failure==="read"?"IMAGE_NOT_FOUND":"IMAGE_UNAVAILABLE"},{status:failure==="read"?404:503}):real(url,init));
  render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
  await screen.findByRole("alert");
  fireEvent.click(await screen.findByRole("checkbox",{name:"共用原始素材 素材 1"}));
  expect(screen.getByRole("button",{name:"准备整套图片计划（2 项）"})).toBeEnabled();
  expect(localStorage.getItem(key)).toBe(runId);
  expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(0);
 });
}
const sheinTarget={Platform:"shein",StoreID:"saved-store",Site:"US",CategoryID:123,RecordID:"saved-record"} as const;
const officialPlacement={Group:"spu",SKC:0,SKU:0,Type:1,Sort:1,Site:"US"};

it.each(["target","generic"] as const)("reselects %s version assets with their own approval identity and the global CAS head",async(poolKind)=>{
 state=poolKind==="generic"?sheinProjection():{...projection("awaiting_final_approval",templateId),plan:{Source:{...source,EffectiveVersion:"2"},Target:projection().plan.Target}};
 const approvalAction="44444444-4444-4444-8444-444444444444",globalHead={action_id:"55555555-5555-4555-8555-555555555555",payload_hash:digest};
 const pool={approval_action_id:approvalAction,head:globalHead,assets:[{id:"formal-version-two",role:"gallery",url:"https://images.test/formal-version-two.png",presentation:{group:"carousel",order:1}}]},empty={approval_action_id:"",head:globalHead,assets:[]};
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>{
  if(String(url).endsWith("/inventory"))return Promise.resolve(Response.json({target:poolKind==="target"?pool:empty,generic:poolKind==="generic"?pool:null}));
  if(String(url).endsWith("/requirements"))return Promise.resolve(Response.json(sheinRequirements));
  if(String(url).endsWith("/preview")){
   const body=JSON.parse(String(init!.body));
   if(body.choices[0]?.approval_action_id!==approvalAction)return Promise.resolve(Response.json({code:"IMAGE_APPROVAL_CONFLICT"},{status:409}));
   return Promise.resolve(Response.json({digest,head:globalHead,assets:pool.assets}));
  }
  return real(url,init);
 });
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 fireEvent.click(await screen.findByRole("button",{name:"采用素材 1"}));
 if(poolKind==="generic")fireEvent.change(screen.getAllByLabelText("官方图片位置").at(-1)!,{target:{value:"spu:0:0"}});
 fireEvent.click(screen.getByRole("button",{name:"预览完整选择"}));
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/preview"))).toBe(true));
 const preview=fetch.mock.calls.find(([url])=>String(url).endsWith("/preview"))!,command=JSON.parse(String(preview[1]!.body));
 expect(command).toMatchObject({expectedHead:globalHead,choices:[{kind:"approved",approval_action_id:approvalAction,asset_id:"formal-version-two"}]});
 if(poolKind==="generic")expect(command.choices[0].generic_head).toEqual(globalHead);else expect(command.choices[0]).not.toHaveProperty("generic_head");
 fireEvent.click(await screen.findByRole("button",{name:"人工批准并保存素材"}));
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/approve"))).toBe(true));
 const approve=fetch.mock.calls.find(([url])=>String(url).endsWith("/approve"))!;
 expect(JSON.parse(String(approve[1]!.body))).toEqual({...command,selectionDigest:digest});
 expect(fetch.mock.calls.some(([url])=>/\/(prepare|regenerate|confirm)$/.test(String(url)))).toBe(false);
});
const sheinRequirements={platform:"shein",site:"US",categoryId:123,version:"current-official-rules",nativeWidth:1024,nativeHeight:1024,groups:[{group:"spu",skc:0,sku:0,types:[{type:1,minimum:1,maximum:1,nativeCompatible:true},{type:2,minimum:0,maximum:7,nativeCompatible:true}]}]};
function sheinProjection(){const p=projection("awaiting_final_approval",templateId);return {...p,plan:{...p.plan,Source:{...source,EffectiveVersion:"2",ApplyReceiptID:operation},Target:sheinTarget},slots:p.slots.map(slot=>({...slot,recipe:{...slot.recipe,OfficialPlacement:officialPlacement}}))}}
it.each(["source","approved"] as const)("validates final official %s choices before preview",async(kind)=>{
 const p=sheinProjection(),real=fetch.getMockImplementation()!;
 const rules={...sheinRequirements,groups:[{...sheinRequirements.groups[0],types:[...sheinRequirements.groups[0].types,{type:5,minimum:0,maximum:1,nativeCompatible:false}]}]};
 const pool={approval_action_id:templateId,head:{action_id:templateId,payload_hash:sha},assets:[0,1].map(i=>({id:`formal-${i}`,role:"main",url:`https://images.test/formal-${i}.png`,presentation:{group:"carousel",order:i+1},official_placement:{group:"spu",skc:0,sku:0,type:2,sort:1,site:"US"}}))};
 fetch.mockImplementation((url,init)=>{
  const path=new URL(String(url),"http://localhost").pathname;
  if(path.endsWith("/requirements"))return Promise.resolve(Response.json(rules));
  if(path.endsWith(`/runs/${runId}`))return Promise.resolve(Response.json(p));
  if(path.endsWith("/inventory"))return Promise.resolve(Response.json({target:pool,generic:null}));
  if(path.endsWith("/sources"))return real(url,init).then(async response=>Response.json({...await response.json(),originals:[0,1].map(i=>({id:`original-${i}`,displayUrl:`https://images.test/original-${i}.png`,width:1024,height:1024}))}));
  return real(url,init);
 });
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByText(/规则 current-official-rules/);
 for(const i of [1,2])fireEvent.click(screen.getByRole("button",{name:kind==="source"?`选择原图 ${i}`:`采用素材 ${i}`}));
 const selection=within(screen.getByRole("list")),positions=selection.getAllByLabelText("官方图片位置"),types=selection.getAllByLabelText("官方图片类型"),sorts=selection.getAllByLabelText("官方图片排序");
 if(kind==="source")positions.forEach(position=>fireEvent.change(position,{target:{value:"spu:0:0"}}));
 types.forEach(type=>fireEvent.change(type,{target:{value:"2"}}));
 const preview=screen.getByRole("button",{name:"预览完整选择"});
 expect(preview).toBeDisabled();fireEvent.click(preview);expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/preview"))).toBe(false);
 fireEvent.change(sorts[1],{target:{value:"2"}});types.forEach(type=>fireEvent.change(type,{target:{value:"5"}}));
 expect(preview).toBeDisabled();fireEvent.click(preview);expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/preview"))).toBe(false);
 fireEvent.change(types[1],{target:{value:"2"}});expect(preview).toBeEnabled();fireEvent.click(preview);
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/preview"))).toBe(true));
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith("/preview"))!,command=JSON.parse(String(request[1]!.body));
 expect(command.choices.map((choice:{kind:string;official_placement:unknown})=>({kind:choice.kind,position:choice.official_placement}))).toEqual([{kind,position:{group:"spu",skc:0,sku:0,type:5,sort:1,site:"US"}},{kind,position:{group:"spu",skc:0,sku:0,type:2,sort:2,site:"US"}}]);
 expect(command.choices.map((choice:{presentation:{order:number}})=>choice.presentation.order)).toEqual([1,2]);
 expect(fetch.mock.calls.some(([url])=>/\/(prepare|regenerate|confirm|approve)$/.test(String(url)))).toBe(false);
});
it("blocks a platform-specific default until its target matches",async()=>{
 const pinned={...template,targetPlatform:"shein"},real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).includes("/revisions/")?Promise.resolve(Response.json(pinned)):String(url).includes("/templates")?Promise.resolve(Response.json({items:[pinned],nextCursor:""})):String(url).endsWith("/requirements")?Promise.resolve(Response.json(sheinRequirements)):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} target={sheinTarget}/>);
 fireEvent.click(await screen.findByRole("checkbox",{name:"共用原始素材 素材 1"}));
 const prepare=screen.getByRole("button",{name:"准备整套图片计划（2 项）"});
 expect(prepare).toBeDisabled();
 expect(screen.getByRole("option",{name:"Two groups v1"})).toBeDisabled();
 expect(screen.getByText("当前模板不适用于此素材目标，请选择匹配的模板或切换目标。")).toBeInTheDocument();
 fireEvent.click(prepare);expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/prepare"))).toBe(false);
 fireEvent.change(screen.getByLabelText("素材目标"),{target:{value:"shein"}});
 expect(screen.getByRole("option",{name:"Two groups v1"})).toBeEnabled();
 fireEvent.click(screen.getByRole("button",{name:"读取当前图片规则"}));
 await screen.findByText(/规则 current-official-rules/);
 screen.getAllByLabelText("官方图片位置").forEach(position=>fireEvent.change(position,{target:{value:"spu:0:0"}}));
 fireEvent.change(screen.getAllByLabelText("官方图片类型")[1],{target:{value:"2"}});
 fireEvent.change(screen.getAllByLabelText("官方图片排序")[1],{target:{value:"2"}});
 expect(prepare).toBeEnabled();
 fireEvent.change(screen.getByLabelText("素材目标"),{target:{value:"product"}});
 expect(prepare).toBeDisabled();
 fireEvent.click(prepare);expect(fetch.mock.calls.some(([url])=>/\/(prepare|regenerate|confirm)$/.test(String(url)))).toBe(false);
});
it.each(["prepare","regenerate"] as const)("enforces the official per-type maximum before %s",async(action)=>{
 const p=sheinProjection(),real=fetch.getMockImplementation()!;
 const rules={...sheinRequirements,groups:[{...sheinRequirements.groups[0],types:[...sheinRequirements.groups[0].types,{type:5,minimum:0,maximum:1,nativeCompatible:true}]}]};
 fetch.mockImplementation((url,init)=>String(url).endsWith("/requirements")?Promise.resolve(Response.json(rules)):String(url).endsWith(`/runs/${runId}`)?Promise.resolve(Response.json(p)):String(url).endsWith("/regenerate")?Promise.resolve(Response.json(state,{status:201})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} {...action==="regenerate"?{initialRunId:runId}:{target:sheinTarget}}/>);
 if(action==="prepare"){
  fireEvent.change(await screen.findByLabelText("素材目标"),{target:{value:"shein"}});
  fireEvent.click(screen.getByRole("button",{name:"读取当前图片规则"}));
 }
 await screen.findByText(/规则 current-official-rules/);
 let positions:HTMLElement[],sorts:HTMLElement[],types:HTMLElement[];
 if(action==="regenerate"){
  const slots=screen.getAllByRole("checkbox",{name:"重新生成此项，另行确认点数"});slots.forEach(slot=>fireEvent.click(slot));
  positions=slots.map(slot=>within(slot.closest("article")!).getByLabelText("官方图片位置"));
  sorts=slots.map(slot=>within(slot.closest("article")!).getByLabelText("官方图片排序"));
  types=slots.map(slot=>within(slot.closest("article")!).getByLabelText("官方图片类型"));
 }else{
  fireEvent.click(screen.getByRole("checkbox",{name:"共用原始素材 素材 1"}));
  positions=screen.getAllByLabelText("官方图片位置");sorts=screen.getAllByLabelText("官方图片排序");types=screen.getAllByLabelText("官方图片类型");
 }
 positions.forEach(position=>fireEvent.change(position,{target:{value:"spu:0:0"}}));
 types.forEach(type=>fireEvent.change(type,{target:{value:"5"}}));fireEvent.change(sorts[1],{target:{value:"2"}});
 const prepare=screen.getByRole("button",{name:action==="prepare"?"准备整套图片计划（2 项）":"准备所选 2 项的新计划"});
 expect(prepare).toBeDisabled();fireEvent.click(prepare);
 expect(fetch.mock.calls.some(([url])=>/\/(prepare|regenerate|confirm)$/.test(String(url)))).toBe(false);
 fireEvent.change(types[1],{target:{value:"2"}});expect(prepare).toBeEnabled();fireEvent.click(prepare);
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith(`/${action}`))).toBe(true));
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith(`/${action}`))!;
 expect(JSON.parse(String(request[1]!.body)).officialPlacements).toEqual({main:{...officialPlacement,Type:5},detail:{...officialPlacement,Type:2,Sort:2}});
 expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/confirm"))).toBe(false);
});
it.each(["prepare","regenerate"] as const)("requires unique official group sorts before %s",async(action)=>{
 const p=sheinProjection(),real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/requirements")?Promise.resolve(Response.json(sheinRequirements)):String(url).endsWith(`/runs/${runId}`)?Promise.resolve(Response.json(p)):String(url).endsWith("/regenerate")?Promise.resolve(Response.json(state,{status:201})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} {...action==="regenerate"?{initialRunId:runId}:{target:sheinTarget}}/>);
 if(action==="prepare"){
  fireEvent.change(await screen.findByLabelText("素材目标"),{target:{value:"shein"}});
  fireEvent.click(screen.getByRole("button",{name:"读取当前图片规则"}));
 }
 await screen.findByText(/规则 current-official-rules/);
 let positions:HTMLElement[],sorts:HTMLElement[],types:HTMLElement[];
 if(action==="regenerate"){
  const slots=screen.getAllByRole("checkbox",{name:"重新生成此项，另行确认点数"});slots.forEach(slot=>fireEvent.click(slot));
  positions=slots.map(slot=>within(slot.closest("article")!).getByLabelText("官方图片位置"));
  sorts=slots.map(slot=>within(slot.closest("article")!).getByLabelText("官方图片排序"));
  types=slots.map(slot=>within(slot.closest("article")!).getByLabelText("官方图片类型"));
 }else{
  fireEvent.click(screen.getByRole("checkbox",{name:"共用原始素材 素材 1"}));
  positions=screen.getAllByLabelText("官方图片位置");sorts=screen.getAllByLabelText("官方图片排序");
  types=screen.getAllByLabelText("官方图片类型");
 }
 positions.forEach(position=>fireEvent.change(position,{target:{value:"spu:0:0"}}));
 const prepare=screen.getByRole("button",{name:action==="prepare"?"准备整套图片计划（2 项）":"准备所选 2 项的新计划"});
 expect(prepare).toBeDisabled();
 fireEvent.click(prepare);expect(fetch.mock.calls.some(([url])=>/\/(prepare|regenerate|confirm)$/.test(String(url)))).toBe(false);
 expect(screen.getAllByText("请选择当前规则允许的位置和类型，并为同一图片组设置不重复的排序。").length).toBeGreaterThan(0);
 fireEvent.change(sorts[1],{target:{value:"2"}});expect(prepare).toBeDisabled();
 expect(fetch.mock.calls.some(([url])=>/\/(prepare|regenerate|confirm)$/.test(String(url)))).toBe(false);
 fireEvent.change(sorts[1],{target:{value:"1"}});fireEvent.change(types[1],{target:{value:"2"}});expect(prepare).toBeDisabled();
 fireEvent.change(sorts[1],{target:{value:"2"}});expect(prepare).toBeEnabled();fireEvent.click(prepare);
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith(`/${action}`))).toBe(true));
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith(`/${action}`))!;
 expect(JSON.parse(String(request[1]!.body)).officialPlacements).toEqual({main:officialPlacement,detail:{...officialPlacement,Type:2,Sort:2}});
 expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/confirm"))).toBe(false);
});
it("requires current compatible placements before regenerating a historical SHEIN slot",async()=>{
 const p=sheinProjection(),real=fetch.getMockImplementation()!;
 const changed={...sheinRequirements,version:"changed-rules",groups:[{...sheinRequirements.groups[0],types:[{type:2,minimum:1,maximum:8,nativeCompatible:true}]}]};
 fetch.mockImplementation((url,init)=>String(url).endsWith("/requirements")?Promise.resolve(Response.json(changed)):String(url).endsWith(`/runs/${runId}`)?Promise.resolve(Response.json(p)):String(url).endsWith("/regenerate")?Promise.resolve(Response.json(state,{status:201})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByText(/规则 changed-rules/);
 const regenerateSlot=screen.getAllByRole("checkbox",{name:"重新生成此项，另行确认点数"})[0];
 fireEvent.click(regenerateSlot);
 const prepareSubset=screen.getByRole("button",{name:"准备所选 1 项的新计划"});
 expect(prepareSubset).toBeDisabled();
 expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/regenerate"))).toBe(false);
 const slot=within(regenerateSlot.closest("article")!);
 const position=slot.getByLabelText("官方图片位置");
 fireEvent.change(position,{target:{value:"spu:0:0"}});
 expect(slot.getByLabelText("官方图片类型")).toHaveValue("2");
 fireEvent.change(slot.getByLabelText("官方图片排序"),{target:{value:"3"}});
 expect(prepareSubset).toBeEnabled();fireEvent.click(prepareSubset);
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/regenerate"))).toBe(true));
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith("/regenerate"))!;
 expect(JSON.parse(String(request[1]!.body))).toMatchObject({target:sheinTarget,effectiveCatalogVersion:"2",applyReceiptId:operation,selectedTaskIds:["main"],officialPlacements:{main:{...officialPlacement,Type:2,Sort:3}}});
 expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/confirm"))).toBe(false);
});
it("returns from a historical platform run to the current page target for fresh preparation",async()=>{
 const p=sheinProjection(),liveTarget={...sheinTarget,StoreID:"current-store",RecordID:"current-record",CategoryID:456},real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/requirements")?Promise.resolve(Response.json({...sheinRequirements,categoryId:JSON.parse(String(init!.body)).target.CategoryID})):String(url).endsWith(`/runs/${runId}`)?Promise.resolve(Response.json(p)):String(url).endsWith("/images/runs")?Promise.resolve(Response.json({items:[{runId,contextKind:"acquisition",contextId:operation,status:p.status,targetPlatform:"shein",createdAt:now}],nextCursor:""})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} target={liveTarget}/>);
 fireEvent.change(await screen.findByLabelText("本商品最近任务"),{target:{value:runId}});
 await screen.findByText("店铺 saved-store · 站点 US · 类目 123");
 fireEvent.click(screen.getByRole("button",{name:"使用当前页面目标准备新计划"}));
 expect(screen.getByText("店铺 current-store · 站点 US · 类目 456")).toBeInTheDocument();
 expect(screen.queryByRole("button",{name:"准备所选 0 项的新计划"})).not.toBeInTheDocument();
 fireEvent.click(screen.getByRole("button",{name:"读取当前图片规则"}));
 await screen.findByText(/规则 current-official-rules/);
 screen.getAllByLabelText("官方图片位置").forEach(input=>fireEvent.change(input,{target:{value:"spu:0:0"}}));
 fireEvent.change(screen.getAllByLabelText("官方图片类型")[1],{target:{value:"2"}});
 fireEvent.change(screen.getAllByLabelText("官方图片排序")[1],{target:{value:"2"}});
 await prepare();
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/prepare"))).toBe(true));
 const rules=fetch.mock.calls.filter(([url])=>String(url).endsWith("/requirements")).at(-1)!;
 expect(JSON.parse(String(rules[1]!.body))).toEqual({target:liveTarget,effectiveCatalogVersion:"1"});
 const request=fetch.mock.calls.find(([url])=>String(url).endsWith("/prepare"))!;
 expect(JSON.parse(String(request[1]!.body))).toMatchObject({target:liveTarget,effectiveCatalogVersion:"1"});
 expect(JSON.parse(String(request[1]!.body))).not.toHaveProperty("applyReceiptId");
});
for(const restoration of ["initial","recent"]){
it(`restores a SHEIN ${restoration} run's saved target and current rules for original selection and subset regeneration`,async()=>{
 const p=sheinProjection(),real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/requirements")?Promise.resolve(Response.json(sheinRequirements)):String(url).endsWith(`/runs/${runId}`)?Promise.resolve(Response.json(p)):String(url).endsWith("/images/runs")?Promise.resolve(Response.json({items:[{runId,contextKind:"acquisition",contextId:operation,status:p.status,targetPlatform:"shein",createdAt:now}],nextCursor:""})):String(url).endsWith("/regenerate")?Promise.resolve(Response.json(state,{status:201})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} {...restoration==="initial"?{initialRunId:runId}:{target:{...sheinTarget,StoreID:"unrelated-store",RecordID:"unrelated-record",CategoryID:456}}}/>);
 if(restoration==="recent")fireEvent.change(await screen.findByLabelText("本商品最近任务"),{target:{value:runId}});
 await screen.findByText(/规则 current-official-rules/);
 expect(screen.getByLabelText("素材目标")).toHaveValue("shein");
 expect(screen.getByText("店铺 saved-store · 站点 US · 类目 123")).toBeInTheDocument();
 const rules=fetch.mock.calls.find(([url])=>String(url).endsWith("/requirements"))!;
 expect(JSON.parse(String(rules[1]!.body))).toEqual({target:sheinTarget,effectiveCatalogVersion:"2",applyReceiptId:operation});
 fireEvent.click(screen.getByRole("button",{name:"选择原图 1"}));
 const position=screen.getAllByLabelText("官方图片位置").at(-1)!;expect(position).toBeEnabled();
 fireEvent.change(position,{target:{value:"spu:0:0"}});fireEvent.click(screen.getByRole("button",{name:"预览完整选择"}));
 await screen.findByRole("button",{name:"人工批准并保存素材"});
 const preview=fetch.mock.calls.find(([url])=>String(url).endsWith("/preview"))!;
 expect(JSON.parse(String(preview[1]!.body)).choices[0].official_placement).toEqual({group:"spu",skc:0,sku:0,type:1,sort:1,site:"US"});
 fireEvent.click(screen.getByRole("checkbox",{name:"共用原始素材 素材 1"}));
 fireEvent.click(screen.getAllByRole("checkbox",{name:"重新生成此项，另行确认点数"})[0]);
 fireEvent.click(screen.getByRole("button",{name:"准备所选 1 项的新计划"}));
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/regenerate"))).toBe(true));
 const regeneration=fetch.mock.calls.find(([url])=>String(url).endsWith("/regenerate"))!;
 expect(JSON.parse(String(regeneration[1]!.body))).toMatchObject({target:sheinTarget,effectiveCatalogVersion:"2",applyReceiptId:operation,selectedTaskIds:["main"],officialPlacements:{main:officialPlacement}});
});
}
it("does not expose stale rules when switching SHEIN runs and can reload the saved target after a rules failure",async()=>{
 const first=sheinProjection(),secondId="44444444-4444-4444-8444-444444444444",second={...first,runId:secondId,plan:{...first.plan,Target:{...sheinTarget,StoreID:"second-store",RecordID:"second-record",CategoryID:456}}};
 let unavailable=true;const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith("/requirements")?Promise.resolve(JSON.parse(String(init!.body)).target.RecordID==="second-record"&&unavailable?Response.json({code:"IMAGE_UNAVAILABLE"},{status:503}):Response.json({...sheinRequirements,categoryId:JSON.parse(String(init!.body)).target.CategoryID})):String(url).endsWith(`/runs/${runId}`)?Promise.resolve(Response.json(first)):String(url).endsWith(`/runs/${secondId}`)?Promise.resolve(Response.json(second)):String(url).endsWith("/images/runs")?Promise.resolve(Response.json({items:[first,second].map(p=>({runId:p.runId,contextKind:"acquisition",contextId:operation,status:p.status,targetPlatform:"shein",createdAt:now})),nextCursor:""})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByText(/规则 current-official-rules/);
 fireEvent.change(screen.getByLabelText("本商品最近任务"),{target:{value:secondId}});await screen.findByRole("alert");
 expect(screen.queryByText(/规则 current-official-rules/)).not.toBeInTheDocument();
 fireEvent.click(screen.getByRole("button",{name:"选择原图 1"}));expect(screen.getAllByLabelText("官方图片位置").at(-1)).toBeDisabled();
 unavailable=false;fireEvent.click(screen.getByRole("button",{name:"读取当前图片规则"}));
 await waitFor(()=>expect(screen.getAllByLabelText("官方图片位置").at(-1)).toBeEnabled());
 const last=fetch.mock.calls.filter(([url])=>String(url).endsWith("/requirements")).at(-1)!;
 expect(JSON.parse(String(last[1]!.body))).toEqual({target:second.plan.Target,effectiveCatalogVersion:"2",applyReceiptId:operation});
});
it("discards late SHEIN rules from a previously refreshed run",async()=>{
 const first=sheinProjection(),secondId="44444444-4444-4444-8444-444444444444",second={...first,runId:secondId,plan:{...first.plan,Target:{...sheinTarget,StoreID:"second-store",RecordID:"second-record",CategoryID:456}}};
 let resolve!:(response:Response)=>void,firstReads=0;const late=new Promise<Response>(ok=>{resolve=ok}),real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>{
  if(String(url).endsWith("/requirements")){const target=JSON.parse(String(init!.body)).target;if(target.RecordID==="saved-record"&&++firstReads===2)return late;return Promise.resolve(Response.json({...sheinRequirements,version:target.RecordID,categoryId:target.CategoryID}))}
  if(String(url).endsWith(`/runs/${runId}`))return Promise.resolve(Response.json(first));
  if(String(url).endsWith(`/runs/${secondId}`))return Promise.resolve(Response.json(second));
  if(String(url).endsWith("/images/runs"))return Promise.resolve(Response.json({items:[first,second].map(p=>({runId:p.runId,contextKind:"acquisition",contextId:operation,status:p.status,targetPlatform:"shein",createdAt:now})),nextCursor:""}));
  return real(url,init);
 });
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} initialRunId={runId}/>);
 await screen.findByText(/规则 saved-record/);fireEvent.click(screen.getByRole("button",{name:"刷新原任务"}));
 await waitFor(()=>expect(firstReads).toBe(2));fireEvent.change(screen.getByLabelText("本商品最近任务"),{target:{value:secondId}});
 await screen.findByText(/规则 second-record/);await act(async()=>resolve(Response.json({...sheinRequirements,version:"late-first-record"})));
 expect(screen.queryByText(/规则 late-first-record/)).not.toBeInTheDocument();expect(screen.getByText("店铺 second-store · 站点 US · 类目 456")).toBeInTheDocument();
});
it("keeps an UNKNOWN parent's successful-looking candidates unavailable for reuse",async()=>{
 const parentId="44444444-4444-4444-8444-444444444444";
 const parent={...projection("blocked",templateId),runId:parentId,candidateSelectionAvailable:false,resultDigest:"",recoverableEffects:[{SlotID:"detail",Attempt:1,Code:"provider_outcome_unknown"}]};
 const child={...projection("awaiting_final_approval",operation),plan:{...state.plan,Regeneration:{RunID:parentId}}};state=child;
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith(`/runs/${parentId}`)?Promise.resolve(Response.json(parent)):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 const choices=await screen.findAllByRole("button",{name:"采用原任务结果"});
 choices.forEach(button=>expect(button).toBeDisabled());
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

for (const action of ["prepare","regenerate"]) {
it(`can replay an unresolved ${action} with its original key and frozen body after not-found`,async()=>{
 const requestKey=templateId,body={expectedModeRevision:"0",target:{Platform:"product"},sharedOriginalIds:["original"],selectedTaskIds:["main","detail"],template:{templateId,revision:"1"}};
 const intent={action,...action==="regenerate"?{runId:operation}:{},requestKey,body};
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:intent`,JSON.stringify(intent));
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).includes("/by-key/")?Promise.resolve(Response.json({code:"IMAGE_NOT_FOUND"},{status:404})):String(url).endsWith(`/${action}`)?Promise.resolve(Response.json(state,{status:201})):real(url,init));
 const view=render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));await screen.findByRole("alert");
 expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).not.toBeNull();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
 view.unmount();render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"继续原操作"}));
 await screen.findByRole("button",{name:"确认点数并生成"});
 const requests=fetch.mock.calls.filter(([url])=>String(url).endsWith(`/${action}`));expect(requests).toHaveLength(1);
 expect(new Headers(requests[0][1]!.headers).get("Idempotency-Key")).toBe(requestKey);
 expect(JSON.parse(String(requests[0][1]!.body))).toEqual(body);
 expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/confirm"))).toBe(false);
 expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).toBeNull();
});
}

it("clears lost recovery intent only when its exact original effect has a known closed result",async()=>{
 state=projection("awaiting_final_approval",templateId);state.slots[0].closure={Kind:"settled",Points:10};
 const body={actionId:operation,planRevision:1,slotId:"main",attempt:1};
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:intent`,JSON.stringify({action:"recover",runId,body}));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).toBeNull());
 expect(screen.getAllByRole("button",{name:"选择采用"})[0]).toBeEnabled();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

for(const mismatch of ["unknown","attempt","revision","closure"]){
it(`keeps a recovery intent with ${mismatch} unresolved and replays its exact action`,async()=>{
 const body={actionId:operation,planRevision:1,slotId:"main",attempt:1};
 state=projection("blocked",templateId);state.slots[0].closure={Kind:"settled",Points:10};
 const raw={...state,recoverableEffects:mismatch==="unknown"?[{SlotID:"main",Attempt:1,Code:"provider_outcome_unknown"}]:null};
 if(mismatch==="attempt")raw.slots[0].attempt=2;
 if(mismatch==="revision")raw.planRevision=2;
 if(mismatch==="closure")raw.slots[0].closure=null;
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:intent`,JSON.stringify({action:"recover",runId,body}));
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith(`/runs/${runId}`)?Promise.resolve(Response.json(raw)):String(url).endsWith("/recover")?Promise.resolve(Response.json({code:"IMAGE_BLOCKED"},{status:409})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"核实原请求"})).toBeEnabled());
 expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).not.toBeNull();
 fireEvent.click(screen.getByRole("button",{name:"继续原操作"}));await screen.findByRole("alert");
 const requests=fetch.mock.calls.filter(([url])=>String(url).endsWith("/recover"));expect(requests).toHaveLength(1);
 expect(JSON.parse(String(requests[0][1]!.body))).toEqual(body);
 expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).not.toBeNull();
});
}

for(const receiptFound of [true,false]){
it(`notifies a saved SHEIN result and verifies its original resume receipt despite unavailable current rules (receipt=${receiptFound})`,async()=>{
 const p={...sheinProjection(),status:"completed",approvalAvailable:false,regenerationAvailable:false},saved=vi.fn(),real=fetch.getMockImplementation()!;
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:intent`,JSON.stringify({action:"resume",runId,body:{actionId:operation}}));
 fetch.mockImplementation((url,init)=>String(url).endsWith("/requirements")?Promise.resolve(Response.json({code:"IMAGE_UNAVAILABLE"},{status:503})):String(url).endsWith(`/runs/${runId}`)?Promise.resolve(Response.json(p)):String(url).endsWith(`/approvals/${operation}`)?Promise.resolve(receiptFound?Response.json({actionId:operation,selectionDigest:digest,assets:[{id:"image",role:"main",url:"https://images.test/result.png"}]}):Response.json({code:"IMAGE_NOT_FOUND"},{status:404})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation} onSaved={saved}/>);
 await waitFor(()=>expect(saved).toHaveBeenCalledOnce());fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith(`/approvals/${operation}`))).toBe(true));
 if(receiptFound)await waitFor(()=>expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).toBeNull());
 else expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).not.toBeNull();
 expect(saved).toHaveBeenCalledOnce();expect(screen.queryByText(/规则 current-official-rules/)).not.toBeInTheDocument();
 expect(fetch.mock.calls.some(([url])=>/\/(prepare|regenerate|confirm|resume|approve)$/.test(String(url)))).toBe(false);
});
}
for (const receiptFound of [true,false]) {
it(`verifies a lost resume response using its original immutable approval receipt (found=${receiptFound})`,async()=>{
 state=projection("completed",templateId);
 const body={actionId:operation};
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:intent`,JSON.stringify({action:"resume",runId,body}));
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith(`/approvals/${operation}`)?Promise.resolve(receiptFound?Response.json({actionId:operation,selectionDigest:digest,assets:[{id:"image",role:"main",url:"https://images.test/result.png"}]}):Response.json({code:"IMAGE_NOT_FOUND"},{status:404})):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 if(receiptFound)await waitFor(()=>expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).toBeNull());
 else {await screen.findByRole("alert");expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).not.toBeNull()}
 expect(fetch.mock.calls.some(([url])=>String(url).endsWith(`/approvals/${operation}`))).toBe(true);
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
}

it("clears a lost original confirmation after exact admitted terminal evidence",async()=>{
 state=projection("completed",operation);
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:intent`,JSON.stringify({action:"confirm",runId,body:{actionId:operation,planRevision:1,planDigest:sha,quoteDigest:digest}}));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).toBeNull());
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it("unlocks original effect recovery once the exact confirmation produced durable blocked effects",async()=>{
 const blocked={...projection("blocked",operation),recoverableEffects:[{SlotID:"main",Attempt:1,Code:"provider_outcome_unknown"}]};
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:intent`,JSON.stringify({action:"confirm",runId,body:{actionId:operation,planRevision:1,planDigest:sha,quoteDigest:digest}}));
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>String(url).endsWith(`/runs/${runId}`)?Promise.resolve(Response.json(blocked)):real(url,init));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(localStorage.getItem(`product-image-set:actor:org:acquisition:${operation}:intent`)).toBeNull());
 expect(screen.getByRole("button",{name:"核实原调用"})).toBeEnabled();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it("restores an unfinished failed run through its original workflow without a new preparation or confirmation",async()=>{
 state=projection("failed",operation);state.slots[1]={...state.slots[1],status:"pending",attempt:0,candidates:[],closure:null};
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 const real=fetch.getMockImplementation()!;
 fetch.mockImplementation(async(url,init)=>{
  if(String(url).endsWith("/restart")){state=projection("awaiting_final_approval",operation);return Response.json({runId,status:"accepted"},{status:202})}
  return real(url,init);
 });
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"恢复原失败任务"}));
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/restart"))).toBe(true));
 await screen.findByText("等待人工选择");
 const requests=fetch.mock.calls.filter(([,init])=>init?.method==="POST");
 expect(requests).toHaveLength(1);expect(String(requests[0][0])).toContain(`/runs/${runId}/restart`);
 expect(JSON.parse(String(requests[0][1]!.body))).toEqual({planRevision:1,planDigest:sha,quoteDigest:digest});
});

it("keeps a lost restart acknowledgement across reload and explicitly replays only its original plan",async()=>{
 state=projection("failed",operation);
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 const real=fetch.getMockImplementation()!;let attempts=0;
 fetch.mockImplementation(async(url,init)=>{
  if(String(url).endsWith("/restart")){
   if(++attempts===1)throw new Error("lost acknowledgement");
   state=projection("blocked",operation);return Response.json({runId,status:"accepted"},{status:202});
  }
  return real(url,init);
 });
 const view=render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"恢复原失败任务"}));
 await screen.findByRole("button",{name:"核实原请求"});
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`,frozen=localStorage.getItem(key);
 expect(JSON.parse(frozen!)).toEqual({action:"restart",runId,body:{planRevision:1,planDigest:sha,quoteDigest:digest}});
 view.unmount();render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await screen.findByText("原操作尚未得到明确回执，请继续核实同一编号。");
 expect(attempts).toBe(1);expect(localStorage.getItem(key)).toBe(frozen);
 fireEvent.click(screen.getByRole("button",{name:"继续原操作"}));
 await waitFor(()=>expect(localStorage.getItem(key)).toBeNull());
 const posts=fetch.mock.calls.filter(([,init])=>init?.method==="POST");
 expect(posts).toHaveLength(2);expect(posts[1][0]).toBe(posts[0][0]);expect(posts[1][1]!.body).toBe(posts[0][1]!.body);
});

it("requires the original plan before read-only progress can verify a failed-run restart",async()=>{
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`;
 localStorage.setItem(key,JSON.stringify({action:"restart",runId,body:{planRevision:1,planDigest:sha,quoteDigest:digest}}));
 state={...projection("executing",operation),planDigest:"c".repeat(64)};
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await screen.findByText("原操作尚未得到明确回执，请继续核实同一编号。");
 expect(localStorage.getItem(key)).not.toBeNull();
 state=projection("executing",operation);
 fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(localStorage.getItem(key)).toBeNull());
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

it("keeps a known closed failed run on the result-review and regeneration path",async()=>{
 state={...projection("failed",operation),regenerationAvailable:true};
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 await screen.findByRole("button",{name:"准备所选 0 项的新计划"});
 expect(screen.queryByRole("button",{name:"恢复原失败任务"})).not.toBeInTheDocument();
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});

for(const status of ["cancelled","completed","failed"]){
it(`releases a lost cancel intent when the original plan is already ${status}`,async()=>{
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`;
 state=projection(status,operation);
 if(status==="failed")state.slots[1]={...state.slots[1],status:"pending",attempt:0,candidates:[],closure:null};
 localStorage.setItem(key,JSON.stringify({action:"cancel",runId,body:{actionId:operation,planRevision:1}}));
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(localStorage.getItem(key)).toBeNull());
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
 if(status==="failed")expect(screen.getByRole("button",{name:"恢复原失败任务"})).toBeEnabled();
});
}

for(const mismatch of ["executing","blocked","revision"]){
it(`retains an unresolved cancel intent for ${mismatch} without another mutation`,async()=>{
 const key=`product-image-set:actor:org:acquisition:${operation}:intent`;
 state=projection(mismatch==="revision"?"completed":mismatch,operation);
 if(mismatch==="revision")state.planRevision=2;
 const frozen=JSON.stringify({action:"cancel",runId,body:{actionId:operation,planRevision:1}});
 localStorage.setItem(key,frozen);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await screen.findByText("原操作尚未得到明确回执，请继续核实同一编号。");
 expect(localStorage.getItem(key)).toBe(frozen);
 expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(false);
});
}


it("restores a paid original through the existing recovery command and keeps unknown siblings",async()=>{
 state=projection("blocked","44444444-4444-4444-8444-444444444444");
 Object.assign(state,{resultDigest:"",approvalAvailable:false,candidateSelectionAvailable:false,regenerationAvailable:false,settledPoints:10,outputRecoverySummary:{settledPoints:10,slots:[{slotId:"main",attempt:1,points:10,unavailable:false}]},recoverableEffects:[{SlotID:"main",Attempt:1,Code:"slot_provider_outcome_unknown"},{SlotID:"detail",Attempt:1,Code:"slot_provider_outcome_unknown"}],block:{Code:"slot_provider_outcome_unknown"}});
 for(const slot of state.slots)Object.assign(slot,{status:"blocked",candidates:[],closure:null,errorCode:"slot_provider_outcome_unknown"});
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 expect(await screen.findByRole("heading",{name:"已生成，结果待恢复"})).toBeVisible();
 expect(screen.getByText(/已结算用量 10 点/)).toBeVisible();
 expect(screen.getByText(/detail：原调用待核实，点数预留保留/)).toBeVisible();
 fireEvent.click(screen.getByRole("button",{name:"恢复原结果"}));
 await waitFor(()=>expect(fetch.mock.calls.some(([u,init])=>String(u).endsWith("/recover")&&init?.method==="POST")).toBe(true));
 const [,request]=fetch.mock.calls.find(([u,init])=>String(u).endsWith("/recover")&&init?.method==="POST")!;
 expect(JSON.parse(request!.body as string)).toMatchObject({planRevision:1,slotId:"main",attempt:1});
 expect(fetch.mock.calls.some(([u])=>/\/(prepare|confirm|regenerate|approve)$/.test(String(u)))).toBe(false);
});

it("does not offer recovery of a proved unavailable paid original",async()=>{
 state=projection("blocked","44444444-4444-4444-8444-444444444444");
 Object.assign(state,{resultDigest:"",approvalAvailable:false,candidateSelectionAvailable:false,regenerationAvailable:false,settledPoints:20,outputRecoverySummary:{settledPoints:20,slots:[{slotId:"main",attempt:1,points:10,unavailable:true},{slotId:"detail",attempt:1,points:10,unavailable:true}]},recoverableEffects:[{SlotID:"main",Attempt:1,Code:"slot_provider_outcome_unknown"},{SlotID:"detail",Attempt:1,Code:"slot_provider_outcome_unknown"}]});
 for(const slot of state.slots)Object.assign(slot,{status:"blocked",candidates:[],closure:null,errorCode:"slot_provider_outcome_unknown"});
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 expect(await screen.findByRole("heading",{name:"已扣点，原结果不可用"})).toBeVisible();
 expect(screen.queryByRole("button",{name:"恢复原结果"})).toBeNull();
});


it("restores a paid output whose original recoverable binding exists only in Run.Block",async()=>{
 state=projection("blocked","44444444-4444-4444-8444-444444444444");
 Object.assign(state,{approvalAvailable:false,candidateSelectionAvailable:false,regenerationAvailable:false,outputRecoverySummary:{settledPoints:10,slots:[{slotId:"main",attempt:1,points:10,unavailable:false}]},recoverableEffects:null,block:{Code:"slot_provider_outcome_unknown",SlotID:"main"}});
 Object.assign(state.slots[0]!,{status:"blocked",candidates:[],closure:null,errorCode:"slot_provider_outcome_unknown"});
 localStorage.setItem(`product-image-set:actor:org:acquisition:${operation}:run`,runId);
 render(<ProductImageSetPanel kind="acquisition" contextId={operation}/>);
 expect(await screen.findByRole("heading",{name:"已生成，结果待恢复"})).toBeVisible();
 expect(screen.getByRole("button",{name:"恢复原结果"})).toBeEnabled();
});
