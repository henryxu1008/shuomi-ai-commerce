import {cleanup,render,screen,within} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {OwnProductDetailPage} from "./own-product-detail-page";
const state=vi.hoisted(()=>({read:vi.fn(),drafts:vi.fn(),context:{} as Record<string,unknown>}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
vi.mock("@/lib/api/product-collection",async original=>({...await original<object>(),readCollectionItem:state.read}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),listItemCompletionDrafts:state.drafts}));
const itemId="550e8400-e29b-41d4-a716-446655440002";
beforeEach(()=>{state.read.mockReset();state.drafts.mockResolvedValue({items:[],total:0});state.context={user:{id:"actor-a"},effectiveOrganization:{id:"org-a"},permissions:["workbench.collection.manage"]}});
afterEach(cleanup);
it("offers content optimization only for an available runtime and current permissions",async()=>{
 state.context.permissions=["workbench.collection.read","local_agent.write","workbench.agent.use"];
 state.read.mockResolvedValue({item:{id:itemId,source:{kind:"own",version:"1",productKey:"own-test"}},product:{title:"商品"}});
 const view=render(<OwnProductDetailPage itemId={itemId} optimizationAvailable/>);
 expect(await screen.findByRole("link",{name:"优化商品内容"})).toHaveAttribute("href",`/workbench/data/mine/own/${itemId}/optimize`);
 state.context.permissions=["workbench.collection.read"];
 view.rerender(<OwnProductDetailPage itemId={itemId} optimizationAvailable/>);
 expect(screen.queryByRole("link",{name:"优化商品内容"})).toBeNull();
});
it("opens the independent draft entry only when completion and current permissions are available",async()=>{
 state.context.permissions=["workbench.collection.read","workbench.collection.manage","workbench.supply.read","workbench.supply.manage"];
 state.read.mockResolvedValue({item:{id:itemId,source:{kind:"own",version:"1",productKey:"own-test"}},product:{title:"商品",variants:[],images:[]}});
 const view=render(<OwnProductDetailPage itemId={itemId} completionAvailable/>);
 expect(await screen.findByRole("link",{name:"创建待补全草稿"})).toHaveAttribute("href",`/workbench/data/mine/own/${itemId}/prepare`);
 expect(await screen.findByRole("region",{name:"已有待补全草稿"})).toBeVisible();
 view.rerender(<OwnProductDetailPage itemId={itemId}/>);expect(screen.queryByRole("region",{name:"已有待补全草稿"})).not.toBeInTheDocument();expect(screen.queryByRole("link",{name:"创建待补全草稿"})).not.toBeInTheDocument();
});
it("shows a saved SKU name even when it also has sales specifications",async()=>{
 state.read.mockResolvedValue({item:{id:itemId,source:{kind:"own",version:"1",productKey:"own-test"}},product:{title:"商品",variants:[{source_id:"red",sku:"RED",title:"红色常规款",attributes:[{name:"颜色",value:"红色"}],price:{currency:"CNY",amount:0},stock:0}]}});
 render(<OwnProductDetailPage itemId={itemId}/>);
 const region=await screen.findByRole("region",{name:"选中SKU资料"});
 expect(within(region).getByText("红色常规款")).toBeVisible();
});
it("shows common attributes and previews the selected SKU with its own image and declared values",async()=>{
 state.read.mockResolvedValue({item:{id:itemId,createdAt:"2026-10-13T12:00:00Z",source:{kind:"own",version:"1",productKey:"own-test"}},product:{title:"棉质上衣",brand:"测试品牌",description:"商品描述",attributes:[{name:"材质",value:"棉"},{name:"适用季节",value:"夏季"}],images:[{url:"https://images.example.org/main.png"}],variants:[{source_id:"red",sku:"RED",attributes:[{name:"颜色",value:"红色"}],price:{currency:"CNY",amount:12},stock:3,images:[{url:"https://images.example.org/red.png"}]},{source_id:"blue",sku:"BLUE",attributes:[{name:"颜色",value:"蓝色"}],price:{currency:"CNY",amount:0},stock:0,images:[{url:"https://images.example.org/blue.png"}]}]}});
 render(<OwnProductDetailPage itemId={itemId}/>);
 const attrs=await screen.findByRole("region",{name:"商品属性"});expect(within(attrs).getByText("材质")).toBeVisible();expect(within(attrs).getByText("棉")).toBeVisible();
 expect(screen.getByRole("link",{name:"编辑商品"})).toHaveAttribute("href",`/workbench/data/mine/own/${itemId}/edit`);
 await userEvent.selectOptions(screen.getByLabelText("当前SKU"),screen.getByRole("option",{name:"BLUE · 颜色：蓝色"}));
 expect(screen.getByRole("img",{name:"商品大图"})).toHaveAttribute("src",expect.stringContaining(encodeURIComponent("https://images.example.org/blue.png")));
 expect(within(screen.getByRole("region",{name:"选中SKU资料"})).getByText("CNY 0")).toBeVisible();
 expect(within(screen.getByRole("region",{name:"选中SKU资料"})).getByText("0 件")).toBeVisible();
 expect(screen.queryByRole("dialog")).not.toBeInTheDocument();expect(state.read).toHaveBeenCalledWith({userId:"actor-a",organizationId:"org-a"},itemId,expect.any(AbortSignal));
});
it("opens the exact variant thumbnail and keeps gallery navigation on the selected image",async()=>{
 const a="https://images.example.org/front.png",b="https://images.example.org/back.png",common="https://images.example.org/common.png";
 state.read.mockResolvedValue({item:{id:itemId,source:{kind:"own",version:"1",productKey:"own-test"}},product:{title:"商品",images:[{url:common}],variants:[{source_id:"red",sku:"RED",images:[{url:a},{url:b}],price:{currency:"CNY",amount:1},stock:2}]}});
 render(<OwnProductDetailPage itemId={itemId}/>);await screen.findByRole("img",{name:"商品大图"});
 await userEvent.click(screen.getByRole("button",{name:"查看RED图片2"}));
 expect(screen.getByRole("img",{name:"商品大图"})).toHaveAttribute("src",expect.stringContaining(encodeURIComponent(b)));
 await userEvent.click(screen.getByRole("button",{name:"上一张图片"}));
 expect(screen.getByRole("img",{name:"商品大图"})).toHaveAttribute("src",expect.stringContaining(encodeURIComponent(a)));
 await userEvent.click(screen.getByRole("button",{name:"查看RED图片2"}));await userEvent.click(screen.getByRole("button",{name:"下一张图片"}));
 expect(screen.getByRole("img",{name:"商品大图"})).toHaveAttribute("src",expect.stringContaining(encodeURIComponent(common)));
 expect(screen.getByRole("button",{name:"下一张图片"})).toBeDisabled();
});
