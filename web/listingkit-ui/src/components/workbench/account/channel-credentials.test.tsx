import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {cleanup,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {ChannelCredentialsManager} from "./channel-credentials";

const clients:QueryClient[]=[];
const connection={organizationId:"org-a",connectionId:"shared",provider:"openai",channel:"openai-direct",region:"global",account:"",revision:"0",enabled:false,keyPresent:false,models:[{id:"mini",version:"v1",model:"mini",category:"text"},{id:"mini",version:"v1",model:"mini",category:"multimodal"}]};
const scope={userId:"manager",organizationId:"org-a"};
function mount(){const client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);return render(<QueryClientProvider client={client}><ChannelCredentialsManager scope={scope}/></QueryClientProvider>)}
function fixture(){const fetch=vi.fn(async(url:string,init:RequestInit={})=>init.method==="POST"?Promise.reject(new Error("lost ACK")):url.includes("/receipts/")?Response.json({code:"NOT_FOUND"},{status:404}):Response.json({organizationId:"org-a",canManage:true,connections:[connection]}));vi.stubGlobal("fetch",fetch);return fetch}
beforeEach(()=>{
 Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true}});
 Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false}});
});
afterEach(()=>{cleanup();for(const c of clients)c.clear();clients.length=0;sessionStorage.clear();localStorage.clear();vi.unstubAllGlobals()});
it("clears a submitted Key, retains only sanitized references after ACK loss and never infers success from current revision",async()=>{
 const user=userEvent.setup(),fetch=fixture();const view=mount();
 await user.click(await screen.findByRole("button",{name:"录入 Key"}));
 await user.type(screen.getByLabelText("账号备注"),"account");await user.type(screen.getByLabelText("新的渠道 Key"),"synthetic-ui-secret");
 await user.click(screen.getByRole("button",{name:"确认保存 Key"}));
 await screen.findByRole("heading",{name:"保存结果待核实"});
 expect(screen.getByLabelText("新的渠道 Key")).toHaveValue("");
 const storage=JSON.stringify({...sessionStorage});expect(storage).not.toContain("synthetic-ui-secret");expect(storage).not.toContain("apiKey");expect(JSON.stringify({...localStorage})).not.toContain("synthetic-ui-secret");
 const post=fetch.mock.calls.find(([,init])=>init?.method==="POST")!;const key=new Headers(post[1]?.headers).get("Idempotency-Key")!;
 view.unmount();
 fetch.mockImplementation(async url=>url.includes("/receipts/")?Response.json({code:"NOT_FOUND"},{status:404}):Response.json({organizationId:"org-a",canManage:true,connections:[{...connection,account:"account",revision:"9",enabled:true,keyPresent:true}]}));
 mount();await screen.findByRole("heading",{name:"保存结果待核实"});
 await user.click(screen.getByRole("button",{name:"查询本次结果"}));
 await waitFor(()=>expect(fetch.mock.calls.some(([url])=>url.endsWith("/receipts/"+key))).toBe(true));
 expect(screen.getByRole("heading",{name:"保存结果待核实"})).toBeVisible();
 expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(1);
 expect(screen.getByRole("button",{name:"轮换 Key"})).toBeDisabled();
});
it("recovers the exact scoped receipt without resending a Key and keeps other-target replies pending",async()=>{
 const user=userEvent.setup(),fetch=fixture();mount();
 await user.click(await screen.findByRole("button",{name:"录入 Key"}));await user.type(screen.getByLabelText("账号备注"),"account");await user.type(screen.getByLabelText("新的渠道 Key"),"synthetic-ui-secret");await user.click(screen.getByRole("button",{name:"确认保存 Key"}));
 await screen.findByRole("heading",{name:"保存结果待核实"});
 const key=new Headers(fetch.mock.calls.find(([,init])=>init?.method==="POST")![1]?.headers).get("Idempotency-Key")!;
 const original=fetch.getMockImplementation()!;
 let actor="other";
 fetch.mockImplementation(async(url,init={})=>url.includes("/receipts/")?Response.json({receipt:{organizationId:"org-a",actorId:actor,requestKey:key,connectionId:"shared",expectedRevision:"0",action:"set-key",connection:{...connection,account:"account",revision:"1",enabled:true,keyPresent:true,models:[]}}}):original(url,init));
 await user.click(screen.getByRole("button",{name:"查询本次结果"}));await screen.findByRole("alert");expect(screen.getByRole("heading",{name:"保存结果待核实"})).toBeVisible();
 actor="manager";await user.click(screen.getByRole("button",{name:"查询本次结果"}));await screen.findByRole("heading",{name:"本次保存回执"});
 expect(screen.queryByRole("heading",{name:"保存结果待核实"})).not.toBeInTheDocument();
 expect(fetch.mock.calls.filter(([,init])=>init?.method==="POST")).toHaveLength(1);expect(sessionStorage.length).toBe(0);
});
