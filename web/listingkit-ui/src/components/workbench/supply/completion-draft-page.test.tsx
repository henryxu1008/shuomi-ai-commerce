import {cleanup,fireEvent,render,screen,waitFor,within} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {SupplyAPIError} from "@/lib/api/supply-chain";
import {CompletionDraftPage} from "./completion-draft-page";
const f=vi.hoisted(()=>({context:{user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.supply.read","workbench.supply.manage"],supplyIntentReady:true,pendingSupplyIntent:null as unknown,setPendingSupplyIntent:vi.fn<(intent:unknown)=>boolean>(),registerOrganizationSwitchGuard:()=>()=>{}},source:vi.fn(),read:vi.fn(),save:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),readSupplySource:f.source,readSupplyCompletionDraft:f.read,supplyCommand:f.save}));
const id="11111111-1111-4111-8111-111111111111",preparation="22222222-2222-4222-8222-222222222222";
const source={id,preparationId:preparation,collectionItemId:preparation,collectionRevision:1,source:{kind:"own",version:"1",productKey:"fixed-own",publicationId:preparation}};
const detail={source,images:[],product:{title:"固定原商品V1",description:"原始描述",attributes:[{name:"材质",value:"棉"}],images:[],variants:[{source_id:"red",sku:"RED",title:"红色常规款",price:{currency:"CNY",amount:0},stock:0}]}};
const record={id,draftId:preparation,revision:1,source,content:{title:"候选内容R1",description:"候选描述"},status:"pending_completion",createdAt:"2026-10-14T00:00:00Z"};
beforeEach(()=>{vi.clearAllMocks();f.context.pendingSupplyIntent=null;f.context.effectiveOrganization.id="org";f.context.setPendingSupplyIntent.mockImplementation(v=>{f.context.pendingSupplyIntent=v;return true});f.source.mockResolvedValue(detail);f.read.mockRejectedValue(new SupplyAPIError("NOT_FOUND",404));f.save.mockResolvedValue({record,replayed:false})});
afterEach(cleanup);
it("independently edits and saves candidate text while inspecting the fixed own snapshot",async()=>{
 render(<CompletionDraftPage sourceId={id}/>);const title=await screen.findByRole("textbox",{name:"候选标题"});expect(title).toHaveValue("固定原商品V1");expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
 fireEvent.click(screen.getByText("查看固定原始商品资料"));expect(within(screen.getByRole("region",{name:"选中SKU资料"})).getByText("红色常规款")).toBeVisible();expect(screen.getByText("CNY 0")).toBeVisible();expect(screen.getByText("0 件")).toBeVisible();expect(screen.getByText("棉")).toBeVisible();
 fireEvent.change(title,{target:{value:record.content.title}});fireEvent.change(screen.getByRole("textbox",{name:"候选描述"}),{target:{value:record.content.description}});f.read.mockResolvedValue(record);fireEvent.click(screen.getByRole("button",{name:"保存待补全草稿"}));
 await waitFor(()=>expect(f.save).toHaveBeenCalledWith(expect.objectContaining({route:"save-completion-draft",command:{sourceId:id,expectedRevision:0,content:record.content}}),expect.any(AbortSignal)));
 await screen.findByText(/^未绑定店铺 · 已保存 R1/);expect(f.source).toHaveBeenCalledWith({userId:"actor",organizationId:"org"},id,expect.any(AbortSignal));
});
it("refuses candidate content whose retained source differs from the fixed detail",async()=>{
 f.read.mockResolvedValue({...record,source:{...source,source:{...source.source,version:"2"}}});render(<CompletionDraftPage sourceId={id}/>);
 await screen.findByText("草稿或原资料读取失败");expect(screen.queryByRole("textbox",{name:"候选标题"})).not.toBeInTheDocument();expect(screen.queryByText(record.content.title)).not.toBeInTheDocument();
});
