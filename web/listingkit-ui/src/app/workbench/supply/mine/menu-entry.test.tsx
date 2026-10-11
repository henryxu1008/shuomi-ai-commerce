import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import Introduction from "./page";
import Stage from "./[stage]/page";
const fixture = vi.hoisted(() => ({ preparations: vi.fn(), supply: vi.fn() }));
vi.mock("next/server", () => ({ connection: async () => {} }));
vi.mock("next/navigation", () => ({ notFound: () => { throw new Error("NOT_FOUND"); } }));
vi.mock("@/components/workbench/supply/preparations-page", () => ({ PreparationsPage: fixture.preparations }));
vi.mock("@/components/workbench/supply/supply-page", () => ({ SupplyPage: fixture.supply }));
afterEach(() => { cleanup(); vi.unstubAllEnvs(); vi.clearAllMocks(); });
it("keeps the introduction and selected batch without mounting a processing consumer", async () => {
 vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED", "true");
 vi.stubEnv("LISTINGKIT_ACQUISITION_PLANS_ENABLED", "true");
 vi.stubEnv("LISTINGKIT_SUPPLY_CHAIN_ENABLED", "false");
 const id="550e8400-e29b-41d4-a716-446655440000";
 render(await Introduction({searchParams:Promise.resolve({preparation:id})}));
 expect(screen.getByRole("heading", {level:1,name:"统一管理商品，按目标平台完成发布"})).toBeVisible();
 expect(screen.getByRole("link",{name:/查看待适配商品/})).toHaveAttribute("href",`/workbench/supply/mine/waiting?preparation=${id}`);
 expect(screen.queryByRole("link",{name:/完成商品发布/})).not.toBeInTheDocument();
 expect(fixture.preparations).not.toHaveBeenCalled();
 expect(fixture.supply).not.toHaveBeenCalled();
});
it.each(["ready","missing","review","uploaded"])("explains admission for recognized closed stage %s instead of returning 404", async stage => {
 vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED", "true");
 vi.stubEnv("LISTINGKIT_ACQUISITION_PLANS_ENABLED", "true");
 vi.stubEnv("LISTINGKIT_SUPPLY_CHAIN_ENABLED", "false");
 render(await Stage({params:Promise.resolve({stage}),searchParams:Promise.resolve({})}));
 expect(screen.getByText("平台处理尚未配置")).toBeVisible();
 expect(fixture.preparations).not.toHaveBeenCalled();
 expect(fixture.supply).not.toHaveBeenCalled();
});
it("does not admit unknown stage names", async () => {
 await expect(Stage({params:Promise.resolve({stage:"unknown"}),searchParams:Promise.resolve({})})).rejects.toThrow("NOT_FOUND");
});

it("keeps the requested saved operation through the introduction and each full-chain stage", async () => {
 vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED", "true");
 vi.stubEnv("LISTINGKIT_SUPPLY_CHAIN_ENABLED", "true");
 const preparation="550e8400-e29b-41d4-a716-446655440000",store="550e8400-e29b-41d4-a716-446655440001",operation="550e8400-e29b-41d4-a716-446655440002";
 render(await Introduction({searchParams:Promise.resolve({preparation,store,operation})}));
 for(const name of [/查看待适配商品/,/完成商品发布/]){
  const query=new URL(screen.getByRole("link",{name}).getAttribute("href")!,"https://localhost").searchParams;
  expect(query.get("preparation")).toBe(preparation);
  expect(query.get("store")).toBe(store);
  expect(query.get("operation")).toBe(operation);
 }
 cleanup();
 for(const stage of ["waiting","ready"]){
  render(await Stage({params:Promise.resolve({stage}),searchParams:Promise.resolve({preparation,store,operation})}));
  expect(fixture.supply.mock.lastCall?.[0]).toMatchObject({initialStage:stage,initialPreparation:preparation,initialStore:store,initialOperation:operation});
  cleanup();
 }
});
