import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {CompletionHistoryPage} from "./completion-history-page";
const f=vi.hoisted(()=>({context:{user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.supply.read"]},list:vi.fn(),read:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),listCompletionHistory:f.list,readCompletionRevision:f.read}));
const id="11111111-1111-4111-8111-111111111111",batch="22222222-2222-4222-8222-222222222222";
const source={id,preparationId:batch,collectionItemId:batch,collectionRevision:1,source:{kind:"own",version:"1",productKey:"fixed-own",publicationId:batch}};
const old={id,draftId:batch,revision:1,source,content:{title:"旧标题",description:"\n"},status:"pending_completion",createdAt:"2026-10-14T00:00:00Z"};
const current={...old,id:batch,revision:2,content:{title:"新标题",description:"新版描述\n第二行"}};
beforeEach(()=>{vi.clearAllMocks();f.context.effectiveOrganization.id="org";f.list.mockResolvedValue({source,draftId:batch,currentRevision:2,items:[current,old]});f.read.mockImplementation((_scope,_id,revision)=>Promise.resolve(revision===1?old:current))});
afterEach(cleanup);
it("lists saved versions and keeps draft, product and batch navigation for read-only users",async()=>{
 render(<CompletionHistoryPage sourceId={id} preparationId={batch} itemId={batch}/>);
 await screen.findByText("新标题");expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
 expect(screen.getByRole("link",{name:"返回草稿"})).toHaveAttribute("href",`/workbench/supply/mine/missing/${id}?preparation=${batch}&item=${batch}`);
 expect(screen.getByRole("link",{name:"返回商品"})).toHaveAttribute("href",`/workbench/data/mine/own/${batch}`);
 expect(screen.getByRole("link",{name:"返回待补全列表"})).toHaveAttribute("href",`/workbench/supply/mine/missing?preparation=${batch}`);
 expect(screen.getByRole("link",{name:"查看 R1"})).toHaveAttribute("href",`/workbench/supply/mine/missing/${id}/history/1?preparation=${batch}&item=${batch}`);
 fireEvent.click(screen.getByRole("checkbox",{name:"选择 R1"}));fireEvent.click(screen.getByRole("checkbox",{name:"选择 R2"}));
 expect(screen.getByRole("link",{name:"对比所选版本"})).toHaveAttribute("href",`/workbench/supply/mine/missing/${id}/history/compare?preparation=${batch}&item=${batch}&from=1&to=2`);
 expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
});
it("reads an exact old revision as read-only and compares only the two candidate text fields",async()=>{
 const view=render(<CompletionHistoryPage sourceId={id} revision={1} preparationId={batch} itemId={batch}/>);
 await screen.findByText("旧标题");expect(screen.getByText("固定来源 V1")).toBeVisible();expect(screen.getByText("保存版本 R1 · 只读")).toBeVisible();expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
 view.rerender(<CompletionHistoryPage sourceId={id} from={1} to={2} preparationId={batch} itemId={batch}/>);
 expect(await screen.findByRole("table")).toBeVisible();expect(screen.getByText((_,element)=>element?.tagName==="P"&&element.textContent===current.content.description)).toBeVisible();expect(screen.getAllByText("已变化")).toHaveLength(2);
});
it("shows real empty/error states and blocks mismatched navigation or cross-draft comparisons",async()=>{
 f.list.mockResolvedValue({source,draftId:batch,currentRevision:0,items:[]});let view=render(<CompletionHistoryPage sourceId={id}/>);await screen.findByText("尚无保存历史");view.unmount();
 view=render(<CompletionHistoryPage sourceId={id} preparationId={id}/>);await screen.findByText("无法读取保存历史");expect(screen.queryByRole("link",{name:"返回草稿"})).not.toBeInTheDocument();view.unmount();
 f.read.mockResolvedValueOnce(old).mockResolvedValueOnce({...current,draftId:id});view=render(<CompletionHistoryPage sourceId={id} from={1} to={2}/>);await screen.findByText("无法读取保存历史");expect(screen.queryByRole("table")).not.toBeInTheDocument();
});
it("distinguishes identical saved text from whitespace changes",async()=>{
 f.read.mockImplementation((_scope,_id,revision)=>Promise.resolve({...old,id:revision===1?id:batch,revision}));
 const view=render(<CompletionHistoryPage sourceId={id} from={1} to={2}/>);
 await screen.findByText("标题和描述没有变化。");expect(screen.getAllByText("未变化")).toHaveLength(2);view.unmount();
 f.read.mockImplementation((_scope,_id,revision)=>Promise.resolve({...old,id:revision===1?id:batch,revision,content:{title:"",description:revision===1?"":"\n"}}));
 render(<CompletionHistoryPage sourceId={id} from={1} to={2}/>);await screen.findByText("已变化");expect(screen.getByText("仅含空白文本 · 1 个字符")).toBeVisible();expect(screen.queryByText("标题和描述没有变化。")).not.toBeInTheDocument();
});
it("keeps cross-page selections and removes old content immediately on scope or navigation changes",async()=>{
 f.list.mockResolvedValueOnce({source,draftId:batch,currentRevision:2,items:[current],nextBefore:2}).mockResolvedValueOnce({source,draftId:batch,currentRevision:2,items:[old]});
 const view=render(<CompletionHistoryPage sourceId={id} preparationId={batch}/>);await screen.findByText("新标题");fireEvent.click(screen.getByRole("checkbox",{name:"选择 R2"}));fireEvent.click(screen.getByRole("button",{name:"查看更早保存"}));await screen.findByText("旧标题");fireEvent.click(screen.getByRole("checkbox",{name:"选择 R1"}));expect(screen.getByRole("link",{name:"对比所选版本"})).toBeVisible();
 let resolve!:(value:unknown)=>void;f.list.mockReturnValue(new Promise(r=>{resolve=r}));f.context.effectiveOrganization.id="other-org";view.rerender(<CompletionHistoryPage sourceId={id} preparationId={batch}/>);expect(screen.queryByText("旧标题")).not.toBeInTheDocument();
 view.rerender(<CompletionHistoryPage sourceId={id} preparationId={id}/>);resolve({source,draftId:batch,currentRevision:2,items:[old]});await waitFor(()=>expect(screen.queryByText("旧标题")).not.toBeInTheDocument());
});
