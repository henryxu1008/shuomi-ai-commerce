import {afterEach,expect,it,vi} from "vitest";
import {cleanup,render,screen,fireEvent,waitFor} from "@testing-library/react";
import {SheinPublicationPanel} from "./shein-publication-panel";
import {readSheinPublicationStatus,prepareSheinShelf} from "@/lib/api/supply-chain";
import {sheinPublicationStatusSchema} from "@/lib/contracts/shein-publication";
import type {SupplyCommandState} from "./use-supply-command";
vi.mock("@/lib/api/supply-chain",()=>({readSheinPublicationStatus:vi.fn(),prepareSheinShelf:vi.fn()}));
afterEach(()=>{cleanup();vi.resetAllMocks()});

it.each([0,7])("shows fully managed PI %s only as a reference and keeps usability unknown",async(quantity)=>{
 const value=make("fully_managed");value.stockType="PI";value.stockComplete=false;value.stockError="全托管履约库存类型未核实，PI库存仅供参考";value.skcs[0].skus[0].reportedUsable=quantity;
 vi.mocked(readSheinPublicationStatus).mockResolvedValue(value);
 render(<SheinPublicationPanel scope={{organizationId:"org-a",userId:"actor-a"}} sourceId={id} storeId={other} receiptId={id} storeName="美国店" command={command()} canSubmit/>);
 fireEvent.click(screen.getByRole("button",{name:"查询平台结果"}));await screen.findByText("SKU 可售条件 · PI 参考库存");
 expect(screen.getByText("库存未知（履约库存类型未核实）")).toBeVisible();
 expect(screen.getByText("PI 参考库存汇总："+quantity+"，履约库存类型未核实。")).toBeVisible();
 expect(screen.queryByText("美国可用库存为 0")).toBeNull();expect(screen.queryByText("美国可用库存："+quantity)).toBeNull();
 expect(screen.queryByText(/本商品库存条件已完整读取/)).toBeNull();expect(screen.queryByRole("button",{name:"上架到美国站"})).toBeNull();
});
it.each(["self_operated","semi_managed"])("retains qualified VI stock for %s",async(mode)=>{
 const value=make(mode);value.skcs[0].skus[0]={...value.skcs[0].skus[0],usable:0,reportedUsable:0,inventoryScopeKnown:true};
 vi.mocked(readSheinPublicationStatus).mockResolvedValue(value);
 render(<SheinPublicationPanel scope={{organizationId:"org-a",userId:"actor-a"}} sourceId={id} storeId={other} receiptId={id} storeName="美国店" command={command()} canSubmit/>);
 fireEvent.click(screen.getByRole("button",{name:"查询平台结果"}));await screen.findByText("美国可用库存为 0");
 expect(screen.queryByText(/PI 参考库存/)).toBeNull();expect(screen.getByText(/本商品库存条件已完整读取/)).toBeVisible();
});


it("shows the original fully managed scheduled plan without adding an action",async()=>{
 const value=make("fully_managed");value.skcs[0].shelfWay="2";value.skcs[0].hopeOnSaleDate="2026-10-20 08:00:00";
 vi.mocked(readSheinPublicationStatus).mockResolvedValue(value);
 render(<SheinPublicationPanel scope={{organizationId:"org-a",userId:"actor-a"}} sourceId={id} storeId={other} receiptId={id} storeName="美国店" command={command()} canSubmit/>);
 fireEvent.click(screen.getByRole("button",{name:"查询平台结果"}));await screen.findByText(/预约上架（北京时间）：2026-10-20 08:00:00/);
 expect(screen.queryByRole("button",{name:"上架到美国站"})).toBeNull();
});

const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222";
const make=(mode="self_operated",unknown=false)=>sheinPublicationStatusSchema.parse({sourceId:id,storeId:other,receiptId:id,spu:"spu-a",version:"v1",mode,observedAt:"2026-10-19T00:00:00Z",stockType:"VI",stockOffset:0,stockTotal:1,stockComplete:true,skcs:[{skc:"skc-a",auditState:2,auditLabel:"审核通过",reasons:[],shelfState:unknown?1:0,shelfWay:mode==="semi_managed"?"2":"1",hopeOnSaleDate:mode==="semi_managed"?"2099-01-01 08:00:00":"",shelfRequire:"0",canShelf:mode==="self_operated"&&!unknown,blockingReason:unknown?"原请求待核实":"等待平台",attempt:unknown?{id,key:id,status:"outcome_unknown"}:undefined,skus:[{code:"sku-a",mallState:1,usable:null,reportedUsable:12,inventoryScopeKnown:false}]}]});
const command=(execute=vi.fn())=>({execute,busy:false,pending:null,ready:true,error:null,foreign:false,verify:vi.fn(),retry:vi.fn()}) as unknown as SupplyCommandState;
it("requires explicit query and confirmation before sending the original prepared key",async()=>{
 const execute=vi.fn().mockResolvedValue({key:id,attemptId:id,sourceId:id,storeId:other,receiptId:id,skc:"skc-a",status:"succeeded",message:"请求成功"});
 vi.mocked(readSheinPublicationStatus).mockResolvedValue(make());
 vi.mocked(prepareSheinShelf).mockResolvedValue({key:id,sourceId:id,storeId:other,receiptId:id,spu:"spu-a",version:"v1",skc:"skc-a",site:"shein-us",mode:"self_operated",token:other,expiresAt:new Date(Date.now()+300000).toISOString(),shelfWay:"",hopeOnSaleDate:""});
 render(<SheinPublicationPanel scope={{organizationId:"org-a",userId:"actor-a"}} sourceId={id} storeId={other} receiptId={id} storeName="美国店" command={command(execute)} canSubmit/>);
 expect(readSheinPublicationStatus).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole("button",{name:"查询平台结果"}));await screen.findByText("审核通过");
 expect(execute).not.toHaveBeenCalled();expect(screen.getByText(/库存未知/)).toBeVisible();
 fireEvent.click(screen.getByRole("button",{name:"上架到美国站"}));await screen.findByRole("button",{name:"确认首次上架"});
 expect(execute).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole("button",{name:"确认首次上架"}));
 await waitFor(()=>expect(execute).toHaveBeenCalledWith("shein-shelf-confirm",{sourceId:id,storeId:other,receiptId:id,skc:"skc-a",token:other},id));
});
it.each(["fully_managed","semi_managed"])("does not offer a forbidden manual action for %s",async(mode)=>{
 vi.mocked(readSheinPublicationStatus).mockResolvedValue(make(mode));
 render(<SheinPublicationPanel scope={{organizationId:"org-a",userId:"actor-a"}} sourceId={id} storeId={other} receiptId={id} storeName="美国店" command={command()} canSubmit/>);
 fireEvent.click(screen.getByRole("button",{name:"查询平台结果"}));await screen.findByText("审核通过");
 expect(screen.queryByRole("button",{name:"上架到美国站"})).toBeNull();
});
it("keeps current shelf state separate from the original UNKNOWN",async()=>{
 vi.mocked(readSheinPublicationStatus).mockResolvedValue(make("self_operated",true));
 render(<SheinPublicationPanel scope={{organizationId:"org-a",userId:"actor-a"}} sourceId={id} storeId={other} receiptId={id} storeName="美国店" command={command()} canSubmit/>);
 fireEvent.click(screen.getByRole("button",{name:"查询平台结果"}));await screen.findByText("当前已上架，原请求结果待核实");
 expect(screen.queryByRole("button",{name:"上架到美国站"})).toBeNull();
});

