import {render,screen} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,expect,it,vi} from "vitest";
const state=vi.hoisted(()=>({search:"",push:vi.fn(),retry:vi.fn(),switching:false,org:"org-a",permissions:["workbench.store.read"],query:{} as Record<string,unknown>,filters:vi.fn()}));
vi.mock("next/navigation",()=>({usePathname:()=>"/workbench/stores",useRouter:()=>({push:state.push}),useSearchParams:()=>new URLSearchParams(state.search)}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({user:{id:"actor"},effectiveOrganization:{id:state.org,name:"A"},permissions:state.permissions,isSwitching:state.switching,retry:state.retry})}));
vi.mock("@/lib/query/use-store-catalog",()=>({useStoreCatalog:(filters:unknown,valid:boolean)=>{state.filters(filters,valid);return state.query}}));
vi.mock("@tanstack/react-query",()=>({useQueryClient:()=>({invalidateQueries:vi.fn()}),useQuery:()=>({})}));
vi.mock("./store-catalog-dialogs",()=>({StoreCatalogDialogs:()=> <div role="dialog">归属设置</div>}));
vi.mock("./store-lifecycle-actions",()=>({StoreLifecycleActions:()=>null}));
import {StoreListPage} from "./store-list-page";
const metric={value:null,unknown:0,complete:false,reason:"unavailable"};
const item={id:"11111111-1111-4111-8111-111111111111",name:"A 店铺",platform:"shein",region:"US",externalStoreId:"",recordStatus:"active",serviceStatus:"pending_activation",connectionStatus:"disconnected",version:1,profile:{storeId:"11111111-1111-4111-8111-111111111111",groupId:null,responsibleMemberId:null,lowStockThreshold:null,profileVersion:0,updatedAt:null},attention:"attention",signals:{applicationType:"",products:metric,orders:metric,inventory:{...metric,reason:"threshold_not_set"},syncHealthy:false,productSync:"",orderSync:"",observedAt:null}};
const data={items:[item],pagination:{page:1,pageSize:20,total:1},metrics:{total:1,normal:null,attention:null,syncHealthy:null,unknown:1,complete:false},facets:{regions:["US"],groups:[],applicationTypes:[],canManageGroups:false}};
afterEach(()=>{state.query={};state.search="";state.switching=false;state.org="org-a";state.permissions=["workbench.store.read"];state.push.mockReset();state.filters.mockReset();state.retry.mockReset()});
it("uses server count and shows unknown facts without fake zero or write controls",()=>{
 state.query={data,isPending:false,isError:false};render(<StoreListPage/>);
 expect(screen.getByText("A 店铺")).toBeInTheDocument();expect(screen.getByText("第 1 页，共 1 家店铺")).toBeInTheDocument();expect(screen.getAllByText("未知").length).toBeGreaterThan(0);
 expect(screen.queryByRole("button",{name:"绑定新店铺"})).not.toBeInTheDocument();expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
 expect(screen.getByRole("link",{name:/进入店铺/})).toHaveAttribute("href",`/workbench/store-products?storeId=${item.id}`);
 expect(screen.getByRole("button",{name:"上一页"})).toBeDisabled();expect(screen.getByRole("button",{name:"下一页"})).toBeDisabled();
});
it.each(["ORGANIZATION_ACCESS_REVOKED","ORGANIZATION_CONTEXT_CHANGED","ORGANIZATION_ACCESS_DENIED"])("hides stale rows and metrics and refreshes context for %s",async(code)=>{
 state.query={data,isPending:false,isError:true,error:{code},refetch:vi.fn()};render(<StoreListPage/>);
 expect(screen.getByRole("alert")).toHaveTextContent("当前企业访问已不可用");expect(screen.queryByText("A 店铺")).not.toBeInTheDocument();expect(screen.queryByText(/共 1 家/)).not.toBeInTheDocument();
 await userEvent.click(screen.getByRole("button",{name:"重试"}));expect(state.retry).toHaveBeenCalledOnce();
});
it("hides prior organization results while switching",()=>{state.query={data,isPending:false,isError:false};const view=render(<StoreListPage/>);expect(screen.getByText("A 店铺")).toBeInTheDocument();state.switching=true;view.rerender(<StoreListPage/>);expect(screen.queryByText("A 店铺")).not.toBeInTheDocument();expect(screen.getByRole("status")).toHaveTextContent("正在加载企业权限")});
it("rejects duplicate URL predicates instead of showing results for a weaker query",()=>{state.search="attention=normal&attention=attention";state.query={data,isPending:false,isError:false};render(<StoreListPage/>);expect(state.filters).toHaveBeenCalledWith({page:1,pageSize:20},false);expect(screen.getByRole("alert")).toHaveTextContent("筛选参数无效");expect(screen.queryByText("A 店铺")).not.toBeInTheDocument()});
it.each(["PERMISSION_DENIED","STORE_NOT_FOUND","ORGANIZATION_CONTEXT_CHANGED","ORGANIZATION_ACCESS_REVOKED","ORGANIZATION_ACCESS_DENIED","IDENTITY_CONTEXT_CHANGED","AUTHENTICATION_REQUIRED","ORGANIZATION_SUSPENDED"])("clears selected stores and their dialog when scope fails with %s",async(code)=>{
 state.permissions=["workbench.store.read","workbench.store.update"];state.query={data,isPending:false,isError:false};const view=render(<StoreListPage/>);
 await userEvent.click(screen.getByRole("checkbox",{name:"选择A 店铺"}));await userEvent.click(screen.getByRole("button",{name:"加入分组"}));expect(screen.getByRole("dialog")).toBeInTheDocument();
 state.query={data,isPending:false,isError:true,error:{code}};view.rerender(<StoreListPage/>);expect(screen.queryByRole("dialog")).not.toBeInTheDocument();expect(screen.queryByText("A 店铺")).not.toBeInTheDocument();
 state.query={data,isPending:false,isError:false};view.rerender(<StoreListPage/>);expect(screen.getByRole("checkbox",{name:"选择A 店铺"})).not.toBeChecked();
});
