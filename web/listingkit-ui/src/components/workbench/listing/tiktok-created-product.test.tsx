import {afterEach,expect,it,vi} from "vitest";
import {cleanup,render,screen,fireEvent,waitFor} from "@testing-library/react";
import {TikTokCreatedProduct} from "./tiktok-created-product";
import {readSupplyPublicationStatus} from "@/lib/api/supply-chain";
vi.mock("@/lib/api/supply-chain",()=>({readSupplyPublicationStatus:vi.fn()}));
afterEach(()=>{cleanup();vi.resetAllMocks()});
const id="11111111-1111-4111-8111-111111111111",hash="a".repeat(64);
it("queries only on explicit action and separates actual status from draft receipt",async()=>{
 vi.mocked(readSupplyPublicationStatus).mockResolvedValue({sourceId:id,storeId:id,tiktokStatus:{productId:"local",status:"ACTIVATE",auditStatus:"APPROVED",reasons:[],observedAt:"2026-10-14T00:00:00Z"}});
 render(<TikTokCreatedProduct value={{draftHash:hash,localSaveMode:"AS_DRAFT",local:{id:"local",skus:[{id:"sku",seller_sku:"SKU"}],responseHash:hash}}} scope={{organizationId:"org",userId:"user"}} sourceId={id} storeId={id}/>);
 expect(readSupplyPublicationStatus).not.toHaveBeenCalled();
 expect(screen.queryByText(/平台当前在售/)).toBeNull();
 fireEvent.click(screen.getByRole("button",{name:"查询平台当前状态"}));
 await waitFor(()=>expect(screen.getByRole("status").textContent).toContain("平台当前在售"));
 expect(readSupplyPublicationStatus).toHaveBeenCalledWith({organizationId:"org",userId:"user"},id,id,expect.any(AbortSignal));
 expect(screen.getByText("已创建平台草稿")).toBeTruthy();
});
it("retains the confirmed product when an official query fails",async()=>{
 vi.mocked(readSupplyPublicationStatus).mockRejectedValue(new Error("unavailable"));
 render(<TikTokCreatedProduct value={{draftHash:hash,localSaveMode:"LISTING",local:{id:"local",skus:[{id:"sku",seller_sku:"SKU"}],responseHash:hash}}} scope={{organizationId:"org",userId:"user"}} sourceId={id} storeId={id}/>);
 fireEvent.click(screen.getByRole("button",{name:"查询平台当前状态"}));
 await waitFor(()=>expect(screen.getByRole("alert").textContent).toContain("原创建结果已保留"));
 expect(screen.getByText("商品：local")).toBeTruthy();
 expect(screen.queryByRole("status")).toBeNull();
});
