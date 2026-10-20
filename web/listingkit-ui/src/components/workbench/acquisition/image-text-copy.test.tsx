import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {ImageTextCopy} from "./image-text-copy";
import {contentProposalFixture} from "@/test/product-content-review-fixture";
import {parseProductTitleProposal} from "@/lib/api/product-title-review";
import {ProductTitleReviewError} from "@/lib/api/product-title-review-client";

const f=vi.hoisted(()=>({read:vi.fn(),decide:vi.fn(),panel:vi.fn(),context:{roles:[] as string[],permissions:["local_agent.write","listingkit.admin.write"]}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("@/lib/api/product-title-review-client",async original=>({...await original<typeof import("@/lib/api/product-title-review-client")>(),fetchProductTitleProposal:f.read,decideProductTitleProposal:f.decide}));
vi.mock("./product-agent-panel",()=>({ProductAgentPanel:(props:unknown)=>{f.panel(props);return <p>原文本预算确认与执行入口</p>}}));
function proposal(state="accepted",language="en",title="Fabric storage box"){
 const p=contentProposalFixture(),selection={targetLanguage:language,title:true,sellingPoints:true};
 const values=[{field:"title",value:title},{field:"selling_points",value:'["Holds everyday items"]'}];
 return parseProductTitleProposal({...p,state,revision:"2",content:{...p.content,selection,before:values,after:values},execution:{...p.execution,ContentSelection:selection}})!;
}
const p=proposal();
const base={scope:{userId:"owner",organizationId:"B",kind:"acquisition" as const,contextId:p.proposal_id},source:{ContextKind:"acquisition" as const,ProductID:p.input.product_key,OperationID:p.proposal_id,OriginalPublicationID:"publication",OriginalVersion:"1",EffectiveVersion:"1"},language:"en",templateKey:"template:1",taskIds:["detail"],disabled:false};
const storage=`image-text:owner:B:acquisition:${p.proposal_id}:1:template:1:en`;
beforeEach(()=>{vi.clearAllMocks();localStorage.clear();f.context.permissions=["local_agent.write","listingkit.admin.write"];f.read.mockResolvedValue(p);f.decide.mockResolvedValue({})});
afterEach(()=>{cleanup();localStorage.clear()});
async function read(){fireEvent.change(screen.getByRole("combobox",{name:"图片文字方式"}),{target:{value:"reviewed"}});fireEvent.change(screen.getByLabelText("文案审核编号"),{target:{value:p.proposal_id}});fireEvent.click(screen.getByRole("button",{name:"读取审核译文"}));await screen.findByText("本次图片文案 · 最多两条")}
it("freezes only an exact accepted Review reference after explicit human confirmation, with no product Apply",async()=>{
 const changed=vi.fn();const view=render(<ImageTextCopy {...base} onChange={changed}/>);await read();
 expect(screen.getByText("目标语言：英语（en）")).toBeVisible();
 expect(screen.queryByRole("button",{name:/应用到商品|采用到商品/})).toBeNull();
 expect(f.decide).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole("checkbox",{name:"确认目标语言图片文案"}));
 expect(changed).toHaveBeenLastCalledWith({mode:"reviewed",review:{proposalId:p.proposal_id,revision:"2"},taskIds:["detail"]});
 view.rerender(<ImageTextCopy {...base} taskIds={["detail","main"]} onChange={changed}/>);
 expect(screen.getByRole("checkbox",{name:"确认目标语言图片文案"})).not.toBeChecked();
 expect(f.read).toHaveBeenCalledTimes(1);
});
it.each(["pending","wrong-language","too-long","automatic"])("blocks %s copy without truncation or source-language fallback",async kind=>{
 let value=proposal(kind==="pending"?"pending":"accepted",kind==="wrong-language"?"zh":"en",kind==="too-long"?"x".repeat(33):"Fabric storage box");
 if(kind==="automatic")value={...value,execution_mode:{...value.execution_mode!,mode:"auto_approve"}};
 f.read.mockResolvedValue(value);render(<ImageTextCopy {...base} onChange={vi.fn()}/>);await read();
 expect(screen.getByRole("checkbox",{name:"确认目标语言图片文案"})).toBeDisabled();
 expect(f.panel).not.toHaveBeenCalled();expect(f.decide).not.toHaveBeenCalled();
});
it("confirms a text-task subset while keeping the main picture text-free",async()=>{
 const changed=vi.fn();render(<ImageTextCopy {...base} taskIds={["main","detail"]} taskLabels={{main:"标准主图",detail:"详情图"}} onChange={changed}/>);await read();
 fireEvent.click(screen.getByRole("checkbox",{name:"使用文字 标准主图"}));
 fireEvent.click(screen.getByRole("checkbox",{name:"确认目标语言图片文案"}));
 expect(changed).toHaveBeenLastCalledWith({mode:"reviewed",review:{proposalId:p.proposal_id,revision:"2"},taskIds:["detail"]});
 fireEvent.click(screen.getByRole("checkbox",{name:"使用文字 详情图"}));
 expect(changed).toHaveBeenLastCalledWith(null);
 expect(screen.getByRole("checkbox",{name:"确认目标语言图片文案"})).not.toBeChecked();
 expect(screen.getByRole("checkbox",{name:"确认目标语言图片文案"})).toBeDisabled();
});
it("keeps ordinary member review permissions and allows explicit text-free images",async()=>{
 f.context.permissions=["local_agent.write"];f.read.mockResolvedValue(proposal("pending"));const changed=vi.fn();render(<ImageTextCopy {...base} onChange={changed}/>);await read();
 expect(screen.queryByRole("button",{name:"接受建议"})).toBeNull();
 expect(screen.getByText(/须由已有审核权限的成员接受/)).toBeVisible();
 fireEvent.change(screen.getByRole("combobox",{name:"图片文字方式"}),{target:{value:"none"}});
 expect(changed).toHaveBeenLastCalledWith({mode:"none"});expect(f.decide).not.toHaveBeenCalled();
});
it("retains the original review command on UNKNOWN and permits only same-key verification",async()=>{
 f.read.mockResolvedValue(proposal("pending"));f.decide.mockRejectedValue(new ProductTitleReviewError(503,"UNKNOWN",{code:"UNKNOWN",message:"unknown",requestId:"",fieldErrors:[],outcome:"unknown"},"unknown"));
 render(<ImageTextCopy {...base} onChange={vi.fn()}/>);await read();
 const accept=screen.getByRole("button",{name:"接受提案"});fireEvent.click(accept);
 await waitFor(()=>expect(f.decide).toHaveBeenCalledTimes(1));
 const raw=localStorage.getItem(storage+":review");expect(raw).toBeTruthy();
 const verify=await screen.findByRole("button",{name:"核实原审核操作"});fireEvent.click(verify);
 await waitFor(()=>expect(f.decide).toHaveBeenCalledTimes(2));
 expect(f.decide.mock.calls[1]).toEqual(f.decide.mock.calls[0]);expect(localStorage.getItem(storage+":review")).toBe(raw);
 expect(screen.getByRole("combobox",{name:"图片文字方式"})).toBeDisabled();
});
