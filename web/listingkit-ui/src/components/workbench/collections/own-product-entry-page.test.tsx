import {act,cleanup,render,screen,waitFor,within} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {OwnProductEntryPage} from "./own-product-entry-page";
import {OwnProductDetailPage} from "./own-product-detail-page";
import {OwnProductEditPage} from "./own-product-edit-page";
import {CollectionPage} from "./collection-page";
import {CollectionAPIError,type CollectionIntent} from "@/lib/api/product-collection";

const state=vi.hoisted(()=>({context:{} as Record<string,unknown>,mutate:vi.fn(),read:vi.fn(),item:vi.fn(),push:vi.fn(),replace:vi.fn(),upload:null as null|((url:string)=>void),block:null as null|((busy:boolean)=>void),allowUpload:true}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
vi.mock("next/navigation",()=>({useRouter:()=>({push:state.push,replace:state.replace})}));
vi.mock("./source-image-uploader",()=>({SourceImageUploader:({onImage,onBlocked,allowNewUpload}:{onImage:(url:string)=>void;onBlocked:(busy:boolean)=>void;allowNewUpload:boolean})=>{state.upload=onImage;state.block=onBlocked;state.allowUpload=allowNewUpload;return null}}));
vi.mock("@/lib/api/product-collection",async original=>({...await original<object>(),mutateCollection:state.mutate,readCollectionOperation:state.read,readCollectionItem:state.item,listOwnProducts:async()=>({items:[],total:0}),listCollectionBatches:async()=>({items:[],total:0})}));
const itemId="550e8400-e29b-41d4-a716-446655440002";
beforeEach(()=>{state.mutate.mockReset();state.read.mockReset();state.item.mockReset();state.push.mockReset();state.context={user:{id:"actor-a"},effectiveOrganization:{id:"org-a"},permissions:["workbench.collection.manage"],pendingCollectionIntent:null,collectionIntentReady:true,pendingSupplyIntent:null,supplyIntentReady:true,registerOrganizationSwitchGuard:()=>()=>{},setPendingCollectionIntent:(value:CollectionIntent|null)=>{state.context.pendingCollectionIntent=value;return true}}});
afterEach(cleanup);

const savedDetail={item:{id:itemId,batchId:itemId,revision:3,createdAt:"2026-10-14T01:00:00Z",source:{kind:"own" as const,productKey:"own-product",publicationId:itemId,version:"5"}},product:{title:"已有商品",brand:"原品牌",description:"原描述",attributes:[{name:"材质",value:"棉"}],images:[{url:"https://images.example.org/public.png"}],variants:[{source_id:"stable-sku",title:"原SKU标题",sku:"RED",attributes:[{name:"颜色",value:"红色"}],price:{amount:0,currency:"CNY"},images:[{url:"https://images.example.org/red.png"},{url:"https://images.example.org/red-detail.png"}]}]}};
it("prefills a saved product and appends with exact base while preserving SKU facts",async()=>{
 state.item.mockResolvedValue(savedDetail);state.mutate.mockResolvedValue({operationId:itemId,itemId,revision:4,replayed:false});
 render(<OwnProductEditPage itemId={itemId}/>);
 expect(await screen.findByDisplayValue("已有商品")).toBeInTheDocument();expect(screen.getByLabelText("库存 1")).toHaveValue(0);
 await userEvent.clear(screen.getByLabelText("商品名称"));await userEvent.type(screen.getByLabelText("商品名称"),"编辑后的商品");
 await userEvent.click(screen.getByRole("button",{name:"保存新版本"}));
 await waitFor(()=>expect(state.mutate).toHaveBeenCalledOnce());
 expect(state.mutate.mock.calls[0][0].command).toMatchObject({action:"revise_product",itemId,expectedRevision:3,expectedSourceVersion:"5",expectedPublicationId:itemId,product:{title:"编辑后的商品",variants:[{sourceId:"stable-sku",title:"原SKU标题",price:0,stock:0,images:["https://images.example.org/red.png","https://images.example.org/red-detail.png"]}]}});
 await waitFor(()=>expect(state.push).toHaveBeenCalledWith(`/workbench/data/mine/own/${itemId}`));expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});
it("shows a read error instead of a blank editable form for another source",async()=>{
 state.item.mockResolvedValue({...savedDetail,item:{...savedDetail.item,source:{...savedDetail.item.source,kind:"acquisition"}}});
 render(<OwnProductEditPage itemId={itemId}/>);
 expect(await screen.findByText(/未找到当前身份下的记录/)).toBeInTheDocument();expect(screen.queryByRole("button",{name:"保存新版本"})).not.toBeInTheDocument();
});
it("keeps a pending revision's original content and key when current source has advanced",async()=>{
 const product={title:"原待核实资料",description:"",images:[]};
 const pending:CollectionIntent={userId:"actor-a",organizationId:"org-a",key:itemId,command:{action:"revise_product",itemId,expectedRevision:1,expectedSourceVersion:"1",expectedPublicationId:itemId,product}};
 state.context.pendingCollectionIntent=pending;state.item.mockResolvedValue(savedDetail);state.read.mockRejectedValue(new CollectionAPIError("NOT_FOUND",404));state.mutate.mockResolvedValue({operationId:itemId,itemId,revision:2,replayed:true});
 render(<OwnProductEditPage itemId={itemId}/>);
 expect(await screen.findByDisplayValue("原待核实资料")).toBeDisabled();expect(screen.getByText("基于资料版本 1")).toBeInTheDocument();
 await userEvent.click(screen.getByRole("button",{name:"核实原操作"}));
 await userEvent.click(await screen.findByRole("button",{name:"重试原请求"}));
 await waitFor(()=>expect(state.mutate).toHaveBeenCalledWith(pending,expect.any(AbortSignal)));
 expect(state.push).toHaveBeenCalledWith(`/workbench/data/mine/own/${itemId}`);
});
it("keeps local changes after a conflict and asks before abandoning them",async()=>{
 state.item.mockResolvedValue(savedDetail);state.mutate.mockRejectedValue(new CollectionAPIError("REVISION_CONFLICT",409));
 const confirm=vi.spyOn(window,"confirm").mockReturnValue(false);render(<OwnProductEditPage itemId={itemId}/>);
 await screen.findByDisplayValue("已有商品");await userEvent.type(screen.getByLabelText("商品名称"),"修改");await userEvent.click(screen.getByRole("button",{name:"保存新版本"}));
 expect(await screen.findByText("资料已发生变化，请刷新后重试。")).toBeInTheDocument();expect(state.push).not.toHaveBeenCalled();
 await userEvent.click(screen.getByRole("button",{name:"取消并返回详情"}));expect(confirm).toHaveBeenCalledOnce();expect(state.push).not.toHaveBeenCalled();confirm.mockRestore();
});
it("does not reuse a fresh base with stale restored content after original-key conflict",async()=>{
 const pending:CollectionIntent={userId:"actor-a",organizationId:"org-a",key:itemId,command:{action:"revise_product",itemId,expectedRevision:1,expectedSourceVersion:"1",expectedPublicationId:itemId,product:{title:"旧版待核实内容",description:"",images:[]}}};
 state.context.pendingCollectionIntent=pending;state.item.mockResolvedValue(savedDetail);state.read.mockRejectedValue(new CollectionAPIError("NOT_FOUND",404));state.mutate.mockRejectedValue(new CollectionAPIError("REVISION_CONFLICT",409));
 render(<OwnProductEditPage itemId={itemId}/>);await screen.findByDisplayValue("旧版待核实内容");
 await userEvent.click(screen.getByRole("button",{name:"核实原操作"}));await userEvent.click(await screen.findByRole("button",{name:"重试原请求"}));
 await screen.findByText("资料已发生变化，请刷新后重试。");
 expect(screen.getByRole("button",{name:"保存新版本"})).toBeDisabled();expect(screen.getByLabelText("商品名称")).toBeDisabled();
 await userEvent.click(screen.getByRole("button",{name:"保存新版本"}));expect(state.mutate).toHaveBeenCalledOnce();
 expect(state.mutate.mock.calls[0][0]).toEqual(pending);
});

it("merges delayed upload callbacks into current images and locks image editing during pending upload",async()=>{
 render(<OwnProductEntryPage/>);
 const delayed=state.upload!;
 await userEvent.type(screen.getByLabelText("图片链接"),"https://images.example.org/a.png");await userEvent.click(screen.getByRole("button",{name:"添加图片链接"}));
 await userEvent.type(screen.getByLabelText("图片链接"),"https://images.example.org/b.png");await userEvent.click(screen.getByRole("button",{name:"添加图片链接"}));
 await userEvent.click(screen.getByRole("button",{name:"移除图片 1"}));
 act(()=>state.block!(true));expect(screen.getByLabelText("图片链接")).toBeDisabled();expect(screen.getByRole("button",{name:"移除图片 1"})).toBeDisabled();
 act(()=>delayed("https://images.example.org/c.png"));
 expect(screen.getAllByRole("img")).toHaveLength(2);expect(screen.getAllByRole("img")[0]).toHaveAttribute("src",expect.stringContaining("b.png"));
 act(()=>state.block!(false));expect(screen.getByLabelText("图片链接")).toBeEnabled();
});
it("blocks new uploads at 40 public images without replacing the existing recovery component",()=>{
 render(<OwnProductEntryPage/>);for(let i=0;i<40;i++)act(()=>state.upload!(`https://images.example.org/${i}.png`));
 expect(state.allowUpload).toBe(false);expect(screen.getAllByRole("img")).toHaveLength(40);
});
it("blocks save for a pending supply command but still allows scoped read-only recovery after write permission revocation",async()=>{
 const pending={userId:"actor-a",organizationId:"org-a",key:itemId,command:{action:"create_product" as const,product:{title:"已发请求",description:"",images:[]}}};
 state.context={...state.context,permissions:["workbench.collection.read"],pendingCollectionIntent:pending,pendingSupplyIntent:{key:itemId}};
 state.read.mockResolvedValue({operationId:itemId,itemId,revision:1,replayed:true});render(<OwnProductEntryPage/>);
 expect(screen.getByRole("button",{name:"保存商品"})).toBeDisabled();await userEvent.click(screen.getByRole("button",{name:"核实原操作"}));
 await waitFor(()=>expect(state.read).toHaveBeenCalledWith(pending,expect.any(AbortSignal)));expect(state.mutate).not.toHaveBeenCalled();
});

it("navigates from the library to a normal page instead of opening the old add dialog",async()=>{
 render(<CollectionPage initialView="own"/>);
 expect(await screen.findByRole("link",{name:"添加自有商品"})).toHaveAttribute("href","/workbench/data/mine/own/new");
 expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});
it("saves six user-declared combinations and their images in a single original intent",async()=>{
 state.mutate.mockResolvedValue({operationId:itemId,itemId,revision:1,replayed:false});
 render(<OwnProductEntryPage/>);
 await userEvent.type(screen.getByLabelText("商品名称"),"多规格上衣");
 await userEvent.type(screen.getByLabelText("图片链接"),"https://images.example.org/red.png\nhttps://images.example.org/blue.png");
 await userEvent.click(screen.getByRole("button",{name:"添加图片链接"}));
 await userEvent.click(screen.getByRole("radio",{name:"多规格商品"}));
 await userEvent.type(screen.getByLabelText("规格名称 1"),"颜色");await userEvent.type(screen.getByLabelText("规格值 1"),"红色,蓝色");
 await userEvent.click(screen.getByRole("button",{name:"添加规格"}));
 await userEvent.type(screen.getByLabelText("规格名称 2"),"尺码");await userEvent.type(screen.getByLabelText("规格值 2"),"S,M,L");
 await userEvent.click(screen.getByRole("button",{name:"生成变体组合"}));
 expect(screen.getByLabelText("SKU 6")).toBeVisible();
 await userEvent.type(screen.getByLabelText("SKU前缀"),"SHIRT");await userEvent.type(screen.getByLabelText("公共币种"),"CNY");await userEvent.type(screen.getByLabelText("公共价格"),"0");await userEvent.type(screen.getByLabelText("公共库存"),"0");
 await userEvent.click(screen.getByRole("button",{name:"应用到全部变体"}));
 await userEvent.selectOptions(screen.getByLabelText("变体图片 1"),"https://images.example.org/red.png");
 await userEvent.selectOptions(screen.getByLabelText("变体图片 4"),"https://images.example.org/blue.png");
 await userEvent.click(screen.getByRole("button",{name:"保存商品"}));
 await waitFor(()=>expect(state.mutate).toHaveBeenCalledOnce());
 const intent=state.mutate.mock.calls[0][0] as CollectionIntent;
 if(intent.command.action!=="create_product")throw new Error("wrong command");
 const variants=intent.command.product.variants!;expect(variants).toHaveLength(6);expect(new Set(variants.map(v=>v.sourceId)).size).toBe(6);
 expect(variants[0]).toMatchObject({attributes:{颜色:"红色",尺码:"S"},price:0,stock:0,images:["https://images.example.org/red.png"]});
 expect(variants[3].images).toEqual(["https://images.example.org/blue.png"]);
 expect(state.push).toHaveBeenCalledWith(`/workbench/data/mine?view=own&saved=${itemId}`);
});
it("keeps the exact unknown request locked and retries only its original key after NOT_FOUND",async()=>{
 state.mutate.mockRejectedValueOnce(new CollectionAPIError("OUTCOME_UNKNOWN",503));
 render(<OwnProductEntryPage/>);await userEvent.type(screen.getByLabelText("商品名称"),"原商品");await userEvent.click(screen.getByRole("button",{name:"保存商品"}));
 await screen.findByText("结果待核实");const original=state.mutate.mock.calls[0][0];expect(screen.getByLabelText("商品名称")).toBeDisabled();
 state.read.mockRejectedValueOnce(new CollectionAPIError("NOT_FOUND",404));await userEvent.click(screen.getByRole("button",{name:"核实原操作"}));
 await screen.findByRole("button",{name:"重试原请求"});state.mutate.mockResolvedValueOnce({operationId:itemId,itemId,revision:1,replayed:true});
 await userEvent.click(screen.getByRole("button",{name:"重试原请求"}));await waitFor(()=>expect(state.push).toHaveBeenCalledOnce());
 expect(state.mutate.mock.calls[1][0]).toBe(original);expect(state.read.mock.calls[0][0]).toBe(original);
});
it("does not navigate if the confirmed intent cannot be cleared from browser storage",async()=>{
 state.context.setPendingCollectionIntent=(intent:CollectionIntent|null)=>{if(!intent)return false;state.context.pendingCollectionIntent=intent;return true};
 state.mutate.mockResolvedValue({operationId:itemId,itemId,revision:1,replayed:false});render(<OwnProductEntryPage/>);
 await userEvent.type(screen.getByLabelText("商品名称"),"商品");await userEvent.click(screen.getByRole("button",{name:"保存商品"}));
 expect(await screen.findByRole("alert")).toHaveTextContent("无法保存原请求");expect(state.push).not.toHaveBeenCalled();
});
it("discards unsubmitted old identity input and prevents a foreign pending operation",async()=>{
 const view=render(<OwnProductEntryPage/>);await userEvent.type(screen.getByLabelText("商品名称"),"私有资料");
 state.context={...state.context,user:{id:"actor-b"},effectiveOrganization:{id:"org-b"},pendingCollectionIntent:{userId:"actor-a",organizationId:"org-a",key:itemId,command:{action:"create_product",product:{title:"原商品",description:"",images:[]}}}};
 view.rerender(<OwnProductEntryPage/>);expect(screen.getByLabelText("商品名称")).toHaveValue("");expect(screen.getByRole("button",{name:"保存商品"})).toBeDisabled();expect(state.mutate).not.toHaveBeenCalled();
});
it("reads every own variant on the separate detail page, including zero values",async()=>{
 state.item.mockResolvedValue({item:{source:{kind:"own",version:"1",productKey:"own-fixture"}},product:{title:"商品",variants:[{source_id:"red",sku:"RED",attributes:[{name:"颜色",value:"红"}],price:{currency:"CNY",amount:0}},{source_id:"blue",sku:"BLUE",attributes:[{name:"颜色",value:"蓝"}],price:{currency:"CNY",amount:12},stock:9}]}});
 render(<OwnProductDetailPage itemId={itemId}/>);const table=await screen.findByRole("table");expect(within(table).getByText("RED")).toBeVisible();expect(within(table).getByText("BLUE")).toBeVisible();expect(within(table).getByText("颜色：红")).toBeVisible();expect(within(table).getAllByText("0")).toHaveLength(2);expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});
it("submits common attributes through the original product request and blocks duplicate names",async()=>{
 state.mutate.mockResolvedValue({operationId:itemId,itemId,revision:1,replayed:false});render(<OwnProductEntryPage/>);
 await userEvent.type(screen.getByLabelText("商品名称"),"上衣");await userEvent.type(screen.getByLabelText("属性名称 1"),"材质");await userEvent.type(screen.getByLabelText("属性值 1"),"棉");
 await userEvent.click(screen.getByRole("button",{name:"添加商品属性"}));await userEvent.type(screen.getByLabelText("属性名称 2"),"材质");await userEvent.type(screen.getByLabelText("属性值 2"),"麻");
 await userEvent.click(screen.getByRole("button",{name:"保存商品"}));expect(await screen.findByRole("alert")).toHaveTextContent("属性名称不能重复");expect(state.mutate).not.toHaveBeenCalled();
 await userEvent.click(screen.getByRole("button",{name:"删除商品属性 2"}));await userEvent.click(screen.getByRole("button",{name:"保存商品"}));
 await waitFor(()=>expect(state.mutate).toHaveBeenCalledOnce());expect(state.mutate.mock.calls[0][0].command.product.attributes).toEqual({材质:"棉"});
});

it("preserves a saved rich description byte-for-byte while editing only its title",async()=>{
 const detail={...savedDetail,product:{...savedDetail.product,description:'<h2>材质</h2><p><strong>棉</strong><img src="https://images.example.org/public.png"></p>',description_format:"html-v1" as const}};
 state.item.mockResolvedValue(detail);state.mutate.mockResolvedValue({operationId:itemId,itemId,revision:4,replayed:false});
 render(<OwnProductEditPage itemId={itemId}/>);expect(await screen.findByRole("textbox",{name:"商品描述"})).toHaveTextContent("材质棉");
 await userEvent.clear(screen.getByLabelText("商品名称"));await userEvent.type(screen.getByLabelText("商品名称"),"只改名称");await userEvent.click(screen.getByRole("button",{name:"保存新版本"}));
 await waitFor(()=>expect(state.mutate).toHaveBeenCalledOnce());
 expect(state.mutate.mock.calls[0][0].command.product).toMatchObject({description:detail.product.description,descriptionFormat:"html-v1"});
});
