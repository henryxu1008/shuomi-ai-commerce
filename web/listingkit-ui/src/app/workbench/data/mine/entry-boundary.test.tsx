import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import Page from "./page";

vi.mock("next/server", () => ({ connection: async () => {} }));
vi.mock("next/navigation", () => ({ notFound: () => { throw new Error("NOT_FOUND"); } }));
vi.mock("@/components/workbench/collections/collection-page", () => ({
  CollectionPage: ({ supplyAvailable, initialBatchId, initialItemId }: { supplyAvailable: boolean; initialBatchId?: string; initialItemId?: string }) =>
    <div><p>{initialBatchId}</p><p>{initialItemId}</p>{supplyAvailable ? <button>加入我的供应链</button> : null}</div>,
}));

afterEach(() => { cleanup(); vi.unstubAllEnvs(); });

it.each([
  { plans: "true", fullChain: "false", available: true },
  { plans: "false", fullChain: "true", available: true },
  { plans: "false", fullChain: "false", available: false },
])("preserves the existing batch transfer entry for plans=$plans and fullChain=$fullChain", async ({ plans, fullChain, available }) => {
  vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED", "true");
  vi.stubEnv("LISTINGKIT_ACQUISITION_PLANS_ENABLED", plans);
  vi.stubEnv("LISTINGKIT_SUPPLY_CHAIN_ENABLED", fullChain);
  const batchId = "550e8400-e29b-41d4-a716-446655440000";
  const item = "550e8400-e29b-41d4-a716-446655440001";
  render(await Page({ searchParams: Promise.resolve({ batchId, item }) }));
  expect(screen.getByText(batchId)).toBeVisible();
  expect(screen.getByText(item)).toBeVisible();
  if (available) expect(screen.getByRole("button", { name: "加入我的供应链" })).toBeVisible();
  else expect(screen.queryByRole("button", { name: "加入我的供应链" })).not.toBeInTheDocument();
});

it("keeps the collection page closed when collections are disabled even with preparations enabled", async () => {
  vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED", "false");
  vi.stubEnv("LISTINGKIT_ACQUISITION_PLANS_ENABLED", "true");
  vi.stubEnv("LISTINGKIT_SUPPLY_CHAIN_ENABLED", "true");
  await expect(Page({ searchParams: Promise.resolve({}) })).rejects.toThrow("NOT_FOUND");
});
