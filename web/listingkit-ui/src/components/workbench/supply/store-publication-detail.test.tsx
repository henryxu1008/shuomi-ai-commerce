import {render,screen,cleanup} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {StorePublicationDetail} from "./store-publication-detail";
const f=vi.hoisted(()=>({read:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.supply.read"],supplyIntentReady:true,registerOrganizationSwitchGuard:()=>()=>{}})}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),readSupplyPublication:f.read}));
vi.mock("./shein-uploaded-product",()=>({SheinUploadedProduct:({management}:{management:boolean})=><p>{management?"原回执店铺商品详情":"只读"}</p>}));
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
