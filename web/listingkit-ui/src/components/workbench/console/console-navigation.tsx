"use client";
import {fulfillmentID} from "@/lib/api/fulfillment";
import {canDiscoverNativeRuns} from "../ai-workbench/native-run-records";

import Link from "next/link";
import { Gauge, PanelsTopLeft, Package, Bot, Wrench, Network, Database, Store, Ticket, UserRound, Circle, ChevronDown, Target, LayoutGrid, BellRing, Lightbulb, MessageSquare, ListChecks, FolderKanban, BookOpen, Sparkles, FileChartColumn, ScanLine, Building2, Star, Boxes, GitBranch, ClipboardList, ClipboardCheck, BotMessageSquare, Settings2, BriefcaseBusiness, Handshake, UserPlus, Braces, FolderArchive, Truck, Layers, BadgeCheck, ChartNoAxesCombined, Wallet, ReceiptText, IdCard, Megaphone, type LucideIcon } from "lucide-react";

const navigationIcons: Record<string, LucideIcon> = {
  "/workbench": Gauge, "/workbench/ai": PanelsTopLeft, "/workbench/supply": Package,
  "/workbench/agents": Bot, "/workbench/tools": Wrench, "/workbench/services": Network,
  "/workbench/data": Database, "/workbench/store-center": Store, "/workbench/plans": Ticket, "/workbench/account": UserRound,
  "/workbench/overview/goals": Target, "/workbench/overview/stores": LayoutGrid,
  "/workbench/overview/alerts": BellRing, "/workbench/overview/advice": Lightbulb,
  "/workbench/ai/chat": MessageSquare, "/workbench/ai/tasks": ListChecks,
  "/workbench/ai/projects": FolderKanban, "/workbench/ai/knowledge": BookOpen,
  "/workbench/agents/skills": Sparkles, "/workbench/ai/reports": FileChartColumn,
  [onlineAcquisitionHref]: ScanLine, "/workbench/supply/official": Building2,
  "/workbench/supply/selected": Star, "/workbench/supply/catalogs": Boxes,
  [listingHome]: GitBranch, "/workbench/supply/applications": ClipboardList,
  "/workbench/supply/operator": ClipboardCheck,
  "/workbench/agents/market": Bot, "/workbench/agents/mine": BotMessageSquare,
  "/workbench/agents/custom": Settings2,
  "/workbench/tools/official": Wrench, "/workbench/tools/mine": BriefcaseBusiness,
  "/workbench/tools/custom": Settings2,
  "/workbench/services/market": Handshake, "/workbench/services/mine": BriefcaseBusiness,
  "/workbench/services/join": UserPlus, "/workbench/services/review": ClipboardCheck,
  "/workbench/data/research": ChartNoAxesCombined,"/workbench/supply/selection": ClipboardCheck, "/workbench/data/market": Database, "/workbench/data/api": Braces, "/workbench/data/mine": FolderArchive,
  "/workbench/stores": Store, "/workbench/store-products": Package, "/workbench/store-orders": Truck,
  "/workbench/plans/options": Layers, "/workbench/plans/entitlements": BadgeCheck,
  "/workbench/plans/usage": ChartNoAxesCombined, "/workbench/plans/top-up": Wallet,
  "/workbench/plans/orders": ReceiptText,
  "/workbench/account/profile": IdCard, "/workbench/account/organization": Building2,
  "/workbench/account/referrals": Megaphone,
};
import {useSearchParams} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {listingHome} from "@/lib/workbench/listing-navigation";
import { useId, useState, type ReactNode } from "react";
import { AnimatePresence, motion, useIsPresent, useReducedMotion } from "motion/react";
import { useQuery } from "@tanstack/react-query";
import { ecoPageSchema, ecoRequest } from "@/lib/api/ecoservices";
import { listMarketRecords } from "@/lib/api/supply-market";
import { amazonRealtimeAcquisitionHref, consoleNavigation, findConsoleRoute, onlineAcquisitionHref, type ConsoleNavNode } from "@/lib/workbench/console-navigation";
import { firstConnectedConsoleEntry, type ConsolePrimaryCapabilities } from "@/lib/workbench/console-primary-entry";
import { researchLocation, researchNavigationHref, type ResearchLocation } from "@/lib/workbench/research-navigation";

type SidebarAccess = Omit<ConsolePrimaryCapabilities, "knowledgeAvailable"> & {
  userId?: string;
  hasOrganization?: boolean;
  permissions: readonly string[];
  marketResearchAvailable?: boolean;
  storeObservationsAvailable?: boolean;
  fulfillmentAvailable?: boolean;
};

type ConsoleNavigationProps = SidebarAccess & {
  pathname: string;
  ariaLabel: string;
  onNavigate?: () => void;
};

// These rules mirror the existing page/read gates. The static tree still
// describes routes and breadcrumbs for bookmarks and direct navigation.
function sidebarPages(access: SidebarAccess): Set<string> {
  const has = (permission: string) => access.permissions.includes(permission);
  const pages = new Set<string>();
  const include = (allowed: boolean | undefined, ...paths: string[]) => {
    if (!allowed) return;
    for (const path of paths) pages.add(`/workbench/${path}`);
  };

  for (const mode of ["goals", "stores", "alerts", "advice"]) {
    include(access.operationsCockpitAvailable && has(`workbench.cockpit.${mode}.read`), `overview/${mode}`);
  }
  include(access.aiWorkbenchAvailable && has("workbench.chat.read"), "ai/chat");
  include(
    access.aiWorkbenchAvailable && has("workbench.task.read") ||
    access.nativeRunsAvailable && canDiscoverNativeRuns(access.permissions) ||
    access.productReviewAvailable && has("local_agent.write") ||
    access.sheinRecordsAvailable && has("listingkit.admin.read"),
    "ai/tasks",
  );
  include(access.projectCenterAvailable && has("workbench.project.read"), "ai/projects");
  // Official knowledge remains available when enterprise knowledge is disabled.
  include(has("workbench.knowledge.read"), "ai/knowledge");
  include(access.reportCenterAvailable && has("workbench.report.read"), "ai/reports");
  // Official and personal skills have an independent signed-in route.
  include(!!access.userId, "agents/skills");

  const dataMarket = access.dataServicesAvailable && has("workbench.data-market.use");
  include(access.productAcquisitionAvailable && has("product_sourcing.write") || dataMarket, "supply/acquisition");
  include(access.supplyMarketAvailable && has("workbench.supply-market.read"), "supply/official", "supply/selected", "supply/catalogs", "supply/applications");
  include(access.supplyPreparationAvailable && has("workbench.supply.read"), "supply");
  if (has("workbench.supply.read") && firstConnectedConsoleEntry(listingHome, access)) pages.add(listingHome);
  const canReadAgents = has("workbench.agent.read");
  const canReadResearch = access.marketResearchAvailable && access.projectCenterAvailable && has("workbench.project.read");
  include(canReadAgents || canReadResearch, "agents/market");
  include(canReadAgents, "agents/mine", "agents/custom");
  include(access.toolMarketAvailable && has("workbench.tools.read"), "tools/official", "tools/mine", "tools/custom");
  include(access.ecoservicesAvailable && has("workbench.ecoservices.read"), "services/market", "services/mine");
  include(access.ecoservicesAvailable && has("workbench.ecoservices.join"), "services/join");

  include(dataMarket, "data/market");
  include(access.dataServicesAvailable && has("workbench.data-api.manage"), "data/api");
  include(access.productCollectionsAvailable && has("workbench.collection.read"), "data/mine");
  include(has("workbench.store.read"), "stores");
  include(access.storeObservationsAvailable && has("workbench.store.read") && has("workbench.store.products.read"), "store-products");
  include(access.storeObservationsAvailable && has("workbench.store.read") && has("workbench.store.orders.read"), "store-center", "store-orders/records");
  include(access.fulfillmentAvailable && has("workbench.fulfillment.read"),
    "store-center", "store-orders/fulfill?step=source", "store-orders/fulfill?step=purchase",
    "store-orders/fulfill?step=stock", "store-orders/fulfill?step=logistics");
  include(has("workbench.commercial.read"), "plans/options", "plans/entitlements", "plans/usage", "plans/top-up", "plans/orders");
  include(access.hasOrganization, "account/organization");
  include(true, "account/profile", "account/referrals");
  return pages;
}

export function ConsoleNavigation({ pathname, ariaLabel, onNavigate, ...access }: ConsoleNavigationProps) {
  const query=useSearchParams(), selection=new URLSearchParams();
  const research = researchLocation(pathname, new URLSearchParams(query?.toString()));
  const researchPath = research?.step ? `${research.workflow.root}?step=${research.step.id}` : pathname;
  const inListing = pathname === listingHome || pathname.startsWith(listingHome + "/");
  const navigationPath = pathname.startsWith("/workbench/supply/catalogs/") ? "/workbench/supply/catalogs" : pathname.startsWith("/workbench/supply/applications/") ? "/workbench/supply/applications" : researchPath;
  const trail = research ? ["/workbench/agents", "/workbench/agents/market"] : findConsoleRoute(navigationPath)?.trail.map(node=>node.href) ?? (inListing ? [listingHome] : []);
  if(inListing){
    for(const key of ["preparation","store","operation","plan"]){
      const values=query?.getAll(key);
      if(values?.length===1 && collectionID.safeParse(values[0]).success)selection.set(key,values[0]);
    }
  }
  const orderSteps=["source","purchase","stock","platform","logistics"];
  const requestedStep=query?.getAll("step");
  const orderStep=requestedStep?.length===1 && orderSteps.includes(requestedStep[0]) ? requestedStep[0] : "source";
  const orderPath=pathname==="/workbench/store-orders/fulfill" ? `${pathname}?step=${orderStep==="platform"?"stock":orderStep}` : pathname;
  const orderSelection=new URLSearchParams();
  if(pathname==="/workbench/store-orders/fulfill"){
    for(const key of ["storeId","sourceId"]){const values=query?.getAll(key);if(values?.length===1&&fulfillmentID.safeParse(values[0]).success)orderSelection.set(key,values[0]);}
    const kinds=query?.getAll("kind");if(kinds?.length===1&&(kinds[0]==="consumer_order"||kinds[0]==="platform_supply"))orderSelection.set("kind",kinds[0]);
  }
  const listingQuery=selection.size?`?${selection}`:"";
  const pages = sidebarPages(access);
  // The combined page defaults to 1688 when both platforms are deployed.
  const acquisitionHref = access.productAcquisitionAvailable && !access.permissions.includes("product_sourcing.write")
    ? amazonRealtimeAcquisitionHref
    : onlineAcquisitionHref;
  return <nav aria-label={ariaLabel} className="console-nav"><ul>{consoleNavigation.map(entry => {
    const children = entry.children?.filter(child => pages.has(child.href))
      .map(child => ({
        ...child,
        href: child.href === onlineAcquisitionHref ? acquisitionHref : child.href,
        children: undefined,
        availability: "connected" as const,
      }));
    // Personal account links remain usable when the enterprise overview cannot open.
    const hasHome = entry.href === "/workbench/account"
      ? access.hasOrganization
      : pages.has(entry.href) || children?.some(child => child.href !== "/workbench/agents/skills");
    const node: ConsoleNavNode = {
      ...entry,
      availability: "connected",
      children: children?.length ? children : undefined,
      disclosureOnly: !hasHome,
    };
    const key = `${pathname}:${entry.href}`;
    if (access.userId && access.supplyMarketAvailable && entry.href === "/workbench/supply") {
      return <MarketBranch key={key} node={node} pathname={pathname} trail={trail} onNavigate={onNavigate} userId={access.userId} />;
    }
    if (access.userId && access.ecoservicesAvailable && entry.href === "/workbench/services") {
      return <EcoservicesBranch key={key} node={node} pathname={pathname} trail={trail} onNavigate={onNavigate} userId={access.userId} />;
    }
    if (!hasHome && !node.children) return null;
    const isOrders = entry.href === "/workbench/store-orders";
    const activePath = isOrders ? orderPath : researchPath;
    return <NavBranch key={`${activePath}:${entry.href}`} node={node} pathname={activePath} research={research} trail={trail} depth={1} onNavigate={onNavigate} listingQuery={listingQuery} orderQuery={isOrders ? orderSelection.toString() : ""} />;
  })}</ul></nav>;
}
function MarketBranch({node,pathname,trail,onNavigate,userId}:{node:ConsoleNavNode;pathname:string;trail:readonly string[];onNavigate?:()=>void;userId:string}){
 const authorized=useQuery({queryKey:["supply-market",userId,"","records",true,"selected",false,undefined],queryFn:({signal})=>listMarketRecords({userId,organizationId:""},{kind:"selected",ended:false},true,signal),retry:false});
 const branch=authorized.isSuccess&&!authorized.isError ? {...node,children:[...(node.children??[]),{label:"发布管理",href:"/workbench/supply/operator",availability:"connected" as const}]}:node;
 if (branch.disclosureOnly && !branch.children) return null;
 const currentTrail=pathname==="/workbench/supply/operator" || pathname.startsWith("/workbench/supply/operator/") ? [...trail,node.href,"/workbench/supply/operator"]:trail;
 return <NavBranch node={branch} pathname={pathname} trail={currentTrail} depth={1} onNavigate={onNavigate}/>;
}

function EcoservicesBranch({ node, pathname, trail, onNavigate, userId }: { node: ConsoleNavNode; pathname: string; trail: readonly string[]; onNavigate?: () => void; userId: string }) {
  // Reuse the page's global platform read and cache. Enterprise roles/grants
  // cannot advertise review; the backend's configured platform gate decides.
  const review = useQuery({
    queryKey: ["ecoservices", userId, "", "platform", "applications", 1],
    queryFn: ({ signal }) => ecoRequest({ userId, organizationId: "" }, "applications?page=1&pageSize=20", ecoPageSchema, { signal }, true),
    retry: false,
  });
  const authorized = review.isSuccess && !review.isError;
  const currentTrail = pathname === "/workbench/services/review" ? [...trail, node.href] : trail;
  const branch = authorized ? { ...node, children: [...node.children ?? [], { label: "平台审核", href: "/workbench/services/review", availability: "connected" as const }] } : node;
  if (branch.disclosureOnly && !branch.children) return null;
  return <NavBranch node={branch} pathname={pathname} trail={currentTrail} depth={1} onNavigate={onNavigate} />;
}

function NavBranch({ node, pathname, trail, depth, onNavigate, listingQuery = "", orderQuery = "", research }: { node: ConsoleNavNode; pathname: string; trail: readonly string[]; depth: number; onNavigate?: () => void; listingQuery?:string; orderQuery?:string; research?:ResearchLocation }) {
  const orderParameters=new URLSearchParams(node.href.split("?")[1]);
  if(node.href.startsWith("/workbench/store-orders/fulfill?"))for(const [key,value] of new URLSearchParams(orderQuery))orderParameters.set(key,value);
  const originalHref = node.href === "/workbench/ai/chat" ? "/workbench/ai/chat/new"
    : node.href === "/workbench/agents/skills" ? "/workbench/agents/skills/official"
    : node.href.startsWith("/workbench/store-orders/fulfill?")
    ? `/workbench/store-orders/fulfill?${orderParameters}`
    : node.href===listingHome&&listingQuery ? node.href+listingQuery : node.href;
  const href = researchNavigationHref(originalHref, research);
  const menuHref = node.href === amazonRealtimeAcquisitionHref ? onlineAcquisitionHref : node.href;
  const active = trail.includes(menuHref) || pathname === node.href;
  // The route-keyed subtree resets disclosure even when navigating back to an earlier route.
  const [expanded, setExpanded] = useState(active);
  const id = useId();
  const Icon = navigationIcons[menuHref] ?? Circle;
  return <li className={`console-nav-level-${depth}`}>
    <div className="console-nav-row" data-active={active || undefined}>
      {node.disclosureOnly ? <button className="console-nav-group" type="button" aria-label={`${expanded ? "收起" : "展开"}${node.label}`} aria-expanded={expanded} aria-controls={id} onClick={() => setExpanded(value => !value)}><Icon className="console-nav-icon" size={16} aria-hidden="true" /><span className="console-nav-label">{node.label}</span><ChevronDown className="console-nav-chevron" size={16} aria-hidden="true" /></button> : <Link href={href} prefetch={false} aria-current={pathname === node.href || (!node.children && active) ? "page" : undefined} onClick={onNavigate} title={node.availability === "unavailable" ? `${node.label}：业务暂未启用` : node.label}>
        {depth <= 2 ? <Icon className="console-nav-icon" size={depth === 1 ? 20 : 16} aria-hidden="true" /> : null}<span className="console-nav-label">{node.label}</span>
      </Link>}
      {node.children && !node.disclosureOnly ? <button type="button" aria-label={`${expanded ? "收起" : "展开"}${node.label}`} aria-expanded={expanded} aria-controls={id} onClick={() => setExpanded(value => !value)}><ChevronDown className="console-nav-chevron" size={16} aria-hidden="true" /></button> : null}
    </div>
    <AnimatePresence initial={false}>{node.children && expanded ? <NavigationChildren key={id} id={id}>{node.children.map((child) => <NavBranch key={child.href} node={child} pathname={pathname} trail={trail} depth={depth + 1} onNavigate={onNavigate} listingQuery={listingQuery} orderQuery={orderQuery} research={research} />)}</NavigationChildren> : null}</AnimatePresence>
  </li>;
}

function NavigationChildren({ id, children }: { id: string; children: ReactNode }) {
  const present = useIsPresent();
  const reduceMotion = useReducedMotion();
  // AnimatePresence retains closing DOM briefly; it must stop accepting focus immediately.
  return <motion.ul id={id} aria-hidden={present ? undefined : true} inert={!present}
    initial={{ height: 0, opacity: 0.65, overflow: "hidden" }}
    animate={{ height: "auto", opacity: 1, transitionEnd: { overflow: "visible" } }}
    exit={{ height: 0, opacity: 0, overflow: "hidden" }}
    transition={{ duration: reduceMotion ? 0 : 0.2, ease: [0.2, 0.8, 0.2, 1] }}>
    {children}
  </motion.ul>;
}
