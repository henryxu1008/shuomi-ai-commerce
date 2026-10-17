import {cleanup,render,screen,waitFor} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {afterEach,expect,it,vi} from "vitest";
import MineRoute from "@/app/workbench/services/mine/page";
import {ecoRequest} from "@/lib/api/ecoservices";
const context=vi.hoisted(()=>({user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.ecoservices.read","workbench.ecoservices.manage"]}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>context}));
vi.mock("@/lib/server/ecoservices-availability",()=>({isEcoservicesAvailable:()=>true,isEcoservicesNonPaymentOnly:()=>false}));
vi.mock("@/lib/api/ecoservices",async original=>({...await original<object>(),ecoRequest:vi.fn()}));
afterEach(()=>{cleanup();vi.clearAllMocks()});
it.each([["provider",true,"provider"],["provider",false,"buyer"],[undefined,true,"buyer"],["invalid",true,"buyer"]] as const)("resolves side=%s with management=%s to %s",async(side,manage,expected)=>{
 context.permissions=["workbench.ecoservices.read",...(manage?["workbench.ecoservices.manage"]:[])];vi.mocked(ecoRequest).mockResolvedValue({requests:[],total:"0"});
 const child=await MineRoute({searchParams:Promise.resolve({side})});const client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 render(<QueryClientProvider client={client}>{child}</QueryClientProvider>);
 await waitFor(()=>expect(vi.mocked(ecoRequest).mock.calls.some(([,path])=>new URL(path,"https://test.invalid").searchParams.get("side")===expected)).toBe(true));
 expect(screen.getByLabelText("当前身份")).toHaveValue(expected);client.clear();
});
