import {act,cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import userEvent from "@testing-library/user-event";
import {afterEach,expect,it,vi} from "vitest";
import {observationRequest} from "@/lib/api/store-observations";
import {SheinSupplyOrders} from "./shein-supply-orders";
vi.mock("@/lib/api/store-observations",async original=>({...await original<typeof import("@/lib/api/store-observations")>(),observationRequest:vi.fn()}));
const scope={organizationId:"org-a",userId:"user-a"},a="10501f2e-24bd-48b6-8a23-5172dc011111",b="10501f2e-24bd-48b6-8a23-5172dc022222";
const applicable=vi.fn();const clients:QueryClient[]=[];
afterEach(()=>{cleanup();clients.splice(0).forEach(c=>c.clear());vi.mocked(observationRequest).mockReset();applicable.mockClear();sessionStorage.clear();});
function wrapper(){const client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);return function TestQueryProvider({children}:{children:React.ReactNode}){return <QueryClientProvider client={client}>{children}</QueryClientProvider>;};}
it("switches stores with original-date parameters and retains original read intent after an uncertain response",async()=>{
 const writes:unknown[]=[];
 vi.mocked(observationRequest).mockImplementation(async(path,_scope,_schema,_signal,write)=>{
  if(write){writes.push(write);throw new Error("response lost");}
  return {storeId:path.includes(a)?a:b,status:"not_synced",savedOrderIds:[],canSync:true};
 });
 const view=render(<SheinSupplyOrders scope={scope} storeId={a} startDate="2026-11-01" endDate="2026-11-03" onApplicable={applicable}/>,{wrapper:wrapper()});
 await screen.findByText("尚无已保存批次");await userEvent.type(screen.getByRole("textbox"),"PO-A\nPO-A\nPO-B");await userEvent.click(screen.getByRole("button",{name:"读取并保存本批"}));
 await screen.findByRole("button",{name:"重试原读取"});expect(screen.getByRole("textbox")).toBeDisabled();
 await userEvent.click(screen.getByRole("button",{name:"重试原读取"}));await waitFor(()=>expect(writes).toHaveLength(2));expect(writes[1]).toEqual(writes[0]);
 expect(writes[0]).toMatchObject({input:{kind:"supply-orders",stores:[a],supplyOrderIds:["PO-A","PO-B"]}});
 view.rerender(<SheinSupplyOrders key={b} scope={scope} storeId={b} startDate="2026-11-01" endDate="2026-11-03" onApplicable={applicable}/>);await waitFor(()=>expect(observationRequest).toHaveBeenCalledWith(`supply-orders/observation?storeId=${b}&startDate=2026-11-01&endDate=2026-11-03`,scope,expect.anything(),expect.any(AbortSignal)));
 expect(await screen.findByRole("textbox")).toHaveValue("");
 expect(vi.mocked(observationRequest).mock.calls.filter(c=>!c[4]).every(c=>c[0].includes("startDate=2026-11-01&endDate=2026-11-03"))).toBe(true);
});
it("labels the saved subset, separates mother counts and keeps unknown quantities visible",async()=>{
 const known={value:"2",missingLines:0},unknown={value:null,missingLines:1};
 vi.mocked(observationRequest).mockResolvedValue({storeId:a,status:"incomplete",canSync:false,savedOrderIds:["PO-A"],source:{orderIds:["PO-A","PO-B"],missingIds:["PO-B"],notes:[]},summary:{orderCount:1,motherCount:0,unknownTypeCount:0,quantities:{ordered:known,delivered:known,received:unknown,stored:known,defective:known}}});
 render(<SheinSupplyOrders scope={scope} storeId={a} startDate="2026-11-01" endDate="2026-11-03" onApplicable={applicable}/>,{wrapper:wrapper()});
 await screen.findByText("覆盖不足：仅展示本批已保存部分");await userEvent.click(screen.getByText("批次详情",{exact:true}));expect(screen.getByText("1 行缺失或有歧义")).toBeVisible();expect(screen.getByText("待核实",{exact:true})).toBeVisible();expect(screen.queryByRole("textbox")).toBeNull();expect(applicable).toHaveBeenCalledWith(a);
});
it("resumes the same acknowledged sync without creating a new batch",async()=>{
 vi.mocked(observationRequest).mockImplementation(async(_path,_scope,_schema,_signal,write)=>write?{}:{storeId:a,status:"running",canSync:true,savedOrderIds:[],source:{syncId:b,orderIds:["PO-A"],missingIds:[],notes:[]}});
 render(<SheinSupplyOrders scope={scope} storeId={a} startDate="2026-11-01" endDate="2026-11-03" onApplicable={applicable}/>,{wrapper:wrapper()});
 await userEvent.click(await screen.findByRole("button",{name:"继续原读取"}));
 expect(observationRequest).toHaveBeenCalledWith(`supply-orders/syncs/${b}/ensure`,scope,expect.anything(),undefined,{});
 expect(vi.mocked(observationRequest).mock.calls.some(c=>c[0]==="supply-orders/syncs")).toBe(false);
});
it("hides cached facts and read controls when the same shop's refresh fails",async()=>{
 const quantity={value:"12",missingLines:0};
 vi.mocked(observationRequest).mockResolvedValue({storeId:a,status:"saved",canSync:true,savedOrderIds:["PO-A"],source:{syncId:b,orderIds:["PO-A"],missingIds:[],notes:[]},summary:{orderCount:1,motherCount:0,unknownTypeCount:0,quantities:{ordered:quantity,delivered:quantity,received:quantity,stored:quantity,defective:quantity}}});
 const view=render(<SheinSupplyOrders scope={scope} storeId={a} startDate="2026-11-01" endDate="2026-11-03" onApplicable={applicable}/>,{wrapper:wrapper()});
 await screen.findByText("本批单号均已保存");expect(view.container.querySelectorAll("dd").length).toBe(8);
 vi.mocked(observationRequest).mockRejectedValue(new Error("binding or authorization changed"));
 await act(async()=>{await clients.at(-1)!.refetchQueries({queryKey:["operations-cockpit",scope.userId,scope.organizationId,"shein-supply-orders",a]});});
 await screen.findByRole("alert");expect(screen.queryByText("本批单号均已保存")).toBeNull();expect(view.container.querySelectorAll("dd").length).toBe(0);expect(screen.queryByRole("textbox")).toBeNull();
});
it("rejects a long multibyte batch before storing intent and keeps input editable",async()=>{
 vi.mocked(observationRequest).mockResolvedValue({storeId:a,status:"not_synced",canSync:true,savedOrderIds:[]});
 render(<SheinSupplyOrders scope={scope} storeId={a} startDate="2026-11-01" endDate="2026-11-03" onApplicable={applicable}/>,{wrapper:wrapper()});
 const input=await screen.findByRole("textbox");
 fireEvent.change(input,{target:{value:Array.from({length:100},(_,i)=>"采".repeat(125)+String(i).padStart(3,"0")).join("\n")}});
 await userEvent.click(screen.getByRole("button",{name:"读取并保存本批"}));await screen.findByRole("alert");
 expect(screen.getByRole("textbox")).not.toBeDisabled();expect(sessionStorage.length).toBe(0);expect(vi.mocked(observationRequest).mock.calls.some(c=>!!c[4])).toBe(false);
});
it("joins the existing cockpit refresh using source date labels",async()=>{
 vi.mocked(observationRequest).mockResolvedValue({storeId:a,status:"not_synced",canSync:false,savedOrderIds:[]});
 render(<SheinSupplyOrders scope={scope} storeId={a} startDate="2026-11-01" endDate="2026-11-03" onApplicable={applicable}/>,{wrapper:wrapper()});await screen.findByText("尚无已保存批次");
 vi.mocked(observationRequest).mockResolvedValue({storeId:a,status:"incomplete",canSync:false,savedOrderIds:[]});
 await act(async()=>{await clients.at(-1)!.invalidateQueries({queryKey:["operations-cockpit",scope.userId,scope.organizationId]});});
 await screen.findByText("覆盖不足：仅展示本批已保存部分");expect(observationRequest).toHaveBeenCalledTimes(2);expect(vi.mocked(observationRequest).mock.calls[1][0]).toBe(`supply-orders/observation?storeId=${a}&startDate=2026-11-01&endDate=2026-11-03`);
});

it("updates source date groups together and hides old dates while another range loads",async()=>{
 const quantity={value:"5",missingLines:0},zero={value:"0",missingLines:0};
 const result=(startDate:string,matched:boolean)=>({storeId:a,status:"saved",canSync:false,savedOrderIds:["PO-A"],source:{syncId:b,orderIds:["PO-A"],missingIds:[],notes:[]},summary:{orderCount:1,motherCount:0,unknownTypeCount:0,quantities:{ordered:quantity,delivered:quantity,received:quantity,stored:quantity,defective:quantity}},originalDates:{range:{startDate,endDate:"2026-11-03"},urgent:{orderIds:matched?["PO-A"]:[],ordered:matched?quantity:zero},stock:{orderIds:[],ordered:zero},undated:[],motherIds:[],unknownMotherIds:[],unknownPurchaseTypeIds:[],outsideRangeIds:matched?[]:["PO-A"]}});
 let release:((value:unknown)=>void)|undefined;
 vi.mocked(observationRequest).mockImplementation(async path=>path.includes("startDate=2026-11-01")?result("2026-11-01",true):new Promise(resolve=>{release=resolve}));
 const view=render(<SheinSupplyOrders scope={scope} storeId={a} startDate="2026-11-01" endDate="2026-11-03" onApplicable={applicable}/>,{wrapper:wrapper()});
 await screen.findByRole("region",{name:"已保存供货单原文日期观察"});expect(screen.getByText("本批全部数量（不按日期筛选）")).not.toBeVisible();
 view.rerender(<SheinSupplyOrders scope={scope} storeId={a} startDate="2026-11-02" endDate="2026-11-03" onApplicable={applicable}/>);
 expect(screen.queryByRole("region",{name:"已保存供货单原文日期观察"})).toBeNull();
 await waitFor(()=>expect(release).toBeDefined());await act(async()=>release!(result("2026-11-02",false)));
 await screen.findByText(/2026-11-02 至 2026-11-03/);expect(screen.getByText(/区间外 1 单/)).toBeVisible();
 expect(vi.mocked(observationRequest).mock.calls.at(-1)?.[0]).toBe(`supply-orders/observation?storeId=${a}&startDate=2026-11-02&endDate=2026-11-03`);
});

it("keeps saved-batch details and read form behind separate disclosures while retaining their controls",async()=>{
 const quantity={value:"5",missingLines:0};
 vi.mocked(observationRequest).mockResolvedValue({storeId:a,status:"saved",canSync:true,savedOrderIds:["PO-A"],source:{syncId:b,orderIds:["PO-A"],missingIds:[],notes:[]},summary:{orderCount:1,motherCount:0,unknownTypeCount:0,quantities:{ordered:quantity,delivered:quantity,received:quantity,stored:quantity,defective:quantity}}});
 render(<SheinSupplyOrders scope={scope} storeId={a} startDate="2026-11-01" endDate="2026-11-03" onApplicable={applicable}/>,{wrapper:wrapper()});
 await screen.findByText("本批单号均已保存");
 expect(screen.getByLabelText("读取并保存采购单（每批最多 200 个单号，一行一个）")).not.toBeVisible();
 await userEvent.click(screen.getByText("读取采购单",{exact:true}));expect(screen.getByRole("textbox")).toBeVisible();
 await userEvent.click(screen.getByText("读取采购单",{exact:true}));expect(screen.getByRole("textbox")).not.toBeVisible();
 await userEvent.click(screen.getByText("批次详情",{exact:true}));expect(screen.getByText("本批全部数量（不按日期筛选）")).toBeVisible();expect(screen.getByText("查看业务时间原始值",{exact:true})).toBeVisible();
});
