// jsdom does not implement the browser modal dialog API.
Object.defineProperty(HTMLDialogElement.prototype, "showModal", {configurable:true, value:function(this:HTMLDialogElement){this.open=true;}});
Object.defineProperty(HTMLDialogElement.prototype, "close", {configurable:true, value:function(this:HTMLDialogElement){this.open=false;}});
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {ChatStorePicker,chatStoreChoiceKey,type ChatStoreState} from "./chat-store-picker";

const calls=vi.hoisted(()=>({scope:vi.fn(),connection:vi.fn()}));
vi.mock("@/lib/api/ai-workbench",()=>({requestAIWorkbench:calls.scope}));
vi.mock("@/lib/api/store-connection",()=>({getStoreConnection:calls.connection}));
const scope={userId:"user-a",organizationId:"org-a"},a="11111111-1111-4111-8111-111111111111",b="22222222-2222-4222-8222-222222222222",token="a".repeat(64);
const option=(id:string,name:string)=>({scope:{version:"workbench-scope-v1",kind:"store",storeId:id},storeName:name,confirmation:token});
const wrapper=(onChange:(value:ChatStoreState)=>void)=><QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><ChatStorePicker scope={scope} authorizationKey="auth" context="conversation:one" disabled={false} onChange={onChange}/></QueryClientProvider>;
afterEach(()=>{cleanup();sessionStorage.clear();vi.resetAllMocks()});

it("reads only after opening, offers confirmed connections and allows clearing without sending",async()=>{
 calls.scope.mockResolvedValue({options:[option(a,"店铺 A"),option(b,"店铺 B")],next:"",incomplete:false});
 calls.connection.mockImplementation(async(_scope,id)=>({connectionStatus:id===a?"connected":"expired",version:3,observedAt:"2026-11-14T00:00:00Z"}));
 const onChange=vi.fn();render(wrapper(onChange));
 await waitFor(()=>expect(onChange).toHaveBeenCalledWith({choice:undefined,ready:true}));
 expect(calls.scope).not.toHaveBeenCalled();expect(calls.connection).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole("button",{name:"选择店铺"}));
 expect(await screen.findByRole("button",{name:"店铺 A"})).toBeVisible();expect(screen.queryByRole("button",{name:"店铺 B"})).toBeNull();
 expect(screen.getByRole("link",{name:"连接新店铺"})).toHaveAttribute("href","/workbench/stores");
 fireEvent.click(screen.getByRole("button",{name:"店铺 A"}));
 expect(onChange).toHaveBeenLastCalledWith({choice:{kind:"store",storeId:a,scopeConfirmation:token,connectionVersion:3},ready:true});
 fireEvent.click(screen.getByRole("button",{name:"取消选择"}));
 expect(onChange).toHaveBeenLastCalledWith({choice:{kind:"none"},ready:true});
});

it("does not expose a saved name or enable use after permission is revoked",async()=>{
 sessionStorage.setItem(chatStoreChoiceKey(scope,"conversation:one"),JSON.stringify({kind:"store",storeId:a,scopeConfirmation:token,connectionVersion:3}));
 calls.scope.mockRejectedValue(new Error("revoked"));calls.connection.mockRejectedValue(new Error("revoked"));
 const onChange=vi.fn();render(wrapper(onChange));
 expect(await screen.findByText("所选店铺暂不可用，请重新选择。")).toBeVisible();
 expect(onChange).toHaveBeenLastCalledWith({choice:undefined,ready:false});expect(screen.queryByText("店铺 A")).toBeNull();
});

it("switches and clears in the same conversation without changing the saved draft",async()=>{
 calls.scope.mockResolvedValue({options:[option(a,"店铺 A"),option(b,"店铺 B")],next:"",incomplete:false});
 calls.connection.mockResolvedValue({connectionStatus:"connected",version:3,observedAt:"2026-11-14T00:00:00Z"});
 sessionStorage.setItem("unrelated-draft","保留原文");const onChange=vi.fn();render(wrapper(onChange));
 fireEvent.click(screen.getByRole("button",{name:"选择店铺"}));fireEvent.click(await screen.findByRole("button",{name:"店铺 A"}));
 fireEvent.click(screen.getByRole("button",{name:"店铺 A"}));fireEvent.click(await screen.findByRole("button",{name:"店铺 B"}));
 expect(onChange).toHaveBeenLastCalledWith({choice:{kind:"store",storeId:b,scopeConfirmation:token,connectionVersion:3},ready:true});
 fireEvent.click(screen.getByRole("button",{name:"取消选择"}));
 expect(JSON.parse(sessionStorage.getItem(chatStoreChoiceKey(scope,"conversation:one"))!)).toEqual({kind:"none"});
 expect(sessionStorage.getItem("unrelated-draft")).toBe("保留原文");
});

it("opens an accessible drawer and restores focus after Escape",async()=>{
 calls.scope.mockResolvedValue({options:[],next:"",incomplete:false});
 render(wrapper(vi.fn()));
 const trigger=screen.getByRole("button",{name:"选择店铺"});trigger.focus();fireEvent.click(trigger);
 const dialog=await screen.findByRole("dialog",{name:"选择店铺"});
 expect(screen.getByRole("button",{name:"关闭选择店铺"})).toHaveFocus();
 fireEvent(dialog,new Event("cancel",{cancelable:true}));
 await waitFor(()=>expect(screen.queryByRole("dialog",{name:"选择店铺"})).toBeNull());
 expect(trigger).toHaveFocus();
});
