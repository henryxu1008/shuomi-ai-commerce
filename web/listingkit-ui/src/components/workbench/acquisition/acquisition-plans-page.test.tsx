import {act,cleanup,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,it,expect,vi} from "vitest";
import {AcquisitionPlansPage} from "./acquisition-plans-page";
import {AcquisitionAPIError} from "@/lib/api/product-acquisition";
const calls=vi.hoisted(()=>({create:vi.fn(),advance:vi.fn(),read:vi.fn(),key:vi.fn()}));
vi.mock("@/lib/api/acquisition-plan",()=>({listAcquisitionPlans:async()=>({plans:[],canAutoTransfer:true}),listAcquisitionRules:async()=>({rules:[]}),createAcquisitionPlan:calls.create,advanceAcquisitionPlan:calls.advance,readAcquisitionPlan:calls.read,readAcquisitionPlanByKey:calls.key,mutateAcquisitionRule:vi.fn()}));
beforeEach(()=>{sessionStorage.clear();vi.clearAllMocks();HTMLDialogElement.prototype.showModal=function(){this.setAttribute("open","")};HTMLDialogElement.prototype.close=function(){this.removeAttribute("open")};});
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
