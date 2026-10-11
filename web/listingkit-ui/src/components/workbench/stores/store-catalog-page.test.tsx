import {render,screen} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,expect,it,vi} from "vitest";
const state=vi.hoisted(()=>({push:vi.fn(),search:"",filters:vi.fn(),data:{items:[],pagination:{page:1,pageSize:20,total:0},metrics:{total:0,normal:0,attention:0,syncHealthy:0,unknown:0,complete:true,observedAt:"2026-10-11T00:00:00Z"},facets:{regions:["US"],applicationTypes:[],groups:[],canManageGroups:false}}}));
vi.mock("next/navigation",()=>({usePathname:()=>"/workbench/stores",useRouter:()=>({push:state.push}),useSearchParams:()=>new URLSearchParams(state.search)}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({user:{id:"actor"},effectiveOrganization:{id:"org-a",name:"A"},permissions:["workbench.store.read"],isSwitching:false})}));
vi.mock("@/lib/query/use-store-catalog",()=>({useStoreCatalog:(filters:unknown)=>{state.filters(filters);return {data:state.data,isPending:false,isError:false,refetch:vi.fn()}}}));
vi.mock("@tanstack/react-query",()=>({useQueryClient:()=>({invalidateQueries:vi.fn()}),useQuery:()=>({})}));
import {StoreListPage} from "./store-list-page";
afterEach(()=>{state.push.mockClear();state.filters.mockClear();state.search=""});
it("shows four factual overviews and applies search to the server query",async()=>{
 render(<StoreListPage/>);expect(screen.getByText("店铺总数")).toBeInTheDocument();expect(screen.getByText("正常店铺")).toBeInTheDocument();expect(screen.getAllByText("需要处理")[0]).toBeInTheDocument();expect(screen.getByText("数据同步正常")).toBeInTheDocument();
 const user=userEvent.setup();await user.type(screen.getByRole("searchbox",{name:"搜索店铺"}),"名称 100%_");await user.click(screen.getByRole("button",{name:"筛选"}));
 expect(state.push).toHaveBeenCalledWith("/workbench/stores?page=1&pageSize=20&q=%E5%90%8D%E7%A7%B0+100%25_");
 expect(screen.queryByRole("button",{name:"绑定新店铺"})).not.toBeInTheDocument();
});
