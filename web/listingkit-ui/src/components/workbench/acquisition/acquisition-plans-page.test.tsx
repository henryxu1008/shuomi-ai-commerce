import {act,cleanup,render,screen,waitFor,within} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,it,expect,vi} from "vitest";
import {AcquisitionPlansPage} from "./acquisition-plans-page";
import {AcquisitionPage} from "./acquisition-page";
import {AcquisitionAPIError} from "@/lib/api/product-acquisition";
const calls=vi.hoisted(()=>({list:vi.fn(),create:vi.fn(),advance:vi.fn(),read:vi.fn(),key:vi.fn(),rules:vi.fn(),mutate:vi.fn()}));
vi.mock("@/lib/api/acquisition-plan",()=>({listAcquisitionPlans:calls.list,listAcquisitionRules:calls.rules,createAcquisitionPlan:calls.create,advanceAcquisitionPlan:calls.advance,readAcquisitionPlan:calls.read,readAcquisitionPlanByKey:calls.key,mutateAcquisitionRule:calls.mutate}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({
 user:{id:"actor"},
 effectiveOrganization:{id:"org-A"},
 acquisitionIntentReady:true,
 registerOrganizationSwitchGuard:()=>()=>{},
})}));
vi.mock("next/navigation",()=>({useSearchParams:()=>new URLSearchParams()}));
beforeEach(()=>{sessionStorage.clear();vi.clearAllMocks();calls.rules.mockResolvedValue({rules:[]});calls.list.mockResolvedValue({plans:[],canAutoTransfer:true});HTMLDialogElement.prototype.showModal=function(){this.setAttribute("open","")};HTMLDialogElement.prototype.close=function(){this.removeAttribute("open")};});
afterEach(()=>vi.restoreAllMocks());
afterEach(cleanup);
const scope={userId:"actor",organizationId:"org-A"};
async function start(){await userEvent.click(await screen.findByRole("button",{name:"1688在线采集"}));await userEvent.type(screen.getByLabelText("1688 商品链接"),"123");await userEvent.click(screen.getByRole("button",{name:"开始采集"}));}

it("shows the 1688 plan list before the user opens the capture form", async () => {
 render(<AcquisitionPage plansEnabled/>);
 await screen.findByText("还没有采集计划。");
 expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

 const trigger=screen.getByRole("button",{name:"1688在线采集"});
 await userEvent.click(trigger);
 expect(screen.getByRole("dialog",{name:"1688在线采集"})).toBeVisible();
 await userEvent.click(screen.getByRole("button",{name:"取消"}));
 expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
 expect(trigger).toBeEnabled();
 expect(calls.create).not.toHaveBeenCalled();
});
it("opens plan results for review without advancing and filters failed items", async () => {
 const id="11111111-1111-4111-8111-111111111111";
 const operationId="22222222-2222-4222-8222-222222222222";
 const source=(offer:number)=>`https://detail.1688.com/offer/${offer}.html`;
 const plan={id,createdAt:"2026-10-11T00:00:00Z",input:{mode:"product",inputs:[source(123)]},discoveryIndex:1,state:"paused",revision:2,
  items:[{source:source(123),state:"published",operationId,evaluation:{state:"not_matched",startingPrice:"12.50",minimumQuantity:"4"}},
   {source:source(456),state:"failed",failureCode:"source_unavailable"},{source:source(789),state:"needs_verification"},{source:source(987),state:"pending"}],
  discoveryFailures:[],preparationIds:[],truncated:false};
 calls.list.mockResolvedValue({plans:[plan],canAutoTransfer:true});
 calls.read.mockResolvedValue(plan);
 render(<AcquisitionPlansPage scope={scope}/>);
 const trigger=await screen.findByRole("button",{name:"查看结果"});
 await userEvent.click(trigger);
 const results=await screen.findByRole("region",{name:"采集结果"});
 expect(within(results).getByRole("heading",{name:"采集结果"})).toHaveFocus();
 expect(within(results).getByText("¥12.50")).toBeVisible();
 expect(within(results).getByText("未通过规则，保留在我的数据")).toBeVisible();
 expect(within(results).getByRole("link",{name:"查看商品"})).toHaveAttribute("href",`/workbench/supply/acquisition/operation/${operationId}`);
 await userEvent.selectOptions(within(results).getByRole("combobox",{name:"按保存状态筛选"}),"failed");
 expect(within(results).getByText("来源不可用或需要登录、验证。")).toBeVisible();
 expect(within(results).queryByRole("link",{name:"查看商品"})).not.toBeInTheDocument();
 await userEvent.click(within(results).getByRole("button",{name:"返回采集计划"}));
 expect(screen.queryByRole("region",{name:"采集结果"})).not.toBeInTheDocument();
 expect(trigger).toHaveFocus();
 expect(calls.create).not.toHaveBeenCalled();
 expect(calls.advance).not.toHaveBeenCalled();
});

it("keeps the selected content operation in published-plan and operation-recovery links", async () => {
 const operationId="22222222-2222-4222-8222-222222222222";
 const plan={id:"11111111-1111-4111-8111-111111111111",createdAt:"2026-10-11T00:00:00Z",input:{mode:"product",inputs:["https://detail.1688.com/offer/123.html"]},discoveryIndex:1,state:"completed",revision:1,items:[{source:"https://detail.1688.com/offer/123.html",state:"published",operationId}],discoveryFailures:[],preparationIds:[],truncated:false};
 calls.create.mockResolvedValue(plan);
 render(<AcquisitionPlansPage scope={scope} agentKind="content"/>);
 await start();
 expect(await screen.findByRole("link",{name:"查看商品"})).toHaveAttribute("href",`/workbench/supply/acquisition/operation/${operationId}?agent_kind=content`);
 await userEvent.type(screen.getByLabelText("操作 ID"),operationId);
 expect(screen.getByRole("link",{name:"读取"})).toHaveAttribute("href",`/workbench/supply/acquisition/operation/${operationId}?agent_kind=content`);
});
it("shows a definite source conflict and stops offering to continue its frozen transfer",async()=>{
 const plan={id:"11111111-1111-4111-8111-111111111111",createdAt:"2026-10-11T00:00:00Z",input:{mode:"product",inputs:["https://detail.1688.com/offer/123.html"]},discoveryIndex:1,state:"pending",revision:1,items:[],discoveryFailures:[],preparationIds:[],truncated:false};
 calls.create.mockResolvedValue(plan);
 calls.advance.mockResolvedValue({...plan,state:"transfer_pending",revision:2,failureCode:"transfer_source_conflict"});
 render(<AcquisitionPlansPage scope={scope}/>);await start();
 expect(await screen.findByText(/来源批次已移动、归档或版本变化/)).toBeInTheDocument();
 expect(screen.getByRole("button",{name:"继续原计划"})).toBeDisabled();
 expect(calls.advance).toHaveBeenCalledOnce();
});
it("retains a creation key after unknown outcome for explicit recovery",async()=>{
 calls.create.mockRejectedValue(new AcquisitionAPIError("OUTCOME_UNKNOWN",503));
 render(<AcquisitionPlansPage scope={scope}/>);await start();
 await waitFor(()=>expect(calls.create).toHaveBeenCalledOnce());
 const key=calls.create.mock.calls[0][1];
 expect(screen.getByRole("button",{name:"读取原计划"})).toBeInTheDocument();
 calls.key.mockRejectedValue(new AcquisitionAPIError("ACQUISITION_NOT_FOUND",404));
 await userEvent.click(screen.getByRole("button",{name:"读取原计划"}));
 await waitFor(()=>expect(calls.key).toHaveBeenCalledWith(scope,key,expect.any(AbortSignal)));
 expect(calls.create).toHaveBeenCalledTimes(1);expect(calls.advance).not.toHaveBeenCalled();
});
it("aborts on unmount and never advances a late created plan",async()=>{
 let resolve!:(v:unknown)=>void;calls.create.mockReturnValue(new Promise(ok=>{resolve=ok}));
 const view=render(<AcquisitionPlansPage scope={scope}/>);await start();
 await waitFor(()=>expect(calls.create).toHaveBeenCalledOnce());
 const signal=calls.create.mock.calls[0][3] as AbortSignal;
 view.unmount();expect(signal.aborted).toBe(true);
 await act(async()=>resolve({id:"11111111-1111-4111-8111-111111111111",state:"pending",revision:1}));
 expect(calls.advance).not.toHaveBeenCalled();
});

it("does not dispatch a new plan when original intent storage fails",async()=>{
 vi.spyOn(Storage.prototype,"setItem").mockImplementation(()=>{throw new Error("blocked")});
 render(<AcquisitionPlansPage scope={scope}/>);await start();
 expect(calls.create).not.toHaveBeenCalled();
 expect(screen.getByRole("alert")).toHaveTextContent("未提交");
 expect(screen.getByRole("button",{name:"开始采集"})).toBeEnabled();
 expect(screen.queryByRole("button",{name:"读取原计划"})).not.toBeInTheDocument();
});

it("preserves the original plan when replay storage fails and still allows its read",async()=>{
 const key="22222222-2222-4222-8222-222222222222",slot=`listingkit:acquisition:create:${scope.userId}:${scope.organizationId}`;
 const raw=JSON.stringify({key,input:{mode:"product",inputs:["123"],autoTransfer:false}});
 sessionStorage.setItem(slot,raw);
 render(<AcquisitionPlansPage scope={scope}/>);
 await screen.findByText("还没有采集计划。");
 vi.spyOn(Storage.prototype,"setItem").mockImplementation(()=>{throw new Error("blocked")});
 await userEvent.click(screen.getByRole("button",{name:"使用原请求确认创建"}));
 expect(calls.create).not.toHaveBeenCalled();
 expect(sessionStorage.getItem(slot)).toBe(raw);
 calls.key.mockRejectedValue(new AcquisitionAPIError("ACQUISITION_NOT_FOUND",404));
 await userEvent.click(screen.getByRole("button",{name:"读取原计划"}));
 expect(calls.key).toHaveBeenCalledWith(scope,key,expect.any(AbortSignal));
 expect(sessionStorage.getItem(slot)).toBe(raw);
 expect(screen.getByRole("button",{name:"使用原请求确认创建"})).toBeInTheDocument();
});

for(const action of ["create","update","rename","archive","replay"] as const){
 it(`does not dispatch rule ${action} when original intent storage fails`,async()=>{
  const id="11111111-1111-4111-8111-111111111111",key="22222222-2222-4222-8222-222222222222";
  if(action!=="create"&&action!=="replay")calls.rules.mockResolvedValue({rules:[{id,name:"原模板",revision:1,rule:{},archived:false}]});
  if(action==="replay")sessionStorage.setItem(`listingkit:acquisition:rule:${scope.userId}:${scope.organizationId}`,JSON.stringify({key,input:{action:"create",name:"原模板",rule:{}}}));
  render(<AcquisitionPlansPage scope={scope}/>);
  await userEvent.click(await screen.findByRole("button",{name:"1688在线采集"}));
  await userEvent.click(screen.getByRole("button",{name:"设置规则"}));
  if(action==="create")await userEvent.type(screen.getByLabelText("模板名称"),"新模板");
  const storageBefore=sessionStorage.getItem(`listingkit:acquisition:rule:${scope.userId}:${scope.organizationId}`);
  vi.spyOn(Storage.prototype,"setItem").mockImplementation(()=>{throw new Error("blocked")});
  await userEvent.click(screen.getByRole("button",{name:action==="rename"?"保存名称":action==="archive"?"删除模板":action==="replay"?"使用原请求核实模板":"保存模板"}));
  expect(calls.mutate).not.toHaveBeenCalled();
  expect(screen.getAllByRole("alert").at(-1)).toHaveTextContent("未提交");
  expect(sessionStorage.getItem(`listingkit:acquisition:rule:${scope.userId}:${scope.organizationId}`)).toBe(storageBefore);
 });
}
