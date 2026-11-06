import {cleanup,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,expect,it,vi} from "vitest";
import {UploadPlanPanel} from "./upload-plan-panel";
import type {SupplyCommandState} from "./use-supply-command";
const f=vi.hoisted(()=>({read:vi.fn(),preview:vi.fn()}));
vi.mock("@/lib/api/supply-chain",()=>({readUploadPlan:f.read,previewUploadPlan:f.preview,listUploadPlans:vi.fn()}));
const id="11111111-1111-4111-8111-111111111111",source="22222222-2222-4222-8222-222222222222";
const plan={id,input:{preparationId:source,expectedRevision:1,storeId:source,rule:"standard" as const,publicationMode:"platform" as const},storeBindingHash:"a".repeat(64),count:1005,revision:3,createdAt:"2026-11-03T00:00:00Z"};
afterEach(cleanup);
function setup(){f.read.mockResolvedValue(plan);f.preview.mockResolvedValue({plan,items:[],count:1005,actionable:3,counts:{upload:3,unknown:2,succeeded:1000},hash:"b".repeat(64)});const execute=vi.fn();const props={scope:{userId:"actor",organizationId:"org"},planId:id,reload:0,command:{execute} as unknown as SupplyCommandState,blocked:false,canSubmit:true,onOpen:vi.fn(),onLoaded:vi.fn(),onItem:vi.fn(),onOperation:vi.fn(),onReview:vi.fn()};return {execute,props};}
it("only continues an explicit complete preview of the original plan and leaves successful/unknown items out",async()=>{
 const {execute,props}=setup();const view=render(<UploadPlanPanel {...props}/>);
 await waitFor(()=>expect(props.onLoaded).toHaveBeenCalledWith(plan));expect(execute).not.toHaveBeenCalled();expect(f.preview).not.toHaveBeenCalled();
 await userEvent.click(screen.getByRole("button",{name:"检查进度并继续"}));
 await screen.findByRole("dialog",{name:"确认继续上传"});
 expect(screen.getByText(/本次确认覆盖完整 1005 件/)).toBeInTheDocument();
 await userEvent.click(screen.getByRole("button",{name:"确认继续 3 件"}));
 expect(execute).toHaveBeenCalledExactlyOnceWith("continue-upload-plan",{planId:id,expectedRevision:3,previewHash:"b".repeat(64)});
 view.unmount();render(<UploadPlanPanel {...props}/>);await waitFor(()=>expect(props.onLoaded).toHaveBeenCalledTimes(2));expect(execute).toHaveBeenCalledTimes(1);
});
it("invalidates the preview after a save or Apply instead of sending its stale approval",async()=>{
 const {execute,props}=setup();const view=render(<UploadPlanPanel {...props}/>);
 await waitFor(()=>expect(props.onLoaded).toHaveBeenCalled());await userEvent.click(screen.getByRole("button",{name:"检查进度并继续"}));await screen.findByRole("dialog");
 view.rerender(<UploadPlanPanel {...props} reload={1}/>);await waitFor(()=>expect(screen.queryByRole("dialog")).not.toBeInTheDocument());expect(execute).not.toHaveBeenCalled();
});

it("shows the original Walmart rejection and wire hash before confirming a new command", async () => {
 const {execute, props} = setup();
 const rejectedAttempt = "33333333-3333-4333-8333-333333333333";
 const wireHash = "c".repeat(64);
 const retryPlan = {...plan, count:1, input:{...plan.input, sourceIds:[source]}};
 const rejection = `Walmart 明确拒绝 · 原尝试 ${rejectedAttempt} · 同载荷重试`;
 f.read.mockResolvedValue(retryPlan);
 f.preview.mockResolvedValue({plan:retryPlan, count:1, actionable:1, counts:{upload:1}, hash:"d".repeat(64), items:[{
  item:{sourceId:source}, step:"upload", facts:{hasTarget:true,target:{recordId:source,revision:2},publicationSummary:[`Walmart · MP_ITEM_MATCH · 1 SKU · ${wireHash}`,rejection]},
 }]});
 render(<UploadPlanPanel {...props}/>);
 const inspect = screen.getByRole("button",{name:"检查进度并继续"});
 await waitFor(()=>expect(inspect).toBeEnabled());
 await userEvent.click(inspect);
 expect(await screen.findByText(rejection)).toBeInTheDocument();
 expect(screen.getByText(new RegExp(wireHash))).toBeInTheDocument();
 expect(execute).not.toHaveBeenCalled();
 await userEvent.click(screen.getByRole("button",{name:"确认继续 1 件"}));
 expect(execute).toHaveBeenCalledExactlyOnceWith("continue-upload-plan",{planId:id,expectedRevision:3,previewHash:"d".repeat(64)});
});
