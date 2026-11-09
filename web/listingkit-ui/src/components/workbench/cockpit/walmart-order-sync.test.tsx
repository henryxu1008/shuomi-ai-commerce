import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {afterEach,it,expect,vi} from "vitest";
import {WalmartOrderSync} from "./walmart-order-sync";
import {observationRequest,ObservationError} from "@/lib/api/store-observations";
vi.mock("@/lib/api/store-observations",async original=>({...await original<typeof import("@/lib/api/store-observations")>(),observationRequest:vi.fn()}));
const scope={organizationId:"org",userId:"user"};
const store="123e4567-e89b-42d3-a456-426614174000";
const id="123e4567-e89b-42d3-a456-426614174001";
const permissions=["workbench.store.orders.sync"];
afterEach(()=>{cleanup();sessionStorage.clear();vi.mocked(observationRequest).mockReset();});
it("syncs explicit date/store, retains the original key after a lost response, and refreshes home on completion",async()=>{
 const client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 const invalidate=vi.spyOn(client,"invalidateQueries");
 let committed=false,reveal=false;
 vi.mocked(observationRequest).mockImplementation(async(path,_scope,_schema,_signal,write)=>{
  if(path==="orders/capabilities")return {canSync:true};
  if(write){committed=true;throw new ObservationError("DEPENDENCY_UNAVAILABLE",0);}
  if(!committed||!reveal)throw new ObservationError("NOT_FOUND",404);
  return {id,input:{kind:"orders",stores:[store],start:"2026-11-01T00:00:00+08:00",end:"2026-11-09T00:00:00+08:00"},syncs:[{id,storeId:store,status:"completed",progress:{seen:2}}]};
 });
 const tree=(startDate:string)=> <QueryClientProvider client={client}><WalmartOrderSync scope={scope} storeId={store} startDate={startDate} endDate="2026-11-08" permissions={permissions}/></QueryClientProvider>;
 const view=render(tree("2026-11-01"));
 await waitFor(()=>expect(screen.getByRole("button",{name:"同步所选日期订单"})).toBeEnabled());
 fireEvent.click(screen.getByRole("button",{name:"同步所选日期订单"}));
 await screen.findByText(/同步结果尚未确认/);
 const write=vi.mocked(observationRequest).mock.calls.find(c=>c[4])!;
 expect(write[4]).toMatchObject({input:{kind:"orders",stores:[store],start:"2026-11-01T00:00:00+08:00",end:"2026-11-09T00:00:00+08:00"}});
 view.rerender(tree("2026-10-10"));
 reveal=true;
 fireEvent.click(screen.getByRole("button",{name:"确认原同步结果"}));
 await screen.findByText(/遍历完成/);
 expect(vi.mocked(observationRequest).mock.calls.filter(c=>c[4])).toHaveLength(1);
 expect(invalidate).toHaveBeenCalledWith({queryKey:["operations-cockpit","user","org"]});
 client.clear();
});
