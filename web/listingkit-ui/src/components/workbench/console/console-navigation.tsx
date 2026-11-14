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
import { consoleNavigation, findConsoleRoute, onlineAcquisitionHref, type ConsoleNavNode } from "@/lib/workbench/console-navigation";
import { firstConnectedConsoleEntry } from "@/lib/workbench/console-primary-entry";
import { researchLocation, researchNavigationHref, researchWorkflows, type ResearchLocation } from "@/lib/workbench/research-navigation";

export function ConsoleNavigation({ pathname, ariaLabel, onNavigate, dataServicesAvailable = false, reportCenterAvailable = false, toolMarketAvailable = false, supplyMarketAvailable = false, productAcquisitionAvailable = false, productCollectionsAvailable = false, supplyChainAvailable = false, supplyPreparationAvailable = false, supplyCompletionDraftsAvailable = false, knowledgeAvailable = false, marketResearchAvailable = false, projectCenterAvailable = false, operationsCockpitAvailable = false, cockpitPermissions = [], storeProductsAvailable = false, storeOrdersAvailable = false, fulfillmentAvailable = false, aiWorkbenchAvailable = false, nativeRunsAvailable = false, productReviewAvailable = false, ecoservicesAvailable = false, sheinRecordsAvailable = false, userId }: { pathname: string; ariaLabel: string; onNavigate?: () => void; dataServicesAvailable?: boolean; reportCenterAvailable?: boolean; toolMarketAvailable?: boolean; supplyMarketAvailable?: boolean; productAcquisitionAvailable?: boolean; productCollectionsAvailable?: boolean; supplyChainAvailable?: boolean; supplyPreparationAvailable?: boolean; supplyCompletionDraftsAvailable?: boolean; knowledgeAvailable?: boolean; marketResearchAvailable?: boolean; projectCenterAvailable?: boolean; operationsCockpitAvailable?: boolean; cockpitPermissions?: readonly string[]; storeProductsAvailable?: boolean; storeOrdersAvailable?: boolean; fulfillmentAvailable?: boolean; aiWorkbenchAvailable?: boolean; nativeRunsAvailable?: boolean; productReviewAvailable?: boolean; ecoservicesAvailable?: boolean; sheinRecordsAvailable?: boolean; userId?: string }) {
  const query=useSearchParams(), selection=new URLSearchParams();
  const research = researchLocation(pathname, new URLSearchParams(query?.toString()));
  const researchPath = research?.step ? `${research.workflow.root}?step=${research.step.id}` : pathname;
  const inListing = pathname === listingHome || pathname.startsWith(listingHome + "/");
  const trail = findConsoleRoute(researchPath)?.trail.map(node=>node.href) ?? (inListing ? [listingHome] : []);
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
  const nativeAvailable=nativeRunsAvailable&&canDiscoverNativeRuns(cockpitPermissions);
  const independentTaskAvailable = nativeAvailable || productReviewAvailable || sheinRecordsAvailable;
  const entryCapabilities = { operationsCockpitAvailable, cockpitPermissions, productAcquisitionAvailable, supplyMarketAvailable, supplyChainAvailable, supplyPreparationAvailable, supplyCompletionDraftsAvailable, dataServicesAvailable, productCollectionsAvailable, toolMarketAvailable, ecoservicesAvailable, aiWorkbenchAvailable, projectCenterAvailable, knowledgeAvailable, reportCenterAvailable, productReviewAvailable, sheinRecordsAvailable };
  const nodes = consoleNavigation.map(entry => {
    const connected = entry.href === "/workbench" && operationsCockpitAvailable || firstConnectedConsoleEntry(entry.href, entryCapabilities) !== null;
    const node = { ...entry, availability: connected ? "connected" as const : "unavailable" as const };
    if (researchWorkflows.some(workflow => workflow.root === node.href)) {
      const availability = marketResearchAvailable ? "connected" as const : "unavailable" as const;
      return { ...node, availability, children: node.children?.map(step => ({ ...step, availability })) };
    }
    if (node.href === "/workbench" && operationsCockpitAvailable) return { ...node, availability: "connected" as const, children: node.children?.map(child => {
      const mode = child.href.split("/").at(-1), permission = mode === "stores" ? "workbench.cockpit.stores.read" : `workbench.cockpit.${mode}.read`;
      return { ...child, availability: cockpitPermissions.includes(permission) ? "connected" as const : "unavailable" as const };
    }) };
    if (node.href === "/workbench/tools" && toolMarketAvailable) return {...node, availability:"connected" as const, children:node.children?.map(child=>({...child, availability:"connected" as const, children:child.children?.map(entry=>({...entry, availability:"connected" as const}))}))};
 if(node.href === "/workbench/store-orders") return {...node,availability:storeOrdersAvailable||fulfillmentAvailable?"connected" as const:"unavailable" as const,children:node.children?.map(child=>({...child,availability:(child.href.endsWith("/records")?storeOrdersAvailable:fulfillmentAvailable)?"connected" as const:"unavailable" as const}))};
 if(node.href === "/workbench/store-center") return {...node,children:node.children?.map(child => child.href === "/workbench/store-products" ? {...child,availability:storeProductsAvailable ? "connected" as const:"unavailable" as const} : child.href === "/workbench/store-orders" ? {...child,availability:storeOrdersAvailable ? "connected" as const:"unavailable" as const} :child)};
    if(node.href==="/workbench/services"&&ecoservicesAvailable)return {...node,availability:"connected" as const,children:node.children?.map(child=>({...child,availability:"connected" as const}))};
    if (node.href === "/workbench/supply") return { ...node, children: node.children?.flatMap(child => {
      if (child.href === onlineAcquisitionHref) {
        return productAcquisitionAvailable || dataServicesAvailable ? [child] : [];
      }
      return [supplyMarketAvailable && ["/workbench/supply/official", "/workbench/supply/selected", "/workbench/supply/catalogs", "/workbench/supply/applications"].includes(child.href)
        ? { ...child, availability: "connected" as const, children: child.children?.map(entry => ({ ...entry, availability: "connected" as const })) }
        : child];
    }) };
    if (node.href === "/workbench/data") return { ...node, availability: dataServicesAvailable || productCollectionsAvailable ? "connected" as const : node.availability, children: node.children?.map(child => child.href === "/workbench/data/mine" && productCollectionsAvailable || ["/workbench/data/market", "/workbench/data/api"].includes(child.href) && dataServicesAvailable ? { ...child, availability: "connected" as const } : child) };
    if (node.href !== "/workbench/ai") return node;
    const children = (node.children ?? [])
      .filter(child => aiWorkbenchAvailable || child.href !== "/workbench/ai/chat" && (child.href !== "/workbench/ai/tasks" || independentTaskAvailable))
      .map(child => child.href === "/workbench/ai/reports" ? {...child, availability:reportCenterAvailable ? "connected" as const:"unavailable" as const, children:undefined} : child.href === "/workbench/ai/projects" ? {...child,availability:projectCenterAvailable?"connected" as const:"unavailable" as const,children:undefined} : child.href === "/workbench/ai/tasks" || child.href === "/workbench/ai/chat" || child.href === "/workbench/agents/skills" ? { ...child, children: undefined } : child.href === "/workbench/ai/knowledge" ? { ...child, availability: "connected" as const, children: undefined } : child);
    return { ...node, availability: connected ? "connected" as const : "unavailable" as const, children };
  });
  return <nav aria-label={ariaLabel} className="console-nav"><ul>{nodes.map(node => supplyMarketAvailable && userId && node.href==="/workbench/supply" ? <MarketBranch key={`${pathname}:${node.href}`} node={node} pathname={pathname} trail={trail} onNavigate={onNavigate} userId={userId}/> : ecoservicesAvailable && userId && node.href === "/workbench/services" ? <EcoservicesBranch key={`${pathname}:${node.href}`} node={node} pathname={pathname} trail={trail} onNavigate={onNavigate} userId={userId} /> : <NavBranch key={`${node.href==="/workbench/store-orders"?orderPath:researchPath}:${node.href}`} node={node} pathname={node.href==="/workbench/store-orders"?orderPath:researchPath} research={research} trail={trail} depth={1} onNavigate={onNavigate} listingQuery={listingQuery} orderQuery={node.href==="/workbench/store-orders"?orderSelection.toString():""} />)}</ul></nav>;
}
function MarketBranch({node,pathname,trail,onNavigate,userId}:{node:ConsoleNavNode;pathname:string;trail:readonly string[];onNavigate?:()=>void;userId:string}){
 const authorized=useQuery({queryKey:["supply-market",userId,"","records",true,"selected",false,undefined],queryFn:({signal})=>listMarketRecords({userId,organizationId:""},{kind:"selected",ended:false},true,signal),retry:false});
 const branch=authorized.isSuccess&&!authorized.isError ? {...node,children:[...(node.children??[]),{label:"发布管理/处理申请",href:"/workbench/supply/operator",availability:"connected" as const}]}:node;
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
  const active = trail.includes(node.href) || pathname === node.href;
  // The route-keyed subtree resets disclosure even when navigating back to an earlier route.
  const [expanded, setExpanded] = useState(active);
  const id = useId();
  const Icon = navigationIcons[node.href] ?? Circle;
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
