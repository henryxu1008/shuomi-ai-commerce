import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {OwnProductDrafts} from "./own-product-drafts";
const f=vi.hoisted(()=>({list:vi.fn()}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),listItemCompletionDrafts:f.list}));
const item="11111111-1111-4111-8111-111111111111",prep="22222222-2222-4222-8222-222222222222",source="33333333-3333-4333-8333-333333333333";
const scope={userId:"actor",organizationId:"org"};
const record={id:source,draftId:source,revision:3,source:{id:source,preparationId:prep,collectionItemId:item,source:{kind:"own",version:"1"}},content:{title:"已保存候选标题"},createdAt:"2026-10-14T00:00:00Z"};
beforeEach(()=>{f.list.mockReset();f.list.mockResolvedValue({collectionItemId:item,items:[record],total:2,nextCursor:source})});
afterEach(cleanup);
it("ignores a late prior enterprise list after remounting",async()=>{
 let finish!:(value:unknown)=>void;f.list.mockReturnValueOnce(new Promise(resolve=>{finish=resolve}));
 const view=render(<OwnProductDrafts scope={scope} itemId={item} canManage/>);await waitFor(()=>expect(f.list).toHaveBeenCalledOnce());
 f.list.mockResolvedValue({collectionItemId:item,items:[],total:0});view.rerender(<OwnProductDrafts scope={{...scope,organizationId:"new-org"}} itemId={item} canManage/>);await screen.findByText("此商品还没有已保存的待补全草稿。");
 finish({collectionItemId:item,items:[record],total:1});await waitFor(()=>expect(screen.queryByText(record.content.title)).not.toBeInTheDocument());
});
it("distinguishes Vn and Rn and continues the same source with item and preparation context",async()=>{
 render(<OwnProductDrafts scope={scope} itemId={item} canManage/>);
 expect(await screen.findByText("已保存候选标题")).toBeVisible();expect(screen.getByText("V1")).toBeVisible();expect(screen.getByText("R3")).toBeVisible();
 expect(screen.getByRole("link",{name:"继续编辑"})).toHaveAttribute("href",`/workbench/supply/mine/missing/${source}?preparation=${prep}&item=${item}`);
 f.list.mockResolvedValue({collectionItemId:item,items:[],total:2});fireEvent.click(screen.getByRole("button",{name:"下一页草稿"}));await waitFor(()=>expect(f.list).toHaveBeenLastCalledWith(scope,item,{limit:20,after:source},expect.any(AbortSignal)));
});
it("keeps read-only discovery and reports errors without inventing zero drafts",async()=>{
 const view=render(<OwnProductDrafts scope={scope} itemId={item} canManage={false}/>);expect(await screen.findByRole("link",{name:"查看草稿"})).toBeVisible();expect(screen.queryByRole("link",{name:"继续编辑"})).not.toBeInTheDocument();
 f.list.mockRejectedValue(new Error("offline"));view.rerender(<OwnProductDrafts scope={{...scope,organizationId:"new-org"}} itemId={item} canManage={false}/>);
 expect(await screen.findByText("已有草稿读取失败")).toBeVisible();expect(screen.queryByText("已保存候选标题")).not.toBeInTheDocument();expect(screen.queryByText(/共 0/)).not.toBeInTheDocument();
 f.list.mockResolvedValue({collectionItemId:item,items:[],total:0});fireEvent.click(screen.getByRole("button",{name:"刷新已有草稿"}));expect(await screen.findByText("此商品还没有已保存的待补全草稿。" )).toBeVisible();
});
