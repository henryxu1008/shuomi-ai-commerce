import {cleanup,render as renderView,screen,waitFor,fireEvent} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import type {ReactElement} from "react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {OwnProductOptimizePage} from "./own-product-optimize-page";
const state=vi.hoisted(()=>({read:vi.fn(),panel:vi.fn(),context:{} as Record<string,unknown>}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
vi.mock("@/lib/api/product-agent",async original=>({...await original<object>(),readOwnContentSource:state.read}));
vi.mock("../acquisition/product-agent-panel",()=>({ProductAgentPanel:(props:unknown)=>{state.panel(props);return <p>优化面板</p>}}));
beforeEach(()=>{state.read.mockReset();state.panel.mockReset();state.context={user:{id:"actor"},effectiveOrganization:{id:"org"}}});
afterEach(cleanup);
let client:QueryClient;
function render(ui:ReactElement){client=new QueryClient({defaultOptions:{queries:{staleTime:30000}}});return renderView(ui,{wrapper:({children})=><QueryClientProvider client={client}>{children}</QueryClientProvider>})}
const itemId="550e8400-e29b-41d4-a716-446655440002";
it("reads exact own facts without starting a model and links back to fresh choice",async()=>{
 state.read.mockResolvedValue({itemId,productKey:"own",originalVersion:2,catalogVersion:"3",publicationId:"applied",title:"已保存商品"});
 render(<OwnProductOptimizePage itemId={itemId} enabled/>);
 await screen.findByText("优化面板");
 expect(state.panel).toHaveBeenLastCalledWith(expect.objectContaining({operationId:itemId,productKey:"own",ownSource:{catalogVersion:"3",publicationId:"applied"}}));
 expect(screen.getByRole("link",{name:"检查优选资格"})).toHaveAttribute("href","/workbench/supply/applications/new");
});
it("hides old company source and ignores its late response after switching",async()=>{
 let resolve!:(value:unknown)=>void;state.read.mockImplementationOnce(()=>new Promise(r=>{resolve=r}));
 const view=render(<OwnProductOptimizePage itemId={itemId} enabled/>);
 state.context={user:{id:"actor"},effectiveOrganization:{id:"other"}};
 state.read.mockRejectedValueOnce(Error("unavailable"));
 view.rerender(<OwnProductOptimizePage itemId={itemId} enabled/>);
 resolve({itemId,productKey:"old",originalVersion:1,catalogVersion:"1",publicationId:"old",title:"旧企业商品"});
 await screen.findByText("当前商品或内容优化依赖不可用");
 await waitFor(()=>expect(state.read).toHaveBeenCalledTimes(2));
 expect(screen.queryByText("旧企业商品")).toBeNull();expect(state.panel).not.toHaveBeenCalled();
});
it("keeps an unavailable runtime explicit",()=>{
 render(<OwnProductOptimizePage itemId={itemId} enabled={false}/>);
 expect(screen.getByText("当前环境尚未开放内容优化")).toBeInTheDocument();expect(state.read).not.toHaveBeenCalled();
});
it("invalidates fresh eligibility caches for the current scope before returning to choice",async()=>{
 state.read.mockResolvedValue({itemId,productKey:"own",originalVersion:1,catalogVersion:"2",publicationId:"applied",title:"已采用商品"});
 render(<OwnProductOptimizePage itemId={itemId} enabled/>);await screen.findByText("优化面板");
 const keys=["application-products","choice","application-overview"].map(kind=>["supply-market","actor","org",kind]);
 for(const key of keys)client.setQueryData(key,{optimized:false});
 const other=["supply-market","actor","other","application-products"];client.setQueryData(other,{optimized:false});
 fireEvent.click(screen.getByRole("link",{name:"检查优选资格"}));
 for(const key of keys)expect(client.getQueryState(key)?.isInvalidated).toBe(true);
 expect(client.getQueryState(other)?.isInvalidated).toBe(false);
});
