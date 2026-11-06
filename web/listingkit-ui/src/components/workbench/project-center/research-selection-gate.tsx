"use client";

import Link from "next/link";
import {useEffect,useState,type ReactNode} from "react";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import type {ProjectScope} from "@/lib/api/project-center";
import {verifyResearchHandoff} from "@/lib/api/research-handoff";
import type {ResearchInspection} from "@/lib/contracts/research-analysis";
import {handoffToken} from "@/lib/workbench/research-handoff";
import {researchHandoffScope,useResearchHandoff} from "./research-handoff-provider";

export function ResearchSelectionGate({scope,projectId,archived,token,children}:{
  scope:ProjectScope; projectId:string; archived:boolean; token:string;
  children:(inspection:ResearchInspection,after:string)=>ReactNode;
}) {
  const {store,draft}=useResearchHandoff();
  const context=useWorkbenchContext();
  const scopeKey=researchHandoffScope(context);
  const permitted=context.permissions.includes("workbench.project.manage") && context.permissions.includes("workbench.data-market.use");
  const valid=!!draft?.confirmed && draft.scope===scopeKey && draft.projectId===projectId && handoffToken(draft)===token && !archived && permitted;
  const [checked,setChecked]=useState<{token:string; inspection:ResearchInspection}|null>(null);
  const [failed,setFailed]=useState("");
  useEffect(()=>{
    if (!store || !valid) return;
    const current=store.getSnapshot();
    if (!current?.confirmed || handoffToken(current)!==token) return;
    const controller=new AbortController();
    verifyResearchHandoff(scope,current,controller.signal).then(({inspection})=>{
      if (!controller.signal.aborted && store.getSnapshot()===current) setChecked({token,inspection});
    }).catch(()=>{
      if (!controller.signal.aborted && store.getSnapshot()===current) {
        store.start(current.scope,current.projectId,current.job);
        setFailed(token);
      }
    });
    return ()=>controller.abort();
  },[store,valid,scope,token]);
  if (!valid || failed===token) return <p role="alert">交接已失效。<Link href={"/workbench/data/research/"+projectId} prefetch={false}>回市场调研重新确认</Link></p>;
  if (checked?.token!==token) return <p role="status">正在核对本次交接的原版资料…</p>;
  return children(checked.inspection,draft.confirmed!.after);
}
