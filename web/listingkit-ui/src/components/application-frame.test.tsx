import { fireEvent, render, screen } from "@testing-library/react";
import { PathnameContext } from "next/dist/shared/lib/hooks-client-context.shared-runtime";
import { useContext, type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/components/providers/theme-provider", () => ({
  ThemeProvider: ({ children }: { children: ReactNode }) => (
    <div data-testid="theme-provider">{children}</div>
  ),
}));
vi.mock("@/components/providers/query-provider", () => ({
  QueryProvider: ({ children }: { children: ReactNode }) => (
    <div data-testid="query-provider">{children}</div>
  ),
}));
vi.mock("@/components/providers/toast-provider", () => ({
  ToastProvider: ({ children }: { children: ReactNode }) => (
    <div data-testid="toast-provider">{children}</div>
  ),
}));
const context = vi.hoisted(() => ({user:{id:"actor"},effectiveOrganization:{id:"org"},effectiveMemberId:"member",permissions:[]}));
vi.mock("@/components/providers/workbench-context-provider", () => ({
  useWorkbenchContext: () => context,
  WorkbenchContextProvider: ({ children }: { children: ReactNode }) => (
    <div data-testid="workbench-context-provider">{children}</div>
  ),
}));
vi.mock("@/components/providers/zitadel-auth-gate", () => ({
  ZitadelAuthGate: ({ children }: { children: ReactNode }) => (
    <div data-testid="legacy-auth-gate">{children}</div>
  ),
}));
vi.mock("@/components/workbench/workspace-app-shell", () => ({
  WorkspaceAppShell: ({ children }: { children: ReactNode }) => (
    <div key={useContext(PathnameContext)} data-testid="workspace-app-shell">{children}</div>
  ),
}));
vi.mock("@/components/listingkit/shared/listingkit-app-shell", () => ({
  ListingKitAppShell: ({ children }: { children: ReactNode }) => (
    <div data-testid="legacy-listingkit-shell">{children}</div>
  ),
}));

import {
  ApplicationFrame,
  isPublicRoute,
  isWorkbenchRoute,
} from "./application-frame";

describe("isPublicRoute", () => {
  it("keeps every published policy document outside the authenticated workspace shell", () => {
    for (const pathname of ["/privacy-policy", "/user-agreement", "/ai-compute-billing", "/service-agreement"]) {
      expect(isPublicRoute(pathname)).toBe(true);
    }

    expect(isPublicRoute("/listing-kits/home")).toBe(false);
  });

  it("admits only the exact anonymous referral registration route", () => {
    expect(isPublicRoute("/referrals/register")).toBe(true);
    expect(isPublicRoute("/referrals")).toBe(false);
    expect(isPublicRoute("/referrals/register/extra")).toBe(false);
    expect(isPublicRoute("/referrals/register-lookalike")).toBe(false);
  });
});

describe("isWorkbenchRoute", () => {
  it("admits only the frozen Browser capture page, never as public", () => {
    expect(isWorkbenchRoute("/capture/1688")).toBe(true);
    expect(isWorkbenchRoute("/capture/amazon")).toBe(true);
    expect(isPublicRoute("/capture/1688")).toBe(false);
    for (const path of ["/capture/1688/", "/capture/1688/child", "/capture/16880", "/capture/other", null]) expect(isWorkbenchRoute(path)).toBe(false);
  });
  it("matches the Workbench root and descendants without matching lookalikes", () => {
    expect(isWorkbenchRoute("/workbench")).toBe(true);
    expect(isWorkbenchRoute("/workbench/no-organization")).toBe(true);
    expect(isWorkbenchRoute("/workbenches")).toBe(false);
    expect(isWorkbenchRoute(null)).toBe(false);
    expect(isWorkbenchRoute("/workbench/account/referrals/complete")).toBe(true);
  });
});

import WorkbenchLayout from "@/app/workbench/layout";
import {researchHandoffScope,useResearchHandoff} from "@/components/workbench/project-center/research-handoff-provider";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import type {DataJob} from "@/lib/contracts/data-services";

function HandoffProbe() {
  const {store,draft}=useResearchHandoff();
  const scope=researchHandoffScope(useWorkbenchContext());
  return <><button onClick={()=>store?.start(scope,"fixture-project",{} as DataJob)}>确认交接</button><span>{draft?.projectId??"空交接"}</span></>;
}

function atPath(path:string, children:ReactNode) {
  return <PathnameContext.Provider value={path}><ApplicationFrame>{children}</ApplicationFrame></PathnameContext.Provider>;
}

describe("ApplicationFrame", () => {
  it("keeps the same-tab handoff across a keyed shell remount and clears it on member change", () => {
    const tree=(path:string)=>atPath(path,<WorkbenchLayout><HandoffProbe/></WorkbenchLayout>);
    const selection="/workbench/supply/selection/fixture-project";
    const {rerender}=render(tree("/workbench/data/research/fixture-project"));
    fireEvent.click(screen.getByRole("button",{name:"确认交接"}));
    expect(screen.getByText("fixture-project")).toBeInTheDocument();
    rerender(tree(selection));
    expect(screen.getByText("fixture-project")).toBeInTheDocument();
    context.effectiveMemberId="changed";
    rerender(tree(selection));
    expect(screen.getByText("空交接")).toBeInTheDocument();
    context.effectiveMemberId="member";
  });
  it.each(["/capture/1688", "/capture/amazon"])("mounts the current provider/shell chain for %s", (path) => {
    render(atPath(path,<p>capture child</p>));
    expect(screen.getByTestId("theme-provider")).toContainElement(screen.getByTestId("query-provider"));
    expect(screen.getByTestId("query-provider")).toContainElement(screen.getByTestId("toast-provider"));
    expect(screen.getByTestId("toast-provider")).toContainElement(screen.getByTestId("workbench-context-provider"));
    expect(screen.getByTestId("workbench-context-provider")).toContainElement(screen.getByTestId("workspace-app-shell"));
    expect(screen.getByTestId("workspace-app-shell")).toContainElement(screen.getByText("capture child"));
    expect(screen.queryByTestId("legacy-auth-gate")).not.toBeInTheDocument();
    expect(screen.queryByTestId("legacy-listingkit-shell")).not.toBeInTheDocument();
  });
  it("routes Workbench pages through the isolated provider and shell chain", () => {
    render(atPath("/workbench/no-organization",<p>route child</p>));

    expect(screen.getByTestId("theme-provider")).toContainElement(
      screen.getByTestId("query-provider"),
    );
    expect(screen.getByTestId("toast-provider").closest(".console-theme")).not.toBeNull();
    expect(screen.getByTestId("workspace-app-shell").closest(".console-theme")).toBe(
      screen.getByTestId("toast-provider").closest(".console-theme"),
    );
    expect(screen.getByTestId("query-provider")).toContainElement(
      screen.getByTestId("toast-provider"),
    );
    expect(screen.getByTestId("toast-provider")).toContainElement(
      screen.getByTestId("workbench-context-provider"),
    );
    expect(screen.getByTestId("workbench-context-provider")).toContainElement(
      screen.getByTestId("workspace-app-shell"),
    );
    expect(screen.queryByTestId("legacy-auth-gate")).not.toBeInTheDocument();
    expect(screen.queryByTestId("legacy-listingkit-shell")).not.toBeInTheDocument();
  });

  it("keeps authentication without mounting the retired navigation shell", () => {
    render(atPath("/listing-kits/home",<p>legacy child</p>));

    expect(screen.getByTestId("legacy-auth-gate")).toContainElement(screen.getByText("legacy child"));
    expect(screen.queryByTestId("legacy-listingkit-shell")).not.toBeInTheDocument();
    expect(
      screen.queryByTestId("workbench-context-provider"),
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId("workspace-app-shell")).not.toBeInTheDocument();
    expect(screen.getByTestId("toast-provider").closest(".console-theme")).toBeNull();
  });

  it("keeps public routes shell-free", () => {
    render(atPath("/referrals/register",<p>public child</p>));

    expect(screen.getByText("public child")).toBeInTheDocument();
    expect(screen.queryByTestId("theme-provider")).not.toBeInTheDocument();
    expect(screen.queryByTestId("legacy-auth-gate")).not.toBeInTheDocument();
    expect(
      screen.queryByTestId("workbench-context-provider"),
    ).not.toBeInTheDocument();
  });
});
