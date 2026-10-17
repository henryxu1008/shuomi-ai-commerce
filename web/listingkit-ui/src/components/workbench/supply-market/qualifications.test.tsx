import {cleanup,render,screen,waitFor,act} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {Qualifications} from "./qualifications";
import {MarketAPIError} from "@/lib/api/supply-market";
import {useState} from "react";
import {SelectedApplicationEditor} from "./selected-application-editor";
import {emptySupply,type SelectedDraft} from "./selected-batch-intent";
const state=vi.hoisted(()=>({upload:vi.fn(),context:{user:{id:"u"},effectiveOrganization:{id:"o"},isLoading:false,isSwitching:false,error:null,blockingError:null,registerOrganizationSwitchGuard:()=>()=>{}}}));
vi.mock("@/lib/api/supply-market",async original=>({...await original<object>(),uploadQualification:state.upload}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
const scope={userId:"u",organizationId:"o"},bytes=new Uint8Array([255,216,255,224]);
const file=()=>{const value=new File([bytes],"qualification.jpg",{type:"image/jpeg"});Object.defineProperty(value,"arrayBuffer",{value:async()=>bytes.buffer});return value};
beforeEach(()=>{localStorage.clear();state.upload.mockReset();Object.defineProperty(navigator,"locks",{configurable:true,value:{request:async(_name:string,_options:unknown,fn:(lock:unknown)=>unknown)=>fn({})}})});
afterEach(cleanup);
it("keeps an unresolved file bound to the original item namespace and never delivers it to a replacement editor",async()=>{
  let finish!:(value:unknown)=>void;state.upload.mockImplementation(()=>new Promise(resolve=>{finish=resolve}));
  const onA=vi.fn(),onB=vi.fn(),locked=vi.fn();const user=userEvent.setup();
  const h=render(<Qualifications key="A" namespace="qualification-item-A" scope={scope} ids={[]} onIDs={onA} onLocked={locked}/>);
  await user.upload(screen.getByLabelText("选择资质附件"),file());await user.click(screen.getByRole("button",{name:"保存附件"}));
  await waitFor(()=>expect(state.upload).toHaveBeenCalledTimes(1));
  const original=localStorage.getItem('resource.pending:["u","o","supply-market","qualification-item-A"]');expect(original).not.toBeNull();
  h.rerender(<Qualifications key="B" namespace="qualification-item-B" scope={scope} ids={[]} onIDs={onB} onLocked={locked}/>);
  expect(screen.queryByText(/原附件结果待确认/)).not.toBeInTheDocument();
  const hash=Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256",bytes)),n=>n.toString(16).padStart(2,"0")).join("");
  await act(async()=>finish({id:crypto.randomUUID(),sha256:hash,size:bytes.length,contentType:"image/jpeg",createdAt:"2026-10-17T00:00:00Z"}));
  expect(onA).not.toHaveBeenCalled();expect(onB).not.toHaveBeenCalled();
  expect(localStorage.getItem('resource.pending:["u","o","supply-market","qualification-item-A"]')).toBe(original);
  h.rerender(<Qualifications key="A" namespace="qualification-item-A" scope={scope} ids={[]} onIDs={onA} onLocked={locked}/>);
  expect(screen.getByText(/原附件结果待确认/)).toBeVisible();
});
it("preserves the existing common namespace for a pending upload",async()=>{
  state.upload.mockRejectedValue(new MarketAPIError("OUTCOME_UNKNOWN",502));const user=userEvent.setup();
  render(<Qualifications scope={scope} ids={[]} onIDs={vi.fn()} onLocked={vi.fn()}/>);
  await user.upload(screen.getByLabelText("选择资质附件"),file());await user.click(screen.getByRole("button",{name:"保存附件"}));
  await screen.findByText(/原附件结果待确认/);
  expect(localStorage.getItem('resource.pending:["u","o","supply-market","qualification"]')).not.toBeNull();
});
it("observes every selected item's pending upload and offers its exact recovery target",async()=>{
  const ids=[crypto.randomUUID(),crypto.randomUUID()],drafts:SelectedDraft[]=ids.map((id,index)=>({title:"商品"+index,selection:{itemId:id,expectedRevision:1,originalPublicationId:"publication",originalVersion:"1",effectiveVersion:"2"},supply:emptySupply(),fileIds:[],inheritLocation:true,inheritAfterSale:true,useCommonFiles:true}));
  const review=vi.fn();
  function Editor(){const [locked,setLocked]=useState(false);return <SelectedApplicationEditor scope={scope} drafts={drafts} onDraft={vi.fn()} common={{province:"",city:"",afterSaleNote:""}} onCommon={vi.fn()} commonFiles={[]} onCommonFiles={vi.fn()} locked={false} uploadLocked={locked} onUploadLocked={setLocked} onReview={review} onBack={vi.fn()}/>}
  render(<Editor/>);const user=userEvent.setup();await user.selectOptions(screen.getByRole("combobox",{name:"正在编辑商品"}),ids[1]!);
  await act(async()=>{localStorage.setItem(`resource.pending:${JSON.stringify(["u","o","supply-market","qualification-item-"+ids[0]])}`,JSON.stringify({key:crypto.randomUUID(),hash:"a".repeat(64),size:4,type:"image/jpeg"}));window.dispatchEvent(new Event("storage"))});
  expect(screen.getByRole("button",{name:"核对本批申请"})).toBeDisabled();expect(screen.getByRole("combobox",{name:"正在编辑商品"})).toBeDisabled();
  await user.click(screen.getByRole("button",{name:"继续处理 商品0 的原附件"}));
  expect(screen.getByRole("combobox",{name:"正在编辑商品"})).toHaveValue(ids[0]);expect(screen.getByText(/原附件结果待确认/)).toBeVisible();expect(review).not.toHaveBeenCalled();
});
