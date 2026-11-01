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
        <ConsoleNavigation pathname="/workbench/store-orders" ariaLabel="主导航" storeOrdersAvailable={observation} />
      </FulfillmentNavigation>
    </QueryClientProvider>);
    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.getByRole("link", { name: "订单处理" })).toHaveAttribute("title", fulfillment ? "订单处理" : "订单处理：业务暂未启用"));
    expect(screen.getByRole("link", { name: "平台订单记录" })).toHaveAttribute("title", observation ? "平台订单记录" : "平台订单记录：业务暂未启用");
    expect(screen.getByRole("link", { name: "订单管理" })).toHaveAttribute("title", observation || fulfillment ? "订单管理" : "订单管理：业务暂未启用");
  },
);
