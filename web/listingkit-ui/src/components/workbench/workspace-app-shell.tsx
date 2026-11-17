"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Suspense, useEffect, useRef, useState, useSyncExternalStore, type ReactNode } from "react";
import * as Popover from "@radix-ui/react-popover";
import { skipToken, useQuery } from "@tanstack/react-query";
import { useTheme } from "next-themes";
import { FulfillmentNavigation } from "./fulfillment/fulfillment-navigation";
import { ChevronDown, Menu, X, PanelLeftClose, PanelLeftOpen } from "lucide-react";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { OrganizationSwitcher, workbenchErrorMessage } from "@/components/workbench/organization-switcher";
import { AccountDialog } from "@/components/workbench/account/account-dialog";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { SupportLauncher } from "@/components/workbench/support-launcher";
import { ConsoleNavigation } from "@/components/workbench/console/console-navigation";
import { parseCaptureEntry } from "@/app/capture/1688/capture-handoff";
import { NotificationPopover } from "@/components/workbench/notifications/notification-center";
import { getAccountProfile } from "@/lib/api/account";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";

const NO_ORGANIZATION_ROUTE = "/workbench/no-organization";
const MOBILE_NAVIGATION_ID = "workbench-mobile-navigation";
const subscribeToHydration = () => () => {};
const getClientHydrationSnapshot = () => true;
const getServerHydrationSnapshot = () => false;
export function WorkspaceAppShell({ children, dataServicesAvailable = false, reportCenterAvailable = false, toolMarketAvailable = false, supplyMarketAvailable = false, productAcquisitionAvailable = false, productCollectionsAvailable = false, supplyChainAvailable = false, supplyPreparationAvailable = false, supplyCompletionDraftsAvailable = false, knowledgeAvailable = false, nativeRunsAvailable = false, productReviewAvailable = false, ecoservicesAvailable = false, sheinRecordsAvailable = false, notificationCenterAvailable = false, supportAvailable = false }: { children: ReactNode; dataServicesAvailable?: boolean; reportCenterAvailable?: boolean; toolMarketAvailable?: boolean; supplyMarketAvailable?: boolean; productAcquisitionAvailable?: boolean; productCollectionsAvailable?: boolean; supplyChainAvailable?: boolean; supplyPreparationAvailable?: boolean; supplyCompletionDraftsAvailable?: boolean; knowledgeAvailable?: boolean; nativeRunsAvailable?: boolean; productReviewAvailable?: boolean; ecoservicesAvailable?: boolean; sheinRecordsAvailable?: boolean; notificationCenterAvailable?: boolean; supportAvailable?: boolean }) {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const pathname = usePathname() ?? "/workbench";
  const router = useRouter();
  const context = useWorkbenchContext();
  // Personal identity is bootstrapped by the server page, independently of enterprise grants.
  const isPersonalAccountRoute = [
    "/workbench/account",
    "/workbench/account/profile",
    "/workbench/account/referrals",
    "/workbench/account/referrals/complete",
    "/workbench/account/referrals/earnings",
    "/workbench/account/referrals/withdrawals",
    "/workbench/account/profile/verification",
    "/workbench/notifications",
    "/workbench/notifications/announcements",
    "/workbench/agents/skills",
    "/workbench/agents/skills/official",
    "/workbench/agents/skills/mine",
  ].includes(pathname) || /^\/workbench\/notifications\/(official|business)\/[A-Za-z0-9_-]{1,2048}$/.test(pathname);
  const isBrowserCaptureRoute = pathname === "/capture/1688" || pathname === "/capture/amazon";
  const marketOperatorRoute = pathname === "/workbench/supply/operator" || pathname.startsWith("/workbench/supply/operator/") && isAcquisitionUUID(pathname.slice("/workbench/supply/operator/".length));
  const hasIndependentIdentityRoute=isPersonalAccountRoute || dataServicesAvailable && pathname === "/workbench/admin/data-customization" || ecoservicesAvailable&&pathname==="/workbench/services/review" || toolMarketAvailable&&pathname==="/workbench/tools/custom/review" || supplyMarketAvailable && marketOperatorRoute;
  const authenticationError = [context.blockingError, context.error].find(error => error?.code === "AUTHENTICATION_REQUIRED");

  const shouldRedirectToNoOrganization =
    !context.isLoading &&
    !context.error &&
    !context.blockingError &&
    context.organizations.length === 0 &&
    !hasIndependentIdentityRoute &&
    !isBrowserCaptureRoute &&
    pathname !== NO_ORGANIZATION_ROUTE;
  const shouldLeaveNoOrganization =
    !context.isLoading &&
    !context.error &&
    !context.blockingError &&
    context.organizations.length > 0 &&
    pathname === NO_ORGANIZATION_ROUTE;

  const redirectToLogin = () => {
    const returnTo = `${pathname}${window.location.search}`;
    router.replace(`/login?returnTo=${encodeURIComponent(returnTo)}`);
  };

  useEffect(() => {
    if (shouldRedirectToNoOrganization) {
      router.replace(NO_ORGANIZATION_ROUTE);
    } else if (shouldLeaveNoOrganization) {
      router.replace("/workbench");
    }
  }, [router, shouldLeaveNoOrganization, shouldRedirectToNoOrganization]);

  if (authenticationError) {
    const browserEntry = isBrowserCaptureRoute && typeof window !== "undefined"
      ? parseCaptureEntry(window.location.href, pathname === "/capture/amazon" ? "amazon" : "1688") : null;
    const browserRecovery = browserEntry?.kind === "recovery" || browserEntry?.kind === "handoff";
    return <AccessState action={browserRecovery ? () => window.location.reload() : redirectToLogin} code={authenticationError.code} browserRecovery={browserRecovery} />;
  }

  if (context.isLoading && !hasIndependentIdentityRoute) {
    return (
      <main className="flex min-h-svh items-center justify-center bg-background px-6">
        <p className="text-sm text-muted-foreground" role="status">
          正在加载工作台...
        </p>
      </main>
    );
  }

  if (context.blockingError && !hasIndependentIdentityRoute) {
    return (
      <AccessState
        action={
          context.blockingError.code === "AUTHENTICATION_REQUIRED"
            ? redirectToLogin
            : () => window.location.reload()
        }
        code={context.blockingError.code}
      />
    );
  }

  // A temporary read failure on this page leaves the already-issued save receipt
  // on screen. The page hides configuration and disables actions until context is read again.
  const keepModelReceipt = pathname === "/workbench/account/organization/model-route-pricing"
    && !!context.user && !!context.effectiveOrganization
    && (context.error?.status === 0 || (context.error?.status ?? 0) >= 500);
  if (context.error && !hasIndependentIdentityRoute && !keepModelReceipt) {
    return (
      <AccessState
        action={
          context.error.code === "AUTHENTICATION_REQUIRED"
            ? redirectToLogin
            : context.retry
        }
        code={context.error.code}
      />
    );
  }

  if (shouldRedirectToNoOrganization || shouldLeaveNoOrganization) {
    return (
      <main className="flex min-h-svh items-center justify-center bg-background px-6">
        <p className="text-sm text-muted-foreground" role="status">
          正在打开企业访问说明...
        </p>
      </main>
    );
  }

  return (
    <WorkbenchFrame sidebarCollapsed={sidebarCollapsed} onToggleSidebar={() => setSidebarCollapsed(value => !value)} supportAvailable={supportAvailable} dataServicesAvailable={dataServicesAvailable} reportCenterAvailable={reportCenterAvailable} toolMarketAvailable={toolMarketAvailable} supplyMarketAvailable={supplyMarketAvailable} key={pathname} pathname={pathname} productCollectionsAvailable={productCollectionsAvailable} supplyChainAvailable={supplyChainAvailable} supplyPreparationAvailable={supplyPreparationAvailable} supplyCompletionDraftsAvailable={supplyCompletionDraftsAvailable} productAcquisitionAvailable={productAcquisitionAvailable} knowledgeAvailable={knowledgeAvailable} nativeRunsAvailable={nativeRunsAvailable} productReviewAvailable={productReviewAvailable} ecoservicesAvailable={ecoservicesAvailable} sheinRecordsAvailable={sheinRecordsAvailable} notificationCenterAvailable={notificationCenterAvailable}>
      {context.selectionRequired && !hasIndependentIdentityRoute && !isBrowserCaptureRoute ? (
        <section
          className="flex min-h-[40vh] items-center justify-center px-6 text-center"
          role="status"
        >
          <div>
            <h1 className="text-3xl font-semibold">请选择企业以继续</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              请打开右上角「我的账户」，在「切换企业」中选择要使用的企业。
            </p>
          </div>
        </section>
      ) : (
        children
      )}
    </WorkbenchFrame>
  );
}

function WorkbenchFrame({ sidebarCollapsed, onToggleSidebar, children, pathname, dataServicesAvailable, reportCenterAvailable, toolMarketAvailable, supplyMarketAvailable, productAcquisitionAvailable, productCollectionsAvailable, supplyChainAvailable, supplyPreparationAvailable, supplyCompletionDraftsAvailable, knowledgeAvailable, nativeRunsAvailable, productReviewAvailable, ecoservicesAvailable, sheinRecordsAvailable, notificationCenterAvailable, supportAvailable }: { sidebarCollapsed: boolean; onToggleSidebar: () => void; children: ReactNode; pathname: string; dataServicesAvailable: boolean; reportCenterAvailable: boolean; toolMarketAvailable: boolean; supplyMarketAvailable: boolean; productAcquisitionAvailable: boolean; productCollectionsAvailable: boolean; supplyChainAvailable: boolean; supplyPreparationAvailable: boolean; supplyCompletionDraftsAvailable: boolean; knowledgeAvailable: boolean; nativeRunsAvailable: boolean; productReviewAvailable: boolean; ecoservicesAvailable: boolean; sheinRecordsAvailable: boolean; notificationCenterAvailable: boolean; supportAvailable: boolean }) {
  const [mobileOpen, setMobileOpen] = useState(false);
  const [organizationSwitcherOpen, setOrganizationSwitcherOpen] = useState(false);
  const trigger = useRef<HTMLButtonElement>(null);
  const accountTrigger = useRef<HTMLButtonElement>(null);
  const context = useWorkbenchContext();
  const contextConfirmed = !context.isLoading && !context.isSwitching && !context.error && !context.blockingError;
  const canSwitchOrganization = contextConfirmed && context.organizations.length > 1;
  const currentOrganization = contextConfirmed ? context.effectiveOrganization : null;
  const fulfillmentScope={organizationId:context.effectiveOrganization?.id??"",userId:context.user?.id??""};
  const fulfillmentReady=contextConfirmed&&!!fulfillmentScope.organizationId&&!!fulfillmentScope.userId&&context.permissions?.includes("workbench.fulfillment.read")===true;
  function closeNavigation() { setMobileOpen(false); trigger.current?.focus(); }
  function closeOrganizationSwitcher() { setOrganizationSwitcherOpen(false); }
  return <div onKeyDown={(event) => { if (mobileOpen && !organizationSwitcherOpen && event.key === "Escape" && !event.defaultPrevented) { event.preventDefault(); closeNavigation(); } }}>
    <a className="sr-only focus:not-sr-only" href="#console-main">跳到页面内容</a>
    <div className="console-frame" data-sidebar-collapsed={sidebarCollapsed || undefined}>
      <aside className="console-sidebar" id="workbench-desktop-navigation">
        <div className="console-sidebar-head"><Link href="/workbench" className="console-brand" aria-label="硕米智能引擎首页" prefetch={false}><Image src="/console/sumi-logo.png" alt="" width={42} height={42} unoptimized /><span className="console-brand-label"><strong>硕米智能引擎</strong><small>SUMI AI ENGINE</small></span></Link><Button className="console-sidebar-toggle" variant="ghost" size="icon" aria-label={sidebarCollapsed ? "展开侧栏" : "收窄侧栏"} title={sidebarCollapsed ? "展开侧栏" : "收窄侧栏"} aria-expanded={!sidebarCollapsed} aria-controls="workbench-desktop-navigation" onClick={onToggleSidebar}>{sidebarCollapsed ? <PanelLeftOpen aria-hidden="true" size={20} /> : <PanelLeftClose aria-hidden="true" size={20} />}</Button></div>
        <Suspense fallback={null}><FulfillmentNavigation enabled={fulfillmentReady} scope={fulfillmentScope} permissions={context.permissions}><ConsoleNavigation projectCenterAvailable={context.projectCenterAvailable} operationsCockpitAvailable={context.operationsCockpitAvailable} cockpitPermissions={context.permissions} dataServicesAvailable={dataServicesAvailable} reportCenterAvailable={reportCenterAvailable} toolMarketAvailable={toolMarketAvailable} supplyMarketAvailable={supplyMarketAvailable} key={pathname} pathname={pathname} ariaLabel="工作台导航" productCollectionsAvailable={productCollectionsAvailable} supplyChainAvailable={supplyChainAvailable} supplyPreparationAvailable={supplyPreparationAvailable} supplyCompletionDraftsAvailable={supplyCompletionDraftsAvailable} productAcquisitionAvailable={productAcquisitionAvailable} knowledgeAvailable={knowledgeAvailable} storeProductsAvailable={context.storeObservationsAvailable && context.effectiveOrganization?.permissions.includes("workbench.store.read") === true && context.effectiveOrganization.permissions.includes("workbench.store.products.read")} storeOrdersAvailable={context.storeObservationsAvailable && context.effectiveOrganization?.permissions.includes("workbench.store.read") === true && context.effectiveOrganization.permissions.includes("workbench.store.orders.read")} aiWorkbenchAvailable={context.aiWorkbenchAvailable} nativeRunsAvailable={nativeRunsAvailable} productReviewAvailable={productReviewAvailable} ecoservicesAvailable={ecoservicesAvailable} sheinRecordsAvailable={sheinRecordsAvailable} userId={context.user?.id} /></FulfillmentNavigation></Suspense>

      </aside>
      <div className="console-body">
        <header className="console-topbar">
          <Button ref={trigger} className="md:hidden" variant="ghost" size="icon" aria-controls={MOBILE_NAVIGATION_ID} aria-expanded={mobileOpen} aria-label={mobileOpen ? "关闭工作台导航" : "打开工作台导航"} onClick={() => setMobileOpen((value) => !value)}>{mobileOpen ? <X aria-hidden="true" /> : <Menu aria-hidden="true" />}</Button>
          <div className="console-top-context">
            <p className="console-tagline">打造属于自己的AI电商团队</p>
          </div>
          <div className="console-global-actions">
            <SupportLauncher available={supportAvailable && !context.isSwitching} key={`${context.user?.id ?? ""}:${context.effectiveOrganization?.id ?? ""}:${context.isSwitching}`} />
            {notificationCenterAvailable ? <NotificationPopover /> : <Button disabled title="通知中心尚未启用" variant="outline">通知</Button>}
            <WorkbenchThemeSelect />
            <Popover.Root>
              <Popover.Trigger ref={accountTrigger} className="console-user-trigger">
                我的账户
                <ChevronDown aria-hidden="true" className="console-user-chevron" size={16} strokeWidth={1.5} />
              </Popover.Trigger>
              <Popover.Portal>
                <Popover.Content className="console-user-popover" aria-label="账户快捷操作" align="end" sideOffset={8} collisionPadding={12}>
                  <p className="text-xs text-muted-foreground">已登录账号</p>
                  <AccountDisplayName userId={contextConfirmed ? context.user?.id : undefined} />
                  {currentOrganization ? (
                    <div className="mt-3 border-t border-border pt-3">
                      <p className="text-xs text-muted-foreground">当前企业</p>
                      <p className="mt-1 break-words text-sm">{currentOrganization.name.trim() || "名称未提供"}</p>
                    </div>
                  ) : null}
                  {canSwitchOrganization ? (
                    <Popover.Close className="console-user-action" aria-haspopup="dialog" onClick={() => setOrganizationSwitcherOpen(true)}>
                      切换企业
                    </Popover.Close>
                  ) : null}
                  <Popover.Close asChild type={undefined}><Link href="/workbench/account/profile" prefetch={false}>账户资料</Link></Popover.Close>
                  <a href="/api/zitadel-auth/logout">退出登录</a>
                </Popover.Content>
              </Popover.Portal>
            </Popover.Root>
          </div>
        </header>
        {organizationSwitcherOpen ? (
          <AccountDialog title="切换企业" onClose={closeOrganizationSwitcher} returnFocusRef={accountTrigger}>
            {canSwitchOrganization ? (
              <OrganizationSwitcher onConfirm={closeOrganizationSwitcher} onCancel={closeOrganizationSwitcher} />
            ) : <p role="status">暂时无法切换企业，请稍后重试。</p>}
          </AccountDialog>
        ) : null}
        {contextConfirmed && context.effectiveOrganization && context.effectiveOrganization.id !== context.homeOrganizationId ? <div className="console-delegation"><DelegatedOperationIndicator effectiveOrganization={context.effectiveOrganization} homeOrganizationId={context.homeOrganizationId} organizations={context.organizations} /></div> : null}
        {mobileOpen ? <div className="console-mobile-nav" id={MOBILE_NAVIGATION_ID}><Suspense fallback={null}><FulfillmentNavigation enabled={fulfillmentReady} scope={fulfillmentScope} permissions={context.permissions}><ConsoleNavigation projectCenterAvailable={context.projectCenterAvailable} operationsCockpitAvailable={context.operationsCockpitAvailable} cockpitPermissions={context.permissions} dataServicesAvailable={dataServicesAvailable} reportCenterAvailable={reportCenterAvailable} toolMarketAvailable={toolMarketAvailable} supplyMarketAvailable={supplyMarketAvailable} key={pathname} pathname={pathname} ariaLabel="移动工作台导航" onNavigate={closeNavigation} productCollectionsAvailable={productCollectionsAvailable} supplyChainAvailable={supplyChainAvailable} supplyPreparationAvailable={supplyPreparationAvailable} supplyCompletionDraftsAvailable={supplyCompletionDraftsAvailable} productAcquisitionAvailable={productAcquisitionAvailable} knowledgeAvailable={knowledgeAvailable} storeProductsAvailable={context.storeObservationsAvailable && context.effectiveOrganization?.permissions.includes("workbench.store.read") === true && context.effectiveOrganization.permissions.includes("workbench.store.products.read")} storeOrdersAvailable={context.storeObservationsAvailable && context.effectiveOrganization?.permissions.includes("workbench.store.read") === true && context.effectiveOrganization.permissions.includes("workbench.store.orders.read")} aiWorkbenchAvailable={context.aiWorkbenchAvailable} nativeRunsAvailable={nativeRunsAvailable} productReviewAvailable={productReviewAvailable} ecoservicesAvailable={ecoservicesAvailable} sheinRecordsAvailable={sheinRecordsAvailable} userId={context.user?.id} /></FulfillmentNavigation></Suspense></div> : null}
        <main className="console-content" id="console-main" tabIndex={-1}>{children}</main>
      </div>
    </div>
  </div>;
}

function WorkbenchThemeSelect() {
  const { theme, setTheme } = useTheme();
  const mounted = useSyncExternalStore(subscribeToHydration, getClientHydrationSnapshot, getServerHydrationSnapshot);
  return (
    <Select
      className="h-9 w-auto px-2 py-0"
      aria-label="显示主题"
      disabled={!mounted}
      value={mounted ? (theme ?? "dark") : ""}
      onChange={event => setTheme(event.currentTarget.value)}
    >
      <option value="" disabled hidden>显示主题</option>
      <option value="light">浅色</option>
      <option value="dark">深色</option>
      <option value="system">跟随系统</option>
    </Select>
  );
}

function AccountDisplayName({ userId }: { userId?: string }) {
  const profile = useQuery({
    queryKey: ["account", "popover-display-name", userId],
    queryFn: userId ? async ({ signal }) => {
      const account = await getAccountProfile({ expectedUserId: userId, signal });
      return account.displayName?.trim() || null;
    } : skipToken,
    gcTime: 0,
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
  const displayName = profile.isSuccess ? profile.data : null;
  const fallback = userId && profile.isPending ? "正在读取姓名…" : "当前账户";

  return <p className="mt-1 break-words text-sm" role="status">{displayName ?? fallback}</p>;
}

function DelegatedOperationIndicator({
  effectiveOrganization,
  homeOrganizationId,
  organizations,
}: {
  effectiveOrganization: ReturnType<
    typeof useWorkbenchContext
  >["effectiveOrganization"];
  homeOrganizationId: string | null;
  organizations: ReturnType<typeof useWorkbenchContext>["organizations"];
}) {
  if (
    !effectiveOrganization ||
    !homeOrganizationId ||
    effectiveOrganization.id === homeOrganizationId
  ) {
    return null;
  }
  const effectiveOrganizationName =
    effectiveOrganization.name.trim() || "当前企业";
  const homeOrganizationName =
    organizations.find((organization) => organization.id === homeOrganizationId)
      ?.name || "其他归属企业";
  return (
    <p
      aria-label="企业代管状态"
      className="min-w-0 text-right text-xs font-medium text-warning-foreground dark:text-warning-foreground"
      role="status"
    >
      <span className="block truncate">
        正在代管{effectiveOrganizationName}
      </span>
      <span className="block truncate text-muted-foreground">
        账号归属{homeOrganizationName}
      </span>
    </p>
  );
}

function AccessState({ action, code, browserRecovery = false }: { action: () => void; code: string; browserRecovery?: boolean }) {
  return (
    <main className="flex min-h-svh items-center justify-center bg-background px-6">
      <section className="max-w-md rounded-xl border border-border bg-card p-6 text-center shadow-sm">
        <h1 className="text-xl font-semibold" role="alert">
          {workbenchErrorMessage(code)}
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          {browserRecovery ? "请保留此恢复页面。在新标签页登录后，返回此页刷新并核实原操作；不会重新提交。" : "为保护企业数据，工作台内容已停止加载。"}
        </p>
        {browserRecovery ? <a className="mt-4 block text-sm underline" href="/login?returnTo=%2Fworkbench" target="_blank" rel="noopener noreferrer" referrerPolicy="no-referrer">在新标签页重新登录</a> : null}
        <Button className="mt-5" onClick={action} variant="outline">
          {browserRecovery ? "登录完成后刷新此页核实" : "重新加载"}
        </Button>
      </section>
    </main>
  );
}
