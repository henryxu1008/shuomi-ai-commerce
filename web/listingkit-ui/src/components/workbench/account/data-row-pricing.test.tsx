import {afterEach,expect,it,vi} from "vitest";
import {cleanup,render,screen} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {DataRowPriceEditor} from "./data-row-pricing";
const offer={offer_id:"initial",product_kind:"DATA_ROW",resource_type:"data_row",currency:"CNY",unit_price_minor:"5",pricing_version:"initial",min_quantity:"1",max_quantity:"100000",settlement_mode:"WALLET"};
afterEach(()=>{cleanup();vi.unstubAllGlobals();sessionStorage.clear()});
it("previews a zero price and reconciles a lost response using the original key",async()=>{
 let key="";let writes=0;const receipt={id:"receipt-1",actor_id:"maintainer",organization_id:"org-a",previous_offer_id:"initial",previous_version:"initial",offer:{...offer,offer_id:"zero",pricing_version:"zero-v1",unit_price_minor:"0",settlement_mode:"ZERO_PRICE"},reason:"isolated test",created_at:"2026-11-08T00:00:00Z"};
 const fetcher=vi.fn(async(url:string,init?:RequestInit)=>{if(init?.method==="POST"){writes++;key=new Headers(init.headers).get("Idempotency-Key")!;expect(JSON.parse(String(init.body))).toEqual({expected_version:"initial",unit_price_minor:"0",reason:"isolated test"});throw Error("synthetic response lost")};if(url.includes("/receipts/")){expect(url.endsWith(key)).toBe(true);return Response.json(receipt)};return Response.json(url.includes("/history")?{items:[],next_cursor:""}:offer)});vi.stubGlobal("fetch",fetcher);
 const c=new QueryClient({defaultOptions:{queries:{retry:false}}});render(<QueryClientProvider client={c}><DataRowPriceEditor scope={{userId:"maintainer",organizationId:"org-a"}} canWrite/></QueryClientProvider>);
 await screen.findByText("¥0.05 / 条");const user=userEvent.setup();await user.type(screen.getByLabelText("单价（人民币分）"),"0");await user.type(screen.getByLabelText("调整原因"),"isolated test");await user.click(screen.getByRole("button",{name:"预览价格调整"}));expect(screen.getByText(/零价发布后/)).toBeVisible();await user.click(screen.getByRole("button",{name:"确认发布价格"}));await screen.findByText(/发布结果尚未确认/);await user.click(screen.getByRole("button",{name:"核对原发布回执"}));await screen.findByText(/发布回执 receipt-1/);expect(writes).toBe(1);expect(sessionStorage.length).toBe(0);c.clear();
});
