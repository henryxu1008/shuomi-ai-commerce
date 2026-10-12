import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {render,screen,fireEvent,waitFor,cleanup} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {EnterpriseStorage} from "./enterprise-storage";
const wallet=vi.hoisted(()=>vi.fn());
const api=vi.hoisted(()=>({getStorageSummary:vi.fn(),getStorageOffers:vi.fn(),createStorageQuote:vi.fn(),createStorageOrder:vi.fn(),readStorageOrder:vi.fn()}));
vi.mock("@/lib/api/storage-capacity",()=>api);
vi.mock("@/lib/api/commercial-billing",async importOriginal=>({...await importOriginal<object>(),getCommercialWallet:wallet}));
let client:QueryClient;
beforeEach(()=>{
 sessionStorage.clear();client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.setAttribute("open","")}});Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.removeAttribute("open")}});
 Object.values(api).forEach(f=>f.mockReset());api.getStorageSummary.mockResolvedValue({organization_id:"org",summary:{more_lots:false,base_bytes:"5368709120",quota_bytes:"5368709120",occupied_bytes:"0",reserved_bytes:"0",remaining_bytes:"5368709120",overage_bytes:"0",categories:{},active_lots:[],observed_at:"2026-10-12T00:00:00Z"}});
 wallet.mockReset().mockResolvedValue({available_minor:"10000"});
 api.getStorageOffers.mockResolvedValue({organization_id:"org",items:[{offer_id:"storage-offer",storage_unit_bytes:"53687091200",unit_price_minor:"1990"}]});
 api.createStorageQuote.mockResolvedValue({quote_id:"quote",offer_id:"storage-offer",resource_quantity:"1",total_minor:"1990",storage_unit_bytes:"53687091200",expires_at:"2027-01-01T00:00:00Z"});
});
it("requotes the selected package count and blocks payment while the new quote is loading",async()=>{
 show();await waitFor(()=>expect(screen.getByRole("button",{name:"购买扩容包"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"购买扩容包"}));await screen.findByRole("button",{name:"确认支付 ¥19.90"});
 let resolve!:(value:unknown)=>void;api.createStorageQuote.mockImplementationOnce(()=>new Promise(r=>{resolve=r}));
 fireEvent.change(screen.getByLabelText("扩容包数量"),{target:{value:"2"}});expect(screen.queryByRole("button",{name:"确认支付 ¥19.90"})).not.toBeInTheDocument();expect(api.createStorageOrder).not.toHaveBeenCalled();
 resolve({quote_id:"quote-two",offer_id:"storage-offer",resource_quantity:"2",total_minor:"3980",storage_unit_bytes:"53687091200",expires_at:"2027-01-01T00:00:00Z"});
 const confirm=await screen.findByRole("button",{name:"确认支付 ¥39.80"});expect(api.createStorageQuote.mock.calls[1].slice(0,4)).toEqual(["actor","org","storage-offer","2"]);
 api.createStorageOrder.mockResolvedValue({order_id:"order-two",status:"FULFILLED"});fireEvent.click(confirm);await screen.findByText(/扩容已生效/);expect(api.createStorageOrder.mock.calls[0][2]).toBe("quote-two");
});
it("does not submit a quote that exceeds the known wallet balance",async()=>{
 wallet.mockResolvedValue({available_minor:"1989"});show();await waitFor(()=>expect(screen.getByRole("button",{name:"购买扩容包"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"购买扩容包"}));const confirm=await screen.findByRole("button",{name:"确认支付 ¥19.90"});expect(confirm).toBeDisabled();fireEvent.click(confirm);expect(api.createStorageOrder).not.toHaveBeenCalled();
});
it.each([["4294967296","1073741824","0","已达到当前容量"],["4026531840","268435456","1073741824","已达到 80%"]])("warns using both occupied and reserved bytes",async(occupied,reserved,remaining,label)=>{
 const value=await api.getStorageSummary();value.summary.occupied_bytes=occupied;value.summary.reserved_bytes=reserved;value.summary.remaining_bytes=remaining;value.summary.categories={product_media:occupied};show();expect(await screen.findByText(new RegExp(label))).toBeVisible();
});
afterEach(()=>{cleanup();client.clear();sessionStorage.clear()});
function show(canPurchase=true){return render(<QueryClientProvider client={client}><EnterpriseStorage userId="actor" organizationId="org" scope="current" sequence={0} canPurchase={canPurchase}/></QueryClientProvider>)}
it("reads an existing pending order after purchase permission is revoked",async()=>{
 sessionStorage.setItem('storage-purchase.pending:["actor","org"]',JSON.stringify({offerId:"storage-offer",quoteId:"quote",key:crypto.randomUUID(),orderId:"order"}));
 api.readStorageOrder.mockResolvedValue({order_id:"order",status:"FULFILLED"});show(false);
 const recover=await screen.findByRole("button",{name:"恢复原订单"});expect(recover).toBeEnabled();fireEvent.click(recover);
 await screen.findByText(/扩容已生效/);expect(api.readStorageOrder.mock.calls[0].slice(0,3)).toEqual(["actor","org","order"]);expect(api.createStorageOrder).not.toHaveBeenCalled();expect(api.createStorageQuote).not.toHaveBeenCalled();expect(sessionStorage.getItem('storage-purchase.pending:["actor","org"]')).toBeNull();
});
it("does not create a pending order without purchase permission",async()=>{
 sessionStorage.setItem('storage-purchase.pending:["actor","org"]',JSON.stringify({offerId:"storage-offer",quoteId:"quote",key:crypto.randomUUID()}));show(false);
 const recover=await screen.findByRole("button",{name:"恢复原订单"});expect(recover).toBeDisabled();fireEvent.click(recover);expect(api.createStorageOrder).not.toHaveBeenCalled();expect(api.readStorageOrder).not.toHaveBeenCalled();
});
it("shows unavailable facts without substituting zero occupancy",async()=>{
 api.getStorageSummary.mockRejectedValue(new Error("unavailable"));show();await screen.findByText(/本次未取得用量/);expect(screen.queryByText("0.00 GiB")).not.toBeInTheDocument();expect(screen.getByRole("button",{name:"购买扩容包"})).toBeDisabled();
});
it("keeps the original request after acknowledgement loss and guards double clicks",async()=>{
 api.createStorageOrder.mockRejectedValue({code:"DEADLINE_EXCEEDED"});const first=show();await waitFor(()=>expect(screen.getByRole("button",{name:"购买扩容包"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"购买扩容包"}));const confirm=await screen.findByRole("button",{name:"确认支付 ¥19.90"});fireEvent.click(confirm);fireEvent.click(confirm);
 await screen.findByText(/本次结果未确认/);expect(api.createStorageOrder).toHaveBeenCalledTimes(1);const raw=sessionStorage.getItem('storage-purchase.pending:["actor","org"]');expect(raw).not.toBeNull();const original=JSON.parse(raw!);
 first.unmount();show();await screen.findByRole("button",{name:"恢复原订单"});api.createStorageOrder.mockResolvedValue({kind:"STORAGE_CAPACITY_PURCHASE",order_id:"order",status:"FULFILLED"});fireEvent.click(screen.getByRole("button",{name:"恢复原订单"}));await screen.findByText(/扩容已生效/);expect(api.createStorageOrder.mock.calls[1].slice(0,4)).toEqual(["actor","org",original.quoteId,original.key]);expect(sessionStorage.getItem('storage-purchase.pending:["actor","org"]')).toBeNull();
});
