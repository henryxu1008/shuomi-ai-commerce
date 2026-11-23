import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ConsoleNavigation } from "../console/console-navigation";
import { FulfillmentNavigation } from "./fulfillment-navigation";

vi.mock("next/navigation", () => ({ useSearchParams: () => new URLSearchParams() }));
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

it.each([[true, false], [false, true], [false, false], [true, true]])(
  "keeps observation=%s and fulfillment=%s independently advertised through normal capability discovery",
  async (observation, fulfillment) => {
    const fetch = vi.fn(async () => Response.json({ available: fulfillment, addressConfigured: true, realPlatformValidation: "NOT_RUN" }));
    vi.stubGlobal("fetch", fetch);
    render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <FulfillmentNavigation enabled scope={{ organizationId: "org", userId: "user" }} permissions={["workbench.fulfillment.read"]}>
        <ConsoleNavigation pathname="/workbench/store-orders" ariaLabel="主导航" storeObservationsAvailable={observation} permissions={["workbench.fulfillment.read", "workbench.store.read", "workbench.store.orders.read"]} />
      </FulfillmentNavigation>
    </QueryClientProvider>);
    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(1));
    if (fulfillment) expect(await screen.findByRole("link", { name: "订单处理" })).toBeVisible();
    else expect(screen.queryByRole("link", { name: "订单处理" })).not.toBeInTheDocument();
    expect(!!screen.queryByRole("link", { name: "平台订单记录" })).toBe(observation);
    expect(!!screen.queryByRole("link", { name: "订单管理" })).toBe(observation || fulfillment);
  },
);
