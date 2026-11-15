import {beforeEach,expect,it,vi} from "vitest";
import {render,screen,waitFor,cleanup,within,fireEvent} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {ScheduleManager} from "./schedule-manager";
import {ScheduleForm} from "./schedule-form";
import {AIWorkbenchError,requestAIWorkbench} from "@/lib/api/ai-workbench";
vi.mock("@/lib/api/ai-workbench",async original=>{const actual=await original<typeof import("@/lib/api/ai-workbench")>();return {...actual,requestAIWorkbench:vi.fn()};});
vi.mock("./chat-task-selection",()=>({ChatTaskSelection:()=>null}));
vi.mock("@/lib/api/store-center",()=>({catalogSchema:{},storeCenterRequest:vi.fn(async()=>({items:[{id:"33333333-3333-4333-8333-333333333333",name:"日报合成店",platform:"shein"}],pagination:{page:1,pageSize:100,total:1}}))}));
const request=vi.mocked(requestAIWorkbench),scope={userId:"user",organizationId:"org"},id="11111111-1111-4111-8111-111111111111";
const plan={id,revision:"2",state:"ACTIVE" as const,registrationState:"MATCHED" as const,detailsAvailable:true,goalSummary:"完善商品描述",rule:{frequency:"daily" as const,timeZone:"Asia/Shanghai",hour:9,minute:0,weekday:0},maxOccurrences:"3",consumedOccurrences:"1",unit:"ai-micropoint-v1" as const,perRunUpperBoundMicros:"19200000",totalUpperBoundMicros:"57600000",createdAt:"2026-11-01T00:00:00Z"};
const storageKey="ai-workbench-schedule-command:user:org";
Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true;}});
Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false;}});
function mount(){return render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><ScheduleManager scope={scope} authorizationKey="roles" canUse onBack={vi.fn()}/></QueryClientProvider>)}
beforeEach(()=>{cleanup();sessionStorage.clear();window.history.replaceState(null,"","/?view=schedules");request.mockReset();request.mockImplementation(async input=>{
 if(input.route==="schedule-availability")return {enabled:true,ready:true} as never;
 if(input.route==="schedule-list")return {plans:[plan],next:""} as never;
 if(input.route==="schedule-read")return {plan} as never;
 if(input.route==="schedule-history")return {occurrences:[],next:""} as never;
 throw new Error("unexpected mutation");
});});
it.each(["结束计划","删除计划"])("confirms %s in the shared dialog, supports cancel and Esc, and restores focus",async label=>{
 mount();await userEvent.click(await screen.findByRole("button",{name:/完善商品描述/}));
 const trigger=await screen.findByRole("button",{name:label});await userEvent.click(trigger);
 let dialog=screen.getByRole("dialog");expect(within(dialog).getByRole("button",{name:"取消"})).toHaveFocus();
 expect(request.mock.calls.some(([input])=>input.method==="POST")).toBe(false);
 await userEvent.click(within(dialog).getByRole("button",{name:"取消"}));expect(trigger).toHaveFocus();
 await userEvent.click(trigger);dialog=screen.getByRole("dialog");fireEvent(dialog,new Event("cancel",{cancelable:true}));
 expect(screen.queryByRole("dialog")).toBeNull();expect(trigger).toHaveFocus();
 expect(request.mock.calls.some(([input])=>input.method==="POST")).toBe(false);
});
it("deletes an ended plan only after confirmation and removes it from the list",async()=>{
 let removed=false;const current=request.getMockImplementation()!;
 request.mockImplementation(async input=>{
  if(input.route==="schedule-list")return {plans:removed?[]:[{...plan,state:"CLOSED"}],next:""} as never;
  if(input.route==="schedule-read")return {plan:{...plan,state:"CLOSED"}} as never;
  if(String(input.route)==="schedule-delete"){removed=true;expect(input.body).toEqual({expectedRevision:"2"});return {plan:{...plan,state:"CLOSED",deleted:true},planSaved:true} as never;}
  return current(input);
 });
 mount();await userEvent.click(await screen.findByRole("button",{name:/完善商品描述/}));
 await userEvent.click(await screen.findByRole("button",{name:"删除计划"}));
 await userEvent.click(within(screen.getByRole("dialog")).getByRole("button",{name:"删除计划"}));
 await screen.findByText("计划已删除，已有结果和执行记录仍保留。");
 await screen.findByRole("heading",{name:"还没有定时任务"});
 expect(request.mock.calls.filter(([input])=>String(input.route)==="schedule-delete")).toHaveLength(1);
});
it("restores an uncertain pause after reload and retries the exact original version and key",async()=>{
 const command={action:"pause",path:`schedules/${id}/pause`,key:id,body:{expectedRevision:"2"}};
 sessionStorage.setItem(storageKey,JSON.stringify(command));
 request.mockImplementationOnce(async()=>({enabled:true,ready:true}) as never);
 mount();await screen.findByRole("heading",{name:"上次操作结果待核实"});
 expect(screen.getByRole("button",{name:"创建定时任务"})).toBeDisabled();
 request.mockImplementation(async input=>{
  if(input.route==="schedule-pause"){expect(input.key).toBe(id);expect(input.body).toEqual({expectedRevision:"2"});return {plan:{...plan,state:"PAUSED"},planSaved:true} as never;}
  if(input.route==="schedule-availability")return {enabled:true,ready:true} as never;
  if(input.route==="schedule-list")return {plans:[plan],next:""} as never;
  if(input.route==="schedule-read")return {plan:{...plan,state:"PAUSED"}} as never;
  return {occurrences:[],next:""} as never;
 });
 await userEvent.click(screen.getByRole("button",{name:"核实原操作"}));
 await screen.findByRole("button",{name:"恢复未来执行"});expect(sessionStorage.getItem(storageKey)).toBeNull();
});
it("recovers an uncertain deletion with the same key after reload without losing the original command",async()=>{
 const command={action:"delete",path:`schedules/${id}/delete`,key:id,body:{expectedRevision:"2"}};
 sessionStorage.setItem(storageKey,JSON.stringify(command));const current=request.getMockImplementation()!;let tries=0;
 request.mockImplementation(async input=>{
  if(input.route==="schedule-delete"){
   expect(input.key).toBe(command.key);expect(input.body).toEqual(command.body);
   if(++tries===1)throw new AIWorkbenchError("OUTCOME_UNKNOWN");
   return {plan:{...plan,state:"CLOSED",deleted:true},planSaved:true} as never;
  }
  if(input.route==="schedule-list"&&tries>1)return {plans:[],next:""} as never;
  return current(input);
 });
 mount();await userEvent.click(await screen.findByRole("button",{name:"核实原操作"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"核实原操作"})).toBeEnabled());
 expect(JSON.parse(sessionStorage.getItem(storageKey)!)).toEqual(command);
 await userEvent.click(screen.getByRole("button",{name:"核实原操作"}));
 await screen.findByText("计划已删除，已有结果和执行记录仍保留。");expect(sessionStorage.getItem(storageKey)).toBeNull();expect(tries).toBe(2);
});
it.each(["OUTCOME_UNKNOWN","DEPENDENCY_UNAVAILABLE","DEADLINE_EXCEEDED","TASK_OUTCOME_UNKNOWN"])("keeps the original command when %s cannot prove rejection",async code=>{
 mount();await userEvent.click(await screen.findByRole("button",{name:/完善商品描述/}));
 await screen.findByRole("button",{name:"暂停未来执行"});
 const current=request.getMockImplementation()!;
 request.mockImplementation(async input=>{if(input.route==="schedule-pause")throw new AIWorkbenchError(code);return current(input)});
 await userEvent.click(screen.getByRole("button",{name:"暂停未来执行"}));
 await screen.findByRole("heading",{name:"上次操作结果待核实"});
 const saved=JSON.parse(sessionStorage.getItem(storageKey)!);expect(saved.body).toEqual({expectedRevision:"2"});
 await userEvent.click(screen.getByRole("button",{name:"核实原操作"}));
 await waitFor(()=>expect(request.mock.calls.filter(([i])=>i.route==="schedule-pause")).toHaveLength(2));
 const sends=request.mock.calls.filter(([i])=>i.route==="schedule-pause");expect(sends[0][0].key).toBe(sends[1][0].key);
});
it("retains time inputs after a quote failure and requires a fresh successful quote before saving",async()=>{
 request.mockRejectedValue(new AIWorkbenchError("PROPOSAL_STALE"));const save=vi.fn();
 render(<ScheduleForm scope={scope} authorizationKey="roles" proposalId={id} disabled={false} onSave={save} onCancel={vi.fn()}/>);
 const at=screen.getByLabelText("执行日期与时间（本机时区）");await userEvent.type(at,"2030-11-20T09:00");
 // Native datetime controls are not text inputs in all test environments.
 const {fireEvent}=await import("@testing-library/react");fireEvent.change(at,{target:{value:"2030-11-20T09:00"}});
 await userEvent.click(screen.getByRole("button",{name:"核对时间与预算"}));await screen.findByRole("alert");
 expect(at).toHaveValue("2030-11-20T09:00");expect(save).not.toHaveBeenCalled();expect(screen.queryByRole("button",{name:"确认并保存计划"})).toBeNull();
});
it("retains an earlier unknown create through a later permission failure",async()=>{
 const body={proposalId:id,rule:{frequency:"once",timeZone:"UTC",atUtc:"2030-11-20T00:00:00Z",hour:0,minute:0,weekday:0},maxOccurrences:"1",confirmation:"a".repeat(64)};
 const command={action:"create",path:"schedules",key:id,body};sessionStorage.setItem(storageKey,JSON.stringify(command));
 const current=request.getMockImplementation()!;let tries=0;
 request.mockImplementation(async input=>{if(input.route==="schedule-create"){expect(input.key).toBe(id);expect(input.body).toEqual(body);if(++tries===1)throw new AIWorkbenchError("FORBIDDEN");return {plan,planSaved:true} as never;}return current(input)});
 mount();await userEvent.click(await screen.findByRole("button",{name:"核实原操作"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"核实原操作"})).toBeEnabled());expect(JSON.parse(sessionStorage.getItem(storageKey)!)).toEqual(command);
 await userEvent.click(screen.getByRole("button",{name:"核实原操作"}));await screen.findByRole("button",{name:"暂停未来执行"});expect(tries).toBe(2);expect(sessionStorage.getItem(storageKey)).toBeNull();
});
it("retires an unknown command only after an authoritative durable rejection",async()=>{
 const body={proposalId:id,rule:{frequency:"once",timeZone:"UTC",atUtc:"2020-11-20T00:00:00Z",hour:0,minute:0,weekday:0},maxOccurrences:"1",confirmation:"a".repeat(64)};
 const command={action:"create",path:"schedules",key:id,body};sessionStorage.setItem(storageKey,JSON.stringify(command));
 const current=request.getMockImplementation()!;let tries=0;
 request.mockImplementation(async input=>{if(input.route==="schedule-create")throw ++tries===1?new AIWorkbenchError("INVALID_REQUEST"):new AIWorkbenchError("SCHEDULE_COMMAND_REJECTED",undefined,undefined,false);return current(input)});
 mount();await userEvent.click(await screen.findByRole("button",{name:"核实原操作"}));await waitFor(()=>expect(screen.getByRole("button",{name:"核实原操作"})).toBeEnabled());expect(sessionStorage.getItem(storageKey)).not.toBeNull();
 await userEvent.click(screen.getByRole("button",{name:"核实原操作"}));await waitFor(()=>expect(screen.queryByRole("heading",{name:"上次操作结果待核实"})).toBeNull());
 expect(sessionStorage.getItem(storageKey)).toBeNull();expect(screen.getByRole("button",{name:"创建定时任务"})).toBeEnabled();
});

it("confirms a deterministic store report without product or model fields",async()=>{
 const save=vi.fn<(input:unknown)=>Promise<void>>(async()=>{});
 request.mockImplementation(async input=>{expect(input.route).toBe("schedule-quote");expect(input.body).not.toHaveProperty("selection");expect(input.body).not.toHaveProperty("proposalId");return {kind:"store.operating-daily-report",report:(input.body as {report:unknown}).report,confirmation:"a".repeat(64),goalSummary:"店铺经营日报",rule:(input.body as {rule:unknown}).rule,maxOccurrences:"1",unit:"none",perRunUpperBoundMicros:"0",totalUpperBoundMicros:"0",nextAt:"2030-11-20T01:00:00Z"} as never;});
 render(<QueryClientProvider client={new QueryClient()}><ScheduleForm scope={scope} authorizationKey="auth" disabled={false} dailyReportEnabled contentEnabled={false} onSave={save} onCancel={vi.fn()}/></QueryClientProvider>);
 await screen.findByRole("option",{name:/日报合成店/});
 await userEvent.selectOptions(await screen.findByLabelText("店铺"),"33333333-3333-4333-8333-333333333333");
 await userEvent.type(screen.getByLabelText("执行日期与时间（本机时区）"),"2030-11-20T09:00");
 await userEvent.click(screen.getByRole("button",{name:"核对计划"}));
 await screen.findByText("每次执行自动保存一份个人报告");expect(save).not.toHaveBeenCalled();expect(screen.queryByText(/AI 积分/)).toBeNull();
 await userEvent.click(screen.getByRole("button",{name:"确认并保存计划"}));expect(save).toHaveBeenCalledOnce();expect(save.mock.calls[0][0]).toMatchObject({kind:"store.operating-daily-report",report:{storeId:"33333333-3333-4333-8333-333333333333"}});
});

it("opens a full plan editor without changing the plan or starting a task",async()=>{
 mount();await userEvent.click(await screen.findByRole("button",{name:/完善商品描述/}));
 await userEvent.click(await screen.findByRole("button",{name:"编辑计划"}));
 expect(await screen.findByRole("heading",{name:"编辑计划"})).toBeVisible();
 expect(screen.getByRole("textbox",{name:"任务目标"})).toHaveValue(plan.goalSummary);
 expect(request.mock.calls.some(([input])=>input.method!=="GET")).toBe(false);
});

it("copies an ended plan into an unsaved form with its settings and a fresh date",async()=>{
 const current=request.getMockImplementation()!;
 request.mockImplementation(async input=>input.route==="schedule-read"?{plan:{...plan,state:"CLOSED",rule:{...plan.rule,frequency:"once",atUtc:"2026-11-01T01:00:00Z"}}} as never:current(input));
 mount();await userEvent.click(await screen.findByRole("button",{name:/完善商品描述/}));
 await userEvent.click(await screen.findByRole("button",{name:"复制新建"}));
 expect(await screen.findByRole("heading",{name:"复制新建定时任务"})).toBeVisible();
 expect(screen.getByRole("textbox",{name:"任务目标"})).toHaveValue(plan.goalSummary);
 expect(screen.getByLabelText("执行日期与时间（本机时区）")).toHaveValue("");
 expect(request.mock.calls.some(([input])=>input.method!=="GET")).toBe(false);
});

it("requotes all future settings and distinguishes remaining budget from past runs",async()=>{
 const save=vi.fn(async()=>{}),selection={subject:{kind:"own_product" as const,id,catalogVersion:"1",publicationId:id},capability:"product.content.localize" as const,parameters:{targetLanguage:"en-US",description:true}};
 request.mockImplementation(async input=>{expect(input.route).toBe("schedule-quote");expect(input.body).toMatchObject({scheduleId:id,expectedRevision:"2",goalSummary:"Updated future description",maxOccurrences:"4",selection});return {selection,confirmation:"a".repeat(64),goalSummary:"Updated future description",rule:plan.rule,maxOccurrences:"4",consumedOccurrences:"1",unit:plan.unit,perRunUpperBoundMicros:"19200000",totalUpperBoundMicros:"76800000",nextAt:"2030-11-20T01:00:00Z",executionMode:"default"} as never;});
 render(<QueryClientProvider client={new QueryClient()}><ScheduleForm scope={scope} authorizationKey="auth" plan={{...plan,deleted:false,selection}} disabled={false} onSave={save} onCancel={vi.fn()}/></QueryClientProvider>);
 await userEvent.clear(screen.getByRole("textbox",{name:"任务目标"}));await userEvent.type(screen.getByRole("textbox",{name:"任务目标"}),"Updated future description");
 await userEvent.clear(screen.getByLabelText("最多执行次数"));await userEvent.type(screen.getByLabelText("最多执行次数"),"4");
 await userEvent.click(screen.getByRole("button",{name:"核对时间与预算"}));
 await screen.findByText("已执行 1 次，后续可执行 3 次");expect(screen.getByText("57.6 AI 积分")).toBeVisible();expect(save).not.toHaveBeenCalled();
 await userEvent.click(screen.getByText("查看所选商品资料"));
 expect(within(screen.getByRole("region",{name:"确认定时计划"})).getByText(`商品标识 ${id}`)).toBeVisible();
 await userEvent.clear(screen.getByLabelText("最多执行次数"));await userEvent.type(screen.getByLabelText("最多执行次数"),"5");
 expect(screen.queryByRole("button",{name:"确认修改计划"})).toBeNull();expect(screen.getByRole("textbox",{name:"任务目标"})).toHaveValue("Updated future description");
});

it("keeps the authorized original store identifiable when outside loaded catalog pages",async()=>{
 const storeId="44444444-4444-4444-8444-444444444444";
 render(<QueryClientProvider client={new QueryClient()}><ScheduleForm scope={scope} authorizationKey="auth" copyFrom={{...plan,deleted:false,kind:"store.operating-daily-report",storeName:"原计划授权店铺",report:{storeId,version:1,periodPolicy:"previous_natural_day",businessZone:"Asia/Shanghai"}}} disabled={false} onSave={vi.fn()} onCancel={vi.fn()}/></QueryClientProvider>);
 await screen.findByRole("option",{name:/日报合成店/});
 expect(screen.getByRole("option",{name:/原计划授权店铺/})).toHaveAttribute("value",storeId);
 expect(screen.getByLabelText("店铺")).toHaveDisplayValue("原计划授权店铺");
 expect(request).not.toHaveBeenCalled();
});

it("rejects a count already consumed before requesting a quote",async()=>{
 render(<QueryClientProvider client={new QueryClient()}><ScheduleForm scope={scope} authorizationKey="auth" plan={{...plan,deleted:false}} disabled={false} onSave={vi.fn()} onCancel={vi.fn()}/></QueryClientProvider>);
 await userEvent.clear(screen.getByLabelText("最多执行次数"));await userEvent.type(screen.getByLabelText("最多执行次数"),"1");await userEvent.click(screen.getByRole("button",{name:"核对时间与预算"}));
 expect(screen.getByRole("alert")).toHaveTextContent("次数上限需大于已执行的 1 次");expect(request).not.toHaveBeenCalled();
});
