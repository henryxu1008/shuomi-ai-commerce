import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import ListingEntry from "./page";
import ProcurementEntry from "./procurement/page";
import Stage from "./[stage]/page";
const fixture = vi.hoisted(() => ({ preparations: vi.fn(), supply: vi.fn(),completion:vi.fn() }));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({effectiveOrganization:null,user:null,permissions:[]})}));
vi.mock("next/server", () => ({ connection: async () => {} }));
vi.mock("@/components/workbench/supply/preparations-page", () => ({ PreparationsPage: fixture.preparations }));
vi.mock("@/components/workbench/supply/supply-page", () => ({ SupplyPage: fixture.supply }));
vi.mock("@/components/workbench/supply/completion-drafts-page",()=>({CompletionDraftsPage:fixture.completion}));
afterEach(() => { cleanup(); vi.unstubAllEnvs(); vi.clearAllMocks(); });
it("enters the waiting stage directly, preserving the validated listing context", async () => {
 const id="550e8400-e29b-41d4-a716-446655440000";
 const params={preparation:id,store:id,operation:id,plan:id};
 await expect(ListingEntry({searchParams:Promise.resolve(params)})).rejects.toMatchObject({message:"NEXT_REDIRECT",digest:expect.stringContaining(";/workbench/supply/mine/waiting?"+new URLSearchParams(params)+";")});
 await expect(ListingEntry({searchParams:Promise.resolve({plan:[id,id]})})).rejects.toThrow("NEXT_HTTP_ERROR_FALLBACK;404");
});
it("takes the previous procurement link to the Orders purchase step with its original order", async () => {
 const searchParams=Promise.resolve({storeId:"store-1",sourceId:"source-1",kind:"platform_supply",step:"stock"});
 await expect(ProcurementEntry({searchParams})).rejects.toMatchObject({message:"NEXT_REDIRECT",digest:expect.stringContaining(";/workbench/store-orders/fulfill?step=purchase&storeId=store-1&sourceId=source-1&kind=platform_supply;")});
});
it.each(["ready","missing","review","uploaded"])("explains admission for recognized closed stage %s instead of returning 404", async stage => {
 vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED", "true");
 vi.stubEnv("LISTINGKIT_ACQUISITION_PLANS_ENABLED", "true");
 vi.stubEnv("LISTINGKIT_SUPPLY_CHAIN_ENABLED", "false");
 const preparation="550e8400-e29b-41d4-a716-446655440000";
 render(await Stage({params:Promise.resolve({stage}),searchParams:Promise.resolve({preparation})}));
 expect(screen.getByText("平台处理尚未配置")).toBeVisible();
 expect(screen.getByRole("navigation",{name:"商品刊登阶段"})).toBeVisible();
 expect(screen.getByRole("link",{name:"待上传"})).toHaveAttribute("href","/workbench/supply/mine/waiting?preparation="+preparation);
 expect(fixture.preparations).not.toHaveBeenCalled();
 expect(fixture.supply).not.toHaveBeenCalled();
});
it("does not admit unknown stage names", async () => {
 await expect(Stage({params:Promise.resolve({stage:"unknown"}),searchParams:Promise.resolve({})})).rejects.toThrow("NEXT_HTTP_ERROR_FALLBACK;404");
});
it("opens native waiting and missing entries with drafts alone, keeping platform execution closed",async()=>{
 vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED","true");vi.stubEnv("LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED","true");vi.stubEnv("LISTINGKIT_SUPPLY_CHAIN_ENABLED","false");vi.stubEnv("LISTINGKIT_ACQUISITION_PLANS_ENABLED","false");
 const preparation="550e8400-e29b-41d4-a716-446655440000",source="550e8400-e29b-41d4-a716-446655440001";
 render(await Stage({params:Promise.resolve({stage:"missing"}),searchParams:Promise.resolve({preparation,source})}));
 expect(fixture.completion.mock.lastCall?.[0]).toMatchObject({initialPreparation:preparation,initialSource:source});expect(fixture.supply).not.toHaveBeenCalled();cleanup();
 render(await Stage({params:Promise.resolve({stage:"waiting"}),searchParams:Promise.resolve({preparation})}));expect(fixture.preparations.mock.lastCall?.[0]).toMatchObject({completionAvailable:true});cleanup();
 render(await Stage({params:Promise.resolve({stage:"ready"}),searchParams:Promise.resolve({})}));expect(screen.getByText("平台处理尚未配置")).toBeVisible();cleanup();
});

it("keeps the requested saved operation on full-chain stage links", async () => {
 vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED", "true");
 vi.stubEnv("LISTINGKIT_SUPPLY_CHAIN_ENABLED", "true");
 const preparation="550e8400-e29b-41d4-a716-446655440000",store="550e8400-e29b-41d4-a716-446655440001",operation="550e8400-e29b-41d4-a716-446655440002";
 for(const stage of ["waiting","missing","review","uploaded","ready"]){
  render(await Stage({params:Promise.resolve({stage}),searchParams:Promise.resolve({preparation,store,operation, ...stage==="missing"?{mode:"platform"}:{}})}));
  expect(fixture.supply.mock.lastCall?.[0]).toMatchObject({initialStage:stage,initialPreparation:preparation,initialStore:store,initialOperation:operation});
  cleanup();
 }
});
