import {cleanup,fireEvent,render,screen,waitFor,within} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {SupplyAPIError} from "@/lib/api/supply-chain";
import {CompletionDraftPage} from "./completion-draft-page";
// JSDOM does not lay out text selections; ProseMirror needs these browser APIs.
Object.defineProperty(Range.prototype,"getClientRects",{configurable:true,value:()=>[]});
Object.defineProperty(Range.prototype,"getBoundingClientRect",{configurable:true,value:()=>new DOMRect()});
const f=vi.hoisted(()=>({context:{user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.supply.read","workbench.supply.manage"],supplyIntentReady:true,pendingSupplyIntent:null as unknown,setPendingSupplyIntent:vi.fn<(intent:unknown)=>boolean>(),registerOrganizationSwitchGuard:()=>()=>{}},source:vi.fn(),read:vi.fn(),save:vi.fn(),upload:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),readSupplySource:f.source,readSupplyCompletionDraft:f.read,supplyCommand:f.save}));
vi.mock("@/lib/api/product-collection",async original=>({...await original<object>(),uploadSourceImage:f.upload}));
const id="11111111-1111-4111-8111-111111111111",preparation="22222222-2222-4222-8222-222222222222";
const source={id,preparationId:preparation,collectionItemId:preparation,collectionRevision:1,source:{kind:"own",version:"1",productKey:"fixed-own",publicationId:preparation}};
const detail={source,images:[],product:{title:"固定原商品V1",description:"原始描述",attributes:[{name:"材质",value:"棉"}],images:[],variants:[{source_id:"red",sku:"RED",title:"红色常规款",price:{currency:"CNY",amount:0},stock:0}]}};
const record={id,draftId:preparation,revision:1,source,content:{title:"候选内容R1",description:"候选描述"},status:"pending_completion",createdAt:"2026-10-14T00:00:00Z"};
beforeEach(()=>{vi.clearAllMocks();f.context.pendingSupplyIntent=null;f.context.effectiveOrganization.id="org";f.context.setPendingSupplyIntent.mockImplementation(v=>{f.context.pendingSupplyIntent=v;return true});f.source.mockResolvedValue(detail);f.read.mockRejectedValue(new SupplyAPIError("NOT_FOUND",404));f.save.mockResolvedValue({record,replayed:false})});
afterEach(()=>{cleanup();localStorage.clear();vi.unstubAllGlobals();f.context.permissions=["workbench.supply.read","workbench.supply.manage"]});
it("links the exact saved candidate to SHEIN with verified item and batch context",async()=>{
 f.read.mockResolvedValue(record);render(<CompletionDraftPage sourceId={id} preparationId={preparation} itemId={preparation} sheinAvailable/>);expect(await screen.findByRole("link",{name:"SHEIN 带入预览（已保存 R1）"})).toHaveAttribute("href",`/workbench/supply/mine/missing/${id}/shein?preparation=${preparation}&item=${preparation}&revision=1`);
});
it("locks text editing after CAS conflict until the shared head is explicitly reloaded",async()=>{
 f.read.mockResolvedValue(record);f.save.mockRejectedValueOnce(new SupplyAPIError("REVISION_CONFLICT",409));
 render(<CompletionDraftPage sourceId={id}/>);await screen.findByRole("textbox",{name:"候选标题"});fireEvent.click(screen.getByRole("button",{name:"保存待补全草稿"}));await screen.findByText("资料已发生变化，请重新读取后补齐。");expect(screen.getByRole("button",{name:"保存待补全草稿"})).toBeDisabled();
 f.read.mockResolvedValue({...record,revision:2,content:{...record.content,structured:{attributes:[],variants:[{sourceIndex:0,price:null,stock:0}]}}});fireEvent.click(screen.getByRole("button",{name:"重新读取保存版本"}));await waitFor(()=>expect(screen.getByRole("button",{name:"保存待补全草稿"})).toBeEnabled());
});
it("links to an independent structured editor with the same source, item and batch",async()=>{
 render(<CompletionDraftPage sourceId={id} preparationId={preparation} itemId={preparation}/>);
 await screen.findByRole("textbox",{name:"候选标题"});
 expect(screen.getByRole("link",{name:"编辑商品属性和 SKU"})).toHaveAttribute("href",`/workbench/supply/mine/missing/${id}/structure?preparation=${preparation}&item=${preparation}`);
});
it("keeps current structured candidates when saving title and description",async()=>{
 const structured={attributes:[],variants:[{sourceIndex:0,price:null,stock:0}]};
 const current={...record,content:{...record.content,structured}};
 f.read.mockResolvedValue(current);f.save.mockResolvedValue({record:{...current,revision:2},replayed:false});
 render(<CompletionDraftPage sourceId={id} preparationId={preparation} itemId={preparation}/>);
 await screen.findByRole("textbox",{name:"候选标题"});fireEvent.click(screen.getByRole("button",{name:"保存待补全草稿"}));
 await waitFor(()=>expect(f.save).toHaveBeenCalledWith(expect.objectContaining({command:{sourceId:id,expectedRevision:1,content:current.content}}),expect.any(AbortSignal)));
});
it("keeps both return links, reads the latest head, and freezes same-source navigation changes",async()=>{
 f.read.mockResolvedValue({...record,revision:3});
 const view=render(<CompletionDraftPage sourceId={id} preparationId={preparation} itemId={preparation}/>);
 expect(await screen.findByRole("textbox",{name:"候选标题"})).toHaveValue(record.content.title);expect(screen.getByText(/^未绑定店铺 · 已保存 R3/)).toBeVisible();
 expect(screen.getByRole("link",{name:"返回商品"})).toHaveAttribute("href",`/workbench/data/mine/own/${preparation}`);expect(screen.getByRole("link",{name:"返回待补全列表"})).toHaveAttribute("href",`/workbench/supply/mine/missing?preparation=${preparation}`);
 expect(screen.getByRole("link",{name:"保存历史"})).toHaveAttribute("href",`/workbench/supply/mine/missing/${id}/history?preparation=${preparation}&item=${preparation}`);
 f.source.mockReturnValue(new Promise(()=>{}));view.rerender(<CompletionDraftPage sourceId={id} preparationId={id} itemId={preparation}/>);expect(screen.queryByRole("textbox",{name:"候选标题"})).not.toBeInTheDocument();expect(screen.queryByRole("button",{name:"保存待补全草稿"})).not.toBeInTheDocument();
});
it("rejects a mismatching item or preparation before reading or prefilling saved content",async()=>{
 for(const props of [{preparationId:id,itemId:preparation},{preparationId:preparation,itemId:id}]){
  f.read.mockClear();const view=render(<CompletionDraftPage sourceId={id} {...props}/>);await screen.findByText("草稿或原资料读取失败");expect(f.read).not.toHaveBeenCalled();expect(screen.queryByRole("textbox")).not.toBeInTheDocument();view.unmount();
 }
});
it("freezes the form when only the item context changes and keeps read-only saves disabled",async()=>{
 f.read.mockResolvedValue(record);const view=render(<CompletionDraftPage sourceId={id} preparationId={preparation} itemId={preparation}/>);await screen.findByRole("textbox",{name:"候选标题"});
 f.source.mockReturnValue(new Promise(()=>{}));view.rerender(<CompletionDraftPage sourceId={id} preparationId={preparation} itemId={id}/>);expect(screen.queryByRole("textbox")).not.toBeInTheDocument();view.unmount();
 f.source.mockResolvedValue(detail);f.context.permissions=["workbench.supply.read"];render(<CompletionDraftPage sourceId={id} preparationId={preparation} itemId={preparation}/>);expect(await screen.findByRole("textbox",{name:"候选标题"})).toBeDisabled();expect(screen.getByRole("button",{name:"保存待补全草稿"})).toBeDisabled();f.context.permissions=["workbench.supply.read","workbench.supply.manage"];
});
it("independently edits and saves candidate text while inspecting the fixed own snapshot",async()=>{
 render(<CompletionDraftPage sourceId={id}/>);const title=await screen.findByRole("textbox",{name:"候选标题"});expect(title).toHaveValue("固定原商品V1");expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
 fireEvent.click(screen.getByText("查看固定原始商品资料"));expect(within(screen.getByRole("region",{name:"选中SKU资料"})).getByText("红色常规款")).toBeVisible();expect(screen.getByText("CNY 0")).toBeVisible();expect(screen.getByText("0 件")).toBeVisible();expect(screen.getByText("棉")).toBeVisible();
 fireEvent.change(title,{target:{value:record.content.title}});const editor=screen.getByRole("textbox",{name:"候选描述"});await userEvent.clear(editor);await screen.findByText(/^7 \/ 32768 字节/);fireEvent.paste(screen.getByRole("textbox",{name:"候选描述"}),{clipboardData:{getData:(type:string)=>type==="text/plain"?record.content.description:"",types:["text/plain"],files:[]}});await screen.findByText(/^19 \/ 32768 字节/);f.read.mockResolvedValue(record);fireEvent.click(screen.getByRole("button",{name:"保存待补全草稿"}));
 await waitFor(()=>expect(f.save).toHaveBeenCalledWith(expect.objectContaining({route:"save-completion-draft",command:{sourceId:id,expectedRevision:0,content:{...record.content,description:`<p>${record.content.description}</p>`,descriptionFormat:"html-v1"}}}),expect.any(AbortSignal)));
 await screen.findByText(/^未绑定店铺 · 已保存 R1/);expect(f.source).toHaveBeenCalledWith({userId:"actor",organizationId:"org"},id,expect.any(AbortSignal));
});
it("refuses candidate content whose retained source differs from the fixed detail",async()=>{
 f.read.mockResolvedValue({...record,source:{...source,source:{...source.source,version:"2"}}});render(<CompletionDraftPage sourceId={id}/>);
 await screen.findByText("草稿或原资料读取失败");expect(screen.queryByRole("textbox",{name:"候选标题"})).not.toBeInTheDocument();expect(screen.queryByText(record.content.title)).not.toBeInTheDocument();
});

it.each([false,true])("saves only referenced uploaded resources after undo/redo (redo=%s) without changing the fixed source",async redo=>{
 localStorage.clear();f.context.permissions.push("workbench.collection.manage");
 const image="https://images.example.org/candidate.png";
 f.save.mockImplementation(({command}:{command:{expectedRevision:number;content:Record<string,unknown>}})=>{const saved={...record,revision:command.expectedRevision+1,content:command.content};f.read.mockResolvedValue(saved);return Promise.resolve({record:saved,replayed:false})});
 vi.stubGlobal("crypto",{randomUUID:()=>id,subtle:{digest:vi.fn().mockResolvedValue(new Uint8Array(32).buffer)}});
 f.upload.mockResolvedValue({url:image,hash:"00".repeat(32),bytes:3,mediaType:"image/png",width:2,height:2});
 render(<CompletionDraftPage sourceId={id}/>);await screen.findByRole("textbox",{name:"候选描述"});
 fireEvent.click(screen.getByRole("button",{name:"图片"}));
 const file=new File([new Uint8Array([1,2,3])],"candidate.png",{type:"image/png"});
 Object.defineProperty(file,"arrayBuffer",{value:async()=>new Uint8Array([1,2,3]).buffer});
 fireEvent.change(screen.getByLabelText("图片文件"),{target:{files:[file]}});
 await waitFor(()=>expect(screen.getByRole("button",{name:"上传并添加图片链接"})).toBeEnabled());
 fireEvent.click(screen.getByRole("button",{name:"上传并添加图片链接"}));
 await waitFor(()=>expect(screen.getByRole("textbox",{name:"候选描述"}).querySelector("img")).toHaveAttribute("src","/api/image-proxy?url="+encodeURIComponent(image)));
 await waitFor(()=>expect(screen.getByRole("button",{name:"撤销"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"撤销"}));
 await waitFor(()=>expect(screen.getByRole("textbox",{name:"候选描述"}).querySelector("img")).toBeNull());
 if(redo){await waitFor(()=>expect(screen.getByRole("button",{name:"重做"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"重做"}));await waitFor(()=>expect(screen.getByRole("textbox",{name:"候选描述"}).querySelector("img")).toHaveAttribute("src","/api/image-proxy?url="+encodeURIComponent(image)))}
 await waitFor(()=>expect(screen.getByRole("button",{name:"保存待补全草稿"})).toBeEnabled());
 fireEvent.click(screen.getByRole("button",{name:"保存待补全草稿"}));
 await waitFor(()=>expect(f.save).toHaveBeenCalled());
 const savedContent=f.save.mock.calls[0][0].command.content;
 expect(savedContent.descriptionFormat).toBe("html-v1");expect(savedContent.descriptionImages).toEqual(redo?[image]:undefined);expect(savedContent.description.includes(image)).toBe(redo);
 await screen.findByText(/^未绑定店铺 · 已保存 R1/);
 expect(detail.product.images).toEqual([]);expect(detail.source.source.version).toBe("1");
});
it("restores saved candidate image references when undoing a description deletion",async()=>{
 const image="https://images.example.org/saved.png";
 f.read.mockResolvedValue({...record,content:{...record.content,description:`<p>已保存的描述</p><img src="${image}" alt="商品描述插图">`,descriptionFormat:"html-v1",descriptionImages:[image]}});
 render(<CompletionDraftPage sourceId={id}/>);const editor=await screen.findByRole("textbox",{name:"候选描述"});
 await userEvent.clear(editor);await waitFor(()=>expect(editor.querySelector("img")).toBeNull());
 await waitFor(()=>expect(screen.getByRole("button",{name:"撤销"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"撤销"}));
 await waitFor(()=>expect(editor.querySelector("img")).toHaveAttribute("src","/api/image-proxy?url="+encodeURIComponent(image)));
 fireEvent.click(screen.getByRole("button",{name:"保存待补全草稿"}));await waitFor(()=>expect(f.save).toHaveBeenCalled());
 expect(f.save.mock.calls[0][0].command.content.descriptionImages).toEqual([image]);
});
it("keeps pending upload recovery visible and blocks save when upload permission was removed",async()=>{
 localStorage.setItem("shuomi_source_media_intent",JSON.stringify({userId:"actor",organizationId:"org",hash:"00".repeat(32),bytes:3}));
 render(<CompletionDraftPage sourceId={id}/>);await screen.findByRole("textbox",{name:"候选描述"});
 await waitFor(()=>expect(screen.getByRole("button",{name:"保存待补全草稿"})).toBeDisabled());
 expect(screen.getByRole("button",{name:"重新读取保存版本"})).toBeDisabled();
 expect(fireEvent.click(screen.getByRole("link",{name:"返回待补全列表"}))).toBe(false);
 expect(fireEvent.click(screen.getByRole("link",{name:"编辑商品属性和 SKU"}))).toBe(false);
 fireEvent.click(screen.getByRole("button",{name:"图片"}));
 expect(screen.getByRole("button",{name:"核实原图片"})).toBeDisabled();expect(localStorage.getItem("shuomi_source_media_intent")).not.toBeNull();
 expect(f.save).not.toHaveBeenCalled();
});
