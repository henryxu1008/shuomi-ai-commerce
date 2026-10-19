"use client";
import {useEffect,useState} from "react";
import Link from "next/link";
import {skillRequest,type SkillScope} from "@/lib/api/skill";
import {skillCandidatesSchema,type SkillVersion} from "@/lib/contracts/skill";
import {skillLaunchMatches,type SkillLaunchInput,type SkillLaunch} from "@/lib/contracts/skill-launch";
import {Button} from "@/components/ui/button";
import {ConsoleState} from "../console/console-page";

export function SkillLaunchError(){return <ConsoleState kind="unavailable" title="技能入口已失效或身份、企业已变化"><p>原技能引用未替换。请回到当前企业技能库重新选择准确版本，原任务和请求继续保留。</p><Button asChild variant="outline"><Link href="/workbench/agents/skills/official">返回官方模板</Link></Button></ConsoleState>}
export function SkillLaunchSummary({launch,version}:{launch:SkillLaunch;version?:SkillVersion}){return <div className="space-y-2 rounded-xl border border-border bg-muted/40 p-4"><p className="font-semibold">本次技能：{version?.package.displayName||version?.package.name||"已选准确版本"} · v{launch.selection.version}</p><p className="break-all text-xs text-muted-foreground">{launch.selection.skillId} · {launch.selection.digest}</p><p className="text-sm text-muted-foreground">入口只带入选择，生成前仍需确认。原任务保留其准确版本。</p></div>}
export function useLaunchVersion(launch:SkillLaunchInput,scope:SkillScope){
 const key=JSON.stringify([launch,scope.userId,scope.organizationId]);
 const [read,setRead]=useState<{key:string;version?:SkillVersion;error?:string}>();
 useEffect(()=>{
  if(!launch||"invalid" in launch||!scope.organizationId||!skillLaunchMatches(launch,{userId:scope.userId,organizationId:scope.organizationId}))return;
  const controller=new AbortController();
  void skillRequest(scope,launch.target==="product-image"?"organization/image-candidates":"organization/candidates",skillCandidatesSchema,{signal:controller.signal}).then(page=>{
   if(controller.signal.aborted)return;const selected=page.items.find(v=>v.skill.owner.kind==="PLATFORM"&&(v.package.target??"product-title")===launch.target&&v.skill.id===launch.selection.skillId&&v.version===launch.selection.version&&v.package.digest===launch.selection.digest);
   setRead({key,...(selected?{version:selected}:{error:"所选准确版本当前不可执行，请回技能库重新选择。原引用与任务保持。"})});
  }).catch(()=>{if(!controller.signal.aborted)setRead({key,error:"暂时无法核实所选准确版本，请重新打开入口。"})});return()=>controller.abort();
 // Scope fields, rather than caller object identity, define this read.
 // eslint-disable-next-line react-hooks/exhaustive-deps
 },[key]);
 return {ready:!launch||read?.key===key&&!!read.version,version:read?.key===key?read.version:undefined,error:read?.key===key?read.error:undefined};
}
