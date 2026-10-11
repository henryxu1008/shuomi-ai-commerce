import {render,screen,within} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,expect,it,vi} from "vitest";
import type {CatalogItem} from "@/lib/api/store-center";
const state=vi.hoisted(()=>({summary:{} as Record<string,unknown>}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({user:{id:"actor"},effectiveOrganization:{id:"org-a"},permissions:["workbench.store.read","workbench.store.update"],roles:[]})}));
vi.mock("@tanstack/react-query",()=>({useQueryClient:()=>({invalidateQueries:vi.fn()}),useQuery:({queryKey}:{queryKey:string[]})=>queryKey[0]==="store-summary"?state.summary:{data:{items:[]}}}));
vi.mock("./store-catalog-dialogs",()=>({StoreCatalogDialogs:()=> <div role="dialog">归属设置</div>}));
import {StoreDetailPresentation} from "./store-detail-presentation";
const metric={value:0,complete:true,unknown:0,reason:""};
const item={id:"11111111-1111-4111-8111-111111111111",name:"A 店铺",region:"US",platform:"shein",version:1,recordStatus:"active",serviceStatus:"pending_activation",connectionStatus:"disconnected",profile:{storeId:"11111111-1111-4111-8111-111111111111",profileVersion:0,groupId:null,responsibleMemberId:null,lowStockThreshold:null,updatedAt:null},signals:{products:metric,orders:metric,inventory:metric,syncHealthy:false,observedAt:null},attention:"attention"} as CatalogItem;
afterEach(()=>{state.summary={};window.history.replaceState(null,"",window.location.pathname)});
it("distinguishes an unset threshold and incomplete inventory coverage in detail overviews",()=>{
 const props={store:item,canUpdate:true,onRefresh:vi.fn(),actions:null,editForm:null};state.summary={data:{...item,signals:{...item.signals,inventory:{value:null,complete:false,unknown:0,reason:"threshold_not_set"}}},isPending:false,isError:false};const view=render(<StoreDetailPresentation {...props}/>);
 expect(screen.getByText("未设置")).toBeInTheDocument();state.summary={data:{...item,signals:{...item.signals,inventory:{value:0,complete:false,unknown:2,reason:"inventory_unknown"}}},isPending:false,isError:false};view.rerender(<StoreDetailPresentation {...props}/>);expect(screen.getByText("另 2 项库存未知")).toBeInTheDocument();
});
it("hides stale profile facts and its open dialog after the Store summary access is revoked",async()=>{
 state.summary={data:item,isPending:false,isError:false};const props={store:item,canUpdate:true,onRefresh:vi.fn(),actions:null,editForm:null};const view=render(<StoreDetailPresentation {...props}/>);
 await userEvent.click(screen.getByRole("tab",{name:"店铺设置"}));await userEvent.click(screen.getByRole("button",{name:"修改归属与预警设置"}));expect(screen.getByRole("dialog")).toBeInTheDocument();
 state.summary={data:item,isPending:false,isError:true,error:{code:"PERMISSION_DENIED"}};view.rerender(<StoreDetailPresentation {...props}/>);expect(screen.queryByRole("dialog")).not.toBeInTheDocument();expect(screen.getByRole("button",{name:"修改归属与预警设置"})).toBeDisabled();expect(within(screen.getByText("分组与业务归属").closest("section")!).getAllByText("未知")).toHaveLength(3);expect(screen.queryByText("未设置")).not.toBeInTheDocument();
});
