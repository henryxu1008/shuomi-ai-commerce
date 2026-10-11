import {render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {StoreCenterError,type CatalogItem,type Group} from "@/lib/api/store-center";
const state=vi.hoisted(()=>({request:vi.fn(),getStore:vi.fn(),refetch:vi.fn(),groups:[] as Group[]}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({permissions:["workbench.store.update"],roles:[]})}));
vi.mock("@tanstack/react-query",()=>({useQuery:()=>({data:{items:state.groups},refetch:state.refetch})}));
vi.mock("@/lib/api/store-center",async(importOriginal)=>({...await importOriginal<typeof import("@/lib/api/store-center")>(),storeCenterRequest:state.request}));
vi.mock("@/lib/api/workbench-stores",async(importOriginal)=>({...await importOriginal<typeof import("@/lib/api/workbench-stores")>(),getWorkbenchStore:state.getStore}));
vi.mock("./store-form",()=>({StoreForm:({mode,store,conflict,onConflict,onPendingChange,onCreated}:{mode:string;store:CatalogItem;conflict?:{latest:CatalogItem;changedFields:string[]};onConflict?: (draft:unknown,base:CatalogItem)=>void;onPendingChange?:(busy:boolean)=>void;onCreated?:(store:CatalogItem)=>void})=>mode==="create"?<><button onClick={()=>onPendingChange?.(true)}>开始记录创建</button><button onClick={()=>onCreated?.(item)}>返回已创建店铺</button></>:<><button onClick={()=>onConflict?.({name:"保留草稿"},store)}>触发名称冲突</button>{conflict?<p>最新版本 {conflict.latest.version}：{conflict.changedFields.join(",")}</p>:null}</>}));
vi.mock("./store-official-connection",()=>({StoreOfficialConnection:()=> <p>官方授权入口</p>}));
import {StoreCatalogDialogs} from "./store-catalog-dialogs";
const scope={organizationId:"org-a",userId:"actor"};
const item={id:"11111111-1111-4111-8111-111111111111",name:"A 店铺",region:"US",version:1,recordStatus:"active",profile:{storeId:"11111111-1111-4111-8111-111111111111",profileVersion:0,groupId:null,responsibleMemberId:null,lowStockThreshold:null,updatedAt:null}} as CatalogItem;
beforeEach(()=>{state.groups=[];sessionStorage.clear();state.refetch.mockResolvedValue({data:{items:[]}});HTMLDialogElement.prototype.showModal=function(){this.setAttribute("open","")};HTMLDialogElement.prototype.close=function(){this.removeAttribute("open")}});
afterEach(()=>vi.clearAllMocks());
it("rereads the Store after a rename conflict and passes the current CAS version to the retained form",async()=>{
 state.getStore.mockResolvedValue({...item,name:"他人名称",version:2});render(<StoreCatalogDialogs dialog={{kind:"rename",item}} scope={scope} groups={[]} canManageGroups={false} onClose={vi.fn()} onChanged={vi.fn()}/>);
 await userEvent.click(screen.getByRole("button",{name:"触发名称冲突"}));expect(await screen.findByText("最新版本 2：name")).toBeInTheDocument();expect(state.getStore).toHaveBeenCalledWith(item.id,"org-a");
});
it("unlocks the binding dialog after record creation completes and the form unmounts",async()=>{
 render(<StoreCatalogDialogs dialog={{kind:"bind"}} scope={scope} groups={[]} canManageGroups={false} onClose={vi.fn()} onChanged={vi.fn().mockResolvedValue(null)}/>);
 await userEvent.click(screen.getByRole("button",{name:"开始记录创建"}));expect(screen.getByRole("button",{name:"关闭"})).toBeDisabled();await userEvent.click(screen.getByRole("button",{name:"返回已创建店铺"}));expect(screen.getByText("官方授权入口")).toBeInTheDocument();expect(screen.getByRole("button",{name:"关闭"})).toBeEnabled();
});
it("refreshes and closes a rejected profile intent before another edit can capture current versions",async()=>{
 state.request.mockRejectedValue(new StoreCenterError(409,"STORE_VERSION_CONFLICT"));const close=vi.fn(),changed=vi.fn().mockResolvedValue(null);
 render(<StoreCatalogDialogs dialog={{kind:"profile",item}} scope={scope} groups={[]} canManageGroups={false} onClose={close} onChanged={changed}/>);
 await waitFor(()=>expect(screen.getByRole("button",{name:"保存设置"})).toBeEnabled());await userEvent.click(screen.getByRole("button",{name:"保存设置"}));
 expect(await screen.findByText(/操作被拒绝/)).toBeInTheDocument();expect(sessionStorage.getItem("store-metadata:org-a:actor")).not.toBeNull();
 await userEvent.click(screen.getByRole("button",{name:"刷新列表并关闭"}));await waitFor(()=>expect(close).toHaveBeenCalledOnce());expect(changed).toHaveBeenCalledOnce();expect(state.refetch).toHaveBeenCalledOnce();expect(sessionStorage.getItem("store-metadata:org-a:actor")).toBeNull();
});

it("shows authorized catalog group counts and keeps absent counts unknown",async()=>{
 const known={id:"22222222-2222-4222-8222-222222222222",name:"核心店铺",groupVersion:3,updatedAt:"2026-10-11T00:00:00Z"},unknown={...known,id:"33333333-3333-4333-8333-333333333333",name:"新分组"};state.groups=[known,unknown];
 render(<StoreCatalogDialogs dialog={{kind:"groups"}} scope={scope} groups={[{...known,count:42}]} canManageGroups={true} onClose={vi.fn()} onChanged={vi.fn()}/>);
 expect(await screen.findByText("42 家店铺")).toBeInTheDocument();expect(screen.getByText("数量待确认")).toBeInTheDocument();expect(screen.queryByText("版本 3")).not.toBeInTheDocument();
});
