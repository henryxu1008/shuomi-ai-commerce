import {beforeEach,expect,it,vi} from "vitest";
import {render,screen,waitFor,cleanup} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {ScheduleManager} from "./schedule-manager";
import {ScheduleForm} from "./schedule-form";
import {AIWorkbenchError,requestAIWorkbench} from "@/lib/api/ai-workbench";
vi.mock("@/lib/api/ai-workbench",async original=>{const actual=await original<typeof import("@/lib/api/ai-workbench")>();return {...actual,requestAIWorkbench:vi.fn()};});
vi.mock("./chat-task-selection",()=>({ChatTaskSelection:()=>null}));
const request=vi.mocked(requestAIWorkbench),scope={userId:"user",organizationId:"org"},id="11111111-1111-4111-8111-111111111111";
const plan={id,revision:"2",state:"ACTIVE" as const,registrationState:"MATCHED" as const,detailsAvailable:true,goalSummary:"完善商品描述",rule:{frequency:"daily" as const,timeZone:"Asia/Shanghai",hour:9,minute:0,weekday:0},maxOccurrences:"3",consumedOccurrences:"1",unit:"ai-micropoint-v1" as const,perRunUpperBoundMicros:"19200000",totalUpperBoundMicros:"57600000",createdAt:"2026-11-01T00:00:00Z"};
const storageKey="ai-workbench-schedule-command:user:org";
function mount(){return render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><ScheduleManager scope={scope} authorizationKey="roles" canUse onBack={vi.fn()}/></QueryClientProvider>)}
beforeEach(()=>{cleanup();sessionStorage.clear();window.history.replaceState(null,"","/?view=schedules");request.mockReset();request.mockImplementation(async input=>{
 if(input.route==="schedule-availability")return {enabled:true,ready:true} as never;
 if(input.route==="schedule-list")return {plans:[plan],next:""} as never;
 if(input.route==="schedule-read")return {plan} as never;
 if(input.route==="schedule-history")return {occurrences:[],next:""} as never;
 throw new Error("unexpected mutation");
});});
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
