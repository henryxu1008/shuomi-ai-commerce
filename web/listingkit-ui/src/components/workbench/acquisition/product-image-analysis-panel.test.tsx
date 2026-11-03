import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {ProductImageAnalysisPanel} from "./product-image-analysis-panel";
const context=vi.hoisted(()=>({user:{id:"actor"},effectiveOrganization:{id:"B"},permissions:["workbench.agent.use","local_agent.write"]}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>context}));
vi.mock("../agents/agent-configuration-hooks",()=>({useConfigurationRead:()=>({data:{mode:"review",revision:"0"}})}));
const id="11111111-1111-4111-8111-111111111111",key="22222222-2222-4222-8222-222222222222",hash="a".repeat(64),base=`/api/workbench/sourcing/1688/acquisitions/${id}/images/analysis`;
let fetch:ReturnType<typeof vi.fn>,current:Record<string,unknown>,uncertain:boolean;
beforeEach(()=>{localStorage.clear();uncertain=false;context.permissions=["workbench.agent.use","local_agent.write"];current={contextKind:"acquisition",contextId:id,requestKey:key,catalogVersion:"1",publicationId:"pub",unitVersion:"ai-micropoint-v1",status:"prepared",stopReason:"",runId:"",kind:"source",imageId:"img",modelId:"enterprise-mini",preparationId:id,configurationDigest:hash,quoteDigest:hash,selectionDigest:hash,contentSha256:hash,maximumMicropoints:"1000001",actualMicropoints:"",width:2,height:2};
 fetch=vi.fn(async(input,init)=>{const url=String(input);if(url===base)return Response.json({contextKind:"acquisition",contextId:id,productKey:"p",catalogVersion:"1",publicationId:"pub",ready:true,reason:"",inventoryStatus:"NOT_READY",inventoryDigest:"",choices:[{kind:"source",id:"img",url:"https://images.test/1.png"}]});if(url.endsWith("/runs"))return Response.json({contextKind:"acquisition",contextId:id,items:[],nextCursor:""});if(url.endsWith("/prepare")){current={...current,requestKey:new Headers(init.headers).get("Idempotency-Key")};return Response.json(current)}if(url.endsWith("/confirm")){if(uncertain)throw Error("lost reply");current={...current,status:"completed",runId:id,actualMicropoints:"31",opinion:{observations:[],suggestions:[],unknowns:["仅离线示例"]}};return Response.json(current)}if(url.includes("/by-key/"))return Response.json(current);throw Error(url)});vi.stubGlobal("fetch",fetch);
});
afterEach(()=>{cleanup();vi.unstubAllGlobals();localStorage.clear()});
it("requires explicit image-and-price consent then renders readonly advice with no second model selection",async()=>{
 render(<ProductImageAnalysisPanel kind="acquisition" contextId={id}/>);
 fireEvent.change(await screen.findByLabelText("待检查图片"),{target:{value:"source:img"}});
 fireEvent.click(screen.getByRole("button",{name:"准备单图检查"}));
 expect(await screen.findByText(/1.000001 点/)).toBeVisible();const confirm=screen.getByRole("button",{name:"确认发送图片并检查"});expect(confirm).toBeDisabled();
 expect(fetch.mock.calls.filter(([,i])=>i.method==="POST")).toHaveLength(1);
 fireEvent.click(screen.getByRole("checkbox"));fireEvent.click(confirm);
 expect(await screen.findByText("仅离线示例")).toBeVisible();expect(screen.getByText(/0.000031 点/)).toBeVisible();
 expect(screen.queryByRole("button",{name:/采用|保存结果|重试生成/})).toBeNull();
 expect(fetch.mock.calls.filter(([u])=>String(u).endsWith("/confirm"))).toHaveLength(1);
});
it("retains an uncertain original request across reload without automatically posting again",async()=>{
 uncertain=true;const view=render(<ProductImageAnalysisPanel kind="acquisition" contextId={id}/>);
 fireEvent.change(await screen.findByLabelText("待检查图片"),{target:{value:"source:img"}});fireEvent.click(screen.getByRole("button",{name:"准备单图检查"}));await screen.findByRole("checkbox");fireEvent.click(screen.getByRole("checkbox"));fireEvent.click(screen.getByRole("button",{name:"确认发送图片并检查"}));
 await screen.findByText(/结果尚未确认/);expect(screen.getByRole("button",{name:"准备单图检查"})).toBeDisabled();const posts=fetch.mock.calls.filter(([,i])=>i.method==="POST").length;
 view.unmount();render(<ProductImageAnalysisPanel kind="acquisition" contextId={id}/>);fireEvent.click(await screen.findByRole("button",{name:"核实原请求"}));
 await waitFor(()=>expect(fetch.mock.calls.filter(([u])=>String(u).includes("/by-key/")).length).toBeGreaterThan(0));expect(fetch.mock.calls.filter(([,i])=>i.method==="POST")).toHaveLength(posts);
});

it.each([[503,"PRODUCT_AGENT_UNAVAILABLE"],[503,"DEPENDENCY_UNAVAILABLE"],[409,"AGENT_CONFLICT"]])("retains the original key after confirm readback fails with %s %s",async(status,code)=>{
 let rejectRead=true;const original=fetch.getMockImplementation()! as (input:unknown,init:RequestInit)=>Promise<Response>;fetch.mockImplementation(async(input,init)=>{if(String(input).endsWith("/confirm")){current={...current,status:"completed",runId:id,actualMicropoints:"31",opinion:{observations:[],suggestions:[],unknowns:["原结果已保存"]}};return Response.json({code},{status})}if(String(input).includes("/by-key/")&&init.method==="GET"&&rejectRead){rejectRead=false;return Response.json({code:"FORBIDDEN"},{status:403})}return original(input,init)});
 render(<ProductImageAnalysisPanel kind="acquisition" contextId={id}/>);fireEvent.change(await screen.findByLabelText("待检查图片"),{target:{value:"source:img"}});fireEvent.click(screen.getByRole("button",{name:"准备单图检查"}));await screen.findByRole("checkbox");fireEvent.click(screen.getByRole("checkbox"));fireEvent.click(screen.getByRole("button",{name:"确认发送图片并检查"}));
 await screen.findByRole("alert");expect(screen.getByRole("button",{name:"准备单图检查"})).toBeDisabled();expect(screen.getByRole("button",{name:"确认发送图片并检查"})).toBeDisabled();
 const storage=localStorage.getItem(`product-image-analysis:actor:B:acquisition:${id}`)!;expect(JSON.parse(storage).uncertain).toBe(true);
 fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));await screen.findByText("当前身份无权执行此操作。");expect(screen.getByRole("button",{name:"准备单图检查"})).toBeDisabled();expect(JSON.parse(localStorage.getItem(`product-image-analysis:actor:B:acquisition:${id}`)!).uncertain).toBe(true);
 fireEvent.click(screen.getByRole("button",{name:"核实原请求"}));expect(await screen.findByText("原结果已保存")).toBeVisible();expect(fetch.mock.calls.filter(([,i])=>i.method==="POST")).toHaveLength(2);
});
it("allows a new preparation after a proven initial rejection, never after a rejected history read",async()=>{
 const original=fetch.getMockImplementation()! as (input:unknown,init:RequestInit)=>Promise<Response>;fetch.mockImplementation(async(input,init)=>String(input).endsWith("/prepare")?Response.json({code:"CONFIGURATION_CHANGED"},{status:409}):original(input,init));
 render(<ProductImageAnalysisPanel kind="acquisition" contextId={id}/>);fireEvent.change(await screen.findByLabelText("待检查图片"),{target:{value:"source:img"}});fireEvent.click(screen.getByRole("button",{name:"准备单图检查"}));await screen.findByRole("alert");expect(screen.getByRole("button",{name:"准备单图检查"})).not.toBeDisabled();
});
