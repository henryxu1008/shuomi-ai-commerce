import {render,screen,cleanup,fireEvent,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {StorePublicationDetail} from "./store-publication-detail";
import type {SupplyIntent} from "@/lib/api/supply-chain";
const f=vi.hoisted(()=>({read:vi.fn(),send:vi.fn(),commandRead:vi.fn(),pending:null as SupplyIntent|null}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.supply.read"],supplyIntentReady:true,pendingSupplyIntent:f.pending,setPendingSupplyIntent:(intent:SupplyIntent|null)=>{f.pending=intent;return true},registerOrganizationSwitchGuard:()=>()=>{}})}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),readSupplyPublication:f.read,supplyCommand:f.send,readSupplyCommand:f.commandRead}));
vi.mock("./shein-uploaded-product",()=>({SheinUploadedProduct:({management}:{management:boolean})=><p>{management?"原回执店铺商品详情":"只读"}</p>}));
beforeEach(()=>{vi.clearAllMocks();f.pending=null});
afterEach(cleanup);
it("opens the exact original receipt without requiring a synced store observation",async()=>{
 f.read.mockResolvedValue({sourceId:"source",storeId:"store",recordId:"record",receiptId:"receipt",product:{spu_name:"spu"}});
 render(<StorePublicationDetail sourceId="source" storeId="store" recordId="record" receiptId="receipt"/>);
 expect(await screen.findByText("原回执店铺商品详情")).toBeInTheDocument();expect(f.read).toHaveBeenCalledWith({userId:"actor",organizationId:"org"},"source","store",expect.any(AbortSignal));
});
it("rejects a different receipt rather than substituting the current object",async()=>{
 f.read.mockResolvedValue({sourceId:"source",storeId:"store",recordId:"record",receiptId:"other",product:{spu_name:"spu"}});
 render(<StorePublicationDetail sourceId="source" storeId="store" recordId="record" receiptId="receipt"/>);
 expect(await screen.findByRole("alert")).toHaveTextContent("原上传对象");expect(screen.queryByText("原回执店铺商品详情")).not.toBeInTheDocument();
});

const original={sourceId:"345fab2e-b0d9-5bb7-968a-54179873a8ba",storeId:"12c2116f-086d-467d-89e1-bd54218241a4",recordId:"376343ab-5588-585d-8600-bda4293108e2",receiptId:"01a1bac6-8241-73ae-acae-70bc4b7fd5b6"};
const key="ec24fafc-0f20-45f7-9848-808b37ca6e0f",attemptId="01a1bb21-960d-781f-87c6-b91166aeea0c";
function restoredIntent():SupplyIntent{return {userId:"actor",organizationId:"org",key,route:"shein-shelf-confirm",command:{sourceId:original.sourceId,storeId:original.storeId,receiptId:original.receiptId,skc:"synthetic-skc-local-001",token:"5aabdc5d-4c68-4071-a980-2e2d24af1089"}}}
it.each(["succeeded","outcome_unknown"] as const)("shows the original-key %s result after refreshing, without resending",async status=>{
 f.pending=restoredIntent();f.read.mockResolvedValue({...original,product:{spu_name:"synthetic-spu-local-001"}});
 const message=status==="succeeded"?"上架请求成功；当前平台状态与库存另行回读":"原上架请求结果待核实；请查询平台结果，不会重新发送";
 f.commandRead.mockResolvedValue({key,attemptId,sourceId:original.sourceId,storeId:original.storeId,receiptId:original.receiptId,skc:"synthetic-skc-local-001",status,message});
 render(<StorePublicationDetail {...original}/>);fireEvent.click(await screen.findByRole("button",{name:"核实原操作"}));
 const result=await screen.findByRole("status",{name:"原上架操作结果"});expect(result).toHaveTextContent(message);expect(result).toHaveTextContent(key);expect(result).toHaveTextContent(attemptId);
 expect(f.commandRead).toHaveBeenCalledTimes(1);expect(f.commandRead).toHaveBeenCalledWith(restoredIntent(),expect.any(AbortSignal));expect(f.send).not.toHaveBeenCalled();
});
it.each(["receiptId","key"] as const)("does not display a recovered result for another %s",async field=>{
 f.pending=restoredIntent();f.read.mockResolvedValue({...original,product:{spu_name:"synthetic-spu-local-001"}});
 f.commandRead.mockResolvedValue({key,attemptId,sourceId:original.sourceId,storeId:original.storeId,receiptId:original.receiptId,skc:"synthetic-skc-local-001",status:"succeeded",message:"other operation",[field]:"11111111-1111-4111-8111-111111111111"});
 render(<StorePublicationDetail {...original}/>);fireEvent.click(await screen.findByRole("button",{name:"核实原操作"}));await waitFor(()=>expect(f.commandRead).toHaveBeenCalledTimes(1));
 expect(screen.queryByRole("status",{name:"原上架操作结果"})).not.toBeInTheDocument();expect(f.send).not.toHaveBeenCalled();
});
