import {cleanup,render,screen,waitFor} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {afterEach,it,expect,vi} from "vitest";
import {StoreOrderSync} from "./store-order-sync";
import {observationRequest} from "@/lib/api/store-observations";
vi.mock("@/lib/api/store-observations",async original=>({...await original<typeof import("@/lib/api/store-observations")>(),observationRequest:vi.fn()}));
afterEach(()=>{cleanup();sessionStorage.clear();vi.mocked(observationRequest).mockReset();});
it.each(["tiktokshop","temu"] as const)("isolates %s recovery from an unresolved Walmart command in the same store context",async(platform)=>{
 const client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 const store="123e4567-e89b-42d3-a456-426614174000";
 sessionStorage.setItem(JSON.stringify(["walmart-order-sync","org","user",store]),JSON.stringify({key:"123e4567-e89b-42d3-a456-426614174001",input:{kind:"orders",stores:[store],start:"2026-11-01T00:00:00+08:00",end:"2026-11-09T00:00:00+08:00"}}));
 vi.mocked(observationRequest).mockImplementation(async path=>{if(path!=="orders/capabilities")throw new Error("Walmart command crossed platform");return {canSync:true}});
 render(<QueryClientProvider client={client}><StoreOrderSync platform={platform} scope={{organizationId:"org",userId:"user"}} storeId={store} startDate="2026-11-01" endDate="2026-11-08" permissions={["workbench.store.orders.sync"]}/></QueryClientProvider>);
 await waitFor(()=>expect(screen.getByRole("button",{name:"同步所选日期订单"})).toBeEnabled());
 expect(screen.getByRole("region",{name:platform==="temu"?"Temu US 订单同步":"TikTok Shop US 订单同步"})).toBeVisible();
 expect(vi.mocked(observationRequest).mock.calls.map(c=>c[0])).toEqual(["orders/capabilities"]);
 client.clear();
});
