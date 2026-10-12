import { cleanup,fireEvent,render,screen,waitFor } from "@testing-library/react";
import { afterEach,expect,it,vi } from "vitest";
import { SupplyAPIError } from "@/lib/api/supply-chain";
import { CompletionDraftsPage } from "./completion-drafts-page";
const f=vi.hoisted(()=>({context:{user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.supply.read","workbench.supply.manage"],supplyIntentReady:true,pendingSupplyIntent:null as unknown,setPendingSupplyIntent:vi.fn<(intent:unknown)=>boolean>().mockReturnValue(true),registerOrganizationSwitchGuard:()=>()=>{}},batches:vi.fn(),read:vi.fn(),source:vi.fn(),list:vi.fn(),save:vi.fn(),recover:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("@/lib/api/supply-chain",async importOriginal=>({...await importOriginal<object>(),listSupplyPreparations:f.batches,listSupplyCompletionDrafts:f.list,readSupplyCompletionDraft:f.read,readSupplySource:f.source,supplyCommand:f.save,readSupplyCommand:f.recover}));
const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222";
const source={id,preparationId:other,collectionItemId:other,collectionRevision:1,source:{productKey:"original",publicationId:other,version:"1",kind:"own"}};
const record={id,draftId:other,revision:1,source,content:{title:"候选 R1",description:"描述"},status:"pending_completion",createdAt:"2026-10-12T00:00:00Z"};
afterEach(()=>{cleanup();vi.clearAllMocks();f.context.pendingSupplyIntent=null;f.context.effectiveOrganization.id="org"});
function ready(){f.context.setPendingSupplyIntent.mockImplementation(value=>{f.context.pendingSupplyIntent=value;return true});f.batches.mockResolvedValue({items:[{id:other,name:"真实批次",count:1}],total:1});f.list.mockResolvedValue({items:[record],total:1});f.source.mockResolvedValue({source,product:{title:"原始标题",description:"原始描述",variants:[],images:[]},images:[]});f.read.mockResolvedValue(record)}
it("shows actual saved drafts with unknown official fields and no fabricated AI actions",async()=>{
 ready();render(<CompletionDraftsPage initialPreparation={other}/>);
 expect(await screen.findByText("商品列表（共 1 个）")).toBeVisible();expect(screen.getByText("候选 R1")).toBeVisible();expect(screen.getByText("未绑定店铺，平台规则待配置")).toBeVisible();
 expect(screen.queryByRole("button",{name:/AI补全|批量确认/})).not.toBeInTheDocument();
});
it("keeps local source selection on the creation link when the full platform flow is enabled",async()=>{
 ready();render(<CompletionDraftsPage initialPreparation={other} fullAvailable/>);
 await screen.findByText("商品列表（共 1 个）");
 expect(screen.getByRole("link",{name:"从待适配创建草稿"})).toHaveAttribute("href",`/workbench/supply/mine/waiting?preparation=${other}&mode=local`);
});
it("moves to the next real batch page after entering through an explicit preparation",async()=>{
 ready();const later="33333333-3333-4333-8333-333333333333";
 f.batches.mockResolvedValueOnce({items:[{id:other,name:"真实批次",count:1}],total:2,nextCursor:other}).mockResolvedValueOnce({items:[{id:later,name:"下一页真实批次",count:1}],total:2});
 render(<CompletionDraftsPage initialPreparation={other}/>);await screen.findByText("商品列表（共 1 个）");
 fireEvent.click(screen.getByRole("button",{name:"下一页批次"}));
 await waitFor(()=>expect(f.list).toHaveBeenCalledWith(expect.anything(),later,expect.anything(),expect.any(AbortSignal)));
 expect(screen.getByRole("combobox",{name:"商品批次"})).toHaveValue(later);
});
it("retains an uncertain save and original key after NOT_FOUND, then reloads the actual head on recovery",async()=>{
 ready();f.save.mockRejectedValue(new SupplyAPIError("OUTCOME_UNKNOWN",503));render(<CompletionDraftsPage initialPreparation={other} initialSource={id}/>);
 await screen.findByRole("textbox",{name:"候选标题"});fireEvent.click(screen.getByRole("button",{name:"保存待补全草稿"}));
 expect(await screen.findByText("结果待核实")).toBeVisible();const intent=f.save.mock.lastCall![0];expect(screen.getByRole("textbox",{name:"候选标题"})).toBeDisabled();
 f.recover.mockRejectedValueOnce(new SupplyAPIError("NOT_FOUND",404));fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));await screen.findByRole("button",{name:"重试原请求"});expect(f.context.pendingSupplyIntent).toEqual(intent);expect(f.save).toHaveBeenCalledOnce();
 f.recover.mockResolvedValue({record,replayed:true});f.read.mockResolvedValue({...record,revision:2,content:{title:"另一个已保存版本",description:""}});fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));
 expect(await screen.findByText(/^未绑定店铺 · 已保存 R2/)).toBeVisible();expect(screen.getByRole("textbox",{name:"候选标题"})).toHaveValue("另一个已保存版本");expect(f.recover.mock.lastCall![0]).toEqual(intent);expect(f.context.pendingSupplyIntent).toBeNull();
});
it("ignores a late draft response after the enterprise changes",async()=>{
 ready();let finish!:(value:unknown)=>void;f.read.mockReturnValueOnce(new Promise(resolve=>{finish=resolve}));const view=render(<CompletionDraftsPage initialPreparation={other} initialSource={id}/>);
 await waitFor(()=>expect(f.read).toHaveBeenCalled());f.context.effectiveOrganization.id="org-b";f.read.mockResolvedValue({...record,content:{title:"新企业候选",description:""}});view.rerender(<CompletionDraftsPage initialPreparation={other} initialSource={id}/>);
 await waitFor(()=>expect(screen.getByRole("textbox",{name:"候选标题"})).toHaveValue("新企业候选"));finish(record);await waitFor(()=>expect(screen.getByRole("textbox",{name:"候选标题"})).toHaveValue("新企业候选"));
});
it("reads exact saved revision and saves independent text, then reloads the latest head",async()=>{
 ready();render(<CompletionDraftsPage initialPreparation={other} initialSource={id}/>);
 const title=await screen.findByRole("textbox",{name:"候选标题"});expect(title).toHaveValue("候选 R1");expect(screen.getByText("原始标题")).toBeVisible();
 fireEvent.change(title,{target:{value:"候选 R2"}});
 const next={...record,revision:2,content:{title:"候选 R2",description:"描述"}};f.save.mockResolvedValue({record:next,replayed:false});f.read.mockResolvedValue(next);
 fireEvent.click(screen.getByRole("button",{name:"保存待补全草稿"}));
 await waitFor(()=>expect(f.save).toHaveBeenCalledWith(expect.objectContaining({route:"save-completion-draft",command:{sourceId:id,expectedRevision:1,content:next.content}}),expect.any(AbortSignal)));
 expect(await screen.findByText(/^未绑定店铺 · 已保存 R2/)).toBeVisible();expect(screen.getByRole("textbox",{name:"候选标题"})).toHaveValue("候选 R2");
});
