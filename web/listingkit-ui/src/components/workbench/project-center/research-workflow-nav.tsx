import Link from "next/link";
import { researchLocation, researchNavigationHref, researchWorkflow, type ResearchCenter } from "@/lib/workbench/research-navigation";
import { WorkbenchViewNav } from "../console/workbench-view-nav";
import styles from "./project-page.module.css";

export function ResearchWorkflowNav({center,projectId,step,mode,search,sample,candidateAfter,researchToken}:{center:ResearchCenter;projectId?:string;step?:string;mode:string;search:string;sample?:string;candidateAfter?:string;researchToken?:string}) {
  const workflow=researchWorkflow(center);
  const query=new URLSearchParams({mode,search,...(step?{step}:{}),...(sample?{sample}:{}),...(candidateAfter?{candidateAfter}:{})});
  const location=researchLocation(projectId ? `${workflow.root}/${projectId}` : workflow.root,query);
  return <WorkbenchViewNav className={styles.viewNavigation} label={`${workflow.label}步骤`}>
    {workflow.phases.flatMap(phase=>phase.steps.map(item=>{
      const href=researchNavigationHref(`${workflow.root}?step=${item.id}`,location);
      const sameProjectHandoff=center==="selection" && !!location?.projectId && researchToken!==undefined;
      return <Link key={item.id} href={sameProjectHandoff ? `${href}&handoff=${encodeURIComponent(researchToken)}` : href} prefetch={false} aria-current={step===item.id?"page":undefined}>{item.label}</Link>;
    }))}
  </WorkbenchViewNav>;
}
