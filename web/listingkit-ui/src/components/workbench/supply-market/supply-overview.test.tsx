import {cleanup,render,screen,waitFor} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import SupplyPage from "@/app/workbench/supply/page";

const fixture=vi.hoisted(()=>({redirect:vi.fn(),read:vi.fn(),context:{user:{id:"user-a"},effectiveOrganization:{id:"org-a"},isLoading:false,isSwitching:false,selectionRequired:false,error:null,blockingError:null}}));
vi.mock("next/navigation",()=>({redirect:fixture.redirect}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>fixture.context}));
vi.mock("@/lib/api/supply-market",()=>({readApplicationOverview:fixture.read}));
beforeEach(()=>{
 fixture.redirect.mockReset().mockImplementation(()=>{throw new Error("unexpected supply redirect")});
 fixture.read.mockReset().mockResolvedValue({eligibleProducts:22,reviewing:7,supplementRequired:3,approved:9,products:[]});
 fixture.context.isSwitching=false;
 for(const flag of ["LISTINGKIT_PRODUCT_ACQUISITION_ENABLED","LISTINGKIT_SUPPLY_MARKET_ENABLED","LISTINGKIT_SUPPLY_CHAIN_ENABLED","LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED","LISTINGKIT_ACQUISITION_PLANS_ENABLED"])vi.stubEnv(flag,"true");
});
afterEach(()=>{cleanup();vi.unstubAllEnvs()});
function renderPage(){const client=new QueryClient({defaultOptions:{queries:{retry:false}}});return {...render(<QueryClientProvider client={client}><SupplyPage/></QueryClientProvider>),client}}

it("keeps Supply Market as its own Figma page instead of redirecting to capture",()=>{
 renderPage();
 expect(fixture.redirect).not.toHaveBeenCalled();
 expect(screen.getByRole("heading",{level:1,name:"让每一种货源，都能进入自己的供应链"})).toBeVisible();
 expect(screen.getByRole("list",{name:"从货源到店铺"}).children).toHaveLength(4);
 for(const title of ["硕米自营","硕米优选","货盘集成","我的供应链","优选申请"])expect(screen.getByRole("heading",{level:2,name:title})).toBeVisible();
 expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});
it("uses only exact current summary facts and leaves unavailable counts as dashes",async()=>{
 renderPage();
 await waitFor(()=>expect(screen.getByLabelText("审核中数量")).toHaveTextContent("7"));
 expect(screen.getByLabelText("已通过数量")).toHaveTextContent("9");
 for(const label of ["商品总数","待完善","可刊登","官方接入货盘","可选商品","待提交"])expect(screen.getByLabelText(label+"数量")).toHaveTextContent("—");
 expect(fixture.read).toHaveBeenCalledWith({userId:"user-a",organizationId:"org-a"},expect.any(AbortSignal));
});
it("retains the introduction while disconnected cards expose no fake usable links",()=>{
 for(const flag of ["LISTINGKIT_PRODUCT_ACQUISITION_ENABLED","LISTINGKIT_SUPPLY_MARKET_ENABLED","LISTINGKIT_SUPPLY_CHAIN_ENABLED","LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED","LISTINGKIT_ACQUISITION_PLANS_ENABLED"])vi.stubEnv(flag,"false");
 renderPage();
 expect(screen.getByRole("heading",{level:1,name:"让每一种货源，都能进入自己的供应链"})).toBeVisible();
 expect(screen.queryByRole("link",{name:/进入功能/})).not.toBeInTheDocument();
 expect(fixture.read).not.toHaveBeenCalled();
});
it("hides the old enterprise summary as soon as an enterprise switch begins",async()=>{
 const view=renderPage();await waitFor(()=>expect(screen.getByLabelText("审核中数量")).toHaveTextContent("7"));
 fixture.context.isSwitching=true;
 view.rerender(<QueryClientProvider client={view.client}><SupplyPage/></QueryClientProvider>);
 expect(screen.getByLabelText("审核中数量")).toHaveTextContent("—");
 expect(screen.getByLabelText("已通过数量")).toHaveTextContent("—");
});
it("retains the existing Collections prerequisite for the Preparation link",()=>{
 vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED","false");
 renderPage();
 expect(screen.queryByRole("link",{name:"进入功能：我的供应链"})).not.toBeInTheDocument();
 expect(screen.getByRole("link",{name:"进入功能：货盘集成"})).toBeVisible();
});
