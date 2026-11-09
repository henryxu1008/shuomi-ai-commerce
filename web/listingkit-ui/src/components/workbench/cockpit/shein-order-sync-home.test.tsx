import {act,cleanup,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {afterEach,expect,it,vi} from "vitest";
import {CockpitOverview} from "./cockpit-overview";
import {cockpitRequest} from "@/lib/api/operations-cockpit";
import {readSavedOrderAmount} from "@/lib/api/shein-order-amount";
import {observationRequest} from "@/lib/api/store-observations";

vi.mock("@/lib/api/operations-cockpit",async original=>({...await original<typeof import("@/lib/api/operations-cockpit")>(),cockpitRequest:vi.fn()}));
vi.mock("@/lib/api/shein-order-amount",()=>({readSavedOrderAmount:vi.fn()}));
vi.mock("@/lib/api/store-observations",async original=>({...await original<typeof import("@/lib/api/store-observations")>(),observationRequest:vi.fn()}));
vi.mock("@/components/ui/chart",()=>({EChart:()=>null}));
const scope={organizationId:"org",userId:"user"};
const ids=["123e4567-e89b-42d3-a456-426614174000","123e4567-e89b-42d3-a456-426614174001","123e4567-e89b-42d3-a456-426614174002","123e4567-e89b-42d3-a456-426614174003"];
const access={goalsRead:false,goalsCreate:false,goalsManage:false,storesRead:false,factsWrite:false,alertsRead:false,adviceRead:false};
const source={syncId:ids[3],observedAt:"2026-11-09T00:00:00Z",range:{start:"2026-10-10T00:00:00+08:00",end:"2026-11-08T23:59:59+08:00"}};
const clients:QueryClient[]=[];
function mount(){
 const client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);
 return render(<QueryClientProvider client={client}><CockpitOverview scope={scope} enabled permissions={["workbench.store.orders.read","workbench.store.orders.sync"]}/></QueryClientProvider>);
}
function configure(){
 vi.mocked(cockpitRequest).mockResolvedValue({access,today:"2026-11-09",stores:ids.map((id,i)=>({id,name:["半托","普通自运营","全托","POP-US"][i],platform:"shein",region:"US",status:"active"}))});
 vi.mocked(observationRequest).mockImplementation(async path=>path.startsWith("supply-orders/observation")?{storeId:new URLSearchParams(path.split("?")[1]).get("storeId"),status:path.includes(ids[2])?"not_synced":"not_applicable",canSync:false,savedOrderIds:[]}:{canSync:true});
}
afterEach(()=>{cleanup();clients.splice(0).forEach(c=>c.clear());sessionStorage.clear();vi.mocked(cockpitRequest).mockReset();vi.mocked(observationRequest).mockReset();vi.mocked(readSavedOrderAmount).mockReset();});

it.each([0,1])("eligible SHEIN mode %s syncs the selected closed dates and updates amounts without reload",async(mode)=>{
 configure();let saved=false;let input:unknown;
 vi.mocked(readSavedOrderAmount).mockImplementation(async()=>({result:saved?{status:"complete",capturedAt:"2026-11-09T00:00:00Z",source,matchedOrderCount:3,amounts:[{currency:"USD",amount:"35.00",orderCount:3}]}:{status:"coverage_incomplete",capturedAt:"2026-11-09T00:00:00Z",reason:"no_saved_sync"}}));
 vi.mocked(observationRequest).mockImplementation(async(path,_scope,_schema,_signal,write)=>{
  if(path.startsWith("supply-orders/observation"))return {storeId:ids[mode],status:"not_applicable",canSync:false,savedOrderIds:[]};
  if(path==="orders/capabilities")return {canSync:true};
  if(write){input=(write as {input:unknown}).input;saved=true;}
  return {id:ids[3],input,syncs:[{id:ids[3],storeId:ids[mode],status:"completed"}]};
 });
 mount();await userEvent.selectOptions(await screen.findByRole("combobox",{name:"店铺"}),ids[mode]);
 await screen.findByRole("region",{name:"SHEIN US 订单同步"});
 await userEvent.click(screen.getByRole("button",{name:"近30天"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"同步所选日期订单"})).toBeEnabled());
 await userEvent.click(screen.getByRole("button",{name:"同步所选日期订单"}));
 await screen.findByText("35.00");
 expect(input).toEqual({kind:"orders",stores:[ids[mode]],start:"2026-10-10T00:00:00+08:00",end:"2026-11-08T23:59:59+08:00"});
 expect(screen.queryByText("SHEIN 供货概览")).not.toBeInTheDocument();
});

it("does not infer consumer eligibility from platform, supply non-applicability or an error",async()=>{
 configure();vi.mocked(readSavedOrderAmount).mockResolvedValue({message:"已保存订单金额暂不可用"});
 mount();const shop=await screen.findByRole("combobox",{name:"店铺"});
 await userEvent.selectOptions(shop,ids[2]);await screen.findByText("SHEIN 供货概览");
 expect(screen.queryByRole("region",{name:"SHEIN US 订单同步"})).not.toBeInTheDocument();
 await userEvent.selectOptions(shop,ids[3]);await waitFor(()=>expect(screen.queryByText("SHEIN 供货概览")).not.toBeInTheDocument());
 expect(screen.queryByRole("region",{name:"SHEIN US 订单同步"})).not.toBeInTheDocument();
 expect(vi.mocked(observationRequest).mock.calls.filter(c=>c[4])).toHaveLength(0);
});

it("does not reuse a late consumer eligibility result after switching to a fully managed store",async()=>{
 configure();let release!:(value:Awaited<ReturnType<typeof readSavedOrderAmount>>)=>void;
 vi.mocked(readSavedOrderAmount).mockImplementation(async(_scope,query)=>new URLSearchParams(query).get("storeId")===ids[0]?new Promise(resolve=>{release=resolve;}):{message:"不适用消费者订单"});
 mount();const shop=await screen.findByRole("combobox",{name:"店铺"});await userEvent.selectOptions(shop,ids[0]);
 await waitFor(()=>expect(release).toBeDefined());expect(screen.queryByRole("region",{name:"SHEIN US 订单同步"})).not.toBeInTheDocument();
 await userEvent.selectOptions(shop,ids[2]);await screen.findByText("SHEIN 供货概览");
 await act(async()=>release({result:{status:"coverage_incomplete",capturedAt:"2026-11-09T00:00:00Z",reason:"no_saved_sync"}}));
 expect(screen.queryByRole("region",{name:"SHEIN US 订单同步"})).not.toBeInTheDocument();expect(vi.mocked(observationRequest).mock.calls.filter(c=>c[4])).toHaveLength(0);
});
