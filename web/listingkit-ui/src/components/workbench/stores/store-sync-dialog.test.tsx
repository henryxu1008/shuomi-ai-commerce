import {render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import type {CatalogItem} from "@/lib/api/store-center";
const state=vi.hoisted(()=>({request:vi.fn(),admitted:true}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({storeObservationsAvailable:state.admitted,permissions:["workbench.store.read","workbench.store.update","workbench.store.products.read","workbench.store.products.sync","workbench.store.orders.read","workbench.store.orders.sync"]})}));
vi.mock("@/lib/api/store-observations",async(importOriginal)=>({...await importOriginal<typeof import("@/lib/api/store-observations")>(),observationRequest:state.request}));
import {StoreSyncDialog} from "./store-sync-dialog";
import {ObservationError} from "@/lib/api/store-observations";
const store={id:"11111111-1111-4111-8111-111111111111",name:"TEMU 店铺",platform:"temu"} as CatalogItem;
const scope={organizationId:"org-a",userId:"actor"};
beforeEach(()=>{
 sessionStorage.clear();state.request.mockReset();state.admitted=true;
 state.request.mockImplementation(async(path:string)=>{if(path.includes("/commands/"))throw Error("receipt pending");return path.endsWith("/capabilities")?{available:true,canSync:true,site:"configured"}:{syncs:[]}});
 HTMLDialogElement.prototype.showModal=function(){this.setAttribute("open","")};HTMLDialogElement.prototype.close=function(){this.removeAttribute("open")};
});
afterEach(()=>{sessionStorage.clear();vi.clearAllMocks()});
function show(initialKind:"products"|"orders"="products",platform="temu"){
 const client=new QueryClient({defaultOptions:{queries:{retry:false},mutations:{retry:false}}});
 const content=()=> <QueryClientProvider client={client}><StoreSyncDialog store={{...store,platform} as CatalogItem} scope={scope} initialKind={initialKind} onClose={vi.fn()} onChanged={vi.fn().mockResolvedValue(null)}/></QueryClientProvider>;
 const view=render(content());return {...view,rerenderDialog:()=>view.rerender(content())};
}
it("blocks new Walmart orders and explicitly releases inaccessible local tracking",async()=>{
 const view=show("products","walmart");
 await waitFor(()=>expect(screen.getByRole("button",{name:"开始同步"})).toBeEnabled());
 expect(screen.getByRole("option",{name:"订单"})).toBeDisabled();view.unmount();
 const key="22222222-2222-4222-8222-222222222222";
 sessionStorage.setItem("store-dialog-sync:org-a:actor",JSON.stringify({key,input:{kind:"orders",stores:[store.id]}}));
 state.request.mockImplementation(async(path:string)=>{if(path.endsWith("/capabilities"))return {available:true,canSync:true};throw new ObservationError("NOT_FOUND",404)});
 show("products","walmart");await userEvent.click(await screen.findByRole("button",{name:"清除此页同步记录"}));
 expect(sessionStorage.getItem("store-dialog-sync:org-a:actor")).toBeNull();
 expect(screen.getByRole("combobox",{name:"同步数据"})).toHaveValue("products");expect(state.request.mock.calls.some(([path])=>path.endsWith("/syncs"))).toBe(false);
});
it("starts TEMU products with the retained command and blocks a new TEMU order intent",async()=>{
 const view=show();await waitFor(()=>expect(screen.getByRole("button",{name:"开始同步"})).toBeEnabled());
 expect(screen.getByRole("option",{name:"订单"})).toBeDisabled();
 await userEvent.click(screen.getByRole("button",{name:"开始同步"}));
 await waitFor(()=>expect(state.request).toHaveBeenCalledWith("products/syncs",scope,expect.anything(),expect.anything(),expect.objectContaining({input:{kind:"products",stores:[store.id]}})));
 view.unmount();sessionStorage.clear();state.request.mockClear();show("orders");
 await waitFor(()=>expect(state.request).toHaveBeenCalledWith("orders/capabilities",scope,expect.anything(),expect.anything()));
 expect(screen.getByRole("button",{name:"开始同步"})).toBeDisabled();
 await userEvent.click(screen.getByRole("button",{name:"开始同步"}));
 expect(state.request.mock.calls.some(([path])=>path==="orders/syncs")).toBe(false);
});
it("keeps an original other-store order command unchanged when reopened from TEMU",async()=>{
 const key="22222222-2222-4222-8222-222222222222",other="33333333-3333-4333-8333-333333333333";
 sessionStorage.setItem("store-dialog-sync:org-a:actor",JSON.stringify({key,input:{kind:"orders",stores:[other]}}));show();
 await waitFor(()=>expect(screen.getByRole("button",{name:"重试原同步请求"})).toBeEnabled());
 expect(screen.queryByRole("link",{name:/查看本店授权信息/})).not.toBeInTheDocument();
 await userEvent.click(screen.getByRole("button",{name:"重试原同步请求"}));
 await waitFor(()=>expect(state.request).toHaveBeenCalledWith("orders/syncs",scope,expect.anything(),expect.anything(),{key,input:{kind:"orders",stores:[other]}}));
});
it("explains provider preparation without replacing the original request or dispatching sync",async()=>{
 const key="22222222-2222-4222-8222-222222222222",saved=JSON.stringify({key,input:{kind:"products",stores:[store.id]}});
 sessionStorage.setItem("store-dialog-sync:org-a:actor",saved);
 state.request.mockResolvedValue({available:false,canSync:false,runtimeReady:true,providerConfigured:false});
 show("products","shein");
 expect(await screen.findByText("平台读取：待配置")).toBeVisible();
 expect(screen.getByText("内部运行：已就绪")).toBeVisible();
 expect(screen.getByRole("button",{name:"重试原同步请求"})).toBeDisabled();
 expect(sessionStorage.getItem("store-dialog-sync:org-a:actor")).toBe(saved);
 expect(state.request.mock.calls.every(([path])=>path.endsWith("/capabilities"))).toBe(true);
});
it("removes old ready facts when the dialog loses runtime admission",async()=>{
 state.request.mockResolvedValue({available:true,canSync:true,runtimeReady:true,providerConfigured:true});
 const view=show();expect(await screen.findByText("内部运行：已就绪")).toBeVisible();
 state.admitted=false;view.rerenderDialog();
 expect(screen.getByText("内部运行：待确认")).toBeVisible();
 expect(screen.getByRole("button",{name:"开始同步"})).toBeDisabled();
 expect(state.request).toHaveBeenCalledTimes(1);
});
