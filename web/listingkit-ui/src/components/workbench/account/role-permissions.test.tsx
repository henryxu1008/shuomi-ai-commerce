import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, afterEach, expect, it, vi } from "vitest";
import { RolePermissions } from "./role-permissions";
import type { EnterpriseRoles } from "@/lib/api/enterprise-roles";
const api = vi.hoisted(() => ({ read: vi.fn(), create: vi.fn(), save: vi.fn() }));
vi.mock("@/lib/api/enterprise-roles", async original => ({ ...(await original<typeof import("@/lib/api/enterprise-roles")>()), getEnterpriseRoles: api.read, createEnterpriseRole: api.create, saveEnterpriseRole: api.save }));
const key="sumi_role_e87cb45c05ad389dff6dea6e7bf581ee_01";
const role={id:key,name:"客服",modules:["members"],version:1,system:false};
const data: EnterpriseRoles={schemaVersion:"enterprise-roles-v1",userId:"actor",organizationId:"org",items:[{id:"listingkit_admin",name:"管理员",modules:["members"],version:0,system:true},role],catalog:[{id:"members",group:"我的账户",label:"成员与权限",available:true,permissions:[]},{id:"reports",group:"AI工作台",label:"我的报告",available:false,permissions:[]}],canManage:true,canCreate:true,remainingSlots:63};
beforeEach(()=>{Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true}});Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false}});});
afterEach(()=>{cleanup();sessionStorage.clear();vi.clearAllMocks();});
function mount(value = data){api.read.mockResolvedValue(value);const client=new QueryClient({defaultOptions:{queries:{retry:false}}});return render(<QueryClientProvider client={client}><RolePermissions scope={{expectedUserId:"actor",expectedOrganizationId:"org"}} onChanged={vi.fn()} onAuthorityFailure={vi.fn()} /></QueryClientProvider>);}
const browserModule={id:"store-browser",group:"店铺中心",label:"店铺浏览器使用",available:true,permissions:["workbench.store.read","workbench.store.browser.use"]};
it.each([true,false])("shows fixed system modules as read-only inclusion when canManage=%s",async canManage=>{
 mount({...data,canManage,canCreate:canManage,catalog:[...data.catalog,browserModule]});const user=userEvent.setup();
 const editor=await screen.findByRole("region",{name:"角色模块权限"});
 for(const group of ["我的账户","店铺中心","AI工作台"])await user.click(within(editor).getByText(group,{selector:"strong"}));
 expect(within(screen.getByText("成员与权限").parentElement!).getByText("已包含")).toBeVisible();
 expect(within(screen.getByText("店铺浏览器使用").parentElement!).getByText("未包含")).toBeVisible();
 expect(within(screen.getByText("我的报告").parentElement!).getByText("尚未开放")).toBeVisible();
 expect(within(editor).queryByRole("button",{name:/添加|移除|保存/})).not.toBeInTheDocument();
 expect(within(editor).queryByText("展开一级菜单，添加或移除子菜单，完成后点击保存。")).not.toBeInTheDocument();
 expect(api.save).not.toHaveBeenCalled();expect(api.create).not.toHaveBeenCalled();
});
it("keeps the browser module explicitly selectable for custom roles",async()=>{
 api.save.mockResolvedValue({role:{...role,modules:["members","store-browser"],version:2}});
 mount({...data,catalog:[...data.catalog,browserModule]});const user=userEvent.setup();
 await user.click(await screen.findByRole("button",{name:"客服"}));await user.click(screen.getByText("店铺中心",{selector:"strong"}));
 await user.click(screen.getByRole("button",{name:"添加店铺浏览器使用"}));
 expect(screen.getByRole("button",{name:"移除店铺浏览器使用"})).toBeEnabled();
 await user.click(screen.getByRole("button",{name:"保存"}));await waitFor(()=>expect(api.save).toHaveBeenCalledOnce());
 expect(api.save.mock.calls[0][2]).toBe(key);expect(api.save.mock.calls[0][3]).toEqual({modules:["members","store-browser"],expectedVersion:1});
});
it("saves only available modules with the displayed role revision",async()=>{
 api.save.mockResolvedValue({role:{...role,modules:[],version:2}});mount();const user=userEvent.setup();
 await user.click(await screen.findByRole("button",{name:"客服"}));await user.click(screen.getByText("我的账户",{selector:"strong"}));await user.click(screen.getByRole("button",{name:"移除成员与权限"}));
 await user.click(screen.getByRole("button",{name:"保存"}));await waitFor(()=>expect(api.save).toHaveBeenCalledOnce());expect(api.save.mock.calls[0][2]).toBe(key);expect(api.save.mock.calls[0][3]).toEqual({modules:[],expectedVersion:1});
});
it("retains the exact creation key and payload after a lost response",async()=>{
 api.create.mockRejectedValueOnce(new Error("response lost")).mockResolvedValueOnce({role});mount();const user=userEvent.setup();
 await user.click(await screen.findByRole("button",{name:"＋ 新建角色"}));await user.type(screen.getByRole("textbox",{name:/角色名称/}),"客服");await user.click(screen.getByRole("button",{name:"创建"}));
 await screen.findByRole("button",{name:"核实原保存"});expect(screen.queryByRole("dialog",{name:"新建角色"})).not.toBeInTheDocument();
 await user.click(screen.getByRole("button",{name:"核实原保存"}));await waitFor(()=>expect(api.create).toHaveBeenCalledTimes(2));expect(api.create.mock.calls[1][1]).toBe(api.create.mock.calls[0][1]);expect(api.create.mock.calls[1][2]).toEqual(api.create.mock.calls[0][2]);expect(api.create.mock.calls[1][2]).toEqual({name:"客服",modules:[]});
});
