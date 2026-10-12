import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {ExecutionModeSettings,executionModeLabels} from "./execution-mode-settings";
import {ConfigurationError} from "@/lib/api/agent-configuration";
const state=vi.hoisted(()=>({view:{} as Record<string,unknown>,send:vi.fn(),guard:vi.fn(()=>()=>{})}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({registerOrganizationSwitchGuard:state.guard})}));
vi.mock("./agent-configuration-hooks",async()=>({...await vi.importActual<object>("./agent-configuration-hooks"),useConfigurationRead:()=>({data:state.view})}));
vi.mock("@/lib/api/agent-configuration",async()=>({...await vi.importActual<object>("@/lib/api/agent-configuration"),configurationRequest:state.send}));
const scope={userId:"actor",organizationId:"org"},subject={kind:"registered_agent" as const,id:"product.title.agent"};
beforeEach(()=>{
 localStorage.clear();state.send.mockReset();state.guard.mockClear();
 state.view={subject,definition:{id:subject.id,version:"v1.0.0"},mode:"default",revision:"0",policyVersion:"agent-execution-policy-v1",canConfigure:true,updatedAt:"2026-10-12T00:00:00Z",updatedBy:""};
 Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true}});
 Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false}});
});
afterEach(()=>{cleanup();localStorage.clear()});
function saveReceipt(_scope:unknown,_path:unknown,_schema:unknown,options:{body:string;headers:Record<string,string>}){
 const payload=JSON.parse(options.body),key=options.headers["Idempotency-Key"];
 const value={...state.view,mode:payload.mode,revision:"1",updatedBy:"actor",receipt:{subject:state.view.subject,commandId:"2b9ac69f-bf25-4d28-a85a-8036ea005013",requestKey:key,operation:"set-execution-mode",mode:payload.mode,beforeRevision:payload.expectedRevision,revision:"1",noop:false,committedAt:"2026-10-12T00:00:00Z"}};
 state.view={...value,receipt:undefined};return Promise.resolve(value);
}
it.each(["default","auto_approve","read_only","full_access"] as const)("saves %s with the observed revision and reads its returned mode",async mode=>{
 state.send.mockImplementation(saveReceipt);render(<ExecutionModeSettings scope={scope} subject={subject}/>);
 fireEvent.click(await screen.findByRole("button",{name:"配置权限"}));fireEvent.click(screen.getByRole("radio",{name:new RegExp("^"+executionModeLabels[mode])}));fireEvent.click(screen.getByRole("button",{name:"保存配置"}));
 await screen.findByText(/权限模式已保存/);expect(state.send).toHaveBeenCalledOnce();
 const args=state.send.mock.calls[0];expect(args[0]).toEqual(scope);expect(args[1]).toBe("execution-modes/registered_agent/product.title.agent");expect(JSON.parse(args[3].body)).toEqual({mode,expectedRevision:"0"});expect(args[3].headers["Idempotency-Key"]).toMatch(/^[a-f0-9-]{36}$/);
 expect(screen.getByText(`${executionModeLabels[mode]} · 配置版本 1`)).toBeInTheDocument();expect(localStorage.length).toBe(0);
});
it("retains an uncertain original key and body across refresh and verifies only that operation",async()=>{
 state.send.mockRejectedValueOnce(new ConfigurationError("OUTCOME_UNKNOWN"));const first=render(<ExecutionModeSettings scope={scope} subject={subject}/>);
 fireEvent.click(await screen.findByRole("button",{name:"配置权限"}));fireEvent.click(screen.getByRole("radio",{name:/^只读/}));fireEvent.click(screen.getByRole("button",{name:"保存配置"}));await screen.findByText(/操作结果尚未确认/);
 const original=state.send.mock.calls[0][3];expect(localStorage.length).toBe(1);first.unmount();state.send.mockImplementation(saveReceipt);
 render(<ExecutionModeSettings scope={scope} subject={subject}/>);fireEvent.click(await screen.findByRole("button",{name:"核实原保存操作"}));await screen.findByText(/权限模式已保存/);
 const next=state.send.mock.calls[1][3];expect(next.body).toBe(original.body);expect(next.headers["Idempotency-Key"]).toBe(original.headers["Idempotency-Key"]);expect(localStorage.length).toBe(0);
});
it("removes configuration controls when the current grant is absent",async()=>{
 state.view={...state.view,canConfigure:false};render(<ExecutionModeSettings scope={scope} subject={subject}/>);await screen.findByText("默认权限 · 配置版本 0");expect(screen.queryByRole("button",{name:"配置权限"})).not.toBeInTheDocument();expect(state.send).not.toHaveBeenCalled();
});
it.each([400,403,404,409])("retains the original uncertain command when verification returns %i",async status=>{
 state.send.mockRejectedValueOnce(new ConfigurationError("OUTCOME_UNKNOWN"));render(<ExecutionModeSettings scope={scope} subject={subject}/>);
 fireEvent.click(await screen.findByRole("button",{name:"配置权限"}));fireEvent.click(screen.getByRole("button",{name:"保存配置"}));await screen.findByText(/操作结果尚未确认/);
 const original=state.send.mock.calls[0][3];state.send.mockRejectedValueOnce(new ConfigurationError("FORBIDDEN",status));fireEvent.click(screen.getByRole("button",{name:"核实原保存操作"}));
 await waitFor(()=>expect(state.send).toHaveBeenCalledTimes(2));await screen.findByText("当前身份没有操作权限。");
 expect(screen.getByRole("button",{name:"核实原保存操作"})).toBeInTheDocument();expect(localStorage.length).toBe(1);expect(state.send.mock.calls[1][3].body).toBe(original.body);
});
it("does not replace the current revision after a known conflict",async()=>{
 state.send.mockRejectedValue(new ConfigurationError("REVISION_MISMATCH",409));render(<ExecutionModeSettings scope={scope} subject={subject}/>);fireEvent.click(await screen.findByRole("button",{name:"配置权限"}));fireEvent.click(screen.getByRole("button",{name:"保存配置"}));await screen.findByText(/配置已被更新/);await waitFor(()=>expect(localStorage.length).toBe(0));expect(screen.getByText("默认权限 · 配置版本 0")).toBeInTheDocument();
});
