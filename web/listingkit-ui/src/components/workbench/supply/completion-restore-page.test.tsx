import {act,cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {SupplyAPIError,type SupplyIntent} from "@/lib/api/supply-chain";
import {CompletionRestorePage} from "./completion-restore-page";
import RestoreRoute from "@/app/workbench/supply/mine/[stage]/[sourceId]/history/[revision]/restore/page";

const f=vi.hoisted(()=>({context:{user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.supply.read","workbench.supply.manage"],supplyIntentReady:true,pendingSupplyIntent:null as SupplyIntent|null,setPendingSupplyIntent:vi.fn<(intent:SupplyIntent|null)=>boolean>(),registerOrganizationSwitchGuard:()=>()=>{}},source:vi.fn(),head:vi.fn(),old:vi.fn(),save:vi.fn(),verify:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),readSupplySource:f.source,readSupplyCompletionDraft:f.head,readCompletionRevision:f.old,supplyCommand:f.save,readSupplyCommand:f.verify}));
vi.mock("next/server",()=>({connection:async()=>{}}));
vi.mock("next/navigation",()=>({notFound:()=>{throw new Error("NOT_FOUND")}}));
const id="11111111-1111-4111-8111-111111111111",batch="22222222-2222-4222-8222-222222222222",other="33333333-3333-4333-8333-333333333333";
const source={id,preparationId:batch,collectionItemId:batch,collectionRevision:1,source:{kind:"own",version:"1",productKey:"fixed-own",publicationId:batch}};
const old={id,draftId:batch,revision:1,source,content:{title:"旧标题",description:"\n 旧描述\n"},status:"pending_completion",createdAt:"2026-10-14T00:00:00Z"};
const head={...old,id:other,revision:3,content:{title:"当前标题",description:"新版描述"}};
const saved={...head,revision:4,content:old.content};
const props={sourceId:id,preparationId:batch,itemId:batch,revision:1};
const confirm=()=>screen.getByRole("button",{name:"恢复为新版本"});
const receipt=(record:unknown=saved)=>({record,replayed:false});
beforeEach(()=>{
 vi.clearAllMocks();f.context.user.id="actor";f.context.effectiveOrganization.id="org";f.context.permissions=["workbench.supply.read","workbench.supply.manage"];f.context.supplyIntentReady=true;f.context.pendingSupplyIntent=null;
 f.context.setPendingSupplyIntent.mockImplementation(v=>{f.context.pendingSupplyIntent=v;return true});
 f.source.mockResolvedValue({source,images:[],product:{title:"固定商品V1"}});f.head.mockResolvedValue(head);f.old.mockResolvedValue(old);f.save.mockResolvedValue(receipt());f.verify.mockResolvedValue({...receipt(),replayed:true});
 vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED","true");vi.stubEnv("LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED","true");
});
afterEach(()=>{cleanup();vi.unstubAllEnvs()});

it("previews exact current and historical text on an independent page with all return contexts",async()=>{
 render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");
 expect(screen.getByText("固定来源 V1")).toBeVisible();expect(screen.getByText((_,e)=>e?.tagName==="P"&&e.textContent===old.content.description)).toBeVisible();
 expect(screen.queryByRole("dialog")).not.toBeInTheDocument();expect(screen.queryByRole("textbox")).not.toBeInTheDocument();expect(f.save).not.toHaveBeenCalled();expect(confirm()).toBeEnabled();
 expect(screen.getByRole("link",{name:"返回草稿"})).toHaveAttribute("href",`/workbench/supply/mine/missing/${id}?preparation=${batch}&item=${batch}`);
 expect(screen.getByRole("link",{name:"返回商品"})).toHaveAttribute("href",`/workbench/data/mine/own/${batch}`);
 expect(screen.getByRole("link",{name:"返回待补全列表"})).toHaveAttribute("href",`/workbench/supply/mine/missing?preparation=${batch}`);
 expect(screen.getByRole("link",{name:"返回旧版详情"})).toHaveAttribute("href",`/workbench/supply/mine/missing/${id}/history/1?preparation=${batch}&item=${batch}`);
});
it("uses the existing Save command with previewed head CAS and exact old text only after confirmation",async()=>{
 const original=JSON.stringify(old);render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");
 f.head.mockResolvedValue(saved);fireEvent.click(confirm());fireEvent.click(confirm());await screen.findByText("已保存为 R4");
 expect(f.save).toHaveBeenCalledTimes(1);expect(f.save).toHaveBeenCalledWith(expect.objectContaining({route:"save-completion-draft",userId:"actor",organizationId:"org",command:{sourceId:id,expectedRevision:3,content:old.content}}),expect.any(AbortSignal));
 expect(JSON.stringify(old)).toBe(original);expect(confirm()).toBeDisabled();expect(screen.getByRole("link",{name:"查看已保存 R4"})).toHaveAttribute("href",`/workbench/supply/mine/missing/${id}/history/4?preparation=${batch}&item=${batch}`);
});
it("keeps read-only users out of restore and unmounts immediately when management is revoked",async()=>{
 f.context.permissions=["workbench.supply.read"];const view=render(<CompletionRestorePage {...props}/>);expect(screen.getByText("当前身份没有草稿恢复权限")).toBeVisible();expect(f.old).not.toHaveBeenCalled();expect(f.save).not.toHaveBeenCalled();
 f.context.permissions.push("workbench.supply.manage");view.rerender(<CompletionRestorePage {...props}/>);await screen.findByText("旧标题");
 f.context.permissions=["workbench.supply.read"];view.rerender(<CompletionRestorePage {...props}/>);expect(screen.queryByText("旧标题")).not.toBeInTheDocument();expect(screen.queryByRole("button",{name:"恢复为新版本"})).not.toBeInTheDocument();
});
it("rejects wrong source/draft/revision and navigation before showing a confirmable preview",async()=>{
 for(const change of [{...old,revision:2},{...old,draftId:id},{...old,source:{...source,id:other}},{...old,source:{...source,source:{...source.source,version:"2"}}}]){
  f.old.mockResolvedValue(change);const view=render(<CompletionRestorePage {...props}/>);await screen.findByText("无法读取恢复内容");expect(screen.queryByText("旧标题")).not.toBeInTheDocument();expect(screen.queryByRole("button",{name:"恢复为新版本"})).not.toBeInTheDocument();view.unmount();
 }
 f.old.mockResolvedValue(old);
 for(const navigation of [{preparationId:id,itemId:batch},{preparationId:batch,itemId:id}]){
  const view=render(<CompletionRestorePage {...props} {...navigation}/>);await screen.findByText("无法读取恢复内容");expect(screen.queryByRole("table")).not.toBeInTheDocument();view.unmount();
 }
 f.head.mockRejectedValue(new SupplyAPIError("PERMISSION_DENIED",403));render(<CompletionRestorePage {...props}/>);await screen.findByText("无法读取恢复内容");expect(f.save).not.toHaveBeenCalled();
});
it("does not append when selecting the current version or identical content, preserving whitespace differences",async()=>{
 f.head.mockResolvedValue(old);let view=render(<CompletionRestorePage {...props}/>);await screen.findByText("所选版本已是当前版本，无需恢复。");expect(confirm()).toBeDisabled();view.unmount();
 f.head.mockResolvedValue({...head,content:{...old.content,structured:{attributes:[{name:"材质",value:"当前候选"}],variants:[]}}});view=render(<CompletionRestorePage {...props}/>);await screen.findByText("标题和描述与当前版本相同，无需恢复。");expect(confirm()).toBeDisabled();expect(f.save).not.toHaveBeenCalled();view.unmount();
 f.old.mockResolvedValue({...old,content:{title:"",description:"\n"}});f.head.mockResolvedValue({...head,content:{title:"",description:""}});f.save.mockResolvedValue(receipt({...saved,content:{title:"",description:"\n"}}));render(<CompletionRestorePage {...props}/>);await screen.findByText("仅含空白文本 · 1 个字符");fireEvent.click(confirm());await waitFor(()=>expect(f.save).toHaveBeenCalledWith(expect.objectContaining({command:{sourceId:id,expectedRevision:3,content:{title:"",description:"\n"}}}),expect.any(AbortSignal)));
});
it("locks a conflicted preview until explicit reload and requires a new confirmation against the new head",async()=>{
 f.save.mockRejectedValueOnce(new SupplyAPIError("REVISION_CONFLICT",409));render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");fireEvent.click(confirm());await screen.findByText("当前版本已变化，请重新读取并确认差异后再恢复。");expect(confirm()).toBeDisabled();expect(f.save).toHaveBeenCalledTimes(1);
 f.head.mockResolvedValue({...head,revision:4});fireEvent.click(screen.getByRole("button",{name:"重新读取当前版本"}));await screen.findByText("当前已保存 R4");expect(confirm()).toBeEnabled();expect(f.save).toHaveBeenCalledTimes(1);
 f.save.mockResolvedValue(receipt({...saved,revision:5}));f.head.mockResolvedValue({...saved,revision:5});fireEvent.click(confirm());await screen.findByText("已保存为 R5");expect(f.save).toHaveBeenCalledTimes(2);
 expect(f.save.mock.calls.map(([intent])=>intent.command.expectedRevision)).toEqual([3,4]);expect(f.save.mock.calls[0]![0].key).not.toBe(f.save.mock.calls[1]![0].key);
});
it("retains the original intent across re-entry and verifies its exact receipt without claiming it is the latest",async()=>{
 f.save.mockRejectedValueOnce(new SupplyAPIError("OUTCOME_UNKNOWN",503));let view=render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");fireEvent.click(confirm());await screen.findByText("结果待核实");expect(confirm()).toBeDisabled();const intent=f.context.pendingSupplyIntent!;view.unmount();
 f.head.mockResolvedValue({...head,revision:5});view=render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R5");fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));await screen.findByText("已保存为 R4");expect(f.verify).toHaveBeenCalledWith(intent,expect.any(AbortSignal));expect(f.save).toHaveBeenCalledTimes(1);expect(f.context.pendingSupplyIntent).toBeNull();expect(confirm()).toBeDisabled();expect(screen.getByText("当前已保存 R5")).toBeVisible();
});
it("retries only the original key and original payload after an unresolved operation reads NOT_FOUND",async()=>{
 f.save.mockRejectedValueOnce(new SupplyAPIError("OUTCOME_UNKNOWN",503));f.verify.mockRejectedValueOnce(new SupplyAPIError("NOT_FOUND",404));render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");fireEvent.click(confirm());await screen.findByText("结果待核实");const original=f.context.pendingSupplyIntent!;
 fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));await screen.findByRole("button",{name:"重试原请求"});expect(confirm()).toBeDisabled();fireEvent.click(screen.getByRole("button",{name:"重试原请求"}));await screen.findByText("已保存为 R4");expect(f.save).toHaveBeenCalledTimes(2);expect(f.save.mock.calls[1]![0]).toEqual(original);
});
it("keeps a verified receipt while the selected old version is still loading after re-entry",async()=>{
 let resolve!:(value:unknown)=>void;f.old.mockReturnValue(new Promise(r=>{resolve=r}));f.head.mockResolvedValue({...head,revision:5});
 f.context.pendingSupplyIntent={userId:"actor",organizationId:"org",key:other,route:"save-completion-draft",command:{sourceId:id,expectedRevision:3,content:old.content}};
 render(<CompletionRestorePage {...props}/>);fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));await waitFor(()=>expect(f.context.pendingSupplyIntent).toBeNull());await act(async()=>resolve(old));
 await screen.findByText("已保存为 R4");expect(screen.getByText("当前已保存 R5")).toBeVisible();expect(confirm()).toBeDisabled();expect(f.save).not.toHaveBeenCalled();
});
it("does not report another pending save as this historical content restoration",async()=>{
 for(const command of [{sourceId:other,expectedRevision:3,content:old.content},{sourceId:id,expectedRevision:3,content:head.content}]){
  f.context.pendingSupplyIntent={userId:"actor",organizationId:"org",key:other,route:"save-completion-draft",command};f.verify.mockResolvedValue(receipt({...saved,source:{...source,id:command.sourceId},content:command.content}));
  const view=render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");expect(confirm()).toBeDisabled();fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));await waitFor(()=>expect(confirm()).toBeEnabled());expect(screen.queryByText("已保存为 R4")).not.toBeInTheDocument();view.unmount();
 }
 expect(f.save).not.toHaveBeenCalled();
});
it("blocks new writes when browser intent storage is unavailable",async()=>{
 f.context.supplyIntentReady=false;const view=render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");expect(confirm()).toBeDisabled();view.unmount();
 f.context.supplyIntentReady=true;f.context.setPendingSupplyIntent.mockReturnValue(false);render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");fireEvent.click(confirm());await screen.findByText(/浏览器无法保存或清除原操作键/);expect(f.save).not.toHaveBeenCalled();
});
it("keeps foreign-scope pending commands isolated and leaves live save authorization to the backend",async()=>{
 f.context.pendingSupplyIntent={userId:"another-actor",organizationId:"org",key:other,route:"save-completion-draft",command:{sourceId:id,expectedRevision:3,content:old.content}};
 const view=render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");expect(screen.getByText("请回到原登录身份与企业后核实。")).toBeVisible();expect(confirm()).toBeDisabled();expect(screen.queryByRole("button",{name:"核实原操作"})).not.toBeInTheDocument();expect(f.verify).not.toHaveBeenCalled();view.unmount();
 f.context.pendingSupplyIntent=null;f.save.mockRejectedValueOnce(new SupplyAPIError("PERMISSION_DENIED",403));render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");fireEvent.click(confirm());await screen.findByText("当前权限不足，请确认原身份与企业权限。");expect(screen.queryByText("已保存为 R4")).not.toBeInTheDocument();expect(f.context.pendingSupplyIntent).toBeNull();
});
it("refuses to present a receipt from a different draft as a successful restoration",async()=>{
 f.save.mockResolvedValue(receipt({...saved,draftId:other}));render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");fireEvent.click(confirm());await screen.findByText("无法读取恢复内容");expect(screen.queryByText("已保存为 R4")).not.toBeInTheDocument();expect(screen.queryByRole("button",{name:"恢复为新版本"})).not.toBeInTheDocument();
});
it("drops late old content immediately when scope or same-source navigation/revision changes",async()=>{
 let resolve!:(value:unknown)=>void;f.old.mockReturnValueOnce(new Promise(r=>{resolve=r}));const view=render(<CompletionRestorePage {...props}/>);
 f.context.effectiveOrganization.id="another-org";f.old.mockReturnValue(new Promise(()=>{}));view.rerender(<CompletionRestorePage {...props}/>);await act(async()=>resolve(old));expect(screen.queryByText("旧标题")).not.toBeInTheDocument();
 view.rerender(<CompletionRestorePage {...props} revision={2}/>);expect(screen.queryByRole("button",{name:"恢复为新版本"})).not.toBeInTheDocument();view.rerender(<CompletionRestorePage {...props} itemId={other}/>);expect(screen.queryByText("当前标题")).not.toBeInTheDocument();expect(f.save).not.toHaveBeenCalled();
});
it("restores only old text while keeping the current structured candidate",async()=>{
 const current={attributes:[{name:"材质",value:"当前"}],variants:[{sourceIndex:0,price:{currency:"CNY",amount:0},stock:0}]};
 const previous={attributes:[],variants:[{sourceIndex:0,price:null,stock:null}]};
 f.head.mockResolvedValue({...head,content:{...head.content,structured:current}});f.old.mockResolvedValue({...old,content:{...old.content,structured:previous}});
 f.save.mockResolvedValue(receipt({...saved,content:{...old.content,structured:current}}));
 render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");fireEvent.click(confirm());
 await waitFor(()=>expect(f.save).toHaveBeenCalledWith(expect.objectContaining({command:{sourceId:id,expectedRevision:3,content:{...old.content,structured:current}}}),expect.any(AbortSignal)));
});
it("guards the independent route, feature flags, revision syntax and strict navigation query",async()=>{
 const route=(params={stage:"missing",sourceId:id,revision:"1"},query:Record<string,string|string[]|undefined>={preparation:batch,item:batch})=>RestoreRoute({params:Promise.resolve(params),searchParams:Promise.resolve(query)});
 const element=await route();expect(element.props).toEqual(props);
 for(const params of [{stage:"adapted",sourceId:id,revision:"1"},{stage:"missing",sourceId:"bad",revision:"1"},...["0","01","-1","1.0","9007199254740992"].map(revision=>({stage:"missing",sourceId:id,revision}))])await expect(route(params)).rejects.toThrow("NOT_FOUND");
 for(const query of [{item:[batch,batch]},{preparation:"bad"},{returnTo:"https://example.org"},{from:"1",to:"2"}])await expect(route(undefined,query)).rejects.toThrow("NOT_FOUND");
 vi.stubEnv("LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED","false");await expect(route()).rejects.toThrow("NOT_FOUND");vi.stubEnv("LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED","true");vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED","false");await expect(route()).rejects.toThrow("NOT_FOUND");
});

it("restores the old description format and preserves current structured candidates in both directions",async()=>{
 const structured={attributes:[{name:"材质",value:"棉"}],variants:[]};
 for(const richOld of [true,false]){
  const previous={...old,content:{title:"same",description:"<p>same</p>",...(richOld?{descriptionFormat:"html-v1" as const}:{})}};
  const current={...head,content:{title:"same",description:"<p>same</p>",...(!richOld?{descriptionFormat:"html-v1" as const}:{}),structured}};
  f.old.mockResolvedValue(previous);f.head.mockResolvedValue(current);f.save.mockClear();
  const view=render(<CompletionRestorePage {...props}/>);await screen.findByText("当前已保存 R3");expect(confirm()).toBeEnabled();
  fireEvent.click(confirm());await waitFor(()=>expect(f.save).toHaveBeenCalledOnce());
  const content=f.save.mock.calls[0]![0].command.content;
  expect(content).toEqual({...previous.content,structured});if(!richOld)expect(content).not.toHaveProperty("descriptionFormat");
  view.unmount();f.context.pendingSupplyIntent=null;
 }
});
