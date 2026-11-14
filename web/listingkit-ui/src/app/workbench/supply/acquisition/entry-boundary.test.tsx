import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import AcquisitionEntry from "./page";
import AcquisitionDetailEntry from "./operation/[operation_id]/page";

const state = vi.hoisted(() => ({ available: false }));
vi.mock("next/server", () => ({ connection: vi.fn() }));
vi.mock("next/navigation", () => ({ notFound: () => { throw new Error("NEXT_HTTP_ERROR_FALLBACK;404"); } }));
vi.mock("@/lib/server/product-acquisition-availability", () => ({ isProductAcquisitionAvailable: () => state.available }));
vi.mock("@/components/workbench/acquisition/acquisition-page", () => ({ AcquisitionPage: ({ operationId }: { operationId?: string }) => <p>{operationId ?? "entry"}</p> }));
vi.mock("@/components/workbench/acquisition/online-acquisition-hub", () => ({ OnlineAcquisitionHub: ({ has1688, hasAmazon, initial }: { has1688:boolean;hasAmazon:boolean;initial:string }) => <p>{`${has1688}:${hasAmazon}:${initial}`}</p> }));

beforeEach(() => { state.available = false;vi.stubEnv("LISTINGKIT_DATA_SERVICES_ENABLED", "false"); });
afterEach(()=>{cleanup();vi.unstubAllEnvs()});

it("keeps both direct acquisition URLs closed by default", async () => {
  await expect(AcquisitionEntry()).rejects.toThrow("NEXT_HTTP_ERROR_FALLBACK;404");
  await expect(AcquisitionDetailEntry({ params: Promise.resolve({ operation_id: "11111111-1111-4111-8111-111111111111" }) })).rejects.toThrow("NEXT_HTTP_ERROR_FALLBACK;404");
});

it("opens both URLs only for an explicitly enabled serving deployment", async () => {
  state.available = true;
  render(await AcquisitionEntry());
  expect(screen.getByText("true:false:1688")).toBeVisible();
  cleanup();
  render(await AcquisitionDetailEntry({ params: Promise.resolve({ operation_id: "11111111-1111-4111-8111-111111111111" }) }));
  expect(screen.getByText("11111111-1111-4111-8111-111111111111")).toBeVisible();
});
it("allows the unified entry when only Amazon is enabled without opening 1688 detail", async () => {
  vi.stubEnv("LISTINGKIT_DATA_SERVICES_ENABLED", "true");
  render(await AcquisitionEntry());expect(screen.getByText("false:true:amazon")).toBeVisible();
  await expect(AcquisitionDetailEntry({params:Promise.resolve({operation_id:"11111111-1111-4111-8111-111111111111"})})).rejects.toThrow("NEXT_HTTP_ERROR_FALLBACK;404");
});
