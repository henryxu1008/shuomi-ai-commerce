import {act,cleanup,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,it,expect,vi} from "vitest";
import {AcquisitionPlansPage} from "./acquisition-plans-page";
import {AcquisitionAPIError} from "@/lib/api/product-acquisition";
const calls=vi.hoisted(()=>({create:vi.fn(),advance:vi.fn(),read:vi.fn(),key:vi.fn(),rules:vi.fn(),mutate:vi.fn()}));
vi.mock("@/lib/api/acquisition-plan",()=>({listAcquisitionPlans:async()=>({plans:[],canAutoTransfer:true}),listAcquisitionRules:calls.rules,createAcquisitionPlan:calls.create,advanceAcquisitionPlan:calls.advance,readAcquisitionPlan:calls.read,readAcquisitionPlanByKey:calls.key,mutateAcquisitionRule:calls.mutate}));
beforeEach(()=>{sessionStorage.clear();vi.clearAllMocks();calls.rules.mockResolvedValue({rules:[]});HTMLDialogElement.prototype.showModal=function(){this.setAttribute("open","")};HTMLDialogElement.prototype.close=function(){this.removeAttribute("open")};});
afterEach(()=>vi.restoreAllMocks());
afterEach(cleanup);
const scope={userId:"actor",organizationId:"org-A"};
async function start(){await userEvent.click(await screen.findByRole("button",{name:"1688在线采集"}));await userEvent.type(screen.getByLabelText("1688 商品链接"),"123");await userEvent.click(screen.getByRole("button",{name:"开始采集"}));}
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
