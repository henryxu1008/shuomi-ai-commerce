import {cleanup,render,screen,within} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {OwnProductHistoryPage} from "./own-product-history-page";
const state=vi.hoisted(()=>({list:vi.fn(),read:vi.fn(),context:{} as Record<string,unknown>}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
vi.mock("@/lib/api/product-collection",async original=>({...await original<object>(),listOwnHistory:state.list,readOwnVersion:state.read}));
const id="550e8400-e29b-41d4-a716-446655440002";
const item={id,batchId:id,revision:2,createdAt:"2026-10-14T00:00:00Z",source:{kind:"own",productKey:"own-test",publicationId:id,version:"3"}};
const entry=(version:string)=>({version,publicationId:id,title:`资料${version}`,savedAt:"2026-10-14T00:00:00Z"});
beforeEach(()=>{state.list.mockReset();state.read.mockReset();state.context={user:{id:"actor-a"},effectiveOrganization:{id:"org-a"},permissions:["workbench.collection.read"]}});afterEach(cleanup);
it("lists saved versions and navigates to independent old/detail compare pages without manage permission",async()=>{
 state.list.mockResolvedValue({item,versions:[entry("3"),entry("1")]});
 render(<OwnProductHistoryPage itemId={id}/>);
 await screen.findByText("资料3");expect(screen.getByRole("link",{name:"查看版本 1"})).toHaveAttribute("href",`/workbench/data/mine/own/${id}/history/1`);
 await userEvent.click(screen.getByRole("checkbox",{name:"选择版本 3"}));await userEvent.click(screen.getByRole("checkbox",{name:"选择版本 1"}));
 expect(screen.getByRole("link",{name:"对比所选版本"})).toHaveAttribute("href",`/workbench/data/mine/own/${id}/history/compare?from=1&to=3`);
 expect(screen.queryByRole("dialog")).not.toBeInTheDocument();expect(screen.queryByRole("button",{name:"保存新版本"})).not.toBeInTheDocument();
});
it("shows immutable old data and zero values without an old-version edit action",async()=>{
 state.read.mockResolvedValue({item,version:entry("1"),product:{title:"资料1",variants:[{source_id:"red",sku:"RED",price:{currency:"CNY",amount:0},stock:0}]}});
 render(<OwnProductHistoryPage itemId={id} version="1"/>);
 await screen.findByRole("heading",{name:/历史版本 1/});
 expect(within(screen.getByRole("region",{name:"选中SKU资料"})).getByText("CNY 0")).toBeVisible();
 expect(screen.queryByRole("link",{name:"编辑商品"})).not.toBeInTheDocument();
});
it("does not display a delayed history result from a previous enterprise",async()=>{
 let release:(value:unknown)=>void=()=>{};
 state.list.mockImplementationOnce(()=>new Promise(resolve=>{release=resolve})).mockResolvedValue({item,versions:[{...entry("3"),title:"当前企业资料"}]});
 const view=render(<OwnProductHistoryPage itemId={id}/>);
 state.context={...state.context,effectiveOrganization:{id:"org-b"}};view.rerender(<OwnProductHistoryPage itemId={id}/>);
 await screen.findByText("当前企业资料");
 release({item,versions:[{...entry("3"),title:"旧企业私有资料"}]});
 expect(screen.queryByText("旧企业私有资料")).not.toBeInTheDocument();
 expect(state.list).toHaveBeenLastCalledWith({userId:"actor-a",organizationId:"org-b"},id,undefined,expect.any(AbortSignal));
});
