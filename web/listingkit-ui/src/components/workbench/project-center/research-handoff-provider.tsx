"use client";

import {createContext,useContext,useLayoutEffect,useState,useSyncExternalStore,type ReactNode} from "react";
import {usePathname} from "next/navigation";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {createResearchHandoff,researchScopeKey,type ResearchHandoffStore} from "@/lib/workbench/research-handoff";

const HandoffContext=createContext<ResearchHandoffStore|null>(null);
const noSubscribe=()=>()=>{};
const empty=()=>null;

export function researchHandoffScope(context:ReturnType<typeof useWorkbenchContext>) {
  if (!context.user || !context.effectiveOrganization || !context.effectiveMemberId || context.isSwitching ||
      context.isLoading || context.selectionRequired || context.error || context.blockingError) return "";
  return researchScopeKey(context.user.id,context.effectiveOrganization.id,context.effectiveMemberId,context.permissions);
}

export function ResearchHandoffProvider({children}:{children:ReactNode}) {
  const [store]=useState(createResearchHandoff);
  const scope=researchHandoffScope(useWorkbenchContext());
  const pathname=usePathname();
  useLayoutEffect(()=>{
    const draft=store.getSnapshot();
    const project=pathname?.match(/^\/workbench\/(?:data\/research|supply\/selection|ai\/projects)\/([^/]+)\/?$/)?.[1];
    const enteringMarket=pathname==="/workbench/data/market" || pathname?.startsWith("/workbench/data/market/");
    if (draft && (draft.scope!==scope || enteringMarket || project && project!==draft.projectId)) store.clear();
  },[store,scope,pathname]);
  return <HandoffContext.Provider value={store}>{children}</HandoffContext.Provider>;
}

export function useResearchHandoff() {
  const store=useContext(HandoffContext);
  const draft=useSyncExternalStore(store?.subscribe??noSubscribe,store?.getSnapshot??empty,empty);
  return {store,draft};
}
