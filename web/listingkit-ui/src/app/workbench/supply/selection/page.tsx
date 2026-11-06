import {ProjectPage} from "@/components/workbench/project-center/project-page";
import {ResearchHomepage} from "@/components/workbench/console/research-homepage";
import {researchStep} from "@/lib/workbench/research-navigation";
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){
 const q=await searchParams,step=typeof q.step==="string"?q.step:"",mode=q.mode==="archived"?"archived":"active",search=typeof q.search==="string"?q.search:"";
 if(!researchStep("selection",step))return <ResearchHomepage center="selection" mode={mode} initialSearch={search}/>;
 return <ProjectPage center="selection" workflowStep={step} mode={mode} initialSearch={search} sample={typeof q.sample==="string"?q.sample:""} candidateAfter={typeof q.candidateAfter==="string"&&/^[0-9a-f-]{36}$/.test(q.candidateAfter)?q.candidateAfter:""}/>;
}
