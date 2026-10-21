import {render,screen,waitFor} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {afterEach,expect,it,vi} from "vitest";
import type {CatalogItem} from "@/lib/api/store-center";
const state=vi.hoisted(()=>({capabilities:vi.fn(),summary:vi.fn()}));
vi.mock("@/lib/api/store-observations",async original=>({...await original<typeof import("@/lib/api/store-observations")>(),observationRequest:state.capabilities}));
vi.mock("@/lib/api/store-center",async original=>({...await original<typeof import("@/lib/api/store-center")>(),storeCenterRequest:state.summary}));
import {SelectedStoreReadiness,StoreReadinessNotice,useStoreReadiness} from "./store-readiness";
const id="11111111-1111-4111-8111-111111111111";
const store={id,name:"试用店铺",connectionStatus:"disconnected",serviceStatus:"pending_activation"} as CatalogItem;
const permissions=["workbench.store.read","workbench.store.update","workbench.store.products.read","workbench.store.products.sync"];
const pending={available:false,canSync:false,runtimeReady:true,providerConfigured:false,kind:"products" as const,site:"shein-us" as const,platformUrl:"https://sellerhub.shein.com/" as const};
afterEach(()=>vi.resetAllMocks());
it("keeps global capability separate from disconnected store and inactive service",()=>{
 render(<StoreReadinessNotice permissions={permissions} store={store} caps={{data:{...pending,available:true,canSync:true,providerConfigured:true}}}/>);
 expect(screen.getByText("平台读取：已配置")).toBeVisible();
 expect(screen.getByText(/本店连接：未连接 · 店铺服务：待激活/)).toBeVisible();
 expect(screen.getByRole("link",{name:/查看本店授权信息/})).toHaveAttribute("href",`/workbench/stores/${id}#authorization`);
 expect(screen.queryByText("可以同步")).not.toBeInTheDocument();
});
it.each([{isError:true},{isFetching:true},{isPending:true}])("hides stale capability facts during error or loading %j",flags=>{
 render(<StoreReadinessNotice permissions={permissions} caps={{data:pending,...flags}}/>);
 expect(screen.getByText("平台读取：待确认")).toBeVisible();
 expect(screen.getByText("内部运行：待确认")).toBeVisible();
 expect(screen.queryByText("平台读取：待配置")).not.toBeInTheDocument();
});
it("explains member permissions without offering a privileged connection action",()=>{
 render(<StoreReadinessNotice permissions={["workbench.store.read"]} store={store} caps={{}}/>);
 expect(screen.getByText("商品读取权限：未开放")).toBeVisible();
 expect(screen.getByText(/请联系企业配置管理员/)).toBeVisible();
 expect(screen.queryByRole("link",{name:/查看本店授权信息/})).not.toBeInTheDocument();
});
function Harness({organizationId="org-a",allowed=permissions,storeId=id,admitted=true}:{organizationId?:string;allowed?:string[];storeId?:string;admitted?:boolean}){
 const scope={organizationId,userId:"actor"};
 const caps=useStoreReadiness(scope,allowed,admitted);
 return <SelectedStoreReadiness storeId={storeId} scope={scope} permissions={allowed} caps={caps} kind="products"/>;
}
function mount(props:Parameters<typeof Harness>[0]={}){
 const client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 const wrap=(p:Parameters<typeof Harness>[0])=><QueryClientProvider client={client}><Harness {...p}/></QueryClientProvider>;
 return {view:render(wrap(props)),wrap};
}
it("does not request module capabilities without read permission",async()=>{
 state.summary.mockResolvedValue(store);
 mount({allowed:["workbench.store.read"]});
 await waitFor(()=>expect(state.summary).toHaveBeenCalled());
 expect(state.capabilities).not.toHaveBeenCalled();
});
it("retains the selected readable store and clears its facts on organization switch",async()=>{
 state.capabilities.mockResolvedValue(pending);
 state.summary.mockImplementation((_path,scope)=>scope.organizationId==="org-a"?Promise.resolve(store):new Promise(()=>{}));
 const {view,wrap}=mount();
 expect(await screen.findByRole("link",{name:/查看本店授权信息/})).toHaveAttribute("href",`/workbench/stores/${id}#authorization`);
 expect(state.summary).toHaveBeenCalledWith(`summaries/${id}`,{organizationId:"org-a",userId:"actor"},expect.anything(),expect.anything());
 view.rerender(wrap({organizationId:"org-b"}));
 expect(screen.queryByRole("link",{name:/查看本店授权信息/})).not.toBeInTheDocument();
 expect(screen.queryByText(/本店连接：未连接/)).not.toBeInTheDocument();
});
it("does not trust an invalid or inaccessible query-string store selection",async()=>{
 state.capabilities.mockResolvedValue(pending);state.summary.mockRejectedValue(new Error("forbidden"));
 const {view,wrap}=mount({storeId:"not-a-store"});
 await waitFor(()=>expect(state.capabilities).toHaveBeenCalled());
 expect(state.summary).not.toHaveBeenCalled();
 view.rerender(wrap({storeId:id}));
 await waitFor(()=>expect(state.summary).toHaveBeenCalled());
 expect(screen.queryByRole("link",{name:/查看本店授权信息/})).not.toBeInTheDocument();
 expect(screen.getByRole("link",{name:/前往我的店铺/})).toHaveAttribute("href","/workbench/stores");
});
it("hides a previously ready capability when runtime admission closes",async()=>{
 state.capabilities.mockResolvedValue({...pending,available:true,canSync:true,providerConfigured:true});state.summary.mockResolvedValue(store);
 const {view,wrap}=mount();
 expect(await screen.findByText("内部运行：已就绪")).toBeVisible();
 view.rerender(wrap({admitted:false}));
 expect(screen.getByText("内部运行：待确认")).toBeVisible();
 expect(screen.getByText("平台读取：待确认")).toBeVisible();
 expect(screen.getByText(/核对同步运行配置与当前权限/)).toBeVisible();
 expect(state.capabilities).toHaveBeenCalledTimes(1);
});
it("directs a connected active store to runtime maintenance when runtime is unavailable",()=>{
 render(<StoreReadinessNotice permissions={permissions} store={{...store,connectionStatus:"connected",serviceStatus:"active"}} caps={{data:{...pending,providerConfigured:true,runtimeReady:false}}}/>);
 expect(screen.getByText("内部运行：暂不可用")).toBeVisible();
 expect(screen.getByText(/核对店铺同步运行配置和服务健康/)).toBeVisible();
 expect(screen.queryByText(/同步以执行入口的检查为准/)).not.toBeInTheDocument();
});
