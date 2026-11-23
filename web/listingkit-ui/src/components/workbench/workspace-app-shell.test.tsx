import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

const navigation = vi.hoisted(() => ({
  pathname: "/workbench",
  replace: vi.fn(),
}));

const injectedWorkbenchContext = vi.hoisted(() => ({ value: null as unknown }));

vi.mock("next/navigation", () => ({
  usePathname: () => navigation.pathname,
  useSearchParams: () => new URLSearchParams(),
  useRouter: () => ({ replace: navigation.replace }),
}));

vi.mock(
  "@/components/providers/workbench-context-provider",
  async (importOriginal) => {
    const actual =
      await importOriginal<
        typeof import("@/components/providers/workbench-context-provider")
      >();
    return {
      ...actual,
      useWorkbenchContext: () =>
        injectedWorkbenchContext.value ?? actual.useWorkbenchContext(),
    };
  },
);

import { WorkbenchContextProvider } from "@/components/providers/workbench-context-provider";
import { WorkspaceAppShell } from "@/components/workbench/workspace-app-shell";

const ACTIVE_CONTEXT = {
  user: { id: "user-1" },
  homeOrganizationId: "org-a",
  effectiveOrganizationId: "org-a",
  selectionRequired: false,
  organizations: [
    { id: "org-a", name: "硕米科技", roles: ["role-a"], permissions: [] },
    { id: "org-b", name: "星海贸易", roles: ["role-b"], permissions: [] },
  ],
};

const STORE_CONTEXT = {
  ...ACTIVE_CONTEXT,
  operationsCockpitAvailable: true,
  projectCenterAvailable: true,
  organizations: [
    { ...ACTIVE_CONTEXT.organizations[0], permissions: ["workbench.store.read", "workbench.cockpit.goals.read", "workbench.project.read"] },
    ACTIVE_CONTEXT.organizations[1],
  ],
};

function renderShell(child = <p>organization child</p>, reportCenterAvailable = false) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <WorkbenchContextProvider>
        <WorkspaceAppShell reportCenterAvailable={reportCenterAvailable}>{child}</WorkspaceAppShell>
      </WorkbenchContextProvider>
    </QueryClientProvider>,
  );
}

async function openSwitchDialog(user: ReturnType<typeof userEvent.setup>) {
  Object.defineProperties(HTMLDialogElement.prototype, {
    showModal: { configurable: true, value(this: HTMLDialogElement) { this.open = true; } },
    close: { configurable: true, value(this: HTMLDialogElement) { this.open = false; } },
  });
  await user.click(await screen.findByRole("button", { name: "我的账户" }));
  const menu = within(screen.getByRole("dialog", { name: "账户快捷操作" }));
  await user.click(menu.getByRole("button", { name: "切换企业" }));
  return within(await screen.findByRole("dialog", { name: "切换企业" }));
}

describe("WorkspaceAppShell", () => {
  it("projects desktop and mobile navigation from the selected enterprise and clears old grants while switching", async () => {
    navigation.pathname = "/workbench/ai/projects";
    const user = userEvent.setup();
    const payload = {
      ...ACTIVE_CONTEXT,
      projectCenterAvailable: true,
      aiWorkbenchAvailable: true,
      operationsCockpitAvailable: true,
      organizations: [
        { ...ACTIVE_CONTEXT.organizations[0], roles: ["custom-project-knowledge"], permissions: ["workbench.project.read", "workbench.knowledge.read"] },
        { ...ACTIVE_CONTEXT.organizations[1], roles: ["listingkit_admin"], permissions: ["workbench.store.read", "workbench.report.read", "workbench.cockpit.goals.read"] },
      ],
    };
    let finishSwitch!: (response: Response) => void;
    const switching = new Promise<Response>(resolve => { finishSwitch = resolve; });
    let effectiveOrganizationId = "org-a";
    const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (path, init) => {
      if (path === "/api/account/profile") return new Response(null, { status: 404 });
      if (init?.method === "PUT") return switching;
      return Response.json({ ...payload, effectiveOrganizationId });
    });
    vi.stubGlobal("fetch", fetchMock);
    renderShell(<p>organization child</p>, true);

    const desktop = within(await screen.findByRole("navigation", { name: "工作台导航" }));
    await user.click(screen.getByRole("button", { name: "打开工作台导航" }));
    const mobile = within(screen.getByRole("navigation", { name: "移动工作台导航" }));
    for (const nav of [desktop, mobile]) {
      expect(nav.getByRole("link", { name: "项目中心" })).toBeVisible();
      expect(nav.getByRole("link", { name: "知识库" })).toBeVisible();
      for (const name of ["运营驾驶舱", "店铺中心", "我的报告", "任务中心", "硕米Chat"]) {
        expect(nav.queryByRole("link", { name })).not.toBeInTheDocument();
      }
    }
    const dialog = await openSwitchDialog(user);
    await user.selectOptions(dialog.getByRole("combobox", { name: "选择企业" }), "org-b");
    await user.click(dialog.getByRole("button", { name: "确认切换" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/workbench/context/effective-organization", expect.objectContaining({ method: "PUT" }),
    ));
    for (const name of ["工作台导航", "移动工作台导航"]) {
      const nav = within(screen.getByRole("navigation", { name }));
      expect(nav.queryByRole("link", { name: "项目中心" })).not.toBeInTheDocument();
      expect(nav.queryByRole("link", { name: "知识库" })).not.toBeInTheDocument();
      expect(nav.queryByRole("link", { name: "店铺中心" })).not.toBeInTheDocument();
    }

    effectiveOrganizationId = "org-b";
    finishSwitch(Response.json({ ...payload, effectiveOrganizationId }));
    await waitFor(() => {
      for (const name of ["工作台导航", "移动工作台导航"]) {
        const nav = within(screen.getByRole("navigation", { name }));
        expect(nav.getByRole("link", { name: "店铺中心" })).toBeVisible();
        expect(nav.getByRole("link", { name: "运营驾驶舱" })).toBeVisible();
        expect(nav.getByRole("link", { name: "我的报告" })).toBeVisible();
        expect(nav.queryByRole("link", { name: "项目中心" })).not.toBeInTheDocument();
        expect(nav.queryByRole("link", { name: "知识库" })).not.toBeInTheDocument();
      }
    });
  });

  it.each([
    ["/workbench/account/organization/model-route-pricing", 503, true],
    ["/workbench/account/organization/model-route-pricing", 403, false],
    ["/workbench/account/organization/members", 503, false],
  ])("keeps the model settings receipt area only on a temporary failure (%s, %s)", (pathname, status, visible) => {
    navigation.pathname = pathname;
    injectProfileContext({
      user: { id: "user-1" },
      effectiveOrganization: ACTIVE_CONTEXT.organizations[0],
      error: { status, code: status === 403 ? "ORGANIZATION_ACCESS_REVOKED" : "DEPENDENCY_UNAVAILABLE" },
    });
    render(<WorkspaceAppShell><p>Model settings receipt area</p></WorkspaceAppShell>);
    if (visible) expect(screen.getByText("Model settings receipt area")).toBeVisible();
    else expect(screen.queryByText("Model settings receipt area")).not.toBeInTheDocument();
  });

  it.each([
    ["123456789", " 陈小明 ", "陈小明"],
    ["123456789", " ", "当前账户"],
    ["another-user", "其他账户姓名", "当前账户"],
  ])("uses the matching account display name instead of the user ID (%s, %s)", async (profileUserId, displayName, expectedName) => {
    const userId = "123456789";
    const fetch = vi.fn((path: string) => Promise.resolve(Response.json(path === "/api/account/profile" ? {
      schemaVersion: "account-v1", userId: profileUserId, homeOrganizationId: "org-a", displayName,
      email: null, emailVerified: null, phoneNumber: null, phoneNumberVerified: null,
      readAt: "2026-11-07T00:00:00Z", source: "zitadel_userinfo",
    } : { ...ACTIVE_CONTEXT, user: { id: userId } })));
    vi.stubGlobal("fetch", fetch);
    const user = userEvent.setup();
    renderShell();

    const trigger = await screen.findByRole("button", { name: "我的账户" });
    expect(fetch).not.toHaveBeenCalledWith("/api/account/profile", expect.anything());
    await user.click(trigger);
    const panel = within(screen.getByRole("dialog", { name: "账户快捷操作" }));
    expect(await panel.findByText(expectedName)).toBeVisible();
    expect(panel.queryByText(userId)).not.toBeInTheDocument();
    expect(panel.queryByText("其他账户姓名")).not.toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith("/api/account/profile", expect.objectContaining({ method: "GET" }));
  });

  it("closes the account popover with Escape and restores the trigger focus", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json(ACTIVE_CONTEXT)));
    const user = userEvent.setup();
    renderShell();

    const trigger = await screen.findByRole("button", { name: "我的账户" });
    await user.click(trigger);
    const profile = screen.getByRole("link", { name: /^账户资料$/ });
    profile.focus();
    await user.keyboard("{Escape}");

    await waitFor(() => expect(profile).not.toBeVisible());
    expect(trigger).toHaveFocus();
  });

  it("narrows desktop navigation without remounting the page or losing draft input", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json(ACTIVE_CONTEXT)));
    const user = userEvent.setup();
    renderShell(<input aria-label="未保存内容" defaultValue="" />);
    const input = await screen.findByLabelText("未保存内容");
    await user.type(input, "保留输入");
    await user.click(screen.getByRole("button", { name: "收窄侧栏" }));
    expect(screen.getByRole("button", { name: "展开侧栏" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByLabelText("未保存内容")).toBe(input);
    expect(input).toHaveValue("保留输入");
    expect(screen.getByRole("navigation", { name: "工作台导航" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "展开侧栏" }));
    expect(input).toHaveValue("保留输入");
  });
  it.each(["/workbench/agents/skills", "/workbench/agents/skills/official", "/workbench/agents/skills/mine"])("allows the identity-only skill route %s without enterprise selection", pathname => {
    navigation.pathname = pathname;
    injectProfileContext({error: {code: "ORGANIZATION_ACCESS_REVOKED"}});
    render(<WorkspaceAppShell><p>personal skill consumer</p></WorkspaceAppShell>);
    expect(screen.getByText("personal skill consumer")).toBeVisible();
    expect(navigation.replace).not.toHaveBeenCalled();
  });
  it.each(["/workbench/agents/skills/team", "/workbench/agents/skills/mine/extra", "/workbench/agents/mine"])("retains the enterprise gate on the skill sibling %s", pathname => {
    navigation.pathname = pathname;
    injectProfileContext();
    render(<WorkspaceAppShell><p>protected skill consumer</p></WorkspaceAppShell>);
    expect(screen.queryByText("protected skill consumer")).not.toBeInTheDocument();
    expect(navigation.replace).toHaveBeenCalledWith("/workbench/no-organization");
  });
  it.each(["/workbench/supply/operator", "/workbench/supply/operator/22222222-2222-4222-8222-222222222222"])("opens the enabled specialist route %s without enterprise selection", pathname => {
    navigation.pathname=pathname; injectProfileContext({blockingError:{code:"ORGANIZATION_ACCESS_REVOKED"}});
    render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><WorkspaceAppShell supplyMarketAvailable><p>market specialist consumer</p></WorkspaceAppShell></QueryClientProvider>);
    expect(screen.getByText("market specialist consumer")).toBeVisible();
    expect(navigation.replace).not.toHaveBeenCalled();
  });
  it.each(["/workbench/supply/operator/extra", "/workbench/supply/applications"])("retains enterprise gates on %s", pathname => {
    navigation.pathname=pathname; injectProfileContext();
    render(<WorkspaceAppShell supplyMarketAvailable><p>hidden market consumer</p></WorkspaceAppShell>);
    expect(screen.queryByText("hidden market consumer")).not.toBeInTheDocument();
  });
  it("allows the enabled exact tool review route without customer enterprise membership", () => {
    navigation.pathname = "/workbench/tools/custom/review";
    injectProfileContext({ selectionRequired: true });
    render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><WorkspaceAppShell toolMarketAvailable><p>tool specialist consumer</p></WorkspaceAppShell></QueryClientProvider>);
    expect(screen.getByText("tool specialist consumer")).toBeVisible();
    expect(navigation.replace).not.toHaveBeenCalled();
  });
  it.each(["/workbench/tools/custom", "/workbench/tools/custom/review/extra"])("keeps enterprise gates on the tool sibling %s", pathname => {
    navigation.pathname = pathname;
    injectProfileContext();
    render(<WorkspaceAppShell toolMarketAvailable><p>protected tool consumer</p></WorkspaceAppShell>);
    expect(screen.queryByText("protected tool consumer")).not.toBeInTheDocument();
    expect(navigation.replace).toHaveBeenCalledWith("/workbench/no-organization");
  });
  it("does not apply Browser recovery presentation to the personal profile", () => {
    navigation.pathname = "/workbench/account/profile";
    injectProfileContext({ error: { code: "AUTHENTICATION_REQUIRED" } });
    window.history.replaceState(null, "", "/workbench/account/profile#operationKey=22222222-2222-4222-8222-22222222222b");
    render(<WorkspaceAppShell><p>hidden profile</p></WorkspaceAppShell>);
    expect(screen.queryByText("hidden profile")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "在新标签页重新登录" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "重新加载" })).toBeVisible();
    window.history.replaceState(null, "", "/");
  });
  function injectProfileContext(overrides: Record<string, unknown> = {}) {
    injectedWorkbenchContext.value = { user: { id: "stale-user" }, homeOrganizationId: "org-a", organizations: [], effectiveOrganization: null, roles: [], permissions: [], selectionRequired: false, isLoading: false, isSwitching: false, error: null, blockingError: null, retry: vi.fn(), switchOrganization: vi.fn(), ...overrides };
  }

  it("allows the exact global ecosystem review route without choosing an enterprise",()=>{
    navigation.pathname="/workbench/services/review";injectProfileContext({selectionRequired:true});
    render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><WorkspaceAppShell ecoservicesAvailable><p>platform review consumer</p></WorkspaceAppShell></QueryClientProvider>);
    expect(screen.getByText("platform review consumer")).toBeVisible();expect(navigation.replace).not.toHaveBeenCalled();
  });
  it.each([{}, { isLoading: true }, { selectionRequired: true }, { error: { code: "DEPENDENCY_UNAVAILABLE" } }, { blockingError: { code: "ORGANIZATION_ACCESS_REVOKED" } }])("allows only exact profile through enterprise gates: %j", overrides => {
    navigation.pathname = "/workbench/account/profile"; injectProfileContext(overrides);
    render(<WorkspaceAppShell><p>personal profile</p></WorkspaceAppShell>);
    expect(screen.getByText("personal profile")).toBeVisible(); expect(navigation.replace).not.toHaveBeenCalled();
  });
  it.each([{}, { isLoading: true }, { selectionRequired: true }, { error: { code: "DEPENDENCY_UNAVAILABLE" } }, { blockingError: { code: "ORGANIZATION_ACCESS_REVOKED" } }])("allows exact account root to finish its personal redirect: %j", overrides => {
    navigation.pathname = "/workbench/account"; injectProfileContext(overrides);
    render(<WorkspaceAppShell><p>personal redirect</p></WorkspaceAppShell>);
    expect(screen.getByText("personal redirect")).toBeVisible(); expect(navigation.replace).not.toHaveBeenCalled();
  });
  it.each([
    "/workbench/account/referrals",
    "/workbench/account/referrals/complete",
  ].flatMap(path => [
    [path, {}],
    [path, { isLoading: true }],
    [path, { selectionRequired: true }],
    [path, { error: { code: "DEPENDENCY_UNAVAILABLE" } }],
    [path, { blockingError: { code: "ORGANIZATION_ACCESS_REVOKED" } }],
  ]))("allows exact personal referral route %s through enterprise gates", (path, overrides) => {
    navigation.pathname = path as string; injectProfileContext(overrides as Record<string, unknown>);
    render(<WorkspaceAppShell><p>personal referrals</p></WorkspaceAppShell>);
    expect(screen.getByText("personal referrals")).toBeVisible(); expect(navigation.replace).not.toHaveBeenCalled();
  });
  it.each(["/workbench/account/organization", "/workbench/account/organization/members", "/workbench/account/organization/resources", "/workbench/account/organization/audit", "/workbench/account/unknown", "/workbench/account-evil", "/workbench/account/profile/extra", "/workbench/account/referrals/extra", "/workbench/account/referrals/complete/extra", "/workbench/plans/options", "/workbench/ai/tasks", "/workbench/stores"])("preserves enterprise gate for %s", path => {
    navigation.pathname = path; injectProfileContext(); render(<WorkspaceAppShell><p>protected child</p></WorkspaceAppShell>);
    expect(screen.queryByText("protected child")).not.toBeInTheDocument(); expect(navigation.replace).toHaveBeenCalledWith("/workbench/no-organization");
  });
  it.each(["/capture/1688", "/capture/amazon"])("preserves %s recovery URL when the account has no organizations", (path) => {
    navigation.pathname = path;
    window.history.replaceState(null, "", `${path}#operationKey=22222222-2222-4222-8222-22222222222b`);
    injectProfileContext();
    render(<WorkspaceAppShell><p>capture child</p></WorkspaceAppShell>);
    expect(screen.getByText("capture child")).toBeVisible();
    expect(navigation.replace).not.toHaveBeenCalled();
  });
  it.each(["/capture/1688", "/capture/amazon"])("preserves %s handoff while the user logs in in a new tab", (path) => {
    navigation.pathname = path;
    window.history.replaceState(null, "", `${path}#extensionId=${"a".repeat(32)}&handoffId=11111111-1111-4111-8111-11111111111a&idempotencyKey=22222222-2222-4222-8222-22222222222b`);
    injectProfileContext({ error: { code: "AUTHENTICATION_REQUIRED" } });

    render(<WorkspaceAppShell><p>capture child</p></WorkspaceAppShell>);

    expect(screen.getByRole("link", { name: "在新标签页重新登录" })).toHaveAttribute(
      "href",
      "/login?returnTo=%2Fworkbench",
    );
    expect(screen.getByRole("link", { name: "在新标签页重新登录" })).toHaveAttribute("target", "_blank");
    expect(screen.getByRole("button", { name: "登录完成后刷新此页核实" })).toBeVisible();
    expect(navigation.replace).not.toHaveBeenCalled();
  });
  it.each(["error", "blockingError"])("keeps %s authentication denial above the profile exemption", field => {
    navigation.pathname = "/workbench/account/profile"; injectProfileContext({ [field]: { code: "AUTHENTICATION_REQUIRED" }, isLoading: true });
    render(<WorkspaceAppShell><p>personal profile</p></WorkspaceAppShell>); expect(screen.queryByText("personal profile")).not.toBeInTheDocument(); expect(screen.getByRole("alert")).toBeVisible();
  });
  it.each(["error", "blockingError"])("keeps %s authentication denial above the account-root exemption", field => {
    navigation.pathname = "/workbench/account"; injectProfileContext({ [field]: { code: "AUTHENTICATION_REQUIRED" }, isLoading: true });
    render(<WorkspaceAppShell><p>personal redirect</p></WorkspaceAppShell>);
    expect(screen.queryByText("personal redirect")).not.toBeInTheDocument(); expect(screen.getByRole("alert")).toBeVisible();
  });
  it.each([
    ["/workbench/account/referrals", "error"],
    ["/workbench/account/referrals", "blockingError"],
    ["/workbench/account/referrals/complete", "error"],
    ["/workbench/account/referrals/complete", "blockingError"],
  ])("keeps authentication denial above personal referral route %s via %s", (path, field) => {
    navigation.pathname = path; injectProfileContext({ [field]: { code: "AUTHENTICATION_REQUIRED" }, isLoading: true });
    render(<WorkspaceAppShell><p>personal referrals</p></WorkspaceAppShell>);
    expect(screen.queryByText("personal referrals")).not.toBeInTheDocument(); expect(screen.getByRole("alert")).toBeVisible();
  });
  it("hides stale identity, enterprise switcher and delegation metadata on grant failure", async () => {
    navigation.pathname = "/workbench/account/profile";
    injectProfileContext({
      error: { code: "DEPENDENCY_UNAVAILABLE" },
      organizations: ACTIVE_CONTEXT.organizations,
      effectiveOrganization: ACTIVE_CONTEXT.organizations[1],
    });
    render(<QueryClientProvider client={new QueryClient()}><WorkspaceAppShell><p>personal profile</p></WorkspaceAppShell></QueryClientProvider>);
    await userEvent.click(screen.getByRole("button", { name: "我的账户" }));
    const menu = within(screen.getByRole("dialog", { name: "账户快捷操作" }));
    expect(screen.getByText("personal profile")).toBeVisible();
    expect(menu.getByText("当前账户")).toBeVisible();
    expect(menu.queryByRole("button", { name: "切换企业" })).not.toBeInTheDocument();
    expect(menu.queryByText("当前企业")).not.toBeInTheDocument();
    expect(screen.queryByText(/星海贸易/)).not.toBeInTheDocument();
    expect(screen.queryByLabelText("企业代管状态")).not.toBeInTheDocument();
  });
  afterEach(() => {
    navigation.pathname = "/workbench";
    navigation.replace.mockReset();
    injectedWorkbenchContext.value = null;
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("shows a bounded loading shell before rendering Organization children", () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockReturnValue(new Promise(() => {})),
    );

    renderShell();

    expect(screen.getByRole("status")).toHaveTextContent("正在加载工作台");
    expect(screen.queryByText("organization child")).not.toBeInTheDocument();
  });

  it("renders granted business navigation and expands the Store entry", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValue(Response.json(STORE_CONTEXT)),
    );

    renderShell();

    expect(await screen.findByText("organization child")).toBeInTheDocument();
    expect(screen.getByText("硕米智能引擎")).toBeInTheDocument();
    expect(within(screen.getByRole("banner")).queryByText(/硕米科技/)).not.toBeInTheDocument();
    const navigationLandmark = screen.getByRole("navigation", {
      name: "工作台导航",
    });
    expect(
      within(navigationLandmark).getByRole("link", { name: "运营驾驶舱" }),
    ).toHaveAttribute("href", "/workbench");
    expect(within(navigationLandmark).getByRole("link", { name: "AI工作台" })).toHaveAttribute("href", "/workbench/ai");
    await userEvent.click(within(navigationLandmark).getByRole("button", { name: "展开店铺中心" }));
    expect(screen.getByText("店铺中心")).toBeInTheDocument();
    expect(
      within(navigationLandmark).getByRole("link", { name: "我的店铺" }),
    ).toHaveAttribute("href", "/workbench/stores");
    expect(screen.getAllByRole("main")).toHaveLength(1);
  });

  it("reveals a reachable mobile navigation without using the desktop sidebar trigger", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValue(Response.json(STORE_CONTEXT)),
    );
    const user = userEvent.setup();

    renderShell();

    const mobileNavigationButton = await screen.findByRole("button", {
      name: "打开工作台导航",
    });
    expect(mobileNavigationButton).toHaveAttribute("aria-expanded", "false");
    expect(mobileNavigationButton).toHaveClass("md:hidden");
    expect(screen.queryByRole("button", { name: "折叠桌面导航" })).not.toBeInTheDocument();
    expect(
      screen.queryByRole("navigation", { name: "移动工作台导航" }),
    ).not.toBeInTheDocument();

    mobileNavigationButton.focus();
    await user.keyboard("[Enter]");

    expect(mobileNavigationButton).toHaveAttribute("aria-expanded", "true");
    await user.keyboard("[Escape]");
    expect(mobileNavigationButton).toHaveAttribute("aria-expanded", "false");
    expect(mobileNavigationButton).toHaveFocus();
    expect(screen.queryByRole("navigation", { name: "移动工作台导航" })).not.toBeInTheDocument();
    await user.keyboard("[Enter]");
    const mobileNavigation = screen.getByRole("navigation", {
      name: "移动工作台导航",
    });
    const mobileWorkbenchLink = within(mobileNavigation).getByRole("link", {
      name: "运营驾驶舱",
    });
    expect(mobileWorkbenchLink).toHaveAttribute("href", "/workbench");
    mobileWorkbenchLink.addEventListener("click", (event) =>
      event.preventDefault(),
    );

    await user.click(mobileWorkbenchLink);

    expect(mobileNavigationButton).toHaveAttribute("aria-expanded", "false");
    expect(
      screen.queryByRole("navigation", { name: "移动工作台导航" }),
    ).not.toBeInTheDocument();
  });

  it("uses exact Workbench and prefix Store active matching in both navigation modes", async () => {
    navigation.pathname = "/workbench/stores/11111111-1111-4111-8111-111111111111";
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValue(Response.json(STORE_CONTEXT)),
    );
    const user = userEvent.setup();
    renderShell();

    const desktopNavigation = await screen.findByRole("navigation", { name: "工作台导航" });
    expect(within(desktopNavigation).getByRole("link", { name: "我的店铺" })).toHaveAttribute("aria-current", "page");
    expect(within(desktopNavigation).getByRole("link", { name: "运营驾驶舱" })).not.toHaveAttribute("aria-current");
    await user.click(screen.getByRole("button", { name: "打开工作台导航" }));
    expect(within(screen.getByRole("navigation", { name: "移动工作台导航" })).getByRole("link", { name: "我的店铺" })).toHaveAttribute("aria-current", "page");
  });

  it("persistently identifies delegated operation with safe Organization names", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValue(
        Response.json({
          ...ACTIVE_CONTEXT,
          effectiveOrganizationId: "org-b",
        }),
      ),
    );

    renderShell();

    expect(
      await screen.findByRole("status", { name: "企业代管状态" }),
    ).toHaveTextContent("正在代管星海贸易");
    expect(
      screen.getByRole("status", { name: "企业代管状态" }),
    ).toHaveTextContent("账号归属硕米科技");
  });

  it("uses a stable non-ID fallback for an injected unnamed delegated target", async () => {
    injectedWorkbenchContext.value = {
      user: { id: "user-1" },
      homeOrganizationId: "org-a",
      organizations: [
        { id: "org-a", name: "硕米科技", roles: ["role-a"], permissions: [] },
        { id: "org-b", name: "", roles: ["role-b"], permissions: [] },
      ],
      effectiveOrganization: { id: "org-b", name: "", roles: ["role-b"], permissions: [] },
      roles: ["role-b"], permissions: [],
      selectionRequired: false,
      isLoading: false,
      isSwitching: false,
      error: null,
      blockingError: null,
      retry: vi.fn(),
      switchOrganization: vi.fn(),
    };

    render(
      <WorkspaceAppShell>
        <p>organization child</p>
      </WorkspaceAppShell>,
    );

    const indicator = await screen.findByRole("status", {
      name: "企业代管状态",
    });
    expect(indicator).toHaveTextContent("正在代管当前企业");
    expect(indicator).not.toHaveTextContent("org-b");
  });

  it("selects the initial enterprise through the account menu before showing scoped children", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (path, init) => {
      if (path === "/api/account/profile") return new Response(null, { status: 404 });
      if (init?.method === "PUT") return Response.json({ ...ACTIVE_CONTEXT, effectiveOrganizationId: "org-b" });
      return Response.json({ ...ACTIVE_CONTEXT, effectiveOrganizationId: null, selectionRequired: true });
    });
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();

    renderShell();

    expect(await screen.findByRole("heading", { name: "请选择企业以继续" })).toBeVisible();
    expect(screen.getByText(/我的账户.*切换企业/)).toBeVisible();
    expect(screen.queryByText("organization child")).not.toBeInTheDocument();
    const sidebar = within(screen.getByRole("navigation", { name: "工作台导航" }));
    expect(sidebar.getByRole("button", { name: "展开我的账户" })).toBeVisible();
    expect(sidebar.queryByRole("link", { name: "我的账户" })).not.toBeInTheDocument();
    expect(screen.getAllByRole("combobox")).toEqual([screen.getByRole("combobox", { name: "显示主题" })]);

    const dialog = await openSwitchDialog(user);
    const switcher = dialog.getByRole("combobox", { name: "选择企业" });
    expect(switcher).toHaveValue("");
    await user.selectOptions(switcher, "org-b");
    expect(screen.queryByText("organization child")).not.toBeInTheDocument();
    await user.click(dialog.getByRole("button", { name: "确认切换" }));

    expect(await screen.findByText("organization child")).toBeVisible();
    expect(within(screen.getByRole("navigation", { name: "工作台导航" })).getByRole("link", { name: "我的账户" }))
      .toHaveAttribute("href", "/workbench/account");
    await user.click(screen.getByRole("button", { name: "我的账户" }));
    expect(within(screen.getByRole("dialog", { name: "账户快捷操作" })).getByText("星海贸易")).toBeVisible();
    expect(fetchMock).toHaveBeenCalledWith("/api/workbench/context/effective-organization", expect.objectContaining({
      method: "PUT", body: JSON.stringify({ organizationId: "org-b" }),
    }));
  });

  it("replaces a zero-Organization route without redirect-looping on the destination", async () => {
    const noOrganizationContext = {
      ...ACTIVE_CONTEXT,
      effectiveOrganizationId: null,
      selectionRequired: false,
      organizations: [],
    };
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockImplementation(() =>
          Promise.resolve(Response.json(noOrganizationContext)),
        ),
    );

    const firstRender = renderShell();

    await waitFor(() => {
      expect(navigation.replace).toHaveBeenCalledWith(
        "/workbench/no-organization",
      );
    });
    expect(screen.queryByText("organization child")).not.toBeInTheDocument();
    firstRender.unmount();

    navigation.replace.mockReset();
    navigation.pathname = "/workbench/no-organization";
    renderShell(<p>no Organization access page</p>);

    expect(
      await screen.findByText("no Organization access page"),
    ).toBeInTheDocument();
    expect(navigation.replace).not.toHaveBeenCalled();
  });

  it("leaves the no-Organization route after a later context refresh grants access", async () => {
    navigation.pathname = "/workbench/no-organization";
    injectedWorkbenchContext.value = {
      user: { id: "user-1" },
      homeOrganizationId: "org-a",
      organizations: [{ id: "org-a", name: "硕米科技", roles: ["role-a"], permissions: [] }],
      effectiveOrganization: { id: "org-a", name: "硕米科技", roles: ["role-a"], permissions: [] },
      roles: ["role-a"], permissions: [],
      selectionRequired: false,
      isLoading: false,
      isSwitching: false,
      error: null,
      blockingError: null,
      retry: vi.fn(),
      switchOrganization: vi.fn(),
      registerOrganizationSwitchGuard: vi.fn(() => vi.fn()),
    };

    render(<WorkspaceAppShell><p>organization child</p></WorkspaceAppShell>);

    await waitFor(() => expect(navigation.replace).toHaveBeenCalledWith("/workbench"));
    expect(screen.queryByText("organization child")).not.toBeInTheDocument();
  });

  it("renders stable-code copy for initial errors and recovers only on explicit retry", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(
        Response.json(
          {
            code: "DEPENDENCY_UNAVAILABLE",
            message: "do not render this raw text",
            requestId: "req-down",
            fieldErrors: [],
          },
          { status: 503 },
        ),
      )
      .mockResolvedValueOnce(Response.json(ACTIVE_CONTEXT));
    vi.stubGlobal("fetch", fetchMock);

    renderShell();

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "企业权限服务暂时不可用，请稍后重试",
    );
    expect(screen.queryByText(/do not render/)).not.toBeInTheDocument();
    expect(screen.queryByText("organization child")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "重新加载" }));

    expect(await screen.findByText("organization child")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("redirects authentication failures to login instead of retrying the stale session", async () => {
    const retry = vi.fn();
    injectedWorkbenchContext.value = {
      user: null,
      homeOrganizationId: null,
      organizations: [],
      effectiveOrganization: null,
      roles: [], permissions: [],
      selectionRequired: false,
      isLoading: false,
      isSwitching: false,
      error: { code: "AUTHENTICATION_REQUIRED" },
      blockingError: null,
      retry,
      switchOrganization: vi.fn(),
    };

    render(<WorkspaceAppShell><p>organization child</p></WorkspaceAppShell>);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "登录状态已失效，请重新登录",
    );
    await userEvent.click(screen.getByRole("button", { name: "重新加载" }));

    expect(navigation.replace).toHaveBeenCalledWith(
      "/login?returnTo=%2Fworkbench",
    );
    expect(retry).not.toHaveBeenCalled();
  });

  it("removes old scoped children and blocks after an explicit switch failure", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (path, init) => {
      if (path === "/api/account/profile") return new Response(null, { status: 404 });
      if (init?.method === "PUT") return Response.json(
        {
          code: "ORGANIZATION_ACCESS_REVOKED",
          message: "ignored",
          requestId: "req-revoked",
          fieldErrors: [],
        },
        { status: 403 },
      );
      return Response.json(ACTIVE_CONTEXT);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderShell();
    const user = userEvent.setup();
    const dialog = await openSwitchDialog(user);
    await user.selectOptions(dialog.getByRole("combobox", { name: "选择企业" }), "org-b");
    await user.click(dialog.getByRole("button", { name: "确认切换" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "你对所选企业的访问权限已被撤销",
    );
    expect(screen.queryByText("organization child")).not.toBeInTheDocument();
    expect(fetchMock.mock.calls.filter(([, init]) => init?.method === "PUT")).toHaveLength(1);
  });
});
