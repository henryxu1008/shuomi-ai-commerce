import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { commercialResourcesFixture } from "@/test/fixtures/commercial-resources";
import type { WorkbenchStore } from "@/lib/api/workbench-stores";

const state = vi.hoisted(() => ({ roles: [] as string[], funding: undefined as boolean | undefined, balance: vi.fn(), members: vi.fn() }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => ({
  user: { id: "actor" }, effectiveOrganization: { id: "org-a", capabilities: { storeServiceEnterpriseFunding: state.funding } },
  roles: state.roles, permissions: ["workbench.store.read", "workbench.store.update", "workbench.organization_member.manage"], isSwitching: false,
}) }));
vi.mock("@/lib/api/commercial-billing", async original => ({ ...await original<typeof import("@/lib/api/commercial-billing")>(), getCommercialResources: state.balance }));
vi.mock("@/lib/api/member-resources", async original => ({ ...await original<typeof import("@/lib/api/member-resources")>(), getMemberResources: state.members }));
vi.mock("@/lib/api/store-connection", async original => ({ ...await original<typeof import("@/lib/api/store-connection")>(),
  getStoreConnection: vi.fn().mockResolvedValue({ connectionStatus: "connected", state: "verified", attemptId: "attempt" }),
  getStoreApplications: vi.fn().mockResolvedValue([]),
}));
vi.mock("@/lib/api/store-center", async original => ({ ...await original<typeof import("@/lib/api/store-center")>(),
  storeCenterRequest: vi.fn().mockImplementation(async (path: string) => path === "groups" ? { items: [] } : { ...store, profile: { groupId: null }, signals: {} }),
}));
vi.mock("./store-form", () => ({ StoreForm: ({ onCreated }: { onCreated: (store: WorkbenchStore) => void }) => <button onClick={() => onCreated(store)}>返回已创建店铺</button> }));
import { StoreDetailPresentation } from "./store-detail-presentation";
import { StoreCatalogDialogs } from "./store-catalog-dialogs";

const store: WorkbenchStore = { id: "11111111-1111-4111-8111-111111111111", name: "店铺 A", platform: "shein", region: "US", externalStoreId: "", recordStatus: "active", serviceStatus: "active", connectionStatus: "connected", version: 2 } as WorkbenchStore;
let client: QueryClient;
beforeEach(() => {
  sessionStorage.clear(); window.history.replaceState(null, "", "/workbench/stores/" + store.id + "#authorization");
  HTMLDialogElement.prototype.showModal = function () { this.open = true; };
  HTMLDialogElement.prototype.close = function () { this.open = false; };
  state.balance.mockReset().mockResolvedValue(commercialResourcesFixture("org-a"));
  state.members.mockReset().mockResolvedValue({ organizationId: "org-a", members: [{ memberId: "member-a", userId: "actor", periods: { free: "0" } }] });
  client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
});
afterEach(() => { cleanup(); client.clear(); });

it.each([
  { name: "builtin administrator", roles: ["listingkit_admin"], funding: true },
  { name: "configured administrator role", roles: ["configured-role"], funding: true },
  { name: "configured administrator user", roles: [], funding: true },
  { name: "member with member-management permission", roles: ["custom-role"], funding: false },
  { name: "role name without server authority", roles: ["admin"], funding: false },
  { name: "absent server projection", roles: ["admin"], funding: undefined },
])("uses server funding for $name in detail and binding entries", async ({ roles, funding }) => {
  state.roles = roles; state.funding = funding;
  for (const entry of ["detail", "binding"]) {
    const view = render(<QueryClientProvider client={client}>{entry === "detail"
      ? <StoreDetailPresentation store={store} canUpdate onRefresh={vi.fn().mockResolvedValue(null)} actions={null} editForm={null} />
      : <StoreCatalogDialogs dialog={{ kind: "bind" }} scope={{ organizationId: "org-a", userId: "actor" }} groups={[]} canManageGroups={false} onClose={vi.fn()} onChanged={vi.fn().mockResolvedValue(null)} />}</QueryClientProvider>);
    if (entry === "binding") await userEvent.click(screen.getByRole("button", { name: "返回已创建店铺" }));
    await userEvent.click(await screen.findByRole("button", { name: "续费服务" }));
    await screen.findByText(`可用 ${funding ? "1" : "0"} 期`);
    const confirm = screen.getByRole("button", { name: "确认续费服务" });
    if (funding) {
      expect(confirm).toBeEnabled(); expect(state.balance).toHaveBeenCalled(); expect(state.members).not.toHaveBeenCalled();
    } else {
      expect(confirm).toBeDisabled(); expect(state.members).toHaveBeenCalled(); expect(state.balance).not.toHaveBeenCalled();
    }
    view.unmount(); client.clear(); state.balance.mockClear(); state.members.mockClear();
  }
});
